---
id: 000863
status: codecomplete
created: 2026-10-24
updated: 2026-10-24
estimate_hours: 1.333
github_issue:
started: 2026-10-24T00:18:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
actual_hours: N/A
tracker:
    version: 1
    completion:
        token: close-3329bb9357a7
        repository: github.com/xianxu/pair
        reviewed_head: 8b13b536b24b1a30bf122d7c40acbdc4dd37aafe
        evidence_commit: 2d0c5d775d3948d4315a393e41347b04e19c6892
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
