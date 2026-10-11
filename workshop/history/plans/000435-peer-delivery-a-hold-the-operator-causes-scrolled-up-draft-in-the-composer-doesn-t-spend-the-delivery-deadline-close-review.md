# Boundary Review — pair#435 (whole-issue close)

| field | value |
|-------|-------|
| issue | 435 — Peer delivery: a hold the operator causes (scrolled up, draft in the composer) doesn't spend the delivery deadline |
| repo | pair |
| issue file | workshop/issues/000435-peer-delivery-a-hold-the-operator-causes-scrolled-up-draft-in-the-composer-doesn-t-spend-the-delivery-deadline.md |
| boundary | whole-issue close |
| milestone | — |
| window | c08a527d1db7b1a736e96c4888b1e9a9b5cd109c..499c7746b2a527b136a59954622ab2196cfd87e9 |
| command | sdlc close --issue 435 |
| reviewer | claude |
| timestamp | 2026-10-10T23:47:25-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Still reviewing — checking every place that branches on receipt status, so the new `held` value doesn't fall through any of them.

Running the targeted tests now; the one failure so far is a socket-bind permission error from the environment, not the change.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The change does what the issue asks for its main case. When the operator has the composer scrolled out of view, or a draft is sitting in it, the paste-by clock stops. One static cap (`MaxHold` = 30m) bounds the wait. Both the broker and the wrapper get their outer bound from `Message.Horizon()`, so they agree on it. The sender sees `held` plus the reason through a new interim-progress path.

One spec gap should be fixed before shipping: the clock still runs while the operator is actively typing a draft. During typing the wrapper reports `waiting for input to settle` instead of `waiting for empty composer`, and that branch closes the hold. The cap message, the non-operator expiry and the model's handling of late or terminal progress are all correct and tested.

**Test runs in this environment:**
- The four new wrapper tests pass, plus `TestPeerDraftHoldIsBounded`.
- The `couchmessage` tests that ran pass, including the model progress and broker status tests.
- I could not run the endpoint tests (`TestRemoteEndpoint*`), the negotiation tests or `TestPeerRuntimeBrokerToWrapperSubmitsOnce`. They need a unix socket or a temporary directory under `/tmp`, which the environment refused ("operation not permitted"). The failure is the environment, not the code.

### 1. Strengths
- **One bound, read everywhere (ARCH-DRY).** `Horizon()` = `HoldLimit + DeliveryTimeout` (`model.go:77`). The broker's expiry backstop, the job context and `RemoteEndpoint.Deliver` all get the longer bound from this one function, with no new protocol operation.
- **Clock logic is small and in one place.** `deadlineLocked`/`operatorHoldLocked` (`peer_delivery.go:550-579`). An open hold returns `HoldLimit`, a closed hold adds its time back, and the result is capped. The `defer` makes every exit from the dispatch path settle the hold.
- **The scrollback hold starts only on evidence.** `!d.view.at.IsZero() && !d.view.visible` (`peer_delivery.go:476`): the first viewport reading, still in flight, is not yet counted as the operator's hold.
- **Model guards against stale progress.** `DeliveryProgress` (`model.go:265`) ignores progress before the delivery starts or after the outcome, and rejects a terminal status sent as progress. All three cases are tested in `TestDeliveryProgressReportsHeld`.
- **Wrapper tests use the real composer harness and a controlled clock** (`peer_hold_test.go`), not mocks. The mutation checks in the Log match what these tests assert.

### 2. Critical findings
None.

### 3. Important findings
- **The clock runs while the operator types a draft (spec gap; ARCH-PURPOSE).** `peer_delivery.go:447-460`, plus the earlier waits at `:420-432` for forwarded, incomplete or buffered input.
  - Each keystroke sets `lastInput`. For one second after it, dispatch returns `waiting for input to settle` with `operatorHold=false`. That closes the hold and flips the receipt back from `Held` to `Queued`.
  - So someone typing a long reply with gaps under one second spends the full 90 s paste-by and the message expires. The Spec says operator-caused holds stop the clock, and typing is exactly the "draft in the composer" case.
  - Each flip also sends the sender's status between `held` and `delivering`.
  - Fix sketch: in the waits caused by operator input (input settle, held operator input, forwarded/incomplete/buffered input), keep an already-open hold open, e.g. `operatorHold = !d.heldSince.IsZero()`. Add a test that enqueues with a draft, feeds keystrokes under one second apart past the deadline, then empties the composer and expects a paste.

