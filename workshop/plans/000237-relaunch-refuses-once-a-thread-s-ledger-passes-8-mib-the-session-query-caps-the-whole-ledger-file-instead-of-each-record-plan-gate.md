---
gate: plan-quality
issue: 237
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-12T17:41:14-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Minor
          title: readJSONLArtifact duplicates the visitJSONLinesAt chunk loop; state the deliberate deltas (tolerant tail, no CR strip) or share a core
          detail: scan_helpers.go:162 vs scan_helpers.go:31. Acceptable for a bugfix, but the plan should say why two loops rather than one with a tolerant-tail option. ARCH-DRY.
          family: shared-helper-reuse
          round: 1
        - id: PQ-2
          severity: Minor
          title: Plan removes the file cap without naming the new memory envelope or the polling caller that pays for unbounded ledger growth
          detail: The ledger is still read whole; titlepoller/runtime.go:117 calls QuerySession on a cadence. Name issue 238's row shrink as the bound and note the poller multiplier. ARCH-CONSTRAINTS.
          family: operating-envelope-unstated
          round: 1
        - id: PQ-3
          severity: Minor
          title: No adversarial guard named for the new byte scanner; seed scan_fuzz_test.go with readJSONLArtifact
          detail: 'One strategy line: fuzz for equivalence to visitJSONLinesAt on terminated input and partial-tail passthrough on unterminated input, using FakeRuntime.ReadAt.'
          family: risky-function-guard-missing
          round: 1
        - id: PQ-4
          severity: Minor
          title: Caller roster omits titlepoller and contextcmd
          detail: cmd/internal/titlepoller/runtime.go:117 and cmd/internal/contextcmd/contextcmd.go:68 also call QuerySession; list them so the close review's sweep matches the code. ARCH-PURPOSE.
          family: consumer-enumeration-incomplete
          round: 1
      blocked: false
content_hash: 413f6402b9ecf80537d9515a89de4580036e828e36001d16f5902582a19a106f
---

# Gate ledger — pair#237 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-12T17:41:14-07:00 (claude) — passed

### Raised

- **PQ-1** [Minor] `shared-helper-reuse` readJSONLArtifact duplicates the visitJSONLinesAt chunk loop; state the deliberate deltas (tolerant tail, no CR strip) or share a core
  scan_helpers.go:162 vs scan_helpers.go:31. Acceptable for a bugfix, but the plan should say why two loops rather than one with a tolerant-tail option. ARCH-DRY.
- **PQ-2** [Minor] `operating-envelope-unstated` Plan removes the file cap without naming the new memory envelope or the polling caller that pays for unbounded ledger growth
  The ledger is still read whole; titlepoller/runtime.go:117 calls QuerySession on a cadence. Name issue 238's row shrink as the bound and note the poller multiplier. ARCH-CONSTRAINTS.
- **PQ-3** [Minor] `risky-function-guard-missing` No adversarial guard named for the new byte scanner; seed scan_fuzz_test.go with readJSONLArtifact
  One strategy line: fuzz for equivalence to visitJSONLinesAt on terminated input and partial-tail passthrough on unterminated input, using FakeRuntime.ReadAt.
- **PQ-4** [Minor] `consumer-enumeration-incomplete` Caller roster omits titlepoller and contextcmd
  cmd/internal/titlepoller/runtime.go:117 and cmd/internal/contextcmd/contextcmd.go:68 also call QuerySession; list them so the close review's sweep matches the code. ARCH-PURPOSE.

## Open findings

- **PQ-1** [Minor] `shared-helper-reuse` readJSONLArtifact duplicates the visitJSONLinesAt chunk loop; state the deliberate deltas (tolerant tail, no CR strip) or share a core
- **PQ-2** [Minor] `operating-envelope-unstated` Plan removes the file cap without naming the new memory envelope or the polling caller that pays for unbounded ledger growth
- **PQ-3** [Minor] `risky-function-guard-missing` No adversarial guard named for the new byte scanner; seed scan_fuzz_test.go with readJSONLArtifact
- **PQ-4** [Minor] `consumer-enumeration-incomplete` Caller roster omits titlepoller and contextcmd
