---
gate: boundary-review
issue: 573
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 15ce5ed6-357a-4e41-8c31-1db418eb5595
            repository: github.com/xianxu/pair
            issue_id: "000573"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/7ef127e48f5f4414
            reviewed_head: 52a0ff84621e6a41f057af5a3b8d1c4d03c0d560
            slot: pair:5
            requested_at: "2026-10-15T19:57:01.859517Z"
          phase: finished
          pid: 98632
          started_at: "2026-10-15T19:57:02.10891Z"
          finished_at: "2026-10-15T19:59:40.001749Z"
          session_id: 01a14123-b8c4-76a1-b0ee-e416704ea4bc
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 52a0ff84621e6a41f057af5a3b8d1c4d03c0d560
        tracker_head: 09e2c47977b577821fb8eac6a7fce89431044b4c
        branch_point: d6da7d4b809729db8a8766bd71258ca8f3326d9a
        since: "2026-10-15T10:56:32-07:00"
        until: "2026-10-15T12:56:24-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-15T10:56:32-07:00"
              end: "2026-10-15T12:56:24-07:00"
              began: "2026-10-15T10:56:32-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot5/pair
        sources:
            - harness: codex
              id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 1f502758993f760c5dc442232077eb26ca20ab91e84477626922874f26d9856a
              started: "2026-10-15T03:06:35.036Z"
              spawned: false
              bytes: 13738297
              windows:
                - start: "2026-10-15T10:56:32-07:00"
                  end: "2026-10-15T12:56:24-07:00"
            - harness: codex
              id: 01a140b5-e539-7301-b7cf-0082310a7bc4
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 3b18b67365f6828afd7347bbda17a9e4df2576434f07234af887cd60f970f542
              started: "2026-10-15T17:57:05.01Z"
              spawned: true
              bytes: 947597
              windows:
                - start: "2026-10-15T17:57:05.01Z"
                  end: "2026-10-15T12:56:24-07:00"
            - harness: codex
              id: 01a140ba-1c00-70d0-9c5b-08da3d2bf056
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 0a78fbc8d4db263c1e4a431e451be1c198b022cec2d1ca8640de3ab950699039
              started: "2026-10-15T18:01:42.668Z"
              spawned: false
              bytes: 606915
              windows:
                - start: "2026-10-15T18:01:42.668Z"
                  end: "2026-10-15T12:56:24-07:00"
            - harness: codex
              id: 01a140bc-d433-7991-9270-798d625318dd
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: bf573990502088b7b88da236f02a9d544220fb72c54899cb21d01e95a563a43b
              started: "2026-10-15T18:04:41.148Z"
              spawned: false
              bytes: 461974
              windows:
                - start: "2026-10-15T18:04:41.148Z"
                  end: "2026-10-15T12:56:24-07:00"
            - harness: codex
              id: 01a140bd-fca4-7711-990d-cc9e591b005e
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: eb8a9d2483c94214c247a57805ce651731e481599ccf9767cbf940957863d0e5
              started: "2026-10-15T18:05:56.635Z"
              spawned: false
              bytes: 443958
              windows:
                - start: "2026-10-15T18:05:56.635Z"
                  end: "2026-10-15T12:56:24-07:00"
            - harness: codex
              id: 01a140be-6458-7a91-9764-caea54f76cbb
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              digest: 61f03c9b48eb8551cd697bd3cf2330ab6774c0c33ba24be2f29c63b71169939b
              started: "2026-10-15T18:06:22.518Z"
              spawned: false
              bytes: 384768
              windows:
                - start: "2026-10-15T18:06:22.518Z"
                  end: "2026-10-15T12:56:24-07:00"
            - harness: codex
              id: 01a140bf-dd65-7cf3-a13b-795ffd04e917
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: e7601d47b02be43a0c9468969b1751a2c8eda33a612ee217635096dc7fcb20f4
              started: "2026-10-15T18:07:58.475Z"
              spawned: true
              bytes: 499863
              windows:
                - start: "2026-10-15T18:07:58.475Z"
                  end: "2026-10-15T12:56:24-07:00"
            - harness: codex
              id: 01a140c9-1723-7412-87c2-1f5333b222f5
              cwd: /Users/xianxu/workspace/worktree/pair-slot5/pair
              parent_id: 01a13d86-9ceb-7a10-b665-c109fdb6aa6b
              digest: 034fcf83107a5311e22f8c8e118e42779edb601c63156112a14dc89a5499eafd
              started: "2026-10-15T18:18:03.165Z"
              spawned: true
              bytes: 308941
              windows:
                - start: "2026-10-15T18:18:03.165Z"
                  end: "2026-10-15T12:56:24-07:00"
      "n": 1
      timestamp: "2026-10-15T12:59:41-07:00"
      dispatched: "2026-10-15T12:56:59-07:00"
      agent: codex
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Repeated registration discards additional aliases and permits writes under their operator locks
          detail: 'cmd/internal/sessionsync/registry.go:256 returns from existing-registration handling without retaining the requested spelling; documentLockPaths at line 529 can inspect only the canonical path, current request, and stored spellings. Reproduced by registering FILE, registering a symlink ALIAS to FILE, creating ALIAS.lock, then writing FILE: the write succeeds despite the operator lock. Preserve all admitted live spellings through the registry transition model, with bounded growth, and use them consistently for lock checks and legacy handoff. ARCH-ORDER / ARCH-PURPOSE.'
          family: canonical-document-lock-authority
          round: 1
        - kind: defect
          id: BR-2
          severity: Critical
          title: Shadow writers exceed the registered reader limit and strand successful operations
          detail: 'cmd/internal/sessionsync/sync.go:417 bounds only the resulting document; line 432 can publish a larger section-preserving shadow, while registry.go:496 refuses shadows over 4 MiB. Reproduced with an approximately 2 MiB baseline section, an operator deletion elsewhere, and an approximately 2 MiB section write: Write succeeds, then Read fails with registry-limit. Independently, registry.go:287-311 imports an oversized legacy shadow without validation, returns registration success, and leaves subsequent reads failing. Validate complete prospective metadata before document mutation or registry publication; sweep every metadata writer against its reader limits and add regressions for both paths. ARCH-CONSTRAINTS / ARCH-FUNERAL.'
          family: persisted-writer-reader-envelope
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: ebaa2ee4-69d0-4665-99f2-c9449f6667e5
            repository: github.com/xianxu/pair
            issue_id: "000573"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/0b3e50c38dcb4b3a
            reviewed_head: 284821c42463ed6ba9baebcece98f2f83f087928
            slot: pair:5
            requested_at: "2026-10-15T21:20:18.897774Z"
          phase: finished
          pid: 77518
          started_at: "2026-10-15T21:20:19.247907Z"
          finished_at: "2026-10-15T21:23:50.055358Z"
          session_id: 01a1416f-f910-7f20-8c3a-35e6858bf076
          outcome: returned
      "n": 2
      timestamp: "2026-10-15T14:23:51-07:00"
      dispatched: "2026-10-15T14:20:15-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Additional spellings now pass through the bounded registry transition model and feed lock checks and legacy export. Removing existing-registration spelling admission in a disposable copy made both alias-lock and alias-export regression tests fail; restoring it made them pass.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: The import, preserved-shadow and fold-state regressions fail when their bounds are removed and pass with the fixes restored. However, registry.go:612-620 adds SubmitPaths during legacy export without accounting for the import reader's 1 MiB limit. A valid 1,048,440-byte state registered successfully; forget succeeded after exporting 1,048,829 bytes; re-registration then failed with registry-limit. Complete the original persisted-writer-reader-envelope rule across import, read refresh, write and legacy handoff, with a round-trip regression. ARCH-CONSTRAINTS / ARCH-FUNERAL / ARCH-PURPOSE.
          round: 2
      findings:
        - kind: defect
          id: BR-3
          severity: Important
          title: Symlink regression assumes the temporary directory has a noncanonical spelling
          detail: cmd/internal/sessionsync/registry_test.go:408 requires Changed=true when registering realPath after registering its symlink alias. When the temporary directory is already canonical, realPath is the stored canonical File, so Changed=false is correct. The targeted package run and an isolated rerun both fail at line 409. Create another distinct alias to test admission, and separately assert that registering the already-known canonical spelling is unchanged.
          family: filesystem-independent-test-fixtures
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 15980d7a-743f-4d5d-94e1-e7dc003f5be5
            repository: github.com/xianxu/pair
            issue_id: "000573"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/7a80633e3ddc7205
            reviewed_head: e4468d7250773c91e65f0fbc1ef41694bd4089ca
            slot: pair:5
            requested_at: "2026-10-15T21:43:44.532088Z"
          phase: finished
          pid: 88042
          started_at: "2026-10-15T21:43:44.895579Z"
          finished_at: "2026-10-15T21:47:11.70255Z"
          session_id: 01a14185-6c7a-70a1-be39-49484dd9feda
          outcome: returned
      "n": 3
      timestamp: "2026-10-15T14:47:17-07:00"
      dispatched: "2026-10-15T14:43:40-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: The shared spelling inventory feeds lock checks and legacy export; the additional-alias regressions pass in the sessionsync package run. The prior disposition stands.
          round: 3
        - id: BR-2
          disposition: not-addressed
          note: 'The handoff reservation fixes have meaningful regression coverage: removing SubmitPaths reservation makes admission, alias, rename, and write tests fail. However, sync.go:442 validates post-fold state, then changes the document at line 446; saveShadow at line 449 revalidates the old f.state at line 376. Reproduced with valid handoff state sized to 1 MiB minus 16 bytes, obsolete folds, a new delivery gap, and write --accept-gap: post-fold preflight passes, but shadow publication returns registry-limit with documentChanged=true. This remains the persisted-writer-reader-envelope family: use one prepared, validated metadata state throughout publication, and sweep every subsequent writer/validator for stale-state checks. ARCH-CONSTRAINTS / ARCH-ORDER.'
          round: 3
        - id: BR-3
          disposition: addressed
          note: registry_test.go:406-425 now resolves the canonical spelling and expects an unchanged registration, then uses a distinct leaf alias to test admission and retry. The corrected test passes with an explicitly canonical TMPDIR, and the complete sessionsync package passes.
          round: 3
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 40ad9eab-6c92-48e1-b58f-434a0238b203
            repository: github.com/xianxu/pair
            issue_id: "000573"
            recipe: milestone-review
            round: 4
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/fa3c44f2938b357b
            reviewed_head: 9a428a8590bb98e779e633abcaa63deaa3163b55
            slot: pair:5
            requested_at: "2026-10-15T22:11:30.299399Z"
          phase: finished
          pid: 18870
          started_at: "2026-10-15T22:11:30.639819Z"
          finished_at: "2026-10-15T22:14:34.344449Z"
          session_id: 01a1419e-d7d8-7580-baf5-a1820566424b
          outcome: returned
      "n": 4
      timestamp: "2026-10-15T15:14:36-07:00"
      dispatched: "2026-10-15T15:11:27-07:00"
      agent: codex
      dispose:
        - id: BR-2
          disposition: addressed
          note: Metadata admission/growth reserves complete handoff state. Current regressions pass; restoring the pre-fix sync.go makes gap acceptance fail with documentChanged=true and makes ambiguous-result refusal change metadata. Temporary mutation copy removed.
          round: 4
        - id: BR-1
          disposition: addressed
          note: Additional-alias lock and legacy-export regressions pass; admitted spellings feed the shared canonical-document authority checks.
          round: 4
        - id: BR-3
          disposition: addressed
          note: registry_test.go:406 explicitly resolves the canonical spelling and uses a distinct leaf alias for Changed=true; the session-sync suite passes.
          round: 4
      recipe: milestone-review
      isolation: fenced
      reviewed: 9a428a8590bb98e779e633abcaa63deaa3163b55
      blocked: false
