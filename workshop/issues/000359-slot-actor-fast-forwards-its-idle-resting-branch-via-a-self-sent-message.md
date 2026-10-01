---
id: 000359
status: open
deps: [pair#353]
github_issue:
created: 2026-09-30
updated: 2026-09-30
estimate_hours:
card_mirror: 'ce37a379471c0e61b5497c611f8db58f3ac950f3' # card fields mirrored from issue-cards; edit via sdlc
---

# Slot actor fast-forwards its idle resting branch via a self-sent message

## Problem

An idle slot's resting branch (`main` for :0, `main-slotN` for slot N) falls
behind `origin` until someone pulls by hand, and work started from it starts
on a stale base. On 2026-09-30, #358 branched from a `main-slot1` that was 53
commits behind `main`, missing #355's Pair registration. Built into pair:0, it
could not register with a couch built from `main`, and pair:0 would not start.

The slot already knows when it is free. It should keep its own checkout fresh
while it is free, so new work starts from a current base.

## Spec

This is the first **actor-triggered automation**: maintenance that a slot's
couch actor performs on its own slot. Later maintenance kinds use the same
path.

- **Slot actor event loop.** Each slot's actor processes its messages one at a
  time, in an event loop over the per-slot queue that #353 introduces for
  machine messages. While it handles a message, the actor is busy, and other
  messages for that slot wait.
- **Maintenance is a message, not a side effect.** When the actor sees that its
  slot is free, it does not act directly. It sends itself a message (first
  kind: `freshen`) through the same queue. The queue is what serializes access
  to the slot: a #353 dispatch that arrives during a pull waits for it, and a
  pull never starts in the middle of a dispatch.
- **A slot with queued messages is busy.** Free requires an empty queue, in
  addition to the agent being idle: a queued message is work the slot must
  still process. The message being handled no longer counts, since the loop
  has taken it off the queue.
- **Busy for the whole message.** The actor stays busy from the start of
  `freshen` until it finishes, including the network fetch. The slot does not
  read as free, and does not accept dispatch, while it runs.
- **Conditions are checked both when the message is sent and when it runs.**
  Checking at send keeps a slot that is doing other work from collecting
  `freshen` messages; because a queued `freshen` makes the queue non-empty, the
  slot also stops reading as free once it has sent one. Checking again at run
  covers what changed in between. `freshen` pulls only if all hold:
  - the slot is free: idle, and nothing else is queued behind `freshen` (a
    dispatch that arrived meanwhile wins, and the pull is skipped);
  - the checkout is on its resting branch (`couchcore.RestingBranch(n)`), with
    an upstream and a clean working tree;
  - after a fetch, it is behind its upstream and not ahead, so a fast-forward
    is possible. `SlotGitStatus.Behind` is only as fresh as the last fetch
    (`couchcore/slotgit.go`), so the check needs a fetch first.

  If they all hold, it runs `git pull --ff-only`. `--ff-only` keeps a race
  between the check and the pull from producing a merge. Otherwise it does
  nothing.
- **Duplicates collapse.** Repeated `freshen` requests for one slot collapse to
  one, which is the collapse-by-kind rule `couchcore.Enqueue`
  (`mailbox.go`) already implements. Maintenance messages are never `Control`.
- **Failure is quiet but recorded.** A failed fetch or pull (network,
  credentials, a non-fast-forward) leaves the checkout as it was. It is
  recorded where the slot's state can be inspected, not raised as an operator
  interrupt.
- **Open to more kinds.** Maintenance message kinds form a small registry, so a
  later kind adds a handler, not a new loop.
- **Nothing durable.** Messages live in the actor's memory and die with it; the
  only lasting effect is the fast-forwarded checkout.

Questions to settle at design:

- **What counts as free?** An empty message queue is required (decided). The
  idle-agent part should be the same predicate #353 uses to pick a free slot
  for dispatch, so both features agree.
- **When does the actor send `freshen`?** Proposed: whenever the slot becomes
  free. Decide whether it also re-checks while the slot stays free, and how
  often, given each run costs a network fetch.
- **Binaries after a pull in :0.** pair:0 runs live from its checkout's `bin/`,
  which a pull leaves stale until `make build`. Decide whether `freshen` in :0
  rebuilds, flags the stale build, or leaves it to the operator.

## Done when

- A free slot whose clean resting branch is behind its upstream and not ahead
  is fast-forwarded by a `freshen` message the actor sent itself, with no
  operator action.
- No pull happens when the slot is busy, has other messages queued, is off its
  resting branch, dirty, without an upstream, ahead, or diverged. A test covers
  each case.
- A slot with any queued message reads as busy, and the actor sends no
  `freshen` while it is busy.
- While `freshen` runs, the slot reads as busy, and a dispatch that arrives
  then is handled after it finishes, not alongside it. A test pins the order.
- Several `freshen` requests queued together run once.
- A failed fetch or pull leaves the checkout unchanged and is recorded; the
  slot returns to free.

## Plan

- [ ] Settle the three design questions.
- [ ] Slot actor event loop over #353's per-slot queue, with a busy state.
- [ ] `freshen` handler: check conditions, fetch, `git pull --ff-only`.
- [ ] Self-send `freshen` when the slot becomes free.
- [ ] Tests against a stateful git fake (or temp repositories), including the
      dispatch-during-pull order.

## Log

### 2026-09-30

- Filed at the operator's request, depending on #353 (per-slot machine-message
  queue). Prompted by the #358 slot-staleness incident described in Problem.
- Existing pieces: `couchcore.Enqueue` (`mailbox.go`, #145) has collapse by
  kind and `Control` priority but no production caller yet; `couchtty`'s
  `operationQueue` serializes console operations, but globally, not per slot;
  `couchcore/slotgit.go` reads ahead/behind/dirty and names resting branches.
- Operator review: conditions are checked at send as well as at run, so a busy
  slot doesn't accumulate `freshen` messages; and a non-empty message queue
  makes a slot busy, because each queued message is work still to process.

