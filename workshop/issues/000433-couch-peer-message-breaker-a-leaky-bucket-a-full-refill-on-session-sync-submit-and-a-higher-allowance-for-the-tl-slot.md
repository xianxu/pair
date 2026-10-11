---
id: 000433
status: working
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: 'ac9056eec2d330fdd44b9b4b19c564a9fe2a607a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-10T23:14:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
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

Quick flow (one close). Decisions:
- **Bucket in the pure model.** `ActorState` keeps `remaining`, `capacity` and `refilledAt`; every event first
  credits `floor(elapsed / RefillInterval)` (capped at capacity; the clock restarts when full). `RemainingAt(now)`
  is the same arithmetic for listings, which have no event. `RefillInterval = 3m`: a ping-pong delivers about one
  message per QuietInterval (30 s) per side, so an 8-burst still empties in minutes, while worker reports minutes
  apart never drain it.
- **Session-sync submit = a self-addressed message.** Parley shells `couch --send-to <owner>` from the operator's
  editor, which inherits the slot's launch env, so the broker resolves the caller to the recipient's own binding
  (`From == To`). The model treats a self-addressed admission as `OperatorSubmit`: full refill, nothing spent.
  Residual: an agent messaging itself also refills. That is a one-agent continuation, not a cross-slot loop; the
  two-agent ping-pong this breaker exists for is unaffected.
- **TL slot is observable, no flag.** Couch resolves its startup cwd to a worktree root; the actor whose workspace
  `WorktreeRoot` equals it registers with `HubAllowance = 64` (`Register` carries the allowance). A ping-pong with
  the hub is still cut by the worker's 8. Couch started outside a checkout has no hub.
- **Visible.** `couch --list` prints `allowance=N/CAP` from `Candidate.Capacity`.

- [ ] model: bucket + self-send refill + register allowance; tests for each Done-when row
- [ ] broker/service: register with an allowance, hub root on `messageAuthority`, listing shows remaining/capacity
- [ ] atlas note; targeted tests, then full `make test`

## Log

### 2026-10-10
