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
| ParkRecoveryResult | cmd/internal/couchcore/park_recovery.go | new |
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

## Test strategies by function

Each row names the callable production boundary to verify. Concrete values and permutations belong in executable tests, not this plan.

| Function | Strategy and mechanical guard |
|---|---|
| DecideParkRecovery | Generate the cross-product of transaction/evidence states; assert effects require explicit publication/identity/death authority and unknown evidence never authorizes retirement. Pure unit tests, without IO fakes. |
| ClassifyRelaunchParkOutcome | Vary original/current record and active-work observations independently; assert outcomes describe current durable state rather than the error channel. Pure unit tests. |
| ThreadStore.RetireInterruptedPark | Inject revision/identity mismatch and journal interruption around persisted updates; assert one atomic after-image, exact owner retirement, preserved unrelated fields and reopen convergence. Actual private ThreadStore, no process mocks. |
| PairLifecycleController.rollbackUnpublishedPark | Cancel around the publication boundary under an owned worker reservation; assert rollback only when the initiating worker holds positive no-publication authority and its bounded store update settles before reservation release. |
| PairLifecycleController.recoverInterruptedPark | Drive adversarial asynchronous event sequences through stateful lifecycle/session/process fakes and journal hooks; assert single-owner effects, completion authority and safe repeatability, with futures settled before fixture cleanup. |
| Couch.RecoverParkExpected | Compare retained lifecycle and native-resume authority before/after recovery through real private storage and existing fakes; assert no automatic launch and no fabricated VerifiedPark. |
| ParkRecoveryNextAction | Vary established, absent and unavailable conversation/session observations; assert advice uses the existing continue/resume/Fresh eligibility and unavailable evidence produces inspection/refusal rather than guessed success. |
| ParkRecoveryAdvice | Exercise arbitrary target forms and operation outcomes; parse each advertised CLI command through production ParseCLI, and resolve local action labels through the menu authority. |
| ParseCLI / slotOperations.handle | Generate malformed argv/confirmation and authenticated-caller/receipt sequences; assert declaration-derived reachability, current owning-hub checks and idempotent receipt semantics before effects. |
| PrepareSlotOperation / ActorActions / menuRowActions | Generate row/park/lifecycle interleavings with changed targets; assert offered actions and executable admission share the same exact address and authority. |
| RunConformanceScenario / realParkConformanceDriver | Gate owned private helper effects and interrupt/reopen them; compare semantic traces and independently checked conversation/ownership invariants with the stateful fake. |
| lifecycleCandidates / storagegc.Decide | Use an interrupted historical owner bundle; assert ownership enumeration and unchanged visible protection/archived retention policy govern its lifetime. |

## Task 1: Permanent interruption proofs and recovery core

**Files:** cmd/internal/couchcore/park_recovery.go, park_recovery_test.go, relaunch_test.go, park.go, threadstore.go, storejournal_test.go; existing lifecycle fakes/helpers.

- [x] Promote the retained diagnostic into independent regression assertions following the function strategies above; run and retain the expected red output before production edits.
- [x] Implement the pure decision and thin recovery/atomic store operation using existing transitions, exact ownership screening and worker reservation. Add a context-aware checked update only for the bounded recovery/rollback store path, reusing existing checked journal and coordinator primitives; ordinary writers retain their existing semantics.
- [x] Make Relaunch classify current durable state after failure and wire the three operator outcomes; rerun the function strategies and inspect independently stated invariants.
- [x] Run go test ./cmd/internal/couchcore -run 'Park|Relaunch|LifecycleDebris|Resume' -count=1; commit code/proof.

## Task 2: Public recovery and actionable guidance

**Files:** cmd/internal/couchcore/ops.go, operationdispatch.go, slot_operation.go, actor-action projection/advice, resume_route.go, recovery_prepared.go; cmd/internal/couchcmd/cli.go, slot_operations.go, help/skill sources; cmd/internal/couchmessage protocol declaration/validation; cmd/internal/couchtty/menu_actions.go and execution; corresponding colocated tests.

