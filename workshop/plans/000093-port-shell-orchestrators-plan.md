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

## M2–M5 (milestone-level; detailed when reached)

- **M2 — scrollback/changelog openers:** port `bin/pair-scrollback-open` (+ the
  changelog opener) orchestration to Go; `nvim/*.lua` viewers stay native. Reuse
  `scrollbackcmd`/`changelogcmd`.
- **M3 — review helpers:** port `bin/pair-review-target` / `pair-review-open` /
  `pair-review-readiness` orchestration to Go.
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
