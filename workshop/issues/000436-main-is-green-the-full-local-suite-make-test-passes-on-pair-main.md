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
flow: {kind: quick, provenance: inferred, spec: "cd7688a9", done: "5fa541c3"}
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

- [x] `make test` passes on main from a clean slot, twice in a row, with the wall time recorded.
- [x] No package takes over 5 minutes in the default lane.
- [x] Every quarantined test has an owning issue.

## Plan

- [x] Baseline `make -k test` on main in a scrubbed env (`env -i HOME PATH TMPDIR USER`, default `/var` TMPDIR).
- [x] Fix `test-changelog` at the root (owner-frame paths).
- [x] Commit weave's ariadne#320 union-merge `.gitattributes` block (TL fold-in).
- [x] Baseline `go test ./...`: per-package times, failures.
- [x] couchcore: find what makes it slow; fix or move to an explicit lane.
- [x] `TestSpawnComposesProductionPairRegistrationBoundary`: root-cause and fix.
- [x] Fix the other failures the baseline surfaced (artifactpath, gcruntime, couchcmd, host-creation lease).
- [x] `make test` green twice from a clean slot, wall time recorded.

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
- Go baseline (clean env, `go test ./... -json`): rc=1, 645s. Failures: artifactpath
  `TestProductionArtifactReferencesAreExactlyClassified`, couchcmd `TestColdResumeOfAParkedPrimaryRegistersFromBothOrigins`
  (switcher, ~1 in 3), gcruntime `TestCouchReferencesLocalArchiveLocatorRoundTrip`; couchcore hit the 10m timeout
  (601.7s). Over 5 min: couchcore; gcruntime 408s (one test, `…HundredThousandFilenames…`, 396s).
- couchcore solo, serial (`-timeout 40m`): 1,315s, all pass. 1,084 tests, none parallel; heavy tail (matrix/property/
  real-git), no shared wait to remove.
