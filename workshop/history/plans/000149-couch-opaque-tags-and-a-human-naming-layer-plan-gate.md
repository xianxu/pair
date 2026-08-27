---
gate: plan-quality
issue: 149
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-26T10:17:20-07:00"
      agent: codex
      blocked: true
      protocol_error: no valid findings block
    - "n": 2
      timestamp: "2026-08-26T10:20:39-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Critical
          title: M1 admits same-path concurrency before distinct durable thread tags exist
          detail: M1 retains the existing path-derived tag while activating provider admission and removing the same-tree bypass. Current Spawn launches pair resume with DefaultTag(tree), so multiple Brain starts admitted at one path can attach to one Pair session and share artifacts until M2 introduces opaque tags. Move opaque allocation into M1 or keep same-path admission fail-closed until identity lands (ARCH-PURPOSE).
          family: milestone-admission-before-thread-identity
          round: 2
        - id: PQ-2
          severity: Important
          title: Compress enumerated test cases and procedural diff instructions into named function strategies
          detail: The plan contains extensive prose case inventories and diff restatements, while risky functions such as strict policy decoding, journal recovery, and migration transitions are not consistently named. Replace these with each unit-tested function by name and one adversarial-input/mechanical-guard strategy line.
          family: test-plan-is-executable-code-preimage
          round: 2
        - id: PQ-3
          severity: Important
          title: The external policy seam has no recurring live conformance cadence
          detail: The plan adds a stateful fake and an opt-in one-time check against the local Ariadne binary, but does not identify the post-merge trigger, cadence, or owner that will detect provider protocol drift while ordinary CI remains hermetic (ARCH-MOCK).
          family: live-conformance-cadence
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-08-26T10:25:41-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: M1 now atomically claims the final composite opaque address before admission or fork and launches Pair with that exact address.
          round: 3
        - id: PQ-2
          disposition: not-addressed
          note: The named strategy table was added, but extensive enumerated test scenarios and procedural diff restatements remain throughout the task steps.
          round: 3
        - id: PQ-3
          disposition: addressed
          note: Pair now owns weekly and manual live conformance, plus M1-boundary and resolver-wire-change triggers.
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-08-26T10:29:16-07:00"
      agent: codex
      dispose:
        - id: PQ-2
          disposition: addressed
          note: The named risky-function strategy table now owns one adversarial-input and mechanical-guard line per function, while task steps reference it without repeating prose test-case inventories.
          round: 4
      blocked: false
content_hash: e9cf4880ea2841bfa12b236f4075f7ba4f8ec61f3ebc74a4a71d954e6393f3c0
---

# Gate ledger — pair#149 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-26T10:17:20-07:00 (codex) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 2 — 2026-08-26T10:20:39-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Critical] `milestone-admission-before-thread-identity` M1 admits same-path concurrency before distinct durable thread tags exist
  M1 retains the existing path-derived tag while activating provider admission and removing the same-tree bypass. Current Spawn launches pair resume with DefaultTag(tree), so multiple Brain starts admitted at one path can attach to one Pair session and share artifacts until M2 introduces opaque tags. Move opaque allocation into M1 or keep same-path admission fail-closed until identity lands (ARCH-PURPOSE).
- **PQ-2** [Important] `test-plan-is-executable-code-preimage` Compress enumerated test cases and procedural diff instructions into named function strategies
  The plan contains extensive prose case inventories and diff restatements, while risky functions such as strict policy decoding, journal recovery, and migration transitions are not consistently named. Replace these with each unit-tested function by name and one adversarial-input/mechanical-guard strategy line.
- **PQ-3** [Important] `live-conformance-cadence` The external policy seam has no recurring live conformance cadence
  The plan adds a stateful fake and an opt-in one-time check against the local Ariadne binary, but does not identify the post-merge trigger, cadence, or owner that will detect provider protocol drift while ordinary CI remains hermetic (ARCH-MOCK).

## Round 3 — 2026-08-26T10:25:41-07:00 (codex) — BLOCKED

### Disposed

- PQ-1 — addressed — M1 now atomically claims the final composite opaque address before admission or fork and launches Pair with that exact address.
- PQ-2 — not-addressed — The named strategy table was added, but extensive enumerated test scenarios and procedural diff restatements remain throughout the task steps.
- PQ-3 — addressed — Pair now owns weekly and manual live conformance, plus M1-boundary and resolver-wire-change triggers.

## Round 4 — 2026-08-26T10:29:16-07:00 (codex) — passed

### Disposed

- PQ-2 — addressed — The named risky-function strategy table now owns one adversarial-input and mechanical-guard line per function, while task steps reference it without repeating prose test-case inventories.

## Open findings

(none — every finding has been disposed)
