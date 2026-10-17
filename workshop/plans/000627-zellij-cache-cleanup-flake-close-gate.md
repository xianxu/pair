---
gate: boundary-review
issue: 627
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 59343053-62c8-43e9-8565-0f9b699e17fb
            repository: github.com/xianxu/pair
            issue_id: "000627"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/77a3b7eaa394c528
            reviewed_head: 73e3445407d4f2effedb1583a4eeab86cbd99d46
            slot: pair:1
            requested_at: "2026-10-17T03:00:32.852243Z"
          phase: finished
          pid: 97754
          started_at: "2026-10-17T03:00:32.959925Z"
          finished_at: "2026-10-17T03:03:17.868765Z"
          session_id: 01a147cd-d12f-76e3-98c6-7b91191aa522
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 73e3445407d4f2effedb1583a4eeab86cbd99d46
        tracker_head: 811aece5c5f6af48996d3797aa3b6398864ca63b
        branch_point: 9e5c1bd33ac31606f68fdc5bc3c1f0485d1d98e0
        since: "2026-10-16T18:58:29-07:00"
        until: "2026-10-16T20:00:08-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T18:58:29-07:00"
              end: "2026-10-16T20:00:08-07:00"
              began: "2026-10-16T18:58:29-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a14367-3aab-7a22-b89b-0908f54a9846
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 92783d38c1776cd451de24c5855c1d29193c09a04f3338f5e347ed6fc6fe9667
              started: "2026-10-16T06:30:01.402Z"
              spawned: false
              bytes: 23259135
              windows:
                - start: "2026-10-16T18:58:29-07:00"
                  end: "2026-10-16T20:00:08-07:00"
      "n": 1
      timestamp: "2026-10-16T20:03:24-07:00"
      dispatched: "2026-10-16T20:00:31-07:00"
      agent: codex
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Linux process-read failures can be mistaken for confirmed daemon exit
          detail: 'cmd/internal/pairlifecycletest/live_zellij.go:194-206 treats an omitted PID as terminal evidence, but cmd/internal/procutil/table_linux.go:23-29 silently omits every stat-read or parse failure, including failures that do not establish disappearance. The same family has two instances in this window: the initial SnapshotNamed capture at live_zellij.go:145 can omit the daemon and leave servers empty, and the shutdown poll at :194 can falsely declare a captured daemon absent. Both paths can authorize cleanup while the writer remains alive, violating the Spec''s explicit unknown-read contract (ARCH-PURPOSE). Preserve unreadable PID rows as unknown or propagate non-disappearance failures, and add regression coverage for stat-read and parse failures at both consumers.'
          family: unknown-process-read-is-not-exit
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#627 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-16T20:03:24-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `unknown-process-read-is-not-exit` (kind: defect) Linux process-read failures can be mistaken for confirmed daemon exit
  cmd/internal/pairlifecycletest/live_zellij.go:194-206 treats an omitted PID as terminal evidence, but cmd/internal/procutil/table_linux.go:23-29 silently omits every stat-read or parse failure, including failures that do not establish disappearance. The same family has two instances in this window: the initial SnapshotNamed capture at live_zellij.go:145 can omit the daemon and leave servers empty, and the shutdown poll at :194 can falsely declare a captured daemon absent. Both paths can authorize cleanup while the writer remains alive, violating the Spec's explicit unknown-read contract (ARCH-PURPOSE). Preserve unreadable PID rows as unknown or propagate non-disappearance failures, and add regression coverage for stat-read and parse failures at both consumers.

## Open findings

- **BR-1** [Critical] `unknown-process-read-is-not-exit` (kind: defect) Linux process-read failures can be mistaken for confirmed daemon exit
