---
id: '000048'
status: done
created: 2026-06-03
updated: 2026-06-03
estimate_hours: 1.5
actual_hours: 1.5
---

# PairDoctor nvim command — agent-agnostic doctor entry

## Problem

pair-doctor's entry points today are (a) running `doctor/doctor.sh` by hand and
(b) the unregistered `doctor/SKILL.md` Claude skill. Both are wrong as the
*primary* entry:

- **Claude-only.** `.claude/skills/` is a Claude Code feature, so the skill path
  only works when the agent is claude — it vanishes under codex/agy/vanilla,
  contradicting pair's agent-agnostic premise.
- **cwd-fragile.** pair runs in arbitrary project dirs; the agent's cwd is the
  user's project, not the pair checkout. A static `bash doctor/doctor.sh` only
  resolves when cwd happens to be the pair repo. The doctor is inherently
  pair-repo-scoped (reads `$PAIR_DATA_DIR` logs, points at `cmd/pair-wrap`
  source), so it needs `$PAIR_HOME`-absolute references to run from anywhere.

nvim is the one substrate present under *every* agent, and it knows `$PAIR_HOME`
(`vim.env.PAIR_HOME`, exported by `bin/pair`). A `:PairDoctor` command can build a
correct, absolute-pathed instruction and hand it to whatever agent is running via
the existing send mechanism (`nvim/init.lua:696 send_to_agent`). Related: #000046
(pair-dev), #000047 (stale-binary probe).
