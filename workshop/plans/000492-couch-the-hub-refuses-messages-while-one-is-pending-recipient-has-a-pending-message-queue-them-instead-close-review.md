# Boundary Review — pair#492 (whole-issue close)

| field | value |
|-------|-------|
| issue | 492 — couch: the hub refuses messages while one is pending ('recipient has a pending message'); queue them instead |
| repo | pair |
| issue file | workshop/issues/000492-couch-the-hub-refuses-messages-while-one-is-pending-recipient-has-a-pending-message-queue-them-instead.md |
| boundary | whole-issue close |
| milestone | — |
| window | a7d9779b301b3c98b6ee269c9327f81d515a6330..e776e654c75b5463d648b8ff4dc56223a48bb35d |
| command | sdlc close --issue 492 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T12:29:47-07:00 |
| verdict | REWORK |

## Review

Reviewing the broker's queue/promote logic; I suspect a queued message can strand when a concurrent reservation attempt fails. Running tests and a quick repro.

```verdict
verdict: REWORK
confidence: high
```

The queue design is sound, but there is one confirmed bug that breaks the issue's main promise, so this needs rework. The pure model handles admit, promote, cancel and expiry in one place. The wrapper's busy error now reaches the broker. The tests cover the 10-12 case and the hold-while-busy case. The bug is in the broker: a queued message can get stuck forever. If a send to a recipient finds a reservation attempt already in flight, it is queued and the worker is woken. The worker sees the reservation, returns `false` (no retry), and goes idle. If that in-flight attempt then fails with anything other than busy (wrapper unreachable, a family route's `ErrNoRecipient`, a cancelled context), the reservation is cleared but nothing wakes the worker. The queued message stays `queued, position 1` and is never delivered. It never expires either, because `QueueWait` expiry only runs inside `promote`. I reproduced this with a temporary test against the pinned head (output: `b stranded: queued position 1`) and then deleted the test. The "all are delivered in order once it settles" done-when item fails under this interleaving. `go test ./cmd/internal/couchmessage` passes. I could not run `./cmd/internal/couchcmd` here: setup fails because the embedded asset `assets/runtime/files` has not been built, so the CLI-side tests are unverified.

1. **Strengths**
   - All queue state changes go through `Advance` (`model.go`: `dequeue`/`positions`/`leave`). Ordering, expiry, allowance and position re-reporting live in one pure place, so the plan's ARCH-DRY rationale holds.
   - `endpoint.go:122` maps the wrapper's busy string to `ErrRecipientBusy`. Without this the hold-not-refuse path could never trigger in production. Good catch, and it has a test.
   - Pulling `deliver` out of `runActor` (`broker.go:820`) keeps the worker loop readable and moves the delivery code over unchanged.
   - The TL's revision is delivered: `leave()` ends queued messages with an actionable `Cancelled` detail ("disconnected (relaunched or exited) before delivery; resend"), and `TestQueuedMessagesEndWithActionableReceipts` asserts it.
   - Queue bounds are explicit and justified: `MaxQueued`=32, and `QueueWait`=45 min, which is longer than hold plus delivery and shorter than `ReceiptLifetime`.

2. **Critical**
   - **Queued message strands after a concurrent attempt fails** (`broker.go:640`, `promote` early return; `broker.go:532-560`, `admit` failure path). The sequence:
     1. Sender A's exact or family `tryAdmission` holds `a.reservation`.
     2. Sender B is enqueued and wakes the worker.
     3. `promote` sees `a.reservation != ""` and returns `false`.
     4. A fails with a non-busy error and clears the reservation without waking the worker.
     5. B is never promoted and never ticked, and its non-terminal receipt never leaves `b.receipts`.

     **Fix:**
     - In `admit`, after clearing `a.reservation`, send on `a.wake` (non-blocking) when `a.state.Queued() > 0`.
     - Or have `promote` return `true` (retry) when a non-empty queue is blocked only by someone else's reservation, pending delivery or inbox.

     Add a regression test that gates `Observe` on the first attempt and fails it. (ARCH-ORDER)

3. **Important**
   - **The tests see only one interleaving.** The broker queue tests (`broker_queue_test.go`) run sends one after another. None makes an admission and a promotion race, so the bug above passed CI. The state that matters (`pending` / `reservation` / `inbox` / `queue` / worker idle with no retry) is spread across several flags whose valid combinations are not written down anywhere. "Queue non-empty, everything else empty, worker idle, no timer" is reachable and wrong. Add a test where an in-flight attempt is held open while another send arrives, then failed. (ARCH-ORDER)

4. **Minor**
   - **Order can invert:** if A's in-flight exact attempt fails busy because B was queued meanwhile, A is enqueued behind B, even though A was sent first (`broker.go:545`).
   - **`AlreadyCommittedError` path in `promote` is untested** (`broker.go:658-665`). It cancels, deletes the receipt and re-records it.
   - **Unhelpful detail:** the `Cancel` applied in the `AlreadyCommitted` path has an empty `Detail`. The receipt is overwritten straight away, but if `recordRecoveredLocked` finds an old receipt, the result has no explanation.

5. **Test coverage**
   - The model queue tests (`queue_test.go`) pin admission order, promotion with a fresh paste-by, expiry, a full queue, cancellation and re-admit deduplication without any IO. That is genuinely pure.
   - The broker tests use stateful fakes and the real worker.
   - Missing:
     - concurrent admission vs. promotion
     - the `AlreadyCommitted` path during promotion
     - a family admission in flight while an exact send queues

6. **Architecture**
   - **ARCH-DRY: pass.** The `deliver` extraction is clean. The `emitFor` / `dequeue` helpers are shared.
   - **ARCH-PURE: pass.** The queue logic is pure and the broker only schedules.
   - **ARCH-PURPOSE: pass.** Hub-specific special-casing was avoided, and every exact recipient queues.
   - **ARCH-MOCK: pass.** The fakes behind `DeliveryEndpoint` are stateful (reservation, sequence).
   - **ARCH-CONSTRAINTS: pass.** The queue length and wait are bounded, and the retry interval is a 1 s real timer.
   - **ARCH-SECURE: pass.** No new untrusted parsing. The wrapper's error string is matched exactly against the sentinel.
   - **ARCH-ORDER: flag.** See both findings above. The broker's worker liveness depends on who signals `wake`, and that is never enumerated.
   - **ARCH-FUNERAL: flag (part of the Critical bug).** Queued receipts are non-terminal, so `sweep` never removes them, and their only removal path (Tick in `promote`) depends on the worker running. A stuck entry also uses up `MaxReceipts` capacity until the recipient disconnects.

7. **Plan revisions**
   - Add a `## Revisions` entry recording that every path that clears `a.reservation` while the queue is non-empty must re-arm the worker (wake or retry), so the worker can never be idle while the queue is non-empty.

```findings
findings:
  - id: new
    severity: Critical
    family: worker-wake-liveness
    title: |
      Queued message strands forever when a concurrent admission attempt fails without waking the worker
    detail: |
      promote returns false (no retry) when a.reservation is held by another attempt; admit's failure path clears the reservation without signalling a.wake, so the queue head is never promoted nor Tick-expired. Reproduced at head with a gated Observe failure: "b stranded: queued position 1". Fix: wake when clearing a reservation with Queued()>0, or retry in promote when blocked by another reservation.
  - id: new
    severity: Important
    family: single-interleaving-tests
    title: |
      Broker queue tests observe only sequential sends; no admission/promotion race is exercised
    detail: |
      broker_queue_test.go serializes every send, so the pending/reservation/inbox/queue flag constellation's idle-with-nonempty-queue state is never reached. Add a gated-endpoint test holding an attempt in flight while another sender queues, then failing it (ARCH-ORDER).
  - id: new
    severity: Minor
    family: queue-fifo-order
    title: |
      An in-flight exact attempt that fails busy is enqueued behind later senders, inverting send order
  - id: new
    severity: Minor
    family: untested-recovery-path
    title: |
      promote's AlreadyCommittedError adoption path has no test
```

---

## Re-review — 2026-10-12T12:54:50-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 492 — couch: the hub refuses messages while one is pending ('recipient has a pending message'); queue them instead |
| repo | pair |
| issue file | workshop/issues/000492-couch-the-hub-refuses-messages-while-one-is-pending-recipient-has-a-pending-message-queue-them-instead.md |
| boundary | whole-issue close |
| milestone | — |
| window | a7d9779b301b3c98b6ee269c9327f81d515a6330..446554ee1d5d21b933bb674022651c71d5febbb4 |
| command | sdlc close --issue 492 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T12:54:50-07:00 |
| verdict | REWORK |

## Review

Checking prior findings now: running the test suite and reverting the BR-1 fix to confirm its regression test goes red.

Still verifying BR-1's regression test (the reverted run waits out its deadline). I also spotted a possible hot retry loop in `promote`, which I'll confirm next.

BR-1's regression test fails without the fix, as it should. Now checking how often `promote` retries a busy wrapper.

```verdict
verdict: REWORK
confidence: high
```

**Verdict: REWORK.** Most of the design holds up. The pure model owns the queue, and exact routes queue while family routes still skip busy slots. The tests cover the 10-12 case, holding behind a busy wrapper, and the actionable cancel the TL asked for. BR-1 is fixed at its root, and its regression test fails without the fix. But that fix (446554ee) causes a new Critical bug. In `promote`, a failed attempt now signals `a.wake` itself, so the 1 s `PromoteRetry` timer is never reached and the worker retries without pause. I measured **1,830,116 wrapper refusals in 2 s** against `busyEndpoint`. In production each retry is an Observe + Reserve + Release round trip to the wrapper. That runs whenever the wrapper refuses or can't be reached and messages are waiting, for up to `QueueWait` (45 min). The existing test doesn't catch it because it only checks "≥2 refusals", with no upper bound.

### Strengths
- **Pure queue model:** `Advance` handles Admit/Enqueue, Promote, Cancel and Tick expiry in one place (`model.go:330-385`). `dequeue` and `positions` re-report each message's position after any removal. `queue_test.go` tests all of this without IO.
- **Root-cause fix for BR-1:** every reservation release now goes through one helper, `releaseReservationLocked` (`broker.go:637`).
- **The BR-2 test doesn't depend on timing:** a gated `Observe` plus the `blockedPromotions` counter forces the bad interleaving every run.
- **Wrapper busy error is mapped correctly:** the wrapper's busy refusal is now mapped back to `ErrRecipientBusy` when it crosses the socket (`endpoint.go:122`). Without this the queue-on-busy fallback could never trigger in production.
- **The TL's revision is delivered and tested:** a queued message whose recipient leaves ends `cancelled` with "… before delivery; resend".

### Critical
- **`broker.go:679` / `broker.go:640-643` / `runActor` (ARCH-CONSTRAINTS, ARCH-ORDER; 2nd finding in family `worker-wake-liveness`):**
  - **What happens:** on promote's failure path, `releaseReservationLocked(head.ID)` sees `Queued()>0` and fills `a.wake`. `promote` then returns true and the loop arms `retry`, but the next `select` takes `wake` immediately. The result is a hot loop of wrapper RPCs.
  - **Rule to fix (not just this instance):** a wake is owed only when someone *other than the worker* makes the actor promotable again: an admission attempt released, a message queued, or a recipient-side commit adopted. The worker's own failed attempt must never wake itself; only the `PromoteRetry` timer paces it.
  - **Fix sketch:** let `promote` release with a non-waking variant on its own paths. Alternatively, drain `a.wake` before returning true. Keep the explicit `wakeLocked()` on the AlreadyCommitted path, because that one is real progress.
  - **Test:** bound the retry rate. Against an always-busy wrapper, refusals over about 2.5 s must be ≤ about 4. That fails at HEAD.

### Important
- **`broker_test.go:59` (ARCH-MOCK):** when the reservation is taken or the sequence moved, `FakeDeliveryEndpoint.Reserve` returns `errors.New("changed or reserved")`. The real wrapper (`wrapcmd/peer_delivery.go:354`) returns `ErrRecipientBusy`, which is exactly what the new queue-on-busy branch checks. Tests on the standard fake therefore never reach that branch, and only the ad hoc `busyEndpoint` does. Fix: make the fake return `ErrRecipientBusy`, matching the wrapper.

### Minor
- `blockedPromotions` is a field used only by tests on the production `SlotActor` (`broker.go:54`). It's acceptable as an observation seam, but an injected hook would keep it out of the production struct.
- Without the BR-1 fix, its regression test only goes red when the whole-package deadline expires (589 s). A bounded wait would give faster feedback.
- Queue expiry runs only inside `promote`. While a held delivery occupies the worker (up to about 32 min), queued receipts can read `queued` somewhat past `QueueWait` until the next promote.
- Admit's dedupe now compares IDs only (`model.go:335`). It used to compare the full message, so a same-ID, different-body readmission is now a silent no-op. Upstream receipt checks probably cover this; worth confirming.

### Test coverage notes
- `go test -race ./cmd/internal/couchmessage/` passes at HEAD.
- I couldn't run `couchcmd` and `wrapcmd` in this checkout: `runtimebundle/embed.go` needs generated assets ("no matching files found"). That's an environment limit, not a code failure. The CLI and wait-notice changes were checked by reading only.
- BR-1: I removed the wake from `releaseReservationLocked`, and `TestMessageQueuedBehindAFailingAttemptIsDelivered` failed with "nothing delivered". I then restored the code; the checkout is clean.
- Missing: an upper bound on the retry rate, which is the gap the Critical finding slipped through.

### Architecture
| Marker | Result |
|---|---|
| ARCH-DRY | Pass. `tryAdmission` and `promote` share the Observe → Reserve → reconcile shape; worth a helper later. |
| ARCH-PURE | Pass. |
| ARCH-PURPOSE | Pass. The Done-when items are delivered; the live hub check is still the operator's step after a restart. |
| ARCH-MOCK | **Flag** (Important above). |
| ARCH-CONSTRAINTS | **Flag** (Critical: unpaced retries). |
| ARCH-SECURE | Pass. The queue is bounded at 32 and bodies are validated. |
| ARCH-ORDER | **Flag.** The self-wake is an unwritten (state, event) transition. The pending / reservation / inbox / queue flags still form an implicit state set. |
| ARCH-FUNERAL | Pass. The queue is bounded by `MaxQueued` and `QueueWait`, and receipts by `ReceiptLifetime`. |

### Architectural notes
The worker's loop has three wake sources (inbox, wake, retry). Their priority should be written down. A small "worker-step" decision function, returning promote-now, retry-at-time or idle, would make pacing testable without real timers.

### Plan revision recommendations
- Add a "## Revisions" entry: worker pacing. Wakes come only from other actors' releases or admissions; retries come only from the `PromoteRetry` timer. A test bounds the retry rate.
- Add a "## Revisions" entry: `FakeDeliveryEndpoint` returns `ErrRecipientBusy`, matching the wrapper.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      releaseReservationLocked wakes on release; reverting the wake makes TestMessageQueuedBehindAFailingAttemptIsDelivered fail (nothing delivered). The fix itself introduced a new self-wake spin, raised separately.
  - id: BR-2
    disposition: addressed
    note: |
      TestMessageQueuedBehindAFailingAttemptIsDelivered holds an attempt in flight via gated Observe and waits on blockedPromotions; verified red without the fix.
  - id: BR-3
    disposition: withdrawn
    note: |
      Accepted design with stated rationale in the issue Log (order is admission-completion order, as for concurrent sends before the change).
  - id: BR-4
    disposition: addressed
    note: |
      TestPromotionAdoptsAnAlreadyCommittedMessage drives promote's AlreadyCommittedError branch via the fake's retained receipt and asserts the queue moves on.
findings:
  - id: new
    severity: Critical
    family: worker-wake-liveness
    title: |
      promote's own failed attempt self-wakes the worker, bypassing PromoteRetry and hot-looping wrapper RPCs
    detail: |
      2nd finding in family worker-wake-liveness. On promote's failure path releaseReservationLocked(head.ID) fills a.wake (queue non-empty), so runActor re-promotes immediately instead of after PromoteRetry. Measured 1,830,116 busyEndpoint refusals in 2s. Rule: a wake is owed only when someone other than the worker makes the actor promotable; the worker's own failure is paced by the retry timer alone. Fix: non-waking release (or drain a.wake) on promote's own paths, keep the explicit wake on AlreadyCommitted adoption, and add a test bounding refusals per interval against an always-busy wrapper.
  - id: new
    severity: Important
    family: fake-contract-drift
    title: |
      FakeDeliveryEndpoint.Reserve returns a plain error where the real wrapper returns ErrRecipientBusy
    detail: |
      broker_test.go:59 returns errors.New("changed or reserved"); wrapcmd/peer_delivery.go:354 returns couchmessage.ErrRecipientBusy, which the new admit queue-on-busy branch keys on. Tests on the standard fake cannot reach that branch (ARCH-MOCK); make the fake return ErrRecipientBusy.
  - id: new
    severity: Minor
    family: untested-recovery-path
    title: |
      BR-1 regression test only goes red at the package deadline (about 590s)
    detail: |
      2nd finding in family untested-recovery-path. Rule: recovery-path tests should fail within a bounded wait, not the global go test timeout. Use a short bounded wait for the delivery after the gate opens.
  - id: new
    severity: Minor
    family: queue-expiry-lazy
    title: |
      Queued-message expiry runs only inside promote, so receipts can read queued past QueueWait while the worker is in a long held delivery
```

---

## Re-review — 2026-10-12T13:04:54-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 492 — couch: the hub refuses messages while one is pending ('recipient has a pending message'); queue them instead |
| repo | pair |
| issue file | workshop/issues/000492-couch-the-hub-refuses-messages-while-one-is-pending-recipient-has-a-pending-message-queue-them-instead.md |
| boundary | whole-issue close |
| milestone | — |
| window | a7d9779b301b3c98b6ee269c9327f81d515a6330..8ec985d3f259f0ca2f2bf7d75ba6648bff4f180d |
| command | sdlc close --issue 492 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T13:04:54-07:00 |
| verdict | SHIP |

## Review

The couchmessage tests pass. couchcmd/wrapcmd need generated runtime assets, which I'm looking for now; after that I'll revert each fix to confirm its test goes red.

All three packages pass. Now reverting each claimed fix one at a time to confirm its test fails without it.

BR-5 and BR-8 both fail when reverted, so their tests are real. Reverting BR-6 (the fake's busy error) breaks no test, so I'm checking whether anything pins the fake to the real wrapper's contract.

BR-7 is unchanged: `delivered()` still waits on the package deadline. Checking the issue log for how the implementor disposed BR-6–BR-8.

The new Status tick also runs the model's pending-delivery horizon check, not just queue expiry. Checking whether a held delivery could get finished early by a status read.

```verdict
verdict: SHIP
confidence: high
```

The round-2 fixes hold. **BR-5:** I put promote's old waking release back and `TestPromotionHoldsWhileWrapperBusy` failed with "2161 reservation attempts". With the fix in place it passes, because the worker now frees its own attempt without waking itself (`broker.go:686-688`). The AlreadyCommitted adoption path still wakes the worker on purpose (`broker.go:696`). That is safe because each adoption removes the queue head, so the wakes are bounded. **BR-8:** I removed the new Status expiry block and `TestQueuedMessageReadsExpiredWhileTheWorkerIsBusy` failed, reading `queued` past QueueWait. **BR-6:** the fake now matches `peerDelivery.reserve`. **BR-7** was kept deliberately, and the reason given is consistent with the repo's convention. The couchmessage, couchcmd and wrapcmd packages all pass, and the queue tests pass under `-race -count=3`. couchcmd and wrapcmd only ran after I generated the runtime assets, which this checkout doesn't include. One new Minor finding, which doesn't block: Send's same-ID replay path can still return a stale `queued` receipt.

1. **Strengths**
   - `broker.go:639-651` and `broker.go:684-688` now state who wakes the worker: every release except the worker's own wakes it, and the worker paces its own retries with PromoteRetry. That rule resolves the `worker-wake-liveness` family as a whole, not just the site that was named.
   - The BR-5 test bounds refusals per interval (`broker_queue_test.go:114-122`). It fails right away without the fix instead of hanging.
   - `TestMessageQueuedBehindAFailingAttemptIsDelivered` waits on `blockedPromotions` instead of sleeping, so it reliably hits the blocked-then-released ordering.
   - All queue state changes go through the pure `Advance` model (Tick, Enqueue, Promote and Cancel in `model.go:307-391`).

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `broker.go:456-462`: when a sender retries the same ID, Send returns the stored receipt without the expiry check that Status now applies. A queued message past QueueWait can still come back as `queued` on a resend (family `queue-expiry-lazy`, see below).

5. **Test coverage notes**
   - When I put the plain error back in the fake (BR-6), no test failed. The busy-to-enqueue branch in admit is covered through `busyEndpoint` (`broker_queue_test.go:83`) and `wait_notice_test.go:189`, not through the standard fake. The fake's own busy path (already reserved, or a sequence mismatch) is not exercised.
   - A queued message expiring through Status can collide with promote while promote is between locks. The Promote event is then rejected, and the next message waits one PromoteRetry (1s) before promotion. That outcome is harmless and untested.

6. **Architectural notes**
   - **ARCH-DRY:** pass. promote clears its reservation inline instead of calling the waking helper. That is intentional and commented.
   - **ARCH-PURE:** pass. The model is pure and the broker is the IO shell.
   - **ARCH-PURPOSE:** pass. Exact sends to a busy recipient are queued, never refused.
   - **ARCH-MOCK:** pass, with a note. The fake and `peerDelivery.reserve` now agree, but nothing enforces it. A shared conformance test run against both would stop the `fake-contract-drift` family from recurring.
   - **ARCH-CONSTRAINTS:** pass. Retries are paced by PromoteRetry and the test bounds them; the queue is capped by MaxQueued.
   - **ARCH-SECURE:** pass. The only external input is the wrapper's AlreadyCommitted receipt, and `adoptRetained` checks its sender and body before trusting it.
   - **ARCH-ORDER:** pass. Ordering-sensitive tests use gated seams. The wake rule is enforced only by comment and convention. A future reservation-release path must pick the right helper.
   - **ARCH-FUNERAL:** pass. The queue is bounded by MaxQueued and QueueWait, and receipts by MaxReceipts and RetainUntil.

7. **Plan revisions:** none needed.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      Reverting broker.go:686-688 to releaseReservationLocked makes TestPromotionHoldsWhileWrapperBusy fail with 2161 refusals; it passes with the fix. AlreadyCommitted keeps its explicit wake (bounded: each adoption dequeues the head).
  - id: BR-6
    disposition: addressed
    note: |
      broker_test.go:59-61 now returns ErrRecipientBusy, as wrapcmd/peer_delivery.go:354 does. The admit busy-enqueue branch is covered by busyEndpoint and wait_notice_test; no fake/wrapper conformance test exists yet.
  - id: BR-7
    disposition: withdrawn
    note: |
      The implementor kept the deadline-bounded wait, citing the repo convention against guessed short timeouts (lessons 479). The test still fails on regression, just at the deadline; acceptable.
  - id: BR-8
    disposition: addressed
    note: |
      Status (broker.go:345-350) now ticks a queued receipt past its Deadline. Disabling it makes TestQueuedMessageReadsExpiredWhileTheWorkerIsBusy fail with status queued.
findings:
  - id: new
    severity: Minor
    family: queue-expiry-lazy
    title: |
      Send's same-ID replay (broker.go:456) returns a stale queued receipt past QueueWait
    detail: |
      This is the 2nd finding in family queue-expiry-lazy. Rule: every path that returns a stored receipt goes through one current-receipt helper that applies any due expiry. The enumeration is Status (fixed), StatusContext (delegates to Status), and Send's idempotent replay at broker.go:456-462 (not fixed). Extract the Status block into that helper and call it from both places.
```
