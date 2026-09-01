# Couch Start Path Completion Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a directories-only, bounded, asynchronous tab-completion menu to Couch's start-thread path field.

**Architecture:** Keep `ReduceMenu` as the transition authority and `RenderMenuView` as the pure presentation boundary. A batched filesystem seam classifies directory entries in a single cancellable worker; pure query/accumulator logic retains the lexical top 200, and exact frame-instance/generation identities reject stale results. Generalize the existing latest-wins preview scheduler so preview and completion share one scheduling algorithm.

**Tech Stack:** Go, the existing `couchtty` reducer/effect architecture, `os.File.ReadDir`, table-driven Go tests, fake host/filesystem seams.

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

- **`CompletionQuery` / `SplitCompletionPath`** — converts editable path text into the directory to enumerate, the literal editable prefix to preserve, and the basename prefix to match.
  - **Relationships:** One query per explicit completion request; one query produces zero or more directory-entry facts.
  - **DRY rationale:** One source owns empty, relative, absolute, dot, repeated-separator, hidden, and trailing-separator semantics.
  - **Future extensions:** Additional path syntaxes widen this parser without changing reducer or OS traversal.
- **`CompletionAccumulator`** — consumes already-classified entry facts and retains the bytewise lexical top 200 matching directory names plus overflow.
  - **Relationships:** One accumulator per request; it consumes 1:N batches and produces one result.
  - **DRY rationale:** Filtering, ordering, hidden-name policy, and bounding remain one pure decision shared by production enumeration and tests.
  - **Future extensions:** The bound or ordering policy can become explicit input without widening the filesystem seam.
- **`LatestSchedule` / `advanceLatestSchedule`** — generic one-running/one-latest-pending scheduling state used by both start preview and completion.
  - **Relationships:** One schedule per asynchronous operation family; each owns at most one running and one pending request.
  - **DRY rationale:** Reuses the proven preview cancellation/coalescing protocol instead of cloning it for filesystem completion (`ARCH-DRY`).
  - **Future extensions:** Other latest-wins Couch UI effects can wrap the same scheduler.
- **`MenuFrame` / `ReduceMenu`** — owns completion identity, candidates, selection, invalidation, notices, and approved key behavior.
  - **Relationships:** One start frame owns zero or one current completion request and up to 200 candidates; effects/results carry its frame instance plus completion generation.
  - **DRY rationale:** All menu input remains under the existing total reducer rather than adding a parallel controller (`ARCH-PURE`).
  - **Future extensions:** Candidate presentation can gain metadata without changing filesystem ownership.
- **`RenderMenuView`** — renders completion candidates within terminal bounds while preserving cursor placement on the path field.
  - **Relationships:** One rendered view consumes one immutable `MenuState` snapshot.
  - **DRY rationale:** Reuses existing clipping, selected-row, notice, and cursor helpers.
  - **Future extensions:** Candidate annotations remain pure view data.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `DirectoryBatchReader` | `cmd/internal/couchtty/console_completion.go` | new | `os.Open`, batched `ReadDir`, symlink `Info` |
| Console completion worker | `cmd/internal/couchtty/console_completion.go`, `console.go`, `console_menu.go` | modified | goroutine lifecycle, cancellation, reducer result delivery |

- **`DirectoryBatchReader`** — opens a directory and returns at most 128 classified entries per read, checking cancellation between batches.
  - **Injected into:** `Console`; production receives the OS implementation and tests install a stateful fake whose directories, batches, blocking points, symlinks, and failures persist across calls (`ARCH-MOCK`).
  - **Future extensions:** Platform-specific directory metadata can remain behind this seam.
- **Console completion worker** — schedules one running and one latest pending request, drives the batch reader, and returns identity-bound results.
  - **Injected into:** The existing `MenuEffect` dispatch loop and `Console.Run` result loop.
  - **Future extensions:** Progress reporting can be added without changing reducer correctness.

### Operating envelope

