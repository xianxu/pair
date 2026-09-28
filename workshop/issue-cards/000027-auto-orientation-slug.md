---
id: '000027'
status: done
created: 2026-05-31
updated: 2026-05-31
estimate_hours: 7
actual_hours: 11
---

# Auto-maintained orientation slug in the winbar (`=== branch | focus ===`)

## Problem

Working multiple small issues across several peer-repo tabs in one
session, it's easy to lose track of which tab is doing what when
switching back. The existing `=== comment ===` mechanism (sticky line-1
annotation, pinned to the winbar — `nvim/init.lua:175` `pair_pin_header`)
addresses this, but it relies on the user *remembering to type it*, which
they routinely forget. `/recap` (built-in Claude Code away-summary) is a
different thing: multi-line, in-transcript, fires on return — it does not
set the winbar comment and its output isn't capturable.

We want the orientation cue **auto-maintained**: updated each time Claude
finishes responding (the agent-went-idle condition). Note: today's
notifications are not a Stop hook — they come from `pair-wrap` sniffing
the agent's OSC `notify` escapes (`.claude/settings.json` has only a
`SessionStart` hook). This issue adds the *first* `Stop` hook; it fires on
the same logical idle condition but is independent of the OSC notify path.
