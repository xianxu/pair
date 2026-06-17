---
id: 000063
status: done
deps: []
github_issue:
created: 2026-06-16
updated: 2026-06-17
estimate_hours: 2
actual_hours: 0.66
---

# key the changelog on session_id so a fresh session starts a fresh changelog

## Problem

The change log (#53, `Alt+l`) is keyed on `changelog-<tag>-<agent>`. A pair tag
outlives individual coding sessions — `Alt+Shift+N` restarts a **fresh
conversation** under the same tag — so the per-tag log accretes every session
forever, with no way to start from scratch.

Mechanism (verified): the distiller anchors incrementally on the **last 3
verbatim cleaned scrollback lines** + a turn count (`cmd/pair-changelog/main.go:148`,
`anchorSnippet`). The scrollback (`scrollback-<tag>-<agent>.raw`) is per-*launch*
— O_TRUNC'd on every start, including a resume. On a fresh session the new
transcript doesn't contain the old anchor → `locate` returns **`FullRedistill`**
(`distill.go:187`) → the whole new conversation is re-fed against the existing
per-tag log and appended (leaning on the model to dedup). That's the pile-up.

## Spec

Key the changelog's whole file set on the **persisted agent `session_id`**, not
just `<tag>-<agent>`:

```
changelog-<tag>-<agent>-<session_id>.{md,anchor,cleaned,openlock,distill.lock,status,ready}
```

`session_id` already has exactly the lifetime we want: pair mints a new one on a
fresh start (`new_session`, no resume) and reuses it on resume (Alt+n hands the
agent `--resume <id>` / `resume <id>` / `--conversation <id>` — `bin/pair:782-784`;
persisted in the per-tag config JSON, `bin/pair:2011,2077`). So:

- Fresh session → new `session_id` → new **empty** changelog (starts from scratch).
- Resume → same `session_id` → the same changelog keeps accreting (correct for an
  extended session).
- Each session gets its own `.anchor`, so a fresh session starts anchorless = a
  clean first-run distill of just that session — which *also* removes the
  cross-session `FullRedistill` conflation above.

The key **is** the reset — no pointer file, no archive-on-mismatch, no separate
"epoch" concept.

**Plumbing:** `bin/pair-changelog-open` only gets `PAIR_TAG/PAIR_AGENT/PAIR_DATA_DIR/
PAIR_HOME` today. Export `PAIR_SESSION_ID` from `bin/pair` (it already reads
`.session_id` from the config for resume), and thread it into both the opener's
base path **and** the draft-nvim `.ready` watcher (`nvim/init.lua`
`pair_start_changelog_ready_watch`, which builds `changelog-<tag>-<agent>.ready`).
The long uuid can be truncated/hashed for the filename (cosmetic).

**Rejected alternatives:** (1) couple the changelog to the *scrollback* lifetime —
scrollback is per-launch (wiped on every restart incl. resume), so it would reset
on a continuing Alt+n; too granular. (2) a pair-minted "epoch" token — duplicates
`session_id`, which already exists with the right turnover.

**Accepted caveats:** `/clear` rotates claude's *live* session_id mid-session but
pair keeps the launch-time id (atlas:479 gap), so `/clear` won't start a fresh
changelog — fine as a default. And one `changelog-…-<sid>.md` per conversation now
accumulates on disk instead of one ever-growing file (strictly better for growth);
reaping the old per-session files is a noted follow-up, not in scope.

## Done when

- Changelog files (log + anchor + cleaned + locks/status/ready) are keyed on the
  persisted `session_id`: a fresh session (Alt+Shift+N) opens an **empty**
  changelog; a resume (Alt+n) reopens the **same growing** one.
- `pair-changelog-open` resolves the session_id (exported `PAIR_SESSION_ID` or the
  config) and builds the per-session base path; the draft-nvim `.ready` watcher
  matches the same path.
- Changelog tests (`cmd/pair-changelog/...`, `tests/changelog-*`) green; a test
  covers fresh-vs-resume keying.

## Plan

- [x] Export `PAIR_SESSION_ID` to the session env from `bin/pair` (reuse the
      `.session_id` it already reads from the per-tag config).
- [x] Key the changelog base on `<tag>-<agent>-<session_id>` (full uuid — see Log,
      truncation declined) in `bin/pair-changelog-open`; update the draft-nvim
      `.ready` watcher path (`nvim/init.lua` `pair_start_changelog_ready_watch`).
- [x] Tests: fresh → new/empty changelog, resume → same changelog; update the
      e2e/render tests for the new base.
- [x] Atlas: note the per-session keying in the Change-log section of
      `atlas/architecture.md`.

## Log

### 2026-06-16
- Filed from an in-session brainstorm. Verified the anchor / `FullRedistill`
  mechanism (content-based 3-line anchor + turn count; scrollback is per-launch,
  so a reconstructed transcript misses the anchor → re-feeds the whole
  conversation → cross-session append). Landed on `session_id` as the key (it
  already turns over on fresh-vs-resume); rejected coupling-to-scrollback (too
  granular) and a parallel epoch token (duplicates session_id). Old-per-session-
  file reaping deferred. **Not started.**

### 2026-06-17 — implemented (durable plan: `workshop/plans/000063-…-plan.md`)
- 2026-06-17: closed — Fresh session opens an empty change log; resume reopens the same growing one — pair-changelog-open and the draft-nvim .ready watcher both resolve session_id (PAIR_SESSION_ID -> per-tag config -> legacy unsuffixed) to the keyed base. Full `make test` green, incl. new tests/changelog-session-key-test.sh (fresh/resume/config-fallback/legacy) and extended changelog-notify-test.sh driving the watcher legacy/env/config branches.; review verdict: SHIP
- **Key discovery beyond the spec:** session_id is known at `bin/pair` launch
  **only** for claude-fresh (minted via `uuidgen --session-id`) and **any resume**
  (`--resume <id>`, incl. the Alt+n restart re-exec). For a **codex/agy fresh
  session there is no `--session-id` flag** — the id is discovered *async* by
  `pair-session-watch.sh` and written to the config *after* zellij/nvim start. So
  the per-tag **config is the canonical source** (ARCH-DRY, reusing the existing
  `jq -r '.session_id // empty'` idiom); `PAIR_SESSION_ID` is a launch-time fast
  path. Both consumers resolve `PAIR_SESSION_ID → config → none`; the nvim watcher
  re-resolves each tick so a late codex/agy id is picked up without a restart.
- **Truncation declined (spec said "truncate/hash the uuid").** All three agents'
  ids are path-safe `8-4-4-4-12` hex uuids (~36 chars; total path well under any
  limit). Truncating would introduce a (tiny) collision risk that doesn't exist
  today for a cosmetic gain on a non-user-facing data-dir filename — Root-Cause /
  Simplicity-First. Kept the **full uuid**; no truncation logic to duplicate
  across the shell + Lua builders.
- **Backward compat:** no id ⇒ the legacy unsuffixed base `changelog-<tag>-<agent>`,
  so pre-existing logs/tests stay valid. Accepted caveat: the first resume after
  this orphans any pre-#63 per-tag log (harmless — that's the accreting pile #63
  targets; reaping deferred).
- **Tests:** new `tests/changelog-session-key-test.sh` (fresh/resume/config-fallback/
  legacy via a fake nvim) under `make test-changelog`; extended
  `tests/changelog-notify-test.sh` to drive the watcher's legacy/env/config branches
  in one boot via `vim.env` mutation (closes the config-fallback coverage gap the
  plan-quality judge flagged). Full `make test` green.
- **Reviews:** plan-quality judge `VERDICT: INFO` (high confidence, all claims
  verified); fresh-eyes plan review **Approved**.