- Keystrokes and painting never perform filesystem IO (`ARCH-CONSTRAINTS`).
- One completion worker runs at a time, with one replaceable pending request.
- Each IO batch is at most 128 entries; retained matches are at most 200 names plus overflow.
- Huge directories cost one sequential O(N) scan and bounded O(N log 200) selection; cancellation/supersession is checked between batches.
- External service/network/database constraints are N/A. Network-mounted directories may be slow, but work remains asynchronous and bounded; stale results are inert (`ARCH-PURPOSE`).

## Chunk 1: Pure completion and scheduling core

### Task 1: Specify and implement path parsing and bounded accumulation

**Files:**
- Create: `cmd/internal/couchtty/menu_completion.go`
- Create: `cmd/internal/couchtty/menu_completion_test.go`

- [ ] **Step 1: Write failing table tests for path-query semantics**

Use this exact table (shown with `/`, the supported host separator):

| Input | Directory | EditablePrefix | NamePrefix | IncludeHidden | Immediate | `CompletedPath("src")` |
|-------|-----------|----------------|------------|---------------|-----------|------------------------|
| `""` | `"."` | `""` | `""` | false | `""` | `"src/"` |
| `"."` | `""` | `""` | `""` | false | `"./"` | N/A |
| `".."` | `""` | `""` | `""` | false | `"../"` | N/A |
| `"./"` | `"./"` | `"./"` | `""` | false | `""` | `"./src/"` |
| `"../src"` | `"../"` | `"../"` | `"src"` | false | `""` | `"../src/"` |
| `"/repo/sr"` | `"/repo/"` | `"/repo/"` | `"sr"` | false | `""` | `"/repo/src/"` |
| `"foo//ba"` | `"foo//"` | `"foo//"` | `"ba"` | false | `""` | `"foo//src/"` |
| `"/"` | `"/"` | `"/"` | `""` | false | `""` | `"/src/"` |
| `"~"` | `"."` | `""` | `"~"` | false | `""` | `"~repo/"` for name `"~repo"` |
| `".ca"` | `"."` | `""` | `".ca"` | true | `""` | `".cache/"` |

`Immediate` bypasses IO for exact `.` and `..`; its value is applied through the same reducer completion-acceptance helper as a one-result scan. `CompletedPath` is called only for names that already match `NamePrefix`.

The tests reference the following planned production API and initially fail to
compile with undefined symbols. Do not redeclare these types/functions in the
test file; Step 3 adds them to `menu_completion.go`:

