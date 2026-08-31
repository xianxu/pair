---
gate: boundary-review
issue: 151
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-30T20:57:48-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Required start tokens break the existing Console start consumer
          detail: start now requires an implicit token, but Console still dispatches start with only path; both existing Console start regressions time out. Sweep every shared-operation consumer and make Console perform prepare-start followed by token-bound start, or stage the schema change atomically.
          family: shared-operation-consumer-sweep
          round: 1
        - id: BR-2
          severity: Critical
          title: The actionable projector accepts corrupt verified-park evidence
          detail: ProjectActionableThreads checks only that VerifiedPark is non-nil and never validates the durable record, so a zero identity/attempt and other malformed records can become actionable despite the fail-closed Spec. Validate the record and add malformed live and parked regressions using valid positive fixtures.
          family: lifecycle-evidence-validation
          round: 1
        - id: BR-3
          severity: Important
          title: New production files are absent from the exhaustive source inventory
          detail: TestProductionArtifactReferencesAreExactlyClassified rejects actionableinventory.go, startgrant.go, and startresolution.go. Classify all three in the artifact authority inventory.
          family: exhaustive-production-source-inventory
          round: 1
        - id: BR-4
          severity: Important
          title: Atlas claims the ordinary switcher already consumes the actionable inventory
          detail: The atlas describes ActionableThreadInventory as the current switcher source, while run.go still wires ThreadInventory into the transitional panel. Document the milestone staging accurately or complete the wiring.
          family: documentation-current-state-accuracy
          round: 1
      boundary: M1
      blocked: true
---

# Gate ledger — pair#151 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-30T20:57:48-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `shared-operation-consumer-sweep` Required start tokens break the existing Console start consumer
  start now requires an implicit token, but Console still dispatches start with only path; both existing Console start regressions time out. Sweep every shared-operation consumer and make Console perform prepare-start followed by token-bound start, or stage the schema change atomically.
- **BR-2** [Critical] `lifecycle-evidence-validation` The actionable projector accepts corrupt verified-park evidence
  ProjectActionableThreads checks only that VerifiedPark is non-nil and never validates the durable record, so a zero identity/attempt and other malformed records can become actionable despite the fail-closed Spec. Validate the record and add malformed live and parked regressions using valid positive fixtures.
- **BR-3** [Important] `exhaustive-production-source-inventory` New production files are absent from the exhaustive source inventory
  TestProductionArtifactReferencesAreExactlyClassified rejects actionableinventory.go, startgrant.go, and startresolution.go. Classify all three in the artifact authority inventory.
- **BR-4** [Important] `documentation-current-state-accuracy` Atlas claims the ordinary switcher already consumes the actionable inventory
  The atlas describes ActionableThreadInventory as the current switcher source, while run.go still wires ThreadInventory into the transitional panel. Document the milestone staging accurately or complete the wiring.

## Open findings

- **BR-1** [Critical] `shared-operation-consumer-sweep` Required start tokens break the existing Console start consumer
- **BR-2** [Critical] `lifecycle-evidence-validation` The actionable projector accepts corrupt verified-park evidence
- **BR-3** [Important] `exhaustive-production-source-inventory` New production files are absent from the exhaustive source inventory
- **BR-4** [Important] `documentation-current-state-accuracy` Atlas claims the ordinary switcher already consumes the actionable inventory
