---
gate: boundary-review
issue: 488
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T11:38:18-07:00"
      dispatched: "2026-10-12T11:34:19-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: README web-cast notes paragraph still says "tap an element, or drag a region" and pins "to its element (or region)"
          detail: 'README.md:1017-1020. M1 changed the user-facing gesture: every note is a box, a tap gets the default 1/10 box, touch needs 16 px, and the pin is the outlined box with the chip inside its top-right corner. Update the paragraph to match atlas/webcast-format.md:408-416.'
          family: docs-gate-readme-surface
          round: 1
        - id: BR-2
          severity: Minor
          title: Atlas and record.go doc comment reference broadcast.NoteAnchor and terminal cell boxes, which do not exist until M2/M3
          detail: atlas/webcast-format.md:206-209,216 and cmd/internal/webcast/recording/record.go:355 describe the terminal anchor and boxes in the present tense; grep finds no NoteAnchor type. Word them as planned, or add them when M2/M3 land.
          family: docs-describe-unbuilt-surface
          round: 1
        - id: BR-3
          severity: Minor
          title: ErrStale doc comment not updated for the new "box off the page" refusal
          detail: board.go:34 lists another document or a gone element; board.go:189 now also returns ErrStale when ClampToPage finds nothing on the page.
          family: error-doc-drift
          round: 1
        - id: BR-4
          severity: Minor
          title: Three distinct noteBox functions (collab gesture, annotate display, Go crop); annotate imports one as boxOfGesture
          detail: Rename the collab.js gesture function (e.g. gestureBox) so each name reads as one meaning.
          family: name-collision-across-layers
          round: 1
        - id: BR-5
          severity: Minor
          title: 26 px pin chip now sits inside the box and spills well outside thin boxes (e.g. a 12x1 px drag)
          family: ui-chip-fit
          round: 1
        - id: BR-6
          severity: Minor
          title: mountAnnotate glue (pageView, pointerType capture, boxCentre to nodeAt) has no DOM-level test
          detail: 'Gap predates #488; pointermode.test.mjs shows a harness that M2 could extend to annotate.'
          family: glue-untested
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T11:43:08-07:00"
      dispatched: "2026-10-12T11:39:45-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: README.md:1014-1022 now describes drag-or-tap-for-default-box, the touch threshold, and the outlined box with the chip inside top-right, matching annotate.css and collab.js DRAG_MIN.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: atlas:206-209 and record.go:355 fixed; atlas/webcast-format.md:217 still states terminal notes have a cell box and layout anchor in present tense (M2/M3).
          round: 2
        - id: BR-3
          disposition: addressed
          note: board.go:34-35 now lists the off-page box as an ErrStale cause, matching board.go:186-189.
          round: 2
        - id: BR-4
          disposition: addressed
          note: collab.js gesture function renamed gestureBox; annotate.js imports it unaliased; the remaining noteBox (JS display, Go crop.go) now share one meaning.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: Unchanged; annotate.css still fixes a 26 px chip at right:2px top:2px regardless of box size. Minor, non-blocking.
          round: 2
        - id: BR-6
          disposition: withdrawn
          note: A DOM-level test of mountAnnotate exists (viewer_annotate_live_test.go, Chrome-gated); the real gap is that it is stale, raised as a new finding.
          round: 2
      findings:
        - id: BR-7
          severity: Important
          title: Chrome-gated live annotate test still asserts the retired .pin.region class and element-sized tap pins
          detail: 'viewer_annotate_live_test.go:146 waits for .pin.region, which annotate.js:222 no longer sets, and :128-135 expects A''s tapped pin at the element box nb, but a tap now gets the default 1/10-view box. Select .pin[data-note=id] and expect noted.Region. Rule: when retiring a class, function or field, grep all consumers, including env-gated tests and the atlas, in the same commit.'
          family: retired-surface-sweep
          round: 2
        - id: BR-8
          severity: Minor
          title: 'atlas still mentions crop snapping after #488 retired it'
          detail: atlas/index.md:123 lists "snapped crops"; atlas/webcast-format.md:241 says the parent fallback is "not for snapping". Same sweep rule as the live-test finding.
          family: retired-surface-sweep
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 3
      timestamp: "2026-10-12T11:50:21-07:00"
      dispatched: "2026-10-12T11:47:17-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: not-addressed
          note: 'atlas/webcast-format.md:206-209 now says the path is one "#488 M3 adds", but :217 still says in the present tense that a terminal note has "a box in cells and Couch''s layout anchor", and record.go:356-357 says #488 "gives those a layout anchor". Neither exists until M2/M3. Reword both as planned (M2/M3).'
          round: 3
        - id: BR-5
          disposition: not-addressed
          note: annotate.css:17 still puts the 26 px chip at right:2px top:2px inside every box. The 50x1 px drag in annotate.test.mjs still gets a chip far larger than the box, and no rule for small boxes was added.
          round: 3
        - id: BR-7
          disposition: addressed
          note: 'viewer_annotate_live_test.go:123-160 now checks noted.Region, centres the expected pin on that box, and waits on .pin[data-note=id]. The :240 check expects kind/path and a region. Verified by reading only: Chrome would not start in this checkout (cdp EOF at createTarget). lessons.md records the sweep rule.'
          round: 3
        - id: BR-8
          disposition: addressed
          note: atlas/index.md:123 now says "crops that are the note's box plus a margin (#488)". atlas/webcast-format.md:240-241 now says the landmark is "used for the context's text but not named in an anchor's path". landmark.go:57 was updated to match. A grep for snap* finds only "retired" or "never snapped" wording.
          round: 3
      findings:
        - id: BR-9
          severity: Minor
          title: The plan still names the shared helper noteBox and its test file notes.test.mjs; the code has gestureBox and notebox.test.mjs
          detail: Plan lines 32, 41, 55, 102 and 149 say noteBox(from, to, view, min). The code's collab.js:127 is gestureBox(from, to, view, {min, cells}) plus dragMin and boxCentre, renamed under BR-4. Add a Revisions entry so M2 builds against the real name.
          family: plan-names-match-code
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: checkout
      reviewed: 9bfeae3c25ebc4e24fdcc359c19d1e0bd87997a6
      blocked: false
