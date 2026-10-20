# Boundary Review — pair#706 (whole-issue close)

| field | value |
|-------|-------|
| issue | 706 — Drive backup: store bundles in year/month folders instead of one flat folder |
| repo | pair |
| issue file | workshop/issues/000706-drive-backup-store-bundles-in-year-month-folders-instead-of-one-flat-folder.md |
| boundary | whole-issue close |
| milestone | — |
| window | 72a25ac7b601f005fa3199f2622299794a885da3..edad1fb9759600c2ad25aceb49e787dce7deffb7 |
| command | sdlc close --issue 706 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T14:33:27-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range delivers month-folder uploads, legacy compatibility, atomic discovery and recorded-location restore. Both prior fixes are verified; no new findings block shipping.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      rclone_folders.go:161 retains confirmed and uncertain capacity reservations. Both successive-year cases and the uncertain-month regression pass on head and fail when the reservation update is removed through a temporary Go overlay.
  - id: BR-2
    disposition: addressed
    note: |
      README.md:2086 now documents UTC month destinations, legacy conflict refusal, per-folder batches and the shared 4 GiB budget; these match drive_months.go and drive.go.
```

1. **Strengths**

   - Shared layout validation governs upload and recorded locations: `drive_layout.go:12`.
   - Index publication follows complete tree confirmation, preserving the previous snapshot on failure: `label_index_drive.go:46–75`.
   - Receipt tests independently pin legacy encoding and preserve nested unknown fields: `drive_receipt_json_test.go:22`.
   - Stateful failure tests cover uncertain creation, partial progress, folder rebinding and mixed-layout recovery.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed archive, fake, archive CLI, artifact-path and repolint packages; targeted Couch checks; targeted race checks; installed-rclone folder and root/month/mixed restore qualification; and real-key mixed-layout recovery after metadata loss.

   The GPG qualification initially failed under the checkout’s long temporary path, then passed with `TMPDIR=/private/tmp`. The capacity negative control failed as expected. Repository files remain unchanged. This review did not repeat the entire repository suite or contact production Drive.

6. **Architectural notes**

   - **ARCH-DRY — Pass:** shared layout, receipt and refresh helpers avoid parallel implementations.
   - **ARCH-PURE — Pass:** layout, transitions and index transformations have direct tests without IO.
   - **ARCH-PURPOSE — Pass:** upload, discovery, receipt projection and restore support both layouts.
   - **ARCH-MOCK — Pass:** production and stateful fake share the runner seam; native qualification passed.
   - **ARCH-CONSTRAINTS — Pass:** aggregate budgets, serial batches and counted listing tests cover declared bounds.
   - **ARCH-SECURE — Pass:** recorded paths, identities, sizes and hashes are checked before promotion or decoding.
   - **ARCH-ORDER — Pass:** folder transitions require positive observations; uncertainty retains capacity without asserting existence.
   - **ARCH-FUNERAL — Pass:** private stages have cleanup; retained archival folders have an explicit lifecycle policy.
   - **ARCH-EVOLVE — Pass:** receipt and index tests preserve unknown members while retaining strict known-field validation.

7. **Plan revision recommendations:** None. The core-concept classifications and documented transport-bound revision match the implementation.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ed9ef26c-a0cc-435d-96e6-95cb51b96af4","repository":"github.com/xianxu/pair","issue_id":"000706","recipe":"milestone-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/2bc08e3a7d57cd53","reviewed_head":"edad1fb9759600c2ad25aceb49e787dce7deffb7","slot":"pair:6","requested_at":"2026-10-20T21:30:05.105607Z"},"phase":"finished","pid":33524,"started_at":"2026-10-20T21:30:05.23475Z","finished_at":"2026-10-20T21:33:26.931922Z","session_id":"01a15b38-b579-7530-9670-537a4357ad0b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.115514959s
head: "edad1fb9759600c2ad25aceb49e787dce7deffb7"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T14:50:08-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 706 — Drive backup: store bundles in year/month folders instead of one flat folder |
| repo | pair |
| issue file | workshop/issues/000706-drive-backup-store-bundles-in-year-month-folders-instead-of-one-flat-folder.md |
| boundary | whole-issue close |
| milestone | — |
| window | e00984eb04bccc752811fafd3963e388ecacf631..2e938cdceac80d01c0e93f4ab6045fe8d6bdca3e |
| command | sdlc close --issue 706 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T14:50:08-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff correctly repairs eight reader-inventory entries without weakening the scanner. Tracker changes record review outcomes; Spec, Plan and scope remain unchanged. No new defects found. BR-1 and BR-2 remain disposed; neither fix changes in this window.

```findings
{}
```

1. **Strengths**
   - All eight entries in `tests/affected-go-readers.json:294` have correct hashes and fixture classifications.
   - Rejected path literals and private temporary-directory walks are appropriately distinguished from repository-source readers.
   - Scanner enforcement and test assertions remain intact.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage:** Seven reader-guard tests passed. The repository-inventory regression fails with the base inventory and passes with head, demonstrating that this correction resolves the eight reported mismatches. The expensive seeded integration test and full repository suite were not rerun.

6. **Architecture, limited to this interdiff:**
   - **ARCH-DRY — Pass:** existing inventory and scanner reused.
   - **ARCH-PURE — Pass:** no business logic or IO boundaries changed.
   - **ARCH-PURPOSE — Pass:** all eight affected entries audited and corrected.
   - **ARCH-EVOLVE — Pass:** inventory schema and existing classifications preserved.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5dcc5c28-6854-4119-b95a-1bf600c83506","repository":"github.com/xianxu/pair","issue_id":"000706","recipe":"fix-review","round":5,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/9863a64e4d18ac80","reviewed_head":"2e938cdceac80d01c0e93f4ab6045fe8d6bdca3e","slot":"pair:6","requested_at":"2026-10-20T21:49:13.015753Z"},"phase":"finished","pid":50734,"started_at":"2026-10-20T21:49:13.108276Z","finished_at":"2026-10-20T21:50:08.446814Z","session_id":"01a15b4a-3912-7c10-9433-e26e0a437892","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.540151542s
head: "2e938cdceac80d01c0e93f4ab6045fe8d6bdca3e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
