---
id: '000020'
status: done
created: 2026-05-10
updated: 2026-05-27
actual_hours: N/A
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
