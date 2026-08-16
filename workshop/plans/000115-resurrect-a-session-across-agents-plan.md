# Repo Agent Defaults Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `pair <agent>` remember and reuse the last explicit launch arguments for that agent in the current repo, without changing existing per-tag native resume behavior.

**Architecture:** Add a small pure policy layer for launch-argument precedence and local repo-agent default codecs, then wire it into the existing launcher create path behind a nonce-bound readiness record written by `pair wrap` after the agent PTY starts. This deliberately avoids the abandoned live handoff coordinator; `ARCH-PURPOSE` is served by landing the reusable substrate first, while `ARCH-DRY` keeps tag configs and repo-agent defaults in one precedence function and `ARCH-PURE` keeps policy IO-free.

**Tech Stack:** Go launcher (`cmd/internal/launcher`), existing repo-scoped Pair data dir, zellij launch readiness/create flow, README and atlas docs.

**Spec:** `workshop/issues/000115-resurrect-a-session-across-agents.md`

---

## Chunk 1: Repo-Agent Defaults

### Core concepts

#### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `LaunchArgs` intent fields | `cmd/internal/launcher/args.go` | modified |
| `AgentDefault` | `cmd/internal/launcher/agent_defaults.go` | new |
| `LaunchArgDecision` | `cmd/internal/launcher/launch_args_policy.go` | new |
| `ReadyRecord` | `cmd/internal/readiness/record.go` | new |
| `ScopedPaths.AgentDefault` | `cmd/internal/launcher/scoped_paths.go` | modified |

- **`LaunchArgs` intent fields** — records whether the user explicitly named an agent and whether `--` appeared, independently of the final defaulted `Agent` value.
  - **Relationships:** one parse result feeds one launcher create decision.
  - **DRY rationale:** prevents create-flow and default persistence from re-inferring operator intent from empty argument slices.
  - **Future extensions:** explicit work-selector intent can widen the same struct later.
- **`AgentDefault`** — local repo-scoped default launch args for one agent; contains no tag and no native session ID.
  - **Relationships:** one repo has zero or one default per agent; many tag creates may consume it.
  - **DRY rationale:** separates agent defaults from `config-<tag>-<agent>.json`, which is tag/native-session state.
  - **Future extensions:** add schema version or profile metadata if agent defaults become richer.
- **`LaunchArgDecision`** — pure precedence result for explicit args, tag saved config, repo-agent default, and native resume availability.
  - **Relationships:** consumes at most one tag config and one repo default; produces final args, optional resume ID, warnings, and whether to persist a default after readiness.
  - **DRY rationale:** one function owns `explicit > tag config > repo default > empty`, so future picker/recovery flows reuse it.
  - **Future extensions:** agent-specific resume capability stays in existing `composeResumeArgs` helpers.
- **`ReadyRecord`** — validated wire record proving the exact launched agent process for `(tag, agent, session, nonce)` reached PTY-start.
  - **Relationships:** one create launch mints one nonce; one `pair wrap` child writes one matching readiness record.
  - **DRY rationale:** readiness is a shared wire schema, not separate ad hoc JSON in launcher and wrapper.
  - **Future extensions:** later work-selector recovery can reuse the same readiness commit point without importing handoff journals.
- **`ScopedPaths.AgentDefault`** — repo-scoped path for `agent-default-<agent>.json`.
  - **Relationships:** sibling of existing scoped tag/config paths under the repo data dir.
  - **DRY rationale:** all repo-scoped launcher paths remain centralized.
  - **Future extensions:** reused by `pair list` or diagnostics if defaults become visible.

#### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `AgentDefaultOps` | `cmd/internal/launcher/runtime.go`, `cmd/internal/launcher/osruntime.go` | new | local filesystem default files |
| `ReadinessOps` | `cmd/internal/launcher/runtime.go`, `cmd/internal/launcher/osruntime.go`, `cmd/internal/wrapcmd/wrap.go` | new | `agent-ready-<tag>-<agent>.json` sidecar |
| Create-flow default persistence | `cmd/internal/launcher/createflow.go` | modified | launcher readiness / zellij handoff |

- **`AgentDefaultOps`** — runtime methods for reading and atomically writing one repo-agent default through the pure codec/path.
  - **Injected into:** create flow after it has resolved the target agent and tag.
  - **Future extensions:** diagnostics can read through the same seam.
- **`ReadinessOps`** — launcher removes stale ready records, mints/exports a launch nonce, waits briefly for the matching record, and checks the recorded PID is alive; `pair wrap` writes the record only after `pty.Start` succeeds.
  - **Injected into:** create flow before default persistence. Tests use the existing launcher fake plus a wrapcmd unit test; no fake zellij writes readiness on behalf of Pair (`ARCH-MOCK`).
  - **Future extensions:** if target startup needs richer health checks later, they extend this record after PTY-start rather than using blocking zellij exit.
