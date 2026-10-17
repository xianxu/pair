# Interrupted Park Recovery Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** An interrupted relaunch preserves its conversation and has a named, executable recovery path instead of requiring Start fresh.

**Architecture:** Keep the existing park transaction, worker, completion artifacts and native-resume authority. Add a pure recovery decision and a thin confirmed recovery operation that finishes the exact transaction or safely closes interrupted bookkeeping when its exact owner and terminal are proved gone. Before the initiating worker has attempted publication, roll back a failed/canceled begin without touching the live child. Never turn missing completion or incomplete scrollback into verified cleanup success.

**Tech Stack:** Go, journaled ThreadStore, Pair lifecycle artifacts, existing stateful process/terminal/lifecycle fakes, Couch typed operations/socket receipts/TUI.

## Evidence and scope

The issue Log records eighteen isolated reproductions at source6df6562411120a83c21262ffc95dfe6e7fa496d6. Raw source/log/hash manifest is retained under /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair651-repro.6hqyct. Diagnostic fault source is not implementation. Original pair:6 failure ordering remains unproved; do not claim the fixture reconstructs it.

Existing behavior already preserves identity and refuses unsafe starts. The defect combines incomplete outcome classification, inaccessible recovery, and no resolution of a dead owner whose completion is missing/failed. Preserve those guards while resolving that class. No automatic fresh conversation, clearing user input, process termination, wrapper restart, background retry daemon or replay of a whole relaunch.

## Alternatives

1. **Recommended: converge the existing park, expose recovery.** Smallest authority boundary: existing per-address worker + operation gate + journal CAS. Handles pre-effect rollback and post-effect uncertainty explicitly.
2. Only expose existing Retry/Recover. Insufficient: reproduced absent-terminal/missing-completion and failed-cleanup cases still refuse forever.
3. Automatically abandon every failed park, or run Fresh. Rejected: an uncertain publication can still stop a live child; missing acknowledgement is not proof of no effect. Fresh unnecessarily discards the conversation. Uncanceling all work could execute a delayed quit after the caller thought it stopped.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| ParkRecoveryDecision | cmd/internal/couchcore/park_recovery.go | new |
| ParkRecoveryOutcome / ParkRecoveryResult | cmd/internal/couchcore/park_recovery.go | new |
| RelaunchOutcome classification | cmd/internal/couchcore/relaunch.go | modified |

- ParkRecoveryDecision consumes explicit observed facts: current park/attempts, matching completion disposition, exact owner status, exact terminal presence and pre-publication authority. It chooses complete, drive-existing, roll-back-unpublished, retire-interrupted, already-settled, or refuse. Unit tests are pure; no filesystem/process fakes in those tests.
- Recovery result distinguishes successful verified park from interrupted cleanup retired and already settled. Include retained diagnostic/scrollback uncertainty and next action. It does not implement StartedChild and cannot implicitly launch a replacement. Unresolved errors retain partial progress.
- Relaunch reuses existing outcome values where truthful: refusal before mutation, park incomplete only while actually open, park-ok-resume-failed after durable finalization. A rolled-back begin must report that the original child remains rather than asserting no record write happened. If this needs a new outcome, add the explicit rolled-back outcome and all receipt consumers; do not overload RefusedBeforePark's unchanged-record promise.
- Existing ParkPhase and persisted schema remain unchanged. Reuse AdvanceParkTransaction and ParkAbandoned for a closed historical interruption; never set a VerifiedPark for an aborted transaction. No lifecycle CUE model for park exists; persisted legality is threadrecord/lifecycle.go and is read before changing its rules.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| PairLifecycleController park recovery | cmd/internal/couchcore/park.go, park_recovery.go | modified | lifecycle request/completion store, terminal and exact process probes |
| ThreadStore interrupted-park retirement | cmd/internal/couchcore/threadstore.go | new | single journaled revision/identity CAS |
| Couch recover-park operation | cmd/internal/couchcore/ops.go, operationdispatch.go, slot_operation.go | new | existing per-thread admission and operation execution |
| Confirmed slot command/receipt | cmd/internal/couchcmd/cli.go, slot_operations.go, cmd/internal/couchmessage | modified | authenticated owning-hub socket protocol |
| Thread recovery menu/advice | cmd/internal/couchtty/menu_actions.go and operation consumers | modified | existing actor-action authority and confirmation UI |

- Reuse envWithLiveThread, fakeControllerLifecycle, pairlifecycletest.Fake, fake ProcOps and artifact/session observer; do not introduce a second teardown model. Portable backing is the existing isolated namespace folder and actual journaled ThreadStore. Promote relevant diagnostic cases to permanent tests after change-code.
- The store operation closes the exact park and retires only its exact dead recorded incarnation in one journal transaction. Reuse/extract existing AbandonPark/RetireIncarnation guards, not parallel validation. Its authority is the controller's screened evidence plus exact revision/identity CAS; it has no direct process probing. Preserve launch profile, native binding, session references, history and incomplete-cleanup diagnostics.
- If a foreign/replacement incarnation or start claim exists, reuse inspectLifecycleDebris screening and refuse before any historical tombstone. Never abandon first then discover retirement is forbidden.

