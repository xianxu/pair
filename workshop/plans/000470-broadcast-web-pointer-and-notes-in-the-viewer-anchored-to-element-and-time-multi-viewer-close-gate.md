---
gate: boundary-review
issue: 470
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T03:22:17-07:00"
      dispatched: "2026-10-12T03:17:37-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Plan Architecture still describes CastConfig.Board, couchcmd construction and board.OnDoc; the code wires in StartCast/WithBoard and uses recording.State.OnDoc
          detail: The issue Log records the change but the durable plan does not; add a Revisions entry covering the wiring, the OnDoc location, Cast.Notes/ResolveNote, the composed stroke id and the final refusal sentences.
          family: plan-drift-after-rewire
          round: 1
        - id: BR-2
          severity: Minor
          title: A broken board answers ErrStale forever with no log, and Put returns ErrClosed after Close despite the never-fails-the-Tee claim
          detail: board.go Put/pageLocked. Give it a distinct error or a log hook so the operator can see the board has stopped (ARCH-SECURE).
          family: silent-degradation
          round: 1
        - id: BR-3
          severity: Minor
          title: The Tee feeds the Hub before the board, so a viewer can act on a seg the board has not applied yet and get a spurious 409
          detail: 'cast.go recordingOutput: append bd before live (ARCH-ORDER).'
          family: sink-order-before-consumer-check
          round: 1
        - id: BR-4
          severity: Minor
          title: A viewer-supplied note box is bounded only to 1e6 px and widens the page box in crops
          detail: Clamp it to the element's layout box or the page before it reaches NoteCrops in wave 3 (ARCH-SECURE).
          family: untrusted-geometry-bound
          round: 1
        - id: BR-5
          severity: Minor
          title: The board keeps a second full recording.State, including every stroke for the cast's life, beside the recorder's mirror
          detail: Bounded only by the stroke rate; the board needs neither old strokes nor older documents (ARCH-CONSTRAINTS/ARCH-FUNERAL).
          family: duplicate-reducer-state-growth
          round: 1
        - id: BR-6
          severity: Minor
          title: The atlas stroke row does not say the record id is the viewer id plus "." plus the stroke id, which M2's JS store joins on
          family: atlas-wire-detail
          round: 1
        - id: BR-7
          severity: Minor
          title: Nothing checks that board.Writer (synchronous, returns ErrStale) and recorder.Emit (asynchronous, drops silently) behave the same
          detail: 'ARCH-MOCK: a shared contract run over both writers would catch drift.'
          family: fake-conformance
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T03:26:36-07:00"
      dispatched: "2026-10-12T03:23:33-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Plan Revisions 2026-10-12 covers the StartCast/WithBoard wiring, State.OnDoc, Cast.Notes/ResolveNote, the stroke id, the answers and budgets; all checked against cast.go, chrome.go, recorder.go and server_board.go.
          round: 2
        - id: BR-2
          disposition: addressed
          note: ErrBroken (503, logged once through Logf, wired from cfg.Logf); Put returns nil after Close; pinned by TestBrokenBoardStopsAndSaysSo.
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: The order is fixed in recordingOutput, but swapping bd and live back leaves go test ./cmd/internal/webcast green; add a test that pins dir, board, hub.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: Deferred to wave 3 with a reason; NoteCrops has no production caller yet, so it is still open but not due at M1.
          round: 2
        - id: BR-5
          disposition: addressed
          note: Board.Put truncates st.Strokes after each Apply (board.go:126); pinned by board_test.go:431.
          round: 2
        - id: BR-6
          disposition: addressed
          note: The atlas stroke row now says the id is the viewer id, a dot, then the shell's stroke id.
          round: 2
        - id: BR-7
          disposition: not-addressed
          note: 'Declined with a reason; the gap remains: tests on board.Writer see 409 for a navigation race where production answers 204 and drops silently.'
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: checkout
      reviewed: b9e409c6697e10ab4f30010b3ea0be541955f2d6
      blocked: false
    - "n": 3
      timestamp: "2026-10-12T03:46:27-07:00"
      dispatched: "2026-10-12T03:42:42-07:00"
      agent: claude
      findings:
        - id: BR-8
          severity: Important
          title: strokeStore drops catch-up strokes only when a newer record arrived first; a late joiner redraws every stroke of the document
          detail: 'annotate.js:94-99 compares each stroke with the running newest time at arrival; CatchUp replays the document''s strokes oldest-first, so each passes. Reproduced: catch-up strokes at t=1000 and t=31000 then a record at t=61000 leave both drawn. The node test seeds seen(60000) before the old stroke, an order a real catch-up never has. Fix: keep each stroke''s end time and filter in visible() at draw time; test in stream order.'
          family: filter-on-partial-stream
          round: 3
        - id: BR-9
          severity: Important
          title: Pin chips are rebuilt every frame and swallowed by the layer's pointerdown in point/note mode
          detail: 'drawPins replaceChildren()s every frame while strokes are visible, so a tap spanning a rebuild loses its click. In point/note mode a chip pointerdown bubbles to the layer handler, which captures the pointer and starts a stroke or region draft instead of opening the note. Fix: ignore pointerdown targets inside .chip; rebuild pins only when the live set changes.'
          family: pin-input-routing
          round: 3
        - id: BR-10
          severity: Important
          title: 'No test opens a pin or resolves a note from the viewer (Spec: resolved notes hide)'
          detail: The e2e never taps a chip or presses Resolve; the viewer's note-state request and the pin disappearing in both viewers are unverified. Extend TestAnnotateLiveTwoViewers to cover this.
          family: spec-path-untested
          round: 3
        - id: BR-11
          severity: Important
          title: README not updated for web cast pointing and notes (rail modes, name prompt, notes and strokes recorded to disk)
          detail: 'README''s #447 paragraph says terminal notes are never saved to disk; web cast notes are recording records. Add the viewer surface, or state in the plan that M3 adds it.'
          family: readme-surface-gap
          round: 3
        - id: BR-12
          severity: Minor
          title: Refused or failed stroke chunk POSTs are swallowed with no viewer signal
          detail: '2nd finding in family silent-degradation. The rule: every viewer POST answered with something other than 204 produces at least one visible signal. The note dialog follows it; annotate.js:271 (.catch(()=>{}), non-204 ignored) is the remaining exception. Prevalence in this diff: 1 of 3 POST paths.'
          family: silent-degradation
          round: 3
        - id: BR-13
          severity: Minor
          title: postJSON, the storage try/catch wrapper and the sticky-note/swatch CSS are restated across both viewers
          detail: annotate.js:157 duplicates broadcast/web/viewer.js:352; identityStore repeats colorStore's guards; annotate.css restates viewer.css's dialog and swatch styles (ARCH-DRY).
          family: shared-helper-duplication
          round: 3
        - id: BR-14
          severity: Minor
          title: A stroke in progress across a navigation posts its later chunks with the new seg
          detail: Call endStroke() when onRecord sees docSeg change.
          family: stroke-spans-document
          round: 3
      boundary: M2
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 4
      timestamp: "2026-10-12T04:02:36-07:00"
      dispatched: "2026-10-12T03:58:47-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: addressed
          note: visible() filters by each stroke's end against newest at draw time (annotate.js:113); node test replays in stream order; reverting the check turns annotate.test.mjs red (verified here).
          round: 4
        - id: BR-9
          disposition: addressed
          note: Pins reconciled per id (annotate.js:385-396); .chip exempt at :291; e2e taps the chip by touch in note mode (viewer_annotate_live_test.go:168-174). The e2e could not be executed here (Chrome fails to start in this environment); the sibling swatch rebuild is raised separately.
          round: 4
        - id: BR-10
          disposition: addressed
          note: e2e opens A's pin in B, presses Resolve, asserts the pin is gone in both viewers and a resolving note-state is in the recording (viewer_annotate_live_test.go:166-187).
          round: 4
        - id: BR-11
          disposition: addressed
          note: README.md "Pointing and notes on a web cast (#470)" covers the rail modes, the name prompt, and that notes and strokes are saved in notes.jsonl/pointer.jsonl; it matches annotate.js and the recording records.
          round: 4
        - id: BR-12
          disposition: not-addressed
          note: 'The fix is in the code (answered() and the toast, annotate.js:194-201,279), but no test fails without it: answered is internal and the e2e never triggers a refusal. Export a pure answered(say) and node-test a 409 and a rejected fetch.'
          round: 4
        - id: BR-13
          disposition: addressed
          note: poster and safeStorage now live in collab.js and both viewers use them; the CSS stays duplicated for a recorded reason (different selectors per page); existing tests cover the refactor.
          round: 4
        - id: BR-14
          disposition: not-addressed
          note: s.seg is captured at stroke start and endStroke runs at a new document (annotate.js:299,446), but no test covers a stroke spanning a navigation; add an e2e step or a node seam for it.
          round: 4
      findings:
        - id: BR-15
          severity: Important
          title: drawRail rebuilds the swatch buttons every frame (renderSwatches at annotate.js:255), so swatch clicks are lost during activity
          detail: '2nd finding in family pin-input-routing. Rule: nothing that takes input is recreated on the per-frame path; frame work only moves or restyles, and elements are rebuilt only when their data changes. Sweep: pins fixed, swatches violate it, mode buttons and dialog fine. schedule() runs on every record, so the swatches are replaced many times a second during a live cast, and a click spanning a replace goes to the swatches div, which has no listener. Fix: render swatches only at mount and on pick; add an e2e step that picks a swatch while a stroke fades.'
          family: pin-input-routing
          round: 4
        - id: BR-16
          severity: Minor
          title: setMode and the cast-end handler end a stroke but leave a region drag in progress (annotate.js:266-272)
          detail: '2nd finding in family stroke-spans-document. Rule: one endGesture() ends the stroke and the drag, and every boundary (new document, mode change, cast end, pointercancel) calls it. Also, a resync that lands on another document skips the clear (was && docSeg); the server''s ErrStale covers correctness.'
          family: stroke-spans-document
          round: 4
        - id: BR-17
          severity: Minor
          title: The e2e checks that notes survive a pause but never a resume, though Done-when says "survive pause/resume"
          detail: '2nd finding in family spec-path-untested. Rule: every Done-when clause gets a named assertion in the e2e. Resume is the only clause without one; put a seg record with why resume after the pause and assert both pins are still shown.'
          family: spec-path-untested
          round: 4
        - id: BR-18
          severity: Minor
          title: identityStore checks the stored viewer id but not the stored name (annotate.js:50)
          detail: A hand-edited name over 40 runes would be refused by the board on every note, and the name field is hidden once a name exists, so the viewer can't recover. Apply the board's name limit at read time and ask again otherwise (ARCH-SECURE).
          family: persisted-input-parsed-at-read
          round: 4
      boundary: M2
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 5
      timestamp: "2026-10-12T04:15:45-07:00"
      dispatched: "2026-10-12T04:12:45-07:00"
      agent: claude
      dispose:
        - id: BR-12
          disposition: not-addressed
          note: 'Unchanged since round 2: answered() and the toast exist (annotate.js:292-299,383) but nothing tests them. Rule for BR-12/14/16 together: move the shell''s decisions into pure exports, answered(say) and a gesture(state, event) reducer, and node-test a 409, a rejected fetch, a stroke across a new document and a drag across a mode change.'
          round: 5
        - id: BR-14
          disposition: not-addressed
          note: The code is right (onRecord calls endGesture on a docSeg change, annotate.js:504-511, and a trailing chunk is answered ErrStale) but no test covers a stroke spanning a navigation; this gets the gesture-reducer test noted on BR-12.
          round: 5
        - id: BR-15
          disposition: addressed
          note: renderRail runs only on a pick or mode change and placeRail per frame (annotate.js:355-368,495); the e2e slowClick step at viewer_annotate_live_test.go:102-105 would lose the click on old code. Chrome could not launch here, so this was judged from the code, not a revert run.
          round: 5
        - id: BR-16
          disposition: not-addressed
          note: endGesture covers mode change, cast end, pointercancel and a new document, and the resync guard is fixed (annotate.js:395-398,445,504-513), but no test drives a drag across a mode change or the cast's end.
          round: 5
        - id: BR-17
          disposition: addressed
          note: The e2e puts a seg with why resume plus the top-document snapshot and asserts the pin is still shown in both viewers (viewer_annotate_live_test.go, after the pause step).
          round: 5
        - id: BR-18
          disposition: addressed
          note: validName is applied at read (annotate.js:53) and node-tested for 41 runes; a separate gap in its character class is raised as a new finding.
          round: 5
      findings:
        - id: BR-19
          severity: Minor
          title: validName rejects fewer characters than the board's nameText, and setName stores a name without checking it
          detail: '2nd finding in this family. Rule: a client copy of a board check uses the same predicate, here /[\p{Cc}\p{Cf}]/u plus the rune count, on every path that writes the value (read and setName). Checked in node: U+2067, U+061C, U+180E, U+FFF9 and U+E0041 pass validName (annotate.js:66) and are refused by board/wire.go:203-208. A typed bidi-isolate name is then stored, refused on every note, and its field is hidden. Add U+2067 to the node test.'
          family: persisted-input-parsed-at-read
          round: 5
        - id: BR-20
          severity: Minor
          title: strokesAt has no production caller and restates replay.js drawMarks's stroke-at-t and note-box logic with different semantics
          detail: '2nd finding in this family. Rule: one implementation for each drawing-at-t concept. strokesAt (annotate.js:167) joins chunks by id and fades over FADE_MS. drawMarks (replay.js:458-470) works per chunk with STROKE_FADE. noteBox puts the region first, drawMarks puts it last. The atlas wrongly says the live viewer draws with strokesAt. Either have drawMarks call noteBox/strokesAt now, or correct the atlas and record the convergence in #477.'
          family: shared-helper-duplication
          round: 5
      boundary: M2
      recipe: milestone-review
      isolation: checkout
      reviewed: 98d5a7a19abf7d2e5a71756108716e64c8061187
      blocked: false
