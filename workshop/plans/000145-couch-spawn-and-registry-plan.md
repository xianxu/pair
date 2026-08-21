# couch: spawn and registry — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** couch can register, list, inspect and spawn agent actors keyed on a working tree, refusing a second agent on an occupied tree with a loud override, exposing every operation through one declared surface.

**Architecture:** A new binary `cmd/couch` over a new domain package `cmd/internal/couchcore`. Actors are goroutines holding a bounded mailbox; the agent harness is a **child `pair` process** inheriting couch's stdio (host-pair-whole; the pty arrives with `#146`). Identity is a canonicalised worktree path. All IO sits behind injected seams so the domain is unit-testable without processes, disk, wall-clock, or randomness (ARCH-PURE).

**Tech Stack:** Go 1.26, `os/exec`, `encoding/json`, stdlib only. Reuses `cmd/internal/launcher` (`ResolveRepoScope`, `ResolveDataDir`), `cmd/internal/osfs`, `cmd/internal/procutil`.

**Issue:** pair#145. **Project:** `workshop/projects/couch.md`.

## How to read this plan

**This plan specifies contracts and test intent, not finished code.** Two rounds of fresh-eyes review established why: hand-written Go in a markdown document cannot be validated without executing it, and executing it means implementing it. The second review found a test that deadlocked and a test that passed with the seam it named deleted — both invisible to inspection, both caught in minutes by running them.

So each task states **the contract**, **what each test must catch** (the bug that makes it fail), and **the deletion check** — the thing you remove to prove the test is load-bearing. Write the code at the keyboard, red first, and run it. Where a signature is genuinely subtle it is given exactly; otherwise the contract is the spec.

**Standing review move, applied per task:** for every test that asserts a *seam is used*, delete the seam call and confirm the test goes red. A test that survives that deletion is testing nothing.

---

## Decisions

1. **couch spawns `pair`, not `claude`.** The agent child is never spawned by Go today — zellij spawns it from a KDL layout (`zellij/layouts/main-2.kdl:45`), and `entrypoint.ValidRootMarkers` (`asset_root.go:73`) *defines* a valid pair install as having `main-2.kdl` and `main-3.kdl`. Going around that means implementing `launcher.Runtime` — 59 methods across 14 embedded sub-interfaces (`runtime.go:229-244`) — whose semantics are zellij-shaped.

2. **No pty here; children inherit couch's stdio.** Same shape as `ZellijOps.LaunchSession` ("a BLOCKING fork+wait child with the tty passed through", `runtime.go:31-34`). **Consequence: `couch start` blocks until the child exits.** So every read operation (`list`, `show`) runs in a *second process* against the persisted snapshot — see Decision 6.

3. **Identity is the canonical worktree root path; `cwd` may be a subdirectory.** `kbench/competition/arc-agi-3/` starts there and registers under `/Users/xianxu/workspace/kbench`, because the collision hazards (one index lock, one branch, one `git status`) are tree-scoped.

4. **Case folding applies to the lookup key only.** `Worktree` holds the canonical path in original case; `Key()` folds. pair feeds `ResolveRepoScope` a raw path (`datadir.go:13` → `scope.go:20-25`, Clean + sha256), so a folded path would derive a *different* scope key than pair's for the same tree.

5. **The mailbox is a mutex-guarded queue, not two channels.** The issue's Spec suggests two channels with a priority select; a queue with `Enqueue` applied under the mutex is used instead, because the bounded/collapse policy has to be applied at insertion and a bare buffered channel cannot do that. **Recorded as a deviation from the Spec, not an oversight.**

