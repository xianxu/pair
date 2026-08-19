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
| `pidFileFresh` | `cmd/internal/sessionwatch/run.go` | new |
| `sessionWatcherSpawnArgv` | `cmd/internal/launcher/osruntime.go` | new |

- **`Options.PIDNotBefore`** — lower bound authorizing the PID-file generation for this watcher.
  - **Relationships:** One bound belongs to one watcher launch; `Run` compares every observed PID-file mtime against it.
  - **DRY rationale:** Supplies one bound to both PID reads instead of adding a second PID-selection path (ARCH-DRY).
  - **Future extensions:** Other sidecars can carry generation bounds through their own argv without widening this session identity contract.
- **`sessionwatch.CommandArgs`** — pure serializer for the internal watcher process contract.
  - **Relationships:** One launcher spawn produces one argv; `buildOptions` is its inverse consumer.
  - **DRY rationale:** Whole-workbench and agent-only restart reuse one command shape rather than maintaining parallel argv literals (ARCH-DRY, ARCH-PURPOSE).
  - **Future extensions:** New watcher-only metadata belongs before the `--` agent-argument delimiter.
- **`pidFileFresh`** — pure timestamp policy selecting precise native-generation comparison or legacy same-second comparison.
  - **Relationships:** `Run` calls it for both PID wait-loop and final bind; one policy owns all freshness decisions.
  - **DRY rationale:** Prevents separate native/legacy scan loops while making their intentional precision difference explicit (ARCH-PURE).
  - **Future extensions:** Remove legacy tolerance when old direct invocations are retired.
- **`sessionWatcherSpawnArgv`** — pure whole-workbench producer helper carrying
  the captured generation bound into the shared serializer.
  - **Relationships:** `OSRuntime.SpawnSessionWatcher` captures the clock and
    delegates its complete argv to this helper.
  - **DRY rationale:** The helper itself delegates command shape to
    `sessionwatch.CommandArgs`; it exists only to make producer wiring testable.
  - **Future extensions:** Additional launcher metadata stays in the shared
    serializer contract.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `OSRuntime.SpawnSessionWatcher` | `cmd/internal/launcher/osruntime.go` | modified | wall clock and detached process spawn |
| `freshAgentInvocation` | `cmd/internal/wrapcmd/wrap.go` | modified | Shift+Alt+N replacement process spawn |
| `buildOptions` | `cmd/internal/sessionwatch/runcli.go` | modified | internal process argv/environment |
| `pidFileCurrent` | `cmd/internal/sessionwatch/run.go` | new | PID-file mtime lookup through `Runtime` |
| `Runtime.ProcessIdentity` | `cmd/internal/sessionwatch/run.go` | new | stable OS process-incarnation lookup |
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
- **`pidFileCurrent`** — reads the PID-file mtime through the injected runtime and delegates the timestamp decision to pure `pidFileFresh`.
  - **Injected into:** `Run` through `sessionwatch.Runtime`.
  - **Future extensions:** None; all freshness policy belongs in `pidFileFresh`.
- **`Runtime.ProcessIdentity`** — captures and revalidates the bound process's
  kernel start token so a recycled numeric PID cannot extend watcher lifetime.
  - **Injected into:** `Run`; production uses `procutil.Identity`, while the
    stateful fake can replace one incarnation with another under the same PID.
  - **Future extensions:** Reuse this token wherever a long-lived sidecar owns
    process identity rather than mere liveness.
- **`sessionwatch.Runtime` fake** — models watcher start time, PID-file mtime, process liveness, and discovered rollout state across polls (ARCH-MOCK).
  - **Injected into:** `Run` tests.
  - **Future extensions:** None for this fix; the existing state model already covers the race.
- **Session-watch shell fixture** — executes the real self-dispatched watcher against temporary files and a stateful `lsof` fake.
  - **Injected into:** Repository integration tests; the fake persists process/path behavior across real CLI calls.
  - **Future extensions:** Add real-agent conformance only if platform mtime behavior diverges from the temporary filesystem.

### Test strategy

| Function / boundary | Adversarial strategy and mechanical guard |
|---|---|
| `sessionwatch.CommandArgs` + `buildOptions` | Round-trip arbitrary agent argv containing internal-flag-looking tokens; the first `--` mechanically separates watcher metadata from untouched agent args. |
| `freshAgentInvocation` | Drive every `SpecForAgent` member plus synchronous Claude through one table; watcher presence must equal registry membership, preventing a second agent list. |
| `pidFileFresh` | Generate subsecond mtime relations around the bound without filesystem IO; native mode uses exact ordering, while a separate zero-bound legacy assertion pins whole-second tolerance. |
| `sessionWatcherSpawnArgv` | Assert the helper used by `OSRuntime.SpawnSessionWatcher` carries the fixed generation bound into the shared serializer. |
| `Run` | Use the stateful clock/filesystem/process fake to permute PID write, watcher start, rollout appearance, process death, and PID reuse during discovery; config writes require incarnation authorization both before discovery and immediately before persistence. |
| Real `pair session-watch` process | Persist PID files on the temporary filesystem before process start on both sides of a fixed bound; the stateful `lsof` fake proves serializer/parser/mtime behavior through the production CLI seam (ARCH-MOCK). |

