---
id: 000347
status: open
deps: []
github_issue:
created: 2026-09-29
updated: 2026-09-29
estimate_hours:
card_mirror: '49cc3f7da8a1d4f57f9e184fcbbf60bec7c5b437' # card fields mirrored from issue-cards; edit via sdlc
---

# Give each TTY capture a permanent identity

## Problem

TTY captures currently reuse an active per-tag pathname. Archiving that path on restart can invalidate a continuation's reference or make it point to another session. Looking for the latest archive is imprecise. Compaction currently avoids this by copying a specifically named snapshot, at the cost of duplicate capture data.

## Spec

Give every live wrapper capture a permanent unique identity and raw/timing-sidecar paths at creation. Restart creates another capture; a current pointer locates the active one without renaming historical captures. Continuation/orientation references use the exact capture identity, optionally with a recorded end offset for a snapshot boundary. Define ownership and retention of referenced captures, and migration/read compatibility for existing active and parked paths. Reuse artifactpath and retention authorities rather than adding a competing log registry.

This is a separate TTY improvement requested during #346. For #346 retain the current compaction copy arrangement and startup archival of reusable paths; this task is not a prerequisite for that recovery fix.

## Done when

- Each wrapper launch writes a unique permanent capture family; ordinary restart never moves or reuses an earlier capture's paths.
- Current-capture consumers follow an explicit pointer, while continuation references remain exact across subsequent launches.
- Snapshot consumers can identify their intended capture boundary without guessing the latest archive.
- Referenced captures have a defined retention lifetime; existing capture formats/paths remain readable or have a tested migration.
- Tests exercise multiple restarts, crash recovery, continuation resolution and collection of referenced/unreferenced captures.

## Plan

- [ ] Design permanent identity, current pointer, reference lifetime and migration against existing artifact/retention APIs.
- [ ] Implement capture creation and consumer migration with production-boundary regression tests.

## Log

### 2026-09-29

Operator selected permanent per-launch capture identities as a separate improvement. #346 continues with existing compaction copies; no implementation of this task has started.
