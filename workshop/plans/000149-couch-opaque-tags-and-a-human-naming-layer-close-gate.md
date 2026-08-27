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
    - "n": 11
      timestamp: "2026-08-26T15:16:18-07:00"
      agent: codex
      dispose:
        - id: BR-12
          disposition: addressed
          note: Post-ack cleanup retains the handle until process-group and session absence are proven, and reverting reusable wait ownership makes TestPostAckHandleRetriesReuseOneWaiterUntilReap fail with three waiters instead of one.
          round: 11
        - id: BR-17
          disposition: not-addressed
          note: The live decorator sets killAttempted before calling the inner KillServer, so it does not prove that the production killProcess operation actually dispatches; delete-session can produce final absence while that external operation is ineffective.
          round: 11
      boundary: M2
      blocked: false
    - "n": 12
      timestamp: "2026-08-26T15:26:01-07:00"
      agent: codex
      dispose:
        - id: BR-17
          disposition: addressed
          note: 'The live test now makes OS termination load-bearing: an exact-argv sentinel survives ordinary zellij deletion, must be discovered and identity-reauthorized through production, and sets killedSentinel only at the injected killProcess boundary.'
          round: 12
      boundary: M2
      blocked: false
    - "n": 13
      timestamp: "2026-08-26T16:19:43-07:00"
      agent: codex
      findings:
        - id: BR-18
          severity: Critical
          title: Corrupt ThreadIndex errors silently fall back to launching the input as a legacy tag
          detail: LoadThreadIndex explicitly fails closed on corrupt or incomplete state, but createflow.go:164-166 ignores that error and resolveResumeTag at createflow.go:852-862 returns the original argument for every read failure. Thus `pair resume compiler` can create or resume a direct `compiler` tag when the authoritative index is corrupt. Distinguish an absent store from malformed/incomplete state, surface the latter, and add a production-flow regression that fails without the error propagation (ARCH-PURPOSE).
          family: durable-index-read-failure-authority
          round: 13
        - id: BR-19
          severity: Critical
          title: Required-argument validation makes documented metadata clearing unreachable
          detail: validateOperationCall at operationdispatch.go:90-93 treats an empty value as omission. Consequently `couch name <ref> ""` and `couch publish-description ""` are rejected even though ThreadMetadataPatch promises explicit empty values clear those fields. Validate required argument presence rather than non-empty content where empty is meaningful, and pin both operation paths with red-without-fix tests.
          family: metadata-empty-value-contract
          round: 13
        - id: BR-20
          severity: Critical
          title: CLI metadata operations cannot supply the repository scope needed to address repeated tags
          detail: 'This is the 3rd finding in family `composite-address-collision-domain`. name/describe declare repo-scope as implicit at ops.go:140-155, but bindArgs excludes implicit fields and RunWithRuntime only populates scope/tag for publish-description at run.go:127-137. DirectStoreExecutor therefore resolves CLI name/describe with an empty scope at operationdispatch.go:124-138, making a repeated legacy tag globally ambiguous even when invoked from its repository. Do not patch only name: state and enforce the rule that every composite-address consumer either carries an exact address or derives the current repository scope, then sweep show/name/describe and future metadata clients. Add a CLI-level repeated-tag regression (ARCH-PURPOSE).'
          family: composite-address-collision-domain
          round: 13
        - id: BR-21
          severity: Critical
          title: Initial console attachment bypasses the declared attach operation
          detail: The plan requires every effectful human action to flow through DispatchOperation, and attach is declared specifically for a newly started terminal. Panel starts comply at console.go:1061-1069, but the initial `couch start` path calls AttachThreadActor directly at run.go:248-266. Route this path through the same typed attach executor and add a wiring test that fails if the declaration is bypassed (ARCH-DRY, ARCH-PURPOSE).
          family: operation-dispatch-single-authority
          round: 13
        - id: BR-22
          severity: Critical
          title: The Core concepts table no longer describes the M3 entities and purity boundary
          detail: 'This is the 3rd finding in family `core-concept-kind-contract`. The plan names a nonexistent `ThreadMetadata` entity and classifies its mixed pure/store file as PURE at plan.md:47, while `Operation` remains classified INTEGRATION at line 48 after its declarations became pure and its executors moved to the integration table at line 81. Do not patch these two cells in isolation: state the rule that each row names a greppable entity and one architectural kind, audit the full table, and append a plan revision recording the corrected PURE metadata transition/declarations and INTEGRATION store/executor surfaces.'
          family: core-concept-kind-contract
          round: 13
        - id: BR-23
          severity: Important
          title: README still documents the pre-M3 tree and actor interface
          detail: 'This is the 2nd finding in family `user-facing-policy-docs`. README.md:267-272 still says list shows actors, show targets one tree, and name/describe mutate tree metadata; README.md:326 documents only resume-by-tag and omits human-name resolution and picker behavior. Do not fix only these lines: enumerate every M3 user-facing command, lookup, rendering, and clearing behavior and sweep the README against that inventory.'
          family: user-facing-policy-docs
          round: 13
      boundary: M3
      blocked: true
    - "n": 14
      timestamp: "2026-08-26T16:43:44-07:00"
      agent: codex
      dispose:
        - id: BR-18
          disposition: addressed
          note: Production launch now distinguishes typed store absence from corruption, with corrupt-index and absent-store flow regressions.
          round: 14
        - id: BR-19
          disposition: addressed
          note: Required arguments now validate map presence, and CLI tests pin empty name, description, and published-summary clearing.
          round: 14
        - id: BR-20
          disposition: addressed
          note: CLI show, name, and describe derive Git-root repository scope; a repeated-tag regression covers reads and writes across scopes.
          round: 14
        - id: BR-21
          disposition: addressed
          note: Initial attachment dispatches the typed attach operation, and exact pane registration is private to the console executor.
          round: 14
        - id: BR-22
          disposition: addressed
          note: The audited Core concepts table names greppable PURE entities separately from INTEGRATION store and executor surfaces.
          round: 14
        - id: BR-23
          disposition: addressed
          note: README now inventories M3 commands, scoped lookup, rendering, clearing, standalone resolution, and picker behavior.
          round: 14
      findings:
        - id: BR-24
          severity: Critical
          title: Named couch show output drops the durable tag promised for diagnostics
          detail: The Spec requires list to stop leading with the system id while retaining it for show and diagnostics, but run.go:432-433 sends list and show through the same renderer and run.go:454-480 emits only ThreadSummary.Label(), which replaces a named thread's tag completely. A named `couch show compiler` therefore cannot reveal the immutable address needed for exact follow-up operations. Preserve name-first list output while making show include the opaque tag, and add a named-show CLI regression that fails when the tag is absent (ARCH-PURPOSE).
          family: detail-view-preserves-durable-identity
          round: 14
        - id: BR-25
          severity: Critical
          title: Panel callbacks silently turn authoritative ThreadStore failures into empty results
          detail: 'This is the 2nd finding in family `durable-index-read-failure-authority`. run.go:331-341 discards errors from both ResolveThreadReference and ThreadInventory, while console.go:190-218 and console.go:868-906 expose callbacks that cannot return an error. A corrupt or incomplete store can therefore replace the authoritative panel with an empty list or no matches without any notice. Do not patch only one closure: state the rule that every durable-record read either returns valid state or surfaces its failure, change the callback boundary accordingly, and add a production-wiring regression using a failing store read (ARCH-PURPOSE).'
          family: durable-index-read-failure-authority
          round: 14
      boundary: M3
      blocked: true
    - "n": 15
      timestamp: "2026-08-26T16:57:06-07:00"
      agent: codex
      dispose:
        - id: BR-24
          disposition: addressed
          note: Named show output now includes the immutable composite address, and the CLI regression fails if the opaque tag is removed.
          round: 15
        - id: BR-25
          disposition: addressed
          note: Both production panel callbacks propagate ThreadStore failures, with real corrupt-store wiring tests and visible-error console regressions.
          round: 15
      findings:
        - id: BR-26
          severity: Critical
          title: Standalone Pair accepts ThreadStore records that Couch rejects as invalid
          detail: 'This is the 3rd finding in family `durable-index-read-failure-authority`. The shadow record schema in thread_index.go:54-63 omits required `starting_path` and `claim_generation`, and LoadThreadIndex at lines 99-110 consequently accepts the fixture at thread_index_test.go:49-58 even though Couch rejects that same record at thread.go:73-80 and threadstore.go:578-590. This permits malformed or incomplete authoritative records to resolve human names and launch opaque tags while Couch refuses the store. Do NOT patch only these two fields: state the rule that every durable-record reader shares the authoritative structural acceptance contract, enumerate all persisted invariants, and derive the portable projection from a common lower-layer decoder or enforce acceptance parity with conformance tests (ARCH-DRY, ARCH-PURPOSE).'
          family: durable-index-read-failure-authority
          round: 15
      boundary: M3
      blocked: true
    - "n": 16
      timestamp: "2026-08-26T17:16:34-07:00"
      agent: codex
      dispose:
        - id: BR-26
          disposition: addressed
          note: Both readers use the complete shared persisted-record decoder, and the real-store mutation test fails when Launcher is reverted to its former partial schema.
          round: 16
      boundary: M3
      blocked: false
    - "n": 17
      timestamp: "2026-08-26T17:51:53-07:00"
      agent: codex
      findings:
        - id: BR-27
          severity: Critical
          title: M4's Core concepts inventory omits and misstates implemented architectural entities
          detail: 'This is the 4th finding in family `core-concept-kind-contract`. Earlier rounds fixed instances. Do NOT fix only one row: state the rule that every milestone-added or modified architectural entity must have a greppable row with its correct kind, path, and current status, then sweep the full M4 diff. The table at plan line 37 omits LaunchProfile, PathLaunchPreference, and the strict Couch profile-wire integration, while the ThreadRecord and threadrecord.Record statuses do not acknowledge their M4 widening (ARCH-PURE, ARCH-PURPOSE).'
          family: core-concept-kind-contract
          round: 17
        - id: BR-28
          severity: Important
          title: An explicitly empty agent selection silently launches the fallback agent
          detail: bindArgs preserves `--agent=` as a present empty string at couchcmd/run.go:400, but CouchLiveOwnerExecutor passes only the value and ResolveLaunchProfile treats empty as no explicit selection. Reject missing or empty values for value-bearing flags before Spawn, distinguish them from boolean switches, and add public CLI plus generic-dispatch tests that fail without the fix.
          family: value-bearing-flag-contract
          round: 17
      boundary: M4
      blocked: true
    - "n": 18
      timestamp: "2026-08-26T18:03:09-07:00"
      agent: codex
      dispose:
        - id: BR-27
          disposition: addressed
          note: The plan now states the whole-milestone inventory rule and the full M4 sweep accurately inventories the material PURE and INTEGRATION entities, paths, and statuses.
          round: 18
        - id: BR-28
          disposition: not-addressed
          note: CLI binding rejects bare and empty agent flags, but transport-neutral DispatchOperation does not enforce ValueRequired and has no regression proving an empty agent cannot reach the live executor.
          round: 18
      findings:
        - id: BR-29
          severity: Minor
          title: The M4 disposition commit fails exact-window whitespace verification
          detail: 'This is the 2nd finding in family `verification-window-cleanliness`. Earlier rounds fixed an instance. Do NOT fix only these lines: state and apply the rule that every boundary and disposition artifact is included in the final exact-window `git diff --check` sweep. The current failure is trailing whitespace at the M4 review artifact lines 62-63.'
          family: verification-window-cleanliness
          round: 18
      boundary: M4
      blocked: true
    - "n": 19
      timestamp: "2026-08-26T18:12:58-07:00"
      agent: codex
      dispose:
        - id: BR-28
          disposition: addressed
          note: Shared dispatch now rejects empty value-bearing arguments before executor selection; the generic regression fails when that validation is removed, and public CLI tests pin bare and empty agent flags.
          round: 19
        - id: BR-29
          disposition: addressed
          note: The complete M4 boundary range, including generated review and ledger artifacts, now passes exact-window git diff --check.
          round: 19
      findings:
        - id: BR-30
          severity: Important
          title: Inherited repository-default policy can reject a valid remembered path profile
          detail: 'ARCH-PURPOSE and ARCH-MOCK: Couch emits PAIR_USE_REPO_DEFAULT only for repository-default provenance, while ExecRunner inherits the parent environment. A stale inherited value of 1 therefore reaches Pair during a path-derived launch and contradicts the otherwise valid profile. Sanitize the inherited launch-owned key and emit one authoritative state for both provenance outcomes, with a production-runner regression that starts under stale parent state.'
          family: child-policy-environment-isolation
          round: 19
      boundary: M4
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

