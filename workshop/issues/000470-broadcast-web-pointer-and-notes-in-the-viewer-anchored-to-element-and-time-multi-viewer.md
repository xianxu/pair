---
id: 000470
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'a315479db608d1bc6b4b64d60f5d14e1493c3e8d' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast-web: pointer and notes in the viewer — anchored to element and time, multi-viewer

## Problem

Wave 2 of couch-broadcast-web (ops `teams/devinfra/projects/couch-broadcast-web/project.md`). The live viewer (pair#460) shows the cast; viewers can't yet point or leave notes on it. The terminal broadcast already has pointer colors (pair#446), the swatch rail (pair#449) and sticky notes (pair#447); the web cast needs the same, anchored to elements and moments instead of grid cells, using the notes model (pair#461).

## Spec

- **Pointer:** viewers pick a color from the swatch rail (#446/#449's six colors, same rail) and their strokes stream to everyone as `stroke` records (top-page CSS px), drawn over the scaled page.
- **Notes:** a viewer clicks an element (or drags a region) and types a note; the note is a `note` record anchored with the notes model (`notes.Model.Anchor`): node id, stable selector, text, and the box the viewer computed at click time (`couchViewer.boxOf`, approved 10-12). Everyone sees it pinned on its element; resolved notes hide; the operator sees them in Couch (#447's 💭N box). Live notes clear on a new document, per the project's Direction.
- **Region notes:** fix the seam gap pair:5 reported: `notes.Model.Crops` takes `[]NoteAt`, which can't carry a dragged region; revise the seam additively so a region note's region reaches crops.
- Reuse #447's note UI pieces where they fit; desktop Chrome and iPad Safari (touch: tap to anchor, drag to region).
- No submission to an agent yet (wave 3's DOM submission).

## Done when

- Two viewers point and leave notes on the blog cast (once the recorder #458 lands; fixtures and the fake hub until then); notes pin to the right element, survive pause/resume, clear on navigation, and are in the recording.
- Region notes reach crops.
- Tests against fixtures; the operator's smoke test (desktop + iPad), scheduled by the TL.

## Plan

- [ ]

## Log

### 2026-10-12
