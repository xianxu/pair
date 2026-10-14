# Couch tick inventory reuse implementation plan

> **For agentic workers:** Follow AGENTS.md §3; bounded read-only exploration may use subagents. This is one atomic issue with one close boundary.

**Goal:** Remove repeated unchanged JSON work from Couch observations without weakening ownership or scanner validation.

**Architecture:** Long-lived callers reuse validated durable evidence while metadata fingerprints still match. IO measures current inputs; existing parsers and validators own meaning. Live process ownership is never cached by this work.

**Tech stack:** Go, existing file metadata/scanner/catalog APIs, real temporary files and stateful runtime fixtures, Go benchmarks/pprof.

## Core concepts

| Pure entity | Lives in | Change |
|---|---|---|
| Fingerprint reuse decision | sessioninventory existing ArtifactFingerprint/equality helpers; launcher adapter | Reuse exact identity/generation/mutation/size/time tuple |
| Current session-name projection | launcher/session_index.go and couchcore effectiveBindings | Derive latest-per-owner once per file generation; preserve raw decoder semantics |
| Owner launch evidence | sessioninventory/launch_cache.go | Extend existing cached parse for resume fallback; do not duplicate parse ownership |
| Activity observation | sessioninventory/activity.go | Small timestamp/result plus dependency fingerprints; no cached transcript text or whole query tree |

| Integration | Lives in | Wraps |
|---|---|---|
| Session-name evidence cache | launcher/session_index_cache.go (new) | Strict decoder and regular-file stat/read seam |
| Shared collision cache ownership | couchcore/artifactcollision.go | Long-lived checker constructor, copies share pointers; no package-global cache |
| Cached activity query | sessioninventory/activity_cache.go (new), threadactivity/os.go | Existing QuerySessionContext and ActivityForSession; same validation on misses |
| Exact dependency observation | sessioninventory query/runtime helpers | Owner ledger, optional catalog, and scanner-selected native artifacts |

ARCH-DRY: reuse strict DecodeSessionNameIndex and existing ledger/scanner reducers. ARCH-PURE: pure reuse/projection decisions; filesystem reads and synchronization stay in thin cache owners. ARCH-SECURE: cache hits require current file observations, never a remembered pathname alone.

## Alternatives and scope

A UI throttle lowers frequency but leaves the same unbounded decode and message checks exposed. A whole-catalog cache still clones large scanner states per query and broadens the change. Incremental suffix parsing is useful for growing files, but this issue first removes unchanged-input work; existing native incremental validation stays authoritative on misses. Chosen approach: fingerprint-backed parsed index evidence plus compact validated activity observations. #157 and #508 stay separate, with no dependencies needed to fix this loop.

A metadata-only native-ID activity shortcut was rejected: resume identity can remain established after transcript replacement/deletion, whereas existing activity requires scanner-authorized current-root evidence. Keep that invariant.

## Operating envelope and lifetime

- UI observations and broker checks recur across up to 128 live actors; actual observed CPU 95.3%, ledger scale 15–33 MiB reported by dispatch. Cold parsing cost remains proportional to current source size; warm unchanged observations must not read bodies.
- Deterministic performance budget: a warmed valid owner incurs zero body bytes/decode calls for unchanged index/ledger/catalog/native artifacts. Benchmarks report ns/op, B/op and allocations; no flaky wall-time assertion under host load.
- One caller-owned cache reference survives per-scope runtime creation. Retain at most 128 owner activity entries and 128 file/index entries, evict least-recently-used entries; no persistent files, goroutines or descriptors. Each replacement drops superseded evidence. Cache results contain compact projections, not raw history or duplicate scanner blobs; measure retained bytes in scale fixture. Existing owner cache extension must not retain an entire historical-record array solely for one fallback.
- File IO happens outside map mutexes. Per-key work is single-flight; unrelated scope reads remain independent. Waiters honor context cancellation. Any in-flight task ends with its query; no background refresh loop.
- Entry saturation causes eviction and a normal safe cold read, never weakened admission. Semantic absence/errors may be reused only with a complete stable dependency set; operational/cancellation failures are never authoritative successes.

## State and ordering

Cache model: Absent, Ready(fingerprint,evidence), Loading(previous token). All authoritative changes go through small get/load/commit methods.

