# Repo Agent Defaults Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `pair <agent>` remember and reuse the last explicit launch arguments for that agent in the current repo, without changing existing per-tag native resume behavior.

**Architecture:** Add a small pure policy layer for launch-argument precedence and local repo-agent default codecs, then wire it into the existing launcher create path. This deliberately avoids the abandoned live handoff coordinator; `ARCH-PURPOSE` is served by landing the reusable substrate first, while `ARCH-DRY` keeps tag configs and repo-agent defaults in one precedence function and `ARCH-PURE` keeps policy IO-free.

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
- **`ScopedPaths.AgentDefault`** — repo-scoped path for `agent-default-<agent>.json`.
  - **Relationships:** sibling of existing scoped tag/config paths under the repo data dir.
  - **DRY rationale:** all repo-scoped launcher paths remain centralized.
  - **Future extensions:** reused by `pair list` or diagnostics if defaults become visible.

#### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `AgentDefaultOps` | `cmd/internal/launcher/runtime.go`, `cmd/internal/launcher/osruntime.go` | new | local filesystem default files |
| Create-flow default persistence | `cmd/internal/launcher/createflow.go` | modified | launcher readiness / zellij handoff |

- **`AgentDefaultOps`** — runtime methods for reading and atomically writing one repo-agent default through the pure codec/path.
  - **Injected into:** create flow after it has resolved the target agent and tag.
  - **Future extensions:** diagnostics can read through the same seam.
- **Create-flow default persistence** — applies `LaunchArgDecision`, launches with final args, and writes/clears the repo-agent default only after successful launch readiness.
  - **Injected into:** existing `runCreate`; no new process coordinator.
  - **Future extensions:** later work-selector recovery can call the same policy before launching the target driver.

### Task 1: Preserve explicit launch intent

**Files:**
- Modify: `cmd/internal/launcher/args.go`
- Test: `cmd/internal/launcher/args_test.go`

- [ ] **Step 1: Write failing parser tests**

Add cases for:

```go
// argv nil: Agent="claude", AgentExplicit=false, AgentArgsExplicit=false
// ["codex"]: Agent="codex", AgentExplicit=true, AgentArgsExplicit=false
// ["codex", "--", "--sandbox", "danger-full-access"]: AgentExplicit=true, AgentArgsExplicit=true
// ["codex", "--"]: AgentExplicit=true, AgentArgsExplicit=true, AgentArgs=[]
```

- [ ] **Step 2: Run the focused tests and confirm RED**

Run: `go test ./cmd/internal/launcher -run TestParseArgs -count=1`

Expected: FAIL because `LaunchArgs` has no explicit-intent fields.

- [ ] **Step 3: Add `AgentExplicit` and `AgentArgsExplicit`**

Set `AgentExplicit` only when a positional agent was typed. Set `AgentArgsExplicit` when `--` appears, even if no args follow. Keep `resume`, `continue`, `rename`, `restart`, and default bare `pair` behavior unchanged.

- [ ] **Step 4: Re-run parser tests and confirm GREEN**

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

- [ ] **Step 1: Write failing codec/path tests**

Cover JSON round-trip, wrong embedded agent rejection, malformed JSON rejection, defensive slice copies, and `AgentDefaultPath("/data", "codex") == "/data/agent-default-codex.json"`.

- [ ] **Step 2: Implement the pure codec and path**

Create:

```go
type AgentDefault struct {
    Agent string   `json:"agent"`
    Args  []string `json:"args"`
}
```

Add parse/build helpers that reject empty or mismatched agents and normalize nil args to `[]`.

- [ ] **Step 3: Write failing precedence tests**

Cover:

```text
explicit args -> explicit args, persist default
explicit empty -- -> empty args, persist default clear
valid tag config + resumable id -> saved args + resume id, no default persist
valid tag config + stale id -> saved args fresh, warning, no default persist
no tag config + repo default -> default args, no default persist
malformed tag config + repo default -> default args + warning
no inputs -> empty args
```

- [ ] **Step 4: Implement `DecideLaunchArgs`**

Keep this function pure. It should return final args, optional resume ID, warnings, and default persistence intent. Use existing `composeResumeArgs`, `extractExplicitResume`, and `persistedConfigArgs` helpers rather than duplicating agent-specific resume parsing (`ARCH-DRY`).

- [ ] **Step 5: Run focused pure tests**

Run: `go test ./cmd/internal/launcher -run 'Test(AgentDefault|LaunchArg|ScopedPaths)' -count=1`

Expected: PASS.

### Task 3: Wire defaults into create flow

**Files:**
- Modify: `cmd/internal/launcher/runtime.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/createflow.go`
- Modify: `cmd/internal/launcher/createflow_test.go`
- Modify: `cmd/internal/launcher/osruntime_test.go`

- [ ] **Step 1: Write failing create-flow tests**

Add fake-runtime tests for:

```text
pair codex -- --sandbox danger-full-access persists codex default after ready
pair codex reuses codex default when no tag config exists
pair codex -- clears codex default after ready
tag config beats repo default
aborted config/name/picker path does not persist a default
failed launch/readiness does not persist a default
```

- [ ] **Step 2: Add `AgentDefaultOps` to the runtime seam**

Implement `ReadAgentDefault(agent string) AgentDefaultCandidate` and `WriteAgentDefault(agent string, args []string) error` in OS and fake runtimes. OS writes atomically under the repo-scoped data dir.

- [ ] **Step 3: Replace the config-picker argument choice with `DecideLaunchArgs`**

Read the tag config/ledger as today, read the repo-agent default, compute the final args, print warnings, then continue through the existing session-id minting and codex `--no-alt-screen` normalization. If a valid native session ID is selected, compose the canonical resume invocation once.

- [ ] **Step 4: Persist explicit defaults only after launch readiness**

Current create flow uses a blocking `LaunchSession`. If no readiness seam exists on current `main`, use the existing successful handoff return as the readiness point for M1 and document the limitation in the issue log. If the readiness seam can be ported cleanly without the handoff coordinator, add it as a small follow-up task before persistence.

- [ ] **Step 5: Run focused launcher tests**

Run: `go test ./cmd/internal/launcher -run 'TestRunLaunch.*(Default|Config|Codex|Resume)' -count=1`

Expected: PASS.

### Task 4: Document and verify

**Files:**
- Modify: `README.md`
- Modify: `atlas/architecture.md`
- Modify: `atlas/session-identity.md`
- Modify: `workshop/issues/000115-resurrect-a-session-across-agents.md`

- [ ] **Step 1: Update user docs**

Document that `pair <agent> -- <args>` records local repo-scoped defaults after a successful launch, `pair <agent>` reuses them for new sessions, and `pair <agent> --` clears them.

- [ ] **Step 2: Update atlas**

Map the distinction between tag-specific configs and repo-agent defaults. Note that this is local machine state under Pair's repo-scoped data dir, not committed repo config.

- [ ] **Step 3: Append issue log evidence**

Record tests, decisions, and the intentional exclusion of live handoff in `## Log` under `2026-08-16`.

- [ ] **Step 4: Run verification**

Run:

```sh
go test ./cmd/internal/launcher -count=1
go test ./... -count=1
git diff --check
```

Expected: PASS.

- [ ] **Step 5: Commit the milestone**

```sh
git add cmd/internal/launcher README.md atlas workshop/issues/000115-resurrect-a-session-across-agents.md workshop/plans/000115-resurrect-a-session-across-agents-plan.md
git commit -m "#115 M1: remember repo agent launch defaults" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```
