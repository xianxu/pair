---
id: '000093'
status: done
started: 2026-07-01T14:39:06-07:00
created: 2026-07-01
updated: 2026-07-03
estimate_hours: 17.4
actual_hours: 5.41
---

# port stateful shell orchestrators to Go

## Problem

The runtime's stateful orchestration still lives in shell. `bin/pair-shell`
(the launcher) owns the zellij lifecycle, prompt UI, restart/quit cleanup, cmux
ownership, dev rebuild, continuation, rename, config/session migration, and the
title poller; `bin/pair-title.sh`, `bin/pair-scrollback-open`, the
`bin/pair-review-*` helpers, and the clipboard helpers
(`clipboard-to-pane.sh`, `copy-on-select.sh`, `flash-pane.sh`) are all shell
orchestrators. Until this logic moves into Go, the binary can never stop
extracting a shell tree (#94), so this is the load-bearing step toward a native
single binary. `atlas/go-migration-inventory.md` already flags each surface with
a migration priority (P0 launcher, P1 title/scrollback, P2 review helpers).
