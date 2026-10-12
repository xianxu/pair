---
id: 000463
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'f0d6283d2f86a98f895ea2933bcb199109e252fc' # card fields mirrored from issue-cards; edit via sdlc
---

# couch --send-to hub: a reserved target the supervisor resolves live to the hub slot

## Problem

ariadne#336 adds a constitution rule: every agent in a Couch slot reports each item's outcome to the hub (the slot Couch started from, where the TL works) and asks the hub instead of an in-pane question prompt, whether the hub dispatched the work or the operator started it by hand. A slot has no way to learn the hub's address. Couch already knows it (`cmd/internal/couchcmd/message_service.go` `hubWorktree`, used for the TL's 64-message allowance and for waiting notices), but exposes it to no slot: no environment variable, no query.

## Spec

- `couch --send-to hub --message TEXT` resolves `hub`, a reserved target, in the running supervisor to the hub's connected binding (`hubBinding`), and delivers as an ordinary peer message (same receipt, admission and allowance rules).
- It resolves live, at send time. An environment variable such as `COUCH_HUB` was rejected: it goes stale when a detached thread is resumed under a Couch started from a different tree.
- Refusals name the reason: no hub (Couch started outside a checkout), the hub isn't connected, or the sender is the hub itself.
- `hub` is reserved ahead of repository-name and prefix resolution. If a repository could be named `hub`, its slots stay reachable as `hub:N`, and `couch --skill` documents the reservation.
- `couch --skill` documents `couch --send-to hub` in its addressing section.

## Done when

- `couch --send-to hub --message …` from a non-hub slot delivers to the hub slot, verified live in a running Couch.
- Each of the three refusals is covered by a test and names its reason.
- `couch --skill` documents the target and the reservation.

## Plan

- [ ]

## Log

### 2026-10-12
- Filed from ariadne#336 (ariadne:1), dispatched by the TL (ops:0). ariadne#336 stays open with a dep on this issue, and closes with its live check once this ships.
