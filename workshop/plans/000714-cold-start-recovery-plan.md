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
| ReconcileStart | `cmd/internal/couchcore/starttransaction.go` | existing policy reused |
| StartEvent / AdvanceStartTransaction | `cmd/internal/couchcore/starttransaction.go` | modified |
| Lifecycle failure target composition | `cmd/internal/couchcore/operationattempt_record.go` | modified |

Existing ReconcileStart consumes typed helper/session/registration observations and returns promote, rollback, or keep-occupied. It performs no probing or storage; reuse it rather than introduce a redundant pure decider. StartEvent remains the only owner of creating-incarnation transitions. A release event retains the exact start's selection/context, park and thread, removes only that creating incarnation, and invalidates all later events for that nonce.

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
- ARCH-EVOLVE: no persisted format or enum change. A transient StartEvent stores the existing no-incarnation/retained-choice representation. Retain the current strict decode/refusal contract: threadrecord.DecodePersisted and TestRecoverySchemaRejectsUnknownNestedFields reject unknown nested members. This issue changes no schema and does not introduce a decoder/preservation migration; that compatibility work is outside this fix. Validate unchanged supported-record round trips and unchanged unsupported-record refusal.
- ARCH-FUNERAL: no new durable file/process family. Diagnostics use existing bounded failure retention; the failed start's selection uses the current retained-context lifecycle and is superseded by the next start.
- ARCH-DRY / ARCH-PURE: use current hold, transition, store and recorder owners; one pure disposition and one complete-address composition policy.
- ARCH-MOCK: existing stateful Runner/Proc/Artifacts fakes plus portable file-backed stores cover the production seam; disposable live qualification confirms early exit/retry on the staged binary.
- ARCH-PURPOSE: #714's approved contract is now failed-launch retry/diagnostics only. Native recovery remains a separately filed dependency successor, not an undocumented deferral.

## Function-level test strategy

| Risky function / seam | Adversarial class and mechanical guard |
|---|---|
| `ReconcileStart` (existing pure policy) | Exhaust helper/session/registration knowledge values; independent literal policy asserts only proved failure releases and established registration wins. |
| `AdvanceStartTransaction` / new terminal failure event | Generate duplicate, stale and reordered nonce/event sequences; assert immutable input, retained exact selection and no removal of a registered/newer incarnation. |
| `completeFailureAddress` (new pure composition owner) | Table/fuzz partial and conflicting addresses; assert an output is one complete input or zero, never a fieldwise hybrid. |
| `lifecycleFailureTargetForCall`, `OperationAttempt.ObserveCall`, `lifecycleFailureForAttempt` | Dispatch arbitrary partial target evidence through real recorder normalization; assert complete nested evidence survives and unresolved targets remain persistable. |
| `Couch.finalizeFailedRecoveryLaunch` (new IO finalizer) | Gate helper completion, exact-session replies and revision changes with stateful fakes; use contended real metadata locks and an injected deadline to prove bounded return without stale release or automatic cleanup. |
| `DispatchOperation` -> `ResumeTargetWith` -> `launchTrackedThread` | File-backed restart regression at the production scope-only resume boundary; assert one readable diagnostic and immediate same-conversation retry with no clock advance. |
| `OperationAttempt.FinishContext` and nested capture | Inject failure-store refusal and duplicate completion; assert lifecycle result unchanged, one bounded publication, and zero diagnostic IO on successful launch. |

## Task — Definite launch failure and durable diagnostics

Files: `couchcore/{recovery_launch_disposition.go,starttransaction.go,launch_existing.go,operationattempt_record.go,lifecyclefailure_record.go}` and colocated tests; `couchcmd/lifecyclefailure_service_test.go`; `internal/repolint` local catalog and rule tests; `atlas/couch.md`.

- [x] Write the production-dispatch immediate-retry and retained-diagnostics regressions using the function strategies above; implement no production changes yet.
- [x] Run `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd -run 'RecoveryLaunch|LifecycleFailure|OperationAttempt|StartTransaction' -count=1`; observe the intended failure at the wedge/identity assertions.
- [x] Reuse `ReconcileStart` and implement `Couch.finalizeFailedRecoveryLaunch`: after owned exit completion, observe the exact proposed session under the existing operation hold and apply one bounded nonce/revision transition. Retain all uncertain outcomes; ordinary warm/spawn cleanup stays unchanged.
- [x] Extend `AdvanceStartTransaction` with the terminal failed-start event, retaining its selection/park/thread while clearing only the matching creating incarnation; execute it through bounded context-aware ThreadStore CAS.
- [x] Implement `completeFailureAddress` and use it at call extraction, ObserveCall and terminal failure enrichment. Sweep these composition owners for nonzero-address and fieldwise overwrites; keep normalizeLaunchFailure strict.
- [x] Add a Pair-local lint guard for unsafe direct partial-address overrides in the three diagnostic composition owners using the existing local catalog. Exercise bad variants and the centralized complete-address form through rule fixtures.
- [x] Run the named function strategies, focused package tests and race checks. Record counted O(1) failure observation/storage work and bounded lock-contention results. Update atlas/lessons and Log; commit with Agent/Co-Authored-By trailers.
- [x] Obtain hub qualification decision: no stage or contrived live check; production-path RED/GREEN is sufficient. Publish PR and request recorded suite.
- [ ] Require recorded suite pass before landing; close #714 directly through `sdlc close --issue 714 --verified ...`, allowing boundary review to overlap its queued suite per tl-worker. On SHIP enter `sdlc merge --yes`, report landing, then take #721 from main.

## Revisions

- 2026-10-19, TL approval: split original M2/M3 into a new issue. This plan now ends at the former M1 with one issue-close boundary. All failed-disposition uncertainty is displayed as Launch unconfirmed with Inspect; no separate visible disposition state. Original design remains in the prior plan commit and transferred successor Spec.

- 2026-10-19, plan gate PQ-1/PQ-2: replaced prose test inventories with one named-function adversarial strategy/guard table. Corrected unsupported ThreadStore preservation claim after reading DecodePersisted and TestRecoverySchemaRejectsUnknownNestedFields; no schema migration enters this issue.

- 2026-10-19, implementation refinement: ReconcileStart already encodes the approved dead-helper/absent-session policy, so reuse it instead of adding a duplicate pure decider. The new finalizer is IO and stays in Integration points. A direct-child exit with live owned group retains the claim. Final observation and CAS share one five-second context (a stricter combined budget than separate phases); diagnostic save-failure notice is retained at OperatorError. No policy or scope expansion.

- 2026-10-19, TL qualification decision: do not stage :0 or induce a disposable failure. Existing production-path regression is the qualification for this rare event; TL will record a next-natural-failed-resume smoke observation. No live-state mutation requested.
