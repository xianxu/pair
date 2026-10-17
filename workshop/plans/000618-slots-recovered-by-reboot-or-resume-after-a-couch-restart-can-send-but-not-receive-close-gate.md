---
gate: boundary-review
issue: 618
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-16T16:17:20-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: The plan doesn't say what an admission does before the poller's first snapshot exists
          detail: |-
            Right after the supervisor starts, no listing has completed yet. Say whether an admission then waits for the first listing within AdmissionTimeout or refuses with a distinct reason, and cover it in the Task 1 or Task 2 tests.
            (carried from plan-quality PQ-8, deferred to the boundary review)
          family: operating-envelope-arithmetic
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 9d0b3176-5f36-4a8b-9f9b-e2053b2a5f11
            repository: github.com/xianxu/pair
            issue_id: "000618"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/f48f7c976a894f08
            reviewed_head: c09811358886808dd2f508e7441125e0055cf01e
            slot: pair:2
            requested_at: "2026-10-16T23:03:17.155788Z"
          phase: finished
          pid: 12544
          started_at: "2026-10-16T23:03:17.468156Z"
          finished_at: "2026-10-16T23:17:19.332222Z"
          session_id: dc455cf2-a491-4438-bd15-9974924e1e9d
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: c09811358886808dd2f508e7441125e0055cf01e
        tracker_head: 0f3bfabeb5b4fdaab89fa6bb1593f45693455389
        branch_point: 5254fc022774f55cefafc6496c2bcbeef7b4d13b
        since: "2026-10-16T15:08:44-07:00"
        until: "2026-10-16T16:01:56-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T15:08:44-07:00"
              end: "2026-10-16T16:01:56-07:00"
              began: "2026-10-16T15:08:44-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 07f507a1-f545-4353-94e5-fdb696746544
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: b165e6321f8a1d73f9389567b851cc270577e2f8ffa5048368ef347f068adb70
              started: "2026-10-16T22:52:05.608Z"
              spawned: false
              bytes: 400819
              windows:
                - start: "2026-10-16T22:52:05.608Z"
                  end: "2026-10-16T16:01:56-07:00"
            - harness: claude
              id: 160fb503-f0f1-4a8f-8326-9e0cfe725840
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: ee2a83a7d6648515c3c29745af3b57764d62397578827289264307fdd66b44c0
              started: "2026-10-16T22:48:36.145Z"
              spawned: false
              bytes: 533539
              windows:
                - start: "2026-10-16T22:48:36.145Z"
                  end: "2026-10-16T16:01:56-07:00"
            - harness: claude
              id: 5c968657-789a-43c4-ac98-9785c4f1a7d8
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 6be6e9e7f58fb25c7b1a9f0721271b4a15aea2266429589c4d15ff1c1f1b11be
              started: "2026-10-16T22:51:33.47Z"
              spawned: false
              bytes: 513772
              windows:
                - start: "2026-10-16T22:51:33.47Z"
                  end: "2026-10-16T16:01:56-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 71e1559bbdde1ef0190e8a886b8ee72f9b4d5f731f65661731bc5455e09357e3
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 15712319
              windows:
                - start: "2026-10-16T15:08:44-07:00"
                  end: "2026-10-16T16:01:56-07:00"
            - harness: claude
              id: ef855715-99b9-41fa-8900-e650e2074c61
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 64a17b9d26b212b883938ab33f78ba71a40471c9365508e44606078f3cbdf368
              started: "2026-10-16T22:45:51.536Z"
              spawned: false
              bytes: 717215
              windows:
                - start: "2026-10-16T22:45:51.536Z"
                  end: "2026-10-16T16:01:56-07:00"
            - harness: claude
              id: f05e1f4c-b0b9-47e7-a879-88650252a387
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 9136de5157c597cb5a7d651a276ef2214fed0f608bc3576879406655e351a61b
              started: "2026-10-16T22:49:27.576Z"
              spawned: false
              bytes: 499639
              windows:
                - start: "2026-10-16T22:49:27.576Z"
                  end: "2026-10-16T16:01:56-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 10.169466666666667
      "n": 2
      timestamp: "2026-10-16T16:17:20-07:00"
      dispatched: "2026-10-16T16:03:13-07:00"
      agent: claude
      findings:
        - kind: defect
          id: BR-2
          severity: Critical
          title: CLI send's AdmissionTimeout+1s is clamped to TransportTimeout (2s) by Call, and the server drops the reply when its 2s ctx expires, so a timed-out admission never reports its refusal reason
          detail: couchmessage.Call wraps ctx with TransportTimeout=2s, and Server.serve runs the handler under the same 2s budget, returning with no reply once ctx.Err() is set. The "admission timed out" refusal is stored, but the send still prints the uncertain/query-status text. TestMessageCLISendWaitsForTheBrokersVerdict injects a fake call, so it cannot catch this, and atlas/couch.md states the false claim. Fix the send transport budget on both sides and add a test that goes through the real Server and Call.
          family: transport-budget-clamps-caller-timeout
          round: 2
        - kind: defect
          id: BR-3
          severity: Important
          title: 'Task 2 tests are missing: no admission through the wired resolver under a slow zellij, and no bound on listings caused by promote retries'
          detail: message_listing_test.go calls messageListedObserver directly. Nothing checks that startMessageService passes the poller to messageThreadResolver, and nothing checks that a 5s queued-message promote retry causes at most about 5 listing pairs, both of which the plan lists as failing tests for Task 2.
          family: plan-test-not-delivered
          round: 2
        - kind: defect
          id: BR-4
          severity: Important
          title: M1 live step (Task 5) has no recorded listing-rate or CPU measurement
          detail: 'The plan''s M1 includes Task 5 live: under a full suite, sends land, plus the supervisor''s zellij listings per second and CPU before and after. The issue Log records no such measurement, so the 2-per-second claim is unsupported (ARCH-CONSTRAINTS).'
          family: unmeasured-operating-claim
          round: 2
        - kind: defect
          id: BR-5
          severity: Important
          title: startMessageService runs a synchronous, uncancellable listing refresh on context.Background before the service starts
          detail: listings.Refresh(context.Background()) at message_service.go:158 runs two zellij listings serially, each bounded only by zellijQueryTimeout (5s), so supervisor startup can block for up to about 10s under the load this issue measured. Run then lists again immediately. Bound it with the service context, or let Run do the first refresh.
          family: blocking-startup-io
          round: 2
        - kind: defect
          id: BR-6
          severity: Minor
          title: LatestRefusals has no M1 consumer and infers refusal time from RetainUntil minus retention
          detail: Task 4 moved to M2. Store an explicit RefusedAt time and compare it against resolved slot bindings when --list starts consuming this.
          family: unconsumed-api-surface
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 9398105f-5f66-4b2b-a9cf-c86a2be21a6e
            repository: github.com/xianxu/pair
            issue_id: "000618"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/76b4167f1c28ebd3
            reviewed_head: 19363f2714db86ef31b23e71d3678eb655a88b05
            slot: pair:2
            requested_at: "2026-10-16T23:22:24.320167Z"
          phase: finished
          pid: 24980
          started_at: "2026-10-16T23:22:24.648152Z"
          finished_at: "2026-10-16T23:37:19.413739Z"
          session_id: f0d70dad-e3df-4600-921d-82e29411f88f
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 19363f2714db86ef31b23e71d3678eb655a88b05
        tracker_head: 31afcf7bbb71bab7715063f7deafc0822ea8c849
        branch_point: 5254fc022774f55cefafc6496c2bcbeef7b4d13b
        since: "2026-10-16T15:08:44-07:00"
        until: "2026-10-16T16:20:39-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T15:08:44-07:00"
              end: "2026-10-16T16:20:39-07:00"
              began: "2026-10-16T15:08:44-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 07f507a1-f545-4353-94e5-fdb696746544
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: b165e6321f8a1d73f9389567b851cc270577e2f8ffa5048368ef347f068adb70
              started: "2026-10-16T22:52:05.608Z"
              spawned: false
              bytes: 400819
              windows:
                - start: "2026-10-16T22:52:05.608Z"
                  end: "2026-10-16T16:20:39-07:00"
            - harness: claude
              id: 160fb503-f0f1-4a8f-8326-9e0cfe725840
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: ee2a83a7d6648515c3c29745af3b57764d62397578827289264307fdd66b44c0
              started: "2026-10-16T22:48:36.145Z"
              spawned: false
              bytes: 533539
              windows:
                - start: "2026-10-16T22:48:36.145Z"
                  end: "2026-10-16T16:20:39-07:00"
            - harness: claude
              id: 5c968657-789a-43c4-ac98-9785c4f1a7d8
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 6be6e9e7f58fb25c7b1a9f0721271b4a15aea2266429589c4d15ff1c1f1b11be
              started: "2026-10-16T22:51:33.47Z"
              spawned: false
              bytes: 513772
              windows:
                - start: "2026-10-16T22:51:33.47Z"
                  end: "2026-10-16T16:20:39-07:00"
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 9bfcf9df519feb8b08acd8f05c0cd521a8516e01cc3ff44bae59a4d0c2f3d94a
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 16006756
              windows:
                - start: "2026-10-16T15:08:44-07:00"
                  end: "2026-10-16T16:20:39-07:00"
            - harness: claude
              id: ef855715-99b9-41fa-8900-e650e2074c61
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 64a17b9d26b212b883938ab33f78ba71a40471c9365508e44606078f3cbdf368
              started: "2026-10-16T22:45:51.536Z"
              spawned: false
              bytes: 717215
              windows:
                - start: "2026-10-16T22:45:51.536Z"
                  end: "2026-10-16T16:20:39-07:00"
            - harness: claude
              id: f05e1f4c-b0b9-47e7-a879-88650252a387
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 9136de5157c597cb5a7d651a276ef2214fed0f608bc3576879406655e351a61b
              started: "2026-10-16T22:49:27.576Z"
              spawned: false
              bytes: 499639
              windows:
                - start: "2026-10-16T22:49:27.576Z"
                  end: "2026-10-16T16:20:39-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 10.169466666666667
        cuts:
            - boundary: M1
              dispatched: "2026-10-16T16:03:13-07:00"
              finalized: "2026-10-16T16:17:20-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-16T16:37:21-07:00"
      dispatched: "2026-10-16T16:22:21-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Decided as a distinct refusal ("zellij liveness is not yet known"); covered in zellij_listing_poller_test.go and TestMessageThreadResolverChecksReadOnlyThePoller.
          round: 3
        - id: BR-2
          disposition: addressed
          note: requestBudget shared by Call and Server.serve; disabling the send case turns TestTransportSendOutlastsAdmission red (context deadline exceeded), restored is green.
          round: 3
        - id: BR-3
          disposition: addressed
          note: TestMessageThreadResolverChecksReadOnlyThePoller drives messageThreadResolver(c, listings) with a 1s-per-listing stub; 50 checks run zero listings, so the retry-driven listing rate is bounded by the poller interval.
          round: 3
        - id: BR-4
          disposition: addressed
          note: BEFORE numbers are in the Log (6.1/s, 68% CPU); a plan revision defers AFTER to the post-landing restart and requires it before M2 closes. Hold M2 close to it.
          round: 3
        - id: BR-5
          disposition: addressed
          note: The startup Refresh on context.Background is gone; Run does the first listing in a worker on service.lifetime (message_service.go:475-476); checks refuse not-yet-known until then.
          round: 3
        - id: BR-6
          disposition: addressed
          note: refusal now carries an explicit at time used by eviction and LatestRefusals; comparing against slot bindings belongs to the M2 --list consumer.
          round: 3
      findings:
        - kind: defect
          id: BR-7
          severity: Minor
          title: Broker.Send restarts a fresh AdmissionTimeout after the sender check instead of deriving from the exchange deadline
          detail: '2nd finding in this family. Rule: any budget nested inside a socket exchange must derive from that exchange''s deadline (minus the reply margin), never start its own clock. Today the margin holds only because authority.current is cheap with the poller (message_service.go:829-833, broker.go:523).'
          family: transport-budget-clamps-caller-timeout
          round: 3
        - kind: defect
          id: BR-8
          severity: Minor
          title: Server.serve decodes the request three times to pick its budget; Call marshals the request twice
          detail: requestBudget already returns RecoveryPreviewTimeout for resume-candidates; serve should apply it once and key only the EOF-watcher branch on the op (transport.go:192-212, 251-254). ARCH-DRY.
          family: duplicated-request-classification
          round: 3
        - kind: defect
          id: BR-9
          severity: Minor
          title: Plan says a failed listing is reported once per failure streak; the poller never reports it
          detail: '2nd finding in this family. Rule: every behaviour the plan''s Core concepts table promises is either implemented with a test or removed by a Revisions entry at the same boundary. Here the failure only appears inside ErrListingStale text (zellij_listing_poller.go:68-77).'
          family: plan-test-not-delivered
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: 19363f2714db86ef31b23e71d3678eb655a88b05
      blocked: false
