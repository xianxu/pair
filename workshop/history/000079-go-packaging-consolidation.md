---
id: 000079
status: done
deps: [000077, 000078]
github_issue:
created: 2026-06-26
updated: 2026-06-30
estimate_hours: 3.13
started: 2026-06-30T16:59:55-07:00
actual_hours: 1.02
---

# pair Go packaging consolidation

## Problem

The migration is not complete until packaging actually becomes simpler. A Go entrypoint with a pile of stale aliases, duplicated binaries, and unclear asset handling would miss the purpose of #72.

## Spec

Consolidate release/install packaging around the primary Go `pair` binary and its required assets.

Expected work:

- Decide whether `nvim/` and `zellij/` assets are embedded in the binary or installed adjacent to it.
- Update Homebrew/install/build docs and `Makefile.local` accordingly.
- Remove obsolete compatibility shims only after all callers derive from the Go command or documented native assets.
- Verify clean install/upgrade behavior on the supported platform and, if practical, a Linux smoke path.
- Update atlas/README so the packaging architecture is discoverable.

This is where the migration proves its value: fewer installed moving parts and a clearer release story (`ARCH-PURPOSE`).

Approved design: make the installed public `pair` command Go-owned while keeping native assets adjacent. Build `cmd/pair-go` as both `bin/pair` and `bin/pair-go`: direct `pair ...` invokes the compatibility launch handoff, while `pair-go ...` keeps the explicit development dispatcher surface (`pair-go launch`, helper routes). Move the existing Bash launcher behind an internal compatibility name (`bin/pair-shell`) for this issue rather than pretending the full zellij lifecycle is already native Go. Keep `nvim/` and `zellij/` adjacent assets because they are loaded by Neovim/Zellij directly and are heavily tested as files; embedding would require extraction/path rewrites across many surfaces with little packaging payoff right now. `ARCH-PURPOSE`: #79 is only done if installed `pair` is the Go public entrypoint, not just if docs say it will be later. `ARCH-DRY` and `ARCH-PURE`: direct `pair` and `pair-go launch` share one pure compatibility request builder; filesystem/exec behavior stays in a thin runtime seam.

## Done when

- [x] Packaging installs the primary Go `pair` command and required assets coherently.
- [x] Obsolete compatibility shims are removed or explicitly retained with a reason.
- [x] README and atlas describe the new install/runtime layout.
- [x] Clean install and upgrade paths are verified.
- [x] Pair remains usable after merge.

## Plan

- [x] Choose embedded vs adjacent asset strategy.
- [x] Update build/install/Homebrew wiring.
- [x] Remove or document remaining shims.
- [x] Run clean install/upgrade verification.
- [x] Update docs and atlas.

Detailed implementation plan: `workshop/plans/000079-go-packaging-consolidation-plan.md`.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.20 impl=0.08
item: cross-cutting-refactor design=0.60 impl=0.64
item: smaller-go-module design=0.35 impl=0.40
item: atlas-docs design=0.25 impl=0.20
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 3.13
```

## Log

### 2026-06-26

Created from #72 as the final consolidation milestone. This should land only after the command migration has already made packaging simpler in practice.

### 2026-06-30
- 2026-06-30: closed — go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1; make build; make test-pair-go-install-layout; bin/pair --help; bin/pair-go launch --help; bin/pair-dev --help; make test-dev-rebuild test-session-watch test-continue; make -n -B test-continue test-cmux-ownership confirmed bin/pair prerequisite; make test-continue test-cmux-ownership; go test ./... -count=1; ruby -c ../homebrew-pair/Formula/pair.rb; homebrew tap commit 3aeb2a6. brew test --formula is unsupported by this Homebrew; Linux smoke not run because workspace is Darwin-only.; review verdict: FIX-THEN-SHIP

Claimed after #78 landed. Chose adjacent native assets and Go public entrypoint: build `cmd/pair-go` as installed `pair`, retain `pair-go` as the development dispatcher alias, and keep the current shell launcher as an internal compatibility handoff while the zellij lifecycle remains shell-owned. Wrote durable plan at `workshop/plans/000079-go-packaging-consolidation-plan.md`. Plan-quality found missing Homebrew and upgrade specificity; tightened the plan to include sibling formula `../homebrew-pair/Formula/pair.rb`, a concrete old-symlink-to-Go-binary upgrade test, and a single decided asset strategy: local installs stay source-tree adjacent, Homebrew installs an adjacent `libexec` tree. Second plan-quality pass found asset-root and tracked-file ambiguity; tightened the plan again so pure `AssetRoot` resolves `PAIR_HOME` / sibling root / build-time `defaultPairHome`, and so `bin/pair-shell` is tracked while generated `bin/pair` is ignored. Estimate derived with v3.1 calibration; calibration source is marked stale by `sdlc estimate-source`, so the number is provisional but uses the required method.

Implemented #79 packaging consolidation. `bin/pair` is now generated from `cmd/pair-go`, `bin/pair-shell` is the tracked compatibility launcher, local install copies a regular Go `pair` binary, and Homebrew builds Go `pair` / `pair-go` plus required runtime helpers into `libexec/bin` with adjacent native assets. Homebrew tap evidence: sibling repo `../homebrew-pair` commit `3aeb2a6 pair: build Go public entrypoint` updates `Formula/pair.rb`. Verification: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`; `make build`; `make test-pair-go-install-layout`; `bin/pair --help`; `bin/pair-go launch --help`; `bin/pair-dev --help`; `make test-dev-rebuild test-session-watch test-continue`; `go test ./... -count=1`; `ruby -c ../homebrew-pair/Formula/pair.rb`; stale-doc grep for old #77 packaging wording. `brew test --formula ../homebrew-pair/Formula/pair.rb` was not available on this Homebrew (`invalid option: --formula`), so the formula was syntax-checked locally rather than installed over the operator environment. Linux smoke was not run because this workspace is Darwin-only (`uname -s` => `Darwin`) and no Linux runner is configured.

Close review returned `FIX-THEN-SHIP`. Addressed the findings before committing close state: updated `atlas/how-to-bring-up-a-new-harness-cli.md` launcher-recovery guidance from generated `bin/pair` to retained `bin/pair-shell`, and changed stale `pair-go launch` unsupported-subcommand guidance from `use bin/pair` to `use pair`. Verified with `go test ./cmd/internal/launcher ./cmd/internal/entrypoint ./cmd/pair-go -count=1`, stale-text grep, and `git diff --check`.
