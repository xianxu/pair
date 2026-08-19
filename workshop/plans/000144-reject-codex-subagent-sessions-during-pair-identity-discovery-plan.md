# Reject Codex Subagent Sessions Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ensure every automatic Codex identity path accepts only the operator's root session and never persists or resumes a subagent rollout.

**Architecture:** Put semantic rollout classification in `cmd/internal/transcript` as a pure path-plus-first-event decision with a thin bounded file reader. Existing process/filesystem seams gather candidates; every consumer scans past rejected candidates and delegates authorization to the shared classifier. Automatic resume validates persisted IDs through the same contract, while Neovim stops reimplementing rollout discovery and consumes only validated Pair state.

**Tech Stack:** Go, JSONL, existing `procutil` and runtime fakes, Neovim Lua, shell-driven headless Neovim tests.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `CodexRootSessionID` | `cmd/internal/transcript/transcript.go` | new |
| `AutomaticResumeConfig` | `cmd/internal/launcher/createflow.go` | new |

- **`CodexRootSessionID`** — authorizes a session UUID from a rollout path plus its first JSONL event.
  - **Relationships:** N:1 with a Codex process tree: many rollout candidates may be visible, exactly one accepted root ID is selected by each consumer.
  - **DRY rationale:** replaces filename-only authorization duplicated by sessionwatch, launcher, codexsid, slug, and Neovim (ARCH-DRY).
  - **Future extensions:** widen the accepted root source enum only when a captured upstream `session_meta` fixture proves a new root shape.
- **`AutomaticResumeConfig`** — projects a saved config into safe automatic-resume state: preserve args, retain a validated ID, or clear an invalid ID and request quarantine.
  - **Relationships:** 1:1 with a loaded saved config; consumed by both config-picker and restart-marker flows.
  - **DRY rationale:** prevents two automatic-resume boundaries from independently deciding whether persisted Codex identity is trustworthy.
  - **Future extensions:** agent-specific persisted-identity validators can join without weakening explicit user-supplied resume authority.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ReadCodexRootSessionID` | `cmd/internal/transcript/transcript.go` | new | bounded first-line filesystem read |
| `sessionwatch.Runtime.ReadFirstLine` | `cmd/internal/sessionwatch/run.go`, `runtime.go` | new | watcher filesystem seam |
| Existing process candidate seams | `cmd/internal/launcher/osruntime.go`, `cmd/internal/codexsid/codexsid.go`, `cmd/internal/slugcmd/slugcmd.go` | modified | `ps`/`lsof` candidate discovery |
| Validated Pair config | `nvim/init.lua` | modified | asynchronous Go-authored session identity |

- **`ReadCodexRootSessionID`** — reads only the first JSONL event, then calls the pure classifier; unreadable, oversized, or unterminated/incomplete metadata fails closed.
  - **Injected into:** process candidate scans call this adapter after their existing `ps`/`lsof` seam returns paths; sessionwatch uses its own injected first-line reader and calls the pure classifier directly.
  - **Future extensions:** expose classification diagnostics if adaptation telemetry needs to distinguish malformed metadata from explicit subagents.
- **`sessionwatch.Runtime.ReadFirstLine`** — keeps watcher scheduling and candidate ordering deterministic under its stateful fake without reading whole, potentially large transcripts.
  - **Injected into:** `discover` and `discoverByBirth` candidate authorization.
  - **Future extensions:** none planned; deliberately narrower than general transcript reads.
- **Existing process candidate seams** — continue to own external `ps`/`lsof`; tests provide temp rollout trees plus the existing fake command/state model (ARCH-MOCK).
  - **Injected into:** launcher `LiveAgentSessionID`, `codexsid.ResolveSessionID`, and slug transcript resolution.
  - **Future extensions:** consolidate process traversal separately only if an active issue owns that refactor.
- **Validated Pair config** — Neovim reads `PAIR_SESSION_ID` or `config-<tag>-<agent>.json`; it no longer shells out or parses rollout filenames.
  - **Injected into:** review-target scoping.
  - **Future extensions:** a typed Pair identity sidecar could replace config reads if more UI consumers emerge.

## Chunk 1: Root classifier and consumers

### Task 1: Add the canonical Codex root-session classifier

**Files:**
- Modify: `cmd/internal/transcript/transcript.go`
- Modify: `cmd/internal/transcript/transcript_test.go`

- [ ] **Step 1: Write failing pure-classifier tests**

Add table tests for:

```go
func TestCodexRootSessionIDFromEvent(t *testing.T) {
    // accept matching session_meta with parent_thread_id null/absent and source "cli" or "exec"
    // reject subagent object source + parent, mismatched payload.id, non-session_meta,
    // malformed/incomplete JSON, unknown string/object source, and malformed filename
}
```

Add a file-adapter test with a temp rollout tree proving `ReadCodexRootSessionID` reads a valid first event, rejects a subagent first event, and does not authorize a later `session_meta` when the first event is invalid.

- [ ] **Step 2: Run the focused tests and verify RED**

Run: `go test ./cmd/internal/transcript -run 'TestCodexRootSessionID|TestReadCodexRoot' -count=1 -v`

Expected: FAIL because the root classifier/file adapter does not exist.

- [ ] **Step 3: Implement the minimal classifier and bounded reader**

Add:

```go
func CodexRootSessionID(path string, firstEvent []byte) string
func ReadCodexRootSessionID(path string) string
```

`CodexRootSessionID` must first extract the filename UUID, decode exactly one `session_meta`, require matching `payload.id`, nil/absent parent, and source `cli` or `exec`. `ReadCodexRootSessionID` reads one bounded line and delegates; it returns `""` for every IO/size/parse failure.

- [ ] **Step 4: Run the focused tests and verify GREEN**

Run: `go test ./cmd/internal/transcript -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the classifier**

