---
id: 000311
status: working
deps: []
github_issue:
created: 2026-09-23
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T13:07:35-07:00
flow: {kind: quick, provenance: inferred, spec: "73a4f0a4", done: "de0c83b9"}
---

# Click right-pane tab to switch tabs

## Problem

The right pane renders a tab strip, but clicking a tab does not switch the
right-pane terminal to that tab. Operators must use the keyboard tab controls
instead, even when the desired tab is visible.

This is the focused successor to the punted `#200`: implement the interaction
without reintroducing a second mouse-mode arbitration path.

## Spec

Clicking a visible tab chip in the right pane's tab strip selects that tab.
Clicks outside a visible chip remain pass-through or no-op according to the
existing mouse ownership rules. Hit testing must use the display-column spans
emitted by the same render pass, including clipped names and wide characters;
it must not reconstruct geometry from tab names or rune counts.

Reuse the existing right-pane tab-switch operation and mouse arbitration. Do
not change keyboard tab shortcuts, child mouse tracking, rename-field behavior,
or the handling of clicks outside the strip.

## Done when

- Clicking each visible inactive tab switches the right pane to that tab.
- Clicking the active tab is harmless and does not start rename mode or alter
  the tab order.
- Clicks on clipped-away tabs, separators, the rename field, and empty strip
  space do not select a different tab or leak an unintended event to the
  child.
- Hit testing remains correct for wide-glyph names and narrow/clipped strips.
- Tests exercise the production mouse-routing boundary and verify the existing
  child mouse-tracking behavior is preserved.
- The issue's implementation does not introduce a second mouse-mode tracker;
  the relevant reuse/ownership rule is recorded in the implementation log.

## Plan

- [ ] Trace the existing tab-strip render spans, mouse routing, and tab-switch
      operation; identify the smallest shared integration point.
- [ ] Route clicks on visible tab spans to the existing tab-switch operation,
      preserving pass-through/no-op behavior for all other coordinates.
- [ ] Add regression coverage for inactive/active tabs, clipped and wide-glyph
      spans, rename/empty space, and child mouse-mode preservation.
- [ ] Run focused and full relevant verification, then record the evidence.

## Log

### 2026-09-23

Filed from the operator request to switch right-pane tabs by clicking their
visible tab labels. Existing context: `#199` owns the strip, historical `#200`
was punted while resolving mouse arbitration, and `#258` covers keyboard/global
tab chords rather than pointer interaction.

### 2026-09-23

Resumed in slot 2 (the original claim's session is unknown; no code existed).
Trace:

- Strip: `termcmd.RenderStrip` already emits display-column `TabSpan`s from the
  clipping pass; `chromeForGeometryLocked` discards them (`.Body` only).
- Routing: `pumpStdinContext` sends every non-wheel mouse event to
  `mux.writeEvents` → `Presenter.mouseInput`. A press on the strip row
  (`Y >= childRows`) is already a `ParentPressMouse` gesture — owned by the
  parent, never forwarded to the child, release swallowed. So the hook is
  "consume a left press on a span before the presenter" — the exact shape of
  couch's `routeMouseEvent` → `RenderedStatusRow.ColumnToActor` (ARCH-DRY).
- Switch op: `selectLocked(index)` (+ `renamePane`) is what `switchRelative` uses.
- Rename: the pump drops all non-key events while a rename session is open.

**Blocker:** `pair term` builds its presenter with `terminal.ChildRequested`, so
the parent enables mouse reporting ONLY while the active child holds tracking
(`desiredParentModes`). With a plain shell, zellij never forwards clicks to the
pane at all — the strip is unclickable exactly in the common case. Delivering
the Spec requires the parent to request tracking itself, which changes what
zellij does with drag-select in a shell tab. That is the #200 territory; needs
an operator decision before planning further.
