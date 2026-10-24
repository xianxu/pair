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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 5a1ca516-4168-4699-b5c0-7b9931065abc
            repository: github.com/xianxu/pair
            issue_id: "000841"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/23a39a5bba2b3856
            reviewed_head: 4719c824927fb0285bb4432fd380d632c7ea7621
            slot: pair:9
            requested_at: "2026-10-24T00:12:28.411525Z"
          phase: finished
          pid: 30966
          started_at: "2026-10-24T00:12:28.719874Z"
          finished_at: "2026-10-24T00:15:02.613226Z"
          session_id: 01a16b40-76e6-7d83-8d04-04b39930be71
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 4719c824927fb0285bb4432fd380d632c7ea7621
        tracker_head: 7374522f096bfc8fd1bcc2d141bd4c7d2259628a
        branch_point: 1ee96e9872ef195450d5c53f096f530ca7e95ee3
        since: "2026-10-23T16:55:31-07:00"
        until: "2026-10-23T17:11:24-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T16:55:31-07:00"
              end: "2026-10-23T17:11:24-07:00"
              began: "2026-10-23T16:55:31-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a16862-faa4-76e2-bce3-5bc33019d381
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: 0ae26600d0e05864d34c67f7bcf7f3f3d04ee5473080c4de458b91f371ca5949
              started: "2026-10-23T10:51:20.121Z"
              spawned: false
              bytes: 18725159
              windows:
                - start: "2026-10-23T16:55:31-07:00"
                  end: "2026-10-23T17:11:24-07:00"
        cuts:
            - dispatched: "2026-10-23T17:04:32-07:00"
              finalized: "2026-10-23T17:07:12-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-23T17:15:06-07:00"
      dispatched: "2026-10-23T17:12:24-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: console_attention.go:129 excludes generated timestamp bytes. Fixed-minute tests cover 09, 10 and 59, numeric repositories, all three notification kinds and both hover states. The timezone suite passes; shortening the exclusion in a disposable copy makes prose and numeric-repository regressions fail.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: 'attention_references.go:12 still consumes closing Markdown delimiters and subsequent references as part of a URL. Render regressions fail for [x](https://x)(pair:3), [x](https://x)[ariadne#575], [x](https://x)[#841], and [pair:3](https://x)[#841](https://y): the references following the URL remain plain. All four cases fail across notify, prompt and question rows, with and without hover. This remains family reference-exclusion-respects-token-boundaries (ARCH-PURPOSE). Define and enforce where an enclosing Markdown URL ends while preserving brackets internal to paths/URLs; sweep all three reference forms and adjacent links rather than patching individual examples.'
          round: 2
      findings:
        - id: BR-3
          severity: Critical
          title: Clipping can remove every reference accent from the final visible row
          detail: 'attention_references.go:59-62 requires each rendered line to be a contiguous source substring, but notes_menu.go:188 clips through rowtext.Fit, which can remove zero-width characters. With source "10:09  pair:0  first\nsecond\npair:3 \u200b plain\nend", the visible third row becomes "pair:3  plain…" and projection returns nil, leaving pair:3 plain. Supplemental render tests fail independently for pair:3, ariadne#575 and #841. The family has one projector and one layout caller in this window; it affects all bell kinds and hover states whenever clipping removes an interior zero-width character. Preserve source-to-display positions through fitting instead of relying on substring identity, and assert exact accents on the clipped row (ARCH-PURPOSE).'
          family: reference-projection-preserves-visible-spans
          round: 2
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: edf27a24-f45a-4b99-8740-6cedd93c2763
            repository: github.com/xianxu/pair
            issue_id: "000841"
            recipe: small-diff-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/91da7eb11ff49149
            reviewed_head: d30f7c517f25996e15b8bf190d2ad6327e7bd4a5
            slot: pair:9
            requested_at: "2026-10-24T00:21:32.580264Z"
          phase: finished
          pid: 90932
          started_at: "2026-10-24T00:21:32.808064Z"
          finished_at: "2026-10-24T00:23:08.724842Z"
          session_id: 01a16b48-c45c-7a63-9b1c-5e90d88e6d1d
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: d30f7c517f25996e15b8bf190d2ad6327e7bd4a5
        tracker_head: 89d2e6e55d772370119fde0f6d1dcab83680dec5
        branch_point: 1ee96e9872ef195450d5c53f096f530ca7e95ee3
        since: "2026-10-23T16:55:31-07:00"
        until: "2026-10-23T17:20:30-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-23T16:55:31-07:00"
              end: "2026-10-23T17:20:30-07:00"
              began: "2026-10-23T16:55:31-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot9/pair
        sources:
            - harness: codex
              id: 01a16862-faa4-76e2-bce3-5bc33019d381
              slot: pair:9
              cwd: /Users/xianxu/workspace/worktree/pair-slot9/pair
              digest: fe0ad555611d1fb6bca9fab76c07b793ff7dec62796f34eb766daac1ab01b70b
              started: "2026-10-23T10:51:20.121Z"
              spawned: false
              bytes: 19180116
              windows:
                - start: "2026-10-23T16:55:31-07:00"
                  end: "2026-10-23T17:20:30-07:00"
        cuts:
            - dispatched: "2026-10-23T17:04:32-07:00"
              finalized: "2026-10-23T17:07:12-07:00"
              finalizing: false
            - dispatched: "2026-10-23T17:12:24-07:00"
              finalized: "2026-10-23T17:15:06-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-23T17:23:09-07:00"
      dispatched: "2026-10-23T17:21:29-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-2
          disposition: withdrawn
          note: The issue's “TL narrows acceptance to a conservative visual aid” revision explicitly permits uncolored Markdown labels and references adjacent to paths/URLs. attention_references.go:25-44 implements that conservative whole-word rule; atlas/couch.md documents it. Bracket/link render controls pass at head and fail with the previous renderer restored in a disposable copy.
          round: 3
        - id: BR-3
          disposition: addressed
          note: 'notes_menu.go:155 matches final fitted rows, removing the source-substring projection dependency. TestAttentionReferenceClippedRow asserts exact clipped-row accents for pair:3, ariadne#575 and #841. All pass at head; restoring the previous renderer makes all three fail with empty accents.'
          round: 3
      recipe: small-diff-review
      isolation: fenced
      reviewed: d30f7c517f25996e15b8bf190d2ad6327e7bd4a5
      blocked: false
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