## Round 11 — 2026-08-26T15:16:18-07:00 (codex) — passed

### Disposed

- BR-12 — addressed — Post-ack cleanup retains the handle until process-group and session absence are proven, and reverting reusable wait ownership makes TestPostAckHandleRetriesReuseOneWaiterUntilReap fail with three waiters instead of one.
- BR-17 — not-addressed — The live decorator sets killAttempted before calling the inner KillServer, so it does not prove that the production killProcess operation actually dispatches; delete-session can produce final absence while that external operation is ineffective.

## Round 12 — 2026-08-26T15:26:01-07:00 (codex) — passed

### Disposed

- BR-17 — addressed — The live test now makes OS termination load-bearing: an exact-argv sentinel survives ordinary zellij deletion, must be discovered and identity-reauthorized through production, and sets killedSentinel only at the injected killProcess boundary.

## Round 13 — 2026-08-26T16:19:43-07:00 (codex) — BLOCKED

### Raised

- **BR-18** [Critical] `durable-index-read-failure-authority` Corrupt ThreadIndex errors silently fall back to launching the input as a legacy tag
  LoadThreadIndex explicitly fails closed on corrupt or incomplete state, but createflow.go:164-166 ignores that error and resolveResumeTag at createflow.go:852-862 returns the original argument for every read failure. Thus `pair resume compiler` can create or resume a direct `compiler` tag when the authoritative index is corrupt. Distinguish an absent store from malformed/incomplete state, surface the latter, and add a production-flow regression that fails without the error propagation (ARCH-PURPOSE).
