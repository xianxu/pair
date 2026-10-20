---
gate: boundary-review
issue: 711
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 2da21866-4cee-4387-a74d-117dea7c0e10
            repository: github.com/xianxu/pair
            issue_id: "000711"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/b52028d24f0f238a
            reviewed_head: 734d84a932b3c64e1043cf625a2038e71bb294a4
            slot: pair:7
            requested_at: "2026-10-20T13:10:28.650937Z"
          phase: finished
          pid: 15274
          started_at: "2026-10-20T13:10:28.908748Z"
          finished_at: "2026-10-20T13:13:22.529271Z"
          session_id: 01a1596f-4f3c-7b50-b9ce-3551a4df83ef
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 734d84a932b3c64e1043cf625a2038e71bb294a4
        tracker_head: 2d351a820733b016b5bff93228db2c320236e6c2
        branch_point: 7d4e0c3aba6b39fb12d5125ee967dcde9e5be4be
        since: "2026-10-20T05:51:05-07:00"
        until: "2026-10-20T06:09:28-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T05:51:05-07:00"
              end: "2026-10-20T06:09:28-07:00"
              began: "2026-10-20T05:51:05-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: a4ac4f00686ac83297c00cda7919503cf889d211ba8d1297a566278d56319377
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 190348201
              windows:
                - start: "2026-10-20T05:51:05-07:00"
                  end: "2026-10-20T06:09:28-07:00"
            - harness: codex
              id: 01a1595d-fe1a-7d60-b4df-d526bd4328c3
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 4b0c7115e7efcc40d4223b3ab5d0f51d9b804b8ca71968931063f3f12d6cdd62
              started: "2026-10-20T12:51:34.625Z"
              spawned: true
              bytes: 1194949
              windows:
                - start: "2026-10-20T12:51:34.625Z"
                  end: "2026-10-20T06:09:28-07:00"
            - harness: codex
              id: 01a1595e-faf3-7473-9eeb-792eec7597c8
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 815997fe1d3bdbcd3955d92e995102cd0a39ac95b14468e26045d0a47d1ebbec
              started: "2026-10-20T12:52:39.378Z"
              spawned: true
              bytes: 1096860
              windows:
                - start: "2026-10-20T12:52:39.378Z"
                  end: "2026-10-20T06:09:28-07:00"
      "n": 1
      timestamp: "2026-10-20T06:13:24-07:00"
      dispatched: "2026-10-20T06:10:25-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-1
          severity: Critical
          title: Same-binding sender reconnect can admit a stale reservation
          detail: 'At cmd/internal/couchmessage/broker.go:726, b.caller(from) checks current connectivity without comparing the sender registration with the one that began admission. A disposable regression disconnected and re-registered the sender after successful Reserve, then invoked recipient OperatorInput; Send returned a queued receipt with no error. This violates the Spec''s unchanged-both-registrations condition. Enumerated admission sites requiring the same sender lifetime token are send''s initial validation (574), admit''s loop validation (631), tryAdmission''s post-IO validation (726), and the busy-to-queue validation (684). Capture the sender actor/generation and preserve it across these paths. Add sender-reconnect controls alongside the existing recipient-reconnect control. ARCH-PURPOSE: sweep both registration lifetimes.'
          family: admission-registration-lifetime-fencing
          round: 1
        - id: BR-2
          severity: Critical
          title: Runtime listing discards validated exited states and reclassifies names incorrectly
          detail: 'At cmd/internal/launcher/osruntime.go:334-353, ListSessions extracts names from sessionListingStates but discards its state values. Its downstream buildListRowsForScope at list.go:78 calls zellijsession.RowState, which uses Fields[0] and Contains("EXITED"). Disposable probes showed a live pair-EXITED-is-a-name displayed as exited, while exited names containing spaces or tabs displayed as detached and triggered client queries. The remaining production instance in this window is ListSessions -> buildListRowsForScope -> RowState; ZellijSource.snapshot already consumes validated states correctly. Pass the validated state map into the runtime row builder and add these three edge cases. ARCH-DRY and ARCH-PURPOSE: all promised inventory consumers must use the same identity/state interpretation.'
          family: validated-inventory-state-consumers
          round: 1
        - id: BR-3
          severity: Important
          title: Admission races still return bare, misleading unavailable errors
          detail: 'The send path retains bare ErrUnavailable returns at cmd/internal/couchmessage/broker.go:576, :633, and :685. These are the complete unlabelled registration-refusal instances in the admission path: initial caller validation, caller validation after routing IO, and combined caller/recipient validation after reservation release. A disposable regression disconnected the caller inside the injected family source; Send returned "recipient is not live", losing both the failing side and phase. Wrap each caller failure with caller/current context, split the combined fallback guard by side, preserve errors.Is classification, and retain regression coverage. ARCH-PURPOSE: diagnostics must cover concurrent invalidation as well as initially missing registrations.'
          family: availability-refusal-phase-context
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: e3336ff8-d20a-4362-bf88-d4d6f13ed6b7
            repository: github.com/xianxu/pair
            issue_id: "000711"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/e2c7214096953d25
            reviewed_head: a85138a58b338cdba337fbb4b2e504ebfc5efffe
            slot: pair:7
            requested_at: "2026-10-20T13:27:26.995158Z"
          phase: finished
          pid: 65126
          started_at: "2026-10-20T13:27:27.088372Z"
          finished_at: "2026-10-20T13:29:25.974424Z"
          session_id: 01a1597e-d765-75f2-a946-077c9d9910c3
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: a85138a58b338cdba337fbb4b2e504ebfc5efffe
        tracker_head: 2d351a820733b016b5bff93228db2c320236e6c2
        branch_point: 7d4e0c3aba6b39fb12d5125ee967dcde9e5be4be
        since: "2026-10-20T05:51:05-07:00"
        until: "2026-10-20T06:26:56-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T05:51:05-07:00"
              end: "2026-10-20T06:26:56-07:00"
              began: "2026-10-20T05:51:05-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: ee15601201df84d6aab0a7a3da173ed69e80ed594c3beed14907474badf50170
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 191123362
              windows:
                - start: "2026-10-20T05:51:05-07:00"
                  end: "2026-10-20T06:26:56-07:00"
            - harness: codex
              id: 01a1595d-fe1a-7d60-b4df-d526bd4328c3
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 4b0c7115e7efcc40d4223b3ab5d0f51d9b804b8ca71968931063f3f12d6cdd62
              started: "2026-10-20T12:51:34.625Z"
              spawned: true
              bytes: 1194949
              windows:
                - start: "2026-10-20T12:51:34.625Z"
                  end: "2026-10-20T06:26:56-07:00"
            - harness: codex
              id: 01a1595e-faf3-7473-9eeb-792eec7597c8
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 740d924d66204ffa0888e119de25da1eb2210214a0c0f2015e71c1fc15543dd3
              started: "2026-10-20T12:52:39.378Z"
              spawned: true
              bytes: 1187226
              windows:
                - start: "2026-10-20T12:52:39.378Z"
                  end: "2026-10-20T06:26:56-07:00"
        cuts:
            - dispatched: "2026-10-20T06:10:25-07:00"
              finalized: "2026-10-20T06:13:24-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-20T06:29:27-07:00"
      dispatched: "2026-10-20T06:27:25-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: broker.go captures the sender actor under lock and checks its identity through routing, reservation, and queue fallback. Removing the identity comparison makes TestBrokerSenderReconnectCannotAuthorizeOldAdmission fail in all three phases.
          round: 2
        - id: BR-2
          disposition: addressed
          note: OSRuntime.ListSessions passes validated states into buildListRowsForScope. Restoring RowState interpretation makes TestRuntimeListSessionsUsesValidatedOpaqueNameState fail for all three opaque-name cases.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Admission refusals identify caller or recipient and phase while preserving errors.Is classification. Removing the contextual wrappers makes TestBrokerAdmissionLossNamesSideAndPhase fail in all four cases.
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: a85138a58b338cdba337fbb4b2e504ebfc5efffe
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 2329e34d-9420-4200-94fb-d74c5ad7db20
            repository: github.com/xianxu/pair
            issue_id: "000711"
            recipe: fix-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/bac572951b144431
            reviewed_head: 6231932580842773fd14e5349f4b52a48d34c4cc
            slot: pair:7
            requested_at: "2026-10-20T13:58:24.353386Z"
          phase: finished
          pid: 47960
          started_at: "2026-10-20T13:58:24.571573Z"
          finished_at: "2026-10-20T13:59:30.110898Z"
          session_id: 01a1599b-3044-77c1-8860-b85eeb20eef2
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 6231932580842773fd14e5349f4b52a48d34c4cc
        tracker_head: e52129114c9fedeffbf179f3f3289eab76d199c9
        branch_point: 7d4e0c3aba6b39fb12d5125ee967dcde9e5be4be
        since: "2026-10-20T05:51:05-07:00"
        until: "2026-10-20T06:57:10-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T05:51:05-07:00"
              end: "2026-10-20T06:57:10-07:00"
              began: "2026-10-20T05:51:05-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: b5ae096b3fcbcf381130dcc357bd306b4d9bee5d02795f1ebc8cfa037d8ecb9b
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 192574592
              windows:
                - start: "2026-10-20T05:51:05-07:00"
                  end: "2026-10-20T06:57:10-07:00"
            - harness: codex
              id: 01a1595d-fe1a-7d60-b4df-d526bd4328c3
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 4b0c7115e7efcc40d4223b3ab5d0f51d9b804b8ca71968931063f3f12d6cdd62
              started: "2026-10-20T12:51:34.625Z"
              spawned: true
              bytes: 1194949
              windows:
                - start: "2026-10-20T12:51:34.625Z"
                  end: "2026-10-20T06:57:10-07:00"
            - harness: codex
              id: 01a1595e-faf3-7473-9eeb-792eec7597c8
              slot: pair:7
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 740d924d66204ffa0888e119de25da1eb2210214a0c0f2015e71c1fc15543dd3
              started: "2026-10-20T12:52:39.378Z"
              spawned: true
              bytes: 1187226
              windows:
                - start: "2026-10-20T12:52:39.378Z"
                  end: "2026-10-20T06:57:10-07:00"
        queue_waits:
            - attempt_id: 0d6db2414b5bdc28294ef055d6fd039b
              slot: pair:7
              record_digest: e4e8cd560d9eae68959b2a742e7b0770e886fcebfb7e6c4b39b90008121079c9
              start: "2026-10-20T13:27:10.393854Z"
              end: "2026-10-20T13:47:41.019786Z"
              queue_seconds: 1230.625932
              raw_seconds: 1230.625932
              exact: true
        cuts:
            - dispatched: "2026-10-20T06:10:25-07:00"
              finalized: "2026-10-20T06:13:24-07:00"
              finalizing: false
            - dispatched: "2026-10-20T06:27:25-07:00"
              finalized: "2026-10-20T06:29:27-07:00"
              finalizing: true
      "n": 3
      timestamp: "2026-10-20T06:59:32-07:00"
      dispatched: "2026-10-20T06:58:21-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      recipe: fix-review
      isolation: fenced
      reviewed: 6231932580842773fd14e5349f4b52a48d34c4cc
      blocked: false
