---
id: 000341
status: working
deps: []
github_issue:
target: review-protocol
created: 2026-09-28
updated: 2026-09-28
estimate_hours:
card_mirror: '2e74fcd73db5667c843ae48ea47f356dc6ee3f2a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-28T20:39:30-07:00
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

### 2026-09-28 — investigation and proposed design

Claimed and entered planning on `000341-branch-review-restore`. Confirmed that a live pane bypasses target/branch checks, readiness picks the first path from any latest review commit, reconstruction reads any latest agent round, and handoff polling unlinks before identity can be checked. `RunOpen` also kills an existing pane whose exit handler may save stale text into the newly checked-out branch.

The proposed full-flow design is in [the durable plan](../plans/000341-branch-review-restore-plan.md). Git owns review identity; the pane owns safe activation; pending work blocks replacement; context validation precedes handoff consumption (ARCH-PURPOSE, ARCH-DRY, ARCH-ORDER). Implementation has not started. Estimate follows approval and the plan-quality gate.

Baseline checks passed: `go test ./cmd/internal/reviewcmd -count=1` and `bash tests/review-resume-test.sh`. Fresh-context plan review identified three gaps: first opens have no round history, retained buffers need byte reconciliation, and context must extend through agent-owned Git effects. The plan now specifies those cases and includes a linked Ariadne producer-instruction change because Pair's xx-fix skill resolves to `../ariadne/construct/local/fix/SKILL.md`. Ariadne's `AGENTS.local.md` was read; `MEMORY.md` is absent. No peer files changed.

Fresh-context re-review approved the revised plan with no remaining blocking gaps. The linked producer-instruction change and recovery-storage regression remain required. Awaiting operator approval of the committed full-flow plan before implementation.

## Revisions

2026-09-28 — Expanded the initial two-step outline into a durable full-flow plan after discovering cross-process activation, unscoped handoffs, and unsafe exit-save behavior. The original Spec and Done when remain the contract; proposed edge-case behavior and verification live in the linked plan pending operator approval.
