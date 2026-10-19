---
collaboration-mode: discussion
---

# Lifecycle failure persistence implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Use superpowers-executing-plans for the session-warm dispatcher integration; bounded codec work may be delegated. The SDLC binary owns the formal reviews.

**Goal:** retain the reason and exact attempted target of lifecycle failures from CLI, switcher and reattach, including failures before a new launch, and show the latest unresolved failure after transient receipts disappear.

**Architecture:** extend the existing launch diagnostic ring, preserving legacy records and unknown JSON members. One recorder at `DispatchOperation` owns failure publication and successful supersession. Queue-owned attempts finish there only after preparation, execution, adoption and any readiness check have produced their terminal outcome.

**Tech stack:** Go; existing Couch dispatcher, stateful failure-store fake, namespace-backed atomic JSON store, console queue and receipt reducer.

**Approval:** proposed; hub approval and pair:1 contract check precede change-code. This is one atomic close boundary, with plain tasks rather than milestone tags.

## 1. Scope and retained evidence

The hub assigned only failure persistence from the original #683 incident. #695 owns folding completed parks and proven teardown; #684 owns convergence and Fresh ordering. The initiating 17:49–17:51 error remains unknown. Do not reconstruct it from the later 18:15 launch failure or treat an archived open park as verified finalization. Preserve the old success-completion/owner-death/session-absence fixture shape for #695, and never operate on the operator's session 1-188.

Reference contract: #695 plan §§2a/P3 on origin/000695-lifecycle-primitives-transcript-binding-on-the-record-the-two-kind-address-proven-teardown. Pair:1's verified delivery fc8d4000 adds optional superseded_at, generic empty Phase, and an errors.As capability defined here.

Covered: admitted lifecycle actions, their preparation/guard/effect/adoption/readiness errors, cancellation after admission, and returned park errors. Not covered: unauthenticated requests, argv/schema errors with no verified target, process death before any terminal outcome can be observed, or root-cause repair. Those cannot be reported as durable observed failures. Diagnostics never authorize a recovery action.

## 2. Core concepts

Paths in the tables are relative to `cmd/internal/` unless otherwise stated.

| Name | Lives in | Status |
|---|---|---|
| `LifecycleFailure`, `LifecycleFailureHistory` (legacy LaunchFailure aliases) | `couchcore/launchfailure.go` | modified |
| `ProcessIdentityLine`, `LifecycleFailureDetails` | `couchcore/lifecyclefailure.go` | new |
| `LifecycleFailureTarget`, centralized slot-key derivation | `couchcore/lifecyclefailure.go` | new |
| `OperationAttempt`, terminal completion envelope | `couchcore/operationattempt.go` | new |
| preserving failure codec | `couchcore/lifecyclefailure_codec.go` | new |
| `ApplyLifecycleFailureEvent`, latest unresolved projection | `couchcore/lifecyclefailure.go` | new |
| `OperationReceipt`, `ReceiptOutcome`, `ApplyReceiptEvent` | `couchmessage/operation.go` | modified |
| `ActionableThreadSummary`, `ShowResult` | `couchcore/actionableinventory.go`, `couchcore/slotreport.go` | modified |

The history owns up to 64 failures; a slot has many retained failures but at most one latest visible failure. An attempt owns one immutable nonce and target, with nested operations borrowing that attempt. Identity/merge/projection/codec tests use literal values and bytes, with no filesystem or process mocks. ProcessIdentityLine owns an exact ProcessIdentity plus a bounded command line; it is evidence only, never a signal target. #695 implements LifecycleFailureDetails on its teardown errors rather than adding capture sites.

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| `LifecycleFailureStorage` / real store, legacy aliases | `couchcore/launchfailure_store.go` | modified | hardened JSON read, flock and atomic replacement |
| stateful failure-store fake | `couchcore/launchfailure_store_fake.go` | modified | reopen, enrichment, supersession, injected publication failures |
| `LifecycleFailureRecorder` | `couchcore/lifecyclefailure_record.go` | new | clock, nonce generation and bounded storage publication |
| dispatcher recorder capability | `couchcore/operationdispatch.go`, `couchcmd/run.go` | modified | operation execution and final typed error |
| queue terminal-outcome adapters | `couchtty/console_remote.go`, `couchtty/console_operation_adoption.go`, `couchtty/console.go`, `couchcmd/console_slot_operations.go`, `couchcmd/live_restart_probe.go` | modified | preparation, guard, adoption, readiness and cancellation completion |
| cached diagnostic projection / views | `couchcore/lifecyclefailure_projection.go`, `couchcore/launchfailure_show.go`, `couchtty/menu_render.go`, `couchcmd/launchfailure.go` | new / modified | metadata-checked store snapshot, switcher and show rendering |

