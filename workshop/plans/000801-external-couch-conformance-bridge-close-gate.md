---
gate: boundary-review
issue: 801
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: f43077d7-8315-4d80-8a27-f29b1c8cf320
            repository: github.com/xianxu/pair
            issue_id: "000801"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/6551e851dafaa65f
            reviewed_head: 70954412f60ce3ca578ea72314a8a20f1e6b8a94
            slot: pair:7
            requested_at: "2026-10-22T21:50:39.865009Z"
          phase: finished
          pid: 73933
          started_at: "2026-10-22T21:50:40.115273Z"
          finished_at: "2026-10-22T21:52:49.407591Z"
          session_id: 01a16598-45d3-7ab0-a038-9627f6dc4eb6
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 70954412f60ce3ca578ea72314a8a20f1e6b8a94
        tracker_head: 94005784862c3cec1877ca08b645d25d441d9269
        branch_point: 651e30092e7ca245e5537ba1450b7ea7eb36ddae
        since: "2026-10-22T07:20:32-07:00"
        until: "2026-10-22T14:48:47-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T07:20:32-07:00"
              end: "2026-10-22T14:48:47-07:00"
              began: "2026-10-22T07:20:32-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: cc329a431412698d6d6e72d51235d10081564171579f6351dcee1b3b53cb59d0
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 252386121
              windows:
                - start: "2026-10-22T07:20:32-07:00"
                  end: "2026-10-22T14:48:47-07:00"
            - harness: codex
              id: 01a164fe-5fe7-7161-8046-98e6d9175d18
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: b834909c55928df4c3c469a6d485b5b3a262145229740c241b0602e306442176
              started: "2026-10-22T19:02:34.901Z"
              spawned: true
              bytes: 1197993
              windows:
                - start: "2026-10-22T19:02:34.901Z"
                  end: "2026-10-22T14:48:47-07:00"
            - harness: codex
              id: 01a1655d-8357-7e81-a12f-1336dfb04a78
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 6f1990322cf8cea84749b2a3abec49eaa1b6da4958b36a04ad1fcaff36bd5d05
              started: "2026-10-22T20:46:29.812Z"
              spawned: true
              bytes: 925117
              windows:
                - start: "2026-10-22T20:46:29.812Z"
                  end: "2026-10-22T14:48:47-07:00"
            - harness: codex
              id: 01a16560-d506-73a1-b544-7c38d85e80b6
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 03e41226f6b08780aeb338667b82bd8b35c7dd7d32d22b0071053240ff932891
              started: "2026-10-22T20:50:07.738Z"
              spawned: false
              bytes: 657973
              windows:
                - start: "2026-10-22T20:50:07.738Z"
                  end: "2026-10-22T14:48:47-07:00"
            - harness: codex
              id: 01a16565-5631-77e2-a79f-3347c20e76c8
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 98be070038e872b0c1f01af2fb404f7a957553552fb038dec03de09ac2851cb7
              started: "2026-10-22T20:55:02.849Z"
              spawned: false
              bytes: 427329
              windows:
                - start: "2026-10-22T20:55:02.849Z"
                  end: "2026-10-22T14:48:47-07:00"
            - harness: codex
              id: 01a16567-f4f9-7711-8df4-e61e40024a28
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: d1903d3bbb52c238f3de19a28d5c6e084b687b30fa2ed1af047a96d40d7a3a44
              started: "2026-10-22T20:57:54.509Z"
              spawned: false
              bytes: 456909
              windows:
                - start: "2026-10-22T20:57:54.509Z"
                  end: "2026-10-22T14:48:47-07:00"
            - harness: codex
              id: 01a16568-7edc-7160-91b0-b95037a753c1
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: b6820bbb59510aaf9229f8b661947afb48ea7541c58ea88b506e02752cfd81c0
              started: "2026-10-22T20:58:31.149Z"
              spawned: false
              bytes: 470588
              windows:
                - start: "2026-10-22T20:58:31.149Z"
                  end: "2026-10-22T14:48:47-07:00"
            - harness: codex
              id: 01a16569-b0f9-7fc3-af24-1f50ff938142
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 5aa3374acc5c0259ca8dd29e32a2a839eaadbcf4ec74c9a1eabaddd293dcfa3d
              started: "2026-10-22T20:59:47.915Z"
              spawned: true
              bytes: 2463442
              windows:
                - start: "2026-10-22T20:59:47.915Z"
                  end: "2026-10-22T14:48:47-07:00"
            - harness: codex
              id: 01a16569-f83a-7773-9444-ca8593d784a7
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 623748d6808975c5da2642a8ba6781a1e7b39148ed580f623f416f5893826adb
              started: "2026-10-22T21:00:06.179Z"
              spawned: true
              bytes: 2056319
              windows:
                - start: "2026-10-22T21:00:06.179Z"
                  end: "2026-10-22T14:48:47-07:00"
            - harness: codex
              id: 01a1656a-d28e-7c92-83b9-80f5821d140a
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: c3b556e644a2ca3f5bc02beb835f3addf92df13a0e25159e4ebd5f03fb903f94
              started: "2026-10-22T21:01:02.054Z"
              spawned: true
              bytes: 1192784
              windows:
                - start: "2026-10-22T21:01:02.054Z"
                  end: "2026-10-22T14:48:47-07:00"
            - harness: codex
              id: 01a1657f-515f-7213-a85d-08e9a80f8b88
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 6563ba1d3fafa5f2d4a9d1074842b2e02d4d5572170e9edef3d2115e5f317ef3
              started: "2026-10-22T21:23:25.362Z"
              spawned: true
              bytes: 788531
              windows:
                - start: "2026-10-22T21:23:25.362Z"
                  end: "2026-10-22T14:48:47-07:00"
        protected:
            - issue: "773"
              slot: pair:7
              start: "2026-10-22T15:54:16.793Z"
              end: "2026-10-22T16:16:55.184Z"
            - issue: "773"
              slot: pair:7
              start: "2026-10-22T16:32:41.468Z"
              end: "2026-10-22T16:35:39.432263Z"
            - issue: "773"
              slot: pair:7
              start: "2026-10-22T18:59:01.372Z"
              end: "2026-10-22T19:01:20.839401Z"
            - issue: "773"
              slot: pair:7
              start: "2026-10-22T19:13:56.328565Z"
              end: "2026-10-22T19:18:52.944Z"
            - issue: "773"
              slot: pair:7
              start: "2026-10-22T19:59:02.161Z"
              end: "2026-10-22T19:59:21.948Z"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 49.55413333333333
      "n": 1
      timestamp: "2026-10-22T14:52:51-07:00"
      dispatched: "2026-10-22T14:50:36-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Case-variant response keys can overwrite a failed admission result
          detail: 'cmd/internal/testcouchbridge/native.go:239-273 checks exact duplicate keys, then uses case-insensitive json.Unmarshal. An endpoint reply {"Version":1,"Code":"unavailable","code":"ok","Error":"","EditorEndpoint":null} passes both checks and establishes readiness. Error:null also becomes an accepted empty string. ARCH-SECURE: decode known members with exact names, required-field/type validation and rejection of null strings, while tolerating additive metadata. Sweep registration and endpoint responses and add regression cases.'
          family: strict-boundary-decoding
          round: 1
        - kind: defect
          id: BR-2
          severity: Critical
          title: Build now rejects symlinked platform temporary directories
          detail: cmd/internal/testcouch/binaries.go:35-39 passes os.TempDir() unchanged into BuildOwned, whose owner_build.go:119-120 rejects physical-path aliases. Without /Volumes/TestScratch, supported configurations such as macOS TMPDIR=/tmp now fail before building; the previous implementation accepted these paths. Reuse legacyScratchParent(), which already canonicalizes this selection for New, and add a regression exercising alias selection without the TestScratch override.
          family: legacy-wrapper-compatibility
          round: 1
        - kind: defect
          id: BR-3
          severity: Critical
          title: The Core concepts table does not identify the implemented entities
          detail: 'workshop/plans/000801-external-couch-bridge-plan.md:27-42 names BridgeOptions, BridgeState, BridgeEvent, ReduceBridge and BridgeRunner, but the code declares Options, State, Event, Reduce and RunBridge. It locates BridgeIO in nonexistent testcouchbridge/io.go; the interface is in run.go, with implementations split across native.go and pipes_unix.go. Under the required Core concepts cross-check, this contradiction is blocking. ARCH-PURPOSE: correct the inventory and record the implementation mapping in Revisions.'
          family: core-concept-traceability
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#801 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-22T14:52:51-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `strict-boundary-decoding` (kind: defect) Case-variant response keys can overwrite a failed admission result
  cmd/internal/testcouchbridge/native.go:239-273 checks exact duplicate keys, then uses case-insensitive json.Unmarshal. An endpoint reply {"Version":1,"Code":"unavailable","code":"ok","Error":"","EditorEndpoint":null} passes both checks and establishes readiness. Error:null also becomes an accepted empty string. ARCH-SECURE: decode known members with exact names, required-field/type validation and rejection of null strings, while tolerating additive metadata. Sweep registration and endpoint responses and add regression cases.
