# Boundary Review — pair#678 (whole-issue close)

| field | value |
|-------|-------|
| issue | 678 — history backup: upload encrypted bundles to Google Drive via rclone (verified), keeping local copies |
| repo | pair |
| issue file | workshop/issues/000678-history-backup-upload-encrypted-bundles-to-google-drive-via-rclone-verified-keeping-local-copies.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7eaf4fa8e7f26b8a8590b47be5971150d46a442f..e513ccbe821c784c08c4f2c5028c05c3e8338bca |
| command | sdlc close --issue 678 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T19:20:11-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation preserves local backup success, verifies remote pairs before issuing receipts, and handles partial batches conservatively. Relevant tests and installed-rclone local conformance passed. One capacity defect blocks shipping: uploads can grow the remote folder beyond the inventory reader’s limit, permanently preventing further replication.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: writer-reader-capacity-consistency
    title: |
      Upload admission can exceed the remote inventory reader's permanent capacity
    detail: |
      cmd/internal/historyarchive/drive.go:199 limits only the current batch to 50,000 pairs, ignoring objects already present in the destination. DecodeFlatInventory rejects more than 100,000 total entries (rclone.go:179). With 49,999 recorded pairs and two new pairs, admission uploads four objects, producing 100,002 entries; final observation fails before publishing receipts, and every subsequent pass fails its initial observation. ARCH-CONSTRAINTS / ARCH-FUNERAL: reserve destination-wide capacity before copying, account for existing objects and missing members, and visibly defer complete pairs before exceeding reader bounds. Add a regression covering near-capacity admission and rerun recovery.
```

1. **Strengths**

   - Separate upload receipts preserve the existing local catalog’s authority.
   - Final checksums, rather than copy exit status, authorize receipts; partial batch failures can retain independently verified progress.
   - Stateful fake tests cover backlog/rerun counts, partial recovery, late duplicates, cancellation, and staging cleanup.
   - README and atlas document configuration, verification paths, and recovery.

2. **Critical findings**

   - `drive.go:199`: destination-wide capacity is not enforced before upload. Fix as described above.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - `go test ./cmd/internal/historyarchive/... ./cmd/internal/archivecmd -count=1`
   - Installed-rclone local conformance and executable-runner tests.
   - Artifactpath, testpair, and repolint packages.
   - `git diff --check`

   Existing tests cover batch byte admission but omit admission against an already populated remote inventory. No live Google operation was performed during review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** reuses Catalog and LocalSink.
   - **ARCH-PURE — pass:** parsing, validation, admission, and transitions have direct tests.
   - **ARCH-PURPOSE — pass:** replication includes historical backlog and receipt-based reruns.
   - **ARCH-MOCK — pass:** production and stateful fake share the runner seam; local conformance passed.
   - **ARCH-CONSTRAINTS — flag:** batch limits do not preserve destination inventory capacity.
   - **ARCH-SECURE — pass:** bounded decoding, validated names, argv execution, and verified staging.
   - **ARCH-ORDER — pass:** final observations determine partial-batch outcomes; cancellation joins the child.
   - **ARCH-FUNERAL — flag:** the growing remote artifact family can exceed its reader’s bound.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining destination-wide admission capacity, complete-pair deferral at that boundary, and the corresponding regression test. Also reconcile the plan’s promise to register both artifact families with the delivered `NonArtifactSources` classification in `artifactpath/manifest.go:707–711`.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5e918684-6e03-4de2-ab5a-b07506e2574c","repository":"github.com/xianxu/pair","issue_id":"000678","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/fb6ba18604b99f1f","reviewed_head":"e513ccbe821c784c08c4f2c5028c05c3e8338bca","slot":"pair:6","requested_at":"2026-10-19T02:17:04.334174Z"},"phase":"finished","pid":61723,"started_at":"2026-10-19T02:17:04.579983Z","finished_at":"2026-10-19T02:20:10.089042Z","session_id":"01a151f2-bcde-7ef1-aadc-cff060422b2e","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.366957208s
head: "e513ccbe821c784c08c4f2c5028c05c3e8338bca"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T19:25:37-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 678 — history backup: upload encrypted bundles to Google Drive via rclone (verified), keeping local copies |
| repo | pair |
| issue file | workshop/issues/000678-history-backup-upload-encrypted-bundles-to-google-drive-via-rclone-verified-keeping-local-copies.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7eaf4fa8e7f26b8a8590b47be5971150d46a442f..7674d3a24c36cac0ff8109b22499c15382d25fe4 |
| command | sdlc close --issue 678 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T19:25:37-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation delivers verified backlog replication while preserving independent local backup success. BR-1 is addressed with destination-wide capacity reservation and regression tests that fail when the guard is disabled. No new blocking findings emerged; README and atlas cover the new surface.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      drive.go:145 counts all existing destination entries, reserves only missing members before staging/copy, and visibly defers complete pairs. Tests at drive_external_test.go:194 and :222 pass at pinned HEAD. Disabling the admission guard in a disposable HEAD copy made all three cases fail with invalid remote inventory count after excess uploads. Tests also verify rerun stability, vacancy recovery, single-member completion and full-inventory adoption. The disposable copy was removed.
```

