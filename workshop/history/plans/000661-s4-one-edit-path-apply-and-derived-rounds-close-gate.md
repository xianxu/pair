---
gate: boundary-review
issue: 661
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-18T03:57:53-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Tasks 3/5/6 and the fixture list enumerate test cases in prose; compress to one adversarial-strategy line per risky function
          detail: (carried from plan-quality PQ-5, deferred to the boundary review)
          family: test-prose-enumeration
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: b02be400-0d9f-418c-8cfa-7eaa8cf2ce8f
            repository: github.com/xianxu/pair
            issue_id: "000661"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/e1a6bcd139762e5d
            reviewed_head: 3b12a97c4e21796062c23b4bdeb6b5e53cfa4d4f
            slot: pair:2
            requested_at: "2026-10-18T10:53:26.689806Z"
          phase: finished
          pid: 88422
          started_at: "2026-10-18T10:53:26.98853Z"
          finished_at: "2026-10-18T10:57:52.008848Z"
          session_id: 101d5fcf-dfce-4492-94c7-be5f51f02fd8
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 3b12a97c4e21796062c23b4bdeb6b5e53cfa4d4f
        tracker_head: 302d930942f5db1022c1891280b88eed4704e3d7
        branch_point: 3851b7338ff107d9b676d1db4eda7c7f21a2aa62
        since: "2026-10-18T02:23:24-07:00"
        until: "2026-10-18T03:20:58-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T02:23:24-07:00"
              end: "2026-10-18T03:20:58-07:00"
              began: "2026-10-18T02:23:24-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 046904b4-4dd3-4e68-8e67-508e23b168dc
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 56993dd2c5b96e5bdbdcf22ccad639e0210155ee73976293061970b528354f3b
              started: "2026-10-18T09:30:56.653Z"
              spawned: false
              bytes: 396710
              windows:
                - start: "2026-10-18T09:30:56.653Z"
                  end: "2026-10-18T03:20:58-07:00"
            - harness: claude
              id: 0c49ca98-486d-42d1-8ed0-6988665098b7
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: ff45165adcf5e1feaf3f832e1db9b803684f1b061da677efe878b0421b762b34
              started: "2026-10-18T09:29:04.811Z"
              spawned: false
              bytes: 547258
              windows:
                - start: "2026-10-18T09:29:04.811Z"
                  end: "2026-10-18T03:20:58-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: b5fad444d7e496fd1a968542bd75abf9c9f394e37bc04d44d34eda94fafc0bb8
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 27542968
              windows:
                - start: "2026-10-18T02:23:24-07:00"
                  end: "2026-10-18T03:20:58-07:00"
            - harness: claude
              id: bd528e0e-7ecc-450e-a0d8-f81dee8d3f69
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: fdbee876d094d89b39389a47aa34894ae7c62b13b4d71dc0b94e5b363304cfc1
              started: "2026-10-18T09:30:40.656Z"
              spawned: false
              bytes: 533286
              windows:
                - start: "2026-10-18T09:30:40.656Z"
                  end: "2026-10-18T03:20:58-07:00"
            - harness: claude
              id: d9bf1eab-1b9e-4362-859c-9b8fa285882a
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf4f2b03d8b1f63808cf771e01a3c19355867d3e9d9e6a754817fafe4d62c6bd
              started: "2026-10-18T09:27:01.542Z"
              spawned: false
              bytes: 634768
              windows:
                - start: "2026-10-18T09:27:01.542Z"
                  end: "2026-10-18T03:20:58-07:00"
        queue_waits:
            - attempt_id: 523a32db8b741aaf4831b4ff7a9b136e
              slot: pair:2
              record_digest: affdfbf8f8d7150fefbfd9e83b16710bb4704da96900839e79f91869fef48822
              start: "2026-10-18T09:50:18.294226Z"
              end: "2026-10-18T10:06:13.757248Z"
              queue_seconds: 955.463022
              raw_seconds: 955.463022
              exact: true
      "n": 2
      timestamp: "2026-10-18T03:57:53-07:00"
      dispatched: "2026-10-18T03:53:23-07:00"
      agent: claude
      findings:
        - kind: defect
          id: BR-2
          severity: Critical
          title: Propose writes markers inside frontmatter on apply / write --all
          detail: "sync.go:551-566 guards only req.Frontmatter; an apply changing \"presentation: propose\" wrote \"presentation: \U0001F916~propose~{direct}\" (probe on proposeDoc), silently flipping Presentation() to highlight. Propose only the body (split via FrontmatterBody) or refuse frontmatter changes; add a rig test."
          family: propose-frontmatter-exclusion
          round: 2
        - kind: defect
          id: BR-3
          severity: Critical
          title: Blank-line insert/delete under propose writes a bare unparseable robot marker
          detail: "propose.go markLine with an empty unpaired body returns marker(\"\",\"\") = bare robot emoji; ProposeMarkers(\"a\\nb\\n\",\"a\\n\\nb\\n\") and the reverse both give \"a\\n\U0001F916\\nb\\n\", and markers.parse_markers finds 0 markers, so it can never be settled. Refuse as propose-unencodable or encode differently; add propose fixtures."
          family: propose-marker-encodability
          round: 2
        - kind: defect
          id: BR-4
          severity: Important
          title: Plan Core concepts and Myers cap disagree with code without a Revisions entry
          detail: Plan names submitsOfDocument returning []Entry and cap D<=4096; code has submitNames + SubmittedSince (owed.go:92,119) and maxDiffEdits=1024 (linediff.go:14). Only the issue Log records it; add a plan Revisions entry.
          family: plan-core-concepts-drift
          round: 2
        - kind: defect
          id: BR-5
          severity: Minor
          title: ParseEdits accepts occurrence <= 0 (plan requires >= 1 at parse)
          detail: It resolves as "not found" instead of a parse refusal matching handoff.lua valid_payload.
          family: edit-record-parse-validation
          round: 2
        - kind: defect
          id: BR-6
          severity: Minor
          title: A no-op (stamp-only) write keeps the round with a now-mismatched result_hash
          detail: If parley treats result_hash != file hash as stale, any no-op write hides the highlight; confirm the contract or refresh result_hash.
          family: round-result-hash-staleness
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 682c11aa-2cb0-48af-be8a-09700a05bb88
            repository: github.com/xianxu/pair
            issue_id: "000661"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/c418d06957eddb02
            reviewed_head: 3aa68861be845a317cc9443204ac5cc85aca9396
            slot: pair:2
            requested_at: "2026-10-18T10:59:57.883862Z"
          phase: finished
          pid: 27865
          started_at: "2026-10-18T10:59:57.980486Z"
          finished_at: "2026-10-18T11:05:16.401535Z"
          session_id: 8a418d9a-a8f8-46e0-8269-ed7f875680cf
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 3aa68861be845a317cc9443204ac5cc85aca9396
        tracker_head: 302d930942f5db1022c1891280b88eed4704e3d7
        branch_point: 3851b7338ff107d9b676d1db4eda7c7f21a2aa62
        since: "2026-10-18T02:23:24-07:00"
        until: "2026-10-18T03:59:28-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T02:23:24-07:00"
              end: "2026-10-18T03:59:28-07:00"
              began: "2026-10-18T02:23:24-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 046904b4-4dd3-4e68-8e67-508e23b168dc
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 56993dd2c5b96e5bdbdcf22ccad639e0210155ee73976293061970b528354f3b
              started: "2026-10-18T09:30:56.653Z"
              spawned: false
              bytes: 396710
              windows:
                - start: "2026-10-18T09:30:56.653Z"
                  end: "2026-10-18T03:59:28-07:00"
            - harness: claude
              id: 0c49ca98-486d-42d1-8ed0-6988665098b7
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: ff45165adcf5e1feaf3f832e1db9b803684f1b061da677efe878b0421b762b34
              started: "2026-10-18T09:29:04.811Z"
              spawned: false
              bytes: 547258
              windows:
                - start: "2026-10-18T09:29:04.811Z"
                  end: "2026-10-18T03:59:28-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 4329ec1f21826481387c39ebe0953e5494520a544cda86b462bd777dc17f4552
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 27681989
              windows:
                - start: "2026-10-18T02:23:24-07:00"
                  end: "2026-10-18T03:59:28-07:00"
            - harness: claude
              id: bd528e0e-7ecc-450e-a0d8-f81dee8d3f69
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: fdbee876d094d89b39389a47aa34894ae7c62b13b4d71dc0b94e5b363304cfc1
              started: "2026-10-18T09:30:40.656Z"
              spawned: false
              bytes: 533286
              windows:
                - start: "2026-10-18T09:30:40.656Z"
                  end: "2026-10-18T03:59:28-07:00"
            - harness: claude
              id: d9bf1eab-1b9e-4362-859c-9b8fa285882a
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf4f2b03d8b1f63808cf771e01a3c19355867d3e9d9e6a754817fafe4d62c6bd
              started: "2026-10-18T09:27:01.542Z"
              spawned: false
              bytes: 634768
              windows:
                - start: "2026-10-18T09:27:01.542Z"
                  end: "2026-10-18T03:59:28-07:00"
        queue_waits:
            - attempt_id: 523a32db8b741aaf4831b4ff7a9b136e
              slot: pair:2
              record_digest: affdfbf8f8d7150fefbfd9e83b16710bb4704da96900839e79f91869fef48822
              start: "2026-10-18T09:50:18.294226Z"
              end: "2026-10-18T10:06:13.757248Z"
              queue_seconds: 955.463022
              raw_seconds: 955.463022
              exact: true
            - attempt_id: 67554bed01c402dd0a3abcd96460092b
              slot: pair:2
              record_digest: 3ab8d678964642c83bf57dcb99317256cfdc07b2088513e592f99941986b0368
              start: "2026-10-18T10:21:02.285634Z"
              end: "2026-10-18T10:41:01.594754Z"
              queue_seconds: 1199.30912
              raw_seconds: 1199.30912
              exact: true
        cuts:
            - boundary: M1
              dispatched: "2026-10-18T03:53:23-07:00"
              finalized: "2026-10-18T03:57:53-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-18T04:05:17-07:00"
      dispatched: "2026-10-18T03:59:56-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan keeps prose test lists by choice (plan-quality round 1 entry); Minor, non-blocking, carried for the close review.
          round: 3
        - id: BR-2
          disposition: addressed
          note: sync.go:559 now calls ProposeDocument; TestProposeNeverMarksFrontmatter goes red when reverted to ProposeMarkers (verified).
          round: 3
        - id: BR-3
          disposition: addressed
          note: markLine leaves unpaired blank lines unmarked; blank-line-insert/delete and paragraph-insert fixtures go red on revert (verified); Lua test requires every robot marker to parse.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Plan Revisions entry (plan lines 313-320) records the Myers cap 1024, submitNames + SubmittedSince, and ProposeDocument.
          round: 3
        - id: BR-5
          disposition: not-addressed
          note: Guard added at edits.go:75 but no regression test; removing it leaves sessionsync and couchcmd suites green (verified). Add occurrence 0 and -1 to the edits_test.go:72 list.
          round: 3
        - id: BR-6
          disposition: withdrawn
          note: 'Issue line 105 records the contract: parley re-anchors by span text when result_hash differs, and the fixed-width stamp keeps positions exact.'
          round: 3
      findings:
        - kind: defect
          id: BR-7
          severity: Important
          title: README and couch SKILL.md omit the new apply/round verbs and propose behaviour
          detail: 'README.md:454-468 lists session-sync verbs but not `apply EDITS.json FILE --if-hash H` or `round FILE --json`, and nothing on presentation: propose or the round sidecar. cmd/internal/couchcmd/skills/couch/SKILL.md (couch --skill, the agent contract) has no apply, edits-invalid, propose-too-large or propose-unencodable. Only run.go usage and atlas/couch.md were updated.'
          family: docs-gate-new-surface
          round: 3
        - kind: defect
          id: BR-8
          severity: Minor
          title: ProposeDocument trusts any frontmatter split; a moved closer writes body text unproposed
          detail: 'This is the 2nd finding in family propose-frontmatter-exclusion. Rule covering both: under propose, write the after-frontmatter as is only when both sides split unambiguously and the after block holds only frontmatter lines; otherwise refuse (propose-unencodable), never fall back to whole-document marking. Probe: base with a body HR `---`, after deleting the closer and changing body to body2, gives frontmatter containing `body2` plus markers, err nil, IsSessionSync true. Prevalence: two instances, the errA/errB fallback at propose.go:68 and the boundary-shift case.'
          family: propose-frontmatter-exclusion
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 507b0c8d-ea80-4850-8dbd-8ccc80563ebf
            repository: github.com/xianxu/pair
            issue_id: "000661"
            recipe: milestone-review
            round: 4
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/0fc9bf0a82547e0a
            reviewed_head: e40f8c76255534074034737670de11ba1fe8d208
            slot: pair:2
            requested_at: "2026-10-18T11:06:51.587047Z"
          phase: finished
          pid: 77864
          started_at: "2026-10-18T11:06:51.701437Z"
          finished_at: "2026-10-18T11:11:15.315707Z"
          session_id: c99e80d3-cbd3-47bc-85fe-9186f07c360f
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: e40f8c76255534074034737670de11ba1fe8d208
        tracker_head: 302d930942f5db1022c1891280b88eed4704e3d7
        branch_point: 3851b7338ff107d9b676d1db4eda7c7f21a2aa62
        since: "2026-10-18T02:23:24-07:00"
        until: "2026-10-18T04:06:26-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T02:23:24-07:00"
              end: "2026-10-18T04:06:26-07:00"
              began: "2026-10-18T02:23:24-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 046904b4-4dd3-4e68-8e67-508e23b168dc
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 56993dd2c5b96e5bdbdcf22ccad639e0210155ee73976293061970b528354f3b
              started: "2026-10-18T09:30:56.653Z"
              spawned: false
              bytes: 396710
              windows:
                - start: "2026-10-18T09:30:56.653Z"
                  end: "2026-10-18T04:06:26-07:00"
            - harness: claude
              id: 0c49ca98-486d-42d1-8ed0-6988665098b7
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: ff45165adcf5e1feaf3f832e1db9b803684f1b061da677efe878b0421b762b34
              started: "2026-10-18T09:29:04.811Z"
              spawned: false
              bytes: 547258
              windows:
                - start: "2026-10-18T09:29:04.811Z"
                  end: "2026-10-18T04:06:26-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 0a26c88bc1f052baaef92cc894642bf5aded42e3b02c24662589eac8a76b67fb
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 27826696
              windows:
                - start: "2026-10-18T02:23:24-07:00"
                  end: "2026-10-18T04:06:26-07:00"
            - harness: claude
              id: bd528e0e-7ecc-450e-a0d8-f81dee8d3f69
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: fdbee876d094d89b39389a47aa34894ae7c62b13b4d71dc0b94e5b363304cfc1
              started: "2026-10-18T09:30:40.656Z"
              spawned: false
              bytes: 533286
              windows:
                - start: "2026-10-18T09:30:40.656Z"
                  end: "2026-10-18T04:06:26-07:00"
            - harness: claude
              id: d9bf1eab-1b9e-4362-859c-9b8fa285882a
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf4f2b03d8b1f63808cf771e01a3c19355867d3e9d9e6a754817fafe4d62c6bd
              started: "2026-10-18T09:27:01.542Z"
              spawned: false
              bytes: 634768
              windows:
                - start: "2026-10-18T09:27:01.542Z"
                  end: "2026-10-18T04:06:26-07:00"
        queue_waits:
            - attempt_id: 523a32db8b741aaf4831b4ff7a9b136e
              slot: pair:2
              record_digest: affdfbf8f8d7150fefbfd9e83b16710bb4704da96900839e79f91869fef48822
              start: "2026-10-18T09:50:18.294226Z"
              end: "2026-10-18T10:06:13.757248Z"
              queue_seconds: 955.463022
              raw_seconds: 955.463022
              exact: true
            - attempt_id: 67554bed01c402dd0a3abcd96460092b
              slot: pair:2
              record_digest: 3ab8d678964642c83bf57dcb99317256cfdc07b2088513e592f99941986b0368
              start: "2026-10-18T10:21:02.285634Z"
              end: "2026-10-18T10:41:01.594754Z"
              queue_seconds: 1199.30912
              raw_seconds: 1199.30912
              exact: true
        cuts:
            - boundary: M1
              dispatched: "2026-10-18T03:53:23-07:00"
              finalized: "2026-10-18T03:57:53-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-18T03:59:56-07:00"
              finalized: "2026-10-18T04:05:17-07:00"
              finalizing: false
      "n": 4
      timestamp: "2026-10-18T04:11:49-07:00"
      dispatched: "2026-10-18T04:06:50-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: withdrawn
          note: 'Overtaken by implementation: Tasks 3/5/6 are done and their enumerated cases exist as tests, so compressing the finished task prose no longer guides any work.'
          round: 4
        - id: BR-5
          disposition: addressed
          note: edits.go ParseEdits refuses n < 1; edits_test.go TestParseEditsRefusesMalformed adds occurrence 0 and -1, which the code accepted before the fix.
          round: 4
        - id: BR-7
          disposition: addressed
          note: README.md:465-471 documents apply and round, the propose refusals and the side-doc fallback; SKILL.md adds both table rows and the apply/edits-invalid/exit 6/rounds/propose-too-large/propose-unencodable guidance, matching sync.go and propose.go.
          round: 4
        - id: BR-8
          disposition: addressed
          note: propose.go:68 refuses with propose-unencodable unless both sides split and fa == fb, with no whole-document fallback (the stated family rule); putting back the 3aa68861b propose.go makes TestProposeNeverMarksFrontmatter fail.
          round: 4
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: e40f8c76255534074034737670de11ba1fe8d208
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 9a20f231-bbda-4ed5-bd53-09019f59e636
            repository: github.com/xianxu/pair
            issue_id: "000661"
            recipe: milestone-review
            round: 5
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/9abc4d65d8d35136
            reviewed_head: ae49e84daafa33158a904563af74269decedee57
            slot: pair:2
            requested_at: "2026-10-18T13:10:02.855147Z"
          phase: finished
          pid: 91696
          started_at: "2026-10-18T13:10:03.088334Z"
          finished_at: "2026-10-18T13:12:30.751791Z"
          session_id: f121eaa8-6b70-4e6c-b76b-8838be99e6ea
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: ae49e84daafa33158a904563af74269decedee57
        tracker_head: 7023f04d96950faf3e7f9db78e152d4498d1466c
        branch_point: f8b82353d3d0080ccf3c770df927402865b85c68
        since: "2026-10-18T02:23:24-07:00"
        until: "2026-10-18T06:09:33-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T02:23:24-07:00"
              end: "2026-10-18T06:09:33-07:00"
              began: "2026-10-18T02:23:24-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 046904b4-4dd3-4e68-8e67-508e23b168dc
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 56993dd2c5b96e5bdbdcf22ccad639e0210155ee73976293061970b528354f3b
              started: "2026-10-18T09:30:56.653Z"
              spawned: false
              bytes: 396710
              windows:
                - start: "2026-10-18T09:30:56.653Z"
                  end: "2026-10-18T06:09:33-07:00"
            - harness: claude
              id: 0c49ca98-486d-42d1-8ed0-6988665098b7
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: ff45165adcf5e1feaf3f832e1db9b803684f1b061da677efe878b0421b762b34
              started: "2026-10-18T09:29:04.811Z"
              spawned: false
              bytes: 547258
              windows:
                - start: "2026-10-18T09:29:04.811Z"
                  end: "2026-10-18T06:09:33-07:00"
            - harness: claude
              id: 311b2335-741e-4fc3-b9b2-ab4c2958124e
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: a68ca6009c2bbd07d59107afe48d654f3305e9be07328f0cd6d77569ce2f6d54
              started: "2026-10-18T11:55:52.03Z"
              spawned: false
              bytes: 405884
              windows:
                - start: "2026-10-18T11:55:52.03Z"
                  end: "2026-10-18T06:09:33-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 5d54433e6e9be2b8850dd547d6bda6e75f6d5bf00c02ef96b0716e13b9d40d94
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 30457462
              windows:
                - start: "2026-10-18T02:23:24-07:00"
                  end: "2026-10-18T06:09:33-07:00"
            - harness: claude
              id: bd528e0e-7ecc-450e-a0d8-f81dee8d3f69
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: fdbee876d094d89b39389a47aa34894ae7c62b13b4d71dc0b94e5b363304cfc1
              started: "2026-10-18T09:30:40.656Z"
              spawned: false
              bytes: 533286
              windows:
                - start: "2026-10-18T09:30:40.656Z"
                  end: "2026-10-18T06:09:33-07:00"
            - harness: claude
              id: d9bf1eab-1b9e-4362-859c-9b8fa285882a
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf4f2b03d8b1f63808cf771e01a3c19355867d3e9d9e6a754817fafe4d62c6bd
              started: "2026-10-18T09:27:01.542Z"
              spawned: false
              bytes: 634768
              windows:
                - start: "2026-10-18T09:27:01.542Z"
                  end: "2026-10-18T06:09:33-07:00"
            - harness: claude
              id: f9bf5586-2e6c-4cdc-8d7a-3b5e925be7ea
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 7a6f46583f84d0e98f2e701a9c22c36763759ecb261ace1202aa8321faf62f2c
              started: "2026-10-18T11:55:34.795Z"
              spawned: false
              bytes: 576633
              windows:
                - start: "2026-10-18T11:55:34.795Z"
                  end: "2026-10-18T06:09:33-07:00"
        queue_waits:
            - attempt_id: 523a32db8b741aaf4831b4ff7a9b136e
              slot: pair:2
              record_digest: affdfbf8f8d7150fefbfd9e83b16710bb4704da96900839e79f91869fef48822
              start: "2026-10-18T09:50:18.294226Z"
              end: "2026-10-18T10:06:13.757248Z"
              queue_seconds: 955.463022
              raw_seconds: 955.463022
              exact: true
            - attempt_id: 67554bed01c402dd0a3abcd96460092b
              slot: pair:2
              record_digest: 3ab8d678964642c83bf57dcb99317256cfdc07b2088513e592f99941986b0368
              start: "2026-10-18T10:21:02.285634Z"
              end: "2026-10-18T10:41:01.594754Z"
              queue_seconds: 1199.30912
              raw_seconds: 1199.30912
              exact: true
            - attempt_id: e57d69b3ae1bdbb98281261a995b5169
              slot: pair:2
              record_digest: 2803fbdf0605047f52af29d3deb8cf8a153ac0eb0d63b71e3ae5be72d935311d
              start: "2026-10-18T11:16:43.796702Z"
              end: "2026-10-18T11:31:33.940997Z"
              queue_seconds: 890.144295
              raw_seconds: 890.144295
              exact: true
            - attempt_id: e3c020396910d7fdfc3a921e0de645ca
              slot: pair:2
              record_digest: 749af793e582cd9fefb6a9e00ea06e0b81c71c75e5fd7d582e67c7b2915d4a91
              start: "2026-10-18T12:22:15.474833Z"
              end: "2026-10-18T12:37:13.592351Z"
              queue_seconds: 898.117518
              raw_seconds: 898.117518
              exact: true
            - attempt_id: da490acce37f4ea99c6cfc8e16c097bd
              slot: pair:2
              record_digest: f157e62aa0c87079d904d3c019b4faa4d8babc8d33f88c8b9aeaae867ae23e66
              start: "2026-10-18T12:50:30.655393Z"
              end: "2026-10-18T12:54:31.33809Z"
              queue_seconds: 240.682697
              raw_seconds: 240.682697
              exact: true
        cuts:
            - boundary: M1
              dispatched: "2026-10-18T03:53:23-07:00"
              finalized: "2026-10-18T03:57:53-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-18T03:59:56-07:00"
              finalized: "2026-10-18T04:05:17-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-18T04:06:50-07:00"
              finalized: "2026-10-18T04:11:49-07:00"
              finalizing: true
      "n": 5
      timestamp: "2026-10-18T06:12:33-07:00"
      dispatched: "2026-10-18T06:09:59-07:00"
      agent: claude
      findings:
        - kind: defect
          id: BR-9
          severity: Critical
          title: answersSideDoc rule (a) lets an arbitrary multi-line rewrite apply directly under propose if new ends in a side-doc link
          detail: "round.go:983 checks only readyMarkerIn(old), a sideDocLink in new and no ready marker in new; it never requires the text outside the marker to be unchanged. A probe with base \"Line one..Line four. \U0001F916[shorten]\" and new \"Totally rewritten.\\nX\\nY\\nZ\\nW → [proposal](doc.side/x.md)\" returned err=nil and wrote the rewrite with no markers, contradicting README, SKILL and atlas (\"the record that replaces the operator's ready marker with a new link\"). Fix: require new == old[:i] + link + old[j:] where [i,j) is the ready marker's span, and add the probe as a negative test plus a refused conformance fixture."
          family: propose-direct-exception-scope
          round: 5
        - kind: defect
          id: BR-10
          severity: Important
          title: ApplyFile returns an error (exit 1, no Hash, no Body) when the round artifact write fails after the document was already replaced
          detail: 'applyfile.go:273-275: the document has changed, but the CLI reports Code error, `pair review round` returns a null round, the commit body is lost, and the agent''s next apply goes stale. M1 decision 4 makes a round-write failure a warning. Return the new Hash with a warning (exit 0 or a distinct code), and test it by making only the artifact path unwritable.'
          family: committed-effect-reported-as-failure
          round: 5
        - kind: defect
          id: BR-11
          severity: Important
          title: FILE.lock written by oplock.lua has no removal path after a pane crash; agent applies then exit 3 indefinitely
          detail: 'oplock.lua removes the lock only on send or VimLeave, and treats any lock it did not take in this process as someone else''s: take returns false and drop never removes it. After an nvim crash or SIGKILL, the lock stays in the user''s repo, every pair review apply refuses with operator-lock, and the operator is never told. Record an owner (pid or tag) in the lock and reclaim it when that owner is dead or is this same review, or notify the operator; test the crash-leftover case.'
          family: owned-artifact-removal-on-owner-death
          round: 5
        - kind: defect
          id: BR-12
          severity: Important
          title: Plan's "M2 as built" commits readiness to ignore an untracked FILE.lock with a pinning test; neither exists
          detail: 'This is the 2nd finding in family plan-core-concepts-drift. The rule for the class: every "M2 as built" commitment is either in the code with a test or listed in a Revisions entry as dropped or deferred; sweep the whole section, including the planned review-loop-test/fake-review-agent rewrite that review-applied-round-test.sh replaced. reviewcmd/run.go:277 still uses a plain `git status --porcelain`, so a leftover FILE.lock makes the repo "interact" instead of "new".'
          family: plan-core-concepts-drift
          round: 5
        - kind: defect
          id: BR-13
          severity: Minor
          title: round_watch marks a round seen before on_applied_round runs, so a refused reload (modified buffer) loses its decorations for good
          detail: watch sets seen = result_hash before calling cb; on_applied_round returns false on a modified buffer and the round is never re-delivered.
          family: round-watch-delivery
          round: 5
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 18ac178c-7b11-4c0f-b7c0-d096998c709b
            repository: github.com/xianxu/pair
            issue_id: "000661"
            recipe: milestone-review
            round: 6
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/9c78299f55a9a765
            reviewed_head: 6072d90471d52d0ba2f74f9b80e9150468cac04d
            slot: pair:2
            requested_at: "2026-10-18T13:20:08.522782Z"
          phase: finished
          pid: 49473
          started_at: "2026-10-18T13:20:08.628399Z"
          finished_at: "2026-10-18T13:23:20.21868Z"
          session_id: f0a8de62-ee6c-46c2-b161-d9aab448f78f
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 6072d90471d52d0ba2f74f9b80e9150468cac04d
        tracker_head: 7023f04d96950faf3e7f9db78e152d4498d1466c
        branch_point: f8b82353d3d0080ccf3c770df927402865b85c68
        since: "2026-10-18T02:23:24-07:00"
        until: "2026-10-18T06:15:16-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T02:23:24-07:00"
              end: "2026-10-18T06:15:16-07:00"
              began: "2026-10-18T02:23:24-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 046904b4-4dd3-4e68-8e67-508e23b168dc
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 56993dd2c5b96e5bdbdcf22ccad639e0210155ee73976293061970b528354f3b
              started: "2026-10-18T09:30:56.653Z"
              spawned: false
              bytes: 396710
              windows:
                - start: "2026-10-18T09:30:56.653Z"
                  end: "2026-10-18T06:15:16-07:00"
            - harness: claude
              id: 0c49ca98-486d-42d1-8ed0-6988665098b7
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: ff45165adcf5e1feaf3f832e1db9b803684f1b061da677efe878b0421b762b34
              started: "2026-10-18T09:29:04.811Z"
              spawned: false
              bytes: 547258
              windows:
                - start: "2026-10-18T09:29:04.811Z"
                  end: "2026-10-18T06:15:16-07:00"
            - harness: claude
              id: 311b2335-741e-4fc3-b9b2-ab4c2958124e
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: a68ca6009c2bbd07d59107afe48d654f3305e9be07328f0cd6d77569ce2f6d54
              started: "2026-10-18T11:55:52.03Z"
              spawned: false
              bytes: 405884
              windows:
                - start: "2026-10-18T11:55:52.03Z"
                  end: "2026-10-18T06:15:16-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: a082e204e83574b0f967d3afd93e7c48a2a91ba994716b8ade131c0107b3db79
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 30720727
              windows:
                - start: "2026-10-18T02:23:24-07:00"
                  end: "2026-10-18T06:15:16-07:00"
            - harness: claude
              id: bd528e0e-7ecc-450e-a0d8-f81dee8d3f69
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: fdbee876d094d89b39389a47aa34894ae7c62b13b4d71dc0b94e5b363304cfc1
              started: "2026-10-18T09:30:40.656Z"
              spawned: false
              bytes: 533286
              windows:
                - start: "2026-10-18T09:30:40.656Z"
                  end: "2026-10-18T06:15:16-07:00"
            - harness: claude
              id: d9bf1eab-1b9e-4362-859c-9b8fa285882a
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf4f2b03d8b1f63808cf771e01a3c19355867d3e9d9e6a754817fafe4d62c6bd
              started: "2026-10-18T09:27:01.542Z"
              spawned: false
              bytes: 634768
              windows:
                - start: "2026-10-18T09:27:01.542Z"
                  end: "2026-10-18T06:15:16-07:00"
            - harness: claude
              id: f9bf5586-2e6c-4cdc-8d7a-3b5e925be7ea
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 7a6f46583f84d0e98f2e701a9c22c36763759ecb261ace1202aa8321faf62f2c
              started: "2026-10-18T11:55:34.795Z"
              spawned: false
              bytes: 576633
              windows:
                - start: "2026-10-18T11:55:34.795Z"
                  end: "2026-10-18T06:15:16-07:00"
        queue_waits:
            - attempt_id: 523a32db8b741aaf4831b4ff7a9b136e
              slot: pair:2
              record_digest: affdfbf8f8d7150fefbfd9e83b16710bb4704da96900839e79f91869fef48822
              start: "2026-10-18T09:50:18.294226Z"
              end: "2026-10-18T10:06:13.757248Z"
              queue_seconds: 955.463022
              raw_seconds: 955.463022
              exact: true
            - attempt_id: 67554bed01c402dd0a3abcd96460092b
              slot: pair:2
              record_digest: 3ab8d678964642c83bf57dcb99317256cfdc07b2088513e592f99941986b0368
              start: "2026-10-18T10:21:02.285634Z"
              end: "2026-10-18T10:41:01.594754Z"
              queue_seconds: 1199.30912
              raw_seconds: 1199.30912
              exact: true
            - attempt_id: e57d69b3ae1bdbb98281261a995b5169
              slot: pair:2
              record_digest: 2803fbdf0605047f52af29d3deb8cf8a153ac0eb0d63b71e3ae5be72d935311d
              start: "2026-10-18T11:16:43.796702Z"
              end: "2026-10-18T11:31:33.940997Z"
              queue_seconds: 890.144295
              raw_seconds: 890.144295
              exact: true
            - attempt_id: e3c020396910d7fdfc3a921e0de645ca
              slot: pair:2
              record_digest: 749af793e582cd9fefb6a9e00ea06e0b81c71c75e5fd7d582e67c7b2915d4a91
              start: "2026-10-18T12:22:15.474833Z"
              end: "2026-10-18T12:37:13.592351Z"
              queue_seconds: 898.117518
              raw_seconds: 898.117518
              exact: true
            - attempt_id: da490acce37f4ea99c6cfc8e16c097bd
              slot: pair:2
              record_digest: f157e62aa0c87079d904d3c019b4faa4d8babc8d33f88c8b9aeaae867ae23e66
              start: "2026-10-18T12:50:30.655393Z"
              end: "2026-10-18T12:54:31.33809Z"
              queue_seconds: 240.682697
              raw_seconds: 240.682697
              exact: true
        cuts:
            - boundary: M1
              dispatched: "2026-10-18T03:53:23-07:00"
              finalized: "2026-10-18T03:57:53-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-18T03:59:56-07:00"
              finalized: "2026-10-18T04:05:17-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-18T04:06:50-07:00"
              finalized: "2026-10-18T04:11:49-07:00"
              finalizing: true
            - dispatched: "2026-10-18T06:09:59-07:00"
              finalized: "2026-10-18T06:12:33-07:00"
              finalizing: false
      "n": 6
      timestamp: "2026-10-18T06:23:21-07:00"
      dispatched: "2026-10-18T06:20:06-07:00"
      agent: claude
      dispose:
        - id: BR-9
          disposition: addressed
          note: round.go answersSideDoc now requires an exact swap; TestProposeLinkAnswerCannotCarryARewrite and the conformance fixture fail when round.go is reverted.
          round: 6
        - id: BR-10
          disposition: addressed
          note: applyfile.go returns Hash, Round, Body and Warning; TestApplyFileRoundWriteFailureWarns fails when the warning is reverted to return err.
          round: 6
        - id: BR-11
          disposition: addressed
          note: oplock records a pid and reclaims a dead owner at start and on take; round_watch_test covers dead, live and own-pid locks. EPERM edge noted as a new Minor.
          round: 6
        - id: BR-12
          disposition: addressed
          note: cleanBesidesLock is wired into both porcelain checks with a helper test, and the Revisions entry sweeps M2 as built. Gaps in quoting and call-site coverage are raised as a new finding.
          round: 6
        - id: BR-13
          disposition: addressed
          note: watch marks a round seen only when cb does not return false; the redelivery test covers it. A side effect is raised as a new Minor.
          round: 6
      findings:
        - kind: defect
          id: BR-14
          severity: Important
          title: cleanBesidesLock misses the lock for documents whose path git quotes, and readiness wiring is untested
          detail: 'run.go:257 compares against an unquoted "?? rel.lock" line, but porcelain v1 prints "?? \"docs/my doc.md.lock\"" for names with spaces or non-ASCII characters (reproduced), so readiness still reports such repos dirty. Fix: use git status --porcelain -z and compare raw entries. Add a RunReadiness or gatherGitFacts test with a lock-only status, because reverting the call site keeps all tests green.'
          family: git-porcelain-path-parsing
          round: 6
        - kind: defect
          id: BR-15
          severity: Minor
          title: A declined round warns every 100ms and is redelivered without checking the file still hashes to result_hash
          detail: 'This is the 2nd finding in family round-watch-delivery. The rule: the pane acts on a round only while the file on disk hashes to round.result_hash, and it reports each declined round once. Put the hash check in decide, and debounce the notification per result_hash.'
          family: round-watch-delivery
          round: 6
        - kind: defect
          id: BR-16
          severity: Minor
          title: The apply reply's Warning field and the lock's pid/reclaim are not documented in the couch SKILL or the atlas
          detail: 'This is the 2nd finding in family docs-gate-new-surface. The rule: any change to the apply reply or lock contract updates SKILL.md and atlas/review-workbench.md in the same commit. The agent must learn that Warning means "commit Body now, no highlight".'
          family: docs-gate-new-surface
          round: 6
        - kind: defect
          id: BR-17
          severity: Minor
          title: oplock alive() treats EPERM as dead, so it reclaims a lock owned by a live process of another user
          detail: vim.uv.kill returns nil,"EPERM" for a live process the caller cannot signal. Only ESRCH should count as dead.
          family: owned-artifact-removal-on-owner-death
          round: 6
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: f3801f9c-16e0-4a10-bb96-0b1d3ab19500
            repository: github.com/xianxu/pair
            issue_id: "000661"
            recipe: milestone-review
            round: 7
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/f43433ecbffe7816
            reviewed_head: 3862431a728b78c31f179ac7078bb6b7cc069b16
            slot: pair:2
            requested_at: "2026-10-18T13:29:31.480867Z"
          phase: finished
          pid: 67127
          started_at: "2026-10-18T13:29:31.670453Z"
          finished_at: "2026-10-18T13:31:13.079715Z"
          session_id: 344d094f-5cd5-4e2b-8755-da26b54d8334
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 3862431a728b78c31f179ac7078bb6b7cc069b16
        tracker_head: 7023f04d96950faf3e7f9db78e152d4498d1466c
        branch_point: f8b82353d3d0080ccf3c770df927402865b85c68
        since: "2026-10-18T02:23:24-07:00"
        until: "2026-10-18T06:25:20-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T02:23:24-07:00"
              end: "2026-10-18T06:25:20-07:00"
              began: "2026-10-18T02:23:24-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 046904b4-4dd3-4e68-8e67-508e23b168dc
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 56993dd2c5b96e5bdbdcf22ccad639e0210155ee73976293061970b528354f3b
              started: "2026-10-18T09:30:56.653Z"
              spawned: false
              bytes: 396710
              windows:
                - start: "2026-10-18T09:30:56.653Z"
                  end: "2026-10-18T06:25:20-07:00"
            - harness: claude
              id: 0c49ca98-486d-42d1-8ed0-6988665098b7
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: ff45165adcf5e1feaf3f832e1db9b803684f1b061da677efe878b0421b762b34
              started: "2026-10-18T09:29:04.811Z"
              spawned: false
              bytes: 547258
              windows:
                - start: "2026-10-18T09:29:04.811Z"
                  end: "2026-10-18T06:25:20-07:00"
            - harness: claude
              id: 311b2335-741e-4fc3-b9b2-ab4c2958124e
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: a68ca6009c2bbd07d59107afe48d654f3305e9be07328f0cd6d77569ce2f6d54
              started: "2026-10-18T11:55:52.03Z"
              spawned: false
              bytes: 405884
              windows:
                - start: "2026-10-18T11:55:52.03Z"
                  end: "2026-10-18T06:25:20-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: d482d9054d9090239ac1a84e88a13aca4b2d0df229ab6681839b3dbf82fa9d61
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 30909853
              windows:
                - start: "2026-10-18T02:23:24-07:00"
                  end: "2026-10-18T06:25:20-07:00"
            - harness: claude
              id: bd528e0e-7ecc-450e-a0d8-f81dee8d3f69
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: fdbee876d094d89b39389a47aa34894ae7c62b13b4d71dc0b94e5b363304cfc1
              started: "2026-10-18T09:30:40.656Z"
              spawned: false
              bytes: 533286
              windows:
                - start: "2026-10-18T09:30:40.656Z"
                  end: "2026-10-18T06:25:20-07:00"
            - harness: claude
              id: d9bf1eab-1b9e-4362-859c-9b8fa285882a
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf4f2b03d8b1f63808cf771e01a3c19355867d3e9d9e6a754817fafe4d62c6bd
              started: "2026-10-18T09:27:01.542Z"
              spawned: false
              bytes: 634768
              windows:
                - start: "2026-10-18T09:27:01.542Z"
                  end: "2026-10-18T06:25:20-07:00"
            - harness: claude
              id: f9bf5586-2e6c-4cdc-8d7a-3b5e925be7ea
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 7a6f46583f84d0e98f2e701a9c22c36763759ecb261ace1202aa8321faf62f2c
              started: "2026-10-18T11:55:34.795Z"
              spawned: false
              bytes: 576633
              windows:
                - start: "2026-10-18T11:55:34.795Z"
                  end: "2026-10-18T06:25:20-07:00"
        queue_waits:
            - attempt_id: 523a32db8b741aaf4831b4ff7a9b136e
              slot: pair:2
              record_digest: affdfbf8f8d7150fefbfd9e83b16710bb4704da96900839e79f91869fef48822
              start: "2026-10-18T09:50:18.294226Z"
              end: "2026-10-18T10:06:13.757248Z"
              queue_seconds: 955.463022
              raw_seconds: 955.463022
              exact: true
            - attempt_id: 67554bed01c402dd0a3abcd96460092b
              slot: pair:2
              record_digest: 3ab8d678964642c83bf57dcb99317256cfdc07b2088513e592f99941986b0368
              start: "2026-10-18T10:21:02.285634Z"
              end: "2026-10-18T10:41:01.594754Z"
              queue_seconds: 1199.30912
              raw_seconds: 1199.30912
              exact: true
            - attempt_id: e57d69b3ae1bdbb98281261a995b5169
              slot: pair:2
              record_digest: 2803fbdf0605047f52af29d3deb8cf8a153ac0eb0d63b71e3ae5be72d935311d
              start: "2026-10-18T11:16:43.796702Z"
              end: "2026-10-18T11:31:33.940997Z"
              queue_seconds: 890.144295
              raw_seconds: 890.144295
              exact: true
            - attempt_id: e3c020396910d7fdfc3a921e0de645ca
              slot: pair:2
              record_digest: 749af793e582cd9fefb6a9e00ea06e0b81c71c75e5fd7d582e67c7b2915d4a91
              start: "2026-10-18T12:22:15.474833Z"
              end: "2026-10-18T12:37:13.592351Z"
              queue_seconds: 898.117518
              raw_seconds: 898.117518
              exact: true
            - attempt_id: da490acce37f4ea99c6cfc8e16c097bd
              slot: pair:2
              record_digest: f157e62aa0c87079d904d3c019b4faa4d8babc8d33f88c8b9aeaae867ae23e66
              start: "2026-10-18T12:50:30.655393Z"
              end: "2026-10-18T12:54:31.33809Z"
              queue_seconds: 240.682697
              raw_seconds: 240.682697
              exact: true
        cuts:
            - boundary: M1
              dispatched: "2026-10-18T03:53:23-07:00"
              finalized: "2026-10-18T03:57:53-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-18T03:59:56-07:00"
              finalized: "2026-10-18T04:05:17-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-18T04:06:50-07:00"
              finalized: "2026-10-18T04:11:49-07:00"
              finalizing: true
            - dispatched: "2026-10-18T06:09:59-07:00"
              finalized: "2026-10-18T06:12:33-07:00"
              finalizing: false
            - dispatched: "2026-10-18T06:20:06-07:00"
              finalized: "2026-10-18T06:23:21-07:00"
              finalizing: false
      "n": 7
      timestamp: "2026-10-18T06:31:13-07:00"
      dispatched: "2026-10-18T06:29:29-07:00"
      agent: claude
      dispose:
        - id: BR-14
          disposition: addressed
          note: run.go gatherGitFacts/prepare use status --porcelain -z with NUL split; TestReadinessTreatsTheReviewLockAsClean fails on revert of call site or parser; space-path case added to the unit table.
          round: 7
        - id: BR-15
          disposition: not-addressed
          note: Stale-hash guard is in place and pinned by 'a round the file no longer matches is not delivered'; the per-result_hash warn-once in init.lua on_applied_round has no test, and decide's current_hash parameter is unused in production (watch compares inline).
          round: 7
        - id: BR-16
          disposition: addressed
          note: SKILL.md documents Warning means commit Body anyway; atlas/review-workbench.md documents Warning, pid lock owner, ESRCH reclaim, and readiness ignoring the lock via porcelain -z.
          round: 7
        - id: BR-17
          disposition: not-addressed
          note: alive() now treats only ESRCH as dead, and a manual probe confirms reclaim leaves a pid-1 (EPERM) lock; but no test pins it, since the live-owner case uses the signalable ppid, so reverting stays green. Add a pid 1 lock case.
          round: 7
      recipe: milestone-review
      isolation: fenced
      reviewed: 3862431a728b78c31f179ac7078bb6b7cc069b16
      blocked: false
