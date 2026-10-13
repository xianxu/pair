---
id: 000526
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '200462b02099269adf7db87adcff6f6c4e7f2a76' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T23:04:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# broadcast-web: from submission to issue: a privacy flag per bundle (origin policy) and the couch skill's rule for writing notes up as issues

## Problem

When the operator asks an agent to "file an issue from these notes" with a DOM submission (#483), no rule explains how to externalize the evidence. A crop of private data, such as a customer's app or a logged-in page, can land in repository issue history.

## Spec

Operator requirements, 2026-10-12: issues hold evidence as text; only relevant crops go into the issue asset folder under ariadne#337. Recordings stay local (no S3). Privacy is gated rather than left to agent judgment.

1. Couch has a small operator-owned list mapping page origins to `shareable` or `private`, defaulting to `private`. For example, the operator may list `http://localhost:4321` as shareable. Each submission records `privacy: shareable|private` and the matched rule in `submission.json`. Agents read the policy; only the operator edits the list.
2. The Couch skill's "Reading a broadcast-web recording" section and the bundle's `how_to_read` require issue evidence as text: page URL, element and classes, box text, the request, and one line describing what the crop shows. Attach only relevant crops, only when the bundle says `shareable` AND the repository asset policy permits it (ariadne#337). For private origins, summarize instead of quoting box text. Name recordings as local evidence that may expire; the issue must remain understandable without them.
3. Check that typed input values are not quoted in box text for private origins; passwords are already excluded.

Coordinate with #525, which adds the page URL to the same manifest. Build on main rather than its unlanded branch.

## Done when

- A submission from an explicitly shareable origin records `shareable` and its matching rule; unlisted origins record `private`, as do explicit private rules.
- The Couch skill and `how_to_read` carry the text-first evidence rule, the two conditions for crop attachment, private-text summarization, and the local/expiring recording limitation.
- Tests pin the manifest privacy field, matched rule, private default, and exclusion of typed input values for private origins.


## Plan

- [ ] Implement and verify [the durable privacy plan](../plans/000526-submission-privacy-plan.md).
- [ ] Close through the SDLC review, open the PR, and report to hub; merge only on SHIP.

## Log

### 2026-10-12

- Published the operator contract to main at `9e9a182ec00dad07ea31dc57b7be712a7517522e`; started an issue branch from main.
- Inspected submission model/build/store, Cast preparation, recorder capture fences, DOM collector and shipped Couch skill. Input `.value` is not read, but textarea/editable descendants need regression coverage.
- Drafted `workshop/plans/000526-submission-privacy-plan.md`; policy matching is pure (ARCH-PURE), capture identity remains the authority (ARCH-SECURE), and delivery retains its existing reducer (ARCH-ORDER). No implementation or tests run yet.

## Revisions

### 2026-10-12 — hub coordination with #525

Hub confirmed top-level `privacy` and `privacy_rule`, an operator-edited local exact-origin list, default private, and no edit CLI. #525 owns the capture-bound source URL; whichever issue lands second integrates the fields, without building on the other's unlanded branch. The draft plan proposes excluding editable values for all origins to keep the protection independent of publication permission.
