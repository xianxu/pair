---
id: 000279
status: working
deps: []
github_issue:
created: 2026-09-17
updated: 2026-09-18
estimate_hours:
started: 2026-09-18T16:25:00-07:00
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

## Spec

alt+d in the switcher detaches the **selected** thread — stopping its pair
client without tearing down its zellij session — and the switcher reflects the
new state without being dismissed and reopened.

- Lifecycle chords on the panel must be **acted on**, not forwarded. A panel has
  no child to forward to, so "recognized" has to mean "handled here".
- The detach must be the same transition an in-actor alt+d performs, reaching it
  through the same named API — not a second path that happens to produce a
  similar record.
- alt+x (park) shares the panel's no-child problem and is fixed with it if the
  measurement above shows it dead too.
- The switcher's selection, not the console's focus, names the target. Those can
  differ, and detaching the wrong thread is worse than detaching none.

## Done when

- [ ] alt+d in the switcher detaches the selected thread; a test drives the
      panel-focused path and asserts the transition, so a future input-routing
      change cannot silently re-break it.
- [ ] alt+x in the switcher is verified — fixed with it, or shown to already
      work, and the answer recorded.
- [ ] The regression's introducing commit is identified, or its absence is
      recorded (i.e. the switcher path never worked post-port).
- [ ] `park.go:111`'s "in an actor or in the switcher alike" is backed by a test
      rather than by a comment.

## Plan

- [ ] Measure: is alt+x also dead in the switcher? Then trace what the console
      does with `seqDetach` while a panel holds focus.
- [ ] Bisect across `cea10ac4` / `df2a8897` for the introducing change.
- [ ] Fix at the panel input door; act on lifecycle chords instead of forwarding.
- [ ] Regression test at the panel-focused seam; atlas if the key contract moves.

## Log

### 2026-09-17

- Filed from a brain advisor session on the operator's report. No task existed:
  the nearest issues are #177 (drop a parked thread from the switcher), #205
  (batch park and detach run in parallel), #236 (switcher ordering) and #175
  (resume a parked thread from the panel) — none of them this.
- Hypothesis and regression window above are from reading `keys.go`, `park.go`
  and the recent input-routing commits; nothing was executed against a live
  switcher.