---

# Gate ledger — pair#470 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T03:22:17-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `plan-drift-after-rewire` Plan Architecture still describes CastConfig.Board, couchcmd construction and board.OnDoc; the code wires in StartCast/WithBoard and uses recording.State.OnDoc
  The issue Log records the change but the durable plan does not; add a Revisions entry covering the wiring, the OnDoc location, Cast.Notes/ResolveNote, the composed stroke id and the final refusal sentences.
- **BR-2** [Minor] `silent-degradation` A broken board answers ErrStale forever with no log, and Put returns ErrClosed after Close despite the never-fails-the-Tee claim
  board.go Put/pageLocked. Give it a distinct error or a log hook so the operator can see the board has stopped (ARCH-SECURE).
- **BR-3** [Minor] `sink-order-before-consumer-check` The Tee feeds the Hub before the board, so a viewer can act on a seg the board has not applied yet and get a spurious 409
  cast.go recordingOutput: append bd before live (ARCH-ORDER).
- **BR-4** [Minor] `untrusted-geometry-bound` A viewer-supplied note box is bounded only to 1e6 px and widens the page box in crops
  Clamp it to the element's layout box or the page before it reaches NoteCrops in wave 3 (ARCH-SECURE).
- **BR-5** [Minor] `duplicate-reducer-state-growth` The board keeps a second full recording.State, including every stroke for the cast's life, beside the recorder's mirror
  Bounded only by the stroke rate; the board needs neither old strokes nor older documents (ARCH-CONSTRAINTS/ARCH-FUNERAL).