```bash
git add cmd/internal/transcript/transcript.go cmd/internal/transcript/transcript_test.go
git commit -m "transcript: #144: classify root Codex sessions"
```

### Task 2: Route process-based identity consumers through the classifier

**Files:**
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/osruntime_test.go`
- Modify: `cmd/internal/codexsid/codexsid.go`
- Modify: `cmd/internal/codexsid/codexsid_test.go`
- Modify: `cmd/internal/slugcmd/slugcmd.go`
- Modify: `cmd/internal/slugcmd/slugcmd_test.go`

- [ ] **Step 1: Write failing ambiguous-candidate regressions**

For launcher, codexsid, and slug, create root and subagent rollout files in a temp Codex session tree. Have the existing process seam report the subagent first and root second. Assert each consumer skips the subagent and returns the root ID/path. Add a subagent-only case returning empty.

- [ ] **Step 2: Run the consumer tests and verify RED**

Run: `go test ./cmd/internal/launcher ./cmd/internal/codexsid ./cmd/internal/slugcmd -run 'Codex.*(Root|Subagent)|LiveCodex' -count=1 -v`

Expected: FAIL because filename-only scans return the first subagent candidate.

- [ ] **Step 3: Replace filename authorization with the shared adapter**

Keep existing `ps`/`lsof` candidate collection. Replace direct regex or `CodexSessionIDFromPath` success checks with `transcript.ReadCodexRootSessionID(path)`, continuing the scan when it returns empty. Delete `codexsid.rolloutRE`; retain `CodexSessionIDFromPath` only as the classifier's low-level path parser.

- [ ] **Step 4: Run the three consumer packages and verify GREEN**

Run: `go test ./cmd/internal/launcher ./cmd/internal/codexsid ./cmd/internal/slugcmd -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the consumer sweep**

```bash
git add cmd/internal/launcher/osruntime.go cmd/internal/launcher/osruntime_test.go cmd/internal/codexsid cmd/internal/slugcmd
git commit -m "session: #144: reject live Codex subagents"
```

