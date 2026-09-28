---
id: '000009'
status: done
created: 2026-05-02
updated: 2026-05-02
actual_hours: N/A
---

# migrate drafts and logs to XDG data dir

## Problem

Drafts and per-session prompt logs live under `~/scratch/`. That was lazy v1 placement. The right home per XDG Base Directory spec is `${XDG_DATA_HOME:-~/.local/share}/pair/`.