- **BR-6** [Minor] `atlas-wire-detail` The atlas stroke row does not say the record id is the viewer id plus "." plus the stroke id, which M2's JS store joins on
- **BR-7** [Minor] `fake-conformance` Nothing checks that board.Writer (synchronous, returns ErrStale) and recorder.Emit (asynchronous, drops silently) behave the same
  ARCH-MOCK: a shared contract run over both writers would catch drift.

## Round 2 — 2026-10-12T03:26:36-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Plan Revisions 2026-10-12 covers the StartCast/WithBoard wiring, State.OnDoc, Cast.Notes/ResolveNote, the stroke id, the answers and budgets; all checked against cast.go, chrome.go, recorder.go and server_board.go.
- BR-2 — addressed — ErrBroken (503, logged once through Logf, wired from cfg.Logf); Put returns nil after Close; pinned by TestBrokenBoardStopsAndSaysSo.
- BR-3 — not-addressed — The order is fixed in recordingOutput, but swapping bd and live back leaves go test ./cmd/internal/webcast green; add a test that pins dir, board, hub.
- BR-4 — not-addressed — Deferred to wave 3 with a reason; NoteCrops has no production caller yet, so it is still open but not due at M1.
- BR-5 — addressed — Board.Put truncates st.Strokes after each Apply (board.go:126); pinned by board_test.go:431.
- BR-6 — addressed — The atlas stroke row now says the id is the viewer id, a dot, then the shell's stroke id.
- BR-7 — not-addressed — Declined with a reason; the gap remains: tests on board.Writer see 409 for a navigation race where production answers 204 and drops silently.

