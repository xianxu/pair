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
| Lifecycle poll cadence | sessionwatch/cadence.go or existing run.go cadence section | New: reset to Poll on new bytes/backlog; idle doubles to5s maximum. No wall-clock test sleeps. |
| Guarded cursor state | sessioninventory/transcript_cursor.go | New shared state extracted from TitleProjection tail: pinned identity, scalar scanner state, offset/frame and4KiB boundary hash. One cursor per authorized transcript, destroyed with watcher/projection. |
| Lifecycle event batch | sessionwatch/transcript_lifecycle.go | Modify: normalize only this poll's validated records with existing offsets, root filtering and durable watermark. Never reconstruct/sort old event history on idle. |

| Integration seam | Home | Change / fake |
|---|---|---|
| Guarded native tail IO | sessioninventory/transcript_cursor.go and title_projection.go | Extract existing identity, boundary and paired-stat checks; sessioninventorytest.FakeRuntime controls append/replacement/partial reads and counts bytes. Keep title owner/Agy/display behavior outside the extraction. |
| Bound lifecycle runner | sessionwatch/run.go plus a focused follow helper | Enter once after binding; retain only authorized root state, never Observe/Select(TargetNewLaunch), catalog reload/publication or foreign reads during follow. Existing watcherRuntime and FakeRuntime supply clock, owner and IO. |
| Current owner read | sessioninventory/query.go / launch_cache.go; watcher adapter | Reuse OwnerLaunchCache's metadata-fenced compact current-owner reducer for bound checks, not a second mtime-only cache. Expose the narrow existing owner read if necessary. Unbound reads still retain launch boundaries needed for correlation. |

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
- **Bounded append:** Append more than256KiB, including a record crossing passes; count four payload chunks maximum per poll and monotone progress. Append again during every ReadAt to prove no EOF chase. Eventually pause and prove the exact final turn completion once, with correct absolute offset and durable watermark.
- **History negative control:** Seed small and2x large prior lifecycle histories; unchanged polls visit/project zero old events and read zero old transcript/ledger bytes. No replay after already-published batches.
- **Guard permutations:** stable token and APFS-style empty token; replace/truncate/same-size mutation at each boundary/read/stat/commit seam; malformed/oversized/partial records; changed launch/root PID and uncertain journal append. No unauthorized event or rediscovery fallback. Use the same shared-reader cases for title consumers.
- **Cadence pure tests:** progress resets; repeated idle doubles/caps; fake-clock Run verifies the sleeps and process-exit stop, with no raised timeout or timing assertion.
- **Consumer sweeps:** sessionwatch package, sessioninventory/title projection/native fixture suites, focused race repetitions, source inventory/comment guards. Full recorded suite on pushed head before close; SHIP lands through sdlc merge with main integration.

## Execution

- [ ] Add counted failing regressions before production edits; preserve read-only live evidence.
- [ ] Extract the existing guarded tail, integrate bound-only follow and cached owner checks, then pure backoff/event-delta handling.
- [ ] Run guard and consumer suites; document atlas/lessons and retained cold-cost/latency boundaries.
- [ ] Obtain hub-coordinated live acceptance on a newly started Codex slot using the fixed build: sample/watch CPU over60s after initial binding/catch-up; near0% idle, no full scans. Do not stop/relaunch original PIDs or touch pair:0 without the hub's authorization.
- [ ] Run recorded full suite, close review and gated landing; report actual merge to hub and stop.

Single atomic review boundary, no Mx tags. Await hub approval of scope/cadence before change-code; estimate comes only after plan-quality clears.


## Revisions

### 2026-10-18 — approved

Hub approved scope and cadence in7f664349-b72a-4c11-a1b2-91baf70f21ba. Cold discovery remains an explicit slot-lifecycle project follow-up, owned by the TL's project record. The hub schedules the live CPU check on a fresh Codex slot when the build is ready. Proceed through change-code; no new executor, binding rule, timeout increase or corpus fallback.
