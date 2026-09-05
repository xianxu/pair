---
id: 000187
status: working
deps: []
github_issue:
created: 2026-09-04
updated: 2026-09-04
estimate_hours:
started: 2026-09-04T21:48:27-07:00
---

# A thread with no recorded activity renders its age as 106751d ago

## Problem

Operator report from the switcher:

    parley.nvim  /Users/xianxu/workspace/parley.nvim  detached · 106751d ago

Measured, not guessed. The record holds
`last_active_at: "0001-01-01T00:00:00Z"` — the zero time — with
`created_at: 2026-09-04T15:14:59` from minutes earlier. `106751.99` days is
`math.MaxInt64` nanoseconds, so `now.Sub(zeroTime)` did not compute a large age;
it OVERFLOWED and saturated. The number is not wrong by a factor, it is not a
number at all.

Two defects, and they are worth separating because only one of them is about
timestamps.

**1. Data — `LastActiveAt` is written on exactly one path.**
`threadstore.go:425` sets it during park. Nothing sets it on detach, so a thread
created and then DETACHED has never recorded activity. That is not an exotic
path: it is what `Alt+d` does, and detached-first is what couch's startup
prefers.

**2. Display — absence is rendered as a precise value.**
`relativeMenuAge` (`menu_render.go:433`) takes `now.Sub(lastActive)` with no
guard for the zero time, and `rootStateText` concatenates the result
unconditionally for both `detached` and `parked`. The switcher therefore states
an age it does not have, to the day. This is the same shape as couch's
`ProofUnresolved` rule — absence of proof is not proof of absence — arriving in
the renderer instead of the classifier.

## Spec

Fix both, and keep them separate:

- **Detach records activity**, so a detached row has an age for the same reason
  a parked one does. `MonotonicLastActiveAt` already exists and exists for this:
  a backward or equal wall clock must not reduce a recorded activity time.
- **The renderer distinguishes "no recorded activity" from "a long time ago."**
  A zero `LastActiveAt` renders the state alone (`detached`, no age clause)
  rather than a fabricated duration. Not "0d ago", and not `created_at` silently
  substituted — a creation time is not an activity time, and quietly showing one
  as the other is how the display defect happened in the first place.

The saturation is worth pinning on its own: a test that `relativeMenuAge` is
never asked to subtract the zero time is weaker than one asserting what the row
says when it has no timestamp, because the first can pass while the row still
lies.

## Done when

- A thread detached and never parked shows `detached` with no age clause, not a
  number.
- Detaching records `LastActiveAt`, so the row gains a real age from then on.
- `relativeMenuAge` has a test for the zero time that fails if the guard is
  removed, and a test that a real duration still renders as it did.
- No other caller of `LastActiveAt` reads the zero value as a real time — in
  particular `startup.go:49`, which RANKS resumable threads by it, so a zero
  sorts last where it may currently sort in a way nobody intended.

## Plan

- [ ] Guard the renderer, with the zero-time test verified red.
- [ ] Record `LastActiveAt` on detach through `MonotonicLastActiveAt`.
- [ ] Audit the other `LastActiveAt` readers (`startup.go:49` ranking,
      `AgeBandFor` colouring) for the same absence-as-value assumption.

## Log

### 2026-09-04

Found by the operator in live use while `pair#182` was landing; filed rather than
fixed on that branch, which was already nine review rounds deep and about to
merge. Fixing unrelated code inside a close window is how a branch stops being
reviewable — the same reason the holding surface was split to `pair#186`.
