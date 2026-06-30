# Go Entrypoint Switch Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `pair-go launch ...` a meaningful Go-owned entrypoint that runs the existing `pair` launcher with compatible arguments while leaving `pair` and `pair-dev` stable.

**Architecture:** Keep the shell launcher as the behavioral source of truth for this migration window (`ARCH-DRY`, `ARCH-PURPOSE`). Add a small pure decision layer that resolves the sibling launcher path and argv, plus a thin process boundary that execs it (`ARCH-PURE`). The existing dispatcher remains the owner for `pair-go help`, `pair-go context`, and `pair-go scrollback-render`.

**Tech Stack:** Go 1.x, `os.Executable`, `syscall.Exec` or an injected process runner for tests, existing Bash launcher `bin/pair`, existing Makefile build target for `cmd/pair-go`.

---

## Chunk 1: Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `LegacyLaunchRequest` | `cmd/internal/entrypoint/launch.go` | new |
| `ResolveLegacyLaunch` | `cmd/internal/entrypoint/launch.go` | new |

**LegacyLaunchRequest** — the process-independent request to invoke the current shell launcher.

- **Relationships:** 1:1 with a `pair-go launch ...` invocation; owns the resolved launcher path and argv passed to `bin/pair`.
- **DRY rationale:** Centralizes the compatibility mapping from `pair-go launch <pair-args>` to `pair <pair-args>` so tests and the process boundary do not each reconstruct argv handling.
- **Future extensions:** This is the place to widen from shell handoff to native Go launch when #78/#79 remove enough shell-owned behavior.

**ResolveLegacyLaunch** — pure function that converts the running executable path plus `pair-go launch` args into a `LegacyLaunchRequest`.

- **Relationships:** Used by `cmd/pair-go` before the IO boundary; independent of zellij, fzf, or the filesystem except for caller-provided existence checks.
- **DRY rationale:** Reuses the existing `bin/pair` launcher as the only real session lifecycle owner instead of duplicating create/attach/resume/list/rename behavior in Go.
- **Future extensions:** Can accept a `PAIR_GO_LAUNCH_NATIVE=1` mode or replacement implementation later without changing top-level command parsing.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `LegacyLauncherRunner` | `cmd/pair-go/main.go` | new | `os.Executable`, `os.Stat`, `syscall.Exec` |
| `pair-go launch` docs | `README.md`, `atlas/architecture.md`, `atlas/go-migration-inventory.md` | modified | operator-facing migration contract |

**LegacyLauncherRunner** — the thin IO shell that resolves the current binary path, validates sibling `pair`, and replaces the current process with it.

- **Injected into:** `runWithLegacyRunner` in tests, so process behavior is asserted with fakes rather than actually starting zellij.
- **Future extensions:** Native launch can replace only this runner once the pure launcher core owns side effects.

**pair-go launch docs** — documentation that explains the current migration boundary.

- **Injected into:** README command usage and atlas migration notes.
- **Future extensions:** #78/#79 can update the same docs when stateful shell glue or packaging changes.

## Chunk 2: Spec And Handoff Tests

### Task 1: Add pure launch request tests

**Files:**
- Create: `cmd/internal/entrypoint/launch_test.go`
- Create: `cmd/internal/entrypoint/launch.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestResolveLegacyLaunchDropsLaunchVerb(t *testing.T) {
	req := entrypoint.ResolveLegacyLaunch("/repo/bin/pair-go", []string{"claude", "--", "--resume"})
	if req.Path != "/repo/bin/pair" {
		t.Fatalf("Path = %q", req.Path)
	}
	want := []string{"pair", "claude", "--", "--resume"}
	if !reflect.DeepEqual(req.Argv, want) {
		t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
	}
}

func TestResolveLegacyLaunchPreservesSubcommands(t *testing.T) {
	req := entrypoint.ResolveLegacyLaunch("/repo/bin/pair-go", []string{"resume", "demo"})
	want := []string{"pair", "resume", "demo"}
	if !reflect.DeepEqual(req.Argv, want) {
		t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
	}
}
```

- [ ] **Step 2: Run the tests to verify RED**

Run: `go test ./cmd/internal/entrypoint -run TestResolveLegacyLaunch -count=1`

Expected: FAIL because `cmd/internal/entrypoint` does not exist yet.

- [ ] **Step 3: Implement minimal pure resolver**

Create `LegacyLaunchRequest` with `Path string` and `Argv []string`. Implement `ResolveLegacyLaunch(executable string, launchArgs []string)` using `filepath.Dir(executable)` and sibling `pair`. The argv must start with `"pair"` and append all launch args unchanged.

- [ ] **Step 4: Run tests to verify GREEN**

Run: `go test ./cmd/internal/entrypoint -run TestResolveLegacyLaunch -count=1`

Expected: PASS.

### Task 2: Add `pair-go launch` process-boundary tests

**Files:**
- Modify: `cmd/pair-go/main.go`
- Modify: `cmd/pair-go/main_test.go`
- Modify: `cmd/pair-go/launch_process_test.go`
- Modify: `cmd/internal/dispatcher/dispatcher_test.go`

