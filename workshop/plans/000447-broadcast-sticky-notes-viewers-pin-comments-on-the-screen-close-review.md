# Boundary Review — pair#447 (whole-issue close)

| field | value |
|-------|-------|
| issue | 447 — Broadcast sticky notes: viewers pin comments on the screen |
| repo | pair |
| issue file | workshop/issues/000447-broadcast-sticky-notes-viewers-pin-comments-on-the-screen.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8fc225ae26956bcb4a58df2897f76b04fa84ad72..cd5911909669681c42d2620b732b1e313ff4d9ae |
| command | sdlc close --issue 447 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T19:34:23-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking the remaining open findings (BR-13, BR-16, BR-18, BR-19) against the code.

Running the couchtty tests needs the generated runtime assets. I'll add a placeholder in this disposable checkout so I can verify BR-27 and the right-click test.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The work at head does what the issue asks: notes are parsed strictly and held under the pointer lock, the page variants restore their join state at an end, the chip, glow and shared-beat flash work, and the terminal notes menu works. Tests pass: `broadcast` and `couchtty` (after generating the runtime bundle in this disposable checkout), and 60/60 viewer `node --test` tests. Since the M4 close, three operator-smoke commits added a right-click "copy and resolve" in the terminal menu (tested), made the flash shorter with finer steps, and made the resting glow fainter. Nothing blocks the gate. Three prior Minors are still open (BR-16, BR-27; BR-19 is now closed). The smoke changes also left two new doc/plan drifts: the README now contradicts itself about where notes are resolved, and the plan's flash and repaint figures are out of date.

**Strengths**
- `connect()` in `viewer.js:180-218` only sends caps-off at an end if the page saw caps. `connect.test.mjs` pins both the pointer-page and view-page cases, so BR-13 is closed properly.
- A palette reply now goes through `restyleNotes` (`terminal_input.go:66`), and a test at `console_notes_test.go:249` covers it.
- The right click copies before it resolves (`console_notes.go:299-313`), so the text is never lost. It reuses `Session.ResolveNote` → `resolveNote`, the same path as a viewer's Resolve, and `TestNotesMenuRightClickResolves` asserts that the copy, the resolve, the notice and the menu close all happen and that the child gets nothing.
- `noteTextLocked` replaced the inline lookup loop when the second caller arrived (ARCH-DRY).
- The flash tests are now written in terms of the `NoteFlashHold`, `NoteFlashFade` and `noteFadeSteps` constants, not literal times, so the next retune won't break them.

**Critical findings:** none.

**Important findings:** none.

**Minor findings**
- `README.md:903-904` says "Notes are resolved from a viewer's page", but the same paragraph now says a right click in the terminal menu resolves a note. This is the 4th finding in family `docs-gate-user-surface`. The rule: when a commit changes what a surface can do, it sweeps every README and atlas sentence that describes that capability (grep the verb, e.g. "resolve"), not only the sentence it extends.
- The plan's Flash and `NoteNextChange` bullets (plan:287-288) still say a 1.5s hold, 10 steps, and about 11 repaints per beat and per arrival. The code now has a 300ms hold and 25 steps of 20ms, about 26 repaints per beat and per arrival. That means about 50 repaints a second during a fade, and that rate can last while saves arrive near the 2/s limit. Only the Revisions log (plan:401) records this. This is the 3rd finding in family `plan-tracks-code`. The rule: a commit that changes a constant which a Core concepts bullet states (including an ARCH-CONSTRAINTS budget) rewrites that bullet in the same commit. A Revisions line on its own leaves the plan contradicting itself.
- BR-16 and BR-27 are still open (dispositions below).
- The `ResolveNote` error branch ("Couldn't resolve the note.") has no test. It is cheap to reach with pointing off mid-gesture.

**Test coverage notes:** Reverting the `sawCaps` guard turns the view-page test red. The right-click path is covered end to end. Nothing tests the stamp clock from `Config.Now` (BR-16), and nothing tests a drag from outside that crosses the menu (BR-27).

**Architectural notes**
- **ARCH-DRY:** pass. `noteTextLocked` is shared and `blendable` has a single site.
- **ARCH-PURE:** pass. Glow, flash and menu layout are pure and tested without IO.
- **ARCH-PURPOSE:** pass.
- **ARCH-MOCK:** N/A. No external binary or service.
- **ARCH-CONSTRAINTS:** pass on the code, since one shared beat still bounds repaints whatever the number of notes. The stated envelope in the plan is stale (Minor above).
- **ARCH-SECURE:** pass. Bodies are decoded strictly, and the right click resolves only ids the console already holds.
- **ARCH-ORDER:** flag (BR-27). `pressed` is a bool, but the gesture has three owners: none, the menu, or elsewhere.
- **ARCH-FUNERAL:** pass. Notes live only in memory and are cleared on pointer-off and broadcast end; the timers are stopped in `forgetNotes`.

**Plan revision recommendations**
- Rewrite plan:287-288 to say a 300ms hold, a 500ms fade in 25 steps of 20ms, about 26 repaints per beat and per arrival, and a peak of about 50 per second during a fade.
- Bring Task 12's test bullets (plan:297, 301) in line: they still say "strength 1", which is the inverse of `t`, and give 1.4s/1.6s flash timings.
- Add a Revisions note that M4's "read and copy only" scope now includes right-click resolve.

```findings
dispose:
  - id: BR-7
    disposition: addressed
    note: |
      M2 viewer UI (notes.js, mode button, dialog) is present at head, so README's viewer-page note actions now describe shipped surface.
  - id: BR-13
    disposition: addressed
    note: |
      viewer.js:182,195,215 gate caps-off on sawCaps; connect.test.mjs view-page end test asserts no caps, red without the guard.
  - id: BR-16
    disposition: not-addressed
    note: |
      Config.Now still wired (console_broadcast.go:155) but no test drives a note stamp and its drawing through the console's fake clock.
  - id: BR-18
    disposition: addressed
    note: |
      Core concepts plan:285 now states paletteTint in t (0 = full) with a Revisions entry at plan:389; Task 12's stale wording is covered by the new plan-tracks-code finding.
  - id: BR-19
    disposition: addressed
    note: |
      restyleNotes runs on every palette reply (terminal_input.go:66) and the test at console_notes_test.go:249 pins the immediate restyle.
  - id: BR-27
    disposition: not-addressed
    note: |
      pnotes.pressed is still a bool; button-held motion from a press outside the box is taken as hover at console_notes.go:276-283.
findings:
  - id: new
    severity: Minor
    family: docs-gate-user-surface
    title: |
      README says notes are resolved from a viewer's page, but the terminal menu's right click now resolves them too
    detail: |
      4th in this family. README.md:903-904 contradicts the right-click sentence added at 902. Rule: a commit that changes a capability sweeps every README/atlas sentence describing it (grep the verb) in the same commit.
  - id: new
    severity: Minor
    family: plan-tracks-code
    title: |
      Plan Flash and NoteNextChange bullets still state a 1.5s hold, 10 steps and about 11 repaints per beat; code is 300ms, 25x20ms, about 26
    detail: |
      3rd in this family. plan:287-288 and Task 12 (297, 301) are stale; only Revisions plan:401 records the smoke change, and the peak of about 50 repaints/s during a fade is unstated. Rule: a commit that changes a constant a Core concepts bullet states rewrites that bullet in the same commit.
```
