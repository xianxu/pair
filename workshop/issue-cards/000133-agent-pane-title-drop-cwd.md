---
id: '000133'
status: done
started: 2026-07-29T15:42:55-07:00
created: 2026-07-29
updated: 2026-07-29
estimate_hours: 1.60
actual_hours: 0.90
---

# agent pane title drops the cwd suffix

## Problem

zellij composes the outer terminal tab title as `<session name> | <focused pane
title>` (hardcoded in `zellij-utils/src/shared.rs`; no config option, upstream
request open since 2022 — zellij-org/zellij#1495, #2088).

#130 made the session half carry the folder and the tag: `ComposeSessionName`
(`session_index.go:75`) emits `📁{repo}` plus any residual tag tokens. Verified
live — `PAIR_SESSION_NAME=📁pair`, and `zellij list-sessions` shows `📁pair`,
`📁ariadne`, `📁kbench`.

The agent pane still appends `[<tilde-cwd>]`, so the tab reads:

```
📁pair | claude (629k) [~/workspace/pair]
```

The repo is named twice. Three of the five Pair-owned surfaces already match the
target — only the agent pane is off:

| pane | writer | title today | on target? |
|---|---|---|---|
| agent (startup) | `launcher.PaneTitle` `format.go:61` → `PAIR_PANE_TITLE` → `main-{2,3}.kdl` | `claude [~/workspace/pair]` | no |
| agent (steady) | `titlepoller.frameTitle` `titlepoller.go:72` | `claude (629k) [~/workspace/pair]` | no |
| draft | layout `name="draft"` | `draft` | yes |
| right terminal | `termcmd.paneTitleLocked` `run.go:1042` | `terminal 1 [a] terminal 3` | yes |
| right terminal (`Alt+r`) | `termcmd.renamePaneTitleLocked` `run.go:1057` | `[rename: work│] terminal 3` | yes |

**Two** writers emit the suffix, not one. The startup writer matters on its own:
`PAIR_PANE_TITLE` is what the pane is named at launch, so leaving it in place
shows the old shape from launch until the poller's first pass.
