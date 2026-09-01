# Boundary Review — 000161-couch-missed-codex-notifications#161 (milestone M2)

| field | value |
|-------|-------|
| issue | 161 — Couch misses Codex completion notifications |
| repo | 000161-couch-missed-codex-notifications |
| issue file | workshop/issues/000161-couch-missed-codex-notifications.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 9a8e88528ab254e0c1d4765bf0a003b692506599..8d208f188f04666437e4d8345db481665df82948 |
| command | sdlc milestone-close --issue 161 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-09-01T15:05:53-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The lifecycle reducer and OSC progress parsing are cleanly separated and the focused `wrapcmd` suite passes, but M2 cannot close yet. The normal composer submission path does not open a lifecycle generation, so subsequent native completion notifications can be silently discarded. The repository-wide test also identifies the new production file as missing from the mandatory artifact-source inventory.

### 1. Strengths

- `Reduce` is deterministic and free of IO, with proxy-loop-owned timer integration kept outside the reducer ([notification_lifecycle.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/notification_lifecycle.go:52)).
- Progress OSC parsing preserves original terminal bytes across both terminators, normalization modes, and every chunk split.
- Timer expiries are tokenized, and resumed work invalidates pending grace state.
- The Unicode Claude grammar correctly uses `\p{L}+`; focused table, fuzz, and production-path tests cover the stated General Category boundary.
- The new lifecycle architecture is documented in `atlas/architecture.md`. No README update is needed because M2 adds no user-typed command, flag, keybinding, or configuration surface.

### 2. Critical findings

- [wrap.go:1784](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/wrap.go:1784) — `ARCH-PURPOSE`: normal submissions do not reach the lifecycle reducer. `ObservationUserSubmission` is published only when `decision.bytes` is exactly `\r`, but active Codex, Muse, and Agy composers emit `\n`, while active Claude emits `\\\r` ([harness_tty.go:23](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/harness_tty.go:23)). The reducer consequently remains idle and drops native completion at [notification_lifecycle.go:111](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/notification_lifecycle.go:111). This directly undermines the missed-Codex-notification purpose and the plan’s “user submission is the hard boundary” contract. Publish submission based on the accepted plain-return decision—not its encoded output bytes—and exclude overlay confirmation explicitly. Add production-path tests for active Codex and Claude composer submission followed by native completion; each test must fail without the fix.

### 3. Important findings

- [notification_rewriter_test.go:159](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/notification_rewriter_test.go:159) — `ARCH-PURPOSE`: the pre-existing native-notification regression was changed to prepend a progress opener, masking the reachability regression rather than testing the real submission-to-completion flow. Restore coverage for native completion after actual composer submission and retain a separate progress-opened case.
- [artifactpath/manifest.go:698](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/artifactpath/manifest.go:698) — the new `notification_lifecycle.go` production source is absent from the exhaustive artifact-source inventory. `TestProductionArtifactReferencesAreExactlyClassified` fails with this exact diagnostic. Classify it appropriately, likely as a non-artifact source, and rerun the package and repository suites.
- [notification_lifecycle_test.go:10](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/notification_lifecycle_test.go:10) — `ARCH-PURE`/`ARCH-PURPOSE`: the plan claims arbitrary keyed/keyless sequence coverage and source permutations, but tests contain only selected examples. They do not cover native/marker ordering permutations, differing keyed starts, keyed terminal mismatches, abort outcome distinction, or actual submission integration. Add a table/property sweep enforcing at-most-once per generation and the documented transition contract.

### 4. Minor findings

None.

### 5. Test coverage notes

- `go test ./cmd/internal/wrapcmd -count=1`: PASS.
- `git diff --check <base> <head>`: PASS.
- `GOCACHE=/tmp/pair-review-gocache go test ./... -count=1`: FAIL due to the real artifact inventory regression above.
- Several runtime-bundle packages also could not compile because generated `assets/runtime/files` were absent from this checkout; this appears environmental and is separate from the confirmed regression.
- There were no prior findings to dispose.

### 6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass. Lifecycle policy is centralized in `Reduce`; no material duplicated reducer logic was introduced.
- `ARCH-PURE`: Pass structurally, but test breadth is insufficient. The reducer is genuinely pure; timer and notification IO remain in the proxy shell.
- `ARCH-PURPOSE`: Flagged. The primary composer submission path does not open generations, so the implementation delivers only the progress-opened subset.
- `ARCH-MOCK`: Pass for M2. OSC chunking uses the same production parser seam with state carried across calls; no new external binary/service dependency was added.
- `ARCH-CONSTRAINTS`: Pass for implemented M2 scope. One bounded channel and one proxy-owned resettable timer avoid unbounded concurrency; terminal forwarding remains non-blocking. Upcoming durable terminal events must not use the current lossy activity-channel behavior.

### 7. Plan revision recommendations

Add a `## Revisions` entry reopening M2 after boundary review: record that submission detection was incorrectly coupled to emitted key bytes, that the native-notification test masked the missing opener by injecting progress, and that the production-source inventory was omitted. Keep the lifecycle contract itself unchanged.

```findings
findings:
  - id: new
    severity: Critical
    family: submission-boundary-reachability
    title: |
      Normal composer submissions do not open a lifecycle generation
    detail: |
      Submission publication is conditioned on emitting exactly bare CR, excluding active Codex LF and active Claude backslash-CR paths. Native completion is then silently ignored while the reducer is idle; publish from the accepted submission decision and pin both composer paths with production-flow tests.
  - id: new
    severity: Important
    family: regression-test-reachability
    title: |
      Native notification regression injects an unrelated progress opener
    detail: |
      The prior direct native OSC test was changed to prepend OSC progress working, so it no longer catches missing submission-to-native reachability. Restore a real submission-to-completion regression and keep progress opening as a separate case.
  - id: new
    severity: Important
    family: production-source-inventory
    title: |
      New lifecycle source is absent from the exhaustive artifact inventory
    detail: |
      TestProductionArtifactReferencesAreExactlyClassified fails because cmd/internal/wrapcmd/notification_lifecycle.go is unclassified. Add the appropriate inventory entry and rerun the repository suite.
  - id: new
    severity: Important
    family: lifecycle-contract-coverage
    title: |
      Reducer tests do not exercise the plan's keyed and source-order contract
    detail: |
      The checked plan claims arbitrary keyed/keyless sequences and source permutations, but coverage omits ordering permutations, key mismatches, differing keyed starts, abort outcome distinction, and production submission integration.
```