## Round 2 — 2026-10-23T17:15:06-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — console_attention.go:129 excludes generated timestamp bytes. Fixed-minute tests cover 09, 10 and 59, numeric repositories, all three notification kinds and both hover states. The timezone suite passes; shortening the exclusion in a disposable copy makes prose and numeric-repository regressions fail.
- BR-2 — not-addressed — attention_references.go:12 still consumes closing Markdown delimiters and subsequent references as part of a URL. Render regressions fail for [x](https://x)(pair:3), [x](https://x)[ariadne#575], [x](https://x)[#841], and [pair:3](https://x)[#841](https://y): the references following the URL remain plain. All four cases fail across notify, prompt and question rows, with and without hover. This remains family reference-exclusion-respects-token-boundaries (ARCH-PURPOSE). Define and enforce where an enclosing Markdown URL ends while preserving brackets internal to paths/URLs; sweep all three reference forms and adjacent links rather than patching individual examples.

### Raised

- **BR-3** [Critical] `reference-projection-preserves-visible-spans` Clipping can remove every reference accent from the final visible row
  attention_references.go:59-62 requires each rendered line to be a contiguous source substring, but notes_menu.go:188 clips through rowtext.Fit, which can remove zero-width characters. With source "10:09  pair:0  first\nsecond\npair:3 \u200b plain\nend", the visible third row becomes "pair:3  plain…" and projection returns nil, leaving pair:3 plain. Supplemental render tests fail independently for pair:3, ariadne#575 and #841. The family has one projector and one layout caller in this window; it affects all bell kinds and hover states whenever clipping removes an interior zero-width character. Preserve source-to-display positions through fitting instead of relying on substring identity, and assert exact accents on the clipped row (ARCH-PURPOSE).

## Round 3 — 2026-10-23T17:23:09-07:00 (codex) — passed

### Disposed

- BR-2 — withdrawn — The issue's “TL narrows acceptance to a conservative visual aid” revision explicitly permits uncolored Markdown labels and references adjacent to paths/URLs. attention_references.go:25-44 implements that conservative whole-word rule; atlas/couch.md documents it. Bracket/link render controls pass at head and fail with the previous renderer restored in a disposable copy.
- BR-3 — addressed — notes_menu.go:155 matches final fitted rows, removing the source-substring projection dependency. TestAttentionReferenceClippedRow asserts exact clipped-row accents for pair:3, ariadne#575 and #841. All pass at head; restoring the previous renderer makes all three fail with empty accents.

## Open findings

(none — every finding has been disposed)
