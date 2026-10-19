---
id: 000707
status: open
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '7727fe0f9f8376f3bd72a67fa33eec9f0785fb98' # card fields mirrored from issue-cards; edit via sdlc
---

# tests: no-sync mode for pair's durability seam, plus fsync-free throwaway git repos and SQLite in tests

## Problem

pair has 54 Sync() calls in production code (atomic write: temp file, sync, rename). On macOS, Go's File.Sync() issues F_FULLFSYNC, the most expensive sync. Production needs them; tests exercise those paths thousands of times and so drive an fsync storm on the shared SSD (550,944 APFS container metadata writes in 64 s during a pair suite build, 2026-10-19). Tests also create throwaway git repos (git syncs by default) and SQLite databases (synchronous=FULL by default).

## Spec

One durability seam: every production sync goes through a small package function (e.g. `durable.Sync(f)` / `durable.SyncDir(path)`), which a test can switch to a no-op for its process (an env var set by the test harness, e.g. PAIR_TEST_NOSYNC=1, read once; never set in production, and refused unless running under `go test`). A lint guard forbids direct `.Sync()` outside the seam. Test helpers that create throwaway git repos set `core.fsync=none` (and `core.fsyncObjectFiles=false`) in those repos; test SQLite opens with `PRAGMA synchronous=OFF`. Production behavior is unchanged and proven by a test that the seam syncs when the variable is unset.

## Done when

- The lint guard passes with zero direct Sync calls outside the seam; production paths still sync (test).
- A measured pair suite run issues far fewer fsyncs (the operator's fs_usage fsync count, or rusage), and wall time does not regress.

## Plan

- [ ]

## Log

### 2026-10-19
