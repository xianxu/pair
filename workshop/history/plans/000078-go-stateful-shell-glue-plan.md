# Go Stateful Shell Glue Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port `bin/pair-session-watch.sh` to a Go-owned command while keeping the legacy script name as a compatibility shim.

**Architecture:** Keep the stateful watcher behavior split into pure session-watch decisions and a thin process/filesystem shell. `cmd/internal/sessionwatch` will own agent support, PID-tree/session-file matching, id extraction, resume-arg stripping, and config payload construction; `cmd/pair-session-watch` will own real time, process commands, atomic writes, and adapt-log emission. `bin/pair-session-watch.sh` remains the stable caller surface and execs the built Go binary for this migration window.

**Tech Stack:** Go standard library, existing `cmd/internal/adapt.Open` / `adapt.Logger` for flight-recorder events, shell compatibility shim, process-level shell tests with fake `ps`/`lsof`/filesystem state.

---

## Scope

#78 ports only `pair-session-watch.sh`. `pair-title.sh` is explicitly deferred because it has a separate long-running UI ownership surface: zellij frame titles, cmux workspace title ownership, activity buckets, singleton poller identity, and session liveness. Keeping these as separate issues reduces review risk and keeps each migrated script meaningfully testable. `ARCH-PURPOSE`: this still satisfies #78 by porting a prioritized stateful shell-glue subset and splitting the rest.

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `AgentSpec` | `cmd/internal/sessionwatch/sessionwatch.go` | new |
| `SessionID` | `cmd/internal/sessionwatch/sessionwatch.go` | new |
| `StripResumeArgs` | `cmd/internal/sessionwatch/sessionwatch.go` | new |
| `ConfigPayload` | `cmd/internal/sessionwatch/sessionwatch.go` | new |

- **AgentSpec** — Per-agent watch metadata for `codex` and `agy`: watch directory suffix, filename pattern, and id extractor.
  - **Relationships:** 1:1 with supported agent names; owned by the watcher planner.
  - **DRY rationale:** The current shell duplicates agent conditionals across directory selection, path matching, and extraction. Centralizing the agent contract avoids drift as more harnesses are added.
  - **Future extensions:** Claude is still synchronous in `bin/pair`; if a future agent needs async discovery, add one spec row and tests.

- **SessionID** — Extracted identifier plus matched file path and near-miss state.
  - **Relationships:** 1:1 with a matched session file candidate.
  - **DRY rationale:** `lsof`, birth-time fallback, and legacy fallback all need the same extract-or-near-miss behavior.
  - **Future extensions:** Add richer confidence reasons if diagnostics need to distinguish lsof vs fallback discovery.

- **StripResumeArgs** — Agent args with resume bindings removed before persistence.
  - **Relationships:** N:1 from raw agent args to saved config args.
  - **DRY rationale:** Mirrors `bin/pair` stripping semantics in one Go function and tests edge cases that are awkward in shell.
  - **Future extensions:** If `bin/pair` becomes Go-owned, the same stripping function should become the single source for launcher and watcher.

