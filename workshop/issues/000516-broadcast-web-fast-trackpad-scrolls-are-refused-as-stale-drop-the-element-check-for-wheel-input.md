---
id: 000516
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'bbbd88a5a607dd0d1144159d42c818cfb5671d17' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T19:10:50-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# broadcast-web: fast trackpad scrolls are refused as stale; drop the element check for wheel input

## Problem

After #505, driven scrolling works when gentle, but a fast trackpad flick breaks with "The page changed. Try again" (operator, 10-12 evening). Each wheel event carries the element the viewer saw under the pointer; the injector re-hit-tests the live page and refuses on a mismatch. A flick's first events scroll the real page far before the next frame reaches the viewer, so later events (and the momentum tail) still name the old element and are refused.

## Spec

The operator decided (10-12): the element-identity check is for clicks, where acting on the wrong target can do harm; a wheel event only scrolls whatever is under the pointer, so **wheel input skips the stale-element check** (keep any bounds check that the point is inside the page). Clicks, typing and other inputs keep the check.
No coalescing up front: measure a fast flick end to end first (injector queue depth, any 503 busy answers, scroll continuing after the gesture ends). Add per-frame coalescing of wheel deltas only if that measurement shows a backlog or 503s; record the numbers in the Log either way.

## Done when

- A fast trackpad-style burst (a test sending a realistic flick: ~100 wheel events in ~1 s, with momentum) scrolls the real cast page without any refusal; the test fails on main.
- Clicks after a scroll still refuse a stale target (existing tests stay green).
- The flick measurement is in the Log, with the coalescing decision.

## Plan

- [ ]

## Log

### 2026-10-12

## Revisions

- 2026-10-12 (operator via TL): if the flick measurement shows coalescing is needed, merge by TIME, not by count or
  "whatever queued up": bucket wheel deltas by their arrival time at the viewer, one bucket per animation frame
  (~16 ms), and send the buckets in order with their time, so the momentum curve macOS produces (e.g.
  1 1 2 3 4 4 3 1 1 → 2 5 8 4 1) keeps its shape at frame resolution. Never collapse a backlog into one jump.
- 2026-10-12 (operator via TL): keep the gesture's acceleration shape, not just totals. Tag each bucket with the
  viewer's timestamp; if the measurement shows uneven arrival at Couch, add a small playout buffer (hold ~1-2 frames,
  inject each bucket at its original time offset). Measure arrival spacing at Couch in the flick test and record it
  in the Log either way.
