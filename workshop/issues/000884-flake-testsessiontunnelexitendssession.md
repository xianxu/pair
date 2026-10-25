---
id: 000884
status: working
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: '06d72f07c9ab3b153c48bdbef25e7747403d0d81' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-25T07:59:14-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/broadcast.TestSessionTunnelExitEndsSession]
flow: {kind: quick, provenance: inferred, spec: "7eeb4c81", done: "36a31fc1"}
---

# flake: TestSessionTunnelExitEndsSession

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/broadcast.TestSessionTunnelExitEndsSession: Boundary review round11 for pair#759 on76fb12fb1efc18a8b08e2d3a961e727c10460521 observed stream ended: EOF at session_test.go:171. Same exact baseline9e2fc4fc294beb5a1cbbed3677273fcb794eab84 reproduced39 failures in1000 automatic reviewer repetitions (retained reviewer session01a172e3-6b3b-7112-9af7-9eadc376dc1f, CommandExecution exec-ae854726-8fd9-4d19-bb01-4818be6de86d, BASE_CONTROL_EXIT1). Subsequent bounded single invocation of that baseline passed0.193s exit0: go test ./cmd/internal/broadcast -run ^TestSessionTunnelExitEndsSession$ -count=1. Earlier reviewer100-count baseline also passed. Historical high-count reviewer repeats exceeded the fleet cap and were not repeated by worker. Current #759 full recorded suite PASS12m14s. Evidence workshop/evidence/000759/review-broadcast-flake.md plus committed round11 review. Cause unclassified, separate root-cause follow-up; do not tune timeout. No production/test source change.

## Spec

A tunnel/server loss must retain the exact Host shutdown join through every affected Mount. Session teardown must not revoke its locks or publish Done until that owned shutdown has completed. This preserves the existing opportunity to deliver the final SSE end event and the existing two-second server shutdown bound.

Host.watch will attach the existing hostAttempt.done to each lost Mount before publishing Exited. Mount.Unmount will join that same receipt after registry removal; the last ordinary Unmount will retain its own close receipt for repeated callers too. The receipt belongs to that Host generation and cannot join a later opening/close. No public API, retries, timeout increases, SSE priority heuristic, extra goroutine or persisted format.

ARCH-ORDER/ARCH-FUNERAL: map absence means routing removal, not joined resource teardown. ARCH-DRY: reuse the close attempt already owned by Host. ARCH-SECURE: leave lock rotation/revocation behavior intact and delay only the existing Session teardown's End until its owned close finishes. Work is a shutdown path, O(1) per Unmount and the existing O(mounts) loss fanout; one channel reference per Mount, collected with it.

Internal state | Visible result | Recovery | Loss
---|---|---|---
Live mount | Stream serves | Stop | None
Lost/last-unmounted, shutdown pending | Final end may drain; Done pending | Wait under existing shutdown bound | Slow/dead remote writer retains existing forced-close behavior
Owned shutdown complete | Stream and tunnel ended; Done closes | Start a new session | Existing ended session only
Old Mount after Host reopen | Old receipt already complete | Unmount remains idempotent | No effect on new generation

## Done when

- A deterministic channel-held shutdown regression fails without the join and passes with it, including repeated Unmount and old/new generation isolation.
- Real tunnel-exit SSE and operator-stop tests retain final end/reason, token revocation, joined tunnel cleanup and shared-Host behavior; targeted normal/race runs use count at most10, one at a time.
- Existing cancellation, lock rotation and two-second shutdown behavior remain; no retry or longer timeout. Closing unregisters the test.

## Plan

- [x] Pin lost/last/repeated Unmount shutdown ownership with synctest and the existing held fake handle; observe baseline RED before production code.
- [x] Retain and join the exact existing shutdown receipt; qualify generation isolation and real SSE lifecycle with bounded targeted/race controls.
- [ ] Record the lifecycle rule, inventory audit and final-head verification.

## Log

### 2026-10-25

Source diagnosis: Host.watch currently calls Mount.lose, deletes it from mounts, and starts Host.beginCloseLocked. Session.watch then calls Stop/teardown; Unmount sees absence and returns immediately, and locks.End revokes the SSE gate before the pending end event necessarily drains. Server.events selects closed subscription against canceled request, so some schedules return EOF. The existing teardown comment explicitly requires locks to end after mount shutdown. Proposed correction retains the existing exact close attempt per Mount under Host.mu before Exited becomes observable; approval requested from hub07f05913 before code. Historical high-count reviewer evidence was read but not rerun. #886 is the next dispatched item after884 lands.

Hub1bc491d0 approved exact-attempt receipts and quick flow before code. change-code admitted quick. RED: TestMountUnmountJoinsOwnedShutdown uses synctest with the real Host.watch/Unmount/beginCloseLocked, idle HTTP server and existing channel-held fake handle. Baseline fails deterministically: all3 calls return early after tunnel loss and server loss;2 repeated calls return early after last ordinary Unmount. No network timing or high repetition. It also pins old/new close generation isolation. Baseline test exit1 in0.190s.

GREEN: exact-attempt receipt correction is15 added production lines. New deterministic regression PASS0.215s; seven selected lifecycle tests count10 PASS1.357s, their race count10 PASS28.805s. Final additional generation-completion assertion qualified by complete broadcast package PASS75.037s and focused join race PASS1.244s. Catalog PASS11.539s. Staged reader inventory PASS: the new regression constructs only in-memory Host/HTTP-server/channel state and uses existing fake handles; it reads no repository artifact and needs no reader certificate change. Atlas and lesson record routing-removal versus joined teardown; this semantic lifetime rule is enforced by controlled ordering rather than an unreliable syntactic lint heuristic. No retries, deadline changes or live operator state.

## Revisions

### 2026-10-25 — executable delivery checklist

Removed future close/landing actions from the final Plan checkbox; they remain workflow postconditions, not code deliverables that must be claimed before the close gate. No change to the approved lifetime design or Done-when.
