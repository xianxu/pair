---
collaboration-mode: discussion
---

# Cold-start recovery implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development for bounded work or superpowers-executing-plans for session-warm work. Steps use checkbox syntax.

**Goal:** Make confirmed failed recovery launches immediately retryable and retain their diagnostics without releasing uncertain attempts.

**Architecture:** Keep the existing per-thread operation hold and nonce/revision CAS as the launch owner. Add a terminal failure disposition to the pure start transition, independent of optional diagnostic publication.

**Tech Stack:** Go, Couch thread records, existing Runner/Proc/Artifacts stateful fakes, native transcript runtime, bounded file IO, existing recovery authority profile.

## Status and scope

Approved by TL 2026-10-19 with scope split: this issue lands only the former M1. Native discovery and native-only context move to a separate issue with the original M2/M3 sections as its Spec. One atomic boundary: close #714 directly; no milestone-close.

Covered: acknowledged recovery launch exit, failure disposition, durable diagnostics, immediate retry, bounded observation/storage and exact identity fences. Out of scope: native candidate discovery/context v2, capture identity model, scan limits, fleet orchestration, missing Pair logs, #699 and speculative supervision-log repair.

## Evidence and alternatives

- `launch_existing.go::unconfirmed` publishes failure without a disposition even when its owned helper has exited. Keeping every outcome occupied is safe but strands a definite failure behind the five-minute stale-start policy. Calling ordinary post-ack cleanup unconditionally would instead destroy a possibly successful recovery session. Use an evidence-based terminal disposition.
- `operationattempt_record.go::lifecycleFailureForAttempt` replaces a captured complete address when the outer call supplies only repo-scope. `ObserveCall` also accepts partial addresses. The strict failure codec correctly rejects the resulting empty tag. Fix evidence composition, not codec strictness.
## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| Recovery launch disposition | `cmd/internal/couchcore/recovery_launch_disposition.go` | new |
| StartEvent / AdvanceStartTransaction | `cmd/internal/couchcore/starttransaction.go` | modified |
| Lifecycle failure target composition | `cmd/internal/couchcore/operationattempt_record.go` | modified |

The disposition consumes typed helper/session/registration observations and returns promote, release-failed, or retain-unconfirmed. It performs no probing or storage. StartEvent remains the only owner of creating-incarnation transitions. A release event retains the exact start's selection/context, park and thread, removes only that creating incarnation, and invalidates all later events for that nonce.

Failure target composition uses complete addresses atomically. A partial outer scope never replaces a complete captured address or combines with a tag from another address. A different complete outer target remains the operation target; the captured launch nonce still correlates its child. Scope-only failures use the unresolved reference/path evidence and an empty Address, rather than an invalid half-address. Diagnostic target evidence never grants lifecycle authority.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Recovery failure finalizer | `cmd/internal/couchcore/launch_existing.go` | modified | helper handle, exact terminal observation, bounded ThreadStore CAS |
| LifecycleFailureRecorder | `cmd/internal/couchcore/operationattempt_record.go` | modified | existing failure store and one terminal publication |
Reuse Runner/Proc/Artifacts fakes for process identity, exit, session presence and revision races; use the existing file-backed record/failure stores for restart tests. Live conformance uses a disposable slot only when staged and approved by the hub; no tests touch real native home or current slots.

## Actions and visible state

Actions: inspect exact attempt; retry resume; attach surviving terminal.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Active owned start; helper live; no registration yet | Starting | Inspect exact attempt | None; no duplicate launch |
| Final exact registration established despite helper error/timeout | Running or detached | Attach surviving terminal | No conversation loss |
| Helper exit confirmed, exact proposed terminal absent, no registration | Launch failed; retry available | Retry resume | Failed helper output only if diagnostic storage itself fails |
| Helper dead but terminal present and registration not established | Launch unconfirmed | Inspect exact attempt | No automatic cleanup or new launch |
| Helper/session probe unknown, cancelled observation, or deadline | Launch unconfirmed | Inspect exact attempt | No automatic cleanup or new launch |
| Evidence proves failure but CAS loses revision/nonce or persistence times out | Launch unconfirmed | Inspect exact attempt | No authority discarded |
| Optional diagnostics cannot be saved | Same lifecycle state plus save-failed notice | Inspect exact attempt | Diagnostic bytes may be unavailable; failure/result unchanged |
## Ordering and invariants

ARCH-ORDER: the existing per-thread operation hold spans final observation and disposition; the exact start nonce/revision fences metadata changes. This is the resource preventing a concurrent launch, rather than absence inferred from a scan. A helper must have completed its owned exit/Wait before confirmed release. Its exact proposed terminal is then observed under a fresh bounded context. Timeout or failed observation never means absent. No automatic signal, terminal deletion or second launch is introduced.

State/event table for an acknowledged recovery claim:

