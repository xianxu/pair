---
id: 000020
status: working
deps: []
created: 2026-05-10
updated: 2026-05-10
related: [bin/pair, bin/pair-session-watch.sh, cmd/pair-wrap/main.go]
---

# Bind agent session id to the pair tag deterministically

## Problem

`bin/pair-session-watch.sh` discovers an agent's session id by snapshotting
`~/.claude/projects/<encoded-cwd>/` (or codex/gemini equivalents) and
grabbing the first new `*.jsonl` to appear. The watch dir is keyed by
**cwd**, not by tag, so two pair sessions launched in the same repo race
for whichever session file is created first.

Failure modes observed:

- `config-2-claude.json` (tag `2`) and `config-pair-claude.json` (tag
  `pair`) ended up pointing at the same `session_id`. Whichever watcher
  snapshotted before the *other* tag's claude wrote its file claimed the
  wrong id.
- Once a wrong id is on disk, `pair resume <tag>` faithfully restores the
  *other* tag's conversation. The watcher never auto-corrects — the
  config sticks until manually wiped.

## Spec

Two-track fix, picked per agent's CLI surface.

**Claude** supports `--session-id <uuid>`. We pre-generate a v4 UUID in
`bin/pair`, pass it to claude on the new-session path, and write
`config-<tag>-claude.json` synchronously before zellij launch. No
watcher, no race. Defensive: if claude rejects the UUID as already in
use (collision astronomically unlikely with v4, but possible on shared
session dirs), retry with a fresh UUID.

**Codex/Gemini** have no equivalent flag. We bind discovery to the
agent's PID via `lsof`:

1. `cmd/pair-wrap/main.go` writes its child's PID to
   `$PAIR_DATA_DIR/agent-pid-<tag>` immediately after `pty.Start`.
2. `bin/pair-session-watch.sh` waits for that pidfile, then polls
   `lsof -p <pid> -Fn` for a path under the agent's known session dir
   matching the agent's filename pattern.
3. Extract id (codex: trailing UUID in filename; gemini: `sessionId`
   from JSON body). Atomic write to `config-<tag>-<agent>.json`.

Race-free across concurrent sessions in the same cwd because lsof
output is scoped to a specific PID.

## Plan

- [x] M1: Issue file.
- [x] M2: `cmd/pair-wrap/main.go` — `agentPIDPath` field + write `agent-pid-<tag>` after `pty.Start`, remove on shutdown.
- [x] M3: `bin/pair` — generate UUID for claude new-session path, inject `--session-id`, write config synchronously. Guards against existing `--session-id` / `--fork-session` and existing-jsonl collisions.
- [x] M4: `bin/pair-session-watch.sh` — claude becomes no-op; codex/gemini use lsof against agent PID tree, with birth-time-filtered single-candidate fallback when lsof can't see the file.
- [x] M6: Update atlas/architecture.md — refresh "Discovery — three layers" + per-agent table + storage section.
- [ ] M5: Verify — relaunch with the new code, start two pair sessions in same cwd (claude × claude), confirm distinct session ids. Test codex/gemini at least once each.

## Log

**2026-05-10 — implementation.**

- Confirmed `claude --help` exposes `--session-id <uuid>`; codex and gemini do not. Confirmed claude refuses already-existing UUIDs (`Error: Session ID ... is already in use`), so the launcher retries up to 5 fresh UUIDs against `~/.claude/projects/<encoded-cwd>/<id>.jsonl`. v4 collisions are astronomically unlikely; the retry is just a defense-in-depth.
- Investigation of running claude PIDs (`56453`, `65650`) showed claude does **not** keep its session jsonl open via lsof. Codex/gemini behavior could not be verified live (no running instances + neither launches cleanly without a real TTY). To avoid regressing them, the watcher uses lsof first and falls back to a birth-time-filtered directory walk: candidate files must have `stat -f %B >= mtime(agent-pid-<tag>)`, eliminating files from earlier pair sessions. The fallback only commits when there's exactly one candidate — a multi-candidate state means a true concurrent race, and refusing is safer than guessing wrong.
- pair-wrap writes the agent PID to `$PAIR_DATA_DIR/agent-pid-<tag>` next to the existing `pair-wrap-pid-<tag>` (capture-arm) file, using the same path-resolution and shutdown-removal pattern.

**2026-05-10 — codex/gemini regression in the rewritten watcher.**

User flagged that codex/gemini capture stopped working. Root cause: the rewritten watcher waited unconditionally for `agent-pid-<tag>` to appear, but the *installed* `bin/pair-wrap` binary (built before this change) doesn't write that file. The watcher hit its 10s timeout and exited without capturing.

Fix: cap the pidfile wait at 2s and degrade gracefully when none appears — fall through to the pre-#000020 snapshot-diff loop. Old pair-wrap installs continue to capture sessions in the common single-pair case; rebuilding pair-wrap upgrades to the PID-bound path. Tested both branches end-to-end with synthetic codex session files; both capture correctly. Picker dedup (saved == current → collapse "use new" rows) added the same session in response to user feedback on the picker UI.