---

# Gate ledger — pair#661 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-18T03:57:53-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `test-prose-enumeration` Tasks 3/5/6 and the fixture list enumerate test cases in prose; compress to one adversarial-strategy line per risky function
  (carried from plan-quality PQ-5, deferred to the boundary review)

## Round 2 — 2026-10-18T03:57:53-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Critical] `propose-frontmatter-exclusion` (kind: defect) Propose writes markers inside frontmatter on apply / write --all
  sync.go:551-566 guards only req.Frontmatter; an apply changing "presentation: propose" wrote "presentation: 🤖~propose~{direct}" (probe on proposeDoc), silently flipping Presentation() to highlight. Propose only the body (split via FrontmatterBody) or refuse frontmatter changes; add a rig test.
- **BR-3** [Critical] `propose-marker-encodability` (kind: defect) Blank-line insert/delete under propose writes a bare unparseable robot marker
  propose.go markLine with an empty unpaired body returns marker("","") = bare robot emoji; ProposeMarkers("a\nb\n","a\n\nb\n") and the reverse both give "a\n🤖\nb\n", and markers.parse_markers finds 0 markers, so it can never be settled. Refuse as propose-unencodable or encode differently; add propose fixtures.
- **BR-4** [Important] `plan-core-concepts-drift` (kind: defect) Plan Core concepts and Myers cap disagree with code without a Revisions entry
  Plan names submitsOfDocument returning []Entry and cap D<=4096; code has submitNames + SubmittedSince (owed.go:92,119) and maxDiffEdits=1024 (linediff.go:14). Only the issue Log records it; add a plan Revisions entry.
