---
gate: plan-quality
issue: 172
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-05T12:24:23-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: Plan builds a new mouse-mode scanner and SGR parser; both already exist and are unnamed
          detail: |-
            Spec claims "the only handling is a blanket ResetInteractiveModes at teardown". But
            ptychild/screen.go:412-416 already scans child DECSET/DECRST for 1000/1002/1003/1006 into
            Screen.mouse, per-child (child.go:62,114,133), exposed as Child.Mouse() (child.go:282-285),
            consumed by termcmd/run.go:873 — and screen_test.go:164 already asserts the whole-vs-split
            chunk-boundary equivalence Task 5 proposes to write from scratch. termcmd/run.go:583-637
            already decodes SGR (parseSGRMousePress/Prefix, findSGRMousePress, isSGRMousePrefix). The
            real delta is widening Screen.mouse bool into a mode set, not a second scanner in couchtty
            (ARCH-DRY). Enumerate the existing surface and justify reuse-or-replace per item. Also the
            citation is wrong: ResetInteractiveModes is console.go:891, not console.go:737.
          family: reimplements-existing-mechanism
          round: 1
        - id: PQ-2
          severity: Critical
          title: The swallow requirement forces changes to keys.go Interceptor, which no task lists
          detail: |-
            sequenceAt (keys.go:302-317) matches fixed literals from knownSequences; a variable-length
            SGR report classifies seqNone and FeedHit copies its bytes to the child (keys.go:286-291).
            A split report is not seqPartial either, so half leaks. FeedHit's (before, hit, rest)
            contract has no DROP channel. The Done-when "child receives zero mouse bytes" is
            unachievable without variable-length mouse framing plus a new Interceptor outcome, yet
            Tasks 7-9 list only console.go and hostty/control.go.
          family: unnamed-seam-change
          round: 1
        - id: PQ-3
          severity: Important
          title: HitMouse cannot dispatch through hitHandlers() — the table is map[InterceptorHit]func()
          detail: |-
            Task 8 says dispatch "like every other intercepted input", but console.go:1465-1472 is
            parameterless and console_relaunch_chord_test.go:316-322 pins
            len(handlers) == len(AllInterceptorHits()). A click carries (col,row). Widening the handler
            signature touches all five handlers and the walk test; bypassing the table breaks the
            one-table-a-test-can-walk property defended at console.go:631-636. State the choice.
          family: unnamed-seam-change
          round: 1
        - id: PQ-4
          severity: Important
          title: No button filter and no press/release policy; wheel events and unmatched presses are live hazards
          detail: |-
            ?1000 reports wheel as buttons 64/65 and sends both M and m. Nothing in Tasks 6 or 8 says
            which button/edge acts, so a wheel scroll over the status row would switch actors and a
            click would double-fire. termcmd/run.go:575-581 records the cost of splitting a pair: an
            unmatched press leaves nvim stuck in a visual-selection drag. Press and release must cross
            the couch/forward/swallow boundary together.
          family: unspecified-event-policy
          round: 1
        - id: PQ-5
          severity: Important
          title: Replay-on-switch already re-asserts the child's mouse modes; the plan adds a second writer
          detail: |-
            switchTo repaints via takeOverScreen(child.ReplayThrough(...)) (console.go:486-490) and
            replay.go:44-46 deliberately preserves DECSET \x1b[?1006h in the replay stream. So switching
            already writes the arriving child's modes when the DECSET is inside the ring window — likely
            the root of pair#166. The ARCH-ORDER row "switch to B, write B's not A's" needs to say which
            writer wins and what happens when the replay carries a stale or absent DECSET (ARCH-ORDER).
          family: undeclared-concurrent-writer
          round: 1
        - id: PQ-6
          severity: Important
          title: The plan's Core concepts table is not registered in conceptPlans, so all twelve rows go unpinned
          detail: |-
            conceptPlans (core_concepts_contract_test.go:227-236) is a hand-maintained list of three
            plans and does not include 000172. The test's own comment says this is "exactly how pair#182's
            table went unenforced and shipped two rows naming symbols that exist nowhere." Task 10 covers
            the menuControls README guard and atlas but not this. Add 000172 to conceptPlans and the rows
            to conceptInventory (:28-70), or state the dependency on pair#188 (ARCH-PURPOSE).
          family: guard-not-registered
          round: 1
        - id: PQ-7
          severity: Important
          title: '"nvim selection and scroll still work in an attached session" has no task and no manual step'
          detail: |-
            Task 6 asserts `forward` at the pure level only; nothing exercises the live attached path.
            AGENTS.md section 2 requires an automated check or written manual steps in the Plan.
          family: donewhen-without-verification
          round: 1
        - id: PQ-8
          severity: Minor
          title: Actor extents already exist as rootLine.actorStart; and RenderMenuView shifts rows after the fact
          detail: |-
            renderRootMenuFrame (menu_render.go:340-388) already builds rootLine with actorStart, already
            appends per-actor attention rows (:365-369), and already scroll-windows on actor boundaries
            (:372-378) — so multi-line actors are testable today without #173. RenderMenuView (:100-127)
            then replaces lines[0] with the breadcrumb and inserts the notice at index 1, shifting every
            actor row down one, so extents must be re-based there.
          family: reimplements-existing-mechanism
          round: 1
        - id: PQ-9
          severity: Minor
          title: RenderStatusRow's signature change has a cross-package caller
          detail: |-
            cmd/internal/wrapcmd/codex_working_test.go:166 calls couchtty.RenderStatusRow. Compiler-caught,
            but the plan marks the row "modified" without naming who else breaks.
          family: undeclared-blast-radius
          round: 1
        - id: PQ-10
          severity: Minor
          title: No non-goals section, and no ARCH-SECURE statement for input the component did not produce
          detail: |-
            Unstated: drag, hover, right/middle click, wheel, text selection and copy on couch's row, and
            whether the status-row notice area is clickable. The plan has an ARCH-ORDER section but no
            ARCH-SECURE line covering terminal-supplied reports and the child's DECSET stream.
          family: missing-non-goals
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-05T12:32:49-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: Plan file fixed; the issue Spec/Done-when/M2 row still instruct the scanner and still cite console.go:737 (it is 891), with no Revisions block.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: seqMouse + HitMouse + the all-splits test give the Interceptor the DROP channel it lacked.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: 'Choice stated: keep map[InterceptorHit]func(), carry the payload out of FeedHit.'
          round: 2
        - id: PQ-4
          disposition: not-addressed
          note: '"Release always forwards" breaks the press/release pairing in exactly the no-mouse-child case, contradicting the zero-bytes Done-when.'
          round: 2
        - id: PQ-5
          disposition: addressed
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Verified the `planned` status really is skipped at core_concepts_contract_test.go:93,293.
          round: 2
        - id: PQ-7
          disposition: addressed
          round: 2
        - id: PQ-8
          disposition: not-addressed
          note: Right layer chosen, but the notice-insert row shift and actorStart reuse are still unnamed and untested.
          round: 2
        - id: PQ-9
          disposition: addressed
          round: 2
        - id: PQ-10
          disposition: addressed
          round: 2
      findings:
        - id: PQ-11
          severity: Critical
          title: 'The switcher surface has no input path: PointToActor is planned but no task names console_menu.go or panelkeys.go'
          detail: |-
            This is the 2nd finding in family `donewhen-without-verification`. Do NOT fix
            this instance alone — state the rule and sweep it: EVERY Done-when bullet maps
            to a named task with a named test, written into the plan as an explicit
            Done-when-to-Task map. PQ-7 was the first instance (nvim scroll had no step);
            this is the second, and writing that map is what would have caught it.
            The instance: three Done-when bullets cover the switcher, and M1 builds and
            tests PointToActor, but nothing delivers a click to it. Panel input flows
            processInput -> it.FeedHit -> route(before) -> onMenuInput -> DecodePanelKeys
            (console.go:606-627, console_menu.go:170). Task 7 makes the Interceptor CONSUME
            the report as HitMouse, so it is stripped from `before` and never reaches
            route, hence never reaches the panel. PanelKeyKind has no mouse member
            (panelkeys.go:14-25) and DecodePanelKeys drops unused sequences — its comment
            at :36-42 is about SGR mouse reports specifically. Neither panelkeys.go nor
            console_menu.go appears in any task or in Core concepts. Compounding it,
            RouteMouseReport(event, hostRows, childWantsMouse) has no focus input, so a
            panel-open click above the last row returns swallow/forward when it must
            return couch — the panel state is unrepresentable in the signature
            (ARCH-ORDER). The panel occupies size.Rows-1 (console_menu.go:186), so both
            surfaces are live at once and focus is genuinely part of the routing state.
          family: donewhen-without-verification
          round: 2
        - id: PQ-12
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
          family: unspecified-event-policy
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-05T12:40:30-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: '"What already exists" table added; parser MOVED not rewritten; all four citations verified.'
          round: 3
        - id: PQ-4
          disposition: addressed
          note: Button-0-only rule, wheel and release rows, plus a non-goals entry for the other buttons.
          round: 3
        - id: PQ-8
          disposition: addressed
          note: Extents derived from rootLine.actorStart and emitted from RenderMenuView after the notice insert.
          round: 3
        - id: PQ-11
          disposition: addressed
          note: Done-when-to-Task map written; the switcher input path is named and its "no change" justified.
          round: 3
        - id: PQ-12
          disposition: not-addressed
          note: |-
            Instances A/B/C fixed, but the table never crosses the classes with the panel-focused
            state PQ-12 named, and RouteMouseReport's signature still cannot express it.
          round: 3
      findings:
        - id: PQ-13
          severity: Critical
          title: A switcher click cannot both take Return's path and be an unconditional manual switch
          detail: |-
            This is the 3rd finding in family `unnamed-seam-change` (PQ-2, PQ-9). Do not fix only
            this instance: the rule is that every task's Files list must name every production site
            its steps assert, and any "reuse the path X takes" claim must say what X decides where
            the new caller differs. Sweep all twelve tasks; Task 11 and Task 10 fail it today.
            The instance: KeyEnter dispatches "switch" (menu.go:445-449) into runMenuOperation,
            which captures pending attention (console.go:1434-1436, attention.go:75-79), and
            ExecuteConsoleOperation then sets how = arrivalNotification whenever that capture is
            nonzero (console.go:1625-1628). So a click on the notification line of a notifying
            actor, taking exactly Return's path, is classified arrivalNotification -- the one thing
            the Done-when forbids. Task 11 says to "mutate the call to arrivalNotification", but no
            caller passes an arrival on that path, and Task 11 lists only console_mouse_test.go.
            Say which side gives and name the production site that carries it.
          family: unnamed-seam-change
          round: 3
        - id: PQ-14
          severity: Minor
          title: New production file couchtty/mouse.go has no NonArtifactSources registration step
          detail: |-
            This is the 2nd finding in family `guard-not-registered` (PQ-6 was conceptPlans). The
            rule, not the instance: every guard whose input is a hand-maintained list gets its
            registration step in the task that creates the artifact. NonArtifactSources
            (artifactpath/manifest.go:482-560) lists every couchtty/*.go file; Task 1 registers
            mouseinput but Task 6 creates couchtty/mouse.go with no entry. make test catches it.
          family: guard-not-registered
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-05T12:46:13-07:00"
      agent: claude
      dispose:
        - id: PQ-12
          disposition: not-addressed
          note: The three instances are fixed; the enumeration PQ-12 asked for was written on the child-mode axis only.
          round: 4
        - id: PQ-13
          disposition: addressed
          note: Resolution names the site (console.go:1434 capture skipped for a manual origin); Task 10/11 Files lists now carry it.
          round: 4
        - id: PQ-14
          disposition: addressed
          note: Task 6 registers couchtty/mouse.go in NonArtifactSources and states the rule for every future production file.
          round: 4
      findings:
        - id: PQ-15
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
          family: unnamed-seam-change
          round: 4
      blocked: false
content_hash: cbb46be433aa8d23567a820ea22642124c66f770e26f958cd9f2f6b89cc4836f
---

# Gate ledger — pair#172 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-05T12:24:23-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `reimplements-existing-mechanism` Plan builds a new mouse-mode scanner and SGR parser; both already exist and are unnamed
  Spec claims "the only handling is a blanket ResetInteractiveModes at teardown". But
  ptychild/screen.go:412-416 already scans child DECSET/DECRST for 1000/1002/1003/1006 into
  Screen.mouse, per-child (child.go:62,114,133), exposed as Child.Mouse() (child.go:282-285),
  consumed by termcmd/run.go:873 — and screen_test.go:164 already asserts the whole-vs-split
  chunk-boundary equivalence Task 5 proposes to write from scratch. termcmd/run.go:583-637
  already decodes SGR (parseSGRMousePress/Prefix, findSGRMousePress, isSGRMousePrefix). The
  real delta is widening Screen.mouse bool into a mode set, not a second scanner in couchtty
  (ARCH-DRY). Enumerate the existing surface and justify reuse-or-replace per item. Also the
  citation is wrong: ResetInteractiveModes is console.go:891, not console.go:737.
- **PQ-2** [Critical] `unnamed-seam-change` The swallow requirement forces changes to keys.go Interceptor, which no task lists
  sequenceAt (keys.go:302-317) matches fixed literals from knownSequences; a variable-length
  SGR report classifies seqNone and FeedHit copies its bytes to the child (keys.go:286-291).
  A split report is not seqPartial either, so half leaks. FeedHit's (before, hit, rest)
  contract has no DROP channel. The Done-when "child receives zero mouse bytes" is
  unachievable without variable-length mouse framing plus a new Interceptor outcome, yet
  Tasks 7-9 list only console.go and hostty/control.go.
- **PQ-3** [Important] `unnamed-seam-change` HitMouse cannot dispatch through hitHandlers() — the table is map[InterceptorHit]func()
  Task 8 says dispatch "like every other intercepted input", but console.go:1465-1472 is
  parameterless and console_relaunch_chord_test.go:316-322 pins
  len(handlers) == len(AllInterceptorHits()). A click carries (col,row). Widening the handler
  signature touches all five handlers and the walk test; bypassing the table breaks the
  one-table-a-test-can-walk property defended at console.go:631-636. State the choice.
- **PQ-4** [Important] `unspecified-event-policy` No button filter and no press/release policy; wheel events and unmatched presses are live hazards
  ?1000 reports wheel as buttons 64/65 and sends both M and m. Nothing in Tasks 6 or 8 says
  which button/edge acts, so a wheel scroll over the status row would switch actors and a
  click would double-fire. termcmd/run.go:575-581 records the cost of splitting a pair: an
  unmatched press leaves nvim stuck in a visual-selection drag. Press and release must cross
  the couch/forward/swallow boundary together.
- **PQ-5** [Important] `undeclared-concurrent-writer` Replay-on-switch already re-asserts the child's mouse modes; the plan adds a second writer
  switchTo repaints via takeOverScreen(child.ReplayThrough(...)) (console.go:486-490) and
  replay.go:44-46 deliberately preserves DECSET \x1b[?1006h in the replay stream. So switching
  already writes the arriving child's modes when the DECSET is inside the ring window — likely
  the root of pair#166. The ARCH-ORDER row "switch to B, write B's not A's" needs to say which
  writer wins and what happens when the replay carries a stale or absent DECSET (ARCH-ORDER).
- **PQ-6** [Important] `guard-not-registered` The plan's Core concepts table is not registered in conceptPlans, so all twelve rows go unpinned
  conceptPlans (core_concepts_contract_test.go:227-236) is a hand-maintained list of three
  plans and does not include 000172. The test's own comment says this is "exactly how pair#182's
  table went unenforced and shipped two rows naming symbols that exist nowhere." Task 10 covers
  the menuControls README guard and atlas but not this. Add 000172 to conceptPlans and the rows
  to conceptInventory (:28-70), or state the dependency on pair#188 (ARCH-PURPOSE).
- **PQ-7** [Important] `donewhen-without-verification` "nvim selection and scroll still work in an attached session" has no task and no manual step
  Task 6 asserts `forward` at the pure level only; nothing exercises the live attached path.
  AGENTS.md section 2 requires an automated check or written manual steps in the Plan.
- **PQ-8** [Minor] `reimplements-existing-mechanism` Actor extents already exist as rootLine.actorStart; and RenderMenuView shifts rows after the fact
  renderRootMenuFrame (menu_render.go:340-388) already builds rootLine with actorStart, already
  appends per-actor attention rows (:365-369), and already scroll-windows on actor boundaries
  (:372-378) — so multi-line actors are testable today without #173. RenderMenuView (:100-127)
  then replaces lines[0] with the breadcrumb and inserts the notice at index 1, shifting every
  actor row down one, so extents must be re-based there.
- **PQ-9** [Minor] `undeclared-blast-radius` RenderStatusRow's signature change has a cross-package caller
  cmd/internal/wrapcmd/codex_working_test.go:166 calls couchtty.RenderStatusRow. Compiler-caught,
  but the plan marks the row "modified" without naming who else breaks.
- **PQ-10** [Minor] `missing-non-goals` No non-goals section, and no ARCH-SECURE statement for input the component did not produce
  Unstated: drag, hover, right/middle click, wheel, text selection and copy on couch's row, and
  whether the status-row notice area is clickable. The plan has an ARCH-ORDER section but no
  ARCH-SECURE line covering terminal-supplied reports and the child's DECSET stream.

## Round 2 — 2026-09-05T12:32:49-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — not-addressed — Plan file fixed; the issue Spec/Done-when/M2 row still instruct the scanner and still cite console.go:737 (it is 891), with no Revisions block.
- PQ-2 — addressed — seqMouse + HitMouse + the all-splits test give the Interceptor the DROP channel it lacked.
- PQ-3 — addressed — Choice stated: keep map[InterceptorHit]func(), carry the payload out of FeedHit.
- PQ-4 — not-addressed — "Release always forwards" breaks the press/release pairing in exactly the no-mouse-child case, contradicting the zero-bytes Done-when.
- PQ-5 — addressed
- PQ-6 — addressed — Verified the `planned` status really is skipped at core_concepts_contract_test.go:93,293.
- PQ-7 — addressed
- PQ-8 — not-addressed — Right layer chosen, but the notice-insert row shift and actorStart reuse are still unnamed and untested.
- PQ-9 — addressed
- PQ-10 — addressed

### Raised

- **PQ-11** [Critical] `donewhen-without-verification` The switcher surface has no input path: PointToActor is planned but no task names console_menu.go or panelkeys.go
  This is the 2nd finding in family `donewhen-without-verification`. Do NOT fix
  this instance alone — state the rule and sweep it: EVERY Done-when bullet maps
  to a named task with a named test, written into the plan as an explicit
  Done-when-to-Task map. PQ-7 was the first instance (nvim scroll had no step);
  this is the second, and writing that map is what would have caught it.
  The instance: three Done-when bullets cover the switcher, and M1 builds and
  tests PointToActor, but nothing delivers a click to it. Panel input flows
  processInput -> it.FeedHit -> route(before) -> onMenuInput -> DecodePanelKeys
  (console.go:606-627, console_menu.go:170). Task 7 makes the Interceptor CONSUME
  the report as HitMouse, so it is stripped from `before` and never reaches
  route, hence never reaches the panel. PanelKeyKind has no mouse member
  (panelkeys.go:14-25) and DecodePanelKeys drops unused sequences — its comment
  at :36-42 is about SGR mouse reports specifically. Neither panelkeys.go nor
  console_menu.go appears in any task or in Core concepts. Compounding it,
  RouteMouseReport(event, hostRows, childWantsMouse) has no focus input, so a
  panel-open click above the last row returns swallow/forward when it must
  return couch — the panel state is unrepresentable in the signature
  (ARCH-ORDER). The panel occupies size.Rows-1 (console_menu.go:186), so both
  surfaces are live at once and focus is genuinely part of the routing state.
- **PQ-12** [Important] `unspecified-event-policy` No complete disposition table: release contradicts the zero-bytes Done-when, and an unterminated SGR prefix has no bound
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

## Round 3 — 2026-09-05T12:40:30-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — "What already exists" table added; parser MOVED not rewritten; all four citations verified.
- PQ-4 — addressed — Button-0-only rule, wheel and release rows, plus a non-goals entry for the other buttons.
- PQ-8 — addressed — Extents derived from rootLine.actorStart and emitted from RenderMenuView after the notice insert.
- PQ-11 — addressed — Done-when-to-Task map written; the switcher input path is named and its "no change" justified.
- PQ-12 — not-addressed — Instances A/B/C fixed, but the table never crosses the classes with the panel-focused
state PQ-12 named, and RouteMouseReport's signature still cannot express it.

### Raised

- **PQ-13** [Critical] `unnamed-seam-change` A switcher click cannot both take Return's path and be an unconditional manual switch
  This is the 3rd finding in family `unnamed-seam-change` (PQ-2, PQ-9). Do not fix only
  this instance: the rule is that every task's Files list must name every production site
  its steps assert, and any "reuse the path X takes" claim must say what X decides where
  the new caller differs. Sweep all twelve tasks; Task 11 and Task 10 fail it today.
  The instance: KeyEnter dispatches "switch" (menu.go:445-449) into runMenuOperation,
  which captures pending attention (console.go:1434-1436, attention.go:75-79), and
  ExecuteConsoleOperation then sets how = arrivalNotification whenever that capture is
  nonzero (console.go:1625-1628). So a click on the notification line of a notifying
  actor, taking exactly Return's path, is classified arrivalNotification -- the one thing
  the Done-when forbids. Task 11 says to "mutate the call to arrivalNotification", but no
  caller passes an arrival on that path, and Task 11 lists only console_mouse_test.go.
  Say which side gives and name the production site that carries it.
- **PQ-14** [Minor] `guard-not-registered` New production file couchtty/mouse.go has no NonArtifactSources registration step
  This is the 2nd finding in family `guard-not-registered` (PQ-6 was conceptPlans). The
  rule, not the instance: every guard whose input is a hand-maintained list gets its
  registration step in the task that creates the artifact. NonArtifactSources
  (artifactpath/manifest.go:482-560) lists every couchtty/*.go file; Task 1 registers
  mouseinput but Task 6 creates couchtty/mouse.go with no entry. make test catches it.

## Round 4 — 2026-09-05T12:46:13-07:00 (claude) — passed

### Disposed

- PQ-12 — not-addressed — The three instances are fixed; the enumeration PQ-12 asked for was written on the child-mode axis only.
- PQ-13 — addressed — Resolution names the site (console.go:1434 capture skipped for a manual origin); Task 10/11 Files lists now carry it.
- PQ-14 — addressed — Task 6 registers couchtty/mouse.go in NonArtifactSources and states the rule for every future production file.

### Raised

- **PQ-15** [Minor] `unnamed-seam-change` Task 12 names neither the menuControls file nor the atlas file its steps change
  This is the 4th finding in family `unnamed-seam-change`. Do NOT fix this instance alone —
  the rule is already written in the plan at the end of the "Same path as Return" section
  ("each task's Files list names every production site its steps assert"); Task 12 is the
  one task that does not apply it. It has no Files block at all, and its Step 1 changes
  `menuControls` (`cmd/internal/couchtty/menu.go:19`), the README guard's scoped couch
  section, and an atlas file. Apply the already-stated rule to Task 12 rather than
  re-deriving it.

## Open findings

- **PQ-12** [Important] `unspecified-event-policy` No complete disposition table: release contradicts the zero-bytes Done-when, and an unterminated SGR prefix has no bound
- **PQ-15** [Minor] `unnamed-seam-change` Task 12 names neither the menuControls file nor the atlas file its steps change