Reuse the existing storage fake and portable namespace folder. No new native service or process observer is needed. Register every new production source in artifactpath's exhaustive classifications before the full suite; keep actual concept locations synchronized with the diff.

## 3. Durable format and compatibility

Keep `<namespace>/launch-diagnostics/failures.json`, its top-level array, lock, private staging files and atomic replacement. Do not change the global strictjson decoder or bump the format version. Older pre-tolerance binaries still refuse new members; rollout uses the normal Couch rebuild/restart.

Widen LaunchFailure into LifecycleFailure with aliases for existing capture callers. Preserve existing launch fields and meanings. Add optional `phase`, `holders`, `outcome`, slot identity/display metadata and `superseded_at`. Phase is empty for unclassified errors or one of observe/preserve/stop/sweep/prove/bind/launch/commit. Outcome defaults to failed for legacy or generic rows; new classified rows use failed/unproven/refused. A slot failure may omit the thread address/tag; it must carry a verified canonical slot path. A legacy tagged row remains valid without new fields.

Use `LifecycleFailureTarget` to freeze canonical physical StartingPath, optional canonical slot label, observed ThreadAddress and park nonce before execution. Centralize the slot key in one function over that canonical slot identity; do not key by tag or a mutable alias. Cover primary :0 and numbered slots. #695 replaces this function with LifecycleAddress.Key(), retaining the slot arm's mapping and existing file. Exact thread-only legacy records remain searchable by address. Never resolve a replacement target at publication time.

Expose this typed capability through errors.As:

```go
type LifecycleFailureDetails interface {
    error
    FailurePhase() string
    FailureHolders() []ProcessIdentityLine
    FailureOutcome() string
}
```

Empty Phase is deliberately not inferred as observe. Generic errors retain the operation name and failed outcome. Known typed refusal/error chains remain discoverable through Unwrap. Capture the original operation message before adding the diagnostic-save message, avoiding recursive amplification.

The local codec first rejects duplicate keys recursively, then decodes known fields with explicit type/presence/null rules. Required legacy identity/time fields remain required; helper_started remains optional on read for backward compatibility and is emitted by the writer. Optional exit pointers may be null/absent; known scalar/object members otherwise reject null. New fields are optional for legacy rows, with enum/identity/bounds checks when present. Do not impose new fields on old rows.

Capture `Unknown map[string]json.RawMessage` per entry and diagnostics-private opaque maps for nested address, slot and holder/process objects. Keep opaque members out of shared identity types. Deep-copy maps/bytes through clone, normalization, fake reopen, enrichment and projection. Same-attempt enrichment preserves old unknown members absent from the update; a supplied newer opaque value replaces only that same member. Known owned fields cannot be resurrected from unknown maps through omitempty.

**Byte preservation:** do not send opaque values through a container json.Marshal that compacts RawMessage. A deterministic object/array encoder marshals owned field names/values and appends validated unknown RawMessage values verbatim. Tests compare each unknown raw value, including whitespace, escaped strings and precise large integers, after real rewrite/reopen. Preserve current structural numeric validation unless separately approved; no arbitrary float conversion of opaque values.

Keep current 1 MiB encoded ceiling, 16 KiB raw launch-output tail, 4 KiB per error and existing identity/metadata bounds. Raise count 16→64 with oldest-first eviction. Proposed holder bound: 32, each command line at most 512 UTF-8 bytes, identities at existing identity limits; trim display command text, never fabricate an identity. Normalize a supplied oversized holder list to its bounded evidence tail with an explicit truncation count. Unknown fields are never truncated to fit: an oversized encoded update refuses atomically, preserving prior bytes. The count cap is not a promise that 64 maximum-sized entries fit the byte cap.

## 4. One recorder and the full terminal boundary

Add an optional recorder capability to OperationExecutors. Wire production constructors once from Couch, both direct CLI and the singleton console dispatcher; individual lifecycle implementations do not save diagnostic records. Operation declarations select lifecycle mutations using their existing effect/owner metadata plus one central lifecycle predicate; read-only show/list/peek/status and schema/authentication probes neither clear nor manufacture failures. Add an exhaustive declaration coverage test so new lifecycle verbs cannot silently bypass the recorder.

