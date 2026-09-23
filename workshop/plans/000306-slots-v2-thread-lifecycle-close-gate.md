---
gate: boundary-review
issue: 306
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-23T16:20:36-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Important
          title: 'This is the 3rd finding in family `local-authority-consumer-enumeration`: enumerate every direct ThreadStore consumer, not only lifecycle delegates'
          detail: |-
            Earlier rounds established the rule that local authority requires an exhaustive consumer sweep, but the plan still says only “audit all ThreadStore receiver methods for direct IO” and does not name direct retention/metadata consumers such as ApplyThreadMetadata, RetentionSnapshot, OnboardArchiveGrace, DetachArchive, and ForgetArchiveReceipt. State the complete direct-IO enumeration, assign each to the routed local backend, and give each risky function one adversarial production-boundary guard; otherwise a local slot can still be read or mutated through an unlisted path.
            (carried from plan-quality PQ-5, deferred to the boundary review)
          family: local-authority-consumer-enumeration
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-23T16:20:36-07:00"
      agent: codex
      findings:
        - id: BR-2
          severity: Critical
          title: Local lifecycle readers bypass symlink and metadata-boundary validation
          detail: threadstore.go uses os.ReadFile directly for local current records in update, snapshot, successful-start, and readThreadLocked paths, allowing a symlinked thread.json to be treated as authoritative. Route every local payload read through the guarded reader and add a regression test proving the mutation fails without touching the symlink target. ARCH-SECURE.
          family: local-metadata-boundary-validation
          round: 2
        - id: BR-3
          severity: Important
          title: README ships unresolved edit markers
          detail: "README.md:405 contains an unresolved \U0001F916 deletion/replacement marker, leaving the user-facing numbered-slot documentation malformed."
          family: unresolved-human-markers
          round: 2
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-09-23T16:38:43-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: The plan now enumerates direct lifecycle, metadata, inventory, archive, continuation, and all five GC consumers; the implementation routes them through shared backend resolvers.
          round: 3
        - id: BR-2
          disposition: addressed
          note: All local current-record reads use readPayload/readOptionalPayload/readRetentionFile, and threadstore_reader_test.go proves exported reads and mutations reject symlinks without changing the target.
          round: 3
        - id: BR-3
          disposition: addressed
          note: README.md:405-417 now contains clean numbered-slot documentation with no unresolved edit marker.
          round: 3
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#306 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-23T16:20:36-07:00 (sdlc) — passed

### Raised

- **BR-1** [Important] `local-authority-consumer-enumeration` This is the 3rd finding in family `local-authority-consumer-enumeration`: enumerate every direct ThreadStore consumer, not only lifecycle delegates
  Earlier rounds established the rule that local authority requires an exhaustive consumer sweep, but the plan still says only “audit all ThreadStore receiver methods for direct IO” and does not name direct retention/metadata consumers such as ApplyThreadMetadata, RetentionSnapshot, OnboardArchiveGrace, DetachArchive, and ForgetArchiveReceipt. State the complete direct-IO enumeration, assign each to the routed local backend, and give each risky function one adversarial production-boundary guard; otherwise a local slot can still be read or mutated through an unlisted path.
  (carried from plan-quality PQ-5, deferred to the boundary review)

## Round 2 — 2026-09-23T16:20:36-07:00 (codex) — BLOCKED

### Raised

- **BR-2** [Critical] `local-metadata-boundary-validation` Local lifecycle readers bypass symlink and metadata-boundary validation
  threadstore.go uses os.ReadFile directly for local current records in update, snapshot, successful-start, and readThreadLocked paths, allowing a symlinked thread.json to be treated as authoritative. Route every local payload read through the guarded reader and add a regression test proving the mutation fails without touching the symlink target. ARCH-SECURE.
- **BR-3** [Important] `unresolved-human-markers` README ships unresolved edit markers
  README.md:405 contains an unresolved 🤖 deletion/replacement marker, leaving the user-facing numbered-slot documentation malformed.

## Round 3 — 2026-09-23T16:38:43-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — The plan now enumerates direct lifecycle, metadata, inventory, archive, continuation, and all five GC consumers; the implementation routes them through shared backend resolvers.
- BR-2 — addressed — All local current-record reads use readPayload/readOptionalPayload/readRetentionFile, and threadstore_reader_test.go proves exported reads and mutations reject symlinks without changing the target.
- BR-3 — addressed — README.md:405-417 now contains clean numbered-slot documentation with no unresolved edit marker.

## Open findings

(none — every finding has been disposed)
