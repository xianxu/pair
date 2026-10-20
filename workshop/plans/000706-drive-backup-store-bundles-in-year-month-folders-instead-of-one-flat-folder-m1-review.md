# Boundary Review — pair#706 (milestone M1)

| field | value |
|-------|-------|
| issue | 706 — Drive backup: store bundles in year/month folders instead of one flat folder |
| repo | pair |
| issue file | workshop/issues/000706-drive-backup-store-bundles-in-year-month-folders-instead-of-one-flat-folder.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | fc2932e416e5dfb4096e30c8f1ef8baaf355c780..0e07fa2cbad5092e837f181d802ac6500028fd48 |
| command | sdlc milestone-close --issue 706 --milestone M1 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T11:14:07-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 delivers month routing, legacy-pair adoption, compatible receipt codecs, and stateful folder tests. The reviewed packages pass. One correctness bug blocks admission: repeated year creation can exceed the root inventory limit and make subsequent passes fail. README also needs an update for the behavior introduced at this boundary. Discovery, restore, and reporting remain explicitly scoped to M2.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: remote-capacity-reservation
    title: |
      Year creation reuses stale root capacity and can exceed the listing limit
    detail: |
      cmd/internal/historyarchive/drive_months.go:199-202 passes the original root inventory to every EnsureChild call without accounting for newly created years. With 99,999 root entries and pending bundles in two absent years, both mkdir attempts are admitted; the second creates entry 100,001, whose confirmation and subsequent root listings fail DecodeFlatInventory's limit. ARCH-CONSTRAINTS and ARCH-ORDER: carry forward confirmed parent observations and reserve capacity for potentially completed creates. Add a coordinator regression covering successive year creations at capacity, including an uncertain first outcome.
  - id: new
    severity: Important
    kind: defect
    family: user-surface-documentation
    title: |
      README update appears missing for month-folder uploads and per-folder batching
    detail: |
      README.md:2009 still describes one upload batch, while M1 routes new pairs to YYYY/MM and runs a batch per active physical folder. README.md is unchanged in the reviewed range despite this new file-tree convention and the boundary's docs gate. Update the current upload description, legacy behavior, and shared byte budget now; document M2 recovery features when delivered.
```

1. **Strengths**

   - The layout model validates recorded locations against the configured root and the bundle’s canonical UTC month.
   - Receipt tests independently pin legacy encoding and preserve nested unknown fields while rejecting malformed known fields.
   - The folder adapter requires positive unique-ID evidence after mkdir, including lost acknowledgments. Stateful tests exercise ambiguity, cancellation, and retained partial effects.
   - Counted tests cover constant per-folder command cost, unrelated-history exclusion, shared byte admission, and progress surviving another month’s failure.

2. **Critical findings**

   - `drive_months.go:199–202`: root capacity is stale across year creations. Preserve or reserve parent capacity across the entire pass. The decoder’s refusal at `rclone.go:182` makes exceeding this bound operationally persistent until remote entries are removed.

3. **Important findings**

   - `README.md:2009`: update the upload description at M1, alongside the atlas changes already present.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - `go test ./cmd/internal/historyarchive ./cmd/internal/historyarchive/archivetest ./cmd/internal/artifactpath`
   - `go test ./internal/repolint`
   - Pinned-range `git diff --check`

   The full-parent adapter test checks one attempt; it does not cover successive creations using cached parent evidence. The capacity finding follows directly from the production control flow; no regression was added or mutation performed during this read-only review. There were no prior findings to dispose.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** shared layout owner and reused upload verifier.
   - **ARCH-PURE — pass:** layout, receipt codecs, and folder transitions have direct tests without IO.
   - **ARCH-PURPOSE — pass for M1:** month uploads and legacy checks are delivered; remaining consumers are explicitly assigned to M2.
   - **ARCH-MOCK — pass for M1:** production and stateful fake share `RcloneRunner`; native folder qualification remains an explicit M2 obligation.
   - **ARCH-CONSTRAINTS — flag:** cached root capacity permits writes beyond the reader bound.
   - **ARCH-SECURE — pass:** recorded locations, child names, identities, and receipt fields receive boundary validation.
   - **ARCH-ORDER — flag:** individual folder attempts model uncertainty, but parent capacity does not retain progress between attempts.
   - **ARCH-FUNERAL — pass:** archival directory residue is documented; private staging uses existing cleanup paths.
   - **ARCH-EVOLVE — pass:** receipt and nested-object unknown fields survive round trips, with strict known-field validation.

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying pass-wide parent-capacity accounting, including uncertain creates, and moving documentation of M1’s upload layout and batching into this boundary. Keep M2-only discovery and restore documentation attached to M2.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"81cc1f91-ad07-4e84-9a23-cfb004d5152d","repository":"github.com/xianxu/pair","issue_id":"000706","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/42c9bd609fad4fab","reviewed_head":"0e07fa2cbad5092e837f181d802ac6500028fd48","slot":"pair:6","requested_at":"2026-10-20T18:11:40.162716Z"},"phase":"finished","pid":23639,"started_at":"2026-10-20T18:11:40.341101Z","finished_at":"2026-10-20T18:14:06.336751Z","session_id":"01a15a83-0e33-7871-9678-4ae718f246cb","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.523023292s
head: "0e07fa2cbad5092e837f181d802ac6500028fd48"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
