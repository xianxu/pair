---
id: '000185'
status: done
started: 2026-09-04T15:18:33-07:00
created: 2026-09-04
updated: 2026-09-04
actual_hours: 0.75
---

# Status-row notices never expire, so a momentary refusal reads as current state

## Problem

Operator report, mid-#182 smoke testing: the couch status row showed

    [pair]  brain  tools  · previous: nowhere to return to

long after the ctrl+backspace that produced it, with the question "when do we
clear the status bar's message?"

The answer was **never**. `paintNow` renders `c.feed.Latest()` — the newest entry
of a bounded rolling queue — and the feed had exactly two producers (an exit
notice and `setNotice`) and no consumer that ever retired anything. No timeout,
no clear-on-keystroke, no expiry. A refusal about a keystroke pressed a minute
ago sat there until some unrelated notice happened to displace it, reading as
current state when it was a stale event.
