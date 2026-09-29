// Copyright (C) 2026 Leandro Lisboa Penz <lpenz@lpenz.org>
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

//! Parse a stream of raw strace lines into structured syscall items.
//!
//! [`SyscallStream`] turns the [`String`] lines produced by
//! [`super::runner::StraceLines`] into [`StraceItem`] values. See the
//! [module documentation](super) for the trace format being parsed.

use color_eyre::Result;
use color_eyre::eyre::eyre;
use std::collections::HashMap;
use std::pin::Pin;
use std::task::Context;
use std::task::Poll;
use tokio_stream::Stream;

/// A single syscall parsed out of a strace trace.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct StraceItem {
    /// Timestamp in the `HH:MM:SS.microseconds` format emitted by `strace -tt`.
    ///
    /// Kept as text rather than parsed into a duration because strace emits it
    /// without a date or epoch base, so it only orders events within a single
    /// run and across a day boundary.
    pub timestamp: String,
    /// PID of the process that made the syscall.
    pub pid: i32,
    /// Name of the syscall, for example `openat`.
    pub syscall: String,
    /// Arguments between the parentheses, verbatim, including any
    /// `-y`-decoded descriptor paths such as `3</etc/hostname>`.
    pub arguments: String,
    /// Return value and any errno, verbatim, for example `0` or
    /// `-1 ENOENT (No such file or directory)`.
    pub result: String,
}

impl StraceItem {
    /// Return the result parsed as a signed integer, if it is a plain number.
    ///
    /// Returns [`None`] for error results (`-1 ENOENT ...`) and for values
    /// that are not decimal integers, such as pointers (`0x7f...`) or strace's
    /// `?` marker for restarted syscalls.
    pub fn result_as_i64(&self) -> Option<i64> {
        let value = self.result.trim();
        // An error result carries errno text (`-1 ENOENT (No such file...)`)
        // and is not a plain integer.
        if self.is_error() {
            return None;
        }
        // Reject strace's `?` marker, hex pointers, and descriptor decorations
        // such as `3</etc/hostname>`, none of which are decimal integers.
        if value.is_empty() || !value.bytes().all(|b| b.is_ascii_digit()) {
            return None;
        }
        value.parse().ok()
    }
    /// Whether this syscall failed, i.e. returned a negative errno.
    pub fn is_error(&self) -> bool {
        self.result.starts_with("-1")
    }
}

/// A [`Stream`] that parses raw strace lines into [`StraceItem`] values.
///
/// strace splits a syscall across two lines whenever another process's output
/// interleaves with it:
///
/// ```text
/// 12811 rt_sigprocmask(SIG_SETMASK, [] <unfinished ...>
/// 12812 <... rt_sigprocmask resumed>, NULL, 8) = 0
/// ```
///
/// This stream buffers the first half per PID and emits a single item when the
/// matching `resumed` line arrives, so consumers never see a half-syscall.
///
/// Lines that are not syscalls, such as `+++ exited with 0 +++` or
/// `--- SIGCHLD ... ---`, are skipped.
pub struct SyscallStream<S> {
    inner: Pin<Box<S>>,
    pending: HashMap<i32, PendingSyscall>,
}

impl<S> SyscallStream<S>
where
    S: Stream<Item = Result<String>> + Send + 'static,
{
    /// Wrap a stream of raw strace lines, yielding parsed [`StraceItem`] values.
    pub fn new(lines: S) -> Self {
        Self {
            inner: Box::pin(lines),
            pending: HashMap::new(),
        }
    }

    /// Number of syscalls whose `unfinished ...` half is still buffered.
    pub fn pending_len(&self) -> usize {
        self.pending.len()
    }
}

impl<S> Stream for SyscallStream<S>
where
    S: Stream<Item = Result<String>> + Send + 'static,
{
    type Item = Result<StraceItem>;

    fn poll_next(mut self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        let this = &mut *self;
        loop {
            match this.inner.as_mut().poll_next(cx) {
                Poll::Pending => return Poll::Pending,
                Poll::Ready(None) => return Poll::Ready(None),
                Poll::Ready(Some(Err(e))) => return Poll::Ready(Some(Err(e))),
                Poll::Ready(Some(Ok(line))) => match this.handle_line(&line) {
                    // Not a syscall, or a syscall awaiting its `resumed` half.
                    None => continue,
                    Some(item) => return Poll::Ready(Some(item)),
                },
            }
        }
    }
}

/// A syscall whose opening arguments have been seen but whose result has not.
struct PendingSyscall {
    timestamp: String,
    pid: i32,
    syscall: String,
    arguments: String,
}

