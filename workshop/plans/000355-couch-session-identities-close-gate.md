---
gate: boundary-review
issue: 355
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-30T10:37:08-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Cold launch, registration, and cleanup do not consistently use the terminal incarnation
          detail: launch_existing.go:463 allocates a new M for an attached-live conversation, while createflow.go:427 filters out its existing terminal, permitting a duplicate agent. Registration and cleanup still resolve the old address index; require proven absence before creation and use the proposed binding for registration and teardown, with composed regressions (ARCH-ORDER, ARCH-PURPOSE).
          family: terminal-binding-authority
          round: 1
        - id: BR-2
          severity: Critical
          title: Warm attach revalidates ownership before blocking preparation rather than at handoff
          detail: createflow.go:397 checks the generation before lifecycle.go:43-110 performs retention and cmux preparation. Replacement during that interval reaches AttachSession unchecked; retain the original proof and revalidate immediately before attachment, testing replacement during preparation (ARCH-ORDER, ARCH-SECURE).
          family: owner-proof-at-effect-boundary
          round: 1
      boundary: M1
      recipe: milestone-review
      blocked: true
---

# Gate ledger — pair#355 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-30T10:37:08-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `terminal-binding-authority` Cold launch, registration, and cleanup do not consistently use the terminal incarnation
  launch_existing.go:463 allocates a new M for an attached-live conversation, while createflow.go:427 filters out its existing terminal, permitting a duplicate agent. Registration and cleanup still resolve the old address index; require proven absence before creation and use the proposed binding for registration and teardown, with composed regressions (ARCH-ORDER, ARCH-PURPOSE).
- **BR-2** [Critical] `owner-proof-at-effect-boundary` Warm attach revalidates ownership before blocking preparation rather than at handoff
  createflow.go:397 checks the generation before lifecycle.go:43-110 performs retention and cmux preparation. Replacement during that interval reaches AttachSession unchecked; retain the original proof and revalidate immediately before attachment, testing replacement during preparation (ARCH-ORDER, ARCH-SECURE).

## Open findings

- **BR-1** [Critical] `terminal-binding-authority` Cold launch, registration, and cleanup do not consistently use the terminal incarnation
- **BR-2** [Critical] `owner-proof-at-effect-boundary` Warm attach revalidates ownership before blocking preparation rather than at handoff
