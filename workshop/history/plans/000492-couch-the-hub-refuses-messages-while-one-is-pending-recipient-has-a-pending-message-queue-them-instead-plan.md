# Peer-message queue per recipient Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A message to an exactly addressed slot that is already taking one waits in a FIFO queue and is delivered in order; the sender gets `queued` with its position, never `busy`.

**Architecture:** The one-at-a-time rule stays where it protects something: the wrapper pastes one message at a time, and the broker's `ActorState.pending` is that one delivery. What changes is admission. Behind a pending delivery, the pure model (`Advance`) appends the message to a bounded per-recipient queue, spends the sender's allowance at admission as today, and emits `Queued` with a position. The actor's worker promotes the head when the slot is free: it reserves on the wrapper (observe, then reserve) and admits the message as pending with a fresh paste-by. A wrapper that reports busy is retried after a short wait, which holds the message instead of refusing it. Family routing (`--send-to pair`) still skips busy slots, because it means "a free slot".

**Tech Stack:** Go, `cmd/internal/couchmessage` (pure model + broker), fakes in `broker_test.go`.

---

## Why the limit exists (findings)

- `couchmessage/model.go` `ActorState` holds one `pending Message`; `Advance(Admit)` refuses a second message with `ErrRecipientBusy`. `routing.go` refuses an exact target whose candidate is `Pending`, and `broker.tryAdmission` refuses when the actor's inbox is full (capacity 1).
- The wrapper (`wrapcmd/peer_delivery.go` `reserve`) also allows one reservation or current delivery at a time and refuses when its observation sequence moved (operator typed). That is the real protection: pastes are serialized, never typed over a draft, a picker or the operator's input. Held deliveries (a draft, a scrolled pane, a picker) can keep the one pending slot for up to `MaxHold` (30 min, pair#435).
- Together: while the hub holds one message (for example behind its own draft), every other sender is refused, for up to about 32 minutes. That matches the 10-12 case: pair:3 was refused for about 17 minutes.
- So: keep one delivery at a time on the broker and the wrapper (the protection); queue admissions behind it (the fix).

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ActorState.queue` | `cmd/internal/couchmessage/model.go` | new |
| `Advance` (Admit, Promote, Tick, Disconnect, sender cancel) | `cmd/internal/couchmessage/model.go` | modified |
| `ResolveRecipient` (exact target) | `cmd/internal/couchmessage/routing.go` | modified |
| `Candidate.Pending` | `cmd/internal/couchmessage/routing.go` | modified (now covers a queue) |
| `MaxQueued`, `QueueWait` | `cmd/internal/couchmessage/model.go` | new |

- **ActorState.queue** — `[]Message` in admission order, behind `pending`. 1:N with an actor; the actor owns it, and the broker never touches it except through `Advance`.
  - A queued message's `Deadline` is its **latest promotion** (`admission + QueueWait`). Promotion replaces it with the paste-by (`now + PasteTimeout`), so `Message.Deadline` keeps one meaning per phase and `Horizon()` still bounds a pending delivery.
  - **Admit:** with `pending` empty and no queue, it becomes pending (today's path). Otherwise it appends, if `len(queue) < MaxQueued`, spends the allowance exactly as today (a self-addressed submit refills), and emits `Queued` with detail `position N`. A full queue refuses `ErrRecipientBusy` with "queue full": a bounded resource. The same message re-admitted is a no-op (deduplicated against pending and the queue).
  - **Promote (new event):** with `pending` empty, it pops the head, sets `Deadline = e.At + PasteTimeout`, makes it pending in `Queued` phase, and re-emits `Queued` (detail empty) with the new message. A head whose `Deadline` has passed is finished `Expired` ("waited in the queue past N min"), and the next head is tried.
  - **Tick:** expires queued messages past their `Deadline`, in addition to today's pending expiry.
  - **Disconnect (recipient):** cancels every queued message ("wrapper disconnected"), like pending.
  - **Sender cancel:** a new `Cancel` event (`Message` names the message) removes it from the queue with `Cancelled` "sender disconnected". The broker's `disconnectLocked` already finishes a pending message whose sender left; it now also cancels that sender's queued ones.
  - **DRY rationale:** one place decides ordering, expiry and allowance; the broker stays a scheduler.
  - **Future extensions:** a priority (operator over peers) would order the queue here.
- **MaxQueued = 32** per recipient; the global `MaxReceipts` (256) still bounds all receipts. **QueueWait = 45 min**: longer than a full hold plus delivery (`MaxHold + PasteTimeout + DeliveryTimeout`, about 32 min), shorter than `ReceiptLifetime` (1 h), so an expired queued receipt stays readable.
- **ResolveRecipient:** an exact target that is `Pending` resolves to its binding (the model queues). The `Remaining <= 0` breaker is unchanged. Family routing still requires `!Pending`.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Broker.tryAdmission` | `cmd/internal/couchmessage/broker.go` | modified | wrapper Observe/Reserve |
| `Broker.runActor` / `promote` | `cmd/internal/couchmessage/broker.go` | modified / new | wrapper Reserve + delivery |
| `FakeDeliveryEndpoint` | `cmd/internal/couchmessage/broker_test.go` | reused | wrapper endpoint |

