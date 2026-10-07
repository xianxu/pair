---
gate: boundary-review
issue: 405
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-07T14:06:17-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Important
          title: Capture tests do not verify the newly required highlight
          detail: cmd/internal/couchtty/capture_status_test.go:73 strips ANSI before assertions, leaving reserve.go:159's new behavior untested. Cover bold inverse styling and its reset for recording, draining, closed, queue/full/IO failure badges, plus disabled and clipped/zero-width output. Assert subsequent content is outside the highlight and confirm reverting the styling change makes the regression test fail.
          family: presentation-contract-coverage
          round: 1
      recipe: small-diff-review
      blocked: true
---

# Gate ledger — pair#405 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-07T14:06:17-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `presentation-contract-coverage` Capture tests do not verify the newly required highlight
  cmd/internal/couchtty/capture_status_test.go:73 strips ANSI before assertions, leaving reserve.go:159's new behavior untested. Cover bold inverse styling and its reset for recording, draining, closed, queue/full/IO failure badges, plus disabled and clipped/zero-width output. Assert subsequent content is outside the highlight and confirm reverting the styling change makes the regression test fail.

## Open findings

- **BR-1** [Important] `presentation-contract-coverage` Capture tests do not verify the newly required highlight