## Round 3 — 2026-10-12T03:46:27-07:00 (claude) — BLOCKED

### Raised

- **BR-8** [Important] `filter-on-partial-stream` strokeStore drops catch-up strokes only when a newer record arrived first; a late joiner redraws every stroke of the document
  annotate.js:94-99 compares each stroke with the running newest time at arrival; CatchUp replays the document's strokes oldest-first, so each passes. Reproduced: catch-up strokes at t=1000 and t=31000 then a record at t=61000 leave both drawn. The node test seeds seen(60000) before the old stroke, an order a real catch-up never has. Fix: keep each stroke's end time and filter in visible() at draw time; test in stream order.
- **BR-9** [Important] `pin-input-routing` Pin chips are rebuilt every frame and swallowed by the layer's pointerdown in point/note mode
  drawPins replaceChildren()s every frame while strokes are visible, so a tap spanning a rebuild loses its click. In point/note mode a chip pointerdown bubbles to the layer handler, which captures the pointer and starts a stroke or region draft instead of opening the note. Fix: ignore pointerdown targets inside .chip; rebuild pins only when the live set changes.
- **BR-10** [Important] `spec-path-untested` No test opens a pin or resolves a note from the viewer (Spec: resolved notes hide)
  The e2e never taps a chip or presses Resolve; the viewer's note-state request and the pin disappearing in both viewers are unverified. Extend TestAnnotateLiveTwoViewers to cover this.
