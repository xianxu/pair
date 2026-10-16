---
id: 000606
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '50e51dbd7a58f0ed8690dbc8ece01fb2955a61c0' # card fields mirrored from issue-cards; edit via sdlc
---

# Resource monitor: a slot's detached test suite (sdlc local-check) is reported as orphans; attribute it to the slot

## Problem

pair#587's sampler alerts the hub about "orphans" that are really a slot's own test suite: 2026-10-15 23:5x,
"pair:6 pid=39601 name=couchcore.test orphan-rss 338 MB for 60 s". The process was a test binary of pair:6's recorded
suite (`sdlc local-check run --name suite`), which runs detached from the slot's process tree, so its test processes
aren't under the slot's wrapper. It exited a minute later. Every full suite on every slot will raise these.

## Spec

- Attribute processes started by `sdlc local-check run` (and its `make test` / `go test` / test binaries) to the slot
  that started the run: the runner's in-flight record (under the worktree's `.git/sdlc/`) names the worktree, which
  maps to the slot; or the runner records its process-group ID at start.
- Such processes count toward the slot's resources (CPU, memory, writes), not as orphans.
- A true orphan (no live slot or run claims it) still alerts as today.

## Done when

- Fake-backed test: a detached suite's test binary is attributed to its slot and raises no orphan alert; an unclaimed
  process still does. Live: a full suite on one slot produces no orphan alert.

## Plan

- [ ]

## Log

### 2026-10-15
