---
id: 000238
status: open
deps: []
github_issue:
created: 2026-09-12
updated: 2026-09-12
estimate_hours:
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

## Spec

Brainstorm before designing — the fix is a ledger-format decision, not a
read-path one. Candidates, in order of preference:
- Record only the boundaries the round window actually consumes: the
  artifacts of the bound root (or the owner's scope), not the whole storage
  root. Check `RoundsAfterLaunch` for what it reads.
- Store the snapshot out of line (one sidecar per launch, referenced by
  ordinal) so the ledger stays a small index.
- Compact the boundary encoding.
Whichever lands, old ledgers with full-root snapshots must still parse.

## Done when

- A launch row's size is bounded by the artifacts of its own thread, not by
  the machine's transcript count; assert with a fake root holding many
  unrelated transcripts.
- `RoundsAfterLaunch` still delimits the offline window correctly (existing
  tests), on both old full-root rows and new rows.
- The operator's ledger, after one new launch, grows by kilobytes, not
  hundreds of kilobytes.

## Plan

- [ ] Read `RoundsAfterLaunch` and every consumer of `LaunchArtifactBoundaries`; decide the smallest snapshot that satisfies them
- [ ] Brainstorm the format choice with the operator; record it in the Spec
- [ ] Implement + compatibility test against a full-root row
- [ ] Measure a new launch row on the real store

## Log

### 2026-09-12

- Filed while fixing #237 from the row profile of the operator's ledger: 14 launch rows × ~600 KB. Snapshot source traced to `sessionwatch.prepareRuntimeLaunch`.
- From #237's close review: `OSRuntime.ReadAt` opens the file per 64 KiB
  chunk, so one owner query on this 8.5 MB ledger is ~130 opens, multiplied
  by `titlepoller`'s cadence. The row shrink is the bound on that too; if it
  is not enough, the reader keeping one handle across chunks is the next
  lever.
