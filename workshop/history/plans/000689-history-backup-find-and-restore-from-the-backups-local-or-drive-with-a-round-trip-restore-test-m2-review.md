# Boundary Review — pair#689 (milestone M2)

| field | value |
|-------|-------|
| issue | 689 — history backup: find and restore from the backups (local or Drive), with a round-trip restore test |
| repo | pair |
| issue file | workshop/issues/000689-history-backup-find-and-restore-from-the-backups-local-or-drive-with-a-round-trip-restore-test.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 2be1cbf32ae63759733a76a75587db3b28e02320..2a47ec08c1ca739d5ba5ef73faba23517785be42 |
| command | sdlc milestone-close --issue 689 --milestone M2 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T09:22:02-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The restore pipeline has strong verification and publication safeguards, and the default package tests pass. SHIP is blocked by a decompressed-stream limit that exceeds the plan’s declared overhead allowance. Real GPG/zstd qualification also remains unverified in this review environment because gpg-agent could not start.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: resource-bounds-follow-admitted-input
    title: |
      Restore permits tar overhead far beyond the declared 32 MiB allowance
    detail: |
      cmd/internal/historyarchive/restore_extract.go:147 budgets rawBundleLimit(label.Kind) + 32 MiB instead of label.RawBytes + MaxRestoreTarOverhead. A valid tiny bundle followed by 64 MiB of zero padding therefore passes extraction, despite the plan's separate 32 MiB overhead cap (workshop/plans/000689-find-restore-index-plan.md:121). Snapshot kinds permit up to roughly 8 GiB of this extra work per bundle. ARCH-CONSTRAINTS: derive the complete-stream budget from the validated declared payload and shared overhead constant; add a regression rejecting excess padding on a small bundle. The existing overflow test at restore_extract_test.go:270 only exercises the larger kind-based ceiling.
  - id: new
    severity: Important
    kind: unverifiable
    family: real-tool-qualification-evidence
    title: |
      Real-key round-trip qualification could not reach restore
    detail: |
      Running PAIR_ARCHIVE_LIVE=1 go test ./cmd/internal/historyarchive -run '^(TestCodecLiveRoundTrip|TestRestoreLiveRemoteOnlyRoundTrip)$' -count=1 -v failed both tests during disposable key generation: gpg-agent reported "File name too long" and "No agent running". NewGPGHome uses os.TempDir() at cmd/internal/archivetestenv/gpg.go:47; this environment's review temporary directory makes the agent socket path too long. ARCH-MOCK: provide a measured passing run of these tests on the pinned code with an isolated short temporary root, including agent/home cleanup, or make the fixture support this environment and rerun. The implementation log alone does not verify this executable acceptance contract.
```

1. **Strengths**

   - All staged ciphertext is hashed before decoding; advertised remote hashes cannot bypass verification ([restore_input.go:106](cmd/internal/historyarchive/restore_input.go:106)).
   - Extraction verifies excluded members, checks manifest identity and summaries, and rejects unsafe member types.
   - Decoder completion is joined before publication; exclusive rename preserves concurrent targets, and parent-sync failure reports uncertainty ([restore.go:172](cmd/internal/historyarchive/restore.go:172)).
   - README and atlas describe the command, scratch ownership, limits, and recovery behavior.

2. **Critical findings**

   The stream-budget defect above needs a payload-relative bound and a regression that fails on this head.

3. **Important findings**

   The real-tool qualification failure above needs measured evidence answering the environment restriction.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Default historyarchive/archivecmd tests passed; historyarchive completed in 172.442 seconds.
   - Focused restore/decoder tests passed in 68.178 seconds.
   - Tests cover late integrity failure, cancellation, corrupt ciphertext, partial downloads, concurrent targets, and uncertain publication.
   - Real GPG/zstd tests failed before exercising restore. Linux publication was not executed in this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing hashing, safe filesystem, staging, and rclone helpers are reused.
   - **ARCH-PURE — pass:** planning and transitions have direct tests; IO stays in integration components.
   - **ARCH-PURPOSE — pass:** production paths cover selective restoration and catalog-loss recovery.
   - **ARCH-MOCK — flag:** real-tool conformance remains unverified as detailed above; stateful Drive tests pass.
   - **ARCH-CONSTRAINTS — flag:** decompressed overhead does not follow the admitted payload.
   - **ARCH-SECURE — pass:** ciphertext, manifests, members, and filesystem paths are validated.
   - **ARCH-ORDER — pass:** ordered transitions, joined children, and publication uncertainty are explicit.
   - **ARCH-FUNERAL — pass:** owned staging is cleaned or recovered; published scratch output belongs to its operator.
   - **ARCH-EVOLVE — pass:** this range leaves label/index formats unchanged and preserves restored manifest bytes.

7. **Plan revision recommendations**

   Add a `## Revisions` entry recording the corrected payload-relative stream budget and its regression. Record the successful real-tool qualification environment and distinguish Linux cross-compilation from runtime execution.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"2bf663ba-ab07-408e-b84b-a2c84bab3a13","repository":"github.com/xianxu/pair","issue_id":"000689","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/2d6a752cb533564c","reviewed_head":"2a47ec08c1ca739d5ba5ef73faba23517785be42","slot":"pair:6","requested_at":"2026-10-19T16:18:10.751319Z"},"phase":"finished","pid":11002,"started_at":"2026-10-19T16:18:10.948589Z","finished_at":"2026-10-19T16:22:00.695308Z","session_id":"01a154f4-cc78-7192-834b-e992c90ceb16","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.54375575s
