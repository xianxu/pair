---
id: 000507
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '721df1a2b82974c9a728740e1b49358182cca78c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T15:58:19-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "6972e849", done: "c8233b80"}
---

# couch: idle fading re-decodes session transcripts every tick (~1.5 cores at rest)

## Problem

Live Couch (pid 31713, build c8770843) burns 1.25-1.8 cores with no cast running. Measured 2026-10-12: **12.45 s of CPU per 10 s wall**. Two polling paths, measured with a harness against the operator's real store (22 live threads):

1. **Idle fading** (pair#247): one pass costs **8.0 s of CPU** (~360 ms per thread); ~90% is `sessioninventory.readOwnerLaunch` → `sessionledger.ParseLedger`, a strict JSON decode of each thread's whole session ledger on every probe. Ledgers are 15-33 MB (~93 rows of ~1 MB; size issue filed as #508), append-only, and change only on a launch or binding. Passes are every 60 s plus one per switch or inventory landing, and a request during a pass queues another at once, so they can run back to back.
2. **Continuation watcher**: every 500 ms it calls `GetThread` for every hosted thread; each one takes the store lock and re-routes the address (manifest load, symlink evaluation, slot enumeration): **~105-130 ms of CPU per tick** (~21-26% steady).

## Spec

- **Ledger parse cache.** `sessioninventory.OwnerLaunchCache` keeps `readOwnerLaunch`'s answer (current launch, ok, malformed-row diagnostics) per (ledger file, owner), keyed by the ledger's `ArtifactFingerprint` (stable id, generation, ctime mutation token, size, times), which the listing `readOwnerLaunch` already makes provides. A changed fingerprint re-parses; nothing else does. It is an optional runtime capability (like `SessionCatalogAdvancer`), so callers without one, and the caller that needs every record, are unchanged. `threadactivity.NewOSRuntime` owns one cache for Couch's lifetime.
- **Continuation scan cache.** `Couch.ContinuationRequests` remembers, per address, the record file it read and that file's stamp (inode, size, mtime; records are replaced by rename and a migration removes the source, so ctime adds nothing), taken under the store lock just before the read. Each tick `lstat`s that path; unchanged means the cached status stands, and anything else (changed, gone, migrated) re-reads through `GetThread`. `continuationStatus` is a pure function of the record, so this is exact.
- Lifecycle: both caches are in memory, die with Couch, one entry per probed ledger or hosted thread (dozens); an entry for a thread no longer asked about stays until exit (bounded by threads seen in one Couch run).

## Done when

- Idle Couch CPU measured before (12.45 s per 10 s); the probe and the watcher measured before and after on the operator's store, numbers in the Log. The live after-number is taken by the TL at the next restart onto main (Revisions).
- Tests: a cached ledger answer is reused while the fingerprint holds and re-parsed when the ledger grows; the continuation status is reused while the record is unchanged and re-read after a write; a fake runtime counts reads.

## Plan

- [x] `OwnerLaunchCache` in sessioninventory + capability on `OSRuntime`; `QuerySessionContext` uses it; threadactivity owns one.
- [x] Continuation fingerprint cache in couchcore (`ThreadStore` returns the record path and its stat taken under the lock).
- [x] Tests (above), full suite.
- [x] Measure the harness before/after; live Couch after a rebuild is deferred to the TL on main (Revisions).

## Revisions

- 2026-10-12 (TL): the live after-number moves out of Done-when: it is taken by the TL at the operator's next restart onto main after this lands, since pair:0 holds #501 for its smoke test. The harness before/after on the operator's store (activity pass 8.4 s -> 0.34 s CPU warm; continuation tick ~100 ms -> 27 us warm) is the evidence at close.

## Log

### 2026-10-12
- Before (live Couch pid 31713, build c8770843, no cast): **12.45 s of CPU per 10 s wall**.
- Harness on the operator's real store, 22 live threads (throwaway env-gated test, not committed; it builds the production probe and `Couch.ContinuationRequests` over the store's records):
  - before: activity pass 8.0-8.4 s CPU; continuation scan 94-131 ms per tick (every 500 ms).
  - after: activity pass cold 8.6 s (first parse, once per ledger), warm **0.34 s**; continuation scan cold 94 ms, warm **27 µs**.
  - pprof found a second walk in the warm pass: `resampleArtifact` listed the whole native session store (`~/.claude/projects`) to re-stat one transcript, a pair#420 site that was missed. Now `StatFile`.
- Live after-measurement needs a Couch built from this branch: pending (pair:0 holds #501 for the operator's smoke test).
- Mutation checks: each new test fails with its cache disabled, with the ledger cache never invalidated, and with the old `resampleArtifact`.
- Close round 1 (FIX-THEN-SHIP): BR-1 `getThreadStamped` copied `GetThread` and dropped `AfterGetThread`; now `GetThread` wraps it. Minors fixed: the ledger cache keys on the Pair root's path, a hit returns its own Current/Bindings/diagnostics, a wiring test for the shared cache, Spec wording for the stamp (no ctime) and the migration fact. Lesson added.