- **BR-5** [Minor] `edit-record-parse-validation` (kind: defect) ParseEdits accepts occurrence <= 0 (plan requires >= 1 at parse)
  It resolves as "not found" instead of a parse refusal matching handoff.lua valid_payload.
- **BR-6** [Minor] `round-result-hash-staleness` (kind: defect) A no-op (stamp-only) write keeps the round with a now-mismatched result_hash
  If parley treats result_hash != file hash as stale, any no-op write hides the highlight; confirm the contract or refresh result_hash.

## Round 3 — 2026-10-18T04:05:17-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Plan keeps prose test lists by choice (plan-quality round 1 entry); Minor, non-blocking, carried for the close review.
- BR-2 — addressed — sync.go:559 now calls ProposeDocument; TestProposeNeverMarksFrontmatter goes red when reverted to ProposeMarkers (verified).
- BR-3 — addressed — markLine leaves unpaired blank lines unmarked; blank-line-insert/delete and paragraph-insert fixtures go red on revert (verified); Lua test requires every robot marker to parse.
- BR-4 — addressed — Plan Revisions entry (plan lines 313-320) records the Myers cap 1024, submitNames + SubmittedSince, and ProposeDocument.
- BR-5 — not-addressed — Guard added at edits.go:75 but no regression test; removing it leaves sessionsync and couchcmd suites green (verified). Add occurrence 0 and -1 to the edits_test.go:72 list.
- BR-6 — withdrawn — Issue line 105 records the contract: parley re-anchors by span text when result_hash differs, and the fixed-width stamp keeps positions exact.

