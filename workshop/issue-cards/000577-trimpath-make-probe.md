---
id: 000577
status: open
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
---

# Trimpath command-line assignment breaks the required suite environment probe

## Problem

After #572, the declared required suite is `make PAIR_GO_TRIMPATH=-trimpath test`. Its Make environment probe fails deterministically before the Go suite: `FAIL: knob PAIR_GO_TRIMPATH did not reach the test recipe`. This blocks #556 PR349 after its mandatory main integration. Bare make test-make-env passes.
