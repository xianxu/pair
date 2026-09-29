---
gate: boundary-review
issue: 346
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-29T10:36:31-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Operator-registry sentinel in couchcmd stale_store_test is unreachable, so isolation is unproven
          detail: The sentinel is written at operatorRoot/stores.json, not .retention/stores.json, and the env loop immediately blanks PAIR_DATA_DIR, so no code path can modify it. Build an operator-shaped default root with a real .retention/stores.json sentinel and run the production path (ideally a subprocess with a polluted inherited environment). The ticked smoke-isolation plan item has no matching change to tests/couch-recovery-smoke.sh.
          family: test-oracle-cannot-fail
          round: 1
        - id: BR-2
          severity: Important
          title: README does not document pair gc --forget-missing-store
          detail: README.md around lines 812-816 lists the pair gc store flags. Add the new abandonment flag, its migration reset, the --complete-migration re-acknowledgment, and remounting as the alternative for a temporarily unavailable store.
          family: readme-surface-gap
          round: 1
        - id: BR-3
          severity: Important
          title: pair gc gives no recovery guidance or store list when a registered store is unavailable
          detail: gccmd/run.go:143 hides the registered-store list when ReadRegistry fails, and the collector block reason is a raw lstat error. Print the structurally valid inventory with unavailable stores marked, and name both next actions (restore/remount, or --forget-missing-store with the exact path) to meet the Spec's "communicates recovery requirements".
          family: outage-diagnostic-actionability
          round: 1
        - id: BR-4
          severity: Minor
          title: UnregisterStore still requires every unrelated store to be available
          detail: stores.go:187 uses the full ReadRegistry, so removing an intact store with an empty-store proof is blocked by an unrelated outage. This is safe but inconsistent with the structure/availability split.
          family: availability-split-consistency
          round: 1
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-09-29T10:43:03-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Sentinel now at operatorRoot/.retention/stores.json with real subprocess via tests/with-isolated-pair.sh; mutating the wrapper to leak roots turned the test red (verified in a scratch copy), restored passes; smoke script now execs through the wrapper.
          round: 2
        - id: BR-2
          disposition: addressed
          note: README.md 818-824 documents --forget-missing-store, the migration reset, re-acknowledgment with --complete-migration --store, and restore/remount for temporary outages.
          round: 2
        - id: BR-3
          disposition: addressed
          note: gccmd/run.go now lists stores via InspectRegistry with unavailable markers plus restore/remount/forget guidance; TestMissingStorePreviewExplainsRecovery pins it (the list was hidden before, so it would have failed).
          round: 2
        - id: BR-4
          disposition: addressed
          note: Deliberately kept conservative; rationale documented in the UnregisterStore comment (stores.go ~181) and the issue Log. It is safe and the explicit abandonment path covers missing stores.
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: false
    - "n": 3
      timestamp: "2026-09-29T11:11:01-07:00"
      agent: claude
      findings:
        - id: BR-5
          severity: Important
          title: Unmaterialized Pair-chosen session ID is returned as a resume target
          detail: QueryResumeTarget (query.go:449) returns a chosen-id RequestedNativeID as Provisional before any native file exists, and createflow.go:960 dropped the AgentSessionExists check and fresh fallback. A fresh Claude/Qoder thread quit before its first message gets resumed with --resume X for a conversation that does not exist. Admit an unconfirmed chosen-id only after a metadata-only filename check; otherwise start fresh. Add a production-boundary test.
          family: unconfirmed-identity-admitted-as-resumable
          round: 3
        - id: BR-6
          severity: Important
          title: session-repair re-implements binding resolution instead of reusing ResolveBindings
          detail: recover.go:87 requires exactly one root across the union of all rounds, while the live watcher uses ResolveBindings (authorized filter plus per-group intersection). The two rules diverge, e.g. groups {D} and {D,E}. Call ResolveBindings and accept a unique Provisional root.
          family: correlation-rule-single-source
          round: 3
        - id: BR-7
          severity: Important
          title: pair session-repair operator CLI is missing from README
          detail: '2nd finding in this family. Rule: every dispatcher family an operator is expected to run must appear in README or be explicitly marked internal. Enforce it with a test walking dispatcher.Families() against README with an internal allowlist, not a one-off line.'
          family: readme-surface-gap
          round: 3
        - id: BR-8
          severity: Minor
          title: Watcher legacy proof migration also fires for v3 bindings that have no proof
          detail: run.go:101 lacks the Version < 3 guard that QuerySession has. On watcher restart it parses the native body, appends a v2 row that hides ConfirmationReason, and skips Codex lifecycle following.
          family: legacy-path-version-guard
          round: 3
        - id: BR-9
          severity: Minor
          title: decideAutomaticResumeConfig quarantine can no longer trigger
          detail: hasResumable is now always true when a session ID is set, so the Codex quarantine branch is dead.
          family: dead-code-after-policy-change
          round: 3
        - id: BR-10
          severity: Minor
          title: ConfirmationReason is untyped string literals, unlike RequestOrigin
          family: typed-enum-for-persisted-vocabulary
          round: 3
        - id: BR-11
          severity: Minor
          title: requested_native_id is not shape-validated before becoming a --resume argv value
          family: untrusted-id-to-argv
          round: 3
        - id: BR-12
          severity: Minor
          title: Watcher in-memory phases are separate fields rather than a tagged state enum
          detail: observationEpoch, boundRootNodeID and trackedTargets together encode unknown-baseline, epoch, handshake, correlation, confirmed and lifecycle phases without a written transition set (ARCH-ORDER).
          family: implicit-watcher-state
          round: 3
      boundary: M2
      recipe: milestone-review
      blocked: true
    - "n": 4
      timestamp: "2026-09-29T11:26:47-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: not-addressed
          note: Intentionally unchanged pending operator decision (issue Log 2026-09-29); QueryResumeTarget still admits an unmaterialized chosen-id and runConfigPicker has no existence check. Resolve by recorded operator Revision (then withdraw) or add the metadata-only gate plus a boundary test.
          round: 4
        - id: BR-6
          disposition: addressed
          note: recover.go recoveryRoot now calls ResolveBindings; TestRepairUsesSameRoundIntersectionAsLiveWatcher pins the {D},{D,E} divergence the union rule got wrong.
          round: 4
        - id: BR-7
          disposition: addressed
          note: README documents session-repair and dispatcher/readme_test.go enforces the family-level rule with an internal allowlist and stale-entry check.
          round: 4
        - id: BR-8
          disposition: addressed
          note: Version<3 guard at run.go call site and in migrateProoflessBinding; TestRestartedV3ProoflessWatcherPreservesConfirmationAndFollowsLifecycle asserts no ledger writes and lifecycle followed.
          round: 4
        - id: BR-9
          disposition: addressed
          note: decideAutomaticResumeConfig and its dead quarantine branch removed along with its test.
          round: 4
        - id: BR-10
          disposition: addressed
          note: ConfirmationReason is a typed enum validated in validateRecord; all call sites use the constants.
          round: 4
        - id: BR-11
          disposition: not-addressed
          note: RequestedNativeID is validated, but the sibling binding RootNativeID that also reaches --resume via ResumeTargetForLaunch is not; apply the same argv-safety predicate to every ledger ID that can become argv.
          round: 4
        - id: BR-12
          disposition: not-addressed
          note: Only a descriptive comment was added at run.go:88; phases remain independent fields. Acceptable to defer as Minor.
          round: 4
      findings:
        - id: BR-13
          severity: Important
          title: HEAD fails TestREADMEDocumentsSessionInventoryContract after the BR-7 README rewrite
          detail: 'The fix commit replaced the README paragraph naming the provisional/established statuses with probation prose, so go test ./cmd/internal/sessioninventory is red (a pure string check, not the sandbox). Restore the status vocabulary next to the probation paragraph. The rule: every boundary commit runs the full non-sandboxed make test before sdlc milestone-close, because a docs-only-looking edit can break an enforced contract test.'
          family: full-suite-before-boundary
          round: 4
      boundary: M2
      recipe: milestone-review
      blocked: true
    - "n": 5
      timestamp: "2026-09-29T13:31:56-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: addressed
          note: ResumeTargetForRuntimeLaunch gates chosen-id on root filename via metadata; FreshRequired honoured by Alt+n, createflow, Couch; tests in resume_target_test.go:84, osruntime_test.go:771, relaunch_test.go:413 fail without it.
          round: 5
        - id: BR-11
          disposition: addressed
          note: record.go:437 applies safeNativeArg to binding RootNativeID too; probation_test.go:92 covers v1-v3 unsafe binding IDs.
          round: 5
        - id: BR-12
          disposition: not-addressed
          note: run.go:84-92 still separate fields with a descriptive comment only; acceptable to defer as Minor.
          round: 5
        - id: BR-13
          disposition: addressed
          note: go test ./cmd/internal/sessioninventory (incl. TestREADMEDocumentsSessionInventoryContract) passes at HEAD, as do the other touched packages unsandboxed.
          round: 5
      findings:
        - id: BR-14
          severity: Important
          title: Chosen-id materialization probe collapses a native listing failure into FreshRequired
          detail: ObserveAgentMetadata skips roots whose ListFiles fails (non-ErrStorageAbsent) or returns partial listings, and ResumeTargetForRuntimeLaunch (query.go:467) ignores the diagnostics and sets FreshRequired, so a transient EACCES/EIO abandons a materialized conversation X for a new UUID (createflow also removes the config). Make the probe tri-state (present / confirmed-absent / unknown); on unknown set neither NativeID nor FreshRequired and keep the provisional refusal; add a fake ListFiles-error test.
          family: failed-probe-treated-as-absence
          round: 5
      boundary: M2
      recipe: milestone-review
      blocked: true
    - "n": 6
      timestamp: "2026-09-29T13:47:01-07:00"
      agent: claude
      dispose:
        - id: BR-12
          disposition: not-addressed
          note: sessionwatch/run.go:84-92 still carries trackedTargets/boundRootNodeID/observationEpoch as independent fields; Minor, deferral previously accepted.
          round: 6
        - id: BR-14
          disposition: addressed
          note: query.go:470-483 now returns neither NativeID nor FreshRequired when any non-absent diagnostic exists; TestChosenTargetListingFailureIsUnknown (failed + partial listing) goes red against the old else-branch; restart/picker/couch refuse before effects (restart_test, osruntime_test, relaunch_test).
          round: 6
      findings:
        - id: BR-15
          severity: Minor
          title: Chosen-ID "materialization unknown" is an implicit field combination re-derived in three places
          detail: 'ResumeTarget encodes unknown as NativeID=="" and !FreshRequired; couchcore/resume.go:377 (redundant diagnostics loop) and launcher/ledger.go:75 (ResumeBlocked) re-derive it, and pure ParseLedger sets ResumeBlocked for every unconfirmed chosen entry. Rule: a state with more than two legal values goes on the producing type as a tagged enum (e.g. Materialization present/absent/unknown) that consumers switch on; apply it to ResumeTarget and to the watcher phases from BR-12.'
          family: implicit-watcher-state
          round: 6
        - id: BR-16
          severity: Minor
          title: A persistent incomplete listing (such as a symlinked project dir) refuses with a "retry" message that never succeeds
          detail: 'A ListingIssuesError from a non-regular entry is permanent, not transient. Rule: every unknown-identity refusal names the failing root or entry and the explicit fresh-start escape hatch (pair restart --new-session or a fresh launch), rather than only telling the operator to retry.'
          family: outage-diagnostic-actionability
          round: 6
      boundary: M2
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#346 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-29T10:36:31-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `test-oracle-cannot-fail` Operator-registry sentinel in couchcmd stale_store_test is unreachable, so isolation is unproven
  The sentinel is written at operatorRoot/stores.json, not .retention/stores.json, and the env loop immediately blanks PAIR_DATA_DIR, so no code path can modify it. Build an operator-shaped default root with a real .retention/stores.json sentinel and run the production path (ideally a subprocess with a polluted inherited environment). The ticked smoke-isolation plan item has no matching change to tests/couch-recovery-smoke.sh.
