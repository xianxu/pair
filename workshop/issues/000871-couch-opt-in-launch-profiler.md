---
id: 000871
status: codecomplete
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours: 0.804
card_mirror: 'ad8ca1cda5cfc93215d8791e75f2ea00cb0ce864' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T08:26:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: N/A
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

- [x] Execute [the extraction plan](../plans/000871-couch-opt-in-launch-profiler-plan.md): preserve the approved finite lifecycle and launch/env ownership, exclude scaled/native harness consumers.
- [x] Qualify deterministic ownership/fault/disabled controls, counted output bounds, real pprof parsing and the audited inventory.

Workflow still required before landing: finish the retained recorded pre-close, pass close review, self-merge through the normal gates and report the invocation. These are pending workflow postconditions, not an already-completed implementation claim.

## Log

Filed unclaimed by pair:9 at TL request for pair:7. Existing native four-actor control passed415.469s at88c6b328 with exact cleanup; summary workshop/evidence/000865/native-terminal-controls.md. Subsequent scaled run exposed early worker redraw termination and N10 caller-proof failure; neither is a profiler implementation failure, and neither is qualified here.

### 2026-10-24 — planning checkpoint
- 2026-10-24: closed — Approved profiler-only extraction. Focused runtime CPU and both alloc profiles parse with real pprof; focused race/fault/child-env/N2N cohort PASS2.908s, bounds removal overlay fails both N/2N cases. Artifact classification3.259s, audited inventory and diff pass. Implementation checklist complete; recorded verification/review/landing explicitly remain workflow postconditions. Runtime64ca9573ea78dac536e567dd3897764bc58477fb pre-close lease-484894997 attempt4174e87836b2cc9812c66c20b0e0eae5 is retained, final pass required before merge; later commits docs/evidence-only. Normal actual adoption refused snapshot serialization over65536bytes, exact refusal committed; standing TLf039e10f authorizes --no-actual for ariadne593, no invented actual or restored evidence. Hub owns project. No live restart, native/scaled harness or CPU optimization claim.; review verdict: SHIP

Claimed in pair:7 and start-plan created this branch from fresh main3470debf43a86144a9ca993e9065e2d0c8dfa8ef. Existing finite diagnostic design is approved via TL dispatch and #865 provenance; no live relaunch is authorized. Read-only extraction audit found no scaled dependencies in the four core files and confirmed real pprof parsing already exists in profiling_test.go. Small missing fault cases (partial creation, allocation-after, short write, pre-cancelled parent) and counted writer growth are planned as tests, not a new mechanism. Full flow applies because imported production exceeds100lines; durable plan posted to TL before change-code.

## Revisions

### 2026-10-24 — approved full-flow extraction

TLbb7d18ab approved the finite extraction boundary and added the two-line operator handoff. The initial no-code change-code inferred quick because code is not imported yet; pin full for the approved >100production-line extraction so plan-quality and estimate gates run before implementation. Renamed the durable plan to the canonical issue stem; no design delta.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; calibration is reported stale/provisional by estimate-source. Three smaller-go-module primitives cover pure reducer extraction, effect adapter/fault tests, and launch/env integration. Existing source and approved plan resolve design: v2 design picks0.15/0.30/0.15hr x0.2; implementation picks0.30/0.50/0.30hr x0.40. Docs and one close boundary use0.10design x0.2, implementation0.10/0.35 x0.40. Go stdlib runtime/pprof and existing adapters supply the implementation; no novel library discovery. Familiarity1.0, thorough-plan design buffer15%, no cross-repo propagation. Design subtotal0.16 x1.15 + scaled implementation0.62 =0.804hr. Queue idle is not implementation effort.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.03 impl=0.12
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.03 impl=0.12
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0.02 impl=0.14
design-buffer: 0.15
total: 0.804
```

### 2026-10-24 — plan-quality cleared

PQ-1 addressed by function-level adversarial strategies, no behavioral design delta; round2 CLEAN. Derived the estimate only after that result. TL confirmed no native/scaled consumers and requested two operator lines; supplied exact COUCH_PROFILE_DIR invocation/new private destination and fixed output names, receipt59fe53b3 submitted.

### 2026-10-24 — selective extraction and verification

Change-code full flow passed plan-quality (PQ-1 addressed) and estimate-quality INFO. The adapter primitive includes controlled/race/runtime qualification; launch integration includes reader audits and merge adaptation, and the review primitive includes active pre-close/landing checks. Queue idle alone is excluded; no check/reviewer execution is excluded from actuals.

Imported exactly the four profiler/model source and test files from the named commits, unchanged (git diff against88c6b328 empty for all four). Main reconciled the selected finisher, child env, README, original atlas, index and two manifest exclusions. No native or scaled consumer, workload.go or Fleet change. Initial focused compile RED on missing profiler types; after source import, child-env regression RED1.321s demonstrated leaked opt-in. Environment clearing GREEN in original focused cohort1.867s, including real CPU and both alloc profiles parsed by go tool pprof against os.Executable.

Added profiling_faults_test.go through a bounded test-only subagent: partial reservation closure, after-snapshot failure retention, short-progress handling, pre-cancel zero effects and counted N16/2N32 output saturation. Combined race qualification of all profile/parser/child-env tests PASS2.908s; earlier original ownership race PASS1.689s. Counted sink permits35bytes/5calls/1close independent of N/2N; bypass negative control exposes all calls/bytes. A compiler overlay removing only the production truncation guard makes both N/2N cases fail on8bytes versus3 at saturation (package0.382s, exit1); actual source untouched, retained writer-bound-negative-control.log. Artifact classification PASS3.259s.

Reader audit: singleton_runtime_test.go reads only its disposable roots/child-env file, same local_or_literal_fixture ownership; updated only its named certificate. New profiler/model/fault tests read private output paths or use the current test executable and Go tool, with no repository path scan or native-owner helper; the inventory does not classify them as repository readers. No stale #865 certificate copied. Current make test-affected-inventory passed. Profiling is still opt-in, no real fleet/window/restart/stage or workload cost claim.

### 2026-10-24 — recorded pre-close queued

Runtime/test head64ca9573ea78dac536e567dd3897764bc58477fb pushed. Pre-close selected affected, then full suite because the audited singleton reader certificate changed. Retained lease-484894997, attempt4174e87836b2cc9812c66c20b0e0eae5, tool session8988, log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/4174e87836b2cc9812c66c20b0e0eae5/run.log; FIFO position3 on admission. No result claimed. No continuation restart while this run is queued/running. Mandatory close review may overlap the queue under tl-worker; final recorded pass is required before landing, and any executable correction requires a new head check. Do not chase main manually after this branch check; merge owns integration. Exact next step is check this result, close review and self-merge on SHIP.

### 2026-10-24 — known actual-snapshot limit refused close

Normal actual adoption attempted first. Close session1378 refused before review: measurement branch proof: measurement snapshot does not serialize: json: error calling MarshalJSON for type *gatestate.MeasurementInputs: measurement exceeds65536bytes. Exact refusal retained in measurement-snapshot-refusal.txt. Standing TLf039e10f explicitly retains --no-actual with evidence for this separate ariadne#593 limit; using that precise exception, no guessed hours or rewritten history. The retained pre-close lease advanced to FIFO1 and remains untouched; no test result claimed.

### 2026-10-24 — plan-check structure corrected

Close correctly refused the unchecked Plan row that bundled delivered deterministic qualification with the not-yet-completed close/landing itself. Split the completed implementation criteria from explicit pending workflow postconditions; no unfinished check, review or landing is marked done and no plan-check bypass is used. Runtime/test head and retained suite are unchanged.
