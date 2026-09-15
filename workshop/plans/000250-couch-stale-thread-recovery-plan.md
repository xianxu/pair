# Couch stale-thread recovery Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Recover stale Couch rows by preserving an exactly owned surviving session or starting a new conversation from an explicitly selected, durable checkpoint, while making explicit archive reachable.

**Architecture:** Observe helper identity and exact Pair session independently, decide available actions in a pure recovery projection, and retire only proved-dead settled incarnations through the existing store transaction. Reuse warm reattachment and the existing continuation launcher; extend the continuation request with truthful source-absence authority instead of manufacturing a verified park. Use the existing Console operation queue and singleton supervisor lease for lifecycle ordering, routing archive through the same owner path and retaining store revisions and existing start/park transactions as durable crash boundaries.

**Tech Stack:** Go, existing Couch ThreadStore, checkpoint/orientation packages, Couch terminal menu reducer, stateful process/session fixtures, Zellij disposable smoke.

---

## Approval and scope

The operator approved the coordinated #248 → #249 → #250 design and implementation and has since approved a disposable fault fixture for #250 operational acceptance. The original blocked row was manually reconciled earlier; do not damage or restart the real conversation to reproduce this issue. Root owns the issue log and SDLC gates. This plan contains one atomic review boundary at issue close, with plain task checkboxes rather than milestone tags.

The feature includes stale settled live incarnations and already-retired records. Open start/park transactions remain owned by their existing recovery mechanisms and produce an actionable refusal here. A surviving detached session always takes precedence for the ordinary Recover action. Checkpoint recovery is visibly a new conversation; it never silently becomes an empty launch or native resume.

## Core concepts

### Pure entities

| Name | Kind | Lives in | Status |
|------|------|----------|--------|
| RecoveryEvidence / RecoveryDecision | PURE | `cmd/internal/couchcore/recovery.go` | new |
| SourceAbsence / checkpoint.Request | PURE | `cmd/internal/checkpoint/request.go` | modified |
| Recovery menu text/action behavior | PURE | `cmd/internal/couchtty/menu.go`, `menu_render.go` | modified |
| ActionableThreadSummary recovery choices | PURE | `cmd/internal/couchcore/actionableinventory.go` | modified |

- **RecoveryEvidence / RecoveryDecision:** One immutable observation bundle per address, containing record revision, settled helper identity/liveness, session presence and unique detached proof, source generation, checkpoint availability, native binding status, and observation diagnostics. One pure decider returns warm reattach, checkpoint recovery, archive eligibility, or a concrete refusal. Do not infer absence from an empty detached-session result. Relationships are one record to one decision; observations are short-lived and execution gathers them again. Future evidence types extend this bundle rather than adding a second UI classifier.
- **SourceAbsence:** A continuation request may have either the existing source park receipt or explicit source absence, never an invented park. The absence witness records exact source session/generation, observation time, and record revision; it optionally records the proved-dead helper identity when an incarnation existed. It is authorizing evidence only when persisted by the recovery transaction after exact observation, not because a path or menu payload claims it. Existing source-present requests continue to require `Source.Helper`. A legacy import may omit that helper only with the explicit absence witness. Target validation accepts real park OR absence authority; the existing phases and exact attempt correlation remain unchanged. The schema is backward compatible for existing version-1 requests.
- **Checkpoint provenance:** `checkpoint.Checkpoint` remains unchanged and authoritative: absolute original path, embedded bytes, digest, and agent. Import uses `checkpoint.ReadFile` and newest exact launch generation from `OSContinuationSourceReader.Read`; require matching checkpoint/profile/source agent. For a record retired by the manual repair, do not invent a helper identity. Missing or ambiguous ledger/session provenance is an actionable refusal with archive still available. Explicit operator path selection is authority to seed that thread from those bytes; a filename is not proof that the document was produced by that thread.
- **Native resume:** Existing parked native resume remains unchanged and requires verified binding. Source-gone recovery offers checkpoint seeding, not a new native-resume authority. Do not broaden `ResumeOptions` or fabricate a `VerifiedPark` to make the old cold path eligible.

### Integration points

| Name | Kind | Lives in | Status | Wraps |
|------|------|----------|--------|-------|
| Recovery observation/execution | INTEGRATION | `cmd/internal/couchcore/recovery.go` | new | ProcOps, PairSessionIO, DetachedSessionResolver, source reader, ThreadStore |
| Owner lifecycle routing | INTEGRATION | `cmd/internal/couchcore/operationdispatch.go`, `ops.go`; existing Console operation queue | modified | existing supervisor lease and owner operation queue |
| Continuation source preparation/execution | INTEGRATION | `cmd/internal/couchcore/continuation.go`, `continuation_recovery.go`, `continuation_store.go` | modified | exact checkpoint import, durable request, existing launcher |
| Explicit reconciled archive | INTEGRATION | `cmd/internal/couchcore/detach.go`, `threadstore.go` | modified | exact session stop and archive transaction |
| Recovery operations and Console adoption | INTEGRATION | `cmd/internal/couchcore/ops.go`, `operationdispatch.go`, `cmd/internal/couchtty/console.go`, `console_continuation.go` | modified | owner operation dispatch and returned StartedChild |
| Process/session/terminal test environment | INTEGRATION | existing fixtures in `cmd/internal/couchcore/continuation_recovery_test.go`, `warmresume_test.go`; `cmd/internal/couchcmd/continuation_acceptance_test.go` | reused | stateful fake process identities, sessions, receipts, IO |

