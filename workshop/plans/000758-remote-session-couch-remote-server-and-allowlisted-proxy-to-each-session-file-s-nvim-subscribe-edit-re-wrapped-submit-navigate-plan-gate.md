---
gate: plan-quality
issue: 758
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 9d00d1b4-ec26-444e-bc6e-842651dfd5f4
            repository: github.com/xianxu/pair
            issue_id: "000758"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot6/pair
            reviewed_head: 07a2dc0a96bcb9b1a6af1f8b4f917e746281fd22
            slot: pair:6
            requested_at: "2026-10-22T12:11:05.435905Z"
          phase: finished
          pid: 48313
          started_at: "2026-10-22T12:11:05.567513Z"
          finished_at: "2026-10-22T12:12:00.879226Z"
          session_id: 01a16385-a739-7d02-bb3c-91b494692f62
          outcome: returned
      "n": 1
      timestamp: "2026-10-22T05:12:01-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: PQ-1
          severity: Important
          title: Name an error-preserving per-PID seam for positive-death cleanup
          detail: The plan selects procutil.Source for EndpointProcessSource while requiring positive-death collection without host-table scans. Source.Identity discards the native error at cmd/internal/procutil/source.go:46, so it cannot distinguish confirmed death from unavailable inspection; its snapshot methods enumerate the host table. Specify the per-PID observation API, typed alive/replaced/dead/unknown evidence and platform classification that production and the stateful fake share. This is necessary to execute the ARCH-ORDER and ARCH-FUNERAL cleanup contract without retaining dead evidence indefinitely or treating uncertainty as death.
          family: preserve-process-observation-evidence
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace prose test-case inventories with named functions and mechanical strategies
          detail: 'The Tasks section at workshop/plans/000758-editor-endpoints-plan.md:141-147 enumerates cases but does not name the functions to unit-test; even the pure transition function and codec operations remain unnamed. Compress these inventories into one strategy line per risky function, identifying adversarial input classes and mechanical guards: generated event sequences with independent authority invariants, fuzzed strict decoding and lossless unknown-field round trips, and controllable IO ordering with publication/count bounds. Retain the native-conformance and integration scope separately.'
          family: function-level-test-strategy
          round: 1
      blocked: true
---

# Gate ledger — pair#758 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-22T05:12:01-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `preserve-process-observation-evidence` Name an error-preserving per-PID seam for positive-death cleanup
  The plan selects procutil.Source for EndpointProcessSource while requiring positive-death collection without host-table scans. Source.Identity discards the native error at cmd/internal/procutil/source.go:46, so it cannot distinguish confirmed death from unavailable inspection; its snapshot methods enumerate the host table. Specify the per-PID observation API, typed alive/replaced/dead/unknown evidence and platform classification that production and the stateful fake share. This is necessary to execute the ARCH-ORDER and ARCH-FUNERAL cleanup contract without retaining dead evidence indefinitely or treating uncertainty as death.
- **PQ-2** [Important] `function-level-test-strategy` Replace prose test-case inventories with named functions and mechanical strategies
  The Tasks section at workshop/plans/000758-editor-endpoints-plan.md:141-147 enumerates cases but does not name the functions to unit-test; even the pure transition function and codec operations remain unnamed. Compress these inventories into one strategy line per risky function, identifying adversarial input classes and mechanical guards: generated event sequences with independent authority invariants, fuzzed strict decoding and lossless unknown-field round trips, and controllable IO ordering with publication/count bounds. Retain the native-conformance and integration scope separately.

## Open findings

- **PQ-1** [Important] `preserve-process-observation-evidence` Name an error-preserving per-PID seam for positive-death cleanup
- **PQ-2** [Important] `function-level-test-strategy` Replace prose test-case inventories with named functions and mechanical strategies
