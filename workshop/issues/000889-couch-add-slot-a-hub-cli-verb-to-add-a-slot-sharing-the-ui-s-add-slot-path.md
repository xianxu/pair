---
id: 000889
status: open
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: '971730037603a2e0f69c7b268c4a7a2c156f6a88' # card fields mirrored from issue-cards; edit via sdlc
---

# couch --add-slot: a hub CLI verb to add a slot, sharing the UI's add-slot path

## Problem

## Spec

**There's no CLI verb to add a slot (operator and TL, 10-25).** The operator added pair:10 and pair:11 from the Couch
UI and asked the TL to add pair:12; the TL can't: `couch --switch-agent pair:12 --agent codex --confirm` refuses with
`lstat …/worktree/pair-slot12: no such file or directory`, and no `couch` verb creates a slot's workspace (the
`pair-slotN` directory with its sibling checkouts, `.couch` state and resting branch `main-slotN`). Couch operations
should be CLI verbs the hub can run (agent-first operations); the UI action becomes an optional follow-up.

Add a hub-only verb, e.g. `couch --add-slot repo --agent NAME [--agent-args …] --confirm`, that does exactly what the
UI's add-slot does (same code path, not a second implementation): picks the next free number, creates the workspace
and its resting branch, registers the slot, and starts the agent. Refuses if the number exists; reports what it
created. pair#847's bench-start shares the launch path; keep production and bench slots distinct.

## Done when

- The verb adds a working slot identical to one added from the UI (same files, branch, registration), via the same
  implementation; tests over the fake fleet, and one native run adding and then retiring a disposable slot.
- `couch --help` documents it with its cost tag.

## Plan

- [ ]

## Log

### 2026-10-25
