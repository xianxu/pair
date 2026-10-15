# Boundary Review — pair#559 (whole-issue close)

| field | value |
|-------|-------|
| issue | 559 — Stop reparsing inventory on Couch ticks |
| repo | pair |
| issue file | workshop/issues/000559-couch-tick-inventory.md |
| boundary | whole-issue close |
| milestone | — |
| window | c6af746c94f18ef4a58cb092a739cc9fe0887124..e541a6d669708ef6743504511c9d9868c53541e3 |
| command | sdlc close --issue 559 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T17:28:05-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The main cache wiring and warm-path performance regressions are sound. Shipping is blocked by incomplete-observation invalidation gaps. The previous-launch cache also retains unnecessary historical payloads, and activity publication lacks the adversarial ordering tests promised by the plan.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: incomplete-observation-invalidates-evidence
    title: |
      Failed observations can preserve or publish reusable evidence
    detail: |
      cmd/internal/sessioninventory/query.go:370-389 returns on failed listing/stat or missing ledger without invalidating the existing owner entry; after recovery with unchanged metadata, QueryResumeTarget reuses the old parse. Separately, activity_cache.go:378-379 does not record catalog-load failure, which QuerySessionContext intentionally tolerates; successful native fallback can then publish a ready activity entry at lines 212-216 despite incomplete catalog observation. This contradicts the Spec's explicit invalidation rule (ARCH-SECURE, ARCH-ORDER). Invalidate owner evidence on incomplete source observations and prevent activity publication after failed catalog observation; retain uncached fallback behavior. Add failure→recovery regressions proving fresh observation occurs.
  - id: new
    severity: Important
    kind: defect
    family: compact-cache-projections
    title: |
      Previous-launch reuse retains full historical boundary and proof payloads
    detail: |
      cmd/internal/sessioninventory/query.go:476-478 retains the complete previous sessionledger.Current in ownerLaunchEntry.previous (launch_cache.go:34), including launch boundary arrays and binding scanner proofs. ResumeTargetForLaunch consumes only identity/status fields, so these potentially large payloads remain resident unnecessarily across up to 128 entries (ARCH-CONSTRAINTS, ARCH-FUNERAL). Store a compact fallback projection and audit both retained launch projections for unused payloads. Add a substantive large-boundary fixture and retained-memory evidence; the current scale fixture expands files with whitespace and cannot detect this retention.
  - id: new
    severity: Important
    kind: defect
    family: temporal-cache-regression-coverage
    title: |
      Activity publication lacks the promised controlled ordering regressions
    detail: |
      cmd/internal/sessioninventory/activity_cache_test.go:89-106 cancels the purported waiter before Query starts, so it never exercises waiting behind the loader. The activity tests also do not replace dependencies during a blocked load or force repeated drift before publication. Owner/index race tests do not exercise the activity tracker and commit path (ARCH-ORDER). Add barrier-controlled activity mutation, active-loader cancellation, and genuinely queued-waiter tests that fail when the corresponding publication guards are removed.
