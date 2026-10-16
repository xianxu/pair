---
id: 000592
status: open
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
github_issue:
started: 2026-10-16T08:50:30-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# history-backup: continuous archiving of agent transcripts, Codex databases, operator docs and sdlc data

## Problem

Agent transcripts, Codex databases, ops operator-local docs and sdlc runtime data have no copy anywhere, and Claude
Code prunes transcripts after 30 days by default. Ops project history-backup, design §2.1 "continuous" mode.
