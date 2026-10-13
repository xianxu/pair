# Boundary Review — pair#507 (whole-issue close)

| field | value |
|-------|-------|
| issue | 507 — couch: idle fading re-decodes session transcripts every tick (~1.5 cores at rest) |
| repo | pair |
| issue file | workshop/issues/000507-couch-idle-fading-re-decodes-session-transcripts-every-tick-1-5-cores-at-rest.md |
| boundary | whole-issue close |
| milestone | — |
| window | 38d6cc78747f31166adc1cc314ada97b6e5db5df..722070ee053d14f673572792d2e70c4320c703f4 |
| command | sdlc close --issue 507 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T17:52:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the diff now; checking cache aliasing and running the tests.

Tests pass apart from sandbox/env failures; doing one quick mutation check on the continuation cache.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The change does what pair#507 set out to do. Couch's idle fading now re-parses a ledger only when the ledger's fingerprint changes. The continuation watcher reuses a thread's status while its record file is still the one it read. `resampleArtifact` stops walking the whole native store and now does one stat. I checked each against the Spec. The ledger cache is safe when the ledger grows between listing and reading: it stores the newer parse under the older fingerprint, so the next query just re-parses. The new tests pin real behavior, and one of them goes red when I disable the cache hit (mutation run below). `sessioninventory` and `threadactivity` tests pass. In `couchcore`, the continuation, recovery and detach tests pass. The only failures are environmental: two niceness tests can't run `/bin/ps` in the sandbox, and two `artifactpath` tests need generated `runtimebundle` assets that aren't in this checkout. Nothing blocks shipping. The one cheap fix worth making first is that `getThreadStamped` is a copy of `GetThread` that drops its `AfterGetThread` hook.

**1. Strengths**
- `query.go:368-437`: `readOwnerLaunch` is split into `findOwnerLedger` and `parseOwnerLedger`. `readCurrentLaunch` reuses both, so there is still one parser, and the caller that needs every record is unchanged.
- `launch_cache.go`: the cache is optional, like `SessionCatalogAdvancer`, and its `get`/`put` do nothing on a nil cache. Runtimes without a cache keep the exact old behavior.
- `threadstore.go:329-333`: the stamp is taken under the lock *before* the read, so a stored record is never newer than its stamp. That is the right order.
- `incremental_inventory.go:451-464`: `resampleArtifact` now uses `StatFile`, closing a pair#420 site that had been missed (pprof found it). `query_test.go` pins it with a test that asserts zero `ListFiles` calls.
- `continuation_store_test.go`: overwriting the record in place proves the cache really skips the read. A mutation that forces a cache miss makes it fail (I ran this).

**2. Critical findings**
None.

**3. Important findings**
- `couchcore/threadstore.go:315-341` (ARCH-DRY): `getThreadStamped` repeats `GetThread` almost line for line (validate, route, delegate to backend, lock, read, clone) but leaves out the `hooks.AfterGetThread` call at `threadstore.go:304`. That hook is how the recovery and detach tests (`recovery_execute_test.go:259,512`, `detach_test.go:35`) inject interleavings. Continuation-scan reads now skip it silently, and the two functions will drift apart. Fix: make `GetThread` a thin wrapper over `getThreadStamped` that drops the stamp, with the hook firing in one shared place.

**4. Minor findings**
- `continuation.go:64`: a cache hit skips `storeForAddress`. If the address later routes to a different backend (for example, a slot repository added to the manifest) while the old record file stays untouched, the stale status keeps being served. The Spec says "migrated" causes a re-read, which holds only if the source file is removed.
- `threadstore.go:357`: `current()` compares inode, size and mtime, but not ctime as the Spec says. An in-place edit that keeps mtime is deliberately invisible (the test relies on that), and after two renames an inode could in principle be reused. Either align the Spec wording or add ctime.
- `launch_cache.go:26` and `query.go:386`: the cache key uses `Artifact.StorageRoot`, which is the constant `"pair-data"` for every scope directory. Different scope directories are told apart only by `scopeKey` and the file's identity in the fingerprint. Adding `runtime.PairDataRoot().Path` to the key would make it unambiguous.
- `query.go:387-388`: a cache hit returns the cached `sessionledger.Current` (with its `Binding *Record` and `Bindings` slice) and the cached `Diagnostic` pointers without copying them. `&current.Binding.RootNativeID` then ends up in the returned `query.Diagnostics` (the publish-failure path), so a caller that mutates the result would corrupt the cache. Clone on a hit.

