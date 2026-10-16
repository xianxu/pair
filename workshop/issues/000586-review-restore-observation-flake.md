---
id: 000586
status: codecomplete
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '2eafcb02c3fb1d6899874b7d6355379d3d7d9630' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T19:17:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "017f15c1", done: "76b4d5e2"}
actual_hours: 2.19
---

# Diagnose intermittent review restoration observation mismatch

## Problem

`tests/review-observation-test.sh` intermittently refuses its repeated branch restoration during a full suite, with insufficient evidence to distinguish a stale observation, a timed-out observation, and an actual branch mismatch. This blocks unrelated #571 verification; the adapter changes none of the review observation/controller/test code.

## Spec

Reproduce and diagnose the first restore identity check without weakening authorization. The test's `resolve()` forwards the readiness JSON without asserting its status. The readiness CLI can return invalid status with exit zero, and identity resolution has a shared two-second Git budget. Under load an incomplete wanted observation may reach the controller, whose fresh observation then reports a generic mismatch. This is an unproven hypothesis: the failing log retains neither identity. Capture both wanted and observed status/identity/diagnostic, and distinguish malformed/incomplete evidence from genuine changed-branch refusal. No sleep or larger timeout as a substitute for root cause.

## Done when

- A deterministic regression identifies the failing observation boundary and exercises incomplete/stale input.
- The test validates resolver status before forwarding identity and retains actionable failure evidence.
- Real branch changes still refuse restoration; repeated isolated and loaded suite checks pass with measured results.

- If the original flake does not reproduce within about one hour of loaded runs, retain the measured observations, ship the diagnostics with deterministic regressions and a positive control, and move to #589 as explicitly directed by the hub. Do not claim an unmeasured original cause.

## Plan

Approved quick-flow investigation and fix. Keep #586 separate from #591 until retained evidence demonstrates a shared cause; continue #589, #591 and #590 in the agreed order after this item. Expected small boundary change; if demonstrated scope leaves the shell, revise and use a durable plan before implementation.

- [x] Make the fixture fail diagnostically at its observation boundary: validate resolver exit, JSON shape, resolved status and complete identity before forwarding; preserve wanted and fresh status/repo/branch/head/file/diagnostic, command phase and completion ordering on refusal. Reuse the existing isolated fixture wrapper; bounded metadata only, no copied snapshots (ARCH-DRY/CONSTRAINTS).
- [x] Establish deterministic RED with controlled invalid/incomplete wanted and fresh observations at that wrapper/controller seam. Preserve genuine changed-branch refusal and a positive same-identity restoration. Require actionable evidence and no activation on refused observations; no sleeps or altered assertions (ARCH-PURE/ORDER/SECURE).
- [x] Reproduce under the suite's parallel load while changing one variable at a time. Record both observations before selecting the fix. In particular, distinguish the fixture forwarding invalid exit-zero JSON from actual branch movement and the resolver's shared two-second Git deadline. Append a Revision with demonstrated cause and the minimal owning-boundary change; ask the hub if choosing a new fixture time authority or broader production scope is required. Do not turn a timeout into successful identity, retry until green, or enlarge any budget (ARCH-PURPOSE/MOCK).
- [x] Implement the demonstrated fix with regression evidence, preserving production authorization and bounded process ownership. Use event-driven completion or an injected fixture clock if scheduling is the cause. Temporary evidence dies with the existing fixture owner; dump relevant diagnostics before cleanup (ARCH-FUNERAL). Run targeted/loaded checks, make -k build, required recorded suite, atlas update as applicable, then mandatory close review and SHIP landing.

## Log

### 2026-10-15 — first observed during #571 unchanged rerun
- 2026-10-15: closed — Approved diagnostics-only fallback: 202 fresh loaded cases passed in 60.13min, original initial-request cause unproven; retained separate async 50ms WaitDelay lead. Eight deterministic diagnostics tests and positive fixture passed. make -k -B build passed; exact-source a0b3fb0de67824b57d0f42cedd32acd02a7d1778 required suite a434158b4ff3c9fcc7da7a10fc5eedff passed13m19 SHA2562cfda2445836f4ef85c8c6744d781ddbf4ace733dbcba4ecb4b5e72067e293cf. One authorized unchanged rerun after independent startup-cancellation flake #598; no fixture fix, timeout increase, retries in reproduction or skips. Atlas updated. Merge held for hub go after #597/#571. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