impl<S> SyscallStream<S>
where
    S: Stream<Item = Result<String>>,
{
    /// Process one raw line, returning an item to emit, if any.
    fn handle_line(&mut self, line: &str) -> Option<Result<StraceItem>> {
        let line = line.trim_end();
        if line.is_empty() {
            return None;
        }
        match parse_line(line) {
            Ok(Parsed::Syscall(item)) => Some(Ok(item)),
            Ok(Parsed::Skip) => None,
            Ok(Parsed::Unfinished {
                pid,
                timestamp,
                syscall,
                arguments,
            }) => {
                self.pending.insert(
                    pid,
                    PendingSyscall {
                        timestamp,
                        pid,
                        syscall,
                        arguments,
                    },
                );
                None
            }
            Ok(Parsed::Resumed { pid, tail }) => {
                let Some(pending) = self.pending.remove(&pid) else {
                    return Some(Err(eyre!(
                        "strace resumed a syscall for pid {pid} with no pending half: {line:?}"
                    )));
                };
                let Some((tail_args, result)) = split_arguments_result(&tail) else {
                    return Some(Err(eyre!("no result in resumed strace line {line:?}")));
                };
                Some(Ok(StraceItem {
                    // The timestamp of the opening half is the syscall entry
                    // time, which is the more useful of the two.
                    timestamp: pending.timestamp,
                    pid: pending.pid,
                    syscall: pending.syscall,
                    arguments: format!("{}{}", pending.arguments, tail_args),
                    result: result.to_owned(),
                }))
            }
            Err(e) => Some(Err(e)),
        }
    }
}

/// The outcome of parsing a single raw trace line.
#[derive(Debug)]
enum Parsed {
    Syscall(StraceItem),
    /// A non-syscall line, or one half of a split syscall.
    Skip,
    Unfinished {
        pid: i32,
        timestamp: String,
        syscall: String,
        arguments: String,
    },
    Resumed {
        pid: i32,
        tail: String,
    },
}

fn parse_line(line: &str) -> Result<Parsed> {
    // `PID TIMESTAMP REST`
    let (pid, timestamp, rest) = split_pid_timestamp(line)?;
    // Non-syscall lines: `+++ exited with 0 +++`, `+++ killed by SIGTERM +++`,
    // `--- SIGCHLD {...} ---`, and strace's own diagnostics.
    if rest.starts_with("+++") || rest.starts_with("---") || rest.starts_with("strace:") {
        return Ok(Parsed::Skip);
    }
    if rest.starts_with('<') {
        return parse_resumed(pid, rest);
    }
    // A detached or attached marker can also appear on its own.
    if !rest.contains('(') {
        return Ok(Parsed::Skip);
    }
    let open = rest.find('(').expect("checked above");
    let syscall = rest[..open].to_owned();
    let body = &rest[open + 1..];

    if let Some(head) = body.strip_suffix(" <unfinished ...>") {
        return Ok(Parsed::Unfinished {
            pid,
            timestamp: timestamp.to_owned(),
            syscall,
            arguments: head.to_owned(),
        });
    }

    let (arguments, result) =
        split_arguments_result(body).ok_or_else(|| eyre!("no result in strace line {line:?}"))?;
    Ok(Parsed::Syscall(StraceItem {
        timestamp: timestamp.to_owned(),
        pid,
        syscall,
        arguments: arguments.to_owned(),
        result: result.to_owned(),
    }))
}

/// Parse the `<... NAME resumed>TAIL` half of a split syscall.
///
/// The name is already known from the opening half, so only the tail is
/// returned; [`SyscallStream`] pairs the two by PID.
fn parse_resumed(pid: i32, rest: &str) -> Result<Parsed> {
    let marker = " resumed>";
    let end = rest
        .find(marker)
        .ok_or_else(|| eyre!("malformed resumed strace line for pid {pid}"))?;
    let tail = &rest[end + marker.len()..];
    // `rest` here is just `<... NAME resumed>TAIL`; `tail` starts with a `,`
    // when arguments remain, or with `)` when the call had none.
    Ok(Parsed::Resumed {
        pid,
        tail: tail.to_owned(),
    })
}

/// Split `PID TIMESTAMP REST` into its parts.
fn split_pid_timestamp(line: &str) -> Result<(i32, &str, &str)> {
    let (pid, rest) = line
        .split_once(' ')
        .ok_or_else(|| eyre!("no timestamp in strace line {line:?}"))?;
    let pid = pid
        .parse::<i32>()
        .map_err(|e| eyre!("invalid pid in strace line {line:?}: {e}"))?;
    let (timestamp, rest) = rest
        .split_once(' ')
        .ok_or_else(|| eyre!("missing syscall in strace line {line:?}"))?;
    Ok((pid, timestamp, rest))
}

