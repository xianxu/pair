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
    - "n": 4
      timestamp: "2026-10-12T12:21:50-07:00"
      dispatched: "2026-10-12T12:17:49-07:00"
      agent: claude
      findings:
        - id: BR-10
          severity: Critical
          title: 'ParseNote box check overflows: box [1,0,MaxInt64,1] passes within() and pinSpots spins ~9e18 iterations (viewer-triggered CPU hang)'
          detail: 'cmd/internal/broadcast/notes.go within() adds untrusted Col+W / Row+H without overflow guards; Col+W reaching 2^63 wraps negative and passes, then pinSpots starts at Col+W-2 which wraps to about MaxInt. Reproduced: ParseNote on that body still running after 3s. Fix: compare by subtraction (W <= cols-Col, H <= rows-Row) after range-checking Col/Row, or cap each component at maxPointGrid first; add the body to noteRefusals with a deadline, and rewrite the FuzzParseNote oracle so it doesn''t repeat the same overflowing addition.'
          family: bound-untrusted-ints-before-arithmetic
          round: 4
        - id: BR-11
          severity: Minor
          title: notes.js boxHasPin hand-mirrors Go pinSpots with no shared case table
          detail: A shared table of cases run by both Go and node tests would catch the two drifting apart (ARCH-DRY).
          family: cross-language-mirror-unpinned
          round: 4
        - id: BR-12
          severity: Minor
          title: pointerMode keeps gesture and stroke as two nullable fields, exclusive only because end() runs first
          detail: Collapse them into one tagged drag state (ARCH-ORDER).
          family: state-as-nullable-constellation
          round: 4
      boundary: M2
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 5
      timestamp: "2026-10-12T12:36:54-07:00"
      dispatched: "2026-10-12T12:25:24-07:00"
      agent: claude
      dispose:
        - id: BR-10
          disposition: addressed
          note: Per-component bound in ParseNote plus a subtraction-based within(); reverting both makes TestParseNoteRefusesQuickly fail at its 5 s deadline; FuzzParseNote now states its bounds without addition.
          round: 5
        - id: BR-11
          disposition: addressed
          note: tests/broadcast-viewer/pinspots.json is run by TestPinSpotsCaseTable (Go, which also checks guard == StatusGuardCols) and by notes.test.mjs (boxHasPin).
          round: 5
        - id: BR-12
          disposition: addressed
          note: viewer.js pointerMode has one tagged drag state (stroke or box), cleared only by end(); cancel and pointing-off are covered in pointermode.test.mjs.
          round: 5
      findings:
        - id: BR-13
          severity: Minor
          title: pinSpots builds the full spot list (about 1M cells) just so ParseNote can check that one exists
          detail: A 1000x1000 request allocates roughly 16-32 MB in ParseNote before landsNow rejects the mismatched grid. Use an early-exit hasPinSpot in the parser, and stop at the first free spot in Add (ARCH-CONSTRAINTS).
          family: untrusted-size-drives-work
          round: 5
        - id: BR-14
          severity: Minor
          title: The plan says the terminal box is clamped to the grid; the code refuses it with ErrNoteRange
          detail: 'This is the 2nd finding in family plan-names-match-code. The rule: every M-close adds an "as built" Revisions entry that reconciles each Design bullet the milestone touched. Add "M2 as built: off-grid boxes are refused; the pin is placed by NoteBoard.Add via pinSpots."'
          family: plan-names-match-code
          round: 5
      boundary: M2
      recipe: milestone-review
      isolation: checkout
      reviewed: f622966e2e280698d0a8a3b0d75d8afec3ec38a8
      blocked: false
    - "n": 6
      timestamp: "2026-10-12T12:55:39-07:00"
      dispatched: "2026-10-12T12:48:35-07:00"
      agent: claude
      findings:
        - id: BR-15
          severity: Critical
          title: Resolver's tried set is keyed by note id only; ids restart per broadcast session, so reused ids never get a pane path
          detail: console_noteanchor.go:134-151. A new Session builds a new PointerState and NoteBoard (nextID 0). tried is pruned only in queueAnchors, which applyNotes skips once pointing is off or the broadcast ends. In a second broadcast, note 1 stays kind screen. Key tried by (session, id) or reset it on session change, and add a two-session test. Fix the "ids are never reused" claims in the atlas and SetNoteAnchor.
          family: identity-scoped-to-owner
          round: 6
        - id: BR-16
          severity: Important
          title: Resolver uses the live frame at resolve time; a slot switch mid-pass goes undetected and draft lines come from another slot
          detail: 'This is the 2nd finding in this family. Rule: a job carries the exact observation it was made from (here the frame and its EndpointID), never a re-read of mutable "latest" state. GeometryEpoch is per endpoint and slots resize together, so the epoch and geometry check passes across a slot switch. The header comment claims a slot check that happens only at queue time. Capture the frame in anchorJob, check EndpointID, and add a slot-switch-mid-pass test.'
          family: state-as-nullable-constellation
          round: 6
        - id: BR-17
          severity: Important
          title: queueAnchors calls GoTracked while holding anchors.mu; GoTracked takes c.mu and, when not started, runs resolveAnchors inline (self-deadlock)
          detail: console_noteanchor.go:153-155 against console.go:2139. This breaks the documented leaf-lock rule. It is latent today because the inline path needs !started with a live broadcast. Decide the start under the lock, unlock, then call GoTracked.
          family: leaf-lock-held-across-calls
          round: 6
        - id: BR-18
          severity: Important
          title: Note.Anchor (file names, an on-disk Markdown heading, text under the box) now goes out on every page's notes event
          detail: notes.go json tag "anchor,omitempty" and server.go:346 wireNotes. The spec says the anchor is for the agent's context, and no page reads it. sectionOf can publish a heading that is not on screen to view-only viewers. Exclude it from the wire (json:"-" or a wire struct), keep it in Session.Notes(), and add a test that the wire has no anchor.
          family: operator-context-on-public-wire
          round: 6
        - id: BR-19
          severity: Important
          title: Session.SetNoteAnchor, NoteBoard.SetAnchor and Config.AnchorNote have no broadcast-package tests; drop paths are tested only through a fake that restates them
          detail: 'This is the 2nd finding in family glue-untested. Rule: each exported state-changing seam method a milestone adds gets an own-package test of its refuse and drop paths (here: after resolve, after a flip to off, and on an unknown id), not only through a caller''s fake.'
          family: glue-untested
          round: 6
        - id: BR-20
          severity: Minor
          title: noteanchor.go header names noteanchor_resolve.go (nonexistent); the plan's Design still names noteanchor_io.go, AnchorFor(box, l, frame) and view.json fuzzing
          detail: 'This is the 3rd finding in family plan-names-match-code. Rule: every file or identifier named in prose (plan, atlas, doc comment) must exist at the pinned head. Enforce it with a grep of the diff''s backticked names against the tree at each boundary, rather than fixing instances. Prevalence: 3 findings across M1-M3.'
          family: plan-names-match-code
          round: 6
        - id: BR-21
          severity: Minor
          title: reviewFile restates SpawnReviewPane's argv format ("/nvim/review.lua" followed by the file) with no shared constant or cross-test
          detail: note_panes.go:117. If reviewcmd's command changes, review anchors silently lose their file. Share the marker constant, or add a test that builds SpawnReviewPane's argv and parses it back (ARCH-DRY).
          family: cross-component-format-unpinned
          round: 6
      boundary: M3
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 7
      timestamp: "2026-10-12T13:23:29-07:00"
      dispatched: "2026-10-12T12:59:34-07:00"
      agent: claude
      dispose:
        - id: BR-15
          disposition: addressed
          note: tried is keyed by triedNote{session,id}; reverting the key to id-only turns TestQueueAnchorJobsPerSession red; atlas and SetNoteAnchor wording corrected.
          round: 7
        - id: BR-16
          disposition: addressed
          note: anchorJob carries frame; apply checks EndpointID/epoch/geometry against it; removing the EndpointID check turns the slot-switch subtest red.
          round: 7
        - id: BR-17
          disposition: addressed
          note: queueAnchorJobs decides under anchors.mu and returns a bool; queueAnchors calls GoTracked after unlock; calling GoTracked under the lock deadlocks TestQueueAnchorJobsPerSession (20s timeout).
          round: 7
        - id: BR-18
          disposition: addressed
          note: Note.Anchor is json:"-"; restoring the omitempty tag turns TestNoteAnchorLifecycle red (anchor shows up in the notes event).
          round: 7
        - id: BR-19
          disposition: addressed
          note: 'TestNoteAnchorLifecycle (session: unknown id, resolved, pointing off) and TestNoteBoardSetAnchor (board) cover the refuse and drop paths in their own packages.'
          round: 7
        - id: BR-20
          disposition: not-addressed
          note: noteanchor.go header fixed, but plan Design lines 65/76 still name noteanchor_io.go and the AnchorFor(box,l,frame) order; as-built adds no mapping and says tried is pruned to open ids; anchorJob comment says it carries a geometry epoch.
          round: 7
        - id: BR-21
          disposition: addressed
          note: reviewcmd.ReviewScript is shared by RunOpen and reviewFile; TestReviewFile parses a command built from the constant.
          round: 7
      findings:
        - id: BR-22
          severity: Minor
          title: tried keeps a pointer to an ended broadcast Session until the next broadcast's first notes snapshot
          detail: console_noteanchor.go:155 prunes only in queueAnchorJobs, which applyNotes skips once the broadcast ends. Size is bounded by that session's open notes; clear tried (and pending) when the broadcast ends, or accept it and note the bound in the plan.
          family: residue-outlives-owner
          round: 7
      boundary: M3
      recipe: milestone-review
      isolation: checkout
      reviewed: e1dac86e0588bac2b5d38bc86c73f866f89e602e
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

