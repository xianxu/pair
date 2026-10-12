---
id: 000494
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '929a19c67982ffbb41b3fddf946f614ad115a02a' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast-web: the operator annotates in the cast Chrome itself (pointer strokes and notes, injected overlay)

## Problem

In the web cast, viewers point and leave notes (pair#470), but the operator, who is using the page in the cast Chrome, can't: they'd need a separate viewer window. The operator (10-12): the Chrome window Couch spins up should have the pointer and annotation facility itself, so the operator can use that screen directly and mark what they're pointing at for everyone.

## Spec

- **Injected overlay:** reuse the viewer's annotation pieces (`collab.js`, the swatch rail, pen strokes, notes, and #488's boxes once landed) in the cast tab, injected through CDP into the same isolated world as the Go-live button (#454), so the page's scripts can't see or affect it.
- **Never recorded as page content:** the overlay's nodes must not enter the recording as DOM (the recorder filters Couch's own injected nodes, as for the Go-live button); its strokes and notes enter as `stroke`/`note` records through the Board, authored as the operator.
- **Modes:** off by default; a toggle (in the overlay and a Couch key or the Chrome cell's menu) turns on pen or note mode, during which pointer input goes to the overlay, not the page; off returns input to the page. The operator's plain pointer is already streamed (`ptr`, `hover`).
- **While a viewer drives** (#467), the overlay stays available to the operator and never steals the driver's input.
- Depends on pair#493 (notes and strokes reach the recording) for replay and queries.

## Done when

- The operator draws and leaves notes in the cast Chrome; viewers see them live, attributed to the operator; the page's DOM in the recording has no overlay nodes; replay and queries show the operator's notes (after #493).
- Tests; the operator smoke-tested it.

## Plan

- [ ]

## Log

### 2026-10-12
