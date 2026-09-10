---
id: 000221
status: open
deps: []
github_issue:
created: 2026-09-09
updated: 2026-09-09
estimate_hours:
---

# ctrl+return jumps straight to the newest paging thread

## Problem

Reaching a thread that paged takes two gestures: `ctrl+space` opens the switcher
already focused on whoever paged, then `Return` lands on it. The panel is pure
ceremony in that path — the operator did not want to *browse*, they wanted to
answer the page.

Collapse it: **`ctrl+return` from an actor jumps directly to the thread
`ctrl+space` would have focused**, with no panel in between.

### Both halves already exist

**The selection logic is one call.** `onHotkey` (`console.go:1356-1370`) already
computes exactly the thing this feature needs:

```go
focus := c.attention.NewestActor()
if focus == (couchcore.ThreadAddress{}) {
    // The defined default with nothing paging: the thread being left.
    focus = c.menu.ActiveAddress
}
```

`attention.NewestActor()` **is** "the latest one with a notification". This issue
reuses that call rather than re-deriving the ordering — two answers to "who
paged most recently" would drift (`ARCH-DRY`).

**The chord is one table row.** `ctrl+space` is `\x1b[32;5u` — codepoint 32,
Kitty modifier bitmask 4 encoded as 4+1 (`keys.go:157`). **`ctrl+return` is
`\x1b[13;5u`** by the same construction. `keys.go` is already shaped for this:
one `seqKind`, one `hit()` case, one `knownSequences` row, one entry in
`hitHandlers()`. Its own comment records why that surface is small on purpose —
`intercepts()` was once a second switch over the same enum, and "two switches
over one enum agree until someone edits one".

### The one decision this needs

**What does `ctrl+return` do when nothing is paging?**

`ctrl+space` falls back to `menu.ActiveAddress` — the thread being left — which
is right for a *browser*: it opens somewhere sensible. For a *direct jump* that
same fallback means "jump to where you already are", i.e. a silent no-op that
looks like a dropped keypress.

Recommended: **do nothing, and say so** — a brief notice on the status row
(`nothing paging`) rather than a silent swallow or a pointless switch. The point
of the key is to answer a page; with no page there is nothing to answer.

Same question, second case: the newest paging thread **is** the current one.
Recommended: acknowledge the attention (which `switchTo` already does via
`c.attention.Acknowledge`) and stay put, so the badge clears and the key is not
inert.

Both are behaviour choices, not implementation details, and should be stated in
`## Spec` before the code exists.

### Legacy-encoding caveat, accepted not discovered

Under the Kitty keyboard protocol `ctrl+return` is `\x1b[13;5u` and separates
cleanly. **In legacy encoding it is indistinguishable from plain `Return`**
(both CR, `0x0d`), so it cannot be intercepted there without stealing every
Return from the child — which is unacceptable.

This is the same shape as the documented `ctrl+backspace` trade at
`keys.go:20-30`, and the resolution is the same: zellij pushes the Kitty
protocol (`support_kitty_keyboard_protocol true` in `zellij/config.kdl`), so the
chord works in practice. **Do not add a legacy fallback** — record the
limitation the way `previousByte`'s comment does.

## Spec

**A new intercepted chord, `ctrl+return` (`\x1b[13;5u`), that switches to
`attention.NewestActor()` without opening the panel.**

- Reuse `attention.NewestActor()`; do not re-implement recency ordering.
- Route the switch through the existing `switchTo` path so acknowledgement,
  the switch tracker, and the `previous` target all behave exactly as they do
  for a panel-driven switch. A second switch path is what this issue is trying
  not to create.
- **Nothing paging** ⇒ no switch, plus a status-row notice. Not the
  `ActiveAddress` fallback `ctrl+space` uses — that is a browser's default, not
  a jump's.
- **Newest paging thread is the current one** ⇒ acknowledge and stay.
- From the panel, `ctrl+return` is unclaimed by this issue — the panel already
  has `Return`. Leave it forwarded rather than inventing a second meaning.

Add the chord in the four places `keys.go`'s structure requires and nowhere
else; the file's comment explains why that count is the point.

## Done when

- `ctrl+return` from an actor pane switches to the newest paging thread with no
  panel frame drawn.
- The landed thread is **identical** to what `ctrl+space` then `Return` would
  have reached — asserted by a test that drives both paths against one attention
  state and compares the result, so the two cannot drift.
- Nothing paging ⇒ no switch, and the operator sees why.
- Current thread is the newest pager ⇒ attention is acknowledged, no switch.
- Acknowledgement, switch-tracker, and `previous` behave as they do for a
  panel switch — asserted, since the whole point is that this is not a second
  switch path.
- The legacy-encoding limitation is recorded in `keys.go` beside the existing
  `ctrl+backspace` note.
- `atlas/couch.md` lists the chord.

## Plan

- [ ] Decide the two behaviour cases above; record them in `## Spec`.
- [ ] Add `seqNotify` + `hit()` case + `knownSequences` row + `hitHandlers()`
      entry.
- [ ] Handler: `attention.NewestActor()` → existing `switchTo`.
- [ ] Test: both paths land on the same thread from one attention state.
- [ ] Tests for the two edge behaviours.
- [ ] `keys.go` comment + `atlas/couch.md`.

## Log

### 2026-09-09

Operator request: *"ctrl-return jumps to pair thread with latest notification,
same logic how to determine where to focus when ctrl-space is pressed."*

Scoped after reading the two seams rather than designing from the request:
`attention.NewestActor()` already is the selection rule, and `keys.go` already
has a four-site shape for adding a chord. So the work is small and the only real
content is the two behaviour decisions plus keeping the landing identical to the
two-gesture path.
