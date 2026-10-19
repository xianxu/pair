---
id: 000678
status: codecomplete
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours: 2.48
collaboration-mode: collaboration
card_mirror: '172a561d38e1de6b171708f41f99edb79555567b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T17:51:28-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 1.90
---

# history backup: upload encrypted bundles to Google Drive via rclone (verified), keeping local copies

## Problem

pair's continuous history backup (historyarchive) only has a LocalSink: bundles land in ~/Backups/pair-history on the same SSD (758 MB, 2026-10-18). The operator chose Google Drive for the off-host copy: account xian@xldigit.com, folder backups/pair-history, via rclone with drive.file scope (the operator configures the rclone remote `pair-backup` themselves; checklist in ops qa 2026-10-18-16-history-backup-review.side/02-drive-backup-checklist.md).

## Spec

Add a Drive sink to historyarchive that shells out to rclone (remote and folder from continuous.json config, default pair-backup:backups/pair-history). Upload the already GPG-encrypted .tar.zst.gpg bundle and its .json manifest; verify by checksum (rclone check or hash compare) before recording the upload in the catalog. Local copies stay (deletion is a later retention step). Missing rclone or remote: the pass reports the sink unavailable visibly and keeps backing up locally; never fails the local backup. The first run uploads every catalog bundle not yet in Drive. A stateful fake rclone behind the same seam for tests (ARCH-MOCK), plus a live conformance step the operator runs once.

## Done when

- With the remote configured, one continuous pass uploads all bundles, each verified and recorded in the catalog; a rerun uploads nothing new.
- Missing rclone or remote leaves the local backup intact with a clear message.
- The operator sees the bundles in Drive (live check).

## Plan

Durable design: [000678-drive-history-backup-plan.md](../plans/000678-drive-history-backup-plan.md). Full flow; hub approved before change-code (delivery f7e7fe62-5f41-4133-ba43-4296e82fd885).

- [x] Define destination/receipt identities and pure upload transitions; test configuration defaults and remote-only failures.
- [x] Implement the bounded rclone seam with a stateful fake and disposable installed-rclone local conformance.
- [x] Integrate verified backlog replication after local completion, immutable upload receipts, private staging and counted N/2N recovery tests.
- [x] Document configuration/recovery and obtain operator-run Drive conformance. Recorded suite, close review and landing remain binary-owned workflow steps tracked in the Log.

## Log

### 2026-10-18
- 2026-10-18: closed — Operator live acceptance: pass1 56 verified plus2 pending; pass2 7 verified (2 retries plus5 new),56 already recorded,0 pending,0 downloads. Final batch tests enforce <=4 rclone processes at N8/16, <=5 with downloaded hash batch; partial progress, 4GiB completepair cap, staging cleanup and bounded diagnostics covered. BR-1 capacity regression RED6.496s then GREEN7.863s; broader Drive/model/fake passed14.780s/0.136s; artifactpath passed9.763s. Corrected-head recorded suite queued on7674d3a2 and must pass before merge. Earlier fullrace and installedrclone conformance passed; focused capacity race pending. README/atlas updated. TL owns history-backup project, so --no-project avoids worker edits.; review verdict: SHIP


- Recovery evidence: requested previous terminal capture returned unavailable. Git and sdlc state show #670 already landed and #678 claimed/started on its own branch; no implementation edits. Historical screen instructions are context, not fresh permission. Verified the original #678 dispatch receipts before continuing its existing planning work.
- Read-only map: existing local publication independently verifies ciphertext and public BundleLabel before CatalogRecord append. Continuous passes with no fresh sources still need a full catalog backlog phase. Catalog identity is bundle ID alone and its index treats every record as local; proposed remote receipts therefore live in a separately named upload catalog rather than changing existing record authority (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).
- Plan checkpoint: workshop/plans/000678-drive-history-backup-plan.md records the full proposed flow, explicit remote/local visible states, uncertainty/cancellation recovery, stateful FakeRclone seam, N=8/16 counts, lifecycle and operating limits. Separate upload catalog and limits await hub approval; no change-code or production edit yet.
- Discovery: installed rclone v1.75.2 supports lsjson hash selection and hashsum --download. Drive may omit SHA256 on some objects, so explicit downloaded hashing is the fallback; never accept copy command success or size alone. Only version/help and official docs were consulted; no Drive request, credential/config read, upload, operator-session change or restart occurred.

## Revisions

- 2026-10-18: expanded the initial empty Plan into a full-flow design after inspecting local catalog identity and publication. Proposed a destination-specific upload catalog, keeping the existing local schema and authority unchanged; retained the original Spec and Done when for hub review.

