---
id: 000187
status: done
deps: []
github_issue:
created: 2026-09-04
updated: 2026-09-05
estimate_hours: 0.87
started: 2026-09-04T21:48:27-07:00
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

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module          design=0.04 impl=0.14
item: smaller-go-module          design=0.04 impl=0.14
item: smaller-go-module          design=0.06 impl=0.16
item: atlas-docs                 design=0.02 impl=0.05
item: milestone-review           design=0.00 impl=0.20
design-buffer: 0.15
total: 0.87
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

**One line per instance**, so a close-time miss is attributed to a primitive
rather than to the issue. In order:

| Slug | Instances |
| --- | --- |
| `smaller-go-module` | `hasRecordedActivity` + `relativeMenuAge` + `rootStateText`'s clause |
| `smaller-go-module` | `AgeUnknown` in `AgeBandFor` + `ageColor` rendering no escape |
| `smaller-go-module` | `RetireIncarnation` takes the activity time; `Detach` reads `c.Clock.Now()` once before its retry loop |
| `atlas-docs` | the `startup.go` ranking comment, and the atlas note on absence-vs-age |
| `milestone-review` | one close boundary; this is single-pass atomic work, so no `Mx` |

Design hours carry the ×0.2 spec-quality discount: the Plan names every function,
its test file, and its adversarial input class, so the decisions are already made
and the estimator is reading rather than deciding. Buffer is +15% rather than
+30% for the same reason.

Deliberately NOT budgeted: `startup.go`'s ranking, which the audit found already
correct. Budgeting a fix I have decided not to make would be padding.

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

- [x] **`hasRecordedActivity`** — unit test: zero ⇒ false; `time.Unix(0,0)` ⇒
      true; `now` ⇒ true.
- [x] **`relativeMenuAge`** — SHIPPED IN A DIFFERENT SHAPE, see the note below. reports whether
      there is an age, and **`rootStateText`** builds the ` · <age>` clause only
      when there is one. Unit-test `rootStateText`, not just the helper: the
      helper can be correct while the row still lies, which is exactly how the
      bug shipped. Inputs: zero (no clause), `time.Unix(0,0)` (a real age,
      ~20000d), `now-2h` ("2h ago"), `now+1h` (clock skew — already clamped to
      `<1h ago`, pinned so the clamp is not lost).
- [x] **`AgeBandFor(now, lastActive) AgeBand`** gains `AgeUnknown`, **and
      `ageColor` renders it as NO age colouring at all** rather than as another
      dim escape. Without that this guard is unobservable — `ageColor` maps
      `AgeUnknown` and `AgeOld` to the same bytes, so a test could only assert
      the enum against itself (PQ-1). "We do not know how old this is" and "this
      is ancient" must not look identical. Unit-test `ageColor(AgeBandFor(...))`
      end to end: zero ⇒ no escape; `time.Unix(0,0)` ⇒ the old-age escape;
      `now-1h` ⇒ the recent escape; the 24h and 7d boundaries pinned.
- [x] **`Detach` records the activity time.** It lands in
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

**Deviation on bullet 2, stated rather than ticked away.** The plan had
`relativeMenuAge` return `(string, bool)` — the guard inside the formatter. What
shipped puts the decision one level up: `withMenuAge(state, now, lastActive)`
asks `hasRecordedActivity` and only then calls `relativeMenuAge`, which keeps its
`string` signature and stays a pure formatter that is never handed a timestamp it
cannot format. Same behaviour, and the tests are the ones the plan named
(`rootStateText` at the row). Recorded because "a formatter that also decides
whether to format" was the plan's shape and is the worse one.

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
- 2026-09-04: closed — The switcher states an age only when it has one: TestRootStateTextOmitsTheAgeItDoesNotHave asserts the ROW text for a zero timestamp, red-verified, and carries the pair that matters — time.Unix(0,0) is 1970 and must still render 19675d, so a guard written as "very old implies absent" fails it. TestAgeColourSaysNothingWhenThereIsNoAge asserts through ageColor rather than the band, because a band rendering identically to another is unobservable. Detach records LastActiveAt through MonotonicLastActiveAt, read once before the retry loop. Round-1 findings fixed: BR-1 pins the monotonic fold on the detach path (red-verified by reverting it to a bare assignment); BR-2 was my wrong conclusion — I had recorded the retry branch as untestable and the review named the missing seam instead, so threadStoreHooks.AfterGetThread now opens the read-then-CAS window and the branch is entered for real. Verified three ways: moving the clock read into the loop reddens the test, replacing the retry continue with a panic reaches the panic, and the branch old test does NOT reach it — which is why TestCouchDetachRetriesARevisionConflictAfterTeardown is renamed to AbsorbsAConcurrentWriteBeforeItsLoop, its name being why the branch went untested. SelectResumableRoot deliberately unchanged, commented not fixed. Full ./cmd/... suite green.; review verdict: FIX-THEN-SHIP

