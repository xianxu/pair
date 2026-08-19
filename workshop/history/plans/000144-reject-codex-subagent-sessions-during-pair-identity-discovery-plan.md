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
| `decideAutomaticResumeConfig` | `cmd/internal/launcher/markers.go` | new |

- **`CodexRootSessionID`** — authorizes a session UUID from a rollout path plus its first JSONL event.
  - **Relationships:** N:1 with a Codex process tree: many rollout candidates may be visible, exactly one accepted root ID is selected by each consumer.
  - **DRY rationale:** replaces filename-only authorization duplicated by sessionwatch, launcher, codexsid, slug, and Neovim (ARCH-DRY).
  - **Future extensions:** widen the accepted root source enum only when a captured upstream `session_meta` fixture proves a new root shape.
- **`decideAutomaticResumeConfig`** — projects a saved config plus already-observed validation facts into safe automatic-resume intent: preserve args, retain a validated ID, or clear an invalid ID and request quarantine; thin callers own warning/removal IO.
  - **Relationships:** 1:1 with a loaded saved config; consumed by both config-picker and restart-marker flows.
  - **DRY rationale:** prevents two automatic-resume boundaries from independently deciding whether persisted Codex identity is trustworthy.
  - **Future extensions:** agent-specific persisted-identity validators can join without weakening explicit user-supplied resume authority.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ReadCodexRootSessionID` | `cmd/internal/transcript/transcript.go` | new | bounded first-line filesystem read |
| `sessionwatch.Runtime.ReadFirstLine` | `cmd/internal/sessionwatch/run.go`, `runtime.go` | new | watcher filesystem seam |
| Shared process candidate seam | `cmd/internal/procutil/procutil.go` | existing/reused | `ps`/`lsof` parsing and traversal |
| Validated Pair config | `nvim/init.lua` | modified | asynchronous Go-authored session identity |

- **`ReadCodexRootSessionID`** — reads only the first JSONL event, then calls the pure classifier; unreadable, oversized, or unterminated/incomplete metadata fails closed.
  - **Injected into:** process candidate scans call this adapter after their existing `ps`/`lsof` seam returns paths; sessionwatch uses its own injected first-line reader and calls the pure classifier directly.
  - **Future extensions:** expose classification diagnostics if adaptation telemetry needs to distinguish malformed metadata from explicit subagents.
- **`sessionwatch.Runtime.ReadFirstLine`** — keeps watcher scheduling and candidate ordering deterministic under its stateful fake without reading whole, potentially large transcripts.
  - **Injected into:** `discover` and `discoverByBirth` candidate authorization.
  - **Future extensions:** none planned; deliberately narrower than general transcript reads.
- **Shared process candidate seam** — one implementation owns external `ps`/`lsof` parsing and traversal; tests provide temp rollout trees plus the existing fake command/state model (ARCH-MOCK).
  - **Injected into:** launcher `LiveAgentSessionID`, `codexsid.ResolveSessionID`, slug transcript resolution, and sessionwatch's OS runtime while sessionwatch retains its injected runtime interface.
  - **Future extensions:** command execution injection can widen here without duplicating parsers in consumers.
- **Validated Pair config** — Neovim reads `PAIR_SESSION_ID` or `config-<tag>-<agent>.json`; it no longer shells out or parses rollout filenames.
  - **Injected into:** review-target scoping.
  - **Future extensions:** a typed Pair identity sidecar could replace config reads if more UI consumers emerge.

## Chunk 1: Root classifier and consumers

### Task 1: Add the canonical Codex root-session classifier

**Files:**
- Modify: `cmd/internal/transcript/transcript.go`
- Modify: `cmd/internal/transcript/transcript_test.go`

- [x] **Step 1: Write failing pure-classifier tests**

Add table tests for:

```go
func TestCodexRootSessionIDFromEvent(t *testing.T) {
    // accept matching session_meta with parent_thread_id null/absent and source "cli" or "exec"
    // reject subagent object source + parent, mismatched payload.id, non-session_meta,
    // malformed/incomplete JSON, unknown string/object source, and malformed filename
}
```

Add file-adapter tests with a temp rollout tree proving `ReadCodexRootSessionID` reads a valid first event, rejects a subagent first event, and does not authorize a later `session_meta` when the first event is invalid. Define the bound as 1 MiB including the terminating newline, then cover nonexistent/unreadable input, exactly-at-limit acceptance, over-limit rejection, an unterminated first line, and a read-error path (directory or closed/erroring fixture) before implementation.

- [x] **Step 2: Run the focused tests and verify RED**

Run: `go test ./cmd/internal/transcript -run 'TestCodexRootSessionID|TestReadCodexRoot' -count=1 -v`

Expected: FAIL because the root classifier/file adapter does not exist.

- [x] **Step 3: Implement the minimal classifier and bounded reader**

Add:

```go
func CodexRootSessionID(path string, firstEvent []byte) string
func ReadCodexRootSessionID(path string) string
```

`CodexRootSessionID` must first extract the filename UUID, decode exactly one `session_meta`, require matching `payload.id`, nil/absent parent, and source `cli` or `exec`. `ReadCodexRootSessionID` reads one bounded line and delegates; it returns `""` for every IO/size/parse failure.

- [x] **Step 4: Run the focused tests and verify GREEN**

Run: `go test ./cmd/internal/transcript -count=1`

Expected: PASS.

- [x] **Step 5: Commit the classifier**

```bash
git add cmd/internal/transcript/transcript.go cmd/internal/transcript/transcript_test.go
git commit -m "transcript: #144: classify root Codex sessions" -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
```

### Task 2: Route process-based identity consumers through the classifier

**Files:**
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/osruntime_test.go`
- Modify: `cmd/internal/procutil/procutil.go`
- Modify: `cmd/internal/procutil/procutil_test.go`
- Modify: `cmd/internal/codexsid/codexsid.go`
- Modify: `cmd/internal/codexsid/codexsid_test.go`
- Modify: `cmd/internal/slugcmd/slugcmd.go`
- Modify: `cmd/internal/slugcmd/slugcmd_test.go`
- Modify: `cmd/internal/contextcmd/contextcmd_test.go`
- Modify: `cmd/internal/reviewcmd/run.go`
- Modify: `cmd/internal/reviewcmd/reviewcmd_test.go`
- Modify: `cmd/internal/reviewcmd/runtime.go`

