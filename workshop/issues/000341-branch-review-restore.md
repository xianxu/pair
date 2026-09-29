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

- [x] Design branch-to-document resolution and safe pane/pending-round transitions.
- [x] Implement restoration with regression coverage and update the review-workbench atlas.

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

### 2026-09-28 — implementation checkpoint

Implemented bounded Git identity/receipts, pane-owned RPC activation, async Alt+C resolution, scoped record/definition/landed context, bounded atomic recovery, and producer contract guards. Linked ariadne#268 closed with SHIP on `5484941c591c` (not yet published). Review documentation updated. Baselines and current targeted verification: package race tests, Lua suite, full `make test-review`, and real draft + review process A → B → A restoration passed. Tests prove old activation handoffs are preserved, modified buffers refuse switching, completed rounds unblock, and mismatched exit snapshots recover in a new process without touching the wrong checkout. Three source mutations (resolver bypass, pending guard bypass, handoff admission bypass) each failed the production-path end-to-end assertion; originals restored from byte copies. Fresh-session launcher and peer explicit-selection regressions are being completed before final full verification and SDLC boundary review.

### 2026-09-28 — final verification before boundary

`go test -race ./cmd/internal/reviewcmd -count=1`, `make test-lua`, `make test-review`, `make build`, and `git diff --check` passed. Full review suite includes fresh-session actual opener/RPC tests, real draft-to-review A → B → A, unsafe retained-buffer writes, producer branch-race tests, first-open receipts, malformed/context-mismatched payload preservation and scoped history repaint. An additional temporary-source mutation bypassing fresh-open acknowledgment passed the normal case but failed the wrong-session publication assertion. Recovery write failure injection verifies ordinary quit keeps unsaved text in memory; normal storage then preserves it across forced exit and restart. Explicit forced quit can ignore autocmd errors if recovery storage fails; this native limitation is documented rather than claimed safe. All implementation steps complete; running the single SDLC boundary review next.

### 2026-09-28 — boundary review REWORK

Close round 1 reported three blocking families: consume-after-acceptance (handoff deleted before a later authorization refusal), nonblocking-editor-observation (TextChanged performs synchronous history resolution), and user-surface-documentation (README omits restore/recovery). It also found activation-resource-ownership (render autocmds accumulate). Verified each against production code; appended regression/fix scope to the durable plan. User requested continuation after being informed that smoke testing should wait for the fixes.

### 2026-09-28 — boundary findings fixed and verified

BR-1: handoff consumption now follows explicit apply/defer acceptance; captured activation is rechecked at the final effect, callback reentry is suppressed, uncertain partial application stays preserved without automatic replay, accepted cleanup failures never reapply, and deferred rounds survive apply failure. Shared artifact generation receipts protect handoff and definition replacements; refused definition selections retain their response. BR-2: one owned asynchronous observer coalesces typing/focus events, cancels stale work, preserves snapshots under the original binding and refreshes only matching clean buffers; empty definition polls perform no history scan. BR-3: README documents branch restoration and recovery. BR-4: activation owns a clearable rendering group.

Fresh verification: `go test -race ./cmd/internal/reviewcmd -count=1`, `make test-lua`, and `make test-review` pass. The final deferred-failure adjustment also passed the handoff acceptance unit and real review-window suite. `make build`, forced `make -B bin/pair` (including generated embedded assets), and `git diff --check` pass. Delayed-resolver regression reproduced ~1.1-second blocking before the fix; now edit/focus/empty-definition callbacks return below 250ms while a one-second resolver runs, bursts coalesce, same-branch external edits reload and other-branch bytes do not. Rendering counts stay stable through repeated A → B → A. Running the next SDLC boundary round; leaving the issue branch available for the operator's requested smoke test.

### 2026-09-28 — smoke feedback and second boundary

Round 2 disposed all previous findings but raised BR-5: async focus refresh read current disk bytes after accepting stale identity evidence. Fixing with identity-bound snapshots and controlled late completion tests. Operator smoke found hidden pane return refused with “could not query review visibility.” Installed Zellij help confirms hidden is stdout false / exit 1; the implementation and all relevant hidden-state fakes incorrectly treated it as exit 0. Red client regression reproduced, corrected strict predicate decoding passes; real-process fresh restore now starts an agent request, hides the same pane, then reopens it successfully without replacing it or clearing pending work. Hot-reloaded only restore_client methods into the idle draft paired with the operator's sole active review pane, preserving opts/agent/pending round; no full editor restart.
