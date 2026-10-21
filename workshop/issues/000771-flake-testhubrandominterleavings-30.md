---
id: 000771
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '78a6d5324fb47bcfe8dc7f8ac7d3487183f2f0f8' # card fields mirrored from issue-cards; edit via sdlc
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
- [ ] Verify500 seeds repeatedly, broadcast under race detection, lint/inventory and recorded suite; close and land to retire the broad registration.

## Log

### 2026-10-21

- Controlled RED: TestManualClockWaitsForTimerConsumption fails immediately with fireGrace returned after a request, before timer consumption. The fake loop deliberately processes the request before the timer, with synctest quiescence proving the helper returned; this isolates the scheduling race without sleeps or probabilistic seed replay. GREEN after an unbuffered receive rendezvous followed by h.sync. The active-channel query runs on the hub loop, so canceled/never-armed/fired/ended timers remain no-ops. End-during-delivery releases through Done. Production Hub.run is unchanged.
- Protocol tests count100 pass0.199s, both-watch tests count20 pass0.204s, lint rule RED2 positive cases then GREEN5 cases; full repolint6.081s. Full broadcast package with race detection passes147.847s. Inventory and diff checks pass. All500 seeds count20 remains retained as targeted exec43708; no recorded full suite queued yet.

- #767 landed PR520 and its registry entry retired before starting this main-based issue. Authenticated TL dispatch8b70bb19 assigns this issue next while #752 remains parked.
- The failed train234 full candidate was d6efc430bb4fbd8b1e943981688eec3914f8c1ca (#765+#768, not held #767). Attempt88fb48b8719d5a57732691bd0fe8cb2f failed in9m47s, log hashf1a767397c5d. Same-head exact seed30 count100 passed1.126s and all500 seeds passed10.528s. The parser coarsened indented subtest output to the top test, so the child registration was replaced by one parent registration. This broad scope is temporary pending this fix; the original failure remains recorded.
