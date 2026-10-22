---
gate: boundary-review
issue: 809
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 2ec51ec3-8281-479f-b400-c95a3f6eb527
            repository: github.com/xianxu/pair
            issue_id: "000809"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/6891b03376a948f0
            reviewed_head: 15c56a7a55ebe7363d566a640d2e94be3471c41b
            slot: pair:8
            requested_at: "2026-10-22T21:15:09.579029Z"
          phase: finished
          pid: 3367
          started_at: "2026-10-22T21:15:09.864261Z"
          finished_at: "2026-10-22T21:16:36.58377Z"
          session_id: 22e8b16d-1209-4591-90ca-83e36c1aef15
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 15c56a7a55ebe7363d566a640d2e94be3471c41b
        tracker_head: bbb642a189060eeba5fe89bd867a948e727de6d4
        branch_point: 7b9440e8c92663f8ee189f54a1260e3043a78f07
        since: "2026-10-22T12:41:48-07:00"
        until: "2026-10-22T14:14:07-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T12:41:48-07:00"
              end: "2026-10-22T14:14:07-07:00"
              began: "2026-10-22T12:41:48-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot8/pair
        sources:
            - harness: qoder
              id: d0d6b474-f0bb-4817-a59c-ca5442029afe
              slot: pair:8
              cwd: /Users/xianxu/workspace/worktree/pair-slot8/pair
              digest: 1b17394dc182509bc8be35ca8be5a0d478bb765cc40615f648771d2c18e81a24
              started: "2026-10-22T19:40:15.447Z"
              spawned: false
              bytes: 2727240
              windows:
                - start: "2026-10-22T12:41:48-07:00"
                  end: "2026-10-22T14:14:07-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 34.2714
      "n": 1
      timestamp: "2026-10-22T14:16:38-07:00"
      dispatched: "2026-10-22T14:15:05-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: Fixtures in scribecmd still call pty.Open directly despite the doc claim that they use openPTY
          detail: 'pty_darwin_test.go:625 (TestDarwinRunInteractiveResizeAndRestore) calls pty.Open() directly for its outer terminal, so it is exposed to the #809 race. pty_darwin.go:32-34 and atlas/architecture.md:1354-1356 state that fixtures allocate through openPTY. Second instance in the window: the new test''s injected opener at pty_darwin_test.go:427 calls raw pty.Open inside the counted closure, so a real ENXIO shifts the opened count, and in the last-attempt case uses up the bound. Fix both to call openPTY(). Line 367 is already wrapped by openPTYRetry and needs no change.'
          family: fixture-pty-alloc-bypasses-shared-retry
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Done-when still says never a retry loop, but the change ships a bounded retry
          detail: The Spec records the hub ruling that allows a bounded, ENXIO-only, darwin-only production retry, but Done-when was not updated and its literal text contradicts the delivered openPTYRetry loop. Reword it to ban test-side or blind retries and timeout extensions, and allow the ruled bounded allocation retry.
          family: done-when-tracks-contract
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: Atlas says the negated errno comes from the raw syscall path only
          detail: The Spec and the code comment say the negated errno was measured from syscall.Open as well as from the raw ioctl syscalls. The atlas sentence narrows this to the raw syscall path only.
          family: doc-claim-narrower-than-measurement
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#809 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-22T14:16:38-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `fixture-pty-alloc-bypasses-shared-retry` (kind: defect) Fixtures in scribecmd still call pty.Open directly despite the doc claim that they use openPTY
  pty_darwin_test.go:625 (TestDarwinRunInteractiveResizeAndRestore) calls pty.Open() directly for its outer terminal, so it is exposed to the #809 race. pty_darwin.go:32-34 and atlas/architecture.md:1354-1356 state that fixtures allocate through openPTY. Second instance in the window: the new test's injected opener at pty_darwin_test.go:427 calls raw pty.Open inside the counted closure, so a real ENXIO shifts the opened count, and in the last-attempt case uses up the bound. Fix both to call openPTY(). Line 367 is already wrapped by openPTYRetry and needs no change.
- **BR-2** [Important] `done-when-tracks-contract` (kind: defect) Done-when still says never a retry loop, but the change ships a bounded retry
  The Spec records the hub ruling that allows a bounded, ENXIO-only, darwin-only production retry, but Done-when was not updated and its literal text contradicts the delivered openPTYRetry loop. Reword it to ban test-side or blind retries and timeout extensions, and allow the ruled bounded allocation retry.
- **BR-3** [Minor] `doc-claim-narrower-than-measurement` (kind: defect) Atlas says the negated errno comes from the raw syscall path only
  The Spec and the code comment say the negated errno was measured from syscall.Open as well as from the raw ioctl syscalls. The atlas sentence narrows this to the raw syscall path only.

## Open findings

- **BR-1** [Important] `fixture-pty-alloc-bypasses-shared-retry` (kind: defect) Fixtures in scribecmd still call pty.Open directly despite the doc claim that they use openPTY
- **BR-2** [Important] `done-when-tracks-contract` (kind: defect) Done-when still says never a retry loop, but the change ships a bounded retry
- **BR-3** [Minor] `doc-claim-narrower-than-measurement` (kind: defect) Atlas says the negated errno comes from the raw syscall path only