`observeExactProcess`, `ThreadStore.RetireIncarnation`, `ResumeContextWith(WarmOnly:true)`, `CommitStartClaim`, `launchTrackedThread`, `materializeContinuation`, and exact readiness/orientation receipt handling are unchanged reuse unless extraction is necessary. These are not parallel implementations or new conceptual entities.

## Architecture decisions

- **ARCH-DRY:** Generalize #249's dead-helper retirement in `ensureContinuationAttached` into the shared recovery path; preserve its special exact registered-target start promotion. Both recovery and continuation use it. Keep one launcher and one checkpoint materializer.
- **ARCH-PURE:** Classification and request transitions take values; process/session/filesystem reads and effects stay in injected orchestration.
- **ARCH-PURPOSE:** Deliver warm, source-gone checkpoint, legacy checkpoint import, and archive escape. Unit-only completion or only clearing the stale label does not satisfy the issue.
- **ARCH-MOCK:** Extend existing stateful process/session fixtures; assert agent identity, spawned count, input/output, durable bytes and terminal receipts across multiple calls.
- **ARCH-CONSTRAINTS:** Keep current checkpoint 256 KiB and path 4096-byte bounds, bounded revision retries and cancellable operations. Require one settled incarnation or none. Unknown and ambiguous evidence refuse effects. Recovery does not scan history or every worktree for a plausible checkpoint.
- **ARCH-SECURE:** Exact address, latest launch ordinal, process-start identity and session binding are authority. User-supplied checkpoint paths are data passed through the existing bounded regular-file reader, never shell interpolation. Revalidate selected bytes before publication; persist the accepted snapshot before retiring source state for an imported request.
- **ARCH-ORDER:** Existing singleton supervisor lease excludes independent live owners. Console continuation scans enqueue lifecycle work through the same operation queue as ordinary menu actions. Move archive from direct-store to live-owner execution so it crosses that boundary too. Test ordering at the production queue/lease seam; do not introduce a redundant executor mutex. Keep record revisions and start/park transactions as durable ownership fences and do not hold the ThreadStore flock across external IO.
- **ARCH-FUNERAL:** Successful recovery consumes a start claim through registration and tracks delivery with the existing attempt. Failed launches retain snapshot and retry state. Materialized checkpoint files remain derived and archive removes them through existing cleanup while retaining embedded bytes. Explicit archive preserves the continuation request and all history in the archived record.

## Chunk 1: Implement and prove recovery

### Task 1: Reproduce the complete stale-row dead end

**Files:** Create `cmd/internal/couchcore/recovery_test.go`; extend `cmd/internal/couchtty/menu_action_sweep_test.go` and `cmd/internal/couchcore/archive_test.go`.

- [x] Add a stateful record with one settled live incarnation, mark that exact helper dead, and prove current inventory offers an unusable row whose archive fails. Add surviving detached session and absent-session variants.
- [x] Add an already-retired legacy record with checkpoint in a second worktree and no published continuation request. Preserve source ledger/session binding provenance; intentionally omit any helper identity from the retired record.
- [x] Run `go test ./cmd/internal/couchcore ./cmd/internal/couchtty -run 'Recovery|Archive' -count=1`; verify new recovery assertions fail for the expected missing behavior.
- [x] Keep fixture changes isolated from real data directories; count actual fake launch/session-stop calls through resulting state, not only mocked expectations.

### Task 2: Share lifecycle ordering and exact stale-owner reconciliation

**Files:** Modify `cmd/internal/couchcore/couch.go`, `operationdispatch.go`, `ops.go`, `detach.go`, `continuation_recovery.go`; create `cmd/internal/couchcore/recovery.go`; extend `operationdispatch_test.go`, `ops_declarations_test.go`, `recovery_test.go`.

