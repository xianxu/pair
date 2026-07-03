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

---

## Re-review — 2026-07-03T12:43:52-07:00 (SHIP)

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
| timestamp | 2026-07-03T12:43:52-07:00 |
| verdict | SHIP |

## Review

Both prior close-review findings are resolved at HEAD — the `a0c3062` commit genuinely applied them. My independent verification is complete. Writing up the verdict.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** This whole-issue close for pair#94 is a clean, byte-faithful, well-tested delivery that genuinely fulfills the issue's purpose: all seven orchestrator shell shims are removed from the tree and the runtime bundle, every consumer repointed to the Go binaries/subcommands. I verified against the *deleted shell source* and running behavior, not commit messages. Independently confirmed: (1) `pair restart`/`pair quit` route end-to-end — `restart`/`quit` are absent from `dispatcher.Families()`, so `ClassifyInvocation` returns `ModePublicPair` → `LaunchNative`'s restart/quit branches (not the buffered dispatcher, which lacks the Runtime seam and `ZELLIJ_SESSION_NAME`); (2) the marker round-trips (`serializeRestartMarker`↔`parseRestartMarker`, the `new_session=0` omission inert because the reader treats absent as false); (3) `WriteRestartMarker`/`TouchQuitMarker` auto-`MkdirAll` the cache dir, so the smoke's dropped pre-`mkdir` is load-bearing; (4) the manifest carries **zero** orchestrator `.sh` and the Go binaries `bin/pair-title`/`bin/pair-session-watch`/`bin/{copy-on-select,flash-pane,clipboard-to-pane}` instead; the source shims are deleted; git is clean after `make pair` (regen idempotent). I re-ran the affected Go packages (launcher/runtimebundle/clipcmd/entrypoint/dispatcher — all ok) and both smokes (`pair-restart-quit` PASS, `copy-on-select` PASS). The two stale-doc findings from the prior FIX-THEN-SHIP close-review are **resolved at HEAD** (verified `cmd/pair-session-watch/main.go` + `atlas/how-to-bring-up-a-new-harness-cli.md`). Nothing blocks the close; the single residual is one trivial internal comment.

**1. Strengths**
- **Routing correctness pinned, not just asserted** — explicit `case "restart"`/`case "quit"` in `ParseArgs` (`args.go:66-69`) with `TestParseRestart`/`TestParseQuit` asserting `Agent == ""`, closing the silent-misroute trap where an unrecognized verb becomes an agent name. I confirmed the `ModePublicPair` path independently against `entrypoint/mode.go:19-25`.
- **Genuine seam reuse, no duplication** — `runRestart`/`runQuit` (`restart.go:15,40`) drive the existing `WriteRestartMarker`/`TouchQuitMarker`/`ExecKillSession`/`InferAgent`; no new Runtime methods, `RestartMarker`/`serializeRestartMarker` untouched, tests reuse the existing `fakeRuntime` (ARCH-DRY).
- **Real end-to-end proof** — `tests/pair-restart-quit-test.sh` drives the *built* binary under `PAIR_KILL_CMD=true`, asserts marker *contents* with `grep -qx` (whole-line), and covers negatives (quit writes no restart marker; missing session → exit 1, no marker). Re-ran: PASS.
- **Removal guarded both directions** — `embed_test.go:37-44` puts all seven shims in `excluded` + the two Go binaries in `want`; `tests/pair-embedded-runtime-test.sh:47-52` asserts the Go binaries present *and* the five `.sh` absent in the extracted tree.
- **The M2 silent-spawn gap the prior review flagged is closed** — pure `sessionWatcherArgv`/`titlePollerArgv` + `TestSidecarSpawnArgvTargetsGoBinaries` pin the spawn targets to suffix-free Go binaries, so a regression back to a `.sh` (which `spawnDetached` would swallow) is caught.
- **Honest endpoint** — "shell-reduced, not shell-free" (six non-orchestrator utilities kept: `bin/lib/{adapt-log,dev-rebuild}.sh`, `bin/pair-help`, `bin/pair-notify`, `doctor/{doctor,emitter-health}.sh`) is documented in issue/plan/atlas rather than glossed. Manifest confirms exactly those remain.

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**
- `cmd/internal/clipcmd/runtime.go:127` — the `RunSubprocess` doc still names the retired shim: "inheriting the environment so **flash-pane.sh** sees PAIR_FLASH_* / XDG_CACHE_HOME." The subprocess is now the Go `bin/flash-pane` (`run.go:87`), so this is stale-current (not provenance). One-word fix: `flash-pane`. Same class the M2 stale-comment sweep addressed, but clipcmd wasn't in that file list, so it was missed. Cosmetic, no behavior impact. (The sibling `clipcmd.go:24,58,88` and `runtime.go:90` are legitimate "mirrors/faithful-to X.sh" provenance — leave them.)

