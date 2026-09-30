---
gate: boundary-review
issue: 211
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-29T21:40:09-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Agent-agnostic contract now requires bracketed paste (DECSET 2004); atlas and harness bring-up guide do not say so
          detail: atlas/architecture.md:1255 still claims any TUI agent that accepts typed input works; a harness without 2004 would get raw ESC[200~/ESC[201~ around every draft send and poke. State the requirement in the bring-up guide (optionally strip the markers in pair-wrap when the child has not enabled 2004).
          family: docs-track-new-contract
          round: 1
        - id: BR-2
          severity: Minor
          title: orientation.go:146 hardcodes paste markers instead of workbenchshortcut.PasteStart/PasteEnd
          family: single-source-constant
          round: 1
        - id: BR-3
          severity: Minor
          title: send-audit.py restates the === comment strip rule from nvim/normalization.lua; drift risk
          family: single-source-constant
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-09-29T21:55:38-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: childAcceptsPaste strips markers for non-2004 children in translateChunk and passThroughChunk; paste_capability_test.go expects typed output the verbatim path cannot produce; atlas architecture.md and the bring-up guide state the contract.
          round: 2
        - id: BR-2
          disposition: addressed
          note: orientation.go:147 now uses workbenchshortcut.PasteStart/PasteEnd.
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: scripts/send-audit.py:94 still hand-restates the === strip regex from nvim/normalization.lua:11; Minor, non-blocking.
          round: 2
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#211 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-29T21:40:09-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `docs-track-new-contract` Agent-agnostic contract now requires bracketed paste (DECSET 2004); atlas and harness bring-up guide do not say so
  atlas/architecture.md:1255 still claims any TUI agent that accepts typed input works; a harness without 2004 would get raw ESC[200~/ESC[201~ around every draft send and poke. State the requirement in the bring-up guide (optionally strip the markers in pair-wrap when the child has not enabled 2004).
- **BR-2** [Minor] `single-source-constant` orientation.go:146 hardcodes paste markers instead of workbenchshortcut.PasteStart/PasteEnd
- **BR-3** [Minor] `single-source-constant` send-audit.py restates the === comment strip rule from nvim/normalization.lua; drift risk

## Round 2 — 2026-09-29T21:55:38-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — childAcceptsPaste strips markers for non-2004 children in translateChunk and passThroughChunk; paste_capability_test.go expects typed output the verbatim path cannot produce; atlas architecture.md and the bring-up guide state the contract.
- BR-2 — addressed — orientation.go:147 now uses workbenchshortcut.PasteStart/PasteEnd.
- BR-3 — not-addressed — scripts/send-audit.py:94 still hand-restates the === strip regex from nvim/normalization.lua:11; Minor, non-blocking.

## Open findings

- **BR-3** [Minor] `single-source-constant` send-audit.py restates the === comment strip rule from nvim/normalization.lua; drift risk
