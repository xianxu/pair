# Boundary Review — pair#470 (whole-issue close)

| field | value |
|-------|-------|
| issue | 470 — broadcast-web: pointer and notes in the viewer — anchored to element and time, multi-viewer |
| repo | pair |
| issue file | workshop/issues/000470-broadcast-web-pointer-and-notes-in-the-viewer-anchored-to-element-and-time-multi-viewer.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1da5c76c60b687fc25a90cb1dc9f1268daa9172e..186d2b496d0e79d45e348e87a9546ac84a8c866d |
| command | sdlc close --issue 470 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T04:52:15-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Disposing the prior findings: BR-19 and BR-20 are fixed. Next I'm checking the remaining open ones in the code.

I found a test that fails intermittently under `-race`. Now checking whether it's a test race or a product bug.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The issue delivers what it set out to do. The `board` package handles parsing, admission and the ordering re-check. The server routes give fixed answers. `annotate.js` and the shared `collab.js` cover the viewer, and Couch now shows `💭N` with the notes menu. The plan's Revisions match the code. Node tests pass (68/68), as do the Go tests in `webcast/...` and `broadcast`. Of the open findings, BR-19 and BR-20 are now fixed. One new problem blocks a clean SHIP. `TestWebNotesMenuSourceAndStableIDs` fails 10 of 20 runs under `-race`, which is the repo's `make test-race` target, and 0 of 50 without it. The test writes the web-menu state from the test goroutine while the cast's delivery goroutine is still applying the board's first snapshot. So it only ever checks one interleaving. Every other open finding is Minor and still open as it was. The commit after round 7 changed only workshop files.

Environment note: couchtty first failed to build because the generated runtime bundle was missing. I ran `generatecmd` to create it. Several older PTY tests (`TestCouchProductionSoak`, `TestCouchPressure*`, `TestNotificationPTYConformance`) then failed with `operation not permitted` from this review environment. They aren't in this diff, and I did not count them.

**1. Strengths**
- The rule that the writer re-checks the document has a single statement, `recording.State.OnDoc`. Both the recorder at dequeue and `board.Writer` use it, and the plan's Revision records this honestly.
- `validName` now uses the board's own predicate (`/[\p{Cc}\p{Cf}]/u`, `annotate.js:63`), and the node test lists each character that was refused (`annotate.test.mjs:98`). `beforeSave` checks the name before `setName`.
- Web notes keep a stable menu id (`webMenuIDLocked`, `console_notes.go:444`). This fixes the act-by-position hazard from BR-21 at its root.
- The atlas now says plainly that the live viewer and replay's `drawMarks` diverge until #477 converges them, instead of claiming they are the same code.

**2. Critical:** none.

**3. Important**
- `cmd/internal/couchtty/console_webnotes_test.go:173-203`: flaky under `-race`. Failure: `switched to web: open false source 1 hover 0`. `newWebNotesCast` starts the real `cast.Notes()` delivery loop (`console_webcast.go:136-147`). Its first `applyWebNotes(cast, [])` can land after the test has written `webItems` directly. `settleMenuLocked` then closes the menu.
  - This is the 2nd finding in family `shared-state-loop-confinement`. The rule: web-menu state is written only on the terminal loop, by its single producer, and that includes tests.
  - Fix: drive the notes through the board (`r.note(...)` for a and b, wait for `webItems` to show both). Then resolve a through `Cast.ResolveNote` or a viewer note-state while the drawn menu is kept. Alternatively, wait for the first delivery and then inject inside `run(func(){ c.applyWebNotes(cast, live(...)) })`.
  - Verify with `go test -race -count=20 -run TestWebNotesMenuSourceAndStableIDs`.

**4. Minor**
- These carry over unchanged: BR-3, BR-4, BR-7, BR-12, BR-14, BR-16 and BR-24 (see dispositions).

