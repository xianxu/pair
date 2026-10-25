---
id: 000865
status: working
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours: 2.63925
card_mirror: '3f55fb7ce324c01ed39691d0cf4fab43cbef60c9' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T02:51:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# Live Couch sustains ~1.3 cores: attribute and bound it

## Problem

The original hours-old Couch used115–140% CPU while fseventsd was also busy. A short sample suggested image recording, but denser attribution did not support that owner. Historical measurements and decisions are retained in [the pre-M2 issue](../evidence/000865/pre-m2-issue.md) and [the pre-M2 design](../evidence/000865/pre-m2-design.md), pinned to preserved commit01bdfbe72710456ee1170dd54c5b86deba50f3e5.

## Spec

M1 attribution is complete. The opt-in finite profiler was extracted and landed independently as pair#871 (PR592); its source, controls and launch documentation are on main. Disposable controls did not reproduce the sustained cost, and the TL explicitly parked further scale-harness work. The operator-approved live restart produced the exact-build profile analyzed in [live evidence](../evidence/000865/live-profile.md).

The live startup/reattachment window uses1.48 cores, with terminal.Endpoint.capture accounting for46.11% of the sampled allocation difference, principally at DEC2026 synchronized-output begin. A later same-process46s observation uses about0.41 cores. The hours-old1.3-core report may reflect an uptime effect; this startup profile does not resolve it. No recorder-dominance or fseventsd causality claim follows.

TL-approved M2 reuses each endpoint's privately owned Frame.Cells and Frame.Rows backing arrays at unchanged geometry. Keep at most one current-geometry capture buffer per endpoint, no global pool. Overwrite every cell and row, explicitly zeroing absent backend cells; replace storage at a different geometry. Snapshot and Publication retain their existing independent clones. Preserve synchronized holds, timeout/release, resize, frozen history and final EOF publication under the existing endpoint mutex.

The deterministic acceptance is zero additional capture cell/row backing allocations after warmup at unchanged geometry, for N and2N captures and screen-cell counts, with at least90% reduction in capture-owned allocated bytes against the unchanged baseline. Required cell-copy work remains linear in captures times screen cells. Consumer clones/history have their existing independent ownership and separate allocation cost. Benchmark CPU is corroborating evidence, not a promise that this one correction eliminates inventory, PTY-read or aged whole-process cost.

## Done when

- M1 reconciles the independently landed profiler and records exact live CPU/allocation owners, separate startup/later windows, negative native controls and honest causality limits.
- M2 meets N/2N counted backing-allocation and byte budgets without changing immutable consumer publications or existing synchronization/history/EOF semantics.
- Regression controls cover stale erased cells/styles/links, resize growth/shrinkage, retained-frame isolation, same-feed pre-sync state, repeated begin, release/timeout and EOF; bounded baseline/candidate replay quantifies the correction.
- The final issue has mandatory boundary reviews, targeted verification and one recorded pre-close after #879's active runner ends. Later live staging or an aged profile remains operator/TL controlled.

## Plan

- [x] M1 — Reconcile landed #871 and close measured live attribution with retained historical limits.
- [ ] M2 — Reuse endpoint-owned frame backing, qualify allocation bounds and publication semantics, then close the correction.

Durable active plan: [live CPU plan](../plans/000865-live-couch-cpu-plan.md).

## Log

### 2026-10-24 — earlier work preserved
- 2026-10-24: closed M1 — Profiler delivery independently landed as871 PR592; preserved historical native controls415.469s and honest invalid scaled qualification/park decision. Operator-approved live profile metadata and exact executable agree on clean9bfd7ab51cef5f9d563b19dea087c9ef084c19c9; pprof parses all files. CPU44.80s/30.22s startup; sampled allocation delta20.86GiB with capture9.62GiB, mostly DEC2026 begin; later exact-PID18.77s/46s about0.41cores. Cumulative overlap, sampled accounting, aged-workload and fseventsd causality limits are explicit in evidence000865/live-profile.md and profiling atlas. Baseline Endpoint sync/resize/publication/EOF controls PASS0.157s; reader inventory PASS. M2 scope approved by huba1a107d5 and isolated execution by f6c298b1; planning admitted, no M2 production code or second full check. Lessons deferred until879 lands; pair0 unchanged.; review verdict: SHIP

