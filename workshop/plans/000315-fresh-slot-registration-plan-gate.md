---
gate: plan-quality
issue: 315
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-23T21:09:57-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: The active Spec contradicts the superseding ordinary-launch design
          detail: Lines 22 and 49 describe the explicit RegisterFreshCouchThread reserved-claim protocol, while lines 26-27 and the latest revision at lines 71-81 require removing it and using ordinary StartSpawn registration. Reconcile the Spec, Done-when, Plan, and atlas before implementation; otherwise an implementer cannot know which registration and nonce contract is authoritative. The current code still uses RegisterFreshCouchThread at cmd/internal/launcher/createflow.go:512-519, and atlas/couch.md:1961-1975 documents the superseded behavior.
          family: plan-spec-reconciliation
          round: 1
        - id: PQ-2
          severity: Important
          title: The Plan does not name test functions or mechanical guards for the risky seams
          detail: The three checklist rows only say “extend the ordinary-launch integration test” and “verify ... regressions”; they do not identify the functions to test or give one adversarial-input/mechanical-guard strategy per risky function. Name the production seams, at minimum StartFreshSlot, the fresh-launch routing in runOnce/RunLaunch, BuildCouchLaunchProfile or its replacement, and ValidateFreshAgentArgs, with strategies covering reserved/stale identity, malformed restoration arguments, live-owner refusal, failed launch, and preservation of current metadata/history/preferences. Keep this as strategy lines rather than a prose case inventory.
          family: test-strategy-contract
          round: 1
        - id: PQ-3
          severity: Important
          title: Documentation and generated-surface updates are underspecified and currently stale
          detail: “update docs and publish” at issue:000315-fresh-slot-registration.md:35 does not name atlas/couch.md or specify which retired vocabulary must be removed. That atlas currently asserts RegisterFreshCouchThread and launch_nonce are the supported fresh-slot contract at atlas/couch.md:1961-1975. The plan must name the documentation/atlas sweep and acceptance check so ARCH-PURPOSE is fulfilled across every consumer of the lifecycle contract.
          family: public-contract-sweep
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-23T21:12:18-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: The active Spec and Done-when now consistently require ordinary StartSpawn registration and removal of the superseded fresh-registration/nonce contract.
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: The Plan names several functions, but still lacks one strategy line per risky seam with an adversarial input class and mechanical guard, including fresh routing, malformed arguments, failed launch, and metadata preservation.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: The Plan now names atlas/couch.md and explicitly requires scanning and removing retired RegisterFreshCouchThread, LaunchNonce, and launch_nonce vocabulary.
          round: 2
      findings:
        - id: PQ-4
          severity: Important
          title: The ordinary-launch replacement transition and failure semantics are not specified
          detail: StartFreshSlot currently replaces slot metadata at slotrecovery.go:334 before workspace preparation and launch, while launch_existing.go:169-180 selects distinct Fresh versus ordinary registration paths. The Plan must state the state/event ordering and expected recovery for claim failure, current replacement failure, helper launch failure, acknowledgement failure, registration timeout, and concurrent mutation; “keep ... tests as guards” is not an executable transition contract.
          family: lifecycle-transition-contract
          round: 2
        - id: PQ-5
          severity: Important
          title: '“Restore #315-only edits” does not define a safe mechanical revert boundary'
          detail: 'The Plan names broad files but not exact symbols, hunks, or a commit boundary. Those files contain behavior that must remain for #313 and switch/checkpoint flows; define the #315-owned surface and an acceptance check proving unrelated behavior remains intact before implementation.'
          family: reversion-scope-boundary
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-23T21:16:17-07:00"
      agent: codex
      dispose:
        - id: PQ-2
          disposition: addressed
          note: The plan now names the production seams and provides adversarial-input/mechanical-guard strategies; a separate unresolved test-name issue remains below.
          round: 3
        - id: PQ-4
          disposition: addressed
          note: The explicit ordering and failure table specifies claim, replacement, launch, acknowledgement, registration, concurrency, and retry behavior.
          round: 3
        - id: PQ-5
          disposition: addressed
          note: The plan now pins removal to commits c58e7944 and 0879c270, baseline 54aeb96d, exact files, deletion targets, and diff acceptance checks.
          round: 3
      findings:
        - id: PQ-6
          severity: Important
          title: 'This is the 2nd finding in family `test-strategy-contract`: the owner-admission guard still names no executable test'
          detail: The strategy at issue:000315-fresh-slot-registration.md:71 says `TestOpenSlotRefusesMissingCurrentAndFreshRefusesLiveOrUnknown (use actual existing test name in code)`, but no such function exists in the current tree. Resolve the actual existing function name or specify the exact test to add, with its guard proving live/unknown ownership prevents spawn. The family has 2 findings across 2 rounds; fix the rule by requiring every named strategy to resolve to an existing or explicitly-created test symbol.
          family: test-strategy-contract
          round: 3
        - id: PQ-7
          severity: Important
          title: The full-flow plan omits explicit operating-envelope and trust-boundary contracts
          detail: Under ARCH-CONSTRAINTS and ARCH-SECURE, state the startup interaction's existing timeout/retry bounds and bounded behavior, plus how truncated, malformed, hand-edited, or older claim/profile data is parsed and rejected before mutation. The plan currently asserts real claim storage and malformed restoration coverage, but does not define those contracts or their acceptance guards.
          family: architecture-boundary-contract
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-23T21:19:41-07:00"
      agent: codex
      dispose:
        - id: PQ-6
          disposition: addressed
          note: The owner-admission strategy now names existing executable tests, including TestSlotOpenDoesNotFreshOnLostRecordAndFreshRefusesUncertainty and TestSlotFreshRechecksOldOwnershipAfterReadiness.
          round: 4
        - id: PQ-7
          disposition: addressed
          note: The plan now specifies startup bounds, cancellation behavior, strict parsing, version rejection, and acceptance guards for malformed or untrusted claim/profile data.
          round: 4
      findings:
        - id: PQ-8
          severity: Important
          title: 'This is the 2nd finding in family `reversion-scope-boundary`: the baseline-diff acceptance contradicts the intentional launch_existing.go change'
          detail: The plan says to restore `cmd/internal/couchcore/launch_existing.go` and requires `git diff 54aeb96d -- cmd/internal/launcher cmd/internal/couchcore/launch_existing.go` to be empty at lines 77-80, but the stated implementation must remove the Fresh registration branch in `launch_existing.go:169-175` and therefore necessarily leave a diff. State the general rule that baseline checks apply only to revert-only files, and explicitly exclude intentional StartFreshSlot/ordinary-registration hunks from the zero-diff assertion.
          family: reversion-scope-boundary
          round: 4
      blocked: false
