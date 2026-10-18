# Boundary Review — pair#665 (milestone M1)

| field | value |
|-------|-------|
| issue | 665 — Adopt ariadne's Go lint catalog (pkg/golint) in pair |
| repo | pair |
| issue file | workshop/issues/000665-adopt-ariadne-s-go-lint-catalog-pkg-golint-in-pair.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | d8ccc4577351b3dde5f829e83f5d83e4fedced5d..974cd69ba8974fb2e99b5d208327d7ae03230899 |
| command | sdlc milestone-close --issue 665 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-18T06:12:00-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 delivers the upstream catalog integration, exact sleep identities, shrink validation, fixture ownership, and atlas documentation. Independent inspection confirmed 103 bootstrap sleep sites, 100 remaining debt sites, and no current unadmitted findings across 2,556 sources. Shipping is blocked by a reproduced suppression bypass; the README and external-tool testing contract also need attention.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: sleep-debt-exclusive-authority
    title: |
      Extra whitespace lets inline suppressions bypass the sleep-debt guard
    detail: |
      internal/repolint/catalog_test.go:304 rejects only the exact prefix "//lint:allow test-fixed-sleep". Upstream Check trims whitespace after "//lint:allow ", so two spaces or a space followed by a tab suppress a new sleep without triggering this guard. An overlay regression exercising scanTracked and checkDebt reproduced acceptance with an empty debt list for both spellings. Match the upstream directive grammar and retain regression cases covering its accepted whitespace forms. This violates ARCH-PURPOSE and ARCH-SECURE.
  - id: new
    severity: Important
    kind: defect
    family: readme-surface-discoverability
    title: |
      README omits the new developer lint workflow
    detail: |
      atlas/go-lint-catalog.md:25 introduces suppression syntax, and lines 34–53 introduce debt-file maintenance and the targeted check command, but README.md is unchanged in the pinned range. Add a short entry beside README.md:1878's testing instructions explaining the catalog, linking its atlas page, and identifying the main-history prerequisite. This satisfies the requested README documentation gate.
  - id: new
    severity: Important
    kind: defect
    family: external-tool-model-contract
    title: |
      The new Git boundary has no injectable stateful fake or conformance contract
    detail: |
      internal/repolint/catalog_test.go:255 directly executes Git, and anchorDebt separately executes cat-file; inventory_test.go exercises disposable real repositories only. There is no shared injectable command boundary backed by a stateful fake or scheduled conformance check, as ARCH-MOCK requires. Inject the consumed Git operations, model inventory/ref/history/blob state behind that same seam, and retain real-Git fixtures as conformance coverage with a stated cadence.
```

1. **Strengths**

   - Catalog rules remain owned upstream through the module pin; Pair does not copy their implementation.
   - Debt validation rejects malformed, duplicate, stale, and replacement entries. The independent subset oracle provides useful coverage.
   - `testbinary.Cache` has explicit ownership, concurrent build sharing, retained failures, and cleanup tests.
   - The linked-worktree dependency regression checks both ownership and substrate commit reporting.

2. **Critical findings**

   - `internal/repolint/catalog_test.go:304`: normalize the directive remainder consistently with upstream before checking its rule. Extend `TestScanTrackedRefusesEmptyAndSleepSuppression` beyond its current single-space fixture.

3. **Important findings**

   - Add the README testing entry described above.
   - Complete the Git seam, fake, and conformance contract required by ARCH-MOCK.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Focused dependency, lifecycle, lock, recorder, Chrome, pin-search, cancellation, and probe tests passed. Scanner/debt/cache tests also passed under race detection.

   The ordinary repository test failed because this checkout has neither main tracking configuration nor a remote main ref. A temporary overlay independently exercised the actual bootstrap, scanner, matching, and shrink functions successfully. Separate overlay regressions demonstrated both suppression bypasses. Temporary files were removed; tracked sources remain unchanged. No full-suite PASS is claimed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** upstream rule reuse and shared binary ownership.
   - **ARCH-PURE — pass:** parsing and matching tests require no IO; repository operations remain at the boundary.
   - **ARCH-PURPOSE — flag:** inline whitespace variants defeat exclusive debt authority.
   - **ARCH-MOCK — flag:** missing injected stateful Git model and conformance contract.
   - **ARCH-CONSTRAINTS — pass:** batch workload; counted N/2N source reads.
   - **ARCH-SECURE — flag:** suppression parsing accepts an authorization path the guard misses.
   - **ARCH-ORDER — pass:** inspected timing seams expose relevant ordering; cache access is serialized.
   - **ARCH-FUNERAL — pass:** suite caches have cleanup; debt is bounded and shrink-only.

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying the complete suppression-whitespace regression matrix, the Git fake/conformance contract, and the README deliverable. The latest core-concept tables otherwise match the inspected entities and paths.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"23cfa6d3-7e6b-4133-83ef-17ff7f4b5dba","repository":"github.com/xianxu/pair","issue_id":"000665","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/1e05ed04b6ecf9ad","reviewed_head":"974cd69ba8974fb2e99b5d208327d7ae03230899","slot":"pair:1","requested_at":"2026-10-18T13:09:42.206904Z"},"phase":"finished","pid":79837,"started_at":"2026-10-18T13:09:42.307044Z","finished_at":"2026-10-18T13:11:59.258331Z","session_id":"01a14f21-e041-75c1-a2c0-8e8223fe3c66","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.576858458s
head: "974cd69ba8974fb2e99b5d208327d7ae03230899"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T06:22:23-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 665 — Adopt ariadne's Go lint catalog (pkg/golint) in pair |
| repo | pair |
| issue file | workshop/issues/000665-adopt-ariadne-s-go-lint-catalog-pkg-golint-in-pair.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | d8ccc4577351b3dde5f829e83f5d83e4fedced5d..e3ffaf6c845891bd0d88aa989ff91cd8cbb51c53 |
| command | sdlc milestone-close --issue 665 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-18T06:22:23-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-1 and BR-2 are addressed with verified evidence. The catalog passes against the pinned review base, and the new suppression regression fails without the fix. BR-3 remains open: the measured real-Git coverage is valid, but it does not satisfy the supplied ARCH-MOCK requirement for an injected stateful fake shared with production flow.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      catalog_test.go now matches upstream's prefix, TrimSpace and ASCII-space rule boundary. All 20 accepted whitespace/placement controls pass; restoring the old guard in a disposable pinned-head checkout makes 18 controls fail. The copy was removed afterward.
  - id: BR-2
    disposition: addressed
    note: |
      README.md:1890–1899 adds ordinary/focused lint usage, suppression and debt maintenance, main-history prerequisites and the atlas link. These descriptions agree with catalog_test.go and atlas/go-lint-catalog.md.
  - id: BR-3
    disposition: not-addressed
    note: |
      Re-ran the exact five-test dispute selection: PASS in 1.597s, confirming inventory, 8/16 reads, history authority and malformed/deleted/detached cases. This establishes the claimed functional coverage, but no injectable stateful fake exists: catalog_test.go:261 directly executes Git, and :433 independently executes batch Git. Real-repository fixtures do not establish the fake/real shared boundary required by ARCH-MOCK. Consolidate all consumed Git operations behind one injected boundary, provide a bounded stateful fake for those operations, and retain these real-repository tests as conformance checks.
```

