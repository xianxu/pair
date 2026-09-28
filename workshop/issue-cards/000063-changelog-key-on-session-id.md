---
id: '000063'
status: done
created: 2026-06-16
updated: 2026-06-17
estimate_hours: 2
actual_hours: 0.66
---

# key the changelog on session_id so a fresh session starts a fresh changelog

## Problem

The change log (#53, `Alt+l`) is keyed on `changelog-<tag>-<agent>`. A pair tag
outlives individual coding sessions — `Alt+Shift+N` restarts a **fresh
conversation** under the same tag — so the per-tag log accretes every session
forever, with no way to start from scratch.

Mechanism (verified): the distiller anchors incrementally on the **last 3
verbatim cleaned scrollback lines** + a turn count (`cmd/pair-changelog/main.go:148`,
`anchorSnippet`). The scrollback (`scrollback-<tag>-<agent>.raw`) is per-*launch*
— O_TRUNC'd on every start, including a resume. On a fresh session the new
transcript doesn't contain the old anchor → `locate` returns **`FullRedistill`**
(`distill.go:187`) → the whole new conversation is re-fed against the existing
per-tag log and appended (leaning on the model to dedup). That's the pile-up.