- Fixes (one commit each):
  - gcruntime 100k-file budget test → injected `Collector.MaxEntries` at 1,000 (`DefaultMaxEntries` names the 100k);
    mutation under budget fails. 396s → ~5s.
  - gcruntime local-archive fixture: stale since #355 M2 (471a2056) made slot routing require the record's scope to
    name its checkout; it hard-coded a scope (subagent root cause).
  - couchcmd cold-resume: Enter raced the first inventory (header paints first); wait for the parked row, as
    warm_reattach_test does. 8/8 after, ~1/3 failing before.
  - artifactpath: 59 known violations quarantined as a shrink-only ratchet
    (`testdata/known-artifact-violations.txt`), owned by **pair#348**. New violations and stale lines both fail;
    mutation-checked both ways. This is the one quarantine.
  - couchcore `TestSpawnComposes…`: fails only inside a live Couch session, because it inherited
    `COUCH_PAIR_DATA_DIR` (read by LaunchNative since #366, cc2f3ce1). The test now selects its own root as Couch does.
  - couchcore parallelism exposed a **production race**: `AcquireHostCreationLease`'s `LOCK_NB` retake failed
    "workspace host creation is busy" when a fork in another goroutine still held a copy of the released descriptor
    (cleared in 2-9ms, measured). Couch forks git from other goroutines constantly. The retake now settles for up to
    100ms (500ms first; it doubled the scenarios that hold the lease on purpose). Test: `TestProvisionLeaseOutlastsATransientForkCopy`.
  - couchcore: 1,026 tests `t.Parallel()` per a static audit (helpers followed); serial: env/chdir/global/real-signal/
    wall-clock. Package 159s / 174s, `-race` clean (258s). Wall-clock budgets (#387 ARCH-CONSTRAINTS, 94-182ms vs
    150ms by host load) assert only under `make test-perf-budgets` (`PAIR_PERF_BUDGETS=1`); documented in atlas/couch.md.
- TL relay (pair:2): couchcmd cold-resume (fixed above), `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees`
  and `make test-pair-embedded-runtime` fail on main from agent shells. Both pass in a clean env: they are the
  session-env leak class. A live slot's shell exports ~80 PAIR_*/COUCH_*/ZELLIJ* values, most of them paths to the
  session's own files. Root fix in Makefile.local: when every goal is `test`/`test-*`, `unexport` each
  environment-origin PAIR_*/COUCH_*/ZELLIJ* name (GNU Make 3.81 has no target-scoped unexport). Workflow goals keep
  the slot identity. `make test-pair-embedded-runtime` from a live slot: passes; with the scrub disabled it fails
  "PAIR_DATA_DIR … conflicts with selected repository scope".
- Under load, parallel couchcore exposed two timing-fragile tests: the lease settle test (its 20ms timer fired after
  the 100ms window at load 30; it now takes the window as a parameter) and 1s event-wait guards (five now allow 10s).
  The tests mostly wait on subprocesses (1.9 of 12 cores busy), so couchcore's TestMain defaults `-test.parallel` to
  3x GOMAXPROCS: 236s → 139s.
- **Wall time, before → after** (branch tip merged with main 8d7c50ef):
  - Before: `make -k test` on main stopped at test-changelog after 214s, so the Go suite never ran. `go test ./...`
    took 645s with 3 failures and couchcore timing out at 10m. couchcore solo, serial: 1,315s.
  - After, clean env: `make test` passed twice, 1,076s (first build, concurrent with verification runs) and 429s.
  - After, from the live slot (session env present), final tip: `make test` passed twice, **434s and 438s**. Largest
    package couchcore 207s / 205s; no other package over 100s. `go test -race ./cmd/internal/couchcore`: clean, 201s.
- Quarantined: artifactpath's 59 known violations → **pair#348** (ratchet). No other test is skipped or excluded.
- Close review round 1 (FIX-THEN-SHIP, not finalized), fixed by class:
  - BR-1 Important: the session-env scrub also dropped opt-in knobs (PAIR_PROBE_*, PAIR_PERF_BUDGETS, PAIR_LIVE_*,
    PAIR_ARTIFACT_VIOLATIONS_UPDATE). Makefile.local now keeps `TEST_ENV_KNOBS` (PAIR_TEST_/LIVE_/PERF_/PROBE_/NATIVE_/
    ARTIFACT_ families, PAIR_COUCH_PRESSURE, PAIR_TERMINAL_ORACLE). `tests/make-test-env-test.sh` (`make
    test-make-env`, in `make test`) derives the knob list from every `NAME=… make|go test` in README/atlas/Makefile.local/
    doctor, so a newly documented knob is checked automatically. It asserts that each knob reaches a test recipe and
    that session identity does not. Both mutations fail it: an empty keep-list, and no scrub.
  - Minor: `OwnerFramePath` kept a dangling symlink's name as if missing; it now refuses (test added).
  - Minor: the artifact ratchet deduped normalized lines, masking a second identical violation; it now compares
    counts (duplicate known line → STALE, mutation-checked).
  - Also gofmt'd `storagegc/collector.go` (my DefaultMaxEntries edit had dropped a blank line).
- Close review round 2 (BR-4, same family): the prefix keep-list still dropped knobs that only tests read
  (PAIR_UPDATE_GROUPED_FIXTURES, PAIR_TERMINAL_*, PAIR_MENU_PERF_TARGET, PAIR_PARK_LATENCY_SMOKE). The rule is now
  structural. A session's names are the ones production code mentions, in any syntax (SetEnv, constants, KDL, shell),
  plus zellij's ZELLIJ*. The scrub drops exactly those environment-origin names, minus TEST_ENV_KNOBS (families a
  production tool reads as an operator knob, e.g. doctor/perf.sh's PAIR_PERF_*). A name only tests read passes
  through by construction. `tests/make-test-env-test.sh` derives the knobs (every name test code reads that production
  never mentions, plus every documented `NAME=… make|go test`) and requires each of the 165 to reach a test recipe.
  It also requires session identity not to. It works without .git (find, not git ls-files) for test-clean-bootstrap.
  Mutations: scrub-all and no-scrub both fail it. The live slot has 0 session names left in a test recipe.
  - BR-3: the counted ratchet's comparison is now a pure `diffKnownViolations` with a duplicate-count test.
- New flake from main (#433, landed 10-11): couchcmd `TestMessageWrapperExitDisconnectsAndForgetsWorkspace` read the
  workspace map the instant the binding showed disconnected. The service loop publishes the connected set before it
  runs the Disconnect effect, by design ("Publish first"). The test now waits, bounded, for the forget (20/20).

