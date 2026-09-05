---
gate: plan-quality
issue: 181
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-03T17:02:40-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: ThreadEvidence cannot distinguish "proof unresolved" from "proof says no", so a total classifier invents archive-eligible reasons
          detail: |-
            ClassifyThread is total over ThreadEvidence, but the type carries no
            "unresolved" marker for Parked/Detached (only PathError for the path
            case). Task 5 hands it ThreadEvidence{Live: fromProcOps(...)} with no
            parked/detached proof, so a healthy verified park classifies
            ReasonBindingLost and a healthy detach classifies ReasonSessionGone in
            couch --list, permanently. The same defect fires in the switcher:
            actionableinventory.go:285-292 sets detached=nil on a DetachedSessions
            failure and today calls that "not proved this round"; under the new
            classifier that transient failure asserts session-gone on every detached
            row, and session-gone is a reason M3's DecideRetirement acts on.
          family: evidence-absence-vs-negative
          round: 1
        - id: PQ-2
          severity: Important
          title: Shell-derived refusals are ordered ahead of the live branch, so today's live rows can classify unusable
          detail: |-
            ClassifyThread checks PathError, LatestLaunchProfile == nil and
            IsSupportedAgent before the live branch, and the new shell physicalizes
            every record rather than only resume candidates. Today
            actionableinventory.go:237 skips records with incarnations, so a live
            record never reaches c.Path.Physical and actionableThreadState never
            reads LatestLaunchProfile on the live path. After the change, a running
            agent whose working directory was removed becomes unusable/path-missing
            and Task 4 refuses Enter on it; c.Path == nil turns every row unusable
            instead of only suppressing resume candidates. This falsifies the
            plan's own characterization test and M1's "no behaviour change to what
            is actionable".
          family: classifier-branch-order
          round: 1
        - id: PQ-3
          severity: Important
          title: M2 relaxes the binding diagnostic in DecideResume but ResumeContext refuses earlier, at resume.go:212
          detail: |-
            SessionInventoryNativeBindingResolver.ResolveEstablished returns
            refuseResume(...) itself at resume.go:172-175 for a provisional binding,
            and ResumeContext returns that error at resume.go:212-215 before
            DecideResume runs (FakeThreadArtifactCollisionChecker does the same at
            artifactcollision_fake.go:123). Task 8's file list covers
            resume.go:93-126,269-278 and omits :212, so pair-couch-24 still refuses
            with resume-binding-provisional after M2 and its Done-when is
            unreachable.
          family: unnamed-refusal-site
          round: 1
        - id: PQ-4
          severity: Important
          title: Dropping the binding gate on detached rows widens startup auto-selection, and a warm refusal is fatal to couch startup
          detail: |-
            StartInteractive returns resumeErr directly (startup.go:55-58), so a
            resume refusal stops couch in that tree — the incident the comment at
            actionableinventory.go:249-258 says the detached binding gate exists to
            prevent. Task 8 removes that gate and RequireResumeBoundary refuses a
            warm profile that reaches a create boundary (session died between
            projection and launch). The plan never states what startup does with
            that refusal.
          family: startup-selection-widening
          round: 1
        - id: PQ-5
          severity: Important
          title: BenchmarkMenu100 is cited as the cost guard but never exercises the inventory
          detail: |-
            menu_perf_test.go:91 benchmarks NewMenuState/RenderMenu/reduceKey over a
            fixture []ActionableThreadSummary. It never calls
            ActionableThreadInventoryContext, c.Path.Physical, the binding resolver
            or zellij, so it cannot observe the per-record physicalization the plan
            adds or the enlarged detach-candidate set (ARCH-CONSTRAINTS). Name a
            guard that counts resolver/Physical calls against a fake, or state the
            envelope and accept it.
          family: unbacked-existing-behavior-claim
          round: 1
        - id: PQ-6
          severity: Minor
          title: ThreadBusy rows reach Enter and menuActionItems, where !Live() offers switch and resume
          detail: |-
            menu.go:388-401 sets operation="switch" for any non-Resumable row and
            menuActionItems returns {"resume","name","describe"} for any non-Live
            row, so a park-in-flight row gets both. Task 4 specifies only the
            unusable case. Downstream refusals catch it, but the row lies about what
            it offers.
          family: new-state-unhandled-at-consumers
          round: 1
        - id: PQ-7
          severity: Minor
          title: Task 2's migration scope understates the affected files
          detail: |-
            ProjectActionableThreads and ActionableThreadInventory* appear in 14
            files (29 direct calls), not "36 call sites across six files"; the plan
            names four, one of which (artifactpath/deadsymbols_test.go) is outside
            the couch packages.
          family: unbacked-existing-behavior-claim
          round: 1
        - id: PQ-8
          severity: Minor
          title: CurrentLaunch parses a cross-version append-only ledger; Task 9 gives it two hand-written cases and no adversarial class
          detail: |-
            The ledger is written by other processes and older Pair versions and is
            hand-editable (ARCH-SECURE). One strategy line is missing: a property
            test over shuffled, perforated and duplicated record sets, asserting the
            pending-vs-committed invariant, is the mechanical guard the two named
            cases cannot supply.
          family: adversarial-input-class-unnamed
          round: 1
        - id: PQ-9
          severity: Minor
          title: Spec item 5 says archive is reversible "by moving the file back", but Snapshot walks the manifest
          detail: |-
            ThreadStore.Snapshot (threadstore.go:504) enumerates manifest.Threads, so
            a file moved back without a manifest re-add produces no row. Task 12
            Step C gets this right; the Spec sentence should match it.
          family: spec-plan-contract-drift
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-03T17:08:23-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: ProofStatus makes unresolved representable; ReasonUnknown is never archive-eligible; Task 5 now resolves the same proofs instead of stubbing them.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Live branch is ordered ahead of path/profile/agent, with the :237 justification stated inline.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Step 1b names resume.go:212 and states the covering rule (no resolver on the warm path at all), which also reaches the :269 recheck.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: 'Task 8b decides: keep no-fallback, wrap the refusal at the startup seam with an actionable message.'
          round: 2
        - id: PQ-5
          disposition: addressed
          note: BenchmarkMenu100 explicitly retired as the guard; replaced by a resolver/Physical call-counting test.
          round: 2
        - id: PQ-6
          disposition: not-addressed
          note: Task 4 still specifies only the unusable case; ThreadBusy still reaches Resumable()/Live()-keyed Enter and menuActionItems.
          round: 2
        - id: PQ-7
          disposition: not-addressed
          note: Still says "36 call sites across six files"; actual is 14 files. Delete the count rather than correcting it.
          round: 2
        - id: PQ-8
          disposition: addressed
          note: Task 9 Step 2b adds the property test over shuffled, perforated and duplicated record sets.
          round: 2
        - id: PQ-9
          disposition: addressed
          note: Spec item 5 now says "moving the file back and re-adding it to the manifest", matching Task 12 Step C.
          round: 2
      findings:
        - id: PQ-10
          severity: Minor
          title: Task 2 states the per-record Physical cost three ways and the three disagree
          detail: |-
            Step 3's snippet calls c.Path.Physical unconditionally per record; Step 5's
            guard asserts PhysicalCalls()==2 over a six-record fixture; the cost-discipline
            paragraph describes a third, lazier shape. 2nd in family: the rule, not the
            instance, is that when the plan states one contract in prose, snippet and guard
            test, the guard test binds and the other two are reconciled to it in the same edit.
          family: spec-plan-contract-drift
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-09-03T17:12:01-07:00"
      agent: claude
      dispose:
        - id: PQ-6
          disposition: not-addressed
          note: Task 4 still scopes only the unusable case; ThreadBusy is absent from menu.go routing.
          round: 3
        - id: PQ-7
          disposition: not-addressed
          note: plan:464 and issue:161,192 still say 36 across six files; measured 26 across six, and both named outlier files have zero calls.
          round: 3
        - id: PQ-10
          disposition: addressed
          note: Prose, snippet and the PhysicalCalls()==2 guard now agree on the resume-shaped contract.
          round: 3
      blocked: false
