// Copyright (C) 2026 Leandro Lisboa Penz <lpenz@lpenz.org>
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

//! Run a command line under [`strace`](https://strace.io) and yield its raw
//! trace output, one [`String`] per line, with no interpretation applied.
//!
//! This layer deliberately does no parsing. [`super::parser`] builds a stream
//! of structured items on top of [`trace_lines`].

use color_eyre::Result;
use color_eyre::eyre::eyre;
use std::ffi::OsStr;
use std::os::fd::AsRawFd as _;
use std::os::fd::FromRawFd;
use std::os::fd::OwnedFd;
use std::pin::Pin;
use std::process::Stdio;
use std::task::Context;
use std::task::Poll;
use tokio::io::AsyncBufReadExt as _;
use tokio::process::Command;
use tokio_stream::Stream;
use tokio_stream::StreamExt as _;

/// File descriptor the trace is written to, inside the strace child.
///
/// The trace cannot go to the child's stdout or stderr: strace passes the
/// traced process's stdout and stderr straight through to its own, so trace
/// output and program output would interleave on the same pipe, producing torn
/// lines such as `write(1</dev/null>, "boom\n", 5boom` that no parser can
/// recover. A dedicated descriptor avoids the collision entirely.
const TRACE_FD: i32 = 3;

/// strace arguments used by [`trace_lines`].
///
/// * `-f` follows `fork`, `vfork`, `clone` and `execve`, so the whole process
///   tree is traced rather than just the top-level process.
/// * `-tt` prefixes every line with a `HH:MM:SS.microseconds` timestamp.
/// * `-s 4096` raises the string truncation limit from its 32-byte default.
///   Paths in the cargo registry and `node_modules` exceed the default and
///   would otherwise be silently cut off.
/// * `-y` decodes file descriptors into the paths they refer to. Without it
///   `read(3, ...)` carries only a bare descriptor number and cannot be
///   attributed to a file.
/// * `-e trace=%file,%desc` retains path-resolution and descriptor syscalls,
///   which is what dependency tracking needs, and drops scheduling and memory
///   management noise.
const STRACE_ARGS: &[&str] = &[
    "-f",
    "-tt",
    "-s",
    "4096",
    "-y",
    "-e",
    "trace=%file,%desc",
    "-o",
    "/proc/self/fd/3",
];

/// A [`Stream`] of raw strace output lines, in the order strace emitted them.
///
/// Lines are yielded verbatim. This includes strace's own non-syscall lines
/// (`+++ exited with 0 +++`, `--- SIGCHLD ... ---`) and the two halves of a
/// syscall interrupted by another process's output
/// (`... <unfinished ...>` / `<... resumed> ...`); interpreting those is the
/// job of [`super::parser`].
pub struct StraceLines {
    inner: Pin<Box<dyn Stream<Item = Result<String>> + Send>>,
}

impl StraceLines {
    /// Run `command` under strace and return a stream of its raw trace lines.
    ///
    /// `command` is the program to execute followed by its arguments, and is
    /// passed through verbatim as an argument vector, so it is not subject to
    /// shell interpretation, quoting or glob expansion.
    ///
    /// The traced process's own stdout and stderr are redirected to
    /// `/dev/null` so that they cannot corrupt the trace. The stream ends once
    /// the whole process tree has exited, at which point the trace pipe reaches
    /// end-of-file.
    ///
    /// # Errors
    ///
    /// Returns an error if the strace pipe cannot be created, or if `strace`
    /// cannot be spawned. Note that a non-zero exit from `command` is not an
    /// error here; it is reported through the trace, not by this function.
    pub fn new<I, S>(command: I) -> Result<Self>
    where
        I: IntoIterator<Item = S>,
        S: AsRef<OsStr>,
    {
        let mut fds = [0; 2];
        // SAFETY: `pipe2` writes exactly two descriptors into the provided
        // array, and the caller has verified the array is that long.
        if unsafe { libc::pipe2(fds.as_mut_ptr(), libc::O_CLOEXEC) } != 0 {
            return Err(eyre!(
                "failed to create strace pipe: {}",
                std::io::Error::last_os_error()
            ));
        }
        // SAFETY: `pipe2` initialised both descriptors, so each is valid and
        // owned by exactly one `OwnedFd`.
        let (read_fd, write_fd) =
            unsafe { (OwnedFd::from_raw_fd(fds[0]), OwnedFd::from_raw_fd(fds[1])) };

        let write_raw = write_fd.as_raw_fd();
        let mut cmd = Command::new("strace");
        cmd.args(STRACE_ARGS);
        cmd.arg("--");
        cmd.args(command);
        cmd.stdin(Stdio::null());
        // The traced process inherits these. They must not be the trace pipe.
        cmd.stdout(Stdio::null());
        cmd.stderr(Stdio::null());

        // SAFETY: this runs in the forked child between fork and exec. `dup2`
        // is async-signal-safe, allocates nothing, and takes effect before
        // `execve` runs strace. The write end is deliberately left open so
        // that the trace pipe stays valid while strace runs.
        unsafe {
            cmd.pre_exec(move || {
                if libc::dup2(write_raw, TRACE_FD) == -1 {
                    return Err(std::io::Error::last_os_error());
                }
                Ok(())
            });
        }

        let child = cmd
            .spawn()
            .map_err(|e| eyre!("failed to spawn strace: {e}"))?;
        // Only strace's copy may hold the write end open. Keeping ours would
        // hold the pipe open forever and the stream would never end.
        drop(write_fd);
        let _ = child;

        let read_file = std::fs::File::from(read_fd);
        let lines = tokio::io::BufReader::new(tokio::fs::File::from_std(read_file)).lines();
        let lines = tokio_stream::wrappers::LinesStream::new(lines).map(|line| match line {
            Ok(line) => Ok(line),
            Err(e) => Err(eyre!("failed to read strace output: {e}")),
        });
        Ok(Self {
            inner: Box::pin(lines),
        })
    }
}

