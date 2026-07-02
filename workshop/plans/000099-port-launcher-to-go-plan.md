# Plan: port the pair-shell launcher to Go (#99)

Extracted from #93 M5. Ports `bin/pair-shell` (2287 lines) — the last and largest
shell orchestrator — onto the `cmd/internal/launcher` pure core, behind a new
`launcher.Runtime` effect seam, retaining a compatibility shim. zellij/nvim stay
external (#95 boundary). Follows the M1–M4 template proven in #93.

## What already exists vs the gap (survey, 2026-07-02)

`cmd/internal/launcher` (from #75) already implements the **entire decision
phase**, well unit-tested — but it is a *prototype currently bypassed*:
`cmd/pair-go` `syscall.Exec`s `bin/pair-shell` with argv `["pair", …]` + `PAIR_HOME`
and the decision core never runs on the live path. Done already:
`ParseArgs` (decision subset — refuses `continue`/`rename`/`list`/`ls`),
`NormalizeTag`, `DefaultTag`, `ResolveDataDir`, `DecideLaunch`
(+ `nextFreeTag`/`sessionBlocksReuse`/`isHistorical`/`sessionName`),
`ZellijSource.Snapshot` (session classification), `HistorySource.Scan`, and
`Run(argv, env, sessions, history) → LaunchOutcome` (decision only — no exec).

Rough size of `bin/pair-shell`: ~600 lines pure logic (~26%, much already ported),
**~900 lines IO orchestration (~39%, the real work — no Go home)**, ~90 lines that
already delegate to Go (~4%), ~700 comments/help (~31%).

The gap set (no Go home) — all stateful:
- the two **blocking zellij handoffs** (`attach`, `--new-session-with-layout`);
- three **UIs**: fzf session picker, fzf config/tag-restart picker (#000016), zsh
  `vared` editable name-prompt (bash 3.2 lacks `read -i`);
- **restart/quit lifecycle**: `handle_restart_marker` (re-exec `$0`),
  `cleanup_quit_marker` (~130 lines: delete-session, reap nvim, park-nudge, rm
  sidecars, kill poller, release cmux), `park_scrollback`;
- **cmux** ownership + rename (presence-beats-stale owner file, emoji title subst);
- **config/session migration**: `resolve_config_file` (legacy `-codex-codex`),
  `~/scratch`→XDG one-time migration, `agent_session_exists`, tag-restart config
  picker + per-agent resume-token compose;
- **per-agent launch args**: claude deterministic `--session-id` mint (uuidgen +
  collision retry), codex `--no-alt-screen` idempotent strip/append, explicit-resume
  config writes;
- **nvim orphan reaping**: `reap_nvim_for_tag`, `sweep_orphan_nvim`;
- **guards/effects**: `in_zellij_pane` (PPID ancestry), `record_outer_tty`, env
  exports, dev-rebuild;
- **subcommands**: `list`/`ls`, `rename` (self-contained, ~240 lines), `help`;
- **two child-spawns**: `ensure_title_poller` → Go `pair-title`, session-watcher →
  Go `pair-session-watch` (both already Go — only the *spawn* is shell).

Integration points (already Go — wire, don't re-port): `bin/pair-title.sh`,
`bin/pair-session-watch.sh`, `bin/pair-wrap`, and the `$0` self-re-exec (restart /
in-session-compaction) which becomes an **in-process loop**, not a subprocess.

## Core architectural move (ARCH-DRY, ARCH-PURE)

Build one native orchestration entry `launcher.RunLaunch(...)` on top of the
existing pure core, behind a **new `launcher.Runtime` effect seam** (the M1–M4
`OSRuntime`+`osfs.FS` pattern — the launcher today has only the two narrow
`SessionSource`/`HistoricalScanner` sources, not a unified effect seam). The seam
covers: zellij exec/query (`zj` timeout wrapper + blocking attach/new-session),
fzf/prompt UIs, marker read/write, cmux, config read/write (jq → `encoding/json`),
nvim reap, child-spawns, tty, env. Pure decisions stay pure and unit-tested;
`RunLaunch` drives decision → effects → blocking handoff → post-handoff
cleanup/restart, and is exercised by a fake-`Runtime`.

## Compatibility shim strategy

End-state: the Go `pair` binary runs the launcher **in-process** (no exec to
`bin/pair-shell`); `bin/pair-shell` becomes a thin shim → `pair-go launch` for any
residual external caller; the restart re-exec becomes an in-process loop. During
transition, keep the existing `entrypoint.ResolveLegacyLaunch` + `legacyRuntime.Exec`
path (cmd/pair-go/main.go — the effect seam to widen) as a **flag-gated fallback**
(`PAIR_NATIVE_LAUNCH`), so `bin/pair-shell` remains the default until native
parity is proven, then the default flips and the shell path is retired.

## Phased plan (each an `Mx` review boundary, independently mergeable, M1–M4 template)

Boundaries are tagged `Mx` (not `Lx`) — `sdlc`'s boundary discovery + the
final-close milestone-verdict gate only recognize `M\d+`, so `Lx` rows would make
that gate a silent no-op (change-code plan-quality finding, 2026-07-02).

- **M1 — pure-logic completion (no wiring, zero behavior change).** Port the
  remaining pure pieces into `launcher`: full `ParseArgs` (`continue`/`rename`/
  `list`), resume-token strip/compose (4 duplicated shell loops → one helper —
  ARCH-DRY), config-migration decision rules, per-agent launch-arg composition
  (claude session-id shape, codex alt-screen idempotence), `rename` plan-build
  (`rename_paths_for` enumeration + transform), title/`format_age`/`age_color`
  formatting. Unit-tested directly.
- **M2 — Runtime seam + create-flow orchestration.** Define `launcher.Runtime`;
  build `RunLaunch` for the **create** path (native create behind
  `PAIR_NATIVE_LAUNCH`; shell stays default). **`RunLaunch` stays a thin
  orchestrator over the pure deciders — the tag-restart picker selection,
  per-agent-arg composition, etc. are pure functions fed by the Runtime, not
  branching business logic inline** (ARCH-PURE). Fake-`Runtime` tests: create,
  name-prompt, tag-restart config picker.
- **M3 — attach / restart / quit / compaction orchestration.** Native attach; the
  restart-marker re-entry as an **in-process loop** (not `exec $0`); in-session
  compaction; quit cleanup (`cleanup_quit_marker` — the ~130-line effect
  sequence). Fake-`Runtime` loop tests for each.
- **M4 — cutover.** Flip `cmd/pair-go` to run the native launcher in-process
  under `PAIR_NATIVE_LAUNCH`; convert `bin/pair-shell` to a thin shim →
  `pair-go launch`. Full e2e vs the shell (create/attach/restart/quit/compaction),
  then flip the default.
- **M5 — subcommands + retirement.** Port `list`/`rename`/`continue`; retire the
  shell fallback + `bin/pair-restart.sh` markers → in-process; drop the flag;
  resolve `bin/pair-shell` shim-vs-remove via an explicit `git ls-files bin/` +
  caller check. This is what lets #94 (stop extracting a shell tree) proceed.

## Tests

Follow the M1–M4 convention exactly: pure decisions unit-tested directly; the
orchestration driven by a fake `Runtime`; the concrete `OSRuntime` sources tested
against on-disk/exec fixtures (the established `ZellijSource` bash-stub +
`HistorySource` sidecar-file pattern). Keep the existing `PAIR_TEST_CALL` /
`PAIR_DEBUG_*` shell contract tests green against whichever launcher is active per
phase; add Go coverage for every gap-set behavior before retiring its shell.

## Verification

Per phase: `go test ./cmd/internal/launcher …` green; the launcher shell tests
(`tests/*launch*`, `PAIR_TEST_CALL` seams) green; a real create + attach +
restart + quit + compaction exercised end-to-end (this is a lifecycle port —
process-level fakes miss interaction bugs, so drive the real flow); drift-check
clean; `git ls-files bin/` shows `bin/pair-shell` as a thin shim by M4.

## Atlas (per-milestone)

Update `atlas/go-migration-inventory.md` (the `bin/pair`/`pair-shell`/`launcher`/
`entrypoint` row → Go-owned; Coverage Ledger) and `atlas/architecture.md` (the
launch-flow section — the Go↔shell boundary moves each phase) at each `Mx` close.

## Revisions

### 2026-07-02 — extracted from #93 M5

The design was surveyed + approved in the #93 plan and moved here on the operator's
call to make the launcher its own ticket (#99). No content change vs the #93 M5
detail; this file is now the record of truth.

### 2026-07-02 — M1 shipped surface (matches issue #99 M1) + review follow-ups

- **M1 scope reduced.** The M1 bullet above still lists "full `ParseArgs`
  (`continue`/`rename`/`list`)" and "`rename` plan-build" as M1 work; both were
  deferred out of M1 (to M2 / M5) — front-loading only the create/restart-flow pure
  logic M2/M3 need, avoiding unwired M5-only code and a risky change to the live
  `pair-go launch` parser. M1 shipped `agentargs.go` (per-agent resume compose,
  codex alt-screen idempotence, claude session-id mint/skip, flag strip helpers),
  `config.go` (config paths + legacy-codex migration decision + transcript paths),
  `format.go` (age/title formatting).
- **M1 milestone-close review (FIX-THEN-SHIP → SHIP).** No Critical. Fixed the one
  Important: the persist-strip only covered claude's `--session-id`/`--resume`, but
  `composeResumeArgs` handles all three agents — so `persistedConfigArgs` now also
  strips agy `--conversation` (space + inline `=` forms) and codex's leading
  `resume <id>` (position-sensitive), with tests, so an agy/codex resume can't
  compound in saved args (shell 2079-2082's guard). Minor: dropped the hand-rolled
  `itoa`/`itoa64` for `strconv` (ARCH-DRY); noted `TildeAbbrev`'s `home==""` guard
  as a defensive extension. The review ran via `sdlc judge milestone-review --base
  <branch-base>` because the auto-window picked a wrong far-back base (6.77 MB diff
  → `fork/exec claude: argument list too long`); see the issue Log + lessons.