- [ ] **Step 1: Write failing main tests for handoff**

Add a fake runner test that calls the test seam with args `[]string{"launch", "claude", "--", "--resume"}` and asserts:

- exit code is the fake runner's code;
- no dispatcher prototype text is printed;
- runner receives sibling `bin/pair`;
- runner receives argv `["pair", "claude", "--", "--resume"]`.
- runner receives inherited environment entries, including a sentinel such as `PAIR_TEST_ENV=kept`, so the shell launcher sees the same env that `pair-go` received (`ARCH-PURPOSE` compatibility surface).

Also add a missing-launcher test that returns a not-found stat result and asserts stderr mentions `pair-go launch`, `bin/pair`, `make build`, `make install`, and `dev-aliases.sh`.

- [ ] **Step 2: Run targeted tests to verify RED**

Run: `go test ./cmd/pair-go -run 'TestRunLaunch' -count=1`

Expected: FAIL because current `pair-go launch` returns prototype exit code 3 and never invokes the fake runner.

- [ ] **Step 3: Update main seam**

Refactor `cmd/pair-go/main.go` so `run` delegates to `runWithRuntime`. For `args[0] == "launch"`, use the injected `LegacyLauncherRunner` and `entrypoint.ResolveLegacyLaunch`; otherwise preserve existing dispatcher behavior. The real runner should:

- get `os.Executable()`;
- resolve sibling `pair`;
- `os.Stat` it and require a non-directory executable path;
- call `syscall.Exec(path, argv, os.Environ())`.

In tests, fake these methods and return deterministic codes.

- [ ] **Step 4: Update stale prototype expectations**

Change launch tests that currently expect `"prototype decision"` and exit code `3` to expect legacy handoff behavior or delete them if covered by the new fake-runner tests. Keep dispatcher tests for `context`, `scrollback-render`, help, version, and planned command errors.

- [ ] **Step 5: Run targeted tests to verify GREEN**

Run: `go test ./cmd/pair-go ./cmd/internal/dispatcher ./cmd/internal/entrypoint -count=1`

Expected: PASS.

## Chunk 3: Build Wiring And Documentation

### Task 3: Keep Makefile wiring accurate

**Files:**
- Modify: `Makefile.local`

- [ ] **Step 1: Write or update dependency expectation**

Inspect the `$(BIN_DIR)/pair-go` dependency list and add `cmd/internal/entrypoint/launch.go` so `make pair-go` rebuilds when the new resolver changes.

- [ ] **Step 2: Run build**

Run: `make pair-go`

Expected: `bin/pair-go` builds successfully.

### Task 4: Document the migration boundary

**Files:**
- Modify: `README.md`
- Modify: `atlas/architecture.md`
- Modify: `atlas/go-migration-inventory.md`
- Modify: `workshop/issues/000077-go-entrypoint-switch.md`

- [ ] **Step 1: Update README**

Add a short development note near Command Usage:

```markdown
`pair-go launch ...` is the Go-owned migration entrypoint for testing the launcher path. It accepts the same arguments after `launch` that `pair` accepts directly, then hands off to the current `pair` launcher for one migration window. In a dev shell sourced from `../ariadne/construct/dev-aliases.sh`, `pair-go` rebuilds from `cmd/pair-go` automatically before running; no `pair-go-dev` command is needed.
```

- [ ] **Step 2: Update atlas**

Update `atlas/architecture.md` and `atlas/go-migration-inventory.md` so they no longer describe `pair-go launch` as decision-phase only. State that #77 makes it a Go-owned compatibility handoff to `bin/pair`, while `bin/pair` remains the public stable entrypoint and the real zellij lifecycle remains shell-owned.

- [ ] **Step 3: Tick issue plan items and log verification intent**

In #77, tick completed plan rows as they land and add a log entry with the exact commands run.

## Chunk 4: Verification And Close

### Task 5: Full verification

**Files:**
- No new code files.

- [ ] **Step 1: Run focused Go tests**

Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1`

Expected: PASS.

- [ ] **Step 2: Run full Go suite**

Run: `go test ./... -count=1`

Expected: PASS.

- [ ] **Step 3: Run build**

Run: `make pair-go`

Expected: PASS and `bin/pair-go` exists.

- [ ] **Step 4: Smoke test non-zellij routes**

Run:

```bash
make test-dev-rebuild
bin/pair-go help
bin/pair-go launch --help
bin/pair --help
bin/pair-dev --help
```

Expected: `make test-dev-rebuild` passes, proving the existing `PAIR_DEV` rebuild hook still works. Help output succeeds. `bin/pair-go launch --help` should print the existing `pair` help because it hands off to `bin/pair --help`.

- [ ] **Step 5: SDLC close**

Run: `sdlc close --issue 77 --verified '<focused tests, full go test, build, and help smoke evidence>'`.

Expected: SDLC close runs its review gate; fix any Critical/Important findings before merge.
