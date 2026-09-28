---
id: '000316'
status: done
started: 2026-09-23T21:53:28-07:00
created: 2026-09-23
updated: 2026-09-23
actual_hours: 0.22
---

# Couch Alt+n refuses a fresh thread for up to 60s after its first round

## Problem

A thread stays **provisional** (no ledger `binding` row) for up to 60 s after
its first round has already completed. During that gap Couch's Alt+n refuses
("its agent has not completed a turn yet…") although the turn is done, and
every other consumer of the binding (Couch's parked-row resume, the saved
`config-<tag>-<agent>.json`) lags the same way.

Observed on `couch-b929512be2acbf12` (2026-09-23, PDT): launch ≈21:47:03,
first send 21:48:01, round completed shortly after, binding + config written at
**21:49:04**, exactly one `SlowPoll` after the 60 s startup window closed at
≈21:48:03.