- Exact tested head `8bd4baa138c63cda357c3a447ddb2cba294f1df7`; run `119840be5020067f99a3413fd951e608`.
- Log `/Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/119840be5020067f99a3413fd951e608.log`, SHA256 prefix `58ed1e657115`; suite stopped at test-review after 4m33s, before Go packages. A keep-going run was started to complete remaining checks.
- Python line55 in the repeated checkout/request loop: `AssertionError: {'error': 'branch changed during restoration; invoke Alt+C again', 'ok': False}`.
- Read-only source audit locates the first gate at `nvim/review/restore_controller.lua:137-139`, before activation/confirmation. A controller invalid observation normally preserves its own diagnostic; the generic text is consistent with an incomplete wanted observation but does not prove it.
- Unchanged isolated `bash tests/review-observation-test.sh` passed exit0 on the same code (subsequent branch changes are issue documents only). Earlier full suite run `b2dd3d164d5caad0fd4f624661697ade` passed this test and failed only the independent #585 broadcast test.
- Filed unassigned as independent verification followup; do not fold a review-controller fix into #571.

### 2026-10-15 — claimed after #585 landed

#585 landed via PR #357 at git-verified merge27a12fec10e7df7df86e5a5fb7a4f2a2f655b199; hub outcome receipt5a62e02b-0d83-46dc-bdaf-8549cb39b517 is submitted. Started #586 from main archive6f0e8c656bcfebc2051e85a8d0eb1c8e7366825b; claim and start-plan succeeded. No implementation changes or new test run for #586 yet.

Read-only audit: Python resolve() decodes check_output without checking status; RunReadinessCLI --resolve returns exit0 for a successfully encoded invalid observation. resolveIdentityWithSnapshot shares a real two-second context across root/index/branch/HEAD/history/integrity operations. C:request checks only wanted's table/repo shape before resolving again; its generic changed-branch refusal can compare a partial wanted identity against a fresh resolved identity. The observed historical generic message is consistent with that ordering, but neither original observation was retained, so it remains a hypothesis. Both #585 source suites passed the unchanged observation fixture; those green samples do not refute the recorded intermittent failure.

Alternatives: immediately tightening only the wanted-status check improves evidence but does not establish or eliminate the scheduling flake; replaying a fake resolved identity would remove the production Git deadline but risks bypassing the behavior under investigation. Recommend diagnostics first, deterministic invalid/stale refusal cases, then a measured Revision at the actual time/observation boundary. Keep authorization fail-closed and use the fixture's actual state rather than a larger wall-clock budget.

### 2026-10-15 — diagnostic boundary RED/GREEN

change-code passed quick flow before code edits. Shared wanted helper initially kept the existing check_output/json.loads behavior: deterministic exit-zero invalid JSON status was forwarded, so the regression failed with “AssertionError not raised” (0.023s). It now captures one real subprocess outcome, validates JSON object/resolved status/complete identity, retains phase/order/returncode/diagnostic and rejects before forwarding. Three boundary tests passed (0.156s). A later real-process already-expired-budget regression found command payload leakage through TimeoutExpired stringification; RED retained the sentinel text, fixed structured timeout metadata omits arguments, and all four wanted tests passed (0.162s). No document snapshot or review body is retained in traces.

Fixture-only Neovim shim observes actual vim.system wait and callback boundaries, preserves the same process object/options/results/inline callback behavior and exceptions, and records bounded metadata in memory (256 recent events, dropped count, sequence/request/call IDs and monotonic timing). It adds no subprocess, timer, scheduling callback or hot-path disk write. It captures wanted requests and both fresh resolution/confirmation calls; optional VimLeavePre fallback flush is owned by the existing fixture temp tree. Shared wrapper’s real exec/slow/hold behavior stays unchanged. Python failure output obtains the live trace before teardown and retains the pane returncode/log. Optional PAIR_TEST_OBSERVATION_REPORT retains successful probe evidence outside the temp tree for the timeboxed experiment.

