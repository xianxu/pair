---
id: 000046
status: done
deps: []
github_issue:
created: 2026-06-03
updated: 2026-06-03
estimate_hours: 1
actual_hours: 1
---

# pair-dev dev-mode rebuild entrypoint

## Problem

The agent pane is spawned by `zellij/layouts/main.kdl` via
`sh -c '... exec pair-wrap ... $PAIR_AGENT ...'`, which resolves `pair-wrap`
through PATH (repo `bin/` is first, per `.zshrc`). That path **never rebuilds**:
`sh -c` doesn't source `.zshenv`, and `exec` bypasses shell functions, so the
`construct/dev-aliases.sh` rebuild-on-call function cannot reach it. When
`repo/bin/pair-wrap` is stale — or absent, since it's gitignored — PATH silently
falls through to an old `~/.local/bin/pair-wrap`, and the running wrapper drifts
from source with no error.

Observed this session: the adaptation flight recorder (#000045) went silent for
all Go-emitted aspects (1/2/4/5) while only nvim's Lua emitter (aspect 7) logged
— because the installed `pair-wrap`/`pair-slug` binaries predated #000045. See
`workshop/lessons.md` (stale-`~/.local/bin` footgun) and `atlas/architecture.md`
§ pair-wrap staleness.

We want **two modes**: deployed (prebuilt binary, no toolchain dependency) and
dev (always fresh from source). dev-aliases' freshness is an interactive-shell
property that structurally does not extend to the zellij-launched wrapper.

## Spec

- New `bin/pair-dev` entrypoint: sets `PAIR_DEV=1` and `exec`s the sibling
  `bin/pair "$@"`. No build logic of its own — just the mode flag + handoff.
- `bin/pair` gains ONE gated block, immediately before the
  `zellij --new-session-with-layout` create call (~line 1906): when `PAIR_DEV`
  is set, run `make -C "$PAIR_HOME" build` (incremental — only changed binaries
  recompile) so `repo/bin/` holds fresh binaries before the layout's
  `exec pair-wrap` resolves them via PATH. Build failure warns loudly and
  continues with last-good binaries (don't strand the user mid-restart;
  loudness is the opposite of the silent staleness we're fixing).
- **Restart correctness (the crux):** Alt+n / Shift+Alt+N / Ctrl+Alt+n route
  through `bin/pair-restart.sh` → `handle_restart_marker` → `exec "$0"` where
  `$0` is `bin/pair`. Because `exec` preserves the environment, an exported
  `PAIR_DEV` survives every restart, so the gated build re-fires on each
  restart — the session keeps whichever mode it was launched in. A build placed
  only in `pair-dev` would be skipped on restart (the rejected design).
- Deployed mode: `PAIR_DEV` unset → block skipped → no `make`/`go` invocation.
- Build at the **create path only** (line 1906): a plain attach to a live
  session spawns no new pair-wrap, so it correctly skips the rebuild; a restart
  kills-then-recreates, so it correctly triggers one.
- `make build` covers all `GO_BINS` (pair-wrap, pair-slug,
  pair-scrollback-render, pair-scribe), including pair-slug which the layout
  never launches.

Out of scope: doctor.sh color table (separate pending change); a doctor
stale-binary check (separate follow-up).

## Done when

- `bin/pair-dev` exists, sets `PAIR_DEV=1`, execs the sibling `pair`, forwards args.
- `bin/pair` rebuilds via `make build` iff `PAIR_DEV` is set, before the zellij launch.
- Deployed path (`PAIR_DEV` unset) invokes no toolchain.
- Restart (Alt+n / Shift+Alt+N) re-fires the rebuild because `PAIR_DEV` survives `exec "$0"`.
- `bin/pair-dev --help` proves the handoff: forwards args to `pair`, prints usage, exits 0. (It does NOT build — `--help` short-circuits at `bin/pair:60`, before the create-path rebuild; building on `--help` is intentionally avoided since `--help` is not a launch.)

## Plan

- [x] Add `bin/pair-dev` (flag + symlink-safe sibling exec of `pair`).
- [x] Add the `PAIR_DEV`-gated rebuild to `bin/pair` before line 1906 — via sourced `bin/lib/dev-rebuild.sh` (`dev_rebuild`).
- [x] Update help text in `bin/pair` + README to mention dev mode + `pair-dev`.
- [x] Regression test `tests/dev-rebuild-test.sh` (deployed=no-toolchain, dev=builds, errexit-safe) wired into `make test`.
- [x] Verify: `pair --help` clean; `bin/pair-dev --help` chains handoff+args; deployed path skips build (test); restart env-survival traced.
- [x] Atlas: pair-dev + dev/deployed binary-freshness section in `atlas/architecture.md`.

## Revisions

### 2026-06-03 — plan-quality sharpenings (sdlc change-code judge: INFO)
- **errexit safety:** the gated build must not abort `bin/pair` (`set -euo
  pipefail`) on failure — least of all mid-restart. Resolved by isolating the
  hook in `bin/lib/dev-rebuild.sh::dev_rebuild`, which warns-and-`return 0`s on
  build failure (loud, continues with last-good binaries).
- **deployed-mode invariant test:** added `tests/dev-rebuild-test.sh` asserting
  PAIR_DEV-unset invokes NO toolchain (the silent-regression guard the judge
  flagged), plus dev-builds and errexit-safety, via a stubbed `make`.
- **export, not bare assign:** `pair-dev` uses `export PAIR_DEV=1` — restart
  survival depends on it riding through `exec "$0"`.

## Log

### 2026-06-03
- 2026-06-03: closed — tests/dev-rebuild-test.sh 3/3 (deployed=no-toolchain, dev builds, errexit-safe); pair-dev --help forwards args + shows DEV MODE help; create-only placement confirmed (attach branch exits at bin/pair:1369, restart re-execs into create path with PAIR_DEV surviving exec); make build green; doctor test unaffected; review verdict: SHIP
- Diagnosed the stale-binary root cause via the flight recorder going silent
  (only nvim aspect-7 logged). Confirmed installed `pair-wrap`/`pair-slug`
  predated #000045; `make install` + restart fixed the live session.
- Established why dev-aliases freshness can't reach the wrapper: `sh -c … exec`
  uses PATH, not the rebuild function. Restart re-execs `$0`=`bin/pair`, so the
  rebuild hook must live in `bin/pair` gated by an exported `PAIR_DEV`.
- Implemented: `bin/pair-dev` (export + symlink-safe exec of sibling `pair`),
  `bin/lib/dev-rebuild.sh::dev_rebuild` (no-op unless PAIR_DEV; errexit-safe),
  sourced + called on `bin/pair`'s create path before the zellij launch
  (verified create-only: the attach branch `exit $rc`s at bin/pair:1369).
- Verified line 1906 is create-only; restart-from-attach re-execs into the
  create path so it rebuilds too. Tests: `tests/dev-rebuild-test.sh` 3/3 green;
  `bin/pair-dev --help` forwards args + shows the new DEV MODE help. `bash -n`
  clean on all touched scripts.
