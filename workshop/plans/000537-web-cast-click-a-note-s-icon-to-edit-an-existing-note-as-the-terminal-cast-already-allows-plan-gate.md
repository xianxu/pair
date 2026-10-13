---
gate: plan-quality
issue: 537
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-13T16:17:44-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Replace enumerated test cases with named function-level strategies.
          detail: Tasks 1–4 contain prose test-case inventories that this gate explicitly rejects. Compress them into named unit-test surfaces with one adversarial-input and mechanical-guard strategy per risky function, including ownership projection, edit parsers, reducers, admission and dialog transitions; retain integration commands separately.
          family: function-level-test-strategy
          round: 1
        - id: PQ-2
          severity: Important
          title: Define the shared editor's authoritative transition model.
          detail: 'ARCH-ORDER and ARCH-PURE: the presentation table and generation-check promise do not specify a state/event model or structural enforcement for pending saves and uncertain reconciliation. Name the pure transition function and encapsulated state, define how retirement, capability loss, dismissal and late completion affect an outstanding operation, and specify controllable sequence tests against that production function.'
          family: explicit-event-state-ownership
          round: 1
        - id: PQ-3
          severity: Minor
          title: Document the retention and incremental cost of edit records.
          detail: 'ARCH-FUNERAL: NoteEdit appends replacement text to an existing durable recording, so absence of a new artifact family does not establish its lifecycle. Name the inherited recording removal policy and storage cost per accepted edit, with where that cost is measured; this need not introduce a separate cleanup mechanism.'
          family: durable-growth-lifecycle
          round: 1
      blocked: true
---

# Gate ledger — pair#537 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T16:17:44-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `function-level-test-strategy` Replace enumerated test cases with named function-level strategies.
  Tasks 1–4 contain prose test-case inventories that this gate explicitly rejects. Compress them into named unit-test surfaces with one adversarial-input and mechanical-guard strategy per risky function, including ownership projection, edit parsers, reducers, admission and dialog transitions; retain integration commands separately.
- **PQ-2** [Important] `explicit-event-state-ownership` Define the shared editor's authoritative transition model.
  ARCH-ORDER and ARCH-PURE: the presentation table and generation-check promise do not specify a state/event model or structural enforcement for pending saves and uncertain reconciliation. Name the pure transition function and encapsulated state, define how retirement, capability loss, dismissal and late completion affect an outstanding operation, and specify controllable sequence tests against that production function.
- **PQ-3** [Minor] `durable-growth-lifecycle` Document the retention and incremental cost of edit records.
  ARCH-FUNERAL: NoteEdit appends replacement text to an existing durable recording, so absence of a new artifact family does not establish its lifecycle. Name the inherited recording removal policy and storage cost per accepted edit, with where that cost is measured; this need not introduce a separate cleanup mechanism.

## Open findings

- **PQ-1** [Important] `function-level-test-strategy` Replace enumerated test cases with named function-level strategies.
- **PQ-2** [Important] `explicit-event-state-ownership` Define the shared editor's authoritative transition model.
- **PQ-3** [Minor] `durable-growth-lifecycle` Document the retention and incremental cost of edit records.