content_hash: 6897e52d93a0a521d398d1d18dca4c0d5b54da638f7e03f136c837586bc0d170
---

# Gate ledger — pair#181 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-03T17:02:40-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `evidence-absence-vs-negative` ThreadEvidence cannot distinguish "proof unresolved" from "proof says no", so a total classifier invents archive-eligible reasons
  ClassifyThread is total over ThreadEvidence, but the type carries no
  "unresolved" marker for Parked/Detached (only PathError for the path
  case). Task 5 hands it ThreadEvidence{Live: fromProcOps(...)} with no
  parked/detached proof, so a healthy verified park classifies
  ReasonBindingLost and a healthy detach classifies ReasonSessionGone in
  couch --list, permanently. The same defect fires in the switcher:
  actionableinventory.go:285-292 sets detached=nil on a DetachedSessions
  failure and today calls that "not proved this round"; under the new
  classifier that transient failure asserts session-gone on every detached
  row, and session-gone is a reason M3's DecideRetirement acts on.
- **PQ-2** [Important] `classifier-branch-order` Shell-derived refusals are ordered ahead of the live branch, so today's live rows can classify unusable
  ClassifyThread checks PathError, LatestLaunchProfile == nil and
  IsSupportedAgent before the live branch, and the new shell physicalizes
  every record rather than only resume candidates. Today
  actionableinventory.go:237 skips records with incarnations, so a live
  record never reaches c.Path.Physical and actionableThreadState never
  reads LatestLaunchProfile on the live path. After the change, a running
  agent whose working directory was removed becomes unusable/path-missing
  and Task 4 refuses Enter on it; c.Path == nil turns every row unusable
  instead of only suppressing resume candidates. This falsifies the
  plan's own characterization test and M1's "no behaviour change to what
  is actionable".
