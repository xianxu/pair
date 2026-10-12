# Boundary Review — pair#474 (whole-issue close)

| field | value |
|-------|-------|
| issue | 474 — broadcast-web: retire the prototype fixtures once #470 and #471 land |
| repo | pair |
| issue file | workshop/issues/000474-broadcast-web-retire-the-prototype-fixtures-once-470-and-471-land.md |
| boundary | whole-issue close |
| milestone | — |
| window | 11439465af7016eba928ee5b5c1667d4b00fd37e..1747d09ab1e8316e6c51ce5305affa532e5f6cb3 |
| command | sdlc close --issue 474 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T06:17:48-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking a stale reference to the deleted converter, and whether one live test's timings were re-pinned.

Tests pass; now checking that the new caret-fold rule's test fails when the rule is removed.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The retirement is complete. The prototype fixtures `stress`, `app` and `blog` are gone, along with their goldens, `record-mjs.patch`, `PrototypeFixtures`, `observerLag`/`frameOffsets` and the converter they needed. Every test now runs on `rec-*` or the hand-made fixtures. The budget is back to 4.5 MB, and the fixtures measure 3.3 MB on disk. `go test ./cmd/internal/webcast/...` passes and so do all 79 node tests in `tests/webcast/*.test.mjs`. `artifactpath` fails, but only because this checkout lacks the gitignored `cmd/internal/runtimebundle/assets/`; the change didn't cause that. I removed the new caret-fold rule as a check, and 4 tests failed, including `TestEventLinesFoldCaret`. That beats the 2 the implementor claimed. One thing to fix before shipping: `TestLiveReplayServerMarks` was moved to `rec-app` but kept the prototype's note times, so it no longer covers the resolved note. There is also one stale comment.

**1. Strengths**
- `recording.Fixtures()` (`cmd/internal/webcast/recording/fixture.go:18`) finds fixtures the same way `tests/webcast/fixtures.mjs` does. `TestFixtureCoverage` now also fails when a coverage check exists for a fixture that's gone (`fixture_test.go:111`), so the list can't silently fall out of step.
- The caret fold (`query/events.go:41,54,76`) is small and only affects the person typing. `TestEventLinesFoldCaret` covers the cases that should split: another person's caret, a selection range, and a driver's own caret.
- `typedJustBefore` in `agent_test.go:93` restates the fold rule on purpose, as an independent oracle. That is not ARCH-DRY duplication.
- Fixture-based expectations were re-pinned with explanations: `TestMainJSON`'s window and node id, `TestEventLinesFoldTyping`, and both viewer live tests' `at` value (10041 → 8284).

**2. Critical:** none.

**3. Important**
- `cmd/internal/webcast/replay_live_test.go:123-127`: `TestLiveReplayServerMarks` moved to `rec-app` but still samples at `12200, 12400, 12600, 13900, 14100`. Those times surround the prototype's `#open-modal` note (made at 12500, resolved at 14000). `rec-app`'s matching note is `#close-modal`, made at 9800 and resolved at 11000 (`fixturegen/notes.go:34`). The test still passes because it compares against the reducer. But it no longer checks a note appearing with its stroke, or a resolve removing the pin.
  - **Fix:** use moments around 9800 and 11000 (e.g. `9700, 9900, 10100, 10900, 11100`).
  - **Other tests of this kind:** I checked every other test that moved to a new fixture and reads its times. All were re-pinned, and this is the only one left.

**4. Minor**
- `cmd/internal/webcast/recorder/mirror.go:22` still calls `#457's fixturegen/cdpconv.go` "the reference this mapping was ported from", but this change deletes that file. Reword it to point at git history. A sweep for `cdpconv`/`fixturegen -raw` found no other stale references.

**5. Test coverage notes**
- The caret-fold removal check failed 4 tests: `TestAgentCanTellWhatEachNoteRefersTo`, `TestEventLinesFoldTyping`, `TestEventLinesFoldCaret` and `TestGolden`. The new fold rule is well pinned.
- I couldn't run the live Chrome tests here, so I haven't verified the "305 of 306 on rec-blog" figure in `atlas/webcast.md:72`.

**6. Architecture**
- **ARCH-DRY: pass.** The hand-written fixture lists are now derived from one place. The remaining hand-written lists in the selector tests intentionally pick only the recorded fixtures.
- **ARCH-PURE: pass.** The fold logic is pure and tested with an in-memory builder.
- **ARCH-PURPOSE: pass**, with the marks-test finding above. The spec's harness viewport item was already done by #478: `harness.js:3` says so. Skipping the optional rename was a reasoned choice: `rec-segments` would collide with the hand-made `segments`.

**7. Plan revisions:** none needed beyond a Log line when the marks test is re-pinned.

