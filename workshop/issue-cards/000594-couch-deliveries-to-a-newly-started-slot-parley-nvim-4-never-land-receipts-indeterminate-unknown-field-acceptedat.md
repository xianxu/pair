---
id: 000594
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
started: 2026-10-15T18:58:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
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