### Raised

- **BR-7** [Important] `docs-gate-new-surface` (kind: defect) README and couch SKILL.md omit the new apply/round verbs and propose behaviour
  README.md:454-468 lists session-sync verbs but not `apply EDITS.json FILE --if-hash H` or `round FILE --json`, and nothing on presentation: propose or the round sidecar. cmd/internal/couchcmd/skills/couch/SKILL.md (couch --skill, the agent contract) has no apply, edits-invalid, propose-too-large or propose-unencodable. Only run.go usage and atlas/couch.md were updated.
- **BR-8** [Minor] `propose-frontmatter-exclusion` (kind: defect) ProposeDocument trusts any frontmatter split; a moved closer writes body text unproposed
  This is the 2nd finding in family propose-frontmatter-exclusion. Rule covering both: under propose, write the after-frontmatter as is only when both sides split unambiguously and the after block holds only frontmatter lines; otherwise refuse (propose-unencodable), never fall back to whole-document marking. Probe: base with a body HR `---`, after deleting the closer and changing body to body2, gives frontmatter containing `body2` plus markers, err nil, IsSessionSync true. Prevalence: two instances, the errA/errB fallback at propose.go:68 and the boundary-shift case.

## Round 4 — 2026-10-18T04:11:49-07:00 (claude) — passed

