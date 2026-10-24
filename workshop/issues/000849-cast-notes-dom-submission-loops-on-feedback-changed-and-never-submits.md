---
id: 000849
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '16b2df0542cff7dfc0e87a1fce8967bc62f1ec20' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T17:38:50-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "46dabe43", done: "f7c72a4b"}
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
- The only refusal is a note with an open edit or unsaved draft at the press, naming it; resolving uses the captured
  revision, so a note edited after the press stays open (tests).
- Sketch images render from the recording at each sketch's recorded anchors, never from the live painter (test).
- The Pending feedback panel is informational: a preview that cannot refresh marks it out of date and never gates
  a submit (test).
- An identical repeated selection record neither reaches the recording twice nor invalidates feedback visibility
  (tests).

## Plan

- [x] Name the cause, protection unchanged (step 1, ebb4242be): named refusals, worker's refresh problem, panel marked
  out of date.
- [x] Stage on pair:0; the operator reproduced it on echo: "the panel did not show it: board: the page changed".
- [ ] Press-time selection (option one, TL 10-23): a pure core function picks from the board selection minus
  delivered versions (sketches oldest 8 first, remaining counted); `Board.Selection` leaves out a sketch whose
  witness is unfinished instead of failing; prepare takes it at the press with no preview gate, no stamp comparison
  and no freeze proposal. The selection carries no current view, so `appendSubmissionSketches` renders each sketch at
  its recorded finish scene.
- [ ] Note-edit refusal: the page reports an open note dialog or unsaved draft (one CDP evaluation); prepare refuses
  naming the note ("note N is being edited; save or cancel it"), or "an unsaved note draft is open".
- [ ] Panel: the preview worker keeps publishing for display only; prepare no longer reads it. Drop the preview
  refusal plumbing that only served the gate (`previewRefusal` on prepare, `prepareMoved`), keep the panel's
  out-of-date mark.
- [ ] Noise at its source: the recorder's observer drops a selection identical to the last one it sent (a resync
  still sends); operator.js does not invalidate feedback visibility for selection records.
- [ ] Tests: live Chrome page changing every 50 ms (selection + text): old code refuses, new code submits unchanged
  notes; a note edited after the press is not resolved; an open note edit refuses naming it; unit tests for the
  selection rules and the observer dedupe; atlas/webcast-submission.md rewritten for the press-time contract.

## Revisions

- 2026-10-23 (TL): the stall did not reproduce on the staged build, so step 1's diagnostics are #849's deliverable.
  Done-when moves from "fix the root cause, test reproducing the loop first" to the named refusal and stale panel,
  with deterministic tests for an unchanged press admitted after a refusal and a genuine change still refused. If
  the loop recurs, its message names the cause and a follow-up issue fixes it.

- 2026-10-23 (TL, option one): the operator reproduced the refusal on echo; the cause is page churn, not feedback
  churn. Submit no longer depends on the preview: the press takes the board's feedback, the only refusal is an open
  note edit or draft, sketch images come from the recording, the panel is informational, and identical selection
  records stop invalidating visibility. Done-when and Plan replaced accordingly.

## Log

### 2026-10-23

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
