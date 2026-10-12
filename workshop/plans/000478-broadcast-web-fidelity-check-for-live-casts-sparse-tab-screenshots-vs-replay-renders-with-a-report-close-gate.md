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

## Open findings

- **BR-1** [Important] `replay-continuation-state` Replayer.replayAt finishes paused animations on a clipped (continued) window, so retries and close frames are drawn at the wrong phase
- **BR-2** [Important] `docs-gate-readme` README update appears missing for couch --broadcast-web-fidelity [ID|DIR]
- **BR-3** [Minor] `fail-fast-on-dead-peer` headless.open polls for 30 s after Chrome has died instead of failing on the closed CDP connection
- **BR-4** [Minor] `untrusted-persisted-input` CheckFidelity trusts frame records from disk: File joined unchecked, W/H/Scale unbounded
- **BR-5** [Minor] `shared-helper-duplication` replay_test.go headlessPage repeats the launchHeadless pipe/launch sequence now in product code
- **BR-6** [Minor] `residue-without-sweep` fidelity.Write's .fidelity-* temp dir inside the recording is left behind on a crash mid-write
- **BR-7** [Minor] `unbounded-task-extent` The end-of-cast check goroutine runs on context.Background with no cancellation path
