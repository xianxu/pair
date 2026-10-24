---
id: 000848
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '692cc55a7e505ccb675b78dcb18d97d545f839ad' # card fields mirrored from issue-cards; edit via sdlc
---

# couch recover plan re-enters orphaned merges from merge_intents

## Problem

After a full shutdown, merge runners that were in flight need re-entry by hand: nothing in the restart plan names them. ariadne#565 (PR 419, merge be2b8436) now records each in-flight `sdlc merge` as a per-branch intent file held by the runner; a killed or interrupted runner leaves an orphan. `sdlc state --json` carries the slot's own as `merge_intents[]`, each with `state` (live, orphaned, stale or unreadable), `branch`, `issue`, `head`, `root`, `workspace`, `pid`, `started_at`, `reentry` (orphaned only: the exact command, run in `root`) and `note` (set when the machine queue is paused). `sdlc landing status` prints the repository's orphans as lines.

`couch --recover-plan-from-sdlc` reads `sdlc fleet inventory --json` (cmd/internal/couchcore/recoverplan_fake.go models it), which does not carry merge intents yet.

## Spec

The restart plan lists each orphaned merge intent with its slot and its re-entry command, so a restart re-enters every interrupted landing once.

- Decide the source: either ariadne's `sdlc fleet inventory --json` gains `merge_intents[]` per workspace (an ariadne pair-side change, then Couch reads it), or Couch runs `sdlc state --json` per slot. Prefer the inventory: it is already the plan's one source.
- A `note` (queue paused) carries through to the plan line; re-entry waits for the resume.
- live and stale intents are not re-entry steps; unreadable ones are reported for inspection.
- Extend FakeFleetSDLC's state model with merge intents.

## Done when

- `couch --recover-plan-from-sdlc` names each orphaned merge with its slot and the exact `sdlc merge --branch B --yes` re-entry, tested through the fake.
- A paused queue shows its note in the plan line.

## Plan

- [ ]

## Log

### 2026-10-23
