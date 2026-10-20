---
gate: boundary-review
issue: 722
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: ac69fd01-0a4e-4d9c-9c1b-66e0aaaa3e78
            repository: github.com/xianxu/pair
            issue_id: "000722"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/153ebf1c46781d65
            reviewed_head: e35f03c8dfbed9e588b532bafd25dc54ab9e3aa8
            slot: pair:5
            requested_at: "2026-10-20T13:12:57.477815Z"
          phase: finished
          pid: 48497
          started_at: "2026-10-20T13:12:57.672805Z"
          finished_at: "2026-10-20T13:17:35.348601Z"
          session_id: 01a15971-93c3-7de2-a7e3-09f0e3336582
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: e35f03c8dfbed9e588b532bafd25dc54ab9e3aa8
        tracker_head: 2d351a820733b016b5bff93228db2c320236e6c2
        branch_point: 7d4e0c3aba6b39fb12d5125ee967dcde9e5be4be
        since: "2026-10-19T23:16:03-07:00"
        until: "2026-10-20T06:11:48-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T23:16:03-07:00"
              end: "2026-10-20T06:11:48-07:00"
              began: "2026-10-19T23:16:03-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot5/pair
        sources:
            - harness: codex
              id: 01a1574c-1cb1-7e92-adfe-354342364799
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 3a57617f291d6d29c0aab990e5d51a317038ff8956d860607bad770a0ce30910
              started: "2026-10-20T03:12:49.04Z"
              spawned: false
              bytes: 17687537
              windows:
                - start: "2026-10-19T23:16:03-07:00"
                  end: "2026-10-20T06:11:48-07:00"
            - harness: codex
              id: 01a157f3-f7ac-78f2-acba-b237ce0efc23
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a1574c-1cb1-7e92-adfe-354342364799
              digest: 45aeb07250b6a5e19ede5d9c18cf57c7617545d286dca8dc2cf7354b69600b8d
              started: "2026-10-20T06:16:08.957Z"
              spawned: true
              bytes: 682423
              windows:
                - start: "2026-10-20T06:16:08.957Z"
                  end: "2026-10-20T06:11:48-07:00"
            - harness: codex
              id: 01a157fa-d67a-7163-8d4a-4e23922ea04b
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a1574c-1cb1-7e92-adfe-354342364799
              digest: bea9808a30b41948c658eac64f2b7b2ffee6c305ed31f4b1586242847428e520
              started: "2026-10-20T06:23:39.29Z"
              spawned: true
              bytes: 389828
              windows:
                - start: "2026-10-20T06:23:39.29Z"
                  end: "2026-10-20T06:11:48-07:00"
            - harness: codex
              id: 01a157fd-beea-77c0-b715-f6ef06489a73
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: b34830b58dbebc6a84742988f995abf18757a4c76c6757b4a69cb0652cae3043
              started: "2026-10-20T06:26:50.351Z"
              spawned: false
              bytes: 657147
              windows:
                - start: "2026-10-20T06:26:50.351Z"
                  end: "2026-10-20T06:11:48-07:00"
            - harness: codex
              id: 01a157ff-9966-74e1-9f60-02c9e32b4374
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 5254611a88d85bb81aec06d486955831bed7ca24fe05e38c0c0a87f62b686b5d
              started: "2026-10-20T06:28:51.508Z"
              spawned: false
              bytes: 469926
              windows:
                - start: "2026-10-20T06:28:51.508Z"
                  end: "2026-10-20T06:11:48-07:00"
            - harness: codex
              id: 01a15800-cb97-7662-a421-e3a375c49042
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: b1a82507c714500ebd835dbbc4ae5cd684adb21ce05ca1c3dec833eb4cb2e973
              started: "2026-10-20T06:30:09.961Z"
              spawned: false
              bytes: 425176
              windows:
                - start: "2026-10-20T06:30:09.961Z"
                  end: "2026-10-20T06:11:48-07:00"
            - harness: codex
              id: 01a15801-408c-7552-ad19-252d46f54385
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: f1ef77f6c8b33449e2e52539f3f6e17676d1457a425eeb9659af63a6247b5115
              started: "2026-10-20T06:30:40.044Z"
              spawned: false
              bytes: 470596
              windows:
                - start: "2026-10-20T06:30:40.044Z"
                  end: "2026-10-20T06:11:48-07:00"
            - harness: codex
              id: 01a15802-26f6-7b30-9815-4edab86b2a53
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a1574c-1cb1-7e92-adfe-354342364799
              digest: 6f876c0ba147081a423cef9e61b1991db62db5cb913c6baa8a8e0dd327bfb74d
              started: "2026-10-20T06:31:38.563Z"
              spawned: true
              bytes: 2990604
              windows:
                - start: "2026-10-20T06:31:38.563Z"
                  end: "2026-10-20T06:11:48-07:00"
            - harness: codex
              id: 01a15802-56b5-7543-84a4-a267317bf6ca
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a1574c-1cb1-7e92-adfe-354342364799
              digest: 1ac3c69863dbb1ab5588511237dbd55a5b5d97b21bc0e0b81998334e7241d4ac
              started: "2026-10-20T06:31:50.769Z"
              spawned: true
              bytes: 2817178
              windows:
                - start: "2026-10-20T06:31:50.769Z"
                  end: "2026-10-20T06:11:48-07:00"
            - harness: codex
              id: 01a15802-81d6-79c0-9efc-05af0082b5cf
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a1574c-1cb1-7e92-adfe-354342364799
              digest: 1af0419441838788010ed5953b12ed66b3fd8a3f276606fbe3386b8cbebe41c5
              started: "2026-10-20T06:32:01.794Z"
              spawned: true
              bytes: 3047463
              windows:
                - start: "2026-10-20T06:32:01.794Z"
                  end: "2026-10-20T06:11:48-07:00"
            - harness: codex
              id: 01a1584a-4077-7770-b374-d76ec32b4fd4
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 2862f968fb0b05a5a75a73ae5573f82dd931ab99d0430266c3c12774f06d833d
              started: "2026-10-20T07:50:23.998Z"
              spawned: false
              bytes: 497987
              windows:
                - start: "2026-10-20T07:50:23.998Z"
                  end: "2026-10-20T06:11:48-07:00"
            - harness: codex
              id: 01a1584a-c580-7be1-b9e4-27e176dcf190
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 12ddd65f645191f83d5ed6bff2278f106909524cf3544b6d595ad8f58ceb8737
              started: "2026-10-20T07:50:58.07Z"
              spawned: false
              bytes: 497320
              windows:
                - start: "2026-10-20T07:50:58.07Z"
                  end: "2026-10-20T06:11:48-07:00"
        queue_waits:
            - attempt_id: df95e0cfbda9be7b41d7893410010f82
              slot: pair:5
              record_digest: c928d081991a64af397814568a448b5b3712d8721210b5429e5e82b618c79d08
              start: "2026-10-20T07:12:10.849393Z"
              end: "2026-10-20T07:16:23.755121Z"
              queue_seconds: 252.905728
              raw_seconds: 252.905728
              exact: true
            - attempt_id: 79b1d82b88f87e7c11357772461723b5
              slot: pair:5
              record_digest: 23633054f964dd027d90985e2ede1cf91df6bc69e187dd93926edceca73c498c
              start: "2026-10-20T07:26:06.16617Z"
              end: "2026-10-20T07:33:48.231009Z"
              queue_seconds: 462.064839
              raw_seconds: 462.064839
              exact: true
            - attempt_id: c0d394fe9f10eadec7b6b422bda9faf7
              slot: pair:5
              record_digest: 6e39d247d66b8767771262aa146fe1ae739597bdf1ed1646272bdd9bf1e68c52
              start: "2026-10-20T08:09:01.204418Z"
              end: "2026-10-20T08:18:25.748278Z"
              queue_seconds: 564.54386
              raw_seconds: 564.54386
              exact: true
            - attempt_id: 6e514d7547f8b78cb5cce642db3567ca
              slot: pair:5
              record_digest: b888e82ea06874bcdeaa1f9e83ff4bb42430767ffc27420c5f48d2f159e176a3
              start: "2026-10-20T11:20:01.955271Z"
              end: "2026-10-20T12:01:51.958873Z"
              queue_seconds: 2510.003602
              raw_seconds: 2510.003602
              exact: true
      "n": 1
      timestamp: "2026-10-20T06:17:36-07:00"
      dispatched: "2026-10-20T06:12:54-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Successful acknowledgements can be discarded by activity that preceded them
          detail: cmd/internal/couchcmd/drain_service.go:266-274 reads the current member snapshot but uses only its identity; DrainAcknowledged instead captures the previous reconciliation's Submission at cmd/internal/couchcore/drain.go:255. If operator activity advances the counter before ack, but reconciliation has not consumed it, ack succeeds with the old baseline and the next observation clears it at drain.go:240, despite no activity after acknowledgement. Establish an authoritative activity baseline at acknowledgement and add a deterministic delayed-observation regression while preserving subsequent-activity invalidation (ARCH-ORDER, ARCH-PURPOSE).
          family: acknowledgement-observation-ordering
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Retain a runnable real SDLC queue conformance check and its cadence
          detail: cmd/internal/couchcmd/drain_queue_test.go:13 exercises a generated shell script, while the stateful tests exercise FakeDrainQueue. The issue Log explicitly records removal of the real executable consumer probes; this range contains no retained real-queue conformance test or scheduled check. Preserve an opt-in isolated-root harness through sdlcDrainQueue and document when it runs, covering owner isolation, completeness, interruption and lost-response reconciliation so future dependency drift is detected (ARCH-MOCK).
          family: external-dependency-conformance
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#722 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T06:17:36-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `acknowledgement-observation-ordering` (kind: defect) Successful acknowledgements can be discarded by activity that preceded them
  cmd/internal/couchcmd/drain_service.go:266-274 reads the current member snapshot but uses only its identity; DrainAcknowledged instead captures the previous reconciliation's Submission at cmd/internal/couchcore/drain.go:255. If operator activity advances the counter before ack, but reconciliation has not consumed it, ack succeeds with the old baseline and the next observation clears it at drain.go:240, despite no activity after acknowledgement. Establish an authoritative activity baseline at acknowledgement and add a deterministic delayed-observation regression while preserving subsequent-activity invalidation (ARCH-ORDER, ARCH-PURPOSE).
- **BR-2** [Important] `external-dependency-conformance` (kind: defect) Retain a runnable real SDLC queue conformance check and its cadence
  cmd/internal/couchcmd/drain_queue_test.go:13 exercises a generated shell script, while the stateful tests exercise FakeDrainQueue. The issue Log explicitly records removal of the real executable consumer probes; this range contains no retained real-queue conformance test or scheduled check. Preserve an opt-in isolated-root harness through sdlcDrainQueue and document when it runs, covering owner isolation, completeness, interruption and lost-response reconciliation so future dependency drift is detected (ARCH-MOCK).

## Open findings

- **BR-1** [Critical] `acknowledgement-observation-ordering` (kind: defect) Successful acknowledgements can be discarded by activity that preceded them
- **BR-2** [Important] `external-dependency-conformance` (kind: defect) Retain a runnable real SDLC queue conformance check and its cadence
