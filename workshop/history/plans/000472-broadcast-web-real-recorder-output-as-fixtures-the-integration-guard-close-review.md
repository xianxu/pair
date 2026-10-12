# Boundary Review — pair#472 (whole-issue close)

| field | value |
|-------|-------|
| issue | 472 — broadcast-web: real recorder output as fixtures (the integration guard) |
| repo | pair |
| issue file | workshop/issues/000472-broadcast-web-real-recorder-output-as-fixtures-the-integration-guard.md |
| boundary | whole-issue close |
| milestone | — |
| window | 670c398db9add1cb8f8c35354f5bd9a6cd722d9a..32c95712d951839576d43bed81d961da36ec7b6f |
| command | sdlc close --issue 472 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T04:12:42-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Still reviewing: checking how screenshots relate to segment boundaries in the recorder, then running the tests.

Node tests pick up fixtures by directory. Now running the node model test and checking the fidelity tool's viewport handling.

Fidelity already uses the recorded viewport. Next I'll revert the frame-time fix in a scratch copy to confirm its regression test fails without it.

The regression test fails without the fix, as intended. Now checking whether the new "never before its segment" clamp is tested.

The segment clamp has no test: removing it leaves the suite green. Now checking the documented fixture durations against the recorded data.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

This boundary delivers what #472 asked for. Four real-recorder fixtures (`rec-stress`, `rec-app`, `rec-blog`, `rec-segments`) sit beside the prototype ones. The fixture-iterating component tests pick them up: Go tests through `recording.Fixtures`, node tests through a directory scan in `tests/webcast/fixtures.mjs`. Observer lag is 0 for the real fixtures, and retiring the prototype fixtures is filed as #474 with a concrete trigger and sweep list.

Tests I ran:
- `go test ./cmd/internal/webcast/...` passes.
- `node --test tests/webcast/model.test.mjs` passes, 14/14.
- I reverted the `shotDone` fix and `TestRecorderFrameKeepsItsCaptureTime` failed ("frame at 300, want the capture's midpoint 60"), so the claimed regression test is real.

I did not run the Chrome-gated `LiveReplay` or fidelity runs. Both need Chrome, so I can't independently confirm those numbers from the Log.

Nothing blocks shipping. One cheap gap: the new "never before its segment" clamp (`r.segT`) has no test. Removing it leaves the whole recorder suite green.

**1. Strengths**
- `recorder/side.go:407-413`: the frame keeps its capture midpoint, and only the layout record takes the stream's time. This matches the format contract, the comment explains why, and the regression test exercises it.
- `replay_test.go:52-83`: `observerLag`, `frameTimes` and `captureSkew` are small pure helpers. `knownGaps` fails once a known gap starts passing, so the #475 exemption can't go stale silently.
- `fixturegen adopt` (`main.go:160-188`) converts nothing. It reuses `annotate`, `mergeByTime` and `write`, so the notes-and-pins path is shared with the prototype converter.
- `fixture_test.go:104-116`: coverage checks for `rec-*` are derived from their prototype twins, and the one difference (no FontFace font) is explicit. It also adds a guard that any fixture without coverage checks fails.
- #474 lists every hard-coded prototype consumer that needs moving (query, notes, replay, `viewer_live_test`, fidelity) and says the budget goes back to 4.5 MB.

**2. Critical findings:** none.

**3. Important findings**
- **`recorder/side.go:413`: the `r.segT` clamp is untested.** When I replaced `max(r.t(d.at), r.segT)` with `r.t(d.at)`, every recorder test still passed. The clamp only matters when a shot's midpoint falls before a segment's `Seg` record (a navigation or tab switch during the screenshot), and no fixture or test reaches that case. Fix: in the style of `TestRecorderFrameKeepsItsCaptureTime`, start a new segment inside the fake `Page.captureScreenshot`, then assert that `f.T` equals the segment's `t` and the result validates. Same family: the two `r.segT = r.m.st.T` assignments (`recorder.go:679`, `:724`) are also only reachable through this path.

