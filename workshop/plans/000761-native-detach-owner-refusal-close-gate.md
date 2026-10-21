---
gate: boundary-review
issue: 761
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: b3547ed7-dd71-4027-b427-696a0350f235
            repository: github.com/xianxu/pair
            issue_id: "000761"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/86768c4d44ff1549
            reviewed_head: e5bb4257eab53a300f3a52ed854f44b95778dd2d
            slot: pair:5
            requested_at: "2026-10-21T09:09:47.205831Z"
          phase: finished
          pid: 84785
          started_at: "2026-10-21T09:09:47.367593Z"
          finished_at: "2026-10-21T09:14:58.735062Z"
          session_id: 01a15db9-4e4d-77d2-a865-c918e37d2c48
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: e5bb4257eab53a300f3a52ed854f44b95778dd2d
        tracker_head: 54d0c0137485814ee06b8eae87c67bb3dda7bb0a
        branch_point: bc79cbaeae75b518dcfa789fc2bd0e40ccf06f97
        since: "2026-10-21T00:55:41-07:00"
        until: "2026-10-21T02:09:09-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T00:55:41-07:00"
              end: "2026-10-21T02:09:09-07:00"
              began: "2026-10-21T00:55:41-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot5/pair
        sources:
            - harness: codex
              id: 01a1574c-1cb1-7e92-adfe-354342364799
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: faa0ced8c806a1ddaffe563a2614274eb4e44d0d28627341bab91d3581454ab9
              started: "2026-10-20T03:12:49.04Z"
              spawned: false
              bytes: 47316155
              windows:
                - start: "2026-10-21T00:55:41-07:00"
                  end: "2026-10-21T02:09:09-07:00"
            - harness: codex
              id: 01a15d76-5d32-7611-8937-2ae8220080be
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a1574c-1cb1-7e92-adfe-354342364799
              digest: 8421206e17bbdff88d386cce38fc8ac945b9b270395f1c24bf91aee139884aea
              started: "2026-10-21T07:56:40.671Z"
              spawned: true
              bytes: 2345517
              windows:
                - start: "2026-10-21T07:56:40.671Z"
                  end: "2026-10-21T02:09:09-07:00"
            - harness: codex
              id: 01a15d76-883d-7272-918b-503eb29cf337
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a1574c-1cb1-7e92-adfe-354342364799
              digest: d75c7833f57e68546dc006f0ba74422ea4a395188c71ef8a675087ba4efe752d
              started: "2026-10-21T07:56:51.688Z"
              spawned: true
              bytes: 1909074
              windows:
                - start: "2026-10-21T07:56:51.688Z"
                  end: "2026-10-21T02:09:09-07:00"
            - harness: codex
              id: 01a15d88-eea1-7ff2-86a4-7c402fc62671
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 6054c79fabdd2554d6dfb09a18376e52c9cdf9aa3ddd73497ea2a405e4961fe0
              started: "2026-10-21T08:16:58.191Z"
              spawned: false
              bytes: 626481
              windows:
                - start: "2026-10-21T08:16:58.191Z"
                  end: "2026-10-21T02:09:09-07:00"
            - harness: codex
              id: 01a15d8b-ab4a-7032-9d24-c618d0c8beef
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 2ae00fb7deca15fb7a4ca8d2ad8f09816f0bca7904f6232cdea7189ff136f56f
              started: "2026-10-21T08:19:57.36Z"
              spawned: false
              bytes: 389127
              windows:
                - start: "2026-10-21T08:19:57.36Z"
                  end: "2026-10-21T02:09:09-07:00"
            - harness: codex
              id: 01a15d8d-538c-7383-8709-a325d12123b6
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 0809a13bd44c0a778340458a9fd89eee08550a6f8ead27d831ee9717e3cbbe9c
              started: "2026-10-21T08:21:45.782Z"
              spawned: false
              bytes: 476379
              windows:
                - start: "2026-10-21T08:21:45.782Z"
                  end: "2026-10-21T02:09:09-07:00"
            - harness: codex
              id: 01a15d8d-fbf7-74a2-bd62-1af1b40f8d70
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 7422d2cb0b2a9189c8fee3bc9f142b183db414f8270a80a0e7790521fd4db071
              started: "2026-10-21T08:22:28.962Z"
              spawned: false
              bytes: 158750
              windows:
                - start: "2026-10-21T08:22:28.962Z"
                  end: "2026-10-21T02:09:09-07:00"
            - harness: codex
              id: 01a15d8e-0865-7ee2-9c60-3839930cf00f
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: e0539b9c7d7401388bc42d0d7ffb23ca7a46b260bdbc1a3b7d2f0828a42b9990
              started: "2026-10-21T08:22:32.421Z"
              spawned: false
              bytes: 422024
              windows:
                - start: "2026-10-21T08:22:32.421Z"
                  end: "2026-10-21T02:09:09-07:00"
        queue_waits:
            - attempt_id: 0b7f92c1bacfa84d218f99423185fa89
              slot: pair:5
              record_digest: 432345151d95bd11f26218ac6288672ce665a09bf42795cf6ae30c5264cbb6cb
              start: "2026-10-21T08:51:12.692344Z"
              end: "2026-10-21T08:51:48.415458Z"
              queue_seconds: 35.723114
              raw_seconds: 35.723114
              exact: true
      "n": 1
      timestamp: "2026-10-21T02:14:59-07:00"
      dispatched: "2026-10-21T02:09:44-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Preserve native socket-root spelling when checking retry admission
          detail: cmd/internal/launcher/zellij_read_recovery.go:118 uses filepath.Join, which cleans symlink/.. segments that ResolveZellijSocketRoot deliberately preserves (zellij_socket.go:18–19). For root /tmp/example/alias/../sockets, where alias points to real/child, Zellij addresses real/sockets while this check addresses example/sockets. A filesystem reproduction confirmed the native socket existed while the cleaned path did not; the reverse arrangement can admit retries based on an unrelated socket. This violates the approved exact-target-socket condition (ARCH-SECURE). Append validated path components without cleaning the root, and add positive and negative symlink/parent-segment regressions.
          family: native-path-spelling-preservation
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Bound the socket fixture path independently of inherited TMPDIR
          detail: 'cmd/internal/launcher/zellij_lookup_miss_test.go:46 creates its socket root beneath inherited TMPDIR without accounting for the Unix socket pathname limit (ARCH-CONSTRAINTS). The requested package run failed four new tests with bind: invalid argument under this review''s TMPDIR; the generated paths were approximately 119 bytes. The focused tests passed with TMPDIR=/tmp. Use a short private socket root with cleanup, following existing repository socket fixtures, so ordinary test execution works under long temporary roots.'
          family: unix-socket-fixture-path-budget
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 90800df8-3dc0-4e8e-b16c-44bc6b4e8105
            repository: github.com/xianxu/pair
            issue_id: "000761"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/0cfbfc9bba29a20d
            reviewed_head: f7e4c03f134362d8173bb73725f014677aa464be
            slot: pair:5
            requested_at: "2026-10-21T09:29:49.668376Z"
          phase: finished
          pid: 16967
          started_at: "2026-10-21T09:29:49.854414Z"
          finished_at: "2026-10-21T09:34:25.288721Z"
          session_id: 01a15dcb-a7a7-7c11-8b4e-cc36cb9adf62
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: f7e4c03f134362d8173bb73725f014677aa464be
        tracker_head: fc3551376ff18b49bd0eb254e5c398f6395122ed
        branch_point: bc79cbaeae75b518dcfa789fc2bd0e40ccf06f97
        since: "2026-10-21T00:55:41-07:00"
        until: "2026-10-21T02:29:00-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T00:55:41-07:00"
              end: "2026-10-21T02:29:00-07:00"
              began: "2026-10-21T00:55:41-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot5/pair
        sources:
            - harness: codex
              id: 01a1574c-1cb1-7e92-adfe-354342364799
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 93034174d359d4133792c357ffa036636d4c5e4d5eadc09904f1ab5e77f81787
              started: "2026-10-20T03:12:49.04Z"
              spawned: false
              bytes: 48159330
              windows:
                - start: "2026-10-21T00:55:41-07:00"
                  end: "2026-10-21T02:29:00-07:00"
            - harness: codex
              id: 01a15d76-5d32-7611-8937-2ae8220080be
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a1574c-1cb1-7e92-adfe-354342364799
              digest: ebb0c5926d10554dd51a09e382e99922c409aa1a4e5072a0aafbe514fd5d75f6
              started: "2026-10-21T07:56:40.671Z"
              spawned: true
              bytes: 2615865
              windows:
                - start: "2026-10-21T07:56:40.671Z"
                  end: "2026-10-21T02:29:00-07:00"
            - harness: codex
              id: 01a15d76-883d-7272-918b-503eb29cf337
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a1574c-1cb1-7e92-adfe-354342364799
              digest: 6ad4cb98a6f878641bcb1a875efcb16172196be707e4d330ea2ab44a2c675a10
              started: "2026-10-21T07:56:51.688Z"
              spawned: true
              bytes: 2255341
              windows:
                - start: "2026-10-21T07:56:51.688Z"
                  end: "2026-10-21T02:29:00-07:00"
            - harness: codex
              id: 01a15d88-eea1-7ff2-86a4-7c402fc62671
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 6054c79fabdd2554d6dfb09a18376e52c9cdf9aa3ddd73497ea2a405e4961fe0
              started: "2026-10-21T08:16:58.191Z"
              spawned: false
              bytes: 626481
              windows:
                - start: "2026-10-21T08:16:58.191Z"
                  end: "2026-10-21T02:29:00-07:00"
            - harness: codex
              id: 01a15d8b-ab4a-7032-9d24-c618d0c8beef
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 2ae00fb7deca15fb7a4ca8d2ad8f09816f0bca7904f6232cdea7189ff136f56f
              started: "2026-10-21T08:19:57.36Z"
              spawned: false
              bytes: 389127
              windows:
                - start: "2026-10-21T08:19:57.36Z"
                  end: "2026-10-21T02:29:00-07:00"
            - harness: codex
              id: 01a15d8d-538c-7383-8709-a325d12123b6
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 0809a13bd44c0a778340458a9fd89eee08550a6f8ead27d831ee9717e3cbbe9c
              started: "2026-10-21T08:21:45.782Z"
              spawned: false
              bytes: 476379
              windows:
                - start: "2026-10-21T08:21:45.782Z"
                  end: "2026-10-21T02:29:00-07:00"
            - harness: codex
              id: 01a15d8d-fbf7-74a2-bd62-1af1b40f8d70
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 7422d2cb0b2a9189c8fee3bc9f142b183db414f8270a80a0e7790521fd4db071
              started: "2026-10-21T08:22:28.962Z"
              spawned: false
              bytes: 158750
              windows:
                - start: "2026-10-21T08:22:28.962Z"
                  end: "2026-10-21T02:29:00-07:00"
            - harness: codex
              id: 01a15d8e-0865-7ee2-9c60-3839930cf00f
              slot: pair:5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: e0539b9c7d7401388bc42d0d7ffb23ca7a46b260bdbc1a3b7d2f0828a42b9990
              started: "2026-10-21T08:22:32.421Z"
              spawned: false
              bytes: 422024
              windows:
                - start: "2026-10-21T08:22:32.421Z"
                  end: "2026-10-21T02:29:00-07:00"
        queue_waits:
            - attempt_id: 0b7f92c1bacfa84d218f99423185fa89
              slot: pair:5
              record_digest: 432345151d95bd11f26218ac6288672ce665a09bf42795cf6ae30c5264cbb6cb
              start: "2026-10-21T08:51:12.692344Z"
              end: "2026-10-21T08:51:48.415458Z"
              queue_seconds: 35.723114
              raw_seconds: 35.723114
              exact: true
        cuts:
            - dispatched: "2026-10-21T02:09:44-07:00"
              finalized: "2026-10-21T02:14:59-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-21T02:34:26-07:00"
      dispatched: "2026-10-21T02:29:47-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'The explicit-root regressions pass and fail when its fix is removed, but family native-path-spelling-preservation remains incomplete. zellij_read_recovery.go:114 calls ResolveZellijSocketRoot, whose TMPDIR fallback at zellij_socket.go:28 still uses filepath.Join. With ZELLIJ_SOCKET_DIR unset and TMPDIR containing alias/../sockets, a disposable pinned-head regression observed native-present/admission-false and native-absent/admission-true. Preserve native spelling across all three root sources: explicit override, Linux runtime directory, and temporary-directory fallback; add positive and negative fallback regressions.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: 'All three launcher Unix listener sites use the budgeted private allocator. The long-TMPDIR regression passes and fails with bind: invalid argument when the lookup fixture returns to inherited temporary allocation. The full launcher package also passed under a 261-byte inherited TMPDIR; temporary review checkouts were removed.'
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#761 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-21T02:14:59-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `native-path-spelling-preservation` (kind: defect) Preserve native socket-root spelling when checking retry admission
  cmd/internal/launcher/zellij_read_recovery.go:118 uses filepath.Join, which cleans symlink/.. segments that ResolveZellijSocketRoot deliberately preserves (zellij_socket.go:18–19). For root /tmp/example/alias/../sockets, where alias points to real/child, Zellij addresses real/sockets while this check addresses example/sockets. A filesystem reproduction confirmed the native socket existed while the cleaned path did not; the reverse arrangement can admit retries based on an unrelated socket. This violates the approved exact-target-socket condition (ARCH-SECURE). Append validated path components without cleaning the root, and add positive and negative symlink/parent-segment regressions.