### Disposed

- BR-1 — withdrawn — Overtaken by implementation: Tasks 3/5/6 are done and their enumerated cases exist as tests, so compressing the finished task prose no longer guides any work.
- BR-5 — addressed — edits.go ParseEdits refuses n < 1; edits_test.go TestParseEditsRefusesMalformed adds occurrence 0 and -1, which the code accepted before the fix.
- BR-7 — addressed — README.md:465-471 documents apply and round, the propose refusals and the side-doc fallback; SKILL.md adds both table rows and the apply/edits-invalid/exit 6/rounds/propose-too-large/propose-unencodable guidance, matching sync.go and propose.go.
- BR-8 — addressed — propose.go:68 refuses with propose-unencodable unless both sides split and fa == fb, with no whole-document fallback (the stated family rule); putting back the 3aa68861b propose.go makes TestProposeNeverMarksFrontmatter fail.

## Round 5 — 2026-10-18T06:12:33-07:00 (claude) — BLOCKED

### Raised

- **BR-9** [Critical] `propose-direct-exception-scope` (kind: defect) answersSideDoc rule (a) lets an arbitrary multi-line rewrite apply directly under propose if new ends in a side-doc link
  round.go:983 checks only readyMarkerIn(old), a sideDocLink in new and no ready marker in new; it never requires the text outside the marker to be unchanged. A probe with base "Line one..Line four. 🤖[shorten]" and new "Totally rewritten.\nX\nY\nZ\nW → [proposal](doc.side/x.md)" returned err=nil and wrote the rewrite with no markers, contradicting README, SKILL and atlas ("the record that replaces the operator's ready marker with a new link"). Fix: require new == old[:i] + link + old[j:] where [i,j) is the ready marker's span, and add the probe as a negative test plus a refused conformance fixture.