| Event/evidence | Transition | Effects |
|---|---|---|
| Exact registration established | Existing StartRegistered | Adopt existing attempt once |
| Owned helper completed + exact terminal absent + no established registration | New terminal failure event | Retain selection, clear exact creating incarnation via CAS, capture failure |
| Any live/unknown helper or present/unknown terminal | Retain same claim | Capture unconfirmed outcome; no cleanup |
| Cancellation before ack | Existing pre-ack cleanup | Reap blocked helper and rollback exact claim |
| Cancellation/timeout after ack | Final bounded observation, then table above | Preserve partial progress |
| Concurrent revision/nonce change or late registration | Refuse stale release | Preserve newer state; no retry effects |
| Duplicate terminal disposition | Reject old nonce or return already completed result | No second launch/publication |
| Supervisor crash during finalization | Existing occupied claim survives | Existing stale-start recovery remains conservative |

## Operating envelope and persisted formats

- ARCH-CONSTRAINTS: failure finalization reuses the existing five-second recovery observation budget and a separate bounded persistence budget; neither diagnostics nor lock acquisition may extend it indefinitely. No cost on successful registration beyond existing finalization. Count observer/store calls for failure sequences; contend real store locks to prove bounded return.
- Growth: O(1) work per failed launch: one final registration look, one exact terminal observation and one disposition CAS; no native enumeration or transcript scan. Count calls across 8/16 failures; at most linear growth. Diagnostic publication remains one bounded write per failed outer operation and zero on success.
- ARCH-SECURE: persisted witnesses, records, native JSONL and paths are untrusted. Keep strict typed decoding, regular-file/root/cwd checks, full interactive provenance and exact fingerprints. Failure text uses existing sanitization and output bounds. No credential or transcript content enters the issue Log.
- ARCH-EVOLVE: no persisted format or enum change. A transient StartEvent stores the existing no-incarnation/retained-choice representation. Preserve unknown fields through the existing ThreadStore rewrite and verify codec round trips.
- ARCH-FUNERAL: no new durable file/process family. Diagnostics use existing bounded failure retention; the failed start's selection uses the current retained-context lifecycle and is superseded by the next start.
- ARCH-DRY / ARCH-PURE: use current hold, transition, store and recorder owners; one pure disposition and one complete-address composition policy.
- ARCH-MOCK: existing stateful Runner/Proc/Artifacts fakes plus portable file-backed stores cover the production seam; disposable live qualification confirms early exit/retry on the staged binary.
- ARCH-PURPOSE: #714's approved contract is now failed-launch retry/diagnostics only. Native recovery remains a separately filed dependency successor, not an undocumented deferral.

## Task — Definite launch failure and durable diagnostics

Files: `couchcore/{recovery_launch_disposition.go,starttransaction.go,launch_existing.go,operationattempt_record.go,lifecyclefailure_record.go}` and colocated tests; `couchcmd/lifecyclefailure_service_test.go`; `atlas/couch.md`.

- [ ] Add failing production-dispatch regression with repo-scope but no tag: recovery helper exits before registering, exact terminal absent, complete captured launch identity and output reach failure store exactly once; a second resume starts immediately without advancing the clock.
- [ ] Add a pure disposition table and pure start-event sequences: exact selection/park retained, input immutable, stale revision/nonce rejected, late registration cannot be cleared. Preserve current helper-exited uncertainty tests when terminal observation fails.
- [ ] Run `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd -run 'RecoveryLaunch|LifecycleFailure|OperationAttempt|StartTransaction' -count=1`; confirm the added regressions fail for the intended wedge/identity reason.
- [ ] Implement the terminal disposition. Observe using exact proposed session binding under the current operation hold, use bounded context-aware metadata reads/writes, then apply one nonce/revision event. A completed helper plus absent terminal releases only this claim; all uncertain cases retain it. Keep ordinary warm/spawn cleanup unchanged.
- [ ] Centralize complete-address diagnostic composition at both ObserveCall and terminal enrichment. Sweep every partial-address assignment in that family. Preserve exact launch address when outer evidence is partial; unresolved refusals remain persistable by path/reference. Do not weaken normalizeLaunchFailure.
- [ ] Test live owner with failed helper, unknown/live terminal, failed probe, canceled wait, late registration, CAS conflict, held store lock, diagnostic store failure, duplicate Finish and successful operation (zero diagnostic IO). Include integration assertions through CLI dispatcher and restarted file-backed store.
- [ ] Add a Pair-local lint rule for the mechanical partial-address override family using the existing local catalog; test the safe centralized composition and bad whole-address nonzero guards. Restrict rule to actual diagnostic composition owners to avoid generic ThreadAddress false positives.
- [ ] Run focused tests plus race checks, update atlas/lessons and Log, commit with Agent/Co-Authored-By trailers, push and run the recorded suite, then close #714 directly through `sdlc close --issue 714 --verified ...`. On SHIP enter `sdlc merge --yes`, report the landing and stop before starting the successor.


## Revisions

- 2026-10-19, TL approval: split original M2/M3 into a new issue. This plan now ends at the former M1 with one issue-close boundary. All failed-disposition uncertainty is displayed as Launch unconfirmed with Inspect; no separate visible disposition state. Original design remains in the prior plan commit and transferred successor Spec.
