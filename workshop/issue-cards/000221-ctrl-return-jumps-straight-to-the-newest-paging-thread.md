---
id: '000221'
status: done
started: 2026-09-10T10:28:19-07:00
created: 2026-09-09
updated: 2026-09-10
estimate_hours: 0.81
actual_hours: 0.85
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
