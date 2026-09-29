// Copyright (C) 2026 Leandro Lisboa Penz <lpenz@lpenz.org>
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

//! Trace the files that a process tree reads and writes.
//!
//! This is built in two layers. [`runner`] runs a command under
//! [`strace`](https://strace.io) and streams its raw output, one line at a
//! time, without interpreting anything. [`parser`] takes that stream and yields
//! structured [`StraceItem`] values.
//!
//! ```no_run
//! use busyna::strace::StraceLines;
//! use busyna::strace::SyscallStream;
//! use tokio_stream::StreamExt;
//!
//! # async fn run() -> color_eyre::Result<()> {
//! let lines = StraceLines::new(["cargo", "build"])?;
//! let mut syscalls = SyscallStream::new(lines);
//! while let Some(item) = syscalls.next().await {
//!     let item = item?;
//!     println!("[{}] {} {}", item.pid, item.syscall, item.result);
//! }
//! # Ok(())
//! # }
//! ```
//!
//! # Trace format
//!
//! With `-tt -f -o`, every line strace writes has the shape
//!
//! ```text
//! 234762 22:04:36.854676 openat(AT_FDCWD</tmp>, "/etc/hostname", O_RDONLY) = 3</etc/hostname>
//! ```
//!
//! that is, a PID, a `HH:MM:SS.microseconds` timestamp, then the syscall with
//! its arguments in parentheses and its result after `= `. Results are
//! space-padded by strace to keep columns aligned, and may contain parentheses
//! of their own (`-1 ENOENT (No such file or directory)`).
//!
//! # Two things the trace does not give you
//!
//! Paths appear exactly as the program passed them, so the same file can show
//! up as `/usr/include/stdio.h`, `./stdio.h` and `../include/stdio.h`. There is
//! no canonicalisation anywhere in the kernel's syscall interface; resolving
//! them is left to the caller.
//!
//! `stat`, `lstat`, `access` and `readlink` also do not appear. Fanotify cannot
//! substitute for them either: those syscalls are resolved in the kernel and
//! never generate a notification. A compiler resolving an `#include` is mostly
//! a sequence of `stat` calls over include paths, so this is the main gap in
//! any strace-based dependency graph and it must be handled separately.

mod parser;
mod runner;

pub use parser::StraceItem;
pub use parser::SyscallStream;
pub use runner::StraceLines;
