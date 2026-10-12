# Boundary Review — pair#478 (whole-issue close)

| field | value |
|-------|-------|
| issue | 478 — broadcast-web: fidelity check for live casts — sparse tab screenshots vs replay renders, with a report |
| repo | pair |
| issue file | workshop/issues/000478-broadcast-web-fidelity-check-for-live-casts-sparse-tab-screenshots-vs-replay-renders-with-a-report.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1da5c76c60b687fc25a90cb1dc9f1268daa9172e..575368d85f062ef95f5d1de011837084fe51e264 |
| command | sdlc close --issue 478 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T04:37:47-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the #478 diff: I've read the core Go and JS. Next I'm checking the couchcmd wiring, the tests, and the atlas.

Still reviewing: the retry logic in the replayer has a phase bug. Now checking how frame records from disk are validated.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The issue's purpose is delivered, and the architecture is good. A pure `webcast/fidelity` package does the scoring, report and crops. A thin `CheckFidelity` shell drives the viewer's own renderer through a shared `Replayer`. Cast gets a `Check` hook that couchcmd wires up, and there is a CLI that returns exit codes. The pure Go tests and the node tests pass here, and so do the hook and replay-server tests.

Two things should be fixed before shipping:

- **Retry and close-frame replays show the wrong animation phase.** When the replayer continues from what it has already drawn, it finishes every running animation instead of resuming it. So three of the four ±150/±300 ms retries, and any frame less than 1 s after the previous one, are drawn with transitions completed rather than at their phase. The documented "replayed at ±150/±300 ms" doesn't hold, and a mismatch in transition phase can be scored as a match.
- **README.md doesn't document `--broadcast-web-fidelity`.** The sibling `--broadcast-web-replay` and `--broadcast-web-query` flags are documented there.

Confidence is medium because I couldn't run two things:
- **Live Chrome tests:** headless Chrome exited with `cdp: read: EOF` in this sandbox, so the Done-when measurements (live blog cast within tolerance, broken render caught) are unverified here.
- **couchcmd tests:** the package wouldn't build because generated runtime assets are missing (`runtimebundle/embed.go: pattern assets/runtime/files: no matching files found`).

**1. Strengths**
- The pure/shell split is clean. `fidelity/` (diff, `Summarize`, `DiffBox`, `SideBySide`, `Markdown`) has no Chrome dependency and is unit-tested directly (`fidelity/fidelity_test.go`). The shell only produces PNGs and shots.
- The DRY consolidation is real (ARCH-DRY). The harness's `replayAt` became product code in `web/replayer.js`, and both the #460 harness (`testdata/replay/harness.js`) and `fidelity.js` use it. The old static Go renderer, which was not what viewers see, is deleted.
- Memory and residue are bounded:
  - `keepWorst` (`fidelity_check.go:779`) caps the replays held in memory at `MaxWorst`, and `TestKeepWorstBoundsReplays` tests it.
  - `fidelity.Write` replaces the report directory whole, and a test shows no stale crop survives a rerun.
  - The Chrome profile is removed in `close`.
- Cast never imports the check. The hook is a `func` on `CastConfig`, with a process-wide one-at-a-time slot (`fidelitySlot`). Both are tested with fakes (`TestCastChecksItsRecordingWhenItEnds`, `TestFidelityHookOneAtATime`).
- The report keeps `offset` and `at_t`, so a timing slip stays visible rather than being silently absorbed.

**2. Critical findings**
None.

**3. Important findings**
- **The replayer's continuation finishes animations instead of resuming them** (`cmd/internal/webcast/web/replayer.js`, `replayAt`). When `this.at > t - windowMs`, the window is clipped to `start = this.at`. The code then calls `settle()` and `finishAnimations()`, which jumps every animation that was paused at the previous `t` to its end state.
  - In `score` (`fidelity_check.go:966`) the retries run in the order 0, −300, −150, +150, +300. The −300 retry resets and gets a full window. The −150, +150 and +300 retries are clipped continuations, so a 1 s fade that started at t−800 is drawn complete at t+150 instead of about 95% through.
  - The same applies to any real frame less than 1 s after the previous one, for example a segment-start frame close to a 5 s tick.
  - Fix sketch: when continuing (`start === this.at`), `play()` the paused animations rather than `finish()` them. Add a node test with the fake clock showing that a continued `replayAt` resumes and does not finish.
  - This also breaks the plan's "a recording is replayed once" claim: every over-tolerance frame resets and re-feeds the recording from the start. A fully broken cast costs O(frames × records). The plan's envelope (~1.5 s per frame) doesn't hold for over-tolerance frames, which take about 2.6 s of real-time windows plus a full re-feed (ARCH-CONSTRAINTS). One option is to run the retries forward from t−300 with a single full window.
- **README update appears missing for `couch --broadcast-web-fidelity [ID|DIR]`.** The new flag and its exit codes (0 within tolerance, 3 over, 1 on error) are documented in `run.go` usage and in the atlas, but not in README.md. The sibling `--broadcast-web-replay` and `--broadcast-web-query` flags are in README.md at lines 401–405 and 866–872 (docs gate).

