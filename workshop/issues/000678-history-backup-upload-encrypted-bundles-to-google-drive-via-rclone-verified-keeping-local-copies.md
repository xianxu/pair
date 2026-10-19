---
id: 000678
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'b2421882cf988bca4fa54bbc9f5eaba68e694999' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T17:51:28-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
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

Durable design: [000678-drive-history-backup-plan.md](../plans/000678-drive-history-backup-plan.md). Full flow; awaiting hub approval before change-code.

- [ ] Define destination/receipt identities and pure upload transitions; test configuration defaults and remote-only failures.
- [ ] Implement the bounded rclone seam with a stateful fake and disposable installed-rclone local conformance.
- [ ] Integrate verified backlog replication after local completion, immutable upload receipts, private staging and counted N/2N recovery tests.
- [ ] Document configuration/recovery, obtain operator-run Drive conformance, and land through recorded suite and fresh close review.

## Log

### 2026-10-18


- Recovery evidence: requested previous terminal capture returned unavailable. Git and sdlc state show #670 already landed and #678 claimed/started on its own branch; no implementation edits. Historical screen instructions are context, not fresh permission. Verified the original #678 dispatch receipts before continuing its existing planning work.
- Read-only map: existing local publication independently verifies ciphertext and public BundleLabel before CatalogRecord append. Continuous passes with no fresh sources still need a full catalog backlog phase. Catalog identity is bundle ID alone and its index treats every record as local; proposed remote receipts therefore live in a separately named upload catalog rather than changing existing record authority (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).
- Plan checkpoint: workshop/plans/000678-drive-history-backup-plan.md records the full proposed flow, explicit remote/local visible states, uncertainty/cancellation recovery, stateful FakeRclone seam, N=8/16 counts, lifecycle and operating limits. Separate upload catalog and limits await hub approval; no change-code or production edit yet.
- Discovery: installed rclone v1.75.2 supports lsjson hash selection and hashsum --download. Drive may omit SHA256 on some objects, so explicit downloaded hashing is the fallback; never accept copy command success or size alone. Only version/help and official docs were consulted; no Drive request, credential/config read, upload, operator-session change or restart occurred.

## Revisions

- 2026-10-18: expanded the initial empty Plan into a full-flow design after inspecting local catalog identity and publication. Proposed a destination-specific upload catalog, keeping the existing local schema and authority unchanged; retained the original Spec and Done when for hub review.
