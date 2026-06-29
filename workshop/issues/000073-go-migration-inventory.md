---
id: 000073
status: working
deps: []
github_issue:
created: 2026-06-26
updated: 2026-06-29
estimate_hours:
started: 2026-06-29T13:25:37-07:00
---

# pair Go migration inventory

## Problem

The Go consolidation needs a factual contract before code moves. Pair currently installs a mix of Go binaries, shell scripts, Lua files, zellij KDL assets, and helper libraries. Without an inventory, later migration issues risk either under-delivering the single-primary-binary target or breaking a hidden caller.

## Spec

Create an inventory that maps every installed or runtime-called artifact to:

- Current path and type: Go binary, shell script, Lua asset, zellij asset, helper library, test-only seam.
- Callers: user shell, `bin/pair`, zellij config/layout, nvim Lua, tests, other helper scripts, Homebrew/install.
- Runtime contract: argv, required env vars, files read/written, exit-code expectations, long-running behavior.
- Target disposition: Go subcommand, embedded/adjacent asset, temporary compatibility shim, or native asset that should stay as-is.
- Migration priority and reason, led by packaging impact first and reliability/testability second.

This issue does not port behavior. It produces the contract later issues use.

## Done when

- [ ] Inventory covers `bin/*`, `bin/lib/*`, `cmd/*`, `nvim/*`, `zellij/*`, `Makefile.local`, and packaging/install docs.
- [ ] Each artifact has a target disposition and migration priority.
- [ ] Hidden callers are identified, especially zellij keybinds/layout commands and nvim shell-outs.
- [ ] The inventory names which scripts can remain compatibility shims during migration.
- [ ] Pair still works exactly as before; no public behavior changes land in this issue.

## Plan

- [ ] Build the artifact/caller table.
- [ ] Review docs/tests for caller coverage.
- [ ] Record the target disposition for each artifact.
- [ ] Update #72 or atlas with any material adjustment to the migration sequence.

## Log

### 2026-06-26

Created from #72 as the first executable migration milestone. This is the guardrail against an unfocused rewrite.
