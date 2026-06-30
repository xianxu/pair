# Go Helper Dispatch Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `pair-go context` and `pair-go scrollback-render` routes that reuse the existing helper implementations while preserving the legacy helper binaries.

**Architecture:** Extract reusable helper runners from the selected `package main` commands and route the dispatcher through those runners instead of reimplementing command behavior (`ARCH-DRY`). Keep helper business logic pure or close to existing pure cores, with IO at thin command boundaries: runners accept argv/stdout/stderr/env-style inputs and return exit status or error, while `main()` remains a tiny process shell (`ARCH-PURE`). The slice intentionally proves the helper-dispatch pattern without moving live zellij/nvim call sites or changing public `pair` behavior (`ARCH-PURPOSE`).

**Tech Stack:** Go standard library, existing `cmd/internal/dispatcher`, existing `cmd/internal/ctxmeter` and `cmd/internal/transcript`, existing scrollback renderer core, `go test`, `make`.

---

## Core Concepts

### Command Surfaces

| Name | Kind | Lives in | Status |
|------|------|----------|--------|
| `contextcmd.Env` | INTEGRATION input value | `cmd/internal/contextcmd/contextcmd.go` | new |
| `contextcmd.Run` | INTEGRATION | `cmd/internal/contextcmd/contextcmd.go` | new |
| `scrollbackcmd.Run` | INTEGRATION | `cmd/internal/scrollbackcmd/scrollbackcmd.go` | new |
| `dispatcher.Dispatch` | INTEGRATION | `cmd/internal/dispatcher/dispatcher.go` | modified |

**contextcmd.Env** — Injected environment values for the context helper: home, XDG data home, and explicit Pair data dir.
- **Relationships:** Passed to `contextcmd.Run`; populated from `EnvFromOS` only at the process boundary.
- **DRY rationale:** Both legacy binary and dispatcher route need the same env/default resolution without re-reading OS env in tests.
- **Future extensions:** Can widen if more commands need shared Pair environment resolution.

**contextcmd.Run** — Shared command runner for `pair-context <tag> <agent>` behavior. It remains tolerant: missing config/transcript/input prints nothing and exits 0.
- **Injected into:** legacy `cmd/pair-context/main.go` and dispatcher context route.
- **Future extensions:** The title poller can continue using the legacy binary until #77/#78 moves call sites.

**scrollbackcmd.Run** — Shared command runner for `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out`.
- **Injected into:** legacy `cmd/pair-scrollback-render/main.go` and dispatcher scrollback-render route.
- **Future extensions:** `bin/pair-scrollback-open`, `bin/pair-changelog-open`, and `nvim/scrollback.lua` can move to the dispatcher after the public entrypoint is Go-owned.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ContextRunner` (`contextcmd.Run`) | `cmd/internal/contextcmd/contextcmd.go` | new | filesystem, env, transcript files, stdout |
| `ScrollbackRenderRunner` (`scrollbackcmd.Run`) | `cmd/internal/scrollbackcmd/scrollbackcmd.go` | new | filesystem, flag parsing, stdout/stderr |
| `dispatcher.Dispatch` | `cmd/internal/dispatcher/dispatcher.go` | modified | in-process calls to selected helper runners |

**ContextRunner** — Shared command runner for `pair-context <tag> <agent>` behavior. It remains tolerant: missing config/transcript/input prints nothing and exits 0.
- **Injected into:** legacy `cmd/pair-context/main.go` and dispatcher context route.
- **Future extensions:** The title poller can continue using the legacy binary until #77/#78 moves call sites.

**ScrollbackRenderRunner** — Shared command runner for `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out`.
- **Injected into:** legacy `cmd/pair-scrollback-render/main.go` and dispatcher scrollback-render route.
- **Future extensions:** `bin/pair-scrollback-open`, `bin/pair-changelog-open`, and `nvim/scrollback.lua` can move to the dispatcher after the public entrypoint is Go-owned.

**dispatcher.Dispatch** — Dispatcher routes for `context` and `scrollback-render`.
- **Injected into:** `cmd/pair-go`.
- **Future extensions:** Later helper routes should follow the same runner extraction pattern, not duplicate command logic.

## Revisions

- 2026-06-30T12:24:41-0700 — Boundary review found the Core Concepts table still named planned parsed-args structs that the implementation did not need. Replaced `ContextRunArgs`/`ScrollbackRenderArgs` with the shipped `contextcmd.Env`, `contextcmd.Run`, and `scrollbackcmd.Run` surfaces, and classified the runners as INTEGRATION rather than PURE entities.

---

## Chunk 1: Extract Context Runner

### Task 1: Make `pair-context` Reusable

**Files:**
- Create: `cmd/internal/contextcmd/contextcmd.go`
- Create: `cmd/internal/contextcmd/contextcmd_test.go`
- Modify: `cmd/pair-context/main.go`
- Modify: `cmd/pair-context/main_test.go`
- Modify: `Makefile.local`

- [x] **Step 1: Add failing runner tests**

Create `cmd/internal/contextcmd/contextcmd_test.go` with tests that call `Run(args []string, env Env, stdout io.Writer) int` directly:

```go
func TestRunClaude(t *testing.T) {
    // Arrange the same config/pane/transcript fixture as TestPairContext_Claude.
    // Call Run([]string{"T", "claude"}, env, &stdout).
    // Assert code == 0 and stdout == "398k\n".
}