- **BR-11** [Important] `readme-surface-gap` README not updated for web cast pointing and notes (rail modes, name prompt, notes and strokes recorded to disk)
  README's #447 paragraph says terminal notes are never saved to disk; web cast notes are recording records. Add the viewer surface, or state in the plan that M3 adds it.
- **BR-12** [Minor] `silent-degradation` Refused or failed stroke chunk POSTs are swallowed with no viewer signal
  2nd finding in family silent-degradation. The rule: every viewer POST answered with something other than 204 produces at least one visible signal. The note dialog follows it; annotate.js:271 (.catch(()=>{}), non-204 ignored) is the remaining exception. Prevalence in this diff: 1 of 3 POST paths.
- **BR-13** [Minor] `shared-helper-duplication` postJSON, the storage try/catch wrapper and the sticky-note/swatch CSS are restated across both viewers
  annotate.js:157 duplicates broadcast/web/viewer.js:352; identityStore repeats colorStore's guards; annotate.css restates viewer.css's dialog and swatch styles (ARCH-DRY).
- **BR-14** [Minor] `stroke-spans-document` A stroke in progress across a navigation posts its later chunks with the new seg
  Call endStroke() when onRecord sees docSeg change.

## Round 4 — 2026-10-12T04:02:36-07:00 (claude) — BLOCKED

