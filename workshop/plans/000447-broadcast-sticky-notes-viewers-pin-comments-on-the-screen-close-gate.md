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

## Open findings

- **BR-1** [Minor] `plan-self-consistency` Task 3 Step 3 says to blank cells with uv.EmptyCell; Core concepts says EmptyCell drops the background and blanking must keep style
- **BR-7** [Minor] `docs-gate-user-surface` README describes viewer-page note actions that only land in M2