**5. Test coverage notes**
- Node tests and the Go tests for `webcast/...` and `broadcast` pass. The board tests pass under the race detector.
- The headless Chrome e2e (`WEBCAST_LIVE_CHROME=1`) was not run here.
- The gaps are still the BR-12/14/16 set: no pure seam to node-test the shell's gesture and answer decisions. Also the BR-3 sink order and the BR-24 prune.

**6. Architectural notes**
- **ARCH-DRY:** pass. `collab.js` is shared, and the `strokesAt`/`drawMarks` split is recorded for #477.
- **ARCH-PURE:** pass for `board` and the `annotate.js` helpers. The shell's gesture logic is still in the IO layer (BR-12/16).
- **ARCH-PURPOSE:** pass. Every Done-when item is delivered.
- **ARCH-MOCK:** flag only as BR-7 (no shared contract between `board.Writer` and `recorder.Emit`).
- **ARCH-CONSTRAINTS:** pass. Rate, in-flight and size budgets are enforced at the routes.
- **ARCH-SECURE:** pass, apart from BR-4. Bodies are parsed strictly into typed values, and note text only ever reaches the page as `textContent`.
- **ARCH-ORDER:** flag. This is the new Important finding: a test that can only observe one interleaving. BR-3 is also still open.
- **ARCH-FUNERAL:** pass. Records go with the recording directory, `pending` lives at most one document, and `webMenuIDs` is cleared when the cast ends.

**7. Plan revision recommendations:** none needed. The plan matches the code.

```findings
dispose:
  - id: BR-3
    disposition: not-addressed
    note: |
      The order (dir, board, hub) in recordingOutput is still correct, but still no test pins it.
  - id: BR-4
    disposition: not-addressed
    note: |
      Still bounded only by 1e6 px. Deferred to wave 3, where NoteCrops gets its first production caller.
  - id: BR-7
    disposition: not-addressed
    note: |
      No shared conformance contract between board.Writer and the recorder's Emit path yet.
  - id: BR-12
    disposition: not-addressed
    note: |
      answered() and the toast exist but are untested. Unchanged since round 5.
  - id: BR-14
    disposition: not-addressed
    note: |
      endGesture runs on a docSeg change, but no test covers a stroke that spans a navigation.
  - id: BR-16
    disposition: not-addressed
    note: |
      endGesture covers every boundary, but no test drives a drag across a mode change or the cast's end.
  - id: BR-19
    disposition: addressed
    note: |
      validName uses /[\p{Cc}\p{Cf}]/u (annotate.js:63), the board's nameText predicate. beforeSave checks it before setName. The node test covers U+2067, U+061C, U+180E, U+FFF9, U+E0041 and U+0085, which the old class let through. Tests pass.
  - id: BR-20
    disposition: addressed
    note: |
      The atlas now says the live viewer draws with liveNotes, pinLayer and paintStrokes. strokesAt is the at-t counterpart, and #477 converges drawMarks onto it. This is prose-only and correct against annotate.js.
  - id: BR-24
    disposition: not-addressed
    note: |
      The prune loop in applyWebNotes (console_notes.go:431-435) is still untested. Removing it fails no test.
findings:
  - id: new
    severity: Important
    family: shared-state-loop-confinement
    title: |
      TestWebNotesMenuSourceAndStableIDs fails about half the time under -race: it writes web-menu state off the terminal loop while the cast's delivery goroutine also writes it
    detail: |
      This is the 2nd finding in family shared-state-loop-confinement. Rule: web-menu state is written only on the terminal loop, by its single producer (applyWebNotes from the cast.Notes loop), and tests included. The test sets c.pnotes.webItems directly at console_webnotes_test.go:173 and :199, while console_webcast.go:136-147 delivers the board's first snapshot ([]) concurrently. settleMenuLocked then closes the menu. Measured: 10 of 20 runs fail with -count=20 -race and 0 of 50 without -race. make test-race runs this package. Fix: drive the notes through the board (r.note), wait for delivery, and resolve through the board, or wait for the first delivery and then inject via run(applyWebNotes).
```

---

