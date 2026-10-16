---
id: 000616
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
severity: medium
card_mirror: 'f357c41eafbc55779618b5bcc8be3ac21c67f1c0' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T11:06:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
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

- [x] One kind-based `rawBundleLimit` (snapshot kinds `MaxSnapshotRawBytes` 8 GiB, others 1 GiB) used by Describe (`SelectedFiles.Limit`), PlanBundles, ValidateManifest and ValidateLabel; the snapshot pre-check uses the snapshot bound.
- [x] `ErrCiphertextLimit` from the bounded writer's own record; the continuous pass skips that snapshot with a note.
- [x] Tests (RED-checked) and a live codex-database pass.

## Log

### 2026-10-16
- 2026-10-16: closed — Round-1 fixes: BR-1 one request per snapshot so an over-ciphertext snapshot is skipped alone (test archives the small DB and skips the big one; RED with batching); BR-2 the daily snapshot storage cost (~115 MB/day for thread_history, ~40 GB/year, never-deleting sink) stated in atlas history-backup.md and raised to the TL as a retention decision. Recorded suite green on fba9593f; historyarchive green after fixes. Live: thread_history_1.sqlite 1.60 GB -> 115 MB bundle, verified.; review verdict: FIX-THEN-SHIP

### 2026-10-16 — fix and live pass (pair:2)

Kind-based raw bound, because ValidateManifest also runs on catalog reads: a request flag would let a >1 GiB snapshot be written and then fail every catalog read. Live `pair archive continuous --kind codex-database` (niced, 633 s): 3 files archived, no skips; thread_history_1.sqlite 1.60 GB raw -> 115 MB bundle (13.9x), bundle sha256 and size verified in the sink.