## Round 4 — 2026-10-12T12:21:50-07:00 (claude) — BLOCKED

### Raised

- **BR-10** [Critical] `bound-untrusted-ints-before-arithmetic` ParseNote box check overflows: box [1,0,MaxInt64,1] passes within() and pinSpots spins ~9e18 iterations (viewer-triggered CPU hang)
  cmd/internal/broadcast/notes.go within() adds untrusted Col+W / Row+H without overflow guards; Col+W reaching 2^63 wraps negative and passes, then pinSpots starts at Col+W-2 which wraps to about MaxInt. Reproduced: ParseNote on that body still running after 3s. Fix: compare by subtraction (W <= cols-Col, H <= rows-Row) after range-checking Col/Row, or cap each component at maxPointGrid first; add the body to noteRefusals with a deadline, and rewrite the FuzzParseNote oracle so it doesn't repeat the same overflowing addition.
- **BR-11** [Minor] `cross-language-mirror-unpinned` notes.js boxHasPin hand-mirrors Go pinSpots with no shared case table
  A shared table of cases run by both Go and node tests would catch the two drifting apart (ARCH-DRY).
- **BR-12** [Minor] `state-as-nullable-constellation` pointerMode keeps gesture and stroke as two nullable fields, exclusive only because end() runs first
  Collapse them into one tagged drag state (ARCH-ORDER).

