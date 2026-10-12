# Boundary Review — pair#470 (milestone M2)

| field | value |
|-------|-------|
| issue | 470 — broadcast-web: pointer and notes in the viewer — anchored to element and time, multi-viewer |
| repo | pair |
| issue file | workshop/issues/000470-broadcast-web-pointer-and-notes-in-the-viewer-anchored-to-element-and-time-multi-viewer.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 97a667ee330c1cd38a1f9da1b886849f58d91238..9f8ed7e25c848d6eba162e971f5848f44be1808b |
| command | sdlc milestone-close --issue 470 --milestone M2 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T03:46:27-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the M2 diff now — reading the new annotate.js and its tests next.

Checking the wire parsers match what the viewer posts, then running the tests.

Running the node and Go tests now.

Tests are green. Now checking a suspected bug: whether a late joiner's catch-up redraws every stroke on the page.

Trying the headless two-viewer e2e test once.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M2 delivers what the plan says it will. `collab.js` is extracted and both viewers share it. The terminal viewer still passes all 60 of its node tests, and its storage-policy test now covers `collab.js` too. `annotate.js` keeps its pure helpers separate from a self-contained mount, and `viewer.js` only gains `fromPage` and a single mount call. The atlas is updated. `go test ./cmd/internal/webcast/... ./cmd/internal/broadcast/...` passes and the webcast node tests pass (40/40).

I could not run the headless e2e (`TestAnnotateLiveTwoViewers`) here: Chrome fails at `Target.createTarget: cdp: read: EOF`. The existing `TestLiveViewerPage` fails the same way, so this is the review environment, not the change.

Two things should be fixed before shipping, and neither is a rework:
- **Catch-up strokes are drawn anyway.** The rule meant to hide them depends on arrival order, so a late joiner sees every stroke on the current page flash at once. I reproduced this.
- **Opening a note isn't safe or tested.** Tapping a pin can fail, and no test opens or resolves a note from the viewer.

### 1. Strengths
- **Shared module done right.** `collab.js` is one file, served by both servers (`broadcast.CollabJS`, `server_board.go:122`). It's a symlink in `webcast/web/`, so the node tests read the same bytes. `noteDialog.draft(payload)` plus `resolveRequest` and `beforeSave` cleanly generalize #447's dialog without forking it.
- **Mounting is gated and safe.** `palette.json` answers only when the cast has a board, and the module refuses a palette that isn't all `#rrggbb`. Note text and author only ever reach the page via `textContent`, and a pin's color is checked against `HEX_COLOR` (`annotate.js:359`).
- **The wire matches the parsers.** The fields posted for stroke, note and note-state match `board/wire.go`'s strict parsers. The stroke-id prefix `<viewer>.` can't collide because viewer ids can't contain `.`.
- **The e2e test is strong.** It runs two real Chromes, one using touch, and checks a pin's position to within ±2 px against the operator's layout. It also covers pause keeping notes, navigation clearing them, and the recording reducing with both authors.

### 2. Critical findings
None.

### 3. Important findings
- **I-1: catch-up strokes are drawn** (`annotate.js:94-99`, ARCH-ORDER). The filter compares each stroke with the newest time seen *so far*. The catch-up arrives oldest first (`catchup.go:82` keeps every stroke of the current page), so each old stroke is "newest" when it arrives.
  - Reproduction: catch-up strokes at t=1000 and t=31000, then a record at t=61000, leaves both strokes drawn.
  - Effect: every new viewer, and every reconnect, sees all of the page's strokes appear and fade together. This contradicts the plan's rule for catch-up strokes.
  - The node test (`annotate.test.mjs:50`) calls `seen(60000)` *before* the old stroke, an order a real catch-up never produces, so it only confirms the implementation.
  - Fix: store each stroke's end time and drop it in `visible()` when `newest - end > FADE_MS`. That moves the decision to draw time, after the burst has arrived. Then test with records in stream order.
- **I-2: tapping a pin is fragile** (`annotate.js:282-285, 350-376`; plausible, not reproduced).
  - `drawPins` rebuilds every pin each frame, and frames keep running while any stroke is visible. A tap that spans a rebuild loses its click, so pins can't be opened while someone draws.
  - The chips sit inside `.annotate`. In point or note mode, the layer's `pointerdown` handler captures the pointer, so tapping a chip likely starts a stroke or a new draft instead of opening the note.
  - Fix: ignore `pointerdown` whose target is inside `.chip`. Rebuild the pin list only when the set of live notes changes, and otherwise just reposition the pins.
