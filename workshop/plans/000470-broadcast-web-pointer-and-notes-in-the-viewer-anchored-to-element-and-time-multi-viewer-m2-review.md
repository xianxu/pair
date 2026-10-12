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
