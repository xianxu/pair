---
gate: boundary-review
issue: 198
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-06T17:59:46-07:00"
      agent: sdlc
      findings:
        - id: BR-1
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
            (carried from plan-quality PQ-9, deferred to the boundary review)
          family: persisted-decode-compat-unverified
          round: 1
        - id: BR-2
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
            (carried from plan-quality PQ-10, deferred to the boundary review)
          family: store-mutation-unspecified
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-06T17:59:46-07:00"
      agent: claude
      findings:
        - id: BR-3
          severity: Important
          title: layoutConflictRefusal prints `couch --unknown` when conflicts disagree or carry LayoutUnknown
          detail: |-
            cmd/internal/couchcore/layout.go:112-127 degrades `other` to LayoutUnknown and
            interpolates it into "park them first: couch --%s". Reachable via a hand-edited or
            newer-version witness (the case LayoutUnknown exists to surface visibly) and via a
            mixed blocking set after the process-death-mid-CAS event the plan documents. In the
            mixed case there is no single correct layout to suggest, so the message's premise
            fails, not just its wording. No test renders the refusal for either case.
          family: refusal-must-name-a-runnable-remedy
          round: 2
        - id: BR-4
          severity: Important
          title: README update missing for the new `--layout2|--layout3` flag
          detail: |-
            usage() gained the flag (run.go:672,681) and a test pins that, but README.md:267-270
            still shows `couch [<repo>]` with no layout line. The repo already owns the
            enumeration -- readme_test.go:129 TestREADMEDocumentsTheOperatorFacingSurface covers
            exactly "flags that change what couch does with the operator's terminal" -- and it was
            not extended, so the guard passed vacuously. Fix both the README block and that test's
            want list (ARCH-PURPOSE: the class, not the instance).
          family: operator-surface-undocumented
          round: 2
        - id: BR-5
          severity: Important
          title: couchcore.Layout re-declares launcher.LayoutMode; nothing pins that pair parses the emitted flag
          detail: |-
            launcher/layout.go:8-11 already owns layout2/layout3, ParseLayoutMode, and the
            "default is Layout2" rule; launcher/args.go:156-158 owns the accepted flag spellings.
            couchcore imports launcher in ~8 non-test files, so there is no cycle constraint.
            The cross-binary contract couch now depends on is evidenced only by a prose comment at
            couch.go:450-458 saying it was measured by hand. Minimum: a conformance test asserting
            launcher.ParseArgs("resume","tag",Layout3.Flag()) yields {Mode: launcher.Layout3,
            Explicit: true}. Better: alias the type so a third layout is one edit in one place
            (ARCH-DRY, ARCH-MOCK).
          family: vocabulary-duplicated-across-packages
          round: 2
        - id: BR-6
          severity: Important
          title: Empty Layout means three different things, and one of them formats as a bare `--`
          detail: |-
            ThreadRecord.Layout=="" means pre-#198/layout2; StartEvent.Layout=="" means
            warm/do-not-record (starttransaction.go:36-39, launch_existing.go:124-131);
            Couch.Layout=="" means constructed without New, and Flag() (layout.go:54) emits a bare
            "--" into argv. The plan calls the third unreachable, but Couch is built by struct
            literal in ~10 test files, so the invariant is comment-enforced on an exported field.
            #199/#200 will consume this surface. Make Flag() total, and give StartEvent a *Layout
            or an explicit RecordLayout bool (ARCH-ORDER).
          family: untagged-empty-sentinel
          round: 2
        - id: BR-7
          severity: Minor
          title: Three comments still name `--layout2` as the flag couch sends or drops
          detail: |-
            couch.go:424 opens the rationale block with "`pair resume <tag> --layout2` rather than
            a bare `pair`", contradicting the new rule below it. launch_existing.go:35,46 say
            "`--layout2` is dropped" where they now mean the layout flag. detach.go:27 describes a
            warm reattach as "a fresh `pair resume <tag> --layout2`", which is the inverse of the
            #179 invariant (pre-existing, but this was the sweep).
          family: stale-layout-pin-comment
          round: 2
        - id: BR-8
          severity: Minor
          title: No JSON-level test pins the persisted `layout` key
          detail: |-
            layout_projection_test.go:74-95 round-trips Go structs through
            toPersistedThreadRecord/fromPersistedThreadRecord. The plan's Task 3 test unmarshalled
            a raw `{"schema_version":2,...}` record instead. As written, renaming the json tag in
            threadrecord/record.go:81 would keep the suite green while silently dropping every
            witness on disk.
          family: on-disk-format-untested
          round: 2
        - id: BR-9
          severity: Minor
          title: 'The refusal renders as "couch: couch cannot start in layout3"'
          detail: |-
            renderError (run.go:665) already prefixes "couch: ". startupResumeRefusal avoids the
            stutter by leading with %w; the new message leads with the literal word "couch".
          family: doubled-error-prefix
          round: 2
        - id: BR-10
          severity: Minor
          title: Plan cites test files the work did not create
          detail: |-
            Task 3 names thread_test.go and Task 5 names startup_test.go; the tests landed in
            layout_projection_test.go and layout_guard_test.go (a better grouping). Task 2's
            fixtures use PairTag where the code uses ThreadTag. Worth a "## Revisions" entry so
            the plan stops describing paths the tree does not have.
          family: plan-names-files-that-do-not-exist
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-06T18:20:54-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Corrected in plan Revisions section 4 and at thread.go:74-79; forward-only compatibility now stated, with TestPre198RecordDecodesThroughTheProductionPath pinning the old-record direction.
          round: 3
        - id: BR-2
          disposition: addressed
          note: 'Verified by reverting: recording the layout unconditionally in the StartRegistered arm turns TestWarmReattachSendsNoLayoutEvenInLayout3 red on the witness assertion.'
          round: 3
        - id: BR-3
          disposition: not-addressed
          note: Code fix is correct and present, but no test fails without it -- deleting hostLayoutFor's agreement loop leaves TestRefusalNeverNamesACommandThatWouldNotRun green, because a total Flag() renders LayoutUnknown as --layout2.
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: README content added, but the guard's new want strings are satisfied by pair's own flag docs at README.md:13,18,441 -- deleting README.md:268-269 leaves TestREADMEDocumentsTheOperatorFacingSurface green.
          round: 3
        - id: BR-5
          disposition: addressed
          note: Type aliased to launcher.LayoutMode, Flag() moved beside the parser, and the conformance test goes red when launchArgsAcceptLayout stops admitting resume.
          round: 3
        - id: BR-6
          disposition: addressed
          note: StartEvent.Layout is a *Layout and Flag() is total; TestFlagIsTotalOverUnsetAndUnknownLayouts plus the warm-reattach witness assertion pin both halves.
          round: 3
        - id: BR-7
          disposition: addressed
          note: couch.go:424, launch_existing.go:35,46 and detach.go:27 all now name the layout flag generically rather than --layout2.
          round: 3
        - id: BR-8
          disposition: addressed
          note: TestLayoutWitnessPersistsUnderItsOnDiskKey goes red when the json tag is renamed to layout_mode.
          round: 3
        - id: BR-9
          disposition: addressed
          note: Message now leads with "cannot start in ..."; TestRefusalDoesNotDoubleTheProgramPrefix pins it.
          round: 3
        - id: BR-10
          disposition: addressed
          note: Plan Revisions records the test-file grouping decision and the PairTag/ThreadTag fixture naming.
          round: 3
      findings:
        - id: BR-11
          severity: Important
          title: Two of the nine tests added to close round 2's findings pass with their fix reverted
          detail: |-
            Measured prevalence: 2 of 9 tests added by 70dd36c7. Deleting hostLayoutFor's
            agreement loop (layout.go:116-136) leaves layout_test.go:170 green, because
            BR-6's total Flag() renders LayoutUnknown as --layout2, so the message is
            well-formed but tells an operator blocked by a mixed set to run a couch that
            would itself refuse. Deleting README.md:268-269 leaves readme_test.go:144-145
            green, because pair's own flag docs at README.md:13,18,441 already contain
            those substrings. The rule, not the two edits, is the deliverable: a test
            written to close a finding is not done until the fix is reverted and it goes
            red, and a whole-document substring guard must anchor on a string unique to
            the surface it guards -- README's sibling test already uses the
            command-prefixed form ("couch --list"). Concretely: assert the NEGATIVE
            direction of the refusal (no concrete `couch --layoutN` host remedy when no
            single host exists, only the requested layout), and switch the README want
            strings to "couch --layout2"/"couch --layout3".
          family: guard-passes-without-the-fix
          round: 3
        - id: BR-12
          severity: Minor
          title: A blocking set that is entirely LayoutUnknown leaves the operator with no reachable remedy at all
          detail: |-
            This is the 2nd finding in family refusal-must-name-a-runnable-remedy. Do not
            fix this instance -- the covering rule is that a refusal must terminate in an
            action reachable with the tools the operator has, and when no in-tool action
            exists it must name the out-of-tool one. `park` is PresentationTUI/RowAction
            (ops.go:189), so it needs a running couch, and with a lone LayoutUnknown
            session-holder every `couch --layoutN` refuses. The fallback at layout.go:160
            says "from whichever couch can host it" when there is none; it should name the
            zellij session to kill or the record to repair. Reachable only via a
            hand-edited or newer-version witness, hence Minor.
          family: refusal-must-name-a-runnable-remedy
          round: 3
        - id: BR-13
          severity: Minor
          title: ActionableThreadSummary.Layout is documented as normalized but the Unreadable branch never normalizes
          detail: |-
            This is the 2nd finding in family untagged-empty-sentinel. Do not fix this
            instance -- the covering rule is that a field documented as normalized must be
            normalized at every construction site of its struct. actionableinventory.go:208-212
            builds the input.Unreadable rows without touching Layout, so they carry a raw
            Layout("") while the field comment (actionableinventory.go:121-125) promises
            Layout2 or LayoutUnknown. Measured prevalence: 2 construction sites in
            ProjectActionableThreads, 1 unnormalized. Harmless today because ThreadUnusable
            never holds a session, but pair#199/#200 consume this struct.
          family: untagged-empty-sentinel
          round: 3
      blocked: true
