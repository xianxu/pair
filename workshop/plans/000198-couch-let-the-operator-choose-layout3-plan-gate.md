---
gate: plan-quality
issue: 198
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-06T13:09:23-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: Guard compares the raw persisted layout, so every pre-change record blocks a default startup
          detail: |-
            Task 2 populates ActionableThreadSummary.Layout straight from record.Layout
            in ProjectActionableThreads (actionableinventory.go:204-221) and compares
            `row.Layout == requested`. Old records have no layout field, so the value is
            Layout(""), and a bare `couch` (Layout2) reports every detached thread as a
            conflict. ParseLayout("") -> Layout2 is defined but never wired in. Name the
            normalization point and say what the row holds when ParseLayout errors, since
            ProjectActionableThreads has no error return.
          family: untrusted-persisted-field-unnormalized
          round: 1
        - id: PQ-2
          severity: Important
          title: Guard is placed in a run.go branch where no Couch exists, and reaching the inventory there breaks its own budget
          detail: |-
            RunWithRuntime's cliLaunch branch is run.go:159-166; the Couch is built at
            run.go:252 and the console at :246-249, before it. The only startup inventory
            read is StartInteractive's ActionableThreadInventoryContext at startup.go:132,
            inside couchcore. A guard in couchcmd must therefore make a second
            enumeration, which the plan's ARCH-CONSTRAINTS budget declares wrong by
            construction. Choose: guard inside StartInteractive reusing rows, or revise
            the budget.
          family: guard-placement-contradicts-io-budget
          round: 1
        - id: PQ-3
          severity: Important
          title: No named route for the layout from ParseCLI into Couch.Layout
          detail: |-
            couchcore.New is at couch.go:81 (not :99) and takes no layout. The CLI builds
            the Couch through Runtime.NewCouchWith(Runner, CouchNamespace)
            (run.go:36-39, :90, :252), implemented by testRT at run_test.go:96, and
            StartInteractive is reached via dispatchInteractiveStart's map of path/agent
            (run.go:307). Threading the layout changes one of these seams; the plan names
            none of them.
          family: unnamed-plumbing-seam
          round: 1
        - id: PQ-4
          severity: Important
          title: Witness write names no ThreadStore mutator, revision, or conflict path
          detail: |-
            "Record the witness c.Layout on the thread" leaves open whether it rides in
            the existing AdvanceStart transaction (launch_existing.go:78-83) or is a
            second UpdateExistingThread (threadstore.go:262), which needs an
            expectedRevision that has just moved. The ARCH-ORDER table covers process
            death but not a revision conflict on a write that happens after the child is
            already running.
          family: store-mutation-unspecified
          round: 1
        - id: PQ-5
          severity: Important
          title: Blocking set omits ThreadBusy and ThreadArchived, and the table asserts a guarantee that follows
          detail: |-
            ActionableThreadState has six values (actionableinventory.go:19-37). The plan
            dispositions four. ThreadBusy is a park in flight -- a session still alive --
            so it does not block, and a failed park returns the thread to the old layout.
            The ARCH-ORDER row claiming a warm reattach's witness equals L "guaranteed by
            the guard" is therefore false; weaken it to what #179 provides.
          family: incomplete-state-enumeration
          round: 1
        - id: PQ-6
          severity: Minor
          title: Old-record test fixture uses schema_version 1, but current records are schema 2
          detail: |-
            threadrecord.SchemaVersion is 2 (record.go:15) and Validate hard-refuses a
            mismatch (:108-109). A pre-change record is schema 2 with no layout field.
            The "no version bump" decision itself is correctly reasoned; only the fixture
            premise is wrong.
          family: stale-fixture-premise
          round: 1
        - id: PQ-7
          severity: Minor
          title: Plan states no non-goals
          detail: |-
            It never says that a menu affordance, mid-session layout switching, or a
            per-thread override are deliberately not built, though the issue's Spec left
            the operator surface open and the Revisions narrowed it to a CLI flag.
          family: no-stated-non-goals
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-06T13:16:52-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: NormalizeLayout + LayoutUnknown, with ProjectActionableThreads named as the normalization point and pinned by TestProjectionNormalizesAbsentLayoutToLayout2.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Guard moved into StartInteractive after startup.go:132, reusing the rows already read; budget pinned by TestGuardAddsNoSessionEnumeration.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Typed parameter through runTypedOperation to c.Layout at run.go:252; Runtime and testRT unchanged, default set in New's literal at couch.go:99.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Rides the existing StartRegistered AdvanceStart, so there is no second write and no expectedRevision to reconcile.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: All six ActionableThreadState values dispositioned, ThreadBusy blocks, and the warm-reattach row now claims no guarantee from the guard.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Fixture is schema_version 2; the no-bump reasoning is retained and correct against record.go:108-109.
          round: 2
        - id: PQ-7
          disposition: addressed
          note: Non-goals section names the menu affordance, mid-session switching, per-thread override, and live-session migration.
          round: 2
      findings:
        - id: PQ-8
          severity: Minor
          title: The StartRegistered AdvanceStart is cited at launch_existing.go:78-83, which is the StartHelperRecorded call
          detail: |-
            launch_existing.go:82-83 is `StartEvent{Kind: StartHelperRecorded, ...}`; the
            StartRegistered advance is the one-liner at launch_existing.go:124. The plan
            repeats the wrong anchor twice (integration-points table and Task 4 Step 3).
            Attaching `Layout: c.Layout` at :82 while applying it in the StartRegistered
            case leaves the witness never written -- silent until
            TestColdResumeRewritesTheWitnessInOneTransaction runs. Same class, smaller
            stakes: startupResumeRefusal is at startup.go:152 (cited :143) and cli.go's
            "path cannot be combined" guard is at :87-88 (cited :84).
          family: miscited-code-anchor
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-09-06T13:21:24-07:00"
      agent: claude
      dispose:
        - id: PQ-8
          disposition: addressed
          note: 'Verified in-tree: starttransaction.go:74 is the StartRegistered case, launch_existing.go:124 is the StartRegistered AdvanceStart, startup.go:152 is startupResumeRefusal, cli.go:87-88 is the path guard.'
          round: 3
      blocked: false
    - "n": 4
      timestamp: "2026-09-06T13:29:34-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Normalization table plus NormalizeLayout/LayoutUnknown; Task 3 pins the pre-change-record startup.
          round: 4
        - id: PQ-2
          disposition: addressed
          note: Guard now sits at startup.go:132 on rows already read; the run.go rationale is verified correct.
          round: 4
        - id: PQ-3
          disposition: addressed
          note: cliInvocation.layout as a typed parameter through runTypedOperation to run.go:252, no Runtime change.
          round: 4
        - id: PQ-4
          disposition: addressed
          note: Rides the existing StartRegistered CAS; no second write, so no revision-conflict path to design.
          round: 4
        - id: PQ-5
          disposition: addressed
          note: All six ActionableThreadState values dispositioned, and the warm row now claims no guarantee.
          round: 4
        - id: PQ-6
          disposition: addressed
          note: Fixture is schema_version 2, matching threadrecord.SchemaVersion.
          round: 4
        - id: PQ-7
          disposition: addressed
          note: Four non-goals stated with reasons.
          round: 4
        - id: PQ-8
          disposition: addressed
          note: 'Verified: :124 is StartRegistered, :82-83 is StartHelperRecorded, and the plan now warns about the confusion.'
          round: 4
      findings:
        - id: PQ-9
          severity: Important
          title: Task 3 claims an older binary ignores the new layout field, but strictjson rejects unknown fields
          detail: |-
            Persisted records load via threadrecord.DecodePersisted -> strictjson.Decode
            (record.go:201), which sets DisallowUnknownFields (strictjson/decode.go:23).
            compat_test.go:13-18 documents this exact hazard and the tombstone discipline
            built for it. Since Task 4 writes the witness on every StartRegistered, once
            the feature runs, reverting couch makes every touched record undecodable and
            those threads vanish into ThreadUnusable/ReasonUnreadable rows. The no-bump
            decision stands on its first reason; correct the second sentence and add the
            read-by-an-older-version direction to the ARCH-SECURE analysis, which
            currently covers only written-by-an-older-version.
          family: persisted-decode-compat-unverified
          round: 4
        - id: PQ-10
          severity: Minor
          title: No named test pins that a warm reattach leaves the layout witness unchanged
          detail: |-
            This is the 2nd finding in family store-mutation-unspecified; the covering
            rule is that every store-mutation invariant the plan states in prose needs a
            named test asserting it, not just the argv it implies. Task 4 requires the
            layout be passed only when !in.Warm, but StartEvent.Layout's zero value is
            Layout(""), which NormalizeLayout maps back to Layout2 -- so an unconditional
            apply in the StartRegistered arm silently downgrades a layout3 thread's
            witness on its first reattach and no planned test would catch it. Extend
            TestWarmReattachSendsNoLayoutEvenInLayout3 to assert the witness too.
          family: store-mutation-unspecified
          round: 4
      blocked: false
