---
id: '000072'
status: done
started: 2026-06-26T10:06:43-07:00
created: 2026-06-25
updated: 2026-06-26
actual_hours: 0.5
---

# migration all to Go

## Problem

Pair started as a set of shell scripts, then performance- and correctness-sensitive pieces moved to Go (`pair-wrap`, `pair-scrollback-render`, `pair-slug`, `pair-changelog`, `pair-continuation`, `pair-context`, `pair-scribe`). The remaining installed surface is now a mixed shell/Go/Lua/zellij asset tree. That works for development, but it makes packaging and distribution harder than the desired end state: a primary `pair` binary that can be installed, upgraded, and reasoned about as one command.

The question is not "rewrite everything because Go is nicer." The question is whether a packaging-led Go consolidation is worth doing, and if so, how to sequence it so every intermediate merge leaves Pair working.
