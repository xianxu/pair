---
issue: 16
title: tag-restart — tags as durable session identities
status: working
---

# Tag-restart: tags as durable session identities

## Spec

Today a `pair` tag is the zellij session name (`pair-<tag>`); it survives
detach (`Alt+d`) but dies on full-quit (`Alt+x`). The agent's own session
(claude/codex/gemini) is captured nowhere, so a fresh `pair claude` after
`Alt+x` starts the agent with no continuity — the user has to remember the
agent's session id and the original startup args, and re-pass both.

This issue makes a tag a durable name for a coding session: launch args
plus agent session id are recorded per (tag, agent) pair on disk, and the
new-session create flow surfaces them as a one-prompt restart.

### Discovery: agent session id

Each agent prints its own session id at startup; the discovery mechanism
differs:

- **claude.** Writes `~/.claude/projects/<encoded-cwd>/<session-id>.jsonl`
  on session start (encoded-cwd = `pwd | tr / -`). Filename is the id.
- **codex.** Writes `~/.codex/sessions/<YYYY>/<MM>/<DD>/rollout-<ts>-<session-id>.jsonl`.
  Id is the trailing UUID in the filename.
- **gemini.** Writes `~/.gemini/tmp/<project>/chats/session-<ts>-<short>.json`,
  where `<short>` is the first 8 chars of the session id and the full id
  lives inside the JSON under `"sessionId"`. Resume requires the full id,
  so the watcher reads the JSON, not just the filename.

For all three, the watcher mechanic is the same: snapshot existing session
files at launch, then poll the agent's session dir at ~100ms and pick the
first new file. No `fswatch` dep — polling is cheap and stops on first hit.
Claude is implemented first; codex/gemini follow once claude is solid.

### Storage

`$PAIR_DATA_DIR/config-<tag>-<agent>.json`:

```json
{
  "agent": "claude",
  "args": ["--dangerously-skip-permissions"],
  "session_id": "8d745d08-4ecc-4474-969a-53c98a6fa5f0"
}
```

Single write path: when the watcher captures the session id, it writes
the full file (agent + args + session_id) atomically. No partial states.
Args come from the create flow that spawned the watcher, passed via env
or argv to the watcher process.

The file is keyed by `(tag, agent)` so the same tag can hold separate
configs for claude vs codex; matches the existing `agent-<tag>` per-tag
agent-tracking shape in `bin/pair`.

### Stale-id check

Before offering the "use params + session" option, verify the session
file still exists:

- claude: `[ -f ~/.claude/projects/<encoded-cwd>/<id>.jsonl ]`
- codex: `find ~/.codex/sessions -name "*<id>*.jsonl" | head -1`
- gemini: `rg -l "\"sessionId\": \"<id>\"" ~/.gemini/tmp/`

Cheap, no external deps beyond what pair already requires. If stale, drop
the resume option silently and the prompt collapses to "use params /
none".

### Prompt UX

After the user picks/types a tag in the new-session create flow and
before the zellij `new-session` exec, if `config-<tag>-<agent>.json`
exists, fzf-prompt with three options. Each option line shows what it
will do with the saved values inline so the user sees what they're
picking without flipping context:

```
Saved config for tag "<tag>":
  use params + session  →  args=[--dangerously-skip-permissions]  resume=8d745d08
  use params            →  args=[--dangerously-skip-permissions]  fresh session
  use none              →  args=[<whatever was passed on this command>]  fresh session
```

If the saved session_id is stale or absent, the first row is omitted.
Selecting "use none" deletes the saved file before proceeding (clean
overwrite, the next watcher run rewrites it).

fzf is already a hard dep (used in the session picker), so reusing it
keeps the UI consistent. Plain `read -n1` would work too but the
multi-line "shows the values" requirement makes fzf the natural fit.

## Plan

- [x] **M1 — claude session-id watcher.** New `bin/pair-session-watch.sh`,
      backgrounded from `bin/pair` on the new-session path. Polls the
      claude projects dir at 100ms; writes `config-<tag>-claude.json`
      atomically on first new file; exits silently after 60s if nothing
      appears. Non-claude agents are no-ops at this milestone.
- [x] **M2 — create-flow prompt.** `bin/pair` reads
      `config-<tag>-<agent>.json` after tag commit, runs the per-agent
      stale-id check (claude only at this milestone), fzf-prompts with
      the inline-value options, and composes the final agent_extra.
      "use params + session" strips any pre-existing `--resume <X>` from
      saved args before appending `--resume <session_id>`. ESC aborts
      the create flow.
- [ ] **M3 — codex + gemini watchers.** Add per-agent discovery
      (filename for codex, JSON-content scan for gemini). Same storage
      shape, same prompt code path — just per-agent stale-check.
- [ ] **M4 — atlas + README.** Document tag-restart flow in
      `atlas/architecture.md`; add a short note in README's "session
      names" section.

## Log

### M1 — 2026-05-05

- `bin/pair-session-watch.sh` created. Snapshots existing `*.jsonl` under
  `~/.claude/projects/<encoded-cwd>/`, polls every 100ms for a new file,
  writes JSON atomically via jq + mktemp + mv. 60s deadline.
- bash 3.2 compatibility: `set -u` plus empty array required the
  `${args[@]+"${args[@]}"}` expansion guard.
- Verified in isolation against a mock projects dir:
  - new-file detection (post-snapshot create) → captures id
  - empty agent args → `"args": []`
  - args containing spaces → preserved correctly via `jq --args`
  - non-claude agents → silent exit 0
- Wired into `bin/pair` right before the zellij `--new-session-with-layout`
  exec, backgrounded with stdio redirected to `/dev/null`. word-split on
  `$agent_extra` matches the existing `$PAIR_AGENT_ARGS` convention.
- Manual end-to-end with a real claude session not exercised here (would
  disrupt the pair session this work is happening in); deferred to M2's
  end when the full read-back path exists.

### M2 — 2026-05-05

- Tag-restart prompt block added to `bin/pair` between tag-commit and the
  watcher spawn. fzf with `--header` for the heading and three numbered
  options whose lines spell out the values to be applied (per the user's
  ask: show what each selection will do inline).
- Stale-id check is per-agent: claude verifies
  `~/.claude/projects/<encoded-cwd>/<id>.jsonl` exists. If absent the
  "use params + session" row is silently dropped, leaving "use params /
  use none".
- "use params + session" strips any pre-existing `--resume <X>` from the
  saved args before appending `--resume <session_id>`, so the composed
  command line never carries a duplicate `--resume`.
- Tested unit-style against fixtures:
  - all option-building paths (valid / stale-cwd / missing-file /
    empty-args)
  - all choice-application paths (option 1 with/without prior `--resume`
    in args; option 2 with empty args)
- bash 3.2 compatibility: `agent_extra="${saved_args[*]:-}"` (empty-array
  guard) and `${stripped[*]:+...}` (conditional separator) for cases
  that interact with `set -u`.
- Full end-to-end verification still needs a live `pair claude` run by
  the user — can't be exercised from inside the current pair session.
