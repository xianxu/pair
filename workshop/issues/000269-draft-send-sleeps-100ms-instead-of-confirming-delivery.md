---
id: 000269
status: open
deps: []
github_issue:
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
---

# Draft send sleeps 100ms instead of confirming delivery

## Problem

`nvim/draft_send.lua` waits ~100 ms after every successful body write before
sending the submit chord. The sleep exists because `zellij action write-chars`
returns once the bytes are *queued*, not once the wrapped harness has consumed
them: a short draft's submit arrived 18 ms after its body and overtook it, so
the text sat unsent in the Muse composer (#266).

The fix is right about the ordering and wrong about the mechanism. 100 ms is a
budget derived from one measurement, it is paid by every send whether or not
delivery was slow, and nothing reports the case it is meant to prevent — a
delivery that takes longer than the budget still loses the race, silently, and
looks exactly like the original bug. Raised as `#266` close BR-11
(`timing-heuristic-without-bound`).

`draft_send.lua` already carries a phase state machine (`start` → `written` →
`dispatched`, plus `indeterminate`), which is the shape a real confirmation
would fit into: the submit should be gated on evidence that the body reached the
harness, not on elapsed time.

## Spec

- Replace the fixed sleep with a delivery confirmation the draft can observe:
  the wrapper already sees the body arrive on stdin, so the signal exists on the
  pair-wrap side and needs a seam the draft can wait on (bounded, with a
  timeout that reports rather than hides).
- Keep the phase state machine as the authority for what may be retried; a
  confirmation timeout is an `indeterminate` outcome, not a silent success.
- Report, don't hide: a delivery that exceeds the bound must surface (draft
  notify + `pairlog`/telemetry) so the frequency is measurable instead of
  assumed.
- The 100 ms sleep is the fallback only while no confirmation is available, and
  must not remain the primary mechanism.
- `ARCH-CONSTRAINTS` (a timing budget needs a measured bound and a detector),
  `ARCH-ORDER` (ordering enforced by evidence, not by delay).

## Done when

- A draft send waits on delivery evidence, not a fixed sleep, and a fast send is
  no longer charged the full budget.
- A delivery slower than the bound produces a visible, logged outcome instead of
  an unsent draft.
- Tests cover: confirmed delivery, delayed-then-confirmed delivery, and a
  confirmation timeout leaving the transaction retryable.
- Measured before/after latency for a short draft recorded in the Log.

## Plan

- [ ] Find or add the wrapper-side signal that the body reached the harness, and
      the seam the draft waits on.
- [ ] Replace the sleep with the bounded wait; keep the sleep as an explicit
      fallback when no signal is available.
- [ ] Surface a timeout through the existing notify/log path.
- [ ] Tests for confirmed / delayed / timed-out delivery.
- [ ] Measure and record short-draft send latency before and after.

## Log

### 2026-09-16

- Filed from `#266`'s close boundary review (BR-11). The settle itself is the
  confirmed fix for the observed failure and stays until this replaces it; what
  is missing is the bound and the detector.