**4. Minor findings**
- ARCH-DRY: `testdata/replay/harness.js:865-868` restates the viewport sizing in `viewer.js:114-119`. It's three lines in a test harness, and the comment names the source. Acceptable, but it can drift.
- ARCH-DRY: fixture names are listed by hand in four places: `recording.Fixtures`, `recording.PrototypeFixtures`, `fidelity_test.go:106` (which could filter `rec.Fixtures` to those with frames) and `fixture_test.go`'s `want`. #474 covers retiring these.
- A shot that straddles a segment change is stamped at the new segment's start but shows the old page. This predates the change (the old clamp did the same) and isn't a regression. Worth a line on #475 or in the format's frame-`t` note.
- `agent_test.go:361`: the expected string `"selection ? … "` ends in a space. It matches `query.UserAction` today, but that output looks unintended.

**5. Test coverage notes**
- Covered on the `rec-*` fixtures by iterating tests: reducer/validator/pins (`fixture_test.go`), the node model, notes (`structural`, `fake`), queries (`querier`, `agent`, `golden`, with new goldens), and replay (`replay_test.go` loops at 89, 133 and 188).
- Still prototype-only, as #474 lists: `viewer_live_test.go` (`blog`), `notes/landmark_test.go`, `selector_test.go`, `query/cli_test`, `events_test` and `render_test`.
- Assets are exercised through replay, not through the `assets` package.
- Both modes of the Done-when are exercised: lag 150 for prototype names and lag 0 for the rest.

