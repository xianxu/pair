---
gate: boundary-review
issue: 602
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 965278e6-3aab-4d71-822d-93ae9d3ea6df
            repository: github.com/xianxu/pair
            issue_id: "000602"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/f78f5df7c7076496
            reviewed_head: ae4ddaf5c2a262304293d0edc6b5700dd848fdbd
            slot: pair:1
            requested_at: "2026-10-16T15:52:28.281709Z"
          phase: finished
          pid: 3817
          started_at: "2026-10-16T15:52:28.405121Z"
          finished_at: "2026-10-16T15:54:18.124053Z"
          session_id: 01a1456a-2cf8-7392-a0a4-e4e69202f9ed
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: ae4ddaf5c2a262304293d0edc6b5700dd848fdbd
        tracker_head: da0cae6ececd11c66030a084b2d1309df95b81d0
        branch_point: 53c96ded8854c4da834c594a9901b08c6f8078f2
        since: "2026-10-16T08:07:27-07:00"
        until: "2026-10-16T08:51:58-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T08:07:27-07:00"
              end: "2026-10-16T08:51:58-07:00"
              began: "2026-10-16T08:07:27-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a14367-3aab-7a22-b89b-0908f54a9846
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 0fbf8a9538c4e12d9b378d05821d21e1c8b7925dbea34eaa2e7f559c434be943
              started: "2026-10-16T06:30:01.402Z"
              spawned: false
              bytes: 15846410
              windows:
                - start: "2026-10-16T08:07:27-07:00"
                  end: "2026-10-16T08:51:58-07:00"
            - harness: codex
              id: 01a14541-1b26-7f51-aaeb-a2cf23ea6506
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a14367-3aab-7a22-b89b-0908f54a9846
              digest: dbac292fbfd1b5c85864568a56de03cca06ab560e9dd59774e4313e640bb317f
              started: "2026-10-16T15:07:37.322Z"
              spawned: true
              bytes: 950518
              windows:
                - start: "2026-10-16T15:07:37.322Z"
                  end: "2026-10-16T08:51:58-07:00"
            - harness: codex
              id: 01a14548-543b-73d2-bf8d-5aca96f5a211
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a14367-3aab-7a22-b89b-0908f54a9846
              digest: 650bc27489553b6dfa3cf19fedc17b9746077f11acd37e8f0c8ea5d430cd177c
              started: "2026-10-16T15:15:30.684Z"
              spawned: true
              bytes: 638788
              windows:
                - start: "2026-10-16T15:15:30.684Z"
                  end: "2026-10-16T08:51:58-07:00"
      "n": 1
      timestamp: "2026-10-16T08:54:20-07:00"
      dispatched: "2026-10-16T08:52:26-07:00"
      agent: codex
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Centralized fallback breaks the embedded-runtime script from a checkout subdirectory
          detail: 'tests/lib/pair-test-binary.sh:26 resolves ./cmd/pair-go against the caller''s directory. tests/pair-embedded-runtime-test.sh:18 previously built an absolute package path and does not change directory. From tests/, the new helper exits 1 with "tests/cmd/pair-go: directory not found"; the previous absolute-path build exits 0. Family sweep: this is the affected migrated caller; adapt-schema explicitly changes to the root, retention/submission already required root-relative builds, and Go/native/wrapper callers establish the root. Resolve the builder''s checkout independently of caller cwd, preserving relative output-path semantics, and add a non-root fallback regression test. ARCH-PURPOSE: standalone routes must remain functional.'
          family: standalone-build-location-independence
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: README update appears missing for PAIR_TEST_BINARY and shared test builds
          detail: Makefile.local:224-231 and cmd/internal/testpair/binary.go:43 introduce PAIR_TEST_BINARY as a caller-controlled test setting, but README.md is unchanged. The complete new documentation surface is the supplied absolute executable, suite-owned plain binary, standalone package fallback, and their cleanup ownership. Document these together near README.md:1695 and clarify how plain test binaries differ from development binaries carrying defaultPairHome. atlas/index.md:40-48 already covers the architectural surface.
          family: readme-new-test-controls
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 5d552aed-d39f-41f3-8656-2faa556e8b19
            repository: github.com/xianxu/pair
            issue_id: "000602"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/8fe64bf90af9ddec
            reviewed_head: 0f07e6243f86ec8807daf6c1a7991c4189bdce62
            slot: pair:1
            requested_at: "2026-10-16T16:10:03.748738Z"
          phase: finished
          pid: 535
          started_at: "2026-10-16T16:10:03.891458Z"
          finished_at: "2026-10-16T16:12:34.505336Z"
          session_id: 01a1457a-485e-7072-8bf6-0ba91087b947
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 0f07e6243f86ec8807daf6c1a7991c4189bdce62
        tracker_head: 86846d4387b812d95571c19980dbc9078baafbf5
        branch_point: 53c96ded8854c4da834c594a9901b08c6f8078f2
        since: "2026-10-16T08:07:27-07:00"
        until: "2026-10-16T09:09:24-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T08:07:27-07:00"
              end: "2026-10-16T09:09:24-07:00"
              began: "2026-10-16T08:07:27-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: codex
              id: 01a14367-3aab-7a22-b89b-0908f54a9846
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 50e9e3e5e7c8a258fb4e3643aa446d456c5e8bda10c6f521fbbd2bda32b7611a
              started: "2026-10-16T06:30:01.402Z"
              spawned: false
              bytes: 16365955
              windows:
                - start: "2026-10-16T08:07:27-07:00"
                  end: "2026-10-16T09:09:24-07:00"
            - harness: codex
              id: 01a14541-1b26-7f51-aaeb-a2cf23ea6506
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a14367-3aab-7a22-b89b-0908f54a9846
              digest: dbac292fbfd1b5c85864568a56de03cca06ab560e9dd59774e4313e640bb317f
              started: "2026-10-16T15:07:37.322Z"
              spawned: true
              bytes: 950518
              windows:
                - start: "2026-10-16T15:07:37.322Z"
                  end: "2026-10-16T09:09:24-07:00"
            - harness: codex
              id: 01a14548-543b-73d2-bf8d-5aca96f5a211
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              parent_id: 01a14367-3aab-7a22-b89b-0908f54a9846
              digest: 650bc27489553b6dfa3cf19fedc17b9746077f11acd37e8f0c8ea5d430cd177c
              started: "2026-10-16T15:15:30.684Z"
              spawned: true
              bytes: 638788
              windows:
                - start: "2026-10-16T15:15:30.684Z"
                  end: "2026-10-16T09:09:24-07:00"
        cuts:
            - dispatched: "2026-10-16T08:52:26-07:00"
              finalized: "2026-10-16T08:54:20-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-16T09:12:36-07:00"
      dispatched: "2026-10-16T09:10:01-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: tests/lib/pair-test-binary.sh:25-30 preserves caller-relative outputs before selecting its checkout. The new regressions at tests/lib/pair-test-binary-test.sh:31-33 pass with the fix and fail against the pre-fix builder. The real embedded-runtime script also passes from tests/ with PAIR_TEST_BINARY unset.
          round: 2
        - id: BR-2
          disposition: addressed
          note: README.md:1703-1713 now documents the plain suite binary, supplied absolute executable, lazy package fallback, cleanup ownership, and development linker distinction. These statements match Makefile.local:224-231, testpair.BinaryContext/Run, and the shell wrapper.
          round: 2
      findings:
        - kind: defect
          id: BR-3
          severity: Important
          title: Source guard accepts the relative package paths used by six original offenders
          detail: 'cmd/internal/testpair/build_guard_test.go:194-196 recognizes ./cmd/pair-go and paths ending /cmd/pair-go, but misses ../../pair-go. Running the guard against all eleven original Go sources detects only five; the missed family comprises couchcmd/continuation_acceptance_test.go, couchcmd/switchagent_acceptance_test.go, hoprttcmd/hoprtt_test.go, launcher/agent_command_acceptance_test.go, launcher/couch_launch_prompt_acceptance_test.go, and launcher/selected_root_test.go, all under cmd/internal/. Additional direct, CommandContext, and table mutation controls using ../../pair-go, plus a sibling ../pair-go control from cmd/couch, all pass undetected. ARCH-PURPOSE: these per-test builds can be reintroduced without failing the guard. Normalize package targets against the relevant working directory and add regression controls covering every original recipe while retaining unrelated-build negative controls.'
          family: full-pair-target-normalization
          round: 2
      recipe: small-diff-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#602 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-16T08:54:20-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `standalone-build-location-independence` (kind: defect) Centralized fallback breaks the embedded-runtime script from a checkout subdirectory
  tests/lib/pair-test-binary.sh:26 resolves ./cmd/pair-go against the caller's directory. tests/pair-embedded-runtime-test.sh:18 previously built an absolute package path and does not change directory. From tests/, the new helper exits 1 with "tests/cmd/pair-go: directory not found"; the previous absolute-path build exits 0. Family sweep: this is the affected migrated caller; adapt-schema explicitly changes to the root, retention/submission already required root-relative builds, and Go/native/wrapper callers establish the root. Resolve the builder's checkout independently of caller cwd, preserving relative output-path semantics, and add a non-root fallback regression test. ARCH-PURPOSE: standalone routes must remain functional.
