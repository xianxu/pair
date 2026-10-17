---
id: 000618
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours: 4.39
card_mirror: 'ea381246f500919a30881823574373b7a748d77c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T15:08:44-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# Slots recovered by reboot or resume after a Couch restart can send but not receive

## Problem

After the 2026-10-16 14:57 Couch restart, a slot recovered with `couch --reboot` (pair:3, fresh, tag 1-pair-58) or
`couch --resume` (ariadne:5, tag 1-ariadne-46) can **send** to the hub but **cannot receive**: every
`couch --send-to` to it returns "Query couch --message-status … before any further send", and the status says
"receipt unavailable: no connected recipient retains this message". The agent is live and idle at its prompt.

- ariadne:5 recovered only after `couch --relaunch ariadne:5 --confirm --same-binary` (a plain relaunch refused as
  stale-binary, "changes nothing").
- pair:3 can't be relaunched either: "resume-binding-provisional: its agent has not completed a turn yet", although
  its agent completed a turn (it sent the hub its recovery report).
- The first `--relaunch` of pair:7 and of ariadne:5 each timed out at ~16 s and left a "recover / abandon" choice;
  a second call then succeeded.

So a recovered slot looks healthy in `couch --list` and to the operator, yet silently drops the TL's dispatches.

## Spec

- A `couch --send-to` lands, or says why it did not, whatever the machine load. The measured cause (Log) is not
  reboot/resume-specific: the supervisor's per-admission ownership check runs ~6 synchronous `zellij list-sessions`
  inside the 2 s admission budget, and the failure leaves no receipt.
- The message authority reads zellij liveness from one background-polled snapshot; a stale snapshot refuses with its
  age as the reason.
- A failed admission leaves a `refused` receipt with its reason; `--message-status`, the send's last line and
  `couch --list` (one state, one action: `can't receive: <reason>` and the fixing command) show it.
- A codex slot driven only by peer messages gets a confirmed binding, so relaunch works.
- The first `--relaunch` doesn't strand a recover/abandon choice on the 15 s park wait.

## Done when

- Tests: with a 1 s-per-listing stub zellij an admission lands inside the budget and reads no zellij; a failed admission
  reports `refused` with its reason; `--list` shows the receive state; a peer-message-only codex thread confirms its
  binding; a 20 s quit relaunches on the first call.
- Live: under a full suite, sends to a rebooted and a resumed slot land; the supervisor's zellij listing rate drops to
  the poller's 2 per second.

## Estimate

