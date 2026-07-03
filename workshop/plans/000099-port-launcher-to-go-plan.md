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

### 2026-07-02 — M3 shipped surface (scope narrowed vs the M3 bullet)

The M3 bullet above over-scoped by listing **in-session compaction** as M3 work.
M3 shipped native **attach**, the in-process **restart loop** (Alt+n resume /
Shift+Alt+N fresh, replacing `exec $0`), **quit cleanup** (`cleanup_quit_marker`
— delete-session + nvim reap + gated park-nudge + sidecar removal + resume hint +
poller kill + cmux reset), and **nvim reap/sweep**. Deferred to **M5** (they
couple to M5's fzf picker + `continue` parsing): in-session compaction detection,
the `continue`/`rename` restart re-entries, and the fzf session **pick**. The
deferral is safe — in-pane launches, `ActionPick`, and `continue`/`rename` restart
markers resolve to `ErrFallbackToShell` → `bin/pair-shell`, so no partial native
path ships. `RestartMarker.RenameTo`/`Continue` + `restartPlan.ShellFallback` are
already the seam M5 converts to native. Shell stays default until the M4 cutover.
(M3 milestone-review FIX-THEN-SHIP; the two Importants were doc-accuracy — this
Revision — and recording the exec-seam boundary smoke in the close evidence.)

### 2026-07-02 — M4/M5 scope corrected (pre-implementation; plan-quality FAILURE fix)

The **M4 bullet is not executable as written** — it pairs "flip the default to
native" with "convert `bin/pair-shell` to a thin shim → `pair-go launch`". But M3
deliberately routes the fzf session **pick**, in-session **compaction**, and the
**continue/rename** restart re-entries to `ErrFallbackToShell` → the *real*
`bin/pair-shell`. A shim in M4 would loop:
`native → ErrFallbackToShell → bin/pair-shell (shim) → pair-go launch → native → …`.

**Corrected split:**
- **M4 = flip the default ONLY.** Make the native launcher run by default
  (native-first), gated by a `PAIR_LEGACY_LAUNCH=1` **kill-switch** that forces the
  shell for the whole launch (rollout safety; dropped in M5), replacing the M2/M3
  opt-in `PAIR_NATIVE_LAUNCH` gate. `bin/pair-shell` is **retained as the real
  fallback launcher** for the still-`ErrFallbackToShell` surfaces — NOT shimmed.
  The native launch moves behind the `cmd/pair-go` `legacyRuntime` seam so the flip
  is unit-testable without real zellij (ARCH-PURE). **Verification** must assert
  BOTH: (a) create / attach / Alt+n / Shift+Alt+N / quit run natively by default;
  AND (b) the still-deferred surfaces (pick / compaction / continue+rename restart)
  still reach the real `bin/pair-shell` and do **not** loop.
- **M5 = the actual retirement.** Port the remaining flows native — `list` /
  `rename` / `continue`, the fzf session **pick**, in-session **compaction**
  detection, and the **continue/rename restart re-entries** — so NO flow needs the
  shell; only THEN convert `bin/pair-shell` to a thin shim (or remove it), retire
  `bin/pair-restart.sh` markers → in-process, and drop `PAIR_LEGACY_LAUNCH`.
  "Retire the shell fallback" is a strict superset of the M4 bullet's premature
  shim (ARCH-PURPOSE: the single-Go-owner purpose only lands when *every* consumer
  flow derives from Go — a shim in M4 would falsely claim the retirement while
  pick/compaction/continue/rename still require the real shell).

This narrows the M4/M5 `## Phased plan` bullets above (they stay as the original
record; this Revision supersedes them), mirroring the M1/M3 deferral precedent.
The crux was first caught in the M4 continuation and is pinned here in the durable
plan per the M4 change-code plan-quality FAILURE (the plan is the record of truth;
an agent reading only the plan must not walk into the loop).

### 2026-07-02 — M5 split into M5a/M5b/M5c (pre-implementation, at start-plan)

M5 as a single boundary (pick + list + compaction + continue/rename restart +
retirement ≈ 3.2h) bundled **three distinct risk profiles plus a load-bearing,
irreversible retirement** into one review — the change-code plan-quality judge
flagged it (INFO), and the M4 status recommended splitting. Split into three
review boundaries, each closed on its own (`sdlc milestone-close`; the final one
`sdlc close --issue 99`). **Lettered tags M5a/M5b/M5c** — `sdlc`'s boundary
discovery + milestone-verdict gate match `M\d+[a-z]?` (verified: `close.go`
`milestonePlanRE`, whose doc-comment lists `M4b`), so a letter suffix is
recognized; the lettering keeps the "M5 = final milestone" framing (contrast the
earlier plan note, which understated the regex as `M\d+`).

- **M5a — read-only surfaces (lowest risk; mostly pure).** Port the fzf session
  **pick** (`ActionPick`, the `runOnce` `default:` arm that today → `ErrFallbackToShell`)
  and the `list`/`ls` subcommand. **Pure/IO seam (ARCH-PURE):** enrich the snapshot
  — `HistorySource.Scan` already computes each tag's mtime (`latest[tag]`) but
  discards it; project it onto `HistoricalTag.MTime`, and add a queue-count source —
  so the pick-**row** build (detached-first ordering, `FormatAge` grey-grading, the
  amber `[⏎ N queued]` badge, the `+ new` label) is a **pure function fed by the
  enriched snapshot**; only the fzf call + queue-count read are Runtime effects. The
  **queue-count source is concrete** (`queue_count_for`, shell 1335): the number of
  `[0-9]*.md` files under `$DATA_DIR/queue-<tag>/` — a `QueueCount(tag) int` Runtime
  effect; the badge text is pure over that int.
  **Reuse M2's `UIOps.PickFromList`** (ARCH-DRY — same fake-`Runtime` pattern as the
  config picker). Map the pure selection back into the existing `runAttach`/`runCreate`
  paths (picked live tag → attach; historical/`+ new` → create). `ParseArgs` gains
  `list`/`ls` (M1-deferred); `runList` is pure formatting over one snapshot read and
  exits (no zellij handoff). **Tests:** pure pick-row build (rows/order/badge from the
  enriched snapshot), fake-`Runtime` pick selection → attach vs create, pure `list`
  formatting.
