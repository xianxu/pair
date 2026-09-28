---
id: '000137'
status: done
started: 2026-08-16T21:35:17-07:00
created: 2026-08-16
updated: 2026-08-16
estimate_hours: 1.66
actual_hours: 0.41
---

# Codex Return rewrite only in composer

## Problem

Pair currently rewrites plain Return in the Codex pane by default, then tries to
discover every menu/permission picker that needs bare Return. That inverts the
safer rule: the multiline rewrite only belongs in Codex's visible composer
box. Codex has multiple user-facing menu systems, so chasing every menu footer
will keep drifting.
