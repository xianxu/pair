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

## Open findings

- **BR-1** [Important] `plan-drift-after-rewire` Plan Architecture still describes CastConfig.Board, couchcmd construction and board.OnDoc; the code wires in StartCast/WithBoard and uses recording.State.OnDoc
- **BR-2** [Minor] `silent-degradation` A broken board answers ErrStale forever with no log, and Put returns ErrClosed after Close despite the never-fails-the-Tee claim
- **BR-3** [Minor] `sink-order-before-consumer-check` The Tee feeds the Hub before the board, so a viewer can act on a seg the board has not applied yet and get a spurious 409
- **BR-4** [Minor] `untrusted-geometry-bound` A viewer-supplied note box is bounded only to 1e6 px and widens the page box in crops
- **BR-5** [Minor] `duplicate-reducer-state-growth` The board keeps a second full recording.State, including every stroke for the cast's life, beside the recorder's mirror
- **BR-6** [Minor] `atlas-wire-detail` The atlas stroke row does not say the record id is the viewer id plus "." plus the stroke id, which M2's JS store joins on
- **BR-7** [Minor] `fake-conformance` Nothing checks that board.Writer (synchronous, returns ErrStale) and recorder.Emit (asynchronous, drops silently) behave the same
