---
gate: boundary-review
issue: 653
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: f5e0085b-1c46-4e2f-a169-2fd9aef817d9
            repository: github.com/xianxu/pair
            issue_id: "000653"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/3ba6e1fd9d7d5bf3
            reviewed_head: 64d5798322f7471c617b93761bcd3b958d4d1829
            slot: pair:2
            requested_at: "2026-10-17T21:55:22.515509Z"
          phase: finished
          pid: 19853
          started_at: "2026-10-17T21:55:22.614269Z"
          finished_at: "2026-10-17T21:57:21.143556Z"
          session_id: 63e47e0a-2e5b-49bf-b563-49c9859065d7
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 64d5798322f7471c617b93761bcd3b958d4d1829
        tracker_head: 3597b84a465cff01364fbd060e71a8ea4344f93e
        branch_point: 07b6a897be5e67809b3b476606040460484cccce
        since: "2026-10-17T14:37:15-07:00"
        until: "2026-10-17T14:54:54-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T14:37:15-07:00"
              end: "2026-10-17T14:54:54-07:00"
              began: "2026-10-17T14:37:15-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 4198c83bb6e14ec1bb48b3a99d3156c5b233ca9506c0bf26b5de699e1d061785
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 24129797
              windows:
                - start: "2026-10-17T14:37:15-07:00"
                  end: "2026-10-17T14:54:54-07:00"
        queue_waits:
            - attempt_id: 6c110e7db419c845c611e8699c5cbac5
              slot: pair:2
              record_digest: d2497e07908766b09710dc83927e523cc1822845a02fe9a8960201491fc02e77
              start: "2026-10-17T21:39:39.090712Z"
              end: "2026-10-17T21:39:39.153749Z"
              queue_seconds: 0.063037
              raw_seconds: 0.063037
              exact: true
      "n": 1
      timestamp: "2026-10-17T14:57:21-07:00"
      dispatched: "2026-10-17T14:55:21-07:00"
      agent: claude
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: 'Three of four new errTitleReadmit sites survive mutation: boundary-read growth, post-fold growth, same-launch rebind'
          detail: 'I disabled each site in turn in the pinned checkout and ran TestTitleProjection*. Only the onlyGrew site (title_projection.go:200) fails a test. grewFrom(p.transcript,current) at :251, grewFrom(current,after) at :265, and the rebind reclassification in owner() at :137 can each be removed with every test green. The drift test''s afterRead fires on the first small read, so it only reaches onlyGrew. Add a fixture per site: an append timed to the boundary read and to the post-fold stat; a same-launch binding to a new root that must re-admit rather than stop; and a negative test that Binding==nil still stops.'
          family: reclassified-error-path-untested
          round: 1
        - kind: defect
          id: BR-2
          severity: Minor
          title: TitleProjection comment says only a lost owner or authority mutation is terminal; I/O, read-limit, malformed-record and identity errors are too
          detail: 'title_projection.go:29-30. Other errors also stop for good: ReadAt/StatFile errors, ErrReadLimit, ErrRootTranscript, malformed complete records, failed root validation, and missing invocation identity. Reword it to match the atlas list.'
          family: doc-claim-overstates-exhaustiveness
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: For agy, a growing p.other database is still terminal in checkArtifacts during admission and tail
          detail: title_projection.go:305 requires an equal fingerprint for every non-transcript artifact, so agy database growth still stops the title for good. Outside this issue's Claude/Codex scope; note it for a follow-up.
          family: growth-treated-as-mutation
          round: 1
        - kind: defect
          id: BR-4
          severity: Minor
          title: The Done-when live check after the next restart (every bound slot shows its figure) is not recorded in the Log
          detail: The Log records a 100-admission probe on 1-pair-6, not a fleet check after a restart. Record the live verification before close.
          family: done-when-clause-unrecorded
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: e6e56125-c5aa-4c4e-9fb5-f3b8d96d11af
            repository: github.com/xianxu/pair
            issue_id: "000653"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/bd89c1d09b505b0d
            reviewed_head: 55c26dbea83dc71be68e161b7f851192778e8799
            slot: pair:2
            requested_at: "2026-10-17T22:13:03.396254Z"
          phase: finished
          pid: 94334
          started_at: "2026-10-17T22:13:03.553665Z"
          finished_at: "2026-10-17T22:15:16.324657Z"
          session_id: 442ed130-5999-4427-8535-338589eaf7ac
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 55c26dbea83dc71be68e161b7f851192778e8799
        tracker_head: 8c084d85c3d16b6aab4eca52eccfa530a5616353
        branch_point: 07b6a897be5e67809b3b476606040460484cccce
        since: "2026-10-17T14:37:15-07:00"
        until: "2026-10-17T15:12:24-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T14:37:15-07:00"
              end: "2026-10-17T15:12:24-07:00"
              began: "2026-10-17T14:37:15-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 9cf66d37ec732b45f4d0ba33f585f7c8f2d80a9b72640093debd42439a908007
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 24575868
              windows:
                - start: "2026-10-17T14:37:15-07:00"
                  end: "2026-10-17T15:12:24-07:00"
        queue_waits:
            - attempt_id: 6c110e7db419c845c611e8699c5cbac5
              slot: pair:2
              record_digest: d2497e07908766b09710dc83927e523cc1822845a02fe9a8960201491fc02e77
              start: "2026-10-17T21:39:39.090712Z"
              end: "2026-10-17T21:39:39.153749Z"
              queue_seconds: 0.063037
              raw_seconds: 0.063037
              exact: true
        cuts:
            - dispatched: "2026-10-17T14:55:21-07:00"
              finalized: "2026-10-17T14:57:21-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-17T15:15:17-07:00"
      dispatched: "2026-10-17T15:13:01-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Mutation-checked in the pinned checkout. Disabling onlyGrew fails SnapshotDrift and GrowthNeverStopsTitle; disabling either grewFrom site fails GrowthAtEachAdmissionSite. The rebind reclassification was removed and owner() matches base.
          round: 2
        - id: BR-2
          disposition: addressed
          note: TitleProjection comment (title_projection.go:27-31) now names I/O, read-limit, malformed-record and root-validation failures as terminal.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Out of scope and filed as pair#656, recorded in the Log.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: The Log says the live post-restart fleet check stays open until the next Couch restart onto a build with this fix. That Done-when clause is still unrecorded.
          round: 2
      findings:
        - kind: defect
          id: BR-5
          severity: Critical
          title: Growth during ValidateBindingProof's fallback revalidation still stops the title for good when the generation token is empty (darwin st_gen 0)
          detail: 'admit at title_projection.go:183-190 returns ValidateBindingProof''s error unclassified. With an empty generation token and a transcript grown past its proof, AdvanceTargetValidation (incremental_inventory.go:226) falls back to ValidateTargetWork (query.go:216-219), and ObserveStableArtifact returns ErrArtifactChanged on mid-read growth (incremental_inventory.go:450). Reproduced: fixture with gen "", one append before Poll, one during the first read; both polls error terminally, while gen:7 succeeds. This is the 2nd finding in this family, so fix the rule, not the site. An admission that fails with ErrArtifactChanged while the proof artifacts only grew since admission began (stat once before validation) re-admits. Classify that once and replace the per-site grewFrom/onlyGrew checks. Add an empty-generation, stale-proof regression test.'
          family: growth-treated-as-mutation
          round: 2
        - kind: defect
          id: BR-6
          severity: Minor
          title: errTitleReadmit comment still lists a same-launch rebind, a case this range removed
          detail: 'title_projection.go:57-59. 2nd finding in this family. Rule: describe the one classification point, not a hand-kept list of cases; once the growth rule is a single check, both this comment and the TitleProjection comment should just state it.'
          family: doc-claim-overstates-exhaustiveness
          round: 2
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 39b1f75b-ef67-4eec-9359-3e17d293ed80
            repository: github.com/xianxu/pair
            issue_id: "000653"
            recipe: small-diff-review
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/548dd285e861dc12
            reviewed_head: 6ca99b831757b243024f8456d204933f57203761
            slot: pair:2
            requested_at: "2026-10-17T22:21:33.662473Z"
          phase: finished
          pid: 98899
          started_at: "2026-10-17T22:21:33.769419Z"
          finished_at: "2026-10-17T22:23:36.212921Z"
          session_id: fc52f0a5-be56-45b7-924d-fb9b788a01ef
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 6ca99b831757b243024f8456d204933f57203761
        tracker_head: 8c084d85c3d16b6aab4eca52eccfa530a5616353
        branch_point: 07b6a897be5e67809b3b476606040460484cccce
        since: "2026-10-17T14:37:15-07:00"
        until: "2026-10-17T15:21:02-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T14:37:15-07:00"
              end: "2026-10-17T15:21:02-07:00"
              began: "2026-10-17T14:37:15-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 61b926477ee93c20f92d788371269cb6fd6d2e3c7c6c4687ace59862816736c8
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 25106664
              windows:
                - start: "2026-10-17T14:37:15-07:00"
                  end: "2026-10-17T15:21:02-07:00"
        queue_waits:
            - attempt_id: 6c110e7db419c845c611e8699c5cbac5
              slot: pair:2
              record_digest: d2497e07908766b09710dc83927e523cc1822845a02fe9a8960201491fc02e77
              start: "2026-10-17T21:39:39.090712Z"
              end: "2026-10-17T21:39:39.153749Z"
              queue_seconds: 0.063037
              raw_seconds: 0.063037
              exact: true
        cuts:
            - dispatched: "2026-10-17T14:55:21-07:00"
              finalized: "2026-10-17T14:57:21-07:00"
              finalizing: false
            - dispatched: "2026-10-17T15:13:01-07:00"
              finalized: "2026-10-17T15:15:17-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-17T15:23:59-07:00"
      dispatched: "2026-10-17T15:21:32-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: addressed
          note: classifyAdmission plus the proof pre-stat; GrowthNeverStopsTitle/stale-proof goes red when the pre-stat is reverted, and three tests go red when the classifier is reverted. The proofless sibling is raised as a new finding.
          round: 3
        - id: BR-6
          disposition: addressed
          note: The errTitleReadmit comment (title_projection.go:57-58) and the TitleProjection comment now point to the one rule (classifyAdmission) and no longer list cases.
          round: 3
        - id: BR-4
          disposition: withdrawn
          note: The dispute holds. The post-restart live check can only run once the fix is on a deployed build. The Log (issue line 63) records it as open, and pre-landing live evidence is recorded (2/100 terminal vs 0/100).
          round: 3
      findings:
        - kind: defect
          id: BR-7
          severity: Important
          title: Proofless admission has no pre-validation baseline, so growth during ValidateNamedContent still stops the title for good
          detail: '3rd finding in this family. The rule: every admission path stats the artifacts it is about to validate before its first read. title_projection.go:193 sets p.since=nil and the proofless branch (:202) never seeds it, so onlyGrew(nil) returns false and the error stays terminal. Reproduced with titleProviderFixture (Claude and Codex) and one append after the first read: both polls return ErrArtifactChanged. Instances: the proofless call at :202; atlas/architecture.md:377, which claims growth during validation is transient without qualification; the classifyAdmission comment saying every artifact admission had observed. Fix: run Observe+Select for the named root before validation and seed p.since from it (one extra walk, admission only). Add a proofless subtest to GrowthNeverStopsTitle.'
          family: growth-treated-as-mutation
          round: 3
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 70106634-9de5-424e-bf04-2d48508c580b
            repository: github.com/xianxu/pair
            issue_id: "000653"
            recipe: small-diff-review
            round: 4
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/f4377a7efc875f11
            reviewed_head: 810a4ef4930b21b5729f1898671e363659a3cc9c
            slot: pair:2
            requested_at: "2026-10-17T22:27:09.602705Z"
          phase: finished
          pid: 43631
          started_at: "2026-10-17T22:27:09.731222Z"
          finished_at: "2026-10-17T22:28:23.657533Z"
          session_id: ceefbcf9-8744-41b4-b25d-815c73965814
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 810a4ef4930b21b5729f1898671e363659a3cc9c
        tracker_head: 8c084d85c3d16b6aab4eca52eccfa530a5616353
        branch_point: 07b6a897be5e67809b3b476606040460484cccce
        since: "2026-10-17T14:37:15-07:00"
        until: "2026-10-17T15:26:27-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-17T14:37:15-07:00"
              end: "2026-10-17T15:26:27-07:00"
              began: "2026-10-17T14:37:15-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: a0df622c1e7c9794f555ebe4d2f18c9cee8231f60bb0b3383426e7e1c8a2bea3
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 25228773
              windows:
                - start: "2026-10-17T14:37:15-07:00"
                  end: "2026-10-17T15:26:27-07:00"
        queue_waits:
            - attempt_id: 6c110e7db419c845c611e8699c5cbac5
              slot: pair:2
              record_digest: d2497e07908766b09710dc83927e523cc1822845a02fe9a8960201491fc02e77
              start: "2026-10-17T21:39:39.090712Z"
              end: "2026-10-17T21:39:39.153749Z"
              queue_seconds: 0.063037
              raw_seconds: 0.063037
              exact: true
        cuts:
            - dispatched: "2026-10-17T14:55:21-07:00"
              finalized: "2026-10-17T14:57:21-07:00"
              finalizing: false
            - dispatched: "2026-10-17T15:13:01-07:00"
              finalized: "2026-10-17T15:15:17-07:00"
              finalizing: false
            - dispatched: "2026-10-17T15:21:32-07:00"
              finalized: "2026-10-17T15:23:59-07:00"
              finalizing: false
      "n": 4
      timestamp: "2026-10-17T15:28:25-07:00"
      dispatched: "2026-10-17T15:27:07-07:00"
      agent: claude
      dispose:
        - id: BR-7
          disposition: addressed
          note: baseline() now runs Observe+Select for proofless bindings before validation; removing that branch turns GrowthNeverStopsTitle/proofless and GrowthAtEachAdmissionSite/proofless/{post-fold,boundary} red; atlas:377 and classifyAdmission comment now accurate.
          round: 4
      recipe: small-diff-review
      isolation: fenced
      reviewed: 810a4ef4930b21b5729f1898671e363659a3cc9c
      blocked: false
