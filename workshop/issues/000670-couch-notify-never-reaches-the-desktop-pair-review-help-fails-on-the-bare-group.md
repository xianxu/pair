---
id: 000670
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'eaedcf15594258917dc6de5556a9b220bb69d61b' # card fields mirrored from issue-cards; edit via sdlc
---

# couch --notify never reaches the desktop; pair review --help fails on the bare group

## Problem

Two operator-facing gaps, both seen 2026-10-18:

1. `couch --notify` never reaches the operator as a desktop notification. The hub has sent about ten today (each new Needs-you item); the operator saw none, including a test titled "Couch test". Couch reports "Notification written to the operator terminal."
2. `pair review --help` fails with `pair-go: unknown command "review"`, though `pair review apply --help` and `pair review round --help` work: review is registered only as two-word subcommands.

## Spec

1. Find where the notification is lost: the escape Couch writes (OSC 9, 777 or 99), whether zellij or another layer between Couch and the terminal swallows it, and whether the operator's terminal app turns it into a macOS notification. Fix it so a hub `--notify` reliably raises a desktop notification on this machine, or, if that depends on a terminal setting, say exactly which and have `couch --notify` report when delivery is unconfirmed rather than claiming success.
2. `pair review` with no subcommand (and `pair review --help`) lists its subcommands.

## Done when

- A hub `couch --notify` raises a desktop notification on the operator's machine (operator-confirmed), or the command states plainly what the operator must enable.
- `pair review --help` lists the review subcommands; a test covers it.

## Plan

- [ ]

## Log

### 2026-10-18
