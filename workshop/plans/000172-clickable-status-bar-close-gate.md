---
gate: boundary-review
issue: 172
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-05T13:19:15-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Important
          title: 'No complete disposition table: release contradicts the zero-bytes Done-when, and an unterminated SGR prefix has no bound'
          detail: |-
            This is the 2nd finding in family `unspecified-event-policy` (PQ-4 is the
            first and is re-raised not-addressed). Do NOT fix the release case alone —
            state the rule and write the enumeration it implies: every event class the
            terminal can emit gets an explicit row, crossed with the routing state.
            Classes: button-0 press, other buttons, wheel (64/65), release, unparseable
            report, never-terminated prefix, out-of-range coordinate. States: panel
            focused, child-wants-mouse, child-without-mouse. Measured prevalence: this
            one table is the fix for three separate defects.
            Instance A: Task 6's "a RELEASE always forwards, even on couch's row"
            hands a no-mouse child the release bytes — the panelkeys.go:36-42 typeahead
            hazard, and a direct contradiction of Task 8's "ZERO mouse bytes" assertion.
            termcmd's rule at run.go:449-452 is sound only where the child is already
            receiving events. Instance B: Task 7 reuses isSGRMousePrefix inside the
            Interceptor, whose seqPartial arm holds with no bound (keys.go:257-263,
            returns rest=nil), so a stray or pasted `\x1b[<` with no terminator parks
            every following keystroke — #127's dead keyboard, the exact failure
            run.go:575-580 records having shipped once. Say what releases the hold.
            Instance C: the `forward` branch needs the RAW bytes; Task 7 only widens
            FeedHit to return the decoded event, and re-encoding from it would be a
            second source of truth (ARCH-DRY) — findSGRMousePress already returns raw.
            (carried from plan-quality PQ-12, deferred to the boundary review)
          family: unspecified-event-policy
          round: 1
        - id: BR-2
          severity: Minor
          title: Task 12 names neither the menuControls file nor the atlas file its steps change
          detail: |-
            This is the 4th finding in family `unnamed-seam-change`. Do NOT fix this instance alone —
            the rule is already written in the plan at the end of the "Same path as Return" section
            ("each task's Files list names every production site its steps assert"); Task 12 is the
            one task that does not apply it. It has no Files block at all, and its Step 1 changes
            `menuControls` (`cmd/internal/couchtty/menu.go:19`), the README guard's scoped couch
            section, and an atlas file. Apply the already-stated rule to Task 12 rather than
            re-deriving it.
            (carried from plan-quality PQ-15, deferred to the boundary review)
          family: unnamed-seam-change
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-05T13:19:15-07:00"
      agent: claude
      findings:
        - id: BR-3
          severity: Critical
          title: Core-concepts row declares hostty.MouseClickTracking `new` when it exists nowhere
          detail: |-
            workshop/plans/000172-mouse-support-status-bar-and-switcher-plan.md:182 lists
            `hostty.MouseClickTracking` at cmd/internal/hostty/control.go with status `new`.
            grep -rn Mouse cmd/internal/hostty/ returns nothing; it is M2 Task 8's work. The
            plan's own Task 5 requires unshipped rows to carry `planned`, and the contract
            test's planned-skip is what makes the status column the build tracker. The row's
            path is outside conceptPackage, so nothing catches the wrong status. Flip it to
            `planned — M2` plus a Revisions entry.
          family: plan-table-claims-unshipped-code
          round: 2
        - id: BR-4
          severity: Important
          title: The clipped-chip case is asserted only from above, so a lost span passes
          detail: |-
            reserve_mouse_test.go:46-49 bounds spans with chip.End > width but its only
            lower bound is len(Chips) > len(Actors), which cannot fire with three distinct
            threads. Mutating reserve.go:159 to `used > start && used-start >=
            textwidth.Width(label)` makes every truncated chip silently unclickable and the
            whole couchtty suite stays green. Assert the expected span count per width, and
            assert ColumnToActor inside a truncated chip's columns.
          family: onedirectional-geometry-assertion
          round: 2
        - id: BR-5
          severity: Important
          title: clampExtents and the scrolled list are unreachable from any test
          detail: |-
            Replacing clampExtents' body with `return extents` leaves menu_render green, and
            so does computing the extent index from the inventory row instead of len(lines).
            The fixture is 2 actors in 14 rows, so neither rowBudget scrolling nor the height
            clamp is ever entered. Plan Task 4 Step 4 and the Done-when both require the
            scrolled and clipped cases.
          family: onedirectional-geometry-assertion
          round: 2
        - id: BR-6
          severity: Important
          title: mouseinput ships with no test file, and its Core-concepts row is out of the guard's scope
          detail: |-
            Plan Task 1 names cmd/internal/mouseinput/mouseinput_test.go; it does not exist
            (go test reports "no test files"). ParsePrefix, MaxReport, WheelUp, WheelDown and
            Terminators have no direct coverage. The plan's `mouseinput.Event / mouseinput.Find`
            row would have been caught by assertDirectTest, but conceptPackage filters it out,
            so Task 5 pinned 6 of the 8 non-planned rows. Third instance of the family PQ-6
            and PQ-14 named.
          family: guard-not-registered
          round: 2
        - id: BR-7
          severity: Important
          title: ChipSpan and ActorExtent state no coordinate base, on a struct that already carries a 1-based one
          detail: |-
            RenderMenuView documents Cursor as "1-based terminal cells in Body"; the new
            Extents field on the same struct is 0-based rows of Body and says so nowhere.
            ChipSpan.Start/End are 0-based display columns while mouseinput.Event.X is 1-based
            and the row is painted from terminal column 1. M2/M3 consume this surface; state
            the base and origin in the ChipSpan, ActorExtent and PointToActor doc comments.
          family: coordinate-base-unstated-at-seam
          round: 2
        - id: BR-8
          severity: Important
          title: The 173-to-192 renumber left the issue path in the same comment block stale
          detail: |-
            deadsymbols_test.go:47 still reads workshop/issues/000173-disposition-six-production-
            symbols-reachable-only-from-tests.md, a file that no longer exists, two lines above
            the six entries the sweep updated to pair#192. The comment's own next sentence makes
            the ticket's existence load-bearing. grep -rn "000173-disposition" returns this one
            site.
          family: stale-issue-reference
          round: 2
        - id: BR-9
          severity: Minor
          title: RenderStatusRow's doc block is now attached to ChipSpan
          detail: |-
            reserve.go:86-93 explains untrusted labels and control-byte stripping, but the new
            ChipSpan type was inserted between it and RenderStatusRow (:126), which now has no
            doc comment. godoc attributes the sanitize rationale to the wrong type.
          family: orphaned-doc-comment
          round: 2
        - id: BR-10
          severity: Minor
          title: WheelUp/WheelDown have zero call sites while termcmd keeps the 64/65 literals
          detail: |-
            mouseinput.go:36-37 introduces the named constants; termcmd/run.go:451,457 still
            switch on event.Button == 64 / == 65. The promotion added a second source of truth
            for the wheel codes rather than removing one (ARCH-DRY).
          family: promoted-constant-with-surviving-literal
          round: 2
        - id: BR-11
          severity: Minor
          title: mousePressEvent alias and parseSGRMousePressPrefix have no callers after the move
          detail: |-
            termcmd/run.go:579 and :585. parseSGRMousePressPrefix was only ever reached through
            the old findSGRMousePress, which now delegates to mouseinput.Find. deadSymbolScope
            is cmd/internal/couchcore, so the orphan guard cannot see termcmd.
          family: move-residue
          round: 2
        - id: BR-12
          severity: Minor
          title: MaxReport and the atlas both state a bound that no code enforces yet
          detail: |-
            mouseinput.go:31 and atlas/couch.md read as shipped invariants; the bound is M2
            Task 7 Step 3b. Mark the constant as consumed in M2 so it does not read as enforced.
          family: doc-ahead-of-enforcement
          round: 2
        - id: BR-13
          severity: Minor
          title: The one-extent-per-actor run shape is unobservable through PointToActor
          detail: |-
            Removing the run-merge at menu_render.go:437-441 leaves the suite green, because
            per-line extents still resolve to the right thread. The exported Extents field's
            documented shape ("where each actor was DRAWN") is therefore unpinned for any
            consumer that reads it directly rather than through PointToActor.
          family: unpinned-exported-shape
          round: 2
      boundary: M1
      blocked: true
