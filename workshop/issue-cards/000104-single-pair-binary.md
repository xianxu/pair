---
id: '000104'
status: done
started: 2026-07-06T16:05:29-07:00
created: 2026-07-06
updated: 2026-07-06
estimate_hours: 9
actual_hours: 8.53
---

# Fold pair repo binaries into a single pair Go program

## Problem

Dev builds compile ~19 Go binaries; `pair-dev` re-runs `make build` on every
launch and every in-session restart, so the whole link cost is paid constantly.
`bin/pair` and `bin/pair-go` are the same program linked twice into two 81 MB
files, and `pair` embeds 17 helper binaries in its runtime bundle (~55–65 MB of
its 81 MB). The many `pair-*` names on PATH also create "which do I call?"
confusion. Only one caller is external — `pair-scribe`, wired into the user's
`~/.zshrc` — and that stays maintained separately.
