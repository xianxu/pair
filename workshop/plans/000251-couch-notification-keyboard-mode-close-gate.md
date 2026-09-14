---
gate: boundary-review
issue: 251
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-14T09:46:45-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Important
          title: README still describes the previous Ctrl+Return protocol contract
          detail: README.md:404-408 attributes keyboard enablement to Zellij and says only CSI 13;5u is recognized, while Console now maintains disambiguation and keys.go accepts explicit press/repeat forms. Update the operator documentation with Couch ownership, supported events, and the unsupported-terminal fallback; atlas alone does not satisfy the README gate (ARCH-PURPOSE).
          family: public-behavior-doc-parity
          round: 1
      blocked: true
---

# Gate ledger — pair#251 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-14T09:46:45-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `public-behavior-doc-parity` README still describes the previous Ctrl+Return protocol contract
  README.md:404-408 attributes keyboard enablement to Zellij and says only CSI 13;5u is recognized, while Console now maintains disambiguation and keys.go accepts explicit press/repeat forms. Update the operator documentation with Couch ownership, supported events, and the unsupported-terminal fallback; atlas alone does not satisfy the README gate (ARCH-PURPOSE).

## Open findings

- **BR-1** [Important] `public-behavior-doc-parity` README still describes the previous Ctrl+Return protocol contract
