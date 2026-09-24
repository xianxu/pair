---
id: 000315
status: working
deps: []
github_issue:
created: 2026-09-23
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T20:26:43-07:00
---

# Allow fresh slot launches to establish their reserved conversation

## Problem

pair:2 fresh-slot times out because Pair rejects the newly reserved conversation ID as not already established.

## Spec

Fresh Couch registration accepts an exact owned reserved or established claim, establishing only reserved claims. Resume keeps its existing read-only established-only validation. Missing, malformed and mismatched claims are refused without adoption. Route the fresh launcher through this explicit operation and test the real filesystem boundary behind its runtime seam (ARCH-PURPOSE, ARCH-MOCK).

## Done when

- New fresh-slot reservations reach the launcher handoff and become established.
- Established-address fresh launches still work; resume and invalid-claim refusal remain unchanged.
- Regression tests exercise production claim storage through the launcher.

## Plan

- [ ] Reproduce reserved fresh failure in a launcher integration test.
- [ ] Add fresh registration and verify new/existing/invalid/resume behavior.
- [ ] Build for smoke testing, record evidence, close and publish.

## Log

### 2026-09-23

Read-only diagnosis plus isolated real-binary probe reproduced: existing Couch thread registration does not match requested established address. Live tag couch-d41b8dc708afd578 has a reserved claim and exited helper. Original setup error is separate.
