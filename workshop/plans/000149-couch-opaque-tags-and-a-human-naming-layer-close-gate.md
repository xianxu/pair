---
gate: boundary-review
issue: 149
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-26T11:42:03-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: a dead Pair client frees capacity while its zellij session can still write
          detail: admission.go:188 prunes an incarnation when its recorded Pair client PID is dead, but the issue contract and probes/zellijpark establish that zellij and its workspace-writing panes survive that client. A subsequent bounded-policy start can therefore admit a second writer. Retain the incarnation as unknown/occupied until whole-incarnation quiescence is proven, and pin dead-client-plus-live-session behavior with a stateful test (ARCH-PURPOSE, ARCH-MOCK).
          family: incarnation-quiescence-before-capacity-release
          round: 1
        - id: BR-2
          severity: Critical
          title: opaque tag allocation checks ThreadStore records but not scoped artifacts
          detail: threadtag.go:15 retries only ThreadExistsError from CreateThread, while the Spec and plan require collision checks against both composite records and tag-scoped artifacts. An orphaned draft, ledger, config, or session can be claimed as a new thread and opened by pair resume. Add the scoped-artifact collision seam and a test that pre-creates an artifact for the first entropy result and requires allocation to retry (ARCH-PURPOSE).
          family: composite-address-collision-domain
          round: 1
        - id: BR-3
          severity: Critical
          title: the Core concepts table labels a filesystem-backed namespace PURE and names a nonexistent integration
          detail: The plan lists CouchNamespace as PURE and NamespaceResolver as its integration, but no NamespaceResolver exists; ResolveCouchNamespace performs MkdirAll and EvalSymlinks directly, and namespace tests require mutable filesystem IO. Revise the plan to classify the actual entity as INTEGRATION or split a genuinely pure value from an injected filesystem resolver (ARCH-PURE).
          family: core-concept-kind-contract
          round: 1
        - id: BR-4
          severity: Important
          title: README still advertises the removed one-tree guard and --same-tree bypass
          detail: README.md:276 says registry membership enforces one agent per tree and --same-tree overrides it. M1 removes that flag and makes normalized provider policy the sole admission authority. Update the README and add a negative documentation check for removed flags (ARCH-PURPOSE).
          family: user-facing-policy-docs
          round: 1
        - id: BR-5
          severity: Important
          title: policy epoch exhaustion uses four attempts and a generic error instead of the promised typed refusal
          detail: The plan specifies at most three whole-cohort retries and a typed policy-unstable refusal; admission.go:62 performs four and returns fmt.Errorf. Align the retry count and type, and test exact calls, rollback, and zero forks under persistent mixed epochs.
          family: typed-retry-exhaustion-contract
          round: 1
        - id: BR-6
          severity: Important
          title: the plan's live-provider command uses the wrong variable and a nonportable relative binary
          detail: The plan sets ARIADNE_SDLC_BIN, while Makefile.local:62 consumes SDLC_BIN. Passing ../ariadne/bin/sdlc also fails because the test runs from its package directory; the absolute SDLC_BIN invocation passes. Canonicalize the target input and update the recorded command.
          family: live-conformance-target-interface
          round: 1
        - id: BR-7
          severity: Minor
          title: git diff --check fails over the requested boundary
          detail: workshop/issues/000150-in-session-continuation-style-compacting.md:13 has trailing whitespace, despite the M1 Log claiming git diff --check passed.
          family: verification-window-cleanliness
          round: 1
        - id: BR-8
          severity: Minor
          title: Spawn still comments that tags derive from trees and same-tree starts resume one session
          detail: couch.go:125-134 describes the superseded path-derived tag behavior immediately above code that launches the newly allocated opaque thread tag.
          family: identity-comment-truthfulness
          round: 1
      boundary: M1
      blocked: true
    - "n": 2
      timestamp: "2026-08-26T12:10:20-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: The dead-client release path is removed, and mutation of incumbent counting makes the whole-incarnation regression fail.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The requested sequential scoped-artifact collision seam and retry regression are present and fail when the collision branch is removed; a separate atomicity sibling is raised below.
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: The namespace text changed but has no red-without-fix contract test, and the same table still labels the effectful current Operation surface PURE.
          round: 2
        - id: BR-4
          disposition: addressed
          note: README now documents provider-owned admission, and its negative removed-flag test would fail on the prior text.
          round: 2
        - id: BR-5
          disposition: addressed
          note: PolicyUnstableError, three exact cohorts, call count, and rollback are pinned; restoring four attempts makes the regression fail.
          round: 2
        - id: BR-6
          disposition: addressed
          note: The target canonicalizes SDLC_BIN and the documented relative invocation passes; the prior direct relative invocation still demonstrates the original failure.
          round: 2
        - id: BR-7
          disposition: addressed
          note: Exact-window git diff --check now exits successfully.
          round: 2
        - id: BR-8
          disposition: not-addressed
          note: The comment is corrected, but no test fails if the obsolete path-derived identity claim returns.
          round: 2
      findings:
        - id: BR-9
          severity: Critical
          title: scoped artifact checking is a racy preflight rather than an atomic address claim
          detail: 'AllocateThreadTag checks artifacts before independently acquiring ThreadStore state. An artifact can appear between those operations and allocation still succeeds. This is the 2nd finding in this family: define one claim rule shared by every record, scoped-artifact, and session-binding producer rather than fixing another individual filename or adding a second scan.'
          family: composite-address-collision-domain
          round: 2
        - id: BR-10
          severity: Important
          title: README advertises couch stop although an active supervisor makes the CLI route refuse
          detail: stop requires ExecuteLiveOwner, so a second CLI process cannot acquire the lease held by the running console. Document the pre-147 limitation or route stop through the existing owner, with a held-lease regression.
          family: owner-required-command-reachability
          round: 2
        - id: BR-11
          severity: Important
          title: the project records M1 closed while its milestone row and boundary remain open
          detail: workshop/projects/couch.md leaves pair#149 M1 unchecked but records actual and closed metadata, contradicting the issue log's acceptance rule.
          family: milestone-state-truthfulness
          round: 2
      boundary: M1
      blocked: true
    - "n": 3
      timestamp: "2026-08-26T12:39:39-07:00"
      agent: codex
      dispose:
        - id: BR-2
          disposition: not-addressed
          note: The new claim scans only the ScopedPaths inventory; a source sweep found at least 18 current tag-owned filename shapes outside it, so orphaned artifacts such as draft-pane-, image-capture-, parked-, slug-, and review-* remain claimable. This is the 3rd finding in family composite-address-collision-domain.
          round: 3
        - id: BR-3
          disposition: addressed
          note: The plan classifies namespace resolution and the current effectful Operation surface as integration, and TestIssue149CurrentCoreConceptKinds fails on the prior table.
          round: 3
        - id: BR-8
          disposition: addressed
          note: The comment now describes opaque allocation, and TestOpaqueIdentityCommentDoesNotReintroducePathDerivedContract rejects both exact obsolete claims from the prior text.
          round: 3
        - id: BR-9
          disposition: addressed
          note: Couch and native Pair now acquire one O_EXCL marker before creation; concurrent-winner and cross-producer tests fail if that shared atomic authority is removed.
          round: 3
        - id: BR-10
          disposition: addressed
          note: README removes couch stop from the external command list, documents the pre-147 owner-routing limitation, and the regression fails on the prior advertised spelling.
          round: 3
        - id: BR-11
          disposition: not-addressed
          note: The premature closed field was removed, but no regression fails if closed metadata returns beside the unchecked M1 row; the gate contract therefore does not permit an addressed disposition.
          round: 3
      boundary: M1
      blocked: true
    - "n": 4
      timestamp: "2026-08-26T12:54:06-07:00"
      agent: codex
      dispose:
        - id: BR-2
          disposition: addressed
          note: The shared O_EXCL authority now precedes both Couch and native Pair creation, and the structural tag-boundary rule detects current non-ScopedPaths and unknown future artifact families. Disabling that rule makes the production-path collision regressions fail.
          round: 4
        - id: BR-11
          disposition: addressed
          note: Premature closed metadata is absent, and a repository-wide invariant rejects closed metadata beside any unchecked milestone. Restoring pair#149 M1's prior closed line makes the regression fail.
          round: 4
      boundary: M1
      blocked: false
    - "n": 5
      timestamp: "2026-08-26T13:31:44-07:00"
      agent: codex
      findings:
        - id: BR-12
          severity: Critical
          title: Post-ack failures return an error while leaving the workspace writer unowned
          detail: 'This is the 2nd finding in family `incarnation-quiescence-before-capacity-release`. After acknowledgement, registration failure at cmd/internal/couchcore/couch.go:228, promotion failure at line 234, and registry-save failure at line 250 can all return with the child still running. Operations.Invoke discards that handle on error at cmd/internal/couchcore/ops.go:89, and the CLI exits, violating the no-untracked-writer contract. Do not patch only the named registration branch: state one rule for every post-ack exit before ownership handoff—either quiesce and verify the exact incarnation, retaining occupied state when verification is unknown, or transfer the handle to a supervisor that remains responsible. Add table-driven integration tests for the complete failure-site enumeration; the current registration-failure test instead asserts that the writer survives.'
          family: incarnation-quiescence-before-capacity-release
          round: 5
        - id: BR-13
          severity: Critical
          title: PURE start entities are tested through mutable filesystem setup
          detail: This is the 2nd finding in family `core-concept-kind-contract`. The plan classifies ThreadRecord and StartTransaction as PURE, but validThreadRecord at cmd/internal/couchcore/thread_test.go:9 calls testCouchNamespace, which creates and resolves temporary directories; every admittedStartRecord test inherits that IO. Sweep every Core concepts row marked PURE and give its direct tests literal absolute paths and no exec, network, mutable filesystem, or integration fake. Keep the separate Runner lifecycle test explicitly integration-oriented.
          family: core-concept-kind-contract
          round: 5
        - id: BR-14
          severity: Important
          title: The blocked-start pipe protocol has two copy-pasted production authorities
          detail: 'ARCH-DRY: ExecRunner.StartBlocked at cmd/internal/couchcore/runner.go:67 and PtyRunner.StartBlocked at cmd/internal/couchcore/ptyrunner.go:55 duplicate the complete pipe creation, helper wrapping, descriptor-close, error-join, and acknowledged-handle protocol. Extract one shared helper parameterized by the underlying child-start function so future safety changes cannot drift between console and no-console starts.'
          family: blocked-runner-handshake-authority
          round: 5
      boundary: M2
      blocked: true
    - "n": 6
      timestamp: "2026-08-26T13:51:42-07:00"
      agent: codex
      dispose:
        - id: BR-12
          disposition: not-addressed
          note: Acknowledge can deliver the exec byte and then return a close error, after which Spawn calls an already-resolved Cancel and returns without quiescence; moreover the post-ack test models no durable descendants, while quiesceHandle proves only the Pair client exited.
          round: 6
        - id: BR-13
          disposition: addressed
          note: Direct ThreadRecord, StartTransaction, and Admission tests now use literal values, and TestIssue149PureCoreTestsStayAtPureBoundary fails if the forbidden IO or fake seams return.
          round: 6
        - id: BR-14
          disposition: not-addressed
          note: Both production runners now delegate to startBlockedChild, but no test fails if either runner reintroduces its own handshake protocol; the claimed-fix contract therefore remains unmet.
          round: 6
      findings:
        - id: BR-15
          severity: Critical
          title: The durable registration oracle is published through a truncate-and-rewrite window
          detail: 'ARCH-PURPOSE and ARCH-MOCK: establishReservedThreadAddress truncates the live claim before writing established state at cmd/internal/launcher/thread_claim.go:147, while Couch concurrently polls and strictly decodes that same path. A reader can observe empty or partial JSON and abort a valid start, and a crash can permanently strand malformed recovery evidence. Publish the transition atomically and add a synchronized real-filesystem test that proves readers observe only complete reserved or established records.'
          family: registration-evidence-atomic-publication
          round: 6
      boundary: M2
      blocked: true
    - "n": 7
      timestamp: "2026-08-26T14:19:04-07:00"
      agent: codex
      dispose:
        - id: BR-12
          disposition: not-addressed
          note: The real-descendant regression kills the descendant inside FakeThreadArtifactCollisionChecker.QuiesceHook; production ScopedThreadArtifactCollisionChecker.Quiesce only deletes an indexed zellij session, so the test proves a stronger fake behavior than the shipped boundary and leaves the whole-incarnation contract unverified.
          round: 7
        - id: BR-14
          disposition: addressed
          note: Both production runners delegate to startBlockedChild, and TestIssue149BlockedRunnersDelegateToOneHandshakeAuthority fails if either restores local pipe or acknowledged-handle construction.
          round: 7
        - id: BR-15
          disposition: addressed
          note: Registration now uses synced same-directory temporary publication, atomic rename, and directory sync; the synchronized filesystem regression observes complete reserved state before rename and complete established state afterward.
          round: 7
      boundary: M2
      blocked: true
    - "n": 8
      timestamp: "2026-08-26T14:32:42-07:00"
      agent: codex
      dispose:
        - id: BR-12
          disposition: not-addressed
          note: 'Process-group cleanup now covers real stdio and PTY descendants, but detached zellij quiescence is still unverified: OSRuntime.DeleteSession swallows the lingering-server kill result and returns success without proving the exact session absent, while the tests only assert that a recording deleter was called.'
          round: 8
      boundary: M2
      blocked: true
    - "n": 9
      timestamp: "2026-08-26T14:47:29-07:00"
      agent: codex
      dispose:
        - id: BR-12
          disposition: not-addressed
          note: This is the 3rd finding in family `incarnation-quiescence-before-capacity-release`. Successful cleanup now proves absence, but query, deletion, kill, timeout, or process-group quiescence errors return from Spawn with the handle discarded by Operations.Invoke; durable occupied state does not supervise a surviving workspace writer.
          round: 9
      findings:
        - id: BR-16
          severity: Critical
          title: Zellij cleanup can kill a process that reused an observed server PID
          detail: 'ARCH-PURPOSE: SessionServerPIDs returns bare integers at cmd/internal/launcher/session_quiescence.go:19 and KillProcess signals that PID later at lines 104-113 without checking a process-start identity. If the observed server exits and its PID is reused, cleanup can kill an unrelated process. Carry exact process-incarnation evidence through the seam and reauthorize immediately before signaling; add a stateful PID-reuse regression.'
          family: destructive-process-action-exact-identity
          round: 9
        - id: BR-17
          severity: Important
          title: The new zellij quiescence seam has no live conformance target
          detail: This is the 2nd finding in family `live-conformance-target-interface`. ARCH-MOCK requires live checks for the external behavior being modeled, but the committed coverage exercises only the stateful fake and parser literals; no target verifies the actual list-sessions, process-table, delete-session, and server-termination interface. State the live-interface rule for every SessionQuiescence operation and add an ephemeral-session conformance probe with a documented target and cadence.
          family: live-conformance-target-interface
          round: 9
      boundary: M2
      blocked: true
    - "n": 10
      timestamp: "2026-08-26T15:00:41-07:00"
      agent: codex
      dispose:
        - id: BR-12
          disposition: not-addressed
          note: Persistent handle-quiescence failure retries quiesceHandle with a new blocked Wait goroutine each attempt, while the retry regression exercises only artifact quiescence after the fake handle has already been reaped.
          round: 10
        - id: BR-16
          disposition: addressed
          note: Server observations carry PID plus start identity, production reauthorizes identity-command-identity immediately before signaling, and the PID-reuse table fails if bare-PID killing returns.
          round: 10
        - id: BR-17
          disposition: not-addressed
          note: The scheduled live target exists, but its test can pass when SessionServers returns no servers or KillServer is removed because ordinary delete-session may perform all termination itself.
          round: 10
      boundary: M2
      blocked: true