**5. Test coverage notes.** Coverage matches every bug class this diff could ship: misroute (unit + `Agent==""`), marker content/order (unit + smoke), quit-writes-no-restart (both), missing-session exit code (both), bundle contents both directions (`embed_test` + embedded-runtime smoke), clipcmd exec paths (`run_test.go` asserts suffix-free `/h/bin/flash-pane` + `/h/bin/clipboard-to-pane`), copy_command chain (copy-on-select smoke drives the real binary), and the spawn-argv silent-failure path. `test-pair-restart-quit` is wired into the aggregate `test:` target (`Makefile.local`). No gap for the shipped behavior.

**6. Architectural notes.**
- **ARCH-DRY — pass.** Seam + fake reused, no new Runtime methods, bundle single-sourced through `explicitAssetPaths` → generator → manifest (regen idempotent, git clean). The three lifecycle runners (`runQuit`/`runRestart`/`runCompaction`) share the marker *seam* but differ in source-of-tag, park step, and payload — extracting a helper from three ~3-line functions would be over-abstraction, correctly avoided.
- **ARCH-PURE — pass.** `parseRestart` is pure `[]string→(LaunchArgs,error)` tested without IO; `runRestart`/`runQuit` are thin glue over the injected `Runtime`; the M2 argv change extracted pure `sessionWatcherArgv`/`titlePollerArgv`. Clean pure-core / thin-IO-shell split.
- **ARCH-PURPOSE — pass.** Shadow-swept every consumer: nvim keybinds → `{'pair','quit'|'restart'}` (`init.lua:3185,3288`), launcher spawns → `pair-title`/`pair-session-watch` (`osruntime.go`), zellij `copy_command "copy-on-select"` (`config.kdl:41`), clipcmd flash/clip execs → suffix-free Go binaries (`run.go:87,97`). Manifest verified to carry zero orchestrator `.sh`. Every consumer *derives* from the Go binary — no hand-maintained restatement remains as a live caller. The purpose is delivered, not a subset.
- Note (not a finding): `pair quit`/`restart` now resolve the asset root via `runLegacyLaunch` before the marker write, where the shell needed none. In the live keybind flow `PAIR_HOME` is always exported + valid, so resolution is trivial and the smoke confirms it — arguably more robust (same validated launcher path), not a regression.
- The documented `InferAgent`-broader-than-shell divergence (`restart.go:23-27`) is safe (only fills an otherwise-empty `agent=`, unreachable in the live flow since `agent-<tag>` is always present) — resolved by the M1 follow-up, not re-raised.

**7. Plan revision recommendations.** None — the plan still matches the code. Every Core-Concepts row verified at its stated `file:line` (pure entities `LaunchArgs`/`parseRestart`/`RestartMarker`/`serializeRestartMarker`; integration `runRestart`/`runQuit`/`LaunchNative` branches/nvim sites/spawn argv/clipcmd exec/`copy_command`/`explicitAssetPaths`/`embed_test`), all four Done-when items delivered, and the "## Revisions" scope note already records the port-then-delete reframe. The one Minor comment is a code fix, not plan drift.
