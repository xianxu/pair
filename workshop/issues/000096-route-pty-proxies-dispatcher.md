---
id: 000096
status: open
deps: [000092]
github_issue:
created: 2026-07-01
updated: 2026-07-01
estimate_hours:
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

## Done when

- [ ] `pair wrap` runs the PTY proxy via the dispatcher's streaming route with
      real stdio; `cmd/pair-wrap/main.go` is a thin shim over `wrapcmd.Run`.
- [ ] pair-scribe is either routed (`pair scribe` + thin shim) or retired with a
      recorded rationale; no unused dispatcher surface is added.
- [ ] No logic changed: PTY raw-mode, SIGWINCH, signal forwarding, child spawn,
      and exit codes are verified identical to the pre-change binaries.
- [ ] A real wrapped session works end-to-end (start → turns → slug refresh →
      exit) after the change.
- [ ] Runtime bundle manifest / build rules updated for any shim/removal.

## Plan

- [ ] Extract `cmd/internal/wrapcmd` `Run(...) int` from `cmd/pair-wrap/main.go`;
      thread `os.Exit`/`log.Fatal` → return codes; keep signal/PTY code intact.
- [ ] Reduce `cmd/pair-wrap/main.go` to a thin shim over `wrapcmd.Run`.
- [ ] Add the `wrap` route to #92's streaming dispatch seam.
- [ ] Decide pair-scribe: confirm callers → route-or-retire; act accordingly.
- [ ] Tests: exit-code parity, arg passthrough, and a PTY-level behavior check;
      run the session/wrap integration suite.

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