- **Create-flow default persistence** — applies `LaunchArgDecision`, launches with final args, and writes/clears the repo-agent default only after successful launch readiness.
  - **Injected into:** existing `runCreate`; no new process coordinator.
  - **Future extensions:** later work-selector recovery can call the same policy before launching the target driver.

### Task 1: Preserve explicit launch intent

**Files:**
- Modify: `cmd/internal/launcher/args.go`
- Test: `cmd/internal/launcher/args_test.go`

- [x] **Step 1: Write failing parser tests**

Function strategy: `ParseArgs` over positional/separator-shaped argv -> table
test implicit agent, explicit agent, non-empty `--`, and empty `--`; the guard
is separate booleans rather than inferring intent from `Agent` or
`len(AgentArgs)`.

- [x] **Step 2: Run the focused tests and confirm RED**

Run: `go test ./cmd/internal/launcher -run TestParseArgs -count=1`

Expected: FAIL because `LaunchArgs` has no explicit-intent fields.

- [x] **Step 3: Add `AgentExplicit` and `AgentArgsExplicit`**

Set `AgentExplicit` only when a positional agent was typed. Set `AgentArgsExplicit` when `--` appears, even if no args follow. Keep `resume`, `continue`, `rename`, `restart`, and default bare `pair` behavior unchanged.

- [x] **Step 4: Re-run parser tests and confirm GREEN**

Run: `go test ./cmd/internal/launcher -run TestParseArgs -count=1`

Expected: PASS.

### Task 2: Add default codec, path, and pure precedence

**Files:**
- Create: `cmd/internal/launcher/agent_defaults.go`
- Create: `cmd/internal/launcher/agent_defaults_test.go`
- Create: `cmd/internal/launcher/launch_args_policy.go`
- Create: `cmd/internal/launcher/launch_args_policy_test.go`
- Modify: `cmd/internal/launcher/scoped_paths.go`
- Modify: `cmd/internal/launcher/scoped_paths_test.go`

- [x] **Step 1: Write failing codec/path tests**

Function strategy: `ParseAgentDefault` / `BuildAgentDefault` over malformed or
mismatched JSON -> reject wrong agent and preserve defensive copies;
`ScopedPaths.AgentDefault` over repo scopes and agent names -> path stays under
`ScopeDir`.

- [x] **Step 2: Implement the pure codec and path**

Create:

```go
type AgentDefault struct {
    Agent string   `json:"agent"`
    Args  []string `json:"args"`
}
```

Add parse/build helpers that reject empty or mismatched agents and normalize nil args to `[]`.

- [x] **Step 3: Write failing precedence tests**

Function strategy: `DecideLaunchArgs` over explicit args, tag config, repo
default, and stale native-session evidence -> assert precedence order and that
resume tokens are composed once via existing helpers.

- [x] **Step 4: Implement `DecideLaunchArgs`**

Keep this function pure. It should return final args, optional resume ID, warnings, and default persistence intent. Use existing `composeResumeArgs`, `extractExplicitResume`, and `persistedConfigArgs` helpers rather than duplicating agent-specific resume parsing (`ARCH-DRY`).

- [x] **Step 5: Run focused pure tests**

Run: `go test ./cmd/internal/launcher -run 'Test(AgentDefault|LaunchArg|ScopedPaths)' -count=1`

Expected: PASS.

### Task 3: Add nonce-bound launch readiness

**Files:**
- Create: `cmd/internal/readiness/record.go`
- Create: `cmd/internal/readiness/record_test.go`
- Create: `cmd/internal/launcher/readiness.go`
- Create: `cmd/internal/launcher/readiness_os.go`
- Test: `cmd/internal/launcher/readiness_test.go`
- Modify: `cmd/internal/launcher/runtime.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/wrapcmd/wrap.go`
- Test: `cmd/internal/wrapcmd/readiness_test.go`

- [x] **Step 1: Write failing pure readiness tests**

Function strategy: `readiness.Encode` / `readiness.Decode` over missing
identity fields and malformed JSON -> reject incomplete or mismatched records
before launcher IO trusts them.

- [x] **Step 2: Implement `ReadyRecord` codec**

Create a tiny shared package with fields `tag`, `agent`, `session`, `nonce`,
and `pid`. Validate non-empty identity and positive PID.

- [x] **Step 3: Write failing launcher readiness tests**

Function strategy: launcher readiness matcher over stale nonce/session/PID
records -> accept only exact identity and live PID; stale files are removed
before launch.

