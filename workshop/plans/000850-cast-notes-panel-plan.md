---
collaboration-mode: discussion
---

# Cast notes panel implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to choose execution approach. Keep #849 coordination explicit; no unlanded stacking.

**Goal:** Make cast notes easy to move out of the way and reopen, and make each sketch's note discoverable.

**Architecture:** Retain the existing note list and note/sketch command owners. Add one session-local floating-panel controller with a pure transition/geometry core, shared by its expanded and minimized forms. Sketch affordances call existing label actions or display an already-final label read-only; submission selection and visibility witnesses remain with their existing owners.

**Tech stack:** Existing browser ES modules, CSS, Node tests and Go/Chrome integration harness. No dependencies or persisted preferences.

## Evidence and boundaries

The operator's screenshot in ops-echo/session.side/2026-10-23-03-echo-23-board-feedback.md shows the obstructing white `.annotation-notes` stack at top right. The dark Pending feedback preview at bottom is a separate projection. Target the former; do not redesign the latter or its publication acknowledgment. Evidence was inspected locally; no screenshot or application content is copied into this public repository.

Pair:6 owns #849 and has an unlanded diagnostic stage. Its cause is not yet known. Their confirmed invariants: Shown means preview publication acknowledged; prepare re-reads the identical visible selection, host and board/delivered stamps; Freeze uses exactly that selection; finished witnesses gate sketches; visibility reads fail until both paints complete. Keep shared sketch changes at the UI affordance boundary; pair:6 will report root cause and shared-file changes before editing. No #849 fix is included or cherry-picked.

## Proposed operator behavior (approval requested)

- The all-open-notes stack becomes a light-yellow panel, matching existing authoring background `#fff59d` and text `#2b2b2b`. Existing note rows and off-view indication remain, with keyed DOM identity and the latest note opened on click.
- Initial position stays near top right. Default panel width is 300 CSS pixels and height 320 CSS pixels, each bounded by the viewport minus 12-pixel margins. Header stays fixed; only the list scrolls. Note count never grows panel height.
- A labeled Minimize button collapses the list to a yellow `Notes (N)` button, at least 44 pixels high. Clicking or keyboard-activating it restores the panel at its current anchor. Both forms share one anchor; dragging the small button moves where the panel reopens. Mode/position survive list updates and document navigation within this mounted context, and reset on context disposal/reload.
- Drag the expanded header or minimized button with mouse, pen or touch. A movement threshold distinguishes click from drag. Dragging never opens a note, restores accidentally, draws ink, or sends pointer events to the page beneath it. Note text remains selectable; rows, scrollbar and editor controls do not start a drag. A focused move handle supports arrow-key repositioning; normal button activation still works.
- Pointer cancellation/lost capture restores the pre-drag anchor; resizing cancels an in-progress drag and clamps its resting anchor. At tiny dimensions, controls remain reachable and the scroll body shrinks. Empty lists hide the entire widget; incoming notes preserve the user's current expanded/minimized choice and update the count.
- Each on-view sketch gets a visible note affordance beside its painted bounds, with only that control taking pointer input. `Add note` for an unfinished owned sketch enters the existing finish/label flow; interrupted/pending-label sketches reopen that flow. `View note` for a final labeled sketch opens a read-only sticky view with Close. Existing labels are final in the recording reducer; this issue does not add relabeling or multiple notes per sketch. Active/pending toolbar text also says Add note rather than the vague Note.
- A sketch with no valid current placement gets no floating page control; its existing pending-label recovery remains reachable. Finished labels stay inspectable without granting edit rights. Controls are private UI under the existing excluded host, not drawing or recording content.

## Alternatives

1. **Recommended:** wrap the current note list; share one panel owner and use the existing sketch note pipeline. Small semantic footprint, no submission protocol changes.
2. Individual draggable sticky notes: adds one position/lifetime per annotation and does not fulfill the requested whole-panel control as directly.
3. Merge the list with Pending feedback: confuses open notes with the exact selected submission and collides with #849's authority boundary. Reject.

## State, actions and invariants

Actions: open a note, minimize/restore, start/move/finish/cancel drag, keyboard move, save/cancel label, close read-only note, resize, receive updated notes, dispose.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| No open notes | Widget absent | Add a note through existing tool | None |
| Expanded, idle | Yellow fixed-height scrolling list | Minimize or open a row | None |
| Minimized, idle | Yellow Notes (N) button | Restore | None |
| Captured pointer, threshold not passed | Current form unchanged | Click/restore or cancel drag | None |
| Dragging | Same form follows pointer within viewport | Finish or cancel drag | Only canceled position change |
| Resize or lost capture during drag | Drag canceled; last resting anchor clamped | Move again | Only canceled position change |
| New/removed/edited rows while minimized | Count updates; remains minimized | Restore | None; source notes unchanged |
| Label confirmation pending/refused | Existing saved/uncertain/held UI | Existing Check confirmation or retry action | Existing semantics, no new send |
| Final sketch note open | Read-only yellow note | Close | None |
| View/cast authority changes | Existing held/disabled controls; no new edit grant | Existing cancel/retry after authority returns | Existing behavior |
| Context disposed with pointer held | Widget gone; listeners/capture retired | Open a new context | Ephemeral panel position only |

