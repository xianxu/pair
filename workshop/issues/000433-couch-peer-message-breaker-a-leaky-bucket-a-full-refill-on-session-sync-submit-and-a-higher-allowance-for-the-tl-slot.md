---
id: 000433
status: open
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: 'f8a75f766cc2adad145c4631e9bb5f103f4d2255' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch peer-message breaker: a leaky bucket, a full refill on session-sync submit, and a higher allowance for the TL slot

## Problem

Couch gives each slot `InboundAllowance = 8` peer messages, refilled only by an operator submit in that slot's
agent pane (`cmd/internal/couchmessage/model.go`). It's a loop breaker: two agents can't ping-pong without a
human. But the TL slot (ops:0 in the ariadne-robustness-1 run) is the fleet's hub: every worker reports to it,
and the operator mostly reads there and submits through the session-sync status file (parley.nvim#313), which
arrives as a peer message and so spends allowance instead of refilling it. On 10-10 at 19:12, pair:1's landing
report for pair#324 was refused with `breaker-exhausted: recipient peer-message allowance exhausted`; the TL
learned of the landing two hours later (ops learnings kink 57).

## Spec

- **A leaky bucket instead of a count.** Burst of 8 as today; one message's worth refills every few minutes
  (pick the interval from the data: worker reports arrive minutes apart, a runaway loop seconds apart). A loop
  drains it in seconds and is cut off; spaced reports never exhaust it.
- **Session-sync submit refills fully.** A message whose sender is the recipient's own operator through
  session-sync (parley's Alt+Return submit) counts as `OperatorSubmit`, not as an admitted peer message.
- **The TL slot gets a much larger bucket.** Convention (operator, 10-10): the slot Couch was started from is
  the TL slot. Settle in the plan whether that's observable or needs an explicit `--hub repo:N` override.
- Refusals stay visible to the sender as today; `--list` or the receipt shows the recipient's remaining
  allowance, so a TL can see a hub running low.

## Done when

- [ ] A burst beyond the allowance is refused; after the refill interval, one more is admitted (model test).
- [ ] A session-sync submit refills the recipient's allowance fully (test).
- [ ] The TL slot admits well past the ordinary burst without an operator keystroke (test).
- [ ] A two-agent ping-pong is still cut off within seconds (test).

## Plan

- [ ]

## Log

### 2026-10-10
