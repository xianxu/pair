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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 61c316c8-c663-46a6-8f26-7379f9068e45
            repository: github.com/xianxu/pair
            issue_id: "000809"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/81202c01342861a5
            reviewed_head: b54e8064574d5962a25811704f3ad4998078a30a
            slot: pair:8
            requested_at: "2026-10-22T21:21:37.102485Z"
          phase: finished
          pid: 27182
          started_at: "2026-10-22T21:21:37.26276Z"
          finished_at: "2026-10-22T21:23:29.298624Z"
          session_id: 8eeca7db-7271-4824-9bd5-133b269d9074
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: b54e8064574d5962a25811704f3ad4998078a30a
        tracker_head: a4dbb49b937e278cbfc3f3ec29c23c7bc36f4b11
        branch_point: 7b9440e8c92663f8ee189f54a1260e3043a78f07
        since: "2026-10-22T12:41:48-07:00"
        until: "2026-10-22T14:20:34-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T12:41:48-07:00"
              end: "2026-10-22T14:20:34-07:00"
              began: "2026-10-22T12:41:48-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot8/pair
        sources:
            - harness: qoder
              id: d0d6b474-f0bb-4817-a59c-ca5442029afe
              slot: pair:8
              cwd: /Users/xianxu/workspace/worktree/pair-slot8/pair
              digest: a44aa9b7903a410572c9a7f9590f7d529ae3fccdedf6627070021b6deccfa8ba
              started: "2026-10-22T19:40:15.447Z"
              spawned: false
              bytes: 2866274
              windows:
                - start: "2026-10-22T12:41:48-07:00"
                  end: "2026-10-22T14:20:34-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 34.2714
        cuts:
            - dispatched: "2026-10-22T14:15:05-07:00"
              finalized: "2026-10-22T14:16:38-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-22T14:23:31-07:00"
      dispatched: "2026-10-22T14:21:33-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: addressed
          note: pty_darwin_test.go:630 and :432 now call openPTY; :367 is the documented exception inside the outer openPTYRetry; atlas states the rule. No raw pty.Open remains in scribecmd otherwise.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Done-when now bans test-side or blind retries and names the ruled production bounded retry, citing delivery 4b2538ba; a Revisions entry records it.
          round: 2
        - id: BR-3
          disposition: withdrawn
          note: 'Mistaken premise: Spec lines on the step classifier say every failure was at the ptmx open and the ioctls never failed, so "measured from both" was itself an overclaim; superseded by the new finding covering every site.'
          round: 2
      findings:
        - kind: defect
          id: BR-4
          severity: Minor
          title: Comments misattribute the measured negated errno to the ioctl raw-syscall path
          detail: 'This is the 2nd finding in family doc-claim-narrower-than-measurement. The Spec measured -6 only at syscall.Open on /dev/ptmx, and TIOCPTYGNAME/GRANT/UNLK never failed. Instances: pty_darwin.go isTransientPTYAlloc comment "measured arriving from BOTH"; atlas/architecture.md:1351-1352 "measured from both ... and the raw ioctl syscalls"; pty_darwin_test.go rawENXIO comment, which says the ioctls are the production spelling''s source; pty_darwin_test.go:128-131 "whichever of creack/pty''s two syscall conventions happened to lose the race"; lessons.md "a failing raw syscall can hand back a negated errno". Rule: state where the errno came from once, in the isTransientPTYAlloc comment, matching the Spec (measured at the syscall.Open ptmx open; the ioctl path is covered as a precaution only). The other sites should point to that comment instead of restating it.'
          family: doc-claim-narrower-than-measurement
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: b54e8064574d5962a25811704f3ad4998078a30a
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 5402068b-f958-4519-afba-a2fea00404ba
            repository: github.com/xianxu/pair
            issue_id: "000809"
            recipe: small-diff-review
            round: 3
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/6998f4655fd79299
            reviewed_head: 1266347a901798d554eb428c2b65801d190923fb
            slot: pair:8
            requested_at: "2026-10-22T21:30:42.471459Z"
          phase: finished
          pid: 10146
          started_at: "2026-10-22T21:30:42.591618Z"
          finished_at: "2026-10-22T21:31:20.699383Z"
          session_id: 583a2a4e-4f42-46ed-81cb-b0016dc19539
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 1266347a901798d554eb428c2b65801d190923fb
        tracker_head: 8973fb5840deb9135f2893df9571a19d09a54bec
        branch_point: 7b9440e8c92663f8ee189f54a1260e3043a78f07
        since: "2026-10-22T12:41:48-07:00"
        until: "2026-10-22T14:29:59-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-22T12:41:48-07:00"
              end: "2026-10-22T14:29:59-07:00"
              began: "2026-10-22T12:41:48-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot8/pair
        sources:
            - harness: qoder
              id: d0d6b474-f0bb-4817-a59c-ca5442029afe
              slot: pair:8
              cwd: /Users/xianxu/workspace/worktree/pair-slot8/pair
              digest: 44198cfbf4ab5e4a7a38c689d4ad07fda875416efbc83eb269d7eac903d2affe
              started: "2026-10-22T19:40:15.447Z"
              spawned: false
              bytes: 3097618
              windows:
                - start: "2026-10-22T12:41:48-07:00"
                  end: "2026-10-22T14:29:59-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 34.2714
        cuts:
            - dispatched: "2026-10-22T14:15:05-07:00"
              finalized: "2026-10-22T14:16:38-07:00"
              finalizing: false
            - dispatched: "2026-10-22T14:21:33-07:00"
              finalized: "2026-10-22T14:23:31-07:00"
              finalizing: true
      "n": 3
      timestamp: "2026-10-22T14:31:22-07:00"
      dispatched: "2026-10-22T14:30:40-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-4
          disposition: addressed
          note: 'All five sites verified: pty_darwin.go:80-92 states the measurement once (syscall.Open on /dev/ptmx, ioctls as a precaution only); the rawENXIO comment and test doc point to it; the atlas and lessons.md restate it correctly with the precaution caveat. Comment-only change, nothing behavioural to regress.'
          round: 3
      recipe: small-diff-review
      isolation: fenced
      reviewed: 1266347a901798d554eb428c2b65801d190923fb
      blocked: false
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

