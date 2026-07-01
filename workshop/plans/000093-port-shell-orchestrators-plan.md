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

## M4–M5 (milestone-level; detailed when reached)

- **M4 — clipboard helpers:** port `clipboard-to-pane.sh`, `copy-on-select.sh`,
  `flash-pane.sh` (or fold behind the dispatcher).
- **M5 — launcher / session lifecycle:** port `bin/pair-shell`'s orchestration
  onto the `cmd/internal/launcher` core, retaining a shim; zellij/nvim stay
  external. Largest surface — may split into its own ticket if scope grows
  (per the issue's granularity note).

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
  copy (keep the domain methods per-package).
