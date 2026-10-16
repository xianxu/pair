---
id: 000597
status: open
deps: []
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
severity: critical
card_mirror: 'd09330f6892166992dfe9df3fdfb07056d80bbd6' # card fields mirrored from issue-cards; edit via sdlc
---

# Reject stale legacy native owners

## Problem

Tonight's main Couch restart parked ariadne:4, then refused its resume because legacy readiness named reused PID 1111, now an unrelated system service born before the wrapper. ariadne:5 also refuses on lsof timeouts for unrelated legacy owners under load (TL receipt 368e95f0).

## Spec

Filter legacy ownership candidates by live child, birth after the source wrapper/launch, and matching native command before admitting them as evidence. Positively stale/reused candidates are absent; unknown possible owners retain uncertainty. A writable-file timeout for an owner whose exact claimed native ID differs from the target must not block that target. Preserve same-ID collision and uncertain possible-owner refusals. No live-slot mutation; TL restages and recovers affected slots.

## Done when

- RED/GREEN fake process-table regressions cover reused legacy PID, unrelated lsof timeout, and same-ID/unknown candidate controls.
- Targeted launcher/race checks, build, required recorded suite and mandatory close review pass; land on SHIP and report hash to hub.

## Plan

- Reproduce failures with existing owner-proof fakes; implement the smallest candidate filters and same-ID expensive-probe scope.
- Verify, close, land and return to #595, whose checkpoint deletes this authority later.

## Log

### 2026-10-15

Urgent TL dispatch 368e95f0 verified submitted. #595 checkpointed at eb6528a5; this independent hotfix starts from main, with no stacked code. No live process or slot changes authorized or performed.
