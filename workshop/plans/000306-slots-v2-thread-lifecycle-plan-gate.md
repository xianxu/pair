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
    - "n": 2
      timestamp: "2026-09-23T15:25:54-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: The plan now specifies local routing for AllocateThreadTag, no global fallback, fresh-replacement reuse, and a production-boundary test.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: The plan now specifies the spawnResolved/FinalizePark ordering seam, both interleavings, operationQueue coverage, parked blocking, existing-slot recovery, and ownership assertions.
          round: 2
      findings:
        - id: PQ-3
          severity: Important
          title: 'This is the 2nd finding in family `local-authority-consumer-enumeration`: enumerate every lifecycle mutation routed to local slot authority'
          detail: The plan lists AllocateThreadTag, CreateThread, reads, selected update/archive/continuation methods, and GC consumers, but omits the park/start mutation family including BeginPark, AdvancePark, AppendParkAttempt, FinalizePark, CommitStartClaim, and the incarnation/recovery transitions visible in cmd/internal/couchcore/threadstore.go:345-495 and 549-643. State the rule that every ThreadStore operation reachable from numbered-slot open, park, resume, fresh, continuation, and recovery must resolve through the slot backend, enumerate those consumers in the plan, and add one production-boundary test proving a numbered slot's park/resume/start transitions never mutate or read the global store.
          family: local-authority-consumer-enumeration
          round: 2
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

## Round 2 — 2026-09-23T15:25:54-07:00 (codex) — BLOCKED

### Disposed

- PQ-1 — addressed — The plan now specifies local routing for AllocateThreadTag, no global fallback, fresh-replacement reuse, and a production-boundary test.
- PQ-2 — addressed — The plan now specifies the spawnResolved/FinalizePark ordering seam, both interleavings, operationQueue coverage, parked blocking, existing-slot recovery, and ownership assertions.

### Raised

- **PQ-3** [Important] `local-authority-consumer-enumeration` This is the 2nd finding in family `local-authority-consumer-enumeration`: enumerate every lifecycle mutation routed to local slot authority
  The plan lists AllocateThreadTag, CreateThread, reads, selected update/archive/continuation methods, and GC consumers, but omits the park/start mutation family including BeginPark, AdvancePark, AppendParkAttempt, FinalizePark, CommitStartClaim, and the incarnation/recovery transitions visible in cmd/internal/couchcore/threadstore.go:345-495 and 549-643. State the rule that every ThreadStore operation reachable from numbered-slot open, park, resume, fresh, continuation, and recovery must resolve through the slot backend, enumerate those consumers in the plan, and add one production-boundary test proving a numbered slot's park/resume/start transitions never mutate or read the global store.

## Open findings

- **PQ-3** [Important] `local-authority-consumer-enumeration` This is the 2nd finding in family `local-authority-consumer-enumeration`: enumerate every lifecycle mutation routed to local slot authority
