---
id: '000187'
status: done
started: 2026-09-04T21:48:27-07:00
created: 2026-09-04
updated: 2026-09-05
estimate_hours: 0.87
actual_hours: 1.64
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
