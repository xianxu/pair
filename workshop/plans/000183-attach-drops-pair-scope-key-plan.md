# Attach Drops PAIR_SCOPE_KEY — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A reattached thread's zellij frame shows `<agent> (<count>)` again. The title poller's launch contract becomes one declaration that both launch paths hand to the spawn.

**Architecture:** The title poller stops inheriting its session variables from whatever the launcher happened to export. It receives them explicitly: a `titlepoller.SessionEnv` is built by a positional `NewSessionEnv(dataDir, scopeKey)`, passed to `SpawnTitlePoller`, and rendered into the child's environment. This mirrors how `SpawnSessionWatcher` already takes `scopeKey` as an argument. Attach resolves the scope the same way create does. `contextcmd` reports a missing scope key as its own status instead of the silent zero it shares with an unbound session. Every check is tied to the contract's own names or to what the poller *reads*, observed through an injected `getenv`, never to a hand-kept list of names: two hand-kept lists are what drifted.

**Tech Stack:** Go. `cmd/internal/{contextcmd,titlepoller,launcher,dispatcher}`. Tests are colocated `_test.go` files, run with `go test`; the full suite with `make test`.

Issue: `workshop/issues/000183-attach-drops-pair-scope-key.md` (Spec + Done-when are the requirement).

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `SessionEnv` / `NewSessionEnv` / `Environ` | `cmd/internal/titlepoller/runcli.go` | new |
| `optionsFromCLI` | `cmd/internal/titlepoller/runcli.go` | new (extracted from `RunCLI`) |
| `EnvFrom`, `EnvDataDir`, `EnvScopeKey` | `cmd/internal/contextcmd/contextcmd.go` | new |
| `ExitNoScopeKey` / `RunWithRuntime` | `cmd/internal/contextcmd/contextcmd.go` | modified (distinct status; gains `stderr`) |
| `titlePollerSpawn` | `cmd/internal/launcher/osruntime.go` | new (replaces `titlePollerArgv`) |

- **SessionEnv**: what whoever spawns a title poller must hand it: the data dir (`PAIR_DATA_DIR`) and the scope key (`PAIR_SCOPE_KEY`). It lives in `runcli.go`, **beside the parse it inverts** (`optionsFromCLI`), because the two change together. A new production file would also have to join `artifactpath`'s exhaustive source inventory (`manifest.go:704`, enforced by `TestProductionArtifactReferencesAreExactlyClassified`); `runcli.go` is already listed.
  - **Shape:** Its fields are unexported, and **`NewSessionEnv(dataDir, scopeKey string)` is positional on purpose.** A new poller read adds a parameter, and both launcher call sites then fail to compile until they supply it. A keyed literal would compile with the new field silently empty. `Environ` would then hand the child an explicit empty `PAIR_X=`, which overrides a correct inherited value, because exec keeps the last duplicate key. The plan review found that hazard.
  - **Relationships:** 1:1 with a poller spawn. The two launcher call sites (create, attach) construct it, and `titlePollerSpawn` renders it.
  - **DRY rationale:** It replaces the two hand-maintained export lists (`createflow.go:553-560`, `lifecycle.go:44-47`) *as the poller's source*. The ambient lists stay, because they legitimately differ: they feed zellij and its panes, not the poller.
  - **Boundary, stated rather than overclaimed:** The contract is tied to reads made through `contextcmd.EnvFrom` and `optionsFromCLI`. Today those are every env read in both packages: `git grep "os.Getenv\|os.LookupEnv\|os.Environ" cmd/internal/titlepoller cmd/internal/contextcmd` finds only `EnvFromOS`'s four, which become `EnvFrom(os.Getenv)`. A future direct `os.Getenv` in either package would bypass the test, and `SessionEnv`'s doc comment says so. A static guard was weighed and rejected: the injected-`getenv` pattern it would have to allow is the same shape as an evasion (`g := os.Getenv; g("PAIR_X")`). It would be a partial guard presented as a closing one, which the lessons warn against.
