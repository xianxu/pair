# Boundary Review — pair#94 (whole-issue close)

| field | value |
|-------|-------|
| issue | 94 — stop extracting shell scripts from runtime bundle |
| repo | pair |
| issue file | workshop/issues/000094-drop-shell-runtime-assets.md |
| boundary | whole-issue close |
| milestone | — |
| window | d39fdd513fb05e0117c0ec68fa7f20446dd1f7d3..HEAD |
| command | sdlc close --issue 94 |
| reviewer | claude |
| timestamp | 2026-07-03T12:33:43-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I have everything I need. I verified the port against the deleted shell source, ran the Go unit tests and all four affected shell smokes (green), confirmed the bundle was actually regenerated (idempotent, zero orchestrator `.sh`), and traced the routing end-to-end. Two stale documentation lines survive — both cosmetic.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** This is a clean, byte-faithful two-milestone change that genuinely fulfills #94's purpose: all seven orchestrator shell shims are removed from the tree and the runtime bundle, with every consumer repointed to the Go binaries/subcommands. I verified against the *deleted shell source* and running behavior, not the commit messages — the marker protocol round-trips (`serializeRestartMarker`↔`parseRestartMarker` cover tag/agent/new_session/rename_to), routing reaches `LaunchNative`'s restart/quit branches (restart/quit aren't in `DispatchNames()`), the bundle regen is idempotent (git clean after `make pair`), the manifest + `files/` carry zero orchestrator `.sh`, and all smokes pass (`pair-restart-quit`, `copy-on-select`, `pair-embedded-runtime`, `pair-session-watch`) plus the launcher/runtimebundle/clipcmd Go tests. Nothing blocks the close. The only findings are two stale doc lines the M2 follow-up's stale-comment sweep missed — both describe `bin/pair-session-watch.sh` as a *current* shim when it was deleted in M2. One-line fixes; non-blocking.

**1. Strengths**
- Genuine seam reuse, not duplication — `runRestart`/`runQuit` (`cmd/internal/launcher/restart.go:15,40`) drive the existing `WriteRestartMarker`/`TouchQuitMarker`/`ExecKillSession`/`InferAgent`, reuse `createflow_test.go`'s `fakeRuntime` rather than a second fake, and leave `RestartMarker`/`serializeRestartMarker` untouched (ARCH-DRY).
- Silent-misroute trap closed *and pinned* — explicit `case "restart"`/`case "quit"` in `ParseArgs` (`args.go:66-69`) with `TestParseRestart`/`TestParseQuit` asserting `Agent == ""`, the exact regression that would turn `restart` into an agent name.
- Real end-to-end proof — `tests/pair-restart-quit-test.sh` drives the built binary under `PAIR_KILL_CMD=true`, asserts marker *contents* with `grep -qx` (whole-line), and covers the negatives (quit writes no restart marker; missing session → exit 1, no marker). I re-ran it: PASS.
- The M2 silent-failure gap the prior review flagged was actually closed — `sessionWatcherArgv`/`titlePollerArgv` pure helpers + `TestSidecarSpawnArgvTargetsGoBinaries` (`osruntime_test.go`) pin the spawn targets to the suffix-free Go binaries, so a regression back to a `.sh` target (which `spawnDetached` would swallow) is now caught.
- Removal guarded in both directions — `embed_test.go` puts all seven shims in `excluded` + the two Go binaries in `want`; `tests/pair-embedded-runtime-test.sh:47-52` asserts the Go binaries present *and* the five `.sh` absent in the extracted tree.
- Docs are largely thorough and honest — `atlas/architecture.md` + `go-migration-inventory.md` reframe every shim mention, and the "shell-reduced, not shell-free" endpoint (six non-orchestrator utilities kept) is documented in issue/plan/atlas rather than glossed.

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**
- `cmd/pair-session-watch/main.go:3-4` — stale package-doc: "`bin/pair-session-watch.sh` is the by-path re-exec shim the launcher spawns." The shim was deleted in M2; the launcher spawns `bin/pair-session-watch` directly. The sibling `cmd/pair-title/main.go:3-4` was correctly updated ("directly since #94 M2") — this file was simply never touched in the window (the M2 follow-up fixed seven such comments but missed this one). Fix: rephrase to "the launcher spawns `bin/pair-session-watch` directly (the `.sh` shim was retired #94 M2)."
- `atlas/how-to-bring-up-a-new-harness-cli.md:59` — stale: "(`bin/pair-session-watch.sh` remains a compatibility shim)". Points a contributor-facing how-to at a now-deleted file. Not touched in the #94 window. Fix: drop the parenthetical or change to "(the launcher spawns the Go binary directly since #94 M2)".

**5. Test coverage notes.** Coverage matches the bug classes this diff could ship: misroute (unit), marker content/order (unit + smoke), quit-writes-no-restart (both), missing-session exit code (both), bundle contents both directions (embed_test + embedded-runtime smoke), clipcmd exec paths (`run_test.go`), copy_command chain (copy-on-select smoke drives the real binary), and the spawn-argv silent-failure path (the M2 follow-up test). I verified `test-pair-restart-quit` is wired into the aggregate `test:` target (`Makefile.local`). No coverage gap for the shipped behavior.

**6. Architectural notes.**
- ARCH-DRY — **pass.** Seam + fake reused, no new Runtime methods, bundle single-sourced through `explicitAssetPaths`. The three lifecycle runners (`runQuit`/`runRestart`/`runCompaction`) share the marker *seam* but differ in source-of-tag, park step, and payload — extracting a shared helper from three 3-line functions would be over-abstraction, correctly avoided.
- ARCH-PURE — **pass.** `parseRestart` is pure `[]string→(LaunchArgs,error)` tested without IO; `runRestart`/`runQuit` are thin glue over the injected `Runtime`; the M2 argv change extracted pure `sessionWatcherArgv`/`titlePollerArgv` helpers so the join is unit-testable. Clean pure-core / thin-IO-shell split.
- ARCH-PURPOSE — **pass.** Shadow-sweep of the consumers: nvim keybinds → `{'pair','quit'|'restart'}` (M1), launcher spawns → `pair-title`/`pair-session-watch` (M2), zellij `copy_command "copy-on-select"` (M2), clipcmd flash/clip execs → suffix-free Go binaries (M2). Manifest + `files/` verified to carry zero orchestrator `.sh`. The "shell-reduced not shell-free" endpoint is the honest, documented delivery of the purpose (the six kept utilities were never in #93's scope, recorded in the Revision + plan Scope note + atlas) — not a deferred point.
- Note: the `InferAgent`-broader-than-shell divergence (`restart.go:23-27`) is documented at the call site and safe (only fills an otherwise-empty `agent=`, unreachable during the live keybind flow since `agent-<tag>` is always present) — resolved by the M1 follow-up, not re-raised.

**7. Plan revision recommendations.** None — the plan still matches the code; every Core-Concepts row verified at its stated file:line, and the "## Revisions" scope note already records the port-then-delete reframe. The two stale doc lines above are code/atlas fixes, not plan drift.