---

# Gate ledger — pair#149 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-26T11:42:03-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `incarnation-quiescence-before-capacity-release` a dead Pair client frees capacity while its zellij session can still write
  admission.go:188 prunes an incarnation when its recorded Pair client PID is dead, but the issue contract and probes/zellijpark establish that zellij and its workspace-writing panes survive that client. A subsequent bounded-policy start can therefore admit a second writer. Retain the incarnation as unknown/occupied until whole-incarnation quiescence is proven, and pin dead-client-plus-live-session behavior with a stateful test (ARCH-PURPOSE, ARCH-MOCK).
- **BR-2** [Critical] `composite-address-collision-domain` opaque tag allocation checks ThreadStore records but not scoped artifacts
  threadtag.go:15 retries only ThreadExistsError from CreateThread, while the Spec and plan require collision checks against both composite records and tag-scoped artifacts. An orphaned draft, ledger, config, or session can be claimed as a new thread and opened by pair resume. Add the scoped-artifact collision seam and a test that pre-creates an artifact for the first entropy result and requires allocation to retry (ARCH-PURPOSE).
- **BR-3** [Critical] `core-concept-kind-contract` the Core concepts table labels a filesystem-backed namespace PURE and names a nonexistent integration
  The plan lists CouchNamespace as PURE and NamespaceResolver as its integration, but no NamespaceResolver exists; ResolveCouchNamespace performs MkdirAll and EvalSymlinks directly, and namespace tests require mutable filesystem IO. Revise the plan to classify the actual entity as INTEGRATION or split a genuinely pure value from an injected filesystem resolver (ARCH-PURE).
