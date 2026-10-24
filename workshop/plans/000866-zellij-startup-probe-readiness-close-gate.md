---
gate: boundary-review
issue: 866
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 037e5dc7-d71f-41fb-84cc-8706e978e6da
            repository: github.com/xianxu/pair
            issue_id: "000866"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/e29ceee8e8c2fbbb
            reviewed_head: 50db793d663d30bd059e6358e2e54652395f8b31
            slot: pair:5
            requested_at: "2026-10-24T11:44:01.332137Z"
          phase: finished
          pid: 78764
          started_at: "2026-10-24T11:44:01.509011Z"
          finished_at: "2026-10-24T11:50:01.495891Z"
          session_id: 01a16db9-976a-73d1-b461-2171a522f9ba
          outcome: returned
      "n": 1
      timestamp: "2026-10-24T04:50:38-07:00"
      dispatched: "2026-10-24T04:42:24-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Non-list helper reads bypass positive readiness qualification
          detail: cmd/internal/zellijquery/command.go:86-101 classifies only list-sessions and action list-* as reads. Against a private unmarked Unix endpoint, the actual CLI helper refused list-clients but invoked native commands for are-floating-panes-visible, current-tab-info and dump-layout. The migrated review client uses are-floating-panes-visible at nvim/review/restore_client.lua:166. This contradicts the governing plan's requirement that helper reads retain qualification. Enumerate the complete supported read class and add production-helper regression tests proving every read refuses unmarked endpoints while explicit mutations/recovery retain exclusion-only admission. ARCH-PURPOSE, ARCH-SECURE.
          family: native-read-admission
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: The new recovery regression fails with a long inherited temporary directory
          detail: 'cmd/internal/zellijquery/guard_controls_test.go:303 creates its socket root using os.MkdirTemp("", "q-"), which does not ensure a short pathname. The default command go test ./cmd/internal/zellijquery -run ''^TestExplicitRecoveryPreservesUnmarkedEndpointAccess$'' -count=1 reproducibly fails at line 318 with bind: invalid argument under this review environment''s TMPDIR; the same test passes with TMPDIR=/tmp. Use the existing testpair.ShortRoot helper, already used by the launcher recovery regression. ARCH-CONSTRAINTS.'
          family: unix-fixture-path-budget
          round: 1
        - kind: defect
          id: BR-3
          severity: Important
          title: README startup and rollout guidance was not updated
          detail: README.md:873-882 remains unchanged across the range despite the plan requiring README startup semantics. Automatic observations now refuse namespaces containing unmarked sessions, and protection depends on rebuilding cooperating clients. Document the legacy-session impact, supported recovery/relaunch action and coverage limits near the existing startup guidance, linking the updated atlas. ARCH-PURPOSE.
          family: user-facing-migration-docs
          round: 1
        - kind: defect
          id: BR-4
          severity: Minor
          title: PlanProbeAdmission is an unused parallel admission implementation
          detail: cmd/internal/zellijquery/readiness.go:66-77 has no production or test callers; production instead uses qualifyCandidates in readiness_os.go:182-205. The governing concept table nevertheless claims direct counted tests for PlanProbeAdmission. Remove the unused API and correct the table, or make one pure policy serve production and direct tests. ARCH-DRY, ARCH-PURE.
          family: single-admission-policy
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#866 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-24T04:50:38-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `native-read-admission` (kind: defect) Non-list helper reads bypass positive readiness qualification
  cmd/internal/zellijquery/command.go:86-101 classifies only list-sessions and action list-* as reads. Against a private unmarked Unix endpoint, the actual CLI helper refused list-clients but invoked native commands for are-floating-panes-visible, current-tab-info and dump-layout. The migrated review client uses are-floating-panes-visible at nvim/review/restore_client.lua:166. This contradicts the governing plan's requirement that helper reads retain qualification. Enumerate the complete supported read class and add production-helper regression tests proving every read refuses unmarked endpoints while explicit mutations/recovery retain exclusion-only admission. ARCH-PURPOSE, ARCH-SECURE.
- **BR-2** [Important] `unix-fixture-path-budget` (kind: defect) The new recovery regression fails with a long inherited temporary directory
  cmd/internal/zellijquery/guard_controls_test.go:303 creates its socket root using os.MkdirTemp("", "q-"), which does not ensure a short pathname. The default command go test ./cmd/internal/zellijquery -run '^TestExplicitRecoveryPreservesUnmarkedEndpointAccess$' -count=1 reproducibly fails at line 318 with bind: invalid argument under this review environment's TMPDIR; the same test passes with TMPDIR=/tmp. Use the existing testpair.ShortRoot helper, already used by the launcher recovery regression. ARCH-CONSTRAINTS.
- **BR-3** [Important] `user-facing-migration-docs` (kind: defect) README startup and rollout guidance was not updated
  README.md:873-882 remains unchanged across the range despite the plan requiring README startup semantics. Automatic observations now refuse namespaces containing unmarked sessions, and protection depends on rebuilding cooperating clients. Document the legacy-session impact, supported recovery/relaunch action and coverage limits near the existing startup guidance, linking the updated atlas. ARCH-PURPOSE.
- **BR-4** [Minor] `single-admission-policy` (kind: defect) PlanProbeAdmission is an unused parallel admission implementation
  cmd/internal/zellijquery/readiness.go:66-77 has no production or test callers; production instead uses qualifyCandidates in readiness_os.go:182-205. The governing concept table nevertheless claims direct counted tests for PlanProbeAdmission. Remove the unused API and correct the table, or make one pure policy serve production and direct tests. ARCH-DRY, ARCH-PURE.

## Open findings

- **BR-1** [Critical] `native-read-admission` (kind: defect) Non-list helper reads bypass positive readiness qualification
- **BR-2** [Important] `unix-fixture-path-budget` (kind: defect) The new recovery regression fails with a long inherited temporary directory
- **BR-3** [Important] `user-facing-migration-docs` (kind: defect) README startup and rollout guidance was not updated
- **BR-4** [Minor] `single-admission-policy` (kind: defect) PlanProbeAdmission is an unused parallel admission implementation
