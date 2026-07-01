---
id: 000079
status: working
deps: [000077, 000078]
github_issue:
created: 2026-06-26
updated: 2026-06-30
estimate_hours: 3.13
started: 2026-06-30T16:59:55-07:00
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

- [ ] Packaging installs the primary Go `pair` command and required assets coherently.
- [ ] Obsolete compatibility shims are removed or explicitly retained with a reason.
- [ ] README and atlas describe the new install/runtime layout.
- [ ] Clean install and upgrade paths are verified.
- [ ] Pair remains usable after merge.

## Plan

- [ ] Choose embedded vs adjacent asset strategy.
- [ ] Update build/install/Homebrew wiring.
- [ ] Remove or document remaining shims.
- [ ] Run clean install/upgrade verification.
- [ ] Update docs and atlas.

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

Claimed after #78 landed. Chose adjacent native assets and Go public entrypoint: build `cmd/pair-go` as installed `pair`, retain `pair-go` as the development dispatcher alias, and keep the current shell launcher as an internal compatibility handoff while the zellij lifecycle remains shell-owned. Wrote durable plan at `workshop/plans/000079-go-packaging-consolidation-plan.md`. Plan-quality found missing Homebrew and upgrade specificity; tightened the plan to include sibling formula `../homebrew-pair/Formula/pair.rb`, a concrete old-symlink-to-Go-binary upgrade test, and a single decided asset strategy: local installs stay source-tree adjacent, Homebrew installs an adjacent `libexec` tree. Second plan-quality pass found asset-root and tracked-file ambiguity; tightened the plan again so pure `AssetRoot` resolves `PAIR_HOME` / sibling root / build-time `defaultPairHome`, and so `bin/pair-shell` is tracked while generated `bin/pair` is ignored. Estimate derived with v3.1 calibration; calibration source is marked stale by `sdlc estimate-source`, so the number is provisional but uses the required method.