### 4. Minor findings
- **An expiry at the cap does not always name the hold.** If the hold closed earlier but the credit already reached `HoldLimit`, the expiry happens with `heldSince` zero, so the "held past 30m0s" prefix is missing (`peer_delivery.go:409`). Low frequency.
- **The 30-minute horizon also slows failure detection.** A recipient wrapper that dies or hangs mid-delivery now ties up the sender's pending slot until the broker's expiry check, which can be up to about 30.5 minutes away instead of about 2. `atlas/couch.md` does not mention this cost. It is acceptable if liveness and disconnect handling cancel sooner, but it is worth a sentence.
- **`endpoint.go:207` resets the poll ticker on every receipt.** This is harmless, but it only needs to happen when the status changes.

### 5. Test coverage notes
- Each Done-when item has a test:
  - draft hold: `TestPeerDraftHoldStopsTheDeadline`
  - scrollback hold: `TestPeerScrollbackHoldStopsTheDeadline`
  - non-operator hold: `TestPeerOtherHoldKeepsTheDeadline`
  - cap: `TestPeerHoldLimitExpiresNamingTheHold`
  - `held` shown to the sender: `TestBrokerStatusShowsRecipientHold` and `TestRemoteEndpointStatefulExchange`
- Missing: a test with operator keystrokes during a hold (the Important finding above), and one for an expiry at the cap after the hold has closed.

### 6. Architectural notes
- **ARCH-DRY: pass.** The single `Horizon`/`HoldLimit` source is used by all its readers.
- **ARCH-PURE: pass.** The deadline arithmetic is small and lives behind a lock, but the clock is injected and the tests drive it deterministically.
- **ARCH-PURPOSE: flag.** See the typing gap above.
- **ARCH-MOCK: pass.** The fake endpoint gained an interim-receipt channel, which keeps it stateful.
- **ARCH-CONSTRAINTS: pass.** Polling while held is paced at 1 s, and the 30m cap is stated and justified. See the minor note on dead-wrapper latency.
- **ARCH-SECURE: pass.** Interim statuses from the endpoint are checked against an allowlist (`endpoint.go:200`, `model.go:271`).
- **ARCH-ORDER: pass, with a note.** The hold is two fields (`heldSince`/`heldFor`) plus the receipt status, and only `operatorHoldLocked` changes them. That is readable, but the `Held`/`Queued` relationship is implicit. The fix for the Important finding should keep "a hold is open" as the single source of truth.
- **ARCH-FUNERAL: pass.** The new state is in memory only, reset on each enqueue, and nothing durable is created.

### 7. Plan revision recommendations
- If the typing fix lands, add a `## Revisions` entry: operator-input waits (input settle, held operator input) keep an open hold open.

```findings
findings:
  - id: new
    severity: Important
    family: operator-hold-classification
    title: |
      Operator keystroke waits (input settle / held / forwarded input) close the hold, so typing a draft spends the paste-by
    detail: |
      peer_delivery.go dispatch returns early with operatorHold=false on "waiting for input to settle" and the other input-caused waits, closing the hold and flipping Held back to Queued; continuous typing for 90s expires the message and makes the sender's status flicker. Keep an open hold open in those branches and add a keystroke-during-hold test.
  - id: new
    severity: Minor
    family: hold-expiry-naming
    title: |
      Expiry at HoldLimit via accumulated credit (hold already closed) omits the "held past" naming
  - id: new
    severity: Minor
    family: horizon-liveness-cost
    title: |
      Horizon grew by 30m, so a hung recipient keeps a sender pending far longer; atlas does not mention this cost
```

---

## Re-review — 2026-10-10T23:57:16-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 435 — Peer delivery: a hold the operator causes (scrolled up, draft in the composer) doesn't spend the delivery deadline |
| repo | pair |
| issue file | workshop/issues/000435-peer-delivery-a-hold-the-operator-causes-scrolled-up-draft-in-the-composer-doesn-t-spend-the-delivery-deadline.md |
| boundary | whole-issue close |
| milestone | — |
| window | c08a527d1db7b1a736e96c4888b1e9a9b5cd109c..a577977b8e3f5f46a004d1c78ac136afc2897c7d |
| command | sdlc close --issue 435 |
| reviewer | claude |
| timestamp | 2026-10-10T23:57:16-07:00 |
| verdict | SHIP |

## Review

Prior finding BR-1 looks fixed in code. Running the affected packages' tests now.

```verdict
verdict: SHIP
confidence: medium
```

