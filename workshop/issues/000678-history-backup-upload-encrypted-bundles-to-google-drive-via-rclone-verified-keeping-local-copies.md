---
id: 000678
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours: 2.48
collaboration-mode: collaboration
card_mirror: '719194db68a1d69ba90ec2b87e2458bca01d15e8' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T17:51:28-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
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
- [ ] Document configuration/recovery, obtain operator-run Drive conformance, and land through recorded suite and fresh close review.

## Log

### 2026-10-18


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
