---
gate: boundary-review
issue: 447
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T14:10:32-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Task 3 Step 3 says to blank cells with uv.EmptyCell; Core concepts says EmptyCell drops the background and blanking must keep style
          detail: |-
            2nd finding in this family. Rule: a task step names what to build and points to its Core concepts entry for how, without restating the mechanism. Follow Core concepts (set content " " and width 1, keep the style) and drop the uv.EmptyCell wording in Task 3.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: plan-self-consistency
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-10-11T14:10:32-07:00"
      dispatched: "2026-10-11T14:00:03-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: "README update appears missing for the \U0001F4ADN tab-bar count and notes cleared on pointing off"
          detail: "README.md:876-891 documents the live status row as LIVE ⏸ \U0001F446 \U0001F47D; M1 adds an operator-visible \U0001F4ADN after \U0001F47D (reserve.go:227-230) and deletes all notes when pointing turns off. Add a sentence or two to the Remote pointer paragraph."
          family: docs-gate-user-surface
          round: 2
        - id: BR-3
          severity: Minor
          title: ParsePointBatch duplicates the strict-decode logic now in decodeStrict (ARCH-DRY)
          detail: point.go:52-58 repeats DisallowUnknownFields plus the trailing-token check that notes.go:140 decodeStrict provides; have ParsePointBatch call decodeStrict.
          family: shared-helper-not-reused
          round: 2
        - id: BR-4
          severity: Minor
          title: forgetNotes at broadcast end is not exercised with notes held
          detail: TestNotesIgnoredAfterBroadcastEndsOrFromAnotherSession stops a broadcast that holds no notes, so plan Task 6's count-is-0-after-stop case with notes present is not tested.
          family: untested-cleanup-path
          round: 2
        - id: BR-5
          severity: Minor
          title: Per-color note limit uses a client-supplied color, so one viewer can use all six colors
          detail: ParseNote accepts any color 0-5 that is not tied to the stream's claimed color; the 48-note total cap and the rate limit still hold, and the setting is collaborative by design.
          family: client-asserted-identity
          round: 2
        - id: BR-6
          severity: Minor
          title: drawnNotes comment says empty unless pointing is on, but the function does not check
          detail: This holds only because every pointer-phase change resyncs the snapshot (console_notes.go); reword the comment to say that.
          family: comment-claims-unenforced-invariant
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 3
      timestamp: "2026-10-11T14:22:15-07:00"
      dispatched: "2026-10-11T14:12:51-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan line 147 still says "f.Clone() and uv.EmptyCell for blanks"; the code is correct (notes.go:205 blank keeps style), only the plan step is stale.
          round: 3
        - id: BR-2
          disposition: addressed
          note: "README.md:894-901 adds the Sticky notes paragraph covering the \U0001F4ADN count after the controls and that pointing off deletes notes; matches reserve.go:227-230 and pointer.go:103."
          round: 3
        - id: BR-3
          disposition: addressed
          note: point.go:50 now calls decodeStrict; the broadcast point tests pass unchanged.
          round: 3
        - id: BR-4
          disposition: addressed
          note: TestNotesForgottenWhenBroadcastEnds (console_notes_test.go:113) fails when the forgetNotes call in resetPointer is removed ("console still holds 1 notes (v1)"), so the fix is reachable.
          round: 3
        - id: BR-5
          disposition: withdrawn
          note: 'Consistent with the spec''s trust model: the pointer link is a shared capability, point colors (#446) are client-asserted too, and the 48-note total cap plus rate limit are the real bound.'
          round: 3
        - id: BR-6
          disposition: addressed
          note: console_notes.go:78-80 comment now says emptiness relies on the resync at each pointer-phase change; applyNotes also gates on pointer==pointerOn.
          round: 3
      findings:
        - id: BR-7
          severity: Minor
          title: README describes viewer-page note actions that only land in M2
          detail: "README.md:894-901 says pointer-link viewers can pin notes and anyone can tap a \U0001F4AD to read it, but M1 ships no viewer UI (notes are curl-only until M2 Tasks 8-10). This is the 2nd finding in this family; the rule: README changes describe only the surface present at the boundary's head, so tag M2-only behavior as upcoming or add it with M2."
          family: docs-gate-user-surface
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: checkout
      reviewed: 1c3a0b160976359ea6565092c6ff5dd8017f3feb
      blocked: false
    - "n": 4
      timestamp: "2026-10-11T15:12:41-07:00"
      dispatched: "2026-10-11T15:09:56-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan line 147 (Task 3 Step 3) still reads "using f.Clone() and uv.EmptyCell for blanks"; code blank() at notes.go:208 keeps style, so only the plan text remains wrong.
          round: 4
      findings:
        - id: BR-8
          severity: Important
          title: Viewer keeps notes, pointing-on, mode button and open dialog after event end (viewer.js:200-206, 380-395)
          detail: deps.reset only resets the terminal; server.go selects end/caps/notes in random order, so stale notes stay tappable (with Resolve) on the blank ended screen and comment mode still drafts. Reset all stream-derived state (notes list, setOn(false)) on the terminal event and add a test. ARCH-ORDER.
          family: stream-end-resets-derived-state
          round: 4
        - id: BR-9
          severity: Minor
          title: noteDialog applies a save/resolve response to whatever dialog is open now, not the one that sent it (notes.js:74-81)
          detail: 'Cancel during an in-flight save and open a new draft: the old 204 closes the new draft (text lost), or an old 409 shows on it. Capture a token per request and drop mismatched completions. ARCH-ORDER.'
          family: late-completion-identity
          round: 4
        - id: BR-10
          severity: Minor
          title: "README Sticky notes paragraph never says how a viewer places a note (the \U0001F446/\U0001F4AD mode button)"
          detail: '3rd finding in this family. Rule: the README feature paragraph names every control a person uses for the feature, and each milestone that adds or removes a control updates it; fold into M3''s README task (hint bar removal, tooltips).'
          family: docs-gate-user-surface
          round: 4
        - id: BR-11
          severity: Minor
          title: Escape does not close a viewed note; aria-modal dialog has no focus move or trap in view mode
          detail: 'The keydown listener is on #note, and only a draft moves focus into it (notes.js:89, draft focuses input).'
          family: dialog-keyboard-reach
          round: 4
        - id: BR-12
          severity: Minor
          title: Plan Task 8 gives noteAt(notes, col, row, cols, rows, guard); code takes noteAt(list, col, row, cols, rows) with list.guard
          detail: '2nd in family. Rule: task steps cite Core concepts for signatures rather than restating them, which also covers BR-1; update Task 8 to match.'
          family: plan-self-consistency
          round: 4
      boundary: M2
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 5
      timestamp: "2026-10-11T15:15:54-07:00"
      dispatched: "2026-10-11T15:14:12-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: plan.md:147 Task 3 Step 3 still reads "using f.Clone() and uv.EmptyCell for blanks", contradicting Core concepts line 37.
          round: 5
        - id: BR-8
          disposition: addressed
          note: viewer.js:208-209 delivers caps-off and an empty notes list on end; connect.test.mjs BR-8 test fails with the pre-fix viewer.js (verified by revert). New view-link side effect raised separately.
          round: 5
        - id: BR-9
          disposition: addressed
          note: notes.js:62-91 generation token; notes.test.mjs "an answer applies only to the dialog that sent it" fails on revert (verified).
          round: 5
        - id: BR-10
          disposition: addressed
          note: "README.md:895-896 now names the \U0001F446/\U0001F4AD mode button under the swatches."
          round: 5
        - id: BR-11
          disposition: addressed
          note: notes.js:141 focuses Close on view; test fails on revert. No focus trap yet, acceptable at Minor.
          round: 5
        - id: BR-12
          disposition: addressed
          note: plan.md:232 Task 8 now cites Core concepts for the noteAt signature instead of restating it.
          round: 5
      findings:
        - id: BR-13
          severity: Minor
          title: End sends a made-up caps-off to view links, which never get caps, so they show the "Pointing is off" hint band
          detail: 'viewer.js:208 calls deps.caps({pointer:false}) on every page. server.go:325-340 sends caps only to pointer pages, so on a view link setOn(false) un-hides #hint and band() lifts the stage after the end. 2nd in family. Rule: an end puts stream-derived state back to the page variant''s join state and never replays events that variant does not receive. Fix: only send caps-off when a caps event was seen (or use one ended() dep), and add a view-link end test.'
          family: stream-end-resets-derived-state
          round: 5
      boundary: M2
      recipe: milestone-review
      isolation: checkout
      reviewed: 706a32676d10c63478c5fdae9f61984e62812423
      blocked: false