## Round 5 — 2026-10-12T12:36:54-07:00 (claude) — passed

### Disposed

- BR-10 — addressed — Per-component bound in ParseNote plus a subtraction-based within(); reverting both makes TestParseNoteRefusesQuickly fail at its 5 s deadline; FuzzParseNote now states its bounds without addition.
- BR-11 — addressed — tests/broadcast-viewer/pinspots.json is run by TestPinSpotsCaseTable (Go, which also checks guard == StatusGuardCols) and by notes.test.mjs (boxHasPin).
- BR-12 — addressed — viewer.js pointerMode has one tagged drag state (stroke or box), cleared only by end(); cancel and pointing-off are covered in pointermode.test.mjs.

### Raised

- **BR-13** [Minor] `untrusted-size-drives-work` pinSpots builds the full spot list (about 1M cells) just so ParseNote can check that one exists
  A 1000x1000 request allocates roughly 16-32 MB in ParseNote before landsNow rejects the mismatched grid. Use an early-exit hasPinSpot in the parser, and stop at the first free spot in Add (ARCH-CONSTRAINTS).
- **BR-14** [Minor] `plan-names-match-code` The plan says the terminal box is clamped to the grid; the code refuses it with ErrNoteRange
  This is the 2nd finding in family plan-names-match-code. The rule: every M-close adds an "as built" Revisions entry that reconciles each Design bullet the milestone touched. Add "M2 as built: off-grid boxes are refused; the pin is placed by NoteBoard.Add via pinSpots."

## Round 6 — 2026-10-12T12:55:39-07:00 (claude) — BLOCKED

### Raised

- **BR-15** [Critical] `identity-scoped-to-owner` Resolver's tried set is keyed by note id only; ids restart per broadcast session, so reused ids never get a pane path
  console_noteanchor.go:134-151. A new Session builds a new PointerState and NoteBoard (nextID 0). tried is pruned only in queueAnchors, which applyNotes skips once pointing is off or the broadcast ends. In a second broadcast, note 1 stays kind screen. Key tried by (session, id) or reset it on session change, and add a two-session test. Fix the "ids are never reused" claims in the atlas and SetNoteAnchor.
- **BR-16** [Important] `state-as-nullable-constellation` Resolver uses the live frame at resolve time; a slot switch mid-pass goes undetected and draft lines come from another slot
  This is the 2nd finding in this family. Rule: a job carries the exact observation it was made from (here the frame and its EndpointID), never a re-read of mutable "latest" state. GeometryEpoch is per endpoint and slots resize together, so the epoch and geometry check passes across a slot switch. The header comment claims a slot check that happens only at queue time. Capture the frame in anchorJob, check EndpointID, and add a slot-switch-mid-pass test.
- **BR-17** [Important] `leaf-lock-held-across-calls` queueAnchors calls GoTracked while holding anchors.mu; GoTracked takes c.mu and, when not started, runs resolveAnchors inline (self-deadlock)
  console_noteanchor.go:153-155 against console.go:2139. This breaks the documented leaf-lock rule. It is latent today because the inline path needs !started with a live broadcast. Decide the start under the lock, unlock, then call GoTracked.