- 2026-10-18 approval: canonical submitted hub receipt f7e7fe62-5f41-4133-ba43-4296e82fd885 approves the plan, receipt family and bounds. Added per-bundle backend/downloaded checksum-path reporting; hub arranges live Drive pass after fake-backed readiness. Proceeding to change-code.

- Plan-quality round 1 blocked on PQ-1 (stat cannot prove unique Drive name identity) and PQ-2 (duplicated prose test inventories). Verified against official rclone lsjson/Drive contracts. Revised plan names bounded initial/final inventories, pinned destination ID, deferred receipt publication and identity-fenced downloaded hashes; compressed test prose. This is a new observation mechanism, so hub approval is requested before another change-code attempt. No code was edited.

- 2026-10-18 hub direction 892e9607-9074-46ed-9d53-a2e5de49637d: simplified PQ-1 disposition to flat content-addressed remote names, one resolved/pinned folder ID and two flat hash listings. No recursive inventories, spooling or depth limits. Receipts require unique names with matching size/hash; duplicates/mismatches are reported and never repaired. Backend SHA256/MD5 verification paths are logged; only absent backend hashes cause downloaded SHA256. Existing requirements are met; proceeding with change-code rerun.



### Implementation checkpoint — 2026-10-18

- change-code passed full flow: plan-quality CLEAN, PQ-1/PQ-2 addressed; estimate-quality INFO (optimistic integration/conformance allowances). Derived card estimate 2.48h using the named calibration, not a guessed total.
- Implemented pure destination/receipt/state contracts, strict JSON decoder, bounded rclone runner and stateful fake, flat before/after inventories, verified encrypted staging, immutable separate receipts, and post-local continuous replication/CLI reporting. New production sources are registered in artifactpath. No operator environment/config or Google remote has been touched.
- RED evidence: configuration rejected unknown drive_remote before its implementation; mirror/continuous wiring symbols absent; CLI omitted Drive output; receipt adoption returned found=true with verify=0. GREEN evidence: task model/adapter tests; mirror 8/16 counted cycles and zero-copy rerun; partial/uncertain upload recovery, final duplicate refusal, independent local success/marker, and receipt durability re-verification. Candidate projection test counts exactly one Catalog.Each scan at both 8 and 16.
- Installed rclone v1.75.2 local-backend conformance found --immutable alone overwrote conflicting bytes. Added --ignore-existing to the same operation; isolated conformance proves preserved bytes and correct flat listing/downloaded checksum. Per-bundle verification method and aggregate downloaded-checksum counts are reported. This corrects the existing no-overwrite contract without a new mechanism/query.
- Latest complete targeted command with PAIR_RCLONE_LOCAL_CONFORMANCE=1 passed historyarchive (82.049s), archivetest (0.225s), archivecmd (0.907s). Subsequent new projection/adoption checksum tests passed (25.873s). Guard packages passed internal/repolint (37.658s), artifactpath (21.534s), testpair (16.762s); these will be rechecked with newly tracked files. The first broad invocation had an incorrect cmd/internal/repolint path; reran the actual internal/repolint package, no bypass.
- Race qualification is still running; stronger in-flight child cancellation handshake is being added. README/atlas describe configuration, honest status, verification paths and operator live steps. Live Drive pass, recorded full suite and close review remain outstanding; hub arranges live check once candidate ready.

- Additional qualification: race tests passed historyarchive (47.417s), archivetest (1.182s), archivecmd (1.181s). Tracked-file guard rerun passed internal/repolint (20.563s), artifactpath (19.536s), testpair (26.256s). In-flight exec cancellation uses authenticated readiness and owned-process collection (race test 1.221s). Semantic JSON/hash alias regressions were RED then GREEN; latest focused race check passed historyarchive (2.471s) and archivecmd (1.186s).
- Estimate advisory acknowledged: adapter allowance includes its fake and local conformance; integration allowance includes projection/staging/receipt/CLI wiring and tests. Active verification and landing are included in the review/integration work; idle machine queue time is excluded. The calibration is provisional and may underpredict this integration-heavy scope; actuals will remain measured.

### Candidate and outstanding acceptance — 2026-10-18

