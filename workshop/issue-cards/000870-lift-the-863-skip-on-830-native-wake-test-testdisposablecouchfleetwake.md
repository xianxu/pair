---
id: 000870
status: open
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
github_issue:
started: 2026-10-24T07:08:58-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# Lift the #863 skip on #830 native wake test (TestDisposableCouchFleetWake)

## Problem

#830's native acceptance test `TestDisposableCouchFleetWake` (cmd/couch/fleet_acceptance_test.go, in `make test-couch-fleet`) is skipped with a reason naming pair#863. #863 landed as PR 586 at 109986901: fresh-namespace startup now starts both configured hubs automatically, while a known prior shutdown still keeps them parked. The skip is now the only thing stopping the run (TL: pair:6 owns lifting it).