disputes:
    - id: BR-4
      rationale: The clause asks for a check after the next Couch restart onto a build with this fix. That build exists only once the fix lands on main and pair:0 rebuilds, so it cannot be recorded before close. The TL (ops:0) owns the restart and the live check is assigned to this slot afterward; the Log records it as open.
      evidence: 'Pre-landing live evidence that was re-run: the 1-pair-6 probe of 100 admissions against a live transcript gave 2/100 terminal on main and 0/100 with the fix. go test ./cmd/internal/sessioninventory -run TestTitleProjection on origin/main code fails GrowthNeverStopsTitle/admit and /stale-proof; it passes on fe0d3d83.'
      after: 2
      timestamp: "2026-10-17T22:21:00Z"
      by: Xian Xu on Xian’s MacBook Pro at pair:2 (/Users/xianxu/workspace/worktree/pair-slot2/pair)
---

# Gate ledger — pair#653 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-17T14:57:21-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `reclassified-error-path-untested` (kind: defect) Three of four new errTitleReadmit sites survive mutation: boundary-read growth, post-fold growth, same-launch rebind
  I disabled each site in turn in the pinned checkout and ran TestTitleProjection*. Only the onlyGrew site (title_projection.go:200) fails a test. grewFrom(p.transcript,current) at :251, grewFrom(current,after) at :265, and the rebind reclassification in owner() at :137 can each be removed with every test green. The drift test's afterRead fires on the first small read, so it only reaches onlyGrew. Add a fixture per site: an append timed to the boundary read and to the post-fold stat; a same-launch binding to a new root that must re-admit rather than stop; and a negative test that Binding==nil still stops.
