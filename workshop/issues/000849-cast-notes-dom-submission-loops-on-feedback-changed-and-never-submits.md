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

- With unchanged pending annotations, a second press submits them (test reproducing the loop first).
- A genuine change between review and submit is still caught (test).

## Plan

Approved by the TL (10-23): diagnose first through the operator, no headed repro.

- [x] Name the cause, protection unchanged: each prepare refusal names its condition (`PreviewRefuse.Reason`,
  `previewRefusal` still `errors.Is` `errSubmissionPreview`); the preview worker keeps why it last could not
  refresh and adds it to a refusal made while nothing current is shown; the panel reads "Pending feedback is out of
  date: …" in place of stale counts. Tests: core reasons; the manager over the operator's shape (shown notes, then
  a sketch the panel cannot show, refused with the reason, admitted once shown); live Chrome: the operator's sequence
  prepares and freezes, the panel goes stale and recovers; the real manager over a real cast admits notes.
- [ ] Stage on pair:0; the operator uses cast notes as usual and reports any refusal message.
- [ ] Fix the cause the message names; test reproducing the loop first, and a genuine change still refused.

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
