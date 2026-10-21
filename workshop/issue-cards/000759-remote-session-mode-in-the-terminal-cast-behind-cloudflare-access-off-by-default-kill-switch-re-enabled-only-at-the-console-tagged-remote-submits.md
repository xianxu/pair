---
id: 000759
status: working
created: 2026-10-20
updated: 2026-10-21
estimate_hours:
github_issue:
started: 2026-10-21T01:17:25-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# remote-session: 👽 mode in the terminal cast behind Cloudflare Access, off by default, kill switch re-enabled only at the console, tagged remote submits

## Problem

remote-session (ops project, operator-approved 10-20): write access to a session file instructs agents that run with
bypass permissions, so remote access is a security surface. Chrome cast and terminal cast assume the operator is at the
machine; 👽 mode assumes they're not.