- **BR-4** [Important] `user-facing-policy-docs` README still advertises the removed one-tree guard and --same-tree bypass
  README.md:276 says registry membership enforces one agent per tree and --same-tree overrides it. M1 removes that flag and makes normalized provider policy the sole admission authority. Update the README and add a negative documentation check for removed flags (ARCH-PURPOSE).
- **BR-5** [Important] `typed-retry-exhaustion-contract` policy epoch exhaustion uses four attempts and a generic error instead of the promised typed refusal
  The plan specifies at most three whole-cohort retries and a typed policy-unstable refusal; admission.go:62 performs four and returns fmt.Errorf. Align the retry count and type, and test exact calls, rollback, and zero forks under persistent mixed epochs.
- **BR-6** [Important] `live-conformance-target-interface` the plan's live-provider command uses the wrong variable and a nonportable relative binary
  The plan sets ARIADNE_SDLC_BIN, while Makefile.local:62 consumes SDLC_BIN. Passing ../ariadne/bin/sdlc also fails because the test runs from its package directory; the absolute SDLC_BIN invocation passes. Canonicalize the target input and update the recorded command.
- **BR-7** [Minor] `verification-window-cleanliness` git diff --check fails over the requested boundary
  workshop/issues/000150-in-session-continuation-style-compacting.md:13 has trailing whitespace, despite the M1 Log claiming git diff --check passed.
