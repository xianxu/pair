# Slot quick-status glyph Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Show one quick-status glyph per slot (`` off-resting, `*` dirty, `+` unpublished, none) in the Couch switcher rows and tab bar, from one derivation, refreshed off the render path.

**Architecture:** One `git --no-optional-locks status --porcelain=v2 --branch` read per slot checkout yields branch, upstream ahead count and dirtiness together; a pure parser turns it into `couchcore.SlotGitStatus`, and a pure `SlotGlyph(status, resting)` applies the precedence. A console-owned single-flight refresh (reusing `RefreshSchedule`) probes the slot paths of the current inventory on a ticker and on switcher-open/switch, and lands the result in `MenuState.SlotGit` through `ReduceMenu`. `PresentThreads` derives `ThreadPresentation.Glyph` once from `MenuState.SlotGit`; the switcher and the tab bar both render that field.

**Tech Stack:** Go (`cmd/internal/couchcore`, `cmd/internal/couchtty`, `cmd/internal/couchcmd`), git porcelain v2.

---

## Design decisions (resolving the issue's open questions)

- **Where the data comes from (revises the Spec).** The Spec assumed `branch`/`resting_branch` already reach presentation via `WorkspaceIdentity`. They do not: `sdlc workspace` runs only on explicit operations, never on the render path, and nothing caches it. Rather than add a second, slower per-slot `sdlc workspace` call, the one porcelain-v2 read supplies `branch.head`, and the resting branch comes from the slot number through `couchcore.RestingBranch(n)`. `WorkspaceIdentity.validate()` switches to the same helper, so the convention that `sdlc workspace` asserts and the one the glyph uses have one owner in couch (ARCH-DRY).
- **Upstream, not hard-coded origin/main.** Porcelain v2's `# branch.ab +A -B` compares against the branch's configured upstream (`main-slotN` and `main` track `origin/main` today). No upstream means no `+`: we have no evidence either way, so we show nothing.
- **`:0` gets the same rules.** In a group that has slots, the primary checkout's ordinary rows (Path == group root) carry the `:0` glyph with resting branch `main`. Ordinary repos without slots get no glyph, because they have no resting-branch concept.
- **Nerd Font glyph, no ASCII fallback (YAGNI).** The glyph is one constant, `slotGlyphBranch`, so a fallback later is a one-line change. The operator's terminal renders U+E0A0 today.
- **Detached HEAD** counts as off-resting (``).
- **Placement.** The glyph goes directly after the label with no space, the same in both views: `pair:1`, `[:2*]`, `pair+`.
- **`--no-optional-locks`** keeps the background status from taking `index.lock` and colliding with the operator's own git command in that slot.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `SlotGitStatus` | `cmd/internal/couchcore/slotgit.go` | new |
| `ParseSlotGitStatus` | `cmd/internal/couchcore/slotgit.go` | new |
| `SlotGlyph` | `cmd/internal/couchcore/slotgit.go` | new |
| `RestingBranch` | `cmd/internal/couchcore/slotgit.go` | new |
| `WorkspaceIdentity.validate` | `cmd/internal/couchcore/workspace_identity.go` | modified (uses `RestingBranch`) |
| `ThreadPresentation.Glyph` / `PresentThreads` | `cmd/internal/couchtty/thread_presentation.go` | modified (takes the status map, derives `Glyph`) |
| `MenuState.SlotGit` + `MenuEventSlotGit` | `cmd/internal/couchtty/menu.go` | modified |
| `StatusActor.Glyph` / `RenderStatusRow` | `cmd/internal/couchtty/reserve.go` | modified |
| `renderRootMenuFrame` | `cmd/internal/couchtty/menu_render.go` | modified |
| `slotGitProbePaths` | `cmd/internal/couchtty/console_slotgit.go` | new |

- **SlotGitStatus** — `{Branch string; Detached, Dirty, HasUpstream bool; Ahead int}`. It is one observation of one checkout.
  - **Relationships:** 1 per checkout path; held in `MenuState.SlotGit map[string]SlotGitStatus`, keyed by worktree root.
  - **DRY rationale:** This is the only git-derived slot fact in couch. Both views read it through `ThreadPresentation.Glyph`.
  - **Future extensions:** A behind-origin count (`branch.ab -B`) or a conflict state are extra fields on the same parse.