- PR: https://github.com/xianxu/pair/pull/444 . Qualified code/build revision from Git/go version: 1ed7a6e86e1e4aa2be0e97d67a8ad03ce4538d5f, vcs.modified=false. Private binary: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-678-live.OOCygr/pair . Its archive --help smoke passed. Build entrypoint is cmd/pair-go (initial cmd/pair path was incorrect and was corrected before producing a binary).
- Hub readiness message b9af8bee-7b09-48fb-ab91-2d13ecb5e580 is canonically submitted. Hub arranges operator command: that exact private binary archive continuous, then rerun and inspect Drive names/local copies. No live Google call, :0 stage, restart, relaunch, personal rclone config read or credential operation by this worker.
- Recorded suite launched on pushed source head 1ed7a6e86e1e4aa2be0e97d67a8ad03ce4538d5f, currently waiting in machine capacity line (position 3 when last observed). Running exec session 19377; inflight log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/b9fa9937cb7f8be6385ed9881047a05e/run.log . Keep this one run; do not duplicate or kill another slot's process. Initial server checks: merge-check and local-gates guard passed; conformance pending.
- Still owed: full recorded suite result, actual operator Drive pass/rerun evidence, binary-owned close review and landing. No cause to change source or repeat passing targeted checks while these are pending. A newly identified defect would require its own focused regression and final-head suite as usual.

### Live-driven transport revision — 2026-10-18

- Canonical hub receipts b4588b23-0c2c-4250-bf98-508602961c6a and 89180927-8227-4f02-8f4f-a093e1a771d2 report operator pass works but 116 serial copyto invocations for 58 bundles / 790 MB is slow on gigabit. Operator uses explicit PAIR_DATA_DIR=$HOME/.local/share/pair from their slot. This is real performance evidence, not final batch acceptance.
- Full replan: one copy batch (--transfers 8), same two flat listings, one filtered downloaded-hash batch if needed, partial-batch per-pair receipts. Flat staging reason is local kind/year/month paths versus flat remote names; use descriptor-checked hard links normally, EXDEV verified copy fallback. Apply hub's 4 GiB logical complete-pair cap, defer overflow visibly, and clean staging on every outcome. Fake counts actual subprocess invocations constant at N/2N, not merely per-object transfers.
- Current running suite on the serial source remains evidence for that earlier source only, not batch qualification. No second suite started. The revised plan is committed and change-code is rerun before production changes. New live candidate/acceptance is owed after batching.

### Live acceptance and follow-up — 2026-10-18

- Operator pass 1 (hub 64519fc8-111a-425a-8607-423a59680640; log /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/drive-pass1.log): 56 verified pairs, 0 already recorded, 2 pending, all successful verification via Drive SHA256, 0 downloads. The two exit-1 IDs are older codex-transcripts bundles from 2026-10-16, 22076408 and 29807877 encrypted bytes; neither is a new/actively copied database. Previous runner had discarded stderr, so the precise remote causes are unproved.
- Operator pass 2 (canonical hub cb95f294-af33-4a95-99d0-b583012e94da): 7 verified (2 recovered plus 5 newly archived pairs), 56 already recorded, 0 pending, 0 downloads. Hub explicitly accepts the live check and directs finishing batching and landing. This proves retry/adoption/dedup with the earlier serial candidate; no claim of measured batch performance yet. Run from a slot with explicit PAIR_DATA_DIR=$HOME/.local/share/pair.
- Hub 99090e02-f6db-4b07-b87d-0f9677fc3286 approves flat hard-link staging, EXDEV verified-copy fallback, 4 GiB logical pair cap with overflow deferred, one batch copy and optional hash batch, and cleanup after every exit. Revised change-code passed plan-quality CLEAN and estimate-quality INFO. The original 2.48h estimate is the whole-issue forecast, not a new estimate for remaining work; actuals stay measured.
- Runner now includes the final bounded 16 KiB stderr diagnostic in pending errors, removes terminal controls and redacts credential values. Targeted diagnostics race tests passed 2.393s; aggregate verification is still in progress.
- Separate freshness check: local catalog has 10 codex-database records. logs_2/state_5 were fresh at 2026-10-19T01:53:42Z/01:54:02Z; thread_history_1 latest remains 2026-10-16T18:22:42Z (~55h old). #616 fixed size limits, not snapshot progress under writes. Filed independent unclaimed #688 (codex-snapshot-freshness), with those observations; no database snapshot code changed under #678.

### Batch qualification checkpoint — 2026-10-18