Delegated only the isolated Lua tracer and its standalone test to observation_trace_seam. Agent reports semantic RED/GREEN for wait exceptions, monotonic metadata and table request handling; standalone test passes, including a real SystemObj echo control, async/late callback correlation, bounded retention and one final flush. No other files or suites changed by the agent. Main end-to-end negative controls inject incomplete fresh evidence and perform a real checkout between wanted and fresh; both refuse without changing activation or buffer. All six initial diagnostics tests passed (2.350s); original positive observation fixture passed. Adding success-report retention produced a deterministic RED after a passing fixture (missing report,15.015s); full seven-test group then passed (16.401s), including matching wanted/fresh identities and no copied snapshots/body. Existing fixture clocks, latency assertions and deliberate slow/held effect models are unchanged; no time budget increased.

Original historical cause remains unproven. Ready for committed-source build, recorded suite and bounded loaded sampling, with a Revision required before any root behavioral fix. #585 is landed and reported; #579 live IO sample remains deferred to the hub-provided post-restart PIDs/build.

### Loaded reproduction checkpoint — 2026-10-15T19:44:11.873180-07:00

Committed sourcef6512608053de4d07e29acf206d870b31d169b51 rebuilt with make -k -B build, pushed, and required recorded suite7ce208b67b715382a02adcd7ffaacb6c is actively executing (not queued). Baseline one-case probe with no additional owned full suite passed17.574s: five wanted and41fresh observations all resolved, max wanted439.447ms, max fresh1169.286ms including the existing1s slow-wrapper model. Baseline host load varied4.87–9.45, so this is not an idle-host claim. Evidence root /private/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair586-repro.nyUWcD.

Started a one-hour, one-fixture-at-a-time experiment with the owned recorded suite plus ambient fleet load; each case is fresh isolated storage, same source/environment, no retries after a failure and no sleeps between cases. /tmp/pair586-load-probe.py preserves per-case stdout/stderr/observations and summary.json under the loaded subdirectory, records elapsed observations and host load, and stops at the first failure or the timebox between cases. The actual owned-suite overlap ends when that suite completes; later samples must not be presented as overlapping it. No production observation or authorization behavior has been changed.

### Early loaded evidence (not an original-cause claim)

All eight diagnostics tests passed22.128s on the committed diagnostics source; the recorded suite also passed the same eight tests27.621s and its unchanged positive observation fixture. At the eight-minute loaded checkpoint,28fresh isolated cases passed. Max wanted401.603ms; max fresh1810.347ms includes existing deliberate slow-wrapper modeling. All wanted identities and initial restoration acknowledgements remained valid.

Loaded case8 retained a real asynchronous invalid exit-zero observation: status invalid, branch review/b, diagnostic “checkout HEAD changed during resolution”, mode callback, timeout2500, elapsed1810.347ms, no active request ID. The fixture passed; this is an expected refusal across an intentional checkout, not the original initial-request mismatch or proof of a deadline expiry. Its full evidence is loaded/run-0008/observations.json under the experiment root. The one-hour experiment and required suite continue; no failed case has been retried and no production fix has been selected.

### Required diagnostics-source suite PASS

Recorded suite7ce208b67b715382a02adcd7ffaacb6c onf6512608053de4d07e29acf206d870b31d169b51 completed exit0 in14m1s. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/f6512608053de4d07e29acf206d870b31d169b51/suite.log; SHA2561c30fcf198d46bbc5b21bdc564b551e397f2c2409cc9b16daf2a43ceb2fbcf31. New eight-case diagnostics group, unchanged positive observation fixture, other editor tests and all Go packages passed. No source change since; later commits are issue Log only.

The last suite-log write is2026-10-16T02:57:30.132529Z; confirmed command exit0 by02:57:53Z. Experiment root now includes owned-suite-finished.json so later samples can be distinguished from the true full-suite-overlap interval. At that point44loaded cases passed. The experiment continues until its one-hour boundary or first failure, under ambient fleet load after the owned suite ends; no future sample will be claimed as still overlapping that finished suite.

### Half-hour reproduction checkpoint

At 31.48minutes, 110completed fresh probes passed; failures0. Max wanted 433.950084ms; max fresh 1810.347417ms including existing slow/held effects. Source remainsf6512608053de4d07e29acf206d870b31d169b51 (only issue Log commits since); the required suite already passed14m1s. No initial-request failure or demonstrated deadline expiry has been retained. Expected async invalid statuses include writer-held index, staged/index transition and obsolete selection receipt refusals across deliberate checkout; all observed examples are callback completions without active request IDs. OSRuntime.GitContext explicitly sets GIT_OPTIONAL_LOCKS=0 and IndexGeneration observes the writer lock without acquiring it, so these samples do not show resolver-created index locks. No root fix selected.

