---
gate: plan-quality
issue: 346
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-29T10:08:52-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Specify capture ownership and partial-transfer recovery before implementation
          detail: 'ARCH-ORDER: M3 leaves lock inheritance versus reacquisition and partial archive recovery undecided. Name the capture transition authority, lock order and handoff across wrapper/quit/exec, durable family identity, and recovery after interruption between member transfers or metadata publication; specify a reproducible sequence-testing seam.'
          family: durable-effect-recovery-contract
          round: 1
        - id: PQ-2
          severity: Important
          title: Define launch observation when baseline enumeration is incomplete
          detail: 'ARCH-ORDER and ARCH-SECURE: prepareRuntimeLaunch currently refuses DiagnosticStorageUnreadable, while the plan promises nonblocking resume and uses baseline absence to authorize new-file evidence. Specify startup behavior and represent unknown baseline coverage separately from confirmed absence, including safe later observation without counting historical bytes or accepting a false UUID handshake.'
          family: incomplete-observation-is-not-absence
          round: 1
        - id: PQ-3
          severity: Important
          title: Compress test inventories into named risky functions and mechanical strategies
          detail: The milestone checklists enumerate test cases but do not consistently name the functions to unit-test or their adversarial-input strategy. Replace those inventories with concise strategy lines for the ledger fold, target projection, suffix selection, root classification, and capture transitions, retaining production-boundary acceptance checks.
          family: function-level-test-strategy
          round: 1
        - id: PQ-4
          severity: Minor
          title: Quantify the new capture-lock wait
          detail: 'ARCH-CONSTRAINTS: Replace “bounded exclusive-lock acquisition” with a deadline, its basis, and the visible refusal behavior when exceeded.'
          family: explicit-operating-envelope
          round: 1
        - id: PQ-5
          severity: Minor
          title: State the lifecycle of expanded ledger history
          detail: 'ARCH-FUNERAL: “Append-only ledger owns requested/observed history” does not identify removal or bound growth. Name the existing owner cleanup that removes it, or state retained bytes per launch and how growth is measured.'
          family: durable-history-lifecycle
          round: 1
        - id: PQ-6
          severity: Minor
          title: Match live conformance to the native behavior being modeled
          detail: 'ARCH-MOCK: A metadata-only incident inventory cannot check the append/source/correlation semantics modeled by native runtime fixtures. Name a safe conformance check and cadence for that dependency surface without interrupting brain:0.'
          family: external-model-conformance
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-29T10:12:03-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: NextCaptureAction owns capture transitions; lock ordering, exec handoff, durable transaction identity, partial-transfer reconciliation and effect-by-effect failure injection are specified.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: BaselineComplete distinguishes unknown coverage from absence; startup remains available, and later observation establishes fresh transcript and prompt boundaries before accepting evidence.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Strategy tables name risky functions and adversarial mechanical checks while retaining production-boundary acceptance coverage.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Writer acquisition refuses immediately; archive serialization has a five-second operational deadline with a visible diagnostic and untouched sources.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Ledger history follows existing owner/session retention; ordinary launch growth and duplicate-observation idempotence are stated.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: M2 conformance compares sanitized native structural observations and established send matches against adapter/round behavior, repeats on adapter upgrades, and preserves brain:0.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-09-29T10:14:49-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Capture ownership, lock ordering, transaction phases, and partial-transfer recovery are explicit.
          round: 3
        - id: PQ-2
          disposition: addressed
          note: Incomplete baselines remain unknown; later complete scans establish conservative observation epochs.
          round: 3
        - id: PQ-3
          disposition: addressed
          note: Test tables name risky functions and mechanical strategies instead of enumerating cases.
          round: 3
        - id: PQ-4
          disposition: addressed
          note: Writer acquisition is nonblocking; archive transaction acquisition has a five-second deadline.
          round: 3
        - id: PQ-5
          disposition: addressed
          note: Ledger history follows owner/session retention, with stated per-launch growth and idempotent confirmation.
          round: 3
        - id: PQ-6
          disposition: addressed
          note: Native conformance compares incident structural observations with adapter and round behavior and names upgrade-time reruns.
          round: 3
      blocked: false
    - "n": 4
      timestamp: "2026-09-29T10:22:26-07:00"
      agent: claude
      blocked: false
      protocol_error: no valid findings block
