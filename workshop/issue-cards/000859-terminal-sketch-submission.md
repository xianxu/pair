---
id: 000859
status: open
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
---

# Terminal cast: missing sketch submission path

## Problem

While planning #842, source tracing found that Shift+Ctrl+Alt+Return enters nvim PairSendDOM, whose server transaction selects only the Chrome cast. The terminal LIVE group can count NoteBoard notes, but no terminal sketch selection/submission adapter exists. A Send button cannot honestly promise to send terminal-cast feedback through this transaction. This is a missing path, not evidence of an existing terminal sketch being lost.

Evidence on #842 baseline bbee4ced16c388ce4643a525585a2100643c35a5: nvim/init.lua PairSendDOM and nvim/dom_submission.lua own the draft action; cmd/internal/couchtty/console_submission.go submissionCast returns wcast.cast only; console_notes.go uses broadcast.NoteBoard separately. No matching terminal sketch submission implementation was found in that audit.
