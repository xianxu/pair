---
id: 000044
status: working
deps: []
github_issue:
created: 2026-06-02
updated: 2026-06-02
estimate_hours: 1
---

# Fix mouse click selection in completion popup menu

## Problem

The previous `<LeftMouse>` mapping in `nvim/init.lua` designed to select and confirm popup menu items did not work for the user in practice. We need to debug why the mapping is not firing, or why the click coordinates check is not matching, and implement a robust fix.

## Spec

- Add temporary file-based logging to `/tmp/pair_pum_debug.log` inside the `<LeftMouse>` mapping to capture `pum_visible()`, `pum_getpos()`, and `getmousepos()` when clicked.
- Investigate log outputs to determine why the check failed.
- Adjust the click detection / key mapping behavior so that clicking a spelling suggestion confirms the selection.

## Done when

- [ ] Mouse clicks on spelling suggestion menu items successfully select and confirm the item.
- [ ] No regression on cursor positioning when clicking elsewhere.

## Plan

- [ ] Add debugging log statements to `<LeftMouse>` mapping in `nvim/init.lua`.
- [ ] Ask user to click and inspect log outputs.
- [ ] Implement and verify the final coordinate/mapping correction.

## Log

### 2026-06-02

- Created issue #000044 to investigate and fix the `<LeftMouse>` click selection menu behavior.
