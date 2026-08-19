# Delayed Watcher Start PID Generation Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bind the session watcher to the new agent PID even when the detached watcher starts after that PID file is written.

**Architecture:** Both native watcher producers capture one generation lower bound immediately before spawning `pair session-watch` and call one serializer owned by `sessionwatch`. The watcher CLI parses the bound into `Options`; the injected `Run` core compares generation-bound mtimes precisely, while legacy/direct invocations retain watcher-start fallback and historical same-second tolerance.

**Tech Stack:** Go, injected launcher/sessionwatch runtimes, table-driven and stateful-fake tests.

---

## Chunk 1: Generation-bound PID discovery

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Options.PIDNotBefore` | `cmd/internal/sessionwatch/run.go` | modified |
| `sessionwatch.CommandArgs` | `cmd/internal/sessionwatch/runcli.go` | new |
| `pidFileCurrent` | `cmd/internal/sessionwatch/run.go` | new |

- **`Options.PIDNotBefore`** — lower bound authorizing the PID-file generation for this watcher.
  - **Relationships:** One bound belongs to one watcher launch; `Run` compares every observed PID-file mtime against it.
  - **DRY rationale:** Reuses `freshPID` as the single freshness predicate instead of adding a second PID-selection path (ARCH-DRY).
  - **Future extensions:** Other sidecars can carry generation bounds through their own argv without widening this session identity contract.
- **`sessionwatch.CommandArgs`** — pure serializer for the internal watcher process contract.
  - **Relationships:** One launcher spawn produces one argv; `buildOptions` is its inverse consumer.
  - **DRY rationale:** Whole-workbench and agent-only restart reuse one command shape rather than maintaining parallel argv literals (ARCH-DRY, ARCH-PURPOSE).
  - **Future extensions:** New watcher-only metadata belongs before the `--` agent-argument delimiter.
- **`pidFileCurrent`** — pure timestamp policy selecting precise native-generation comparison or legacy same-second comparison.
  - **Relationships:** `Run` calls it for both PID wait-loop and final bind; one policy owns all freshness decisions.
  - **DRY rationale:** Prevents separate native/legacy scan loops while making their intentional precision difference explicit (ARCH-PURE).
  - **Future extensions:** Remove legacy tolerance when old direct invocations are retired.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `OSRuntime.SpawnSessionWatcher` | `cmd/internal/launcher/osruntime.go` | modified | wall clock and detached process spawn |
| `freshAgentInvocation` | `cmd/internal/wrapcmd/wrap.go` | modified | Shift+Alt+N replacement process spawn |
| `buildOptions` | `cmd/internal/sessionwatch/runcli.go` | modified | internal process argv/environment |
| `sessionwatch.Runtime` fake | `cmd/internal/sessionwatch/run_test.go` | reused | PID file/process lifecycle/filesystem state |
| Session-watch shell fixture | `tests/pair-session-watch-test.sh` | modified | real CLI parsing and filesystem mtimes |

- **`OSRuntime.SpawnSessionWatcher`** — captures `time.Now()` at the IO boundary and passes it to the pure argv serializer.
  - **Injected into:** `RunLaunch` through the existing `ProcOps` seam; no clock call enters create-flow policy (ARCH-PURE).
  - **Future extensions:** An injected OS clock is unnecessary unless launcher spawn timing itself needs deterministic integration testing.
- **`freshAgentInvocation`** — captures the agent-only restart generation before watcher spawn and replacement-wrapper exec.
  - **Injected into:** Shift+Alt+N's existing `freshExecRequest`; tests pass a fixed time into the request builder. Watchability derives from `sessionwatch.SpecForAgent`, the existing Codex/Agy/Muse registry; Claude remains synchronous.
  - **Future extensions:** Other agent-only sidecars must reuse the same generation value for one replacement attempt.
- **`buildOptions`** — parses the internal `--pid-not-before` timestamp before the `--` delimiter and fails closed on malformed values.
  - **Injected into:** `RunCLI`, which already owns watcher CLI normalization.
  - **Future extensions:** Keep internal watcher flags typed here rather than leaking parsing into `Run`.
- **`sessionwatch.Runtime` fake** — models watcher start time, PID-file mtime, process liveness, and discovered rollout state across polls (ARCH-MOCK).
  - **Injected into:** `Run` tests.
  - **Future extensions:** None for this fix; the existing state model already covers the race.
- **Session-watch shell fixture** — executes the real self-dispatched watcher against temporary files and a stateful `lsof` fake.
  - **Injected into:** Repository integration tests; the fake persists process/path behavior across real CLI calls.
  - **Future extensions:** Add real-agent conformance only if platform mtime behavior diverges from the temporary filesystem.

### Task 1: Pin the launcher-to-watcher process contract

**Files:**
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/osruntime_test.go`
- Modify: `cmd/internal/sessionwatch/runcli.go`
- Modify: `cmd/internal/wrapcmd/wrap.go`
- Modify: `cmd/internal/wrapcmd/agent_restart_test.go`

