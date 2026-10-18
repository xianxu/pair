---
id: 000676
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
severity: critical
started: 2026-10-18T15:12:17-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# Exact Claude resume fails above native tree metadata cap

## Problem

Claude relaunch on the current main build exits before registration with `native resume unavailable (incomplete): native root inspection incomplete`, while Codex relaunch succeeds. The operator cannot restart Claude hubs safely. Both reported requested transcripts still exist; starting fresh abandoned their prior conversations rather than recovering them.
