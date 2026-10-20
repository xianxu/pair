# Boundary Review — pair#689 (milestone M1)

| field | value |
|-------|-------|
| issue | 689 — history backup: find and restore from the backups (local or Drive), with a round-trip restore test |
| repo | pair |
| issue file | workshop/issues/000689-history-backup-find-and-restore-from-the-backups-local-or-drive-with-a-round-trip-restore-test.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 390fa467b6f89b8f5f3cfd864cc275af5d1b5dfb..12d4787e53209cb5a0ecc8bc05803e169cb08c25 |
| command | sdlc milestone-close --issue 689 --milestone M1 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T00:19:55-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 delivers local find, bounded label synchronization, additive codecs, and atomic index publication, with substantial passing coverage. One evidence-ordering defect blocks shipping: historical upload receipts can override a newer complete Drive observation.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: complete-observation-precedence
    title: |
      Historical receipts can resurrect locations after a complete Drive refresh
    detail: |
      cmd/internal/historyarchive/label_index_view.go:149-163 projects a receipt as available unless that exact bundle/folder identity already exists. DriveIndexSync.Run reads only the existing index at label_index_drive.go:39, so an initial successful empty Drive sync publishes an empty index even when local catalog records and old upload receipts exist. Subsequent find or local sync rediscovers those records and reports the missing Drive copies as available/verified. Folder rebinding has the same gap when an old receipt identity was never indexed. ARCH-ORDER / ARCH-PURPOSE: preserve complete destination-observation time and binding, including empty observations, and reconcile historical receipts against that evidence. Add regression sequences for initial empty sync followed by find/local sync, and folder rebinding followed by receipt projection.
```

1. **Strengths**
   - Atomic publication distinguishes pre-rename failure from uncertain durability after rename ([label_index_store.go:192](/Users/xianxu/Library/Caches/sdlc/review-checkouts/1a1df1ea5bb06b01/cmd/internal/historyarchive/label_index_store.go:192)).
   - Drive synchronization brackets label downloads with inventories and refuses incomplete observations.
   - Legacy serialization and nested unknown-field preservation have direct tests.
   - README and atlas document both commands, evidence semantics, limits, and M2 scope.

2. **Critical findings**

   The finding above requires destination-wide observation precedence. Checking only whether an exact location already exists cannot represent a successful observation of absence.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:
   - All four changed packages: historyarchive, archivetest, archivecmd, artifactpath.
   - Focused M1 race tests.
   - Focused model and integration tests.
   - Installed-rclone download conformance.
   - Diff whitespace checks.

   Existing tests cover disappearance after an earlier indexed presence. They omit initial empty synchronization and folder rebinding followed by offline receipt projection. No prior findings required disposition.

6. **Architectural notes for upcoming work**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass: shared key derivation and existing filesystem/rclone seams reused. |
   | ARCH-PURE | Pass: selection, merging, and codecs have direct pure tests. |
   | ARCH-PURPOSE | Flag: location knowledge can contradict complete synchronization. |
   | ARCH-MOCK | Pass: stateful fake shares the production runner seam; live download conformance passed. |
   | ARCH-CONSTRAINTS | Pass: metadata/transfer bounds and counted N/2N coverage present. |
   | ARCH-SECURE | Pass: strict metadata admission and ownership-checked cleanup. |
   | ARCH-ORDER | Flag: complete destination observations lack durable precedence over later receipt projection. |
   | ARCH-FUNERAL | Pass: snapshots supersede; owned staging is recovered and removed. |
   | ARCH-EVOLVE | Pass: additive label/index fields survive nested round trips. |

   M1 concept locations and classifications match the implementation; restore entities remain explicitly assigned to M2.

7. **Plan revision recommendations**

   Add a `## Revisions` entry specifying how complete destination observations—including empty inventories and folder rebinding—govern historical receipt projection. Require sequence tests through both `LoadLabelView` and `SyncLocalLabelIndex`.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"72fe9d5c-897a-402e-a536-864ee86c3e50","repository":"github.com/xianxu/pair","issue_id":"000689","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/1a1df1ea5bb06b01","reviewed_head":"12d4787e53209cb5a0ecc8bc05803e169cb08c25","slot":"pair:6","requested_at":"2026-10-19T07:14:46.359217Z"},"phase":"finished","pid":1387,"started_at":"2026-10-19T07:14:46.561021Z","finished_at":"2026-10-19T07:19:54.630358Z","session_id":"01a15303-4a39-7da0-8f48-ca8fdb8917c1","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.169834708s