- **M5b — lifecycle write flows (highest risk; stateful).** Port in-session
  **compaction** and the **continue/rename restart re-entries**. (a) The
  `InZellijPane()` guard (today → `ErrFallbackToShell`) becomes the native #55
  compaction branch: `park_scrollback --copy` + write the restart marker
  (`continue=<slug>`, `new_session=1`). (b) `planRestart`'s `RenameTo`/`Continue`
  arms (today `ShellFallback`) go native: **`rename`** runs the **M1-deferred
  `rename_paths_for` plan-build** — a pure enumeration + path transform, unit-tested
  directly (ARCH-PURE); **`continue`** re-seeds the draft from the slug then
  re-launches fresh. `ParseArgs` gains `continue`/`rename` (M1-deferred). **Tests:**
  pure `rename_paths_for` plan-build, compaction-marker detection, fake-`Runtime`
  `planRestart` rename/continue loop tests, a real-OSRuntime smoke driving a native
  rename + continue restart (the exec seam fakes can't cover).
- **M5c — retirement (LAST; ARCH-PURPOSE).** Only lands once M5a+M5b leave **no flow
  needing the shell**. Convert/remove `bin/pair-shell` (**caller check:** `git
  ls-files bin/` + grep the tree — **remove** if nothing outside the Go binary
  invokes it [#94's point], else a thin shim → `pair-go launch`); port `pair --help`
  natively (args.go's leading-flag → shell fallback has no shell to defer to now);
  convert the **defensive** `ErrFallbackToShell` returns (Sessions/ScanHistory/
  DecideLaunch errors in `runOnce`) into real error exits; retire `bin/pair-restart.sh`
  markers → in-process; drop `PAIR_LEGACY_LAUNCH` + the whole `ErrFallbackToShell`
  fallback arm in `cmd/pair-go/main.go`. Close with **`sdlc close --issue 99`** (not
  milestone-close) — this closes the issue, unblocking #93/#94. **Review must verify
  NO surface reaches a (now-absent) shell** (the shadow-sweep: every consumer flow
  derives from Go).

**Sequencing (ARCH-PURPOSE):** retirement is #99's *actual purpose* (single Go
owner; the thing that unblocks #94), so it must land **complete and last** — a
shim/removal before pick+compaction+continue+rename are native would either loop
(native → `ErrFallbackToShell` → shim → native) or falsely claim the single-owner
purpose while those flows still require the real shell. **`bin/pair-shell` fate is
decided empirically at M5c** via the caller check (leaning remove); a thin shim
only survives a real external caller.
