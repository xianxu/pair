---
id: '000067'
status: done
started: 2026-06-23T00:13:50-07:00
created: 2026-06-22
updated: 2026-06-23
estimate_hours: 1.8
actual_hours: 0.81
---

# Fix Codex pair tag quit and resume

## Problem

`Alt+x` can leave a zellij resurrect entry such as `pair-2 (EXITED - attach to resurrect)`.
Pair skips EXITED rows in the picker, but forced resume and name-collision checks
still treat the same row as occupying the tag. The next `pair` launch then cannot
reuse the workspace tag even though the user intended a full quit.

Separately, Codex resume can fail to surface for older sessions. Pair's current
canonical config name is `config-<tag>-codex.json`, but this machine has older
Codex configs such as `config-2-codex-codex.json`. When the canonical file is
absent, Pair misses the saved Codex session even though the state exists.
