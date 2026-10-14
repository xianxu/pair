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
    - "n": 2
      timestamp: "2026-10-13T16:23:05-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Step authority, explicit uncertainty transitions, reconciliation evidence, bounded escalation and worker shutdown now cover the requested ownership model.
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: Named strategies were added, but Tasks 1–4 retain their prose test-case inventories. Explicitly supersede those inventories with the strategy table and name Board.ResolveSelection instead of “Board conditional resolution”; preserve revision history while making the active test contract unambiguous.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: The revision defines quota accounting, reference and intent lifetimes, dedupe collection, recording-deletion sweeping and preservation of published bundle evidence.
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-10-13T16:24:14-07:00"
      agent: codex
      dispose:
        - id: PQ-2
          disposition: addressed
          note: The active revision supersedes prose inventories with named production functions, adversarial strategies, controlled ordering and independent ownership invariants.
          round: 3
      blocked: false
    - "n": 4
      timestamp: "2026-10-13T16:27:40-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Step owns transitions, with evidence-based reconciliation and protected uncertain outcomes.
          round: 4
        - id: PQ-2
          disposition: addressed
          note: Named production functions and adversarial strategies explicitly supersede earlier prose inventories.
          round: 4
        - id: PQ-3
          disposition: addressed
          note: Reservations, intents, references and retired records have explicit accounting and cleanup rules.
          round: 4
      blocked: false
content_hash: 5e2f018a63047e3f3377fde47b99b8c1ec1a8ca3966f99a8b9fb39cb63f56e63
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

## Round 2 — 2026-10-13T16:23:05-07:00 (codex) — BLOCKED

### Disposed

- PQ-1 — addressed — Step authority, explicit uncertainty transitions, reconciliation evidence, bounded escalation and worker shutdown now cover the requested ownership model.
- PQ-2 — not-addressed — Named strategies were added, but Tasks 1–4 retain their prose test-case inventories. Explicitly supersede those inventories with the strategy table and name Board.ResolveSelection instead of “Board conditional resolution”; preserve revision history while making the active test contract unambiguous.
- PQ-3 — addressed — The revision defines quota accounting, reference and intent lifetimes, dedupe collection, recording-deletion sweeping and preservation of published bundle evidence.

## Round 3 — 2026-10-13T16:24:14-07:00 (codex) — passed

### Disposed

- PQ-2 — addressed — The active revision supersedes prose inventories with named production functions, adversarial strategies, controlled ordering and independent ownership invariants.

## Round 4 — 2026-10-13T16:27:40-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Step owns transitions, with evidence-based reconciliation and protected uncertain outcomes.
- PQ-2 — addressed — Named production functions and adversarial strategies explicitly supersede earlier prose inventories.
- PQ-3 — addressed — Reservations, intents, references and retired records have explicit accounting and cleanup rules.

## Open findings

(none — every finding has been disposed)
