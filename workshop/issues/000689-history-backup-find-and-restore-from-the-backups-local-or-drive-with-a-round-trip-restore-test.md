---
id: 000689
status: codecomplete
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-19
estimate_hours: 4.93
collaboration-mode: collaboration
card_mirror: '8885ff5d6c47262398c33dcf2d80c5ef52763891' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T23:32:01-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 5.25
---

# history backup: find and restore from the backups (local or Drive), with a round-trip restore test

## Problem

pair archive can create and upload backups (continuous pass, Drive sink #678) but has no way to read them back: no restore command and no index. The operator asked (2026-10-18) whether we have tested reading the backups back, for example fetching the logs from a few days selectively. We have not: verification today proves the bytes in Drive match the local ciphertext, not that a bundle decrypts, unpacks, and yields the files someone wants. Untested restore means unproven backup.

## Spec

Each bundle already has a plain JSON label (kind, from/to time range, file_count, raw and encrypted bytes, recipients, sha256), stored beside it locally and in Drive. Add: (1) `pair archive find --kind K --since T --until T` listing matching bundles from the labels (local catalog first, Drive listing when asked); (2) `pair archive restore ID... --to DIR [--from drive]` fetching the bundle (from Drive by its flat content name if not local), verifying its sha256 against the label, decrypting with the operator's GPG key via gpg-agent, and unpacking into DIR (never over live data); optional path filter inside the bundle. (3) A restore drill: an automated test with a throwaway GPG key round-trips create, upload (fake rclone), find, restore, and compares files byte for byte; plus a documented one-time live drill the operator runs against Drive (decryption needs their key).

**Label index across locations (operator, 2026-10-18).** The `.json` labels stay local too (they already sit beside
each bundle and in the catalog), and they are the index: `find` queries the local labels, never a remote. With more
than one destination (local, Drive, later others), each label records where its bundle is present and verified (the
#678 upload receipts already hold Drive presence). `pair archive sync-index [--from drive]` downloads only the labels
(small: about 1 KB each) from a destination and rebuilds the local index of which bundle is where, so a machine that
lost its local catalog can reconstruct it from Drive alone, then restore selectively.

## Done when

- find/restore exist; the round-trip test passes with a throwaway key.
- sync-index rebuilds the location index from Drive's labels alone (test: delete the local catalog in a fixture, rebuild, and find the same bundles).
- The operator restores one real day of Claude transcripts from Drive into a scratch folder and the files match the originals.

## Plan

Durable plan: [find, restore and label/location index](../plans/000689-find-restore-index-plan.md).

- [x] M1 — Preserve additive label/index fields; implement local-only find and atomic local/Drive index synchronization, including catalog-loss rebuild.
- [x] M2 — Implement verified fresh-destination restore, throwaway-key full round trip, and operator-owned real Claude-day Drive drill; qualify and land before retention.


## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* Calibration is stale, so this is provisional. The approved detailed plan earns the 0.2 design multiplier and 15% design buffer. Familiarity is 1.0: #678's Drive, filesystem and fake seams are already familiar.

Library check: encoding/json, archive/tar, existing safe filesystem/commitoutcome helpers and x/sys exclusive-rename bindings cover the novel modules' mechanics; use half the greenfield design midpoint (1.25/2), then the 0.2 spec discount = 0.125 per concern. Smaller extensions use 0.15*0.2=0.03. Drive integration uses design midpoint 2*0.2=0.4. Implementation values are already 40% of the v2 table (0.4 for smaller modules, 0.7–0.8 for greenfield modules, 1.0 for API integration, 0.4 per review). No second scale is applied. Total design 1.26 h, scaled implementation 3.48 h, yielding 1.26*1.15+3.48=4.929 h (rounded 4.93 h). Operator scheduling/idle time is excluded.

- compatible label codec/shared keys: smaller-go-module.
- index entities, merge and find: greenfield-go-module.
- atomic store and owned staging: greenfield-go-module.
- read-only catalog and local discovery: smaller-go-module.
- batched Drive label synchronization/stateful fake: api-integration.
- isolated streaming decoder: greenfield-go-module.
- bounded tar verifier/extractor: greenfield-go-module.
- restore orchestration/exclusive publication: greenfield-go-module.
- find CLI: smaller-go-module.
- sync-index CLI: smaller-go-module.
- restore CLI: smaller-go-module.
- docs and drill: atlas-docs.
- M1 review: milestone-review.
- M2 review: milestone-review.
- final close review: milestone-review.
- installed rclone conformance and operator drill preparation: real-api-discovery.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.03 impl=0.16
item: greenfield-go-module design=0.125 impl=0.28
item: greenfield-go-module design=0.125 impl=0.28
item: smaller-go-module design=0.03 impl=0.16
item: api-integration design=0.4 impl=0.4
item: greenfield-go-module design=0.125 impl=0.28
item: greenfield-go-module design=0.125 impl=0.32
item: greenfield-go-module design=0.125 impl=0.32
item: smaller-go-module design=0.03 impl=0.16
item: smaller-go-module design=0.03 impl=0.16
item: smaller-go-module design=0.03 impl=0.16
item: atlas-docs design=0.025 impl=0.08
item: milestone-review design=0.02 impl=0.16
item: milestone-review design=0.02 impl=0.16
item: milestone-review design=0.02 impl=0.16
item: real-api-discovery design=0 impl=0.24
design-buffer: 0.15
total: 4.93
```

## Log

### 2026-10-18


### 2026-10-18 — design checkpoint

- Claimed while unowned; start-plan refreshed the stale ariadne tooling, then was rerun as instructed to start this issue branch. No implementation edits or tests/builds initiated by this worker; the tool-owned catch-up rebuilt its own binaries.
- Read the entire Spec, including the local-only find/sync-index addition. Existing CatalogRecord requires a private Manifest, so public labels cannot reconstruct its authority. Uploaded labels are immutable and cannot gain location fields without invalidating #678 hashes.
- Hub bb8ab7c9-1c77-43ae-9d13-d2e28ff82dd7 approved separate index envelopes, observed/verified evidence, no ciphertext download during sync, complete refresh marking unavailable while retaining history, and no changes from failed/partial refreshes. The plan uses one atomically replaced snapshot, preserves unknown fields at every label/index nesting level, and distinguishes publication uncertainty from a confirmed failed observation.
- Restore always verifies staged ciphertext against the label before GPG/zstd and validates all tar members, including unselected files, before exclusive publication to a new scratch target. The real Drive/day/key drill remains the operator's. No retention or live-data import is included.
- Read-only component mapping reused existing safe paths, catalog/receipt/hash helpers, rclone runner/fake and isolated GPG fixture. Full design and initial batch limits are awaiting hub approval before change-code.

### 2026-10-19 — implementation admitted
- 2026-10-19: closed — M1 and M2 closed SHIP; local find and atomic label-only sync survive catalog loss; stateful remote restore, bounded extraction/download and diagnostic regressions pass with race checks; real throwaway-key GPG/zstd full and filtered restore and native rclone bounds pass on code 27019abd; operator Drive drill passed for 2026-10-16: 63 labels, 2 bundles, 339 files, 101518369 raw bytes, original SHA-256 agreement and stage absent. Corrected-code recorded suite 429890b0fa5edf870059aab33e54dade remains active and must pass before landing.; review verdict: SHIP
- 2026-10-19: closed M2 — BR2 fixed: label raw payload plus32MiB complete-stream bound; exact and one-byte-over normal/snapshot tests RED then GREEN. Resource class sweep ties existing remote transfer caps to selected sum/largest object, post-listing growth RED/GREEN and native exact-bound conformance pass. Member diagnostics quoted and CLI shared rowtext sanitizer regressed. Corrected-head27019abd real GPG/zstd full/filtered remote-only round trip plus native rclone controls passed2.371s with entries0/agents0/root removed; Drive acquisition race12.653s, broad restore/CLI races and guards pass. BR3 disputed with measured pinned-host qualification, not a fixture change. Operator real Drive drill accepted:63 labels,2 bundles/2127158 ciphertext bytes,339 files/101518369 raw bytes, SHA256 agrees, stage absent and decrypted scratch removed. Prior recorded suite16m21s passed; fresh corrected-code suite active and must pass before landing.; review verdict: SHIP
- 2026-10-19: closed M1 — BR-1 approved refinement: complete destination markers retain empty inventory and folder/time precedence; old receipts remain unavailable, newer same-binding receipt promotes only its own location. Four public sequence tests confirmed RED then GREEN; strict optional schema/legacy older-envelope/count bounds pass; focused M1 run59.389s and races23.248s pass, staged repolint8.229s and codec fuzz599 executions pass. New recorded suite queued on pushed 247f8fee6; no suite pass/landing claimed. M2/operator drill remain outstanding. Another index evidence hole requires hub simplification before patch.; review verdict: SHIP

- Plan quality CLEAN after PQ-1 disposition; change-code passed with derived estimate 4.93 h. Estimate review INFO: cross-component fixture/race/Linux/full-suite work and active operator qualification are tracked explicitly in this Log; idle operator scheduling is excluded. The library discount covers existing mechanics; remaining ownership/evidence/publication contracts stay custom work.
- M1 RED/GREEN: compatible label/key codecs, index refresh/history/unknown-field codecs, UTC selection, read-only catalog, local catalog-loss discovery, ownership-stamped staging and atomic index store. Focused catalog/store tests passed with race; codec fuzz passed 51,035 executions. M1 is not closed yet; CLI/Drive integration and normal guards remain.
- Installed rclone v1.75.2 refused files-from-raw with max-size. Switched the new exact-name download seam to tested rooted filter rules, preserving actual max-size/hard max-transfer caps. Conformance downloaded the selected 5-byte label, skipped a 9-byte object under a 4-byte max-size, and refused a 4 MiB transfer under a 1 MiB cap without retained partial output. Upload transport unchanged.
- Independent existing fixture defect filed unclaimed as pair#696. Fractional mtimes make pair byte sizes unequal, but TestDriveMirrorBatchBudgetDefersCompletePairs budgets only pair 0. A controlled .1Z/.123456789Z case produces totals 3481/4009 and reproduces second-pass deferral with byte-identical legacy labels. No unrelated fix/assertion weakening. Hub message 62c9378c-5ebc-4036-a3a0-1b40c6310719 submitted after an initial pre-admission unavailable refusal.

### 2026-10-19 — M1 qualification checkpoint

- CLI create/find/local-sync fixture passed with public JSON output and no private filename/root exposure; find does not publish an index. An invalid optional Drive locator no longer blocks local discovery (RED invalid-remote refusal, then GREEN focused view/sync tests).
- Counted local N=8/16 work: two shard passes and 2N record visits, one catalog directory enumeration and N label reads. Catalog-loss fallback performs three kind/year/month enumerations and 2N label reads. A repeated-lookup control increases measured counters. Combined catalog/local/view/sync races passed (20.261 s).
- Drive label sync N=8/16 uses <=4 subprocess calls, two flat listings and one label batch, zero ciphertext downloads/checksum fallback. Partial/conflicting observations preserve the prior snapshot. Index codec fuzz passed 1,194 executions; broader M1 focused race passed (50.155 s).
- Full changed-package run: archivecmd and archivetest passed; historyarchive failed only the previously filed #696 fixture at the same deferred-retry boundary. Artifact coverage initially found three stale ignored runtime mirror files absent from current source; regeneration through runtimebundlegen cleared it and artifactpath passed (13.461 s). Repository lint passed (11.748 s). No assertion/deadline change, retry qualification or broad-suite success is claimed.
- Preparing the pushed checkpoint and recorded full suite. M1 mandatory review and M2 implementation/live acceptance remain outstanding.


### 2026-10-19 — M2 implementation checkpoint

- M1 second boundary review returned SHIP and closed M1; no further index-evidence hole was found. The approved third-refinement stop rule remains binding.
- Added pure RestorePlan/NextRestore, streaming CommandDecoder, strict manifest/member extraction, whole-batch ciphertext acquisition, and exclusive Darwin/Linux publication. The coordinator stages and hashes all ciphertext before decoding, verifies excluded members too, joins decoders before publication, and classifies post-rename parent-sync failure as publication uncertainty. No retention/live import/remote repair added.
- Local coordinator tests prove complete private output/source preservation, corrupt-input refusal before decoder, late integrity refusal, concurrent target preservation, uncertain sync reporting, foreign-stage preservation and no-match refusal. A shared sync-budget regression was RED (sibling trees reused the remaining budget), then GREEN after one request-wide counter. Focused restore/model/decoder/extractor/publication tests passed (4.146 s); focused race passed (63.366 s).
- Drive restore stateful tests passed (23.590 s) and race passed (36.830 s). N=8/16 each used exactly three runner invocations (resolve, one flat listing, one ciphertext batch), zero label/checksum downloads, and ciphertext bytes equal the selected labels. Corrupt bytes with plausible advertised SHA, partial batch, cancellation and duplicate names refuse before decoder/publication; cleanup and unchanged index/no manufactured local authority are checked.
- Actual isolated GPG/zstd conformance passed (10.640 s combined): throwaway RSA key, real create, fake Drive upload, removal of every local archive copy/metadata, label-only sync, local find, full and exact-path production restore. Unicode/binary/empty fixture bytes and hashes matched independent originals; decoder ran twice, remote/source bytes were unchanged, and owned stages/key agent were cleaned. Default opt-in tests skip correctly. No operator keys, real Drive or :0 were used.
- Restore CLI help/parsing/config/default validation tests passed (0.184 s); artifactpath (21.155 s), archivecmd (1.311 s), and neutral helper compilation passed. Linux historyarchive test binary cross-compiled successfully and its owned temporary output was removed. Initial lint command used a nonexistent cmd/internal/repolint path; corrected to internal/repolint (passed 10.840 s), no source change for that command error.
- After staging all new sources, tracked-file guards passed: repolint 7.677 s, artifactpath 6.763 s, archivecmd 0.447 s.
- Cancelled only this checkout's obsolete queued M1 suite on 247f8fee after M2 implementation; it never supplied a suite pass. Another checkout's run was retained. Final pushed M2 recorded suite, guards, operator real Claude-day Drive drill and M2/issue close remain outstanding. Prior TTY capture is unavailable; branch/code/review/card evidence is the recovery source.

- M2 candidate committed/pushed as aa87c202e7973df5b2313393fe0727d3fd4375c3; PR #450 opened. Private pair-go binary under /private/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-689-candidate.rpHTZy reports that vcs.revision and vcs.modified=false; help runs. Server local-gates guard is green. No :0 stage/build/restart.
- Final-code recorded suite attempt 738172d262add15b8c7d3683c9f7deea started through sdlc local-check and is queued on aa87c202. No suite pass is claimed. Its log is created when it acquires capacity; the initial queued state has only an owner record.
- Operator drill commands and independent SHA-256/original comparison are in /tmp/pair-689-operator-drill.md. Hub send 5fe2daa6-c03e-46d8-86ab-50633ff2368e is submitted. Verified hub receipt 3d2fd896-3a8c-469b-9f9d-f61abaf83bdc arranges the morning drill for 2026-10-16, instructs letting the recorded suite finish and holding landing until acceptance. M2 boundary and issue close remain unrun pending that evidence; no real Drive/operator key access by this worker.

- Recorded suite 738172d262add15b8c7d3683c9f7deea acquired after 274 s. Historyarchive (199.152 s), archivecmd, archivetest, artifactpath and couchcore passed, but termcmd's TestNoDeclarationCarriesTwoStackedGodocs found a branch-owned misplaced opener: RestoreValidateOutput's paragraph was attached to RestoreEvent. Reproduced with the existing guard (RED), moved the event-specific sentence onto its enum constant and documented RestoreEvent itself, then the unchanged guard passed (0.381 s). This changes comments only, no guard/assertion/behavior change. The original recorded run is retained through completion for other failures; its failure cannot qualify landing. A fresh recorded suite is required on the corrected pushed head.

- First full recorded suite finished exit 2 in 14m42s, with exactly one failure: termcmd.TestNoDeclarationCarriesTwoStackedGodocs. All other packages/checks completed, including historyarchive/archivetest/wrapcmd/repolint. Saved failure log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/738172d262add15b8c7d3683c9f7deea.log. This is a fixed branch comment defect, not a flake/unchanged retry.
- Corrected code head b513994f9a311d27bf88280a7cf89d1c8d4d35f7 is pushed in PR #450. The private candidate at the same path was rebuilt at that revision with vcs.modified=false; operator commands/behavior are unchanged. Hub update 4cbe9d92-5303-49b0-92d5-584851adff3d is submitted and the private drill note is updated.
- Replacement recorded suite is active/queued on b513994f: attempt aa8fba55ca19ee5947649e00c90b25d0; exec session 48458. Detached worktree /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/sdlc-local-2140393263; future run log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/aa8fba55ca19ee5947649e00c90b25d0/run.log. It is first waiting behind the active Ariadne suite; leave this run active, do not duplicate/cancel it. No corrected full-suite pass is yet claimed. Read its final result/status before M2 close; a code fix needs its own recorded pass. Landing remains held for the operator's morning 2026-10-16 real Drive drill plus M2/issue reviews.

### 2026-10-19 — M2 automated qualification passed; operator gate held

- Corrected recorded suite aa8fba55ca19ee5947649e00c90b25d0 PASSED, exit 0 in 16m21s after 685 s queue, and posted sdlc-local/suite on b513994f9a311d27bf88280a7cf89d1c8d4d35f7. Saved log: /Users/xianxu/workspace/pair/.git/sdlc/local-checks/b513994f9a311d27bf88280a7cf89d1c8d4d35f7/suite.log; log SHA-256 prefix e816b16668e3. The complete command ran all shell/editor/Python and Go checks; termcmd 3.101 s, historyarchive 291.081 s, couchcore 517.033 s, and repolint 6.913 s passed. No flake rerun/guard waiver/assertion change. The earlier 14m42s failure and its comment-only fix are retained above.
- The installed-rclone local-backend download conformance from M1 covers the unchanged DownloadBatch seam. M2's real throwaway-key create/upload/catalog-loss/sync/find/full-and-filtered-restore drill, counted remote acquisition tests, targeted races, source/lint guards and Linux cross-compilation are complete. Linux publication was cross-compiled, not executed on a Linux host.
- No heavy run remains owned by pair:6. Preserve PR #450 and the clean private candidate b513994f at the existing path. The operator's morning 2026-10-16 real Drive restore and independent original/hash comparison remain required. Do not land, close M2 or close the issue before that accepted evidence. On acceptance: append the exact result, close M2 through the binary-owned review, close the issue with measured actuals, push evidence, and enter normal sdlc merge. A further index-evidence review hole must go to the hub before patching, per b322a666.

### 2026-10-19 — operator real Drive drill accepted

- Verified submitted hub receipt d2041a34-aed7-4c2c-b759-74b2411916a4 reports PASSED for the arranged 2026-10-16 Claude-day drill: Drive index synchronized 63 labels; selected 2 bundles totaling 2,127,158 ciphertext bytes; restored 339 files totaling 101,518,369 raw bytes; restored and retained-original SHA-256 agree; owned stage absent. This satisfies the real-environment acceptance gate on the private candidate. The worker did not access the real Drive/key or decrypted contents.
- The operator-owned scratch was /private/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-689-drive-drill.2awz9t. Hub performed cleanup because it contained decrypted transcripts; a later read-only path-existence check returned false. No plaintext payload was copied into this Log.
- Hub authorizes landing through the remaining normal gates. After landing, file an unclaimed follow-up for pair archive drill: select a random archived day, restore from Drive, verify labels and retained originals, remove owned scratch, and expose it to the coming Couch schedule. Periodic use (for example monthly) requires the operator present because GPG needs their passphrase; no unattended key/passphrase storage implied. This follow-up is separate from #689.
- SDLC workaround until ariadne#461 lands: bare slot sdlc for start-plan only; every other command uses /Users/xianxu/workspace/ariadne/bin/sdlc per verified hub correction 1550b8f0-e311-4fcf-9352-5581e8275b6e. No existing merge runner was active to replace.

### 2026-10-19 — M2 review corrections

- M2 review REWORK on pinned 2a47ec08c1ca739d5ba5ef73faba23517785be42: BR-2 Critical correctly found that the decompressed stream budget followed the kind's maximum instead of admitted label.RawBytes plus the promised 32 MiB overhead. BR-3 Important/unverifiable hit a long-review-TMPDIR GPG socket limit before restore. Neither concerns index evidence; no further index model refinement is made.
- BR-2 RED: exact payload-plus-overhead succeeded, but the one-byte-over regressions incorrectly succeeded for both ops-documents and codex-database. Fixed the reader budget to label.RawBytes + MaxRestoreTarOverhead and corrected atlas wording. The existing hidden-PAX coverage remains; the new stimulus is small and streams rather than allocating a worst-case kind-sized buffer.
- Independent diagnostic RED: virtual-overlay probes and then retained regressions reproduced raw ESC/BEL/C1/DEL emission from the corrupted manifest member and CLI error seam. Fixed the member error with quoted identity and reuse of rowtext.Sanitize at archivecmd's shared diagnostic boundary. No escape-parser copy or filename policy change; all archive command errors receive the same terminal-safety owner.
- BR-3 measured on the exact pinned source before these code edits: PAIR_ARCHIVE_LIVE=1 with an isolated canonical /tmp/pair-689-live.XXXXXX TMPDIR, running TestCodecLiveRoundTrip and TestRestoreLiveRemoteOnlyRoundTrip. Both PASSED, package18.210s, codec8.54s, remote-only full/filtered restore9.48s. Post-test entries0 and matching gpg-agent processes0, empty root removed and absence confirmed. Output: /tmp/pair-689-br3-live.log. Disputed BR-3 through the maintained binary with this host evidence; next review must explicitly answer it. No personal keyring, deadline, fixture assertion or product change was used to pass.
- The operator's real Drive acceptance remains valid. Targeted/race, final-head live proof, recorded suite and M2 re-review remain due before close/landing after these corrections.

- Resource-bound class sweep also made Drive download caps follow the admitted remote subset (sum and largest selected ciphertext) instead of the phase's maximum ceilings. A stateful post-listing growth regression was RED (3,622 bytes downloaded despite the smaller admitted object), then GREEN with zero transferred growth bytes and no decoder/publication. No extra listing/query or new restore mode.
- Installed-rclone exact-bound conformance PASSED (0.553 s) on the unchanged DownloadBatch seam with the tighter caller bounds: exact single and multi-object totals succeed; an object beyond max-size is omitted; an over-total transfer fails with no retained object; all owned filter files are removed. This is local-backend fixture IO, not real Drive. Focused correction tests passed (archivecmd0.217 s, extractor0.500 s); earlier broad restore/CLI races passed (38.348 s/1.210 s). Final changed-head live proof, focused acquisition race, guard run, recorded suite and M2 re-review follow this checkpoint.

- Corrected final code head 27019abd1964197aad880dc7ce8f3a38af95e9a0 is pushed. Native qualification on that exact head passed (2.371 s combined): real codec1.00 s, real remote-only full/filtered restore1.06 s, installed-rclone exact-bound controls0.22 s. The isolated short-root test left entries0, matching gpg-agents0, and the empty root was removed. Output: /tmp/pair-689-final-live.log. This also rechecks valid decoding under the corrected payload/download limits.
- Final focused Drive acquisition race passed12.653 s. Earlier broad restore/extractor/CLI race passed38.348 s/1.210 s; tracked lint, artifact inventories and godoc guard on corrected source passed2.151 s/1.413 s/0.242 s. No deadline/assertion waiver. The corrected-head recorded suite is now being started through the maintained SDLC binary; its pass and M2 re-review remain due.
- Verified hub dispatch 100d7776-57b9-4c32-8f8b-a71a6f2df739 queues #707 only after #689 lands. Keep that separate from this landing and the requested unclaimed periodic-drill follow-up.

### 2026-10-19 — landing integration

The corrected-code recorded suite passed on 27019abd1964197aad880dc7ce8f3a38af95e9a0 in 11m48s (attempt 429890b0fa5edf870059aab33e54dade; log SHA-256 prefix 09745fbca330). M2 and whole-issue reviews both returned SHIP, with BR-2 addressed and BR-3 withdrawn. The normal landing turn then required integration of origin/main; the runner preserved the branch and named conflicts in README.md and archivecmd/run.go. The manual resolution combines the continuous --json help with find/sync-index/restore, and preserves main's compression levels, coherent SQLite snapshot/freshness notes and Drive file-count warning beside restore documentation. No index-evidence model change.

After integration, real GPG/zstd codec conformance (including levels 9/19/20/22), installed-rclone exact download bounds, and full/filtered remote-only restore passed in 9.905s with a throwaway key and isolated short TMPDIR. Original/restored bytes agree; source and fake-remote bytes are unchanged. The private root had zero entries and was removed. No operator key, plaintext or real Drive access was used. The integrated recorded full suite and the required narrow fix review remain outstanding before retrying merge.

Targeted integrated archive CLI tests passed (1.262s), restore/extraction/discovery/catalog tests passed (72.640s), and the tracked lint checks passed (4.650s). The artifactpath package compiled under that selection; its full checks remain part of the recorded suite.

### 2026-10-19 — integrated suite failure classified; fresh combined-head qualification

Integrated suite d963805085758c53cfbbab3363ad6432 failed on f2333e342cc325ded2344fe69dd0d7309224dca7 after 26m12s (log SHA-256 prefix 2f2ab4699a85): TestSpawnComposesProductionPairRegistrationBoundary/fresh-slot reported that its fake terminal was not listed live, spent 1172.59s in the launch path, then failed to join its spawn goroutine during cleanup. couchcore timed out at the unchanged 20m package limit, with many other tests still pending at t.Parallel. Every other package passed, including historyarchive (265.311s). No full-suite success is claimed.

The unchanged same-head targeted composed test passed in 6.666s (ordinary 2.94s, fresh-slot 3.44s; original 20m timeout and fake Zellij retained). The hub split and filed #712; the test is registered there with the fail/pass evidence. Its targeted pass proves intermittency, not completion of the other couchcore tests. The hub explicitly requires fresh full-suite coverage and forbids substituting a one-test rerun for a timed-out package.

The next merge turn integrated newer main, then refused publish replay because archivecmd/run.go's prior conflict resolution remains unreviewed; it restored the branch before push. Per the named remedy and hub approval, manually integrated the landed #698/#708 code without further conflicts. The resolved archive help and README surface are unchanged from the prior resolution. A recorded full suite on this combined head must pass before the narrow fixed-then-ship review and merge retry. #706 replaces parked #707 in the queue; the operator-present restore-drill follow-up is still to be filed unclaimed after #689 lands.

### 2026-10-19 — TestScratch substrate failure; qualification held

The fresh combined-head recorded suite de4cedafdeb5bff358280f93fa6c6523 on 65a7d797a6a43123051704208d7aaa053f84478e acquired capacity after 36m52s and failed in 9m38s (log SHA-256 prefix 4c134300188d). Worktree and temp roots were on /Volumes/TestScratch; diskutil reports HFS+ (Mac OS Extended). All script/editor phases passed. Go failures were:

- couchcmd.TestMessageAuthorityUnchangedFilesDecodeZeroBytes and TestWorkbenchPerformanceMessageUnchangedEvidence: same-size index mutation left decode/read counters at zero.
- historyarchive.TestDriveMirrorBatchBudgetDefersCompletePairs: both pair sizes were 3477 despite the fixture's different fractional mtimes.
- launcher.TestSessionNameIndexCacheUnchangedAndMutations: same-size rewrite reused the prior index.
- sessioninventory.TestFingerprintFileInfoDetectsSameSizeRewrite and TestClaudeExactChangedContentRefuses: rewritten content reused evidence/resolved when refusal was expected.
- workbenchperftest.TestFixturePaddingPhysicalWriteBudget: 10502144 bytes versus 2097152 cap.
- wrapcmd.TestWrapSpawnsOneWatcherForItsLaunch: agent-pid mtime rounded to a whole second before the process start.

Package write caps also failed: couchcmd 821989376/100663296; couchcore 1909970432/134217728; couchtty 665907712/67108864; sessionsync 153626112/100663296; workbenchperftest 133996544/67108864. couchcore's tests themselves passed; there was no #712 hang on this run. The suite is an unattributable failure, not a success or a known-flake pass.

A bounded unchanged-head A/B test establishes filesystem-dependent metadata failure: TestFingerprintFileInfoDetectsSameSizeRewrite passed with TMPDIR=/private/tmp (0.097s) and failed with TMPDIR=/Volumes/TestScratch/tmp (0.077s, same-size rewrite with restored mtime reused evidence). No source, assertion or deadline changed. Fractional timestamp loss and clone fallback appear implicated in the broader set; their complete root causes are not claimed. The hub received every failure and the A/B proof. Further qualification is held for its substrate/root decision; no mount, write budget, identity guard or assertion is changed under #689. The full-suite pass, narrow review, landing, unclaimed drill follow-up and subsequent #706 remain outstanding.

### 2026-10-19 — APFS recovery authorized; waiting for operator recreation

Hub receipt bf5f4f78-4281-4bbf-a560-8d4e8c74399a identifies the RAM disk's HFS+ whole-second timestamps and lack of clonefile as the substrate cause. The hub reports changing the ops creation script to APFS and probing nanosecond timestamps plus clonefile; these are hub-owned measurements, not worker filesystem changes. It filed ariadne#474 to refuse non-APFS scratch storage. The operator will recreate the disk after the currently active suite finishes. The hub explicitly authorizes one fresh #689 combined-head recorded suite after its readiness message. Until that message, start no new qualification on the old scratch disk. No new flakes are registered for the substrate failures, and all existing guards remain intact.

### 2026-10-19 — normal-disk fallback authorized

Hub receipt ef3f857b-fae0-401b-ac40-4bd4c560074e confirms ariadne#474 landed and ariadne:0 was rebuilt. The maintained sdlc now refuses HFS+ scratch storage and selects the normal temp folder with an explanatory note until the operator recreates APFS storage. The hub explicitly authorizes rerunning this combined-head suite now on that fallback; disk recreation is no longer a precondition for this run. No source, budget, identity check or assertion changes. The filesystem-related failures above remain failed evidence, not registered flakes.

### 2026-10-19 — fallback runner blocked before tests

The authorized fallback run 402744d9523c334429d959ef764a84af pins d344514ce196074da7b8a92e8804bed78d5b6e52 and selected the normal-temp worktree /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/sdlc-local-2593999494. It acquired a suite slot after 388s, then emitted no test-phase output; its run.log remained empty. Read-only process inspection identified the owned runner as PID 92467 in UNs state, with watch cat 95136 still live. A one-second sample at /tmp/pair-689-sdlc-wait.sample.txt shows a thread blocked in the kernel stat call. No tests have run or failed in this attempt; no result or filesystem path being statted is inferred.

Both the reserved hub address and exact ops:0 reporting attempt were explicitly refused by Couch with “message terminal is not positively listed,” without a receipt. A read-only actors query still reported both ops:0 and pair:6 as Known. No alternative messaging channel or slot restart was attempted. To prevent a delayed test launch and release only this checkout's check, requested maintained sdlc local-check cancel --name suite. That cancellation remains pending while the original runner is blocked; it is not reported as complete. No PID-based kill, deadline, assertion, machine setting or source change. Await host filesystem recovery and confirmed cancellation before another check.

## Revisions

- 2026-10-18: clarified the later sync-index Spec supersedes remote queries from find. Recorded the hub-approved label/location evidence and atomic refresh contract; original Done-when obligations are retained without reduction. Added the durable plan with M1/M2 review boundaries.

- One-pass read-only plan editing aid identified two gaps before code: Drive-only source-root overlap is knowable only after decrypting the manifest, and a staging pathname/lock alone does not prove ownership of existing contents. The durable plan now requires the post-manifest check and an ownership-stamped staging protocol with refusal on unknown contents; tests preserve a preexisting operator staging directory and a target below a recovered source root. Created files/directories also share a whole-request bound so small-file/directory growth cannot bypass byte admission.

- Hub d9acf9a3-51ac-4eda-9dc9-1a70065e04ac approved the full plan as written and instructed change-code. They will arrange the operator real-day Drive restore when M2 is ready; that acceptance remains mandatory before landing.

- 2026-10-18 PQ-1 (function-level-test-contract): consolidated the class of repeated unit-case inventories into named-function adversarial strategies and mechanical guards. Kept catalog-loss and real-key/operator acceptance drills and all approved invariants; no implementation begun.

- 2026-10-19 BR-1 / complete-observation-precedence: M1 review REWORK (Critical), not finalized. Confirmed code loses complete empty destination observations and can re-project an unseen historical receipt as available. Proposed additive bounded destination observation markers and uniform receipt precedence in the durable plan; awaiting hub approval before production changes. Cancelled only this checkout's obsolete queued suite on 12d4787e; other checkout runs were retained. No full suite ran or pass was recorded. M2 remains unstarted.

- Hub b322a666-7176-43fc-baa1-2bd67c49571c approved BR-1 refinement and tests. Explicit stop rule: a further index-evidence hole goes back to the hub before code, for simplification. Implement this approved complete-observation precedence only; M1 remains unclosed.

- 2026-10-19 BR-1 fix: added optional unknown-preserving complete destination markers (max 1,024) with pinned folder/time, published by ApplyIndexRefresh even for empty observations. Stale empty refreshes refuse. Receipt composition now uses the marker plus per-location observation time; newer same-binding receipts affect only their own location, and contradictory available locations are rejected on decode. Legacy snapshots have no markers; an older-envelope round trip preserves the additive marker unchanged.
- BR-1 RED/GREEN: all four public sequence regressions first failed against M1, then passed after integration (empty initial sync/find/local-sync, unseen old folder receipt, newer same-binding own-ID promotion, partial refresh unchanged). Pure marker/optional schema/legacy/count-bound tests passed (0.218 s). M1 model/local/find/codec/Drive focused run passed (59.389 s); receipt/index/view/sync races passed (23.248 s). archivecmd/archivetest/repolint passed. No additional index-evidence design is added; a new review hole must go to the hub before another patch.