- [x] Move archive to `ExecuteLiveOwner` and update operation declarations and dispatch tests. Verify menu archive, continuation execution/retry/status, recovery and ordinary lifecycle actions all cross Console's existing operation queue; internal CLI owner operations acquire the existing singleton supervisor lease.
- [x] Inspect startup, leave and park recovery workers for concurrent effects. Preserve their existing durable start/park guards and avoid new recursive locking around `Relaunch → Park → Resume` or `Continue → Resume/Park`.
- [x] Test the actual queue with recovery/archive submitted while another lifecycle effect is paused, and test the namespace lease against another owner. Assert no duplicate launch or interleaving stop effects. Core tests that bypass the queue exercise store CAS conflicts as lower-level transaction behavior, not a new public concurrent-owner contract.
- [x] Handle canceled park wait correctly: `PairLifecycleController.submit` can return while its worker continues. Prove its durable open Park guard blocks later destructive/recovery effects until settlement; do not equate returning from Await with quiescence. Test cancellation in that interval.
- [x] Implement shared observations and pure decisions. `observeExactProcess == Dead` is required before retirement; mismatch of PID start identity counts as dead for the recorded process. Session observation error/active client/ambiguous ownership refuses recovery. Exact absence permits reconciliation without fabricating park state.
- [x] Retire with `RetireIncarnation(address, revision, identity, record.LastActiveAt)`, rereading and reobserving on bounded revision conflict. Refuse creating/unknown state, multiple incarnations, open start or park. Extraction from `ensureContinuationAttached` must retain exact continuation source/target checks and registration promotion.
- [x] Run `go test ./cmd/internal/couchcore -run 'Recovery|Continuation|Warm|Archive|Operation' -count=1`; verify unchanged warm resume never consults native binding and no recovery observation stops a session.

### Task 3: Add truthful source-absence authority and checkpoint import

**Files:** Modify `cmd/internal/checkpoint/request.go`, `request_test.go`; `cmd/internal/couchcore/continuation.go`, `continuation_store.go`, `continuation_recovery.go`; extend `continuation_store_test.go`, `continuation_recovery_test.go`, `recovery_test.go`.

- [x] Add pure request tests for valid legacy absent-source import with no helper, rejection of missing source authority, rejection of both park and absence authority, target registration with real absence witness, obsolete attempt/event rejection, and backward compatibility of existing park requests.
- [x] Keep the phase enum unchanged. Update ThreadRecord continuation identity validation to cover the source-absence representation and deterministic request ID, including legacy omitted-helper imports. Add a named source-absence event and store transaction that accepts the exact observed source generation and revision. If publishing a legacy request, atomically preserve its checkpoint and absence witness before dropping the sole proved-dead incarnation; a crash cannot discard the only source authority. An already-retired record is valid with exact absence/generation proof.
- [x] Keep publication idempotent by address, generation and checkpoint digest. A different imported checkpoint must not overwrite an unresolved existing request: expose retained recovery or explicit archive instead. Request.Source remains bound to the observed latest ledger generation; source identity is not inferred from document text.
- [x] Refactor `executeContinuation` to accept real source park OR persisted source absence. For absence, reobserve exact source generation and session absence before `CommitStartClaim`; after claim, recheck before child effects. Reuse `materializeContinuation`, fresh argv sanitation, `StartFreshExisting`, orientation and target registration code unchanged where possible.
- [x] Extend retry so a registered surviving target is observed/reattached, not duplicated; a new attempt requires exact target absence and dead helper/owner proof. Source absence does not authorize killing any newly appearing source or target session.
- [x] Test crash/retry after publication, retirement, start claim, helper registration, and delivery submission; demonstrate one target and immutable checkpoint bytes after original file moves or changes. Missing/unreadable/oversized/wrong-agent documents fail before spawning.
- [x] Run `go test ./cmd/internal/checkpoint ./cmd/internal/couchcore -run 'Request|Recovery|Continuation' -count=1` and commit the verified core unit with an issue reference and author trailer.

### Task 4: Make explicit archive truthful

**Files:** Modify `cmd/internal/couchcore/detach.go`, `threadstore.go`, `thread.go`, `recovery.go`; extend `archive_test.go`, `recovery_test.go`.

- [x] Have explicit archive reobserve/reconcile a dead settled incarnation before applying the existing occupancy guard. Do not relax `archivableRecord` globally based only on persisted stale state.
- [x] Remove the incomplete-continuation refusal from archive eligibility only, retaining all occupancy/open-park guards and owner routing. A live source or target remains occupied and refuses; an empty settled record may be explicitly archived with its pending/failed request preserved. A queued continuation then observes the absent active record and stops. Reobserve exact source/target sessions and helper evidence before effects; do not mark the request Complete or delete its snapshot. Add revision checking at the final archive transaction if concurrent direct-store publication could otherwise replace the inspected request.
- [x] Keep Quiesce behind explicit archive confirmation and exact ownership checks. Recovery and inspection never call it. With unknown observation, report a refusal rather than guessing absence. Existing unreadable-record archive behavior retains its non-signalling warning.
- [x] Test incomplete request archive preserves body/native references/history, missing checkpoint still allows safe archive, failed observation performs no stop, target appearance races refuse, and ordinary parked native resume still requires a verified binding.
- [x] Run `go test ./cmd/internal/couchcore -run 'Recovery|Archive|Resume|ContinuationGuard' -count=1`.

### Task 5: Expose actionable recovery in Couch

