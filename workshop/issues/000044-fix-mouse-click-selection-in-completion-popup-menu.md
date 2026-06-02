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

- Add temporary file-based logging to `/tmp/pair_pum_debug.log` inside the `<LeftMouse>` mapping. This log will capture the outputs of `pum_visible()`, `pum_getpos()`, and `getmousepos()` when clicked.
- Investigate these log outputs to determine why the previous coordinate check failed or why the mapping did not fire.
- Correctly calculate the click coordinates and adjust the key mapping behavior. If the click coordinates are inside the popup menu boundary, we will translate the click to a confirm action. If they are outside, we will fallback to standard cursor navigation.
- Ensure that spelling suggestions can be successfully clicked and confirmed on a single mouse click.

## Done when

- [x] Mouse clicks on spelling suggestion menu items successfully select and confirm the item.
- [x] No regression on cursor positioning when clicking elsewhere.

## Plan

- [x] Add debugging log statements to `<LeftMouse>` mapping in `nvim/init.lua`.
- [x] Ask user to click and inspect log outputs.
- [x] Implement and verify the final coordinate/mapping correction.

## Log

### 2026-06-02

- Created issue #000044 to investigate and fix the `<LeftMouse>` click selection menu behavior.
- Added temporary file-based logging to `/tmp/pair_pum_debug.log` and had the user test it.
- Identified that returning `<LeftMouse>` alongside `<C-y>` caused Neovim to process the mouse click first, moving the cursor and immediately closing the popup menu (rendering `<C-y>` a no-op).
- Fixed the issue by simulating relative scroll/navigation keypresses (`<C-n>` / `<C-p>`) followed by confirmation (`<C-y>`) when clicking inside the popup menu coordinates, without propagating the standard cursor-moving `<LeftMouse>`.
- Verified that spelling suggestions (e.g. "sesion" correcting to "session") and other completion menus work seamlessly with mouse clicks.
- Checked that clicking outside the popup menu correctly dismisses it and positions the cursor normally.
- Verified all tests pass successfully.
