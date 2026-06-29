---
id: 000073
status: working
deps: []
github_issue:
created: 2026-06-26
updated: 2026-06-29
estimate_hours: 2.56
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

- [x] Inventory covers `bin/*`, `bin/lib/*`, `cmd/*`, `nvim/*`, `zellij/*`, `Makefile.local`, and packaging/install docs.
- [x] Each artifact has a target disposition and migration priority.
- [x] Hidden callers are identified, especially zellij keybinds/layout commands and nvim shell-outs.
- [x] The inventory names which scripts can remain compatibility shims during migration.
- [x] Pair still works exactly as before; no public behavior changes land in this issue.

## Plan

- [x] Build the artifact/caller table.
- [x] Review docs/tests for caller coverage.
- [x] Record the target disposition for each artifact.
- [x] Update #72 or atlas with any material adjustment to the migration sequence.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v2.md` against `baseline-v2.md`. Method A only.*

```estimate
model: estimate-logic-v2
familiarity: 1.0
item: atlas-docs        design=0.1 impl=0.2
item: pensive           design=0.5 impl=0.2
item: cross-cutting-refactor design=0.5 impl=0.3
item: milestone-review  design=0.1 impl=0.3
design-buffer: 0.3
total: 2.56
```

## Log

### 2026-06-26

Created from #72 as the first executable migration milestone. This is the guardrail against an unfocused rewrite.

### 2026-06-29

Claimed #73 and entered planning. Durable plan created at `workshop/plans/000073-go-migration-inventory-plan.md`. Estimate uses Method A from estimate-logic-v2; scope is inventory/atlas documentation only, with no runtime behavior changes.

Plan-quality gate returned INFO, not blocking but requested stronger coverage checks and full `make test` verification. Estimate-quality returned INFO and requested the spec-quality discount plus standard 30% design buffer; estimate revised from 1.66 to 0.96.

Second plan-quality gate returned FAILURE: 0.96 undercounted the repo-wide inventory depth, and test-only seams were under-specified. Estimate revised to 2.56 with a cross-cutting-refactor item for the multi-surface inspection/classification work; plan clarified that tests are caller/fake evidence grouped by seam rather than exhaustive per-test artifact rows.

Third plan-quality gate returned FAILURE on ARCH-DRY: `atlas/architecture.md` already carries the high-level piece list and #72 packaging target, so the inventory plan must cross-link/supersede that list instead of creating a parallel source. Plan revised to include `atlas/architecture.md`, both `Makefile` and `Makefile.local`, and explicit grouped-surface caveats for `cmd/`, `nvim/`, and tests.

Created `atlas/go-migration-inventory.md` as the authoritative artifact/caller/runtime/disposition table for #74-#79. Linked it from `atlas/index.md` and `atlas/architecture.md`; architecture remains the narrative map while the inventory owns contract facts. Coverage check over `bin`, `cmd`, `nvim`, `zellij`, `Makefile`, `Makefile.local`, packaging docs, and grouped test seams printed no missing paths.

Verification: `make build` passed; `make test` passed; coverage `comm -23` check printed no missing paths; `git diff -- bin cmd nvim zellij Makefile.local` printed no runtime changes; `git diff --check` passed.
