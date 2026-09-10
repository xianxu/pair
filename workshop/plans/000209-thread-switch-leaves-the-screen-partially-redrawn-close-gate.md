---
gate: boundary-review
issue: 209
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-09T21:16:33-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Tests row still lists cursor-save in the mode-assertion cross-product the design just dropped
          detail: |-
            This is the 2nd finding in family asserted-state-must-be-reconstructible, but
            the rule is already stated correctly in the plan (Mode-4 item 2: no sequence
            injects a previously-saved position, so HoldsCursorSave is a gate input, not
            replayable state). Only the Tests row lags it, still asking for
            {alt-screen, mouse, SGR-mouse, cursor-save} x {set, unset} — half of whose
            cells assert bytes the design will not emit. Drop cursor-save from that
            cross-product so the test surface matches the three fields that carry an
            observed bit.
            (carried from plan-quality PQ-12, deferred to the boundary review)
          family: asserted-state-must-be-reconstructible
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-09T21:16:33-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Critical
          title: couch's repaint request and both consumers' ChildModes wiring are pinned by no test — deleting them leaves the suite green
          detail: |-
            Measured in a scratch copy: removing c.requestRepaint(p.child) (console.go:504),
            or replacing the ChildModes at console.go:498 with the zero value, or dropping
            modes.AltScreen/AltScreenObserved at run.go:1333, each leaves the suite with an
            identical failure set (only the documented pty-sandbox class). Only termcmd's
            nudge is defended. couch is the operator's reported bug. Add a couchtty nudge
            test plus the differential byte-identity row the plan promised.
          family: claimed-fix-unpinned-by-test
          round: 2
        - id: BR-3
          severity: Critical
          title: The zellij conformance probe measures a resize sequence with a 1.5s gap that production never issues
          detail: |-
            probes/zellijrepaint/main.go:104-113 sleeps 1500ms between the shrink and the
            restore; production issues both TIOCSWINSZ ioctls back-to-back (console.go:1023-1025,
            run.go:1353-1356, via a bare pty.Setsize at child.go:206). Standard signals do not
            queue, so zellij may take one SIGWINCH, read an unchanged winsize, and re-render
            nothing — the fix would be a no-op while the fake (which only records resize calls)
            and the recorded probe are both green. Make the probe drive the production sequence
            and re-record the verdict.
          family: external-behavior-unverified
          round: 2
        - id: BR-4
          severity: Critical
          title: removeTab lost its unconditional clear and has no nudge, so a closed tab's screen can persist
          detail: |-
            applyTakeover used to write HomeAndClear unconditionally; it now routes through
            Repaint(..., RepaintReplace), which emits zero bytes on an empty replay.
            run.go:1433 passes clear=false and, unlike switchRelative, issues no repaint
            request. Alt+T, Alt+Left, Alt+W closes tab 1 while tab 2's ring is still empty:
            nothing is written and the destroyed tab's content stays on the pane. Pass
            clear=true — a takeover replacing a tab that no longer exists is deliberate.
          family: absent-data-is-not-intent
          round: 2
        - id: BR-5
          severity: Important
          title: The resize nudge is copy-pasted into couchtty and termcmd, contradicting the plan's "one answer, not two"
          detail: |-
            console.go:1012-1026 and run.go:1345-1357 are the same five lines with different
            receivers. Plan step 4 named a shared home so the fix would exist once; only the
            composition landed in hostty. Any settle delay or coalescing rule added later has
            to be written twice. Lift it to Child.RequestRepaint(size) on ptychild (no hostty
            constants needed, so no import cycle).
          family: shared-mechanism-duplicated
          round: 2
        - id: BR-6
          severity: Important
          title: The composed alt-screen assertion is written to the host but never fed back to hostScan
          detail: |-
            console.go:1036 writes Repaint(...) but console.go:1050 feeds only body; run.go:996
            writes Repaint(...) but run.go:1000 feeds only replay. The ?1049h prefix exists
            because the body lacks it, so this is the common case on every couch switch. With a
            ?1048h in the tail, hostScan then holds cursorSaved=true, altScreen=false and
            SafeToPaint() is false for the session — BR-79's frozen strip by the road BR-82's
            comment was written to close, and the hazard PQ-3 named. Feed the composed bytes.
          family: scanner-must-see-what-was-written
          round: 2
        - id: BR-7
          severity: Important
          title: In termcmd the nudge's shrink/restore races resizeAll across two goroutines
          detail: |-
            requestChildRepaint runs on the input goroutine (handleTerminalChord -> nextTab,
            run.go:575) while resizeAll runs on the writer goroutine (resizeThroughWriter ->
            inheritSize). A host resize landing between the two Resize calls is overwritten by
            the restore leg, leaving the child at the stale size with no event to correct it —
            the "permanently mis-sized" hazard PQ-4 named. couch is safe (both on the Run
            goroutine); termcmd is not. Route the nudge through m.enqueue({onWriter: ...}),
            which also gives an ordering test a seam.
          family: nudge-ordering-and-extent
          round: 2
        - id: BR-8
          severity: Important
          title: couch's menu takeover is documented as a deliberate clear but wired as a replace
          detail: |-
            console_menu.go:194-196 says "a deliberate clear"; takeOverScreen hardcodes
            RepaintReplace at console.go:1036. termcmd got two doors (redrawTab/clearTab);
            couch still has one. The class enumeration is four takeover call sites — switchTo,
            showMenu, newTab, removeTab — and only newTab was swept. Give takeOverScreen an
            intent parameter and pass RepaintClear from showMenu.
          family: absent-data-is-not-intent
          round: 2
        - id: BR-9
          severity: Important
          title: 'Three ticked Plan rows are not delivered: the #204 counted invariant, the four regression conversions, and the differential row'
          detail: |-
            workshop/issues/000204-*.md is untouched in this window and its invariant table
            still lists only #201/#202/#203; the phrase appears nowhere outside #209's own
            file. The four replay_insufficiency_test.go rows remain pure reproductions ending
            in t.Log with no assertion against the new path. No test compares couchtty's and
            termcmd's repaint output for byte identity. All three rows are marked [x].
          family: plan-row-ticked-not-delivered
          round: 2
        - id: BR-10
          severity: Minor
          title: requestRepaint was inserted between takeOverScreen's doc comment and its func
          detail: |-
            console.go:987-1010 now documents requestRepaint with takeOverScreen's paragraph,
            and takeOverScreen at console.go:1028 has no doc at all.
          family: godoc-attached-to-wrong-symbol
          round: 2
        - id: BR-11
          severity: Minor
          title: A botched shell escape left literal '+chr(39)+' in a test failure message
          detail: termcmd/run_test.go:1365 — "want the child'+chr(39)+'s own size %v".
          family: escaping-artifact-in-source
          round: 2
        - id: BR-12
          severity: Minor
          title: Child.AltScreenObserved has zero callers, production or test
          detail: |-
            ptychild/child.go:283-289. Both consumers use RepaintModes(). Newly-added dead
            exported surface — the #192 class.
          family: dead-exported-surface
          round: 2
        - id: BR-13
          severity: Minor
          title: The nudge's cost is stated as "one extra repaint" but the probe measured 19,317 bytes on a single-pane session
          detail: |-
            A rows-only resize reflows the whole zellij layout, and couch runs 10+ panes, on a
            keystroke path. PQ-7 was disposed on prose; the number belongs in #204's invariant
            table alongside the repaint-request count.
          family: operating-envelope-unstated
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-09T22:22:28-07:00"
      agent: claude
      blocked: true
      protocol_error: no valid findings block
    - "n": 4
      timestamp: "2026-09-09T22:57:58-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Tests row now names the two replacement tests; no cross-product remains.
          round: 4
        - id: BR-2
          disposition: addressed
          note: 'Verified by revert: dropping RequestRepaint from takeOverScreen reds 2 couchtty tests.'
          round: 4
        - id: BR-3
          disposition: addressed
          note: Probe drives the production sequence and reads ptychild.RepaintSettle; own target, not test-smoke.
          round: 4
        - id: BR-4
          disposition: addressed
          note: 'Verified by revert: RepaintClear -> RepaintReplace reds TestClosingATabBlanksTheDeadTabsScreen...'
          round: 4
        - id: BR-5
          disposition: addressed
          note: Child.RequestRepaint is the single home; both consumers call it, no copy remains.
          round: 4
        - id: BR-6
          disposition: addressed
          note: Both consoles feed the composed bytes; byte-inert today and honestly recorded as such.
          round: 4
        - id: BR-7
          disposition: addressed
          note: 'Verified by revert: unlocking geom across the settle reds the goroutine-independence test.'
          round: 4
        - id: BR-8
          disposition: addressed
          note: Intent parameter landed and showMenu passes RepaintClear; byte-inert exception recorded, not papered over.
          round: 4
        - id: BR-9
          disposition: addressed
          note: 204's invariant row and envelope table landed; the four reproductions carry answer tests; differential closes transitively.
          round: 4
        - id: BR-10
          disposition: addressed
          note: Doc restored AND a mechanical guard added; I confirmed the guard fires on a synthetic recurrence.
          round: 4
        - id: BR-11
          disposition: addressed
          note: No chr(39) artifact anywhere in cmd/.
          round: 4
        - id: BR-12
          disposition: addressed
          note: Child.AltScreenObserved is gone; RepaintModes is the only door. New instances raised separately.
          round: 4
        - id: BR-13
          disposition: addressed
          note: 6.6 KB back-to-back / 19.3 KB with a gap now sit in 204's table beside the counted invariant.
          round: 4
      findings:
        - id: BR-14
          severity: Critical
          title: An empty replay leaves the OUTGOING surface on screen, and no takeover site wants that
          detail: |-
            3rd in this family, so the deliverable is the rule, not the row. The rule:
            what an empty body means is a property of whose frame is already on the
            screen, not of the call site's name. "A stale frame beats a blank one"
            (repaint.go:26-29) holds only for a same-child repaint, and the enumeration
            of takeover sites contains none - switchTo, switchRelative, showMenu,
            newTab and removeTab all replace a DIFFERENT surface. Reachable on the
            primary flow: starting a thread from the panel attaches (console.go:366
            seeds replayCutoff = ReplaySafeEnd) and dispatches switch on the same
            operationQueue goroutine before Run drains a batch, so ReplayThrough
            returns nothing, Repaint emits zero bytes, and the PANEL'S OWN MENU BODY
            stays on the terminal under the new thread's label until zellij's first
            frame. Before this diff it was HomeAndClear plus nothing - blank, which is
            honest. This is BR-4's shape at the sites BR-4 did not name. Fix: make the
            enum name the question the site must answer, and record that the
            keep-stale branch currently has zero correct callers.
          family: absent-data-is-not-intent
          round: 4
        - id: BR-15
          severity: Important
          title: A mandatory Resize waits a full settle on the nudge's lock, while 204 states the cost as zero
          detail: |-
            2nd in this family, so state the rule. nudge() holds geom across
            time.Sleep(RepaintSettle) (child.go:277-290) and Resize takes the same
            lock; measured in a scratch copy, a concurrent Resize blocked 20.8 ms.
            termcmd's resizeAll (run.go:1600) runs on the WRITER goroutine via
            resizeThroughWriter, so a SIGWINCH landing inside a switch stalls all pane
            output for a settle. 204's table says "event-loop time per nudge | 0",
            which is true only for the caller that requests the nudge. The rule: an
            envelope must be stated for every path the mechanism can block, not only
            the path that invokes it - the contention sibling of the rule 204 already
            writes for cost. Enumerate the geom contenders (Console.applyLayout,
            terminalMux.resizeAll, teardown) and give each a row, or let Resize
            preempt an in-flight nudge via a generation the restore leg checks.
          family: operating-envelope-unstated
          round: 4
        - id: BR-16
          severity: Important
          title: takeOverScreen still claims Run-goroutine-only, and this round's own test drives it from the operationQueue
          detail: |-
            2nd in this family, so state the rule: a goroutine-ownership claim in a
            comment is not a mechanism - which is exactly what C2 concluded for the
            nudge and did not apply to the writer. console.go:996 asserts "still
            Run-goroutine-only, like every other writer", but switchTo is reached from
            ExecuteConsoleOperation (console.go:1862) and
            TestASwitchThroughTheOperationQueueNudgesLikeAnyOther exercises that path.
            takeOverScreen writes c.host directly; hostty.Host is a bare io.Writer and
            couch has five unsynchronized write sites (console.go:915,983,1009,1058;
            console_menu.go:193,199,202). termcmd already solved this next door -
            paneWriter is deliberately not an io.Writer so a door that skips the
            reasoning does not compile (run.go:993). Either give couch the same typed
            single-writer door or delete the false sentence and file the divergence;
            leaving it is what lets C2's rule read as satisfied. Same lens, smaller:
            go c.nudge() has no tie to c.done, so the goroutine outlives Close by a
            settle with no cancellation path.
          family: nudge-ordering-and-extent
          round: 4
        - id: BR-17
          severity: Important
          title: NewFakeChild starts with no geometry while Start records opts.Size, and geometry is what the nudge reads
          detail: |-
            Start sets size: opts.Size (child.go:129-135) so a real child always
            nudges; NewFakeChild (fake.go:37-46) sets nothing, so a fake silently
            declines via size.Rows < 2 until a test resizes it. run_test.go:1341
            rationalises the workaround as "what production does", but production
            cannot reach the zero-geometry state the fake starts in - a consumer that
            forgot to size a child would be invisible in tests and would nudge in
            production. TestFakeAndRealChildAgreeAfterTheChildHasEnded drives
            Write/Resize/Signal and does not cover Size or RequestRepaint. ARCH-MOCK:
            give NewFakeChild a starting size mirroring Start, and add both to the
            conformance stimulus set.
          family: fake-diverges-from-real
          round: 4
        - id: BR-18
          severity: Minor
          title: Three newly-exported identifiers have no consumer outside their own package's tests
          detail: |-
            2nd in this family, so fix the rule rather than the three. hostty.EnterAltScreen
            (control.go:68) is referenced only by repaint_test.go's forbidden list IN
            THE SAME PACKAGE, so it never needed exporting; Child.Size() (child.go:346)
            has zero production callers and two test ones; hostty.Repaint is reachable
            only through RepaintFor and its own package's tests. Measured prevalence:
            three instances this round plus BR-12 last round. The rule - exported
            surface needs a consumer outside its own package's tests - is AST-checkable,
            and this repo already writes that kind of guard (doccomment_test.go).
          family: dead-exported-surface
          round: 4
        - id: BR-19
          severity: Minor
          title: The mutation recipe in console_test.go names a signature and a call site that do not exist at HEAD
          detail: |-
            console_test.go:1056 says the test reds on "dropping
            p.child.RequestRepaint(c.ChildSize()) from switchTo". RequestRepaint takes
            no argument since C2 and the call moved into takeOverScreen. A reader
            following the recipe finds nothing to delete and may conclude the test is
            unpinned - the precise failure mode this issue's ledger exists to prevent.
          family: comment-cites-code-that-moved
          round: 4
        - id: BR-20
          severity: Minor
          title: The goroutine-independence race test sleeps to let the nudge take the lock, so it can fail spuriously
          detail: |-
            replay_insufficiency_test.go:196 sleeps RepaintSettle/4 after
            RequestRepaint, assuming the spawned nudge has taken geom. Under load or
            -race it may not have; the racing Resize then lands first and the
            assertion got[1].Rows != before.Rows-1 fails on a correct tree. Wait on
            waitForResizes(t, child, 2) instead - the same lesson the test's own
            comment teaches about the other party.
          family: test-sleeps-instead-of-synchronizing
          round: 4
      blocked: true
    - "n": 5
      timestamp: "2026-09-09T23:25:17-07:00"
      agent: claude
      dispose:
        - id: BR-14
          disposition: addressed
          note: Branch and RepaintIntent both deleted; repaint always blanks, pinned by TestATakeoverAlwaysBlanksEvenWithNothingToDraw over nil/empty/non-empty.
          round: 5
        - id: BR-15
          disposition: addressed
          note: Verified by revert — holding geom across the settle reds TestAResizeDuringANudgeWinsWhicheverGoroutineItComesFrom with a 4th resize.
          round: 5
        - id: BR-16
          disposition: addressed
          note: False sentence deleted, divergence filed as pair#224, settle cancellable on c.done, Close takes geom.
          round: 5
        - id: BR-17
          disposition: not-addressed
          note: 'Measured — deleting `size: FakeChildSize` leaves all four packages at the baseline failure set; Size/RequestRepaint still absent from the conformance stimuli.'
          round: 5
        - id: BR-18
          disposition: not-addressed
          note: Instances unexported individually, no AST guard written, and the class grew — ChildModes, Screen.AltScreenObserved, FakeChildSize.
          round: 5
        - id: BR-19
          disposition: addressed
          note: The named recipe now cites child.RequestRepaint() in takeOverScreen, which exists; the class recurred elsewhere — see the new finding.
          round: 5
        - id: BR-20
          disposition: addressed
          note: waitForResizes(t, child, 2) replaces the sleep, exactly as recommended.
          round: 5
      findings:
        - id: BR-21
          severity: Important
          title: The C-1 withdrawal reached the code and the named Plan rows, but not the seven other places the withdrawn rule is written
          detail: |-
            2nd in this family, so the deliverable is the rule: deleting an exported
            identifier obliges a sweep of every PROSE reference to it, not only the
            code that stopped compiling. Measured prevalence 7 at HEAD, all created by
            the commit that deleted RepaintIntent — atlas/architecture.md:531 (wrong
            RepaintFor signature) and :537-540 (teaches "a stale frame beats a blank
            one" as current, in the always-current map); termcmd/run.go:1693-1696
            (contradicts clearTab's own doc 20 lines below); termcmd/run_test.go:1431
            and couchtty/console_test.go:1149 (MUTATION RECIPES naming RepaintReplace /
            RepaintClear / TestRepaintCarriesIntent..., so a reviewer verifying a pin
            finds nothing to flip — BR-19's exact failure mode one file over);
            ptychild/replay_insufficiency_test.go:131,137; ptychild/child.go:484. The
            set is fully mechanical (grep for the four deleted names over non-history
            files), and doccomment_test.go is the precedent for turning it into a guard.
          family: comment-cites-code-that-moved
          round: 5
        - id: BR-22
          severity: Important
          title: Done-when clause 2 states a criterion the shipped code deliberately does not meet, with no Revisions entry
          detail: |-
            2nd in this family, so the rule: a reversed commitment must be revised
            everywhere it was written — Plan rows AND Done-when AND the atlas — not
            only in the rows a review named. "The cutoff < ringStart path cannot
            present a cleared screen with nothing drawn" was met by the middle version
            and is now explicitly declined: repaint always blanks, so an empty replay
            IS a cleared screen with nothing drawn until the child's frame lands, and
            permanently for a bare-shell pair term tab. The third-pass Revisions Delta
            lists three Plan rows and never touches Done-when, which is the section the
            merge-time specs judge reads.
          family: plan-row-ticked-not-delivered
          round: 5
      blocked: false
---

# Gate ledger — pair#209 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-09T21:16:33-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `asserted-state-must-be-reconstructible` Tests row still lists cursor-save in the mode-assertion cross-product the design just dropped
  This is the 2nd finding in family asserted-state-must-be-reconstructible, but
  the rule is already stated correctly in the plan (Mode-4 item 2: no sequence
  injects a previously-saved position, so HoldsCursorSave is a gate input, not
  replayable state). Only the Tests row lags it, still asking for
  {alt-screen, mouse, SGR-mouse, cursor-save} x {set, unset} — half of whose
  cells assert bytes the design will not emit. Drop cursor-save from that
  cross-product so the test surface matches the three fields that carry an
  observed bit.
  (carried from plan-quality PQ-12, deferred to the boundary review)

## Round 2 — 2026-09-09T21:16:33-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Critical] `claimed-fix-unpinned-by-test` couch's repaint request and both consumers' ChildModes wiring are pinned by no test — deleting them leaves the suite green
  Measured in a scratch copy: removing c.requestRepaint(p.child) (console.go:504),
  or replacing the ChildModes at console.go:498 with the zero value, or dropping
  modes.AltScreen/AltScreenObserved at run.go:1333, each leaves the suite with an
  identical failure set (only the documented pty-sandbox class). Only termcmd's
  nudge is defended. couch is the operator's reported bug. Add a couchtty nudge
  test plus the differential byte-identity row the plan promised.
- **BR-3** [Critical] `external-behavior-unverified` The zellij conformance probe measures a resize sequence with a 1.5s gap that production never issues
  probes/zellijrepaint/main.go:104-113 sleeps 1500ms between the shrink and the
  restore; production issues both TIOCSWINSZ ioctls back-to-back (console.go:1023-1025,
  run.go:1353-1356, via a bare pty.Setsize at child.go:206). Standard signals do not
  queue, so zellij may take one SIGWINCH, read an unchanged winsize, and re-render
  nothing — the fix would be a no-op while the fake (which only records resize calls)
  and the recorded probe are both green. Make the probe drive the production sequence
  and re-record the verdict.
- **BR-4** [Critical] `absent-data-is-not-intent` removeTab lost its unconditional clear and has no nudge, so a closed tab's screen can persist
  applyTakeover used to write HomeAndClear unconditionally; it now routes through
  Repaint(..., RepaintReplace), which emits zero bytes on an empty replay.
  run.go:1433 passes clear=false and, unlike switchRelative, issues no repaint
  request. Alt+T, Alt+Left, Alt+W closes tab 1 while tab 2's ring is still empty:
  nothing is written and the destroyed tab's content stays on the pane. Pass
  clear=true — a takeover replacing a tab that no longer exists is deliberate.
- **BR-5** [Important] `shared-mechanism-duplicated` The resize nudge is copy-pasted into couchtty and termcmd, contradicting the plan's "one answer, not two"
  console.go:1012-1026 and run.go:1345-1357 are the same five lines with different
  receivers. Plan step 4 named a shared home so the fix would exist once; only the
  composition landed in hostty. Any settle delay or coalescing rule added later has
  to be written twice. Lift it to Child.RequestRepaint(size) on ptychild (no hostty
  constants needed, so no import cycle).
- **BR-6** [Important] `scanner-must-see-what-was-written` The composed alt-screen assertion is written to the host but never fed back to hostScan
  console.go:1036 writes Repaint(...) but console.go:1050 feeds only body; run.go:996
  writes Repaint(...) but run.go:1000 feeds only replay. The ?1049h prefix exists
  because the body lacks it, so this is the common case on every couch switch. With a
  ?1048h in the tail, hostScan then holds cursorSaved=true, altScreen=false and
  SafeToPaint() is false for the session — BR-79's frozen strip by the road BR-82's
  comment was written to close, and the hazard PQ-3 named. Feed the composed bytes.
- **BR-7** [Important] `nudge-ordering-and-extent` In termcmd the nudge's shrink/restore races resizeAll across two goroutines
  requestChildRepaint runs on the input goroutine (handleTerminalChord -> nextTab,
  run.go:575) while resizeAll runs on the writer goroutine (resizeThroughWriter ->
  inheritSize). A host resize landing between the two Resize calls is overwritten by
  the restore leg, leaving the child at the stale size with no event to correct it —
  the "permanently mis-sized" hazard PQ-4 named. couch is safe (both on the Run
  goroutine); termcmd is not. Route the nudge through m.enqueue({onWriter: ...}),
  which also gives an ordering test a seam.
- **BR-8** [Important] `absent-data-is-not-intent` couch's menu takeover is documented as a deliberate clear but wired as a replace
  console_menu.go:194-196 says "a deliberate clear"; takeOverScreen hardcodes
  RepaintReplace at console.go:1036. termcmd got two doors (redrawTab/clearTab);
  couch still has one. The class enumeration is four takeover call sites — switchTo,
  showMenu, newTab, removeTab — and only newTab was swept. Give takeOverScreen an
  intent parameter and pass RepaintClear from showMenu.
- **BR-9** [Important] `plan-row-ticked-not-delivered` Three ticked Plan rows are not delivered: the #204 counted invariant, the four regression conversions, and the differential row
  workshop/issues/000204-*.md is untouched in this window and its invariant table
  still lists only #201/#202/#203; the phrase appears nowhere outside #209's own
  file. The four replay_insufficiency_test.go rows remain pure reproductions ending
  in t.Log with no assertion against the new path. No test compares couchtty's and
  termcmd's repaint output for byte identity. All three rows are marked [x].
- **BR-10** [Minor] `godoc-attached-to-wrong-symbol` requestRepaint was inserted between takeOverScreen's doc comment and its func
  console.go:987-1010 now documents requestRepaint with takeOverScreen's paragraph,
  and takeOverScreen at console.go:1028 has no doc at all.
- **BR-11** [Minor] `escaping-artifact-in-source` A botched shell escape left literal '+chr(39)+' in a test failure message
  termcmd/run_test.go:1365 — "want the child'+chr(39)+'s own size %v".
- **BR-12** [Minor] `dead-exported-surface` Child.AltScreenObserved has zero callers, production or test
  ptychild/child.go:283-289. Both consumers use RepaintModes(). Newly-added dead
  exported surface — the #192 class.
- **BR-13** [Minor] `operating-envelope-unstated` The nudge's cost is stated as "one extra repaint" but the probe measured 19,317 bytes on a single-pane session
  A rows-only resize reflows the whole zellij layout, and couch runs 10+ panes, on a
  keystroke path. PQ-7 was disposed on prose; the number belongs in #204's invariant
  table alongside the repaint-request count.

## Round 3 — 2026-09-09T22:22:28-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 4 — 2026-09-09T22:57:58-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Tests row now names the two replacement tests; no cross-product remains.
- BR-2 — addressed — Verified by revert: dropping RequestRepaint from takeOverScreen reds 2 couchtty tests.
- BR-3 — addressed — Probe drives the production sequence and reads ptychild.RepaintSettle; own target, not test-smoke.
- BR-4 — addressed — Verified by revert: RepaintClear -> RepaintReplace reds TestClosingATabBlanksTheDeadTabsScreen...
- BR-5 — addressed — Child.RequestRepaint is the single home; both consumers call it, no copy remains.
- BR-6 — addressed — Both consoles feed the composed bytes; byte-inert today and honestly recorded as such.
- BR-7 — addressed — Verified by revert: unlocking geom across the settle reds the goroutine-independence test.
- BR-8 — addressed — Intent parameter landed and showMenu passes RepaintClear; byte-inert exception recorded, not papered over.
- BR-9 — addressed — 204's invariant row and envelope table landed; the four reproductions carry answer tests; differential closes transitively.
- BR-10 — addressed — Doc restored AND a mechanical guard added; I confirmed the guard fires on a synthetic recurrence.
- BR-11 — addressed — No chr(39) artifact anywhere in cmd/.
- BR-12 — addressed — Child.AltScreenObserved is gone; RepaintModes is the only door. New instances raised separately.
- BR-13 — addressed — 6.6 KB back-to-back / 19.3 KB with a gap now sit in 204's table beside the counted invariant.

### Raised

- **BR-14** [Critical] `absent-data-is-not-intent` An empty replay leaves the OUTGOING surface on screen, and no takeover site wants that
  3rd in this family, so the deliverable is the rule, not the row. The rule:
  what an empty body means is a property of whose frame is already on the
  screen, not of the call site's name. "A stale frame beats a blank one"
  (repaint.go:26-29) holds only for a same-child repaint, and the enumeration
  of takeover sites contains none - switchTo, switchRelative, showMenu,
  newTab and removeTab all replace a DIFFERENT surface. Reachable on the
  primary flow: starting a thread from the panel attaches (console.go:366
  seeds replayCutoff = ReplaySafeEnd) and dispatches switch on the same
  operationQueue goroutine before Run drains a batch, so ReplayThrough
  returns nothing, Repaint emits zero bytes, and the PANEL'S OWN MENU BODY
  stays on the terminal under the new thread's label until zellij's first
  frame. Before this diff it was HomeAndClear plus nothing - blank, which is
  honest. This is BR-4's shape at the sites BR-4 did not name. Fix: make the
  enum name the question the site must answer, and record that the
  keep-stale branch currently has zero correct callers.
- **BR-15** [Important] `operating-envelope-unstated` A mandatory Resize waits a full settle on the nudge's lock, while 204 states the cost as zero
  2nd in this family, so state the rule. nudge() holds geom across
  time.Sleep(RepaintSettle) (child.go:277-290) and Resize takes the same
  lock; measured in a scratch copy, a concurrent Resize blocked 20.8 ms.
  termcmd's resizeAll (run.go:1600) runs on the WRITER goroutine via
  resizeThroughWriter, so a SIGWINCH landing inside a switch stalls all pane
  output for a settle. 204's table says "event-loop time per nudge | 0",
  which is true only for the caller that requests the nudge. The rule: an
  envelope must be stated for every path the mechanism can block, not only
  the path that invokes it - the contention sibling of the rule 204 already
  writes for cost. Enumerate the geom contenders (Console.applyLayout,
  terminalMux.resizeAll, teardown) and give each a row, or let Resize
  preempt an in-flight nudge via a generation the restore leg checks.
- **BR-16** [Important] `nudge-ordering-and-extent` takeOverScreen still claims Run-goroutine-only, and this round's own test drives it from the operationQueue
  2nd in this family, so state the rule: a goroutine-ownership claim in a
  comment is not a mechanism - which is exactly what C2 concluded for the
  nudge and did not apply to the writer. console.go:996 asserts "still
  Run-goroutine-only, like every other writer", but switchTo is reached from
  ExecuteConsoleOperation (console.go:1862) and
  TestASwitchThroughTheOperationQueueNudgesLikeAnyOther exercises that path.
  takeOverScreen writes c.host directly; hostty.Host is a bare io.Writer and
  couch has five unsynchronized write sites (console.go:915,983,1009,1058;
  console_menu.go:193,199,202). termcmd already solved this next door -
  paneWriter is deliberately not an io.Writer so a door that skips the
  reasoning does not compile (run.go:993). Either give couch the same typed
  single-writer door or delete the false sentence and file the divergence;
  leaving it is what lets C2's rule read as satisfied. Same lens, smaller:
  go c.nudge() has no tie to c.done, so the goroutine outlives Close by a
  settle with no cancellation path.
- **BR-17** [Important] `fake-diverges-from-real` NewFakeChild starts with no geometry while Start records opts.Size, and geometry is what the nudge reads
  Start sets size: opts.Size (child.go:129-135) so a real child always
  nudges; NewFakeChild (fake.go:37-46) sets nothing, so a fake silently
  declines via size.Rows < 2 until a test resizes it. run_test.go:1341
  rationalises the workaround as "what production does", but production
  cannot reach the zero-geometry state the fake starts in - a consumer that
  forgot to size a child would be invisible in tests and would nudge in
  production. TestFakeAndRealChildAgreeAfterTheChildHasEnded drives
  Write/Resize/Signal and does not cover Size or RequestRepaint. ARCH-MOCK:
  give NewFakeChild a starting size mirroring Start, and add both to the
  conformance stimulus set.
- **BR-18** [Minor] `dead-exported-surface` Three newly-exported identifiers have no consumer outside their own package's tests
  2nd in this family, so fix the rule rather than the three. hostty.EnterAltScreen
  (control.go:68) is referenced only by repaint_test.go's forbidden list IN
  THE SAME PACKAGE, so it never needed exporting; Child.Size() (child.go:346)
  has zero production callers and two test ones; hostty.Repaint is reachable
  only through RepaintFor and its own package's tests. Measured prevalence:
  three instances this round plus BR-12 last round. The rule - exported
  surface needs a consumer outside its own package's tests - is AST-checkable,
  and this repo already writes that kind of guard (doccomment_test.go).
- **BR-19** [Minor] `comment-cites-code-that-moved` The mutation recipe in console_test.go names a signature and a call site that do not exist at HEAD
  console_test.go:1056 says the test reds on "dropping
  p.child.RequestRepaint(c.ChildSize()) from switchTo". RequestRepaint takes
  no argument since C2 and the call moved into takeOverScreen. A reader
  following the recipe finds nothing to delete and may conclude the test is
  unpinned - the precise failure mode this issue's ledger exists to prevent.
- **BR-20** [Minor] `test-sleeps-instead-of-synchronizing` The goroutine-independence race test sleeps to let the nudge take the lock, so it can fail spuriously
  replay_insufficiency_test.go:196 sleeps RepaintSettle/4 after
  RequestRepaint, assuming the spawned nudge has taken geom. Under load or
  -race it may not have; the racing Resize then lands first and the
  assertion got[1].Rows != before.Rows-1 fails on a correct tree. Wait on
  waitForResizes(t, child, 2) instead - the same lesson the test's own
  comment teaches about the other party.

## Round 5 — 2026-09-09T23:25:17-07:00 (claude) — passed

### Disposed

- BR-14 — addressed — Branch and RepaintIntent both deleted; repaint always blanks, pinned by TestATakeoverAlwaysBlanksEvenWithNothingToDraw over nil/empty/non-empty.
- BR-15 — addressed — Verified by revert — holding geom across the settle reds TestAResizeDuringANudgeWinsWhicheverGoroutineItComesFrom with a 4th resize.
- BR-16 — addressed — False sentence deleted, divergence filed as pair#224, settle cancellable on c.done, Close takes geom.
- BR-17 — not-addressed — Measured — deleting `size: FakeChildSize` leaves all four packages at the baseline failure set; Size/RequestRepaint still absent from the conformance stimuli.
- BR-18 — not-addressed — Instances unexported individually, no AST guard written, and the class grew — ChildModes, Screen.AltScreenObserved, FakeChildSize.
- BR-19 — addressed — The named recipe now cites child.RequestRepaint() in takeOverScreen, which exists; the class recurred elsewhere — see the new finding.
- BR-20 — addressed — waitForResizes(t, child, 2) replaces the sleep, exactly as recommended.

### Raised

- **BR-21** [Important] `comment-cites-code-that-moved` The C-1 withdrawal reached the code and the named Plan rows, but not the seven other places the withdrawn rule is written
  2nd in this family, so the deliverable is the rule: deleting an exported
  identifier obliges a sweep of every PROSE reference to it, not only the
  code that stopped compiling. Measured prevalence 7 at HEAD, all created by
  the commit that deleted RepaintIntent — atlas/architecture.md:531 (wrong
  RepaintFor signature) and :537-540 (teaches "a stale frame beats a blank
  one" as current, in the always-current map); termcmd/run.go:1693-1696
  (contradicts clearTab's own doc 20 lines below); termcmd/run_test.go:1431
  and couchtty/console_test.go:1149 (MUTATION RECIPES naming RepaintReplace /
  RepaintClear / TestRepaintCarriesIntent..., so a reviewer verifying a pin
  finds nothing to flip — BR-19's exact failure mode one file over);
  ptychild/replay_insufficiency_test.go:131,137; ptychild/child.go:484. The
  set is fully mechanical (grep for the four deleted names over non-history
  files), and doccomment_test.go is the precedent for turning it into a guard.
- **BR-22** [Important] `plan-row-ticked-not-delivered` Done-when clause 2 states a criterion the shipped code deliberately does not meet, with no Revisions entry
  2nd in this family, so the rule: a reversed commitment must be revised
  everywhere it was written — Plan rows AND Done-when AND the atlas — not
  only in the rows a review named. "The cutoff < ringStart path cannot
  present a cleared screen with nothing drawn" was met by the middle version
  and is now explicitly declined: repaint always blanks, so an empty replay
  IS a cleared screen with nothing drawn until the child's frame lands, and
  permanently for a bare-shell pair term tab. The third-pass Revisions Delta
  lists three Plan rows and never touches Done-when, which is the section the
  merge-time specs judge reads.

## Open findings

- **BR-17** [Important] `fake-diverges-from-real` NewFakeChild starts with no geometry while Start records opts.Size, and geometry is what the nudge reads
- **BR-18** [Minor] `dead-exported-surface` Three newly-exported identifiers have no consumer outside their own package's tests
- **BR-21** [Important] `comment-cites-code-that-moved` The C-1 withdrawal reached the code and the named Plan rows, but not the seven other places the withdrawn rule is written
- **BR-22** [Important] `plan-row-ticked-not-delivered` Done-when clause 2 states a criterion the shipped code deliberately does not meet, with no Revisions entry
