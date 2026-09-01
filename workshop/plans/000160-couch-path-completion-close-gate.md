---
gate: boundary-review
issue: 160
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-01T13:12:56-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Important
          title: Directory close failures are silently discarded
          detail: OSDirectoryBatchReader defers file.Close without joining its error into the returned scan error, contrary to Task 5's joined-terminal-error contract. Propagate the close failure and add a test that fails when it is swallowed.
          family: filesystem-terminal-error-propagation
          round: 1
        - id: BR-2
          severity: Important
          title: The directory-reader fake does not model bounded batching
          detail: The fake ignores batchSize and yields all configured entries at once, so the integration path cannot enforce the production seam's bounded-batch contract or several claimed cancellation and error behaviors. Make the fake stateful and batch-faithful, then pin those worker behaviors.
          family: external-double-contract-fidelity
          round: 1
      blocked: true
---

# Gate ledger — 000160-couch-path-completion#160 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-01T13:12:56-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `filesystem-terminal-error-propagation` Directory close failures are silently discarded
  OSDirectoryBatchReader defers file.Close without joining its error into the returned scan error, contrary to Task 5's joined-terminal-error contract. Propagate the close failure and add a test that fails when it is swallowed.
- **BR-2** [Important] `external-double-contract-fidelity` The directory-reader fake does not model bounded batching
  The fake ignores batchSize and yields all configured entries at once, so the integration path cannot enforce the production seam's bounded-batch contract or several claimed cancellation and error behaviors. Make the fake stateful and batch-faithful, then pin those worker behaviors.

## Open findings

- **BR-1** [Important] `filesystem-terminal-error-propagation` Directory close failures are silently discarded
- **BR-2** [Important] `external-double-contract-fidelity` The directory-reader fake does not model bounded batching
