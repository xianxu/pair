# Couch Start Path Completion Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a directories-only, bounded, asynchronous tab-completion menu to Couch's start-thread path field.

**Architecture:** `ReduceMenu` remains the transition authority and `RenderMenuView` the pure presentation boundary. A batched filesystem seam classifies directory entries in one cancellable worker; pure query/accumulator logic retains the lexical top 200, and frame-instance/generation identities reject stale results. The existing latest-wins preview schedule is generalized so completion does not duplicate its concurrency protocol.

**Tech Stack:** Go, existing `couchtty` reducer/effect architecture, `os.File.ReadDir`, table-driven unit tests, fake host/filesystem integration tests.

**Non-goals:** No file completion, fuzzy ranking, shell/tilde expansion, recursive indexing/cache, cursor-in-the-middle editing, or changes to start preview/token authorization.

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

- **`CompletionQuery` / `SplitCompletionPath`** — splits editable text into an OS lookup directory, preserved editable prefix, basename prefix, hidden-name policy, and exact-dot immediate completion. One query owns one request. This is the only source for empty/relative/absolute/dot/repeated-separator/literal-tilde semantics.
- **`CompletionAccumulator`** — consumes classified entry batches and retains the bytewise lexical first 200 matching directory paths plus overflow. One accumulator consumes 1:N batches without retaining full directory input.
- **`LatestSchedule` / `advanceLatestSchedule`** — payload-preserving generic one-running/one-latest-pending transition used by preview and completion (`ARCH-DRY`). Each wrapper supplies a comparable identity and fail-safe validity predicate.
- **`MenuFrame` / `ReduceMenu`** — owns completion request identity, candidates, selection, truncation, invalidation, notice ownership, and the approved key transitions. One start frame owns zero or one current request and up to 200 candidates (`ARCH-PURE`).
- **`RenderMenuView`** — renders `path → candidate viewport → mandatory truncation row → agent` while reserving notice/control rows and keeping the selected candidate and path cursor visible.

Every retained slice remains immutable-by-copy: `cloneMenuState` copies candidates, and results never donate caller-owned storage.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `DirectoryBatchReader` | `cmd/internal/couchtty/console_completion.go` | new | `os.Open`, batched `ReadDir`, target-following `Stat` |
| Console completion worker | `cmd/internal/couchtty/console_completion.go`, `console.go`, `console_menu.go` | modified | cancellation, scheduling, reducer delivery, worker join |

- **`DirectoryBatchReader`** — opens one directory and returns at most 128 classified entries per read. Directory symlinks keep their entry name; file/dangling/unstatable symlinks are skipped. Production uses the OS implementation; integration tests install a stateful fake with persistent directory/batch/error/blocking/close state (`ARCH-MOCK`).
- **Console completion worker** — owns exactly-once cursor close, joined scan/close errors, one running request and one replaceable pending request, cancellation checks between batches, result delivery, repaint, and teardown join.

### Operating envelope

- Keystrokes and paint never perform filesystem IO (`ARCH-CONSTRAINTS`).
- One completion worker runs with one replaceable pending request.
- IO batches hold at most 128 entries; pure state retains at most 200 names plus overflow.
- A huge directory costs one O(N) sequential scan and bounded O(N log 200) selection; supersession/cancellation is checked between batches.
- Network-mounted directory latency is asynchronous and may occupy the single worker, but cannot block input/paint or create unbounded work. External service/database constraints are N/A.
- Completion is advisory and never grants start authority (`ARCH-PURPOSE`).

## Chunk 1: Pure completion and scheduling core

### Task 1: Path query and bounded accumulation

**Files:**
- Create: `cmd/internal/couchtty/menu_completion.go`
- Create: `cmd/internal/couchtty/menu_completion_test.go`

- [ ] **Write RED tests for `SplitCompletionPath` and `CompletionQuery.CompletedPath`.** Strategy: adversarial lexical forms (empty, dot/dot-dot, relative/absolute, repeated/trailing separators, hidden prefix, literal tilde) must preserve editable spelling; exact dot forms use an explicit immediate result rather than IO.
- [ ] **Run RED:** `go test ./cmd/internal/couchtty -run '^TestSplitCompletionPath' -count=1` — expected undefined production symbols or assertion failure for missing behavior.
- [ ] **Implement the minimal pure query/reconstruction API** in `menu_completion.go`; do not resolve, canonicalize, or expand paths.
- [ ] **Write RED tests for `CompletionAccumulator.Add/Result`.** Strategy: arbitrary batch order, non-directories, hidden entries, and over-capacity input must yield deterministic case-sensitive lexical paths while retaining no more than the configured limit and reporting only true overflow.
- [ ] **Run RED:** `go test ./cmd/internal/couchtty -run '^TestCompletionAccumulator' -count=1` — expected missing behavior.
- [ ] **Implement bounded top-N accumulation** and reconstructed trailing-separator paths.
- [ ] **Run GREEN:** `go test ./cmd/internal/couchtty -run '^(TestSplitCompletionPath|TestCompletionAccumulator)' -count=1` — expected PASS.
- [ ] **Commit:** `git add cmd/internal/couchtty/menu_completion.go cmd/internal/couchtty/menu_completion_test.go && git commit -m "couch: #160 add bounded directory completion core"`.

