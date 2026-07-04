# Go Migration Inventory

Issue: #73. Parent roadmap: #72.

This document is the authoritative migration contract table for Pair's move
toward a primary Go `pair` binary. `atlas/architecture.md` remains the narrative
map of how Pair works; this file owns the artifact/caller/runtime/disposition
facts needed by #74-#79.

## Scope

Covered:

- installed or runtime-called artifacts under `bin/`, `bin/lib/`, `cmd/`,
  `nvim/`, and `zellij/`;
- `Makefile`, `Makefile.local`, install/packaging docs, and process-level
  test seams that protect these artifacts;
- hidden callers, especially zellij KDL commands and nvim shell-outs.

Out of scope:

- porting behavior;
- changing public command behavior;
- removing adjacent asset packaging while Homebrew/source layouts still rely on
  it.

## Single-Binary Deployment Path

#79 made the installed public `pair` command Go-owned. #90 added a
self-contained deployment mode: the Go binary embeds the Pair-owned runtime
assets and extracts them to `$PAIR_DATA_DIR/runtime/<digest>/pair-home` on
demand when no adjacent/source/Homebrew asset root is available.

This is not the same as "no external dependencies." The single-binary target is
one Pair artifact. System programs such as `zellij`, `nvim`, clipboard tools,
`fzf`, `jq` while retained shell needs it, and agent CLIs remain external unless
a later issue explicitly replaces them.

Execution path:

