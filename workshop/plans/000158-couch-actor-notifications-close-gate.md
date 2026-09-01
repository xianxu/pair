---
gate: boundary-review
issue: 158
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-31T23:37:57-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Core concepts table names a nonexistent replay entity in an unchanged file
          detail: The plan claims `ptychild.ReplayWindow` lives in modified `cmd/internal/ptychild/replay.go`, but no such entity exists and that file is unchanged. The delivered replay-safe cutoff and notification-span behavior lives across `Screen` and `Child`; append a plan revision and correct the table and description.
          family: plan-code-traceability
          round: 1
        - id: BR-2
          severity: Important
          title: Nonblocking notification emission silently accepts short TTY writes
          detail: '`OSRuntime.WriteNonblocking` discards the count returned by `unix.Write`, so a partial canonical envelope with nil error is treated as successful. Check for `n != len(p)` and add a fake-runtime regression that fails without the fix (`ARCH-MOCK`).'
          family: external-write-completeness
          round: 1
        - id: BR-3
          severity: Important
          title: README omits the new pair notify command
          detail: The dispatcher adds the user-invokable `pair notify` surface and legacy option behavior, but README.md is unchanged. Document its invocation, canonical OSC behavior, and tolerant hook failure semantics.
          family: user-facing-surface-documentation
          round: 1
      blocked: true
---

# Gate ledger — pair#158 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-31T23:37:57-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `plan-code-traceability` Core concepts table names a nonexistent replay entity in an unchanged file
  The plan claims `ptychild.ReplayWindow` lives in modified `cmd/internal/ptychild/replay.go`, but no such entity exists and that file is unchanged. The delivered replay-safe cutoff and notification-span behavior lives across `Screen` and `Child`; append a plan revision and correct the table and description.
- **BR-2** [Important] `external-write-completeness` Nonblocking notification emission silently accepts short TTY writes
  `OSRuntime.WriteNonblocking` discards the count returned by `unix.Write`, so a partial canonical envelope with nil error is treated as successful. Check for `n != len(p)` and add a fake-runtime regression that fails without the fix (`ARCH-MOCK`).
- **BR-3** [Important] `user-facing-surface-documentation` README omits the new pair notify command
  The dispatcher adds the user-invokable `pair notify` surface and legacy option behavior, but README.md is unchanged. Document its invocation, canonical OSC behavior, and tolerant hook failure semantics.

## Open findings

- **BR-1** [Critical] `plan-code-traceability` Core concepts table names a nonexistent replay entity in an unchanged file
- **BR-2** [Important] `external-write-completeness` Nonblocking notification emission silently accepts short TTY writes
- **BR-3** [Important] `user-facing-surface-documentation` README omits the new pair notify command
