---
id: 000544
status: open
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '20a51fbd9a839a73ec572fb60a91b499e81485fc' # card fields mirrored from issue-cards; edit via sdlc
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

Two layers, hub-only, idle slots only (refuse when the agent is busy or the draft has unsent text):
1. **`couch --keys repo:N KEYSPEC…`**: inject named keys into the slot's agent pane, with no chrome: `text:…`, `enter` (plain CR), `submit` (Alt+Enter), `up`, `down`, `esc`, `clear-line`, `wait:MS`. Couch resolves the session and pane itself (no env scraping), writes through the slot's terminal, and returns the agent pane's tail after the last key, like `--peek`. One request, one bounded sequence; no retries.
2. **`couch --set-model repo:N --model NAME [--effort LEVEL] --confirm`**: per-agent recipes built on (1) that open the picker, find the row by its visible label in the peeked screen (not by position), select it, handle the effort step where the agent has one, and verify the result from the agent's own status line (e.g. Codex's footer `GPT-6-Astra medium`). If the label isn't found, back out with ESC and report the visible choices; never guess.

Agents' pickers differ and change across versions; recipes live in one table per agent, tested against recorded screens. The slot-continuity target's spirit applies: one action, an honest result, no limbo (a failed drive backs out to the agent's prompt).

## Done when

- `couch --keys` drives an idle scratch slot's agent pane (text, submit, arrows, esc) and returns the tail; it refuses busy slots and non-hub callers.
- `couch --set-model` switches a Codex slot's model and effort and a Claude slot's model, verified from each agent's status line, with recorded-screen tests per agent; an unknown model backs out cleanly and lists the choices.
- The couch skill documents both, and the hub's TL skill notes the recipe.

## Plan

- [ ]

## Log

### 2026-10-13
