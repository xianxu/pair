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

---

## Re-review — 2026-08-26T13:51:42-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 149 — couch: opaque tags and a human naming layer |
| repo | pair |
| issue file | workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | eb47a9bb07846d149c9dc971f3f25dfea1bd5fef..cff62523d73943922a3857e7f4dd941111e2fb27 |
| command | sdlc milestone-close --issue 149 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-08-26T13:51:42-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The durable transaction model, PURE transition boundary, shared blocked-start implementation, and restart probe are solid advances. M2 still cannot cross the boundary: acknowledgement can authorize exec and nevertheless return an error without entering post-ack cleanup; even the cleanup path proves only the Pair client exited, not that its potentially persistent zellij descendants stopped. The registration marker transition is also non-atomic despite being polled concurrently and serving as crash-recovery evidence.

```findings
dispose:
  - id: BR-12
    disposition: not-addressed
    note: |
      Acknowledge can deliver the exec byte and then return a close error, after which Spawn calls an already-resolved Cancel and returns without quiescence; moreover the post-ack test models no durable descendants, while quiesceHandle proves only the Pair client exited.
  - id: BR-13
    disposition: addressed
    note: |
      Direct ThreadRecord, StartTransaction, and Admission tests now use literal values, and TestIssue149PureCoreTestsStayAtPureBoundary fails if the forbidden IO or fake seams return.
  - id: BR-14
    disposition: not-addressed
    note: |
      Both production runners now delegate to startBlockedChild, but no test fails if either runner reintroduces its own handshake protocol; the claimed-fix contract therefore remains unmet.
findings:
  - id: new
    severity: Critical
    family: registration-evidence-atomic-publication
    title: |
      The durable registration oracle is published through a truncate-and-rewrite window
    detail: |
      ARCH-PURPOSE and ARCH-MOCK: establishReservedThreadAddress truncates the live claim before writing established state at cmd/internal/launcher/thread_claim.go:147, while Couch concurrently polls and strictly decodes that same path. A reader can observe empty or partial JSON and abort a valid start, and a crash can permanently strand malformed recovery evidence. Publish the transition atomically and add a synchronized real-filesystem test that proves readers observe only complete reserved or established records.
```

1. Strengths

- `AdvanceStartTransaction` and `ReconcileStart` remain deterministic, conservative PURE functions with nonce-addressed transitions.
- Helper identity is persisted before acknowledgement, and the test inspects both durable state and zero target executions ([couch_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/couch_test.go:153)).
- Both production runner variants now use the same `startBlockedChild` implementation ([runner.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/runner.go:66), [ptyrunner.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/ptyrunner.go:54)).
- Atlas coverage accurately describes the newly introduced transaction and recovery surfaces. No README update is required for the internal helper or developer-only probe target.
- The Core concepts entities exist at their declared paths, deleted entities remain absent, and the M2 PURE/INTEGRATION classifications otherwise match the code.

2. Critical findings

- **BR-12 remains open — `incarnation-quiescence-before-capacity-release`.** At [launchhelper.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/launchhelper.go:184), the acknowledgement byte may be written successfully before `writer.Close()` supplies the returned error. The error branch at [couch.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/couch.go:221) then calls `Cancel`, but the handle has already cleared its writer, so cancellation cannot revoke execution and `failPostAckStart` is never reached.
- The same BR-12 test only models a single fake child. Pair establishes the marker at [createflow.go](/Users/xianxu/workspace/pair/cmd/internal/launcher/createflow.go:372), then can spawn persistent helpers and zellij later at lines 531–540. `quiesceHandle` signals only the Pair PID. A real integration regression must spawn a durable descendant, inject each post-ack failure, and prove the whole workspace-writing incarnation is gone.
- **Registration evidence is not atomically published.** Replace in-place truncation with an atomic same-directory write/rename transaction, including the durability treatment appropriate for this crash-recovery marker.

3. Important findings