- **BR-19** [Critical] `metadata-empty-value-contract` Required-argument validation makes documented metadata clearing unreachable
  validateOperationCall at operationdispatch.go:90-93 treats an empty value as omission. Consequently `couch name <ref> ""` and `couch publish-description ""` are rejected even though ThreadMetadataPatch promises explicit empty values clear those fields. Validate required argument presence rather than non-empty content where empty is meaningful, and pin both operation paths with red-without-fix tests.
- **BR-20** [Critical] `composite-address-collision-domain` CLI metadata operations cannot supply the repository scope needed to address repeated tags
  This is the 3rd finding in family `composite-address-collision-domain`. name/describe declare repo-scope as implicit at ops.go:140-155, but bindArgs excludes implicit fields and RunWithRuntime only populates scope/tag for publish-description at run.go:127-137. DirectStoreExecutor therefore resolves CLI name/describe with an empty scope at operationdispatch.go:124-138, making a repeated legacy tag globally ambiguous even when invoked from its repository. Do not patch only name: state and enforce the rule that every composite-address consumer either carries an exact address or derives the current repository scope, then sweep show/name/describe and future metadata clients. Add a CLI-level repeated-tag regression (ARCH-PURPOSE).
- **BR-21** [Critical] `operation-dispatch-single-authority` Initial console attachment bypasses the declared attach operation
  The plan requires every effectful human action to flow through DispatchOperation, and attach is declared specifically for a newly started terminal. Panel starts comply at console.go:1061-1069, but the initial `couch start` path calls AttachThreadActor directly at run.go:248-266. Route this path through the same typed attach executor and add a wiring test that fails if the declaration is bypassed (ARCH-DRY, ARCH-PURPOSE).
