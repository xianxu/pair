---
id: 000098
status: working
deps: []
github_issue:
created: 2026-07-01
updated: 2026-07-05
estimate_hours:
started: 2026-07-05T21:45:50-07:00
---

# context-window count on agent pane stuck (e.g. at 151k)

## Problem

The context-window size shown in the agent-pane frame title — the `(<count>)` in
`<agent> (<count>) [<cwd>]` (#71) — appears **stuck**. Reported: it used to track
down and, when it dropped to low 100k, compaction happened; now it seems frozen
at ~151k and no longer moves.

## Spec

Not yet root-caused — this issue captures the report + the leads for whoever picks
it up.

### Where the number comes from

- `pair-title.sh` → `update_frame_titles()` sets the count via
  `count=$(pair-context "$TAG" "$agent")`
  (`cmd/internal/runtimebundle/assets/runtime/files/bin/pair-title.sh:105`).
- The count itself is computed by `pair-context`
  (`cmd/pair-context/main.go`, `cmd/internal/contextcmd/contextcmd.go`) — it reads
  the agent's live transcript and derives the current context-window size.

### Leads to check

- **Pinned-file assumption (most likely).** `pair-title.sh:178-184` documents that
  claude's `/clear` + compaction keep writing the **same pinned** `…/<sid>.jsonl`
  in place (verified for #71), so the cached path stays valid. If that assumption
  is now violated — e.g. a newer claude version rotates the transcript on
  compaction, or writes a new session file — `pair-context` would keep reading the
  **old** file and report a frozen count. "Stuck right around a compaction boundary
  (151k)" fits a stale-file-handle / stale-path read.
- **Caching:** confirm neither `pair-context` nor the frame updater caches the
  count across compaction (the `agent_file_cache` in pair-title.sh is only for the
  mtime check, but verify `pair-context` re-reads fresh each call).
- **Counting method:** confirm the size is derived from the *current* transcript
  tail, not a monotonic max that can't decrease.

### Reproduce / verify

- Watch `pair-context <tag> claude` directly across a compaction and compare to
  the agent's own reported context size; confirm whether the stuck value tracks a
  stale transcript path.

## Done when

- [ ] Root cause identified (stale transcript path across compaction vs. caching
      vs. counting method).
- [ ] `pair-context` reports a count that moves across `/clear` + compaction,
      matching the agent's real context-window size.
- [ ] Regression coverage for the compaction/rotation case that broke it.

## Plan

- [ ] Reproduce: `pair-context <tag> claude` across a compaction; check the file
      it reads vs. the live transcript.
- [ ] Fix the stale-path / caching / counting bug in `pair-context`
      (+ `contextcmd`), and the pinned-file assumption in `pair-title.sh` if claude
      now rotates.
- [ ] Add a test with a rotated/compacted transcript fixture.

## Log

### 2026-07-01

- Moved from `ariadne#150` (filed in ariadne's inbox; the code lives here in
  pair: `pair-context` + `pair-title.sh`). Reported symptom: count stuck ~151k,
  no longer dropping/tracking compaction. Not yet root-caused — leads captured
  above (pinned-transcript-file assumption from #71 is the prime suspect).
