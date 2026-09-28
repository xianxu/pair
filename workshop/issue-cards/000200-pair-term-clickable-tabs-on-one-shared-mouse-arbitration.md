---
id: '000200'
status: punt
created: 2026-09-06
updated: 2026-09-06
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
