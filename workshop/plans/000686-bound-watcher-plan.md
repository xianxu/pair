---
collaboration-mode: discussion
---

# Bound session watcher work

**Goal:** Stop fresh Codex watchers from continuously re-inventorying and validating the native sessions tree after their conversation is already bound. Idle follow work must be independent of unrelated transcripts and retained transcript/event history.

**Evidence:** Both dispatched live PIDs already have ordinal2 correlation bindings. Three-second samples show Run -> incrementalWatcherInventorySnapshot -> ValidateTargetWork -> ValidateCodexDelta/JSON decode. The native tree has3,549 JSONL files totalling6,467,935,788 bytes. RunCLI enables lifecycle follow; the bound branch currently walks and reconciles the store and republishes catalog on every100ms pass. Samples/CPU and metadata census are recorded in the issue Log. The original unbound-wait hypothesis is disproved for these processes.

## Alternatives and scope

1. **Only increase Poll:** rejected; each wake still scales with the entire corpus and replays history.
2. **Only follow named artifacts with AdvanceTargetValidation:** removes discovery, but that API chases a moving EOF, retains/copies old events and rejects append advancement without a generation token. Budget-error wrapping would discard progress and restart prefixes.
3. **Recommended: reuse the guarded title transcript tail as a shared, budgeted cursor and separate bound follow from discovery.** Preserve existing scanner and owner authority, consume only new records and back off idle polling. No new executor, filesystem watcher or persistent cache.