6. **Liveness is read out of process.** Because `start` blocks, `list` cannot hold a `Handle`. `ActorRecord` persists `{PID, Identity}` where `Identity` is `procutil.Identity`'s kernel start token; a second process recomputes it and compares. Equal ⇒ alive; different or absent ⇒ the actor is gone and the entry is stale. This is the only defence against PID reuse — `sessionwatch/run.go:143` re-authorizes for exactly this reason. **`Store.Save` fires before `Handle.Wait`**, or the second shell sees an empty registry.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `NormalizePath` | `couchcore/path.go` | new |
| `FoldKey` | `couchcore/path.go` | new |
| `Worktree` | `couchcore/worktree.go` | new |
| `ActorID` | `couchcore/actorid.go` | new |
| `StartArgs` | `couchcore/startargs.go` | new |
| `ActorRecord` | `couchcore/registry.go` | new |
| `Registry` | `couchcore/registry.go` | new |
| `TreeOccupiedError` | `couchcore/registry.go` | new |
| `NamingTable` | `couchcore/naming.go` | new |
| `Message` / `Enqueue` | `couchcore/mailbox.go` | new |
| `Actor` | `couchcore/actor.go` | new |
| `PolicyTable` / `Mode` | `couchcore/policy.go` | new |
| `Operation` / `ArgSpec` / `Operations` | `couchcore/ops.go` | new |

- **NormalizePath** — `string → string`: `filepath.Abs` + `filepath.Clean`. **Documented impurity:** `filepath.Abs` reads `os.Getwd()` for a relative input. That is accepted rather than hidden — the alternative is threading `cwd` through every caller for a case that only arises at the CLI edge. Symlink resolution is *not* here; it needs `lstat` and lives on `PathOps`.

- **FoldKey** — lookup-key folding, lowercase on darwin/windows. Keys on `runtime.GOOS`, though case-sensitivity is really a *volume* property: a case-sensitive APFS dev volume would conflate two genuinely distinct trees. It fails **closed** (refuses a spawn rather than allowing a collision), which is the acceptable direction.

- **Worktree** — `type Worktree string`, canonical path in original case. `Key() string` folds; `Repo() string` is the unfolded basename (convention from `launcher.repoDisplayName`, `scope.go:33`, unexported).

- **ActorID** — `couch-<8 hex>`. An *incarnation*, not an address — Erlang's pid to `Worktree`'s registered name. Bites in `#147`.

- **StartArgs** — `{Worktree; Cwd; Stack; Issue string; ExtraArgs []string; SameTree bool}`. `Cwd` defaults to `Worktree`. `Stack` selects the agent (`claude` default) and is passed to `pair`. `Issue` is optional metadata per the issue's Spec. **`Layout` is dropped** — nothing in this milestone consumes it.

- **ActorRecord** — `{ID; Args StartArgs; StartedAt time.Time; PID int; Identity string}`. **`SameTreeOverride` is dropped**: it duplicated `Args.SameTree`, and a test setting both could not tell you which one `Register` read (ARCH-DRY). `State` is dropped — liveness is computed from `Identity`, not stored.

- **Registry** — a **struct wrapping** `map[string][]ActorRecord` keyed on `Worktree.Key()`, with `Register`/`Unregister` returning a new `Registry` after copying the map. Value semantics must be real: a bare map is a reference type, so a "functional" signature over one is a lie that lets a failed `Register` mutate the caller's state anyway. Value is a **slice** so `--same-tree` produces two enumerable actors rather than orphaning the incumbent's handle.
  - `Get(w Worktree) []ActorRecord` — takes a `Worktree`, folds internally. Never takes a raw string, or the folding is bypassable.

- **TreeOccupiedError** — `{Tree Worktree; Incumbents []ActorRecord; Mode Mode}`. Carries the policy mode so the caller renders the *right* offer: under `WorktreeParallel` a new worktree is the natural suggestion; under `HeavyLocalState` it is not, and switching or `--same-tree` are.

- **NamingTable** — `Key → {Name, Description}`. Free prose; deliberately **not** validated with `launcher.NormalizeTag` (`tag.go:10`), which rejects spaces and silently strips a `pair-` prefix — rules for zellij session names, not human labels. `Lookup` is case-insensitive substring over both fields, returning **every** candidate.

- **Message / Enqueue** — `Message{Kind string; Control bool; Body string}`. `Enqueue(queue []Message, incoming Message, capacity int) (out []Message, dropped Message, ok bool)`. The `ok` return exists because the issue's Spec requires a full mailbox to be a **loud bug signal**; a signature that cannot fail makes that impossible.

- **Actor** — `{ID; queue []Message; mu sync.Mutex; cond *sync.Cond; OnMessage func(Message)}` with `Send`, `Loop`, `Close`. Declared here because the previous draft left it in prose only.

