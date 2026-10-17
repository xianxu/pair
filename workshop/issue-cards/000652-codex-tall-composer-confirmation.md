---
id: 000652
status: working
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T15:33:36-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Codex tall composer misses submission confirmation beyond 20 rows

## Problem

The Codex composer recognizer scans at most 20 rows above the cursor (`codexComposerMaxRows`, `cmd/internal/wrapcmd/composer_recognizers.go`). A tall pasted composer is therefore Unknown even when its visible bold prompt and continuation rows are intact. During an already active turn, the wrapper writes the paste and submit, then sees a genuine empty composer but never sets `sawOccupied`; `turnAtSubmit` was already true, so neither confirmation branch succeeds. It reports Indeterminate after the existing 30-second delivery window despite the input submission. A legacy 26-line singleton reproduces this on main; #648 batching amplified the same preexisting issue.