---

# Gate ledger — pair#198 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-06T17:59:46-07:00 (sdlc) — passed

### Raised

- **BR-1** [Important] `persisted-decode-compat-unverified` Task 3 claims an older binary ignores the new layout field, but strictjson rejects unknown fields
  Persisted records load via threadrecord.DecodePersisted -> strictjson.Decode
  (record.go:201), which sets DisallowUnknownFields (strictjson/decode.go:23).
  compat_test.go:13-18 documents this exact hazard and the tombstone discipline
  built for it. Since Task 4 writes the witness on every StartRegistered, once
  the feature runs, reverting couch makes every touched record undecodable and
  those threads vanish into ThreadUnusable/ReasonUnreadable rows. The no-bump
  decision stands on its first reason; correct the second sentence and add the
  read-by-an-older-version direction to the ARCH-SECURE analysis, which
  currently covers only written-by-an-older-version.
  (carried from plan-quality PQ-9, deferred to the boundary review)
- **BR-2** [Minor] `store-mutation-unspecified` No named test pins that a warm reattach leaves the layout witness unchanged
  This is the 2nd finding in family store-mutation-unspecified; the covering
  rule is that every store-mutation invariant the plan states in prose needs a
  named test asserting it, not just the argv it implies. Task 4 requires the
  layout be passed only when !in.Warm, but StartEvent.Layout's zero value is
  Layout(""), which NormalizeLayout maps back to Layout2 -- so an unconditional
  apply in the StartRegistered arm silently downgrades a layout3 thread's
  witness on its first reattach and no planned test would catch it. Extend
  TestWarmReattachSendsNoLayoutEvenInLayout3 to assert the witness too.
  (carried from plan-quality PQ-10, deferred to the boundary review)