1. **Embedded runtime bundle (#90):** the current Pair-owned runtime tree
   (`bin/pair-shell`, shell helpers, helper binaries or dispatcher shims,
   `bin/lib/`, `nvim/`, `zellij/`, and doctor/help assets needed at runtime) is
   generated into a manifest-backed embedded bundle. On run, copied binaries
   extract to a digest-named Pair data root, write a runtime marker, prune stale
   older extracted runtimes without deleting the selected digest, and set
   `PAIR_HOME` there before the existing launch handoff.
2. **Dispatcher consolidation:** move helper binaries behind `pair <subcommand>`
   routes and leave old command names as generated compatibility shims only
   where native callers still need them.
3. **Go-owned orchestration:** port stateful shell orchestrators into Go in
   dependency order: launch/session lifecycle, scrollback and changelog openers,
   title poller, review helpers, clipboard helpers, then small quit/restart/help
   shims.
4. **Native single binary:** once shell ownership is gone, stop extracting shell
   scripts. Keep only the native assets that external runtimes require, such as
   Neovim Lua and Zellij KDL, either embedded-and-extracted or generated at
   startup. **#95 decided the endpoint:** the digest-versioned extraction is
   *kept* but reframed as a content-addressed runtime *cache* (deterministic
   digest, idempotent skip-unchanged writes, upgrade-safe + self-pruning keep-2)
   confined to the **copied-binary layout** — source and Homebrew point the asset
   root at an adjacent real tree and never extract. True zero-tree is unreachable
   while `nvim`/`zellij` stay native (they read config from real filesystem paths;
   `nvim/init.lua` `dofile()`s siblings by absolute path, needing a real directory
   that persists for the session — `embed.FS` is not a path), so the endpoint is
   one Pair *executable* + a self-provisioned config cache + external platform
   tools, not literally zero bytes on disk (the accepted, documented residual
   gap).

**Tracking (#91).** The remaining path is carried by roadmap #91 (native single
binary), one sub-ticket per step, deps-chained in order: #90 (embedded bundle,
done) → #92 (dispatcher consolidation) → #93 (Go-owned orchestration) → #94 (stop
extracting shell scripts) → #95 (native nvim/zellij startup assets). #94/#95
together are execution-path step 4 above; **#95 reaches that endpoint** — a native
single *executable* + a content-addressed config cache for the two external tools +
system platform tools, with the residual zero-tree gap documented as an accepted
limitation (see step 4 and the #95 milestone note below).

`ARCH-PURPOSE`: #90 is only complete if a copied binary can supply all Pair-owned
runtime assets without falling back to a source checkout. `ARCH-DRY`: use one
runtime manifest for embedding, extraction, install verification, and package
metadata. `ARCH-PURE`: keep manifest planning and runtime selection testable as
pure functions, with filesystem writes confined to thin seams.

## Disposition Vocabulary

- **go-subcommand**: should route through the future Go dispatcher as
  `pair <subcommand>`.
- **go-entrypoint**: should become Go-owned public `pair` behavior.
- **compat-shim**: keep the old command name temporarily while callers migrate.
- **native-asset**: should remain Lua/KDL or another native runtime asset.
- **adjacent-asset**: should be packaged beside, or eventually embedded by, the
  primary Go binary.
- **shell-glue**: shell may remain if it is small platform glue; stateful shell
  should be revisited after the Go entrypoint exists.
- **test-only**: test seam/fake/driver, not a shipped runtime artifact.

Priority is packaging impact first, then reliability/testability:

- **P0**: blocks the single-primary-binary route or public entrypoint switch.
- **P1**: reduces installed binary/script surface or stateful shell risk.
- **P2**: native asset or compatibility wrapper that packaging must account for.
- **P3**: test/doc seam used to verify migration but not migrated itself.

## Artifact Inventory

| Artifact | Type | Callers | Runtime contract | Files/env | Disposition | Priority |
|---|---|---|---|---|---|---|
| `bin/pair` / `bin/pair-shell` / `cmd/internal/launcher` / `cmd/internal/entrypoint` / `cmd/internal/runtimebundle` | Go public entrypoint plus retained shell launcher and embedded runtime fallback | user shell, copied-binary installs, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` is generated from `cmd/pair-go` and resolves `PAIR_HOME` / sibling root / build-time `defaultPairHome`; if none exists, it extracts the embedded runtime to `$PAIR_DATA_DIR/runtime/<digest>/pair-home`; then it execs `<asset-root>/bin/pair-shell` with `pair`-compatible argv/env and `PAIR_HOME` pointed at the selected root. `bin/pair-shell` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` shares the same compatibility handoff. | `bin/pair-shell` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves invocation mode, asset root, and compatibility request; `cmd/internal/runtimebundle` owns manifest hashing, extraction planning, runtime markers, and stale-runtime cleanup; `cmd/internal/launcher` keeps the fakeable pure decision core from #75; **#99 M1** extended it with the pure per-agent launch-arg helpers (resume-token compose, codex `--no-alt-screen` idempotence, claude `--session-id` mint/skip decision, the consolidated flag strip/has helpers), config-path + legacy-codex migration decision, and age/title formatting; **#99 M2** adds the `launcher.Runtime` effect seam (composed sub-interfaces for fake-test ISP), the `RunLaunch` **create-flow** orchestrator + concrete `OSRuntime`, wired behind `PAIR_NATIVE_LAUNCH` in `cmd/pair-go` as a create-only preview (attach/pick, in-pane launches, and unsupported verbs return `ErrFallbackToShell` → shell). The blocking zellij handoff is fork+wait so the launcher regains control for M3's quit/restart. **#99 M3** adds the `LifecycleOps` sub-interface + turns `RunLaunch` into the **in-process restart loop** (replaces `exec $0`): native attach (`AttachSession`), quit cleanup (`cleanup_quit_marker` port — delete-session + nvim reap + gated park-nudge + sidecar removal + resume hint + poller kill + cmux reset), and Alt+n/Shift+Alt+N restart via the pure `planRestart`; the fzf pick, in-pane compaction, and rename/continue restart re-entries still return `ErrFallbackToShell`. **#99 M4** is the **cutover**: the native launcher is now the DEFAULT (behind the `cmd/pair-go` `legacyRuntime` seam so the flip is unit-testable), gated only by a `PAIR_LEGACY_LAUNCH=1` kill-switch replacing the opt-in `PAIR_NATIVE_LAUNCH`; a decline (pick/compaction/continue+rename restart + shell-owned `--help`/flags, which `ParseArgs` now refuses as agents) still falls through to the **real** `bin/pair-shell` — retained as the fallback launcher, NOT shimmed (a shim would loop). **#99 M5** ports the last flows native then retires the shell, split into three risk boundaries: **M5a** makes the fzf session **pick** native (`resolvePick` over the pure `buildPickRows`; `HistoricalTag` now carries `MTime`+`QueueCount` so the picker rows are pure, only the fzf call is an effect — reusing `UIOps.PickFromList` with `--ansi`) and adds the **`list`/`ls`** subcommand (`ListOps.ListSessions` + the pure `formatListTable` → stdout); a picked existing tag is resume-by-name (agent inferred from the tag). **M5b** makes the lifecycle write flows native: in-session **compaction** (the `InZellijPane` guard → native branch: pure `compactionDecision` honoring the force/fake/kill seams + park-copy + `WriteRestartMarker`/`TouchQuitMarker`/`ExecKillSession`), the **`continue`** subcommand (`ContinuationOps` bare-list + slug-resolve, reusing `continuationcmd.ContinuationDir`/`NextActionPreview`), the offline **`rename`** subcommand (pure `renamePathsFor`/`renamePlan` zip + `FSOps.Rename` + journal/rollback), and the **`rename_to`/`continue` restart re-entries** (`planRestart` drops `ShellFallback`) — closing the M4-accepted degradation; `PAIR_DEBUG_ARGS`/`PAIR_DEBUG_HISTORY` join `PAIR_TEST_CALL` as shell-routed seams. **#99 M5c RETIRES the shell**: `bin/pair-shell` is **removed**; `--help`/`help` print a native `UsageText`; the `cmd/pair-go` fallback arm is deleted (`PAIR_LEGACY_LAUNCH` + the `Exec` seam + `ErrFallbackToShell` + `shellOnlySeamActive` all gone — `LaunchNative` always returns a real exit code, defensive errors print+exit); the asset-root validity marker moves to `zellij/layouts/main.kdl` (`entrypoint.ValidRootMarker`; `launch.go` deleted); `bin/pair-shell` dropped from the runtime bundle; the `PAIR_TEST_CALL`/`PAIR_DEBUG_*` shell contract tests (`pair-continue-test`, `cmux-ownership-test`) retire (Go equivalents tested). **#94 M1** then ported the last two nvim-keybind marker-writers `bin/pair-restart.sh`/`pair-quit.sh` to in-process `pair restart`/`pair quit` subcommands (`runRestart`/`runQuit` in `cmd/internal/launcher/restart.go`, parsed by `parseRestart`), reusing the launcher's `WriteRestartMarker`/`TouchQuitMarker`/`ExecKillSession`/`InferAgent` seam (marker protocol unchanged) and removing the two `.sh` from tree + runtime bundle. | **`bin/pair-shell` is retired (#99 M5c)** — `pair` is a single Go launcher end-to-end (public entrypoint Go-owned since #79, embedded fallback since #90); this unblocks #94 | P0 |
| `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling Go-built `pair`. | Resolves symlinks; depends on generated `bin/pair`, retained `bin/pair-shell`, and `bin/lib/dev-rebuild.sh`. | retained dev wrapper so developer launches exercise the public Go entrypoint | P1 |
| `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair-shell` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
| `zellij/layouts/main.kdl` | zellij native asset | `bin/pair-shell` via `zellij --new-session-with-layout` | Defines agent and draft panes; shell expands Pair env at pane start. | Calls `pair-wrap`; calls `nvim -u "$PAIR_HOME/nvim/init.lua"`; writes `pane-<tag>-<agent>.json`; writes draft nvim pid file. | native-asset, packaged adjacent/embedded | P0 |
| `zellij/config.kdl` | zellij native asset | zellij session config from `bin/pair-shell` | Global keybinds, copy command, scrollback buffer, pane frames. | Calls `copy-on-select`, `pair-help`, `pair-scrollback-open`, `pair-changelog-open`; routes quit/restart/compact through nvim functions. | native-asset, packaged adjacent/embedded | P0 |
| `bin/pair-wrap` / `cmd/pair-wrap` / `cmd/internal/wrapcmd` | Go binary plus shared runner | zellij agent pane | `pair-wrap [--scrollback-log PATH] agent [args...]`; transparent PTY proxy; long-running; failure in diagnostics is swallowed. | Reads Pair env and agent command; writes `agent-output-<tag>`, `agent-pid-<tag>`, scrollback `.raw`/`.events.jsonl`, image capture files; may invoke `pair slug`. | implemented `pair wrap` route (#96, streaming seam — real stdio via `wrapcmd.Run`); `cmd/pair-wrap` is a thin shim over `cmd/internal/wrapcmd`; KDL still execs `pair-wrap` by PATH | P0 |
| `bin/pair-slug` / `cmd/pair-slug` / `cmd/internal/slugcmd` | Go binary plus shared runner | `pair-wrap` turn-end hook (now `pair slug`), tests | Env-driven, no stdin; resolves native transcript, proposes slug; exits 0 on most failures. | Requires `PAIR_TAG`, `PAIR_DATA_DIR`; reads config/transcripts/git branch; writes `slug-proposed-<tag>`; optional `PAIR_SLUG_*`, `OPENAI_API_KEY`. | implemented `pair slug` route (#92, buffered `slugcmd.Run`); `bin/pair-slug` retained as thin shim | P1 |
| `bin/pair-context` / `cmd/pair-context` / `cmd/internal/contextcmd` | Go binary plus shared runner | `cmd/internal/titlepoller` (in-process, #93 M1); development-only `pair-go context` | `pair-context <tag> <agent>` and `pair-go context <tag> <agent>` print the same humanized token count or nothing; tolerant exit 0 on failure. Exposes `TranscriptPath` for the shared transcript resolution. | Reads `PAIR_DATA_DIR`, `pane-<tag>-<agent>.json`, config, native transcripts. | implemented `pair context` route; the title poller now calls `contextcmd.Run`/`TranscriptPath` **in-process** (#93 M1, no subprocess); `bin/pair-context` retained as thin shim | P1 |
| `bin/pair-scrollback-render` / `cmd/pair-scrollback-render` / `cmd/internal/scrollbackcmd` | Go binary plus shared runner | `cmd/pair-scrollback-open` (in-process, #93 M2), `cmd/pair-changelog-open`'s detached distiller, `nvim/scrollback.lua` refresh; development-only `pair-go scrollback-render` | `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out` and `pair-go scrollback-render ...`; nonzero on render/write failure. | Reads `.raw` and `.events.jsonl`; atomically writes `.ansi` or cleaned text. | implemented `pair scrollback-render` route (#92); the Alt+/ opener now calls `scrollbackcmd.Run` **in-process** (#93 M2, no subprocess); the changelog opener's detached distiller + `nvim/scrollback.lua` still shell `pair scrollback-render`; `bin/pair-scrollback-render` retained as thin shim | P0 |
| `bin/pair-changelog` / `cmd/pair-changelog` / `cmd/internal/changelogcmd` | Go binary plus shared runner | `bin/pair-changelog-open` (now `pair changelog`) | `pair-changelog --cleaned F --log F --anchor F [--agent A] [--model M]`; exits nonzero on required read/model/write failure. | Reads cleaned scrollback/log/anchor; calls agent model through internal model runner; atomically writes log and anchor. | implemented `pair changelog` route (#92, streaming seam — live per-batch stderr spinner); `bin/pair-changelog` retained as thin shim | P1 |
| `bin/pair-continuation` / `cmd/pair-continuation` / `cmd/internal/continuationcmd` | Go binary plus shared runner | nvim compaction prompt instructions, operator/agent shell | `pair-continuation --slug S --agent A --issues CSV --body-file F [--repo-root R ...]`; writes and commits continuation; nonzero on validation/git failure. | Reads body/stdin, git repo state; writes `workshop/continuation/*.md`; runs git commit/push. | implemented `pair continuation` route (#92, streaming seam — reads body from stdin); `bin/pair-continuation` retained as thin shim; no repointed production caller yet (agent-procedure invoked) | P1 |
| `bin/pair-scribe` / `cmd/pair-scribe` / `cmd/internal/scribecmd` | Go binary plus shared runner | user shell rc outside Pair sessions | `pair-scribe -log PATH -- CMD [ARGS...]` and `pair scribe …`; long-running PTY wrapper; SIGUSR1 pauses log, SIGUSR2 resumes. | Writes typescript log; wraps child PTY; independent of `PAIR_*`. | implemented `pair scribe` route (#96, streaming seam — `scribecmd.Run`); `cmd/pair-scribe` is a thin shim so `~/.local/bin/pair-scribe` + the user's `~/.zshrc` wiring keep working; NOT in the runtime bundle (user shell tooling, not runtime) | P2 |
| `cmd/internal/adapt` | Go helper package | `cmd/internal/wrapcmd` (pair-wrap), `pair-slug`, tests | Pure-ish emitter helpers plus file open seam; no command. | Writes `$PAIR_DATA_DIR/adapt-<tag>.jsonl`; schema shared with shell/Lua. | internal package, reuse behind dispatcher | P1 |
| `cmd/internal/ctxmeter` | Go helper package | `pair-context`, tests | Pure transcript token counting and humanization. | No direct IO. | internal package, keep | P1 |
| `cmd/internal/model` | Go helper package | `pair-slug`, `pair-changelog`, tests | Model runner/response parsing. | Calls external agent/model CLIs/APIs at command layer. | internal package, keep | P1 |
| `cmd/internal/transcript` | Go helper package | `pair-slug`, `pair-context`, tests | Resolves native transcript paths and session ids. | Reads Pair config and home paths via callers. | internal package, keep | P1 |
| `cmd/pair-scrollback-open` / `cmd/internal/opener` | Go binary plus shared runner | zellij Alt+/ Run, nvim Alt+b jump | `pair-scrollback-open [--jump prev|next]`; opens read-only nvim viewer; singleton lock. | Requires `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_HOME`; renders in-process (`scrollbackcmd`), zellij IPC (list-panes/dump-screen), nvim; writes `.ansi`, `.viewport`, lock. | ported to Go (#93 M2) on the #78 template — pure viewport scorer in `opener`, IO behind the `Runtime` seam; **replaces** the shell script at the same PATH name (zellij invokes by name → no shim); `nvim/scrollback.lua` stays native | P1 |
| `nvim/scrollback.lua` | Neovim native asset | `cmd/pair-scrollback-open` | Loaded by `nvim -u ... <ansi>`; interactive read-only viewer; refreshes backing render. | Reads Pair env and `.ansi`; may call `pair-scrollback-render`; writes pending marker files. | native-asset, adjacent/embedded | P0 |
| `cmd/pair-changelog-open` / `cmd/internal/opener` | Go binary plus shared runner | zellij Alt+l Run | Opens changelog viewer and starts detached render/distill singleton. | Requires Pair env; launches a `setsid`-detached `pair scrollback-render` / `pair changelog` build (#92), nvim watcher; reads/writes `changelog-*` sidecars. | ported to Go (#93 M2) — shared `opener` package (session keying + detached distiller), IO behind the seam; **replaces** the shell script at the same PATH name (no shim); Go `SysProcAttr.Setsid` replaces setsid/perl | P1 |
| `nvim/changelog.lua` | Neovim native asset | `cmd/pair-changelog-open` | Loaded by `nvim -u ... <log>`; read-only watcher/spinner. | Reads `PAIR_CHANGELOG_*` and Pair env. | native-asset, adjacent/embedded | P1 |
| `bin/pair-title` / `cmd/pair-title` / `cmd/internal/titlepoller` | Go binary plus shared runner | launcher `SpawnTitlePoller` | `pair-title <tag> <agent>`; long-running 60s poller (frame meter + cmux heat-ramp). | Reads/writes title pid, pane json, cmux owner files; calls zellij/cmux/ps + in-process `contextcmd` for the count. | ported to Go (#93 M1) on the #78 sessionwatch template — pure decisions in `titlepoller`, IO behind the `Runtime` seam; the `.sh` re-exec shim was retired in #94 M2 (the launcher spawns `bin/pair-title` directly) | P1 |
| `bin/pair-session-watch` / `cmd/pair-session-watch` / `cmd/internal/sessionwatch` | Go stateful watcher | launcher `SpawnSessionWatcher` (create path) | `pair-session-watch <agent> <tag> <cwd> [agent-args...]`; background 60s watcher; no-op for claude. | Reads agent pidfile, lsof/ps, native session dirs; writes config JSON atomically; logs adapt events through `cmd/internal/adapt`. | Go-owned watcher with implemented `pair session-watch` route (#92, via `sessionwatch.RunCLI`); the `.sh` passthrough shim was retired in #94 M2 (the launcher spawns `bin/pair-session-watch` directly) | P1 |
| `bin/lib/adapt-log.sh` | sourced shell helper | remaining shell emitters | `adapt_log comp agent aspect signal outcome [detail]`; no-op if no `PAIR_TAG` or jq. | Appends JSONL to `$PAIR_DATA_DIR/adapt-<tag>.jsonl`. | keep until remaining shell emitters move; schema stays DRY with Go/Lua emitters | P1 |
| `nvim/adapt.lua` | Lua helper | nvim doctor/adaptation surfaces, tests | Lua adaptation flight recorder emitter. | Writes same JSONL schema as Go/shell. | native-asset; keep schema aligned | P2 |
| `doctor/README.md` / `doctor/SKILL.md` | docs/skill | operator/agent diagnostics | Documents Pair doctor flow. | Refers to `nvim/doctor.lua` and adaptation logs. | adjacent docs/skill; not Go migration target | P3 |
| `nvim/doctor.lua` | Lua helper | `:PairDoctor` in nvim | Builds agent instruction payload. | Reads `PAIR_HOME`; sends text through draft/agent flow. | native-asset | P2 |
| `bin/pair-notify` | Bash notification helper | agent hooks/manual shell inside Pair | `pair-notify [--osc 9|777] "message"`; writes OSC to outer tty; nonzero on bad args/missing tty. | Requires `PAIR_TAG`; reads `outer-tty-<tag>`. | small shell-glue; possible Go subcommand but low packaging impact | P2 |
| `pair quit` (was `bin/pair-quit.sh`; `cmd/internal/launcher/restart.go`) | in-process Go subcommand, ported from a Bash keybind helper | nvim `PairConfirmQuit` (`{ 'pair', 'quit' }`) | Touch quit marker then kill zellij session. | Uses `ZELLIJ_SESSION_NAME`, `PAIR_KILL_CMD`; writes cache marker. | **ported in #94 M1** — `runQuit` reuses the launcher's `TouchQuitMarker`/`ExecKillSession` seam; `.sh` removed from tree + runtime bundle | P2 |
| `pair restart` (was `bin/pair-restart.sh`; `cmd/internal/launcher/restart.go`) | in-process Go subcommand, ported from a Bash keybind helper | nvim restart confirmations (`{ 'pair', 'restart', ... }`) | Writes restart marker then kill zellij session; supports `--new-session` / `--rename-to <tag>`. | Uses `PAIR_TAG`, `PAIR_AGENT`, `ZELLIJ_SESSION_NAME`, cache marker files. | **ported in #94 M1** — `runRestart` reuses `WriteRestartMarker`/`TouchQuitMarker`/`ExecKillSession`/`InferAgent` (marker protocol unchanged); `.sh` removed from tree + runtime bundle | P2 |
| `bin/pair-help` | Bash helper | zellij Alt+h Run | Displays `pair -h` through `less` with escape-to-quit behavior. | Calls `pair`, `less`. | compat-shim; may become `pair help` behavior | P2 |
| `cmd/clipboard-to-pane` / `cmd/internal/clipcmd` (`bin/clipboard-to-pane`) | Go binary + shared runner | `copy-on-select` (execs `bin/clipboard-to-pane` directly), direct zellij run possible | Reads OS clipboard, stages quote at `$PAIR_DATA_DIR/quote-<tag>`, focuses the nvim pane, triggers PairPasteQuote via Ctrl-_. | Uses pbpaste/wl-paste/xclip, zellij, `PAIR_DATA_DIR`, `PAIR_TAG`; nvim-pane pick via `cmd/internal/zellijpane`. | ported to Go (#93 M4); IO behind the `clipcmd.Runtime` seam (embeds `osfs.FS`); the `.sh` re-exec shim was retired in #94 M2 (copy-on-select execs `$PAIR_HOME/bin/clipboard-to-pane` directly) | P2 |
| `cmd/copy-on-select` / `cmd/internal/clipcmd` (`bin/copy-on-select`) | Go binary + shared runner | `zellij/config.kdl` `copy_command "copy-on-select"` | Reads selected text stdin, mirrors OS clipboard, flashes source pane, hands off (execs) to clipboard-to-pane — unless the focused pane was the nvim draft (in_nvim gate on `terminal_command`, not title). | Uses pbcopy/wl-copy/xclip, zellij, `PAIR_HOME`; focused-pane pick via `cmd/internal/zellijpane`; execs the flash/clipboard Go binaries directly. | ported to Go (#93 M4); the `.sh` shim was retired in #94 M2 (zellij's `copy_command "copy-on-select"` invokes `bin/copy-on-select` by name) | P2 |
| `cmd/flash-pane` / `cmd/internal/clipcmd` (`bin/flash-pane`) | Go binary + shared runner | `copy-on-select` (execs `bin/flash-pane` directly); the copy-on-select shell test | `flash-pane [pane-id]`; best-effort pane color flash — synchronous fg set + detached (setsid) bg reset so it doesn't block the caller. | Uses zellij; reads `PAIR_FLASH_*`; focused-pane pick via `cmd/internal/zellijpane`. | ported to Go (#93 M4); the `.sh` re-exec shim was retired in #94 M2 (copy-on-select execs `$PAIR_HOME/bin/flash-pane` directly) | P3 |
| `cmd/pair-review-open` / `cmd/internal/reviewcmd` | Go binary plus shared runner | nvim review flow | Validates target and opens floating `nvim -u nvim/review.lua` (single review pane). | Requires Pair env; calls zellij/nvim; kills the prior review nvim. | ported to Go (#93 M3); IO behind the `Runtime` seam; **replaces** the shell script at the same PATH name (no shim) | P2 |
| `cmd/pair-review-readiness` / `cmd/internal/reviewcmd` | Go binary plus shared runner | `nvim/init.lua` review readiness | Gathers git facts, classifies via `nvim/review/readiness.lua`, emits JSON or performs `--prepare` git effects + marks ready. | Uses `PAIR_HOME`, git, `nvim --headless` classify. | ported to Go (#93 M3); the 4-case decision stays in `readiness.lua` (single source, invoked via `nvim --headless`); replaces the shell script (no shim) | P2 |
| `cmd/pair-review-target` / `cmd/internal/reviewcmd` | Go binary plus shared runner | review readiness/open/tests | Writes JSON target metadata under data dir, session-stamped. | Requires `PAIR_DATA_DIR`; reads config; codex fallback via `cmd/internal/codexsid` (ps/lsof); writes `review-target-<tag>.json`. | ported to Go (#93 M3); session resolution reuses `transcript`-style config read + the extracted `codexsid` walk; replaces the shell script (no shim) | P2 |
| `nvim/init.lua` | Neovim native asset | zellij draft pane | Main draft UI and Pair key handling. | Reads many Pair env vars/data files; shell-outs to zellij, pair quit/restart/open/review helpers. | native-asset; do not port, but audit shell-outs during #77/#78 | P0 |
| `nvim/review.lua` and `nvim/review/*.lua` | Neovim native review workbench | `pair-review-open`, draft review toggle | Review pane UI/modules. | Reads Pair env/data; calls docflow/agent seams through shell tests. | native-asset; adjacent/embedded | P2 |
| `nvim/annotate.lua`, `nvim/marker_codec.lua`, `nvim/pair_poke.lua`, `nvim/slug.lua`, `nvim/zellij_trace.lua` | Lua native helper modules | draft/viewer/review Lua | Pure or thin Lua helpers used by nvim surfaces. | Pair env/data files; zellij shell-outs in poke/trace surfaces. | native-asset | P2 |
| `Makefile` | build/workflow entry | developer/CI/SDLC | Includes workflow and local makefiles; `help` target. | Uses git remote; includes vendored base fragments. | packaging metadata; keep, update in #79 if install layout changes | P1 |
| `Makefile.local` | build/install/test metadata | developer/CI/`pair-dev` | Builds and installs Go binaries, runs test matrix. | Writes `bin/` and `~/.local/bin`; invokes Go, nvim, shell tests. | build contract; #74/#76/#79 must update as dispatcher changes | P0 |
| `README.md` | install/usage docs | users/package consumers | Homebrew install, CLI usage, keybindings, dev mode. | Documents dependencies and public commands. | docs to update at #77/#79 | P1 |
| `cmd/pair-scribe/README.md` | helper docs | users configuring shell logging | Documents `pair-scribe`/`pair scribe` install/usage; kept next to the shim. | No runtime behavior. | docs; `pair scribe` route added in #96 (README notes the route + shim) | P3 |
| `tests/*.sh`, `tests/lib/*`, `nvim/*_test.lua`, `cmd/**/*_test.go` | test-only seams | `make test`, Go test, headless nvim | Process fakes, shell integration tests, headless nvim drivers, Go unit tests. | Create temp dirs/fake PATH commands; exercise real scripts/binaries/Lua modules. | test-only; not migrated, but must move with callers | P3 |

## Hidden Caller Map

Zellij KDL callers:

- `zellij/layouts/main.kdl` launches `pair-wrap` by PATH and `nvim -u
  "$PAIR_HOME/nvim/init.lua"` by absolute env path.
- `zellij/config.kdl` calls `copy-on-select`, `pair-help`,
  `pair-scrollback-open`, and `pair-changelog-open` by PATH.
- Quit/restart/detach/compact keybinds route through nvim functions first, then
  those functions call shell helpers or zellij actions.

Nvim shell-outs and binary dependencies:

- `nvim/init.lua` calls zellij actions, `pair quit`, `pair restart`,
  `pair-scrollback-open`, `pair-review-readiness`, `pair-review-open`, and uses
  `pair-wrap` pidfiles for image capture.
- `nvim/scrollback.lua` refreshes via `pair-scrollback-render`.
- `nvim/changelog.lua` watches files prepared by `pair-changelog-open`.
- `nvim/review.lua` loads review modules and is launched by `pair-review-open`.

Build/install callers:

- `make runtimebundle-generate` refreshes the gitignored embedded runtime asset
  tree and manifest; `make runtimebundle-drift-check` verifies the generated
  bundle is reproducible.
- `make build` builds `GO_BINS` into `bin/`; `pair` and `pair-go` are both built
  from `cmd/pair-go` with `defaultPairHome=$(CURDIR)`, while copied builds with
  no adjacent/default root use the embedded fallback.
- `make install` copies `GO_BINS` to `~/.local/bin` and symlinks only retained
  shell wrappers such as `pair-dev`. Installed `pair` is a regular Go binary;
  if it has no sibling assets, it falls back to the build-time source root when
  that exists and otherwise extracts the embedded runtime.
- Homebrew installs `bin/`, `nvim/`, and `zellij/` under `libexec`, then builds
  Go `pair`, `pair-go`, and required runtime helpers into `libexec/bin` with
  `defaultPairHome=#{libexec}`.
- `make test-runtimebundle` runs bundle-generation-aware Go tests, and
  `make test-pair-embedded-runtime` exercises copied-binary launch plus stale
  runtime cleanup with fake external dependencies.
- `pair-dev` relies on `make build`, then zellij's PATH lookup resolves fresh
  repo `bin/` binaries.

## Migration Sequence Notes

- #74 should add a Go dispatcher without changing `bin/pair`.
- #75 and #76 can proceed in parallel after #74. The launcher prototype does not
  need helper dispatch to exist first, and helper dispatch does not need the
  launcher prototype.
- #76 established the first helper-dispatch pattern with `pair-go context` and
  `pair-go scrollback-render`, backed by shared internal runners while legacy
  binary names remain live for shell/Lua callers. `pair slug` remains a later
  candidate.
- #77 made `pair-go launch ...` a meaningful Go-owned compatibility handoff to
  the then-shell `bin/pair`, with argv/env preserved and missing-launcher
  diagnostics.
- #78 ported the session-id watcher to `cmd/pair-session-watch` with
  `bin/pair-session-watch.sh` retained as a shim (since retired in #94 M2 — the
  launcher spawns the Go `bin/pair-session-watch` directly).
- #93 M1 ported the title poller to `cmd/pair-title` + `cmd/internal/titlepoller`
  on that same template — pure decisions (heat buckets, cwd abbrev, frame title,
  argv identity guard, unchanged-skip cache) unit-tested directly; zellij/cmux/ps/fs
  behind the `Runtime` seam; the context count reused in-process via `contextcmd`
  (no `pair context` subprocess). `bin/pair-title.sh` became a thin re-exec shim
  (since retired in #94 M2 — the launcher spawns the Go `bin/pair-title` directly).
  A shared `cmd/internal/procutil` (`Alive`/`Command`) now backs both
  sessionwatch's and titlepoller's `OSRuntime`. The remaining stateful shell
  surfaces (scrollback/changelog openers, review helpers, clipboard helpers, the
  `bin/pair-shell` launcher) are #93 M2–M5.
- #93 M2 ported the two floating-pane viewer launchers to `cmd/pair-scrollback-open`
  + `cmd/pair-changelog-open`, sharing one `cmd/internal/opener` package: the pure
  viewport scorer (dump-screen ↔ rendered `.ansi` scroll-position match), session
  keying, and distiller argv are unit-tested directly; zellij IPC / nvim / the
  `setsid`-detached distiller (Go `SysProcAttr.Setsid`) / fs sit behind the seam.
  The Alt+/ opener renders in-process via `scrollbackcmd` (no subprocess); the
  Alt+l opener keeps a detached `pair scrollback-render`/`pair changelog` build so
  it survives the viewer closing. These **replace** the same-named shell scripts
  (zellij invokes by PATH → no shim; the two `.gitignore` negations were dropped).
  The existing `changelog-open` e2e tests now drive the Go binary unchanged.
- #93 M3 ported the three review-start orchestrators to `cmd/pair-review-target`
  / `-open` / `-readiness`, sharing `cmd/internal/reviewcmd`: pure slugify / JSON
  shapes / `--prepare` action-mapping are unit-tested; git effects, the
  `nvim --headless` classify bridge, the `zellij run --floating` spawn, and the
  codex session walk sit behind the `Runtime` seam. The 4-case readiness decision
  stays the single source in `nvim/review/readiness.lua` (invoked via
  `nvim --headless`). Two ARCH-DRY extractions landed with it: `cmd/internal/osfs`
  (the shared string-based fs primitives, now embedded by opener + titlepoller +
  reviewcmd `OSRuntime`s — sessionwatch stays byte-based/separate) and
  `cmd/internal/codexsid` (the ps/lsof rollout-uuid walk, canonical home for the
  copy slug/sessionwatch still carry). The Go binaries replace the shell scripts
  (no shim; 3 `.gitignore` negations dropped); the existing `pair-review-target` /
  `review-readiness-cli` / `review-window` shell tests drive them unchanged.
- #79 made public `pair` a Go-built entrypoint, renamed the shell launcher to
  `bin/pair-shell`, and chose adjacent `nvim/` / `zellij/` assets for local and
  Homebrew installs.
- #92 M1 continued the dispatcher consolidation: `slug`, `changelog`,
  `continuation`, and `session-watch` gained shared `cmd/internal/<name>cmd`
  runners (the #76 `contextcmd`/`scrollbackcmd` pattern) reachable as
  `pair <sub>`. `ClassifyInvocation` now peels the reserved dispatcher
  subcommands (`dispatcher.DispatchNames()`) off the public `pair` before the
  launcher handoff — `pair slug` dispatches while `pair claude`/`resume`/bare
  `pair` still launch. Finite/no-stdin `slug` uses the buffered `Dispatch`
  path; `changelog` (live stderr spinner), `continuation` (stdin), and
  `session-watch` (long-running) use a new streaming seam in `cmd/pair-go`
  (`runStreamingSubcommand`) that hands the runner real stdio. Each standalone
  `bin/pair-<name>` binary is now a thin shim over its runner.
- #92 M2 repointed the Pair-owned call-sites to `pair <sub>`: `bin/pair-title.sh`
  (`pair context`), `bin/pair-changelog-open` and `bin/pair-scrollback-open` and
  `nvim/scrollback.lua` (`pair scrollback-render` / `pair changelog`), and
  `cmd/pair-wrap`'s turn-end spawn (`pair slug`). (At the time, `bin/pair-title.sh`
  called `pair context`; #93 M1 later folded that count in-process via
  `contextcmd`, so the title poller no longer shells out to `pair context` — see
  the #93 M1 bullet below.) At that point the internal callers were still on a
  *shim name*, spawned by the then-shell launcher: `bin/pair-session-watch.sh`
  and (since #93 M1) `bin/pair-title.sh`; those spawns were later collapsed in
  #94 M2 (the launcher spawns the Go binaries directly). The standalone
  dispatcher helper binaries remain built + bundled as shims; dropping them is
  later single-binary work.
- #96 routed the two interactive PTY proxies through the same streaming seam:
  `pair wrap` (`wrapcmd.Run`) and `pair scribe` (`scribecmd.Run`), extracting the
  logic into `cmd/internal/wrapcmd` / `cmd/internal/scribecmd` behind
  `Run(args, stdin, stdout, stderr) int` and leaving `cmd/pair-wrap` /
  `cmd/pair-scribe` as thin shims (so the KDL-execed `pair-wrap` and the user's
  `~/.local/bin/pair-scribe` `~/.zshrc` wiring keep working). `pair-scribe` is
  user shell tooling and stays out of the runtime bundle. That completes step-2
  dispatcher routing (#91); the standalone shim names still resolve.
- #94 M1 ported the two nvim-keybind marker-writers `bin/pair-restart.sh` /
  `bin/pair-quit.sh` to in-process `pair restart` / `pair quit` subcommands and
  removed the two `.sh` from the tree + runtime bundle (see the `pair quit` /
  `pair restart` rows). **#94 M2** then retired the last five orchestrator shell
  **shims** — `bin/copy-on-select.sh`, `bin/flash-pane.sh`,
  `bin/clipboard-to-pane.sh`, `bin/pair-title.sh`, `bin/pair-session-watch.sh`.
  Callers were repointed to the Go binaries (same base name, no `.sh`, unchanged):
  `zellij/config.kdl` now sets `copy_command "copy-on-select"`; the Go
  `copy-on-select` execs `$PAIR_HOME/bin/flash-pane` / `bin/clipboard-to-pane`
  directly (`cmd/internal/clipcmd/run.go`); the launcher's `SpawnTitlePoller` /
  `SpawnSessionWatcher` (`cmd/internal/launcher/osruntime.go`) spawn
  `bin/pair-title` / `bin/pair-session-watch` directly. All five were dropped from
  the tree and from `explicitAssetPaths` (`embed_test.go` asserts them excluded;
  the copied-binary smoke `tests/pair-embedded-runtime-test.sh` asserts they're
  absent). Net: all **seven** orchestrator shims are gone (2 in M1, 5 in M2). The
  runtime bundle is now **shell-reduced, not shell-free** — six non-orchestrator
  shell utilities remain: `bin/pair-help`, `bin/pair-notify`,
  `bin/lib/adapt-log.sh`, `bin/lib/dev-rebuild.sh`, `doctor/doctor.sh`,
  `doctor/emitter-health.sh`.
- **#95 (final step)** decided how the native `nvim`/`zellij` startup assets reach
  the external processes: **keep the digest-versioned extraction, reframed as a
  content-addressed runtime *cache*** (deterministic content digest, idempotent
  skip-unchanged writes, upgrade-safe + self-pruning keep-2) — mechanics
  **unchanged**, and confined to the **copied-binary layout** (source
  `PAIR_HOME`/baked `defaultPairHome`, and Homebrew `defaultPairHome=libexec`,
  point the asset root at an adjacent real tree and never extract). Ephemeral-temp
  and API/flag-driven startup were weighed and rejected (both degenerate to
  re-writing the same nvim tree, and nvim has no in-memory-config API). The only
  **new code** is the **restored PATH prepend**: the Go launcher's `RunLaunch` now
  calls the pure `prependBinToPath($PAIR_HOME/bin, PATH)` once at entry
  (`cmd/internal/launcher/pathenv.go` + `createflow.go`), so zellij and its panes
  (`pair-wrap`, `copy_command "copy-on-select"`, `Run "pair-help"`/openers, the
  nvim viewers) resolve the bundled helpers by bare name across copied/source/
  Homebrew. The retired shell `bin/pair` did this prepend; the Go launcher that
  replaced it dropped it in #99 M5c — a real regression a copied/Homebrew install
  would hit — now guarded by a `prependBinToPath` unit test + a copied-binary
  smoke asserting bare-name helper resolution. **#91's endpoint is reached (as
  documented):** one Pair *executable* + a self-provisioned content-addressed
  config cache for the two external tools + system platform tools. True zero-tree
  stays unreachable while `nvim`/`zellij` read config from real filesystem paths
  and `nvim/init.lua` `dofile()`s siblings by absolute path (needs a real directory
  that persists for the session — `embed.FS` is not a path), so that residual gap
  is the accepted, documented limitation (`ARCH-PURPOSE` permits documenting the
  final gap).

## Coverage Ledger

The logical rows above group files where a per-file migration row would add
noise. The following paths were inspected or are covered by an explicit grouping
rule:

- `Makefile`
- `Makefile.local`
- `README.md`
- `cmd/pair-scribe/README.md`
- `doctor/README.md`
- `doctor/SKILL.md`
- `bin/clipboard-to-pane.sh` (removed #94 M2 — `.sh` passthrough retired; `cmd/clipboard-to-pane` / `bin/clipboard-to-pane` is the owner, still bundled)
- `bin/copy-on-select.sh` (removed #94 M2 — `.sh` passthrough retired; `cmd/copy-on-select` / `bin/copy-on-select` is the owner, still bundled)
- `bin/flash-pane.sh` (removed #94 M2 — `.sh` passthrough retired; `cmd/flash-pane` / `bin/flash-pane` is the owner, still bundled)
- `bin/lib/adapt-log.sh`
- `bin/lib/dev-rebuild.sh`
- `bin/pair`
- `bin/pair-changelog`
- `bin/pair-context`
- `bin/pair-continuation`
- `bin/pair-dev`
- `bin/pair-help`
- `bin/pair-notify`
- `bin/pair-quit.sh` (removed #94 M1 — ported to in-process `pair quit`, `cmd/internal/launcher/restart.go`)
- `bin/pair-restart.sh` (removed #94 M1 — ported to in-process `pair restart`, `cmd/internal/launcher/restart.go`)
- `bin/pair-scribe`
- `bin/pair-scrollback-render`
- `bin/pair-session-watch.sh` (removed #94 M2 — `.sh` passthrough retired; `cmd/pair-session-watch` / `bin/pair-session-watch` is the owner, still bundled)
- `bin/pair-slug`
- `bin/pair-title.sh` (removed #94 M2 — `.sh` passthrough retired; `cmd/pair-title` / `bin/pair-title` is the owner, still bundled)
- `bin/pair-wrap`
- `cmd/internal/adapt/adapt.go`
- `cmd/internal/adapt/adapt_test.go`
- `cmd/internal/changelogcmd/changelogcmd.go`
- `cmd/internal/changelogcmd/distill.go`
- `cmd/internal/changelogcmd/distill_test.go`
- `cmd/internal/changelogcmd/prompt.go`
- `cmd/internal/changelogcmd/prompt_test.go`
- `cmd/internal/contextcmd/contextcmd.go`
- `cmd/internal/contextcmd/contextcmd_test.go`
- `cmd/internal/continuationcmd/continuation.go`
- `cmd/internal/continuationcmd/continuation_test.go`
- `cmd/internal/continuationcmd/continuationcmd.go`
- `cmd/internal/continuationcmd/git.go`
- `cmd/internal/ctxmeter/ctxmeter.go`
- `cmd/internal/ctxmeter/ctxmeter_test.go`
- `cmd/internal/dispatcher/dispatcher.go`
- `cmd/internal/dispatcher/dispatcher_test.go`
- `cmd/internal/codexsid/codexsid.go`
- `cmd/internal/codexsid/codexsid_test.go`
- `cmd/internal/clipcmd/clipcmd.go`
- `cmd/internal/clipcmd/clipcmd_test.go`
- `cmd/internal/clipcmd/run.go`
- `cmd/internal/clipcmd/run_test.go`
- `cmd/internal/clipcmd/runcli.go`
- `cmd/internal/clipcmd/runtime.go`
- `cmd/internal/zellijpane/zellijpane.go`
- `cmd/internal/zellijpane/zellijpane_test.go`
- `cmd/internal/launcher/agentargs.go` (#99 M1 — per-agent launch-arg pure logic)
- `cmd/internal/launcher/agentargs_test.go`
- `cmd/internal/launcher/config.go` (#99 M1 — config-path + legacy-codex migration decision)
- `cmd/internal/launcher/config_test.go`
- `cmd/internal/launcher/format.go` (#99 M1 — age/title formatting)
- `cmd/internal/launcher/format_test.go`
- `cmd/internal/model/model.go`
- `cmd/internal/model/model_test.go`
- `cmd/internal/opener/opener.go`
- `cmd/internal/opener/opener_test.go`
- `cmd/internal/opener/run.go`
- `cmd/internal/opener/run_test.go`
- `cmd/internal/opener/runcli.go`
- `cmd/internal/opener/runtime.go`
- `cmd/internal/osfs/osfs.go`
- `cmd/internal/osfs/osfs_test.go`
- `cmd/internal/procutil/procutil.go`
- `cmd/internal/reviewcmd/reviewcmd.go`
- `cmd/internal/reviewcmd/reviewcmd_test.go`
- `cmd/internal/reviewcmd/run.go`
- `cmd/internal/reviewcmd/run_test.go`
- `cmd/internal/reviewcmd/runcli.go`
- `cmd/internal/reviewcmd/runtime.go`
- `cmd/internal/procutil/procutil_test.go`
- `cmd/internal/scribecmd/scribecmd.go`
- `cmd/internal/scribecmd/scribecmd_test.go`
- `cmd/internal/scrollbackcmd/events_test.go`
- `cmd/internal/scrollbackcmd/render_test.go`
- `cmd/internal/scrollbackcmd/scrollbackcmd.go`
- `cmd/internal/scrollbackcmd/scrollbackcmd_test.go`
- `cmd/internal/scrollbackcmd/serialize_row_test.go`
- `cmd/internal/scrollbackcmd/timestamps_test.go`
- `cmd/internal/transcript/transcript.go`
- `cmd/internal/transcript/transcript_test.go`
- `cmd/internal/sessionwatch/run.go`
- `cmd/internal/sessionwatch/run_test.go`
- `cmd/internal/sessionwatch/runcli.go`
- `cmd/internal/sessionwatch/runcli_test.go`
- `cmd/internal/sessionwatch/runtime.go`
- `cmd/internal/sessionwatch/sessionwatch.go`
- `cmd/internal/sessionwatch/sessionwatch_test.go`
- `cmd/internal/slugcmd/slug.go`
- `cmd/internal/slugcmd/slug_test.go`
- `cmd/internal/slugcmd/slugcmd.go`
- `cmd/internal/slugcmd/slugcmd_test.go`
- `cmd/internal/titlepoller/titlepoller.go`
- `cmd/internal/titlepoller/titlepoller_test.go`
- `cmd/internal/titlepoller/run.go`
- `cmd/internal/titlepoller/run_test.go`
- `cmd/internal/titlepoller/runcli.go`
- `cmd/internal/titlepoller/runtime.go`
- `cmd/internal/wrapcmd/adapt_drift_test.go`
- `cmd/internal/wrapcmd/extract_fg_test.go`
- `cmd/internal/wrapcmd/keymap_registry_test.go`
- `cmd/internal/wrapcmd/osc_test.go`
- `cmd/internal/wrapcmd/overlay_test.go`
- `cmd/internal/wrapcmd/picker_overlay_test.go`
- `cmd/internal/wrapcmd/run_test.go`
- `cmd/internal/wrapcmd/slug_spawn_test.go`
- `cmd/internal/wrapcmd/stdout_batch_test.go`
- `cmd/internal/wrapcmd/stdout_filter_test.go`
- `cmd/internal/wrapcmd/time_event_test.go`
- `cmd/internal/wrapcmd/translate_stdin_test.go`
- `cmd/internal/wrapcmd/translate_test.go`
- `cmd/internal/wrapcmd/update_agent_output_test.go`
- `cmd/internal/wrapcmd/wrap.go`
- `cmd/internal/wrapcmd/wrap_events_test.go`
- `cmd/copy-on-select/main.go`
- `cmd/clipboard-to-pane/main.go`
- `cmd/flash-pane/main.go`
- `cmd/pair-changelog/e2e_test.go`
- `cmd/pair-changelog/main.go`
- `cmd/pair-changelog/main_test.go`
- `cmd/pair-context/main.go`
- `cmd/pair-context/main_test.go`
- `cmd/pair-continuation/main.go`
- `cmd/pair-continuation/main_test.go`
- `cmd/pair-go/helper_equivalence_test.go`
- `cmd/pair-go/main.go`
- `cmd/pair-go/pty_proxy_route_test.go`
- `cmd/pair-changelog-open/main.go`
- `cmd/pair-review-open/main.go`
- `cmd/pair-review-readiness/main.go`
- `cmd/pair-review-target/main.go`
- `cmd/pair-scrollback-open/main.go`
- `cmd/pair-session-watch/main.go`
- `cmd/pair-title/main.go`
- `cmd/pair-scribe/main.go` (thin shim over `cmd/internal/scribecmd`)
- `cmd/pair-scrollback-render/main.go`
- `cmd/pair-slug/main.go`
- `cmd/pair-slug/main_test.go`
- `cmd/pair-wrap/main.go` (thin shim over `cmd/internal/wrapcmd`)
- `nvim/adapt.lua`
- `nvim/adapt_test.lua`
- `nvim/annotate.lua`
- `nvim/annotate_test.lua`
- `nvim/changelog.lua`
- `nvim/changelog_test.lua`
- `nvim/doctor.lua`
- `nvim/doctor_test.lua`
- `nvim/init.lua`
- `nvim/marker_codec.lua`
- `nvim/pair_poke.lua`
- `nvim/review.lua`
- `nvim/review/apply.lua`
- `nvim/review/docflow.lua`
- `nvim/review/handoff.lua`
- `nvim/review/init.lua`
- `nvim/review/markers.lua`
- `nvim/review/markers_test.lua`
- `nvim/review/menu.lua`
- `nvim/review/menu_test.lua`
- `nvim/review/mode.lua`
- `nvim/review/mode_test.lua`
- `nvim/review/poke_bodies.lua`
- `nvim/review/poke_bodies_test.lua`
- `nvim/review/projection.lua`
- `nvim/review/readiness.lua`
- `nvim/review/readiness_test.lua`
- `nvim/review/reconstruct.lua`
- `nvim/review/reconstruct_test.lua`
- `nvim/review/record.lua`
- `nvim/review/record_test.lua`
- `nvim/review/resolve.lua`
- `nvim/review/resolve_test.lua`
- `nvim/review/seam.lua`
- `nvim/review/seam_test.lua`
- `nvim/review/spinner.lua`
- `nvim/review/spinner_test.lua`
- `nvim/review/wrap.lua`
- `nvim/review/wrap_test.lua`
- `nvim/scrollback.lua`
- `nvim/scrollback_test.lua`
- `nvim/slug.lua`
- `nvim/slug_test.lua`
- `nvim/zellij_trace.lua`
- `zellij/config.kdl`
- `zellij/layouts/main.kdl`

Test-only grouping rule: `tests/*.sh` and `tests/lib/*` are grouped by the
runtime artifact or behavior they exercise. Process fakes cover zellij, nvim,
docflow, review agents, model CLIs, cmux, and PATH-selected Pair helpers.
Headless drivers cover `nvim/init.lua`, viewer Lua, review Lua, and the
timeout wrapper. They are migration evidence, not installed artifacts.
