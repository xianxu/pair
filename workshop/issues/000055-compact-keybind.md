---
id: 000055
status: open
deps: []
github_issue:
created: 2026-06-11
updated: 2026-06-11
estimate_hours: 2.5
---

# Alt+Shift+C compaction: continuation + in-session restart via context-aware pair continue

## Problem

A long pair session bloats the agent's context. We want a one-keypress
"compaction" — distill the session into a durable `continuation` doc, then
restart the *same* session with a fresh agent conversation seeded from that
doc. Today this is a manual multi-step dance (ask agent to write a
continuation → `Alt+x` → `pair continue` in a new terminal).

Constraint that shapes the design: **creating a continuation is an agent
judgment step** (distill NEXT ACTION / open threads / decisions, then the
`cmd/pair-continuation` writer commits it) — there is no mechanical one-shot.
And **you cannot fresh-start a session from inside zellij** (`--session` from
within attaches/nests — see `bin/pair:685`), so the restart must go through the
existing kill → `handle_restart_marker` → outer-relaunch path.

## Spec

Bind **`Alt+Shift+C`** to a compaction flow. Decisions (brainstorm, operator):
agent-driven trigger; same tag + same agent + same `-- <args>`, fresh
conversation; park scrollback as a recovery net; confirm before firing.

**No new verb.** `pair continue` becomes **context-aware** — it is both the
trigger and the relaunch:

```
pair continue <slug>
  ├─ run OUTSIDE a pair session            → fresh session seeded from <slug>   [today's behavior]
  └─ run INSIDE its own live pair session  → COMPACT: park scrollback + restart
                                             marker(continue=<slug>, same tag, fresh) + kill-session
```

Self-referential: in-session `pair continue` only *marks + kills*; the waiting
outer `bin/pair` re-execs `PAIR_FORCE_TAG=<tag> pair continue <slug> …`, which
now runs *outside* zellij and so takes the fresh-start-seeded branch. One
command, both roles.

### Components

1. **`zellij/config.kdl`** — `bind "Alt C"` (capital `C` IS shift; there is no
   `Shift` token — cf. `bind "Alt N"` at `config.kdl:223`) routing to nvim
   `:lua PairConfirmCompact()` (same `MoveFocus`/`WriteChars` pattern as
   `Alt+x`/`Alt+n`). **Add the `"Ctrl Alt c"` alias too** — `Alt+n` carries a
   `Ctrl Alt n` alias (`config.kdl:210`) precisely to defeat the macOS
   Option-as-dead-key composer, and Option+c is a live composer, so the alias
   is needed here.

2. **`nvim/init.lua`** — `PairConfirmCompact()`: confirm dialog
   ("Compact this session? continuation + fresh restart"), then
   `send_to_agent(<compaction prompt>)` (`init.lua:696-741`, injects+submits).
   Prompt instructs the agent to: create a continuation now (its normal way —
   **agent-agnostic phrasing, not a claude-only skill name**), then run
   `pair continue <its-slug>`. The agent chose the slug, so it knows it.

3. **`bin/pair` — `pair continue` in-session compaction branch.** Two
   load-bearing corrections from the spec review:
   - **(C1) Placement.** `bin/pair:718-722` hard-exits with "already running
     inside a zellij session" via the `in_zellij_pane` guard. The compaction
     branch must run **before** that guard and `exec`/`exit` so it never falls
     through. Argv (slug/agent) is resolved earlier (continue block
     `bin/pair:598-639`), but `DATA_DIR` is set *after* the guard
     (`bin/pair:726`) and the park needs it — so place the branch after
     DATA_DIR setup but before the guard, or resolve DATA_DIR locally.
   - **(C2) Detection = ground truth, not env vars.** Detect "inside my own
     live pair session" with the existing **`in_zellij_pane` PPID-ancestry
     helper** (`bin/pair:703-717`) AND `$ZELLIJ_SESSION_NAME == "pair-$PAIR_TAG"`
     as a tag-match confirmation. Do NOT key on `$ZELLIJ*` alone: cmux
     propagates `ZELLIJ`/`ZELLIJ_SESSION_NAME`/`PAIR_TAG` to sibling *non-pair*
     panes (`bin/pair:696-702`), so an env-only check false-positives and would
     park+kill a session from a pane that isn't it — catastrophic here.
   - When in-session+confirmed: validate `<slug>`; **print "compacting
     pair-<tag>…" BEFORE the exec** (M4); `park_scrollback` (see below); write
     the restart marker with `tag`, `agent`, `new_session=1`, **`continue=<slug>`**;
     `exec ${PAIR_KILL_CMD:-zellij kill-session} pair-<tag>`.
   - When NOT in-session: today's fresh-start-seeded behavior, unchanged.

4. **`bin/pair` — `park_scrollback` helper (extracted, COPY not move).** The
   park block is currently inlined in `cleanup_quit_marker` (`bin/pair:1229-1243`),
   runs in the *outer* process, `mv`s the `.raw` (destructive), and prompts on
   `/dev/tty`. For a *live* pre-kill park it must be a **non-interactive,
   non-destructive** variant: **copy** the `.raw`/`.events.jsonl` (the live
   `pair-wrap --scrollback-log` is still appending — `mv` would yank the file
   out from under it), skip the tty prompt, and use `$PAIR_AGENT` from the pane
   env (not the `agent-<tag>` file). Extract `park_scrollback <tag> <agent>
   [--copy]` and call it from both the quit path (move) and the compaction path
   (copy). Scrollback `.raw` is confirmed present + non-empty mid-session.

