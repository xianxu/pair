# Pair Go Launcher Prototype Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a guarded `pair-go launch` prototype that reaches Pair launcher decision logic without replacing or invoking the public shell launcher.

**Architecture:** Put launcher business logic in a pure `cmd/internal/launcher` package and keep subprocess/filesystem work in thin, injectable seams (`ARCH-PURE`). Extend the existing #74 dispatcher instead of adding a parallel command parser (`ARCH-DRY`). The prototype prints the decision it would take, then exits with an explicit unsupported-after-decision code so the issue delivers a real launcher vertical slice without changing public `bin/pair` behavior (`ARCH-PURPOSE`).

**Tech Stack:** Go standard library, existing `cmd/internal/dispatcher`, fake `zellij` process tests, `go test`, `make pair-go`.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `LaunchArgs` | `cmd/internal/launcher/args.go` | new |
| `Tag` | `cmd/internal/launcher/tag.go` | new |
| `DataDir` | `cmd/internal/launcher/datadir.go` | new |
| `SessionSnapshot` | `cmd/internal/launcher/session.go` | new |
| `LaunchDecision` | `cmd/internal/launcher/decision.go` | new |

**LaunchArgs** — Parsed launch-specific argv: agent, forced tag, forwarded agent args, help flag, and unsupported subcommand errors.
- **Relationships:** 1:1 with a `pair-go launch` invocation; owns a `Tag` only for forced resume.
- **DRY rationale:** Keeps `pair-go launch` parsing behind the existing dispatcher instead of duplicating parser branches in `cmd/pair-go/main.go`.
- **Future extensions:** `continue`, `rename`, and tag-restart prompts widen this parser after the prototype has tests.

**Tag** — Normalized workspace tag, accepting either bare `demo` or `pair-demo` and rejecting empty or non `[A-Za-z0-9_-]` values. The canonical value is always bare (`demo`); zellij session names are derived only at the boundary as `pair-<tag>`.
- **Relationships:** Used by `LaunchArgs`, `SessionSnapshot`, and `LaunchDecision`.
- **DRY rationale:** Mirrors the shell launcher's `normalize_tag` as a named Go concept so later Go launcher work has one validation point.
- **Future extensions:** Length checks can move here when the Go path reaches real zellij session creation.

**DataDir** — Resolved Pair data directory from `XDG_DATA_HOME` or `$HOME/.local/share/pair`.
- **Relationships:** Provides the root for historical sidecars and future config/session files.
- **DRY rationale:** Prevents each command seam from recomputing Pair's data directory.
- **Future extensions:** Asset/data path resolution can join this with future `PAIR_HOME` discovery.

**SessionSnapshot** — In-memory view of active zellij rows and historical tag candidates relevant to the current cwd.
- **Relationships:** 1:N with zellij sessions and historical rows; consumed by `LaunchDecision`.
- **DRY rationale:** Separates "what exists" from "what should we do", matching the shell launcher's implicit stages.
- **Future extensions:** Can add queue badges, age coloring, and config-derived agent inference without changing decision callers.

**LaunchDecision** — Pure create/attach/picker-required decision for forced resume, empty state, detached sessions, and historical tags. It carries the canonical bare `Tag` and, for attach/create decisions that name zellij, the derived `SessionName` (`pair-<tag>`) so comparisons and printouts cannot accidentally mix forms.
- **Relationships:** N:1 from snapshot plus args to one decision.
- **DRY rationale:** Pulls the business rule out of command execution so unit tests do not need zellij/fzf.
- **Future extensions:** Real fzf selection can become another input shape instead of branching inside IO code.

