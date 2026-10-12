---
id: 000465
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '84162a057c4b98849178311c128b9c260837af37' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: a slot blocked by a picker shows red in the tab bar, notifies the hub with the picker's text, and receipts say so

## Problem

When a slot's agent shows a picker that holds input (a permission prompt, the auto-mode classifier's question, an agent's question or plan approval), Couch correctly holds messages back, but nobody learns of it: the hub gets a generic waiting notice only after 10 minutes, the operator sees nothing, and a held message expires as `delivery deadline elapsed: waiting for picker to close`. 10-11/12: pair:2 sat on a question picker twice, and pair:3 was unreachable for an hour behind a latch armed by a permission prompt that auto-denied (pair#464).

## Spec

Couch reports facts; the hub and the operator decide (the operator's principle).
- **Red in the tab bar** (the operator, 10-12): while a slot is behind a picker that holds input, its tab-bar entry renders red, so the operator sees it at a glance. It clears when the picker closes.
- **Hub notice, early and specific:** after about a minute behind a picker (not 10), one notice per episode to the hub with what the picker is: its kind as far as the screen tells (permission prompt vs question), its title or command, and its options. Same mute file as pair#451's waiting notices.
- **Receipts say why:** a message held by a picker stays queued while the picker is up and reports `blocked by a picker: <title>` in `--message-status`, instead of expiring silently at the deadline; delivered when the picker closes, or expired with that reason.
- Uses pair#464's corrected picker state (the composer coming back closes it).
- Not in scope: answering pickers from the hub (a possible later verb for question pickers only; permission prompts stay the operator's).

## Done when

- A slot behind a picker turns red in the tab bar within seconds and clears when it closes (tests on recorded screens; operator smoke test).
- The hub gets one specific notice per episode after about a minute, with the picker's text.
- `--message-status` for a held message names the picker; delivery happens when it closes.
- The operator smoke-tested it before close.

## Plan

- [ ]

## Log

### 2026-10-12