- **ConfigPayload** — `{agent,args,session_id}` JSON payload for `config-<tag>-<agent>.json`.
  - **Relationships:** 1:1 with a discovered `(tag, agent)` restart config.
  - **DRY rationale:** Moves JSON construction out of shell and into Go's structured encoder, matching the `workshop/lessons.md` rule against printf JSON.
  - **Future extensions:** Add schema fields here if restart config widens.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Runtime` | `cmd/internal/sessionwatch/run.go` | new | injected process list, lsof, find/stat, sleep/clock, filesystem writes |
| `OSRuntime` | `cmd/internal/sessionwatch/runtime.go` | new | real process list, lsof, find/stat, sleep/clock, filesystem writes |
| `PairSessionWatchCommand` | `cmd/pair-session-watch/main.go` | new | CLI, environment, process loop |
| `PairSessionWatchShim` | `bin/pair-session-watch.sh` | modified | legacy shell command name |
| `SessionWatchProcessTest` | `tests/pair-session-watch-test.sh` | modified | fake PATH commands and temp HOME/data dirs |
| `AdaptLogger` | `cmd/internal/adapt/adapt.go` | reused | adaptation flight-recorder JSONL schema |

- **Runtime** — Boundary used by the command loop for process and filesystem side effects.
  - **Injected into:** `sessionwatch.Run` or equivalent orchestration function; pure helpers stay independent.
  - **Future extensions:** Lets tests drive timeout/failure cases without real 60s sleeps.

- **OSRuntime** — Real implementation of `Runtime` for production process and filesystem calls.
  - **Injected into:** `cmd/pair-session-watch/main.go` after env/argv resolution.
  - **Future extensions:** Platform-specific process and birth-time behavior stays isolated here.

- **PairSessionWatchCommand** — Parses `pair-session-watch <agent> <tag> <cwd> [agent-args...]`, no-ops unsupported agents, and runs the watcher.
  - **Injected into:** Called by the shell shim and later directly by `bin/pair` or a Go entrypoint.
  - **Future extensions:** Can become `pair-go session-watch` or internal launch subcommand when `bin/pair` is retired.

- **PairSessionWatchShim** — Keeps existing callers stable by execing `$PAIR_HOME/bin/pair-session-watch`.
  - **Injected into:** Existing `bin/pair` create path remains unchanged.
  - **Future extensions:** Delete once callers move to the Go command directly.

- **SessionWatchProcessTest** — Process-level regression coverage with fake commands and temp files.
  - **Injected into:** `make test-session-watch` and the repo-wide `make test` target.
  - **Future extensions:** Add agy fixtures as its native session format evolves.

- **AdaptLogger** — Existing Go logger for the shared adaptation flight-recorder schema.
  - **Injected into:** `cmd/internal/sessionwatch` runtime setup, not pure helper functions.
  - **Future extensions:** Keeps shell/Lua/Go emitters aligned as the remaining shell emitters move. `ARCH-DRY`: do not hand-maintain a second Go copy of the adapt JSON schema.

## Task 1: Pure Session Watch Decisions

**Files:**
- Create: `cmd/internal/sessionwatch/sessionwatch.go`
- Create: `cmd/internal/sessionwatch/sessionwatch_test.go`

- [x] **Step 1: Write failing tests for supported agent specs and id extraction**

Tests:
- `codex` accepts paths under `~/.codex/sessions/.../rollout-*-<uuid>.jsonl`.
- `agy` accepts paths under `~/.gemini/antigravity-cli/conversations/<uuid>.db`.
- filenames matching the watch pattern but not the id grammar return a near-miss.
- unsupported agents are no-op.

Run: `go test ./cmd/internal/sessionwatch -run 'TestAgentSpec|TestExtract' -count=1`
Expected: FAIL because the package does not exist.

- [x] **Step 2: Implement minimal `AgentSpec` and `ExtractSessionID`**

Use only deterministic string/path logic. Do not shell out or read files in this package function.

- [x] **Step 3: Verify pure extraction tests pass**

Run: `go test ./cmd/internal/sessionwatch -run 'TestAgentSpec|TestExtract' -count=1`
Expected: PASS.

- [x] **Step 4: Write failing tests for resume-arg stripping and config JSON**

Tests:
- codex leading `resume <id>` is removed.
- any `--resume <id>` pair is removed.
- unrelated args keep order.
- JSON payload escapes quotes and preserves arrays through `encoding/json`.

Run: `go test ./cmd/internal/sessionwatch -run 'TestStrip|TestConfig' -count=1`
Expected: FAIL until helpers exist.

- [x] **Step 5: Implement `StripResumeArgs` and `ConfigJSON`**

Keep behavior byte-compatible in structure with existing shell output: object with `agent`, `args`, and `session_id`.

- [x] **Step 6: Verify all pure tests pass**

Run: `go test ./cmd/internal/sessionwatch -count=1`
Expected: PASS.

## Task 2: Go Watcher Command

**Files:**
- Create: `cmd/internal/sessionwatch/runtime.go`
- Create: `cmd/internal/sessionwatch/run.go`
- Create: `cmd/internal/sessionwatch/run_test.go`
- Create: `cmd/pair-session-watch/main.go`
- Modify: `Makefile.local`

- [x] **Step 1: Write failing runtime tests for stale pidfile replacement**

Use a fake runtime:
- initial pidfile mtime predates watcher start and points at a dead/unrelated PID.
- fresh pidfile appears during the wait window.
- lsof on the fresh PID returns a codex rollout file.
- config is written atomically with the discovered id.

Run: `go test ./cmd/internal/sessionwatch -run TestRunUsesFreshPidfile -count=1`
Expected: FAIL because orchestration does not exist.

- [x] **Step 2: Implement watcher orchestration with injected runtime**

Keep the loop behavior faithful:
- return immediately for unsupported agents.
- wait briefly for a fresh `agent-pid-<tag>` file.
- if bound to a root PID, inspect root plus descendants with `lsof -Fn`.
- if lsof misses, use birth-time fallback for files born at or after pidfile mtime, accepting exactly one candidate.
- if no root PID, use legacy snapshot-diff fallback.
- write config via temp file plus rename.
- emit adapt-log `near-miss`, `fired`, and `fail` outcomes.

- [x] **Step 3: Verify runtime stale-pidfile test passes**

Run: `go test ./cmd/internal/sessionwatch -run TestRunUsesFreshPidfile -count=1`
Expected: PASS.

- [x] **Step 4: Add failing tests for near-miss, fail, and agy discovery**

Use fake runtime with a controllable clock so the fail case does not sleep 60s.

Run: `go test ./cmd/internal/sessionwatch -run 'TestRunLogs|TestRunAgy' -count=1`
Expected: FAIL until diagnostics/fallbacks are complete.

- [x] **Step 5: Finish orchestration and CLI command**

Create `cmd/pair-session-watch/main.go` as a thin CLI over the runtime. Update `Makefile.local` explicitly:
- add `pair-session-watch` to `.PHONY`;
- add `pair-session-watch` to `GO_BINS`;
- add a per-binary `pair-session-watch: $(BIN_DIR)/pair-session-watch` alias;
- add a `$(BIN_DIR)/pair-session-watch` build rule;
- make `test-session-watch` depend on `$(BIN_DIR)/pair-session-watch` so repo-wide `make test` cannot run the shim process test before the Go binary exists.

- [x] **Step 6: Verify command package tests pass**

Run: `go test ./cmd/internal/sessionwatch ./cmd/pair-session-watch -count=1`
Expected: PASS.

## Task 3: Compatibility Shim And Process Tests

**Files:**
- Modify: `bin/pair-session-watch.sh`
- Modify: `tests/pair-session-watch-test.sh`
- Modify: `Makefile.local`

- [x] **Step 1: Replace shell implementation with a compatibility shim**

The shim should:
- resolve its real path like other Pair scripts;
- set `PAIR_HOME`;
- exec `$PAIR_HOME/bin/pair-session-watch "$@"`;
- print a clear diagnostic if the Go binary is missing.

- [x] **Step 2: Expand process-level test coverage**

Update `tests/pair-session-watch-test.sh` to exercise the shim invoking the Go binary with fake `ps`/`lsof` and temp HOME/data dirs. Keep the stale pidfile regression. Add a quoted arg in the saved config to prove JSON escaping is structured.

Run: `make pair-session-watch && make test-session-watch`
Expected: PASS.

- [x] **Step 3: Verify direct command and shim both work**

Run:
- `bin/pair-session-watch --help` or unsupported-agent smoke if no help is exposed.
- `go test ./... -count=1`
- `make test-session-watch`

Expected: all PASS.

## Task 4: Docs, Issue Split, And Verification

**Files:**
- Modify: `workshop/issues/000078-go-stateful-shell-glue.md`
- Modify: `atlas/go-migration-inventory.md`
- Modify: `atlas/architecture.md`
- Optionally create: follow-up issue for `pair-title.sh` if #78 does not already leave enough trace.

- [x] **Step 1: Update atlas**

Record that `pair-session-watch` is now Go-owned with a shell shim, while `pair-title.sh` remains stateful shell glue.

- [x] **Step 2: Update #78 issue**

Check off candidate selection and implementation items that are complete. Log the explicit split: `pair-title.sh` remains a follow-up because it owns UI title state rather than restart config discovery.

Also log that short shell scripts and opener scripts remain intentionally shell-owned in this slice because #78's payoff target is stateful session discovery. This directly satisfies the Done-when item about leaving no-payoff shell glue alone.

- [x] **Step 3: Run final verification**

Run:
- `go test ./cmd/internal/sessionwatch ./cmd/pair-session-watch -count=1`
- `go test ./... -count=1`
- `make pair-session-watch`
- `make test-session-watch`
- `bin/pair --help`
- `bin/pair-dev --help`

Expected: all PASS.

- [x] **Step 4: Close through SDLC**

Run `sdlc actual --issue 78`, then `sdlc close --issue 78 --verified '<commands>'` with the verification evidence. Let the boundary review decide whether the title-poller split is sufficiently documented.

## Revisions

- 2026-06-30 — Boundary review found two corrections before shipping: near-miss candidates must not stop discovery before later valid candidates, and the Core Concepts table used the planned `WatcherRuntime` name even though implementation split the injected `Runtime` interface from concrete `OSRuntime`. Updated the plan names/locations and added tests/fixes for near-miss-before-valid ordering plus standalone `PAIR_TAG` fallback logging.
- 2026-06-30 — Re-review found tracker drift only: the Core Concepts table still used the pre-implementation `ResumeArgs` concept name instead of the shipped `StripResumeArgs` function, and the durable-plan checklist had not been marked complete. Updated the table/prose and checked off the delivered implementation steps.