- **BR-10** [Important] `committed-effect-reported-as-failure` (kind: defect) ApplyFile returns an error (exit 1, no Hash, no Body) when the round artifact write fails after the document was already replaced
  applyfile.go:273-275: the document has changed, but the CLI reports Code error, `pair review round` returns a null round, the commit body is lost, and the agent's next apply goes stale. M1 decision 4 makes a round-write failure a warning. Return the new Hash with a warning (exit 0 or a distinct code), and test it by making only the artifact path unwritable.
- **BR-11** [Important] `owned-artifact-removal-on-owner-death` (kind: defect) FILE.lock written by oplock.lua has no removal path after a pane crash; agent applies then exit 3 indefinitely
  oplock.lua removes the lock only on send or VimLeave, and treats any lock it did not take in this process as someone else's: take returns false and drop never removes it. After an nvim crash or SIGKILL, the lock stays in the user's repo, every pair review apply refuses with operator-lock, and the operator is never told. Record an owner (pid or tag) in the lock and reclaim it when that owner is dead or is this same review, or notify the operator; test the crash-leftover case.
- **BR-12** [Important] `plan-core-concepts-drift` (kind: defect) Plan's "M2 as built" commits readiness to ignore an untracked FILE.lock with a pinning test; neither exists
  This is the 2nd finding in family plan-core-concepts-drift. The rule for the class: every "M2 as built" commitment is either in the code with a test or listed in a Revisions entry as dropped or deferred; sweep the whole section, including the planned review-loop-test/fake-review-agent rewrite that review-applied-round-test.sh replaced. reviewcmd/run.go:277 still uses a plain `git status --porcelain`, so a leftover FILE.lock makes the repo "interact" instead of "new".
