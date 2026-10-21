# Boundary Review — pair#755 (whole-issue close)

| field | value |
|-------|-------|
| issue | 755 — couch mock serve: live-reloading Go server for a project's versioned, clickable mocks (product-shaping) |
| repo | pair |
| issue file | workshop/issues/000755-couch-mock-serve-live-reloading-go-server-for-a-project-s-versioned-clickable-mocks-product-shaping.md |
| boundary | whole-issue close |
| milestone | — |
| window | 79ea4ca9e5be10c67c5c356b7f7d4067ebd79a92..5c8ae1b7450f3a3a97b33bba51fb8ba1324beb5d |
| command | sdlc close --issue 755 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T23:23:06-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The feature delivers version selection, clickable samples, authoritative page attributes, live reload, and user documentation. Verification found one blocking contract violation: filesystem reads can follow an in-root ancestor symlink and expose project-root files that HTTP routes promise to keep private.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: filesystem-route-isolation
    title: |
      Ancestor symlinks bypass the private-root HTTP boundary
    detail: |
      cmd/internal/mockserve/project.go:90-97 checks only the final path component with Lstat; os.Root.Open follows ancestor symlinks whose targets remain inside the project root. After starting a valid project, creating mock-1/private -> .. and requesting /mock-1/private/design.md returned HTTP 200 with the private root document before the next scan rejected the symlink. ARCH-SECURE: enforce race-safe, no-follow path resolution at the read boundary rather than relying on an earlier scan. Sweep document, asset and scanner reads, and add a deterministic post-initial-scan ancestor-replacement regression that fails without the fix.
```

1. **Strengths**

   - Cached state requests perform no filesystem scans, with a direct regression guard in `server_test.go:81`.
   - Document bootstrap captures the revision before document IO; browser tests exercise changes before the first poll.
   - Scan tests independently count entry visits and bytes, including empty-directory overflow and same-metadata edits.
   - README and atlas changes cover commands, selection rules, limits, page identity and lifecycle.

2. **Critical findings**

   - **Filesystem route isolation**, detailed above, blocks shipping. `os.Root` confines reads to the project root but does not enforce the narrower public version-directory boundary.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - Tests for `mockserve`, `couchcmd`, `artifactpath` and `repolint`.
   - `mockserve` race tests.
   - Reader-inventory check and range whitespace check.
   - All five Node client tests.
   - Real-browser reload and sample qualification using a temporary Chrome `--no-sandbox` wrapper. Normal Chrome launch failed because sandbox initialization was denied by the review environment.

   Existing confinement tests cover outside-root links, but omit in-root ancestor links introduced after initial validation. The independent HTTP reproduction returned the private document successfully. No tracked files were changed.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** shared version parser, fixture source and scan publisher.
   - **ARCH-PURE — pass:** parsing, document transformation and revision decisions have direct deterministic tests.
   - **ARCH-PURPOSE — pass:** sample and browser qualification exercise clicks, edits, switching and reverting.
   - **ARCH-MOCK — pass:** portable project folders and private HTTP/browser fixtures avoid production dependencies.
   - **ARCH-CONSTRAINTS — pass:** entry/content bounds and counted scan-work tests support the documented envelope.
   - **ARCH-SECURE — flag:** earlier scan validation does not secure subsequent filesystem reads; see the Critical finding.
   - **ARCH-ORDER — pass:** revision changes use the synchronized transition owner; first-poll ordering has controllable tests.
   - **ARCH-FUNERAL — pass:** pointer storage is bounded, ordinary write failures clean temporary files, and server-owned workers are joined.
   - **ARCH-EVOLVE — pass:** fixture bytes are preserved without schema rewrites; selection remains a strict single-value format.

   The core-concept locations and new/modified classifications match the implementation. Pure-function tests do not require IO doubles.

7. **Plan revision recommendations**

   Add a `## Revisions` entry specifying that filesystem confinement must enforce the public subtree at each read, including concurrent ancestor replacement. Name the shared resolution mechanism and deterministic HTTP regression; retain the existing prohibition on symlinks and private-root exposure.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"8a325c2d-ac07-4f9e-8efb-c570bad9fb2d","repository":"github.com/xianxu/pair","issue_id":"000755","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/da1e6dd27e86a5d5","reviewed_head":"5c8ae1b7450f3a3a97b33bba51fb8ba1324beb5d","slot":"pair:4","requested_at":"2026-10-21T06:18:18.297399Z"},"phase":"finished","pid":75745,"started_at":"2026-10-21T06:18:18.43672Z","finished_at":"2026-10-21T06:23:05.697153Z","session_id":"01a15d1c-4f3d-76d3-a0ec-7b5763f68316","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.330918125s
