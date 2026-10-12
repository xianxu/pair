---
gate: boundary-review
issue: 478
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T04:37:47-07:00"
      dispatched: "2026-10-12T04:34:19-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Replayer.replayAt finishes paused animations on a clipped (continued) window, so retries and close frames are drawn at the wrong phase
          detail: 'With start = this.at, settle() is followed by finishAnimations(), which jumps animations paused at the previous t to their end. The -150/+150/+300 retries in headless.score, and any frame less than 1 s after the previous one, are drawn with transitions completed rather than at their phase. A backward retry also resets and re-feeds the whole recording, so over-tolerance frames cost O(records) each, breaking the plan''s per-frame envelope. Fix: play() paused animations when continuing, and add a node test for it.'
          family: replay-continuation-state
          round: 1
        - id: BR-2
          severity: Important
          title: README update appears missing for couch --broadcast-web-fidelity [ID|DIR]
          detail: The new CLI flag and its exit codes are in run.go usage and the atlas but not in README.md, where the sibling --broadcast-web-replay and --broadcast-web-query flags are documented (around lines 401-405 and 866-872).
          family: docs-gate-readme
          round: 1
        - id: BR-3
          severity: Minor
          title: headless.open polls for 30 s after Chrome has died instead of failing on the closed CDP connection
          family: fail-fast-on-dead-peer
          round: 1
        - id: BR-4
          severity: Minor
          title: 'CheckFidelity trusts frame records from disk: File joined unchecked, W/H/Scale unbounded'
          family: untrusted-persisted-input
          round: 1
        - id: BR-5
          severity: Minor
          title: replay_test.go headlessPage repeats the launchHeadless pipe/launch sequence now in product code
          family: shared-helper-duplication
          round: 1
        - id: BR-6
          severity: Minor
          title: fidelity.Write's .fidelity-* temp dir inside the recording is left behind on a crash mid-write
          family: residue-without-sweep
          round: 1
        - id: BR-7
          severity: Minor
          title: The end-of-cast check goroutine runs on context.Background with no cancellation path
          family: unbounded-task-extent
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T04:45:00-07:00"
      dispatched: "2026-10-12T04:43:25-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: replayer.js continue branch play()s paused anims; backward seeks via catchUp. Both new node tests fail with the pre-fix replayer.js restored and pass with the fix.
          round: 2
        - id: BR-2
          disposition: addressed
          note: README.md usage block and prose section now cover --broadcast-web-fidelity, its report path and exit codes 0/3/1, matching run.go.
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: headless.open now selects on h.conn.Done(), which is correct by inspection, but no test fails without it.
          round: 2
        - id: BR-4
          disposition: addressed
          note: frameViewport validates File, W, H and Scale before the read; TestFrameViewport covers the escape and bounds cases.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: replay_test.go headlessPage still repeats launchHeadless's pipe/exec sequence; untouched this round.
          round: 2
        - id: BR-6
          disposition: addressed
          note: Write sweeps stale .fidelity-* dirs; TestReportFiles plants .fidelity-crashed and asserts none are left.
          round: 2
        - id: BR-7
          disposition: not-addressed
          note: cast.go checkRecording still runs c.check(context.Background(), ...) in an untracked goroutine; untouched this round.
          round: 2
      findings:
        - id: BR-8
          severity: Minor
          title: fidelity.Write sweeps every .fidelity-* dir, including one a concurrent check of the same recording is still writing
          detail: 'report.go:157-161 deletes all .fidelity-* dirs before MkdirTemp. If the end-of-cast check and a manual couch --broadcast-web-fidelity run on one recording at once, the second deletes the first one''s in-progress dir and the first Write fails. Fix: sweep only dirs older than some age, or take a lock file in the recording dir.'
          family: sweep-claims-unowned-work
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 8962e0777022f69014a7512cc676181e952b0336
      blocked: false
    - "n": 3
      timestamp: "2026-10-12T05:07:08-07:00"
      dispatched: "2026-10-12T05:05:17-07:00"
      agent: claude
      dispose:
        - id: BR-3
          disposition: not-addressed
          note: Untouched in this window; headless.open has no test that fails without the conn.Done() select. Minor, non-blocking.
          round: 3
        - id: BR-5
          disposition: not-addressed
          note: replay_test.go headlessPage still repeats launchHeadless's pipe/exec sequence. Minor, non-blocking.
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: The end-of-cast check still runs on context.Background in an untracked goroutine. Minor, non-blocking.
          round: 3
        - id: BR-8
          disposition: addressed
          note: report.go sweeps only dirs older than staleWrite (1 min); TestReportFiles keeps a fresh busy dir and removes an aged one, and would fail under the old sweep-all code.
          round: 3
      recipe: fix-review
      isolation: fenced
      reviewed: 5f85ad2640e59a1bc7eb20f6ca83df0016821a75
      blocked: false
