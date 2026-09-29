---
id: 000341
status: open
deps: []
github_issue:
target: review-protocol
created: 2026-09-28
updated: 2026-09-28
estimate_hours:
card_mirror: '47578522d29d69247388eefe8a0f3a9ed17e2140' # card fields mirrored from issue-cards; edit via sdlc
---

# Alt+C restores review target from branch

## Problem

Review history survives in `review/<slug>` branches, but Alt+C uses a conversation-scoped target record and any live review pane rather than resolving the checked-out branch. After reviewing document A, switching to a separate review of B, and returning to A's branch, Alt+C can still show or open B. The operator must close the old pane and explicitly select A again with `:PairReview`.

## Spec

Restore the document associated with the current review branch when Alt+C is invoked. Support leaving review A, working on review B, and returning to A without manually reselecting its document. Keep the existing one-document-per-review-branch scope.

Reconcile an existing pane and cached target with the current branch before showing a document. Preserve unsaved human edits and pending agent work; if restoration cannot proceed safely, explain the conflict instead of silently opening the wrong document or losing work. Branch history remains the authority for durable review context (ARCH-PURPOSE).

Define behavior for missing or ambiguous branch-to-document identity and for non-review branches during design. Preserve conversation isolation while allowing an explicitly invoked Alt+C to restore a review from its branch in a fresh session.

## Done when

- An A → B → A branch-switch sequence restores the correct document on each Alt+C invocation without another `:PairReview <file>` selection.
- A stale target or a live/hidden pane for B cannot cause Alt+C on A's review branch to display B as the active review.
- Resuming restores the available committed review context and decorations; existing human/agent round history is preserved.
- Unsaved edits and pending agent handoffs are handled explicitly without data loss or applying a round to the wrong document.
- Automated regression coverage exercises branch switching, stale targets, live panes, fresh-session restoration, and missing/ambiguous identity; the atlas documents the resulting behavior.

## Plan

- [ ] Design branch-to-document resolution and safe pane/pending-round transitions.
- [ ] Implement restoration with regression coverage and update the review-workbench atlas.


## Log

### 2026-09-28

Filed from the operator's Alt+C workflow questions. Current behavior traced through `nvim/init.lua` (`PairReviewToggle`, target storage), `cmd/internal/reviewcmd/run.go` (readiness and scoped-file discovery), and `nvim/review/init.lua` (reconstruct-on-open). Task capture only; implementation has not started.