head: "5c8ae1b7450f3a3a97b33bba51fb8ba1324beb5d"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T23:37:26-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 755 — couch mock serve: live-reloading Go server for a project's versioned, clickable mocks (product-shaping) |
| repo | pair |
| issue file | workshop/issues/000755-couch-mock-serve-live-reloading-go-server-for-a-project-s-versioned-clickable-mocks-product-shaping.md |
| boundary | whole-issue close |
| milestone | — |
| window | 79ea4ca9e5be10c67c5c356b7f7d4067ebd79a92..6026591abfcf2590379f7e149e70df3c0617de6f |
| command | sdlc close --issue 755 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T23:37:26-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

BR-1 is addressed, with regression tests that fail when the protections are removed. The implementation delivers the main server, CLI, sample and documentation surfaces. Shipping remains blocked by a browser restoration bug, a reproducible concurrency-test failure, and unavailable real-browser qualification in this review environment.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      project.go now pins and verifies directory and file identities across HTTP, selection and scanner reads. Post-scan alias and controlled replacement tests pass. In disposable pinned-head copies, removing directory pinning reproduced HTTP 200 exposure of design.md in both ancestor-alias cases; disabling identity guards failed directory, in-parent file and scanner-file regressions.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: document-lifecycle-restartability
    title: |
      Back/forward-cache restoration permanently disables live reload
    detail: |
      cmd/internal/mockserve/reload.js:70-75 sets stopped=true and clears polling on pagehide, but never handles pageshow. A document restored from the browser's back/forward cache retains that stopped state without rerunning its scripts, so subsequent edits and version switches are never observed. ARCH-ORDER/ARCH-PURPOSE: model suspension and restoration explicitly, resume exactly one poll on restoration, and retain cancellation on subsequent departures. A deterministic test added to the existing JS harness dispatched persisted pagehide/pageshow and failed because zero polling timers remained.
  - id: new
    severity: Important
    kind: defect
    family: concurrency-test-contract
    title: |
      Concurrent selection regression fails under the new identity checks
    detail: |
      cmd/internal/mockserve/project_test.go:139-169 assumes every concurrent pointer read succeeds, although a legitimate atomic rename between Lstat and Open now produces "file changed during open: .mock-current". Running go test ./cmd/internal/mockserve -run 'TestUseAtomicSelectionReaders|TestProjectRejectsReplacementDuringOpen|TestHTTPRejectsInRootAncestorLinksAfterScan' -count=20 reproduced that failure. The fatal exit also skips wg.Wait, allowing project cleanup to close the root while the writer still runs; the same run reported "file already closed". ARCH-ORDER: define the permitted replacement/refusal outcome, exercise it deterministically, assert whole values for successful reads, and join the writer on every exit without weakening confinement.
  - id: new
    severity: Important
    kind: unverifiable
    family: browser-qualification-evidence
    title: |
      Real-browser acceptance behavior could not be qualified in this environment
    detail: |
      WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/mockserve -run 'TestMockBrowserReload|TestSampleBrowser' -count=1 failed all initial DOM checks with CDP read EOF. A separate private-profile Chrome diagnostic exited with signal 5 and "sandbox initialization failed: Operation not permitted". This prevents verification of the planned browser click/edit/switch/recovery behavior; provide passing output for the pinned reviewed head on a host where Chrome can initialize its sandbox, or enable that capability in the review environment. ARCH-PURPOSE.