- [x] **Step 4: Add `ReadinessOps` to the runtime seam**

Add methods to remove stale records, mint/export nonce, wait for a matching
ready record with a short timeout, and test PID liveness. OS implementation
watches the local sidecar; fake runtime models only Pair-owned readiness
writes, not zellij effects (`ARCH-MOCK`).

- [x] **Step 5: Write failing wrap readiness tests**

Function strategy: `wrapcmd` startup over successful vs failed PTY start ->
write readiness only after agent PTY start succeeds, using `PAIR_TAG`,
`PAIR_AGENT`, `PAIR_SESSION_NAME`, and `PAIR_LAUNCH_NONCE`.

- [x] **Step 6: Implement wrap readiness publication**

After `pty.Start` succeeds and the agent PID is known, write the shared
`ReadyRecord` atomically beside the existing `agent-pid-<tag>` sidecar. If
required env is missing, skip readiness publication without breaking non-Pair
use.

- [x] **Step 7: Run readiness tests**

Run: `go test ./cmd/internal/readiness ./cmd/internal/launcher ./cmd/internal/wrapcmd -run 'Test.*Ready|Test.*Readiness' -count=1`

Expected: PASS.

### Task 4: Wire defaults into create flow

**Files:**
- Modify: `cmd/internal/launcher/runtime.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/createflow.go`
- Modify: `cmd/internal/launcher/createflow_test.go`
- Modify: `cmd/internal/launcher/osruntime_test.go`

- [x] **Step 1: Write failing create-flow tests**

Function strategy: `runCreate` over explicit args, empty explicit separator,
tag config, repo default, abort, and readiness timeout -> defaults persist only
after a matching ready record and tag configs keep priority.

- [x] **Step 2: Add `AgentDefaultOps` to the runtime seam**

Implement `ReadAgentDefault(agent string) AgentDefaultCandidate` and `WriteAgentDefault(agent string, args []string) error` in OS and fake runtimes. OS writes atomically under the repo-scoped data dir.

- [x] **Step 3: Replace the config-picker argument choice with `DecideLaunchArgs`**

Read the tag config/ledger as today, read the repo-agent default, compute the final args, print warnings, then continue through the existing session-id minting and codex `--no-alt-screen` normalization. If a valid native session ID is selected, compose the canonical resume invocation once.

- [x] **Step 4: Persist explicit defaults only after launch readiness**

Before `LaunchSession`, remove stale readiness for `(tag, agent)`, mint a
nonce, and export `PAIR_LAUNCH_NONCE`. Start the existing blocking zellij
handoff in the same shape, but observe the matching readiness sidecar
concurrently; write or clear the repo-agent default only after the exact record
appears with a live PID. If readiness times out or the child exits before
readiness, do not persist the default and return a launch failure.

- [x] **Step 5: Run focused launcher tests**

Run: `go test ./cmd/internal/launcher -run 'TestRunLaunch.*(Default|Config|Codex|Resume)' -count=1`

Expected: PASS.

### Task 5: Document and verify

**Files:**
- Modify: `README.md`
- Modify: `atlas/architecture.md`
- Modify: `atlas/session-identity.md`
- Modify: `workshop/issues/000115-resurrect-a-session-across-agents.md`

- [x] **Step 1: Update user docs**

Document that `pair <agent> -- <args>` records local repo-scoped defaults after a successful launch, `pair <agent>` reuses them for new sessions, and `pair <agent> --` clears them.

- [x] **Step 2: Update atlas**

Map the distinction between tag-specific configs and repo-agent defaults. Note that this is local machine state under Pair's repo-scoped data dir, not committed repo config.

- [x] **Step 3: Append issue log evidence**

Record tests, decisions, and the intentional exclusion of live handoff in `## Log` under `2026-08-16`.

- [x] **Step 4: Run verification**

Run:

```sh
go test ./cmd/internal/launcher -count=1
go test ./cmd/internal/readiness ./cmd/internal/wrapcmd -count=1
go test ./... -count=1
git diff --check
```

Expected: PASS.

- [x] **Step 5: Commit the milestone**

```sh
git add cmd/internal/launcher README.md atlas workshop/issues/000115-resurrect-a-session-across-agents.md workshop/plans/000115-resurrect-a-session-across-agents-plan.md
git commit -m "#115 M1: remember repo agent launch defaults" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

## Revisions

- 2026-08-16: plan-quality PQ-1/PQ-2 revision. Made nonce-bound launch
  readiness a concrete M1 deliverable instead of treating blocking zellij exit
  as readiness, and compressed test prose to named function strategies with
  adversarial input classes.
