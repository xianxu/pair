---
gate: boundary-review
issue: 181
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-03T18:35:37-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: ThreadBusy rows reach Enter and menuActionItems, where !Live() offers switch and resume
          detail: |-
            menu.go:388-401 sets operation="switch" for any non-Resumable row and
            menuActionItems returns {"resume","name","describe"} for any non-Live
            row, so a park-in-flight row gets both. Task 4 specifies only the
            unusable case. Downstream refusals catch it, but the row lies about what
            it offers.
            (carried from plan-quality PQ-6, deferred to the boundary review)
          family: new-state-unhandled-at-consumers
          round: 1
        - id: BR-2
          severity: Minor
          title: Task 2's migration scope understates the affected files
          detail: |-
            ProjectActionableThreads and ActionableThreadInventory* appear in 14
            files (29 direct calls), not "36 call sites across six files"; the plan
            names four, one of which (artifactpath/deadsymbols_test.go) is outside
            the couch packages.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: unbacked-existing-behavior-claim
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-03T18:35:37-07:00"
      agent: claude
      boundary: M1
      blocked: false
      protocol_error: no valid findings block
    - "n": 3
      timestamp: "2026-09-03T22:51:05-07:00"
      agent: claude
      boundary: M3
      blocked: false
      protocol_error: no valid findings block
    - "n": 4
      timestamp: "2026-09-03T23:13:54-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: menuThreadActionable now gates Enter and the action list, but nothing pins the busy case and archive is offered on busy rows — see the rule-level finding.
          round: 4
        - id: BR-2
          disposition: not-addressed
          note: plan:463 corrected to 62/15 (measured 63/15 at HEAD); issue:161 still says "36 call sites in 6 files".
          round: 4
      findings:
        - id: BR-3
          severity: Critical
          title: Tab-archive always fails — the switcher sends "tag", the executor reads "ref"
          detail: |-
            operationdispatch.go:167 resolves a["ref"] while couchtty/menu.go:1454 threadEffect sends {repo-scope, tag}.
            Confirmed through RunWithRuntime: archive(tag) exits 1 with "thread reference not found: empty reference";
            the same call with ref archives the record. archive is the first direct-store op dispatched via threadEffect,
            and no test crosses the dispatcher seam. Fix: use resolveOperationThread(c, a), which accepts either form.
          family: dispatch-arg-contract-mismatch
          round: 4
        - id: BR-4
          severity: Important
          title: An unreadable record has no row, cannot be archived, and fails the whole inventory
          detail: |-
            Snapshot (threadstore.go:517) errors on any undecodable record, and DecodePersisted already runs Validate, so
            ClassifyThread's ReasonInvalid branch cannot fire in production — one bad record empties both views instead of
            producing one honest row. ArchiveThread decodes before moving, so an invalid record can never leave; ArchivedThreads
            silently skips what it cannot decode. Four sites, one rule: a record the decoder rejects must still produce a
            visible row and must never remove other rows.
          family: decode-failure-drops-the-row
          round: 4
        - id: BR-5
          severity: Important
          title: README and atlas still assert the startup and label rules this window reversed
          detail: |-
            2nd in family — state the rule, do not patch instances: a milestone reversing a documented rule sweeps every prose
            restatement in the same commit, enumerated by grepping the superseded symbol and the rule's distinctive phrases.
            Measured 7 sites: README:310-312 ("does not refuse a start because a tree is already busy"), README:299-302
            ("does not rank"), README:277 (tag is the label), README:267-270 (--archived missing); atlas:516-532 (names the
            deleted SelectUniqueResumableRoot, "exactness preserved"), atlas:541-558 (binding gate / rows hidden);
            threadreason.go:34 (dead symbol name). PathHoldsUsableThread and the label rule have no atlas entry at all.
          family: unbacked-existing-behavior-claim
          round: 4
        - id: BR-6
          severity: Important
          title: Occupancy is decided in five places with four different definitions
          detail: |-
            2nd in family — state the rule: "can this be acted on / is it occupied" must be one predicate over the classified
            state and one over incarnation states, shared by all consumers. Sites: menuThreadActionable (menu.go:966),
            menuActionItems (menu.go:1006, offers archive to ThreadBusy), ThreadStore.ArchiveThread (threadstore.go:977,
            refuses only IncarnationLive), DecideResume (resume.go:80-95, refuses live+creating+unknown),
            PathHoldsUsableThread (busy does not block). Failure: archiving a mid-start row quiesces the session being created
            and unlists the record while the spawn is in flight.
          family: new-state-unhandled-at-consumers
          round: 4
        - id: BR-7
          severity: Important
          title: The one-thread-per-path refusal names two next steps, neither of which works
          detail: |-
            couch.go:351-355 suggests "couch <path>" — the command the operator just ran (the guard fires interactively only
            when a live row holds the path) and unavailable from inside the TUI (supervisor lease) — and "couch --show <tag>"
            to "retire it", but --show is read-only and there is no CLI archive. couch_test.go:1347 pins both strings.
          family: unnavigable-refusal
          round: 4
        - id: BR-8
          severity: Minor
          title: TestArchiveThreadRefusesALiveOrParkingThread never exercises the parking branch
          detail: |-
            archive_test.go:59 builds only a live incarnation; threadstore.go:974's Park != nil refusal is unpinned. The same
            shape appears at startup_test.go:132, named "ResumesTheNewest" while asserting only "one of the two".
          family: test-name-overclaims
          round: 4
      boundary: M3
      blocked: true
    - "n": 5
      timestamp: "2026-09-04T07:35:28-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Enter now refuses busy via menuThreadActionable and says "it is busy", but nothing pins it and menuActionItems still offers archive to a busy row; unchanged this round.
          round: 5
        - id: BR-2
          disposition: addressed
          note: issue:161 now reads 63 references across 15 files, and the plan records the measurement in its Revisions.
          round: 5
        - id: BR-3
          disposition: addressed
          note: 'Verified by overlay revert: restoring ResolveThreadReference(a["ref"]) makes both new seam tests fail with "empty reference".'
          round: 5
        - id: BR-4
          disposition: not-addressed
          note: 'The row is visible now, but archive is still unreachable for it: through the real dispatcher archive{tag} fails with "couch: EOF" and archive{ref} with "not found" -- the new test calls couch.ArchiveThread directly, below the very seam whose absence caused BR-3.'
          round: 5
        - id: BR-5
          disposition: not-addressed
          note: Six of seven sites swept well; README's synopsis block (255-270) still omits --archived while usage() prints it, and the new Malformed/invalid-row rule has no atlas entry.
          round: 5
        - id: BR-6
          disposition: not-addressed
          note: 'The reachable failure is fixed and well pinned, but the stated rule is not: occupiedIncarnation has one caller, DecideResume still inlines the same three states, and menuThreadActionable/PathHoldsUsableThread remain two copies of one set.'
          round: 5
        - id: BR-7
          disposition: addressed
          note: The refusal now names switcher gestures reachable from where it fires, and couch_test.go:1355 pins them.
          round: 5
        - id: BR-8
          disposition: not-addressed
          note: 'Verified by revert: deleting archivableRecord''s Park != nil branch leaves TestArchiveThreadRefusesALiveOrParkingThread green, because the fixture also carries a live incarnation. The startup recency half is genuinely fixed.'
          round: 5
      findings:
        - id: BR-9
          severity: Critical
          title: The project's new detail blocks break a contract test and record closes the gates never ran
          detail: |-
            3rd in family -- state the rule, do not patch the block: a state or number in a portfolio artifact is
            written by the gate that produced it, and a judged value says so where it is read. go test
            ./cmd/internal/couchcore -run TestUncheckedProjectMilestoneHasNoClosedMetadata FAILS at HEAD and passed at
            the base: M3's block carries closed/actual while its row is unticked. M2's block records closed 2026-09-03
            and actual 0.9h though no milestone-close ever ran (no Review-Verdict trailer, no closed M2 log line, issue
            Plan still unticked, project row hand-ticked in 6572ef69). M1's actual 0.85h drops the "judgment estimate,
            not measured" qualification the issue Log carries. sdlc milestone-close owns the task row AND the detail
            block; hand-writing actuals ahead of it pollutes velocity calibration, which is what the gate exists to stop.
          family: unbacked-existing-behavior-claim
          round: 5
        - id: BR-10
          severity: Important
          title: Snapshot reports "could not read" as the verdict "invalid", and the record leaves the usable set
          detail: |-
            threadstore.go:517-537 folds an os.ReadFile error and a decode error into one Malformed list rendered as
            ReasonInvalid, whose documented exit is archive. DecodePersisted rejects an unknown schema_version and
            strictjson rejects unknown fields, so an older couch reading a newer store classifies every thread as
            debris -- and because the record also leaves snapshot.Records, PathHoldsUsableThread stops blocking, so
            couch <path> creates a fresh thread over live work. That is the ratchet M3 just closed, now silent where
            the old code failed loudly. Same shape as M1's own ProofStatus lesson: separate unreadable from invalid,
            and let an unreadable manifest-listed record still hold its path.
          family: transient-failure-as-verdict
          round: 5
        - id: BR-11
          severity: Important
          title: Malformed rows are an opt-in variadic, so the pre-181 behaviour is the compile-clean default
          detail: |-
            3rd in family -- do not fix the two call sites. The rule: the records and the evidence describing them
            travel as one value, so a consumer cannot opt out of part of the projection. ProjectActionableThreads
            (actionableinventory.go:174) and BuildThreadInventory (threadinventory.go:49) take malformed as a variadic;
            omitting it silently restores "some records get no row" with no compile error and no failing test. Pass
            ThreadSnapshot or a single input struct -- the call sites already moved once for the evidence parameter.
          family: new-state-unhandled-at-consumers
          round: 5
      boundary: M3
      blocked: true
