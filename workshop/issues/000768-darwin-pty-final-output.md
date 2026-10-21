---
id: 000768
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '1899fd8ad34e598d428cdf5e47b3caeda7546e3f' # card fields mirrored from issue-cards; edit via sdlc
---

# Preserve short-lived command output across Darwin PTY slave close

## Problem

Public pair scribe -log PATH -- CMD [ARGS...] accepts arbitrary commands (docs/pair-scribe.md; cmd/pair-go/main.go dispatches directly to scribecmd.Run), including short-lived commands that never open their controlling terminal. On macOS these can exit and lose unread PTY output before the first master read. #766 reproduced 10 empty-stdout results in 10000 runs; an instrumented wrapper copy saw 41/5000 first reads return EOF with zero bytes while the child exited successfully. Scribe already drains before cmd.Wait; this is not fixed by waiting longer or retrying reads after EOF.

XNU's last-slave-close path can discard unread output without a controlling-terminal reference. Opening /dev/tty in the child establishes S_CTTYREF and causes exit to drain. Ruby's primary investigation documents the same mechanism and links the kernel paths: https://bugs.ruby-lang.org/issues/20682 ; Apple source: https://github.com/apple-oss-distributions/xnu/blob/main/bsd/kern/tty_tty.c . #766 fixes the test producer only, as approved by the TL; this production exposure remains.

## Spec

Preserve final output for admitted short-lived non-terminal-aware commands under scribe. Audit other generic Pair PTY entrypoints before generalizing. Consider an exec helper that establishes the child's /dev/tty reference before exec, versus an explicitly owned retained-slave/drain lifecycle. Preserve argv, child exit status, signal delivery and terminal behavior; no timeout increase or blind retry.

## Done when

- A deterministic native regression forces the child to write and exit before the first master read and still preserves exact stdout and log bytes.
- Short-lived non-terminal-aware children and normal interactive shell/agent behavior pass, including child exit status, signals and cleanup ownership.
- The design records which Pair PTY entrypoints are covered and any remaining exposure.

## Plan

- [ ]

## Log

Discovered while investigating #766. Public scribe command is direct evidence of production admission; the shipped documented shell hook normally wraps a long-lived /bin/zsh, whose slave reference protects its ordinary subprocesses. Explicit scribe printf and final shell exec remain exposed. File unclaimed per TL instruction; do not expand #766's approved fixture-only scope.