- [x] **Step 1: Write failing ambiguous-candidate regressions**

For launcher, codexsid, and slug, create root and subagent rollout files in a temp Codex session tree. Have the shared process seam report the subagent first and root second. Assert each consumer skips the subagent and returns the root ID/path. Add a subagent-only case returning empty. Add config-backed regressions proving `transcript.SessionID` makes context and slug reject a polluted subagent config, and review targeting rejects polluted config before falling through to a valid live root.

- [x] **Step 2: Run the consumer tests and verify RED**

Run: `go test ./cmd/internal/transcript ./cmd/internal/launcher ./cmd/internal/procutil ./cmd/internal/codexsid ./cmd/internal/slugcmd ./cmd/internal/contextcmd ./cmd/internal/reviewcmd -run 'Codex.*(Root|Subagent)|LiveCodex|PollutedCodex' -count=1 -v`

Expected: FAIL because filename-only scans return the first subagent candidate.

- [x] **Step 3: Replace filename authorization with the shared adapter**

Make `procutil` the only Go owner of `ps`/`lsof` parsing/traversal and route launcher, codexsid, slug, and sessionwatch's OS runtime through it. Replace direct regex or `CodexSessionIDFromPath` success checks with `transcript.ReadCodexRootSessionID(path)`, continuing the scan when it returns empty. Delete `codexsid.rolloutRE`; retain `CodexSessionIDFromPath` only as the classifier's low-level path parser. Make `transcript.SessionID` validate Codex config IDs through the same file adapter, and expose that validated config resolution through reviewcmd's injected runtime rather than parsing JSON locally.

- [x] **Step 4: Run the three consumer packages and verify GREEN**

Run: `go test ./cmd/internal/transcript ./cmd/internal/launcher ./cmd/internal/procutil ./cmd/internal/codexsid ./cmd/internal/slugcmd ./cmd/internal/contextcmd ./cmd/internal/reviewcmd -count=1`

Expected: PASS.

- [x] **Step 5: Commit the consumer sweep**

```bash
git add cmd/internal/transcript cmd/internal/launcher/osruntime.go cmd/internal/launcher/osruntime_test.go cmd/internal/procutil cmd/internal/codexsid cmd/internal/slugcmd cmd/internal/contextcmd cmd/internal/reviewcmd
git commit -m "session: #144: reject live Codex subagents" -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
```

### Task 3: Make sessionwatch authorize Codex metadata

**Files:**
- Modify: `cmd/internal/sessionwatch/run.go`
- Modify: `cmd/internal/sessionwatch/runtime.go`
- Modify: `cmd/internal/sessionwatch/run_test.go`
- Modify: `cmd/internal/sessionwatch/sessionwatch.go`
- Modify: `cmd/internal/sessionwatch/sessionwatch_test.go`

- [x] **Step 1: Write failing watcher regressions**

Extend the fake runtime with first-event data. Add separate tests proving:

- `lsof` reports a subagent before a root and the root ID wins;
- birth-time discovery sees a newer subagent and an older eligible root and the root wins;
- subagent-only discovery writes no config and continues until process exit/timeout;
- a rejected malformed candidate does not hide a later root.

- [x] **Step 2: Run watcher tests and verify RED**

Run: `go test ./cmd/internal/sessionwatch -run 'Codex.*(Root|Subagent)|ContinuesPastRejected' -count=1 -v`

Expected: FAIL because `AgentSpec.Match` authorizes filename UUIDs without metadata.

- [x] **Step 3: Add the thin injected first-event seam and shared authorization**

Add `ReadFirstLine(path string) ([]byte, error)` to `Runtime` and implement it with the same 1 MiB contract as the transcript adapter. Route OS process traversal/path listing through `procutil`. Keep `AgentSpec.Match` as shape extraction, but before any Codex result becomes returnable, call `transcript.CodexRootSessionID(result.Path, firstEvent)`. Convert explicit subagents/invalid metadata to rejected candidates, not terminal near-misses, and continue scanning.

- [x] **Step 4: Run watcher tests and verify GREEN**

Run: `go test ./cmd/internal/sessionwatch -count=1`

Expected: PASS on both main and, after integration, the #143 lifecycle branch behavior.

- [x] **Step 5: Commit watcher authorization**

```bash
git add cmd/internal/sessionwatch
git commit -m "sessionwatch: #144: persist only root Codex sessions" -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
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

- [x] **Step 1: Write failing config-picker and Alt+n regressions**

First add pure table tests for `decideAutomaticResumeConfig(agent, saved, sessionValid)` returning sanitized config plus quarantine intent without IO. Pin the OS validator against a real on-disk subagent rollout, then drive the config/ledger policy through the launcher's stateful runtime fake:

- config picker warns, removes the polluted config, preserves saved args, and offers no resume action;
- `Alt+n` has no valid live ID, rejects the saved subagent, removes the config, and relaunches fresh with saved non-resume args;
- a valid root saved ID remains resumable;
- an explicit user `codex resume <id>` still bypasses automatic saved-state selection.

- [x] **Step 2: Run launcher tests and verify RED**

Run: `go test ./cmd/internal/launcher -run 'PollutedCodex|SavedCodexRoot|ExplicitCodexResume' -count=1 -v`

Expected: FAIL because restart fallback currently prefers `saved.SessionID` without root validation and config-picker leaves invalid config on disk.

- [x] **Step 3: Implement one automatic-resume validation policy**

Update `AgentSessionExists("codex", ...)` to resolve the rollout and require `ReadCodexRootSessionID`. Implement the pure decision over saved state plus `sessionValid`; thin config-picker and restart callers gather validity through `Runtime.AgentSessionExists`, apply returned quarantine/warning intent with `Remove`/stderr, and pass only sanitized state onward. Exercise both config-origin and ledger-fallback saved state. Do not apply this policy to explicit argv resume IDs.

- [x] **Step 4: Run launcher tests and verify GREEN**

Run: `go test ./cmd/internal/launcher -count=1`

Expected: PASS.

- [x] **Step 5: Commit persisted-state quarantine**

```bash
git add cmd/internal/launcher
git commit -m "launcher: #144: quarantine subagent resume state" -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
```

### Task 5: Remove Neovim's duplicate live rollout scanner

**Files:**
- Modify: `nvim/init.lua`
- Modify: `tests/review-toggle-test.sh`

- [x] **Step 1: Add a failing headless derivation regression**

Extend `tests/review-toggle-test.sh` so `current_session_id`:

- prefers non-empty `PAIR_SESSION_ID`;
- otherwise reads the validated config ID;
- returns nil when neither exists;
- never invokes fake `ps` or `lsof` binaries when config is absent.

- [x] **Step 2: Run the headless test and verify RED**

Run: `bash tests/review-toggle-test.sh`

Expected: FAIL because the current nil-config Codex path calls `live_codex_session_id`.

- [x] **Step 3: Delete Lua process/rollout discovery**

Remove `descendant_pids` and `live_codex_session_id`; keep `current_session_id` as `PAIR_SESSION_ID` then config only. Update comments to state that Go authorizes and quarantines automatic Codex identity.

- [x] **Step 4: Run the headless test and verify GREEN**

Run: `bash tests/review-toggle-test.sh`

Expected: PASS.

- [x] **Step 5: Commit UI derivation**

```bash
git add nvim/init.lua tests/review-toggle-test.sh
git commit -m "nvim: #144: consume validated session identity" -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
```

### Task 6: Update maps and verify the complete change

**Files:**
- Modify if needed: `atlas/session-identity.md`
- Modify: `workshop/issues/000144-reject-codex-subagent-sessions-during-pair-identity-discovery.md`

- [x] **Step 1: Update the atlas at the implemented boundary**

Document that Codex automatic identity requires matching root `session_meta`, persisted IDs are revalidated/quarantined, and Neovim derives from validated Pair state. Confirm `atlas/index.md` already links `session-identity.md`.

- [x] **Step 2: Run focused verification**

Run:

```bash
go test ./cmd/internal/transcript ./cmd/internal/sessionwatch ./cmd/internal/launcher ./cmd/internal/procutil ./cmd/internal/codexsid ./cmd/internal/slugcmd ./cmd/internal/contextcmd ./cmd/internal/reviewcmd -count=1
bash tests/review-toggle-test.sh
```

Expected: PASS.

- [x] **Step 3: Run repository-wide verification**

Run the repository's available full suite from the checkout, including generated runtime assets if required by the current Make targets:

```bash
go test ./... -count=1
if test ! -e ../ariadne && test ! -L ../ariadne; then ln -s /Users/xianxu/workspace/ariadne ../ariadne; fi
test "$(cd ../ariadne && pwd -P)" = /Users/xianxu/workspace/ariadne
make test
git diff --check
```

Expected: PASS with no warnings attributable to the change. The guarded setup materializes the canonical sibling only when no path exists; it refuses an existing wrong/broken link instead of replacing it. This makes Pair's `Makefile -> ../ariadne/Makefile` and nested plain `make -C "$repo_root"` calls resolve normally. `make -n test` was verified after this setup in the planning checkout. If the canonical repo is unavailable, stop before testing.

- [x] **Step 4: Perform a shadow-sweep**

Run:

```bash
rg -n 'CodexSessionIDFromPath|rolloutRE|endUUIDRE|live_codex_session_id|\.codex/sessions|transcript(pkg)?\.SessionID|transcript(pkg)?\.Resolve|sessionFromConfig|session_id' cmd nvim --glob '*.go' --glob '*.lua'
```

Expected: every path that authorizes automatic identity reaches the shared root classifier; no Neovim or package-local filename-only authorizer remains. Low-level path-shape tests may remain only in `transcript`.

- [x] **Step 5: Record evidence and check every issue-plan box**

Append TDD red/green commands, focused/full verification, shadow-sweep result, and atlas disposition to `## Log`; tick all issue and durable-plan checkboxes. Do not hand-edit issue status.

