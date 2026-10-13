---
gate: boundary-review
issue: 507
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T17:52:11-07:00"
      dispatched: "2026-10-12T17:29:08-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: getThreadStamped copies GetThread and drops the AfterGetThread hook
          detail: couchcore/threadstore.go:315-341 repeats GetThread's validate/route/lock/read/clone sequence but omits the hooks.AfterGetThread call at threadstore.go:304, which recovery and detach tests use to inject interleavings. Make GetThread wrap getThreadStamped so the hook fires in one place.
          family: parallel-function-drift
          round: 1
        - id: BR-2
          severity: Minor
          title: Continuation cache hit skips storeForAddress, so a re-routed address can serve a stale status
          detail: continuation.go:64 returns the cached status if the old record path still lstats the same; the Spec says migrated re-reads, which holds only if migration removes the source file.
          family: cache-bypasses-routing
          round: 1
        - id: BR-3
          severity: Minor
          title: recordStamp compares inode, size and mtime but not ctime as the Spec says
          detail: threadstore.go:357. Align the Spec wording or add ctime to the comparison.
          family: spec-impl-drift
          round: 1
        - id: BR-4
          severity: Minor
          title: OwnerLaunchCache key uses the constant "pair-data" root name instead of the root path
          detail: Different scope directories are distinguished only by scopeKey and the file identity in the fingerprint; add PairDataRoot().Path to ownerLaunchKey.
          family: cache-key-incomplete
          round: 1
        - id: BR-5
          severity: Minor
          title: Cached Current and Diagnostics are returned without copying, and a pointer into the cache escapes into results
          detail: query.go:387 returns hit.current (Binding *Record, Bindings slice) and hit.diagnostics directly; &current.Binding.RootNativeID ends up in the returned Diagnostics. Clone on a hit.
          family: cache-aliasing
          round: 1
        - id: BR-6
          severity: Minor
          title: No test that threadactivity.NewOSRuntime shares one ledger cache across probes
          family: missing-wiring-test
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T18:42:06-07:00"
      dispatched: "2026-10-12T18:23:24-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: threadstore.go:284-287 GetThread wraps getThreadStamped; the single body fires AfterGetThread at :319; copy removed.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Spec now states migration removes the source; verified slotmigration.go:367-371 journals source retirement, so the old stamp stops matching.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Spec line 33 now says inode/size/mtime and explains why ctime adds nothing; this matches recordStamp.current at threadstore.go:337-343.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: rootPath was added to ownerLaunchKey (query.go:386), but no test uses two Pair roots with identical artifact name and fingerprint, so reverting it stays green.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: launch_cache.go answer() copies Binding, Bindings and diagnostics, but no test mutates a returned Current/Bindings and re-queries, so reverting to e.current stays green.
          round: 2
        - id: BR-6
          disposition: addressed
          note: threadactivity/os_test.go fails if inventory() drops or recreates the cache; SessionActivity calls inventory().
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: ba6d9bd5478bb0f133aa1fcd18022b7be1c4094b
      blocked: false
---

# Gate ledger — pair#507 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T17:52:11-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `parallel-function-drift` getThreadStamped copies GetThread and drops the AfterGetThread hook
  couchcore/threadstore.go:315-341 repeats GetThread's validate/route/lock/read/clone sequence but omits the hooks.AfterGetThread call at threadstore.go:304, which recovery and detach tests use to inject interleavings. Make GetThread wrap getThreadStamped so the hook fires in one place.
- **BR-2** [Minor] `cache-bypasses-routing` Continuation cache hit skips storeForAddress, so a re-routed address can serve a stale status
  continuation.go:64 returns the cached status if the old record path still lstats the same; the Spec says migrated re-reads, which holds only if migration removes the source file.
- **BR-3** [Minor] `spec-impl-drift` recordStamp compares inode, size and mtime but not ctime as the Spec says
  threadstore.go:357. Align the Spec wording or add ctime to the comparison.
- **BR-4** [Minor] `cache-key-incomplete` OwnerLaunchCache key uses the constant "pair-data" root name instead of the root path
  Different scope directories are distinguished only by scopeKey and the file identity in the fingerprint; add PairDataRoot().Path to ownerLaunchKey.
- **BR-5** [Minor] `cache-aliasing` Cached Current and Diagnostics are returned without copying, and a pointer into the cache escapes into results
  query.go:387 returns hit.current (Binding *Record, Bindings slice) and hit.diagnostics directly; &current.Binding.RootNativeID ends up in the returned Diagnostics. Clone on a hit.
- **BR-6** [Minor] `missing-wiring-test` No test that threadactivity.NewOSRuntime shares one ledger cache across probes

## Round 2 — 2026-10-12T18:42:06-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — threadstore.go:284-287 GetThread wraps getThreadStamped; the single body fires AfterGetThread at :319; copy removed.
- BR-2 — addressed — Spec now states migration removes the source; verified slotmigration.go:367-371 journals source retirement, so the old stamp stops matching.
- BR-3 — addressed — Spec line 33 now says inode/size/mtime and explains why ctime adds nothing; this matches recordStamp.current at threadstore.go:337-343.
- BR-4 — not-addressed — rootPath was added to ownerLaunchKey (query.go:386), but no test uses two Pair roots with identical artifact name and fingerprint, so reverting it stays green.
- BR-5 — not-addressed — launch_cache.go answer() copies Binding, Bindings and diagnostics, but no test mutates a returned Current/Bindings and re-queries, so reverting to e.current stays green.
- BR-6 — addressed — threadactivity/os_test.go fails if inventory() drops or recreates the cache; SessionActivity calls inventory().

## Open findings

- **BR-4** [Minor] `cache-key-incomplete` OwnerLaunchCache key uses the constant "pair-data" root name instead of the root path
- **BR-5** [Minor] `cache-aliasing` Cached Current and Diagnostics are returned without copying, and a pointer into the cache escapes into results
