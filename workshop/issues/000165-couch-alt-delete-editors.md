---
id: 000165
status: open
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-01
estimate_hours:
---

# Support Alt Delete in Couch editors

## Problem

Couch edit boxes lack a quick keyboard action for clearing their contents.
Deleting text manually is especially cumbersome when editing a prefilled value
or a long multiline field.

## Spec

- Support `Alt+Delete` consistently in every Couch edit box.
- In a single-line edit box, `Alt+Delete` clears all text.
- In a multiline edit box, `Alt+Delete` clears only the line containing the
  cursor and preserves the other lines.
- Define predictable cursor placement after clearing text or a line.

## Done when

- `Alt+Delete` clears the complete contents of every single-line edit box.
- `Alt+Delete` clears only the current line in every multiline edit box.
- Clearing the first, middle, and last line preserves valid surrounding
  newlines and leaves the cursor in a usable position.
- Automated tests cover single-line fields and multiline boundary cases.

## Plan

- [ ] Inventory Couch edit-box implementations and their key handling seams.
- [ ] Add failing tests for single-line clearing and multiline line clearing.
- [ ] Implement shared `Alt+Delete` behavior across all edit boxes.
- [ ] Verify cursor placement and first, middle, last, and only-line cases.

## Log

### 2026-09-01

Captured during Couch dogfood testing. The shortcut is intentionally
field-aware: clear everything for a single-line value, but only the current
line for multiline text.
