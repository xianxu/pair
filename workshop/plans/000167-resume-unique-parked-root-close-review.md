# Boundary Review — pair#167 (whole-issue close)

| field | value |
|-------|-------|
| issue | 167 — Resume unique parked root on Couch startup |
| repo | pair |
| issue file | workshop/issues/000167-resume-unique-parked-root.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1047ed5c5c962a1890c74cdd8a99667d3c39583d..b49fbb4dad7f9f9fc1d831a6273e3c26819bd063 |
| command | sdlc close --issue 167 |
| reviewer | codex |
| timestamp | 2026-09-01T18:30:07-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The core selection and startup orchestration match the behavioral design, and focused tests pass. The boundary is not ready because the claimed restart/CLI acceptance test bypasses the production dispatch and console-attachment path, while README.md still documents the former post–Leave Couch behavior.

1. Strengths

- `SelectUniqueParkedRoot` is deterministic, exact-match, and correctly rejects ambiguity ([startup.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/startup.go:11)).
- `StartInteractive` cleanly separates selection from effects and propagates inventory and Resume errors without creating a fallback root ([startup.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/startup.go:29)).
- Actionable inventory physicalizes paths while preserving conservative per-record omission ([actionableinventory.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/actionableinventory.go:143)).
- Atlas documentation covers the new startup rule and operating bound.
- The Core concepts table accurately names entities, kinds, paths, and statuses.

2. Critical findings

None.

3. Important findings

- [run_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/run_test.go:302): The claimed restart-level CLI acceptance test calls `dispatchInteractiveStart` directly. It never traverses `RunWithRuntime`/`runTypedOperation`, the new production branch at [run.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/run.go:255), or `dispatchInitialAttach`. Reverting that branch to the old prepare/start route would leave this test green. Add a stateful production-entry test that starts from the public launch dispatch, reconstructs Couch over persisted state, and observes the resumed address/native session at the initial console attach. This is required by Plan Task 5 and the issue’s “CLI/startup integration and restart-level tests” Done-when.

- [README.md](/Users/xianxu/workspace/pair/README.md:297): The README says a later bare `couch` opens the switcher and requires Enter to resume. The diff changes that public behavior to automatically resume the sole exact eligible parked root, but README.md was not updated. Document the unique-match automatic resume and zero/ambiguous fallback behavior.

4. Minor findings

None.

5. Test coverage notes

Focused tests passed:

- `go test ./cmd/internal/couchcore -run '^(TestSelectUniqueParkedRoot|TestStartInteractive|TestActionableThreadInventoryProjectsPhysicalParkedWorkingPath)' -count=1`
- `go test ./cmd/internal/couchcmd -run '^(TestInteractiveLaunchResumesUniqueParkedRoot|TestInitialConsoleAttachDispatchesDeclaredOperation)$' -count=1`
- `git diff --check`

The pure decision covers exact, empty, ambiguous, live, wrong-scope, and wrong-path cases. Integration tests cover inventory failure and Resume refusal. The missing link is production CLI dispatch through initial home attachment.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass. Eligibility remains sourced from actionable inventory and final authority from `ResumeContext`.
- `ARCH-PURE`: Pass. Matching/cardinality logic is pure; IO remains in the Couch shell.
- `ARCH-PURPOSE`: Flagged. The implementation fulfills the runtime purpose, but the promised restart/CLI acceptance proof does not traverse the production consumer.
- `ARCH-MOCK`: Pass. Existing stateful runtime, ThreadStore, artifact, runner, Git, and path seams are reused.
- `ARCH-CONSTRAINTS`: Pass. Selection is a bounded O(n) local pass with no retry, fan-out, prompt, or fleet scan.

7. Plan revision recommendations

Append a `## Revisions` entry recording that Task 5’s acceptance boundary must traverse public interactive dispatch and initial console attachment; uncheck Task 5 steps 1–2 until that production-path test exists and passes.

```findings
findings:
  - id: new
    severity: Important
    family: production-path-acceptance
    title: |
      Restart acceptance bypasses interactive dispatch and console attachment
    detail: |
      The test calls dispatchInteractiveStart directly, so reverting the production runTypedOperation routing would not make it fail. Exercise the public launch path over reconstructed persisted state and assert the resumed identity reaches initial home attachment.
  - id: new
    severity: Important
    family: user-surface-documentation
    title: |
      README still documents the pre-change Leave Couch restart behavior
    detail: |
      README.md says bare couch reopens the switcher for manual Enter, but the implementation now automatically resumes one exact eligible parked root. Document automatic unique resume and the zero-or-ambiguous new-root fallback.
```

---

## Re-review — 2026-09-01T18:39:12-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 167 — Resume unique parked root on Couch startup |
| repo | pair |
| issue file | workshop/issues/000167-resume-unique-parked-root.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1047ed5c5c962a1890c74cdd8a99667d3c39583d..bf931c652432517b8d0aace07d281ea06be6c910 |
| command | sdlc close --issue 167 |
| reviewer | codex |
| timestamp | 2026-09-01T18:39:12-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation and production-path acceptance are sound, and BR-1 is addressed. BR-2 remains open: the revised README correctly describes automatic resume, but the immediately following paragraph still claims every TUI start allocates a new thread, contradicting resumed-root behavior.