- **BR-2** [Minor] `doc-claim-overstates-exhaustiveness` (kind: defect) TitleProjection comment says only a lost owner or authority mutation is terminal; I/O, read-limit, malformed-record and identity errors are too
  title_projection.go:29-30. Other errors also stop for good: ReadAt/StatFile errors, ErrReadLimit, ErrRootTranscript, malformed complete records, failed root validation, and missing invocation identity. Reword it to match the atlas list.
- **BR-3** [Minor] `growth-treated-as-mutation` (kind: defect) For agy, a growing p.other database is still terminal in checkArtifacts during admission and tail
  title_projection.go:305 requires an equal fingerprint for every non-transcript artifact, so agy database growth still stops the title for good. Outside this issue's Claude/Codex scope; note it for a follow-up.
- **BR-4** [Minor] `done-when-clause-unrecorded` (kind: defect) The Done-when live check after the next restart (every bound slot shows its figure) is not recorded in the Log
  The Log records a 100-admission probe on 1-pair-6, not a fleet check after a restart. Record the live verification before close.

## Round 2 — 2026-10-17T15:15:17-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Mutation-checked in the pinned checkout. Disabling onlyGrew fails SnapshotDrift and GrowthNeverStopsTitle; disabling either grewFrom site fails GrowthAtEachAdmissionSite. The rebind reclassification was removed and owner() matches base.
- BR-2 — addressed — TitleProjection comment (title_projection.go:27-31) now names I/O, read-limit, malformed-record and root-validation failures as terminal.
- BR-3 — addressed — Out of scope and filed as pair#656, recorded in the Log.
- BR-4 — not-addressed — The Log says the live post-restart fleet check stays open until the next Couch restart onto a build with this fix. That Done-when clause is still unrecorded.