content_hash: 79367b19f0ae977d6535c7c8bbc83bdde97183d5115431c6d0a21c54ae3c1c57
---

# Gate ledger — pair#315 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-23T21:09:57-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `plan-spec-reconciliation` The active Spec contradicts the superseding ordinary-launch design
  Lines 22 and 49 describe the explicit RegisterFreshCouchThread reserved-claim protocol, while lines 26-27 and the latest revision at lines 71-81 require removing it and using ordinary StartSpawn registration. Reconcile the Spec, Done-when, Plan, and atlas before implementation; otherwise an implementer cannot know which registration and nonce contract is authoritative. The current code still uses RegisterFreshCouchThread at cmd/internal/launcher/createflow.go:512-519, and atlas/couch.md:1961-1975 documents the superseded behavior.
- **PQ-2** [Important] `test-strategy-contract` The Plan does not name test functions or mechanical guards for the risky seams
  The three checklist rows only say “extend the ordinary-launch integration test” and “verify ... regressions”; they do not identify the functions to test or give one adversarial-input/mechanical-guard strategy per risky function. Name the production seams, at minimum StartFreshSlot, the fresh-launch routing in runOnce/RunLaunch, BuildCouchLaunchProfile or its replacement, and ValidateFreshAgentArgs, with strategies covering reserved/stale identity, malformed restoration arguments, live-owner refusal, failed launch, and preservation of current metadata/history/preferences. Keep this as strategy lines rather than a prose case inventory.
- **PQ-3** [Important] `public-contract-sweep` Documentation and generated-surface updates are underspecified and currently stale
  “update docs and publish” at issue:000315-fresh-slot-registration.md:35 does not name atlas/couch.md or specify which retired vocabulary must be removed. That atlas currently asserts RegisterFreshCouchThread and launch_nonce are the supported fresh-slot contract at atlas/couch.md:1961-1975. The plan must name the documentation/atlas sweep and acceptance check so ARCH-PURPOSE is fulfilled across every consumer of the lifecycle contract.

## Round 2 — 2026-09-23T21:12:18-07:00 (codex) — BLOCKED

### Disposed

