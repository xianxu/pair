---
gate: boundary-review
issue: 851
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 5b3689f0-f2b3-42aa-88f4-31a6c95f0f1e
            repository: github.com/xianxu/pair
            issue_id: "000851"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/a384579be7e3defa
            reviewed_head: 4c251d1ab612e26805c9c387979838b8a8ee7f58
            slot: pair:2
            requested_at: "2026-10-24T17:41:42.430356Z"
          phase: finished
          pid: 60062
          started_at: "2026-10-24T17:41:42.563935Z"
          finished_at: "2026-10-24T17:43:45.773136Z"
          session_id: 01a16f01-11fd-7d02-b49e-5fb85bf5d0d6
          outcome: returned
      "n": 1
      timestamp: "2026-10-24T10:44:05-07:00"
      dispatched: "2026-10-24T10:41:40-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: README update is missing for the changed switcher click behavior.
          detail: README.md:1925-1928 still says any switcher-row click switches immediately and every click is a manual switch; README.md:1226 also promises an unqualified one-click warm-attachment retry. These are all affected README claims in this window. Update them to distinguish selection from activation, chips inside versus outside the switcher, and retrying an already-selected row. atlas/couch.md:3292 already documents the new policy, but README.md has no change in the pinned range.
          family: user-facing-behavior-documentation
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Done-when has not been reconciled with the revised click contract.
          detail: workshop/issues/000851-switcher-single-click-selects-a-click-on-the-selected-row-or-a-double-click.md:67-71 retains the double-click timing-window requirement, while Spec:38-44 explicitly rejects a timer and adds scoped chip behavior. The stale acceptance family comprises the timing-window clause and missing explicit acceptance for inside/outside chips, hidden-target reveal with selective filter/scope preservation, and nested-frame selection before activation. Revise Done-when to name those observable outcomes and rapid/delayed press tests.
          family: acceptance-contract-synchronization
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 0e1a5019-333b-444f-a40c-67d8a501a9cc
            repository: github.com/xianxu/pair
            issue_id: "000851"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/1cf5cd8101fa1ecc
            reviewed_head: 7a8a64ffd9de577d689963a02abab36e2aaabcfd
            slot: pair:2
            requested_at: "2026-10-24T17:51:57.736792Z"
          phase: finished
          pid: 66720
          started_at: "2026-10-24T17:51:57.905696Z"
          finished_at: "2026-10-24T17:54:36.627601Z"
          session_id: 01a16f0a-73be-7e70-9383-bd0720daa837
          outcome: returned
      "n": 2
      timestamp: "2026-10-24T10:55:23-07:00"
      dispatched: "2026-10-24T10:51:46-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: README.md:1226-1227 now qualifies warm retry by selection; README.md:1926-1937 distinguishes selection, activation, scoped chips, nested menus and manual activation. These prose corrections match terminal_input.go:278-289, menu_mouse.go:7-35 and the shared activation path; atlas/couch.md:194 also qualifies retry.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The issue's Done-when:69-72 replaces the timing window with rapid/delayed timer-free presses and explicitly covers scoped chips, hidden/nested reveal, restriction preservation, external chips and keyboard behavior. These match Spec:38-44; existing mouse and reducer tests substantiate the documented gestures.
          round: 2
      findings:
        - kind: defect
          id: BR-3
          severity: Critical
          title: Chip reveal evaluates the filter outside its retained group and can erase valid context.
          detail: 'cmd/internal/couchtty/menu_mouse.go:24-25 tests Filter using a frame without GroupScope. Filtering has exact-over-fuzzy precedence (menu.go:430-433), so results depend on the candidate group. Reproduced with a pair-group slot tagged foo-more, an outside brain-group slot tagged foo, Filter="foo", and Focus view hiding the summaryless target: the target matches the scoped filter, but clicking its chip clears both Focus view and Filter. Only Focus view should clear, per Done-when:70. A temporary overlay regression, TestReviewRevealPreservesScopedFuzzyFilter, failed with filter="" on pinned HEAD. ARCH-PURPOSE: preserve restrictions in their effective context. Evaluate the filter with the retained GroupScope after any necessary scope removal, then evaluate View. Family enumeration: the sole context-losing predicate is lines 24-25; scope membership at lines 21-22 is independent of filtering, and the final view check at lines 27-28 already uses the retained frame. Add regression coverage for this mixed-restriction case.'
          family: preserve-effective-reveal-context
          round: 2
      recipe: small-diff-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#851 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-24T10:44:05-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `user-facing-behavior-documentation` (kind: defect) README update is missing for the changed switcher click behavior.
  README.md:1925-1928 still says any switcher-row click switches immediately and every click is a manual switch; README.md:1226 also promises an unqualified one-click warm-attachment retry. These are all affected README claims in this window. Update them to distinguish selection from activation, chips inside versus outside the switcher, and retrying an already-selected row. atlas/couch.md:3292 already documents the new policy, but README.md has no change in the pinned range.
- **BR-2** [Important] `acceptance-contract-synchronization` (kind: defect) Done-when has not been reconciled with the revised click contract.
  workshop/issues/000851-switcher-single-click-selects-a-click-on-the-selected-row-or-a-double-click.md:67-71 retains the double-click timing-window requirement, while Spec:38-44 explicitly rejects a timer and adds scoped chip behavior. The stale acceptance family comprises the timing-window clause and missing explicit acceptance for inside/outside chips, hidden-target reveal with selective filter/scope preservation, and nested-frame selection before activation. Revise Done-when to name those observable outcomes and rapid/delayed press tests.

## Round 2 — 2026-10-24T10:55:23-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — README.md:1226-1227 now qualifies warm retry by selection; README.md:1926-1937 distinguishes selection, activation, scoped chips, nested menus and manual activation. These prose corrections match terminal_input.go:278-289, menu_mouse.go:7-35 and the shared activation path; atlas/couch.md:194 also qualifies retry.
- BR-2 — addressed — The issue's Done-when:69-72 replaces the timing window with rapid/delayed timer-free presses and explicitly covers scoped chips, hidden/nested reveal, restriction preservation, external chips and keyboard behavior. These match Spec:38-44; existing mouse and reducer tests substantiate the documented gestures.

### Raised

- **BR-3** [Critical] `preserve-effective-reveal-context` (kind: defect) Chip reveal evaluates the filter outside its retained group and can erase valid context.
  cmd/internal/couchtty/menu_mouse.go:24-25 tests Filter using a frame without GroupScope. Filtering has exact-over-fuzzy precedence (menu.go:430-433), so results depend on the candidate group. Reproduced with a pair-group slot tagged foo-more, an outside brain-group slot tagged foo, Filter="foo", and Focus view hiding the summaryless target: the target matches the scoped filter, but clicking its chip clears both Focus view and Filter. Only Focus view should clear, per Done-when:70. A temporary overlay regression, TestReviewRevealPreservesScopedFuzzyFilter, failed with filter="" on pinned HEAD. ARCH-PURPOSE: preserve restrictions in their effective context. Evaluate the filter with the retained GroupScope after any necessary scope removal, then evaluate View. Family enumeration: the sole context-losing predicate is lines 24-25; scope membership at lines 21-22 is independent of filtering, and the final view check at lines 27-28 already uses the retained frame. Add regression coverage for this mixed-restriction case.

## Open findings

- **BR-3** [Critical] `preserve-effective-reveal-context` (kind: defect) Chip reveal evaluates the filter outside its retained group and can erase valid context.
