---
id: 000377
status: open
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: '92849efc0be8e8a4d90dd92b59c34e7eda4a0b44' # card fields mirrored from issue-cards; edit via sdlc
---

# session-watch re-parses whole codex rollout every poll

## Problem

`pair session-watch codex couch-6b111ea230c149dc …` (PID 41981) ran at ~120%
CPU continuously, 241 CPU-minutes over 2.5 days, on 2026-10-01.

Its target is the session's codex rollout
`~/.codex/sessions/2026/09/29/rollout-2026-09-29T07-45-27-…925e.jsonl`
(22.7 MB, still appending ~600 B/s) inside a sessions root of 2,327 files /
4 GB.

A `sample` of the process puts the main loop in
`sessionwatch.Run → incrementalWatcherInventory → ValidateTargetWork →
ObserveStableArtifact → resampleArtifact → OSRuntime.ListFiles
(filepath.WalkDir)`, plus `encoding/json` decode and heavy GC.

Two costs that scale with history instead of new bytes:

1. `ValidateTargetWork` is the *untracked* path: it calls
   `ObserveStableArtifact` with an empty `JSONLFrameState`, i.e. it parses the
   whole file from offset 0. Every sample landing there implies the session's
   target keeps falling out of `tracked` (in
   `incrementalWatcherInventorySnapshot`, `run.go`: either
   `incremental.Select(TargetEstablished…)` reports a size/eligibility mismatch
   or `AdvanceTargetValidation` returns `ErrArtifactChanged`) and is
   re-validated from scratch on every poll (`ActivePoll` = 1 s).
2. `resampleArtifact` (`incremental_inventory.go`) re-lists the entire storage
   root to re-stat one file, and `ObserveStableArtifact` loops on it while the
   file is still growing — so each pass is a full tree walk, possibly several.

Not yet proven: *which* condition drops the tracked target. Hypothesis: an
actively appending rollout trips the fingerprint/stable-EOF checks and is
treated as changed instead of appended.

## Spec

- Steady state for a tracked, appending artifact must cost O(new bytes):
  `AdvanceTargetValidation` from the prior parser offset, never a re-parse
  from 0. Append growth is not "artifact changed".
- Re-sampling one artifact must stat that artifact, not walk the storage root.
- If a target genuinely can't be tracked, back off rather than re-validating
  the full file every poll.

## Done when

- Root cause of the tracked-target drop identified and recorded in the Log
  (reproduced with a test that appends to a rollout between polls).
- Test: N polls over a continuously appending artifact read ~total-appended
  bytes, not N × file size, and do not call `ListFiles` per poll.
- Live check: session-watch on an active codex session stays near idle CPU.

## Plan

- [ ] Reproduce with a fake runtime: growing JSONL across polls; count bytes read + ListFiles calls
- [ ] Identify and fix the tracked-target drop
- [ ] Replace root walk in `resampleArtifact` with a direct stat
- [ ] Live check

## Log

### 2026-10-01

- Found alongside #376 (same session: its `pair wrap` trace log hit 550 MB
  because rotation is blocked). Separate defect; same trigger — a long-lived,
  continuously streaming codex thread.
