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

## Open findings

- **BR-5** [Important] `unconfirmed-identity-admitted-as-resumable` Unmaterialized Pair-chosen session ID is returned as a resume target
- **BR-6** [Important] `correlation-rule-single-source` session-repair re-implements binding resolution instead of reusing ResolveBindings
- **BR-7** [Important] `readme-surface-gap` pair session-repair operator CLI is missing from README
- **BR-8** [Minor] `legacy-path-version-guard` Watcher legacy proof migration also fires for v3 bindings that have no proof
- **BR-9** [Minor] `dead-code-after-policy-change` decideAutomaticResumeConfig quarantine can no longer trigger
- **BR-10** [Minor] `typed-enum-for-persisted-vocabulary` ConfirmationReason is untyped string literals, unlike RequestOrigin
- **BR-11** [Minor] `untrusted-id-to-argv` requested_native_id is not shape-validated before becoming a --resume argv value
- **BR-12** [Minor] `implicit-watcher-state` Watcher in-memory phases are separate fields rather than a tagged state enum