content_hash: 3922615702452baa05b2a08184aa054105f423db42e8f2fdfd0eb9b1ac147a92
---

# Gate ledger — pair#346 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-29T10:08:52-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `durable-effect-recovery-contract` Specify capture ownership and partial-transfer recovery before implementation
  ARCH-ORDER: M3 leaves lock inheritance versus reacquisition and partial archive recovery undecided. Name the capture transition authority, lock order and handoff across wrapper/quit/exec, durable family identity, and recovery after interruption between member transfers or metadata publication; specify a reproducible sequence-testing seam.
- **PQ-2** [Important] `incomplete-observation-is-not-absence` Define launch observation when baseline enumeration is incomplete
  ARCH-ORDER and ARCH-SECURE: prepareRuntimeLaunch currently refuses DiagnosticStorageUnreadable, while the plan promises nonblocking resume and uses baseline absence to authorize new-file evidence. Specify startup behavior and represent unknown baseline coverage separately from confirmed absence, including safe later observation without counting historical bytes or accepting a false UUID handshake.
- **PQ-3** [Important] `function-level-test-strategy` Compress test inventories into named risky functions and mechanical strategies
  The milestone checklists enumerate test cases but do not consistently name the functions to unit-test or their adversarial-input strategy. Replace those inventories with concise strategy lines for the ledger fold, target projection, suffix selection, root classification, and capture transitions, retaining production-boundary acceptance checks.
- **PQ-4** [Minor] `explicit-operating-envelope` Quantify the new capture-lock wait
  ARCH-CONSTRAINTS: Replace “bounded exclusive-lock acquisition” with a deadline, its basis, and the visible refusal behavior when exceeded.
- **PQ-5** [Minor] `durable-history-lifecycle` State the lifecycle of expanded ledger history
  ARCH-FUNERAL: “Append-only ledger owns requested/observed history” does not identify removal or bound growth. Name the existing owner cleanup that removes it, or state retained bytes per launch and how growth is measured.
- **PQ-6** [Minor] `external-model-conformance` Match live conformance to the native behavior being modeled
  ARCH-MOCK: A metadata-only incident inventory cannot check the append/source/correlation semantics modeled by native runtime fixtures. Name a safe conformance check and cadence for that dependency surface without interrupting brain:0.

## Round 2 — 2026-09-29T10:12:03-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — NextCaptureAction owns capture transitions; lock ordering, exec handoff, durable transaction identity, partial-transfer reconciliation and effect-by-effect failure injection are specified.
- PQ-2 — addressed — BaselineComplete distinguishes unknown coverage from absence; startup remains available, and later observation establishes fresh transcript and prompt boundaries before accepting evidence.
- PQ-3 — addressed — Strategy tables name risky functions and adversarial mechanical checks while retaining production-boundary acceptance coverage.
- PQ-4 — addressed — Writer acquisition refuses immediately; archive serialization has a five-second operational deadline with a visible diagnostic and untouched sources.
- PQ-5 — addressed — Ledger history follows existing owner/session retention; ordinary launch growth and duplicate-observation idempotence are stated.
- PQ-6 — addressed — M2 conformance compares sanitized native structural observations and established send matches against adapter/round behavior, repeats on adapter upgrades, and preserves brain:0.

## Round 3 — 2026-09-29T10:14:49-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Capture ownership, lock ordering, transaction phases, and partial-transfer recovery are explicit.
- PQ-2 — addressed — Incomplete baselines remain unknown; later complete scans establish conservative observation epochs.
- PQ-3 — addressed — Test tables name risky functions and mechanical strategies instead of enumerating cases.
- PQ-4 — addressed — Writer acquisition is nonblocking; archive transaction acquisition has a five-second deadline.
- PQ-5 — addressed — Ledger history follows owner/session retention, with stated per-launch growth and idempotent confirmation.
- PQ-6 — addressed — Native conformance compares incident structural observations with adapter and round behavior and names upgrade-time reruns.

## Round 4 — 2026-09-29T10:22:26-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Open findings

(none — every finding has been disposed)
