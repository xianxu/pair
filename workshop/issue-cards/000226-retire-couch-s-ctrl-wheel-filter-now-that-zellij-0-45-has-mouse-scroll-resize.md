---
id: 000226
status: open
created: 2026-09-10
updated: 2026-09-10
estimate_hours:
github_issue:
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
