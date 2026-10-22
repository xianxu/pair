---
id: 000792
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'e121910931ab6f5cd5d8f8601f227944505d1fdb' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch: no verb reaps proven orphans of a live slot's previous incarnation; extend --reap-orphans

## Problem

## Spec

10-21 ~20:15: the operator restarted echo:2 (hub ops-echo). `couch --resources` then listed **proven orphans** of its
previous incarnation: zellij 66561, pair 66575 and 66576 (`pair wrap`, `pair term`), nvim 66577 and 66583, and Claude
66695 (~350 MB RSS). No verb ends them:
- `couch --reap echo:2 --confirm` refuses `not-offered: echo:2 is live` (it would target the live slot);
- `couch --reap-orphans --confirm` covers only unattributed embedded nvim;
- from a different hub, `--reap` refuses with "slot belongs to hub …" (correct).

Fix: `--reap-orphans --confirm` (run by the owning hub, or the machine steward) also ends **proven orphans of a
previous incarnation** of a live slot, the set `--resources` already proves, each signal fenced on the process
identity the sweep recorded, never touching the live incarnation's processes. Report what was ended.

## Done when

- With a live slot plus a proven orphan set from its prior incarnation (fake process table), `--reap-orphans --confirm`
  ends exactly the orphans and none of the live processes (test).
- A pid reused since the sweep is not signalled (identity fence test).

## Plan

- [ ]

## Log

### 2026-10-21
