# couch: spawn and registry — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** couch can register, list, inspect and spawn agent actors keyed on a working tree, refusing a second agent on an occupied tree with a loud override, exposing every operation through one declared surface.

**Architecture:** A new binary `cmd/couch` over a new domain package `cmd/internal/couchcore`. Actors are goroutines holding a bounded mailbox; the agent harness is a **child `pair` process** inheriting couch's stdio (host-pair-whole; the pty arrives with `#146`). Identity is a canonicalised worktree path. All IO sits behind injected seams — process, path, git, store, clock, ids — so the domain is unit-testable without processes, disk, wall-clock, or randomness (ARCH-PURE).

**Tech Stack:** Go 1.26, `os/exec`, `encoding/json` (JSONL), stdlib only. Reuses `cmd/internal/launcher` (`ResolveRepoScope`, `ResolveDataDir`) and `cmd/internal/osfs`.

**Issue:** pair#145. **Project:** `workshop/projects/couch.md`.

**Code in this plan is syntax-checked.** All 11 Go blocks pass `gofmt -e`. Where a task shows a signature instead of a body, that is marked *spec, not listing* and the body is described precisely.

**Import blocks are shown only on the first test file of a package** (Task 1) and on implementation listings. Later test snippets are function-level fragments; add imports as the compiler asks. Blocks needing more than the obvious: Task 5's test uses `os` and `reflect`; Task 6's uses `errors`; Task 14's uses `bytes`, `os`, and the `couchcore` import.

---

## Decisions taken before this plan

1. **couch spawns `pair`, not `claude`.** The agent child is never spawned by Go today — zellij spawns it from a KDL layout (`zellij/layouts/main-2.kdl:45`), and `entrypoint.ValidRootMarkers` (`asset_root.go:73`) *defines* a valid pair install as having `main-2.kdl` and `main-3.kdl`. Going around that means implementing `launcher.Runtime` — 59 methods across 15 sub-interfaces (`runtime.go:229-244`) — whose semantics are zellij-shaped. So couch hosts `pair` whole. **Consequence: this milestone needs none of `agentargs.go`'s resume/session-id knowledge.**

2. **No pty in this milestone.** `#145`'s children inherit couch's stdin/stdout/stderr — a blocking handoff, the same shape as `ZellijOps.LaunchSession` ("a BLOCKING fork+wait child with the tty passed through", `runtime.go:31-34`). That is what "spawned sessions land in the current terminal" means. `#146` allocates ptys when it takes over routing. **A consequence worth stating: `couch start` blocks until the child exits, and only one child can be interactive at a time.** That is expected here and is exactly what `#146` removes.

3. **Identity is the canonical worktree root path; `cwd` may be a subdirectory.** `kbench/competition/arc-agi-3/` starts there and registers under `/Users/xianxu/workspace/kbench`. The collision hazards (shared index lock, shared branch, shared `git status`) are tree-scoped, so a subdirectory does not earn its own identity.

4. **Case folding is a separate concern from the stored path.** `Worktree` holds the canonical path in **original case**; `Worktree.Key()` returns the folded form used only for registry lookup. This matters: pair feeds `ResolveRepoScope` a raw path (`datadir.go:13` → `scope.go:20-21`, `filepath.Clean` + sha256), so a folded path would produce a *different* scope key than pair's for the same tree. Folding the key preserves the collision guard; keeping the path unfolded preserves agreement with pair and correct display.

5. **`launcher.ResolveRepoScope` does not canonicalise.** No symlink resolution, no case folding, no walk-up, no git. couch canonicalises first, then feeds it the unfolded path.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `NormalizePath` | `cmd/internal/couchcore/path.go` | new |
| `FoldKey` | `cmd/internal/couchcore/path.go` | new |
| `Worktree` | `cmd/internal/couchcore/worktree.go` | new |
| `ActorID` | `cmd/internal/couchcore/actorid.go` | new |
| `StartArgs` | `cmd/internal/couchcore/startargs.go` | new |
| `ActorRecord` | `cmd/internal/couchcore/registry.go` | new |
| `Registry` | `cmd/internal/couchcore/registry.go` | new |
| `TreeOccupiedError` | `cmd/internal/couchcore/registry.go` | new |
| `NamingTable` | `cmd/internal/couchcore/naming.go` | new |
| `Message` | `cmd/internal/couchcore/mailbox.go` | new |
| `Enqueue` | `cmd/internal/couchcore/mailbox.go` | new |
| `ConcurrencyPolicy` | `cmd/internal/couchcore/policy.go` | new |
| `Operation` / `Operations` | `cmd/internal/couchcore/ops.go` | new |

- **NormalizePath** — pure `string → string`: `filepath.Abs` + `filepath.Clean`. **No symlink resolution** — `filepath.EvalSymlinks` is `lstat`/`readlink` and returns a different answer depending on what exists on disk, which would make every downstream "pure" entity inherit a filesystem dependency. Symlink resolution lives on `PathOps` (below), following `reviewcmd.Runtime.PhysicalDir` (`reviewcmd/runtime.go:26-35`), the house precedent.

