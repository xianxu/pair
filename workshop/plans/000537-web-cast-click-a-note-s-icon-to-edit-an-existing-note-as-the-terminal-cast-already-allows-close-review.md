# Boundary Review — pair#537 (whole-issue close)

| field | value |
|-------|-------|
| issue | 537 — web cast: click a note's icon to edit an existing note (as the terminal cast already allows) |
| repo | pair |
| issue file | workshop/issues/000537-web-cast-click-a-note-s-icon-to-edit-an-existing-note-as-the-terminal-cast-already-allows.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3a0355cb8ec51e3720c550a0dcfd6b038b7bd009..74f84ee72f674d0d352e9aa19546fa20760aa144 |
| command | sdlc close --issue 537 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T17:12:48-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The core editing paths are well covered: ownership comes from validated sessions, edits preserve creation records, and recording order survives file merging. A shared-dialog regression blocks shipping: typing during a pending new-note save leaves the successful draft able to create again. The plan’s entity map also contradicts the implementation, and README usage guidance is missing.

1. **Strengths**

   - Cookie ownership is enforced server-side, including deterministic revocation-during-body-read tests on both casts.
   - Immutable edit reduction preserves authorship and anchoring while invalidating submission revision guards.
   - Frozen-clock tests exercise actual split/merge/reduction across pause, end, navigation and clear.
   - Operator edit acknowledgments correctly avoid treating an existing target note as proof of success.

2. **Critical findings**

   - **Successful creation remains in creation mode — ARCH-ORDER.** `cmd/internal/broadcast/web/collab.js:245–249`: create a draft, Save, type newer text before the 204 arrives, then Save again. The production reducer emits another `save-new`. The web viewer creates a second note; the operator retries a consumed draft and receives `ErrStale` (`recorder/operator.go:259`). Preserve the newer text while transitioning to an explicit committed-note state, or prevent draft input during creation. Add a delayed-success regression covering the subsequent Save.
   - **Core concepts contradict their declared locations — ARCH-PURPOSE.** `workshop/plans/000537-web-note-editing-plan.md:19–22,33–39,200`: edit parsers, requests and admission methods actually live in the new `note_edit.go` files; `NoteResolution` lives in `note_resolution.go`. The tables also omit the required PURE/INTEGRATION column. Correct the complete entity inventory and add a revision entry. Classified Critical under this review’s explicit Core concepts gate.

3. **Important findings**

   - **README update missing for note editing.** `README.md:1047,1147` still describes pin clicks as reading/resolving only. Document editing in both casts and the operator overlay, Save/Cancel, the edited marker, and session-bound ownership—including loss of edit access after fresh unlock or code rotation. Atlas updates are present.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed: targeted Go edit/ownership/resolution/order/catch-up tests across six packages; all **279 Node tests**; pinned-range `git diff --check`.
   - Full `go test ./cmd/internal/broadcast ./cmd/internal/webcast/... -count=1 -timeout=120s`: every webcast package passed. Broadcast failed `TestNamedTunnelServesFromPrivateSocket` because its origin was TCP instead of a Unix socket; causality was not established.
   - Explicit headless viewer/operator tests failed during Chrome initialization with EOF and sandbox/GPU errors, before feature assertions.
   - A read-only reproduction against `noteDialogTransition` confirmed the creation regression. Existing pending-input success coverage exercises edit mode, missing this draft-mode case.
   - No repository files were changed. No prior findings required disposition. Full `make -k test` was not independently rerun.

6. **Architectural notes**

   | Marker | Result |
   |---|---|
   | ARCH-DRY | Pass: shared dialog, ownership accessor and timestamp-order helper. |
   | ARCH-PURE | Pass: dialog transitions and recording reduction are directly testable without IO. |
   | ARCH-PURPOSE | Flag: entity inventory needs reconciliation; creation semantics regress during the shared-dialog refactor. |
   | ARCH-MOCK | Pass: existing stateful recorder, transport and HTTP seams cover the added behavior. |
   | ARCH-CONSTRAINTS | Pass: existing text, request, rate and single-dialog limits are reused. |
   | ARCH-SECURE | Pass: public attribution cannot confer ownership; parsers reject injected ownership. |
   | ARCH-ORDER | Flag: successful creation with intervening input lacks a correct committed-state transition. |
   | ARCH-FUNERAL | Pass: terminal resolution history is bounded and cleared; edit records inherit explicitly documented recording retention. |

7. **Plan revision recommendations**

   Add `## Revisions` entries defining successful creation with intervening input, its regression tests, and the corrected entity paths/classifications. Update the README as part of the documentation gate.

```findings
findings:
  - id: new
    severity: Critical
    family: committed-operation-state
    title: |
      Successful draft creation remains eligible to create another note
    detail: |
      cmd/internal/broadcast/web/collab.js:245-249 retains draft mode when input changes before a successful save reply. Reproducing open-draft, input, save, input, result(204), save emits save-new again: the web viewer creates a second note, while the operator rejects its consumed draft at cmd/internal/webcast/recorder/operator.go:259. ARCH-ORDER: preserve newer text through an explicit committed-note transition or prevent draft input during creation, and add delayed-success regression coverage.
  - id: new
    severity: Critical
    family: core-concept-traceability
    title: |
      Core concept tables disagree with implementation locations and omit classifications
    detail: |
      workshop/plans/000537-web-note-editing-plan.md:19-22,33-39,200 maps edit entities to wire.go, notes.go, board.go, pointer_notes.go and session.go although their definitions are in the new note_edit.go files; NoteResolution is in note_resolution.go. ARCH-PURPOSE: reconcile every table row, add explicit PURE/INTEGRATION classifications, and record the correction under Revisions. Severity follows the required Core concepts contradiction gate.
  - id: new
    severity: Important
    family: user-surface-documentation
    title: |
      README update is missing for existing-note editing
    detail: |
      README.md:1047 and :1147 still describe pin clicks as read/resolve interactions, and README.md is unchanged in the pinned range. Document both casts and the operator overlay, Save/Cancel, edited markers, and the approved session-bound ownership lifetime; atlas additions alone do not satisfy the README gate.
```
