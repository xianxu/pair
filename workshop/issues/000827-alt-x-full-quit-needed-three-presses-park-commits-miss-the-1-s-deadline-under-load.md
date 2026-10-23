---
id: 000827
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'ec110b131e3848acfa0c2967c1d00281d41f56c3' # card fields mirrored from issue-cards; edit via sdlc
---

# Alt+x full quit needed three presses: park commits miss the 1 s deadline under load

## Problem

## Spec

Operator report (10-22 full shutdown): **Alt+x (park everything and quit Couch) timed out on some slots; it took
three Alt+x presses to shut everything down.** The machine was loaded (load average 40+, 30-odd slots).

Lead (TL reading, unverified): `couchcore/park.go` sets `parkCommitSoftTarget = 100ms` and `parkCommitDeadline =
1s` for each slot's park commit; under load some slots likely miss the 1 s commit and are left unparked, so the
operator has to repeat Alt+x.

Fix: measure park commit times across a full fleet under realistic load (pair#738-style counted timing), derive the
deadline from the measurement with a re-measure note (ARCH-CONSTRAINTS), and make one Alt+x finish the job:
slots that miss the first deadline get retried within one bounded overall budget, and anything still unparked is
named in a single summary (with its recovery) instead of silently needing another press.


**Operator decision (10-22, night):** "there's no reason to set this this tight. I think 5 to 10 seconds is perfectly
fine." Set the per-slot park commit deadline in that range (measurement still recorded to confirm it). Current
parallelism, for context: parks fan out up to `LifecycleParallelism` = half the CPU cores (operator decision 10-07),
so slots park in parallel batches, not all at once nor one by one.

## Done when

- One Alt+x parks a fleet of 30 fake slots where some commits take 2–3 s (test), or names exactly the ones it
  couldn't park.
- The per-slot deadline is derived from measured park commit times, with the measurement recorded.

## Plan

- [ ]

## Log

### 2026-10-22