## Round 2 — 2026-09-06T17:59:46-07:00 (claude) — BLOCKED

### Raised

- **BR-3** [Important] `refusal-must-name-a-runnable-remedy` layoutConflictRefusal prints `couch --unknown` when conflicts disagree or carry LayoutUnknown
  cmd/internal/couchcore/layout.go:112-127 degrades `other` to LayoutUnknown and
  interpolates it into "park them first: couch --%s". Reachable via a hand-edited or
  newer-version witness (the case LayoutUnknown exists to surface visibly) and via a
  mixed blocking set after the process-death-mid-CAS event the plan documents. In the
  mixed case there is no single correct layout to suggest, so the message's premise
  fails, not just its wording. No test renders the refusal for either case.
- **BR-4** [Important] `operator-surface-undocumented` README update missing for the new `--layout2|--layout3` flag
  usage() gained the flag (run.go:672,681) and a test pins that, but README.md:267-270
  still shows `couch [<repo>]` with no layout line. The repo already owns the
  enumeration -- readme_test.go:129 TestREADMEDocumentsTheOperatorFacingSurface covers
  exactly "flags that change what couch does with the operator's terminal" -- and it was
  not extended, so the guard passed vacuously. Fix both the README block and that test's
  want list (ARCH-PURPOSE: the class, not the instance).