- [x] Write and run the named CLI/socket/menu/advice strategy regressions red, then wire the single declared operation to existing authenticated admission/queue and recovery core.
- [x] Centralize guidance for the open-park refusal/partial-result class. Verify production transport and local menu conformance, without treating a recovery result as StartedChild.
- [x] Update atlas/couch.md, user-facing help/skill and README where behavior changes; keep atlas/index links current if adding a page.
- [x] Run go test ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchmessage ./cmd/internal/couchtty -run 'Park|Relaunch|SlotOperation|Resume|Operation' -count=1; commit.

## Task 3: Conformance, required suite, review and live acceptance

- [x] Run targeted races over new ordering/recovery tests. Run ordinary complete relevant packages once after targeted verification.
- [x] Extend the real helper conformance fixture with owned, synchronized interrupt gates at park publication/cleanup/finalization boundaries, comparing fake and real retained outcomes. Each kill is only an exact owned child/group verified from the fixture handle and identity; never select a PID by text search. Preserve diagnostics before fixture cleanup. Private namespace only; coordinate local native fixture scheduling with the hub before running.
- [x] Push exact head and run sdlc local-check run --name suite. Retain every failure; known flakes follow tl-worker's issue-backed one-rerun policy. No gate bypass or operator label.
- [x] Ask hub to schedule a staged live check on an idle disposable/approved slot: normal relaunch, interrupted scrollback/park, named recover-park, same-conversation resume, duplicate recovery and subsequent delivery. No touch to :0 or live peer without hub coordination. No live acceptance claimed from fake results.
- [ ] Record evidence and any review lessons; run sdlc close --issue 651 --verified '<exact proof>' with project ownership respected. Binary owns fresh-context review. On SHIP run sdlc pr then sdlc merge --yes; report actual merge SHA/PR to hub and stop. No whole-issue landing exception presumed.

## Approval checkpoint

Plan posted before production code. Await TL approval of the public command and the dead-owner interrupted-cleanup retirement policy. In particular, incomplete scrollback must remain explicit; recovery guarantees retained conversation authority, not recovery of bytes that were never preserved. Original pair:6 receipt/error can refine evidence without blocking deterministic class coverage.

## Revisions

### 2026-10-17 14:45 PDT — TL approval and terminal operator outcomes

Verified hub receipt4ba546a0 approves both the public confirmed recover-park/menu and the dead-owner interrupted-cleanup retirement policy. After retirement, project into exactly one existing operator outcome: continue live, resume from disk, or start fresh. Success and refusal text name the corresponding next executable command (or exact local menu action); uncertain facts cannot be turned into a guessed outcome. Reuse existing session/native-binding recovery classification for this projection; it is advice, never automatic resume or Fresh. Add tests of established/missing/unknown binding and surviving terminal so the result cannot point at an operation that its own retained record forbids. The original failed pair:6 receipt was not retained by the hub; fixture evidence is the basis, with no claim to reconstruct the original ordering. Approval checkpoint above is now satisfied; change-code owns admission.

### 2026-10-17 — PQ-1 function-level test strategy

Replace prose test inventories in Tasks1–2 with one strategy per named risky function, keeping the existing transition table and acceptance invariants. Concrete cases remain in executable regressions. Clarify that bounded rollback/retirement reuses the existing checked coordinator/journal primitives through a context-aware update, since ordinary updateExistingThread uses blocking Background lock semantics. This is implementation detail within the approved bounded-cleanup policy, not a new recovery authority.

### 2026-10-17 — implementation locations and deferred native/live acceptance

Implementation places bounded checked store IO in park_recovery_store.go and pure recovery facts/results alongside the thin controller in park_recovery.go; shared advice is park_recovery_advice.go. Existing persisted park phases/schema remain unchanged. Permanent fake/real-store interruption matrix and new opt-in native fixture live in park_interruption_matrix_test.go and park_recovery_live_test.go. Generic ordinary store writers retain blocking semantics; only recovery updates/readback select the checked context-aware route.

