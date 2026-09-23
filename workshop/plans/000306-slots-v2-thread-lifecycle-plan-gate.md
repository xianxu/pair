---
gate: plan-quality
issue: 306
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-23T15:24:22-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Enumerate and route AllocateThreadTag into local slot authority
          detail: The plan lists CreateThread and fresh replacement but omits ThreadStore.AllocateThreadTag, which production spawnResolved calls before CommitStartClaim (cmd/internal/couchcore/couch.go:450-475) and same-slot fresh replacement requires (workshop/plans/000306-slots-v2-thread-lifecycle-plan.md:315-319). Specify its routed local-store behavior and add a production-boundary test proving allocation cannot fall back to the global store. ARCH-PURPOSE.
          family: local-authority-consumer-enumeration
          round: 1
        - id: PQ-2
          severity: Important
          title: Add a production-boundary test for park versus new-slot admission ordering
          detail: The plan describes rechecks and operationQueue serialization (workshop/plans/000306-slots-v2-thread-lifecycle-plan.md:338-389) but its function-level strategies do not name the required interleaving where park finalization races new-slot creation. Add the launch/admission function, controllable ordering seam, and assertions for both orderings, parked blocking, existing-slot recovery, and no duplicate live ownership. ARCH-ORDER.
          family: admission-race-state-machine
          round: 1
      blocked: true
---

# Gate ledger — pair#306 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-23T15:24:22-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `local-authority-consumer-enumeration` Enumerate and route AllocateThreadTag into local slot authority
  The plan lists CreateThread and fresh replacement but omits ThreadStore.AllocateThreadTag, which production spawnResolved calls before CommitStartClaim (cmd/internal/couchcore/couch.go:450-475) and same-slot fresh replacement requires (workshop/plans/000306-slots-v2-thread-lifecycle-plan.md:315-319). Specify its routed local-store behavior and add a production-boundary test proving allocation cannot fall back to the global store. ARCH-PURPOSE.
- **PQ-2** [Important] `admission-race-state-machine` Add a production-boundary test for park versus new-slot admission ordering
  The plan describes rechecks and operationQueue serialization (workshop/plans/000306-slots-v2-thread-lifecycle-plan.md:338-389) but its function-level strategies do not name the required interleaving where park finalization races new-slot creation. Add the launch/admission function, controllable ordering seam, and assertions for both orderings, parked blocking, existing-slot recovery, and no duplicate live ownership. ARCH-ORDER.

## Open findings

- **PQ-1** [Important] `local-authority-consumer-enumeration` Enumerate and route AllocateThreadTag into local slot authority
- **PQ-2** [Important] `admission-race-state-machine` Add a production-boundary test for park versus new-slot admission ordering
