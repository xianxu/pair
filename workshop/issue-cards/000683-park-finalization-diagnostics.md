---
id: 000683
status: codecomplete
created: 2026-10-18
updated: 2026-10-19
estimate_hours: 3.851
github_issue:
severity: high
started: 2026-10-19T02:08:51-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
actual_hours: 3.48
tracker:
    version: 1
    completion:
        token: close-a180331a1a0b
        repository: github.com/xianxu/pair
        reviewed_head: bda9b0a4e8d157f5763920eecb477dfd2cf3bbf0
        evidence_commit: c77d705d229f3b398b28d01da75ce76aceefe007
---

# Persist every lifecycle failure reason

## Problem

Operator UI relaunches of ariadne:0 and pair:0 failed on 2026-10-18 around 17:49–17:51 PDT. The reported Couch build included #651, #676 and #670. Each cleanup helper published a matching success completion and preserved scrollback, but Couch left the park awaiting_completion and did not publish VerifiedPark. Subsequent resume refused resume-parking. The original UI failure message was not retained with an inspectable operation, and the hub reported no new launch-diagnostics/failures.json entry; the initiating cause could not be established from the surviving evidence.

The operator later recovered through the menu; both outcomes were reported as recovered (Fresh). Read-only inspection found new live tags, while the old records had been archived with their parks still unresolved. Whether the operator selected Fresh or Couch selected it is not established; do not infer an automatic Fresh transition from the label alone.
