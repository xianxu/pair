---
issue: 000093
created: 2026-07-01
---

# Plan — port stateful shell orchestrators to Go (M1–M5)

Port the shell orchestrators into Go one merge-safe milestone at a time, each
following the #78 precedent (`bin/pair-session-watch.sh` → `cmd/pair-session-watch`
+ `cmd/internal/sessionwatch`, `.sh` kept as a thin re-exec shim). `ARCH-PURE`:
pure decision logic in unit-tested packages; zellij/cmux/nvim/fs interaction
behind a thin, process-tested `Runtime` seam. `ARCH-DRY`: reuse the existing
internal packages (`transcript`, `ctxmeter`, `contextcmd`, `sessionwatch`,
`launcher`) rather than reimplementing.

Ordering (leaf orchestrators first, launcher last — shrink `bin/pair-shell`'s
dependency set before replacing it): M1 title poller → M2 scrollback/changelog
openers → M3 review helpers → M4 clipboard helpers → M5 launcher.

Each `Mx` is its own `sdlc milestone-close` review boundary. This plan details
**M1** (implementing now); M2–M5 are milestone-level and will be detailed as each
is reached.

## The port template (from #78, verified)

- `cmd/internal/<name>/` package split into: pure domain logic +
  `Runtime` interface (the IO/process seam) + `run.go` (loop over the seam) +
  `runcli.go` (`RunCLI(args, getenv, stderr) int`) + `runtime.go` (`OSRuntime`).
- `cmd/pair-<name>/main.go`: 1-line `os.Exit(<pkg>.RunCLI(os.Args[1:], os.Getenv, os.Stderr))`.
- `bin/pair-<name>.sh`: thin shim — resolve `PAIR_HOME`, check the Go binary
  exists, `exec "$PAIR_HOME/bin/pair-<name>" "$@"`.
- Tests inject a mock `Runtime` to unit-test the loop without live zellij/cmux.

## M1 — title poller (`bin/pair-title.sh` → Go)

`bin/pair-title.sh` (338 lines) owns two surfaces: (1) always-on zellij FRAME
meter per agent pane — `"<agent> (<count>) [<cwd>]"`; (2) cmux WORKSPACE title
heat-ramp emoji (cmux-only). Single-instance per tag via pidfile with an
identity-checked liveness guard; self-terminates when the `pair-<tag>` session
disappears (miss-threshold debounced); 30s startup grace for the create-path race.

### New package `cmd/internal/titlepoller/`

- **`titlepoller.go` — pure decisions (direct unit tests, no IO):**
  - `prefixForAge(age time.Duration) string` — the 1d/3d/10d/21d heat buckets →
    🔴/🟠/🟡/🔵/"" (+ trailing space), CJK-wide emoji preserved.
  - `abbrevCwd(path, home string) string` — `$HOME`→`~` on a path boundary.
  - `frameTitle(agent, count, cwdDisp string) string` — `"<agent> (<count>) [<cwd>]"`
    or `"<agent> [<cwd>]"` when count is empty.
  - `cmuxWorkspaceTitle(prefix, session string) string` — prefix + session with
    the `brain→🧠 / book→📗 / pair→♋` substitutions.
  - `pollerArgvMatches(argv, tag string) bool` — the identity guard: argv
    contains `pair-title.sh <tag> ` (trailing space so tag 21 ≠ 211). The shim
    keeps that argv shape so the guard still recognizes a live poller.
  - `frameCache` — per-pane-id last-title map with unchanged-skip; `latest(sources)`
    picks the max mtime; `shouldClaimWorkspace(owner, tag string, ownerAlive bool) bool`
    — cmux-owner takeover decision.
- **`Runtime` interface (the seam):** `Now()`, `Sleep(d)`;
  `SessionAlive(session) bool` (`zellij list-sessions --short` exact-match);
  `RenamePane(session, paneID, title) error`; `CmuxAvailable() bool` +
  `CmuxRenameWorkspace(title) error`; `ProcessAlive(pid) bool` (`kill -0`) +
  `ProcessCommand(pid) string` (`ps -p <pid> -o command=`);
  `PaneFiles(dataDir, tag) []PaneInfo` (glob `pane-<tag>-*.json`, JSON-decode
  pane_id/cwd/cwd_display); `ContextCount(tag, agent) string` (in-process:
  `transcript.SessionID`+`transcript.Resolve`+`ctxmeter.ContextTokens`+`Humanize`
  — NO `pair context` subprocess, `ARCH-DRY`); `ActivityMTime(tag, agent) time.Time`
  (max mtime of `draft-<tag>.md` + resolved agent transcript); pidfile
  read/write/remove; owner-file read/write; `Log(...)` (adapt recorder).
