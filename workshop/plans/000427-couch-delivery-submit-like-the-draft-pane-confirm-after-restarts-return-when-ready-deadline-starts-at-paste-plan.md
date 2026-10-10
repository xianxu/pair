# #427 plan: submit like the draft pane, confirm after; restarts return when ready; deadline starts at paste

Issue: `workshop/issues/000427-…md` (Spec + Done when are the contract).
Principle: **pair renders, never classifies.** No pre-submit match on what
the agent rendered.

## Current shape (read 2026-10-10)

- Wrapper reducer `couchmessage/delivery.go` (`AdvancePeerDelivery`): Waiting →
  Pasting → Rendering → (needs `Matches`) Submitting → Submitted on the submit
  write. `Matches` comes from `wrapcmd/peer_composer.go:peerComposerMatches`
  (exact, wordwrap projection, Claude collapsed-marker). Nothing retries.
- One `Message.Deadline` = admission + 30s (`broker.go:564`). It bounds the queue
  wait, the paste, the submit, the broker model's `Tick` expiry, the endpoint's
  commit/receipt contexts and the broker job context.
- `RestartConversation` returns when a new session token appears; `Relaunch`
  returns when `ResumeContext` returns. Neither waits for #421's Settled.
- Wrapper `settledNow` ignores a pending startup orientation.

## Design

### D1. Delivery reducer: paste → fixed delay → submit → confirm (Spec 1, 2)

`AdvancePeerDelivery` phases become Waiting → Pasting → Pasted → Submitting →
Confirming → Submitted. Changes:

