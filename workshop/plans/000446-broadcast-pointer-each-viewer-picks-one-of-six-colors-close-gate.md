---
gate: boundary-review
issue: 446
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T12:19:56-07:00"
      dispatched: "2026-10-11T12:18:14-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: README pointer section still says marks are amber and omits the color picker
          detail: README.md:880 says "leaving amber marks". Describe the six colors, the per-viewer automatic color, and the swatch pick remembered in the browser.
          family: docs-gate-user-surface
          round: 1
        - id: BR-2
          severity: Minor
          title: Fixed swatch bar covers the top-left terminal cells, which can't be seen or pointed at
          detail: 'viewer.css:856 puts #swatches at top:8px left:8px over the stage, so taps there hit the swatches, not the stage.'
          family: ui-overlay-occludes-content
          round: 1
        - id: BR-3
          severity: Minor
          title: TestPointerPaletteMatchesMarks checks only two of six entries
          family: weak-assertion
          round: 1
        - id: BR-4
          severity: Minor
          title: Fade ladders without truecolor darken toward a dark background, so on light terminals the fade darkens instead of fading out
          family: spec-light-background
          round: 1
        - id: BR-5
          severity: Minor
          title: Done-when live two-viewer broadcast check was replaced by an in-process test and left to the operator
          family: done-when-live-check
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-11T12:23:28-07:00"
      dispatched: "2026-10-11T12:21:11-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: README.md:880-884 now says colored marks, six colorblind-safe colors, a per-viewer automatic color, and a swatch pick remembered by the browser; this matches pointer.go claimColor and viewer.js colorStore.
          round: 2
        - id: BR-2
          disposition: addressed
          note: 'viewer.css #stage.controls reserves a 40px band and fit measures #stage; by inspection the swatches top out about 36px and the hint about 34px from the bottom, so neither covers a cell. There is no DOM harness for an automated check.'
          round: 2
        - id: BR-3
          disposition: addressed
          note: TestPointerPaletteMatchesMarks (marks_test.go:402) checks all six hex values and PointerColors; TestPointerCapsEvents also checks the full palette in caps.
          round: 2
        - id: BR-4
          disposition: withdrawn
          note: The darkening fade without truecolor is amber's existing behavior, now applied to each color; the spec's "reads on both backgrounds" covers the full-strength color, and truecolor blends into the real background. Recorded as a plan revision, not a defect.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: Done-when still says "checked live in a broadcast" and no live check was run; either run and log it, or revise Done-when to name the in-process TestPointerStreamsGetDistinctColors.
          round: 2
      findings:
        - id: BR-6
          severity: Minor
          title: Hint and swatches are positioned separately in the band and overlap on phone-width screens
          detail: '2nd in this family. Rule: every fixed control on a pointer page lives in one reserved band container laid out as a wrapping flex row, never positioned independently over the stage or another control. Put #swatches and #hint in one band element and size #stage.controls bottom from it. At about 390px wide, the swatches (about 182px from the left) and the hint (about 280px from the right) overlap by about 80px.'
          family: ui-overlay-occludes-content
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: bc0137b13d3cca228f2c18b9b42c0ad230cd4a5b
      blocked: false
    - "n": 3
      timestamp: "2026-10-11T12:29:02-07:00"
      dispatched: "2026-10-11T12:28:06-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: addressed
          note: Done-when now cites TestPointerStreamsGetDistinctColors (pointer_test.go:455, which runs through the real server) and the Revisions entry records why the live check was deferred; this records the deferral and adds no behavior.
          round: 3
        - id: BR-6
          disposition: addressed
          note: '#swatches and #hint now share one fixed #controls box laid out as a wrapping row (index.html, viewer.css), and viewer.js band() sets the screen''s bottom from the box''s measured height on caps and on resize, so the general rule is fixed.'
          round: 3
      recipe: fix-review
      isolation: fenced
      reviewed: 1fad8ca5665ba6550c0179c005c0d06df22caa1c
      blocked: false
---

# Gate ledger — pair#446 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T12:19:56-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `docs-gate-user-surface` README pointer section still says marks are amber and omits the color picker
  README.md:880 says "leaving amber marks". Describe the six colors, the per-viewer automatic color, and the swatch pick remembered in the browser.
- **BR-2** [Minor] `ui-overlay-occludes-content` Fixed swatch bar covers the top-left terminal cells, which can't be seen or pointed at
  viewer.css:856 puts #swatches at top:8px left:8px over the stage, so taps there hit the swatches, not the stage.
- **BR-3** [Minor] `weak-assertion` TestPointerPaletteMatchesMarks checks only two of six entries
- **BR-4** [Minor] `spec-light-background` Fade ladders without truecolor darken toward a dark background, so on light terminals the fade darkens instead of fading out
- **BR-5** [Minor] `done-when-live-check` Done-when live two-viewer broadcast check was replaced by an in-process test and left to the operator

## Round 2 — 2026-10-11T12:23:28-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — README.md:880-884 now says colored marks, six colorblind-safe colors, a per-viewer automatic color, and a swatch pick remembered by the browser; this matches pointer.go claimColor and viewer.js colorStore.
- BR-2 — addressed — viewer.css #stage.controls reserves a 40px band and fit measures #stage; by inspection the swatches top out about 36px and the hint about 34px from the bottom, so neither covers a cell. There is no DOM harness for an automated check.
- BR-3 — addressed — TestPointerPaletteMatchesMarks (marks_test.go:402) checks all six hex values and PointerColors; TestPointerCapsEvents also checks the full palette in caps.
- BR-4 — withdrawn — The darkening fade without truecolor is amber's existing behavior, now applied to each color; the spec's "reads on both backgrounds" covers the full-strength color, and truecolor blends into the real background. Recorded as a plan revision, not a defect.
- BR-5 — not-addressed — Done-when still says "checked live in a broadcast" and no live check was run; either run and log it, or revise Done-when to name the in-process TestPointerStreamsGetDistinctColors.

### Raised

- **BR-6** [Minor] `ui-overlay-occludes-content` Hint and swatches are positioned separately in the band and overlap on phone-width screens
  2nd in this family. Rule: every fixed control on a pointer page lives in one reserved band container laid out as a wrapping flex row, never positioned independently over the stage or another control. Put #swatches and #hint in one band element and size #stage.controls bottom from it. At about 390px wide, the swatches (about 182px from the left) and the hint (about 280px from the right) overlap by about 80px.

## Round 3 — 2026-10-11T12:29:02-07:00 (claude) — passed

### Disposed

- BR-5 — addressed — Done-when now cites TestPointerStreamsGetDistinctColors (pointer_test.go:455, which runs through the real server) and the Revisions entry records why the live check was deferred; this records the deferral and adds no behavior.
- BR-6 — addressed — #swatches and #hint now share one fixed #controls box laid out as a wrapping row (index.html, viewer.css), and viewer.js band() sets the screen's bottom from the box's measured height on caps and on resize, so the general rule is fixed.

## Open findings

(none — every finding has been disposed)