- **I-3: no test opens or resolves a note in the viewer.** The spec says resolved notes hide. The `resolveRequest` → `note-state` → pin-gone path in the viewer is never exercised, and nothing covers I-2. Fix: extend the e2e so B taps A's pin, checks the dialog shows "Ana" and the text, presses Resolve, and checks the pin disappears in both viewers.
- **I-4: README update appears missing** for viewer pointing and notes: the ✏️/💭 rail, the name prompt, and the fact that web-cast notes and strokes are **recorded to disk**. The README's #447 paragraph says terminal notes are never saved to disk, so readers may assume the same here. This could be deferred to M3, but the plan should say so.

### 4. Minor findings
- **Refused stroke chunks are swallowed silently.** In `annotate.js:271`, `.catch(()=>{})` drops errors, and 429/409/410 answers are ignored. The person drawing sees the whole stroke while others see gaps, which is plausible with three or more viewers against the 60/s cast budget.
  - This is the 2nd finding in family `silent-degradation`. The rule: every viewer POST whose answer isn't 204 shows at least one visible signal. The dialog already does this for notes and states; the stroke path is the one exception.
- **ARCH-DRY: near-duplicate code.**
  - `postJSON` (`annotate.js:157`) duplicates `broadcast/web/viewer.js:352`. The policy test that forbids `fetch(` in `collab.js` is what forces the copy.
  - `identityStore`'s storage try/catch repeats `colorStore`'s.
  - The sticky-note and swatch CSS is restated in `annotate.css` and `viewer.css`.
- **A stroke can span a navigation.** A stroke still in progress across a navigation keeps posting with the new `v.model.seg`, so the rest of it lands on the new page. Call `endStroke()` when `docSeg` changes.
- **Remote stroke color isn't validated.** `r.color` reaches `strokeStyle` without the `HEX_COLOR` check pins get. This is harmless because canvas ignores invalid colors, but it's inconsistent.

### 5. Test coverage notes
- The pure helpers (identity, chunker, liveNotes, noteBox, regionOf) are well pinned. The strokeStore catch-up case is not (I-1).
- The shell's own logic is covered only by the env-gated e2e test, which doesn't run in default CI. The view/resolve and pin-tap paths have no coverage at all (I-3).

### 6. Architecture notes
- **ARCH-DRY:** flagged (Minor).
- **ARCH-PURE:** passes. The helpers are pure and tested without IO, and the IO is confined to `mountAnnotate`.
- **ARCH-PURPOSE:** passes for M2's scope, with the resolve gap (I-3).
- **ARCH-MOCK:** passes. The e2e runs through Server, Hub, Board and Writer, the same seam as `recorder.Emit`.
- **ARCH-CONSTRAINTS:** a caution. The stroke budget is cast-wide at 60/s, so three viewers drawing at once saturate it. Pins are rebuilt every frame while strokes animate (I-2).
- **ARCH-SECURE:** passes. Text uses `textContent`, the palette and pin colors are validated, the reserved `operator` id is never reused, and the server-side parsers are strict.
- **ARCH-ORDER:** flagged (I-1). A decision that needs the stream's newest time is made before that time is known.
- **ARCH-FUNERAL:** passes. Faded strokes are pruned, strokes are cleared on a new page, and localStorage holds two named keys.

For M3: Couch's `💭N` should read `Board.Subscribe`, not the viewer's model.

### 7. Plan revision recommendations
- Add a Revisions entry recording the actual mount design: `palette.json` as the board probe, the `collab.js` symlink, and `beforeSave`/`resolveRequest` on `noteDialog`.
- Restate the catch-up stroke rule as "evaluated at draw time against the newest record received".
- Say whether the README update lands in M2 or M3.

