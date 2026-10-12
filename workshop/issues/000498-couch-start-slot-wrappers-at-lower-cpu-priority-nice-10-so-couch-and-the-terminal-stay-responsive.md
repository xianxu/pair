---
id: 000498
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'ba25a57db55630ad2fba552b5200602156899f02' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T13:43:15-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# couch: start slot wrappers at lower CPU priority (nice +10) so Couch and the terminal stay responsive

## Problem

10-12: agents' builds, test suites and load loops saturated the machine (load averages 100–200) and starved Couch's UI and the terminal; slots run at the same priority as the operator's interactive apps. Raising Couch's priority needs root; lowering the slots' doesn't.

## Spec

Couch starts each slot wrapper at nice +10 (configurable, default on); children inherit it, so builds, tests and agents yield to Couch and the terminal. Optionally (opt-in, measured first) a macOS QoS clamp (`taskpolicy`) that also deprioritizes disk I/O; background QoS may pin work to efficiency cores, so measure test-suite time before recommending it. Couch itself stays at normal priority.
**Operator permission (the operator, 10-12):** every operation that changes this machine (creating or mounting a volume, moving or deleting caches or logs, changing launch settings, renicing processes) is proposed first, with the exact command, and runs only after the operator approves it, through the TL. Code changes and tests in the repo don't need that; anything outside the repo does.

## Done when

- New and relaunched slots run at the configured niceness (verified by ps); under a heavy suite Couch stays responsive (measured: a light couch verb's latency); documented.
- Renicing already-running slots, if wanted, is proposed to the operator, never done unasked.

## Plan

Design: every slot launch goes through `Runner.StartBlocked`, which starts `pair-launch-helper` in its own process
group and holds it before exec until Couch acknowledges. A `couchcore.NicedRunner` decorator lowers that group's
priority (`setpriority(PRIO_PGRP, pid, n)`) inside that blocked window, so the target and everything it forks inherit
it with no race and no change to the helper, argv or process identity. Couch's own process is untouched.
`COUCH_SLOT_NICE` (0–20, default 10, 0 = off) is parsed once at composition; a bad value refuses to start.
The `taskpolicy` QoS clamp is left out: the Spec makes it opt-in after measurement, and measuring it needs load this
item may not generate.

- [ ] `couchcore.NicedRunner` + unit tests (fake seam: order before return, failure cancels the launch) and a real
      helper test (the exec'd target reads its own niceness through `ps`).
- [ ] `COUCH_SLOT_NICE` parsing + wiring in `NewCouchWith`; tests for default/off/invalid.
- [ ] README + atlas/couch.md.
- [ ] Propose to the hub: relaunching the live Couch on the new build, then verify slot niceness by ps and a light
      couch verb's latency under the existing load; renicing running slots only if the operator wants it.

## Log

### 2026-10-12
