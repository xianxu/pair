---
id: 000501
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '0dc5a0254998dc3ad847da650f943156b7d788f6' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T14:38:05-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# broadcast-web: a button beside the notes popup in Couch's tab bar brings the cast Chrome to the front

## Problem

The operator (10-12): during a web cast they move between Couch and the cast Chrome often; there's no quick way from Couch to get to the browser. They want a small button in the tab bar, right beside the notes popup (`Chrome ⏸ 💭N`), that focuses the cast Chrome.

## Spec

- A small button cell beside the 💭N notes popup in the `Chrome ▶/⏸` group; clicking it brings the cast Chrome's window to the front with the cast tab active (CDP `Page.bringToFront` / `Target.activateTarget` for the tab; macOS app activation of that Chrome process if needed so the window actually comes forward).
- This is the operator's explicit action, so raising the window is intended (unlike background test runs, which must never take focus).
- Choose a glyph that reads as 'go to browser' and fits the tab bar's width rules (#491's narrow-width collapse keeps it with the Chrome group).
- Also reachable from the Chrome cell's menu.

## Done when

- Clicking the button brings the cast Chrome window and tab to the front; tests for the cell and the action (fake browser); the operator smoke-tested it.

## Plan

- [ ]

## Log

### 2026-10-12