```findings
findings:
  - id: new
    severity: Important
    family: repin-fixture-content-expectations
    title: |
      TestLiveReplayServerMarks moved to rec-app but still samples the prototype app's note moments (12500/14000)
    detail: |
      replay_live_test.go:127 checks 12200-14100, which surrounded the prototype's open-modal note and its resolve. rec-app's matching note is made at 9800 and resolved at 11000 (fixturegen/notes.go:34), so the test no longer covers a note appearing with its stroke or a resolve removing the pin. It passes only because it compares against the reducer. Re-pin to moments around 9800 and 11000. No other moved test still uses the old fixture's times.
  - id: new
    severity: Minor
    family: stale-reference-to-deleted-source
    title: |
      recorder/mirror.go:22 cites the deleted fixturegen/cdpconv.go as its reference
    detail: |
      Reword it to point at git history. A sweep for cdpconv and fixturegen -raw found no other stale references.
```

---

## Re-review — 2026-10-12T06:20:52-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 474 — broadcast-web: retire the prototype fixtures once #470 and #471 land |
| repo | pair |
| issue file | workshop/issues/000474-broadcast-web-retire-the-prototype-fixtures-once-470-and-471-land.md |
| boundary | whole-issue close |
| milestone | — |
| window | 11439465af7016eba928ee5b5c1667d4b00fd37e..d7c3d9621eebf05ac5ff680b63bab22379fe0b50 |
| command | sdlc close --issue 474 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T06:20:52-07:00 |
| verdict | SHIP |

## Review

The webcast tests pass. I'm checking whether the artifactpath failures were already there at base before I run the live Chrome test.

```verdict
verdict: SHIP
confidence: medium
```

**VERDICT: SHIP.** Both findings from the earlier round are fixed. The only new code since then is commit `d7c3d9621`. For BR-1, the marks test no longer uses a hand-typed list of times; it now builds its sample times from the fixture's own records. For BR-2, the comment in `mirror.go` now says the source was retired and points to git history. A search of the repo turned up no other stale references to `cdpconv`, `record-mjs.patch` or `fixturegen -raw`; the one mention in the fixtures README describes the retirement. The webcast and recording unit tests pass. Confidence is medium because `TestLiveReplayServerMarks` could not run here: Chrome closed its connection at startup (`cdp: read: EOF`). So the fix is checked by reading the code and the fixture, not by running the test.

1. **Strengths**
   - `cmd/internal/webcast/replay_live_test.go:131-137` takes sample times from every `Note`, `NoteState`, `Stroke` and `Seg` record, 100 ms either side. This fixes the whole class of stale times, not just the one instance: re-recording the fixture can't make the times stale again. For rec-app it now samples 9700/9900 (note n2 appears), 10900/11100 (n2 resolved at the `note-state` record at t=11000), 5500/5700 and 18500/18700 (strokes), and 14007/14207 (navigation to a new page at seg 2).
   - The new `placed` check at `replay_live_test.go:157` follows how `drawMarks` places a pin (`replay.js:441-443`): the note's element box, else `at.box`, else `region`. Without it, the derived times would make the test expect pins that `drawMarks` never draws.
   - `mirror.go:22-23` now records where the code came from without pointing at a missing file.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - The `Seg` record at t=0 produces a sample at t=-100. Both sides should come out empty there, so it's harmless, but it's a wasted iteration.
   - `placed` treats a note as placeable whenever its node exists in the reducer, while the JS needs `renderer.boxOf` to return a box. If a node exists but has no layout box and the note carries no fallback box, the test would fail instead of silently passing. That's the safe direction, so it isn't a finding.

5. **Test coverage notes**
   - The webcast, query, recorder and recording packages all pass.
   - `artifactpath` fails in this checkout because `cmd/internal/runtimebundle/assets/` is gitignored and was never generated here (`embed.go: pattern assets/runtime/files: no matching files found`). That's this environment, not the change. The diff's only edit to `manifest.go` removes the deleted `cdpconv.go` entry, which is correct.
   - The live marks test still needs a run with `WEBCAST_LIVE_CHROME=1` on a machine where Chrome starts.

6. **Architecture**
   - **ARCH-DRY: pass.** The test no longer keeps its own copy of the fixture's times.
   - **ARCH-PURE: pass.** Only test code changed.
   - **ARCH-PURPOSE: pass.** The fix covers the whole class, and the stale-reference search came back clean across the tree.

7. **Plan revisions:** none.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      replay_live_test.go:131-137 derives samples from Note/NoteState/Stroke/Seg records (covers 9800 note and 11000 note-state resolve); placed filter mirrors replay.js drawMarks:441-443. Live run not possible here (Chrome CDP EOF in sandbox).
  - id: BR-2
    disposition: addressed
    note: |
      mirror.go:22-23 now says the source was retired in 474 and lives in git history; tree-wide grep finds no other stale cdpconv / record-mjs / fixturegen -raw references.
```
