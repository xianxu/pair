---
id: 000238
status: open
created: 2026-09-12
updated: 2026-09-12
estimate_hours:
github_issue:
---

# every ledger launch record embeds a boundary snapshot of the whole storage root, ~600 KB per launch

## Problem

Measured on the operator's store, 2026-09-12: `ledger-couch-5003fd4f6c74f514.jsonl`
is 8,482,765 bytes for 39 rows. Its 14 `launch` rows carry 8.47 MB between
them — about 600 KB each, the longest 677,523 bytes — while the 14 `binding`
rows total 10 KB.

Each launch row is that large because `sessionwatch.prepareRuntimeLaunch`
(`lifecycle.go:280`) snapshots a `LaunchArtifactBoundary` for **every
observation in the agent's storage root** — every transcript under
`~/.claude/projects`, all repos, ~1,000 files — and embeds the list in the
record. The snapshot exists so the offline round window after the launch can
be delimited (`RoundsAfterLaunch`), but it is a full-root inventory copied
into a per-thread ledger on every launch, growing with both the number of
transcripts on the machine and the number of relaunches.

Consequences:
- The ledger crossed the query's 8 MiB whole-file cap and the thread became
  unresumable (#237, which fixes the read bound but not the size).
- A single launch row is already 8% of `jsonRecordLimit` (8 MiB per record);
  a machine with ~12k transcripts would produce a launch row the reader must
  refuse.
- Every binding query, resume, relaunch and full inventory parses megabytes
  of boundaries it does not use.
