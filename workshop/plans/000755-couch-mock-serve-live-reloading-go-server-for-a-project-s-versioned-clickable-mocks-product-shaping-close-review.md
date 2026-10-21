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