---

# Gate ledger — pair#447 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T14:10:32-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `plan-self-consistency` Task 3 Step 3 says to blank cells with uv.EmptyCell; Core concepts says EmptyCell drops the background and blanking must keep style
  2nd finding in this family. Rule: a task step names what to build and points to its Core concepts entry for how, without restating the mechanism. Follow Core concepts (set content " " and width 1, keep the style) and drop the uv.EmptyCell wording in Task 3.
  (carried from plan-quality PQ-7, deferred to the boundary review)

## Round 2 — 2026-10-11T14:10:32-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `docs-gate-user-surface` README update appears missing for the 💭N tab-bar count and notes cleared on pointing off
  README.md:876-891 documents the live status row as LIVE ⏸ 👆 👽; M1 adds an operator-visible 💭N after 👽 (reserve.go:227-230) and deletes all notes when pointing turns off. Add a sentence or two to the Remote pointer paragraph.
- **BR-3** [Minor] `shared-helper-not-reused` ParsePointBatch duplicates the strict-decode logic now in decodeStrict (ARCH-DRY)
  point.go:52-58 repeats DisallowUnknownFields plus the trailing-token check that notes.go:140 decodeStrict provides; have ParsePointBatch call decodeStrict.
- **BR-4** [Minor] `untested-cleanup-path` forgetNotes at broadcast end is not exercised with notes held
  TestNotesIgnoredAfterBroadcastEndsOrFromAnotherSession stops a broadcast that holds no notes, so plan Task 6's count-is-0-after-stop case with notes present is not tested.