- **FoldKey** — pure `string → string`, lowercasing on case-insensitive platforms. Separate from `NormalizePath` because the folded form is a *lookup key*, never a path to hand to anything else.

- **Worktree** — `type Worktree string`, a canonical absolute worktree-root path in original case. The named type carries one invariant: *this has been canonicalised*. `Key() string` returns `FoldKey(string(w))`; `Repo() string` returns the basename, unfolded, matching `launcher.repoDisplayName`'s convention (`scope.go:33`, unexported — a convention to follow, not a symbol to import).
  - **Relationships:** 1 Worktree : N ActorRecords (normally ≤1; >1 only under `--same-tree`); 1 Worktree : ≤1 name + ≤1 description.
  - **DRY rationale:** `launcher.ScopedLaunchDataDir` (`datadir.go:13`) passes cwd straight to `ResolveRepoScope` with no walk-up — the same latent bug this type prevents.

- **ActorID** — `type ActorID string`, `couch-<8 hex>`. Identifies an *incarnation*, not an address; Erlang's pid to `Worktree`'s registered name. `#147` is where the distinction bites (a reply referencing a dead incarnation must be droppable); here it is a field with a generator.

- **StartArgs** — `{Worktree Worktree; Cwd string; Stack string; Layout string; ExtraArgs []string; SameTree bool}`. `Cwd` defaults to `Worktree`. `SameTree` is the escape hatch and is recorded on the resulting `ActorRecord`.

- **ActorRecord** — `{ID ActorID; Args StartArgs; State string; StartedAt time.Time; PID int; Identity string; SameTreeOverride bool}`. `Identity` is a kernel start token obtained through the `Runner` seam, **not** a bare PID: `sessionwatch/run.go:143` re-authorizes precisely because PID reuse can transfer ownership.

- **Registry** — `map[string][]ActorRecord` keyed on `Worktree.Key()`. The value is a **slice**, not a single record, so `--same-tree` produces two enumerable live actors rather than orphaning the incumbent's handle. `Register` returns `(Registry, error)`; `*TreeOccupiedError` carries the incumbents.
  - The one-agent-per-tree invariant **is** the collision guard — `Register` failing when the key is taken is Erlang's `register/2`, not a separate check.

- **TreeOccupiedError** — `{Tree Worktree; Incumbents []ActorRecord}`. Named `*Error` per Go convention (`Err*` is for sentinel values).

- **NamingTable** — `Key → {Name, Description}`. Attaches to the tree, not the ActorID, so naming survives revival. `Name` is operator-assigned **free prose** — deliberately *not* validated with `launcher.NormalizeTag` (`tag.go:10`), which rejects spaces and silently strips a `pair-` prefix; those rules exist for zellij session names, not human labels. `Lookup` is case-insensitive substring match over both fields and returns **every** candidate: fuzzy-in/exact-out means the caller disambiguates.

- **Message** — `{Kind string; Control bool; Body string}`. `Control` marks the priority class and makes the capacity rule ("drop the oldest non-control entry") implementable.

- **Enqueue** — pure `(queue []Message, incoming Message, capacity int) []Message`. Bounded, collapse-by-kind. Pure so it is testable without goroutines, which is where this kind of code rots.

- **ConcurrencyPolicy** — `Repo → Mode`. **Reads a recorded file** (`couch-policy.json` in the store dir), not a hardcoded constant — the issue asks for "read from a recorded source, not inferred," and a constant is the definition of inferred. Missing file ⇒ `InPlaceSerial` for all, the conservative default. `ariadne#200` replaces the file with fleet metadata later.

