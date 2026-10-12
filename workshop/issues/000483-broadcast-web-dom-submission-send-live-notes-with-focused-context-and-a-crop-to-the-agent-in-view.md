---
id: 000483
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '92e716903c57ce769efb68e9648440401a124cfc' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast-web: DOM submission — send live notes with focused context and a crop to the agent in view

## Problem

Wave 3 of couch-broadcast-web (ops `teams/devinfra/projects/couch-broadcast-web/project.md`, Direction). Notes are comments in the DOM's time and space, the counterpart of 🤖[] markers in a markdown file; the operator wants them submitted to the agent in view as a "DOM submission", the integrated form of typing a message and attaching a screenshot. Everything it needs has landed: notes in the viewer (#470), the notes model with focused context and structure-snapped crops (#461, #469), agent queries (#462), replay links (#471, #477 in progress).

## Spec

Operator-shaped (the operator shapes it in the slot); the TL's starting proposal, from the project:
- **Ways in:** a button in the notes menu (`💭N`), a hotkey, and a combined submission from the draft nvim (Shift+Ctrl+Alt+Return: the draft's text plus the pending DOM notes).
- **What goes:** per note, the text, author and time; the element and its focused context (`notes.Model.Context`); a screenshot crop of the notes' cluster, snapped to structure (`NoteCrops`, captured from the operator's tab with `Page.captureScreenshot` and a clip: pixels to the agent only); a link to the moment in replay (`#t=`); and the recording's path for `couch --broadcast-web-query`. Notes far apart become one crop per cluster.
- **To whom:** the agent in view in Couch (the slot focused), delivered like a Couch message; submitted notes resolve on every screen.
- **The wire shape:** `notes.Context` gets a JSON shape here; keep it shared with pair#455 (terminal 💭N submission) so both submissions look alike to an agent.
- **Mock loop:** the agent revises, the page reloads, the cast carries on; live notes clear on reload.

## Done when

- On the blog, two viewers leave notes; the operator submits them to an agent slot (blog:0) by button, by hotkey and combined with a draft; the agent fixes at least one correctly from the submission alone, without asking for the full page.
- The submission format is documented in the atlas, shared with #455's.
- The operator smoke-tested it before close.

## Plan

- [ ]

## Log

### 2026-10-12
