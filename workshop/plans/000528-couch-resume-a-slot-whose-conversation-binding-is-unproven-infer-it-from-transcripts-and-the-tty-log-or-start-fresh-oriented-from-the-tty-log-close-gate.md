---
gate: boundary-review
issue: 528
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-13T11:01:13-07:00"
      dispatched: "2026-10-13T10:58:01-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Unknown terminal observations become live or actionable relaunch verdicts
          detail: cmd/internal/couchtty/menu_reattach.go:373 promotes ReasonUnknown to ThreadLive whenever an attached-pass entry exists. cmd/internal/launcher/session_handoff_os.go:73-97 also collapses inspection errors into false usability, which cmd/internal/couchcore/agent_usability.go:7 converts to needs-relaunch. Preserve typed usable/unsafe/unknown outcomes through observers, classifiers, and overlays; test failed refresh after successful attachment and failed metadata inspection. ARCH-ORDER, ARCH-PURPOSE.
          family: preserve-observation-uncertainty
          round: 1
        - id: BR-2
          severity: Critical
          title: Agent usability accepts a recycled agent PID beneath the same server
          detail: cmd/internal/launcher/terminal_agent.go:7-34 compares only the server generation; cmd/internal/launcher/session_handoff_os.go:205 supplies a readiness PID without its expected start identity. A replacement descendant reusing that PID passes stale readiness. Persist and validate the agent generation, with a regression changing only the agent identity while preserving PID and ancestry. ARCH-SECURE.
          family: exact-process-generation-authority
          round: 1
        - id: BR-3
          severity: Important
          title: Policy and readiness reads can defeat the admission timeout
          detail: cmd/internal/launcher/session_handoff_os.go:127 opens policy files without regular-file/nonblocking safeguards, and readiness reads at lines 193 and 423 are unbounded. A FIFO can hang refresh or admission indefinitely because cancellation cannot interrupt these reads and the coordinator joins the observer. Share the cold-check branch's bounded regular-file reader across these consumers and test FIFO and oversized inputs. ARCH-CONSTRAINTS, ARCH-SECURE.
          family: bounded-untrusted-artifact-reads
          round: 1
        - id: BR-4
          severity: Important
          title: Per-terminal coordination files accumulate without an aggregate lifecycle bound
          detail: cmd/internal/artifactpath/session_policy.go:151-165 allocates one lock pathname per canonical socket, while cmd/internal/artifactpath/manifest.go:1460 permanently protects every such inode. Define a bounded coordination scheme or safe retirement protocol without unlinking active or awaited locks; document and measure any intentional permanent per-session cost. ARCH-FUNERAL.
          family: durable-artifact-lifecycle-bound
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-13T11:34:12-07:00"
      dispatched: "2026-10-13T11:30:13-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Typed uncertainty reaches core projection and the attached-pass overlay. Focused tests pass; removing the unknown-state overlay guard or converting inspection errors to unsafe makes the respective regressions fail.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Readiness records carry agent start identity and AgentBelongsToServer validates it. Removing the agent-identity comparison fails the unchanged-PID/ancestry regression.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Policy and readiness consumers use readBoundedRegularMetadata. Its regressions pass; replacing it with os.ReadFile fails FIFO, oversized-input, symlink and cancellation cases. Additional consumers missed by the family sweep are reported separately below.
          round: 2
        - id: BR-4
          disposition: addressed
          note: TerminalLease now selects among 256 permanent stripes. The 10,000-socket regression passes and fails when full-hash per-socket lock paths are restored.
          round: 2
      findings:
        - id: BR-5
          severity: Important
          title: Capture-witness and admission-config reads still permit indefinite blocking
          detail: 'cmd/internal/storagegc/capture_metadata.go:142 opens events with blocking os.Open before its byte limit; a FIFO without a writer blocks capture preservation indefinitely, reproduced by a disposable regression probe. cmd/internal/launcher/session_handoff_os.go:468 also reads config.kdl with unbounded, non-cancellable os.ReadFile, so admission can remain stuck beyond its timeout while the coordinator waits for the observer. This is the 2nd finding in family bounded-untrusted-artifact-reads. Do NOT fix only these instances: enumerate every persisted-input read added by M1 and enforce one bounded, regular-file, nonblocking, cancellation-aware rule through a shared IO owner, with consumer-level FIFO and oversized-input regressions. ARCH-CONSTRAINTS, ARCH-SECURE, ARCH-PURPOSE.'
          family: bounded-untrusted-artifact-reads
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-13T12:06:16-07:00"
      dispatched: "2026-10-13T12:03:38-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Prior disposition retained; unknown-observation and reattach-overlay regressions pass.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Prior disposition retained; AgentBelongsToServer checks both server and agent process generations.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Prior disposition retained; policy and readiness consumers use the shared bounded reader, with passing FIFO regressions.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Prior disposition retained; permanent coordination files remain bounded to 256 stripes, with passing lease tests.
          round: 3
        - id: BR-5
          disposition: addressed
          note: Enumerated metadata consumers use osfs.ReadRegularMetadata. Consumer FIFO, oversized-input and cancellation tests pass; substituting os.ReadFile in a disposable checkout makes the regressions fail. Both disposable checkouts were removed.
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: dfedd894650cbf1f3cc2dbdb5782b8724131f01d
      blocked: false
    - "n": 4
      timestamp: "2026-10-13T13:26:16-07:00"
      dispatched: "2026-10-13T13:20:56-07:00"
      agent: codex
      findings:
        - id: BR-6
          severity: Critical
          title: Native ownership substitutes the latest tag-level launch for the live wrapper's launch
          detail: 'cmd/internal/launcher/recovery_native_owner.go:85 uses CurrentLaunch(scope, tag, agent) without associating that launch with the observed wrapper PID/start identity. A regression retaining the old wrapper while appending a newer launch targeting another conversation incorrectly returns success. This is the 2nd finding in family `exact-process-generation-authority`. Earlier rounds fixed instances. Do NOT fix only this instance: state and enforce the rule that every native-ownership assertion must bind to the exact observed process generation; sweep preflight, Pair admission, source exemptions, and their doubles. Missing association must remain unresolved. ARCH-SECURE, ARCH-ORDER, ARCH-MOCK.'
          family: exact-process-generation-authority
          round: 4
        - id: BR-7
          severity: Critical
          title: The advertised primary Relaunch action never opens the new recovery preview
          detail: cmd/internal/couchtty/menu.go:864 handles only the new resume-candidates item, appended separately at menu_switchagent.go:204. Unsafe-row Enter/mouse still select relaunch at menu.go:803 and open the old confirmation through menu_slot.go:34; the old executor requires established binding. The primary-route regression fails with the confirmation frame and no preview effect. Route Enter, mouse, and the unsafe-row Relaunch item through the same evidence preview while preserving ordinary healthy relaunch behavior. ARCH-PURPOSE.
          family: recovery-primary-action-routing
          round: 4
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 5
      timestamp: "2026-10-13T13:39:34-07:00"
      dispatched: "2026-10-13T13:34:13-07:00"
      agent: codex
      dispose:
        - id: BR-6
          disposition: addressed
          note: Ownership now resolves the launch ordinal associated with the observed wrapper's live readiness generation. Checked preflight, final Couch check, Pair admission, source exemptions, and the stateful double. TestRecoveryNativeOwnerOldWrapperCannotBorrowNewLaunch passes at head and fails when CurrentLaunch receives the entire tag ledger again.
          round: 5
        - id: BR-7
          disposition: addressed
          note: Unsafe Enter, mouse, Relaunch item, and restart shortcut share the recovery preview route. Production reducer tests pass; disabling the shared routing predicate fails all four cases. Healthy relaunch retains its existing confirmation.
          round: 5
      findings:
        - id: BR-8
          severity: Important
          title: Recovery-specific native parsing is outside live conformance
          detail: 'ARCH-MOCK: cmd/internal/sessioninventory/recovery_candidate.go:192 introduces eligibility based on Claude entrypoint and related provenance, but conformance.go:52 checks only the existing scanners; conformance_live_test.go:21 never invokes the recovery parser or collector. Native changes can therefore leave conformance green while recovery eligibility or tail evidence stops working. Extend the existing read-only conformance path through the production recovery parser, covering provenance, cwd, activity and authored tails, with explicit unsupported/absent-sample outcomes and the existing contract-change/monthly cadence.'
          family: external-contract-conformance
          round: 5
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 6
      timestamp: "2026-10-13T13:49:02-07:00"
      dispatched: "2026-10-13T13:44:33-07:00"
      agent: codex
      dispose:
        - id: BR-8
          disposition: addressed
          note: conformance_live_test.go:24 invokes RunRecoveryConformance, which exercises bounded production parsing and checks provenance, cwd, activity, and authored tails. Removing the provenance rejection at conformance.go:281 made the Codex drift and Claude missing/changed-entrypoint regressions fail. Unsupported and absent samples report skip explicitly; atlas/session-identity.md preserves contract-change/monthly cadence.
          round: 6
        - id: BR-6
          disposition: addressed
          note: recovery_native_owner.go resolves the readiness-proved launch ordinal only after associating the exact live agent generation with its wrapper. Exact-generation and source-exemption regressions passed.
          round: 6
        - id: BR-7
          disposition: addressed
          note: Unsafe primary activation and relaunch routes use the recovery chooser. TestReviewNeedsRelaunchPrimaryOpensRecovery and the targeted UI routing tests passed.
          round: 6
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      reviewed: 81a4fc7d5a62180663d7b84245c0c985fedbec21
      blocked: false
    - "n": 7
      timestamp: "2026-10-13T21:42:34-07:00"
      dispatched: "2026-10-13T21:37:49-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: The approved replacement contract distinguishes positive listing from failed observation. TestListedLiveIsContinueAuthority and TestAttributedOwnerCannotResumeDisk pass; unknown observations do not authorize Continue.
          round: 7
        - id: BR-2
          disposition: addressed
          note: Exact agent-generation comparison remains in terminal_agent.go. TestTerminalAgentRequiresExactServerAndLiveDescendant passes and fails when that comparison is removed.
          round: 7
        - id: BR-3
          disposition: addressed
          note: Policy reads were removed from admission. Remaining readiness reads in session_handoff_os.go use contextual ReadRegularMetadata; bounded metadata regressions pass.
          round: 7
        - id: BR-4
          disposition: addressed
          note: session_lock.go preserves the 256-stripe bound and policy publication is removed. TestTerminalLeasePathsHaveFixedBound passes and fails when full socket hashes replace stripe keys.
          round: 7
        - id: BR-5
          disposition: addressed
          note: Admission-config reads were removed. Capture-witness reads still use ReadRegularMetadata in recovery_evidence_os.go:141; bounded metadata and wrong-witness regressions pass.
          round: 7
      findings:
        - id: BR-9
          severity: Critical
          title: Continue is offered where the resume executor still refuses attachment
          detail: 'actionableinventory.go:494 now projects any positively listed session as Continue, but ResumeContextWith calls DecideResume, whose resume.go:186 preconditions still reject missing/incomplete profiles, unsupported saved agents, and unavailable working paths. A temporary review regression reproduced all four refusals. This is the 2nd finding in family recovery-primary-action-routing: state and enforce the rule that every offered Continue reaches a warm attach using listed-session authority; sweep all downstream warm-path gates rather than patching one profile check. ARCH-PURPOSE.'
          family: recovery-primary-action-routing
          round: 7
        - id: BR-10
          severity: Important
          title: README update appears missing for changed Couch startup and action semantics
          detail: startup.go:73 removes automatic launch/resume on ordinary Couch startup, and menu.go:1446 changes visible action labels. README.md:389 remains unchanged throughout the pinned range. Document observation-only startup, explicit action selection, and the M1 recovery limitations alongside the existing usage instructions; atlas/couch.md already describes these changes.
          family: user-facing-contract-documentation
          round: 7
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#528 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T11:01:13-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `preserve-observation-uncertainty` Unknown terminal observations become live or actionable relaunch verdicts
  cmd/internal/couchtty/menu_reattach.go:373 promotes ReasonUnknown to ThreadLive whenever an attached-pass entry exists. cmd/internal/launcher/session_handoff_os.go:73-97 also collapses inspection errors into false usability, which cmd/internal/couchcore/agent_usability.go:7 converts to needs-relaunch. Preserve typed usable/unsafe/unknown outcomes through observers, classifiers, and overlays; test failed refresh after successful attachment and failed metadata inspection. ARCH-ORDER, ARCH-PURPOSE.