- **Operation / Operations** — `Operation{Name, Summary, Args []ArgSpec, Invoke func(*Couch, map[string]string) (any, error)}`. The CLI dispatches **through** `Operations()` — there is no argv switch — so the operator's surface and the advisor's cannot diverge (ARCH-PURPOSE).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Runner` / `Handle` | `cmd/internal/couchcore/runner.go` | new | `os/exec` |
| `FakeRunner` | `cmd/internal/couchcore/runner_fake.go` | new | stateful double |
| `PathOps` | `cmd/internal/couchcore/pathops.go` | new | `filepath.EvalSymlinks` |
| `GitRunner` | `cmd/internal/couchcore/git.go` | new | `git` binary |
| `Store` | `cmd/internal/couchcore/store.go` | new | disk (JSON) |
| `Clock` | `cmd/internal/couchcore/clock.go` | new | wall clock |
| `IDGen` | `cmd/internal/couchcore/actorid.go` | new | randomness |
| `Couch` | `cmd/internal/couchcore/couch.go` | new | composes all seams |

- **Runner / Handle** — `Start(dir string, argv, env []string) (Handle, error)`; `Handle` has `ID() string`, `PID() int`, `Identity() string`, `Alive() bool`, `Signal(os.Signal) error`, `Wait() int`.
  - **Verified new.** No `Start(dir, argv, env) (Handle, error)`, no `type Handle`, and no async child-lifecycle seam exists under `cmd/internal/`. `launcher.ProcOps` is sidecar-named (`SpawnSessionWatcher`, `SpawnTitlePoller`, `DevRebuild` — `runtime.go:82-92`); `ZellijOps.LaunchSession` is blocking and zellij-specific; `wrapcmd`/`clipcmd`/`opener` call `exec.Command` below their own seams. Searched: `grep -rn 'type Handle' cmd/`, `grep -rn 'Start(dir' cmd/`.
  - Contract note: this is a **larger** contract than the cited `ariadne/cmd/weave/internal/weavefs/runner.go`, whose `Run(dir, argv) error` is synchronous with no handle. Copy its *shape discipline* (one small interface, one production impl, a `var _ Runner = ExecRunner{}` assertion), not its surface.
  - `ExecRunner.Start` sets `cmd.Stdin/Stdout/Stderr` to the process's own, per Decision 2.
  - **A kill goes through the seam.** `osruntime.go:66-84` records why: a `delete-session` once inlined below the seam, so a test asserting "a foreign session is never deleted" passed while the hazard sat untouched.

- **FakeRunner** — the stateful double ARCH-MOCK requires. **State model, written down before Task 5 so it is a contract and not an inference:**
  - `Start` records `{argv, dir, env}`, marks the child **alive**, returns a handle with a deterministic id (`couch-fake-N`).
  - `Signal` appends to that child's `signals` log and **does not kill it** — real processes catch, ignore, or delay SIGINT, and pair's restart/quit loop depends on that. Encoding "signal ⇒ dead" would model a false behaviour that no live check contradicts.
  - `SetExited(id, code)` is the only thing that ends a child: it marks it dead and unblocks `Wait`.
  - `Wait` blocks until the child is exited; on an already-exited child it returns immediately.
  - `Ops []string` records an ordered log the tests assert against. (House style: `termcmd/run_test.go:868` carries `ops []string`. Note `launcher/createflow_test.go:20` does **not** — it records via typed fields — so `termcmd` is the model here.)

- **PathOps** — `Physical(path string) string`, wrapping `filepath.EvalSymlinks` with a fall-back to the input. Precedent: `reviewcmd.Runtime.PhysicalDir` (`reviewcmd/runtime.go:26-35`).

- **GitRunner** — `Run(dir string, args ...string) (string, error)`. Only `rev-parse --show-toplevel` is needed; `--git-common-dir` is **not** used in this milestone and is not declared.
  - **ARCH-DRY note, stated honestly:** this is a *third* copy of a seam shape that already exists — `reviewcmd.Runtime.Git(dir string, args ...string) (string, error)` (`reviewcmd/run.go:34-35`, impl `runtime.go:61`, fake `run_test.go:69`) is byte-identical in signature, and its `--show-toplevel` call at `run.go:222` goes *through* that seam rather than inline. Duplicating it is consistent with this repo's per-command-Runtime pattern, but it is duplication and should be named as such rather than justified as a gap.
  - **Live conformance (Task 15):** ARCH-MOCK names `git` explicitly. A gated test runs `rev-parse --show-toplevel` against a real temp repo *and a real linked worktree*, because the linked-worktree case is what the whole identity model rests on.

- **Store** — a JSON snapshot (`Save(Registry, NamingTable)` / `Load()`), **not append-only** — `osfs.FS.WriteAtomic` (`osfs/osfs.go:34-53`) is temp+rename and cannot append. If append semantics are ever wanted, the house idiom is `sessionwatch.appendSessionLedger` (`sessionwatch/run.go:184-198`).
  - **Location is unscoped:** `ResolveDataDir(home, xdg)/couch/registry.json`. The registry spans all worktrees — that is the point of `couch list` — so a per-tree `ScopedLaunchDataDir` would mean spawning in `/a` and listing from `/b` reads a different file. Portable by construction: the dir is passed in, so a test points at `t.TempDir()`.

- **Clock** — `Now() time.Time`. Injected; the domain never calls `time.Now`. Task 11 asserts `StartedAt`, so it is exercised.
- **IDGen** — `NewID() ActorID`. Injected; `couch-ah8d` must be reproducible.
- **Couch** — the composition root: `{Runner, Path, Git, Store, Clock, IDs, Policy}` plus in-memory `Registry`/`NamingTable`. Every operation is a method on it.

---

## Chunk 1: identity and seams

### Task 1: NormalizePath and FoldKey (pure)

**Files:** Create `cmd/internal/couchcore/path.go`, `cmd/internal/couchcore/path_test.go`

- [ ] **Step 1: Write the failing test**

```go
package couchcore

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNormalizePathCollapsesSpellings(t *testing.T) {
	base := "/Users/x/workspace/pair"
	want := NormalizePath(base)
	for _, s := range []string{
		base,
		base + "/",
		base + "/.",
		base + "/cmd/..",
		"/Users/x/workspace/./pair",
	} {
		if got := NormalizePath(s); got != want {
			t.Errorf("NormalizePath(%q) = %q, want %q", s, got, want)
		}
	}
}

