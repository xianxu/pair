---
id: 000717
status: open
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
---

# couch --keys: support special keys (tab, shift-tab, arrows, backspace, function keys, ctrl/alt chords)

## Problem

`couch --keys` accepts only `text:…`, `enter`, `submit`, `up`, `down`, `esc`, `clear-line` and `wait:MS`. The hub
regularly needs keys outside that set to drive a slot's agent UI without the operator: Tab and Shift+Tab (Claude's
mode cycling, completion), Left/Right, Backspace, Ctrl+C / Ctrl+O style chords, Page Up/Down, and function keys
(Codex's `f2 to view` warnings). Today each such moment becomes a "please press X" ask to the operator (10-19: the
operator pressed Esc on a `/usage` panel for the TL).