- **optionsFromCLI**: `RunCLI`'s argv and env parse without its side effects (`signal.Ignore`, the run loop), so the poller's own env reads can be observed.
- **EnvFrom**: `contextcmd.EnvFromOS` generalised over an injected `getenv`. The two name constants are the single spelling of the variables.
- **ExitNoScopeKey**: `RunWithRuntime` returns `2` with a stderr reason when `PairScopeKey == ""`. An empty key can never match a session, because `sessionledger.validateRecord` rejects scope-less records (`record.go:357`). An unbound or provisional session keeps status 0 with nothing printed. `2` is also the dispatcher's usage and unknown-route code, and stderr tells them apart; the atlas records this.
- **titlePollerSpawn**: `(argv, extraEnv)` for the title sidecar. The argv shape is unchanged (`<exe> title <tag> <agent> <session>`, which the single-instance guard matches), and `extraEnv` is `env.Environ()`.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ProcOps.SpawnTitlePoller` | `cmd/internal/launcher/runtime.go` | modified (gains `titlepoller.SessionEnv`) | `spawnDetached` → `exec.Cmd` |
| `fakeRuntime.SpawnTitlePoller` | `cmd/internal/launcher/createflow_test.go` | modified (records the env the child starts with) | — |
| `dispatchContext` | `cmd/internal/dispatcher/dispatcher.go` | modified (forwards stderr) | `pair context` CLI |

- **ProcOps.SpawnTitlePoller**: `OSRuntime` passes `titlePollerSpawn(...)` straight to `spawnDetached(argv, extraEnv)` (`spawnDetached` sets `cmd.Env = append(os.Environ(), extraEnv...)`).
  - **Injected into:** `runCreate` and `AttachExistingSession`, through the `Runtime` seam. Only `OSRuntime` and `fakeRuntime` implement it (verified by the plan review).
- **fakeRuntime.SpawnTitlePoller**: it records the environment the child would start with: the exports so far, overlaid by the spawn's `Environ()`, which is the precedence exec applies. The regression test reads this record through the contract's own names and through `contextcmd.EnvFrom`, the poller's real reader.

`AttachExistingSession`'s exported signature is **unchanged**, so its live-test caller (`couchcore/park_lifecycle_live_test.go:473`) compiles as-is. That env has no `Cwd`, so its scope key resolves to empty, which is harmless for a quit-intent test.

### Transition note (for verification)

`titlepoller.Run`'s single-instance guard keeps whichever poller started first (`run.go:95-100`). A scope-less poller spawned by a pre-fix attach therefore outlives the upgrade until its process group dies. A couch detach signals the whole actor group (`detach.go:89`), so **the real-stack check is: install, then detach and reattach.** A relaunch works too.

### Imports this touches (so no step stalls on the compiler)

| File | Adds |
|---|---|
| `launcher/runtime.go`, `osruntime.go`, `createflow.go`, `lifecycle.go`, `createflow_test.go`, `osruntime_test.go`, `poller_env_test.go` | `github.com/xianxu/pair/cmd/internal/titlepoller` |
| `launcher/createflow_test.go` | `maps` (plus `strings`, which it already has) |
| `launcher/poller_env_test.go` | `strings`, `github.com/xianxu/pair/cmd/internal/contextcmd` |
| `titlepoller/runtime.go` | `io` |
| `titlepoller/runcli.go`, `dispatcher/dispatcher_test.go` | `github.com/xianxu/pair/cmd/internal/contextcmd` |

Run `gofmt -l cmd/` after every task; the fake's struct gains a field.

---

## Chunk 1: Contract, wiring, verification

### Task 0: Enter implementation

- [ ] **Step 1:** After plan approval, run `sdlc change-code --issue 183`. It owns branching and the plan-quality gate, and it asks for `estimate_hours` plus the `## Estimate` block only after the plan clears. Point the issue's `## Plan` at this file.

### Task 1: `contextcmd` names its variables and gives a missing scope its own status

**Files:**
- Modify: `cmd/internal/contextcmd/contextcmd.go`
- Modify: `cmd/internal/contextcmd/contextcmd_test.go`
- Modify: `cmd/internal/titlepoller/runtime.go:98-102` (caller of `Run`)
- Modify: `cmd/internal/dispatcher/dispatcher.go:239-243` (caller of `Run`)
- Modify: `cmd/internal/dispatcher/dispatcher_test.go`

- [ ] **Step 1: Write the failing tests.** In `contextcmd_test.go`:

```go
// An absent scope key is its own status, not the silent zero an unbound session
// gets (pair#183). Against the ESTABLISHED fixture, so the only reason nothing
// can print is the missing key.
func TestRunWithoutAScopeKeyIsItsOwnStatus(t *testing.T) {
	t.Parallel()
	runtime := contextRuntime(t, true)
	var stdout, stderr bytes.Buffer
	code := RunWithRuntime([]string{"T", "codex"}, Env{}, runtime, &stdout, &stderr)
	if code != ExitNoScopeKey || stdout.Len() != 0 || !strings.Contains(stderr.String(), EnvScopeKey) {
		t.Fatalf("code=%d stdout=%q stderr=%q, want ExitNoScopeKey and a reason naming %s",
			code, stdout.String(), stderr.String(), EnvScopeKey)
	}
}
```

Update the two existing tests to pass `&stderr` (a new `bytes.Buffer`). In `TestRunProvisionalBindingPrintsNothing`, also assert `stderr.Len() == 0`: that is the other half of the distinction.

In `dispatcher_test.go`, beside `TestDispatchContextReturnsHelperOutput`:

```go
// The CLI surface for a missing scope: a reason on stderr and a non-zero status,
// where it used to print nothing and exit 0 -- which is how pair#183 hid.
func TestDispatchContextWithoutAScopeKeySaysWhy(t *testing.T) {
	home, data := writeContextFixture(t)
	t.Setenv("HOME", home)
	t.Setenv("PAIR_DATA_DIR", data)
	t.Setenv("PAIR_SCOPE_KEY", "")

	res := Dispatch([]string{"context", "T", "claude"})
	if res.ExitCode != contextcmd.ExitNoScopeKey || res.Stdout != "" || !strings.Contains(res.Stderr, "PAIR_SCOPE_KEY") {
		t.Fatalf("Dispatch(context) = %+v, want ExitNoScopeKey with a stderr reason", res)
	}
}
```

- [ ] **Step 2: Run to confirm red.** `go test ./cmd/internal/contextcmd/ ./cmd/internal/dispatcher/` should fail to compile (`ExitNoScopeKey`, `EnvScopeKey`, and the 5-arg `RunWithRuntime` are undefined).

- [ ] **Step 3: Implement.** In `contextcmd.go`:

```go
// The Pair-session variables this package reads. Named once, because the title
// poller's spawn contract (titlepoller.SessionEnv) has to render exactly these
// (pair#183).
const (
	EnvDataDir  = "PAIR_DATA_DIR"
	EnvScopeKey = "PAIR_SCOPE_KEY"
)

// ExitNoScopeKey is Run's status when no repo scope key was supplied.
//
// Kept apart from "no established binding" -- status 0, nothing printed -- which
// is an ordinary state for a session that has not bound yet. An empty key is a
// caller that forgot to pass scope: sessionledger rejects every record without
// one, so no session can ever match it. pair#183 was exactly that caller, and it
// read as a missing feature because both states printed the same nothing.
const ExitNoScopeKey = 2

func EnvFromOS() Env { return EnvFrom(os.Getenv) }

// EnvFrom reads Env through getenv -- injected, so a spawner's contract can be
// checked against what this package actually reads rather than against a list.
func EnvFrom(getenv func(string) string) Env {
	return Env{
		Home:         getenv("HOME"),
		XDGDataHome:  getenv("XDG_DATA_HOME"),
		PairDataDir:  getenv(EnvDataDir),
		PairScopeKey: getenv(EnvScopeKey),
	}
}
```

`Run(args, env, stdout, stderr io.Writer)` passes `stderr` through. `RunWithRuntime(args, env, runtime, stdout, stderr io.Writer)` inserts, right after the `len(args) < 2` guard:

```go
	if env.PairScopeKey == "" {
		fmt.Fprintf(stderr, "pair context: %s is not set, so no session can match; whoever started this did not pass the repo scope\n", EnvScopeKey)
		return ExitNoScopeKey
	}
```

Callers:
- `dispatcher.go` `dispatchContext`: declare `var stdout, stderr bytes.Buffer`, pass `&stderr`, and return `Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}`.
- `titlepoller/runtime.go` `ContextCount`: pass `io.Discard`, with this comment:

```go
	// io.Discard: the poller's stdio is /dev/null anyway, and after pair#183 it
	// is handed its scope by the spawn contract (SessionEnv) -- an empty key here
	// means the launcher could not resolve a repo root at all, where no session
	// could match regardless.
```

- [ ] **Step 4: Run to confirm green.** `go test ./cmd/internal/contextcmd/ ./cmd/internal/dispatcher/ ./cmd/internal/titlepoller/` should pass. `TestDispatchContextReturnsHelperOutput` (`dispatcher_test.go:229`) and `cmd/pair-go`'s route test (`helper_equivalence_test.go:27`) already set `PAIR_SCOPE_KEY`, so they are unaffected.

- [ ] **Step 5: Commit.** `#183: contextcmd names its variables, and a missing scope key is its own status`

### Task 2: `titlepoller.SessionEnv`, the one declaration, tied to the reads

**Files:**
- Modify: `cmd/internal/titlepoller/runcli.go` (add `SessionEnv`, `NewSessionEnv`, `Environ`; extract `optionsFromCLI`)
- Create: `cmd/internal/titlepoller/runcli_test.go`

- [ ] **Step 1: Write the failing test** (`runcli_test.go`):

