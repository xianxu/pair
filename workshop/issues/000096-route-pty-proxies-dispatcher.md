---
id: 000096
status: working
deps: [000092]
github_issue:
created: 2026-07-01
updated: 2026-07-01
estimate_hours: 3.25
started: 2026-07-01T13:38:04-07:00
---

# route pair-wrap and pair-scribe through dispatcher

Tracking: #91 (native single binary) — carved out of step 2 (#92). Depends on #92.

## Problem

`pair-wrap` (the PTY proxy that wraps every agent turn) and `pair-scribe` (a PTY
logging wrapper) are the two remaining Pair-owned helper binaries not routed
through the Go dispatcher. Both are **interactive PTY proxies** — they take over
real stdio, set raw terminal mode, handle signals (SIGWINCH), and run for the
life of a session — so they do NOT fit the buffered `Dispatch(args) Result`
path that #76 used for `context`/`scrollback-render`. They were carved out of
#92 (which routes the finite internal-call helpers) because they are session
*entrypoints*, not internal calls Pair makes, and because pair-wrap wraps every
turn: a regression there breaks all sessions, so it deserves its own review
boundary and focused verification.

Note: `pair-scribe` also appears **orphaned** — built as a binary but not in the
runtime bundle manifest and with no Pair-owned caller found in the tree — so this
issue must first decide whether to route it for surface consistency or retire it.

## Spec

Both binaries are **already Go**, so this is repackaging, not a rewrite — no
logic changes. Reuse #92's streaming dispatch seam (`ARCH-DRY`).

For `pair-wrap`:

- Extract `cmd/pair-wrap/main.go`'s body into a reusable
  `cmd/internal/wrapcmd` package with a `Run(args []string, stdin io.Reader,
  stdout, stderr io.Writer) int` entry (the #76 runner shape, extended for real
  stdio), threading every `os.Exit`/`log.Fatal` exit path into clean return
  codes.
- Reduce `cmd/pair-wrap/main.go` to a thin shim that calls `wrapcmd.Run` with
  real `os.Stdin/os.Stdout/os.Stderr`.
- Add a `wrap` route on #92's streaming dispatch seam (real stdio, not buffered).
- The turn-end `pair-slug` spawn inside pair-wrap is repointed by #92; this issue
  must not regress it.

For `pair-scribe`: confirm whether any caller (Pair-owned or external agent
config) invokes it. If genuinely dead, retire it (drop the binary + build rule)
rather than adding an unused `pair scribe` surface (Simplicity-First). If a
caller exists, repackage it the same way as pair-wrap.

Behavior must be **byte-for-byte preserved**: PTY raw-mode handling, window-size
(SIGWINCH) propagation, signal forwarding, child-process spawn, and exit codes
are identical before/after. This is the load-bearing verification, not the
extraction.

Merge-safe: `pair-wrap` continues to wrap agent turns identically; a real
session (agent start → turns → slug refresh → exit) works after the change.

