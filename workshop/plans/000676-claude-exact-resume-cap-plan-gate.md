---
gate: plan-quality
issue: 676
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 5a58f493-0c19-4f9e-b8ed-b9bb05339491
            repository: github.com/xianxu/pair
            issue_id: "000676"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot5/pair-claude-resume
            reviewed_head: 8c7d5d101d109618c32e736ea2c5058ba37304f3
            slot: pair:5
            requested_at: "2026-10-18T22:16:31.617709Z"
          phase: finished
          pid: 18393
          started_at: "2026-10-18T22:16:31.777075Z"
          finished_at: "2026-10-18T22:17:18.718557Z"
          session_id: 01a15116-82cb-7d90-874f-f68c20850237
          outcome: returned
      "n": 1
      timestamp: "2026-10-18T15:17:19-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: PQ-1
          severity: Important
          title: Replace the case inventory with function-specific test strategies
          detail: Task 3 enumerates cases without naming the production functions tested or a mechanical adversarial guard per risky function. Name VisitNativeSelectorFiles, nativeSelectorRuntime.ListFiles and ResolveNativeSelector as applicable; specify counted production IO for growth and controllable filesystem-operation ordering for replacement/absence races. The existing mutable fallback fake does not exercise the new capability (ARCH-MOCK, ARCH-ORDER).
          family: function-specific-adversarial-test-strategy
          round: 1
        - id: PQ-2
          severity: Important
          title: Specify the directory proof rather than relying on regular-file helpers
          detail: Task 2 proposes reusing sameRecoveryInfo, which rejects directories, while openRecoveryFile reopens the root and closes intermediate descriptors. Define the directory-specific comparison and descriptor ownership that implement the promised root/project stability checks, including detection of entry changes that could invalidate absence or hide a conflicting target (ARCH-DRY, ARCH-SECURE).
          family: directory-evidence-stability-contract
          round: 1
      blocked: true
---

# Gate ledger — pair-claude-resume#676 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-18T15:17:19-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `function-specific-adversarial-test-strategy` Replace the case inventory with function-specific test strategies
  Task 3 enumerates cases without naming the production functions tested or a mechanical adversarial guard per risky function. Name VisitNativeSelectorFiles, nativeSelectorRuntime.ListFiles and ResolveNativeSelector as applicable; specify counted production IO for growth and controllable filesystem-operation ordering for replacement/absence races. The existing mutable fallback fake does not exercise the new capability (ARCH-MOCK, ARCH-ORDER).
- **PQ-2** [Important] `directory-evidence-stability-contract` Specify the directory proof rather than relying on regular-file helpers
  Task 2 proposes reusing sameRecoveryInfo, which rejects directories, while openRecoveryFile reopens the root and closes intermediate descriptors. Define the directory-specific comparison and descriptor ownership that implement the promised root/project stability checks, including detection of entry changes that could invalidate absence or hide a conflicting target (ARCH-DRY, ARCH-SECURE).

## Open findings

- **PQ-1** [Important] `function-specific-adversarial-test-strategy` Replace the case inventory with function-specific test strategies
- **PQ-2** [Important] `directory-evidence-stability-contract` Specify the directory proof rather than relying on regular-file helpers
