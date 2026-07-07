# Architecture

## What pair is

A launcher that starts a zellij session with a fixed two-pane split. The top pane runs a TUI coding agent; the bottom pane runs Neovim on a persistent draft file. Keystrokes — and mouse-up after a selection — drive bidirectional flow between the panes via `zellij action write-chars` and `zellij action focus-pane-id`.

The whole thing is deliberately small — a handful of shell scripts, one nvim init, and two zellij KDL files. Required deps: `zellij`, `nvim`, `fzf`, `jq`, `par`, plus the agent itself.

## Pieces

This section is the narrative map. The exhaustive artifact/caller/runtime
contract for the Go packaging migration lives in
[Go migration inventory](go-migration-inventory.md).

```
bin/pair                     # THE Go binary (cmd/pair-go): launcher + EVERY helper as `pair <sub>` (#104)
bin/pair-slug -> pair        # busybox symlink: the external Claude Stop hook's bare name (#104 M3)
bin/pair-dev                 # dev wrapper: rebuild-on-launch (make build), then exec pair
bin/pair-help                # shell shim: `pair -h` in an ESC-to-quit pager
bin/pair-notify              # hook-driven OSC notifier (e.g. claude Notification)
#   former standalone helpers are now subcommands of the single binary:
#     pair wrap · pair scribe · pair session-watch · pair title · pair context ·
#     pair slug · pair continuation · pair scrollback render|open ·
#     pair changelog render|open · pair review target|open|readiness ·
#     pair clip copy-on-select|clipboard-to-pane|flash-pane
nvim/init.lua                # bundled nvim config (loaded via -u)
nvim/scrollback.lua          # read-only ANSI viewer for the scrollback dump
nvim/changelog.lua           # read-only viewer for the distilled change log (#53)
nvim/annotate.lua            # shared 🤖-marker annotation subsystem (Alt+q) for both viewers (#57)
zellij/config.kdl            # mouse, copy_command, keybinds, pane frames
zellij/layouts/main.kdl      # the split + agent/draft commands + swap layouts
```

## Packaging migration target (#72)

**Complete as of #104.** Pair is a single primary Go `pair` binary: a Go-owned
CLI/dispatcher that owns session lifecycle, data/config path resolution, asset
discovery, restart/quit/continue flows, and subprocess orchestration. Every
former Go command surface is now an internal subcommand (`pair wrap`,
`pair slug`, `pair context`, `pair scrollback render`, `pair changelog render`,
`pair continuation`, `pair scribe`, `pair review …`, `pair clip …`, `pair title`,
`pair session-watch`) reached inside a session because the launcher fronts pair's
own dir on the session PATH. The only other installed artifacts are the shell
shims (`pair-dev`, `pair-help`, `pair-notify`) and the `pair-slug`→`pair` busybox
symlink for the external Claude Stop hook. See the
[Go migration inventory](go-migration-inventory.md) #104 rows for the endpoint.

