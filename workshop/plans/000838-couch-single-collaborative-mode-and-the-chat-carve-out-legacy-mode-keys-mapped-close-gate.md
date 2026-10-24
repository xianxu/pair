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
---

# Gate ledger — pair#838 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-24T10:59:09-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `operator-authority-association` (kind: defect) Chat tags can transfer to another heading without an ownership refusal
  cmd/internal/editorial/edits.go:132 checks declaration position, key and bytes, while lines 153–171 check the original heading's identity and parent; neither verifies that the declaration still belongs to that heading. With frontmatter collaboration-mode: chat followed by @@chat@@, # Questions and 1. Why?, inserting # Injected directly between the tag and # Questions succeeds through ValidateEdits, BuildEdit and Sync.Apply, which publishes the changed document. Whole-document legacy chat keeps effective routing unchanged, so the surviving-byte check cannot detect the transfer. This violates the promised tag-attachment protection (ARCH-SECURE, ARCH-PURPOSE). Compare each candidate declaration's owning heading with the mapped original owner, sweep both tags and heading directives, and add rejection tests through the public write seams.

## Open findings

- **BR-1** [Critical] `operator-authority-association` (kind: defect) Chat tags can transfer to another heading without an ownership refusal
