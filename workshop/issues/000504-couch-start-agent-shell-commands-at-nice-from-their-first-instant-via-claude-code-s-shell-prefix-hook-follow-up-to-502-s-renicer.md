---
id: 000504
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '2f7a8c8ae8e027494042c9fb63d8352fbfbf93e5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T15:17:34-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# couch: start agent shell commands at nice from their first instant via Claude Code's shell-prefix hook (follow-up to #502's renicer)

## Problem

#502's renicer lowers what a slot's agent spawns to `COUCH_SLOT_NICE`, but only on its next scan (every 5 s). A
command an agent starts runs at normal priority for its first seconds. For a short burst (a build, a test file)
that can be the whole run, which costs the UI its responsiveness while it lasts. The operator accepted the delay
for #502 and asked for this as an optional follow-up (10-12, via the TL).

## Spec

- For Claude Code slots: start each Bash tool command at the slot's level from its first instant. Use Claude
  Code's shell-prefix setting (believed to be `CLAUDE_CODE_SHELL_PREFIX`; verify the name and its semantics
  first), pointing it at a small wrapper that runs `nice -n <level>` around the command, for the agents Couch or
  pair launches.
- Agent-agnostic stays with #502: the renicer remains the catch-all for Codex, MCP servers and anything a prefix
  doesn't cover. This is an addition, not a replacement.
- `COUCH_SLOT_NICE=0` turns both off.
- Never lower the agent itself, the wrapper or zellij.

## Done when

- In a Claude Code slot, a Bash tool command shows the slot's nice from its first `ps` sample, not only after the
  renicer's next pass; the agent stays at 0.
- With `COUCH_SLOT_NICE=0` the prefix isn't set.
- The setting's name and behavior are verified against the installed Claude Code and recorded.

## Plan

- [ ]

## Log

### 2026-10-12
