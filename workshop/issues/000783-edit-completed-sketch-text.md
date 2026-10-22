---
id: 000783
status: open
deps: [pair#752]
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'f1a6868f2c112573493e79e3987e87c639d24928' # card fields mirrored from issue-cards; edit via sdlc
---

# Edit completed sketch text

## Problem

The operator wants a Note button on a finished sketch to open and edit attached text. #752 can expose the unfinished label editor, but completed labels are immutable in the existing Go/JS reducers and validated receipt history. TL approved splitting completed editing from #752.

## Spec

Design explicit sketch-text revision events with author permission, durable ordered receipts, replay/seek/export semantics, length/history budgets, and Go/JS/snapshot parity. Reuse existing note dialog affordances where suitable while preserving sketch identity and anchors. Never mutate a completed label outside the event stream.

## Done when

- Completed sketch text can be opened and edited through a visible Note button with clear save/cancel and refusal/unknown recovery.
- Ordered revision events replay identically in Go/JS and snapshots; original label history remains intact and unauthorized viewers cannot edit.
- Persistence compatibility and bounded storage are explicitly designed and reviewed before implementation.

## Plan

- [ ]

## Log

2026-10-21: unclaimed follow-up requested by TL72ca720e-748c-40b0-9b66-45ade56d934a during #752 live smoke. Current sketch_state.go rejects second labels; browser reducer and snapshot receipt validation enforce the same finality. Existing bounding-box notes use separate identities and cannot silently overwrite sketch labels.