- **BR-8** [Minor] `identity-comment-truthfulness` Spawn still comments that tags derive from trees and same-tree starts resume one session
  couch.go:125-134 describes the superseded path-derived tag behavior immediately above code that launches the newly allocated opaque thread tag.

## Round 2 — 2026-08-26T12:10:20-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The dead-client release path is removed, and mutation of incumbent counting makes the whole-incarnation regression fail.
- BR-2 — addressed — The requested sequential scoped-artifact collision seam and retry regression are present and fail when the collision branch is removed; a separate atomicity sibling is raised below.
- BR-3 — not-addressed — The namespace text changed but has no red-without-fix contract test, and the same table still labels the effectful current Operation surface PURE.
- BR-4 — addressed — README now documents provider-owned admission, and its negative removed-flag test would fail on the prior text.
- BR-5 — addressed — PolicyUnstableError, three exact cohorts, call count, and rollback are pinned; restoring four attempts makes the regression fail.
- BR-6 — addressed — The target canonicalizes SDLC_BIN and the documented relative invocation passes; the prior direct relative invocation still demonstrates the original failure.
- BR-7 — addressed — Exact-window git diff --check now exits successfully.
- BR-8 — not-addressed — The comment is corrected, but no test fails if the obsolete path-derived identity claim returns.

