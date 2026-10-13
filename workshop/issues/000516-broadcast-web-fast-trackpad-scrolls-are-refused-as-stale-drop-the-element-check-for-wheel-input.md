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
flow: {kind: quick, provenance: inferred, spec: "672f84c7", done: "e7e7fbd0"}
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

- [x] `TabInjector.inject`: a wheel skips `onTarget` and keeps the bounds check (`hit == 0` → ErrStale); downs and
  clicks unchanged. `Injector` doc and atlas updated.
- [x] Contract (`RunInjectorContract`, every injector): a wheel aimed at #save where #q is goes in; a wheel with
  nothing under its point is ErrStale. Fails with the old rule (mutation-checked): the deterministic red-on-main test.
- [x] Live: `TestDriveRealCastFlick` (WEBCAST_LIVE_CHROME): 100 wheel events in ~1.1 s from the viewer page itself
  (10 ms apart, decaying like macOS momentum) over a page of 18 px text rows, through the real cast chain; logs the
  measurement; asserts no refusal or 503 and every px injected. `startRealCast(t, path)` shared with TestDriveRealCast.
- [x] Coalescing: decided against (measurement below); no playout buffer.

## Log

### 2026-10-12
- **Flick measurement, before the fix** (headless, real cast chain): the flick reproduces the refusal only sometimes
  (1 of 2 runs: 2 refused, 1065 of 1102 px injected); with 60 px rows, 0 of 3. A CDP call per event is too slow to
  flick (100 events took 3.6 s), so the test dispatches the events inside the viewer page.
- **After the fix, 3 runs:** 100 events in 1.08-1.12 s, 1102 px sent (page px), 22-25 batches (37-42 wheel acts),
  1102 px injected every run, 0 refused, 0 busy (503). Arrival spacing at Couch 39-40 ms min, 40-43 median, 67-69
  max (the viewer's 40 ms pacing; the max is the first gap). Injection per batch 0-50 ms (median 16-17). Injection
  ended 11-40 ms after the gesture: no backlog, no scroll running on after it. Per-batch dy keeps the momentum
  shape, e.g. 200 164 135 112 92 40 47 41 37 25 30 27 18 21 19 16 15 20 9 16 14 7.
- **Coalescing decision:** none added, and no playout buffer: no backlog, no 503s, even arrival. Note for the
  operator's rule: the viewer already merges consecutive wheel acts on the same element while a batch is paced
  (`batcher`, PACE_MS 40), i.e. ~40 ms buckets of what queued, not per-frame time buckets; the measured curve keeps
  its shape at that resolution. Moving to timestamped 16 ms buckets is the operator's call (not built).

## Revisions

- 2026-10-12 (operator via TL): if the flick measurement shows coalescing is needed, merge by TIME, not by count or
  "whatever queued up": bucket wheel deltas by their arrival time at the viewer, one bucket per animation frame
  (~16 ms), and send the buckets in order with their time, so the momentum curve macOS produces (e.g.
  1 1 2 3 4 4 3 1 1 → 2 5 8 4 1) keeps its shape at frame resolution. Never collapse a backlog into one jump.
- 2026-10-12 (operator via TL): keep the gesture's acceleration shape, not just totals. Tag each bucket with the
  viewer's timestamp; if the measurement shows uneven arrival at Couch, add a small playout buffer (hold ~1-2 frames,
  inject each bucket at its original time offset). Measure arrival spacing at Couch in the flick test and record it
  in the Log either way.
