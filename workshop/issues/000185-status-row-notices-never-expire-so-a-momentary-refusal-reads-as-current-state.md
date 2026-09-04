---
id: 000185
status: working
deps: []
github_issue:
created: 2026-09-04
updated: 2026-09-04
estimate_hours:
started: 2026-09-04T15:18:33-07:00
---

# Status-row notices never expire, so a momentary refusal reads as current state

## Problem

Operator report, mid-#182 smoke testing: the couch status row showed

    [pair]  brain  tools  · previous: nowhere to return to

long after the ctrl+backspace that produced it, with the question "when do we
clear the status bar's message?"

The answer was **never**. `paintNow` renders `c.feed.Latest()` — the newest entry
of a bounded rolling queue — and the feed had exactly two producers (an exit
notice and `setNotice`) and no consumer that ever retired anything. No timeout,
no clear-on-keystroke, no expiry. A refusal about a keystroke pressed a minute
ago sat there until some unrelated notice happened to displace it, reading as
current state when it was a stale event.

## Spec

The distinction was already in the type and only needed to be honoured.
`Notice.Control` separates an **obligation** from an **event**:

- An exit — "brain [couch-b1] exited (1)" — says why a pane disappeared. It is
  still true an hour later, and it stands until something displaces it.
- A refusal — "previous: nowhere to return to" — is about the keystroke just
  pressed. It is meaningless a minute later and must retire itself.

So a transient notice gets a lifetime (`NoticeLifetime`, 12s: long enough to read
a sentence you did not expect, short enough to be gone before you wonder whether
it still applies), and a control notice gets none.

Three consequences that are the design, not extras:

1. **`Feed.Row()` walks from the tail and skips what has expired**, rather than
   returning the newest unconditionally. An older transient is staler than the
   one that just expired, so it is never a better answer — only a control notice
   survives to be found underneath.
2. **Expiry is keyed by `Message.Kind`**, which IS `couchcore.Enqueue`'s collapse
   identity, so a replaced notice inherits the slot with a fresh lifetime instead
   of accumulating one. The map is pruned to the retained queue on every push, so
   a bounded queue cannot grow an unbounded shadow.
3. **The row must repaint when it expires.** Nothing else is guaranteed to happen
   at that moment — an idle console with no children producing output would keep
   painting a stale sentence forever — so the Run loop arms one timer for the
   row's own deadline, the same shape `syncSpinner` already uses, and re-arms at
   the bottom of every iteration. It dedups on the deadline's identity: without
   that, an event-heavy console would push the deadline out on every iteration
   and the notice would never actually go.

**Both the clock and the lifetime are injected**, and that pair is load-bearing
rather than over-parameterisation. Pure expiry tests hand-advance a fake clock;
a console test has to let a REAL `time.Timer` fire, so it needs the real clock
and a short lifetime instead. Injecting only the clock produced a test that
advanced fake time while the console sat on a twelve-second real timer — the
arming looked exercised and was not. That was caught by writing the test, not by
reading the code, and it is the same lesson pair#182's BR-16 taught: a guard can
be correct and unreachable.

Deliberately unchanged: an exit notice still stands indefinitely. There is no
"seen" mechanism, so the alternative is discarding the only explanation for a
vanished pane. That is the existing intent (`ExitNotice`'s own comment: capacity
pressure may discard an activity hint, never the fact that an actor ended).

## Done when

- A transient notice disappears from the status row on its own, on an idle
  console with nothing else happening.
- A control (exit) notice does not.
- A replaced notice of the same kind gets a fresh lifetime, not the old deadline.
- An expired transient uncovers a control notice underneath, but never an older
  transient.
- The expiry bookkeeping stays within the feed's capacity.
- `go test -race ./cmd/internal/couchtty/` passes: the timer and the feed are
  touched from the Run goroutine while `paintNow` reads under the mutex.

## Plan

- [x] `Feed` gains an injected clock and lifetime, an expiry map keyed by the
      collapse identity, and `Row() FeedRow` replacing `Latest() string` —
      `FeedRow` carries `Expires` (identity, for timer dedup) and `Standing`
      (delay, measured on the feed's own clock).
- [x] The Run loop arms `syncNoticeExpiry` beside `syncSpinner`; `paintNow`
      renders `Row().Body`.
- [x] Pure tests for the four policy rules, plus console tests at the seam the
      operator's own gesture travels (ctrl+backspace as BYTES, so the notice is
      pushed from the Run goroutine where the timer is armed).

## Log

### 2026-09-04

Found during pair#182 M1 smoke testing and filed separately rather than widening
that issue: it is couch-lite paper cut, not relaunch.

Verified red both ways before shipping — with `Row()` restored to "return the
newest unconditionally", `TestAnIdleConsoleRepaintsWhenItsNoticeExpires` times
out waiting for the row to lose the sentence and
`TestATransientNoticeStopsStandingAndAControlOneDoesNot` fails on the pure
policy.

One design note worth keeping: `setNotice` does not itself repaint, so a notice
reaches the screen on the next paint from some other cause. That is why the
console test forces one `repaint()` before testing the expiry — the initial
paint is not what is under test. Whether `setNotice` should repaint is a real
question and deliberately NOT answered here; it would change when every notice
in couch first appears.
