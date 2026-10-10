---
id: 000427
status: codecomplete
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: '839f3f7dfa28cf46b55594f38e44d9ef1a62d4df' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-10T11:51:07-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 1.10
---

# couch delivery: submit like the draft pane, confirm after; restarts return when ready; deadline starts at paste

## Problem

A TL dispatch sent seconds after `couch --reload-context ariadne:4` was pasted into the fresh Claude Code composer and never submitted. The receipt read `expired | delivery deadline elapsed: waiting for pasted envelope to render`, and the operator had to press Enter (10-10; ops learnings for ariadne-robustness-1, kink 43). Code read:
- `--reload-context` / `--relaunch` report success when the new wrapper session connects (`couchcmd/live_restart_probe.go:117`), not when the agent is ready.
- An exact-route send (`repo:N`) skips the readiness/quiet check (`couchmessage/broker.go:530-538`), and its 30s deadline starts at admission (`broker.go:564`). It ran down while Claude booted.
- Before submitting, the wrapper requires the rendered composer to match the paste (`wrapcmd/peer_composer.go:152-210`: exact, `[Pasted text #N]` marker, word-wrap projection). The early paste rendered as full text, the matcher never matched, and nothing retries.

Meanwhile the operator's draft pane pastes, sleeps 100ms and sends Alt+Enter (`nvim/draft_send.lua:28-65`), with no render matching, and it works. So does the startup orientation prompt.

## Spec

**Principle (operator, 10-10, as in pair#425): pair renders, never classifies.** A pre-submit match on the agent's rendered composer is agent- and version-specific classification. It broke for #418's collapsed paste, and now for a startup render, and it will break again.

1. **Deliver like the draft pane:** keep the existing pre-paste gates (no other automatic input, no recent operator keystroke, bracketed paste on, composer present), paste, wait a short fixed delay, submit. Drop the pre-submit render match.
2. **Confirm after the fact, agent-agnostically:** the submit counts as landed when the composer clears or a turn starts (output advances with a working signal). Otherwise the receipt reads `uncertain` with the evidence; never a silent `expired` with text left in the box.
3. **Restarts return when ready:** `--relaunch` and `--reload-context` report success only once the new session is settled (#421's Settled), not merely connected, with a bounded wait whose expiry is reported.
4. **The deadline starts at paste, not admission,** so a slot that is still booting doesn't burn the delivery window.

## Done when

- A send right after `couch --reload-context pair:N --confirm` submits and the agent starts a turn (live check on a real slot, plus a test that delivers during a simulated agent boot).
- A long multi-line message (the #418 case) still submits.
- A paste the agent never consumes yields `uncertain` with evidence, and no text is left unsubmitted silently.
- `atlas/couch.md` describes the new delivery and restart-readiness contract.

## Plan

Durable plan: `workshop/plans/000427-couch-delivery-submit-like-the-draft-pane-confirm-after-restarts-return-when-ready-deadline-starts-at-paste-plan.md`.

- [x] D1 reducer: paste → fixed delay → submit → confirm; drop the render match and its helpers; post-paste interrupts give `uncertain` (Indeterminate)
- [x] D2 deadline: `Message.Deadline` = paste-by (90s); the 30s window starts at paste; `Message.Horizon()` bounds the broker side
- [x] D3 restarts: Settled waits for orientation; relaunch/reload-context receipts wait for a settled new session (2m, `unready` on expiry)
- [x] Tests per the plan; live peer conformance against real Claude
- [x] atlas/couch.md delivery + restart-readiness contract

## Log

### 2026-10-10
- 2026-10-10: closed — Merged origin/main (#425): only conflict was workshop/lessons.md, both sides appended a lesson, kept both. Post-merge: go build ./... ok; couchmessage + wrapcmd Peer/Settle/Delivery/Automatic tests ok; couchcmd Restart/SlotOperation/LiveRestart/Peek tests ok. Prior evidence unchanged.; review verdict: SHIP
- 2026-10-10: closed — Post-close: added turn-already-running confirmation cases (queued behind a running turn → submitted on composer clear; running turn with text staying → indeterminate "a turn was already running"); wrapcmd TestPeerDeliverySubmitsAfterDelayThenConfirms passes 7/7. Plan Revisions records the pre-Build unready and 50ms poll deltas. Prior evidence unchanged (make -k test green with env scrubbed; go test ./... residual failures identical on main; live Claude 2.1.296 short+collapsed submitted+confirmed).; review verdict: SHIP
- 2026-10-10: closed — Targeted: couchmessage (reducer table, Horizon expiry), wrapcmd (submit-after-delay+confirm table, simulated boot past old 30s, never-consumed → indeterminate with evidence, collapsed multi-line, settle waits for orientation + re-arm with mutation check), couchcmd TestRestartReceiptWaitsForReadiness (relaunch/reload succeed only on a settled NEW session; unready on expiry; caught+fixed empty ReceiptCode overwrite). Live: TestPeerLiveConformance -peer-live-submit vs real Claude Code 2.1.296, short (render=exact) and collapsed #418 body (render=collapsed), both submitted+confirmed. Full: make -k test green with PAIR_/COUCH_ env scrubbed + scratch TMPDIR; go test ./... — remaining failures (artifactpath inventory, couchsingleton SelectionSize x2, gcruntime ArchiveLocator, couchcmd ColdResume switcher flake 5/10 on main) all fail identically on main. Pending: live couch --reload-context + send on a real slot needs Couch on the new binary (ops).; review verdict: SHIP
- 2026-10-10: flow upgraded quick → full — 266 added lines in code files (limit 100)

- Design: `workshop/plans/000427-…-plan.md`. "uncertain" is the existing
  `indeterminate` status with a detail that starts `uncertain:`. A new enum
  value would ripple through the broker model and CLI for nothing (ARCH-DRY).
- Restart readiness waits off the console queue (`awaitReadiness`), so a
  2-minute wait never blocks other slots' operations. It is bounded, and it
  is in-memory only (ARCH-FUNERAL: nothing durable).
- Found while testing: `slotOperationOutcome` let a completed relaunch's
  empty `ReceiptCode()` overwrite the error's code. That would have hidden
  `unready`. Now only a non-empty partial code overrides.
- Settled now also waits for the startup orientation to finalize. Finalizing
  re-arms the settle check (lessons: every writer of a derived state notifies
  it). A mutation check confirmed the test fails without the re-arm.
- Live: `TestPeerLiveConformance -peer-live-submit -peer-live-use-local-auth`
  against Claude Code 2.1.296. Short body: render=exact, submitted and
  confirmed. Collapsed long body (#418): render=collapsed, submitted and
  confirmed.