- **BR-2** [Important] `readme-surface-gap` README does not document pair gc --forget-missing-store
  README.md around lines 812-816 lists the pair gc store flags. Add the new abandonment flag, its migration reset, the --complete-migration re-acknowledgment, and remounting as the alternative for a temporarily unavailable store.
- **BR-3** [Important] `outage-diagnostic-actionability` pair gc gives no recovery guidance or store list when a registered store is unavailable
  gccmd/run.go:143 hides the registered-store list when ReadRegistry fails, and the collector block reason is a raw lstat error. Print the structurally valid inventory with unavailable stores marked, and name both next actions (restore/remount, or --forget-missing-store with the exact path) to meet the Spec's "communicates recovery requirements".
- **BR-4** [Minor] `availability-split-consistency` UnregisterStore still requires every unrelated store to be available
  stores.go:187 uses the full ReadRegistry, so removing an intact store with an empty-store proof is blocked by an unrelated outage. This is safe but inconsistent with the structure/availability split.

## Round 2 — 2026-09-29T10:43:03-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Sentinel now at operatorRoot/.retention/stores.json with real subprocess via tests/with-isolated-pair.sh; mutating the wrapper to leak roots turned the test red (verified in a scratch copy), restored passes; smoke script now execs through the wrapper.
- BR-2 — addressed — README.md 818-824 documents --forget-missing-store, the migration reset, re-acknowledgment with --complete-migration --store, and restore/remount for temporary outages.
- BR-3 — addressed — gccmd/run.go now lists stores via InspectRegistry with unavailable markers plus restore/remount/forget guidance; TestMissingStorePreviewExplainsRecovery pins it (the list was hidden before, so it would have failed).
- BR-4 — addressed — Deliberately kept conservative; rationale documented in the UnregisterStore comment (stores.go ~181) and the issue Log. It is safe and the explicit abandonment path covers missing stores.