```go
package titlepoller

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/contextcmd"
)

// The contract is tied to the reads, not to a list. Every Pair-session variable
// the poller reads -- OBSERVED through contextcmd.EnvFrom and the CLI parse --
// must be one SessionEnv supplies, so a new read fails here until the contract
// carries it; NewSessionEnv then gains a parameter, and both launcher call sites
// stop compiling until they fill it (pair#183). PAIR_ is the repo's prefix for
// session variables; the rest of what the poller reads (HOME, XDG_DATA_HOME,
// CMUX_WORKSPACE_ID) belongs to the operator's terminal and is inherited.
func TestSessionEnvSuppliesEveryPairVariableThePollerReads(t *testing.T) {
	const dataDir, scopeKey = "/data/repos/k", "k"
	supplied := map[string]string{}
	for _, kv := range NewSessionEnv(dataDir, scopeKey).Environ() {
		name, value, _ := strings.Cut(kv, "=")
		supplied[name] = value
	}
	var read []string
	getenv := func(name string) string {
		read = append(read, name)
		return supplied[name]
	}

	context := contextcmd.EnvFrom(getenv)
	opts, ok := optionsFromCLI([]string{"T", "claude", "📁repo-T"}, getenv)
	if !ok {
		t.Fatal("optionsFromCLI refused a complete argv")
	}

	for _, name := range read {
		if strings.HasPrefix(name, "PAIR_") && supplied[name] == "" {
			t.Errorf("the poller reads %s but SessionEnv does not supply it", name)
		}
	}
	// The round trip, so a supplied-but-misnamed variable cannot pass.
	if context.PairDataDir != dataDir || context.PairScopeKey != scopeKey || opts.DataDir != dataDir {
		t.Fatalf("round trip lost the contract: context=%+v opts.DataDir=%q", context, opts.DataDir)
	}
}
```

- [ ] **Step 2: Run to confirm red.** `go test ./cmd/internal/titlepoller/ -run SessionEnv` should fail to compile (`NewSessionEnv` and `optionsFromCLI` are undefined).

- [ ] **Step 3: Implement** in `runcli.go`:

```go
// SessionEnv is what whoever spawns a title poller must hand it: the ONE
// declaration of its launch contract (pair#183).
//
// The launcher starts a poller from two places, create and attach. Both used to
// satisfy this by exporting variables before the spawn and letting the child
// inherit them, each from its own hand-kept list. Attach's lacked PAIR_SCOPE_KEY,
// and every reattached thread lost its context meter with nothing to say so. It
// lives beside optionsFromCLI because Environ is that parse, inverted.
//
// Its reach is the reads made through contextcmd.EnvFrom and optionsFromCLI --
// every env read in this package and contextcmd today. A direct os.Getenv added
// elsewhere would not be seen by TestSessionEnvSuppliesEveryPairVariableThePollerReads.
// (optionsFromCLI's adapt.DataDir() fallback reads PAIR_DATA_DIR in another
// package, unseen -- harmless: the same variable, reached only when it is empty.)
//
// HOME, XDG_DATA_HOME and CMUX_WORKSPACE_ID are deliberately absent: they belong
// to the operator's terminal, and inheriting them is correct.
type SessionEnv struct {
	dataDir  string
	scopeKey string
}

// NewSessionEnv is POSITIONAL on purpose: a new poller read adds a parameter
// here, and both launcher call sites then fail to compile until they supply it.
// A keyed literal would compile with the new field silently empty -- and Environ
// would hand the child an explicit empty value that overrides a correct inherited
// one, since exec keeps the last duplicate key.
func NewSessionEnv(dataDir, scopeKey string) SessionEnv {
	return SessionEnv{dataDir: dataDir, scopeKey: scopeKey}
}

// Environ renders the contract as KEY=value entries for the child, under the
// names contextcmd reads.
func (e SessionEnv) Environ() []string {
	return []string{
		contextcmd.EnvDataDir + "=" + e.dataDir,
		contextcmd.EnvScopeKey + "=" + e.scopeKey,
	}
}
```

Move the parse into `optionsFromCLI(args, getenv) (Options, bool)`. It returns `false` when `len(args) < 2`, reads `getenv(contextcmd.EnvDataDir)` with the `adapt.DataDir()` fallback, then reads `CMUX_WORKSPACE_ID` and the optional session. `RunCLI` becomes: parse (returning 0 on `!ok`), `signal.Ignore(syscall.SIGHUP)` with its existing comment, then `Run(opts, NewOSRuntime())`. Behaviour is unchanged.

- [ ] **Step 4: Run to confirm green.** Run `go test ./cmd/internal/titlepoller/ ./cmd/internal/artifactpath/`. The second package is the source-inventory guard, which confirms no new production file was introduced.

- [ ] **Step 5: Commit.** `#183: titlepoller.SessionEnv declares the poller's launch contract`

### Task 3: The regression, measured against the contract's own names (red)

**Files:**
- Create: `cmd/internal/launcher/poller_env_test.go`
- Modify: `cmd/internal/launcher/createflow_test.go` (the `fakeRuntime` struct around `:103`, and `SpawnTitlePoller` at `:238`)

- [ ] **Step 1: Make the fake record the environment the child starts with.** Add the field `pollerEnvs []map[string]string` (the env each title poller started with). In today's `SpawnTitlePoller`, after the `pollers` append, add `f.pollerEnvs = append(f.pollerEnvs, maps.Clone(f.env))`. Task 4 adds the overlay once the spawn carries a contract.

