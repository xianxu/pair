# Boundary Review — pair#752 (whole-issue close)

| field | value |
|-------|-------|
| issue | 752 — cast-sketch: live console and driver drawing with pen pressure |
| repo | pair |
| issue file | workshop/issues/000752-cast-sketch-drawing.md |
| boundary | whole-issue close |
| milestone | — |
| window | bc79cbaeae75b518dcfa789fc2bd0e40ccf06f97..60a400cb13be0e54e0469e36ce5d3805461cc07c |
| command | sdlc close --issue 752 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T02:55:07-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation delivers both drawing surfaces, strict ingress, ownership fencing, bounded delivery, and README/atlas documentation. Existing automated tests pass, but two reproduced UI failures block acceptance: interrupted console strokes can become impossible to label, and queue saturation can strand an open stroke or crash pointer-up. The plan’s entity map also needs correction. Live Chrome conformance could not run successfully in this environment.

```findings
findings:
  - id: new
    severity: Critical
    family: authoritative-state-reaches-ui
    title: |
      Console handoff leaves partial-stroke label saving permanently blocked
    detail: |
      cmd/internal/webcast/web/sketch.js:470 refuses label saving while cached status.open_stroke is populated, but its lifecycle refresh at :663 requires model.sketchMetadata and actual ctl/stop/sketch-interrupt records. The console model in operator.js:173 has no such query, publishes synthetic view records, and its sketch journal receiver only refreshes painting. Reproduced an accepted open stroke followed by handoff and authoritative sealing: labelRequests remained zero with “Wait for the stroke confirmation before saving its label.” ARCH-ORDER/ARCH-PURPOSE: feed authoritative sketch metadata and lifecycle transitions into the console tool, including handoff, pause, expiry, clear and end; add an interrupted-contact console regression.
  - id: new
    severity: Critical
    family: bounded-input-retains-terminal-work
    title: |
      Queue saturation cancels the gesture without sealing it and can crash pointer-up
    detail: |
      cmd/internal/webcast/web/sketch-policy.js:207 emits only stop-gesture when sixteen point chunks are queued; sketch.js handles that by canceling the gesture and clearing contact without retaining terminal work or entering visible recovery. Holding the first points reply and filling the queue reproduced a drained queue with the server stroke still open, zero stroke-end requests, and no status message; when saturation occurs inside finish() at sketch.js:322, line 324 throws “Cannot read properties of null (reading 'pointer')”. ARCH-CONSTRAINTS/ARCH-ORDER: preserve a truncated terminal operation behind accepted queued work, expose reconciliation, and make finish safe against synchronous cancellation.
  - id: new
    severity: Critical
    family: core-concept-map-matches-code
    title: |
      The Core concepts table names nonexistent and unchanged implementation locations
    detail: |
      workshop/plans/000752-cast-sketch-drawing-plan.md:50 names board/sketch_authority.go, which does not exist; SketchAuthority is in board/sketch.go:14. Rows 52–58 also claim modifications to recording/hub.go, browser.go, server_drive.go, web/annotation-placement.js, recorder/operator.go, board/operator.go, fake.go and fake_drive.go that are absent from the pinned diff. Per the required Core concepts cross-check, reconcile the complete table with actual files and statuses, including sketch_source.go, cast_sketch.go and writer_sketch.go, and record the correction under Revisions.
  - id: new
    severity: Important
    kind: unverifiable
    family: live-conformance-evidence
    title: |
      Required Chrome conformance assertions could not execute
    detail: |
      Attempted WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'SketchLive|OperatorSketchLive' -count=1 -v. Both operator tests failed before opening their page with GPU-process/sandbox initialization failures, and TestSketchLiveTwoViewers failed with CDP EOF. Supply successful host results for these tests on the reviewed implementation, or provide an environment where Chrome starts, to verify private-scene isolation and real two-viewer/console behavior.
```

1. **Strengths**

   - Driving permits fence stale turns before control settlement; held-write, release-failure and ABA tests exercise this behavior.
   - Source preparation captures an exact ordered prefix and refuses independent prefix/state limits before committing begin.
   - Additive truncation has codec, semantic-identity, snapshot, cancellation and legacy-preservation coverage.
   - README and atlas explain controls, recovery, limits and private-scene architecture.

2. **Critical findings**

   The three Critical findings above require two behavioral fixes with regression tests and a correction to the plan’s implementation map.

3. **Important findings**

   Live Chrome conformance needs executable host evidence. The separately documented TL console smoke remains pending before landing; this review does not treat it as completed.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - `go test ./cmd/internal/strictjson ./cmd/internal/webcast/...`
   - All 388 Node webcast tests.
   - Race-enabled sketch/drawing tests for Board, drive and recorder.
   - `git diff --check`.

   Read-only Node probes using the existing harness reproduced both behavioral findings. Current handoff coverage does not exercise a console contact interrupted while its stroke remains open; queue tests do not cover saturation during pointer-up.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass: shared tool, renderer, protocol and writer seams are reused. |
   | ARCH-PURE | Pass: parser, policy and projection logic are separated from transport and DOM effects. |
   | ARCH-PURPOSE | Flag: interrupted console labeling fails a required workflow. |
   | ARCH-MOCK | Pass structurally: stateful writer/controller doubles share production seams; live conformance remains unverifiable here. |
   | ARCH-CONSTRAINTS | Flag: queue capacity is enforced without safe terminal recovery. |
   | ARCH-SECURE | Isolation structure inspected; actual Chrome privacy assertion remains unverifiable here. |
   | ARCH-ORDER | Flag: authoritative sealing does not reach console policy/status, and cancellation invalidates state during `finish()`. |
   | ARCH-FUNERAL | Pass: source leases, transfer handles, blobs and workers have retirement owners. |
   | ARCH-EVOLVE | Pass: optional truncation preserves absent/false semantics and unknown-field round trips. |

7. **Plan revision recommendations**

   Add `## Revisions` entries that:

   - Correct every Core concepts path/status against the pinned implementation.
   - Describe authoritative console metadata delivery and its interrupted-contact regression.
   - Specify queue-full terminal sealing and pointer-up recovery tests.
   - Record successful host conformance evidence while retaining the pre-landing console-smoke hold.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"49c4f4f9-2fc4-41fe-9ee7-c8d91eed8d50","repository":"github.com/xianxu/pair","issue_id":"000752","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/c1af59ffccdebf8f","reviewed_head":"60a400cb13be0e54e0469e36ce5d3805461cc07c","slot":"pair:9","requested_at":"2026-10-21T09:50:14.707355Z"},"phase":"finished","pid":59176,"started_at":"2026-10-21T09:50:14.802301Z","finished_at":"2026-10-21T09:55:06.46759Z","session_id":"01a15dde-581e-7a91-ad27-a35d3046dd27","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.920169125s
head: "60a400cb13be0e54e0469e36ce5d3805461cc07c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
