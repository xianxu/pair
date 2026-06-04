---
id: 000046
status: working
deps: []
github_issue:
created: 2026-06-03
updated: 2026-06-03
estimate_hours: 1
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
- `bin/pair-dev --help` proves the chain (builds, then prints `pair`'s usage, exits 0).

## Plan

- [ ] Add `bin/pair-dev` (flag + symlink-safe sibling exec of `pair`).
- [ ] Add the `PAIR_DEV`-gated `make build` block to `bin/pair` before line 1906.
- [ ] Update help text in `bin/pair`/README to mention dev mode + `pair-dev`.
- [ ] Verify: `pair --help` clean; `bin/pair-dev --help` chains build→pair; deployed path skips build; trace restart env-survival.
- [ ] Atlas: note pair-dev + dev/deployed binary-freshness in `atlas/architecture.md`.

## Log

### 2026-06-03
- Diagnosed the stale-binary root cause via the flight recorder going silent
  (only nvim aspect-7 logged). Confirmed installed `pair-wrap`/`pair-slug`
  predated #000045; `make install` + restart fixed the live session.
- Established why dev-aliases freshness can't reach the wrapper: `sh -c … exec`
  uses PATH, not the rebuild function. Restart re-execs `$0`=`bin/pair`, so the
  rebuild hook must live in `bin/pair` gated by an exported `PAIR_DEV`.
