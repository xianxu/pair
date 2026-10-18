---
id: 000678
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '842c20f5f5ca53a3c55105df10014cc035e02f26' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-18
