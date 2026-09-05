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

The audit named in the original third bullet was done FIRST, because it decides
the shape of the others. It removed one change (`startup.go`) and added one
(`AgeBandFor`).

**Shared predicate first, so the two guards cannot drift** (ARCH-DRY, and the
plan-quality gate's own note): `hasRecordedActivity(lastActive time.Time) bool`
in `menu_render.go`, `= !lastActive.IsZero()`. Both readers ask it rather than
each testing the zero time in its own way.

Its adversarial input class is the whole point and is the same for every bullet
below: **only the ZERO VALUE means absent.** `time.Time{}` (year 1) is "never
recorded"; `time.Unix(0, 0)` (1970) is a real, genuinely ancient timestamp that
must still render as a real age. A guard written as "very old ⇒ absent" passes
the operator's bug and fails this, which is why the tests carry both.

- [ ] **`hasRecordedActivity`** — unit test: zero ⇒ false; `time.Unix(0,0)` ⇒
      true; `now` ⇒ true.
- [ ] **`relativeMenuAge(now, lastActive) (string, bool)`** reports whether
      there is an age, and **`rootStateText`** builds the ` · <age>` clause only
      when there is one. Unit-test `rootStateText`, not just the helper: the
      helper can be correct while the row still lies, which is exactly how the
      bug shipped. Inputs: zero (no clause), `time.Unix(0,0)` (a real age,
      ~20000d), `now-2h` ("2h ago"), `now+1h` (clock skew — already clamped to
      `<1h ago`, pinned so the clamp is not lost).
- [ ] **`AgeBandFor(now, lastActive) AgeBand`** gains `AgeUnknown`, **and
      `ageColor` renders it as NO age colouring at all** rather than as another
      dim escape. Without that this guard is unobservable — `ageColor` maps
      `AgeUnknown` and `AgeOld` to the same bytes, so a test could only assert
      the enum against itself (PQ-1). "We do not know how old this is" and "this
      is ancient" must not look identical. Unit-test `ageColor(AgeBandFor(...))`
      end to end: zero ⇒ no escape; `time.Unix(0,0)` ⇒ the old-age escape;
      `now-1h` ⇒ the recent escape; the 24h and 7d boundaries pinned.
- [ ] **`Detach` records the activity time.** It lands in
      `ThreadStore.RetireIncarnation`, which takes no clock today; the park path
      is the precedent (`threadstore.go:425` folds `parkedAt` through
      `MonotonicLastActiveAt`). The time comes from `c.Clock.Now()`
      (`couch.go:29`) and is read **ONCE before `detach.go`'s revision-conflict
      retry loop**, not per iteration — a retry must not shift the recorded
      activity time, or the value depends on how much contention there was.
      One production caller (`detach.go:113`), so the signature change is
      contained. Test in `detach_test.go`: a detach sets `LastActiveAt`; a
      detach that retries on a revision conflict records the SAME time as one
      that does not; `MonotonicLastActiveAt` still refuses a backward clock.

**Not changing `startup.go:49`, and that is a finding rather than an omission.**
The ranking is `row.LastActiveAt.After(best.LastActiveAt)`, and the zero time is
`Before` everything, so a never-active row can never displace a better one — it
already sorts last within its rank class, which is what we would want. The tie
between two zero rows is decided by iteration order, and that order is
deterministic: `ProjectActionableThreads` sorts by `(RepoScope, Tag)`
(`actionableinventory.go:222`). Correct, and correct for a reason worth writing
down rather than left as an accident the next reader re-derives. Add the
comment, not a fix.

## Log

### 2026-09-04

Found by the operator in live use while `pair#182` was landing; filed rather than
fixed on that branch, which was already nine review rounds deep and about to
merge. Fixing unrelated code inside a close window is how a branch stops being
reviewable — the same reason the holding surface was split to `pair#186`.
