---
id: 000097
status: open
deps: []
github_issue:
created: 2026-07-01
updated: 2026-07-01
estimate_hours:
---

# agent name on pane is wrong — stale pane-<tag>-<agent>.json collides on pane_id

## Problem

The zellij agent-pane frame title shows the wrong agent: the pane is running
`claude`, but the frame reads `codex`. (`PAIR_AGENT`, `PAIR_PANE_TITLE`, and
`$PAIR_DATA_DIR/agent-<tag>` all correctly say `claude` — only the frame label is
wrong.)

## Spec

### Root cause (diagnosed)

`pair-title.sh` → `update_frame_titles()`
(`cmd/internal/runtimebundle/assets/runtime/files/bin/pair-title.sh:95-116`)
derives each pane's agent label from the **filename** of
`$PAIR_DATA_DIR/pane-<tag>-<agent>.json` and renames the zellij pane to
`<agent> (<count>) [<cwd>]`. It loops over **every** `pane-<tag>-*.json` for the
tag. When two of them point at the **same `pane_id`**, it renames that one pane
once per file, and — because the glob expands alphabetically — the last write
wins.

Observed on tag `ariadne`, both files carrying `pane_id: "0"`:

| file | mtime |
|------|-------|
| `pane-ariadne-claude.json` | Jul 1 00:22 (current session) |
| `pane-ariadne-codex.json`  | **Jun 29 12:27 (stale)** |

`claude` sorts before `codex`, so each 60s poll renames pane 0 to `claude …`
then immediately to `codex …` → the pane settles on **codex**. The stale
`pane-ariadne-codex.json` is left over from a codex session that used pane 0 on
Jun 29 and was never removed when claude later took the same pane_id.

**Not tag-specific.** The same stale twins exist for other tags:
`pane-2-codex.json`, `pane-pair-codex.json`, `pane-parley_nvim-codex.json` all
sit beside their `-claude.json` counterparts, so any tag where codex previously
held a pane_id now reassigned to claude will mislabel.

### Fix direction

Root cause is the pane-file lifecycle + the blind all-files loop. Options
(pick during design):

1. **Label only the active agent.** `update_frame_titles` already knows the live
   agent for the tag (the `$AGENT` arg it was spawned with, and/or
   `$PAIR_DATA_DIR/agent-<tag>`); skip any `pane-<tag>-<agent>.json` whose
   `<agent>` isn't the active one (or, when multiple panes legitimately coexist,
   dedupe by `pane_id` preferring the active agent).
2. **Clean up on reassignment.** When a pane is (re)assigned to an agent, remove
   the prior agent's `pane-<tag>-*.json` for that `pane_id` so no stale twin
   survives.

Add a regression test: two `pane-<tag>-*.json` sharing a `pane_id` must yield a
single frame rename to the active agent (mirror the existing
`PAIR_TITLE_TEST_CALL` / `_test_frame_titles_twice` harness).

### Immediate remediation (symptom, no code change)

`rm` the stale `pane-<tag>-<agent>.json` whose `pane_id` collides with a newer
twin (e.g. `pane-ariadne-codex.json`). Corrects the frame on the next poll;
recurs on the next agent switch until the code fix lands.

## Done when

- [ ] A tag with a stale `pane-<tag>-<other-agent>.json` on the same `pane_id`
      renders the frame as the **active** agent, not the alphabetically-last one.
- [ ] Stale `pane-<tag>-*.json` twins are prevented (cleaned on reassignment) or
      ignored by the frame updater.
- [ ] Regression test: two pane files sharing a `pane_id` → one rename, active
      agent.

## Plan

- [ ] Decide fix (active-agent filter vs. reassignment cleanup) in `pair-title.sh`
      + the pane-file writer.
- [ ] Implement + `PAIR_TITLE_TEST_CALL`-based regression test.
- [ ] Optionally ship a one-time cleanup of existing stale `pane-*-*.json` twins.

## Log

### 2026-07-01

- Moved from `ariadne#151` (filed in ariadne's inbox; the bug + fix live here in
  pair). Root cause diagnosed during that session — see the evidence above:
  `pane-ariadne-{claude,codex}.json` both on `pane_id 0`, alphabetical last-wins
  in `update_frame_titles`.
