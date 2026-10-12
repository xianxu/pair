---
id: 000492
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '2cb7e780e8a24826f9b4522a52b246c6af6868bf' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T11:23:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "b4dd6b99", done: "4efa8f08"}
---

# couch: the hub refuses messages while one is pending ('recipient has a pending message'); queue them instead

## Problem

10-12 ~10:47–11:04: pair:3 tried for about 17 minutes to send its #488 plan to `hub` (ops:0) and was refused every time: `couch: refused: recipient has a pending message`. With one hub and six or more slots reporting to it (now the constitution's rule, ariadne#336), a one-pending-message limit per recipient makes the hub a bottleneck: whenever the TL is busy, every other slot's report or question is refused, and each slot has to keep retrying or give up. pair:3 gave up and left its plan on screen.

## Spec

Find where the one-pending limit lives (per recipient, per sender pair, or the hub's admission) and why it exists (ordering? avoiding typing over a draft?). Then let a recipient, at least the hub, queue several pending messages and deliver them in order as its turns end, within the existing per-sender rate limits and capacity (the hub's bucket is already 64). The sender gets `queued` with a position, not `refused`. Keep any genuine protection (never typing into a picker or a draft), but as holding, not refusing.

## Done when

- Several slots can send to the hub while it is mid-turn; each gets `queued`, and all are delivered in order once it settles.
- Tests for ordering and for holding behind a picker/draft; the 10-12 case reproduced and fixed.

## Plan

Durable plan: `workshop/plans/000492-…-plan.md` (Tasks 1-4, then verification).

- [x] Task 1: model queue (pure `Advance`: Admit queues, Promote, Cancel, Tick expiry)
- [x] Task 2: exact routing resolves a busy recipient (family routing unchanged)
- [x] Task 3: broker queues at admission, worker promotes in order, holding on a busy wrapper
- [x] Task 4: CLI prints the position; skill, README, atlas

## Log

### 2026-10-12
Why the limit exists: one paste at a time. `ActorState.pending` is the one delivery
(model.go), `routing.go` refuses an exact target that is `Pending`, and the wrapper's
`reserve` (wrapcmd/peer_delivery.go) admits one reservation and refuses when the operator's
input moved its sequence. A held delivery (draft, scrolled pane, picker) can occupy the one
slot for up to MaxHold (30 min, pair#435), so every other sender is refused meanwhile: the
10-12 case. Keep the serialization (the protection); queue admissions behind it (the fix).
TL (ops:0) approved the plan with one ask, recorded in the plan's Revisions: a queued
message whose recipient disconnects or relaunches ends `cancelled` with
"<slot> disconnected (relaunched or exited) before delivery; resend", shown by
`--message-status`. The CLI already prints a receipt's detail under its status, so it
needed only a test.
Found while wiring: the wrapper's busy refusal crossed the socket as a plain string
(`RemoteEndpoint.call`), so the admission fallback could never see it. It is now mapped
to `ErrRecipientBusy`, as the other sentinel errors are, with a test.
The broker applies no `Tick` in production (expiry ran on the delivery job's timer), so
`promote` applies a Tick before reading the head; that is where `QueueWait` expiry runs.
Close review round 1 (REWORK). BR-1 Critical: an admission attempt that blocked a promotion and
then failed cleared the reservation without waking the worker, which stranded the queue. The
class is any path that frees the actor without a wake. Fixed at its root: every reservation
release goes through `releaseReservationLocked`, which wakes the worker whenever messages wait,
and the already-committed path wakes it to move on. BR-2: `TestMessageQueuedBehindAFailingAttemptIsDelivered`
holds an attempt in flight with a gated Observe, waits on `blockedPromotions` (no timing),
then fails the attempt. Mutation check: with the wake removed, the test fails (stranded until
its deadline); with it, the package passes under -race. Minor (untested already-committed path):
`TestPromotionAdoptsAnAlreadyCommittedMessage`. Minor (FIFO): an attempt already in flight
that ends busy joins the queue behind senders that arrived while it ran. Order is the order
admissions complete, the same as two concurrent sends before this change. Kept, and stated here.