- **BR-22** [Critical] `core-concept-kind-contract` The Core concepts table no longer describes the M3 entities and purity boundary
  This is the 3rd finding in family `core-concept-kind-contract`. The plan names a nonexistent `ThreadMetadata` entity and classifies its mixed pure/store file as PURE at plan.md:47, while `Operation` remains classified INTEGRATION at line 48 after its declarations became pure and its executors moved to the integration table at line 81. Do not patch these two cells in isolation: state the rule that each row names a greppable entity and one architectural kind, audit the full table, and append a plan revision recording the corrected PURE metadata transition/declarations and INTEGRATION store/executor surfaces.
- **BR-23** [Important] `user-facing-policy-docs` README still documents the pre-M3 tree and actor interface
  This is the 2nd finding in family `user-facing-policy-docs`. README.md:267-272 still says list shows actors, show targets one tree, and name/describe mutate tree metadata; README.md:326 documents only resume-by-tag and omits human-name resolution and picker behavior. Do not fix only these lines: enumerate every M3 user-facing command, lookup, rendering, and clearing behavior and sweep the README against that inventory.

## Round 14 — 2026-08-26T16:43:44-07:00 (codex) — BLOCKED

### Disposed

- BR-18 — addressed — Production launch now distinguishes typed store absence from corruption, with corrupt-index and absent-store flow regressions.
- BR-19 — addressed — Required arguments now validate map presence, and CLI tests pin empty name, description, and published-summary clearing.
- BR-20 — addressed — CLI show, name, and describe derive Git-root repository scope; a repeated-tag regression covers reads and writes across scopes.
- BR-21 — addressed — Initial attachment dispatches the typed attach operation, and exact pane registration is private to the console executor.
- BR-22 — addressed — The audited Core concepts table names greppable PURE entities separately from INTEGRATION store and executor surfaces.
- BR-23 — addressed — README now inventories M3 commands, scoped lookup, rendering, clearing, standalone resolution, and picker behavior.

