---
id: 000616
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
severity: medium
card_mirror: 'bc8e077c3f07aaf5a776830a20d4574b87bfabeb' # card fields mirrored from issue-cards; edit via sdlc
---

# history-backup: bound database snapshots by compressed bundle size, not source size

## Problem

history-backup's per-file bound (`MaxBundleBytes`, 1 GiB, checked on the source file in `SelectedFiles.Describe`) skips Codex's `thread_history_1.sqlite` (1.5 GB) in every continuous pass (pair#592), so the largest Codex database has no backup. Database snapshots compress well.

## Spec

For database snapshots, bound the compressed bundle (ciphertext limit), not the source file size; ordinary files keep today's bound. Streaming stays bounded in memory.

## Done when

- A snapshot larger than 1 GiB whose compressed bundle fits the ciphertext limit is archived and verified; one whose compressed bundle exceeds it is refused with a clear note.
- The live daily pass archives thread_history_1.sqlite.

## Plan

- [ ]

## Log

### 2026-10-16