- **ParseSlotGitStatus(out string) (SlotGitStatus, error)** parses porcelain v2 with `--branch`. It treats the output as a closed grammar (lessons: Interfaces): a missing `# branch.head` is an error, a malformed `branch.ab` is an error, and unknown `#` headers are ignored. Any `1`/`2`/`u`/`?` entry line means dirty. A `!` (ignored) line cannot appear without `--ignored`, but it is treated as not dirty.
- **SlotGlyph(s SlotGitStatus, resting string) string** is the precedence table: detached or `Branch != resting` → ``; `Dirty` → `*`; `HasUpstream && Ahead > 0` → `+`; otherwise `""`.
- **RestingBranch(n int) string** returns `"main"` for 0 and `"main-slotN"` otherwise.
- **PresentThreads(rows, git map[string]couchcore.SlotGitStatus)** sets `Glyph` for a slot row from `git[Slot.WorktreeRoot]` with `RestingBranch(Slot.Number)`. For an ordinary row in a `hasSlots` group whose `Path == g.root`, it uses `git[g.root]` with `RestingBranch(0)`. It sets `Glyph` only when the map holds an entry, so there is no guessing. `orderedMenuInventory` passes `nil`, since it only orders.
- **slotGitProbePaths(rows) []string** returns the sorted, deduplicated worktree roots of valid slot targets plus their `PrimaryRoot`s. It is pure, and it defines the probe set.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ProbeSlotGit` | `cmd/internal/couchcore/slotgit.go` | new | `GitRunner.RunContext` (git status) |
| `SlotGitProbe` + `Console.SetSlotGitProbe` | `cmd/internal/couchtty/console_slotgit.go` | new | injected probe func |
| slot-git refresh owner (`advanceSlotGit` / `finishSlotGit`, ticker in `Run`) | `cmd/internal/couchtty/console_slotgit.go`, `console.go` | new | goroutine + ticker |
| wiring | `cmd/internal/couchcmd/run.go` (`wireResolver`) | modified | `couchcore.ExecGit` |
| `fakeSlotGitProbe` | `cmd/internal/couchtty/console_slotgit_test.go` | new (stateful fake) | — |

- **ProbeSlotGit(ctx, git GitRunner, dir) (SlotGitStatus, error)** runs `git --no-optional-locks status --porcelain=v2 --branch` in `dir` and calls `ParseSlotGitStatus`. Its tests use the existing `couchcore.FakeGit`, which is keyed by dir+args, so the test proves the directory and the flags.
- **Refresh owner.** `Console.Run` owns `slotGitSchedule RefreshSchedule`, which reuses the pure single-flight machine. It is fed by `slotGitRequests chan struct{}` (non-blocking send, buffer 1) and `slotGitResults chan slotGitResult`, plus a `time.Ticker(slotGitInterval = 10s)` that is stopped by a defer in `Run`. On `RefreshStart`, the loop snapshots the probe paths under `c.mu`, then a goroutine under `c.workers` probes each path sequentially: concurrency 1, each probe under `context.WithTimeout(c.lifetime, slotGitProbeTimeout = 3s)`. It sends one result, guarded by `<-c.stop`. `finishSlotGit` reduces `MenuEventSlotGit` and `repaint()`s. When the panel is focused it also calls `showMenu()`: lessons #307 require the tab bar to repaint even while an idle child holds focus. **Triggers:** the ticker, `onHotkey` (switcher open), `onSwitch`, and `finishMenuRefresh` (the inventory may bring new slot paths).
- **Failure semantics.** The result carries `map[path]SlotGitStatus` for the probes that succeeded and a set of failed paths. The reducer builds the new map over the probe set: a success replaces the entry, a failure keeps the previous entry (and absence stays absent), and a path outside the probe set is dropped. Nothing ever reaches chrome as an error; a failure is only traced.
- **ARCH-FUNERAL.** This work creates nothing durable. `MenuState.SlotGit` is in-memory, bounded by the inventory's slot count, and rebuilt over the current probe set on every pass, so removed slots fall out. The goroutine is joined by `c.workers`, and the ticker is stopped by `Run`'s defer.
- **ARCH-CONSTRAINTS.** Each pass costs N sequential git statuses, each ≤3s, and runs at most once concurrently, plus one dirty follow-up. The render/keystroke path does no IO: `statusModelLocked` and `RenderMenu` only read `MenuState.SlotGit`.
- **fakeSlotGitProbe** is stateful: per-dir scripted replies (`status` or `err`), a `gate chan struct{}` that blocks probes until released, and a call log. Tests use it to prove that the render never blocks while a probe hangs, that a failure keeps the last value, and that success updates and repaints.

## Tasks

### Task 1: pure core in couchcore

**Files:** Create `cmd/internal/couchcore/slotgit.go` and `slotgit_test.go`; Modify `cmd/internal/couchcore/workspace_identity.go` (validate: `"main"` / `"main-slot"+n` → `RestingBranch`).

- [x] Write table tests: `TestSlotGlyphPrecedence` covers detached, off-resting+dirty (branch wins), resting+dirty+ahead (`*` wins), resting+clean+ahead, resting+clean+no-upstream+ahead=0, and resting clean → `""`. `TestParseSlotGitStatus` covers clean-with-upstream, `+3 -0`, no upstream, `(detached)`, each entry kind `1`/`2`/`u`/`?`, and missing `branch.head` plus malformed `branch.ab` as errors. `TestRestingBranch` covers 0 and 3. `TestProbeSlotGitRunsInDirWithNoOptionalLocks` uses `FakeGit` keyed on `{Dir, "--no-optional-locks status --porcelain=v2 --branch"}` and also checks that a cancelled ctx returns an error.
- [x] Run `go test ./cmd/internal/couchcore -run 'SlotGit|SlotGlyph|RestingBranch'` and expect a compile FAIL.
- [x] Implement `slotgit.go`:

```go
package couchcore