---

# Gate ledger — pair#573 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-15T12:59:41-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `canonical-document-lock-authority` (kind: defect) Repeated registration discards additional aliases and permits writes under their operator locks
  cmd/internal/sessionsync/registry.go:256 returns from existing-registration handling without retaining the requested spelling; documentLockPaths at line 529 can inspect only the canonical path, current request, and stored spellings. Reproduced by registering FILE, registering a symlink ALIAS to FILE, creating ALIAS.lock, then writing FILE: the write succeeds despite the operator lock. Preserve all admitted live spellings through the registry transition model, with bounded growth, and use them consistently for lock checks and legacy handoff. ARCH-ORDER / ARCH-PURPOSE.
- **BR-2** [Critical] `persisted-writer-reader-envelope` (kind: defect) Shadow writers exceed the registered reader limit and strand successful operations
  cmd/internal/sessionsync/sync.go:417 bounds only the resulting document; line 432 can publish a larger section-preserving shadow, while registry.go:496 refuses shadows over 4 MiB. Reproduced with an approximately 2 MiB baseline section, an operator deletion elsewhere, and an approximately 2 MiB section write: Write succeeds, then Read fails with registry-limit. Independently, registry.go:287-311 imports an oversized legacy shadow without validation, returns registration success, and leaves subsequent reads failing. Validate complete prospective metadata before document mutation or registry publication; sweep every metadata writer against its reader limits and add regressions for both paths. ARCH-CONSTRAINTS / ARCH-FUNERAL.

