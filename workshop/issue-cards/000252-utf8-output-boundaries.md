---
id: 000252
status: open
created: 2026-09-14
updated: 2026-09-14
estimate_hours:
github_issue:
---

# Preserve UTF-8 across terminal output chunks

## Problem

Operator screenshots show replacement glyphs in Codex transcript separators and
the surrounding Zellij pane border, plus stray dots in the composer. The source
Codex raw capture is valid UTF-8; the saved ANSI transcript has no replacement
characters. Corruption therefore occurs downstream of that capture.

Confirmed defect: `ptychild.Screen.MidSequence` tracks incomplete escape
sequences but not incomplete UTF-8 characters. Couch's `writeChild` uses that
answer to append keyboard-disambiguation controls after each output chunk. A
chunk ending inside the three-byte `─` character is treated as complete; the
injected control sequence separates its bytes. `SafeToPaint` also incorrectly
returns true at this boundary. This affects the whole Zellij output, explaining
why both agent content and pane borders can be affected.

Whether the stray composer dots share this cause remains unconfirmed.
