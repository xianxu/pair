---
id: 000433
status: done
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-11
estimate_hours:
card_mirror: 'e92a002366cf41ab8391491d309f8ac089fb5d96' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-10T23:14:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "c6fb6451", done: "640d27d9"}
actual_hours: 1.12
---

# Couch peer-message breaker: a leaky bucket, a full refill on session-sync submit, and a higher allowance for the TL slot

## Problem

Couch gives each slot `InboundAllowance = 8` peer messages, refilled only by an operator submit in that slot's
agent pane (`cmd/internal/couchmessage/model.go`). It's a loop breaker: two agents can't ping-pong without a
human. But the TL slot (ops:0 in the ariadne-robustness-1 run) is the fleet's hub: every worker reports to it,
and the operator mostly reads there and submits through the session-sync status file (parley.nvim#313), which
arrives as a peer message and so spends allowance instead of refilling it. On 10-10 at 19:12, pair:1's landing
report for pair#324 was refused with `breaker-exhausted: recipient peer-message allowance exhausted`; the TL
learned of the landing two hours later (ops learnings kink 57).

## Spec

- **A leaky bucket instead of a count.** Burst of 8 as today; one message's worth refills every few minutes
  (pick the interval from the data: worker reports arrive minutes apart, a runaway loop seconds apart). A loop
  drains it in seconds and is cut off; spaced reports never exhaust it.
- **Session-sync submit refills fully.** A message whose sender is the recipient's own operator through
  session-sync (parley's Alt+Return submit) counts as `OperatorSubmit`, not as an admitted peer message.
- **The TL slot gets a much larger bucket.** Convention (operator, 10-10): the slot Couch was started from is
  the TL slot. Settle in the plan whether that's observable or needs an explicit `--hub repo:N` override.
- Refusals stay visible to the sender as today; `--list` or the receipt shows the recipient's remaining
  allowance, so a TL can see a hub running low.

## Done when

- [x] A burst beyond the allowance is refused; after the refill interval, one more is admitted (model test).
- [x] A session-sync submit refills the recipient's allowance fully (test).
- [x] The TL slot admits well past the ordinary burst without an operator keystroke (test).
- [x] A two-agent ping-pong is still cut off within seconds (test).

## Plan

Quick flow (one close). Decisions:
- **Bucket in the pure model.** `ActorState` keeps `remaining`, `capacity` and `refilledAt`; every event first
  credits `floor(elapsed / RefillInterval)` (capped at capacity; the clock restarts when full). `RemainingAt(now)`
  is the same arithmetic for listings, which have no event. `RefillInterval = 3m`: a runaway exchange of exact
  sends arrives seconds apart and empties the 8-burst long before a refill, while worker reports minutes apart
  never drain it.
- **Session-sync submit = a self-addressed message.** Parley shells `couch --send-to <owner>` from the operator's
  editor, which inherits the slot's launch env, so the broker resolves the caller to the recipient's own binding
  (`From == To`). The model treats a self-addressed admission as `OperatorSubmit`: full refill, nothing spent.
  Residual: an agent messaging itself also refills. That is a one-agent continuation, not a cross-slot loop; the
  two-agent ping-pong this breaker exists for is unaffected.
- **TL slot is observable, no flag.** Couch resolves its startup cwd to a worktree root; the actor whose workspace
  `WorktreeRoot` equals it registers with `HubAllowance = 64` (`Register` carries the allowance). A ping-pong with
  the hub is still cut by the worker's 8. Couch started outside a checkout has no hub.
- **Visible.** `couch --list` prints `allowance=N/CAP` from `Candidate.Capacity`.

- [x] model: bucket + self-send refill + register allowance; tests for each Done-when row
- [x] broker/service: register with an allowance, hub root on `messageAuthority`, listing shows remaining/capacity
- [x] atlas note + couch skill text; targeted tests, then full `make test`

## Log


- 2026-10-11: closed — model tests per Done-when + broker-path TestSelfAddressedSendRefillsThroughTheBroker (mutation-checked) + service TestMessageHubSlotGetsTheLargerAllowance + TestHubWorktreeIsTheStartupCheckout pass; couchmessage ok; couchcmd only TestColdResumeOfAParkedPrimaryRegistersFromBothOrigins + TestContinuationWriterPublishesExactCheckpointAcrossWorktrees fail, identically with the diff stashed; after merging main (#435) make -k test green except test-pair-embedded-runtime (fails identically on origin/main: PAIR_DATA_DIR scope env) and test-changelog (passes with scratchpad TMPDIR per #360); review verdict: SHIP
### 2026-10-10

- Dispatched by ops:0. Overlap with #435 (deadline vs holds in the same `model.go`): this change touches only the
  budget fields and the Admit spend, not the deadline/horizon logic.
- Session-sync provenance: parley.nvim `session_sync.lua` runs `couch --send-to <owner>` from the editor; the slot's
  launch env (`PAIR_LAUNCH_NONCE` etc.) is session-wide, so the caller resolves to the recipient's own binding.
  Telling the editor from the agent would need a process-ancestry check against `Binding.PID`; not worth it for a
  one-agent self-continuation (see Plan).
- `go test ./cmd/internal/couchcmd/`: `TestColdResumeOfAParkedPrimaryRegistersFromBothOrigins` and
  `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees` fail identically with this diff stashed
  (pre-existing).
- TL (ops:0) relayed parley.nvim#314: session-sync will also send `session-sync ping: <path> owes a response
  since <ts>`. No marker needed: both kinds are self-addressed when sent from the owner slot's session, so both
  refill (a ping refilling is harmless; only that session can send it). Text prefixes are not trusted. Replied
  to ops:0.
- Merged main after #435 landed: no conflict, builds, couch tests unchanged.
- Close round 1 FIX-THEN-SHIP: BR-1 (self-send refill only proven in the model) → added
  `TestSelfAddressedSendRefillsThroughTheBroker`, mutation-checked (fails with the rule disabled). BR-2 (hub glue
  untested) → extracted `hubWorktree(getwd, resolve)` + `TestHubWorktreeIsTheStartupCheckout` (checkout, subdir,
  outside, no cwd). Minor (self-messaging evasion) → couch skill text forbids it. Lessons added.