- **BR-14 remains open under the explicit claimed-fix rule.** The implementation satisfies `ARCH-DRY`, but no structural regression prevents either runner from restoring a parallel pipe/descriptor protocol. Add a test or executable source contract that requires both `StartBlocked` methods to delegate to the shared authority.

4. Minor findings

None.

5. Test coverage notes

Verified successfully:

- `go test ./... -count=1`
- `go test -race ./cmd/internal/couchcore ./cmd/internal/launcher ./cmd/internal/ptychild -count=1`
- `go vet ./...`
- Real `couchstartrecovery` probe
- `git diff --check`

Missing regressions:

- Acknowledgement byte delivered followed by a transport-close error.
- Post-ack failure after Pair has spawned a durable child/session.
- A synchronized reader during reserved-to-established publication, plus crash interruption.
- A mechanical shared-authority check for BR-14.

6. Architectural notes for upcoming work

- `ARCH-DRY`: pass in production code; BR-14 remains a regression-enforcement gap.
- `ARCH-PURE`: pass. M2 transition logic is isolated from IO and its direct tests use literal fixtures.
- `ARCH-PURPOSE`: flag. “No untracked workspace writer” requires whole-incarnation quiescence across every possibly delivered acknowledgement, not only the directly held client process.
- `ARCH-MOCK`: flag. `FakeRunner` cannot represent acknowledgement-delivered-plus-error or persistent descendants, and the artifact fake models atomic registration while production rewrites the marker in place.

7. Plan revision recommendations

Append a `## Revisions` entry stating:

- acknowledgement outcomes are tri-state—definitely withheld, possibly delivered, or delivered—and every latter outcome enters whole-incarnation quiescence;
- Pair client exit alone is not zellij/workspace quiescence;
- the registration oracle must be atomically published and tested against concurrent real-filesystem readers;
- the M2 integration test inventory includes transport ambiguity and persistent descendants, not only a single fake handle.

---

## Re-review — 2026-08-26T14:19:04-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 149 — couch: opaque tags and a human naming layer |
| repo | pair |
| issue file | workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | eb47a9bb07846d149c9dc971f3f25dfea1bd5fef..3709f777769005e88756a0640302d3bf830b6afc |
| command | sdlc milestone-close --issue 149 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-08-26T14:19:04-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-14 and BR-15 are addressed with load-bearing regressions. BR-12 remains open: the new real-descendant test kills the descendant through a test-only hook, while production quiescence only deletes an indexed zellij session. Consequently, the test does not prove that every post-ack workspace writer is stopped through the production boundary.

```findings
dispose:
  - id: BR-12
    disposition: not-addressed
    note: |
      The real-descendant regression kills the descendant inside FakeThreadArtifactCollisionChecker.QuiesceHook; production ScopedThreadArtifactCollisionChecker.Quiesce only deletes an indexed zellij session, so the test proves a stronger fake behavior than the shipped boundary and leaves the whole-incarnation contract unverified.
  - id: BR-14
    disposition: addressed
    note: |
      Both production runners delegate to startBlockedChild, and TestIssue149BlockedRunnersDelegateToOneHandshakeAuthority fails if either restores local pipe or acknowledged-handle construction.
  - id: BR-15
    disposition: addressed
    note: |
      Registration now uses synced same-directory temporary publication, atomic rename, and directory sync; the synchronized filesystem regression observes complete reserved state before rename and complete established state afterward.
```

1. Strengths

- Acknowledgement errors now correctly enter the possibly-delivered cleanup path at [couch.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/couch.go:221).
- All four post-ack exit sites are enumerated in integration coverage: acknowledgement ambiguity, registration, promotion, and registry persistence.
- `startBlockedChild` is now the single handshake authority for both exec and PTY runners, with a structural regression preventing drift.
- Registration publication follows the correct write–sync–rename–directory-sync protocol at [thread_claim.go](/Users/xianxu/workspace/pair/cmd/internal/launcher/thread_claim.go:158).
- The pure transaction core remains deterministic and directly tested without IO.

2. Critical findings