One controller owns the widget DOM, listeners and single captured pointer for the mounted context. Pointer identity and tagged idle/pressed/dragging state prevent overlapping gestures. Minimize/restore/move are local presentation, not feedback-mask mutations: anchored notes/strokes remain painted and selected exactly as before. Never hold a feedback paint busy while dragging/minimized. Data edits still use their existing invalidate/paint path and must still fail stale prepare.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| `floatingPanelTransition` (mode, anchor, pointer and clamp transitions) | `cmd/internal/webcast/web/floating-panel.js` | new |
| `sketchNoteEntries` (one affordance per currently placed sketch, note capability/label) | `cmd/internal/webcast/web/sketch-notes.js` | new |

Pure functions take explicit viewport, size, metadata and projected positions. No DOM, timers, storage, network or injected IO mocks in their unit tests. Panel state belongs 1:1 to a mounted list; entries are a projection of existing sketch identity, never a new command authority.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| `mountFloatingPanel` | `cmd/internal/webcast/web/floating-panel.js` | new | DOM, pointer capture, keyboard, viewport resize |
| `mountAnnotate` / `noteList` | `cmd/internal/webcast/web/annotate.js` | modified / reused | Existing all-open-notes rows and guarded editor |
| `mountSketch` note affordance wiring | `cmd/internal/webcast/web/sketch.js` | modified | Existing finish/label actions and paint projection |
| `mountSketchNoteControls` | `cmd/internal/webcast/web/sketch-notes.js` | new | Keyed private controls and read-only label view |
| Browser embedding/module assembly | `cmd/internal/webcast/server.go`, `operator_script.go` | modified | Same modules in viewer and isolated operator world |
| Existing feedback visibility, preview transport and note reducers | Existing respective modules | reused unchanged | Authority, selection, persistence and paints |

The DOM adapter owns effects; pure transitions return next state and effects. Reuse `noteList` and the accepted sketch metadata/placement pass (ARCH-DRY), not a second note database or selection calculation. The read-only view never calls label/edit/send. Existing DOM fixtures provide stateful pointer capture/event/viewport behavior; the real headless Chrome harness checks browser layout and events against that model (ARCH-MOCK).

## Constraints and lifecycle

- Drag hot path: O(1) work and at most one anchor style update per pointer event, independent of N notes or S sketches; measure size only at gesture start/mode/resize, not a note-list scan per move. Count DOM/geometry operations for N/2N list sizes and fixed move sequences.
- Existing row reconciliation remains O(N). Sketch affordances derive in one pass over already placed P paths plus S keyed controls, O(P+S); no new placement requests, polling, per-point rescans, network or board writes. Count membership/projection visits at P/2P and reject quadratic grouping.
- Sizes are CSS-pixel product choices above; test desktop 1280x800 and narrow 320x480, plus shrinking below normal panel size. Clip/clamp controls rather than add page scrolling or let the handle escape (ARCH-CONSTRAINTS).
- Controller, pointer capture, resize and keyboard listeners live only until context dispose; retire exactly once even mid-drag. No file, cache, interval, persistent preference or new process family (ARCH-FUNERAL).
- Note text is rendered with textContent; preserve existing private host/shadow exclusions, off/driver guards and authenticated sketch actions. No API, payload, recording, receipt, schema or version change (ARCH-SECURE/EVOLVE).
- Explicit tagged interaction transitions handle cancellation, late pointer events and disposal. Existing async note/label owners keep their epochs and partial-outcome policy; no new waiter or optimistic success (ARCH-ORDER).
- The screenshot's blocking UI and discovery request are both covered, including mobile input and operator stage; no submit-loop workaround is smuggled in (ARCH-PURPOSE). Keep pure state independent of browser effects (ARCH-PURE).

## Implementation tasks (one atomic close, no Mx boundaries)

### Task 1 — pure panel state and owned browser effects

Files: create `web/floating-panel.js`, `tests/webcast/floating-panel.test.mjs`; extend the existing DOM fixture only as real pointer/viewport behavior requires.

