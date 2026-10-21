---
id: 000760
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '82815d5b106a38ad2f0c4351da69ae34b0fbb217' # card fields mirrored from issue-cards; edit via sdlc
---

# remote-session: iPhone home-screen web app (thin nvim view, reflow, gesture navigation, three ways to send, keyboard = lock)

## Problem

remote-session (ops project, operator-approved 10-20): the iPhone client. MVP is iPhone only, a home-screen web app,
session files and side docs only.

## Spec

- **A thin view:** text and highlight ranges come from nvim through Couch's proxy; keystrokes and edits go back. A
  touch-friendly editor (CodeMirror 6 or similar) paints them.
- **Reflow for a small screen:** consecutive text lines join into paragraphs that soft-wrap; a blank line is a
  paragraph break with visible spacing; list items, headings, table rows and code fences start new blocks; list
  items wrap with a hanging indent; code blocks and tables are not reflowed (they scroll sideways).
- **Navigation (nvim keys as gestures):** tap a link = Alt+o; an "↑ parent" chip on side docs = Shift+Alt+o; swipe in
  from the left edge = Ctrl+o; from the right edge = Ctrl+i (free, since a home-screen web app has no browser back
  gesture); swipe sideways on the header bar = change hub.
- **Send:** the keyboard's Send key (`enterkeyhint="send"`; newline via a small key or long-press), a ➤ in the
  keyboard accessory bar, and a two-finger swipe up with visible feedback (text slides away, "sent" toast).
- **Lock:** showing the keyboard takes the operator lock (as parley.nvim#404 does on InsertEnter); hiding it with
  no change releases it.

## Done when

- On a real iPhone (installed to the home screen): open a hub's session.md and a side doc, read reflowed text with
  parley's colours, follow a link and come back by edge swipe, switch hub, edit and send by each of the three ways;
  the TL receives a normal "submitted:"; the desktop nvim shows the same text.

## Plan

- [ ]

## Log

### 2026-10-20
