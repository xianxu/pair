---
id: 000530
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '411f605eab881e1f5f90e422a45a35eeefb19028' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T20:35:26-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "971ba722", done: "a8801e87"}
---

# couch: score stale session files and stuck slots from activity Couch already sees (baseline: 15 min without a write → ping)

## Problem

Couch sees every event across slots (deliveries and receipts, slot state changes, landings, notices, session-file writes) but uses it only for binary rules: the TL's session file 'owes a response' ping fires on any unanswered delivery, mostly for progress updates that change nothing, while on 2026-10-12/13 three slots finished their work and waited for hours overnight because their reports never reached the hub and nothing noticed.

## Spec

The operator (2026-10-13): a simple local model (not an LLM), kept simple.
1. **Baseline rule (ship first):** a session file with no write for 15 minutes while its owner has unanswered deliveries gets one ping. Anything learned later must beat this baseline on the logged data.
2. **Feature log:** at every session-file write, record the feature vector (deliveries to the hub since the last write, by kind: landed / waiting-for-operator / progress; slots whose state changed; minutes since the last write; operator submitted since); that's a positive. Negatives: the vector one minute after each write, plus random moments not followed by a write within 10 minutes. Same log for slots: moments the hub or operator acted on a slot after silence (positive) vs sampled idle stretches (negative).
3. **Stuck-slot nudge:** a slot idle on an issue branch whose last hub message is unanswered for N minutes (or whose hub delivery failed) nudges the hub; unacted, then the operator. Start rule-based; same logging.
4. Later: fit logistic weights from the log, explainable ('pinged: 3 landings, 40 min since the last write'), switched on only if it beats the baseline.

## Done when

- A quiet settled worker with an unanswered report nudges the hub at 15 minutes
  and produces one operator Couch notice at 30 minutes, coalescing earlier wait
  notices; completion reports remain eligible after return to main.
- Each worker address receives one startup resend-prompt attempt per supervisor
  lifetime through normal Send. No reports are replayed across restarts, and
  admission or later terminal failures are visible without regenerating prompts.
- Generated prompts and waiting notices do not create session-file obligations
  or count as conversational replies. Saturation cannot silently disable report
  or reply observations. Tests cover rules, real socket/queue outcomes and the
  reconstructed overnight scenario; the full suite runs to completion.
- parley.nvim#321 owns the 15-minute file-ping baseline, and dependent pair#543
  owns fact logging with offline labels. Neither is claimed delivered by #530.

## Plan

Durable plan: [supervision baseline](../plans/000530-supervision-baseline-plan.md).

- [x] Obtain TL approval of the durable plan, then pass change-code.
- [x] Add pure rules for stuck reports, coalescing and restart prompts.
- [x] Wire existing supervision tick and normal Send startup prompts.
- [x] Verify reconstructed overnight scenario, queues, uncertainty and full make -k;
  document and hand off parley.nvim#321 and pair#543, then close and land through SDLC.

## Log

### 2026-10-13

- Claimed in pair:7; start-plan completed after its requested weave refresh.
- Read-only exploration found the ping in Parley's existing session_sync.lua
  probe: five minutes from oldest owed delivery, not latest file write. Filed
  and published parley.nvim#321 with deps pair#530; no Parley implementation.
- Couch cannot see a failed connection made while its supervisor is absent.
  Startup requests cover that blind spot without storing or replaying reports.
  Sessionsync owed means not folded into a file, not conversationally unanswered.
- Drafted the durable plan for TL review. No change-code or implementation yet.

## Revisions

- 2026-10-13 20:48 PDT — proposed defaults for TL review: stuck hub nudge at
  15 minutes and operator notice at 30, coalesced with existing wait notices.
  Explicit completion reports remain eligible after return to a resting branch;
  otherwise lost landing reports would disappear from supervision. “Unanswered”
  means no later hub message or operator submission observed, not a semantic
  claim. Pair delivers producer/supervision and hands off the baseline consumer
  to parley.nvim#321; overall baseline acceptance requires both halves.

- 2026-10-13 — operator correction, dispatched by TL in receipt
  bf36b43c-848f-4890-ac1c-028230440da0: Couch messages remain EPHEMERAL.
  Supersede the queued/redelivered-on-restart proposal below: add no offline or
  restart queue and never replay old reports. When Couch restarts and the hub
  returns, ask each connected slot once to reconsider and resend its last
  unanswered hub report only if still relevant; the slot decides. The prompt
  itself bypasses queueing and retries. TL clarified in receipt
  9afd9c9b-0949-45d1-9b98-f673f228f51f that existing #492 live busy-recipient
  queues and existing wait-notice retries remain unchanged.