content_hash: b1fe4146c91731cd3d67c8a6ca500efb1c786a66447a638c724e444ec740bcb0
---

# Gate ledger — pair#198 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-06T13:09:23-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `untrusted-persisted-field-unnormalized` Guard compares the raw persisted layout, so every pre-change record blocks a default startup
  Task 2 populates ActionableThreadSummary.Layout straight from record.Layout
  in ProjectActionableThreads (actionableinventory.go:204-221) and compares
  `row.Layout == requested`. Old records have no layout field, so the value is
  Layout(""), and a bare `couch` (Layout2) reports every detached thread as a
  conflict. ParseLayout("") -> Layout2 is defined but never wired in. Name the
  normalization point and say what the row holds when ParseLayout errors, since
  ProjectActionableThreads has no error return.
- **PQ-2** [Important] `guard-placement-contradicts-io-budget` Guard is placed in a run.go branch where no Couch exists, and reaching the inventory there breaks its own budget
  RunWithRuntime's cliLaunch branch is run.go:159-166; the Couch is built at
  run.go:252 and the console at :246-249, before it. The only startup inventory
  read is StartInteractive's ActionableThreadInventoryContext at startup.go:132,
  inside couchcore. A guard in couchcmd must therefore make a second
  enumeration, which the plan's ARCH-CONSTRAINTS budget declares wrong by
  construction. Choose: guard inside StartInteractive reusing rows, or revise
  the budget.
