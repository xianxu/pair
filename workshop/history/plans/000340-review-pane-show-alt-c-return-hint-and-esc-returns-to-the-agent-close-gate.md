---
gate: boundary-review
issue: 340
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-28T18:54:22-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Important
          title: Agent-selection regression does not assert the selected destination
          detail: 'tests/review-controls-test.sh:26 accepts writes to any pane despite placing an unrelated terminal before the agent; tests/review-poke-test.sh:18 checks destinations but places the agent first. ARCH-MOCK: add the reordered terminal to the destination-checking fixture, assert both body and submit target pane 7, and verify the old title-based selector makes the test fail.'
          family: regression-oracle-covers-corrected-behavior
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-09-28T18:57:38-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Both reordered-pane fixtures assert body delivery and submit target agent pane 7. Both pass on HEAD and fail when the old title-based selector is restored in a temporary copy.
          round: 2
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#340 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-28T18:54:22-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `regression-oracle-covers-corrected-behavior` Agent-selection regression does not assert the selected destination
  tests/review-controls-test.sh:26 accepts writes to any pane despite placing an unrelated terminal before the agent; tests/review-poke-test.sh:18 checks destinations but places the agent first. ARCH-MOCK: add the reordered terminal to the destination-checking fixture, assert both body and submit target pane 7, and verify the old title-based selector makes the test fail.

## Round 2 — 2026-09-28T18:57:38-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Both reordered-pane fixtures assert body delivery and submit target agent pane 7. Both pass on HEAD and fail when the old title-based selector is restored in a temporary copy.

## Open findings

(none — every finding has been disposed)
