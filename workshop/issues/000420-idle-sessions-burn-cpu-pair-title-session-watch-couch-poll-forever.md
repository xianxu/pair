---
id: 000420
status: open
deps: []
github_issue:
created: 2026-10-09
updated: 2026-10-09
estimate_hours:
card_mirror: 'a0cb78acd0bd5916f51b34c7aa555ab04e2d52c5' # card fields mirrored from issue-cards; edit via sdlc
---

# idle sessions burn CPU: pair title / session-watch / couch poll forever

## Problem

Idle sessions burn CPU. Reported by ariadne:1 (TL) via Couch, 2026-10-09:

- 27 `pair title` and 5 `session-watch` processes poll forever.
- The top title process used 104 CPU minutes over 8 idle days.
- Each `session-watch` uses about 30 CPU minutes per day.
- The couch server sits at 20–40% CPU and peaks above 100%.
- Under that load, couch disconnected when Ghostty stalled for 5s.

Operator idea: start polling only after real user input, when a transcript is
expected, and back off when idle.

## Spec

## Done when

- An idle session (no operator input, no agent output) costs near-zero CPU:
  `pair title`, `session-watch` and the couch server stop polling or back off
  to a bounded slow cadence, measured over an idle window.
- Polling resumes promptly after real user input.
- Process count is bounded: no orphaned pollers outlive their session
  (ARCH-FUNERAL).
- Couch stays connected through a multi-second terminal stall.

## Plan

- [ ]

## Log

### 2026-10-09

- Filed only (not implemented) at ariadne:1's request.
