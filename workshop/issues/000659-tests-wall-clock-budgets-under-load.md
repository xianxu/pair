---
id: 000659
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
severity: high
collaboration-mode: collaboration
card_mirror: '848db29645457793dab875290b59de1058464d60' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T22:43:28-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "abd9131f", done: "22b0c546"}
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

## Spec

Sort each of the seven into the shape it actually is, and fix in kind — no constant
gets raised just to make a loaded host fit:

1. **The budget is the product's.** A 5 s `git worktree list` ceiling belongs to the
   recovery path, not to the test. Give the product's timer a seam (as #480 did) or
   drive the step with a fake tool that is slow on purpose, and assert what the
   product does when the tool is slow — the refusal, the retry, the preserved state.
2. **The budget is only the test's patience.** Scale it with the repo's existing
   `testwait.Budget(t)` so a loaded host gets a longer window, and keep the assertion
   on the event it was waiting for.
3. **The test needs a tool that may not answer.** `lsof` and zellij IPC can be absent,
   slow, or incomplete. Skip with a named reason when the tool cannot answer, and keep
   one conformance path that *requires* it, so a permanently broken probe still shows
   up somewhere loud.

And for all seven: **the failure message names the host state.** Each of these printed
a bare deadline. A message that carries the load average (and, where cheap, the
process-table row count) lets the next reader separate "the product got slow" from
"the machine was busy" without re-running anything — which is exactly what #644's
cost gate now does for the sampler.

## Done when

- Each of the seven either passes 20 consecutive runs on a deliberately loaded host —
  the load a full fleet produces, with the generator named in the issue — or skips
  with a named reason; and no fix is a raised constant.
- Before each change, a temporary injected delay reproduces the failure the fix
  addresses (#485's rule: prove the race first, then remove it).
- Every one of the seven names the host state it measured in its failure message.
- A full `make test` under that same load gives one verdict for these seven, and the
  run is recorded here with its loadavg.

## Plan

- [ ] couchcore `TestRecoveryFreshOrdinarySlotKeepsOrdinaryLaunch` and
      `TestRecoveryFreshMissingRecordLeavesForeignOwnersUntouched`: the product's 5 s
      `git worktree list` budget — seam or slow fake, and assert the recovery outcome
      when git is slow rather than that git was fast.
- [ ] couchcmd `TestParkHelperAttachmentRequiresClientGrowth` (#645): the zellij
      `list-clients` deadline — fake the action or scale the patience, keeping the
      assertion on client growth.
- [ ] couchcmd `TestClipboardCopyConsoleKeysReachContentFreeFactLog` (#647): find the
      real-time window the evidence check races at 1108/2177 ms elapsed.
- [ ] couchcmd `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees`:
      decide whether the fixture's missing `origin` is the real defect or the
      "signal: killed" budget is — the two failures are in the same subtest and one
      may be hiding the other.
- [ ] couchtty `TestWorkbenchPerformanceIdleUITick`: eight activity-probe deadlines
      then "activity worker did not finish" — check against #626/#628's earlier
      activity-worker timeout work before adding a seam.
- [ ] diagnosticlog `TestOpenFileInspectionConformance` and
      `TestUnregisteredChildOpenFileBlocksLegacyExpiry`: distinguish an absent `lsof`
      (skip, named) from a slow one (patience), and keep a conformance path that
      requires it.
- [ ] broadcast `TestPointerRateLimit`: a rate-limit window measured in real time —
      inject the clock or assert the limiter's decision, not a count inside a
      wall-clock window.
- [ ] Soak: 20 runs of the touched packages under the named load generator, plus one
      full `make test` under the same load, recorded here with its loadavg.

## Log

### 2026-10-17

Filed by pair:4 out of #644's suite attempt 1, at the hub's direction: one issue for
the class, evidence and table included, left unclaimed for dispatch. The hub's ruling
on scope, verbatim: "The class is tests that put a wall-clock budget on an external
tool or real time (git worktree, zellij IPC, lsof, a rate-limit window) and fail at
load near 100." #644's own unchanged rerun is not blocked on this issue — its patch
touches none of these packages — and it will report both suite attempts in its Log.

### 2026-10-17 — pair:7 planning

Claimed unowned card; start-plan completed after dependency refresh and explicit retry. No package installation. Read all named tests with two independent read-only subagent investigations. Seven groups contain nine functions; park growth is in couchcore, already fake-backed. Clipboard's exit7 assertion conflicts with valid product1s timeout, while continuation's missing-origin warning is explicitly nonfatal. Current measured load `{ 60.65 65.52 86.42 }` is context only, not a reproduction count.

Durable plan: [test budgets](../plans/000659-test-budgets-plan.md). No implementation yet; send to hub before code. #576 preempts immediately on hub's #651-landed announcement; preserve this issue with a clean committed checkpoint.

### 2026-10-17 — delay controls and corrections

Hub approved plan (32b8b32e-f83d-4891-b47d-96437f6ceff6); change-code admitted quick flow. Product constants unchanged. All seven old delay controls reproduced their intended assertion failures before their correction. Temporary controls removed from source after corrected evidence:

- Pointer: 40ms per HTTP post refilled the bucket, old test limited0/50. New frozen logical clock still limits exactly20 with that same delayed transport, then refills after logical time advances. Normal test has no sleeps. Logs `/tmp/pair659-pointer-{old-delay,frozen-red,frozen-delay-green}.log`.
- Recovery: delayed exact worktree-list failed the old ordinary Fresh fixture at unchanged5s. Named ownership tests now use stateful fake catalog before discovery; real catalog timeout/error and no-launch/no-repair/retained-file preservation checked separately. Four foreign/ordinary cases and timeout regression pass (`/tmp/pair659-recovery-final.log`, package2.480s). Initial fake fixture omitted required workspace identity fields; targeted tests caught it, fixed before green.
- Clipboard: stalled private fake caused old exit7 assertion failures across all three key encodings. Corrected typed timeout/fallback/executable/privacy checks pass; exact exit/stderr tests remain in clipcmd.
- Activity: context-aware2300ms delay lost exactly one row under product2s; same delay with existing test budget seam passed all cold/unchanged/mutated counters. Delay removed.
- Park: fake reply held2s exceeded old caller1s. Corrected test-only gated query waits for real growth; unchanged/unknown responses remain negative. Native default query wrapper and fake CLI contract retained.
- Continuation: private invocation held21s exceeded old parent20s and was killed; same delayed control with testwait patience passed exact publication. All five final modes pass.
- Lsof: old delayed native wrapper failed at2s (measured load94.12). New optional mode skips explicitly, required mode fails, complete wrong holders fail both. Fourteen policy controls pass. Real required conformance passed both native tests (1.95s/2.18s); `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair659-lsof.uR1i9R/evidence.txt`.

Detailed clipboard/activity and park/continuation evidence: `/tmp/pair659-io-evidence.md`, `/tmp/pair659-tools-evidence.md`. Copy of historical suite retained at `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair659-evidence.pxXt0x/historical-suite.log`, SHA256 `0c150ba9b316c2e550901cd7e36e9021ccbb824f5bfaddd80844e01a3fb1adf0` matching issue evidence. Preliminary compile failures from concurrent test-file edits were discarded, never counted as behavioral evidence. Loaded repetitions and recorded full suite still pending.
