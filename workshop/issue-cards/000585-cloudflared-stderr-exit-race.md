---
id: 000585
status: working
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
started: 2026-10-15T18:08:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Preserve cloudflared stderr on early process exit

## Problem

The full recorded suite intermittently loses the fake cloudflared process's stderr failure before reporting its exit. This is independent of #571's Codex adapter; no real tunnel or network is used by this test.
