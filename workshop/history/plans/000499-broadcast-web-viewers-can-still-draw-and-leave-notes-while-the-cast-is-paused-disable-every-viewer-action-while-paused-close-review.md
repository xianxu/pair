# Boundary Review — pair#499 (whole-issue close)

| field | value |
|-------|-------|
| issue | 499 — broadcast-web: viewers can still draw and leave notes while the cast is paused; disable every viewer action while paused |
| repo | pair |
| issue file | workshop/issues/000499-broadcast-web-viewers-can-still-draw-and-leave-notes-while-the-cast-is-paused-disable-every-viewer-action-while-paused.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3ef92d4d4ff488c3baeef21a4fc03ebe65d69d5e..511a17a9ac9f7f4200bdc7fcf9ed758ef4d2af84 |
| command | sdlc close --issue 499 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T15:40:25-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Tests pass. I'm now reverting the `OnDoc` fix to check that the recorder test fails without it, and checking which sentence a dropped record gets back.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** The server-side guard is correct and well tested. `Board.pageLocked` refuses all three viewer routes while paused. `recording.State.OnDoc` now requires `Live`, so a record admitted just before the pause can't be written during it, and the recorder and `board.Writer` share that one check. I confirmed the regression test is real: with the old `OnDoc` restored, `TestRecorderEmitDropsARecordDuringAPause` fails ("records from the pause were written: … stop:pause note stroke …"). With the fix in place, `go test ./cmd/internal/webcast/...` passes. What stops a plain SHIP is a gap in the viewer. The rail is disabled, but the note dialog is not. A dialog that is open when the pause arrives, or a pin opened during the pause, still shows active Save and Resolve buttons. The README also says viewers "can't … resolve notes: the rail's controls are greyed out", but Resolve isn't on the rail. The server still refuses those posts, so the cost is user experience, not correctness. The "operator smoke-tested it" clause can't be checked from the code.

1. **Strengths**
   - One rule shared by both writers (ARCH-DRY): `OnDoc` (`recording/reduce.go:125`) is the single admission check for `recorder.go:514` and `board/writer.go:40`. `Live` covers the old `Seg > 0 && !Ended`, because `Ended` always means `!Live` and `Live` is only set by a `Seg`.
   - `pageLocked` (`board/board.go:284`) checks `ErrPaused` after `ErrEnded`/`ErrNotLive`, so an ended cast still answers 410, not 409.
   - `TestDocumentRules` exercises both states: everything refused while paused, and after the resume only the pre-pause note is live and no stroke was recorded. `TestBoardRoutesRefuseWhilePaused` checks all three routes for the 409 status and the exact sentence.
   - In `annotate.js`, `onRecord` runs after the model applies the record, so `paused()` reads the current state. Calling `setMode('watch')` on `stop` also ends any gesture in progress.
   - The plan checked for a terminal-broadcast pause state rather than assuming one ("no pause anywhere in cmd/internal/broadcast").

2. **Critical:** none.

3. **Important**
   - **Note dialog controls stay active while paused** (`web/annotate.js:531`, `:486-490`). The dialog only closes on `stop` with `why === 'end'`. The pin chip handler still opens `dialog.view(..., { resolvable: true })` while paused. The README line at `README.md:1044` claims resolving is disabled through the rail. Every viewer-action control outside the rail is affected:
     - Save on an open draft dialog;
     - Resolve on an open viewing dialog;
     - Resolve on a pin dialog opened during the pause.

     Fix: on `stop`, close the dialog (or disable `save`/`resolve` while `paused()`), and pass `resolvable: !paused()` from the pin callback. Extend the live Chrome test to cover it.

4. **Minor**
   - Stale comments: `recorder.go:515` ("The viewer's document has gone") and the `Writer.Emit` doc at `board/writer.go:35-36` ("isn't current any more") now also cover the paused case.
   - A request admitted just before the pause and dropped by `board.Writer` comes back as `ErrStale`, so the viewer sees "The page changed. Try again." instead of "The cast is paused."
   - On pause, `setMode('watch')` → `endStroke` flushes the remaining stroke chunks. The server refuses them, so a viewer who was drawing sees the "The cast is paused." toast. That's acceptable, but not mentioned anywhere.

5. **Test coverage**
   - The Board, the routes and the recorder are covered in both states (paused and resumed).
   - The viewer's rail disable/re-enable is only covered by the opt-in test `WEBCAST_LIVE_CHROME`, which I didn't run. The dialog path isn't covered at all.
   - `board.Writer` dropping a paused record has no direct test. `TestDocumentRules` refuses at the Board, so it never reaches the Writer.