**Necessary refinement to P3:** DispatchOperation is the effect entry, but today it does not encompass every terminal outcome. PrepareSlotOperation/guard errors can return before it; console adoption and asynchronous readiness can fail afterward; attached-pane reuse can skip it. Merely wrapping its current return would miss the incident class and clear failures too soon.

For queue-owned actions, create OperationAttempt before preparation from the already verified menu row or canonical socket target/cache. Carry it through the queue using typed owner-local context/payload. Dispatch performs the effect while recording is deferred. The shared terminal callback then re-enters DispatchOperation with a typed completed-outcome envelope; that branch validates attempt ownership, records the final value/error, and performs **no executor, queue, signal or launch**. This is not a wire argument or arbitrary alternate executor. A closed attempt cannot execute or publish twice. Nested switch/resume/attach calls inherit the attempt and cannot supersede independently.

Attach the terminal adapter at shared queue completion paths, including pre-prepare cancellation, hub guard refusal, hosted resume reuse and adoption failure; remote readiness retains terminal ownership until its existing asynchronous check ends. Do not move readiness onto the lifecycle worker or add another goroutine. Preserve its bounded wait and close/cancellation behavior. A completed dispatch alone is not terminal success when adoption/readiness is owed.

For synchronous calls, the same DispatchOperation wrapper owns terminal recording directly. Reuse an existing captured launch failure nonce and enrich that row rather than duplicating launch-stage evidence; retain its output/exit observations. Otherwise allocate one operation nonce and retain the separate park transaction nonce. Diagnostic-save failure wraps the original error with explicit save status and inspect advice, never changes failed/unproven to success. A lifecycle effect that succeeded but whose supersession write fails remains successful with a visible diagnostic-store warning, not an invitation to repeat the effect.

ReceiptOutcome and OperationReceipt gain optional FailureNonce, copied by ApplyReceiptEvent. Extract it once after typed outcome classification so all refusal/error branches preserve it. The operation error keeps its original errors.Is/As behavior and reports nonce/store path. Show includes the nonce in full output; the operator can correlate it after the five-minute receipt expires, until bounded ring eviction. Keep the existing ShowResult LaunchFailures JSON field and launch-stage fields compatible through the type aliases; no parallel duplicate history array is introduced.

Park trigger, completion-observation and finalization errors already propagate; retain that chain with a regression. Also inspect ParkResult.CleanupError, which can currently accompany nil error: record it as unproven preservation evidence without converting a completed external effect into a safely retryable failure. Define this warning projection explicitly in the recorder's outcome adapter; no park transition or finalization authority changes here.

## 5. State, ordering and operator actions

Actions: inspect `couch --show SLOT`; refresh after a diagnostic read error; take the existing row's offered recovery action after inspection. This feature adds no automatic retry or recovery verb.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| no retained failure | existing row status | existing offered action | none added |
| admitted attempt preparing/executing/adopting/waiting | existing running status; previous failure retained | wait for terminal outcome, then inspect | process death may lose unobserved error text |
| terminal failed/unproven/refused, save confirmed | bounded operation/phase and message, nonce in show/receipt | inspect show, then existing offered action | evidence eventually evicted at ring bound |
| terminal failure, save refused/uncertain | original failure plus diagnostics-save warning | inspect show and refresh; do not infer effect rollback | this message may not survive restart |
| later terminal success, supersession confirmed | existing successful state; older failures hidden | existing offered action | evidence retained until eviction |
| success but supersession save failed | success plus store warning; older failure may remain | inspect show; no repeat of successful effect merely to clear text | stale visible diagnostic |
| malformed/unreadable store | diagnostic-unavailable annotation, lifecycle state unchanged | refresh/show after correcting the reported store problem | unknown evidence; no overwrite |

Pure `ApplyLifecycleFailureEvent(history,event)` owns failure enrichment and supersession. Events are RecordFailure and SupersedeSlot through admitted publication. Unrelated-slot success changes nothing. Supersede only existing earlier failures for the exact bound slot, retaining unknown fields and immutable original failure time. Late enrichment cannot remove superseded_at or make an old failure visible. New later failure becomes visible. No success rows or success side file.

