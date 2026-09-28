---
id: 000269
status: open
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
github_issue:
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