// SlotGitStatus is one observation of a checkout, from a single porcelain v2
// status read (pair#317). It is display evidence, never mutation authority.
type SlotGitStatus struct {
	Branch      string
	Detached    bool
	Dirty       bool
	HasUpstream bool
	Ahead       int
}

const slotGlyphBranch = ""

// RestingBranch is the branch a checkout rests on: main for :0, main-slotN
// for slot N. sdlc workspace asserts the same convention (validate()).
func RestingBranch(n int) string {
	if n == 0 {
		return "main"
	}
	return "main-slot" + strconv.Itoa(n)
}

// SlotGlyph: off-resting beats dirty beats unpublished.
func SlotGlyph(s SlotGitStatus, resting string) string {
	switch {
	case s.Detached || s.Branch != resting:
		return slotGlyphBranch
	case s.Dirty:
		return "*"
	case s.HasUpstream && s.Ahead > 0:
		return "+"
	}
	return ""
}

var slotGitStatusArgs = []string{"--no-optional-locks", "status", "--porcelain=v2", "--branch"}

func ProbeSlotGit(ctx context.Context, git GitRunner, dir string) (SlotGitStatus, error) {
	out, err := git.RunContext(ctx, dir, slotGitStatusArgs...)
	if err != nil {
		return SlotGitStatus{}, err
	}
	return ParseSlotGitStatus(out)
}

func ParseSlotGitStatus(out string) (SlotGitStatus, error) {
	var s SlotGitStatus
	sawHead := false
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			sawHead = true
			head := strings.TrimPrefix(line, "# branch.head ")
			if head == "(detached)" {
				s.Detached = true
			} else {
				s.Branch = head
			}
		case strings.HasPrefix(line, "# branch.upstream "):
			s.HasUpstream = true
		case strings.HasPrefix(line, "# branch.ab "):
			var ahead, behind int
			if n, err := fmt.Sscanf(strings.TrimPrefix(line, "# branch.ab "), "+%d -%d", &ahead, &behind); err != nil || n != 2 || ahead < 0 {
				return SlotGitStatus{}, fmt.Errorf("malformed branch.ab %q", line)
			}
			s.Ahead = ahead
		case line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "! "):
		case len(line) > 2 && strings.ContainsRune("12u?", rune(line[0])) && line[1] == ' ':
			s.Dirty = true
		default:
			return SlotGitStatus{}, fmt.Errorf("unknown status line %q", line)
		}
	}
	if !sawHead {
		return SlotGitStatus{}, fmt.Errorf("missing branch.head")
	}
	return s, nil
}
```

- [x] In `validate()`, replace the literals `"main"` (primary check) and `rest := "main-slot" + strconv.Itoa(n)` with `RestingBranch(0)` / `RestingBranch(n)`.
- [x] Run `go test ./cmd/internal/couchcore` and expect PASS (existing workspace-identity tests guard the validate refactor).
- [x] Commit `#317: couchcore: slot git status parse + glyph precedence`.

