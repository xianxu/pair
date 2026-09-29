---
id: 000341
status: working
deps: []
github_issue:
target: review-protocol
created: 2026-09-28
updated: 2026-09-28
estimate_hours: 3.24
card_mirror: '76cb1933ef953584b093d269e2bd56d9bed5585a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-28T20:39:30-07:00
flow: {kind: full, provenance: operator}
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

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration is marked stale by `sdlc estimate-source`, so these values are provisional. The approved detailed plan earns the ×0.2 design discount and 15% design buffer; implementation uses 40% of the v2/v2.1 table, familiarity 1.0 for this existing Go/Lua stack.

Resolver: greenfield single-concern Go module, base design 1.0 and implementation 0.6. Existing Runtime and standard Git/process seams are reused; no novel-stack library assumption. Pane activation/recovery: Lua feature at base design 2.0, implementation 1.5. Round admission and producer integration: Lua feature at base design 1.5, implementation 1.0. Draft activation and end-to-end wiring: Lua feature at base design 1.0, implementation 1.0. Docs and one review boundary use base design 0.2 each, implementation 0.2 and 0.4. The separately tracked Ariadne instruction edit is excluded; Pair's protocol integration tests are included.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.20 impl=0.24
item: lua-neovim design=0.40 impl=0.60
item: lua-neovim design=0.30 impl=0.40
item: lua-neovim design=0.20 impl=0.40
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.16
design-buffer: 0.15
total: 3.24
```


## Log

### 2026-09-28

Filed from the operator's Alt+C workflow questions. Current behavior traced through `nvim/init.lua` (`PairReviewToggle`, target storage), `cmd/internal/reviewcmd/run.go` (readiness and scoped-file discovery), and `nvim/review/init.lua` (reconstruct-on-open). Task capture only; implementation has not started.

### 2026-09-28 — investigation and proposed design

Claimed and entered planning on `000341-branch-review-restore`. Confirmed that a live pane bypasses target/branch checks, readiness picks the first path from any latest review commit, reconstruction reads any latest agent round, and handoff polling unlinks before identity can be checked. `RunOpen` also kills an existing pane whose exit handler may save stale text into the newly checked-out branch.

The proposed full-flow design is in [the durable plan](../plans/000341-branch-review-restore-plan.md). Git owns review identity; the pane owns safe activation; pending work blocks replacement; context validation precedes handoff consumption (ARCH-PURPOSE, ARCH-DRY, ARCH-ORDER). Implementation has not started. Estimate follows approval and the plan-quality gate.

Baseline checks passed: `go test ./cmd/internal/reviewcmd -count=1` and `bash tests/review-resume-test.sh`. Fresh-context plan review identified three gaps: first opens have no round history, retained buffers need byte reconciliation, and context must extend through agent-owned Git effects. The plan now specifies those cases and includes a linked Ariadne producer-instruction change because Pair's xx-fix skill resolves to `../ariadne/construct/local/fix/SKILL.md`. Ariadne's `AGENTS.local.md` was read; `MEMORY.md` is absent. No peer files changed.

Fresh-context re-review approved the revised plan with no remaining blocking gaps. The linked producer-instruction change and recovery-storage regression remain required. Awaiting operator approval of the committed full-flow plan before implementation.

Operator approved execution. Plan-quality rounds PQ-1/PQ-2 required concrete durable recovery and function-level test strategy; both were addressed and the gate passed. Linked producer issue is ariadne#268. Estimate derived after gate passage, as required.

## Revisions

2026-09-28 — Expanded the initial two-step outline into a durable full-flow plan after discovering cross-process activation, unscoped handoffs, and unsafe exit-save behavior. The original Spec and Done when remain the contract; proposed edge-case behavior and verification live in the linked plan pending operator approval.
