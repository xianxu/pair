---
id: '000099'
status: done
started: 2026-07-02T11:20:01-07:00
created: 2026-07-02
updated: 2026-07-03
estimate_hours: 17.7
actual_hours: 11.57
---

# port the pair-shell launcher to Go

## Problem

`bin/pair-shell` (2287 lines) is the last and largest shell orchestrator: it owns
the zellij session lifecycle, the create/attach/pick decision, three UIs (fzf
session picker, fzf config/tag-restart picker, zsh `vared` name-prompt), the
restart/quit marker lifecycle, cmux ownership, config/session migration, per-agent
launch-arg composition, nvim orphan reaping, the `list`/`rename`/`continue`
subcommands, and the spawns of the (already-Go) title poller + session watcher.
Until it moves into Go, `pair` can't stop `syscall.Exec`ing a shell launcher and
#94 (stop extracting a shell tree) can't proceed. It's P0 in
`atlas/go-migration-inventory.md`.
