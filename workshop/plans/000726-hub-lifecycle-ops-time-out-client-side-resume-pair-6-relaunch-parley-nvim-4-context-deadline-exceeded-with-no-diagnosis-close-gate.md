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

## Open findings

- **BR-8** [Minor] `admission-sync-discover-over-exchange` (kind: defect) hubReceiptFilter waits on a cold memo resolution with no budget
- **BR-12** [Minor] `refusal-must-carry-reason` (kind: defect) A failed resolution that runs 30 s or more is stored already expired, so admission says "warming" forever again
