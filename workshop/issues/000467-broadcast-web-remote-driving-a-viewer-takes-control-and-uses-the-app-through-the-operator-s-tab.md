---
id: 000467
status: open
deps: [000466, 000458, 000460]
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'f53dfea3ac79938a0d0cde9418213667306ea6e6' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast-web: remote driving — a viewer takes control and uses the app through the operator's tab

## Problem

Build remote driving for internal viewers (the operator, 10-12; MVP), against the remote-driving contract filed alongside this issue.

## Spec

- Viewer: request control; while driving, capture pointer and keyboard on the scaled, script-free frame, map them back to element ids and page coordinates, and send them over the reverse channel; give control back.
- Couch: grant/revoke from the `Chrome ⏸` cell (and its menu); show who is driving in the tab bar and in the page; inject input through the browser-control seam into the cast tab only.
- Recorder: actions recorded with their source; control changes recorded.
- Works from a desktop browser and an iPad (touch maps to pointer).

## Done when

- A second person (desktop Chrome) requests control, the operator grants it, they click, type and scroll in the blog through the cast, then give control back; the operator can revoke at any time; nothing outside the page is reachable.
- The recording attributes their actions to them; the agent queries show it.
- The operator smoke-tested it with a second device before close.

## Plan

- [ ]

## Log

### 2026-10-12
