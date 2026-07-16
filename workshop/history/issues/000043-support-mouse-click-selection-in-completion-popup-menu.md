---
id: 000043
status: done
deps: []
github_issue:
created: 2026-06-02
updated: 2026-06-02
estimate_hours: 1
actual_hours: 0.5
---

# Support mouse click selection in completion popup menu

## Problem

When the built-in completion popup menu (such as spelling suggestions shown via `z=`, path completion, or word completion) is active, clicking on a menu item with the mouse only highlights/selects the option in the UI, but does not confirm or select/insert the option. The user has to either double-click or press Enter/C-y to confirm the item, which is frustrating when navigating with the mouse.

## Spec

Configure an insert-mode keymap for `<LeftMouse>` in `nvim/init.lua` that checks if the completion popup menu (`pumvisible()`) is active:
- Check if `pum_visible()` is true.
- If true, retrieve the popup menu position and size using `vim.fn.pum_getpos()`, and the click position using `vim.fn.getmousepos()`.
- Determine if the mouse click coordinates (1-based screenrow/screencol) are within the bounds of the popup menu.
- If the click is inside the menu boundary, return `<LeftMouse><C-y>` as an expression mapping to select the item and confirm it instantly.
- If the click is outside the menu boundary, or if the popup is not visible, return `<LeftMouse>` verbatim to preserve default mouse selection and dismissal behavior.

## Done when

- [x] Spelling suggestions completion menu allows selection and confirmation of the item on a single mouse click.
- [x] Clicking outside the popup menu correctly dismisses it and moves the cursor without confirming a selection.
- [x] Existing completions (path/word) also benefit from single-click mouse selection.

## Plan

- [x] Add the `<LeftMouse>` insert-mode expression mapping to `nvim/init.lua`.
- [x] Run test suite to verify no syntax or mapping errors.
- [x] Perform manual verification.

## Log


- 2026-06-02: closed — make test passes, spelling suggestions click selection verified
### 2026-06-02

- Identified that Neovim's default popup menu (PUM) doesn't confirm on click.
- Created issue #000043 to address this using an expression mapping on `<LeftMouse>` paired with `pum_getpos()` and `getmousepos()`.
- Implemented the `<LeftMouse>` insert-mode mapping in `nvim/init.lua` right after the `<CR>` mapping.
- Verified that the Neovim TUI checks compile and pass cleanly via `make test`.
- Updated `atlas/architecture.md` to document the new click-to-confirm behavior of the completion popup menu.
