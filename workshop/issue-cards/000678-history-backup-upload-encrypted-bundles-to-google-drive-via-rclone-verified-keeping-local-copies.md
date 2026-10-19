---
id: 000678
status: working
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
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