- **PQ-3** [Important] `unnamed-refusal-site` M2 relaxes the binding diagnostic in DecideResume but ResumeContext refuses earlier, at resume.go:212
  SessionInventoryNativeBindingResolver.ResolveEstablished returns
  refuseResume(...) itself at resume.go:172-175 for a provisional binding,
  and ResumeContext returns that error at resume.go:212-215 before
  DecideResume runs (FakeThreadArtifactCollisionChecker does the same at
  artifactcollision_fake.go:123). Task 8's file list covers
  resume.go:93-126,269-278 and omits :212, so pair-couch-24 still refuses
  with resume-binding-provisional after M2 and its Done-when is
  unreachable.
- **PQ-4** [Important] `startup-selection-widening` Dropping the binding gate on detached rows widens startup auto-selection, and a warm refusal is fatal to couch startup
  StartInteractive returns resumeErr directly (startup.go:55-58), so a
  resume refusal stops couch in that tree — the incident the comment at
  actionableinventory.go:249-258 says the detached binding gate exists to
  prevent. Task 8 removes that gate and RequireResumeBoundary refuses a
  warm profile that reaches a create boundary (session died between
  projection and launch). The plan never states what startup does with
  that refusal.
- **PQ-5** [Important] `unbacked-existing-behavior-claim` BenchmarkMenu100 is cited as the cost guard but never exercises the inventory
  menu_perf_test.go:91 benchmarks NewMenuState/RenderMenu/reduceKey over a
  fixture []ActionableThreadSummary. It never calls
  ActionableThreadInventoryContext, c.Path.Physical, the binding resolver
  or zellij, so it cannot observe the per-record physicalization the plan
  adds or the enlarged detach-candidate set (ARCH-CONSTRAINTS). Name a
  guard that counts resolver/Physical calls against a fake, or state the
  envelope and accept it.
