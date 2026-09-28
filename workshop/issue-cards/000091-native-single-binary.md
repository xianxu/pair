---
id: '000091'
status: done
created: 2026-07-01
updated: 2026-07-04
actual_hours: N/A
---

# native single binary pair

## Problem

After #79 (`pair` is Go-owned) and #90 (a copied `pair` binary embeds and
extracts the Pair-owned runtime), Pair can be deployed as one artifact — but the
artifact is not yet a *true* native single binary. At runtime it still extracts a
mixed shell/Go/Lua/zellij tree to `$PAIR_DATA_DIR/runtime/<digest>/pair-home`
and execs `bin/pair-shell`; the shell lifecycle, the Lua/zellij assets, and the
several legacy helper binaries are all still live.

#90's Spec (lines 49–65) documents the remaining execution path toward the true
native single binary, and `atlas/go-migration-inventory.md` classifies every
remaining shell/asset surface with a migration priority. But that path is
**un-ticketed**: the prior tracking roadmap #72 deliberately closed at #79, and
#90 was created ad hoc afterward without spawning successors. This issue is the
umbrella tracker that carries the remaining phases so they don't stay buried in
#90's Spec and the atlas.
