---
id: 000776
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '2db2f70873120537394808f4fe3f3e3cac419a07' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T12:04:14-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
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

- A measured profile of detach on the current build (per step, 4 and 8 slots) is in the Log.
- The cause of the slowdown is identified with evidence and fixed, with a regression test that bounds detach time or
  asserts parallel retirement; or the Log shows with numbers that detach is not slower and what the operator saw.

## Plan

- [x] Preserve the completed current-main four-slot profile and the failed eight-slot startup separately; preserve #759 on its own pushed branch.
- [ ] Break retirement into GetThread/backend resolution, observeRecordSession/process census, RetireIncarnation/backend routing, coordinator and store lock wait, journal recovery and publication. Measure elapsed wall time and process CPU so waiting is not mislabeled CPU work.
- [ ] Use only a bounded private shared fleet, installed Zellij, explicit empty-base environments, exact owned-process cleanup and an in-memory bounded timing buffer. Keep temporary observers in the disposable clone; no operator sessions or tracing changes.
- [ ] Compare only the measured dominant dependency's history against 856876acd. Report evidence before choosing a fix or raising fan-out. ARCH-CONSTRAINTS: operator scale is about29 slots, configured fan-out6; extrapolated waves are a hypothesis, not a29-slot benchmark.
- [ ] If a product fix is justified, send its narrow plan to the hub before implementation and change-code; defend the measured behavior with a regression through existing fake seams (ARCH-DRY, ARCH-MOCK), then normal recorded checks and review. No persisted format change is currently proposed (ARCH-EVOLVE).

## Log

### 2026-10-21


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
