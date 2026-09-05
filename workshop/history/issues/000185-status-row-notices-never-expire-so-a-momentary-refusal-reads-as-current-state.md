---
id: 000185
status: done
deps: []
github_issue:
created: 2026-09-04
updated: 2026-09-04
estimate_hours:
started: 2026-09-04T15:18:33-07:00
actual_hours: 0.75
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
   the bottom of every iteration. It dedups on the deadline's identity, which is
   an OPTIMISATION rather than a correctness rule — re-arming every iteration
   would still fire at the right moment, since the remaining duration shrinks
   with the deadline. Said plainly because this Spec first claimed the notice
   "would never actually go" without it, which the close review disproved
   (BR-2); a false rationale is worse than none, because it invites the next
   reader to preserve a line for a reason that was never true.
4. **Pushing a notice and painting it are one operation** (`publishNotice`).
   They were two, and that was merely LATE while notices stood forever: the
   sentence appeared whenever something else next painted. A lifetime turns it
   into a correctness bug — on an idle console nothing else paints, so a refusal
   can expire entirely UNSEEN, which is worse than one that overstayed. Found by
   the close review as BR-1 (Critical), and it is a regression this change
   introduced: the Log below had already noticed `setNotice` does not repaint and
   deliberately left the question open, without connecting that adding expiry is
   exactly what makes it load-bearing.

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
- 2026-09-04: closed — ACTUAL IS A LABELED JUDGMENT ESTIMATE, NOT MEASURED: sdlc actual reports "no measurable activity for #185" — the work was done during #182 smoke testing and the issue filed after it, so no commit window attributes to it; 0.75h is my judgment of the focused time. Behaviour: a transient notice paints itself when pushed and retires itself on an idle console while a control (exit) notice stands — TestAnIdleConsoleRepaintsWhenItsNoticeExpires drives the operator own ctrl+backspace as bytes through Run with no other event, verified red both against a restored "return the newest unconditionally" and against publishNotice not repainting. Replacement lifetime, uncover-order and capacity-bound rules each have a pure test. Round-2 BR-7: the -race Done-when was quoted from ONE run of a 30%-flaky test; reproduced at 2/10, fixed at the cause (a wait predicate the fixture already satisfied), now 12/12 on that test and 6/6 on the whole package under -race. Both Minors fixed (the test no longer discards the paint it waits for; teardown clears started). Feed.Push ok return left alone with reason: Enqueue reports false whenever capacity forced a drop, which for a rolling feed is the policy working, and narrowing that to control-only overflow is a change to the mailbox contract rather than a status-row fix. Full ./cmd/... suite green.; review verdict: SHIP

Found during pair#182 M1 smoke testing and filed separately rather than widening
that issue: it is couch-lite paper cut, not relaunch.

Verified red both ways before shipping — with `Row()` restored to "return the
newest unconditionally", `TestAnIdleConsoleRepaintsWhenItsNoticeExpires` times
out waiting for the row to lose the sentence and
`TestATransientNoticeStopsStandingAndAControlOneDoesNot` fails on the pure
policy.

**The open question above was the bug.** The first version of this Log noted
that `setNotice` does not repaint, called it "a real question deliberately NOT
answered here", and had the console test force a `repaint()` so expiry could be
tested past it. The close review named that BR-1, Critical, and was right: once
transients expire, a notice nothing repaints is a notice the operator never
sees — worse than one that overstayed. The forced `repaint()` in the test was
the tell; a test step that exists to work around the code is usually reporting a
defect rather than accommodating one. `publishNotice` now pushes and paints
together, the test asserts the sentence reaches the row with no other event, and
it is verified red.

Lesson: an open question I raise about my own change is not neutral. If I can
see the question, I can see the case where the answer matters — and here the
same change that raised it is what made the answer load-bearing.

BR-2, same round: the timer dedup was documented in four places (code, Spec,
atlas, commit message) as the thing preventing the notice from never retiring.
It is not — re-arming every iteration still fires at the right moment, because
the remaining duration shrinks with the deadline. It is an optimisation, and all
four now say so.
