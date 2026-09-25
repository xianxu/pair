# Boundary Review — pair#321 (whole-issue close)

| field | value |
|-------|-------|
| issue | 321 — Diverged slot glyph uses amber, not red |
| repo | pair |
| issue file | workshop/issues/000321-diverged-slot-glyph-uses-amber-not-red.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7cd1ba5f814dcc7f63906e66f1a901e82812b3af..986af598a59ffa429ae3fa1396ea14a6249d2ecc |
| command | sdlc close --issue 321 |
| reviewer | claude |
| timestamp | 2026-09-24T20:04:42-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

Verdict: SHIP. This is a small change and it does what the issue asked. `slotGlyphSGR` (`cmd/internal/couchtty/reserve.go:67-72`) now gives `±` (diverged) and `*` (dirty) the same `attentionSGR` amber. `slotAlertSGR` is gone, and the pinned tree has no other reference to it. I grepped for `slotAlertSGR` and `38;5;196`: the only remaining `196` outside the test is an unrelated terminal-qualification fixture. The README glyph table and `atlas/couch.md` were both updated to say amber. The glyph set in `couchcore.SlotGlyph` is unchanged, as the spec requires. `TestSlotGlyphColoursInBothViews` passes at HEAD. It checks amber `±` and `*` in the tab bar across 8 glyph combinations and adds a check that fails if red `196` appears anywhere. The switcher check requires amber `±*`, and the plain (no 256-colour) menu check still requires no attention colour. The implementor's mutation claim holds: removing `SlotGlyphDiverged` from the amber case would break the `{"±", true, false}` tab-bar case and the switcher assertion. The Done-when "Live check" can't be verified from a diff. It's the operator's smoke test, not a code gap.

1. **Strengths**
   - One styling decision still covers both views: a single merged `case` in `slotGlyphSGR` (`reserve.go:70`), with no second place that decides colour.
   - The added regression check at `slotgit_presentation_test.go:94` hard-codes the old red escape sequence on purpose, so bringing red back fails even if someone reintroduces a new constant.
   - The docs moved with the code in the same range (README:441, `atlas/couch.md:82`).
   - The test fields were renamed from colour names (`red`/`amber`) to meanings (`diverged`/`dirty`), so the test no longer names a colour choice that could change again.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - The switcher half of the test uses only the `±*` fixture. `+`, `-` and the branch glyph are shown uncoloured in the tab bar but not in the switcher. This gap existed before #321, and because the styling helper is shared the risk is low.

5. **Test coverage:** Both Done-when states are covered. With 256-colour support, `±` and `*` are amber in both views. Without it, the plain `RenderMenu` has no colour. The reserved-colour check was reduced to `attentionSGR` only, which is correct now that `slotAlertSGR` no longer exists. The goldens strip ANSI, so they are unaffected.

6. **Architecture**
   - ARCH-DRY: pass. The two cases were merged into one and the constant that became redundant was deleted.
   - ARCH-PURE: pass. `slotGlyphSGR` is a pure function from a glyph to an SGR string, tested through the pure render functions.
   - ARCH-PURPOSE: pass. I checked every place that states the colour: the code, the test, README and atlas all say amber, and no text claiming red remains.

7. **Plan revisions:** none.

```findings
findings:
  - id: new
    severity: Minor
    family: both-views-assertion-symmetry
    title: |
      Switcher half of TestSlotGlyphColoursInBothViews only exercises the ±* fixture
    detail: |
      The tab-bar loop checks all 8 glyph combos, including that +, - and the branch glyph stay uncoloured. The switcher half checks only ±*. This predates #321 and the shared slotGlyphSGR keeps the risk low.
```