---

# Gate ledger — pair#618 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-16T16:17:20-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `operating-envelope-arithmetic` The plan doesn't say what an admission does before the poller's first snapshot exists
  Right after the supervisor starts, no listing has completed yet. Say whether an admission then waits for the first listing within AdmissionTimeout or refuses with a distinct reason, and cover it in the Task 1 or Task 2 tests.
  (carried from plan-quality PQ-8, deferred to the boundary review)

## Round 2 — 2026-10-16T16:17:20-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Critical] `transport-budget-clamps-caller-timeout` (kind: defect) CLI send's AdmissionTimeout+1s is clamped to TransportTimeout (2s) by Call, and the server drops the reply when its 2s ctx expires, so a timed-out admission never reports its refusal reason
  couchmessage.Call wraps ctx with TransportTimeout=2s, and Server.serve runs the handler under the same 2s budget, returning with no reply once ctx.Err() is set. The "admission timed out" refusal is stored, but the send still prints the uncertain/query-status text. TestMessageCLISendWaitsForTheBrokersVerdict injects a fake call, so it cannot catch this, and atlas/couch.md states the false claim. Fix the send transport budget on both sides and add a test that goes through the real Server and Call.
- **BR-3** [Important] `plan-test-not-delivered` (kind: defect) Task 2 tests are missing: no admission through the wired resolver under a slow zellij, and no bound on listings caused by promote retries
  message_listing_test.go calls messageListedObserver directly. Nothing checks that startMessageService passes the poller to messageThreadResolver, and nothing checks that a 5s queued-message promote retry causes at most about 5 listing pairs, both of which the plan lists as failing tests for Task 2.