### Task 1: Establish the shared process contract (TDD)

**Files:** `cmd/internal/sessionwatch/runcli.go`, `cmd/internal/sessionwatch/runcli_test.go`, `cmd/internal/launcher/osruntime.go`, `cmd/internal/launcher/osruntime_test.go`, `cmd/internal/wrapcmd/wrap.go`, `cmd/internal/wrapcmd/agent_restart_test.go`

- [x] Add failing tests for the `CommandArgs`/`buildOptions` round trip and `freshAgentInvocation` registry strategy; run the focused packages and confirm failure for the absent contract.
- [x] Implement the shared RFC3339Nano command serializer, typed parser, producer clock capture, and `SpecForAgent`-derived Shift+Alt+N decision.
- [x] Run `go test ./cmd/internal/launcher ./cmd/internal/sessionwatch ./cmd/internal/wrapcmd -count=1 -timeout=30s`; expect PASS.

### Task 2: Make PID freshness generation-aware (TDD)

**Files:** `cmd/internal/sessionwatch/run.go`, `cmd/internal/sessionwatch/run_test.go`

- [x] Add failing property/table coverage for `pidFileCurrent` and stateful timing coverage for `Run`; confirm the current watcher-start comparison fails the native-generation strategy.
- [x] Implement one freshness policy used by both PID reads: exact comparison for nonzero launcher bounds and historical whole-second comparison for zero-bound legacy calls.
- [x] Run `go test ./cmd/internal/sessionwatch -count=1 -timeout=15s`; expect PASS across #143 lifecycle and #144 root-identity coverage.

### Task 3: Verify the real boundary and reconcile artifacts

**Files:** `tests/pair-session-watch-test.sh`, `atlas/architecture.md` (if stale), `atlas/how-to-bring-up-a-new-harness-cli.md` (if stale), `workshop/issues/000143-keep-agent-session-discovery-alive-after-startup-timeout.md`

- [x] Extend the stateful shell fixture for the real-process timing strategy; run `env -u PAIR_TAG -u PAIR_AGENT -u PAIR_SESSION_ID -u PAIR_DATA_DIR bash tests/pair-session-watch-test.sh`; expect PASS.
- [x] Shadow-search with `rg -n 'fresh PID|watchStart|watcher start|PID file|session-watch' atlas cmd/internal/sessionwatch cmd/internal/launcher cmd/internal/wrapcmd` and update only descriptions that contradict launcher-generation freshness (ARCH-PURPOSE).
- [x] Run `env -u PAIR_TAG -u PAIR_AGENT -u PAIR_SESSION_ID -u PAIR_DATA_DIR go test ./... -count=1`, `env -u PAIR_TAG -u PAIR_AGENT -u PAIR_SESSION_ID -u PAIR_DATA_DIR make test`, and `git diff --check`; all must exit 0.
- [x] Tick the reopened issue rows and record RED/GREEN, integration, and shadow-sweep evidence in `## Log` before the boundary close.

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

### 2026-08-19 09:15 PDT — Plan-quality gate PQ-1

- Compressed enumerated cases and procedural diff instructions into named-function adversarial strategies while retaining explicit TDD and verification commands.

### 2026-08-19 09:32 PDT — Boundary review corrections

- Extracted pure `pidFileFresh(mod, bound, watchStart)` and reclassified
  `pidFileCurrent` as the injected filesystem integration that delegates to it
  (ARCH-PURE). Removed the obsolete `freshPID` reuse claim.
- Made the incomplete-readiness test clear `PAIR_SESSION_NAME` and
  `PAIR_LAUNCH_NONCE` itself, so the exact focused command is hermetic inside a
  live Pair session.
- Replaced the dispatcher's Codex/Agy help restatement with registry-independent
  “async agent” wording (ARCH-PURPOSE).

### 2026-08-19 09:40 PDT — Process-incarnation binding

- Extended lifecycle binding from numeric PID liveness to a stable kernel
  process-start token, revalidated on every poll. The stateful fake now models
  PID reuse and proves an unrelated replacement process cannot authorize a
  session (ARCH-MOCK, ARCH-PURPOSE).
- Routed whole-workbench argv construction through tested
  `sessionWatcherSpawnArgv`, which is the helper called by
  `OSRuntime.SpawnSessionWatcher`; this pins the producer wiring, not only the
  downstream serializer.

### 2026-08-19 09:50 PDT — Persistence reauthorization

- Closed the mid-discovery TOCTOU window by revalidating process incarnation
  after `discover` returns a candidate and before ledger/config persistence. A
  stateful `LsofPaths` hook now recycles the PID during discovery and proves the
  candidate is discarded (ARCH-MOCK, ARCH-PURPOSE).
- Completed the current-surface shadow sweep for the dispatcher route, recovery
  flag owner, signal registry, transcript comment, and migration inventory.