## Re-review — 2026-10-12T05:01:10-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 470 — broadcast-web: pointer and notes in the viewer — anchored to element and time, multi-viewer |
| repo | pair |
| issue file | workshop/issues/000470-broadcast-web-pointer-and-notes-in-the-viewer-anchored-to-element-and-time-multi-viewer.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1da5c76c60b687fc25a90cb1dc9f1268daa9172e..484950d804bed5bc2b3ce97417acea0297295d7e |
| command | sdlc close --issue 470 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T05:01:10-07:00 |
| verdict | SHIP |

## Review

The last commit is the one addressing BR-24/BR-25. Let me see it.

Generating the runtime bundle so the couchtty tests can build, then running the race test for BR-25.

BR-25 passes 20/20 under -race. Now I'll revert the prune loop to check that BR-24's test goes red.

BR-24 confirmed: removing the prune makes the test fail. Next I'm checking the older open Minor findings (BR-3/4/7/12/14/16) at head.

Every open Minor is unchanged in the latest commit, so its prior disposition stands. Last step: the full suites.

```verdict
verdict: SHIP
confidence: high
```

Round 9 fixes both findings that were blocking or tied to the latest change. **BR-25 is fixed.** The test now sends notes through the board, waits for them to arrive, resolves a note with a real `POST note-state`, and makes every state change on the terminal loop. It passed 20 of 20 runs with `-count=20 -race`; last round it failed 10 of 20. **BR-24 is fixed.** I deleted the prune loop in `applyWebNotes` (`console_notes.go:431-435`), and `TestWebNotesMenuSourceAndStableIDs` failed with "menu ids kept for 2 notes, want only b's". Putting the loop back made it pass. The six older open findings (BR-3, 4, 7, 12, 14, 16) are all Minor and none blocks. This round's commit changed only a test and docs, so their earlier dispositions still apply. Nothing blocks SHIP, and I found nothing new.

1. **Strengths**
   - `console_webnotes_test.go:159-237` now drives the real producer: notes go through the board with `r.note` and are resolved over HTTP, so the test checks the same path production uses, not hand-set state. It runs reliably under the race detector.
   - The prune is now covered by a test that a mutation proves: the `kept != 1` assertion goes red when the loop is removed.
   - The code for BR-3, 12, 14 and 16 is correct as written:
     - `cast.go:149-157` feeds the board before the live hub.
     - `annotate.js:298` routes every refused or failed POST through `answered()` to a toast.
     - `endGesture()` (`annotate.js:397`) is called on a mode change, a new document (`:512`), the cast's end (via `setMode`) and `pointercancel`.
   - These pass at head:
     - `go test -race` on `webcast/...` and `broadcast/...`
     - the issue's couchtty web-notes tests
     - all 131 node tests in `tests/webcast` and `tests/broadcast-viewer`

2. **Critical:** none.

3. **Important:** none.

4. **Minor:** these are carried over and still open, as listed in the dispose block. BR-3, 12, 14 and 16 have correct code but no test that fails without it. BR-4 is deferred to wave 3, and BR-7 was declined with a reason.

5. **Test coverage notes**
   - Mutation check for BR-24: red without the prune, green with it.
   - BR-25: 20 of 20 race runs pass.
   - In the full couchtty `-race` run, 4 tests failed: `TestNotificationPTYConformance`, `TestCouchPressureControl`, `TestCouchPressureTrailingPTYOutput` and `TestCouchProductionSoak`. None of their files is in this diff. The first three failed with `operation not permitted` (a `/tmp` mkdir and a pty spawn), which points to this review environment, not the change. The soak test's output did not include its failure reason.
   - The JS gesture and toast handling still has no unit test. A pure `gesture(state, event)` reducer would close BR-12, 14 and 16 together.

