---
id: 000776
status: codecomplete
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '6b209f28ee97d1d31535aaf4a59d8fa7a7a272d9' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T12:04:14-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "eb22dab6", done: "b1a03087"}
actual_hours: 0.96
---

# couch detach got slower and looks sequential after the 10-21 restart: profile and fix the regression

## Problem

## Spec

Operator report (10-21, ops session.side/2026-10-21-06-detach-slower.md): "couch detach is a lot slower and seemingly
operating more sequentially now than before"; detach had been fast after an earlier fix, so this is a regression.

Known so far (pair:6, read-only):
- The running build is pair main 53a9d42e (installed at the 08:44 restart); the previous one was built 10-20 ~10:32
  (main around 856876acd).
- The leave/detach path's own source is unchanged between those; fan-out is still 6 on this 12-core Mac.
- The only timing on record: the 08:44 quit of the *old* build retired 29 slots over 11.7 s. No post-restart detach
  timing exists (the decision log was down until pair#774; no trace was on).
- An old-versus-new comparison on the shared fleet (pair#744) isn't possible: the 10-20 build can't register with the
  current harness (registration protocol changed).

Next: profile the current build's detach on the shared disposable fleet (4 and 8 slots), time each step (per-slot
retire, pane queries, lifecycle-lane waits), establish whether slots retire in parallel or serially; if a step
dominates or serializes, find the change in its history since 856876acd; fix the root cause.

## Done when

- A measured four-slot profile before and after the fix is in the Log, including retirement and post-Leave completion; the incomplete eight-slot attempt is retained separately under #773 (approved scope refinement below).
- The measured post-Leave delay is identified and fixed. A deterministic regression proves cancellation completes before the grace expires when the whole owned process group is confirmed absent (ESRCH), while unavailable probes retain the original two-second grace and group escalation. Native exited-leader/TERM-resistant-descendant controls prove inherited lease cleanup remains joined; the four-slot before/after timing remains required.

## Plan

- [x] Preserve the completed current-main four-slot profile and failed eight-slot startup separately; keep #759 on its own pushed branch.
- [x] Measure retirement substeps, coordinator/store waits, publication, process CPU and the post-Leave completion path, retaining attribution limits.
- [x] Use bounded private shared fleets with installed Zellij, explicit isolated environments, bounded memory observers and exact owned-process cleanup; preserve cited evidence and collect scratch.
- [x] Inspect the measured dominant dependency's history against856876acd; prove the unconditional cancellation-grace mechanism without claiming an unproved introduction commit or changing fan-out.
- [x] Implement the hub-approved completion-aware group cancellation, deterministic stateful group regressions, native descendant/lease controls, shared-caller/race checks, atlas update and private-fleet after timing.

The recorded suite, mandatory close review and sdlc landing gates follow these completed implementation tasks; their actual outcomes are recorded in the Log, not preclaimed here.

## Log

### 2026-10-21
- 2026-10-21: closed — BR-1 acceptance clause aligned with TL-approved cancellation contract; no code change. Normal/shared-caller race PASS34.190s, catalog PASS5.932s, native descendant lease controls PASS; reviewer independently confirmed baseline overlay RED and tests/race GREEN. Private4-slot service shutdown2001ms before/12.3ms after with exact cleanup. Recorded suite c9c08a90b06acc5e10090d6338d6399b remains queued on identical production/tests; required recorded pass precedes landing. ENOSPC is infrastructure-only.; review verdict: SHIP


### 2026-10-21 — profile transferred from held #759

Authenticated TL dispatches71dba091 and3bf72bb4 place this investigation on its own branch from main. #759 remains held and pushed at41bd06b3a41d1282389a0fca1060852fbc17e257; none of its implementation is included here.

Pinned current profile base: eaa758a6f0cd38837655be5334400098849410ec. Old candidate856876acd cannot register with the current #744 harness, so no valid old/new latency comparison exists. Do not adapt that protocol or call this a proved regression.

Four total slots: physical gesture3152.545ms; core Leave3014.998ms; fan-out6 with four concurrent retirements and four concurrent queries. Console queue0.010ms; thread and fan-out lane waits each at most0.004ms. Twenty-four successful list-panes queries, no retry; overlapped duration sum1521.739ms, maximum117.356ms. Exit waits10–76ms. Retirement685.882/1400.165/1345.310/1489.178ms by fixture slot0–3; summed4920.536ms over2065.331ms wall interval. Two queries inside each retirement account for117.503/82.724/203.701/129.116ms respectively. Remaining568–1360ms per slot is unattributed, including reads, owner process census and publication; storage coordinator/store locks are candidates, not measured causes. Observer append cost0.144ms and initial flush11.023ms, no overflow; final footer sync was not separately measured.

Eight-slot attempt failed before detach at worker4 registration: missing expected private agent-ready record, Pair exit1, context deadline exceeded. Fleet creation18:56:56.198242Z, request18:58:06.588319Z–18:58:29.370249Z. TL classifies this as #773's fixed10s readiness-bound defect under load; no further startup investigation or blind retry here. Cross-reference sent to #773 owner pair:7 for its Log. Fixture agents have no lifespan deadline. Both private fleet roots were cleaned with owned-generation checks and IO joins; no native run survives.

Core Detach/Leave/threadstore, storage coordinator/RegisterStore and process/quiescence implementations are unchanged between856876acd and the pinned head. No culprit is established. At operator scale29 slots, fan-out6 suggests roughly five waves;7–8s is a TL extrapolation, not a measurement. Next profile must attribute the inner retirement work before changing concurrency.

Evidence copied below; original scratch /Volumes/TestScratch/detach-compare.O0eTeh retains temporary clones, observer patch, logs and traces. These records contain only disposable fixture metadata, not operator transcripts. Private fleets are already removed. Evidence remains with this issue for review; temporary clones can be collected after the investigation.


### 2026-10-21 — bounded inner-step capture

Before the next capture, recorded exact observer seams in retirement-substeps-plan.json beside this issue evidence. Reuse pinned eaa758a6: current main a9522724 changes only issue details, with no product-code delta. Exactly one four-total-slot private capture; no eight-slot retry. Preserve nil-context and lock semantics. Whole-supervisor user/system CPU around Leave is aggregate CPU including background goroutines, never per-slot CPU. The collector remains bounded, metadata-only and in memory, with flush outside the measured core. Separate coordinator pre-lock directory syncs from actual lock wait and held work. No product fix is proposed yet.


### 2026-10-21 — inner-step capture result, cause still unproved

Exactly one authorized four-total-slot capture PASSED (54.611s including setup/cleanup). All owned generations were absent, direct child and IO joined, root /Volumes/TestScratch/pc-3450868507 removed. No eight-slot run or blind retry. Scratch observer initially failed to compile on a Darwin timeval cast; corrected before any native run. Final observer patch SHA-256 e8d0b936982139d5c8109b079176b6c0471147e55c120af408cbf76329958f1c. No product behavior changed.

This run did NOT reproduce slow retirement: physical gesture2700.922ms, core Leave601.849ms, fan-out6. Retirement121.697–226.045ms versus the earlier685.882–1489.178ms. Exact partitions in fixture slot order0–3:

| Sub-step (ms) | Slot0 | Slot1 | Slot2 | Slot3 |
|---|---:|---:|---:|---:|
| Retirement |226.045|121.697|216.983|218.209|
| GetThread including routing |24.251|6.997|30.418|30.428|
| Owner observation |183.638|89.344|147.385|164.608|
| Publication including routing |18.110|21.439|39.101|22.823|
| Two native queries inside owner |163.533|82.042|140.328|154.909|
| Owner process census |16.542|6.367|6.427|7.469|
| Journal transaction inside publication |14.389|14.400|12.289|15.780|

Uniquely matched publication coordinator intervals: combined pre-lock directory sync0.033–0.049ms, flock wait0.001ms, held12.683–16.250ms. Publication admission after routing0.216–0.272ms. Journal recovery and store-lock wait were small in these matched intervals. GetThread includes writable routing/coordinator/store recovery; its inner work was not guessed from overlapping background events.

All-window84 coordinator calls include background supervisor work: root-sync186.553ms summed, retention-sync83.737ms, flock wait123.504ms, held112.845ms. These overlap; they are NOT an additive per-slot or critical-path breakdown. Whole-supervisor CPU234.052ms user +441.387ms system across601.849ms wall includes other goroutines and is not per-slot CPU. Twenty-four queries, no reported failure; collector1330 rows, no overflow, total record cost0.782ms and first flush10.323ms.

Important remaining gap: physical gesture minus core is2099.073ms. Retained timestamps put about27ms before core and2072ms after core. Leave began19:12:19.053901Z, ended19:12:19.655754Z; trace footer mtime19:12:19.666135705Z, driver return stamp19:12:21.727759Z. The post-core interval can include footer sync/close, console teardown and process/PTY join plus helper25ms polling; it is NOT measured retirement time, and exact attribution remains unknown. Do not name a culprit or raise fan-out on this evidence. Next useful measurement is the post-Leave completion path and attribution of a genuinely slow retirement sample. Hub report sent; no further native run started.

#773 owner pair:7 authenticated the cross-reference and will append it when safely back from #774's active merge; no branch switch was requested under that runner. This is reproduction evidence, not #773 guarded acceptance.


## Revisions

### 2026-10-21 12:15 PDT — measure completion after Leave

Authenticated TL7308a242 directs a bounded post-Leave observer (console teardown, PTY join, footer sync) plus one further retirement sample with the existing timers, then a fix to the measured dominant work; no fan-out change. Capture console worker join, presenter/child disposal, message-service and runtime/lease cleanup, private recorder IO, and the harness process/PTY completion boundaries where practical. One four-slot run is the next sample; do not loop until a preferred slow result. Keep incomplete or fast samples explicit. Fix design and change-code follow measured attribution; no product change approved by inference alone.


### 2026-10-21 — post-Leave profile identifies service-close bucket

One four-slot capture PASSED31.899s with exact cleanup of /Volumes/TestScratch/pc-4266580487. Physical gesture2352.149ms, core Leave291.812ms, retirement78–111ms (slow sample again not reproduced). Console teardown1.331ms; runConsole.messages.Close2001.235ms dominates the tail. Final synced trace flush3.306ms; cmd.Wait returned5.6ms after main.before_exit, PTY reader join0.003ms. This identifies the message-service close bucket, not its internal cause. Temporary observer patch SHA-2568738aae03afb4d1bed48dc3d59beee237d545b569aa1e1d16e6aebba9604df67. No fan-out or product change.

Continue the authorized post-Leave attribution with one bounded private four-slot capture separating service cancel, server.Close, sessions.Close, broker.Close, workers.Wait, hubTargets.wait and facts.Close, plus worker-return milestones. Preserve cancellation/close order. No automatic repeat if retirement remains fast. Existing source closes sockets on lifetime cancellation and session Closed callbacks use that canceled lifetime; these observations alone do not identify the2s cause.


### 2026-10-21 — internal close fast control and configured-hub cohort

Internal close capture PASSED and exact cleanup confirmed. Gesture402.313ms, Leave325.584ms, messages.Close1.079ms; cancel0.035/server0.316/sessions0.019/broker0.308/workers0.305/facts0.090ms. No hub-target memo existed in this fixture. Producer-return milestones all arrived within about0.98ms of console return. Thus no internal attribution of the prior2001.235ms slow close is possible from this fast control. No product fix is justified yet.

Authenticated TL6e4a43d1 says identify completion waiting out its deadline; do not shorten a timer. TLbb7bac55 approves one bounded stack-on-block observer and several configured-hub attempts. Declared next cohort: at most three serial four-total-slot private attempts, stop at the first adequate slow-close stack attribution. Collect at most512KiB once if service.Close remains blocked past100ms, in memory, owned/joined watcher; flush only after operation. This threshold is diagnostic only and never changes cancellation or effect lifetime.

Harness audit corrected the initial hub hypothesis: ALL earlier runs, including the slow one, used implicit cwd and no explicit hub directories. Existing Fleet.Start has no hub/argv option. Minimal scratch-only adaptation passes its existing Workspaces.Primary as the explicit Couch directory argument, proving hubRuntime/memo creation while preserving private environment and cleanup. No live hub paths, no production test-harness change, no old-binary adaptation. Hub informed of this correction.


### 2026-10-21 12:30 PDT — measured cause and approved narrow fix

Configured-hub cohort stopped after attempt1 captured adequate evidence; attempts2/3 never started. Gesture2302.208ms, Leave231.654ms, service workers.Wait2001.050ms; hubTargets.wait0.000208ms. The blocked stack in hubstack-1-blocked-stack.json shows resourceInventory.refresh -> OSSlotCatalog.ResolveWorkspace -> WorkspaceProvisioner.identity -> OSProvisionIO.Run -> Cmd.Wait. Cmd.Wait is receiving watchCtx after Process.Wait has completed; Cancel is blocked on the unconditional2s timer at provision_io.go82. Thus leader exit is confirmed, group/descendant state is not. Exact cleanup of /Volumes/TestScratch/pc-3657925727 confirmed, all owned generations absent and direct IO joined. Stack5247bytes, captured0.364ms, private0600, no in-close IO. Evidence files beside this issue preserve the observed slow wait and its source boundary. No product-code introduction commit proved: provision_io.go and resources_inventory.go are unchanged856..eaa.

Authenticated TL04f6e7f5 approves the fix below and shared-caller sweep; TLd2e10979 requests collection of1.1GiB profiling scratch after cited logs/JSON are preserved. Eight-slot capture remains incomplete due #773 and is not acceptance. Approved qualification is four-slot before/after plus deterministic group and native descendant cases.

### Implementation plan (quick flow, no new public surface)

- [ ] In cmd/internal/couchcore/provision_io.go, extract the existing cancellation group stop into a private testable helper with signal and clock-channel dependencies. Preserve2s TERM grace, final group SIGKILL, joined exec cancellation, WaitDelay, output bounds and lease inheritance.
- [ ] Write deterministic stateful group tests: cooperative whole-group disappearance avoids waiting/escalation; a departed leader with surviving descendant does not; unavailable/permission probes retain ownership until existing grace; initial ESRCH and final ESRCH retain existing outcomes. Extract baseline semantics first and observe the new regression fail before changing behavior.
- [ ] During grace, poll the exact owned negative PGID; only errors.Is(ESRCH) confirms absence. All other probes remain unknown/present until grace expires. This is completion detection, not a shorter grace or a leader-done shortcut. No background helper outlives Cmd.Wait (ARCH-FUNERAL, ARCH-SECURE).
- [ ] Extend provision_subprocess_test.go native lease control so a leader exits on TERM while its descendant ignores TERM and retains the inherited creation lease; Run must still kill that group at the original grace and release the lease. Existing TERM-resistant leader/output/timeout cases remain.
- [ ] Sweep OSProvisionIO callers; run targeted production-cancellation and provision regressions, race checks and lint/catalog. Run a bounded private-fleet after capture, comparing matching shutdown/retirement brackets. Record fast/slow observations honestly; no fan-out change (ARCH-CONSTRAINTS, ARCH-MOCK).
- [ ] Update atlas map of provisioning cancellation, record required affected/suite check, close through the mandatory fresh review and land through sdlc. No new persisted format (ARCH-EVOLVE); reuse existing owned group signal semantics (ARCH-DRY).


### 2026-10-21 — cancellation fix and verification checkpoint

change-code admitted quick flow. Extracted the original callback unchanged into private cancelProvisionGroup, then regression RED: whole_group_gone signaled [TERM,KILL], wanted [TERM] (test0.274s). Completion-aware ESRCH polling GREEN0.320s. Additional deterministic case finishes after a descendant disappears on the second poll with no grace event delivered. Native exited-leader/TERM-resistant-descendant control keeps the inherited creation lease busy until joined group cleanup; full TestProvision(Process|Group) pass4.290s. Existing output bounds, timeout and TERM-resistant leader tests pass in that set. New helper is private and adds no stored format or public configuration.

Shared caller sweep: provisioning/setup, slot lookup/report, fleet inventory, Couch preparation, wrapper native-lock/peer runtime identity and shared testcouch provisioning all use this same OSProvisionIO boundary; normal exit/output semantics remain unchanged. No fan-out change.

Profiling evidence retention: verified all cited logs/JSON byte-for-byte and the exact blocked stack in JSON; retained the final observer diff/hash as hubstack-observer-source.json for after-fix reconstruction. Removed /Volumes/TestScratch/detach-compare.O0eTeh (about1.1GiB) after all native roots/generations had been confirmed absent and joined, per TLd2e10979. No live files removed.

Race and catalog checks initially could not start: shared /Volumes/AgentScratch/go-build is full (df186Gi used,100Mi available; exact error no space left on device). No test execution or flake claim for these setup failures. Hub informed; task-local RAM cache /Volumes/TestScratch/pair-776-checks.QKsQe4 now holds only temporary verification artifacts and will be collected after checks/qualification. No shared cache deletion or machine setting change. Race/shared-callers and catalog checks running there.


### 2026-10-21 — fixture ordering correction during race qualification

Race/shared-caller run23.078s found a defect in the NEW native test, not an unrelated flake: TestProvisionProcessExitedLeaderRetainsDescendantLease asserted that the lease must still be busy after observing the leader TERM marker, but its2.04s failure showed the original2s cleanup could already have completed when the test goroutine resumed. The lease being free at that point is legal. Moved inherited-lease proof BEFORE cancellation, retained leader TERM acknowledgment and lease-free proof AFTER Run joins; no sleep/deadline change and no production-code change. The final assertion still detects a leader-only shortcut because a surviving descendant would retain the lease after Run returns. Rerunning the meaningful race/shared-caller set on this corrected fixture, not registering or retrying it as a flake.


### 2026-10-21 — after-fix qualification

Corrected native fixture plus shared provisioning/slot/fleet/workspace caller race set PASSED34.190s. Tracked catalog PASSED5.932s. Earlier failing race was the new fixture's ordering assertion and is retained above, not a flake.

One configured-private-hub four-total-slot after capture at7141a33f0 PASSED, exact owned generations absent, direct IO joined and private fleet root removed. Production provision_io.go blob8f8a31e7155736679e475c0adf16f92a25e94e47 matched the fixed commit; observer patch did not modify it. Later5f75748fa changes only test ordering/lesson/log, no production delta. Matched observer/brackets: gesture1030.296ms, Leave882.045ms; Console teardown39.207ms; messages.Close12.333ms (workers11.834ms, hubTargets0.0005ms). Before fix configured-hub slow sample: gesture2302.208ms, Leave231.654ms, Close2001.296ms. Other core work varied (after retirement376–461ms), so do not claim a uniform fleet speedup. No stack threshold reached after fix. Direct Cancel/ESRCH call counts were not recorded; this sample alone does not establish changed-path execution. Deterministic group sequences exercise that boundary, native descendant/lease controls protect ownership, and the matched fleet capture supplies observed before/after operator-path timing.

After evidence byte-verified in 000776-detach-profile.side/after (summary, analyses, traces, command/build metadata/log, cleanup/disposal). Qualification clones and binaries removed; no native run remains. The temporary evidence-only source bundle and Go verification cache will be collected after this checkpoint/gates. PR531 open; server merge-check/conformance were pending when last inspected. Recorded affected/suite gate is held on real substrate failure: shared Go cache disk full. No bypass or global cache setting change; hub asked to restore capacity or authorize a supported per-run override.


### 2026-10-21 13:04 PDT — storage restored, resume recorded gate

Authenticated TL94985f9d restored Go's default cache configuration machine-wide. Verified go env GOCACHE now /Users/xianxu/Library/Caches/go-build (GOTMPDIR remains the slot's usable RAM scratch). Prior ENOSPC setup failures remain infrastructure-only; they executed no race/catalog test and are not test evidence or flakes. No recorded suite was started during that blocker, so there is no owned failed local-check attempt to reclassify. Resume exactly one recorded suite on the current pushed head with no override; report any same-head refusal without bypass.