The scope is the measured bound-follow loop, including unchanged ledger reads, native reads and old-event work. Initial unbound correlation/discovery remains the existing authority mechanism: it can cost O(N) metadata and candidate validation, and its existing send/output gating stays. A restarted proofless watcher can require one cold discovery/owned validation before following; this is not claimed O(1). The plan does not redesign all unbound inventory discovery (#157). The hub must explicitly accept this boundary before code, because the issue's initial wording describes an unbound wait while the measured defect is bound follow.

## Core concepts

| Pure entity | Home | Change / contract |
|---|---|---|
| Bound follow state / checkpoint | sessionwatch/bound_state.go: boundFollowState, boundCheckpoint, reduceBoundFollow | New: pure Reading/Publishing/Stopped transitions own compact scanner state and committed cursor; only whole acknowledged batches commit candidates. |
| Lifecycle poll cadence | sessionwatch/bound_state.go: nextLifecycleDelay | New pure function: progress/backlog resets to Poll; idle doubles to5s maximum. |
| Guarded cursor state | sessioninventory/transcript_cursor.go: TranscriptCursor, TranscriptBatch | New value state: pinned metadata, offset/frame and4KiB boundary hash, with no scanner state or history. IO methods are a separate integration seam below. |
| Lifecycle event batch | sessionwatch/transcript_lifecycle.go: lifecyclePublicationsFromEvents, LifecycleRecordsFromEvents | Pure delta projection: normalize this pass's validated records with existing offsets/root/watermark. No old-event history retained. |

| Integration seam | Home | Change / fake |
|---|---|---|
| Guarded native tail IO | sessioninventory/transcript_cursor.go: NewTranscriptCursor, TranscriptCursor.Next; title_projection.go: TitleProjection.tail/PollContext | Shared identity/boundary/paired-stat guards; stateful FakeRuntime drives mutation and byte counts. Title owner/Agy/display behavior remains outside the extraction. |
| Bound lifecycle runner | sessionwatch/bound_follow.go: followBoundCodex; run.go: Run | Enter once after binding; retain only authorized root state, no Observe/Select, catalog or foreign reads during follow. Stateful watcherRuntime owns scheduling/IO. |
| Current owner read | sessioninventory/query.go: ReadCurrentLaunchAt, readOwnerLaunchFromLedger; launch_cache.go: OwnerLaunchCache | Existing metadata-fenced compact strict reducer reused through exact named ledger stats; stateful runtime counts read/decode work and invalidation. Unbound full launch reads unchanged. |
| Publication under current confirmation | sessionledger/store.go: WithCurrentConfirmationReader; sessionwatch/transcript_lifecycle.go: publishLifecycleBatch | Existing ledger lock holds metadata-fenced owner reread and exact confirmation before effects; existing journal append/reconciliation reports typed acknowledgments to the pure reducer. Stateful actual-store and journal controls cover uncertainty and replacement. |

All new ordinary Go sources join artifactpath's exhaustive inventory. Pure cursor state has no filesystem authority on its own; IO guards and existing validators supply evidence (ARCH-PURE/SECURE). The shared tail avoids duplicating APFS-specific guards (ARCH-DRY). No binding/admission policy changes.

## Guarded follow contract

- Seed from an existing complete TargetValidation after its identity and boundary are fenced. Fresh correlation already has it; resumed/restarted binding uses existing exact-ID/proof validation once. Drop foreign tracked validations immediately on transition. Preserve already-observed lifecycle events/watermark exactly once at this handoff.
- The shared tail retains TitleProjection's same-file identity rules, including APFS's empty generation token with stable file ID and birthtime, and its4KiB boundary hash plus paired before/after metadata checks. It never upgrades display telemetry into binding or resume authority.
- Freeze each pass's endpoint at min(size observed at entry, raw offset +256KiB). At most four64KiB payload reads per follow pass; growth after that snapshot waits for the next poll. Retain parser offset and partial record across passes; replacement/truncation/same-size mutation fails closed, never falls back to a store walk.
- Reuse geometric partial-record framing and the existing32MiB record ceiling. A completed large record can require decoding up to that ceiling in one pass; the256KiB read budget is not falsely described as a strict256KiB CPU/decode ceiling. Oversized/malformed records surface existing unavailable diagnostics, with no fabricated lifecycle event.
- Keep only compact scanner state, one partial record and boundary window. Return only newly validated event records. Preserve exact turn ID, transcript offset, launch ordinal, generation and existing journal deduplication/watermark semantics. Never retain full prior Events/Results.Records for follow.
- Check root process identity and exact current launch ordinal/binding around each poll before committing events/cursor. Reuse cached owner evidence only while the existing full metadata fingerprint matches; replacement, conflicting/new launch or changed binding stops this watcher. Unchanged ledger bytes decoded/read per follow tick =0.
- Unchanged transcript: one named metadata observation, no body/boundary reads, decoding, catalog IO, event projection or journal append. Idle cadence backs off from existing Poll(default100ms) to5s; actual progress/backlog resets to Poll. Thus an idle-to-new-event delay can be up to5s, plus bounded catch-up for a large append. The hub must approve that latency tradeoff.
- TitleProjection consumes the same extracted guards with its existing snapshot extent; do not refactor its owner admission, Agy/database rules, output reducers or UX. Run the complete title/native fixture suite because the guard is shared.
- Native file changes, uncertain reads and process/owner loss never cause a fallback to corpus discovery after binding. Retain durable binding/journal; stop telemetry with the existing diagnostic and require the existing relaunch path. No timer grants authority.

## State projection

Actions: continue working, relaunch the affected slot through the normal declared action.

| Internal state | Visible state | One recovery action | Possible loss |
|---|---|---|---|
| Unbound/discovering | Existing native compatibility / waiting evidence | Continue working to complete a native round | Existing unbound behavior unchanged |
| Bound, caught up and idle | Normal slot; no new notice | Continue working | Up to5s observation latency |
| Bound, catch-up bytes pending | Normal slot; lifecycle notice waits for validated batch | Continue working | Delay proportional to pending bytes; no invented completion |
| Transcript/owner invalid or unreadable | Existing unavailable diagnostic; no fabricated lifecycle result | Relaunch through the normal slot action | Telemetry gap; native transcript and durable binding unchanged |
| Root process exits or launch changes | Watcher exits | Normal declared slot relaunch if needed | No new data deletion |

No UI state/menu/policy expansion. The cursor/cache are in-memory, bounded and die with the watcher; existing durable journal retention remains unchanged (ARCH-FUNERAL).

## Counted work and tests

Growth drivers: unrelated tree files N, owned transcript suffix B, one partial record R<=32MiB, new lifecycle events E, owner ledger L, cold admitted history H. Separate cold O(N+H+L) admission from steady follow. Steady follow: zero listings/foreign reads/catalog IO, fixed named metadata/owner checks, payload<=256KiB plus fixed boundary windows, new decode bounded by new complete records (including R), event work O(E), zero historical-event visits. Existing durable event append/dedup remains the existing writer contract, performed only for new events; no idle append.

- **Red root-cause regression:** Run a fresh correlation through binding, then K idle passes with N=4,000 and8,000 unrelated metadata-only files. Count post-binding ListFiles, foreign ReadAt/ReadFile, catalog reads/writes, ledger bytes read/decoded, own stat/body/decode work and lifecycle projections. Expected: zero tree/catalog/foreign/history IO, constant own/owner metadata work independent of N. Keep cold discovery counters separately visible. Existing code must fail.
- **Consumer sweeps:** sessionwatch package, sessioninventory/title projection/native fixture suites, focused race repetitions, source inventory/comment guards. Full recorded suite on pushed head before close; SHIP lands through sdlc merge with main integration.

## Execution

- [x] Add counted failing regressions before production edits; preserve read-only live evidence.
- [x] Extract the existing guarded tail, integrate bound-only follow and cached owner checks, then pure backoff/event-delta handling.
- [x] Run guard and consumer suites; document atlas/lessons and retained cold-cost/latency boundaries.
- [ ] Obtain hub-coordinated live acceptance on a newly started Codex slot using the fixed build: sample/watch CPU over60s after initial binding/catch-up; near0% idle, no full scans. Do not stop/relaunch original PIDs or touch pair:0 without the hub's authorization.
- [ ] Run recorded full suite, close review and gated landing; report actual merge to hub and stop.

Single atomic review boundary, no Mx tags. Await hub approval of scope/cadence before change-code; estimate comes only after plan-quality clears.


## Revisions

### 2026-10-18 — approved

Hub approved scope and cadence in7f664349-b72a-4c11-a1b2-91baf70f21ba. Cold discovery remains an explicit slot-lifecycle project follow-up, owned by the TL's project record. The hub schedules the live CPU check on a fresh Codex slot when the build is ready. Proceed through change-code; no new executor, binding rule, timeout increase or corpus fallback.


### 2026-10-18 — concrete shared seam before implementation

Use transactional cursor advancement: NewTranscriptCursor(runtime, validated TargetValidation) seeds only pinned byte/frame/boundary state; cursor.Next(ctx,runtime,maxBytes) returns a candidate next cursor and TranscriptBatch {Entry, Records, BytesRead, Backlog}. The caller owns compact scanner state, validates the returned records with the existing scanner and rechecks owner/process before committing candidate cursor/state or publishing lifecycle facts. A rejected scanner/owner observation leaves the prior cursor intact. TitleProjection delegates to the same internal advance helper with its existing checkArtifacts callback, preserving Agy/other-artifact guards and its outer PollContext transaction. It uses its existing initial snapshot extent, while lifecycle passes256KiB. No scanner/events/history in the cursor.

Cached owner checking must also avoid a Pair-root listing: factor the existing metadata-fenced owner-cache core so the bound watcher can supply its exact validated ledger artifact; ordinary inventory lookup still finds the ledger through its existing route. Do not add a new cache or mtime-only authority. Count both Pair-root and native-root listings in the zero-follow regression. Runtime fakes must present the same authoritative Pair ledger to this adapter, rather than weakening the production guard for tests.


### 2026-10-18 — PQ-1/PQ-2: enforced transition and function strategies

The authoritative bound-follow production component is pure reduceBoundFollow(state,event), with phases Reading, Publishing and Stopped. It owns the committed cursor/scanner snapshot, acknowledged watermark, at most one prepared batch/candidate cursor and its next publication index. These are internal phases, not extra operator UI states. The IO loop only executes the reducer's next read/publish/stop effect; it cannot independently advance cursor or watermark (ARCH-ORDER).

| Phase / event | Transition and commit |
|---|---|
| Reading + valid batch, owner/process fences passed | Stage candidate cursor/scanner and the bounded new-record batch; enter Publishing. No cursor or watermark advance yet. |
| Reading + valid batch with no lifecycle output | Commit candidate cursor/scanner after the owner/process fence; remain Reading, update cadence from bytes/backlog. |
| Publishing + exact acknowledged append at expected index | Advance acknowledged watermark/index only. Existing journal reconciliation/dedup owns append certainty. |
| Publishing + final acknowledged append | Commit candidate cursor/scanner; discard staged records, return Reading. |
| Publishing + confirmed failure or unconfirmed outcome | Enter Stopped, retain the acknowledged prefix watermark and original committed cursor; do not claim the failed/unconfirmed record committed and do not retry automatically. Already durable journal records remain. |
| Any active phase + changed process/launch/binding, invalid transcript or canceled observation | Discard uncommitted cursor/state/batch, enter Stopped; preserve acknowledged journal progress. |
| Stopped + any late/duplicate result | No effects. Exact mismatched/stale publication identities never advance the watermark. |

An already committed prefix is never rolled back because a later append fails. A failed append which existing journal reconciliation proves committed is an acknowledgment; an uncertain result stops the watcher. Relaunch uses the existing bootstrap/watermark policy, not a persisted retry queue or history replay: a telemetry gap is possible, native work and durable records are preserved. Use the existing current-confirmation fence for the accepted generation around publication, so a changed owner cannot authorize another batch; this reuses the ledger's existing guard, not a new lock. The IO publisher reports one typed result per expected record; pure reduction decides advancement. Add controlled success-prefix/failure/uncertain/reconciled-commit traces through this actual reducer, not a parallel model used only by tests.

This table replaces the procedural case inventory and is the authoritative named-function strategy required by PQ-2; functions touching IO are explicitly tested with stateful runtimes, not called pure:

| Production function | Adversarial input / mechanical guard |
|---|---|
| reduceBoundFollow (pure) | Generated phase/event traces, permuted/stale acks, acknowledged prefix then error/uncertainty and cancellation; only expected acknowledgments advance watermark, only complete acknowledged batches commit the cursor, Stopped emits no effects. |
| nextLifecycleDelay (pure) | Boundary/property tests over base/previous duration and progress/backlog booleans; monotone idle doubling capped at5s, progress resets, no negative/overflow delay. |
| LifecycleRecordsFromEvents (pure existing projection) | Bounded fresh batches with old/foreign/unkeyed/malformed events and reordered positions; exact offset/turn filtering, watermark monotonicity and no dependence on retained old history. |
| NewTranscriptCursor and TranscriptCursor.Next (IO integration) | FakeRuntime append/replace/truncate/mutate/cancel at every guard; maximum four64KiB payload chunks, fixed snapshot under continuous append, partial framing and retry from discarded candidate, no foreign read or unsafe commit. |
| Exact cached owner reader / factored readOwnerLaunchFromLedger (IO integration) | Large ledger cold/warm/mutated/replaced/conflicting inputs plus mid-read drift; unchanged bytes read/decoded=0, exact artifact stats without listing, strict latest-launch authority unchanged. |
| publishLifecycleBatch (IO integration) | Stateful existing journal writer/reconciler acknowledges a prefix then fails/returns uncertain/proves committed; reducer receives exact results once, no automatic retry, retained durable prefix and no duplicate record after reconciliation. |
| followBoundCodex / Run (IO integration) | Fresh correlation into bound follow with4k/8k unrelated trees, explicit fake sleeps and post-binding counts; zero native/Pair listings, foreign/catalog reads and history decode; exact suffix progress and launch/PID shutdown. |
| TitleProjection.tail / PollContext (IO integration) | Complete recorded/native fixtures and existing title/Agy mutation/cancellation permutations through the shared reader; preserve current admission, database and display behavior. |

These refinements specify the already approved owner/cursor commit guarantees and failure behavior. They add no new user action, retry, data format or scope beyond the approved bound follow. Re-run the full plan gate before implementation.

### 2026-10-18 — PQ-2 duplicate inventory removed

Round2 accepted the publication transition contract, but retained PQ-2 because the old procedural test list duplicated the named-function strategies. Removed that duplicate inventory; the counted Run regression, consumer sweep and live acceptance remain. Hub approved the publication contract in ae598ecd-9080-4d36-a3a3-6c9736304c0f. No behavior or scope change.

### 2026-10-18 — active publication owner guard

Narrow review found that existing WithCurrentConfirmation rereads/parses the full Pair ledger under lock for every nonempty lifecycle batch, outside NativeRuntime counters. Hub chose option1 in cd83e66c-02ca-4a93-b3a0-f82ce8d28254: reuse the exact OwnerLaunchCache reader under the same ledger lock, with the same strict ordinal/conflict/root confirmation check. Extract a reader-bearing variant of the existing guard; legacy callers retain the ordinary reader. Only bound lifecycle supplies its cached exact-ledger reader. Full fingerprint identity/size/time mismatch triggers the existing paired full reread before effects, never stale confirmation. Add active unchanged-byte counters including the actual guard path, and append/replacement stale-owner regressions. No new lock/cache/admission policy. This closes the per-event-tick cost rather than narrowing the acceptance to idle only (ARCH-DRY/SECURE/ORDER).

### 2026-10-18 — live acceptance moved after landing

Hub6593f812-a3ff-468a-ad5e-63aa04669518 replaces the pre-landing fresh-slot smoke with a post-landing check of the actual two offenders. No staging; land on recorded suite and SHIP close review. The hub owns rebuild/relaunch and must observe near0% idle and return to idle after a short round. The live execution row remains an owed acceptance, not a worker landing blocker; no live result claimed.

### 2026-10-18 — BR-1: delivered concept ownership map

Close round1 found no runtime defect and passed all four affected package suites, but the initial Core concepts rows were stale after refinement. Corrected the current map to actual declarations and explicit PURE/INTEGRATION boundaries. Prior cadence wording was `sessionwatch/cadence.go or existing run.go cadence section`; delivered cadence is nextLifecycleDelay in bound_state.go. Prior cursor wording included `scalar scanner state`; delivered TranscriptCursor excludes scanner state, which belongs to boundCheckpoint and is committed by reduceBoundFollow. The reader methods are IO integration, separate from the cursor's pure value state. The final map also names the exact owner reader, publication lock, delta projector and bound runner. This is documentation of the already approved/delivered design, with no code or mechanism change. No wording-presence test added; the recorded suite continues on unchanged code.
