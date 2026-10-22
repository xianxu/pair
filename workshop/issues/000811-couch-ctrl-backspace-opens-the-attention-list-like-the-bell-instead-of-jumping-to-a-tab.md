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

Operator request (10-22, corrected the same day): rebind **Ctrl+Return** under Couch to **open the attention list**,
exactly as clicking the bell icon does (the same panel Ctrl+Alt+a opens, pair#673/#733). Today Ctrl+Return jumps to
the tab that needs attention (answers the newest page, pair#221); the operator doesn't use that, since notifications
come often and jumping into a tab is too heavy. **Ctrl+Backspace is unchanged** (it jumps to the previous tab, which
the operator uses).

- Ctrl+Return opens the attention panel (toggle: pressing it again closes it, like the bell). The jump-to-tab
  behaviour is removed from that chord; jumping stays available from inside the panel.
- Every place Ctrl+Return is recognised today keeps recognising it (encodings, focused panes); check the
  actor/page answering paths that use Ctrl+Return (pair#221, atlas/couch.md) and say what replaces them, if anything.
- README's shortcut tables, the hosted-session note and atlas/couch.md updated; Ctrl+Alt+a keeps working.

## Done when
- A test drives Ctrl+Return and sees the attention panel open (and close on a second press), with no tab switch.
- Ctrl+Backspace still switches to the previous tab (test).
- Docs name the new behaviour.

## Plan

- [ ]

## Log

### 2026-10-22