The launcher package must not define a second stdout/stderr/exit-code result type. `cmd/internal/dispatcher.Result` remains the single process-facing result abstraction (`ARCH-DRY`). Launcher functions return domain values (`LaunchDecision`, snapshots, parse errors); the dispatcher route converts those values into `dispatcher.Result`.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ZellijSource` | `cmd/internal/launcher/zellij.go` | new | `zellij list-sessions`, `zellij --session ... action list-clients` |
| `HistorySource` | `cmd/internal/launcher/history.go` | new | filesystem scan of `draft-*.md` and `log-*.md` under Pair data dir |
| `LaunchCommand` | `cmd/internal/dispatcher/dispatcher.go` | modified | existing dispatcher route for `launch` |
| `PairGoMain` | `cmd/pair-go/main.go` | modified | process stdout/stderr exit handling |

**ZellijSource** — Reads zellij session rows and client counts through `exec.Command`.
- **Injected into:** `launcher.Run`, which converts IO into a pure `SessionSnapshot`.
- **Future extensions:** Timeout behavior can be added when the Go launcher owns real launch flow; do not port the shell `zj` timeout in this prototype unless the process fake exposes the need.

**HistorySource** — Scans cwd-prefixed historical sidecars from the resolved data dir.
- **Injected into:** `launcher.Run`.
- **Future extensions:** Queue badges and age display can be layered onto historical rows when picker UI is implemented.

**LaunchCommand** — Routes `pair-go launch` from the existing dispatcher to the launcher runner.
- **Injected into:** `cmd/pair-go` via the existing `run(args, stdout, stderr)` path.
- **Future extensions:** Other implemented subcommands can follow the same dispatcher pattern.
- **Boundary rule:** production environment reads (`os.Getenv`, `os.Getwd`) and `exec.Command` construction live in a small launcher IO constructor used by the dispatcher route. Tests may call a dispatcher test seam with an explicit launcher runtime. The launcher core returns domain outcomes; only the dispatcher maps those outcomes to stdout/stderr/exit code.

**PairGoMain** — No business logic; writes dispatcher-returned streams and exits.
- **Injected into:** none.
- **Future extensions:** May eventually become the public `pair` binary entrypoint in #77, but not here.

## Revisions

### 2026-06-29 — Close review correction

Reason: the close review found the integration table claimed `HistorySource` scanned `queue-*`, while the implemented #75 prototype only uses draft/log sidecars as historical tag candidates.

Delta: revised the `HistorySource` integration row to list `draft-*.md` and `log-*.md` only. Queue badges remain future picker UI scope, as already noted in the `HistorySource` future extensions.

## Chunk 1: Pure Launcher Core

### Task 1: Parse `pair-go launch` Args

**Files:**
- Create: `cmd/internal/launcher/args.go`
- Create: `cmd/internal/launcher/args_test.go`

- [ ] **Step 1: Write failing parse tests**

Cover:
- no args: default agent `claude`;
- `<agent>`: custom agent;
- `<agent> -- <args>` and `-- <args>` forwarding;
- `resume <tag>` strips `pair-` and records forced tag;
- unexpected extra positional includes `unexpected positional arg` and `use '--' to forward args to the agent`;
- unsupported `continue`, `rename`, and `list` under `launch` return explicit prototype errors.

Run: `go test ./cmd/internal/launcher -run 'TestParseLaunchArgs' -count=1`
Expected: FAIL because package/files do not exist.

- [ ] **Step 2: Implement minimal parser**

Create `LaunchArgs`, `ParseArgs(args []string) (LaunchArgs, error)`, and a typed `UsageError`.

- [ ] **Step 3: Verify parse tests pass**

Run: `go test ./cmd/internal/launcher -run 'TestParseLaunchArgs' -count=1`
Expected: PASS.

### Task 2: Add Tag and Data-Dir Pure Helpers

**Files:**
- Create: `cmd/internal/launcher/tag.go`
- Create: `cmd/internal/launcher/tag_test.go`
- Create: `cmd/internal/launcher/datadir.go`
- Create: `cmd/internal/launcher/datadir_test.go`

- [ ] **Step 1: Write failing helper tests**

Cover:
- `NormalizeTag("pair-demo") == "demo"`;
- invalid characters and empty string return errors;
- `DefaultTag("/Users/xianxu/workspace/pair") == "pair"`;
- empty/symbol-only cwd basename falls back to `pair`;
- `ResolveDataDir(home, xdg)` returns `$XDG_DATA_HOME/pair` or `$HOME/.local/share/pair`.

Run: `go test ./cmd/internal/launcher -run 'TestNormalizeTag|TestDefaultTag|TestResolveDataDir' -count=1`
Expected: FAIL.

- [ ] **Step 2: Implement helpers**

Keep these functions pure; do not read environment variables directly inside them.

- [ ] **Step 3: Verify helper tests pass**

Run: `go test ./cmd/internal/launcher -run 'TestNormalizeTag|TestDefaultTag|TestResolveDataDir' -count=1`
Expected: PASS.

### Task 3: Model Sessions and Decisions

**Files:**
- Create: `cmd/internal/launcher/session.go`
- Create: `cmd/internal/launcher/decision.go`
- Create: `cmd/internal/launcher/decision_test.go`

- [ ] **Step 1: Write failing decision tests**

Cover:
- forced resume + blocking session -> attach `pair-<tag>`;
- forced resume + no blocking session -> create canonical bare tag `<tag>` with derived session name `pair-<tag>` and no prompt;
- no detached/no historical -> create next free tag and prompt;
- detached or historical present -> picker required;
- selected historical row -> create canonical bare tag with derived session name;
- exited rows do not block reuse;
- live and detached rows block reuse.

Run: `go test ./cmd/internal/launcher -run 'TestDecideLaunch' -count=1`
Expected: FAIL.

- [ ] **Step 2: Implement models**

Add small structs:

```go
type SessionState string

