# Boundary Review — pair#706 (milestone M2)

| field | value |
|-------|-------|
| issue | 706 — Drive backup: store bundles in year/month folders instead of one flat folder |
| repo | pair |
| issue file | workshop/issues/000706-drive-backup-store-bundles-in-year-month-folders-instead-of-one-flat-folder.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 92f5179af794d25206879df566161b1e25431690..a659b2823d24ecdf08b1e9ac8bf7d7924da1d77b |
| command | sdlc milestone-close --issue 706 --milestone M2 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T14:28:44-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned M2 range delivers dual-layout discovery, recorded-location restore, and physical-folder reporting while preserving legacy copies and existing evidence rules. No blocking defects found. README and atlas updates cover the delivered behavior and revised download limits.

```findings
{}
```

1. **Strengths**

   - Index publication follows complete tree confirmation and one atomic refresh: `cmd/internal/historyarchive/label_index_drive.go:65`.
   - Restore checks recorded folder bindings and exact listed sizes before downloading; ciphertext hashing still precedes decoding: `cmd/internal/historyarchive/restore_input.go:220`.
   - CLI and Couch share a projection that prefers final observations without inventing a tree total: `cmd/internal/historyarchive/drive_count_model.go:100`.
   - Tests cover failed-refresh atomicity, disappeared/rebound folders, mixed-layout recovery, unknown-field preservation, and aggregate limits.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Complete historyarchive, archivetest, archivecmd, and repolint packages passed. Focused reporting, Couch warnings, discovery, restore, and artifactpath checks passed.

   Installed-rclone qualification passed for folder creation/retry, odd-sized downloads, transport boundaries, root/month/mixed restore, and listed-size refusal. Real GPG round trips—including mixed recovery after deleting local metadata—passed with a short temporary path.

   The broad run reported Couch picker and artifactpath scope failures; both passed isolated reruns. Their test files and directly exercised consumers are unchanged in this range. The broad invocation therefore was not wholly green. `git diff --check` passed; the checkout remains clean.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Single and batch index refreshes share one implementation.
   - **ARCH-PURE — pass:** Planning, evidence merging, and report projection remain separate from filesystem/network acquisition.
   - **ARCH-PURPOSE — pass:** Recovery supports both legacy root and month locations.
   - **ARCH-MOCK — pass:** Production and stateful fake use the same runner seam; native conformance exercises the adapter.
   - **ARCH-CONSTRAINTS — pass:** Aggregate metadata, destination, ciphertext, and entry limits are enforced.
   - **ARCH-SECURE — pass:** Persisted locations and live identities are validated; actual ciphertext is verified.
   - **ARCH-ORDER — pass:** Failed or unstable observations cannot publish partial snapshots; acquisition failure prevents decoding/publication.
   - **ARCH-FUNERAL — pass:** Private stages retain owned cleanup; archival directories have an explicit retention policy.
   - **ARCH-EVOLVE — pass:** Refreshes preserve unknown fields and historical proofs without weakening validation.

   Core-concept entities exist at the documented locations; the reviewed implementation is consistent with their classifications.

7. **Plan revision recommendations:** None. The existing revision documents the coarse transport backstop and exact application-level verification.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c5c8b3a4-1e39-40f7-94fe-0f9472dce8cf","repository":"github.com/xianxu/pair","issue_id":"000706","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/0a332015a3ee69e8","reviewed_head":"a659b2823d24ecdf08b1e9ac8bf7d7924da1d77b","slot":"pair:6","requested_at":"2026-10-20T21:23:11.917034Z"},"phase":"finished","pid":58234,"started_at":"2026-10-20T21:23:12.120335Z","finished_at":"2026-10-20T21:28:43.789417Z","session_id":"01a15b32-67fe-7151-87ce-9734136d5bab","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.804332917s
head: "a659b2823d24ecdf08b1e9ac8bf7d7924da1d77b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