func TestRunMissingConfigPrintsNothing(t *testing.T) {
    // Call Run with empty data dir.
    // Assert code == 0 and stdout == "".
}
```

- [x] **Step 2: Run the focused tests and confirm they fail**

Run: `go test ./cmd/internal/contextcmd -count=1`

Expected: FAIL because the package does not exist yet.

- [x] **Step 3: Extract the runner**

Move the reusable context behavior into `cmd/internal/contextcmd`. Keep `cmd/pair-context/main.go` as:

```go
func main() {
    os.Exit(contextcmd.Run(os.Args[1:], contextcmd.EnvFromOS(), os.Stdout))
}
```

The runner must:
- return 0 for missing args, matching the current tolerant behavior;
- resolve `PAIR_DATA_DIR` from env or `$XDG_DATA_HOME/pair` or `$HOME/.local/share/pair`;
- write the same humanized token count to the injected stdout;
- never call `os.Exit`.

- [x] **Step 4: Run the focused tests and existing package tests**

Run: `go test ./cmd/internal/contextcmd ./cmd/pair-context -count=1`

Expected: PASS.

- [x] **Step 5: Commit the context extraction**

Run:

```bash
git add cmd/internal/contextcmd cmd/pair-context Makefile.local
git commit -m "#76: extract pair-context runner"
```

---

## Chunk 2: Extract Scrollback Renderer Runner

### Task 2: Make `pair-scrollback-render` Reusable

**Files:**
- Create: `cmd/internal/scrollbackcmd/scrollbackcmd.go`
- Create: `cmd/internal/scrollbackcmd/scrollbackcmd_test.go`
- Modify: `cmd/pair-scrollback-render/main.go`
- Modify or create tests in: `cmd/pair-scrollback-render/*_test.go`
- Modify: `Makefile.local`

- [x] **Step 1: Add failing runner tests**

Create tests that call `scrollbackcmd.Run(args []string, stdout, stderr io.Writer) int`:

```go
func TestRunUsage(t *testing.T) {
    var stderr bytes.Buffer
    code := Run([]string{}, io.Discard, &stderr)
    // Assert code == 2 and usage is written to stderr.
}

func TestRunWritesOutput(t *testing.T) {
    // Use a tiny raw/events fixture compatible with existing renderer tests.
    // Call Run([]string{raw, events, out}, io.Discard, &stderr).
    // Assert code == 0 and out exists.
}
```

- [x] **Step 2: Run the focused tests and confirm they fail**

Run: `go test ./cmd/internal/scrollbackcmd -count=1`

Expected: FAIL because the package does not exist yet.

- [x] **Step 3: Extract the runner**

Move the renderer command wrapper into `cmd/internal/scrollbackcmd`. If `render(...)` cannot be imported from `package main`, move the rendering core into this internal package too and leave the legacy command as a tiny wrapper:

```go
func main() {
    os.Exit(scrollbackcmd.Run(os.Args[1:], os.Stdout, os.Stderr))
}
```

Use a local `flag.FlagSet` so dispatcher and tests can parse independently. Preserve current behavior:
- usage to stderr and exit 2 for wrong arity;
- `scrollback-render: <err>` to stderr and exit 1 for render errors;
- exit 0 for success;
- same defaults for `--plain`, `--max-lines`, and `--with-timestamps`.

- [x] **Step 4: Run focused and package tests**

Run: `go test ./cmd/internal/scrollbackcmd ./cmd/pair-scrollback-render -count=1`

Expected: PASS.

- [x] **Step 5: Commit the renderer extraction**

Run:

```bash
git add cmd/internal/scrollbackcmd cmd/pair-scrollback-render Makefile.local
git commit -m "#76: extract scrollback renderer runner"
```

---

## Chunk 3: Wire Dispatcher Routes

### Task 3: Route Selected Helpers Through `pair-go`

**Files:**
- Modify: `cmd/internal/dispatcher/dispatcher.go`
- Modify: `cmd/internal/dispatcher/dispatcher_test.go`
- Modify: `cmd/pair-go/main_test.go`
- Create: `cmd/pair-go/helper_equivalence_test.go`
- Modify: `Makefile.local`

- [x] **Step 1: Add failing dispatcher tests**

Add tests for:
- `Dispatch([]string{"context", "T", "claude"})` through a fake or temp fixture returns the same output shape as `contextcmd.Run`;
- `Dispatch([]string{"scrollback-render"})` returns renderer usage with exit 2;
- top-level help lists `context` and `scrollback-render` as implemented helper routes, not planned-only commands.

- [x] **Step 2: Run dispatcher tests and confirm failure**

Run: `go test ./cmd/internal/dispatcher -run 'TestDispatch(Context|Scrollback|Help)' -count=1`

Expected: FAIL because routes are not implemented yet.

- [x] **Step 3: Add failing process-level equivalence test**

Create `cmd/pair-go/helper_equivalence_test.go` with a context helper fixture that builds both commands and runs them against the same temp Pair data/transcript tree:

```go
func TestPairGoContextMatchesLegacyPairContext(t *testing.T) {
    // Build ./cmd/pair-context and ./cmd/pair-go into a temp bin dir.
    // Arrange config-T-claude.json, pane-T-claude.json, and a Claude transcript.
    // Run: pair-context T claude
    // Run: pair-go context T claude
    // Assert stdout, stderr, and exit code match exactly.
}
```

This is the representative compatibility proof required by the issue done-when. It should fail before the dispatcher route exists because `pair-go context` is still planned-only.

- [x] **Step 4: Run the equivalence test and confirm failure**

Run: `go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1`

Expected: FAIL because `pair-go context` returns planned-but-not-implemented.

- [x] **Step 5: Implement routes**

Update `Families()` statuses for selected helpers and add dispatcher branches:

```go
case "context":
    return dispatchContext(args[1:])
case "scrollback-render":
    return dispatchScrollbackRender(args[1:])
```

The dispatcher should continue returning `dispatcher.Result`. Helper runners should write into buffers so dispatcher can map stdout/stderr/exit code without adding a second process-result abstraction.

- [x] **Step 6: Run route and process tests**

Run:

```bash
go test ./cmd/internal/dispatcher ./cmd/pair-go ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render -count=1
make pair-context pair-scrollback-render pair-go
```

Expected: PASS. The `make` command is deliberately not `-B`; it verifies the updated dependency graph can rebuild normally after source changes.

- [x] **Step 7: Commit dispatcher wiring**

Run:

```bash
git add cmd/internal/dispatcher cmd/pair-go cmd/internal/contextcmd cmd/internal/scrollbackcmd cmd/pair-context cmd/pair-scrollback-render Makefile.local
git commit -m "#76: route selected helpers through pair-go"
```

---

## Chunk 4: Verify Legacy Compatibility And Docs

### Task 4: Verify Builds, Callers, And Atlas

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `atlas/go-migration-inventory.md`
- Modify: `workshop/issues/000076-go-helper-dispatch.md`
- Modify: `Makefile.local`

- [x] **Step 1: Verify legacy binaries still build**

Run:

```bash
make pair-context pair-scrollback-render pair-go
make -B pair-context pair-scrollback-render pair-go
```

Expected: PASS. The non-`-B` run verifies incremental prerequisites include `cmd/internal/contextcmd`, `cmd/internal/scrollbackcmd`, and dispatcher dependencies; the `-B` run remains the forced clean rebuild check.

- [x] **Step 2: Verify selected command equivalence**

Run focused commands against test fixtures or package tests:

```bash
go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1
go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1
```

Expected: PASS; the equivalence test demonstrates the legacy `pair-context` binary and `pair-go context` process path produce identical stdout/stderr/exit code on the same fixture.

- [x] **Step 3: Verify full Go test suite**

Run: `go test ./... -count=1`

Expected: PASS.

- [x] **Step 4: Verify no live call sites moved**

Run:

```bash
git diff -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollback-open bin/pair-changelog-open
```

Expected: empty diff, unless a test-only or documentation-only change was explicitly made.

- [x] **Step 5: Update atlas**

Update:
- `atlas/architecture.md` to say `pair-go context` and `pair-go scrollback-render` are implemented helper routes while public launcher and live zellij/nvim callers remain legacy.
- `atlas/go-migration-inventory.md` rows for `pair-context` and `pair-scrollback-render` to record dispatcher availability and preserved legacy names.

- [x] **Step 6: Update issue checklist and log**

Tick the #76 plan/done items that are complete and add a log entry with verification commands and `ARCH-*` notes.

- [x] **Step 7: Run final verification before close**

Run:

```bash
go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1
go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1
make pair-context pair-scrollback-render pair-go
make -B pair-context pair-scrollback-render pair-go
go test ./... -count=1
git diff -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollback-open bin/pair-changelog-open
rg -n "pair-go context|pair-go scrollback-render|helper dispatch" atlas/architecture.md atlas/go-migration-inventory.md
git diff --check
```

Expected: all tests/builds pass, caller diff empty, atlas grep finds the new helper-dispatch documentation, and whitespace check passes.

- [ ] **Step 8: Close through SDLC**

Run:

```bash
sdlc close --issue 76 --verified 'go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1; go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1; make pair-context pair-scrollback-render pair-go; make -B pair-context pair-scrollback-render pair-go; go test ./... -count=1; git diff live callers empty; rg atlas helper dispatch; git diff --check'
```

Expected: close gate runs the boundary review and reports SHIP or actionable findings.
