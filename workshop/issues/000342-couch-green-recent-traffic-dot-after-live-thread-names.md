---
id: 000342
status: open
deps: []
github_issue:
created: 2026-09-28
updated: 2026-09-28
estimate_hours:
card_mirror: '3ef5b7b565fe9955be6c436fa8c979acb57f7d3e' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch: green recent-traffic dot after live thread names

## Problem

The operator wants to see at a glance which Couch threads are producing output
right now: a green period after the thread name, in the status bar and the
switcher. Split out of #247 on 2026-09-28. #247 fades threads by idle age,
measured in hours from file mtimes; this dot is live, in-memory terminal
traffic over a 15 s window, a different signal and a separate review.

## Spec

Operator-approved on 2026-09-14 (originally recorded in #247's Revisions):

Approved dot behavior:

- Display `ariadne.` with only the final period green when at least 15 bytes of fresh child PTY output arrived within the preceding 15 seconds. Spinners, redraws and control traffic count; no screen comparison or agent-work inference.
- Apply the same recent-traffic predicate to thread names in the status bar and switcher, including background threads. The dot is presentation only, not part of the thread's name or identity.
- Remove the dot when the rolling-window threshold is no longer met or the attachment ends. Replay and Couch's own paints do not create new activity. A new attachment starts with no observed activity; keep this transient state in memory.
- Preserve existing selection and notification cues. Use existing console scheduling for expiry; no per-thread timer, per-byte persistence, or new background process. Bound storage and per-batch work by the small threshold.
- Call the internal signal recent terminal activity. It is evidence of output, not proof of process health or useful progress. In color-disabled rendering the period still indicates activity without escape sequences.

Coordinate with #247: both styles decorate the same labels in both renderers.
Keep the dot's styling and #247's idle fade independent, so either can land first.

## Done when

- An injected clock verifies the 15-byte/15-second boundaries, expiry without further output, fresh versus replayed bytes, background activity and attachment reset. Production-path tests reach both renderers and preserve clipping/click spans and existing emphasis.

## Plan

- [ ]

## Log

### 2026-09-28

- Split out of #247 at the operator's choice. The spec above is #247's approved
  2026-09-14 revision, unchanged.
