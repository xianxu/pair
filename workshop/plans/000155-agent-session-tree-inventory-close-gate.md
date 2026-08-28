---
gate: boundary-review
issue: 155
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-28T13:48:39-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: New inventory sources leave deterministic repository contract tests failing
          detail: artifactpath coverage and the issue-149 source-set contract both fail at the pinned head; classify the new sources and update the governed catalog before closing M1.
          family: repository-contracts-stay-green
          round: 1
        - id: BR-2
          severity: Critical
          title: M1 omits the promised ParentEdge and EdgeProvenance entities
          detail: model.go exposes only ParentID on Node, while the Core Concepts table requires explicit edge and provenance entities; implement and projection-test them, and reconcile the renamed Fact/Node/Forest types through a plan revision.
          family: core-concepts-match-code
          round: 1
        - id: BR-3
          severity: Critical
          title: Diagnostic severity, identity, ordering, coalescing, and absence behavior violate the registry
          detail: Every diagnostic is currently warning, storage_absent is omitted, IDs include free-form detail, sorting ignores the required severity order, and duplicates are retained; centralize and exhaustively test the documented registry.
          family: diagnostic-registry-single-source
          round: 1
        - id: BR-4
          severity: Critical
          title: Equal timestamps are ordered by time source instead of native ID
          detail: compareNativeTime inserts TimeSource into the node comparator even though the specified tuple falls through from time directly to native_id; add an equal-time mixed-source regression and implement the exact tuple.
          family: documented-total-order
          round: 1
        - id: BR-5
          severity: Critical
          title: Filesystem enumeration admits blocking special files and discards valid partial results
          detail: ListFiles rejects symlinks but accepts FIFOs, sockets, and devices, while scannerFiles drops valid entries returned with non-listing walk errors; reject every non-regular candidate and retain partial facts with structured diagnostics.
          family: storage-boundary-regular-partial
          round: 1
        - id: BR-6
          severity: Critical
          title: Missing Claude and Codex usage objects overwrite valid token usage with zero
          detail: Value-typed nested usage structs cannot distinguish absent or null usage from a real zero; require explicit presence before accepting a record and test that missing records do not replace the last valid sample.
          family: optional-metrics-require-presence
          round: 1
      boundary: M1
      blocked: true
---

# Gate ledger — pair#155 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-28T13:48:39-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `repository-contracts-stay-green` New inventory sources leave deterministic repository contract tests failing
  artifactpath coverage and the issue-149 source-set contract both fail at the pinned head; classify the new sources and update the governed catalog before closing M1.
- **BR-2** [Critical] `core-concepts-match-code` M1 omits the promised ParentEdge and EdgeProvenance entities
  model.go exposes only ParentID on Node, while the Core Concepts table requires explicit edge and provenance entities; implement and projection-test them, and reconcile the renamed Fact/Node/Forest types through a plan revision.
- **BR-3** [Critical] `diagnostic-registry-single-source` Diagnostic severity, identity, ordering, coalescing, and absence behavior violate the registry
  Every diagnostic is currently warning, storage_absent is omitted, IDs include free-form detail, sorting ignores the required severity order, and duplicates are retained; centralize and exhaustively test the documented registry.
- **BR-4** [Critical] `documented-total-order` Equal timestamps are ordered by time source instead of native ID
  compareNativeTime inserts TimeSource into the node comparator even though the specified tuple falls through from time directly to native_id; add an equal-time mixed-source regression and implement the exact tuple.
- **BR-5** [Critical] `storage-boundary-regular-partial` Filesystem enumeration admits blocking special files and discards valid partial results
  ListFiles rejects symlinks but accepts FIFOs, sockets, and devices, while scannerFiles drops valid entries returned with non-listing walk errors; reject every non-regular candidate and retain partial facts with structured diagnostics.
- **BR-6** [Critical] `optional-metrics-require-presence` Missing Claude and Codex usage objects overwrite valid token usage with zero
  Value-typed nested usage structs cannot distinguish absent or null usage from a real zero; require explicit presence before accepting a record and test that missing records do not replace the last valid sample.

## Open findings

- **BR-1** [Critical] `repository-contracts-stay-green` New inventory sources leave deterministic repository contract tests failing
- **BR-2** [Critical] `core-concepts-match-code` M1 omits the promised ParentEdge and EdgeProvenance entities
- **BR-3** [Critical] `diagnostic-registry-single-source` Diagnostic severity, identity, ordering, coalescing, and absence behavior violate the registry
- **BR-4** [Critical] `documented-total-order` Equal timestamps are ordered by time source instead of native ID
- **BR-5** [Critical] `storage-boundary-regular-partial` Filesystem enumeration admits blocking special files and discards valid partial results
- **BR-6** [Critical] `optional-metrics-require-presence` Missing Claude and Codex usage objects overwrite valid token usage with zero
