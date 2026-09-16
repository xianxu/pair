---
id: 000267
status: open
deps: []
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
---

# muse transcript not found after many turns prevents relaunch

## Problem

Operator report 2026-09-15: after a `muse` session has run for many turns, `pair` can no longer locate its transcript — the session becomes unresumable and `relaunch` (and resume) refuses. Fresh `muse` sessions on the same repo/tag start fine, and the same repo/tag resumes fine with fewer turns. The failure is specific to long-lived `muse` sessions with large transcript history.

This was reported alongside two other `muse` lifecycle bugs (relaunch crashing from switcher, `Alt+Return` staying in composer — #266). The symptom here is distinct: the transcript lookup itself fails, not the key-remap seam. The ledgers for affected threads have grown large; see #238 (`~600 KB per launch` full-root `LaunchArtifactBoundary` snapshot) and #237 (8 MiB whole-file cap made threads unresumable, now fixed to per-record cap). Even with #237's per-record cap, a single `launch` row approaches `jsonRecordLimit` (8 MiB) as the machine's transcript count grows, and `muse` sessions that have accumulated many turns produce more observations to snapshot.

Open questions to resolve during fix:
- Is the lookup failing because the `launch` row itself is over `jsonRecordLimit`, because the proof's artifact set references a transcript that has rotated/cleaned up on disk, or because the incremental inventory's stable-id/generation check no longer matches after many turns?
- Why `muse` specifically surfaces this more readily than `claude`/`codex`/`agy` — is it the storage-root size (`muse-sessions` vs `claude/projects`), the scanner schema (`muse-v1`), or the number of observations snapshotted in `prepareRuntimeLaunch`?

## Spec

- Preserve the report verbatim in `Problem` (muse + many turns → transcript not found → relaunch blocked; fresh start works).
- Trace the failing path for `muse`: `prepareRuntimeLaunch` → `LaunchArtifactBoundaries` snapshot → ledger size → `QuerySession`/`ValidateBindingProof` → relaunch/resume. Identify which bound or proof check actually fails after many turns and why the ledger snapshot scales with transcript count.
- Decide the ledger-format fix per #238's brainstorm (candidates: snapshot only the bound root / owner's scope, store out-of-line sidecar, or compact encoding). Old ledgers with full-root snapshots must still parse.
- Keep the fix in the sessionwatch/sessionledger/sessioninventory seam; do not add a `muse`-only workaround in couch/viewer layer.

## Done when

- A long-lived `muse` session (many turns, large `muse-sessions` root) remains resumable/relaunchable; `QuerySession` finds its transcript and returns `BindingEstablished` with the expected `Root`, not `Unbound`/`Ambiguous`/`Stale`.
- A new `launch` row's size is bounded by the artifacts of its own thread/scope, not by the machine's total transcript count (assert with a fake root holding many unrelated transcripts, as required by #238).
- `RoundsAfterLaunch` still delimits the offline window correctly on both old full-root rows and new rows; existing tests pass.
- Regression coverage exists for the large-ledger / many-turns path (ledger over 8 MiB and per-record limit handling).

## Estimate

```estimate
# refined estimate pending plan approval
```

## Plan

- [ ] Reproduce / narrow: log ledger size, `LaunchArtifactBoundaries` length, and `QuerySession` diagnostics after many turns on `muse`; confirm which check fails.
- [ ] Read `RoundsAfterLaunch` and every consumer of `LaunchArtifactBoundaries`; decide minimal snapshot per #238.
- [ ] Brainstorm format choice with operator; record decision in Spec.
- [ ] Implement + compatibility test against a full-root row; measure new row size on real store.
- [ ] Verify relaunch/resume on the previously-failing `muse` session.

## Log

### 2026-09-15

- Recorded from operator report: `muse` after many turns → transcript not found → relaunch blocked; filed as 000267 via `sdlc issue new` (operator: "well, another muse bug, pair can't locate its transcript after many turns. this prevents relaunch it. file a bug on it."). Related to #237/#238 ledger growth.
