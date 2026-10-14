---
gate: boundary-review
issue: 547
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-14T01:00:28-07:00"
      dispatched: "2026-10-14T00:57:23-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Important
          title: Reparented test helper leaves descendants running after timeout
          detail: tests/without-ancestors.py:36 kills only the immediate subprocess on timeout; the worker reports failure and the supervisor cancels its group-cleanup alarm at line 63. A controlled reproduction returned after approximately 0.37 seconds while its sleep grandchild remained alive. ARCH-ORDER and ARCH-FUNERAL require cleanup of the owned process group on timeout and supervisor cancellation, with regression tests proving descendants terminate.
          family: owned-process-lifetime
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-14T01:10:41-07:00"
      dispatched: "2026-10-14T01:06:58-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: tests/without-ancestors.py:30-106 establishes group ownership and a startup handshake, then kills the group on failure or cancellation. All five lifecycle tests pass at head. Against the pre-fix helper, the timeout regression fails with a surviving grandchild; cancellation regressions also fail. The disposable regression fixture was removed.
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: false
    - actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 5526b83153794bb1df358002477a0ff91112a288
        tracker_head: 42ec6ec8dd458c6f09662de01e0f07a2e3d65f4c
        branch_point: 895c7868e0336379c8a6aa33be79b31247999544
        since: "2026-10-14T00:00:06-07:00"
        until: "2026-10-14T08:37:11-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-14T00:00:06-07:00"
              end: "2026-10-14T08:37:11-07:00"
              began: "2026-10-14T00:00:06-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot5/pair
        sources:
            - harness: codex
              id: 01a138a2-9eab-76d0-860a-978e5ce8d556
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: c6ef52bf833f8e3accf741925cb82e2280230fcd4de531adbda8dfc0df2112b2
              started: "2026-10-14T04:19:04.475Z"
              spawned: false
              bytes: 4419573
              windows:
                - start: "2026-10-14T00:00:06-07:00"
                  end: "2026-10-14T08:37:11-07:00"
            - harness: codex
              id: 01a13937-d7ad-7a73-afc0-c0d4a9507608
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a138a2-9eab-76d0-860a-978e5ce8d556
              digest: 013c09fa7764ee6dba884e8622d60a0cbd487f4edd85d5bf55735970e3fb4ac2
              started: "2026-10-14T07:02:03.488Z"
              spawned: true
              bytes: 521173
              windows:
                - start: "2026-10-14T07:02:03.488Z"
                  end: "2026-10-14T08:37:11-07:00"
            - harness: codex
              id: 01a1393b-3446-70d0-82a5-69734a332945
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: b0a226791d58b6d895dcdf887cc0d23f3a3afc10aa62050252880e5b38928208
              started: "2026-10-14T07:05:44.034Z"
              spawned: false
              bytes: 196805
              windows:
                - start: "2026-10-14T07:05:44.034Z"
                  end: "2026-10-14T08:37:11-07:00"
            - harness: codex
              id: 01a1393d-1cda-7f03-b906-4ee7f709fb8b
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: b5a569169aca6f179d2d2178827de542f630c0b48852c70e15207cfb39ec54cc
              started: "2026-10-14T07:07:48.957Z"
              spawned: false
              bytes: 459480
              windows:
                - start: "2026-10-14T07:07:48.957Z"
                  end: "2026-10-14T08:37:11-07:00"
            - harness: codex
              id: 01a1393f-d4d6-7f82-9b67-8e961d415089
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 5f53316bc8fd2d4ef076ba3b61b195371d3e1702a90f7e18f66cb648d949a376
              started: "2026-10-14T07:10:47.209Z"
              spawned: false
              bytes: 412778
              windows:
                - start: "2026-10-14T07:10:47.209Z"
                  end: "2026-10-14T08:37:11-07:00"
            - harness: codex
              id: 01a13941-c33d-7ef1-b670-d4ec25b813fb
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 0828d8eede3927de552ce5bb95c9582d70e8446091e2b20987191a4c0605f9b5
              started: "2026-10-14T07:12:55.569Z"
              spawned: false
              bytes: 413234
              windows:
                - start: "2026-10-14T07:12:55.569Z"
                  end: "2026-10-14T08:37:11-07:00"
            - harness: codex
              id: 01a13942-61d5-73c1-afff-edfe55ef44d5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: b4801f5b6c1a3682466fcb9476adc34d0b13bdd700927e55b09be64283089dbb
              started: "2026-10-14T07:13:34.37Z"
              spawned: false
              bytes: 356693
              windows:
                - start: "2026-10-14T07:13:34.37Z"
                  end: "2026-10-14T08:37:11-07:00"
            - harness: codex
              id: 01a13943-f8d8-7bc0-8abc-82b4fba500b0
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a138a2-9eab-76d0-860a-978e5ce8d556
              digest: 1f1f25e95d0b42057a67ec5a3589042b454f84656347b92482b3163c2a372a66
              started: "2026-10-14T07:15:18.406Z"
              spawned: true
              bytes: 614214
              windows:
                - start: "2026-10-14T07:15:18.406Z"
                  end: "2026-10-14T08:37:11-07:00"
            - harness: codex
              id: 01a1396e-25c5-70f2-8c98-e576833ab08e
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a138a2-9eab-76d0-860a-978e5ce8d556
              digest: d7170e38e0f10f74206f541c760e4d584c183ab149fe9ab47c1907f13326787f
              started: "2026-10-14T08:01:22.521Z"
              spawned: true
              bytes: 450009
              windows:
                - start: "2026-10-14T08:01:22.521Z"
                  end: "2026-10-14T08:37:11-07:00"
        cuts:
            - dispatched: "2026-10-14T00:57:23-07:00"
              finalized: "2026-10-14T01:00:28-07:00"
              finalizing: false
            - dispatched: "2026-10-14T01:06:58-07:00"
              finalized: "2026-10-14T01:10:41-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-14T08:40:42-07:00"
      dispatched: "2026-10-14T08:37:24-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: All five lifecycle tests pass at head. In a temporary fixture using the pre-fix helper, the timeout test fails with a surviving grandchild and both cancellation cases fail. The temporary fixture was removed.
          round: 3
      findings:
        - kind: defect
          id: BR-2
          severity: Important
          title: Zombie fixture returns before the child reaches terminal kernel state
          detail: 'cmd/internal/procutil/zombie_test.go:35-38 treats any nativeCommand error as completion. The focused package run failed TestDarwinUnreapedChildStatus: command inspection returned EINVAL, but P_stat was still 2 rather than SZOMB=5. ARCH-ORDER requires synchronization on the state being asserted: wait with a bounded deadline for native terminal metadata before returning the fixture, and verify both terminal-process tests under repetition and concurrent package load.'
          family: process-test-state-synchronization
          round: 3
        - kind: unverifiable
          id: BR-3
          severity: Important
          title: Required seatbelt acceptance cannot be verified in this review environment
          detail: 'Running sandbox-exec -p ''(version 1)(allow default)'' /tmp/pair547-review-procutil.test with the selected native process probes exited 71 with “sandbox_apply: Operation not permitted”; no test executed under that profile. Direct native probes pass, but /bin/ps also executes here, so those results do not establish the required setuid-denying environment. Provide measured host results from binaries built at the reviewed head for the native probes and launcher process acceptance under the required profile, or an environment permitting that verification.'
          family: sandbox-acceptance-evidence
          round: 3
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#547 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-14T01:00:28-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `owned-process-lifetime` Reparented test helper leaves descendants running after timeout
  tests/without-ancestors.py:36 kills only the immediate subprocess on timeout; the worker reports failure and the supervisor cancels its group-cleanup alarm at line 63. A controlled reproduction returned after approximately 0.37 seconds while its sleep grandchild remained alive. ARCH-ORDER and ARCH-FUNERAL require cleanup of the owned process group on timeout and supervisor cancellation, with regression tests proving descendants terminate.