impl Stream for StraceLines {
    type Item = Result<String>;

    fn poll_next(mut self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        self.inner.as_mut().poll_next(cx)
    }
}

impl std::fmt::Debug for StraceLines {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str("StraceLines")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Assert every line is well-formed for the parser, i.e. it carries the
    /// `PID TIMESTAMP` prefix that [`super::parser`] splits on.
    async fn trace(command: &[&str]) -> Vec<String> {
        StraceLines::new(command.iter())
            .expect("spawn strace")
            .collect::<Vec<Result<String>>>()
            .await
            .into_iter()
            .collect::<Result<Vec<String>>>()
            .expect("read trace")
    }

    #[tokio::test]
    async fn every_line_carries_pid_and_timestamp() {
        let lines = trace(&["sh", "-c", "exit 0"]).await;
        assert!(!lines.is_empty(), "no trace output");
        for line in &lines {
            let mut fields = line.splitn(3, ' ');
            let pid = fields.next().expect("no pid");
            pid.parse::<i32>()
                .unwrap_or_else(|e| panic!("bad pid in {line:?}: {e}"));
            let timestamp = fields
                .next()
                .unwrap_or_else(|| panic!("no timestamp in {line:?}"));
            assert_eq!(timestamp.split(':').count(), 3, "bad timestamp in {line:?}");
        }
    }

    #[tokio::test]
    async fn program_output_does_not_contaminate_the_trace() {
        // The traced program writes to both stdout and stderr. If either
        // leaked into the trace stream these lines would not parse.
        let lines = trace(&["sh", "-c", "echo busyna-out; echo busyna-err >&2"]).await;
        for line in &lines {
            assert!(
                line.split(' ').next().unwrap().parse::<i32>().is_ok(),
                "program output leaked into the trace: {line:?}"
            );
        }
    }

    #[tokio::test]
    async fn no_line_is_split_by_concurrent_writes() {
        // Two processes writing simultaneously is what produces torn lines when
        // the trace shares a pipe with program output.
        let lines = trace(&[
            "sh",
            "-c",
            "cat /etc/hostname >/dev/null & ls /etc >/dev/null & wait",
        ])
        .await;
        for line in &lines {
            let complete = line.splitn(3, ' ').count() == 3
                && line
                    .split(' ')
                    .nth(1)
                    .is_some_and(|t| t.split(':').count() == 3);
            assert!(complete, "torn line in trace: {line:?}");
        }
    }

    #[tokio::test]
    async fn follows_the_whole_process_tree() {
        let lines = trace(&["sh", "-c", "sleep 0.1"]).await;
        let pids: std::collections::BTreeSet<i32> = lines
            .iter()
            .filter_map(|line| line.split(' ').next()?.parse().ok())
            .collect();
        assert!(pids.len() >= 2, "expected a forked child, got {pids:?}");
    }

    #[tokio::test]
    async fn decodes_descriptors_into_paths() {
        let lines = trace(&["cat", "/etc/hostname"]).await;
        assert!(
            lines
                .iter()
                .any(|l| l.contains("openat") && l.contains("/etc/hostname")),
            "no openat of the target file in the trace"
        );
    }
}
