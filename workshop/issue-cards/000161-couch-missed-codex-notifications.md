---
id: '000161'
status: done
started: 2026-09-01T13:35:54-07:00
created: 2026-09-01
updated: 2026-09-01
estimate_hours: 6.93
actual_hours: 0.45
---

# Couch misses Codex completion notifications

## Problem

During Couch dogfood testing, Codex completed work in Pair but Couch did not
surface a completion notification. No notification appeared in either the
status bar or the switcher.

The same joint failure can occur for Claude. One confirmed marker is:

```text
✻ Sautéed for 34s · done 1:39 PM
```

Claude marker mode currently requires the verb after `✻` to match ASCII-only
`[A-Za-z]+`. The accented `é` therefore prevents this otherwise valid marker
from reaching the shared notification sink. Codex remains a separate failure
mode because it currently relies on native OSC notification output.

A captured Codex completion renders as `─ Worked for 3m 57s ─…`. The same
capture also showed a copy prefixed by `> `. Investigation must distinguish the
live colored status line from quoted or replayed copies before granting either
form notification authority.
