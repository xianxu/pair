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
    - "n": 5
      timestamp: "2026-10-08T10:13:47-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan Task 3.2 still enumerates console test cases in prose; the plan file is unchanged in this window.
          round: 5
      findings:
        - id: BR-10
          severity: Important
          title: "README.md doesn't document the new status-row controls (left-click \U0001F446 toggles pointing and copies the link, right-click re-copies, \U0001F47D inert)"
          detail: README.md:808-819 describes only LIVE ⏸. M3 adds mouse controls an operator clicks, and no plan task (including 4.2) updates the README. Add a short paragraph now, or add an explicit M4 README step via a plan Revisions entry.
          family: readme-covers-new-surface
          round: 5
        - id: BR-11
          severity: Minor
          title: The Console's OnPoints and OnPointerOff aren't tied to the session that triggered them, so a late callback from an old broadcast can act on the next one
          detail: 'This is the 2nd finding in this family; BR-9 fixed the session-layer instance. The rule: every deferred report (watch fire, point batch, timer) carries the generation it observed, and its consumer compares that with the current generation before acting. Sweep: startBroadcast binds cfg.OnPoints/OnPointerOff to the Console unconditionally (console_broadcast.go:117-122). Bind a closure capturing the attempt and check c.bcast.attempt in applyPoints and pointerOffByWatch, as broadcastEnded already does with s. Reaching it needs a request goroutine descheduled across an operator stop, start and enable; the impact is a transient mark or a fail-safe ''off''.'
          family: stale-observation-acts-on-new-generation
          round: 5
        - id: BR-12
          severity: Minor
          title: endBroadcastForShutdown and a failed SetTap in broadcastStarted bypass detachBroadcastScreen, leaving the fade timer armed and the overlay installed
          detail: ARCH-FUNERAL. The plan's lifetimes section says the fade timer stops at Console teardown, but only stopBroadcast and broadcastEnded call resetPointer and SetOverlay(nil). Every path that ends or abandons a broadcast should go through detachBroadcastScreen. It's harmless today (no marks means the overlay does nothing, and the timer's command fails once stopped), but the contract isn't met as written.
          family: teardown-path-skips-detach
          round: 5
      boundary: M3
      recipe: milestone-review
      blocked: true
    - "n": 6
      timestamp: "2026-10-08T10:14:43-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan Task 3.2 still enumerates test cases in prose; the plan file is unchanged in this window.
          round: 6
        - id: BR-10
          disposition: addressed
          note: "README.md:840-851 (ed2e412e) documents the three click behaviours of \U0001F446 and \U0001F47D and the end-of-broadcast rule; the claims match markOverlay, Marks.Add/Overlay and the existing pointer tests."
          round: 6
        - id: BR-11
          disposition: not-addressed
          note: startBroadcast still binds c.onPoints and c.pointerOffByWatch unconditionally (console_broadcast.go:116-122); no attempt check in applyPoints or pointerOffByWatch.
          round: 6
        - id: BR-12
          disposition: not-addressed
          note: endBroadcastForShutdown and the failed-SetTap branch of broadcastStarted still skip detachBroadcastScreen; unchanged in this window.
          round: 6
      boundary: M3
      recipe: milestone-review
      blocked: false
    - "n": 7
      timestamp: "2026-10-08T12:18:29-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Task 4.1 still enumerates test cases in prose; Minor, never blocks.
          round: 7
      findings:
        - id: BR-13
          severity: Important
          title: Issue Spec/Done-when still say "3 steps over 3s" and "status row dropped"; atlas fade sentence self-contradicts
          detail: '2nd in family. Rule: a smoke-driven contract change revises Spec, Done-when, plan Revisions and atlas in the same commit; sweep "3 steps", "3s", "status row" across all four. Plan Revisions omits the fade change; atlas/broadcast.md says "fade in 3 steps ... 10 steps".'
          family: docs-match-code
          round: 7
        - id: BR-14
          severity: Important
          title: Core concepts table lists batchPoints and Marks.Expired; code ships chunkStroke and Live/NextChange
          detail: '2nd in family. Rule: at each boundary grep every table row''s symbol and record renames in a Revisions entry.'
          family: plan-table-matches-code
          round: 7
        - id: BR-15
          severity: Important
          title: pointerMode (setOn/hint/.pointer class/flush on pointerup) untested despite Task 4.1's promised caps-toggle test
          detail: '2nd in family. Rule: page logic beyond a DOM call sits behind a seam node can drive (inject post plus fake stage/hint), and every promised page test exercises that seam. The node test only checks that the caps callback fires.'
          family: untested-page-wiring
          round: 7
        - id: BR-16
          severity: Minor
          title: Console's SetBlend(palette...) wiring has no test; a mutation to false stays green
          family: untested-blend-wiring
          round: 7
      boundary: M4
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

## Round 5 — 2026-10-08T10:13:47-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Plan Task 3.2 still enumerates console test cases in prose; the plan file is unchanged in this window.

### Raised

- **BR-10** [Important] `readme-covers-new-surface` README.md doesn't document the new status-row controls (left-click 👆 toggles pointing and copies the link, right-click re-copies, 👽 inert)
  README.md:808-819 describes only LIVE ⏸. M3 adds mouse controls an operator clicks, and no plan task (including 4.2) updates the README. Add a short paragraph now, or add an explicit M4 README step via a plan Revisions entry.
- **BR-11** [Minor] `stale-observation-acts-on-new-generation` The Console's OnPoints and OnPointerOff aren't tied to the session that triggered them, so a late callback from an old broadcast can act on the next one
  This is the 2nd finding in this family; BR-9 fixed the session-layer instance. The rule: every deferred report (watch fire, point batch, timer) carries the generation it observed, and its consumer compares that with the current generation before acting. Sweep: startBroadcast binds cfg.OnPoints/OnPointerOff to the Console unconditionally (console_broadcast.go:117-122). Bind a closure capturing the attempt and check c.bcast.attempt in applyPoints and pointerOffByWatch, as broadcastEnded already does with s. Reaching it needs a request goroutine descheduled across an operator stop, start and enable; the impact is a transient mark or a fail-safe 'off'.
- **BR-12** [Minor] `teardown-path-skips-detach` endBroadcastForShutdown and a failed SetTap in broadcastStarted bypass detachBroadcastScreen, leaving the fade timer armed and the overlay installed
  ARCH-FUNERAL. The plan's lifetimes section says the fade timer stops at Console teardown, but only stopBroadcast and broadcastEnded call resetPointer and SetOverlay(nil). Every path that ends or abandons a broadcast should go through detachBroadcastScreen. It's harmless today (no marks means the overlay does nothing, and the timer's command fails once stopped), but the contract isn't met as written.

## Round 6 — 2026-10-08T10:14:43-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Plan Task 3.2 still enumerates test cases in prose; the plan file is unchanged in this window.
- BR-10 — addressed — README.md:840-851 (ed2e412e) documents the three click behaviours of 👆 and 👽 and the end-of-broadcast rule; the claims match markOverlay, Marks.Add/Overlay and the existing pointer tests.
- BR-11 — not-addressed — startBroadcast still binds c.onPoints and c.pointerOffByWatch unconditionally (console_broadcast.go:116-122); no attempt check in applyPoints or pointerOffByWatch.
- BR-12 — not-addressed — endBroadcastForShutdown and the failed-SetTap branch of broadcastStarted still skip detachBroadcastScreen; unchanged in this window.

## Round 7 — 2026-10-08T12:18:29-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Task 4.1 still enumerates test cases in prose; Minor, never blocks.

### Raised

- **BR-13** [Important] `docs-match-code` Issue Spec/Done-when still say "3 steps over 3s" and "status row dropped"; atlas fade sentence self-contradicts
  2nd in family. Rule: a smoke-driven contract change revises Spec, Done-when, plan Revisions and atlas in the same commit; sweep "3 steps", "3s", "status row" across all four. Plan Revisions omits the fade change; atlas/broadcast.md says "fade in 3 steps ... 10 steps".
- **BR-14** [Important] `plan-table-matches-code` Core concepts table lists batchPoints and Marks.Expired; code ships chunkStroke and Live/NextChange
  2nd in family. Rule: at each boundary grep every table row's symbol and record renames in a Revisions entry.
- **BR-15** [Important] `untested-page-wiring` pointerMode (setOn/hint/.pointer class/flush on pointerup) untested despite Task 4.1's promised caps-toggle test
  2nd in family. Rule: page logic beyond a DOM call sits behind a seam node can drive (inject post plus fake stage/hint), and every promised page test exercises that seam. The node test only checks that the caps callback fires.
- **BR-16** [Minor] `untested-blend-wiring` Console's SetBlend(palette...) wiring has no test; a mutation to false stays green

## Open findings

- **BR-1** [Minor] `enumerated-test-prose` Tasks 1.3, 2.2, 2.3, 3.2 enumerate test cases in prose, restating Done-when
- **BR-4** [Minor] `plan-table-matches-code` Plan Core concepts lists Marks.Expired; the code has Live/NextChange and Add(points, cols, rows, now)
- **BR-5** [Minor] `test-filler` marks_test.go ends with var _ = terminal.FramePrivate, which asserts nothing
- **BR-6** [Minor] `untested-page-wiring` No test covers the viewer loading the Unicode 11 add-on; a wrong global would break all viewing
- **BR-7** [Minor] `untrusted-input-work-unbounded` Marks.Add walks Bresenham over raw coordinates before the off-grid filter, so its cost bound holds only if M2's parser validates range
- **BR-8** [Minor] `docs-match-code` atlas/broadcast.md says the POST read deadline is 5s; code (pointReadBudget) is 2s
- **BR-9** [Minor] `stale-observation-acts-on-new-generation` PointerState has no stopped/generation state: EnablePointer after Stop mints a link, and a late pointerHidden can turn off a re-enabled pointing
- **BR-11** [Minor] `stale-observation-acts-on-new-generation` The Console's OnPoints and OnPointerOff aren't tied to the session that triggered them, so a late callback from an old broadcast can act on the next one
- **BR-12** [Minor] `teardown-path-skips-detach` endBroadcastForShutdown and a failed SetTap in broadcastStarted bypass detachBroadcastScreen, leaving the fade timer armed and the overlay installed
- **BR-13** [Important] `docs-match-code` Issue Spec/Done-when still say "3 steps over 3s" and "status row dropped"; atlas fade sentence self-contradicts
- **BR-14** [Important] `plan-table-matches-code` Core concepts table lists batchPoints and Marks.Expired; code ships chunkStroke and Live/NextChange
- **BR-15** [Important] `untested-page-wiring` pointerMode (setOn/hint/.pointer class/flush on pointerup) untested despite Task 4.1's promised caps-toggle test
- **BR-16** [Minor] `untested-blend-wiring` Console's SetBlend(palette...) wiring has no test; a mutation to false stays green