**5. Test coverage notes**
- Every behavior the Done-when names has a test: ledger reuse and re-parse after growth, continuation reuse and re-read after a write, and read counting through the fake runtime.
- No test covers the `threadactivity.NewOSRuntime` wiring (one cache shared across probes). A small test with two probes on the same thread would pin it.
- Mutation check: forcing `continuationScanCache.get` to always miss makes `TestContinuationRequestsReuseAnUnchangedRecord` fail. I restored the file afterwards and the tree is clean.

**6. Architectural notes**
- **ARCH-DRY:** flagged (Important, above).
- **ARCH-PURE:** pass. The cache logic is small, and the IO stays at the runtime and store seams.
- **ARCH-PURPOSE:** pass. Both hot paths named in the Problem are addressed. Ledger size is properly split out as #508, and the live after-measurement deferral is recorded in Revisions.
- **ARCH-MOCK:** pass. The tests use `FakeRuntime` with operation counts and a real temp-directory thread store.
- **ARCH-CONSTRAINTS:** pass, with a caveat: the harness numbers are logged, but the live idle-CPU number is still pending under the Revisions entry.
- **ARCH-SECURE:** pass. Cached values come from the same validated parse, and nothing touches credentials.
- **ARCH-ORDER:** pass. Both caches are keyed by fingerprint and miss conservatively. The one routing gap is noted above as Minor.
- **ARCH-FUNERAL:** pass. Both caches live in memory, die with Couch, and are bounded by the threads seen in one run, as the Spec declares.
- Docs: `atlas/couch.md` and `atlas/session-identity.md` are updated. There is no new user-facing surface, so README needs no change.

**7. Plan revision recommendations**
- If ctime stays out of the stamp, revise the Spec's "Continuation scan cache" bullet to say "(inode, size, mtime)".

```findings
findings:
  - id: new
    severity: Important
    family: parallel-function-drift
    title: |
      getThreadStamped copies GetThread and drops the AfterGetThread hook
    detail: |
      couchcore/threadstore.go:315-341 repeats GetThread's validate/route/lock/read/clone sequence but omits the hooks.AfterGetThread call at threadstore.go:304, which recovery and detach tests use to inject interleavings. Make GetThread wrap getThreadStamped so the hook fires in one place.
  - id: new
    severity: Minor
    family: cache-bypasses-routing
    title: |
      Continuation cache hit skips storeForAddress, so a re-routed address can serve a stale status
    detail: |
      continuation.go:64 returns the cached status if the old record path still lstats the same; the Spec says migrated re-reads, which holds only if migration removes the source file.
  - id: new
    severity: Minor
    family: spec-impl-drift
    title: |
      recordStamp compares inode, size and mtime but not ctime as the Spec says
    detail: |
      threadstore.go:357. Align the Spec wording or add ctime to the comparison.
  - id: new
    severity: Minor
    family: cache-key-incomplete
    title: |
      OwnerLaunchCache key uses the constant "pair-data" root name instead of the root path
    detail: |
      Different scope directories are distinguished only by scopeKey and the file identity in the fingerprint; add PairDataRoot().Path to ownerLaunchKey.
  - id: new
    severity: Minor
    family: cache-aliasing
    title: |
      Cached Current and Diagnostics are returned without copying, and a pointer into the cache escapes into results
    detail: |
      query.go:387 returns hit.current (Binding *Record, Bindings slice) and hit.diagnostics directly; &current.Binding.RootNativeID ends up in the returned Diagnostics. Clone on a hit.
  - id: new
    severity: Minor
    family: missing-wiring-test
    title: |
      No test that threadactivity.NewOSRuntime shares one ledger cache across probes
```

---

## Re-review — 2026-10-12T18:42:06-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 507 — couch: idle fading re-decodes session transcripts every tick (~1.5 cores at rest) |
| repo | pair |
| issue file | workshop/issues/000507-couch-idle-fading-re-decodes-session-transcripts-every-tick-1-5-cores-at-rest.md |
| boundary | whole-issue close |
| milestone | — |
| window | 38d6cc78747f31166adc1cc314ada97b6e5db5df..ba6d9bd5478bb0f133aa1fcd18022b7be1c4094b |
| command | sdlc close --issue 507 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T18:42:06-07:00 |
| verdict | SHIP |

## Review

Checking the round-1 fixes against the code, then running the tests.

Verifying whether a record can be re-routed while its old file stays on disk (BR-2).

Tests ran; sorting real failures from sandbox ones (blocked `ps`, missing generated assets).

```verdict
verdict: SHIP
confidence: high
```

This round fixes the one Important finding from round 1 (BR-1). The continuation cache and the ledger cache both match the revised Spec. Two Minor fixes, BR-4 and BR-5, changed behavior but came with no regression test, so by the claimed-fix rule they stay `not-addressed`. Minors never block, so they don't stop SHIP. I ran the affected packages. `sessioninventory` and `threadactivity` pass. In `couchcore`, all the continuation, detach, recovery and GetThread tests pass. The only failures I saw were caused by this checkout's sandbox, not the change: `ps` is blocked (`nice_test.go`), and the generated runtimebundle assets are missing, which breaks the `artifactpath` coverage and integration tests.