- **BR-13** [Minor] `round-watch-delivery` (kind: defect) round_watch marks a round seen before on_applied_round runs, so a refused reload (modified buffer) loses its decorations for good
  watch sets seen = result_hash before calling cb; on_applied_round returns false on a modified buffer and the round is never re-delivered.

## Round 6 — 2026-10-18T06:23:21-07:00 (claude) — BLOCKED

### Disposed

- BR-9 — addressed — round.go answersSideDoc now requires an exact swap; TestProposeLinkAnswerCannotCarryARewrite and the conformance fixture fail when round.go is reverted.
- BR-10 — addressed — applyfile.go returns Hash, Round, Body and Warning; TestApplyFileRoundWriteFailureWarns fails when the warning is reverted to return err.
- BR-11 — addressed — oplock records a pid and reclaims a dead owner at start and on take; round_watch_test covers dead, live and own-pid locks. EPERM edge noted as a new Minor.
- BR-12 — addressed — cleanBesidesLock is wired into both porcelain checks with a helper test, and the Revisions entry sweeps M2 as built. Gaps in quoting and call-site coverage are raised as a new finding.
- BR-13 — addressed — watch marks a round seen only when cb does not return false; the redelivery test covers it. A side effect is raised as a new Minor.

### Raised

- **BR-14** [Important] `git-porcelain-path-parsing` (kind: defect) cleanBesidesLock misses the lock for documents whose path git quotes, and readiness wiring is untested
  run.go:257 compares against an unquoted "?? rel.lock" line, but porcelain v1 prints "?? \"docs/my doc.md.lock\"" for names with spaces or non-ASCII characters (reproduced), so readiness still reports such repos dirty. Fix: use git status --porcelain -z and compare raw entries. Add a RunReadiness or gatherGitFacts test with a lock-only status, because reverting the call site keeps all tests green.
