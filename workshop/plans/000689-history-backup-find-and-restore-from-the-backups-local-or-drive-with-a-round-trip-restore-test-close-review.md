# Boundary Review — pair#689 (whole-issue close)

| field | value |
|-------|-------|
| issue | 689 — history backup: find and restore from the backups (local or Drive), with a round-trip restore test |
| repo | pair |
| issue file | workshop/issues/000689-history-backup-find-and-restore-from-the-backups-local-or-drive-with-a-round-trip-restore-test.md |
| boundary | whole-issue close |
| milestone | — |
| window | 390fa467b6f89b8f5f3cfd864cc275af5d1b5dfb..8727648bcc78727ab73c9d1215fe91c6e021974c |
| command | sdlc close --issue 689 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T09:44:30-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation delivers local-only find, label-only Drive index recovery, and verified restore into a fresh destination. Relevant package tests, focused race tests, real-tool round trips, download conformance and repository lint passed. No new blocking findings.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Complete destination markers and receipt precedence are exercised by empty-inventory and folder-rebinding regressions. Both fail with the precedence guard disabled through a temporary compiler overlay.
  - id: BR-2
    disposition: addressed
    note: |
      Extraction limits the complete stream to label.RawBytes plus 32 MiB. Normal and snapshot overflow regressions both fail when the previous kind-wide allowance is restored through a temporary compiler overlay.
  - id: BR-3
    disposition: withdrawn
    note: |
      Both real GPG/zstd tests passed on pinned head using an isolated short TMPDIR, including full and filtered remote-only restore after catalog loss. Empty temporary-root removal succeeded.
```

1. **Strengths**

   - Discovery evidence remains separate from local archive authority; Drive recovery creates neither catalog records nor upload receipts.
   - Restore hashes staged ciphertext before decoding and validates excluded members before publication.
   - Exclusive publication, late decoder failure, cancellation and uncertain durability have meaningful failure-path tests.
   - README and atlas document the new commands, limits and recovery flow.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Passed: `historyarchive`, `archivecmd`, `artifactpath`, and `internal/repolint`.
   - Focused restore/index/CLI race tests passed.
   - Real GPG/zstd create–upload–catalog-loss–sync–find–restore passed.
   - Installed-rclone exact-bound, overflow and selected-download conformance passed.
   - Both prior behavioral fixes have independently confirmed failing-without-fix regressions.
   - Linux publication was not executed in this macOS review environment. The operator’s real Drive acceptance is recorded in the tracker; this review independently exercised isolated real tools.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared key derivation, staging ownership and configuration helpers.
   - **ARCH-PURE — pass:** selection, admission, evidence reconciliation and transitions remain pure; IO uses integration seams.
   - **ARCH-PURPOSE — pass:** offline discovery, catalog-loss recovery and actual decrypted restoration are delivered.
   - **ARCH-MOCK — pass:** stateful fake Drive shares production boundaries; real-tool conformance passed.
   - **ARCH-CONSTRAINTS — pass:** byte, entry, batch and process bounds have boundary tests.
   - **ARCH-SECURE — pass:** strict parsing, safe filesystem traversal and verification precede publication.
   - **ARCH-ORDER — pass:** explicit restore transitions and controllable failure ordering cover publication safety.
   - **ARCH-FUNERAL — pass:** snapshots supersede previous snapshots; owned staging has cleanup/recovery; restored output has explicit operator ownership.
   - **ARCH-EVOLVE — pass:** nested unknown fields survive round trips while known fields remain strict.

7. **Plan revision recommendations:** None required.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f2f623dd-3500-4d4d-85e2-cd5a5c89dcbd","repository":"github.com/xianxu/pair","issue_id":"000689","recipe":"milestone-review","round":5,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/3bf18622ebba7cb2","reviewed_head":"8727648bcc78727ab73c9d1215fe91c6e021974c","slot":"pair:6","requested_at":"2026-10-19T16:37:54.659282Z"},"phase":"finished","pid":42401,"started_at":"2026-10-19T16:37:54.816847Z","finished_at":"2026-10-19T16:44:02.4992Z","session_id":"01a15506-db31-7cd0-965c-6f6123c768a8","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.325697542s
head: "8727648bcc78727ab73c9d1215fe91c6e021974c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-19T16:58:10-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 689 — history backup: find and restore from the backups (local or Drive), with a round-trip restore test |
| repo | pair |
| issue file | workshop/issues/000689-history-backup-find-and-restore-from-the-backups-local-or-drive-with-a-round-trip-restore-test.md |
| boundary | whole-issue close |
| milestone | — |
| window | 609ba83c9cdf7e487744ae009a751257de058228..6bb343364a02a37e1a7a343ab0b1a4491b3c5dc9 |
| command | sdlc close --issue 689 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T16:58:10-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The narrow interdiff preserves both sides’ intent in README and CLI help. It introduces no new runtime logic, architectural surface, persisted format, or Spec/Plan scope change. No blocking findings.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The fix and regression tests are unchanged by this window. Empty-observation, folder-rebinding, newer-receipt and partial-refresh tests passed.
  - id: BR-2
    disposition: addressed
    note: |
      The payload-relative stream bound is unchanged. Exact-bound and one-byte-over regressions passed for ordinary and snapshot kinds.
  - id: BR-3
    disposition: withdrawn
    note: |
      This window changes neither real-tool fixtures nor restore behavior; the prior withdrawal remains applicable. Real-tool qualification was not repeated.
```

1. **Strengths**

   - `cmd/internal/archivecmd/run.go:20` retains `continuous --json` alongside find, sync-index and restore.
   - `README.md:2005` preserves main’s compression, SQLite freshness and Drive-count guidance alongside recovery instructions.
   - Issue Spec, Done when and durable plan remain unchanged.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** The complete `archivecmd` package and focused receipt-precedence and extraction-bound regressions passed. `git diff --check` passed. Full-suite and real-tool qualification were not repeated for this narrow window.

6. **Architectural notes**

   - **ARCH-DRY — pass:** no duplicated implementation introduced.
   - **ARCH-PURE — pass:** no IO or business-logic changes.
   - **ARCH-PURPOSE — pass:** conflict resolutions preserve both existing continuous-mode guidance and recovery commands.
   - **ARCH-EVOLVE — pass:** no persisted-format changes. Tracker updates append review evidence without replacing existing fields.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c1c95d93-a1ad-4462-a18a-731454930e9f","repository":"github.com/xianxu/pair","issue_id":"000689","recipe":"fix-review","round":6,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/98374323075854a9","reviewed_head":"6bb343364a02a37e1a7a343ab0b1a4491b3c5dc9","slot":"pair:6","requested_at":"2026-10-19T23:57:22.579758Z"},"phase":"finished","pid":94282,"started_at":"2026-10-19T23:57:22.779577Z","finished_at":"2026-10-19T23:58:09.198121Z","session_id":"01a15699-3336-7502-aee5-db162eb6847c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.993645916s
head: "6bb343364a02a37e1a7a343ab0b1a4491b3c5dc9"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