```go
type CompletionQuery struct {
	Directory string
	EditablePrefix string
	NamePrefix string
	IncludeHidden bool
	Immediate string
}

func SplitCompletionPath(path string) CompletionQuery
func (q CompletionQuery) CompletedPath(name string) string
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `go test ./cmd/internal/couchtty -run '^TestSplitCompletionPath' -count=1`

Expected: FAIL because `CompletionQuery` and `SplitCompletionPath` do not exist.

- [ ] **Step 3: Implement the minimal lexical parser/reconstructor**

Use `filepath.Separator` only for the emitted trailing separator and preserve the user's editable prefix. Do not call `Abs`, `EvalSymlinks`, or expand `~`; the OS seam resolves the query directory.

- [ ] **Step 4: Write failing accumulator tests**

The tests reference the following planned production result shape and initially
fail with undefined symbols; Step 6 adds it to `menu_completion.go`:

```go
type CompletionMatches struct {
	Paths []string
	Truncated bool
}
```

- Query prefix `"s"`, batch `{src dir, sample dir, setup file, .secret dir}` yields `Paths == ["sample/", "src/"]`, `Truncated == false`.
- Query prefix `"."`, split across two batches `{.zeta dir, plain dir}` then `{.alpha dir, .env file}` yields `[".alpha/", ".zeta/"]`.
- Query prefix `""`, fed reverse-ordered directories `d200` through `d000` with limit 200, yields exactly `d000/` through `d199/` and `Truncated == true`.
- The same 200 directories with limit 200 yields all 200 and `Truncated == false`; overflow means a 201st matching directory, not merely a full accumulator.

Directory enumeration guarantees unique names, so the accumulator does not add duplicate-defense state.

Reference the following planned production entry/accumulator API from the tests;
do not declare it in `_test.go`:

```go
type CompletionEntry struct { Name string; Directory bool }
type CompletionAccumulator struct { /* private bounded state */ }
func NewCompletionAccumulator(query CompletionQuery, limit int) CompletionAccumulator
func (a *CompletionAccumulator) Add(entries []CompletionEntry)
func (a CompletionAccumulator) Result() CompletionMatches
```

- [ ] **Step 5: Run accumulator tests and verify RED**

Run: `go test ./cmd/internal/couchtty -run '^TestCompletionAccumulator' -count=1`

Expected: FAIL because accumulator symbols do not exist.

- [ ] **Step 6: Implement bounded top-N accumulation**

Maintain at most `limit` names, replacing the current lexical maximum when a smaller matching name arrives after capacity. Set overflow whenever more than `limit` distinct matching directory names exist. Sort only the bounded retained slice for the final result.

- [ ] **Step 7: Run pure completion tests and verify GREEN**

Run: `go test ./cmd/internal/couchtty -run '^(TestSplitCompletionPath|TestCompletionAccumulator)' -count=1`

Expected: PASS.

- [ ] **Step 8: Commit the pure completion core**

```bash
git add cmd/internal/couchtty/menu_completion.go cmd/internal/couchtty/menu_completion_test.go
git commit -m "couch: #160 add bounded directory completion core"
```

### Task 2: Generalize the proven latest-wins scheduler

**Files:**
- Modify: `cmd/internal/couchtty/menu_async.go`
- Modify: `cmd/internal/couchtty/menu_async_test.go`

- [ ] **Step 1: Add failing parity tests for generic scheduling**

Define the internal generic boundary before extraction so it retains full request payloads while comparing only identities:

```go
type latestSchedule[T any] struct {
	Running *T
	Pending *T
	CancelRequested bool
}
type latestScheduleEvent[T any, K comparable] struct {
	Kind latestScheduleEventKind
	Request T
	Finished K
}
type latestScheduleEffect[T any, K comparable] struct {
	Kind latestScheduleEffectKind
	Request T
	Cancel K
}
func advanceLatestSchedule[T any, K comparable](
	state latestSchedule[T],
	event latestScheduleEvent[T, K],
	key func(T) K,
	valid func(K) bool,
) (latestSchedule[T], []latestScheduleEffect[T, K])
```

Use requests `{ID: 1, Payload: "one"}`, `{ID: 2, Payload: "two"}`, and `{ID: 3, Payload: "three"}`. Assert: request 1 starts with its payload intact; duplicate ID 1 emits nothing; request 2 becomes pending and emits cancel 1 once; request 3 replaces pending without another cancel; finish 99 is inert; finish 1 starts request 3 with `"three"`; duplicate finish 1 is inert. Mutate the caller's request and returned state copies after each transition to prove pointer/value ownership is cloned.

Add a wrapper parity table for existing `AdvancePreviewSchedule`: zero generation is rejected; the 1→2→3 request and stale/matching finish sequence yields the exact existing `PreviewScheduleEffect` values; cancellation remains requested until running generation 1 finishes. This is the preservation oracle, not merely a request to retain old assertions.

- [ ] **Step 2: Run scheduler tests and verify RED**

Run: `go test ./cmd/internal/couchtty -run '^Test(AdvancePreviewSchedule|AdvanceLatestSchedule)' -count=1`

Expected: FAIL because the generic scheduler is absent.

- [ ] **Step 3: Extract the generic state transition and keep preview wrappers**

Implement the API above. Make `AdvancePreviewSchedule` translate between its existing public state/event/effect vocabulary and the generic state, using `key(request) == request.Generation` and `valid(generation) == generation != 0`; existing callers retain their API. Completion will use the same transition with `CompletionIdentity{FrameInstance, Generation}` and a validity predicate requiring both fields nonzero.

- [ ] **Step 4: Run scheduler and package tests**

Run: `go test ./cmd/internal/couchtty -run '^Test(AdvancePreviewSchedule|AdvanceLatestSchedule)' -count=1`

Expected: PASS with existing preview behavior unchanged.

- [ ] **Step 5: Commit the scheduler extraction**

```bash
git add cmd/internal/couchtty/menu_async.go cmd/internal/couchtty/menu_async_test.go
git commit -m "couch: #160 share latest-wins async scheduling"
```

## Chunk 2: Reducer interaction and bounded rendering

### Task 3: Add completion identity and approved key transitions

**Files:**
- Modify: `cmd/internal/couchtty/menu.go`
- Modify: `cmd/internal/couchtty/menu_test.go`
- Modify: `cmd/internal/couchtty/menu_async_test.go`

- [ ] **Step 1: Write failing reducer tests for request/result identity**

Tests consume this exact production vocabulary (added in Step 3, not redeclared in
`_test.go`):

```go
type CompletionIdentity struct { FrameInstance, Generation uint64 }
type CompletionRequest struct { Identity CompletionIdentity; Path string }
type CompletionResult struct {
	Identity CompletionIdentity
	Matches CompletionMatches
	Error string
}
// MenuFrame additions:
CompletionRequest CompletionIdentity
CompletionPath string
CompletionPending bool
CompletionCandidates []string
CompletionSelected int
CompletionTruncated bool
// MenuState addition: CompletionSequence uint64
// MenuEffect addition: Completion *CompletionRequest
// MenuEventKind addition: MenuEventCompletionResult
// MenuEvent addition: Completion *CompletionResult
// MenuNotice ownership addition: Completion CompletionIdentity
```

`completionMatches(frame, result)` is true only when the visible frame is
`MenuFrameStart`, both identity fields are nonzero, `result.FrameInstance ==
frame.Instance`, and the whole identity equals `frame.CompletionRequest`.

Use a start frame with `Instance: 2`, `Path: "sr"`. First `Tab` increments the
state sequence from 0 to 1, stores `{2,1}`/`"sr"`/pending, and emits exactly one
`CompletionRequest{{2,1}, "sr"}`. A second `Tab` before a result returns an
equal state and no effect. For each of success, error, no-match, and truncation,
send `{2,0}`, `{2,2}`, and `{3,1}` results and assert deep equality with the
pre-event state plus zero effects. After edit, field change, exit, and reopened
frame instance 3, assert candidates/pending/current identity and an owned notice
are cleared immediately; a `{2,1}` result is inert even if the reopened frame's
numeric generation is 1.

After accepting a multi-candidate result, mutate the caller-owned result slice
and a later returned state's candidate slice; assert the prior `MenuState` and
event input retain their original values. This directly preserves the reducer's
immutable-by-copy contract for the new retained slice.

- [ ] **Step 2: Run identity tests and verify RED**

Run: `go test ./cmd/internal/couchtty -run '^TestReduceMenuStartCompletionIdentity' -count=1`

Expected: FAIL because completion request/result vocabulary is absent.

- [ ] **Step 3: Add reducer vocabulary and exact matching**

Add the exact vocabulary above. Allocate generations from a completion-specific
monotonic sequence on `MenuState`; overflow publishes a fail-safe local error and
emits no request. Never use start-preview grants as completion authority.
Centralize invalidation so edits, field changes, acceptance, and frame exit clear
candidates and notices only when `Notice.Completion` equals the invalidated
identity. Extend `cloneMenuState`/frame cloning to copy
`CompletionCandidates`; never retain the result event's slice. Exact-dot
immediate completion creates no request/pending identity.

- [ ] **Step 4: Write failing interaction tests**

Use these executable cases:

- Path `"."` + `Tab` emits no effect, changes path to `"./"`, invalidates an accepted preview token, and leaves no candidate menu.
- Pending `{2,1}` + exact result `Paths:["src/"]` changes path to `"src/"`, clears pending/candidates/owned notice, and emits no preview or start effect.
- Exact result `Paths:["sample/","src/"]` leaves path `"s"`, candidates in that order, selection 0; `Tab` selects 1; another `Tab` wraps to 0; `Down` selects 1; `Up` selects 0; `Enter` accepts `"sample/"` and closes candidates.
- With candidates visible, `Escape` only clears completion state; the next `Escape` pops the form. With candidates closed, `Down` moves path→agent, `Up` moves agent→path, and neither emits completion. On agent, `Left`/`Right` retain existing selection behavior.
- Exact error leaves path unchanged and publishes an owned error. Exact empty matches publishes owned `"no matching directories"`. Exact `Truncated:true` with candidates opens them and owns the truncation state. Editing clears each immediately.
- With candidates closed, `Enter` continues to request/reuse the ordinary preview token and dispatch start exactly once; completion result fields never populate preview authorization.

- [ ] **Step 5: Run interaction tests and verify RED**

Run: `go test ./cmd/internal/couchtty -run '^TestReduceMenuStartCompletionInteraction' -count=1`

Expected: FAIL on the old Tab-to-agent behavior.

- [ ] **Step 6: Implement the approved reducer transitions**

Keep all behavior in `reduceStartKey` plus focused pure helpers. `Tab` requests only when candidates are closed, cycles only when visible, and coalesces when the same identity/path is pending. `Up`/`Down` switch fields only with no candidate menu. Accepting a candidate updates `Path`, invalidates any start preview token, and leaves the path field focused.

- [ ] **Step 7: Run reducer and preview regressions**

Run: `go test ./cmd/internal/couchtty -run '^(TestReduceMenuStart|TestAdvancePreviewSchedule)' -count=1`

Expected: PASS; update old tests that used `Tab` for field navigation to use `Down`/`Up`, without weakening their assertions.

- [ ] **Step 8: Commit reducer behavior**

```bash
git add cmd/internal/couchtty/menu.go cmd/internal/couchtty/menu_test.go cmd/internal/couchtty/menu_async_test.go
git commit -m "couch: #160 add path completion reducer behavior"
```

### Task 4: Render the candidate menu within the terminal viewport

**Files:**
- Modify: `cmd/internal/couchtty/menu_render.go`
- Modify: `cmd/internal/couchtty/menu_render_test.go`

- [ ] **Step 1: Write failing render tests**

Assert this exact vertical order: breadcrumb, optional notice, blank row, path,
candidate viewport, optional `"  … more directories; type to narrow"`, agent,
and optional args provenance. At 40x10 with a notice, truncation, and args, assert
at least one selected candidate remains visible, both path and agent controls are
present, truncation text is present (clipped by width only), and the cursor row
still resolves to the path edit cell. With 100 candidates and selection 99,
assert the viewport contains candidate 99 and never exceeds terminal height.
Include an ANSI/control/wide-glyph candidate and assert sanitization plus column
bounds.

- [ ] **Step 2: Run render tests and verify RED**

Run: `go test ./cmd/internal/couchtty -run '^TestRenderMenuStartCompletion' -count=1`

Expected: FAIL because the start frame does not render candidates.

- [ ] **Step 3: Extend the existing start-frame renderer**

Build rows with existing `selectedMenuLine`, `clipMenuLine`, and terminal-width
helpers. Before choosing the candidate viewport, reserve rows for breadcrumb,
the already-known optional notice insertion, blank, path, agent, optional args,
and the truncation row whenever `CompletionTruncated` is true. Candidates occupy
only the remaining middle rows between path and truncation/agent, and the
viewport must contain the selected candidate. Truncation has priority over extra
candidates and is always rendered as `"  … more directories; type to narrow"`
on every supported terminal; width clipping is allowed, row omission is not.
Keep `fitMenuBlock` only as a final invariant backstop, not as the row-priority
decision.

- [ ] **Step 4: Run render/cursor tests and verify GREEN**

Run: `go test ./cmd/internal/couchtty -run '^(TestRenderMenuStartCompletion|TestRenderMenuCursorIntent|TestChooseMenuLayout)' -count=1`

Expected: PASS at minimum and ordinary terminal sizes.

- [ ] **Step 5: Commit rendering**

```bash
git add cmd/internal/couchtty/menu_render.go cmd/internal/couchtty/menu_render_test.go
git commit -m "couch: #160 render bounded path candidates"
```

## Chunk 3: Filesystem shell, end-to-end wiring, and documentation

### Task 5: Implement batched filesystem enumeration behind a stateful fake

**Files:**
- Create: `cmd/internal/couchtty/console_completion.go`
- Create: `cmd/internal/couchtty/console_completion_test.go`
- Modify: `cmd/internal/couchtty/console.go`
- Modify: `cmd/internal/couchtty/console_menu.go`

- [ ] **Step 1: Write the stateful fake and failing batch-reader tests**

The fake records directory state, ordered batches (including a final non-empty
batch returned with `io.EOF`), symlink targets, injected open/read/classify/close
errors, blocking/cancellation points, and per-cursor close counts across calls.
Test the production reader separately with `t.TempDir()` containing directories,
regular files, hidden directories, a directory symlink, a file symlink, and a
dangling symlink. Directory symlinks produce `CompletionEntry{Name:
entry.Name(), Directory:true}` so the editable symlink spelling is preserved;
file and dangling/unstatable symlinks are skipped rather than failing the whole
request.

Use the narrow seam:

```go
type DirectoryBatchReader interface {
	Open(context.Context, string) (DirectoryBatchCursor, error)
}
type DirectoryBatchCursor interface {
	Read(context.Context, int) ([]CompletionEntry, error)
	Close() error
}
```

- [ ] **Step 2: Run filesystem seam tests and verify RED**

Run: `go test ./cmd/internal/couchtty -run '^TestOSDirectoryBatchReader' -count=1`

Expected: FAIL because the seam and OS implementation do not exist.

- [ ] **Step 3: Implement the OS reader with complete cleanup**

Open the query directory once and transfer sole cursor ownership to the worker.
Read at most 128 entries per batch and process returned entries before examining
an accompanying `io.EOF`. Classify ordinary entries from `DirEntry`/`Info`; for
every symlink call `os.Stat(filepath.Join(openedDirectory, entry.Name()))` and
mark it a directory only when the resolved target is a directory. Skip
dangling/unstatable symlinks while retaining the symlink's entry name for valid
directory targets. Check `ctx.Err()` before and after each batch and before
symlink classification.

The worker calls `Close` exactly once on every successful-open terminal path:
success, read failure, classification/cancellation exit, and stop. It processes
partial entries first, then joins any non-EOF scan error with the close error
using `errors.Join`; a close-only failure is delivered as the request error.
Open failure owns no cursor and performs no close. Never materialize the entire
directory.

- [ ] **Step 4: Write failing Console scheduling/integration tests**

Prove one running plus one latest pending request, duplicate coalescing,
supersession checks between batches, Stop cancellation and worker join, exact
identity delivery, local error/no-match/truncation notices, and stale results
after edit/exit/reopen remaining inert. For success, partial-batch EOF, read
failure, cancellation, close-only failure, and joined read+close failure, assert
exactly one close, the expected accumulated entries/error chain, and no result
after Console stop. Assert the fake's maximum concurrent cursor count is one and
retained request count is bounded.

- [ ] **Step 5: Run Console completion tests and verify RED**

Run: `go test ./cmd/internal/couchtty -run '^TestConsoleCompletion' -count=1`

Expected: FAIL because Console does not dispatch completion effects.

- [ ] **Step 6: Wire the worker through Console's existing effect/result loop**

Add the injected reader, completion schedule, cancel function, running identity, and buffered result channel to `Console`. Dispatch completion effects beside preview effects, drive the shared latest-wins scheduler, feed batches through `CompletionAccumulator`, send one result, retire the exact running identity, start the latest pending request, repaint only while panel-focused, and include completion workers in the existing `workers` join.

- [ ] **Step 7: Run integration and race tests**

Run: `go test -race ./cmd/internal/couchtty -run '^(TestConsoleCompletion|TestOSDirectoryBatchReader)' -count=1`

Expected: PASS with no race reports, leaked workers, or unbounded concurrency.

- [ ] **Step 8: Commit the filesystem shell**

```bash
git add cmd/internal/couchtty/console_completion.go cmd/internal/couchtty/console_completion_test.go cmd/internal/couchtty/console.go cmd/internal/couchtty/console_menu.go
git commit -m "couch: #160 wire asynchronous directory completion"
```

### Task 6: Exercise the real input path and update user-facing contracts

**Files:**
- Modify: `cmd/internal/couchtty/console_run_menu_test.go`
- Modify: `README.md`
- Modify: `atlas/couch.md`
- Modify: `workshop/issues/000160-couch-path-completion.md`
- Modify: `workshop/plans/000160-couch-path-completion-plan.md`

- [ ] **Step 1: Write a failing live-loop input test**

Through the fake host/stdin and injected directory reader, open the switcher/start form, type a relative prefix, press literal Tab, wait for candidates, navigate with Down/Tab/Enter, verify the path text and cursor, move to agent with Down, then submit through the unchanged preview token flow.

- [ ] **Step 2: Run the input-path test and verify RED**

Run: `go test ./cmd/internal/couchtty -run '^TestConsoleRunStartPathCompletion' -count=1`

Expected: FAIL until the real console loop and render path expose completion end to end.

- [ ] **Step 3: Complete wiring gaps and make the test GREEN**

Fix only integration omissions surfaced by the test; keep completion semantics in the pure reducer and accumulator.

- [ ] **Step 4: Update README and Atlas**

Document directories-only `Tab`, `Up`/`Down` field/candidate behavior, `Left`/`Right` agent behavior, literal relative/absolute path semantics, bounded async enumeration, and the injected filesystem seam in the existing Couch sections. Keep `atlas/index.md` unchanged because `atlas/couch.md` is already linked.

- [ ] **Step 5: Run focused and full verification**

Run:

```bash
go test ./cmd/internal/couchtty -count=1
go test ./... -count=1
git diff --check
```

Expected: all tests PASS and `git diff --check` prints nothing.

- [ ] **Step 6: Update issue evidence and commit**

Tick completed Task 1–6 rows only, append test/results and architectural
decisions to `## Log`, and keep Task 7 unchecked until each boundary/publish
command succeeds. Keep the issue open until the SDLC close gate runs its
mandatory fresh-context review.