---

# Gate ledger — pair#172 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-05T13:19:15-07:00 (sdlc) — passed

### Raised

- **BR-1** [Important] `unspecified-event-policy` No complete disposition table: release contradicts the zero-bytes Done-when, and an unterminated SGR prefix has no bound
  This is the 2nd finding in family `unspecified-event-policy` (PQ-4 is the
  first and is re-raised not-addressed). Do NOT fix the release case alone —
  state the rule and write the enumeration it implies: every event class the
  terminal can emit gets an explicit row, crossed with the routing state.
  Classes: button-0 press, other buttons, wheel (64/65), release, unparseable
  report, never-terminated prefix, out-of-range coordinate. States: panel
  focused, child-wants-mouse, child-without-mouse. Measured prevalence: this
  one table is the fix for three separate defects.
  Instance A: Task 6's "a RELEASE always forwards, even on couch's row"
  hands a no-mouse child the release bytes — the panelkeys.go:36-42 typeahead
  hazard, and a direct contradiction of Task 8's "ZERO mouse bytes" assertion.
  termcmd's rule at run.go:449-452 is sound only where the child is already
  receiving events. Instance B: Task 7 reuses isSGRMousePrefix inside the
  Interceptor, whose seqPartial arm holds with no bound (keys.go:257-263,
  returns rest=nil), so a stray or pasted `\x1b[<` with no terminator parks
  every following keystroke — #127's dead keyboard, the exact failure
  run.go:575-580 records having shipped once. Say what releases the hold.
  Instance C: the `forward` branch needs the RAW bytes; Task 7 only widens
  FeedHit to return the decoded event, and re-encoding from it would be a
  second source of truth (ARCH-DRY) — findSGRMousePress already returns raw.
  (carried from plan-quality PQ-12, deferred to the boundary review)
