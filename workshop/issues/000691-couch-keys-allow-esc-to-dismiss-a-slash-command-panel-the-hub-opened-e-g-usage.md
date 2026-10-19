---
id: 000691
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '50176f54098329677bab3603bbb547ae7206be21' # card fields mirrored from issue-cards; edit via sdlc
---

# couch --keys: allow Esc to dismiss a slash-command panel the hub opened (e.g. /usage)

## Problem

The hub reads Claude quota by running /usage in an idle Claude slot with `couch --keys SLOT clear-line text:/usage submit` and reading it with `couch --peek`. The /usage panel then stays open ("Esc to cancel"), and `couch --keys SLOT esc` refuses ("wait until the agent composer is empty and its turn finishes"), so only the operator can close it. On 2026-10-18 two slots (brain:1 since 12:38, parley.nvim:4) sat with open panels, raising Couch's 10-minute waiting notice and needing the operator's Esc.

## Spec

Let `couch --keys` send Esc (only Esc) to a slot whose agent shows a dismissable modal (a slash-command panel recognized by the composer recognizers) even though the composer is not empty, so the sender can close a panel it opened. Keep refusing every other key in that state. Alternatively, or as well, a read-only `couch --usage` that runs /usage, captures the panel, and dismisses it in one operation.

## Done when

- After `--keys SLOT clear-line text:/usage submit`, `--keys SLOT esc` closes the panel and the slot returns to an idle composer (recognizer fixture plus a live check).
- Other keys still refuse while a panel is open.

## Plan

- [ ]

## Log

### 2026-10-18
