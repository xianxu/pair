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
- deciding final embedded-vs-adjacent asset packaging. #79 owns that decision.

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
| `bin/pair` / `cmd/internal/launcher` / `cmd/internal/entrypoint` | Bash public launcher plus Go-owned launch handoff | user shell, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` (#77) resolves sibling `bin/pair` from the `pair-go` executable and execs it with `pair`-compatible argv/env. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves the compatibility handoff; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Go handoff entrypoint with `bin/pair` retained as stable public launcher and compat target through #77; real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
| `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling `pair`. | Resolves symlinks; depends on `bin/pair` and `bin/lib/dev-rebuild.sh`. | compat-shim, likely stays as developer wrapper until Go entrypoint has dev mode | P1 |
| `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
| `zellij/layouts/main.kdl` | zellij native asset | `bin/pair` via `zellij --new-session-with-layout` | Defines agent and draft panes; shell expands Pair env at pane start. | Calls `pair-wrap`; calls `nvim -u "$PAIR_HOME/nvim/init.lua"`; writes `pane-<tag>-<agent>.json`; writes draft nvim pid file. | native-asset, packaged adjacent/embedded | P0 |
| `zellij/config.kdl` | zellij native asset | zellij session config from `bin/pair` | Global keybinds, copy command, scrollback buffer, pane frames. | Calls `copy-on-select.sh`, `pair-help`, `pair-scrollback-open`, `pair-changelog-open`; routes quit/restart/compact through nvim functions. | native-asset, packaged adjacent/embedded | P0 |
| `bin/pair-wrap` / `cmd/pair-wrap` | Go binary | zellij agent pane | `pair-wrap [--scrollback-log PATH] agent [args...]`; transparent PTY proxy; long-running; failure in diagnostics is swallowed. | Reads Pair env and agent command; writes `agent-output-<tag>`, `agent-pid-<tag>`, scrollback `.raw`/`.events.jsonl`, image capture files; may invoke `pair-slug`. | go-subcommand `pair wrap`; keep old binary as compat alias for KDL until caller moves | P0 |
| `bin/pair-slug` / `cmd/pair-slug` | Go binary | `pair-wrap` turn-end hook, tests | Env-driven, no stdin; resolves native transcript, proposes slug; exits 0 on most failures. | Requires `PAIR_TAG`, `PAIR_DATA_DIR`; reads config/transcripts/git branch; writes `slug-proposed-<tag>`; optional `PAIR_SLUG_*`, `OPENAI_API_KEY`. | go-subcommand `pair slug`; legacy binary retained during #76 | P1 |
| `bin/pair-context` / `cmd/pair-context` / `cmd/internal/contextcmd` | Go binary plus shared runner | `bin/pair-title.sh`; development-only `pair-go context` | `pair-context <tag> <agent>` and `pair-go context <tag> <agent>` print the same humanized token count or nothing; tolerant exit 0 on failure. | Reads `PAIR_DATA_DIR`, `pane-<tag>-<agent>.json`, config, native transcripts. | implemented helper route in `pair-go context`; legacy binary retained while title poller calls it | P1 |
| `bin/pair-scrollback-render` / `cmd/pair-scrollback-render` / `cmd/internal/scrollbackcmd` | Go binary plus shared runner | `bin/pair-scrollback-open`, `bin/pair-changelog-open`, `nvim/scrollback.lua` refresh; development-only `pair-go scrollback-render` | `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out` and `pair-go scrollback-render ...`; nonzero on render/write failure. | Reads `.raw` and `.events.jsonl`; atomically writes `.ansi` or cleaned text. | implemented helper route in `pair-go scrollback-render`; legacy binary retained for shell/Lua callers | P0 |
| `bin/pair-changelog` / `cmd/pair-changelog` | Go binary | `bin/pair-changelog-open` | `pair-changelog --cleaned F --log F --anchor F [--agent A] [--model M]`; exits nonzero on required read/model/write failure. | Reads cleaned scrollback/log/anchor; calls agent model through internal model runner; atomically writes log and anchor. | go-subcommand `pair changelog`; legacy binary retained for opener | P1 |
| `bin/pair-continuation` / `cmd/pair-continuation` | Go binary | nvim compaction prompt instructions, operator/agent shell | `pair-continuation --slug S --agent A --issues CSV --body-file F [--repo-root R ...]`; writes and commits continuation; nonzero on validation/git failure. | Reads body/stdin, git repo state; writes `workshop/continuation/*.md`; runs git commit/push. | go-subcommand `pair continuation`; legacy binary retained for agent instructions until docs change | P1 |
| `bin/pair-scribe` / `cmd/pair-scribe` | Go binary | user shell rc outside Pair sessions | `pair-scribe -log PATH -- CMD [ARGS...]`; long-running PTY wrapper; SIGUSR1 pauses log, SIGUSR2 resumes. | Writes typescript log; wraps child PTY; independent of `PAIR_*`. | go-subcommand candidate is low value; may remain separate installed helper or become `pair scribe` with alias | P2 |
| `cmd/internal/adapt` | Go helper package | `pair-wrap`, `pair-slug`, tests | Pure-ish emitter helpers plus file open seam; no command. | Writes `$PAIR_DATA_DIR/adapt-<tag>.jsonl`; schema shared with shell/Lua. | internal package, reuse behind dispatcher | P1 |
| `cmd/internal/ctxmeter` | Go helper package | `pair-context`, tests | Pure transcript token counting and humanization. | No direct IO. | internal package, keep | P1 |
| `cmd/internal/model` | Go helper package | `pair-slug`, `pair-changelog`, tests | Model runner/response parsing. | Calls external agent/model CLIs/APIs at command layer. | internal package, keep | P1 |
| `cmd/internal/transcript` | Go helper package | `pair-slug`, `pair-context`, tests | Resolves native transcript paths and session ids. | Reads Pair config and home paths via callers. | internal package, keep | P1 |
| `bin/pair-scrollback-open` | POSIX shell orchestrator | zellij Alt+/ Run, nvim Alt+b jump | `pair-scrollback-open [--jump prev|next]`; opens read-only nvim viewer; singleton lock. | Requires `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_HOME`; calls renderer, zellij IPC, nvim; writes `.ansi`, `.viewport`, lock. | shell-glue now; candidate Go orchestration after entrypoint, while `nvim/scrollback.lua` remains native | P1 |
| `nvim/scrollback.lua` | Neovim native asset | `bin/pair-scrollback-open` | Loaded by `nvim -u ... <ansi>`; interactive read-only viewer; refreshes backing render. | Reads Pair env and `.ansi`; may call `pair-scrollback-render`; writes pending marker files. | native-asset, adjacent/embedded | P0 |
| `bin/pair-changelog-open` | POSIX shell orchestrator | zellij Alt+l Run | Opens changelog viewer and starts detached render/distill singleton. | Requires Pair env; calls renderer, `pair-changelog`, setsid/perl, nvim; reads/writes `changelog-*` sidecars. | shell-glue now; candidate Go orchestration after entrypoint | P1 |
| `nvim/changelog.lua` | Neovim native asset | `bin/pair-changelog-open` | Loaded by `nvim -u ... <log>`; read-only watcher/spinner. | Reads `PAIR_CHANGELOG_*` and Pair env. | native-asset, adjacent/embedded | P1 |
| `bin/pair-title.sh` | Bash stateful poller | `bin/pair` ensure_title_poller | `pair-title.sh <tag> <agent>`; long-running 60s poller; test hook `PAIR_TITLE_TEST_CALL`. | Reads/writes title pid, pane json, cmux owner files; calls `pair-context`, zellij, ps, cmux. | stateful shell-glue; explicit #78 candidate | P1 |
| `bin/pair-session-watch.sh` | Bash stateful watcher | `bin/pair` create path | `pair-session-watch.sh <agent> <tag> <cwd> [agent-args...]`; background 60s watcher; no-op for claude. | Reads agent pidfile, lsof/ps, native session dirs; writes config JSON atomically; logs adapt events. | stateful shell-glue; explicit #78 candidate | P1 |
| `bin/lib/adapt-log.sh` | sourced shell helper | `pair-session-watch.sh` | `adapt_log comp agent aspect signal outcome [detail]`; no-op if no `PAIR_TAG` or jq. | Appends JSONL to `$PAIR_DATA_DIR/adapt-<tag>.jsonl`. | keep until shell emitters move; schema must stay DRY with Go/Lua emitters | P1 |
| `nvim/adapt.lua` | Lua helper | nvim doctor/adaptation surfaces, tests | Lua adaptation flight recorder emitter. | Writes same JSONL schema as Go/shell. | native-asset; keep schema aligned | P2 |
| `doctor/README.md` / `doctor/SKILL.md` | docs/skill | operator/agent diagnostics | Documents Pair doctor flow. | Refers to `nvim/doctor.lua` and adaptation logs. | adjacent docs/skill; not Go migration target | P3 |
| `nvim/doctor.lua` | Lua helper | `:PairDoctor` in nvim | Builds agent instruction payload. | Reads `PAIR_HOME`; sends text through draft/agent flow. | native-asset | P2 |
| `bin/pair-notify` | Bash notification helper | agent hooks/manual shell inside Pair | `pair-notify [--osc 9|777] "message"`; writes OSC to outer tty; nonzero on bad args/missing tty. | Requires `PAIR_TAG`; reads `outer-tty-<tag>`. | small shell-glue; possible Go subcommand but low packaging impact | P2 |
| `bin/pair-quit.sh` | Bash keybind helper | nvim `PairConfirmQuit` | Touch quit marker then kill zellij session. | Uses `ZELLIJ_SESSION_NAME`, `PAIR_KILL_CMD`; writes cache marker. | small compat shell; can fold into Go/nvim flow after entrypoint | P2 |
| `bin/pair-restart.sh` | Bash keybind helper | nvim restart confirmations | Writes restart marker then kill zellij session; supports `--new-session`. | Uses `PAIR_TAG`, `PAIR_AGENT`, `ZELLIJ_SESSION_NAME`, cache marker files. | small compat shell; can fold after entrypoint | P2 |
| `bin/pair-help` | Bash helper | zellij Alt+h Run | Displays `pair -h` through `less` with escape-to-quit behavior. | Calls `pair`, `less`. | compat-shim; may become `pair help` behavior | P2 |
| `bin/clipboard-to-pane.sh` | Bash copy/paste helper | `copy-on-select.sh`, direct zellij run possible | Reads OS clipboard, stages quote, focuses nvim, triggers Lua paste. | Uses pbpaste/wl-paste/xclip, jq, zellij, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_HOME`; writes quote and debug log. | shell-glue; keep until zellij copy flow has Go owner | P2 |
| `bin/copy-on-select.sh` | Bash copy_command helper | `zellij/config.kdl` `copy_command` | Reads selected text stdin, mirrors OS clipboard, flashes source, delegates paste unless selection was in nvim. | Uses pbcopy/wl-copy/xclip, jq, zellij, `PAIR_HOME`; calls flash and clipboard scripts. | shell-glue tied to zellij native surface | P2 |
| `bin/flash-pane.sh` | Bash visual helper | `copy-on-select.sh`, nvim flows/tests | `flash-pane.sh [pane-id]`; best-effort pane color flash. | Uses zellij, jq; reads `PAIR_FLASH_*`. | small shell-glue | P3 |
| `bin/pair-review-open` | POSIX shell review helper | nvim review flow | Validates target and opens floating `nvim -u nvim/review.lua`. | Requires Pair env; calls zellij/nvim. | shell-glue, review workbench can move later if packaging needs it | P2 |
| `bin/pair-review-readiness` | POSIX shell review helper | `nvim/init.lua` review readiness | Emits readiness data from git and target helper. | Uses `PAIR_HOME`, `PAIR_REVIEW_TARGET_BIN`, git/jq. | shell-glue; possible later Go helper | P2 |
| `bin/pair-review-target` | Bash review helper | review readiness/open/tests | Emits JSON target metadata under data dir. | Requires `PAIR_DATA_DIR`; reads config/pid files/lsof; writes `review-target-<tag>.json`. | shell-glue; possible #78 candidate if review packaging matters | P2 |
| `nvim/init.lua` | Neovim native asset | zellij draft pane | Main draft UI and Pair key handling. | Reads many Pair env vars/data files; shell-outs to zellij, pair quit/restart/open/review helpers. | native-asset; do not port, but audit shell-outs during #77/#78 | P0 |
| `nvim/review.lua` and `nvim/review/*.lua` | Neovim native review workbench | `pair-review-open`, draft review toggle | Review pane UI/modules. | Reads Pair env/data; calls docflow/agent seams through shell tests. | native-asset; adjacent/embedded | P2 |
| `nvim/annotate.lua`, `nvim/marker_codec.lua`, `nvim/pair_poke.lua`, `nvim/slug.lua`, `nvim/zellij_trace.lua` | Lua native helper modules | draft/viewer/review Lua | Pure or thin Lua helpers used by nvim surfaces. | Pair env/data files; zellij shell-outs in poke/trace surfaces. | native-asset | P2 |
| `Makefile` | build/workflow entry | developer/CI/SDLC | Includes workflow and local makefiles; `help` target. | Uses git remote; includes vendored base fragments. | packaging metadata; keep, update in #79 if install layout changes | P1 |
| `Makefile.local` | build/install/test metadata | developer/CI/`pair-dev` | Builds and installs Go binaries, runs test matrix. | Writes `bin/` and `~/.local/bin`; invokes Go, nvim, shell tests. | build contract; #74/#76/#79 must update as dispatcher changes | P0 |
| `README.md` | install/usage docs | users/package consumers | Homebrew install, CLI usage, keybindings, dev mode. | Documents dependencies and public commands. | docs to update at #77/#79 | P1 |
| `cmd/pair-scribe/README.md` | helper docs | users configuring shell logging | Documents `pair-scribe` install/usage. | No runtime behavior. | docs; update if `pair scribe` route added | P3 |
| `tests/*.sh`, `tests/lib/*`, `nvim/*_test.lua`, `cmd/**/*_test.go` | test-only seams | `make test`, Go test, headless nvim | Process fakes, shell integration tests, headless nvim drivers, Go unit tests. | Create temp dirs/fake PATH commands; exercise real scripts/binaries/Lua modules. | test-only; not migrated, but must move with callers | P3 |

## Hidden Caller Map

Zellij KDL callers:

- `zellij/layouts/main.kdl` launches `pair-wrap` by PATH and `nvim -u
  "$PAIR_HOME/nvim/init.lua"` by absolute env path.
- `zellij/config.kdl` calls `copy-on-select.sh`, `pair-help`,
  `pair-scrollback-open`, and `pair-changelog-open` by PATH.
- Quit/restart/detach/compact keybinds route through nvim functions first, then
  those functions call shell helpers or zellij actions.

Nvim shell-outs and binary dependencies:

- `nvim/init.lua` calls zellij actions, `pair-quit.sh`, `pair-restart.sh`,
  `pair-scrollback-open`, `pair-review-readiness`, `pair-review-open`, and uses
  `pair-wrap` pidfiles for image capture.
- `nvim/scrollback.lua` refreshes via `pair-scrollback-render`.
- `nvim/changelog.lua` watches files prepared by `pair-changelog-open`.
- `nvim/review.lua` loads review modules and is launched by `pair-review-open`.

Build/install callers:

- `make build` builds `GO_BINS` into `bin/`.
- `make install` copies `GO_BINS` to `~/.local/bin` and symlinks `SHELL_BINS`
  (`pair`, `pair-dev`) beside them so installed `pair-go launch ...` can resolve
  sibling `pair`.
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
  `bin/pair`, with argv/env preserved and missing-launcher diagnostics. The
  stable public `pair` script remains unchanged for this migration window.
- #78 should prioritize `pair-title.sh` and `pair-session-watch.sh` if stateful
  shell remains a packaging/reliability problem after #77.
- #79 owns whether `nvim/` and `zellij/` are embedded or installed adjacent.

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
- `bin/clipboard-to-pane.sh`
- `bin/copy-on-select.sh`
- `bin/flash-pane.sh`
- `bin/lib/adapt-log.sh`
- `bin/lib/dev-rebuild.sh`
- `bin/pair`
- `bin/pair-changelog`
- `bin/pair-changelog-open`
- `bin/pair-context`
- `bin/pair-continuation`
- `bin/pair-dev`
- `bin/pair-help`
- `bin/pair-notify`
- `bin/pair-quit.sh`
- `bin/pair-restart.sh`
- `bin/pair-review-open`
- `bin/pair-review-readiness`
- `bin/pair-review-target`
- `bin/pair-scribe`
- `bin/pair-scrollback-open`
- `bin/pair-scrollback-render`
- `bin/pair-session-watch.sh`
- `bin/pair-slug`
- `bin/pair-title.sh`
- `bin/pair-wrap`
- `cmd/internal/adapt/adapt.go`
- `cmd/internal/adapt/adapt_test.go`
- `cmd/internal/contextcmd/contextcmd.go`
- `cmd/internal/contextcmd/contextcmd_test.go`
- `cmd/internal/ctxmeter/ctxmeter.go`
- `cmd/internal/ctxmeter/ctxmeter_test.go`
- `cmd/internal/dispatcher/dispatcher.go`
- `cmd/internal/dispatcher/dispatcher_test.go`
- `cmd/internal/model/model.go`
- `cmd/internal/model/model_test.go`
- `cmd/internal/scrollbackcmd/events_test.go`
- `cmd/internal/scrollbackcmd/render_test.go`
- `cmd/internal/scrollbackcmd/scrollbackcmd.go`
- `cmd/internal/scrollbackcmd/scrollbackcmd_test.go`
- `cmd/internal/scrollbackcmd/serialize_row_test.go`
- `cmd/internal/scrollbackcmd/timestamps_test.go`
- `cmd/internal/transcript/transcript.go`
- `cmd/internal/transcript/transcript_test.go`
- `cmd/pair-changelog/distill.go`
- `cmd/pair-changelog/distill_test.go`
- `cmd/pair-changelog/e2e_test.go`
- `cmd/pair-changelog/main.go`
- `cmd/pair-changelog/main_test.go`
- `cmd/pair-changelog/prompt.go`
- `cmd/pair-changelog/prompt_test.go`
- `cmd/pair-context/main.go`
- `cmd/pair-context/main_test.go`
- `cmd/pair-continuation/continuation.go`
- `cmd/pair-continuation/continuation_test.go`
- `cmd/pair-continuation/git.go`
- `cmd/pair-continuation/main.go`
- `cmd/pair-continuation/main_test.go`
- `cmd/pair-go/helper_equivalence_test.go`
- `cmd/pair-go/main.go`
- `cmd/pair-scribe/main.go`
- `cmd/pair-scrollback-render/main.go`
- `cmd/pair-slug/main.go`
- `cmd/pair-slug/main_test.go`
- `cmd/pair-slug/slug.go`
- `cmd/pair-slug/slug_test.go`
- `cmd/pair-wrap/adapt_drift_test.go`
- `cmd/pair-wrap/extract_fg_test.go`
- `cmd/pair-wrap/keymap_registry_test.go`
- `cmd/pair-wrap/main.go`
- `cmd/pair-wrap/osc_test.go`
- `cmd/pair-wrap/overlay_test.go`
- `cmd/pair-wrap/picker_overlay_test.go`
- `cmd/pair-wrap/stdout_filter_test.go`
- `cmd/pair-wrap/time_event_test.go`
- `cmd/pair-wrap/translate_stdin_test.go`
- `cmd/pair-wrap/translate_test.go`
- `cmd/pair-wrap/update_agent_output_test.go`
- `cmd/pair-wrap/wrap_events_test.go`
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
