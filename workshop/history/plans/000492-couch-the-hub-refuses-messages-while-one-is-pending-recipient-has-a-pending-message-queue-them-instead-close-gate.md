---
gate: boundary-review
issue: 492
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T12:29:47-07:00"
      dispatched: "2026-10-12T12:27:50-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: Queued message strands forever when a concurrent admission attempt fails without waking the worker
          detail: 'promote returns false (no retry) when a.reservation is held by another attempt; admit''s failure path clears the reservation without signalling a.wake, so the queue head is never promoted nor Tick-expired. Reproduced at head with a gated Observe failure: "b stranded: queued position 1". Fix: wake when clearing a reservation with Queued()>0, or retry in promote when blocked by another reservation.'
          family: worker-wake-liveness
          round: 1
        - id: BR-2
          severity: Important
          title: Broker queue tests observe only sequential sends; no admission/promotion race is exercised
          detail: broker_queue_test.go serializes every send, so the pending/reservation/inbox/queue flag constellation's idle-with-nonempty-queue state is never reached. Add a gated-endpoint test holding an attempt in flight while another sender queues, then failing it (ARCH-ORDER).
          family: single-interleaving-tests
          round: 1
        - id: BR-3
          severity: Minor
          title: An in-flight exact attempt that fails busy is enqueued behind later senders, inverting send order
          family: queue-fifo-order
          round: 1
        - id: BR-4
          severity: Minor
          title: promote's AlreadyCommittedError adoption path has no test
          family: untested-recovery-path
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T12:54:50-07:00"
      dispatched: "2026-10-12T12:43:04-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: releaseReservationLocked wakes on release; reverting the wake makes TestMessageQueuedBehindAFailingAttemptIsDelivered fail (nothing delivered). The fix itself introduced a new self-wake spin, raised separately.
          round: 2
        - id: BR-2
          disposition: addressed
          note: TestMessageQueuedBehindAFailingAttemptIsDelivered holds an attempt in flight via gated Observe and waits on blockedPromotions; verified red without the fix.
          round: 2
        - id: BR-3
          disposition: withdrawn
          note: Accepted design with stated rationale in the issue Log (order is admission-completion order, as for concurrent sends before the change).
          round: 2
        - id: BR-4
          disposition: addressed
          note: TestPromotionAdoptsAnAlreadyCommittedMessage drives promote's AlreadyCommittedError branch via the fake's retained receipt and asserts the queue moves on.
          round: 2
      findings:
        - id: BR-5
          severity: Critical
          title: promote's own failed attempt self-wakes the worker, bypassing PromoteRetry and hot-looping wrapper RPCs
          detail: '2nd finding in family worker-wake-liveness. On promote''s failure path releaseReservationLocked(head.ID) fills a.wake (queue non-empty), so runActor re-promotes immediately instead of after PromoteRetry. Measured 1,830,116 busyEndpoint refusals in 2s. Rule: a wake is owed only when someone other than the worker makes the actor promotable; the worker''s own failure is paced by the retry timer alone. Fix: non-waking release (or drain a.wake) on promote''s own paths, keep the explicit wake on AlreadyCommitted adoption, and add a test bounding refusals per interval against an always-busy wrapper.'
          family: worker-wake-liveness
          round: 2
        - id: BR-6
          severity: Important
          title: FakeDeliveryEndpoint.Reserve returns a plain error where the real wrapper returns ErrRecipientBusy
          detail: broker_test.go:59 returns errors.New("changed or reserved"); wrapcmd/peer_delivery.go:354 returns couchmessage.ErrRecipientBusy, which the new admit queue-on-busy branch keys on. Tests on the standard fake cannot reach that branch (ARCH-MOCK); make the fake return ErrRecipientBusy.
          family: fake-contract-drift
          round: 2
        - id: BR-7
          severity: Minor
          title: BR-1 regression test only goes red at the package deadline (about 590s)
          detail: '2nd finding in family untested-recovery-path. Rule: recovery-path tests should fail within a bounded wait, not the global go test timeout. Use a short bounded wait for the delivery after the gate opens.'
          family: untested-recovery-path
          round: 2
        - id: BR-8
          severity: Minor
          title: Queued-message expiry runs only inside promote, so receipts can read queued past QueueWait while the worker is in a long held delivery
          family: queue-expiry-lazy
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-12T13:04:54-07:00"
      dispatched: "2026-10-12T12:58:37-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: addressed
          note: 'Reverting broker.go:686-688 to releaseReservationLocked makes TestPromotionHoldsWhileWrapperBusy fail with 2161 refusals; it passes with the fix. AlreadyCommitted keeps its explicit wake (bounded: each adoption dequeues the head).'
          round: 3
        - id: BR-6
          disposition: addressed
          note: broker_test.go:59-61 now returns ErrRecipientBusy, as wrapcmd/peer_delivery.go:354 does. The admit busy-enqueue branch is covered by busyEndpoint and wait_notice_test; no fake/wrapper conformance test exists yet.
          round: 3
        - id: BR-7
          disposition: withdrawn
          note: The implementor kept the deadline-bounded wait, citing the repo convention against guessed short timeouts (lessons 479). The test still fails on regression, just at the deadline; acceptable.
          round: 3
        - id: BR-8
          disposition: addressed
          note: Status (broker.go:345-350) now ticks a queued receipt past its Deadline. Disabling it makes TestQueuedMessageReadsExpiredWhileTheWorkerIsBusy fail with status queued.
          round: 3
      findings:
        - id: BR-9
          severity: Minor
          title: Send's same-ID replay (broker.go:456) returns a stale queued receipt past QueueWait
          detail: 'This is the 2nd finding in family queue-expiry-lazy. Rule: every path that returns a stored receipt goes through one current-receipt helper that applies any due expiry. The enumeration is Status (fixed), StatusContext (delegates to Status), and Send''s idempotent replay at broker.go:456-462 (not fixed). Extract the Status block into that helper and call it from both places.'
          family: queue-expiry-lazy
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: 8ec985d3f259f0ca2f2bf7d75ba6648bff4f180d
      blocked: false
    - "n": 4
      timestamp: "2026-10-12T13:07:15-07:00"
      dispatched: "2026-10-12T13:06:22-07:00"
      agent: claude
      dispose:
        - id: BR-9
          disposition: addressed
          note: receiptLocked (broker.go:350) now backs Status, Send replay (:462) and recordRecoveredLocked (:742); test at broker_queue_test.go:286 fails (Status queued) when the Send call is reverted to the raw map read.
          round: 4
      recipe: fix-review
      isolation: fenced
      reviewed: c3e9e8158ff93055e72c940aa8a3a959e513f79b
      blocked: false
