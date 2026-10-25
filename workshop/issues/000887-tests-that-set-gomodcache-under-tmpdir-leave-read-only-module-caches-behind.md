---
id: 000887
status: working
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: 'f63a9ed09c229a0959161b9c8b358662e5b5c903' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-25T08:28:36-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "d1fced33", done: "31311ba4"}
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

- [ ] Pin nested read-only cache deletion, symlink preservation, idempotence and automatic cleanup.
- [ ] Add testpair.RemoveCacheTree and CacheTempDir; convert launcher teardown and named cache-capable fixture allocations. Cleanup errors fail the owner after children join.
- [ ] Add a Pair catalog guard with positive/negative fixtures; audit disposable-HOME Go writers.
- [ ] Run targeted checks, inventory, pushed pre-close, close and merge.

## Log

### 2026-10-25

Hub and ariadne:5 independently identify retained pair-launcher-*/go/pkg/mod trees: 566 read-only directories, roughly6000 entries/77MB, from ignored cleanup after private HOME. No retained evidence tree touched. Current testdisk already reuses ambient GOMODCACHE; testcouch build settings are fakes. Hub dfef3fa6 approved the shared cleanup plan.

## Revisions

### 2026-10-25 — Correct the producer census; approved by hub dfef3fa6

The retained producer is launcher/session_quiescence_live_test.go's TestMain: private HOME implicitly supplies GOPATH and GOMODCACHE, and ignored RemoveAll leaves extracted read-only modules. Preserve the named fixtures' isolation while giving cache-capable owned roots shared cleanup. One chmod-before-descent directory walk plus RemoveAll skips symlinks and returns errors; CacheTempDir registers cleanup before returning. Launcher cleanup runs after testpair.Run and converts failure to exit1. No shared cache or retained ariadne#631 evidence is removed. No new user state, format, timer or registry. Cost O(N) walk plus O(N) removal, once per owned tree. ARCH-DRY/ARCH-RETENTION: the allocating fixture owns teardown.

The catalog guard rejects raw temporary GOMODCACHE allocations and private-HOME TestMain removal that bypasses shared cleanup; syntax enforcement does not prove arbitrary computed environments. Filesystem tests cover nested0555 directories, files0444, symlink targets unchanged, absence and cleanup order.