- **BR-2** [Critical] `exact-process-generation-authority` Agent usability accepts a recycled agent PID beneath the same server
  cmd/internal/launcher/terminal_agent.go:7-34 compares only the server generation; cmd/internal/launcher/session_handoff_os.go:205 supplies a readiness PID without its expected start identity. A replacement descendant reusing that PID passes stale readiness. Persist and validate the agent generation, with a regression changing only the agent identity while preserving PID and ancestry. ARCH-SECURE.
- **BR-3** [Important] `bounded-untrusted-artifact-reads` Policy and readiness reads can defeat the admission timeout
  cmd/internal/launcher/session_handoff_os.go:127 opens policy files without regular-file/nonblocking safeguards, and readiness reads at lines 193 and 423 are unbounded. A FIFO can hang refresh or admission indefinitely because cancellation cannot interrupt these reads and the coordinator joins the observer. Share the cold-check branch's bounded regular-file reader across these consumers and test FIFO and oversized inputs. ARCH-CONSTRAINTS, ARCH-SECURE.
- **BR-4** [Important] `durable-artifact-lifecycle-bound` Per-terminal coordination files accumulate without an aggregate lifecycle bound
  cmd/internal/artifactpath/session_policy.go:151-165 allocates one lock pathname per canonical socket, while cmd/internal/artifactpath/manifest.go:1460 permanently protects every such inode. Define a bounded coordination scheme or safe retirement protocol without unlinking active or awaited locks; document and measure any intentional permanent per-session cost. ARCH-FUNERAL.

