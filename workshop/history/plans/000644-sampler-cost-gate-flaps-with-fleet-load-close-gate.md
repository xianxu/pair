---
gate: boundary-review
issue: 644
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 114ae4ca-ee58-420a-8c07-f702c615c480
            repository: github.com/xianxu/pair
            issue_id: "000644"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/45504772956dc72f
            reviewed_head: 988914e084893cdf03b8d1467b18d463d1879e42
            slot: pair:4
            requested_at: "2026-10-18T06:27:20.249017Z"
          phase: finished
          pid: 3683
          started_at: "2026-10-18T06:27:20.466081Z"
          finished_at: "2026-10-18T06:28:33.595349Z"
          session_id: b2f6b2ee-c668-4cbc-895d-fe84a101a66c
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 988914e084893cdf03b8d1467b18d463d1879e42
        tracker_head: 30228b6117a0a0f89e3ef01df623460b4121a2bb
        branch_point: a8f352e17f3a1449df8d87c55151e76f3f4c6b31
        since: "2026-10-17T21:09:19-07:00"
        until: "2026-10-17T23:26:26-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T21:09:19-07:00"
              end: "2026-10-17T23:26:26-07:00"
              began: "2026-10-17T21:09:19-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: qoder
              id: 948602c0-7ead-4b32-9b7f-481e804dff72
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: 9b8f68daab8062d6f740f172cc6688ca705b72cd65e4d2f7aa46c433fbf948be
              started: "2026-10-14T15:43:14.672Z"
              spawned: false
              bytes: 28587588
              windows:
                - start: "2026-10-17T21:09:19-07:00"
                  end: "2026-10-17T23:26:26-07:00"
        queue_waits:
            - attempt_id: dbdf8d64d556e87195180367761a97a2
              slot: pair:4
              record_digest: a04d3a96833d621e71fd8996d3842e7d51c4d826f04f2eafbd8f327d191fbb47
              start: "2026-10-18T05:05:16.513959Z"
              end: "2026-10-18T05:13:31.313778Z"
              queue_seconds: 494.799819
              raw_seconds: 494.799819
              exact: true
            - attempt_id: a17fa89bf77fdb41b6805e0268c374c7
              slot: pair:4
              record_digest: da53792936cf42bc977424777dc40ef756cd9658470d1ab2f79e5daa8a4ac46d
              start: "2026-10-18T05:50:36.919956Z"
              end: "2026-10-18T06:10:09.617027Z"
              queue_seconds: 1172.697071
              raw_seconds: 1172.697071
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.34708333333333335
      "n": 1
      timestamp: "2026-10-17T23:28:35-07:00"
      dispatched: "2026-10-17T23:27:17-07:00"
      agent: claude
      findings:
        - kind: defect
          id: BR-1
          severity: Minor
          title: Atlas and test comments restate measured numbers that are wrong or already stale
          detail: 'Every instance in this window: (1) atlas/couch.md:4471 says the conformance "gates work rather than wall time", but the old gate was absolute CPU, not wall time, and the new gate still checks steady CPU as a ratio. (2) The TestResourceSamplerNativeCost doc comment says native calls "moved under 0.2%" across loadavg 15.10-112.12; that figure came from runs A-C at 30.70-34.77, while the verification runs show cold environment calls 1545-1555 (about 0.6%) and steady native calls 60-66. (3) The costSteadyRatioK comment says a cold-shaped pass costs "3.04-3.76" of a write, but run 4 recorded 2.62 and this review measured 3.78. (4) The per-pass table-read comment says "at least 38% of rows of slack", which is what one host happened to show, not a guarantee. Cite #644 instead of restating the numbers, as the hub ruled for the atlas.'
          family: prose-claims-restate-measurements
          round: 1
        - kind: defect
          id: BR-2
          severity: Minor
          title: The persistence-error t.Fatal in the pass loop does not name the host state
          detail: In TestResourceSamplerNativeCost, `t.Fatal(e)` for a persistence error is the only failure inside the pass loop that does not append the `(rows=... loadavg=...)` host string; the incomplete-measurement Fatalf, the work-bound Errorf and the steady-ratio Errorf all include it.
          family: failure-messages-name-host-state
          round: 1
      recipe: small-diff-review
      isolation: fenced
      reviewed: 988914e084893cdf03b8d1467b18d463d1879e42
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 44dfc06b-546a-4b44-89ac-422bc5438b61
            repository: github.com/xianxu/pair
            issue_id: "000644"
            recipe: fix-review
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/85ee5b0da3e511b5
            reviewed_head: 9f0c6eec78e7b9417a08817df5e273d0982f3784
            slot: pair:4
            requested_at: "2026-10-18T06:34:43.016476Z"
          phase: finished
          pid: 87153
          started_at: "2026-10-18T06:34:43.254956Z"
          finished_at: "2026-10-18T06:35:29.421945Z"
          session_id: 097011ba-878f-44d0-b975-8a2f032f7506
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 9f0c6eec78e7b9417a08817df5e273d0982f3784
        tracker_head: a60e3b697804d1b5cf064dba7b7d435a8ebc140f
        branch_point: a8f352e17f3a1449df8d87c55151e76f3f4c6b31
        since: "2026-10-17T21:09:19-07:00"
        until: "2026-10-17T23:33:56-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T21:09:19-07:00"
              end: "2026-10-17T23:33:56-07:00"
              began: "2026-10-17T21:09:19-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot4/pair
        sources:
            - harness: qoder
              id: 948602c0-7ead-4b32-9b7f-481e804dff72
              slot: pair:4
              cwd: /Users/xianxu/workspace/worktree/pair-slot4/pair
              digest: f540a66b49b28d9e5e4c783018dc5763c5ebb793b6953880185127438fa7789d
              started: "2026-10-14T15:43:14.672Z"
              spawned: false
              bytes: 28808811
              windows:
                - start: "2026-10-17T21:09:19-07:00"
                  end: "2026-10-17T23:33:56-07:00"
        queue_waits:
            - attempt_id: dbdf8d64d556e87195180367761a97a2
              slot: pair:4
              record_digest: a04d3a96833d621e71fd8996d3842e7d51c4d826f04f2eafbd8f327d191fbb47
              start: "2026-10-18T05:05:16.513959Z"
              end: "2026-10-18T05:13:31.313778Z"
              queue_seconds: 494.799819
              raw_seconds: 494.799819
              exact: true
            - attempt_id: a17fa89bf77fdb41b6805e0268c374c7
              slot: pair:4
              record_digest: da53792936cf42bc977424777dc40ef756cd9658470d1ab2f79e5daa8a4ac46d
              start: "2026-10-18T05:50:36.919956Z"
              end: "2026-10-18T06:10:09.617027Z"
              queue_seconds: 1172.697071
              raw_seconds: 1172.697071
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 0.34708333333333335
        cuts:
            - dispatched: "2026-10-17T23:27:17-07:00"
              finalized: "2026-10-17T23:28:35-07:00"
              finalizing: true
      "n": 2
      timestamp: "2026-10-17T23:35:31-07:00"
      dispatched: "2026-10-17T23:34:38-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: All four instances fixed (atlas/couch.md:4471-4473, resources_test.go:333-338, 346-354, 452-455); they now cite the issue 644 Log instead of restating numbers, and a sweep of the test comments and atlas paragraph found no other restated measurement.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The persistence error now calls t.Fatalf with the host string. The two t.Fatal(err) calls left in the loop are a table-read error raised before host exists and an os.Stat setup error; neither is a measurement verdict.
          round: 2
      recipe: fix-review
      isolation: fenced
      reviewed: 9f0c6eec78e7b9417a08817df5e273d0982f3784
      blocked: false