Architecture: `ARCH-DRY` (one implementation behind the standalone shim and the
dispatcher route; reuse #92's streaming seam), `ARCH-PURPOSE` (routing = every
remaining helper reachable as `pair <subcommand>`).

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.3 impl=1.0
item: smaller-go-module design=0.15 impl=0.5
item: skill-or-dispatcher design=0.15 impl=0.35
item: atlas-docs design=0.1 impl=0.15
item: milestone-review design=0.0 impl=0.25
design-buffer: 0.3
total: 3.25
```

Durable plan: `workshop/plans/000096-route-pty-proxies-plan.md`. The first
`smaller-go-module` item is the load-bearing one — extracting the 71KB
`pair-wrap` into `cmd/internal/wrapcmd` behind a thin shim, threading real stdio
through the `proxy`, and proving byte-for-byte PTY parity. The second is the
parallel (smaller) `pair-scribe` → `scribecmd` extraction. `skill-or-dispatcher`
covers both streaming routes + Makefile. Estimate raised from 2.5→3.25 after the
plan-quality judge: scribe is **routed, not retired** (it's live user shell
tooling per the atlas — see Log), which is more work than deleting it.

## Done when

- [x] `pair wrap` runs the PTY proxy via the dispatcher's streaming route with
      real stdio; `cmd/pair-wrap/main.go` is a thin shim over `wrapcmd.Run`.
- [x] pair-scribe is routed (`pair scribe` + thin shim); no unused dispatcher
      surface added (both wrap + scribe are real implemented streaming routes).
- [x] No logic changed: PTY raw-mode, SIGWINCH, signal forwarding, child spawn,
      and exit codes verified identical (route-equivalence tests compare the
      dispatch route to the standalone shim byte-for-byte; PTY child-exit tests
      confirm code propagation; all 14 moved wrap tests still pass unchanged).
- [x] A real wrapped session works end-to-end: PTY-level tests spawn a child in
      a real pty and propagate its exit code through `Run`; full `make test`
      green (wrapcmd 2.9s, scribecmd 2.6s with real ptys, pair-go routes).
- [x] Runtime bundle manifest / build rules updated: pair-wrap/pair-scribe build
      rules repoint to the internal packages, `PAIR_GO_SRCS` gains both, and the
      (gitignored) manifest regenerates with the new shim digest (drift-check clean).

## Plan

- [x] Extract `cmd/internal/wrapcmd` `Run(...) int` from `cmd/pair-wrap/main.go`;
      thread `os.Exit`/`log.Fatal` → return codes; keep signal/PTY code intact.
- [x] Reduce `cmd/pair-wrap/main.go` to a thin shim over `wrapcmd.Run`.
- [x] Add the `wrap` route to #92's streaming dispatch seam.
- [x] pair-scribe → ROUTE (decided): extract `cmd/internal/scribecmd`, add the
      `pair scribe` streaming route, keep `cmd/pair-scribe` as a thin shim (the
      `~/.local/bin/pair-scribe` install + `~/.zshrc` wiring stay intact).
- [x] Tests: exit-code parity, arg passthrough, and a PTY-level behavior check
      (both wrapcmd + scribecmd); route tests for `pair wrap`/`pair scribe`; run
      the session/wrap integration suite.

## Log

### 2026-07-01

Carved out of #92. pair-wrap/pair-scribe are interactive PTY entrypoints, not
internal finite calls, so they need the streaming dispatch route rather than the
buffered `Dispatch(args) Result` path — and pair-wrap wraps every turn, warranting
its own review boundary + focused PTY-behavior verification. Both are already Go,
so the work is mechanical repackaging (extract `Run()`, thin shim, streaming
route) with no logic change; the effort is in exit-path threading and
behavior-parity verification, not redesign. pair-scribe may be dead code —
route-or-retire decision folded in.

**pair-scribe decision → ROUTE (not retire).** The plan-quality judge blocked an
initial "retire — orphaned" call: an in-tree grep can't see the user's
`~/.zshrc`, but `atlas/architecture.md:44-46,730-732` + `cmd/pair-scribe/README.md`
document scribe as live user shell tooling (a deliberate `script(1)` replacement
installed to `~/.local/bin/pair-scribe` by `make install`, wired into shell
startup), and the binary is in fact installed. The atlas already lists `pair
scribe` as a target dispatch subcommand. User confirmed: route it, mirroring
pair-wrap — extract `cmd/internal/scribecmd`, add a `pair scribe` streaming
route, keep `cmd/pair-scribe` as a thin shim so `~/.local/bin/pair-scribe` and
the user's `~/.zshrc` `exec` line are untouched (non-destructive; scribe stays
out of the runtime bundle — it's shell tooling, not runtime). Estimate raised
2.5→3.25 accordingly. Durable plan: `workshop/plans/000096-route-pty-proxies-plan.md`.

**Implemented.** `git mv`'d the 71KB `cmd/pair-wrap/main.go` + 14 test files →
`cmd/internal/wrapcmd/` (`package main`→`wrapcmd`); threaded real stdio through
the `proxy` (new `stdin/stdinFile/stdout/stdoutFile/stderr` fields, ~10 `os.Std*`
sites repointed) with an `io.Reader`→`*os.File` type-assert for the raw-mode /
winsize ops that genuinely need `*os.File` (production always passes the real
files → byte-for-byte identical; a non-file test reader degrades to the
`isTTY`-guarded no-op). The mid-flight `os.Exit(childExit)` became `run() (int,
error)` returning the code. Same treatment for `cmd/pair-scribe/main.go` →
`cmd/internal/scribecmd/scribecmd.go`, switching the global `flag` to a local
`flag.NewFlagSet` so `Run` is re-callable in tests. Flipped both dispatcher
families `planned`→`implemented`; added `wrap`/`scribe` cases to
`runStreamingSubcommand`. Both `cmd/pair-*` dirs are now thin shims. New tests:
`wrapcmd`/`scribecmd` `Run` arg/usage-error + PTY child-exit parity (real pty
via `creack/pty`, self-skips where `/dev/ptmx` is denied), and `cmd/pair-go`
route-equivalence tests proving `pair wrap`/`pair scribe` match the standalone
shims byte-for-byte. Updated `dispatcher_test`/`main_test` for the status flip
(no `planned` families remain; the streaming-guard is asserted instead).
Makefile build-rule prereqs + `PAIR_GO_SRCS` updated; runtime bundle manifest
(gitignored) regenerates with the new pair-wrap shim digest — `drift-check`
clean. Atlas: `go-migration-inventory.md` contract rows + Coverage Ledger + prose,
`architecture.md` (streaming-seam list, `pair-scribe` adjacent section, the
`sendKeymapByAgent` file pointer), the how-to `file://` links, and the scribe
README all repointed to the internal packages. Full `make test` green
(sandbox-off so the PTY tests ran for real).