- **tryAdmission:** for an exact route to an actor with a pending, reserved or queued message, skip the wrapper and apply `Admit`, which queues; then wake the worker. If the actor is free but the wrapper's `Reserve` fails busy (the sequence moved, or it is mid-release), queue too instead of refusing. Family routes keep today's path. Ordering holds because a non-empty queue sends every later message to the queue.
- **runActor + promote:** the worker selects on `inbox`, a `wake` channel (capacity 1) and a retry timer. When the actor has no pending message, no reservation and a non-empty queue, `promote` sets `a.reservation` to the head's ID (so concurrent admissions queue behind it), then calls `Observe` and `Reserve` without the lock. On success it applies `Promote` and pushes the message to the inbox. On `AlreadyCommittedError` it adopts the retained receipt (as `adoptRetained`) and drops the head. On another error it releases the reservation and retries after `PromoteRetry` (1 s, real timer), which is the hold. `Tick` expiry bounds the wait. After each `DeliveryFinished`, the worker promotes.
- **Fake:** `FakeDeliveryEndpoint` already models one reservation (`reserved` until `Release`) and the sequence check, which is enough to test holding.
- **CLI:** `couch --send-to` prints the receipt's status; add the detail when present (`… queued (position 2)`). Code in `couchcmd/messages.go:108`.

## Tasks

### Task 1: Model queue (pure)

**Files:** Modify `cmd/internal/couchmessage/model.go`; test `cmd/internal/couchmessage/model_test.go`.

- [ ] Write failing tests:
  - `TestAdmitBehindPendingQueuesInOrder`: register; admit m1 (pending); admit m2, m3 → `Queued` with detail `position 1` and `position 2`; the allowance drops by 3.
  - `TestPromoteTakesHeadWithFreshPasteBy`: finish m1; `Promote` at t → pending m2 with `Deadline == t+PasteTimeout`; a receipt is re-emitted for m2.
  - `TestQueuedMessageExpiresAfterQueueWait`: `Tick` past m3's `Deadline` → `Expired`, removed.
  - `TestQueueFullRefuses`: `MaxQueued` queued, the next → `ErrRecipientBusy`.
  - `TestDisconnectCancelsQueue`, `TestCancelRemovesSendersQueued`, `TestReadmitQueuedIsNoop`.
