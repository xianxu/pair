---
id: '000105'
status: done
started: 2026-07-06T22:04:17-07:00
created: 2026-07-06
updated: 2026-07-07
estimate_hours: 2.68
actual_hours: 1.78
---

# alt+shift+c: deterministic writer-triggered restart + fold draft WIP into continuation NEXT ACTION

## Problem

`alt+shift+c` (zellij `config.kdl` `bind "Alt C"`) → `PairConfirmCompact()` (`nvim/init.lua:3327`) → `send_to_agent(COMPACT_PROMPT)` (`nvim/init.lua:3324`). `COMPACT_PROMPT` is a **two-step NL instruction to the agent**: (1) write a continuation via `pair continuation`, then (2) *itself* run `pair continue <slug>`. Step 2 is agent judgment, not code — so if the agent writes the doc but skips `pair continue`, no restart marker is written and the outer reincarnation loop (`createflow.go:63-109`) just exits. **That is the "restart stopped working" bug: the restart was never deterministic.**

Second, on a successful compaction restart the outer loop **overwrites** the draft with a seed line (`createflow.go:227-229`: `"Read workshop/continuation/%s and continue from its NEXT ACTION."`). So any WIP the operator left in the draft ("`*`") is **lost** across the restart.
