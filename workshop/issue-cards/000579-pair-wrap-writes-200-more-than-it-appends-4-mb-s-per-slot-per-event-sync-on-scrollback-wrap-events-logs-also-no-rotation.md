---
id: 000579
status: done
created: 2026-10-15
updated: 2026-10-15
estimate_hours: 2.622
github_issue:
started: 2026-10-15T15:07:36-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: 5.06
tracker:
    version: 1
    handoff:
        token: move-e96d5c11f81f
        repository: github.com/xianxu/pair
        source_branch: refs/heads/main-slot3
        source_base: d59420ed5eee5e3b94113bb8cac0c82cff5d4a59
        source_head: d59420ed5eee5e3b94113bb8cac0c82cff5d4a59
        source_path: workshop/issues/000579-pair-wrap-writes-200-more-than-it-appends-4-mb-s-per-slot-per-event-sync-on-scrollback-wrap-events-logs-also-no-rotation.md
        source_blob: ab045c206f539dd9d7a8e9e9500855301a131431
        destination: workshop/issues/000579-pair-wrap-writes-200-more-than-it-appends-4-mb-s-per-slot-per-event-sync-on-scrollback-wrap-events-logs-also-no-rotation.md
        main_commit: 7af5c7b16a957d41f5d6577c34b3a35cea83f0ab
    completion:
        token: close-1eb1b62f1af8
        repository: github.com/xianxu/pair
        reviewed_head: bd641b5d9b551a6baee522fb955c14ce86efb27a
        evidence_commit: 5ed5708b4837d6f1248228102bf30a94219a4370
        landed_commit: 78b32f71004856a4fdfb5cf223d6e3e8ed41f72d
---

# pair wrap writes ~200× more than it appends (~4 MB/s per slot): per-event sync on scrollback/wrap-events logs; also no rotation

## Problem

2026-10-15 (operator's system monitor): each `pair wrap` process (`--scrollback-log …/scrollback-1-<tag>-codex.raw`)
**writes 3.4–4.1 MB/s to disk**, about 29 MB/s total for the fleet (~2.5 TB/day of write wear). Yet the files it
writes grow only **~17 KB/s** (`wrap-events-1-<tag>.jsonl`) and **~2.3 KB/s** (`scrollback-…raw`) per active slot,
measured over 10 s. That's a **~200× write amplification**, most likely a sync (fsync/F_FULLFSYNC) or a re-open per
small event write, so every few-hundred-byte append costs whole blocks plus filesystem journal writes. It's also likely
behind the FSEvents daemon's ~200% CPU seen on 10-14.

Separately, these files are **never rotated**: wrap-events logs of **3.9 GB** (pair-8), **2.6 GB** (pair-43), 1.9 GB,
1.7 GB, 1.0 GB; the pair data dir was 57 GB on 10-14.