```

1. **Strengths**

   - The shared filesystem boundary now protects document, asset and scanner reads. Mutation checks confirmed the BR-1 tests detect removal of the protections.
   - Cached state polling is separated from filesystem scans, with independent work-counting and zero-rescan tests.
   - README and atlas changes document commands, selection policy, limits, fixtures and screen identity. Both documentation gates are satisfied.

2. **Critical findings**

   - `reload.js:70`: resume polling after persisted `pageshow`; test repeated suspend/restore sequences and pending-request cancellation.

3. **Important findings**

   - `project_test.go:139`: reconcile concurrent-reader expectations with fail-closed identity checks and guarantee writer cleanup.
   - Browser qualification requires host evidence as described above.

4. **Minor findings**

   - None.

5. **Test coverage notes**

   - Passed: normal tests for `mockserve`, `couchcmd` and `artifactpath`; `mockserve` race tests; five existing Node tests; affected-reader inventory.
   - Failed: repeated concurrent-selection test and the added restoration regression.
   - BR-1 negative controls produced behavioral failures, not merely compilation failures. Disposable copies were removed; tracked files remain unchanged.

6. **Architectural notes for upcoming work**

   | Marker | Result |
   |---|---|
   | ARCH-DRY | Pass: shared filesystem guards, version parser and scan publication. |
   | ARCH-PURE | Pass: version, command, HTML and revision logic have direct pure tests. |
   | ARCH-PURPOSE | Flag: restoration loses the promised live-reload behavior; browser qualification remains unverified. |
   | ARCH-MOCK | Pass: portable project folders and private HTTP fixtures exercise production boundaries. |
   | ARCH-CONSTRAINTS | Pass: entry, byte and HTML limits are enforced; scan work is measured. |
   | ARCH-SECURE | Pass: BR-1’s enumerable document/asset/scanner read class is covered. |
   | ARCH-ORDER | Flag: restoration is omitted, and concurrent-reader tests assume one favorable ordering. |
   | ARCH-FUNERAL | Pass: scoped handles, temporary-file cleanup and joined scanner lifecycle are present. |
   | ARCH-EVOLVE | Pass: fixture bytes are preserved; the selection pointer remains a strict single value. |

   Core-concept locations and classifications match the implementation.

7. **Plan revision recommendations**

   Add `## Revisions` entries defining document suspension/restoration transitions and their regression strategy, and the permitted concurrent selection-read outcomes. Keep real-browser qualification pending until measured host evidence is available.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"78250bae-041e-45e0-a9e3-c96c7bae9bb2","repository":"github.com/xianxu/pair","issue_id":"000755","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/96421f6ff64905db","reviewed_head":"6026591abfcf2590379f7e149e70df3c0617de6f","slot":"pair:4","requested_at":"2026-10-21T06:33:20.321511Z"},"phase":"finished","pid":1179,"started_at":"2026-10-21T06:33:20.460896Z","finished_at":"2026-10-21T06:37:25.850331Z","session_id":"01a15d2a-1253-7ec3-924c-15e8513605b8","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.5419625s
head: "6026591abfcf2590379f7e149e70df3c0617de6f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000410

---

## Re-review — 2026-10-20T23:52:30-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 755 — couch mock serve: live-reloading Go server for a project's versioned, clickable mocks (product-shaping) |
| repo | pair |
| issue file | workshop/issues/000755-couch-mock-serve-live-reloading-go-server-for-a-project-s-versioned-clickable-mocks-product-shaping.md |
| boundary | whole-issue close |
| milestone | — |
| window | 79ea4ca9e5be10c67c5c356b7f7d4067ebd79a92..897dbb6cdc7c683b03f6e99ccc0a1c7decd1ad62 |
| command | sdlc close --issue 755 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T23:52:30-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The prior behavior fixes are verified, and the documented host browser evidence answers BR-4 for unchanged executable code. Selection, confinement, reload ordering, CLI dispatch, and documentation are well covered. One Important lifecycle gap remains: interrupted selection commands can leave staging files indefinitely, eventually exhausting the scanner’s entry limit.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      The production client handles persisted pageshow with generation/request ownership fencing. All 11 client tests pass; running the current harness against the pre-fix script in memory produces six lifecycle failures, including restoration and stale-completion cases.
  - id: BR-3
    disposition: addressed
    note: |
      TestUseAtomicSelectionReaders accepts only whole values or typed ErrPathChanged refusal and unconditionally joins its writer. TestConcurrentSelectionReadRefusesChangedIdentity deterministically exercises replacement and verifies the settled selection. Twenty race-enabled repetitions of the selection/confinement tests passed.
  - id: BR-4
    disposition: withdrawn
    note: |
      workshop/plans/000755-mock-browser-evidence.md records the exact browser command, passing subtests, exit zero, and sandbox-enabled private-profile qualification on 8fefb5c78182993499354d9227052209570022ea. Git inspection confirms the reviewed head differs only in documentation. Local qualification still fails initial DOM predicates with CDP read EOF; the supplied host measurement answers the requested invariant for identical executable code.
