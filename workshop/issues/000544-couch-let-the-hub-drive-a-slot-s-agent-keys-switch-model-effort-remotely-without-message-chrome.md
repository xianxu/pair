---
id: 000544
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-14
estimate_hours:
card_mirror: '966620b85b400d58c7b4ed982c8eb5d7047624c3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T01:55:19-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# couch: let the hub drive a slot's agent keys (switch model/effort remotely), without message chrome

## Problem

The operator wants the hub (TL) to switch a slot's model and effort remotely. Every agent does it through an interactive picker (`/model`, then arrow keys and Enter; Codex adds an effort step; Claude and others differ). `couch --send-to` can't do it: it delivers a prompt wrapped in peer-message chrome ("[Couch peer from …]"), and can't send arrow keys. `couch --switch-agent` changes the agent, not its model.

A manual drive of idle pair:6 (Codex) on 2026-10-13 worked, from the hub, with zellij and `couch --peek`:
- Target: the slot's zellij session (`ZELLIJ_SESSION_NAME` from the wrapper process's environment; wrapper PID in `pair-wrap-pid-<tag>`) and the agent pane (`pane_id` in `pane-<tag>-<agent>.json` → `terminal_<id>`).
- `zellij --session S action write-chars --pane-id terminal_N '/model'` typed into the agent's input box.
- **Plain Enter (CR) inserts a newline in the agent pane** (Pair's deliberate mapping, pair#162); **Alt+Enter (ESC CR) submits**. Inside the agent's picker, plain Enter selects.
- Arrows `ESC [ A` / `ESC [ B` moved the picker; a lone ESC (sent separately) backs out; Ctrl+U clears the input line.
- `couch --peek` between steps showed the picker state (the highlighted row is reverse video), so each step can be verified.

## Spec

Two layers, hub-only, idle slots only (refuse when the agent is busy or the agent’s composer has unsent text):
1. **`couch --keys repo:N KEYSPEC…`**: inject named keys into the slot's agent pane, with no chrome: `text:…`, `enter` (plain CR), `submit` (Alt+Enter), `up`, `down`, `esc`, `clear-line`, `wait:MS`. Couch resolves the session and pane itself (no env scraping), writes through the slot's terminal, and returns the agent pane's tail after the last key, like `--peek`. One request, one bounded sequence; no retries.
2. **`couch --set-model repo:N --model NAME [--effort LEVEL] --confirm`**: per-agent recipes built on (1) that open the picker, find the row by its visible label in the peeked screen (not by position), select it, handle the effort step where the agent has one, and verify the result from the agent's own status line (e.g. Codex's footer `GPT-6-Astra medium`). If the label isn't found, back out with ESC and report the visible choices; never guess.

Agents' pickers differ and change across versions; recipes live in one table per agent, tested against recorded screens. The slot-continuity target's spirit applies: one action, an honest result, no limbo (a failed drive backs out to the agent's prompt).

## Done when

- `couch --keys` drives an idle scratch slot's agent pane (text, submit, arrows, esc) and returns the tail; it refuses busy slots and non-hub callers.
- `couch --set-model` switches a Codex slot's model and effort and a Claude slot's model, verified from each agent's status line, with recorded-screen tests per agent; an unknown model backs out cleanly and lists the choices.
- The couch skill documents both, and the hub's TL skill notes the recipe.
- `--set-model` defaults to session-only scope, selects shared defaults only with explicit scope, and reports the observed model/effort/scope.
- Model-control confirmation lines do not generate false “waiting on a prompt” notices; actual questions and pickers still do.
- Exact caller/slot ownership and wrapper identity are revalidated; busy/unknown or occupied agent composers refuse, and failures never automatically replay uncertain keys.

## Plan

Durable implementation plan: [agent keys and model selection](../plans/000544-agent-keys-model-plan.md).

- [ ] Implement typed bounded key driving and verified per-agent model recipes through the exact-bound wrapper.
- [ ] Wire hub ownership/idle guards, receipts and model-control submission accounting; prevent false prompt waits.
- [ ] Verify recorded screens, stateful socket/PTY sequences and isolated live conformance; update docs, close and land.

## Log

### 2026-10-13

## Revisions

- 2026-10-14 02:15 — add (TL, from the first fleet-wide remote switch): (1) `--set-model` takes an explicit scope:
  Codex offers "session only" (`s`) vs "save as default" (Enter); default to session-only, since Codex's default is
  shared by every Codex slot. (2) After a remote switch, the agent's own confirmation line ("Model changed to … for
  this session only") must not trigger Couch's 10-minute "waiting on a prompt" notice: three false notices came from
  ariadne:5, parley.nvim:1 and others on 10-14. (3) The verified recipe used (Codex 0.x): Ctrl+U, `/model`,
  Alt+Enter (plain Enter is a newline in Pair's agent pane), ↑/↓ until the reverse-video row shows the label,
  Enter, ↓ to the effort label, `s`; confirm from the footer `GPT-6.1-Sol medium`. Skip slots whose screen shows
  "esc to interrupt" (mid-turn).

- 2026-10-14 — TL clarified the idle precondition: only the agent’s composer must be empty; the separate Neovim draft is unaffected and needs no guard. Existing wrapper Settled is the required evidence. Spec wording updated accordingly.
- 2026-10-14 — planning survey: exact-bound wrapper endpoint, styled terminal tail and single input writer already exist. Peer delivery adds chrome and can run during turns, so reuse its identity/transport and input ownership seams, not its delivery policy. No versioned model/effort/scope captures exist yet; gathering those in isolated scratch PTYs is the first execution step after approval. #549 PR #328 awaits the operator label and is not the base of this independent branch.

- 2026-10-14 — durable plan ready for TL approval. Shared wrapper transport/input lease and pure screen-driven reducer provide one driver; no implementation or estimate yet. Added scope and false-wait acceptance to Done when, matching the existing Revision. All outstanding design uncertainty about actual picker shapes is handled by the first isolated capture step and a stop/revision on mismatch.

- 2026-10-14 — plan approved by TL at 44e2e723. Full-flow change-code round 1 requested precise registered-wrapper lifetime, isolated Claude model auth (existing native-HOME fallback is unsuitable), and named-function test strategies. Added an active refinement for each; retained the approved feature/bounds. TL will update ops/skills-draft/xx-tl/SKILL.md after landing; this issue updates Pair docs only.