Active probe command is /tmp/pair586-load-probe.py, tool session59763; it stops on first failure or at its3600s between-case bound. Started2026-10-16T02:44:10.698010Z, so the expected final boundary is about03:44:11Z (20:44local), plus the final case. Evidence root and finished-suite marker above remain authoritative; do not count later samples as overlapping the owned suite. Continue the approved diagnostics-only fallback if the original case remains unreproduced.

### Final timeboxed evidence — approved diagnostics-only outcome

The one-hour experiment completed 202 fresh isolated cases, all exit0, in 3607.575s (2026-10-16T02:44:10.698010Z–03:44:18.304356Z), without retries or failed cases. Retained 1010 wanted observations, 8349 fresh completions and 808 successful requests; no trace events dropped. All requested synchronous observations were valid. Max wanted1682.578ms and fresh1900.854ms include existing intentional slow/held effects. Host one-minute load ranged6.60–42.45. Only44cases ran fully inside the owned recorded suite, case45 crossed its end, and157ran afterwards under ambient fleet load. The original initial-request failure did not reproduce. Evidence: experiment root above, loaded/summary.json SHA256045af301c1cc52c8c92fa4fbf4856e88d672e49a43e81e5e4fa7b1de414b144e, aggregate analysis.json, per-case observations/stdout/stderr.

Retained asynchronous refusals are65 index-lock,20 obsolete-selection-receipt,8 staged/index-transition,1 changed-HEAD and1 WaitDelay expiry. All have no active request ID; no failed restoration was hidden. Case147 contains invalid exit-zero callback evidence after619.251ms: “cannot resolve branch (detached HEAD or Git failure): exec: WaitDelay expired before I/O complete”, with empty branch/head/file. OSRuntime.GitContext (cmd/internal/reviewcmd/runtime.go) sets cmd.WaitDelay=50ms while exec owns stdout copying. This demonstrates a real drain-budget expiry, not expiry of the shared two-second context. It is a useful lead for a future historical reproduction, but does not prove the original failure had this cause. No production change selected.

Integrated origin/main a1beae74b after the experiment; merge completed cleanly. Required build and recorded suite will be rerun on the integrated source before close. Plan investigation/fix checkboxes are satisfied through the explicitly approved fallback below, not a claim that an unobserved production scheduling cause was fixed. #589 now has pair:2's claim; do not collide with it when continuing the batch.

### Integrated suite blocked by an independent startup-cancellation fixture

Required suite501ea461dee91eaad8c9ae484cbfa18d on a0b3fb0de67824b57d0f42cedd32acd02a7d1778 waited780.079s, then failed exit2 after188.925s, before any review integration fixtures. LifetimeTests.test_cancellation_during_startup (SIGINT,delay0.02) in tests/without-ancestors-test.py:103 timed out waiting3s for helper exit; five methods ran4.850s. Both helper and test are unchanged from main. Log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/501ea461dee91eaad8c9ae484cbfa18d.log, SHA256ddf72cf3b67ba81c2891b26c021302619c0d9120ac6b9d17f401e525646a337a. This is a failed suite, not a passing #586 verification; the previous pre-integration source suite remains historical evidence only. No suite retry launched. Read-only startup audit and an unassigned-issue draft are retained at /tmp/pair586-without-ancestors-failure.md; no cause claimed because the fixture discards stderr/process state. Report to hub for filing/rerun direction without moving main during the #597/#571 priority landings.

### Authorized unchanged rerun; independent failure filed as #598

Hub approval63d2b0cc-0e3e-4227-bd10-c459712f4280 (canonical submitted receipt matches sender/current recipient/body) authorizes filing the independent failure unassigned and one unchanged recorded suite rerun; no fixture fix inside #586, merge still held pending #597/#571 and explicit go. Filed #598 with complete failure evidence on origin. Targeted integrated diagnostics group passed8tests31.189s. Since only issue Log commits advanced the issue branch after the failure, local-check --rev initially refused the old source because it was no longer a remote branch tip (no suite executed). Published temporary verification ref verify-000586-unchanged-suite at exact a0b3fb0de67824b57d0f42cedd32acd02a7d1778 and launched the one authorized recorded rerun there. No code, timeout, assertion or fixture change; remove the temporary ref after its evidence is secured.