const (
    SessionAttached SessionState = "attached"
    SessionDetached SessionState = "detached"
    SessionExited   SessionState = "exited"
)

type LaunchAction string

const (
    ActionAttach LaunchAction = "attach"
    ActionCreate LaunchAction = "create"
    ActionPick   LaunchAction = "pick"
)
```

Keep `DecideLaunch(args LaunchArgs, snap SessionSnapshot) (LaunchDecision, error)` pure.

- [ ] **Step 3: Verify decision tests pass**

Run: `go test ./cmd/internal/launcher -run 'TestDecideLaunch' -count=1`
Expected: PASS.

- [ ] **Step 4: Commit pure core**

```bash
git add cmd/internal/launcher
git commit -m "#75: model Go launcher decisions" -m "Add pure launch argument, tag, data-dir, session, and decision models for the guarded Go launcher prototype." -m "Co-Authored-By: GPT-5 Codex <codex@openai.com>"
```

## Chunk 2: Thin IO Runner and Dispatcher Route

### Task 4: Add Fakeable Zellij and History Sources

**Files:**
- Create: `cmd/internal/launcher/zellij.go`
- Create: `cmd/internal/launcher/history.go`
- Create: `cmd/internal/launcher/run.go`
- Create: `cmd/internal/launcher/run_test.go`

- [ ] **Step 1: Write failing runner tests with in-memory fakes**

Cover:
- runner uses supplied environment/cwd/data-dir fields, not global process state;
- fake zellij rows become `SessionSnapshot` rows;
- fake historical files become historical candidates;
- runner returns a domain outcome that identifies help, parse failure, or a valid decision that is intentionally unsupported after the decision phase.

Run: `go test ./cmd/internal/launcher -run 'TestRunLaunch' -count=1`
Expected: FAIL.

- [ ] **Step 2: Implement interfaces and runner**

Use interfaces:

```go
type SessionLister interface {
    ListSessions() ([]ZellijSession, error)
    ClientCount(session string) (int, error)
}

type HistoricalScanner interface {
    Scan(base string, cutoff time.Time) ([]HistoricalTag, error)
}
```

`Run(args []string, env Env, sessions SessionLister, history HistoricalScanner) (LaunchOutcome, error)` should return domain values only:
- help outcome for `launch help`;
- typed parse/usage errors for invalid args;
- decision outcome for valid create/attach/pick decisions.

The dispatcher route, not the launcher package, maps those outcomes to `dispatcher.Result`:
- exit `0` for help;
- exit `2` for parse errors;
- exit `3` for a valid decision that is intentionally unsupported after the decision phase.

- [ ] **Step 3: Verify runner tests pass**

Run: `go test ./cmd/internal/launcher -run 'TestRunLaunch' -count=1`
Expected: PASS.

### Task 5: Route `pair-go launch`

**Files:**
- Modify: `cmd/internal/dispatcher/dispatcher.go`
- Modify: `cmd/internal/dispatcher/dispatcher_test.go`
- Modify: `cmd/pair-go/main.go`
- Modify: `cmd/pair-go/main_test.go`

- [ ] **Step 1: Update dispatcher tests**

Change the existing planned-command test so `wrap` still reports planned/unsupported, but `launch` routes to the implemented prototype and no longer says "planned but not implemented".

Run: `go test ./cmd/internal/dispatcher -run 'TestDispatch' -count=1`
Expected: FAIL until route is wired.

- [ ] **Step 2: Implement the route**

Add a dispatcher branch for `launch` that delegates to the launcher package. Keep all other planned families unchanged. The dispatcher route constructs `dispatcher.Result` from launcher domain values; the launcher package must not define a parallel stdout/stderr/exit-code result type. For production, the dispatcher route uses a small launcher runtime constructor that reads environment/cwd and creates the real zellij/history sources. For tests, expose a package-private or exported test seam that accepts an explicit launcher runtime.

- [ ] **Step 3: Update `pair-go` tests**

Assert:
- `pair-go help` still lists launch as the guarded prototype;
- `pair-go launch --help` prints launch usage;
- `pair-go launch resume demo` returns a prototype decision message, not a real launch.

Run: `go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
Expected: PASS.

