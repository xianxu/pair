---
id: '000044'
status: done
created: 2026-06-02
updated: 2026-06-02
estimate_hours: 1
actual_hours: 0.5
---

# Fix mouse click selection in completion popup menu

## Problem

The previous `<LeftMouse>` mapping in `nvim/init.lua` designed to select and confirm popup menu items did not work for the user in practice. We need to debug why the mapping is not firing, or why the click coordinates check is not matching, and implement a robust fix.