- **BR-15** [Minor] `round-watch-delivery` (kind: defect) A declined round warns every 100ms and is redelivered without checking the file still hashes to result_hash
  This is the 2nd finding in family round-watch-delivery. The rule: the pane acts on a round only while the file on disk hashes to round.result_hash, and it reports each declined round once. Put the hash check in decide, and debounce the notification per result_hash.
- **BR-16** [Minor] `docs-gate-new-surface` (kind: defect) The apply reply's Warning field and the lock's pid/reclaim are not documented in the couch SKILL or the atlas
  This is the 2nd finding in family docs-gate-new-surface. The rule: any change to the apply reply or lock contract updates SKILL.md and atlas/review-workbench.md in the same commit. The agent must learn that Warning means "commit Body now, no highlight".
- **BR-17** [Minor] `owned-artifact-removal-on-owner-death` (kind: defect) oplock alive() treats EPERM as dead, so it reclaims a lock owned by a live process of another user
  vim.uv.kill returns nil,"EPERM" for a live process the caller cannot signal. Only ESRCH should count as dead.

## Round 7 — 2026-10-18T06:31:13-07:00 (claude) — passed

### Disposed

- BR-14 — addressed — run.go gatherGitFacts/prepare use status --porcelain -z with NUL split; TestReadinessTreatsTheReviewLockAsClean fails on revert of call site or parser; space-path case added to the unit table.
- BR-15 — not-addressed — Stale-hash guard is in place and pinned by 'a round the file no longer matches is not delivered'; the per-result_hash warn-once in init.lua on_applied_round has no test, and decide's current_hash parameter is unused in production (watch compares inline).
- BR-16 — addressed — SKILL.md documents Warning means commit Body anyway; atlas/review-workbench.md documents Warning, pid lock owner, ESRCH reclaim, and readiness ignoring the lock via porcelain -z.
- BR-17 — not-addressed — alive() now treats only ESRCH as dead, and a manual probe confirms reclaim leaves a pid-1 (EPERM) lock; but no test pins it, since the live-owner case uses the signalable ppid, so reverting stays green. Add a pid 1 lock case.

## Open findings

- **BR-15** [Minor] `round-watch-delivery` (kind: defect) A declined round warns every 100ms and is redelivered without checking the file still hashes to result_hash
- **BR-17** [Minor] `owned-artifact-removal-on-owner-death` (kind: defect) oplock alive() treats EPERM as dead, so it reclaims a lock owned by a live process of another user
