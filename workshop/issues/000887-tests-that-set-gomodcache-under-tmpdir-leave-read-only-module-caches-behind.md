---
id: 000887
status: open
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: 'a2bb7be9e9c6d7e6c0ba8b507cb085d8dedcadac' # card fields mirrored from issue-cards; edit via sdlc
---

# Tests that set GOMODCACHE under TMPDIR leave read-only module caches behind; clean them via one helper

## Problem

## Spec

**pair tests leave read-only Go module caches under the shared temp directory (found by ariadne:5, ariadne#631,
10-25).** `testcouch/*_build_test.go` and `testdisk/cache_conformance_test.go` set `GOMODCACHE` under `TMPDIR`; Go marks
module-cache directories read-only, so `os.RemoveAll` (and `t.TempDir` cleanup, and sdlc's sweep) can't delete them.
Each pair suite has left about one 77 MB, 6,000-file tree in `/Volumes/TestScratch/tmp` since 10-24, and sdlc's stale
sweep retried and failed on all of them on every call, which stretched ariadne suites past their 30-minute shard
limit. ariadne#631 fixes sdlc's side (removes read-only trees properly, sweeps once per run); this issue fixes pair's.

Each test that points `GOMODCACHE` at a temp directory cleans it the Go way in its cleanup (`go clean -modcache` with
that `GOMODCACHE`, or make the tree writable then remove it), through one shared helper; or the tests share one
read-only module cache across the suite instead of creating one per test, if that's simpler and keeps isolation.

## Done when

- A pair suite run leaves no module-cache tree in TMPDIR; a test proves the helper removes a read-only cache tree.
- The tests that set GOMODCACHE use the shared helper (a catalog check keeps new ones on it).

## Plan

- [ ]

## Log

### 2026-10-25
