---
id: 000717
status: codecomplete
created: 2026-10-19
updated: 2026-10-20
estimate_hours: 1.24
github_issue:
started: 2026-10-20T01:12:01-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 0.97
tracker:
    version: 1
    completion:
        token: close-5bc4f3926bae
        repository: github.com/xianxu/pair
        reviewed_head: 60203319cfd4e0d7fc137c51b151df8986f54377
        evidence_commit: 7401b2f0cc2ec19bce73922856f47f367192775b
---

# couch --keys: support special keys (tab, shift-tab, arrows, backspace, function keys, ctrl/alt chords)

## Problem

`couch --keys` accepts only `text:…`, `enter`, `submit`, `up`, `down`, `esc`, `clear-line` and `wait:MS`. The hub
regularly needs keys outside that set to drive a slot's agent UI without the operator: Tab and Shift+Tab (Claude's
mode cycling, completion), Left/Right, Backspace, Ctrl+C / Ctrl+O style chords, Page Up/Down, and function keys
(Codex's `f2 to view` warnings). Today each such moment becomes a "please press X" ask to the operator (10-19: the
operator pressed Esc on a `/usage` panel for the TL).
