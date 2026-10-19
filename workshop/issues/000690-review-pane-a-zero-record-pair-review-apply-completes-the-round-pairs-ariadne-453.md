---
id: 000690
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '2abe07f171465ffbe3f5bb7008e06f82626a5dad' # card fields mirrored from issue-cards; edit via sdlc
---

# review pane: a zero-record pair review apply completes the round (pairs ariadne#453)

## Problem

## Spec

Paired with ariadne#453 (xx-fix always writes the round reply, with an empty records list when the agent has no edits). Found by the pair#664 close review (BR-6): after pair#664 the only thing that clears the review pane's awaiting spinner, and Alt+c's "agent request pending" block, is an applied round artifact. A review the agent answers with no edits writes none, so both stay until the operator's next send. The legacy path cleared them on an empty handoff.

The pane treats a zero-record reply as a completed round: clear the spinner and the pending block, reload nothing, decorate nothing, and the agent commits nothing.

Today `pair review apply` with zero records (or records that change nothing) returns Code ok with no Round, no Body, and writes no round artifact (`sessionsync.ApplyFile`, the `!edit.Changed` early return), so the pane cannot see the reply. The pane also identifies a round by `result_hash` (`round_watch.decide`), which a no-edit round leaves unchanged, so an empty round needs its own identity. Shape to settle in planning: the engine writes an empty-round artifact (`{file, round: {result_hash, spans: [], id}}`, no Body) for a zero-change apply under the same flock, operator lock and hash checks; `round_watch` dedups on the round id rather than the hash; `on_applied_round` skips the reload for an empty round and still runs `after_applied_round`. The JSON reply keeps Body empty, so the protocol says commit nothing.

## Done when

- `pair review apply` with an empty records list (`[]` or `{"records":[]}`) exits 0, writes the file unchanged (no write), and writes an empty-round artifact; the reply has no Body. Go tests cover it, plus the refusals (operator lock exit 3, stale exit 6) still applying to an empty apply.
- The pane, on an empty round: no reload, no decoration, the awaiting spinner and the restore pending block clear. A window-test case fails without the change.
- The rule test still holds: an empty round writes nothing from nvim.
- workshop/targets/review-protocol.md seam 2 names the empty round; ariadne#453 is linked.

## Plan

- [ ]

## Log

### 2026-10-18
