---
id: 000315
status: working
deps: []
github_issue:
created: 2026-09-23
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T20:26:43-07:00
flow: {kind: quick, provenance: inferred, spec: "4203fd59", done: "cc5e4c9e"}
---

# Allow fresh slot launches to establish their reserved conversation

## Problem

pair:2 fresh-slot times out because Pair rejects the newly reserved conversation ID as not already established.

## Spec

Fresh Couch registration accepts an exact owned reserved or established claim, establishing only reserved claims. Resume keeps its existing read-only established-only validation. Missing, malformed and mismatched claims are refused without adoption. Route the fresh launcher through this explicit operation and test the real filesystem boundary behind its runtime seam (ARCH-PURPOSE, ARCH-MOCK).

## Done when

- New fresh-slot reservations reach the launcher handoff and become established.
- Established-address fresh launches still work; resume and checkpoint replacement reject reserved claims without mutation. Missing, malformed, mismatched and unowned fresh reservations never launch or establish.
- Regression tests exercise production claim storage through the launcher.

## Plan

- [x] Reproduce reserved fresh failure in a launcher integration test.
- [x] Add fresh registration and verify new/existing/invalid/resume behavior.
- [x] Build for smoke testing and record verification evidence.

## Log

### 2026-09-23

Read-only diagnosis plus isolated real-binary probe reproduced: existing Couch thread registration does not match requested established address. Live tag couch-d41b8dc708afd578 has a reserved claim and exited helper. Original setup error is separate.

## Revisions

### 2026-09-23 — preserve checkpoint replacement policy

The fresh-registration operation is only for exact Couch-owned fresh launches without a checkpoint. Existing checkpoint continuation retains its established-only registration contract. This avoids broadening continuation while fixing new slot launch.

### 2026-09-23 — implementation evidence

New real-filesystem launcher regression failed on the reserved case before implementation and now passes. Established fresh, missing/malformed/wrong-identity claims, unowned reservations and resume refusal are covered. Existing checkpoint reserved rejection remains passing. Launcher suite passed (13.448s); affected race tests passed (1.650s). make pair bin/couch passed. Isolated real-binary probe at pair:2 progressed from claim rejection to the fake Zellij handoff, and the exact reservation became established; no real agent was started. Live metadata is at pair-slot2/.couch, not pair-slot2/pair/.couch. Operator invited to retry fresh-slot using rebuilt binary.

### 2026-09-23 — review window provenance

The branch carries prior Add slot work (#313). Bootstrap/CI gateway changes in the broad boundary window came from `aff72f82` (build: adopt ariadne#239 seeded gateway files), already an ancestor of origin/main; this issue does not change those files against origin/main. Review #315 on its launcher registration change and real-claim regression. Preserve unrelated upstream work.
