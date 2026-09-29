---
id: 000350
status: open
deps: []
github_issue:
created: 2026-09-29
updated: 2026-09-29
estimate_hours:
card_mirror: '599fbe1dbc7b7b1617a8b9889c1a9798ad591a9f' # card fields mirrored from issue-cards; edit via sdlc
---

# Make binding observation states and recovery diagnostics explicit

## Problem

The #346 M2 SHIP review accepted two nonblocking classes of follow-up (BR-12/BR-15/BR-16): watcher phases and chosen-ID materialization are represented by combinations of fields, and persistent native-storage problems only recommend retry without identifying the offending root or an explicit fresh-start option.

## Spec

Give the producer an explicit observation/materialization state that consumers derive from, eliminating repeated unknown-state predicates. Cover watcher epoch/correlation/confirmation phases without creating a competing identity authority. Propagate diagnostic root/entry context to refused restart actions and document an explicit fresh-start escape hatch. Preserve #346 behavior: filenames suffice for chosen-ID presence, only confirmed absence permits a replacement UUID, unknown refuses before side effects, and established/requested-resume UUIDs survive optional native-storage failures.

## Done when

- Observation phases and materialization states have a single typed authority consumed by launcher and Couch.
- State-transition tests preserve missing, present, partial, unreadable and confirmed cases and stale-observer guards.
- Persistent refusal diagnostics identify the failed root/entry and a supported explicit fresh-start action.
- No new transcript parsing requirement or durable identity registry is introduced.

## Plan

- [ ] Design the state projection and diagnostic flow against existing APIs; implement with transition and production-boundary tests.

## Log

### 2026-09-29

Captured nonblocking #346 M2 SHIP review findings. No implementation started; this is not required to close #346.
