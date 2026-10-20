---
id: 000729
status: open
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
---

# recover-plan: an enrolled non-repository directory shows as ignored with a reason, not an unavailable fleet

## Problem

`couch --recover-plan-from-sdlc` reports an enrolled directory that is not a git repository as an unavailable fleet whose error is raw `sdlc` output ("resolve fleet: resolve workspace: sdlc workspace --json: exit status 1: … fatal: not a git repository"). On this machine the home directory `/Users/xianxu` is enrolled, because a thread was started there. The row reads like a working-directory failure. The hub read it that way during pair#726, but the report works from any directory and the main fleet reports present.
