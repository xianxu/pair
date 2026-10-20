---
id: 000713
status: open
deps: [pair#712]
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
severity: low
collaboration-mode: collaboration
card_mirror: '59f87a113717beaaa66a3458e30f9d04116d5bba' # card fields mirrored from issue-cards; edit via sdlc
---

# Consolidate exact incarnation unknown transition

## Problem

#712 close review BR-3 (Minor, single-incarnation-transition-authority) found that Couch.persistUnconfirmedStart repeats the exact process-identity match, live-to-unknown mutation and revision-conflict loop in ThreadStore.MarkIncarnationUnknown. Both currently preserve ownership correctly, but two authorities can drift.

## Spec

Consolidate the exact incarnation transition through a shared pure mutation helper or contextual store operation. Preserve both caller contracts: failed cleanup accepts an already-unknown exact incarnation without rewriting it, while existing strict callers retain their current refusal behavior. Contextual cleanup keeps one persistence context across routing, read, CAS retries and journal locking. A changed PID/birth identity never authorizes mutation or cleanup. No format, state, recovery action or deadline changes.

## Done when

- One implementation owns exact live-to-unknown incarnation matching and mutation.
- Tests cover strict versus idempotent already-unknown behavior, changed birth identity refusal, unrelated metadata/revision conflicts and contextual store-lock contention.
- #712's retained-claim, bounded-persistence, failure-history and receipt regressions remain passing.

## Plan

- [ ]

## Log

2026-10-19: filed from #712 round-2 SHIP advisory BR-3. Deferred consolidation preserves the reviewed code and queued authoritative suite; this is not a blocking correctness defect in #712. Work starts from main after #712 lands.