## Round 3 — 2026-09-29T11:11:01-07:00 (claude) — BLOCKED

### Raised

- **BR-5** [Important] `unconfirmed-identity-admitted-as-resumable` Unmaterialized Pair-chosen session ID is returned as a resume target
  QueryResumeTarget (query.go:449) returns a chosen-id RequestedNativeID as Provisional before any native file exists, and createflow.go:960 dropped the AgentSessionExists check and fresh fallback. A fresh Claude/Qoder thread quit before its first message gets resumed with --resume X for a conversation that does not exist. Admit an unconfirmed chosen-id only after a metadata-only filename check; otherwise start fresh. Add a production-boundary test.
- **BR-6** [Important] `correlation-rule-single-source` session-repair re-implements binding resolution instead of reusing ResolveBindings
  recover.go:87 requires exactly one root across the union of all rounds, while the live watcher uses ResolveBindings (authorized filter plus per-group intersection). The two rules diverge, e.g. groups {D} and {D,E}. Call ResolveBindings and accept a unique Provisional root.
- **BR-7** [Important] `readme-surface-gap` pair session-repair operator CLI is missing from README
  2nd finding in this family. Rule: every dispatcher family an operator is expected to run must appear in README or be explicitly marked internal. Enforce it with a test walking dispatcher.Families() against README with an internal allowlist, not a one-off line.