**6. Architectural notes**
- ARCH-PURE: pass. The new replay helpers are pure, and `adopt` is thin IO glue around shared pure steps.
- ARCH-DRY: pass, with the Minor notes above.
- ARCH-PURPOSE: pass. The Done-when "every component's tests" is met by the iterating tests. The hard-coded single-fixture tests are a separable follow-up (#474) with a stated trigger, not the deferred point of the issue. The driven fixture stays hand-written until #467, as the Spec allows.
- The atlas (`webcast.md`, `webcast-format.md`) and the fixtures README are updated, and their durations, frame counts and sizes match the checked-in data (5.4 s/3, 22.3 s/11, 12.5 s/6, 9.8 s/5; 7.3 MB total under the 9 MB budget).

**7. Plan revision recommendations:** none. The plan matches the code.

```findings
findings:
  - id: new
    severity: Important
    family: new-guard-without-regression-test
    title: |
      recorder/side.go:413 segT clamp (frame never before its segment) has no test; removing it leaves all recorder tests green
    detail: |
      Mutation f.SetTime(r.t(d.at)) passes go test ./cmd/internal/webcast/recorder. Add a test, like TestRecorderFrameKeepsItsCaptureTime, that starts a new segment inside the fake captureScreenshot and asserts f.T equals the segment's t and the result validates. The same family includes the segT assignments at recorder.go:679 and :724, which only this path reaches.
  - id: new
    severity: Minor
    family: hand-maintained-restatement
    title: |
      Fixture-name lists and viewer viewport sizing are restated by hand
    detail: |
      harness.js:865-868 mirrors viewer.js:114-119 sizing. Fixture names are hand-listed in recording.Fixtures, PrototypeFixtures, fidelity_test.go:106 (could filter rec.Fixtures by frames) and fixture_test.go want. Retirement (#474) touches all of them.
  - id: new
    severity: Minor
    family: segment-straddling-shot
    title: |
      A screenshot that straddles a segment change is stamped at the new segment's start but shows the old page
    detail: |
      This predates the change (the old st.T clamp did the same). Worth noting on #475 or in the format's frame-t note.
```

---

## Re-review — 2026-10-12T04:15:17-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 472 — broadcast-web: real recorder output as fixtures (the integration guard) |
| repo | pair |
| issue file | workshop/issues/000472-broadcast-web-real-recorder-output-as-fixtures-the-integration-guard.md |
| boundary | whole-issue close |
| milestone | — |
| window | 670c398db9add1cb8f8c35354f5bd9a6cd722d9a..141bd16abb628ddaf42db3df911612aea351ffd2 |
| command | sdlc close --issue 472 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T04:15:17-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Now mutation-check BR-1: run the test as-is, then remove the clamp, and separately drop the segT assignments.

I've finished the checks and test runs; here is the verdict.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** The issue's purpose is met. Four fixtures made by the real recorder (`rec-stress`, `rec-app`, `rec-blog`, `rec-segments`) are checked in next to the prototype ones. They have coverage checks, query goldens and an observer lag of zero. The retirement of the prototype fixtures is filed as #474 with its trigger stated. `go test ./cmd/internal/webcast/...` passes on the pinned head. The fix for BR-1 is real. When I put the old stamping back (`f.SetTime(r.t(d.at))`), the new `TestRecorderFrameNeverBeforeItsSegment` fails. When I removed the `segT` assignment at `recorder.go:724` (the navigation path), it also fails. But BR-1 also named `recorder.go:679` (`snapshotTop`, which runs on live/resume/tab switch). When I removed that assignment, the whole recorder package still passed. So that part of BR-1 is still open. One test is cheap to add. The other two prior findings are Minor and acceptable.

1. **Strengths**
   - `side.go:413`: the frame and the layout record now get their times separately. Each is set with one `max()` and the comment explains why.
   - `replay_test.go` `knownGaps`: a filed gap (#475) is listed, and the test fails once that frame comes back within tolerance. A known gap can't quietly stay in the list.
   - `fixture_test.go`: the rec-* coverage lists are built from the prototype lists, with the one difference stated (the FontFace font), and any fixture with no checks fails.
   - `fixturegen adopt` is small and converts nothing. It reuses `annotate`, `mergeByTime` and `write`.
   - `fidelity/run.go` skips offsets that land in a different segment, which is the right guard.

2. **Critical:** none.

3. **Important**
   - **BR-1, still open.** The `segT` assignment at `recorder.go:679` (`snapshotTop`) has no test. Removing it leaves every test green, which I checked. The case it covers: a screenshot is still in flight across a pause and resume, or across a tab switch. Without the assignment, that frame would be stamped back in the pause gap or in the previous segment. Fix: copy `TestRecorderFrameNeverBeforeItsSegment`, but send `Pause` and then `Live` (resume) inside the fake `captureScreenshot`. Assert that `f.T` equals the time of the resume segment.

4. **Minor**
   - **New finding, segment-straddling comparison.** `replay_test.go` `frameTimes` compares a frame at `t-50` (`captureSkew`) and at `t+lag` without checking which segment those times fall in. `fidelity/run.go` skips times in another segment; replay does not. A frame stamped at a segment's start (exactly the `segT` clamp case) can be compared against the previous segment's page. That can only produce a false pass.
   - **BR-2 (repeated restatements): moved to #474 but only partly covered there.** #474's Spec covers `PrototypeFixtures`, the fidelity list and the coverage lists. It does not mention `harness.js`'s copy of `viewer.js`'s viewport sizing.
   - `captureSkew` also makes the prototype fixtures more lenient (`t-50` is now tried for them too). This is harmless, but it was not stated anywhere.

5. **Test coverage**
   - Mutation checks I ran on the pinned head:
     - `side.go:413` clamp removed: test fails.
     - `recorder.go:724` assignment removed: test fails.
     - `recorder.go:679` assignment removed: all tests pass, so nothing pins it.
   - I did not run the live-Chrome replay or the node viewer tests. They need `WEBCAST_LIVE_CHROME` / node, and the plan's Log reports them.

6. **Architecture**
   - **ARCH-DRY: flag (Minor).** `harness.js` restates `viewer.js` sizing (BR-2, already on record). The rule "an offset from a frame's time must stay in its segment" exists in `fidelity/run.go` but not in `replay_test.go`. It should become one shared helper (e.g. in `recording`) that both use.
   - **ARCH-PURE: pass.** The changes in `shotDone` are a pure `max()`. `adopt` is a thin IO shell around the pure `annotate` and `mergeByTime`.
   - **ARCH-PURPOSE: pass on the issue.** Every component's tests run on the rec-* fixtures, and retirement is filed. On the finding, BR-1 was fixed at the named site plus `:724`, but the sibling it named at `:679` was left. That is the open item in section 3.

7. **Plan revisions**
   - The Log's BR-1 line implies the clamp is fully pinned. Amend it to say the `snapshotTop` (resume/switch) path gets its own test, or will.
   - Add "replace `harness.js`'s copy of `viewer.js` sizing" to #474's Spec.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      side.go:413 clamp and recorder.go:724 are now pinned (mutation-checked red); recorder.go:679 (snapshotTop, resume/tab-switch) segT assignment removed still leaves go test ./cmd/internal/webcast/recorder green — add a pause→resume-inside-screenshot test.
  - id: BR-2
    disposition: not-addressed
    note: |
      Deferred to 474, whose Spec covers PrototypeFixtures/fidelity/coverage lists but not harness.js's copy of viewer.js viewport sizing; add it there.
  - id: BR-3
    disposition: addressed
    note: |
      atlas/webcast.md now states a straddling call is stamped at the new segment's start though its pixels may show either page.
findings:
  - id: new
    severity: Minor
    family: segment-straddling-shot
    title: |
      replay_test frameTimes compares a frame at t-captureSkew / t+lag without staying in its own segment
    detail: |
      This is the 2nd finding in family segment-straddling-shot. Rule: any time offset applied to a frame's t must stay inside the frame's own segment. fidelity/run.go enforces it inline (st.Seg != own.Seg); replay_test.go frameTimes (t-50, t+150) does not, so a segT-clamped frame can be compared against the previous segment's page (a false pass). Instances in window: fidelity/run.go Offsets loop (guarded), replay_test.go frameTimes (unguarded). Fix the rule once: a shared recording helper filtering offsets by segment, used by both.
```

---

## Re-review — 2026-10-12T04:26:21-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 472 — broadcast-web: real recorder output as fixtures (the integration guard) |
| repo | pair |
| issue file | workshop/issues/000472-broadcast-web-real-recorder-output-as-fixtures-the-integration-guard.md |
| boundary | whole-issue close |
| milestone | — |
| window | 670c398db9add1cb8f8c35354f5bd9a6cd722d9a..f82bd66c96e66f77ff6764b794bb001604de96f2 |
| command | sdlc close --issue 472 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T04:26:21-07:00 |
| verdict | SHIP |

## Review

Run tests and mutation checks for BR-1.

```verdict
verdict: SHIP
confidence: high
```

The three open findings are closed or properly deferred, and I found nothing new worth raising. BR-1 is fixed. `TestRecorderFrameNeverBeforeItsSegment` is now a table with three cases (navigate, resume, tab switch). In this checkout I removed each of the three segment-clamp lines one at a time (`side.go:413`, `recorder.go:679`, `recorder.go:724`), and each removal makes the test fail. With nothing removed, `go test ./cmd/internal/webcast/...` passes. BR-4 is fixed. The offset rule is now a single helper, `recording.FrameTimes`. It has a unit test with edge cases, and both the fidelity tool and `replay_test.go` call it, so neither has its own copy of the rule. BR-2 is Minor and is deferred to #474. #474's Spec names every site involved: `harness.js` sizing, `recording.Fixtures`, the fidelity list and the coverage `want`. That is the change that rewrites those lines anyway.

1. **Strengths**
   - `recording/fixture.go:65` `FrameTimes` is a pure helper. It states the rule once ("an offset stays in the frame's own segment; a pause or the end keeps its page"). `fixture_test.go:252` tests it without IO, across pause, resume, navigate, after-end and no-segment cases.
   - `recorder_paths_test.go` covers every place that sets `segT`. The three sites are each mutation-checked, and I reproduced all three failures myself.
   - `side.go:413-414` keeps the frame's time (the capture midpoint, clamped to its segment) separate from the layout record's time (the stream's time). That fixes the reported 43–87 ms stamping skew.
   - Every component test loops over `recording.Fixtures`: notes, query (golden, agent, querier, fake), replay and the coverage tests. Adding the `rec-*` names there runs them all on the real fixtures, so the Done-when holds.
   - Known gaps are written down so they fail loudly: `knownGaps` turns red once #475 is fixed.

2. **Critical:** none.

3. **Important:** none.

4. **Minor:** BR-2 stays with #474 (see the dispositions below). `FrameTimes` rescans all records for each offset. That costs O(records × offsets) per frame, which is trivial at these fixture sizes and only runs in tests and tooling.

5. **Test coverage**
   - The clamp mutations go red, as described above.
   - `FrameTimes` is pinned directly by its unit test. Replay and fidelity only call it, so their own runs are not the evidence for it.
   - The live-Chrome replay and fidelity runs are env-gated. I did not run them here, so the fidelity numbers come from the issue's Log, not from me.

6. **Architecture**
   - **ARCH-DRY:** pass. The offset rule is consolidated. The remaining hand-written lists are BR-2, tracked in #474.
   - **ARCH-PURE:** pass. `FrameTimes` is pure, and the recorder change is a small clamp.
   - **ARCH-PURPOSE:** pass. Real fixtures run through every component's tests, and retirement is filed as #474 with its trigger.
   - **ARCH-MOCK:** pass. The recorder tests use the stateful `fakeChrome`, and live Chrome checks are gated by env.
   - **ARCH-CONSTRAINTS:** pass. The fixture budget is raised while both sets coexist and is due to drop at retirement (#474: back to 4.5 MB).
   - **ARCH-SECURE:** N/A. Only checked-in test data and in-process recorder state are involved; no secrets and no new untrusted parsing.
   - **ARCH-ORDER:** pass. Straddling a segment change is pinned as a deterministic interleaving: the fake moves the clock and fires the segment event inside the screenshot call.
   - **ARCH-FUNERAL:** pass. The prototype fixtures have their end recorded in #474.

7. **Plan revisions:** none. The plan and Log match the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Table test covers navigate/resume/tab; I removed side.go:413, recorder.go:679 and recorder.go:724 one at a time and each fails TestRecorderFrameNeverBeforeItsSegment.
  - id: BR-2
    disposition: addressed
    note: |
      Minor; deferred with its trigger to #474, whose Spec names the harness.js sizing copy and the hand-listed fixture names, the change that rewrites all of them.
  - id: BR-4
    disposition: addressed
    note: |
      recording.FrameTimes (fixture.go:65) is the single rule, unit-tested in TestFrameTimesStayInTheFramesSegment and used by both fidelity/run.go and replay_test.go.
```

---

## Re-review — 2026-10-12T04:32:10-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 472 — broadcast-web: real recorder output as fixtures (the integration guard) |
| repo | pair |
| issue file | workshop/issues/000472-broadcast-web-real-recorder-output-as-fixtures-the-integration-guard.md |
| boundary | whole-issue close |
| milestone | — |
| window | ddf011e0896b1ee24e656aeedd99edfec37db737..dc6d4f1d146c33aa663bd5d656a2567b6fabe033 |
| command | sdlc close --issue 472 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T04:32:10-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

**Summary.** This window has no new behaviour, and all four prior findings were closed in earlier rounds. It contains three things. First, the `rec-*` notes goldens were regenerated after the merge with main, because main's #469 changed the rendered context text. Second, the merge conflict in `workshop/lessons.md` was resolved. Third, there are tracker-only edits: #472's status and log, the plan checklists for the follow-ups #474 and #475, and the gate and review plan files. I ran `go test ./cmd/internal/webcast/...` at the pinned head and every package passed, including `query`, which owns the regenerated goldens. #472's Spec, Revisions and Done-when are unchanged. The only edits to it are frontmatter (status, card mirror, flow, hours) and the close log entry, so the scope has not changed.

1. **Strengths**
   - The regenerated goldens (`cmd/internal/webcast/query/testdata/golden/rec-*.notes.txt`) show #469's rendering, not a hand edit. Words are separated where element boundaries used to run them together ("alpha beta gamma", "Week 3 274 tasks"). Inline `<script>` source no longer leaks into `rec-app` and `rec-segments`. The golden tests in `query` pass.
   - The `workshop/lessons.md` conflict resolution keeps both sides. The #472 lesson and the #473 lesson are both present, word for word; only their order changed.
   - The #474 and #475 plans write down the cleanup that BR-2 deferred: one place for the viewport sizing and one for the fixture-name lists. They also pin #475's test-first step, which is to drop the `knownGaps` entry. The follow-up work therefore has a concrete record.

2. **Critical findings:** none.
3. **Important findings:** none.
4. **Minor findings:** none.
5. **Test coverage notes:** the regenerated goldens are checked by the existing `query` golden tests, which are green at the head. There is no new executable code in this window, so no new regression test is needed.
6. **Architectural notes:**
   - **ARCH-DRY: pass.** The window adds no duplicated logic. BR-2's restatements are planned in #474.
   - **ARCH-PURE: pass.** The window contains no code.
   - **ARCH-PURPOSE: pass.** The follow-ups are separate pieces of work (retiring the prototype fixtures after #470 and #471 land, and the stylesheet gap), not the purpose of #472 pushed to later.
7. **Plan revision recommendations:** none.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Disposed in prior rounds; the interdiff does not touch the recorder, and the recorder tests pass at the head.
  - id: BR-2
    disposition: addressed
    note: |
      Disposed in prior rounds; #474's plan now lists the cleanup (one source for viewport sizing and fixture-name lists).
  - id: BR-3
    disposition: addressed
    note: |
      Disposed in prior rounds; nothing in this interdiff changes it.
  - id: BR-4
    disposition: addressed
    note: |
      Disposed in prior rounds; nothing in this interdiff changes it.
```