| Event | Decision/effect |
|---|---|
| Query, matching freshly observed dependencies | Return immutable small projection; retain current ownership/effect checks |
| Query, changed/missing/unreadable dependency | Invalidate prior success; load or return unknown/error by existing semantics |
| Stable successful load | Publish only if key generation and pre/post fingerprints still agree |
| Change during load | Discard, retry at most once; continued movement returns unknown/error |
| Late older completion | Token/fingerprint mismatch discards it |
| Caller cancellation/read error | No partial cache publication; other callers may complete independently |
| Eviction/owner removal | Drop retained projection; future query is a cold read |

No query result grants action authority: launcher SessionOwnerProbe and Revalidate remain fresh. Do not cache process liveness, admission decisions, attach state, or filesystem failures as successful absence.

## Task 1 — establish regressions before implementation

**Files:** launcher/session_index_cache_test.go (new); sessioninventory/activity_cache_test.go (new); existing sessioninventory/ledger_read_test.go; threadactivity/os_test.go; couchcore artifactcollision tests.

- [ ] Build valid synthetic indexes and 15/33 MiB session evidence using existing provider/runtime fixture builders. Include current launch, previous-established fallback, and filename-confirmed v3 roots.
- [ ] Repeated production seam test counts reads/decoded bytes and records testing benchmark allocations. Show baseline repeated work on current code; save CPU/memory profiles in /tmp, never commit real transcript content.
- [ ] Use equivalent small/large unchanged fixtures and assert no warm body reads; output equality against fresh uncached validation is the oracle.

## Task 2 — strict session-name and ledger evidence reuse

**Files:** launcher/session_index.go, new session_index_cache.go; couchcore/artifactcollision.go; sessioninventory/launch_cache.go, query.go; tests beside each.

- [ ] Add a caller-owned session-name reader with injected stat/read seam; include global legacy and per-scope paths separately. Reuse existing strong platform file metadata, not mtime alone. Keep strict parse failures and legacy-first/scoped-second precedence.
- [ ] Cache latest-per-owner projection once per source generation for collision consumers, so warm effectiveBindings does not traverse historical rows. Keep the raw index API unchanged for unrelated launcher consumers. Characterize contested/cross-scope cases against existing effectiveBindings.
- [ ] Wire the checker's PairSessionName and resolveScopedBindings through the same shared cache. Verify ListedPairSessionContext/message authority on the main version used for implementation. Constructor initializes pointer ownership; zero-value/literal checkers retain a correct uncached fallback for tests.
- [ ] Reuse existing OwnerLaunchCache parsing for QueryResumeTargetContext. Cache only current plus previous-established evidence needed by fallback, derived with the existing reducer; no duplicate root selection rule and no full historical-record retention. Give checker-created inventory runtimes the shared owner cache.
- [ ] All effect-time ownership revalidation remains uncached. Reads/stat errors discard old authority. Test append/replacement/truncation/same-size rewrite, duplicate owners, missing/corrupt files, incomplete frames and per-scope isolation.

## Task 3 — unchanged validated activity reuse

**Files:** sessioninventory/activity.go, new activity_cache.go, query.go narrow evidence exposure; threadactivity/os.go and tests.

- [ ] Introduce an activity-query owner shared by threadactivity runtime across probes, keyed by canonical Pair root, scope, tag, agent and parser/schema contract. Return SessionActivity and presence; retain only that compact projection and dependency stamps.
- [ ] On miss run the current QuerySessionContext plus ActivityForSession. Expose the validation's observed native artifact fingerprints to this internal query path rather than reconstructing authority from resume identity. Include current owner ledger and catalog present/absent state in dependencies; catalog invalidation matters because it can change proof reuse/authorization.
- [ ] On hit freshly stat those exact dependencies, then return activity with zero catalog/native/ledger body decoding. Preserve activity's current-launch-only semantics. Changed native bytes trigger current scanner validation (including APFS growth/rewrites); no metadata-only new authority.
- [ ] Verify dependency stability across load/publication: compare validation-observed native metadata and ledger/catalog pre/post observations, retry once on drift. Do not admit an unobserved dependency or bless partial/canceled content work. Where dependency observation is incomplete, remain uncached and fail closed.
- [ ] Keep all existing threadactivity consumers on this same implementation; no separate UI-only rule or new watcher. Existing log/pane-birth signals continue to work when native evidence is unavailable.

## Task 4 — validation, documentation, rollout