- 2026-10-13 — ownership boundary: no attach/resume/slot-state changes (#528).
  Read existing activity, liveness and broker observations only. Existing
  session-file pings live in parley.nvim, so the baseline consumer belongs in
  companion parley.nvim#321 (depends on pair#530), not a second Couch watcher.
  Pair owns the observation/feature contract and hub supervision. Both halves
  are required to deliver the overall baseline; the companion must be assigned
  and verified before declaring the cross-repo behavior complete.

- 2026-10-13 — add: a hub-bound message that fails because the hub isn't live (e.g. during a Couch restart) is
  queued and redelivered when the hub returns, rather than the sender giving up (pair:1's #531 report failed this way at
  13:24; reports were lost the same way overnight 10-12). The stuck-slot nudge treats an undelivered hub message as a
  strong signal.

### 2026-10-13 20:53 PDT — TL-approved rules-only scope

Receipts ddcbf61f-f57a-4aa9-a740-f4614830f5c3 and
addd69f2-20e6-4697-8404-8fa5396f0d52 approve 15/30-minute defaults,
completion-after-main and observable-action wording. Startup prompts now use
normal Send, allowing #492's in-memory queue; one coordinator attempt per
address per supervisor lifetime, no persistent/offline replay. This supersedes
the earlier immediate/bypass-queue requirement. Fact logging moves to dependent
pair#543: event rows plus minute snapshots, labels offline, no online sampling.
#530 has one implementation/close boundary; land it before #543 starts.

Revised Done when: rules nudge/escalation and one-shot restart prompts pass the
reconstructed overnight scenario and transport/coalescing tests; full make -k
completes, operator notices are documented. File-ping acceptance is owned by
parley.nvim#321, and logging acceptance by pair#543, not prerequisites falsely
claimed as shipped by this rules-only change.


### Implementation evidence (2026-10-13)

- change-code admitted the current design (inferred quick; close upgrades the
  review for the actual larger diff). No review or validation bypass requested.
- Pure rules and private-socket integrations passed targeted tests: confirmed vs
  uncertain/queued reply, operator submission, failed/forged hub attempt,
  completion on main, coalescing, startup queue/reconnect/uncertain result, and
  generated-message provenance. Repository-qualified reports and mute behavior
  each failed first then passed with their implementation.
- Whole couchcmd and couchmessage packages passed before the final mute/docs
  additions; focused race run passed. Full make -k follows this checkpoint.
- Published dependent pair#543 with the simplified fact-log Spec; parley.nvim#321
  remains the sole file-ping consumer. Neither is implemented in this branch.

- TL accepted the private-socket/clock scenarios as this change's acceptance
  (receipt 018dd090-c2ec-45bc-8425-b9c1394204d9); no staging or live restart.
  The next natural Couch restart will expose the startup prompt for observation.
- Bare make -k invokes this repository's help target; the actual full run is
  make -k test, now queued by the machine gate. No suite result claimed yet.

- First full suite exposed a test synchronization bug: completion-after-main
  advanced its fake clock while the startup prompt was still Pending. Repeated
  instrumentation confirmed worker Pending=true at both fake ticks. The fixture
  now waits for the startup receipt and broker reservation to settle before
  advancing time; production Pending suppression remains unchanged.
- Added a red/green pure regression for an older connected snapshot racing a
  newly received report. Pruning now preserves reports newer than the tick's
  snapshot time; a later disconnected observation still removes them.

- Full make -k test completed (exit 2), /tmp/pair530-full-suite.log: its sole
  failing test was TestSupervisionIntegrationCompletionAfterMain's startup
  Pending race described above; all other targets/packages completed green.
  After the fix: all integration cases passed 50 repetitions (26.819s), focused
  race checks passed, and the entire couchcmd package passed (83.902s), logs
  /tmp/pair530-race-fixed.log and /tmp/pair530-couchcmd-fixed.log. No known
  baseline failures were present. Do not describe the original full run as green.
- Ready for the mandatory close review; TL accepted headless evidence, no stage.


### 2026-10-13 — close review round 1

REWORK BR-1/2/3/4 accepted. All generated notifications now share the same
provenance registration, including ordinary waiting notices. Pending-send
metadata is removed after terminal broker receipts; failed report attempts keep
only the latest per-worker episode, so no separate receipt-history capacity can
suppress report/reply observations. The existing tick reconciles startup prompt
cancellation, expiry, and uncertain outcomes into one readable notice, without
retrying. Normal Send and its queue policy remain unchanged. The primary Done
when now reflects the approved rules-only scope; prior expanded criteria remain
recorded in Spec and the earlier revisions, with #321/#543 as their owners.

- Review regression proof: each BR-1/2/3 test fails against pre-fix HEAD through
  a Go overlay, and passes on the fixed tree. Includes >256 valid authenticated
  sends refused at admission followed by a valid report/reply, all terminal
  prompt outcomes, and queued cancellation with no Deliver callback. Targeted
  rules/wait/broker tests and focused race tests pass. A fresh full make -k test
  follows the review fixes before the next close attempt.

- Post-review full `make -k test` completed with exit 0 on
  `751de842253cf1e76b478323d4719f016cee0763`; log:
  `/tmp/pair530-full-suite-review-fixed.log`. All targets completed, with no
  failures or baseline exclusions. This supersedes the earlier failing full
  run as final verification. Headless private-socket/clock acceptance remains
  approved by TL; no stage is required.

### 2026-10-13 — delivery provenance survives terminal receipts

Round 2 left BR-1 open: sender disconnect can finalize a receipt before the
endpoint returns. The lifetime rule is now explicit: generated provenance stays
until both the receipt is terminal and the broker delivery job has finished
endpoint accounting. The broker snapshot exposes that existing job fact
atomically with the receipt; no new queue or retention timer is introduced.
This covers startup prompts, rules nudges, and ordinary wait notices, including
the interval before endpoint entry. Controlled late-return tests reproduce all
three failures; a genuine hub message with identical prose still creates an
uncertain file obligation. Completed jobs release the metadata on reconciliation.
This supersedes the earlier assumption that a terminal receipt alone permits
cleanup (ARCH-ORDER, ARCH-FUNERAL).

- Focused supervision/wait/broker tests passed after the lifetime fix; focused
  race run also passed, logs `/tmp/pair530-lifetime-fix.log` and
  `/tmp/pair530-lifetime-race.log`. The three generated late-callback cases
  failed before the fix, while the genuine-message control passed. A full
  suite will verify this final code change before the next close.
