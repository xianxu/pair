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