- **BR-2** [Important] `unix-socket-fixture-path-budget` (kind: defect) Bound the socket fixture path independently of inherited TMPDIR
  cmd/internal/launcher/zellij_lookup_miss_test.go:46 creates its socket root beneath inherited TMPDIR without accounting for the Unix socket pathname limit (ARCH-CONSTRAINTS). The requested package run failed four new tests with bind: invalid argument under this review's TMPDIR; the generated paths were approximately 119 bytes. The focused tests passed with TMPDIR=/tmp. Use a short private socket root with cleanup, following existing repository socket fixtures, so ordinary test execution works under long temporary roots.

## Round 2 — 2026-10-21T02:34:26-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — not-addressed — The explicit-root regressions pass and fail when its fix is removed, but family native-path-spelling-preservation remains incomplete. zellij_read_recovery.go:114 calls ResolveZellijSocketRoot, whose TMPDIR fallback at zellij_socket.go:28 still uses filepath.Join. With ZELLIJ_SOCKET_DIR unset and TMPDIR containing alias/../sockets, a disposable pinned-head regression observed native-present/admission-false and native-absent/admission-true. Preserve native spelling across all three root sources: explicit override, Linux runtime directory, and temporary-directory fallback; add positive and negative fallback regressions.
- BR-2 — addressed — All three launcher Unix listener sites use the budgeted private allocator. The long-TMPDIR regression passes and fails with bind: invalid argument when the lookup fixture returns to inherited temporary allocation. The full launcher package also passed under a 261-byte inherited TMPDIR; temporary review checkouts were removed.

## Open findings

- **BR-1** [Critical] `native-path-spelling-preservation` (kind: defect) Preserve native socket-root spelling when checking retry admission
