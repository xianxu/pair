---
id: 000849
status: done
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '80151b1b8d23676ea8e60fef64701a0e4209d342' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T17:38:50-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 4.12
---

# Cast notes: DOM submission loops on 'Feedback changed' and never submits

## Problem

## Spec

**Cast notes: DOM submission loops on "Feedback changed" and never submits (echo operator, 10-23).** In the Chrome
cast (broadcast-web notes), every press of the DOM submission chord prints `couch: Feedback changed; review Pending
feedback and submit again; edit freely and press the DOM chord again`, with the panel showing "Pending feedback: 0
sketches, 4 annotations". Pressing again repeats it; the notes never reach the agent. Evidence: ops-echo
`session.side/2026-10-23-03-echo-23-board-feedback.md` and its screenshots in
`session.side/assets/2026-10-23-03-echo-23-board-feedback/` (the error lines in nvim's :mes, and the board with four
floating notes and the pending-feedback panel).

Likely shape (to confirm first): the confirmation compares a fingerprint of the pending feedback taken when the panel
was shown with one taken at submit, and something in the annotations changes between the two on every press (a
timestamp, draft state, ordering, or the act of reviewing itself), so the check can never pass. Find the cause
before changing anything; no weakening of the "feedback changed" protection.

## Done when

- A press submits the board's open notes at their current revisions and its finished, labelled sketches at their
  current versions (already delivered versions left out, oldest 8 sketches), whatever the page is doing: on a page
  changing every 50 ms, unchanged notes submit (live test that reproduces the refusal on the old code first).
- The only refusal is a note still being written at the press -- a new note not saved, or an edit whose text changed
  and is not saved -- naming it; resolving uses the captured revision, so a note edited after the press stays open
  (tests).
- Sketch images render from the recording at each sketch's recorded anchors, never from the live painter (test).
- The Pending feedback panel is informational: a preview that cannot refresh marks it out of date and never gates
  a submit (test).
- An identical repeated selection record neither reaches the recording twice nor invalidates feedback visibility
  (tests).

## Plan

- [x] Name the cause, protection unchanged (step 1, ebb4242be): named refusals, worker's refresh problem, panel marked
  out of date.
- [x] Stage on pair:0; the operator reproduced it on echo: "the panel did not show it: board: the page changed".
- [x] Press-time selection (option one, TL 10-23): a pure function (`board.Selection.PressSelection`) picks from the board selection minus
  delivered versions (sketches oldest 8 first, remaining counted); `Board.Selection` leaves out a sketch whose
  witness is unfinished instead of failing; prepare takes it at the press with no preview gate, no stamp comparison
  and no freeze proposal. The selection carries no current view, so `appendSubmissionSketches` renders each sketch at
  its recorded finish scene.
- [x] Note-edit refusal: the page reports an unsaved draft or a changed, unsaved edit (one CDP evaluation); prepare
  refuses naming the note ("note \"…\" has unsaved edits; save or cancel it, then submit again"), or "a new note is
  not saved yet".
- [x] Panel: the preview worker keeps publishing for display only; prepare no longer reads it. Drop the preview
  refusal plumbing that only served the gate (`previewRefusal` on prepare, `prepareMoved`), keep the panel's
  out-of-date mark.
- [x] Noise at its source: the recorder's observer drops a selection identical to the last one it sent (a resync
  still sends); operator.js does not invalidate feedback visibility for selection records.
- [x] Tests: live Chrome page changing every 50 ms (selection + text): old code refuses, new code submits unchanged
  notes, and the repeated caret is recorded once (the observer dedupe's test, mutation-checked; live only, since the
  observer needs a real DOM); captured note revisions; an unsaved note refuses naming it; unit tests for the selection
  rules; atlas, the agent reading guide, the couch skill and README rewritten for the press-time contract.

## Revisions

