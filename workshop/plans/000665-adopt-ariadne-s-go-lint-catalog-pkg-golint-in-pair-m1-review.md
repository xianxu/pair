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
