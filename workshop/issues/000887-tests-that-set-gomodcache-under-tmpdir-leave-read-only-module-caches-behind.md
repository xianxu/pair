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

- [x] Pin nested read-only cache deletion, symlink preservation, idempotence and automatic cleanup.
- [x] Add testpair.RemoveCacheTree and CacheTempDir; convert launcher teardown and named cache-capable fixture allocations. Cleanup errors fail the owner after children join.
- [x] Add a Pair catalog guard with positive/negative fixtures; audit disposable-HOME Go writers.
- [x] Run targeted checks and inventory, push the implementation and queue the required pre-close check.

## Log

### 2026-10-25

Hub and ariadne:5 independently identify retained pair-launcher-*/go/pkg/mod trees: 566 read-only directories, roughly6000 entries/77MB, from ignored cleanup after private HOME. No retained evidence tree touched. Current testdisk already reuses ambient GOMODCACHE; testcouch build settings are fakes. Hub dfef3fa6 approved the shared cleanup plan.

### 2026-10-25 — Fixture lifetime fixed and qualified

Original RemoveAll shim fails the filesystem regression with permission denied on nested read-only module content. Shared RemoveCacheTree passes read-only removal, missing-root idempotence, relative/root-path refusal, symlink target preservation and returned removal errors; CacheTempDir proves child cleanup runs before root removal. Focused helper tests PASS0.182s. Launcher child-process regression through the real TestMain PASS0.286s: a synthetic extracted module tree is gone after exit, and a blocked parent makes cleanup fail the process. A temporary source overlay restoring only the original ignored root defer makes BOTH controls fail (cache retained and false-success exit); no source mutation or retained module evidence touched.

Build-owner/legacy compatibility tests PASS1.726s; native cmd/go cache conformance PASS12.494s. Rule controls and full tracked-source catalog PASS6.743s. The rule is deliberately function-local syntax: direct temporary roots, aliases, literal Join descendants, Setenv/env strings and keyed cache fields; exact registered/deferred cleanup, shadow and unused-closure negatives. It does not prove dynamic paths, reassignments or interprocedural flows. Catalog lives in Pair's extension of the shared golint catalog, with incident887.

Reader audit before individual certificate refresh: launcher reads only controlled fixture roots; owner_build and legacy_build retain existing non-Go builder/environment fallback; testdisk retains fixed-scanner ownership of repo go.mod/go.sum and its explicit local fixture; new catalog matcher and filename binding consume AST/literal inputs only. Refreshed each named certificate separately, then inventory PASS. Other real Go consumers already pass/inherit ambient caches or capture buildEnv before disposable HOME; the retained whole-package HOME producer was launcher. No machine settings, shared caches, retained631 trees or pair:0 state changed. Next: commit/push, required recorded pre-close and mandatory close, then merge.

### 2026-10-25 — Recorded check retained; workflow-row correction

PR612 is open. Required pre-close selected the full-suite fallback on pushed8449f364138d32169f871f1b9ca0f0badc267c6c, retained terminal31020 / lease-1191244849 / attempt15583ade57ba5289e60f01326d854790 in its independent snapshot /Volumes/TestScratch/local-checks/sdlc-local-1723631917. Latest observed next in FIFO, not yet a test result. Do not cancel or duplicate it. Close terminal46722 measured exact0.50h but refused before review because its own close/merge gates were incorrectly grouped in an implementation checkbox. The completed check-queue operation is now the checkbox; close and landing remain outstanding here in the Log. No runtime/test change. Next: fresh close, inspect this exact recorded result, then merge when reviewed and green; no measurement waiver.

## Revisions

### 2026-10-25 — Correct the producer census; approved by hub dfef3fa6

The retained producer is launcher/session_quiescence_live_test.go's TestMain: private HOME implicitly supplies GOPATH and GOMODCACHE, and ignored RemoveAll leaves extracted read-only modules. Preserve the named fixtures' isolation while giving cache-capable owned roots shared cleanup. One chmod-before-descent directory walk plus RemoveAll skips symlinks and returns errors; CacheTempDir registers cleanup before returning. Launcher cleanup runs after testpair.Run and converts failure to exit1. No shared cache or retained ariadne#631 evidence is removed. No new user state, format, timer or registry. Cost O(N) walk plus O(N) removal, once per owned tree. ARCH-DRY/ARCH-RETENTION: the allocating fixture owns teardown.

The catalog guard rejects raw temporary GOMODCACHE allocations and private-HOME TestMain removal that bypasses shared cleanup; syntax enforcement does not prove arbitrary computed environments. Filesystem tests cover nested0555 directories, files0444, symlink targets unchanged, absence and cleanup order.

### 2026-10-25 — Separate verification setup from lifecycle gates

The final implementation checkbox records completed targeted checks, inventory, push and recorded-check queueing. Close and merge are lifecycle operations tracked in the Log, preventing a circular unchecked close/merge requirement. Required verification and landing criteria are unchanged.
