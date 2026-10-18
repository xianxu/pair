---
id: 000576
status: working
created: 2026-10-15
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T19:21:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Couch switcher: lifecycle actions run in the background per slot; the UI never blocks

## Problem

While the operator runs a lifecycle action in Couch's switcher (bring a slot up, relaunch, resume), the switcher holds
the screen until the action finishes or fails: the operator can't switch to another slot or the hub meanwhile
(operator, 2026-10-15). A slot's lifecycle is already modelled as states, and startup brings many slots up in the
background, so a single slot's action needn't block the UI.