---

# Gate ledger — pair#644 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-17T23:28:35-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `prose-claims-restate-measurements` (kind: defect) Atlas and test comments restate measured numbers that are wrong or already stale
  Every instance in this window: (1) atlas/couch.md:4471 says the conformance "gates work rather than wall time", but the old gate was absolute CPU, not wall time, and the new gate still checks steady CPU as a ratio. (2) The TestResourceSamplerNativeCost doc comment says native calls "moved under 0.2%" across loadavg 15.10-112.12; that figure came from runs A-C at 30.70-34.77, while the verification runs show cold environment calls 1545-1555 (about 0.6%) and steady native calls 60-66. (3) The costSteadyRatioK comment says a cold-shaped pass costs "3.04-3.76" of a write, but run 4 recorded 2.62 and this review measured 3.78. (4) The per-pass table-read comment says "at least 38% of rows of slack", which is what one host happened to show, not a guarantee. Cite #644 instead of restating the numbers, as the hub ruled for the atlas.
- **BR-2** [Minor] `failure-messages-name-host-state` (kind: defect) The persistence-error t.Fatal in the pass loop does not name the host state
  In TestResourceSamplerNativeCost, `t.Fatal(e)` for a persistence error is the only failure inside the pass loop that does not append the `(rows=... loadavg=...)` host string; the incomplete-measurement Fatalf, the work-bound Errorf and the steady-ratio Errorf all include it.

## Round 2 — 2026-10-17T23:35:31-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — All four instances fixed (atlas/couch.md:4471-4473, resources_test.go:333-338, 346-354, 452-455); they now cite the issue 644 Log instead of restating numbers, and a sweep of the test comments and atlas paragraph found no other restated measurement.
- BR-2 — addressed — The persistence error now calls t.Fatalf with the host string. The two t.Fatal(err) calls left in the loop are a table-read error raised before host exists and an os.Stat setup error; neither is a measurement verdict.

## Open findings

(none — every finding has been disposed)
