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

---

## Re-review — 2026-10-18T06:41:15-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 665 — Adopt ariadne's Go lint catalog (pkg/golint) in pair |
| repo | pair |
| issue file | workshop/issues/000665-adopt-ariadne-s-go-lint-catalog-pkg-golint-in-pair.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | d8ccc4577351b3dde5f829e83f5d83e4fedced5d..9128c9a2e8564801ffc3f8301acfa6da807b6cb6 |
| command | sdlc milestone-close --issue 665 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-18T06:41:15-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

M1 matches the revised plan: the pinned upstream catalog covers tracked Go sources, non-sleep findings are resolved, and 100 existing sleeps remain under the shrink-only debt contract. BR-3 is addressed with a shared injectable Git boundary, stateful fake, and executable conformance coverage. No new blocking findings. The recorded final-head full suite remains a separate landing prerequisite.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The guard follows upstream directive grammar; accepted whitespace/placement controls and non-directive controls pass, including under race.
  - id: BR-2
    disposition: addressed
    note: |
      README.md:1890 documents the focused check, suppression policy, exclusive sleep-debt authority and history prerequisites, consistent with catalog_test.go and atlas/go-lint-catalog.md.
  - id: BR-3
    disposition: addressed
    note: |
      repository_test.go centralizes Git execution behind repositoryRead. Shared read/admission contracts exercise fake and real histories; atlas documents every-run conformance. Bypassing injection makes the fake admission test fail, and removing blob-copy protection makes the snapshot-immutability contract fail.
```

1. **Strengths**

   - Repository admission and historical bootstrap reads share the same typed boundary.
   - Fake state distinguishes worktree, index, immutable commits, ancestry and refs; generated DAG tests use an independent closure oracle.
   - Timing repairs replace scheduling races with observable events while retaining production defaults.
   - README and atlas cover the new workflow, debt lifecycle and fixture ownership.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Full repolint package passed in a disposable checkout using the supplied base as explicit test authority: **2,559 sources, five rules, 100 debt sites**.
   - The original review checkout correctly refused scanning without main authority. Its remaining repolint tests passed under race.
   - Focused dependency, lifecycle, FIFO/lock, timer, telemetry, build-cache and shim tests passed.
   - Mutations proved rejection of a newly tracked sleep, a bypassed repository seam and mutable committed blobs.
   - Disposable checkouts were removed; reviewed files remain unchanged. Full-suite/native qualification was not rerun here.

6. **Architectural notes**

   - **ARCH-DRY — pass:** upstream rules remain authoritative; build ownership is shared.
   - **ARCH-PURE — pass:** debt validation and fake state logic remain separate from Git/filesystem IO.
   - **ARCH-PURPOSE — pass:** tracked consumers are scanned; M2 debt is explicitly bounded.
   - **ARCH-MOCK — pass:** production and fake paths share the boundary and conformance contracts.
   - **ARCH-CONSTRAINTS — pass:** historical reads are batched; N/2N tests enforce single source reads.
   - **ARCH-SECURE — pass:** structured argv, NUL-delimited paths and explicit malformed/unknown-state failures.
   - **ARCH-ORDER — pass:** state transitions and event-controlled timing are exercised.
   - **ARCH-FUNERAL — pass:** fixture artifacts have cleanup owners; sleep debt has a retirement contract.

7. **Plan revision recommendations:** None; R1–R5 cover the delivered design.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"acc93e56-4b86-4700-8ada-07fa59f31486","repository":"github.com/xianxu/pair","issue_id":"000665","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/15b932c99605f348","reviewed_head":"9128c9a2e8564801ffc3f8301acfa6da807b6cb6","slot":"pair:1","requested_at":"2026-10-18T13:38:20.87862Z"},"phase":"finished","pid":91905,"started_at":"2026-10-18T13:38:21.018198Z","finished_at":"2026-10-18T13:41:13.504604Z","session_id":"01a14f3c-19fb-71b3-a908-4b989e103ef4","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.626495125s
head: "9128c9a2e8564801ffc3f8301acfa6da807b6cb6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T07:14:00-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 665 — Adopt ariadne's Go lint catalog (pkg/golint) in pair |
| repo | pair |
| issue file | workshop/issues/000665-adopt-ariadne-s-go-lint-catalog-pkg-golint-in-pair.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 331978755afa0a9aa2d89f5c3add23ac2d65828c..6d7314450a44b1a84a84605c691678390398e10e |
| command | sdlc milestone-close --issue 665 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-18T07:14:00-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The socket-path correction matches R8 and passes meaningful regression controls. No new code defect was found. SHIP is blocked by incomplete verification: this review checkout lacks main authority, so the ordinary catalog test fails before scanning the repository.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Grammar regressions pass; removing TrimSpace through a temporary Go overlay makes the extra-space regression fail.
  - id: BR-2
    disposition: addressed
    note: |
      README.md:1890–1899 documents the focused check, tracking prerequisite and exclusive sleep-debt authority, consistent with catalog_test.go and atlas/go-lint-catalog.md. This passage is unchanged in the current range.
  - id: BR-3
    disposition: addressed
    note: |
      Shared fake/real repository-read and admission contracts pass. Scanner decisions consume repositoryRead; the stateful fake and every-run real-Git conformance remain intact.
findings:
  - id: new
    severity: Important
    kind: unverifiable
    family: review-evidence-completeness
    title: |
      Review checkout cannot verify catalog admission against main
    detail: |
      At internal/repolint/catalog_test.go:447, go test ./internal/repolint -count=1 fails with “need exactly one remote main ref without main tracking config, found 0” before scanning. git show-ref contains no remote main, and neither a remote nor main tracking configuration is present. Supply authentic main tracking/ref/history in the review environment and rerun, or provide measured host evidence for this exact head showing the ordinary catalog test passes against identified main authority. Do not weaken the refusal.
```

