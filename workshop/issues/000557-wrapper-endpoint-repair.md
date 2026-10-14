---
id: 000557
status: open
deps: [pair#528]
github_issue:
target: slot-continuity
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: '05c0eca490a3ff480c1460d24314d1ed41d7bceb' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T15:11:19-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Pair wrappers recreate their own missing message endpoint

## Problem

Follow-up to #528's accepted M1. Live wrappers reconnect to Couch's broker
independently of panes, but an unlinked own message endpoint cannot currently be
recreated by older wrappers. A functioning agent must remain Continue by terminal;
a broken channel never authorizes restarting its child.

## Spec

New wrappers recreate only their own missing message endpoint and re-register
the same authenticated binding, without restarting the agent, replaying input or
creating a launch-ledger entry. Preserve receiver state. Reuse the wrapper's
existing runtime lifetime, bounded retry/cancellation, endpoint inode/owner checks
and broker admission proofs. Do not add durable messaging or a persistent repair
service. Never remove or take over a foreign endpoint. Older wrappers retain the
truthful messaging-unavailable-until-relaunch notice.

## Done when

- Missing-own-listener, initial bootstrap failure, broker restart, foreign endpoint
  collision and lost registration reply have stateful regression coverage.
- Recovery preserves exact agent PID/native session and receiver state, produces
  no input replay/child restart/launch-ledger row, and ends on runtime cancellation.
- Conflicting or uncertain ownership never removes another endpoint; retry and
  failure remain bounded and visible without changing the slot's Continue outcome.
- Targeted and recorded full tests plus boundary review pass; TL schedules a new
  scratch wrapper acceptance: remove its own endpoint, observe recreation and
  authenticated messaging return with unchanged child/session. Contrast the old
  wrapper notice. Never delete a working slot's endpoint for testing.

## Plan

- [ ] Inspect existing endpoint/receiver owners and refine the #528 M3 design
  before implementation; preserve its independent operator acceptance boundary.

## Log

### 2026-10-14

Split from #528 at operator/TL direction84fbcea7. Source: #528 revised M3;
M1 broker reconnection is already implemented, endpoint recreation is not.
