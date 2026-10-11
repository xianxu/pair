---
id: 000451
status: working
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: 'c4f2ac6edb3aa6b37da3a8e4da7f3d8871172a14' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T13:19:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "64437f8d", done: "1637714c"}
---

# Couch tells the hub when a slot is stuck waiting for someone

## Problem

Overnight 10-10/11 several slots sat for hours and the TL (ops:0) never knew: ariadne:2 asked a design question it couldn't deliver (its sandbox blocked couch's socket), ariadne:3 waited on a permission dialog to merge, ariadne:1 later waited on a confirm dialog. Message delivery is now robust (pair#433 leaky bucket, pair#438 busy agents, ariadne#318 socket), but a worker that is waiting sends nothing, so no amount of delivery robustness reports it. The TL's fallback would be a periodic peek at every slot; the operator wants the need for that to go away.

## Spec

Couch already watches each slot's terminal (`--peek`, `LastActivity`, composer state). When a slot has been waiting for input longer than a threshold (default 10 min: its turn ended with a question, or a permission or selection dialog is open, or it is idle off its resting branch with no message in flight), Couch sends one notice to the hub slot (the slot Couch started from): `slot ariadne:3 waiting 12 min: <last lines or the dialog's title>`. Once per waiting episode; a new episode starts when the slot works again. Self-addressed hub notices obey #441's rule (not owed when the hub notifies itself). Opt-out per slot.

## Done when

- A slot left on a permission dialog for longer than the threshold produces exactly one hub notice naming the dialog (test with a fake terminal).
- A slot whose turn ended with a question produces one notice; one that ended normally on its resting branch produces none (tests).
- Checked live: the TL receives a notice for a slot left waiting.

## Plan

Durable plan: `workshop/plans/000451-couch-tells-the-hub-when-a-slot-is-stuck-waiting-for-someone-plan.md`.

- [ ] Pure waiting classifier over memory facts + the slot's screen tail (Task 1).
- [ ] Watcher on the message service; one notice per episode, from the waiting slot to the hub (Task 2).
- [ ] Atlas + couch skill; full tests; live check that ops:0 receives a notice (Task 3).

## Log

### 2026-10-11

- Design: supervisor-side watcher, no wrapper protocol change. Waiting = quiet (LastActivity) past threshold AND (unsettled | turn ended with `?` | idle off resting with nothing pending). Hub is not watched (#441 strict form). Opt-out file `<store>/wait-notice-off`; threshold `COUCH_WAIT_NOTICE_AFTER`. ARCH-FUNERAL: episode map in memory, pruned per tick.
