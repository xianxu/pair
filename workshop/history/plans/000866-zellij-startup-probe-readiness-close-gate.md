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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 00d15af6-47e8-43b7-a559-82a493327a24
            repository: github.com/xianxu/pair
            issue_id: "000866"
            recipe: fix-review
            round: 4
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/110b3f66cbea9318
            reviewed_head: 0465f3846b724d833335a1540709f2d3e4327c40
            slot: pair:5
            requested_at: "2026-10-24T13:09:37.978742Z"
          phase: finished
          pid: 25720
          started_at: "2026-10-24T13:09:38.122543Z"
          finished_at: "2026-10-24T13:11:47.508785Z"
          session_id: 01a16e07-fb57-75a0-8cac-b98165870ab6
          outcome: returned
      "n": 4
      timestamp: "2026-10-24T06:11:50-07:00"
      dispatched: "2026-10-24T06:09:10-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      recipe: fix-review
      isolation: fenced
      reviewed: 0465f3846b724d833335a1540709f2d3e4327c40
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: d4465ca7-60a6-433d-9d80-c6c8a2123a77
            repository: github.com/xianxu/pair
            issue_id: "000866"
            recipe: fix-review
            round: 5
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/154ac811aefe243f
            reviewed_head: bb896e034063c6891c3639ae7ee7ef1418eef261
            slot: pair:5
            requested_at: "2026-10-24T15:11:46.355224Z"
          phase: finished
          pid: 84044
          started_at: "2026-10-24T15:11:46.485313Z"
          finished_at: "2026-10-24T15:16:21.773702Z"
          session_id: 01a16e77-cbea-79a0-bddd-b83b217b261c
          outcome: returned
      "n": 5
      timestamp: "2026-10-24T08:16:30-07:00"
      dispatched: "2026-10-24T08:11:43-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: The complete native-read classifier remains intact; TestInternalZellijCommandQualifiesEveryNativeRead passes at pinned head.
          round: 5
        - id: BR-2
          disposition: addressed
          note: Recovery still uses testpair.ShortRoot; the long inherited TMPDIR regression passes.
          round: 5
        - id: BR-3
          disposition: addressed
          note: README.md:884-897 retains legacy-session refusal, rebuild/relaunch guidance, authorized recovery and coverage limits, consistent with the admission implementation.
          round: 5
        - id: BR-4
          disposition: addressed
          note: PlanProbeAdmission remains absent. The plan identifies the production qualifyCandidates loop; its counted qualification tests pass.
          round: 5
        - id: BR-5
          disposition: addressed
          note: Plan lines 367-378 retain the corrected resolver mapping, supported by constructors in artifactpath/zellij_readiness.go and bindings in manifest.go.
          round: 5
      findings:
        - kind: defect
          id: BR-6
          severity: Important
          title: Make selects a candidate directory rather than the exact supplied executable
          detail: 'Makefile.local:325 prefixes dirname(PAIR_TEST_BINARY), but native callers execute bare pair. The existing builder accepts any absolute executable filename, and README.md:2208 illustrates plain-pair. Running the new Make/bootstrap regression with shared/plain-pair as the candidate produces [''installed'', ''installed''] for the two targets instead of [''shared'', ''shared'']; a sibling pair could likewise select a different build. ARCH-PURPOSE: create a suite-owned pair alias to the exact candidate using the existing builder, and extend the regression to cover candidate basenames other than pair.'
          family: exact-test-candidate-ownership
          round: 5
      recipe: fix-review
      isolation: fenced
      reviewed: bb896e034063c6891c3639ae7ee7ef1418eef261
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: ef69f78a-f7e0-4d79-a720-280dd9334065
            repository: github.com/xianxu/pair
            issue_id: "000866"
            recipe: fix-review
            round: 6
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/3a3a88c83cff4ad0
            reviewed_head: 456228a42c042e58b6a068fce31a913526e93a00
            slot: pair:5
            requested_at: "2026-10-24T16:17:48.701721Z"
          phase: finished
          pid: 99461
          started_at: "2026-10-24T16:17:48.808862Z"
          finished_at: "2026-10-24T16:19:39.63765Z"
          session_id: 01a16eb4-425b-79c2-b605-d06a19c34953
          outcome: returned
      "n": 6
      timestamp: "2026-10-24T09:19:42-07:00"
      dispatched: "2026-10-24T09:17:46-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-6
          disposition: addressed
          note: Makefile.local:329 routes both targets through with-pair-test-binary.sh, which aliases the exact candidate and exports its PATH. The regression at tests/make-test-env-test.sh:79 passes; a directory-only mutation fails with ['wrong-sibling', 'wrong-sibling'].
          round: 6
      recipe: fix-review
      isolation: fenced
      reviewed: 456228a42c042e58b6a068fce31a913526e93a00
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 6fd7dee6-abe7-493d-b4b3-a18a32dd7b49
            repository: github.com/xianxu/pair
            issue_id: "000866"
            recipe: fix-review
            round: 7
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/8a0049f20efdc5c2
            reviewed_head: abd991c4b2486fc9fafad261550ad79de9d614e6
            slot: pair:5
            requested_at: "2026-10-24T18:48:19.838584Z"
          phase: finished
          pid: 171
          started_at: "2026-10-24T18:48:20.002599Z"
          finished_at: "2026-10-24T18:56:48.699083Z"
          session_id: 01a16f3e-0e84-7191-8a48-1b1eb3cb934d
          outcome: returned
      "n": 7
      timestamp: "2026-10-24T11:56:50-07:00"
      dispatched: "2026-10-24T11:48:16-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-7
          severity: Important
          title: scope changed after close — run a full `sdlc close`
          detail: cmd/internal/launcher/session_handoff_os.go:197 adds a new phase/session error wrapper, documented as new behavior in workshop/plans/000866-zellij-startup-readiness-plan.md:404. None of BR-1–BR-6 requested this runtime diagnostic; its regression fails against the base implementation, confirming an executable change. Run a full sdlc close covering the expanded scope, or remove this addition from the narrow fix bundle.
          family: post-close-scope-control
          round: 7
      recipe: fix-review
      isolation: fenced
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: d35ecbff-f66a-494a-b2d1-df3abeb3096b
            repository: github.com/xianxu/pair
            issue_id: "000866"
            recipe: milestone-review
            round: 8
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/ab6df6f7c13e4c05
            reviewed_head: f443e10748b4febc409571bd29ace9835dec3369
            slot: pair:5
            requested_at: "2026-10-24T18:58:39.592094Z"
          phase: finished
          pid: 32410
          started_at: "2026-10-24T18:58:39.715448Z"
          finished_at: "2026-10-24T19:01:54.589022Z"
          session_id: 01a16f47-8346-7ab3-9015-39cb423e050a
          outcome: returned
      "n": 8
      timestamp: "2026-10-24T12:02:00-07:00"
      dispatched: "2026-10-24T11:58:37-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: The complete native-read classifier and real CLI admission regression remain intact; the pair-go package passes.
          round: 8
        - id: BR-2
          disposition: addressed
          note: Recovery uses testpair.ShortRoot, and the query package passes its long inherited TMPDIR regression.
          round: 8
        - id: BR-3
          disposition: addressed
          note: README.md:884-897 documents legacy-session refusal, coordinated rollout, explicit recovery and accepted coverage limits consistently with the implementation.
          round: 8
        - id: BR-4
          disposition: addressed
          note: PlanProbeAdmission is absent; production uses qualifyCandidates, exercised by the passing counted-work controls.
          round: 8
        - id: BR-5
          disposition: addressed
          note: The authoritative plan revision at lines 367-378 matches the pinned file statuses and actual resolver and handoff implementations.
          round: 8
        - id: BR-6
          disposition: addressed
          note: Both Make targets use the exact-candidate alias owner. The Make/bootstrap regression passes with plain-pair beside an incompatible sibling pair.
          round: 8
        - id: BR-7
          disposition: addressed
          note: This full whole-issue close review covers the expanded diagnostic scope recorded at plan lines 400-408. TestCreatorMissingPaneReadinessNamesSessionAtBound passes at head and fails with the diagnostic wrapper removed through a temporary Go overlay.
          round: 8
      recipe: milestone-review
      isolation: fenced
      reviewed: f443e10748b4febc409571bd29ace9835dec3369
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 84022fc7-53fa-4aa7-a385-f73130744617
            repository: github.com/xianxu/pair
            issue_id: "000866"
            recipe: milestone-review
            round: 9
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/836662232fea2cb2
            reviewed_head: 2af137563f9f375555febae4129e2eada17a78f7
            slot: pair:5
            requested_at: "2026-10-25T00:58:10.5387Z"
          phase: finished
          pid: 32626
          started_at: "2026-10-25T00:58:10.672629Z"
          finished_at: "2026-10-25T01:00:57.870273Z"
          session_id: 01a17090-aa45-7a72-80c6-45c92495ce40
          outcome: returned
      "n": 9
      timestamp: "2026-10-24T18:05:35-07:00"
      dispatched: "2026-10-24T17:58:07-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      recipe: milestone-review
      isolation: fenced
      reviewed: 2af137563f9f375555febae4129e2eada17a78f7
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 18d89b6f-254a-40f9-91e1-06d042a26577
            repository: github.com/xianxu/pair
            issue_id: "000866"
            recipe: fix-review
            round: 10
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/ce282ad58ccf5185
            reviewed_head: 69dfd71b963c5935245ea414e6d69d0792c39f28
            slot: pair:5
            requested_at: "2026-10-25T04:59:26.790509Z"
          phase: finished
          pid: 61760
          started_at: "2026-10-25T04:59:26.925542Z"
          finished_at: "2026-10-25T05:00:26.98345Z"
          session_id: 01a1716d-8ccb-7273-86e3-a876cbb7d985
          outcome: returned
      "n": 10
      timestamp: "2026-10-24T22:00:28-07:00"
      dispatched: "2026-10-24T21:59:23-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      recipe: fix-review
      isolation: fenced
      reviewed: 69dfd71b963c5935245ea414e6d69d0792c39f28
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 4fa7b253-22cb-47b6-a50a-0f7bbd8cea6c
            repository: github.com/xianxu/pair
            issue_id: "000866"
            recipe: fix-review
            round: 11
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/9a41d153df3bb8fa
            reviewed_head: 691839ae94bbf2787c170d59a478cb5582a5013c
            slot: pair:5
            requested_at: "2026-10-25T06:22:56.024947Z"
          phase: finished
          pid: 35491
          started_at: "2026-10-25T06:22:56.188877Z"
          finished_at: "2026-10-25T06:24:11.770233Z"
          session_id: 01a171b9-fc5e-79e3-8c63-dad4ae7b351a
          outcome: returned
      "n": 11
      timestamp: "2026-10-24T23:24:13-07:00"
      dispatched: "2026-10-24T23:22:52-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      recipe: fix-review
      isolation: fenced
      reviewed: 691839ae94bbf2787c170d59a478cb5582a5013c
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