- **BR-4** [Important] `unmeasured-operating-claim` (kind: defect) M1 live step (Task 5) has no recorded listing-rate or CPU measurement
  The plan's M1 includes Task 5 live: under a full suite, sends land, plus the supervisor's zellij listings per second and CPU before and after. The issue Log records no such measurement, so the 2-per-second claim is unsupported (ARCH-CONSTRAINTS).
- **BR-5** [Important] `blocking-startup-io` (kind: defect) startMessageService runs a synchronous, uncancellable listing refresh on context.Background before the service starts
  listings.Refresh(context.Background()) at message_service.go:158 runs two zellij listings serially, each bounded only by zellijQueryTimeout (5s), so supervisor startup can block for up to about 10s under the load this issue measured. Run then lists again immediately. Bound it with the service context, or let Run do the first refresh.
- **BR-6** [Minor] `unconsumed-api-surface` (kind: defect) LatestRefusals has no M1 consumer and infers refusal time from RetainUntil minus retention
  Task 4 moved to M2. Store an explicit RefusedAt time and compare it against resolved slot bindings when --list starts consuming this.

## Round 3 — 2026-10-16T16:37:21-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Decided as a distinct refusal ("zellij liveness is not yet known"); covered in zellij_listing_poller_test.go and TestMessageThreadResolverChecksReadOnlyThePoller.
- BR-2 — addressed — requestBudget shared by Call and Server.serve; disabling the send case turns TestTransportSendOutlastsAdmission red (context deadline exceeded), restored is green.
- BR-3 — addressed — TestMessageThreadResolverChecksReadOnlyThePoller drives messageThreadResolver(c, listings) with a 1s-per-listing stub; 50 checks run zero listings, so the retry-driven listing rate is bounded by the poller interval.
- BR-4 — addressed — BEFORE numbers are in the Log (6.1/s, 68% CPU); a plan revision defers AFTER to the post-landing restart and requires it before M2 closes. Hold M2 close to it.
- BR-5 — addressed — The startup Refresh on context.Background is gone; Run does the first listing in a worker on service.lifetime (message_service.go:475-476); checks refuse not-yet-known until then.
- BR-6 — addressed — refusal now carries an explicit at time used by eviction and LatestRefusals; comparing against slot bindings belongs to the M2 --list consumer.