**Files:** Modify `cmd/internal/couchcore/actionableinventory.go`, `threadreason.go`, `ops.go`, `operationdispatch.go`; `cmd/internal/couchtty/menu.go`, `menu_render.go`, `console.go`, `console_continuation.go`; extend `menu_test.go`, `menu_action_sweep_test.go`, `operationdispatch_test.go`; create `cmd/internal/couchtty/menu_recovery_test.go`.

- [x] Add recovery diagnosis and choices to the inventory projection without mutating the record. Replace the claim that Couch necessarily exited with a helper/session-specific diagnosis. Project available choices from the shared decider, not a second UI ownership policy.
- [x] Add Recover for warm session or retained checkpoint. Add Recover from checkpoint using existing `MenuFrameText` and `reduceTextKey`: one absolute-path text field, no file picker or filesystem scan. Field limit is 4096 bytes and the command passes structured `path`, `repo-scope`, and exact `tag/ref` arguments.
- [x] Render the path action as starting a new conversation from the selected checkpoint. Existing retained checkpoint action shows its source path/digest. Keep existing parked native resume unchanged and visibly distinct from checkpoint recovery. Enter on a stale recoverable row chooses safe ordinary Recover, never silently selects native resume or empty fresh start.
- [x] Make failed/running continuation rows expose retained retry and safe explicit archive when the recovery projection permits it. Missing evidence supplies a diagnostic and path-entry/archive choices rather than an action that always fails.
- [x] Declare and dispatch all new actions as owner lifecycle effects. Route returned `ContinuationResult`/StartedChild through existing Console installation and continuation watching, including a source-only warm attachment result. Failed attachment cleanup preserves the surviving agent and accepted snapshot.
- [x] Test generated menu payloads through `DispatchOperation` into a stateful Couch and confirm usable fake terminal input/output, not merely action labels. Cover refresh invalidation, path editing/cancel/error retention, concurrency with continuation polling, and final menu completion notices.
- [x] Run `go test ./cmd/internal/couchtty ./cmd/internal/couchcmd ./cmd/internal/couchcore -run 'Recovery|Continuation|ActionSweep|Operation|Archive' -count=1`.

### Task 6: Verify full behavior and disposable operational acceptance

**Files:** Extend `cmd/internal/couchcmd/continuation_acceptance_test.go` or create `cmd/internal/couchcmd/recovery_acceptance_test.go`; update `atlas/couch.md` (verify actual Couch atlas path first), `atlas/index.md`, operator README/help as appropriate; root updates `workshop/issues/000250-couch-stale-thread-recovery.md`.

- [x] Add disposable process acceptance using an actual helper PID and process-start identity (a fake-only death assertion is insufficient), a random fixture-owned ControlledZellij session, temporary namespace/data directory and Git repo plus sibling worktree. Cover dead helper + surviving exact session and dead helper + absent session + checkpoint written in another worktree, through real operation dispatch/Console adoption. Reuse existing process helper harness rather than another launcher implementation.
- [x] Run `go test ./cmd/internal/checkpoint ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd -count=1`, then `go test -race ./cmd/internal/couchcore ./cmd/internal/couchtty -count=1`, and `git diff --check`. Run additional repo-required checks named by the SDLC gate; broaden tests only for a changed surface or unresolved concern.
- [x] Build the current binaries using the existing Makefile target. Create a distinct disposable namespace, data directory, exact thread/session and checkpoint; record their identities before fault injection. Use only those recorded fixture identities for helper/session termination.
- [x] In fixture A, remove only the helper and leave its agent/session alive. Use Couch Recover and verify same agent identity and actual input/output. In fixture B, remove only the fixture-owned disposable source helper/session, recover from a checkpoint in another worktree via the text path form, and verify the replacement reads the exact digest/NEXT ACTION and accepts input.
- [x] Exercise explicit archive of a disposable stale row with missing checkpoint. Confirm the row disappears while archived metadata/history persist, and no unrelated session/process/store record changes. Record before/after identity evidence and cleanup only owned fixture resources.
- [ ] Record the already-repaired original incident and the operator-approved fixture substitution in the issue. Request operator confirmation of usable fixture Couch access when the acceptance result is concrete; do not restart or fault the real thread.
- [x] Document recovery choices and absence/unknown diagnostics, update atlas links, and compare concept-table rows against the actual diff. Append plan revisions for deviations rather than overwriting the design history.
- [ ] Commit verified work, then let root run `sdlc close --issue 250 --verified '<actual evidence>'`. Its fresh-context review is the issue boundary; fix Critical/Important findings before publication.

## Revisions

Initial executable plan drafted 2026-09-14 from #248/#249 implementation inspection, the #250 incident/manual repair log, and the operator-approved disposable fixture strategy. Source-absence authority, legacy path import and shared owner executor ordering address the gaps that reusing ordinary Continue/Retry alone cannot cover.

### 2026-09-14 — Root review scope correction