- [ ] **Step 1: Write the failing argv test**

Add failing producer tests with a fixed `time.Date(...)`: pin the launcher shape below, and table-test Shift+Alt+N so Codex, Agy, and Muse produce the same generation-bound shape while Claude produces no watcher. Watchability must come from `sessionwatch.SpecForAgent`, not a second agent list.

```go
[]string{exe, "session-watch", "codex", "work", "/cwd/sub",
    "--pid-not-before", "2026-08-19T08:47:30.123456789-07:00",
    "--repo-root", "/cwd", "--repo-name", "pair", "--", "--no-alt-screen"}
```

- [ ] **Step 2: Run the launcher test and verify RED**

Run: `go test ./cmd/internal/launcher -run TestSidecarSpawnArgvSelfExecsPair -count=1`

Expected: compile failure because `sessionwatch.CommandArgs` does not exist and `freshAgentInvocation` has no generation input.

- [ ] **Step 3: Implement the minimal serializer and OS capture**

Export `sessionwatch.CommandArgs`, format its `time.Time` with `time.RFC3339Nano`, and replace both the launcher helper and wrapcmd's literal argv. Have `OSRuntime.SpawnSessionWatcher` pass `time.Now()` immediately before `spawnDetached`; pass a fixed generation into `freshAgentInvocation` tests and `time.Now()` from `mustFreshExecRequest`. Replace wrapcmd's Codex/Agy condition with `sessionwatch.SpecForAgent(agent, home)`. Keep `ProcOps` unchanged: clock capture belongs inside each process integration seam.

- [ ] **Step 4: Run the launcher test and verify GREEN**

Run: `go test ./cmd/internal/launcher ./cmd/internal/wrapcmd -run 'TestSidecarSpawnArgvSelfExecsPair|TestFreshAgentInvocation.*Watcher|TestRunLaunchCodexAltScreen' -count=1`

Expected: PASS; agent arguments remain after the `--` delimiter.

### Task 2: Reproduce and fix delayed watcher startup

**Files:**
- Modify: `cmd/internal/sessionwatch/run.go`
- Modify: `cmd/internal/sessionwatch/run_test.go`
- Modify: `cmd/internal/sessionwatch/runcli.go`
- Modify: `cmd/internal/sessionwatch/runcli_test.go`

- [ ] **Step 1: Write failing CLI and lifecycle tests**

Add a CLI test proving `--pid-not-before <RFC3339Nano>` parses before repo flags without consuming agent args. Add a `Run` regression where the launcher bound is `08:47:30`, the PID file is written at `08:47:31`, and watcher `Now()` begins at `08:47:32`; the live PID exposes a valid root rollout and must produce the new config. Add the negative twin with a PID mtime one nanosecond before the bound in the same second; it must remain rejected even while live. Retain `TestRunTreatsSameSecondPidfileAsFresh` for the zero-bound legacy path.

- [ ] **Step 2: Run both regressions and verify RED**

Run: `go test ./cmd/internal/sessionwatch -run 'TestBuildOptions.*PIDNotBefore|TestRun.*LauncherGeneration' -count=1`

