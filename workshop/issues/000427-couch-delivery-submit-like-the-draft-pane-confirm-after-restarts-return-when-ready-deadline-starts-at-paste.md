---
id: 000427
status: open
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: '75f9d1a1b47ff98c05c3ccb877f05573b22b1d44' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-10
