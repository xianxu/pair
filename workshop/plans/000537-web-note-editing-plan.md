# Own-note editing on both casts Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a viewer edit only notes made in their current unlocked browser session, and the cast-Chrome operator edit only operator notes, preserving authorship and showing an edited marker.

**Architecture:** Reuse the existing HttpOnly unlock cookie as private ownership proof; the lock gate supplies a trusted request principal, never the request body. Web edits are ordered recording events through Board and recorder; terminal edits update the existing NoteBoard under its pointer lock. Both surfaces share one dialog edit mode.

**Tech Stack:** Go HTTP/Board/recording, browser JavaScript, Node test runner, headless Chrome.

## Core concepts

| Name | Lives in | Status |
|---|---|---|
| NoteOwner | cmd/internal/broadcast/note_owner.go | new |
| NoteEdit | cmd/internal/webcast/recording/record.go | new |
| Note | cmd/internal/webcast/recording/record.go | modified |
| NoteView | cmd/internal/webcast/recording/reduce.go | modified |
| EditIn / ParseNoteEdit | cmd/internal/webcast/board/wire.go | new |
| OperatorEdit | cmd/internal/webcast/board/operator.go | new |
| Note / NoteBoard | cmd/internal/broadcast/notes.go | modified |
| NoteEditRequest | cmd/internal/broadcast/notes.go | new |

NoteOwner is an opaque non-secret projection of a validated session: a domain-separated SHA-256 digest of its random cookie. It can be published for display equality, but is NEVER accepted from a body/header as authority. Empty owner means legacy/read-only. A distinct fixed operator owner is supplied only by the isolated-context command path. One owner has many notes; ownership never changes. This reuses admission's credential (ARCH-DRY) without expanding castlock or introducing another token lifecycle.

NoteEdit contains Header, ID and replacement Text. Original Note gains optional owner and edited fields for its reduced projection; edit reduction copies the Note before changing Text/Edited and increments NoteView.Revision. The creation event, original author, creation time, anchor and region remain immutable. Each edit event records a revision in stream order; latest accepted edit wins (ARCH-ORDER). No expected-revision edit conflict protocol. Existing submission revision guards must reject a resolution selected before an edit.

EditIn contains Seg, ID, Text and server-supplied Owner. Its strict wire parser takes only seg/id/text (and existing viewer attribution if needed by the established adapter), never owner. OperatorEdit has the same target/text shape; ownership comes from the operator session. Terminal NoteEditRequest carries ID/Text and trusted Owner, parsed separately from HTTP identity injection. Terminal edits preserve all placement and attribution fields.

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| gate / session owner projection | cmd/internal/broadcast/lockgate.go | modified | validated cookie and HTTP request context |
| Board edit admission | cmd/internal/webcast/board/board.go | modified | ordered record emitter and current note state |
| operator edit admission | cmd/internal/webcast/board/operator.go | modified | bounded request ledger |
| web note-edit route | cmd/internal/webcast/server_board.go | new | HTTP role, generation, size/rate budgets |
| recorder edit handling | cmd/internal/webcast/recorder/operator.go | modified | trusted CDP context and ordered sink |
| terminal note-edit route | cmd/internal/broadcast/server.go | new | HTTP gate and session callback |
| terminal edit admission | cmd/internal/broadcast/pointer_notes.go | new | pointer mutex/generation and NoteBoard |
| terminal edit callback | cmd/internal/broadcast/session.go | new | notes notifications |
| noteDialog | cmd/internal/broadcast/web/collab.js | modified | DOM and injected post transport |
| web pins / mountAnnotation | cmd/internal/webcast/web/annotate.js | modified | current model, identity projection and dialog |
| terminal note adapter | cmd/internal/broadcast/web/viewer.js | modified | current notes, identity projection and dialog |
| operatorTransport adapter | cmd/internal/webcast/operator.js | modified | isolated command replies and request ledger |

Existing stateful recorder/CDP harnesses, HTTP test servers with independent cookie jars, and dialog DOM fixtures cover integrations. No new external service, worker, timer, unbounded request map or retained credential registry. The digest is not a credential; raw cookies never enter records, SSE, URLs or logs. Identity lookup failure fails closed to read-only.

## Ownership and admission contract

