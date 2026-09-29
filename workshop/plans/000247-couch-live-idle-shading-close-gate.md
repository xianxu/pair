---
gate: boundary-review
issue: 247
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-28T22:19:41-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Tasks 3, 4, 6 and 7 enumerate test cases in prose instead of one strategy line per risky function
          detail: (carried from plan-quality PQ-4, deferred to the boundary review)
          family: test-prose-enumeration
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-28T22:19:41-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Minor
          title: Glyphs with their own colour are painted faded amber no matter what colour slotGlyphSGR returned
          detail: colorMenuGlyph (menu_render.go) and RenderStatusRow's glyph loop (reserve.go) only check that slotGlyphSGR is non-empty, then substitute amber. That is correct while slotGlyphSGR only returns attentionSGR, but a new glyph colour would silently turn amber. Fading from the glyph's own base (e.g. slotGlyphSGR returns a styleBase) removes the coupling.
          family: glyph-color-derives-from-source
          round: 2
        - id: BR-3
          severity: Minor
          title: The atlas says both views read MenuState.Activity/Palette, but the tab bar reads StatusActor.Idle and StatusModel.Palette
          detail: atlas/couch.md idle-fading paragraph. It also describes the feature as live, but nothing writes these fields until M2; correct it when M2 wiring lands.
          family: atlas-describes-actual-surface
          round: 2
        - id: BR-4
          severity: Minor
          title: Each surface encodes the fade-is-weakest-cue precedence rule separately
          detail: The tab bar uses !Placeholder && !Active && !Bell; the switcher uses branch order plus a no-attention check. Acceptable because the inputs differ; a cross-reference comment would keep the two in step.
          family: fade-precedence-single-source
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: false
    - "n": 3
      timestamp: "2026-09-28T23:46:13-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Tasks 3, 4, 6 and 7 each now carry a one-line "Test strategy:" (risky function, guard, mutation); the remaining prose describes delivered, ticked steps.
          round: 3
      findings:
        - id: BR-5
          severity: Important
          title: M2's operator smoke on dark and light themes (and NO_COLOR) is unticked and unlogged
          detail: Plan Task 9's live-smoke item is open, and the M2 row plus the Done-when ("Dark/light theme and no-color checks pass") depend on it. The light-theme 65 % amber legibility check can only be done live. Run it and log the result before the M2 close.
          family: plan-item-claimed-undelivered
          round: 3
        - id: BR-6
          severity: Important
          title: The switcher-side activity test recomputes IdleLevelFor instead of rendering through showMenu or renderRootMenuFrame
          detail: switcherIdle() in console_activity_test.go restates the classifier, so reverting showMenu's c.now() to time.Now() or breaking the live-row lookup would not fail any test. Assert the faded bytes in a rendered switcher frame taken from menuSnapshot and the injected clock.
          family: test-reaches-production-render-path
          round: 3
        - id: BR-7
          severity: Minor
          title: The activity pass measures 114 ms per thread (~2.3 s for 20), over the plan's < 2 s budget, with no recorded decision
          detail: Each probe rebuilds sessioninventory.NewOSRuntime and lists the store (threadactivity/os.go). Either record that the operator accepts the overrun or build the planned shared listing per agent per pass.
          family: operating-envelope-enforced
          round: 3
        - id: BR-8
          severity: Minor
          title: repaintAfterPalette repeats the panel-focus repaint tail also found in finishActivity and finishSlotGit
          detail: Three copies of "showMenu if the panel has focus, else repaint". The two new sites could call one helper.
          family: repaint-after-background-pass
          round: 3
      boundary: M2
      recipe: milestone-review
      blocked: true
    - "n": 4
      timestamp: "2026-09-28T23:57:40-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: addressed
          note: 'Dark-theme smoke passed and is logged; light/NO_COLOR live checks explicitly moved to #343 via Revisions in plan+issue, unit tests cover both.'
          round: 4
        - id: BR-6
          disposition: addressed
          note: New switcher test reads the rendered cell under a week-offset clock; reverting console_menu.go:209 to time.Now() turns it red (verified in scratch copy).
          round: 4
        - id: BR-7
          disposition: addressed
          note: 'Probe-cost overrun recorded as operator-deferred to #343 in the plan Revisions and issue log.'
          round: 4
        - id: BR-8
          disposition: addressed
          note: finishActivity, finishSlotGit and the palette path all call repaintVisible (console_palette.go:84).
          round: 4
      boundary: M2
      recipe: milestone-review
      blocked: false
    - "n": 5
      timestamp: "2026-09-28T23:59:35-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: addressed
          note: slotGlyphBase now returns a styleBase consumed by FadeStyle in both colorMenuGlyph and RenderStatusRow; a new glyph colour must add a base, it cannot silently become amber.
          round: 5
        - id: BR-3
          disposition: addressed
          note: atlas/couch.md now says the tab bar reads Activity/Palette via statusModelLocked into StatusActor.Idle/StatusModel.Palette, matching console_presentation.go, and M2 wiring is live.
          round: 5
        - id: BR-4
          disposition: addressed
          note: Cross-reference comments now sit in RenderStatusRow (reserve.go) and renderRootMenuFrame's live-row branch (menu_render.go), each naming the other.
          round: 5
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#247 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-28T22:19:41-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `test-prose-enumeration` Tasks 3, 4, 6 and 7 enumerate test cases in prose instead of one strategy line per risky function
  (carried from plan-quality PQ-4, deferred to the boundary review)