findings:
  - id: new
    severity: Important
    kind: defect
    family: transient-artifact-reclamation
    title: |
      Interrupted selection writers leave staging files without a reclamation path
    detail: |
      cmd/internal/mockserve/project.go:227-244 creates a fresh .mock-current-<nonce>.tmp for every selection, with removal owned only by a defer. Process termination before rename/defer leaves that file permanently; no startup, subsequent writer, or scanner path reclaims abandoned staging files. These files consume the scanner's 1,024-entry allowance (scan.go:70-72), so repeated interrupted commands eventually prevent serving an otherwise valid project. ARCH-FUNERAL: define bounded, concurrency-safe staging ownership/reclamation, preserve active writers and the published pointer, and add interrupted-writer recovery coverage.
```

1. **Strengths**

   - Shared directory/file identity checks protect HTTP, scanner, and selection reads; adversarial replacement tests remain green.
   - Bootstrap revisions precede document reads, and deterministic client tests cover restoration, duplicate events, queued timers, and late completions.
   - Cached state requests perform no scans; counted filesystem tests validate bounded traversal and content reads.
   - README and atlas updates cover commands, layout, limits, identity attributes, and reload behavior.

2. **Critical findings**

   None.

3. **Important findings**

   The staging-file lifecycle finding above. Add automatic reclamation or a bounded staging design that cannot remove another active writer’s files.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - Race-enabled `mockserve`, `couchcmd`, and `artifactpath` package tests.
   - Twenty repetitions of selection/confinement regressions.
   - All 11 Node client tests.
   - Artifactpath/repolint checks and affected-reader inventory.

   The pre-fix JavaScript negative control failed six regressions as expected. Local browser tests remain environment-limited; the recorded host qualification covers identical code. Interrupted-writer residue is currently untested.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** shared parsing, filesystem admission, fixtures, and scan publication.
   - **ARCH-PURE — pass:** parsing, document transformation, and revision decisions are tested independently of IO.
   - **ARCH-PURPOSE — pass:** sample and qualification cover clickable screens, edits, selection, and revert.
   - **ARCH-MOCK — pass:** portable project folders and private HTTP/browser boundaries support integration qualification.
   - **ARCH-CONSTRAINTS — pass:** explicit entry/byte bounds and cached polling are exercised.
   - **ARCH-SECURE — pass:** confined reads, explicit public routes, and adversarial replacement coverage.
   - **ARCH-ORDER — pass:** restoration ownership and concurrent selection outcomes are explicit and exercised.
   - **ARCH-FUNERAL — flag:** abandoned selection staging files have no end.
   - **ARCH-EVOLVE — pass:** strict single-value selection format; fixture bytes are not rewritten.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining selection staging ownership, recovery after process death before publication, reclamation bounds, and concurrency-safe cleanup tests. Update the lifecycle passage at plan line 62 to describe that mechanism.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"af847332-9ef2-46cb-b622-d83f57b31291","repository":"github.com/xianxu/pair","issue_id":"000755","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/f338fd9be858bcae","reviewed_head":"897dbb6cdc7c683b03f6e99ccc0a1c7decd1ad62","slot":"pair:4","requested_at":"2026-10-21T06:49:00.415541Z"},"phase":"finished","pid":43045,"started_at":"2026-10-21T06:49:00.5162Z","finished_at":"2026-10-21T06:52:29.481412Z","session_id":"01a15d38-6a5f-7dd1-ae46-046dfff405d2","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.5454375s
head: "897dbb6cdc7c683b03f6e99ccc0a1c7decd1ad62"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000410

---

## Re-review — 2026-10-21T00:03:00-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 755 — couch mock serve: live-reloading Go server for a project's versioned, clickable mocks (product-shaping) |
| repo | pair |
| issue file | workshop/issues/000755-couch-mock-serve-live-reloading-go-server-for-a-project-s-versioned-clickable-mocks-product-shaping.md |
| boundary | whole-issue close |
| milestone | — |
| window | 79ea4ca9e5be10c67c5c356b7f7d4067ebd79a92..1a41927bc3d9689c62c58d032a8c7bef9b6c69a9 |
| command | sdlc close --issue 755 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T00:03:00-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation delivers the documented versioned mock server, shared fixtures, page identity, reload behavior and sample. BR-5 is addressed with bounded staging and meaningful interruption/recovery coverage. No new blocking findings were identified.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Identity-checked directory/file reads remain shared by HTTP and scanner paths; ancestor-link and replacement regressions pass.
  - id: BR-2
    disposition: addressed
    note: |
      Generation/request ownership fencing remains reachable; repeated restoration and stale response, body, rejection and timer tests pass.
  - id: BR-3
    disposition: addressed
    note: |
      Concurrent readers accept only complete values or ErrPathChanged, verify the settled value, and join writers; package race tests pass.
  - id: BR-4
    disposition: withdrawn
    note: |
      Retain the prior withdrawal. Recorded sandboxed browser qualification targets 584a03cf62abf80bad6e7041f5426a6fbd1cf906; its diff to the reviewed head changes documentation only.
  - id: BR-5
    disposition: addressed
    note: |
      Fixed staging is reclaimed only under the persistent OS lock and excluded from scan content/caps. The killed-writer recovery regression passes; disabling locking makes it fail. Removing reserved-name scan exclusion independently fails its regression. Both negative controls ran in a disposable copy, subsequently removed.
```

