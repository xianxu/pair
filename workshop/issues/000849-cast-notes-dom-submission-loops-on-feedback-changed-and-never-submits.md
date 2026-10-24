---
id: 000849
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'a693ff04b7a34dcb1d1d63d1ece31ef5ab44fd00' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-23
