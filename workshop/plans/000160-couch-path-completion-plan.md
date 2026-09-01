# Couch Start Path Completion Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a directories-only, bounded, asynchronous tab-completion menu to Couch's start-thread path field.

**Architecture:** `ReduceMenu` remains the transition authority and `RenderMenuView` the pure presentation boundary. A batched filesystem seam classifies directory entries in one cancellable worker; pure completion logic retains at most 200 lexical matches, and frame/generation identity rejects stale results. Preview and completion share a generalized latest-wins scheduler.

**Tech Stack:** Go, the existing `couchtty` reducer/effect architecture, `os.File.ReadDir`, unit tests, fake host/filesystem integration tests.

**Non-goals:** File completion, fuzzy ranking, shell expansion, recursive indexing/cache, cursor-in-the-middle editing, and changes to start preview/token authorization.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `CompletionQuery` / `SplitCompletionPath` | `cmd/internal/couchtty/menu_completion.go` | new |
| `CompletionAccumulator` | `cmd/internal/couchtty/menu_completion.go` | new |
| `LatestSchedule` / `advanceLatestSchedule` | `cmd/internal/couchtty/menu_async.go` | new |
| `MenuFrame` / `ReduceMenu` | `cmd/internal/couchtty/menu.go` | modified |
| `RenderMenuView` | `cmd/internal/couchtty/menu_render.go` | modified |

- **`CompletionQuery` / `SplitCompletionPath`** — owns lexical interpretation and reconstruction of editable directory paths.
- **`CompletionAccumulator`** — consumes classified batches and retains the deterministic lexical top 200 plus overflow.
- **`LatestSchedule` / `advanceLatestSchedule`** — payload-preserving one-running/one-latest-pending transition shared by preview and completion (`ARCH-DRY`).
- **`MenuFrame` / `ReduceMenu`** — owns identity, candidates, selection, invalidation, notice ownership, and approved keys; retained slices preserve immutable-by-copy state (`ARCH-PURE`).
- **`RenderMenuView`** — owns bounded candidate viewport, control priority, truncation visibility, and cursor intent.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `DirectoryBatchReader` | `cmd/internal/couchtty/console_completion.go` | new | `os.Open`, bounded `ReadDir`, target-following `Stat` |
| Console completion worker | `cmd/internal/couchtty/console_completion.go`, `console.go`, `console_menu.go` | modified | scheduling, cancellation, result delivery, teardown |

- **`DirectoryBatchReader`** — produces bounded directory facts while preserving valid directory-symlink names. Production uses OS IO; integration tests use a stateful fake behind the same seam (`ARCH-MOCK`).
- **Console completion worker** — owns one running scan, one replaceable pending request, cursor cleanup, joined failures, reducer delivery, repaint, and worker join.

### Operating envelope

This is a keystroke UI path: input and paint perform no IO; one worker plus one pending request bounds concurrency; 128-entry batches and 200 retained names bound memory; a huge directory is one asynchronous O(N) scan with cancellation checks between batches (`ARCH-CONSTRAINTS`). Completion remains advisory and cannot grant start authority (`ARCH-PURPOSE`). Network latency may occupy the single worker but cannot block UI or multiply work. External service/database constraints are N/A.

## Chunk 1: Pure completion core

### Task 1: Lexical completion

**Files:** Create `cmd/internal/couchtty/menu_completion.go`; create `cmd/internal/couchtty/menu_completion_test.go`.

