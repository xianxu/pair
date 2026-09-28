---
id: '000108'
status: done
started: 2026-07-07T17:32:26-07:00
created: 2026-07-07
updated: 2026-07-07
estimate_hours: 0.25
actual_hours: 0.11
---

# pair review mode spinner should be seconds

## Problem

The review-pane statusline spinner uses a compact elapsed-time formatter while
waiting for the agent. Once elapsed time reaches 60 seconds, the formatter drops
the seconds component (`2m`), which makes it harder to tell whether the review is
still actively advancing.