head: "12d4787e53209cb5a0ecc8bc05803e169cb08c25"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-19T00:34:41-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 689 — history backup: find and restore from the backups (local or Drive), with a round-trip restore test |
| repo | pair |
| issue file | workshop/issues/000689-history-backup-find-and-restore-from-the-backups-local-or-drive-with-a-round-trip-restore-test.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 390fa467b6f89b8f5f3cfd864cc275af5d1b5dfb..247f8fee61f2b94e90f16fd1dea80e3c3ce38102 |
| command | sdlc milestone-close --issue 689 --milestone M1 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T00:34:41-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

M1 delivers local-only find, bounded local/Drive synchronization, catalog-loss recovery, and additive metadata compatibility. BR-1 is addressed with reachable regression coverage that fails when receipt precedence is disabled. No new blocking findings were found. Restore and the operator drill remain explicitly scoped to M2.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      label_index_model.go:272-286 persists complete destination observations even for empty inventories; label_index_view.go:151-153 applies the shared receipt-precedence rule. Empty-sync and folder-rebinding regressions pass at head and both fail with the precedence guard disabled in a disposable checkout, which was removed afterward. Newer same-binding receipt and partial-refresh preservation tests also pass.
```

1. **Strengths**

   - Complete observation evidence now governs indexed and previously unseen receipts consistently.
   - Atomic snapshot publication distinguishes failed publication from uncertain durability.
   - Drive recovery downloads labels without manufacturing private catalog records or receipts.
   - README and atlas cover the new commands, evidence semantics, limits, and M2 scope.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed historyarchive, archivecmd, archivetest, artifactpath, repository lint, focused BR-1 race tests, installed-rclone download conformance, and diff whitespace checks. Disabling the precedence guard made both principal BR-1 regressions fail.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass: shared key derivation and receipt reconciliation. |
   | ARCH-PURE | Pass: selection, codecs, and evidence transitions tested directly without IO. |
   | ARCH-PURPOSE | Pass: M1 discovery and catalog-loss recovery delivered. |
   | ARCH-MOCK | Pass: stateful fake uses the production runner seam; conformance passes. |
   | ARCH-CONSTRAINTS | Pass: metadata/transfer bounds and N/2N work-count tests. |
   | ARCH-SECURE | Pass: strict metadata admission and ownership-checked cleanup. |
   | ARCH-ORDER | Pass: durable completeness markers, serialized mutation, atomic publication. |
   | ARCH-FUNERAL | Pass: bounded snapshots supersede; owned staging is collected. |
   | ARCH-EVOLVE | Pass: nested unknown fields and legacy envelopes survive round trips. |

   M1 concept classifications match their implementation. M2 entities remain deferred as documented.

7. **Plan revision recommendations:** None additional; the BR-1 revision captures the implemented rule.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9aaeb79e-5ba7-455c-8b7d-4a5dc528f9cf","repository":"github.com/xianxu/pair","issue_id":"000689","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/ab2a213fa3429c1c","reviewed_head":"247f8fee61f2b94e90f16fd1dea80e3c3ce38102","slot":"pair:6","requested_at":"2026-10-19T07:31:11.606955Z"},"phase":"finished","pid":30173,"started_at":"2026-10-19T07:31:11.829429Z","finished_at":"2026-10-19T07:34:34.603461Z","session_id":"01a15312-52e6-72c2-9b24-bf72057a1e74","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.758260959s
head: "247f8fee61f2b94e90f16fd1dea80e3c3ce38102"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