Inspection confirmed continuation work already enters Console's ordinary operation queue; reuse queue plus singleton lease, moving archive into owner routing, instead of adding another mutex. Existing verified parked native resume satisfies the native-binding constraint; absent-session native recovery is outside this implementation. Archive preserves incomplete requests as archived history without inventing a terminal continuation phase. Operational fixtures must observe an actual disposable helper identity, not only simulated death.

### 2026-09-14 — Fresh review: target generations and absence phase contract

**Reason:** Task 3's unconditional comparison with the immutable source launch ordinal would reject a legitimate retry after this request's own failed target advanced the ledger. The initial absence model also left import and retained-request phase transitions implicit. These rules supersede the affected Task 3 wording and add the following required work; the original source identity and checkpoint never change to follow the latest ledger.

- [x] **Separate source identity from generation admission.** Before any target was launched, accept only the exact recorded source generation. After this request launches a target, accept the current source generation OR the exact generation durably proven to belong to this request's prior target attempt. Require matching address, agent, session, attempt nonce and launch ordinal. A greater ordinal, same agent/session, dead helper, or missing registration alone is not ownership proof. Foreign or uncorrelated advancement refuses all new launches with an actionable diagnostic.
- [x] **Persist target-generation correlation.** Extend the existing target witness/registration transition to retain its attempt and observed exact launch generation, and preserve that witness when `RetryAbsent` allocates the next attempt; keep only the last authorized generation needed for subsequent admission, not an unbounded scan/history. Bind the evidence before replacing target state. The current `checkpoint.Target` has only process/time and `readiness.ReadyRecord` has no launch ordinal, so `FreshRegistration == true` alone cannot provide this correlation. Use an exact launch-generation-bearing readiness receipt (extend its optional schema/producer and tests if no existing durable witness provides that join), then persist the joined witness. Recover a crash between receipt publication and target persistence using that exact receipt. Missing correlation remains unresolved rather than allowing any newest ledger row.
- [x] **Sweep the admission checks.** Apply the same source-or-owned-target rule to absence recovery execution, retry, and post-claim pre-launch recheck; avoid a second unconditional source-generation comparison later in the call chain. Existing target observation/reattachment still wins over replacement. New launch additionally requires exact session absence and proved-dead target/helper/owner, independently of generation admission.
- [x] **Add deterministic advancing-ledger regression.** Publish/import checkpoint at source ordinal S, launch request attempt A, append its real target launch ordinal T, persist/correlate A↔T, then prove target/helper/session death. Retry must preserve source S and checkpoint digest, allocate attempt B, and produce exactly one replacement. Repeat with an unrelated ordinal U after T and require refusal with zero additional spawns. Cover interruption before target witness persistence using an exact receipt, plus missing/foreign receipt refusal. Test two successive owned target failures to ensure only the newest authorized target generation is carried forward.

**Explicit absence phase contract (no new phase values):**

| Existing phase / operation | Allowed authority transition | Result |
|----------------------------|------------------------------|--------|
| New legacy import | Atomically publish Pending with validated checkpoint and SourceAbsence; no attempt or target; source helper may be omitted only with absence authority | Pending |
| Pending retained request, source now absent | Persist SourceAbsence against exact source generation/revision after dead-helper or already-retired and session-absence proof; no park witness may coexist | Pending |
| Begin | Preserve checkpoint, source and SourceAbsence; allocate exact attempt | Running |
| Running retained request, source died before park/target | Add SourceAbsence only after proving no target/registration and exact source absence; an unresolved target claim must use target reconciliation instead | Running |
| Running with park or absence authority | Registered accepts an exact target and its correlated generation; Submitted requires that target | Running, then Complete |
| Failure | Preserve source authority, checkpoint and any target-generation witness | Failed |
| Failed retained request lacking source absence | Only an explicit recovery transaction may add SourceAbsence after the same source/target checks; preserve attempt and failure until retry transition | Failed |
| RetryObserve | Preserve source authority and exact existing attempt/target; observe or reattach it | Running |
| RetryAbsent | Require target absence/death and source-or-owned-target generation admission; retain source authority and last authorized generation, clear replaced target, allocate a new attempt | Running |
| Complete | No automatic reopen, source-absence rewrite, or checkpoint overwrite | Complete |

- [x] Encode this table in `checkpoint.Request.Validate`, `checkpoint.Advance` and revision-checked recovery/store transitions, with pure tests for every allowed row and representative invalid cross-phase transitions. Pending permits absence authority but still forbids attempts/targets/failure; target validation requires exactly one of genuine park or absence authority. Adding absence never erases an existing target or authorizes a launch by itself. Extend ThreadRecord/request identity validation and round-trip tests for omitted legacy helpers and target generation witnesses while accepting existing valid park-based records.

The newly reported Astro process/session incident is runtime investigation owned by root. This plan revision authorizes no changes to that incident's processes or state.

### 2026-09-14 — Plan-quality round 1: PQ-1, PQ-2 and PQ-3

**Reason:** The gate requires function-level adversarial strategies, explicit observation/retry budgets and an ongoing live comparison for the reused external seams. This section is the authoritative test strategy and operating envelope for Tasks 2–6; their earlier case lists describe scenario scope only and must not become redundant parallel suites.