- The lock gate derives the owner only after castlock.Valid succeeds and adds it through a private context key. Expose the public owner digest on the existing successful GET session response header (204 remains unchanged). Browser code fetches that projection once during mount; no polling or extra credential storage.
- HTTP creation overwrites any internal owner from the trusted context. Strict request decoding refuses an owner field. Missing owner creates a legacy read-only note, retaining existing direct Board/test callers; editing always requires a nonempty trusted owner.
- HTTP edit checks link role, current annotation generation, live document, open target and exact owner equality. Copying another note's public Viewer.ID, name or owner digest grants nothing. Neither a drive seat nor operator status overrides ownership.
- The operator Board path stamps fixed operator ownership when creating notes. An isolated operator edit uses that owner; public HTTP cannot acquire it. Old notes with no ownership proof remain read-only, including legacy operator notes.
- Approved lifetime: refresh preserves ownership; fresh unlock, code rotation, another browser or another link makes prior notes read-only. Operator notes stay editable across overlay document contexts only when the target remains on the current document. Navigation still retires notes.
- Web reducer rejects editing unknown/resolved/old-document notes and never recreates one. Ordered recorder entry rechecks liveness and target; OFF generation checks remain authoritative at admission and write. Terminal pointer lock checks ON/generation and ownership in the same mutation.
- Operator edit uses existing request identity/reconciliation, with no Note ID in its acknowledgement proof. Existing target presence cannot prove an uncertain edit committed. Same request with changed payload is refused.

## Visible states and recovery

| Internal state | Viewer/operator sees | One action |
|---|---|---|
| Own open note; annotation available | Populated editor, Save/Cancel, edited marker when revised | Save |
| Other owner, legacy note, missing session proof, or view link | Read-only text; no Save | Close |
| Annotation OFF | Existing notes read-only if present; ordinary OFF clear still removes notes | Close |
| Pause while editing | Text held; Save disabled with paused notice | Wait for resume |
| Validation/refused save | Text retained with reason | Correct text and Save, or Close if no longer eligible |
| Lost/uncertain operator reply | Text retained; notice says Save checks the same operation | Save |
| Session revoked | Existing lock page flow; old notes read-only after unlock | Unlock |
| Note retired/navigation | Editor closes, cannot revive note | Open a current note |
| Late reply after another dialog opens | Current dialog untouched | Continue current editor |

Cancel before Save emits nothing. Cancel after sending only dismisses the dialog; it cannot roll back an accepted operation. While a save is pending, disable duplicate sends even if the input changes. Shared dialog generation checks cover success, failure and finally handlers. Remote edits refresh read-only text and pin tooltips, but do not overwrite unsaved edit text; subsequent Save wins in Board order.

Operator watch mode may open existing pins without enabling pen/gesture capture. Scope hit testing to pins/dialog, leaving the rest of Chrome interactive. Driving/disabled context policy must still prevent writes. No changes to draft prepare/save/cancel capture semantics; pair#538 owns that parallel work.

## Implementation tasks (single full-flow review boundary)

### 1. Session ownership seam and record model

Files: broadcast/note_owner.go, lockgate.go and lockgate_test.go; webcast/recording/{record,reduce,catchup}.go and colocated tests; webcast/web/{model,replay}.js; tests/webcast/{model,replay}.test.mjs and catchup_corpus.json.

- [ ] Write failing tests: two unlocked cookie jars project distinct owners; public identity copying cannot affect the context; refresh retains owner, re-unlock replaces it; no raw cookie in response projection.
- [ ] Run `go test ./cmd/internal/broadcast -run 'Test.*(Owner|LockGate)' -count=1`; confirm new assertion failure.
- [ ] Implement private context accessor and public projection header without changing castlock. Add optional owner metadata to note creation.
- [ ] Write codec/reducer tests for NoteEdit, immutable original creation record, exact one-note update, revision increment, edited marker, unknown/resolved/retired rejection, and submission NoteUnchanged becoming false.
- [ ] Implement registry/check/Kind, reducer copy-on-edit and Go/JS catch-up retention. Add edit cases to the shared all-cuts corpus; keep cleared/old-document edits out of catch-up.
- [ ] Run `go test ./cmd/internal/webcast/recording ./cmd/internal/broadcast -count=1` and `node --test tests/webcast/model.test.mjs tests/webcast/replay.test.mjs`; expect pass. Commit with #537 reference and model coauthor trailer.

### 2. Authoritative web and terminal edit paths

Files: webcast/board/{wire,board,operator}.go; webcast/server_board.go; webcast/recorder/{recorder,operator}.go; broadcast/{notes,pointer_notes,session,server}.go; corresponding *_test.go.