```findings
findings:
  - id: new
    severity: Important
    family: filter-on-partial-stream
    title: |
      strokeStore drops catch-up strokes only when a newer record arrived first; a late joiner redraws every stroke of the document
    detail: |
      annotate.js:94-99 compares each stroke with the running newest time at arrival; CatchUp replays the document's strokes oldest-first, so each passes. Reproduced: catch-up strokes at t=1000 and t=31000 then a record at t=61000 leave both drawn. The node test seeds seen(60000) before the old stroke, an order a real catch-up never has. Fix: keep each stroke's end time and filter in visible() at draw time; test in stream order.
  - id: new
    severity: Important
    family: pin-input-routing
    title: |
      Pin chips are rebuilt every frame and swallowed by the layer's pointerdown in point/note mode
    detail: |
      drawPins replaceChildren()s every frame while strokes are visible, so a tap spanning a rebuild loses its click. In point/note mode a chip pointerdown bubbles to the layer handler, which captures the pointer and starts a stroke or region draft instead of opening the note. Fix: ignore pointerdown targets inside .chip; rebuild pins only when the live set changes.
  - id: new
    severity: Important
    family: spec-path-untested
    title: |
      No test opens a pin or resolves a note from the viewer (Spec: resolved notes hide)
    detail: |
      The e2e never taps a chip or presses Resolve; the viewer's note-state request and the pin disappearing in both viewers are unverified. Extend TestAnnotateLiveTwoViewers to cover this.
  - id: new
    severity: Important
    family: readme-surface-gap
    title: |
      README not updated for web cast pointing and notes (rail modes, name prompt, notes and strokes recorded to disk)
    detail: |
      README's #447 paragraph says terminal notes are never saved to disk; web cast notes are recording records. Add the viewer surface, or state in the plan that M3 adds it.
  - id: new
    severity: Minor
    family: silent-degradation
    title: |
      Refused or failed stroke chunk POSTs are swallowed with no viewer signal
    detail: |
      2nd finding in family silent-degradation. The rule: every viewer POST answered with something other than 204 produces at least one visible signal. The note dialog follows it; annotate.js:271 (.catch(()=>{}), non-204 ignored) is the remaining exception. Prevalence in this diff: 1 of 3 POST paths.
  - id: new
    severity: Minor
    family: shared-helper-duplication
    title: |
      postJSON, the storage try/catch wrapper and the sticky-note/swatch CSS are restated across both viewers
    detail: |
      annotate.js:157 duplicates broadcast/web/viewer.js:352; identityStore repeats colorStore's guards; annotate.css restates viewer.css's dialog and swatch styles (ARCH-DRY).
  - id: new
    severity: Minor
    family: stroke-spans-document
    title: |
      A stroke in progress across a navigation posts its later chunks with the new seg
    detail: |
      Call endStroke() when onRecord sees docSeg change.
```

---

## Re-review — 2026-10-12T04:02:36-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 470 — broadcast-web: pointer and notes in the viewer — anchored to element and time, multi-viewer |
| repo | pair |
| issue file | workshop/issues/000470-broadcast-web-pointer-and-notes-in-the-viewer-anchored-to-element-and-time-multi-viewer.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 97a667ee330c1cd38a1f9da1b886849f58d91238..edb470f14c8f1b628e9ec8a88bfcdaac8db89cfe |
| command | sdlc milestone-close --issue 470 --milestone M2 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T04:02:36-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Still reviewing: I've read annotate.js and its tests, and now I'm checking the record model and the e2e.

Node invocation with directories fails; running the files directly.

Chrome won't launch in this environment, and the older live test fails the same way. Next I'll check the BR-8 fix by reverting it and confirming the node test goes red.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M2 delivers what it set out to do. `collab.js` is now shared by both viewers. `annotate.js` keeps its pure helpers apart from a single mount shell. The e2e covers both viewers (one by mouse, one by touch) on the blog fixture: a stroke, an element note, a region note, a pause, a resolve and a navigation. Four of the seven prior findings are fixed with evidence: BR-8, BR-9, BR-10 and BR-11. BR-13 is fixed for the JS; the CSS stays duplicated, with a stated reason. BR-12 and BR-14 have code fixes but no test that fails without them. One new Important finding should be fixed before the gate. It is the same class as BR-9: on the per-frame path, `drawRail()` rebuilds the swatch buttons, so a click on a swatch is likely lost whenever records are streaming.

**Verification done here:**
- Ran the stat and name-status recipes, then read the targeted diffs.
- `go test ./cmd/internal/webcast/... ./cmd/internal/broadcast/...` passes.
- The node tests pass: `tests/webcast/*.mjs` and `tests/broadcast-viewer/*.mjs`, 101 of 101.
- BR-8 mutation: I removed the `newest - s.end > FADE_MS` check in `visible()`. `annotate.test.mjs` then fails, and passes again once restored.
- `TestAnnotateLiveTwoViewers` could not run. Chrome dies at `Target.createTarget` (`cdp: read: EOF`) in this environment, and the older `TestLiveViewerPage` fails the same way, so the cause is the environment. The e2e evidence for BR-9 and BR-10 is read from the code, not executed.