**PQ-1 — Risky functions and mechanical oracles.** Use the named functions below for the new pure boundaries; if implementation names differ, append the final mapping in the issue log. Each test exercises the function named in its row, not a helper that repeats its implementation.

| Risky function | Adversarial input/event class | Mechanical oracle |
|----------------|-----------------------------|-------------------|
| `DecideRecovery(RecoveryEvidence)` (new pure decider) | Cartesian table over settled/no/multiple incarnations, live/dead/unknown exact identity, absent/uniquely detached/occupied/ambiguous/unobserved session, open transaction, retained checkpoint present/absent | Expected action set from the explicit ownership rules; unknown/occupied/open-transaction rows contain no launch/retire action; warm wins when exact detached session survives; permuting unrelated evidence cannot change the decision for the address. Inputs remain deeply equal before/after. |
| `AdmitRecoveryGeneration` (new pure function) | Source S, owned target T with correct/wrong attempt and receipt, foreign U, missing witnesses, two successive owned failures | Admit iff current generation is S or exactly correlated latest owned T; reject greater-but-unowned ordinals, mismatched scope/agent/session/attempt and incomplete witness. Keep source/checkpoint equality after every call. |
| `checkpoint.Request.Validate` | Mutate one field of valid legacy-absence and existing parked requests: omitted helper, absent/both authorities, target without authority, pending attempt, bad target-generation join | Error/no-error equals the phase/authority table above, never panic; JSON round-trip preserves accepted authority and source/digest; all existing valid parked fixtures remain accepted. |
| `checkpoint.Advance` | Generate bounded legal and illegal event sequences across Pending/Running/Failed/Complete, injecting stale request IDs/attempts and duplicate events | Every accepted result passes Validate, keeps immutable source/checkpoint, changes only table-authorized fields; rejected events leave the original value unchanged. RetryAbsent changes attempt and retains latest authorized target generation; Submitted cannot precede an exact target. |
| Recovery observation/execution (new `Couch.RecoverThread`) | Stateful fake barriers after process observation, before retirement CAS and after claim: reuse PID, change session ownership, mutate revision, cancel context | Compare persisted record and fake process/session/launcher state: no signal during observation, only exact dead settled identity retired, no more than one claimed target, unknown evidence yields zero new processes. Snapshot digest and historical activity remain unchanged. |
| `Couch.executeContinuation` / `RetryContinuation` | Actual fixture ledger S→owned T, target death, retry; substitute foreign U; interrupt between ready receipt and target persistence | Count fake/live target starts and inspect durable request: exactly one replacement on authorized retry, zero on foreign/uncorrelated generation, source S and digest unchanged, correct new attempt with retained generation witness. Exact receipt recovers lost target persistence without a second spawn. |
| `Couch.ArchiveThread` / `ThreadStore.ArchiveThread` | Stale settled record, incomplete request, open park/start, active target, concurrent metadata/publication revision, failed session observation | Compare active/archived store snapshots and stateful session set: successful archive removes only the selected active record and preserves embedded request/history; refusal performs no session stop/removal; explicit stop affects only verified selected session. |
| `menuActionItems` / `reduceTextKey` / `DispatchOperation` | Produce recovery menu actions, edit/cancel/paste invalid or 4096-byte path, refresh selected row mid-form; feed emitted payload through real declaration validation | Every offered action dispatches with required exact address/path fields; canceled form emits nothing; invalid path leaves usable form and visible error; successful result installs the returned terminal and establishes bidirectional test IO. |
| Console operation queue and `AcquireSupervisorLease` | Pause recovery between observation and effect, enqueue archive/resume/continuation from production producers; attempt second owner namespace lease | Event log shows serial effect intervals, one owner acquired, and no duplicated target or premature archive. Cancellation with an open park leaves later actions refused by the durable park guard. |

- [x] Implement the pure tests in `recovery_test.go` / `request_test.go` and the stateful tests in the existing core/Console/command fixtures using these oracles. Use fixed deterministic barriers and fixture events rather than sleep-based races. Run each regression red before its implementation, then the package command recorded in its task. The phase table and generation regression are retained; do not duplicate them in several table-test files.

**PQ-2 — Observation scheduling and bounded retry.** Recovery does not add per-row ledger, checkpoint filesystem, or session queries to ordinary inventory refresh. `ProjectActionableThreads` derives tentative Recover/path-entry visibility from the existing ThreadRecord snapshot, existing `ThreadEvidence` and presence of an embedded continuation; it performs no new IO. A tentative action means “inspect this exact row and recover if proved safe,” not a fresh ownership guarantee. Exact source path/digest from a retained request can be displayed from that already-loaded snapshot. Existing batched detached-session observation retains its current scope/index batching and per-query timeout; no new fan-out is added by #250.