## Round 2 — 2026-10-15T14:23:51-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Additional spellings now pass through the bounded registry transition model and feed lock checks and legacy export. Removing existing-registration spelling admission in a disposable copy made both alias-lock and alias-export regression tests fail; restoring it made them pass.
- BR-2 — not-addressed — The import, preserved-shadow and fold-state regressions fail when their bounds are removed and pass with the fixes restored. However, registry.go:612-620 adds SubmitPaths during legacy export without accounting for the import reader's 1 MiB limit. A valid 1,048,440-byte state registered successfully; forget succeeded after exporting 1,048,829 bytes; re-registration then failed with registry-limit. Complete the original persisted-writer-reader-envelope rule across import, read refresh, write and legacy handoff, with a round-trip regression. ARCH-CONSTRAINTS / ARCH-FUNERAL / ARCH-PURPOSE.

### Raised

- **BR-3** [Important] `filesystem-independent-test-fixtures` (kind: defect) Symlink regression assumes the temporary directory has a noncanonical spelling
  cmd/internal/sessionsync/registry_test.go:408 requires Changed=true when registering realPath after registering its symlink alias. When the temporary directory is already canonical, realPath is the stored canonical File, so Changed=false is correct. The targeted package run and an isolated rerun both fail at line 409. Create another distinct alias to test admission, and separately assert that registering the already-known canonical spelling is unchanged.