---

# Gate ledger — pair#492 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T12:29:47-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `worker-wake-liveness` Queued message strands forever when a concurrent admission attempt fails without waking the worker
  promote returns false (no retry) when a.reservation is held by another attempt; admit's failure path clears the reservation without signalling a.wake, so the queue head is never promoted nor Tick-expired. Reproduced at head with a gated Observe failure: "b stranded: queued position 1". Fix: wake when clearing a reservation with Queued()>0, or retry in promote when blocked by another reservation.
- **BR-2** [Important] `single-interleaving-tests` Broker queue tests observe only sequential sends; no admission/promotion race is exercised
  broker_queue_test.go serializes every send, so the pending/reservation/inbox/queue flag constellation's idle-with-nonempty-queue state is never reached. Add a gated-endpoint test holding an attempt in flight while another sender queues, then failing it (ARCH-ORDER).
- **BR-3** [Minor] `queue-fifo-order` An in-flight exact attempt that fails busy is enqueued behind later senders, inverting send order
- **BR-4** [Minor] `untested-recovery-path` promote's AlreadyCommittedError adoption path has no test

## Round 2 — 2026-10-12T12:54:50-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — releaseReservationLocked wakes on release; reverting the wake makes TestMessageQueuedBehindAFailingAttemptIsDelivered fail (nothing delivered). The fix itself introduced a new self-wake spin, raised separately.
- BR-2 — addressed — TestMessageQueuedBehindAFailingAttemptIsDelivered holds an attempt in flight via gated Observe and waits on blockedPromotions; verified red without the fix.
- BR-3 — withdrawn — Accepted design with stated rationale in the issue Log (order is admission-completion order, as for concurrent sends before the change).
- BR-4 — addressed — TestPromotionAdoptsAnAlreadyCommittedMessage drives promote's AlreadyCommittedError branch via the fake's retained receipt and asserts the queue moves on.