- [ ] Add failing Board/HTTP tests for own edit, same public identity with different trusted session, missing proof, operator-on-viewer denial, viewer-on-operator denial, OFF/view-only, stale generation, removed note, blank/oversized text, and preservation of author/placement.
- [ ] Run `go test ./cmd/internal/webcast/board ./cmd/internal/webcast -run 'Test.*(Edit|Owner)' -count=1`; confirm intended failures.
- [ ] Implement strict ParseNoteEdit and owner-bound Board.Edit/AnnotationAdmission.Edit using existing command lock and emitter. Add fixed forbidden response and shared note rate budget. Stamp trusted owner on creation.
- [ ] Add OperatorEdit parser/tag and EditAdmitted through the existing request ledger. Route recorder edit-note and include NoteEdit in permitted recorder input. Return empty acknowledgement Note ID for edits; test uncertain writes cannot self-confirm from target existence.
- [ ] Add queued writer tests: edit after resolve/clear/navigation rejected; edited submission selection cannot resolve newer text. Preserve generation checks at final ordered entry.
- [ ] Add terminal pure NoteBoard.Edit and PointerState edit under mutex. Wire session callback/notification and strict server route with context-derived owner. A stale edit after OFF/on cannot act on a new note; IDs are never reused.
- [ ] Test two actual HTTP cookie jars with copied public name/ID/owner fields on BOTH casts. Use Host gate, not test-only injected identities, for this security contract. Test operator cannot edit a viewer note.
- [ ] Run `go test ./cmd/internal/broadcast ./cmd/internal/webcast/board ./cmd/internal/webcast/recorder ./cmd/internal/webcast -count=1`; expect pass. Commit.

### 3. Shared dialog, viewer and overlay

Files: broadcast/web/{collab,notes,viewer}.js and existing dialog CSS; webcast/web/{annotate.js,annotate.css}; webcast/operator.js; tests/broadcast-viewer/{notes,pointermode}.test.mjs; tests/webcast/{annotate,operator,operator-scope}.test.mjs.

- [ ] Add failing dialog tests for populated edit, Save same ID, Cancel no request, held/readonly, pending input cannot duplicate Save, and late reply/finally cannot affect another dialog.
- [ ] Run `node --test tests/broadcast-viewer/notes.test.mjs tests/webcast/annotate.test.mjs`; confirm intended failures.
- [ ] Add noteDialog.edit(note) and editRequest dependency, reusing text validation, generation checks and draft layout. Edit sends only ID/text plus adapter-specific seg; no new draft capture and no author changes.
- [ ] Add one shared session-owner fetch helper; missing/failed projection is read-only. Terminal/web adapters compare public note owner only for display; server remains authority. Preserve edited marker in terminal parseNotes and web model/snapshots.
- [ ] Update pins to use latest note on click/tooltip while retaining element identity. Show edited marker in editable/read-only note. Ensure overlay watch pin hit testing works without intercepting background Chrome input.
- [ ] Map web note-edit to operatorTransport.mutate('edit-note', {seg,id,text}); retain exact request identity and changed-payload reconciliation. No prepare-note call for editing.
- [ ] Run `node --test tests/broadcast-viewer/*.test.mjs tests/webcast/*.test.mjs`; expect pass. Commit.

### 4. Integration, documentation and landing

Files: cmd/internal/webcast/{operator_live_test,viewer_annotate_live_test}.go; atlas/{broadcast,webcast,webcast-format}.md; issue and this plan.

- [ ] Add bounded headless viewer/overlay checks: create/open/edit/save/reopen, original author/anchor retained, Cancel, other-owner read-only, OFF refusal and draft invariants. Register cleanup before failure paths; wait on committed state/ACK, not sleeps.
- [ ] Run targeted headless tests with `WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'Test.*Live.*(Operator|Annotat)' -count=1 -timeout=10m`, adjusting the regex to the actual declared test names and recording the exact command.
- [ ] Document new note-edit wire record, own-session lifetime and edited marker in existing atlas pages; no new atlas family. Record approved scope and test evidence in issue.
- [ ] Run full `make -k test` to completion and cite all failures, including known unrelated failures; use targeted race tests for changed admission/queue paths. Check `git diff --check`.
- [ ] Report real `git rev-parse HEAD` and smoke instructions to hub; wait for authorized visible staging/acceptance. Close through sdlc with evidence, fix review findings, and on SHIP merge main then `sdlc merge --yes` with all landing gates. Report PR/hash to hub and stop.

## Revisions

### 2026-10-13 16:15 PDT — approved scope before implementation

TL relayed operator choice: own notes only including operator, edited marker, terminal included; reuse existing session cookie with session-bound ownership and no castlock expansion. Earlier proposed editor attribution/operator override is superseded. Pre-existing notes without proof remain read-only. Terminal edit is new code because investigation found no existing text-update path.

### 2026-10-13 16:20 PDT — plan review PQ-1/PQ-2/PQ-3

The following function-level strategies replace the prose test inventories in
tasks 1–4 as the authoritative unit-test design. Task commands and integration
checks remain. They exercise production functions, not a mirrored test model.

