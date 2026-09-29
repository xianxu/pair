# Boundary Review — pair#247 (milestone M2)

| field | value |
|-------|-------|
| issue | 247 — Shade live Couch threads by idle time |
| repo | pair |
| issue file | workshop/issues/000247-couch-live-idle-shading.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | ddb02f76611629759cb1515dd82236dd2437af18..d002bae0f4beec03fec862ba279bdb95be9f4d7f |
| command | sdlc milestone-close --issue 247 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-28T23:46:13-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

**Summary:** Most of M2 is done. There is one shared `threadactivity.Latest`, and the title poller now reads it. The console activity pass follows the slot-git pass: single-flight, a timeout per thread, work kept outside `c.mu`, and `mergeObservations` shared by both passes. The OSC 10/11 palette query goes out before the first frame, and replies are captured in `routeInputEvent`. The README and atlas are updated.

Every package the window touches passes. I ran them outside the sandbox because the pty tests need it. The one failure is `artifactpath`, which fails only on `reviewcmd/*` and `nvim/review*` paths. This window doesn't touch those paths, and the issue log records the same failure on a clean origin/main.

Two things stand between this and SHIP:
- **Operator smoke not done.** The M2 row promises a dark- and light-theme operator smoke, and it hasn't been run or recorded.
- **Switcher not tested through its real render path.** The Done-when asks for tests that deliver activity into *both renderers*. The switcher check recomputes `IdleLevelFor` from state instead of rendering through `showMenu`.

**1. Strengths**
- `threadactivity.Latest` (`cmd/internal/threadactivity/activity.go:785`) is pure given its `Runtime`. It explains why the draft is excluded, and `TestLatestIgnoresTheDraft` pins that. `TestLatestPicksTheNewestSignal` "launch only" pins the PQ-1 regression.
- Turning `mergeSlotGit` into the generic `mergeObservations` (`menu.go:2071`) makes one merge rule serve both background passes (ARCH-DRY).
- `slotGlyphBase` returns a base colour instead of an escape, so the tab bar and the switcher fade glyphs through the same `FadeStyle` call (`reserve.go:650`, `menu_render.go:629`).
- `ensureMenuLocked` removes two duplicated lazy builds and keeps any palette that arrived before the menu existed. It has a direct test.
- The lesson about terminal fakes draining replies is general, and both affected fakes (soak, codex) were fixed.

**2. Critical findings:** none.

**3. Important findings**
- **Operator smoke still open.** In the plan, Task 9's "Live smoke by the operator on a dark AND a light theme…" is unchecked, and the Log records no smoke. The issue's M2 row and the Done-when line "Dark/light theme and no-color checks pass" both depend on it. The light-theme check of the 65 % amber (≈`#fff1a6`) is the one visual risk tests can't catch.
  - Fix: run the smoke (dark, light, `NO_COLOR=1`) and log it before the M2 close.
- **Switcher assertion bypasses the switcher.** `console_activity_test.go` `switcherIdle()` recomputes `IdleLevelFor(clock.Now(), state.Activity[...])` instead of rendering.
  - Consequence: if `showMenu` went back to `time.Now()` (`console_menu.go:211`), or `renderRootMenuFrame` read the wrong address, no test would fail. The plan promised "rendered through `showMenu` with the injected clock".
  - Fix: capture the switcher frame through the production path (focus the panel, or call `RenderMenuView(f.con.menuSnapshot(), …, f.con.now(), true)`). Assert that the attached row carries `FadeStyle(p, IdleDay, baseDefault)` bytes, then `IdleStale` after the clock advances.

**4. Minor findings**
- **Budget overrun not closed.** The measured 114 ms per thread (~2.3 s for 20 threads) is over the plan's < 2 s budget. The cause is that each probe rebuilds `sessioninventory.NewOSRuntime` and lists the whole store (`threadactivity/os.go:837`). The fix (one listing per agent per pass) is already named, but "raised with the operator" has no recorded outcome. (ARCH-CONSTRAINTS)
- **Repaint pattern copied three times.** `repaintAfterPalette` repeats the "`showMenu` if the panel has focus, else `repaint`" tail that `finishActivity` and `finishSlotGit` also carry. `finishActivity` could call the helper. (ARCH-DRY)
- **Palette state split across two places.** `paletteFG` and `paletteBG` live on `Console`, while `Palette.Known` lives in `MenuState`. `Known` could be derived from the palette alone, or the two flags moved into `Palette`. (ARCH-ORDER, small)
- **Manifest entry misplaced.** `manifest.go`: `cmd/internal/threadactivity/os.go` sits inside the `sessioninventory` block of `NonArtifactSources`, which breaks the grouping.
- **Switch trigger untested.** No test covers the pass that a thread switch requests (`console.go:550`); the inventory-landing trigger is covered.
- **Nothing tests `wireIdleFading`.** The plan accepts this and leaves it to the live smoke, which is one more reason to run the smoke.

