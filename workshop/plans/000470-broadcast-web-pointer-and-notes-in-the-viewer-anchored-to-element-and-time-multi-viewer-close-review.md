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