The complete earlier issue/plan, negative native control summary and invalid scaled qualification are preserved under evidence/000865. #871 delivered the profiler independently (PR592); no profiler or scaled-harness source is copied into this fresh-main correction. The archived branch is archive/000865-profile-attribution-pre-m2 at01bdfbe72710456ee1170dd54c5b86deba50f3e5 in the original common repository. Original remote issue head424ecac0ea56c6af0eeca557c617bd5466941166 was explicitly fetched and inspected before any new leased publication.

### 2026-10-24 — actual live profile and M2 approval

Hub b5f57312 supplied operator-created ~/couch-profile-871-restage-20261024 and authorized read-only pprof against ~/workspace/pair/bin/couch, keeping staged pair:0 unchanged. Metadata and executable both identify clean9bfd7ab51cef5f9d563b19dea087c9ef084c19c9. CPU44.80s/30.22s during profiler60–90s startup window; sampled allocation delta20.86GiB, capture9.62GiB, mostly its synchronized-begin callback. Later exact-PID CPU18.77s/46s ≈0.41 cores. Startup is therefore not an aged steady-state qualification; the hours-old1.3-core condition may need a later aged profile. The evidence records owners, overlapping cumulative costs, file hashes, source trace, sampled-allocation/RSS distinctions and the unresolved fseventsd causality.

Hub a1a107d5 accepted the analysis and approved the exact private-buffer/consumer-clone/N2N proposal before code. Hub f6c298b1 authorizes isolated fresh-main planning/implementation now, with lessons.md deferred until #879 lands, targeted tests one at a time, and no second full check while #879's runner is active. #879 remains fixed in the original checkout and its queue ticket is retained.

### 2026-10-24 — isolated main-based admission

Prepared fresh main64e0a8452388843b1fd1791059368682eec1236e at /Users/xianxu/workspace/worktree/pair-slot9/pair-m2. A sibling linked worktree was rejected because numbered-environment dependencies must be independent ordinary clones. Removed that clean unused linked checkout and created an independent no-hardlinks clone at the same path, preserving the original #879 tree. Standard tracker fetch and main upstream configuration satisfied the clone's publication preconditions. sdlc workspace verifies the same numbered environment host pair:9; start-plan865 admitted planning and refreshed the unchanged owner/card mirror. Its initial push warned stale info; fetched/inspected the remote issue head as instructed. Next: commit the design, baseline targeted endpoint checks, change-code plan-quality, estimate reconciliation, M1 attribution boundary, then M2 failing allocation tests and implementation. No M2 production code written yet.

Baseline focused Endpoint sync/publication/resize/EOF/input-failure checks PASS0.157s on unchanged main-based production code. Design checkpointb9388b9b9 preserves the complete active scope and earlier artifacts; no lessons edit or full M2 check was run. Next is full-flow change-code admission for the two genuine attribution/correction boundaries; estimate reconciliation follows plan-quality clearance.

Plan-quality round1 returned PQ-1 existing-transition-fidelity and PQ-2 function-test-strategy. Source Endpoint.Resize358–386 confirms same-geometry returns early, failed acknowledgement changes no state, and a successful geometry change clears syncState before the next publication. Corrected the plan's previously wrong held-resize statement and swept its ordering claims. Replaced the semantic case inventory with named-function adversarial strategies and mechanical guards, preserving allocation counts. No M2 production code written; the original M2 buffer ownership scope is unchanged. Lessons append remains deferred until #879 lands.

Plan-quality round3 remained CLEAN and estimate-quality INFO; change-code admitted implementation atfbff1659a after the settled derivation. Advisory concerns are retained: this is a whole-issue estimate, M2 semantic/performance verification may exceed its optimistic module allowance, and active final-integration work must be accounted explicitly. M1 attribution is ready for its mandatory boundary review; M2 production code remains unchanged. Added the live-attribution link and causal limits to the already-indexed profiling atlas page.


2026-10-24 M2 RED: unchanged capture failed the new allocation tests exactly: 64 captures allocated128 objects,128 allocated256, at both80x24 and160x24. Actual synchronized Feed/Snapshot replaced fixed-geometry backing on cycle0. Erased glyph/style/link and row metadata, retained consumer/edit isolation, absent real-backend cells and held no-op/refused/successful resize controls passed on baseline. Capture benchmark100 iterations:80x24 221568B/op2allocs/op49362ns/op;160x24 434562B/op2allocs/op112715ns/op. Tests and DEC2026 replay are committed before production correction; M1 closed SHIP7688351eb1ae, no findings. No second full check and no lessons edit while879 waits.