- **BR-12 remains open — third finding in family `incarnation-quiescence-before-capacity-release`.** The real descendant is killed explicitly by `QuiesceHook` at [couch_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/couch_test.go:417). Production instead calls `QuiesceThreadSession`, which only resolves an index entry and invokes `DeleteSession` at [thread_claim.go](/Users/xianxu/workspace/pair/cmd/internal/launcher/thread_claim.go:238). It does not own arbitrary detached descendants, and missing session binding returns success. Do not patch this test instance: state the complete production rule for every process class Pair can start before ownership transfer, implement that rule behind `ThreadArtifactController.Quiesce`, and run the descendant regression through the production controller or a live-conformant equivalent.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

Passed:

- `go test ./cmd/internal/couchcore ./cmd/internal/launcher ./cmd/internal/ptychild -count=1`
- `go test ./... -count=1`
- `go test -race ./cmd/internal/couchcore ./cmd/internal/launcher ./cmd/internal/ptychild -count=1`
- `go vet ./...`
- Real `couchstartrecovery` probe
- `git diff --check`

BR-14 and BR-15 have regressions that fail when their fixes are removed. BR-12’s regression is reachable but substitutes test-only descendant termination for production behavior.

6. Architectural notes for upcoming work

- `ARCH-DRY`: pass. Both blocked runners share one protocol authority.
- `ARCH-PURE`: pass. Transaction decisions are isolated from IO and directly unit-tested.
- `ARCH-PURPOSE`: flag. “No untracked workspace writer” is not proven by killing the held client plus a test-hook descendant.
- `ARCH-MOCK`: flag. The fake’s `QuiesceHook` models capabilities absent from production, and no live conformance check proves the real quiescence boundary stops the persistent descendant class.

Atlas coverage exists and README needs no change because the helper is internal. However, atlas’s claim that no error can leave an unowned writer currently overstates the verified behavior.

7. Plan revision recommendations

Append a `## Revisions` entry stating that whole-incarnation quiescence must enumerate every process class started before handoff, that production and test flows must share the same cleanup boundary, and that M2 requires a live or production-controller descendant test—not a fake hook that performs the cleanup itself.

---

## Re-review — 2026-08-26T14:32:42-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 149 — couch: opaque tags and a human naming layer |
| repo | pair |
| issue file | workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | eb47a9bb07846d149c9dc971f3f25dfea1bd5fef..c2062a79cdbd619dc38571e20c238a99992dd402 |
| command | sdlc milestone-close --issue 149 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-08-26T14:32:42-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The blocked-helper transaction, process-group ownership, atomic registration publication, and restart reconciliation are well implemented. BR-12 nevertheless remains open: process-group cleanup is now production-shaped, but detached zellij cleanup reports success without proving the server and panes are gone. The regression tests use recording deleters and therefore remain green if the best-effort lingering-server kill is removed or fails.

```findings
dispose:
  - id: BR-12
    disposition: not-addressed
    note: |
      Process-group cleanup now covers real stdio and PTY descendants, but detached zellij quiescence is still unverified: OSRuntime.DeleteSession swallows the lingering-server kill result and returns success without proving the exact session absent, while the tests only assert that a recording deleter was called.
```

1. Strengths

- All four post-ack exits—ambiguous acknowledgement, registration failure, promotion conflict, and registry persistence failure—share `failPostAckStart` and are exercised with real TERM-resistant descendants under both runners.
- Both runners use one blocked-start authority, and a structural regression prevents local handshake implementations from returning.
- `AdvanceStartTransaction` and `ReconcileStart` remain deterministic PURE entities with literal, IO-free direct tests.
- Registration evidence now uses write–sync–rename–directory-sync publication with a synchronized reader regression.
- Session identity is resolved through the exact `{scope, tag}` binding before deletion; unrelated or malformed evidence fails closed.

2. Critical findings