- **PQ-3** [Important] `unnamed-plumbing-seam` No named route for the layout from ParseCLI into Couch.Layout
  couchcore.New is at couch.go:81 (not :99) and takes no layout. The CLI builds
  the Couch through Runtime.NewCouchWith(Runner, CouchNamespace)
  (run.go:36-39, :90, :252), implemented by testRT at run_test.go:96, and
  StartInteractive is reached via dispatchInteractiveStart's map of path/agent
  (run.go:307). Threading the layout changes one of these seams; the plan names
  none of them.
- **PQ-4** [Important] `store-mutation-unspecified` Witness write names no ThreadStore mutator, revision, or conflict path
  "Record the witness c.Layout on the thread" leaves open whether it rides in
  the existing AdvanceStart transaction (launch_existing.go:78-83) or is a
  second UpdateExistingThread (threadstore.go:262), which needs an
  expectedRevision that has just moved. The ARCH-ORDER table covers process
  death but not a revision conflict on a write that happens after the child is
  already running.
- **PQ-5** [Important] `incomplete-state-enumeration` Blocking set omits ThreadBusy and ThreadArchived, and the table asserts a guarantee that follows
  ActionableThreadState has six values (actionableinventory.go:19-37). The plan
  dispositions four. ThreadBusy is a park in flight -- a session still alive --
  so it does not block, and a failed park returns the thread to the old layout.
  The ARCH-ORDER row claiming a warm reattach's witness equals L "guaranteed by
  the guard" is therefore false; weaken it to what #179 provides.
- **PQ-6** [Minor] `stale-fixture-premise` Old-record test fixture uses schema_version 1, but current records are schema 2
  threadrecord.SchemaVersion is 2 (record.go:15) and Validate hard-refuses a
  mismatch (:108-109). A pre-change record is schema 2 with no layout field.
  The "no version bump" decision itself is correctly reasoned; only the fixture
  premise is wrong.
- **PQ-7** [Minor] `no-stated-non-goals` Plan states no non-goals
  It never says that a menu affordance, mid-session layout switching, or a
  per-thread override are deliberately not built, though the issue's Spec left
  the operator surface open and the Revisions narrowed it to a CLI flag.

## Round 2 — 2026-09-06T13:16:52-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — NormalizeLayout + LayoutUnknown, with ProjectActionableThreads named as the normalization point and pinned by TestProjectionNormalizesAbsentLayoutToLayout2.
- PQ-2 — addressed — Guard moved into StartInteractive after startup.go:132, reusing the rows already read; budget pinned by TestGuardAddsNoSessionEnumeration.
- PQ-3 — addressed — Typed parameter through runTypedOperation to c.Layout at run.go:252; Runtime and testRT unchanged, default set in New's literal at couch.go:99.
- PQ-4 — addressed — Rides the existing StartRegistered AdvanceStart, so there is no second write and no expectedRevision to reconcile.
- PQ-5 — addressed — All six ActionableThreadState values dispositioned, ThreadBusy blocks, and the warm-reattach row now claims no guarantee from the guard.
- PQ-6 — addressed — Fixture is schema_version 2; the no-bump reasoning is retained and correct against record.go:108-109.
- PQ-7 — addressed — Non-goals section names the menu affordance, mid-session switching, per-thread override, and live-session migration.

