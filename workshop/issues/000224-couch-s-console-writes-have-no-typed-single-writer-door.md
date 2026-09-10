---
id: 000224
status: open
deps: []
github_issue:
created: 2026-09-09
updated: 2026-09-09
estimate_hours:
---

# couch's console writes have no typed single-writer door

## Problem

Found while closing `#209`, and recorded there as I-2 rather than fixed there,
because the change is couch-wide.

`Console.takeOverScreen` carried the sentence *"It is still Run-goroutine-only,
like every other writer."* It is false. `switchTo` — which calls it — is reached
from the **operationQueue** goroutine as well as `Run`:
`ExecuteConsoleOperation`'s `switch` case, wired through `couchcore`'s
`EffectConsole` dispatch. That is the switcher's Enter and the status-chip
click, i.e. the operator's primary way of changing threads, not an edge.

`#209` C2 found this while fixing the repaint nudge's ordering and fixed the
half it owned: `ptychild.Child` now owns its geometry, so no caller has an
ordering obligation about resizes. The WRITER's half is untouched. `c.host` is a
bare `io.Writer` with no serialization, and couch writes to it from several
sites (`console.go` around the takeover, the row paint, the diagnostics; plus
`console_menu.go`'s hide-cursor and cursor-position writes). Two goroutines
writing escape sequences to one terminal interleave at byte granularity, which
is the corruption class `#199` M2 spent a milestone on for `pair term`.

**The answer already exists next door.** `termcmd.paneWriter` is deliberately
NOT an `io.Writer`: `io.WriteString(m.pane, …)` and `m.pane.Write(…)` are
compile errors, so a new door cannot skip the gate's reasoning. `#199` M2 landed
that after a scanning test proved narrower than its claim — the type replaced
the test. couch runs the same primitives (`hostty.Reservation`,
`ptychild.Screen.SafeToPaint`) through an untyped writer.

This is the divergence `BR-77` and `BR-82` are both instances of: one shared
primitive, two consumers, and the reasoning lands at one of them.

## Spec

**couch gets the same typed single-writer door, and the claim about goroutines
becomes a mechanism rather than a sentence.**

Two things to settle in the plan:

1. **Whether the door is a type or a queue.** termcmd's answer is a non-`Writer`
   type plus a single writer goroutine draining a channel. couch has a `Run`
   loop that could serve the same role, but `ExecuteConsoleOperation` currently
   runs console effects inline on the operationQueue goroutine — so either the
   console effects post to `Run`, or the writer becomes independently
   serialized. These differ in what happens to an operation that must observe
   the paint it caused.
2. **What the enumeration is.** Every `c.host.Write` and every
   `hostty.*` string reaching the terminal, with a stated exemption per site
   that legitimately bypasses the gate — the shape `termcmd`'s `paneWriter.raw`
   reason strings already have.

Out of scope: the paint GATE itself (`SafeToPaint`), which couch already
consults. This is about who may write, not about when.

## Done when

- couch's terminal writes go through one typed door, and a write that skips it
  does not compile — the `termcmd` standard, not a test that reads source.
- The goroutine claim in `takeOverScreen`'s doc is true, or gone.
- `#209`'s note pointing here is resolved.

## Plan

- [ ] Settle the two questions above.
- [ ] Enumerate every current write site and its intent.
- [ ] Land the door; make the divergence a shared primitive rather than a copy.

## Log

### 2026-09-10

Filed from `#209`'s third boundary review (I-2). The finding's own framing is
the reason this is a separate issue: *"a goroutine-ownership claim in a comment
is not a mechanism"* — `#209` proved that for the nudge and left the writer
claiming the same thing on the same function.
