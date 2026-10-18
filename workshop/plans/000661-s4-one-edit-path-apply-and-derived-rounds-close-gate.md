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

## Open findings

(none — every finding has been disposed)