Expected: compile/assertion failure because `PIDNotBefore` and the internal flag do not exist.

- [ ] **Step 3: Implement typed parsing and one freshness threshold**

Add `PIDNotBefore time.Time` to `Options`. Parse the internal flag with `time.Parse(time.RFC3339Nano, value)`; malformed/missing values after the flag return `ok=false`. Replace `freshPID` with one policy helper used by both call sites: nonzero `PIDNotBefore` requires `!mod.Before(bound)` at full precision, while a zero bound falls back to `mod.Unix() >= watchStart.Unix()` for legacy compatibility.

- [ ] **Step 4: Run sessionwatch tests and verify GREEN**

Run: `go test ./cmd/internal/sessionwatch -count=1 -timeout=15s`

Expected: PASS, including #143 delayed lifecycle cases and #144 root/subagent classification cases.

### Task 3: Verify the process boundary and document the corrected clock

**Files:**
- Modify if stale: `atlas/architecture.md`
- Modify if stale: `atlas/how-to-bring-up-a-new-harness-cli.md`
- Modify: `tests/pair-session-watch-test.sh`
- Modify: `workshop/issues/000143-keep-agent-session-discovery-alive-after-startup-timeout.md`

- [ ] **Step 1: Run shadow searches and update only stale descriptions**

Run: `rg -n 'fresh PID|watchStart|watcher start|PID file|session-watch' atlas cmd/internal/sessionwatch cmd/internal/launcher`

Expected: operational prose describes PID freshness relative to launcher generation for native launches and watcher start only for legacy/direct invocation (ARCH-PURPOSE).

- [ ] **Step 2: Add and run the real CLI/filesystem regression**

Extend `tests/pair-session-watch-test.sh` with a generation-bound case that writes a live PID file after a fixed RFC3339Nano bound but starts `pair session-watch` only afterward; assert the config is written. Add a stale case whose PID mtime is before the bound and assert no config is written. This covers serializer/parser/filesystem semantics through the production process boundary (ARCH-MOCK).

Run: `env -u PAIR_TAG -u PAIR_AGENT -u PAIR_SESSION_ID -u PAIR_DATA_DIR bash tests/pair-session-watch-test.sh`

Expected: PASS for legacy stale-replacement, delayed-start acceptance, and stale-generation rejection.

- [ ] **Step 3: Run focused verification**

Run: `go test ./cmd/internal/launcher ./cmd/internal/sessionwatch -count=1 -timeout=30s`

Expected: PASS.

- [ ] **Step 4: Run repository verification**

Run: `env -u PAIR_TAG -u PAIR_AGENT -u PAIR_SESSION_ID -u PAIR_DATA_DIR go test ./... -count=1`

Run: `env -u PAIR_TAG -u PAIR_AGENT -u PAIR_SESSION_ID -u PAIR_DATA_DIR make test`

Run: `git diff --check`

Expected: all commands exit 0 with no failures or whitespace errors.

- [ ] **Step 5: Record implementation evidence**

Tick the two reopened issue-plan rows, append RED/GREEN and live-race evidence to `## Log`, and commit with the repository convention and `Co-Authored-By` trailer.

## Revisions

### 2026-08-19 09:00 PDT — Reopened after live scheduler race

- Replaces watcher-process-start freshness for native launches with a launcher-generation lower bound.
- Keeps the original lifecycle implementation and legacy fallback intact; adds no new session identity classifier or readiness protocol.

### 2026-08-19 09:05 PDT — Fresh-context plan review

- Added both watcher producers and centralized their command serialization in `sessionwatch`.
- Split precise native-generation comparison from legacy whole-second tolerance and required a same-second stale regression.
- Added a real CLI/filesystem boundary test matching the observed write-before-watcher-start ordering.

### 2026-08-19 09:10 PDT — Async-agent shadow sweep

- Added Muse to Shift+Alt+N by deriving watchability from `sessionwatch.SpecForAgent`; table coverage pins Codex/Agy/Muse and excludes Claude.