### Task 2: one derivation, two views

**Files:** Modify `couchtty/thread_presentation.go`, `couchtty/menu.go` (`MenuState.SlotGit`, `MenuEventSlotGit`, the reducer case, and `cloneMenuState` copying the map), `couchtty/menu_render.go`, `couchtty/reserve.go`, `couchtty/console_presentation.go`, `couchtty/menu_reattach.go` (`orderedMenuInventory` passes `nil`), and every test call of `PresentThreads`. Test: `couchtty/grouped_fixture_test.go`, `couchtty/thread_presentation_test.go`.

- [x] Extend `TestGroupedRenderedFixtures` with scenarios `glyph_branch`, `glyph_dirty`, `glyph_ahead`, `glyph_clean` and `glyph_narrow` (width 40). Each sets `state.SlotGit` so that `:0` (`/workspace/pair`, Branch `main`), `:1` and `:2` show the scenario's state on `:1`, and the others are distinct where it helps. Example for `glyph_branch`: `:1` Branch `000317-x`, `:2` Branch `main-slot2` Dirty, `:0` main clean ahead 2. Add a unit test in `thread_presentation_test.go`: a row with no map entry has `Glyph == ""`, an ordinary non-slot repo row gets no glyph even with a map entry at its path, and every ordinary row at the primary root of a slot group gets the `:0` glyph.
- [x] Run `go test ./cmd/internal/couchtty -run 'Grouped|PresentThreads'` and expect FAIL (compile error or missing fixtures).
- [x] Implement:
  - `ThreadPresentation.Glyph string`. `PresentThreads(rows []couchcore.ActionableThreadSummary, git map[string]couchcore.SlotGitStatus)`: in the first loop, for a valid slot row, `if s, ok := git[p.Path]; ok { p.Glyph = couchcore.SlotGlyph(s, couchcore.RestingBranch(slot.Number)) }`. In the second loop, after `p.Path` is resolved for an ordinary row with `g.hasSlots && p.Path == g.root`, do the same with `RestingBranch(0)`.
  - `menuRender`: `entry.Label+entry.Glyph+"  "+detail`, and `PresentThreads(menuRows(state), state.SlotGit)`.
  - `StatusActor.Glyph string`. In `RenderStatusRow`, after the `:N` substitution, `label += a.Glyph` before the spinner and brackets. In `statusModelLocked`, pass `c.menu.SlotGit` and copy `entry.Glyph`. The glyph is plain text; its width is counted by `appendText` via `textwidth`.
  - `MenuEventSlotGit` carries `SlotGit map[string]couchcore.SlotGitStatus` and `SlotGitFailed map[string]bool`. The reducer applies the Integration-points failure semantics over the key set `SlotGit ∪ SlotGitFailed`: it keeps the previous value for a failed path and drops the rest.
- [x] Run `PAIR_UPDATE_GROUPED_FIXTURES=1 go test ./cmd/internal/couchtty -run TestGroupedRenderedFixtures`, then **read each new `testdata/slots_grouped_glyph_*.txt`** and confirm by eye that the switcher and tab bar carry the same glyph per slot and that the narrow layout clips sensibly. Confirm the existing five fixtures are byte-unchanged (`git diff --stat testdata/`).
- [x] Add a reducer test: prior `{a: dirty, b: clean, gone: x}`; the event has success `{a: clean}` and failed `{b}`; the result is `{a: clean, b: clean(prior)}`, `gone` is dropped, and a failed path with no prior entry stays absent.
- [x] Mutation check: delete the `label += a.Glyph` line and confirm a glyph fixture fails; restore. Delete the ordinary-row glyph branch and confirm the `:0` assertion fails; restore.
- [x] Run `go test ./cmd/internal/couchtty` and expect PASS. Commit `#317: couchtty: derive slot glyph once, render in switcher and tab bar`.

