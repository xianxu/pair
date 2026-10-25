---
id: 000885
status: open
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: '250b8fcf50876212a4fabe30fcd400a4ff6667ef' # card fields mirrored from issue-cards; edit via sdlc
---

# Sweep 183 wall-clock-bound test sites for ariadne's test-tight-wallclock lint, then enable it

## Problem

## Spec

**ariadne's new `test-tight-wallclock` lint flags 183 pair test sites (ariadne#622, 10-25).** pair runs ariadne's golint
catalog in `internal/repolint` (`pairCatalogFor`). ariadne#622 adds a rule against second-scale wall-clock bounds in
correctness tests (polled deadlines and elapsed-time assertions below the hang ceiling); it flags 183 pair sites, mostly
in webcast and wrapcmd. Same root cause as the flakes that blocked ariadne's landings on 10-24/25 (ariadne#620, #625):
under fleet load, a wall-clock bound fails while the code is correct.

Until this lands, whoever bumps pair's ariadne pin past #622 excludes `test-tight-wallclock` in `pairCatalogFor` (one
line, noted in the bump commit) so pair doesn't go red. This issue removes that exclusion.

Sweep with ariadne#622's four patterns: (A) polled deadlines move to the hang ceiling, the polled event stays the
oracle, fail fast where a worker exits early; (B) elapsed bounds proving a kill or cancel become outcome assertions
with a fake that outlives the ceiling; (C) documented contract bounds get a reviewed `//lint:allow` naming the
contract, or an injected clock; (D) absence-by-observation windows become an event, else an allow with the reason.
Milestones by package group so each review stays readable.

## Done when

- `test-tight-wallclock` is enabled in `pairCatalogFor` and pair's tree passes it.
- Each changed package proven with targeted `-count=10` (one run at a time, per the machine budget).

## Plan

- [ ]

## Log

### 2026-10-25
