---
id: 000673
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9f8b26622cd907a80c47344b2244536871856809' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T12:05:17-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Couch attention list: a top-right panel of what's waiting on the operator, each item jumps to its slot

## Problem

Ghostty skips OSC desktop notifications sent from the focused tab (confirmed by the operator 2026-10-18), and Couch's hub is nearly always the focused tab, so `couch --notify` never reaches the operator in practice. The operator wants Couch to manage attention itself.

## Spec

Couch keeps an attention list of things waiting on the operator, shown as a panel in Couch's top-right corner (operator decision, ops qa 2026-10-18-10-couch-notify.md):

- Items: the hub's `couch --notify`, a slot waiting on a permission prompt, a slot stuck waiting on input. Each carries its source slot and time.
- Use: one key opens the list; arrows and Enter jump to the slot; mouse where the terminal passes clicks. A count badge shows while closed.
- Lifetime: an item clears when the operator jumped to its slot and the condition cleared, or on dismiss. It survives a Couch restart.
- Share the look and jump behavior with the existing Alt+u attention finder (document submissions waiting on agents) rather than adding a second list style.
- The OSC escape stays as an extra for when the terminal is in the background.

## Done when

- A hub `couch --notify` appears in the panel immediately, and Enter jumps to the hub slot; a permission-prompt slot appears and clears when answered (tests with the Console fakes).
- Items survive a Couch restart.
- Operator live check at the next restart.

## Plan

Durable plan: [workshop/plans/000673-couch-attention-list-a-top-right-panel-of-what-s-waiting-on-the-operator-each-item-jumps-to-its-slot-plan.md](../plans/000673-couch-attention-list-a-top-right-panel-of-what-s-waiting-on-the-operator-each-item-jumps-to-its-slot-plan.md), with the state table and five decisions for the TL. Single pass, one close.

- [ ] Reducer and state-table tests
- [ ] Store
- [ ] Sources (notify, blocked-set diff, wait episode start/end)
- [ ] Console panel, badge, key, mouse, jump
- [ ] Atlas, help, inventory, suite, close
- [ ] Operator live check at the next restart

## Log

### 2026-10-18
