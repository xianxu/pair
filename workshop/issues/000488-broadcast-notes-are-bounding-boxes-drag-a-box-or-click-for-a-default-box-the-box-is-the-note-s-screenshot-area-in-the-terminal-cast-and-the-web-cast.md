---
id: 000488
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'd6504b76741ec18828bf5ac29914b2faa1028978' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T10:40:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# broadcast: notes are bounding boxes — drag a box (or click for a default box); the box is the note's screenshot area, in the terminal cast and the web cast

## Problem

A note today is a point (terminal broadcast, pair#447) or, in the web cast, a point or a dragged region (pair#470). What the agent should see with a note (pair#455 terminal submission, pair#483 web submission) has been worked out indirectly: nearby cells, structure-snapped crops, clustering. The operator (10-12): the natural way to comment is to draw a bounding box. The box says exactly what the note is about, and it is the area to screenshot.

## Spec

One mechanism for both casts (the terminal viewer and the web viewer already share `collab.js`):
1. **Note mode:** click places a note as today; dragging draws a rectangle and places the note on that box. Touch: tap = point, drag = box.
2. **Every note has a box.** A point note gets a default box of 1/10 × 1/10 of the screen (the cast's viewport), centered on the point and clamped to the screen.
3. **The box is the screenshot area** for submission (#455, #483): the crop is the box, plus a small margin; no structure snapping, no clustering (overlapping boxes may still merge into one image).
4. **Format:** the box travels with the note in the operator's units: terminal = cells (rows, columns); web = top-page CSS px (the `note` record's `region`, now always set). The web anchor (element, selector, text) stays: the note anchors to the smallest element containing the box's center, for the agent's context.
4b. **Terminal anchor (the operator, 10-12):** the terminal counterpart of the web's element anchor, so the agent gets "what this is", not just pixels. Couch knows its own layout exactly, so each box resolves to a component path plus the text under it: e.g. `tab bar › tabs: pair:1, pair:2 ●, Chrome ⏸` ; `pane pair:3 › agent (Claude Code) › near the prompt` ; `pane ops:0 › draft nvim › tl-status-xian-xu.md, lines 12–18` ; `status line`. For an editor pane, name the file and the line range under the box; for an agent pane, the slot, the agent and the nearest prompt or message; for a review pane, the document and its section. Shared shape with the web anchor (kind, path, text) so #455/#483 submissions read alike.
5. **Display:** viewers see each note's box outlined (in the author's color) with the note pinned at its corner; resolved notes hide.
6. The notes model's structure-snapped crops (#461) become unused for submission; keep landmark context for the agent, retire crop snapping and the per-URL crop hints, or keep them only behind the default box: decide in the plan with the operator.

## Done when

- In both casts, a viewer drags a box or clicks a point; every note carries a box (default 1/10 × 1/10 for points); viewers see the boxes.
- The box is what #455/#483 crop (or, if they haven't landed, the box is exposed through the notes seam they will use).
- Tests in both viewers (mouse and touch); the operator smoke-tested it in both casts.

## Plan

Durable plan: `workshop/plans/000488-…-plan.md` (same slug).

- [ ] M1 — shared `noteBox` (collab.js) + web cast: always-region with default box, BR-4 clamp, outlined pins, NoteAt kind/path, NoteCrops = padded box (snap + Hints retired, hub-coordinated with #483)
- [ ] M2 — terminal boxes: wire `box` + derived pin, viewer drag, DrawNotes outline, `NoteCrop` seam for #455
- [ ] M3 — terminal anchor: pure `AnchorFor` + list-panes rect cache (fake) + nvim view.json
- [ ] Operator smoke test in both casts

## Log

### 2026-10-12
- Claimed from pair:3 (TL dispatch). Plan drafted; sent to hub for UI choices A–D before building.