```bash
git add README.md atlas/couch.md workshop/issues/000160-couch-path-completion.md cmd/internal/couchtty/console_run_menu_test.go workshop/plans/000160-couch-path-completion-plan.md
git commit -m "couch: #160 document and verify path completion"
```

### Task 7: Cross the SDLC boundary

**Files:**
- Modify as directed by the gate: `workshop/issues/000160-couch-path-completion.md`

- [ ] **Step 1: Inspect the final worktree and verification evidence**

Run: `git status --short`, `git diff main...HEAD --stat`, and `sdlc actual --issue 160`.

- [ ] **Step 2: Close with measured actual time and mandatory boundary review**

Run: `sdlc close --issue 160 --verified 'go test ./cmd/internal/couchtty -count=1; go test ./... -count=1; git diff --check; live-loop fake-host completion coverage passes'`

Expected: the gate measures/adopts actual time, runs its owned fresh-context review, and either closes #160 or returns exact findings to fix before retrying. Use `--no-atlas` only if the gate fails to recognize the already-updated `atlas/couch.md`, and record why in `--verified`; do not use `--force`.

- [ ] **Step 3: Apply the post-verdict protocol and commit the reviewed anchor**

Inspect the printed verdict and `git status --short`. On `FIX-THEN-SHIP`, fix
the full class of every Critical/Important finding before committing, update
`workshop/lessons.md` with prevention rules, append the outcome to the issue
Log, and do not re-run close unless a code fix lands after the close commit. On
SHIP, append the clean outcome. After any finding fix, rerun `go test
./cmd/internal/couchtty -count=1`, `go test ./... -count=1`, and `git diff
--check`; all must pass before commit.