### Disposed

- BR-8 — addressed — visible() filters by each stroke's end against newest at draw time (annotate.js:113); node test replays in stream order; reverting the check turns annotate.test.mjs red (verified here).
- BR-9 — addressed — Pins reconciled per id (annotate.js:385-396); .chip exempt at :291; e2e taps the chip by touch in note mode (viewer_annotate_live_test.go:168-174). The e2e could not be executed here (Chrome fails to start in this environment); the sibling swatch rebuild is raised separately.
- BR-10 — addressed — e2e opens A's pin in B, presses Resolve, asserts the pin is gone in both viewers and a resolving note-state is in the recording (viewer_annotate_live_test.go:166-187).
- BR-11 — addressed — README.md "Pointing and notes on a web cast (#470)" covers the rail modes, the name prompt, and that notes and strokes are saved in notes.jsonl/pointer.jsonl; it matches annotate.js and the recording records.
- BR-12 — not-addressed — The fix is in the code (answered() and the toast, annotate.js:194-201,279), but no test fails without it: answered is internal and the e2e never triggers a refusal. Export a pure answered(say) and node-test a 409 and a rejected fetch.
- BR-13 — addressed — poster and safeStorage now live in collab.js and both viewers use them; the CSS stays duplicated for a recorded reason (different selectors per page); existing tests cover the refactor.
- BR-14 — not-addressed — s.seg is captured at stroke start and endStroke runs at a new document (annotate.js:299,446), but no test covers a stroke spanning a navigation; add an e2e step or a node seam for it.

### Raised

- **BR-15** [Important] `pin-input-routing` drawRail rebuilds the swatch buttons every frame (renderSwatches at annotate.js:255), so swatch clicks are lost during activity
  2nd finding in family pin-input-routing. Rule: nothing that takes input is recreated on the per-frame path; frame work only moves or restyles, and elements are rebuilt only when their data changes. Sweep: pins fixed, swatches violate it, mode buttons and dialog fine. schedule() runs on every record, so the swatches are replaced many times a second during a live cast, and a click spanning a replace goes to the swatches div, which has no listener. Fix: render swatches only at mount and on pick; add an e2e step that picks a swatch while a stroke fades.
- **BR-16** [Minor] `stroke-spans-document` setMode and the cast-end handler end a stroke but leave a region drag in progress (annotate.js:266-272)
  2nd finding in family stroke-spans-document. Rule: one endGesture() ends the stroke and the drag, and every boundary (new document, mode change, cast end, pointercancel) calls it. Also, a resync that lands on another document skips the clear (was && docSeg); the server's ErrStale covers correctness.
- **BR-17** [Minor] `spec-path-untested` The e2e checks that notes survive a pause but never a resume, though Done-when says "survive pause/resume"
  2nd finding in family spec-path-untested. Rule: every Done-when clause gets a named assertion in the e2e. Resume is the only clause without one; put a seg record with why resume after the pause and assert both pins are still shown.
- **BR-18** [Minor] `persisted-input-parsed-at-read` identityStore checks the stored viewer id but not the stored name (annotate.js:50)
  A hand-edited name over 40 runes would be refused by the board on every note, and the name field is hidden once a name exists, so the viewer can't recover. Apply the board's name limit at read time and ask again otherwise (ARCH-SECURE).

