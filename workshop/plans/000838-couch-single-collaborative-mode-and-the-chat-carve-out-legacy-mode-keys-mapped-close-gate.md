---
gate: boundary-review
issue: 838
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: fd561ca3-f4a0-4a56-9304-eacc79b9dbb9
            repository: github.com/xianxu/pair
            issue_id: "000838"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/e2fb601e2d20d64a
            reviewed_head: 09f4314e925592fa18e7ce6d9051c475a96a88a4
            slot: pair:3
            requested_at: "2026-10-24T17:52:49.857987Z"
          phase: finished
          pid: 71988
          started_at: "2026-10-24T17:52:50.02454Z"
          finished_at: "2026-10-24T17:58:51.664118Z"
          session_id: 01a16f0b-3ef3-7a02-8097-32b51b25d57b
          outcome: returned
      "n": 1
      timestamp: "2026-10-24T10:59:09-07:00"
      dispatched: "2026-10-24T10:52:35-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Chat tags can transfer to another heading without an ownership refusal
          detail: 'cmd/internal/editorial/edits.go:132 checks declaration position, key and bytes, while lines 153–171 check the original heading''s identity and parent; neither verifies that the declaration still belongs to that heading. With frontmatter collaboration-mode: chat followed by @@chat@@, # Questions and 1. Why?, inserting # Injected directly between the tag and # Questions succeeds through ValidateEdits, BuildEdit and Sync.Apply, which publishes the changed document. Whole-document legacy chat keeps effective routing unchanged, so the surviving-byte check cannot detect the transfer. This violates the promised tag-attachment protection (ARCH-SECURE, ARCH-PURPOSE). Compare each candidate declaration''s owning heading with the mapped original owner, sweep both tags and heading directives, and add rejection tests through the public write seams.'
          family: operator-authority-association
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 19ca5d79-c994-4644-a577-573515e8bc16
            repository: github.com/xianxu/pair
            issue_id: "000838"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/514ed0ceff67ffd3
            reviewed_head: dfdfb47c55bd4655d9401712bec710e1b7593b6f
            slot: pair:3
            requested_at: "2026-10-24T18:05:25.277492Z"
          phase: finished
          pid: 5893
          started_at: "2026-10-24T18:05:25.426849Z"
          finished_at: "2026-10-24T18:07:49.799322Z"
          session_id: 01a16f16-c5bb-7033-b135-2bfcf897a92d
          outcome: returned
      "n": 2
      timestamp: "2026-10-24T11:08:28-07:00"
      dispatched: "2026-10-24T11:05:22-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: edits.go:143–159 binds each declaration to its mapped original heading. Regression tests cover tags and heading directives under redundant authority. Removing the owner comparison in a temporary Go overlay made ValidateEdits, BuildEdit, Write, Apply and ApplyFile regressions fail; pinned-head tests pass.
          round: 2
      findings:
        - kind: defect
          id: BR-2
          severity: Minor
          title: CLI help still advertises resolved editorial briefs
          detail: cmd/internal/couchcmd/run.go:1529–1530 still promises resolved editorial briefs and refers to every mode, although this change removes runtime brief resolution and introduces one collaborative mode. Update help to describe chat regions, protected sources and compatibility notices, matching README.md and the bundled skill (ARCH-PURPOSE).
          family: current-guidance-matches-runtime
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: dfdfb47c55bd4655d9401712bec710e1b7593b6f
      blocked: false
---

# Gate ledger — pair#838 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-24T10:59:09-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `operator-authority-association` (kind: defect) Chat tags can transfer to another heading without an ownership refusal
  cmd/internal/editorial/edits.go:132 checks declaration position, key and bytes, while lines 153–171 check the original heading's identity and parent; neither verifies that the declaration still belongs to that heading. With frontmatter collaboration-mode: chat followed by @@chat@@, # Questions and 1. Why?, inserting # Injected directly between the tag and # Questions succeeds through ValidateEdits, BuildEdit and Sync.Apply, which publishes the changed document. Whole-document legacy chat keeps effective routing unchanged, so the surviving-byte check cannot detect the transfer. This violates the promised tag-attachment protection (ARCH-SECURE, ARCH-PURPOSE). Compare each candidate declaration's owning heading with the mapped original owner, sweep both tags and heading directives, and add rejection tests through the public write seams.

## Round 2 — 2026-10-24T11:08:28-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — edits.go:143–159 binds each declaration to its mapped original heading. Regression tests cover tags and heading directives under redundant authority. Removing the owner comparison in a temporary Go overlay made ValidateEdits, BuildEdit, Write, Apply and ApplyFile regressions fail; pinned-head tests pass.

### Raised

- **BR-2** [Minor] `current-guidance-matches-runtime` (kind: defect) CLI help still advertises resolved editorial briefs
  cmd/internal/couchcmd/run.go:1529–1530 still promises resolved editorial briefs and refers to every mode, although this change removes runtime brief resolution and introduces one collaborative mode. Update help to describe chat regions, protected sources and compatibility notices, matching README.md and the bundled skill (ARCH-PURPOSE).

## Open findings

- **BR-2** [Minor] `current-guidance-matches-runtime` (kind: defect) CLI help still advertises resolved editorial briefs