---

# Gate ledger — pair#478 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T04:37:47-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `replay-continuation-state` Replayer.replayAt finishes paused animations on a clipped (continued) window, so retries and close frames are drawn at the wrong phase
  With start = this.at, settle() is followed by finishAnimations(), which jumps animations paused at the previous t to their end. The -150/+150/+300 retries in headless.score, and any frame less than 1 s after the previous one, are drawn with transitions completed rather than at their phase. A backward retry also resets and re-feeds the whole recording, so over-tolerance frames cost O(records) each, breaking the plan's per-frame envelope. Fix: play() paused animations when continuing, and add a node test for it.
- **BR-2** [Important] `docs-gate-readme` README update appears missing for couch --broadcast-web-fidelity [ID|DIR]
  The new CLI flag and its exit codes are in run.go usage and the atlas but not in README.md, where the sibling --broadcast-web-replay and --broadcast-web-query flags are documented (around lines 401-405 and 866-872).
- **BR-3** [Minor] `fail-fast-on-dead-peer` headless.open polls for 30 s after Chrome has died instead of failing on the closed CDP connection
- **BR-4** [Minor] `untrusted-persisted-input` CheckFidelity trusts frame records from disk: File joined unchecked, W/H/Scale unbounded
- **BR-5** [Minor] `shared-helper-duplication` replay_test.go headlessPage repeats the launchHeadless pipe/launch sequence now in product code
- **BR-6** [Minor] `residue-without-sweep` fidelity.Write's .fidelity-* temp dir inside the recording is left behind on a crash mid-write
- **BR-7** [Minor] `unbounded-task-extent` The end-of-cast check goroutine runs on context.Background with no cancellation path

## Round 2 — 2026-10-12T04:45:00-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — replayer.js continue branch play()s paused anims; backward seeks via catchUp. Both new node tests fail with the pre-fix replayer.js restored and pass with the fix.
- BR-2 — addressed — README.md usage block and prose section now cover --broadcast-web-fidelity, its report path and exit codes 0/3/1, matching run.go.
- BR-3 — not-addressed — headless.open now selects on h.conn.Done(), which is correct by inspection, but no test fails without it.
- BR-4 — addressed — frameViewport validates File, W, H and Scale before the read; TestFrameViewport covers the escape and bounds cases.
- BR-5 — not-addressed — replay_test.go headlessPage still repeats launchHeadless's pipe/exec sequence; untouched this round.
- BR-6 — addressed — Write sweeps stale .fidelity-* dirs; TestReportFiles plants .fidelity-crashed and asserts none are left.
- BR-7 — not-addressed — cast.go checkRecording still runs c.check(context.Background(), ...) in an untracked goroutine; untouched this round.

### Raised

- **BR-8** [Minor] `sweep-claims-unowned-work` fidelity.Write sweeps every .fidelity-* dir, including one a concurrent check of the same recording is still writing
  report.go:157-161 deletes all .fidelity-* dirs before MkdirTemp. If the end-of-cast check and a manual couch --broadcast-web-fidelity run on one recording at once, the second deletes the first one's in-progress dir and the first Write fails. Fix: sweep only dirs older than some age, or take a lock file in the recording dir.

## Round 3 — 2026-10-12T05:07:08-07:00 (claude) — passed

### Disposed

- BR-3 — not-addressed — Untouched in this window; headless.open has no test that fails without the conn.Done() select. Minor, non-blocking.
- BR-5 — not-addressed — replay_test.go headlessPage still repeats launchHeadless's pipe/exec sequence. Minor, non-blocking.
- BR-7 — not-addressed — The end-of-cast check still runs on context.Background in an untracked goroutine. Minor, non-blocking.
- BR-8 — addressed — report.go sweeps only dirs older than staleWrite (1 min); TestReportFiles keeps a fresh busy dir and removes an aged one, and would fail under the old sweep-all code.

## Open findings

- **BR-3** [Minor] `fail-fast-on-dead-peer` headless.open polls for 30 s after Chrome has died instead of failing on the closed CDP connection
- **BR-5** [Minor] `shared-helper-duplication` replay_test.go headlessPage repeats the launchHeadless pipe/launch sequence now in product code
- **BR-7** [Minor] `unbounded-task-extent` The end-of-cast check goroutine runs on context.Background with no cancellation path
