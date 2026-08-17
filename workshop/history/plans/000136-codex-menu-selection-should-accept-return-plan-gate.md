---
gate: plan-quality
issue: 136
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-16T21:02:26-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Test plan does not name the unit-tested function or risky-input strategy
          detail: The plan says “Add a failing Codex overlay detector test” but this gate requires the functions that will be unit-tested by name plus one strategy line per risky function. Name `detectCodexOverlayOpen` or `detectCodexOverlayText` and the strategy, e.g. stripped/ANSI-wrapped visible Codex footer text must match the exact marker while preserving the existing one-shot `emitPlainCR` behavior.
          round: 1
        - id: PQ-2
          severity: Important
          title: Plan has no stated non-goals for the overlay detector change
          detail: The issue is a narrow marker drift fix, but the plan never says what it is deliberately not changing. Add non-goals such as no generic `promptShape` broadening, no change to the `pickerActive` one-shot contract, and no new external/live harness dependency; that keeps ARCH-PURPOSE scoped to the current Codex footer rather than a broader overlay redesign.
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-16T21:03:33-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: The plan names `detectCodexOverlayOpen` and gives the ANSI/stripped visible-footer strategy while preserving existing `emitPlainCR` one-shot coverage.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: The Spec now states non-goals for no promptShape broadening, no pickerActive lifecycle change, and no live Codex dependency.
          round: 2
      blocked: false
content_hash: a8bf7f16ef942ab68a91bad25b36f07ff9ad21f1918243d5617c1ca448976fd4
---

# Gate ledger — pair#136 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-16T21:02:26-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] Test plan does not name the unit-tested function or risky-input strategy
  The plan says “Add a failing Codex overlay detector test” but this gate requires the functions that will be unit-tested by name plus one strategy line per risky function. Name `detectCodexOverlayOpen` or `detectCodexOverlayText` and the strategy, e.g. stripped/ANSI-wrapped visible Codex footer text must match the exact marker while preserving the existing one-shot `emitPlainCR` behavior.
- **PQ-2** [Important] Plan has no stated non-goals for the overlay detector change
  The issue is a narrow marker drift fix, but the plan never says what it is deliberately not changing. Add non-goals such as no generic `promptShape` broadening, no change to the `pickerActive` one-shot contract, and no new external/live harness dependency; that keeps ARCH-PURPOSE scoped to the current Codex footer rather than a broader overlay redesign.

## Round 2 — 2026-08-16T21:03:33-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — The plan names `detectCodexOverlayOpen` and gives the ANSI/stripped visible-footer strategy while preserving existing `emitPlainCR` one-shot coverage.
- PQ-2 — addressed — The Spec now states non-goals for no promptShape broadening, no pickerActive lifecycle change, and no live Codex dependency.

## Open findings

(none — every finding has been disposed)
