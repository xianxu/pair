---
id: 000771
status: codecomplete
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '7a6666e69c9ff2ef13893c1c91daf63b3912293e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T09:04:08-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/broadcast.TestHubRandomInterleavings]
flow: {kind: quick, provenance: inferred, spec: "f0fe9253", done: "a0bbe98b"}
actual_hours: 1.27
---

# flake: TestHubRandomInterleavings/30

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/broadcast.TestHubRandomInterleavings/30: Same exact integrated train234 head d6efc430bb4fbd8b1e943981688eec3914f8c1ca: recorded suite attempt88fb48b8719d5a57732691bd0fe8cb2f raw inflight run.log failed only broadcast.TestHubRandomInterleavings/30 at hub_test.go405 step4, pointer callback0 with armed=true shown=false. On that unchanged detached checkout, exact anchored seed30 count100 passed in1.126s; broader seeds30/130/230/330/430 count100 also passed5.203s. Underlying manual-clock/callback ordering is unproven; register only the evidenced subtest, not sibling seeds. Full suite continues without cancellation; no production source or timeout change.

## Spec

Make the manual timer's completion proof independent of goroutine scheduling. Hypothesis: fireGrace queues a buffered timer value, then h.sync queues an unrelated request; Hub.run may select the request first, so the helper returns before OnPointerHidden. Preserve the real Hub timer-select path and all500 seeded invariants. No retries, sleeps, timeout increases or production policy changes.

Use an unbuffered timer rendezvous after querying the active timer on the hub loop, followed by the existing completion barrier. Canceled, never-armed, fired and ended timers are no-ops. ARCH-DRY/PURE: keep one shared manual-clock helper for both watches and test its protocol through controlled channels. ARCH-STATE: explicitly order delivery and completion; no task survives the helper/test scope. No new product state, IO or durable runtime artifacts.

Alternatives: polling callback counts changes the model and can hide missing effects; invoking timer handlers directly skips the real select path. A rendezvous proves consumption while preserving production dispatch. Add a local catalog guard against buffered time channels in the manualClock constructor/after seam, with documented syntax limits.

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [x] Reproduce request-before-timer ordering with a controlled fake loop and retain RED evidence.
- [x] Correct manual-clock rendezvous, covering active/canceled/fired/ended behavior and both real watches; add the local lint rule and atlas scope.
- [x] Verify500 seeds repeatedly, broadcast under race detection and lint/inventory.

Recorded full-suite success, close review and landing remain gates tracked in Log; the recorded pass must precede landing.

## Log

### 2026-10-21

- Atlas-only recovery narrow re-close returned SHIP with one Minor advisory: trailing whitespace in the generated review record. Trimmed prose whitespace and added its prevention lesson; no executable change. Integrated protocol/all500-seed checks9.012s, lint4.869s and inventory pass. New recorded full fallback lease-1626987605 on37825526aaaf remains retained (exec80392), attemptd8d4426ccb53656af49ceca443824846; required pass still precedes landing.
- 2026-10-21: closed — Atlas-only append conflict with landed #744 resolved preserving both sections; no Go conflict and no own source change. Integrated protocol/pointer/all500-seed checks pass9.012s, lint/catalog4.869s and inventory pass. Branch diff whitespace check passes excluding binary-owned review transcript. Prior final-source recorded full suite passed10m27s without rerun; new integrated affected check retained and required before landing. TL owns project. (fixed-then-ship) (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

- Final-source recorded full suite30ea382b372be802cb4c2b4568c24633e9122d8a PASSED10m27s, exit0, attempt1525ef6aee32b4dd0dca0ef93628db73, log hashbba801866917, no flake rerun. Its fallback suite proof made the reviewed docs-only head ready. Train238 subsequently held this car at build: integrating landed #744 confirms only an atlas append collision; Go catalog merges cleanly. Preserved both #744/#771 rule sections. Train238 retained only #768; #769 was also held. No unlanded code was stacked. Required checks/re-close follow this main integration; #752 remains parked.
- 2026-10-21: closed — Controlled request-before-timer RED, then rendezvous/completion GREEN; protocol count100 passes0.199s, watches count20 passes0.204s, all500 seeds count20 passes275.755s (10000 executions), full broadcast race147.847s, repolint6.081s, inventory/diff pass. Recorded full fallback lease-2145778985 on30ea382b retained and must pass before landing; review overlap authorized by tl-worker. Atlas rule mapped; TL owns project. No production policy or timeout change. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

- Controlled RED: TestManualClockWaitsForTimerConsumption fails immediately with fireGrace returned after a request, before timer consumption. The fake loop deliberately processes the request before the timer, with synctest quiescence proving the helper returned; this isolates the scheduling race without sleeps or probabilistic seed replay. GREEN after an unbuffered receive rendezvous followed by h.sync. The active-channel query runs on the hub loop, so canceled/never-armed/fired/ended timers remain no-ops. End-during-delivery releases through Done. Production Hub.run is unchanged.
- Protocol tests count100 pass0.199s, both-watch tests count20 pass0.204s, lint rule RED2 positive cases then GREEN5 cases; full repolint6.081s. Full broadcast package with race detection passes147.847s. Inventory and diff checks pass. All500 seeds count20 remains retained as targeted exec43708; no recorded full suite queued yet.

- #767 landed PR520 and its registry entry retired before starting this main-based issue. Authenticated TL dispatch8b70bb19 assigns this issue next while #752 remains parked.
- The failed train234 full candidate was d6efc430bb4fbd8b1e943981688eec3914f8c1ca (#765+#768, not held #767). Attempt88fb48b8719d5a57732691bd0fe8cb2f failed in9m47s, log hashf1a767397c5d. Same-head exact seed30 count100 passed1.126s and all500 seeds passed10.528s. The parser coarsened indented subtest output to the top test, so the child registration was replaced by one parent registration. This broad scope is temporary pending this fix; the original failure remains recorded.

## Revisions

### 2026-10-21 — verification sequencing

All500 seeds count20 passed275.755s (10000 seeded executions) and the full broadcast race run passed147.847s. Split the completed targeted verification from the suite/review/landing gates so close can overlap the queued recorded check under tl-worker. The affected runner selected the full suite due the existing runtimebundle embed graph gap; retained exec14819 on source30ea382b372be802cb4c2b4568c24633e9122d8a must pass before landing. No scope or product policy changed.
