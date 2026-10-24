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
flow: {kind: quick, provenance: inferred, spec: "a96fc805", done: "c2d1c028"}
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

- A refused submission names its cause, and the panel reads out of date instead of showing stale counts (tests).
- With unchanged pending feedback, a press after the preview is shown submits it, including after a refusal
  while the preview could not refresh (test).
- A genuine change between review and submit is still caught (test).

## Plan

Approved by the TL (10-23): diagnose first through the operator, no headed repro.

- [x] Name the cause, protection unchanged: each prepare refusal names its condition (`PreviewRefuse.Reason`,
  `previewRefusal` still `errors.Is` `errSubmissionPreview`); the preview worker keeps why it last could not
  refresh and adds it to a refusal made while nothing current is shown; the panel reads "Pending feedback is out of
  date: …" in place of stale counts. Tests: core reasons; the manager over the operator's shape (shown notes, then
  a sketch the panel cannot show, refused with the reason, admitted once shown); live Chrome: the operator's sequence
  prepares and freezes, the panel goes stale and recovers; the real manager over a real cast admits notes.
- [x] Stage on pair:0; the operator uses cast notes as usual and reports any refusal message.
- [x] ~~Fix the cause the message names~~: the loop did not recur on the staged build (see Revisions); step 1 lands
  as the deliverable, and a recurrence names its cause for a follow-up.

## Revisions

- 2026-10-23 (TL): the stall did not reproduce on the staged build, so step 1's diagnostics are #849's deliverable.
  Done-when moves from "fix the root cause, test reproducing the loop first" to the named refusal and stale panel,
  with deterministic tests for an unchanged press admitted after a refusal and a genuine change still refused. If
  the loop recurs, its message names the cause and a follow-up issue fixes it.

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