- **BR-5** [Minor] `client-asserted-identity` Per-color note limit uses a client-supplied color, so one viewer can use all six colors
  ParseNote accepts any color 0-5 that is not tied to the stream's claimed color; the 48-note total cap and the rate limit still hold, and the setting is collaborative by design.
- **BR-6** [Minor] `comment-claims-unenforced-invariant` drawnNotes comment says empty unless pointing is on, but the function does not check
  This holds only because every pointer-phase change resyncs the snapshot (console_notes.go); reword the comment to say that.

## Round 3 — 2026-10-11T14:22:15-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Plan line 147 still says "f.Clone() and uv.EmptyCell for blanks"; the code is correct (notes.go:205 blank keeps style), only the plan step is stale.
- BR-2 — addressed — README.md:894-901 adds the Sticky notes paragraph covering the 💭N count after the controls and that pointing off deletes notes; matches reserve.go:227-230 and pointer.go:103.
- BR-3 — addressed — point.go:50 now calls decodeStrict; the broadcast point tests pass unchanged.
- BR-4 — addressed — TestNotesForgottenWhenBroadcastEnds (console_notes_test.go:113) fails when the forgetNotes call in resetPointer is removed ("console still holds 1 notes (v1)"), so the fix is reachable.
- BR-5 — withdrawn — Consistent with the spec's trust model: the pointer link is a shared capability, point colors (#446) are client-asserted too, and the 48-note total cap plus rate limit are the real bound.
- BR-6 — addressed — console_notes.go:78-80 comment now says emptiness relies on the resync at each pointer-phase change; applyNotes also gates on pointer==pointerOn.

### Raised

- **BR-7** [Minor] `docs-gate-user-surface` README describes viewer-page note actions that only land in M2
  README.md:894-901 says pointer-link viewers can pin notes and anyone can tap a 💭 to read it, but M1 ships no viewer UI (notes are curl-only until M2 Tasks 8-10). This is the 2nd finding in this family; the rule: README changes describe only the surface present at the boundary's head, so tag M2-only behavior as upcoming or add it with M2.

