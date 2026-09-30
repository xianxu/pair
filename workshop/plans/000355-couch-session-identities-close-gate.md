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
    - "n": 2
      timestamp: "2026-09-30T10:59:00-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Cold admission requires absence; registration and cleanup select the pending/current terminal binding. Passing terminal_incarnation_test.go regressions cover attached-terminal refusal, stale registration, wrong-terminal cleanup, and interrupted recovery; launcher tests cover late terminal appearance.
          round: 2
        - id: BR-2
          disposition: addressed
          note: lifecycle.go:134-145 revalidates the original ownership proof immediately before AttachSession, after blocking preparation. TestCouchSessionWarmGenerationRevalidatedAtAttachEffect injects replacement during retention/cmux through both entrypoints and asserts refusal and poller cleanup.
          round: 2
      findings:
        - id: BR-3
          severity: Critical
          title: Unicode-only repository names now prevent conversation creation
          detail: cmd/internal/couchidentity/identity.go:41-60 discards every non-ASCII character and rejects the resulting empty token. Both couch.go:488 and slotrecovery.go:320 pass the repository basename, so supported names such as 项目 now fail allocation. Use a deterministic safe fallback when normalization yields nothing; C/N already provide uniqueness. Add pure formatter tests and composed new/fresh launch regressions for Unicode-only and punctuation-only names. ARCH-PURPOSE.
          family: descriptive-label-must-not-gate-identity
          round: 2
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

## Round 2 — 2026-09-30T10:59:00-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Cold admission requires absence; registration and cleanup select the pending/current terminal binding. Passing terminal_incarnation_test.go regressions cover attached-terminal refusal, stale registration, wrong-terminal cleanup, and interrupted recovery; launcher tests cover late terminal appearance.
- BR-2 — addressed — lifecycle.go:134-145 revalidates the original ownership proof immediately before AttachSession, after blocking preparation. TestCouchSessionWarmGenerationRevalidatedAtAttachEffect injects replacement during retention/cmux through both entrypoints and asserts refusal and poller cleanup.

### Raised

- **BR-3** [Critical] `descriptive-label-must-not-gate-identity` Unicode-only repository names now prevent conversation creation
  cmd/internal/couchidentity/identity.go:41-60 discards every non-ASCII character and rejects the resulting empty token. Both couch.go:488 and slotrecovery.go:320 pass the repository basename, so supported names such as 项目 now fail allocation. Use a deterministic safe fallback when normalization yields nothing; C/N already provide uniqueness. Add pure formatter tests and composed new/fresh launch regressions for Unicode-only and punctuation-only names. ARCH-PURPOSE.

## Open findings

- **BR-3** [Critical] `descriptive-label-must-not-gate-identity` Unicode-only repository names now prevent conversation creation
