# Boundary Review — pair#501 (whole-issue close)

| field | value |
|-------|-------|
| issue | 501 — broadcast-web: a button beside the notes popup in Couch's tab bar brings the cast Chrome to the front |
| repo | pair |
| issue file | workshop/issues/000501-broadcast-web-a-button-beside-the-notes-popup-in-couch-s-tab-bar-brings-the-cast-chrome-to-the-front.md |
| boundary | whole-issue close |
| milestone | — |
| window | 605445b697f4e8845ba50ff7ff434c31b1f967f4..0331d3da8df4bcbaa46d322de2c0473be5de8f8d |
| command | sdlc close --issue 501 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T19:00:57-07:00 |
| verdict | SHIP |

## Review

Running the tests: the webcast package passes. couchtty and artifactpath fail to build because this checkout is missing generated runtime assets, so I'm looking for the target that stages them.

I couldn't find the generator, so I'll add a placeholder file to satisfy the embed in this throwaway checkout and run the couchtty tests.

Targeted couchtty tests pass. Next I'll disable the ↗ click routing to check that its test actually catches the regression.

```verdict
verdict: SHIP
confidence: high
```

The change does what the issue asks for. When a web cast is running, Couch's tab bar now shows `Chrome ⏸ ↗ 👆 👽 💭N`. Clicking `↗` calls the new `Browser.Focus`. On the real Chrome, that sends `Target.activateTarget` for the cast tab and then activates the Chrome process by its PID. Clicking 👆 or 👽 copies the view or drive link. The live cell is now drawn in `broadcast.LiveSGR`. A test confirms the screen broadcast's `IndicatorShown` still doesn't mistake `Chrome ⏸` for LIVE. README and atlas cover the new surface, and the operator's smoke test passed. Nothing blocks shipping; the findings are all Minor.

The review used the pinned stat, name-status and full diff. On test runs:
- **`webcast`:** tests pass.
- **`couchtty`:** the targeted Webcast/WebNotes/Render tests pass. This checkout has no generated runtime-bundle assets, so I added placeholder files for that embed to compile. I left those in the throwaway checkout.
- **`artifactpath`:** fails for the same missing-asset reason, which has nothing to do with this change.
- **Mutation check:** with the `↗` click routing disabled, `TestWebcastFocusButton` never passes. It was still waiting after 300 s, against 1.9 s for the whole targeted run unmodified. I restored the file afterwards.

1. **Strengths**
   - The OS activator is only set by `Launch` for a windowed Chrome (`chrome.go:198`). Headless and scripted test Chromes therefore can't take the operator's focus, and `TestChromeFocusActivatesTheCastTab` checks this (`activate == nil`).
   - Activating by PID through JXA `NSRunningApplication` avoids raising the operator's own Chrome, which has the same bundle id. The command is an argv array, the PID is an `%d` int, and it has a timeout.
   - The conformance suite now covers `Focus`, including `ErrEnded` after the cast ends. So the fake and the real Chrome are held to the same contract.
   - Placing the three buttons before `💭N` keeps their columns fixed as the note count changes. Each button has its own span, ready for the hover tooltips #495 needs.
   - The test pins `IndicatorShown` not reading the restyled live cell as LIVE, which was the real risk of Revision 1.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `reserve.go:280`: the `button` closure is nearly a copy of the `capability` closure at `reserve.go:248`. Both could share one `span(label, sgr)` helper (ARCH-DRY).
   - `console_webcast.go:255`: if the cast ends between the phase check and `cast.Focus()`, the user sees a raw notice like "Chrome cast: …ended". It's a harmless race but could be ignored silently.
   - The `osascript` activator is a function field (nil in tests), not a seam with a stateful fake and a conformance check. It's acceptable for a single fire-and-forget OS call that the smoke test exercised (ARCH-MOCK).
   - The Chrome group now always takes about 7 more columns, which leaves less room for actor chips at narrow widths. #491's chip fitting handles this correctly.

5. **Test coverage**
   - Covered: the rendered row (spans, widths and adjacency in both phases, nothing drawn without Chrome, live style, the indicator check), console clicks reaching the fake's `Focuses` count and the link copies, and the Chrome loop sending `activateTarget` for the active tab.
   - The 👽 test only takes the fake's "can't be driven" branch. Copying an actual drive link is covered only by the earlier Alt+right-click path.

6. **Architecture**
   - ARCH-DRY: flag (Minor, above).
   - ARCH-PURE: pass. Rendering is pure, and `Focus` is thin glue.
   - ARCH-PURPOSE: pass. Every Done-when item is delivered. Dropping "reachable from the cell's menu" is justified in the Revisions section: the cell has no menu.
   - ARCH-MOCK: Minor (above).
   - ARCH-CONSTRAINTS: pass. One `osascript` per click, bounded by `callTimeout`, off the UI path.
   - ARCH-SECURE: pass. No untrusted input, and link text never reaches a notice.
   - ARCH-ORDER: pass. The phase is read under the lock, and `Focus` after the cast ends returns `ErrEnded` in both the fake and the real Chrome.
   - ARCH-FUNERAL: pass. Nothing durable is created, and the child process is bounded by its context.

7. **Plan revisions**
   - Revision 3 says the glyphs are the screen broadcast's `PointerLabel`/`ControlLabel`. Since #503 removed `ControlLabel`, 👽 is `couchtty.WebcastDriveLinkLabel`. The Log says so, but the Revisions entry is now stale and should be amended.

```findings
findings:
  - id: new
    severity: Minor
    family: duplicated-span-helper
    title: |
      reserve.go button closure duplicates the capability closure
    detail: |
      reserve.go:280 button() repeats capability() at reserve.go:248 (space, start, append, empty-span check); hoist one span(label, sgr) helper (ARCH-DRY).
  - id: new
    severity: Minor
    family: race-surfaces-raw-error
    title: |
      focusWebcast shows a raw ErrEnded notice if the cast ends mid-click
    detail: |
      console_webcast.go:255 checks the phase, then calls cast.Focus() in a goroutine; an end in between shows "Chrome cast: ...ended". Could ignore errors.Is(err, webcast.ErrEnded).
  - id: new
    severity: Minor
    family: plan-revision-stale
    title: |
      Revision 3 still names broadcast.ControlLabel for the Chrome group's alien glyph
    detail: |
      #503 removed ControlLabel; the code uses couchtty.WebcastDriveLinkLabel. Amend the Revisions entry so it matches the code.
```

The mutation check has now finished and confirms what the review assumed. With the `↗` click routing disabled, `TestWebcastFocusButton` fails with "timed out waiting for browser focused" (`console_webcast_test.go:355`) after its 590 s wait. So that test does catch the regression.

The checkout's tracked files are back to the pinned head. The placeholder `manifest.json` and `files/` directory I added for the embed may still be there, but `git status` doesn't list them.

The verdict is unchanged: SHIP, high confidence, with the three Minor findings as reported.