- **BR-2** [Important] `readme-new-test-controls` (kind: defect) README update appears missing for PAIR_TEST_BINARY and shared test builds
  Makefile.local:224-231 and cmd/internal/testpair/binary.go:43 introduce PAIR_TEST_BINARY as a caller-controlled test setting, but README.md is unchanged. The complete new documentation surface is the supplied absolute executable, suite-owned plain binary, standalone package fallback, and their cleanup ownership. Document these together near README.md:1695 and clarify how plain test binaries differ from development binaries carrying defaultPairHome. atlas/index.md:40-48 already covers the architectural surface.

## Round 2 — 2026-10-16T09:12:36-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — tests/lib/pair-test-binary.sh:25-30 preserves caller-relative outputs before selecting its checkout. The new regressions at tests/lib/pair-test-binary-test.sh:31-33 pass with the fix and fail against the pre-fix builder. The real embedded-runtime script also passes from tests/ with PAIR_TEST_BINARY unset.
- BR-2 — addressed — README.md:1703-1713 now documents the plain suite binary, supplied absolute executable, lazy package fallback, cleanup ownership, and development linker distinction. These statements match Makefile.local:224-231, testpair.BinaryContext/Run, and the shell wrapper.

### Raised

- **BR-3** [Important] `full-pair-target-normalization` (kind: defect) Source guard accepts the relative package paths used by six original offenders
  cmd/internal/testpair/build_guard_test.go:194-196 recognizes ./cmd/pair-go and paths ending /cmd/pair-go, but misses ../../pair-go. Running the guard against all eleven original Go sources detects only five; the missed family comprises couchcmd/continuation_acceptance_test.go, couchcmd/switchagent_acceptance_test.go, hoprttcmd/hoprtt_test.go, launcher/agent_command_acceptance_test.go, launcher/couch_launch_prompt_acceptance_test.go, and launcher/selected_root_test.go, all under cmd/internal/. Additional direct, CommandContext, and table mutation controls using ../../pair-go, plus a sibling ../pair-go control from cmd/couch, all pass undetected. ARCH-PURPOSE: these per-test builds can be reintroduced without failing the guard. Normalize package targets against the relevant working directory and add regression controls covering every original recipe while retaining unrelated-build negative controls.

## Open findings

- **BR-3** [Important] `full-pair-target-normalization` (kind: defect) Source guard accepts the relative package paths used by six original offenders
