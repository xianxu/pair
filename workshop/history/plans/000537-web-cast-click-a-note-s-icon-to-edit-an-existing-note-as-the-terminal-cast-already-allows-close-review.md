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

---

## Re-review — 2026-10-13T17:18:42-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 537 — web cast: click a note's icon to edit an existing note (as the terminal cast already allows) |
| repo | pair |
| issue file | workshop/issues/000537-web-cast-click-a-note-s-icon-to-edit-an-existing-note-as-the-terminal-cast-already-allows.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3a0355cb8ec51e3720c550a0dcfd6b038b7bd009..8d1719f3363b6039a67043f5f9af9d603930ebbb |
| command | sdlc close --issue 537 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T17:18:42-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

All three prior findings are addressed. The implementation matches the revised ownership and editing contract, preserves recording history, and rejects stale edits. No new blocking findings emerged. Confidence is limited by headless Chrome crashing before the browser checks could exercise editing.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      collab.js:233,271 blocks draft input during creation. Four regressions at tests/broadcast-viewer/notes.test.mjs:330-364 pass at HEAD and fail when the fix is removed in a disposable copy, which was subsequently deleted.
  - id: BR-2
    disposition: addressed
    note: |
      workshop/plans/000537-web-note-editing-plan.md:290 explicitly supersedes the earlier inventories with corrected definition paths and PURE/INTEGRATION classifications. Checked the replacement rows against source, including both note_edit.go files and note_resolution.go.
  - id: BR-3
    disposition: addressed
    note: |
      README.md:1053 and :1168 now document both casts, operator ownership, Save/Cancel, edited markers and session lifetime. These descriptions agree with cookie-derived ownership, Board/operator admission and the shared dialog.
```

1. **Strengths**

   - Ownership comes from validated session context; strict edit parsers reject injected ownership fields.
   - `recording/reduce.go:240` copies creation records before editing and increments revisions, preserving history and submission guards.
   - `board/operator.go:319` correctly requires operation-specific confirmation rather than treating target existence as proof of success.
   - Frozen-clock tests exercise actual split/merge persistence ordering.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - All **283 Node tests passed**.
   - Go packages **board, recording, query, recorder and webcast passed**.
   - Targeted Board/recorder race tests passed.
   - Broadcast’s package run failed only `TestNamedTunnelServesFromPrivateSocket`; that test and the targeted ownership/edit tests passed with `TMPDIR=/tmp`.
   - Both headless editing tests failed during Chrome startup with CDP EOF/GPU-process errors, including a retry with the shorter temporary path. Browser behavior was therefore not independently confirmed this round.
   - `git diff --check` passed. Full `make -k test` was not rerun; its recorded evidence was inspected.

6. **Architectural notes**

   - **ARCH-DRY — Pass:** shared dialog, ownership seam and timestamp-ordering helper are reused.
   - **ARCH-PURE — Pass:** parsers, reducers and dialog transitions separate decisions from injected effects.
   - **ARCH-PURPOSE — Pass:** viewer, terminal, operator, replay and query paths implement the revised scope.
   - **ARCH-MOCK — Pass:** stateful recorder, transport and query fixtures use production boundaries; live verification has the limitation above.
   - **ARCH-CONSTRAINTS — Pass:** edits reuse request, text-size, rate and single-dialog bounds.
   - **ARCH-SECURE — Pass:** public attribution grants no edit authority; revoked ownership and malformed requests have regression coverage.
   - **ARCH-ORDER — Pass:** production transitions own dialog state; tests control delayed replies, retirement and uncertain operator outcomes.
   - **ARCH-FUNERAL — Pass:** terminal resolution history is bounded and cleared at cast end; edit records inherit the documented recording deletion policy.

7. **Plan revision recommendations:** None. The existing revisions reconcile scope, implementation locations and the corrected pending-creation behavior.