## Round 3 — 2026-10-15T14:47:17-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The shared spelling inventory feeds lock checks and legacy export; the additional-alias regressions pass in the sessionsync package run. The prior disposition stands.
- BR-2 — not-addressed — The handoff reservation fixes have meaningful regression coverage: removing SubmitPaths reservation makes admission, alias, rename, and write tests fail. However, sync.go:442 validates post-fold state, then changes the document at line 446; saveShadow at line 449 revalidates the old f.state at line 376. Reproduced with valid handoff state sized to 1 MiB minus 16 bytes, obsolete folds, a new delivery gap, and write --accept-gap: post-fold preflight passes, but shadow publication returns registry-limit with documentChanged=true. This remains the persisted-writer-reader-envelope family: use one prepared, validated metadata state throughout publication, and sweep every subsequent writer/validator for stale-state checks. ARCH-CONSTRAINTS / ARCH-ORDER.
- BR-3 — addressed — registry_test.go:406-425 now resolves the canonical spelling and expects an unchanged registration, then uses a distinct leaf alias to test admission and retry. The corrected test passes with an explicitly canonical TMPDIR, and the complete sessionsync package passes.

## Round 4 — 2026-10-15T15:14:36-07:00 (codex) — passed

### Disposed

- BR-2 — addressed — Metadata admission/growth reserves complete handoff state. Current regressions pass; restoring the pre-fix sync.go makes gap acceptance fail with documentChanged=true and makes ambiguous-result refusal change metadata. Temporary mutation copy removed.
- BR-1 — addressed — Additional-alias lock and legacy-export regressions pass; admitted spellings feed the shared canonical-document authority checks.
- BR-3 — addressed — registry_test.go:406 explicitly resolves the canonical spelling and uses a distinct leaf alias for Changed=true; the session-sync suite passes.

## Open findings

(none — every finding has been disposed)
