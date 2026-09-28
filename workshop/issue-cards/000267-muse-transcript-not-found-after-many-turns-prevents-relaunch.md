---
id: 000267
status: open
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
github_issue:
---

# muse transcript not found after many turns prevents relaunch

## Problem

Operator report 2026-09-15: after a `muse` session has run for many turns, `pair` can no longer locate its transcript — the session becomes unresumable and `relaunch` (and resume) refuses. Fresh `muse` sessions on the same repo/tag start fine, and the same repo/tag resumes fine with fewer turns. The failure is specific to long-lived `muse` sessions with large transcript history.

This was reported alongside two other `muse` lifecycle bugs (relaunch crashing from switcher, `Alt+Return` staying in composer — #266). The symptom here is distinct: the transcript lookup itself fails, not the key-remap seam. The ledgers for affected threads have grown large; see #238 (`~600 KB per launch` full-root `LaunchArtifactBoundary` snapshot) and #237 (8 MiB whole-file cap made threads unresumable, now fixed to per-record cap). Even with #237's per-record cap, a single `launch` row approaches `jsonRecordLimit` (8 MiB) as the machine's transcript count grows, and `muse` sessions that have accumulated many turns produce more observations to snapshot.

Open questions to resolve during fix:
- Is the lookup failing because the `launch` row itself is over `jsonRecordLimit`, because the proof's artifact set references a transcript that has rotated/cleaned up on disk, or because the incremental inventory's stable-id/generation check no longer matches after many turns?
- Why `muse` specifically surfaces this more readily than `claude`/`codex`/`agy` — is it the storage-root size (`muse-sessions` vs `claude/projects`), the scanner schema (`muse-v1`), or the number of observations snapshotted in `prepareRuntimeLaunch`?
