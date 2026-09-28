---
id: '000031'
status: done
created: 2026-05-31
updated: 2026-05-31
estimate_hours: 1.0
actual_hours: 0.5
---

# codex ak-user-question flow need alt+return to confirm

## Problem

for ask-user-question flow, normal return should work. this likely is caused by our tinkering around how return, alt+return works in agent pane. I remember we have code to detect the presence of ask-user-question (probably per agent) and remap temporarily to allow return to pass through. probably that's not working for codex.