- **PQ-6** [Minor] `new-state-unhandled-at-consumers` ThreadBusy rows reach Enter and menuActionItems, where !Live() offers switch and resume
  menu.go:388-401 sets operation="switch" for any non-Resumable row and
  menuActionItems returns {"resume","name","describe"} for any non-Live
  row, so a park-in-flight row gets both. Task 4 specifies only the
  unusable case. Downstream refusals catch it, but the row lies about what
  it offers.
- **PQ-7** [Minor] `unbacked-existing-behavior-claim` Task 2's migration scope understates the affected files
  ProjectActionableThreads and ActionableThreadInventory* appear in 14
  files (29 direct calls), not "36 call sites across six files"; the plan
  names four, one of which (artifactpath/deadsymbols_test.go) is outside
  the couch packages.
- **PQ-8** [Minor] `adversarial-input-class-unnamed` CurrentLaunch parses a cross-version append-only ledger; Task 9 gives it two hand-written cases and no adversarial class
  The ledger is written by other processes and older Pair versions and is
  hand-editable (ARCH-SECURE). One strategy line is missing: a property
  test over shuffled, perforated and duplicated record sets, asserting the
  pending-vs-committed invariant, is the mechanical guard the two named
  cases cannot supply.
- **PQ-9** [Minor] `spec-plan-contract-drift` Spec item 5 says archive is reversible "by moving the file back", but Snapshot walks the manifest
  ThreadStore.Snapshot (threadstore.go:504) enumerates manifest.Threads, so
  a file moved back without a manifest re-add produces no row. Task 12
  Step C gets this right; the Spec sentence should match it.

## Round 2 — 2026-09-03T17:08:23-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — ProofStatus makes unresolved representable; ReasonUnknown is never archive-eligible; Task 5 now resolves the same proofs instead of stubbing them.
- PQ-2 — addressed — Live branch is ordered ahead of path/profile/agent, with the :237 justification stated inline.
- PQ-3 — addressed — Step 1b names resume.go:212 and states the covering rule (no resolver on the warm path at all), which also reaches the :269 recheck.
- PQ-4 — addressed — Task 8b decides: keep no-fallback, wrap the refusal at the startup seam with an actionable message.
- PQ-5 — addressed — BenchmarkMenu100 explicitly retired as the guard; replaced by a resolver/Physical call-counting test.
- PQ-6 — not-addressed — Task 4 still specifies only the unusable case; ThreadBusy still reaches Resumable()/Live()-keyed Enter and menuActionItems.
- PQ-7 — not-addressed — Still says "36 call sites across six files"; actual is 14 files. Delete the count rather than correcting it.
- PQ-8 — addressed — Task 9 Step 2b adds the property test over shuffled, perforated and duplicated record sets.
- PQ-9 — addressed — Spec item 5 now says "moving the file back and re-adding it to the manifest", matching Task 12 Step C.

### Raised

- **PQ-10** [Minor] `spec-plan-contract-drift` Task 2 states the per-record Physical cost three ways and the three disagree
  Step 3's snippet calls c.Path.Physical unconditionally per record; Step 5's
  guard asserts PhysicalCalls()==2 over a six-record fixture; the cost-discipline
  paragraph describes a third, lazier shape. 2nd in family: the rule, not the
  instance, is that when the plan states one contract in prose, snippet and guard
  test, the guard test binds and the other two are reconciled to it in the same edit.

## Round 3 — 2026-09-03T17:12:01-07:00 (claude) — passed

### Disposed

- PQ-6 — not-addressed — Task 4 still scopes only the unusable case; ThreadBusy is absent from menu.go routing.
- PQ-7 — not-addressed — plan:464 and issue:161,192 still say 36 across six files; measured 26 across six, and both named outlier files have zero calls.
- PQ-10 — addressed — Prose, snippet and the PhysicalCalls()==2 guard now agree on the resume-shaped contract.

## Open findings

- **PQ-6** [Minor] `new-state-unhandled-at-consumers` ThreadBusy rows reach Enter and menuActionItems, where !Live() offers switch and resume
- **PQ-7** [Minor] `unbacked-existing-behavior-claim` Task 2's migration scope understates the affected files
