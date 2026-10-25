---
id: 000865
status: working
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours: 2.92325
card_mirror: 'c2760f428b48d58b38fcb8ca4965925094a3ca17' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T02:51:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
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

- [ ] M1 — Reconcile landed #871 and close measured live attribution with retained historical limits.
- [ ] M2 — Reuse endpoint-owned frame backing, qualify allocation bounds and publication semantics, then close the correction.

Durable active plan: [live CPU plan](../plans/000865-live-couch-cpu-plan.md).

## Log

### 2026-10-24 — earlier work preserved

The complete earlier issue/plan, negative native control summary and invalid scaled qualification are preserved under evidence/000865. #871 delivered the profiler independently (PR592); no profiler or scaled-harness source is copied into this fresh-main correction. The archived branch is archive/000865-profile-attribution-pre-m2 at01bdfbe72710456ee1170dd54c5b86deba50f3e5 in the original common repository. Original remote issue head424ecac0ea56c6af0eeca557c617bd5466941166 was explicitly fetched and inspected before any new leased publication.

### 2026-10-24 — actual live profile and M2 approval

Hub b5f57312 supplied operator-created ~/couch-profile-871-restage-20261024 and authorized read-only pprof against ~/workspace/pair/bin/couch, keeping staged pair:0 unchanged. Metadata and executable both identify clean9bfd7ab51cef5f9d563b19dea087c9ef084c19c9. CPU44.80s/30.22s during profiler60–90s startup window; sampled allocation delta20.86GiB, capture9.62GiB, mostly its synchronized-begin callback. Later exact-PID CPU18.77s/46s ≈0.41 cores. Startup is therefore not an aged steady-state qualification; the hours-old1.3-core condition may need a later aged profile. The evidence records owners, overlapping cumulative costs, file hashes, source trace, sampled-allocation/RSS distinctions and the unresolved fseventsd causality.

Hub a1a107d5 accepted the analysis and approved the exact private-buffer/consumer-clone/N2N proposal before code. Hub f6c298b1 authorizes isolated fresh-main planning/implementation now, with lessons.md deferred until #879 lands, targeted tests one at a time, and no second full check while #879's runner is active. #879 remains fixed in the original checkout and its queue ticket is retained.

### 2026-10-24 — isolated main-based admission

Prepared fresh main64e0a8452388843b1fd1791059368682eec1236e at /Users/xianxu/workspace/worktree/pair-slot9/pair-m2. A sibling linked worktree was rejected because numbered-environment dependencies must be independent ordinary clones. Removed that clean unused linked checkout and created an independent no-hardlinks clone at the same path, preserving the original #879 tree. Standard tracker fetch and main upstream configuration satisfied the clone's publication preconditions. sdlc workspace verifies the same numbered environment host pair:9; start-plan865 admitted planning and refreshed the unchanged owner/card mirror. Its initial push warned stale info; fetched/inspected the remote issue head as instructed. Next: commit the design, baseline targeted endpoint checks, change-code plan-quality, estimate reconciliation, M1 attribution boundary, then M2 failing allocation tests and implementation. No M2 production code written yet.

Baseline focused Endpoint sync/publication/resize/EOF/input-failure checks PASS0.157s on unchanged main-based production code. Design checkpointb9388b9b9 preserves the complete active scope and earlier artifacts; no lessons edit or full M2 check was run. Next is full-flow change-code admission for the two genuine attribution/correction boundaries; estimate reconciliation follows plan-quality clearance.

## Estimate

The existing card estimate2.92325h belongs to the historical full M1/provisional-M2 plan and is preserved verbatim in pre-m2-issue.md. Reconcile the now-measured M2 design after plan-quality clears; do not backfit to elapsed work or type actuals.

## Revisions

### 2026-10-24 — measured owner and fresh-main M2

Reason: the original recorder hypothesis was unsupported, #871 independently delivered the profiler, scaled controls were parked, and an operator live profile now names the allocation owner. Delta: current Spec/Done-when/Plan replace the speculative optimization and pending-profile state with the TL-approved private-buffer correction and counted budget; complete former artifacts remain linked historical captures rather than disappearing. M1 retains its genuine attribution review boundary, and M2 its correction boundary. Aged CPU and fseventsd causality remain explicit limits, not implied fixes. Fresh-main isolation and deferred common lessons follow hub f6c298b1; no stack on #879 or unlanded profiler/harness code.
