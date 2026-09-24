---
gate: boundary-review
issue: 317
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-24T13:02:38-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Important
          title: ParseSlotGitStatus accepts malformed porcelain fields
          detail: cmd/internal/couchcore/slotgit.go:70-84 accepts trailing branch.ab tokens and an empty branch.head despite documenting a closed grammar; reject malformed external output and add regression tests that fail without the fix. ARCH-SECURE.
          family: porcelain-grammar-validation
          round: 1
      recipe: milestone-review
      blocked: true
---

# Gate ledger — pair#317 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-24T13:02:38-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `porcelain-grammar-validation` ParseSlotGitStatus accepts malformed porcelain fields
  cmd/internal/couchcore/slotgit.go:70-84 accepts trailing branch.ab tokens and an empty branch.head despite documenting a closed grammar; reject malformed external output and add regression tests that fail without the fix. ARCH-SECURE.

## Open findings

- **BR-1** [Important] `porcelain-grammar-validation` ParseSlotGitStatus accepts malformed porcelain fields