### Task 3: background refresh owner + wiring

**Files:** Create `couchtty/console_slotgit.go` and `couchtty/console_slotgit_test.go`; Modify `couchtty/console.go` (fields, channel init next to `refreshResults`, ticker + select cases in `Run`, and trigger calls in `onHotkey` and `onSwitch`), `couchtty/console_menu.go` (`finishMenuRefresh` → `c.requestSlotGit()`), and `couchcmd/run.go` (`wireResolver`: `console.SetSlotGitProbe(func(ctx context.Context, dir string) (couchcore.SlotGitStatus, error) { return couchcore.ProbeSlotGit(ctx, c.Git, dir) })`).

- [x] Write `console_slotgit_test.go` with the stateful `fakeSlotGitProbe` (scripted per dir, gate channel, call log, mutex) and a running console (follow the `Run` harness in `console_run_menu_test.go`) with the grouped inventory from `groupedRow`:
  1. `TestSlotGitRenderNeverBlocksOnSlowProbe`: close the gate only after asserting that, while a probe is blocked (the fake reports it entered), `con.statusModelLocked()` under `c.mu` and `RenderMenu(con.menuSnapshot(), …)` both return within 100ms. Then release, and wait for the tab bar written to the fake host to contain `:1` + the glyph.
  2. `TestSlotGitFailureKeepsLastGlyph`: first pass succeeds (dirty `*`); second pass for the same dir returns an error; `c.menu.SlotGit[path]` is still dirty and no error text appears in the rendered status row.
  3. `TestSlotGitTriggers`: with a long ticker interval (make `slotGitInterval` a Console field set in the test), `onHotkey` and a switch each cause a new probe call, and overlapping requests while one pass runs collapse into one follow-up (reusing `RefreshSchedule`).
  4. `TestSlotGitStopsWithConsole`: stopping the console while a probe is gated returns (the probe ctx is cancelled and the worker joins); the fake records ctx cancellation.
  5. `TestSlotGitProbePaths` (pure): slot worktree roots + primary roots, deduplicated and sorted, invalid slot targets excluded.
- [x] Run and expect FAIL.
- [x] Implement `console_slotgit.go` (`SlotGitProbe` type, `SetSlotGitProbe`, `requestSlotGit`, `advanceSlotGit`, `finishSlotGit`, `slotGitProbePaths`), mirroring `advanceMenuRefresh`/`finishMenuRefresh`. With a nil probe, finish the generation immediately and do nothing. Trace `slotgit` passes via `c.traceEvent` with `ok=N failed=M`.
- [x] Mutation checks: (a) make `advanceSlotGit` call the probe synchronously under `c.mu` and confirm test 1 fails; (b) have the reducer overwrite failed paths with zero values and confirm test 2 fails; (c) remove the `onHotkey` trigger and confirm test 3 fails. Restore each.
- [x] `TMPDIR=<scratchpad> make test` must be fully green (memory: full make test before close). Commit `#317: couch: background slot git refresh`.

### Task 4: docs + live check

- [x] `atlas/couch.md`: describe the slot glyph legend, the refresh owner and the cost envelope, and cross-link #317. Update the README's couch section if it describes the tab bar or switcher row format (grep `pair:1` / `tab bar`).
- [x] `make install`, then ask the operator to smoke test (memory: dogfood live). Move a branch into :1, dirty :2, and commit unpublished work on :0's `main`; the glyphs should update within ~10s or immediately on opening the switcher.
- `sdlc close --issue 317 --verified '…'`.

## Revisions

- 2026-09-24: Implementation tasks verified and checked off. Final validation used make build and live operator smoke rather than repeating installation; full make -k test passed with inherited session variables removed. BR-1 tightens full-field parsing with failing-then-passing malformed-header regressions; the state and display design is unchanged. Close/publication is the remaining workflow action.
