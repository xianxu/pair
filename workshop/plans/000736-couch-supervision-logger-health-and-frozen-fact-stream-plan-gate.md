---
gate: plan-quality
issue: 736
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 75bde4c6-9a33-4629-91dc-5005ae1d42fa
            repository: github.com/xianxu/pair
            issue_id: "000736"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot7/pair
            reviewed_head: 42d27381690baf5a334265abf982b04ca9adb02e
            slot: pair:7
            requested_at: "2026-10-21T03:07:08.256407Z"
          phase: finished
          pid: 62843
          started_at: "2026-10-21T03:07:08.442253Z"
          finished_at: "2026-10-21T03:08:21.625909Z"
          session_id: 01a15c6d-4acf-75e3-8312-0e8f19cb96ed
          outcome: returned
      "n": 1
      timestamp: "2026-10-20T20:08:22-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: PQ-1
          severity: Important
          title: Resolve the false claim that older clients ignore additive response fields
          detail: 'ARCH-EVOLVE: The plan promises old-client compatibility, but couchmessage.Call uses strictjson.Decode, whose DisallowUnknownFields rejects the new health member (transport.go:293; strictjson/decode.go:24). Specify an executable mixed-version policy, such as negotiated publication or an explicit coordinated-upgrade requirement, and verify it against the existing strict reader; changing only the new client cannot repair already-running old clients.'
          family: wire-compatibility-contract
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace test-case inventories with named-function strategies
          detail: M1 steps 1 and 4 enumerate cases instead of providing the required production-function test strategies. Name factLogLifecycle.advance, the health snapshot/error-capture functions, messageService.handle, projectResourceView and renderResources as applicable, with one compact adversarial-input and mechanical-guard strategy per risky function; keep detailed cases in executable tests.
          family: function-test-strategy
          round: 1
      blocked: true
---

# Gate ledger — pair#736 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T20:08:22-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `wire-compatibility-contract` Resolve the false claim that older clients ignore additive response fields
  ARCH-EVOLVE: The plan promises old-client compatibility, but couchmessage.Call uses strictjson.Decode, whose DisallowUnknownFields rejects the new health member (transport.go:293; strictjson/decode.go:24). Specify an executable mixed-version policy, such as negotiated publication or an explicit coordinated-upgrade requirement, and verify it against the existing strict reader; changing only the new client cannot repair already-running old clients.
- **PQ-2** [Important] `function-test-strategy` Replace test-case inventories with named-function strategies
  M1 steps 1 and 4 enumerate cases instead of providing the required production-function test strategies. Name factLogLifecycle.advance, the health snapshot/error-capture functions, messageService.handle, projectResourceView and renderResources as applicable, with one compact adversarial-input and mechanical-guard strategy per risky function; keep detailed cases in executable tests.

## Open findings

- **PQ-1** [Important] `wire-compatibility-contract` Resolve the false claim that older clients ignore additive response fields
- **PQ-2** [Important] `function-test-strategy` Replace test-case inventories with named-function strategies
