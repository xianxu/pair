---
id: '000064'
status: done
created: 2026-06-16
updated: 2026-06-16
estimate_hours: 0.2
actual_hours: N/A
---

# Park-nudge prompt: 5s timeout, auto-default N

## Problem

On Alt+x quit, `cleanup_quit_marker` (`bin/pair`) prompts:

```
pair: preserve "pair-N" scrollback to distill into a continuation later? [y/N]:
```

It blocks forever on `read` waiting for an answer. A quit shouldn't hang on an
unattended prompt — if the operator walks away (or a wrapper kills the pane
slowly), the cleanup stalls. The default is already N (preserve nothing), so an
unanswered prompt should just auto-pick N after a short timeout.
