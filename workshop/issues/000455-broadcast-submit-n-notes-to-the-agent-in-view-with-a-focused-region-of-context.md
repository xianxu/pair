---
id: 000455
status: open
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '94875329b9ccbb26acb0559c03a749fd5448d039' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast: submit 💭N notes to the agent in view, with a focused region of context

## Problem

Viewers of a Couch broadcast leave sticky notes on the screen (pair#447); the operator sees them in the 💭N box and today can only copy and resolve them one at a time (right-click). The notes are instructions in context, the terminal counterpart of 🤖[] markers in a markdown file, but nothing delivers them to an agent with that context. couch-broadcast-web plans the same for web pages (a "DOM submission", ops `teams/devinfra/projects/couch-broadcast-web/project.md`, Direction); the terminal version is a place to experiment with the submission format first, and to develop Couch and pair further.

## Spec

Experimental; the operator shapes it in the slot.

- **Submit:** from the 💭N box (a button or row action) and a hotkey, send the open notes to the agent in view (the slot pane focused in Couch), each with its context. Submitted notes resolve on every screen, as right-click does today.
- **Focused context, not the whole screen** (the operator, 10-11): a full screenshot carries too much; by hand the operator crops to the area that matters. Each note carries only a region around its anchor: the cells near it (a few rows and columns, the line it sits on, the enclosing block when one is detectable), as text with style marks where they matter, not pixels; plus the note's author, time and text, and a way for the agent to ask for more (e.g. the slot's tail at that time via `couch --peek`).
- **Screenshot crop (the operator, 10-11):** alongside the text, an image of the bounding box of the submitted notes plus a generous margin, rendered from the screen's cells; notes far apart become one crop per cluster. It shows what text loses (layout, color, alignment). The crop snaps to Couch's own layout, which Couch knows exactly (the operator, 10-11): a note on the tab bar → the whole tab bar, full width, its own height; a note in an agent pane → the pane's full width including its frame, with the height limited to the notes plus a margin; a note in the status line → the whole line. A per-region rule table in code, not configuration, since Couch owns the layout.
- **Region notes (to try):** let a note be a dragged rectangle as well as a point, so the author picks the region, as a screenshot crop does.
- **Combined submission (later, shared with broadcast-web):** Shift+Ctrl+Alt+Return from the draft nvim sends the draft's text together with the pending notes.
- The format is expected to change with use; keep it in one place so both terminal and web submissions can share its shape.

## Done when

- The operator submits a broadcast's open notes to the agent in view from the 💭N box and by hotkey; each note arrives with its focused region, and the agent acts on at least one correctly without asking for the full screen.
- The submission format is written down (atlas or the issue), with what was tried and why it settled where it did.
- The operator smoke-tested it in Couch before close.

## Plan

- [ ] Shape with the operator: what a focused region is, point vs rectangle notes, the message format
- [ ] Submit action in the 💭N box, and a hotkey
- [ ] Region extraction around a note's anchor
- [ ] Delivery to the agent in view; notes resolve on submit
- [ ] Operator smoke test; record the format

## Log

### 2026-10-11
