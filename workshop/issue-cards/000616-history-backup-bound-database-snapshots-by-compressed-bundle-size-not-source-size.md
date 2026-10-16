---
id: 000616
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
severity: medium
---

# history-backup: bound database snapshots by compressed bundle size, not source size

## Problem

history-backup's per-file bound (`MaxBundleBytes`, 1 GiB, checked on the source file in `SelectedFiles.Describe`) skips Codex's `thread_history_1.sqlite` (1.5 GB) in every continuous pass (pair#592), so the largest Codex database has no backup. Database snapshots compress well.