- [ ] Named tests for index cache load/commit, owner evidence fallback, cached activity query and production checker/runtime wiring exercise stateful file worlds and real temp files. Barrier-controlled races cover changed-during-read, canceled waiters and stale completion; run with -race.
- [ ] Scale benchmark over repeated unchanged queries reports CPU and allocation profiles before/after. Require no size-proportional warm reads or allocations; growth/mutation triggers validation once, then reaches a warm state again. Do not claim the entire supervisor CPU is fixed solely from a microbenchmark.
- [ ] Run affected packages (launcher, sessioninventory, threadactivity, couchcore, couchcmd/couchtty seams as applicable). Sweep artifactpath's source inventory for new production files and run its tests.
- [ ] Update atlas/couch.md or inventory atlas with cache owners, fingerprint invalidation and remaining cold/changed-input costs; README only if operator flow changes.
- [ ] Push and record full suite via sdlc local-check run --name suite; if a known failure occurs finish the remainder with make -k and report every failure. sdlc close owns fresh review.
- [ ] Ask hub for rollout slot/time; no stage or restart of the live supervisor without direction. Live acceptance: hub/operator rebuild/restart, repeat same bounded sample/CPU observation at idle and one normal delivery; inspect uncertain outcomes by receipts, never auto-resend. Landing order follows hub's acceptance decision and SDLC gates.

## Test strategy by risky function

| Function/seam | Independent invariant / adversarial strategy |
|---|---|
| Session-name cache read/commit | Fresh strict decoder parity under file-world mutations; decoded-byte counter independent of result |
| effective binding projection | Existing uncached effectiveBindings oracle over legacy/scoped histories and reordered reads |
| owner-launch cached resume read | Same Current/PreviousEstablished reducer output for chosen launch fallback; unchanged ReadAt count stays zero |
| cached activity query | Same result as uncached QuerySessionContext/ActivityForSession; dependency mutation never returns prior successful evidence |
| cache publication | Barrier-controlled replacement/cancellation/late completion cannot publish stale generation |
| production owner wiring | Two scopes share one long-lived cache but never share authority; newly created checker/runtime lifetime starts cold |
| warm scale benchmark | Small vs 33 MiB history: zero warm body bytes and bounded small-projection allocations; report measured profiles |

## Approval checkpoint

No production or regression implementation before TL approves this committed design. After approval use full-flow change-code/plan-quality for the expected multi-package change; estimate only after plan-quality as the binary requires.

## Revisions

### 2026-10-14 — explicit message and idle-tick counted release invariants

Reason: TL relayed operator requirement that both paths have named deterministic regressions, reusable by #204 before release. This sharpens Task 1/4; no scope change.

- `TestMessageAuthorityUnchangedFilesDecodeZeroBytes`: drive the production message-authority resolver through ListedPairSessionContext for a 20-slot fixture, with 30 MB owner ledgers and representative legacy/scoped session indexes. Warm every relevant path once, reset counters, then admit/observe further normal peer messages. Require zero decoded bytes and zero body-read bytes for unchanged session indexes/ledgers; retain current process/session authority probes and assert correct recipient identity. Cover exact message admission and subsequent endpoint authority checks, not just direct cache calls.
- `TestIdleActivityTickUnchangedFilesDecodeZeroBytes`: drive the production idle activity probe across all 20 rows with 30 MB ledgers plus valid catalogs/transcripts. Warm once, reset counters, advance complete repeated activity passes and require zero decoded bytes and zero body-read bytes across ledgers/catalogs/transcripts. Assert the same timestamps and unknown states as uncached query semantics; clock-only ticks do not mutate source evidence.
- Count actual IO bytes and actual parser/decoder input bytes separately, by source family (session index, ledger, catalog, native transcript). Counter seams wrap the real readers/decoders; a decoder repeatedly consuming cached raw bytes must fail even when filesystem reads are zero. Return immutable count snapshots/deltas from reusable test helpers so #204 can consume these exact metrics without copying product cache logic.
- Stateful fixtures expose 20 distinct scoped owners; stream/reuse immutable synthetic source buffers rather than allocating twenty 30 MB copies at once. Use production-equivalent valid content, no personal transcripts. Cold loads must prove nonzero counters so a disconnected counter cannot pass. Mutating one owner must cause nonzero validation work for that owner and zero extra work for unchanged owners, followed by zero again on the next stable pass. Include same-size rewrite and replace cases.
- Keep CPU/allocation profiles as diagnostic comparison only; release assertions use exact byte/decode counts, never elapsed-time thresholds. The 20 × 30 MB invariant is part of #559 acceptance, not deferred to #204.
