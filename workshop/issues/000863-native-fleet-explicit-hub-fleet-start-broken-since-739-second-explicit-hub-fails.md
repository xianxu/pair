---
id: 000863
status: working
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '94479282d6a5c7dcae81ae54446c85a675575045' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T00:18:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Native fleet: explicit-hub Fleet.Start broken since #739; second explicit hub fails attach at a fresh start

## Problem

Found by pair:6 while writing #830's T6 fleet test (2026-10-24).

1. **Harness:** since #739, `testcouch.Fleet.Start` cannot start an explicit-hub fleet. `Start` waits for the primary hub's agent (`WaitAgent(ctx, 0)`), but #739's startup launches no hub: the switcher shows each configured hub as `hub · Ctrl+Space: start DIR`, with the notice `startup: full-shutdown; missing hubs: …`. This happens even in a fresh namespace. Every `ExplicitHub` fleet test therefore times out at "steward startup": `TestDisposableEditedRestart`, `TestDisposableStartupArguments`, and `TestDisposableCouchSYNTHETICLimitAndBusyForcePreserveDirtyFiles`. Reproduced on clean origin/main 54c4a59ca:
   `PAIR_NATIVE_COUCH_FLEET=1 go test ./cmd/couch -run '^TestDisposableEditedRestart$/claude' -count=1 -v` → `startup_restart_fleet_test.go:44: steward startup: context deadline exceeded; screen=threads`.
   These tests are outside the normal suite, so nothing caught it.
2. **Likely real bug:** with two explicit hubs (`couch PRIMARY WORKER1_DIR`) in a fresh namespace, the second hub's `start-hub` fails. The notices were `hub failed; open it in the switcher to retry: …` and `clean after a failed attach of : attach record/handle process identity mismatch`. The attach target name was empty. The failure evidence from the harness read `final="attach record/handle process identity mismatch"`, with slot and stage empty. A fresh namespace has no records, so `StartInteractive` should have nothing to attach.

Screen at the failure:

```
threads
Ctrl+Y copy last notice / selected error · Ctrl+G hub automation on/off
  hub · Ctrl+Space: start /…/fleet/fixture
  hub · Ctrl+Space: start /…/fleet/worktree/fixture-slot1/fixture
  (no match)
startup: full-shutdown; missing hubs: /…
```

## Spec

- `Fleet.Start` drives the same start gesture the operator uses after #739 (the switcher's start for the primary hub), so explicit-hub fleet tests run again.
- Find and fix the second explicit hub's attach failure at a fresh start.
- Decide whether a fresh namespace (no records at all) should count as a full shutdown. The startup notice currently says `full-shutdown` for a Couch that never ran.

## Done when

- The `ExplicitHub` fleet tests pass natively on Claude and Codex.
- A fresh start with two explicit hubs raises no `hub failed` notice, and a test covers it.
- The fresh-namespace classification is decided, and the notice says what actually happened.

## Plan

- [ ]

## Log

### 2026-10-24 — intake and production-case diagnosis first

Claimed in pair:7, then ran bare start-plan from main-slot7 after #842 finalized SHIP and entered external check/stage wait. Branch base99846a5015f753f05edaf4778618849f377a2f71; no #842 commits stacked. Refresh only deferred the seeded workflow; no generated vocabulary edits to discard. #842 retains recorded lease3238290542 and its own branch; do not duplicate or cancel it for #863.

Hub563a3e8a decides zero history is fresh: hubs start normally, hub automation OFF, no automatic agent input. Full shutdown is known prior records whose sessions are absent; retain #739 no-auto-resume there. Hub843a4c5e prioritizes confirming or ruling out the production caller failure before any fix plan. No production edits until measurement and approved design.

Pair:6 f29ca79c reports second-hub error at375adcb6c with no gesture: fresh couch PRIMARY WORKER1_DIR, automatic queueExplicitHubs -> start-hub -> StartInteractive. Original /Volumes/TestScratch/pc-3413210420 was removed, so only reported teardown text remains (identity mismatch; empty slot/stage, no wrapper log). Single explicit hub separately times out in Fleet.Start. Pair:6 later8175ab45 reports manually starting primary with Ctrl+Space form produces a ready Pair-wrapped Claude fixture, but inherited ActorCLI --actors returns caller lookup: recipient is not live for120s. Treat these as reported, not retained/replayed evidence.

#830 has unlanded Hubs and StartCold helper options. Earlier no-cycle advice was corrected by9bfb9542: its T6 is blocked by caller registration; TL decides sequencing. Do not import/stack #830 source. Production-case experiment will reuse landed Fleet.Start in an owned goroutine, drive the real form through existing Input/Press, cancel/join only Start's observer after fixture readiness, then invoke real caller CLI from inherited fixture credentials. This measures current main without changing production or inventing a duplicate fleet owner. Save full bounded test output outside the cleanup root, and join cleanup.

### 2026-10-24