### Raised

- **PQ-8** [Minor] `miscited-code-anchor` The StartRegistered AdvanceStart is cited at launch_existing.go:78-83, which is the StartHelperRecorded call
  launch_existing.go:82-83 is `StartEvent{Kind: StartHelperRecorded, ...}`; the
  StartRegistered advance is the one-liner at launch_existing.go:124. The plan
  repeats the wrong anchor twice (integration-points table and Task 4 Step 3).
  Attaching `Layout: c.Layout` at :82 while applying it in the StartRegistered
  case leaves the witness never written -- silent until
  TestColdResumeRewritesTheWitnessInOneTransaction runs. Same class, smaller
  stakes: startupResumeRefusal is at startup.go:152 (cited :143) and cli.go's
  "path cannot be combined" guard is at :87-88 (cited :84).

## Round 3 — 2026-09-06T13:21:24-07:00 (claude) — passed

### Disposed

- PQ-8 — addressed — Verified in-tree: starttransaction.go:74 is the StartRegistered case, launch_existing.go:124 is the StartRegistered AdvanceStart, startup.go:152 is startupResumeRefusal, cli.go:87-88 is the path guard.

## Round 4 — 2026-09-06T13:29:34-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Normalization table plus NormalizeLayout/LayoutUnknown; Task 3 pins the pre-change-record startup.
- PQ-2 — addressed — Guard now sits at startup.go:132 on rows already read; the run.go rationale is verified correct.
- PQ-3 — addressed — cliInvocation.layout as a typed parameter through runTypedOperation to run.go:252, no Runtime change.
- PQ-4 — addressed — Rides the existing StartRegistered CAS; no second write, so no revision-conflict path to design.
- PQ-5 — addressed — All six ActionableThreadState values dispositioned, and the warm row now claims no guarantee.
- PQ-6 — addressed — Fixture is schema_version 2, matching threadrecord.SchemaVersion.
- PQ-7 — addressed — Four non-goals stated with reasons.
- PQ-8 — addressed — Verified: :124 is StartRegistered, :82-83 is StartHelperRecorded, and the plan now warns about the confusion.

### Raised

- **PQ-9** [Important] `persisted-decode-compat-unverified` Task 3 claims an older binary ignores the new layout field, but strictjson rejects unknown fields
  Persisted records load via threadrecord.DecodePersisted -> strictjson.Decode
  (record.go:201), which sets DisallowUnknownFields (strictjson/decode.go:23).
  compat_test.go:13-18 documents this exact hazard and the tombstone discipline
  built for it. Since Task 4 writes the witness on every StartRegistered, once
  the feature runs, reverting couch makes every touched record undecodable and
  those threads vanish into ThreadUnusable/ReasonUnreadable rows. The no-bump
  decision stands on its first reason; correct the second sentence and add the
  read-by-an-older-version direction to the ARCH-SECURE analysis, which
  currently covers only written-by-an-older-version.
- **PQ-10** [Minor] `store-mutation-unspecified` No named test pins that a warm reattach leaves the layout witness unchanged
  This is the 2nd finding in family store-mutation-unspecified; the covering
  rule is that every store-mutation invariant the plan states in prose needs a
  named test asserting it, not just the argv it implies. Task 4 requires the
  layout be passed only when !in.Warm, but StartEvent.Layout's zero value is
  Layout(""), which NormalizeLayout maps back to Layout2 -- so an unconditional
  apply in the StartRegistered arm silently downgrades a layout3 thread's
  witness on its first reattach and no planned test would catch it. Extend
  TestWarmReattachSendsNoLayoutEvenInLayout3 to assert the witness too.

## Open findings

- **PQ-9** [Important] `persisted-decode-compat-unverified` Task 3 claims an older binary ignores the new layout field, but strictjson rejects unknown fields
- **PQ-10** [Minor] `store-mutation-unspecified` No named test pins that a warm reattach leaves the layout witness unchanged
