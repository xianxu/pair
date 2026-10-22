---
id: 000811
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '832c78929fe1f6e88ba446d0b130f6d115966537' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch: Ctrl+Backspace opens the attention list (like the bell) instead of jumping to a tab

## Problem

## Spec

Operator request (10-22): rebind **Ctrl+Backspace** (the Mac Delete key with Ctrl; the operator calls it Ctrl+Delete)
under Couch to **open the attention list**, exactly as clicking the bell icon does (the same panel Ctrl+Alt+a opens,
pair#673/#733). Today Ctrl+Backspace jumps to the tab that needs attention; the operator doesn't use that, since
notifications come often and jumping into a tab is too heavy.

- Ctrl+Backspace opens the attention panel (toggle: pressing it again closes it, like the bell). The jump-to-tab
  behaviour is removed from that chord; jumping stays available from inside the panel.
- Both encodings still recognised (legacy `^H` and Kitty `CSI 127;5u`), as today.
- README's shortcut tables, the hosted-session note and atlas/couch.md updated; Ctrl+Alt+a keeps working.

## Done when

- A test drives Ctrl+Backspace in both encodings and sees the attention panel open (and close on a second press),
  with no tab switch.
- Docs name the new behaviour.

## Plan

- [ ]

## Log

### 2026-10-22
