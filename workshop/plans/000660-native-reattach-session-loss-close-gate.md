---
gate: boundary-review
issue: 660
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: edf784e1-3b2c-4333-832c-57d34877b0f9
            repository: github.com/xianxu/pair
            issue_id: "000660"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/a083832786ba84ec
            reviewed_head: 8022d0d29a753185dacd7e090348c3339ac08f93
            slot: pair:3
            requested_at: "2026-10-18T13:39:55.761878Z"
          phase: finished
          pid: 53672
          started_at: "2026-10-18T13:39:55.961141Z"
          finished_at: "2026-10-18T13:41:55.452858Z"
          session_id: 01a14f3d-8d55-7763-a1af-26c62688b559
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 8022d0d29a753185dacd7e090348c3339ac08f93
        tracker_head: f72d8519ba6b860266c7fcda504221492bef63ba
        branch_point: 3c76da091ea324423d755942121418905dd8c625
        since: "2026-10-18T06:05:07-07:00"
        until: "2026-10-18T06:39:10-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T06:05:07-07:00"
              end: "2026-10-18T06:39:10-07:00"
              began: "2026-10-18T06:05:07-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 6023b7b3d5bacce2e9bba4aa7c1e23f6c148f5639a891b934132497dcb0484d5
              started: "2026-10-16T22:02:33.467Z"
              spawned: false
              bytes: 74203214
              windows:
                - start: "2026-10-18T06:05:07-07:00"
                  end: "2026-10-18T06:39:10-07:00"
            - harness: codex
              id: 01a14f1e-24a0-7be1-b4ea-04299e0d589c
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: e14178c3c9cbe9f134b19c936832d7a683cd1c0b4576462d8e7b99ef1c002fe7
              started: "2026-10-18T13:05:38.041Z"
              spawned: true
              bytes: 581046
              windows:
                - start: "2026-10-18T13:05:38.041Z"
                  end: "2026-10-18T06:39:10-07:00"
            - harness: codex
              id: 01a14f23-84c5-7440-8aa7-907234155948
              slot: pair:3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: 7e6d48600ff70feaeae71ac43f3bd4687e60c5521c7196726b5ef503cc307def
              started: "2026-10-18T13:11:30.331Z"
              spawned: true
              bytes: 512667
              windows:
                - start: "2026-10-18T13:11:30.331Z"
                  end: "2026-10-18T06:39:10-07:00"
        queue_waits:
            - attempt_id: 3c238d9a21bbcf478fa82ee4c080d288
              slot: pair:3
              record_digest: b50d316d5202d587717fb30071be7835f53beacd88cd67d3a6441cac16d656ec
              start: "2026-10-18T13:20:33.436642Z"
              end: "2026-10-18T13:24:50.13599Z"
              queue_seconds: 256.699348
              raw_seconds: 256.699348
              exact: true
      "n": 1
      timestamp: "2026-10-18T06:41:57-07:00"
      dispatched: "2026-10-18T06:39:52-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Important
          title: Failure reporting can discard the newly collected exit evidence
          detail: 'terminal_native_test.go:153–164 still reports the first 4096 bytes of child.raw, parent.raw, wrapper.log and receipts.txt. Consequently, a live replacement client''s timeout can omit its latest wire output, and an EXIT receipt appended after 4096 receipt bytes is discarded before scratch removal. I executed the actual Python instrumentation after 4096 padding bytes: the receipt was written, absent from the reported prefix, and present in a bounded tail. The remaining family instances are detach output at line 293, labelled output-tail but also using the prefix helper, and its unbounded failure output at line 295. ARCH-PURPOSE: sweep these diagnostic consumers to bounded latest evidence, reusing nativeChildWireTail/readNativeDiagnosticTail and a shared byte-tail helper where appropriate. Add a regression through the failure-report collection path with oversized streams ending in distinct exit markers; the existing helper-only tests do not catch this wiring gap.'
          family: bounded-latest-diagnostic-evidence
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#660 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-18T06:41:57-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `bounded-latest-diagnostic-evidence` Failure reporting can discard the newly collected exit evidence
  terminal_native_test.go:153–164 still reports the first 4096 bytes of child.raw, parent.raw, wrapper.log and receipts.txt. Consequently, a live replacement client's timeout can omit its latest wire output, and an EXIT receipt appended after 4096 receipt bytes is discarded before scratch removal. I executed the actual Python instrumentation after 4096 padding bytes: the receipt was written, absent from the reported prefix, and present in a bounded tail. The remaining family instances are detach output at line 293, labelled output-tail but also using the prefix helper, and its unbounded failure output at line 295. ARCH-PURPOSE: sweep these diagnostic consumers to bounded latest evidence, reusing nativeChildWireTail/readNativeDiagnosticTail and a shared byte-tail helper where appropriate. Add a regression through the failure-report collection path with oversized streams ending in distinct exit markers; the existing helper-only tests do not catch this wiring gap.

## Open findings

- **BR-1** [Important] `bounded-latest-diagnostic-evidence` Failure reporting can discard the newly collected exit evidence