| Production function | Strategy and mechanical guard | Test location |
|---|---|---|
| gate / NoteOwnerFromContext | Drive real HTTP requests through Host using two cookie jars; mutate public identity inputs while asserting only validated cookies determine ownership; preserve 204 session probe contract | broadcast/lockgate_test.go |
| ParseNoteEdit / ParseOperatorRequest / ParseNoteEditRequest | Table plus fuzz malformed/duplicate/oversized keys; only canonical target/text/segment decode, owner can never be injected by parsing | webcast/board/wire_test.go, operator_test.go; broadcast/notes_test.go |
| recording.State.Apply(NoteEdit) | Fold generated legal/retired-note event sequences; each accepted edit changes exactly Text/Edited/Revision, failed edits leave state byte-equivalent, source record unchanged | webcast/recording/note_edit_test.go |
| recording.CatchUp / JS catchUp / reduce | Extend common corpus and evaluate every prefix/cut against full reduction; compare note identity/text/edited/revision and no resurrection after clear/document change | webcast/recording/catchup_test.go; tests/webcast/{model,replay}.test.mjs |
| AnnotationAdmission.Edit / OperatorSession.EditAdmitted | Stateful ordered writer with explicit barriers; vary owner, generation, target state and request identity, asserting accepted writes only and at-most-once same operator request | webcast/board/note_edit_test.go, operator_test.go |
| Recorder.writeOperator / Emit handler | Queue competing lifecycle events in controlled order; final sink admission rejects invalid target/doc/generation and never confirms uncertain edit from existing target ID | webcast/recorder/operator_test.go, recorder_test.go |
| NoteBoard.Edit / PointerState.editNote | Pure table plus controlled flip/edit sequence; denied edits preserve full board, accepted edits preserve anchor/author/ID; version changes only on accepted mutation | broadcast/notes_test.go, pointer_notes_test.go |
| noteDialogTransition | Enumerate short event permutations against production reducer; invariants: at most one pending effect, held states emit none, stale completions leave current editor identical | tests/broadcast-viewer/notes.test.mjs |
| pinLayer / mountAnnotation / operatorTransport | Stateful DOM fixtures with deferred transport answers; compare rendered current note and emitted command to the committed model; same request ID reused until settled | tests/webcast/{annotate,operator,operator-scope}.test.mjs |

**Additional PURE entity:** `noteDialogTransition` in
`cmd/internal/broadcast/web/collab.js` (new). One shared reducer owns the entire
dialog lifecycle; noteDialog owns its state privately and renders its result.
DOM handlers dispatch events and execute emitted effects, never mutate the
authoritative state directly (ARCH-PURE/ARCH-ORDER).

State is `{epoch, mode, target, payload, text, held, pending, error}`. Mode is
closed/draft/view/edit; pending is null or `{epoch, kind, text}`. Events are
open-draft/open-view/open-edit, input, hold, save, resolve, result, retire,
dismiss. Transition returns `{state, effect}`; effects are save-new, save-edit,
resolve or none. Validation is the existing bounded plain-text rule.

- Open/dismiss/retire increment epoch and discard the dialog's pending token;
  they do not cancel or pretend to roll back a dispatched server operation.
- Save requires draft/edit, valid text, no hold and no pending token. It records
  the exact submitted text in pending before producing one effect. Input while
  pending changes unsent text but cannot issue another effect.
- Resolve requires an allowed current note and no hold/pending; permission is
  supplied by the adapter, independently of edit ownership.
- Result includes epoch and submitted text. Wrong epoch is an identity
  transition. Accepted current results close only if input still equals the
  submitted text; otherwise retain new input and say the earlier text saved.
  Refused/unknown current results clear dialog pending, retain text and show
  transport recovery. The operator transport, separately, retains uncertain
  request identity; a subsequent Save reconciles it rather than duplicating it.
- Hold prevents new effects without changing text or cancelling the pending
  operation. Removal/navigation retires an existing-note editor. An accepted
  result after hold cannot enable controls. Each async handler dispatches result
  once; no unscoped finally handler refreshes another editor.
- noteDialog.update sends retire if target disappears; read-only text can
  refresh from current notes, while edit text remains the user's unsaved text.

**Storage lifecycle (PQ-3):** NoteEdit appends bounded replacement text to the
existing notes.jsonl, via recording/dir.go:89. Its incremental cost is the
encoded JSON bytes plus newline per accepted save, measured with Encode in
recording/note_edit_test.go at the existing maximum note size. Replay history
therefore grows with accepted edits under the existing note rate budget; there
is no whole-recording byte cap. As atlas/webcast-format.md:337 documents,
recordings remain until the operator deletes the directory. No extra retention
registry or automatic cleanup is introduced.
