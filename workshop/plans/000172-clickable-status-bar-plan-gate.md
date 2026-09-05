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

## Open findings

- **PQ-1** [Critical] `reimplements-existing-mechanism` Plan builds a new mouse-mode scanner and SGR parser; both already exist and are unnamed
- **PQ-2** [Critical] `unnamed-seam-change` The swallow requirement forces changes to keys.go Interceptor, which no task lists
- **PQ-3** [Important] `unnamed-seam-change` HitMouse cannot dispatch through hitHandlers() — the table is map[InterceptorHit]func()
- **PQ-4** [Important] `unspecified-event-policy` No button filter and no press/release policy; wheel events and unmatched presses are live hazards
- **PQ-5** [Important] `undeclared-concurrent-writer` Replay-on-switch already re-asserts the child's mouse modes; the plan adds a second writer
- **PQ-6** [Important] `guard-not-registered` The plan's Core concepts table is not registered in conceptPlans, so all twelve rows go unpinned
- **PQ-7** [Important] `donewhen-without-verification` "nvim selection and scroll still work in an attached session" has no task and no manual step
- **PQ-8** [Minor] `reimplements-existing-mechanism` Actor extents already exist as rootLine.actorStart; and RenderMenuView shifts rows after the fact
- **PQ-9** [Minor] `undeclared-blast-radius` RenderStatusRow's signature change has a cross-package caller
- **PQ-10** [Minor] `missing-non-goals` No non-goals section, and no ARCH-SECURE statement for input the component did not produce
