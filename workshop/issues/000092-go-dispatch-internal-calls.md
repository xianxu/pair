---
id: 000092
status: working
deps: [000090]
github_issue:
created: 2026-07-01
updated: 2026-07-01
estimate_hours: 6.52
started: 2026-07-01T09:40:38-07:00
---

# route internal calls through Go dispatcher

Tracking: #91 (native single binary) — step 2 of 5. Depends on #90.

## Problem

The runtime tree still ships several standalone Go helper binaries
(`pair-wrap`, `pair-slug`, `pair-changelog`, `pair-continuation`, `pair-context`,
`pair-scribe`, `pair-scrollback-render`, `pair-session-watch`) that generated
internal call-sites (nvim Lua, shell hooks, `pair-wrap` turn-end) invoke by
name. #76 proved the pattern — `pair-go context` and `pair-go scrollback-render`
dispatch through a shared internal runner while the legacy binary names stay
live — but it stopped there (`pair slug` was explicitly left as a later
candidate). As long as every helper is its own binary that callers hardcode,
the runtime bundle can't shrink toward a single executable.

## Spec

Fold the remaining Pair-owned **internal-call** helper binaries behind the Go
dispatcher so the canonical invocation is `pair <subcommand>` — `pair slug`,
`pair changelog`, `pair continuation`, `pair session-watch` — reusing the
internal runner packages rather than duplicating logic (`ARCH-DRY`). `context`
and `scrollback-render` are already routed (#76). The two interactive PTY
proxies (`pair-wrap`, `pair-scribe`) are **carved out to #96** (see Revisions):
they are session entrypoints, not internal calls, and need the streaming route.

- Two dispatch styles already exist: the **buffered** `Dispatch(args) → Result`
  path (`context`, `scrollback-render`) and the **streaming** process handoff
  (`launch`, via `syscall.Exec` with real stdio). `slug` and `changelog` are
  finite, no-stdin → buffered path. `continuation` (reads stdin) and
  `session-watch` (long-running) need a small **streaming dispatch seam**, added
  here and reused by #96 (`ARCH-DRY`).
- Keep the standalone `bin/pair-<name>` binaries as **thin shims** that call the
  shared internal runner (the `cmd/pair-context/main.go` shape — `os.Exit(
  <name>cmd.Run(...))`). Do not maintain two divergent implementations.
- Migrate Pair-owned generated call-sites (nvim Lua, shell orchestrators,
  `pair-wrap`'s turn-end `pair-slug` spawn) to the dispatcher form.
- Preserve every helper's current CLI contract (flags, env, stdin/exit codes) so
  callers and tests are unaffected; changes are invocation-path only.
- Scope is dispatch routing + shims **only**. Porting stateful shell
  orchestrators to Go is #93; this issue must not change shell lifecycle
  behavior.

Merge-safe: after this lands, `pair`, `pair-dev`, keybindings, scrollback,
changelog, continuation, and review flows all still work; Pair remains usable.

Architecture: `ARCH-DRY` (one implementation behind dispatch, shims not forks),
`ARCH-PURE` (dispatch/arg parsing stays unit-testable).

## Done when

- [ ] Each remaining Pair-owned helper is invocable as `pair <subcommand>`
      through the dispatcher, reusing its existing internal runner package.
- [ ] Legacy `bin/pair-<name>` binaries are thin shims (re-exec) or removed where
      no external caller needs the name; no duplicated helper logic remains.
- [ ] Generated Pair-owned call-sites use the dispatcher form.
- [ ] Every helper's CLI contract (flags/env/stdin/exit) is unchanged, with
      tests covering the dispatch path and unsupported-subcommand errors.
- [ ] Pair remains usable after merge through the existing `pair` entrypoint.

## Plan

Two merge-safe review boundaries. Detailed steps:
`workshop/plans/000092-go-dispatch-internal-calls-plan.md`.

- [x] M1 — dispatcher reachability + runner consolidation (backward-compatible):
      `Families()` routing metadata + `session-watch` entry; entrypoint peel-off
      so `pair <sub>` dispatches; extract `slugcmd`/`changelogcmd`/`continuationcmd`
      runners + thin shims; buffered route (`slug`) + streaming seam
      (`changelog`/`continuation`/`session-watch`); dispatch/classification/parity
      tests.
- [x] M2 — repoint Pair-owned call-sites (shadow-sweep): `pair-title.sh`,
      `pair-changelog-open`, `pair-scrollback-open`, `nvim/scrollback.lua`,
      `pair-wrap` turn-end spawn → `pair <sub>`; regenerate the runtime bundle;
      full-suite verification.

## Estimate

Produced via `estimate-logic-v3.1` primitives (Method A), same lineage as #90.
`sdlc estimate-source` reports the calibration source as stale, so the number is
provisional but uses the required method. The four `smaller-go-module` items are:
three runner extractions (`slugcmd`, `changelogcmd`, `continuationcmd` — refactors
of existing Go: move code + `Run()` wrapper + test relocation) plus the net-new
dispatcher plumbing (`Families()` routing metadata, the `ClassifyInvocation`
peel-off, and the `runStreamingSubcommand` streaming seam), which is genuinely new
abstraction rather than relocation. `session-watch`'s runner already exists, so it
adds only a route (folded into the plumbing item).

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.20 impl=0.08
item: smaller-go-module design=0.35 impl=0.48
item: smaller-go-module design=0.35 impl=0.48
item: smaller-go-module design=0.35 impl=0.48
item: smaller-go-module design=0.35 impl=0.48
item: cross-cutting-refactor design=0.80 impl=1.12
item: atlas-docs design=0.25 impl=0.20
item: milestone-review design=0.00 impl=0.20
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 6.52
```

## Log

### 2026-07-01
- 2026-07-01: closed M2 — M2: repointed all 5 Pair-owned call-sites to pair <sub> (pair-title.sh→pair context; pair-changelog-open→pair scrollback-render/changelog via collapsed PCL_BIN; pair-scrollback-open→pair scrollback-render; nvim/scrollback.lua→pair scrollback-render; pair-wrap turn-end→pair slug via testable slugSpawnCmd). Full `make test` passes (all shell suites incl. pair-title/changelog-open/session-watch/embedded-runtime + go test ./...). Shadow-sweep: every remaining pair-<name> ref intentionally retained (session-watch chain→#93, bundle manifest+shim binaries, adapt logger names, runner usage strings, equivalence tests). Runtime bundle is gitignored+regenerated on make build (plan I3 revised; no commit needed).; review verdict: FIX-THEN-SHIP
- 2026-07-01: closed M1 — M1: go test ./... all pass (env-scrubbed); make build produces pair + shims; route equivalence confirmed (pair slug≡pair-slug exit 0; pair changelog≡pair-changelog identical usage+exit 1; pair continuation exit 1); ClassifyInvocation grammar unit-tested (pair slug→dispatch, pair claude/resume/bare→launcher); streaming-seam tests (changelog live-stderr, continuation stdin passthrough, session-watch no-op); dispatcher DispatchNames/IsStreaming/IsImplemented tests. Callers still on shim names until M2.; review verdict: FIX-THEN-SHIP

Created as step 2 of the native-single-binary tracker (#91). Continues the
helper-dispatch pattern #76 began (`pair-go context`, `pair-go
scrollback-render`); `pair slug` and the remaining helpers are the concrete
targets.

**M2 shadow-sweep (ARCH-PURPOSE).** Repointed all five Pair-owned call-sites to
`pair <sub>`: `bin/pair-title.sh` (`pair context`), `bin/pair-changelog-open`
(`pair scrollback-render`/`changelog`, collapsing the two-token `PCL_BIN`),
`bin/pair-scrollback-open` (`pair scrollback-render`), `nvim/scrollback.lua`
(`pair scrollback-render`), `cmd/pair-wrap` turn-end (`pair slug`, via a
testable `slugSpawnCmd`). `grep -rnE 'pair-(slug|changelog|continuation|context|scrollback-render|session-watch)'`
confirms every remaining hit is intentionally retained: the
`bin/pair-shell → pair-session-watch.sh → binary` chain (shell-owned, #93),
the runtime-bundle manifest/generator + shim binaries (removal is later
single-binary work), adapt logger channel names (`"pair-slug"`), runner usage
strings, `nvim/init.lua`'s continuation-writer prose, and the equivalence
tests. `pair session-watch` and `pair continuation` have routes but no
repointed *production* caller here (session-watch's caller is the shell
launcher → #93; continuation is invoked by agent procedure) — intentional
symmetry for #93/#96 reuse, not dead code.

**Runtime bundle (revises plan I3).** The plan expected to regenerate + commit
the bundle. In this repo the entire `cmd/internal/runtimebundle/assets/` tree
(manifest + `files/`) is **gitignored** — regenerated from source on every
`make build` and `//go:embed`-ed — so there is nothing to commit and no
stale-bundle / dirty-tree-at-close risk. Editing the bundled shell/lua sources
is sufficient; the embedded runtime rebuilds from them. Verified: the generated
`assets/.../bin/pair-title.sh` carries `pair context`.

## Revisions

### 2026-07-01 — narrow scope: carve pair-wrap + pair-scribe out to #96

Design mapping (via an explore of the dispatcher + each helper) showed the
remaining helpers split by dispatch style, not size:

- `slug`, `changelog` — finite, no-stdin → existing **buffered** dispatch path.
- `continuation` (stdin), `session-watch` (long-running, runner already
  extracted) → need a small **streaming dispatch seam**, added by this issue.
- `pair-wrap` (PTY proxy wrapping every agent turn) and `pair-scribe` (PTY
  logging wrapper, apparently orphaned) → interactive PTY **entrypoints**, not
  internal calls. They need the streaming route and, for pair-wrap, byte-for-byte
  PTY/signal/exit parity verification because it wraps every turn.

**Delta:** #92 now scopes only the internal-call helpers (`slug`, `changelog`,
`continuation`, `session-watch`) + call-site repointing + the streaming seam.
`pair-wrap`/`pair-scribe` move to new sub-ticket **#96** (deps `[000092]`, reuses
this issue's streaming seam). Rationale: keep #92 the cohesive "route internal
calls" consolidation; pair-wrap deserves its own review boundary. Both remain
already-Go repackaging (no logic change), so the carve-out is about cohesion +
focused verification, not risk. #91's sequence updated to include #96.
