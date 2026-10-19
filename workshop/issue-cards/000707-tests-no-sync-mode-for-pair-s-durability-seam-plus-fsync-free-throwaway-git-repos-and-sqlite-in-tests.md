---
id: 000707
status: open
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
---

# tests: no-sync mode for pair's durability seam, plus fsync-free throwaway git repos and SQLite in tests

## Problem

pair has 54 Sync() calls in production code (atomic write: temp file, sync, rename). On macOS, Go's File.Sync() issues F_FULLFSYNC, the most expensive sync. Production needs them; tests exercise those paths thousands of times and so drive an fsync storm on the shared SSD (550,944 APFS container metadata writes in 64 s during a pair suite build, 2026-10-19). Tests also create throwaway git repos (git syncs by default) and SQLite databases (synchronous=FULL by default).
