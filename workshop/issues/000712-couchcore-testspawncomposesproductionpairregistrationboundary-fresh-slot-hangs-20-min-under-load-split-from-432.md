---
id: 000712
status: open
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'd010d56175ec5a722390b398072d3ff910e92a48' # card fields mirrored from issue-cards; edit via sdlc
---

# couchcore: TestSpawnComposesProductionPairRegistrationBoundary/fresh-slot hangs 20 min under load (split from #432)

## Problem

`couchcore.TestSpawnComposesProductionPairRegistrationBoundary/fresh-slot` hung in pair#689's integrated suite (2026-10-19,
f2333e34): 1172.59 s, LaunchNative reported "terminal not listed live", the cleanup spawn goroutine failed to exit, then the
package hit its 20-minute timeout. The whole suite took 26 min (median 12.5). pair#432 records the same test "failing
identically on origin/main" (10-10) and asks to triage it separately; most pair suites since pass it, so it's intermittent,
probably load-sensitive. It now blocks a landing as an unattributable failure.

## Spec

- Split from #432 (its Spec asks for this). Reproduce under load (the suite's own concurrency, or `stress`), with the
  goroutine dump at the hang.
- Root-cause the hang: what waits forever when the terminal isn't listed live, and why the cleanup goroutine can't exit.
  A wait without a deadline in production code is a bug in its own right (the product side, not just the test).
- Register it as a known flake under this issue meanwhile (same-head fail→pass evidence from pair#689's run), so landings
  rerun it instead of blocking; the registration ends when this closes.

## Done when

- The hang is explained and fixed; the test passes 20/20 under a loaded run; no unbounded wait remains on that path.

## Plan

- [ ]

## Log

### 2026-10-19