- **BR-5** [Important] `vocabulary-duplicated-across-packages` couchcore.Layout re-declares launcher.LayoutMode; nothing pins that pair parses the emitted flag
  launcher/layout.go:8-11 already owns layout2/layout3, ParseLayoutMode, and the
  "default is Layout2" rule; launcher/args.go:156-158 owns the accepted flag spellings.
  couchcore imports launcher in ~8 non-test files, so there is no cycle constraint.
  The cross-binary contract couch now depends on is evidenced only by a prose comment at
  couch.go:450-458 saying it was measured by hand. Minimum: a conformance test asserting
  launcher.ParseArgs("resume","tag",Layout3.Flag()) yields {Mode: launcher.Layout3,
  Explicit: true}. Better: alias the type so a third layout is one edit in one place
  (ARCH-DRY, ARCH-MOCK).
- **BR-6** [Important] `untagged-empty-sentinel` Empty Layout means three different things, and one of them formats as a bare `--`
  ThreadRecord.Layout=="" means pre-#198/layout2; StartEvent.Layout=="" means
  warm/do-not-record (starttransaction.go:36-39, launch_existing.go:124-131);
  Couch.Layout=="" means constructed without New, and Flag() (layout.go:54) emits a bare
  "--" into argv. The plan calls the third unreachable, but Couch is built by struct
  literal in ~10 test files, so the invariant is comment-enforced on an exported field.
  #199/#200 will consume this surface. Make Flag() total, and give StartEvent a *Layout
  or an explicit RecordLayout bool (ARCH-ORDER).
- **BR-7** [Minor] `stale-layout-pin-comment` Three comments still name `--layout2` as the flag couch sends or drops
  couch.go:424 opens the rationale block with "`pair resume <tag> --layout2` rather than
  a bare `pair`", contradicting the new rule below it. launch_existing.go:35,46 say
  "`--layout2` is dropped" where they now mean the layout flag. detach.go:27 describes a
  warm reattach as "a fresh `pair resume <tag> --layout2`", which is the inverse of the
  #179 invariant (pre-existing, but this was the sweep).
- **BR-8** [Minor] `on-disk-format-untested` No JSON-level test pins the persisted `layout` key
  layout_projection_test.go:74-95 round-trips Go structs through
  toPersistedThreadRecord/fromPersistedThreadRecord. The plan's Task 3 test unmarshalled
  a raw `{"schema_version":2,...}` record instead. As written, renaming the json tag in
  threadrecord/record.go:81 would keep the suite green while silently dropping every
  witness on disk.
- **BR-9** [Minor] `doubled-error-prefix` The refusal renders as "couch: couch cannot start in layout3"
  renderError (run.go:665) already prefixes "couch: ". startupResumeRefusal avoids the
  stutter by leading with %w; the new message leads with the literal word "couch".
- **BR-10** [Minor] `plan-names-files-that-do-not-exist` Plan cites test files the work did not create
  Task 3 names thread_test.go and Task 5 names startup_test.go; the tests landed in
  layout_projection_test.go and layout_guard_test.go (a better grouping). Task 2's
  fixtures use PairTag where the code uses ThreadTag. Worth a "## Revisions" entry so
  the plan stops describing paths the tree does not have.