Found by the operator in live use while `pair#182` was landing; filed rather than
fixed on that branch, which was already nine review rounds deep and about to
merge. Fixing unrelated code inside a close window is how a branch stops being
reviewable — the same reason the holding surface was split to `pair#186`.

### 2026-09-04 — implementation notes

**The retry-time test now exists, and the reviewer was right that my conclusion
was wrong.** I had concluded the retry branch was untestable and recorded it as a
gap. The close review's answer was better: not "the branch is untestable" but
"add the seam, here" — `threadStoreHooks.AfterGetThread`, fired once `GetThread`
releases the lock, which is the one moment a caller's read-then-CAS window is
open. `threadStoreHooks` was already a test-only seam (`AfterJournal`,
`AfterTarget`), so this is consistent rather than new machinery.

Getting it to fail took a third attempt, and the reason is worth keeping: Detach
reads the record TWICE — once for its preconditions (`detach.go:64`) and once per
loop attempt (`detach.go:118`). A hook that bumps on the first read is absorbed,
because the loop re-reads and its CAS succeeds. Only a bump after the LOOP's read
opens a window it must retry through. Verified three ways: moving the clock read
into the loop reddens the test, replacing the retry `continue` with a panic
reaches the panic, and removing the `MonotonicLastActiveAt` fold reddens the
backward-clock test.

`TestCouchDetachRetriesARevisionConflictAfterTeardown` is renamed to
`...AbsorbsAConcurrentWriteBeforeItsLoop`, because that is what it does: it bumps
before the loop, the re-read absorbs it, and the retry branch is never reached.
Its old name is why the branch went untested for so long — the suite appeared to
cover it.

**The original attempt, kept because the wrong conclusion is the lesson.** The Plan
asked for "a detach that retries on a revision conflict records the SAME time as
one that does not", and the estimate review had already warned that the fixture's
`FixedClock` would make it unfalsifiable. I wrote it with a `sequenceClock`
instead — and it still could not fail, for a different reason: **nothing can
force a retry.** The conflict hook (`BeforePairSession`) fires before the loop,
and the loop re-reads `GetThread` on every iteration, so a bump before it is
absorbed by the first read. The store's own hooks (`AfterJournal`,
`AfterTarget`) fire inside the commit, after the revision check. There is no seam
between the loop's read and its CAS.

**That conclusion — "unreachable from tests" — was WRONG, and the paragraph above
is kept only because the wrong step is the lesson.** What was true is that no
EXISTING seam could force a retry. What I inferred is that none could exist, and
the close review corrected it by naming exactly where one belongs. The read-once
placement is pinned; see the correction entry below. What survives from this
paragraph is the observation about
`TestCouchDetachRetriesARevisionConflictAfterTeardown`, which really did never
reach the branch it was named for — it is now
`...AbsorbsAConcurrentWriteBeforeItsLoop`.

**A nil clock was a segfault, not a refusal.** Adding `c.Clock.Now()` to `Detach`
gave it a precondition it did not declare, and `TestLeaveDetachesLiveThreads...`
panicked at `detach.go:111` because its `Couch` had no clock. Detach's own guard
now names the clock alongside the store, proc ops and artifact controller — a
missing dependency should refuse where the message says which one, not crash in
the retry loop.

**Side-quest, committed separately (`e903d546`).** `main` was red before this
issue started: `TestNoCurrentSourcesAdvertiseObsoleteCouchArgv` named
`workshop/issues/000153-...md` by path, and `pair#182`'s merge archived that
issue to `workshop/history/`. Confirmed pre-existing by stashing. Widening the
guard to every current issue was tried and rejected — it flags `pair#170` and
`pair#172` for prose naming the resume OPERATION, which is not an argv
advertisement. An issue Log is working notes; the guard defends what the operator
reads.