## Round 2 — 2026-10-13T11:34:12-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Typed uncertainty reaches core projection and the attached-pass overlay. Focused tests pass; removing the unknown-state overlay guard or converting inspection errors to unsafe makes the respective regressions fail.
- BR-2 — addressed — Readiness records carry agent start identity and AgentBelongsToServer validates it. Removing the agent-identity comparison fails the unchanged-PID/ancestry regression.
- BR-3 — addressed — Policy and readiness consumers use readBoundedRegularMetadata. Its regressions pass; replacing it with os.ReadFile fails FIFO, oversized-input, symlink and cancellation cases. Additional consumers missed by the family sweep are reported separately below.
- BR-4 — addressed — TerminalLease now selects among 256 permanent stripes. The 10,000-socket regression passes and fails when full-hash per-socket lock paths are restored.

### Raised

- **BR-5** [Important] `bounded-untrusted-artifact-reads` Capture-witness and admission-config reads still permit indefinite blocking
  cmd/internal/storagegc/capture_metadata.go:142 opens events with blocking os.Open before its byte limit; a FIFO without a writer blocks capture preservation indefinitely, reproduced by a disposable regression probe. cmd/internal/launcher/session_handoff_os.go:468 also reads config.kdl with unbounded, non-cancellable os.ReadFile, so admission can remain stuck beyond its timeout while the coordinator waits for the observer. This is the 2nd finding in family bounded-untrusted-artifact-reads. Do NOT fix only these instances: enumerate every persisted-input read added by M1 and enforce one bounded, regular-file, nonblocking, cancellation-aware rule through a shared IO owner, with consumer-level FIFO and oversized-input regressions. ARCH-CONSTRAINTS, ARCH-SECURE, ARCH-PURPOSE.

