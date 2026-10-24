---
gate: boundary-review
issue: 841
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 9bdd736f-4f6c-4aa5-8b9e-03626fafa986
            repository: github.com/xianxu/pair
            issue_id: "000841"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/66b08b03d8458ede
            reviewed_head: 1f9fed7b419af84537105f82a99e9b764bf13776
            slot: pair:9
            requested_at: "2026-10-24T00:04:35.505926Z"
          phase: finished
          pid: 53162
          started_at: "2026-10-24T00:04:35.684297Z"
          finished_at: "2026-10-24T00:07:10.905189Z"
          session_id: 01a16b39-3f05-7ce1-ac89-41331f3c43f1
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 1f9fed7b419af84537105f82a99e9b764bf13776
        tracker_head: 10cf99e91953b7faad47548a391d03319a4f24f5
        branch_point: 1ee96e9872ef195450d5c53f096f530ca7e95ee3
        since: "2026-10-23T16:55:31-07:00"
        until: "2026-10-23T17:03:41-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T16:55:31-07:00"
              end: "2026-10-23T17:03:41-07:00"
              began: "2026-10-23T16:55:31-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a16862-faa4-76e2-bce3-5bc33019d381
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 77eedd0a48c77eee515505789e505e30f96635c4b0e62cbc93bc318f754439fe
              started: "2026-10-23T10:51:20.121Z"
              spawned: false
              bytes: 18234857
              windows:
                - start: "2026-10-23T16:55:31-07:00"
                  end: "2026-10-23T17:03:41-07:00"
      "n": 1
      timestamp: "2026-10-23T17:07:12-07:00"
      dispatched: "2026-10-23T17:04:32-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - id: BR-1
          severity: Critical
          title: Generated notification timestamps are highlighted as slot references
          detail: 'cmd/internal/couchtty/console_attention.go:129 opts the entire composed row into matching, including the timestamp added at line 100. attention_references.go:12-20 accepts numeric repository names, so every HH:MM timestamp with minutes 10–59 receives the accent. This affects notify, prompt and question rows, with and without hover. Confirmed by running env TZ=Asia/Kolkata go test ./cmd/internal/couchtty -run ''^TestAttentionReferenceCells/prose$'' -count=1: all six assertions failed with an unexpected colored "05:36" prefix. Exclude generated timestamp metadata from reference spans, preserve legitimate numeric repository names, and use fixed timestamps covering minutes 09, 10 and 59 in regression tests. This violates the "and only those" contract (ARCH-PURPOSE).'
          family: reference-highlighting-respects-source-role
          round: 1
        - id: BR-2
          severity: Critical
          title: Path and URL exclusion suppresses separate punctuation-delimited references
          detail: cmd/internal/couchtty/attention_references.go:32-36 rejects an entire whitespace chunk before separating punctuation-delimited tokens. Consequently [pair:3](https://example.test), [ariadne#575](https://example.test), and [#841](https://example.test) receive no reference accent even though their labels are outside the URL. The same guard suppresses separate references beside slash paths, backslash paths, or @-containing text; these are all instances of this single exclusion site. Apply exclusion to the actual path/URL token while retaining adjacent references, and add render regressions for all three reference forms alongside positive path/URL exclusions. Update the plan's whitespace-chunk rejection design accordingly (ARCH-PURPOSE).
          family: reference-exclusion-respects-token-boundaries
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#841 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-23T17:07:12-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `reference-highlighting-respects-source-role` Generated notification timestamps are highlighted as slot references
  cmd/internal/couchtty/console_attention.go:129 opts the entire composed row into matching, including the timestamp added at line 100. attention_references.go:12-20 accepts numeric repository names, so every HH:MM timestamp with minutes 10–59 receives the accent. This affects notify, prompt and question rows, with and without hover. Confirmed by running env TZ=Asia/Kolkata go test ./cmd/internal/couchtty -run '^TestAttentionReferenceCells/prose$' -count=1: all six assertions failed with an unexpected colored "05:36" prefix. Exclude generated timestamp metadata from reference spans, preserve legitimate numeric repository names, and use fixed timestamps covering minutes 09, 10 and 59 in regression tests. This violates the "and only those" contract (ARCH-PURPOSE).
- **BR-2** [Critical] `reference-exclusion-respects-token-boundaries` Path and URL exclusion suppresses separate punctuation-delimited references
  cmd/internal/couchtty/attention_references.go:32-36 rejects an entire whitespace chunk before separating punctuation-delimited tokens. Consequently [pair:3](https://example.test), [ariadne#575](https://example.test), and [#841](https://example.test) receive no reference accent even though their labels are outside the URL. The same guard suppresses separate references beside slash paths, backslash paths, or @-containing text; these are all instances of this single exclusion site. Apply exclusion to the actual path/URL token while retaining adjacent references, and add render regressions for all three reference forms alongside positive path/URL exclusions. Update the plan's whitespace-chunk rejection design accordingly (ARCH-PURPOSE).

## Open findings

- **BR-1** [Critical] `reference-highlighting-respects-source-role` Generated notification timestamps are highlighted as slot references
- **BR-2** [Critical] `reference-exclusion-respects-token-boundaries` Path and URL exclusion suppresses separate punctuation-delimited references