- [ ] **Step 2: Write the failing test** (`poller_env_test.go`):

```go
package launcher

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/contextcmd"
	"github.com/xianxu/pair/cmd/internal/titlepoller"
)

// pair#183: every reattached thread lost its context meter because attach
// spawned the title poller without PAIR_SCOPE_KEY, and an empty key matches no
// session. Asserted as what the poller STARTS WITH, on both launch paths, so the
// two cannot drift again -- and against the names the contract itself declares,
// never a list kept here: two hand-kept lists are what drifted.
func TestTitlePollerStartsWithItsWholeContractOnBothPaths(t *testing.T) {
	want := mustScope(t, "/home/u/work").Key // baseOpts' Cwd; RepoRoot unset

	t.Run("attach", func(t *testing.T) {
		rt := newFakeRuntime()
		opts := baseOpts(LaunchArgs{})
		if _, err := AttachExistingSession(opts, opts.Env, rt, "live", "📁work-live", "claude"); err != nil {
			t.Fatal(err)
		}
		assertPollerStartsWith(t, rt, want, opts.Env.DataDir)
	})

	t.Run("create", func(t *testing.T) {
		rt := newFakeRuntime()
		rt.uuids = []string{"MINTED-1"}
		opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"})
		if code, err := run(t, opts, rt); err != nil || code != 0 {
			t.Fatalf("create = %d, %v", code, err)
		}
		assertPollerStartsWith(t, rt, want, opts.Env.DataDir)
	})
}

func assertPollerStartsWith(t *testing.T, rt *fakeRuntime, scopeKey, dataDir string) {
	t.Helper()
	if len(rt.pollerEnvs) != 1 {
		t.Fatalf("title pollers spawned = %d, want 1", len(rt.pollerEnvs))
	}
	started := rt.pollerEnvs[0]

	// Every name the contract declares arrives non-empty. The names come from
	// the contract, so a new field joins this check without anyone listing it,
	// and an explicit empty value -- which would override a good inherited one --
	// fails here too. (The arguments only have to be non-empty for Environ to
	// render the names; the names are what is being read.)
	for _, kv := range titlepoller.NewSessionEnv("probe", "probe").Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if started[name] == "" {
			t.Errorf("the poller starts without %s -- its context meter cannot resolve", name)
		}
	}
	// And the values are the right ones, read through the poller's own reader.
	got := contextcmd.EnvFrom(func(name string) string { return started[name] })
	if got.PairScopeKey != scopeKey || got.PairDataDir != dataDir {
		t.Errorf("the poller reads scope %q / data dir %q, want %q / %q", got.PairScopeKey, got.PairDataDir, scopeKey, dataDir)
	}
}
```

- [ ] **Step 3: Run to confirm red, for the right reason.** Run `go test ./cmd/internal/launcher/ -run TestTitlePollerStartsWithItsWholeContractOnBothPaths -v`. Expected: `/attach` fails with `starts without PAIR_SCOPE_KEY` plus a scope mismatch against `""`, and `/create` passes. That is the bug, reproduced at the seam. **If `/create` also fails, stop:** the fixture is wrong, not the code. The plan review reproduced exactly this red.

- [ ] **Step 4: Do not commit yet.** The red test lands with Task 4's commit, so every commit stays green.

### Task 4: Both launch paths hand the poller its contract (Task 3 goes green)