## Round 3 — 2026-10-13T12:06:16-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Prior disposition retained; unknown-observation and reattach-overlay regressions pass.
- BR-2 — addressed — Prior disposition retained; AgentBelongsToServer checks both server and agent process generations.
- BR-3 — addressed — Prior disposition retained; policy and readiness consumers use the shared bounded reader, with passing FIFO regressions.
- BR-4 — addressed — Prior disposition retained; permanent coordination files remain bounded to 256 stripes, with passing lease tests.
- BR-5 — addressed — Enumerated metadata consumers use osfs.ReadRegularMetadata. Consumer FIFO, oversized-input and cancellation tests pass; substituting os.ReadFile in a disposable checkout makes the regressions fail. Both disposable checkouts were removed.

## Round 4 — 2026-10-13T13:26:16-07:00 (codex) — BLOCKED

### Raised

- **BR-6** [Critical] `exact-process-generation-authority` Native ownership substitutes the latest tag-level launch for the live wrapper's launch
  cmd/internal/launcher/recovery_native_owner.go:85 uses CurrentLaunch(scope, tag, agent) without associating that launch with the observed wrapper PID/start identity. A regression retaining the old wrapper while appending a newer launch targeting another conversation incorrectly returns success. This is the 2nd finding in family `exact-process-generation-authority`. Earlier rounds fixed instances. Do NOT fix only this instance: state and enforce the rule that every native-ownership assertion must bind to the exact observed process generation; sweep preflight, Pair admission, source exemptions, and their doubles. Missing association must remain unresolved. ARCH-SECURE, ARCH-ORDER, ARCH-MOCK.
- **BR-7** [Critical] `recovery-primary-action-routing` The advertised primary Relaunch action never opens the new recovery preview
  cmd/internal/couchtty/menu.go:864 handles only the new resume-candidates item, appended separately at menu_switchagent.go:204. Unsafe-row Enter/mouse still select relaunch at menu.go:803 and open the old confirmation through menu_slot.go:34; the old executor requires established binding. The primary-route regression fails with the confirmation frame and no preview effect. Route Enter, mouse, and the unsafe-row Relaunch item through the same evidence preview while preserving ordinary healthy relaunch behavior. ARCH-PURPOSE.