- **BR-18** [Important] `operator-context-on-public-wire` Note.Anchor (file names, an on-disk Markdown heading, text under the box) now goes out on every page's notes event
  notes.go json tag "anchor,omitempty" and server.go:346 wireNotes. The spec says the anchor is for the agent's context, and no page reads it. sectionOf can publish a heading that is not on screen to view-only viewers. Exclude it from the wire (json:"-" or a wire struct), keep it in Session.Notes(), and add a test that the wire has no anchor.
- **BR-19** [Important] `glue-untested` Session.SetNoteAnchor, NoteBoard.SetAnchor and Config.AnchorNote have no broadcast-package tests; drop paths are tested only through a fake that restates them
  This is the 2nd finding in family glue-untested. Rule: each exported state-changing seam method a milestone adds gets an own-package test of its refuse and drop paths (here: after resolve, after a flip to off, and on an unknown id), not only through a caller's fake.
- **BR-20** [Minor] `plan-names-match-code` noteanchor.go header names noteanchor_resolve.go (nonexistent); the plan's Design still names noteanchor_io.go, AnchorFor(box, l, frame) and view.json fuzzing
  This is the 3rd finding in family plan-names-match-code. Rule: every file or identifier named in prose (plan, atlas, doc comment) must exist at the pinned head. Enforce it with a grep of the diff's backticked names against the tree at each boundary, rather than fixing instances. Prevalence: 3 findings across M1-M3.
- **BR-21** [Minor] `cross-component-format-unpinned` reviewFile restates SpawnReviewPane's argv format ("/nvim/review.lua" followed by the file) with no shared constant or cross-test
  note_panes.go:117. If reviewcmd's command changes, review anchors silently lose their file. Share the marker constant, or add a test that builds SpawnReviewPane's argv and parses it back (ARCH-DRY).

## Round 7 — 2026-10-12T13:23:29-07:00 (claude) — passed

### Disposed

- BR-15 — addressed — tried is keyed by triedNote{session,id}; reverting the key to id-only turns TestQueueAnchorJobsPerSession red; atlas and SetNoteAnchor wording corrected.
- BR-16 — addressed — anchorJob carries frame; apply checks EndpointID/epoch/geometry against it; removing the EndpointID check turns the slot-switch subtest red.
- BR-17 — addressed — queueAnchorJobs decides under anchors.mu and returns a bool; queueAnchors calls GoTracked after unlock; calling GoTracked under the lock deadlocks TestQueueAnchorJobsPerSession (20s timeout).
- BR-18 — addressed — Note.Anchor is json:"-"; restoring the omitempty tag turns TestNoteAnchorLifecycle red (anchor shows up in the notes event).
- BR-19 — addressed — TestNoteAnchorLifecycle (session: unknown id, resolved, pointing off) and TestNoteBoardSetAnchor (board) cover the refuse and drop paths in their own packages.
- BR-20 — not-addressed — noteanchor.go header fixed, but plan Design lines 65/76 still name noteanchor_io.go and the AnchorFor(box,l,frame) order; as-built adds no mapping and says tried is pruned to open ids; anchorJob comment says it carries a geometry epoch.
- BR-21 — addressed — reviewcmd.ReviewScript is shared by RunOpen and reviewFile; TestReviewFile parses a command built from the constant.

### Raised

- **BR-22** [Minor] `residue-outlives-owner` tried keeps a pointer to an ended broadcast Session until the next broadcast's first notes snapshot
  console_noteanchor.go:155 prunes only in queueAnchorJobs, which applyNotes skips once the broadcast ends. Size is bounded by that session's open notes; clear tried (and pending) when the broadcast ends, or accept it and note the bound in the plan.

## Open findings

- **BR-2** [Minor] `docs-describe-unbuilt-surface` Atlas and record.go doc comment reference broadcast.NoteAnchor and terminal cell boxes, which do not exist until M2/M3
- **BR-5** [Minor] `ui-chip-fit` 26 px pin chip now sits inside the box and spills well outside thin boxes (e.g. a 12x1 px drag)
- **BR-9** [Minor] `plan-names-match-code` The plan still names the shared helper noteBox and its test file notes.test.mjs; the code has gestureBox and notebox.test.mjs
- **BR-13** [Minor] `untrusted-size-drives-work` pinSpots builds the full spot list (about 1M cells) just so ParseNote can check that one exists
- **BR-14** [Minor] `plan-names-match-code` The plan says the terminal box is clamped to the grid; the code refuses it with ErrNoteRange
- **BR-20** [Minor] `plan-names-match-code` noteanchor.go header names noteanchor_resolve.go (nonexistent); the plan's Design still names noteanchor_io.go, AnchorFor(box, l, frame) and view.json fuzzing
- **BR-22** [Minor] `residue-outlives-owner` tried keeps a pointer to an ended broadcast Session until the next broadcast's first notes snapshot
