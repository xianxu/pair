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
    - "n": 2
      timestamp: "2026-10-13T09:37:18-07:00"
      dispatched: "2026-10-13T09:33:34-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Policy envelopes and nested rule evidence are fixed; removing the exact-key guard makes their regression tests fail. However, cmd/internal/submission/store.go:219 still decodes top-level permission fields case-insensitively. Appending Privacy:"shareable" and Privacy_rule containing a matching shareable rule to an explicitly private manifest makes readMetadata return a validated shareable bundle. Complete the ambiguous-permission-decoding family sweep across all permission-bearing JSON boundaries, including persisted manifest fields, and add a store-reload regression (ARCH-SECURE, ARCH-PURPOSE).
          round: 2
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

## Round 2 — 2026-10-13T09:37:18-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — not-addressed — Policy envelopes and nested rule evidence are fixed; removing the exact-key guard makes their regression tests fail. However, cmd/internal/submission/store.go:219 still decodes top-level permission fields case-insensitively. Appending Privacy:"shareable" and Privacy_rule containing a matching shareable rule to an explicitly private manifest makes readMetadata return a validated shareable bundle. Complete the ambiguous-permission-decoding family sweep across all permission-bearing JSON boundaries, including persisted manifest fields, and add a store-reload regression (ARCH-SECURE, ARCH-PURPOSE).

## Open findings

- **BR-1** [Critical] `ambiguous-permission-decoding` Case-variant JSON keys can override private policy and grant sharing
