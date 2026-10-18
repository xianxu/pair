---
id: 000659
status: codecomplete
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-18
estimate_hours:
severity: high
collaboration-mode: collaboration
card_mirror: 'c1be28ad2bd797b32df61f8bfdf39b0b4e8fc17c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T22:43:28-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "abd9131f", done: "22b0c546"}
actual_hours: 1.07
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

- Each of the seven either passes20 consecutive runs (all subtests included) or
  skips with a named reason. Record actual concurrent fleet work and measured load;
  the observed-load cohort supports the primary injected-delay proof. No fix is a
  raised constant, and no target load number or additional load campaign is required.
- Before each change, a temporary injected delay reproduces the failure the fix
  addresses (#485's rule: prove the race first, then remove it).
- Every one of the seven names the host state it measured in its failure message.
- A corrected-head recorded full suite passes. Record its loadavg separately from
  the repetition cohort; the two runs need not have matching loads.

## Plan

- [x] couchcore `TestRecoveryFreshOrdinarySlotKeepsOrdinaryLaunch` and
      `TestRecoveryFreshMissingRecordLeavesForeignOwnersUntouched`: the product's 5 s
      `git worktree list` budget — seam or slow fake, and assert the recovery outcome
      when git is slow rather than that git was fast.
- [x] couchcmd `TestParkHelperAttachmentRequiresClientGrowth` (#645): the zellij
      `list-clients` deadline — fake the action or scale the patience, keeping the
      assertion on client growth.
- [x] couchcmd `TestClipboardCopyConsoleKeysReachContentFreeFactLog` (#647): find the
      real-time window the evidence check races at 1108/2177 ms elapsed.
- [x] couchcmd `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees`:
      decide whether the fixture's missing `origin` is the real defect or the
      "signal: killed" budget is — the two failures are in the same subtest and one
      may be hiding the other.
- [x] couchtty `TestWorkbenchPerformanceIdleUITick`: eight activity-probe deadlines
      then "activity worker did not finish" — check against #626/#628's earlier
      activity-worker timeout work before adding a seam.
- [x] diagnosticlog `TestOpenFileInspectionConformance` and
      `TestUnregisteredChildOpenFileBlocksLegacyExpiry`: distinguish an absent `lsof`
      (skip, named) from a slow one (patience), and keep a conformance path that
      requires it.
- [x] broadcast `TestPointerRateLimit`: a rate-limit window measured in real time —
      inject the clock or assert the limiter's decision, not a count inside a
      wall-clock window.
- [x] Soak:20 runs of the named functions, with actual concurrent work and load
      recorded, plus a passing recorded full suite with its own measured load.
      Injected-delay controls are primary proof under the hub-approved Revision.

## Log


- 2026-10-18: closed — Nine reported functions and all subtests passed20/20, plus recovery deadline20/20; deterministic old delay controls fail and corrected controls pass; required native lsof passed. Hub accepted measured-load support. Corrected recorded full suite on75277904 passed15m29s, log bffa54097006; first failed suite retained with only inventory omission, fixed and focused guards green. BR-1/2/3 addressed solely in plan/acceptance/README documentation; no executable sources changed since passing suite.; review verdict: SHIP
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

### 2026-10-17 — repeated targeted verification

Implementation head `193e8727c05e751ea3cfe465404f69c52a1b9f5d` passed the combined targeted run (`/tmp/pair659-targeted-all.log`) across testwait, broadcast, couchcore, couchcmd, couchtty, diagnosticlog and clipcmd. It then passed all nine originally named functions20/20 each, all subtests included, plus the new recovery deadline regression20/20. No failures or skips. Command: `go test ./cmd/internal/broadcast ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/diagnosticlog -p 2 -run '^(TestPointerRateLimit|TestRecoveryFreshOrdinarySlotKeepsOrdinaryLaunch|TestRecoveryFreshMissingRecordLeavesForeignOwnersUntouched|TestRecoveryFreshCatalogDeadlinePreservesRetainedState|TestParkHelperAttachmentRequiresClientGrowth|TestContinuationWriterPublishesExactCheckpointAcrossWorktrees|TestClipboardCopyConsoleKeysReachContentFreeFactLog|TestWorkbenchPerformanceIdleUITick|TestOpenFileInspectionConformance|TestUnregisteredChildOpenFileBlocksLegacyExpiry)$' -count=20 -timeout=30m -json`. Log `/tmp/pair659-soak20.jsonl`, SHA256 `c3e6f4ad92491b79218716da4c05064d55507109ec32d93c01678f4a86eb3649`; slowest package couchcmd225.073s.

Load was ordinary concurrent fleet work: gate-owned ariadne suites from ariadne:1 then ariadne:2, plus these two-package targeted repetitions. Sampled load averages declined from `{ 50.07 55.86 68.87 }` to `{ 39.58 51.43 66.16 }`, `{ 34.70 46.97 63.12 }`, then `{ 30.12 42.32 59.80 }`. This is **not** a20-run claim at load100. The earlier delayed native control measured94.12; that is a different cohort. Our own recorded suite was queued throughout, initially sixth then fifth; it has not run yet. No artificial host load burner or production timeout change.

Recorded suite attempt is queued on the same implementation head: `sdlc local-check run --name suite`, inflight log `.git/sdlc/test-stats/inflight/db25bcb0f104165571bc9f60bcde47e2/run.log`. Full-suite verdict, close and landing remain outstanding. #576 priority still applies.

### 2026-10-17 — full-suite inventory finding

The recorded suite acquired its slot after1089s. All shell/native-editor/review prerequisites passed, including observation diagnostics. Go's exhaustive `TestProductionArtifactReferencesAreExactlyClassified` found a branch defect: new `cmd/internal/testwait/host.go` lacked source classification. Added that exact file beside the existing testwait helper in `NonArtifactSources`: host-load diagnostics construct no Pair session artifact. No vocabulary allowance or guard bypass. The original failing guard plus `TestNonArtifactSourceCannotImportArtifactpath` now pass (`/tmp/pair659-inventory-fix.log`,4.533s). Added the prevention rule to lessons. The first suite continues to completion to collect all other failures; it will not count as a passing record, and the corrected code needs a fresh recorded run. Measured full-suite start load `{ 19.70 31.67 43.92 }`.

The first suite completed exit2 in13m16s, log SHA256 prefix `5429b2148e38`: **only** the inventory guard failed. All other packages passed, including couchcmd137.455s, couchcore372.809s, couchtty52.271s and diagnosticlog41.168s. Full failed log retained under the earlier evidence directory as `suite193e8727-failed.log` as well as SDLC's saved attempt. This is a corrected branch defect, not a flake waiver. New recorded suite on corrected head `75277904ed4f07d4914d34cd4a6bb3a65b119e2c` is queued (attempt `c5ff20b0a514c353698bd714be553db5`); its CI checks all pass, including conformance2m45s. Sample at queue entry `{ 14.81 11.88 23.18 }`.

### 2026-10-17 — corrected recorded suite passes

Attempt `c5ff20b0a514c353698bd714be553db5` acquired after711s and **passed exit0 in15m29s** on `75277904ed4f07d4914d34cd4a6bb3a65b119e2c`, recorded as `sdlc-local/suite`; log `/Users/xianxu/workspace/pair/.git/sdlc/local-checks/75277904ed4f07d4914d34cd4a6bb3a65b119e2c/suite.log`, SHA256 prefix `bffa54097006`. The inventory guard passed4.601s, couchcore422.989s, couchcmd76.837s, couchtty49.638s and diagnosticlog52.973s. All prerequisites and Go packages completed. Load sample after acquisition `{ 24.12 31.50 31.92 }`, completion `{ 11.88 16.65 22.88 }`. Only documentation changed after that source head. Draft PR420 is open; close review is next. No flake waiver or manually posted status.

## Revisions

### 2026-10-17 — measured-load acceptance

Hub accepted the observed cohort (7b1e37d3-9f5e-4f37-9565-fd234eea0376): injected-delay controls are the primary proof because they reproduce each old failure and show the correction removes that dependency;20 repetitions at the recorded real loads support that proof. Do not chase load100 or run another cohort merely for a larger load number. Record actual loads as above. Land after the recorded suite and normal close/merge gates, subject to #576 preemption.

### 2026-10-17 — close review BR-1/BR-2/BR-3 documentation synchronization

Promote the optional/required probe-verdict helper from the plan's PURE row to test integration, matching its actual environment/diagnostic/testing effects. Synchronize Done when and both plans' load clauses with the already-approved acceptance; no new scope or verification waiver. Preserve the original superseded load clauses here:

> Each of the seven either passes20 consecutive runs on a deliberately loaded host — the load a full fleet produces, with the generator named in the issue — or skips with a named reason; and no fix is a raised constant.
>
> A full make test under that same load gives one verdict for these seven, and the run is recorded here with its loadavg.

README's testing guidance now documents the sole new operator-set test switch, `PAIR_REQUIRE_LSOF_CONFORMANCE=1`, including fresh invocation and optional-skip versus required-failure behavior. No production or test source changes after the passing recorded suite. Re-review will dispose BR-1 through BR-3; none is being bypassed.

### 2026-10-18 — landing recovery and main conflicts

First integration suite b6ecb5a8c0c61c236def180e67ec41cc passed exit 0 in 11m3s on b937f71654bfec9e2f8684cbd4c88a57713bda2b (log SHA256 prefix ecfc463ae17a). A subsequent main integration to 46e0bd0379ccf8ccabb972fa3b62aea2a9a36871 stopped on native CI run39111981159: independent Node/xterm oracle SIGSEGV at terminal_native_test.go:345, after the wrapped child retained its original boot identity and counter advanced to2. Filed separate unassigned #668, retained /tmp/pair659-ci-39111981159.log, distinguished from #660 discovery absence. One unchanged failed-job rerun (attempt2) passed; no code/deadline change.

Resumed normal sdlc merge, which refused newer-main conflicts in README, park_client_readiness_test.go and diagnosticlog/proof_test.go. Manual resolution retains both README contracts; keeps main's shared typed native fixture replies in other park tests while preserving this issue's explicitly gated semantic growth test; keeps procutil import for lsof holder identity checks. No assertion or product deadline removed. Targeted checks and the required full recorded suite/re-close follow before another landing attempt.
