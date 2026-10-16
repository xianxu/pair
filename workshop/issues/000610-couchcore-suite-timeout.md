---
id: 000610
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
severity: high
card_mirror: '18fb05e228a6566e7ed77cf87ee44e097126abd9' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T03:54:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "2ad97ea0", done: "2e9ba8a8"}
---

# couchcore package takes ~10 min, at the go test timeout

## Problem

Couchcore takes nearly Go's default ten-minute per-package test timeout. This blocks otherwise verified pair landings under fleet load, including #591 and #589. In #591's recorded integration suite abe0a5ed883f06ffe58305cb45a29c09, couchcore timed out at 600.245 s; every other recipe/package passed. The automatic main comparison 2064bde22de3d40f5d15091fe41ef443 passed, but couchcore alone took 580.585 s. This is a pair item in ops/test-economy; the hub owns that project.

## Spec

Apply the hub's explicit temporary unblock: give the full suite's broad Go invocation an explicit 20m per-package timeout in Makefile.local, with this issue reference and a temporary-budget explanation beside the recipe. Preserve the complete package selection, cache-disabled execution, assertions and fixture deadlines. Profile/split/shard work is a separate unassigned follow-up. No local-gates configuration changes or retries are part of this fix.

## Done when

- [ ] The suite invokes go test with an explicit 20m timeout and retains ./... and -count=1; the Makefile explains the temporary budget with this issue reference.
- [ ] An unassigned test-economy follow-up records profiling, package splitting or sharding, and removal of the temporary allowance after a measured execution budget is restored.
- [ ] Recipe verification and the recorded full suite pass; close review is SHIP and the unblock lands before #591 and #589 resume.

## Plan

- [ ] From main, change only the broad suite recipe and its temporary-budget comment; verify actual Make argv before/after.
- [ ] File the performance follow-up; run build and the recorded suite, close, and land before resuming #591 unchanged.

## Log

### 2026-10-16 — approved fleet unblock

Hub decision 173f498b-9796-4fd6-a777-07e28ce4107c explicitly authorizes 20m as a deliberate temporary budget. #591 remains clean at 47a221c079675acc1415c1a85db3ac559fcb1722 and is not rerun yet. Integrated suite failed only the package-wide timeout; automatic main comparison passed in 946.514 s with couchcore at 580.585 s. The fixed duration is visible at the suite owner (ARCH-DRY), not injected through shared GOFLAGS or hidden retry machinery. Production behavior is unchanged.