- **BR-2** [Minor] `unnamed-seam-change` Task 12 names neither the menuControls file nor the atlas file its steps change
  This is the 4th finding in family `unnamed-seam-change`. Do NOT fix this instance alone —
  the rule is already written in the plan at the end of the "Same path as Return" section
  ("each task's Files list names every production site its steps assert"); Task 12 is the
  one task that does not apply it. It has no Files block at all, and its Step 1 changes
  `menuControls` (`cmd/internal/couchtty/menu.go:19`), the README guard's scoped couch
  section, and an atlas file. Apply the already-stated rule to Task 12 rather than
  re-deriving it.
  (carried from plan-quality PQ-15, deferred to the boundary review)

## Round 2 — 2026-09-05T13:19:15-07:00 (claude) — BLOCKED

### Raised

- **BR-3** [Critical] `plan-table-claims-unshipped-code` Core-concepts row declares hostty.MouseClickTracking `new` when it exists nowhere
  workshop/plans/000172-mouse-support-status-bar-and-switcher-plan.md:182 lists
  `hostty.MouseClickTracking` at cmd/internal/hostty/control.go with status `new`.
  grep -rn Mouse cmd/internal/hostty/ returns nothing; it is M2 Task 8's work. The
  plan's own Task 5 requires unshipped rows to carry `planned`, and the contract
  test's planned-skip is what makes the status column the build tracker. The row's
  path is outside conceptPackage, so nothing catches the wrong status. Flip it to
  `planned — M2` plus a Revisions entry.
- **BR-4** [Important] `onedirectional-geometry-assertion` The clipped-chip case is asserted only from above, so a lost span passes
  reserve_mouse_test.go:46-49 bounds spans with chip.End > width but its only
  lower bound is len(Chips) > len(Actors), which cannot fire with three distinct
  threads. Mutating reserve.go:159 to `used > start && used-start >=
  textwidth.Width(label)` makes every truncated chip silently unclickable and the
  whole couchtty suite stays green. Assert the expected span count per width, and
  assert ColumnToActor inside a truncated chip's columns.
- **BR-5** [Important] `onedirectional-geometry-assertion` clampExtents and the scrolled list are unreachable from any test
  Replacing clampExtents' body with `return extents` leaves menu_render green, and
  so does computing the extent index from the inventory row instead of len(lines).
  The fixture is 2 actors in 14 rows, so neither rowBudget scrolling nor the height
  clamp is ever entered. Plan Task 4 Step 4 and the Done-when both require the
  scrolled and clipped cases.
- **BR-6** [Important] `guard-not-registered` mouseinput ships with no test file, and its Core-concepts row is out of the guard's scope
  Plan Task 1 names cmd/internal/mouseinput/mouseinput_test.go; it does not exist
  (go test reports "no test files"). ParsePrefix, MaxReport, WheelUp, WheelDown and
  Terminators have no direct coverage. The plan's `mouseinput.Event / mouseinput.Find`
  row would have been caught by assertDirectTest, but conceptPackage filters it out,
  so Task 5 pinned 6 of the 8 non-planned rows. Third instance of the family PQ-6
  and PQ-14 named.