## Round 3 — 2026-09-06T18:20:54-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Corrected in plan Revisions section 4 and at thread.go:74-79; forward-only compatibility now stated, with TestPre198RecordDecodesThroughTheProductionPath pinning the old-record direction.
- BR-2 — addressed — Verified by reverting: recording the layout unconditionally in the StartRegistered arm turns TestWarmReattachSendsNoLayoutEvenInLayout3 red on the witness assertion.
- BR-3 — not-addressed — Code fix is correct and present, but no test fails without it -- deleting hostLayoutFor's agreement loop leaves TestRefusalNeverNamesACommandThatWouldNotRun green, because a total Flag() renders LayoutUnknown as --layout2.
- BR-4 — not-addressed — README content added, but the guard's new want strings are satisfied by pair's own flag docs at README.md:13,18,441 -- deleting README.md:268-269 leaves TestREADMEDocumentsTheOperatorFacingSurface green.
- BR-5 — addressed — Type aliased to launcher.LayoutMode, Flag() moved beside the parser, and the conformance test goes red when launchArgsAcceptLayout stops admitting resume.
- BR-6 — addressed — StartEvent.Layout is a *Layout and Flag() is total; TestFlagIsTotalOverUnsetAndUnknownLayouts plus the warm-reattach witness assertion pin both halves.
- BR-7 — addressed — couch.go:424, launch_existing.go:35,46 and detach.go:27 all now name the layout flag generically rather than --layout2.
- BR-8 — addressed — TestLayoutWitnessPersistsUnderItsOnDiskKey goes red when the json tag is renamed to layout_mode.
- BR-9 — addressed — Message now leads with "cannot start in ..."; TestRefusalDoesNotDoubleTheProgramPrefix pins it.
- BR-10 — addressed — Plan Revisions records the test-file grouping decision and the PairTag/ThreadTag fixture naming.

### Raised

- **BR-11** [Important] `guard-passes-without-the-fix` Two of the nine tests added to close round 2's findings pass with their fix reverted
  Measured prevalence: 2 of 9 tests added by 70dd36c7. Deleting hostLayoutFor's
  agreement loop (layout.go:116-136) leaves layout_test.go:170 green, because
  BR-6's total Flag() renders LayoutUnknown as --layout2, so the message is
  well-formed but tells an operator blocked by a mixed set to run a couch that
  would itself refuse. Deleting README.md:268-269 leaves readme_test.go:144-145
  green, because pair's own flag docs at README.md:13,18,441 already contain
  those substrings. The rule, not the two edits, is the deliverable: a test
  written to close a finding is not done until the fix is reverted and it goes
  red, and a whole-document substring guard must anchor on a string unique to
  the surface it guards -- README's sibling test already uses the
  command-prefixed form ("couch --list"). Concretely: assert the NEGATIVE
  direction of the refusal (no concrete `couch --layoutN` host remedy when no
  single host exists, only the requested layout), and switch the README want
  strings to "couch --layout2"/"couch --layout3".
- **BR-12** [Minor] `refusal-must-name-a-runnable-remedy` A blocking set that is entirely LayoutUnknown leaves the operator with no reachable remedy at all
  This is the 2nd finding in family refusal-must-name-a-runnable-remedy. Do not
  fix this instance -- the covering rule is that a refusal must terminate in an
  action reachable with the tools the operator has, and when no in-tool action
  exists it must name the out-of-tool one. `park` is PresentationTUI/RowAction
  (ops.go:189), so it needs a running couch, and with a lone LayoutUnknown
  session-holder every `couch --layoutN` refuses. The fallback at layout.go:160
  says "from whichever couch can host it" when there is none; it should name the
  zellij session to kill or the record to repair. Reachable only via a
  hand-edited or newer-version witness, hence Minor.
- **BR-13** [Minor] `untagged-empty-sentinel` ActionableThreadSummary.Layout is documented as normalized but the Unreadable branch never normalizes
  This is the 2nd finding in family untagged-empty-sentinel. Do not fix this
  instance -- the covering rule is that a field documented as normalized must be
  normalized at every construction site of its struct. actionableinventory.go:208-212
  builds the input.Unreadable rows without touching Layout, so they carry a raw
  Layout("") while the field comment (actionableinventory.go:121-125) promises
  Layout2 or LayoutUnknown. Measured prevalence: 2 construction sites in
  ProjectActionableThreads, 1 unnormalized. Harmless today because ThreadUnusable
  never holds a session, but pair#199/#200 consume this struct.

## Open findings

- **BR-3** [Important] `refusal-must-name-a-runnable-remedy` layoutConflictRefusal prints `couch --unknown` when conflicts disagree or carry LayoutUnknown
- **BR-4** [Important] `operator-surface-undocumented` README update missing for the new `--layout2|--layout3` flag
- **BR-11** [Important] `guard-passes-without-the-fix` Two of the nine tests added to close round 2's findings pass with their fix reverted
- **BR-12** [Minor] `refusal-must-name-a-runnable-remedy` A blocking set that is entirely LayoutUnknown leaves the operator with no reachable remedy at all
- **BR-13** [Minor] `untagged-empty-sentinel` ActionableThreadSummary.Layout is documented as normalized but the Unreadable branch never normalizes
