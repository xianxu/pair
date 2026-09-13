---
gate: plan-quality
issue: 184
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-13T12:39:17-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Keep fresh-switch preference writes inside the declared registration authority
          detail: 'ARCH-DRY / ARCH-ORDER: ApplyCouchLaunchProfile sets AgentArgsExplicit (cmd/internal/launcher/launch_args_policy.go:151), so a fresh non-resume launch reaches startAgentDefaultPersistence (cmd/internal/launcher/createflow.go:494), which writes repository defaults on readiness independently of Couch registration (:643–648). Explicitly suppress that writer for fresh Couch launches while preserving required readiness/nonce setup, and name a failure-injection strategy proving unsuccessful registration changes neither preference store.'
          family: preference-write-authority
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace enumerated test cases with named function strategies
          detail: Tasks 1–6 contain extensive prose case inventories contrary to this gate's explicit requirement. Compress them into one adversarial-input-class and mechanical-guard line per risky function, including ParseLaunchParameters/FormatLaunchParameters, DecodeAgentCommand, ValidateFreshAgentArgs, BuildPrompt, AdvanceDelivery, and ReduceSwitchForm; retain production-boundary acceptance and scheduler strategies without duplicating their eventual executable cases.
          family: plan-test-strategy-compression
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-13T12:41:14-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Fresh Couch launches suppress repository-default persistence while retaining nonce/readiness setup; writer audits and registration-failure injection enforce successful registration as the preference-write authority.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Tasks 1–6 now use named function strategies with adversarial inputs and mechanical guards, retaining production-boundary acceptance and scheduler coverage.
          round: 2
      blocked: false
content_hash: a37228097c18d9d2a7ae0654e6d8ce5de4fa994d3ee984c5b015f8bb8d250587
---

# Gate ledger — pair#184 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-13T12:39:17-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `preference-write-authority` Keep fresh-switch preference writes inside the declared registration authority
  ARCH-DRY / ARCH-ORDER: ApplyCouchLaunchProfile sets AgentArgsExplicit (cmd/internal/launcher/launch_args_policy.go:151), so a fresh non-resume launch reaches startAgentDefaultPersistence (cmd/internal/launcher/createflow.go:494), which writes repository defaults on readiness independently of Couch registration (:643–648). Explicitly suppress that writer for fresh Couch launches while preserving required readiness/nonce setup, and name a failure-injection strategy proving unsuccessful registration changes neither preference store.
- **PQ-2** [Important] `plan-test-strategy-compression` Replace enumerated test cases with named function strategies
  Tasks 1–6 contain extensive prose case inventories contrary to this gate's explicit requirement. Compress them into one adversarial-input-class and mechanical-guard line per risky function, including ParseLaunchParameters/FormatLaunchParameters, DecodeAgentCommand, ValidateFreshAgentArgs, BuildPrompt, AdvanceDelivery, and ReduceSwitchForm; retain production-boundary acceptance and scheduler strategies without duplicating their eventual executable cases.

## Round 2 — 2026-09-13T12:41:14-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Fresh Couch launches suppress repository-default persistence while retaining nonce/readiness setup; writer audits and registration-failure injection enforce successful registration as the preference-write authority.
- PQ-2 — addressed — Tasks 1–6 now use named function strategies with adversarial inputs and mechanical guards, retaining production-boundary acceptance and scheduler coverage.

## Open findings

(none — every finding has been disposed)