### Raised

- **BR-9** [Critical] `composite-address-collision-domain` scoped artifact checking is a racy preflight rather than an atomic address claim
  AllocateThreadTag checks artifacts before independently acquiring ThreadStore state. An artifact can appear between those operations and allocation still succeeds. This is the 2nd finding in this family: define one claim rule shared by every record, scoped-artifact, and session-binding producer rather than fixing another individual filename or adding a second scan.
- **BR-10** [Important] `owner-required-command-reachability` README advertises couch stop although an active supervisor makes the CLI route refuse
  stop requires ExecuteLiveOwner, so a second CLI process cannot acquire the lease held by the running console. Document the pre-147 limitation or route stop through the existing owner, with a held-lease regression.
- **BR-11** [Important] `milestone-state-truthfulness` the project records M1 closed while its milestone row and boundary remain open
  workshop/projects/couch.md leaves pair#149 M1 unchecked but records actual and closed metadata, contradicting the issue log's acceptance rule.

## Round 3 — 2026-08-26T12:39:39-07:00 (codex) — BLOCKED

### Disposed

- BR-2 — not-addressed — The new claim scans only the ScopedPaths inventory; a source sweep found at least 18 current tag-owned filename shapes outside it, so orphaned artifacts such as draft-pane-, image-capture-, parked-, slug-, and review-* remain claimable. This is the 3rd finding in family composite-address-collision-domain.
- BR-3 — addressed — The plan classifies namespace resolution and the current effectful Operation surface as integration, and TestIssue149CurrentCoreConceptKinds fails on the prior table.
- BR-8 — addressed — The comment now describes opaque allocation, and TestOpaqueIdentityCommentDoesNotReintroducePathDerivedContract rejects both exact obsolete claims from the prior text.
- BR-9 — addressed — Couch and native Pair now acquire one O_EXCL marker before creation; concurrent-winner and cross-producer tests fail if that shared atomic authority is removed.
- BR-10 — addressed — README removes couch stop from the external command list, documents the pre-147 owner-routing limitation, and the regression fails on the prior advertised spelling.
- BR-11 — not-addressed — The premature closed field was removed, but no regression fails if closed metadata returns beside the unchecked M1 row; the gate contract therefore does not permit an addressed disposition.