Inspect `git status --short`, then stage only the issue-scoped paths already
named by Tasks 1–6, `workshop/issues/000160-couch-path-completion.md`,
`workshop/plans/000160-couch-path-completion-plan.md`,
`workshop/lessons.md` if changed, and the exact #160 close review/gate sidecars
or project files printed by `sdlc close`. Never use `git add -A`. Tick Task 7
Steps 1–3 and commit the reviewed anchor:

```bash
git add cmd/internal/couchtty/menu_completion.go cmd/internal/couchtty/menu_completion_test.go cmd/internal/couchtty/menu_async.go cmd/internal/couchtty/menu_async_test.go cmd/internal/couchtty/menu.go cmd/internal/couchtty/menu_test.go cmd/internal/couchtty/menu_render.go cmd/internal/couchtty/menu_render_test.go cmd/internal/couchtty/console_completion.go cmd/internal/couchtty/console_completion_test.go cmd/internal/couchtty/console.go cmd/internal/couchtty/console_menu.go cmd/internal/couchtty/console_run_menu_test.go README.md atlas/couch.md workshop/issues/000160-couch-path-completion.md workshop/plans/000160-couch-path-completion-plan.md
git commit -m "couch: #160 close path completion"
```

If `sdlc close` reports additional exact #160 sidecar/project paths, add those
explicit paths to the `git add` invocation. Add `workshop/lessons.md` explicitly
only when findings changed it.

