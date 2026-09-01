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
    - "n": 2
      timestamp: "2026-08-31T23:44:46-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: The plan now names the delivered Screen replay-safe offset and Child replay-span entities, with an appended revision explaining the ownership change.
          round: 2
        - id: BR-2
          disposition: addressed
          note: OSRuntime rejects n != len(p), and TestOSRuntimeReportsShortNonblockingWrite directly fails against the prior implementation.
          round: 2
        - id: BR-3
          disposition: addressed
          note: README now documents pair notify, canonical OSC 777 output, legacy options, and tolerant hook failure behavior.
          round: 2
      findings:
        - id: BR-4
          severity: Critical
          title: Core concepts table still marks unchanged generated and index files as modified
          detail: 'workshop/plans/000158-couch-actor-notifications-plan.md:74 claims the runtime pair-notify mirror, runtime manifest, and atlas/index.md are modified, but none appears in the pinned range; the runtime assets are ignored generated files rather than tracked changes. This is the 2nd finding in family plan-code-traceability. Do not patch only this row: state and apply the rule that every Core concepts path/status must resolve against the committed boundary diff, separating tracked modified surfaces from unchanged indexes and derived ignored outputs.'
          family: plan-code-traceability
          round: 2
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

## Round 2 — 2026-08-31T23:44:46-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The plan now names the delivered Screen replay-safe offset and Child replay-span entities, with an appended revision explaining the ownership change.
- BR-2 — addressed — OSRuntime rejects n != len(p), and TestOSRuntimeReportsShortNonblockingWrite directly fails against the prior implementation.
- BR-3 — addressed — README now documents pair notify, canonical OSC 777 output, legacy options, and tolerant hook failure behavior.

### Raised

- **BR-4** [Critical] `plan-code-traceability` Core concepts table still marks unchanged generated and index files as modified
  workshop/plans/000158-couch-actor-notifications-plan.md:74 claims the runtime pair-notify mirror, runtime manifest, and atlas/index.md are modified, but none appears in the pinned range; the runtime assets are ignored generated files rather than tracked changes. This is the 2nd finding in family plan-code-traceability. Do not patch only this row: state and apply the rule that every Core concepts path/status must resolve against the committed boundary diff, separating tracked modified surfaces from unchanged indexes and derived ignored outputs.

## Open findings

- **BR-4** [Critical] `plan-code-traceability` Core concepts table still marks unchanged generated and index files as modified
