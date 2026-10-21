---
id: 000768
status: working
created: 2026-10-21
updated: 2026-10-21
estimate_hours: 1.3
github_issue:
started: 2026-10-21T05:20:49-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Preserve short-lived command output across Darwin PTY slave close

## Problem

Public pair scribe -log PATH -- CMD [ARGS...] accepts arbitrary commands (docs/pair-scribe.md; cmd/pair-go/main.go dispatches directly to scribecmd.Run), including short-lived commands that never open their controlling terminal. On macOS these can exit and lose unread PTY output before the first master read. #766 reproduced 10 empty-stdout results in 10000 runs; an instrumented wrapper copy saw 41/5000 first reads return EOF with zero bytes while the child exited successfully. Scribe already drains before cmd.Wait; this is not fixed by waiting longer or retrying reads after EOF.

XNU's last-slave-close path can discard unread output without a controlling-terminal reference. Opening /dev/tty in the child establishes S_CTTYREF and causes exit to drain. Ruby's primary investigation documents the same mechanism and links the kernel paths: https://bugs.ruby-lang.org/issues/20682 ; Apple source: https://github.com/apple-oss-distributions/xnu/blob/main/bsd/kern/tty_tty.c . #766 fixes the test producer only, as approved by the TL; this production exposure remains.