Expected: issue status is `codecomplete`; the tracked worktree is clean.

### Post-publish procedure

The commands below are intentionally non-checkable observations: `sdlc merge`
or `sdlc push` archives this plan during the command, so no truthful later edit
can tick rows in the archived artifact.

**Publish through the branch SDLC path.**

The expected topology after `sdlc change-code` is a feature branch. Run:

```bash
sdlc pr
gh pr checks --watch
sdlc merge --yes
```

If and only if `git branch --show-current` is `main` because the work was
explicitly kept on main, use `sdlc push --yes` instead. Do not substitute raw
`git push`; the SDLC publish gate flips `codecomplete` to `done` and archives
the issue/plan.

**Verify published terminal state.**

Run: `git branch --show-current`, `git status --short`, and `git log -3 --oneline`.

Expected: current branch is `main`, worktree is clean, the merge/push is present,
and the publish gate reported #160 `done` and archived its issue/plan.

## Revisions

### 2026-09-01 — close Chunk 1 plan-review gaps

Added exact path-query fixtures and result shapes, removed unnecessary duplicate
defense, specified top-200 expectations, and defined the payload-preserving
generic latest-wins API plus its preview parity oracle.

### 2026-09-01 — make Chunk 1 RED steps compile-safe

Corrected the literal-tilde fixture and clarified that tests reference planned
production declarations rather than redeclaring conflicting Go symbols.

### 2026-09-01 — close Chunk 2 plan-review gaps

Pinned completion identity, request/result, frame, event/effect, and notice
fields; added executable reducer cases; and fixed render priority to path,
candidates, mandatory truncation, then agent within the bounded viewport.

### 2026-09-01 — preserve completion slice ownership

Required deep cloning of retained candidates and an aliasing regression across
event input, prior reducer state, and later transitions.

### 2026-09-01 — complete Chunk 3 IO and publish lifecycle

Specified target-following symlink classification, partial-EOF processing,
exactly-once cursor close and joined errors; corrected checkbox timing; and
extended the close handoff through verdict fixes, anchor commit, PR/merge (or
explicit main push), and clean published-state verification.

### 2026-09-01 — make Chunk 3 terminal tracking truthful

Required fresh tests after verdict fixes, replaced broad staging with explicit
issue-scoped paths, and made post-archive publish verification explicitly
non-checkable because the publish command moves the plan before a later tick.