### Task 2: Shared latest-wins schedule

**Files:**
- Modify: `cmd/internal/couchtty/menu_async.go`
- Modify: `cmd/internal/couchtty/menu_async_test.go`

- [ ] **Write RED tests for `advanceLatestSchedule`.** Strategy: duplicate, replacement, stale-finish, zero/invalid identity, and caller-mutation sequences must preserve full request payloads, emit one cancellation, retain one latest pending request, and clone pointer-owned state. Existing `AdvancePreviewSchedule` outcomes are the parity oracle.
- [ ] **Run RED:** `go test ./cmd/internal/couchtty -run '^Test(AdvancePreviewSchedule|AdvanceLatestSchedule)' -count=1` — expected missing generic transition.
- [ ] **Extract the generic pure transition** keyed by caller-supplied identity/validity functions; retain the existing preview vocabulary as a wrapper.
- [ ] **Run GREEN:** same command — expected PASS with preview behavior unchanged.
- [ ] **Commit:** `git add cmd/internal/couchtty/menu_async.go cmd/internal/couchtty/menu_async_test.go && git commit -m "couch: #160 share latest-wins async scheduling"`.

## Chunk 2: Reducer interaction and bounded rendering

### Task 3: Identity-bound reducer behavior

**Files:**
- Modify: `cmd/internal/couchtty/menu.go`
- Modify: `cmd/internal/couchtty/menu_test.go`
- Modify: `cmd/internal/couchtty/menu_async_test.go`

- [ ] **Write RED tests for `ReduceMenu` completion identity/invalidation.** Strategy: stale success/error/no-match/truncation across edits, field changes, exit/reopen, generation reuse, and sequence overflow must be inert unless both frame instance and generation match; result/candidate slices must not alias prior state or event input.
- [ ] **Run RED:** `go test ./cmd/internal/couchtty -run '^TestReduceMenuStartCompletionIdentity' -count=1` — expected missing request/result vocabulary.
- [ ] **Add completion request/result, frame state, event/effect, sequence, exact-match, owned-notice, clone, and centralized invalidation vocabulary** without coupling it to preview grants.
- [ ] **Write RED tests for `reduceStartKey`.** Strategy: immediate/single/multiple matches and every approved key must prove `Tab` request/cycle/coalesce, candidate versus field `Up`/`Down`, candidate `Enter`, two-level `Escape`, agent `Left`/`Right`, and unchanged Enter-to-preview/start authority.
- [ ] **Run RED:** `go test ./cmd/internal/couchtty -run '^TestReduceMenuStartCompletionInteraction' -count=1` — expected old Tab-to-agent failures.
- [ ] **Implement reducer transitions** and update legacy start-form tests from Tab field movement to Up/Down without weakening their assertions.
- [ ] **Run GREEN/regression:** `go test ./cmd/internal/couchtty -run '^(TestReduceMenuStart|TestAdvancePreviewSchedule)' -count=1` — expected PASS.
- [ ] **Commit:** `git add cmd/internal/couchtty/menu.go cmd/internal/couchtty/menu_test.go cmd/internal/couchtty/menu_async_test.go && git commit -m "couch: #160 add path completion reducer behavior"`.

### Task 4: Candidate rendering

**Files:**
- Modify: `cmd/internal/couchtty/menu_render.go`
- Modify: `cmd/internal/couchtty/menu_render_test.go`

- [ ] **Write RED tests for `RenderMenuView`.** Strategy: long/untrusted/wide candidate sets at minimum and ordinary terminal sizes must keep path/agent controls, selected candidate, cursor, column/row bounds, and mandatory truncation text coherent with and without notices/args.
- [ ] **Run RED:** `go test ./cmd/internal/couchtty -run '^TestRenderMenuStartCompletion' -count=1` — expected absent candidate rows.
- [ ] **Implement reserved-row budgeting and selected-candidate viewport** using existing sanitation, clipping, selection, and cursor helpers; truncation displaces candidate capacity, never controls.
- [ ] **Run GREEN/regression:** `go test ./cmd/internal/couchtty -run '^(TestRenderMenuStartCompletion|TestRenderMenuCursorIntent|TestChooseMenuLayout)' -count=1` — expected PASS.
- [ ] **Commit:** `git add cmd/internal/couchtty/menu_render.go cmd/internal/couchtty/menu_render_test.go && git commit -m "couch: #160 render bounded path candidates"`.

## Chunk 3: Filesystem shell, end-to-end wiring, and documentation