### Raised

- **BR-5** [Critical] `growth-treated-as-mutation` (kind: defect) Growth during ValidateBindingProof's fallback revalidation still stops the title for good when the generation token is empty (darwin st_gen 0)
  admit at title_projection.go:183-190 returns ValidateBindingProof's error unclassified. With an empty generation token and a transcript grown past its proof, AdvanceTargetValidation (incremental_inventory.go:226) falls back to ValidateTargetWork (query.go:216-219), and ObserveStableArtifact returns ErrArtifactChanged on mid-read growth (incremental_inventory.go:450). Reproduced: fixture with gen "", one append before Poll, one during the first read; both polls error terminally, while gen:7 succeeds. This is the 2nd finding in this family, so fix the rule, not the site. An admission that fails with ErrArtifactChanged while the proof artifacts only grew since admission began (stat once before validation) re-admits. Classify that once and replace the per-site grewFrom/onlyGrew checks. Add an empty-generation, stale-proof regression test.
- **BR-6** [Minor] `doc-claim-overstates-exhaustiveness` (kind: defect) errTitleReadmit comment still lists a same-launch rebind, a case this range removed
  title_projection.go:57-59. 2nd finding in this family. Rule: describe the one classification point, not a hand-kept list of cases; once the growth rule is a single check, both this comment and the TitleProjection comment should just state it.