## Round 2 — 2026-10-14T01:10:41-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — tests/without-ancestors.py:30-106 establishes group ownership and a startup handshake, then kills the group on failure or cancellation. All five lifecycle tests pass at head. Against the pre-fix helper, the timeout regression fails with a surviving grandchild; cancellation regressions also fail. The disposable regression fixture was removed.

## Round 3 — 2026-10-14T08:40:42-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — All five lifecycle tests pass at head. In a temporary fixture using the pre-fix helper, the timeout test fails with a surviving grandchild and both cancellation cases fail. The temporary fixture was removed.

### Raised

- **BR-2** [Important] `process-test-state-synchronization` (kind: defect) Zombie fixture returns before the child reaches terminal kernel state
  cmd/internal/procutil/zombie_test.go:35-38 treats any nativeCommand error as completion. The focused package run failed TestDarwinUnreapedChildStatus: command inspection returned EINVAL, but P_stat was still 2 rather than SZOMB=5. ARCH-ORDER requires synchronization on the state being asserted: wait with a bounded deadline for native terminal metadata before returning the fixture, and verify both terminal-process tests under repetition and concurrent package load.
- **BR-3** [Important] `sandbox-acceptance-evidence` (kind: unverifiable) Required seatbelt acceptance cannot be verified in this review environment
  Running sandbox-exec -p '(version 1)(allow default)' /tmp/pair547-review-procutil.test with the selected native process probes exited 71 with “sandbox_apply: Operation not permitted”; no test executed under that profile. Direct native probes pass, but /bin/ps also executes here, so those results do not establish the required setuid-denying environment. Provide measured host results from binaries built at the reviewed head for the native probes and launcher process acceptance under the required profile, or an environment permitting that verification.

## Open findings

- **BR-2** [Important] `process-test-state-synchronization` (kind: defect) Zombie fixture returns before the child reaches terminal kernel state
- **BR-3** [Important] `sandbox-acceptance-evidence` (kind: unverifiable) Required seatbelt acceptance cannot be verified in this review environment
