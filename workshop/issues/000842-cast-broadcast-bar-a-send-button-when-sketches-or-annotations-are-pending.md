---
id: 000842
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'a7b8f3d11de248ab1ccb334a734cbd63d89817de' # card fields mirrored from issue-cards; edit via sdlc
---

# Cast broadcast bar: a ⏎ send button when sketches or annotations are pending

## Problem

## Spec

**A ⏎ send button on the cast broadcast bar when sketches or annotations are pending (operator, 10-23).** In the
terminal cast (`LIVE ⏸ …`) and the Chrome cast (`Chrome ⏸ …`), when there are sketches or annotations pending
submission, a ⏎ button appears on the broadcast bar. Clicking it does exactly what Shift+Ctrl+Alt+Return does:
send the draft nvim content plus the pending sketches and annotations. The hotkey is hard to discover; the button
makes the action visible. It disappears when nothing is pending.

## Done when

- With pending sketches or annotations, the ⏎ button shows on both broadcast bars and clicking it sends the same
  payload as the hotkey (test); with nothing pending it's absent (test).
- A live look on a real cast (operator, briefly).

## Plan

- [ ]

## Log

### 2026-10-23