### Raised

- **BR-7** [Minor] `transport-budget-clamps-caller-timeout` (kind: defect) Broker.Send restarts a fresh AdmissionTimeout after the sender check instead of deriving from the exchange deadline
  2nd finding in this family. Rule: any budget nested inside a socket exchange must derive from that exchange's deadline (minus the reply margin), never start its own clock. Today the margin holds only because authority.current is cheap with the poller (message_service.go:829-833, broker.go:523).
- **BR-8** [Minor] `duplicated-request-classification` (kind: defect) Server.serve decodes the request three times to pick its budget; Call marshals the request twice
  requestBudget already returns RecoveryPreviewTimeout for resume-candidates; serve should apply it once and key only the EOF-watcher branch on the op (transport.go:192-212, 251-254). ARCH-DRY.
- **BR-9** [Minor] `plan-test-not-delivered` (kind: defect) Plan says a failed listing is reported once per failure streak; the poller never reports it
  2nd finding in this family. Rule: every behaviour the plan's Core concepts table promises is either implemented with a test or removed by a Revisions entry at the same boundary. Here the failure only appears inside ErrListingStale text (zellij_listing_poller.go:68-77).

## Open findings

- **BR-7** [Minor] `transport-budget-clamps-caller-timeout` (kind: defect) Broker.Send restarts a fresh AdmissionTimeout after the sender check instead of deriving from the exchange deadline
- **BR-8** [Minor] `duplicated-request-classification` (kind: defect) Server.serve decodes the request three times to pick its budget; Call marshals the request twice
- **BR-9** [Minor] `plan-test-not-delivered` (kind: defect) Plan says a failed listing is reported once per failure streak; the poller never reports it