## Round 2 — 2026-09-28T22:19:41-07:00 (claude) — passed

### Raised

- **BR-2** [Minor] `glyph-color-derives-from-source` Glyphs with their own colour are painted faded amber no matter what colour slotGlyphSGR returned
  colorMenuGlyph (menu_render.go) and RenderStatusRow's glyph loop (reserve.go) only check that slotGlyphSGR is non-empty, then substitute amber. That is correct while slotGlyphSGR only returns attentionSGR, but a new glyph colour would silently turn amber. Fading from the glyph's own base (e.g. slotGlyphSGR returns a styleBase) removes the coupling.
- **BR-3** [Minor] `atlas-describes-actual-surface` The atlas says both views read MenuState.Activity/Palette, but the tab bar reads StatusActor.Idle and StatusModel.Palette
  atlas/couch.md idle-fading paragraph. It also describes the feature as live, but nothing writes these fields until M2; correct it when M2 wiring lands.
- **BR-4** [Minor] `fade-precedence-single-source` Each surface encodes the fade-is-weakest-cue precedence rule separately
  The tab bar uses !Placeholder && !Active && !Bell; the switcher uses branch order plus a no-attention check. Acceptable because the inputs differ; a cross-reference comment would keep the two in step.

## Round 3 — 2026-09-28T23:46:13-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Tasks 3, 4, 6 and 7 each now carry a one-line "Test strategy:" (risky function, guard, mutation); the remaining prose describes delivered, ticked steps.

### Raised

- **BR-5** [Important] `plan-item-claimed-undelivered` M2's operator smoke on dark and light themes (and NO_COLOR) is unticked and unlogged
  Plan Task 9's live-smoke item is open, and the M2 row plus the Done-when ("Dark/light theme and no-color checks pass") depend on it. The light-theme 65 % amber legibility check can only be done live. Run it and log the result before the M2 close.
- **BR-6** [Important] `test-reaches-production-render-path` The switcher-side activity test recomputes IdleLevelFor instead of rendering through showMenu or renderRootMenuFrame
  switcherIdle() in console_activity_test.go restates the classifier, so reverting showMenu's c.now() to time.Now() or breaking the live-row lookup would not fail any test. Assert the faded bytes in a rendered switcher frame taken from menuSnapshot and the injected clock.
- **BR-7** [Minor] `operating-envelope-enforced` The activity pass measures 114 ms per thread (~2.3 s for 20), over the plan's < 2 s budget, with no recorded decision
  Each probe rebuilds sessioninventory.NewOSRuntime and lists the store (threadactivity/os.go). Either record that the operator accepts the overrun or build the planned shared listing per agent per pass.
- **BR-8** [Minor] `repaint-after-background-pass` repaintAfterPalette repeats the panel-focus repaint tail also found in finishActivity and finishSlotGit
  Three copies of "showMenu if the panel has focus, else repaint". The two new sites could call one helper.

## Round 4 — 2026-09-28T23:57:40-07:00 (claude) — passed

### Disposed

- BR-5 — addressed — Dark-theme smoke passed and is logged; light/NO_COLOR live checks explicitly moved to #343 via Revisions in plan+issue, unit tests cover both.
- BR-6 — addressed — New switcher test reads the rendered cell under a week-offset clock; reverting console_menu.go:209 to time.Now() turns it red (verified in scratch copy).
- BR-7 — addressed — Probe-cost overrun recorded as operator-deferred to #343 in the plan Revisions and issue log.
- BR-8 — addressed — finishActivity, finishSlotGit and the palette path all call repaintVisible (console_palette.go:84).

## Round 5 — 2026-09-28T23:59:35-07:00 (claude) — passed

### Disposed

- BR-2 — addressed — slotGlyphBase now returns a styleBase consumed by FadeStyle in both colorMenuGlyph and RenderStatusRow; a new glyph colour must add a base, it cannot silently become amber.
- BR-3 — addressed — atlas/couch.md now says the tab bar reads Activity/Palette via statusModelLocked into StatusActor.Idle/StatusModel.Palette, matching console_presentation.go, and M2 wiring is live.
- BR-4 — addressed — Cross-reference comments now sit in RenderStatusRow (reserve.go) and renderRootMenuFrame's live-row branch (menu_render.go), each naming the other.

## Open findings

(none — every finding has been disposed)