## Round 3 — 2026-10-17T15:23:59-07:00 (claude) — BLOCKED

### Disposed

- BR-5 — addressed — classifyAdmission plus the proof pre-stat; GrowthNeverStopsTitle/stale-proof goes red when the pre-stat is reverted, and three tests go red when the classifier is reverted. The proofless sibling is raised as a new finding.
- BR-6 — addressed — The errTitleReadmit comment (title_projection.go:57-58) and the TitleProjection comment now point to the one rule (classifyAdmission) and no longer list cases.
- BR-4 — withdrawn — The dispute holds. The post-restart live check can only run once the fix is on a deployed build. The Log (issue line 63) records it as open, and pre-landing live evidence is recorded (2/100 terminal vs 0/100).

### Raised

- **BR-7** [Important] `growth-treated-as-mutation` (kind: defect) Proofless admission has no pre-validation baseline, so growth during ValidateNamedContent still stops the title for good
  3rd finding in this family. The rule: every admission path stats the artifacts it is about to validate before its first read. title_projection.go:193 sets p.since=nil and the proofless branch (:202) never seeds it, so onlyGrew(nil) returns false and the error stays terminal. Reproduced with titleProviderFixture (Claude and Codex) and one append after the first read: both polls return ErrArtifactChanged. Instances: the proofless call at :202; atlas/architecture.md:377, which claims growth during validation is transient without qualification; the classifyAdmission comment saying every artifact admission had observed. Fix: run Observe+Select for the named root before validation and seed p.since from it (one extra walk, admission only). Add a proofless subtest to GrowthNeverStopsTitle.

## Round 4 — 2026-10-17T15:28:25-07:00 (claude) — passed

### Disposed

- BR-7 — addressed — baseline() now runs Observe+Select for proofless bindings before validation; removing that branch turns GrowthNeverStopsTitle/proofless and GrowthAtEachAdmissionSite/proofless/{post-fold,boundary} red; atlas:377 and classifyAdmission comment now accurate.

## Disputes

Findings the implementer declined (`sdlc finding dispute`), each answered by the
next review round.

- **BR-4** — after round 2 (2026-10-17T22:21:00Z, Xian Xu on Xian’s MacBook Pro at pair:2 (/Users/xianxu/workspace/worktree/pair-slot2/pair))
  rationale: The clause asks for a check after the next Couch restart onto a build with this fix. That build exists only once the fix lands on main and pair:0 rebuilds, so it cannot be recorded before close. The TL (ops:0) owns the restart and the live check is assigned to this slot afterward; the Log records it as open.
  evidence: Pre-landing live evidence that was re-run: the 1-pair-6 probe of 100 admissions against a live transcript gave 2/100 terminal on main and 0/100 with the fix. go test ./cmd/internal/sessioninventory -run TestTitleProjection on origin/main code fails GrowthNeverStopsTitle/admit and /stale-proof; it passes on fe0d3d83.

## Open findings

(none — every finding has been disposed)