### Task 5: Batched filesystem worker

**Files:**
- Create: `cmd/internal/couchtty/console_completion.go`
- Create: `cmd/internal/couchtty/console_completion_test.go`
- Modify: `cmd/internal/couchtty/console.go`
- Modify: `cmd/internal/couchtty/console_menu.go`

- [ ] **Write RED production-seam tests for the OS `DirectoryBatchReader`.** Strategy: real temporary directories containing files, directories, directory/file/dangling symlinks, partial EOF, and metadata failures must return only navigable directory names in batches no larger than 128 while preserving symlink spelling.
- [ ] **Run RED:** `go test ./cmd/internal/couchtty -run '^TestOSDirectoryBatchReader' -count=1` — expected absent seam.
- [ ] **Implement the OS cursor** with target-following `Stat`, context checks, and no whole-directory materialization.
- [ ] **Write RED integration tests for Console completion scheduling/cleanup.** Strategy: the stateful fake drives duplicate/superseding/blocking requests and every success/read/classification/cancel/close/join terminal path; assert one active cursor, one pending request, exactly one close, joined errors, worker join, and identity-bound reducer delivery.
- [ ] **Run RED:** `go test ./cmd/internal/couchtty -run '^TestConsoleCompletion' -count=1` — expected missing dispatch/result wiring.
- [ ] **Wire completion through Console's existing effect/result/run/teardown loop**, using the shared latest-wins transition and pure accumulator. Process entries before accompanying EOF; close once after successful open; deliver `errors.Join(scan, close)`; do not deliver after stop.
- [ ] **Run GREEN/race:** `go test -race ./cmd/internal/couchtty -run '^(TestConsoleCompletion|TestOSDirectoryBatchReader)' -count=1` — expected PASS without races/leaks.
- [ ] **Commit:** `git add cmd/internal/couchtty/console_completion.go cmd/internal/couchtty/console_completion_test.go cmd/internal/couchtty/console.go cmd/internal/couchtty/console_menu.go && git commit -m "couch: #160 wire asynchronous directory completion"`.

### Task 6: Real input path, docs, and verification

**Files:**
- Modify: `cmd/internal/couchtty/console_run_menu_test.go`
- Modify: `README.md`
- Modify: `atlas/couch.md`
- Modify: `workshop/issues/000160-couch-path-completion.md`
- Modify: `workshop/plans/000160-couch-path-completion-plan.md`

- [ ] **Write RED test for the real Console input loop.** Strategy: literal terminal keys through fake stdin/host plus the stateful directory seam must expose candidates, navigate/accept them, move to agent, retain the path cursor, and submit only through the unchanged preview token flow.
- [ ] **Run RED:** `go test ./cmd/internal/couchtty -run '^TestConsoleRunStartPathCompletion' -count=1` — expected missing end-to-end behavior.
- [ ] **Close only integration gaps surfaced by the test;** keep semantics in pure reducer/accumulator functions.
- [ ] **Update README and `atlas/couch.md`** with keys, directories-only/literal path behavior, async bounds, and filesystem seam; `atlas/index.md` already links Couch.
- [ ] **Run full verification:** `go test ./cmd/internal/couchtty -count=1`, `go test ./... -count=1`, `git diff --check` — expected all PASS/clean.
- [ ] **Tick completed issue/plan Task 1–6 rows, append verification/architecture decisions to the issue Log, and commit** all named Task 6/bookkeeping paths with `couch: #160 document and verify path completion`.

### Task 7: SDLC close and publish

- [ ] **Inspect:** `git status --short`, `git diff main...HEAD --stat`, `sdlc actual --issue 160`.
- [ ] **Close:** `sdlc close --issue 160 --verified 'go test ./cmd/internal/couchtty -count=1; go test ./... -count=1; git diff --check; live-loop fake-host completion coverage passes'`.
- [ ] **Apply the printed post-verdict protocol.** Fix the full class of every blocking finding, update `workshop/lessons.md`, rerun both test commands plus `git diff --check`, stage only explicitly inspected issue-scoped paths/sidecars, and commit the reviewed `codecomplete` anchor as `couch: #160 close path completion`.

Post-publish commands are deliberately non-checkable because merge/push archives this plan during the command: on the expected feature branch run `sdlc pr`, `gh pr checks --watch`, and `sdlc merge --yes`; only on an explicitly retained main topology use `sdlc push --yes`. Verify current branch `main`, clean status, published commit, and the gate-reported `done`/archive state.

## Revisions

### 2026-09-01 — compress to executable test strategies

Plan-quality finding PQ-1 (`executable-test-strategy`) applied across Tasks 1–6:
removed fixture tables, case-by-case assertions, field inventories, and
procedural diff restatements. Retained named functions, one adversarial-input
class/mechanical guard per risky surface, architectural ownership, observable
acceptance, and exact RED/GREEN/commit commands.
