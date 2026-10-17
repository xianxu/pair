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
- [x] Verify queue drain, cancellation, native receiver behavior and the recorded full suite. Close review, landing and reporting follow through their normal gates.

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

- Full wrapper race verification passed: go test -race ./cmd/internal/wrapcmd -count=1 -timeout=15m (743.753s, /tmp/pair629-wrapcmd-final.log). The additional race run excluding only the exhaustive terminal fixture also passed (58.624s, /tmp/pair629-wrapper-rest-race.log), proving the remaining cases were exercised. The earlier 10-minute stop was fixture workload, not a delivery wait. The recorded repository suite remains queued.

- Queue cleanup: interrupting the superseded suite reservation left test-lock.py PID 42255 orphaned under PID 1 with dead watched PID 42240, retaining ticket 9 for sdlc-local-1087195770. Verified the exact owned ticket/run, interrupted only that helper, and confirmed both process and ticket disappeared; no gate file was manually removed. Proposed the independent cancellation-cleanup finding to the hub for ariadne (receipt def74365-9589-4cbc-b702-d0d5b779a252); the hub filed ariadne#416 and dispatched it to ariadne:5 (receipt 7243a07f-c75b-46ae-ae60-275e0ca21a3e).

- First recorded suite on 74e0a6b6e8b16e1340717f7eed17967bad161828 finished in 11m12s after a 2022s queue wait. One deterministic branch failure: termcmd/TestNoDeclarationCarriesTwoStackedGodocs interpreted the MaxQueued comment opening word Connected as another declaration. Corrected the comment to open with MaxQueued; targeted guard passed (0.581s, /tmp/pair629-godoc-green.log). All other packages passed, including couchcmd 53.406s, couchcore 291.658s, couchmessage 17.595s and wrapcmd 53.941s. Retained full failed log at .git/sdlc/test-stats/failed/c491c39823deb69a3efb0862ee8fa1c6.log (sha256 71d94bf2c651). This is a corrected branch defect, not a flake retry; run the required suite again on the corrected/integrated head.

- Corrected integration head: 57350d7eef208c568fcabec337501e9fd6cd0a37, including current main/#619. The declaration-comment guard passed again after integration (0.372s). Required suite queued on that pushed head as run 0ee3855e9e788af24f895643528ef406, initially position 8; active log .git/sdlc/test-stats/inflight/0ee3855e9e788af24f895643528ef406/run.log. No executable message behavior changed after the first recorded run.

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

### 2026-10-16 — complete the indefinite-wait resource bound

Retire pre-paste diagnostic history: distinct picker titles otherwise grow the old expiry accumulator indefinitely, despite its reset at paste. The standing receipt reason remains; duration summaries remain only in the bounded post-paste phase. The plan's latest Revisions entry records the evidence and minimal choice. Cancelled the superseded queued suite on 57350d7e before admission and cleaned only its own orphan helper (watch 16752; helper 16769); confirmed both and ticket 22 were gone. Requeue the completed head after its regression passes.

- Resource regression evidence: 200 distinct picker episodes retained 199 obsolete pre-paste reasons before the guard (/tmp/pair629-picker-history-red.log). With pre-paste accounting retired, all peer wrapper race tests pass (26.361s, /tmp/pair629-peer-history-green.log), including current picker reasons and one paste after return; godoc guard passes (0.372s). Standing queued/held detail is unchanged and the existing post-paste duration summary remains.

### 2026-10-16 — early draft PR exception

The hub explicitly approved opening an early draft during the machine-suite wait (submitted receipt 43543b66-d791-42fe-8640-cd30439d8745). Opened draft https://github.com/xianxu/pair/pull/389 on a5e6c71025f80cc04b0d8f48ac89835428cc0b9f after sdlc pr --dry-run validated the same-repository target and branch push. The binary has no draft flag, so gh pr create --draft supplies only that approved exception. The required suite, close review and normal merge gates still precede landing. Completed-head suite run bfb8ab80f2d169d8307d75e93cac91dd is queued (initial position 10), active log .git/sdlc/test-stats/inflight/bfb8ab80f2d169d8307d75e93cac91dd/run.log.

