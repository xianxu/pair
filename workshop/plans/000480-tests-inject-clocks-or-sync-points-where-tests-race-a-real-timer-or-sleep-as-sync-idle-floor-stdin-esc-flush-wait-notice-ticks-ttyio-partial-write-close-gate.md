---
gate: boundary-review
issue: 480
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T06:51:00-07:00"
      dispatched: "2026-10-12T06:45:46-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: ttyio poll-count assertions still have a small wall-clock margin
          detail: 'writer_test.go:76 sizes the busy-poll limit from `progressed`, but the reader goroutine''s polls started earlier. writer_test.go:285 reads `before` before cancel(), so a gap longer than one pollSlice gives after > before+1. Both instances are in this window. Fix: take the start time before launching the reader goroutine, and read `before` and call cancel back to back (or allow before+2).'
          family: residual-wall-time-window
          round: 1
        - id: BR-2
          severity: Minor
          title: idle-floor comment says a chunk on stdout means the bump already ran
          detail: idle_floor_test.go:205-207. handleChunk queues to the async stdout pump, and bumpIdleDeadline runs after it (wrap.go:3263-3266), so the pump can write first. The test is still sound because the previous chunk's bump has finished; only the comment needs correcting.
          family: comment-claim-accuracy
          round: 1
      recipe: small-diff-review
      isolation: fenced
      reviewed: 1fc4a3fbff3cf5e41eeee0c0e94a90748b3a4452
      blocked: false
    - "n": 2
      timestamp: "2026-10-12T06:53:59-07:00"
      dispatched: "2026-10-12T06:52:42-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: started now precedes both goroutines (writer_test.go:44); cancel() precedes reading before (writer_test.go:284-285), exact given stopped() runs before polls.Add; 20x -race at -cpu=1,4 passes
          round: 2
        - id: BR-2
          disposition: addressed
          note: idle_floor_test.go:205-208 now says the bump follows the read in the same loop step, before the next chunk is read, matching wrap.go:3262-3265
          round: 2
      recipe: fix-review
      isolation: fenced
      reviewed: 8e94173bce9deb53ae2ff5e0548ce5640c968a50
      blocked: false
---

# Gate ledger — pair#480 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T06:51:00-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `residual-wall-time-window` ttyio poll-count assertions still have a small wall-clock margin
  writer_test.go:76 sizes the busy-poll limit from `progressed`, but the reader goroutine's polls started earlier. writer_test.go:285 reads `before` before cancel(), so a gap longer than one pollSlice gives after > before+1. Both instances are in this window. Fix: take the start time before launching the reader goroutine, and read `before` and call cancel back to back (or allow before+2).
- **BR-2** [Minor] `comment-claim-accuracy` idle-floor comment says a chunk on stdout means the bump already ran
  idle_floor_test.go:205-207. handleChunk queues to the async stdout pump, and bumpIdleDeadline runs after it (wrap.go:3263-3266), so the pump can write first. The test is still sound because the previous chunk's bump has finished; only the comment needs correcting.

## Round 2 — 2026-10-12T06:53:59-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — started now precedes both goroutines (writer_test.go:44); cancel() precedes reading before (writer_test.go:284-285), exact given stopped() runs before polls.Add; 20x -race at -cpu=1,4 passes
- BR-2 — addressed — idle_floor_test.go:205-208 now says the bump follows the read in the same loop step, before the next chunk is read, matching wrap.go:3262-3265

## Open findings

(none — every finding has been disposed)