### Task 3: Make sessionwatch authorize Codex metadata

**Files:**
- Modify: `cmd/internal/sessionwatch/run.go`
- Modify: `cmd/internal/sessionwatch/runtime.go`
- Modify: `cmd/internal/sessionwatch/run_test.go`
- Modify: `cmd/internal/sessionwatch/sessionwatch.go`
- Modify: `cmd/internal/sessionwatch/sessionwatch_test.go`

- [ ] **Step 1: Write failing watcher regressions**

Extend the fake runtime with first-event data. Add separate tests proving:

- `lsof` reports a subagent before a root and the root ID wins;
- birth-time discovery sees a newer subagent and an older eligible root and the root wins;
- subagent-only discovery writes no config and continues until process exit/timeout;
- a rejected malformed candidate does not hide a later root.

- [ ] **Step 2: Run watcher tests and verify RED**

Run: `go test ./cmd/internal/sessionwatch -run 'Codex.*(Root|Subagent)|ContinuesPastRejected' -count=1 -v`

Expected: FAIL because `AgentSpec.Match` authorizes filename UUIDs without metadata.

- [ ] **Step 3: Add the thin injected first-event seam and shared authorization**

Add `ReadFirstLine(path string) ([]byte, error)` to `Runtime` and implement a bounded OS reader. Keep `AgentSpec.Match` as shape extraction, but before any Codex result becomes returnable, call `transcript.CodexRootSessionID(result.Path, firstEvent)`. Convert explicit subagents/invalid metadata to rejected candidates, not terminal near-misses, and continue scanning.

- [ ] **Step 4: Run watcher tests and verify GREEN**

Run: `go test ./cmd/internal/sessionwatch -count=1`

Expected: PASS on both main and, after integration, the #143 lifecycle branch behavior.

- [ ] **Step 5: Commit watcher authorization**

```bash
git add cmd/internal/sessionwatch
git commit -m "sessionwatch: #144: persist only root Codex sessions"
```

## Chunk 2: Persisted-state safety and UI derivation

### Task 4: Quarantine polluted automatic-resume state

**Files:**
- Modify: `cmd/internal/launcher/createflow.go`
- Modify: `cmd/internal/launcher/createflow_test.go`
- Modify: `cmd/internal/launcher/lifecycle.go`
- Modify: `cmd/internal/launcher/markers_test.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/osruntime_test.go`

- [ ] **Step 1: Write failing config-picker and Alt+n regressions**

Add tests where saved config/ledger contains a real on-disk subagent rollout ID:

- config picker warns, removes the polluted config, preserves saved args, and offers no resume action;
- `Alt+n` has no valid live ID, rejects the saved subagent, removes the config, and relaunches fresh with saved non-resume args;
- a valid root saved ID remains resumable;
- an explicit user `codex resume <id>` still bypasses automatic saved-state selection.

- [ ] **Step 2: Run launcher tests and verify RED**

Run: `go test ./cmd/internal/launcher -run 'PollutedCodex|SavedCodexRoot|ExplicitCodexResume' -count=1 -v`

Expected: FAIL because restart fallback currently prefers `saved.SessionID` without root validation and config-picker leaves invalid config on disk.

- [ ] **Step 3: Implement one automatic-resume validation policy**

Update `AgentSessionExists("codex", ...)` to resolve the rollout and require `ReadCodexRootSessionID`. Add one launcher helper that validates saved automatic Codex IDs through `Runtime.AgentSessionExists`, clears invalid IDs while preserving args, removes the polluted config, and emits the warning. Use it in both config-picker and restart-marker re-entry before `planRestart`; do not apply it to explicit argv resume IDs.

- [ ] **Step 4: Run launcher tests and verify GREEN**

Run: `go test ./cmd/internal/launcher -count=1`

Expected: PASS.

- [ ] **Step 5: Commit persisted-state quarantine**

