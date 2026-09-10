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

## Open findings

- **BR-1** [Minor] `asserted-state-must-be-reconstructible` Tests row still lists cursor-save in the mode-assertion cross-product the design just dropped
- **BR-2** [Critical] `claimed-fix-unpinned-by-test` couch's repaint request and both consumers' ChildModes wiring are pinned by no test — deleting them leaves the suite green
- **BR-3** [Critical] `external-behavior-unverified` The zellij conformance probe measures a resize sequence with a 1.5s gap that production never issues
- **BR-4** [Critical] `absent-data-is-not-intent` removeTab lost its unconditional clear and has no nudge, so a closed tab's screen can persist
- **BR-5** [Important] `shared-mechanism-duplicated` The resize nudge is copy-pasted into couchtty and termcmd, contradicting the plan's "one answer, not two"
- **BR-6** [Important] `scanner-must-see-what-was-written` The composed alt-screen assertion is written to the host but never fed back to hostScan
- **BR-7** [Important] `nudge-ordering-and-extent` In termcmd the nudge's shrink/restore races resizeAll across two goroutines
- **BR-8** [Important] `absent-data-is-not-intent` couch's menu takeover is documented as a deliberate clear but wired as a replace
- **BR-9** [Important] `plan-row-ticked-not-delivered` Three ticked Plan rows are not delivered: the #204 counted invariant, the four regression conversions, and the differential row
- **BR-10** [Minor] `godoc-attached-to-wrong-symbol` requestRepaint was inserted between takeOverScreen's doc comment and its func
- **BR-11** [Minor] `escaping-artifact-in-source` A botched shell escape left literal '+chr(39)+' in a test failure message
- **BR-12** [Minor] `dead-exported-surface` Child.AltScreenObserved has zero callers, production or test
- **BR-13** [Minor] `operating-envelope-unstated` The nudge's cost is stated as "one extra repaint" but the probe measured 19,317 bytes on a single-pane session