- PQ-1 — addressed — The active Spec and Done-when now consistently require ordinary StartSpawn registration and removal of the superseded fresh-registration/nonce contract.
- PQ-2 — not-addressed — The Plan names several functions, but still lacks one strategy line per risky seam with an adversarial input class and mechanical guard, including fresh routing, malformed arguments, failed launch, and metadata preservation.
- PQ-3 — addressed — The Plan now names atlas/couch.md and explicitly requires scanning and removing retired RegisterFreshCouchThread, LaunchNonce, and launch_nonce vocabulary.

### Raised

- **PQ-4** [Important] `lifecycle-transition-contract` The ordinary-launch replacement transition and failure semantics are not specified
  StartFreshSlot currently replaces slot metadata at slotrecovery.go:334 before workspace preparation and launch, while launch_existing.go:169-180 selects distinct Fresh versus ordinary registration paths. The Plan must state the state/event ordering and expected recovery for claim failure, current replacement failure, helper launch failure, acknowledgement failure, registration timeout, and concurrent mutation; “keep ... tests as guards” is not an executable transition contract.
- **PQ-5** [Important] `reversion-scope-boundary` “Restore #315-only edits” does not define a safe mechanical revert boundary
  The Plan names broad files but not exact symbols, hunks, or a commit boundary. Those files contain behavior that must remain for #313 and switch/checkpoint flows; define the #315-owned surface and an acceptance check proving unrelated behavior remains intact before implementation.

## Round 3 — 2026-09-23T21:16:17-07:00 (codex) — BLOCKED

### Disposed

- PQ-2 — addressed — The plan now names the production seams and provides adversarial-input/mechanical-guard strategies; a separate unresolved test-name issue remains below.
- PQ-4 — addressed — The explicit ordering and failure table specifies claim, replacement, launch, acknowledgement, registration, concurrency, and retry behavior.
- PQ-5 — addressed — The plan now pins removal to commits c58e7944 and 0879c270, baseline 54aeb96d, exact files, deletion targets, and diff acceptance checks.

### Raised

- **PQ-6** [Important] `test-strategy-contract` This is the 2nd finding in family `test-strategy-contract`: the owner-admission guard still names no executable test
  The strategy at issue:000315-fresh-slot-registration.md:71 says `TestOpenSlotRefusesMissingCurrentAndFreshRefusesLiveOrUnknown (use actual existing test name in code)`, but no such function exists in the current tree. Resolve the actual existing function name or specify the exact test to add, with its guard proving live/unknown ownership prevents spawn. The family has 2 findings across 2 rounds; fix the rule by requiring every named strategy to resolve to an existing or explicitly-created test symbol.
- **PQ-7** [Important] `architecture-boundary-contract` The full-flow plan omits explicit operating-envelope and trust-boundary contracts
  Under ARCH-CONSTRAINTS and ARCH-SECURE, state the startup interaction's existing timeout/retry bounds and bounded behavior, plus how truncated, malformed, hand-edited, or older claim/profile data is parsed and rejected before mutation. The plan currently asserts real claim storage and malformed restoration coverage, but does not define those contracts or their acceptance guards.

## Round 4 — 2026-09-23T21:19:41-07:00 (codex) — passed

### Disposed

- PQ-6 — addressed — The owner-admission strategy now names existing executable tests, including TestSlotOpenDoesNotFreshOnLostRecordAndFreshRefusesUncertainty and TestSlotFreshRechecksOldOwnershipAfterReadiness.
- PQ-7 — addressed — The plan now specifies startup bounds, cancellation behavior, strict parsing, version rejection, and acceptance guards for malformed or untrusted claim/profile data.

### Raised

- **PQ-8** [Important] `reversion-scope-boundary` This is the 2nd finding in family `reversion-scope-boundary`: the baseline-diff acceptance contradicts the intentional launch_existing.go change
  The plan says to restore `cmd/internal/couchcore/launch_existing.go` and requires `git diff 54aeb96d -- cmd/internal/launcher cmd/internal/couchcore/launch_existing.go` to be empty at lines 77-80, but the stated implementation must remove the Fresh registration branch in `launch_existing.go:169-175` and therefore necessarily leave a diff. State the general rule that baseline checks apply only to revert-only files, and explicitly exclude intentional StartFreshSlot/ordinary-registration hunks from the zero-diff assertion.

## Open findings

- **PQ-8** [Important] `reversion-scope-boundary` This is the 2nd finding in family `reversion-scope-boundary`: the baseline-diff acceptance contradicts the intentional launch_existing.go change
