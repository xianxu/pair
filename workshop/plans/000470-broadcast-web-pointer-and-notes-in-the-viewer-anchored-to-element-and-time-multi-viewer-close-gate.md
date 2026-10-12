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

## Open findings

- **BR-3** [Minor] `sink-order-before-consumer-check` The Tee feeds the Hub before the board, so a viewer can act on a seg the board has not applied yet and get a spurious 409
- **BR-4** [Minor] `untrusted-geometry-bound` A viewer-supplied note box is bounded only to 1e6 px and widens the page box in crops
- **BR-7** [Minor] `fake-conformance` Nothing checks that board.Writer (synchronous, returns ErrStale) and recorder.Emit (asynchronous, drops silently) behave the same
- **BR-8** [Important] `filter-on-partial-stream` strokeStore drops catch-up strokes only when a newer record arrived first; a late joiner redraws every stroke of the document
- **BR-9** [Important] `pin-input-routing` Pin chips are rebuilt every frame and swallowed by the layer's pointerdown in point/note mode
- **BR-10** [Important] `spec-path-untested` No test opens a pin or resolves a note from the viewer (Spec: resolved notes hide)
- **BR-11** [Important] `readme-surface-gap` README not updated for web cast pointing and notes (rail modes, name prompt, notes and strokes recorded to disk)
- **BR-12** [Minor] `silent-degradation` Refused or failed stroke chunk POSTs are swallowed with no viewer signal
- **BR-13** [Minor] `shared-helper-duplication` postJSON, the storage try/catch wrapper and the sticky-note/swatch CSS are restated across both viewers
- **BR-14** [Minor] `stroke-spans-document` A stroke in progress across a navigation posts its later chunks with the new seg