As of #90, the public `bin/pair` command is a Go-built entrypoint from
`cmd/pair-go` with an embedded Pair-owned runtime bundle. It resolves the Pair
asset root, then drives the native launcher (`cmd/internal/launcher`) in-process.
(The `cmd/pair-go` package still handles an internal `<self> launch …` mode; the
`pair-go` *output* binary was dropped in #104 M3 — there is one binary, `pair`.)
Asset root resolution is ordered: explicit `PAIR_HOME`, executable
sibling root, the build-time `defaultPairHome` injected by Make/Homebrew, then
an extracted embedded runtime under `$PAIR_DATA_DIR/runtime/<digest>/pair-home`
when no adjacent/source asset root exists. Native `nvim/` and `zellij/` assets
remain native files inside whichever asset root was selected.

**#95 — extraction reframed as a content-addressed runtime *cache*; the residual
zero-tree gap.** The `$PAIR_DATA_DIR/runtime/<digest>/pair-home` extraction runs
**only for the copied-binary layout**: source (`PAIR_HOME`=repo, or a baked
`defaultPairHome`) and Homebrew (baked `defaultPairHome`=`libexec`) point the
asset root at an adjacent real tree and never extract. It is best understood as a
content-addressed runtime *cache* — not a Pair-owned install tree: a deterministic
content digest names the directory, writes are idempotent (skip-unchanged), and it
is upgrade-safe + self-pruning (keep-2). #95 changed **none** of that mechanism; it
reframed the extraction and made the conscious decision to keep it. **True
zero-tree is unreachable** while `nvim`/`zellij` stay native — they read config
from real filesystem paths, and `nvim/init.lua` `dofile()`s its siblings by
absolute path, so a real directory must exist and persist for the whole session
(mid-session viewers spawn fresh `nvim -u $PAIR_HOME/nvim/*.lua`); Go's `embed.FS`
is in-memory, not a path. The native-single-binary endpoint (#91) is therefore one
Pair **executable** + a self-provisioned content-addressed config cache for the two
external tools + system platform tools — not literally zero bytes on disk.
`ARCH-PURPOSE` permits documenting this final gap.

The embedded runtime is generated from a deterministic manifest before builds
and tests. Since #104 M3 that manifest carries **config + shell shims only** —
the two shell shims (`bin/pair-help`, `bin/pair-notify`), `bin/lib/`, `nvim/`,
`zellij/`, and doctor assets — and **no helper binaries** (every former helper is
a `pair <sub>`, reached via the single `pair` the launcher fronts on the session
PATH). `pair` itself is never self-embedded. External programs such as `zellij`,
`nvim`, `fzf`, `jq`, clipboard tools, and agent CLIs remain system dependencies.

A developer shell sourced from `../ariadne/construct/dev-aliases.sh` (or
`pair-dev`) rebuilds `cmd/pair-go` automatically before running.

The earlier #75 pure launcher core is now the foundation of the native Go
launcher (`cmd/internal/launcher`): real zellij lifecycle, prompt/fzf UI,
restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename,
config/session migration, and title-poller behavior are all Go-owned as of
#99 M1–M5c. `bin/pair-shell`, the original shell launcher, is retired.

The dispatcher hosts implemented helper routes: `context` and `scrollback-render`
(#76), then `slug`, `changelog`, `continuation`, and `session-watch` (#92 M1).
Each route calls the shared internal Go runner (`cmd/internal/<name>cmd`) that the
legacy `bin/pair-<name>` binary now also calls as a thin shim (`ARCH-DRY`), so a
single implementation backs both entry points.

Two new architectural surfaces landed with #92 M1:

- **Public `pair <sub>` peel-off.** `entrypoint.ClassifyInvocation` takes the
  reserved dispatcher-subcommand set (`dispatcher.DispatchNames()`, derived from
  the single `Families()` table) and returns `ModeDispatch` when the public
  `pair` is invoked with one — so `pair slug` dispatches while `pair claude`,
  `pair resume`, and bare `pair` route to the native launcher. The
  reserved names are passed in, not imported, keeping `entrypoint` free of a
  `dispatcher` dependency.
- **Buffered vs streaming dispatch.** Finite, no-stdin routes (`context`,
  `scrollback-render`, `slug`) use the buffered `Dispatch(args) → Result` path.
  Routes needing real stdio — `changelog` (live per-batch stderr for the Alt+l
  spinner), `continuation` (reads the body from stdin), `session-watch`
  (long-running), and the interactive PTY proxies `wrap` / `scribe` (#96, raw
  terminal for the life of a session) — use `cmd/pair-go`'s
  `runStreamingSubcommand` seam, which hands the runner real
  `os.Stdin/Stdout/Stderr`. `Families().Streaming` marks which. The proxies keep
  standalone shims (`bin/pair-wrap` for the KDL PATH-exec, `bin/pair-scribe` for
  the user's `~/.zshrc`) that call the same `wrapcmd.Run` / `scribecmd.Run`.

Pair-owned call-sites were repointed to `pair <sub>` in #92 M2: the changelog
opener's detached distiller and `nvim/scrollback.lua` (`pair scrollback-render` /
`pair changelog`), and `pair-wrap`'s turn-end spawn (`pair slug`). (Two of those
callers have since moved in-process: the title poller no longer shells `pair
context` — #93 M1 folded the count in via `contextcmd`; and the Alt+/ scrollback
opener no longer shells `pair scrollback-render` — #93 M2 renders via
`scrollbackcmd` in-process. The changelog opener's *detached* distiller still
shells `pair scrollback-render` / `pair changelog`, since the build must survive
the viewer closing.) The launcher's two sidecar spawns
(`SpawnSessionWatcher` / `SpawnTitlePoller` in `cmd/internal/launcher/osruntime.go`)
now invoke the Go binaries `bin/pair-session-watch` / `bin/pair-title` directly —
the `.sh` passthrough shims were retired in #94 M2.
The standalone `bin/pair-<name>` dispatcher binaries remain
as thin shims (still built + bundled); dropping them is later single-binary work.

Native integration layers stay native: `nvim/*.lua` remains the bundled Neovim
surface and `zellij/*.kdl` remains the zellij layout/config surface. Packaging
may embed those assets or install them adjacent to the binary, but the migration
does not force Lua or KDL into Go.

The migration is deliberately staged through issue #73 onward. Each step must be
merge-safe: after any sub-issue lands, the public `pair` command, `pair-dev`,
keybindings, scrollback, changelog, continuation, and review flows still work.
The detailed disposition table is maintained in
[Go migration inventory](go-migration-inventory.md), not duplicated here.

### `bin/pair` / `cmd/internal/launcher` — launcher

`bin/pair` is the Go public entrypoint (built from `cmd/pair-go`). It resolves
the asset root, then drives the native launcher (`cmd/internal/launcher`)
in-process with argv[0] presented as `pair`; the launcher owns the full session
lifecycle described below. `bin/pair-shell`, the original 2287-line shell
launcher, was retired in #99 M5c.

**Native launcher (#99, DONE).** The port of `bin/pair-shell` onto the Go
`cmd/internal/launcher` core is staged M1–M5. The pure decision + per-agent-arg
core (#75, #99 M1) is joined in **#99 M2** by the `launcher.Runtime` effect seam
(composed sub-interfaces — zellij / snapshot / ui / proc / env / id / fs — for fake-test
ISP), the `RunLaunch` **create-flow** orchestrator (a thin driver over the pure
deciders: decision → name prompt → tag-restart config picker → config/id mint →
env + sidecar spawns → the blocking `--new-session-with-layout` handoff), and the
concrete `OSRuntime`. **#99 M3** adds the **`LifecycleOps`** sub-interface and turns
`RunLaunch` into an **in-process restart loop** (replacing the shell's `exec $0`):
each iteration runs `runOnce` (one create OR native **attach** — `AttachSession`,
the blocking twin of `LaunchSession`), then `runCleanup` (the `cleanup_quit_marker`
port: `TakeQuitMarker` → `DeleteSession` + `ReapNvim` + gated park-nudge
[`IsTTY`+non-empty raw+`!RestartMarkerPresent` → `ConfirmParkNudge` → `ParkScrollback`]
+ sidecar removal + resume hint + `KillTitlePoller` + cmux reset), then peeks the
restart marker (`TakeRestartMarker`) and re-decides via the pure `planRestart`
(Alt+n resumes; Shift+Alt+N drops the config; **rename/continue re-entries return
`ErrFallbackToShell`** — M5). `SweepOrphanNvim` runs once up front (startup nvim
hygiene). The in-pane guard + sweep are **first-entry only** (a restart re-launch
is the same outer process). **#99 M4** was the **cutover**: it made the native launcher
the **DEFAULT** — `cmd/pair-go` runs it in-process for create / attach /
Alt+n & Shift+Alt+N restart / quit. At the time this was gated by a **`PAIR_LEGACY_LAUNCH=1`**
kill-switch (forced the shell for the whole launch; a rollout safety hatch, since
removed in M5c) that replaced the M2/M3 opt-in `PAIR_NATIVE_LAUNCH`. The native launch sits
behind the `cmd/pair-go` `legacyRuntime` seam (`LaunchNative` → `handled` bool) so
the default-flip was unit-testable without real zellij. During the M4 window a native **decline**
(`ErrFallbackToShell`) fell through to the real `bin/pair-shell` — **not a shim** (a shim
would loop native → fallback → shim → native); **M5c deleted that fallback arm
entirely** (below), so no shell fallback, `ErrFallbackToShell`, or kill-switch
remains. **#99 M5** ported the last flows native, then retired the shell; it was
split into three boundaries by risk. **M5a** (read-only surfaces) makes the fzf
session **pick** native (`resolvePick` over the pure `buildPickRows` — detached-green
rows, age-graded-grey historical rows with the amber `[⏎ N queued]` badge, the
`+ new` sentinel; `HistoricalTag` now carries `MTime`+`QueueCount` from `Scan`, so
the row build is pure and only the fzf call is an effect, reusing `UIOps.PickFromList`
with `--ansi`) and adds the **`list`/`ls`** subcommand (`ListOps.ListSessions` +
the pure `formatListTable`, printed to stdout). A picked existing tag is
resume-by-name — its agent is **inferred from the tag**, not the bare-`pair` default.
**M5b** (lifecycle write flows) makes the last three native: (1) in-session
**compaction** — the `InZellijPane` guard becomes the native branch (pure
`compactionDecision` honoring the `PAIR_FORCE_IN_SESSION`/`PAIR_FAKE_IN_ZELLIJ`/
`ZELLIJ_SESSION_NAME` seams; park `--copy` + `WriteRestartMarker`/`TouchQuitMarker`
+ terminal `ExecKillSession`), else a native "already inside a zellij session"
reject; (2) the **`continue`** subcommand (`ParseArgs` + `ContinuationOps`: bare
lists docs, `<slug>` resolves the newest doc to seed the draft + pick the agent —
reusing `continuationcmd.ContinuationDir`/`NextActionPreview`); (3) the offline
**`rename`** subcommand (pure `renamePathsFor`/`renamePlan` zip, `FSOps.Rename`,
journal + reverse-rollback) and the **`rename_to`/`continue` restart re-entries**
(`planRestart` drops `ShellFallback`; the loop moves sidecars then relaunches under
the new tag, or re-seeds the draft from the slug) — closing the M4-accepted
degradation. **M5c retires the shell entirely** — `bin/pair-shell` is **removed**;
`--help`/`help` print a native `UsageText`; the `cmd/pair-go` fallback arm
(`PAIR_LEGACY_LAUNCH`, the `Exec` seam, `ErrFallbackToShell`, `shellOnlySeamActive`)
is deleted, so `LaunchNative` always returns a real exit code; the defensive
error paths (Sessions/ScanHistory/DecideLaunch/os.Getwd) print + exit instead of
falling back. The asset-root validity marker moves from `bin/pair-shell` to the
always-present **`zellij/layouts/main.kdl`** (tracked + bundled, unlike the built
`bin/pair-wrap`). The `PAIR_TEST_CALL`/`PAIR_DEBUG_*`-driven shell contract tests
(`pair-continue-test`, `cmux-ownership-test`) retire with the shell — every shell
function they pinned has a tested Go equivalent. So `pair` is now a single Go
launcher end-to-end; #94 (stop extracting a shell tree) unblocks.

**Alt+Shift+C compaction is writer-owned + draft-preserving (#105, unblocked by
#104's single binary).** Previously `COMPACT_PROMPT` asked the agent to write a
continuation *and then* run `pair continue <slug>` — a two-step NL instruction
whose second step the agent could skip, so the "automatic restart" was
non-deterministic (the reported "restart stopped working"). Now the `pair
continuation` writer (`cmd/internal/continuationcmd`) owns the restart: when it
detects it is running inside its own live pane (pure `InCompactionContext` =
`PAIR_TAG` + matching `ZELLIJ_SESSION_NAME`, mirroring `compactionDecision`'s
tag-match), it re-invokes `pair continue <slug>` on `os.Executable()` after a
successful write+commit — reusing the tested `runCompaction` → reincarnation-loop
path, with `PAIR_DEV` riding the env so pair/pair-dev config is preserved. The
prompt is a single step; `--no-restart` opts out (a deliberate manual in-pane
write). The writer also **folds the draft pane's WIP** into the continuation's
`## NEXT ACTION` before writing (pure `StripStickyComments` drops the `=== … ===`
stickies — a Go mirror of nvim's `strip_comments` — then `FoldDraftIntoNextAction`
inserts it), so parked draft text survives the restart that would otherwise
overwrite the draft with a seed line. `nvim/init.lua`'s `PairConfirmCompact` saves
the draft first so the writer reads fresh WIP off disk.

The writer's restart exec (`newContinueRestartCmd`) sets **`PAIR_FAKE_IN_ZELLIJ=1`**:
the child `pair continue` re-derives "in a pane?" via `InZellijPane`'s process-ancestry
walk, which the **agent's command sandbox blocks** (process introspection → EPERM) —
the actual "restart stopped working" root cause found by #105's live smoke, since the
old agent-run `pair continue` hit the same wall. The writer already confirmed the
context via the `ZELLIJ_SESSION_NAME` / `PAIR_SESSION_NAME` exact match (no introspection),
so it fakes *only* that ancestry half; `pair continue`'s own session-name match still
guards the session identity.

**Restart/quit ported (#94 M1).** The two nvim-keybind marker-writers
`bin/pair-restart.sh`/`pair-quit.sh` are now in-process Go subcommands —
`pair restart [--new-session] [--rename-to <tag>]` and `pair quit`
(`runRestart`/`runQuit` in `cmd/internal/launcher/restart.go`, parsed by
`parseRestart` in `args.go`, routed via `LaunchNative`). They reuse the launcher's
existing `WriteRestartMarker`/`TouchQuitMarker`/`ExecKillSession`/`InferAgent`
seam — no new marker logic, and the `{quit,restart}-<session>` marker protocol is
unchanged (only the writer moved from shell into Go). The two `.sh` are removed
from the source tree and the runtime bundle (`explicitAssetPaths`); `nvim/init.lua`
now invokes `{ 'pair', 'quit' }` / `{ 'pair', 'restart', ... }`.

The launcher resolves `$PAIR_HOME` from its own executable path, prepends `$PAIR_HOME/bin` to `$PATH` (the Go launcher's `RunLaunch` calls the pure `prependBinToPath` once at entry, idempotent across re-launches — #95 restored this: the retired shell `bin/pair` did the prepend, but the Go launcher that replaced it dropped it in #99 M5c, a regression a copied/Homebrew install would hit) so the bundled helpers resolve by bare name in zellij configs and keybinds, parses argv — first positional is `$PAIR_AGENT` (default `claude`), everything after `--` is joined into `$PAIR_AGENT_ARGS`, extra positionals before `--` are an error with a usage hint, resolves the git root for repo scope, and exports a repo-scoped `$PAIR_DATA_DIR` under `${XDG_DATA_HOME:-$HOME/.local/share}/pair/repos/<scope-key>`. The user-facing tag remains repo-local (`work`, `bugfix`); zellij gets a globally unique public name assigned through the global `session-names.jsonl` index. See [Session identity and storage](session-identity.md) for the identity and storage contract.

A leading `pair resume <tag>` is recognized as a subcommand verb (alongside `list` / `help`): it skips both the picker and the name prompt, attaches if the tag's current-scope public zellij session still exists in any state, otherwise creates with that repo-local tag. When `resume` is in play, the agent is inferred from the scoped tag ledger, with `agent-<tag>` and config filenames kept as compatibility caches — so a single tag is enough to restart, regardless of which agent was originally paired with it. See "Tag-restart" below.

**Decision tree.** Finds live/detached Pair sessions owned by the current repo scope through `session-names.jsonl`; unindexed `pair-*` rows are legacy candidates, not automatically current-scope sessions. It also surfaces **historical tags from this repo scope** (#000024) by scanning scoped tag sidecars and ledgers touched within the last `$PAIR_HISTORY_DAYS` (default 14) but no longer having a live current-scope session. Then:

- 0 detached + 0 historical → run create flow directly (validate agent, prompt for name, create).
- ≥1 detached or ≥1 historical → fzf picker over current-scope detached sessions, then historical rows annotated `(Nd ago, no live session)`, then a `+ new <agent> session` sentinel. Pick a detached row → attach its public session name. Pick a historical row → create by repo-local tag (same path as `pair resume <tag>`, which re-uses scoped `draft-<tag>.md`, `ledger-<tag>.jsonl`, and saved config cache). Pick the sentinel → fall through to create with `free_slot_tag`. `PAIR_DEBUG_HISTORY=1 pair` exits early printing the scan results. A historical row also gets an amber `[⏎ N queued]` badge when `queue_count_for` finds N `<digits>.md` items under `$PAIR_DATA_DIR/queue-<tag>/`, so a forgotten queue is visible before resume.

The agent argument doesn't filter the picker — reattach is agent-agnostic (the existing session already runs whatever it runs). The agent argument only matters for the create path: it labels the sentinel, drives the auto-suggested default name, and is the binary that gets exec'd in the new session.

There is **no silent auto-attach**. Every reattach goes through the picker so the user explicitly sees what they're connecting to.

Detection of attached-vs-detached uses `zellij --session NAME action list-clients`, which prints a header plus one row per connected client. Zero rows = detached.

**Tag reuse & stale-EXITED residue (#67).** A repo-local Pair tag maps to a public zellij session name assigned by `session-names.jsonl` (`pair-<repo>-<tag>` with suffixes when needed). `Alt+x` can leave a resurrect record like `pair-pair-work (EXITED - attach to resurrect)`; that row still shows in `list-sessions`, so collision checks run against the assigned public name, not a reconstructed `pair-<tag>`. The single helper `session_blocks_reuse <session>` centralizes the decision (`ARCH-DRY`): an `EXITED` row is stale full-quit residue — it deletes the zellij record (`delete-session --force`) and reports the session name reusable; a running/detached row still blocks; an absent session never blocks. `pair rename` keeps its own offline-only resurrectable-session contract and gates by current-scope tag ownership.

**Title poller (`cmd/pair-title` + `cmd/internal/titlepoller`; #93 M1) — two surfaces.** A single always-on per-tag 60s background poller, spawned via `ensure_title_poller` on *every* entry (create, attach, restart) so a poller a host sleep/reboot/SIGKILL killed is reliably revived. Ported from `bin/pair-title.sh` to Go on the #78 sessionwatch template — pure decisions (heat buckets, cwd abbrev, frame title, argv identity guard, unchanged-skip cache) unit-tested directly; zellij/cmux/ps/fs behind the `Runtime` seam; the launcher's `SpawnTitlePoller` spawns the Go `bin/pair-title` directly (the `.sh` re-exec shim was retired in #94 M2, so the running process is `pair-title <tag> <agent>` — the argv shape the guard matches). Single-instance guard is identity-checked (`pollerArgvMatches` `ps`-matches the command line for this tag; pidfile `$PAIR_DATA_DIR/title-pid-<tag>`; not a bare `kill -0`) so a recycled PID can't suppress the respawn. It owns two title surfaces (tested in `cmd/internal/titlepoller`):

- **Per-pane context meter in the zellij FRAME (#71).** Each agent pane's frame title reads `<agent> (<count>) [<cwd>]`, where `<count>` is the agent's current context-window occupancy — an absolute humanized token count (`970k`), so no model→window catalog is needed. Source of truth is the agent's own session transcript: the pure `cmd/internal/ctxmeter` reader (`ContextTokens` sums the last *real* claude `message.usage`, skipping `isSidechain`/`<synthetic>` records; codex `last_token_usage.input_tokens` of the last `token_count` event; agy none) + `Humanize`, over the path from the shared `cmd/internal/transcript` resolver (extracted from `pair-slug`, ARCH-DRY). The one-shot `cmd/pair-context <tag> <agent>` wires it (tolerant: any failure prints nothing). Each pane records `{pane_id, cwd, cwd_display}` to a single-writer `pane-<tag>-<agent>.json` at startup (`main.kdl`, beside the startup rename — dodges the 3-writer race on `config-*`); the poller resolves the count **in-process** via `contextcmd` (the same resolver `pair context` uses — no subprocess, #93 M1 ARCH-DRY), and renames the pane through the actual public zellij session name passed from the launcher, gated on recent activity with a per-pane unchanged-skip cache. The glob `pane-<tag>-*.json` can also match a **stale twin** left by a prior session that paired the tag with a different agent (same recycled `pane_id`); the poller renders only the pane whose `Agent == opts.Agent` — the active agent, resolved fresh from `agent-<tag>` on each respawn, so the two-pane invariant guarantees exactly one match (#97, ignoring the twin rather than alphabetical last-wins). The twin is also cleaned at its source: `runCleanup` removes `pane-<tag>-<agent>.json` on Alt+x quit alongside the other per-(tag,agent) sidecars. Always-on (the frame exists with or without cmux). Carried through `pair rename` like `config-*`.

- **cmux workspace-title activity heat-ramp & ownership (#69, cmux-only).** Inside cmux (block-local gate), the workspace title mirrors the public zellij session name with an activity-heat prefix (🔴 <1d / 🟠 <3d / 🟡 <10d / 🔵 <21d / none). Ownership of a shared workspace is recorded in `$PAIR_DATA_DIR/cmux-owner-<CMUX_WORKSPACE_ID>` as `tag<TAB>public-session`; older one-field `tag` files are read as legacy and probed as `pair-<tag>`. A poller defers to a foreign owner while that stored public session is still alive, reclaims stale owners, and writes its own repo-local tag plus public session name when it claims the workspace.

**Saved-config resolution & legacy Codex migration (#67).** `resolve_config_file <tag> <agent>` resolves the canonical `config-<tag>-<agent>.json`. Older Codex sessions on disk use a doubled shape `config-<tag>-codex-codex.json`; when the canonical file is absent and the agent is `codex`, the helper migrates the legacy file to the canonical name *iff* its JSON declares `"agent":"codex"` — a narrow, agent-checked compatibility path, **not** a glob resolver, so unrelated stale files can't silently win (`ARCH-DRY`, `ARCH-PURE`). It is used only where both tag and agent are known (restart-marker read, cleanup resume hint, the tag-restart picker that surfaces native Codex resume, and the two config writes); the agent-inference glob loop is deliberately left alone, since it is *discovering* the agent and already sees the legacy filename.

**Naming prompt.** When the create flow runs, the launcher prompts the user with the auto-suggested tag as the default — the cwd basename, sanitized (so `~/workspace/pair` → `Session name: pair`). The prompt is editable inline (delegated to zsh's `vared` since bash 3.2 has no `read -i`). The `pair-` prefix is implicit — the prompt shows just the tag, since `pair-` is always prepended. Pressing Enter accepts; typing a custom name (`bugfix`, or `pair-bugfix` — leading `pair-` is stripped) overrides it. `pair resume <tag>` skips this prompt entirely.

**Agent validation deferred.** `command -v "$AGENT"` runs only inside the create branch, not at startup, so attaching to a custom-named session whose tag isn't a real binary still works.

**Title.** The launcher emits an OSC 0 escape sequence right before invoking zellij, so the terminal title shows the session name on both create and attach paths (zellij itself only sets it on create).

**Cleanup on quit.** zellij is run as a child (not `exec`) so the launcher resumes when zellij exits. On resume it checks for `~/.cache/pair/quit-<session>` (the marker that `pair quit` writes when Alt+x fires) and, if present, runs `zellij delete-session --force <session>` to clear the resurrect entry. It then SIGKILLs any leftover children that didn't follow the session down: a lingering `zellij --server` (rare but seen), and `nvim --embed` orphans (every `nvim FILE` is internally TUI parent + embed child; the embed sometimes survives RPC-pipe EOF and gets reparented to launchd). The embed reap is two-layered — primary path reads `nvim-pid-<tag>-{draft,scrollback}` files written by VimEnter autocmds inside `nvim/init.lua` and `nvim/scrollback.lua` (so the embed pid is known deterministically); fallback is a tag-scoped `pkill -f`. If a `config-<tag>-<agent>.json` was captured during the session, it also prints a one-liner naming the resume command (`pair resume <session>`) so the user can pick the work back up later. No marker → leave the session as zellij left it (running if Alt+d detached).

**Startup orphan sweep.** The Alt+x reaper only runs when the user quit through pair. External terminations (`zellij kill-session`, host reboot during a session, pair upgrade mid-session) leave the embed orphaned with no marker. `SweepOrphanNvim` runs once per `pair` invocation, just after the live session list is computed: it resolves live public session names through `session-names.jsonl` for the current scope, keeps legacy unindexed `pair-<tag>` rows as legacy live tags, collects candidate tags from both pidfiles and the argv of every running `nvim --embed` referencing `$PAIR_DATA_DIR/`, then calls `ReapNvim` on any tag with no live current-scope session. The argv walk is what catches embeds with no pidfile (autocmd errored before VimEnter, or panes that predate the autocmd). The same reaper is shared with `runCleanup`, so there's exactly one reaper definition; adding a new nvim surface in pair means routing it through `$PAIR_NVIM_PID_FILE` and naming it under `$PAIR_DATA_DIR/{draft,scrollback}-<tag>...`, not extending the reaper.

**Reload / restart in place (Alt+n, Shift+Alt+N).** A second marker, `~/.cache/pair/restart-<session>`, is written alongside `quit-` by `pair restart`, carrying the agent name + a `new_session` flag. After `runCleanup` tears the session down, the launcher's restart loop reads the marker (`TakeRestartMarker`/`planRestart`) and re-enters **in-process** (no `exec $0`), pinned to the killed session's tag from the marker (skipping both the picker and the name prompt). The flag controls what happens to the saved config:

- `new_session=0` (Alt+n) — keep `config-<tag>-<agent>.json`. Append the agent-appropriate resume token to the re-exec'd argv: `--resume <id>` for claude, `resume <id>` for codex, `--conversation <id>` for agy. Result: pure pair reload — same tag, same draft, same agent conversation. Useful after a binary or config rebuild.
- `new_session=1` (Shift+Alt+N) — drop `config-<tag>-<agent>.json` so the next launch's claude `--session-id` injection (or the codex/agy watcher) writes a brand-new entry. Result: fresh agent conversation, same tag and draft.

The picker is bypassed in either flavor — Alt+n's argv carries an explicit resume token, and Shift+Alt+N has no saved config to pick against. A third marker field, `continue=<slug>` (#55, written by the launcher's in-session compaction branch (`compaction.go`), not `pair restart`), rides the `new_session=1` path but re-execs `pair continue <slug> <agent> -- <args>` instead of a plain restart — see "In-session compaction" below.

### `zellij/layouts/main.kdl` — pane split + swap-layout ladder

Horizontal split. Top pane runs `$PAIR_AGENT $PAIR_AGENT_ARGS` (auto-fills remaining height). Bottom pane is `size=12` (fixed 12 rows) running `nvim -u $PAIR_HOME/nvim/init.lua` on the per-tag draft file. Integer sizes are FIXED in zellij (refusing the `resize` action), but pair drives all rung changes through swap layouts, not resize, so FIXED is harmless.

Both panes wrap their command in `sh -c "..."` so the shell expands `$PAIR_AGENT`, `$PAIR_AGENT_ARGS`, `$PAIR_TAG`, and `$PAIR_HOME` at exec time — zellij itself does not interpolate env vars in `command`/`args` fields.

`$PAIR_AGENT_ARGS` is appended on the agent pane command line as a single space-separated string; the shell word-splits it. Args containing spaces are *not* preserved (rare for CLI flags; documented in README).

The bottom pane has `focus=true` (drafting pane gets focus on launch), `borderless=true` (so the `minimized` rung can collapse to 1 row — see "pane frame asymmetry" below), and `name="draft"` — used by zellij in the OSC 0 terminal title (`pair-<tag>: draft`) which propagates to the user's terminal/multiplexer tab title. The draft is borderless so it has no frame title slot; the keybind cheatsheet that used to live in the frame title lives in nvim's statusline (right-aligned, see `nvim/init.lua`).

**Pane frame asymmetry.** `pane_frames true` is set globally in `zellij/config.kdl` so the **agent pane** renders a frame — the value is the scroll-position indicator zellij draws in the top-right of a framed pane (e.g. `500/540`), which is the only way to see scrollback position (zellij doesn't expose scroll offset to plugins or the CLI). The **draft pane** opts out via `borderless=true` in every layout (default + both swap layouts), because a framed pane has a ~3-row minimum and the `minimized` rung needs `size=1`. Cost: the agent pane loses 2 rows + 2 cols to the frame chrome.

**Swap layouts.** Two `swap_tiled_layout` entries — `minimized` (draft `size=1`) and `third` (draft `size="33%"`) — sit alongside the default layout above. Each is gated by `exact_panes=2` so it only applies when the current pane structure matches what pair builds. `nvim/init.lua` drives them via `zellij action next-swap-layout` / `previous-swap-layout`, which re-tile the existing agent + nvim panes onto the target layout positionally — running pane processes (`pair-wrap`, `nvim`) survive each swap. Cycle from default(small) is `[minimized, third]`: `next-swap-layout` from small → minimized, from minimized → third, from third → wraps to small. The lua side maps Alt+Down to next-swap (smaller rung) and Alt+Up to prev-swap (bigger rung), with a state-machine clamp at the rung extremes.

### `zellij/config.kdl` — mouse, copy, keybinds

Top-level config:

- `mouse_click_through true` — first click on an unfocused pane goes through to the pane (so click-and-drag selects in one motion) instead of being consumed by zellij just to change focus.
- `copy_command "copy-on-select"` — on every selection finalize (mouse-up after drag), zellij pipes the selected text to this binary. `copy_command` replaces zellij's default OS-clipboard write, so the binary does that part too. Resolved by PATH (which the launcher populated).
- `pane_frames true` — frames are enabled globally so the agent pane shows zellij's scroll-position indicator (top-right of the frame) when scrolled. The draft pane opts out via `borderless=true` in `zellij/layouts/main.kdl` so the `minimized` rung can still collapse to 1 row (a framed pane's minimum is ~3 rows). The cheatsheet still renders in nvim's statusline rather than a frame title — the draft has no frame to hold one.

Keybinds added on top of zellij defaults (`clear-defaults=false`):

- `unbind "Alt i"` — release Alt+i (zellij's default binds it to MoveTab; we want nvim to see it for image attach).
- `unbind "Alt n"` — release Alt+n (zellij's default `NewPane` would break pair's two-pane invariant; we rebind it below for restart).
- Mode-locking — every default chord that would switch zellij modes (`Ctrl+g/p/t/n/h/s/o/b`) is unbound, and `Ctrl+q` (zellij's resurrect-leaving Quit) is unbound too — Alt+x is the only quit path.
- `Alt+d` — routed through nvim to `:lua PairConfirmDetach()` — Y/N modal then detach.
- `Alt+x` — routed through nvim to `:lua PairConfirmQuit()` — Y/N modal then `pair quit` (full quit).
- `Alt+n` — routed through nvim to `:lua PairConfirmRestart()` — Y/N modal then `pair restart` (reload pair, keep agent session).
- `Shift+Alt+N` — routed through nvim to `:lua PairConfirmRestartNewSession()` — Y/N modal then `pair restart --new-session` (restart with a fresh agent conversation). See "Reload / restart in place" under the launcher section.
- `Alt+Shift+C` (`Alt C` / `Ctrl Alt c`) — routed through nvim to `:lua PairConfirmCompact()` — Y/N modal then `send_to_agent(<compaction prompt>)` (#55). Unlike the restart binds it does NOT shell out directly: distilling a continuation needs the agent's judgment, so it asks the agent to write a continuation via `pair continuation`; the **writer then triggers the restart itself** (#105) — the agent no longer runs `pair continue` (removing that skippable step was the fix). See "In-session compaction" below.
- `Alt+h` — `Run "pair-help" { floating true; close_on_exit true; ... }` — pops a floating pane running `pair -h | less`.
- `Alt+↑` / `Alt+↓` — route to nvim's `PairLayoutBigger` / `PairLayoutSmaller` — step the nvim pane along the swap-layout ladder (`minimized ↔ small (12 rows) ↔ third`).
- `Alt+j` — `FocusNextPane` — toggle focus between the agent and draft panes. Works from either pane because it's a global zellij bind, intercepted before the focused pane sees the key. Overrides zellij's default Alt+j (`MoveFocus "Down"`), which only reached the draft and was a dead key once you were already there; the two-pane invariant makes `FocusNextPane` a clean toggle with no direction to track.

The Alt+x/d/n confirms route through nvim rather than running directly so a single fat-finger doesn't tear the session down (Alt+x in particular is unrecoverable). The lua side also auto-grows out of `minimized` before showing the modal, since otherwise the prompt would land on a 1-row pane where nothing is visible.

### `clipboard-to-pane` — clipboard read + hand off to nvim

Go binary `cmd/clipboard-to-pane` (logic in `cmd/internal/clipcmd`, #93 M4); `copy-on-select` execs `$PAIR_HOME/bin/clipboard-to-pane` directly (the `.sh` re-exec shim was retired in #94 M2).

Read OS clipboard (`pbpaste` / `wl-paste` / `xclip`). Stage the raw body to `$PAIR_DATA_DIR/quote-<tag>`. All formatting decisions (par reflow, `> ` prefix) live in nvim now, conditional on cursor position — this is just a transport.

Find the nvim pane via `zellij action list-panes --json` (parsed by `cmd/internal/zellijpane`), looking for the pane whose `terminal_command` contains `nvim`. Focus it via `zellij action focus-pane-id <id>` (bare then `terminal_<id>` form) — critical because it runs as a child of the zellij server (invoked via `copy_command`), so we cannot rely on positional `move-focus` to land on nvim.

Once nvim is focused, send a single `Ctrl-_` (ASCII 31, `zellij action write 31`). On the nvim side `<C-_>` is mapped to `PairPasteQuote` **only in insert mode** — that mapping IS the gate: in normal mode (e.g. browsing prompt history with Alt+←/→) Ctrl-_ hits nvim's near-no-op default and the buffer isn't touched, so we deliberately do NOT force-normal-mode here (doing so would destroy the very mode signal that drives the gate). `PairPasteQuote` reads the staged body and dispatches on cursor column — see the `nvim/init.lua` section below.

Diagnostic log lives at `${XDG_CACHE_HOME:-~/.cache}/pair/clipboard-debug.log` (appended, best-effort).

### `copy-on-select` — zellij copy_command wrapper

Go binary `cmd/copy-on-select` (logic in `cmd/internal/clipcmd`, #93 M4); zellij's `copy_command "copy-on-select"` invokes `bin/copy-on-select` directly by name (the `.sh` shim was retired in #94 M2).

**Split into a fast hook + a detached orchestrator (#100).** The paste chain makes ~5 sequential `zellij action` client round-trips (~400ms each cold) plus binary cold-starts — ~1.5–2s total — but zellij SIGKILL-**reaps** its `copy_command` child after ~1s. Running the whole chain inside the hook meant the first (cold) copy after a restart was killed mid-paste and dropped. So the two halves now run in separate processes:

- **Hook** (what zellij's `copy_command` runs): receives the selection on stdin, mirrors it to the OS clipboard (zellij's default write is bypassed when `copy_command` is set, so this is mandatory), then `SpawnDetached`'s `copy-on-select --orchestrate` and **returns immediately**. Nothing slow runs here, so the reap can't truncate the paste — on any machine, cold or warm.
- **Orchestrator** (`copy-on-select --orchestrate`, `setsid`-detached so it outlives the hook and the reap): checks whether the focused pane is the nvim draft; if so exits without acting (selecting in nvim shouldn't loop back). Otherwise flashes the source pane via `flash-pane <id>` (its own setsid `--reset` survives), then execs `clipboard-to-pane` to hand off to nvim. It reads the selection back off the OS clipboard (which the hook already populated), so no data crosses the process boundary. Flash defaults: `#50fa7b` dracula green, 100ms, override via `$PAIR_FLASH_BG` / `$PAIR_FLASH_MS`.

The detach idiom (`setsid` + `/dev/null` stdio + `Start`, no wait) is the shared `startDetached` helper in `clipcmd/runtime.go`, also used by the flash's `ResetPaneColorAfter`.

Pane detection: parse `list-panes --json --command` (via `cmd/internal/zellijpane`), find the focused non-plugin/non-floating pane, and match its `terminal_command` — **not** its `title` — against `nvim|draft`. Keying on `terminal_command` is the #copy-on-select-test fix: the agent overwrites its pane title with `claude [<cwd>]`, so a repo path containing `nvim` (e.g. `parley.nvim`) would misclassify the agent pane as the draft and skip the paste; `terminal_command` never embeds the cwd.

### `pair quit` — Alt+x handler

In-process Go subcommand (`runQuit` in `cmd/internal/launcher/restart.go`, ported from `bin/pair-quit.sh` in #94 M1). Touches the marker file `~/.cache/pair/quit-$ZELLIJ_SESSION_NAME` (via `TouchQuitMarker`), then execs `zellij kill-session $ZELLIJ_SESSION_NAME` (`ExecKillSession`). The kill terminates the session including the subcommand itself; on the launcher side, the launcher's restart loop resumes when the child zellij exits, and `runCleanup` sees the marker and runs `delete-session --force` to clean up the resurrect entry.

Alt+x leaves the draft, queue, and history intact — the next session resumes them. Use Shift+Alt+Backspace (`forget_all`) for the destructive "start anew" path.

### Outer-TTY capture and notification routing — `bin/pair-wrap`, `bin/pair-notify`

**Why.** Zellij parses every escape on the way out for its virtual-screen reconstruction and drops sequences it doesn't recognize. OSC 9 and OSC 777 (the notification escapes outer wrappers like cmux watch for) fall in that bucket and never reach the host terminal. BEL is forwarded since zellij 0.44, but cmux specifically watches OSC, not BEL — so BEL forwarding doesn't help that integration. Filed as #000011.

**Mechanism, in two layers:**

1. **Outer-TTY capture (in the launcher — `RecordOuterTTY`).** Before invoking zellij, on every attach (both create and reattach branches), the launcher resolves the path of its controlling TTY — which is precisely the outer PTY (the one allocated by whatever wraps pair: cmux, a terminal emulator, etc.). That path gets written to `$PAIR_DATA_DIR/outer-tty-<tag>`. Refreshed on every attach because the outer PTY changes across detach/reattach, while pane-shell env stays frozen at zellij session-creation time (env-var approaches would go stale).

2. **Two consumers** of the captured path:

   - **`bin/pair-wrap`** (Go, `cmd/pair-wrap`). Transparent PTY proxy. The zellij agent pane runs `pair-wrap $PAIR_AGENT $PAIR_AGENT_ARGS` instead of the agent directly (wired in `zellij/layouts/main.kdl`). The wrapper allocates a fresh PTY for the agent, forwards stdin/stdout transparently with SIGWINCH propagation, and watches the agent's output stream for OSC notifications. On detection it writes OSC 9 directly to the recorded outer-TTY path — bypassing zellij.

     **Stdin raw mode.** The wrapper switches its stdin (zellij's pane PTY) into termios raw mode for the duration. Without this the kernel's line discipline does local echo + canonical buffering on the bytes flowing toward the wrapped TUI, which double-echoes keystrokes and corrupts terminal-response sequences. Saved/restored in a `finally` block.

     **Stdin Enter remap (per-agent).** `sendKeymapByAgent` (`cmd/internal/wrapcmd/wrap.go`) translates the user's Enter / Alt+Enter to per-agent send/newline bytes so the convention matches pair's nvim draft pane (Enter = newline, Alt+Enter = send). For `claude` the user's plain Enter becomes `\<CR>` (claude's portable "insert newline" sequence); Alt+Enter becomes a bare `\r` (send). For Codex / agy, pair sends LF for plain Enter (their newline gesture) and CR for Alt+Enter (send) — Codex reads a lone `\r` as the Enter key (submit / picker confirm), so the submit chord must collapse to `\r`, not a modified `ESC CR`. The same keymap carries `altBS`: Alt+Backspace (legacy `\x1b\x7f` or KKP `\x1b[127;3u`, the same two-protocol shape as Alt+Enter) rewrites to **Ctrl+U** (`0x15`, kill-to-line-start) for every agent — so Alt+Delete in the agent pane matches the agent's Cmd+Delete and the draft pane's Alt+Delete. A lone `0x7f` (plain Backspace) isn't ESC-prefixed, so it passes through untouched. Opt out of the whole remap with `PAIR_WRAP_REMAP_RETURN=0`.

     **Stdout filtering and batching (Codex).** Codex inline mode emits DEC synchronized-output markers (`ESC[?2026h` / `ESC[?2026l`) around frequent redraw batches. It can also enable terminal focus-event mode (`ESC[?1004h`) even though pair/zellij do not use focus events for the agent pane. `pair-wrap` strips those markers from the stdout stream sent to zellij, because zellij scrollback/mouse scrolling can behave poorly while a pane is in synchronized-output or extra terminal-event modes during generation. The filtered, user-visible stdout stream is then queued and flushed to zellij on a 100ms cadence (plus EOF) to lower redraw pressure from dense Codex repaint bursts (#85). The raw scrollback log remains immediate and unfiltered so forensic replay still captures the agent's original PTY stream and offset-keyed resize/time events stay aligned.

     **Overlay-aware suspension (per-agent).** Textarea Enter remaps are wrong while a blocking overlay / picker has focus: the overlay needs a bare `\r` to confirm the highlighted option. pair-wrap registers per-agent overlay detectors in `overlayDetectorByAgent`, sets `pickerActive` when one fires, and emits a bare `\r` for the next plain Enter only. The flag clears after that one Enter, so normal textarea remapping resumes for the following keystroke. Claude uses the stable `OSC 777;notify;Claude Code;Claude needs your permission` body. Codex question prompts use `OSC 9;Plan mode prompt:...`; other Codex pickers fall back to stripped visible output plus a short text carryover watching for labels such as `Use session directory (` / `Use current directory (`, `Press enter to continue`, and `Press enter to confirm or esc to go back`. Codex image attach uses a stronger local signal: Alt+i arms pair-wrap capture immediately before Ctrl+V, and that capture arm also enables the next-Enter overlay bypass. The carryover is cleared when the confirming Enter is consumed so stale picker text cannot re-arm the flag. Known edge inherited from the one-shot design: dismissing an overlay without Enter leaves the flag set until the next plain Enter.

     **OSC filter (`is_actionable_osc`).** Parsing every OSC `<Ps>;<body>` and discriminating is essential — naive "any BEL → emit" over-fires constantly because claude (and similar agents) update OSC 0 (window title) every second with a spinner, and every title set's BEL terminator looks like a "lone bell." The filter:
     - **Skip** OSC 0/1/2 (title sets), OSC 9;4;... (iTerm progress codes — fire on every tool-call cycle).
     - **Forward** OSC 777;... (urxvt-style `Notify`) and OSC 9;`<text>` (iTerm-style notification with content).
     - Bare BEL (no OSC framing in the rolling buffer) → **logged but not forwarded by default**; set `PAIR_WRAP_BELL_FALLBACK=1` to re-enable forwarding (issue #000014).

     Rate-limited to one emit per 0.5s. Empirically: claude emits `OSC 777;notify;Claude Code;Claude is waiting for your input` after ~60s of idle waiting — that's the actionable signal that gets through.

     **Why bare BEL is opt-in.** When an OSC sequence's terminating `\x07` arrives in a read whose preceding bytes (the `\x1b]<ps>;` opener) were already consumed by a prior match, `OSC_RE` can't reconstruct the boundary, and the trailing `\x07` looks like a standalone BEL. Live data from a single 2hr Claude Code session showed 76 emits, only 8 legitimate (all OSC 777); the other 68 were BEL fallback firing on tails of OSC 8 hyperlinks (claude renders file references as clickable links) and OSC 0 spinner title sets. Modern TUI agents signal attention via OSC 9/777 explicitly — the BEL fallback's defensive value never materialized. The detection branch still runs (so `PAIR_WRAP_LOG` shows `BEL-skip` lines), it just doesn't write to the outer TTY unless the env flag is set.

     **Debug log.** `PAIR_WRAP_LOG=<path>` enables a per-detection forensic trail (timestamp, OSC/BEL match, emit/skip outcome). Off by default. Used to discover an unfamiliar agent's notification protocol the first time, then update `is_actionable_osc()` if the agent uses a family the current filter doesn't recognize.

     ```sh
     PAIR_WRAP_LOG=~/pair-wrap.log pair codex
     # use the agent normally; let it idle, finish tasks, etc.
     # detach with Alt+d when done
     cat ~/pair-wrap.log
     ```

     Log lines:

     | Line | Meaning |
     |---|---|
     | `OSC<N>: b'<body>'` | OSC `<N>` recognized as actionable; emit fired |
     | `OSC<N>-skip: b'<body>'` | OSC `<N>` recognized but filtered (title set, progress, etc.) |
     | `BEL: b'<context>'` | bare BEL fallback fired (only with `PAIR_WRAP_BELL_FALLBACK=1`) |
     | `BEL-skip: b'<context>'` | bare BEL detected but not forwarded (default) |
     | `EMIT: 'wrote OSC 9 to <path>'` | successful write to outer TTY (cmux should have badged) |
     | `EMIT-skip: 'rate-limited (...)'` | within 0.5s of last emit; collapsed |
     | `EMIT-skip: 'no outer-tty file...'` | not running under pair, or `record_outer_tty` failed |
     | `EMIT-fail: '<path>: ...'` | tried to write but the recorded path is gone or unwritable |

     Reading strategy: look for `OSC` or `BEL` lines that fired around moments where the agent was waiting — that's the actionable signal. If only `-skip` lines appear, either (a) the agent has no attention notification protocol and you'll need a hook-based path (`pair-notify`), or (b) the agent uses an OSC family `is_actionable_osc()` doesn't yet recognize — extend the filter.

   - **`bin/pair-notify`** (bash). Hook-driven helper for richer signals. `pair-notify [--osc 9|777] "msg"` reads the same outer-TTY file and writes the OSC. Intended for Claude Code `Notification`/`Stop` hooks where you want semantic events with custom message text rather than relying on the agent's native OSC stream.

**Failure mode.** Both are designed to never block the agent. `pair-wrap` swallows exceptions in the detection/emission path and keeps proxying. `pair-notify` exits 0 with a stderr warning when `PAIR_TAG` is unset, the file is missing, or the recorded path isn't writable.

### Colored scrollback dump — `pair-wrap`, `pair-scrollback-render`, `pair-scrollback-open`, `nvim/scrollback.lua`

**Why.** zellij now renders a frame on the agent pane, which surfaces a scroll-position indicator (e.g. `500/540`) in the top-right. Knowing the position is half the value — the other half is being able to *jump back* to a remembered line. zellij's built-in `EditScrollback` strips ANSI styles when dumping (its scrollback is a styled cell grid internally, but the dump is plain text) and opens in a new tiled pane that breaks pair's two-pane invariant. Filed as #000017.

**Capture (in `pair-wrap`).** When invoked with `--scrollback-log <path>`, pair-wrap opens `<path>` (truncated) and tees every chunk read from the agent's master PTY into it. Alongside it, `<path-without-.raw>.events.jsonl` collects one offset-keyed JSON line per out-of-band event — `resize` boundaries and (#59) minute-debounced `time` stamps:

```
{"type":"resize","offset":<bytes>,"cols":N,"rows":N}
{"type":"time","offset":<bytes>,"ts":"<RFC3339>"}
```

The `time` events (one generic `logScrollbackEvent` writer, ARCH-DRY; pure `dueForTimeEvent` debounce + a `p.now` clock seam, ARCH-PURE) let the change-log render date entries by real change-time — the raw byte stream stays byte-faithful (the scrollback render replays it), since the timestamp lives in the sidecar, not injected into the TTY (#59).

The existing `set_winsize()` is the single entry point for both the initial PTY size (called once after `pty.fork`) and every SIGWINCH (the registered handler). Threading `log_scrollback_event()` through it covers both. `SCROLLBACK_BYTES` is bumped after each successful write to the raw fd, so the offset on each resize event demarcates "from this byte onward, apply these new (cols, rows)" — which is what the renderer needs to replay each segment at its correct width. Failure mode is unchanged: any tee or sidecar write error is `debug()`-logged and swallowed; the proxy never blocks the agent on a logging hiccup. `zellij/layouts/main.kdl` passes the flag by default, so capture runs automatically for every pair session.

**Replay (`bin/pair-scrollback-render`, Go).** Reads `<raw>` and `<events.jsonl>`, feeds the bytes to a `charmbracelet/x/vt` emulator in a single offset-ordered walk over all events (`feedSegments`): write up to each offset, then `Resize` on a resize event or snapshot `Scrollback().Len()` on a `time` event (#59). The emulator runs the same VT100 interpretation zellij does live (width-based wrap, alternate-screen flips, scroll regions), so its row count matches what the user saw in zellij's indicator. After feeding, the renderer walks the scrolled-out history followed by the visible buffer, and emits one ANSI-decorated line per row to `<out.ansi>`: full-reset SGR + per-row attrs + the row's characters + `\x1b[0m`. With `--with-timestamps` (the change-log path only — never the Alt+/ viewer) the pure `interleaveDateMarkers` then inserts `⟦pair:ts DATE⟧` lines at each day boundary from the time snapshots (#59). Built into `bin/pair-scrollback-render` via `make pair-scrollback-render`; single static binary, no runtime dep. Its raw inputs live in `$PAIR_DATA_DIR` as `scrollback-<tag>-<agent>.{raw,events.jsonl}` (RAW VT bytes, NOT in the repo); `:PairTTYRawPath` / `_G.PairTTYRawPath()` (nvim, #56) prints the current session's live `.raw` path on demand and copies it to the `+` register — useful for grabbing the byte stream mid-session, since an Alt+x quit deletes it unless preserved.

**Plain projection (`--plain`, `--max-lines`).** The same emulator state can be emitted *without* SGR: `serializeRow` in plain mode drops the per-row attrs and the trailing `\x1b[0m`, and trims trailing blanks by visible *content* — a bg-only "visible" cell (inverse-video / box fill) is kept in colored mode but dropped in plain (else a bordered region becomes space-padding toward terminal width). This is the **sessionView** abstraction's second decoration: one pipeline, colored for the Alt+/ viewer, plain for distillation — the substrate a `continuation` is built from (see `construct/datatype/continuation.md` and `cmd/pair-continuation`). `--max-lines N` overrides the 2000-row viewer cap (`<=0` = uncapped) so a continuation distills the whole session, not just the viewer window.

**Continuation writer (`cmd/pair-continuation`, `continuation` datatype).** A *continuation* is the human-understanding cousin of a native `pair resume`: `resume` restores machine state (the agent's own session id, byte-faithful); a continuation distills the *rendered* session (the plain projection above) into a durable, portable markdown doc — `workshop/continuation/<YYYYMMDDTHHMMSS>-slug.md`, the `continuation` datatype defined in `construct/datatype/continuation.md` (ariadne#91) — so work resumes across time / machines / people / agent stacks. The `xx-datatype` dispatcher does the distillation (judgment); `cmd/pair-continuation` does the *mechanics* deterministically: render conformant frontmatter, allocate a collision-safe timestamped name, write, then `git add` + a **path-scoped** `commit -- <file>` (so a dirty index isn't swept in) + `push origin HEAD` — on the current branch, which lands on main when that branch merges (disaster-recovery — an unpushed recovery doc can't save state; a push failure is non-fatal so a detached park still keeps the local commit). Pure core (frontmatter / name allocation / assemble / validate) is IO-free; a thin clock/fs/git seam (the `git -C` shell-out pattern from `cmd/pair-slug`, no git library) is exercised against a real temp repo with a bare origin. `pair continue [slug] [agent]` resumes *from* a continuation: bare lists them; `<slug>` seeds a fresh session (via `draft-<tag>.md`, create-path only) to read the doc and do its NEXT ACTION; `[agent]` ports to a different stack. Unlike `resume` it does **not** force the tag — it flows through the normal name prompt so the operator picks it (a long slug must never become an over-long zellij `--session` name: zellij caps it at the sun_path socket-budget and rejects overflow with a cryptic "must be less than 0 characters" clap error, so a launch-time guard probes zellij's own validator via a no-op `action list-clients` and fails with a clear message instead — #54) — and forwards `-- <args>` to the agent like a plain `pair <agent> -- <args>`. It never reads `session_id` — that's `resume`'s job. On Alt+x, `cleanup_quit_marker` offers to **park** the session: preserve its scrollback (`.raw` + `.events.jsonl`) under a non-recyclable `parked-scrollback-<tag>-<ts>.*` so a live session can distill it later (no live agent exists at quit, so the nudge only *preserves*). The prompt is timeout-bounded — it auto-defaults to **N** (preserve nothing) after `PAIR_PARK_PROMPT_TIMEOUT` seconds (default 5, integer seam) so an unattended quit never blocks on it (#64). The park mechanics live in a shared `park_scrollback <tag> <agent> [--copy]` helper (#55, extracted from `cleanup_quit_marker` — ARCH-DRY): the quit path *moves* (session dying), the compaction path *copies* (the live `pair-wrap --scrollback-log` is still appending to `.raw`).

**In-session compaction (#55, `Alt+Shift+C`).** `pair continue <slug>` is *context-aware*: run from a normal shell it fresh-starts (above); run from INSIDE its own live pane it **compacts** — copy-parks the scrollback as a recovery net, writes a restart marker carrying a new `continue=<slug>` field (same tag, `new_session=1`), and kills the session. The launcher's in-process restart loop (`RunLaunch` reads the marker via `TakeRestartMarker`/`planRestart`) then re-enters with `pair continue <slug> <agent> -- <args>` (now outside zellij → the fresh-start branch), so the session reincarnates under the same tag with a clean conversation seeded from the continuation. Detection is **ancestry-based** (`InZellijPane`) plus exact public-session identity: the launcher exports `PAIR_SESSION_NAME`, and compaction requires `ZELLIJ_SESSION_NAME == PAIR_SESSION_NAME` for scoped sessions, with exact `pair-<tag>` kept only as legacy fallback when the exported session is absent. It never trusts `$ZELLIJ*` env alone, since cmux propagates those to sibling non-pair panes (a false positive would park+kill the wrong session). The pure `compactionDecision` runs *before* the `InZellijPane` guard (which otherwise rejects any in-pane `pair`), so an in-pane `pair continue` compacts instead of being rejected. Seams: `PAIR_FORCE_IN_SESSION`, `PAIR_KILL_CMD` (test-only), and `PAIR_FAKE_IN_ZELLIJ` — the last is **also used in production** (#105): the `pair continuation` writer sets it on the `pair continue` it execs, because a **sandboxed agent shell blocks the `InZellijPane` proc-ancestry walk** (process introspection → EPERM), so the child would otherwise fail to detect the pane and misfire; the writer already confirmed the context via the exact session-name match, so faking only the ancestry half is safe (the child's own session-name match still guards). The trigger is the `Alt+Shift+C` keybind (`Alt C` / `Ctrl Alt c` → `PairConfirmCompact` → an agent-agnostic prompt that **defers to the `continuation` datatype procedure** — flush-first, then write the continuation via the `pair continuation` writer, which **triggers the restart itself** (#105; the agent does not run `pair continue`) — rather than enumerating a section skeleton inline, so the prompt can't drift out of sync with the datatype; that drift was the bug pair#61 fixed); the outer process suppresses the Alt+x park nudge whenever a restart marker is pending (a restart isn't a quit).

### Shared 🤖-marker annotation — `nvim/annotate.lua`

**Why.** Both read-only viewers — the scrollback viewer (`Alt+/`, `nvim/scrollback.lua`) and the change-log viewer (`Alt+l`, `nvim/changelog.lua`) — want the same `Alt+q` affordance: drop a 🤖 comment/question on a line (`🤖[Y]`) or a selection (`🤖<X>[Y]`), and on quit ship the user-added markers to the draft pane (→ the agent) via the `scrollback-pending-<tag>.md` sidecar the draft picks up on `FocusGained`. Rather than duplicate ~400 lines, the marker subsystem lives once in `nvim/annotate.lua`, `dofile`'d by both viewers (same dir-relative load as `adapt.lua`, since each viewer launches with `nvim -u <viewer>.lua`). Filed #57 (split from #53).

**Shape (ARCH-PURE).** A **pure core** — `find_markers_in_line` (byte-walk parser), the `>`/`]` escape-unescape (`esc_x`/`esc_y`/`unescape`, backslash-parity walk so a selection containing the delimiters survives), `strip_markers`, `marker_key`, `collect_markers_by_line` (the load-time baseline so only *newly-added* markers extract), `format_extraction`, `new_marker_count`, width helpers — is exposed on the module table and unit-tested directly in `nvim/annotate_test.lua` (no buffer, no IO, no mocks). A **thin IO/UI seam** — `M.attach{bufnr, pending_path, footer, source_label, quit_noun}` — wires the `Alt+q` keymaps, the floating prompt (`open_marker_prompt`), the read-only unlock→insert→relock rewrite, the `VimLeavePre`→sidecar `M.emit`, and the `M.confirm_quit` gate (confirms only when there are user-added markers / a footer comment to ship).

**Per-viewer parameters.** `footer=true` adds the scrollback-only overall-comment affordance line (gated entirely by `footer_row` being nil otherwise); `source_label` tags each emitted quote `> [<label>] <quote>` so the agent can tell a change-log question from a raw-scrollback one — a *per-quote* prefix (not a header line) so the draft pickup's `\n> ` marker count (`init.lua` `pair_pickup_scrollback_pending`) stays faithful. `M.has_new_markers`/`M.on_reloaded` let a viewer with an async refresh (changelog) skip a destructive reload when the user has annotated since open. Both viewers consume annotate: `nvim/scrollback.lua` with `footer=true` + no `source_label` (byte-identical pre-#57 UX), and `nvim/changelog.lua` with `footer=false` + `source_label='change log'` + the `start_refresh` reload guard (`safe_reload` skips the distiller's line-replace when `has_new_markers`, so a marker added during the spinner survives — annotations win; the fresh log is on disk for the next `Alt+l`).

**Viewer (`nvim/scrollback.lua`).** Plugin-free init loaded via `nvim -u`. On `BufReadPost`, an SGR state machine walks each line: peels every `\x1b[...m` escape, mutates a running state (fg/bg/bold/italic/underline/reverse/strike/blink), and emits an extmark span for each contiguous run of visible bytes under a single state. Color resolution: 30-37/90-97 fg + 40-47/100-107 bg map through an xterm-default palette; `38;5;n` indexed maps via the standard 256-color formula (16 anchored to the same palette, 16-231 = 6×6×6 cube, 232-255 = greyscale ramp); `38;2;r;g;b` uses RGB directly. State→hl-group cache is keyed by stringified attrs and uses an explicit counter (not `#hl_cache` — that's 0 on string-keyed tables, a bug caught during the test pass). Buffer is locked read-only (`modifiable = false`, `buftype = nofile`, no swapfile); only `<Esc>` quits via `<cmd>qa<CR>` — `q`, `ZZ`, `ZQ` are deliberately shadowed so a fat-fingered `q` (instead of `Alt+q` for the marker comment) can't slam the viewer shut and drop pending markers.

`G` is a semi-live refresh affordance (#84): before jumping to EOF, the viewer derives sibling `.raw` / `.events.jsonl` paths from the current `.ansi`, reruns `pair-scrollback-render`, reloads the same buffer in place, reapplies ANSI extmarks, relocks the read-only options, and then lands at the refreshed bottom. If the user has pending `Alt+q` annotations or an overall footer comment, the render still updates the backing `.ansi` but the visible buffer is not destructively replaced; the next clean refresh or reopen will show the new snapshot after the comment is shipped. Render/read failures warn and keep the existing snapshot visible, so refresh never replaces usable scrollback with a broken buffer. This deliberately reuses the existing floating viewer instead of stacking another `pair-scrollback-open` pane.

**Open (`cmd/pair-scrollback-open` / `cmd/internal/opener`, Go — #93 M2).** Validates `PAIR_DATA_DIR` / `PAIR_TAG` / `PAIR_AGENT`, renders **in-process** via `scrollbackcmd` (no `pair scrollback-render` subprocess), then *launches* `nvim -u $PAIR_HOME/nvim/scrollback.lua $ANSI` as a held child (`RunViewer`) — deliberately **not** an exec-replace, so the launcher stays alive as nvim's parent and a `defer` clears the re-entrancy lock on quit (the Go analog of the old shell `EXIT`/`INT`/`TERM` trap). Errors print and `Sleep` briefly so the message is readable before the floating pane self-closes. The Go binary **replaced** the old POSIX-sh script at the same PATH name (zellij invokes it by name, so no shim is kept); the viewport scorer + session keying are pure and unit-tested in `opener`, with zellij/nvim/fs behind its `Runtime` seam. Bound in `zellij/config.kdl` to `Alt+/` as a 100% × 100% floating pane with `close_on_exit=true` — the user's `:q` in the viewer dismisses the pane and returns to pair's two-pane layout untouched. **Re-entrancy guard:** `Alt+/` is a global zellij bind, so pressing it again while the viewer is already focused fires another `Run` and would stack a second nvim (one `:q`/Esc per layer to unwind). zellij can't conditionally skip a `Run`, so the launcher self-guards: before launching nvim it writes its own PID to `$PAIR_DATA_DIR/scrollback-<tag>-<agent>.openlock`, and on entry it returns immediately if that lock already holds a *live* PID — the redundant floating pane then self-dismisses via `close_on_exit` and focus falls back to the open viewer. A stale lock (hard kill) carries a dead PID and is reclaimed by the next open's liveness check (`procutil.Alive`, i.e. `kill -0`). The draft pane's `Alt+b` (`--jump prev`) runs the same launcher, so it's covered too.

**Jump-on-open shortcut — draft `Alt+b` = "Alt+/ then Alt+b".** `pair-scrollback-open` takes an optional `--jump prev|next`; it exports `PAIR_SCROLLBACK_JUMP` before launching nvim, and `scrollback.lua` calls `jump_to_prompt()` right after its normal viewport positioning — so the viewer opens already sitting on the previous (or next) user prompt, behaviourally identical to opening with Alt+/ and then pressing Alt+b. The draft pane's `Alt+b` (`nvim/init.lua`, `pair_scrollback_prev_prompt`) is the one-key trigger: it opens the same floating pane via `zellij run --floating … -- pair-scrollback-open --jump prev` (geometry mirrored from the `Alt+/` bind). Env-scoped rather than a sentinel file, so there's no staleness across plain `Alt+/` opens.

**Comment markers — `Alt+q` in viewer → draft pickup (#000018).** While reading scrollback, `Alt+q` drops a parley-style `🤖[]` marker at the cursor (or `🤖<selection>[…]` in visual mode). The buffer is read-only, so the keymap lifts `modifiable`/`readonly` for the insert and re-locks immediately. (#57: this whole marker subsystem was extracted to the shared `nvim/annotate.lua` — the change-log viewer uses the identical flow; see "Shared 🤖-marker annotation" above.) On viewer exit (`VimLeavePre`), `nvim/annotate.lua` (`M.emit`) walks every line, parses each `🤖<X>?[Y]` marker by literal-byte scan (Lua patterns aren't UTF-8 aware), and writes a formatted block to `$PAIR_DATA_DIR/scrollback-pending-<tag>.md`:

```
> <X | line stripped of all markers>
<Y>
```

The draft pane's `nvim/init.lua` registers a `FocusGained` autocmd that picks up the sidecar: on the `*` slot, it appends the block directly into the buffer and `:write`s (going through nvim_buf_set_lines, not an autoread + checktime dance, sidesteps the sub-second mtime resolution issue). Off-slot (`-N` / `+N`), it appends to `draft-<tag>.md` so the next nav-to-`*` reads it from disk. Sidecar is removed in both cases, and a `vim.notify` flashes "🤖 picked up N scrollback comment(s)". Round-trip: read scrollback → `Alt+q` to mark → `:q` → focus the draft → see the formatted block ready to send via `Alt+Return`.

**Overall comment affordance (#000021).** After inline annotations, users often want a standalone summary not tied to any line. `BufReadPost` appends one trailing row — `For overall comment, Alt+q on this line.` — rendered in default Normal color (not dimmed; the affordance is positional, not visual). `nvim`'s `virt_lines` aren't cursor-navigable, so this is a real line. `Alt+q` on that row routes to `add_footer_comment` (no inline-quote context) and stores the text in `state[bufnr].footer_text`; the visible row becomes `Overall comment: <text>` and edits via the same chord. Empty submit clears, restoring the hint. `M.emit` strips the affordance row from the marker scan and appends the stored text as a trailing standalone block (no `> quote` prefix) in the sidecar. The Esc exit-confirm folds it into the prompt ("3 🤖[] markers + overall comment will be sent"). (#57: this footer flow now lives in `nvim/annotate.lua`, gated behind `attach{footer=true}` — so only the scrollback viewer shows the affordance; the change-log viewer attaches with `footer=false`.)

### `nvim/init.lua` — drafting buffer config

Loaded via `nvim -u`, fully isolated from the user's main nvim config. Provides:

- Drafting-friendly defaults: no line numbers, wrap, linebreak, breakindent, spell, persistent undo under `~/.local/share/pair/undo/`, `cmdheight=0` to keep the cmdline out of the way, custom statusline (see "prompt history & queue" below).
- `<M-CR>` (Alt+Return, normal+insert) — `send_and_clear`: append buffer to log, send to agent pane via `zellij action move-focus up` + `write-chars` + `send-keys "Alt Enter"`, clear `*` (when source was `*`, or when a send from `+N` parked a non-empty draft into the queue — see "Prompt history & queue"), save, drop into insert mode.
- `<S-M-CR>` (Alt+Shift+Return, normal+insert) — `send_and_clear(no_submit=true)`: identical flow (strip, log, queue handling, clear, reset) but writes a bare CR (`write 13`) instead of the semantic Alt+Enter submit event. pair-wrap rewrites a bare CR into the agent's insert-newline sequence rather than its submit byte, so the draft lands in the agent's composer on a fresh line, **unsubmitted** — append-without-send.
- `<M-Left>` / `<M-Right>` — navigate the prompt-history / queue position one slot at a time (see below).
- `<S-M-Left>` / `<S-M-Right>` — jump to the next region boundary (oldest history, newest history, `*`, front-of-queue, back-of-queue). Lets the user skip over long histories or queues without N taps.
- `<M-b>` — `pair_scrollback_prev_prompt`: open the scrollback viewer already positioned on the previous agent-conversation prompt — a one-key shortcut for `Alt+/` then `Alt+b`. Shells out `zellij run --floating … -- pair-scrollback-open --jump prev`. See the scrollback section's "Jump-on-open shortcut".
- `<M-q>` — push the current buffer to the front of the queue. From `*` also clears `*`; from `+N` it's move-to-front (removes the source queue file).
- `<M-BS>` — delete the current `+N` queue item without sending, in **both normal and insert mode** (#62 — the gesture doesn't change meaning mid-edit); "stay-near" behavior (items behind shift down, position label keeps its number, so the next item is now under the cursor for repeat-delete). Off the queue (`*`/`-N`) it's a no-op in normal mode and a kill-to-line-start (`<C-U>`) in insert mode, so the line-kill editing convenience stays on the draft.
- `<M-i>` (Alt+i, normal+insert) — `attach_image`: capture-driven image attach. 1) Verify the OS clipboard holds image data (macOS: AppleScript `clipboard info` enumerates `PNGf`/`TIFF`/etc.; Linux: `wl-paste --list-types` or `xclip -t TARGETS`) — if not, flash `[no image in clipboard]` as inline virt_text for 1s and bail. 2) Read pair-wrap's pid from `$PAIR_DATA_DIR/pair-wrap-pid-<tag>` (notify+abort if missing/dead, since pair-wrap is the whole agent I/O path). 3) `kill -USR1 <pid>` to arm a ~200ms capture window in pair-wrap, then `zellij action write 22` to send Ctrl+V to the agent pane. 4) Poll `image-capture-<tag>.done` (20ms cadence, 600ms cap); on hit, read `image-capture-<tag>`, strip ANSI, regex `%[Image[ #][^%]]+%]` (matches both claude's `[Image #N]` and agy's `[Image N-M]`) and insert the captured marker verbatim at cursor. The agent is the source of truth for the marker text — no local counter, no per-agent format hardcoded.
- `PairPasteQuote()` (mapped to `<C-_>` in insert mode; triggered by `clipboard-to-pane` sending Ctrl-_ / ASCII 31): reads the raw selection from `$PAIR_DATA_DIR/quote-<tag>` and dispatches on cursor column.
  - **col == 0 (`paste_as_quote`)**: par-reflow with width 1000, prefix every line with `> `; if the cursor's line is empty, replace it, else insert above (existing line slides down); scroll first inserted line to top via `zt`; cursor on a single empty line directly below the block in insert mode; flash the quoted lines with `IncSearch` (full-line, per-line `nvim_buf_add_highlight`).
  - **col > 0 (`paste_inline`)**: par-reflow (so hard-wrapped sources collapse to one continuous run, paragraph breaks preserved), insert at the cursor via `nvim_buf_set_text` (handles multi-line splits); cursor at the end of the inserted span in insert mode; no scroll; flash the inserted span with a single multi-line extmark.
  - In both modes the highlight is cleared 500ms later via `vim.defer_fn`. Selection-finalize visual cue (issue #12).
- Autosave on `BufLeave`, `FocusLost`, `InsertLeave` so disk and buffer agree.
- As-you-type fuzzy path completion (issue #13). `TextChangedI`/`TextChangedP` autocmd splits the trailing path token on the last `/` into `<dir>` + `<filter>`, lists `<dir>` via `getcompletion`, fuzzy-filters with built-in `matchfuzzy`, hands the result to `vim.fn.complete()`. Triggers only when the token contains `/` or starts with `~` (plain words stay quiet). `<Tab>`/`<S-Tab>` cycle; `<CR>` routes through the pure `cr_keys(visible, has_selection, momentary)` decision (#65): accepts the highlighted item (`<C-y>`) when one is selected, else — the common case under `completeopt=noselect` — dismisses the popup AND inserts a newline (`<C-e><CR>`; a bare `<CR>` while the menu is up is swallowed without a newline, so the explicit `<C-e>` cancel is required). The shared `<CR>` map passes `momentary=spell_popup_active`, so the transient normal-mode `z=` spell popup keeps its clean-dismiss contract (bare `<CR>`, no spurious newline) while as-you-type completions get the newline. Mouse clicks on menu items are intercepted via insert-mode `<LeftMouse>` mapping which calculates popup bounds (`pum_getpos()`) and mouse coordinates (`getmousepos()`) to select and confirm the clicked item instantly. Plugin-free.
- **Completer chain (`run_completers`).** The `TextChangedI`/`TextChangedP` autocmd runs three completers in priority order, short-circuiting on the first that calls `complete()` (each returns `true` when it does): `path_complete` (slash/tilde tokens) → `word_complete` (draft-buffer words + agent-output spans) → `spell_complete` (spelling fixes). `spell_complete` is the as-you-type counterpart to the `z=` popup: at end-of-word, if the alphabetic word being typed (≥ `SPELL_TRIGGER_MIN`=4 chars) is flagged by `spellbadword`, it offers `spellsuggest` results as a plain (unlabelled) `complete()` menu — picked via CR/Tab/arrows. It deliberately skips the `indexed_items` ⌥N labels: the quick-pick keys don't function in this mid-type fallback (Alt+N is bound for path/word menus, bare digits stay literal), so advertising them would mislead. Being last in the chain, it fires only when path/word completion found nothing, so real completions are never crowded out. Unlike `spell_suggest_popup` it does **not** set `spell_popup_active` (we're mid-type — bare digits stay literal, and `CompleteDone` must not bounce to normal mode); it also bails when the cursor sits inside a word so `complete()`'s replace span can't strand a tail.
- **Completion quick-pick + `z=` spell popup.** The first 9 completion items are abbr-tagged with a pick key (`indexed_items`, optional `label_prefix`). Insert-mode path/word completions tag `⌥1`…`⌥9` and pick via `<M-i>` (`pair_pick_completion` feeds `<C-n>`/`<C-p>` to land on item N, then `<C-y>`) — bare digits there stay literal text. The normal-mode `z=` spell popup (`spell_suggest_popup`) reuses the same menu but is a momentary "pick a fix" gesture, not a typing session: it tags plain `1`…`9` and lets **bare digits** pick (`spell_pick_digit`, gated on a `spell_popup_active` flag so the behavior never leaks into ordinary insert-mode typing). `z=` enters insert mode only to host the popup; `CompleteDone` `stopinsert`s back to normal mode on accept *or* dismiss, and `InsertLeave` clears the flag as a safety net.
- All autocmds live in the `pair` augroup (`clear=true`), so iterating via `:luafile $PAIR_HOME/nvim/init.lua` reloads cleanly without duplicating handlers.
- **Layout ladder** — `PairLayoutBigger` / `PairLayoutSmaller` derive the current rung from `vim.o.lines` (the kdl pins each rung to an exact size — 1 / 12 / 33% — so nvim's pane height is ground truth) and call `zellij action next-swap-layout` / `previous-swap-layout` accordingly. Reading actual height makes drift self-correcting: a silently-rejected swap can't desync state, since the next press recomputes from reality rather than a counter that was incremented optimistically. `pair_layout_state` mirrors the rung in-memory for callers like `pair_spinner_start` and `pair_ensure_visible_then` to check without re-reading; an on-disk copy at `${XDG_DATA_HOME:-~/.local/share}/pair/layout-mode-<tag>` is purely diagnostic. Landing in `minimized` also `MoveFocus`es up to the agent pane (the draft is unusable at 1 row) and the focus-grab spinner suppresses itself when `pair_layout_state == 'minimized'`.
- **Statusline cheatsheet (right-aligned, progressive disclosure).** `PAIR_CHEATS` lists `Alt+h help`, `Alt+⏎ send`, `Alt+q queue`, `Alt+x quit`, `Alt+d detach` in priority order. `pair_compose_statusline` measures the variable left segment (history/queue/position cluster), reserves a 6-cell minimum gap, and accumulates as many cheat entries as fit in the remaining columns — Alt+h is always the last entry to drop. Spinner takes the right slot when active (vim only honors a single `%=` per statusline). The minimized rung shows a standalone "Alt+↑ for pair input box" hint instead, with 4 leading spaces so the terminal cursor (which lands on the statusline row when the buffer has zero visible lines) sits on whitespace rather than the hint text.
- **Alt+x / Alt+d / Alt+n / Shift+Alt+N confirm modals.** `PairConfirmQuit` / `PairConfirmDetach` / `PairConfirmRestart` / `PairConfirmRestartNewSession` shell out to `pair quit` / `zellij action detach` / `pair restart` / `pair restart --new-session` after a Y/N modal that defaults to No. All four are wrapped in `pair_ensure_visible_then`, which auto-grows out of `minimized` (calls `PairLayoutBigger` and defers the modal 100ms) so the prompt renders on visible rows. The two restart modals share a single `pair_confirm_restart_impl(new_session)` helper.

### Prompt history & queue (issue #000015)

The nvim buffer is a virtual cursor over a sequence of slots:

```
[ -N ... -2  -1 ]   *   [ +1  +2 ... +M ]
   history (log)    draft     queue (future)
```

The status line shows position state:

```
 Alt: <- history H < pos[*][ (⌫=del)] > Q queued -> 
```

- `H` / `Q` = total counts of history / queue entries.
- `pos` = `*` | `-N` | `+N`.
- Trailing `*` on `-N` means the buffer differs from the loaded baseline (a pending fork awaiting `Send` / `Queue` / `Discard`).
- A contextual `[key=action]` hint appears inside the brackets — `[q=queue]` on `*`/`-N` (Alt+q parks/forks to queue front), `[⌫=del]` on `+N` (Alt+BS deletes the item). Bracket convention: TUI status-bar "key badge" idiom (`[Esc] cancel` etc.). Distinct from the prompt convention `( ) [ ]` for access-key-vs-default, which only applies to interactive dialogs.
- The flanking `<-` / `->` text and the `Alt:` prefix make the navigation gesture self-documenting (Alt+← / Alt+→).
- Highlight is linked to `Comment` rather than the default inverted `StatusLine` so the bar reads as muted secondary info; reapplied on `ColorScheme`.

**Slot mutability is the central distinction:**

| Slot | Storage | Mutable? | Edit autosave? |
|---|---|---|---|
| `*` | `draft-<tag>.md` | yes | yes (existing autocmd) |
| `+N` | `queue-<tag>/NNNNNN.md` | yes | yes (same autocmd) |
| `-N` | parsed from `log-<tag>.md` | **no — immutable** | no; edit becomes a pending fork |

**Navigation (Alt+←/→):** on navigate-away from a mutable slot, the buffer is autosaved to its underlying file. On navigate-away from a dirty `-N`, a single-line prompt fires:

```
(S)end, (Q)ueue, (D)iscard, [S]tay:
```

- **s/S** — Send the fork (append to log), return to `*`.
- **q/Q** — push to queue front (`+1`), return to `*`.
- **d/D** — drop the edit, proceed with the navigation.
- **anything else (Enter, ESC, ...)** — Stay; cancel the navigation.

`*` is preserved across navigation: when leaving `*`, its content is autosaved, so navigating into history/queue and back never destroys the draft. Sending from `-N` preserves `*` (the "clear the draft" semantic of `Alt+Return` only fires when the source slot was `*`). **Sending from `+N` while `*` holds an in-progress draft parks that draft as a queue item (`push_front`) before shipping the selected item** — so `*` ends up empty (sent item's stickies + a fresh line) and the WIP survives as the new `+1`, rather than dangling at `*`. The selected item is resolved by its filename **key captured before** the enqueue, never by the display index: the `push_front` shifts every index by one, and removing by a stale index is what previously left the sent item in *both* `+N` and `-1` (duplication). Regression-guarded by `tests/queue-send-test.sh` (`make test-queue`). Empty / comment-only drafts have nothing to park, so that case is unchanged.

**Queue store:** `queue-<tag>/` directory of one file per queued prompt. Filenames are 6-digit zero-padded sortable keys; sort order = display order (`+1` is the lowest key). New keys at `push_front` decrement the current min; `push_back` increments the current max. Initial midpoint at `500000` to leave room either way.

**Forget-all (Shift+Alt+BS):** wipes `log-<tag>.md`, `draft-<tag>.md`, and every file in `queue-<tag>/` after a confirmation prompt that defaults to No. Hard delete, not an archive — symmetric with the per-item `Alt+BS` queue delete. The confirm-default-No is the safety: a stray Shift+Alt+BS doesn't nuke the session.

**Comments (`=== ...`):** whole lines matching `^%s*===` are stripped from the body at send time only. Draft, queue, and log files store the unstripped text so annotations survive history navigation. A comment-only prompt is a silent no-op (no queue consumption, no log append). Stripping is line-based and not fence-aware. Implementation: `strip_comments` in `nvim/init.lua`, called from `send_and_clear` and `ship_buffer_and_reset`.

**Comment-only edits to history are persisted.** `is_dirty_history_slot` compares `strip_comments(buffer)` against `strip_comments(baseline)`, so adding/removing comments on a `-N` slot doesn't read as a fork. `autosave_current_slot` then rewrites the corresponding log entry's body in place (preserving its timestamp header) via `write_history_entry`. Real forks — anything that changes the stripped body — are still left unsaved so the next `go_to` raises the Send/Queue/Discard/Stay prompt.

Implementation in `nvim/init.lua`: see helpers grouped under `is_dirty_history_slot`, `autosave_current_slot`, `leave_dirty_history`, `go_to`, `nav_left`/`nav_right`, `nav_boundary` (Shift+Alt jumps), `queue_current`, `delete_current_queue_item`, `forget_all`, plus the `queue_*` file ops. State lives in module-local `nav = { pos, baseline }` — `pos` is `'*'` or `{ kind='history'|'queue', n=N }`.

**Insert-mode-only auto-insert from mouse selection.** `bin/copy-on-select` mirrors any selection to the OS clipboard; for selections outside the nvim pane it then triggers `PairPasteQuote` by sending Ctrl-_ (ASCII 31) to the focused nvim pane. The `<C-_>` keymap is bound **only in insert mode**, which is structurally the gate: when the user is in normal mode (e.g. browsing prompt history with Alt+←/→), Ctrl-_ hits nvim's near-no-op default and the buffer isn't mutated. The selection is still on the OS clipboard for manual paste. No mode-probing files or shell-side state needed.

### Auto-orientation slug — `cmd/pair-slug`, pair-wrap trigger, nvim winbar (issue #000027)

When juggling several pair tabs, the `=== comment ===` on draft line 1 (pinned
to the winbar by `pair_pin_header`) tells you what a tab is about — but only if
you remember to type it. This feature auto-maintains it. A **propose / dispose**
split keeps the model out of the live buffer:

- **Trigger** — `pair-wrap`'s turn-end detection (`emitOuter`, the agent-agnostic
  notify sink: marker-regex for claude, idle/native OSC for codex/agy). On
  turn-end it spawns `pair-slug` in the background (debounced `slugDebounceS`,
  `PAIR_AGENT` set, repo cwd inherited). This is agent-agnostic by design — *not*
  a claude `Stop` hook — so the slug works for every agent and needs no
  `~/.claude` config (pair-wrap wraps every session).
- **Propose** — `cmd/pair-slug` (thin shim over `cmd/internal/slugcmd`; also `pair slug`, #92). Resolves its own transcript from
  `config-<tag>-<agent>.json` (session_id) + the per-agent path, and parses each
  **native format** into `{role,text}` turns: claude jsonl, codex rollout
  (`response_item`/`payload.message`), agy jsonl (USER_INPUT transcript). Derives the
  left from the git branch (`git -C <cwd>`); asks a small model (`$PAIR_SLUG_MODEL`,
  default `claude-haiku-4-5` via `claude -p`, or `gpt-5.4-mini` when
  `PAIR_AGENT=codex`) for the `<focus>` right over a **user-biased**
  window (`selectWindow` extends back past tool-only turns to include real user
  prompts). Codex uses the direct OpenAI Responses API when `OPENAI_API_KEY` is
  exported; otherwise it shells through `codex exec` so subscription-authenticated
  Codex CLI sessions still work. The per-agent model dispatch
  (claude/codex/agy/OpenAI-Responses) lives in the shared **`cmd/internal/model`**
  package (`model.Run`), extracted from pair-slug in #53 so `cmd/pair-changelog`
  (the Alt+l change-log distiller) shares one dispatch; the OpenAI output-token
  cap is a per-call parameter (pair-slug passes 64, the change-log a larger
  budget). It writes a validated `=== <branch> | <focus> ===`
  to `slug-proposed-<tag>`. Gates: KEEP keeps the focus but refreshes the left,
  validate-or-keep-last, left always stomped with the authoritative branch.
  `PAIR_SLUG_NESTED` breaks any recursion. Failures are non-fatal.
- **Dispose** — nvim (`nvim/slug.lua`) watches `slug-proposed-<tag>` and applies
  it to draft line 1 only when safe (never touches the prompt below, not
  mid-compose, freeform no-pipe stays manual). An empty draft is an initialization
  case: nvim inserts the slug on line 1, adds a blank line 2, and moves the
  cursor there so composition continues below the header. nvim mirrors the
  effective line 1 back into `slug-<tag>` — the `prev` the proposer reads next
  turn (so a user edit reaches the model, soft policy). Single writer per file
  (proposer→`slug-proposed`, nvim→`slug-<tag>`) makes the channel race-free.

Pure cores are tested: `cmd/internal/slugcmd/slug.go` (normalize/parse/decide) via
`go test`, the nvim decision via `nvim -l` (`make test-lua`). Per-agent parsers
validated against real codex/agy transcripts. Tests that drive `nvim --headless`
through real keymap callbacks (e.g. `queue-send-test.sh` exercising `<M-CR>` →
`send_to_agent`) often run *inside* a live zellij session, so every `zellij
action` shell-out in `nvim/init.lua` is guarded by `has_ui()`
(`#vim.api.nvim_list_uis() > 0`) — headless nvim has no UI attached, so the
guard turns those shell-outs into no-ops and test inputs never leak into the
active agent pane (#000042). Every headless-nvim driver routes its boot through a
shared timeout watchdog (`tests/lib/run-headless.sh`, `run_headless`) that bounds
the boot, kills + returns `124` loud on overrun, and surfaces nvim output on
failure — because a driver that dirties its buffer then runs bare `vim.cmd('qall')`
hits `E37: No write since last change`, refuses to quit, and hangs `nvim --headless`
forever (#000060). **Buffer-mutating headless drivers must `qall!`**; the watchdog's
124-on-timeout contract is pinned by `tests/run-headless-test.sh` (the path a green
suite can no longer exercise once `qall!` lands).

### Change log — `Alt+l`, `cmd/pair-changelog`, `nvim/changelog.lua` (issue #53)

The distilled counterpart to `Alt+/`: `Alt+/` shows the *raw* scroll, `Alt+l`
the *distilled* one — an append-mostly list of session milestones/decisions an
operator can glance at. On-demand only (no per-turn cost); a *furtherance of the
slug* (LLM distillation of what the operator saw), but accumulating instead of a
one-liner.

- **Orchestrate** — `Alt l` (`zellij/config.kdl`, next to `Alt /`) runs
  `pair-changelog-open` (`cmd/pair-changelog-open` / `cmd/internal/opener`, Go
  since #93 M2 — the shared `opener` package with the scrollback launcher) in a
  floating pane. It opens `nvim -u nvim/changelog.lua` on the existing log
  **immediately**, and launches the render+distill in its **own session** (Go
  `SysProcAttr.Setsid`, which replaced the shell's `setsid` / `perl POSIX::setsid`
  shim) — NOT a child of nvim, and outside the pane's process group — so a long
  batched build keeps running even if the operator closes the viewer mid-build
  (#58). (`nohup` alone wasn't enough: closing the zellij floating pane
  tears down the pane's process group, killing a plain background child; a new
  session escapes it.) Two locks: `openlock` (viewer singleton — a second `Alt l`
  while a viewer is up refocuses) and `distill.lock` holding the distiller PID
  (distiller singleton — a press won't start a second distiller while one runs; it
  re-attaches a viewer to the in-progress build). The distiller's stderr →
  `.status` (batch progress for the spinner). The viewer is a pure **watcher**
  (`PAIR_CHANGELOG_LOG`/`DLOCK`/`STATUS`).
- **Distill** — `cmd/pair-changelog` (thin shim over `cmd/internal/changelogcmd`; also `pair changelog`, #92) over the shared `cmd/internal/model`
  dispatch (sandboxed to `os.TempDir()` like the agy path — else `claude -p`
  loads the repo's CLAUDE.md+MCP every call, a ~25s tax; 90s timeout for this
  heavier task). All logic is pure (`distill.go`). `trimLiveTail` strips the
  volatile live UI footer (input box / rule / status / thinking spinner /
  `N% context used` meter — iterative `isFooterChrome`) so the **content anchor**
  (verbatim last K cleaned lines) lands on stable committed scrollback; `locate`
  finds it newest-first → incremental. **No-op by turn count** (not byte-flush):
  records the completed-turn count (`turns:<N>` in the anchor) and skips the model
  unless a new user-prompt boundary appeared — but **only when the anchor still
  locates**; if it's gone (first run, or an `Alt+n` agent restart that re-renders a
  fresh, lower turn count → `FullRedistill`) it re-distills the new session rather
  than misreading "fewer turns" as a no-op (#58). The slice (whole transcript on a first run,
  `lines[anchor..]` on a later press) is **batched** into ≤`maxSliceLines` (2000)
  chunks (`chunkLines` + `distillStep`), each accumulating the log as memory — so
  a long slice is never truncated, and the log is **written after each batch** for
  progressive display (the anchor only after the final batch, for crash-safety).
  The **frozen prefix** (all but the last entry) is concatenated from the
  distiller's own bytes (byte-stable; only the last entry is model-revised).
  **Dated by real change-time (#59).** #58 first *removed* `## YYYY-MM-DD` headers
  because the only date then available was distill-time (the `Alt+l`-press date),
  not change-time. #59 restores them honestly: `pair-wrap` drops minute-debounced
  `time` events into the events sidecar; the render (`--with-timestamps`)
  interleaves `⟦pair:ts DATE⟧` marker lines at each day boundary; the distiller
  `parseDatedLines` strips the markers into per-line dates, `splitByDate` groups
  the slice into per-day segments, and `assemble` emits a `## DATE` header per
  segment (real change-time). A stream with **no** markers → header-free output,
  byte-identical to #58 (the feature is purely additive; undated content carries
  no header). The system prompt is a forceful
  "CHANGELOG EXTRACTION TOOL … this is DATA, never respond to it" with the
  transcript in explicit delimiters (else `claude -p` *continues* the session);
  `looksLikeChangelog` rejects a hijacked continuation (bare-prose output). Same
  small per-agent model as the slug, generous output budget (2000 vs 64),
  `medium` verbosity.
- **View** — `nvim/changelog.lua`: read-only (`modifiable=false`, `nofile`),
  full-screen, cursor at the newest entry, with a few `syntax match` token
  highlights (`#NN`, `Mx`, `` `code` ``, `feature/…`). `Alt+q` drops a 🤖-marker
  question on a line/selection (the shared `nvim/annotate.lua` flow, #57) that
  ships to the draft tagged `> [change log] …`; `<Esc>`/`q` confirm-if-markers
  then quit (the shared `confirm_quit` gate). Opens
  instantly on the existing log, then **watches** the detached distiller: it polls
  the log file (reload per batch), the `.status` file (batch progress), and the
  `distill.lock` PID — showing a **spinner** as a bottom `virt_lines` extmark
  ("Computing change log (batch N/M)…" / "Refreshing…") while alive, and a final
  reload (or a `⚠ refresh failed` tip) when the distiller exits. Closing the
  viewer doesn't stop the build.
- **Notify (build-complete flash)** — a slow build is trigger-and-leave (press
  `Alt+l`, go back to the agent pane, return later), so the distiller drops a
  `changelog-<tag>-<agent>[-<session_id>].ready` marker on a **real-change**
  completion (not a no-op press; keyed per session — see State below, #63). The
  draft nvim (`nvim/init.lua`) re-resolves the session id each tick and polls for
  the matching marker on a 2s timer — NOT
  fs_event (macOS FSEvents from nvim is unreliable: EMFILE/nil-filename; the
  scrollback-pending watcher only survives that via a FocusGained fallback this
  signal can't use, since its job is to fire while focus is elsewhere) — and on
  arrival flashes the **right end of the draft statusline** green (`✓ change log
  ready · Alt+l`) for ~2s via `pair_flash_notify`, then reverts to the cheatsheet,
  consuming the marker (one-shot). The draft statusline is always on screen, so the
  flash lands while the operator works in the agent pane (#58).
- **State** (`$PAIR_DATA_DIR`, per `(tag, agent, session)` — the base is
  `changelog-<tag>-<agent>-<session_id>`, keyed on the persisted agent session id
  so a fresh session (Alt+Shift+N) starts an **empty** log and a resume (Alt+n)
  reopens the **same growing** one; a different id is a different file, which *is*
  the reset — and each session's own `.anchor` removes the cross-session
  `FullRedistill` pile-up, #63). The id is resolved the same way by both builders
  (the opener `bin/pair-changelog-open` and the draft-nvim `.ready` watcher):
  the exported `PAIR_SESSION_ID` (set by the launcher at launch for claude-fresh /
  any resume) → the per-tag `config-<tag>-<agent>.json` `session_id` (the
  `pair-session-watch` codex/agy async path) → the **legacy unsuffixed base**
  when no id is known (backward compat). Files off `<base>`: `.md` (the log,
  plain markdown; `## YYYY-MM-DD` day headers from real change-time when the
  session has `time` events, header-free bullets otherwise — #59), `.anchor` (`turns:<N>` header + content
  snippet), `.cleaned` (transient rendered TTY), `.status` (distiller batch
  progress), `.ready` (build-complete marker the draft statusline polls),
  `.openlock` (viewer), `.distill.lock` (distiller PID). (Old per-tag logs are
  orphaned on the first post-#63 resume — harmless; reaping them is a follow-up.)

Tests: pure core + a process-level fake-model integration test in
`cmd/pair-changelog`; a headless viewer test (`nvim/changelog_test.lua`) and a
headless statusline-flash test (`tests/changelog-notify-test.sh`, `make
test-statusline`, which also covers the watcher's per-session keying — legacy /
`PAIR_SESSION_ID` / config-resolved, #63); an end-to-end orchestrator smoke
(`tests/changelog-open-test.sh`) plus a focused per-session keying test
(`tests/changelog-session-key-test.sh`), both under `make test-changelog`.

## Quit / restart semantics

Four ways to end (or refresh) a session, with different aftermath:

- **Alt+d** — detach. The session keeps running (claude/nvim processes alive); `pair` surfaces it in the picker for re-attach.
- **Alt+x** — full quit. Kills the session AND removes the resurrect entry. After Alt+x, the session is fully gone (but the `config-<tag>-<agent>.json` survives, so `pair resume <tag>` later replays the saved launch args + agent session id).
- **Alt+n** — reload pair. Kills the session AND keeps the saved `config-<tag>-<agent>.json` AND re-launches pair on the same tag with the same agent + args + agent session: the conversation resumes via `--resume <id>` (claude) or `resume <id>` (codex) or `--conversation <id>` (agy). Pair itself is the only thing that restarts — useful after a binary or config rebuild.
- **Shift+Alt+N** — restart with a fresh agent conversation. Same as Alt+n but drops `config-<tag>-<agent>.json` first, so the relaunched agent starts a brand-new session.

Mechanically Alt+n and Shift+Alt+N share two markers (`quit-` + `restart-`) plus a `PAIR_FORCE_TAG` env var on re-exec; the restart marker carries a `new_session` flag that selects the keep-vs-drop branch. See the launcher's "Reload / restart in place" section.

All three route through a Y/N confirm modal in nvim before firing, so a single fat-finger Alt-key can't tear the session down. The lua side auto-grows the nvim pane out of the `minimized` rung first, so the modal lands on visible rows.

Zellij's default `Ctrl+q` (Quit with resurrect) is **unbound** in pair's config — it would otherwise leave a half-state where the processes inside die but the session record stays as a "resurrect candidate," which is confusing for pair's long-lived-agent model. Alt+x is the only full-quit path.

## Tag-restart (issue #000016)

A pair *tag* is a durable identity for a coding session: it survives Alt+d (detach) trivially, and survives Alt+x because pair captures both the original launch args and the agent's own session id to disk, keyed by `(tag, agent)`. After Alt+x, the user sees a one-liner naming the resume command; running it short-circuits the picker and replays the saved configuration.

**Discovery — two layers.** The session id needs to be on disk by Alt+x time so `pair resume <tag>` can replay it. Two mechanisms, picked by agent and launch shape:

1. **Pre-write at launch (the launcher).** Two paths:
   - `--resume <id>` / `resume <id>` / `--conversation <id>` explicit on argv: pair writes `config-<tag>-<agent>.json` directly with that id, before zellij launch.
   - **Claude fresh launch (issue #000020):** claude supports `--session-id <uuid>`, so on the new-session path pair generates a v4 UUID, injects the flag into the agent argv, and writes the config synchronously *before* spawning the watcher. The id is deterministic from the launcher's perspective, so the watcher is a no-op for claude — and the cross-tag race that existed when two pair sessions shared a cwd is structurally eliminated.
2. **Watcher (`cmd/pair-session-watch` / `bin/pair-session-watch`, codex/agy only).** Spawned in the background by the launcher on the create path, right before the zellij launch — the launcher execs the Go binary directly (the `.sh` passthrough shim was retired in #94 M2). The stateful discovery logic lives in Go. Two discovery paths:
   - **PID-bound (preferred).** Reads `$PAIR_DATA_DIR/agent-pid-<tag>` (written by pair-wrap right after `pty.Start`) only when the pidfile's mtime is at-or-after the watcher's start, so a stale pidfile from a prior launch is ignored until pair-wrap overwrites it. Then it inspects open files in that PID's process tree via `lsof -p <pid> -Fn`. Race-free across concurrent pair sessions because lsof output is scoped to specific PIDs. Falls back internally to a birth-time-filtered directory walk if the agent doesn't keep its session file open: candidates are files with `stat -f %B >= agent_start_epoch`, and only a *single* candidate is accepted (multiple = concurrent race, refuse rather than guess).
   - **Legacy snapshot-diff (fallback).** Used when a fresh pidfile doesn't appear within 2s (`PAIR_SESSION_WATCH_PID_WAIT_SECONDS` in tests) — i.e., when the installed pair-wrap binary predates #000020 and doesn't publish the pidfile, or a stale pidfile is never refreshed. It snapshots the watch dir at start, scans new matching files, accepts the first candidate with a valid extracted id, and logs `near-miss` only when matching candidates cannot produce an id. Cross-tag races re-emerge in this path, so the proper resolution is to rebuild pair-wrap.

   Times out after 60s in either path.

Known gap: `/clear` rotates claude's session id mid-session, allocating a new jsonl that neither layer above sees. The launch-time `--session-id` is captured at create time, the watcher's 60s window is long gone by then, and there is no Alt+x trigger anymore. After a `/clear` + Alt+x, `pair resume <tag>` will replay the pre-clear conversation. (Pair previously sent a `bye\n` to the agent on Alt+x specifically to refresh the saved id past a `/clear`; that layer was retired because it polluted the conversation log and the rotation case is rare in practice. `/compact` doesn't rotate.)

Per-agent surface:

| Agent | Path | Id source | Capture mechanism |
|---|---|---|---|
| claude | `~/.claude/projects/<encoded-cwd>/<id>.jsonl` | filename | `--session-id` pre-injected by the launcher (deterministic) |
| codex | `~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<id>.jsonl` | trailing UUID in filename (regex) | `lsof -p <pid>` against agent PID + `ps`-discovered descendants, birth-time fallback |
| agy | `~/.gemini/antigravity-cli/conversations/<id>.db` | UUID database filename | `lsof -p <pid>` against agent PID + `ps`-discovered descendants, birth-time fallback |

**Stored shape.** `$PAIR_DATA_DIR/config-<tag>-<agent>.json`:

```json
{ "agent": "claude", "args": ["--dangerously-skip-permissions"], "session_id": "8d745d08-..." }
```

Single write posture: structured JSON plus temp-file rename, only after the id is in hand. The launcher writes the synchronous claude/explicit-resume prewrites, and the Go watcher writes the codex/agy config once the id is discovered — both via `encoding/json` plus `os.CreateTemp`/rename. So a concurrent reader either sees a complete prior config or a complete new one — never a partial. Keyed by `(tag, agent)` because the same tag can hold separate configs for different agents.

**Create-flow prompt (the launcher).** When the create path commits a tag, the launcher reads `config-<tag>-<agent>.json`. If present, it runs the per-agent stale-id check (claude: `[ -f .../<id>.jsonl ]`; codex: `find ~/.codex/sessions -name "*<id>*"`; agy: check conversation DB) and fzf-prompts the user with up to three options:

```
1) use params + session   args=[...]   resume=<id>
2) use params             args=[...]   fresh session
3) use none               args=[<current>]   fresh session
```

fzf renders each option multi-line via `--read0` so long args / full session ids stay visible without truncation. ESC aborts the create flow. Option 3 deletes the saved config before proceeding so the watcher writes a fresh one cleanly.

**Resume composition.** "use params + session" is per-agent because the resume surface differs:

- claude — flag style. Strip any pre-existing `--resume <X>` from saved args, then append `--resume <session_id>`.
- agy — flag style. Strip any pre-existing `--conversation <X>` from saved args, then append `--conversation <session_id>`.
- codex — subcommand. `codex resume <id>` is the syntax, so prepend `resume <id>` ahead of any saved flags. The strip phase also drops a leading `resume <X>` at args[0..1] from saved args (the codex case where the user originally launched with `codex resume <foo>`).

The shape `compose = saved_args (stripped of any prior resume tokens) + agent's resume invocation` keeps the composed line idempotent under repeated restarts.

**Post-Alt+x hint.** `cleanup_quit_marker` reads `agent-<tag>` *before* clearing it (so the hint names the right binary even though that file is about to disappear), then prints:

```
pair: saved session config for tag "pair-2" (claude).
      resume with: pair resume pair-2
```

`SESSION` rather than `PAIR_TAG` is shown — that's what the user just saw in the UI tab. `pair resume <tag>` accepts both forms (it strips a leading `pair-`).

## Tag rename (issue #000022)

A tag is durable but historically frozen-at-create. `pair rename <old> <new>` lifts that: every tag-scoped file in `$PAIR_DATA_DIR` is renamed in one transactional pass, so the agent's saved session, draft buffer, scrollback artefacts, log, queue, and per-pane pidfiles all follow the new name. Renaming is offline-only — zellij has no live-rename for a session, so the inside-session UX wraps quit → rename → re-exec around this primitive: Ctrl+Alt+n's confirm offers `&Yes / &No / &Rename`, and the (R) path prompts for a new tag, pre-validates via `pair rename --restart-check`, then triggers the restart with `--rename-to <new>`. Orthogonal to Shift+Alt+N's `--new-session` — rename + fresh agent is one gesture.

**File-family enumeration is the canonical place to look up "what is scoped to a tag."** The launcher walks two shapes:

1. **Tag-only families** (filename is `<prefix>-<tag>[<ext>]`, no further structure): `agent`, `agent-pid`, `agent-output`, `agent-picks`, `outer-tty`, `pair-wrap-pid`, `title-pid`, `layout-mode`, `queue` (dir), `quote`, `image-capture` + `.done`, `draft-<tag>.md`, `log-<tag>.md`, `nvim-pid-<tag>-{draft,scrollback}`.
2. **Per-(tag, agent) families** anchored on `config-<tag>-<agent>.json` — also `pane-<tag>-<agent>.json` (#71 frame-meter pane id + cwd), `scrollback-<tag>-<agent>.{ansi,raw,viewport,events.jsonl}` and the per-agent draft `draft-<tag>-<agent>.md`. The set of agent suffixes is hardcoded (`claude codex agy`) — adding a new agent to pair requires updating that list in lockstep.

**Substring safety is enforced by construction**, never by filtering. The enumerator computes exact filenames like `$DD/config-$old-claude.json`; it never globs `$DD/config-$old-*.json`. This is why `pair rename brain newname` cannot accidentally pick up `brain-2`'s files — the `brain-2`'s filenames are never constructed.

**Atomicity.** The full `(src, dst)` plan is written to `$PAIR_DATA_DIR/.rename-<old>-to-<new>.journal` before any `mv` runs. On mid-flight failure, the renamer reads the first N journal lines, swaps columns, and `mv`s the completed renames back to their original paths. The journal is cleared on success and retained on rollback failure as a forensic breadcrumb (M3 will add crash-recovery: a stale journal on startup gets finished or rolled back automatically).

**Refusals.** The CLI refuses upfront when: (a) `pair-<old>` or `pair-<new>` is in `zellij list-sessions` (live, detached, or resurrectable), (b) any file matching the `<new>` family exists, (c) `<old>` has no files. Tested via `tests/pair-rename.sh`. `--restart-check` skips (a) for `pair-<old>` only (the inside-flow case: `pair-<old>` is the current session, about to be killed) and exits without touching disk.

**Inside-flow choreography.** `nvim/init.lua`'s `pair_confirm_restart_impl` shells out `pair rename --restart-check` after the user enters a new tag, re-prompting on each rejection. On accept it execs `pair restart --rename-to <new>`. `pair restart` writes `rename_to=<new>` into the restart marker (`~/.cache/pair/restart-<SESSION>`) alongside the existing `tag`, `agent`, `new_session` fields. In the launcher's restart loop, `planRestart` reads the marker after `runCleanup` (so the zj delete-session has cleared the live-old gate) and if `rename_to` is set, invokes the native `rename <old> <new>` — full check. On success, the working tag for the re-launch is swapped to `<new>` (so `config-<new>-<agent>.json`, the just-renamed file, is what gets resumed). On failure, a 2-second visible stderr warning is printed and the restart continues with the original tag — the user is never stranded.

## Data layout

Drafts and prompt history live under `${XDG_DATA_HOME:-~/.local/share}/pair/` (per XDG Base Directory spec), keyed by tag (the agent name, or a custom name from the create-flow prompt):

- `draft-<tag>.md` — the active draft file (the `*` slot). Cleared by `send_and_clear` only when sending from `*`, persists across launches and navigation.
- `log-<tag>.md` — append-only log of every send, with timestamp. Doubles as the source for the `-N` history slots (parsed at navigation time). Searchable via `rg`.
- `queue-<tag>/NNNNNN.md` — one file per queued prompt (the `+N` slots). Filenames sort to display order (lowest = `+1`). Created lazily by `Alt+q` or auto-front-push from a dirty-`-N` "Queue" choice. Removed when the corresponding queue item is sent.
- `quote-<tag>` — transient hand-off file written by `bin/clipboard-to-pane` and read by nvim's `PairPasteQuote()`. Overwritten on every selection.
- `scrollback-<tag>-<agent>.raw` / `.events.jsonl` / `.ansi` — pair-wrap's raw PTY capture, the resize sidecar, and the rendered viewer file (#000017). The .raw + .events are written live during the session (truncated on each launch); the .ansi is regenerated on every `Alt+/` press and on scrollback-viewer `G` refresh (#84). Per (tag, agent) so multiple agents on the same tag don't clobber each other.

The launcher exports `$PAIR_DATA_DIR` so `nvim/init.lua` can compute the same path without re-deriving the XDG fallback chain.

Per-tag files mean `pair claude`, `pair codex`, and a custom-named `pair-bugfix` (entered at the prompt) all have independent draft state.

Internal: `~/.cache/pair/quit-<session>` — marker file used to communicate "user asked for full quit" between `pair quit` (or `pair restart`) and the launcher. Touched on Alt+x, Alt+n, and Shift+Alt+N; removed by the launcher after delete-session.

Internal: `~/.cache/pair/restart-<session>` — marker written alongside `quit-` by `pair restart` (Alt+n / Shift+Alt+N). Holds `tag`, `agent`, and `new_session` (0 = keep config and resume, 1 = drop config and start fresh) as `key=value` lines so the launcher can reconstruct the relaunch params after `cleanup_quit_marker` has wiped `agent-<tag>`. Removed by `handle_restart_marker` immediately before `exec`-ing pair on itself.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/outer-tty-<tag>` — single-line file containing the path to pair's controlling TTY at attach time. Read by `pair-notify` to emit OSC escapes that reach the outer terminal/wrapper. Rewritten on every attach (create or reattach); removed on full quit.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-<tag>` — single-line file recording which agent binary was launched in the session (`claude`, `codex`, ...). Written once at session create; read by `pair list` to display the agent column, and by the launcher's tag-restart agent-inference. Removed on full quit. The agent isn't otherwise recoverable post-create — env vars are frozen in pane shells, and custom session names (e.g. `pair-bugfix`) don't carry the agent in the name.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/config-<tag>-<agent>.json` — saved restart configuration for `(tag, agent)` (issue #000016, #000020). `{ agent, args, session_id }`. For claude, written synchronously by the launcher before zellij launch (`--session-id` is deterministic). For codex/agy, written by the Go `pair-session-watch` command once the agent's session file is discovered via lsof. Read by the launcher's create-flow prompt and by the post-Alt+x hint. Survives Alt+x (unlike `agent-<tag>`, which is cleared) — that's the whole point: it's the bridge between two pair launches against the same tag.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-pid-<tag>` — child agent PID written by `cmd/pair-wrap` immediately after `pty.Start`, removed on shutdown. Consumed by `cmd/pair-session-watch` to scope `lsof` discovery to a specific process tree (issue #000020). Mtime is also used as the agent-start epoch in the watcher's birth-time fallback.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/nvim-pid-<tag>-{draft,scrollback}` — single-line file containing the pid of an `nvim --embed` server child. Written at VimEnter by `nvim/init.lua` (for the draft pane) and `nvim/scrollback.lua` (for the Alt+/ floating viewer) when `$PAIR_NVIM_PID_FILE` is set; the launch sites (`zellij/layouts/main.kdl` for draft, `bin/pair-scrollback-open` for scrollback) export the env var pointing at a tag-scoped path. Read and removed by `cleanup_quit_marker` on Alt+x to SIGKILL the embed deterministically — without this, the embed sometimes survives zellij's pane teardown and accumulates as a PPID=1 orphan, dragging the host into memory pressure across many quits.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/pair-wrap-pid-<tag>` — single-line file containing pair-wrap's pid, written at startup by `bin/pair-wrap` if `PAIR_TAG` is set. Read by nvim's Alt+i (`attach_image`) so it can `kill -USR1 <pid>` to arm an image-capture window. Removed by pair-wrap on exit (the `finally` block in `main()`) and by `cleanup_quit_marker` as belt-and-suspenders on Alt+x.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/image-capture-<tag>` + `image-capture-<tag>.done` — paired files driving the Alt+i image-marker pickup. On SIGUSR1, pair-wrap buffers bytes from the agent's PTY for `PAIR_WRAP_CAPTURE_S` seconds (default 0.2), then writes the buffer to the first file and touches the `.done` sentinel. nvim polls the sentinel (20ms cadence, 600ms cap), reads the buffer, strips ANSI, regex-matches the agent's image marker (claude `[Image #N]`, agy `[Image N-M]`), and inserts it at cursor. Both files are removed by nvim after the pickup and by `cleanup_quit_marker` on Alt+x.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/slug-proposed-<tag>` and `slug-<tag>` — the orientation-slug channel (issue #000027). `pair-slug` (spawned by pair-wrap at turn-end) writes the proposed `=== <branch> | <focus> ===` to `slug-proposed-<tag>` (atomic temp+rename); nvim applies it to draft line 1 and writes the effective line back to `slug-<tag>`, which is the `prev` the proposer reads next turn. For Codex, if `config-<tag>-codex.json` is missing, `pair-slug` can recover the live rollout by reading `agent-pid-<tag>`, walking descendants via `ps`, and checking their `lsof` paths for `~/.codex/sessions/.../rollout-*.jsonl`. Agy has two artifacts: restart/session discovery uses `~/.gemini/antigravity-cli/conversations/<session_id>.db`, while transcript summarization still reads `~/.gemini/antigravity-cli/brain/<session_id>/.system_generated/logs/transcript.jsonl`. Codex model auth is API-key first, then Codex CLI subscription auth via `codex exec`. Single writer each, so the channel is race-free.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/adapt-<tag>.jsonl` — the adaptation flight recorder (issue #000045). One JSON line per harness-adaptation trigger (`{ts, comp, agent, aspect, signal, outcome, detail}`), appended concurrently by `cmd/pair-wrap`, `cmd/pair-slug`, `cmd/pair-session-watch`, and `nvim/adapt.lua` — all writing one shared schema. Truncated once at session launch by the launcher (so multi-process `O_APPEND` never races) and removed on full quit. Read by `doctor/doctor.sh` to surface integration drift (near-miss/fail signals). See `atlas/how-to-bring-up-a-new-harness-cli.md` §3 for the signal registry.

**Migration from v1:** the launcher detects old `~/scratch/pair-{draft,log}-*.md` files on startup and moves them to the new XDG location, stripping the redundant `pair-` prefix from filenames.

## Path resolution

The Go launcher's `RunLaunch` prepends `$PAIR_HOME/bin` to `$PATH` once at entry (via the pure `prependBinToPath`, `cmd/internal/launcher/pathenv.go` + `createflow.go`), before exec'ing zellij. zellij and all its child processes (panes — `pair-wrap`, `copy_command "copy-on-select"`, `Run "pair-help"`/openers, the nvim viewers) inherit the PATH and resolve `clipboard-to-pane`, `copy-on-select`, and the `pair` binary (e.g. nvim's `pair quit` / `pair restart` keybinds) by bare name. This lets the zellij KDL configs reference these helpers without `sh -c` env-var quoting hacks. The retired shell `bin/pair` did this prepend; the Go launcher that replaced it dropped it in #99 M5c — a real regression for a copied/Homebrew install whose `bin/` isn't already on the user's PATH — and #95 restored it (guarded by a `prependBinToPath` unit test + a copied-binary smoke asserting bare-name helper resolution).

## Binary freshness: deployed vs dev (`pair-dev`)

The Go binaries (`pair-wrap`, `pair-slug`, …) live in `$PAIR_HOME/bin` (first on PATH per *Path resolution* above) and, after `make install`, in `~/.local/bin`. `bin/` is **gitignored** — built on demand, absent in a fresh tree. Because the agent pane launches as `sh -c '… exec pair-wrap …'`, the wrapper is resolved by a **PATH lookup**: no shell function or `.zshenv` can intercept it (`exec` bypasses functions, and `sh` ≠ zsh), so `construct/dev-aliases.sh`'s rebuild-on-call freshness does **not** reach it. When `$PAIR_HOME/bin/pair-wrap` is stale or absent, PATH silently falls through to an old `~/.local/bin` copy and the running wrapper drifts from source — the failure mode is *silence*, not an error (diagnosed once via the #000045 flight recorder going quiet for every Go-emitted aspect while only nvim's Lua emitter still logged).

Two launch modes resolve this:
- **Deployed** — `pair`. Runs whatever prebuilt binary PATH finds; zero toolchain dependency. Keep `~/.local/bin` current with `make install`.
- **Dev** — `pair-dev` (#000046). Exports `PAIR_DEV=1` and execs `pair`; the launcher's `DevRebuild` then runs `make build` (still via `bin/lib/dev-rebuild.sh`'s `dev_rebuild`, sourced from Go) on the **create path**, before the layout execs pair-wrap, so `$PAIR_HOME/bin` holds a fresh build. Restart-safe: `PAIR_DEV` survives the launcher's in-process restart loop, so Alt+n / Shift+Alt+N rebuild too; a plain attach (no new wrapper spawned) correctly skips it. Deployed launches (`PAIR_DEV` unset) invoke no toolchain.

`pair-doctor` *diagnoses* the same staleness `pair-dev` prevents: its emitter-health probe (`doctor/emitter-health.sh`, #000047) greps the *running* `pair-wrap`/`pair-slug` (resolved via the `pair-wrap-pid-<tag>` pidfile, else PATH) for its adapt signal strings and flags `[STALE]` when a binary has no logging code — turning the silent-emitter failure into a named finding.

## Adjacent: `pair-scribe`

`scribe` is a `script(1)` replacement that lives in the pair repo for build-system convenience but is not part of pair's runtime — it's user shell tooling, typically wired at the top of `~/.zshrc` to swap for `script -q -F`. The user's preexec/precmd hooks send `SIGUSR1`/`SIGUSR2` to pause/resume the on-disk typescript around commands whose output (e.g. TUI redraws) shouldn't be captured, enabling a clean "capture last command output" flow that pair can read back from `$_ZSH_SCRIPT_LOG`. The logic lives in `cmd/internal/scribecmd` (`scribecmd.Run`) and is reachable two ways (#96): the `pair scribe` dispatcher route (streaming seam) and the standalone `cmd/pair-scribe` thin shim, which stays installed at `~/.local/bin/pair-scribe` after `make install` so the existing `~/.zshrc` `exec` line keeps working. Because it's shell tooling, not runtime, it is deliberately **not** in the runtime bundle. Full design notes and the zshrc snippet: `cmd/pair-scribe/README.md`.

## Design intent

- **Asymmetric panes by design.** Most chat UIs cram input and output into the same constrained box. The split makes the asymmetry explicit — agent owns *output*, nvim owns *input* — and lets each side specialize.
- **Selection is the gesture.** Click-and-drag in the agent pane, mouse up — the quote is in nvim, ready for your reaction. No keystroke between.
- **Self-contained.** Uses `--config-dir` and `nvim -u` to fully isolate from the user's normal configs. No invasive install.
- **Agent-agnostic.** Same plumbing works for any TUI agent that accepts typed input. Switching is one keystroke.
- **Prompt history is just a markdown file.** Aligns with the "data into central location, shell-ed agent runs free" pattern: every send appends to a grep-able log.

## Future work

Tracked in workshop issues. v2 candidates include a real nvim plugin (for users who want LSP/snippets/telescope inside the input pane).