- `PeerRenderObserved{Ready}` in Pasted submits. The wrapper sets Ready when
  `PeerSubmitDelay` (150ms, the draft pane's 100ms plus margin) has passed since
  the paste completed. `Matches` is deleted from the event.
- `PeerSubmitCompleted` (full write) → Confirming, no publish yet.
- New `PeerConfirmObserved{Ready}` in Confirming → Submitted. The wrapper sets
  Ready on agent-agnostic evidence: the composer was seen **occupied** after the
  paste and now reads **empty**, or a turn **opened** (`turnActive` false at
  submit, true now). The wrapper's own submit publishes no turn observation, so
  the second piece of evidence comes from the agent.
- A deadline, operator input, an image or an overlay after the body was written
  ends **Indeterminate**, not Expired/Cancelled. The reason starts `uncertain:`
  and the wrapper appends its evidence (composer state, turn state) in
  `Receipt.Detail`, through the existing reason + detail join. `indeterminate`
  is the receipt status that already means "outcome uncertain". A new enum
  value would ripple through the broker model and CLI for nothing (ARCH-DRY).
- Delete `peerComposerMatches`, `peerSpaceWordwrap`, `peerClaudeCollapsedPaste`,
  `peerClaudeCollapsedMarker` and `peerWordwrapSafe`, with their tests (a
  removal sweeps its identifier: `git grep` after). `peerComposerText` and
  `peerComposerState` stay; the pre-paste gate and the settle check use them.
- The poll ticker runs at 100ms while a delivery is past the paste. It stays
  at 1s while queued, so the delay and the confirmation don't wait a whole
  second.

### D2. Deadline starts at paste (Spec 4)

- `Message.Deadline` becomes the **paste-by** time: admission + `PasteTimeout`
  (90s). A booting slot uses this budget. The wrapper refuses to paste after
  it, and the existing effect-time check before `PeerPaste` stays.
- After the paste, the wrapper bounds submit + confirm by
  `pastedAt + DeliveryTimeout` (30s, unchanged constant, new start point).
  `dispatchPeer` and the effect-time check before `PeerSubmit` read this
  `deliverBy`, not `Message.Deadline`.
- `Message.Horizon() = Deadline + DeliveryTimeout` is the one outer bound the
  broker side reads (ARCH-DRY: one definition): the model's `Tick` expiry and
  late `DeliveryFinished`, `runActor`'s job context (`Horizon + ReceiptTimeout`)
  and `RemoteEndpoint.Deliver`'s receipt context. The commit RPC context stays
  `Deadline`, because a commit after paste-by is pointless.
- Mixed versions: a new Couch with an old wrapper gets a 90s all-in window,
  which the broker allows. An old Couch with a new wrapper treats a late
  submit as Indeterminate. Both are safe.

### D3. Restarts return when ready (Spec 3)

- Wrapper: `settledNow` also requires the startup orientation to be finalized
  (`orientation == nil` or `finalized` closed). Finalizing orientation re-arms
  the settle check through a new `peerDelivery.settleSourceChanged()`. The
  lessons rule applies: every writer of a derived state notifies it. This
  matters because orientation can finalize silently, at its deadline.
- Couch: `liveRestartProbe.AwaitReady(ctx, scope, tag, before SessionToken)`
  polls `LivenessForThread` until a session other than `before` reports
  `Settled == true`. It is bounded by `readyWithin` (2 min) and returns
  `*couchcore.RestartUnready{Detail}` on expiry. If the new session has no
  Build (a pre-hello-v2 wrapper, Settled never comes), it returns at once
  with a warning.
- Placement: in `consoleSlotOperations`, not on the console queue (a 2-minute
  wait must not block other slot operations). The prepare closure records the
  thread address from the call args and `before` = the current session token,
  for `relaunch`/`reload-context` only. On a successful finish, a goroutine
  awaits readiness, then calls the real `finished`. On expiry `finished` gets
  the value plus `RestartUnready`. `slotOperationOutcome` maps that to
  `failed`, code `unready`, with detail "restart took; not settled within 2m;
  do not retry, peek the slot". The goroutine is bounded by `readyWithin`;
  it creates nothing durable (ARCH-FUNERAL: an in-memory wait that dies with
  its timeout).
- CLI `slotOperationPollBudget` goes from 3 to 5 min, to cover a restart plus
  the readiness wait.
- The console's own Alt+n is unchanged. It is not routed through
  `slotOperations`, and the operator is watching.

## Tests (TDD; colocated)

- `couchmessage/delivery_test.go`: a table over phases × events. It covers:
  submit without any render match; Confirming → Submitted only on confirm; a
  deadline, operator input or overlay after the paste or submit gives
  Indeterminate with an `uncertain:` reason; a deadline before the paste gives
  Expired.
- `couchmessage` model: `Tick` expires at `Horizon`, not `Deadline`; a
  `DeliveryFinished` Submitted between the two is accepted.
- `wrapcmd/peer_delivery_test.go`:
  - The paste renders as arbitrary text (the boot case: full text, not the
    envelope), then a submit after the delay, then the composer clears →
    Submitted.
  - **Simulated boot:** the composer comes up only after the original 30s.
    Delivery pastes (paste-by not reached), submits and confirms when the
    turn opens.
  - A paste never consumed (the composer keeps the text, no turn) gives
    Indeterminate, with evidence in the detail.
  - A multi-line envelope (#418) submits.
  - The deadline is measured from the paste.
- `wrapcmd/peer_settle` tests: orientation pending → not settled; finalize
  re-arms the check.
- `couchcmd`: `AwaitReady` (new session settled → nil; old session settled
  → keeps waiting; timeout → RestartUnready; no Build → immediate warning).
  A `consoleSlotOperations`/`slotOperations` test: a relaunch or reload
  receipt stays `running` until the session settles, then reaches
  `succeeded`. Unready gives `failed`/`unready`.
- Live: `TestPeerLiveConformance -peer-live-submit -peer-live-use-local-auth`
  with the short and collapsed bodies, against real Claude. Then, after
  landing, `couch --reload-context pair:N --confirm` followed by a send to
  that slot (needs Couch on the new binary; ops/operator).

## Atlas

`atlas/couch.md`: the delivery contract (pre-paste gates, paste → delay →
submit → confirm, `uncertain` receipts, paste-by + post-paste window) and
restart readiness (relaunch/reload-context return when settled, the `unready`
code).

## Out of scope

Readiness/quiet checks on exact-route sends: the paste-by budget covers a
booting slot. The console Alt+n readiness.
