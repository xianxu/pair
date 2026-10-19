---
id: 000677
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'fe5ec13f8d1d7e1483b98eeea70396ea80e624ef' # card fields mirrored from issue-cards; edit via sdlc
---

# diagnostic logs: untie from backup, enforce the per-file cap, keep ~7 days (48 GB, +1.5 GB/day)

## Problem

Pair's diagnostic logs (wrap-events, .pair-diagnostics segments, adapt, fullscreen) are ~48 GB and grow ~1.5 GB a day (survey 2026-10-18, ops qa 2026-10-18-16-history-backup-review.md). Their deletion is blocked by diagnosticlog.KeepDiagnosticsUntilBackup = true ("diagnostic eviction awaits history backup"), and that backup (pair#394) does not exist. Single files reach 4.9 GB despite the 64 MB per-generation cap. The operator agreed debugging logs can be trimmed without a backup.

## Spec

Untie diagnostic retention from history backup: diagnostics are not history. Enforce the existing per-file/generation cap so files rotate, and keep about 7 days of generations (configurable), deleting older ones without archiving. Find why a single wrap-events file grew to 4.9 GB past the cap and fix that rotation. Review whether the wrap-events detail level is needed at this volume (a measured bytes-per-turn figure). Existing 48 GB: the first sweep reclaims it; report how much.

## Done when

- No diagnostic file exceeds the cap; a test grows one past it and sees rotation.
- Generations older than the window are deleted without a backup; a counted test covers the sweep.
- The first live sweep's reclaimed bytes are recorded.

## Plan

- [ ]

## Log

### 2026-10-18