1. Strengths

- The acceptance test traverses `runTypedOperationWithConsole`, the production interactive routing branch, and `dispatchInitialAttach`; reverting that routing would attach a new root and fail the address assertion ([run_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/run_test.go:303)).
- `SelectUniqueParkedRoot` is pure, exact, and rejects ambiguous candidates ([startup.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/startup.go:11)).
- Inventory and Resume failures propagate without creating a fallback root ([startup.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/startup.go:29)).
- Physical path projection remains in the integration shell with conservative per-record omission ([actionableinventory.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/actionableinventory.go:143)).
- Atlas documents the new surface and bounded behavior.

2. Critical findings

None.

3. Important findings

- BR-2 remains open at [README.md](/Users/xianxu/workspace/pair/README.md:304): “Every TUI start allocates a distinct opaque durable thread” contradicts lines 297–300, because a uniquely resumed startup reuses an existing thread. This is the second instance in family `user-surface-documentation`; apply the rule across the complete README startup description, not merely the originally identified sentence. Qualify allocation as applying to new-root startup.

4. Minor findings

None.

5. Test coverage notes

Passed:

- `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/artifactpath -count=1`
- `go test ./... -count=1`
- `git diff --check <base> <head>`

BR-1’s replacement test reaches production routing and initial attachment and checks the parked address and saved native identity. Pure and integration tests cover exact selection, ambiguity, inventory failure, path physicalization, and Resume refusal.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass. Eligibility and Resume authority reuse existing sources.
- `ARCH-PURE`: Pass. Selection is deterministic; IO remains in thin orchestration.
- `ARCH-PURPOSE`: Runtime behavior passes. Documentation still contains one shadow description of the old always-allocate model.
- `ARCH-MOCK`: Pass. Existing stateful seams are used; no dependency was introduced.
- `ARCH-CONSTRAINTS`: Pass. One bounded local inventory pass and O(n) selection; no retries, fleet scan, prompt, or fan-out.

7. Plan revision recommendations

Append a revision for BR-2 noting that the README sweep includes both the Leave Couch paragraph and the general TUI allocation statement, then correct the latter.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The replacement test traverses production interactive routing and initial console attachment; reverting the routing would attach a new root and fail its exact parked-address assertion.
  - id: BR-2
    disposition: not-addressed
    note: |
      README now describes automatic unique resume, but its immediately following “Every TUI start allocates” statement still contradicts resumed startup.
```

---

## Re-review — 2026-09-01T18:43:55-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 167 — Resume unique parked root on Couch startup |
| repo | pair |
| issue file | workshop/issues/000167-resume-unique-parked-root.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1047ed5c5c962a1890c74cdd8a99667d3c39583d..94252cd63a1d56a82251ab7d877f8fee7ebe89fd |
| command | sdlc close --issue 167 |
| reviewer | codex |
| timestamp | 2026-09-01T18:43:55-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range fulfills the issue’s Spec and Plan. Automatic unique parked-root resume crosses the production CLI and console-attachment boundary, conservative fallbacks and failures are covered, documentation now matches runtime behavior, and the full Go suite passes. BR-2 is addressed.

1. Strengths

- `SelectUniqueParkedRoot` is deterministic, exact, and rejects ambiguous candidates ([startup.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/startup.go:11)).
- `StartInteractive` cleanly reuses actionable inventory and existing Resume authority while preserving new-root fallback ([startup.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/startup.go:29)).
- The acceptance test traverses production interactive dispatch through initial console attachment and verifies both parked-thread identity and saved native-session identity ([run_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/run_test.go:303)).
- Physical-path normalization remains in the IO shell, with per-record failures conservatively omitted ([actionableinventory.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/actionableinventory.go:143)).
- README and atlas both document unique resume, zero/ambiguous fallback, and refusal behavior.

2. Critical findings

None.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

Passed:

- Focused selector, startup, and actionable-inventory tests.
- Production-boundary `TestInteractiveLaunchResumesUniqueParkedRoot`.
- `go test ./... -count=1`.
- Pinned-range `git diff --check`.

BR-1 was also mutation-verified in a scratch archive: disabling the production `StartInteractive` routing made `TestInteractiveLaunchResumesUniqueParkedRoot` fail because a newly allocated root was attached instead of the parked address.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass. Eligibility and Resume authority remain single-sourced.
- `ARCH-PURE`: Pass. Cardinality/identity selection is pure; filesystem, inventory, launch, and attachment remain in thin shells.
- `ARCH-PURPOSE`: Pass. The public startup path delivers unique resume through final root attachment, with complete documented fallback behavior.
- `ARCH-MOCK`: Pass. No new external dependency was introduced; integration coverage uses existing stateful seams and durable temporary storage.
- `ARCH-CONSTRAINTS`: Pass. Startup performs one bounded local inventory pass plus O(n) selection, without retries, prompts, fleet scans, or fan-out.

7. Plan revision recommendations

None. The Core concepts table matches the delivered entities and paths, and the existing BR-2 revision accurately records the completed README sweep.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      README now documents automatic unique resume, zero-or-ambiguous new-root fallback, and correctly limits durable-thread allocation to startups that create a new root.
```
