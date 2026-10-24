---
id: 000871
status: working
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '378445042606522f14fb39aecea914f19f537c2c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T08:26:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "73245055", done: "70de31b1"}
---

# Land the opt-in launch-only Couch profiler independently

## Problem

The live Couch CPU investigation (#865) needs an opt-in profile on the operator's next separately approved restart. Land its already implemented diagnostic independently so scaled synthetic controls and optimization do not hold it up. TL ops:0 explicitly requested this extraction; pair:7 owns implementation/landing from main.

## Spec

Extract the finite launch-only COUCH_PROFILE_DIR profiler selectively from #865 commits a40b597e493d104c76999ad88d1ba3fbc9600d1d and 88c6b3288e1ac5dadfdf707b36b776e9621c5c13. Disabled by default; private new leaf, 60s warmup/30s CPU, allocation-before/after and compiled build identity; no HTTP listener. One pure lifecycle transition owner, one serialized effect interpreter, exclusive profiler ownership and joined cancellation, bounded retained-first-error output (32MiB/profile,64KiB metadata), non-clobbering files. Console notices own active-terminal reporting; shutdown errors report only after terminal restoration. Clear opt-in from singleton child environment. Preserve exit behavior. No live relaunch is authorized here.

Seam: profiling_model.go/model_test.go from a40b597e; profiling.go/profiling_test.go, run.go finisher hunk, singleton.go child-env hunk, singleton_runtime_test.go control, README profiler section and atlas/couch-profiling.md plus atlas/index link from 88c6b328. Artifact manifest adds only the two profiler NonArtifactSources entries. Re-audit current-main reader certificates; do not copy stale hashes. No later production profiler fixes exist on #865 through 1eb2135667e8b608cd369529ffdff19353f8d486. Later changes only append scaled documentation and workload classification; exclude them. Existing initial native profiling_acceptance_test.go and certificate may be retained if needed for the original disposable control, but no profiling_scale_acceptance_test.go, workload.go, scaled Fleet changes or scaled testing docs. Own issue/plan/log rather than copying #865 process artifacts. Canonical accepted diagnostic design is workshop/plans/000865-live-couch-cpu-plan.md through the Enforced lifecycle core and adapter notes; exclude scaled revision. Reconcile against current main.

## Done when

- The profiler and launch integration land independently from main, with concrete operator invocation and finite/private lifecycle documented.
- Deterministic lifecycle, cancellation, write/close failure, hidden runtime writer errors, child env and disabled-path controls pass; real pprof parses all three profiles against matching executable.
- Reader inventory, affected pre-close check and mandatory binary-owned boundary review pass; PR lands via SDLC gates.
- #865 retains measurement, scaled harness repair and CPU optimization; this item claims no CPU-cost fix or full native matrix qualification.

## Plan

- [ ] Execute [the extraction plan](../plans/000871-couch-opt-in-launch-profiler-plan.md): preserve the approved finite lifecycle and launch/env ownership, exclude scaled/native harness consumers.
- [ ] Qualify deterministic ownership/fault/disabled controls, counted output bounds, real pprof parsing, inventory and pre-close; close, land and report the invocation for the operator's later restart decision.

## Log

Filed unclaimed by pair:9 at TL request for pair:7. Existing native four-actor control passed415.469s at88c6b328 with exact cleanup; summary workshop/evidence/000865/native-terminal-controls.md. Subsequent scaled run exposed early worker redraw termination and N10 caller-proof failure; neither is a profiler implementation failure, and neither is qualified here.

### 2026-10-24 — planning checkpoint

Claimed in pair:7 and start-plan created this branch from fresh main3470debf43a86144a9ca993e9065e2d0c8dfa8ef. Existing finite diagnostic design is approved via TL dispatch and #865 provenance; no live relaunch is authorized. Read-only extraction audit found no scaled dependencies in the four core files and confirmed real pprof parsing already exists in profiling_test.go. Small missing fault cases (partial creation, allocation-after, short write, pre-cancelled parent) and counted writer growth are planned as tests, not a new mechanism. Full flow applies because imported production exceeds100lines; durable plan posted to TL before change-code.

## Revisions

### 2026-10-24 — approved full-flow extraction

TLbb7d18ab approved the finite extraction boundary and added the two-line operator handoff. The initial no-code change-code inferred quick because code is not imported yet; pin full for the approved >100production-line extraction so plan-quality and estimate gates run before implementation. Renamed the durable plan to the canonical issue stem; no design delta.
