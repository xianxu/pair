---
id: '000097'
status: done
started: 2026-07-05T21:11:05-07:00
created: 2026-07-01
updated: 2026-07-05
estimate_hours: 0.5
actual_hours: 0.25
---

# agent name on pane is wrong — stale pane-<tag>-<agent>.json collides on pane_id

## Problem

The zellij agent-pane frame title shows the wrong agent: the pane is running
`claude`, but the frame reads `codex`. (`PAIR_AGENT`, `PAIR_PANE_TITLE`, and
`$PAIR_DATA_DIR/agent-<tag>` all correctly say `claude` — only the frame label is
wrong.)