### Raised

- **BR-5** [Critical] `worker-wake-liveness` promote's own failed attempt self-wakes the worker, bypassing PromoteRetry and hot-looping wrapper RPCs
  2nd finding in family worker-wake-liveness. On promote's failure path releaseReservationLocked(head.ID) fills a.wake (queue non-empty), so runActor re-promotes immediately instead of after PromoteRetry. Measured 1,830,116 busyEndpoint refusals in 2s. Rule: a wake is owed only when someone other than the worker makes the actor promotable; the worker's own failure is paced by the retry timer alone. Fix: non-waking release (or drain a.wake) on promote's own paths, keep the explicit wake on AlreadyCommitted adoption, and add a test bounding refusals per interval against an always-busy wrapper.
- **BR-6** [Important] `fake-contract-drift` FakeDeliveryEndpoint.Reserve returns a plain error where the real wrapper returns ErrRecipientBusy
  broker_test.go:59 returns errors.New("changed or reserved"); wrapcmd/peer_delivery.go:354 returns couchmessage.ErrRecipientBusy, which the new admit queue-on-busy branch keys on. Tests on the standard fake cannot reach that branch (ARCH-MOCK); make the fake return ErrRecipientBusy.
- **BR-7** [Minor] `untested-recovery-path` BR-1 regression test only goes red at the package deadline (about 590s)
  2nd finding in family untested-recovery-path. Rule: recovery-path tests should fail within a bounded wait, not the global go test timeout. Use a short bounded wait for the delivery after the gate opens.
- **BR-8** [Minor] `queue-expiry-lazy` Queued-message expiry runs only inside promote, so receipts can read queued past QueueWait while the worker is in a long held delivery

## Round 3 — 2026-10-12T13:04:54-07:00 (claude) — passed

### Disposed

- BR-5 — addressed — Reverting broker.go:686-688 to releaseReservationLocked makes TestPromotionHoldsWhileWrapperBusy fail with 2161 refusals; it passes with the fix. AlreadyCommitted keeps its explicit wake (bounded: each adoption dequeues the head).
- BR-6 — addressed — broker_test.go:59-61 now returns ErrRecipientBusy, as wrapcmd/peer_delivery.go:354 does. The admit busy-enqueue branch is covered by busyEndpoint and wait_notice_test; no fake/wrapper conformance test exists yet.
- BR-7 — withdrawn — The implementor kept the deadline-bounded wait, citing the repo convention against guessed short timeouts (lessons 479). The test still fails on regression, just at the deadline; acceptable.
- BR-8 — addressed — Status (broker.go:345-350) now ticks a queued receipt past its Deadline. Disabling it makes TestQueuedMessageReadsExpiredWhileTheWorkerIsBusy fail with status queued.

### Raised

- **BR-9** [Minor] `queue-expiry-lazy` Send's same-ID replay (broker.go:456) returns a stale queued receipt past QueueWait
  This is the 2nd finding in family queue-expiry-lazy. Rule: every path that returns a stored receipt goes through one current-receipt helper that applies any due expiry. The enumeration is Status (fixed), StatusContext (delegates to Status), and Send's idempotent replay at broker.go:456-462 (not fixed). Extract the Status block into that helper and call it from both places.

## Round 4 — 2026-10-12T13:07:15-07:00 (claude) — passed

### Disposed

- BR-9 — addressed — receiptLocked (broker.go:350) now backs Status, Send replay (:462) and recordRecoveredLocked (:742); test at broker_queue_test.go:286 fails (Status queued) when the Send call is reverted to the raw map read.

## Open findings

(none — every finding has been disposed)
