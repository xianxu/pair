---
id: 000094
status: open
deps: [000093]
github_issue:
created: 2026-07-01
updated: 2026-07-01
estimate_hours:
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

- [ ] Shell scripts that #93 replaced are removed from the runtime manifest and
      no longer extracted; the generated bundle reflects this.
- [ ] The runtimebundle drift check + copied-binary smoke tests assert the
      reduced extracted tree and fail if a shell asset reappears.
- [ ] A copied `pair` binary runs launch/session/scrollback/review/continuation
      flows with the reduced bundle (external platform tools installed).
- [ ] `atlas/go-migration-inventory.md` reflects the shell-free (or
      shell-reduced) runtime bundle.

## Plan

- [ ] Enumerate which bundled shell assets #93 retired vs which shims remain
      live; confirm no Pair-owned caller still needs each removed name.
- [ ] Remove the dead shell entries via the manifest / bundle generator; regen
      the bundle.
- [ ] Update drift check + copied-binary smoke tests to the reduced tree.
- [ ] Verify a copied-binary run across launch/session/scrollback/review flows.

## Log

### 2026-07-01

Created as step 4 of the native-single-binary tracker (#91). This is the step
where the deployed footprint actually loses shell: it is gated on #93 retiring
the shell orchestrators, and it works through the runtime manifest so packaging
stays single-sourced (the ARCH-DRY property #90 established).