## State, events and invariants

| Durable state / evidence | Event or observation | Permitted effect / result |
|---|---|---|
| no park, original live child | failure before BeginPark commits | unchanged child; no claim of a stop |
| newly begun requested, initiating worker has not attempted PublishRequest | cancellation or definite failure | journal rollback of park bookkeeping; original incarnation untouched |
| requested/awaiting/unknown, request may be published | cancellation, lost reply or crash | retain exact transaction; stop at caller bounds; name recover-park |
| open transaction, matching successful completion, exact child dead | recover or restart reconciliation | existing FinalizePark once, then artifact cleanup |
| matching completion but child live/unknown | recover | bounded wait/refusal; never native resume or retire |
| open transaction, exact terminal positively present | confirmed recover | existing Recover drives/reconciles same nonce/attempt rules within existing wait budget |
| open transaction, exact terminal positively absent and exact owner dead, cleanup missing/failed | confirmed recover | atomic close/retire with interrupted-cleanup result; no VerifiedPark; preserve artifacts and native conversation |
| terminal unknown, owner unknown/live, foreign ownership or replacement | recover | actionable refusal, no state destruction; repeat same command only after inspection/changed evidence |
| finalization journal durable but reply lost | reopen/read current store | roll forward existing journal; report actual settled state, not ParkIncomplete |
| park already finalized or retired | repeated recover | already-settled result, no trigger/launch/deletion |
| late completion after aborted transaction | observe | existing closed/tombstoned history ignores it; cannot retire or overwrite a newer incarnation |

**Single resource / ownership:** Existing worker entry reserves one address through completion; all recovery paths join by the retained nonce or wait for that exact work to settle. The operation gate protects admission and the store CAS prevents a stale actor from mutating a replacement. Do not release the worker reservation before any rollback/reconciliation effect settles. A returned canceled caller can precede worker settlement; its message must describe that uncertainty, not assert the child stopped.

**Bounded extent:** Keep existing cooperative cancellation and bounded waits. Do not use an unbounded Background/WithoutCancel continuation. Rollback bookkeeping after caller cancellation uses a separately bounded cleanup scope whose lifetime ends before the worker releases its address. If cleanup cannot commit, retain the park and return the recovery command. Controller shutdown/crash leaves durable state for explicit recovery; startup inspection does not blindly send another quit.

**Irreversibility:** Retirement after cleanup began rolls back lifecycle bookkeeping, not filesystem effects. Successfully preserved scrollback remains; a failure to preserve scrollback is reported as unknown/incomplete, with retained artifact paths. Native conversation is independently verified by the normal subsequent resume. A failed cleanup cannot be relabeled a successful park.

**Public contract:** Add couch --recover-park repo:N --confirm [--json], authenticated with the same owning-hub and current-target checks as destructive slot recovery. One typed operation feeds both the remote command and a confirmed “Recover park” menu action on an open-park row; ordinary threads use that menu's exact scope/tag. It resolves or refuses the existing transaction and never resumes automatically. On resolved result, advise couch --resume repo:N (or Enter for a local row). Exact returned receipt IDs retain normal operation-status semantics; unknown receipt is not a failure or a resend instruction. Don't expose ordinary park, retry or abandon as additional public destructive verbs.

**Advice sweep:** relaunch error, resume_route open-park refusal, recovery_prepared open-park refusal, controller recovery failures, show/list row advice and final operation receipt all derive from one guidance formatter. Slot targets get the executable command; non-slot rows get the real Recover park action. Errors that cannot identify a target must not print a placeholder command as if executable.

## Operating envelope and residue

ARCH-CONSTRAINTS: operator-triggered lifecycle recovery, no keystroke hot path. One active park per address; reuse LifecycleParallelism admission and existing CompletionTimeout/RelaunchParkWait rather than inventing retry loops. A recovery may inspect all attempts already retained by the transaction as existing Recover does, but creates at most the existing permitted next attempt, never repeated submissions on a timer. Actual native waits remain bounded; on timeout retain state and return precise diagnostics. No per-output counters, buffers or goroutines. Network/load throughput is irrelevant to the decision; native probe uncertainty is an explicit refusal.

ARCH-FUNERAL: no new persisted sidecar family or background process. Closed park history uses existing retention rules and unchanged per-attempt shape; successful completion cleanup remains existing ownership. Do not eagerly remove missing/failed-completion evidence when retiring an interruption; leave it under existing owner retention: artifactpath/gc.go lifecycleCandidates enumerates request/completion/trigger/lock families; storagegc.Collector removes an eligible owner bundle using storagegc.Decide. Visible threads are protected; otherwise proven-dead owners become eligible after the existing 365-day RetentionPeriod and complete ownership/reference inventory. Add a regression that the interrupted historical bundle stays enumerated and protected while visible, then is collectible under the existing policy after archival; no separate TTL is introduced. In-memory decisions/results die with their operation; receipt map retains its existing cap/expiry. Tests own and clean private directories only after their worker futures settle.

