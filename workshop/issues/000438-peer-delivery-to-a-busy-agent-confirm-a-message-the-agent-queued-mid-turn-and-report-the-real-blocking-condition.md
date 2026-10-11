---
id: 000438
status: open
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: 'ff8f8a1671f85f5da9e4277d4648d7aaecabc1b8' # card fields mirrored from issue-cards; edit via sdlc
---

# Peer delivery to a busy agent: confirm a message the agent queued mid-turn, and report the real blocking condition

## Problem

On 10-10 (23:44–23:48) the ops TL sent three messages to parley.nvim:1 while it was four minutes into a turn with its
spinner streaming (ops learnings kink 63):
- 034cf323 **expired before its paste** (90 s paste-by) and never arrived.
- de593285 **was delivered** (Claude Code queued it behind the running turn) but its receipt read `indeterminate`:
  no new turn opened within the 30 s confirmation window, so the wrapper had no evidence of delivery.
- The TL's resend, 42e4e03e, was then a **duplicate** the worker saw twice, and it spent another unit of the
  worker's 8-message allowance.

All three receipts said "waiting for rendered output". In `dispatchPeer` (`wrapcmd/peer_delivery.go`) the checks
run in order and the receipt keeps the last one that failed. `outputPending > 0` comes before the composer checks,
and a streaming agent nearly always has output in flight, so the detail hides what actually blocked the paste.

## Spec

- **Confirm a queued message.** When the agent accepts input mid-turn and shows it as queued (Claude Code: the
  queued-message line above the composer, "Press up to edit queued messages"), that is delivery evidence: the
  receipt reads `submitted` (or a distinct `queued-by-agent`), not `indeterminate`. Agent-specific recognition
  lives with the existing composer recognizers; agents without it keep today's behavior.
- **Report the real blocker.** The receipt's detail names the condition that held the delivery for most of the
  window (or every condition seen), not the last check evaluated. A streaming agent's transient output-in-flight
  doesn't mask a composer that isn't ready.
- **Busy-agent policy.** Decide in the plan between pasting mid-turn (today) and holding until the agent's turn
  ends (`LastActivity` quiet for `QuietInterval`), with the deadline counted per pair#435's rule. Whichever is
  chosen, a delivery to an agent that streams for minutes must not expire silently.
- Overlap: pair#435 (pair:3) changes how holds spend the deadline; pair#433 (pair:2) the allowance. Coordinate
  through the TL.

## Done when

- [ ] A message queued by a busy agent reads `submitted` (or `queued-by-agent`), never `indeterminate` (wrapper test with a recorded Claude Code queued-message screen).
- [ ] With continuous output and a not-ready composer, the receipt names the composer condition (test).
- [ ] A delivery to an agent streaming for 3+ minutes either lands or reports a hold, per the chosen policy (test).

## Plan

- [ ]

## Log

### 2026-10-10