1. **Strengths**

   - Local completion and its success marker precede remote work; remote errors remain visible without failing local backup (`continuous.go:128`).
   - Final observations authorize receipts independently of copy exit status, preserving verified progress after partial failures (`drive.go:291`).
   - Stateful fake tests exercise retained bytes, late duplicates, retry, complete-pair deferral and zero-copy reruns (`drive_external_test.go`).
   - Descriptor-checked staging reuses immutable local files, with verified EXDEV copying and cleanup on failure/cancellation (`drive_staging.go:145`).

2. **Critical findings:** None remaining.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed:

   - `go test ./cmd/internal/historyarchive/... ./cmd/internal/archivecmd -count=1`
   - Installed-rclone local-backend conformance and started-child cancellation tests.
   - `go test ./cmd/internal/artifactpath ./cmd/internal/testpair ./internal/repolint -count=1`
   - Patch whitespace checks.

   BR-1’s guard-disabled run reproduced the original failure. Review left the checkout unchanged. Real Drive acceptance was not rerun; the tracker distinguishes its earlier serial-transport evidence from batch qualification.

6. **Architectural notes**

   - **ARCH-DRY — Pass:** Reuses Catalog, LocalSink and existing validation/publication machinery.
   - **ARCH-PURE — Pass:** Destination, receipt, admission and transition logic have direct tests without IO; orchestration owns effects.
   - **ARCH-PURPOSE — Pass:** Projects historical catalog entries and retries backlog independently of newly archived sources.
   - **ARCH-MOCK — Pass:** Production and stateful fake share the runner seam; installed-local conformance passed.
   - **ARCH-CONSTRAINTS — Pass:** Bounds cover subprocess output, phase duration, transfer concurrency, staged bytes and destination entry admission.
   - **ARCH-SECURE — Pass:** Typed validation, duplicate-key rejection, argv invocation, verified staging and bounded/redacted diagnostics protect the relevant boundaries.
   - **ARCH-ORDER — Pass:** Receipt transitions require final pair verification; partial outcomes reconcile through fresh observations. Controlled failure and ordering tests exercise production flow.
   - **ARCH-FUNERAL — Pass:** Staging has synchronous cleanup and next-owner recovery. Receipts have explicit size/count bounds and documented retention.

7. **Plan revision recommendations:** None. The destination-capacity revision matches the implementation; remaining suite/landing checkboxes are workflow obligations.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"3591ece7-ec66-4160-b68d-ef8bb7ec56fd","repository":"github.com/xianxu/pair","issue_id":"000678","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/64322459ac648b05","reviewed_head":"7674d3a24c36cac0ff8109b22499c15382d25fe4","slot":"pair:6","requested_at":"2026-10-19T02:24:07.321391Z"},"phase":"finished","pid":68929,"started_at":"2026-10-19T02:24:07.425413Z","finished_at":"2026-10-19T02:25:36.990453Z","session_id":"01a151f9-2ff9-7dd0-9063-1a10b621b700","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.488014416s
head: "7674d3a24c36cac0ff8109b22499c15382d25fe4"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
