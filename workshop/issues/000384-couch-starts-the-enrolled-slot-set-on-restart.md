---
id: 000384
status: open
deps: []
github_issue:
created: 2026-10-02
updated: 2026-10-02
estimate_hours:
card_mirror: 'b834da4b7088898783108fb63604183b1c1e62ac' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch starts the enrolled slot set on restart

## Problem

After a restart Couch reattaches only agents that are still running
(detached/unknown threads, `couchtty/menu_reattach.go`). Parked slots stay
parked, so the operator has to wake each slot by hand before any of them can
receive work. pair#367's recovery report and its "continue #N" step assume the
enrolled slots are live and idle once Couch is up.

## Spec

Project: pair/workshop/projects/cross-slot-work-scheduling.md. Captured for
operator review; no implementation is authorized by this issue creation.

On start, Couch brings the enrolled slot set (`slot_repositories` ×
`worktree/<repo>-slotN/<repo>`, as `OSSlotCatalog.Discover` enumerates it, plus
the primary `:0`) to live, idle agents: reattach what is running, resume what is
parked, and leave unusable rows (path missing, session gone, never started)
reported, not repaired. A started slot receives no work: being live does not
mean being assigned. Starting stays bounded and observable (per-slot outcome,
failures named), and an operator opt-out disables it. Deliberately parked slots
need a decision: the record only implies intent through `verified_park`; this
issue settles whether a deliberate park is honored, and how that intent is
recorded, rather than guessing from record shape.

## Done when

- A restart with live, detached, parked and unusable slots ends with every
  usable enrolled slot live and idle, and each unusable one reported with its
  reason; nothing is sent to any agent.
- Deliberate-park behavior is specified and tested, distinct from a park left
  by a crash.
- The opt-out disables the slot start and leaves today's reattach-only path.
- A restart acceptance test drives the real Couch start path with isolated
  stores (`COUCH_ISOLATED_ROOT`), including a slot whose start fails.

## Plan

Implementation plan to be designed after issue claim and start-plan.

## Log

### 2026-10-02

Split out of pair#367 at the operator's direction: #367's survey found Couch
does not start a slot set on restart. #367 depends on this issue.
