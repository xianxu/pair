---
id: 000640
status: working
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T09:08:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Native fullscreen chord fixture loses its disposable session during conformance

## Problem

Native TestFullscreenChordZellijLive intermittently loses its disposable Zellij session: list-panes --json --all returns exit 1, "There is no active session!". Seen while qualifying #639, before any toggle on the candidate attempt; reproduced later during the original three-state scenario with the original layoutcmd code and test body. Root cause is unknown. This is a session-loss symptom, distinct from #636's ErrZellijNoReply.
