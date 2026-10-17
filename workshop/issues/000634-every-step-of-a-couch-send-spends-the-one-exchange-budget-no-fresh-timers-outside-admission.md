---
id: 000634
status: working
deps: [pair#618]
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
card_mirror: '350a913e42eedeb3f813254567ec1e2bbcbf430b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T04:42:51-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "2113fcef", done: "82ba6caa"}
---

# every step of a couch send spends the one exchange budget; no fresh timers outside admission

## Problem

A `couch --send-to` exchange has one budget, `SendTransportTimeout` (`AdmissionTimeout` plus 1 s), on both ends
(pair#618). Two steps of the send spend time outside the broker's `AdmissionTimeout`:

- the sender's authority check (`messageAuthority.current`, `message_service.go` around line 851), which runs before
  `Broker.Send` starts its own fresh `AdmissionTimeout`;
- `refusalFamilies` (`couchmessage/refusal.go`), which, after a failed admission, starts its own 250 ms timer from the
  broker context and repeats the families read `admit` already did.

The margin holds today: 2.25 s of broker time against 3 s. But a slow sender check moves the broker's verdict past the
exchange deadline, and the CLI then drops it. That is pair#618's original symptom: a refusal reason that never
reaches the sender. Found by pair#618's close review (BR-7).

Optional, same files: `serve` decodes each request twice (`requestBudget` and `requestIsRecoveryPreview`), and `Call`
marshals twice (BR-8). Fold that in only if the budget change touches those lines.

## Spec

- Every step of a send derives its deadline from the one exchange deadline: the sender check, admission and refusal
  bookkeeping. None starts a fresh timer.
- `admit` returns the families it read, so the refusal reuses them instead of reading again.

## Done when

- A test with a sender check that takes most of `AdmissionTimeout` still gets the broker's verdict, a refusal with its
  reason, to the CLI inside `SendTransportTimeout`.
- No `context.WithTimeout` in the send path starts from a context other than the exchange's.

## Plan

- [x] Measure the dropped verdict over real service sockets with a 1.5 s sender check followed by a recipient waiting for admission cancellation; retain the failing regression before production edits.
- [x] Start the existing admission budget before the sender check, as a child of the exchange context, and carry it into Send; return admit's families snapshot for refusal bookkeeping instead of reading again (ARCH-DRY).
- [x] Regress shared deadlines, cancellation, one families read and canonical refusal reasons; run targeted race checks and the recorded full suite.
- [x] Update the atlas and prepare acceptance evidence and deferred live-check steps.

After these implementation checks: close through fresh review, land through normal gates and report to the hub. Operator restart check is deferred with #629/#631 by TL approval, with no pair:0 staging.

## Log

### 2026-10-16

### 2026-10-17 — design

Claimed in pair:3 and started from fresh main. Inspection confirms sender authority precedes Broker.Send's new 2 s timer, and refusalFamilies starts an independent 250 ms context from broker lifetime. Share the existing admission context across authority and broker, preserving the transport's existing 1 s response margin; reuse the admission families snapshot. Increasing the transport timeout leaves the additive-budget defect; an extra deadline helper adds an owner unnecessarily. No new state, retry, persisted data or timeout value. Optional BR-8 decoding/marshalling cleanup is outside this change. TL approved this plan and deferring live validation to the operator restart alongside #629/#631 (delivery 4e1cd205-fd63-47c0-942d-41435a3c860d). The real-socket regression injects delay in the existing authority seam; no operator environment is touched.

Before production edits, TestMessageSendSlowSenderStillReturnsRefusal failed over the real service socket: a 1.5 s authority delay followed by recipient Observe waiting on cancellation exhausted the CLI exchange in 3.001686625 s; response was empty and Call returned context deadline exceeded, dropping the broker verdict. Evidence: /tmp/pair634-send-budget-red.log. The separate routing-snapshot regression also exercises refusal lookup duplication and canonical target drift, including an expired initial lookup.

Families regression failed on the old code with two reads both on ordinary refusal and initial lookup timeout; a changed second snapshot rewrote p:3 to other:3. After the fix it reads once and keeps pair:3. The real-socket regression passed under race at 2.00257475 s with a definite refused response and timeout reason, and exact equality of sender/recipient deadlines (/tmp/pair634-send-budget-green.log).

The send-path sweep found targeted registry posting waited only on service lifetime. TL approved including caller cancellation (delivery 493d236c-3a26-4761-9978-0bab4a5ba4a1). With the registry loop deliberately absent, the new regression failed on the old post after its deadline-derived 4 s guard (/tmp/pair634-registry-red.log); the fixture then released and joined the blocked request. postContext now bounds both post/reply waits by caller and service lifetime; send returns a definite pre-admission refusal if posting fails. Existing registry callers still use service lifetime. Targeted race regressions passed, including sender deadline, stalled registry, refusal snapshot, lookup timeout/cancellation, hub routing and dormant-target re-admission (/tmp/pair634-all-regressions-green.log).

Broader race runs passed couchcmd 108.151 s and couchmessage 27.457 s (/tmp/pair634-messaging-race.log); that run preceded the registry-post delta, whose targeted race regression passed subsequently. Declaration-comment checks passed (0.358 s). Source head f94a9ba28e70b06f3db6e6cce431641ac0272667 is pushed; required recorded suite queued as run 9f4aada897a31bf454d5180e8f211f0e, without cancelling other holders. TL approved an early draft PR to overlap macOS CI with this wait (delivery 0423ad3e-8d82-49e3-815e-5c30149c2d9e); fresh close review and all normal landing gates remain required.

Draft PR https://github.com/xianxu/pair/pull/397 opened. The final source passed all TestMessage* cases under race (12.605 s; /tmp/pair634-final-message-race.log), including the registry-post change. GitHub merge-check and local-gates-guard passed; native macOS conformance did not trigger because its path filters exclude these message-path files. Required local suite remains queued; no flaky failure or retry so far. Live-check deferral remains explicit, not a completed observation.

Required recorded full suite passed, exit 0 in 13m44s, after 889 s queue wait: source f94a9ba28e70b06f3db6e6cce431641ac0272667, run 9f4aada897a31bf454d5180e8f211f0e, log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/f94a9ba28e70b06f3db6e6cce431641ac0272667/suite.log, SHA256 prefix b045883b058f. couchcmd 90.910 s, couchmessage 26.765 s, wrapcmd 66.704 s, couchcore 448.351 s; no failed attempt or flake retry. Only issue documentation follows this tested source head.

Deferred operator check after restarting onto the landed build: from a connected worker, send a harmless test message to the busy hub; the CLI must return its message ID and admission/queued verdict with no transport uncertainty. Query that ID while the hub's composer is occupied to see held/queued with its composer wait and queue position, then confirm submission after the composer returns. A pre-admission refusal must reach the CLI with its reason. Deterministic slow-authority and stalled-registry cases remain covered by the private real-socket/controlled-loop regressions, rather than delaying the operator's live service.

## Revisions

### 2026-10-17 — include targeted registry posting

Inspection found an additional send step that could ignore the request deadline. Within the original every-step contract, carry the existing caller context into targeted event posting, retain service cancellation, and regress a stalled loop. No new state, retry or timeout; TL approved before this implementation delta.

### 2026-10-17 — separate completed verification from landing actions

Split the last Plan row into completed atlas/evidence work and the subsequent SDLC gate actions, so checking implementation complete does not claim that review, landing or the deferred operator check already happened.
