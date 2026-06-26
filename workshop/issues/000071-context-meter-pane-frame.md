---
id: 000071
status: working
deps: []
github_issue:
created: 2026-06-25
updated: 2026-06-25
estimate_hours:
started: 2026-06-25T22:10:36-07:00
---

# per-agent context-window meter in the zellij pane frame

## Problem

The operator has no at-a-glance signal for how full each agent's context window is, so
deciding when to start a fresh session (Shift+Alt+N) is guesswork. We want the live context
size shown in each agent's **zellij pane frame title**, beside the agent name and cwd —
`claude (970k) [~/brain]`.

(Migrated from ariadne#131, which was misfiled — this is 100% `pair` code. The ariadne issue
originally proposed *estimating* context from scrollback line count; brainstorm + review
superseded that with reading precise token usage from each agent's transcript. Full review
history lives in ariadne#131's Log; the durable design is below.)

## Done when

- Each agent pane's **zellij frame title** shows live context size as an absolute,
  humanized token count — `claude (970k) [~/brain]`, `codex (60k) [~/pair]` — for
  agents whose transcript exposes usage (claude, codex). Agy panes keep the current
  `agy [~/cwd]` (no number; no usable token source).
- The number is read from the agent's **real transcript token usage** (precise), not
  estimated from scrollback line count, and needs **no model→window catalog** (absolute
  count, not %).
- The title **refreshes while the session is active** (≈60s cadence, gated on recent
  draft-or-transcript activity) and does **not** churn titles on idle/background sessions.
- Unit tests cover the per-agent token read against captured transcript fixtures
  (claude = sum of three input fields of the last **real** `assistant` — skipping
  `isSidechain:true` and `model:"<synthetic>"` records; codex =
  `payload.info.last_token_usage.input_tokens` of the last `token_count` event; agy = none).
- Falls back gracefully to `<agent> [cwd]` whenever no count is available (agy, no
  transcript yet, parse failure) — never a broken/blank title.

## Spec

### Goal
Give the operator an at-a-glance feel for how full each agent's context window is, so
they can decide when to start a fresh session (Shift+Alt+N). The signal lives in the
zellij pane **frame title**, beside the agent name and cwd.

### Why precise transcript reads (not a line-count estimate)
The agent already records **exact token usage** in its session transcript, and `pair`
already resolves those transcript paths per pane (`pair-slug`, `pair-cmux-title.sh`). So we
read the real number instead of estimating from scrollback line count. Two consequences:
- **The multi-model denominator problem dissolves.** We display an **absolute count**
  (`970k`), not a percentage — so we never need to know whether the window is 200k or 1M,
  and no model catalog is required.
- **The line-count estimate is dropped.** Its only remaining possible job was agy
  (no token transcript); for v1 agy simply shows no number (YAGNI).

### Signal: current-context tokens, per agent
`pair` already maps each pane → its transcript (sid from `config-<tag>-<agent>.json`).
Read the **last** relevant record and compute current-context occupancy:

| Agent | Transcript path | Read |
|---|---|---|
| **claude** | `~/.claude/projects/<enc-cwd>/<sid>.jsonl` — `sid` = the **pinned `--session-id`** from `config` (unique per pane → disambiguates same-cwd sessions; survives compaction *and* `/clear` in-place, see Edge cases) | last `type=="assistant"` that is **real** — `isSidechain != true` (skip Task sub-agent records, whose usage reflects the sub-agent's smaller context) **AND `message.model != "<synthetic>"`** (skip injected/interrupt records, whose usage is 0 → would flicker the count to 0) → `message.usage` → `input_tokens + cache_creation_input_tokens + cache_read_input_tokens` |
| **codex** | `~/.codex/sessions/…/rollout-*<sid>.jsonl` (single-file; sid stable) | last record with `type=="event_msg"` & `payload.type=="token_count"` → `payload.info.last_token_usage.input_tokens` (already the full prompt; **NOT** `payload.info.total_token_usage.input_tokens`, which is cumulative-across-session ~38M) |
| **agy** | `~/.gemini/antigravity-cli/brain/<sid>/…/transcript.jsonl` | **none** — records are semantic actions; usage only lives in opaque SQLite blobs → omit the number |

The last record's input-side sum is current occupancy to within one turn's output
(negligible vs. a ~1M window). Note: codex transcripts *also* carry
`payload.info.model_context_window` (e.g. 258400), so a true codex-% is possible later —
but v1 keeps the absolute count uniform across agents (claude's transcript has no window).

### Display
Frame title becomes `<agent> (<count>) [<cwd>]`, humanized with a **pinned rule** (tests
lock it): `<1000` → exact; `1000 ≤ n < 1_000_000` → `round(n/1000)` + `k` (`397556 → 398k`,
nearest, half-up); `≥ 1_000_000` → one-decimal `M`, floor to avoid premature rollover
(`999999 → 1000k`; `1_000_000 → 1.0M`; `1_490_000 → 1.4M`). cwd keeps `bin/pair`'s existing
tilde-abbreviation. No count available → exactly today's `<agent> [<cwd>]`.

**Skip redundant renames:** the poller caches each pane's last-emitted title string and
calls `rename-pane` only when it changes (mirrors `pair-cmux-title.sh`'s `last_prefix`
guard) — avoids per-tick IPC churn during active-but-stable stretches.

### Architecture — Approach B (one per-session poller + recorded pane id)
1. **Shared pure reader (Go, DRY/PURE).** Factor `ContextTokens(agent, transcriptPath)
   (int, bool)` — pure, table-tested per agent (claude sum-of-three of the last real
   assistant; codex `last_token_usage.input_tokens` of the last `token_count` event; agy →
   `false`). **Path resolution reuses `pair-slug`'s `resolveTranscript(agent, sid, cwd)`**
   for all three agents (sid→path): claude → `<enc-cwd>/<sid>.jsonl`, codex → rollout glob,
   agy → gemini path. `sid` comes from `config` (`session_id`); `cwd` from the pane file.
   Exposed via a one-shot `pair-context <tag> <agent>` printing the humanized count (empty
   when none); tolerant — unparseable/empty → no count.
2. **Record the pane id at startup — in a DEDICATED file, not the shared config.**
   `config-<tag>-<agent>.json` already has **three concurrent writers** (`bin/pair` writes
   the claude config synchronously at launch; `pair-session-watch.sh` writes codex/agy
   asynchronously via atomic tmp+rename = **full-file replace**, up to 60s later) — a naive
   in-pane "append one line" would clobber `session_id` or be clobbered by the watcher. So
   the in-pane startup writes `{zellij_pane_id, cwd}` to a **separate, single-writer** file
   `pane-<tag>-<agent>.json` (where `$ZELLIJ_PANE_ID` is in scope, beside the existing
   startup rename). Sid still comes from `config-…` as today. *(Alt considered: no recorded
   id, discover panes via `zellij --session pair-<tag> action dump-layout` — rejected as
   more parsing for no gain.)*
3. **One unified always-on poller (ARCH-DRY) — generalize `pair-cmux-title.sh` into
   `pair-title`.** Rather than a second near-identical sibling (~80% skeleton duplication —
   pidfile, SIGHUP trap, startup grace, session-miss exit, `latest_activity()` — on the same
   cadence), **fold the meter into the existing poller** and drop its cmux gate: the poller
   becomes always-on (the zellij frame exists with or without cmux) and owns **two title
   surfaces** — the cmux workspace title *(only when `$CMUX_WORKSPACE_ID` is set, as today)*
   and the zellij **frame** title for every pane. Each active tick it loops the tag's panes
   (`pane-<tag>-*.json` → pane id + cwd; `config-…` → `session_id` for all agents), gets
   `pair-context`'s count, and renames each pane:
   `zellij --session pair-<tag> action rename-pane --pane-id <id> "<agent> (<count>) [<cwd>]"`
   (`zellij --session <name>` lets the external poller target the pane; the startup
   counterpart `main.kdl` uses the in-pane `--pane-id "$ZELLIJ_PANE_ID"` form).
   *Plan obligation for the rename:* the cmux-internal gate (`command -v cmux` whole-script
   `exit 0` at `pair-cmux-title.sh:73`) must become **block-local** alongside the
   `$CMUX_WORKSPACE_ID` gate, and every old-name reference must move in lockstep — pidfile
   `cmux-title-pid-$TAG`, the `poller_alive()` argv match, the spawn (`bin/pair:1659`), the
   existence check (`bin/pair:1588`), and both cleanup sweeps (`bin/pair:398,446`) — miss one
   → double-spawn or orphan.
4. **Refresh policy.** Tick ≈60s. Do work only when `draft-<tag>.md` **or** the agent
   transcript was touched within the last interval (user typed **or** agent produced a
   turn) — honoring "only when active" while still advancing the count after a long agent
   turn the user hasn't replied to. Idle → skip; unchanged title → skip rename (Display).
   Reuses the existing `latest_activity()` mtime model.

### Reuse vs. new
- **Reused:** transcript resolver (`pair-slug`); sid from `config-<tag>-<agent>.json`; the
  whole `pair-cmux-title.sh` poller skeleton (pidfile, SIGHUP trap, startup grace,
  session-gone exit, `latest_activity()`) — **extended in place**, not duplicated; the
  startup in-pane rename hook in `main.kdl`; `bin/pair`'s existing spawn of the poller.
- **New:** `ContextTokens` reader + `pair-context` one-shot; the meter logic + zellij
  frame-title rename + cmux-gate removal folded into the generalized poller (renamed
  `pair-title`); the dedicated `pane-<tag>-<agent>.json` startup write (pane id + cwd).

### Out of scope (YAGNI)
- Percentage display + any model→window catalog (absolute count avoids both). Note codex
  *does* carry `model_context_window`, so a codex-only true-% is a cheap later add — but v1
  keeps one uniform format across agents.
- Scrollback line-count estimation model (superseded; no agy fallback in v1).
- agy token counts (no accessible source).
- Threshold coloring / auto-nudge to new session (could follow once a coarse window guess
  is acceptable; v1 is just the number).

### Edge cases & risks
- **Same-cwd disambiguation (the load-bearing case — operator runs multiple sessions per
  cwd routinely).** Solved by the **pinned `--session-id`**: `bin/pair` (#000020) generates a
  fresh uuid per new-session, passes `--session-id <sid>`, and writes `config` synchronously,
  so each pane has a unique, known sid and reads exactly `<enc-cwd>/<sid>.jsonl` — no
  aliasing among co-located sessions. (An earlier draft used newest-`*.jsonl`-by-mtime; that
  was a regression — the project dir is keyed by cwd only, so it would alias this exact case.
  Reverted to the pinned sid.)
- **`/clear` / compaction — resolved by data, NOT a problem.** Empirically (analyzing
  `~/.claude/projects/.../13256418*.jsonl`): context-reduction events keep writing to the
  **same pinned file** — `isCompactSummary:true` compaction (998k→47k) continues in-file, and
  after a reset-to-0 the context climbed back to ~989k *within the same jsonl* (~500 turns).
  So the pinned sid file is always the live file; its last record reflects current (post
  compact/clear) context. The `pair-cmux-title.sh:124–126` comment claiming `/clear` "rotates
  the file" is contradicted by the data (predates sid-pinning, or describes unpinned claude).
  *Plan should still smoke-test one real `/clear` in a pinned session to confirm.*
- **`<synthetic>` / sidechain records mislead the read** — interrupt/injected records carry
  `model:"<synthetic>"` with `usage`≈0 (would flicker the count to 0), and `Task` sub-agent
  records carry `isSidechain:true` with the sub-agent's smaller context (would undercount).
  Take the last assistant record that is **neither** (in Signal). Fixture pins both.
- **Config write race (3 writers)** — addressed by the dedicated single-writer
  `pane-<tag>-<agent>.json` (Architecture step 2); the plan must not touch `config-…`.
  The plan also adds `pane-<tag>-*.json` to `bin/pair`'s per-tag cleanup sweep, and the
  poller must tolerate a not-yet-written pane file on the create-path race (skip that pane's
  frame rename for the tick, retry next — mirrors `latest_activity()==0` handling).
- **Transcript envelope is undocumented/versioned** — the `usage`/`token_count` *payloads*
  are stable public API shapes, but the record wrapper can drift across CC/codex versions.
  Keep the reader tolerant (skip unparseable records, fall back to no-count).
- **One agent instance per (tag) assumption** — keys are `(tag, agent)`; two panes of the
  same agent in one tag would collide. Confirm pair's invariant in the plan.
- **agy** intentionally shows no number — verify the fallback title is identical to today's.

### Testing
- Unit: `ContextTokens` against committed fixtures — a claude jsonl tail (incl. an
  `isSidechain:true` record before a real one, to pin the filter), a codex rollout tail
  (a real `token_count` `event_msg` with both `last_token_usage` and `total_token_usage`,
  to pin last-not-total), and an agy transcript → empty. Plus humanization table
  (`397556→398k`, `999999→1000k`, `1_000_000→1.0M`, `1_490_000→1.4M` to pin the M-branch floor).
- Process-level: drive the poller against a temp `pane-<tag>-*.json` + `config-…` + fixture
  transcripts + a fake `zellij` shim capturing `rename-pane` args; assert the title string,
  the activity-gate (idle → no rename; touched → rename), and the unchanged-title skip.

## Plan

- [ ]

## Log

### 2026-06-25

Created by migrating ariadne#131 (misfiled — feature is 100% pair code). Spec is the product
of a brainstorm + 3 spec-review rounds in ariadne#131; key conclusions carried here:
- **Signal = precise transcript token usage**, not a scrollback line-count estimate.
- **Display = absolute humanized count** in the zellij frame → no model→window catalog.
- **claude:** pinned `--session-id` (from `config`) is the exact per-pane key — disambiguates
  same-cwd sessions (operator's routine case) and survives `/clear`/compaction in-place
  (data-proven against real transcripts; refutes the stale `pair-cmux-title.sh` "rotates"
  comment). Read the last assistant record that is `isSidechain != true` AND
  `model != "<synthetic>"`.
- **codex:** `payload.info.last_token_usage.input_tokens` of the last `token_count` event
  (NOT cumulative `total_token_usage`). **agy:** no accessible token source → omit.
- **Architecture:** pure `ContextTokens` reader + `pair-context` one-shot; generalize
  `pair-cmux-title.sh` into one always-on `pair-title` poller owning both title surfaces;
  pane id+cwd in a dedicated `pane-<tag>-<agent>.json` (dodges the 3-writer config race).
- Full review history + the lesson it produced ("verify behavioral claims vs runtime data
  before redesigning") are in ariadne#131 + ariadne/workshop/lessons.md.

Durable plan written → `workshop/plans/000071-context-meter-pane-frame-plan.md`.

Plan review (fresh context; reviewer **compiled + ran** the Go core against a real 7.27MB
transcript — `ContextTokens`→536887 matched `jq`, codex→60287, all `Humanize` vectors, valid
`main.kdl` JSON: pure core verified correct). Shell/refactor fixes folded in:
- Task 1: deleting `sessionID` orphans the `encoding/json` import → must drop it (build break).
- Task 7: lines 398/446 are `rename_paths_for` (the `pair rename` cmd), NOT a session sweep;
  pane files are per-(tag,agent) like config → carry them in the `for a` loop + a `case` arm
  (no reaper; they persist + overwrite like config).
- Task 7: test-hook fires before `DATA_DIR`/`SESSION` assigned → reorder; fake-`zellij` arg
  index is `$4` not `$3`; meter must refresh OUTSIDE the cmux bucket-change guard (else
  refreshes once then freezes); grep-gate also catches comment sites 870/1651.
- Task 5: add `transcript.go` prereq to the pair-slug Makefile rule; `gofmt -w` before commit.
Plan review converged (1 round + fixes); Go core independently verified.

Next: set `estimate_hours` → `sdlc change-code` → execute (subagent-driven per AGENTS.md §3).
