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
    - "n": 3
      timestamp: "2026-09-23T15:27:38-07:00"
      agent: codex
      dispose:
        - id: PQ-3
          disposition: addressed
          note: The revised plan states the universal routing rule, enumerates the park/start/incarnation/continuation/recovery families, and adds a production-boundary local-versus-global authority test.
          round: 3
      findings:
        - id: PQ-4
          severity: Important
          title: Name every GC consumer function and give each its own test strategy
          detail: The strategy names “Snapshot / ArchivedThreads / CouchReferences” rather than the concrete functions CouchReferences.Snapshot, Detach, Forget, Onboard, Recover, and ThreadStore.ArchivedThreads. Under ARCH-PURPOSE, name each production function and provide one adversarial-input/mechanical-guard line for each so the five GC routing consumers cannot be silently omitted.
          family: function-level-test-strategy
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-23T15:29:22-07:00"
      agent: codex
      dispose:
        - id: PQ-4
          disposition: addressed
          note: The plan names ThreadStore.ArchivedThreads and each concrete CouchReferences consumer with an individual adversarial test strategy.
          round: 4
      findings:
        - id: PQ-5
          severity: Important
          title: 'This is the 3rd finding in family `local-authority-consumer-enumeration`: enumerate every direct ThreadStore consumer, not only lifecycle delegates'
          detail: Earlier rounds established the rule that local authority requires an exhaustive consumer sweep, but the plan still says only “audit all ThreadStore receiver methods for direct IO” and does not name direct retention/metadata consumers such as ApplyThreadMetadata, RetentionSnapshot, OnboardArchiveGrace, DetachArchive, and ForgetArchiveReceipt. State the complete direct-IO enumeration, assign each to the routed local backend, and give each risky function one adversarial production-boundary guard; otherwise a local slot can still be read or mutated through an unlisted path.
          family: local-authority-consumer-enumeration
          round: 4
      blocked: false
content_hash: 2b10b9e4d3b7fc7db695b6de7931b49559cc5cfa367228dbaf79b35564c0146f
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

## Round 3 — 2026-09-23T15:27:38-07:00 (codex) — BLOCKED

### Disposed

- PQ-3 — addressed — The revised plan states the universal routing rule, enumerates the park/start/incarnation/continuation/recovery families, and adds a production-boundary local-versus-global authority test.

### Raised

- **PQ-4** [Important] `function-level-test-strategy` Name every GC consumer function and give each its own test strategy
  The strategy names “Snapshot / ArchivedThreads / CouchReferences” rather than the concrete functions CouchReferences.Snapshot, Detach, Forget, Onboard, Recover, and ThreadStore.ArchivedThreads. Under ARCH-PURPOSE, name each production function and provide one adversarial-input/mechanical-guard line for each so the five GC routing consumers cannot be silently omitted.

## Round 4 — 2026-09-23T15:29:22-07:00 (codex) — passed

### Disposed

- PQ-4 — addressed — The plan names ThreadStore.ArchivedThreads and each concrete CouchReferences consumer with an individual adversarial test strategy.

### Raised

- **PQ-5** [Important] `local-authority-consumer-enumeration` This is the 3rd finding in family `local-authority-consumer-enumeration`: enumerate every direct ThreadStore consumer, not only lifecycle delegates
  Earlier rounds established the rule that local authority requires an exhaustive consumer sweep, but the plan still says only “audit all ThreadStore receiver methods for direct IO” and does not name direct retention/metadata consumers such as ApplyThreadMetadata, RetentionSnapshot, OnboardArchiveGrace, DetachArchive, and ForgetArchiveReceipt. State the complete direct-IO enumeration, assign each to the routed local backend, and give each risky function one adversarial production-boundary guard; otherwise a local slot can still be read or mutated through an unlisted path.

## Open findings

- **PQ-5** [Important] `local-authority-consumer-enumeration` This is the 3rd finding in family `local-authority-consumer-enumeration`: enumerate every direct ThreadStore consumer, not only lifecycle delegates
