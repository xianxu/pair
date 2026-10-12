---
id: 000489
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '86dcc3e03f89078bb7760f78c41cc52e093c810f' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: relaunch/resume fails under load (pane not born in 15 s) and leaves the thread parked; make startup resilient

## Problem

10-12 08:3x, load average ~200: `couch --relaunch` failed for ariadne:0, brain:0, parley.nvim:1, parley.nvim:2 and pair:0; each is left parked (conversation kept). A later `couch --resume pair:0` failed the same way: `await Pair registration …: agent pane not born — context deadline exceeded [pane-birth wait consumed the budget: pane birth 15.002s …] (NO Pair session is live. Pair never started, or exited before registering)`. Relaunch parks first (the old agent is stopped on purpose) and then cold-starts a new wrapper on the current binary; with PAIR_DEV set it also rebuilds pair. On a loaded machine the new wrapper (and Claude Code) can take longer than a fixed 15 s to register, so the operation fails and the slot is left without a live agent. The operator: startup should be resilient; Couch's own share of the work should be small, so why a timeout even on a hot machine?

## Spec

Investigate, then fix at the root:
1. **Where the time goes** on a loaded machine: the pair rebuild (PAIR_DEV), process spawn, the agent's own startup, registration. Measure each step (the error already has a step breakdown; extend it).
2. **Whether waiting longer works:** does the wrapper eventually register after the 15 s budget? If yes, the budget is the bug: wait on the registration event with a generous hang ceiling and progress reporting, the same rule as the test sweeps (#480): no fixed short window for work whose duration depends on load.
3. **What a failure leaves behind:** if the new wrapper starts late, Couch must adopt it rather than leave the thread parked with a wrapper running unclaimed (or kill it cleanly). Never a state the operator has to puzzle out.
4. **The rebuild:** relaunching many slots shouldn't rebuild pair once per slot; build once per sweep (or not at all when the binary is current).
5. A relaunch sweep could check load and pace itself (one at a time, waiting for each), instead of the TL guessing.

## Done when

- The cause of each 08:3x failure is explained in the Log with measurements.
- Under a synthetic load (e.g. CPU burners matching ~200 load), relaunch and resume succeed, possibly slowly, with progress shown; a genuinely dead launch still fails clearly and leaves a clean state.
- Tests for the adopt-late and clean-failure paths.

## Plan

- [ ]

## Log

### 2026-10-12
