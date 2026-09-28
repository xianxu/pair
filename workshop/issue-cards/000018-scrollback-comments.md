---
id: '000018'
status: done
created: 2026-05-09
updated: 2026-05-27
actual_hours: N/A
---

# 🤖[] comment markers in scrollback viewer → draft

## Problem

#000017 added the scrollback viewer. While reading scrollback, the
user often wants to capture follow-up notes — "check this", "fix
that line", "this looks wrong". Today they'd have to switch back to
the draft, type a prompt, lose the visual context.

The parley.nvim convention (`/xx-fix` skill) already defines a marker
syntax that claude understands:

- `🤖<X>[Y]` — scoped human comment about quoted text X
- `🤖[Y]` — bare human comment

Use it. Bind `Alt+q` in the scrollback viewer to drop a marker. On
`:q`, extract markers, format as `> <quote>\n<comment>\n\n`, append
to the draft. User comes back to the draft, sees the formatted
context, reviews, sends with `Alt+Return`.
