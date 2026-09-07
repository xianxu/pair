---
id: 000200
status: punt
deps: ["#199"]
github_issue:
created: 2026-09-06
updated: 2026-09-06
estimate_hours:
---

# pair term: clickable tabs on one shared mouse arbitration

## Problem

Once `pair term` draws its own tab strip (`#199`), the operator should be able
to click a tab to switch to it — the same affordance `#172` built for couch's
status row, one level down.

**The constraint is the whole issue.** A clickable strip makes `pair term` a
*third* surface arbitrating mouse modes, and this exact arbitration has now been
got wrong four times in a row inside couch:

| round | face | what was wrong |
|---|---|---|
| BR-16 | the missing re-assert | a child's `?1000l` silently turned couch's clicks off for good |
| BR-22 | the WRITE | `paintNow` asserted `?1000` over a child holding `?1002`, demoting it to press/release |
| BR-26 | the OBSERVATION's shape | `Screen` folded tracking and encoding into one bool, so `?1002h` then `?1006l` read as "no mouse" |
| BR-33 | the BELIEF (`#196`) | `Mouse() == false` conflated "child asked for none" with "we have not seen the child say anything"; a reattach produces the second while couch acted on the first |

The `#172` commit that fixed BR-26 states the lesson plainly: *"Guarding one
side of a two-sided defect is why it came back."* Each round guarded a face and
left the class.

**And a second implementation already exists.** `termcmd/run.go:815`
(`appMouseMode()`, consumed at `:454` and `:460`) is the same question couch
asks as `childWantsMouse()` (`couchtty/console.go:1507`) — two independent
answers to "does the child hold mouse tracking", written at different times.
Today they only have to be right separately. A clickable strip makes `pair term`
assert modes of its own, which is precisely the position from which couch
produced all four defects above.

So the risk is concrete: adding a click handler on top of `appMouseMode()` grows
a **fifth face** of a defect class that is finally, as of `#196`, correct in
exactly one place.

## Spec

**One arbitration, two consumers.** The tri-state belief `#196` landed —
`unknown` / `observed-none` / `observed-tracking`, where *unknown means stand
back* — becomes the shared answer, and `appMouseMode()` is replaced by it rather
than extended alongside it.

The rule the tri-state encodes, which must survive the move verbatim: a
supervisor writes its own mouse mode **only** when it has observed the child
holding none. Silence is not consent — a fresh `Screen` for a still-running
child means "not yet seen", and asserting over that is what stole the
operator's drag in `#196`.

- `pair term` gains click-to-switch on the tab strip, and no second mouse-mode
  tracker.
- The forwarding rule `#172` established holds here too: forward SGR reports
  only to a child that asked for the encoding, and never re-encode to legacy.
- Geometry follows `#172`'s correction: **strip spans are display columns**, not
  rune counts (BR-32 — a rune count puts every following span one column left of
  what was drawn, and an all-ASCII test suite stays green while it does).

### To settle in the plan

1. **Where the shared arbitration lives.** `Screen` already holds the state and
   is in the shared child half (`ptychild`), which argues the *belief* is
   already shared and only the two *callers* diverge — in which case this is a
   deletion, not an abstraction. Confirm before designing anything larger.
2. **Whether the enumeration `#196` asked for gets written here.** `#196`'s
   acceptance asked for the divergence paths to be enumerated rather than
   patched one at a time (fresh launch, pane switch, reattach, child exit,
   mid-session mode change). If it closed without that, adding a second consumer
   is the moment it stops being optional — a table shared by two callers is
   cheaper than the fifth recurrence.
3. **Nested mouse ownership under layout3 + couch.** couch arbitrates for the
   host terminal; `pair term` would arbitrate inside its zellij pane. They are
   different terminals and should compose, but zellij sits between them and
   `#125` already found the right terminal behaves differently enough to need
   its own gate. Verify rather than assume.

## Done when

- Clicking a tab in `pair term`'s strip switches to it.
- `grep` finds **one** implementation of "does the child hold mouse tracking";
  `appMouseMode()` is gone, not deprecated.
- The `#196` scenario still passes for couch — a reattached child keeps its
  tracking mode — with its test unmodified by this work.
- A `pair term` child holding `?1002` (nvim, a pager) keeps its drag: selecting
  inside it is never demoted to press/release by the strip's own mode.
- Strip hit-testing is pinned with a wide-glyph case, so a rune-count regression
  cannot pass an ASCII-only suite (BR-32).
- Under layout3 inside couch, both surfaces work in one session: couch's status
  row is clickable and the right pane's strip is clickable, neither stealing the
  other's reports.
