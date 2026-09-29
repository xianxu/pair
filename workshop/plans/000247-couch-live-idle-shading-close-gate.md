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

## Open findings

- **BR-1** [Minor] `test-prose-enumeration` Tasks 3, 4, 6 and 7 enumerate test cases in prose instead of one strategy line per risky function
- **BR-2** [Minor] `glyph-color-derives-from-source` Glyphs with their own colour are painted faded amber no matter what colour slotGlyphSGR returned
- **BR-3** [Minor] `atlas-describes-actual-surface` The atlas says both views read MenuState.Activity/Palette, but the tab bar reads StatusActor.Idle and StatusModel.Palette
- **BR-4** [Minor] `fade-precedence-single-source` Each surface encodes the fade-is-weakest-cue precedence rule separately