> *Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec              design=1.0  impl=0.1
item: smaller-go-module       design=0.1  impl=0.16
item: smaller-go-module       design=0.1  impl=0.12
item: smaller-go-module       design=0.2  impl=0.16
item: smaller-go-module       design=0.2  impl=0.16
item: greenfield-go-module    design=0.5  impl=0.24
item: smaller-go-module       design=0.2  impl=0.16
item: atlas-docs              design=0.05 impl=0.04
item: milestone-review        design=0.1  impl=0.16
item: milestone-review        design=0.1  impl=0.16
design-buffer: 0.15
total: 4.39
```

Items in order: the investigation and spec (done), ListingPoller, authority wiring, refused receipts and CLI, `--list`
receive state, peer-delivery ledger confirmation (new record kind plus watcher correlation), relaunch park wait, atlas,
M1 and M2 reviews.

## Plan

Durable plan: `workshop/plans/000618-slots-recovered-by-reboot-or-resume-after-a-couch-restart-can-send-but-not-receive-plan.md`.

- [x] M1 — polled listing for the message authority, refused receipts with their reason (TL: land these first)
- [ ] M2 — receive state in `--list`; a delivered peer message confirms a codex binding; the first relaunch waits inside its operation

## Log

### 2026-10-16
- 2026-10-16: closed M1 — Round 2 after REWORK: BR-2 send exchange budget (requestBudget, SendTransportTimeout = admission + 1 s) applied by both Call and the server; real-socket test TestTransportSendOutlastsAdmission RED without it. BR-3 TestMessageThreadResolverChecksReadOnlyThePoller: real checker + seeded thread + 1 s stub zellij, 50 checks run no zellij (RED without the poller). BR-5 no startup listing; poller loop lists at once, checks refuse not-yet-known. BR-4 before-numbers logged (6.1 zellij/s, 68% CPU); after-numbers follow the TL-scheduled restart after landing (plan revision). Minor: refusals carry explicit time. couchmessage, launcher, couchcmd Message tests green.; review verdict: SHIP

- 2026-10-16 (pair:2, read-only on pair:3): pair:3's wrapper (pid 36072) is registered and connected to the live
  supervisor (71982); binding, endpoint socket hash, pid file, ready-file nonce/session and session index all match; the
  broker never displaced it (supervision.jsonl subject cf45653c, one incarnation). Its codex rollout shows the TL dispatch
  4a662842 delivered at 15:03:40, after the reboot; my probe d166c04c at 15:23 was submitted. The TL's failed sends to
  pair:3 were at 15:06:46 and 15:07:47 (6ba71ae7, 0803aeee); to ariadne:5 at 15:04:44 and 15:05:07. No file in pair:3's
  scope changed between the failures and the success.
- Cause, measured: every send runs the supervisor's use-time ownership check about three times (sender once; recipient
  before Observe and before Reserve), each calling `zellij list-sessions` twice, inside the 2 s AdmissionTimeout. At load
  average 136 (~50 test processes) one listing took 0.13-1.07 s, so admission times out before Admit, no receipt is
  stored, and `--message-status` can only say "no connected recipient retains this message" (the status path skips the
  check). Hub deliveries in the same 15:05-15:09 window also show uncertain outcomes. Not reboot/resume-specific.
- Provisional: a codex launch has no chosen session ID; only the session watcher confirms the binding, from Pair-log
  turns that only the draft pane writes. A slot driven only by peer messages never confirms, so relaunch refuses.
- ~16 s relaunch: the park polls 15 s (`CompletionTimeout`) for the quit completion and child death, then returns
  ParkIncomplete with recover/abandon.
- Plan sent to the TL for approval:

  > pair:2 #618 PLAN (needs your approval: a new failure record and a cached liveness input). Root cause of can't-receive, measured: NOT reboot/resume-specific, it is load. Every send runs the supervisor's use-time ownership check about 3 times (sender once, recipient before Observe and before Reserve), and each run calls zellij list-sessions twice: ~6 synchronous listings inside the 2 s admission budget. Under today's load (load avg 136, ~50 test processes) one listing takes 0.13-1.07 s, so admission times out before Admit, no receipt exists, and --message-status can only say "no connected recipient retains". pair:3's files were identical at the 15:06/15:07 failures and my 15:23 success; hub deliveries in that same 15:05-15:09 window also show uncertain outcomes. Recovered slots were just the ones you were messaging then.
  > Plan:
  > 1. Root fix: the hot-path authority check answers terminal liveness from the supervisor's polled session presence (bounded freshness), keeping the cheap exact checks (pid file, ready file nonce/session, session index) per request; one check per party per admission, not three. (Deterministic request path, no per-item probes.)
  > 2. Visibility: a failed admission leaves a short-lived failure receipt with its reason, so --message-status says why (e.g. "recipient ownership check timed out") instead of "no connected recipient", and the send prints that reason on its last line.
  > 3. One state, one action: a slot whose last admission failed (with no later success) shows "can't receive: <reason>" in couch --list with the single command that fixes it.
  > 4. Provisional (root-caused): only Pair-log turns written by the draft pane confirm a codex binding, so a slot driven only by peer messages never confirms. Fix: the session watcher also confirms from a native round whose user text matches a message the broker delivered to that wrapper (exact body, after pid-not-before).
  > 5. 16 s relaunch: the park waits a fixed 15 s for the quit and child exit, then strands recover/abandon. Fix: the first call keeps waiting within the operation (async, operation-status) up to a longer bound instead of stranding.
  > Tests: a fake slow zellij (1 s per listing) still admits within budget; a failed admission reports its reason; a peer-message-only codex thread confirms; relaunch with a slow quit completes on the first call. Live: under a full suite, sends to rebooted and resumed slots land.
  > Full flow, durable plan in workshop/plans; I start with 1+2 (the root and the visibility) once you approve.
- TL approved items 1-2 first, 3 minimal (the reason from 2 in `--list` with the one fixing command, no new tracking),
  4-5 after 1-2 land. Load: the 136 reading was 15:36, not 15:05; recorded suites stayed within WF_TEST_SUITES; the
  supervisor's own ~1 core is hosted-client vt ingest (`ptychild.(*Child).ingest`), out of scope, reported to the TL.
- 2026-10-16 16:12 BEFORE (live Couch 71982, pre-#618 build): 183 zellij children in 30 s (6.1/s, a lower bound since
  polling misses the shortest), supervisor CPU 20.3 s per 30 s (68%), load average 62. AFTER is taken once the TL
  rebuilds pair:0 and the operator restarts Couch on the landed M1 (TL's order: land first, then measure).
- M1 review round 1 (REWORK): BR-2 Critical, the transport clamped every call to 2 s on both ends, so the send's verdict
  after a full admission never arrived; fixed by one per-request budget (`requestBudget`) shared by Call and the
  server, with sends at `SendTransportTimeout` (admission + 1 s); real-socket test RED without it. BR-3: a test through
  `messageThreadResolver` with the real checker and a 1 s stub zellij: 50 checks run no zellij (RED without the poller).
  BR-5: the startup refresh is gone; the poller loop lists at once and checks refuse "not yet known" until then.
  BR-4: after-numbers follow the restart (plan revision). Minor: refusals carry an explicit time.
- 2026-10-16 17:42 AFTER (live Couch 63216 on 4a97b611, 6 min after the restart, load average 39): 67 zellij children
  in 60 s (1.1/s, same lower-bound polling as BEFORE's 6.1/s), supervisor CPU 14.2 s per 60 s (24%, BEFORE 68%). Both
  numbers also include the operator-approved session cleanup (80 sessions to 24), which cut each listing's cost
  (0.045 s per listing at load 22) but not the call rate. The rate drop is the poller.
- 2026-10-16 18:47 AFTER, second sample (same Couch): 43 zellij children in 60 s (0.7/s), CPU 24%, load 47 (suites).
  Admission, from supervision.jsonl `hub_delivery` outcomes: the previous build's generation had 3 uncertain of 124 in
  2.6 h; this one has 0 uncertain of 37 in 1.2 h. Its 31 expired deliveries are pair#629's shape, not admission: one
  expiry exactly every 90 s from 17:37 to 18:22, from 11 senders, the hub wrapper holding its queue head for the whole
  composer wait each time.

## Revisions

- 2026-10-16 (pair:2): Spec and Done-when rewritten to the measured cause. Before: "find why the recovered wrapper is
  not a connected recipient (registration on reboot/resume vs. first turn)" and "a rebooted and a resumed slot each
  receive a send without a relaunch". The Log shows registration was fine and pair:3 received before and after the
  failures; the failures were admission timeouts under load. The relaunch and provisional items are kept; the receive
  item now targets admission under load and its visibility.
