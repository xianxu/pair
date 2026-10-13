---
id: 000515
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '0a42a09688af9eac36570e28a28bb522d6840d87' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T18:31:01-07:00
---

# broadcast-web: low-rate canvas snapshots (toDataURL when not tainted) so the viewer shows canvas content

## Problem

A `<canvas>`'s pixels are never in the DOM, so the web cast's viewer shows a sized `canvas: not captured` placeholder (pair#513). Pages that draw real content on canvas lose it: the operator's blog embed (asciinema-player 3.17 draws its background tint and cursor on a canvas under the text layer), charts, diagrams, games.

## Spec

The common session-replay practice (rrweb's recordCanvas, FullStory): a low-rate image snapshot of each visible canvas, sent as a recording event and drawn in the viewer in place of the placeholder.
- Capture through CDP in the recorder's isolated world: `canvas.toDataURL('image/webp', q)` (or `captureScreenshot` clipped to the element's box when the canvas is tainted by cross-origin content, which `toDataURL` refuses).
- Rate: on change only, at most ~1 per second per canvas, and only canvases in or near the viewport; skip unchanged pixels (hash). WebGL canvases need `preserveDrawingBuffer` or the screenshot path.
- Budget: a per-cast byte cap; past it, fall back to the placeholder and say so on Couch's notice line.
- Replay and queries (#471, #462) carry the snapshots like other events; the placeholder stays as the fallback.
- Measure cost on the operator's blog and a chart-heavy page first; never steal focus.

## Done when

- The blog's asciinema embed shows its background tint and cursor in the viewer, within the rate and byte budget.
- A tainted canvas falls back cleanly; a test covers snapshot, unchanged-skip and the budget fallback.

## Plan

- [ ]

## Log

### 2026-10-12
