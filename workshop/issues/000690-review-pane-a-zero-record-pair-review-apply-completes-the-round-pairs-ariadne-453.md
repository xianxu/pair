---
id: 000690
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'ed9b2db0ad43f113b31a793800122f7f717b2dbc' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T02:30:24-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
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
- workshop/targets/review-protocol.md seam 2 names the empty round contract (the producer always applies, records [] when it has no edits; an empty reply commits nothing); ariadne#453 is linked.
- Live no-edits pane check, after this and ariadne#453 have both landed: a review the agent answers with no edits clears the spinner and the pending block in a live pane.

## Plan

Design (quick flow). A round is identified by (result_hash, at), not result_hash alone: `at` already exists and differs for every apply, so the empty round needs no new field (ARCH-EVOLVE: the artifact format does not change). An older pane dedups on result_hash and ignores an empty round, which is today's behavior. An empty round is a Round with no spans whose base_hash equals result_hash.

- [ ] Engine, `sessionsync.ApplyFile`: an empty records list skips BuildEdit (which refuses empty input for the Couch path, unchanged). A zero-change apply, from an empty list or from records that change nothing, writes the empty-round artifact `{file, round: {base_hash = result_hash = H, at, presentation, spans: []}, body: ""}` after the same flock, operator lock and hash checks. The previous artifact is removed first, the file is not written, and the reply carries the Round with no Body. The previous `!edit.Changed` early return goes away.
- [ ] Pane, `round_watch`: `M.key(artifact)` returns result_hash plus at; decide, watch and the declined-round warning all key on it (ARCH-ORDER: one identity for dedup and redelivery).
- [ ] Pane, `init.lua on_applied_round`: for an empty round, skip the modified check and the reload/decorate (there is nothing to reload, so unsaved operator text is safe), and still run `after_applied_round`, which clears `awaiting_since`, the spinner and the restore pending block.
- [ ] Tests:
  - Go: empty `[]` and `{"records":[]}` plus a no-op record each write the artifact and leave the file bytes unchanged; operator lock (exit 3) and stale (exit 6) still refuse an empty apply; the CLI JSON reply has a Round and no Body.
  - Lua: round_watch decide accepts the same hash with a new `at`, and rejects the same key.
  - Window test: an empty round clears the spinner without reloading, and fails without the change.
  - The rule test keeps the empty round case: nvim writes nothing.
- [ ] Target: review-protocol.md seam 2 and the applying state name the empty round (the producer always applies, records [] when it has no edits; an empty Body commits nothing); link ariadne#453; add a Revisions entry.
- [ ] Atlas review page, if it describes the round artifact.
- [ ] Live no-edits pane check: the TL holds it for the next Couch restart.

ARCH-FUNERAL: no new artifact; the one round file is replaced on every apply, as before. ARCH-SECURE: the pane already decodes the artifact defensively (`decide` type checks); `at` is read as an opaque string.

## Log

### 2026-10-18

### 2026-10-19

Hand-off: the hub split the empty-round work. ariadne#453 (landed as ariadne PR 316) is the producer side only: shared xx-fix always calls pair review apply, with records [] when it has no edits, and an empty reply never makes the agent commit. Two pieces moved here from ariadne:3: the empty-round contract wording in workshop/targets/review-protocol.md, and the live no-edits pane check once both issues have landed. ariadne:3 records the same hand-off in ariadne#453.