- **PolicyTable / Mode** — `Mode` is `InPlaceSerial | WorktreeParallel | HeavyLocalState` (three, per the issue's Spec). `PolicyTable` is a **pure** `map[string]Mode` with `Mode(repo string) Mode` defaulting to `InPlaceSerial`. **Loading is the `Store`'s job**, not the table's — a pure entity that reads a file is the defect this plan already corrected once elsewhere.

- **Operation / ArgSpec / Operations** — `ArgSpec{Name, Summary string, Required bool}`. `Operation{Name, Summary string, Args []ArgSpec, Invoke func(*Couch, map[string]string) (any, error)}`. The descriptor is pure data; `Invoke` is a function into the IO shell, which is the boundary, stated rather than glossed.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Runner` / `Handle` | `couchcore/runner.go` | new | `os/exec` |
| `FakeRunner` | `couchcore/runner_fake.go` | new | stateful double |
| `PathOps` | `couchcore/pathops.go` | new | `filepath.EvalSymlinks` |
| `GitRunner` / `FakeGit` | `couchcore/git.go`, `git_fake.go` | new | `git` |
| `Store` | `couchcore/store.go` | new | disk |
| `Clock`, `IDGen` | `couchcore/clock.go`, `actorid.go` | new | time, randomness |
| `Couch` | `couchcore/couch.go` | new | composition root |

- **Runner / Handle** — `Start(dir string, argv, env []string) (Handle, error)`; `Handle` has `ID() string`, `PID() int`, `Identity() string`, `Alive() bool`, `Signal(os.Signal) error`, `Wait() int`.
  - **Verified genuinely new.** `grep -rn 'type Handle' cmd/` and `grep -rn 'Start(dir' cmd/` both return nothing. `launcher.ProcOps` is sidecar-named (`runtime.go:82-92`); `ZellijOps.LaunchSession` is blocking and zellij-specific. Note `wrapcmd` spawns its child *inline and unseamed* (`wrap.go:2330-2332`) — it is not a counter-example of a seam, it is an absence of one.
  - Larger contract than `ariadne/cmd/weave/internal/weavefs/runner.go`, whose `Run(dir, argv) error` is synchronous. Copy its shape discipline (one small interface, one prod impl, `var _ Runner = ExecRunner{}`), not its surface.
  - `ExecRunner.Start` inherits the process's stdio (Decision 2). A kill goes **through** the seam — `osruntime.go:66-84` records what happens when it doesn't.

- **FakeRunner — state model (contract, fixed before Task 5):**
  - `Start` records `{argv, dir, env}`, marks the child alive, returns a handle with id `couch-fake-N`.
  - `Signal` appends to the child's signal log and **does not kill it**. Real processes catch, ignore or delay SIGINT and pair's restart loop depends on that; "signal ⇒ dead" would model a falsehood.
  - `SetExited(id, code)` is the only thing that ends a child; it unblocks `Wait`.
  - `Wait` blocks until exited; returns immediately if already exited.
  - **The handle writes back to the Runner's `Ops` log.** Stated explicitly: implementing "the child holds its own signal log" literally makes the ops-ordering test fail.

- **PathOps** — `Physical(path string) (string, error)`. Precedent is `reviewcmd.Runtime.PhysicalDir` (`reviewcmd/runtime.go:45-57`, interface at `run.go:30-32`) — **which returns `""` on failure, deliberately, so the caller can detect a nonexistent directory.** couch returns an error rather than falling back to the input: silently accepting an unresolvable path would let it become its own identity.

- **GitRunner / FakeGit** — `Run(dir string, args ...string) (string, error)`. Only `rev-parse --show-toplevel` is needed.
  - **ARCH-DRY, stated plainly:** this duplicates `reviewcmd.Runtime.Git` (`run.go:34-35`, impl `runtime.go:61`, fake `run_test.go:69`) byte-for-byte in signature, and that seam's `--show-toplevel` call at `run.go:222` goes through it. A shared `cmd/internal/gitrun` would be the DRY move; it is **not** taken here because every command package in this repo owns its own Runtime, and extracting one seam out of that pattern for a single consumer is a larger change than this issue's purpose. Revisit at the third consumer.
  - **`FakeGit` must key its canned replies on `dir` as well as argv.** Ignoring `dir` is what made a previous draft's symlink test vacuous, and it makes "was git run in the right directory" — the only bug `Resolve` can have — untestable.

- **Store** — JSON snapshot at `ResolveDataDir(home, xdg)/couch/registry.json`, **unscoped**: the registry spans all worktrees, so a per-tree `ScopedLaunchDataDir` would mean spawning in `/a` and listing from `/b` reads different files. `Save` via `osfs.FS.WriteAtomic` (`osfs/osfs.go:35-54`, temp+rename — it cannot append; the append idiom, if ever needed, is `sessionwatch.appendSessionLedger`, `run.go:184-198`). `Load` on a missing file returns empty state and **no** error. The store also loads `PolicyTable` from `couch/policy.json`.

- **Couch** — `{Runner, Path, Git, Store, Clock, IDs}` plus in-memory `Registry`, `NamingTable`, `PolicyTable`. Every operation is a method.

---

## Tasks

Each task: write the tests red, implement, run, apply the deletion check, commit.

### Task 1 — `NormalizePath`, `FoldKey`

**Contract:** as declared above.
**Tests must catch:** a spelling that fails to collapse (`trailing/`, `/.`, `/x/..`, `//`); case being folded *into the stored path* (it must not be); a relative input staying relative; `FoldKey` collapsing case on darwin and **not** collapsing on a case-sensitive platform (assert both directions, not one with a skip).
**Deletion check:** remove `filepath.Clean` — the `/x/..` case must go red.

### Task 2 — `PathOps`, `GitRunner`, and their fakes

**Contract:** `Physical` returns an error, not a fallback. `FakeGit` keys on `(dir, argv)`.
**Tests must catch:** a fake that ignores `dir` (assert a reply canned for `/a` is *not* returned when run in `/b`); a `Physical` implementation that swallows errors; missing newline trimming (`"/repo\n"` → `"/repo"` — note this is required by the test and must therefore be in the implementation spec, not just the test).
**Deletion check:** drop the `dir` from `FakeGit`'s key — the wrong-directory test must go red.

### Task 3 — `Worktree.Resolve`

**Contract:** `Resolve(path string, git GitRunner, p PathOps) (Worktree, error)` — `Physical` ∘ `NormalizePath` on input, `git rev-parse --show-toplevel` in that directory, `Physical` ∘ `NormalizePath` on output.
**Tests must catch:** a subdirectory not walking up (the kbench case); the **input-side** `Physical` being skipped — assert `FakeGit.Ops[0]` records the *physical* dir, so a symlinked input reaches git resolved; the **output-side** `Physical` being skipped — have git return a symlinked path and assert the result is the physical one; a non-repo not erroring; `Repo()` folding case (it must not).
**Deletion check:** remove either `Physical` call independently — a distinct test must go red for each. This is the check the previous draft failed.

### Task 4 — `Clock`, `IDGen`

**Contract:** `Now() time.Time`; `NewID() ActorID`. `FixedClock`, `NewFixedIDGen(seq...)`.
**Tests must catch:** a random generator producing the wrong shape; a fixed generator not advancing.

### Task 5 — `Runner`, `ExecRunner`, `FakeRunner`

**Contract:** the state model above, verbatim.
**Tests must catch:** a fake that dies on any signal (assert alive after `Signal`); `SetExited` not unblocking `Wait`; the ops log missing the signal entry (this is the one that fails if the handle does not write back to the Runner).
**Note:** every test here must have a timeout guard; `Wait` blocking forever is a real failure mode of a wrong implementation.

### Task 6 — `Registry`, `StartArgs`, `TreeOccupiedError`, `PolicyTable`

**Contract:** value semantics with a real map copy; slice values; `Get(Worktree)`; `TreeOccupiedError` carries `Mode`.
**Tests must catch:** a second actor on one tree not being refused; **case-differing spellings not colliding** (the milestone's central invariant — and assert the complementary fact on case-sensitive platforms, that they are correctly distinct); a linked worktree of the same repo being wrongly refused; `--same-tree` orphaning the incumbent (assert both remain in `Get`); `Register` mutating the receiver on failure (register, fail, assert the original is unchanged — this is what catches the reference-type lie).
**Deletion check:** replace the map copy with a direct assignment — the mutation test must go red.

### Task 7 — `NamingTable`

**Contract:** free prose, substring match, all candidates returned.
**Tests must catch:** a name with a space being rejected (it must not be); an ambiguous label returning only one candidate; a lookup matching `Description` as well as `Name`.

### Task 8 — `Message`, `Enqueue`

**Contract:** `(out []Message, dropped Message, ok bool)`.
**Tests must catch:** an implementation that drops the *incoming* message instead of collapsing (assert the surviving element's identity **and** position, not just length); a control message being dropped at capacity; `ok` not reporting a drop.

### Task 9 — `Actor`

**Contract:** mutex+cond queue applying `Enqueue` at insertion; drains control-first.
**Tests must catch:** control not being drained first. **Design the fixture so collapse and the assertion do not fight** — distinct kinds for the normal messages, since `Enqueue` collapses same-kind. The previous draft's version of this test deadlocked for exactly that reason.
**Deletion check:** replace `Enqueue` with a plain `append` — a capacity or collapse test must go red, or `Enqueue` is orphaned.

### Task 10 — `Store`

**Contract:** snapshot `Save`/`Load` of registry + naming + policy; missing file ⇒ empty, no error.
**Tests must catch:** naming not surviving a round trip; a missing file erroring; policy defaulting wrongly.

### Task 11 — `Couch`, `Spawn`, and out-of-process liveness

**Contract:** `newTestEnv(t)` returns `{Couch, Runner *FakeRunner, Git *FakeGit, Clock FixedClock, Dir string}` wired with `NewFixedIDGen("ah8d","b2c1")` and a fixed clock. `Spawn` resolves → policy → register → `Runner.Start` → record `{PID, Identity}` → **`Store.Save`** → `Wait`.
**Tests must catch:** `Save` firing after `Wait` (assert the snapshot is on disk while the child is still alive — this is what makes `couch list` work at all); a refused spawn starting a process anyway; a failed `Start` leaving the tree registered; `StartedAt` not coming from the clock; `Identity` not being recorded.
**Also:** `IsLive(record)` recomputes `procutil.Identity(pid)` and compares. Tests must catch a stale record with a reused PID reading as alive.

### Task 12 — Name and description resolve to an actor, and both mutate

Closes Done-when **3** (`name AND description both resolve to the right actor, both change mid-session`) — orphaned in both previous drafts.
**Contract:** `Couch.ResolveRef(ref string) ([]ActorRecord, error)` composing `NamingTable.Lookup` (tree) with `Registry.Get` (actors).
**Tests must catch:** a rename mid-session not taking effect; a description change not taking effect; a lookup returning a tree with no live actor being treated as a match; ambiguity collapsing to one result.

### Task 13 — Name survives revival

Closes Done-when **4**. Real lifecycle: spawn, name, exit the child, unregister, spawn again, assert the name still resolves.

### Task 14 — Description sidecar

The cached half of Done-when 3's description. Child writes `<store>/desc/<key>`; `Couch.Describe` reads it. Not the published-status artifact returning: a one-line **label** tolerates staleness because a stale label still finds the right tree; state does not.

### Task 15 — Operations, table-driven dispatch, CLI

**Contract:** `Operations()` returns all six — `start`, `list`, `show`, `stop`, `name`, `describe`. `couchcmd` builds its dispatch map **from** `Operations()` and exposes it for test; there is no argv switch.
**The audit asserts identity, not overlap:** the CLI's reachable operation set must equal `Operations()` exactly — `reflect.DeepEqual` on sorted key sets. A hand-written expected list, or a `bytes.Contains` grep for `switch args[0]`, is a spot-check and is what Done-when 6 explicitly rules out.
**Arity assertion must not be tautological:** assert each operation's `Args` against a per-operation expectation declared in the *test*, not against itself.
**CLI signature:** `RunWithRuntime(args []string, stdin io.Reader, stdout, stderr io.Writer, rt Runtime) int`, with `Run` as the thin `OSRuntime{}` wrapper — `termcmd/run.go:46-50` verbatim. Tests call `RunWithRuntime` with a temp store so they never read the developer's real `~/.local/share/pair`.
**`start` renders worktree-or-switch** from `TreeOccupiedError`, shaped by its `Mode` (Done-when 2).
**Makefile:** add `couch` to `GO_BINS` (`Makefile.local:30`) **and** a per-binary recipe — `build:`/`install:` iterate `$(GO_BINS)` (`:42`,`:46`), so the recipe alone builds nothing. `PAIR_HOME_LDFLAGS` is pair-specific; couch needs its own or none. Note `PAIR_GO_SRCS` is `$(shell find cmd …)` (`:272`), so couch sources become prerequisites of `bin/pair` — add a check that `make pair && ./bin/pair --help` still works (Done-when 7).

### Task 16 — Live conformance, real-vs-fake

**Gate:** `PAIR_LIVE_COUCH=1`, `t.Skip` otherwise, **no build tag** — `harness_tty_live_test.go:543` is the house pattern, and a `//go:build` tag would stop the file compiling under `go test ./cmd/...` so it rots invisibly.
**Conformance means comparing two implementations, not running each separately.** Encode scenarios once — `exit 3`; a child that ignores SIGINT; a child that exits on SIGTERM — and assert `ExecRunner` and `FakeRunner` agree on `Alive`/`Wait`/signal outcome for each. Driving the fake by hand to the asserted value tests nothing.
**Flake control:** `ExecRunner.Start` returns once `cmd.Start()` succeeds, so a signal sent immediately can race the child reaching its `trap`. Poll for a readiness marker before signalling; do not rely on a short `sleep`.
**`git` conformance** (ARCH-MOCK names git explicitly): a real temp repo **and a real linked worktree**; assert `Resolve` from a subdirectory of each returns the right root, that the two are distinct, and that `FakeGit` returns the same answers for the same inputs.

### Task 17 — Smoke: host one real `pair` child

Manual; record in the issue `## Log`.
**What it settles:** does `pair` start correctly as a couch child, with zellij and nvim underneath, when couch hands it the real terminal. **What it does not:** attach/detach or multi-child routing — those need `#146`'s pty. Record which question the result answers so a negative is not over-read.

- [ ] `go build ./cmd/couch && ./couch start ../pair`
- [ ] second shell: `./couch list`, `./couch show ../pair` — proves the out-of-process read path
- [ ] `./couch start ../pair` again → refused, incumbent named, offer shaped by policy mode
- [ ] `./couch start ../kbench/competition/arc-agi-3` → registers under `.../kbench`
- [ ] Write the layering-fork answer into `## Log`

---

## Acceptance mapping

Walked against the issue's `## Done when`, in order. This mapping is a claim to verify at close, not evidence.

| # | Bullet | Closed by |
|---|--------|-----------|
| 1 | registers / lists / spawns against a peer repo, claude stack, no issue required | T11 (spawn), T15 (list), T17 (real stack) |
| 2 | second actor on a tree refused by name registration, worktree-or-switch offered | T6 (refusal), T15 (rendering, shaped by Mode) |
| 3 | operator name **and** agent description both resolve to the right actor; both change mid-session | T12 (resolution + mutation), T14 (description source) |
| 4 | a name assigned before an actor dies still resolves after revival | T13 |
| 5 | per-repo concurrency policy read from a recorded source, not inferred | T6 (`PolicyTable` consumed by the refusal), T10 (loading) |
| 6 | every operation invocable with structured args; every state queryable; audited over the operation set | T15 (identity audit, `ArgSpec`, `show`) |
| 7 | `pair-go` standalone unaffected | T15 (`make pair` check) |

## Verification before close

- `go test ./cmd/... -race` green, with timeouts on every test that can block.
- `PAIR_LIVE_COUCH=1 go test ./cmd/internal/couchcore/` green.
- `make build` produces both `pair` and `couch`; `./bin/pair --help` still works.
- Every deletion check in Tasks 1–9 confirmed red when the seam is removed.
- The case-spelling collision test passes on darwin and its complement passes on linux.
