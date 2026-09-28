---
id: '000283'
status: done
started: 2026-09-18T07:14:08-07:00
created: 2026-09-17
updated: 2026-09-18
estimate_hours: 1.06
actual_hours: 0.60
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