- **BR-8** [Minor] `legacy-path-version-guard` Watcher legacy proof migration also fires for v3 bindings that have no proof
  run.go:101 lacks the Version < 3 guard that QuerySession has. On watcher restart it parses the native body, appends a v2 row that hides ConfirmationReason, and skips Codex lifecycle following.
- **BR-9** [Minor] `dead-code-after-policy-change` decideAutomaticResumeConfig quarantine can no longer trigger
  hasResumable is now always true when a session ID is set, so the Codex quarantine branch is dead.
- **BR-10** [Minor] `typed-enum-for-persisted-vocabulary` ConfirmationReason is untyped string literals, unlike RequestOrigin
- **BR-11** [Minor] `untrusted-id-to-argv` requested_native_id is not shape-validated before becoming a --resume argv value
- **BR-12** [Minor] `implicit-watcher-state` Watcher in-memory phases are separate fields rather than a tagged state enum
  observationEpoch, boundRootNodeID and trackedTargets together encode unknown-baseline, epoch, handshake, correlation, confirmed and lifecycle phases without a written transition set (ARCH-ORDER).

## Round 4 — 2026-09-29T11:26:47-07:00 (claude) — BLOCKED

### Disposed

- BR-5 — not-addressed — Intentionally unchanged pending operator decision (issue Log 2026-09-29); QueryResumeTarget still admits an unmaterialized chosen-id and runConfigPicker has no existence check. Resolve by recorded operator Revision (then withdraw) or add the metadata-only gate plus a boundary test.
- BR-6 — addressed — recover.go recoveryRoot now calls ResolveBindings; TestRepairUsesSameRoundIntersectionAsLiveWatcher pins the {D},{D,E} divergence the union rule got wrong.
- BR-7 — addressed — README documents session-repair and dispatcher/readme_test.go enforces the family-level rule with an internal allowlist and stale-entry check.
- BR-8 — addressed — Version<3 guard at run.go call site and in migrateProoflessBinding; TestRestartedV3ProoflessWatcherPreservesConfirmationAndFollowsLifecycle asserts no ledger writes and lifecycle followed.
- BR-9 — addressed — decideAutomaticResumeConfig and its dead quarantine branch removed along with its test.
- BR-10 — addressed — ConfirmationReason is a typed enum validated in validateRecord; all call sites use the constants.
- BR-11 — not-addressed — RequestedNativeID is validated, but the sibling binding RootNativeID that also reaches --resume via ResumeTargetForLaunch is not; apply the same argv-safety predicate to every ledger ID that can become argv.
- BR-12 — not-addressed — Only a descriptive comment was added at run.go:88; phases remain independent fields. Acceptable to defer as Minor.