- [ ] Write table/sequence tests for expanded/minimized anchors, threshold click versus drag, pointer identity, cancel/lost capture, resize clamping, keyboard movement, late events after dispose and no-data hiding. Run `node --test tests/webcast/floating-panel.test.mjs`; observe semantic RED.
- [ ] Implement deterministic transition and minimal adapter. Use pointer capture and scoped listeners; clear state before releasing capture to ignore resulting lost-capture events. Assert no underlying page action from drag, while inputs/scrollbar remain ordinary. Run tests GREEN and the counted N/2N no-list-scan pin. Commit.

### Task 2 — annotation panel styling and integration

Files: modify `web/annotate.js`, `web/annotate.css`, `server.go`, `operator_script.go`; extend `tests/webcast/annotation-list.test.mjs`, `tests/webcast/annotate.test.mjs`, relevant operator module/server tests.

- [ ] Add failing integration tests around real keyed `noteList`: count changes/edits preserve collapsed mode, anchor, scroll and row identity; restore opens latest text; hidden/off/driver behavior applies to both forms; empty/repopulated lists preserve the chosen mode. Existing remote/late-result editor guards stay intact.
- [ ] Wrap existing list body with the owned panel. Apply sticky colors and bounded fixed dimensions; header controls stay outside overflow. Update both explicit embedding paths/import maps. Run Node tests and focused module assembly/asset-serving Go tests GREEN. Commit.

### Task 3 — sketch note discovery without command changes

Files: create `web/sketch-notes.js`, `tests/webcast/sketch-notes.test.mjs`; modify `web/sketch.js`, `web/sketch.css`, embedding maps and `tests/webcast/sketch.test.mjs`.

- [ ] Add failing pure tests for unique per-sketch entries, valid-placement filtering, owned unfinished Add note, final read-only View note, interrupted/pending labels and unavailable/retired sketches. Count visits at P and2P.
- [ ] Add keyed DOM controls from the existing placement pass. Route Add note to current leave/openLabel; show final accepted text read-only, with no send/edit capability. Test labels using metacharacters, stale callbacks, handoff, final-label immutability and control removal/dispose. Do not change witness commits, preview publish or feedback visibility. Keep pair:6 informed before shared-file edits; reconcile any newly landed #849 implementation via main, never its unlanded branch. Commit GREEN.

### Task 4 — browser and protected-submission verification

Files: new `cmd/internal/webcast/cast_notes_panel_live_test.go` and/or extend existing operator/viewer Chrome fixtures; update `tests/webcast/operator-submission-preview.test.mjs` only for integration assumptions, atlas/webcast.md, atlas/webcast-submission.md and README's existing cast-note instructions.

- [ ] Add real headless Chrome cases for sticky computed colors, fixed scrolling height with long notes, mouse/touch drag of both forms, click versus drag, pointer cancellation, resize recovery and click-through to the formerly covered note/page. Verify visible sketch controls enter the existing draft path and final View note cannot mutate a record.
- [ ] Prove presentation-only moves/minimize do not alter visible scene selection, stamps, receiver cursor or shown acknowledgment; actual note edits/label/save/hide/undo still invalidate and refuse stale prepare. Run retained tests with #849 fixes when they land; coordinate the exact shared seams before implementation. No bypass if submission still fails.
- [ ] Commands: `node --test tests/webcast/*.test.mjs`; targeted `go test ./cmd/internal/webcast/... -run 'TestOperatorScript|TestServer|TestCastNotesPanel' -count=1`; headless `WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'TestCastNotesPanelLive|TestOperatorSketchLiveDrawing' -count=1 -timeout 5m`. Tests must prove non-skipped cases. Audit each named reader before individual certificate repair; run `make test-affected-inventory`.
- [ ] Commit/push final candidate, use `sdlc local-check run --pre-close` and retain its exact record. Required inventory immediately before close; SDLC supplies fresh review. At TL stage go after #849's diagnostic lease clears, `sdlc stage --build` and give the operator the short look below. Land only after SHIP, pre-close proof and acceptance; clear stage, notify hub. No new suite beside the train's full suite unless explicitly required by a gate.

## Operator look

Restart onto the staged head. Create several long notes, confirm yellow appearance and a fixed scrolling list; minimize, drag the Notes button, restore, move its header, and click the previously covered page/note. Draw and finish a sketch using its visible Add note affordance, then reopen its saved note with View note. Submit unchanged feedback successfully, and verify a real edit still prompts a fresh review. Pair:6 owns the separate submit-loop correctness fix; acceptance cannot assume a failed submission is harmless.

## Approval questions for the hub

Confirm the proposed 300x320 panel size and shared anchor behavior. Confirm final sketches expose read-only View note, while Add note retains the existing one final-label workflow; mutable sketch labels/additional notes would require a deliberate recording-contract scope change. No code before approval. Stage timing remains with the hub while #849 holds its diagnostic build.
