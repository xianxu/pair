---
id: 000389
status: working
deps: []
github_issue:
created: 2026-10-03
updated: 2026-10-03
estimate_hours:
card_mirror: 'f39b123c75914dd6ca9a17ebe981e421ff257525' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-03T11:53:54-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "4a04ace5", done: "1093ba85"}
---

# couch detach fails when client exits between liveness probes

## Problem

Couch global detach intermittently fails with
`error: leave couch: detach couch-…: cannot observe whether pid 13090 exited`;
an immediate retry succeeds.

`observeExactProcess` (`cmd/internal/couchcore/couch.go`) checks liveness with two
separate probes: `kill(pid, 0)`, then a `sysctl kern.proc.pid` read of the process
start time. When the SIGTERMed client is reaped between those probes, the pid exists
for the first probe but has no identity for the second, so the check reports
`Unknown`. `awaitExactProcessExit` (`detach.go`) treats `Unknown` as an immediate
failure, so detach aborts at the very moment the client finishes exiting.

## Spec

When the identity probe fails, `observeExactProcess` re-runs `Exists`. If the pid is
now gone (ESRCH), the result is `Dead`. It returns `Unknown` only while the pid still
reports present without an identity. Real uncertainty keeps the existing
"leave it alone" rule. The fix sits in the shared observation, so every caller
(detach, inventory, liveness) gets it, not just detach.

## Done when

- A FakeProcOps test where the pid dies between the `Exists` and `Identity` probes
  shows `observeExactProcess` returning `Dead` and `awaitExactProcessExit` returning nil.
- A test where the pid stays present with no identity still returns `Unknown`.
- `go test ./cmd/internal/couchcore/...` passes.

## Plan

- [ ] Add failing tests (FakeProcOps hook that kills the pid on an Identity call)
- [ ] Re-probe Exists in observeExactProcess on an Identity failure

## Log

### 2026-10-03

- Reported by the operator: global detach errored, and a retry worked. The error
  came at once, not after the 15s timeout, which points to a race rather than a timeout.