- [x] **RED — `SplitCompletionPath` / `CompletionQuery.CompletedPath`:** arbitrary path text → preserve literal editable spelling while producing one bounded lookup query; guard exact-dot navigation with explicit non-IO completion.
- [x] Run `go test ./cmd/internal/couchtty -run '^TestSplitCompletionPath' -count=1`; expect failure for absent behavior.
- [x] Implement the named pure functions.
- [x] **RED — `CompletionAccumulator.Add/Result`:** arbitrary classified batches → deterministic directory-only lexical output; guard memory with a fixed-cap top-N structure and overflow bit.
- [x] Run `go test ./cmd/internal/couchtty -run '^TestCompletionAccumulator' -count=1`; expect failure for absent behavior.
- [x] Implement the named pure functions.
- [x] Run `go test ./cmd/internal/couchtty -run '^(TestSplitCompletionPath|TestCompletionAccumulator)' -count=1`; expect PASS.
- [x] Commit the two files as `couch: #160 add bounded directory completion core`.

### Task 2: Latest-wins scheduling

**Files:** Modify `cmd/internal/couchtty/menu_async.go`; modify `cmd/internal/couchtty/menu_async_test.go`.

- [x] **RED — `advanceLatestSchedule`:** adversarial request/finish ordering and caller mutation → preserve payload and value ownership; guard concurrency with one running slot, one replaceable pending slot, and one cancellation latch. Existing `AdvancePreviewSchedule` behavior is the parity oracle.
- [x] Run `go test ./cmd/internal/couchtty -run '^Test(AdvancePreviewSchedule|AdvanceLatestSchedule)' -count=1`; expect missing generic behavior.
- [x] Extract the generic transition and retain preview's existing wrapper API.
- [x] Run the same command; expect PASS.
- [x] Commit both files as `couch: #160 share latest-wins async scheduling`.

## Chunk 2: Reducer and renderer

### Task 3: Completion interaction state

**Files:** Modify `cmd/internal/couchtty/menu.go`, `menu_test.go`, and `menu_async_test.go`.

- [x] **RED — `ReduceMenu` completion results:** adversarial asynchronous identity and slice ownership → stale work is inert and prior states remain immutable; guard mutation with exact frame/generation matching, owned notices, centralized invalidation, and deep cloning.
- [x] Run `go test ./cmd/internal/couchtty -run '^TestReduceMenuStartCompletionIdentity' -count=1`; expect missing completion vocabulary.
- [x] Implement completion request/result state and identity handling.
- [x] **RED — `reduceStartKey`:** arbitrary approved start-form key sequences → completion, field navigation, agent navigation, escape, and start submission remain unambiguous; guard authorization by leaving preview-token dispatch as the only start path.
- [x] Run `go test ./cmd/internal/couchtty -run '^TestReduceMenuStartCompletionInteraction' -count=1`; expect old key behavior.
- [x] Implement completion key transitions and update affected legacy expectations.
- [x] Run `go test ./cmd/internal/couchtty -run '^(TestReduceMenuStart|TestAdvancePreviewSchedule)' -count=1`; expect PASS.
- [x] Commit the three files as `couch: #160 add path completion reducer behavior`.

### Task 4: Candidate presentation

**Files:** Modify `cmd/internal/couchtty/menu_render.go` and `menu_render_test.go`.

- [x] **RED — `RenderMenuView`:** adversarial candidate text/count and terminal geometry → preserve controls, cursor, selection, truncation, and terminal bounds; guard layout with reserved fixed rows plus a selected-item viewport.
- [x] Run `go test ./cmd/internal/couchtty -run '^TestRenderMenuStartCompletion' -count=1`; expect absent candidates.
- [x] Implement the bounded start-frame candidate renderer using existing sanitation/clipping helpers.
- [x] Run `go test ./cmd/internal/couchtty -run '^(TestRenderMenuStartCompletion|TestRenderMenuCursorIntent|TestChooseMenuLayout)' -count=1`; expect PASS.
- [x] Commit both files as `couch: #160 render bounded path candidates`.

## Chunk 3: IO shell and delivery

### Task 5: Filesystem completion worker

**Files:** Create `cmd/internal/couchtty/console_completion.go` and `console_completion_test.go`; modify `console.go` and `console_menu.go`.