```bash
git add cmd/internal/launcher
git commit -m "launcher: #144: quarantine subagent resume state"
```

### Task 5: Remove Neovim's duplicate live rollout scanner

**Files:**
- Modify: `nvim/init.lua`
- Modify: `tests/review-toggle-test.sh`

- [ ] **Step 1: Add a failing headless derivation regression**

Extend `tests/review-toggle-test.sh` so `current_session_id`:

- prefers non-empty `PAIR_SESSION_ID`;
- otherwise reads the validated config ID;
- returns nil when neither exists;
- never invokes fake `ps` or `lsof` binaries when config is absent.

- [ ] **Step 2: Run the headless test and verify RED**

Run: `bash tests/review-toggle-test.sh`

Expected: FAIL because the current nil-config Codex path calls `live_codex_session_id`.

- [ ] **Step 3: Delete Lua process/rollout discovery**

Remove `descendant_pids` and `live_codex_session_id`; keep `current_session_id` as `PAIR_SESSION_ID` then config only. Update comments to state that Go authorizes and quarantines automatic Codex identity.

- [ ] **Step 4: Run the headless test and verify GREEN**

Run: `bash tests/review-toggle-test.sh`

Expected: PASS.

- [ ] **Step 5: Commit UI derivation**

```bash
git add nvim/init.lua tests/review-toggle-test.sh
git commit -m "nvim: #144: consume validated session identity"
```

### Task 6: Update maps and verify the complete change

**Files:**
- Modify if needed: `atlas/session-identity.md`
- Modify: `workshop/issues/000144-reject-codex-subagent-sessions-during-pair-identity-discovery.md`

- [ ] **Step 1: Update the atlas at the implemented boundary**

Document that Codex automatic identity requires matching root `session_meta`, persisted IDs are revalidated/quarantined, and Neovim derives from validated Pair state. Confirm `atlas/index.md` already links `session-identity.md`.

- [ ] **Step 2: Run focused verification**

Run:

```bash
go test ./cmd/internal/transcript ./cmd/internal/sessionwatch ./cmd/internal/launcher ./cmd/internal/codexsid ./cmd/internal/slugcmd -count=1
bash tests/review-toggle-test.sh
```

Expected: PASS.

- [ ] **Step 3: Run repository-wide verification**

Run the repository's available full suite from the checkout, including generated runtime assets if required by the current Make targets:

```bash
go test ./... -count=1
make test
git diff --check
```

Expected: PASS with no warnings attributable to the change. If `make test` owns additional generated steps, follow its next-action output rather than bypassing it.

- [ ] **Step 4: Perform a shadow-sweep**

Run:

```bash
rg -n 'CodexSessionIDFromPath|rolloutRE|endUUIDRE|live_codex_session_id|\.codex/sessions' cmd nvim --glob '*.go' --glob '*.lua'
```

Expected: every path that authorizes automatic identity reaches the shared root classifier; no Neovim or package-local filename-only authorizer remains. Low-level path-shape tests may remain only in `transcript`.

- [ ] **Step 5: Record evidence and check every issue-plan box**

Append TDD red/green commands, focused/full verification, shadow-sweep result, and atlas disposition to `## Log`; tick all issue and durable-plan checkboxes. Do not hand-edit issue status.

- [ ] **Step 6: Commit documentation and verification record**

```bash
git add atlas/session-identity.md workshop/issues/000144-reject-codex-subagent-sessions-during-pair-identity-discovery.md workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan.md
git commit -m "workshop: #144: record root session verification"
```

- [ ] **Step 7: Close through the SDLC boundary**

Run:

```bash
sdlc close --issue 144 --verified '<focused tests, full suite, headless Neovim regression, shadow-sweep, and atlas evidence>'
```

Expected: mandatory fresh-context review passes after any Critical/Important findings are fixed; close records measured actual time and moves the issue to `codecomplete`.