```

1. **Strengths**

   - Collision-checker copies share constructor-owned index and ledger caches; integration tests exercise both consumers.
   - Cache misses retain the existing strict decoders, launch reducers, and scanner validation.
   - The 20-slot activity regression exercises 600 MB of cold ledger parsing and asserts zero unchanged body-read/decoder work.
   - Atlas documentation explains ownership, invalidation, saturation, and remaining cold costs. No new operator-facing command/configuration surface requires README changes.

2. **Critical findings**

   The incomplete-observation finding above violates the Spec’s explicit “invalidate … any incomplete observation” contract. Sweep all cache observation failure paths when correcting it.

3. **Important findings**

   Compact the retained fallback evidence and complete the activity ordering regressions described above.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed launcher, sessioninventory, and threadactivity package tests; targeted cache/wiring tests under `-race`; the named message invariant; and targeted artifact source-inventory checks. `git diff --check` passed.

   The broader package run was interrupted after roughly six minutes while couchcore remained running; it is not a full-suite pass. The checkout remained unchanged.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass: existing parsers, reducers, and validators reused. |
   | ARCH-PURE | Pass: projection/matching functions have direct pure tests; IO remains behind runtime seams. |
   | ARCH-PURPOSE | Flag: invalidation and compact-evidence commitments remain incomplete. |
   | ARCH-MOCK | Pass: injected runtimes and temporary file worlds cover the changed seams; no new external dependency. |
   | ARCH-CONSTRAINTS | Flag: entry-count bounds do not establish compact retained payloads. |
   | ARCH-SECURE | Flag: incomplete observations can become reusable evidence. |
   | ARCH-ORDER | Flag: failure invalidation and activity publication ordering coverage need correction. |
   | ARCH-FUNERAL | Flag: unnecessary previous-generation payloads survive until eviction/caller exit. No new durable file family was introduced. |

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying failure invalidation across each cache, compact current/previous evidence fields, and the concrete activity ordering regressions. Refresh the Core concepts table with actual function names, paths, PURE/INTEGRATION kinds, and new/modified statuses. Keep the affected checklist claims open until those corrections are verified.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.91872875s
head: "e541a6d669708ef6743504511c9d9868c53541e3"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-14T17:46:25-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 559 — Stop reparsing inventory on Couch ticks |
| repo | pair |
| issue file | workshop/issues/000559-couch-tick-inventory.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4780a55286f5a171d64907cf03d9c12c7a2ce656..dbf3acf30224bbaad8ae1c0c1f625d7a538063ae |
| command | sdlc close --issue 559 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T17:46:25-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The cache implementation and measured warm-work reductions look sound. BR-2 and BR-3 are addressed, but BR-1 lacks independent regression evidence for catalog-load failure: its committed test remains green when the load-failure guard is removed. This blocks acceptance under the required “fails without the fix” rule.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Owner invalidation is verified: removing it fails failure→recovery and active-loader regressions. Catalog-load invalidation still lacks independent regression evidence. Removing activity_cache.go:402-404 leaves TestActivityCacheFailedCatalogObservationNeverPublishesFallback green because activity_cache_failure_test.go:30-34 also fails publication, which independently prevents caching. A disposable overlay allowing publication success passes with the load guard and fails without it with “incomplete catalog observation published reusable fallback.” Commit that independent read-failure regression.
  - id: BR-2
    disposition: addressed
    note: |
      Previous evidence is now a ResumeTarget; compactOwnerLaunch removes unused current boundaries, watermarks and binding histories while retaining required authorization proof. The valid substantive fixture measured 906502 source bytes and 615 retained encoded bytes. Removing compaction fails TestCompactOwnerLaunchKeepsCurrentProofDropsUnusedPayloads.
  - id: BR-3
    disposition: addressed
    note: |
      Activity tests now establish an actually queued waiter, cancel an active loader, replace a ledger during blocked loading, and force drift across both bounded attempts. Targeted race checks pass. Bypassing stability validation fails both replacement and repeated-drift regressions.
```

1. **Strengths**
   - Caller-owned caches share production readers while keeping live ownership checks fresh.
   - The idle invariant actually reads/decodes 600 MB cold, then performs zero unchanged ledger/catalog/native body work.
   - Small and large histories both measured 317 warm allocations.
   - Atlas documentation covers cache ownership, invalidation and lifetime; no new operator surface requires README changes.

2. **Critical findings**
   - **BR-1 remains open**, at `cmd/internal/sessioninventory/activity_cache_failure_test.go:30`. Separate catalog read and publication failure controls. Test failed reads with successful publication, uncached fallback, fresh recovery and subsequent reuse. Removing the load-failure guard must fail this committed regression. **ARCH-SECURE, ARCH-ORDER**.

3. **Important findings**
   - None newly raised.

4. **Minor findings**
   - None.

5. **Test coverage notes**
   - Inventory, launcher and threadactivity package checks passed.
   - Targeted inventory/launcher/couchcore race checks, artifactpath tests and both named release invariants passed.
   - Counterfactual overlays were removed; repository files remain unchanged.
   - The broader multi-package run was stopped during couchcore execution. This review does not establish full-suite success.

6. **Architectural notes**
   - **ARCH-DRY — pass:** existing strict parsers and reducers reused.
   - **ARCH-PURE — pass:** projection, equality and transition logic have direct tests.
   - **ARCH-PURPOSE — pass:** both production cache paths are wired and measured.
   - **ARCH-MOCK — pass:** injected, stateful filesystem/catalog fixtures exercise shared seams.
   - **ARCH-CONSTRAINTS — pass:** bounded entries, bounded retries and history-independent warm allocations verified.
   - **ARCH-SECURE — flag:** BR-1’s independent failure regression remains missing.
   - **ARCH-ORDER — flag:** BR-1 remains; BR-3’s controlled ordering coverage is verified.
   - **ARCH-FUNERAL — pass:** caller lifetime and eviction remove retained evidence; no new durable artifacts.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry requiring independent failure injection for each observation/publication seam. Record the catalog read-failure/write-success counterfactual before marking BR-1 complete.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.180338542s
head: "dbf3acf30224bbaad8ae1c0c1f625d7a538063ae"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
