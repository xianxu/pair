---
gate: boundary-review
issue: 472
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T04:12:42-07:00"
      dispatched: "2026-10-12T04:10:48-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: recorder/side.go:413 segT clamp (frame never before its segment) has no test; removing it leaves all recorder tests green
          detail: Mutation f.SetTime(r.t(d.at)) passes go test ./cmd/internal/webcast/recorder. Add a test, like TestRecorderFrameKeepsItsCaptureTime, that starts a new segment inside the fake captureScreenshot and asserts f.T equals the segment's t and the result validates. The same family includes the segT assignments at recorder.go:679 and :724, which only this path reaches.
          family: new-guard-without-regression-test
          round: 1
        - id: BR-2
          severity: Minor
          title: Fixture-name lists and viewer viewport sizing are restated by hand
          detail: harness.js:865-868 mirrors viewer.js:114-119 sizing. Fixture names are hand-listed in recording.Fixtures, PrototypeFixtures, fidelity_test.go:106 (could filter rec.Fixtures by frames) and fixture_test.go want. Retirement (#474) touches all of them.
          family: hand-maintained-restatement
          round: 1
        - id: BR-3
          severity: Minor
          title: A screenshot that straddles a segment change is stamped at the new segment's start but shows the old page
          detail: 'This predates the change (the old st.T clamp did the same). Worth noting on #475 or in the format''s frame-t note.'
          family: segment-straddling-shot
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T04:15:17-07:00"
      dispatched: "2026-10-12T04:13:46-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: side.go:413 clamp and recorder.go:724 are now pinned (mutation-checked red); recorder.go:679 (snapshotTop, resume/tab-switch) segT assignment removed still leaves go test ./cmd/internal/webcast/recorder green — add a pause→resume-inside-screenshot test.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: Deferred to 474, whose Spec covers PrototypeFixtures/fidelity/coverage lists but not harness.js's copy of viewer.js viewport sizing; add it there.
          round: 2
        - id: BR-3
          disposition: addressed
          note: atlas/webcast.md now states a straddling call is stamped at the new segment's start though its pixels may show either page.
          round: 2
      findings:
        - id: BR-4
          severity: Minor
          title: replay_test frameTimes compares a frame at t-captureSkew / t+lag without staying in its own segment
          detail: 'This is the 2nd finding in family segment-straddling-shot. Rule: any time offset applied to a frame''s t must stay inside the frame''s own segment. fidelity/run.go enforces it inline (st.Seg != own.Seg); replay_test.go frameTimes (t-50, t+150) does not, so a segT-clamped frame can be compared against the previous segment''s page (a false pass). Instances in window: fidelity/run.go Offsets loop (guarded), replay_test.go frameTimes (unguarded). Fix the rule once: a shared recording helper filtering offsets by segment, used by both.'
          family: segment-straddling-shot
          round: 2
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-12T04:26:22-07:00"
      dispatched: "2026-10-12T04:25:15-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Table test covers navigate/resume/tab; I removed side.go:413, recorder.go:679 and recorder.go:724 one at a time and each fails TestRecorderFrameNeverBeforeItsSegment.
          round: 3
        - id: BR-2
          disposition: addressed
          note: 'Minor; deferred with its trigger to #474, whose Spec names the harness.js sizing copy and the hand-listed fixture names, the change that rewrites all of them.'
          round: 3
        - id: BR-4
          disposition: addressed
          note: recording.FrameTimes (fixture.go:65) is the single rule, unit-tested in TestFrameTimesStayInTheFramesSegment and used by both fidelity/run.go and replay_test.go.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: f82bd66c96e66f77ff6764b794bb001604de96f2
      blocked: false
---

# Gate ledger — pair#472 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T04:12:42-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `new-guard-without-regression-test` recorder/side.go:413 segT clamp (frame never before its segment) has no test; removing it leaves all recorder tests green
  Mutation f.SetTime(r.t(d.at)) passes go test ./cmd/internal/webcast/recorder. Add a test, like TestRecorderFrameKeepsItsCaptureTime, that starts a new segment inside the fake captureScreenshot and asserts f.T equals the segment's t and the result validates. The same family includes the segT assignments at recorder.go:679 and :724, which only this path reaches.
- **BR-2** [Minor] `hand-maintained-restatement` Fixture-name lists and viewer viewport sizing are restated by hand
  harness.js:865-868 mirrors viewer.js:114-119 sizing. Fixture names are hand-listed in recording.Fixtures, PrototypeFixtures, fidelity_test.go:106 (could filter rec.Fixtures by frames) and fixture_test.go want. Retirement (#474) touches all of them.
- **BR-3** [Minor] `segment-straddling-shot` A screenshot that straddles a segment change is stamped at the new segment's start but shows the old page
  This predates the change (the old st.T clamp did the same). Worth noting on #475 or in the format's frame-t note.

## Round 2 — 2026-10-12T04:15:17-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — side.go:413 clamp and recorder.go:724 are now pinned (mutation-checked red); recorder.go:679 (snapshotTop, resume/tab-switch) segT assignment removed still leaves go test ./cmd/internal/webcast/recorder green — add a pause→resume-inside-screenshot test.
- BR-2 — not-addressed — Deferred to 474, whose Spec covers PrototypeFixtures/fidelity/coverage lists but not harness.js's copy of viewer.js viewport sizing; add it there.
- BR-3 — addressed — atlas/webcast.md now states a straddling call is stamped at the new segment's start though its pixels may show either page.

### Raised

- **BR-4** [Minor] `segment-straddling-shot` replay_test frameTimes compares a frame at t-captureSkew / t+lag without staying in its own segment
  This is the 2nd finding in family segment-straddling-shot. Rule: any time offset applied to a frame's t must stay inside the frame's own segment. fidelity/run.go enforces it inline (st.Seg != own.Seg); replay_test.go frameTimes (t-50, t+150) does not, so a segT-clamped frame can be compared against the previous segment's page (a false pass). Instances in window: fidelity/run.go Offsets loop (guarded), replay_test.go frameTimes (unguarded). Fix the rule once: a shared recording helper filtering offsets by segment, used by both.

## Round 3 — 2026-10-12T04:26:22-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Table test covers navigate/resume/tab; I removed side.go:413, recorder.go:679 and recorder.go:724 one at a time and each fails TestRecorderFrameNeverBeforeItsSegment.
- BR-2 — addressed — Minor; deferred with its trigger to #474, whose Spec names the harness.js sizing copy and the hand-listed fixture names, the change that rewrites all of them.
- BR-4 — addressed — recording.FrameTimes (fixture.go:65) is the single rule, unit-tested in TestFrameTimesStayInTheFramesSegment and used by both fidelity/run.go and replay_test.go.

## Open findings

(none — every finding has been disposed)