- Draft #389 CI on completed code head a5e6c71025f80cc04b0d8f48ac89835428cc0b9f is green: macOS conformance run https://github.com/xianxu/pair/actions/runs/39000060905 completed successfully, along with merge-check and local-gates-guard. Saved native run log /tmp/pair629-ci-39000060905.log; all six required native cases/subcases executed and passed, including wrapped-zellij (23.21s). Required local suite remains queued; no close/landing is claimed.

### 2026-10-16 — completed-head required suite

Required recorded suite passed on a5e6c71025f80cc04b0d8f48ac89835428cc0b9f: exit 0 in 9m59s after 4441s waiting for shared capacity, recorded as sdlc-local/suite. Saved log .git/sdlc/local-checks/a5e6c71025f80cc04b0d8f48ac89835428cc0b9f/suite.log, sha256 prefix ef09fbe9aec9. All packages passed, including couchmessage 20.073s, wrapcmd 52.132s and the corrected termcmd declaration guard 0.613s. This completed-head run follows the retained deterministic comment failure and its correction; no flake retry was needed. Only documentation changed since the tested head. The final Plan verification row now separates completed verification from the upcoming close/landing gates; neither is claimed complete here.

### 2026-10-16 — close round 1 repair plan

Close returned REWORK with BR-1 (sender disconnect infers safety from queued/held presentation after handoff) and BR-2 (existing fake clock does not advance real context timers). Preserve the existing uncertainty contract by classifying sender loss from the active delivery job, not the displayed status; regress both interim states and uncommitted controls. Independently assert broker and endpoint lifetime contexts at their IO seams, including bounded commit and parent cancellation. An injectable endpoint transport seam permits the latter without long wall-clock sleeps or testing a private timeout helper. No new status, protocol or lifecycle mechanism. Post this quick repair plan to the hub, proceeding unless objected (receipt 16f6a4c7-8481-49d4-ad67-8919c4890760). Re-run targeted race, the required recorded suite and close; the first SHIP was not granted.

- BR-1 red/green: TestBusyReceiverSenderDisconnectPreservesHandoffUncertainty failed for both queued and held handed commits (/tmp/pair629-sender-disconnect-red.log); job-owned uncertainty passes with genuinely uncommitted cancellation controls (/tmp/pair629-sender-disconnect-green.log, race 1.228s). Swept receipt-status cancellation consumers: only sender disconnect used presentation state; receiver disconnect already uses internal phase, and queued cancellation remains pre-handoff.
- BR-2: new broker/endpoint IO-boundary tests assert no accepted-wait deadline beyond the parent, retain commit admission and cancellation, and exercise queued/held progress. Restoring each old outer timeout independently fails its own regression (/tmp/pair629-broker-lifetime-mutant-red.log and /tmp/pair629-endpoint-lifetime-mutant-red.log); both temporary mutations were restored. The endpoint fake uses the same request/response transport seam as Call. Full couchmessage race package passed after both fixes (26.853s, /tmp/pair629-review-fixes-race.log). Required suite and close will run on the repaired head. Hub repair-plan receipt 16f6a4c7-8481-49d4-ad67-8919c4890760 is submitted.

- Repair head pushed: 2f7cede0b6b5d67cb7ca411518a723312bddc5c9 (main integration adds only #637 issue details). Required suite run b3c73f60a00ffab075e1a2dbf7104c32 is queued, initial position 8; log .git/sdlc/test-stats/inflight/b3c73f60a00ffab075e1a2dbf7104c32/run.log. PR 389 macOS conformance run 39005656712 is in progress on this repaired head. Do not reuse the earlier suite pass for this executable change.

- Repaired-head GitHub checks all passed: macOS native conformance https://github.com/xianxu/pair/actions/runs/39005656712, merge-check and local-gates-guard. Saved /tmp/pair629-ci-39005656712.log confirms all six required native cases/subcases executed, including wrapped-zellij 22.44s. The required local suite runner survived the operator-run 22:22 Couch restart and continues its original queue reservation; no rerun or process cleanup was needed.