Hub3e82aae1 andc3d1f43e explicitly defer native conformance/live acceptance while permitting full suite and close review on fake/journal proof. The single scheduled native attempt failed before reaching recovery gates; no passing claim or automatic rerun. Native diagnostics and synchronized private scrollback gate are implemented, with cause classification and a separately scheduled diagnostic run still owed. Do not treat fake/native conversation evidence as a real agent resume.

## Deferred operator live acceptance (hub-coordinated)

- [ ] At the next operator Couch restart, hub identifies the running binary revision and obtains approval for one idle disposable slot; no agent stages :0 or restarts Couch itself.
- [ ] Hub records slot/current tag, native conversation ID and a submitted marker. Confirm no draft/turn/permission prompt, then request normal confirmed relaunch; verify the same native conversation and marker remain accessible, old helper exits and subsequent delivery submits once.
- [ ] On that approved disposable fixture, coordinate an observed park/scrollback interruption with the operator. Record the exact operation receipt and retained thread/attempt before any recovery. Do not signal a process selected by text or touch an unapproved slot. If the intended boundary was not reached, report the observation rather than calling it an interrupted-park pass.
- [ ] Run the refusal's exact recover-park command from the hub with confirmation. Verify complete park or interrupted-cleanup-retired outcome, retained native binding/history and an executable next command; incomplete scrollback is visible, and no child/Fresh starts automatically.
- [ ] Execute the offered continue/resume-from-disk action explicitly; verify the original conversation marker. For a deliberately known-missing conversation, verify Start fresh is offered as an explicit separate confirmation, never automatic. Unknown evidence must refuse instead of choosing Fresh.
- [ ] Repeat the same recovery request; verify no extra quit/start and no replacement transaction is consumed. Confirm a later delivery and normal relaunch still work. Save receipts/observations in the issue Log; hub updates portfolio acceptance. Native and operator acceptance remain unchecked until observed.

### 2026-10-17 15:16 PDT — diagnosed fixture handoff divergence

The one hub-approved finalization diagnostic proves a test-only helper exits before consuming quit intent on completed attach code1/error=nil. Shipped pair-launch-helper only gates/execs Pair; shipped RunLaunch already runs cleanup after this completed handoff. Match the fixture to production, retain execution-error refusal, and add deterministic exit0/1/7/error regression through a test-only attach seam. No production scope/authority change, no claim about original pair:6 ordering. Preserve failed native evidence; request a new named native lane after this fix, and refresh the required suite for changed test code.

### 2026-10-17 15:18 PDT — explicit live-owner refusal policy

Hub5175c159 permits deliberate immediate refusal or a surfaced bounded wait. Choose the original immediate-refusal policy when the exact terminal is positively absent and the exact owner is live or unknown. The observed10s was an existing blocking lifecycle reconciliation lock held during helper cleanup, not an intentional bound and not a promise to wait N seconds. Screen that refusal before completion reconciliation so this path cannot silently wait for live ownership to turn dead. Terminal-present recovery retains existing bounded drive/wait behavior; exact-dead recovery still reconciles matching completion before any retirement. Error advice names the same recovery command; no automatic retry. A deterministic lifecycle callback models owner death during completion inspection and must never be reached for live/unknown absent-terminal owners. Keep the native negative assertion, independently verify helper still live after refusal, then interrupt only the exact owned helper and recover. Native rerun remains hub-scheduled.

### 2026-10-17 15:19 PDT — native fixture proof accepted, operator acceptance still deferred

Finalization, publication and interrupted scrollback fixture cases have each passed in their individually hub-authorized runs after diagnosed fixes. All original failures/log hashes remain in the issue. The scrollback case proves immediate live-owner refusal and independent positive liveness before the exact owned kill; recovery then retires without VerifiedPark and is repeatable. Native terminal/storage fixture conformance is complete; genuine same-agent conversation resume remains the separate deferred operator checklist above. Completed implementation/conformance checkboxes are ticked; suite and close await current code evidence.

### 2026-10-17 — integration declaration completeness

