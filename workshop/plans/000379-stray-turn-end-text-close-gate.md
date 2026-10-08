---
gate: boundary-review
issue: 379
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-07T20:17:01-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Minor
          title: Imported parser benchmarks reference missing fixtures
          detail: third_party/vt/ansiparser/parser_test.go:185 and :200 read fixtures/demo.vte and fixtures/UTF-8-demo.txt, neither included in the package. Running go test ./ansiparser -run '^$' -bench '^BenchmarkParser(UTF8)?$' -benchtime=1x fails for both. Include appropriately licensed fixtures or remove the unsupported benchmarks.
          family: benchmark-fixture-completeness
          round: 1
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#379 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-07T20:17:01-07:00 (codex) — passed

### Raised

- **BR-1** [Minor] `benchmark-fixture-completeness` Imported parser benchmarks reference missing fixtures
  third_party/vt/ansiparser/parser_test.go:185 and :200 read fixtures/demo.vte and fixtures/UTF-8-demo.txt, neither included in the package. Running go test ./ansiparser -run '^$' -bench '^BenchmarkParser(UTF8)?$' -benchtime=1x fails for both. Include appropriately licensed fixtures or remove the unsupported benchmarks.

## Open findings

- **BR-1** [Minor] `benchmark-fixture-completeness` Imported parser benchmarks reference missing fixtures
