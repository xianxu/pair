---
id: 000356
status: open
deps: []
github_issue:
created: 2026-09-30
updated: 2026-09-30
estimate_hours:
card_mirror: '06af6710e22116e419a40d85f3873c89f52200f1' # card fields mirrored from issue-cards; edit via sdlc
---

# Audit hotkeys couch/pair intercept, per pane, against the draft-pane-first principle

## Problem

Pair and couch intercept hotkeys at several layers, and no single inventory says
which keys each layer takes, or from which pane. That matters more over time:
coding agents keep claiming new chords. Codex now binds Alt+Up; Claude now
binds Ctrl+Return. Every chord pair or couch intercepts outside the draft pane
is one the agent can never receive. Today such a collision surfaces only when
the operator notices an agent feature that silently doesn't work.

## Spec

**Principle.** Pair/couch hotkeys live in the draft pane (nvim), which pair
owns outright. Intercepting a key globally (zellij binds, which apply in every
pane) or inside the agent pane (pair-wrap's stdin translator) is a much higher
bar. It needs a stated reason why the draft pane can't host it, and a check
that no supported agent uses the chord.

**Audit.** Produce one table, key → layer → pane(s) where it's intercepted →
what it does → whether a supported agent (claude, codex, agy, muse, qoder) binds
it → verdict (keep / move to draft pane / drop / pass through when the agent
owns it). Layers to cover, each derived from source rather than recalled:

- zellij config binds (`zellij/config.kdl`, `zellij/layouts/`), which are global
  to every pane
- pair-wrap stdin in the agent pane: workbench chords
  (`cmd/internal/workbenchshortcut`), Return / Alt+Return / Alt+Backspace remap
  (`cmd/internal/wrapcmd/harness_tty.go` keymaps)
- `pair term` in the right-hand panes (`cmd/internal/termcmd`)
- couch's console and panels (`cmd/internal/couchtty` `keys.go`,
  `panelkeys.go`)
- the draft pane and review pane nvim maps (`nvim/init.lua`, `nvim/review*`),
  which are the baseline and not the concern, but belong in the table so a
  "move to draft pane" verdict has a slot to land in

Agent chord inventory: each agent's current keybinding docs/help, dated with the
agent version, so the table can be re-checked when agents update.

## Done when

- A committed inventory (atlas page) lists every intercepted key per layer and
  pane, derived by enumerating the binding sources (a test or script, not a
  hand-copied list — lessons: "a claim about every site needs a derived
  enumeration")
- Every non-draft-pane intercept has a verdict and a one-line justification;
  known collisions (Codex Alt+Up, Claude Ctrl+Return) are resolved or have a
  follow-up issue
- The principle is written into the atlas (and the harness bring-up guide's
  checklist), so a new global or agent-pane binding has to argue its case

## Plan

- [ ] Enumerate binding sources per layer; decide script vs test for derivation
- [ ] Collect current chord lists for each supported agent (with versions)
- [ ] Build the table; flag collisions
- [ ] Verdicts; file follow-ups for moves/removals that are more than trivial
- [ ] Atlas page + principle + bring-up checklist entry

## Log

### 2026-09-30

- Filed at the operator's request, after #211. Principle and the two known
  collisions (Codex Alt+Up, Claude Ctrl+Return) are the operator's.
