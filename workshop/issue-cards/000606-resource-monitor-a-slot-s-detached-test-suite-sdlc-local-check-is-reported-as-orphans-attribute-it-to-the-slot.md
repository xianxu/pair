---
id: 000606
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
---

# Resource monitor: a slot's detached test suite (sdlc local-check) is reported as orphans; attribute it to the slot

## Problem

pair#587's sampler alerts the hub about "orphans" that are really a slot's own test suite: 2026-10-15 23:5x,
"pair:6 pid=39601 name=couchcore.test orphan-rss 338 MB for 60 s". The process was a test binary of pair:6's recorded
suite (`sdlc local-check run --name suite`), which runs detached from the slot's process tree, so its test processes
aren't under the slot's wrapper. It exited a minute later. Every full suite on every slot will raise these.
