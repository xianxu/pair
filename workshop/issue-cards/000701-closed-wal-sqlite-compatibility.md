---
id: 000701
status: working
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
started: 2026-10-19T12:08:04-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Read closed WAL databases without sidecars on macOS SQLite

## Problem

#688's native acceptance found memories_1.sqlite (45,056 bytes, WAL format) had no -wal/-shm files, and macOS system sqlite3 3.54.0 returned exit 1 at PRAGMA schema_version with unable to open database file. The source remained unmodified by the failed read. A disposable closed-WAL fixture reproduces this independent of BEGIN order and -nofollow; installed Homebrew sqlite3 3.53.4 reads it and creates normal coordination sidecars. Active databases with existing sidecars work.
