---
gate: boundary-review
issue: 731
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 9800e770-407a-4376-a76b-a7b6cf143fb3
            repository: github.com/xianxu/pair
            issue_id: "000731"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/ba22136acb4c22f6
            reviewed_head: a596629fca8c9283755a3d5ac9ed289e3f222b87
            slot: pair:6
            requested_at: "2026-10-21T04:07:04.742808Z"
          phase: finished
          pid: 85826
          started_at: "2026-10-21T04:07:04.867544Z"
          finished_at: "2026-10-21T04:09:06.377737Z"
          session_id: 01a15ca4-2ab9-7e80-b38e-79dc727bdd53
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: a596629fca8c9283755a3d5ac9ed289e3f222b87
        tracker_head: a5e16637f11beadfef44c626fd594baffa32997a
        branch_point: 49566452104f129c38c8db5e4e985a0a5ae5087e
        since: "2026-10-20T20:32:00-07:00"
        until: "2026-10-20T21:06:44-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T20:32:00-07:00"
              end: "2026-10-20T21:06:44-07:00"
              began: "2026-10-20T20:32:00-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: 8a81845ff4e2daee42d59c583675f5cb5ff8d6df8388b81020c0394ec985648e
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 70640297
              windows:
                - start: "2026-10-20T20:32:00-07:00"
                  end: "2026-10-20T21:06:44-07:00"
            - harness: codex
              id: 01a15c96-3f91-7301-bd36-aff56db004d4
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 58ba196d37dd4eac35327b3d22351ac1740992def0366b5f1ab1bfa42b5fd99e
              started: "2026-10-21T03:51:53.145Z"
              spawned: true
              bytes: 567936
              windows:
                - start: "2026-10-21T03:51:53.145Z"
                  end: "2026-10-20T21:06:44-07:00"
            - harness: codex
              id: 01a15c96-7553-7931-bc6e-f0d533f98ad6
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: 537d12a2f020ea6a6fcac4c64a44a186a0d7a03443e03db36fd53ce7f4ed5d21
              started: "2026-10-21T03:52:06.916Z"
              spawned: true
              bytes: 446295
              windows:
                - start: "2026-10-21T03:52:06.916Z"
                  end: "2026-10-20T21:06:44-07:00"
            - harness: codex
              id: 01a15c97-53d1-79a2-a7d9-2966d169f1bc
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              parent_id: 01a14b2c-9890-74f3-a738-58879659bef5
              digest: fce406d0c93449ebac41d8ae994efa6427872894f3c8201e54c214bb78649005
              started: "2026-10-21T03:53:03.877Z"
              spawned: true
              bytes: 758070
              windows:
                - start: "2026-10-21T03:53:03.877Z"
                  end: "2026-10-20T21:06:44-07:00"
      "n": 1
      timestamp: "2026-10-20T21:09:07-07:00"
      dispatched: "2026-10-20T21:07:02-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: Done-when still promises the superseded unconditional ownership policy
          detail: workshop/issues/000731-a-pane-that-fell-back-to-a-native-shell-still-classifies-as-a-pair-terminal.md:44 requires LiveIDs and RoleForPaneWith to stop granting ownership after fallback. LiveIDs remains an injected-predicate storage reader, and terminal_owner.go:56 implements LiveOwnerIDs with the approved unavailable-inspection exception documented at issue line 68. Update Done when to name LiveOwnerIDs, require rejection of confirmed shell argv or identity changes, and explicitly cover retained ownership plus bounded diagnostics when inspection is unavailable. This is an ARCH-PURPOSE traceability gap.
          family: acceptance-criteria-contract-alignment
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#731 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T21:09:07-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `acceptance-criteria-contract-alignment` (kind: defect) Done-when still promises the superseded unconditional ownership policy
  workshop/issues/000731-a-pane-that-fell-back-to-a-native-shell-still-classifies-as-a-pair-terminal.md:44 requires LiveIDs and RoleForPaneWith to stop granting ownership after fallback. LiveIDs remains an injected-predicate storage reader, and terminal_owner.go:56 implements LiveOwnerIDs with the approved unavailable-inspection exception documented at issue line 68. Update Done when to name LiveOwnerIDs, require rejection of confirmed shell argv or identity changes, and explicitly cover retained ownership plus bounded diagnostics when inspection is unavailable. This is an ARCH-PURPOSE traceability gap.

## Open findings

- **BR-1** [Important] `acceptance-criteria-contract-alignment` (kind: defect) Done-when still promises the superseded unconditional ownership policy