- [x] **RED — OS `DirectoryBatchReader`:** adversarial filesystem entry kinds and IO outcomes → emit only navigable directory facts without whole-directory retention; guard classification with target-following stat and each opened cursor with one close owner.
- [x] Run `go test ./cmd/internal/couchtty -run '^TestOSDirectoryBatchReader' -count=1`; expect absent seam.
- [x] Implement the bounded OS reader.
- [x] **RED — Console completion worker:** adversarial scheduling, cancellation, teardown, and error interleavings → one result/cleanup outcome per identity; guard concurrency with the shared scheduler and cleanup with joined terminal errors plus worker join.
- [x] Run `go test ./cmd/internal/couchtty -run '^TestConsoleCompletion' -count=1`; expect absent wiring.
- [x] Wire the reader/worker through Console's existing effect, result, repaint, and teardown paths.
- [x] Run `go test -race ./cmd/internal/couchtty -run '^(TestConsoleCompletion|TestOSDirectoryBatchReader)' -count=1`; expect PASS without races or leaks.
- [x] Commit the four files as `couch: #160 wire asynchronous directory completion`.

### Task 6: End-to-end behavior and docs

**Files:** Modify `cmd/internal/couchtty/console_run_menu_test.go`, `README.md`, `atlas/couch.md`, the issue, and this plan.

- [x] **RED — Console run-loop completion:** arbitrary real key/event timing through fake stdin/host/filesystem → visible completion remains responsive and start remains token-authorized; guard the product boundary with the stateful fake and final rendered/reducer observations.
- [x] Run `go test ./cmd/internal/couchtty -run '^TestConsoleRunStartPathCompletion' -count=1`; expect missing end-to-end behavior.
- [x] Close integration gaps only at their owning seams.
- [x] Update README and `atlas/couch.md` with the shipped keys, behavior, bounds, and seam.
- [x] Run `go test ./cmd/internal/couchtty -count=1`, `go test ./... -count=1`, and `git diff --check`; expect PASS/clean.
- [x] Tick completed Task 1–6 rows, append issue verification/decisions, and commit as `couch: #160 document and verify path completion`.

### Task 7: SDLC close and publish

- [x] Inspect `git status --short`, `git diff main...HEAD --stat`, and `sdlc actual --issue 160`.
- [x] Run `sdlc close --issue 160 --verified 'go test ./cmd/internal/couchtty -count=1; go test ./... -count=1; git diff --check; live-loop fake-host completion coverage passes'`.
- [x] Apply the printed post-verdict protocol, rerun full verification after fixes, explicitly stage only inspected issue paths/sidecars, and commit the reviewed `codecomplete` anchor as `couch: #160 close path completion`.

Post-publish commands are non-checkable because publishing archives this plan: run `sdlc pr`, `gh pr checks --watch`, and `sdlc merge --yes` on the expected feature branch; use `sdlc push --yes` only for an explicitly retained main topology. Verify main, clean status, published commit, and gate-reported `done`/archive state.

## Revisions

### 2026-09-01 — state the executable-test-strategy rule

Plan-quality PQ-1 remained open after compact case inventories survived the
first revision. Applied its class-wide rule to every Task 1–6 test surface:
`named function → one adversarial input class → one mechanical guard`. Removed
all fixture enumerations, state matrices, field inventories, and procedural diff
instructions while retaining exact RED/GREEN commands and architectural owners.

### 2026-09-01 — close-review IO contract corrections

Boundary review BR-1/BR-2 found that production discarded directory-close
errors and the stateful fake did not enforce its declared batch bound. Added an
injected close-capable directory cursor with joined errors, made the fake emit
bounded observable chunks and errors, and pinned both contracts plus local
Console error preservation under the race detector (`ARCH-MOCK`).

### 2026-09-01 — preserve joined failures during cancellation

Close-review round two showed BR-1 remained open because worker cancellation
filtering discarded an entire joined error. Replaced broad `errors.Is`
suppression with cancellation-leaf removal and added a Console regression where
cancellation and close failure arrive together.