- **`run.go`** — the loop: single-instance pidfile guard (identity-checked),
  30s startup-grace wait for the session, `SESSION_MISS_THRESHOLD=5` debounced
  self-terminate, frame-meter refresh gated on `age < 2*interval`, cmux
  workspace-title block gated on `CmuxAvailable() && CMUX_WORKSPACE_ID set`.
  `POLL_INTERVAL=60`. `trap '' HUP` → `signal.Ignore(syscall.SIGHUP)`.
- **`runcli.go`** — `RunCLI(args, getenv, stderr) int`: parse `[tag, agent]`
  (no-op if <2), resolve `PAIR_DATA_DIR`, open the adapt logger, wire `OSRuntime`.
- **`runtime.go`** — `OSRuntime` implementing the seam (reuse sessionwatch's
  `stat -f %B`/`ps`/`kill` shell-out idioms where they overlap).

### Shim + wiring

- `cmd/pair-title/main.go` — 1-line `RunCLI` entry.
- `bin/pair-title.sh` — replace the 338-line body with the session-watch-style
  shim (`exec "$PAIR_HOME/bin/pair-title" "$@"`). `bin/pair-shell`'s
  `ensure_title_poller` still calls `bin/pair-title.sh <tag> <agent> & disown`
  unchanged — the argv the guard matches is preserved by the shim.
- Makefile: add `pair-title` to `GO_BINS`, `.PHONY`, a per-binary target, and
  `RUNTIMEBUNDLE_HELPERS` (bin/pair-shell execs the shim which execs
  `$PAIR_HOME/bin/pair-title`, so the Go binary must be bundled). `make build`
  auto-discovers `cmd/pair-title/`. Regen the runtime bundle.

### Tests (M1)

- Go unit tests in `titlepoller`: `prefixForAge` bucket boundaries;
  `abbrevCwd`; `frameTitle` both shapes; `cmuxWorkspaceTitle` substitutions;
  `pollerArgvMatches` (live-match, recycled-pid mismatch, 21-vs-211 collision,
  empty pid); `frameCache` unchanged-skip (two identical ticks → one rename);
  `shouldClaimWorkspace`. These replicate every assertion the shell harness made
  via `PAIR_TITLE_TEST_CALL`.
- A `Runtime`-mock loop test: one tick renders the expected renames; a second
  identical tick emits none (unchanged-skip); session-miss threshold drives exit.
- The shim loses the `PAIR_TITLE_TEST_CALL` hook, so `tests/pair-title-poller-test.sh`
  can no longer unit-test the helpers. Replace `make test-pair-title` to run the
  Go package tests (`go test ./cmd/internal/titlepoller`); drop/retire the old
  shell harness (its coverage moves to Go, recorded in the Log).

### M1 verification

- `go test ./cmd/internal/titlepoller` green; full `make test` green.
- `bin/pair-title.sh <tag> <agent>` still spawns a working poller (shim →
  Go binary); the argv guard still recognizes a running poller (single-instance
  holds across a re-spawn).
- runtimebundle drift-check clean.

## M2 — scrollback/changelog openers (detailed)

