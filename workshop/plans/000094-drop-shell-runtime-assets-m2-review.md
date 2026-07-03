# Boundary Review — pair#94 (milestone M2)

| field | value |
|-------|-------|
| issue | 94 — stop extracting shell scripts from runtime bundle |
| repo | pair |
| issue file | workshop/issues/000094-drop-shell-runtime-assets.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 2ca2f60fa6f9acccc5fcd34b75926b09f4fb67cc^..HEAD |
| command | sdlc milestone-close --issue 94 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-07-03T12:20:04-07:00 |
| verdict | SHIP |

## Review

I have everything I need. Let me confirm the aggregate `test` wiring one final time and then write up.

The verification is complete. All test-target prereqs build their binaries (fresh-tree safe), all four affected smokes (`test-copy-on-select`, `test-session-watch`, `test-pair-embedded-runtime`, `test-pair-restart-quit`) are in the aggregate `test:` target, and every behavioral claim checks out against the code.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** This M2 boundary is a clean, mechanical caller-repoint-then-delete sweep that genuinely fulfills #94's purpose. I verified every claim against the filesystem and code, not the commit messages: the bundle was actually regenerated (manifest + `files/` carry zero orchestrator `.sh`, and `make pair` regeneration is idempotent — git stays clean), all seven orchestrator shims are gone from the tree, the five live exec-shim callers are repointed (osruntime spawns → `pair-title`/`pair-session-watch`, zellij `copy_command "copy-on-select"`, clipcmd's flash/clipboard exec → suffix-free Go binaries), and the removal is guarded by both `embed_test.go`'s excluded-list and the copied-binary smoke's `test ! -e` assertions. The one load-bearing behavioral claim — that the title poller's single-instance argv guard matches the direct-spawn shape — holds: `pollerArgvMatches` keys on the substring `pair-title <tag> `, which `…/pair-title <tag> <agent>` satisfies identically to the shim's re-exec'd steady state. All Go unit tests + all four repointed/new shell smokes pass. Nothing blocks the boundary; findings are all Minor (stale comments + one silent-spawn coverage note).

**1. Strengths**
- **Bundle single-sourcing respected and verified end-to-end** — `explicitAssetPaths` (`generate.go:19`) → generator → `manifest.json`, with regeneration provably idempotent (git clean after `make pair` re-ran the generator). No hand-edited parallel list; ARCH-DRY intact.
- **The removal is genuinely guarded, not just asserted in prose** — `embed_test.go:37-44` puts all seven shims in `excluded` and the two Go binaries in `want`; `tests/pair-embedded-runtime-test.sh:47-52` asserts both presence of the Go binaries *and* absence of the five `.sh` in the extracted tree. A regression re-adding a shim fails loudly at two layers.
- **The clipcmd exec-path repoint is unit-covered** — `run_test.go` asserts the hand-off execs `/h/bin/flash-pane` and `/h/bin/clipboard-to-pane` (suffix-free), so `run.go:87,97` won't silently regress.
- **PairHome resolution is robust to the shim's removal** — `repoRootFromExe()` (`runcli.go:43-51`) self-locates when `PAIR_HOME` is unset, replacing the shim's readlink fallback; and this predates M2, so direct invocation introduces no new env-dependency risk.
- **ARCH-PURPOSE fulfilled honestly** — all seven orchestrator shims removed; the "shell-reduced, not shell-free" endpoint (six non-orchestrator utilities kept) is documented in the issue, plan, and `go-migration-inventory.md` rather than glossed.

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**
- **Stale present-tense comments describing the now-deleted `.sh` shims.** Several comments still narrate a mechanism M2 removed (provenance "ported from bin/X.sh" is fine and should stay; these describe *current* behavior wrongly):
  - `cmd/internal/titlepoller/titlepoller.go:92` — "the shim re-execs the Go binary" (shim retired; launcher spawns Go directly).
  - `cmd/internal/titlepoller/runcli.go:11` — "shared by the bin/pair-title.sh shim and …".
  - `cmd/internal/launcher/osruntime.go:431` — "where pair-restart.sh drops the {quit,restart}-<session> markers" (now `pair restart`).
  - `cmd/internal/launcher/markers.go:11` — "written by pair-restart.sh (Alt+n / Shift+Alt+N)".
  - `cmd/internal/transcript/transcript.go:18` — "written by bin/pair / pair-session-watch.sh".
  - `cmd/internal/wrapcmd/wrap.go:1920` — "so pair-session-watch.sh can bind discovery".
  - `bin/lib/adapt-log.sh:3` — "Sourced by shell components (pair-session-watch.sh, …)" (pair-session-watch is Go now and doesn't source it).
  A one-word `.sh` strip / re-phrase on each. Cosmetic; no behavior impact.
- **The osruntime spawn-target repoint has no direct test** (`osruntime.go:276,284`). `spawnDetached` silently returns on `cmd.Start()` error (`osruntime.go:310-311`), so a future regression reverting the target to `pair-title.sh`/`pair-session-watch.sh` would fail *silently* at runtime (title poller / codex-agy session-id capture would just not start) with no test catching it. Correct as written now, and the embedded-runtime smoke confirms only the Go binaries are bundled — so this diff does not ship the bug — but the argv string itself is unpinned. See test note below.

**5. Test coverage notes.** The bug classes M2 could ship are covered where it matters most: bundle contents (embed_test + embedded-runtime smoke, both directions), clipcmd exec paths (run_test.go), the copy_command chain (copy-on-select-test drives the real binary through stubs), and the session-watch/restart-quit smokes. The single uncovered surface is the two `spawnDetached` argv strings in `osruntime.go` — a silent-failure path. If cheap, extracting the `filepath.Join(PairHome, "bin", name)` argv construction into a pure helper and asserting the base name (`pair-title`, not `pair-title.sh`) would make that regression visible; it's a judgment call whether that's worth a helper for a two-line join.

**6. Architectural notes.**
- **ARCH-DRY — pass.** The bundle stays single-sourced through `explicitAssetPaths`; no duplicated caller logic introduced; the five repoints are uniform `.sh`-suffix drops. M1's `runRestart`/`runQuit` reuse the existing marker seam (no second fake, no new Runtime methods).
- **ARCH-PURE — pass.** `parseRestart` is pure `[]string→(LaunchArgs,error)`; `runRestart`/`runQuit` are thin glue over the injected `Runtime`; clipcmd keeps decision logic in `RunCopyOnSelect` with all IO behind the `Runtime` seam. The direct-spawn change touches only the thin `OSRuntime` boundary.
- **ARCH-PURPOSE — pass.** Shadow-sweep of the consumers: launcher spawns (2), zellij `copy_command` (1), clipcmd internal execs (2) all repointed; `explicitAssetPaths` + regenerated manifest + `files/` all drop the seven shims; nvim keybinds (M1) invoke `{'pair','quit'|'restart'}`. No hand-maintained restatement of the removed surface remains. The purpose ("stop extracting shell") is delivered, not a subset — the kept six utilities are a documented, deliberate out-of-scope, not a deferral of the point.

**7. Plan revision recommendations.** None — the plan still matches the code (every Core-Concepts M2 row verified at its stated file:line). Optional, low-value: when this closes, a one-liner in the plan/atlas noting the transient double-poller-across-upgrade edge (already called out in "Notes / risks") self-heals, so no follow-up issue is owed.
