---
id: 000055
status: done
deps: []
github_issue:
created: 2026-06-11
updated: 2026-06-12
estimate_hours: 2.5
actual_hours: 0.91
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

Detailed step-by-step plan: `workshop/plans/000055-compact-keybind-plan.md`
(spec-reviewed + plan-reviewed; 2 review boundaries).

- [x] M1 — `bin/pair` compaction mechanics: `park_scrollback` helper (copy|move,
  ARCH-DRY); in-session compaction branch (ancestry-gated via `in_zellij_pane`,
  placed before the guard with DATA_DIR hoisted); `handle_restart_marker`
  re-execs `pair continue <slug>` on a `continue=` marker; tests via the
  `PAIR_FORCE_IN_SESSION` / `PAIR_FAKE_IN_ZELLIJ` / `PAIR_KILL_CMD` /
  `PAIR_TEST_CALL` / `PAIR_REEXEC_CAPTURE` seams (incl. invalid-slug-no-kill).
- [x] M2 — keybind + nvim wiring: `bind "Alt C" "Ctrl Alt c"` → `PairConfirmCompact`
  (confirm via `pair_ensure_visible_then` + `vim.fn.confirm`, then
  `send_to_agent(<agent-agnostic compaction prompt>)`). Static verification done
  (luac clean; zellij config "Well defined"); **runtime e2e is operator-manual**
  (live keypress → agent → restart can't be driven headlessly):
    1. `pair-dev claude -- --dangerously-skip-permissions`; do a little work.
    2. `Alt+Shift+C` → confirm dialog → Yes. Compaction prompt appears in the agent pane + submits.
    3. Agent writes `workshop/continuation/*-<slug>.md` (git-committed) and runs `pair continue <slug>`.
    4. Session restarts: SAME tag, fresh conversation, draft seeded `continue from … — do its NEXT ACTION`, same `-- --dangerously-skip-permissions`; a `parked-scrollback-<tag>-*.raw` recovery copy exists; **no stray "park as a continuation?" prompt** (the M1 FIX-THEN-SHIP suppression).
    5. Repeat once under `pair-dev codex` (agent-agnostic prompt).

## Log

### 2026-06-12
- 2026-06-12: closed — Both milestones reviewed (M1 FIX-THEN-SHIP fixed, M2 SHIP). M1: make test-continue 21/21 (mechanics via injectable seams). M2: live Alt+Shift+C round-trip dogfooded (continuation → pair continue → same-tag fresh restart, copy-not-move recovery net, no stray prompt). make build/bash -n/luac clean; zellij config Well defined. Atlas updated. Codex deferred (agent-agnostic prompt; relaunch carries r_agent).; review verdict: SHIP
- 2026-06-12: closed M2 — M2 runtime e2e PASS (claude), dogfooded live: real Alt+Shift+C → confirm → agent wrote continuation 20260612T002626-compact.md → pair continue compact → same-tag (3) fresh restart seeded from doc (round-trip proof). Recovery net: parked-scrollback-3 (425KB) copy exists AND live scrollback-3-claude.raw intact (copy-not-move); no stray park prompt. Static: luac clean, zellij config Well defined. Atlas updated (keybind + suppression).; review verdict: SHIP
- **M2 runtime e2e PASS (`claude`) — dogfooded live.** Steps 2–4 executed by an
  actual `Alt+Shift+C` keypress in a live `pair-dev claude` session: confirm
  dialog → agent wrote `workshop/continuation/20260612T002626-compact.md` →
  `pair continue compact` → same-tag (`3`) session restarted with a fresh
  conversation seeded from the doc. The continuation was read in a fresh seeded
  conversation under the same tag — that *is* the round-trip proof. Recovery-net
  checks post-restart: `parked-scrollback-3-20260612T002733.raw` (425 KB) exists
  AND live `scrollback-3-claude.raw` still present → park is **copy-not-move** ✓;
  no `restart-*` / `quit-*` markers lingering → **no stray "park as a
  continuation?" prompt** (M1 FIX-THEN-SHIP suppression held) ✓. Step 5 (repeat
  under `pair-dev codex`) is optional — the injected prompt is agent-agnostic.

### 2026-06-11
- 2026-06-11: closed M1 — make test-continue → 21/21 (11 fresh-start + 10 compaction: marker shape, park copy vs move, real tag-match predicate via PAIR_FAKE_IN_ZELLIJ, invalid-slug-no-kill, re-exec argv); make build clean; bash -n clean. Mechanics drive the REAL bin/pair via injectable seams, no live zellij/agent. Atlas updated (compaction flow + park_scrollback).; review verdict: FIX-THEN-SHIP
- **M1 boundary review FIX-THEN-SHIP** → fixed the one Important before M2: compaction `touch quit-<session>` made the outer `cleanup_quit_marker` fire its "park as a continuation?" nudge mid-compaction (the `.raw` survives the copy-park). Guarded the nudge with `[ ! -f restart-$SESSION ]` so it's skipped whenever a restart is pending — also de-noises the inherited Alt+n/Shift+Alt+N paths (a restart isn't a quit). M2 Task 7 e2e must confirm no stray prompt.
- Brainstormed (superpowers-brainstorming). Decisions: agent-driven trigger; same tag + agent + `-- <args>`, fresh convo; park scrollback recovery net; confirm first. Operator pushed back on a separate `pair recompact` verb → unified into a context-aware `pair continue` (in-session = compact+restart; outside = fresh start; self-referential — outer relaunch runs outside zellij so takes the fresh branch).
- **M1 landed** (commit 9b9e0a1): park_scrollback (copy|move, ARCH-DRY), in-session compaction branch (ancestry-gated, before the guard), handle_restart_marker `continue=` re-exec. 21/21 tests green (`make test-continue`), `make build` clean. Impl refinements beyond the plan: (a) `handle_restart_marker` ALSO had to be hoisted (defined ~1323, after the picker, which would hang the test on the name prompt) — moved up with DATA_DIR; (b) a single generic `PAIR_TEST_CALL` dispatcher replaced per-function hooks; (c) `_reexec` must `exit` not `return` in capture mode (else the function falls through into the resume arm and overwrites the capture — caught by the test); (d) the `compact_env` test helper uses `env` so seam assignments arriving via `"$@"` are treated as env, not run as a command (bash only recognizes literal leading assignments). DATA_DIR hoisted just after `unset PAIR_FORCE_TAG`, before the `command -v zellij` gate (per plan-quality finding #1) so the dispatcher tests don't need zellij on PATH.
- Fresh-eyes spec review (APPROVE-WITH-CHANGES). Folded in 2 Critical + Important/Minor: **C1** the `in_zellij_pane` guard (`bin/pair:718`) hard-exits inside zellij → branch must precede it; **C2** detect via the `in_zellij_pane` PPID-ancestry helper, NOT `$ZELLIJ*` env (cmux propagates those to sibling non-pair panes → false-positive park+kill). Park must **copy** not `mv` (live `pair-wrap` appends to `.raw`). Re-exec argv = `continue <slug> <agent> -- <args>`; relaunch args from saved config; `PAIR_DEV` free. Key token `"Alt C"` (+ `"Ctrl Alt c"` alias). Test seams `PAIR_FORCE_IN_SESSION` / `PAIR_KILL_CMD`. Multi-agent (claude+codex) Done-when added.