Only an explicit Recover, path import, or archive execution performs fresh recovery observations, for exactly one selected address. Read its latest source ledger and selected checkpoint at most once per attempt, with no worktree/history scan. Reuse one observation bundle within a pure decision, but recollect helper/session/generation evidence at each specified mutation/pre-launch boundary. A path import reads and snapshots bytes once; revision retries reuse those accepted bytes and digest, not a reread of a potentially changing file. Each fresh observation phase gets a child context of at most **5 seconds**, bounded further by the caller deadline; this budget is for a single address's local file/process probes and bounded Zellij query set, not for agent startup or orientation delivery. It allows headroom over existing roughly 250 ms candidate-session queries while keeping a failed inspection responsive. Existing startup/park/delivery timeouts remain their own contracts.

Revision contention permits **8 attempts total**, matching the existing continuation publication/transition cap. Each retry rereads the record and reobserves exact identity/session/generation before its CAS, checks cancellation before IO and before effects, and never extends the caller deadline. Exhaustion returns a visible “thread changed during recovery; inspect and retry” failure; a 5-second observation timeout returns “recovery state could not be checked; retry” with the failed observer named. Neither automatically retries from inventory or treats timeout as absence. Already-persisted safe retirement/snapshot transitions remain available on a later explicit retry; never roll back by fabricating source liveness or a park receipt.

- [x] Add a counting fixture test that refreshes many stale rows and asserts zero additional ledger/checkpoint/session reads from recovery projection. Add exactly-eight-conflicts, cancellation-before-next-attempt, observer-timeout, and eventual-success-on-attempt-eight tests; oracle is read/effect counts plus unchanged snapshot/process state on refused effects. Adapt context-aware observation adapters if an existing non-context API would otherwise evade the deadline; do not simulate cancellation by abandoning an unbounded background goroutine.

**PQ-3 — Ongoing external conformance.** Reuse `.github/workflows/couch-zellij-conformance.yml`, which runs on macOS, installs Zellij, and invokes `make test-couch-zellij-live`. Its actual cadence is weekly Wednesday **16:41 UTC** (`41 16 * * 3`), manual `workflow_dispatch`, and path-filtered pull requests/main pushes. The Make target in `Makefile.local` already compares real Zellij detach/quiescence/park and continuation seed transport with modeled behavior under `PAIR_LIVE_COUCH=1`.

- [x] Add the recovery real-process/session conformance case to the existing `test-couch-zellij-live` test selection, reusing its controlled random session/helper fixture. Extend both PR and main-push workflow path filters for newly added recovery source/test files and owner archive/dispatch changes so this feature runs conformance before merge as well as weekly/manual. The recurring oracle is exact real helper start identity, same surviving session/agent after reattach, no duplicate session after source-absent checkpoint retry, and unchanged unowned resources. This supplements, rather than replaces, the operator's disposable interactive smoke acceptance.

### 2026-09-14 — Implementation audit and acceptance scope

Reason: reconcile the approved concept table and checklist with the implemented
files and actual verification. Earlier design prose is retained as history.
Checked test-command rows denote the recorded focused red/green runs and later
passing package supersets; they do not claim every original regular expression
was rerun verbatim. The source/owned-target regression combines the stateful
advancing-ledger execution cases with pure successive-retry witness tests.

Final concept mapping (ARCH-PURE, ARCH-DRY):

| Concept | Actual classification and location | Delta from design table |
|---|---|---|
| Recovery evidence, decision and tentative inventory choices | PURE; new `couchcore/recovery.go`, modified `actionableinventory.go` | Inventory uses `ProjectRecoveryChoices` from existing record/evidence; execution gathers fresh authority. No refresh IO is introduced. |
| Recovery observation and execution | INTEGRATION; new `couchcore/recovery_execute.go` | Split from the pure file; shared retirement/observation serves continuation and archive. |
| Absence and target-generation authority | PURE; modified `checkpoint/request.go`, `threadrecord/record.go`, `readiness/record.go` | Adds `SourceAbsence`, retained `TargetGeneration` correlation and optional ready-receipt launch ordinal; phase enum remains unchanged. |
| Generation correlation producer/reader | INTEGRATION; modified `wrapcmd/wrap.go`, `couchcore/switchcontext.go`, `couchcmd/run.go` | `OSOrientationStatusReader.Generation` joins the exact receipt/launch generation; production wrapper publishes the ordinal. |
| Continuation preparation/execution | INTEGRATION; modified `continuation.go`, `continuation_recovery.go`; new recovery execution file | `continuation_store.go` is unchanged reuse; publication uses existing revision-checked store mutation. |
| Owner routing, reconciled archive and Console adoption | INTEGRATION; modified operation declarations/dispatch, `detach.go`, `threadstore.go`, Console files and command composition | Existing queue and supervisor lease retained; archive has a final expected-revision guard. |
| Recovery fixture environment | INTEGRATION; new `recovery_execute_test.go`, `recovery_conformance_live_test.go`, command `recovery_acceptance_test.go`, menu recovery tests and `tests/couch-recovery-smoke.sh` | New cases/files reuse existing fixture seams; these are additions, not merely edits to the old fixture files listed above. |
| Last-session absence normalization | INTEGRATION; modified `launcher/session_quiescence.go` and its tests | Live conformance discovered Zellij exit 1 with `No active zellij sessions found.` after deleting its last session. Shared inventory normalization now recognizes exact empty inventory; other errors remain unknown. |
| Artifact source inventory | PURE registry; modified `artifactpath/manifest.go` | Adds the two recovery implementation paths so the repository source-inventory check covers the new files. |