1. **Strengths**

   - Upstream catalog ownership is preserved; Pair reuses the matcher rather than copying rule logic.
   - Debt validation rejects malformed, duplicate, stale and newly admitted sites, with independent subset-oracle tests.
   - Timing fixtures now control meaningful events; recorder and Chrome tests also pass under the race detector.
   - README and atlas cover the new workflow, debt authority and fixture ownership.

2. **Critical findings**

   None remaining.

3. **Important findings**

   **BR-3 — ARCH-MOCK**, `internal/repolint/catalog_test.go:261` and `:433`: the Git boundary remains non-injectable. The dispute’s measurements hold, but answer functional coverage rather than the missing architectural contract. Apply the shared-boundary correction described in its disposition; no new finding is raised.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Catalog package passed in a disposable pinned-head copy with the pinned review base configured as main: **2,556 sources, five rules, 100 debt sites**.
   - The original review checkout correctly refused its missing main-ref metadata.
   - BR-1 mutation verification failed as expected.
   - Cache, recorder, webcast, panebirth, probe and selected couchcore dependency/lifecycle/lock tests passed.
   - Race checks passed for cache, recorder and webcast.
   - Full-suite and native-lane completion are not claimed. The original checkout remains clean.

6. **Architectural notes for upcoming work**

   | Marker | Result | Evidence |
   |---|---|---|
   | ARCH-DRY | Pass | Shared upstream rules and executable cache. |
   | ARCH-PURE | Pass | Pure debt transformations tested directly; IO remains in integration helpers. |
   | ARCH-PURPOSE | Pass | Complete eligible inventory checked; M2 debt remains explicitly outstanding. |
   | ARCH-MOCK | **Flag** | BR-3 remains open. |
   | ARCH-CONSTRAINTS | Pass | Counted N/2N reads; bounded batch workload. |
   | ARCH-SECURE | Pass | Structured argv, strict debt decoding and grammar-aligned suppression rejection. |
   | ARCH-ORDER | Pass | Changed timing tests expose relevant event order; production defaults remain intact. |
   | ARCH-FUNERAL | Pass | Cache cleanup is suite-owned; debt is bounded and shrink-only. |

7. **Plan revision recommendations**

   Add an R5 entry under `## Revisions` replacing R4’s Git-boundary exception with the shared injected boundary, its bounded fake state model, and the existing real-Git tests’ conformance role. The revised core-concept classifications otherwise match the inspected implementation.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"b45346e2-53f1-4e01-9069-d85c4f79068a","repository":"github.com/xianxu/pair","issue_id":"000665","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/d3a16c8c0803c101","reviewed_head":"e3ffaf6c845891bd0d88aa989ff91cd8cbb51c53","slot":"pair:1","requested_at":"2026-10-18T13:19:39.728973Z"},"phase":"finished","pid":37371,"started_at":"2026-10-18T13:19:39.880486Z","finished_at":"2026-10-18T13:22:22.869492Z","session_id":"01a14f2a-fefd-7551-a9f2-03cc6320c1a2","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.371294375s
head: "e3ffaf6c845891bd0d88aa989ff91cd8cbb51c53"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
