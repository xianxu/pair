---
id: 000577
status: codecomplete
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
severity: high
started: 2026-10-15T13:11:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
actual_hours: N/A
tracker:
    version: 1
    completion:
        token: close-6463b861df83
        repository: github.com/xianxu/pair
        reviewed_head: 41e851b3f87dbb98701ff8849f2fb886c8df7c02
        evidence_commit: c5ca40fe73399b6ac15935b3832b769335be2899
---

# Trimpath command-line assignment breaks the required suite environment probe

## Problem

After #572, the declared required suite is `make PAIR_GO_TRIMPATH=-trimpath test`. Its Make environment probe fails deterministically before the Go suite: `FAIL: knob PAIR_GO_TRIMPATH did not reach the test recipe`. This blocks #556 PR349 after its mandatory main integration. Bare make test-make-env passes.