---

# Gate ledger — pair#488 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T11:38:18-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `docs-gate-readme-surface` README web-cast notes paragraph still says "tap an element, or drag a region" and pins "to its element (or region)"
  README.md:1017-1020. M1 changed the user-facing gesture: every note is a box, a tap gets the default 1/10 box, touch needs 16 px, and the pin is the outlined box with the chip inside its top-right corner. Update the paragraph to match atlas/webcast-format.md:408-416.
- **BR-2** [Minor] `docs-describe-unbuilt-surface` Atlas and record.go doc comment reference broadcast.NoteAnchor and terminal cell boxes, which do not exist until M2/M3
  atlas/webcast-format.md:206-209,216 and cmd/internal/webcast/recording/record.go:355 describe the terminal anchor and boxes in the present tense; grep finds no NoteAnchor type. Word them as planned, or add them when M2/M3 land.
- **BR-3** [Minor] `error-doc-drift` ErrStale doc comment not updated for the new "box off the page" refusal
  board.go:34 lists another document or a gone element; board.go:189 now also returns ErrStale when ClampToPage finds nothing on the page.
- **BR-4** [Minor] `name-collision-across-layers` Three distinct noteBox functions (collab gesture, annotate display, Go crop); annotate imports one as boxOfGesture
  Rename the collab.js gesture function (e.g. gestureBox) so each name reads as one meaning.
- **BR-5** [Minor] `ui-chip-fit` 26 px pin chip now sits inside the box and spills well outside thin boxes (e.g. a 12x1 px drag)
- **BR-6** [Minor] `glue-untested` mountAnnotate glue (pageView, pointerType capture, boxCentre to nodeAt) has no DOM-level test
  Gap predates #488; pointermode.test.mjs shows a harness that M2 could extend to annotate.

