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