Required-suite guards found missing verb cost, exhaustive source classifications and an exported wrapper with only test consumers. Complete the existing declarations, classify metadata/observer routing as non-artifact source, and keep the unbound convenience wrapper only in tests; production exposes RecoverParkExpected through the declared operation. No new authority or public CLI change. The named function strategy above now names the actual production boundary, and the concepts row drops a planned outcome type that implementation represents in the result. Cancel only the superseded queued suite, preserve the running failure evidence, and verify the corrected pushed head with a new suite.

### 2026-10-17 — complete shared menu-domain validation

Extending open-park presence through the shared row domain requires existing primary-Enter and generic-Recover expectations to describe the separate confirmed Recover park action. The action-name checker now recognizes complete longer labels before their constituent action names; a Recover park label must not be misread as separate recover and park commands. Counterfactual primary add-slot advice retains its ordinary-primary domain. The expanded status assertion exposed live rows hiding the new recovery advice; threadStateText now displays unfinished-transaction advice before normal continuity classification, while rootStateText continues appending retained continuation/missed status. This implements the approved shared guidance sweep, without a new action or recovery authority. Add a red/green regression that both recovery advice and running-continuation status remain visible.

### 2026-10-17 — full-suite proof and publication-only recovery gap

Full suite passes on80063e6d8f7f29e4cc83ad0eb8d7d757b551c393; only its post-success GitHub status publication fails. Keep both facts distinct. Hub9f2f29b6 filed ariadne#431 and explicitly approves close on the retained complete suite proof, with normal merge responsible for running/recording its authoritative suite. No hand-posted status or gate bypass. The live-coordination task is complete because hub scheduled the operator check for the next restart; the separate deferred operator acceptance checklist remains unchecked until observed.


### 2026-10-17 17:27 PDT — close round 1: bounded IO and complete advice class

Close review on59c8d4f0 returned REWORK, BR-1 Critical (bounded-lifecycle-recovery) and BR-2 Important (actionable-park-refusals). Hub3bae113e approves carrying the worker context through every blocking step, explicitly forbids detached IO goroutines, and requires any unavoidable legacy operation to settle synchronously with truthful waiting advice. No legacy transaction rewrite or goroutine escape is needed: reuse checked ThreadStore transactions and the existing lifecycle store algorithm with a context-scoped runtime.

BR-1 IO enumeration: initial recovery read/routing; every completion reconciliation lock/read/fsync; drive-existing reread and attempt append; request publication lock/write/sync/rename; phase/failure journal mutation; completion finalization and conflict reread; successful-attempt evidence cleanup. LifecycleIO now takes the operation context for all three methods. Store.WithContext requires cancellable lock admission, checks synchronous protocol IO, and OSRuntime.LockContext retains the stable flock inode while waiting nonblocking with cancellation. Metadata reads use existing regular-file bounds; ordinary in-progress filesystem syscalls and temporary-file cleanup finish synchronously, never in detached goroutines. Advance/append/finalize gain context-aware internal variants over the same mutations; legacy callers keep their original public methods. The completion wait's deadline reaches its observation and finalization too. Cancellation never converts an indeterminate rename into absent publication and never authorizes request/completion deletion.

New integration concept inventory (supersedes the narrower original rows where noted):

| Name | Kind | Lives in | Status | Wraps |
|---|---|---|---|---|
| LifecycleIO / PairLifecycleStoreIO | INTEGRATION | cmd/internal/couchcore/park.go | modified | context-bearing protocol publication/observation/cleanup |
| Store.WithContext / contextStoreRuntime / contextStoreFile | INTEGRATION | cmd/internal/pairlifecycle/store.go | new | existing synchronous runtime, cancellable lock admission and checked publication IO |
| OSRuntime.LockContext / ReadFileContext | INTEGRATION | cmd/internal/pairlifecycle/store_unix.go | new | stable advisory flock and existing bounded regular metadata reader |
| ThreadStore contextual advance/append/finalize | INTEGRATION | cmd/internal/couchcore/threadstore.go | new | existing checked journal/CAS, same domain mutations as legacy entrypoints |
| ThreadStore interrupted-park retirement | INTEGRATION | cmd/internal/couchcore/park_recovery_store.go | new | correct location of the original retirement row; checked journal/CAS |
| Couch.withParkRecoveryAdvice | INTEGRATION | cmd/internal/couchcore/park_recovery_advice.go | new | bounded read-only exact-target lookup, preserving wrapped typed errors |

