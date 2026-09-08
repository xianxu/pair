---
id: 000217
status: open
deps: []
github_issue:
created: 2026-09-08
updated: 2026-09-08
estimate_hours:
---

# dim the right pane's tab strip when the pane loses focus

## Problem

The tab strip (`#199`) renders identically whether the right pane has focus or
not, so it competes for attention while the operator is typing in the draft
pane. The operator wants it muted when unfocused, the way the draft's statusline
reads as secondary information rather than a loud bar.

## Spec

**The strip renders muted when the pane does not have focus.**

### The mechanism is settled

`pair term` learns about focus the same way nvim does: **focus reporting**,
`DECSET 1004`. The terminal then sends `ESC [ I` on focus-in and `ESC [ O` on
focus-out, which `pumpStdin` reads alongside the chords it already parses.

Three things say this works here rather than being hopeful:

- nvim's `FocusLost`/`FocusGained` autocmds already drive real behaviour in the
  draft pane (`init.lua:434,443`), and those are exactly this mode.
- `wrapcmd` already knows the sequences by name (`wrap.go:798-799`), stripping
  them for agents that mishandle them.
- `hostty.ResetInteractiveModes` already includes `1004` in teardown.

Rendering is then pure: `StripModel` gains a `Focused bool`, and `RenderStrip`
emits the muted styling when false. That keeps every hard case a unit test, the
way the rest of the strip is.

### The hazard to design against, named up front

**A child that disables `1004` turns our focus reporting off with no signal** —
which is precisely the shape of `BR-16` in `#172` (a child writing `DECRST
?1000l` silently killed couch's clicks) and the mouse-mode family this repo has
gotten wrong four times. `ptychild.Screen` already scans DECSET/DECRST for
`1049/1047/47` and `1000/1002/1003`; `1004` joins that scan, and the strip must
**degrade to the focused (undimmed) rendering when focus state is unknown**
rather than assume unfocused. Unknown is not a synonym for either state, and a
permanently-dim strip is worse than one that never dims.

## Open question — the visual, which is NOT settled

The operator's reference is the draft's statusline. But
`pair_apply_statusline_hl` (`init.lua:2590`) links **both** `StatusLine` and
`StatusLineNC` to `Comment` — i.e. the draft's bar is *uniformly* muted and does
not currently change on focus. And the two reference screenshots are
indistinguishable to me apart from a cursor artifact.

So the treatment needs the operator to pin it before implementation, since
guessing it wastes the round trip. The proposed default, absent other direction:
**SGR 2 (faint) on the whole strip when unfocused, with the active tab keeping
its brackets** — brackets are a text signal, so "which tab is active" survives
dimming exactly as it survives a mono terminal, which is why `#199` made it text
rather than colour.

## Plan

- [ ] Pin the visual treatment with the operator (see above).
- [ ] `Screen`: scan `1004` alongside the existing DECSET tracking; expose focus
      state as **known-focused / known-unfocused / unknown**, not a bool.
- [ ] `pumpStdin`: consume `ESC [ I` / `ESC [ O` without passing them to the
      child, the way it already consumes chords.
- [ ] `StripModel.Focused`; `RenderStrip` renders muted when unfocused and
      normal when unknown. Pure tests, at least two tabs per case.
- [ ] Enable `1004` on startup and reset it on teardown.
- [ ] Manual: focus the draft, confirm the strip mutes and the active tab is
      still identifiable; focus back, confirm it returns.

## Done when

- The strip is muted when the right pane is unfocused and normal when focused.
- A child disabling `1004` leaves the strip in the FOCUSED rendering, never
  stuck dim.
- The active tab is identifiable in both states without relying on colour.

## Log

- 2026-09-08 — filed from the operator's request during `#199`'s close. Kept out
  of `#199` deliberately: that issue was mid-close and this is a new surface,
  not a defect in the strip.
