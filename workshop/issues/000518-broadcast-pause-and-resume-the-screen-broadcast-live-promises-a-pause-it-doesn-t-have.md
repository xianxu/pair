---
id: 000518
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'd2fdc68cf24bb9beacb2139cf21ed4574b59e48f' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast: pause and resume the screen broadcast (LIVE ⏸ promises a pause it doesn't have)

## Problem

The screen broadcast's cell reads `LIVE ⏸`, and the ⏸ glyph promises a pause, but the screen broadcast has no pause state. A left click on `LIVE ⏸` stops the broadcast outright: viewers see it end, and the link and its codes (#495) die with it. The web cast does have live and paused (`Chrome ⏸`/`Chrome ▶`), so the two cells use the same glyph for different actions.

The #495 smoke test's ergonomics rule (operator, 2026-10-12) wants a left click on both `LIVE ⏸` and `Chrome ⏸` to pause and resume. #495 kept LIVE's click as stop (the hub's decision (a)) and filed this follow-up.

## Spec

## Done when

- A left click on `LIVE ⏸` pauses the screen broadcast: viewers keep their sessions and are shown it is paused (no frames while paused). Another click resumes it. The cell shows `LIVE ▶` while paused, as Chrome does.
- Stopping stays available as an explicit action (e.g. Ctrl+click, as on Chrome, or a Codes-menu entry), and the README states the rule.
- Tests per click kind; nothing captured while paused reaches a viewer.

## Plan

- [ ]

## Log

### 2026-10-12
