---
gate: plan-quality
issue: 156
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-29T12:01:27-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Critical
          title: Birth timestamps do not satisfy the non-reusable generation-token contract
          detail: Plan lines 124-125 treat Darwin Birthtimespec and Linux statx birth time as generation tokens, but both are creation timestamps; Darwin exposes a separate st_gen and Linux statx exposes no inode-generation field. Refine the platform seam to use a genuinely non-reusable generation primitive where available and otherwise report generation unavailable, so the same-inode-reuse Done-when condition can actually be enforced.
          family: file-generation-continuity
          round: 1
        - id: PQ-2
          severity: Important
          title: Compress enumerated test prose into named risky-function strategies
          detail: Plan lines 121, 145, 182, 213-238, 253-255, 292-294, and 313 enumerate cases and procedural diff steps, but do not name several risky functions that will be unit-tested. Name the JSONL framer, per-agent transition functions, catalog transaction, target-selection, and proof-validation functions, then give one adversarial-input class and mechanical guard per function.
          family: test-strategy-contract
          round: 1
        - id: PQ-3
          severity: Important
          title: ARCH-MOCK lacks a live behavioral comparison and cadence
          detail: Fixtures in lines 234-236 and the performance smoke in lines 348-351 do not compare the stateful fake with real provider behavior on a stated cadence. Name the live surfaces for append-only Claude, Codex, Muse, and Agy transcripts plus mutated Agy SQLite behavior, how observations are compared with the fake, and when that conformance check runs.
          family: external-conformance-cadence
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-29T12:05:51-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: The plan uses Darwin stat.st_gen and reports generation unavailable on Linux or any platform lacking a genuine generation primitive.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: The risky-function strategy table names every requested surface and gives each an adversarial-input class plus a mechanical guard.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Live conformance covers all four transcript providers and copied Agy SQLite mutation behavior, with issue-time, contract-change, and monthly cadence.
          round: 2
      blocked: false
content_hash: c3ddf58b741bfb8819a4b1bef74ae83e4d49ec33a77572867cc65d4aec6c3ef0
---

# Gate ledger — pair#156 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-29T12:01:27-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Critical] `file-generation-continuity` Birth timestamps do not satisfy the non-reusable generation-token contract
  Plan lines 124-125 treat Darwin Birthtimespec and Linux statx birth time as generation tokens, but both are creation timestamps; Darwin exposes a separate st_gen and Linux statx exposes no inode-generation field. Refine the platform seam to use a genuinely non-reusable generation primitive where available and otherwise report generation unavailable, so the same-inode-reuse Done-when condition can actually be enforced.
- **PQ-2** [Important] `test-strategy-contract` Compress enumerated test prose into named risky-function strategies
  Plan lines 121, 145, 182, 213-238, 253-255, 292-294, and 313 enumerate cases and procedural diff steps, but do not name several risky functions that will be unit-tested. Name the JSONL framer, per-agent transition functions, catalog transaction, target-selection, and proof-validation functions, then give one adversarial-input class and mechanical guard per function.
- **PQ-3** [Important] `external-conformance-cadence` ARCH-MOCK lacks a live behavioral comparison and cadence
  Fixtures in lines 234-236 and the performance smoke in lines 348-351 do not compare the stateful fake with real provider behavior on a stated cadence. Name the live surfaces for append-only Claude, Codex, Muse, and Agy transcripts plus mutated Agy SQLite behavior, how observations are compared with the fake, and when that conformance check runs.

## Round 2 — 2026-08-29T12:05:51-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — The plan uses Darwin stat.st_gen and reports generation unavailable on Linux or any platform lacking a genuine generation primitive.
- PQ-2 — addressed — The risky-function strategy table names every requested surface and gives each an adversarial-input class plus a mechanical guard.
- PQ-3 — addressed — Live conformance covers all four transcript providers and copied Agy SQLite mutation behavior, with issue-time, contract-change, and monthly cadence.

## Open findings

(none — every finding has been disposed)