## Round 5 — 2026-10-12T04:15:45-07:00 (claude) — passed

### Disposed

- BR-12 — not-addressed — Unchanged since round 2: answered() and the toast exist (annotate.js:292-299,383) but nothing tests them. Rule for BR-12/14/16 together: move the shell's decisions into pure exports, answered(say) and a gesture(state, event) reducer, and node-test a 409, a rejected fetch, a stroke across a new document and a drag across a mode change.
- BR-14 — not-addressed — The code is right (onRecord calls endGesture on a docSeg change, annotate.js:504-511, and a trailing chunk is answered ErrStale) but no test covers a stroke spanning a navigation; this gets the gesture-reducer test noted on BR-12.
- BR-15 — addressed — renderRail runs only on a pick or mode change and placeRail per frame (annotate.js:355-368,495); the e2e slowClick step at viewer_annotate_live_test.go:102-105 would lose the click on old code. Chrome could not launch here, so this was judged from the code, not a revert run.
- BR-16 — not-addressed — endGesture covers mode change, cast end, pointercancel and a new document, and the resync guard is fixed (annotate.js:395-398,445,504-513), but no test drives a drag across a mode change or the cast's end.
- BR-17 — addressed — The e2e puts a seg with why resume plus the top-document snapshot and asserts the pin is still shown in both viewers (viewer_annotate_live_test.go, after the pause step).
- BR-18 — addressed — validName is applied at read (annotate.js:53) and node-tested for 41 runes; a separate gap in its character class is raised as a new finding.

### Raised

- **BR-19** [Minor] `persisted-input-parsed-at-read` validName rejects fewer characters than the board's nameText, and setName stores a name without checking it
  2nd finding in this family. Rule: a client copy of a board check uses the same predicate, here /[\p{Cc}\p{Cf}]/u plus the rune count, on every path that writes the value (read and setName). Checked in node: U+2067, U+061C, U+180E, U+FFF9 and U+E0041 pass validName (annotate.js:66) and are refused by board/wire.go:203-208. A typed bidi-isolate name is then stored, refused on every note, and its field is hidden. Add U+2067 to the node test.
- **BR-20** [Minor] `shared-helper-duplication` strokesAt has no production caller and restates replay.js drawMarks's stroke-at-t and note-box logic with different semantics
  2nd finding in this family. Rule: one implementation for each drawing-at-t concept. strokesAt (annotate.js:167) joins chunks by id and fades over FADE_MS. drawMarks (replay.js:458-470) works per chunk with STROKE_FADE. noteBox puts the region first, drawMarks puts it last. The atlas wrongly says the live viewer draws with strokesAt. Either have drawMarks call noteBox/strokesAt now, or correct the atlas and record the convergence in #477.

## Open findings

- **BR-3** [Minor] `sink-order-before-consumer-check` The Tee feeds the Hub before the board, so a viewer can act on a seg the board has not applied yet and get a spurious 409
- **BR-4** [Minor] `untrusted-geometry-bound` A viewer-supplied note box is bounded only to 1e6 px and widens the page box in crops
- **BR-7** [Minor] `fake-conformance` Nothing checks that board.Writer (synchronous, returns ErrStale) and recorder.Emit (asynchronous, drops silently) behave the same
- **BR-12** [Minor] `silent-degradation` Refused or failed stroke chunk POSTs are swallowed with no viewer signal
- **BR-14** [Minor] `stroke-spans-document` A stroke in progress across a navigation posts its later chunks with the new seg
- **BR-16** [Minor] `stroke-spans-document` setMode and the cast-end handler end a stroke but leave a region drag in progress (annotate.js:266-272)
- **BR-19** [Minor] `persisted-input-parsed-at-read` validName rejects fewer characters than the board's nameText, and setName stores a name without checking it
- **BR-20** [Minor] `shared-helper-duplication` strokesAt has no production caller and restates replay.js drawMarks's stroke-at-t and note-box logic with different semantics