**Verdict: SHIP.** All three prior findings are fixed, and each fix has evidence I could check. I found one new Minor issue.
- **BR-1 is fixed for the whole class, not just the reported case.** `dispatchPeer` now only opens or closes an operator hold after a full composer reading (`judged = true`, `cmd/internal/wrapcmd/peer_delivery.go`). That flag is set only after the input-settle wait and the held-input wait have returned. The earlier waits (automatic input, forwarded input, incomplete input, rendered output, buffered input, image, picker, terminal) all return before `judged` is set, so they leave an open hold as it is.
- **The new test would catch a regression.** `TestPeerTypingKeepsTheHoldOpen` expects `Held` with the reason "waiting for input to settle". If the code always judged, the settle return would close the hold and flip the status back to `Queued`, so the test would fail.
- **Test runs.** The six hold tests in `peer_hold_test.go` and the existing draft tests from #428 pass at `a577977b`. The endpoint tests and the `couchcmd` socket tests fail in this reviewer environment with `mkdir /tmp/...: operation not permitted` or a `bind` error. That is the environment, not the code, so I could not confirm them here.

**Strengths**
- **One limit, read the same way on both sides.** `Message.HoldLimit()` and `Horizon()` (`couchmessage/model.go`) are the single outer bound. The broker's Tick expiry, the job context and `RemoteEndpoint.Deliver` all read it, and no new protocol operation was needed (ARCH-DRY).
- **The deadline wins over an open hold.** `deadlineLocked` returns `HoldLimit` while a hold is open, and `Deadline + heldFor` (capped) otherwise. `Message.Deadline` is never changed, so `matchingReceipt` keeps working.
- **The broker's new event is tightly guarded.** `DeliveryProgress` in `Advance` is ignored unless the message is pending and in the `Delivering` phase. It rejects terminal statuses, and late progress after `DeliveryFinished` does nothing. `TestDeliveryProgressReportsHeld` checks all of these.
- **The progress callback is safe to call.** It takes `b.mu` itself, and `Deliver` runs after `b.mu.Unlock()`, so there is no deadlock.

**Critical:** none.

**Important:** none.

**Minor**
- **Version skew between Couch and wrappers.**
  - A new wrapper reporting `held` to an old `couch` daemon makes the old `RemoteEndpoint.Deliver` fail with "invalid delivery status". The sender then sees `indeterminate`, even though the wrapper may still paste later.
  - The old daemon also uses the 2-minute horizon.
  - This lasts only until Couch is restarted, and `indeterminate` is an honest answer. A one-line note in the atlas ("restart couch with the wrapper upgrade") would cover it.
- **A non-operator wait can keep an open hold open.** If the operator clears the draft and a non-judging wait follows (say a long rendered-output wait), the clock stays stopped. This is the stated rule and the cap still bounds it, so it is not a new finding in `operator-hold-classification`.

**Test coverage**
- **Wrapper:** the tests cover the draft hold, the scrollback hold, a hold the operator didn't cause (with credit added back), the cap while a hold is open, the cap after the hold has closed, and typing during a hold.
- **Model and broker:** the progress event and the `Held` status the sender sees.
- **Endpoint:** the interim `Held` receipts. I could not run these here because of the `/tmp` restriction.

**Architecture**
- **Passes:** ARCH-DRY, ARCH-PURPOSE, ARCH-MOCK, ARCH-SECURE.
  - ARCH-DRY: the single limit described above.
  - ARCH-PURPOSE: every Done-when item is delivered, and the sender can see the hold.
  - ARCH-MOCK: the stateful fake endpoint gained an interim channel behind the same seam.
  - ARCH-SECURE: the endpoint rejects unknown statuses and the model rejects terminal ones.
- **ARCH-CONSTRAINTS passes.** Held deliveries are polled every second, and the hang cost is written in the atlas.
- **ARCH-FUNERAL passes.** Nothing durable is created: two in-memory fields, reset on each enqueue.
- **ARCH-PURE and ARCH-ORDER pass, with one note.** The hold accounting (`heldSince`, `heldFor`, `judged`) is still a set of separate fields in the wrapper, not a state in `AdvancePeerDelivery`. They are few and their combinations are pinned by tests. A future change that adds more hold kinds should move this into the peer delivery model.

**Plan revisions:** none needed. The existing Revisions entry covers BR-1.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      judged is set only after the settle and held-input waits; every earlier wait returns before it. TestPeerTypingKeepsTheHoldOpen fails if the code always judges.
  - id: BR-2
    disposition: addressed
    note: |
      The "held past" naming now keys on reaching HoldLimit, not on an open hold; TestPeerHoldLimitAfterTheHoldClosed pins it.
  - id: BR-3
    disposition: addressed
    note: |
      atlas/couch.md now states that a hung recipient keeps the sender pending for about 32m instead of 2m.
findings:
  - id: new
    severity: Minor
    family: wrapper-couch-version-skew
    title: |
      A new wrapper's held receipt reads as an invalid status to an old couch daemon
    detail: |
      Until couch restarts, the old RemoteEndpoint.Deliver rejects held, so the sender gets indeterminate while the wrapper may still paste later; the old daemon's horizon is also 2m. Add an atlas note to restart couch with the upgrade.
```
