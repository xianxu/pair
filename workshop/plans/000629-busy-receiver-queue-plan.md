# Busy receiver message queue implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Steps use checkbox syntax. One atomic close boundary; no milestone tags.

**Goal:** Retain accepted messages while an exact connected receiver is busy, then deliver each once when its composer can accept input.

**Architecture:** Keep the existing broker queue and wrapper input owner. A pre-paste wait is queued work, not a failed delivery; wall-clock time cannot end it while that receiver remains connected. After a paste, retain the existing bounded confirmation and quarantine rules. Per-receiver workers remain independent and no wrapper RPC runs under the broker's shared lock.

**Tech Stack:** Go; pure couchmessage transitions; stateful fake delivery endpoints; wrapper harness and native receiver conformance.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| ActorState / Advance | cmd/internal/couchmessage/model.go | modified |
| Message timing contract | cmd/internal/couchmessage/model.go | modified |
| PeerDeliveryState / AdvancePeerDelivery | cmd/internal/couchmessage/delivery.go | existing; preserve input uncertainty |

ActorState owns one pending message and a FIFO queue for one exact Binding. Admission spends allowance once; replaying the same ID/body does not spend or paste again. Keep MaxQueued=32 and MaxReceipts=256: exhaustion refuses new admissions explicitly rather than dropping accepted work. Terminal receipts retain their existing one-hour lifetime; active receipts already survive that sweep.

Message.Deadline remains the bounded reservation-to-commit transport admission limit. It must no longer be used as an accepted message's composer-wait timeout, nor may the 45-minute queue deadline or old hold horizon discard accepted work. Retire obsolete timing helpers/constants only after enumerating their production and test consumers.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Broker.runActor / deliver / promote | cmd/internal/couchmessage/broker.go | modified | exact-wrapper lifecycle, bounded worker per receiver |
| RemoteEndpoint.Deliver | cmd/internal/couchmessage/endpoint.go | modified | commit once, then read-only status RPCs |
| proxy.dispatchPeer / peerDelivery | cmd/internal/wrapcmd/peer_delivery.go | modified | composer observation and sole PTY input writer |
| FakeDeliveryEndpoint / busyEndpoint | cmd/internal/couchmessage/broker_test.go and broker_queue_test.go | existing; reuse | retained state, controlled busy/reservation and outcomes |

Remote status calls remain individually bounded by Call's RPC timeout and parent cancellation. The connected receiver owns a pre-paste wait, so remove the independent overall pre-paste horizon at both RemoteEndpoint and Broker. Wrapper dispatch remains the authority for safe readiness and the paste-relative DeliveryTimeout; do not introduce readiness cached across observations or another input writer.

## Receipt and queue rules

| Observation/event | Receipt / action |
|-------------------|------------------|
| Accepted, receiver busy/unrecognized, no body written | queued with actionable wait detail; retain ID/body/AcceptedAt without a composer-wait expiry |
| Operator draft, scrollback or picker blocks paste | held with reason; no overwrite and no timeout-induced discard |
| Composer becomes safe and empty | existing input owner pastes once, then delivering; confirmation window starts at paste |
| Exact submission confirmed | submitted; release delivery and immediately consider the next FIFO message |
| Human input before paste, explicit release, sender disconnect | existing cancellation policy; visible terminal receipt |
| Receiver disconnect/replacement | cancel queued work with exact receiver/relaunch reason; do not move it to another incarnation |
| Timeout/interruption after any input may have been written | indeterminate and existing automatic-input recovery; never replay body |
| Broken/unresponsive endpoint RPC | bounded error with conservative outcome; connectivity uncertainty is not proof of absence |

Same-receiver messages share composer availability, so shuffling its blocked head cannot make a later message safe. Retain FIFO, remove serial 90-second expiry penalties, and promote immediately after each confirmed outcome. If the native composer disappears after a submit, remaining messages stay queued until its next safe observation; this design does not batch unrelated envelopes into one submit. Other recipients must progress even while the first is busy or its release RPC is delayed.

## Ordering, constraints and lifecycle

- ARCH-ORDER: enumerate busy -> ready -> paste -> confirm, repeated status/replayed Send, disconnect/replacement, sender exit, partial writes and cancellation. Keep all model changes through Advance; do not collapse post-input uncertainty into queued or expired.
- ARCH-PURE / ARCH-DRY: alter existing transitions and IO seams rather than add another queue, retry store or readiness policy. No new receipt status strings or protocol fields are planned.
- ARCH-CONSTRAINTS / ARCH-FUNERAL: waits are connection-scoped, not wall-clock-scoped. Existing queue/receipt caps bound memory; disconnect/replacement and terminal receipt retention end ownership. Existing RPC, reservation and post-paste budgets remain bounded. No new files or growing stores at runtime.
- ARCH-SECURE: every queued message retains its complete exact Binding and envelope; human drafts and uncertain automatic input stay protected. Reconnect/replacement never inherits authority to paste old messages.
- ARCH-MOCK / ARCH-PURPOSE: test the broker and wrapper paths with existing stateful fakes, not just deadline helpers. Run native receiver conformance and stage a live test through the hub before landing because this affects the operator's real receiver.

