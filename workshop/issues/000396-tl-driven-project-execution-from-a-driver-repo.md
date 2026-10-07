---
id: 000396
status: open
deps: [pair#362, ariadne#297]
github_issue:
created: 2026-10-06
updated: 2026-10-06
estimate_hours:
card_mirror: 'c75edca1242ee8d0d3a823c4cfb71b3beb649824' # card fields mirrored from issue-cards; edit via sdlc
---

# TL-driven project execution from a driver repo

## Problem

The operator wants to work in larger pieces: drive a whole project, such as
cross-slot-work-scheduling, from one TL slot instead of shepherding each issue by
hand.

The pieces exist or are planned:
- sending work to slots and reading its state (pair#362);
- ownership and observations in SDLC (ariadne#277, ariadne#279, ariadne#280);
- verification authority and a smoke-test state (ariadne#297).
Nothing yet ties them into a working loop for one coordinator.

## Spec

Captured for operator review; no implementation is authorized by this issue
creation.

- **The TL lives in a driver repo, not in a product repo's `:0`** (operator,
  2026-10-06). A product repo's `:0` doubles as the smoke-test bench for developer
  tools: pair#387's test swapped `pair:0`'s branch under it. A product- or
  business-oriented driver repo keeps the TL's checkout stable and can hold the
  `product` charters (ariadne#15).
  - **Open:** whether projects move into the driver repo or stay in their
    center-of-gravity repo with the driver only coordinating.
  - **Open:** whether brain is a candidate. The constitution keeps SDLC process
    artifacts out of brain, which argues against.
- **The loop.**
  - The TL assigns with #362's mechanisms and sets `verify` per item, using the
    product-lead judgement (ariadne#298).
  - It reads progress from repository state (claim, milestones, gates).
  - It surfaces the smoke-test queue (ariadne#297's query) and hands the operator
    each item's `## Smoke test` steps.
  - It merges what is verified.
- **State is pulled from repositories; messages are hints** (operator, 2026-10-06).
- **This issue is filed in pair for now**, where the scheduling project lives. Move
  it to the driver repo once that exists.
- **Out of scope.**
  - A smoke-test bench queue (deferred).
  - Multi-machine coordination (cross-slot-work-scheduling's boundary).
  - Ergonomics and scale tuning, until real projects show the need.

## Done when

- The operator drives one real multi-issue project end to end from a single TL slot
  in a driver repo: assignment, progress, smoke-test hand-offs and merges, with no
  per-issue shepherding outside it. Any project will do (operator, 2026-10-06); the
  next one at hand is the natural candidate.
- Where the TL lives and where project files live are decided and recorded.
- Gaps found in the run are filed as issues, not worked around in the TL's prompts.

## Plan

- [ ]

## Log

### 2026-10-06

Filed from pair#362's brainstorm. #362 was narrowed to mechanisms (sending and state
pulling); this issue is the loop on top. Depends on pair#362 and ariadne#297, and
uses ariadne#298. Details left local for the operator to refine.
