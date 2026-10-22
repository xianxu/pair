---
id: 000800
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
severity: low
collaboration-mode: collaboration
card_mirror: '8b8beb6778a32e25e56d42df9a79f673c8eb1c79' # card fields mirrored from issue-cards; edit via sdlc
---

# Bind runtime jobs to invocations for proven idle attribution

## Problem

ariadne#547 found that a retained landing ticket proves a command was blocked, but cannot prove its transcript source was idle. pair#778's retained car waiter was counted as focused work while it polled another conductor's train; its historical adopted actual must not be rewritten from guesses. Ticket start/bind/end facts will expose counted unverified coverage in ariadne#547, without subtracting time.

Claude tool IDs and Codex call IDs pair invocations/results, while queue/ticket records carry operation ID and PID incarnation. There is no structural binding between these identities and the harness runtime job. An outer Codex exec call can contain arbitrary mixed work; parsing shell/JavaScript command text or guessing from tool output cannot establish pure observation.

## Spec

Low-priority discovery/design: determine the correct owner and minimum producer-owned structural runtime-job/invocation binding needed to distinguish pure observations of a retained waiter from substantive same-source work. Keep substantive activity counted. Cross-agent controls must include Claude and Codex, wrapper mixing, yielded jobs, cancellation, process reuse and lost/unknown outcomes. This is a future prerequisite for real source-idle subtraction, not authorization to change actuals now. Avoid per-poll durable history and command-text heuristics.

## Done when

- Identify an authoritative runtime seam that binds job/invocation to the exact ticket/process incarnation, or document a concrete limitation and owning dependency.
- Define how pure observation is proved independently of command text and how mixed/unsupported calls remain counted and unverified.
- Design regression controls for same-source substantive work, poll-only waits, process reuse, unknown/cancelled jobs and Claude/Codex wrappers before any subtraction change.
- Any implementation or actual correction is separately approved; no automatic rewrite of historical measurements.

## Plan

- [ ]

## Log

Filed unclaimed at low priority by ariadne:4 under TL ops:0's explicit direction on 2026-10-22. Context: ariadne#547 visibility revision; independent code inspection found no existing runtime binding in native transcript provenance, landingline tickets or queue runtime identity. No Pair code or live runtime was changed.
