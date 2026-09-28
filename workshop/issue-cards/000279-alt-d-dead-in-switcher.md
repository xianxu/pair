---
id: '000279'
status: done
started: 2026-09-18T16:25:00-07:00
created: 2026-09-17
updated: 2026-09-18
actual_hours: 0.85
---

# alt+d no longer detaches from the switcher

## Problem

Operator report, 2026-09-17: **alt+d stopped working in the switcher.** It used
to detach the selected thread; now the keystroke does nothing. "Stopped" is the
load-bearing word — this is a regression, not an unbuilt feature.

The contract it violates is stated in the code. `couchcore/park.go:111`:

> Alt+d detaches and Alt+x parks, **in an actor or in the switcher alike**.

So the switcher is explicitly in scope for both lifecycle chords, and detach is
the gesture the rest of the system is built around — `actionableinventory.go:77`
describes detached-with-no-client as "the state an `alt+d` detach leaves behind",
and `detachedsessions.go:28` calls it "exactly what `pair resume` reattaches
onto". Losing it from the switcher removes the operator's way of reaching that
state for a thread they are not currently inside.

### Standing hypothesis — recognized, but forwarded to a child that isn't there

`couchtty/keys.go:228` keeps alt+d recognized for the switcher, and describes the
handling this way:

> Lifecycle candidates remain recognized for the switcher. Console **forwards
> their exact bytes** when prefix routing leaves actor focus.

On the switcher panel there is no child. If that forward is the whole handling,
the bytes land nowhere and the chord is silently inert — which is the reported
symptom exactly. This is a reading of two comments plus the chord table
(`keys.go:230` maps `ChordAltD` → `seqDetach`), **not a traced execution path**;
confirm before building on it.

It predicts **alt+x is equally dead in the switcher**. Checking that is the
cheapest first measurement, and it discriminates: both dead points at the shared
forward-to-child path, alt+d alone points somewhere specific to detach.

### Regression window

Two candidates, both touching input routing this month, newest first:

- `cea10ac4` — **#265: couch: one panel-aware door for every input** (09-16).
  Routes key release, focus and blur through the panel check and drops them on
  the panel because "there is no child, and a key release, focus or blur has no
  panel meaning". Nearest in time and in subject matter.
- `df2a8897` — **#245: pass unreserved keys to the focused agent**.

Neither is accused; both are where to `git log -p` first. Whether the switcher
path ever worked since the Go port is itself worth confirming — the operator's
"stopped" is strong evidence it did, but the bisect is what settles it.
