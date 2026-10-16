---
id: 000594
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'ce799689cb9e15744f794a4dc5e13f8ba4206382' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch: deliveries to a newly started slot (parley.nvim:4) never land, receipts 'indeterminate' (unknown field AcceptedAt)

## Problem

2026-10-15: the operator started a new slot, parley.nvim:4 (Claude Code, ~17:12). Every `couch --send-to parley.nvim:4`
from the hub (ops:0) is accepted ("queued", ids 7d561939…, 115b6364…, 12c756aa…) but never appears in the pane, while
the operator's own typing works and deliveries to every other slot arrive. `couch --message-status ID` prints
"indeterminate" and, with `--json`, fails: `json: unknown field "AcceptedAt"`.

The hub's `couch` CLI is pair:0's staged build (6d65c4c4, main + #573 + #544, built 16:28); the running supervisor and
the new slot's wrapper may be a different build. So two problems:
1. A delivery to a live slot silently never lands, with no failure reported to the sender.
2. CLI/supervisor version skew breaks receipt decoding (strict JSON refusing a newer field), hiding the status.

## Spec

- Find why parley.nvim:4 never receives (wrapper registration, delivery routing, or version skew).
- A delivery that can't be admitted within its deadline must end in a definite failure the sender can see, never
  "queued" forever.
- Receipt decoding tolerates additive fields from a newer supervisor (or the CLI refuses loudly with "CLI older than
  supervisor"), so status is never silently indeterminate.

## Done when

- A regression reproduces the skew (newer receipt field) and the undeliverable slot; the sender gets a definite outcome.
- Live: a newly started slot receives a hub message.

## Plan

- [ ]

## Log

### 2026-10-15