Before a success write, consult the metadata-validated cached snapshot: if no unsuperseded failure exists for that slot, return without a store write. If present, take the existing lock, reread uncached, and apply supersession atomically. The queue's existing per-slot ordering defines which attempt is later; carry attempt start/completion facts so late callbacks cannot clear failures from newer attempts. Test reverse completion explicitly rather than relying on wall-clock resolution. No new lease is warranted: this is evidence, and the existing store flock plus exact attempt identity serialize the bounded rewrite; it never proves process absence.

## 6. Projection, costs and lifetimes

One cached snapshot per store, guarded against concurrent refresh. Use mtime plus file identity/size so atomic replacement or same-time writes invalidate it; preserve no-follow/regular-file validation. Save rereads under its lock and bypasses the display cache. Do not reuse cached absence after a metadata change or turn corrupt data into empty history.

Each inventory refresh does at most one bounded store payload read, not one per row; an unchanged snapshot needs metadata validation only. Build a slot→latest-unresolved index once, then annotate rows in one pass. No Git, process table, zellij or native transcript reads are added. Diagnostic read errors are visible annotations and never change ThreadState, action guards or recovery authority. Show uses the same projection, showing only unsuperseded failures, including failures for an empty slot. Render operation when Phase is empty. Sanitize and clip row text while preserving the slot/selection width floor; full bounded messages remain in show.

Growth drivers: H≤64 retained failures, R inventory rows, W writers and B≤1 MiB encoded bytes. Read/encode is O(B), merge is O(H) plus existing bounded sorting, projection O(H+R), one lock publication per terminal failure or necessary supersession. Count reads/index visits at R=8/16 and H=8/16; reject per-row store reads or quadratic matching. Cache dies with Couch/store instance; retained failures end at 64-entry eviction. Private stages remain cleaned under the existing lock. Receipt lifetime stays five minutes. No network/CPU-throughput workload, worker pool, recurring timer or additional daemon is introduced.

Use the current save admission timeout from saveLaunchFailure and context-aware flock waiting. Cleanup/effect execution is never postponed for diagnostic IO. Cancellation permits a bounded independent diagnostic-save attempt after the terminal error, followed by no background writer. Once publication is admitted, finish the existing atomic write even if cancellation arrives; preserve uncertain publication reporting. The 1 MiB ceiling is enforced by writer and reader.

## 7. Function-level test strategy and implementation sequence

Each task uses red → minimal implementation → green → commit. One close review covers the atomic feature.

| Risky function / seam | Adversarial strategy | Mechanical guard |
|---|---|---|
| preserving decoder/encoder, normalize/clone | generate additive entry/nested values, missing/wrong/null known fields, escaped-equivalent duplicate keys, malformed tails | real Read→Save unrelated/enrich→reopen raw-byte equality; invalid writes preserve old bytes |
| ApplyLifecycleFailureEvent / key | sequence duplicate, unrelated slot, replacement tag, same-time attempts, reversed completion, eviction | literal pure transition tests; superseded evidence cannot reappear |
| DispatchOperation recorder / attempt ownership | prepare/guard/effect/adoption/readiness/cancel failures and nested calls | production-wired stateful fixture; exactly one nonce/publication, no executor on completion replay |
| slotOperationOutcome / ApplyReceiptEvent | every typed refusal and generic/partial outcome | table over actual classifications, FailureNonce retained after finish and visible in JSON |
| cached projection / inventory | concurrent refresh, same-time replacement, corrupt/absent file, 8/16 rows and histories | counted payload reads ≤1 per refresh, unchanged metadata zero payload rereads, O(H+R) visits |
| row/show rendering | unknown phase, oversized/control text, empty slot, store error, superseded record | production renderer assertions preserve selected slot identity and original state/actions |

### Task 1: Compatibility and pure history

- [ ] Add failing `TestLifecycleFailurePreservesUnknownFieldsThroughRewrite` in `couchcore/lifecyclefailure_codec_test.go`, including nested objects, whitespace and large integer bytes; exercise unrelated insertion and matching enrichment through the real store. Add strict wrong-type/duplicate/required/null cases.
- [ ] Run `go test ./cmd/internal/couchcore -run 'LifecycleFailure|LaunchFailure' -count=1`; retain the red result before implementation.
- [ ] Implement the additive codec, aliases, target/key and pure transition; extend the existing fake. Cover 64/65 retention, byte-overflow refusal, deep-copy isolation, legacy launch enrichment, slot-only records and late superseded enrichment.
- [ ] Re-run the targeted command to green and commit `#683: preserve and widen lifecycle diagnostics`.

