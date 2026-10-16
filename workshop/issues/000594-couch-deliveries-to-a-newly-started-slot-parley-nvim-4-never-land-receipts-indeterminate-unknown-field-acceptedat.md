---
id: 000594
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '1f5b350e022057237a4fa2eb6943d2c1534301a0' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T18:58:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "57af933c", done: "664a5825"}
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

- [ ] Omit unset AcceptedAt; prove new wrapper replies decode with the old schema.
- [ ] Name wrapper reply incompatibility and broker release honestly; never retry commit or hide possible effects.
- [ ] Run focused regressions, build, required recorded suite and SHIP review; land and report.

## Log

### 2026-10-15

## Revisions

### 2026-10-15 — measured cause and approved narrow fix

Read-only investigation proved the failure is older supervisor e1311121 decoding newer wrapper receipts, not the staged CLI decoding a newer supervisor. New Message.AcceptedAt used omitempty, which still emits a zero time.Time. All five wrapper-retained receipts were cancelled before submission, with zero AcceptedAt; broker release reused the operator-input event. The exact IDs were recovered from the hub tty capture; only read-only status requests were issued to wrapper 10627. TL approved omitzero, a literal old-schema decoder regression, slot-specific restart advice on reply decode failure, and truthful broker-release cancellation. A failed reply never retries commit; possible input effects retain uncertainty. No general compatibility framework or production lifecycle action. ARCH-ORDER preserves uncertainty; ARCH-DRY reuses transport and delivery reducer.
