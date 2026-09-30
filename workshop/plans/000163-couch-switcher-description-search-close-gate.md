---
gate: boundary-review
issue: 163
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-30T12:40:22-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Description matches bypass exact-reference precedence across row kinds.
          detail: 'cmd/internal/couchtty/menu.go:356-379 applies exact-over-fuzzy matching only to ordinary rows and independently appends slots. Enumerated affected cases: an ordinary exact tag plus a slot description containing that tag; a slot exact tag plus an ordinary description containing it; a slot exact tag plus another slot description containing it; and an explicit repo:N or :N selection plus an ordinary description containing that reference. Each admits an unrelated description match alongside the exact target, in both default and focus views when rows are live and described. Apply reference precedence across the complete inventory before fuzzy matching, and add both-view regressions for every case. The tests at menu_description_test.go:64 cover only ordinary/ordinary tag competition and slot/slot numeric references. ARCH-PURPOSE: the Spec and Done when explicitly promise preserved exact-reference precedence.'
          family: exact-reference-precedence-across-inventory
          round: 1
      recipe: small-diff-review
      blocked: true
---

# Gate ledger — pair#163 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-30T12:40:22-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `exact-reference-precedence-across-inventory` Description matches bypass exact-reference precedence across row kinds.
  cmd/internal/couchtty/menu.go:356-379 applies exact-over-fuzzy matching only to ordinary rows and independently appends slots. Enumerated affected cases: an ordinary exact tag plus a slot description containing that tag; a slot exact tag plus an ordinary description containing it; a slot exact tag plus another slot description containing it; and an explicit repo:N or :N selection plus an ordinary description containing that reference. Each admits an unrelated description match alongside the exact target, in both default and focus views when rows are live and described. Apply reference precedence across the complete inventory before fuzzy matching, and add both-view regressions for every case. The tests at menu_description_test.go:64 cover only ordinary/ordinary tag competition and slot/slot numeric references. ARCH-PURPOSE: the Spec and Done when explicitly promise preserved exact-reference precedence.

## Open findings

- **BR-1** [Critical] `exact-reference-precedence-across-inventory` Description matches bypass exact-reference precedence across row kinds.
