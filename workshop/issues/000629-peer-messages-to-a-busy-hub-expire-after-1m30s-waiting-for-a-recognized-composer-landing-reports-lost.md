---
id: 000629
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '6e4bce549dff28db28517486e64ba633addee2e5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T18:58:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Peer messages to a busy hub expire after 1m30s waiting for a recognized composer; landing reports lost

## Problem

## Spec

Peer messages to a busy hub expire undelivered. `couch --message-status` reads
`expired — delivery deadline elapsed: waiting for recognized composer (1m30s)`.

Seen 2026-10-16 after the 17:35 Couch restart:
- ariadne:2 -> ops:0 33f83c87 and its resend 2fd247eb (ariadne#412's landing report): both expired. The hub never saw the landing; it found it by peeking ariadne:2.
- pair:7 -> ops:0 f854048e (pair#604's landing report) expired; pair:7 resent by hand later.

The hub (ops:0) is a Claude pane that is mid-turn most of the time, running tools. Its composer is not "recognized" while a turn runs, and 90 s is shorter than a typical TL turn. A sender can't tell when to retry, so each slot invents its own resend loop.

Expected: a message to a busy agent waits until its composer is next recognized (Claude Code also queues typed input mid-turn, so delivering while busy may be fine). It should never silently expire on a live, connected receiver. Expiry stays for a receiver that is gone.

Measured by pair:2 (18:47): after the restart, 31 hub-bound deliveries were cancelled, exactly one every 90 s from 17:37 to 18:22, from 11 senders. The hub wrapper holds the head of its queue for the full composer wait, so one stuck head serialises every message behind it: the hub received nothing for 45 minutes. Confirmed deliveries resumed at 18:29.

## Done when

- A message to a connected receiver whose turn runs past the deadline is delivered when its composer next comes back, or queued into the running turn; a test with a fake receiver busy for longer than the deadline shows it delivered once.
- A message that cannot be delivered yet never blocks the ones behind it, or the queue drains in one go once the composer returns.
- Expiry happens only when the receiver is disconnected or gone, and the receipt says which.

## Plan

- Design: [busy receiver queue plan](../plans/000629-busy-receiver-queue-plan.md).
- [ ] Reproduce connected-receiver expiry and independent-recipient blocking with controlled clocks and stateful fakes.
- [ ] Remove accepted pre-paste expiry across model, broker, remote endpoint and wrapper; preserve post-paste uncertainty and exact binding ownership.
- [ ] Verify queue drain, cancellation, native receiver behavior and the recorded full suite; close, land and report the PR.

## Log

### 2026-10-16

- Claimed in pair:3 after parking/releasing #603 with its two CI attempts and draft PR recorded. Started a fresh branch from main. Root cause trace: wrapper reservation accepts before composer readiness; its accepted pre-paste message expires after 90s, while broker queue/model and RemoteEndpoint impose additional 45m/hold horizons. Every layer must agree on connection-scoped waiting; changing one timeout would strand or misclassify the same work elsewhere.
- Additional isolation bug: broker promotion performs a wrapper Release RPC while holding b.mu, blocking independent recipient admission/status until that RPC returns. Plan covers it with a gated receiver test, as part of the requested queue independence.
- Initial receipt/queue plan is committed for hub review before code. Reuses existing status strings and stateful fake endpoints; keeps FIFO per receiver, immediate promotion after confirmation, independent receiver workers, and all post-paste input protections. No body batching or unsafe mid-turn input is introduced. No implementation changes made.
- start-plan reported a nonblocking base catch-up warning: weave refresh rejected --defer-settings; the binary allowed planning on the current checkouts. No peer tools or machine settings changed to work around it.
- Hub approved the connection-scoped wait plan (receipt 573ccccc-8f55-4d1e-ba4f-e1542c3a0cb6). Required addition: held message status must expose the composer wait and queue position. The Release-under-lock finding is excluded: no retained measurement ties it to the 17:37–18:22 outage. File it independently; do not change that path in #629.

## Revisions

### 2026-10-16 19:02 PDT — approval conditions

The approved design adds composer-wait/position status coverage and defers the separately discovered Release-under-lock contention. The durable plan's Revisions section records the exact delta; no status strings or protocol fields are added.