### Raised

- **BR-24** [Critical] `detail-view-preserves-durable-identity` Named couch show output drops the durable tag promised for diagnostics
  The Spec requires list to stop leading with the system id while retaining it for show and diagnostics, but run.go:432-433 sends list and show through the same renderer and run.go:454-480 emits only ThreadSummary.Label(), which replaces a named thread's tag completely. A named `couch show compiler` therefore cannot reveal the immutable address needed for exact follow-up operations. Preserve name-first list output while making show include the opaque tag, and add a named-show CLI regression that fails when the tag is absent (ARCH-PURPOSE).
- **BR-25** [Critical] `durable-index-read-failure-authority` Panel callbacks silently turn authoritative ThreadStore failures into empty results
  This is the 2nd finding in family `durable-index-read-failure-authority`. run.go:331-341 discards errors from both ResolveThreadReference and ThreadInventory, while console.go:190-218 and console.go:868-906 expose callbacks that cannot return an error. A corrupt or incomplete store can therefore replace the authoritative panel with an empty list or no matches without any notice. Do not patch only one closure: state the rule that every durable-record read either returns valid state or surfaces its failure, change the callback boundary accordingly, and add a production-wiring regression using a failing store read (ARCH-PURPOSE).

## Round 15 — 2026-08-26T16:57:06-07:00 (codex) — BLOCKED

### Disposed

- BR-24 — addressed — Named show output now includes the immutable composite address, and the CLI regression fails if the opaque tag is removed.
- BR-25 — addressed — Both production panel callbacks propagate ThreadStore failures, with real corrupt-store wiring tests and visible-error console regressions.

### Raised

