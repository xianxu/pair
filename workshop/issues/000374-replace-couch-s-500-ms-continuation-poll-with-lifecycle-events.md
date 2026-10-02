---
id: 000374
status: open
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: 'b581fe81b69b46769bde9eca7e8fe95c485d0ac3' # card fields mirrored from issue-cards; edit via sdlc
---

# Replace Couch's 500 ms continuation poll with lifecycle events

## Problem

`Console.watchContinuations` (`cmd/internal/couchtty/console_continuation.go`) wakes every 500 ms, whether or not any continuation exists. Each tick it asks the continuation provider for the status of **every** attached thread plus every tracked continuation. With 11 slots that is about 22 store scans a second, for the Couch's whole lifetime.

This is the class pair#365 removed from messaging: rediscovering state on a timer instead of being told when it changes.

After #365, live measurement on the operator's 11-slot setup showed messaging spawning nothing. Couch still spent roughly 13–23% of a core idle (`workshop/history/plans/000365-message-lifecycle-measurements.md` once archived). That load is mostly *system* time; the `sample` hot leaves are `fcntl`, `open`/`openat`, `lstat`, `getdirentries` and `mkdir`. The continuation scan is the prime suspect but is not yet attributed.

## Spec

Captured for operator review; not yet designed.

- **Purpose of each check (ARCH-PURPOSE):** state the user-visible failure the poll prevents. The likely ones are a continuation started outside the Console (e.g. `couch --internal continue-thread` from an agent) not appearing, and the switcher's orientation/progress prompt going stale.
- **Event sources:** find who writes continuation state (the continuation store's publication, recovery and retry paths, orientation receipts). Have those writers, or a store-change notification, wake the Console, instead of the Console re-reading every thread (ARCH-DRY: producers own facts).
- **Scope of any check that remains:** only threads with a continuation in flight, bounded and backed off. Never every attached thread on a fixed tick.
- **Measure first:** attribute the idle Couch load among the continuation scan, the 10 s slot-git pass (the operator says this pass is needed and keeps it), and Couch's own diagnostic writes. #370 found the wrapper's diagnostic writer costs several file operations per record. Use the counted-probe approach from #365.

## Done when

- An idle multi-slot acceptance test counts zero continuation-store scans when no continuation is in flight.
- A continuation started outside the Console still appears in the switcher within a stated bound. A test covers it.
- Before/after idle measurement on the same workload, with reproducible commands, attributing Couch's remaining idle CPU.

## Plan

- [ ]

## Log

### 2026-10-01

Filed from pair#365's close at the operator's request ("that 500ms poll smells bad"). Evidence: #365's measurements file and the M3 project note in `cross-slot-work-scheduling.md`.