**5. Test coverage notes**
- **Pure logic:** `Latest` table tests plus cancellation and draft exclusion. `InScope` rejects path traversal.
- **Console:** the pass probes live threads only, each once. It keeps the last value on failure, drops threads that stop being live, and survives a restart. Crossing a band on the injected clock is covered for the chip model.
- **Palette:** covered are query order and count, two replies before Known, a malformed reply, and replies never reaching the child.
- **Gap:** delivery into both renderers is only asserted at model level (the Important finding above).

**6. Architecture pass**
- **ARCH-DRY:** pass. Minor: the repaint tail above.
- **ARCH-PURE:** pass. `Latest`, `IdleLevelFor` and `FadeStyle` are pure; IO is confined to `OSRuntime`, the query write and the capture.
- **ARCH-PURPOSE:** pass. Couch and the title poller both derive from `threadactivity.Latest`, and no copy of the definition remains.
- **ARCH-MOCK:** pass. The activity probe fake keeps state (replies, failures, a call log), and the `Runtime` fake is map-backed. There is no conformance check of `OSRuntime`, which is acceptable for a thin `os.Stat`/inventory wrapper.
- **ARCH-CONSTRAINTS:** flag (Minor). The budget overrun is measured and disclosed but not resolved; see above.
- **ARCH-SECURE:** pass. ultraviolet parses the OSC replies, a nil colour is ignored, and `InScope` validates the scope.
- **ARCH-ORDER:** pass with a Minor note.
  - The query is written after `MakeRaw` and before the first frame, which a test pins.
  - Results are gated on their generation through the shared `RefreshSchedule`.
  - Workers are joined by `c.workers` and exit on `c.stop`.
- **ARCH-FUNERAL:** pass. Nothing durable is created, and `Activity` is rebuilt over the live set on each pass.

**7. Plan revision recommendations**
- Add to the Revisions entry for M2 that the switcher side of Task 6's "rendered through `showMenu`" case is currently checked at model level only (or fix the test).
- Record the operator's decision on the 2.3 s-per-20-threads overrun: accept it, or build the shared listing.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Tasks 3, 4, 6 and 7 each now carry a one-line "Test strategy:" (risky function, guard, mutation); the remaining prose describes delivered, ticked steps.
findings:
  - id: new
    severity: Important
    family: plan-item-claimed-undelivered
    title: |
      M2's operator smoke on dark and light themes (and NO_COLOR) is unticked and unlogged
    detail: |
      Plan Task 9's live-smoke item is open, and the M2 row plus the Done-when ("Dark/light theme and no-color checks pass") depend on it. The light-theme 65 % amber legibility check can only be done live. Run it and log the result before the M2 close.
  - id: new
    severity: Important
    family: test-reaches-production-render-path
    title: |
      The switcher-side activity test recomputes IdleLevelFor instead of rendering through showMenu or renderRootMenuFrame
    detail: |
      switcherIdle() in console_activity_test.go restates the classifier, so reverting showMenu's c.now() to time.Now() or breaking the live-row lookup would not fail any test. Assert the faded bytes in a rendered switcher frame taken from menuSnapshot and the injected clock.
  - id: new
    severity: Minor
    family: operating-envelope-enforced
    title: |
      The activity pass measures 114 ms per thread (~2.3 s for 20), over the plan's < 2 s budget, with no recorded decision
    detail: |
      Each probe rebuilds sessioninventory.NewOSRuntime and lists the store (threadactivity/os.go). Either record that the operator accepts the overrun or build the planned shared listing per agent per pass.
  - id: new
    severity: Minor
    family: repaint-after-background-pass
    title: |
      repaintAfterPalette repeats the panel-focus repaint tail also found in finishActivity and finishSlotGit
    detail: |
      Three copies of "showMenu if the panel has focus, else repaint". The two new sites could call one helper.
```
