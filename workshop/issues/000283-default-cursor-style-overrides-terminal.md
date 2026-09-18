---
id: 000283
status: working
deps: [pair#255]
github_issue:
created: 2026-09-17
updated: 2026-09-18
estimate_hours:
started: 2026-09-18T07:14:08-07:00
---

# A child's default cursor style overrides the terminal's configured cursor

## Problem

A child that never sets a cursor style, or resets it with `ESC[0 q`, makes pair
send `ESC[1 q` (an explicit blinking block) on every frame. That OVERRIDES the
parent terminal's configured default: Ghostty's `cursor-style` and
`cursor-style-blink`, or any terminal's equivalent. Before #255, the child's
`ESC[0 q`, or its silence, reached the terminal as "use your default".

The default cannot survive the pipeline:

- the vendored DECSCUSR handler maps an absent parameter and `0` to `n = 1`,
  a blinking block (`third_party/vt/handlers.go:848-855`);
- the endpoint maps `Shape: int(cur.Style)+1` (`cmd/internal/terminal/endpoint.go:228`),
  so `Frame.Cursor.Shape` is never 0, although `frame.go:15` documents
  *"0/default"*;
- `cursorEpilogue` (`render.go`) therefore never emits `ESC[0 q`.

Found in #262 M2, while classifying the per-frame DECSCUSR. The flicker fix does
not depend on it.

## Spec

Carry "terminal default" through as a state distinct from an explicit block:

- the emulator records "never set, or reset to 0";
- the endpoint publishes Shape 0 for it;
- `cursorEpilogue` emits `ESC[0 q`.

RIS and a fresh endpoint start at the default. Explicit `ESC[1 q` / `ESC[2 q`
stay explicit. Needs a decision on how much of `third_party/vt` to touch, versus
tracking it at the endpoint through the `CursorStyle` callback. The callback
fires only on CHANGE, so a `0` sent while the cursor is already a blinking block
is invisible to it.

## Done when

- A child that never sets a style, and a child that sends `ESC[0 q`, both
  produce `ESC[0 q` at the parent. An explicit `ESC[1 q` still produces
  `ESC[1 q`. Tested at the endpoint → renderer seam.
- Operator smoke: with Ghostty `cursor-style = bar`, a plain shell under couch
  shows a bar.

## Plan

- [ ] Represent the default through vt → endpoint → frame → epilogue; tests at
      each seam; smoke.

## Log

### 2026-09-17

Filed from #262 M2 (see its `## Log`, "M2 input: caret blink, and a DECSCUSR
fidelity finding").