## Round 4 — 2026-08-26T12:54:06-07:00 (codex) — passed

### Disposed

- BR-2 — addressed — The shared O_EXCL authority now precedes both Couch and native Pair creation, and the structural tag-boundary rule detects current non-ScopedPaths and unknown future artifact families. Disabling that rule makes the production-path collision regressions fail.
- BR-11 — addressed — Premature closed metadata is absent, and a repository-wide invariant rejects closed metadata beside any unchecked milestone. Restoring pair#149 M1's prior closed line makes the regression fail.

## Round 5 — 2026-08-26T13:31:44-07:00 (codex) — BLOCKED

### Raised

- **BR-12** [Critical] `incarnation-quiescence-before-capacity-release` Post-ack failures return an error while leaving the workspace writer unowned
  This is the 2nd finding in family `incarnation-quiescence-before-capacity-release`. After acknowledgement, registration failure at cmd/internal/couchcore/couch.go:228, promotion failure at line 234, and registry-save failure at line 250 can all return with the child still running. Operations.Invoke discards that handle on error at cmd/internal/couchcore/ops.go:89, and the CLI exits, violating the no-untracked-writer contract. Do not patch only the named registration branch: state one rule for every post-ack exit before ownership handoff—either quiesce and verify the exact incarnation, retaining occupied state when verification is unknown, or transfer the handle to a supervisor that remains responsible. Add table-driven integration tests for the complete failure-site enumeration; the current registration-failure test instead asserts that the writer survives.
- **BR-13** [Critical] `core-concept-kind-contract` PURE start entities are tested through mutable filesystem setup
  This is the 2nd finding in family `core-concept-kind-contract`. The plan classifies ThreadRecord and StartTransaction as PURE, but validThreadRecord at cmd/internal/couchcore/thread_test.go:9 calls testCouchNamespace, which creates and resolves temporary directories; every admittedStartRecord test inherits that IO. Sweep every Core concepts row marked PURE and give its direct tests literal absolute paths and no exec, network, mutable filesystem, or integration fake. Keep the separate Runner lifecycle test explicitly integration-oriented.
- **BR-14** [Important] `blocked-runner-handshake-authority` The blocked-start pipe protocol has two copy-pasted production authorities
  ARCH-DRY: ExecRunner.StartBlocked at cmd/internal/couchcore/runner.go:67 and PtyRunner.StartBlocked at cmd/internal/couchcore/ptyrunner.go:55 duplicate the complete pipe creation, helper wrapping, descriptor-close, error-join, and acknowledged-handle protocol. Extract one shared helper parameterized by the underlying child-start function so future safety changes cannot drift between console and no-console starts.

## Round 6 — 2026-08-26T13:51:42-07:00 (codex) — BLOCKED

### Disposed

- BR-12 — not-addressed — Acknowledge can deliver the exec byte and then return a close error, after which Spawn calls an already-resolved Cancel and returns without quiescence; moreover the post-ack test models no durable descendants, while quiesceHandle proves only the Pair client exited.
- BR-13 — addressed — Direct ThreadRecord, StartTransaction, and Admission tests now use literal values, and TestIssue149PureCoreTestsStayAtPureBoundary fails if the forbidden IO or fake seams return.
- BR-14 — not-addressed — Both production runners now delegate to startBlockedChild, but no test fails if either runner reintroduces its own handshake protocol; the claimed-fix contract therefore remains unmet.

