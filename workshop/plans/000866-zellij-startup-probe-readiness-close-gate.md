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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 123e0044-84d3-45cc-a374-e7fc3877b8a0
            repository: github.com/xianxu/pair
            issue_id: "000866"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/c43bff9a480c2b5f
            reviewed_head: 3c25bc2be5b79f3575e6b91ab2abf8b33abaa9ed
            slot: pair:5
            requested_at: "2026-10-24T12:05:23.802307Z"
          phase: finished
          pid: 63142
          started_at: "2026-10-24T12:05:23.999499Z"
          finished_at: "2026-10-24T12:09:52.708006Z"
          session_id: 01a16dcd-2986-7543-a798-40fe67ccfbcc
          outcome: returned
      "n": 2
      timestamp: "2026-10-24T05:09:55-07:00"
      dispatched: "2026-10-24T05:05:20-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: The actual CLI regression covers all nine supported reads across four adapter spellings, plus qualified-read and explicit-mutation controls. It passes at head and fails for five non-list read types when the old classifier is restored in a disposable copy.
          round: 2
        - id: BR-2
          disposition: addressed
          note: 'The fixture uses testpair.ShortRoot and the long-TMPDIR regression passes. Restoring os.MkdirTemp reproduces bind: invalid argument in that regression.'
          round: 2
        - id: BR-3
          disposition: addressed
          note: README.md:884-897 now documents unmarked legacy sessions, rebuilding/relaunching cooperating clients, authorized recovery and accepted coverage limits, linking the atlas. These statements match acquireNative and the documented interactive/death exclusions.
          round: 2
        - id: BR-4
          disposition: addressed
          note: PlanProbeAdmission is removed. The plan revision at line 364 supersedes its earlier row; production uses qualifyCandidates, directly exercised by the counted 8/16 tests.
          round: 2
      findings:
        - kind: defect
          id: BR-5
          severity: Critical
          title: Governing resolver concept row names an unchanged implementation file
          detail: workshop/plans/000866-zellij-startup-readiness-plan.md:255 lists cmd/internal/artifactpath/session_lock.go as modified for the new marker/namespace-lock resolvers, but that file is unchanged across the pinned range. ZellijNamespaceLease, ZellijReadyDirectory and ZellijReadyMarker are instead new in cmd/internal/artifactpath/zellij_readiness.go:12-44. Later revisions correct other implementation locations but not this row. Reconcile this row and append a specific Revisions correction. This is Critical under the supplied core-concepts cross-check rule; it is a traceability defect, not a runtime failure. ARCH-PURPOSE.
          family: core-concept-traceability
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: cd15e139-b7ee-420f-abbc-db816abf46c2
            repository: github.com/xianxu/pair
            issue_id: "000866"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/5e464e32d6e377bc
            reviewed_head: bdbf7367ba61debbe2017f5cf5656e38a7c2e829
            slot: pair:5
            requested_at: "2026-10-24T12:11:42.02409Z"
          phase: finished
          pid: 58653
          started_at: "2026-10-24T12:11:42.139833Z"
          finished_at: "2026-10-24T12:15:00.400979Z"
          session_id: 01a16dd2-ee29-7ef1-b3fd-fc3631fb06fd
          outcome: returned
      "n": 3
      timestamp: "2026-10-24T05:15:01-07:00"
      dispatched: "2026-10-24T05:11:39-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: The passing actual-CLI regression covers all nine supported native reads, four adapter spellings, positive-marker admission and explicit-mutation controls; command.go applies qualification to that class.
          round: 3
        - id: BR-2
          disposition: addressed
          note: The recovery fixture uses testpair.ShortRoot; the long inherited TMPDIR regression passes in the complete zellijquery package.
          round: 3
        - id: BR-3
          disposition: addressed
          note: README.md:884-897 documents legacy-session refusal, rebuild/relaunch guidance, authorized recovery and coverage limits, consistent with acquireNative and the interactive/death controls.
          round: 3
        - id: BR-4
          disposition: addressed
          note: PlanProbeAdmission is absent from production. The revision at plan line 364 supersedes its earlier row; counted 8/16 tests exercise the production qualifyCandidates loop.
          round: 3
        - id: BR-5
          disposition: addressed
          note: Plan lines 367-378 explicitly supersede both earlier resolver rows. Source and pinned name-status confirm zellij_readiness.go is new, manifest.go and session_handoff_os.go are modified, and session_lock.go and session_handoff.go are unchanged.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: bdbf7367ba61debbe2017f5cf5656e38a7c2e829
      blocked: false
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

## Round 2 — 2026-10-24T05:09:55-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The actual CLI regression covers all nine supported reads across four adapter spellings, plus qualified-read and explicit-mutation controls. It passes at head and fails for five non-list read types when the old classifier is restored in a disposable copy.
- BR-2 — addressed — The fixture uses testpair.ShortRoot and the long-TMPDIR regression passes. Restoring os.MkdirTemp reproduces bind: invalid argument in that regression.
- BR-3 — addressed — README.md:884-897 now documents unmarked legacy sessions, rebuilding/relaunching cooperating clients, authorized recovery and accepted coverage limits, linking the atlas. These statements match acquireNative and the documented interactive/death exclusions.
- BR-4 — addressed — PlanProbeAdmission is removed. The plan revision at line 364 supersedes its earlier row; production uses qualifyCandidates, directly exercised by the counted 8/16 tests.

### Raised

- **BR-5** [Critical] `core-concept-traceability` (kind: defect) Governing resolver concept row names an unchanged implementation file
  workshop/plans/000866-zellij-startup-readiness-plan.md:255 lists cmd/internal/artifactpath/session_lock.go as modified for the new marker/namespace-lock resolvers, but that file is unchanged across the pinned range. ZellijNamespaceLease, ZellijReadyDirectory and ZellijReadyMarker are instead new in cmd/internal/artifactpath/zellij_readiness.go:12-44. Later revisions correct other implementation locations but not this row. Reconcile this row and append a specific Revisions correction. This is Critical under the supplied core-concepts cross-check rule; it is a traceability defect, not a runtime failure. ARCH-PURPOSE.

## Round 3 — 2026-10-24T05:15:01-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — The passing actual-CLI regression covers all nine supported native reads, four adapter spellings, positive-marker admission and explicit-mutation controls; command.go applies qualification to that class.
- BR-2 — addressed — The recovery fixture uses testpair.ShortRoot; the long inherited TMPDIR regression passes in the complete zellijquery package.
- BR-3 — addressed — README.md:884-897 documents legacy-session refusal, rebuild/relaunch guidance, authorized recovery and coverage limits, consistent with acquireNative and the interactive/death controls.
- BR-4 — addressed — PlanProbeAdmission is absent from production. The revision at plan line 364 supersedes its earlier row; counted 8/16 tests exercise the production qualifyCandidates loop.
- BR-5 — addressed — Plan lines 367-378 explicitly supersede both earlier resolver rows. Source and pinned name-status confirm zellij_readiness.go is new, manifest.go and session_handoff_os.go are modified, and session_lock.go and session_handoff.go are unchanged.

## Open findings

(none — every finding has been disposed)
