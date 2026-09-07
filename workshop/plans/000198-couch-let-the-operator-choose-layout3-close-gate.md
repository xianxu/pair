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

## Open findings

- **BR-1** [Important] `persisted-decode-compat-unverified` Task 3 claims an older binary ignores the new layout field, but strictjson rejects unknown fields
- **BR-2** [Minor] `store-mutation-unspecified` No named test pins that a warm reattach leaves the layout witness unchanged
- **BR-3** [Important] `refusal-must-name-a-runnable-remedy` layoutConflictRefusal prints `couch --unknown` when conflicts disagree or carry LayoutUnknown
- **BR-4** [Important] `operator-surface-undocumented` README update missing for the new `--layout2|--layout3` flag
- **BR-5** [Important] `vocabulary-duplicated-across-packages` couchcore.Layout re-declares launcher.LayoutMode; nothing pins that pair parses the emitted flag
- **BR-6** [Important] `untagged-empty-sentinel` Empty Layout means three different things, and one of them formats as a bare `--`
- **BR-7** [Minor] `stale-layout-pin-comment` Three comments still name `--layout2` as the flag couch sends or drops
- **BR-8** [Minor] `on-disk-format-untested` No JSON-level test pins the persisted `layout` key
- **BR-9** [Minor] `doubled-error-prefix` The refusal renders as "couch: couch cannot start in layout3"
- **BR-10** [Minor] `plan-names-files-that-do-not-exist` Plan cites test files the work did not create
