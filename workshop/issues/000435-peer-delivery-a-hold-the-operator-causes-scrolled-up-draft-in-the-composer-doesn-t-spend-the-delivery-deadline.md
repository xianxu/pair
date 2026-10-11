---
id: 000435
status: codecomplete
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: '5e2ea80a0c1e9a28fa98d8337ee44560e487b751' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-10T23:14:33-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 0.63
---

# Peer delivery: a hold the operator causes (scrolled up, draft in the composer) doesn't spend the delivery deadline

## Problem

A peer message must start its paste by `Message.Deadline` (`cmd/internal/couchmessage/model.go`). pair#428 made the
recipient's wrapper hold a delivery while the operator has the agent pane scrolled up ("waiting for the composer to
be in view", `wrapcmd/peer_delivery.go`), and it already holds while the composer has a draft in it ("waiting for
empty composer"). Both holds are caused by the operator, and both spend the deadline. On 10-10 (pair#428 live check
on ariadne:2) a test message expired after about 90 s while held for scrollback. The TL slot is where the operator
reads scrollback the longest, so worker reports to the TL are what expire, and a sender learns of it only by
polling the receipt (ops learnings kinks 56, 58). It hurts the TL's effectiveness (operator, 10-10).

## Spec

- **Operator-caused holds stop the clock.** While the wrapper holds only because the composer is out of view or
  holds a draft, the deadline doesn't advance; it resumes when the hold clears. Holds the operator didn't cause
  (paste mode not ready, unrecognized composer, a booting slot) keep today's deadline.
- **A cap still bounds the receipt.** A held message stays pending at most `ReceiptLifetime` (or a stated cap), so
  a pane left scrolled up overnight doesn't keep a sender's message forever; on reaching the cap the status names
  the hold ("expired: held for scrollback").
- **The sender can see the hold.** The receipt reports `held` with the reason, so a TL polling a send sees "held
  for scrollback" rather than "delivering".
- The broker and the wrapper agree on the extended horizon (today `Horizon()` = deadline + `DeliveryTimeout`).

## Done when

- [x] A message held for scrollback past its original deadline submits when the composer comes back into view (model and wrapper tests).
- [x] The same for a held draft once the composer empties.
- [x] A hold not caused by the operator still expires at the deadline (test).
- [x] The cap expires a long hold with a status that names it; the receipt shows `held` and the reason while held (test).

## Plan

Design (settled; ARCH-SIMPLE — one static cap both sides read, no new protocol op):

- **Cap:** `couchmessage.MaxHold = 30m`; `Message.HoldLimit() = Deadline + MaxHold`; `Horizon()` becomes
  `HoldLimit + DeliveryTimeout`, so the broker's Tick backstop, the job context and `RemoteEndpoint.Deliver`'s
  context all cover the longest hold without the broker learning each hold. 30 min, not `ReceiptLifetime`: the
  recipient takes one message at a time, so the cap is also how long a scrolled-up pane can refuse other
  senders (`recipient-busy`, an immediate, retryable answer).
- **Wrapper clock** (`wrapcmd/peer_delivery.go`): `dispatchPeer` flags an operator hold on the two details
  ("waiting for empty composer"; "waiting for the composer to be in view" once a reading said out of view, not
  while the first reading is in flight). `operatorHoldLocked` opens/closes it (`heldSince`, `heldFor`);
  `deadlineLocked` before the paste is `HoldLimit` while a hold is open, else `Deadline + heldFor` capped at
  `HoldLimit`. Every other hold keeps the clock running. Reaching the cap expires with
  "delivery deadline elapsed: held past 30m0s, <hold detail>".
- **`held` status:** new non-terminal `couchmessage.Held`, the wrapper receipt's status while a hold is open.
  `DeliveryEndpoint.Deliver` gains a `progress func(Receipt)`; `RemoteEndpoint.Deliver` hands it each changed
  interim receipt (and polls at 1 s while held instead of 50 ms); the broker applies a `DeliveryProgress`
  actor event, so `couch` status shows `held` + reason (Queued interim → `delivering` + its detail).
  Sender disconnect while held → `cancelled` (held = not pasted), as for any non-`delivering` receipt.
- Lifecycle: creates nothing durable — two in-memory fields per delivery, reset by the next enqueue.

- [x] Model: `MaxHold`, `HoldLimit`, `Horizon`, `Held`, `DeliveryProgress` (+ model test)
- [x] Endpoint/broker: `Deliver` progress, Held accepted, held poll pacing (+ endpoint + broker tests)
- [x] Wrapper: hold accounting + Held receipt (`peer_hold_test.go`: draft, scrollback, non-operator, cap)
- [x] Full `make test` (see close evidence), close

## Log

### 2026-10-10
- 2026-10-10: closed — Tests: model TestDeliveryProgressReportsHeld; broker TestBrokerStatusShowsRecipientHold; endpoint TestRemoteEndpointStatefulExchange (held interim -> progress); wrapper peer_hold_test.go (draft, scrollback, non-operator keeps deadline incl. credited time, cap naming the hold incl. after it closed, typing keeps the hold open [BR-1]). Mutation-checked each. couchmessage + wrapcmd green after the BR-1 fix. Full suite before: make -k test (5-var scrub) failed only test-changelog + test-pair-embedded-runtime, both green rerun (scratchpad TMPDIR; env -i). go test ./... (env -i): artifactpath TestProductionArtifactReferencesAreExactlyClassified pre-existing on main (no new production files); couchcore + gcruntime hit the 10m timeout under suite load and do not depend on couchmessage/wrapcmd (go list -deps).; review verdict: SHIP
- 2026-10-10: flow upgraded quick → full — 111 added lines in code files (limit 100); an earlier round of this close already ran the full review

- Coordination: pair#433 (pair:2) also edits `couchmessage/model.go` (allowance). This change touches the const
  block, `Message`/`Horizon`, `Status`, `EventKind` and adds one `Advance` case; no allowance code.
- Mutation-checked: returning `Deadline` while held fails the draft/scrollback/non-operator tests; dropping
  the `heldFor` credit fails the non-operator test.
- Close review round 1 (FIX-THEN-SHIP): BR-1 Important — the waits a keystroke causes (input settle, forwarded,
  incomplete or held input) closed the hold, so typing a draft spent the paste-by. Fixed the class: only a full
  composer reading (`judged`) opens or closes an operator hold; every wait that can't judge it leaves it as is.
  `TestPeerTypingKeepsTheHoldOpen` (mutation-checked: always judging fails it). Minors: the cap expiry names the
  hold whenever `HoldLimit` is reached, also after the hold closed (`TestPeerHoldLimitAfterTheHoldClosed`); the
  endpoint resets its poll ticker only on a status change; atlas states the longer-horizon cost for a hung
  recipient.

## Revisions

- 2026-10-10 (close review BR-1): waits that can't judge the hold (input settling, forwarded/held input) keep an
  open operator hold open; only the composer reading decides it.
