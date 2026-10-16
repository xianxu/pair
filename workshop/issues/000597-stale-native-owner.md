---
id: 000597
status: working
deps: []
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
severity: critical
card_mirror: 'f83e7afcbe689cb5e7ae098f53cca30fc2a348d2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T20:32:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Reject stale legacy native owners

## Problem

Tonight's main Couch restart parked ariadne:4, then refused its resume because legacy readiness named reused PID 1111, now an unrelated system service born before the wrapper. ariadne:5 also refuses on lsof timeouts for unrelated legacy owners under load (TL receipt 368e95f0).

## Spec

Filter legacy ownership candidates by live child, birth after the source wrapper/launch, and matching native command before admitting them as evidence. Positively stale/reused candidates are absent; unknown possible owners retain uncertainty. A writable-file timeout for an owner whose exact claimed native ID differs from the target must not block that target. Preserve same-ID collision and uncertain possible-owner refusals. No live-slot mutation; TL restages and recovers affected slots.

A park-success/resume-failure must also leave the slot startable for Resume or Fresh; stale persisted registration cannot trap the row. Reuse exact existing registration/claim rollback, preserving established identity and refusing unrelated address changes.

## Done when

- A regression for successful park then failed native start leaves Resume/Fresh startable; report one supported CLI recovery for current ariadne:4 without touching it.
- RED/GREEN fake process-table regressions cover reused legacy PID, unrelated lsof timeout, and same-ID/unknown candidate controls.
- Targeted launcher/race checks, build, required recorded suite and mandatory close review pass; land on SHIP and report hash to hub.

## Plan

- Reproduce failures with existing owner-proof fakes; implement the smallest candidate filters and same-ID expensive-probe scope.
- Verify, close, land and return to #595, whose checkpoint deletes this authority later.

## Log

### 2026-10-15

Urgent TL dispatch 368e95f0 verified submitted. #595 checkpointed at eb6528a5; this independent hotfix starts from main, with no stacked code. No live process or slot changes authorized or performed.

## Revisions

- 2026-10-15, TL 52593bb1: include stale established registration after park-ok-resume-failed and later Fresh refusal. Read-only diagnosis first; no live mutations. Keep existing claim authority and rollback, no new recovery mechanism (ARCH-DRY). Scope remains a narrow correction while #595 replaces the guard.