---

# Gate ledger — pair#181 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-03T18:35:37-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `new-state-unhandled-at-consumers` ThreadBusy rows reach Enter and menuActionItems, where !Live() offers switch and resume
  menu.go:388-401 sets operation="switch" for any non-Resumable row and
  menuActionItems returns {"resume","name","describe"} for any non-Live
  row, so a park-in-flight row gets both. Task 4 specifies only the
  unusable case. Downstream refusals catch it, but the row lies about what
  it offers.
  (carried from plan-quality PQ-6, deferred to the boundary review)
- **BR-2** [Minor] `unbacked-existing-behavior-claim` Task 2's migration scope understates the affected files
  ProjectActionableThreads and ActionableThreadInventory* appear in 14
  files (29 direct calls), not "36 call sites across six files"; the plan
  names four, one of which (artifactpath/deadsymbols_test.go) is outside
  the couch packages.
  (carried from plan-quality PQ-7, deferred to the boundary review)

## Round 2 — 2026-09-03T18:35:37-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 3 — 2026-09-03T22:51:05-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 4 — 2026-09-03T23:13:54-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — menuThreadActionable now gates Enter and the action list, but nothing pins the busy case and archive is offered on busy rows — see the rule-level finding.
- BR-2 — not-addressed — plan:463 corrected to 62/15 (measured 63/15 at HEAD); issue:161 still says "36 call sites in 6 files".

