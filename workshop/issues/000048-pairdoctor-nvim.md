---
id: 000048
status: done
deps: []
github_issue:
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

## Spec

- **ARCH-PURE split** (the pure payload vs the IO seam):
  - **Pure:** extract a `dofile`-able module `nvim/doctor.lua` exposing
    `M.payload(pair_home)` → the resolved-path instruction *string*, or `nil`
    when `pair_home` is empty/unset (the guard sentinel). No IO, no `vim.*`
    side effects at load — mirrors `nvim/slug.lua` / `nvim/adapt.lua`, the only
    headless-testable shape (`init.lua` is a 3450-line file with load-time side
    effects and a `local send_to_agent`, so it is NOT `dofile`-able).
  - **IO seam:** `:PairDoctor` in `init.lua` is a thin caller —
    `local p = require/dofile doctor.payload(vim.env.PAIR_HOME); if not p then
    vim.notify('PairDoctor: PAIR_HOME unset', ERROR) else send_to_agent(p) end`.
- **`:PairDoctor` user command** in `nvim/init.lua` (vim user commands must start
  uppercase). Optionally also expose a `PairDoctor()` global to match the
  `:lua PairConfirmRestart()` pattern.
- **Auto-send** (decision): the thin caller passes the payload to
  `send_to_agent(payload)` directly — no draft insertion. `send_to_agent` already
  handles the bracketed-paste / timing concerns (`init.lua:696–712`).
- **Thin-pointer payload, `$PAIR_HOME`-absolute** (procedure stays single-sourced
  in `doctor/SKILL.md` — DRY, no duplication). `payload()` substitutes the
  *resolved* `pair_home` value into the literal text (don't rely on the agent's
  shell to expand `$PAIR_HOME`). Shape:

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
- `PAIR_HOME` unset ⇒ `payload()` returns nil ⇒ the caller notifies, no broken send.
- Docs (`doctor/README.md`, `atlas/index.md`) name `:PairDoctor` the primary entry.
- A headless Lua test (`nvim -l nvim/doctor_test.lua`) `dofile`s `nvim/doctor.lua`
  and asserts `payload(home)` contains `<home>/doctor/doctor.sh` + the `SKILL.md`
  ref, and `payload(nil)`/`payload('')` returns nil; wired into `make test-lua`.

## Plan

- [x] Extract pure `nvim/doctor.lua` with `M.payload(pair_home)` → instruction string (absolute paths) or nil when unset (mirrors `slug.lua`/`adapt.lua`; no load-time IO).
- [x] Add `:PairDoctor` command + `PairDoctor()` global in `nvim/init.lua` as the thin IO seam: `payload(vim.env.PAIR_HOME)` → nil ? notify : `send_to_agent`.
- [x] Reframe `doctor/README.md` (→ "Entry points") + `atlas/index.md` to present `:PairDoctor` as the primary agent-agnostic entry (skill-install = optional).
- [x] Headless test `nvim/doctor_test.lua` (`dofile doctor.lua`): absolute `doctor.sh` path + `SKILL.md` ref + no-literal-`$PAIR_HOME` + trailing-slash + nil-on-unset; added `nvim -l nvim/doctor_test.lua` to `test-lua` (`Makefile.local`).

## Revisions

### 2026-06-03 — plan-quality FAILURE → ARCH-PURE restructure (sdlc change-code)
The original Plan tested by "capturing `send_to_agent` inside `init.lua`", which
isn't `dofile`-able (3450-line IO file, load-time side effects, `local`
function) — no existing nvim test loads `init.lua`; they all `dofile` a small
extracted module. Restructured per `ARCH-PURE`: the payload is a pure function of
`pair_home` → extract `nvim/doctor.lua::payload`, leave `init.lua` a thin
caller, and test the pure module via `nvim -l` like `slug_test.lua`.

## Log

### 2026-06-03
- 2026-06-03: closed — nvim/doctor_test.lua 8/8 (absolute paths, no literal $PAIR_HOME, trailing-slash trim, nil-on-unset); make test-lua 4/4 suites green; headless init.lua registers :PairDoctor (exists=2) + _G.PairDoctor fn + unset-PAIR_HOME notifies without crash; payload preview is a clean $PAIR_HOME-absolute single-line instruction; review verdict: SHIP
- Filed from the design discussion following #000046/#000047. Core insight: a
  Claude-skill entry couples pair-doctor to one agent and breaks under codex/agy/
  vanilla; the cwd-fragility (`doctor/doctor.sh` only resolving in the pair repo)
  compounds it. nvim is the always-present, `$PAIR_HOME`-aware substrate, so
  `:PairDoctor` → `send_to_agent` is the agent-agnostic, path-correct entry.
- Decisions: auto-send (operator); thin-pointer payload (DRY vs `doctor/SKILL.md`);
  command name `:PairDoctor` (vim uppercase requirement).
- Implemented per the ARCH-PURE restructure: pure `nvim/doctor.lua::payload` +
  thin `:PairDoctor`/`_G.PairDoctor` IO seam in `init.lua` (loads doctor.lua like
  slug.lua; `payload(vim.env.PAIR_HOME)` → nil ? `vim.notify` : `send_to_agent`).
- Verified: `nvim/doctor_test.lua` 8/8 (absolute paths, no literal `$PAIR_HOME`,
  trailing-slash trim, nil-on-unset); `make test-lua` 4/4 suites green; headless
  `init.lua` load registers `:PairDoctor` (exists=2), `_G.PairDoctor` is a fn, and
  the unset-`PAIR_HOME` path notifies without crashing. Payload preview confirms a
  clean single-line `$PAIR_HOME`-absolute instruction.
