---
id: 000870
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '693c80fe6b5e8aeb73e009d6dfd9fbb8a29f6778' # card fields mirrored from issue-cards; edit via sdlc
---

# Lift the #863 skip on #830 native wake test (TestDisposableCouchFleetWake)

## Problem

#830's native acceptance test `TestDisposableCouchFleetWake` (cmd/couch/fleet_acceptance_test.go, in `make test-couch-fleet`) is skipped with a reason naming pair#863. #863 landed as PR 586 at 109986901: fresh-namespace startup now starts both configured hubs automatically, while a known prior shutdown still keeps them parked. The skip is now the only thing stopping the run (TL: pair:6 owns lifting it).

## Spec

- Remove the skip. With #863, a fresh two-hub fleet starts both hubs by itself, so T6's first phase uses `Fleet.Start` (and waits for the second hub) instead of driving the switcher start form. After the full quit, Couch still starts cold with nothing back, and the main hub is resumed from the switcher (Resume from disk) as before.
- Run it natively for Claude and Codex.
- Known unrelated native issues stay recorded: pair#866 (zellij startup panic) and pair#867 (single idle disagreement).

## Done when

- `TestDisposableCouchFleetWake` passes natively for Claude and Codex. It covers the full quit marking all three threads asleep, a cold start bringing nothing back, the main hub resumed from the switcher, `couch --wake` waking exactly the other hub, the plain worker staying asleep, and a repeat starting nothing.

## Plan

- [ ]

## Log

### 2026-10-24
