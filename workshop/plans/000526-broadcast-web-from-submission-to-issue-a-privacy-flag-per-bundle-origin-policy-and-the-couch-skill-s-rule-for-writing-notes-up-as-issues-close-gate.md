---
gate: boundary-review
issue: 526
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-13T09:23:51-07:00"
      dispatched: "2026-10-13T09:21:41-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Case-variant JSON keys can override private policy and grant sharing
          detail: cmd/internal/submission/privacy.go:237 checks duplicate keys case-sensitively, but the struct decoder at line 257 matches fields case-insensitively. Reproduced conflicting privacy/Privacy and origins/Origins inputs that return shareable instead of invalid-policy private fallback (ARCH-SECURE). Enforce exact schema keys or reject decoder-equivalent aliases across policy and rule evidence, and add regression coverage for rejection, private fallback, and the notice.
          family: ambiguous-permission-decoding
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#526 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T09:23:51-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `ambiguous-permission-decoding` Case-variant JSON keys can override private policy and grant sharing
  cmd/internal/submission/privacy.go:237 checks duplicate keys case-sensitively, but the struct decoder at line 257 matches fields case-insensitively. Reproduced conflicting privacy/Privacy and origins/Origins inputs that return shareable instead of invalid-policy private fallback (ARCH-SECURE). Enforce exact schema keys or reject decoder-equivalent aliases across policy and rule evidence, and add regression coverage for rejection, private fallback, and the notice.

## Open findings

- **BR-1** [Critical] `ambiguous-permission-decoding` Case-variant JSON keys can override private policy and grant sharing