1. **Strengths**

   - Selection publication preserves the previous pointer until atomic rename; interruption leaves at most one recoverable stage (`project.go:277`).
   - HTTP routes and scanner reads share directory/file identity checks.
   - Cached observations keep browser polling independent of filesystem scans.
   - README, atlas and the sample document commands, limits, fixtures and version conventions.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed mockserve, couchcmd and artifactpath package tests; mockserve race tests; all 11 JavaScript tests; affected-reader inventory; and diff whitespace checks. Both BR-5 negative controls failed as expected. Real-browser qualification was assessed through the previously accepted host evidence and the documentation-only delta to this head.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared version parsing, read boundary and scan publisher.
   - **ARCH-PURE — pass:** Parsing, HTML transformation and revision decisions have direct deterministic tests.
   - **ARCH-PURPOSE — pass:** Commands, clickable sample, shared fixtures, identity and reload cover the stated purpose.
   - **ARCH-MOCK — pass:** Portable project folders support isolated integration tests; browser conformance uses private profiles.
   - **ARCH-CONSTRAINTS — pass:** Entry, byte and HTML bounds have counted-work and overflow coverage.
   - **ARCH-SECURE — pass:** Explicit public routes, identity-checked reads and unsafe coordination-file refusal.
   - **ARCH-ORDER — pass:** Serialized selection ownership and fenced browser completions have controlled-order regressions.
   - **ARCH-FUNERAL — pass:** Fixed staging bounds residue; owner death releases locking and the next writer reclaims staging.
   - **ARCH-EVOLVE — pass:** Fixture bytes are preserved; the selection pointer remains a strict single-value format.

7. **Plan revision recommendations:** None. The BR-5 revision accurately supersedes the original temporary-file lifecycle.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"99fb5c24-3b89-4f75-8065-11e1da34a1aa","repository":"github.com/xianxu/pair","issue_id":"000755","recipe":"milestone-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/f6904dd752437243","reviewed_head":"1a41927bc3d9689c62c58d032a8c7bef9b6c69a9","slot":"pair:4","requested_at":"2026-10-21T07:00:00.660471Z"},"phase":"finished","pid":84688,"started_at":"2026-10-21T07:00:00.857703Z","finished_at":"2026-10-21T07:03:00.080032Z","session_id":"01a15d42-7f37-7fc1-b91a-c7c75e6ec32c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.024493542s
head: "1a41927bc3d9689c62c58d032a8c7bef9b6c69a9"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T01:42:00-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 755 — couch mock serve: live-reloading Go server for a project's versioned, clickable mocks (product-shaping) |
| repo | pair |
| issue file | workshop/issues/000755-couch-mock-serve-live-reloading-go-server-for-a-project-s-versioned-clickable-mocks-product-shaping.md |
| boundary | whole-issue close |
| milestone | — |
| window | d00a7c4a03509e85cf34d2722d25538552727b20..438872758e60a6f4212cad05b293f677dabae63e |
| command | sdlc close --issue 755 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T01:42:00-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The fix window preserves both sides of the reader-inventory conflict resolution and introduces no new defect. All three entries retain valid digests and appropriate classifications. Tracker changes record completed review and integration activity; they do not change the Spec, Done when, or implementation scope.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Retain the finalized disposition; filesystem implementation and regression tests are unchanged in this window.
  - id: BR-2
    disposition: addressed
    note: |
      Retain the finalized disposition; reload lifecycle implementation and regression tests are unchanged in this window.
  - id: BR-3
    disposition: addressed
    note: |
      Retain the finalized disposition; concurrent-selection tests are unchanged in this window.
  - id: BR-4
    disposition: withdrawn
    note: |
      Retain the prior withdrawal; this window changes neither browser behavior nor the previously accepted qualification evidence.
  - id: BR-5
    disposition: addressed
    note: |
      Retain the finalized disposition; staging ownership, reclamation, scan exclusions and their regression tests are unchanged in this window.
