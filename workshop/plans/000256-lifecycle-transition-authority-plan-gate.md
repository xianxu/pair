---
gate: plan-quality
issue: 256
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-16T21:35:13-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: Task 2 removes the incarnation from the classifier but DecideResume still refuses on it
          detail: |-
            resume.go:98 and resume.go:104 (via occupiedIncarnation, thread.go:377) refuse
            any record with an open park or an occupied incarnation, and no task touches
            resume.go. Post-M1 a pair#272 record classifies ThreadDetached, is ranked
            highest by SelectResumableRoot (startup.go:37) so startup auto-selects it, is
            offered resume by menu.go:1263, and the resume refuses. The Revisions section
            claims this finding is dissolved; that reasoning covers only the classifier.
            Task 8 handles the identical problem for archive and has no counterpart here.
          family: classification-not-authority
          round: 1
        - id: PQ-2
          severity: Important
          title: Narrowing ThreadEvidence.Live to couch's own children strands the CLI consumer
          detail: |-
            Live is today the union of console pty children and ObserveRecordedProcesses
            (actionableinventory.go:413-437), and ThreadInventoryContext passes nil
            observations (threadinventory.go:97-100) so that union is the CLI's only
            liveness proof - the comment names pair#181 "one store, two stories". The plan
            says Live is "sourced from couch's own child table" and that
            ObserveRecordedProcesses is no longer a classification input, but never states
            the fate of the union at :436 or of the CLI. As written, couch --list reads
            every running thread as detached and offers resume on a hosted thread.
          family: evidence-narrowing-drops-consumer
          round: 1
        - id: PQ-3
          severity: Important
          title: ArchivableState(state, reason) cannot be evaluated at threadstore.go:1106
          detail: |-
            That call site is inside s.withLock on a record decoded from disk, with no
            evidence, no session observation and no Couch (threadstore.go:1072-1106). A
            predicate over the classification must either be handed a possibly-stale
            classification by the caller - defeating the second-line-of-defence property
            the comment at :1093-1095 states - or do session IO inside the store lock.
            Task 8 lists the site but not which. Also state the consequence: post-M1
            ArchivableState(detached, "") is true, so the store stops independently
            refusing a record that still carries a live incarnation.
          family: guard-needs-inputs-its-layer-lacks
          round: 1
        - id: PQ-4
          severity: Minor
          title: Two cited symbols/ranges do not match current code
          detail: |-
            Task 8's test calls couchcore.AllThreadStates(), which does not exist - only
            AllThreadReasons() (threadreason.go:67). Task 6's modify range
            recovery_execute.go:94-113 is reconcileRecoveryHelper, which only propagates
            the error at :105; the absent-binding error is raised in
            observeRecoverySession at :66-68.
          family: unbacked-code-reference
          round: 1
        - id: PQ-5
          severity: Minor
          title: The ARCH-CONSTRAINTS budget measures startup but the change is per-refresh
          detail: |-
            gatherThreadEvidence today bounds whether the zellij snapshot runs at all
            (actionableinventory.go:441-447 - "a couch with nothing detachable pays
            nothing"). Task 1 gathers session evidence for every record, so every refresh
            pays one list-sessions. The refresh is async and coalesced
            (console_menu.go:83-113) so it does not block a keystroke; the budget should
            say so and give the per-refresh figure, not only the startup round.
          family: budget-omits-a-path
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-16T21:40:14-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Task 8a plus the "Two questions, two authorities" section fix the class, not the archive instance; occupiedIncarnation is deleted.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Live keeps the console-children union ObserveRecordedProcesses union and becomes positive-only; the CLI consumer is named.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Task 8b gives the store a record-only guard and states that ArchivableState(detached, "") is true post-M1.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: AllThreadStates added as a step; Task 6 re-pointed at observeRecoverySession where the error is raised.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Both startup and per-refresh figures budgeted, with the coalesced-worker note.
          round: 2
      findings:
        - id: PQ-6
          severity: Minor
          title: Task 9 changes an evidence producer without enumerating its consumers; the plan calls it a non-input while Live consumes it
          detail: |-
            RULE (2nd instance in this family, 2 of 2 producer-changing tasks): before
            changing an evidence producer, enumerate every consumer of its output and
            state what each receives after the change. The Integration-points bullet says
            ObserveRecordedProcesses "is no longer a classification input", but its output
            is unioned into Live at actionableinventory.go:436 and Live is classification
            row 4. Task 9's three-way switch therefore needs one line saying only
            confirmed-Live observations enter Live while Unknown is carried to
            DecideRecovery and archive; otherwise an Unknown probe can classify a thread
            live and hide it from both resume and recovery.
          family: evidence-narrowing-drops-consumer
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-09-16T21:46:48-07:00"
      agent: claude
      dispose:
        - id: PQ-6
          disposition: addressed
          note: Integration points now states the rule and enumerates all three consumers with what each receives; only confirmed-Live enters Live.
          round: 3
      blocked: false
content_hash: d70ee72173b905a850ae6968311a37d6f79bd0a4b8cff1147889f9e29de8dc62
---

# Gate ledger — pair#256 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-16T21:35:13-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `classification-not-authority` Task 2 removes the incarnation from the classifier but DecideResume still refuses on it
  resume.go:98 and resume.go:104 (via occupiedIncarnation, thread.go:377) refuse
  any record with an open park or an occupied incarnation, and no task touches
  resume.go. Post-M1 a pair#272 record classifies ThreadDetached, is ranked
  highest by SelectResumableRoot (startup.go:37) so startup auto-selects it, is
  offered resume by menu.go:1263, and the resume refuses. The Revisions section
  claims this finding is dissolved; that reasoning covers only the classifier.
  Task 8 handles the identical problem for archive and has no counterpart here.
- **PQ-2** [Important] `evidence-narrowing-drops-consumer` Narrowing ThreadEvidence.Live to couch's own children strands the CLI consumer
  Live is today the union of console pty children and ObserveRecordedProcesses
  (actionableinventory.go:413-437), and ThreadInventoryContext passes nil
  observations (threadinventory.go:97-100) so that union is the CLI's only
  liveness proof - the comment names pair#181 "one store, two stories". The plan
  says Live is "sourced from couch's own child table" and that
  ObserveRecordedProcesses is no longer a classification input, but never states
  the fate of the union at :436 or of the CLI. As written, couch --list reads
  every running thread as detached and offers resume on a hosted thread.
- **PQ-3** [Important] `guard-needs-inputs-its-layer-lacks` ArchivableState(state, reason) cannot be evaluated at threadstore.go:1106
  That call site is inside s.withLock on a record decoded from disk, with no
  evidence, no session observation and no Couch (threadstore.go:1072-1106). A
  predicate over the classification must either be handed a possibly-stale
  classification by the caller - defeating the second-line-of-defence property
  the comment at :1093-1095 states - or do session IO inside the store lock.
  Task 8 lists the site but not which. Also state the consequence: post-M1
  ArchivableState(detached, "") is true, so the store stops independently
  refusing a record that still carries a live incarnation.
- **PQ-4** [Minor] `unbacked-code-reference` Two cited symbols/ranges do not match current code
  Task 8's test calls couchcore.AllThreadStates(), which does not exist - only
  AllThreadReasons() (threadreason.go:67). Task 6's modify range
  recovery_execute.go:94-113 is reconcileRecoveryHelper, which only propagates
  the error at :105; the absent-binding error is raised in
  observeRecoverySession at :66-68.
- **PQ-5** [Minor] `budget-omits-a-path` The ARCH-CONSTRAINTS budget measures startup but the change is per-refresh
  gatherThreadEvidence today bounds whether the zellij snapshot runs at all
  (actionableinventory.go:441-447 - "a couch with nothing detachable pays
  nothing"). Task 1 gathers session evidence for every record, so every refresh
  pays one list-sessions. The refresh is async and coalesced
  (console_menu.go:83-113) so it does not block a keystroke; the budget should
  say so and give the per-refresh figure, not only the startup round.

## Round 2 — 2026-09-16T21:40:14-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Task 8a plus the "Two questions, two authorities" section fix the class, not the archive instance; occupiedIncarnation is deleted.
- PQ-2 — addressed — Live keeps the console-children union ObserveRecordedProcesses union and becomes positive-only; the CLI consumer is named.
- PQ-3 — addressed — Task 8b gives the store a record-only guard and states that ArchivableState(detached, "") is true post-M1.
- PQ-4 — addressed — AllThreadStates added as a step; Task 6 re-pointed at observeRecoverySession where the error is raised.
- PQ-5 — addressed — Both startup and per-refresh figures budgeted, with the coalesced-worker note.

### Raised

- **PQ-6** [Minor] `evidence-narrowing-drops-consumer` Task 9 changes an evidence producer without enumerating its consumers; the plan calls it a non-input while Live consumes it
  RULE (2nd instance in this family, 2 of 2 producer-changing tasks): before
  changing an evidence producer, enumerate every consumer of its output and
  state what each receives after the change. The Integration-points bullet says
  ObserveRecordedProcesses "is no longer a classification input", but its output
  is unioned into Live at actionableinventory.go:436 and Live is classification
  row 4. Task 9's three-way switch therefore needs one line saying only
  confirmed-Live observations enter Live while Unknown is carried to
  DecideRecovery and archive; otherwise an Unknown probe can classify a thread
  live and hide it from both resume and recovery.

## Round 3 — 2026-09-16T21:46:48-07:00 (claude) — passed

### Disposed

- PQ-6 — addressed — Integration points now states the rule and enumerates all three consumers with what each receives; only confirmed-Live enters Live.

## Open findings

(none — every finding has been disposed)