### Raised

- **BR-13** [Important] `full-suite-before-boundary` HEAD fails TestREADMEDocumentsSessionInventoryContract after the BR-7 README rewrite
  The fix commit replaced the README paragraph naming the provisional/established statuses with probation prose, so go test ./cmd/internal/sessioninventory is red (a pure string check, not the sandbox). Restore the status vocabulary next to the probation paragraph. The rule: every boundary commit runs the full non-sandboxed make test before sdlc milestone-close, because a docs-only-looking edit can break an enforced contract test.

## Round 5 — 2026-09-29T13:31:56-07:00 (claude) — BLOCKED

### Disposed

- BR-5 — addressed — ResumeTargetForRuntimeLaunch gates chosen-id on root filename via metadata; FreshRequired honoured by Alt+n, createflow, Couch; tests in resume_target_test.go:84, osruntime_test.go:771, relaunch_test.go:413 fail without it.
- BR-11 — addressed — record.go:437 applies safeNativeArg to binding RootNativeID too; probation_test.go:92 covers v1-v3 unsafe binding IDs.
- BR-12 — not-addressed — run.go:84-92 still separate fields with a descriptive comment only; acceptable to defer as Minor.
- BR-13 — addressed — go test ./cmd/internal/sessioninventory (incl. TestREADMEDocumentsSessionInventoryContract) passes at HEAD, as do the other touched packages unsandboxed.

### Raised

- **BR-14** [Important] `failed-probe-treated-as-absence` Chosen-id materialization probe collapses a native listing failure into FreshRequired
  ObserveAgentMetadata skips roots whose ListFiles fails (non-ErrStorageAbsent) or returns partial listings, and ResumeTargetForRuntimeLaunch (query.go:467) ignores the diagnostics and sets FreshRequired, so a transient EACCES/EIO abandons a materialized conversation X for a new UUID (createflow also removes the config). Make the probe tri-state (present / confirmed-absent / unknown); on unknown set neither NativeID nor FreshRequired and keep the provisional refusal; add a fake ListFiles-error test.

## Round 6 — 2026-09-29T13:47:01-07:00 (claude) — passed

### Disposed

- BR-12 — not-addressed — sessionwatch/run.go:84-92 still carries trackedTargets/boundRootNodeID/observationEpoch as independent fields; Minor, deferral previously accepted.
- BR-14 — addressed — query.go:470-483 now returns neither NativeID nor FreshRequired when any non-absent diagnostic exists; TestChosenTargetListingFailureIsUnknown (failed + partial listing) goes red against the old else-branch; restart/picker/couch refuse before effects (restart_test, osruntime_test, relaunch_test).

### Raised

- **BR-15** [Minor] `implicit-watcher-state` Chosen-ID "materialization unknown" is an implicit field combination re-derived in three places
  ResumeTarget encodes unknown as NativeID=="" and !FreshRequired; couchcore/resume.go:377 (redundant diagnostics loop) and launcher/ledger.go:75 (ResumeBlocked) re-derive it, and pure ParseLedger sets ResumeBlocked for every unconfirmed chosen entry. Rule: a state with more than two legal values goes on the producing type as a tagged enum (e.g. Materialization present/absent/unknown) that consumers switch on; apply it to ResumeTarget and to the watcher phases from BR-12.
- **BR-16** [Minor] `outage-diagnostic-actionability` A persistent incomplete listing (such as a symlinked project dir) refuses with a "retry" message that never succeeds
  A ListingIssuesError from a non-regular entry is permanent, not transient. Rule: every unknown-identity refusal names the failing root or entry and the explicit fresh-start escape hatch (pair restart --new-session or a fresh launch), rather than only telling the operator to retry.

## Open findings

- **BR-12** [Minor] `implicit-watcher-state` Watcher in-memory phases are separate fields rather than a tagged state enum
- **BR-15** [Minor] `implicit-watcher-state` Chosen-ID "materialization unknown" is an implicit field combination re-derived in three places
- **BR-16** [Minor] `outage-diagnostic-actionability` A persistent incomplete listing (such as a symlinked project dir) refuses with a "retry" message that never succeeds