- **BR-12 remains open — third finding in family `incarnation-quiescence-before-capacity-release`.** [`OSRuntime.DeleteSession`](</Users/xianxu/workspace/pair/cmd/internal/launcher/osruntime.go:711>) runs `zellij delete-session --force`, then calls the best-effort `pkillF` at line 717. `pkillF` discards every error, and no subsequent probe verifies that the exact zellij server/session and its panes are absent. Nevertheless, [`QuiesceThreadSession`](</Users/xianxu/workspace/pair/cmd/internal/launcher/thread_claim.go:275>) treats a nil return as proof of quiescence, allowing `failPostAckStart` to reconcile durable state.

  The implementation’s own comment acknowledges that a lingering server can re-register the session. State the class rule: every cleanup boundary used as quiescence evidence must return success only after exact absence is observed. Make zellij termination observable through an injectable, stateful seam; retry/escalate as needed and fail closed if absence cannot be established.

3. Important findings

None beyond BR-12.

4. Minor findings

None.

5. Test coverage notes

The following passed:

- Focused four-site real-descendant coverage for stdio and PTY
- Focused atomic-publication, sidecar-group, and exact-session-binding tests
- `go test ./... -count=1`
- Race tests for `couchcore`, `launcher`, and `ptychild`
- `go vet ./...`
- `make test-couch-start-recovery`
- `git diff --check`

The BR-12 requirement that a test fail without the fix is unmet for the detached-session class. [`TestQuiesceThreadSessionDeletesOnlyExactIndexedPairSession`](</Users/xianxu/workspace/pair/cmd/internal/launcher/thread_claim_test.go:151>) and [`TestScopedArtifactCheckerQuiescesExactIndexedSession`](</Users/xianxu/workspace/pair/cmd/internal/couchcore/artifactcollision_test.go:61>) only assert invocation of recording deleters. Neither invokes `OSRuntime.DeleteSession`, models re-registration, observes server liveness, or fails if line 717 is deleted.

The M2 Core concepts rows exist at their declared paths and their PURE/INTEGRATION classifications match the implementation.

6. Architectural notes for upcoming work

- `ARCH-DRY`: pass. Blocked-start and post-ack cleanup orchestration each have one authority.
- `ARCH-PURE`: pass. State transitions are isolated from IO and directly tested.
- `ARCH-PURPOSE`: flag. “Whole-incarnation quiescence proven” is not fulfilled until detached zellij absence is observed, rather than inferred from attempted deletion.
- `ARCH-MOCK`: flag. The new external zellij behavior is covered only by call-recording doubles, with no stateful re-registration case or live conformance check through the production boundary.

Atlas coverage is present and README does not need an entry for the internal helper. However, `atlas/couch.md` currently overstates that whole-incarnation quiescence is proven.

7. Plan revision recommendations

Append a `## Revisions` entry stating that:

- successful zellij cleanup requires observing the exact session/server absent after deletion and lingering-server escalation;
- cleanup-command errors cannot be swallowed when their result is used as quiescence evidence;
- M2 requires a stateful or live-conformance regression where a lingering server attempts to survive or re-register;
- atlas’s “no error return can leave an unowned workspace writer” claim becomes valid only after that regression passes.

---

## Re-review — 2026-08-26T14:47:29-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 149 — couch: opaque tags and a human naming layer |
| repo | pair |
| issue file | workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | eb47a9bb07846d149c9dc971f3f25dfea1bd5fef..371e2a4115cf5f72f2d52ba33fc068039cb50f3c |
| command | sdlc milestone-close --issue 149 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-08-26T14:47:29-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The observed-absence loop, stateful re-registration fake, blocked-helper transaction, and process-group cleanup are substantial improvements, and the regression turns red when absence verification is removed. M2 still cannot close: cleanup errors can return through `Operations().start` while runtime ownership is discarded, and zellij server termination uses a bare PID without revalidating the process incarnation. The external zellij seam also lacks live conformance coverage.