### Task 2: Terminal dispatch and receipts

- [ ] Add production-wiring integration tests in `couchcore/lifecyclefailure_record_test.go`, `couchcmd/lifecyclefailure_service_test.go` and `couchtty/lifecyclefailure_completion_test.go`. The stateful fixture begins with an older failure and injects barriers before prepare/effect/adoption/readiness completion; assert that it remains unresolved until genuine terminal success.
- [ ] Exercise a CLI request over the existing private message-service transport and a menu/reattach action through the actual console dispatcher. Fail each named seam independently; assert one durable record, original target/park nonce, typed phase/holders only when supplied, and matching receipt nonce.
- [ ] Add pure attempt replay/ownership and receipt mapping tests; run targeted tests to red.
- [ ] Wire the central recorder and deferred terminal completion, preserving existing queue serialization, hub gates, readiness lifetime and effects. Audit all DispatchOperation production constructors and queue early returns, including shutdown abandonment. No per-operation persistence calls.
- [ ] Cover the returned park error and CleanupError warning shapes without changing park records or recovery decisions. Run focused tests, existing launch/adoption/restart/park regressions and commit.

### Task 3: Cached display and complete verification

- [ ] Add counted snapshot/index tests and row/show rendering regressions to red. Implement the shared cached projection, latest failure annotation and supersession visibility; keep historical raw bytes in the ring.
- [ ] Run `go test ./cmd/internal/couchcore ./cmd/internal/couchmessage ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'LifecycleFailure|LaunchFailure|Receipt|ParkRecovery|Relaunch|Readiness|Adoption|RemoteOperation' -count=1` and appropriate race runs for the new cache/attempt/store tests.
- [ ] Update `atlas/couch.md` and root README's diagnostics entry with path, nonce, bounds, supersession and unavailable-store behavior. Add only review-derived lessons. Run artifactpath/testpair source guards and declaration-comment guards before the suite.
- [ ] Private conformance: use a temporary namespace and real store/service to force a pre-launch failure, stop/reopen service, and show the same nonce/text; then successfully complete the same slot and prove it hides while raw evidence remains. Use no operator slot, no focus, no production state.
- [ ] Ask hub whether operator live acceptance rides the next restart; do not stage :0 without approval. Proposed live probe: disposable slot's confirmed refusal before launch, verify row/show/receipt, then one successful existing action supersedes it. Record any owed live check explicitly.
- [ ] Push the final head, run `sdlc local-check run --name suite` in background, inspect the complete result, then run `sdlc close --issue 683 --verified '<measured evidence>'` with only justified precise exemptions. The binary owns fresh review. On SHIP run `sdlc merge --yes`, let integration checks finish, and report PR/merge plus any owed live acceptance to hub.

## 8. Architecture decisions

- **ARCH-DRY:** existing file/fake/capture; one dispatcher recorder and one terminal adapter, no per-site Save.
- **ARCH-PURE:** merge, key, codec and view projection are pure; storage/clock/queue wrappers are explicit IO seams.
- **ARCH-PURPOSE:** cover pre-dispatch and post-dispatch failures rather than only easy effect returns. Scope split is explicit and must be approved against original Done when.
- **ARCH-MOCK:** reuse the stateful storage/console/service fakes and compare with a portable real namespace; operator environment is separate live acceptance.
- **ARCH-CONSTRAINTS:** 64 rows/1 MiB, bounded messages/holders, no per-row IO, counted N/2N regression, bounded diagnostic admission.
- **ARCH-SECURE:** persisted data is untrusted, duplicates/types strict, safe file mechanics unchanged, frozen verified target; no transcript reads or new secret collection.
- **ARCH-ORDER:** one attempt terminal owner, explicit RecordFailure/SupersedeSlot events, late completion/replay tests, no success before adoption/readiness.
- **ARCH-FUNERAL:** same bounded ring/file, existing stage cleanup, instance cache lifetime, no success log or background writer.
- **ARCH-EVOLVE:** additive schema, legacy aliases/rows, nested opaque byte preservation through every rewrite, no version bump or global strictjson relaxation.

## Revisions

2026-10-19 — Initial plan incorporates pair:1's P3/2a contract and its empty-phase/superseded_at refinement. Read-only audits identified terminal outcomes outside the current dispatcher return; the proposed deferred terminal envelope closes that coverage gap without moving readiness into the lifecycle worker. Hub approval is requested before implementation.
