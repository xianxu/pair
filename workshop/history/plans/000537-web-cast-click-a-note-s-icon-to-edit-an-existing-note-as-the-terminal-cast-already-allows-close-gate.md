---
gate: boundary-review
issue: 537
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-13T17:12:48-07:00"
      dispatched: "2026-10-13T17:08:43-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Successful draft creation remains eligible to create another note
          detail: 'cmd/internal/broadcast/web/collab.js:245-249 retains draft mode when input changes before a successful save reply. Reproducing open-draft, input, save, input, result(204), save emits save-new again: the web viewer creates a second note, while the operator rejects its consumed draft at cmd/internal/webcast/recorder/operator.go:259. ARCH-ORDER: preserve newer text through an explicit committed-note transition or prevent draft input during creation, and add delayed-success regression coverage.'
          family: committed-operation-state
          round: 1
        - id: BR-2
          severity: Critical
          title: Core concept tables disagree with implementation locations and omit classifications
          detail: 'workshop/plans/000537-web-note-editing-plan.md:19-22,33-39,200 maps edit entities to wire.go, notes.go, board.go, pointer_notes.go and session.go although their definitions are in the new note_edit.go files; NoteResolution is in note_resolution.go. ARCH-PURPOSE: reconcile every table row, add explicit PURE/INTEGRATION classifications, and record the correction under Revisions. Severity follows the required Core concepts contradiction gate.'
          family: core-concept-traceability
          round: 1
        - id: BR-3
          severity: Important
          title: README update is missing for existing-note editing
          detail: README.md:1047 and :1147 still describe pin clicks as read/resolve interactions, and README.md is unchanged in the pinned range. Document both casts and the operator overlay, Save/Cancel, edited markers, and the approved session-bound ownership lifetime; atlas additions alone do not satisfy the README gate.
          family: user-surface-documentation
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-13T17:18:42-07:00"
      dispatched: "2026-10-13T17:15:53-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: collab.js:233,271 blocks draft input during creation. Four regressions at tests/broadcast-viewer/notes.test.mjs:330-364 pass at HEAD and fail when the fix is removed in a disposable copy, which was subsequently deleted.
          round: 2
        - id: BR-2
          disposition: addressed
          note: workshop/plans/000537-web-note-editing-plan.md:290 explicitly supersedes the earlier inventories with corrected definition paths and PURE/INTEGRATION classifications. Checked the replacement rows against source, including both note_edit.go files and note_resolution.go.
          round: 2
        - id: BR-3
          disposition: addressed
          note: README.md:1053 and :1168 now document both casts, operator ownership, Save/Cancel, edited markers and session lifetime. These descriptions agree with cookie-derived ownership, Board/operator admission and the shared dialog.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 8d1719f3363b6039a67043f5f9af9d603930ebbb
      blocked: false
---

# Gate ledger — pair#537 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T17:12:48-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `committed-operation-state` Successful draft creation remains eligible to create another note
  cmd/internal/broadcast/web/collab.js:245-249 retains draft mode when input changes before a successful save reply. Reproducing open-draft, input, save, input, result(204), save emits save-new again: the web viewer creates a second note, while the operator rejects its consumed draft at cmd/internal/webcast/recorder/operator.go:259. ARCH-ORDER: preserve newer text through an explicit committed-note transition or prevent draft input during creation, and add delayed-success regression coverage.
- **BR-2** [Critical] `core-concept-traceability` Core concept tables disagree with implementation locations and omit classifications
  workshop/plans/000537-web-note-editing-plan.md:19-22,33-39,200 maps edit entities to wire.go, notes.go, board.go, pointer_notes.go and session.go although their definitions are in the new note_edit.go files; NoteResolution is in note_resolution.go. ARCH-PURPOSE: reconcile every table row, add explicit PURE/INTEGRATION classifications, and record the correction under Revisions. Severity follows the required Core concepts contradiction gate.
- **BR-3** [Important] `user-surface-documentation` README update is missing for existing-note editing
  README.md:1047 and :1147 still describe pin clicks as read/resolve interactions, and README.md is unchanged in the pinned range. Document both casts and the operator overlay, Save/Cancel, edited markers, and the approved session-bound ownership lifetime; atlas additions alone do not satisfy the README gate.

## Round 2 — 2026-10-13T17:18:42-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — collab.js:233,271 blocks draft input during creation. Four regressions at tests/broadcast-viewer/notes.test.mjs:330-364 pass at HEAD and fail when the fix is removed in a disposable copy, which was subsequently deleted.
- BR-2 — addressed — workshop/plans/000537-web-note-editing-plan.md:290 explicitly supersedes the earlier inventories with corrected definition paths and PURE/INTEGRATION classifications. Checked the replacement rows against source, including both note_edit.go files and note_resolution.go.
- BR-3 — addressed — README.md:1053 and :1168 now document both casts, operator ownership, Save/Cancel, edited markers and session lifetime. These descriptions agree with cookie-derived ownership, Board/operator admission and the shared dialog.

## Open findings

(none — every finding has been disposed)
