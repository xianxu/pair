---
id: '000025'
status: done
created: 2026-05-28
updated: 2026-05-28
actual_hours: 0.7
---

# Suspend Enter remap while a codex picker is open

## Problem

`pair-wrap` remaps the user's plain Enter (`\r`) to `\n` for the `codex`
agent (`cmd/pair-wrap/main.go:130-138`). That's correct in codex's
chat textarea — codex treats `\n` as "insert newline" / Shift+Enter
and `\r` as "send". The remap keeps Enter consistent across panes
(pair's nvim draft uses Enter = newline, Alt+Enter = send).

But codex also has **blocking pickers** (e.g. the resume-cwd prompt
that appears when the resumed session's cwd differs from the current
cwd; the "Choose working directory to resume this session" overlay).
In picker context, codex reads `\n` as "down" and `\r` as "select".
With the remap engaged, pressing Enter navigates down instead of
confirming the highlighted option. The user has to know to press
Alt+Enter to select — a confusing UX regression vs. running codex
outside pair.

Mirror of the claude story in #000023: same root cause (per-textarea
remap engaged during a non-textarea overlay), different agent.