- The divergence-path table from plan item 2 exists and is shared by both
  consumers.

## Plan

- [ ] Settle plan item 1: confirm the shared belief is already in `ptychild` and
      this is a caller deletion rather than a new abstraction.
- [ ] Delete `appMouseMode()`; route `termcmd` through the tri-state.
- [ ] Verify couch is untouched — `#196`'s test passes unmodified.
- [ ] Add strip hit-testing (display columns) + click-to-switch, with a
      wide-glyph test.
- [ ] Write or extend the divergence-path table; cover both consumers.
- [ ] Verify layout3-inside-couch: both surfaces clickable in one session, and
      a `?1002` child in the right pane keeps its drag.

## Log

### 2026-09-06

Second of three connected pieces: `#198` adds the layout3 flag, `#199` builds
the strip, this makes it clickable. Split from `#199` deliberately — the strip
is a rendering job, this one carries a defect-class constraint and should not
be smuggled in as a follow-on to it.

**Why this ordering exists at all** is the couch-lite journey (recorded in full
in `#199`'s Log). The short form: layout3 was pinned off on 2026-08-22 partly
because of the right pane's mouse support; `#170` then rescoped couch to
couch-lite, and `#172` made couch's own status row interactive — which is what
turned the mouse arbitration from an assumption into four measured corrections
ending in `#196`. That knowledge is the asset this issue spends. Building the
clickable strip in August would have re-derived all four the hard way; building
it now is a deletion plus a renderer, **provided** the arbitration is reused
rather than re-implemented.

The `#196` round also produced the reason to distrust manual verification here
(BR-34): both smoke runs attached sessions whose nvim and zellij announce
`?1006`, so the mainstream configuration passed while the reported one was never
exercised. Any manual check on this issue must state which mode the child
actually held, or it is evidence of nothing.

## Log

### 2026-09-06 — a reproducible break, and the recovery that did NOT work

Found accidentally while probing `#199`, and worth more than the four
historical rounds because it reproduces on demand.

**The break.** A throwaway shell script ran in the right pane with mouse
reporting on (`?1006h` + `?1000h`) and, on exit, restored what it assumed was
the prior state by writing `?1000l` + `?1006l`. Those bytes travel up through
zellij into couch's child stream, where `ptychild.Screen` scans DECSETs. couch
observed a well-formed "the child turned mouse OFF" and — correctly, by the
tri-state rule `#196` landed — moved from *unknown* to *observed-none*, which is
exactly the state in which a supervisor may assert its own mouse mode. It did,
and **copy-on-select highlighting in the AGENT pane stopped**.

Nothing malfunctioned. The tri-state answered the question it was asked. The
question is the problem:

> **`Screen` reports what the child's byte stream said, which is not the same as
> what the child INTENDS.** A transient borrower of the terminal — a script, a
> pager, anything that tidies up after itself — emits mode changes
> indistinguishable from the long-lived child's own.

That is a fifth face of the same class, and unlike the previous four it is not
a coding error: BR-16/22/26/33 were each a wrong answer to "what does the child
hold". This one is a right answer to the wrong question.

**The recovery finding, which is the actionable half.** Switching couch actors
(`alt+n`, then back) did **NOT** restore the highlight — the operator tried it
first. Only exiting couch and starting it again did.

That matters because an actor switch is precisely the re-evaluation path
`#172`'s BR-33 fix added. So either the re-assert does not run on that path, or
it runs and re-derives the same stale belief because nothing re-observes the
child's intent — the child is not going to re-announce `?1000h` merely because
couch switched panes. A full restart works because it mints a fresh `Screen`,
returning the belief to *unknown*, where the rule says **stand back**.

So the practical shape of the defect today: **couch can enter a wrong belief
that survives every in-session recovery gesture.** A supervisor whose only exit
from a bad state is a restart is the thing to fix, and it is a stronger
motivation for this issue than "clicking a tab would be nice".

**For the plan, when this is picked up:**

1. Reproduce with the script above rather than by hand-waving about children.
2. The enumeration `#196` asked for must include this row: *a transient
   in-pane process changes modes and exits.* It is not in the fresh-launch /
   pane-switch / reattach / exit / mid-session list.
3. Decide whether an observation should DECAY — a mode change seen once and
   never reaffirmed is weaker evidence than a mode the child re-asserts on
   every repaint. "Stand back on unknown" is the right rule; the gap is that
   nothing ever returns the belief TO unknown short of a restart.
4. Whatever is built, `alt+n` must recover it. That is the cheap acceptance
   test and today it fails.