- **BR-26** [Critical] `durable-index-read-failure-authority` Standalone Pair accepts ThreadStore records that Couch rejects as invalid
  This is the 3rd finding in family `durable-index-read-failure-authority`. The shadow record schema in thread_index.go:54-63 omits required `starting_path` and `claim_generation`, and LoadThreadIndex at lines 99-110 consequently accepts the fixture at thread_index_test.go:49-58 even though Couch rejects that same record at thread.go:73-80 and threadstore.go:578-590. This permits malformed or incomplete authoritative records to resolve human names and launch opaque tags while Couch refuses the store. Do NOT patch only these two fields: state the rule that every durable-record reader shares the authoritative structural acceptance contract, enumerate all persisted invariants, and derive the portable projection from a common lower-layer decoder or enforce acceptance parity with conformance tests (ARCH-DRY, ARCH-PURPOSE).

## Round 16 — 2026-08-26T17:16:34-07:00 (codex) — passed

### Disposed

- BR-26 — addressed — Both readers use the complete shared persisted-record decoder, and the real-store mutation test fails when Launcher is reverted to its former partial schema.

## Round 17 — 2026-08-26T17:51:53-07:00 (codex) — BLOCKED

### Raised

- **BR-27** [Critical] `core-concept-kind-contract` M4's Core concepts inventory omits and misstates implemented architectural entities
  This is the 4th finding in family `core-concept-kind-contract`. Earlier rounds fixed instances. Do NOT fix only one row: state the rule that every milestone-added or modified architectural entity must have a greppable row with its correct kind, path, and current status, then sweep the full M4 diff. The table at plan line 37 omits LaunchProfile, PathLaunchPreference, and the strict Couch profile-wire integration, while the ThreadRecord and threadrecord.Record statuses do not acknowledge their M4 widening (ARCH-PURE, ARCH-PURPOSE).
- **BR-28** [Important] `value-bearing-flag-contract` An explicitly empty agent selection silently launches the fallback agent
  bindArgs preserves `--agent=` as a present empty string at couchcmd/run.go:400, but CouchLiveOwnerExecutor passes only the value and ResolveLaunchProfile treats empty as no explicit selection. Reject missing or empty values for value-bearing flags before Spawn, distinguish them from boolean switches, and add public CLI plus generic-dispatch tests that fail without the fix.

## Round 18 — 2026-08-26T18:03:09-07:00 (codex) — BLOCKED

### Disposed

- BR-27 — addressed — The plan now states the whole-milestone inventory rule and the full M4 sweep accurately inventories the material PURE and INTEGRATION entities, paths, and statuses.
- BR-28 — not-addressed — CLI binding rejects bare and empty agent flags, but transport-neutral DispatchOperation does not enforce ValueRequired and has no regression proving an empty agent cannot reach the live executor.

### Raised

- **BR-29** [Minor] `verification-window-cleanliness` The M4 disposition commit fails exact-window whitespace verification
  This is the 2nd finding in family `verification-window-cleanliness`. Earlier rounds fixed an instance. Do NOT fix only these lines: state and apply the rule that every boundary and disposition artifact is included in the final exact-window `git diff --check` sweep. The current failure is trailing whitespace at the M4 review artifact lines 62-63.

## Round 19 — 2026-08-26T18:12:58-07:00 (codex) — BLOCKED

### Disposed

- BR-28 — addressed — Shared dispatch now rejects empty value-bearing arguments before executor selection; the generic regression fails when that validation is removed, and public CLI tests pin bare and empty agent flags.
- BR-29 — addressed — The complete M4 boundary range, including generated review and ledger artifacts, now passes exact-window git diff --check.

### Raised

- **BR-30** [Important] `child-policy-environment-isolation` Inherited repository-default policy can reject a valid remembered path profile
  ARCH-PURPOSE and ARCH-MOCK: Couch emits PAIR_USE_REPO_DEFAULT only for repository-default provenance, while ExecRunner inherits the parent environment. A stale inherited value of 1 therefore reaches Pair during a path-derived launch and contradicts the otherwise valid profile. Sanitize the inherited launch-owned key and emit one authoritative state for both provenance outcomes, with a production-runner regression that starts under stale parent state.

## Open findings

- **BR-30** [Important] `child-policy-environment-isolation` Inherited repository-default policy can reject a valid remembered path profile