- **BR-2** [Critical] `legacy-wrapper-compatibility` (kind: defect) Build now rejects symlinked platform temporary directories
  cmd/internal/testcouch/binaries.go:35-39 passes os.TempDir() unchanged into BuildOwned, whose owner_build.go:119-120 rejects physical-path aliases. Without /Volumes/TestScratch, supported configurations such as macOS TMPDIR=/tmp now fail before building; the previous implementation accepted these paths. Reuse legacyScratchParent(), which already canonicalizes this selection for New, and add a regression exercising alias selection without the TestScratch override.
- **BR-3** [Critical] `core-concept-traceability` (kind: defect) The Core concepts table does not identify the implemented entities
  workshop/plans/000801-external-couch-bridge-plan.md:27-42 names BridgeOptions, BridgeState, BridgeEvent, ReduceBridge and BridgeRunner, but the code declares Options, State, Event, Reduce and RunBridge. It locates BridgeIO in nonexistent testcouchbridge/io.go; the interface is in run.go, with implementations split across native.go and pipes_unix.go. Under the required Core concepts cross-check, this contradiction is blocking. ARCH-PURPOSE: correct the inventory and record the implementation mapping in Revisions.

## Open findings

- **BR-1** [Critical] `strict-boundary-decoding` (kind: defect) Case-variant response keys can overwrite a failed admission result
- **BR-2** [Critical] `legacy-wrapper-compatibility` (kind: defect) Build now rejects symlinked platform temporary directories
- **BR-3** [Critical] `core-concept-traceability` (kind: defect) The Core concepts table does not identify the implemented entities
