---
id: 000873
status: done
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '4e733b35598f37daf6992a6c333e3d109f6767df' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T09:26:10-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/couchcmd.TestFreshStartupRunSerializesLaunchThroughAdoption]
flow: {kind: quick, provenance: inferred, spec: "9c4eb616", done: "fdf57cfe"}
actual_hours: N/A
---

# flake: TestFreshStartupRunSerializesLaunchThroughAdoption

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcmd.TestFreshStartupRunSerializesLaunchThroughAdoption: Failed in sdlc merge suites on 2026-10-24: PR 590 head 8747012d (log .git/sdlc/test-stats/failed/da7653e44c27a012b3893c4378e7053c.log) and the main baseline 2b9334fd (log .git/sdlc/test-stats/failed/668931af27a5603e851eae1f4d974e35.log). Both: fresh_startup_run_test.go:199: startup failed: lock thread store: resource temporarily unavailable; no other test failed. sdlc labelled it a persistent main failure. Registered on TL ops:0 instruction so pair merges can proceed; TL gives the root cause to pair:7 (author of the #863 fresh-startup test).

## Spec

Fresh startup is a foreground empty-only reservation, but withEmptyStartupTarget passes a nonnil check and zero wait at both global/local store locks. withStoreLockChecked therefore selects LOCK_NB, the maintenance yield policy. Console inventory previews legitimately hold store.lock outside the retention coordinator; serial launch/adoption does not exclude those reads. A later hub can fail before its runner starts, matching the retained bench-only effects.

Reuse the existing foreground writer policy (storeReadLockWait) at both checked acquisitions, preserving the custom before/after journal refusal, coordinator→global→local order and empty-only publication. No new retry loop, lock, timeout value, transaction retry, journal replay or general replacement authority. This selects the correct existing operation class; it does not raise a budget. ARCH-DRY/ARCH-PURPOSE: cover both lock sites and preserve maintenance's zero-wait control. Root-cause proof uses held kernel locks and the existing contention observer, without a production test hook.

No new user state: legitimate brief contention remains the existing pending start; cancellation, typed budget exhaustion or newly appeared history refuses before publication. Recovery stays the existing explicit operator start after inspection; no history or child is lost/created on refusal. Scope is the startup lock policy and its deterministic regression, not unrelated store-lock redesign.

## Done when

- A held global or local preview lock deterministically reproduces the old refusal; the corrected empty-only foreground acquisition waits through release and publishes once.
- Cancellation, typed exhaustion and history/journal appearing during admission refuse before mutation; maintenance still yields immediately. N/2N contention/publication counts remain linear with no duplicate effects.
- The original real-Run test and surrounding startup/foreground-lock safety checks pass; pre-close, mandatory review and merge pass. Closing this issue unregisters its registered flake. No new loop or longer timeout.

## Plan

- [x] Add deterministic held-lock regressions around withEmptyStartupTarget using existing kernel-lock and contention helpers; establish RED before modifying production.
- [x] Select existing foreground wait at the global and local acquisitions; verify cancellation, exhaustion, late retained evidence, maintenance and counted N/2N effects plus the original Run case.

After implementation qualification: push, run the recorded pre-close, close with binary-owned review, enter normal merge on SHIP and report the actual landing. No manual main chase after the branch check.

## Log

### 2026-10-24
- 2026-10-24: closed — Held kernel-lock RED/GREEN at both startup sites with cancellation, typed exhaustion and late retained evidence; maintenance control preserved. N=2/4 counts linear, old-policy overlay fails all six cases. Targeted core PASS17.973s; original Console.Run 1/50 failures before vs50/50 PASS96.407s after (not calibrated rate). Inventory PASS immediately before close; atlas updated. Recorded affected lease-1096961115 remains queued on runtime c6644e8d; current descendant adds only Log and refusal evidence. Require recorded pass before landing. Normal actual refused measurement snapshot exceeds65536bytes, exact receipt workshop/evidence/000873/measurement-refusal.txt; --no-actual authorized by standing TL f039e10f / ariadne#593. Hub owns projects. #854 second lander applies context cap to both calls. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

### 2026-10-24 — root trace and bounded measurement

Claimed/start-plan from main after #871 landed. Retained branch and main-baseline logs both stop at bench launch/adopt/adopted with raw EWOULDBLOCK before next runner entry. Read-only trace found the two wait=0 calls in startup_creation.go; threadstore.go explicitly documents nonnil check plus zero wait as maintenance-style immediate refusal. Preview reads do not take the coordinator. Existing #827 withUpdateLockContext already uses the established foreground budget. A bounded one-process50-repeat of the unchanged original test is running as baseline evidence; deterministic held-lock controls, not its rate, will prove causation. No production changes yet. This is a semantic foreground/maintenance classification error; a blanket lint banning zero-wait checked locks would reject valid maintenance. Existing maintenance negative control pins that distinction.

### 2026-10-24 — RED evidence and composition

The unchanged real-Run test failed once in 50 repetitions (102.680 s, one process), with the same raw store-lock refusal and bench-only effects; retained in workshop/evidence/000873/baseline-50.log. The new held-kernel-lock regressions then failed at both acquisitions: global/primary and global/local-slot cases, N=2/4, cancellation, typed exhaustion and journal recheck. The observer could not run because the old code refused immediately. These deterministic failures establish the policy mismatch; the small cohort is not a calibrated flake rate.

## Revisions

### 2026-10-24 — approved foreground policy and #854 composition

Hub approved reuse of storeReadLockWait at both sites, preserving all admission guards. #854 is unlanded and adds a remaining-context cap through withStoreLockBounded/storeLockBound. No stacking or copying its helpers: whichever issue lands second applies that cap to both startup acquisitions. Pair:6 explicitly confirmed it will integrate the two calls if #873 lands first.

### 2026-10-24 — GREEN and negative control

Changed only the two foreground policy arguments plus their explanatory comment. Targeted core checks passed (17.973 s): held global/local acquisition at N=2/4, cancellation, typed exhaustion, current/archive/journal appearing while waiting, existing empty-only admission and foreground/maintenance controls. A compiler overlay restoring both zero arguments failed all six held-lock cases with the original raw refusal; retained in negative-policy.log, with actual source unchanged. The original unchanged Console.Run test then passed all 50 repetitions (96.407 s; corrected-50.log), versus 1/50 failures before. This is supporting comparison, not a stability-rate claim. make test-affected-inventory passed; the new test reads only fixture-owned stores, without repository reader inputs to certify. Atlas now states the existing foreground admission policy. #854 remains unlanded at its announced head; the agreed second-lander composition still applies.

### 2026-10-24 — recorded check and actuals checkpoint

Pushed runtime head c6644e8d0ef6b81e7920e902d5cd7b592ad6c200. Pre-close selected affected, run lease-1096961115, attempt9b1291dfa24852860d0188feb71beb0f, currently in normal FIFO; preserve this runner. Normal close refused before review because measurement exceeds65536bytes (exact receipt in measurement-refusal.txt). Apply only --no-actual under standing TL authorization f039e10f / ariadne#593; no guessed hours or evidence rewrite. Next: join the current check, finish binary close review, then merge on SHIP; the train owns integration.

### 2026-10-24 — park waiting merge for operator #842 correction

Hub9c9abb53 reports #842 live failure and requests its fix/re-look. The retained #873 merge pid4139 was still waiting behind #869, not admitted; sent it SIGINT and joined tool session57819 exit130 before switching this sole checkout. Recorded affected lease-1096961115 (tool91503) remains alive in its detached checkout/FIFO and is not cancelled or duplicated. SHIP and runtime unchanged; after its pass and #842 correction handoff, return here and re-enter sdlc merge --yes. No landing is claimed.

### 2026-10-24 — landing priority corrected

Hub63f445fe clarified #873 must land before the non-urgent #842 re-look. The previous waiting runner had already joined exit130 when this arrived; returned immediately to #873 and re-entering merge, without cancelling/restarting its recorded check. #842 read-only trace found its outer client endpoint does not address the inner draft; no code or test edits made there. This supersedes the prior park instruction; next action is finish #873 landing first.

### 2026-10-24 — pre-close sole #869 failure and authorized recovery

Recorded affected lease-1096961115/attempt9b1291dfa24852860d0188feb71beb0f completed all selected checks in14m2s, FAIL solely on agentfixture.TestCLILiveLeaderCancellationJoinsBeforeCompletion: cli_group_test.go83 parses an empty PID marker. Core/startup and all other packages passed. Saved failed log .git/sdlc/test-stats/failed/9b1291dfa24852860d0188feb71beb0f.log, SHA prefix38851f52d435. Existing #869 is precisely this publication race, with no diff in its package from this source. No new issue or production change.

Hubc8becfbf authorized exact-head targeted evidence, register under869, then resume the saved attempt. A temporary git archive of exact failed head c6644e8d0ef6b81e7920e902d5cd7b592ad6c200 passed the unchanged top-level test (0.325s, exit0), log869-same-head-focused.log; private archive removed. This is explicitly a focused pass, not a full check pass. PR594 opened through sdlc. Previous merge runner joined exit1 on pending CI and missing recorded pass; no runner remains to duplicate. Registration/recovery is next, then CI and merge; no SHIP or gate bypass is implied by the focused test.

### 2026-10-24 — saved-attempt resume admitted

Registration of the single test under869 succeeded using the honest fail/focused-pass evidence authorized by Hubc8becfbf; notified869 owner to retain it in that issue Log. Supported local-check resume accepted exact parent9b1291dfa24852860d0188feb71beb0f at c6644e8d and created child82d1856f1cf35eeb1b90f18c122bdc14, lease-138335625 (tool77994). It reruns only known869, but currently requests a machine suite slot, FIFO position10 at admission; reported this economy surprise to hub, no bypass or cancellation. Preserve the admitted resume. PR594 is open and CI pending. Next: join tool77994, inspect its exact result; if pass, re-enter merge (no live merge runner currently) and follow its printed prerequisites. No unchanged full rerun.

### 2026-10-24 — saved resume PASS; re-enter landing

Supported resume tool77994 joined exit0. Parent9b1291dfa24852860d0188feb71beb0f plus child82d1856f1cf35eeb1b90f18c122bdc14 now records affected PASS at runtime c6644e8d, with the single known #869 rerun; no full rerun was requested. The display retains the parent failure's exit2/14m2s as provenance, while the resume command and combined classification are pass. All runtime files remain identical to that tested commit; later commits are close, Log, evidence and lesson only. Re-enter sdlc merge --yes after #878/#869 per hubd8b9652a; no other merge runner is alive.