---

# Gate ledger — pair#711 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T06:13:24-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `admission-registration-lifetime-fencing` Same-binding sender reconnect can admit a stale reservation
  At cmd/internal/couchmessage/broker.go:726, b.caller(from) checks current connectivity without comparing the sender registration with the one that began admission. A disposable regression disconnected and re-registered the sender after successful Reserve, then invoked recipient OperatorInput; Send returned a queued receipt with no error. This violates the Spec's unchanged-both-registrations condition. Enumerated admission sites requiring the same sender lifetime token are send's initial validation (574), admit's loop validation (631), tryAdmission's post-IO validation (726), and the busy-to-queue validation (684). Capture the sender actor/generation and preserve it across these paths. Add sender-reconnect controls alongside the existing recipient-reconnect control. ARCH-PURPOSE: sweep both registration lifetimes.
- **BR-2** [Critical] `validated-inventory-state-consumers` Runtime listing discards validated exited states and reclassifies names incorrectly
  At cmd/internal/launcher/osruntime.go:334-353, ListSessions extracts names from sessionListingStates but discards its state values. Its downstream buildListRowsForScope at list.go:78 calls zellijsession.RowState, which uses Fields[0] and Contains("EXITED"). Disposable probes showed a live pair-EXITED-is-a-name displayed as exited, while exited names containing spaces or tabs displayed as detached and triggered client queries. The remaining production instance in this window is ListSessions -> buildListRowsForScope -> RowState; ZellijSource.snapshot already consumes validated states correctly. Pass the validated state map into the runtime row builder and add these three edge cases. ARCH-DRY and ARCH-PURPOSE: all promised inventory consumers must use the same identity/state interpretation.
