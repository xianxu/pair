---
id: 000436
status: working
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: '50f0cf3701629a4a926afddca93746a52b976430' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-10T23:14:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
---

# Main is green: the full local suite (make test) passes on pair main

## Problem

pair's CI on main runs only the zellij conformance job (green since pair#324). The full local suite is not green on
main: on 10-10, pair:1 found that couchcore takes over 15 minutes and `TestSpawnComposesProductionPairRegistrationBoundary`
fails, both reproducing on main. While main is red, "it also fails on main" is the evidence every worker cites, and it
hides new breakage. The operator (batch-2 recap, 10-10): "let's ensure ariadne and pair are green on main".

## Spec

- Run `make test` (and `go test ./...`) on a clean main; list every failure and every package over 5 minutes.
- Each failure: fix it at the root, or quarantine it by name with an owning issue (as pair#324 did for pair#432).
  Quarantine is the exception, and every quarantined test is named in the issue's Log.
- couchcore's run time: find what makes it take over 15 minutes (live tests, timeouts, serial waits) and bring it
  down, or move the slow live tests behind an explicit lane that `make test` doesn't run by default, stated in TOOLING.
- Record the suite's wall time before and after in the Log.

## Done when

- [ ] `make test` passes on main from a clean slot, twice in a row, with the wall time recorded.
- [ ] No package takes over 5 minutes in the default lane.
- [ ] Every quarantined test has an owning issue.

## Plan

- [x] Baseline `make -k test` on main in a scrubbed env (`env -i HOME PATH TMPDIR USER`, default `/var` TMPDIR).
- [x] Fix `test-changelog` at the root (owner-frame paths).
- [x] Commit weave's ariadne#320 union-merge `.gitattributes` block (TL fold-in).
- [ ] Baseline `go test ./...`: per-package times, failures.
- [ ] couchcore: find what makes it slow; fix or move to an explicit lane.
- [ ] `TestSpawnComposesProductionPairRegistrationBoundary`: root-cause and fix.
- [ ] `make test` green twice from a clean slot, wall time recorded.

## Log

### 2026-10-10

- Baseline (main c7ee1a33 + weave block, scrubbed env, default TMPDIR): `make -k test` rc=2 in 214s.
  Only failure: `test-changelog` (`viewer: process target is outside selected owner directory`). make then
  skips `test`'s own recipe, so the Go suite never ran — main had no green `make test` at all.
- `test-changelog` root cause: `storagegc.SelectedOwner` evaluates `PAIR_DATA_DIR`'s symlinks, but the
  changelog viewer target (`AcquireRoleProcessTarget`) and the distiller paths (`changelogcmd`) were matched in
  the caller's spelling; macOS `/var` is a symlink to `/private/var`. scrollbackcmd and pairlog already resolved
  their paths, each with a private copy. Fixed with one `storagegc.OwnerFramePath` (deepest existing ancestor
  resolved) used at all four sites (ARCH-DRY). The lease now records targets physically, which also lets the
  collector match a reader target against capture paths. It also refuses a target that escapes the owner
  through a symlinked subdirectory. Tests: `TestRoleTargetThroughSymlinkedDataDirIsRecordedPhysically`
  (failed before the fix), `TestOwnerFramePathResolvesTheExistingPrefixOnly`.
- TL relay (pair:3 during #435): artifactpath fails and couchcore/gcruntime hit the 10m go test timeout under load
  on main. All three are in this inventory: artifactpath ratchet (78172266), gcruntime 100k-file test scaled
  (732b28a3, 396s → ~5s), couchcore parallelization (below).
