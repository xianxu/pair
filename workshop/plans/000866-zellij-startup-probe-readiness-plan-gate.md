---
gate: plan-quality
issue: 866
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 631a7cbf-a6f9-4574-b4b2-44fa5ebcfece
            repository: github.com/xianxu/pair
            issue_id: "000866"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/workspace/worktree/pair-slot5/pair
            reviewed_head: 29771893bc4f64d59413b2e6bd5f37a516e76f39
            slot: pair:5
            requested_at: "2026-10-24T10:22:07.828217Z"
          phase: finished
          pid: 45482
          started_at: "2026-10-24T10:22:07.972074Z"
          finished_at: "2026-10-24T10:23:18.707568Z"
          session_id: 01a16d6e-9e21-7f83-a585-6d048f32108f
          outcome: returned
      "n": 1
      timestamp: "2026-10-24T03:23:18-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: PQ-1
          severity: Important
          title: Define exclusion ownership for interactive attach and exec-replacing commands
          detail: 'ARCH-ORDER / ARCH-CONSTRAINTS: The final design applies exclusion to all globally scanning callers but defines its end as joined CLI completion. Interactive attach outlives startup admission, while ExecKillSession replaces Pair and closes the existing close-on-exec lease before the native scan. Specify the safe release witness and lock owner for each command class, including cancellation and existing per-session lock ordering. Name the affected functions and add a controlled ordering strategy proving scans stay excluded while an admitted interactive client does not block later creates for its lifetime.'
          family: exclusion-covers-native-scan-lifetime
          round: 1
        - id: PQ-2
          severity: Minor
          title: Make marker compatibility verification explicit
          detail: 'ARCH-EVOLVE: The final revision promises additive strict-typed marker decoding but does not name how unknown members, including nested identity members, survive any rewrite. State whether markers have read-modify-write paths; where they do, name the preservation mechanism and an older-reader/newer-record round-trip strategy.'
          family: persisted-unknown-field-preservation
          round: 1
      blocked: true
---

# Gate ledger — pair#866 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-24T03:23:18-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `exclusion-covers-native-scan-lifetime` Define exclusion ownership for interactive attach and exec-replacing commands
  ARCH-ORDER / ARCH-CONSTRAINTS: The final design applies exclusion to all globally scanning callers but defines its end as joined CLI completion. Interactive attach outlives startup admission, while ExecKillSession replaces Pair and closes the existing close-on-exec lease before the native scan. Specify the safe release witness and lock owner for each command class, including cancellation and existing per-session lock ordering. Name the affected functions and add a controlled ordering strategy proving scans stay excluded while an admitted interactive client does not block later creates for its lifetime.
- **PQ-2** [Minor] `persisted-unknown-field-preservation` Make marker compatibility verification explicit
  ARCH-EVOLVE: The final revision promises additive strict-typed marker decoding but does not name how unknown members, including nested identity members, survive any rewrite. State whether markers have read-modify-write paths; where they do, name the preservation mechanism and an older-reader/newer-record round-trip strategy.

## Open findings

- **PQ-1** [Important] `exclusion-covers-native-scan-lifetime` Define exclusion ownership for interactive attach and exec-replacing commands
- **PQ-2** [Minor] `persisted-unknown-field-preservation` Make marker compatibility verification explicit