/// Split the text between an opening and closing parenthesis into arguments and
/// result.
///
/// The closing parenthesis is located by scanning while tracking nesting depth
/// and skipping over double-quoted strings, rather than by searching for the
/// last `) = `. A search for `) = ` is not sufficient, because both arguments
/// and results may contain that sequence: with `-y` a decoded path can contain
/// it (`close(4</tmp/a) = b>)`), and results carry errno text
/// (`-1 ENOENT (No such file or directory)`).
fn split_arguments_result(body: &str) -> Option<(&str, &str)> {
    // `body` starts just after the opening parenthesis, so it opens at depth 1.
    let mut depth = 1usize;
    let mut in_string = false;
    let mut escaped = false;
    let mut in_path = false;
    let mut close = None;
    for (i, c) in body.char_indices() {
        if in_string {
            if escaped {
                escaped = false;
            } else if c == '\\' {
                escaped = true;
            } else if c == '"' {
                in_string = false;
            }
            continue;
        }
        if in_path {
            // strace wraps `-y`-decoded paths in angle brackets, and such a path
            // may contain unbalanced parentheses: `close(3</tmp/a) = b>)`. Skip
            // the whole span so its parentheses are not counted.
            if c == '>' {
                in_path = false;
            }
            continue;
        }
        match c {
            '"' => in_string = true,
            // `->` and `<` also appear in decoded structures such as
            // `[{fd=3, events=POLLIN}]`; only a `<` starting a decoded path
            // matters here, and treating every `<...>` span as opaque is safe
            // because a call's arguments never nest another call's arguments.
            '<' => in_path = true,
            '(' | '[' | '{' => depth += 1,
            ')' | ']' | '}' => {
                depth -= 1;
                if depth == 0 {
                    close = Some(i);
                    break;
                }
            }
            _ => {}
        }
    }
    let close = close?;
    let arguments = body[..close].trim_end();
    // After the closing parenthesis, strace pads and then writes `= result`.
    let result = body[close + 1..].trim();
    let result = result.strip_prefix('=')?;
    Some((arguments, result.trim()))
}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio_stream::StreamExt as _;

    fn item(line: &str) -> StraceItem {
        match parse_line(line) {
            Ok(Parsed::Syscall(item)) => item,
            other => panic!("expected a syscall, got {other:?}"),
        }
    }

    #[test]
    fn parses_pid_timestamp_syscall_arguments_and_result() {
        let item = item(
            "234762 22:04:36.854676 execve(\"/usr/bin/sh\", [\"sh\", \"-c\", \"ls\"], 0x7fff) = 0",
        );
        assert_eq!(item.pid, 234762);
        assert_eq!(item.timestamp, "22:04:36.854676");
        assert_eq!(item.syscall, "execve");
        assert_eq!(
            item.arguments,
            "\"/usr/bin/sh\", [\"sh\", \"-c\", \"ls\"], 0x7fff"
        );
        assert_eq!(item.result, "0");
    }

    #[test]
    fn parses_decoded_descriptor_paths() {
        let item = item(
            "1 1.000001 openat(AT_FDCWD</tmp>, \"/etc/hostname\", O_RDONLY) = 3</etc/hostname>",
        );
        assert_eq!(item.syscall, "openat");
        assert_eq!(
            item.arguments,
            "AT_FDCWD</tmp>, \"/etc/hostname\", O_RDONLY"
        );
        assert_eq!(item.result, "3</etc/hostname>");
    }

    #[test]
    fn result_may_contain_parentheses() {
        let item = item(
            "1 1.000001 openat(AT_FDCWD, \"/nope\", O_RDONLY) = -1 ENOENT (No such file or directory)",
        );
        assert_eq!(item.result, "-1 ENOENT (No such file or directory)");
        assert!(item.is_error());
        assert_eq!(item.result_as_i64(), None);
    }

    #[test]
    fn handles_column_padding_on_short_results() {
        let item = item("1 1.000001 brk(NULL)         = 0x55cad9fb5000");
        assert_eq!(item.syscall, "brk");
        assert_eq!(item.arguments, "NULL");
        assert_eq!(item.result, "0x55cad9fb5000");
    }

    #[test]
    fn result_as_i64_rejects_non_decimal_values() {
        let pointer = item("1 1.000001 mmap(NULL, 8192) = 0x7f3386742000");
        assert_eq!(pointer.result_as_i64(), None);
        // strace marks a restarted syscall with `?`.
        let restarted = item("1 1.000001 poll([{fd=3}], 1, -1) = ?");
        assert_eq!(restarted.result_as_i64(), None);
        // A decorated descriptor result is not a plain integer.
        let decorated = item("1 1.000001 close(3</etc/hostname>)    = 0");
        assert_eq!(decorated.result_as_i64(), Some(0));
        let fd =
            item("1 1.000001 openat(AT_FDCWD, \"/etc/hostname\", O_RDONLY) = 3</etc/hostname>");
        assert_eq!(fd.result_as_i64(), None);
        let ok = item("1 1.000001 read(3</etc/hostname>, \"hi\", 2) = 2");
        assert_eq!(ok.result_as_i64(), Some(2));
    }

    /// A `-y` decoded path may itself contain `) = `, which breaks a naive
    /// "split on the last `) = `" parser.
    #[test]
    fn arguments_may_contain_paren_equals_spacing() {
        let item = item("25557 23:27:20.138043 close(4</tmp/a) = b>)    = 0");
        assert_eq!(item.syscall, "close");
        assert_eq!(item.arguments, "4</tmp/a) = b>");
        assert_eq!(item.result, "0");
    }

    #[test]
    fn recognizes_split_syscall_opening_half() {
        match parse_line("12811 22:04:36.660793 rt_sigprocmask(SIG_SETMASK, [] <unfinished ...>")
            .unwrap()
        {
            Parsed::Unfinished {
                pid,
                syscall,
                arguments,
                ..
            } => {
                assert_eq!(pid, 12811);
                assert_eq!(syscall, "rt_sigprocmask");
                assert_eq!(arguments, "SIG_SETMASK, []");
            }
            other => panic!("expected unfinished, got {other:?}"),
        }
    }

    #[test]
    fn skips_non_syscall_lines() {
        for line in [
            "14918 22:04:36.196776 +++ exited with 0 +++",
            "14917 22:04:36.196877 --- SIGCHLD {si_signo=SIGCHLD, si_pid=14918} ---",
            "14917 22:04:36.194569 <... vfork resumed>) = 14918",
        ] {
            assert!(
                matches!(parse_line(line), Ok(Parsed::Skip | Parsed::Resumed { .. })),
                "expected {line:?} to be skipped or a resumed half"
            );
        }
    }

    #[test]
    fn rejects_malformed_lines() {
        assert!(parse_line("").is_err());
        assert!(parse_line("not a strace line").is_err());
        assert!(parse_line("abcdef 22:04:36.854676 echo").is_err());
    }

    /// Reassemble a full stream and confirm split syscalls become single items.
    #[tokio::test]
    async fn reassembles_split_syscalls() {
        let lines = vec![
            Ok("12811 22:04:36.660793 rt_sigprocmask(SIG_SETMASK, [] <unfinished ...>".to_owned()),
            Ok("12812 22:04:36.660816 write(1</dev/null>, \"x\", 1) = 1".to_owned()),
            Ok("12811 22:04:36.660888 <... rt_sigprocmask resumed>, NULL, 8) = 0".to_owned()),
        ];
        let items: Vec<StraceItem> = SyscallStream::new(tokio_stream::iter(lines))
            .collect::<Vec<Result<StraceItem>>>()
            .await
            .into_iter()
            .collect::<Result<Vec<StraceItem>>>()
            .expect("parse");
        // The split syscall must appear exactly once, not twice or zero times.
        assert_eq!(items.len(), 2, "got {items:?}");
        let sigmask = items
            .iter()
            .find(|i| i.syscall == "rt_sigprocmask")
            .unwrap();
        assert_eq!(sigmask.result, "0");
        assert_eq!(sigmask.pid, 12811);
    }

    /// End-to-end: a real traced process must yield only well-formed items.
    #[tokio::test]
    async fn parses_a_real_trace() {
        let lines = super::super::runner::StraceLines::new([
            "sh",
            "-c",
            "cat /etc/hostname > /dev/null; ls /etc > /dev/null",
        ])
        .expect("spawn strace");
        let mut items = SyscallStream::new(lines);
        let mut seen_openat = false;
        let mut count = 0;
        while let Some(item) = items.next().await {
            let item = item.expect("parse line");
            assert!(!item.syscall.is_empty());
            assert!(!item.timestamp.is_empty());
            count += 1;
            if item.syscall == "openat" && item.arguments.contains("/etc/hostname") {
                seen_openat = true;
            }
        }
        assert!(seen_openat, "no openat of /etc/hostname");
        assert!(count > 10, "suspiciously few items: {count}");
        assert_eq!(items.pending_len(), 0, "syscalls left buffered at end");
    }
}