1. **Strengths**

   - `transport_test.go:23` preserves owned temporary storage while accommodating the full derived endpoint basename.
   - The regression at `transport_test.go:31` binds real endpoints for both current and ten-digit PIDs.
   - Server cleanup runs before root removal.
   - Atlas documents the derived-address constraint; this range introduces no new README-facing command or configuration.

2. **Critical findings**

   None.

3. **Important findings**

   The verification blocker above requires authentic main authority or exact-head host evidence.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Entire couchmessage package passes under ordinary Darwin TMPDIR: normal **29.037s**, race **32.231s**.
   - Reverting the helper through a temporary overlay makes the new regression fail on a **269-byte** derived address.
   - All repolint tests except the authority-dependent repository check pass.
   - `git diff --check` passes; reviewed source remains unchanged.
   - Initial couchmessage runs failed under the review-injected oversized TMPDIR; ordinary Darwin TMPDIR runs resolve that environment mismatch.

6. **Architectural notes**

   - **ARCH-DRY — pass:** reuses production endpoint derivation.
   - **ARCH-PURE — pass:** native binding remains an integration fixture; pure lint logic stays separate.
   - **ARCH-PURPOSE — pass:** R8 repairs the actual cleanup-consumer path.
   - **ARCH-MOCK — pass:** shared stateful-fake/real Git contracts remain exercised.
   - **ARCH-CONSTRAINTS — pass:** tests the complete native socket address.
   - **ARCH-SECURE — pass:** private owned roots; no production rendezvous directory is created by this regression.
   - **ARCH-ORDER — pass:** listener shutdown precedes root cleanup.
   - **ARCH-FUNERAL — pass:** temporary roots and listeners have explicit cleanup.

7. **Plan revision recommendations**

   None for the implementation. Record restored catalog verification evidence before crossing the boundary.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"14699988-ab6c-4b8a-bb8e-a7c2c7cc266c","repository":"github.com/xianxu/pair","issue_id":"000665","recipe":"milestone-review","round":4,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/1e05772b35e8a5ac","reviewed_head":"6d7314450a44b1a84a84605c691678390398e10e","slot":"pair:1","requested_at":"2026-10-18T14:11:06.940865Z"},"phase":"finished","pid":97405,"started_at":"2026-10-18T14:11:07.065887Z","finished_at":"2026-10-18T14:13:47.645041Z","session_id":"01a14f5a-19b8-79f2-946c-7456b2cd2dbd","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.794264791s
head: "6d7314450a44b1a84a84605c691678390398e10e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
