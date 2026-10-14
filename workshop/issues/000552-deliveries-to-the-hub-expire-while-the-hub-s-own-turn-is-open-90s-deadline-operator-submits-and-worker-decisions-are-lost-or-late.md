---
id: 000552
status: open
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: '785eb10589981fe0cbf6baa4b0f3cc0268733f64' # card fields mirrored from issue-cards; edit via sdlc
---

# Deliveries to the hub expire while the hub's own turn is open (90s deadline): operator submits and worker decisions are lost or late

## Problem

2026-10-14: deliveries into a slot's agent pane can wedge in "waiting for previous automatic input to clear" with an
empty composer and no turn open, until each message's 90s deadline expires. Seen twice:
- **parley.nvim:1** from ~07:40 (after the TL drove `/status` into its pane with raw zellij keys): every Couch
  delivery expired, and `couch --relaunch` refused "busy" (composer holds text / input seen) though the composer was
  empty.
- **The hub, ops:0**: the operator's session-sync submits "did nothing", worker decision questions to the hub expired
  (ariadne:1's #240, twice), and `--actors` showed ops:0 `pending` with allowance 64/64 (not the leaky bucket).
  The TL was **not** mid-turn. The operator typed into the hub's draft pane and three queued messages arrived right
  after: a genuine operator input reset the wrapper's "previous automatic input" state.

So the wrapper's automatic-input gate can stay set after the input it waits on is gone, and only an operator
keystroke clears it.

## Spec

Find what sets the "previous automatic input" gate and why it doesn't clear (input injected outside the wrapper, a
delivery whose echo or settle signal was missed, a picker or overlay drawn and dismissed). The gate must clear from
observed composer state (empty composer, no turn) within a bounded time, not only on operator input. Hub-bound
messages and the operator's own submits must never expire because of it. `--message-status` should name the gate's
cause, and `--relaunch` must not report "composer holds text" when the composer is empty.

## Done when

- A reproduction (raw key injection, then a normal delivery) wedges on main and delivers after the fix (test).
- With the composer empty and no turn, a pending delivery goes through within the bound, with no operator keystroke.

## Plan

- [ ]

## Log

### 2026-10-14