5. **`bin/pair` — `handle_restart_marker` (`bin/pair:1323-1402`).** When the
   marker carries `continue=<slug>`: re-exec
   `exec "$0" continue "<slug>" "<agent>" -- <args>` (note `continue` + slug
   come **before** the agent — that's the parse order at `bin/pair:598-639`)
   with `PAIR_FORCE_TAG=<tag>`. Use the `new_session=1` path
   (`bin/pair:1360-1369`): `rm -f` the saved config (fresh conversation, drops
   the old session id). The relaunch's `-- <args>` come from the **saved
   config `args`** (`bin/pair:1354`), NOT the continuation doc. `PAIR_DEV` is
   preserved automatically (exported env survives `exec "$0"`) — no special
   handling needed.

6. **Tests** (`tests/pair-continue-test.sh`, extended). Concrete seams since
   detection is now ancestry-based (env vars won't trip it in a test process):
   - `PAIR_FORCE_IN_SESSION=1` (or `PAIR_IN_SESSION_TAG=<tag>`) override to
     reach the compaction branch without faking process ancestry;
   - `PAIR_KILL_CMD=:` (no-op) so no real session dies.
   Assert: restart marker fields (`continue=<slug>`, `new_session=1`, tag,
   agent), a parked-scrollback copy exists AND the original `.raw` is **still
   present** (copy not move), and the `handle_restart_marker` re-exec line is
   `… continue <slug> <agent> -- <args>` with `PAIR_FORCE_TAG`. Keep the
   existing fresh-start `pair continue` assertions green (no regression).

### Edges & safety

- **Agent botches / never writes the continuation** → nothing restarts; session
  stays alive. Fully recoverable.
- **Bad continuation found later** → re-distill from the parked scrollback copy.
- **`pair continue <slug>` in-session with bad/missing slug** → clear error,
  session untouched (validate before parking/killing).
- **False-positive in-session detection** → prevented by the ancestry guard +
  tag-match (C2); env-var propagation alone never trips it.
- **Manual in-session use** → the "compacting pair-<tag>…" notice prints before
  the exec (the keybind path already confirmed).
- **cwd** → slug resolution uses `git rev-parse` / `$PWD` (`bin/pair:599-600`);
  assumes the agent runs `pair continue` from inside the repo (it does — pane
  cwd is the launch dir). State this assumption.
- **`$ZELLIJ` on the outer relaunch** → the outer `bin/pair` is the launcher
  (parent of zellij), guaranteed by the C1 guard to start from a plain
  terminal, so `in_zellij_pane` is false there → the relaunched `pair continue`
  takes the fresh-start branch (no compaction loop). ✓ verified in review.

## Done when

- `Alt+Shift+C` → confirm → agent writes a continuation → `pair continue <slug>`
  → same-tag session restarts with a fresh conversation seeded from the doc,
  same agent + `-- <args>`.
- Old scrollback is **copied** to a parked recovery file before the kill; the
  live `.raw` is untouched.
- `pair continue <slug>` outside a session is unchanged (fresh start) — existing
  `tests/pair-continue-test.sh` assertions stay green.
- In-session `pair continue` with no/invalid slug errors without killing.
- In-session detection is ancestry-based; an env-only sibling-pane false
  positive does NOT trigger compaction.
- Works for ≥2 agents (claude + codex) — the injected prompt is agent-agnostic.
- `bash -n bin/pair` clean; tests cover the in-session marker + park-copy +
  relaunch argv via the `PAIR_FORCE_IN_SESSION` / `PAIR_KILL_CMD` seams.

## Plan

- [ ] (filled by writing-plans → workshop/plans/000055-*-plan.md)

## Log

### 2026-06-11
- Brainstormed (superpowers-brainstorming). Decisions: agent-driven trigger; same tag + agent + `-- <args>`, fresh convo; park scrollback recovery net; confirm first. Operator pushed back on a separate `pair recompact` verb → unified into a context-aware `pair continue` (in-session = compact+restart; outside = fresh start; self-referential — outer relaunch runs outside zellij so takes the fresh branch).
- Fresh-eyes spec review (APPROVE-WITH-CHANGES). Folded in 2 Critical + Important/Minor: **C1** the `in_zellij_pane` guard (`bin/pair:718`) hard-exits inside zellij → branch must precede it; **C2** detect via the `in_zellij_pane` PPID-ancestry helper, NOT `$ZELLIJ*` env (cmux propagates those to sibling non-pair panes → false-positive park+kill). Park must **copy** not `mv` (live `pair-wrap` appends to `.raw`). Re-exec argv = `continue <slug> <agent> -- <args>`; relaunch args from saved config; `PAIR_DEV` free. Key token `"Alt C"` (+ `"Ctrl Alt c"` alias). Test seams `PAIR_FORCE_IN_SESSION` / `PAIR_KILL_CMD`. Multi-agent (claude+codex) Done-when added.
