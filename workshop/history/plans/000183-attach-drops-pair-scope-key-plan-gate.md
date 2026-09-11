---
gate: plan-quality
issue: 183
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-10T17:51:21-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Minor
          title: optionsFromCLI is extracted from an untested RunCLI; its refusal path stays unpinned
          detail: |-
            git grep RunCLI over cmd/internal/titlepoller/*_test.go returns nothing, so
            "Behaviour is unchanged" rests on reading alone, and the len(args) < 2 path
            makes the poller exit 0 without starting — the same silent failure class as
            pair#183. Add one assertion that optionsFromCLI returns ok == false on a
            short argv; no case list needed.
          family: silent-degradation-untested
          round: 1
        - id: PQ-2
          severity: Minor
          title: Plan does not say why the poller's contract travels by env when its sibling sidecar travels by argv
          detail: |-
            SpawnSessionWatcher hands scopeKey positionally on argv via
            sessionwatch.CommandArgs (osruntime.go:348); SessionEnv instead renders
            KEY=value for the child. The env choice is defensible (contextcmd reads
            os.Getenv, and last-duplicate-wins lets the contract override a stale
            inherited PAIR_SCOPE_KEY), but the rationale is unwritten next to the
            divergent precedent. One sentence in Core concepts.
          family: sidecar-handoff-shape
          round: 1
      blocked: false
content_hash: 173b167f0a8470c1a29c28b56818e909a52d0da5b44186edfa7105aa7d5bab30
---

# Gate ledger — pair#183 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-10T17:51:21-07:00 (claude) — passed

### Raised

- **PQ-1** [Minor] `silent-degradation-untested` optionsFromCLI is extracted from an untested RunCLI; its refusal path stays unpinned
  git grep RunCLI over cmd/internal/titlepoller/*_test.go returns nothing, so
  "Behaviour is unchanged" rests on reading alone, and the len(args) < 2 path
  makes the poller exit 0 without starting — the same silent failure class as
  pair#183. Add one assertion that optionsFromCLI returns ok == false on a
  short argv; no case list needed.
- **PQ-2** [Minor] `sidecar-handoff-shape` Plan does not say why the poller's contract travels by env when its sibling sidecar travels by argv
  SpawnSessionWatcher hands scopeKey positionally on argv via
  sessionwatch.CommandArgs (osruntime.go:348); SessionEnv instead renders
  KEY=value for the child. The env choice is defensible (contextcmd reads
  os.Getenv, and last-duplicate-wins lets the contract override a stale
  inherited PAIR_SCOPE_KEY), but the rationale is unwritten next to the
  divergent precedent. One sentence in Core concepts.

## Open findings

- **PQ-1** [Minor] `silent-degradation-untested` optionsFromCLI is extracted from an untested RunCLI; its refusal path stays unpinned
- **PQ-2** [Minor] `sidecar-handoff-shape` Plan does not say why the poller's contract travels by env when its sibling sidecar travels by argv
