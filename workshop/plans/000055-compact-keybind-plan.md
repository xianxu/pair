# Alt+Shift+C Compaction Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bind `Alt+Shift+C` to a one-keypress "compaction" — the agent distills the session into a durable continuation doc, then the same-tag session restarts with a fresh conversation seeded from that doc.

**Architecture:** No new verb. `pair continue` becomes context-aware: run *outside* a pair session it fresh-starts (today's behavior); run *inside* its own live pane it parks the scrollback, writes a restart marker carrying the continuation slug, and kills the session — the waiting outer `bin/pair` then re-execs `pair continue <slug>` (now outside zellij, so the fresh branch) under the same tag. The keybind just sends the agent a compaction prompt; the agent runs `pair continue <slug>`.

**Tech Stack:** bash (`bin/pair`), zellij KDL (`zellij/config.kdl`), Lua (`nvim/init.lua`), bash test harness (`tests/pair-continue-test.sh`). Spec: `workshop/issues/000055-compact-keybind.md`.

---

## Core concepts

This is a bash IO-shell feature (ARCH-PURE: the codebase is an inherently-IO CLI; there is no pure business core to extract — the discipline here is keeping the IO seams *injectable* so the logic is testable without a real zellij/agent). The "entities" are decision predicates and a marker format; the "integration points" are the filesystem (scrollback copy, marker write) and process control (kill, re-exec), each behind an env seam so tests drive the real script.

### Pure entities (the decision logic)

| Name | Lives in | Status |
|------|----------|--------|
| compaction-context predicate | `bin/pair` (inline, near `in_zellij_pane`) | new |
| restart-marker `continue=` field | `bin/pair` (`handle_restart_marker` + writer) | modified |

- **compaction-context predicate** — decides whether an invocation of `pair continue <slug>` is an in-session compaction vs a normal fresh start.
  - **Relationships:** depends on the existing `in_zellij_pane` ancestry helper (`bin/pair:703-717`) 1:1; reads `$ZELLIJ_SESSION_NAME` + `$PAIR_TAG` only as a tag-match *confirmation*, never as the sole signal.
  - **DRY rationale (ARCH-DRY):** reuses `in_zellij_pane` rather than re-deriving "am I in a zellij pane" from env vars — which is exactly the env-propagation trap the codebase already documents at `bin/pair:696-702`. One source of truth for in-pane detection.
  - **Future extensions:** a `PAIR_FORCE_IN_SESSION` override is the test seam; could later gate other in-session-only verbs.

- **restart-marker `continue=` field** — one new line in the existing restart-marker file (`$HOME/.cache/pair/restart-pair-<tag>`) carrying the continuation slug to seed on relaunch.
  - **Relationships:** read by `handle_restart_marker` alongside the existing `tag`/`agent`/`new_session` fields (`bin/pair:1328-1331`).
  - **DRY rationale:** extends the existing marker format instead of inventing a parallel signalling file. The relaunch reuses `pair continue` itself rather than a bespoke seeding path.
  - **Future extensions:** additional seed kinds (e.g. `resume=`) already exist; `continue=` sits beside them.

### Integration points (where it meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `park_scrollback` | `bin/pair` (early helper) | new | filesystem |
| compaction kill | `bin/pair` (`${PAIR_KILL_CMD:-zellij kill-session}`) | new | zellij process |
| compaction re-exec | `bin/pair` (`handle_restart_marker`) | modified | `exec`/process |
| `PairConfirmCompact` + keybind | `nvim/init.lua`, `zellij/config.kdl` | new | nvim/zellij/agent |

- **`park_scrollback <tag> <agent> [--copy]`** — extracted from the inlined block in `cleanup_quit_marker` (`bin/pair:1229-1243`). Quit path *moves* (session dying); compaction path *copies* (`--copy`) because the live `pair-wrap --scrollback-log` is still appending to `.raw` (a `mv` would yank it). Non-interactive (no `/dev/tty` prompt).
  - **Injected into:** called by both `cleanup_quit_marker` (move) and the in-session compaction branch (copy). DRY: one park implementation, two modes.
  - **Future extensions:** a reaper for accumulated `parked-scrollback-*` (deferred, see #52).
- **compaction kill** — `exec ${PAIR_KILL_CMD:-zellij kill-session} "pair-<tag>"`. Tests set `PAIR_KILL_CMD=true` (an *execable* — `exec :` fails since `:` is a builtin) so the branch exits 0 without killing a real session.
- **compaction re-exec** — `handle_restart_marker` builds `exec "$0" continue "<slug>" "<agent>" -- <args>`; a `PAIR_REEXEC_CAPTURE` seam writes the would-be argv to a file instead of exec'ing, so tests assert it.
- **`PairConfirmCompact` + keybind** — `Alt C` (+`Ctrl Alt c`) → `:lua PairConfirmCompact()` → confirm → `send_to_agent(<prompt>)`. Manually verified (drives real zellij/agent); not unit-tested.

---

## Chunk 1: M1 — `bin/pair` compaction mechanics (testable)

### Task 1: Extract `park_scrollback` helper (ARCH-DRY)

**Files:**
- Modify: `bin/pair` — add helper near `in_zellij_pane` (~after line 717); replace the inlined park block in `cleanup_quit_marker` (`bin/pair:1229-1243`) with a call.

- [ ] **Step 1: Add the helper.** Define after the `in_zellij_pane` helper (so it's in scope for the compaction branch added in Task 2, which runs ~line 700). It depends on `$DATA_DIR`; Task 2 hoists `DATA_DIR` above this point.

```bash
# Park a session's scrollback for later distillation into a continuation.
# Quit path MOVES (session is dying); compaction path COPIES (--copy) because
# the live `pair-wrap --scrollback-log` is still appending to .raw — a move
# would yank the file from under it. Non-interactive. Echoes the parked .raw
# path on success; returns 1 if there's nothing to park.
park_scrollback() {  # <tag> <agent> [--copy]
    local tag="$1" agent="$2" mode="${3:-}"
    local base="$DATA_DIR/scrollback-${tag}-${agent}"
    [ -s "${base}.raw" ] || return 1
    local ts pbase op
    ts="$(date +%Y%m%dT%H%M%S)"
    pbase="$DATA_DIR/parked-scrollback-${tag}-${ts}"
    if [ "$mode" = "--copy" ]; then op=cp; else op=mv; fi
    "$op" "${base}.raw" "${pbase}.raw" || return 1
    [ -f "${base}.events.jsonl" ] && "$op" "${base}.events.jsonl" "${pbase}.events.jsonl"
    : > "$DATA_DIR/parked-${tag}"
    printf '%s\n' "${pbase}.raw"
}
```

- [ ] **Step 2: Rewire `cleanup_quit_marker`** (`bin/pair:1229-1243`) to call the helper in move mode, preserving the existing `[ -t 0 ]` + `/dev/tty` prompt. Replace the inlined `.raw`/`.events.jsonl` `mv` lines with `_pbase_raw="$(park_scrollback "$PAIR_TAG" "$quit_agent")" && _parked=1`. **Leave the `.ansi` handling and the `_parked`-gated cleanup OUTSIDE the helper** — the existing `rm -f …${_sb_base}.ansi` (`bin/pair:1246`) and the `[ "$_parked" = 1 ] || rm -f …` (`bin/pair:1252`) stay in `cleanup_quit_marker` (the helper knows only `.raw`/`.events.jsonl`; `.ansi` is quit-only). Keep the surrounding prompt + messaging.

- [ ] **Step 3: Syntax check.** Run: `bash -n bin/pair` — Expected: clean.

- [ ] **Step 4: Commit.**
```bash
git add bin/pair
git commit -m "#55 M1: extract park_scrollback helper (copy|move) — ARCH-DRY"
```

### Task 2: In-session compaction branch + DATA_DIR hoist

**Files:**
- Modify: `bin/pair` — hoist `DATA_DIR=`/`export PAIR_DATA_DIR` (`bin/pair:726-728`) to right after `unset PAIR_FORCE_TAG` (`bin/pair:668`); add the compaction branch AFTER that hoist and before the `in_zellij_pane` guard (`bin/pair:718`). **Ordering is load-bearing:** `PAIR_FORCE_TAG (665) → unset (668) → DATA_DIR hoist → compaction branch → guard (718)`. The branch calls `park_scrollback`, which dereferences `$DATA_DIR` — placing the branch before the hoist is an unbound-var crash under `set -u` (and `|| true` does NOT rescue it; the abort fires before `||`).

- [ ] **Step 1: Write the failing test** in `tests/pair-continue-test.sh` (append a new section). Drives the REAL `bin/pair` via the `PAIR_FORCE_IN_SESSION` + `PAIR_KILL_CMD` seams.

```bash
# --- in-session compaction (#55) ---
COMPACT_HOME="$RT/cache"; mkdir -p "$COMPACT_HOME"
# fixture scrollback so park has something to copy
mkdir -p "$RT/xdg/pair"
printf 'SCROLLBACK BYTES\n' > "$RT/xdg/pair/scrollback-demo-claude.raw"
run_compact() { # no real kill — PAIR_KILL_CMD=true (an EXECable; `exec :`
                # fails since `:` is a builtin, `exec` only runs externals)
  ( cd "$RT" && HOME="$RT" XDG_DATA_HOME="$RT/xdg" \
    PAIR_TAG=demo PAIR_AGENT=claude \
    PAIR_FORCE_IN_SESSION=1 PAIR_KILL_CMD=true \
    "$PAIR" continue demo >/dev/null 2>&1 )
}
run_compact
MK="$RT/.cache/pair/restart-pair-demo"
grep -q '^continue=demo$'     "$MK" && pass "compact: marker continue=slug" || fail "compact: marker missing continue="
grep -q '^new_session=1$'     "$MK" && pass "compact: marker new_session=1" || fail "compact: marker missing new_session"
grep -q '^tag=demo$'          "$MK" && pass "compact: marker tag"           || fail "compact: marker missing tag"
ls "$RT/xdg/pair"/parked-scrollback-demo-*.raw >/dev/null 2>&1 \
  && pass "compact: scrollback parked (copy)" || fail "compact: no parked scrollback"
[ -s "$RT/xdg/pair/scrollback-demo-claude.raw" ] \
  && pass "compact: original .raw intact (copy not move)" || fail "compact: original .raw lost"

# in-session + invalid slug → error, NO marker, NO kill (spec Done-when)
rm -f "$RT/.cache/pair/restart-pair-demo"
( cd "$RT" && HOME="$RT" XDG_DATA_HOME="$RT/xdg" PAIR_TAG=demo PAIR_AGENT=claude \
  PAIR_FORCE_IN_SESSION=1 PAIR_KILL_CMD=true "$PAIR" continue bogus >/dev/null 2>&1 )
rc=$?
[ "$rc" -eq 1 ] && [ ! -f "$RT/.cache/pair/restart-pair-demo" ] \
  && pass "compact: invalid in-session slug errors without killing" \
  || fail "compact: invalid in-session slug should exit 1 + no marker (rc=$rc)"
```

- [ ] **Step 2: Run it, verify it fails.** Run: `bash tests/pair-continue-test.sh` — Expected: FAIL (no compaction branch yet; no marker written).

- [ ] **Step 3: Hoist `DATA_DIR`.** Move `DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/pair"` + `export PAIR_DATA_DIR="$DATA_DIR"` (`bin/pair:726-728`) up to **immediately after `unset PAIR_FORCE_TAG` (`bin/pair:668`)** — this is above the compaction branch (Step 4) which needs it. Grep that nothing in 668-726 reads `DATA_DIR` before the new position (it's a pure assignment, no side effects). The original site at 726 becomes a no-op; remove it.

- [ ] **Step 4: Add the compaction branch** immediately after the `DATA_DIR` hoist (Step 3) and BEFORE the `in_zellij_pane` guard (`bin/pair:718`):

```bash
# In-session compaction (#55): `pair continue <slug>` run from INSIDE its own
# live pane must NOT fresh-start (the in_zellij_pane guard below rejects it and
# --session would nest). Instead: park scrollback (copy — pair-wrap still
# appends), drop a restart marker carrying the slug, kill the session; the
# outer bin/pair then re-execs `pair continue <slug>` fresh under the same tag.
# Detection is ANCESTRY-based (in_zellij_pane), not $ZELLIJ* env — cmux
# propagates those to sibling non-pair panes (bin/pair:696-702), and a false
# positive here would park+kill the wrong session. PAIR_FORCE_IN_SESSION is the
# test seam.
if [ -n "${CONTINUE_DOC:-}" ]; then
    _compact=0
    if [ "${PAIR_FORCE_IN_SESSION:-0}" = "1" ]; then
        _compact=1
    elif in_zellij_pane && [ -n "${PAIR_TAG:-}" ] \
        && [ "${ZELLIJ_SESSION_NAME:-}" = "pair-${PAIR_TAG}" ]; then
        _compact=1
    fi
    if [ "$_compact" = "1" ]; then
        : "${PAIR_TAG:?compaction needs PAIR_TAG}"
        _cagent="${PAIR_AGENT:-${AGENT:-claude}}"
        printf 'pair: compacting pair-%s — parking scrollback, restarting from continuation…\n' \
            "$PAIR_TAG" >&2
        park_scrollback "$PAIR_TAG" "$_cagent" --copy >/dev/null 2>&1 || true
        mkdir -p "$HOME/.cache/pair"
        {
            printf 'tag=%s\n' "$PAIR_TAG"
            printf 'agent=%s\n' "$_cagent"
            printf 'new_session=1\n'
            printf 'continue=%s\n' "$_cslug"
        } > "$HOME/.cache/pair/restart-pair-${PAIR_TAG}"
        touch "$HOME/.cache/pair/quit-pair-${PAIR_TAG}"
        exec ${PAIR_KILL_CMD:-zellij kill-session} "pair-${PAIR_TAG}"
    fi
fi
```

Notes: `_cslug` is the validated slug from the continue block (`bin/pair:612`); `CONTINUE_DOC` non-empty means a slug was given + resolved, so invalid/missing slugs already errored out *before* this branch (satisfies the "validate before park/kill" edge). `exec :` (the no-op seam) makes the branch return cleanly in tests without killing.

- [ ] **Step 5: Run tests, verify pass.** Run: `bash tests/pair-continue-test.sh` — Expected: PASS (all compaction assertions + existing fresh-start assertions green).

- [ ] **Step 6: Verify the fresh-start path is untouched.** Run: `bash -n bin/pair` and the existing `probe`-based assertions — Expected: `forced_tag` empty / arg-forwarding tests still pass.

- [ ] **Step 7: Commit.**
```bash
git add bin/pair tests/pair-continue-test.sh
git commit -m "#55 M1: in-session compaction branch (ancestry-gated) + DATA_DIR hoist"
```

### Task 3: `handle_restart_marker` seeds the relaunch from `continue=`

**Files:**
- Modify: `bin/pair` — `handle_restart_marker` (`bin/pair:1323-1402`): read `continue=`, and when set, re-exec `pair continue <slug> <agent> -- <args>`.

- [ ] **Step 1: Write the failing test** (append to the compaction section). Use a `PAIR_REEXEC_CAPTURE` seam so `handle_restart_marker` writes the argv instead of exec'ing.

```bash
# handle_restart_marker continue= → re-exec argv
cat > "$RT/.cache/pair/restart-pair-demo" <<EOF
tag=demo
agent=claude
new_session=1
continue=demo
EOF
# config supplies the relaunch -- args
mkdir -p "$RT/xdg/pair"
printf '{"args":["--dangerously-skip-permissions"],"session_id":"x"}' \
  > "$RT/xdg/pair/config-demo-claude.json"
CAP="$RT/reexec.txt"
( cd "$RT" && HOME="$RT" XDG_DATA_HOME="$RT/xdg" \
  PAIR_REEXEC_CAPTURE="$CAP" PAIR_HANDLE_RESTART_ONLY=1 \
  SESSION=pair-demo "$PAIR" >/dev/null 2>&1 )
grep -Eq 'continue demo claude -- .*--dangerously-skip-permissions' "$CAP" \
  && pass "restart: re-exec argv = continue <slug> <agent> -- <args>" \
  || fail "restart: wrong re-exec argv ($(cat "$CAP" 2>/dev/null))"
```

(Implementation note: a tiny `PAIR_HANDLE_RESTART_ONLY=1` hook that calls `handle_restart_marker` then exits — analogous to `PAIR_DEBUG_ARGS` — lets the test drive just this function. **Place it right after `PAIR_TAG`/`AGENT`/`DATA_DIR` are all set (~`bin/pair:882`), NOT near the `PAIR_DEBUG_HISTORY` probe (~1096)** — the history scan does live `zj list-sessions` calls (`bin/pair:919,932,1058`) between them, which would make the test depend on a running zellij daemon. `handle_restart_marker` needs only `SESSION` (env, set by the test), `DATA_DIR`, `PAIR_TAG`, `AGENT` — all available by ~882. Add it in Step 3.)

- [ ] **Step 2: Run it, verify it fails.** Run: `bash tests/pair-continue-test.sh` — Expected: FAIL.

- [ ] **Step 3: Implement.** In `handle_restart_marker`: after reading the existing fields, add `r_continue=$(awk -F= '$1=="continue"{print $2; exit}' "$marker")`. Define the `_reexec` seam once near the top of the function:

```bash
_reexec() { # capture in tests, exec in prod
    if [ -n "${PAIR_REEXEC_CAPTURE:-}" ]; then printf '%s\n' "$*" > "$PAIR_REEXEC_CAPTURE"; return 0; fi
    exec "$@"
}
```

Then **nest the `continue=` arm INSIDE the existing `if [ "$r_new_session" = "1" ]` block** (`bin/pair:1360-1369`) — the compaction marker always sets `new_session=1`, so it belongs in that arm, not as a sibling. Keep `rm -f "$cfg"` (it's already there for new_session):

```bash
if [ "$r_new_session" = "1" ]; then
    rm -f "$cfg"                                   # fresh conversation: drop saved session id
    export PAIR_FORCE_TAG="$r_tag"
    if [ -n "$r_continue" ]; then
        _reexec "$0" continue "$r_continue" "$r_agent" -- $r_args
    else
        _reexec "$0" "$r_agent" -- $r_args          # existing plain fresh-restart
    fi
else
    # ... existing resume path unchanged ...
fi
```
Replace the existing `exec` calls in this arm with `_reexec` so the seam covers both. The relaunch `$r_args` come from the saved config (`bin/pair:1354`), unchanged. **PAIR_DEV** needs no handling — an exported env var survives `exec "$0"`. **Codex:** the `continue=` arm lives in the `new_session` path, which does NOT apply codex's `resume <id>` token reordering (`bin/pair:1391`, the *resume* arm) — so the codex relaunch argv is clean `continue <slug> codex -- <args>`. Add the `PAIR_HANDLE_RESTART_ONLY=1` early hook (Step 1 note, placed ~`bin/pair:882`).

- [ ] **Step 4: Run tests, verify pass.** Run: `bash tests/pair-continue-test.sh` — Expected: PASS.

- [ ] **Step 5: Commit.**
```bash
git add bin/pair tests/pair-continue-test.sh
git commit -m "#55 M1: handle_restart_marker re-execs pair continue <slug> on continue= marker"
```

### Task 4: Wire `test-continue` stays green + close M1

- [ ] **Step 1:** Run the full target: `make test-continue` — Expected: PASS (compaction + fresh-start).
- [ ] **Step 2:** `bash -n bin/pair` — Expected: clean.
- [ ] **Step 3:** Close the milestone: `sdlc milestone-close --issue 55 --milestone M1 --verified '<evidence>'` (auto-dispatches the boundary review; fix Critical/Important before crossing).

---

## Chunk 2: M2 — keybind + nvim wiring (manual-verified)

### Task 5: zellij keybind

**Files:**
- Modify: `zellij/config.kdl` — add a bind near the `Alt N` / `Ctrl Alt n` binds (`config.kdl:210,223`).

- [ ] **Step 1: Add the bind** (mirror the `Alt+x` confirm-routing pattern; `Alt C` = capital ⇒ shift; add the `Ctrl Alt c` alias for the macOS Option-dead-key composer):
```kdl
bind "Alt C" "Ctrl Alt c" {
    MoveFocus "Down";
    Write 28;            // <C-\>
    Write 14;            // <C-n>
    WriteChars ":lua PairConfirmCompact()";
    Write 13;            // <Enter>
}
```
- [ ] **Step 2:** Reload check — `zellij setup --check` (or load a throwaway session) to confirm the KDL parses. Expected: no parse error.
- [ ] **Step 3: Commit.**
```bash
git add zellij/config.kdl
git commit -m "#55 M2: Alt+Shift+C (Alt C / Ctrl Alt c) → PairConfirmCompact"
```

### Task 6: nvim handler + compaction prompt

**Files:**
- Modify: `nvim/init.lua` — add `_G.PairConfirmCompact()` near the other `PairConfirm*` handlers (~`init.lua:2861-2911`), reusing the confirm-dialog + `send_to_agent` helpers.

- [ ] **Step 1: Implement** `PairConfirmCompact()`: confirm dialog ("Compact this session? (writes a continuation, then restarts fresh from it)"), and on confirm `send_to_agent(COMPACT_PROMPT)` (submit). The prompt is **agent-agnostic** (no claude-only skill name):

```lua
local COMPACT_PROMPT = table.concat({
  "Compact this session:",
  "1. Write a continuation doc for this session now — distill the NEXT ACTION,",
  "   open threads, and key decisions/dead-ends into workshop/continuation/",
  "   (use the project's continuation mechanism / pair-continuation writer).",
  "   Choose a short slug.",
  "2. Then run:  pair continue <that-slug>",
  "   This restarts this session with a fresh conversation seeded from the",
  "   continuation. (Run pair-dev continue <slug> if this is a dev checkout.)",
}, "\n")

function _G.PairConfirmCompact()
  pair_ensure_visible_then(function()          -- local, init.lua:2689 (in scope at ~2911)
    local ans = vim.fn.confirm('Compact this session? (continuation + fresh restart)', '&Yes\n&No', 2)
    if ans ~= 1 then return end
    send_to_agent(COMPACT_PROMPT)              -- local, init.lua:696; no 2nd arg ⇒ submits
  end)
end
```
(Verified: `pair_ensure_visible_then` (`init.lua:2689`) and `send_to_agent` (`init.lua:696`) are both `local` but in lexical scope at the ~2861-2911 insertion point. This mirrors the real `PairConfirmQuit`/`pair_confirm_restart_impl` idiom — there is NO `pair_confirm` helper.)

- [ ] **Step 2: Headless sanity** (best-effort — full path needs a live agent): `nvim -l` smoke that the file loads and `PairConfirmCompact` is defined, mirroring `test-lua` style if a seam exists; otherwise note as manual.
- [ ] **Step 3: Commit.**
```bash
git add nvim/init.lua
git commit -m "#55 M2: PairConfirmCompact — confirm + agent-agnostic compaction prompt"
```

### Task 7: End-to-end manual verification (documented in `## Log`)

Automated tests can't drive the full keypress→agent→`pair continue`→restart loop (needs a live agent + real zellij). Manual script:

- [ ] **Step 1:** `pair-dev claude -- --dangerously-skip-permissions`; do a little work so scrollback + a draft exist.
- [ ] **Step 2:** Press `Alt+Shift+C`. Expect the confirm dialog; accept. Expect the compaction prompt to appear in the agent pane and submit.
- [ ] **Step 3:** Agent writes a continuation (a new `workshop/continuation/*-<slug>.md`, git-committed) and runs `pair continue <slug>`.
- [ ] **Step 4:** Expect: session restarts under the SAME tag, fresh conversation, draft seeded `continue from workshop/continuation/<file> — do its NEXT ACTION`, same `-- --dangerously-skip-permissions`. Expect a `parked-scrollback-<tag>-*.raw` recovery copy and the original scrollback untouched at kill time.
- [ ] **Step 5:** Negative: in a normal terminal (not a pane), `pair continue <slug>` still fresh-starts (no compaction). `pair continue bogus` inside a pane errors without killing.
- [ ] **Step 6:** Record the manual evidence in the issue `## Log`.

### Task 8: Close M2 + issue

- [ ] **Step 1:** `sdlc milestone-close --issue 55 --milestone M2 --verified '<manual evidence>'` (boundary review).
- [ ] **Step 2:** `sdlc close --issue 55 --actual <measured> --verified '<summary>'` → `sdlc pr` → `sdlc merge`.

---

## Plan-level notes
- **ARCH-DRY** drove: `park_scrollback` (one impl, copy|move), reusing `in_zellij_pane` for detection, reusing `pair continue` for the relaunch, extending the existing marker format. **Acknowledged parallel (not fully DRY):** the compaction branch re-implements the `restart-<session>` + `touch quit-<session>` + `exec zellij kill-session` sequence that `bin/pair-restart.sh:73-90` also owns. Reuse isn't taken because the in-pane branch already has `PAIR_TAG`/`PAIR_AGENT` in env and avoids `pair-restart.sh`'s subprocess + `agent-<tag>` disk read; the marker *format* is the shared contract (both writers must stay in sync — note it in `## Log`).
- **Codex acceptance (multi-agent):** the only agent-specific surface is the static, agent-agnostic compaction prompt (Task 6). The relaunch carries `r_agent` through unchanged; the `continue=` arm sits in the `new_session` path, which does not apply codex's resume-token reordering — so `continue <slug> codex -- <args>` is clean. Manual Task 7 should run once under `pair-dev codex` to confirm.
- **ARCH-PURE:** no pure core to extract from a bash CLI; the discipline is the injectable IO seams (`PAIR_FORCE_IN_SESSION`, `PAIR_KILL_CMD`, `PAIR_REEXEC_CAPTURE`, `PAIR_HANDLE_RESTART_ONLY`) that let the real script be tested without a live zellij/agent.
- **Milestones:** M1 (mechanics) is independently testable + valuable (compaction works via a hand-run `PAIR_FORCE_IN_SESSION` even before the keybind). M2 (wiring) is manual-verified. Two genuine review boundaries.
