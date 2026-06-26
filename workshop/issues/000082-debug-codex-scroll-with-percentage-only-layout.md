---
id: 000082
status: working
deps: ["#68"]
github_issue:
created: 2026-06-26
updated: 2026-06-26
estimate_hours: 0.8
started: 2026-06-26T15:49:13-07:00
---

# Debug Codex scroll with percentage-only layout

## Problem

Codex scrolling can still wedge in Pair even after #68's tracing work. The
current logs show pair-wrap still forwarding stdout and writing raw scrollback
with no write errors, while zellij reports layout/render errors:

- `Can't combine fixed panes`
- `Failed to focus stacked pane`
- `Failed to find position of flexible pane`

#68 established the tracing substrate and ruled out several earlier theories
(prompt injection, pair-wrap itself, sync-marker stripping as the sole cause).
This issue tracks the next focused experiment: remove integer fixed pane sizes
from Pair's zellij layout and run Codex that way long enough to see whether the
scroll wedge disappears.

## Spec

- Keep #68 as the completed tracing/history issue; do not continue hijacking it
  for layout experiments.
- Change Pair's main zellij layout so the draft pane ladder uses percentage
  sizes only, avoiding zellij's fixed-pane code path.
- Preserve the user-facing ladder intent:
  - small/default draft pane,
  - minimized draft pane, accepting that percentage-only sizing may make this a
    small variable-height pane rather than exactly one statusline row,
  - larger one-third draft pane.
- Keep the change easy to revert if the experiment does not help.
- Use existing #68 logs (`zellij-actions-<tag>.jsonl`,
  `wrap-events-<tag>.jsonl`, zellij server log) as the measurement substrate.

## Done when

- `zellij/layouts/main.kdl` uses no integer `size=` values for the draft pane.
- The layout still starts a normal Pair session and exposes the expected two
  panes.
- A Codex session is run for a meaningful dogfood interval in percentage-only
  mode.
- The log records whether scroll wedged again and whether zellij still emits the
  fixed-pane errors.

## Estimate

```estimate
model: estimate-logic-v2
familiarity: 1.0
item: method-b-decisions design=0.2 impl=0.4
item: atlas-docs design=0.0 impl=0.1
design-buffer: 0.30
total: 0.8
```

## Plan

- [x] Replace fixed integer draft sizes with percentage-only sizes, using a
      small percentage for the minimized rung and recording whether it remains
      usable.
- [x] Update layout comments/atlas if the experiment changes the intended
      ladder semantics.
- [x] Launch Pair with the changed layout and confirm zellij accepts the layout
      with the expected two panes.
- [ ] Start or reuse a Pair Codex session in the new mode, then record whether
      scroll wedged and whether fixed-pane errors recur in the zellij log.

## Log

### 2026-06-26

- Split from #68 after live logs showed tracing is complete but the root-cause
  experiment should be tracked separately. Current hypothesis: the Codex repaint
  storm interacts badly with zellij's fixed-pane/swap-layout path, not with
  pair-wrap capture.
- Changed the layout ladder to percentage-only draft pane sizes (`5%`, `24%`,
  `33%`) and updated the nvim rung detector/docs accordingly.
- Smoke-launched `pair-pct-smoke` under zellij with the changed layout. `zellij
  action list-panes --json --command` reported two terminal panes: agent at 18
  rows and draft at 6 rows in a 24-row test terminal, matching the 24% default
  rung. The temporary session was deleted after verification.