- **BR-3** [Important] `availability-refusal-phase-context` Admission races still return bare, misleading unavailable errors
  The send path retains bare ErrUnavailable returns at cmd/internal/couchmessage/broker.go:576, :633, and :685. These are the complete unlabelled registration-refusal instances in the admission path: initial caller validation, caller validation after routing IO, and combined caller/recipient validation after reservation release. A disposable regression disconnected the caller inside the injected family source; Send returned "recipient is not live", losing both the failing side and phase. Wrap each caller failure with caller/current context, split the combined fallback guard by side, preserve errors.Is classification, and retain regression coverage. ARCH-PURPOSE: diagnostics must cover concurrent invalidation as well as initially missing registrations.

## Round 2 — 2026-10-20T06:29:27-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — broker.go captures the sender actor under lock and checks its identity through routing, reservation, and queue fallback. Removing the identity comparison makes TestBrokerSenderReconnectCannotAuthorizeOldAdmission fail in all three phases.
- BR-2 — addressed — OSRuntime.ListSessions passes validated states into buildListRowsForScope. Restoring RowState interpretation makes TestRuntimeListSessionsUsesValidatedOpaqueNameState fail for all three opaque-name cases.
- BR-3 — addressed — Admission refusals identify caller or recipient and phase while preserving errors.Is classification. Removing the contextual wrappers makes TestBrokerAdmissionLossNamesSideAndPhase fail in all four cases.

## Round 3 — 2026-10-20T06:59:32-07:00 (codex) — passed

## Open findings

(none — every finding has been disposed)
