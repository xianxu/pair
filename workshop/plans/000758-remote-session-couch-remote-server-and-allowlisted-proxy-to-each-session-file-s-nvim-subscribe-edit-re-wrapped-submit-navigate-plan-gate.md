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
    - reviewer_runs:
        - version: 1
          intent:
            id: de453692-7c89-4863-834a-a62d5c434387
            repository: github.com/xianxu/pair
            issue_id: "000758"
            recipe: plan-quality
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot6/pair
            reviewed_head: bc84e05d20c5a00acfb56d8ee03c64f739320b86
            slot: pair:6
            requested_at: "2026-10-22T12:13:48.695346Z"
          phase: finished
          pid: 31810
          started_at: "2026-10-22T12:13:48.783227Z"
          finished_at: "2026-10-22T12:14:17.83968Z"
          session_id: 01a16388-246e-7f63-8a56-8030faac80db
          outcome: returned
      "n": 2
      timestamp: "2026-10-22T05:14:18-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Source.ReadIdentity preserves native errors; ClassifyEndpointProcess specifies shared positive-death and uncertainty semantics without host-table scans.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Named risky functions now have mechanical adversarial strategies; native conformance and integration qualification remain separate.
          round: 2
      blocked: false
content_hash: 64d521378942ea17b7411cf1cffb50d95514712033e76a90ece764cd46cdb334
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

## Round 2 — 2026-10-22T05:14:18-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Source.ReadIdentity preserves native errors; ClassifyEndpointProcess specifies shared positive-death and uncertainty semantics without host-table scans.
- PQ-2 — addressed — Named risky functions now have mechanical adversarial strategies; native conformance and integration qualification remain separate.

## Open findings

(none — every finding has been disposed)
