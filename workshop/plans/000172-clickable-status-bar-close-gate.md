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
    - "n": 3
      timestamp: "2026-09-05T13:44:27-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Disposition table is in the plan and in RouteMouseReport; release-to-a-no-mouse-child swallows, MaxReport bounds the hold, MouseHit.Raw carries the wire bytes.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Task 12 now names menu.go:19, README.md, atlas/couch.md and the readme_test guard.
          round: 3
        - id: BR-3
          disposition: not-addressed
          note: Status flipped to `planned — M2`, but the symbol hostty.MouseClickTracking still exists nowhere and M2 shipped EnableMouseClicks/DisableMouseClicks in this same window; no Revisions entry was appended.
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: The named mutation reddens only because the fixture's `beta` is Active; `(a.Active || used-start >= textwidth.Width(label))` still leaves the whole suite green. Neither ask landed.
          round: 3
        - id: BR-5
          disposition: not-addressed
          note: 'Verified: clampExtents -> `return extents` is green, and `index := len(lines) + start` is green. Neither the clipped nor the scrolled path is reachable.'
          round: 3
        - id: BR-6
          disposition: addressed
          note: mouseinput_test.go exists and covers Parse/ParsePrefix/Find/IsPrefix/MaxReport directly; the out-of-scope row remains pair#188's structural issue.
          round: 3
        - id: BR-7
          disposition: addressed
          note: ChipSpan, ColumnToActor, ActorExtent and PointToActor all state ZERO-BASED and name the 1-based conversion; onMouse does it once.
          round: 3
        - id: BR-8
          disposition: addressed
          note: 000173 -> 000192 in the comment and all six entries; workshop/issues/000192-... exists.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: reserve.go:93-94 still runs the RenderStatusRow doc block straight into ChipSpan's; RenderStatusRow (:133) has no doc comment. A second instance now exists at menu.go:199-208.
          round: 3
        - id: BR-10
          disposition: not-addressed
          note: termcmd/run.go:453,459 still switch on the literals 64 and 65 while mouseinput.WheelUp/WheelDown exist.
          round: 3
        - id: BR-11
          disposition: not-addressed
          note: mousePressEvent (run.go:579) and parseSGRMousePressPrefix (:585) still have zero callers.
          round: 3
        - id: BR-12
          disposition: addressed
          note: keys.go:301 enforces MaxReport and TestAnUnterminatedMousePrefixDoesNotParkTheKeyboard pins it.
          round: 3
        - id: BR-13
          disposition: not-addressed
          note: 'Verified: deleting the run-merge at menu_render.go:463-467 leaves the suite green.'
          round: 3
      findings:
        - id: BR-14
          severity: Critical
          title: Three shipped M2/M3 behaviours are unpinned - removing each leaves the whole couchtty suite green
          detail: |-
            This is the 2nd finding in family `unpinned-exported-shape`. Do NOT patch the three
            sites. State the rule: every behaviour this issue introduces is pinned by a test that
            reddens when the behaviour is removed, and the enumeration is already written - the
            plan's "Done-when to Task map". The check is the mutation, not the green run.
            Measured prevalence, all verified in the working tree and restored: deleting
            c.writeOwn(hostty.EnableMouseClicks) (console.go:528) - green; discarding
            child.child.Write(hit.Raw) (console.go:1503) - green; deleting `&& !origin.Manual`
            (console.go:1455) - green, which is the marker the commit message headlines and the
            one Plan Task 11 Step 2 calls not optional. Four map rows have no test at all:
            TestClickInTheSwitcherTakesTheReturnPath, TestClickIsAManualSwitch,
            TestForwardPreservesRawBytes, TestTeardownDisablesMouseTracking.
          family: unpinned-exported-shape
          round: 3
        - id: BR-15
          severity: Important
          title: M2 and M3 production code landed inside the M1 window, so their own boundary reviews open on an empty range
          detail: |-
            f95da992 ships routing, the Interceptor's SGR arm, couch's mode enable, onMouse and the
            manual-switch marker - all tagged M2/M3 in the plan - inside the window this M1 gate
            reviews. Once M1 closes at f95da992, BASE_SHA for M2 and M3 is that commit, so the
            mandatory fresh-eyes review for the milestone the plan calls "the work" sees nothing.
            That is the gate that was supposed to catch the Critical above. Either split so M2/M3
            land after M1 closes, or close all three against this window and pay their tests now.
          family: milestone-scope-overrun
          round: 3
        - id: BR-16
          severity: Important
          title: couch's own mouse mode is written once and never re-asserted, so a child's DECRST silently ends the feature
          detail: |-
            This is the 3rd finding in family `unspecified-event-policy`. Do NOT add a re-assert
            and stop. The rule the family keeps asking for: the disposition table must be crossed
            with the CHILD's mode-transition events, not only the operator's report events.
            console.go:528 is the only write of EnableMouseClicks; ?1000 is terminal-global, so a
            child emitting \x1b[?1000l - nvim does this on exit - turns couch's clicks off for the
            whole terminal with nothing to restore them. ptychild.Screen.Mouse() already observes
            the transition, so couch has the signal and no policy. Enumeration: child enables;
            child disables; child exits with mouse on; switch between two children with different
            modes; replay re-asserting the child's modes over couch's.
          family: unspecified-event-policy
          round: 3
        - id: BR-17
          severity: Important
          title: The click reducer restates Enter's switch/resume rule and diverges from it, and sets Manual on a refused dispatch
          detail: |-
            menu.go:349-365 duplicates reduceRootKey's operation choice (menu.go:475-479); a
            non-actionable row gets a notice from Enter and silence from a click; and
            state.InFlight.Manual = true (menu.go:363) runs unconditionally, including when
            dispatchMenuOperation refused because another operation was in flight
            (menu.go:1542-1544) - writing the flag onto a different operation's origin. The
            Done-when requires the same handler, not a parallel one (ARCH-DRY). Extract the
            actionable-thread to operation decision into one function both arms call, and set
            Manual only when the dispatch produced effects.
          family: parallel-handler-restates-decision
          round: 3
        - id: BR-18
          severity: Important
          title: atlas and README cover M1's geometry but not the routing, ownership and click gesture that shipped in the same window
          detail: |-
            atlas/couch.md gained the M1 geometry in 1c895d4f and nothing for f95da992: the
            four-way disposition table, the narrowed release rule, couch's ownership of
            ?1000;?1006, and the manual-switch classification. menuControls (menu.go:19-33) has no
            mouse row, so TestREADMEDocumentsEveryPanelControl cannot fire and README never
            mentions that a click in the switcher selects and enters - a user-facing gesture that
            is live in the binary today. Task 12 defers this to M3's close, which only works if
            the code also waits for M3.
          family: docs-lag-shipped-surface
          round: 3
        - id: BR-19
          severity: Minor
          title: MenuEventNotice's doc block is now attached to MenuEventMouseSwitch, a second instance in the commit that left the first
          detail: |-
            This is the 2nd finding in family `orphaned-doc-comment`; BR-9 is the first and is
            re-raised not-addressed. Do NOT fix this instance alone. The rule: a symbol inserted
            into an existing file must go after the symbol an adjacent doc block documents, or the
            block must be re-anchored. Enumeration for this issue: reserve.go:93-94 (RenderStatusRow's
            block now reads as ChipSpan's) and menu.go:199-208 (MenuEventNotice's block now reads as
            MenuEventMouseSwitch's). Both are checkable with `go doc`.
          family: orphaned-doc-comment
          round: 3
        - id: BR-20
          severity: Minor
          title: hostty.DisableMouseClicks joins two other zero-call-site symbols left by the promotion
          detail: |-
            This is the 2nd finding in family `move-residue`; BR-11 is the first and is re-raised
            not-addressed. Do NOT delete this one symbol. The rule: a symbol this issue introduces
            or leaves behind has a production call site by the milestone that introduces it, or it
            is deleted. Enumeration: termcmd/run.go:579 (mousePressEvent alias), :585
            (parseSGRMousePressPrefix), hostty/control.go:62 (DisableMouseClicks), and
            mouseinput.WheelUp/WheelDown which exist only in tests while run.go:453,459 keep the
            literals. deadSymbolScope is cmd/internal/couchcore, so no guard sees any of them.
          family: move-residue
          round: 3
      boundary: M1
      blocked: true
    - "n": 4
      timestamp: "2026-09-05T16:25:23-07:00"
      agent: claude
      dispose:
        - id: BR-3
          disposition: not-addressed
          note: Status flipped, but no plan Revisions entry, the row still names a symbol that exists nowhere, and six rows now claim `planned` for shipped code.
          round: 4
        - id: BR-4
          disposition: not-addressed
          note: 'Verified: dropping the span for any chip truncated below 3 columns leaves the whole suite green.'
          round: 4
        - id: BR-5
          disposition: not-addressed
          note: 'Verified twice: `return extents` and a scroll-offset re-base both stay green; the new fixture enters neither path.'
          round: 4
        - id: BR-9
          disposition: not-addressed
          note: '`go doc ChipSpan` still prints RenderStatusRow''s sanitize rationale; RenderStatusRow still has no doc.'
          round: 4
        - id: BR-10
          disposition: not-addressed
          note: run.go:453,459 still switch on the 64/65 literals; WheelUp/WheelDown remain test-only.
          round: 4
        - id: BR-11
          disposition: not-addressed
          note: run.go:579 and :585 still have zero references anywhere in the tree.
          round: 4
        - id: BR-13
          disposition: not-addressed
          note: 'Verified: deleting the run-merge at menu_render.go:465-469 leaves the suite green.'
          round: 4
        - id: BR-14
          disposition: not-addressed
          note: Two of the three named sites are now pinned, but three of THIS round's own fixes are not; seven unpinned behaviours measured.
          round: 4
        - id: BR-15
          disposition: not-addressed
          note: Merging M3 into M2 does not give M2 a diff; its close still opens on an empty range, carrying Task 12's manual check and pair#166 with it.
          round: 4
        - id: BR-16
          disposition: not-addressed
          note: The re-assert can be deleted with the suite green, and the child-mode-transition enumeration was never written.
          round: 4
        - id: BR-17
          disposition: not-addressed
          note: Both fixes are unpinned by mutation, and a non-actionable row still gets a notice from Enter and silence from a click.
          round: 4
        - id: BR-18
          disposition: addressed
          note: atlas/couch.md, README and the menuControls row all landed; the README guard fires on the new row.
          round: 4
        - id: BR-19
          disposition: not-addressed
          note: menu.go:200-209 unchanged; MenuEventNotice's block still reads as MenuEventMouseSwitch's.
          round: 4
        - id: BR-20
          disposition: not-addressed
          note: DisableMouseClicks, mousePressEvent, parseSGRMousePressPrefix and WheelUp/WheelDown all still have zero production call sites.
          round: 4
      findings:
        - id: BR-21
          severity: Minor
          title: The handler-table entry for HitMouse is a stub the dispatcher never reaches, so the enumeration guard proves nothing for it
          detail: |-
            This is the 2nd finding in family `guard-not-registered`. Do NOT just delete
            or fill in this one entry. The rule: an enumeration guard is satisfied only
            by the thing it guards -- registering a value the production path never
            reads converts the guard into a formality that reports coverage it does not
            have. console.go:647 special-cases HitMouse BEFORE consulting the table, so
            hitHandlers()[HitMouse] (console.go:1557) is a `func() {}` with no caller;
            AllInterceptorHits still counts it as proven. The same shape is one edit
            away for any future payload-carrying hit. Either widen the table's value to
            carry the payload so every hit really does route through it, or have the
            guard assert reachability rather than presence. Related: Interceptor.mouse
            is a payload valid only between one FeedHit and the next, kept correct by a
            hand-written ordering rather than by the type (ARCH-ORDER).
          family: guard-not-registered
          round: 4
      boundary: M1
      blocked: true
    - "n": 5
      timestamp: "2026-09-05T16:47:31-07:00"
      agent: claude
      dispose:
        - id: BR-3
          disposition: addressed
          note: Row is now hostty.EnableMouseClicks/DisableMouseClicks and both symbols exist; the prose bullet at plan:193 still says MouseClickTracking, raised separately as Minor.
          round: 5
        - id: BR-4
          disposition: addressed
          note: 'Verified: mutating reserve.go:175 to require the full label width reddens reserve_mouse_test.go:60 and :124.'
          round: 5
        - id: BR-5
          disposition: not-addressed
          note: Clamp half honestly dispositioned; the SCROLLED half is still unreachable -- the index mutation stays green because every extent test selects inventory[0].
          round: 5
        - id: BR-9
          disposition: not-addressed
          note: go doc ChipSpan still prints RenderStatusRow's rationale and go doc RenderStatusRow prints nothing; the added comment describes the fix instead of being it.
          round: 5
        - id: BR-10
          disposition: addressed
          note: run.go:453,459 now switch on mouseinput.WheelUp/WheelDown; no 64/65 literals remain.
          round: 5
        - id: BR-11
          disposition: not-addressed
          note: parseSGRMousePressPrefix is gone but the mousePressEvent alias at run.go:581 still has zero callers, under a comment claiming the aliases were removed.
          round: 5
        - id: BR-13
          disposition: addressed
          note: 'Verified: deleting the run-merge reddens TestClampAndRunMergeAreLoadBearing.'
          round: 5
        - id: BR-14
          disposition: addressed
          note: All three sites verified RED under mutation, and the four unmapped Done-when rows now have tests that reach production.
          round: 5
        - id: BR-15
          disposition: addressed
          note: The tests were paid inside this window and this M1 review covers the M2/M3 code; residual is that M2's own close now opens on a near-empty range and should be recorded as covered here.
          round: 5
        - id: BR-16
          disposition: not-addressed
          note: The re-assert landed but is unpinned (deleting it leaves the suite green) and it introduces a regression -- see the new Critical; the crossed enumeration was never written.
          round: 5
        - id: BR-17
          disposition: not-addressed
          note: enterOperationFor and the effects-gated Manual flag both landed and mutate RED; the third half survives -- a non-actionable row still gets a notice from Enter and silence from a click.
          round: 5
        - id: BR-19
          disposition: not-addressed
          note: 'Unchanged: go doc MenuEventMouseSwitch prints MenuEventNotice''s block and MenuEventNotice has none.'
          round: 5
        - id: BR-20
          disposition: not-addressed
          note: WheelUp/WheelDown got production callers, but DisableMouseClicks and the mousePressEvent alias still have none, and seqMouse joins the enumeration as a new zero-reachability symbol.
          round: 5
        - id: BR-21
          disposition: not-addressed
          note: console.go:647 still special-cases HitMouse before the table and hitHandlers()[HitMouse] is still an uncalled func(){} counted by the guard.
          round: 5
      findings:
        - id: BR-22
          severity: Critical
          title: The per-paint ?1000h re-assert demotes a child that enabled ?1002/?1003, breaking the "child keeps its own events unchanged" Done-when
          detail: |-
            console.go:1024-1030 writes hostty.EnableMouseClicks on every paint. Mouse-tracking
            modes 1000/1002/1003 are one mutually-exclusive state in xterm (send_mouse_pos),
            Alacritty (removes MOUSE_MODE before inserting), kitty, Ghostty and iTerm2, so
            setting 1000 while a child holds 1002 replaces button-event tracking with
            press/release only. Screen.Mouse() still reports true, so couch keeps forwarding
            presses into a child that will never receive the motion that closes its drag --
            the "nvim stuck in visual selection" symptom mouseinput.go:19-23 documents.
            paintNow is on the rowDirty/bell/attention path, so a full-screen child triggers
            it continuously. The code comment and atlas/couch.md both assert the opposite.
            This is the 3rd finding in family unspecified-event-policy. Do NOT patch this one
            site. The rule the family keeps asking for: the disposition table must be crossed
            with the CHILD's mode-transition events, and each cell must state what couch
            writes AND what mode the child is left holding. Enumeration (unchanged from
            BR-16): child enables 1000; child enables 1002/1003; child disables; child exits
            with mouse on; switch between two children with different modes; replay
            re-asserting over couch's. Measured prevalence: 1 of 6 cells implemented, 0 of 6
            pinned -- deleting the paintNow re-assert leaves the whole couchtty suite green.
          family: unspecified-event-policy
          round: 5
        - id: BR-23
          severity: Important
          title: assertDirectTest is satisfied by the contract's own conceptInventory literal, so every PURE row's coverage assertion is vacuous
          detail: |-
            core_concepts_contract_test.go:377-394 globs *_test.go in the package and greps
            for the row's symbol; the contract file is in that glob and holds every symbol
            name verbatim in conceptInventory, so the match always succeeds. seqMouse is the
            demonstration -- its only occurrence in any _test.go is line 73 of the contract
            itself, and seqMouse is dead code (sequenceAt can never return it; deleting
            `case seqMouse: return HitMouse` leaves the suite green).
            This is the 3rd finding in family guard-not-registered. Do NOT fix this one
            guard. The rule covering both instances: an enumeration guard is satisfied only
            by an artifact the production path reads. hitHandlers()[HitMouse] is a func(){}
            the dispatcher skips; assertDirectTest matches on the inventory that declares the
            requirement. Measured prevalence: 2 of 2 enumeration guards this issue touches
            are self-satisfying. Fix: exclude the contract file from the glob, then give
            whatever goes red a real test.
          family: guard-not-registered
          round: 5
        - id: BR-24
          severity: Minor
          title: sgrMouseSize derives Terminators from mouseinput but restates the report-length rule
          detail: |-
            termcmd/rename_input.go:175-186 computes the report size as idx+3+1 after
            scanning for mouseinput.Terminators. mouseinput.ParsePrefix already answers that
            question as len(raw), and the package doc says it exists so there is ONE answer
            to "where does this sequence end". This is the 2nd finding in family
            promoted-constant-with-surviving-literal. Do NOT just rewrite this call. The
            rule: a promotion is finished when every consumer derives the RULE, not only the
            constant the diff happened to touch. Enumeration for this issue:
            termcmd/run.go:453,459 (fixed this round), rename_input.go:183 (constant derived,
            length rule restated). Note the two differ on malformed input -- ParsePrefix
            validates the numbers and sgrMouseSize does not -- so the sweep must state which
            behaviour wins.
          family: promoted-constant-with-surviving-literal
          round: 5
        - id: BR-25
          severity: Minor
          title: The plan's Integration-points prose bullet still names hostty.MouseClickTracking after the Revisions entry renamed it
          detail: |-
            workshop/plans/000172-mouse-support-status-bar-and-switcher-plan.md:193 reads
            "**hostty.MouseClickTracking**", seven lines below a Revisions entry recording
            that it became EnableMouseClicks/DisableMouseClicks. A reader grepping the plan
            for the symbol finds a name that exists nowhere in the tree.
            This is the 2nd finding in family plan-table-claims-unshipped-code. Do NOT edit
            only this line. The rule: a rename inside an artifact is finished when
            grep <old-name> over the artifact returns zero, not when the machine-parsed row
            is fixed -- the contract test reads the table, so the table is the only part the
            last fix was forced to get right.
          family: plan-table-claims-unshipped-code
          round: 5
      boundary: M1
      blocked: true
    - "n": 6
      timestamp: "2026-09-05T20:53:23-07:00"
      agent: claude
      boundary: M1
      blocked: true
      protocol_error: no valid findings block
    - "n": 7
      timestamp: "2026-09-05T22:22:18-07:00"
      agent: claude
      boundary: M1
      blocked: true
      protocol_error: no valid findings block
    - "n": 8
      timestamp: "2026-09-05T22:40:16-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: not-addressed
          note: 'Run-merge and clamp bounds now pinned, but the scrolled list still enters no test: index := len(lines) + start*1000 leaves the whole package green.'
          round: 8
        - id: BR-9
          disposition: not-addressed
          note: go doc ChipSpan still prints RenderStatusRow's rationale and go doc RenderStatusRow prints nothing; the block was explained, not moved.
          round: 8
        - id: BR-11
          disposition: addressed
          note: mousePressEvent and parseSGRMousePressPrefix are deleted; the two surviving wrappers have live call sites at run.go:418,471.
          round: 8
        - id: BR-16
          disposition: addressed
          note: 'Mutation-verified both ways: deleting the per-paint re-assert reddens 4 subtests, making it unconditional reddens 4 others.'
          round: 8
        - id: BR-17
          disposition: not-addressed
          note: Both halves revert green (enterOperationFor -> "switch"; Manual set unconditionally), and the non-actionable notice divergence remains.
          round: 8
        - id: BR-19
          disposition: addressed
          note: MenuEventMouseSwitch now carries its own doc and MenuEventNotice's block is re-anchored.
          round: 8
        - id: BR-20
          disposition: addressed
          note: 'All four enumerated symbols swept: DisableMouseClicks and the alias deleted, WheelUp/WheelDown now used at run.go:453,459.'
          round: 8
        - id: BR-21
          disposition: addressed
          note: hitHandlers()[HitMouse] is a real handler; no-oping it reddens four console tests.
          round: 8
        - id: BR-23
          disposition: addressed
          note: Verified by deleting mouse_test.go — both PURE rows now report uncovered instead of matching the inventory literal.
          round: 8
        - id: BR-22
          disposition: addressed
          note: The paintNow site is fixed and pinned; the same mechanism survives through Screen.Mouse()'s mode conflation, raised separately.
          round: 8
        - id: BR-24
          disposition: addressed
          note: sgrMouseSize derives the length from ParsePrefix and the malformed-input winner is stated.
          round: 8
        - id: BR-25
          disposition: not-addressed
          note: The named bullet is fixed; the class is not — seqMouse, IsSGRPrefix, FindSGR and DisableMouseClicks still name symbols absent from the tree.
          round: 8
      findings:
        - id: BR-26
          severity: Important
          title: couch writes two mouse modes and governs one, and the state it reads collapses tracking and encoding into a single bool
          detail: |-
            This is the 4th finding in family `unspecified-event-policy`. Do NOT add a fifth
            site fix. Probed and confirmed at the pinned head: a child doing `\x1b[?1002h`
            then `\x1b[?1006l` flips Screen.Mouse() false (screen.go:415 maps 1000/1002/1003
            /1006 onto one bool), so the next paint writes `?1000;1006h` and demotes it —
            BR-22's mechanism reached through the observation instead of the write; and a
            child doing `\x1b[?1006h` alone makes couch stand back for a child holding no
            tracking at all. Two more halves: `?1006` is never stood back from, so a child
            enabling `?1000h` without `?1006h` receives SGR reports it cannot parse (against
            the "receives its own events unchanged" Done-when), and the startup write at
            console.go:528 is gated by nothing. The take-back also has no trigger of its own:
            onChunk repaints only on RowDirty/Bell/attention and the DECSET handler sets
            rowDirty only for alt-screen, so a bare `?1000l` leaves couch's clicks off
            indefinitely; TestMouseModeTransitions cannot see this because it calls repaint()
            by hand and its FakeChild has a nil sink. The rule: couch's mouse ownership is ONE
            gated writer over the full set of modes it touches — tracking AND encoding —
            driven by an observation that keeps them as separate state, and every cell of
            (child mode transition x couch write) states what couch writes, what mode AND
            encoding the child is left holding, and what EVENT re-evaluates the cell.
            Measured prevalence: 3 writers, 1 governed; 2 modes written, 1 governed; 1
            observation collapsing 4 modes into 1 bool; 6 cells pinned, 0 pinning the trigger.
          family: unspecified-event-policy
          round: 8
        - id: BR-27
          severity: Important
          title: atlas/couch.md:356-361 still teaches the refuted model that produced BR-22
          detail: |-
            This is the 2nd finding in family `docs-lag-shipped-surface` (BR-18 was the
            first). Do NOT patch only this paragraph. It reads "couch RE-ASSERTS its own on
            every paint ... DECSET is additive and idempotent, so this cannot clobber a mode
            the child set for itself"; both halves are false at this head, since console.go
            :1040 re-asserts only while !childWantsMouse() precisely because 1000/1002/1003
            replace one another. The rule: when a boundary changes a behaviour the atlas
            already describes, the same commit rewrites the contradicted paragraph — the
            check is grepping the atlas for the mechanism the diff changed, not "did I add a
            section". Measured prevalence: 1 of 1 contradicted paragraph left standing while
            a new section for the same feature landed in the same window.
          family: docs-lag-shipped-surface
          round: 8
        - id: BR-28
          severity: Important
          title: The plan artifact disagrees with the tree in six places, with no Revisions entry
          detail: |-
            This is the 3rd finding in family `plan-table-claims-unshipped-code`. Do NOT fix
            only the row a test reads. Sites: :182 declares DisableMouseClicks status `new`
            and it exists nowhere (declared path is outside conceptPackage, so the contract
            test is blind to it, the same gap BR-3 exploited); :118, :165, :394, :396
            describe seqMouse as shipped work; :396 names mouseinput.IsSGRPrefix and :155
            names mouseinput.FindSGR, which shipped as IsPrefix and Find; the section titled
            "The complete disposition table" (:41-56) has no couchOwnsScreen dimension and so
            omits the two switcher rows the code carries, and its "anything, elsewhere | yes
            | forward verbatim" row is contradicted by the switcher case; :161 states a
            three-argument signature where four shipped; and all 43 checkboxes are unticked
            including M1's Tasks 1-5, whose work demonstrably landed. The rule, covering this
            and BR-25: before the boundary tick, diff the plan against the tree in BOTH
            directions — every backticked symbol resolves in the tree, and every table or
            signature the artifact calls "complete" is re-derived from the code — and each
            delta gets a `## Revisions` entry. Important rather than Critical: the code is
            right and the guard is blind to the row, so this is documentation drift, not the
            silent coverage hole BR-3 was.
          family: plan-table-claims-unshipped-code
          round: 8
        - id: BR-29
          severity: Minor
          title: mouseinput.go:50's `s == ""` is unreachable after the HasPrefix check succeeds
          detail: |-
            Carried over verbatim from termcmd's parseSGRMousePress in the promotion. Dead
            condition, no behaviour change either way.
          family: move-residue
          round: 8
        - id: BR-30
          severity: Minor
          title: TestChildWithoutTrackingReceivesNoMouseBytes only inspects the first byte of each write
          detail: |-
            console_mouse_test.go:145-149 checks w[0] == 0x1b, so a report concatenated
            behind other bytes would pass. The split test covers this case today, but the
            assertion is weaker than its name and would not catch a batching change.
          family: unpinned-exported-shape
          round: 8
        - id: BR-31
          severity: Minor
          title: artifactpath/manifest.go:549,557 break NonArtifactSources' alphabetical order
          detail: |-
            cmd/internal/mouseinput/mouseinput.go is inserted inside the couchcore block and
            cmd/internal/couchtty/mouse.go sits before couchtty/menu.go. Nothing enforces the
            order, which is why it drifted.
          family: docs-lag-shipped-surface
          round: 8
      boundary: M1
      blocked: false
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

## Round 3 — 2026-09-05T13:44:27-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Disposition table is in the plan and in RouteMouseReport; release-to-a-no-mouse-child swallows, MaxReport bounds the hold, MouseHit.Raw carries the wire bytes.
- BR-2 — addressed — Task 12 now names menu.go:19, README.md, atlas/couch.md and the readme_test guard.
- BR-3 — not-addressed — Status flipped to `planned — M2`, but the symbol hostty.MouseClickTracking still exists nowhere and M2 shipped EnableMouseClicks/DisableMouseClicks in this same window; no Revisions entry was appended.
- BR-4 — not-addressed — The named mutation reddens only because the fixture's `beta` is Active; `(a.Active || used-start >= textwidth.Width(label))` still leaves the whole suite green. Neither ask landed.
- BR-5 — not-addressed — Verified: clampExtents -> `return extents` is green, and `index := len(lines) + start` is green. Neither the clipped nor the scrolled path is reachable.
- BR-6 — addressed — mouseinput_test.go exists and covers Parse/ParsePrefix/Find/IsPrefix/MaxReport directly; the out-of-scope row remains pair#188's structural issue.
- BR-7 — addressed — ChipSpan, ColumnToActor, ActorExtent and PointToActor all state ZERO-BASED and name the 1-based conversion; onMouse does it once.
- BR-8 — addressed — 000173 -> 000192 in the comment and all six entries; workshop/issues/000192-... exists.
- BR-9 — not-addressed — reserve.go:93-94 still runs the RenderStatusRow doc block straight into ChipSpan's; RenderStatusRow (:133) has no doc comment. A second instance now exists at menu.go:199-208.
- BR-10 — not-addressed — termcmd/run.go:453,459 still switch on the literals 64 and 65 while mouseinput.WheelUp/WheelDown exist.
- BR-11 — not-addressed — mousePressEvent (run.go:579) and parseSGRMousePressPrefix (:585) still have zero callers.
- BR-12 — addressed — keys.go:301 enforces MaxReport and TestAnUnterminatedMousePrefixDoesNotParkTheKeyboard pins it.
- BR-13 — not-addressed — Verified: deleting the run-merge at menu_render.go:463-467 leaves the suite green.

### Raised

- **BR-14** [Critical] `unpinned-exported-shape` Three shipped M2/M3 behaviours are unpinned - removing each leaves the whole couchtty suite green
  This is the 2nd finding in family `unpinned-exported-shape`. Do NOT patch the three
  sites. State the rule: every behaviour this issue introduces is pinned by a test that
  reddens when the behaviour is removed, and the enumeration is already written - the
  plan's "Done-when to Task map". The check is the mutation, not the green run.
  Measured prevalence, all verified in the working tree and restored: deleting
  c.writeOwn(hostty.EnableMouseClicks) (console.go:528) - green; discarding
  child.child.Write(hit.Raw) (console.go:1503) - green; deleting `&& !origin.Manual`
  (console.go:1455) - green, which is the marker the commit message headlines and the
  one Plan Task 11 Step 2 calls not optional. Four map rows have no test at all:
  TestClickInTheSwitcherTakesTheReturnPath, TestClickIsAManualSwitch,
  TestForwardPreservesRawBytes, TestTeardownDisablesMouseTracking.
- **BR-15** [Important] `milestone-scope-overrun` M2 and M3 production code landed inside the M1 window, so their own boundary reviews open on an empty range
  f95da992 ships routing, the Interceptor's SGR arm, couch's mode enable, onMouse and the
  manual-switch marker - all tagged M2/M3 in the plan - inside the window this M1 gate
  reviews. Once M1 closes at f95da992, BASE_SHA for M2 and M3 is that commit, so the
  mandatory fresh-eyes review for the milestone the plan calls "the work" sees nothing.
  That is the gate that was supposed to catch the Critical above. Either split so M2/M3
  land after M1 closes, or close all three against this window and pay their tests now.
- **BR-16** [Important] `unspecified-event-policy` couch's own mouse mode is written once and never re-asserted, so a child's DECRST silently ends the feature
  This is the 3rd finding in family `unspecified-event-policy`. Do NOT add a re-assert
  and stop. The rule the family keeps asking for: the disposition table must be crossed
  with the CHILD's mode-transition events, not only the operator's report events.
  console.go:528 is the only write of EnableMouseClicks; ?1000 is terminal-global, so a
  child emitting \x1b[?1000l - nvim does this on exit - turns couch's clicks off for the
  whole terminal with nothing to restore them. ptychild.Screen.Mouse() already observes
  the transition, so couch has the signal and no policy. Enumeration: child enables;
  child disables; child exits with mouse on; switch between two children with different
  modes; replay re-asserting the child's modes over couch's.
- **BR-17** [Important] `parallel-handler-restates-decision` The click reducer restates Enter's switch/resume rule and diverges from it, and sets Manual on a refused dispatch
  menu.go:349-365 duplicates reduceRootKey's operation choice (menu.go:475-479); a
  non-actionable row gets a notice from Enter and silence from a click; and
  state.InFlight.Manual = true (menu.go:363) runs unconditionally, including when
  dispatchMenuOperation refused because another operation was in flight
  (menu.go:1542-1544) - writing the flag onto a different operation's origin. The
  Done-when requires the same handler, not a parallel one (ARCH-DRY). Extract the
  actionable-thread to operation decision into one function both arms call, and set
  Manual only when the dispatch produced effects.
- **BR-18** [Important] `docs-lag-shipped-surface` atlas and README cover M1's geometry but not the routing, ownership and click gesture that shipped in the same window
  atlas/couch.md gained the M1 geometry in 1c895d4f and nothing for f95da992: the
  four-way disposition table, the narrowed release rule, couch's ownership of
  ?1000;?1006, and the manual-switch classification. menuControls (menu.go:19-33) has no
  mouse row, so TestREADMEDocumentsEveryPanelControl cannot fire and README never
  mentions that a click in the switcher selects and enters - a user-facing gesture that
  is live in the binary today. Task 12 defers this to M3's close, which only works if
  the code also waits for M3.
- **BR-19** [Minor] `orphaned-doc-comment` MenuEventNotice's doc block is now attached to MenuEventMouseSwitch, a second instance in the commit that left the first
  This is the 2nd finding in family `orphaned-doc-comment`; BR-9 is the first and is
  re-raised not-addressed. Do NOT fix this instance alone. The rule: a symbol inserted
  into an existing file must go after the symbol an adjacent doc block documents, or the
  block must be re-anchored. Enumeration for this issue: reserve.go:93-94 (RenderStatusRow's
  block now reads as ChipSpan's) and menu.go:199-208 (MenuEventNotice's block now reads as
  MenuEventMouseSwitch's). Both are checkable with `go doc`.
- **BR-20** [Minor] `move-residue` hostty.DisableMouseClicks joins two other zero-call-site symbols left by the promotion
  This is the 2nd finding in family `move-residue`; BR-11 is the first and is re-raised
  not-addressed. Do NOT delete this one symbol. The rule: a symbol this issue introduces
  or leaves behind has a production call site by the milestone that introduces it, or it
  is deleted. Enumeration: termcmd/run.go:579 (mousePressEvent alias), :585
  (parseSGRMousePressPrefix), hostty/control.go:62 (DisableMouseClicks), and
  mouseinput.WheelUp/WheelDown which exist only in tests while run.go:453,459 keep the
  literals. deadSymbolScope is cmd/internal/couchcore, so no guard sees any of them.

## Round 4 — 2026-09-05T16:25:23-07:00 (claude) — BLOCKED

### Disposed

- BR-3 — not-addressed — Status flipped, but no plan Revisions entry, the row still names a symbol that exists nowhere, and six rows now claim `planned` for shipped code.
- BR-4 — not-addressed — Verified: dropping the span for any chip truncated below 3 columns leaves the whole suite green.
- BR-5 — not-addressed — Verified twice: `return extents` and a scroll-offset re-base both stay green; the new fixture enters neither path.
- BR-9 — not-addressed — `go doc ChipSpan` still prints RenderStatusRow's sanitize rationale; RenderStatusRow still has no doc.
- BR-10 — not-addressed — run.go:453,459 still switch on the 64/65 literals; WheelUp/WheelDown remain test-only.
- BR-11 — not-addressed — run.go:579 and :585 still have zero references anywhere in the tree.
- BR-13 — not-addressed — Verified: deleting the run-merge at menu_render.go:465-469 leaves the suite green.
- BR-14 — not-addressed — Two of the three named sites are now pinned, but three of THIS round's own fixes are not; seven unpinned behaviours measured.
- BR-15 — not-addressed — Merging M3 into M2 does not give M2 a diff; its close still opens on an empty range, carrying Task 12's manual check and pair#166 with it.
- BR-16 — not-addressed — The re-assert can be deleted with the suite green, and the child-mode-transition enumeration was never written.
- BR-17 — not-addressed — Both fixes are unpinned by mutation, and a non-actionable row still gets a notice from Enter and silence from a click.
- BR-18 — addressed — atlas/couch.md, README and the menuControls row all landed; the README guard fires on the new row.
- BR-19 — not-addressed — menu.go:200-209 unchanged; MenuEventNotice's block still reads as MenuEventMouseSwitch's.
- BR-20 — not-addressed — DisableMouseClicks, mousePressEvent, parseSGRMousePressPrefix and WheelUp/WheelDown all still have zero production call sites.

### Raised

- **BR-21** [Minor] `guard-not-registered` The handler-table entry for HitMouse is a stub the dispatcher never reaches, so the enumeration guard proves nothing for it
  This is the 2nd finding in family `guard-not-registered`. Do NOT just delete
  or fill in this one entry. The rule: an enumeration guard is satisfied only
  by the thing it guards -- registering a value the production path never
  reads converts the guard into a formality that reports coverage it does not
  have. console.go:647 special-cases HitMouse BEFORE consulting the table, so
  hitHandlers()[HitMouse] (console.go:1557) is a `func() {}` with no caller;
  AllInterceptorHits still counts it as proven. The same shape is one edit
  away for any future payload-carrying hit. Either widen the table's value to
  carry the payload so every hit really does route through it, or have the
  guard assert reachability rather than presence. Related: Interceptor.mouse
  is a payload valid only between one FeedHit and the next, kept correct by a
  hand-written ordering rather than by the type (ARCH-ORDER).

## Round 5 — 2026-09-05T16:47:31-07:00 (claude) — BLOCKED

### Disposed

- BR-3 — addressed — Row is now hostty.EnableMouseClicks/DisableMouseClicks and both symbols exist; the prose bullet at plan:193 still says MouseClickTracking, raised separately as Minor.
- BR-4 — addressed — Verified: mutating reserve.go:175 to require the full label width reddens reserve_mouse_test.go:60 and :124.
- BR-5 — not-addressed — Clamp half honestly dispositioned; the SCROLLED half is still unreachable -- the index mutation stays green because every extent test selects inventory[0].
- BR-9 — not-addressed — go doc ChipSpan still prints RenderStatusRow's rationale and go doc RenderStatusRow prints nothing; the added comment describes the fix instead of being it.
- BR-10 — addressed — run.go:453,459 now switch on mouseinput.WheelUp/WheelDown; no 64/65 literals remain.
- BR-11 — not-addressed — parseSGRMousePressPrefix is gone but the mousePressEvent alias at run.go:581 still has zero callers, under a comment claiming the aliases were removed.
- BR-13 — addressed — Verified: deleting the run-merge reddens TestClampAndRunMergeAreLoadBearing.
- BR-14 — addressed — All three sites verified RED under mutation, and the four unmapped Done-when rows now have tests that reach production.
- BR-15 — addressed — The tests were paid inside this window and this M1 review covers the M2/M3 code; residual is that M2's own close now opens on a near-empty range and should be recorded as covered here.
- BR-16 — not-addressed — The re-assert landed but is unpinned (deleting it leaves the suite green) and it introduces a regression -- see the new Critical; the crossed enumeration was never written.
- BR-17 — not-addressed — enterOperationFor and the effects-gated Manual flag both landed and mutate RED; the third half survives -- a non-actionable row still gets a notice from Enter and silence from a click.
- BR-19 — not-addressed — Unchanged: go doc MenuEventMouseSwitch prints MenuEventNotice's block and MenuEventNotice has none.
- BR-20 — not-addressed — WheelUp/WheelDown got production callers, but DisableMouseClicks and the mousePressEvent alias still have none, and seqMouse joins the enumeration as a new zero-reachability symbol.
- BR-21 — not-addressed — console.go:647 still special-cases HitMouse before the table and hitHandlers()[HitMouse] is still an uncalled func(){} counted by the guard.

### Raised

- **BR-22** [Critical] `unspecified-event-policy` The per-paint ?1000h re-assert demotes a child that enabled ?1002/?1003, breaking the "child keeps its own events unchanged" Done-when
  console.go:1024-1030 writes hostty.EnableMouseClicks on every paint. Mouse-tracking
  modes 1000/1002/1003 are one mutually-exclusive state in xterm (send_mouse_pos),
  Alacritty (removes MOUSE_MODE before inserting), kitty, Ghostty and iTerm2, so
  setting 1000 while a child holds 1002 replaces button-event tracking with
  press/release only. Screen.Mouse() still reports true, so couch keeps forwarding
  presses into a child that will never receive the motion that closes its drag --
  the "nvim stuck in visual selection" symptom mouseinput.go:19-23 documents.
  paintNow is on the rowDirty/bell/attention path, so a full-screen child triggers
  it continuously. The code comment and atlas/couch.md both assert the opposite.
  This is the 3rd finding in family unspecified-event-policy. Do NOT patch this one
  site. The rule the family keeps asking for: the disposition table must be crossed
  with the CHILD's mode-transition events, and each cell must state what couch
  writes AND what mode the child is left holding. Enumeration (unchanged from
  BR-16): child enables 1000; child enables 1002/1003; child disables; child exits
  with mouse on; switch between two children with different modes; replay
  re-asserting over couch's. Measured prevalence: 1 of 6 cells implemented, 0 of 6
  pinned -- deleting the paintNow re-assert leaves the whole couchtty suite green.
- **BR-23** [Important] `guard-not-registered` assertDirectTest is satisfied by the contract's own conceptInventory literal, so every PURE row's coverage assertion is vacuous
  core_concepts_contract_test.go:377-394 globs *_test.go in the package and greps
  for the row's symbol; the contract file is in that glob and holds every symbol
  name verbatim in conceptInventory, so the match always succeeds. seqMouse is the
  demonstration -- its only occurrence in any _test.go is line 73 of the contract
  itself, and seqMouse is dead code (sequenceAt can never return it; deleting
  `case seqMouse: return HitMouse` leaves the suite green).
  This is the 3rd finding in family guard-not-registered. Do NOT fix this one
  guard. The rule covering both instances: an enumeration guard is satisfied only
  by an artifact the production path reads. hitHandlers()[HitMouse] is a func(){}
  the dispatcher skips; assertDirectTest matches on the inventory that declares the
  requirement. Measured prevalence: 2 of 2 enumeration guards this issue touches
  are self-satisfying. Fix: exclude the contract file from the glob, then give
  whatever goes red a real test.
- **BR-24** [Minor] `promoted-constant-with-surviving-literal` sgrMouseSize derives Terminators from mouseinput but restates the report-length rule
  termcmd/rename_input.go:175-186 computes the report size as idx+3+1 after
  scanning for mouseinput.Terminators. mouseinput.ParsePrefix already answers that
  question as len(raw), and the package doc says it exists so there is ONE answer
  to "where does this sequence end". This is the 2nd finding in family
  promoted-constant-with-surviving-literal. Do NOT just rewrite this call. The
  rule: a promotion is finished when every consumer derives the RULE, not only the
  constant the diff happened to touch. Enumeration for this issue:
  termcmd/run.go:453,459 (fixed this round), rename_input.go:183 (constant derived,
  length rule restated). Note the two differ on malformed input -- ParsePrefix
  validates the numbers and sgrMouseSize does not -- so the sweep must state which
  behaviour wins.
- **BR-25** [Minor] `plan-table-claims-unshipped-code` The plan's Integration-points prose bullet still names hostty.MouseClickTracking after the Revisions entry renamed it
  workshop/plans/000172-mouse-support-status-bar-and-switcher-plan.md:193 reads
  "**hostty.MouseClickTracking**", seven lines below a Revisions entry recording
  that it became EnableMouseClicks/DisableMouseClicks. A reader grepping the plan
  for the symbol finds a name that exists nowhere in the tree.
  This is the 2nd finding in family plan-table-claims-unshipped-code. Do NOT edit
  only this line. The rule: a rename inside an artifact is finished when
  grep <old-name> over the artifact returns zero, not when the machine-parsed row
  is fixed -- the contract test reads the table, so the table is the only part the
  last fix was forced to get right.

## Round 6 — 2026-09-05T20:53:23-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 7 — 2026-09-05T22:22:18-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 8 — 2026-09-05T22:40:16-07:00 (claude) — passed

### Disposed

- BR-5 — not-addressed — Run-merge and clamp bounds now pinned, but the scrolled list still enters no test: index := len(lines) + start*1000 leaves the whole package green.
- BR-9 — not-addressed — go doc ChipSpan still prints RenderStatusRow's rationale and go doc RenderStatusRow prints nothing; the block was explained, not moved.
- BR-11 — addressed — mousePressEvent and parseSGRMousePressPrefix are deleted; the two surviving wrappers have live call sites at run.go:418,471.
- BR-16 — addressed — Mutation-verified both ways: deleting the per-paint re-assert reddens 4 subtests, making it unconditional reddens 4 others.
- BR-17 — not-addressed — Both halves revert green (enterOperationFor -> "switch"; Manual set unconditionally), and the non-actionable notice divergence remains.
- BR-19 — addressed — MenuEventMouseSwitch now carries its own doc and MenuEventNotice's block is re-anchored.
- BR-20 — addressed — All four enumerated symbols swept: DisableMouseClicks and the alias deleted, WheelUp/WheelDown now used at run.go:453,459.
- BR-21 — addressed — hitHandlers()[HitMouse] is a real handler; no-oping it reddens four console tests.
- BR-23 — addressed — Verified by deleting mouse_test.go — both PURE rows now report uncovered instead of matching the inventory literal.
- BR-22 — addressed — The paintNow site is fixed and pinned; the same mechanism survives through Screen.Mouse()'s mode conflation, raised separately.
- BR-24 — addressed — sgrMouseSize derives the length from ParsePrefix and the malformed-input winner is stated.
- BR-25 — not-addressed — The named bullet is fixed; the class is not — seqMouse, IsSGRPrefix, FindSGR and DisableMouseClicks still name symbols absent from the tree.

### Raised

- **BR-26** [Important] `unspecified-event-policy` couch writes two mouse modes and governs one, and the state it reads collapses tracking and encoding into a single bool
  This is the 4th finding in family `unspecified-event-policy`. Do NOT add a fifth
  site fix. Probed and confirmed at the pinned head: a child doing `\x1b[?1002h`
  then `\x1b[?1006l` flips Screen.Mouse() false (screen.go:415 maps 1000/1002/1003
  /1006 onto one bool), so the next paint writes `?1000;1006h` and demotes it —
  BR-22's mechanism reached through the observation instead of the write; and a
  child doing `\x1b[?1006h` alone makes couch stand back for a child holding no
  tracking at all. Two more halves: `?1006` is never stood back from, so a child
  enabling `?1000h` without `?1006h` receives SGR reports it cannot parse (against
  the "receives its own events unchanged" Done-when), and the startup write at
  console.go:528 is gated by nothing. The take-back also has no trigger of its own:
  onChunk repaints only on RowDirty/Bell/attention and the DECSET handler sets
  rowDirty only for alt-screen, so a bare `?1000l` leaves couch's clicks off
  indefinitely; TestMouseModeTransitions cannot see this because it calls repaint()
  by hand and its FakeChild has a nil sink. The rule: couch's mouse ownership is ONE
  gated writer over the full set of modes it touches — tracking AND encoding —
  driven by an observation that keeps them as separate state, and every cell of
  (child mode transition x couch write) states what couch writes, what mode AND
  encoding the child is left holding, and what EVENT re-evaluates the cell.
  Measured prevalence: 3 writers, 1 governed; 2 modes written, 1 governed; 1
  observation collapsing 4 modes into 1 bool; 6 cells pinned, 0 pinning the trigger.
- **BR-27** [Important] `docs-lag-shipped-surface` atlas/couch.md:356-361 still teaches the refuted model that produced BR-22
  This is the 2nd finding in family `docs-lag-shipped-surface` (BR-18 was the
  first). Do NOT patch only this paragraph. It reads "couch RE-ASSERTS its own on
  every paint ... DECSET is additive and idempotent, so this cannot clobber a mode
  the child set for itself"; both halves are false at this head, since console.go
  :1040 re-asserts only while !childWantsMouse() precisely because 1000/1002/1003
  replace one another. The rule: when a boundary changes a behaviour the atlas
  already describes, the same commit rewrites the contradicted paragraph — the
  check is grepping the atlas for the mechanism the diff changed, not "did I add a
  section". Measured prevalence: 1 of 1 contradicted paragraph left standing while
  a new section for the same feature landed in the same window.
- **BR-28** [Important] `plan-table-claims-unshipped-code` The plan artifact disagrees with the tree in six places, with no Revisions entry
  This is the 3rd finding in family `plan-table-claims-unshipped-code`. Do NOT fix
  only the row a test reads. Sites: :182 declares DisableMouseClicks status `new`
  and it exists nowhere (declared path is outside conceptPackage, so the contract
  test is blind to it, the same gap BR-3 exploited); :118, :165, :394, :396
  describe seqMouse as shipped work; :396 names mouseinput.IsSGRPrefix and :155
  names mouseinput.FindSGR, which shipped as IsPrefix and Find; the section titled
  "The complete disposition table" (:41-56) has no couchOwnsScreen dimension and so
  omits the two switcher rows the code carries, and its "anything, elsewhere | yes
  | forward verbatim" row is contradicted by the switcher case; :161 states a
  three-argument signature where four shipped; and all 43 checkboxes are unticked
  including M1's Tasks 1-5, whose work demonstrably landed. The rule, covering this
  and BR-25: before the boundary tick, diff the plan against the tree in BOTH
  directions — every backticked symbol resolves in the tree, and every table or
  signature the artifact calls "complete" is re-derived from the code — and each
  delta gets a `## Revisions` entry. Important rather than Critical: the code is
  right and the guard is blind to the row, so this is documentation drift, not the
  silent coverage hole BR-3 was.
- **BR-29** [Minor] `move-residue` mouseinput.go:50's `s == ""` is unreachable after the HasPrefix check succeeds
  Carried over verbatim from termcmd's parseSGRMousePress in the promotion. Dead
  condition, no behaviour change either way.
- **BR-30** [Minor] `unpinned-exported-shape` TestChildWithoutTrackingReceivesNoMouseBytes only inspects the first byte of each write
  console_mouse_test.go:145-149 checks w[0] == 0x1b, so a report concatenated
  behind other bytes would pass. The split test covers this case today, but the
  assertion is weaker than its name and would not catch a batching change.
- **BR-31** [Minor] `docs-lag-shipped-surface` artifactpath/manifest.go:549,557 break NonArtifactSources' alphabetical order
  cmd/internal/mouseinput/mouseinput.go is inserted inside the couchcore block and
  cmd/internal/couchtty/mouse.go sits before couchtty/menu.go. Nothing enforces the
  order, which is why it drifted.

## Open findings

- **BR-5** [Important] `onedirectional-geometry-assertion` clampExtents and the scrolled list are unreachable from any test
- **BR-9** [Minor] `orphaned-doc-comment` RenderStatusRow's doc block is now attached to ChipSpan
- **BR-17** [Important] `parallel-handler-restates-decision` The click reducer restates Enter's switch/resume rule and diverges from it, and sets Manual on a refused dispatch
- **BR-25** [Minor] `plan-table-claims-unshipped-code` The plan's Integration-points prose bullet still names hostty.MouseClickTracking after the Revisions entry renamed it
- **BR-26** [Important] `unspecified-event-policy` couch writes two mouse modes and governs one, and the state it reads collapses tracking and encoding into a single bool
- **BR-27** [Important] `docs-lag-shipped-surface` atlas/couch.md:356-361 still teaches the refuted model that produced BR-22
- **BR-28** [Important] `plan-table-claims-unshipped-code` The plan artifact disagrees with the tree in six places, with no Revisions entry
- **BR-29** [Minor] `move-residue` mouseinput.go:50's `s == ""` is unreachable after the HasPrefix check succeeds
- **BR-30** [Minor] `unpinned-exported-shape` TestChildWithoutTrackingReceivesNoMouseBytes only inspects the first byte of each write
- **BR-31** [Minor] `docs-lag-shipped-surface` artifactpath/manifest.go:549,557 break NonArtifactSources' alphabetical order
