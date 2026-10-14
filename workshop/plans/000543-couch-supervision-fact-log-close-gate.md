---
gate: boundary-review
issue: 543
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-13T23:08:05-07:00"
      dispatched: "2026-10-13T23:05:32-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Inclusive horizon is declared complete before all qualifying writes are excluded.
          detail: 'scripts/supervision-labels/labels.py:186 accepts coverage ending exactly at delivery+600000 ms. Reproduced with delivery(seq=1, ms=0), snapshot(seq=2, ms=600000), snapshot(seq=4, ms=600001): the result is false despite missing seq=3. Restoring seq=3 as a same-subject write at ms=600000 changes it to true. Since timestamps can tie, require contiguous evidence strictly beyond the inclusive horizon before emitting false. Add regressions for both a truncated boundary tail and a boundary sequence gap. ARCH-ORDER, ARCH-PURPOSE.'
          family: negative-label-requires-complete-coverage
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#543 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T23:08:05-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `negative-label-requires-complete-coverage` Inclusive horizon is declared complete before all qualifying writes are excluded.
  scripts/supervision-labels/labels.py:186 accepts coverage ending exactly at delivery+600000 ms. Reproduced with delivery(seq=1, ms=0), snapshot(seq=2, ms=600000), snapshot(seq=4, ms=600001): the result is false despite missing seq=3. Restoring seq=3 as a same-subject write at ms=600000 changes it to true. Since timestamps can tie, require contiguous evidence strictly beyond the inclusive horizon before emitting false. Add regressions for both a truncated boundary tail and a boundary sequence gap. ARCH-ORDER, ARCH-PURPOSE.

## Open findings

- **BR-1** [Critical] `negative-label-requires-complete-coverage` Inclusive horizon is declared complete before all qualifying writes are excluded.