2026-10-24 M2 GREEN: minimal12-line capture change reuses only private current-geometry storage under existing mutex. All four N/2N allocation cases now0;128 real synchronized cycles keep backing. Bounded capture replay100 iterations now0B/op0allocs/op (80x24 33135ns/op,160x24 64931ns/op). Complete synchronized ingest retains clones but drops about443KB/op,64–65% allocated bytes; timings corroborating only. Evidence capture-allocation.md records exact before/after. Nil-clearing omission overlay fails on stale B at1,0; normal candidate passes. Focused semantic suite PASS0.228s, race+history PASS2.541s; full two affected packages PASS terminal44.432s/ptychild1.045s. Reader inventory PASS without certificate changes; no new artifact reader. Lessons and full recorded check deferred until879 completion.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only for the now-settled plan; the source is marked stale and this derivation remains provisional. Plan-quality round2 is CLEAN before this reconciliation. The original2.92325h derivation remains verbatim in pre-m2-issue.md; elapsed work and actuals do not set these numbers.

Preserve the original M1 primitives rather than retroactively refitting them: profiler module midpoint design1.25 halved for standard runtime/pprof availability and thorough-spec x0.2 =0.125, impl0.6 x0.4=0.24; launch integration design2 x0.2=0.4, impl1 x0.4=0.4; native control module design0.3 x0.2=0.06, impl0.5 x0.4=0.2; its bounded discovery allowance impl0.6 x0.4=0.24. Two distinct boundary reviews retain impl0.4 x0.4=0.16 each; atlas design0.1 x0.2=0.02 and impl0.1 x0.4=0.04.

Replace only the old unresolved M2 Method B allowance (design0.45, impl0.27) with known Method A work: capture reuse plus semantic controls is one smaller existing Go module at upper design0.3 x0.2=0.06 and impl0.5 x0.4=0.2. The separately bounded allocation-count/before-after replay proof is another smaller Go test/benchmark module at midpoint design0.15 x0.2=0.03 and upper impl0.5 x0.4=0.2. Existing emulator, fake transport and testing allocation/benchmark APIs remove a novel-stack/library search requirement. Familiarity1.0. Design subtotal0.695, implementation1.84, thorough-plan design buffer15%; total0.695 x1.15 +1.84 =2.63925h. No additional API, live fleet change, aged profiling or scale-harness work is costed into M2.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.125 impl=0.24
item: api-integration design=0.4 impl=0.4
item: smaller-go-module design=0.06 impl=0.2
item: real-api-discovery design=0 impl=0.24
item: milestone-review design=0 impl=0.16
item: milestone-review design=0 impl=0.16
item: atlas-docs design=0.02 impl=0.04
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.03 impl=0.2
design-buffer: 0.15
total: 2.63925
```

Accounting boundary:2.63925h estimates the whole #865 authoring/attribution/correction item, not remaining effort. The original profiler implementation was authored under #865 before #871 extracted it; #871's separate selective extraction/landing effort is not included again here. M2's performance-proof module includes reader audits, inventory and active recorded-check handling; the review primitives include active boundary/landing integration work. Idle FIFO waiting for #879 is excluded. The bounded adversarial tests and baseline/candidate replay may exceed this allowance; keep their verification contract intact and record measured actuals rather than fitting implementation to the estimate.

## Revisions

### 2026-10-24 — measured owner and fresh-main M2

Reason: the original recorder hypothesis was unsupported, #871 independently delivered the profiler, scaled controls were parked, and an operator live profile now names the allocation owner. Delta: current Spec/Done-when/Plan replace the speculative optimization and pending-profile state with the TL-approved private-buffer correction and counted budget; complete former artifacts remain linked historical captures rather than disappearing. M1 retains its genuine attribution review boundary, and M2 its correction boundary. Aged CPU and fseventsd causality remain explicit limits, not implied fixes. Fresh-main isolation and deferred common lessons follow hub f6c298b1; no stack on #879 or unlanded profiler/harness code.

### 2026-10-24 — post-quality estimate reconciliation

After plan-quality cleared, replaced only the historical unknown M2 Method B allowance with the two settled smaller-module activities; retained original M1 primitive amounts and their provenance. Added the whole-issue versus #871 extraction accounting boundary and named active verification/integration coverage following estimate-quality INFO. This changes neither the approved implementation nor its test budgets; actuals remain measured by SDLC.