BR-1 test strategy: real OS lifecycle flock held after public admission, both terminal-present and dead/absent owners; cancellation must settle the future and release its address before the foreign lock is released. Gate a later threadstore lock during completion observation and test both the drive reread and successful finalization; restore legacy calls to verify both regressions fail. Test cancellation before publication and after rename against actual private protocol files, preserving NotCommitted versus Indeterminate authority. Confirm canceled observation is an error, published evidence survives canceled cleanup, and unpublished staging residue is removed. Existing journal interruption proof installs an armed closure before transaction routing so hooks survive the contextual backend snapshot; it still interrupts precisely after the finalization journal becomes durable.

BR-2 caller enumeration: cold resume before pending/running continuation routing; direct RecoverThread/DecideRecovery refusal and helper reconciliation; Continue's existing/source park failures; new RequestContinuation source-live predicate; agent-switch prepare/source park and continuation guard; Detach; lifecycle-debris owner/abandon failures used by warm/fresh/archive/switch; public Park and Leave new/existing park failures; prepared recovery's source park. Preserve each original authority decision, append the same bounded exact-target advice only when retained or returned state proves an open park, and keep errors.Is/errors.As provenance. Positive warm attach remains before the cold-resume park guard. Add production-entrypoint tests, not formatter-only assertions.

Review verification remains distinct: focused five-package checks passed in review; the review's complete package run encountered the existing pair#432 registration-boundary failure and a separately passing isolated coalescing test whose 50ms admission assumption was unsafe. Replace that assumption with an explicit worker same-nonce join observation, preserving the production coalescing result assertions. This is a test synchronization correction, not a retry of a failed required suite. New full-suite evidence is required after these code fixes;80063e6d's full pass remains historical evidence only. Hub856c5ced grants one short rerun of the three existing native fixtures after targeted proof; actual operator acceptance remains deferred.


### 2026-10-17 — post-review native fixture hook confirmation

Hub856c5ced's three-case rerun passed publication and interrupted scrollback but exposed the live finalization fixture's same late-hook snapshot issue; retained attempt is a failure, not a flake. Install the finalization journal hook before routing and arm it only at finalization, as proved in the deterministic matrix. Hub567ebdc6's single permitted finalization rerun then passed1.03s with the intended interruption and reopen invariants. No production change or additional authority. Updated source/IO corrections require their own recorded full suite before closing again.


### 2026-10-17 — contextual store API inventory correction

The pre-suite exhaustive dead-symbol guard found that all production consumers had moved to contextual advance/append/finalize, leaving the proposed legacy wrappers used only by fixtures. Remove those wrappers instead of allowlisting an unused API: the existing exported AdvancePark/AppendParkAttempt/FinalizePark methods now take an explicit context, and all internal/test construction callers supply it. Their unchanged mutation bodies still use checked journal/CAS. This supersedes the earlier revision's proposed private-variant/legacy-wrapper naming; no wire/persisted API changes. Update integration inventory status to modified ThreadStore park methods. Final guard/recovery/domain target passes artifactpath1.704s/command0.339s/core8.744s/tty0.287s/gccmd0.141s. Native proofs exercise the identical contextual bodies before this mechanical API rename; no native behavior changed.


### 2026-10-18 — complete suite inventory corrections

The complete unchanged ecf055bee suite failed three branch omissions, retained in issue Log. Extend the earlier fixture-hook correction to the one remaining relaunch journal fault (AST enumeration covers all cmd tests); the remaining late AfterGetThread hook is valid because read observers execute on the original store, proved by the existing unreadable-outcome test. Derive exhaustive lifecycle refusal coverage from the common updateExistingThreadOnBackend sink instead of the legacy wrapper, and explicitly exercise the newly included RetireInterruptedPark guard. Correct Remove's Godoc opener. These are fixture/inventory/comment corrections, with no native operation or policy change; the existing native proof remains applicable. No scope or authority delta.