1. **Strengths**
   - `couchcore/threadstore.go:284-326`: `GetThread` is now a thin wrapper around `getThreadStamped`. There is one read body, so `AfterGetThread` fires for continuation reads as well. The stamp is taken under the store lock, before the read, so a cached record is never newer than its stamp.
   - `couchcore/continuation_store_test.go:259-310`: the test proves the cache is used. It rewrites the record in place with the same inode, size and mtime, and the scan doesn't decode it. Writing the record properly then forces a re-read.
   - `sessioninventory/incremental_inventory.go:451-463`: the extra cost found in pprof (`resampleArtifact` listing the whole native store) is fixed with a single `StatFile`. `query_test.go:87-91` checks that no native `ListFiles` happens.
   - The cache is optional and opt-in, through `OwnerLaunchCacher`, a nil-safe `get`/`put`, and `readOwnerLaunch` left alone for record-hungry callers. Callers that don't use it are untouched.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings:** only the not-addressed BR-4 and BR-5 below. Nothing new.

5. **Test coverage notes**
   - BR-6: `threadactivity/os_test.go` fails if `inventory()` stops passing `r.launches` along (`a == nil`) or makes a new cache per call. `SessionActivity` goes through `inventory()`, so the test covers the production path.
   - BR-1 is a duplication fix. The evidence is that the copied read body is gone. There is no separate behavior to pin beyond what `recovery_execute_test.go` and `detach_test.go` already check through `GetThread`.
   - BR-4 and BR-5 have no test that fails without the fix. Details are in the dispositions.

6. **Architectural notes**
   - **ARCH-DRY: pass.** The parallel read function is gone. `findOwnerLedger` and `parseOwnerLedger` are shared by the cached and uncached paths.
   - **ARCH-PURE: pass.** `continuationStatus` is pure. The caches are thin wrappers keyed by stamp or fingerprint.
   - **ARCH-PURPOSE: pass.** The issue named two polling paths, and both are fixed. pprof found a third (`resampleArtifact`), which is also fixed. The live after-measurement was formally deferred in Revisions, with harness numbers as evidence.
   - **ARCH-MOCK: pass.** The ledger cache test runs against the stateful `FakeRuntime`, which counts ReadAt operations.
   - **ARCH-CONSTRAINTS: pass.** The harness measured before and after: the activity pass went from 8.4 s to 0.34 s, and a continuation tick from about 100 ms to 27 µs.
   - **ARCH-SECURE: pass.** Persisted bytes are trusted only while the file's fingerprint or stamp holds. On any change the code re-parses through the existing strict decoders.
   - **ARCH-ORDER: pass.**
     - The ledger cache's state is just "entry plus fingerprint", replaced under one mutex.
     - The continuation stamp is taken under the lock before the read, which makes reuse safe.
     - For BR-2 I checked migration: `EnrollSlotRepository` retires the source record in its journal (`slotmigration.go:367-371`). Recovery replays before any writer to the slot record, so a re-routed address stops matching its old stamp before a new continuation can be written.
   - **ARCH-FUNERAL: pass.** Both caches live only in memory, die with Couch, and are bounded by the threads seen in one run. The Spec says so explicitly.

7. **Plan revision recommendations:** none. The Spec wording for the stamp and migration now matches `threadstore.go:323-343`.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      threadstore.go:284-287 GetThread wraps getThreadStamped; the single body fires AfterGetThread at :319; copy removed.
  - id: BR-2
    disposition: addressed
    note: |
      Spec now states migration removes the source; verified slotmigration.go:367-371 journals source retirement, so the old stamp stops matching.
  - id: BR-3
    disposition: addressed
    note: |
      Spec line 33 now says inode/size/mtime and explains why ctime adds nothing; this matches recordStamp.current at threadstore.go:337-343.
  - id: BR-4
    disposition: not-addressed
    note: |
      rootPath was added to ownerLaunchKey (query.go:386), but no test uses two Pair roots with identical artifact name and fingerprint, so reverting it stays green.
  - id: BR-5
    disposition: not-addressed
    note: |
      launch_cache.go answer() copies Binding, Bindings and diagnostics, but no test mutates a returned Current/Bindings and re-queries, so reverting to e.current stays green.
  - id: BR-6
    disposition: addressed
    note: |
      threadactivity/os_test.go fails if inventory() drops or recreates the cache; SessionActivity calls inventory().
```