- [ ] **Step 4: Commit IO and route**

```bash
git add cmd/internal/launcher cmd/internal/dispatcher cmd/pair-go
git commit -m "#75: add guarded Go launcher prototype" -m "Route pair-go launch through a fakeable launcher runner that reaches create/attach decisions without replacing bin/pair." -m "Co-Authored-By: GPT-5 Codex <codex@openai.com>"
```

### Task 6: Add Process-Level Fake Test

**Files:**
- Create: `cmd/pair-go/launch_process_test.go`
- Modify: `cmd/pair-go/main.go`

- [ ] **Step 1: Write failing process-level fake test**

The test should:
- create a temp `PATH` with fake `zellij`;
- set temp `HOME`/`XDG_DATA_HOME`;
- create draft/log sidecars for a historical tag;
- invoke the real routed path with `run([]string{"launch", ...}, stdout, stderr)`;
- assert the output names the prototype decision and never invokes real zellij attach/new-session.

Fake `zellij` contract:
- `zellij list-sessions --short` prints newline-separated session names such as `pair-live` and `pair-detached`;
- `zellij list-sessions --no-formatting` prints rows where exited sessions include `EXITED`;
- `zellij --session <name> action list-clients` prints a header plus one client row for attached sessions, and only the header for detached sessions;
- any `attach`, `--new-session-with-layout`, `new-session`, or `delete-session` invocation appends to a log and exits with a test failure marker so the assertion proves the prototype did not launch or mutate zellij state.

Run: `go test ./cmd/pair-go -run 'TestRunLaunchWithFakeZellij' -count=1`
Expected: FAIL until the dispatcher/cmd path has an injectable launcher runtime seam.

- [ ] **Step 2: Make the dispatcher/cmd path support injected launcher runtime for tests**

Keep production `main()` simple. It may call a small `runWithRuntime(args, stdout, stderr, runtime)` helper if needed, but `cmd/pair-go` must still contain no launcher business logic. Environment reads and command lookup belong to the launcher IO constructor or dispatcher route, not the pure launcher core.

- [ ] **Step 3: Verify process test passes**

Run: `go test ./cmd/pair-go -run 'TestRunLaunchWithFakeZellij' -count=1`
Expected: PASS.

## Chunk 3: Documentation and Verification

### Task 7: Document Remaining Shell-Owned Behavior

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `atlas/go-migration-inventory.md`

- [ ] **Step 1: Update atlas**

Record that `pair-go launch` now owns only the decision-phase prototype. Explicitly list shell-owned behavior still out of scope: real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller.

- [ ] **Step 2: Verify docs mention the boundary**

Run: `rg -n "pair-go launch|decision-phase|shell-owned" atlas/architecture.md atlas/go-migration-inventory.md`
Expected: matches in both files.

- [ ] **Step 3: Commit docs**

```bash
git add atlas/architecture.md atlas/go-migration-inventory.md
git commit -m "#75: document launcher prototype boundary" -m "Clarify that pair-go launch is a guarded decision-phase prototype while bin/pair remains the public launcher." -m "Co-Authored-By: GPT-5 Codex <codex@openai.com>"
```

### Task 8: Final Verification

**Files:**
- No planned edits.

- [ ] **Step 1: Run focused Go tests**

Run: `go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
Expected: PASS.

- [ ] **Step 2: Build `pair-go`**

Run: `make -B pair-go`
Expected: builds `bin/pair-go`.

- [ ] **Step 3: Run full Go suite**

Run: `go test ./... -count=1`
Expected: PASS.

- [ ] **Step 4: Verify public launcher unchanged**

Run: `git diff -- bin/pair`
Expected: empty output.

- [ ] **Step 5: Verify docs and whitespace**

Run: `rg -n "pair-go launch|decision-phase|shell-owned" atlas/architecture.md atlas/go-migration-inventory.md && git diff --check`
Expected: atlas matches and no whitespace errors.

- [ ] **Step 6: Close through SDLC**

Run: `sdlc close --issue 75 --verified 'go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; git diff -- bin/pair empty; rg atlas boundary check; git diff --check'`
Expected: close gate runs its mandatory review and records the verdict.
