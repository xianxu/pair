---
gate: boundary-review
issue: 726
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: fb70bc95-dc2b-40bb-bcb8-5bd1af5e4792
            repository: github.com/xianxu/pair
            issue_id: "000726"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/879fd0b1a3c18124
            reviewed_head: 4dd0a6107ef9867aebde532f2027b10b86f72473
            slot: pair:2
            requested_at: "2026-10-20T15:25:14.797205Z"
          phase: finished
          pid: 78181
          started_at: "2026-10-20T15:25:14.931735Z"
          finished_at: "2026-10-20T15:33:43.749211Z"
          session_id: da2e0f22-e2d2-4353-8e81-15232ab15738
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 4dd0a6107ef9867aebde532f2027b10b86f72473
        tracker_head: 03075530c943bfc5a8646ca050c4234f68c4c014
        branch_point: e2ac2a19f253961b715b6edf104ca82d72b5c40a
        since: "2026-10-20T07:54:36-07:00"
        until: "2026-10-20T08:24:40-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T07:54:36-07:00"
              end: "2026-10-20T08:24:40-07:00"
              began: "2026-10-20T07:54:36-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 4221e67d-3549-4c0f-bf5c-bf5d3cf13458
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 77c11275a0674c1f1ae23c7ca50df7c1e8883f40a25b65facbb7f39748efafb9
              started: "2026-10-20T15:10:15.104Z"
              spawned: false
              bytes: 401871
              windows:
                - start: "2026-10-20T15:10:15.104Z"
                  end: "2026-10-20T08:24:40-07:00"
            - harness: claude
              id: 6925e360-3a18-45ee-b050-adbde3ed67ca
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf80fecf6c2d10f1483be7a4fae3b7a2dfb225e693b6ed7e3ad997fd6ef3668c
              started: "2026-10-20T15:09:56.095Z"
              spawned: false
              bytes: 486488
              windows:
                - start: "2026-10-20T15:09:56.095Z"
                  end: "2026-10-20T08:24:40-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: b41c57e39d272c9e306b13dccd58634ceab866736ef60865ee2f533486c55c9b
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 55223201
              windows:
                - start: "2026-10-20T07:54:36-07:00"
                  end: "2026-10-20T08:24:40-07:00"
            - harness: claude
              id: a8921d8e-ff5d-448d-bbbe-7c4096d6664b
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 6fb5337ed3171a0a204a700171391668aa62f2d24fccd650aa0c8a0867b8ea60
              started: "2026-10-20T15:06:32.044Z"
              spawned: false
              bytes: 622144
              windows:
                - start: "2026-10-20T15:06:32.044Z"
                  end: "2026-10-20T08:24:40-07:00"
            - harness: claude
              id: dd2ee03c-3bb0-40c3-8aec-7153c858d9f0
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 7004d633528a081b3018685616f0dbb291cfa1fc1f6fe3791036196047975501
              started: "2026-10-20T15:08:59.765Z"
              spawned: false
              bytes: 564150
              windows:
                - start: "2026-10-20T15:08:59.765Z"
                  end: "2026-10-20T08:24:40-07:00"
      "n": 1
      timestamp: "2026-10-20T08:33:44-07:00"
      dispatched: "2026-10-20T08:25:12-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: Cold target whose resolution fails after the budget drops the error; retries see only "warming" forever
          detail: hub_target_memo.go Warm caches only successes. A failed flight is deleted, so each retry starts a new resolution and, past the 1.5 s budget, refuses with "warming; retry" and never the real error (e.g. slot candidate limit, host replaced, sdlc failure). Remember the last failure briefly and return it to the next Warm with its phase; add a test where the resolver fails after the budget.
          family: refusal-must-carry-reason
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: handleHubMove still runs one fresh Discover per named slot synchronously, with no admission budget
          detail: hub_move.go:84-90 calls authority.hubTarget (memo.Fresh, a full Discover of about 1 s) per comma-separated slot. A two-slot move goes past the 2 s exchange and gets the same undiagnosed client deadline this issue fixes for lifecycle ops (ARCH-PURPOSE sibling). Either add a budget and warm sources with a fresh re-proof in the move job, or record it as a non-goal in the plan Revisions.
          family: admission-sync-discover-over-exchange
          round: 1
        - kind: unverifiable
          id: BR-3
          severity: Minor
          title: couchcore Discover concurrency and replaced-host tests could not be run in the review sandbox
          detail: 'Ran go test -race -run TestDiscover ./cmd/internal/couchcore/. The .sdlc-review git wrapper refused git init under GOTMPDIR=/Volumes/TestScratch/tmp/pair-slot2 ("outside the reviewer''s directories"). The pre-existing TestDiscoverUnionsNumberSources fails the same way. Host evidence: a passing -race run of TestDiscoverVerifiesSlotsConcurrently and TestDiscoverStillFailsWholeInventoryOnReplacedHost, plus a red run with the SameFile fence removed.'
          family: review-env-git-confinement
          round: 1
        - kind: defect
          id: BR-4
          severity: Minor
          title: Discover fan-out and per-reference background resolutions have no small concurrency cap
          detail: Rows run as wide as the row count (up to 128, each spawning sdlc and two git calls), and every distinct target spelling starts its own lifetime-scoped Discover. A semaphore of 8-16 would bound the burst on a co-tenanted host. The concurrency test's p4 > 4 check only re-states the row count.
          family: bounded-fanout
          round: 1
        - kind: defect
          id: BR-5
          severity: Minor
          title: Memo is keyed by raw r.Target, not the "canonical target reference" the plan states
          detail: Aliases and prefixes get separate entries and separate in-flight resolutions. Harmless, but either canonicalize the key or revise the plan text.
          family: plan-code-drift
          round: 1
        - kind: defect
          id: BR-6
          severity: Minor
          title: Admission closure repeats the guard closure and differs only in where origin and target come from
          detail: hub_guards.go:222-242 vs 128-150 (ARCH-DRY). One helper that takes the origin and target sources would keep the two checks from drifting apart.
          family: duplicated-guard-structure
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: fcc001b1-7b4d-4a27-90af-4da7c40ff033
            repository: github.com/xianxu/pair
            issue_id: "000726"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/6fed915de3c23228
            reviewed_head: 906f1c91c30bfdb59be27fff15b74608eb986bb4
            slot: pair:2
            requested_at: "2026-10-20T15:42:08.65861Z"
          phase: finished
          pid: 98973
          started_at: "2026-10-20T15:42:08.751383Z"
          finished_at: "2026-10-20T15:54:14.23728Z"
          session_id: aef88984-fb13-4d53-abe2-97fcf6ef66bd
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 906f1c91c30bfdb59be27fff15b74608eb986bb4
        tracker_head: 03075530c943bfc5a8646ca050c4234f68c4c014
        branch_point: e2ac2a19f253961b715b6edf104ca82d72b5c40a
        since: "2026-10-20T07:54:36-07:00"
        until: "2026-10-20T08:41:40-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T07:54:36-07:00"
              end: "2026-10-20T08:41:40-07:00"
              began: "2026-10-20T07:54:36-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 4221e67d-3549-4c0f-bf5c-bf5d3cf13458
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 77c11275a0674c1f1ae23c7ca50df7c1e8883f40a25b65facbb7f39748efafb9
              started: "2026-10-20T15:10:15.104Z"
              spawned: false
              bytes: 401871
              windows:
                - start: "2026-10-20T15:10:15.104Z"
                  end: "2026-10-20T08:41:40-07:00"
            - harness: claude
              id: 6925e360-3a18-45ee-b050-adbde3ed67ca
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf80fecf6c2d10f1483be7a4fae3b7a2dfb225e693b6ed7e3ad997fd6ef3668c
              started: "2026-10-20T15:09:56.095Z"
              spawned: false
              bytes: 486488
              windows:
                - start: "2026-10-20T15:09:56.095Z"
                  end: "2026-10-20T08:41:40-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: f66e1f3b8c40097feb508e45c7d04f20bd59a6aad366b2d1906b3720c65002b9
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 55590918
              windows:
                - start: "2026-10-20T07:54:36-07:00"
                  end: "2026-10-20T08:41:40-07:00"
            - harness: claude
              id: a8921d8e-ff5d-448d-bbbe-7c4096d6664b
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 6fb5337ed3171a0a204a700171391668aa62f2d24fccd650aa0c8a0867b8ea60
              started: "2026-10-20T15:06:32.044Z"
              spawned: false
              bytes: 622144
              windows:
                - start: "2026-10-20T15:06:32.044Z"
                  end: "2026-10-20T08:41:40-07:00"
            - harness: claude
              id: dd2ee03c-3bb0-40c3-8aec-7153c858d9f0
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 7004d633528a081b3018685616f0dbb291cfa1fc1f6fe3791036196047975501
              started: "2026-10-20T15:08:59.765Z"
              spawned: false
              bytes: 564150
              windows:
                - start: "2026-10-20T15:08:59.765Z"
                  end: "2026-10-20T08:41:40-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T08:25:12-07:00"
              finalized: "2026-10-20T08:33:44-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-20T08:54:17-07:00"
      dispatched: "2026-10-20T08:42:07-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Failures remembered 30 s (hub_target_memo.go:79-81,97-100); TestHubTargetMemoRemembersALateFailure goes red when Fresh stops remembering errors.
          round: 2
        - id: BR-2
          disposition: addressed
          note: handleHubMove admits on warmHubTarget under slotAdmissionBudget and prepare re-resolves fresh; test red with fresh admission (1 resolution) and with the prepare check disabled.
          round: 2
        - id: BR-3
          disposition: withdrawn
          note: With GOTMPDIR inside .sdlc-review/tmp, TestDiscoverVerifiesSlotsConcurrently, TestDiscoverStillFailsWholeInventoryOnReplacedHost and TestDiscoverUnionsNumberSources pass under -race.
          round: 2
        - id: BR-4
          disposition: addressed
          note: slotVerifyConcurrency=8 gate plus hubTargetMaxFlights=8; the 10-slot peak check and the flight-cap test each go red when the cap is removed.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: hubTargetKey is identity on every accepted input, so :N and repo:N still key separately and the atlas claim "keys are canonical repo:N" is wrong; drop the helper and fix the text, or really canonicalize.
          round: 2
        - id: BR-6
          disposition: addressed
          note: checkHubSlotOperation with hubSlotSources serves both guard and admission (hub_guards.go:171-201); existing admission and guard tests pass.
          round: 2
      findings:
        - kind: defect
          id: BR-7
          severity: Important
          title: TestDiscoverVerifiesSlotsConcurrently asserts elapsed time and fails under host load
          detail: slotcatalog_concurrent_test.go:74-77 fails when 4-slot wall time is at least 3/4 of serial; it failed in 1 of 3 isolated -race runs ("took 454ms, close to the serial 600ms") and in the full suite at load ~57. Peak-concurrency counts already prove concurrency; delete the elapsed-time check or use a barrier.
          family: wall-clock-test-oracle
          round: 2
        - kind: defect
          id: BR-8
          severity: Minor
          title: hubReceiptFilter waits on a cold memo resolution with no budget
          detail: '2nd finding in this family. Rule: every request-path read of a target slot must be bounded by the admission budget, so Warm should cap its own wait at slotAdmissionBudget rather than relying on each caller. hub_guards.go:249 calls warmHubTarget on the raw request ctx for operation-status polling.'
          family: admission-sync-discover-over-exchange
          round: 2
        - kind: defect
          id: BR-9
          severity: Minor
          title: Plan design and state table not revised for the round-2 behavior changes
          detail: '2nd finding in this family. Rule: any behavior change made in a review round updates the Plan design and state table (or a Revisions entry) in the same commit, not only the Log. The Plan still says Discover is bounded by MaxSlotCandidates; it lacks the 30 s remembered-failure state and the move prepare re-check.'
          family: plan-code-drift
          round: 2
        - kind: defect
          id: BR-10
          severity: Minor
          title: Memo stamps completion time, so an older in-flight resolution can overwrite a newer Fresh result
          detail: remember() records now() when the resolution finishes; a background flight started before a prepare-time Fresh can land after it and overwrite it. The queued guard re-proves before any effect, so the damage is a wrong admission answer for up to 30 s or 10 min. Stamp the observation start and keep the newer entry.
          family: observation-ordering-stamp
          round: 2
        - kind: defect
          id: BR-11
          severity: Minor
          title: hubTargetMemo entries are never deleted; failures for arbitrary target strings now accumulate
          detail: Entries expire logically but stay in the map for the supervisor's lifetime, and this round adds error entries for any reference a caller sends. Delete aged entries on lookup or sweep them, and correct the plan's ARCH-FUNERAL note.
          family: durable-cache-names-removal
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: ce2f22b6-150a-483b-b9e4-187226fefd93
            repository: github.com/xianxu/pair
            issue_id: "000726"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/ae32fa2026898ccb
            reviewed_head: 5ab066922b8fcd0ffbc58fb4d83e6d420575a72c
            slot: pair:2
            requested_at: "2026-10-20T16:00:31.437011Z"
          phase: finished
          pid: 98682
          started_at: "2026-10-20T16:00:31.52907Z"
          finished_at: "2026-10-20T16:04:13.321601Z"
          session_id: 38388f12-42cc-4551-86be-c57037497a57
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 5ab066922b8fcd0ffbc58fb4d83e6d420575a72c
        tracker_head: 6bf2737669f02b28c392a64f3a1c5a0277030234
        branch_point: e2ac2a19f253961b715b6edf104ca82d72b5c40a
        since: "2026-10-20T07:54:36-07:00"
        until: "2026-10-20T09:00:00-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T07:54:36-07:00"
              end: "2026-10-20T09:00:00-07:00"
              began: "2026-10-20T07:54:36-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 4221e67d-3549-4c0f-bf5c-bf5d3cf13458
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 77c11275a0674c1f1ae23c7ca50df7c1e8883f40a25b65facbb7f39748efafb9
              started: "2026-10-20T15:10:15.104Z"
              spawned: false
              bytes: 401871
              windows:
                - start: "2026-10-20T15:10:15.104Z"
                  end: "2026-10-20T09:00:00-07:00"
            - harness: claude
              id: 6925e360-3a18-45ee-b050-adbde3ed67ca
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf80fecf6c2d10f1483be7a4fae3b7a2dfb225e693b6ed7e3ad997fd6ef3668c
              started: "2026-10-20T15:09:56.095Z"
              spawned: false
              bytes: 486488
              windows:
                - start: "2026-10-20T15:09:56.095Z"
                  end: "2026-10-20T09:00:00-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 0e9e7222bbc4760c9d353d9e68415c564a3d0f7ff89cbfede129c48ced78f3fc
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 55885901
              windows:
                - start: "2026-10-20T07:54:36-07:00"
                  end: "2026-10-20T09:00:00-07:00"
            - harness: claude
              id: a8921d8e-ff5d-448d-bbbe-7c4096d6664b
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 6fb5337ed3171a0a204a700171391668aa62f2d24fccd650aa0c8a0867b8ea60
              started: "2026-10-20T15:06:32.044Z"
              spawned: false
              bytes: 622144
              windows:
                - start: "2026-10-20T15:06:32.044Z"
                  end: "2026-10-20T09:00:00-07:00"
            - harness: claude
              id: dd2ee03c-3bb0-40c3-8aec-7153c858d9f0
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 7004d633528a081b3018685616f0dbb291cfa1fc1f6fe3791036196047975501
              started: "2026-10-20T15:08:59.765Z"
              spawned: false
              bytes: 564150
              windows:
                - start: "2026-10-20T15:08:59.765Z"
                  end: "2026-10-20T09:00:00-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T08:25:12-07:00"
              finalized: "2026-10-20T08:33:44-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T08:42:07-07:00"
              finalized: "2026-10-20T08:54:17-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-20T09:04:14-07:00"
      dispatched: "2026-10-20T09:00:29-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-5
          disposition: addressed
          note: hubTargetKey (hub_target_memo.go:39) makes repo:N keys canonical via ParseWorkspaceReference; pinned by TestHubTargetMemoKeysCanonicallyAndCapsFlights (passes).
          round: 3
        - id: BR-7
          disposition: addressed
          note: Elapsed-time check removed; rendezvousIO waits until min(slots,8) slot probes are in flight together, which a serial pass can never reach. Checked against verifyHost (one probe per slot host); couchcore can't run here because of sandbox git limits.
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: All three warmHubTarget callers now add their own slotAdmissionBudget, but Warm doesn't cap its own wait as the rule asked, and no test fails without the receipt-filter budget (hub_guards.go:250).
          round: 3
        - id: BR-9
          disposition: addressed
          note: Plan Revisions now lists the 30 s failure age, the move prepare re-check, start-time ordering, the 256 cap, 8-wide fan-out and two new state-table rows.
          round: 3
        - id: BR-10
          disposition: addressed
          note: remember() stamps the start time and keeps the newer entry; undoing the old.at.After(started) check turns TestHubTargetMemoOrdersByStartAndSweeps red (checked here).
          round: 3
        - id: BR-11
          disposition: addressed
          note: remember() deletes expired entries and evicts the oldest past hubTargetMaxEntries=256; the test checks both the cap and the expiry sweep.
          round: 3
      findings:
        - kind: defect
          id: BR-12
          severity: Minor
          title: A failed resolution that runs 30 s or more is stored already expired, so admission says "warming" forever again
          detail: '2nd finding in this family. Rule: an error from a finished resolution must reach the next admission for the full hubTargetFailureAge, counted from when it finished. Since BR-10, remember() stamps start time (hub_target_memo.go:158) and Warm and the sweep age failures from it (:102, :145). With 5 s ProbeTimeout probes across 8-wide waves of rows, a hung sdlc pushes Discover past 30 s, every retry starts a new flight and refuses "warming", and the cause never shows. Fix: keep the start time for ordering and add a finish time for failure expiry; test with a fake-clock resolver slower than hubTargetFailureAge.'
          family: refusal-must-carry-reason
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: 5ab066922b8fcd0ffbc58fb4d83e6d420575a72c
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 77f97969-8591-4d37-92ae-7b149fcf94a7
            repository: github.com/xianxu/pair
            issue_id: "000726"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/afb06a2400ca1fd2
            reviewed_head: 2e9772fb0bdc2e99cb1d9379591101ad10f1d847
            slot: pair:2
            requested_at: "2026-10-20T16:14:54.824962Z"
          phase: finished
          pid: 36060
          started_at: "2026-10-20T16:14:54.981843Z"
          finished_at: "2026-10-20T16:19:42.155721Z"
          session_id: 1c8649d8-b80d-49c3-b031-050baad5475f
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 2e9772fb0bdc2e99cb1d9379591101ad10f1d847
        tracker_head: df13eb8fedf40744fb0c47f289bc8c88a2959702
        branch_point: e2ac2a19f253961b715b6edf104ca82d72b5c40a
        since: "2026-10-20T07:54:36-07:00"
        until: "2026-10-20T09:14:16-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T07:54:36-07:00"
              end: "2026-10-20T09:14:16-07:00"
              began: "2026-10-20T07:54:36-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 4221e67d-3549-4c0f-bf5c-bf5d3cf13458
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 77c11275a0674c1f1ae23c7ca50df7c1e8883f40a25b65facbb7f39748efafb9
              started: "2026-10-20T15:10:15.104Z"
              spawned: false
              bytes: 401871
              windows:
                - start: "2026-10-20T15:10:15.104Z"
                  end: "2026-10-20T09:14:16-07:00"
            - harness: claude
              id: 6925e360-3a18-45ee-b050-adbde3ed67ca
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf80fecf6c2d10f1483be7a4fae3b7a2dfb225e693b6ed7e3ad997fd6ef3668c
              started: "2026-10-20T15:09:56.095Z"
              spawned: false
              bytes: 486488
              windows:
                - start: "2026-10-20T15:09:56.095Z"
                  end: "2026-10-20T09:14:16-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 53e14d3dc5c77bcbb6af8d392c58c0df22ce496d942f833bac5e99c618428296
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 56356139
              windows:
                - start: "2026-10-20T07:54:36-07:00"
                  end: "2026-10-20T09:14:16-07:00"
            - harness: claude
              id: a8921d8e-ff5d-448d-bbbe-7c4096d6664b
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 6fb5337ed3171a0a204a700171391668aa62f2d24fccd650aa0c8a0867b8ea60
              started: "2026-10-20T15:06:32.044Z"
              spawned: false
              bytes: 622144
              windows:
                - start: "2026-10-20T15:06:32.044Z"
                  end: "2026-10-20T09:14:16-07:00"
            - harness: claude
              id: dd2ee03c-3bb0-40c3-8aec-7153c858d9f0
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 7004d633528a081b3018685616f0dbb291cfa1fc1f6fe3791036196047975501
              started: "2026-10-20T15:08:59.765Z"
              spawned: false
              bytes: 564150
              windows:
                - start: "2026-10-20T15:08:59.765Z"
                  end: "2026-10-20T09:14:16-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T08:25:12-07:00"
              finalized: "2026-10-20T08:33:44-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T08:42:07-07:00"
              finalized: "2026-10-20T08:54:17-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T09:00:29-07:00"
              finalized: "2026-10-20T09:04:14-07:00"
              finalizing: true
      "n": 4
      timestamp: "2026-10-20T09:19:42-07:00"
      dispatched: "2026-10-20T09:14:52-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-13
          severity: Important
          title: Done-when and Plan Tests fixture (over-cap inventory, oversized transcript, counted reads) not delivered nor revised
          detail: '3rd finding in this family. Rule: before a milestone closes, every Plan Tests bullet and Done-when test it claims is delivered as written or rewritten in Revisions in the same commit. recovery_skip_candidates_test.go uses one 456-byte transcript and infers no reads from empty Ranked; the inference is sound (reverting the skip branch goes red) but the stated acceptance fixture is absent. Add an oversized-transcript plus counting-reader Proven RecoverSelection test, or record the substitution in Revisions.'
          family: plan-code-drift
          round: 4
        - kind: defect
          id: BR-14
          severity: Minor
          title: Capture-identity fence when Source is nil is duplicated in ReadRequest (lines 199-206 and 224-230)
          detail: '2nd finding in this family. Rule: one guard, one site; variants pass in what differs rather than copying it. Wrap the native scan in if !q.SkipCandidates and fall through to the single shared fence (ARCH-DRY).'
          family: duplicated-guard-structure
          round: 4
        - kind: defect
          id: BR-15
          severity: Minor
          title: resume-candidates says an incomplete scan affects only inferred choices even when Incomplete came from the capture fence
          detail: resume_candidates.go:123-126 keys on Incomplete and Proven != nil, but a failed validateRecoveryCaptureSnapshot ("recovery screen changed") also sets Incomplete and does affect the confirmed resume via Preserve. Condition the wording on the scan being the cause, or soften it.
          family: refusal-must-carry-reason
          round: 4
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 32fb8751-afe8-4b86-8bb5-38c1210839f3
            repository: github.com/xianxu/pair
            issue_id: "000726"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/0e5c1334d2d15431
            reviewed_head: b5995e632b94b90d31ac92b9be709134a17d63aa
            slot: pair:2
            requested_at: "2026-10-20T16:24:10.198475Z"
          phase: finished
          pid: 79661
          started_at: "2026-10-20T16:24:10.309067Z"
          finished_at: "2026-10-20T16:26:12.179266Z"
          session_id: 7edb68f7-08bf-4dbd-bfc1-ef31cea8a1cf
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: b5995e632b94b90d31ac92b9be709134a17d63aa
        tracker_head: df13eb8fedf40744fb0c47f289bc8c88a2959702
        branch_point: e2ac2a19f253961b715b6edf104ca82d72b5c40a
        since: "2026-10-20T07:54:36-07:00"
        until: "2026-10-20T09:23:44-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T07:54:36-07:00"
              end: "2026-10-20T09:23:44-07:00"
              began: "2026-10-20T07:54:36-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 4221e67d-3549-4c0f-bf5c-bf5d3cf13458
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 77c11275a0674c1f1ae23c7ca50df7c1e8883f40a25b65facbb7f39748efafb9
              started: "2026-10-20T15:10:15.104Z"
              spawned: false
              bytes: 401871
              windows:
                - start: "2026-10-20T15:10:15.104Z"
                  end: "2026-10-20T09:23:44-07:00"
            - harness: claude
              id: 6925e360-3a18-45ee-b050-adbde3ed67ca
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf80fecf6c2d10f1483be7a4fae3b7a2dfb225e693b6ed7e3ad997fd6ef3668c
              started: "2026-10-20T15:09:56.095Z"
              spawned: false
              bytes: 486488
              windows:
                - start: "2026-10-20T15:09:56.095Z"
                  end: "2026-10-20T09:23:44-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: d274f36490fec49f1d97efb91638fde75be0872ee892ee946c587cea086342c2
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 56620787
              windows:
                - start: "2026-10-20T07:54:36-07:00"
                  end: "2026-10-20T09:23:44-07:00"
            - harness: claude
              id: a8921d8e-ff5d-448d-bbbe-7c4096d6664b
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 6fb5337ed3171a0a204a700171391668aa62f2d24fccd650aa0c8a0867b8ea60
              started: "2026-10-20T15:06:32.044Z"
              spawned: false
              bytes: 622144
              windows:
                - start: "2026-10-20T15:06:32.044Z"
                  end: "2026-10-20T09:23:44-07:00"
            - harness: claude
              id: dd2ee03c-3bb0-40c3-8aec-7153c858d9f0
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 7004d633528a081b3018685616f0dbb291cfa1fc1f6fe3791036196047975501
              started: "2026-10-20T15:08:59.765Z"
              spawned: false
              bytes: 564150
              windows:
                - start: "2026-10-20T15:08:59.765Z"
                  end: "2026-10-20T09:23:44-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 1.3874166666666667
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T08:25:12-07:00"
              finalized: "2026-10-20T08:33:44-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T08:42:07-07:00"
              finalized: "2026-10-20T08:54:17-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T09:00:29-07:00"
              finalized: "2026-10-20T09:04:14-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-20T09:14:52-07:00"
              finalized: "2026-10-20T09:19:42-07:00"
              finalizing: false
      "n": 5
      timestamp: "2026-10-20T09:26:13-07:00"
      dispatched: "2026-10-20T09:24:08-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-13
          disposition: addressed
          note: TestRecoverySkipCandidatesOnAnOversizedNativeStore builds a store over RecoveryMetadataLimit with an over-RecoveryRecordLimit record and counts bytes via BytesRead against a screen-only read; Revisions records the counting substitution; reverting the SkipCandidates guard turns the test red (verified).
          round: 5
        - id: BR-14
          disposition: addressed
          note: The scan moved into collectCandidates behind !q.SkipCandidates; one Source-nil validateRecoveryCaptureSnapshot site remains at recovery_evidence_os.go:202-208.
          round: 5
        - id: BR-15
          disposition: addressed
          note: resume_candidates.go:126 now says only that the confirmed choice comes from the established binding, which holds whatever caused Incomplete; pinned by TestResumeCandidatesCLIIncompleteScanSparesTheConfirmedChoice. Atlas and plan restatements are raised separately.
          round: 5
      findings:
        - kind: defect
          id: BR-16
          severity: Minor
          title: Atlas and the plan M2 bullet still claim an incomplete scan affects only the inferred choices
          detail: 'atlas/couch.md:83-84 and the issue''s M2 bullet keep the wording BR-15 removed from resume_candidates.go:126. This is the 4th finding in this family. Rule: a change to user-visible wording or behavior sweeps every atlas passage and Plan bullet that restates it in the same commit (grep the old phrase).'
          family: plan-code-drift
          round: 5
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      reviewed: b5995e632b94b90d31ac92b9be709134a17d63aa
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 357bd11f-8d43-4d2f-b999-75401f4c938e
            repository: github.com/xianxu/pair
            issue_id: "000726"
            recipe: milestone-review
            round: 6
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/029b9f6d5e0fd8a1
            reviewed_head: dd8a86665c176e52ab7195a431622aada3b1ef06
            slot: pair:2
            requested_at: "2026-10-20T16:50:23.206698Z"
          phase: finished
          pid: 50256
          started_at: "2026-10-20T16:50:23.41367Z"
          finished_at: "2026-10-20T16:52:58.921833Z"
          session_id: aac39a45-9f3f-4335-8d47-bd8400a32dc7
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: dd8a86665c176e52ab7195a431622aada3b1ef06
        tracker_head: 1dec9fc5a441562b94a6c70120460d8af53f735a
        branch_point: e2ac2a19f253961b715b6edf104ca82d72b5c40a
        since: "2026-10-20T07:54:36-07:00"
        until: "2026-10-20T09:30:56-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T07:54:36-07:00"
              end: "2026-10-20T09:30:56-07:00"
              began: "2026-10-20T07:54:36-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 4221e67d-3549-4c0f-bf5c-bf5d3cf13458
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 77c11275a0674c1f1ae23c7ca50df7c1e8883f40a25b65facbb7f39748efafb9
              started: "2026-10-20T15:10:15.104Z"
              spawned: false
              bytes: 401871
              windows:
                - start: "2026-10-20T15:10:15.104Z"
                  end: "2026-10-20T09:30:56-07:00"
            - harness: claude
              id: 6925e360-3a18-45ee-b050-adbde3ed67ca
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf80fecf6c2d10f1483be7a4fae3b7a2dfb225e693b6ed7e3ad997fd6ef3668c
              started: "2026-10-20T15:09:56.095Z"
              spawned: false
              bytes: 486488
              windows:
                - start: "2026-10-20T15:09:56.095Z"
                  end: "2026-10-20T09:30:56-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 77ce229ca85fb8da8e5c85cea59b7c229422a501badcdf99a6e4f7d77dcd545a
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 57059945
              windows:
                - start: "2026-10-20T07:54:36-07:00"
                  end: "2026-10-20T09:30:56-07:00"
            - harness: claude
              id: a8921d8e-ff5d-448d-bbbe-7c4096d6664b
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 6fb5337ed3171a0a204a700171391668aa62f2d24fccd650aa0c8a0867b8ea60
              started: "2026-10-20T15:06:32.044Z"
              spawned: false
              bytes: 622144
              windows:
                - start: "2026-10-20T15:06:32.044Z"
                  end: "2026-10-20T09:30:56-07:00"
            - harness: claude
              id: dd2ee03c-3bb0-40c3-8aec-7153c858d9f0
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 7004d633528a081b3018685616f0dbb291cfa1fc1f6fe3791036196047975501
              started: "2026-10-20T15:08:59.765Z"
              spawned: false
              bytes: 564150
              windows:
                - start: "2026-10-20T15:08:59.765Z"
                  end: "2026-10-20T09:30:56-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 1.3874166666666667
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T08:25:12-07:00"
              finalized: "2026-10-20T08:33:44-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T08:42:07-07:00"
              finalized: "2026-10-20T08:54:17-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T09:00:29-07:00"
              finalized: "2026-10-20T09:04:14-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-20T09:14:52-07:00"
              finalized: "2026-10-20T09:19:42-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-20T09:24:08-07:00"
              finalized: "2026-10-20T09:26:13-07:00"
              finalizing: true
      "n": 6
      timestamp: "2026-10-20T09:53:00-07:00"
      dispatched: "2026-10-20T09:50:20-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-8
          disposition: addressed
          note: Warm caps its own wait at slotAdmissionBudget (hub_target_memo.go:104-107); without it TestHubTargetMemoWarmCapsItsOwnWait fails after 30 s.
          round: 6
        - id: BR-12
          disposition: addressed
          note: Entries now carry started (ordering) and finished (expiry), and live() ages failures from finished; reverting to started turns TestHubTargetMemoAgesFailuresFromTheirEnd red.
          round: 6
        - id: BR-16
          disposition: not-addressed
          note: atlas/couch.md:83-85 and the issue M2 bullet (line 78) are fixed, but grepping the old phrase still finds the resume_candidates_test.go:194 doc comment ("affects only the inferred choices"); reword it.
          round: 6
      recipe: milestone-review
      isolation: fenced
      reviewed: dd8a86665c176e52ab7195a431622aada3b1ef06
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: df361ea6-3d73-4cc7-b2df-0900e748f8f9
            repository: github.com/xianxu/pair
            issue_id: "000726"
            recipe: fix-review
            round: 7
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/256b88f12cc8bd75
            reviewed_head: e75f282584c7efc19abebc5d6ab999a1b0148091
            slot: pair:2
            requested_at: "2026-10-20T18:29:16.820695Z"
          phase: finished
          pid: 42033
          started_at: "2026-10-20T18:29:16.923724Z"
          finished_at: "2026-10-20T18:30:27.376277Z"
          session_id: 01a15a93-2cfc-7fc1-a3cb-f76a6638a9ed
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: e75f282584c7efc19abebc5d6ab999a1b0148091
        tracker_head: 5ddf0fca7cb52c410bf8125b4cc1b8d98c6b0668
        branch_point: e2ac2a19f253961b715b6edf104ca82d72b5c40a
        since: "2026-10-20T07:54:36-07:00"
        until: "2026-10-20T11:28:29-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T07:54:36-07:00"
              end: "2026-10-20T11:28:29-07:00"
              began: "2026-10-20T07:54:36-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 4221e67d-3549-4c0f-bf5c-bf5d3cf13458
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 77c11275a0674c1f1ae23c7ca50df7c1e8883f40a25b65facbb7f39748efafb9
              started: "2026-10-20T15:10:15.104Z"
              spawned: false
              bytes: 401871
              windows:
                - start: "2026-10-20T15:10:15.104Z"
                  end: "2026-10-20T11:28:29-07:00"
            - harness: claude
              id: 6925e360-3a18-45ee-b050-adbde3ed67ca
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf80fecf6c2d10f1483be7a4fae3b7a2dfb225e693b6ed7e3ad997fd6ef3668c
              started: "2026-10-20T15:09:56.095Z"
              spawned: false
              bytes: 486488
              windows:
                - start: "2026-10-20T15:09:56.095Z"
                  end: "2026-10-20T11:28:29-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf660e568368fb1db224aadce1f646603c4640b4d65f7e7fc289cc7be4b5777e
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 57785294
              windows:
                - start: "2026-10-20T07:54:36-07:00"
                  end: "2026-10-20T11:28:29-07:00"
            - harness: claude
              id: a8921d8e-ff5d-448d-bbbe-7c4096d6664b
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 6fb5337ed3171a0a204a700171391668aa62f2d24fccd650aa0c8a0867b8ea60
              started: "2026-10-20T15:06:32.044Z"
              spawned: false
              bytes: 622144
              windows:
                - start: "2026-10-20T15:06:32.044Z"
                  end: "2026-10-20T11:28:29-07:00"
            - harness: claude
              id: dd2ee03c-3bb0-40c3-8aec-7153c858d9f0
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 7004d633528a081b3018685616f0dbb291cfa1fc1f6fe3791036196047975501
              started: "2026-10-20T15:08:59.765Z"
              spawned: false
              bytes: 564150
              windows:
                - start: "2026-10-20T15:08:59.765Z"
                  end: "2026-10-20T11:28:29-07:00"
            - harness: codex
              id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 30c846d6f8440363d497bbc0693a123d05402df12b0f7eda43d0f01e939c18cf
              started: "2026-10-20T18:21:51.812Z"
              spawned: false
              bytes: 1348363
              windows:
                - start: "2026-10-20T18:21:51.812Z"
                  end: "2026-10-20T11:28:29-07:00"
            - harness: codex
              id: 01a15a8c-f035-7473-97a1-e19202726aaa
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: 4d9812015e346d59312b45580af651c5334e3f2ce680968105bce4dae97318ff
              started: "2026-10-20T18:22:28.452Z"
              spawned: true
              bytes: 1023380
              windows:
                - start: "2026-10-20T18:22:28.452Z"
                  end: "2026-10-20T11:28:29-07:00"
        queue_waits:
            - attempt_id: df4e0de83a7b02852a727e603a53986e
              slot: pair:2
              record_digest: 1922c33a4000af22fad2e71748019ce8508f21663c0bee450c2e45ab4d6aef52
              start: "2026-10-20T16:31:08.064303Z"
              end: "2026-10-20T16:40:52.242939Z"
              queue_seconds: 584.178636
              raw_seconds: 584.178636
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 28.503233333333334
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T08:25:12-07:00"
              finalized: "2026-10-20T08:33:44-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T08:42:07-07:00"
              finalized: "2026-10-20T08:54:17-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T09:00:29-07:00"
              finalized: "2026-10-20T09:04:14-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-20T09:14:52-07:00"
              finalized: "2026-10-20T09:19:42-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-20T09:24:08-07:00"
              finalized: "2026-10-20T09:26:13-07:00"
              finalizing: true
            - dispatched: "2026-10-20T09:50:20-07:00"
              finalized: "2026-10-20T09:53:00-07:00"
              finalizing: true
      "n": 7
      timestamp: "2026-10-20T11:30:31-07:00"
      dispatched: "2026-10-20T11:29:15-07:00"
      agent: codex
      reviewer: codex (openai; author anthropic,openai)
      dispose:
        - id: BR-16
          disposition: addressed
          note: atlas/couch.md:83-85 and the issue's M2 bullet at line 79 correctly attribute the confirmed choice to the established binding, matching resume_candidates.go:123-126. The remaining stale test comment is corrected by this interdiff at resume_candidates_test.go:194-196; this is a prose-only correction.
          round: 7
      findings:
        - kind: defect
          id: BR-17
          severity: Critical
          title: scope changed after close — run a full `sdlc close`
          detail: The issue's new Revisions entry at line 105 introduces repository-wide single-flight discovery, startup prewarming, and Tier A batch acceptance beyond the outstanding wording finding. Executable changes in hub_target_memo.go:108-274, message_service.go:701-709, and slotcontext.go:203-222 implement that expansion. Run a full sdlc close covering the expanded scope; this narrow fix review cannot authorize it.
          family: boundary-review-scope
          round: 7
      recipe: fix-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: f58d3a93-3d1e-4e01-94e6-eb42bb68ca43
            repository: github.com/xianxu/pair
            issue_id: "000726"
            recipe: milestone-review
            round: 8
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/63fd352e96effb43
            reviewed_head: 06fde08625b18c4fbbfb7ab4e84d8dbf37a59524
            slot: pair:2
            requested_at: "2026-10-20T18:31:55.466403Z"
          phase: finished
          pid: 64696
          started_at: "2026-10-20T18:31:55.556003Z"
          finished_at: "2026-10-20T18:36:34.392281Z"
          session_id: 01a15a95-989b-7e30-b96e-d1102c354c12
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 06fde08625b18c4fbbfb7ab4e84d8dbf37a59524
        tracker_head: 5ddf0fca7cb52c410bf8125b4cc1b8d98c6b0668
        branch_point: e2ac2a19f253961b715b6edf104ca82d72b5c40a
        since: "2026-10-20T07:54:36-07:00"
        until: "2026-10-20T11:31:27-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T07:54:36-07:00"
              end: "2026-10-20T11:31:27-07:00"
              began: "2026-10-20T07:54:36-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 4221e67d-3549-4c0f-bf5c-bf5d3cf13458
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 77c11275a0674c1f1ae23c7ca50df7c1e8883f40a25b65facbb7f39748efafb9
              started: "2026-10-20T15:10:15.104Z"
              spawned: false
              bytes: 401871
              windows:
                - start: "2026-10-20T15:10:15.104Z"
                  end: "2026-10-20T11:31:27-07:00"
            - harness: claude
              id: 6925e360-3a18-45ee-b050-adbde3ed67ca
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf80fecf6c2d10f1483be7a4fae3b7a2dfb225e693b6ed7e3ad997fd6ef3668c
              started: "2026-10-20T15:09:56.095Z"
              spawned: false
              bytes: 486488
              windows:
                - start: "2026-10-20T15:09:56.095Z"
                  end: "2026-10-20T11:31:27-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf660e568368fb1db224aadce1f646603c4640b4d65f7e7fc289cc7be4b5777e
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 57785294
              windows:
                - start: "2026-10-20T07:54:36-07:00"
                  end: "2026-10-20T11:31:27-07:00"
            - harness: claude
              id: a8921d8e-ff5d-448d-bbbe-7c4096d6664b
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 6fb5337ed3171a0a204a700171391668aa62f2d24fccd650aa0c8a0867b8ea60
              started: "2026-10-20T15:06:32.044Z"
              spawned: false
              bytes: 622144
              windows:
                - start: "2026-10-20T15:06:32.044Z"
                  end: "2026-10-20T11:31:27-07:00"
            - harness: claude
              id: dd2ee03c-3bb0-40c3-8aec-7153c858d9f0
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 7004d633528a081b3018685616f0dbb291cfa1fc1f6fe3791036196047975501
              started: "2026-10-20T15:08:59.765Z"
              spawned: false
              bytes: 564150
              windows:
                - start: "2026-10-20T15:08:59.765Z"
                  end: "2026-10-20T11:31:27-07:00"
            - harness: codex
              id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 5801cd87dc7ea074217929f7f4e3b5b24bcec64e53373b602935d328450bcf57
              started: "2026-10-20T18:21:51.812Z"
              spawned: false
              bytes: 1580945
              windows:
                - start: "2026-10-20T18:21:51.812Z"
                  end: "2026-10-20T11:31:27-07:00"
            - harness: codex
              id: 01a15a8c-f035-7473-97a1-e19202726aaa
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: 4d9812015e346d59312b45580af651c5334e3f2ce680968105bce4dae97318ff
              started: "2026-10-20T18:22:28.452Z"
              spawned: true
              bytes: 1023380
              windows:
                - start: "2026-10-20T18:22:28.452Z"
                  end: "2026-10-20T11:31:27-07:00"
        queue_waits:
            - attempt_id: df4e0de83a7b02852a727e603a53986e
              slot: pair:2
              record_digest: 1922c33a4000af22fad2e71748019ce8508f21663c0bee450c2e45ab4d6aef52
              start: "2026-10-20T16:31:08.064303Z"
              end: "2026-10-20T16:40:52.242939Z"
              queue_seconds: 584.178636
              raw_seconds: 584.178636
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 28.503233333333334
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T08:25:12-07:00"
              finalized: "2026-10-20T08:33:44-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T08:42:07-07:00"
              finalized: "2026-10-20T08:54:17-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T09:00:29-07:00"
              finalized: "2026-10-20T09:04:14-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-20T09:14:52-07:00"
              finalized: "2026-10-20T09:19:42-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-20T09:24:08-07:00"
              finalized: "2026-10-20T09:26:13-07:00"
              finalizing: true
            - dispatched: "2026-10-20T09:50:20-07:00"
              finalized: "2026-10-20T09:53:00-07:00"
              finalizing: true
            - dispatched: "2026-10-20T11:29:15-07:00"
              finalized: "2026-10-20T11:30:31-07:00"
              finalizing: false
      "n": 8
      timestamp: "2026-10-20T11:36:35-07:00"
      dispatched: "2026-10-20T11:31:53-07:00"
      agent: codex
      reviewer: codex (openai; author anthropic,openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Failure entries retain the resolver error; TestHubTargetMemoRemembersALateFailure covers late completion and retry.
          round: 8
        - id: BR-2
          disposition: addressed
          note: Move target resolution uses Warm; TestHubMoveAdmissionUsesWarmSlotsAndRechecksFresh covers admission and fresh revalidation.
          round: 8
        - id: BR-3
          disposition: withdrawn
          note: Discover concurrency and replaced-host tests ran successfully under the race detector in this environment.
          round: 8
        - id: BR-4
          disposition: addressed
          note: Background flights and Discover verification have eight-worker caps, covered by focused race tests.
          round: 8
        - id: BR-5
          disposition: addressed
          note: Memo access and writes consistently use hubTargetKey.
          round: 8
        - id: BR-6
          disposition: addressed
          note: Admission and fresh guards share checkHubSlotOperation with injected sources.
          round: 8
        - id: BR-7
          disposition: addressed
          note: Discover concurrency uses an in-flight rendezvous rather than elapsed time as its success oracle.
          round: 8
        - id: BR-8
          disposition: addressed
          note: Receipt filtering bounds its memo wait; Warm also enforces its own budget.
          round: 8
        - id: BR-9
          disposition: addressed
          note: The issue Revisions records failure retention, move revalidation, caps, ordering, and additional visible states.
          round: 8
        - id: BR-10
          disposition: addressed
          note: remember rejects older-started results; late individual and repository completion regressions pass.
          round: 8
        - id: BR-11
          disposition: addressed
          note: Writes sweep expired entries and cap the map at 256; the sweep/cap regression passes.
          round: 8
        - id: BR-12
          disposition: addressed
          note: Failure expiry uses finished time; TestHubTargetMemoAgesFailuresFromTheirEnd passes.
          round: 8
        - id: BR-13
          disposition: addressed
          note: The oversized-store fixture exercises metadata and record limits, compares counted bytes, and passes with the routing regression.
          round: 8
        - id: BR-14
          disposition: addressed
          note: ReadRequest has one Source-nil capture fence after optional candidate collection.
          round: 8
        - id: BR-15
          disposition: addressed
          note: resume_candidates.go attributes the confirmed choice to the established binding without claiming all incomplete observations affect only inference.
          round: 8
        - id: BR-16
          disposition: addressed
          note: atlas/couch.md:83-85 and the Plan M2 bullet match the CLI wording and established-binding recovery path.
          round: 8
        - id: BR-17
          disposition: addressed
          note: This review inspected the complete pinned range, including repository batching, startup prewarm, production wiring, and Tier A acceptance. The issue Revisions at line 121 explicitly supersedes the original per-reference design.
          round: 8
      findings:
        - kind: defect
          id: BR-18
          severity: Critical
          title: The admission budget does not cover the complete admission path
          detail: hub_guards.go:162-168 starts the 1.5 s budget after handleSlotOperation has authenticated the caller, then cancels it before slotOps.handle. Earlier liveCaller reads can wait up to one second for the store lock (message_service.go:1256; couchcore/retention.go:299), while later queueTarget performs another repository-name read with a separate 500 ms allowance. A caller check that succeeds after 700 ms followed by a cold target resolution reaches the parent 2 s deadline before the inner budget expires; transport.go then drops the refusal. This is the 3rd finding in family admission-sync-discover-over-exchange. Earlier rounds fixed instances; establish one deadline for the entire lifecycle admission path, including authentication, target resolution, queue-key lookup, and enqueue, and sweep slot, move, and status paths against that rule (ARCH-CONSTRAINTS, ARCH-PURPOSE). Add a socket regression combining delayed successful authentication with blocked target resolution, plus delayed queue-name lookup.
          family: admission-sync-discover-over-exchange
          round: 8
        - kind: defect
          id: BR-19
          severity: Critical
          title: Repository warming bypasses sibling-directory precedence
          detail: hub_target_memo.go:119 resolves repository names directly against enrollment, then returns the batched slot at lines 135-146. The existing fresh resolver instead gives an exact sibling directory precedence over enrolled names and prefixes (couchcore/slotcontext.go:152-156), explicitly tested by TestExistingSiblingDirectoryBeatsPrefix. With an existing sibling parley and an enrolled parley.nvim, Warm("parley:1") can return parley.nvim:1 while Fresh resolves parley:1. If those repositories belong to different hubs, admission refuses a valid operation before fresh authority can correct the snapshot. Derive memo repository identity through the same reference-resolution rule, retaining batching after identity is established; cover cold and prewarmed references, including prefixes and aliases (ARCH-DRY, ARCH-PURPOSE).
          family: canonical-reference-resolution
          round: 8
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: a191279c-8a14-4a00-9ef3-28a0c290347d
            repository: github.com/xianxu/pair
            issue_id: "000726"
            recipe: milestone-review
            round: 9
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/87746dfcd6c986cc
            reviewed_head: 08e29b1498ae46a0e4fb5a7b3079ae1c4e974710
            slot: pair:2
            requested_at: "2026-10-20T18:49:56.945242Z"
          phase: finished
          pid: 83213
          started_at: "2026-10-20T18:49:57.097185Z"
          finished_at: "2026-10-20T18:56:27.696195Z"
          session_id: 01a15aa6-1a1e-7d42-b3d5-5c261787b47a
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 08e29b1498ae46a0e4fb5a7b3079ae1c4e974710
        tracker_head: c2a8a6f9c403ff8cba643df85ef2a6dd1d9e573c
        branch_point: e2ac2a19f253961b715b6edf104ca82d72b5c40a
        since: "2026-10-20T07:54:36-07:00"
        until: "2026-10-20T11:48:36-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T07:54:36-07:00"
              end: "2026-10-20T11:48:36-07:00"
              began: "2026-10-20T07:54:36-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 4221e67d-3549-4c0f-bf5c-bf5d3cf13458
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 77c11275a0674c1f1ae23c7ca50df7c1e8883f40a25b65facbb7f39748efafb9
              started: "2026-10-20T15:10:15.104Z"
              spawned: false
              bytes: 401871
              windows:
                - start: "2026-10-20T15:10:15.104Z"
                  end: "2026-10-20T11:48:36-07:00"
            - harness: claude
              id: 6925e360-3a18-45ee-b050-adbde3ed67ca
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf80fecf6c2d10f1483be7a4fae3b7a2dfb225e693b6ed7e3ad997fd6ef3668c
              started: "2026-10-20T15:09:56.095Z"
              spawned: false
              bytes: 486488
              windows:
                - start: "2026-10-20T15:09:56.095Z"
                  end: "2026-10-20T11:48:36-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: cf660e568368fb1db224aadce1f646603c4640b4d65f7e7fc289cc7be4b5777e
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 57785294
              windows:
                - start: "2026-10-20T07:54:36-07:00"
                  end: "2026-10-20T11:48:36-07:00"
            - harness: claude
              id: a8921d8e-ff5d-448d-bbbe-7c4096d6664b
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 6fb5337ed3171a0a204a700171391668aa62f2d24fccd650aa0c8a0867b8ea60
              started: "2026-10-20T15:06:32.044Z"
              spawned: false
              bytes: 622144
              windows:
                - start: "2026-10-20T15:06:32.044Z"
                  end: "2026-10-20T11:48:36-07:00"
            - harness: claude
              id: dd2ee03c-3bb0-40c3-8aec-7153c858d9f0
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 7004d633528a081b3018685616f0dbb291cfa1fc1f6fe3791036196047975501
              started: "2026-10-20T15:08:59.765Z"
              spawned: false
              bytes: 564150
              windows:
                - start: "2026-10-20T15:08:59.765Z"
                  end: "2026-10-20T11:48:36-07:00"
            - harness: codex
              id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 1327ee51fe2b1c5b226d0e00a8fa4d9088b057f819f45d60714289f4c3f29dc2
              started: "2026-10-20T18:21:51.812Z"
              spawned: false
              bytes: 2671923
              windows:
                - start: "2026-10-20T18:21:51.812Z"
                  end: "2026-10-20T11:48:36-07:00"
            - harness: codex
              id: 01a15a8c-f035-7473-97a1-e19202726aaa
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a15a8b-89c9-7e02-a733-8b52dd00ebac
              digest: cfd82bbdcec639b6eec5b6e0557b1968548e29cb79d60eeed6a6fac90915fc6a
              started: "2026-10-20T18:22:28.452Z"
              spawned: true
              bytes: 1482500
              windows:
                - start: "2026-10-20T18:22:28.452Z"
                  end: "2026-10-20T11:48:36-07:00"
        queue_waits:
            - attempt_id: df4e0de83a7b02852a727e603a53986e
              slot: pair:2
              record_digest: 1922c33a4000af22fad2e71748019ce8508f21663c0bee450c2e45ab4d6aef52
              start: "2026-10-20T16:31:08.064303Z"
              end: "2026-10-20T16:40:52.242939Z"
              queue_seconds: 584.178636
              raw_seconds: 584.178636
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 28.503233333333334
        cuts:
            - boundary: M1
              dispatched: "2026-10-20T08:25:12-07:00"
              finalized: "2026-10-20T08:33:44-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T08:42:07-07:00"
              finalized: "2026-10-20T08:54:17-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-20T09:00:29-07:00"
              finalized: "2026-10-20T09:04:14-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-20T09:14:52-07:00"
              finalized: "2026-10-20T09:19:42-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-20T09:24:08-07:00"
              finalized: "2026-10-20T09:26:13-07:00"
              finalizing: true
            - dispatched: "2026-10-20T09:50:20-07:00"
              finalized: "2026-10-20T09:53:00-07:00"
              finalizing: true
            - dispatched: "2026-10-20T11:29:15-07:00"
              finalized: "2026-10-20T11:30:31-07:00"
              finalizing: false
            - dispatched: "2026-10-20T11:31:53-07:00"
              finalized: "2026-10-20T11:36:35-07:00"
              finalizing: false
      "n": 9
      timestamp: "2026-10-20T11:56:28-07:00"
      dispatched: "2026-10-20T11:49:54-07:00"
      agent: codex
      reviewer: codex (openai; author anthropic,openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Late failures are retained and returned; the late-failure regression passes.
          round: 9
        - id: BR-2
          disposition: addressed
          note: Move admission uses warm targets; its regression verifies fresh rechecking before execution.
          round: 9
        - id: BR-3
          disposition: withdrawn
          note: Discover concurrency and replacement-fence tests ran successfully in this environment.
          round: 9
        - id: BR-4
          disposition: addressed
          note: Background flights and Discover row verification are capped at eight; focused tests pass.
          round: 9
        - id: BR-5
          disposition: addressed
          note: Workspace references use canonical keys; canonical-key tests pass.
          round: 9
        - id: BR-6
          disposition: addressed
          note: Admission and execution share checkHubSlotOperation with injected sources.
          round: 9
        - id: BR-7
          disposition: addressed
          note: Discover overlap is asserted through a rendezvous rather than elapsed time.
          round: 9
        - id: BR-8
          disposition: addressed
          note: Receipt filtering uses Warm under the request deadline; deadline and blocked-filter tests pass.
          round: 9
        - id: BR-9
          disposition: addressed
          note: Plan Revisions document failure retention, move rechecks, filtering, ordering, and caps.
          round: 9
        - id: BR-10
          disposition: addressed
          note: Start stamps prevent older resolutions replacing newer observations; ordering regressions pass.
          round: 9
        - id: BR-11
          disposition: addressed
          note: Writes sweep expired entries and enforce the 256-entry cap.
          round: 9
        - id: BR-12
          disposition: addressed
          note: Failure expiry uses completion time; the slow-failure regression passes.
          round: 9
        - id: BR-13
          disposition: addressed
          note: The oversized native-store fixture exercises metadata and record limits and verifies screen-only byte consumption.
          round: 9
        - id: BR-14
          disposition: addressed
          note: ReadRequest retains one Source-nil capture fence after optional candidate collection.
          round: 9
        - id: BR-15
          disposition: addressed
          note: Candidate output attributes the confirmed choice to its established binding without limiting the cause of incompleteness.
          round: 9
        - id: BR-16
          disposition: addressed
          note: The Plan M2 passage and atlas/couch.md now describe the established-binding distinction consistently with SkipCandidates routing.
          round: 9
        - id: BR-17
          disposition: addressed
          note: This review covers the complete pinned range, including repository batching, startup prewarm, and Tier A acceptance.
          round: 9
        - id: BR-18
          disposition: not-addressed
          note: 'Critical; family admission-sync-discover-over-exchange remains open. hub_guards.go:237 synchronously resolves an uncached caller workspace through OSProvisionIO, whose cancellation callback waits another two seconds (provision_io.go:75-87). A disposable socket regression removed the cached caller workspace and used the real resolver/runner with an owned sdlc script executing sleep 30: Call timed out at 2.000 s and the handler completed around 3.53 s. Returning an immediate reconnect refusal made the same test pass. ARCH-CONSTRAINTS, ARCH-PURPOSE, ARCH-MOCK: enforce elapsed response latency including subprocess teardown across every admission fallback, rather than treating context propagation as the bound.'
          round: 9
        - id: BR-19
          disposition: addressed
          note: Memo and queue selection reuse RepositorySiblingPrimary. Cold/prewarmed prefix and alias cases pass; disabling the shared sibling rule makes all six precedence cases fail.
          round: 9
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#726 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T08:33:44-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `refusal-must-carry-reason` (kind: defect) Cold target whose resolution fails after the budget drops the error; retries see only "warming" forever
  hub_target_memo.go Warm caches only successes. A failed flight is deleted, so each retry starts a new resolution and, past the 1.5 s budget, refuses with "warming; retry" and never the real error (e.g. slot candidate limit, host replaced, sdlc failure). Remember the last failure briefly and return it to the next Warm with its phase; add a test where the resolver fails after the budget.
- **BR-2** [Important] `admission-sync-discover-over-exchange` (kind: defect) handleHubMove still runs one fresh Discover per named slot synchronously, with no admission budget
  hub_move.go:84-90 calls authority.hubTarget (memo.Fresh, a full Discover of about 1 s) per comma-separated slot. A two-slot move goes past the 2 s exchange and gets the same undiagnosed client deadline this issue fixes for lifecycle ops (ARCH-PURPOSE sibling). Either add a budget and warm sources with a fresh re-proof in the move job, or record it as a non-goal in the plan Revisions.
- **BR-3** [Minor] `review-env-git-confinement` (kind: unverifiable) couchcore Discover concurrency and replaced-host tests could not be run in the review sandbox
  Ran go test -race -run TestDiscover ./cmd/internal/couchcore/. The .sdlc-review git wrapper refused git init under GOTMPDIR=/Volumes/TestScratch/tmp/pair-slot2 ("outside the reviewer's directories"). The pre-existing TestDiscoverUnionsNumberSources fails the same way. Host evidence: a passing -race run of TestDiscoverVerifiesSlotsConcurrently and TestDiscoverStillFailsWholeInventoryOnReplacedHost, plus a red run with the SameFile fence removed.
- **BR-4** [Minor] `bounded-fanout` (kind: defect) Discover fan-out and per-reference background resolutions have no small concurrency cap
  Rows run as wide as the row count (up to 128, each spawning sdlc and two git calls), and every distinct target spelling starts its own lifetime-scoped Discover. A semaphore of 8-16 would bound the burst on a co-tenanted host. The concurrency test's p4 > 4 check only re-states the row count.
- **BR-5** [Minor] `plan-code-drift` (kind: defect) Memo is keyed by raw r.Target, not the "canonical target reference" the plan states
  Aliases and prefixes get separate entries and separate in-flight resolutions. Harmless, but either canonicalize the key or revise the plan text.
- **BR-6** [Minor] `duplicated-guard-structure` (kind: defect) Admission closure repeats the guard closure and differs only in where origin and target come from
  hub_guards.go:222-242 vs 128-150 (ARCH-DRY). One helper that takes the origin and target sources would keep the two checks from drifting apart.

## Round 2 — 2026-10-20T08:54:17-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Failures remembered 30 s (hub_target_memo.go:79-81,97-100); TestHubTargetMemoRemembersALateFailure goes red when Fresh stops remembering errors.
- BR-2 — addressed — handleHubMove admits on warmHubTarget under slotAdmissionBudget and prepare re-resolves fresh; test red with fresh admission (1 resolution) and with the prepare check disabled.
- BR-3 — withdrawn — With GOTMPDIR inside .sdlc-review/tmp, TestDiscoverVerifiesSlotsConcurrently, TestDiscoverStillFailsWholeInventoryOnReplacedHost and TestDiscoverUnionsNumberSources pass under -race.
- BR-4 — addressed — slotVerifyConcurrency=8 gate plus hubTargetMaxFlights=8; the 10-slot peak check and the flight-cap test each go red when the cap is removed.
- BR-5 — not-addressed — hubTargetKey is identity on every accepted input, so :N and repo:N still key separately and the atlas claim "keys are canonical repo:N" is wrong; drop the helper and fix the text, or really canonicalize.
- BR-6 — addressed — checkHubSlotOperation with hubSlotSources serves both guard and admission (hub_guards.go:171-201); existing admission and guard tests pass.

### Raised

- **BR-7** [Important] `wall-clock-test-oracle` (kind: defect) TestDiscoverVerifiesSlotsConcurrently asserts elapsed time and fails under host load
  slotcatalog_concurrent_test.go:74-77 fails when 4-slot wall time is at least 3/4 of serial; it failed in 1 of 3 isolated -race runs ("took 454ms, close to the serial 600ms") and in the full suite at load ~57. Peak-concurrency counts already prove concurrency; delete the elapsed-time check or use a barrier.
- **BR-8** [Minor] `admission-sync-discover-over-exchange` (kind: defect) hubReceiptFilter waits on a cold memo resolution with no budget
  2nd finding in this family. Rule: every request-path read of a target slot must be bounded by the admission budget, so Warm should cap its own wait at slotAdmissionBudget rather than relying on each caller. hub_guards.go:249 calls warmHubTarget on the raw request ctx for operation-status polling.
- **BR-9** [Minor] `plan-code-drift` (kind: defect) Plan design and state table not revised for the round-2 behavior changes
  2nd finding in this family. Rule: any behavior change made in a review round updates the Plan design and state table (or a Revisions entry) in the same commit, not only the Log. The Plan still says Discover is bounded by MaxSlotCandidates; it lacks the 30 s remembered-failure state and the move prepare re-check.
- **BR-10** [Minor] `observation-ordering-stamp` (kind: defect) Memo stamps completion time, so an older in-flight resolution can overwrite a newer Fresh result
  remember() records now() when the resolution finishes; a background flight started before a prepare-time Fresh can land after it and overwrite it. The queued guard re-proves before any effect, so the damage is a wrong admission answer for up to 30 s or 10 min. Stamp the observation start and keep the newer entry.
- **BR-11** [Minor] `durable-cache-names-removal` (kind: defect) hubTargetMemo entries are never deleted; failures for arbitrary target strings now accumulate
  Entries expire logically but stay in the map for the supervisor's lifetime, and this round adds error entries for any reference a caller sends. Delete aged entries on lookup or sweep them, and correct the plan's ARCH-FUNERAL note.

## Round 3 — 2026-10-20T09:04:14-07:00 (claude) — passed

### Disposed

- BR-5 — addressed — hubTargetKey (hub_target_memo.go:39) makes repo:N keys canonical via ParseWorkspaceReference; pinned by TestHubTargetMemoKeysCanonicallyAndCapsFlights (passes).
- BR-7 — addressed — Elapsed-time check removed; rendezvousIO waits until min(slots,8) slot probes are in flight together, which a serial pass can never reach. Checked against verifyHost (one probe per slot host); couchcore can't run here because of sandbox git limits.
- BR-8 — not-addressed — All three warmHubTarget callers now add their own slotAdmissionBudget, but Warm doesn't cap its own wait as the rule asked, and no test fails without the receipt-filter budget (hub_guards.go:250).
- BR-9 — addressed — Plan Revisions now lists the 30 s failure age, the move prepare re-check, start-time ordering, the 256 cap, 8-wide fan-out and two new state-table rows.
- BR-10 — addressed — remember() stamps the start time and keeps the newer entry; undoing the old.at.After(started) check turns TestHubTargetMemoOrdersByStartAndSweeps red (checked here).
- BR-11 — addressed — remember() deletes expired entries and evicts the oldest past hubTargetMaxEntries=256; the test checks both the cap and the expiry sweep.

### Raised

- **BR-12** [Minor] `refusal-must-carry-reason` (kind: defect) A failed resolution that runs 30 s or more is stored already expired, so admission says "warming" forever again
  2nd finding in this family. Rule: an error from a finished resolution must reach the next admission for the full hubTargetFailureAge, counted from when it finished. Since BR-10, remember() stamps start time (hub_target_memo.go:158) and Warm and the sweep age failures from it (:102, :145). With 5 s ProbeTimeout probes across 8-wide waves of rows, a hung sdlc pushes Discover past 30 s, every retry starts a new flight and refuses "warming", and the cause never shows. Fix: keep the start time for ordering and add a finish time for failure expiry; test with a fake-clock resolver slower than hubTargetFailureAge.

## Round 4 — 2026-10-20T09:19:42-07:00 (claude) — BLOCKED

### Raised

- **BR-13** [Important] `plan-code-drift` (kind: defect) Done-when and Plan Tests fixture (over-cap inventory, oversized transcript, counted reads) not delivered nor revised
  3rd finding in this family. Rule: before a milestone closes, every Plan Tests bullet and Done-when test it claims is delivered as written or rewritten in Revisions in the same commit. recovery_skip_candidates_test.go uses one 456-byte transcript and infers no reads from empty Ranked; the inference is sound (reverting the skip branch goes red) but the stated acceptance fixture is absent. Add an oversized-transcript plus counting-reader Proven RecoverSelection test, or record the substitution in Revisions.
- **BR-14** [Minor] `duplicated-guard-structure` (kind: defect) Capture-identity fence when Source is nil is duplicated in ReadRequest (lines 199-206 and 224-230)
  2nd finding in this family. Rule: one guard, one site; variants pass in what differs rather than copying it. Wrap the native scan in if !q.SkipCandidates and fall through to the single shared fence (ARCH-DRY).
- **BR-15** [Minor] `refusal-must-carry-reason` (kind: defect) resume-candidates says an incomplete scan affects only inferred choices even when Incomplete came from the capture fence
  resume_candidates.go:123-126 keys on Incomplete and Proven != nil, but a failed validateRecoveryCaptureSnapshot ("recovery screen changed") also sets Incomplete and does affect the confirmed resume via Preserve. Condition the wording on the scan being the cause, or soften it.

## Round 5 — 2026-10-20T09:26:13-07:00 (claude) — passed

### Disposed

- BR-13 — addressed — TestRecoverySkipCandidatesOnAnOversizedNativeStore builds a store over RecoveryMetadataLimit with an over-RecoveryRecordLimit record and counts bytes via BytesRead against a screen-only read; Revisions records the counting substitution; reverting the SkipCandidates guard turns the test red (verified).
- BR-14 — addressed — The scan moved into collectCandidates behind !q.SkipCandidates; one Source-nil validateRecoveryCaptureSnapshot site remains at recovery_evidence_os.go:202-208.
- BR-15 — addressed — resume_candidates.go:126 now says only that the confirmed choice comes from the established binding, which holds whatever caused Incomplete; pinned by TestResumeCandidatesCLIIncompleteScanSparesTheConfirmedChoice. Atlas and plan restatements are raised separately.

### Raised

- **BR-16** [Minor] `plan-code-drift` (kind: defect) Atlas and the plan M2 bullet still claim an incomplete scan affects only the inferred choices
  atlas/couch.md:83-84 and the issue's M2 bullet keep the wording BR-15 removed from resume_candidates.go:126. This is the 4th finding in this family. Rule: a change to user-visible wording or behavior sweeps every atlas passage and Plan bullet that restates it in the same commit (grep the old phrase).

## Round 6 — 2026-10-20T09:53:00-07:00 (claude) — passed

### Disposed

- BR-8 — addressed — Warm caps its own wait at slotAdmissionBudget (hub_target_memo.go:104-107); without it TestHubTargetMemoWarmCapsItsOwnWait fails after 30 s.
- BR-12 — addressed — Entries now carry started (ordering) and finished (expiry), and live() ages failures from finished; reverting to started turns TestHubTargetMemoAgesFailuresFromTheirEnd red.
- BR-16 — not-addressed — atlas/couch.md:83-85 and the issue M2 bullet (line 78) are fixed, but grepping the old phrase still finds the resume_candidates_test.go:194 doc comment ("affects only the inferred choices"); reword it.

## Round 7 — 2026-10-20T11:30:31-07:00 (codex) — BLOCKED

### Disposed

- BR-16 — addressed — atlas/couch.md:83-85 and the issue's M2 bullet at line 79 correctly attribute the confirmed choice to the established binding, matching resume_candidates.go:123-126. The remaining stale test comment is corrected by this interdiff at resume_candidates_test.go:194-196; this is a prose-only correction.

### Raised

- **BR-17** [Critical] `boundary-review-scope` (kind: defect) scope changed after close — run a full `sdlc close`
  The issue's new Revisions entry at line 105 introduces repository-wide single-flight discovery, startup prewarming, and Tier A batch acceptance beyond the outstanding wording finding. Executable changes in hub_target_memo.go:108-274, message_service.go:701-709, and slotcontext.go:203-222 implement that expansion. Run a full sdlc close covering the expanded scope; this narrow fix review cannot authorize it.

## Round 8 — 2026-10-20T11:36:35-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Failure entries retain the resolver error; TestHubTargetMemoRemembersALateFailure covers late completion and retry.
- BR-2 — addressed — Move target resolution uses Warm; TestHubMoveAdmissionUsesWarmSlotsAndRechecksFresh covers admission and fresh revalidation.
- BR-3 — withdrawn — Discover concurrency and replaced-host tests ran successfully under the race detector in this environment.
- BR-4 — addressed — Background flights and Discover verification have eight-worker caps, covered by focused race tests.
- BR-5 — addressed — Memo access and writes consistently use hubTargetKey.
- BR-6 — addressed — Admission and fresh guards share checkHubSlotOperation with injected sources.
- BR-7 — addressed — Discover concurrency uses an in-flight rendezvous rather than elapsed time as its success oracle.
- BR-8 — addressed — Receipt filtering bounds its memo wait; Warm also enforces its own budget.
- BR-9 — addressed — The issue Revisions records failure retention, move revalidation, caps, ordering, and additional visible states.
- BR-10 — addressed — remember rejects older-started results; late individual and repository completion regressions pass.
- BR-11 — addressed — Writes sweep expired entries and cap the map at 256; the sweep/cap regression passes.
- BR-12 — addressed — Failure expiry uses finished time; TestHubTargetMemoAgesFailuresFromTheirEnd passes.
- BR-13 — addressed — The oversized-store fixture exercises metadata and record limits, compares counted bytes, and passes with the routing regression.
- BR-14 — addressed — ReadRequest has one Source-nil capture fence after optional candidate collection.
- BR-15 — addressed — resume_candidates.go attributes the confirmed choice to the established binding without claiming all incomplete observations affect only inference.
- BR-16 — addressed — atlas/couch.md:83-85 and the Plan M2 bullet match the CLI wording and established-binding recovery path.
- BR-17 — addressed — This review inspected the complete pinned range, including repository batching, startup prewarm, production wiring, and Tier A acceptance. The issue Revisions at line 121 explicitly supersedes the original per-reference design.

### Raised

- **BR-18** [Critical] `admission-sync-discover-over-exchange` (kind: defect) The admission budget does not cover the complete admission path
  hub_guards.go:162-168 starts the 1.5 s budget after handleSlotOperation has authenticated the caller, then cancels it before slotOps.handle. Earlier liveCaller reads can wait up to one second for the store lock (message_service.go:1256; couchcore/retention.go:299), while later queueTarget performs another repository-name read with a separate 500 ms allowance. A caller check that succeeds after 700 ms followed by a cold target resolution reaches the parent 2 s deadline before the inner budget expires; transport.go then drops the refusal. This is the 3rd finding in family admission-sync-discover-over-exchange. Earlier rounds fixed instances; establish one deadline for the entire lifecycle admission path, including authentication, target resolution, queue-key lookup, and enqueue, and sweep slot, move, and status paths against that rule (ARCH-CONSTRAINTS, ARCH-PURPOSE). Add a socket regression combining delayed successful authentication with blocked target resolution, plus delayed queue-name lookup.
- **BR-19** [Critical] `canonical-reference-resolution` (kind: defect) Repository warming bypasses sibling-directory precedence
  hub_target_memo.go:119 resolves repository names directly against enrollment, then returns the batched slot at lines 135-146. The existing fresh resolver instead gives an exact sibling directory precedence over enrolled names and prefixes (couchcore/slotcontext.go:152-156), explicitly tested by TestExistingSiblingDirectoryBeatsPrefix. With an existing sibling parley and an enrolled parley.nvim, Warm("parley:1") can return parley.nvim:1 while Fresh resolves parley:1. If those repositories belong to different hubs, admission refuses a valid operation before fresh authority can correct the snapshot. Derive memo repository identity through the same reference-resolution rule, retaining batching after identity is established; cover cold and prewarmed references, including prefixes and aliases (ARCH-DRY, ARCH-PURPOSE).

## Round 9 — 2026-10-20T11:56:28-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Late failures are retained and returned; the late-failure regression passes.
- BR-2 — addressed — Move admission uses warm targets; its regression verifies fresh rechecking before execution.
- BR-3 — withdrawn — Discover concurrency and replacement-fence tests ran successfully in this environment.
- BR-4 — addressed — Background flights and Discover row verification are capped at eight; focused tests pass.
- BR-5 — addressed — Workspace references use canonical keys; canonical-key tests pass.
- BR-6 — addressed — Admission and execution share checkHubSlotOperation with injected sources.
- BR-7 — addressed — Discover overlap is asserted through a rendezvous rather than elapsed time.
- BR-8 — addressed — Receipt filtering uses Warm under the request deadline; deadline and blocked-filter tests pass.
- BR-9 — addressed — Plan Revisions document failure retention, move rechecks, filtering, ordering, and caps.
- BR-10 — addressed — Start stamps prevent older resolutions replacing newer observations; ordering regressions pass.
- BR-11 — addressed — Writes sweep expired entries and enforce the 256-entry cap.
- BR-12 — addressed — Failure expiry uses completion time; the slow-failure regression passes.
- BR-13 — addressed — The oversized native-store fixture exercises metadata and record limits and verifies screen-only byte consumption.
- BR-14 — addressed — ReadRequest retains one Source-nil capture fence after optional candidate collection.
- BR-15 — addressed — Candidate output attributes the confirmed choice to its established binding without limiting the cause of incompleteness.
- BR-16 — addressed — The Plan M2 passage and atlas/couch.md now describe the established-binding distinction consistently with SkipCandidates routing.
- BR-17 — addressed — This review covers the complete pinned range, including repository batching, startup prewarm, and Tier A acceptance.
- BR-18 — not-addressed — Critical; family admission-sync-discover-over-exchange remains open. hub_guards.go:237 synchronously resolves an uncached caller workspace through OSProvisionIO, whose cancellation callback waits another two seconds (provision_io.go:75-87). A disposable socket regression removed the cached caller workspace and used the real resolver/runner with an owned sdlc script executing sleep 30: Call timed out at 2.000 s and the handler completed around 3.53 s. Returning an immediate reconnect refusal made the same test pass. ARCH-CONSTRAINTS, ARCH-PURPOSE, ARCH-MOCK: enforce elapsed response latency including subprocess teardown across every admission fallback, rather than treating context propagation as the bound.
- BR-19 — addressed — Memo and queue selection reuse RepositorySiblingPrimary. Cold/prewarmed prefix and alias cases pass; disabling the shared sibling rule makes all six precedence cases fail.

## Open findings

- **BR-18** [Critical] `admission-sync-discover-over-exchange` (kind: defect) The admission budget does not cover the complete admission path
