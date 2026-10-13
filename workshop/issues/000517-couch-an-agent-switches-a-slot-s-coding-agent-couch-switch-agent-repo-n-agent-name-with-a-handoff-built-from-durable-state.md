---
id: 000517
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '3f2027ec27910a8613d20896a8a7f21736ad8b91' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T20:05:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# couch: an agent switches a slot's coding agent (couch --switch-agent repo:N --agent NAME), with a handoff built from durable state

## Problem

The operator (2026-10-12, Claude quota at 80%) wants an agent-guided agent switcher the TL can use, to move slots to Codex as they free up. Today only the operator can switch a slot's coding agent (the thread's actions, **switch coding agent**), and its handoff feels robotic: Couch launches a fresh conversation and asks the new agent to read the outgoing context and summarize its orientation. Related: pair#414 (a Claude → Grok switch lost its orientation and the attach was refused).

## Spec

Agent-guided: the agents who know the work write the handoff; Couch does the mechanics.
1. **A verb for agents:** `couch --switch-agent repo:N --agent NAME [--handoff-file PATH] --confirm [--json]`, the same operation as the switcher's form, with its guards (refuses a busy or unknown slot). Cost-tagged `[heavy]` in help. The TL can call it for any slot; a slot can call it on itself (`--switch-agent self`).
2. **The handoff is written by an agent, not generated:** before the switch, the outgoing agent (or the TL, for a slot it supervises) writes a short brief the way a colleague would: what the issue is, where the work stands (branch, last commit, what's verified), what's next, open questions, and anything not in the issue file. Couch delivers it verbatim as the new agent's first message. With no handoff (an idle slot on its resting branch), the new agent starts fresh with no orientation prompt.
3. **Durable state stays the source:** the brief points at the issue's Spec, Plan, Log and Revisions rather than restating them; the outgoing agent commits its work first.
4. The switcher's form gains the same path: it asks the outgoing agent for the brief, then switches.
5. Fold in or reference pair#414's attach/orientation failure, so delivery is reliable across agents (Claude, Codex, Grok).

## Done when

- The TL switches an idle slot to Codex with one couch call, and a mid-issue slot to Codex with a brief written by its outgoing agent; the new agent resumes the issue from the brief and the issue file.
- The operator approves the handoff's feel on a live switch.
- Tests cover the guards, delivery of a brief verbatim, and the no-handoff fresh start.

## Plan

- [ ]

## Log

### 2026-10-12

## Revisions

- 2026-10-12 — the operator: the switch is driven by the TL's skill. Done when also: a "switch a slot's agent" procedure in
  ops' `skills-draft/xx-tl/SKILL.md` (when to switch, how to write the brief, the exact couch call, how to check the new
  agent picked it up). The TL reviews that section; the first real use moves the paused ariadne items (#230, #282, #305)
  to Codex with briefs.

- 2026-10-12 — operator scope revision relayed by ops:0 (receipt
  `4f8b1134-cc92-4af2-b4e0-a1deecf71f1a`): the incoming agent must be able to
  find and read the previous session itself. The brief names the previous
  agent-native transcript and per-incarnation tty log, stays short, and points
  at durable issue state. Measure the existing recording and discovery paths
  before designing. Add previous-incarnation peeking and access to the whole
  tty log as readable plain text (all or paged), with cost tags. The current
  live-buffer peek is insufficient even with a large requested line count.