Port the two Alt+/ and Alt+l floating-pane launchers — `bin/pair-scrollback-open`
(193 lines) and `bin/pair-changelog-open` (100 lines) — to Go. Both validate env,
hold a viewer re-entrancy lock, invoke the already-Go `pair scrollback-render` /
`pair changelog` subcommands (#92), and launch a **native** nvim viewer
(`nvim/scrollback.lua` / `nvim/changelog.lua` — NOT ported, #95 boundary) with the
same env contract. `nvim/*.kdl` unchanged.

### Structural decision — replace in place (no shim)

The only callers are `zellij/config.kdl`'s `Run "pair-scrollback-open"` /
`Run "pair-changelog-open"` (by PATH) and the changelog e2e tests (by path). A Go
binary of the **same name** owns the invocation identically — so there is nothing
to shim. `git rm` the two tracked shell scripts, drop their two `.gitignore`
negations (`!bin/pair-scrollback-open`, `!bin/pair-changelog-open`) so the built
binaries are ignored, and let `cmd/pair-scrollback-open` / `cmd/pair-changelog-open`
build to those paths. (Done-when: "shell name survives as a shim **or is removed
where no caller needs it**".) Verify `git ls-files bin/` after — the memory lesson
on `bin/` tracking fragility applies.

### New package `cmd/internal/opener`

- **`opener.go` — pure (direct unit tests):**
  - `matchViewport(dump, ansi []string) (line int, ok bool)` — the awk scroll
    -position scorer ported to Go: index `ansi` lines (len ≥ 8) → line numbers,
    for each non-blank `dump` line collect candidate starts, score each start by
    consecutive matches, accept the best iff ≥ 50% of non-blank dump lines match
    (clamp start ≥ 1). This is the load-bearing extraction — unit-test the
    high-confidence hit, the sub-threshold reject, and the top-of-buffer clamp.
  - `changelogBase(dataDir, tag, agent, sid string) string` and
    `resolveSessionID(envSID string, configJSON []byte) string` — the #63 per
    -session keying (env SID → config `session_id` → legacy unsuffixed).
  - scrollback/changelog path sets from the base; the detached-distiller argv.
- **`Runtime` seam:** `RenderScrollback(raw, events, ansi) error` (exec `pair
  scrollback-render`); `ListAgentPaneID() string` + `DumpScreen(paneID) (string,
  error)` (zellij IPC for the viewport overlay); `ReadFile`/`WriteAtomic`/`Stat`;
  `ProcessAlive(pid)` (procutil); `StartDistiller(argv, statusPath) (pid string,
  err error)` (the `setsid`-detached render+distill — Go `SysProcAttr{Setsid:true}`
  replaces the shell's setsid/perl fork); `RunViewer(luaPath, file, env)` (exec
  nvim as a held child, returns on `:q`); `Getpid()`.
- **`run.go`** — `RunScrollback(opts, rt)` / `RunChangelog(opts, rt)` orchestration
  mirroring the shell control flow (lock → render → viewport overlay → nvim for
  scrollback; lock → distiller-singleton → detached distill → nvim watcher for
  changelog). Best-effort viewport overlay (any seam failure leaves the renderer's
  `.viewport`).
- **`runcli.go`** — `RunScrollbackCLI(args, getenv, stderr) int` (parses `--jump
  prev|next`) and `RunChangelogCLI(args, getenv, stderr) int`.
- **`runtime.go`** — `OSRuntime` (zellij/nvim/pair exec + setsid detach + fs +
  procutil), reusing `procutil.Alive` for the lock liveness checks.

### Shims + wiring

- `cmd/pair-scrollback-open/main.go`, `cmd/pair-changelog-open/main.go` — thin
  `RunXCLI` entries.
- Makefile: add both to `GO_BINS`, `.PHONY`, per-binary targets, and
  `RUNTIMEBUNDLE_HELPERS`; explicit build rules. `runtimebundlegen`'s
  `explicitAssetPaths` already lists both paths (now built binaries, not scripts).

### Tests (M2)

- Go unit tests in `opener`: `matchViewport` (high-confidence hit / sub-threshold
  reject / top clamp / empty), `resolveSessionID` (env / config / legacy),
  `changelogBase`, distiller-argv construction.
- The existing `tests/changelog-open-test.sh` + `tests/changelog-session-key-test.sh`
  become **integration tests against the Go binary** unchanged (they invoke
  `bin/pair-changelog-open` by path with fake scrollback/model/nvim and assert the
  distilled log + anchor + nvim-open) — the process-level fake the #93 Spec wants.
  Add a parallel scrollback-open integration smoke (fake render + fake nvim →
  assert nvim opened on the `.ansi`, lock lifecycle) since none exists today.

### M2 verification

- `go test ./cmd/internal/opener` green; `make test` green incl. the changelog
  e2e now driving the Go binary; runtimebundle drift-check clean.
- Alt+/ (scrollback) and Alt+l (changelog) still open their nvim viewers; the
  re-entrancy locks + detached distiller behave as before.
- `git ls-files bin/` no longer lists the two openers; the built Go binaries are
  gitignored.

## M3 — review helpers (detailed)

Port the three review-start orchestrators — `bin/pair-review-target` (60),
`bin/pair-review-open` (54), `bin/pair-review-readiness` (123) — to Go, sharing a
`cmd/internal/reviewcmd` package. Same **replace-in-place** call model as M2:
zellij binds + `nvim/init.lua` invoke them by PATH name, so the Go binaries own
the names (no shim; drop the 3 `.gitignore` negations). `nvim/review/*.lua` stays
native (#95 boundary); in particular `nvim/review/readiness.lua`'s 4-case
`classify()` is the **single source** of the readiness decision (its own
`readiness_test.lua`), so the Go readiness helper keeps invoking it via
`nvim --headless` — it stays "the thin git-fact / git-effect shell" its own
comment describes.

### Two shared extractions folded in (ARCH-DRY)

- **`cmd/internal/osfs`** — the M2 reviewer's forward note. A `FS` struct with the
  string-based fs primitives (`ReadFile (string,error)`, `WriteFile`,
  `WriteAtomic`, `Remove`, `FileSize`, `ModTime`, `Touch`, `Executable`) that
  `opener`, `titlepoller`, and the new `reviewcmd` `OSRuntime`s **embed** (each
  package's `Runtime` interface still declares only the subset it uses; extra
  embedded methods are harmless). `sessionwatch` stays separate — its ReadFile is
  `[]byte`/error-based, a genuine divergence. Retrofit opener + titlepoller to
  embed `osfs.FS` (mechanical; the interfaces + fakes are untouched, existing
  tests catch regressions).
- **`cmd/internal/codexsid`** — `review-target`'s session stamping is `PAIR_SESSION_ID`
  → config `session_id` (both already covered by `transcript.SessionID`) → a
  codex-only `agent-pid` → ps-descendants → `lsof` → `rollout-…-<uuid>.jsonl` walk.
  That walk is a 3rd near-copy (slug + sessionwatch have it). Extract
  `codexsid.ResolveSessionID(dataDir, tag, home) string` as the canonical home and
  use it in review-target; note slug/sessionwatch can adopt it later (don't
  retrofit those tested hot-path packages in M3).

### New package `cmd/internal/reviewcmd`

- **Pure (direct unit tests):** `slugify(path) string` (basename → lowercase →
  non-alnum→`-` → collapse/trim — the review-branch slug), `absPath` normalization,
  the `review-target-<tag>.json` `{file,status,session}` shape, the readiness JSON
  `{case,is_git,is_tracked,branch,on_review_branch,scoped_file,file_matches,is_clean}`
  shape, and the `--prepare` action mapping (stop/track/resume/new/interact →
  git-effect plan).
- **`Runtime` seam** (embeds `osfs.FS`): a git seam
  `Git(dir string, args ...string) (out string, err error)` for the 11
  read/effect git commands (rev-parse/ls-files/branch/status/log/add/commit/
  checkout/show-ref); `Classify(facts) (string, error)` (the `nvim --headless`
  readiness bridge); `SpawnReviewPane(dir, lua, file string) error` (the
  `zellij run --floating … -- nvim -u review.lua` spawn); `ProcessAlive`/`Kill`
  (single-review-pane replacement); `ResolveCodexSessionID(dataDir, tag, home)`
  (via `codexsid`).
- **Three CLIs:** `RunTargetCLI` / `RunOpenCLI` / `RunReadinessCLI(args, getenv,
  stdout, stderr) int`, wiring the OSRuntime. Three thin `cmd/pair-review-*/main.go`.

### Tests (M3)

- Go unit tests in `reviewcmd`: `slugify`, the two JSON shapes, the
  readiness-facts → git-command mapping, the `--prepare` action plan per case
  (fake git seam asserts the add/commit/checkout sequence + the mark-ready write),
  single-pane replacement (fake `ProcessAlive`/`Kill`), and the open-path spawn
  argv. `codexsid` + `osfs` get their own focused unit tests.
- The existing `tests/pair-review-target-test.sh`, `review-readiness-cli-test.sh`,
  and `review-window-test.sh` become **integration tests against the Go binaries**
  unchanged (real git temp-repo + real nvim classify; faked zellij) — the
  process-level fakes the #93 Spec wants.

### M3 verification

- `go test ./cmd/internal/reviewcmd ./cmd/internal/osfs ./cmd/internal/codexsid`
  green; `make test` green incl. the whole `test-review` suite now driving the Go
  binaries; runtimebundle drift-check clean; opener + titlepoller tests still green
  after the `osfs` retrofit.
- Alt+c review-start (readiness → prepare → target) and `:PairReview` (open) flows
  work; `git ls-files bin/` no longer lists the three review helpers.

## M4 — clipboard helpers (detailed)

Port the copy-on-select pipeline — `bin/copy-on-select.sh` (110), the
`bin/clipboard-to-pane.sh` (101) hand-off, and `bin/flash-pane.sh` (57) — to Go,
sharing a `cmd/internal/clipcmd` package. These are thin zellij-IPC + OS-clipboard
glue; the pure logic is small (pane selection + in-nvim classification).

### Structural decision — SHIM pattern (not replace-in-place), the M1 model

Unlike M2/M3 (no-suffix names → replace-in-place), these have **`.sh`** names, and
`.gitignore` has `!bin/*.sh` (all `.sh` tracked), so a Go binary at `bin/*.sh`
would be committed. Also `tests/copy-on-select-test.sh` stubs
`$PAIR_HOME/bin/clipboard-to-pane.sh` **by path** and asserts copy-on-select execs
it. So: keep the 3 `.sh` names as tracked thin re-exec shims → 3 gitignored Go
binaries (`cmd/copy-on-select` / `cmd/clipboard-to-pane` / `cmd/flash-pane`), and
have the ported copy-on-select **still exec** `$PAIR_HOME/bin/{flash-pane,clipboard-to-pane}.sh`
(preserving the chain the test drives). Callers unchanged: zellij
`copy_command "copy-on-select.sh"`; copy-on-select is the only caller of the
other two.

### Extraction — `cmd/internal/zellijpane` (ARCH-DRY)

The `zellij action list-panes --json` recursive-descent walk is now needed 3× (the
existing `opener.firstAgentPaneID` + copy-on-select's focused-pane + clipboard-to-pane's
nvim-pane). Extract `zellijpane.Parse(jsonBytes) []Pane` (`Pane{ID, TerminalCommand,
IsFocused, IsPlugin, IsFloating}`, in recursive-descent order); callers filter by
predicate. Use it for both clip walks (2 consumers immediately); note
`opener.firstAgentPaneID` can adopt it later.

### New package `cmd/internal/clipcmd`

- **Pure (direct unit tests):** `isNvimCommand(cmd) bool` (`nvim|draft` regex on
  `terminal_command`, NOT title — the #copy-on-select-test bug); `quoteFile(dataDir,
  tag)` (`quote-<tag>`, tag = `PAIR_TAG || PAIR_AGENT || claude`); `focusedPane` /
  `nvimPane` selectors over `[]zellijpane.Pane`; flash defaults (`#50fa7b`/`100`).
- **`Runtime` seam** (embeds `osfs.FS`): `ClipboardCopy(text)` / `ClipboardPaste()
  (string,bool)` (pbcopy/pbpaste → wl-copy/wl-paste → xclip); `ListPanes() (string,
  error)` (`zellij action list-panes --json [--command]`); `SetPaneColor(id, bg)` +
  `ResetPaneColorAfter(id, d)` (the detached flash reset — setsid, like the
  changelog distiller, so it survives the caller exiting); `FocusPane(id)`;
  `WriteKey(byte)` (`zellij action write 31`); `Exec(path, args…)` (copy-on-select →
  flash/clipboard `.sh` chain).
- **Three CLIs:** `RunCopyOnSelect(stdin, getenv, stderr)`, `RunClipboardToPane(
  getenv, stderr)`, `RunFlashPane(args, getenv, stderr)` + thin `cmd/*/main.go`.

### Tests (M4)

- Go unit tests: `isNvimCommand`, `quoteFile`, `zellijpane.Parse` + the pane
  selectors (focused non-plugin/non-floating; nvim by terminal_command — pin the
  parley.nvim-cwd false-positive the shell test guards); a fake-`Runtime` loop
  test for each Run (copy-on-select in-nvim→skip vs not→flash+handoff;
  clipboard-to-pane stage quote + focus + Ctrl-_; flash set+reset).
- Keep `tests/copy-on-select-test.sh` driving the Go binary unchanged (it stubs
  the `.sh` handoff path — preserved by the exec chain). Add a `$(BIN_DIR)` prereq
  to `test-copy-on-select` (the M2/M3 missing-prereq lesson).

### M4 verification

- `go test ./cmd/internal/clipcmd ./cmd/internal/zellijpane` green; `make test`
  green incl. `test-copy-on-select` driving the Go binary; drift-check clean;
  `git ls-files bin/` still lists the 3 `.sh` shims (now thin), not the Go binaries.

## M5 — launcher / session lifecycle (detailed)

Port `bin/pair-shell` (2287 lines) — the last and largest surface — onto the
`cmd/internal/launcher` core, retaining a compatibility shim. zellij/nvim stay
external (#95 boundary). **Recommendation: extract this into its own ticket**
(see "Structure" below); the detail here is the design regardless of wrapper.

### What already exists vs the gap (survey, 2026-07-02)

`cmd/internal/launcher` (from #75) already implements the **entire decision
phase**, and it is well unit-tested — but it is a *prototype currently bypassed*:
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

### Core architectural move (ARCH-DRY, ARCH-PURE)

Build one native orchestration entry `launcher.RunLaunch(...)` on top of the
existing pure core, behind a **new `launcher.Runtime` effect seam** (the M1–M4
`OSRuntime`+`osfs.FS` pattern — the launcher today has only the two narrow
`SessionSource`/`HistoricalScanner` sources, not a unified effect seam). The seam
covers: zellij exec/query (`zj` timeout wrapper + blocking attach/new-session),
fzf/prompt UIs, marker read/write, cmux, config read/write (jq → `encoding/json`),
nvim reap, child-spawns, tty, env. Pure decisions stay pure and unit-tested;
`RunLaunch` drives decision → effects → blocking handoff → post-handoff
cleanup/restart, and is exercised by a fake-`Runtime`.

### Compatibility shim strategy

End-state: the Go `pair` binary runs the launcher **in-process** (no exec to
`bin/pair-shell`); `bin/pair-shell` becomes a thin shim → `pair-go launch` for any
residual external caller; the restart re-exec becomes an in-process loop. During
transition, keep the existing `entrypoint.ResolveLegacyLaunch` + `legacyRuntime.Exec`
path (cmd/pair-go/main.go — the effect seam to widen) as a **flag-gated fallback**
(`PAIR_NATIVE_LAUNCH`), so `bin/pair-shell` remains the default until native
parity is proven, then the default flips and the shell path is retired.

### Phased plan (each independently mergeable, M1–M4 template)

- **L1 — pure-logic completion (no wiring, zero behavior change).** Port the
  remaining pure pieces into `launcher`: full `ParseArgs` (`continue`/`rename`/
  `list`), resume-token strip/compose (4 duplicated shell loops → one helper —
  ARCH-DRY), config-migration decision rules, per-agent launch-arg composition
  (claude session-id shape, codex alt-screen idempotence), `rename` plan-build
  (`rename_paths_for` enumeration + transform), title/`format_age`/`age_color`
  formatting. Unit-tested directly.
- **L2 — Runtime seam + native orchestration.** Define `launcher.Runtime`; build
  `RunLaunch` for the full flow. Fake-`Runtime` loop tests: create, attach,
  picker→attach/create, name-prompt, tag-restart config picker, restart-marker
  re-entry, in-session compaction, quit cleanup.
- **L3 — cutover.** Flip `cmd/pair-go` to run the native launcher in-process
  under `PAIR_NATIVE_LAUNCH`; convert `bin/pair-shell` to a thin shim →
  `pair-go launch`; restart re-exec → in-process loop. Full e2e vs the shell
  (create/attach/restart/quit/compaction), then flip the default.
- **L4 — subcommands + retirement.** Port `list`/`rename`/`continue`; retire the
  shell fallback + `bin/pair-restart.sh` markers → in-process; drop the flag.
  This is what lets #94 (stop extracting a shell tree) proceed.

### Tests

Follow the M1–M4 convention exactly: pure decisions unit-tested directly; the
orchestration driven by a fake `Runtime`; the concrete `OSRuntime` sources tested
against on-disk/exec fixtures (the established `ZellijSource` bash-stub +
`HistorySource` sidecar-file pattern). Keep the existing `PAIR_TEST_CALL` /
`PAIR_DEBUG_*` shell contract tests green against whichever launcher is active per
phase; add Go coverage for every gap-set behavior before retiring its shell.

### Verification

Per phase: `go test ./cmd/internal/launcher …` green; the launcher shell tests
(`tests/*launch*`, `PAIR_TEST_CALL` seams) green; a real create + attach +
restart + quit + compaction exercised end-to-end (this is a lifecycle port —
process-level fakes miss interaction bugs, so drive the real flow); drift-check
clean; `git ls-files bin/` shows `bin/pair-shell` as a thin shim by L3.

### Structure — recommend its own ticket (deps #93)

M5 is categorically larger than the M1–M4 leaves (~900 lines of new IO
orchestration + a new effect seam + the trickiest lifecycle logic in the tree,
P0/load-bearing). An honest re-estimate is **~15–22h across L1–L4**, not the 6.0h
placeholder. Extracting it to its own ticket gives it an honest estimate and
isolated actuals, and keeps #93's M1–M4 actuals clean. #93 stays open (its
Done-when includes a Go owner for the launcher) until that ticket lands. If instead
kept as #93's M5, this same design applies — only the wrapper differs. **Awaiting
the operator's call on ticket-vs-milestone before `sdlc issue new` + the durable
plan move; implementation waits for plan approval regardless (2287-line P0 port).**

## Atlas (per-milestone)

Update `atlas/go-migration-inventory.md` (the ported binary's contract row +
Coverage Ledger) and `atlas/architecture.md` where a surface/flow/pointer
changes, at each milestone close — not deferred to the end.

## Revisions

### 2026-07-01 — change-code plan-quality suggestions folded in (INFO verdict)

- **ARCH-DRY: shared `procutil` seam (suggestion #1).** Rather than copy the
  `kill -0` / `ps -p <pid> -o command=` idioms into each `OSRuntime`, extract a
  small `cmd/internal/procutil` package (`Alive(pid) bool`, `Command(pid) string`)
  now — two consumers (sessionwatch + titlepoller) is where DRY starts paying,
  and M2–M5 each add another runtime needing the same primitives. M1 creates it,
  uses it in `titlepoller`, and retrofits `sessionwatch.OSRuntime.ProcessAlive`
  to call `procutil.Alive` (low-risk 1-line change to a tested package).
- **ARCH-PURPOSE: nvim/zellij audit discharge (suggestion #2).** The shim
  strategy preserves every command name (`bin/pair-<name>.sh` still resolves),
  so the "repoint nvim/zellij shell-outs to Go owners" Done-when is a **no-op by
  construction** — callers keep invoking the same names, now backed by Go. The
  audit is discharged per milestone-close by confirming no `.lua`/`.kdl` shell-out
  names changed; recorded in each milestone's Log.
- **M5 (suggestion #3):** when reached, run `sdlc start-plan` and elaborate M5 as
  its own design pass (very likely its own ticket) — not a scaled-up leaf port.

### 2026-07-01 — M1 milestone-close review (FIX-THEN-SHIP) follow-ups

- **No `Log`/adapt seam.** The M1 sketch listed a `Runtime.Log(...)` adapt method
  and a runcli "open the adapt logger" step. Dropped as the *faithful* choice:
  `bin/pair-title.sh` never emitted adapt events (unlike wrap/slug/session-watch),
  so the poller has no adapt telemetry surface. The implemented `Runtime` has no
  `Log`.
- **`latest(sources)` → `activityMTime(opts, rt)`.** The pure-helper name in the
  sketch became a seam function, since the max-mtime read is IO (`rt.ModTime`).
  Its pure selection is exercised by `TestActivityMTimePicksLatest`.
- **Loop-body integration test added (the Important finding).** The initial M1
  tested `updateFrameTitles` directly but not the loop wiring. Added
  `TestRunRendersFrameAndCmuxTitles` (claim path: one tick renders frame + cmux),
  `TestRunDefersCmuxToLiveForeignOwner` (defer path), plus direct
  `updateWorkspaceTitle` reclaim/unchanged-bucket tests — closing the promised
  Runtime-mock loop coverage. Carry a loop-body integration test as a first-class
  deliverable for M2–M5.

### 2026-07-01 — M2 milestone-close review (FIX-THEN-SHIP) follow-ups

- **Seam names differ from the M2 sketch (shipped surface):** `ListAgentPaneID`
  → `AgentPaneID`; `StartDistiller(argv, statusPath)` → `StartDetached(script,
  extraEnv, statusPath)` (the detached build is one `sh -c` string + PCL_* env,
  not a pre-split argv — mirrors the shell). `Stat` → `FileSize`; added
  `Executable` (the shell's `[ -x $PAIR_HOME/bin/pair ]` guard) and `Touch`.
- **`.viewport` write IS atomic.** The sketch said `WriteAtomic`; the first cut
  used a plain `WriteFile` (review Minor). Restored to a real temp+rename
  (`WriteAtomic`) matching the shell's `> .tmp && mv -f`, since a live viewer's
  `G` refresh may re-read `.viewport` concurrently. `WriteFile` (non-atomic)
  stays for the locks (single-writer, no concurrent reader).
- **`test-changelog` gained the `$(BIN_DIR)/pair` prereq** (review Important): the
  changelog e2e SKIPs without a built `bin/pair`, so the detached-distiller path
  was only covered incidentally via a sibling target's build order. Now explicit.
- **Faithful UX restored:** the two error paths (missing-env, no-scrollback) got
  their second explanatory line back.
- **`firstAgentPaneID` map-iteration order** is Go-random vs jq document order —
  moot under the two-pane invariant (exactly one candidate); documented in place
  rather than restructured.
- **Forward (M3–M5):** the reviewer flagged `OSRuntime` fs-primitive duplication
  trending across `opener`/`titlepoller`/`sessionwatch`. Consider a shared
  `osfs`/`osseam` the per-package `OSRuntime`s embed before M3/M4/M5 add a 4th–6th
  copy (keep the domain methods per-package). **[done in M3 — `cmd/internal/osfs`.]**

### 2026-07-01 — M3 milestone-close review (FIX-THEN-SHIP) follow-ups

- **`codexsid.ResolveSessionID(dataDir, tag)` shipped without the sketched
  `home` param** — it was unused (the walk reads `$dataDir/agent-pid-<tag>` and
  greps lsof paths, which already carry `~/.codex/...`). Correct as shipped.
- **Pure-list re-categorization:** the sketch listed `absPath normalization` and
  the `--prepare` action mapping under "pure (direct unit tests)". In the shipped
  code path-resolution is on the IO seam (`AbsFile`/`LogicalDir`/`PhysicalDir` do
  `stat`/`EvalSymlinks`) and `prepare()` is seam-orchestration (its branches
  interleave with `show-ref`/`ls-files`/`status` results) — both tested via the
  fake `Runtime`, not as standalone pure functions.
- **`--prepare` `track` + `resume` coverage added** (review Important): the initial
  cut only tested `new`. Added `TestRunReadinessPrepareTrack` (asserts the
  add→commit→ls-files→status→checkout-b sequence + mark-ready) and
  `TestRunReadinessPrepareResume` (asserts branch kept, no checkout).
- **`test-review` gained the 3 review-binary prereqs** (review Important): they're
  now built Go binaries, so a fresh-tree `make test` must build them first.
- **Target JSON write made atomic** (review Minor): `WriteAtomic` (temp+rename)
  since nvim's Alt+c re-reads `review-target-<tag>.json`. Strengthened
  `TestRunReadinessJSON` to assert `scoped_file`/`file_matches`.
- **Forward:** the codex walk is now triplicated (`codexsid` + `slug` +
  `sessionwatch`); M3 added the canonical `codexsid` + wired review-target to it —
  `slug`/`sessionwatch` adoption to collapse the triplication is a tracked
  follow-up (not retrofitted in M3 to avoid touching those hot-path tested packages).

### 2026-07-01 — M4 shipped surface + milestone-close review (FIX-THEN-SHIP) follow-ups

- **`Exec(path, args…)` split into two named seams (shipped surface).** The M4
  sketch (this doc's M4 section) listed a single `Exec`. The shipped
  `clipcmd.Runtime` has two, because the shell chain has two distinct behaviors:
  `RunSubprocess(path, args…)` (call-and-return — flash-pane, whose bg reset is
  setsid-detached so it must not block the focus change) and `ExecReplace(path,
  args…)` (process replace via `syscall.Exec` — the terminal clipboard-to-pane
  hand-off, the shell's `exec`). Folds in the change-code plan-quality note #2.
- **`clipcmd.Runtime` embeds `osfs.FS`** (declares only the `WriteFile` +
  `Executable` subset it uses), like opener/reviewcmd — not a bespoke fs seam.
- **clipboard-debug log truncates at the pipeline head** (review Minor). The
  source truncated (`> "$LOG"`) inside `clipboard-to-pane.sh` — mid-chain, which
  clobbered copy-on-select's own lines and left the log unbounded across
  standalone runs. The Go port truncates once at the copy-on-select entry
  (`LogFresh`) and appends thereafter, so the diagnostic holds exactly one
  selection's chain (a deliberate improvement over the source, not a faithful copy).
- **Faithful two-regex in-nvim distinction preserved:** copy-on-select's in_nvim
  gate is `(?i)nvim|draft` on the focused pane's `terminal_command`; clipboard-to
  -pane's draft finder is case-sensitive `nvim` (jq `test("nvim")`). Kept as two
  separate checks — matching the two source scripts — not unified.
- **Forward:** `opener.firstAgentPaneID` (cmd/internal/opener/runtime.go) still
  open-codes the list-panes walk that `zellijpane.Parse` now owns; the `Pane`
  struct already carries `Title` so opener's title-keyed pick is a pure swap. Not
  retrofitted in M4 (conservative — avoids touching tested opener); tracked to
  ride the next milestone that touches opener.
