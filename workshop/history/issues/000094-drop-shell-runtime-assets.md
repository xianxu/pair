---
id: 000094
status: done
deps: [000093]
github_issue:
created: 2026-07-01
updated: 2026-07-03
estimate_hours: 4.4
started: 2026-07-03T10:57:55-07:00
actual_hours: 2.50
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

- [x] Every orchestrator shell script is removed from the runtime manifest and no
      longer extracted: the five that #93 replaced (Go owner exists) have their
      callers repointed then are dropped; `pair-restart.sh`/`pair-quit.sh` — which
      had **no** Go sibling — are first ported to in-process `pair restart`/`pair
      quit` subcommands, then dropped. The generated bundle reflects this.
- [x] The runtimebundle drift check + copied-binary smoke tests assert the
      reduced extracted tree and fail if a shell asset reappears.
- [x] A copied `pair` binary runs launch/session/scrollback/review/continuation
      flows with the reduced bundle (external platform tools installed).
- [x] `atlas/go-migration-inventory.md` reflects the shell-free (or
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

- [x] M1 — port `pair-restart.sh` + `pair-quit.sh` into in-process `pair restart`
      `[--new-session] [--rename-to <tag>]` / `pair quit` subcommands (reusing the
      launcher's existing marker seam), repoint the two `nvim/init.lua` keybinds,
      and retire the two shims from the tree + bundle.
- [x] M2 — repoint the five live exec-shim callers to their Go binaries (launcher
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
- 2026-07-03: closed — #94 removes all 7 orchestrator shell shims from the runtime bundle (M1 ports pair-restart.sh/pair-quit.sh -> pair restart/quit; M2 repoints the 5 exec-shim callers to Go binaries then deletes them). Re-close after FIX-THEN-SHIP: the integration review flagged 2 Minor stale-doc lines (cmd/pair-session-watch/main.go + atlas/how-to-bring-up-a-new-harness-cli.md naming the deleted shim) — fixed both + 3 equivalent stale Makefile.local comments. Full make test green (MAKE_EXIT=0); embed_test + copied-binary smoke assert the 7 shims absent + Go binaries present; TestSidecarSpawnArgv pins spawn targets not .sh. --no-verdict: M1/M2 boundary reviews ran (SHIP, in ## Log + m1/m2-review sidecars), trailers just not in the milestone commit messages (commit ordering). --no-reclose-guard: re-review of the FIX-THEN-SHIP doc delta.; review verdict: SHIP
- 2026-07-03: closed — #94 removes all seven orchestrator shell shims from the runtime bundle. M1 ports pair-restart.sh/pair-quit.sh -> in-process `pair restart`/`pair quit` (reusing the launcher marker seam, no new Runtime methods). M2 repoints the five exec-shim callers (launcher SpawnSessionWatcher/SpawnTitlePoller, zellij copy_command, copy-on-selects flash/clip hand-off) to the Go binaries, then deletes them. Verified: full make test green (MAKE_EXIT=0) — embed_test + copied-binary smoke assert the 7 shims absent + Go binaries present; test-copy-on-select/session-watch/pair-restart-quit drive the real binaries; TestSidecarSpawnArgv pins the spawn targets are not .sh. Endpoint: shell-reduced — 6 non-orchestrator utilities remain. --no-verdict: both milestone boundary reviews ran via sdlc milestone-close and returned SHIP (recorded in ## Log + workshop/plans/000094-*-m{1,2}-review.md sidecars); the trailers were not committed to the milestone-close commit messages due to commit ordering (code committed before milestone-close), but the reviews genuinely ran.; review verdict: FIX-THEN-SHIP
- 2026-07-03: closed M2 — M2 repoints the five live exec-shim callers to the Go binaries (launcher SpawnSessionWatcher/SpawnTitlePoller -> pair-session-watch/pair-title; zellij copy_command -> copy-on-select; copy-on-selects flash/clip hand-off -> Go binaries via clipcmd) then deletes all five shims from tree + explicitAssetPaths. Verified: full make test green (MAKE_EXIT=0) incl test-copy-on-select (Go binary in_nvim PASS), test-session-watch (Go binary PASS), test-runtimebundle (embed_test asserts 5 shims excluded + Go binaries present), test-pair-embedded-runtime (copied binary extracts bundle w/ Go binaries + asserts no .sh shim). Endpoint: shell-reduced — all 7 orchestrator shims gone; 6 non-orchestrator utilities remain.; review verdict: SHIP
- **M2 review follow-ups (SHIP → SHIP).** No Critical/Important. (1) Pinned the
  silent-failure gap the reviewer flagged: extracted `sessionWatcherArgv`/
  `titlePollerArgv` pure helpers + `TestSidecarSpawnArgvTargetsGoBinaries` asserting
  the spawn targets are the Go binaries (no `.sh`), so a regression back to a shim
  target (which spawnDetached would swallow) is now caught. (2) Corrected seven
  stale present-tense comments that still narrated the deleted shims (titlepoller,
  osruntime/markers, transcript, wrapcmd, adapt-log) — provenance "ported from"
  lines kept.
- 2026-07-03: closed M1 — M1 ports pair-restart.sh/pair-quit.sh to in-process `pair restart [--new-session] [--rename-to <tag>]` / `pair quit`, reusing the launchers existing WriteRestartMarker/TouchQuitMarker/ExecKillSession/InferAgent seam (no new Runtime methods; runCompaction is the template). nvim keybinds repointed; both shims deleted from tree + explicitAssetPaths. Verified: full make test green (MAKE_EXIT=0) incl new tests/pair-restart-quit-test.sh PASS (real pair binary writes restart/quit markers to ~/.cache/pair under PAIR_KILL_CMD stub) + fake-Runtime unit tests + pure parse tests + embed_test excludes the 2 shims from the bundle.; review verdict: SHIP
- **M1 review follow-ups (SHIP → SHIP).** No Critical/Important. Applied two Minor
  cleanups: (1) dropped the redundant `mkdir -p` from the smoke so `pair restart`
  now exercises the auto-`MkdirAll` cache-dir path (load-bearing); (2) documented
  the deliberate `InferAgent`-broadening-vs-shell divergence with a code comment at
  the call site. Cosmetic stderr-wording + the unported empty-arg skip were noted
  as intentional, no change.
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