### Exact-source authorized rerun PASS; ready for close review

One authorized unchanged recorded rerun a434158b4ff3c9fcc7da7a10fc5eedff on a0b3fb0de67824b57d0f42cedd32acd02a7d1778 completed exit0 in798.634s (13m19), queue0.134s. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/a0b3fb0de67824b57d0f42cedd32acd02a7d1778/suite.log, SHA2562cfda2445836f4ef85c8c6744d781ddbf4ace733dbcba4ecb4b5e72067e293cf. without-ancestors five tests passed1.985s; independent first failure remains filed unassigned as #598. New8diagnostics tests passed17.813s, unchanged observation fixture passed, all fresh-restore cases and all Go packages passed (couchcore463.268s). Build make -k -B build previously passed on the same source; changes since are this issue's Log only. Original root still unproven after202timeboxed passing probes; diagnostics-only shipment remains explicitly approved. Close review is next; merge remains forbidden until the hub's go after #597/#571.

### 2026-10-16 — landing attempt, shared-tool flake and publication hold

Hub GO e63ebab4-7856-4d1f-83ea-8e6a4d548920 was verified submitted and authorized #586 ahead of #571. Ran sdlc merge --yes. It cleanly integrated main e7b676abb2083791028bed45eda22515361cca4c into reviewed branch head e53bd02e6bbc8b367580379f8a370e6b93cedeff; publish replay and CI passed. Its required integrated suite c5ff925188ea9588598fd5d74acac647 failed exit2 in831.222s (13m51), only parsed failure TestFleetInventoryLiveConformance: installed sdlc fleet inventory was killed after0.05s. #586 eight diagnostics passed18.928s and its positive fixture passed; other editor checks and remaining Go packages completed. Log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/c5ff925188ea9588598fd5d74acac647.log, SHA256624cbcaaca6326f5fe59c57ef626ddd17b967764f89a8469b468e0a4b347eadd.

One isolated diagnostic test on unchanged integrated source passed46.22s (package46.462s). Hub49ab53a3-9d50-4e4e-9d8f-3a5c71359676 reported shared installed ariadne bin/sdlc replacement at23:45, close to the immediate SIGKILL; authorized separate unassigned filing and one rerun, no #586 code changes. Filed #608 for the test's externally mutable PATH producer dependency, with correlation and full evidence; report052c9751-bd9c-432d-aef9-c56292bd2e5d is submitted. The exact kernel kill mechanism remains unproven. Approval receipt is indeterminate after submit, with canonical body matching the directly delivered instruction; not claimed as confirmed transport delivery.

Merge automatically ran its diagnostic main-base suite6e6e954f55eb8e7ba8dacebde0bf0e07 on e7b676abb2083791028bed45eda22515361cca4c, queue391.176s, execution1005.649s (16m46), exit2. It failed TestLaunchFailureStoreReopenConcurrent (1.14s), launch diagnostics lock timeout and consequent missing retention record: the existing #590 class. Log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/6e6e954f55eb8e7ba8dacebde0bf0e07.log, SHA25686910833ebc459007a65b3b46b05b4adcdfbd550fb1bd68efd64124da824a013. This diagnostic baseline was unrecorded as a commit status. Merge refused: main red (unparsed), introduced failures cannot be determined. No merge or card completion occurred. No PR-head suite retry has yet run after the shared-tool failure; the baseline is a different source and is not a passing issue verification.

Latest hub hold fbd19711-0163-49d1-818b-928f51fce86b supersedes GO for publication: no pair main publications (merge, issue new/publish or card changes) until the hub explicitly says #571 landed. Branch work is allowed. Keep #586 code unchanged; perform only the one authorized unchanged integrated-head suite rerun, retain its actual outcome, and do not invoke merge while the hold remains. Source hashes above rechecked with git rev-parse HEAD HEAD^2.

### Unchanged integrated-source rerun started under publication hold

