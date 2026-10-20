---
id: 000715
status: working
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '58a16c403218a835d01f017672aebacffe3232c2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T18:06:43-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/couchcmd.TestContinuationWriterPublishesExactCheckpointAcrossWorktrees]
---

# flake: TestContinuationWriterPublishesExactCheckpointAcrossWorktrees

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcmd.TestContinuationWriterPublishesExactCheckpointAcrossWorktrees: Same exact head b1cc1870b39b9111f41a86c27ddd16afdb730735: recorded APFS suite attempt d7b39428f6f4d3de2182c233873dbb15 failed couchcmd.TestContinuationWriterPublishesExactCheckpointAcrossWorktrees/warm-reattached (3.25s parent, 1.01s subtest): launcher.TestCouchLaunchPromptTransportHelper TempDir RemoveAll cleanup failed unlinkat .retention: directory not empty. Full suite remains running to collect all failures. Unchanged exact suite checkout /Volumes/TestScratch/local-checks/sdlc-local-3359052142 targeted go test ./cmd/internal/couchcmd -run ^TestContinuationWriterPublishesExactCheckpointAcrossWorktrees$ -count=1 -v -timeout=2m then PASS all five subtests (10.81s parent, 11.096s package; warm-reattached 2.17s), exit 0. No source edits or suite retry. Investigate retention writer lifetime versus helper TempDir teardown; root cause not yet established.

## Spec

The wrapper owns each session-watch process from creation through joined exit. Keep the watcher in its own session so detaching the Pair client cannot interrupt launch binding, but retain its direct child handle instead of Process.Release. Reuse wrapcmd's existing ownedGroup shutdown: terminate the exact owned group, give TERM-refusing descendants the existing one-second grace, then kill/join before wrapper return. Normal group exit must finish promptly without spending that grace. Binding/lifecycle selection and persisted formats do not change.

The reproduced ordering must hold watcher startup before retention acquisition, allow the native agent to finish, and prove that the old wrapper returns with a writer still able to acquire its actual retention lease. The permanent regression must then require wrapper return to imply joined watcher shutdown, including ignored TERM and watcher descendants. Directory deletion is unchanged; no retries or sleeps added to cleanup.

ARCH-DRY: reuse ownedGroup, not a second process-owner abstraction. ARCH-ORDER/ARCH-FUNERAL: wrapper start -> native/watcher running -> native end -> watcher stop/join -> wrapper return -> test TempDir removal. ARCH-PURE: no new domain decision; the changed seam is process IO. ARCH-PURPOSE: enumerate every startWatcherProcess caller/mock and test the production startup/shutdown composition, not just the standalone stop helper. ARCH-MOCK: native agents/watchers are owned executable fixtures with explicit gates. ARCH-CONSTRAINTS: one watcher per wrapper; normal stop has no intentional delay, TERM refusal alone may use the existing one-second group grace; record normal exit measurements under owned load. ARCH-SECURE: use only the direct child created with Setsid; no pidfile or foreign process grants signaling authority. ARCH-EVOLVE: no persisted format changes.

A Pair-local golint rule will reject Process.Release in the wrapcmd package, citing #715, so owned wrapper children cannot silently regain fire-and-forget lifetime. The common catalog lives upstream; the local rule uses its existing checker interface without editing Ariadne or changing the dependency pin.

## Done when

- Deterministic gated reproduction proves the detached session watcher writes retention state after the old wrapper returns; the fixed wrapper always stops and joins that exact writer before return.
- Detach before first turn still preserves watcher binding, fast native exit and already-finished watchers join safely, and TERM-refusing watcher descendants are stopped within the existing group policy.
- Record normal-case wrapper exit overhead; only TERM refusal consumes the shutdown grace. Repeat the original continuation family under bounded owned load and run focused race/source/lint checks plus the recorded suite and normal close/merge gates.
- No cleanup retry or longer timeout hides the race; closing this issue unregisters its flake test.

## Plan

- [ ] Add a gate-controlled reproduction and observe its expected RED result before changing production ownership.
- [ ] Return/reuse ownedGroup at the watcher spawn seam, defer its shutdown/join, update callers and fixtures, and add the local catalog rule plus failure-path tests.
- [ ] Measure normal versus TERM-refused exit, run original continuation under bounded load, focused race/source/lint checks, recorded suite, close and land.

## Log

### 2026-10-19

Investigation: session-watch RunCLI acquires storagegc.AcquireSelectedProcess before native PID/liveness observation and defers lease.Close; both write under .retention. wrapcmd.startWatcherProcess starts Setsid and immediately releases the child, so helper CombinedOutput does not fence those writes. gcruntime maintenance already Stop/Wait joins. Current sessionwatch exits before scanning/publishing when native identity changes, so watcher shutdown after native completion introduces no new final-event contract. Hub approved existing ownedGroup reuse in delivery 88f5c1b1-818d-4daf-9f45-3ab4058a8eeb, requiring normal exit overhead measurement. Unrelated generated workflow changes from start-plan's dependency catch-up are preserved in named stash 715-start-plan-generated-workflows-preserved; hub identifies them as pending #699 and directs leaving them stashed.