**4. Minor findings**
- `headless.open` (`fidelity_check.go:931`) keeps polling for 30 s after Chrome has died (`cdp: read: EOF`). It should fail fast on a closed connection. I saw this here: each live test spent 30.7 s before failing.
- ARCH-SECURE: frame records from disk are trusted as-is.
  - `f.File` is joined to the recording directory without a `filepath.IsLocal` check (`fidelity_check.go:746`).
  - `W`, `H` and `Scale` are unbounded, so a hand-edited frame can request a huge viewport.
  - Fix: check the file path and clamp the viewport where the frames are read.
- ARCH-DRY: `launchHeadless` is now product code, but `replay_test.go`'s `headlessPage` (around line 280) still repeats the same pipe and launch sequence. The test could build on `launchHeadless`.
- ARCH-FUNERAL: `fidelity.Write` creates `.fidelity-*` temporary directories inside the recording directory. A crash mid-write leaves one there, and nothing sweeps it. Putting the temporary directory under `fidelity/` or the OS temp dir (same filesystem) would avoid this.
- ARCH-ORDER: the `checkRecording` goroutine uses `context.Background()`, so it has no cancellation path. It dies with the process, and Chrome exits when its pipe closes, but its lifetime isn't bounded by the cast's lifetime.
- `Summarize` writes `Over` and `Crop` back into the caller's `shots` slice. That's harmless today but surprising.

**5. Test coverage notes**
- Ran here and passed: `go test ./cmd/internal/webcast/fidelity`, the webcast tests matching `Fidelity|CastChecks|ReplayServer|KeepWorst|SideBySide`, and `node --test tests/webcast/replayer.test.mjs` (4/4).
- Could not run here:
  - couchcmd tests: the package didn't build because generated runtime assets are missing.
  - Live tests (`WEBCAST_LIVE_CHROME=1`): Chrome exited with EOF in the sandbox. The Done-when numbers rest on the issue Log.
- Gaps:
  - No test covers a continued (clipped-window) `replayAt` with running animations. That gap is why the Important finding above shipped.
  - No test covers how `score` orders and keeps retries. A pure helper, or a fake for the `replayFrame` seam, would make this testable without Chrome.

**6. Architectural notes for upcoming work**
- **ARCH-DRY:** pass, apart from the minor launcher duplicate.
- **ARCH-PURE:** pass.
- **ARCH-PURPOSE:** pass. Frames, the check, the report, CLI, end-of-cast run and broken-render detection are all present.
- **ARCH-MOCK:** pass with a note. Chrome has no stateful fake, but the plan argues for this explicitly, the couchcmd and Cast seams use injected fakes, and the live checks are opt-in rather than scheduled.
- **ARCH-CONSTRAINTS:** flagged. Retry resets break the per-frame cost envelope.
- **ARCH-SECURE:** minor flag. Frame paths and dimensions are not validated.
- **ARCH-ORDER:** flagged. The Replayer's `at`/paused-animation state handles continuation incorrectly (Important), and the background check has no cancellation.
- **ARCH-FUNERAL:** minor flag for the temporary-directory residue. `fidelity/` itself is bounded and is removed with the recording.

**7. Plan revision recommendations**
- `## Revisions`: `WriteReport` is implemented as `fidelity.Write` (plus `Report.Summary` and `Within`). The hook test lives in `record_cast_test.go`, not `cast_test.go`.
- `## Revisions`: "Forward-only: a recording is replayed once" no longer holds with `Retries`. A backward retry resets the Replayer. State the actual cost, or the fix.
- `## Revisions`: the Retries semantics. Say whether the retries are true replays at t±δ (after the fix) and how they are ordered.

```findings
findings:
  - id: new
    severity: Important
    family: replay-continuation-state
    title: |
      Replayer.replayAt finishes paused animations on a clipped (continued) window, so retries and close frames are drawn at the wrong phase
    detail: |
      With start = this.at, settle() is followed by finishAnimations(), which jumps animations paused at the previous t to their end. The -150/+150/+300 retries in headless.score, and any frame less than 1 s after the previous one, are drawn with transitions completed rather than at their phase. A backward retry also resets and re-feeds the whole recording, so over-tolerance frames cost O(records) each, breaking the plan's per-frame envelope. Fix: play() paused animations when continuing, and add a node test for it.
  - id: new
    severity: Important
    family: docs-gate-readme
    title: |
      README update appears missing for couch --broadcast-web-fidelity [ID|DIR]
    detail: |
      The new CLI flag and its exit codes are in run.go usage and the atlas but not in README.md, where the sibling --broadcast-web-replay and --broadcast-web-query flags are documented (around lines 401-405 and 866-872).
  - id: new
    severity: Minor
    family: fail-fast-on-dead-peer
    title: |
      headless.open polls for 30 s after Chrome has died instead of failing on the closed CDP connection
  - id: new
    severity: Minor
    family: untrusted-persisted-input
    title: |
      CheckFidelity trusts frame records from disk: File joined unchecked, W/H/Scale unbounded
  - id: new
    severity: Minor
    family: shared-helper-duplication
    title: |
      replay_test.go headlessPage repeats the launchHeadless pipe/launch sequence now in product code
  - id: new
    severity: Minor
    family: residue-without-sweep
    title: |
      fidelity.Write's .fidelity-* temp dir inside the recording is left behind on a crash mid-write
  - id: new
    severity: Minor
    family: unbounded-task-extent
    title: |
      The end-of-cast check goroutine runs on context.Background with no cancellation path
```
