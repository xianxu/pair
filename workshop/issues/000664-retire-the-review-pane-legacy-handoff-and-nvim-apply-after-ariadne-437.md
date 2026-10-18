---
id: 000664
status: open
deps: [ariadne#437]
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'd0b546559a9f8dd08d526cffb4a09ee64d2643f1' # card fields mirrored from issue-cards; edit via sdlc
---

# Retire the review pane legacy handoff and nvim apply after ariadne#437

## Problem

## Spec

This is the second step of pair#661 M2, ordering (a), approved by the TL on 2026-10-18. pair#661 M2 moves the review pane onto `pair review apply`, but keeps the legacy handoff path working, because agents running xx-fix from before ariadne#437 still apply through it. Once ariadne#437 lands (the xx-fix workbench protocol on `pair review apply`), retire the legacy path:
- the handoff and landed halves of `nvim/review/handoff.lua`, `PAIR_REVIEW_HANDOFF_PATH` and `PAIR_REVIEW_LANDED_PATH` (artifactpath), and `did_land`;
- restore_controller's pending-handoff and pending-landed checks;
- `apply.apply` (the buffer apply), `reconcile.lua`, `gate.lua` and on_defer, pending_records, has_pending, the winbar pending state, and `set_base`/v0;
- the `agent_applied` poke, the dropped and conflict notifications, and the tests for all of these (review-reconcile, review-handoff; Lua reconcile, gate, handoff_acceptance, handoff_context).

Add the rule test: nvim never writes the reviewed file or buffer for an agent round, only for the operator's own actions (save on send, marker accept/reject, comments, definitions).

The design is in pair#661's plan, `workshop/plans/000661-s4-one-edit-path-apply-and-derived-rounds-plan.md`, under "M2 as built".

## Done when

- None of the retired code or env vars remains, and `make test-review` and `make test-lua` pass.
- The rule test fails if an agent-round path writes the reviewed buffer.
- A live review round with an ariadne#437 agent applies through `pair review apply`, and the pane only reloads and decorates.

## Plan

- [ ]

## Log

### 2026-10-18
