---
id: 000226
status: open
deps: []
github_issue:
created: 2026-09-10
updated: 2026-09-10
estimate_hours:
---

# retire couch's ctrl+wheel filter now that zellij 0.45 has mouse_scroll_resize

## Problem

`couchtty.stripWheelResizeModifier` (`cmd/internal/couchtty/mouse.go`) strips the
ctrl bit from wheel reports because zellij 0.44.3 maps ctrl+wheel to a pane
resize with no way to turn it off (`#213`). Its doc carries a tripwire: *"DELETE
THIS once zellij is upgraded to a version carrying `mouse_scroll_resize`"*.

That version exists and is now what the operator runs: zellij 0.45.1's
`setup --dump-config` documents `mouse_scroll_resize` (default true). `#223`
moved pair onto zellij 0.45 — its wrap fix is zellij#5357 — and `#223`'s close
review flagged the tripwire as tripped.

## Spec

Replace the filter with configuration: `mouse_scroll_resize false` in
`zellij/config.kdl`, then delete `stripWheelResizeModifier` and its call site,
as the tripwire says.

**Blocked on 0.45 being a real floor, not an assumed one.** On zellij 0.44.x the
key is silently ignored (the very reason the filter exists), so deleting the
filter there brings the ctrl+wheel resize back. Today only the README states
the floor; nothing enforces it. Either land a `pair doctor` / startup check for
zellij >= 0.45.0 first, or accept that a 0.44 user loses the filter.

## Done when

- ctrl+wheel over a couch pane scrolls and does not resize it, on zellij 0.45,
  with the filter deleted — verified live, per `#213`'s reproduction.
- A 0.44 zellij is either refused/warned by pair, or the loss is recorded as
  accepted.

## Plan

- [ ] Decide the floor mechanism (doctor warning vs. startup refusal vs. none).
- [ ] Add `mouse_scroll_resize false`; delete the filter and its tests.
- [ ] Live check on 0.45 with ctrl+wheel.

## Log

### 2026-09-10

Filed from `#223`'s close review (Minor, `version-scoped-workaround-tripwire`).
