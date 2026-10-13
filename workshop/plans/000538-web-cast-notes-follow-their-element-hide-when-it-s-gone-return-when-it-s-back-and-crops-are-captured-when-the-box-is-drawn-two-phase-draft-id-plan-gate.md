---
gate: plan-quality
issue: 538
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-13T16:20:18-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Define and enforce the capture ownership transition model before implementation.
          detail: 'ARCH-ORDER: Name the pure transition function, state/event/effect types, and encapsulation that prevents adapters from changing authoritative ownership directly. Enumerate legal outcomes for attachment and publication uncertainty, cancellation or expiry during an unknown save, and resolution during transfer; specify reconciliation evidence, bounded escalation, and worker shutdown ownership. Store.Prepare can return an error after publishing a bundle, so that outcome must preserve and reconcile both ownership records rather than imply nonpublication.'
          family: explicit-ownership-transitions
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace prose test-case inventories with named functions and adversarial strategies.
          detail: 'ARCH-PURE: Tasks 1–4 enumerate test cases but do not name the new scoring, projection, lifecycle-transition, or recovery functions to be unit-tested. Compress those lists into one strategy line per risky function, naming the adversarial input class and mechanical guard; sequence tests must drive the production transition function with controlled ordering and assert independently stated ownership invariants.'
          family: function-level-test-strategy
          round: 1
        - id: PQ-3
          severity: Important
          title: Specify cleanup and capacity accounting for capture references and recovery intents.
          detail: 'ARCH-FUNERAL: The plan retains stable capture references after moving bytes into bundles but does not say when reference records and completed attachment/transfer intents are removed or whether they consume the 128-record limit. Define those lifetimes and the concrete cleanup hook for captures stored outside a recording when that recording is deleted; ensure repeated submit/resolve cycles reclaim capacity without deleting evidence still needed by open notes or published bundles.'
          family: durable-artifact-lifecycle
          round: 1
      blocked: true
---

# Gate ledger — pair#538 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T16:20:18-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `explicit-ownership-transitions` Define and enforce the capture ownership transition model before implementation.
  ARCH-ORDER: Name the pure transition function, state/event/effect types, and encapsulation that prevents adapters from changing authoritative ownership directly. Enumerate legal outcomes for attachment and publication uncertainty, cancellation or expiry during an unknown save, and resolution during transfer; specify reconciliation evidence, bounded escalation, and worker shutdown ownership. Store.Prepare can return an error after publishing a bundle, so that outcome must preserve and reconcile both ownership records rather than imply nonpublication.
- **PQ-2** [Important] `function-level-test-strategy` Replace prose test-case inventories with named functions and adversarial strategies.
  ARCH-PURE: Tasks 1–4 enumerate test cases but do not name the new scoring, projection, lifecycle-transition, or recovery functions to be unit-tested. Compress those lists into one strategy line per risky function, naming the adversarial input class and mechanical guard; sequence tests must drive the production transition function with controlled ordering and assert independently stated ownership invariants.
- **PQ-3** [Important] `durable-artifact-lifecycle` Specify cleanup and capacity accounting for capture references and recovery intents.
  ARCH-FUNERAL: The plan retains stable capture references after moving bytes into bundles but does not say when reference records and completed attachment/transfer intents are removed or whether they consume the 128-record limit. Define those lifetimes and the concrete cleanup hook for captures stored outside a recording when that recording is deleted; ensure repeated submit/resolve cycles reclaim capacity without deleting evidence still needed by open notes or published bundles.

## Open findings

- **PQ-1** [Important] `explicit-ownership-transitions` Define and enforce the capture ownership transition model before implementation.
- **PQ-2** [Important] `function-level-test-strategy` Replace prose test-case inventories with named functions and adversarial strategies.
- **PQ-3** [Important] `durable-artifact-lifecycle` Specify cleanup and capacity accounting for capture references and recovery intents.