- **BR-7** [Important] `coordinate-base-unstated-at-seam` ChipSpan and ActorExtent state no coordinate base, on a struct that already carries a 1-based one
  RenderMenuView documents Cursor as "1-based terminal cells in Body"; the new
  Extents field on the same struct is 0-based rows of Body and says so nowhere.
  ChipSpan.Start/End are 0-based display columns while mouseinput.Event.X is 1-based
  and the row is painted from terminal column 1. M2/M3 consume this surface; state
  the base and origin in the ChipSpan, ActorExtent and PointToActor doc comments.
- **BR-8** [Important] `stale-issue-reference` The 173-to-192 renumber left the issue path in the same comment block stale
  deadsymbols_test.go:47 still reads workshop/issues/000173-disposition-six-production-
  symbols-reachable-only-from-tests.md, a file that no longer exists, two lines above
  the six entries the sweep updated to pair#192. The comment's own next sentence makes
  the ticket's existence load-bearing. grep -rn "000173-disposition" returns this one
  site.
- **BR-9** [Minor] `orphaned-doc-comment` RenderStatusRow's doc block is now attached to ChipSpan
  reserve.go:86-93 explains untrusted labels and control-byte stripping, but the new
  ChipSpan type was inserted between it and RenderStatusRow (:126), which now has no
  doc comment. godoc attributes the sanitize rationale to the wrong type.
- **BR-10** [Minor] `promoted-constant-with-surviving-literal` WheelUp/WheelDown have zero call sites while termcmd keeps the 64/65 literals
  mouseinput.go:36-37 introduces the named constants; termcmd/run.go:451,457 still
  switch on event.Button == 64 / == 65. The promotion added a second source of truth
  for the wheel codes rather than removing one (ARCH-DRY).
- **BR-11** [Minor] `move-residue` mousePressEvent alias and parseSGRMousePressPrefix have no callers after the move
  termcmd/run.go:579 and :585. parseSGRMousePressPrefix was only ever reached through
  the old findSGRMousePress, which now delegates to mouseinput.Find. deadSymbolScope
  is cmd/internal/couchcore, so the orphan guard cannot see termcmd.
- **BR-12** [Minor] `doc-ahead-of-enforcement` MaxReport and the atlas both state a bound that no code enforces yet
  mouseinput.go:31 and atlas/couch.md read as shipped invariants; the bound is M2
  Task 7 Step 3b. Mark the constant as consumed in M2 so it does not read as enforced.
- **BR-13** [Minor] `unpinned-exported-shape` The one-extent-per-actor run shape is unobservable through PointToActor
  Removing the run-merge at menu_render.go:437-441 leaves the suite green, because
  per-line extents still resolve to the right thread. The exported Extents field's
  documented shape ("where each actor was DRAWN") is therefore unpinned for any
  consumer that reads it directly rather than through PointToActor.

## Open findings

- **BR-1** [Important] `unspecified-event-policy` No complete disposition table: release contradicts the zero-bytes Done-when, and an unterminated SGR prefix has no bound
- **BR-2** [Minor] `unnamed-seam-change` Task 12 names neither the menuControls file nor the atlas file its steps change
- **BR-3** [Critical] `plan-table-claims-unshipped-code` Core-concepts row declares hostty.MouseClickTracking `new` when it exists nowhere
- **BR-4** [Important] `onedirectional-geometry-assertion` The clipped-chip case is asserted only from above, so a lost span passes
- **BR-5** [Important] `onedirectional-geometry-assertion` clampExtents and the scrolled list are unreachable from any test
- **BR-6** [Important] `guard-not-registered` mouseinput ships with no test file, and its Core-concepts row is out of the guard's scope
- **BR-7** [Important] `coordinate-base-unstated-at-seam` ChipSpan and ActorExtent state no coordinate base, on a struct that already carries a 1-based one
- **BR-8** [Important] `stale-issue-reference` The 173-to-192 renumber left the issue path in the same comment block stale
- **BR-9** [Minor] `orphaned-doc-comment` RenderStatusRow's doc block is now attached to ChipSpan
- **BR-10** [Minor] `promoted-constant-with-surviving-literal` WheelUp/WheelDown have zero call sites while termcmd keeps the 64/65 literals
- **BR-11** [Minor] `move-residue` mousePressEvent alias and parseSGRMousePressPrefix have no callers after the move
- **BR-12** [Minor] `doc-ahead-of-enforcement` MaxReport and the atlas both state a bound that no code enforces yet
- **BR-13** [Minor] `unpinned-exported-shape` The one-extent-per-actor run shape is unobservable through PointToActor