- Batch coordinator/fake focused tests passed historyarchive (52.236s), archivetest (0.185s). Revised race qualification passed historyarchive (75.317s), archivetest (1.297s), archivecmd (1.293s). Installed-rclone local batch/filtered-hash/cancellation conformance passed historyarchive (0.818s), archivetest (0.179s). Tracked-file guards passed internal/repolint (17.086s), artifactpath (16.501s), testpair (21.568s).
- Tests enforce actual rclone invocations <=4 for backend-hash N=8/16 passes (one resolve, two listings, one copy), <=5 with one hash batch, and no additional copy batch on rerun. They cover hard-link identity, EXDEV verified copying, complete-pair byte deferral, receipts for verified members of a failed batch, late duplicates, and cleanup on normal/failure/cancel paths.
- Additional cleanup regression was RED when a local snapshot failure left interrupted staging links. Cleanup now runs under the next continuous-lock owner even if local backup fails before Drive replication; cleanup failure is reported as remote-only. Its focused run and ordinary failure/cancel/local-success regressions passed (1.525s). No Codex snapshot algorithm was changed.
- Earlier serial-source full suite finished successfully on 1ed7a6e86e1e4aa2be0e97d67a8ad03ce4538d5f: exit 0 in 17m17s, queue 601s, log sha256 prefix 3d4b30ac4905. Retained as earlier-source evidence only. Batch source requires its own pushed-head recorded suite, which is next; no overlapping old run remains.
- Implementation/acceptance tasks are complete; normal recorded-suite, binary-owned review and landing obligations remain pending. The checkable issue Plan now separates feature acceptance from those process-owned future steps, without dropping any obligation.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration is marked stale by estimate-source, so values remain provisional.*

One greenfield pure model/receipt module: base design 1.5h, halved for existing encoding/crypto/path libraries then multiplied by 0.2 for the accepted concrete design = 0.15h; base implementation 0.6h × 0.4 = 0.24h. One batch external API adapter plus stateful fake: base design 2h × 0.2 = 0.4h, base implementation 1h × 0.4 = 0.4h. One smaller integration/receipt module: design 0.15h × 0.2 = 0.03h, implementation 0.4h × 0.4 = 0.16h. Docs: design 0.1h × 0.2 = 0.02h, implementation 0.15h × 0.4 = 0.06h. Real API discovery 0.45h × 0.4 = 0.18h implementation. One close review: design 0.1h × 0.2 = 0.02h, implementation 0.35h × 0.4 = 0.14h. Library availability: Go standard JSON/crypto/exec plus existing Catalog/LocalSink remove any need for a new Drive client library; rclone owns Google APIs and credentials. Familiarity 1.5 for novel but bounded rclone semantics; thorough approved plan design buffer 15%. Design subtotal 0.62h, implementation subtotal 1.18h; 0.62 × 1.15 + 1.18 × 1.5 = 2.483h, rounded 2.48h. This includes conformance and one review, excludes idle suite/landing queue time.

```estimate
model: estimate-logic-v3.1
familiarity: 1.5
item: greenfield-go-module design=0.15 impl=0.24
item: api-integration design=0.4 impl=0.4
item: smaller-go-module design=0.03 impl=0.16
item: atlas-docs design=0.02 impl=0.06
item: real-api-discovery design=0 impl=0.18
item: milestone-review design=0.02 impl=0.14
design-buffer: 0.15
total: 2.48
```

- 2026-10-18 revision: clarified feature-acceptance checklist versus process-owned suite/review/landing, retaining pending workflow in the Log. Scope/Done when remain unchanged; operator already accepted the live pass/rerun.


### Close review round 1 — 2026-10-18

- Binary-owned review returned REWORK, BR-1 Critical (writer-reader-capacity-consistency): the per-batch entry cap ignored the already populated destination and could exceed the reader bound permanently. Implement destination-wide missing-member reservation, complete-pair deferral and near-capacity retry/adoption regressions before reclose. Actuals were measured/adopted automatically at 1.75h at that boundary; later attribution remains gate-owned.
- Cancelled only this checkout's queued suite on e513ccbe with sdlc local-check cancel --name suite; three other checkout runs were left running. That unrun candidate is superseded by the correction and will receive a new pushed-head suite.
- Recovery check: couch --peek pair:6 --previous --all returned source none, previous recovery screen unavailable. Continued from Git, sdlc state, issue/plan, canonical live receipts and the retained close ledger; no previous terminal bytes are claimed available.
- Plan revision clarifies artifactpath's tag-bearing vocabulary versus content-addressed historyarchive artifacts and the external 64 MiB wire observation limit. Separate #688 remains unclaimed; no database snapshot algorithm changes are included.

- BR-1 correction: reserve actual destination entry count once (including unrelated duplicate entries), charge only missing members after successful preparation, and defer complete pairs before crossing 100,000. Capacity does not block later fitting candidates. Regression RED (invalid remote inventory count after four new objects, 6.496s) then GREEN (7.863s); broader coordinator/model/fake run passed historyarchive 14.780s and archivetest 0.136s. Tests also cover full-inventory receipt adoption, single-member completion and later vacancy recovery. Artifactpath guard passed 9.763s. Final corrected-source suite and close round 2 follow.