```

1. **Strengths**

   - Conflict resolution preserves the browser reader and both sketch readers, including their audited digests ([inventory](/Volumes/TestScratch/review-checkouts/9f930db67db5a909/tests/affected-go-readers.json:155)).
   - The issue revision accurately records integration work without widening the approved contract.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** Inventory check passed; all 13 inventory regression tests passed; diff whitespace checks passed. A read-only negative control confirmed the base inventory fails JSON parsing at the conflict marker, while the resolved inventory parses and preserves all three entries. Product behavior was not requalified because it is unchanged in this narrow window.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Existing inventory remains the single registry; no parallel mechanism added.
   - **ARCH-PURE — pass:** No business logic or IO boundaries changed.
   - **ARCH-PURPOSE — pass:** Resolution preserves both contributions completely.
   - **ARCH-EVOLVE — pass:** Existing schema, classifications and digests remain intact; no persisted members are discarded.

   No new public or architectural surface requires README or atlas updates.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5d504d12-915f-4835-85d7-ab49f4f4fb53","repository":"github.com/xianxu/pair","issue_id":"000755","recipe":"fix-review","round":5,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/9f930db67db5a909","reviewed_head":"438872758e60a6f4212cad05b293f677dabae63e","slot":"pair:4","requested_at":"2026-10-21T08:39:06.92159Z"},"phase":"finished","pid":99298,"started_at":"2026-10-21T08:39:07.107061Z","finished_at":"2026-10-21T08:41:58.198419Z","session_id":"01a15d9d-3a2c-7321-9ec4-2088b00ed4e0","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.773277292s
head: "438872758e60a6f4212cad05b293f677dabae63e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T02:06:37-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 755 — couch mock serve: live-reloading Go server for a project's versioned, clickable mocks (product-shaping) |
| repo | pair |
| issue file | workshop/issues/000755-couch-mock-serve-live-reloading-go-server-for-a-project-s-versioned-clickable-mocks-product-shaping.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5f2ae6a8e78edf39520476dfde8b966de4ba34ff..31384115c5407a4a5f2f3eabe253ff96bff9f651 |
| command | sdlc close --issue 755 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T02:06:37-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The narrow fix window correctly resolves the reader-inventory conflict and records prior review activity. Both contributions are preserved with current classifications and matching digests. No product behavior or approved scope changes, and no new defects were found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Retain the finalized disposition; confinement implementation and regression tests are unchanged.
  - id: BR-2
    disposition: addressed
    note: |
      Retain the finalized disposition; document restoration implementation and regression tests are unchanged.
  - id: BR-3
    disposition: addressed
    note: |
      Retain the finalized disposition; concurrent-selection behavior and regression tests are unchanged.
  - id: BR-4
    disposition: withdrawn
    note: |
      Retain the prior withdrawal; browser behavior and accepted host qualification evidence are unchanged.
  - id: BR-5
    disposition: addressed
    note: |
      Retain the finalized disposition; staging ownership, recovery, scan exclusions and regression tests are unchanged.
```

1. **Strengths**
   - Inventory resolution preserves both sides’ reader registrations and selects current CLI/reply digests.
   - Issue revisions record integration activity without changing Spec or Done when.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage:** Inventory audit passed; all 13 inventory regression tests passed. Diff whitespace checks passed. A read-only negative control confirmed the base inventory fails JSON parsing at its conflict marker; the resolved inventory validates all 467 entries.

6. **Architecture**
   - **ARCH-DRY — pass:** One authoritative reader registry remains.
   - **ARCH-PURE — pass:** No business logic or IO boundaries changed.
   - **ARCH-PURPOSE — pass:** Resolution preserves both contributions.
   - **ARCH-EVOLVE — pass:** Existing schema and valid registrations remain intact.

   No new surface requires README or atlas updates.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"0e519aaa-05fc-46d1-8b8d-8fdcf0e9d628","repository":"github.com/xianxu/pair","issue_id":"000755","recipe":"fix-review","round":6,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/d94093f4a2a9aad1","reviewed_head":"31384115c5407a4a5f2f3eabe253ff96bff9f651","slot":"pair:4","requested_at":"2026-10-21T09:04:45.951514Z"},"phase":"finished","pid":55599,"started_at":"2026-10-21T09:04:46.126824Z","finished_at":"2026-10-21T09:06:36.45221Z","session_id":"01a15db4-b61d-74c0-930c-85c57821ba46","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.551945375s
head: "31384115c5407a4a5f2f3eabe253ff96bff9f651"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
