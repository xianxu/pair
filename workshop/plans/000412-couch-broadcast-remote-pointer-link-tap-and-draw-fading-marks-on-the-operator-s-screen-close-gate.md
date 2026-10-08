---
gate: boundary-review
issue: 412
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-08T09:41:11-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Tasks 1.3, 2.2, 2.3, 3.2 enumerate test cases in prose, restating Done-when
          detail: |-
            Compress to the one-line-per-risky-function strategy already in the Test strategy section.
            (carried from plan-quality PQ-2, deferred to the boundary review)
          family: enumerated-test-prose
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-10-08T09:41:11-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: Marks.Add cap eviction is O(excess x n), up to ~250-850ms per batch under the paint-path lock
          detail: 'marks.go:154-165 rescans the whole map for each evicted cell. Measured 850ms on a full 200x60 grid; a legal 64-point zigzag batch marks about 6400 cells, about 250ms. M3 holds marksMu, which the overlay takes on every paint, so this stalls the operator''s screen. Same-batch cells are also evicted by row/col, not stroke order. Fix: sort once by (time, sequence) and delete the excess; add a worst-case bound test.'
          family: untrusted-input-work-unbounded
          round: 2
        - id: BR-3
          severity: Important
          title: Marks.Overlay can tint the status row after a resize, failing IndicatorShown
          detail: The status-row drop is enforced only in Add, against the batch's rows. Overlay clips only to the frame size (marks.go:254), so marks surviving a shrink tint the new last row. That breaks IndicatorShown (the hub drops frames and the fail-safe can fire), and the tint, colour 214, equals PointerSGR. Skip row >= rows-1 in Overlay and test with a smaller frame.
          family: invariant-enforced-at-paint-time
          round: 2
        - id: BR-4
          severity: Minor
          title: Plan Core concepts lists Marks.Expired; the code has Live/NextChange and Add(points, cols, rows, now)
          family: plan-table-matches-code
          round: 2
        - id: BR-5
          severity: Minor
          title: marks_test.go ends with var _ = terminal.FramePrivate, which asserts nothing
          family: test-filler
          round: 2
        - id: BR-6
          severity: Minor
          title: No test covers the viewer loading the Unicode 11 add-on; a wrong global would break all viewing
          family: untested-page-wiring
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: true
---

# Gate ledger — pair#412 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-08T09:41:11-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `enumerated-test-prose` Tasks 1.3, 2.2, 2.3, 3.2 enumerate test cases in prose, restating Done-when
  Compress to the one-line-per-risky-function strategy already in the Test strategy section.
  (carried from plan-quality PQ-2, deferred to the boundary review)

## Round 2 — 2026-10-08T09:41:11-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `untrusted-input-work-unbounded` Marks.Add cap eviction is O(excess x n), up to ~250-850ms per batch under the paint-path lock
  marks.go:154-165 rescans the whole map for each evicted cell. Measured 850ms on a full 200x60 grid; a legal 64-point zigzag batch marks about 6400 cells, about 250ms. M3 holds marksMu, which the overlay takes on every paint, so this stalls the operator's screen. Same-batch cells are also evicted by row/col, not stroke order. Fix: sort once by (time, sequence) and delete the excess; add a worst-case bound test.
- **BR-3** [Important] `invariant-enforced-at-paint-time` Marks.Overlay can tint the status row after a resize, failing IndicatorShown
  The status-row drop is enforced only in Add, against the batch's rows. Overlay clips only to the frame size (marks.go:254), so marks surviving a shrink tint the new last row. That breaks IndicatorShown (the hub drops frames and the fail-safe can fire), and the tint, colour 214, equals PointerSGR. Skip row >= rows-1 in Overlay and test with a smaller frame.
- **BR-4** [Minor] `plan-table-matches-code` Plan Core concepts lists Marks.Expired; the code has Live/NextChange and Add(points, cols, rows, now)
- **BR-5** [Minor] `test-filler` marks_test.go ends with var _ = terminal.FramePrivate, which asserts nothing
- **BR-6** [Minor] `untested-page-wiring` No test covers the viewer loading the Unicode 11 add-on; a wrong global would break all viewing

## Open findings

- **BR-1** [Minor] `enumerated-test-prose` Tasks 1.3, 2.2, 2.3, 3.2 enumerate test cases in prose, restating Done-when
- **BR-2** [Important] `untrusted-input-work-unbounded` Marks.Add cap eviction is O(excess x n), up to ~250-850ms per batch under the paint-path lock
- **BR-3** [Important] `invariant-enforced-at-paint-time` Marks.Overlay can tint the status row after a resize, failing IndicatorShown
- **BR-4** [Minor] `plan-table-matches-code` Plan Core concepts lists Marks.Expired; the code has Live/NextChange and Add(points, cols, rows, now)
- **BR-5** [Minor] `test-filler` marks_test.go ends with var _ = terminal.FramePrivate, which asserts nothing
- **BR-6** [Minor] `untested-page-wiring` No test covers the viewer loading the Unicode 11 add-on; a wrong global would break all viewing
