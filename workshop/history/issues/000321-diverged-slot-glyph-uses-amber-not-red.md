---
id: 000321
status: done
deps: [pair#319]
github_issue:
created: 2026-09-24
updated: 2026-09-24
estimate_hours:
started: 2026-09-24T19:55:35-07:00
flow: {kind: quick, provenance: inferred, spec: "ce106ab5", done: "c8e1406f"}
actual_hours: 0.17
---

# Diverged slot glyph uses amber, not red

## Problem

#319 draws the diverged slot glyph `±` in red and the dirty mark `*` in amber.
In live use, the operator wants one attention colour: `±` should use the same
amber as `*`.

## Spec

- `couchtty.slotGlyphSGR` (the single per-character styling decision shared by
  the tab bar and the switcher) returns `attentionSGR` for
  `couchcore.SlotGlyphDiverged` as well as `SlotGlyphDirty`.
- Remove `slotAlertSGR`; nothing else uses it.
- No change to which glyphs appear (`couchcore.SlotGlyph` is untouched).
- Update the README glyph table (`±` "(red)" → "(amber)") and the #317/#319
  section of `atlas/couch.md`.

## Done when

- `TestSlotGlyphColoursInBothViews` asserts `±` and `*` are amber in both views,
  `+`, `-` and the branch glyph are uncoloured, and nothing is coloured without
  256-colour support; it fails if `±` loses its colour.
- Goldens are byte-unchanged (they strip ANSI).
- Live check: a diverged slot shows an amber `±`.

## Plan

- [x] Test first: `TestSlotGlyphColoursInBothViews` expects amber `±` in both views
- [x] `slotGlyphSGR`: `±` → `attentionSGR`; delete `slotAlertSGR`
- [x] README + atlas colour wording; mutation-check; package tests unsandboxed

## Log

### 2026-09-24
- 2026-09-24: closed — TestSlotGlyphColoursInBothViews asserts amber ± and * in tab bar and switcher, no red, uncoloured +/-/branch; failed on the red build first; mutation (drop ± from amber case) caught; goldens byte-unchanged; couchtty/couchcmd/artifactpath green unsandboxed with clean env. Operator live check not run: operator asked to ship directly.; review verdict: SHIP

- Filed from the #319 smoke test. Blocked on #319 landing on origin/main; then
  pull slot 1 and implement here.
- Implemented after #319 landed (65a434fa). `slotGlyphSGR` maps `±` and `*` to
  `attentionSGR`; `slotAlertSGR` deleted. Test now asserts amber `±`/`*` in both
  views and no red anywhere; it failed on the red build first. Mutation: drop
  `±` from the amber case → test fails. Goldens byte-unchanged. couchtty,
  couchcmd, artifactpath green unsandboxed (clean env).