```findings
dispose:
  - id: BR-12
    disposition: not-addressed
    note: |
      This is the 3rd finding in family `incarnation-quiescence-before-capacity-release`. Successful cleanup now proves absence, but query, deletion, kill, timeout, or process-group quiescence errors return from Spawn with the handle discarded by Operations.Invoke; durable occupied state does not supervise a surviving workspace writer.
findings:
  - id: new
    severity: Critical
    family: destructive-process-action-exact-identity
    title: |
      Zellij cleanup can kill a process that reused an observed server PID
    detail: |
      ARCH-PURPOSE: SessionServerPIDs returns bare integers at cmd/internal/launcher/session_quiescence.go:19 and KillProcess signals that PID later at lines 104-113 without checking a process-start identity. If the observed server exits and its PID is reused, cleanup can kill an unrelated process. Carry exact process-incarnation evidence through the seam and reauthorize immediately before signaling; add a stateful PID-reuse regression.
  - id: new
    severity: Important
    family: live-conformance-target-interface
    title: |
      The new zellij quiescence seam has no live conformance target
    detail: |
      This is the 2nd finding in family `live-conformance-target-interface`. ARCH-MOCK requires live checks for the external behavior being modeled, but the committed coverage exercises only the stateful fake and parser literals; no target verifies the actual list-sessions, process-table, delete-session, and server-termination interface. State the live-interface rule for every SessionQuiescence operation and add an ephemeral-session conformance probe with a documented target and cadence.
```

1. Strengths

- Both stdio and PTY paths share one blocked-start authority, with structural regression coverage.
- All four post-ack failure sites route through `failPostAckStart`.
- The stateful zellij fake models re-registration and requires two stable absence observations.
- Removing the post-cleanup observation loop in a scratch export makes `TestOSRuntimeDeleteSessionProvesReRegisteringServerAbsent` fail.
- Atlas covers the new helper, transaction, registration, and quiescence surfaces; no README update is needed for the internal helper.

2. Critical findings

- **BR-12 remains open.** [`failPostAckStart`](/Users/xianxu/workspace/pair/cmd/internal/couchcore/couch.go:264) returns immediately when process-group or session quiescence fails. [`Operations().start`](/Users/xianxu/workspace/pair/cmd/internal/couchcore/ops.go:89) then discards the returned handle on error. Fix the class by retaining supervisor ownership and retry responsibility until absence is proven or ownership is successfully transferred.
- **Bare-PID destructive action.** [`KillProcess`](/Users/xianxu/workspace/pair/cmd/internal/launcher/session_quiescence.go:104) must operate on an exact PID/start-token identity, not a PID observed during an earlier process-table scan.

3. Important findings

- Add live conformance for the complete `SessionQuiescence` external interface. Existing zellij tests do not exercise this production lifecycle.

4. Minor findings

None.

5. Test coverage notes

Passed:

- `make test`
- `go test ./... -count=1`
- Race tests for `couchcore`, `launcher`, and `ptychild`
- `go vet ./...`
- `make test-couch-start-recovery`
- `git diff --check`

Missing load-bearing cases:

- Cleanup probe/delete/kill failure followed by proof that a supervisor still owns and retries the live incarnation.
- PID reuse between server observation and destructive signal.
- Live conformance against an ephemeral real zellij session.

6. Architectural notes for upcoming work

- `ARCH-DRY`: **pass** — handshake and session-quiescence orchestration each have one authority.
- `ARCH-PURE`: **pass** — `ThreadRecord` and `StartTransaction` direct tests remain IO-free; integration behavior is separated.
- `ARCH-PURPOSE`: **flag** — occupied durable state alone does not fulfill “no untracked workspace writer,” and bare-PID killing violates exact-incarnation safety.
- `ARCH-MOCK`: **flag** — the stateful fake is valuable, but it cannot model PID reuse through the current API and has no live conformance counterpart.

7. Plan revision recommendations

Append a `## Revisions` entry stating:

- every post-ack exit retains active supervisor responsibility until whole-incarnation absence is proven;
- cleanup failure means retryable owned state, not merely occupied durable capacity;
- destructive server signaling carries and revalidates process-start identity;
- `SessionQuiescence` has a committed live-conformance target and cadence.

Also revise the atlas claim that “No error return can leave an unowned workspace writer” until BR-12 is actually closed.