### 1. Strengths
- **Catch-up filter at draw time.** `annotate.js:113` checks each stroke's end against the newest record time at draw time, and the node test replays a catch-up in stream order (`annotate.test.mjs:49-57`).
- **Pins kept per note id.** `annotate.js:385-396` adds and removes pins by note id and only moves the rest each frame. The `.chip` check at `:291` lets a pin's chip be used in every mode.
- **Segment captured at stroke start.** `seg: s.seg` (`:279`, `:299`) means chunks posted after a navigation carry the old segment, and the board refuses them with `ErrStale`. The client can't file a stroke on the wrong page.
- **The e2e checks the right things.** Touch input on viewer B, the pin within 2 px of its element, a resolve by touch seen in both viewers and in the recording, and a check that the stroke spans the whole drag (`viewer_annotate_live_test.go:87-101`). That last check catches the first-echo bug the implementor found.
- **`collab.js` extraction.** `poster` and `safeStorage` are now the single helpers both viewers use, and the policy test in `server_test.go:184` was moved to follow them.

### 2. Critical findings
None.

### 3. Important findings
- **Swatch buttons are rebuilt every frame, so swatch clicks are likely lost** (`annotate.js:255` inside `drawRail`, called from `schedule` at `:432`).
  - `schedule()` runs on every record (`:450`) and keeps running while strokes are visible. Each run calls `renderSwatches`, which calls `replaceChildren` on the swatches. A cast with activity streams records many times a second.
  - A click whose press and release land on either side of a rebuild goes to the common ancestor, which has no listener. So picking a color during activity mostly fails. No test clicks a swatch.
  - **This is the 2nd finding in family `pin-input-routing`.** The rule: nothing that takes input is recreated on the per-frame path. Frame work only moves or restyles existing elements; elements are rebuilt only when their data changes.
  - Sweep of this module: pins are fixed (BR-9); swatches violate the rule; mode buttons only toggle classes, which is fine; the dialog is never rebuilt.
  - Fix: call `renderSwatches` only at mount and on pick. Keep only placement and `hidden` in `drawRail`. Add an e2e step that picks swatch 3 while viewer A's stroke is fading, then asserts the posted `color`.

### 4. Minor findings
- **`setMode` ends a stroke but not a region drag** (`annotate.js:266-272`).
  - A second finger tapping a mode button, or `stop end` arriving mid-drag, leaves `state.drag` set. The next pointerup then opens a draft in the wrong mode, or after the cast has ended.
  - This is the 2nd finding in family `stroke-spans-document`. The rule: one `endGesture()` ends the stroke and the drag, and every boundary calls it (new document, mode change, cast end, pointercancel).
  - Related: a resync that lands on a different document goes through `docSeg` 0, so `was && docSeg` skips the clear. The server's `ErrStale` covers correctness there.
- **The e2e only checks "survive pause", not resume** (`viewer_annotate_live_test.go:161-165`), although the Done-when says "survive pause/resume".
  - This is the 2nd finding in family `spec-path-untested`. The rule: every Done-when clause gets a named assertion in the e2e.
  - Sweep: resume is the only clause without one. Fix: put a `seg` record with `why: resume` after the pause, then assert both pins are still shown.
- **`identityStore` checks the stored id against `ID_RE` but never checks the stored name** (`annotate.js:50`), an ARCH-SECURE gap.
  - A hand-edited name over 40 runes would be refused by the board on every note. The name field is hidden once a name exists, so the viewer could never fix it.
  - Fix: apply the board's limit at read time (≤ 40 runes, no control characters), and otherwise ask again.

### 5. Test coverage notes
- The pure helpers (`identityStore`, `chunker`, `strokeStore`, `liveNotes`, `noteBox`, `regionOf`, `noteDraft`) are tested without IO, with real assertions.
- The shell is covered only by the Chrome e2e, which is skipped unless `WEBCAST_LIVE_CHROME=1` is set. Paths it doesn't touch:
  - the stroke-refusal message (BR-12);
  - ending a stroke at a new document (BR-14);
  - swatch picking;
  - resume.
- Exporting `answered(say)` as a pure helper would make BR-12 testable in node cheaply.

### 6. Architectural notes
- **ARCH-DRY: pass.** The shared JS lives in `collab.js`. The dialog and swatch CSS stay duplicated for a recorded reason (the two pages style different selectors).
- **ARCH-PURE: pass.** The helpers are pure. `mountAnnotate` is the shell, but `answered` and the gesture state sit inside it and so can't be tested in node.
- **ARCH-PURPOSE: pass for M2's scope.** The Couch side (`💭N`) is M3 by plan, not a hidden deferral.
- **ARCH-MOCK: pass.** The e2e uses the real viewer page, `Server`, `Board` and the reference `Writer` with `FakeBrowser`. `State.OnDoc` is shared with the recorder.
- **ARCH-CONSTRAINTS: flag (minor).** Each frame calls `boxOf` per note (at most 48 notes, which is fine) and rebuilds the swatches (see the Important finding). The cast-wide limit of 60 strokes/s means three viewers drawing at once get 429s; the user now sees a message for them.
- **ARCH-SECURE: pass, with one minor gap.** Note text is only set with `textContent`, colors are checked against `HEX_COLOR`, and `palette.json` is validated. The gap is the stored name.
- **ARCH-ORDER: flag (minor).** The shell's `{mode, drag, stroke, timer}` is a set of independent nullable fields; `drag` and a mode change can be in an illegal combination. A tagged gesture state (`idle | stroking | dragging`) would rule that out. The mount-after-catch-up ordering is now handled.
- **ARCH-FUNERAL: pass.**
  - Faded strokes are deleted in `visible()`, and pins are removed when their note is no longer live.
  - localStorage holds two new keys per browser.
  - Records on disk are bounded by the rate limits and removed with the recording.

