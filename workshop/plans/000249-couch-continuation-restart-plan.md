# Couch Continuation Restart Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Continue a Couch thread into a fresh conversation from the exact saved checkpoint, including after warm reattachment and from another worktree, with durable failure and retry semantics.

**Architecture:** Persist one bounded continuation request on the existing thread record before changing process ownership. Couch parks the exact source and uses its existing tracked fresh-launch path; a Console lifetime worker feeds the existing operation queue. The writer and standalone launcher carry an exact checkpoint reference instead of resolving a slug again after teardown.

**Tech Stack:** Go, existing portable ThreadStore and journal, declared Couch operations, launcher Runtime, Zellij, orientation delivery receipts.

---

## Scope and accepted contract

Issue: `workshop/issues/000249-couch-continuation-restart.md`. The operator authorized implementation after #248 shipment, with another smoke checkpoint after #249. #248 established that warm attachment needs process/session ownership proof, independently of native conversation binding. This issue gives fresh continuation its own durable intent and executor; #250 will reconcile already stranded threads and invoke that executor rather than invent a competing restart path.

Keep the Pair address. Start a fresh native conversation seeded from the saved checkpoint. A native binding to the old conversation is never continuation evidence. The original helper exits through the existing park lifecycle; Couch launches and adopts a replacement helper. The inner writer must not kill Zellij or write the outer launcher's legacy restart marker for hosted continuation.

Checkpoint submission means the request was durably accepted. Completion means the exact fresh target's orientation receipt confirms submission; neither helper registration nor a successful writer commit alone proves this. Failure retains the checkpoint and its source/target evidence. Failed requests require explicit retry; repeated polling cannot create repeated conversations.