Acceptance uses complementary layers. Portable four-mode command fixtures cover
warm/checkpoint/retired-checkpoint/archive through real dispatch and Console
adoption with deterministic attachment and echo IO. Real-Zellij conformance uses
an actual source helper/process-start identity, fixture-owned sessions, temporary
Git repo/sibling worktree, real ledger/readiness files and stand-in agent
processes; target Couch helpers use the existing fake seam. It proves the warm
agent identity survives and accepts input, exact checkpoint bytes reach the
replacement, retry does not duplicate the session, and unrelated resources stay
unchanged. The interactive smoke first runs that real-Zellij conformance and
then exercises the deterministic Console fixture. Thus Task 6 acceptance rows
are satisfied by these combined layers, not a single production-helper-to-real-
coding-agent end-to-end run. The archive/missing-checkpoint case is portable
acceptance, not a real-Zellij archive drill. The fixture's cancelable stdin reader
allows clean Darwin PTY shutdown; this is not evidence of a production stdin
shutdown fix.

Recorded evidence supplied by the executing owners: portable acceptance race
PASS (2.686s); launcher quiescence/empty-inventory PASS (1.046s); both full
`sh tests/couch-recovery-smoke.sh warm` and `checkpoint` PTY runs exited 0,
including real-Zellij conformance, same-tag Console echo and clean Ctrl+D;
checkpoint output contained exact `RECOVERY-EXACT-250`. Final core/TTY race
passed (94.086s/9.204s), and current `bin/pair`, `bin/couch` and helper binaries
were built successfully (`/tmp/pair250-final-build.log`). The original operator
thread remains untouched. Full repository checks, implementation commit, SDLC
review and operator smoke remain tracked by their unchecked composite rows.
Canceled-Park-await and bounded-observation/counting tests remain unchecked
pending the assigned test owner's evidence; Task 3's combined crash/bad-document
matrix is also left open pending its final coverage mapping.


### 2026-09-14 — Final verification audit mapping

The last audit additions change tests only. `TestCanceledParkAwaitStillBlocksRecoveryAndArchiveUntilWorkerSettles` blocks the real park worker inside publication, cancels its caller, and verifies that the unchanged open transaction refuses recovery/archive while the worker remains active. `TestRecoveryObservationDeadlineCancelsBlockingObserver` waits for the actual five-second observation deadline and separately proves a shorter caller deadline wins. `TestRecoveryCancellationBetweenRevisionAttemptsStopsReobservation` injects a first revision conflict, cancels on the retry read, and proves there is no second observation or mutation. Inventory coverage now refreshes eight stale rows three times with the checkpoint source file missing, observing no source-ledger reads or additional session queries.

Task 3's shared protocol boundaries map to `TestContinuationRecordRoundTripAndCAS` and `TestContinuationConcurrentPublicationDeduplicates` (durable publication), `TestRecoverThreadAbsentSourceRetainsExactSnapshot` and `TestRecoverThreadImportsLegacyCheckpointAfterRetirement` (retired/absent source and exact retained bytes), `TestContinuationRetryDoesNotStealUnrecordedHelperFromLiveOwner` together with the existing start reconciliation/cleanup tests (interrupted claim refuses an unproved-dead owner), `TestContinuationForkFailureRetainsParkAndSnapshot` (shared launcher failure and retry), `TestContinuationRegistrationCrashReconcilesWithoutSpawn` and `TestRecoverContinuationRetryDistinguishesOwnTargetGeneration` (registration crash and correlated retry), and existing submitted/indeterminate delivery reconciliation tests (one target, no automatic duplicate input). The new refusal integration cases exercise missing paths, a directory that cannot be read as a regular checkpoint, and a valid document for the wrong agent; they assert unchanged records and zero launches. `TestCheckpointReadFile` retains the bounded oversized-file oracle at the reused reader seam instead of duplicating its parser matrix in Couch.

Final added-test command: `go test -race ./cmd/internal/couchcore -run 'RecoverThreadRefusesUnproved|RecoveryObservationDeadline|RecoveryCancellationBetween|CanceledParkAwait|RecoveryInventory' -count=1` — PASS, 8.222s. The full repository suite was independently reported passing by root in `/tmp/pair250-final-go-test.log`. Verification record for this focused run: `/tmp/pair250-audit-verification.txt`. No new implementation gap was observed; remaining commit/gate/operator acceptance checkboxes stay with root.