### Raised

- **BR-3** [Critical] `dispatch-arg-contract-mismatch` Tab-archive always fails — the switcher sends "tag", the executor reads "ref"
  operationdispatch.go:167 resolves a["ref"] while couchtty/menu.go:1454 threadEffect sends {repo-scope, tag}.
  Confirmed through RunWithRuntime: archive(tag) exits 1 with "thread reference not found: empty reference";
  the same call with ref archives the record. archive is the first direct-store op dispatched via threadEffect,
  and no test crosses the dispatcher seam. Fix: use resolveOperationThread(c, a), which accepts either form.
- **BR-4** [Important] `decode-failure-drops-the-row` An unreadable record has no row, cannot be archived, and fails the whole inventory
  Snapshot (threadstore.go:517) errors on any undecodable record, and DecodePersisted already runs Validate, so
  ClassifyThread's ReasonInvalid branch cannot fire in production — one bad record empties both views instead of
  producing one honest row. ArchiveThread decodes before moving, so an invalid record can never leave; ArchivedThreads
  silently skips what it cannot decode. Four sites, one rule: a record the decoder rejects must still produce a
  visible row and must never remove other rows.
- **BR-5** [Important] `unbacked-existing-behavior-claim` README and atlas still assert the startup and label rules this window reversed
  2nd in family — state the rule, do not patch instances: a milestone reversing a documented rule sweeps every prose
  restatement in the same commit, enumerated by grepping the superseded symbol and the rule's distinctive phrases.
  Measured 7 sites: README:310-312 ("does not refuse a start because a tree is already busy"), README:299-302
  ("does not rank"), README:277 (tag is the label), README:267-270 (--archived missing); atlas:516-532 (names the
  deleted SelectUniqueResumableRoot, "exactness preserved"), atlas:541-558 (binding gate / rows hidden);
  threadreason.go:34 (dead symbol name). PathHoldsUsableThread and the label rule have no atlas entry at all.
