---
id: 000094
status: working
deps: [000093]
github_issue:
created: 2026-07-01
updated: 2026-07-03
estimate_hours: 4.4
started: 2026-07-03T10:57:55-07:00
---

# stop extracting shell scripts from runtime bundle

Tracking: #91 (native single binary) — step 4 of 5. Depends on #93.

## Problem

Once #93 has ported the stateful shell orchestrators into Go, the shell scripts
in the embedded runtime bundle (`bin/*.sh`, `bin/pair-shell`, and any remaining
shell helpers) are dead weight: the copied binary still extracts them to
`$PAIR_DATA_DIR/runtime/<digest>/pair-home` even though nothing execs them. The
runtime manifest (`cmd/internal/runtimebundle/assets/runtime/manifest.json`) is
the single source of what gets packaged and extracted, so shrinking it is the
concrete step that removes shell from the deployed footprint.

## Spec

Remove shell scripts from the embedded runtime bundle so the extracted runtime
carries only native assets (`nvim/`, `zellij/`) plus any Go-owned pieces.

- Drive the removal from the runtime manifest / bundle generator — the single
  packaging source (`ARCH-DRY`). Do not hand-edit a parallel asset list.
- Remove only shell that #93 has actually replaced or retired. Any shim still
  referenced by a live caller blocks its own removal; this issue closes only
  when the shell set is genuinely dead.
- Update the runtimebundle drift check and the copied-binary smoke tests to
  assert the new (shell-free, or shell-reduced) extracted tree, so a regression
  that re-adds a shell asset is caught.
- The runtime selection order (PAIR_HOME → sibling → defaultPairHome → embedded
  extraction) is unchanged; only the *contents* of the embedded bundle shrink.

Merge-safe: launch/session/scrollback/review/continuation flows work from a
copied binary with the reduced bundle before this closes.

Blocked by #93: cannot drop a shell asset until its Go owner exists and every
caller is repointed.

## Done when

- [ ] Every orchestrator shell script is removed from the runtime manifest and no
      longer extracted: the five that #93 replaced (Go owner exists) have their
      callers repointed then are dropped; `pair-restart.sh`/`pair-quit.sh` — which
      had **no** Go sibling — are first ported to in-process `pair restart`/`pair
      quit` subcommands, then dropped. The generated bundle reflects this.
- [ ] The runtimebundle drift check + copied-binary smoke tests assert the
      reduced extracted tree and fail if a shell asset reappears.
- [ ] A copied `pair` binary runs launch/session/scrollback/review/continuation
      flows with the reduced bundle (external platform tools installed).
- [ ] `atlas/go-migration-inventory.md` reflects the shell-free (or
      shell-reduced) runtime bundle.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.2 impl=1.2
item: smaller-go-module design=0.2 impl=1.5
item: milestone-review design=0.0 impl=0.6
item: atlas-docs design=0.1 impl=0.3
total: 4.4
```

Two `smaller-go-module` items (M1 port, M2 repoint-and-drop), the two-boundary
`milestone-review`, and the atlas update. Design hours are weighted **1.6×** per
the v3.1 model, so the raw item sum (4.1) lifts to `total: 4.4` — the 0.3 residual
is that weighting, not a round-number back-fit. Both modules run **low on impl** because
M1 is near-pure reuse (the marker-write seam — `WriteRestartMarker`/`TouchQuitMarker`/
`ExecKillSession`/`InferAgent` — already exists; `runCompaction` is the template)
and M2 is a mechanical caller-repoint sweep (five `.sh`→Go call sites + test/generator
edits), so design is minimal (already done in the durable plan). Durable plan:
`workshop/plans/000094-drop-shell-runtime-assets-plan.md`.

## Plan

The survey (Log 2026-07-03) reframed the scope: only `pair-restart.sh`/`pair-quit.sh`
carry real shell logic (no Go sibling) — the other five are live `.sh`→Go exec-shims
whose callers must be repointed before removal. Two review boundaries:

- [ ] M1 — port `pair-restart.sh` + `pair-quit.sh` into in-process `pair restart`
      `[--new-session] [--rename-to <tag>]` / `pair quit` subcommands (reusing the
      launcher's existing marker seam), repoint the two `nvim/init.lua` keybinds,
      and retire the two shims from the tree + bundle.
- [ ] M2 — repoint the five live exec-shim callers to their Go binaries (launcher
      `SpawnSessionWatcher`/`SpawnTitlePoller`, zellij `copy_command`, and
      `copy-on-select`'s flash/clipboard hand-off), then delete the five shims from
      the tree + `explicitAssetPaths`; tighten `embed_test.go` + the copied-binary
      smoke to assert the shims are gone.

Endpoint: **shell-reduced, not shell-free** — the bundle keeps six non-orchestrator
shell utilities (`bin/lib/{adapt-log,dev-rebuild}.sh`, `bin/pair-help`,
`bin/pair-notify`, `doctor/{doctor,emitter-health}.sh`) that were never in #93's
scope; porting them is out of scope for #94.

## Revisions

### 2026-07-03 — scope: pure-deletion → port-then-delete
The original Spec/Done-when framed #94 as removing "shell that #93 **replaced**."
The survey found `pair-restart.sh`/`pair-quit.sh` were **never replaced by #93** —
they have no Go sibling (they write the restart/quit markers the launcher's loop
reads). So #94 absorbs a small port (M1: `pair restart`/`pair quit`) as the
prerequisite to dropping them — you can't stop extracting `pair-restart.sh`
without providing restart in Go. This is load-bearing scope-absorption toward
#94's purpose ("stop extracting shell scripts"), not creep. **Delta:** Done-when #1
reworded (port-then-delete for the two logic shims; repoint-then-delete for the
five exec-shims); Plan split into M1 (port) / M2 (repoint-and-drop). The five
exec-shims remain pure deletion-after-repoint, unchanged from the original intent.

## Log

### 2026-07-03
- Claimed + planned (durable plan `workshop/plans/000094-drop-shell-runtime-assets-plan.md`).
- **Survey reframed the scope.** `git ls-files bin/` + the manifest + caller grep
  showed the 11 bundled `.sh` split three ways: (1) `pair-restart.sh`/`pair-quit.sh`
  — real shell logic, no Go sibling, one nvim caller each → **M1 ports them**;
  (2) `pair-title.sh`/`pair-session-watch.sh`/`copy-on-select.sh`/`flash-pane.sh`/
  `clipboard-to-pane.sh` — live `.sh`→Go passthrough exec-shims (launcher spawns,
  zellij `copy_command`, copy-on-select's hand-off) → **M2 repoints the callers,
  then drops them**; (3) `lib/{adapt-log,dev-rebuild}.sh`, `pair-help`,
  `pair-notify`, `doctor/*.sh` — non-orchestrator utilities never in #93's scope →
  **kept** (endpoint is shell-reduced, not shell-free). M1 is near-pure reuse: the
  marker-write seam (`WriteRestartMarker`/`TouchQuitMarker`/`ExecKillSession`/
  `InferAgent`) already exists — `runCompaction` is the template.
- Fresh-eyes plan review (fact-checked vs. code) confirmed the reuse claim, router
  placement, all five repoint sites, and no missed callers; fixed three concrete
  plan errors (embed_test `want`-list removal, copy-on-select-test self-refs, the
  fake's `inferAgent` field name).

### 2026-07-01

Created as step 4 of the native-single-binary tracker (#91). This is the step
where the deployed footprint actually loses shell: it is gated on #93 retiring
the shell orchestrators, and it works through the runtime manifest so packaging
stays single-sourced (the ARCH-DRY property #90 established).
