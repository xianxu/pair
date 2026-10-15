---
gate: plan-quality
issue: 528
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-13T09:33:13-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Define the verified handoff event that releases the terminal lock
          detail: 'ARCH-ORDER / ARCH-CONSTRAINTS: the plan requires release before interactive client exit, but LaunchSession and AttachSession use runBlockingHandoff, which waits for that exit (cmd/internal/launcher/osruntime.go:144,192,772). Name the completion evidence, observing owner, bounded cancellation/unknown-outcome behavior and controllable ordering seam before implementing the lock.'
          family: effect-lock-completion-contract
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace the nonexistent session-owner metadata extension with a concrete attestation design
          detail: 'ARCH-DRY / ARCH-SECURE / ARCH-FUNERAL: cmd/internal/artifactpath/session_owner.go:8 is a pure artifact-path decoder, not the persisted metadata and launcher writer/reader claimed by the shipping policy. Identify the actual reusable record or declare a new artifact with its publication owner, path, versioned validation and replacement/removal lifecycle.'
          family: durable-authority-owner-contract
          round: 1
        - id: PQ-3
          severity: Important
          title: Define recovery evidence ownership after the start claim is cleared
          detail: 'ARCH-ORDER / ARCH-FUNERAL: recovery alternatives must survive successful resume and restart, but their proposed StartClaim owner is cleared on registered and unknown promotion (cmd/internal/couchcore/starttransaction.go:120–131). Name the durable destination, transfer/supersession transitions and capture retention or expiry policy; the existing collector protects active readers and otherwise expires captures by age.'
          family: durable-state-lifetime-contract
          round: 1
        - id: PQ-4
          severity: Important
          title: Replace active prose case inventories with named function-level test strategies
          detail: The active revisions enumerate test cases while leaving risky parser, attestation and recovery-intent transition functions unnamed. Append an authoritative compact test contract naming each unit-tested function with one adversarial input/event class and mechanical guard, including reproducible ordering for lifecycle effects, rather than further enumerating cases.
          family: function-level-test-strategy
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-13T09:36:20-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: AdmissionVerified now defines completion evidence, lease ownership, bounded cancellation and channel-controlled ordering independently of client exit.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: The corrected design declares a new versioned policy artifact with an exact path, atomic publisher, identity validation and replacement/removal lifecycle.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: RecoveryContext receives intent before claim clearance and preserves bounded alternatives across restart with capture-aligned expiry and reader leases.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: The authoritative function-level test table supersedes prose case inventories and names adversarial strategies and reproducible lifecycle guards.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-13T09:38:05-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: AdmissionVerified defines lock release; failure paths reap the owned client and join the watcher.
          round: 3
        - id: PQ-2
          disposition: addressed
          note: A new versioned SessionLaunchPolicy has explicit publication, validation and cleanup owners.
          round: 3
        - id: PQ-3
          disposition: addressed
          note: RecoveryContext receives intent before claim clearing and expires with the original capture.
          round: 3
        - id: PQ-4
          disposition: addressed
          note: The authoritative test table names functions and adversarial strategies, superseding historical case inventories.
          round: 3
      blocked: false
    - "n": 4
      timestamp: "2026-10-13T20:45:53-07:00"
      agent: codex
      blocked: false
content_hash: 10238408ee0509ba664b4e2ef2e8509267c3bb3aa02f379ae4804cc2c31c2403
---

# Gate ledger — pair#528 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T09:33:13-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `effect-lock-completion-contract` Define the verified handoff event that releases the terminal lock
  ARCH-ORDER / ARCH-CONSTRAINTS: the plan requires release before interactive client exit, but LaunchSession and AttachSession use runBlockingHandoff, which waits for that exit (cmd/internal/launcher/osruntime.go:144,192,772). Name the completion evidence, observing owner, bounded cancellation/unknown-outcome behavior and controllable ordering seam before implementing the lock.
- **PQ-2** [Important] `durable-authority-owner-contract` Replace the nonexistent session-owner metadata extension with a concrete attestation design
  ARCH-DRY / ARCH-SECURE / ARCH-FUNERAL: cmd/internal/artifactpath/session_owner.go:8 is a pure artifact-path decoder, not the persisted metadata and launcher writer/reader claimed by the shipping policy. Identify the actual reusable record or declare a new artifact with its publication owner, path, versioned validation and replacement/removal lifecycle.
- **PQ-3** [Important] `durable-state-lifetime-contract` Define recovery evidence ownership after the start claim is cleared
  ARCH-ORDER / ARCH-FUNERAL: recovery alternatives must survive successful resume and restart, but their proposed StartClaim owner is cleared on registered and unknown promotion (cmd/internal/couchcore/starttransaction.go:120–131). Name the durable destination, transfer/supersession transitions and capture retention or expiry policy; the existing collector protects active readers and otherwise expires captures by age.
- **PQ-4** [Important] `function-level-test-strategy` Replace active prose case inventories with named function-level test strategies
  The active revisions enumerate test cases while leaving risky parser, attestation and recovery-intent transition functions unnamed. Append an authoritative compact test contract naming each unit-tested function with one adversarial input/event class and mechanical guard, including reproducible ordering for lifecycle effects, rather than further enumerating cases.

## Round 2 — 2026-10-13T09:36:20-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — AdmissionVerified now defines completion evidence, lease ownership, bounded cancellation and channel-controlled ordering independently of client exit.
- PQ-2 — addressed — The corrected design declares a new versioned policy artifact with an exact path, atomic publisher, identity validation and replacement/removal lifecycle.
- PQ-3 — addressed — RecoveryContext receives intent before claim clearance and preserves bounded alternatives across restart with capture-aligned expiry and reader leases.
- PQ-4 — addressed — The authoritative function-level test table supersedes prose case inventories and names adversarial strategies and reproducible lifecycle guards.

## Round 3 — 2026-10-13T09:38:05-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — AdmissionVerified defines lock release; failure paths reap the owned client and join the watcher.
- PQ-2 — addressed — A new versioned SessionLaunchPolicy has explicit publication, validation and cleanup owners.
- PQ-3 — addressed — RecoveryContext receives intent before claim clearing and expires with the original capture.
- PQ-4 — addressed — The authoritative test table names functions and adversarial strategies, superseding historical case inventories.

## Round 4 — 2026-10-13T20:45:53-07:00 (codex) — passed

## Open findings

(none — every finding has been disposed)