## Round 4 — 2026-10-24T06:11:50-07:00 (codex) — passed

## Round 5 — 2026-10-24T08:16:30-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — The complete native-read classifier remains intact; TestInternalZellijCommandQualifiesEveryNativeRead passes at pinned head.
- BR-2 — addressed — Recovery still uses testpair.ShortRoot; the long inherited TMPDIR regression passes.
- BR-3 — addressed — README.md:884-897 retains legacy-session refusal, rebuild/relaunch guidance, authorized recovery and coverage limits, consistent with the admission implementation.
- BR-4 — addressed — PlanProbeAdmission remains absent. The plan identifies the production qualifyCandidates loop; its counted qualification tests pass.
- BR-5 — addressed — Plan lines 367-378 retain the corrected resolver mapping, supported by constructors in artifactpath/zellij_readiness.go and bindings in manifest.go.

### Raised

- **BR-6** [Important] `exact-test-candidate-ownership` (kind: defect) Make selects a candidate directory rather than the exact supplied executable
  Makefile.local:325 prefixes dirname(PAIR_TEST_BINARY), but native callers execute bare pair. The existing builder accepts any absolute executable filename, and README.md:2208 illustrates plain-pair. Running the new Make/bootstrap regression with shared/plain-pair as the candidate produces ['installed', 'installed'] for the two targets instead of ['shared', 'shared']; a sibling pair could likewise select a different build. ARCH-PURPOSE: create a suite-owned pair alias to the exact candidate using the existing builder, and extend the regression to cover candidate basenames other than pair.

