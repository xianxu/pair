---
id: 000526
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '4acd898d8c7f15289272f2284abc1d906ab3b557' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-12