- [ ] Run `go test ./cmd/internal/couchmessage -run 'Queue|Promote|Readmit'`: FAIL.
- [ ] Implement: `queue []Message` field and `Queued()` accessor (`len`). In `Admit`: dedupe against pending and the queue; append path. New `Promote` and `Cancel` kinds; extend `Tick` and `interrupt`/Disconnect. Use `Expired` for queue expiry with detail `"waited in the queue past 45m0s"`.
- [ ] Run: PASS. Commit `#492: model: a recipient queues messages behind its delivery`.

### Task 2: Routing — exact targets queue

**Files:** `routing.go`, `routing_test.go`, `broker.go` (`actorCandidate`).

- [ ] Failing test: exact route to a `Pending` candidate resolves to its binding; family route still skips it.
- [ ] Implement: drop the `c.Pending` refusal in the exact branch. `actorCandidate` keeps `Pending` true for a pending, reserved, inbox or queued message.
- [ ] PASS, commit.

### Task 3: Broker admission and promotion

**Files:** `broker.go`; tests in `broker_test.go`.

- [ ] Failing tests (fake endpoints, real worker):
  - Update `TestBrokerAdmissionIsImmediateAndMailboxIsBounded`: the second send is now `Queued` (position 1), not busy, and it is delivered after the first's outcome.
  - `TestHubQueuesSeveralSendersAndDeliversInOrder` (the 10-12 case): hub `ops:0` with a pending message held (the fake receives it and sends a `Held` interim, then no outcome); three senders `pair:1`, `pair:2`, `pair:3` each get `Queued`. Release the hold with a `Submitted` outcome → the fake's `delivered` yields pair:1, pair:2, pair:3 messages in order, each after the previous outcome.
  - `TestPromotionHoldsWhileWrapperBusy`: the fake's sequence differs from the observed one (operator typing) for the first promotion attempts, so `Reserve` fails; after the sequence settles, the message is delivered. Its receipt is never terminal while held.
  - `TestQueuedSendFromDisconnectedSenderIsCancelled`.
- [ ] Implement in `broker.go`: a `wake chan struct{}` (capacity 1) on `SlotActor`, created in `RegisterWithAllowance`. In `tryAdmission`, queue when the actor is busy (as above) or when the wrapper's `Reserve` fails with anything but `AlreadyCommitted`, then `signal(a.wake)`. In `runActor`, a `promote(ctx, a)` step after each delivery and on `wake` or the retry timer. In `disconnectLocked`, apply `Cancel` for the departed sender's queued messages on every destination.
- [ ] Run `go test ./cmd/internal/couchmessage` (full package): PASS. Commit.

### Task 4: CLI shows the position; docs

**Files:** `couchcmd/messages.go:108`, `couchcmd/skills/couch/SKILL.md`, `README.md`, `atlas/couch.md`.

- [ ] Print ` (position N)` when a `queued` receipt carries a detail; test in `couchcmd/messages_test.go`.
- [ ] Skill: in "Sending and receiving" and the rung table, state that a busy exact recipient queues the message in order (`queued`, position N), and that `busy` now means only a full queue (32). Replace "Delivery waits on an occupied composer… A message still waiting after 30 seconds expires undelivered" with the queue + paste-by wording. README + atlas: the same, one paragraph each.
- [ ] `go test ./cmd/internal/couchcmd -run 'Skill|Readme|Message'`, commit.

### Verification

- [ ] `make -k test` (holds the machine lock), `go test ./...`, `TMPDIR=<scratch> make test-changelog`.
- [ ] Live: after landing, `make build` on pair:0 is the operator's step (a running Couch keeps the old broker until restart), so the live check is the hub's after a Couch restart: three slots send to `hub` while it is mid-turn, and all three arrive in order.

## Revisions

- 2026-10-12, TL approval (ops:0): a queued message cancelled because the recipient
  disconnected or relaunched ends with a receipt a sender can act on:
  `Cancelled`, detail `<slot> disconnected (relaunched or exited) before delivery; resend`.
  It never vanishes. `couch --message-status` (text form) prints the detail after the
  status. Task 1 asserts the detail; Task 4 asserts the CLI line.