## Round 2 — 2026-10-12T11:43:08-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — README.md:1014-1022 now describes drag-or-tap-for-default-box, the touch threshold, and the outlined box with the chip inside top-right, matching annotate.css and collab.js DRAG_MIN.
- BR-2 — not-addressed — atlas:206-209 and record.go:355 fixed; atlas/webcast-format.md:217 still states terminal notes have a cell box and layout anchor in present tense (M2/M3).
- BR-3 — addressed — board.go:34-35 now lists the off-page box as an ErrStale cause, matching board.go:186-189.
- BR-4 — addressed — collab.js gesture function renamed gestureBox; annotate.js imports it unaliased; the remaining noteBox (JS display, Go crop.go) now share one meaning.
- BR-5 — not-addressed — Unchanged; annotate.css still fixes a 26 px chip at right:2px top:2px regardless of box size. Minor, non-blocking.
- BR-6 — withdrawn — A DOM-level test of mountAnnotate exists (viewer_annotate_live_test.go, Chrome-gated); the real gap is that it is stale, raised as a new finding.

### Raised

- **BR-7** [Important] `retired-surface-sweep` Chrome-gated live annotate test still asserts the retired .pin.region class and element-sized tap pins
  viewer_annotate_live_test.go:146 waits for .pin.region, which annotate.js:222 no longer sets, and :128-135 expects A's tapped pin at the element box nb, but a tap now gets the default 1/10-view box. Select .pin[data-note=id] and expect noted.Region. Rule: when retiring a class, function or field, grep all consumers, including env-gated tests and the atlas, in the same commit.
- **BR-8** [Minor] `retired-surface-sweep` atlas still mentions crop snapping after #488 retired it
  atlas/index.md:123 lists "snapped crops"; atlas/webcast-format.md:241 says the parent fallback is "not for snapping". Same sweep rule as the live-test finding.

## Round 3 — 2026-10-12T11:50:21-07:00 (claude) — passed

### Disposed

- BR-2 — not-addressed — atlas/webcast-format.md:206-209 now says the path is one "#488 M3 adds", but :217 still says in the present tense that a terminal note has "a box in cells and Couch's layout anchor", and record.go:356-357 says #488 "gives those a layout anchor". Neither exists until M2/M3. Reword both as planned (M2/M3).
- BR-5 — not-addressed — annotate.css:17 still puts the 26 px chip at right:2px top:2px inside every box. The 50x1 px drag in annotate.test.mjs still gets a chip far larger than the box, and no rule for small boxes was added.
- BR-7 — addressed — viewer_annotate_live_test.go:123-160 now checks noted.Region, centres the expected pin on that box, and waits on .pin[data-note=id]. The :240 check expects kind/path and a region. Verified by reading only: Chrome would not start in this checkout (cdp EOF at createTarget). lessons.md records the sweep rule.
- BR-8 — addressed — atlas/index.md:123 now says "crops that are the note's box plus a margin (#488)". atlas/webcast-format.md:240-241 now says the landmark is "used for the context's text but not named in an anchor's path". landmark.go:57 was updated to match. A grep for snap* finds only "retired" or "never snapped" wording.

### Raised

- **BR-9** [Minor] `plan-names-match-code` The plan still names the shared helper noteBox and its test file notes.test.mjs; the code has gestureBox and notebox.test.mjs
  Plan lines 32, 41, 55, 102 and 149 say noteBox(from, to, view, min). The code's collab.js:127 is gestureBox(from, to, view, {min, cells}) plus dragMin and boxCentre, renamed under BR-4. Add a Revisions entry so M2 builds against the real name.

## Open findings

- **BR-2** [Minor] `docs-describe-unbuilt-surface` Atlas and record.go doc comment reference broadcast.NoteAnchor and terminal cell boxes, which do not exist until M2/M3
- **BR-5** [Minor] `ui-chip-fit` 26 px pin chip now sits inside the box and spills well outside thin boxes (e.g. a 12x1 px drag)
- **BR-9** [Minor] `plan-names-match-code` The plan still names the shared helper noteBox and its test file notes.test.mjs; the code has gestureBox and notebox.test.mjs
