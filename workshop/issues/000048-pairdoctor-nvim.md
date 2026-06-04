---
id: 000048
status: open
deps: []
github_issue:
created: 2026-06-03
updated: 2026-06-03
estimate_hours: 1.5
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

## Spec

- **`:PairDoctor` user command** in pair's nvim (vim user commands must start
  uppercase). Define near the other entry points; optionally also expose a
  `PairDoctor()` global to match the `:lua PairConfirmRestart()` pattern.
- **Auto-send** (decision): on invoke it calls `send_to_agent(payload)` directly —
  no draft insertion. `send_to_agent` already handles the bracketed-paste / timing
  concerns (`init.lua:696–712`).
- **Thin-pointer payload, `$PAIR_HOME`-absolute** (procedure stays single-sourced
  in `doctor/SKILL.md` — DRY, no duplication). nvim substitutes the *resolved*
  `vim.env.PAIR_HOME` value into the literal text (don't rely on the agent's shell
  to expand `$PAIR_HOME`). Shape:

  > Run `bash <PAIR_HOME>/doctor/doctor.sh` (reads this session's adaptation
  > flight recorder), then follow `<PAIR_HOME>/doctor/SKILL.md`: interpret the
  > drift findings against `<PAIR_HOME>/atlas/how-to-bring-up-a-new-harness-cli.md`
  > §3 and propose concrete matcher fixes for me to approve. Don't edit silently.

  Degrade gracefully if `PAIR_HOME` is unset (notify, don't send a broken path).
- **Reframe the docs:** `doctor/README.md` "As a skill" note and `atlas/index.md`
  should present `:PairDoctor` as the *primary*, agent-agnostic entry; the
  `.claude/skills/` registration becomes an optional Claude-only convenience that
  reuses the same `SKILL.md`.

Out of scope: the stale-binary probe (#000047); actually registering the Claude
skill (optional, separate); insert-into-draft mode (auto-send chosen).

## Done when

- `:PairDoctor` exists in pair's nvim and auto-sends the doctor instruction to the
  agent via `send_to_agent`.
- The payload carries `$PAIR_HOME`-absolute paths (works from any cwd, any agent).
- It's a thin pointer — defers the procedure to `doctor/SKILL.md`, no duplication.
- `PAIR_HOME` unset ⇒ a notify, not a broken send.
- Docs (`doctor/README.md`, `atlas/index.md`) name `:PairDoctor` the primary entry.
- A headless Lua test (`nvim -l`) asserts the payload (absolute doctor.sh path +
  SKILL.md ref) and that `send_to_agent` is called; wired into `make test-lua`.

## Plan

- [ ] Add `:PairDoctor` command (+ optional `PairDoctor()` global) in `nvim/init.lua`: build the thin payload with resolved `vim.env.PAIR_HOME`, auto-send via `send_to_agent`; guard unset `PAIR_HOME`.
- [ ] Reframe `doctor/README.md` "As a skill" + `atlas/index.md` to present `:PairDoctor` as the primary agent-agnostic entry (skill-install = optional).
- [ ] Headless Lua test asserting payload construction + `send_to_agent` call (mock/capture send); wire into `make test-lua`.

## Log

### 2026-06-03
- Filed from the design discussion following #000046/#000047. Core insight: a
  Claude-skill entry couples pair-doctor to one agent and breaks under codex/agy/
  vanilla; the cwd-fragility (`doctor/doctor.sh` only resolving in the pair repo)
  compounds it. nvim is the always-present, `$PAIR_HOME`-aware substrate, so
  `:PairDoctor` → `send_to_agent` is the agent-agnostic, path-correct entry.
- Decisions: auto-send (operator); thin-pointer payload (DRY vs `doctor/SKILL.md`);
  command name `:PairDoctor` (vim uppercase requirement).
