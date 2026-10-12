---
id: 000498
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '93fca5815b4e8415f60dc045091dd4576e8fb881' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-12
