---
gate: boundary-review
issue: 282
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-18T12:34:52-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: DecodeOuterRecord, the only untrusted-input parser here, is pinned by six literal cases instead of a fuzz
          detail: |-
            The record is truncatable, hand-editable and cross-version; one FuzzDecodeOuterRecord seeded with the
            legacy one-line form plus the listed malformed strings, asserting no panic and Couch==true only for an
            exact two-line presenter=couch record, covers the class the table cannot. In the same paragraph, the
            ARCH-SECURE claim "a hand edit can never fabricate a Couch section" is false: a well-formed hand-written
            two-line record does exactly that. What strictness buys is visible degradation on a MALFORMED record.
            (carried from plan-quality PQ-1, deferred to the boundary review)
          family: parser-fuzz-over-enumerated-cases
          round: 1
        - id: BR-2
          severity: Minor
          title: Task 3 asserts hosting changes only Alt+d/Alt+n/Ctrl+Alt+n; compaction.go:90 also reroutes Alt+Shift+C
          detail: |-
            When hosted, runCompaction delegates the continuation to Couch instead of restarting in place. No help row
            is wrong, because Alt+Shift+C's wording comes from nvim's "compact session" desc and ChordAltShiftC is not
            a GlobalBinding, so HostedHelp structurally cannot carry it. Say that, rather than claiming sameness.
            (carried from plan-quality PQ-2, deferred to the boundary review)
          family: unbacked-behavior-claim
          round: 1
        - id: BR-3
          severity: Minor
          title: The doc sweep's line-numbered anchors are invalidated by the sweep's own earlier edits
          detail: |-
            Task 8 Step 2 replaces README 378-381 with a shorter passage before the implementer reaches the 496-497
            anchor, so the later citations drift. Content anchors survive edits; line numbers do not.
            (carried from plan-quality PQ-3, deferred to the boundary review)
          family: stale-line-number-anchors
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-18T12:34:52-07:00"
      agent: claude
      findings:
        - id: BR-4
          severity: Minor
          title: Hosted Alt+n row is 124 cols wide and silently disables centering for the whole page
          detail: |-
            Measured: standalone page widest line = 104 cols, hosted/Couch page = 124
            ("Alt+n does not reload under Couch and may end the thread; relaunch from the
            Couch switcher (outside the agent pane)"). keyhelp.Center pads from the widest
            line, so on a <=124-col terminal centering vanishes for every section and the
            row wraps under less. Shorten the HostedHelp wording.
          family: help-row-width-budget
          round: 2
        - id: BR-5
          severity: Minor
          title: couchtty.menuControls still restates Couch's chord inventory by hand
          detail: |-
            couchkeys is now the source for Couch's chords, but menu.go:22 keeps a parallel
            list of the same six chords with its own Action wording (Alt+d reads "detach this
            thread · all + leave couch here" against couchkeys' "detach every live thread and
            leave Couch"). Only TestREADMEDocumentsEveryPanelControl consumes it, and its
            comment claiming the panel renderer consumes it too is stale. Derive the chord
            rows from couchkeys.Bindings() or drop them.
          family: couch-chord-single-source
          round: 2
        - id: BR-6
          severity: Minor
          title: The "Couch switcher" section omits Ctrl+Space's in-switcher meaning
          detail: |-
            console.go:1314 gives Ctrl+Space a second meaning when the panel has focus
            (KeyCtrlSpace = start a thread), but couchkeys declares Ctrl+Space only at
            ScopeEveryPane, so neither `couch --help` nor Alt+h documents it while the
            section title reads as the switcher's key list. A switcher-scope row would
            follow the page's own (key, context) rule.
          family: scope-section-key-coverage
          round: 2
        - id: BR-7
          severity: Minor
          title: keyscmd.Deps panics on a nil seam, defeating the always-exit-0 contract
          detail: |-
            RunWith calls deps.CouchPresents() and passes deps.Getenv unconditionally. A
            caller constructing Deps{Sources: ...} panics; a non-zero exit from `pair keys`
            kills the floating pane before less opens under bin/pair-help's set -euo
            pipefail -- the dead-help-key failure #132 fixed. Default nil Getenv to
            os.Getenv and nil CouchPresents to a false func.
          family: seam-struct-requires-nil-defaults
          round: 2
        - id: BR-8
          severity: Minor
          title: PresentedByCouch matches on tag alone where sibling predicates also match scope
          detail: |-
            createflow.go:511 uses scope+tag and lifecycle.go:105 uses tag+valid-scope, but
            outerrecord.go:54 uses tag only. A `pair resume <same-tag>` run from inside a
            Couch thread's right terminal inherits COUCH_THREAD_TAG and records
            presenter=couch. Harmless today; tighten or record why tag alone is the rule.
          family: couch-ownership-predicate-variants
          round: 2
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#282 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-18T12:34:52-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `parser-fuzz-over-enumerated-cases` DecodeOuterRecord, the only untrusted-input parser here, is pinned by six literal cases instead of a fuzz
  The record is truncatable, hand-editable and cross-version; one FuzzDecodeOuterRecord seeded with the
  legacy one-line form plus the listed malformed strings, asserting no panic and Couch==true only for an
  exact two-line presenter=couch record, covers the class the table cannot. In the same paragraph, the
  ARCH-SECURE claim "a hand edit can never fabricate a Couch section" is false: a well-formed hand-written
  two-line record does exactly that. What strictness buys is visible degradation on a MALFORMED record.
  (carried from plan-quality PQ-1, deferred to the boundary review)
- **BR-2** [Minor] `unbacked-behavior-claim` Task 3 asserts hosting changes only Alt+d/Alt+n/Ctrl+Alt+n; compaction.go:90 also reroutes Alt+Shift+C
  When hosted, runCompaction delegates the continuation to Couch instead of restarting in place. No help row
  is wrong, because Alt+Shift+C's wording comes from nvim's "compact session" desc and ChordAltShiftC is not
  a GlobalBinding, so HostedHelp structurally cannot carry it. Say that, rather than claiming sameness.
  (carried from plan-quality PQ-2, deferred to the boundary review)
- **BR-3** [Minor] `stale-line-number-anchors` The doc sweep's line-numbered anchors are invalidated by the sweep's own earlier edits
  Task 8 Step 2 replaces README 378-381 with a shorter passage before the implementer reaches the 496-497
  anchor, so the later citations drift. Content anchors survive edits; line numbers do not.
  (carried from plan-quality PQ-3, deferred to the boundary review)

## Round 2 — 2026-09-18T12:34:52-07:00 (claude) — passed

### Raised

- **BR-4** [Minor] `help-row-width-budget` Hosted Alt+n row is 124 cols wide and silently disables centering for the whole page
  Measured: standalone page widest line = 104 cols, hosted/Couch page = 124
  ("Alt+n does not reload under Couch and may end the thread; relaunch from the
  Couch switcher (outside the agent pane)"). keyhelp.Center pads from the widest
  line, so on a <=124-col terminal centering vanishes for every section and the
  row wraps under less. Shorten the HostedHelp wording.
- **BR-5** [Minor] `couch-chord-single-source` couchtty.menuControls still restates Couch's chord inventory by hand
  couchkeys is now the source for Couch's chords, but menu.go:22 keeps a parallel
  list of the same six chords with its own Action wording (Alt+d reads "detach this
  thread · all + leave couch here" against couchkeys' "detach every live thread and
  leave Couch"). Only TestREADMEDocumentsEveryPanelControl consumes it, and its
  comment claiming the panel renderer consumes it too is stale. Derive the chord
  rows from couchkeys.Bindings() or drop them.
- **BR-6** [Minor] `scope-section-key-coverage` The "Couch switcher" section omits Ctrl+Space's in-switcher meaning
  console.go:1314 gives Ctrl+Space a second meaning when the panel has focus
  (KeyCtrlSpace = start a thread), but couchkeys declares Ctrl+Space only at
  ScopeEveryPane, so neither `couch --help` nor Alt+h documents it while the
  section title reads as the switcher's key list. A switcher-scope row would
  follow the page's own (key, context) rule.
- **BR-7** [Minor] `seam-struct-requires-nil-defaults` keyscmd.Deps panics on a nil seam, defeating the always-exit-0 contract
  RunWith calls deps.CouchPresents() and passes deps.Getenv unconditionally. A
  caller constructing Deps{Sources: ...} panics; a non-zero exit from `pair keys`
  kills the floating pane before less opens under bin/pair-help's set -euo
  pipefail -- the dead-help-key failure #132 fixed. Default nil Getenv to
  os.Getenv and nil CouchPresents to a false func.
- **BR-8** [Minor] `couch-ownership-predicate-variants` PresentedByCouch matches on tag alone where sibling predicates also match scope
  createflow.go:511 uses scope+tag and lifecycle.go:105 uses tag+valid-scope, but
  outerrecord.go:54 uses tag only. A `pair resume <same-tag>` run from inside a
  Couch thread's right terminal inherits COUCH_THREAD_TAG and records
  presenter=couch. Harmless today; tighten or record why tag alone is the rule.

## Open findings

- **BR-1** [Minor] `parser-fuzz-over-enumerated-cases` DecodeOuterRecord, the only untrusted-input parser here, is pinned by six literal cases instead of a fuzz
- **BR-2** [Minor] `unbacked-behavior-claim` Task 3 asserts hosting changes only Alt+d/Alt+n/Ctrl+Alt+n; compaction.go:90 also reroutes Alt+Shift+C
- **BR-3** [Minor] `stale-line-number-anchors` The doc sweep's line-numbered anchors are invalidated by the sweep's own earlier edits
- **BR-4** [Minor] `help-row-width-budget` Hosted Alt+n row is 124 cols wide and silently disables centering for the whole page
- **BR-5** [Minor] `couch-chord-single-source` couchtty.menuControls still restates Couch's chord inventory by hand
- **BR-6** [Minor] `scope-section-key-coverage` The "Couch switcher" section omits Ctrl+Space's in-switcher meaning
- **BR-7** [Minor] `seam-struct-requires-nil-defaults` keyscmd.Deps panics on a nil seam, defeating the always-exit-0 contract
- **BR-8** [Minor] `couch-ownership-predicate-variants` PresentedByCouch matches on tag alone where sibling predicates also match scope