**Files:**
- Modify: `cmd/internal/launcher/runtime.go:87-89` (the `ProcOps.SpawnTitlePoller` signature and doc)
- Modify: `cmd/internal/launcher/osruntime.go:355-383` (`SpawnTitlePoller`, with `titlePollerArgv` replaced by `titlePollerSpawn`)
- Modify: `cmd/internal/launcher/osruntime_test.go:~490-505` (`TestSidecarSpawnArgvSelfExecsPair`)
- Modify: `cmd/internal/launcher/createflow.go:608`
- Modify: `cmd/internal/launcher/lifecycle.go:42-62`
- Modify: `cmd/internal/launcher/createflow_test.go` (the fake's `SpawnTitlePoller` overlays the contract)

- [ ] **Step 1: Pin the pure spawn first.** In `TestSidecarSpawnArgvSelfExecsPair`, replace the `titlePollerArgv` call with:

```go
	contract := titlepoller.NewSessionEnv("/data/repos/k", "k")
	tp, tpEnv := titlePollerSpawn(exe, "work", "claude", "📁pair-work", contract)
	// (keep the existing argv assertion on tp, unchanged)
	if !reflect.DeepEqual(tpEnv, contract.Environ()) {
		t.Fatalf("title poller env = %v, want its contract %v -- a dropped env is silent, exactly as pair#183 was", tpEnv, contract.Environ())
	}
```

- [ ] **Step 2: Implement.**

`runtime.go`:

```go
	// SpawnTitlePoller backgrounds `pair title` (detached), the per-tag
	// frame/cmux title singleton. env is its launch contract, handed to the child
	// explicitly rather than inherited from whatever was exported first -- the
	// inheritance is what pair#183 lost on attach.
	SpawnTitlePoller(tag, agent, session string, env titlepoller.SessionEnv)
```

`osruntime.go`:

```go
func (r OSRuntime) SpawnTitlePoller(tag, agent, session string, env titlepoller.SessionEnv) {
	spawnDetached(titlePollerSpawn(runningPairExe(r.PairHome), tag, agent, session, env))
}

// titlePollerSpawn builds the detached spawn for the title sidecar: its argv and
// the environment it is handed. Pure (exe injected) so a test pins both halves --
// spawnDetached swallows a start error, and a dropped environment is silent in
// exactly the way pair#183 was. The argv must start "<…>/pair title <tag>
// <agent>", the shape titlepoller's single-instance argv guard matches. The env
// entries are appended after os.Environ(), and exec keeps the LAST duplicate, so
// the contract wins over any stale inherited value.
func titlePollerSpawn(exe, tag, agent, session string, env titlepoller.SessionEnv) ([]string, []string) {
	return []string{exe, "title", tag, agent, session}, env.Environ()
}
```

Delete `titlePollerArgv`. The plan review confirmed Step 1's test was its only other user. Before deleting, sweep comments that name it with `git grep -n titlePollerArgv -- cmd atlas`.

`createflow.go:608`:
`rt.SpawnTitlePoller(chosenTag, agent, session, titlepoller.NewSessionEnv(dataDir, scope.Key))`
(`dataDir := env.DataDir` at `:415`, the same value exported as `PAIR_DATA_DIR` at `:554`.)

`lifecycle.go`: replace the stale comment at `:42-43`, whose "pair-shell exports these globally" premise is false and made the subset look complete. Then resolve the scope and hand over the contract:

```go
	// What zellij's attach client inherits. The title poller does NOT rely on
	// these: its contract is titlepoller.SessionEnv, handed to the spawn below.
	// Relying on them was pair#183 -- this list was a hand-copied subset of
	// create's, it lacked PAIR_SCOPE_KEY, and every reattached thread lost its
	// context meter without a word.
	rt.SetEnv("PAIR_HOME", opts.PairHome)
	rt.SetEnv("PAIR_DATA_DIR", env.DataDir)
	rt.SetEnv("PAIR_TAG", tag)
	rt.SetEnv("PAIR_SESSION_NAME", session)
	...
	// The scope the way create resolves it (createflow.go:386), not parsed back
	// out of the data dir's path shape. A root that will not resolve (cwd `/`)
	// leaves the key empty rather than refusing the attach -- the meter is
	// optional, the handoff is not -- and contextcmd reports an empty key as its
	// own status (ExitNoScopeKey).
	scopeKey := ""
	if scope, err := ResolveRepoScope(envScopeRoot(env)); err == nil {
		scopeKey = scope.Key
	}
	rt.SpawnTitlePoller(tag, agent, session, titlepoller.NewSessionEnv(env.DataDir, scopeKey))
```

Fake (`createflow_test.go`):

```go
func (f *fakeRuntime) SpawnTitlePoller(tag, agent, session string, env titlepoller.SessionEnv) {
	f.pollers = append(f.pollers, tag+"|"+agent)
	// What the child starts with: everything exported so far, overlaid by what
	// the spawn hands it -- the precedence exec gives a later duplicate key.
	started := maps.Clone(f.env)
	for _, kv := range env.Environ() {
		name, value, _ := strings.Cut(kv, "=")
		started[name] = value
	}
	f.pollerEnvs = append(f.pollerEnvs, started)
}
```

- [ ] **Step 3: Run.** `test -z "$(gofmt -l cmd/)" && go build ./... && go vet ./cmd/internal/launcher/ ./cmd/internal/couchcore/ && go test ./cmd/internal/launcher/ ./cmd/internal/titlepoller/ ./cmd/internal/contextcmd/ ./cmd/internal/artifactpath/`.
  - `gofmt -l` exits 0 even when it lists files, so the `test -z` wrapper is what makes it gate.
  - `go vet` on `couchcore` proves the live-test caller still compiles without running its pty tests, which fail in the sandbox. Task 6's unsandboxed `make test` runs them. Task 3's test should now pass on both subtests, and so should `TestSidecarSpawnArgvSelfExecsPair`. The existing `pollers` assertions (`createflow_test.go:722`, `:860`, `:886`) are unchanged.

- [ ] **Step 4: Commit** Tasks 3 and 4 together: `#183: both launch paths hand the title poller its contract; attach resolves scope like create`

### Task 5: Docs

**Files:**
- Modify: `atlas/architecture.md`, the title-poller paragraph at `:311-313`
- Modify: `atlas/go-migration-inventory.md:121`, the `contextcmd` row's "tolerant exit 0 on failure"

- [ ] **Step 1:** `architecture.md`, "Title poller" paragraph. Add that its launch contract is `titlepoller.SessionEnv` (`PAIR_DATA_DIR`, `PAIR_SCOPE_KEY`). It is built positionally by `NewSessionEnv` and handed to the spawn by both create and attach, not inherited. That fixes pair#183, where attach's hand-copied export subset silently dropped the meter. In the context-meter bullet, extend "Provisional, ambiguous, and unbound owners print nothing" with: a missing scope key is not an unbound owner, so `pair context` reports it as `ExitNoScopeKey` (2, shared with the dispatcher's usage errors and told apart by stderr) with a reason.
- [ ] **Step 2:** `go-migration-inventory.md:121`. Replace "tolerant exit 0 on failure" with "exit 0 printing nothing for an unbound/provisional owner; exit 2 with a stderr reason when `PAIR_SCOPE_KEY` is missing (pair#183)".
- [ ] **Step 3:** Run `git grep -n "titlePollerArgv\|tolerant exit 0" -- atlas cmd docs`. The result should be empty.
- [ ] **Step 4: Commit.** `#183: atlas — the poller's launch contract, and a missing scope is not an unbound owner`

### Task 6: Verification

- [ ] **Step 1: Mutation sweep.** Write `mutate183.py` in the session scratchpad. Templates: #221's `/private/tmp/claude-501/-Users-xianxu-workspace-pair/0d000e5f-f61b-4ff7-9878-17f1e60a5654/scratchpad/mutate221.py`, and the plan reviewer's `…/scratchpad/review-r2-mutate183.py` in the same directory, which already carries the fixed needles below. Assert each needle occurs exactly once, restore from saved bytes (never `git checkout`), and check the tree is clean afterwards.
  - **A non-zero exit is not a kill.** For rows 1–6 and 8, the script must find the named `--- FAIL: <test>` line. For row 7, it must find the two `not enough arguments in call to titlepoller.NewSessionEnv` errors. A plain exit code is what let an unused-variable compile error pass as a caught mutation in round 2.
  - Each row must turn the named check red:

  | # | file: needle → replacement | must fail |
  |---|---|---|
  | 1 | `lifecycle.go`: `ResolveRepoScope(envScopeRoot(env)); err == nil` → `ResolveRepoScope("/"); err == nil` (attach failing to resolve a root; keeps `scopeKey` read, so it compiles) | `…OnBothPaths/attach` (and `/create` stays green) |
  | 2 | `createflow.go`: `titlepoller.NewSessionEnv(dataDir, scope.Key)` → `titlepoller.NewSessionEnv(dataDir, "")` | `…OnBothPaths/create` (the empty override beats the ambient export) |
  | 3 | `runcli.go`: delete the line `contextcmd.EnvScopeKey + "=" + e.scopeKey,` | `TestSessionEnvSuppliesEveryPairVariableThePollerReads` and `…/attach`. **`/create` stays green by design**: its ambient export still carries the key. That is the reason the ambient list is not the poller's contract. |
  | 4 | `osruntime.go`: `session}, env.Environ()` → `session}, nil` | `TestSidecarSpawnArgvSelfExecsPair` |
  | 5 | `contextcmd.go`: `if env.PairScopeKey == "" {` → `if false {` | `TestRunWithoutAScopeKeyIsItsOwnStatus` |
  | 6 | `contextcmd.go`: `func EnvFrom(getenv func(string) string) Env {` → the same line followed by `\n\t_ = getenv("PAIR_AGENT")` (a *new* poller read) | `TestSessionEnvSuppliesEveryPairVariableThePollerReads`, the runtime half of the class |
  | 7 | `runcli.go`: `func NewSessionEnv(dataDir, scopeKey string)` → `func NewSessionEnv(dataDir, scopeKey, agent string)` | `go build ./...` at both launcher call sites (`createflow.go`, `lifecycle.go`), the compile-time half of the class |
  | 8 | `dispatcher.go`: `contextcmd.Run(args, contextcmd.EnvFromOS(), &stdout, &stderr)\n\treturn Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}` → the same with `Stderr: stderr.String(), ` removed (anchored to the context route: the bare field occurs 5 times in the file) | `TestDispatchContextWithoutAScopeKeySaysWhy` |

- [ ] **Step 2: Full suite, unsandboxed:** `env -u PAIR_SESSION_ID -u PAIR_TAG make test`. Exit 0 is required.
- [ ] **Step 3: Real stack (operator).** Run `make install`. In couch, detach a thread (`Alt+d` on the actor), then reattach it (`Enter` on its detached row). Within one poll, its frame title should read `<agent> (<count>)`. See the transition note: a poller already running without scope survives until its group dies, which is why this step detaches first.
- [ ] **Step 4: Close prep.** Tick the issue's `## Plan` checkboxes, and add a `## Log` entry with the mutation table, the `make test` result, and the operator's confirmation. Otherwise the plan-unchecked gate refuses.
- [ ] **Step 5: Close.** Run `sdlc close --issue 183 --verified '<operator real-stack confirmation; make test exit 0 + package count; mutation sweep N/N with apply-asserts>'`. Fix any review findings in the close commit, with no `sdlc issue sync` in between. Then run `sdlc pr` and `sdlc merge --yes`.

---

## ARCH notes

- **ARCH-DRY.** The poller's needs are declared once (`SessionEnv` via `NewSessionEnv`), and the variable names are spelled once (the `contextcmd` constants). Both call sites construct the value positionally; neither re-lists names. The regression reads names from the contract itself.
- **ARCH-PURE.** `SessionEnv.Environ`, `NewSessionEnv`, `optionsFromCLI`, `EnvFrom` and `titlePollerSpawn` are pure and unit-tested with no IO. `OSRuntime.SpawnTitlePoller` is one line of glue.
- **ARCH-PURPOSE.** The purpose is the class: "the next thing the poller learns to read breaks the same way". It is closed on two sides:
  - At compile time, a new contract field is a new `NewSessionEnv` parameter, which breaks both call sites (mutation 7).
  - At test time, a new read through the package readers fails the observed-reads test (mutation 6), and a call site supplying an empty value fails the launcher test (mutations 1–2).
  - The boundary is stated, not overclaimed: a *direct* `os.Getenv` added to `titlepoller` or `contextcmd` would escape. There are none today, and the reason for not adding a static guard is in Core concepts.
- **ARCH-ORDER.** No new cross-event state. The poller is fire-and-forget as before. The only ordering fact is the single-instance guard's first-wins rule. The transition note handles it rather than code, because killing a live poller on attach would change unrelated behaviour.
- **ARCH-CONSTRAINTS.** This is a per-launch path (create or attach). The added work is one SHA-256 of a path on attach, which takes microseconds. Disk, network, memory: N/A.
- **ARCH-SECURE.** The scope key is a hash of a local path, not a secret. The child's environment gains no new *kind* of value, only one the create path already exports. N/A beyond that.
- **ARCH-MOCK.** No new external dependency. The `fakeRuntime` is the existing seam, and it now records the env the child starts with.

## Revisions

### 2026-09-10 — plan-document review round 1 (fresh-context, applied Tasks 1–4 in a scratch clone)

Reason: the reviewer found three issues and one broken reference.
Delta:
- **The new file would fail the source-inventory guard.** `SessionEnv` moved into `runcli.go`, beside the parse it inverts (no new production file). Task 2 runs `./cmd/internal/artifactpath/`.
- **The regression test checked two named fields,** a second hand-kept list. It now iterates the contract's own `Environ()` names. The contract also gets a positional `NewSessionEnv` with unexported fields, so a new field breaks both call sites at compile time and cannot render a silent empty override.
- **Mutation row 3 wrongly expected `/create` to fail.** Corrected, with the reason it stays green. Mutation rows 6 and 7 were added for the class.
- **The `mutate221.py` reference had no location.** The template is named with its scratchpad location, and every row now carries its needle.

Advisories taken: Task 0 (`change-code`); the red test folded into Task 4's commit; close prep plus `--verified` content; the imports table; the Task 5 grep scoped to `atlas cmd docs`; the reach boundary stated in the code as well as here; the atlas note on the shared exit code 2.

### 2026-09-10 — plan-document review round 2 (same reviewer, re-applied Tasks 1–4, ran every mutation)

Reason: the design was confirmed. All round-1 issues are resolved. Task 3 goes red for the stated reason and green after Task 4. Row 7 is a real break at both call sites. The reviewer's worst case, a new parameter silenced with `""` at both call sites, now fails both launcher subtests. Two issues remained, both in the mutation table.
Delta:
- **Row 1's needle would fail to compile** (`declared and not used: scopeKey`), not fail `/attach`. It is replaced with `ResolveRepoScope("/")`, which the reviewer ran: it compiles, fails `/attach`, and leaves `/create` green. The sweep must now match the named `--- FAIL` line or the build error, never just an exit code.
- **Row 8's needle occurred 5 times.** It is now anchored to the context route; the reviewer ran it and it applies once and fails the named test.

Both fixes are the replacements the reviewer itself executed, so there is no round 3. Advisories taken: the Task 4 gofmt gate (`test -z`), `go vet` for `couchcore` (the sandbox kills its pty tests), row 6's anchor, absolute template paths, and the `adapt.DataDir()` note in the doc comment.
