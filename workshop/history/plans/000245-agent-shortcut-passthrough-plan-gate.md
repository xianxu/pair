---
gate: plan-quality
issue: 245
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-14T20:20:00-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Resolve interception scope after routing the candidate chord's prefix.
          detail: 'Task 3 reads scope before FeedHit, but console.go:751 identifies the hit before route(before), which can change focus through onMenuInput at console.go:734. Under ARCH-ORDER, separate framing from authorization: deliver preceding bytes, resolve current scope, then consume or forward the raw candidate; verify this through deterministic focus-changing input sequences.'
          family: routing-authority-after-prefix-effects
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace prose test inventories with named function-level strategies.
          detail: Tasks 1–5 enumerate test cases and procedural implementation steps, while the new framing API's production functions remain unnamed. Compress these into named unit-test targets with one adversarial input class and mechanical guard per risky function, retaining integration boundaries and verification commands; use generated stream partitions and invariant checks for the stateful scanner.
          family: function-level-test-strategy
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-14T20:22:39-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: The authoritative revision separates candidate framing from authorization after prefix delivery and names deterministic tests for both focus directions.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: The authoritative revision replaces prose inventories with named production-function strategies, generated stream partitions, and mechanical invariants while retaining integration and verification obligations.
          round: 2
      blocked: false
content_hash: b724c078c639e093979785c8095113605374db53795a43cd411a27d0d864178d
---

# Gate ledger — pair#245 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-14T20:20:00-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `routing-authority-after-prefix-effects` Resolve interception scope after routing the candidate chord's prefix.
  Task 3 reads scope before FeedHit, but console.go:751 identifies the hit before route(before), which can change focus through onMenuInput at console.go:734. Under ARCH-ORDER, separate framing from authorization: deliver preceding bytes, resolve current scope, then consume or forward the raw candidate; verify this through deterministic focus-changing input sequences.
- **PQ-2** [Important] `function-level-test-strategy` Replace prose test inventories with named function-level strategies.
  Tasks 1–5 enumerate test cases and procedural implementation steps, while the new framing API's production functions remain unnamed. Compress these into named unit-test targets with one adversarial input class and mechanical guard per risky function, retaining integration boundaries and verification commands; use generated stream partitions and invariant checks for the stateful scanner.

## Round 2 — 2026-09-14T20:22:39-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — The authoritative revision separates candidate framing from authorization after prefix delivery and names deterministic tests for both focus directions.
- PQ-2 — addressed — The authoritative revision replaces prose inventories with named production-function strategies, generated stream partitions, and mechanical invariants while retaining integration and verification obligations.

## Open findings

(none — every finding has been disposed)
