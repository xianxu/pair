---
id: 000079
status: open
deps: []
github_issue:
created: 2026-06-26
updated: 2026-06-26
estimate_hours:
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

## Log

### 2026-06-26

Created from #72 as the final consolidation milestone. This should land only after the command migration has already made packaging simpler in practice.