- **BR-6** [Important] `new-state-unhandled-at-consumers` Occupancy is decided in five places with four different definitions
  2nd in family — state the rule: "can this be acted on / is it occupied" must be one predicate over the classified
  state and one over incarnation states, shared by all consumers. Sites: menuThreadActionable (menu.go:966),
  menuActionItems (menu.go:1006, offers archive to ThreadBusy), ThreadStore.ArchiveThread (threadstore.go:977,
  refuses only IncarnationLive), DecideResume (resume.go:80-95, refuses live+creating+unknown),
  PathHoldsUsableThread (busy does not block). Failure: archiving a mid-start row quiesces the session being created
  and unlists the record while the spawn is in flight.
- **BR-7** [Important] `unnavigable-refusal` The one-thread-per-path refusal names two next steps, neither of which works
  couch.go:351-355 suggests "couch <path>" — the command the operator just ran (the guard fires interactively only
  when a live row holds the path) and unavailable from inside the TUI (supervisor lease) — and "couch --show <tag>"
  to "retire it", but --show is read-only and there is no CLI archive. couch_test.go:1347 pins both strings.
- **BR-8** [Minor] `test-name-overclaims` TestArchiveThreadRefusesALiveOrParkingThread never exercises the parking branch
  archive_test.go:59 builds only a live incarnation; threadstore.go:974's Park != nil refusal is unpinned. The same
  shape appears at startup_test.go:132, named "ResumesTheNewest" while asserting only "one of the two".

## Round 5 — 2026-09-04T07:35:28-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Enter now refuses busy via menuThreadActionable and says "it is busy", but nothing pins it and menuActionItems still offers archive to a busy row; unchanged this round.
- BR-2 — addressed — issue:161 now reads 63 references across 15 files, and the plan records the measurement in its Revisions.
- BR-3 — addressed — Verified by overlay revert: restoring ResolveThreadReference(a["ref"]) makes both new seam tests fail with "empty reference".
- BR-4 — not-addressed — The row is visible now, but archive is still unreachable for it: through the real dispatcher archive{tag} fails with "couch: EOF" and archive{ref} with "not found" -- the new test calls couch.ArchiveThread directly, below the very seam whose absence caused BR-3.
- BR-5 — not-addressed — Six of seven sites swept well; README's synopsis block (255-270) still omits --archived while usage() prints it, and the new Malformed/invalid-row rule has no atlas entry.
- BR-6 — not-addressed — The reachable failure is fixed and well pinned, but the stated rule is not: occupiedIncarnation has one caller, DecideResume still inlines the same three states, and menuThreadActionable/PathHoldsUsableThread remain two copies of one set.
- BR-7 — addressed — The refusal now names switcher gestures reachable from where it fires, and couch_test.go:1355 pins them.
- BR-8 — not-addressed — Verified by revert: deleting archivableRecord's Park != nil branch leaves TestArchiveThreadRefusesALiveOrParkingThread green, because the fixture also carries a live incarnation. The startup recency half is genuinely fixed.