## Round 5 — 2026-10-13T13:39:34-07:00 (codex) — BLOCKED

### Disposed

- BR-6 — addressed — Ownership now resolves the launch ordinal associated with the observed wrapper's live readiness generation. Checked preflight, final Couch check, Pair admission, source exemptions, and the stateful double. TestRecoveryNativeOwnerOldWrapperCannotBorrowNewLaunch passes at head and fails when CurrentLaunch receives the entire tag ledger again.
- BR-7 — addressed — Unsafe Enter, mouse, Relaunch item, and restart shortcut share the recovery preview route. Production reducer tests pass; disabling the shared routing predicate fails all four cases. Healthy relaunch retains its existing confirmation.

### Raised

- **BR-8** [Important] `external-contract-conformance` Recovery-specific native parsing is outside live conformance
  ARCH-MOCK: cmd/internal/sessioninventory/recovery_candidate.go:192 introduces eligibility based on Claude entrypoint and related provenance, but conformance.go:52 checks only the existing scanners; conformance_live_test.go:21 never invokes the recovery parser or collector. Native changes can therefore leave conformance green while recovery eligibility or tail evidence stops working. Extend the existing read-only conformance path through the production recovery parser, covering provenance, cwd, activity and authored tails, with explicit unsupported/absent-sample outcomes and the existing contract-change/monthly cadence.

## Round 6 — 2026-10-13T13:49:02-07:00 (codex) — passed

### Disposed

- BR-8 — addressed — conformance_live_test.go:24 invokes RunRecoveryConformance, which exercises bounded production parsing and checks provenance, cwd, activity, and authored tails. Removing the provenance rejection at conformance.go:281 made the Codex drift and Claude missing/changed-entrypoint regressions fail. Unsupported and absent samples report skip explicitly; atlas/session-identity.md preserves contract-change/monthly cadence.
- BR-6 — addressed — recovery_native_owner.go resolves the readiness-proved launch ordinal only after associating the exact live agent generation with its wrapper. Exact-generation and source-exemption regressions passed.
- BR-7 — addressed — Unsafe primary activation and relaunch routes use the recovery chooser. TestReviewNeedsRelaunchPrimaryOpensRecovery and the targeted UI routing tests passed.

## Round 7 — 2026-10-13T21:42:34-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The approved replacement contract distinguishes positive listing from failed observation. TestListedLiveIsContinueAuthority and TestAttributedOwnerCannotResumeDisk pass; unknown observations do not authorize Continue.
- BR-2 — addressed — Exact agent-generation comparison remains in terminal_agent.go. TestTerminalAgentRequiresExactServerAndLiveDescendant passes and fails when that comparison is removed.
- BR-3 — addressed — Policy reads were removed from admission. Remaining readiness reads in session_handoff_os.go use contextual ReadRegularMetadata; bounded metadata regressions pass.
- BR-4 — addressed — session_lock.go preserves the 256-stripe bound and policy publication is removed. TestTerminalLeasePathsHaveFixedBound passes and fails when full socket hashes replace stripe keys.
- BR-5 — addressed — Admission-config reads were removed. Capture-witness reads still use ReadRegularMetadata in recovery_evidence_os.go:141; bounded metadata and wrong-witness regressions pass.

### Raised

- **BR-9** [Critical] `recovery-primary-action-routing` Continue is offered where the resume executor still refuses attachment
  actionableinventory.go:494 now projects any positively listed session as Continue, but ResumeContextWith calls DecideResume, whose resume.go:186 preconditions still reject missing/incomplete profiles, unsupported saved agents, and unavailable working paths. A temporary review regression reproduced all four refusals. This is the 2nd finding in family recovery-primary-action-routing: state and enforce the rule that every offered Continue reaches a warm attach using listed-session authority; sweep all downstream warm-path gates rather than patching one profile check. ARCH-PURPOSE.
- **BR-10** [Important] `user-facing-contract-documentation` README update appears missing for changed Couch startup and action semantics
  startup.go:73 removes automatic launch/resume on ordinary Couch startup, and menu.go:1446 changes visible action labels. README.md:389 remains unchanged throughout the pinned range. Document observation-only startup, explicit action selection, and the M1 recovery limitations alongside the existing usage instructions; atlas/couch.md already describes these changes.

## Open findings

- **BR-9** [Critical] `recovery-primary-action-routing` Continue is offered where the resume executor still refuses attachment
- **BR-10** [Important] `user-facing-contract-documentation` README update appears missing for changed Couch startup and action semantics
