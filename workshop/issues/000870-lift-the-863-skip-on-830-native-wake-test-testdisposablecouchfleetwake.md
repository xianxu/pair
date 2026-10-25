---
id: 000870
status: working
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: 'b42f6466728bc4de3693bee04e94b988e7041d79' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T07:08:58-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "eeca357d", done: "1b7e3866"}
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

- [x] Remove the pair#863 skip; T6's first phase uses Fleet.Start, waits for the second hub agent and both hub registrations (WaitActors 2), then the main hub launches the plain worker. The now-unused wakeFleetStartHub helper goes; wakeFleetOpenSwitcher stays for the Resume-from-disk phase.
- [ ] Run natively, unsandboxed, PAIR_NATIVE_COUCH_FLEET=1, for Claude and Codex (go test ./cmd/couch -run ^TestDisposableCouchFleetWake$).
- [ ] Inventory, pre-close, close, land.

## Log

### 2026-10-24
