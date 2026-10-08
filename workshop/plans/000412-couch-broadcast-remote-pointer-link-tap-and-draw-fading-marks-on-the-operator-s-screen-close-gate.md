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
    - "n": 3
      timestamp: "2026-10-08T09:43:51-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Task prose in the plan unchanged; still Minor, non-blocking.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Add now truncates batch to cap and evicts via one sort; restoring aba2176b's Add makes TestMarksAddWorstCaseIsCheap fail at 200ms/batch, fixed runs ~0.3ms.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Overlay skips row >= rows-1 (marks.go:174); mutating to rows makes TestMarksOverlayNeverTintsStatusRow fail; test also asserts IndicatorShown.
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: Plan Core concepts still lists Expired; no Revisions entry records Live/NextChange or Add's signature.
          round: 3
        - id: BR-5
          disposition: not-addressed
          note: var _ = terminal.FramePrivate still present in marks_test.go.
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: No test of viewer add-on wiring; M4 smoke remains the intended backstop.
          round: 3
      findings:
        - id: BR-7
          severity: Minor
          title: Marks.Add walks Bresenham over raw coordinates before the off-grid filter, so its cost bound holds only if M2's parser validates range
          detail: 'This is the 2nd finding in family untrusted-input-work-unbounded. Rule: work derived from untrusted input is bounded at the entity doing the work, not by an upstream validator. A point like [1e9,0] costs ~1e9 iterations under the paint lock despite the doc''s "cost is bounded". Fix at the rule: in Add, drop (or clip to the grid) any off-grid point before line(), and add the case to TestMarksAddWorstCaseIsCheap.'
          family: untrusted-input-work-unbounded
          round: 3
      boundary: M1
      recipe: milestone-review
      blocked: false
    - "n": 4
      timestamp: "2026-10-08T10:04:34-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan tasks 1.3/2.2/2.3/3.2 still enumerate test cases in prose; Minor, non-blocking.
          round: 4
      findings:
        - id: BR-8
          severity: Minor
          title: atlas/broadcast.md says the POST read deadline is 5s; code (pointReadBudget) is 2s
          detail: The security-hardening revision cut the budget to 2s but the atlas step 5 still says 5s; the new 2x-ping event-stream write deadline is also undocumented there.
          family: docs-match-code
          round: 4
        - id: BR-9
          severity: Minor
          title: 'PointerState has no stopped/generation state: EnablePointer after Stop mints a link, and a late pointerHidden can turn off a re-enabled pointing'
          detail: ARCH-ORDER. pointerHidden runs in a spawned goroutine and applies set(false) to whatever generation is current; EnablePointer after Stop succeeds against a shutting-down server. Practically unreachable today (needs M3 phase guard / sub-ms double click); fix by refusing set(true) once stopped and tagging the hub fire with an arm generation.
          family: stale-observation-acts-on-new-generation
          round: 4
      boundary: M2
      recipe: milestone-review
      blocked: false
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

## Round 3 — 2026-10-08T09:43:51-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Task prose in the plan unchanged; still Minor, non-blocking.
- BR-2 — addressed — Add now truncates batch to cap and evicts via one sort; restoring aba2176b's Add makes TestMarksAddWorstCaseIsCheap fail at 200ms/batch, fixed runs ~0.3ms.
- BR-3 — addressed — Overlay skips row >= rows-1 (marks.go:174); mutating to rows makes TestMarksOverlayNeverTintsStatusRow fail; test also asserts IndicatorShown.
- BR-4 — not-addressed — Plan Core concepts still lists Expired; no Revisions entry records Live/NextChange or Add's signature.
- BR-5 — not-addressed — var _ = terminal.FramePrivate still present in marks_test.go.
- BR-6 — not-addressed — No test of viewer add-on wiring; M4 smoke remains the intended backstop.

### Raised

- **BR-7** [Minor] `untrusted-input-work-unbounded` Marks.Add walks Bresenham over raw coordinates before the off-grid filter, so its cost bound holds only if M2's parser validates range
  This is the 2nd finding in family untrusted-input-work-unbounded. Rule: work derived from untrusted input is bounded at the entity doing the work, not by an upstream validator. A point like [1e9,0] costs ~1e9 iterations under the paint lock despite the doc's "cost is bounded". Fix at the rule: in Add, drop (or clip to the grid) any off-grid point before line(), and add the case to TestMarksAddWorstCaseIsCheap.

## Round 4 — 2026-10-08T10:04:34-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Plan tasks 1.3/2.2/2.3/3.2 still enumerate test cases in prose; Minor, non-blocking.

### Raised

- **BR-8** [Minor] `docs-match-code` atlas/broadcast.md says the POST read deadline is 5s; code (pointReadBudget) is 2s
  The security-hardening revision cut the budget to 2s but the atlas step 5 still says 5s; the new 2x-ping event-stream write deadline is also undocumented there.
- **BR-9** [Minor] `stale-observation-acts-on-new-generation` PointerState has no stopped/generation state: EnablePointer after Stop mints a link, and a late pointerHidden can turn off a re-enabled pointing
  ARCH-ORDER. pointerHidden runs in a spawned goroutine and applies set(false) to whatever generation is current; EnablePointer after Stop succeeds against a shutting-down server. Practically unreachable today (needs M3 phase guard / sub-ms double click); fix by refusing set(true) once stopped and tagging the hub fire with an arm generation.

## Open findings

- **BR-1** [Minor] `enumerated-test-prose` Tasks 1.3, 2.2, 2.3, 3.2 enumerate test cases in prose, restating Done-when
- **BR-4** [Minor] `plan-table-matches-code` Plan Core concepts lists Marks.Expired; the code has Live/NextChange and Add(points, cols, rows, now)
- **BR-5** [Minor] `test-filler` marks_test.go ends with var _ = terminal.FramePrivate, which asserts nothing
- **BR-6** [Minor] `untested-page-wiring` No test covers the viewer loading the Unicode 11 add-on; a wrong global would break all viewing
- **BR-7** [Minor] `untrusted-input-work-unbounded` Marks.Add walks Bresenham over raw coordinates before the off-grid filter, so its cost bound holds only if M2's parser validates range
- **BR-8** [Minor] `docs-match-code` atlas/broadcast.md says the POST read deadline is 5s; code (pointReadBudget) is 2s
- **BR-9** [Minor] `stale-observation-acts-on-new-generation` PointerState has no stopped/generation state: EnablePointer after Stop mints a link, and a late pointerHidden can turn off a re-enabled pointing
