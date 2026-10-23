---
gate: boundary-review
issue: 162
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 681d624d-c5ae-49bc-b93c-f5360a7fc527
            repository: github.com/xianxu/pair
            issue_id: "000162"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/336b7fa19b26cf4d
            reviewed_head: 8b74232d951f9480e947e88fe8c832cbed063dab
            slot: pair:2
            requested_at: "2026-10-23T14:11:30.411362Z"
          phase: finished
          pid: 90061
          started_at: "2026-10-23T14:11:30.571211Z"
          finished_at: "2026-10-23T14:16:19.49773Z"
          session_id: 01a1691a-4239-7462-b8e4-db30d16555f0
          outcome: returned
      "n": 1
      timestamp: "2026-10-23T07:17:52-07:00"
      dispatched: "2026-10-23T07:11:27-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Unaccepted faint suggestions can authorize native submission
          detail: cmd/internal/wrapcmd/peer_composer.go:186 copies faint suggestion cells into command text; composerPlaceholder at line 211 rejects suggestions only at the empty input origin. A temporary regression using an authored "/" followed by faint "model", with the cursor immediately after "/", produced text="/model" and native CR. This violates the Spec's suggestion-negative contract (ARCH-SECURE, ARCH-PURPOSE). Refuse command authority when suggestion cells contribute to the proof, preserve Active, and add mixed-style regressions across the supported profiles.
          family: authored-input-proof
          round: 1
        - kind: defect
          id: BR-2
          severity: Critical
          title: Trailing authored spaces make command echo proof impossible
          detail: cmd/internal/agentdrive/command.go:84 retains "/model " as the expected command, while cmd/internal/wrapcmd/peer_composer.go:191 projects its fully rendered echo as "/model". The strict comparison at cmd/internal/agentdrive/model.go:441 therefore waits until the deadline without emitting Enter. A temporary reducer regression reproduced this for a valid supported text-plus-enter sequence (ARCH-PURPOSE). Make authored targets and observed proof represent trailing whitespace consistently without weakening leading-whitespace, multiline or ownership checks, and add a projector-to-writer regression.
          family: command-proof-representation
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#162 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-23T07:17:52-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `authored-input-proof` (kind: defect) Unaccepted faint suggestions can authorize native submission
  cmd/internal/wrapcmd/peer_composer.go:186 copies faint suggestion cells into command text; composerPlaceholder at line 211 rejects suggestions only at the empty input origin. A temporary regression using an authored "/" followed by faint "model", with the cursor immediately after "/", produced text="/model" and native CR. This violates the Spec's suggestion-negative contract (ARCH-SECURE, ARCH-PURPOSE). Refuse command authority when suggestion cells contribute to the proof, preserve Active, and add mixed-style regressions across the supported profiles.
- **BR-2** [Critical] `command-proof-representation` (kind: defect) Trailing authored spaces make command echo proof impossible
  cmd/internal/agentdrive/command.go:84 retains "/model " as the expected command, while cmd/internal/wrapcmd/peer_composer.go:191 projects its fully rendered echo as "/model". The strict comparison at cmd/internal/agentdrive/model.go:441 therefore waits until the deadline without emitting Enter. A temporary reducer regression reproduced this for a valid supported text-plus-enter sequence (ARCH-PURPOSE). Make authored targets and observed proof represent trailing whitespace consistently without weakening leading-whitespace, multiline or ownership checks, and add a projector-to-writer regression.

## Open findings

- **BR-1** [Critical] `authored-input-proof` (kind: defect) Unaccepted faint suggestions can authorize native submission
- **BR-2** [Critical] `command-proof-representation` (kind: defect) Trailing authored spaces make command echo proof impossible
