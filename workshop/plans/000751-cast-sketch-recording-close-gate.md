---
gate: boundary-review
issue: 751
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: f17ab95c-0850-4f9d-ad1a-09152d543277
            repository: github.com/xianxu/pair
            issue_id: "000751"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/182d3d354475ba4e
            reviewed_head: 19bc7a71951e1555bf8e94be342458e8e9d3954c
            slot: pair:9
            requested_at: "2026-10-21T05:11:42.339327Z"
          phase: finished
          pid: 9808
          started_at: "2026-10-21T05:11:42.533718Z"
          finished_at: "2026-10-21T05:15:10.232365Z"
          session_id: 01a15cdf-564d-77e3-bbee-607a691a7afc
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 19bc7a71951e1555bf8e94be342458e8e9d3954c
        tracker_head: c6d5c78624c8a33b74633aebda62f17952bd8942
        branch_point: a1630f0c07c8ac61814c74131799fa53357db8f3
        since: "2026-10-20T21:40:18-07:00"
        until: "2026-10-20T22:10:54-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-20T21:40:18-07:00"
              end: "2026-10-20T22:10:54-07:00"
              began: "2026-10-20T21:40:18-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a15c54-f5ad-7701-bb23-643678932dc5
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 52cc53193a8337e2ec8faf79504fb2ee8e177746fe7f583c7d17d625030b3bb9
              started: "2026-10-21T02:40:36.565Z"
              spawned: false
              bytes: 5805454
              windows:
                - start: "2026-10-20T21:40:18-07:00"
                  end: "2026-10-20T22:10:54-07:00"
            - harness: codex
              id: 01a15cc4-b370-7d60-9efa-989c5b30869b
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: b20919fc471a73fa9e842b50d087f2f386de97cb3f1576307c3c873e99e0197f
              started: "2026-10-21T04:42:37.82Z"
              spawned: false
              bytes: 3617717
              windows:
                - start: "2026-10-21T04:42:37.82Z"
                  end: "2026-10-20T22:10:54-07:00"
            - harness: codex
              id: 01a15cc5-6b8f-71a1-8e17-c4cc52d9e85a
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: cd5d16cba20794a5a84448d5398a36320574589153f150af365f2336b69eb08c
              started: "2026-10-21T04:43:24.808Z"
              spawned: false
              bytes: 755248
              windows:
                - start: "2026-10-21T04:43:24.808Z"
                  end: "2026-10-20T22:10:54-07:00"
            - harness: codex
              id: 01a15cc5-ea5a-7cf0-a5c1-9051238b27d1
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15cc4-b370-7d60-9efa-989c5b30869b
              digest: 325f863a81289a557dfdc537171465d4051b9ceae90f8fec73654e11c351d433
              started: "2026-10-21T04:43:57Z"
              spawned: true
              bytes: 1376983
              windows:
                - start: "2026-10-21T04:43:57Z"
                  end: "2026-10-20T22:10:54-07:00"
            - harness: codex
              id: 01a15cc6-9120-7fb2-bb16-894e702abf37
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15cc4-b370-7d60-9efa-989c5b30869b
              digest: 7defeb83ce407fcb8d54b4e8c0419d0b3a01ae1cfdb6c20154866b62d49c5d97
              started: "2026-10-21T04:44:39.739Z"
              spawned: true
              bytes: 1994113
              windows:
                - start: "2026-10-21T04:44:39.739Z"
                  end: "2026-10-20T22:10:54-07:00"
            - harness: codex
              id: 01a15cc8-fb76-78e2-8c9a-cdf39be72681
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 0d6b70d47812dc688eeff8a6443136dfe2a5ffc1af14546f80a19344585bf995
              started: "2026-10-21T04:47:18.44Z"
              spawned: false
              bytes: 434166
              windows:
                - start: "2026-10-21T04:47:18.44Z"
                  end: "2026-10-20T22:10:54-07:00"
            - harness: codex
              id: 01a15cc9-abec-7092-a674-5e0fe95f3218
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              parent_id: 01a15cc4-b370-7d60-9efa-989c5b30869b
              digest: 7ef015569755c9399f3f2d08ace36feb96585fa58905e34daa8c8cb140832573
              started: "2026-10-21T04:48:03.142Z"
              spawned: true
              bytes: 1465685
              windows:
                - start: "2026-10-21T04:48:03.142Z"
                  end: "2026-10-20T22:10:54-07:00"
            - harness: codex
              id: 01a15cca-866d-7c81-a3c6-38ba2cd75a01
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: e8cf83422e4669e2b9138655b6a3f5c137dba172f889877b599f3866e884ae25
              started: "2026-10-21T04:48:59.411Z"
              spawned: false
              bytes: 429653
              windows:
                - start: "2026-10-21T04:48:59.411Z"
                  end: "2026-10-20T22:10:54-07:00"
            - harness: codex
              id: 01a15cca-fa91-74b0-8ae6-64d332f5fa13
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: fe18e0e86ae75dbf382e9c693e4fec00ac41f3e526086dbe76b9664d08e8ec2a
              started: "2026-10-21T04:49:29.133Z"
              spawned: false
              bytes: 463540
              windows:
                - start: "2026-10-21T04:49:29.133Z"
                  end: "2026-10-20T22:10:54-07:00"
      "n": 1
      timestamp: "2026-10-20T22:15:11-07:00"
      dispatched: "2026-10-20T22:11:39-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Signed zero produces different Go and JavaScript history_bytes
          detail: cmd/internal/webcast/recording/sketch_state.go:149 counts json.Marshal(known), which preserves a float64 negative zero as -0. cmd/internal/webcast/web/sketch-model.js:229 instead counts sketchCanonical output, which normalizes -0 to 0. A legal negative-zero coordinate therefore adds one byte in Go but not JavaScript, violating the required prefix-projection agreement and producing different cap admission arithmetic. Use one defined numeric normalization for model accounting in both languages, retain exact wire accounting in SketchBudget, and add a shared prefix regression using math.Copysign(0, -1).
          family: cross-language-history-accounting
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Browser callers can bypass sketch transitions through public mutable history
          detail: cmd/internal/webcast/web/model.js:75 exposes the authoritative state as model.sketches, including its mutable maps, points, stacks, receipts, and sequence counters. A caller can change accepted geometry or operation authority without nextSketch validation; a direct probe changed a stored point to 999 and next_sequence to 12345, and sketchProjection reported both changes. ARCH-ORDER requires bypass prevention, and the plan explicitly promises read-only projections. Keep production history private and expose owned projections, with a regression proving callers cannot mutate authoritative state.
          family: authoritative-state-encapsulation
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#751 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-20T22:15:11-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `cross-language-history-accounting` (kind: defect) Signed zero produces different Go and JavaScript history_bytes
  cmd/internal/webcast/recording/sketch_state.go:149 counts json.Marshal(known), which preserves a float64 negative zero as -0. cmd/internal/webcast/web/sketch-model.js:229 instead counts sketchCanonical output, which normalizes -0 to 0. A legal negative-zero coordinate therefore adds one byte in Go but not JavaScript, violating the required prefix-projection agreement and producing different cap admission arithmetic. Use one defined numeric normalization for model accounting in both languages, retain exact wire accounting in SketchBudget, and add a shared prefix regression using math.Copysign(0, -1).
- **BR-2** [Important] `authoritative-state-encapsulation` (kind: defect) Browser callers can bypass sketch transitions through public mutable history
  cmd/internal/webcast/web/model.js:75 exposes the authoritative state as model.sketches, including its mutable maps, points, stacks, receipts, and sequence counters. A caller can change accepted geometry or operation authority without nextSketch validation; a direct probe changed a stored point to 999 and next_sequence to 12345, and sketchProjection reported both changes. ARCH-ORDER requires bypass prevention, and the plan explicitly promises read-only projections. Keep production history private and expose owned projections, with a regression proving callers cannot mutate authoritative state.

## Open findings

- **BR-1** [Critical] `cross-language-history-accounting` (kind: defect) Signed zero produces different Go and JavaScript history_bytes
- **BR-2** [Important] `authoritative-state-encapsulation` (kind: defect) Browser callers can bypass sketch transitions through public mutable history