6. **Architecture**
   - **ARCH-DRY: pass.** One `OnDoc` rule serves both writers; `refusePaused` follows the existing `boardAnswer` table.
   - **ARCH-PURE: pass.** The admission logic is a pure state check, tested without IO.
   - **ARCH-PURPOSE: flag.** The issue asks to "disable every viewer action". The rail is done, but the dialog's Save and Resolve are viewer actions too (see the Important finding).

7. **Plan revisions**
   - Add a `## Revisions` entry: the viewer also closes or disables the note dialog's Save/Resolve while paused, so the plan stops implying the rail is the only viewer action surface.

```findings
findings:
  - id: new
    severity: Important
    family: paused-disables-every-viewer-action
    title: |
      Note dialog Save/Resolve stay active while paused; README claims resolve is disabled
    detail: |
      annotate.js closes the dialog only on stop:end (line 531), and the pin callback (486-490) still opens it with resolvable:true during a pause. Instances: Save on an open draft, Resolve on an open view, Resolve on a pin opened while paused. README.md:1044 says resolve is disabled via the greyed rail, but the rail has no resolve. Server still refuses (409). Fix: close the dialog or disable save/resolve while paused(), use resolvable:!paused(), and extend the live Chrome test.
  - id: new
    severity: Minor
    family: doc-claims-track-code
    title: |
      Stale comments and stale-page sentence for the paused writer drop
    detail: |
      recorder.go:515 ("The viewer's document has gone") and board/writer.go Emit doc ("isn't current any more") now also cover the paused case. A Writer drop during a pause returns ErrStale, so the viewer sees "The page changed. Try again." instead of "The cast is paused."
```

---

## Re-review — 2026-10-12T16:05:53-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 499 — broadcast-web: viewers can still draw and leave notes while the cast is paused; disable every viewer action while paused |
| repo | pair |
| issue file | workshop/issues/000499-broadcast-web-viewers-can-still-draw-and-leave-notes-while-the-cast-is-paused-disable-every-viewer-action-while-paused.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0cddecbbcecbbff04453fed5aa755d70f5ee9f4b..95684dc8e3547b33f8a33bf6ed76f2bf3043165f |
| command | sdlc close --issue 499 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T16:05:53-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Diff and issue read; now checking annotate.js ordering (dialog vs. renderRail) and keyboard/pin paths.

Go tests pass for webcast (the one broadcast failure is an unrelated cloudflared socket test). Checking the node unit tests for `noteDialog`.

Trying the opt-in live Chrome test that is the BR-1 regression evidence.

The live Chrome test can't run here (CDP EOF — environmental). I'll check `hold()` directly with a small node script against the existing test fakes.

`hold()` behaves correctly. Now checking BR-2's production path: what the recorder returns to the board on a pause drop.

BR-2 checks out (the recorder drops silently, and `board.Writer` now returns `ErrPaused`). Last check: the plan's claim that the terminal broadcast has no paused state.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Verdict: fix then ship.** The pause rule works at both server layers. The board refuses `stroke`, `note` and `note-state` with `ErrPaused` while paused. `recording.State.OnDoc` now requires `Live`, so the recorder and the reference `board.Writer` also drop a record that was admitted just before a pause. On the viewer, the rail is disabled and the note dialog's Save and Resolve are held. Both prior findings are fixed. `go test ./cmd/internal/webcast/...` passes, and the 71 node viewer tests pass. I checked `noteDialog.hold()` directly with a small node script using fake elements. While held, Save and Resolve are disabled and nothing gets posted. A draft keeps its text. A pin opened during the pause shows Resolve disabled. Releasing the hold turns both back on.

What remains before shipping is cheap. The BR-1 fix is only covered by the opt-in live Chrome test, which I couldn't run here (it failed to start with a CDP EOF error, an environment problem). There's also a small set of stale comments and a repeated "is paused" check.

1. **Strengths**
   - **One rule at two layers.** The board rejects the request up front (`board.go:284`), and when the record is written, `OnDoc` (`reduce.go:125`) rejects anything that slipped through, as one shared check. `OnDoc`'s new form is equivalent to the old one: `Live` already implies `Seg>0` and not ended.
   - **Writer drop message.** `board.Writer.Emit` (`writer.go:42`) tells a pause drop apart from a page change. `TestWriterRefusesARecordDuringAPause` pins both cases.
   - **Dialog hold.** `hold()` is a small, focused addition to the shared dialog (`collab.js:274`). The terminal broadcast is unaffected because its `held` is always `''`.
   - **Tests across layers.** Board unit tests, a recorder test, an HTTP test covering all three routes, and a live Chrome test that checks both paused and resumed states.
   - **No terminal pause.** The plan says the terminal broadcast has no paused state. Confirmed: no file under `cmd/internal/broadcast` outside vendor mentions pause, apart from the new `collab.js` comment.

