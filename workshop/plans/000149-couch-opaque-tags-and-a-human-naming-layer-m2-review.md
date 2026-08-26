# Boundary Review — pair#149 (milestone M2)

| field | value |
|-------|-------|
| issue | 149 — couch: opaque tags and a human naming layer |
| repo | pair |
| issue file | workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | eb47a9bb07846d149c9dc971f3f25dfea1bd5fef..bd75ba37e4910d5b50d3a50fa2825338803d2823 |
| command | sdlc milestone-close --issue 149 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-08-26T13:31:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The durable transition model, exact process identities, restart reconciliation, and real helper boundary are thoughtfully implemented and verified. The boundary cannot ship because post-ack failures can leave an already-execed workspace writer running after the CLI discards its handle. The Core concepts PURE contract is also contradicted by filesystem-dependent unit fixtures. One duplicated handshake implementation should be consolidated before downstream work builds on the new API.

```findings
findings:
  - id: new
    severity: Critical
    family: incarnation-quiescence-before-capacity-release
    title: |
      Post-ack failures return an error while leaving the workspace writer unowned
    detail: |
      This is the 2nd finding in family `incarnation-quiescence-before-capacity-release`. After acknowledgement, registration failure at cmd/internal/couchcore/couch.go:228, promotion failure at line 234, and registry-save failure at line 250 can all return with the child still running. Operations.Invoke discards that handle on error at cmd/internal/couchcore/ops.go:89, and the CLI exits, violating the no-untracked-writer contract. Do not patch only the named registration branch: state one rule for every post-ack exit before ownership handoff—either quiesce and verify the exact incarnation, retaining occupied state when verification is unknown, or transfer the handle to a supervisor that remains responsible. Add table-driven integration tests for the complete failure-site enumeration; the current registration-failure test instead asserts that the writer survives.
  - id: new
    severity: Critical
    family: core-concept-kind-contract
    title: |
      PURE start entities are tested through mutable filesystem setup
    detail: |
      This is the 2nd finding in family `core-concept-kind-contract`. The plan classifies ThreadRecord and StartTransaction as PURE, but validThreadRecord at cmd/internal/couchcore/thread_test.go:9 calls testCouchNamespace, which creates and resolves temporary directories; every admittedStartRecord test inherits that IO. Sweep every Core concepts row marked PURE and give its direct tests literal absolute paths and no exec, network, mutable filesystem, or integration fake. Keep the separate Runner lifecycle test explicitly integration-oriented.
  - id: new
    severity: Important
    family: blocked-runner-handshake-authority
    title: |
      The blocked-start pipe protocol has two copy-pasted production authorities
    detail: |
      ARCH-DRY: ExecRunner.StartBlocked at cmd/internal/couchcore/runner.go:67 and PtyRunner.StartBlocked at cmd/internal/couchcore/ptyrunner.go:55 duplicate the complete pipe creation, helper wrapping, descriptor-close, error-join, and acknowledged-handle protocol. Extract one shared helper parameterized by the underlying child-start function so future safety changes cannot drift between console and no-console starts.
```

1. Strengths

- `AdvanceStartTransaction` and `ReconcileStart` form a small, deterministic transition core with exact nonce matching and conservative unknown handling.
- Helper identity is persisted before acknowledgement, with a test inspecting both durable state and zero target execs at [couch_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/couch_test.go:153).
- Real subprocess tests prove EOF, cancellation, timeout, and acknowledgement behavior across the actual descriptor/exec boundary.
- Exact nonce plus revision protects rollback from stale or concurrent mutations.
- Atlas coverage describes the new helper, registration oracle, and restart flow; no README update is required because `pair-launch-helper` is internal rather than user-invoked.

2. Critical findings

- Post-ack error paths can strand an unowned workspace writer. Fix the entire post-ack failure class and replace the existing test that expects the child to survive.
- PURE entities use filesystem-backed fixtures. Remove IO from their direct unit tests and audit all PURE rows consistently.

3. Important findings

- Consolidate the duplicated `StartBlocked` protocol into one shared implementation (`ARCH-DRY`).

4. Minor findings

None.

5. Test coverage notes

Verified successfully:

- `go test ./... -count=1`
- `go test -race ./cmd/internal/couchcore ./cmd/internal/launcher ./cmd/internal/ptychild -count=1`
- `go vet ./...`
- Real `couchstartrecovery` probe
- `git diff --check`

The passing suite does not mitigate the first blocker: `TestSpawnRegistrationFailureLeavesTrackedCreatingIncarnationOccupied` currently pins the unsafe surviving-child behavior. The correction needs a regression that fails unless no writer remains unowned.

6. Architectural notes for upcoming work

- `ARCH-DRY`: flagged—the blocked runner protocol is duplicated.
- `ARCH-PURE`: flagged—the transition code is pure, but its claimed PURE tests perform mutable filesystem IO.
- `ARCH-PURPOSE`: flagged—the principal promise is that interrupted starts cannot leave an untracked writer; post-ack errors currently do.
- `ARCH-MOCK`: pass—the stateful runner/artifact fakes share production seams, and real subprocess/probe coverage exercises the OS boundary.

7. Plan revision recommendations

None. The plan already requires stopping and verifying post-fork failures and correctly classifies the state entities as PURE; implementation and tests should be brought back into agreement. The issue Log’s statement that registration failure merely “leaves the helper tuple creating and occupied” should be corrected when the lifecycle fix lands.