### Raised

- **BR-9** [Critical] `unbacked-existing-behavior-claim` The project's new detail blocks break a contract test and record closes the gates never ran
  3rd in family -- state the rule, do not patch the block: a state or number in a portfolio artifact is
  written by the gate that produced it, and a judged value says so where it is read. go test
  ./cmd/internal/couchcore -run TestUncheckedProjectMilestoneHasNoClosedMetadata FAILS at HEAD and passed at
  the base: M3's block carries closed/actual while its row is unticked. M2's block records closed 2026-09-03
  and actual 0.9h though no milestone-close ever ran (no Review-Verdict trailer, no closed M2 log line, issue
  Plan still unticked, project row hand-ticked in 6572ef69). M1's actual 0.85h drops the "judgment estimate,
  not measured" qualification the issue Log carries. sdlc milestone-close owns the task row AND the detail
  block; hand-writing actuals ahead of it pollutes velocity calibration, which is what the gate exists to stop.
- **BR-10** [Important] `transient-failure-as-verdict` Snapshot reports "could not read" as the verdict "invalid", and the record leaves the usable set
  threadstore.go:517-537 folds an os.ReadFile error and a decode error into one Malformed list rendered as
  ReasonInvalid, whose documented exit is archive. DecodePersisted rejects an unknown schema_version and
  strictjson rejects unknown fields, so an older couch reading a newer store classifies every thread as
  debris -- and because the record also leaves snapshot.Records, PathHoldsUsableThread stops blocking, so
  couch <path> creates a fresh thread over live work. That is the ratchet M3 just closed, now silent where
  the old code failed loudly. Same shape as M1's own ProofStatus lesson: separate unreadable from invalid,
  and let an unreadable manifest-listed record still hold its path.
- **BR-11** [Important] `new-state-unhandled-at-consumers` Malformed rows are an opt-in variadic, so the pre-181 behaviour is the compile-clean default
  3rd in family -- do not fix the two call sites. The rule: the records and the evidence describing them
  travel as one value, so a consumer cannot opt out of part of the projection. ProjectActionableThreads
  (actionableinventory.go:174) and BuildThreadInventory (threadinventory.go:49) take malformed as a variadic;
  omitting it silently restores "some records get no row" with no compile error and no failing test. Pass
  ThreadSnapshot or a single input struct -- the call sites already moved once for the evidence parameter.

## Open findings

- **BR-1** [Minor] `new-state-unhandled-at-consumers` ThreadBusy rows reach Enter and menuActionItems, where !Live() offers switch and resume
- **BR-4** [Important] `decode-failure-drops-the-row` An unreadable record has no row, cannot be archived, and fails the whole inventory
- **BR-5** [Important] `unbacked-existing-behavior-claim` README and atlas still assert the startup and label rules this window reversed
- **BR-6** [Important] `new-state-unhandled-at-consumers` Occupancy is decided in five places with four different definitions
- **BR-8** [Minor] `test-name-overclaims` TestArchiveThreadRefusesALiveOrParkingThread never exercises the parking branch
- **BR-9** [Critical] `unbacked-existing-behavior-claim` The project's new detail blocks break a contract test and record closes the gates never ran
- **BR-10** [Important] `transient-failure-as-verdict` Snapshot reports "could not read" as the verdict "invalid", and the record leaves the usable set
- **BR-11** [Important] `new-state-unhandled-at-consumers` Malformed rows are an opt-in variadic, so the pre-181 behaviour is the compile-clean default
