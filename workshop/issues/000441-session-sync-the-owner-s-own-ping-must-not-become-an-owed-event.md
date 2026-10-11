---
id: 000441
status: working
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '5745cd1b4a09631d83311408af3f06657f32f187' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T09:43:01-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "18568e1b", done: "56adb3f1"}
---

# session-sync: the owner's own ping must not become an owed event

## Problem

parley.nvim#314 pings the owner slot ("session-sync ping: <file> owes a response since …") when a session-sync file has been owed for 5 minutes. The ping is itself a delivery to the owner slot, and pair#437/#439's ledger records every delivery to the owner as an owed event. So the ping creates a new owed event, which the next ping episode answers with another ping: a self-sustaining loop whenever the agent doesn't fold (seen 10-11 09:41: the ping `ba7bbd61` from ops:0 to ops:0 appeared in `Owed`).

## Spec

A delivery whose sender is the owner slot itself and whose text is a session-sync ping (`session-sync ping:`) is not recorded as owed. Same identity rule the breaker uses (pair#433): sender identity, not text alone, so a peer can't suppress a real owed event by imitating the prefix. A submit (`submitted:`) from the owner's own editor stays owed: it's the operator's turn ending.

## Done when

- A ping from the owner slot leaves `Owed` unchanged (test).
- A `submitted:` from the owner slot and any delivery from another slot are still owed (tests).

## Plan

- [x] Exclude owner-sent pings in the ledger's record path; tests for the three cases.

## Log

### 2026-10-11
- The skip lives in `deliveryRecorder.record` (`cmd/internal/couchcmd/message_deliveries.go`), the one ledger write path; `Owed` stays a pure reader. Identity = `m.From == m.To` on the full binding, the breaker's own predicate (#433), not slot equality. `sessionsync.PingPrefix` sits beside `SubmitPrefix`.
- `TestTheOwnersOwnPingIsNotOwed`: own ping not recorded; own submit, peer ping, peer report recorded. Mutation: dropping the identity check makes `peer-ping` fail.
- `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees` fails with and without this change (pre-existing).

### 2026-10-11
