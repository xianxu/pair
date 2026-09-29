# Boundary Review — pair#247 (milestone M1)

| field | value |
|-------|-------|
| issue | 247 — Shade live Couch threads by idle time |
| repo | pair |
| issue file | workshop/issues/000247-couch-live-idle-shading.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 6f62a7ecbbb62d9d1188ccc7aeed52daf27d9bae..3aee5917af753663357a7f226acd5a43255d2924 |
| command | sdlc milestone-close --issue 247 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-28T22:19:41-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

M1 delivers everything it claims (Plan Tasks 1–4). There is one pure policy file, `idle_shade.go`, with `IdleLevelFor`, `FadeStyle`, `blend` and `quantize256`. Both renderers use it: the tab bar through `RenderStatusRow`, and the switcher's live rows through `renderRootMenuFrame` and `colorMenuGlyph`. Selection, bell/attention and placeholder styling win over the fade, in the order the Spec requires. Level 0 and `NO_COLOR` produce exactly the same bytes as before, so M1 changes nothing on screen until M2 starts filling in `MenuState.Activity` / `Palette` / `StatusActor.Idle`. The plan's 2026-09-28 revision records that deferral explicitly.

I read the stat, name-status and full diffs. The focused tests pass (`go test ./cmd/internal/couchtty -run 'Idle|Blend|Quantize|FadeStyle|RenderStatusRow|Switcher|Menu'` → ok), and `go vet` is clean. Nothing blocks shipping; the findings below are all Minor.

1. **Strengths**
   - `idle_shade.go:29-40`: one classifier with a threshold table. Its boundaries are inclusive, and unknown, zero or future times count as fresh; `TestIdleLevelForBoundaries` checks each case to the second.
   - `FadeStyle` sends everything that doesn't fade (level 0 or NO_COLOR) through a single early return, so existing output can't change. Two tests pin this across all palettes: `TestFadeStyleLevelZeroIsTodaysBytesInEveryPalette` and `TestRenderStatusRowFreshChipIsUnchanged`.
   - The light/dark test states an ordering (fading darkens on dark themes and lightens on light ones) rather than repeating the weight table. That tests the actual requirement, not the implementation.
   - `reserve_idle_test.go` checks that the fade leaves `ChipSpan`s unchanged, so clicks still land on the right chip.
   - `quantize256` skips the 16 system colours, whose RGB the terminal's theme can redefine. That is correct.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `menu_render.go:636` / `reserve.go:~192`: a glyph with its own colour gets the faded amber unconditionally. `colorMenuGlyph` and the tab-bar glyph loop test `slotGlyphSGR(r) != ""` and then paint in amber, not in the colour `slotGlyphSGR` returned. That is correct today because `slotGlyphSGR` only ever returns `attentionSGR`. A future glyph with a different colour would silently turn amber, though. A fix is to fade from the glyph's own base, e.g. have `slotGlyphSGR` return a `styleBase`.
   - `atlas/couch.md:112`: "Both views read `MenuState.Activity` and `MenuState.Palette`" is wrong for the tab bar, which reads `StatusActor.Idle` and `StatusModel.Palette`. The paragraph also describes the feature as live, but nothing writes these fields until M2.
   - ARCH-DRY: each surface encodes the "fade is the weakest cue" rule separately. The tab bar uses `!Placeholder && !Active && !Bell`; the switcher uses the branch order plus a no-attention check. The inputs differ enough that this is acceptable, but a comment linking the two would help keep them in step.
   - The plan says the tests go into `reserve_test.go` / `menu_render_test.go`; they landed in new `*_idle_test.go` files. Cosmetic.

5. **Test coverage notes:** The pure layer is covered well, without any IO. The renderer tests cover the faded, fresh, unprobed, selected, attention, bell, placeholder and non-live cases. Mutation checks are recorded in the Log. Not covered yet, as expected: a chip that is both `Bell` and `Active` at an idle level (by inspection it takes the active path), and anything about how activity gets delivered (M2).

6. **Architecture notes for M2**
   - **ARCH-DRY:** pass. There is one classifier and one style function for both surfaces.
   - **ARCH-PURE:** pass. All the new logic is pure and the renderers take plain values.
   - **ARCH-PURPOSE:** pass for M1. The rest of the purpose (activity source, palette query) is correctly scheduled for M2, not dropped as a follow-up.
   - **ARCH-MOCK:** N/A in M1; M2's `threadactivity.Runtime` fake is planned.
   - **ARCH-CONSTRAINTS:** pass. Render cost is one map lookup per row.
   - **ARCH-SECURE:** N/A in M1; the OSC reply parsing in M2 must treat a malformed reply as an unknown palette.
   - **ARCH-ORDER:** pass. M1 carries no state between events; M2 needs the generation guard on activity results and the query ordering after `MakeRaw`.
   - **ARCH-FUNERAL:** pass. M1 creates nothing durable.
   - Things for M2 to remember: clone `Activity` in the state-copy helper (the revision already notes it), set `Palette.NoColor` and `TrueColor` from the environment, and fix the atlas sentence above.

7. **Plan revision recommendations:** none needed. The existing "M1 delivered" revision already accounts for the core-concepts table rows (`MenuEventActivity`/`MenuEventPalette`, `threadactivity.Latest`) moving to M2.

```findings
findings:
  - id: new
    severity: Minor
    family: glyph-color-derives-from-source
    title: |
      Glyphs with their own colour are painted faded amber no matter what colour slotGlyphSGR returned
    detail: |
      colorMenuGlyph (menu_render.go) and RenderStatusRow's glyph loop (reserve.go) only check that slotGlyphSGR is non-empty, then substitute amber. That is correct while slotGlyphSGR only returns attentionSGR, but a new glyph colour would silently turn amber. Fading from the glyph's own base (e.g. slotGlyphSGR returns a styleBase) removes the coupling.
  - id: new
    severity: Minor
    family: atlas-describes-actual-surface
    title: |
      The atlas says both views read MenuState.Activity/Palette, but the tab bar reads StatusActor.Idle and StatusModel.Palette
    detail: |
      atlas/couch.md idle-fading paragraph. It also describes the feature as live, but nothing writes these fields until M2; correct it when M2 wiring lands.
  - id: new
    severity: Minor
    family: fade-precedence-single-source
    title: |
      Each surface encodes the fade-is-weakest-cue precedence rule separately
    detail: |
      The tab bar uses !Placeholder && !Active && !Bell; the switcher uses branch order plus a no-attention check. Acceptable because the inputs differ; a cross-reference comment would keep the two in step.
```