- [x] **Step 6: Commit documentation and verification record**

```bash
git add atlas/session-identity.md workshop/issues/000144-reject-codex-subagent-sessions-during-pair-identity-discovery.md workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan.md
git commit -m "workshop: #144: record root session verification" -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
```

- [x] **Step 7: Close through the SDLC boundary**

Run:

```bash
sdlc close --issue 144 --verified '<focused tests, full suite, headless Neovim regression, shadow-sweep, and atlas evidence>'
```

Expected: mandatory fresh-context review passes after any Critical/Important findings are fixed; close records measured actual time and moves the issue to `codecomplete`.

## Revisions

### 2026-08-19 07:29 PDT — Fresh-context plan review

- Expanded the consumer sweep to config-backed context, slug, and review-target
  identity, including validated `transcript.SessionID` and a broader shadow
  sweep.
- Consolidated `ps`/`lsof` parsing and traversal in `procutil` instead of
  deferring known duplication.
- Added exact 1 MiB reader-boundary/error tests and separated pure automatic
  resume decisions from caller-owned validation/removal/warning IO.
- Replaced the broken worktree-relative `make test` path with the canonical
  Makefile invocation and added required co-author trailers to every commit.

### 2026-08-19 07:34 PDT — Worktree Makefile validation

- Replaced the still-insufficient absolute `make -f` invocation with a guarded
  sibling-repo symlink setup. Verified `make -n test` resolves the full suite,
  including nested plain `make` calls, from the temporary Pair checkout.

### 2026-08-19 07:55 PDT — Boundary review alignment

- Corrected the Core concepts table to the implemented unexported
  `decideAutomaticResumeConfig` in `markers.go`; warning/removal remain thin
  caller IO.
- Recorded `procutil` as an existing reused process seam rather than a modified
  entity.
- Added the requested real-rollout rejection, subagent-only consumer,
  malformed-before-root watcher, and no-Neovim-subprocess regressions; no
  planned coverage is intentionally dropped.

### 2026-08-19 08:01 PDT — FIX-THEN-SHIP artifact correction

- Removed the generated raw close-review transcript from the shipping change;
  its embedded diff carried upstream whitespace and made the branch-wide
  `git diff --check` fail despite clean source files.
- Re-ran branch-wide `git diff --check` after removal.