## Round 6 — 2026-10-24T09:19:42-07:00 (codex) — passed

### Disposed

- BR-6 — addressed — Makefile.local:329 routes both targets through with-pair-test-binary.sh, which aliases the exact candidate and exports its PATH. The regression at tests/make-test-env-test.sh:79 passes; a directory-only mutation fails with ['wrong-sibling', 'wrong-sibling'].

## Round 7 — 2026-10-24T11:56:50-07:00 (codex) — passed

### Raised

- **BR-7** [Important] `post-close-scope-control` (kind: defect) scope changed after close — run a full `sdlc close`
  cmd/internal/launcher/session_handoff_os.go:197 adds a new phase/session error wrapper, documented as new behavior in workshop/plans/000866-zellij-startup-readiness-plan.md:404. None of BR-1–BR-6 requested this runtime diagnostic; its regression fails against the base implementation, confirming an executable change. Run a full sdlc close covering the expanded scope, or remove this addition from the narrow fix bundle.

## Round 8 — 2026-10-24T12:02:00-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — The complete native-read classifier and real CLI admission regression remain intact; the pair-go package passes.
- BR-2 — addressed — Recovery uses testpair.ShortRoot, and the query package passes its long inherited TMPDIR regression.
- BR-3 — addressed — README.md:884-897 documents legacy-session refusal, coordinated rollout, explicit recovery and accepted coverage limits consistently with the implementation.
- BR-4 — addressed — PlanProbeAdmission is absent; production uses qualifyCandidates, exercised by the passing counted-work controls.
- BR-5 — addressed — The authoritative plan revision at lines 367-378 matches the pinned file statuses and actual resolver and handoff implementations.
- BR-6 — addressed — Both Make targets use the exact-candidate alias owner. The Make/bootstrap regression passes with plain-pair beside an incompatible sibling pair.
- BR-7 — addressed — This full whole-issue close review covers the expanded diagnostic scope recorded at plan lines 400-408. TestCreatorMissingPaneReadinessNamesSessionAtBound passes at head and fails with the diagnostic wrapper removed through a temporary Go overlay.

## Round 9 — 2026-10-24T18:05:35-07:00 (codex) — passed

## Round 10 — 2026-10-24T22:00:28-07:00 (codex) — passed

## Round 11 — 2026-10-24T23:24:13-07:00 (codex) — passed

## Open findings

(none — every finding has been disposed)