ARCH-PURE/DRY: one decision core and guidance formatter, existing transition function/store journal/worker/ownership checks reused. ARCH-SECURE: input records and uncertain observations are not trusted absence; confirmation is not ownership proof. ARCH-ORDER: journal/identity/publication uncertainty and all late events are first-class. ARCH-PURPOSE: cover missing/failed completion and all named callers, not only adding words to the refusal. ARCH-MOCK: stateful fakes plus opt-in real helper conformance; no function-call count as the only recovery oracle.

## Task 1: Permanent interruption proofs and recovery core

**Files:** cmd/internal/couchcore/park_recovery.go, park_recovery_test.go, relaunch_test.go, park.go, threadstore.go, storejournal_test.go; existing lifecycle fakes/helpers.

- [ ] Promote the retained twelve-boundary reproduction into permanent independent assertions. Add every cleanup-stage failure and actual context cancellation; distinguish fake fault injection from native interruption.
- [ ] Add failing regressions for pre-publication rollback, missing-completion/dead-owner recovery, failed scrollback cleanup, repeated recovery, delayed success and journal roll-forward. Assert exact native binding/profile preserved, no second child/trigger, no fake VerifiedPark, truthful receipt and retained evidence.
- [ ] Add adversarial cases: owner unknown/live, terminal query failure, replaced PID identity, foreign park, zero/multiple incarnations, revision race, competing start, coalesced recovery, canceled caller while old worker remains, late completion after retirement and next incarnation.
- [ ] Run targeted red cases and retain output; implement pure decision + thin recovery/atomic store operation with existing transitions. Re-run green and journal-fault cases including crash between intent and each target write. Verify interrupted artifacts remain discoverable by lifecycleCandidates and the owner-bundle collector under the unchanged retention policy.
- [ ] Correct relaunch classification from fresh durable state after errors; test before-begin failure, canceled return while worker active, finalized journal loss and cleanup-time cancellation. No state claim based only on return error.
- [ ] Run go test ./cmd/internal/couchcore -run 'Park|Relaunch|LifecycleDebris|Resume' -count=1; commit code/proof.

## Task 2: Public recovery and actionable guidance

**Files:** cmd/internal/couchcore/ops.go, operationdispatch.go, slot_operation.go, actor-action projection/advice, resume_route.go, recovery_prepared.go; cmd/internal/couchcmd/cli.go, slot_operations.go, help/skill sources; cmd/internal/couchmessage protocol declaration/validation; cmd/internal/couchtty/menu_actions.go and execution; corresponding colocated tests.

- [ ] Add failing CLI parse/confirmation, authenticated socket/receipt and row-menu tests for recover-park. Test missing confirmation, wrong hub, ambiguous/unknown slot, changed current target, unknown receipt and duplicate intent. Derive command reachability/confirmation from declarations, not another independent allowlist.
- [ ] Wire the single typed operation to existing admission/queue; never treat a recovery result as StartedChild. Add exact scope/tag menu admission; offer recovery only when the durable park is open, including unusable/busy-looking rows whose worker can be joined safely.
- [ ] Centralize advice and sweep each open-park refusal/partial receipt. Test advertised command through production ParseCLI → socket → dispatch → controller and confirm it resolves the reproduced dead end. Test ordinary-thread menu path separately.
- [ ] Verify a second recover is harmless, and resume uses normal binding/owner checks after recovery without creating a new conversation. Update atlas/index link only if adding a page; update atlas/couch.md, user-facing help/skill and README where behavior changes.
- [ ] Run go test ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchmessage ./cmd/internal/couchtty -run 'Park|Relaunch|SlotOperation|Resume|Operation' -count=1; commit.

## Task 3: Conformance, required suite, review and live acceptance

- [ ] Run targeted races over new ordering/recovery tests. Run ordinary complete relevant packages once after targeted verification.
- [ ] Extend the real helper conformance fixture with owned, synchronized interrupt gates at park publication/cleanup/finalization boundaries, comparing fake and real retained outcomes. Each kill is only an exact owned child/group verified from the fixture handle and identity; never select a PID by text search. Preserve diagnostics before fixture cleanup. Private namespace only; coordinate local native fixture scheduling with the hub before running.
- [ ] Push exact head and run sdlc local-check run --name suite. Retain every failure; known flakes follow tl-worker's issue-backed one-rerun policy. No gate bypass or operator label.
- [ ] Ask hub to schedule a staged live check on an idle disposable/approved slot: normal relaunch, interrupted scrollback/park, named recover-park, same-conversation resume, duplicate recovery and subsequent delivery. No touch to :0 or live peer without hub coordination. No live acceptance claimed from fake results.
- [ ] Record evidence and any review lessons; run sdlc close --issue 651 --verified '<exact proof>' with project ownership respected. Binary owns fresh-context review. On SHIP run sdlc pr then sdlc merge --yes; report actual merge SHA/PR to hub and stop. No whole-issue landing exception presumed.

## Approval checkpoint

Plan posted before production code. Await TL approval of the public command and the dead-owner interrupted-cleanup retirement policy. In particular, incomplete scrollback must remain explicit; recovery guarantees retained conversation authority, not recovery of bytes that were never preserved. Original pair:6 receipt/error can refine evidence without blocking deterministic class coverage.
