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
    - "n": 6
      timestamp: "2026-09-04T08:03:03-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Unchanged this round -- menu.go's only edit was unusableThreadNotice; menuActionItems still offers archive to a ThreadBusy row and couchtty still has no ThreadBusy behavioural test.
          round: 6
        - id: BR-4
          disposition: addressed
          note: 'Verified by revert: restoring resolveOperationThread in the archive branch fails TestAnUnreadableRecordCanBeArchivedThroughTheRuntime with "couch: EOF". Row is visible and removable through the real dispatcher.'
          round: 6
        - id: BR-5
          disposition: not-addressed
          note: 'The atlas gained the unreadable-record entry, but the same window reversed "debris does not block" and swept none of its restatements: README:319-321, atlas:563-567 (four lines below the new paragraph, and it names the exact hazard now realized), issue Revisions:244-246, and the plan''s round-2 entry still says Snapshot carries "Malformed" emitted as "invalid" rows. README''s synopsis block still omits --archived.'
          round: 6
        - id: BR-6
          disposition: not-addressed
          note: 'Unchanged, and now worse: PathHoldsUnreadableThread is a sixth independent occupancy/actionability predicate. occupiedIncarnation still has one caller; DecideResume (resume.go:80-95) still inlines the same three states.'
          round: 6
        - id: BR-8
          disposition: addressed
          note: 'Verified by revert: deleting archivableRecord''s Park != nil branch now fails at archive_test.go:106. The startup recency half was fixed in the prior round.'
          round: 6
        - id: BR-9
          disposition: addressed
          note: TestUncheckedProjectMilestoneHasNoClosedMetadata is red at f9f6cdd6 and green at HEAD (measured both). M3's block carries no closed/actual and its row is unticked; M2 records the missing gate instead of inventing an actual; M1's "judgment estimate, not measured" qualification is restored. sdlc's upsertField inserts after **est:**, so the gate can still write them.
          round: 6
        - id: BR-10
          disposition: addressed
          note: 'ReasonUnreadable is split from ReasonInvalid at the layer where the read fails, and the record still blocks its scope. Two residues raised separately: ReasonInvalid.Label() still says "unreadable record", and the block has no seam test and no working next step.'
          round: 6
        - id: BR-11
          disposition: addressed
          note: ProjectActionableThreads and BuildThreadInventory both take ThreadProjectionInput, and FromSnapshot keeps records and unreadable together on the production path. The "next omission is a compile error" claim is overstated -- a named-field literal still omits Unreadable -- raised as a Minor.
          round: 6
      findings:
        - id: BR-12
          severity: Critical
          title: The unreadable-record start refusal names two next steps, neither of which works, and has no seam test
          detail: |-
            2nd in family -- do not patch the message. Measured against the real dispatcher with one record
            overwritten as `{"schema_version":99,"nope":`: `couch /repo` exits 1 (run.go:288 renders and returns,
            so the TUI never opens in that repo), `couch --show <tag>` exits 1 with "thread reference not found",
            and `ctrl-space, select it, Tab -> archive` is unreachable from the repository the refusal fires in.
            The working escape -- start couch from a different repository, where the switcher is global -- is
            stated nowhere; and in the version-skew case threadreason.go:36-39 names as the split's whole
            motivation, no record decodes, so every scope is blocked and there is no unblocked repo to start from.
            atlas/couch.md:565-566 states this hazard verbatim as the reason the old rule existed and the new
            paragraph at :546-554 reverses it without answering it. The reason all of this survived: couch.go:350's
            refusal has no test at any seam -- the only coverage is the pure predicate at archive_test.go:269.
            The rule: a refusal that names a command or gesture is pinned by a test that executes that gesture in
            the fixture that produced the refusal and asserts it succeeds. Enumerable today: couch.go:350,
            couch.go:368, startup.go:138 -- only couch.go:368 has one (couch_test.go:1355).
          family: unnavigable-refusal
          round: 6
        - id: BR-13
          severity: Important
          title: ResolveThreadReference still reads snapshot.Records only, so --show reports "not found" for a row --list shows
          detail: |-
            2nd in family -- do not special-case --show. threadmetadata.go:28-34 drops snapshot.Unreadable, so
            every ref-resolving surface (show, name, describe, park, resume, archive-by-ref) answers "thread
            reference not found" about a thread the inventory just printed. The tell that the rule was not stated:
            archive was fixed by ADDING a second resolver (resolveThreadForArchive) that bypasses decoding, rather
            than by making reference resolution total -- a per-consumer patch where a shared rule belongs. State
            it as: every consumer of ThreadSnapshot that answers "does this thread exist" sees the unreadable set.
            Enumeration: 7 `.Snapshot()` sites in couchcore; the four park/start-reconciliation loops legitimately
            filter on record.Park, ResolveThreadReference does not.
          family: decode-failure-drops-the-row
          round: 6
        - id: BR-14
          severity: Important
          title: ReasonInvalid still renders to the operator as "unreadable record", the word the round just gave the other state
          detail: |-
            2nd in family. threadreason.go:100-103 was not updated when menu.go:990-993 was, so one `couch --list`
            can print "unreadable record" for `invalid` and "could not be read - needs a look" for `unreadable`
            side by side. TestEveryReasonHasADistinctOperatorLabel passes because it compares exact strings, and a
            label that borrows another state's defining word clears that bar. The rule: when a state is split,
            every renderer of the old state is re-worded in the same commit, and the vocabulary guard checks
            meaning-collision rather than string equality. Renderers are enumerable: Label(), unusableThreadNotice,
            menu_render.go:286, atlas/couch.md, README.md.
          family: transient-failure-as-verdict
          round: 6
        - id: BR-15
          severity: Minor
          title: Two behavioural claims added this window have no enforcing code, and one contradicts a comment 20 lines away
          detail: |-
            4th in family -- state the rule, do not edit the two comments. (1) threadreason.go:41 says `unreadable`
            is "never archive-eligible": no archive-eligibility rule exists in the tree (DecideRetirement was not
            built), menuActionItems offers archive to it, atlas:551-553 says it CAN be archived on purpose, and
            ReasonUnknown at :59-60 still claims to be "the only one that is never archive-eligible by
            construction". (2) actionableinventory.go:165-172, atlas:556-560 and lessons.md all claim one value
            makes "the next omission a compile error" -- a named-field ThreadProjectionInput literal still omits
            Unreadable silently, and BuildArchivedInventory (threadinventory.go:112) does exactly that. The rule:
            a behavioural claim in a comment, atlas or lesson names the code that enforces it, and is deleted or
            demoted to intent when no such code exists. Enumerable by grepping the window's added prose for
            "never", "always", "cannot" and checking each against an enforcing site plus a test.
          family: unbacked-existing-behavior-claim
          round: 6
      boundary: M3
      blocked: true
    - "n": 7
      timestamp: "2026-09-04T08:31:50-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Enter's switch/resume half is not what the code does (menuThreadActionable excludes ThreadBusy), but no test drives Enter or the "it is busy" arm on a busy row; unchanged this round.
          round: 7
        - id: BR-5
          disposition: addressed
          note: README gained --archived and the blocks-on-unreadable rule; atlas gained the unreadable, label and PathHoldsUsableThread entries; the rule is recorded in lessons.md. Residue is two historical Revisions entries.
          round: 7
        - id: BR-6
          disposition: not-addressed
          note: 'DecideResume now shares occupiedIncarnation, but measured: archiving an unreadable-but-live row quiesces the agent and files the record with no guard, and a park-in-flight row is quiesced before the refusal.'
          round: 7
        - id: BR-12
          disposition: not-addressed
          note: Message and named gestures are fixed and pinned; the guard is not — deleting couch.go:354-370 changes zero test outcomes, and a 12-line seam test goes red without it (verified).
          round: 7
        - id: BR-13
          disposition: addressed
          note: 'Verified by revert: restoring snapshot.Records fails TestARefusalsNamedCommandsActuallyWork with "thread reference not found". Residue: resolveThreadForArchive is now a near-duplicate that could fold back in.'
          round: 7
        - id: BR-14
          disposition: addressed
          note: ReasonInvalid renders "record failed validation", unusableThreadNotice reworded, and the new guard checks meaning-collision — though only over Label(), not unusableThreadNotice.
          round: 7
        - id: BR-15
          disposition: not-addressed
          note: Both claims stand unenforced and were re-stated this window in atlas and lessons.md; "never archive-eligible" is now contradicted by a shipped test that archives an unreadable record.
          round: 7
      findings:
        - id: BR-16
          severity: Minor
          title: 'The switcher offers name and describe on an unreadable row; both fail with the raw decoder error "couch: EOF"'
          detail: |-
            This is the 4th finding in family new-state-unhandled-at-consumers. Do not
            fix the instance — state the rule. Measured through the real dispatcher
            against a record overwritten as {"schema_version":99,"nope": — list and
            show render "unusable: could not be read — may need a newer couch", while
            name and describe both exit 1 with `couch: EOF`, and menuActionItems
            (menu.go:1010-1013) offers both on exactly that row. The rule: when a
            state is added, every consumer that OFFERS an action on a row in that
            state either supports the action or does not offer it, and any refusal it
            produces is couch's own worded message, not a raw decoder error.
            Enumeration: menuActionItems (offers archive/name/describe to every
            non-actionable row, including unreadable and busy), resolveOperationThread,
            ApplyThreadMetadata. Related trap in the same class: the synthesized
            ThreadRecord{Address, Reservation: true} at threadmetadata.go:40 overloads
            a flag ClassifyThread:244 already reads as never-started, so a future
            consumer projecting the resolver's output relabels an unreadable record as
            a known state — the exact conflation this round split apart.
          family: new-state-unhandled-at-consumers
          round: 7
      boundary: M3
      blocked: true
    - "n": 8
      timestamp: "2026-09-04T08:57:26-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Verified by revert on both halves: adding ThreadBusy to menuThreadActionable makes the test fail with a dispatched switch effect, and disabling the new menuActionItems branch fails it with "busy row offers archive".'
          round: 8
        - id: BR-6
          disposition: not-addressed
          note: 'The harms are fixed and pinned, but the rule is not: startup.go:73 and menu.go:968 remain two copies of the state set, and this round made them disagree -- PathHoldsUsableThread returns false for a ThreadBusy row, so a start in flight does not block a second start at that path.'
          round: 8
        - id: BR-12
          disposition: addressed
          note: 'Verified by revert: replacing the guard with `_ = PathHoldsUnreadableThread` fails TestSpawnRefusesWhileAnUnreadableRecordIsInTheRepository. Residue -- startup.go:141''s refusal names `couch --show <tag>` and warmresume_test.go:138 asserts only that the string contains it; nothing executes it.'
          round: 8
        - id: BR-15
          disposition: not-addressed
          note: The site the finding NAMED (threadreason.go:41, "never archive-eligible") still stands, contradicted by menuActionItems and by two shipped tests; only the secondary site at :59 was corrected, while the commit message claims all four artifacts now agree.
          round: 8
        - id: BR-16
          disposition: not-addressed
          note: 'Re-measured through the real dispatcher: name{tag}, name{ref}, describe{tag} and describe{ref} on an unreadable row all exit 1 with the raw "couch: EOF", and menuActionItems still offers both on that row. Unchanged this round.'
          round: 8
      findings:
        - id: BR-17
          severity: Important
          title: UnreadableArchiveWarning is a success delivered on the failure channel, and every consumer reads it as a failed archive
          detail: |-
            5th in family -- do not fix the instance, state the rule. Measured end to end. CLI: archive{tag} on an
            unreadable record exits 1 with "couch: archived ..., but couch could not read its record ...", while
            list reports "no threads" and archived lists the row -- the mutation happened. Switcher (the gesture
            the start refusal names as the recovery path): console.go:1349 sets Success: err == nil, so
            reduceOperationResult (menu.go:1276) takes the failure branch -- red error notice, stays in the archive
            confirmation frame, skips ProjectionPending -- and one refresh later the notice is replaced by "thread
            ... is no longer actionable", so the "a session may still be running" warning, the whole reason the
            value exists, is the one thing the operator loses. The confirmation they accepted reads "archive <label>
            -- stops its session" (menu.go:1063), which this same commit made false for this state. Retrying, the
            natural response to a red error, yields the raw "thread not found: {RepoScope:... Tag:...}". The rule:
            an outcome that is not a failure does not travel on the failure channel -- carry it on the result
            (ArchiveResult{Record, SessionLeftRunning}) so every renderer can show a warning, or update every
            consumer in the same commit. Enumeration is three sites: operationdispatch.go:180 -> exit code,
            console.go:1349's Success, confirmationMenuItems (menu.go:1058-1063). Aggravating: run_test.go:1536
            dropped its `code != 0` assertion and :1584 now passes on an empty error, so the exit-code change is
            both undecided and unpinned.
          family: new-state-unhandled-at-consumers
          round: 8
        - id: BR-18
          severity: Important
          title: The plan's M3 entity tables name a file and a verb that do not exist in the tree
          detail: |-
            5th in family. Verified against the tree, not the prose: cmd/internal/couchcore/retire.go does not
            exist, DecideRetirement and RetirementVerdict are in no file, "couch prune" is in no registry, and
            ThreadStore.Archive shipped as ThreadStore.ArchiveThread -- four rows in the M3 Pure-entities and
            Integration-points tables (plan:871-872, :906-908), four claims, zero backing, while Task 11 and Task 13
            twenty lines below say NOT BUILT. Round 4 raised this as a section-7 plan-revision recommendation and it
            was not actioned, so recommending it again has already failed once. The class rule is BR-15's own: a
            claim in an artifact names the code that backs it or is demoted to intent -- and a greppable entity
            table is the highest-value place to enforce it, because it is what a future agent greps instead of the
            Revisions section. Fix: a `not built -- see Revisions` status on those rows, and the same sweep over the
            M1 table's nine-reason list, which still omits `unreadable` and still carries the archive-eligibility
            clause the code deleted from threadreason.go:59 this round.
          family: unbacked-existing-behavior-claim
          round: 8
      boundary: M3
      blocked: false
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

## Round 6 — 2026-09-04T08:03:03-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Unchanged this round -- menu.go's only edit was unusableThreadNotice; menuActionItems still offers archive to a ThreadBusy row and couchtty still has no ThreadBusy behavioural test.
- BR-4 — addressed — Verified by revert: restoring resolveOperationThread in the archive branch fails TestAnUnreadableRecordCanBeArchivedThroughTheRuntime with "couch: EOF". Row is visible and removable through the real dispatcher.
- BR-5 — not-addressed — The atlas gained the unreadable-record entry, but the same window reversed "debris does not block" and swept none of its restatements: README:319-321, atlas:563-567 (four lines below the new paragraph, and it names the exact hazard now realized), issue Revisions:244-246, and the plan's round-2 entry still says Snapshot carries "Malformed" emitted as "invalid" rows. README's synopsis block still omits --archived.
- BR-6 — not-addressed — Unchanged, and now worse: PathHoldsUnreadableThread is a sixth independent occupancy/actionability predicate. occupiedIncarnation still has one caller; DecideResume (resume.go:80-95) still inlines the same three states.
- BR-8 — addressed — Verified by revert: deleting archivableRecord's Park != nil branch now fails at archive_test.go:106. The startup recency half was fixed in the prior round.
- BR-9 — addressed — TestUncheckedProjectMilestoneHasNoClosedMetadata is red at f9f6cdd6 and green at HEAD (measured both). M3's block carries no closed/actual and its row is unticked; M2 records the missing gate instead of inventing an actual; M1's "judgment estimate, not measured" qualification is restored. sdlc's upsertField inserts after **est:**, so the gate can still write them.
- BR-10 — addressed — ReasonUnreadable is split from ReasonInvalid at the layer where the read fails, and the record still blocks its scope. Two residues raised separately: ReasonInvalid.Label() still says "unreadable record", and the block has no seam test and no working next step.
- BR-11 — addressed — ProjectActionableThreads and BuildThreadInventory both take ThreadProjectionInput, and FromSnapshot keeps records and unreadable together on the production path. The "next omission is a compile error" claim is overstated -- a named-field literal still omits Unreadable -- raised as a Minor.

### Raised

- **BR-12** [Critical] `unnavigable-refusal` The unreadable-record start refusal names two next steps, neither of which works, and has no seam test
  2nd in family -- do not patch the message. Measured against the real dispatcher with one record
  overwritten as `{"schema_version":99,"nope":`: `couch /repo` exits 1 (run.go:288 renders and returns,
  so the TUI never opens in that repo), `couch --show <tag>` exits 1 with "thread reference not found",
  and `ctrl-space, select it, Tab -> archive` is unreachable from the repository the refusal fires in.
  The working escape -- start couch from a different repository, where the switcher is global -- is
  stated nowhere; and in the version-skew case threadreason.go:36-39 names as the split's whole
  motivation, no record decodes, so every scope is blocked and there is no unblocked repo to start from.
  atlas/couch.md:565-566 states this hazard verbatim as the reason the old rule existed and the new
  paragraph at :546-554 reverses it without answering it. The reason all of this survived: couch.go:350's
  refusal has no test at any seam -- the only coverage is the pure predicate at archive_test.go:269.
  The rule: a refusal that names a command or gesture is pinned by a test that executes that gesture in
  the fixture that produced the refusal and asserts it succeeds. Enumerable today: couch.go:350,
  couch.go:368, startup.go:138 -- only couch.go:368 has one (couch_test.go:1355).
- **BR-13** [Important] `decode-failure-drops-the-row` ResolveThreadReference still reads snapshot.Records only, so --show reports "not found" for a row --list shows
  2nd in family -- do not special-case --show. threadmetadata.go:28-34 drops snapshot.Unreadable, so
  every ref-resolving surface (show, name, describe, park, resume, archive-by-ref) answers "thread
  reference not found" about a thread the inventory just printed. The tell that the rule was not stated:
  archive was fixed by ADDING a second resolver (resolveThreadForArchive) that bypasses decoding, rather
  than by making reference resolution total -- a per-consumer patch where a shared rule belongs. State
  it as: every consumer of ThreadSnapshot that answers "does this thread exist" sees the unreadable set.
  Enumeration: 7 `.Snapshot()` sites in couchcore; the four park/start-reconciliation loops legitimately
  filter on record.Park, ResolveThreadReference does not.
- **BR-14** [Important] `transient-failure-as-verdict` ReasonInvalid still renders to the operator as "unreadable record", the word the round just gave the other state
  2nd in family. threadreason.go:100-103 was not updated when menu.go:990-993 was, so one `couch --list`
  can print "unreadable record" for `invalid` and "could not be read - needs a look" for `unreadable`
  side by side. TestEveryReasonHasADistinctOperatorLabel passes because it compares exact strings, and a
  label that borrows another state's defining word clears that bar. The rule: when a state is split,
  every renderer of the old state is re-worded in the same commit, and the vocabulary guard checks
  meaning-collision rather than string equality. Renderers are enumerable: Label(), unusableThreadNotice,
  menu_render.go:286, atlas/couch.md, README.md.
- **BR-15** [Minor] `unbacked-existing-behavior-claim` Two behavioural claims added this window have no enforcing code, and one contradicts a comment 20 lines away
  4th in family -- state the rule, do not edit the two comments. (1) threadreason.go:41 says `unreadable`
  is "never archive-eligible": no archive-eligibility rule exists in the tree (DecideRetirement was not
  built), menuActionItems offers archive to it, atlas:551-553 says it CAN be archived on purpose, and
  ReasonUnknown at :59-60 still claims to be "the only one that is never archive-eligible by
  construction". (2) actionableinventory.go:165-172, atlas:556-560 and lessons.md all claim one value
  makes "the next omission a compile error" -- a named-field ThreadProjectionInput literal still omits
  Unreadable silently, and BuildArchivedInventory (threadinventory.go:112) does exactly that. The rule:
  a behavioural claim in a comment, atlas or lesson names the code that enforces it, and is deleted or
  demoted to intent when no such code exists. Enumerable by grepping the window's added prose for
  "never", "always", "cannot" and checking each against an enforcing site plus a test.

## Round 7 — 2026-09-04T08:31:50-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Enter's switch/resume half is not what the code does (menuThreadActionable excludes ThreadBusy), but no test drives Enter or the "it is busy" arm on a busy row; unchanged this round.
- BR-5 — addressed — README gained --archived and the blocks-on-unreadable rule; atlas gained the unreadable, label and PathHoldsUsableThread entries; the rule is recorded in lessons.md. Residue is two historical Revisions entries.
- BR-6 — not-addressed — DecideResume now shares occupiedIncarnation, but measured: archiving an unreadable-but-live row quiesces the agent and files the record with no guard, and a park-in-flight row is quiesced before the refusal.
- BR-12 — not-addressed — Message and named gestures are fixed and pinned; the guard is not — deleting couch.go:354-370 changes zero test outcomes, and a 12-line seam test goes red without it (verified).
- BR-13 — addressed — Verified by revert: restoring snapshot.Records fails TestARefusalsNamedCommandsActuallyWork with "thread reference not found". Residue: resolveThreadForArchive is now a near-duplicate that could fold back in.
- BR-14 — addressed — ReasonInvalid renders "record failed validation", unusableThreadNotice reworded, and the new guard checks meaning-collision — though only over Label(), not unusableThreadNotice.
- BR-15 — not-addressed — Both claims stand unenforced and were re-stated this window in atlas and lessons.md; "never archive-eligible" is now contradicted by a shipped test that archives an unreadable record.

### Raised

- **BR-16** [Minor] `new-state-unhandled-at-consumers` The switcher offers name and describe on an unreadable row; both fail with the raw decoder error "couch: EOF"
  This is the 4th finding in family new-state-unhandled-at-consumers. Do not
  fix the instance — state the rule. Measured through the real dispatcher
  against a record overwritten as {"schema_version":99,"nope": — list and
  show render "unusable: could not be read — may need a newer couch", while
  name and describe both exit 1 with `couch: EOF`, and menuActionItems
  (menu.go:1010-1013) offers both on exactly that row. The rule: when a
  state is added, every consumer that OFFERS an action on a row in that
  state either supports the action or does not offer it, and any refusal it
  produces is couch's own worded message, not a raw decoder error.
  Enumeration: menuActionItems (offers archive/name/describe to every
  non-actionable row, including unreadable and busy), resolveOperationThread,
  ApplyThreadMetadata. Related trap in the same class: the synthesized
  ThreadRecord{Address, Reservation: true} at threadmetadata.go:40 overloads
  a flag ClassifyThread:244 already reads as never-started, so a future
  consumer projecting the resolver's output relabels an unreadable record as
  a known state — the exact conflation this round split apart.

## Round 8 — 2026-09-04T08:57:26-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Verified by revert on both halves: adding ThreadBusy to menuThreadActionable makes the test fail with a dispatched switch effect, and disabling the new menuActionItems branch fails it with "busy row offers archive".
- BR-6 — not-addressed — The harms are fixed and pinned, but the rule is not: startup.go:73 and menu.go:968 remain two copies of the state set, and this round made them disagree -- PathHoldsUsableThread returns false for a ThreadBusy row, so a start in flight does not block a second start at that path.
- BR-12 — addressed — Verified by revert: replacing the guard with `_ = PathHoldsUnreadableThread` fails TestSpawnRefusesWhileAnUnreadableRecordIsInTheRepository. Residue -- startup.go:141's refusal names `couch --show <tag>` and warmresume_test.go:138 asserts only that the string contains it; nothing executes it.
- BR-15 — not-addressed — The site the finding NAMED (threadreason.go:41, "never archive-eligible") still stands, contradicted by menuActionItems and by two shipped tests; only the secondary site at :59 was corrected, while the commit message claims all four artifacts now agree.
- BR-16 — not-addressed — Re-measured through the real dispatcher: name{tag}, name{ref}, describe{tag} and describe{ref} on an unreadable row all exit 1 with the raw "couch: EOF", and menuActionItems still offers both on that row. Unchanged this round.

### Raised

- **BR-17** [Important] `new-state-unhandled-at-consumers` UnreadableArchiveWarning is a success delivered on the failure channel, and every consumer reads it as a failed archive
  5th in family -- do not fix the instance, state the rule. Measured end to end. CLI: archive{tag} on an
  unreadable record exits 1 with "couch: archived ..., but couch could not read its record ...", while
  list reports "no threads" and archived lists the row -- the mutation happened. Switcher (the gesture
  the start refusal names as the recovery path): console.go:1349 sets Success: err == nil, so
  reduceOperationResult (menu.go:1276) takes the failure branch -- red error notice, stays in the archive
  confirmation frame, skips ProjectionPending -- and one refresh later the notice is replaced by "thread
  ... is no longer actionable", so the "a session may still be running" warning, the whole reason the
  value exists, is the one thing the operator loses. The confirmation they accepted reads "archive <label>
  -- stops its session" (menu.go:1063), which this same commit made false for this state. Retrying, the
  natural response to a red error, yields the raw "thread not found: {RepoScope:... Tag:...}". The rule:
  an outcome that is not a failure does not travel on the failure channel -- carry it on the result
  (ArchiveResult{Record, SessionLeftRunning}) so every renderer can show a warning, or update every
  consumer in the same commit. Enumeration is three sites: operationdispatch.go:180 -> exit code,
  console.go:1349's Success, confirmationMenuItems (menu.go:1058-1063). Aggravating: run_test.go:1536
  dropped its `code != 0` assertion and :1584 now passes on an empty error, so the exit-code change is
  both undecided and unpinned.
- **BR-18** [Important] `unbacked-existing-behavior-claim` The plan's M3 entity tables name a file and a verb that do not exist in the tree
  5th in family. Verified against the tree, not the prose: cmd/internal/couchcore/retire.go does not
  exist, DecideRetirement and RetirementVerdict are in no file, "couch prune" is in no registry, and
  ThreadStore.Archive shipped as ThreadStore.ArchiveThread -- four rows in the M3 Pure-entities and
  Integration-points tables (plan:871-872, :906-908), four claims, zero backing, while Task 11 and Task 13
  twenty lines below say NOT BUILT. Round 4 raised this as a section-7 plan-revision recommendation and it
  was not actioned, so recommending it again has already failed once. The class rule is BR-15's own: a
  claim in an artifact names the code that backs it or is demoted to intent -- and a greppable entity
  table is the highest-value place to enforce it, because it is what a future agent greps instead of the
  Revisions section. Fix: a `not built -- see Revisions` status on those rows, and the same sweep over the
  M1 table's nine-reason list, which still omits `unreadable` and still carries the archive-eligibility
  clause the code deleted from threadreason.go:59 this round.

## Open findings

- **BR-6** [Important] `new-state-unhandled-at-consumers` Occupancy is decided in five places with four different definitions
- **BR-15** [Minor] `unbacked-existing-behavior-claim` Two behavioural claims added this window have no enforcing code, and one contradicts a comment 20 lines away
- **BR-16** [Minor] `new-state-unhandled-at-consumers` The switcher offers name and describe on an unreadable row; both fail with the raw decoder error "couch: EOF"
- **BR-17** [Important] `new-state-unhandled-at-consumers` UnreadableArchiveWarning is a success delivered on the failure channel, and every consumer reads it as a failed archive
- **BR-18** [Important] `unbacked-existing-behavior-claim` The plan's M3 entity tables name a file and a verb that do not exist in the tree