6. **Architecture**
   - **ARCH-DRY:** pass.
   - **ARCH-PURE:** pass for the Go code. The JS gesture logic still lives in the shell (BR-12, 14, 16).
   - **ARCH-PURPOSE:** pass. All the Plan items are delivered.
   - **ARCH-MOCK:** flag, Minor, BR-7 still open. Nothing checks that `board.Writer` and `recorder.Emit` behave the same.
   - **ARCH-CONSTRAINTS:** pass.
   - **ARCH-SECURE:** flag, Minor, BR-4 still open (the viewer-supplied box is bounded only to 1e6 px).
   - **ARCH-ORDER:** pass. Web-menu state now changes only on the terminal loop, and the test follows that rule.
   - **ARCH-FUNERAL:** pass. `webMenuIDs` is pruned on each delivery and cleared by `forgetWebNotes`.

7. **Plan revisions:** none needed.

```findings
dispose:
  - id: BR-25
    disposition: addressed
    note: |
      The test drives notes through the board and makes its state changes on the terminal loop; -race -count=20 on TestWebNotes passes 20 of 20 (it failed 10 of 20 before).
  - id: BR-24
    disposition: addressed
    note: |
      Mutation check: removing the prune loop at console_notes.go:431-435 fails TestWebNotesMenuSourceAndStableIDs ("menu ids kept for 2 notes, want only b's"), and restoring it passes.
  - id: BR-3
    disposition: not-addressed
    note: |
      Order is correct at cast.go:149-157, but still no test pins dir, board, hub; unchanged this round.
  - id: BR-4
    disposition: not-addressed
    note: |
      Still bounded only by maxCoord=1e6 (board/wire.go:25); deferred to wave 3 and unchanged.
  - id: BR-7
    disposition: not-addressed
    note: |
      Still no shared contract run over board.Writer and recorder.Emit; unchanged.
  - id: BR-12
    disposition: not-addressed
    note: |
      answered()/toast exists (annotate.js:298), but nothing tests a 409 or a rejected fetch; unchanged.
  - id: BR-14
    disposition: not-addressed
    note: |
      Code correct (annotate.js:506-512 calls endGesture on a docSeg change), but no test covers a stroke across a navigation; unchanged.
  - id: BR-16
    disposition: not-addressed
    note: |
      Code correct (setMode calls endGesture at annotate.js:372), but no test covers a drag across a mode change or the cast's end; unchanged.
```

---

## Re-review — 2026-10-12T05:13:07-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 470 — broadcast-web: pointer and notes in the viewer — anchored to element and time, multi-viewer |
| repo | pair |
| issue file | workshop/issues/000470-broadcast-web-pointer-and-notes-in-the-viewer-anchored-to-element-and-time-multi-viewer.md |
| boundary | whole-issue close |
| milestone | — |
| window | a9b2fd7d491a1a8dcd571bd63c7fd7424aff81df..2df0a13ad488f1edd8c0e2bb4e3d9c0190b3d512 |
| command | sdlc close --issue 470 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T05:13:07-07:00 |
| verdict | SHIP |

## Review

Tests pass. Next I'll revert the collab.js fix to check that the new test fails without it.

The new test fails without the fix, and the checkout is restored. Last step: check whether the issue file's Spec, Revisions or Done-when changed in this window.

```verdict
verdict: SHIP
confidence: high
```

