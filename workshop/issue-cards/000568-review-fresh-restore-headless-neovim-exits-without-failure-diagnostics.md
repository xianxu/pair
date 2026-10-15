---
id: 000568
status: codecomplete
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
started: 2026-10-15T07:54:13-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: 0.58
tracker:
    version: 1
    completion:
        token: close-42a1668aac6d
        repository: github.com/xianxu/pair
        reviewed_head: 8d114ada12f6af3cd52a27ad1aaae235f93c6dd8
        evidence_commit: eba91f5a00ea02d6bda1d0c46646f7abc637c033
---

# Review fresh-restore headless Neovim exits without failure diagnostics

## Problem

During pair#556 M1 recorded verification, the unchanged shell test tests/review-fresh-restore-test.sh failed at missing-history-does-not-adopt-old-receipt. The Python assertion saw a nonzero headless Neovim return with empty combined output; its assertion omits the return code, so the retained log cannot distinguish a signal from ordinary exit. The isolated full script immediately passed all five cases on the same code. This is not a confirmed root cause or a #556 recovery regression.
