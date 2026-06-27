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

- [x] `zellij/layouts/main.kdl` uses no integer `size=` values for the draft pane.
- [x] The layout still starts a normal Pair session and exposes the expected two
  panes.
- [x] A Codex session is run for a meaningful dogfood interval in percentage-only
  mode.
- [x] The log records whether scroll wedged again and whether zellij still emits the
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
- Dogfood readout from live `pair-ariadne` after the operator reported scroll
  stopped working:
  - Session `pair-ariadne` was running the percentage-only layout in `small`
    mode (`layout-mode-ariadne=small`); `zellij --session pair-ariadne action
    list-panes --json --command --geometry --state` showed the agent pane at 39
    rows and the draft pane at 12 rows.
  - `pair-wrap` was still alive and still writing both stdout and raw scrollback:
    `wrap-events-ariadne.jsonl` had 111k events, and
    `scrollback-ariadne-codex.raw` had grown to 6.8 MB.
  - The zellij server log contained the old `Can't combine fixed panes` /
    `Failed to focus stacked pane` / `Failed to find position of flexible pane`
    storm at 10:59, before the percentage-only `pair-ariadne` session started at
    18:15. The active `pair-ariadne` window showed startup warnings and
    detached-client noise, but no recurrence of the fixed-pane errors.
  - A targeted zellij CLI probe showed scrollback still exists and can move:
    dumping `terminal_0`, running `zellij --session pair-ariadne action
    scroll-up --pane-id terminal_0`, and dumping again produced different
    visible content over a 946-line full dump. A follow-up `scroll-down` restored
    the pane.
  - Current conclusion: percentage-only sizing falsifies the fixed-pane error
    storm as the sole root cause. The remaining scroll failure is narrower:
    likely mouse/wheel routing, hover/focus targeting, or interaction with the
    bottom borderless draft pane, not pair-wrap capture loss or absent zellij
    scrollback.