This window has one code fix and some bookkeeping. The fix: the notes module `annotate.js` imports `./collab.js`, but only the live server served that file. The replay server (#471) serves the viewer from the shared `serveViewerFile` table, so the import 404'd there and the replay page's whole module graph failed to load. `collab.js` now lives in `serveViewerFile` (`cmd/internal/webcast/replay.go:166-171`), which both servers use. The live server reaches it through `server.go:112-115` and the replay server through `replay.go:97`. `boardAsset` now only serves `palette.json`, and its gating (only when the cast has a board) is unchanged.

I checked the fix by reverting it in the checkout. With it removed, both `TestViewerImportsAreServed` and `TestBoardAssets` fail with a 404 on `collab.js`. With it restored, `go test ./cmd/internal/webcast/ ./cmd/internal/broadcast/` passes, and the checkout is clean again.

Nothing changed scope. The issue file only records the close (status, actual hours, a Log line, and a note about what the merge exposed). `## Spec`, `## Revisions` and `## Done when` are untouched. Moving the #472 lesson in `lessons.md` changes its position only, not its text. There were no conflict resolutions listed for this window.

**Strengths**
- `TestViewerImportsAreServed` (`server_board_test.go:216-241`) follows the actual imports, starting from `viewer.js` and fetching each module through the shared table. It tests the whole class of bug ("a module only one server serves"), not just `collab.js`. It also checks that the walk actually reached `annotate.js`, `collab.js` and `replay.js`, so a regex that silently matches nothing can't make it pass.
- One function serves both servers, so there is one place to get this right (ARCH-DRY).
- The lesson at `lessons.md:1237` describes the general rule and the process gap behind it: run the live browser suites after a merge.

**Critical:** none.

**Important:** none.

**Minor**
- The test's import regex only matches `import … from './x.js'` at the start of a line. It would not follow `export {…} from './x.js'`, side-effect `import './x.js'`, or dynamic `import()`. None of these appear in `webcast/web` today, but the terminal viewer already re-exports with `export … from`. If someone adds that form, the walk would quietly stop following it.
- `collab.js` is handled as an `if` branch ahead of the `assets` map, because it comes from a different embedded filesystem. It works, but it's a second lookup path beside the table. A table entry backed by a function would match the lesson's "one table" wording more closely.

**Test coverage notes:** The fix has a regression test that fails without it (checked above). The six open Minors are untouched by this window: `annotate.js`, `cast.go` and `board/` have no changes here.

**Architecture**
- **ARCH-DRY:** pass. Both servers serve files through one function; the separate `collab.js` branch is the Minor above.
- **ARCH-PURE:** pass. The change is plain HTTP serving; the test calls the handler directly with `httptest`, with no mocks.
- **ARCH-PURPOSE:** pass. The fix covers the class: one shared table plus a test over every import, rather than just patching the replay route.

**Plan revisions:** none needed.

```findings
dispose:
  - id: BR-3
    disposition: not-addressed
    note: |
      Untouched by this window (cast.go unchanged); no test pins the order dir, board, hub.
  - id: BR-4
    disposition: not-addressed
    note: |
      Untouched by this window; still bounded only by maxCoord 1e6, deferred to wave 3.
  - id: BR-7
    disposition: not-addressed
    note: |
      Untouched by this window; no shared contract over board.Writer and recorder.Emit.
  - id: BR-12
    disposition: not-addressed
    note: |
      Untouched by this window; nothing tests a 409 or a rejected stroke chunk POST.
  - id: BR-14
    disposition: not-addressed
    note: |
      Untouched by this window; no test covers a stroke across a navigation.
  - id: BR-16
    disposition: not-addressed
    note: |
      Untouched by this window; no test covers a drag across a mode change or the cast's end.
findings:
  - id: new
    severity: Minor
    family: spec-path-untested
    title: |
      TestViewerImportsAreServed's regex follows only line-leading import-from, not export-from, side-effect or dynamic imports
    detail: |
      This is the 5th finding in family spec-path-untested. Rule: a test that walks the served-file closure parses every ES module reference form (import from, export from, bare import, import()). None of the other forms appear in webcast/web today, so the measured prevalence is 0 of 6 modules, but the terminal viewer already uses export from (broadcast/web/viewer.js:10). Widen the regex or fail on an unrecognised './*.js' reference.
```

---

## Re-review — 2026-10-12T05:45:09-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 470 — broadcast-web: pointer and notes in the viewer — anchored to element and time, multi-viewer |
| repo | pair |
| issue file | workshop/issues/000470-broadcast-web-pointer-and-notes-in-the-viewer-anchored-to-element-and-time-multi-viewer.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6a8597f15e6d7921954ae4b0416cd5b96a48138a..6c98f0479540ed28a6433911c36752be42e35255 |
| command | sdlc close --issue 470 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T05:45:09-07:00 |
| verdict | SHIP |

## Review

Interdiff is just the merge-conflict resolution; it builds and webcast tests pass. Now checking whether the open findings were fixed at head.

Wrapping up: the interdiff is only the conflict resolution, so I'm confirming the issue's Spec wasn't touched, then writing the verdict.

```verdict
verdict: SHIP
confidence: high
```

This window contains only the merge-conflict resolution against main (#478 fidelity work). It adds no fixes and no scope. Base `6a8597f1` had conflict markers committed in three files. Head resolves all three and keeps both sides:

- **`cast.go`**: the `Cast` struct has `board` from #470 and `recDir`/`check`/`logf` from #478. `StartCast` sets `recDir` and builds `recordingOutput(cfg, liveSink, bd, id, proxy)`, and `Cast{...}` sets `board: bd, recDir:, check:, logf:` (`cmd/internal/webcast/cast.go:91-134`). `NewServer(...).WithBoard(bd)` is kept.
- **`server.go`**: the `//go:embed` list and the `assets` table are a union of both sides: annotate.js/css from #470, replayer.js, fidelity.html and fidelity.js from #478.
- **`atlas/webcast-format.md`**: both changelog entries are kept.

The `lessons.md` change only moves the #478 lesson below the #470 lessons. The issue file changed only in close metadata; `## Spec`, `## Revisions` and `## Done when` are untouched, so the scope did not change.

What I checked:
- No conflict markers remain in tracked non-workshop files at head.
- `go vet ./cmd/internal/webcast/...` is clean.
- `go test ./cmd/internal/webcast/...` passes in all 10 packages.
- `go build ./...` fails only on `runtimebundle`'s missing embed. That directory is gitignored (`.gitignore:127`), so the failure comes from this checkout's environment, not the diff.
- fidelity.js imports replayer.js; replayer.js imports model.js, render.js and replay.js. All of these are in the merged `assets` table, so the fidelity page's import closure is served.

1. **Strengths**
   - The resolution keeps both sides' intent exactly, with no dropped field, sink or served file (`cast.go:133-135`, `server.go:17,30-44`).
   - `recordingOutput` still puts the board in the Tee before the live fan-out (`cast.go:166-172`), which keeps the BR-3 ordering intact through the merge.
   - The `assets` table now has a single list covering both features' files, consistent with the "one served-file table" lesson.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings:** none new.

5. **Test coverage notes**
   - The existing webcast suites pass after the merge.
   - The open Minor findings BR-3, BR-12, BR-14 and BR-16 have code fixes at head but no regression test. The tests that would show them are: a stroke POST refused with a visible toast, a navigation or mode change in the middle of a gesture, and the order of sinks in the Tee.

6. **Architectural notes**
   - **ARCH-DRY:** pass. The resolution merges the embed list and the asset table without duplicating either.
   - **ARCH-PURE:** pass. No logic moved into IO code.
   - **ARCH-PURPOSE:** pass. Both features stay fully wired after the merge.

7. **Plan revision recommendations:** none.

```findings
dispose:
  - id: BR-3
    disposition: not-addressed
    note: |
      cast.go:166-172 puts the board before the live sink in the Tee, but no test pins the order; still Minor.
  - id: BR-4
    disposition: not-addressed
    note: |
      board/wire.go:25 still bounds coordinates to maxCoord 1e6, with no clamp to the element or page box; deferred to wave 3.
  - id: BR-7
    disposition: not-addressed
    note: |
      There is still no shared contract test run over both board.Writer and recorder.Emit.
  - id: BR-12
    disposition: not-addressed
    note: |
      annotate.js:385 now sends stroke posts through answered() (toast on non-204 or network failure), but no test covers a refused stroke chunk.
  - id: BR-14
    disposition: not-addressed
    note: |
      onRecord now calls endGesture() when docSeg changes (annotate.js:506-512), but no test drives a stroke across a navigation.
  - id: BR-16
    disposition: not-addressed
    note: |
      endGesture() exists and is called from setMode, cast end, pointercancel and a document change (annotate.js:372,397,447,512), but no regression test exercises it.
  - id: BR-26
    disposition: not-addressed
    note: |
      server_board_test.go:220 still matches only line-leading import-from; export-from, bare and dynamic imports are not followed.
```