## Tasks

### 1. Reproduce accepted-message expiry and blocked queue

- [ ] Add fake busy receiver cases to model_test.go / queue_test.go / broker_queue_test.go. Jump an injected clock past 90 seconds, 45 minutes and one hour; query Status, replay Send and snapshot receipts. Assert queued IDs/body/AcceptedAt survive, no input occurs while busy, and another recipient can receive independently.
- [ ] Cover both refusal-to-reserve and an already committed pre-paste wait with queued followers. Release readiness and assert FIFO, exactly one delivery per ID and no artificial retry delay between confirmed deliveries.
- [ ] Add a gated Release fake proving that one recipient's slow release does not block another recipient's Send/Status. Use channels and fake time, not wall-clock sleeps.
- [ ] Run go test ./cmd/internal/couchmessage -run 'Busy|Queue|Promotion|Release' -count=1 and retain expected failures before implementation.

### 2. Correct every pre-paste timing consumer, preserving uncertain-input safety

- [ ] Sweep model Tick/admission/Promote/DeliveryStarted/DeliveryFinished, broker receiptLocked/deliver, RemoteEndpoint.Deliver and wrapper dispatch/deadlineLocked. Remove accepted pre-paste expiry across all of them; retain bounded commit RPC admission and paste-relative confirmation.
- [ ] Move promote's release RPC outside b.mu with exact actor/reservation revalidation. Keep per-actor worker ownership and bounded promotion pacing on reservation refusals.
- [ ] Add wrapper fake-clock regressions: unrecognized mid-turn composer and picker/draft remain queued/held beyond old limits, then paste/submit exactly once when safe. Update the old tests that deliberately required busy expiry; retain post-paste timeout, operator-interruption, partial-write and duplicate-ID controls.
- [ ] Add RemoteEndpoint coverage counting exactly one commit across prolonged queued/held polling, cancellation and eventual submission. Retain per-RPC timeout and identity validation controls.
- [ ] Run go test -race ./cmd/internal/couchmessage ./cmd/internal/wrapcmd with targeted filters during iteration, then both packages in full. Record outcomes and commit the implementation.

### 3. Document, qualify and land

- [ ] Update README, atlas/couch.md and embedded couch skill timing/receipt descriptions. Sweep obsolete 90-second, 30-minute hold and 45-minute queue promises and their tests together.
- [ ] Stage via sdlc stage --build with hub approval; use a real busy receiver and one queued report, verify receipt while busy and one delivery when it returns. Clear the stage afterward. Do not send unsolicited test work to the hub's live composer.
- [ ] Run the applicable native receiver conformance and required full suite via sdlc local-check on the pushed head; apply the documented flake rule to unrelated failures.
- [ ] Close with verification evidence and the binary's mandatory fresh review, open PR, follow the landing line, merge with sdlc merge --yes and report the PR to the hub.

## Revisions

### 2026-10-16 19:02 PDT — TL approval conditions

Approved by ops:0 (receipt 573ccccc-8f55-4d1e-ba4f-e1542c3a0cb6), with these deltas taking precedence over the initial task list:

- Add a status regression for a hung-but-connected receiver: queued/held receipts must name the existing composer wait and queue position. Reuse existing wording; do not add status strings. Include the pending pre-paste head as position 1 when showing the receiver's complete waiting line, and propagate its standing composer wait to followers.
- Remove the proposed Release-outside-b.mu change and its gated Release regression from #629. The Log has no measurement tying it to the outage window; file it as independent work. The existing per-recipient workers and the no-serial-expiry drain are the queue scope here.
- Proceed with implementation, post-paste protections and the specified fake busy receiver regression. The next ordinary Couch restart carries the landed change; do not independently restart the hub.

### 2026-10-16 — implementation evidence

The approved Release exclusion is tracked separately as #631. The broker retains its existing conservative disconnect classification once a commit has been handed off: indeterminate rather than cancelled, because a previously observed pre-paste receipt cannot prove no input occurred before disconnection. Never replay that body. Fake-clock tests now cover refusal, queued and held waits, exact-binding replacement, and independent receivers; the socket test covers one commit across queued/held transitions.