All task-local targeted checks and native qualification have completed; removed /Volumes/TestScratch/pair-776-checks.QKsQe4 after joins. No shared cache removed. The duplicate after-evidence source bundle was also collected after its cited logs/JSON were committed. No test or native process was stopped by this housekeeping.


### 2026-10-21 13:06 PDT — recorded suite queued, close review overlap

One recorded suite admitted on pushed f41bac247: attempt c9c08a90b06acc5e10090d6338d6399b, run lease-3201391114; normal FIFO position5 when observed. No ENOSPC same-head refusal, no bypass or cache override. Log in the git common-dir sdlc/test-stats/inflight attempt directory. No duplicate runner. Close review may overlap this queued check per tl-worker; required passing final-code verification remains a landing prerequisite. Plan rows now distinguish completed implementation from these pending gates; this records workflow truth and changes no approved scope. No suite result is claimed yet.


### 2026-10-21 13:08 PDT — BR-1 acceptance wording aligned

Boundary review round1 found no code defects and independently reproduced the regression RED through an overlay of baseline cancellation (whole_group_gone sends TERM+KILL rather than TERM). Reviewer normal provision/group controls passed4.263s and race provision/group/Leave controls passed6.291s. Its sole Important finding BR-1, family acceptance-contract-traceability, correctly identifies the second Done-when bullet as stale: it still demanded a detach-time bound or parallel-retirement regression after the hub approved the cancellation-completion fix. The existing parallel Leave regression would pass without this fix and was not claimed as its proof.

Revision delta: replace that sole stale acceptance clause with deterministic whole-group completion before grace on ESRCH, conservative unknown-probe behavior, unchanged two-second grace/escalation, and native exited-leader/TERM-resistant-descendant lease protection. Keep measured four-slot before/after timing. This reflects authenticated TL04f6e7f5's approved contract; it adds no code or scope change. Original clause is retained here for history: "regression test that bounds detach time or asserts parallel retirement." Sweep found no other acceptance clause needing this correction. Close was NOT finalized; rerun the ordinary close so the binary review disposes BR-1, with no ledger bypass.
