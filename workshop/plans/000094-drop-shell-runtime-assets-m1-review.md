# Boundary Review — pair#94 (milestone M1)

| field | value |
|-------|-------|
| issue | 94 — stop extracting shell scripts from runtime bundle |
| repo | pair |
| issue file | workshop/issues/000094-drop-shell-runtime-assets.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 2ca2f60fa6f9acccc5fcd34b75926b09f4fb67cc^..HEAD |
| command | sdlc milestone-close --issue 94 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-07-03T11:56:20-07:00 |
| verdict | SHIP |

## Review

I have everything I need. Routing is confirmed end-to-end: invoked as public `pair`, `ClassifyInvocation` → `ModePublicPair` → `LaunchNative`, which holds the restart/quit branches — and the passing smoke test proves the whole path produces correct markers through the real binary.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** This is a clean, byte-faithful port of `bin/pair-{restart,quit}.sh` into in-process `pair restart`/`pair quit` subcommands. I verified the port against the *deleted shell source* (not just the commit messages): session-empty→exit-1, `tag = strip "pair-"`, marker write order (restart-then-quit), `mkdir -p` preserved (both `Touch` and `WriteAtomic` `MkdirAll`), and the marker format round-trips (the `new_session=0` omission is inert because `parseRestartMarker` treats absent as false). The reuse claim holds — no new Runtime methods, no second fake, `runRestart`/`runQuit` are genuine analogs of `runCompaction`. Unit tests (parse + fake-runtime markers) and the process-level smoke both pass; the bundle was actually regenerated (manifest clean, `files/` copies gone, `embed_test` green); atlas + nvim + `explicitAssetPaths` are all updated in-range. Nothing blocks the boundary; all findings are Minor.

**1. Strengths**
- Genuine seam reuse, not duplication — `runRestart`/`runQuit` (`restart.go:15,35`) drive the existing `WriteRestartMarker`/`TouchQuitMarker`/`ExecKillSession`/`InferAgent`, and the tests reuse `createflow_test.go`'s `fakeRuntime` rather than defining a second fake (ARCH-DRY, clean).
- The silent-misroute trap is closed *and pinned*: explicit `case "restart"`/`case "quit"` in `ParseArgs` (`args.go:66-69`), with `TestParseRestart`/`TestParseQuit` asserting `Agent == ""` — exactly the regression that would otherwise turn `restart` into an agent name.
- Real end-to-end coverage: `tests/pair-restart-quit-test.sh` drives the built binary with `PAIR_KILL_CMD=true` and asserts marker contents with `grep -qx` (whole-line), plus the negative "quit writes no restart marker" and "missing session → exit 1, no marker" cases.
- Docs are thorough and honest — `atlas/architecture.md` reframes every `pair-{quit,restart}.sh` mention to the subcommand, and `go-migration-inventory.md` records the ported disposition with the "marker protocol unchanged" caveat.

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**
- `restart.go:23` — `InferAgent(tag)` is a slightly *broader* agent resolution than the shell's `cat agent-<tag>` (it falls back to the `config-<tag>-*.json` filename when `agent-<tag>` is empty/missing). In the restart flow `agent-<tag>` is always present (written at launch, removed only in cleanup which runs *after* the keybind fires), so the fallback is effectively unreachable and, when it does fire, strictly fills an otherwise-empty `agent=` — never contradicts. Harmless, but it's an undocumented deviation from the byte-faithful shell; a one-line note in the atlas ("`InferAgent` fallback is broader than the shell's `agent-<tag>`-only read; safe because it only fills the empty case") would prevent a future reader mistaking it for accidental drift.
- `tests/pair-restart-quit-test.sh:16` — the smoke pre-creates `$MARK` (`mkdir -p`), so it does **not** exercise the auto-`MkdirAll` in `Touch`/`WriteAtomic`. The `mkdir -p` is redundant with the code path it's meant to protect; dropping it (the file's own comment already says "WriteAtomic/Touch MkdirAll it too") would make the smoke actually catch a regression where the cache-dir creation is lost.
- `restart.go`/`runcli.go:64` — stderr wording changed `pair-restart:`→`pair restart:`. Cosmetic and consistent with the new naming; invisible to nvim's fire-and-forget `vim.fn.system`. Noted only for completeness.
- Shell's defensive `""`-arg skip in `pair-restart.sh` has no Go equivalent (`parseRestart` would reject `""` as unknown). nvim never emits an empty arg (`rename_to` is guarded before append), so this dead-in-practice branch is correctly not ported.

**5. Test coverage notes.** Coverage matches the bug classes this diff could ship: misroute (unit), marker content/order (unit + smoke), quit-writes-no-restart (both), missing-session exit code (both). The only untested behavior is fresh-HOME cache-dir auto-creation, and only because the smoke pre-creates the dir — verified correct by reading `osfs.go:35,77`. Consider the `mkdir -p` removal above to make that path load-bearing in a test.

**6. Architectural notes.**
- ARCH-DRY — **pass.** Seam and fake are reused; `RestartMarker`/`serializeRestartMarker` untouched; the three lifecycle runners (`runQuit`/`runRestart`/`runCompaction`) share the marker *seam* but differ in orchestration (source of tag, park step, marker payload), so extracting a shared helper would be over-engineering, not consolidation.
- ARCH-PURE — **pass.** `parseRestart` is a pure `[]string→(LaunchArgs,error)` tested without IO; `runRestart`/`runQuit` are thin glue over the injected `Runtime`, tested against the fake. Clean pure-core / thin-IO-shell split.
- ARCH-PURPOSE — **pass for M1.** Shadow-sweep of the two ported shims' consumers: both nvim keybinds repointed (`init.lua:3185,3288`), both dropped from `explicitAssetPaths` and the regenerated bundle, `embed_test` excluded-list tightened, `.sh` files deleted. The five exec-shims correctly remain for M2 — the diff does **not** prematurely touch `SpawnSessionWatcher`/`SpawnTitlePoller` (still `.sh` at `osruntime.go:276,281`), so the M1 window is scoped exactly to its purpose with no creep.
- Upcoming M2 watch-item (not this boundary): the plan repoints those two spawn sites and the `copy-on-select` chain; the `embed_test` `want`-list edits called out in the plan (removing `bin/pair-title.sh`/`bin/pair-session-watch.sh` from `want`) are the easy-to-miss step there.

**7. Plan revision recommendations.**
- None required — the plan still matches the code. Optional: add a one-line note to the plan's "Notes / risks" (and mirror in `atlas/go-migration-inventory.md`) recording the intentional `InferAgent`-broadening-vs-shell divergence, so it reads as a deliberate choice rather than drift.

Note on verification scope: I ran `go test ./cmd/internal/launcher/ ./cmd/internal/runtimebundle/` (green) and the restart/quit smoke (PASS), and confirmed the regenerated manifest/`files/`. I did **not** run the full `make test` — the `runtimebundle-drift-check` target writes to `/` via its generator and is blocked by this environment's sandbox (an env limitation, not a code fault); the direct manifest+files evidence covers what drift-check would assert.