2. **Critical:** none.

3. **Important**
   - **No default-run test for `hold()`.** The only regression test for BR-1 is the opt-in `WEBCAST_LIVE_CHROME` test (`viewer_annotate_live_test.go:201`). `tests/broadcast-viewer/notes.test.mjs` already exercises `noteDialog` with fake elements and runs by default. Add `hold()` cases there:
     - a draft: Save disabled, message shown, text kept, release turns Save back on;
     - `view()` while held: Resolve disabled and a click posts nothing;
     - releasing the hold clears the message.

     Instances in this window: `hold()` itself, plus the held branches in `draft()` and `view()` (`collab.js:246`, `260`) and the Resolve click guard (`collab.js:229`). The `annotate.js` `paused()` sites need a DOM, so the live test remains their evidence.

4. **Minor**
   - **Stale or truncated comments.** This is the 2nd finding in family `doc-claims-track-code`. The rule: when a diff changes what a function does, re-read every comment that sits on or names that code in the same commit. Instances:
     - `collab.js:180`: the comment is cut off ("why Save and Resolve are off, or ").
     - `annotate.js:354`: the comment "rendered only when their state changes (a pick, a mode)" no longer covers pause and resume. The new `paused` comment and declaration now sit between it and `renderRail`.
   - **ARCH-DRY: the "is paused" check is written out in several places.**
     - `writer.go:42`: `Seg > 0 && !Live && !Ended && DocSeg == doc`
     - `board.go:284`: the same condition, implied by the order of the switch cases
     - `annotate.js:359`: `seg > 0 && !live && !ended`
     - `viewer.js:144`: `s.page && !s.live`

     Add a `recording.State.Paused()` method for the Go sites, and one shared helper on the JS side.

5. **Test coverage notes**
   - Done-when clause 1 is covered in both states. Paused requests are refused (`TestDocumentRules`, `TestBoardRoutesRefuseWhilePaused`). Nothing from the pause survives the resume (`TestDocumentRules`, `TestRecorderEmitDropsARecordDuringAPause`). The controls are disabled, then enabled again (the live test).
   - Done-when clause 2 ("the operator smoke-tested it") isn't recorded in the issue Log yet.
   - `go test ./cmd/internal/broadcast` fails `TestNamedTunnelServesFromPrivateSocket` (cloudflared). That test and its package are outside this diff.

6. **Architectural notes**
   - In production the recorder drops a pause-raced record silently, so the viewer gets 204 and the note never appears. This matches the existing navigation race and is acceptable, but the atlas could say the drop is silent.
   - A pause now sends every viewer back to watch mode, and that also happens when a reconnect replays an old stop record. They have to pick the pen again after a resume. That's acceptable under the spec.

7. **Plan revisions:** none. The plan and Log match the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      noteDialog.hold (collab.js:274) is called from renderRail on every stop/seg; held Save/Resolve verified via node fakes (no post, draft text kept, release re-enables); pin view while held shows Resolve disabled. Regression evidence is only the opt-in live Chrome test (see new finding).
  - id: BR-2
    disposition: addressed
    note: |
      recorder.go:515 comment names the pause; writer.go Emit doc updated and returns ErrPaused for a pause drop, pinned by TestWriterRefusesARecordDuringAPause (ErrPaused paused, ErrStale after navigate).
findings:
  - id: new
    severity: Important
    family: regression-test-runs-by-default
    title: |
      noteDialog.hold has no default-run regression test; only the opt-in WEBCAST_LIVE_CHROME test covers BR-1
    detail: |
      tests/broadcast-viewer/notes.test.mjs already drives noteDialog with fake elements and runs by default. Add hold() cases there: a held draft disables Save, shows the reason and keeps the text, and release re-enables it; view() while held disables Resolve and a click posts nothing; release clears the message. Instances: hold() at collab.js:274, the held branches in draft()/view() at collab.js:246 and 260, and the resolve click guard at collab.js:229. The live test could not launch Chrome in this environment (CDP EOF), so nothing that runs by default protects the fix.
  - id: new
    severity: Minor
    family: doc-claims-track-code
    title: |
      Truncated held comment in collab.js; annotate.js renderRail comment is stale and separated from its function
    detail: |
      This is the 2nd finding in family doc-claims-track-code. Rule: when a diff changes what a function does, re-read and update every comment that sits on or names that code in the same commit. Instances in this window: collab.js:180 ("why Save and Resolve are off, or " is cut off); annotate.js:354 ("rendered only when their state changes (a pick, a mode)" now also changes on pause/resume, and the new paused() comment and const sit between it and renderRail).
  - id: new
    severity: Minor
    family: state-predicate-single-source
    title: |
      ARCH-DRY: the paused predicate is restated in writer.go, board.go, annotate.js and viewer.js
    detail: |
      writer.go:42 (Seg>0 && !Live && !Ended && DocSeg==doc), board.go:284 (implied by switch order), annotate.js:359 (seg>0 && !live && !ended), viewer.js:144 (page && !live). Add a recording.State.Paused() method for the Go sites, and one JS helper (e.g. in model.js) for the viewer sites.
