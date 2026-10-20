---
id: 000740
status: open
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
---

# couch: global park conformance on a disposable fleet (park all, restart, resume; nothing lost)

## Problem

The operator's shutdown procedure (10-20 drain redesign) ends with a global park before restarting the host. Global
park is rarely used and untested on a full fleet, so the procedure rests on a path nobody has proven.