head: "2a47ec08c1ca739d5ba5ef73faba23517785be42"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-19T09:35:45-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 689 — history backup: find and restore from the backups (local or Drive), with a round-trip restore test |
| repo | pair |
| issue file | workshop/issues/000689-history-backup-find-and-restore-from-the-backups-local-or-drive-with-a-round-trip-restore-test.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 2be1cbf32ae63759733a76a75587db3b28e02320..39ff9f9a6fd0283b6be10486f0aed3c890f4a3ba |
| command | sdlc milestone-close --issue 689 --milestone M2 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T09:35:45-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The M2 restore implementation satisfies the reviewed contracts. BR-2 has a regression that fails with the old implementation; BR-3’s qualification dispute is supported by an independently successful real-key run. No new blocking findings. The broader package run encountered an unchanged fixture defect, reproduced on both pinned base and head.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      restore_extract.go:147 now budgets label.RawBytes + MaxRestoreTarOverhead. TestExtractBundleRejectsDecompressedPaddingOverflow covers exact-bound success and one-byte overflow refusal for normal and snapshot kinds. A temporary compiler overlay restoring the old expression made both overflow cases fail.
  - id: BR-3
    disposition: withdrawn
    note: |
      Independently reran both specified opt-in tests on pinned head with a canonical private short TMPDIR. Both passed, including full and filtered production restore after catalog loss. Post-test root entries and matching GPG agents were zero; temporary-root removal was confirmed. This supports the disputed environment explanation.
```

1. **Strengths**

   - All ciphertext is staged and verified before decoding; remote evidence cannot substitute for hashing ([restore_input.go:109](cmd/internal/historyarchive/restore_input.go)).
   - Extraction verifies excluded members and joins decoder completion before publication ([restore.go:167](cmd/internal/historyarchive/restore.go)).
   - Exclusive publication preserves concurrent targets, and parent-sync failure reports uncertainty ([restore.go:202](cmd/internal/historyarchive/restore.go)).
   - README and atlas document the command, resource limits, ownership, and recovery behavior.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed focused race tests, real GPG/zstd round trips, installed-rclone bound conformance, archive CLI tests, fake tests, artifact inventory, and repository lint.

   The broader historyarchive run failed `TestDriveMirrorBatchBudgetDefersCompletePairs`. Controlled fractional-mtime fixtures reproduced the same failure on base and head: the test budgets the first pair but assumes the second fits. This supports the tracker’s pre-existing-defect account; no full-package green result is claimed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared filesystem, hashing, staging, and rclone seams reused.
   - **ARCH-PURE — pass:** admission and transitions are pure; IO remains in integration components.
   - **ARCH-PURPOSE — pass:** verified restore and catalog-loss round trip deliver M2’s purpose; operator acceptance is recorded.
   - **ARCH-MOCK — pass:** stateful remote tests share production boundaries; native conformance passed.
   - **ARCH-CONSTRAINTS — pass:** admitted payload bounds govern extraction and download limits.
   - **ARCH-SECURE — pass:** strict manifest/member validation, private output, and safe diagnostics.
   - **ARCH-ORDER — pass:** explicit transitions, controlled failure ordering, joined children, and publication uncertainty.
   - **ARCH-FUNERAL — pass:** owned staging is collected; published scratch output has explicit operator ownership.
   - **ARCH-EVOLVE — pass:** restore preserves manifest bytes and does not rewrite discovery metadata.

   Core-concept locations and PURE/INTEGRATION classifications match the implementation. Native publication was exercised on Darwin; Linux execution remains a qualification limitation.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"68953888-e5f0-4388-b5d1-4030542f4feb","repository":"github.com/xianxu/pair","issue_id":"000689","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/5f905c98784de45e","reviewed_head":"39ff9f9a6fd0283b6be10486f0aed3c890f4a3ba","slot":"pair:6","requested_at":"2026-10-19T16:33:21.490199Z"},"phase":"finished","pid":12963,"started_at":"2026-10-19T16:33:21.59519Z","finished_at":"2026-10-19T16:35:45.241461Z","session_id":"01a15502-afc2-79b3-a3c9-3e42e0eea028","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.736923458s
head: "39ff9f9a6fd0283b6be10486f0aed3c890f4a3ba"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
