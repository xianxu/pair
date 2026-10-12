---
gate: boundary-review
issue: 482
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T07:14:39-07:00"
      dispatched: "2026-10-12T07:12:40-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: replayer.js:171-172 claims animations started by the loads "stop at its start"; they stop after running through settle()
          detail: 'The new animations are paused only after settle(), which waits for the loads plus two animation frames (up to the 5 s bound), so their phase is not their start. Reword the comment, or set currentTime = 0 on the newly started animations. The other claims in this window hold: replayer.js:12-13, atlas/webcast.md:112-114 and the lessons.md entry.'
          family: comment-claim-unverified
          round: 1
        - id: BR-2
          severity: Minor
          title: The 100-run under-load soak result that Done-when requires is not recorded in the Log
          detail: The plan item is ticked, but the Log gives no count or worst score for rec-blog frame 0 across the 100 runs. Add the measured result so the close has evidence for the first Done-when clause.
          family: done-when-evidence-in-log
          round: 1
      recipe: small-diff-review
      isolation: fenced
      reviewed: e3615fff8d824914a6339db22ae989b3c33ecbda
      blocked: false
---

# Gate ledger — pair#482 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T07:14:39-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `comment-claim-unverified` replayer.js:171-172 claims animations started by the loads "stop at its start"; they stop after running through settle()
  The new animations are paused only after settle(), which waits for the loads plus two animation frames (up to the 5 s bound), so their phase is not their start. Reword the comment, or set currentTime = 0 on the newly started animations. The other claims in this window hold: replayer.js:12-13, atlas/webcast.md:112-114 and the lessons.md entry.
- **BR-2** [Minor] `done-when-evidence-in-log` The 100-run under-load soak result that Done-when requires is not recorded in the Log
  The plan item is ticked, but the Log gives no count or worst score for rec-blog frame 0 across the 100 runs. Add the measured result so the close has evidence for the first Done-when clause.

## Open findings

- **BR-1** [Minor] `comment-claim-unverified` replayer.js:171-172 claims animations started by the loads "stop at its start"; they stop after running through settle()
- **BR-2** [Minor] `done-when-evidence-in-log` The 100-run under-load soak result that Done-when requires is not recorded in the Log
