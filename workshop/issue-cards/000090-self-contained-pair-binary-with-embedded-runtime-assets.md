---
id: '000090'
status: done
started: 2026-07-01T00:18:42-07:00
created: 2026-07-01
updated: 2026-07-01
estimate_hours: 5.44
actual_hours: 0.77
---

# self-contained pair binary with embedded runtime assets

## Problem

After #79, the public `pair` command is Go-owned, but deployment is still an
installed tree: the Go entrypoint must find adjacent Pair-owned runtime assets
such as `bin/pair-shell`, shell helpers, `nvim/`, `zellij/`, and helper
binaries. That is simpler for Homebrew, but it is not the deployment shape we
eventually want: copying one Pair binary around and having it work.

The long-term direction is a true native single binary. Rewriting every
remaining shell and orchestration surface directly into Go is too much risk in
one jump, so the next step should make the current runtime tree derive from one
Go artifact without pretending the shell lifecycle is already gone.