```

---

## Re-review — 2026-10-12T16:45:06-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 499 — broadcast-web: viewers can still draw and leave notes while the cast is paused; disable every viewer action while paused |
| repo | pair |
| issue file | workshop/issues/000499-broadcast-web-viewers-can-still-draw-and-leave-notes-while-the-cast-is-paused-disable-every-viewer-action-while-paused.md |
| boundary | whole-issue close |
| milestone | — |
| window | 79ce19ccc860d16b82d754a2e9f8595535cc9bde..366152fc9adb5d7f317f19861053053147f3038f |
| command | sdlc close --issue 499 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T16:45:06-07:00 |
| verdict | SHIP |

## Review

Tests pass; now reverting the hold fix piece by piece to confirm the new default-run test goes red.

```verdict
verdict: SHIP
confidence: high
```

All three open findings from earlier rounds are fixed, and I found nothing new. While a cast is paused, the server refuses all three viewer routes: `stroke`, `note` and `note-state`. A record the board accepted just before the pause is dropped by both writers, because `OnDoc` now also requires the cast to be live. On the viewer side, the rail and the note dialog are disabled during a pause. In this checkout, `go test ./cmd/internal/webcast/...` and `node --test tests/broadcast-viewer/notes.test.mjs` both pass (15/15). I also checked the new default-run test: I removed each of the three dialog guards in turn, and each removal made it fail. One `## Done when` clause can't be checked here: "the operator smoke-tested it."

**1. Strengths**
- One rule is enforced in three places: the board's admission check (`board.go:284`), the recorder's `OnDoc` check (`reduce.go:315`) and the reference writer (`writer.go:42`). The recorder test sends records after the `Pause`, which covers the race between the board's check and the write.
- `noteDialog.hold` (`collab.js:274`) calls `say` only when the reason changes. Because `renderRail` calls it on every render, an unchanged reason won't overwrite a server error message.
- `TestBoardRoutesRefuseWhilePaused` tests all three routes and the exact 409 message.
- The notes test checks both states the `## Done when` clause names: held and released, for a draft and for a viewed note.

**2. Critical findings:** none.

**3. Important findings:** none.

**4. Minor findings:** none new.

**5. Test coverage notes**
- **BR-3 is fixed.** Three separate edits to `collab.js` each make the new test in `notes.test.mjs` fail (14 pass, 1 fail):
  - dropping `held` from Save's disabled rule;
  - dropping `held === ''` from the Resolve click guard;
  - deleting `els.resolve.disabled = held !== ''`.
- The live Chrome test (`WEBCAST_LIVE_CHROME`) is still opt-in, and I did not run it here. The default-run node test now covers the dialog by itself.

**6. Architectural notes**
- **ARCH-DRY: pass.** Go has `recording.State.Paused()`, used by both board sites. The viewer has `Model.paused`, used by `annotate.js`.
  - `viewer.js` still has its own paused checks: `view()` at lines 40–41 and the `s.page && !s.live` toggle at line 144. Neither is new in this diff, and `view()` builds the status-bar state that live casts and replays share. I accept the stated reason for keeping them.
- **ARCH-PURE: pass.** The paused rule is a pure predicate. `hold` is dialog state tested with fake elements, and the IO stays in `annotate.js`.
- **ARCH-PURPOSE: pass.** Every surface that posts is now gated: the rail, pointer-down on the page, the dialog's Save and Resolve, and the server routes. Viewers on the terminal broadcast stay enabled, but that's correct: it has no paused state, as the plan says.

**7. Plan revision recommendations:** none. The plan and `## Log` match the code.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      notes.test.mjs hold test runs by default; reverting each of the 3 guards in collab.js (Save rule, Resolve click guard, resolve.disabled) turns it red (14 pass / 1 fail).
  - id: BR-4
    disposition: addressed
    note: |
      collab.js:56 held comment is complete; annotate.js renderRail comment names pause/resume and directly precedes renderRail, with paused() moved above it.
  - id: BR-5
    disposition: addressed
    note: |
      recording.State.Paused() used by board.go:284 and writer.go:42; Model.paused used by annotate.js; viewer.js view() is pre-existing show-state shared with replay, kept with a stated reason.
```