### Raised

- **BR-15** [Critical] `registration-evidence-atomic-publication` The durable registration oracle is published through a truncate-and-rewrite window
  ARCH-PURPOSE and ARCH-MOCK: establishReservedThreadAddress truncates the live claim before writing established state at cmd/internal/launcher/thread_claim.go:147, while Couch concurrently polls and strictly decodes that same path. A reader can observe empty or partial JSON and abort a valid start, and a crash can permanently strand malformed recovery evidence. Publish the transition atomically and add a synchronized real-filesystem test that proves readers observe only complete reserved or established records.

## Round 7 — 2026-08-26T14:19:04-07:00 (codex) — BLOCKED

### Disposed

- BR-12 — not-addressed — The real-descendant regression kills the descendant inside FakeThreadArtifactCollisionChecker.QuiesceHook; production ScopedThreadArtifactCollisionChecker.Quiesce only deletes an indexed zellij session, so the test proves a stronger fake behavior than the shipped boundary and leaves the whole-incarnation contract unverified.
- BR-14 — addressed — Both production runners delegate to startBlockedChild, and TestIssue149BlockedRunnersDelegateToOneHandshakeAuthority fails if either restores local pipe or acknowledged-handle construction.
- BR-15 — addressed — Registration now uses synced same-directory temporary publication, atomic rename, and directory sync; the synchronized filesystem regression observes complete reserved state before rename and complete established state afterward.

## Round 8 — 2026-08-26T14:32:42-07:00 (codex) — BLOCKED

### Disposed

- BR-12 — not-addressed — Process-group cleanup now covers real stdio and PTY descendants, but detached zellij quiescence is still unverified: OSRuntime.DeleteSession swallows the lingering-server kill result and returns success without proving the exact session absent, while the tests only assert that a recording deleter was called.

## Round 9 — 2026-08-26T14:47:29-07:00 (codex) — BLOCKED

### Disposed

- BR-12 — not-addressed — This is the 3rd finding in family `incarnation-quiescence-before-capacity-release`. Successful cleanup now proves absence, but query, deletion, kill, timeout, or process-group quiescence errors return from Spawn with the handle discarded by Operations.Invoke; durable occupied state does not supervise a surviving workspace writer.

### Raised

- **BR-16** [Critical] `destructive-process-action-exact-identity` Zellij cleanup can kill a process that reused an observed server PID
  ARCH-PURPOSE: SessionServerPIDs returns bare integers at cmd/internal/launcher/session_quiescence.go:19 and KillProcess signals that PID later at lines 104-113 without checking a process-start identity. If the observed server exits and its PID is reused, cleanup can kill an unrelated process. Carry exact process-incarnation evidence through the seam and reauthorize immediately before signaling; add a stateful PID-reuse regression.
- **BR-17** [Important] `live-conformance-target-interface` The new zellij quiescence seam has no live conformance target
  This is the 2nd finding in family `live-conformance-target-interface`. ARCH-MOCK requires live checks for the external behavior being modeled, but the committed coverage exercises only the stateful fake and parser literals; no target verifies the actual list-sessions, process-table, delete-session, and server-termination interface. State the live-interface rule for every SessionQuiescence operation and add an ephemeral-session conformance probe with a documented target and cadence.

## Round 10 — 2026-08-26T15:00:41-07:00 (codex) — BLOCKED

### Disposed

- BR-12 — not-addressed — Persistent handle-quiescence failure retries quiesceHandle with a new blocked Wait goroutine each attempt, while the retry regression exercises only artifact quiescence after the fake handle has already been reaped.
- BR-16 — addressed — Server observations carry PID plus start identity, production reauthorizes identity-command-identity immediately before signaling, and the PID-reuse table fails if bare-PID killing returns.
- BR-17 — not-addressed — The scheduled live target exists, but its test can pass when SessionServers returns no servers or KillServer is removed because ordinary delete-session may perform all termination itself.

## Open findings

- **BR-12** [Critical] `incarnation-quiescence-before-capacity-release` Post-ack failures return an error while leaving the workspace writer unowned
- **BR-17** [Important] `live-conformance-target-interface` The new zellij quiescence seam has no live conformance target