No general restart manager, new server, generic RPC transport, automatic stale-thread repair, or new thread incarnation status is introduced. Review `construct/vocabulary` via the vocabulary skill before changing an existing lifecycle vocabulary; the new request phase belongs to the shared request model and leaves incarnation/park/start vocabularies intact.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ContinuationRequest` and its phase/event reducer | `cmd/internal/checkpoint/request.go` | new |
| `Checkpoint` | `cmd/internal/checkpoint/checkpoint.go` | new |
| `ThreadRecord` | `cmd/internal/couchcore/thread.go` | modified |
| `threadrecord.Record` | `cmd/internal/threadrecord/record.go` | modified |
| `RestartMarker` and `restartPlan` | `cmd/internal/launcher/markers.go` | modified |
| Continuation operation result implementing `StartedChild` | `cmd/internal/couchcore/continuation.go` | new |

`Checkpoint` owns exact absolute source path, immutable UTF-8 bytes, SHA-256 digest, and schema version. Snapshot size is at most 256 KiB; reject empty, oversized, non-regular, unreadable, or invalid text before publication. The source path is provenance, not a dependency after acceptance. A shared leaf package lets launcher, persistence validation, and Couch agree without an import cycle. Pure validation recomputes the digest and checks the path and size; IO constructs a value by reading a bounded file once.

Each `ThreadRecord` owns zero or one `ContinuationRequest`. The request owns its checkpoint, request ID, exact source agent/session/launch ordinal, source helper PID/start identity, creation time, phase, attempt ID when running, and bounded failure text. An optional target receipt identifies the exact fresh launch attempt; it must not become an independent boolean that can contradict the phase. The shared type appears in both persistence and Couch models; conversion and clone functions preserve it without aliasing mutable snapshot storage. Old records lacking the optional field remain valid.

Source launch ordinal is required: nonce is not universal because ordinary and cold launches clear it. Compare the latest launch for the address across **all agents**, so an old pane cannot regain authority after an agent switch. A warm reattach preserves the running pane's ordinal but changes the helper identity; publication captures the current helper from ThreadStore rather than trusting inherited helper metadata. At execution, prove the same current launch ordinal and uniquely owned source session before refreshing an obsolete helper observation. The source generation is authoritative; a helper replacement alone must not invalidate a pending request.

A request ID is the SHA-256 of canonical address, source launch ordinal, and checkpoint digest, making repeated publication idempotent. Its ID is stable across retries; an attempt ID names one fresh launch and orientation receipt. Allocate a new attempt only after reconciliation proves the preceding attempt cannot still own a target. This separates retrying observation from authorizing a second launch.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Exact writer handoff | `cmd/internal/continuationcmd/continuationcmd.go` | modified | committed checkpoint and `pair` subprocess |
| `Runtime.RequestCouchContinuation` | `cmd/internal/launcher/runtime.go`, `cmd/internal/launcher/osruntime.go` | new | bounded `couch --internal request-continuation` subprocess |
| Request publication and transitions | `cmd/internal/couchcore/continuation_store.go` | new | ThreadStore revision CAS and existing journal |
| Continuation executor | `cmd/internal/couchcore/continuation.go` | new | `ParkExpected`, `launchTrackedThread`, exact receipts |
| Console request provider and worker | `cmd/internal/couchcmd/run.go`, `cmd/internal/couchtty/console_continuation.go` | new | hosted thread slots and existing operation queue |
| Existing orientation delivery | `cmd/internal/couchtty/console_switchagent.go`, `cmd/internal/couchcore/switchcontext.go` | modified | exact-attempt ready/submit receipt |
| Exact standalone marker IO | `cmd/internal/launcher/osruntime.go`, `cmd/internal/launcher/compaction.go` | modified | durable restart marker and launch acknowledgment |

Reuse portable temp-directory ThreadStore, `FakeRunner`, fake process identities, fake Pair session/lifecycle IO, and launcher fake Runtime. Extend their state model to record source alive/parked, unique session ownership, blocked helper acknowledgment, fresh registration, and orientation submission separately. A gated receipt must be able to arrive after cancellation or after another request generation. Do not use a callback count as the acceptance oracle.

## Ordering and recovery contract

The reducer takes typed evidence/events; the IO shell gathers proof, performs CAS, and executes only the resulting allowed operation. Illegal combinations fail validation on decode as well as on mutation.

| State and event | Next state and effects |
|---|---|
| No request; validated exact source submits | `pending`; atomic record write stores snapshot and source identity before any park effect |
| Existing same request/source/digest submitted again | Return existing receipt; no new slot, park, or launch |
| `pending` or `running`; different request submitted | Refuse replacement with actionable current request status |
| `complete`; next current generation submits | Replace slot only after prior target generation is no longer active or the new request proves it is that generation's authorized continuation |
| `pending`; owner claims execution | CAS to `running` with attempt ID; then recheck source generation and exact helper before `ParkExpected` |
| `pending`; worker queue full or Console closes | Remain `pending`; later owner can discover it, no detached worker survives Console |
| `running`; source still exact and park not begun | Continue existing park protocol once; never issue standalone kill |
| `pending`/`running`; old helper died but exact source session survives detached | Reattach through #248 proof, refresh helper observation after matching source ordinal, then continue request; active clients or ambiguity remain diagnostic |
| `running`; park incomplete or observer unavailable | Record `failed` with evidence; retain open park/start transaction and checkpoint, do not manufacture an empty thread |
| `running`; exact source verified parked, no target claim | Commit existing fresh start claim with the same attempt ID; launch through `launchTrackedThread(Fresh=true)` |
| `running`; existing matching start claim/target found | Reconcile that attempt using existing start cleanup/registration evidence; do not launch another helper |
| `running`; registration succeeds | Adopt returned child and observe orientation; remain `running` until exact submitted receipt |
| `running`; exact submitted receipt arrives | CAS to `complete`, retaining last checkpoint and receipt |
| `running`; submit times out, target exits, or delivery fails | `failed`; explain whether text may already be present; retain target/attempt evidence and checkpoint |
| `running`; process interruption after any durable step | Next observer reconstructs from request plus park/start/receipt facts; never treat missing local callback as missing launch |
| `failed`; explicit retry with live registered target | Reobserve same receipt / show copy-and-inspect action; do not park or launch a duplicate target and do not blindly resubmit uncertain text |
| `failed`; explicit retry with target conclusively absent and no unresolved claim | CAS to `running` with new attempt ID, reuse same immutable snapshot and existing fresh executor |
| Any phase; stale completion for other request/attempt | Ignore; it cannot overwrite a newer request |
| Any phase; corrupt/unknown persisted shape or ambiguous identity | Return diagnostic, preserve bytes/state, perform no destructive effect |

Two actors publishing or retrying race through revision CAS, not a process-local mutex alone. Metadata edits can advance a revision; reread and repeat validation rather than overwriting their changes. Park and fresh start remain existing transactions and must reject competing transitions. The continuation guard is checked at each public transition entry and immediately before its CAS, including cold resume, agent switch, relaunch, and archive. `CommitStartClaim` rejects an active continuation unless `StartEvent.Nonce` equals its attempt ID; the narrowly authorized warm-source reattachment proves the same source generation and uses existing #248 ownership checks. Warm attachment of a surviving source is allowed so owner interruption does not strand an accepted request; it does not authorize a different cold conversation. Generic park races are serialized by `ParkExpected` revision CAS and reconciled from existing durable park proof. Metadata, inspection, orientation status, and the continuation executor remain available.

The most dangerous ordering is **fresh registration succeeds, but Couch dies before marking request complete**. Tests must interrupt there and restart the consumer over the same store: its first action is reconciliation of the recorded attempt, never a new launch.

## Storage, lifetime, and operating envelope

| Surface | Budget and basis | Exceeding it / end of life |
|---|---|---|
| Embedded checkpoint | 256 KiB UTF-8 bytes per thread, design choice; measured 18 current checkpoints are 3,222–12,517 bytes (146,127 total), so cap exceeds current largest by >20×; enforce writer and reader | Reject before park; one active or last-completed snapshot replaces the previous slot |
| Materialized checkpoint | One deterministic file per thread, at most 256 KiB, needed because orientation passes a file reference | Materialize only after source park; atomic replacement only when accepted successor request is authorized; archive removes derived live copy after archived record durably owns snapshot; restore/retry can recreate it |
| Request/diagnostic metadata | Fixed field limits, failure text at most 4 KiB, one request slot | Truncate only explanatory diagnostic, never identity/path/digest; malformed identity refused |
| Polling | One worker per Console, 500 ms, operator/design choice; only hosted thread request slots | No whole data-root scan, no native resolution, no Zellij process per poll, no per-keystroke work; queue saturation leaves durable request pending |
| Concurrency | Existing single operation queue, existing capacity and one exact request key per address/request | Duplicate enqueue suppressed; no goroutine per request; worker canceled and joined on Console exit |
| IO/launch latency | Submission subprocess bounded (initial 10 s choice); existing park/start/registration deadlines | Timeout returns explicit durable-status guidance; no automatic repeated launch |
| Memory/disk scale | O(hosted threads) IDs plus one bounded snapshot in the executing operation; disk O(threads) not O(continuations) | Test representative 100 hosted slots and bounded reads; avoid copying all snapshots into every UI frame |
| Network/GPU | No new network or GPU interaction; existing agent startup unchanged | N/A for new mechanism |

ThreadStore archive is durable preservation, not checkpoint deletion: retain embedded source snapshot with the archived record. The materialized file is derived and can be removed only after archive commit; failures surface as cleanup warnings and retryable cleanup, without losing the snapshot. No new per-attempt file history or retry daemon. Existing orientation ready files and lifecycle journals retain their current owners/cleanup; do not introduce an extra receipt file when the request can record its confirmed receipt.

## Architecture review

- **ARCH-DRY:** Reuse `ParkExpected`, `CommitStartClaim`, `launchTrackedThread`, `StartFreshExisting`, orientation request/status, declared operations, and operation queue. Shared checkpoint request type prevents divergence between the ThreadRecord mirror and launcher.
- **ARCH-PURE:** Digest/schema validation and state/event legality live in the leaf package. Process proof, file materialization, CAS, and subprocess calls remain small IO seams.
- **ARCH-PURPOSE:** Deliver writer-to-live-target acceptance, cross-worktree, warm reattach, failure/retry, and exact standalone path propagation. Enumerate restart-marker consumers (`continue`, normal restart, rename) and close the hosted established-address mismatch for each safely in this issue.
- **ARCH-MOCK:** Stateful portable fakes model park, ownership, helper registration, and submitted receipt independently. Optional live smoke compares that model with real Zellij/agent behavior after build; deterministic CI never touches operator state.
- **ARCH-CONSTRAINTS:** Polling is off UI/keystroke paths, bounded to hosted slots and Console lifetime; snapshot and subprocess budgets are enforced on the writer, not merely the reader.
- **ARCH-SECURE:** CLI/env, source files, persisted JSON, and receipt files cross trust boundaries. Validate exact scope/tag agreement, positive source ordinal, supported agent, exact session and helper identity. Use argv arrays; never place checkpoint body in argv/logs. Strict decode rejects unknown/trailing/truncated data and digest mismatch. No credentials created or required; private local snapshot files use owner-only permissions.
- **ARCH-ORDER:** The table above is the reducer/test matrix. Scheduling, cancellation, stale receipts, and crash windows are injected with channels/barriers, not sleeps.
- **ARCH-FUNERAL:** One bounded snapshot/request and one derived file per thread; supersession and archive cleanup are explicit. The worker dies with Console; blocked/start helper cleanup remains owned by the tracked launch path.

## Chunk 1: Durable request and end-to-end handoff

### Function-level implementation and verification contract

This is the test strategy for Tasks 1–5. The task checkboxes below describe execution order; the table identifies each risky function, its adversarial class, and its mechanical oracle. The state/event table remains the architectural contract rather than a second case inventory.

| Function surface | Implementation strategy and adversarial class | Mechanical guard |
|---|---|---|
| `checkpoint.New`, `Checkpoint.Validate` | Absolute source provenance, bounded UTF-8 snapshot, required continuation header/NEXT ACTION and recomputed SHA-256; malformed or oversized input | Pure table test; valid construction round-trips; any changed digest/body mismatch refuses |
| `checkpoint.ReadFile` | Open regular file, bounded read of MaxBytes+1, then pure construction; disappearance/non-regular file | Temp-folder test proves error before any published request or teardown |
| `checkpoint.RequestID`, `Request.Validate`, `Advance` | Deterministic source+digest ID, phase/event reducer with attempt identity; stale completion and illegal event ordering | Pure table tests assert next value and permitted effect; rejected transitions preserve original value |
| `threadrecord.DecodePersisted`, `Validate` | Optional request decoded strictly and delegated to shared validator; absent old field, unknown/trailing/truncated request | Existing record fixture round-trip plus strict rejection; old records retain acceptance |
| `toPersistedThreadRecord`, `fromPersistedThreadRecord`, `cloneThreadRecord` | Preserve the request and deep-copy target pointers; metadata update and aliasing | Round-trip equality and mutation-isolation unit tests |
| `ThreadStore.RequestContinuation`, `ThreadStore.AdvanceContinuation` | Use `UpdateExistingThread` CAS and existing journal; two publishers and interrupted commit | Two stores on one portable directory yield one winner; reopening observes one complete snapshot/request |
| `OSContinuationSourceReader.Read` | Read exact ledger/index and latest all-agent launch; old-agent ordinal or malformed authority | Stateful temp ledger/index fixture rejects obsolete proof without process effects |
| `continuationcmd.run`, `newContinueRestartCmd` | Pass committed absolute path to child; child failure follows durable commit | Temp Git fixture captures actual argv/path and preserves checkpoint while returning failure |
| `ParseArgs`, `RunCLI`, `runCompaction` | Resolve once, hosted request submission before any teardown; cross-worktree and stale env | Stateful Runtime proves exact path transported and zero hosted marker/kill calls |
| `ReadRestartMarker`, `AcknowledgeRestartMarker`, `WriteRestartMarker` | Versioned snapshot plus exact attempt acknowledgment; concurrent successor marker and IO failure | Temp marker store retains successor, retains failed attempt, and never kills after a failed write |
| `planRestart`, `RunLaunch`, `runCreate` | Preserve exact snapshot, fresh-only retry and checked seed write; source removed or replacement fails | Stateful Runtime enforces real claim policy and asserts final draft bytes, no native resume and no repeated automatic attempt |
| `Couch.RequestContinuation` | Read snapshot, validate current source/sole helper, CAS pending; source replacement during preparation | Injected source reader plus real ThreadStore proves stale revision/generation cannot publish authority |
| `Couch.Continue`, `Couch.RetryContinuation` | Reconcile first, CAS attempt, verified park, materialize, existing fresh claim/launch; failure between each durable step | Stateful fake process/session/start receipts plus real store prove one owner, retained checkpoint and no duplicated target |
| `Couch.ReconcileContinuation` | Observe exact ready receipt and phase/attempt, then CAS complete/failed; registration precedes callback or receipt belongs to old attempt | Reopen same store after barrier-controlled interruption; no second StartBlocked on matching live receipt |
| `ThreadStore.CommitStartClaim`, continuation admission guard | Exact request attempt or proven warm shape only; competing cold resume/switch/relaunch/archive | Existing transition tests with active request assert refusal before side effects |
| `Couch.ContinuationRequests` | Read only supplied addresses, return metadata without body; missing/corrupt slot isolation | Portable 100-slot fixture counts accesses and asserts unrelated store addresses are never read |
| `Console.watchContinuations`, `acceptContinuationRequests` | One lifetime worker, bounded channel and queue, persistent accepted-address set; repeated scans/queue full/zero panes | Controllable tick/provider barriers prove one queued request and worker join on stop |
| `Console.onExit`, `finishOperation` | Exact expected-source marker and keep-open surface; failure-before-exit versus exit-before-failure | Both explicit event orders preserve panel and attach only returned replacement |
| `Console.finishOrientation`, continuation receipt handler | Persist exact submitted outcome through declared status operation; canceled/partial delivery | Controlled receipt test retains recoverable request and never automatically resubmits uncertain text |
| `ParseCLI`, `operationOwnsLive`, `runTypedOperation` | Declared retry owner bootstrap, singleton lease, generic StartedChild adoption | Actual CLI acceptance with fake runtime observes lease ownership and terminal attached to replacement |

The end-to-end test enters through the real continuation writer and follows its actual serialized output through request publication, verified source teardown, production existing-address registration, replacement receipt and Console input/output. It runs both initial-host and warm-reattached source setups; a component callback counter is not its success oracle.

**Live conformance cadence:** Extend the existing `couch-zellij-conformance` workflow path filters for checkpoint/continuation code and its `make test-couch-zellij-live` target with the focused continuation lifecycle conformance fixture. The workflow owns the check on relevant PRs/pushes and its existing weekly Wednesday schedule. Use isolated temporary stores and a deterministic agent stand-in under real Zellij to compare session ownership, fresh registration and readiness/seed transport; user smoke separately checks actual agent prompt submission after `make build`. This adds no unattended calls to paid agents.

### Task 1: Exact checkpoint model and persisted slot

**Files:** Create `cmd/internal/checkpoint/checkpoint.go`, `cmd/internal/checkpoint/request.go` and colocated tests. Modify `cmd/internal/threadrecord/record.go`, its tests, `cmd/internal/couchcore/thread.go`, and `cmd/internal/couchcore/thread_test.go`. Create `cmd/internal/couchcore/continuation_store.go` and tests.

- [x] Write the red tests specified by the function-contract rows for `checkpoint`, persisted records, conversions and ThreadStore.
- [x] Run `go test ./cmd/internal/checkpoint ./cmd/internal/threadrecord ./cmd/internal/couchcore -run 'TestCheckpoint|TestContinuation|TestThreadRecord' -count=1`; record expected red failures before implementation.
- [x] Implement `Checkpoint.Validate` with byte length, regular source-path representation, SHA-256 recomputation, and explicit schema. Implement request reducer/validation; inject timestamps/IDs rather than reading clock/entropy in pure code.
- [x] Add optional request field to both record shapes and every conversion/clone. Delegate validation to shared model. Bound record decode consistently with existing store conventions; do not silently omit request fields during metadata writes.
- [x] Implement publication/transition helpers through `UpdateExistingThread` revision CAS. Provide materialization of validated snapshot via atomic owner-only write to deterministic thread path, invoked only after source park; revalidate digest on reads. Preserve embedded bytes as source of truth if derived copy is absent/corrupt.
- [x] Run the focused command to green and commit this coherent unit.

### Task 2: Writer and launcher preserve exact identity

**Files:** Modify `cmd/internal/continuationcmd/continuationcmd.go`, `run_test.go`, `cmd/internal/launcher/args.go`, `args_test.go`, `runtime.go`, `compaction.go`, `createflow.go`, `osruntime.go`, `markers.go`, and affected launcher tests. Find the actual `RunCLI` implementation before adding resolution there.

- [x] Write the red tests specified by the writer, CLI, marker and launcher rows of the function contract.
- [x] Add `pair continue --checkpoint <absolute-path>` parsing while retaining user-facing slug compatibility. Resolve slug once at CLI ingress; explicit missing/unreadable checkpoint is an error before any source teardown. Do not allow `ContinueDoc` to silently become empty.
- [x] Change writer `newContinueRestartCmd` and its restart callback from slug to exact committed path. Return restart failure with checkpoint path rather than logging success; do not undo the durable writer commit.
- [x] Add Runtime `RequestCouchContinuation` and bounded OS subprocess implementation using argv to `couch --internal request-continuation <path>`. Hosted `runCompaction` calls it and returns acceptance/failure; it neither writes legacy marker nor kills the source.
- [x] Validate Couch and Pair scope/tag agreement, required agent/session/ordinal at the declared direct-store operation boundary. Read latest launch across all agents for the exact address and current sole live helper from ThreadStore. Reject stale pane generations before publication.
- [x] Fix standalone marker representation to carry exact validated checkpoint data/reference through `planRestart`, with write errors returned before kill and acknowledgment only after successful replacement. Preserve old marker compatibility explicitly; unreadable continuation intent must not fall back to an unseeded launch.
- [x] Remove final `filepath.Base(opts.ContinueDoc)` prompt truncation: the final seed names the exact validated/materialized absolute path and digest. Ensure FreshRequired argument sanitation does not clear deliberate orientation/checkpoint seed.
- [x] Run `go test ./cmd/internal/continuationcmd ./cmd/internal/launcher -count=1` and commit.

## Chunk 2: Owner execution, recovery, and UI adoption

### Task 3: One continuation executor

**Files:** Create `cmd/internal/couchcore/continuation.go` and `continuation_test.go`. Modify `couch.go`, `ops.go`, `operationdispatch.go`, their tests, `resume.go` (resolve exact current filename), `switchagent.go`, and archive/relaunch entry points located by symbol search.

- [x] Write the red tests specified by the Couch publication, execution, reconciliation and admission rows of the function contract.
- [x] Declare `request-continuation` as `ExecuteDirectStore`, `EffectMetadata`, `ConfirmNone`, `PresentationInternal`; reuse normal CLI binding/dispatch rather than adding a special CLI parser. Declare live-owner execution/retry operations with typed result and exact request ID.
- [x] Implement execution as reread/validate → CAS running+attempt → exact source recheck → `ParkExpected` → materialize checkpoint → fresh profile with preserved non-resume parameters → `CommitStartClaim` using attempt → `launchTrackedThread(Fresh=true, Orientation=...)`. Reuse `ValidateFreshAgentArgs` and the existing fresh-profile builder.
- [x] Build a short orientation request naming immutable materialized checkpoint path/digest, not a huge embedded prompt. Return `StartedChild` immediately after registered launch so Console owns the live handle while receipt observation completes.
- [x] Implement reconciliation before retry: inspect matching start claim, target attempt registration, process/session proof, and orientation receipt. Exact live registered target means observe/adopt it, not spawn again. Unknown state means actionable failure; conclusively absent owned target permits existing cleanup followed by a fresh attempt.
- [x] Add a shared continuation-intent guard to cold resume, switch-agent, relaunch, and archive entry points, with the enforcing `CommitStartClaim` nonce check. Preserve metadata, inspection, and proven same-generation warm-source reattachment; refresh the helper identity from exact current proof. Check guard under the same revision discipline as each mutation; executor's authorized transitions must not bypass source identity checks.
- [x] Run `go test ./cmd/internal/couchcore -run 'TestContinuation|TestSwitchAgent|TestResume|TestArchive' -count=1` and commit.

### Task 4: Console discovery, operation queue, and receipt persistence

**Files:** Create `cmd/internal/couchtty/console_continuation.go` and tests. Modify `cmd/internal/couchcmd/run.go`, `cmd/internal/couchtty/console.go`, `console_completion.go`, `console_switchagent.go`, `console_menu.go`, menu action/render files, and relevant tests.

- [x] Write the red tests specified by the Console and CLI rows of the function contract.
- [x] In `wireResolver`, inject a provider reading only request slots for Console-hosted addresses; it performs no native-binding resolution or session probing. Add a single lifetime-bound 500 ms worker that deduplicates request keys and enqueues through `operationQueue`.
- [x] Route operation completion through existing child adoption and expected-exit/focus handling used by switch-agent. Preserve unrelated focused thread when continuation is for a background hosted thread. Do not reuse `MenuOperationOrigin.Background`, which routes completion into the reattach reducer; add a narrowly named preserve-focus property or continuation origin. Mark the exact source exit expected when the continuation operation is accepted, including operation failures after park, not only successful result adoption. Retain its request address independently of live panes until completion, archive, or Console shutdown; the provider scans that bounded tracked set as well as current hosted addresses. Keep the panel open on its expected last-source exit even if the request already became failed. Cancellation and full queues leave request durable.
- [x] Extend exact-attempt orientation completion handling to persist `complete` only on submitted receipt, or `failed` with delivery uncertainty. Durable reconciliation must not rely solely on this in-memory callback; worker/executor can reconstruct receipt state after restart.
- [x] Render pending/running/failed continuation status and an explicit Retry continuation action invoking the same declared executor. Live target with uncertain submission offers existing copy/inspect affordance without automated duplicate submission. No automatic retry for failed phase.
- [x] Make the declared internal `retry-continuation` operation an explicit owner bootstrap in `operationOwnsLive`, with a required thread reference and current-repo-scope binding. It acquires the same singleton lease and Console as start/resume. Generalize the final Console result handling to `StartedChild` so its replacement is adopted. On startup refusal caused by a retained request, print the exact retry command; when a Couch owner already runs, the switcher action remains the route. This makes recovery reachable after supervisor death even when no root thread can cold-resume.
- [x] Run `go test ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'TestContinuation|Test.*Operation|Test.*Orientation' -count=1` and commit.

### Task 5: Hosted acceptance and restart consumer sweep

**Files:** Create `cmd/internal/couchcmd/continuation_acceptance_test.go`. Extend launcher restart/marker acceptance tests and existing portable stateful fixtures. Modify `README.md`, `atlas/couch.md`, and `atlas/index.md` only if adding a new atlas file.

- [x] Implement and run the end-to-end acceptance objective stated after the function-contract table, composing its actual producer output across the named integration seams.
- [x] Enumerate every restart-marker producer/consumer with `rg 'WriteRestartMarker|TakeRestartMarker|RestartMarker|planRestart' cmd bin`. For normal restart and rename, resolve hosted same-address registration and unsupported address-changing ownership before teardown; ensure no path retains Couch identity while invoking initial reservation registration. Keep normal standalone behavior in regression tests.
- [x] Document accepted/pending/complete distinction, exact checkpoint retention and retry, replacement-helper ownership, and limitations of uncertain delivery. Give smoke steps: hosted checkpoint from sibling worktree; same address reappears; new agent references a unique checkpoint token; failed preflight leaves original session usable.
- [x] Run `env -u PAIR_SESSION_ID -u PAIR_TAG make test`, focused race tests for four affected packages, `make build`, and `git diff --check`. Capture evidence; avoid rerunning broad suites absent new changes.
- [ ] Tick issue/plan steps and record evidence/decisions. Close once with `sdlc close --issue 249 --verified '<actual test evidence>'`; address gate findings. Pause for operator smoke with built binaries and exact steps before starting #250.

## Resolved implementation choices

The parent API audit resolves the four choices below. These strategies govern the corresponding tasks above.

1. **Standalone transport:** Keep standalone's existing outer-loop ownership. Versioned continuation markers carry the bounded checkpoint snapshot, a unique attempt ID and exact source provenance; ordinary legacy markers retain their existing grammar. Replace destructive take with read plus generation-checked acknowledgment for the versioned marker. The current loop remembers which attempt it is executing: a failed replacement returns with the marker intact, and never re-enters that same attempt automatically. A successful blocking replacement handoff can acknowledge its exact marker when it returns; acknowledgment cannot erase a newer marker written by that replacement. Seed the complete snapshot into the draft, checking its write before launch, so the original file can disappear safely. Add explicit standalone `pair continue --retry <tag>` to read the retained exact pending snapshot; require the exact session absent and fresh-only creation, refusing an occupied session. This acknowledgment records successful handoff, not automatic prompt submission: standalone keeps its existing draft workflow. Couch completion alone uses submitted orientation receipts. No per-attempt snapshot files are needed for standalone.
2. **Post-registration reconciliation:** `OSOrientationStatusReader.readReady` already validates tag, agent, nonce, live wrapper PID and exact Pair session. The request retains its attempt ID across promotion, so it can query this receipt even after `Start` is cleared. Persist the registered target process identity on the request when observed. A crash before that write is reconciled by the matching ready receipt plus current incarnation/start claim; never infer absence from the missing request update. If the target session survives but its helper does not, reattach it through #248's proof-bearing warm path and observe the same receipt. An incomplete park uses existing `PairLifecycle.Recover`; unknown process/session/start evidence remains a visible refusal. A fresh retry is permitted only after proving the old attempt absent. Add an explicit start shape to the continuation guard so proven warm attachment can preserve an existing source or target without granting cold resume authority.
3. **Hosted normal restart/rename:** Couch already intercepts Alt+n through `onRelaunchHotkey`, executing its tracked relaunch operation. Inner `pair restart`/address-changing `pair rename` cannot safely use legacy markers under Couch ownership; refuse these hosted marker routes before any artifact mutation or teardown, explaining the supported Couch relaunch and name actions. Standalone routes remain covered. Do not add a second generic hosted restart protocol to fix the inner command. The refusal fixes the same established-address failure class without silently killing a working session.
4. **Receipt and lifecycle locality:** `couch --internal request-continuation` uses the existing direct-store declaration and `publish-description`-style implicit environment binding in `couchcmd/run.go`. Require matching Couch/Pair scope and tag, agent, session, positive ordinal. An injected source reader in couchcore resolves the exact ledger through `artifactpath`, rejects malformed authority, chooses its latest launch across all agents, and compares ordinal/agent/session. `RequestContinuation` reads the current sole live helper from ThreadStore, captures it in the request, and publishes through revision CAS. Recheck both source generation and current owned helper before park. This read-only reader and the existing orientation reader are reused during reconciliation; no launcher-to-couchcore import is introduced.

The request reader returns only small metadata to the Console worker. Snapshot bytes stay in the store/executor and never enter menu frames. Materialization happens after source park, so the old live agent's checkpoint file cannot be overwritten by preparation of its successor.

## Revisions

- 2026-09-14: Parent API audit clarified source-generation identity, publication idempotency, and cleanup ordering. Same-ordinal warm source reattachment remains supported after owner interruption; source helper identity is refreshed only from exact proof. Request ID derives from address/ordinal/digest; `CommitStartClaim` enforces continuation attempt ownership; materialized checkpoint is written after source park. Added observed checkpoint sizes as the snapshot-budget basis.
- 2026-09-14: Resolved standalone marker acknowledgment/retry, exact-ready reconciliation after registration, pre-teardown refusal of unsupported inner hosted restart/rename, and direct-store source-reader placement. These close the draft's open API questions while retaining the existing standalone draft and Couch relaunch workflows.
- 2026-09-14: Fresh plan review found the failure-before-last-exit ordering could still close Console. Track accepted request addresses beyond pane lifetime and preserve the failure panel in either event order. Parent audit also added explicit retry owner bootstrap so supervisor interruption cannot leave recovery behind an unreachable live-owner-only action.
- 2026-09-14: Plan gate PQ-1 requested named function strategies rather than prose case inventories. Added the canonical function/mechanical-oracle table covering reducers, decode/clone, CAS, transport, executor/reconciliation, admission, polling, exit and CLI attachment. PQ-2 requested conformance ownership/cadence; use the existing changed-path and weekly Zellij workflow, extending its isolated fixture to continuation registration and seed transport.
- 2026-09-14: PQ-1 round 2 clarified that the table must replace the duplicated task-level case inventories. Compressed all five tasks' testing checkboxes to reference the canonical function contract; retained execution commands, architectural ordering and the single end-to-end acceptance objective.

- 2026-09-14: Implementation recovery review refined the request evidence without adding a second lifecycle owner. `SourcePark` links the request to its successful, closed source-helper park receipt, preserving exact teardown proof after start promotion clears the active park marker. `Target.ObservedAt` records the first target observation and survives helper reattachment; a missing submission receipt becomes a durable failed request after 30 seconds rather than resetting the timeout on every poll. Unknown delivery remains observation-only on explicit retry while that target may exist.
- 2026-09-14: A newly warm-attached source returns `SourceReattached` with its started child before the executor parks it. Console adopts that child and allows the next queued execution to continue; target reattachment likewise returns its handle before receipt reconciliation. This preserves terminal ownership across recovery failures. Writer/launcher transport carries the validated digest to request ingress, where `ExpectedDigest` binds publication to the exact saved bytes and rejects an intervening file edit.
- 2026-09-14: Core integration verification added the 100-slot bounded-address discovery fixture: requested addresses are deduplicated, an unrelated corrupt slot is never read, and projected JSON omits snapshot contents. Operation argument descriptions and exact `continuation-status` vocabulary classifications were completed together. The artifact guard now recognizes only direct named literal fields and direct literal return values with explicit function/site/count allowances; path calls and constructed strings remain rejected (ARCH-DRY).


### 2026-09-14 — BR-1: Core-concepts traceability against the closing window

Audited all six pure-entity rows and all seven integration rows against the
pinned `7800e968..f5fa755b` diff and the symbols at its final commit. This table
supersedes the original Core-concepts classifications and locations where they
differ; the original design remains above as the planning record. `New` and
`modified` describe the named surface; file changes are stated separately when
a new surface is introduced in an existing file. `Reused unchanged` explicitly
records dependencies outside the implementation diff (ARCH-DRY).

| Original concept row | Actual symbol/location and classification in the pinned diff |
|---|---|
| `ContinuationRequest` and phase/event reducer | Implemented as `checkpoint.Request` and `checkpoint.Advance` in **new** `cmd/internal/checkpoint/request.go`; the planned concept name was descriptive, not the exported Go type name. |
| `Checkpoint` | **New** `Checkpoint` in **new** `cmd/internal/checkpoint/checkpoint.go`; original classification confirmed. |
| `ThreadRecord` | **Modified** in `cmd/internal/couchcore/thread.go`; adds the optional shared request field. |
| `threadrecord.Record` | **Modified** in `cmd/internal/threadrecord/record.go`; persists the same optional request. |
| `RestartMarker` and `restartPlan` | **Modified** in `cmd/internal/launcher/markers.go`; original classification/location confirmed. |
| Continuation operation result implementing `StartedChild` | **New** `ContinuationResult` and its `Started` method in **new** `cmd/internal/couchcore/continuation.go`; original classification confirmed. |
| Exact writer handoff | **Modified** in `cmd/internal/continuationcmd/continuationcmd.go`; `newContinueRestartCmd` carries the committed path/digest. |
| `Runtime.RequestCouchContinuation` | **New interface method** in **modified** `cmd/internal/launcher/runtime.go`; its implementation `OSRuntime.RequestCouchContinuation` is in **new** `cmd/internal/launcher/checkpoint_io.go`, correcting the planned implementation location `osruntime.go`. |
| Request publication and transitions | **New** `PublishContinuation` and `AdvanceContinuation` in **new** `cmd/internal/couchcore/continuation_store.go`; original classification confirmed. |
| Continuation executor | **New** executor in **new** `cmd/internal/couchcore/continuation.go`, with reconciliation/retry split into **new** `cmd/internal/couchcore/continuation_recovery.go`; original classification confirmed and final split recorded. |
| Console request provider and worker | **New integration** wired by **modified** `cmd/internal/couchcmd/run.go` through `SetContinuationProvider(c.ContinuationRequests)`; worker and completion handling are in **new** `cmd/internal/couchtty/console_continuation.go`. The original row's `new` described the integration, not both files. |
| Existing orientation delivery | **Reused unchanged**, not modified: `cmd/internal/couchtty/console_switchagent.go` and `cmd/internal/couchcore/switchcontext.go` have no diff. New receipt caller `ReconcileContinuation` in `continuation_recovery.go` invokes existing `ReadOrientationStatus`; `executeContinuation` in `continuation.go` supplies an existing `orientation.Request` to the tracked launch path. `finishContinuationOperation` in `console_continuation.go` populates the existing menu orientation-copy state. Continuation uses its own durable status polling, not the unchanged switch-agent `watchOrientation` worker. |
| Exact standalone marker IO | **Modified** wrappers in `cmd/internal/launcher/osruntime.go` and producer in `cmd/internal/launcher/compaction.go`; strict read/write/acknowledgment implementations are in **new** `cmd/internal/launcher/checkpoint_io.go`. Original modified classifications confirmed; final implementation split recorded. |

BR-1 is addressed by this documentation correction; no orientation protocol or
production behavior changed for the finding. Verification was the complete
row-by-row `git diff --name-status 7800e968..f5fa755b` audit plus pinned-symbol
inspection. The separate BR-2 focus-ordering finding remains with the Console
implementation and its regression test.

### 2026-09-14 — BR-2 focus-ordering regression evidence

Added `TestContinuationCompletionPreservesInterveningFocus` across four event
orders: select another actor after source exit, select then reopen the panel,
select during attach dispatch, and select before source exit. The existing
atomic installer only changes focus when active is empty; all cases pass with
unchanged production code, including under race detection. No focus-generation
mechanism is added without a reproduced need. The second closing review is asked
to withdraw the stated finding or supply a counterexample.