## Round 4 — 2026-10-11T15:12:41-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Plan line 147 (Task 3 Step 3) still reads "using f.Clone() and uv.EmptyCell for blanks"; code blank() at notes.go:208 keeps style, so only the plan text remains wrong.

### Raised

- **BR-8** [Important] `stream-end-resets-derived-state` Viewer keeps notes, pointing-on, mode button and open dialog after event end (viewer.js:200-206, 380-395)
  deps.reset only resets the terminal; server.go selects end/caps/notes in random order, so stale notes stay tappable (with Resolve) on the blank ended screen and comment mode still drafts. Reset all stream-derived state (notes list, setOn(false)) on the terminal event and add a test. ARCH-ORDER.
- **BR-9** [Minor] `late-completion-identity` noteDialog applies a save/resolve response to whatever dialog is open now, not the one that sent it (notes.js:74-81)
  Cancel during an in-flight save and open a new draft: the old 204 closes the new draft (text lost), or an old 409 shows on it. Capture a token per request and drop mismatched completions. ARCH-ORDER.
- **BR-10** [Minor] `docs-gate-user-surface` README Sticky notes paragraph never says how a viewer places a note (the 👆/💭 mode button)
  3rd finding in this family. Rule: the README feature paragraph names every control a person uses for the feature, and each milestone that adds or removes a control updates it; fold into M3's README task (hint bar removal, tooltips).
- **BR-11** [Minor] `dialog-keyboard-reach` Escape does not close a viewed note; aria-modal dialog has no focus move or trap in view mode
  The keydown listener is on #note, and only a draft moves focus into it (notes.js:89, draft focuses input).
- **BR-12** [Minor] `plan-self-consistency` Plan Task 8 gives noteAt(notes, col, row, cols, rows, guard); code takes noteAt(list, col, row, cols, rows) with list.guard
  2nd in family. Rule: task steps cite Core concepts for signatures rather than restating them, which also covers BR-1; update Task 8 to match.

## Round 5 — 2026-10-11T15:15:54-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — plan.md:147 Task 3 Step 3 still reads "using f.Clone() and uv.EmptyCell for blanks", contradicting Core concepts line 37.
- BR-8 — addressed — viewer.js:208-209 delivers caps-off and an empty notes list on end; connect.test.mjs BR-8 test fails with the pre-fix viewer.js (verified by revert). New view-link side effect raised separately.
- BR-9 — addressed — notes.js:62-91 generation token; notes.test.mjs "an answer applies only to the dialog that sent it" fails on revert (verified).
- BR-10 — addressed — README.md:895-896 now names the 👆/💭 mode button under the swatches.
- BR-11 — addressed — notes.js:141 focuses Close on view; test fails on revert. No focus trap yet, acceptable at Minor.
- BR-12 — addressed — plan.md:232 Task 8 now cites Core concepts for the noteAt signature instead of restating it.

### Raised

- **BR-13** [Minor] `stream-end-resets-derived-state` End sends a made-up caps-off to view links, which never get caps, so they show the "Pointing is off" hint band
  viewer.js:208 calls deps.caps({pointer:false}) on every page. server.go:325-340 sends caps only to pointer pages, so on a view link setOn(false) un-hides #hint and band() lifts the stage after the end. 2nd in family. Rule: an end puts stream-derived state back to the page variant's join state and never replays events that variant does not receive. Fix: only send caps-off when a caps event was seen (or use one ended() dep), and add a view-link end test.

## Open findings

- **BR-1** [Minor] `plan-self-consistency` Task 3 Step 3 says to blank cells with uv.EmptyCell; Core concepts says EmptyCell drops the background and blanking must keep style
- **BR-7** [Minor] `docs-gate-user-surface` README describes viewer-page note actions that only land in M2
- **BR-13** [Minor] `stream-end-resets-derived-state` End sends a made-up caps-off to view links, which never get caps, so they show the "Pointing is off" hint band
