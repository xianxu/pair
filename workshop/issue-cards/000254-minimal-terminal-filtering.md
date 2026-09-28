---
id: 000254
status: open
created: 2026-09-14
updated: 2026-09-14
estimate_hours:
github_issue:
---

# Minimize terminal escape filtering

## Problem

Codex scrolling has remained inconsistent, and the operator reports flashing
and screen artifacts. Pair currently strips Codex synchronized-update controls
(ESC[?2026h/l) and focus-reporting controls (ESC[?1004h/l) by default. An optional
Codex filter strips specific keyboard negotiation sequences. Claude bypasses
these agent-specific output filters. Couch separately strips Ctrl from vertical
mouse-wheel reports for every agent to prevent Zellij pane resizing.

Screenshot-local investigation in #252 found valid source UTF-8 and confirmed
that the running wrapper stripped synchronized-update boundaries during repeated
redraws. This is a plausible flashing contributor, not an established cause.
The confirmed downstream UTF-8/control-injection defect remains owned by #252.