func TestNormalizePathPreservesCase(t *testing.T) {
	// Case is preserved here on purpose: pair feeds ResolveRepoScope a raw
	// path, so a folded path would yield a different scope key for the same
	// tree. Folding belongs to the lookup key only.
	got := NormalizePath("/Users/x/KBench")
	if !strings.HasSuffix(got, "KBench") {
		t.Fatalf("NormalizePath folded case: %q", got)
	}
}

func TestNormalizePathIsAbsolute(t *testing.T) {
	if got := NormalizePath("relative/path"); !filepath.IsAbs(got) {
		t.Fatalf("non-absolute %q", got)
	}
}

func TestFoldKeyCollapsesCaseOnInsensitiveFS(t *testing.T) {
	a, b := FoldKey("/Users/x/pair"), FoldKey("/users/x/pair")
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		if a != b {
			t.Fatalf("FoldKey did not collapse case: %q vs %q", a, b)
		}
		return
	}
	if a == b {
		t.Fatal("FoldKey must not collapse case on a case-sensitive filesystem")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./cmd/internal/couchcore/ -run 'NormalizePath|FoldKey' -v`
Expected: FAIL — `undefined: NormalizePath`

- [ ] **Step 3: Implement**

```go
package couchcore

import (
	"path/filepath"
	"runtime"
	"strings"
)

// NormalizePath canonicalises a path so spellings of one location compare
// equal. Pure: Abs + Clean only. Symlink resolution is deliberately absent --
// filepath.EvalSymlinks is lstat/readlink and returns a different answer
// depending on what exists on disk, which would make every entity that takes
// a Worktree inherit a filesystem dependency. That lives on PathOps.
func NormalizePath(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	return filepath.Clean(p)
}

// FoldKey returns the registry lookup key for a path.
//
// On darwin the default filesystem is case-insensitive-preserving, so
// "/users/x" and "/Users/x" name one directory but differ as Go strings.
// Without folding the key, couch would accept both spellings as distinct
// trees and the one-agent-per-tree guard would fail open. Only the *key* is
// folded; the stored path keeps its case so it still matches pair's scope key
// and renders correctly.
func FoldKey(p string) string {
	if caseInsensitiveFS() {
		return strings.ToLower(p)
	}
	return p
}

func caseInsensitiveFS() bool { return runtime.GOOS == "darwin" || runtime.GOOS == "windows" }
```

- [ ] **Step 4: Run to verify it passes** — `go test ./cmd/internal/couchcore/ -v` → PASS
- [ ] **Step 5: Commit**

```bash
git add cmd/internal/couchcore/path.go cmd/internal/couchcore/path_test.go
git commit -m "#145: canonicalise paths; fold only the lookup key

Folding the stored path would change the scope key pair derives for the same
tree, so the fold is confined to registry lookup."
```

### Task 2: PathOps and GitRunner seams, with fakes

**Files:** Create `pathops.go`, `git.go`, `git_fake.go`, `seams_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestFakeGitRecordsAndTrims(t *testing.T) {
	g := NewFakeGit(map[string]string{"rev-parse --show-toplevel": "/repo\n"})
	out, err := g.Run("/repo/sub", "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out != "/repo" {
		t.Fatalf("out = %q; trailing newline must be trimmed", out)
	}
	if len(g.Ops) != 1 || g.Ops[0] != "/repo/sub: rev-parse --show-toplevel" {
		t.Fatalf("Ops = %v", g.Ops)
	}
}

func TestFakeGitUnknownCommandErrors(t *testing.T) {
	if _, err := NewFakeGit(nil).Run("/repo", "status"); err == nil {
		t.Fatal("expected error for uncanned command")
	}
}

func TestFakePathOpsMapsSymlinks(t *testing.T) {
	p := NewFakePathOps(map[string]string{"/link": "/real"})
	if got := p.Physical("/link"); got != "/real" {
		t.Fatalf("Physical = %q", got)
	}
	if got := p.Physical("/plain"); got != "/plain" {
		t.Fatalf("unmapped path must pass through, got %q", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails** — `undefined: NewFakeGit`
- [ ] **Step 3: Implement** *(spec, not listing, for the obvious parts)*

`git.go`: `type GitRunner interface { Run(dir string, args ...string) (string, error) }`, `ExecGit` running `exec.Command("git", args...)` with `cmd.Dir = dir`, returning `strings.TrimSpace(string(out))`, plus `var _ GitRunner = ExecGit{}`.

`pathops.go`: `type PathOps interface { Physical(string) string }`; `OSPathOps` calls `filepath.EvalSymlinks` and returns the input on error; `FakePathOps` is a map with pass-through.

`git_fake.go`: `FakeGit{replies map[string]string; Ops []string}` — canned lookup keyed by `strings.Join(args, " ")`, appending `dir+": "+key` to `Ops`, erroring on a miss.

Header comment on `git.go` must state the ARCH-DRY position: this duplicates `reviewcmd.Runtime.Git` (`reviewcmd/run.go:34-35`) in signature, consistent with the per-command-Runtime pattern; it is duplication, not a gap.

- [ ] **Step 4: Run to verify it passes** — PASS
- [ ] **Step 5: Commit** — `git commit -m "#145: add PathOps and GitRunner seams with fakes"`

### Task 3: Worktree.Resolve

**Files:** Create `worktree.go`, `worktree_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestResolveWalksUpToWorktreeRoot(t *testing.T) {
	g := NewFakeGit(map[string]string{"rev-parse --show-toplevel": "/Users/x/workspace/kbench\n"})
	p := NewFakePathOps(nil)
	// The kbench/competition/arc-agi-3 case: a subdirectory resolves to the
	// tree that contains it.
	wt, err := Resolve("/Users/x/workspace/kbench/competition/arc-agi-3", g, p)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if wt != Worktree("/Users/x/workspace/kbench") {
		t.Fatalf("Resolve = %q", wt)
	}
}

func TestResolveAppliesSymlinkSeam(t *testing.T) {
	g := NewFakeGit(map[string]string{"rev-parse --show-toplevel": "/real/repo\n"})
	p := NewFakePathOps(map[string]string{"/link/repo": "/real/repo"})
	wt, err := Resolve("/link/repo", g, p)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if wt != Worktree("/real/repo") {
		t.Fatalf("Resolve = %q", wt)
	}
}

func TestResolveRejectsNonRepo(t *testing.T) {
	if _, err := Resolve("/tmp", NewFakeGit(nil), NewFakePathOps(nil)); err == nil {
		t.Fatal("expected error outside a git worktree")
	}
}

func TestWorktreeKeyFoldsButRepoDoesNot(t *testing.T) {
	w := Worktree("/Users/x/KBench")
	if w.Repo() != "KBench" {
		t.Fatalf("Repo = %q; display must keep case", w.Repo())
	}
	if FoldKey(string(w)) != w.Key() {
		t.Fatalf("Key must be the folded path")
	}
}
```

- [ ] **Step 2: Run to verify it fails** — `undefined: Resolve`
- [ ] **Step 3: Implement**

```go
package couchcore

import (
	"fmt"
	"path/filepath"
)

// Worktree is a canonical absolute worktree-root path in original case. The
// named type carries one invariant: this path has been canonicalised.
type Worktree string

// Key is the registry lookup key -- case-folded where the filesystem is.
func (w Worktree) Key() string { return FoldKey(string(w)) }

// Repo is the display name (basename), unfolded.
func (w Worktree) Repo() string { return filepath.Base(string(w)) }

// Resolve canonicalises path, resolves symlinks through the seam, and walks up
// to the worktree root. A subdirectory resolves to its containing tree, so
// identity stays tree-scoped: the collision hazards (one index lock, one
// branch, one git status) are properties of the tree.
func Resolve(path string, git GitRunner, p PathOps) (Worktree, error) {
	norm := p.Physical(NormalizePath(path))
	top, err := git.Run(norm, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("resolve worktree for %s: %w", norm, err)
	}
	if top == "" {
		return "", fmt.Errorf("resolve worktree for %s: empty toplevel", norm)
	}
	return Worktree(p.Physical(NormalizePath(top))), nil
}
```

- [ ] **Step 4: Run to verify it passes** — PASS
- [ ] **Step 5: Commit** — `git commit -m "#145: resolve a path to its worktree root through the path seam"`

### Task 4: Clock and IDGen

**Files:** Create `clock.go`, `actorid.go`, `actorid_test.go`

Tests: `NewFixedIDGen("ah8d","b2c1")` yields `couch-ah8d` then `couch-b2c1`; `NewRandomIDGen().NewID()` has prefix `couch-` and length `len("couch-")+8`; `FixedClock{T}.Now()` returns `T`.

Implementation as in the entity table. Import `crypto/rand`, `encoding/hex`, `time`.

- [ ] Steps 1–5 as above. Commit: `"#145: inject clock and id generation"`

### Task 5: Runner seam and stateful fake

**Files:** Create `runner.go`, `runner_fake.go`, `runner_test.go`

The fake's state model is the contract written in the Integration Points section above. Implement to it exactly.

- [ ] **Step 1: Write the failing test**

```go
func TestFakeRunnerSignalDoesNotKill(t *testing.T) {
	// Real processes catch, ignore, or delay SIGINT -- pair's restart/quit
	// loop depends on it. A fake that dies on any signal models a falsehood.
	r := NewFakeRunner()
	h, err := r.Start("/repo", []string{"pair", "--layout2"}, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := h.Signal(os.Interrupt); err != nil {
		t.Fatalf("Signal: %v", err)
	}
	if !h.Alive() {
		t.Fatal("child must survive a signal it was not told to die on")
	}
	if got := r.Signals(h.ID()); len(got) != 1 {
		t.Fatalf("signals = %v", got)
	}
}

func TestFakeRunnerSetExitedEndsTheChild(t *testing.T) {
	r := NewFakeRunner()
	h, _ := r.Start("/repo", []string{"pair"}, nil)
	r.SetExited(h.ID(), 3)
	if h.Alive() {
		t.Fatal("child must be dead after SetExited")
	}
	if code := h.Wait(); code != 3 {
		t.Fatalf("Wait = %d, want 3", code)
	}
}

func TestFakeRunnerRecordsOps(t *testing.T) {
	r := NewFakeRunner()
	h, _ := r.Start("/repo", []string{"pair", "--layout2"}, nil)
	_ = h.Signal(os.Interrupt)
	want := []string{"start /repo: pair --layout2", "signal couch-fake-1: interrupt"}
	if !reflect.DeepEqual(r.Ops, want) {
		t.Fatalf("Ops = %v, want %v", r.Ops, want)
	}
}
```

Imports: `os`, `reflect`, `testing`.

- [ ] **Step 2: Run to verify it fails** — `undefined: NewFakeRunner`
- [ ] **Step 3: Implement** — interface + `ExecRunner` (stdio inherited per Decision 2; `Identity()` returns `procutil.Identity(strconv.Itoa(pid))` so the unseamed syscall stays below the seam) + `FakeRunner` with `Signals(id)` accessor and `SetExited`. Add `var _ Runner = ExecRunner{}`.
- [ ] **Step 4: `go test ./cmd/internal/couchcore/ -race -v`** → PASS
- [ ] **Step 5: Commit** — `"#145: add process Runner seam and stateful fake (ARCH-MOCK)"`

---

## Chunk 2: registry, naming, mailbox

### Task 6: Registry and the collision guard

**Files:** Create `registry.go`, `startargs.go`, `registry_test.go`

Tests must cover: refusing a second actor and naming the incumbent; **allowing distinct trees of the same repo** (primary + linked worktree); `Unregister` freeing the tree; `Get` returning records; and — the one the whole milestone rests on — **case-differing spellings colliding**:

```go
func TestRegisterCollidesOnCaseDifferingSpellings(t *testing.T) {
	if !caseInsensitiveFS() {
		t.Skip("case folding only applies on a case-insensitive filesystem")
	}
	reg := NewRegistry()
	reg, err := reg.Register(ActorRecord{ID: "couch-a", Args: StartArgs{Worktree: "/Users/x/repo"}})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	_, err = reg.Register(ActorRecord{ID: "couch-b", Args: StartArgs{Worktree: "/users/x/repo"}})
	var occ *TreeOccupiedError
	if !errors.As(err, &occ) {
		t.Fatalf("err = %v; a differently-cased spelling of one tree must collide", err)
	}
}

func TestSameTreeOverrideKeepsIncumbentEnumerable(t *testing.T) {
	reg := NewRegistry()
	reg, _ = reg.Register(ActorRecord{ID: "couch-a", Args: StartArgs{Worktree: "/repo"}})
	reg, err := reg.Register(ActorRecord{ID: "couch-b", Args: StartArgs{Worktree: "/repo", SameTree: true}, SameTreeOverride: true})
	if err != nil {
		t.Fatalf("override refused: %v", err)
	}
	if got := reg.Get("/repo"); len(got) != 2 {
		t.Fatalf("got %d actors; the incumbent must survive, not be orphaned", len(got))
	}
}
```

- [ ] Steps 1–5. Commit: `"#145: registry keyed on the folded worktree; registration is the guard"`

### Task 7: NamingTable

**Files:** Create `naming.go`, `naming_test.go`

Free-prose names (no `NormalizeTag`). `Lookup` returns all candidates. Test that an ambiguous label returns both. **The survives-revival test moves to Task 12**, after spawn exists, because it cannot be meaningful without an actor to kill.

- [ ] Steps 1–5. Commit: `"#145: naming table keyed on the tree, free-prose labels"`

### Task 8: Message and Enqueue

**Files:** Create `mailbox.go`, `mailbox_test.go`

```go
func TestEnqueueCollapsesRepeatedKindKeepingLatest(t *testing.T) {
	q := []Message{{Kind: "status", Body: "old"}, {Kind: "note", Body: "a"}}
	got := Enqueue(q, Message{Kind: "status", Body: "new"}, 8)
	if len(got) != 2 {
		t.Fatalf("len = %d, want collapse", len(got))
	}
	// Asserting identity and position, not just length: an Enqueue that drops
	// the incoming message entirely would pass a length-only check.
	if got[len(got)-1].Kind != "status" || got[len(got)-1].Body != "new" {
		t.Fatalf("tail = %+v, want the newest status at the tail", got[len(got)-1])
	}
}

func TestEnqueueDropsOldestNonControlAtCapacity(t *testing.T) {
	q := []Message{{Kind: "stop", Control: true}, {Kind: "n1"}}
	got := Enqueue(q, Message{Kind: "n2"}, 2)
	if len(got) != 2 || got[0].Kind != "stop" {
		t.Fatalf("got = %+v; control messages are not the ones dropped", got)
	}
}
```

Implementation: full body required (collapse when an entry of the same `Kind` exists — replace and move to tail; then while `len > capacity`, drop the first non-control entry; if all are control, drop nothing and the caller logs loudly).

- [ ] Steps 1–5. Commit: `"#145: pure mailbox policy (bounded, collapse-by-kind)"`

### Task 9: Actor goroutine consuming Enqueue

**Files:** Create `actor.go`, `actor_test.go`

The actor must **use** `Enqueue` — a bare buffered channel is FIFO with no collapse and no capacity policy, which would leave Task 8 orphaned (ARCH-PURPOSE). Shape: an unbounded-in / bounded-queue actor where `Send` applies `Enqueue` under the mutex and signals a condition, and the loop drains control-first.

```go
func TestActorDrainsControlBeforeNormal(t *testing.T) {
	a := NewActor(ActorRecord{ID: "couch-a"}, 8)
	var order []string
	done := make(chan struct{})
	a.OnMessage = func(m Message) {
		order = append(order, m.Kind)
		if len(order) == 4 {
			close(done)
		}
	}
	// Several normal messages before one control message: a naive select over
	// two channels would pass half the time with one of each.
	for i := 0; i < 3; i++ {
		a.Send(Message{Kind: "note"})
	}
	a.Send(Message{Kind: "stop", Control: true})
	go a.Loop()
	<-done
	if order[0] != "stop" {
		t.Fatalf("order = %v, want control first", order)
	}
	a.Close()
}
```

Document in the file: Go shares memory, so **queries are direct calls behind a mutex**, not messages. Message passing is for ordering and decoupling, not fidelity to Erlang.

- [ ] Steps 1–5, with `-race`. Commit: `"#145: actor loop draining control-first via the pure mailbox policy"`

### Task 10: Store and ConcurrencyPolicy

**Files:** Create `store.go`, `policy.go`, `store_test.go`, `policy_test.go`

Store: JSON snapshot at `<dir>/registry.json` via `osfs.FS.WriteAtomic`; `Load` on a missing file returns empty state and **no error** (first run). Round-trip test asserts both registry and naming survive.

Policy: reads `<dir>/couch-policy.json` (`{"repos":{"pair":"in-place-serial"}}`); missing file ⇒ `InPlaceSerial` for all. Test both the recorded and default paths.

- [ ] Steps 1–5. Commit: `"#145: snapshot store and recorded per-repo concurrency policy"`

---

## Chunk 3: composition, surface, verification

### Task 11: Couch composition root and Spawn

**Files:** Create `couch.go`, `spawn.go`, `spawn_test.go`

`newTestEnv(t)` is the harness every later test uses — **specify it here**: returns `{Couch *Couch, Runner *FakeRunner, Git *FakeGit, Clock FixedClock, Dir string}` wired with `NewFixedIDGen("ah8d","b2c1")`, `FixedClock{T: time.Date(2026,8,21,12,0,0,0,time.UTC)}`, a `t.TempDir()` store, and a `FakeGit` whose `--show-toplevel` returns the requested tree.

Tests: spawn registers and starts `pair` with the right argv and cwd; `StartedAt` equals the fixed clock (cashing the Clock seam); a refused spawn **starts no process**; `--same-tree` succeeds and records the override; a failed `Start` leaves the tree unregistered.

- [ ] Steps 1–5, `-race`. Commit: `"#145: composition root and the spawn path with the tree guard"`

### Task 12: Naming survives revival (end-to-end)

**Files:** Modify `naming_test.go`

The test B5 flagged, done properly: spawn on a tree, name it, exit the child, unregister, spawn again, assert `Lookup` still resolves. This is the only test that closes issue Done-when bullet 4, and it must involve an actual actor lifecycle.

- [ ] Steps 1–5. Commit: `"#145: prove naming survives an actor being replaced"`

### Task 13: Description sidecar

**Files:** Create `description.go`, `description_test.go`

Closes the agent-supplied half of Done-when bullet 4. The live query is `#147`; here couch **reads the cached answer**: the child writes `<store>/desc/<key>` and `Couch.Describe(tree)` reads it, falling back to the last stored value.

Document why this is not the published-status artifact returning: it is a one-line **label**, and a stale label still finds the right tree. Labels tolerate staleness; state does not.

- [ ] Steps 1–5. Commit: `"#145: read the agent-supplied description from its cached sidecar"`

### Task 14: Operation surface, structural audit, and CLI

**Files:** Create `ops.go`, `cmd/internal/couchcmd/run.go`, `cmd/couch/main.go`, `ops_test.go`, `cmd/internal/couchcmd/run_test.go`; modify `Makefile.local`

Operations: `start`, `list`, `show`, `stop`, `name`, `describe` — all six built, since `show` is what closes "every state is queryable" and `stop` is what makes a spawned actor terminable.

**The audit must be structural, not a restatement.** A literal list in the test is a spot-check and would not catch a CLI operation that skipped declaration:

```go
// The CLI has no argv switch: it dispatches through Operations(). This test
// asserts that structurally, so an operation added to the CLI without being
// declared cannot exist -- which a hand-written list of expected names would
// not catch.
func TestCLIDispatchesOnlyThroughOperations(t *testing.T) {
	src, err := os.ReadFile("run.go")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if bytes.Contains(src, []byte("switch args[0]")) {
		t.Fatal("couchcmd must dispatch through couchcore.Operations(), not an argv switch")
	}
	names := map[string]bool{}
	for _, op := range couchcore.Operations() {
		if op.Summary == "" {
			t.Errorf("%s: empty summary -- the advisor needs it to choose", op.Name)
		}
		names[op.Name] = true
	}
	for _, want := range []string{"start", "list", "show", "stop", "name", "describe"} {
		if !names[want] {
			t.Errorf("operation %q not declared", want)
		}
	}
}
```

Arity is asserted per-operation (`len(op.Args)` against the operation's own declared arity), **not** `op.Args == nil` — a zero-argument operation legitimately has a nil slice.

CLI signature follows the house pattern — `RunWithRuntime(args []string, stdin io.Reader, stdout, stderr io.Writer, rt Runtime) int` with `Run` as the thin `OSRuntime{}` wrapper (`termcmd/run.go:46-50`). Tests call `RunWithRuntime` with a temp store so they are hermetic and never read the developer's real `$HOME/.local/share/pair`.

`start` renders worktree-or-switch from `*TreeOccupiedError` — Done-when bullet 3 — and a test asserts the rendering names the incumbent and offers all three options.

`Makefile.local`: add a `couch` target beside `pair` (`:279-280`). **Note the side effect:** `PAIR_GO_SRCS` is `$(shell find cmd -name '*.go' …)` (`:272`), so new couch sources become prerequisites of `bin/pair`. Add a check that `make pair && ./bin/pair --help` still works — Done-when bullet 7 ("`pair-go` standalone is unaffected") is otherwise unverified.

- [ ] Steps 1–5. Commit: `"#145: declare the operation surface, dispatch the CLI through it"`

### Task 15: Live conformance for both fakes

**Files:** Create `runner_live_test.go`, `git_live_test.go`

House pattern exactly: **env-var skip, no build tag** (`harness_tty_live_test.go:543` uses `PAIR_LIVE_HARNESS`). A `//go:build live` tag would stop the file compiling under `go test ./cmd/...`, so it would rot invisibly — the exact failure the conformance check exists to prevent. Gate name: `PAIR_LIVE_COUCH`.

```go
package couchcore

import (
	"os"
	"testing"
)

func TestExecRunnerAndFakeAgreeOnLifecycle(t *testing.T) {
	if os.Getenv("PAIR_LIVE_COUCH") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1 to run against real processes")
	}
	// Real: a child that exits 3.
	real, err := ExecRunner{}.Start(t.TempDir(), []string{"sh", "-c", "exit 3"}, nil)
	if err != nil {
		t.Fatalf("ExecRunner.Start: %v", err)
	}
	if code := real.Wait(); code != 3 {
		t.Errorf("ExecRunner Wait = %d, want 3", code)
	}
	// Fake: driven to the same observable end state.
	f := NewFakeRunner()
	fh, _ := f.Start(t.TempDir(), []string{"sh", "-c", "exit 3"}, nil)
	f.SetExited(fh.ID(), 3)
	if code := fh.Wait(); code != 3 {
		t.Errorf("FakeRunner Wait = %d, want 3", code)
	}
	// Signal handling is where the fake is most likely to drift, so check it
	// against a real process that ignores SIGINT.
	ignorer, err := ExecRunner{}.Start(t.TempDir(), []string{"sh", "-c", "trap '' INT; sleep 0.3"}, nil)
	if err != nil {
		t.Fatalf("start ignorer: %v", err)
	}
	_ = ignorer.Signal(os.Interrupt)
	if !ignorer.Alive() {
		t.Error("a process ignoring SIGINT must still be alive -- the fake models this")
	}
	_ = ignorer.Wait()
}
```

`git_live_test.go` (same gate): create a real temp repo **and a linked worktree**, then assert `Resolve` from a subdirectory of each returns the correct root and that the two are distinct `Worktree`s. ARCH-MOCK names `git` explicitly, and the linked-worktree case is what the whole identity model rests on.

- [ ] Run both ways; commit: `"#145: live conformance for the Runner and Git fakes"`

### Task 16: Smoke — host one real pair child

**Files:** none (manual; record the outcome in the issue `## Log`)

**What this settles and what it does not.** With stdio inherited and no pty (Decision 2), this answers *"does `pair` start correctly as a couch child, with zellij and nvim underneath, when couch hands it the real terminal."* It does **not** exercise attach/detach or multi-child routing — those need the pty that `#146` builds. Record which question the result answers so a negative is not over-read.

- [ ] `go build ./cmd/couch && ./couch start ../pair`
- [ ] From a second shell: `./couch list` and `./couch show ../pair`
- [ ] `./couch start ../pair` again → refused, incumbent named, three options offered
- [ ] `./couch start ../kbench/competition/arc-agi-3` → registers under `.../kbench`
- [ ] Write the layering-fork answer into `## Log` either way

---

## Verification before close

- `go test ./cmd/... -race` green; `PAIR_LIVE_COUCH=1 go test ./cmd/internal/couchcore/` green.
- `make pair && ./bin/pair --help` still works (Done-when bullet 7).
- Every Done-when bullet mapped to a task: 1→T11/T14, 2→T6/T14, 3→T14, 4→T12/T13, 5→T10, 6→T14, 7→T14 check.
- The case-spelling collision test (T6) passes on darwin.
