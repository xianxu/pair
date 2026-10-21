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
