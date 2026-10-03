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

- [x] Add failing tests (FakeProcOps hook that kills the pid on an Identity call)
- [x] Re-probe Exists in observeExactProcess on an Identity failure

## Log

### 2026-10-03
- 2026-10-03: closed — TestExactProcessReapedBetweenProbesIsDead and TestLivenessReapedBetweenProbesIsDead failed before the fix and pass now; TestExactProcessPresentWithoutIdentityStaysUnknown keeps real uncertainty as Unknown. BR-1 fixed as a class: every Exists-then-Identity copy (Liveness, VerifiedOwner, supervisor observe, orientation target, couchcmd wrapper check) now goes through observeExactProcess. Unsandboxed make -k test: 145 Go packages ok, no FAIL; test-changelog passes with the scratchpad TMPDIR (#360 quirk). no-atlas: a bugfix inside an existing observation, so no new surface.; review verdict: SHIP

- Reported by the operator: global detach errored, and a retry worked. The error
  came at once, not after the 15s timeout, which points to a race rather than a timeout.
- Added a FakeProcOps `ReapedOnIdentity` hook, which removes the pid during the
  identity read. The new test failed before the fix (`observeExactProcess = unknown`)
  and passes after it. The present-without-identity case still returns Unknown.
- Fix (e791b5ce): `observeExactProcess` calls `Exists` again when the identity read
  fails. Every caller of the shared observation gets it.
- In the sandbox, `go test ./cmd/internal/couchcore/` failed only the two ptychild
  tests, which fail there with "operation not permitted" regardless of this change.
- Close round 1 returned FIX-THEN-SHIP with BR-1 (Important): `Couch.Liveness`,
  `VerifiedOwner`, the supervisor observation, the orientation target check and the
  couchcmd message-service wrapper check each repeated the two probes. All of them
  now go through `observeExactProcess` (couchcmd reaches it through `Couch.Liveness`);
  ARCH-DRY. Added `TestLivenessReapedBetweenProbesIsDead`. (22fb27b2)
- Minor, left open: on darwin, a zombie that `kill(pid,0)` still sees might not
  match its identity through sysctl, so it would still read Unknown. Unverified;
  if the symptom comes back, check that first.
