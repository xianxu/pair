---
id: 000859
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
card_mirror: 'f09da55fc848a5699b7ff0406f28c1664fc313a2' # card fields mirrored from issue-cards; edit via sdlc
---

# Terminal cast: missing sketch submission path

## Problem

While planning #842, source tracing found that Shift+Ctrl+Alt+Return enters nvim PairSendDOM, whose server transaction selects only the Chrome cast. The terminal LIVE group can count NoteBoard notes, but no terminal sketch selection/submission adapter exists. A Send button cannot honestly promise to send terminal-cast feedback through this transaction. This is a missing path, not evidence of an existing terminal sketch being lost.

Evidence on #842 baseline bbee4ced16c388ce4643a525585a2100643c35a5: nvim/init.lua PairSendDOM and nvim/dom_submission.lua own the draft action; cmd/internal/couchtty/console_submission.go submissionCast returns wcast.cast only; console_notes.go uses broadcast.NoteBoard separately. No matching terminal sketch submission implementation was found in that audit.

## Spec

Define whether and how terminal-cast sketches should participate in the draft feedback transaction. Reuse the transaction's snapshot, request identity, acknowledgement and clearing guarantees; do not make terminal notes eligible by counting them. Product scope and design require operator approval before implementation. Keep #842 limited to its existing hotkey behavior.

## Done when

- The supported terminal feedback types and their transaction semantics are explicit and approved.
- For supported types, deterministic tests prove draft plus exact feedback delivery and acknowledgement; unsupported feedback is never advertised as sendable.
- The LIVE Send control reflects actual eligible feedback, with a real-cast acceptance check.

## Plan

- [ ]

## Log

2026-10-23: Filed unclaimed at hub6491d0e0's request as a separate missing terminal-sketch submission path. #842 does not implement it.
