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
flow: {kind: quick, provenance: inferred, spec: "e699b34a", done: "afad8e96"}
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
- [x] Reproduce connected-receiver expiry and independent-recipient progress with controlled clocks and stateful fakes.
- [x] Remove accepted pre-paste expiry across model, broker, remote endpoint and wrapper; preserve post-paste uncertainty and exact binding ownership.
- [ ] Verify queue drain, cancellation, native receiver behavior and the recorded full suite; close, land and report the PR.

## Log

### 2026-10-16

- Claimed in pair:3 after parking/releasing #603 with its two CI attempts and draft PR recorded. Started a fresh branch from main. Root cause trace: wrapper reservation accepts before composer readiness; its accepted pre-paste message expires after 90s, while broker queue/model and RemoteEndpoint impose additional 45m/hold horizons. Every layer must agree on connection-scoped waiting; changing one timeout would strand or misclassify the same work elsewhere.
- Additional isolation bug: broker promotion performs a wrapper Release RPC while holding b.mu, blocking independent recipient admission/status until that RPC returns. Plan covers it with a gated receiver test, as part of the requested queue independence.
- Initial receipt/queue plan is committed for hub review before code. Reuses existing status strings and stateful fake endpoints; keeps FIFO per receiver, immediate promotion after confirmation, independent receiver workers, and all post-paste input protections. No body batching or unsafe mid-turn input is introduced. No implementation changes made.
- start-plan reported a nonblocking base catch-up warning: weave refresh rejected --defer-settings; the binary allowed planning on the current checkouts. No peer tools or machine settings changed to work around it.
- Hub approved the connection-scoped wait plan (receipt 573ccccc-8f55-4d1e-ba4f-e1542c3a0cb6). Required addition: held message status must expose the composer wait and queue position. The Release-under-lock finding is excluded: no retained measurement ties it to the 17:37–18:22 outage. File it independently; do not change that path in #629.

- Implementation checkpoint: fake receiver clock jumps through 91s, 45m and one hour reproduce expiry on the prior implementation (/tmp/pair629-busy-receiver-red.log); wrapper unknown-composer, draft and picker waits at two hours reproduce it independently (/tmp/pair629-wrapper-red.log). Both regression groups pass after retiring pre-paste horizons. The full couchmessage race run passed (21.883s, /tmp/pair629-couchmessage-final.log); all TestPeer wrapper cases passed (44.009s, /tmp/pair629-peer-final.log). CLI/embedded-skill/README checks passed (0.300s, /tmp/pair629-cli-docs.log). Initial package failures were deterministic old status/position expectations, updated to the approved receipt semantics; logs retained. Full wrapper run and suite remain pending.
- Held/queued receipt projection includes the entire receiver line and standing composer wait, using existing words and statuses. Socket tests count one commit across queued/held/queued progression. Queue capacity and terminal receipt retention remain bounded; active accepted receipts survive retention sweeps. The independent Release-lock finding was filed unclaimed as #631; this branch leaves that lock path unchanged.
- Disconnect safety remains conservative: uncommitted queued work cancels with the exact receiver reason; a handed commit has an uncertain-input outcome (indeterminate, wrapper disconnected) because a last queued status cannot prove no subsequent paste. No body replay or rerouting to a new binding is authorized. The automatic-input watchdog keeps its own bounded diagnostic state; only its obsolete pre-paste expiry eligibility check was removed.

- Native qualification: isolated Claude Code 2.1.302 draft and submit conformance passed (6.046s and 5.073s, /tmp/pair629-native-peer-draft.log and /tmp/pair629-native-peer-submit.log). The submit case rendered the exact envelope and confirmed one submission. The recorded suite was requeued after integrating current main/#617; its earlier reservation was interrupted before a suite slot was granted.

- Broad-run limit: the initial manual race command used a 10-minute package limit and stopped in TestHarnessTTYFixtureConformance/qoder/1.1.60/composer.raw after terminal replay had consumed 9m51s (/tmp/pair629-packages.log). A one-second process sample resolved to active replayHarnessTTYSplit/terminal processing, not a peer wait. The later full race command uses 15 minutes; production timeouts were not increased. The recorded suite retains its existing repository-declared 20-minute package budget.

### Post-restart live check — owned by ops:0 and operator

TL explicitly deferred the real-environment check to the next ordinary Couch restart (receipt b66971a3-a6c6-4dab-9503-e837f150720e). Do not stage or rebuild pair:0 from this slot. After restarting with the landed binary:

1. Keep ops:0 in a tool-running turn for longer than 90 seconds. From an idle connected peer, send `couch --send-to hub --message 'pair#629 live check A; acknowledge once when this turn finishes'`, then a second message with body `pair#629 live check B; acknowledge after A`. Record both returned IDs. Do not resend accepted messages.
2. While ops:0 is still mid-turn, run `couch --message-status ID` from that sender after 90 seconds. A must remain queued (or held for a visible operator blocker), with `position 1; waiting for recognized composer` or the wrapper's more specific standing wait. B must remain queued at position 2 with the same standing wait. Neither may expire due to elapsed composer-wait time. A deliberately hung connected receiver keeps this explicit waiting status indefinitely.
3. Let the hub return to its safe empty composer. Verify A arrives once, its receipt becomes submitted, then B arrives once and becomes submitted after its confirmation. The second delivery needs no manual resend and no serial 90-second penalty. If the hub starts another turn after A, B waits for the next safe composer rather than pasting into it.
4. If a separate disposable receiver is available, queue a message while it is busy, then disconnect/relaunch it. Uncommitted queued messages must cancel with the exact receiver/relaunch reason; a handed commit may be indeterminate because input cannot be ruled out. No old message may move into the replacement incarnation. Preserve IDs/receipts if any observation differs and fix forward.

## Revisions

### 2026-10-16 19:02 PDT — approval conditions

The approved design adds composer-wait/position status coverage and defers the separately discovered Release-under-lock contention. The durable plan's Revisions section records the exact delta; no status strings or protocol fields are added.

### 2026-10-16 — real-environment qualification deferred by TL

The hub forbids staging in pair:0, which it is rebuilding for #617. Finish package/suite verification, close and land #629 first; ops:0 and the operator own the busy-hub live check immediately after the next restart. The executable steps above replace the plan's pre-landing staging task; no staging or restart is authorized here.
