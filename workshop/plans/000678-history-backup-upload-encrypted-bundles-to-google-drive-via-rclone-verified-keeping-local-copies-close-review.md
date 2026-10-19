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
