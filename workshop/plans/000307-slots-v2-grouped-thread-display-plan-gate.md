---
gate: plan-quality
issue: 307
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-23T16:57:03-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Define the executable fallback for ordinary checkout roots and full paths
          detail: PresentThreads is required to group ordinary rows by verified checkout scope and render actual checkout paths, but ActionableThreadSummary exposes only RepoScope, StartingPath, and WorkingPath (cmd/internal/couchcore/actionableinventory.go:168-198), while ResolveRepoScope only hashes the supplied path and performs no discovery (cmd/internal/launcher/scope.go:21-37). The plan's “missing/unmatched legacy paths fall back to the scope as sort key and retain existing display-label behavior” does not define the full path, group qualifier, or collision behavior, so the agreed rendering and activation contract is not implementable for that class. Specify the authoritative root source or an exact typed fallback contract and boundary tests.
          family: checkout-root-identity
          round: 1
        - id: PQ-2
          severity: Minor
          title: Replace enumerated test-case prose with one adversarial strategy per risky function
          detail: Task 1, Task 2, and Task 3 enumerate many concrete cases in prose. Name the risky functions and give one strategy line each describing the malformed, permuted, stale, or width-constrained input class and mechanical oracle; let the executable tests contain the individual cases.
          family: test-strategy-compression
          round: 1
      blocked: true
---

# Gate ledger — pair#307 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-23T16:57:03-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `checkout-root-identity` Define the executable fallback for ordinary checkout roots and full paths
  PresentThreads is required to group ordinary rows by verified checkout scope and render actual checkout paths, but ActionableThreadSummary exposes only RepoScope, StartingPath, and WorkingPath (cmd/internal/couchcore/actionableinventory.go:168-198), while ResolveRepoScope only hashes the supplied path and performs no discovery (cmd/internal/launcher/scope.go:21-37). The plan's “missing/unmatched legacy paths fall back to the scope as sort key and retain existing display-label behavior” does not define the full path, group qualifier, or collision behavior, so the agreed rendering and activation contract is not implementable for that class. Specify the authoritative root source or an exact typed fallback contract and boundary tests.
- **PQ-2** [Minor] `test-strategy-compression` Replace enumerated test-case prose with one adversarial strategy per risky function
  Task 1, Task 2, and Task 3 enumerate many concrete cases in prose. Name the risky functions and give one strategy line each describing the malformed, permuted, stale, or width-constrained input class and mechanical oracle; let the executable tests contain the individual cases.

## Open findings

- **PQ-1** [Important] `checkout-root-identity` Define the executable fallback for ordinary checkout roots and full paths
- **PQ-2** [Minor] `test-strategy-compression` Replace enumerated test-case prose with one adversarial strategy per risky function
