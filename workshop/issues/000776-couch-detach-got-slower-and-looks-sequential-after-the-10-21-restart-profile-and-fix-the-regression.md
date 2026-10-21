---
id: 000776
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '72facce9f87b46f07b550f1375f381004b5e5d72' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-21
