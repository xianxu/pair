---
id: 000082
status: open
deps: ["#68"]
github_issue:
created: 2026-06-26
updated: 2026-06-26
estimate_hours:
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
  - minimized draft pane,
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

## Plan

- [ ] Replace fixed integer draft sizes with percentage-only sizes.
- [ ] Update layout comments/atlas if the experiment changes the intended
      ladder semantics.
- [ ] Run syntax/static checks that cover the touched files.
- [ ] Start or reuse a Pair Codex session in the new mode and record the
      dogfood result.

## Log

### 2026-06-26

- Split from #68 after live logs showed tracing is complete but the root-cause
  experiment should be tracked separately. Current hypothesis: the Codex repaint
  storm interacts badly with zellij's fixed-pane/swap-layout path, not with
  pair-wrap capture.