- 2026-10-23 (TL): the stall did not reproduce on the staged build, so step 1's diagnostics are #849's deliverable.
  Done-when moves from "fix the root cause, test reproducing the loop first" to the named refusal and stale panel,
  with deterministic tests for an unchanged press admitted after a refusal and a genuine change still refused. If
  the loop recurs, its message names the cause and a follow-up issue fixes it.

- 2026-10-23 (TL, option one): the operator reproduced the refusal on echo; the cause is page churn, not feedback
  churn. Submit no longer depends on the preview: the press takes the board's feedback, the only refusal is an open
  note edit or draft, sketch images come from the recording, the panel is informational, and identical selection
  records stop invalidating visibility. Done-when and Plan replaced accordingly.

- 2026-10-23 (close review): an open edit whose text is unchanged does not refuse -- there is nothing unsaved to lose
  -- so the refusal is an unsaved draft or a changed, unsaved edit; Done-when and Plan say so. The observer dedupe is
  tested live (it needs a real DOM), not by a unit test.

## Log

### 2026-10-23
- 2026-10-23: closed — Post-close delta is inventory metadata only: re-recorded tests/affected-readers hashes for cmd/internal/submission/model_test.go and sketch_page_test.go (literal term lists changed in the review sweep, disposition local_or_literal_fixture unchanged); make test-affected-inventory passes; sdlc-local/suite pass on f18b57703. (fixed-then-ship); review verdict: SHIP
- 2026-10-23: closed — Option one (TL 10-23): press takes board.Selection().PressSelection(delivered), no preview gate. Live Chrome on the host: TestSubmissionManagerLiveSubmitsOnAChangingPage (page changes every 50 ms, dispatches selectionchange) refused on the old code, now submits and prepares; asserts the caret is recorded once (mutation-checked: 42 records without the observer dedupe); TestOperatorFeedbackLivePrepareAndStalePanel reports an unsaved draft. Unit: TestSubmissionPressNeedsNoPreview, TestSubmissionPressReadsOutsideLockAndIgnoresPageChanges, TestSubmissionRefusesAnUnsavedNote, TestSubmissionPressCapturesNoteRevisions, TestSubmissionPanelOutOfDateNeverGates, TestPressSelectionLeavesOutDeliveredAndKeepsOldestEight, TestOperatorViewIgnoresSelectionRecords (mutation-checked), TestCastPressSelectionRendersFromTheRecording, node capture-dialog unsaved getter; reading guide, skill and README swept (model_test, sketch_page_test, skill_privacy_test). sdlc-local/suite pass on 5cd9a3e78 (known flake #825 rerun). Pre-existing on clean origin/main, not this change: webcast TestLiveConformance and TestLiveCastEndToEnd (in-page pause control click).; review verdict: SHIP
- 2026-10-23: flow upgraded quick → full — 285 added lines in code files (limit 100); an earlier round of this close already ran the full review

Investigation (pair:6), from the operator's recording 20261023T165427-6f1f (17:07 presses ≈ t 700–755 s):

- One message hides about a dozen refusals: the manager maps every prepare-time failure (no preview worker, fresh
  read or resolve error, host/delivery/generation moved, `PreviewPrepare` refusal, freeze rejection) to
  `errSubmissionPreview`, and the preview worker's own failures (read, resolve, publish ack) are never reported.
- **Established:** replaying the recording through a real Board shows its selection held 1 sketch + 4 notes from the
  sketch label at t=653 s (generation 6). The panel the operator saw, "Pending feedback: 0 sketches, 4 annotations"
  with no "N more sketches", therefore predates the label: after it, the preview worker never completed another
  acknowledged publication, so every press compared against a stale preview and was refused.
- **Ruled out:** recorded-prefix drift (no records 700–780 s; live pointer moves pass); the live Couch's build
  (3dedd7b6, the #812 branch: no submission changes); nil/empty slice equality; a publish livelock (no notifications
  during publish).
- **Not reproduced headless:** a live test of the manager's exact sequence on the operator's own page (rebuilt from the
  recording's document + stylesheet), at their viewport (864×997 @2x), with their four notes, their sketch drawn from
  the recorded pen points (first stroke undone), the long label saved by click, the panel shown before the sketch;
  and the real `submissionManager` over a real cast with four notes. All admit on the first press.
- Untested differences: a headed, unfocused or occluded window (rAF throttling), the recorder's periodic screenshots,
  and a worker that ran for ten minutes across sketch-mode transitions.

Staged diagnosis (pair:0, 3de33c10b, then the combined #849 + #850 build 6f0478a3a):

- On the combined build the operator's DOM submission went through ("works now"; a note on Google's "Top stories"
  section reached ops:0 at 20:42). The refusal loop did not recur, so its root cause stays unidentified.
- Replay evidence stands as logged above: the board held the sketch from t=653 s while the panel still showed the
  pre-label preview, so the preview worker stopped acknowledging publications after the sketch label. Neither the
  headless replay of the operator's page and pen points nor the real manager over a real cast reproduced it.
- Done-when tests (deterministic): `TestSubmissionRefusalNamesWhyThePreviewCouldNotRefresh` (the operator's shape: a
  refusal naming why the preview could not refresh, the panel marked out of date, then the unchanged feedback
  admitted once shown); `TestPreviewRefusalNamesItsCondition`; genuine change still refused:
  `TestSubmissionFeedbackPublishesExactNotesAndRejectsLaterEdit`, `TestPreviewPrepareAndFreezeRejectStaleSelection`.
  Live (WEBCAST_LIVE_CHROME=1): `TestOperatorFeedbackLivePrepareAndStalePanel`,
  `TestSubmissionManagerLivePreparesUnchangedNotes`.

Root cause (operator reproduction on the combined build, recording 20261023T205617-99c5):

- Message: "Feedback changed (the current feedback has not been shown yet (the preview is waiting to refresh); the
  preview could not refresh: the panel did not show it: board: the page changed)".
- echo's board emits a `selection` record every 100–200 ms (the same caret, node 3 offset 3, repeated), plus pointer,
  hover and a layout record every 5 s. Every record runs `visibility.invalidate()` (operator.js record callback),
  which bumps the visibility revision and notifies Couch; Couch drops its cached visibility. The preview worker's
  read, resolve, freeze and panel acknowledgement span several CDP round trips under one revision, so on echo they
  never complete, and prepare refused because nothing current was shown. Quiet pages (Google) complete, which is why
  the earlier session worked.

Implementation (option one), cc5810671 + 5cd9a3e78:

- Prepare reads the delivered versions and `SubmissionUnsavedNote` outside the manager lock, re-checks job, cast,
  delivery generation and guard, then takes `SubmissionSelection().PressSelection(delivered)`. The core preview events
  that served the gate (`PreviewPrepare`, `PreviewFreezeProposal`, `PreviewFreezeRejected`, `PreviewRefuse`) and the
  manager's freeze and refusal plumbing are gone; the worker still publishes the panel and marks it out of date.
- The unsaved-note check is `noteDialog.unsaved` (collab.js, shared by broadcast and webcast through the symlink):
  a new draft, or an edit whose text differs from the saved text. An untouched open edit does not refuse.
- `Board.selectionLocked` skips a sketch without finished witnesses instead of returning `ErrBroken`.
- Reproducing echo's noise took `document.dispatchEvent(new Event('selectionchange'))`: re-collapsing the same caret,
  rewriting the caret's text node and a focused input re-set in a headless tab fire no `selectionchange` in Chrome.
  The exact trigger in echo's board re-render was not pinned down; the observer only listens for the event.
- Pre-existing, not this change: webcast `TestLiveConformance` and `TestLiveCastEndToEnd` fail at the in-page pause
  control click on clean origin/main too.
