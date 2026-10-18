---
id: 000659
status: codecomplete
created: 2026-10-17
updated: 2026-10-18
estimate_hours:
github_issue:
severity: high
started: 2026-10-17T22:43:28-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
actual_hours: 1.07
tracker:
    version: 1
    completion:
        token: close-f8a75a4f43b8
        repository: github.com/xianxu/pair
        reviewed_head: 8f59f920b2064566f2da4baddf36d0f56cd6041f
        evidence_commit: 0f347776d6fbda5dd2340b5f2311509676dac067
---

# tests: seven wall-clock budgets on external tools (git worktree, zellij IPC, lsof, a rate-limit window) fail at fleet load

## Problem

One full-suite run (2026-10-17 22:13:31-22:33, 19m51s, exit 2, head `6a7940eec` of
#644's branch, log sha256 `0c150ba9b316` on Mac.localdomain) failed seven tests in
five packages. Not one reported a wrong answer: each put a wall-clock budget on work
it does not own — an external tool's reply, or real time passing — and the host was
busy. `sysctl vm.loadavg` read `{ 5.40 78.11 97.03 }` immediately after the run, so
the 5- and 15-minute averages across the window were 78 and 97 while ariadne's
sharded suite and other slots' builds were also running.

| package | test | verdict under load |
|---------|------|--------------------|
| broadcast | `TestPointerRateLimit` | "only 11 of 50 rapid posts were limited" |
| couchcmd | `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees/warm-reattached` | "signal: killed" after 27.34 s; the fixture's push also reports "'origin' does not appear to be a git repository" |
| couchcmd | `TestClipboardCopyConsoleKeysReachContentFreeFactLog` (both console-key subtests) | "missing real execution evidence", ElapsedMS 1108 and 2177 |
| couchcmd | `TestParkHelperAttachmentRequiresClientGrowth` (both subtests) | "zellij --session pair-readiness action list-clients: context deadline exceeded" |
| couchcore | `TestRecoveryFreshOrdinarySlotKeepsOrdinaryLaunch`, `TestRecoveryFreshMissingRecordLeavesForeignOwnersUntouched/arrives-during-prepare` | "git worktree list --porcelain -z: timed out after 5s: signal: terminated" |
| couchtty | `TestWorkbenchPerformanceIdleUITick` | eight activity probes "context deadline exceeded" at 2.3-6.1 s each, then "activity worker did not finish" |
| diagnosticlog | `TestOpenFileInspectionConformance`, `TestUnregisteredChildOpenFileBlocksLegacyExpiry` | "lsof unavailable or incomplete" |

**This is not a regression report against #644.** That branch's patch is five files —
`resources_test.go` (whose `TestResourceSamplerNativeCost` is env-gated and skipped in
a suite run), two `hostLoadAverage` helpers, `atlas/couch.md` and issue prose — and
cannot reach these packages; `go test ./cmd/internal/procutil` passed in the same run.
The run exposed a class, which is what this issue files.

**The class.** A test that gives an external tool (`git`, `zellij`, `lsof`) a fixed
wall-clock budget, or counts events inside a real-time window, or waits on a probe
that must answer inside a fixed deadline — and then treats that budget as the
assertion. Under fleet load the same code takes longer, so the test reports a timeout
instead of a behavior, and the failure names neither the host state nor the budget's
owner. The cost is not only noise: a red suite record blocks the landing it was run
for, so one busy hour on the machine can stop unrelated work.

Relationship to existing issues:

- #485 is the same shape but narrower — the wrapcmd/couchtty harness reap timers and
  ordering asserts found by #479's `-race` soak, plus a sweep list of specific
  untouched files. This set is different tests, found by a non-`-race` full suite at
  fleet load; the hub ruled it gets its own issue rather than folding into #485.
- #203 (Couch does not bound the build parallelism of the sessions it hosts) is where
  the load comes from; #532 (ping the hub with load attributed per slot when the
  machine runs hot) is the measurement side. Neither fixes a test that asserts a
  wall clock, so this issue stands on its own.
- Two of the seven are recently landed: park client-count readiness from #645, the
  clipboard fact-log check from #647.
- #626 and #628 (both archived) already worked the couchtty activity worker's
  timeouts; `TestWorkbenchPerformanceIdleUITick` may be the same seam again.

Evidence retained: the record's own copy at
`/Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/dbdf8d64d556e87195180367761a97a2.log`
(the record keeps the newest 20 commits, so copy it out before it rolls), a copy at
`/tmp/pair644-suite-attempt1.log`, and the same table in #644's `## Log`.