### 7. Plan revision recommendations
- Architecture › Viewer › Pins says "redrawn each frame". Revise it to say pins are kept per note id and moved each frame, and that frame work never recreates an element that takes input (pins, swatches).

```findings
dispose:
  - id: BR-8
    disposition: addressed
    note: |
      visible() filters by each stroke's end against newest at draw time (annotate.js:113); node test replays in stream order; reverting the check turns annotate.test.mjs red (verified here).
  - id: BR-9
    disposition: addressed
    note: |
      Pins reconciled per id (annotate.js:385-396); .chip exempt at :291; e2e taps the chip by touch in note mode (viewer_annotate_live_test.go:168-174). The e2e could not be executed here (Chrome fails to start in this environment); the sibling swatch rebuild is raised separately.
  - id: BR-10
    disposition: addressed
    note: |
      e2e opens A's pin in B, presses Resolve, asserts the pin is gone in both viewers and a resolving note-state is in the recording (viewer_annotate_live_test.go:166-187).
  - id: BR-11
    disposition: addressed
    note: |
      README.md "Pointing and notes on a web cast (#470)" covers the rail modes, the name prompt, and that notes and strokes are saved in notes.jsonl/pointer.jsonl; it matches annotate.js and the recording records.
  - id: BR-12
    disposition: not-addressed
    note: |
      The fix is in the code (answered() and the toast, annotate.js:194-201,279), but no test fails without it: answered is internal and the e2e never triggers a refusal. Export a pure answered(say) and node-test a 409 and a rejected fetch.
  - id: BR-13
    disposition: addressed
    note: |
      poster and safeStorage now live in collab.js and both viewers use them; the CSS stays duplicated for a recorded reason (different selectors per page); existing tests cover the refactor.
  - id: BR-14
    disposition: not-addressed
    note: |
      s.seg is captured at stroke start and endStroke runs at a new document (annotate.js:299,446), but no test covers a stroke spanning a navigation; add an e2e step or a node seam for it.
findings:
  - id: new
    severity: Important
    family: pin-input-routing
    title: |
      drawRail rebuilds the swatch buttons every frame (renderSwatches at annotate.js:255), so swatch clicks are lost during activity
    detail: |
      2nd finding in family pin-input-routing. Rule: nothing that takes input is recreated on the per-frame path; frame work only moves or restyles, and elements are rebuilt only when their data changes. Sweep: pins fixed, swatches violate it, mode buttons and dialog fine. schedule() runs on every record, so the swatches are replaced many times a second during a live cast, and a click spanning a replace goes to the swatches div, which has no listener. Fix: render swatches only at mount and on pick; add an e2e step that picks a swatch while a stroke fades.
  - id: new
    severity: Minor
    family: stroke-spans-document
    title: |
      setMode and the cast-end handler end a stroke but leave a region drag in progress (annotate.js:266-272)
    detail: |
      2nd finding in family stroke-spans-document. Rule: one endGesture() ends the stroke and the drag, and every boundary (new document, mode change, cast end, pointercancel) calls it. Also, a resync that lands on another document skips the clear (was && docSeg); the server's ErrStale covers correctness.
  - id: new
    severity: Minor
    family: spec-path-untested
    title: |
      The e2e checks that notes survive a pause but never a resume, though Done-when says "survive pause/resume"
    detail: |
      2nd finding in family spec-path-untested. Rule: every Done-when clause gets a named assertion in the e2e. Resume is the only clause without one; put a seg record with why resume after the pause and assert both pins are still shown.
  - id: new
    severity: Minor
    family: persisted-input-parsed-at-read
    title: |
      identityStore checks the stored viewer id but not the stored name (annotate.js:50)
    detail: |
      A hand-edited name over 40 runes would be refused by the board on every note, and the name field is hidden once a name exists, so the viewer can't recover. Apply the board's name limit at read time and ask again otherwise (ARCH-SECURE).
```