## Round 2 — 2026-10-22T14:23:31-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — pty_darwin_test.go:630 and :432 now call openPTY; :367 is the documented exception inside the outer openPTYRetry; atlas states the rule. No raw pty.Open remains in scribecmd otherwise.
- BR-2 — addressed — Done-when now bans test-side or blind retries and names the ruled production bounded retry, citing delivery 4b2538ba; a Revisions entry records it.
- BR-3 — withdrawn — Mistaken premise: Spec lines on the step classifier say every failure was at the ptmx open and the ioctls never failed, so "measured from both" was itself an overclaim; superseded by the new finding covering every site.

### Raised

- **BR-4** [Minor] `doc-claim-narrower-than-measurement` (kind: defect) Comments misattribute the measured negated errno to the ioctl raw-syscall path
  This is the 2nd finding in family doc-claim-narrower-than-measurement. The Spec measured -6 only at syscall.Open on /dev/ptmx, and TIOCPTYGNAME/GRANT/UNLK never failed. Instances: pty_darwin.go isTransientPTYAlloc comment "measured arriving from BOTH"; atlas/architecture.md:1351-1352 "measured from both ... and the raw ioctl syscalls"; pty_darwin_test.go rawENXIO comment, which says the ioctls are the production spelling's source; pty_darwin_test.go:128-131 "whichever of creack/pty's two syscall conventions happened to lose the race"; lessons.md "a failing raw syscall can hand back a negated errno". Rule: state where the errno came from once, in the isTransientPTYAlloc comment, matching the Spec (measured at the syscall.Open ptmx open; the ioctl path is covered as a precaution only). The other sites should point to that comment instead of restating it.

## Round 3 — 2026-10-22T14:31:22-07:00 (claude) — passed

### Disposed

- BR-4 — addressed — All five sites verified: pty_darwin.go:80-92 states the measurement once (syscall.Open on /dev/ptmx, ioctls as a precaution only); the rawENXIO comment and test doc point to it; the atlas and lessons.md restate it correctly with the precaution caveat. Comment-only change, nothing behavioural to regress.

## Open findings

(none — every finding has been disposed)