Required recorded suite3305926cf3ce16ff7dd15d50eda49637 is queued on exact e53bd02e6bbc8b367580379f8a370e6b93cedeff, the still-pushed PR source. Tool session30415. This is the single authorized PR-source retry after #608; baseline6e6e954f55eb8e7ba8dacebde0bf0e07 was a different-source diagnostic, not a retry. No source edits; only local issue Log commits since integration. Report4c5ce451-20cb-44fe-b5a3-34da336d05e9 is queued, not yet confirmed submitted. Do not invoke sdlc merge or publish main/cards until the hub releases the #571 hold.

### 2026-10-16 — #571 landed; latest exact rerun sequence

Hub837574c4-2cdf-4672-b9c9-884b15e5d347 submitted: #571 landed and #586 released to land. At that instant run3305926cf3ce16ff7dd15d50eda49637 was still queued with no recipe execution. Planning fresh integration of #571, identified the exact owned CLI PID76794 (start00:11:25, command fixed to e53bd02e), sent SIGINT and joined tool session30415 exit130 before the suite acquired a slot. No test failed or ran in that cancelled queue attempt; no foreign process touched.

Before any integration or publication, latest submitted hub518e33f3-99ba-459c-9795-e9ed354b1446 clarified sequence: one unchanged rerun, then land if it passes; if the merge tool cannot classify because main is red on #590, report for the TL's decision instead of looping. Requeued exact e53bd02e6bbc8b367580379f8a370e6b93cedeff via sdlc local-check (session45606). This will be the first executed PR-source retry after #608. No code, budget, assertions or fixture changed. The publication hold is now released for #586 only; others remain held until it lands.

### Unchanged integrated-source rerun PASS; released to land

The single executed unchanged PR-source retry bac15c882ca7139e51f95267e961467b on e53bd02e6bbc8b367580379f8a370e6b93cedeff passed exit0 in914.583s (15m15), queue353.312s. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/e53bd02e6bbc8b367580379f8a370e6b93cedeff/suite.log, SHA256d3b82d143f443c6b87c6662ed9917a88388ceaccee3b40f35c63eabe6d31c0c4. All8diagnostics passed29.950s, the positive observation fixture and fresh-restore cases passed, and all Go packages passed (couchcore468.013s). Earlier integrated failure #608 and the independent main-baseline #590 failure remain preserved above; no code or budget change. Latest submitted hub518e33f3-99ba-459c-9795-e9ed354b1446 releases #586 after #571 and directs landing after this pass, with no looping if a later gate is blocked by #590. Push only local Log bookkeeping, then resume sdlc merge --yes; its latest-main validation still owns the final landing.

## Revisions

### 2026-10-15 — approved diagnostics-first timebox

Hub approval2db01686-340a-477c-a046-6d5dab2a3085 matches committed plan5827613c9f7ea1025454f7278e50b2f5b8fd2323 and the directly delivered instruction. Canonical receipt is indeterminate after submit (turn already running), not claimed as confirmed transport submission. Authorized diagnostics, deterministic regressions and positive control, then suite-equivalent loaded reproduction one variable at a time; record a demonstrated root in a Revision before any behavioral fix. New limit: about one hour of loaded reproduction. If no failure reproduces, report retained evidence and ship diagnostics so the next failure explains itself, then move to #589. This explicitly permits a diagnostics-only outcome without pretending the original scheduling cause is established. No larger budgets, new sleeps, retries until green, or skips.

### 2026-10-15 — timebox exhausted; ship retained diagnostics

Under hub approval2db01686-340a-477c-a046-6d5dab2a3085, select the diagnostics-only fallback after202passing cases/60.13minutes. Demonstrated fixture defect: exit-zero invalid wanted JSON was previously forwarded without status/identity validation; deterministic regressions now reject and retain it. Original historical trigger remains unproven. Preserve production Git deadlines, authorization and scheduling; do not broaden this issue into a WaitDelay fix without demonstrating the original causal chain. Deliver validated wanted observations, bounded wanted/fresh phase/order/process evidence, deterministic refusal regressions and positive control, then continue the assigned batch.

### 2026-10-15 — landing order hold

Hub receipt6a0da03a-2e89-45d7-b162-d79515f8a34a confirms diagnostics-only shipment. Complete the integrated suite and close review, but do not invoke sdlc merge until the hub explicitly releases the hold. Required landing order: #597 (operator priority), #571, #586. Continue #591 after landing; #589 is owned by pair:2.
