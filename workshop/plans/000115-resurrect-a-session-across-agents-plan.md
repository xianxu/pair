# Exclusive Agent Handoff Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let `pair <agent>` switch an existing repo-scoped work tag to a different exclusive agent driver while preserving tag input state, native per-agent conversations, and a continuation-ready transcript.

**Architecture:** Extend the launcher's pure decision core with explicit-agent intent, deterministic argument precedence, driver classification, and a journaled handoff state machine. Keep zellij/process/filesystem effects behind focused runtime seams; pair-wrap publishes nonce-bound readiness, while a repo/tag lock and durable journal make source shutdown, queue push-front, target launch, and recovery crash-safe.

**Tech Stack:** Go 1.x, zellij CLI/layout KDL, existing `cmd/internal/launcher` runtime interfaces, `cmd/internal/scrollbackcmd`, `cmd/internal/wrapcmd`, Neovim queue-file protocol, shell-backed process fakes, Markdown atlas.

**Spec:** `workshop/issues/000115-resurrect-a-session-across-agents.md`

---

## Chunk 1: Pure launch, default, and driver policy (M1)

### Core concepts

#### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `LaunchArgs` intent fields | `cmd/internal/launcher/args.go` | modified |
| `AgentDefault` | `cmd/internal/launcher/agent_defaults.go` | new |
| `LaunchArgDecision` | `cmd/internal/launcher/launch_args_policy.go` | new |
| `DriverEvidence` / `DriverDecision` | `cmd/internal/launcher/driver.go` | new |
| `AgentPickWork` / `AgentPickerRow` | `cmd/internal/launcher/pick.go` | new |
| `LaunchDecision` handoff fields | `cmd/internal/launcher/decision.go` | modified |

- **`LaunchArgs` intent fields** — preserve whether the user named an agent and whether `--` appeared, independently of defaulted agent name or argument count.
  - **Relationships:** one parse result owns one agent intent and feeds one launch/handoff decision.
  - **DRY rationale:** one parse-time truth prevents picker, defaults, and handoff code from re-inferring intent from empty slices.
  - **Future extensions:** additional explicit launch modifiers widen this value rather than inspecting raw argv again.
- **`AgentDefault`** — repository-scoped saved argument vector for one agent; it deliberately carries no tag or native session ID.
  - **Relationships:** N defaults per repo, one per agent; a default may feed many tag launches.
  - **DRY rationale:** removes the current temptation to treat an arbitrary tag config as the agent-wide default.
  - **Future extensions:** model/profile metadata can be added with a versioned JSON shape.
- **`LaunchArgDecision`** — deterministic result of explicit args, tag config validity, native artifact validity, and repo default precedence.
  - **Relationships:** one decision consumes at most one tag config and one agent default; it returns args, optional resume ID, warnings, and whether a successful ready launch must persist/clear defaults.
  - **DRY rationale:** replaces config-picker-specific branching and centralizes idempotent resume composition.
  - **Future extensions:** new agents add resume capability through the existing agent-argument helpers.
- **`DriverEvidence` / `DriverDecision`** — normalized live-session, session-index, agent-file, and ledger evidence for a tag.
  - **Relationships:** many evidence records classify to exactly one driver, no driver, or a conflict.
  - **DRY rationale:** picker labels and handoff preflight consume the same classification instead of choosing provenance independently.
  - **Future extensions:** legacy evidence sources append records without changing conflict policy.
- **`AgentPickWork` / `AgentPickerRow`** — classified work-state input and its pure display/selection projection for an explicit target agent.
  - **Relationships:** one classified tag becomes one row; one row maps to exactly one attach, create, handoff, or conflict-disabled selection.
  - **DRY rationale:** M4's fzf effect consumes the same pure labels/selections tested in M1.
  - **Future extensions:** additional badges widen the row without changing driver classification.
- **`LaunchDecision` handoff fields** — add an explicit `ActionHandoff` outcome carrying tag, public session, source agent, and target agent.
  - **Relationships:** one picker selection becomes one attach/create/handoff action.
  - **DRY rationale:** handoff is a first-class launch action, not a side-channel boolean inferred later.
  - **Future extensions:** non-agent driver types can become new actions without overloading create.

#### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Agent-default JSON codec | `cmd/internal/launcher/agent_defaults.go` | new | persisted repo/agent default wire format |
- **Agent-default JSON codec** — validates and renders the state format M2's filesystem store will read/write.
  - **Injected into:** M2 `AgentDefaultOps`; M1 tests the boundary without performing IO.
  - **Future extensions:** a schema version can be added before the store gains migrations.

### Task 1: Preserve explicit launch intent

**Files:**
- Modify: `cmd/internal/launcher/args.go`
- Test: `cmd/internal/launcher/args_test.go`

- [ ] **Step 1: Write failing parser tests for implicit agent, explicit agent, explicit non-empty `--`, and explicit empty `--`**

```go
func TestParseLaunchArgsPreservesIntent(t *testing.T) {
    cases := []struct {
        argv []string
        agentExplicit, argsExplicit bool
    }{
        {nil, false, false},
        {[]string{"codex"}, true, false},
        {[]string{"codex", "--", "--model", "x"}, true, true},
        {[]string{"codex", "--"}, true, true},
    }
    // ParseArgs and compare both booleans for every case.
}
```

- [ ] **Step 2: Run the focused tests and confirm RED**

Run: `go test ./cmd/internal/launcher -run 'TestParseLaunchArgs(PreservesIntent|DefaultsToClaude|AgentAndForwardedArgs)' -count=1`

Expected: FAIL because `LaunchArgs` has no explicit-intent fields.

- [ ] **Step 3: Add `AgentExplicit` and `AgentArgsExplicit` to `LaunchArgs` and set them only from raw argv structure**

```go
type LaunchArgs struct {
    // existing fields...
    AgentExplicit     bool
    AgentArgsExplicit bool
}
```

Keep `Agent="claude"` as the value default without setting `AgentExplicit`; set `AgentArgsExplicit` when the separator is encountered even if no tokens follow it. Subcommands and `resume` leave both false.

- [ ] **Step 4: Re-run parser tests and confirm GREEN**

Run: `go test ./cmd/internal/launcher -run 'TestParseLaunchArgs' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the parser contract**

```bash
git add cmd/internal/launcher/args.go cmd/internal/launcher/args_test.go
git commit -m "#115 M1: preserve explicit agent launch intent" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 2: Add repository-scoped defaults and deterministic argument precedence

**Files:**
- Create: `cmd/internal/launcher/agent_defaults.go`
- Create: `cmd/internal/launcher/agent_defaults_test.go`
- Create: `cmd/internal/launcher/launch_args_policy.go`
- Create: `cmd/internal/launcher/launch_args_policy_test.go`
- Modify: `cmd/internal/launcher/scoped_paths.go`
- Modify: `cmd/internal/launcher/scoped_paths_test.go`
- Modify: `cmd/internal/launcher/createlogic.go`
- Modify: `cmd/internal/launcher/createlogic_test.go`

- [ ] **Step 1: Write RED tests for the default JSON/path contract**

Pin `agent-default-codex.json`, JSON round-trip with `args: []`, wrong embedded agent rejection, malformed JSON rejection, and defensive slice copies.

Run: `go test ./cmd/internal/launcher -run 'Test(AgentDefault|ScopedPathsAgentDefault)' -count=1`

Expected: FAIL because the type/path/parser do not exist.

- [ ] **Step 2: Implement the small pure default codec and scoped path**

```go
type AgentDefault struct {
    Agent string   `json:"agent"`
    Args  []string `json:"args"`
}

func ParseAgentDefault(raw, requestedAgent string) (AgentDefault, error)
func BuildAgentDefault(agent string, args []string) (string, error)
func (p ScopedPaths) AgentDefault(agent string) string
```

Reject empty/mismatched agents; normalize nil arguments to `[]` on write.

- [ ] **Step 3: Run codec/path tests and confirm GREEN**

Run: `go test ./cmd/internal/launcher -run 'Test(AgentDefault|ScopedPathsAgentDefault)' -count=1`

Expected: PASS.

- [ ] **Step 4: Write a table-driven RED test for complete launch-argument precedence**

Cover explicit args (including explicit empty), usable config args, valid/invalid native session IDs, repo defaults, malformed/agent-mismatched config, unknown-agent resume, warnings, and `PersistDefault` intent.

```go
type LaunchArgDecision struct {
    Args           []string
    ResumeID       string
    PersistDefault bool
    DefaultArgs    []string
    Warnings       []string
}

type SavedConfigCandidate struct {
    Present        bool
    Config         savedConfig
    ParseError     string
    SessionExists  bool
}

type AgentDefaultCandidate struct {
    Present    bool
    Value      AgentDefault
    ParseError string
}

type LaunchArgInput struct {
    RequestedAgent    string
    ArgsExplicit      bool
    ExplicitArgs      []string
    Saved             SavedConfigCandidate
    Default           AgentDefaultCandidate
}

func DecideLaunchArgs(in LaunchArgInput) LaunchArgDecision
```

Expected decisions must include:

| Input | Args | Resume | Persist | Warning |
|---|---|---|---|---|
| explicit empty + valid saved Codex session | `resume <sid>` | `<sid>` | clear defaults | none |
| explicit args + stale saved ID | explicit args | none | replace defaults | stale session |
| no explicit + valid config args/session | config args + canonical resume | `<sid>` | no | none |
| no explicit + config args/stale ID | config args | none | no | stale session |
| malformed config + valid default | default args | none | no | malformed config |
| config agent mismatch + valid default | default args | none | no | agent mismatch |
| unknown agent + config ID | config args | none | no | resume unsupported |

Run: `go test ./cmd/internal/launcher -run 'TestDecideLaunchArgs' -count=1`

Expected: FAIL because the policy does not exist.

- [ ] **Step 5: Implement minimal pure precedence using existing `persistedConfigArgs`, `resumeToken`, and `composeResumeArgs` helpers**

Do not perform IO or print warnings in the policy. The policy itself checks requested-agent match and `resumeToken` support; `SessionExists` is the injected native-artifact result. A structurally usable config may contribute args even when its session artifact is stale; only a recognized non-empty ID with a present artifact contributes `ResumeID`. Explicit args win but still compose with a valid saved resume ID. `PersistDefault` mirrors separator presence, not argument length. Use stable warning strings asserted verbatim in the table-driven tests.

- [ ] **Step 6: Keep the current config picker wired until Chunk 2 replaces its caller**

Do not remove `configChoice`, `buildConfigChoices`, `selectAction`, `composeTagRestartArgs`, or `runConfigPicker` in M1: `createflow.go` still calls them. Mark the helpers with a comment pointing to the M2 replacement. M1 adds a tested pure policy with no production caller; Chunk 2 atomically swaps the caller and then deletes the obsolete choice surface.

- [ ] **Step 7: Run all launcher unit tests and confirm GREEN**

Run: `go test ./cmd/internal/launcher -count=1`

Expected: PASS; the existing interactive saved-config picker tests still pass unchanged alongside the new pure automatic-precedence tests. Chunk 2 removes the old expectations only when it replaces the production caller.

- [ ] **Step 8: Commit defaults and policy**

```bash
git add cmd/internal/launcher/agent_defaults.go cmd/internal/launcher/agent_defaults_test.go cmd/internal/launcher/launch_args_policy.go cmd/internal/launcher/launch_args_policy_test.go cmd/internal/launcher/scoped_paths.go cmd/internal/launcher/scoped_paths_test.go cmd/internal/launcher/createlogic.go cmd/internal/launcher/createlogic_test.go
git commit -m "#115 M1: define repo agent defaults and arg precedence" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 3: Classify tag drivers and model handoff selections

**Files:**
- Create: `cmd/internal/launcher/driver.go`
- Create: `cmd/internal/launcher/driver_test.go`
- Modify: `cmd/internal/launcher/session.go`
- Modify: `cmd/internal/launcher/decision.go`
- Modify: `cmd/internal/launcher/decision_test.go`
- Modify: `cmd/internal/launcher/pick.go`
- Modify: `cmd/internal/launcher/pick_test.go`

- [ ] **Step 1: Write RED driver-classification tests**

Cover one consistent live driver, no-live latest-ledger driver, no evidence, conflicting agent-file/session evidence, and multiple live sessions for one tag. Require a conflict to retain every evidence item for the error message.

```go
type DriverDecision struct {
    Agent     string
    Session   string
    State     SessionState
    Conflict  bool
    Evidence  []DriverEvidence
}

func ClassifyDriver(evidence []DriverEvidence) DriverDecision
```

Use these exact pure shapes:

```go
type DriverEvidenceKind string

const (
    EvidenceLiveSession DriverEvidenceKind = "live-session"
    EvidenceSessionIndex DriverEvidenceKind = "session-index"
    EvidenceAgentFile DriverEvidenceKind = "agent-file"
    EvidenceLedger DriverEvidenceKind = "ledger"
)

type DriverEvidence struct {
    Kind        DriverEvidenceKind
    SessionName string
    State       SessionState
    Agent       string
    At          time.Time
}
```

Named table cases and exact outcomes:

- `live-consistent-common-input`: one non-exited live-session record with an empty agent, its matching session-index record, and agent-file `claude` → Claude, that session/state, no conflict; the classifier must not require Zellij discovery to know the agent;
- `live-consistent-enriched-input`: the same records with the live record already annotated `claude` → the same decision;
- `live-agent-conflict`: same live/index plus agent-file `codex` → conflict with every input record retained;
- `multiple-live`: two non-exited session names for one tag → conflict with both retained;
- `historical-latest-ledger`: no non-exited live session and ledger rows Claude at T1/Codex at T2 → Codex;
- `historical-agent-file-not-authoritative`: no live/ledger, only agent-file → no driver;
- `missing-index`: live session without matching index → conflict because repo ownership is not proven; and
- `no-evidence`: empty decision.

Run: `go test ./cmd/internal/launcher -run 'TestClassifyDriver' -count=1`

Expected: FAIL because driver evidence does not exist.

- [ ] **Step 2: Implement pure classification with live-consistent evidence ahead of ledger evidence and no guessing on conflicts**

`DriverEvidence` is a normalized pure input contract in M1; it does not pretend current `Session` rows already carry complete evidence. Chunk 4 will collect source-labelled records from scoped live sessions, the session-name index, `agent-<tag>`, and the ledger, then pass them to this classifier. M1 tests each source and conflict combination synthetically.

Run: `go test ./cmd/internal/launcher -run 'TestClassifyDriver' -count=1`

Expected: PASS.

- [ ] **Step 3: Write RED tests for a separate pure agent-picker row/selection API**

Pin these cases:

- implicit bare `pair` omits attached rows;
- historical rows still obey the supplied cutoff snapshot;
- attached, detached, exited, and recent-inactive rows each carry that exact state label;
- conflict rows are non-selectable and identify the conflict;
- existing detached/historical ordering and queue badges do not regress.

Pin this exact resolution matrix:

| Agent explicit? | Conflict? | Work state | Driver relation | Selection |
|---|---|---|---|---|
| no | no | attached | any | row omitted |
| yes | yes | any | any | disabled/conflict, no action |
| yes | no | attached or detached | same agent | `ActionAttach` |
| yes | no | exited or inactive | same agent | `ActionCreate` on the same tag, resuming when valid config exists |
| yes | no | any | different known agent | `ActionHandoff` with source/target/session |
| yes | no | attached or detached | unknown driver | disabled/conflict because live provenance is inconsistent |
| yes | no | exited or inactive | unknown driver | `ActionHandoff` with empty source, requiring M4's tag-state-only confirmation |

Use this exact classified input rather than raw `SessionSnapshot`:

```go
type AgentPickWork struct {
    Tag         string
    SessionName string
    RepoName    string
    Driver      DriverDecision
    State       SessionState // attached, detached, exited, or inactive
    MTime       time.Time
    QueueCount  int
}

type AgentPickerPolicy struct {
    RequestedAgent string
    AgentExplicit  bool
}
```

Add `SessionInactive` for recent history with no zellij row. An exited zellij row is emitted as exited and deduplicates only the matching inactive history row; it must never disappear merely because historical data for the same tag exists.

Run: `go test ./cmd/internal/launcher -run 'Test(BuildAgentPickRows|ResolveAgentPickSelection)' -count=1`

Expected: FAIL because picker policy cannot express target-agent handoff.

- [ ] **Step 4: Add `buildAgentPickRows` and `resolveAgentPickSelection` as pure functions without changing live `resolvePick`**

```go
const ActionHandoff LaunchAction = "handoff"

type LaunchDecision struct {
    // existing fields...
    SourceAgent string
    TargetAgent string
}
```

`buildAgentPickRows` receives already classified rows plus `AgentPickerPolicy`; `resolveAgentPickSelection` maps a plain row to an attach/create/handoff decision without calling fzf. Do not add handoff rows to `buildPickRows`, do not change `resolvePick`, and do not return `ActionHandoff` from any live caller in M1. Chunk 4 gathers evidence, invokes fzf, then handles the new action in the same change.

- [ ] **Step 5: Run decision/picker tests and the full launcher package**

Run: `go test ./cmd/internal/launcher -count=1`

Expected: PASS.

- [ ] **Step 6: Commit the pure driver/picker core**

```bash
git add cmd/internal/launcher/driver.go cmd/internal/launcher/driver_test.go cmd/internal/launcher/session.go cmd/internal/launcher/decision.go cmd/internal/launcher/decision_test.go cmd/internal/launcher/pick.go cmd/internal/launcher/pick_test.go
git commit -m "#115 M1: model exclusive agent handoff decisions" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 4: Verify and close M1

**Files:**
- Modify: `workshop/issues/000115-resurrect-a-session-across-agents.md`
- Modify: `atlas/session-identity.md`

- [ ] **Step 1: Run milestone verification**

Run: `go test ./cmd/internal/launcher -count=1`

Expected: PASS.

Run: `go test ./...`

Expected: PASS.

Run: `git diff --check`

Expected: no output.

- [ ] **Step 2: Update the issue and atlas before the boundary**

Before running the close, ensure `## Plan` already contains this exact unchecked boundary row (the planning session adds it; do not tick it manually):

```markdown
- [ ] M1 — Define explicit launch intent, repo-agent defaults policy, driver classification, and pure handoff picker decisions.
```

The separate planning checkbox `Write the durable implementation plan after the approved spec passes review` must already be checked by the planning-session commit; do not change it during M1. Append M1 evidence/discoveries under a new dated `## Log` entry without rewriting prior logs. Update `atlas/session-identity.md` with tag-as-work, exclusive driver, explicit-agent intent, and repo-agent-default concepts; `atlas/index.md` already links this file.

- [ ] **Step 3: Commit milestone documentation**

```bash
git add atlas/session-identity.md workshop/issues/000115-resurrect-a-session-across-agents.md
git commit -m "#115 M1: map agent handoff policy" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

- [ ] **Step 4: Run the mechanical close plus auto-dispatched review and preserve its real exit status**

```bash
sdlc milestone-close --issue 115 --milestone M1 \
  --verified 'launcher unit suite and go test ./... pass; explicit intent, defaults policy, driver classification, and handoff picker decisions are covered' \
  > /tmp/pair-115-m1-close.out 2>&1
close_status=$?
sed -n '1,240p' /tmp/pair-115-m1-close.out
test "$close_status" -eq 0
```

Expected: exit 0; the command ticks M1, appends its log line, runs the fresh review, and prints `Review-Verdict:` plus `Review-Window:` trailers.

- [ ] **Step 5: Resolve the boundary review before committing it**

Read `/tmp/pair-115-m1-close.out` and require a real `Review-Verdict: SHIP` or `Review-Verdict: FIX-THEN-SHIP`.

- For `FIX-THEN-SHIP`, implement every Critical/Important fix in the planned M1 files, add a preventing rule to `workshop/lessons.md`, and rerun `go test ./cmd/internal/launcher -count=1`, `go test ./...`, and `git diff --check`. Do not rerun `sdlc milestone-close`; its protocol requires fixes in the one boundary commit.
- For `REWORK`, stop execution and return to planning.
- For `Review-Verdict: not-run`, parse the exact `Review-Window: BASE..HEAD`, run `sdlc judge milestone-review --base BASE --head HEAD --issue 115 --milestone M1 --agent codex > /tmp/pair-115-m1-rerun.out 2>&1`, print that file, and require the rerun output/exit status to classify as `SHIP` or `FIX-THEN-SHIP`. If dispatch still fails, stop: the mandatory fresh review has not happened. Append a dated issue-log correction recording the real rerun verdict; do not rewrite the earlier append-only `not-run` line. Record the rerun's real verdict plus the same exact `BASE..HEAD` as `/tmp/pair-115-m1-rerun.trailers`; this file supersedes the original `not-run` trailer for the boundary commit.

- [ ] **Step 6: Stage only the M1 boundary mutation and any review fixes**

```bash
git add cmd/internal/launcher atlas/session-identity.md workshop/lessons.md workshop/issues/000115-resurrect-a-session-across-agents.md
git diff --cached --check
```

Expected: no whitespace errors; unrelated user changes remain unstaged.

- [ ] **Step 7: Commit the durable M1 boundary with the emitted trailers verbatim**

Run `git commit` with subject `#115 M1: close launch policy milestone`, a body explaining why the policy is pure/unwired, and `Co-Authored-By: OpenAI Codex <noreply@openai.com>`. For an initially real verdict, paste the exact `Review-Verdict:` and `Review-Window:` lines from `/tmp/pair-115-m1-close.out`; after an initial `not-run`, paste the replacement lines from `/tmp/pair-115-m1-rerun.trailers`. Never commit the original `not-run` trailer after a successful rerun.

Expected: `git log -1 --format=%B` contains both review trailers and the co-author trailer; `git show HEAD:workshop/issues/000115-resurrect-a-session-across-agents.md | rg '^- \[x\] M1'` matches.

## Chunk 2: Nonce-bound readiness and automatic defaults (M2)

### Core concepts

#### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ReadyRecord` / `ReadyIdentity` wire contract | `cmd/internal/readiness/record.go` | new |
| `SessionLaunchRequest` | `cmd/internal/launcher/readiness.go` | new |
| `LaunchArgDecision` | `cmd/internal/launcher/launch_args_policy.go` | modified |

- **`ReadyRecord` / `ReadyIdentity`** — the single shared JSON codec and expected identity for one agent PTY startup; launcher and pair-wrap import this package rather than defining parallel schemas.
  - **Relationships:** one launch nonce maps to at most one accepted record; identity includes tag, agent, public session, nonce, and PID.
  - **DRY rationale:** ordinary create, target handoff, and source recovery share one exact matching rule.
  - **Future extensions:** health/version fields can be added without weakening identity.
- **`SessionLaunchRequest`** — immutable zellij create inputs plus ready path/identity/timeout.
  - **Relationships:** one request creates one controllable child launch.
  - **DRY rationale:** every fresh session launch uses the same start/readiness/wait contract.
  - **Future extensions:** another multiplexer maps the request to its own child handle.
- **`LaunchArgDecision`** — M1's pure result gains no IO; M2 consumes it in production and proves default persistence happens only after readiness.
  - **Relationships:** one create attempt owns one decision; its persistence intent survives until ready.
  - **DRY rationale:** production no longer reimplements precedence around the old config picker.
  - **Future extensions:** per-launch overrides remain values on the decision.

#### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `SessionLaunch` / `ZellijOps.StartSession` | `cmd/internal/launcher/runtime.go`, `cmd/internal/launcher/osruntime.go` | new | asynchronously started zellij child and exit status |
| `ReadinessOps` | `cmd/internal/launcher/runtime.go`, `cmd/internal/launcher/readiness_os.go` | new | ready sidecar removal, polling, PID liveness |
| `AgentDefaultOps` | `cmd/internal/launcher/runtime.go`, `cmd/internal/launcher/osruntime.go` | new | scoped agent-default JSON files |
| Pair-wrap ready emitter | `cmd/internal/wrapcmd/readiness.go`, `cmd/internal/wrapcmd/wrap.go` | new | atomic post-PTY-start ready record |

- **`SessionLaunch` / `ZellijOps.StartSession`** — start zellij without waiting, expose `WaitReady`, `Wait`, and nonce-scoped `Stop`, and keep attach behavior separate.
  - **Injected into:** `runCreate` now; M3/M4 handoff/recovery later.
  - **Future extensions:** backend-specific session launch handles.
- **`ReadinessOps`** — remove stale evidence, read a candidate record, and determine PID liveness under a bounded wait.
  - **Injected into:** the OS `SessionLaunch.WaitReady` loop; the pure matcher remains mock-free.
  - **Future extensions:** filesystem notification can replace polling without changing callers.
- **`AgentDefaultOps`** — read and atomically write one default through the M1 codec/path.
  - **Injected into:** create-flow source gathering and the post-readiness callback.
  - **Future extensions:** defaults inspection/reset CLI.
- **Pair-wrap ready emitter** — write a record only after `pty.Start` and successful agent PID publication.
  - **Injected into:** the pair-wrap startup sequence through inherited `PAIR_LAUNCH_NONCE` and `PAIR_SESSION_NAME`.
  - **Future extensions:** startup failure evidence.

### Task 5: Define and emit exact readiness evidence

**Files:**
- Create: `cmd/internal/readiness/record.go`
- Create: `cmd/internal/readiness/record_test.go`
- Create: `cmd/internal/launcher/readiness.go`
- Create: `cmd/internal/launcher/readiness_test.go`
- Modify: `cmd/internal/launcher/scoped_paths.go`
- Modify: `cmd/internal/launcher/scoped_paths_test.go`
- Create: `cmd/internal/wrapcmd/readiness.go`
- Create: `cmd/internal/wrapcmd/readiness_test.go`
- Modify: `cmd/internal/wrapcmd/wrap.go`

- [ ] **Step 1: Write RED launcher tests for ready JSON and identity matching**

```go
type ReadyRecord struct {
    Tag, Agent, Session, Nonce string
    PID                        int
}

type ReadyIdentity struct {
    Tag, Agent, Session, Nonce string
}

func (r ReadyRecord) Matches(want ReadyIdentity, pidAlive bool) bool
```

Cover exact match, wrong nonce/tag/agent/session, zero PID, dead PID, malformed JSON, nil slice/empty fields, and `ScopedPaths.Ready()` = `agent-ready-<tag>.json`.

Run: `go test ./cmd/internal/launcher -run 'Test(Ready|ScopedPathsReady)' -count=1`

Expected: FAIL because readiness types/path do not exist.

- [ ] **Step 2: Implement the pure codec/matcher once in `cmd/internal/readiness` and the launcher path separately**

Use `encoding/json` with stable keys `tag`, `agent`, `session`, `nonce`, `pid`; reject missing identity fields and non-positive PID before matching. `cmd/internal/launcher/readiness.go` may expose type aliases for local call-site readability, but owns no JSON fields or marshal/unmarshal implementation. Both launcher reads and wrapcmd writes call the shared codec; a golden round-trip test encodes through the producer-facing API and decodes/matches through the consumer-facing API.

Run: `go test ./cmd/internal/launcher -run 'Test(Ready|ScopedPathsReady)' -count=1`

Expected: PASS.

- [ ] **Step 3: Write RED pair-wrap tests for emission timing and contents**

Extract a helper that accepts an injected atomic writer. Assert no write before child start, no write without nonce/tag/session, and one exact record after a successful PID publication. Drive the existing `run` seam with a short-lived fake agent so a failed `pty.Start` never emits readiness.

Run: `go test ./cmd/internal/wrapcmd -run 'TestReadyRecord' -count=1`

Expected: FAIL because pair-wrap has no emitter.

- [ ] **Step 4: Implement pair-wrap emission immediately after the existing `agent-pid-<tag>` write using the shared codec**

Read `PAIR_TAG`, `PAIR_AGENT`, `PAIR_SESSION_NAME`, and `PAIR_LAUNCH_NONCE`; construct `readiness.Record`, encode it with `cmd/internal/readiness`, and publish `agent-ready-<tag>.json` through injected `osfs.FS.WriteAtomic`. Do not add another sibling-temp/rename implementation; if readiness later requires stronger durability, extend and test the shared primitive. Treat a readiness write failure as startup-significant: log it and continue the agent so the launcher times out visibly rather than killing a usable process.

- [ ] **Step 5: Run wrap and launcher readiness tests**

Run: `go test ./cmd/internal/wrapcmd ./cmd/internal/launcher -run 'Test(Ready|ScopedPathsReady)' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit readiness evidence**

```bash
git add cmd/internal/readiness cmd/internal/launcher/readiness.go cmd/internal/launcher/readiness_test.go cmd/internal/launcher/scoped_paths.go cmd/internal/launcher/scoped_paths_test.go cmd/internal/wrapcmd/readiness.go cmd/internal/wrapcmd/readiness_test.go cmd/internal/wrapcmd/wrap.go
git commit -m "#115 M2: publish nonce-bound agent readiness" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 6: Make fresh session launch observable while attached

**Files:**
- Modify: `cmd/internal/launcher/runtime.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Create: `cmd/internal/launcher/readiness_os.go`
- Create: `cmd/internal/launcher/readiness_os_test.go`
- Modify: `cmd/internal/launcher/createflow_test.go`

- [ ] **Step 1: Write RED fake-runtime tests for ready, timeout, child-exits-before-ready, and stop-before-ready**

Use this interface:

```go
type SessionLaunch interface {
    WaitReady() error
    Wait() (int, error)
    Stop() error
}

type SessionLaunchRequest struct {
    Session, ConfigDir, Layout string
    ReadyPath                  string
    Ready                      ReadyIdentity
    Timeout                    time.Duration
}

type ZellijOps interface {
    // existing methods...
    StartSession(SessionLaunchRequest) (SessionLaunch, error)
}
```

The fake records event order. Assert `start → ready → wait`; failure paths call `stop → wait` and never report ready.

Run: `go test ./cmd/internal/launcher -run 'TestSessionLaunch' -count=1`

Expected: FAIL because only blocking `LaunchSession` exists.

- [ ] **Step 2: Implement `osSessionLaunch` with one child waiter goroutine and pre-start stale cleanup**

`StartSession` removes the request's stale ready path before `cmd.Start`, then starts the child; this is the only stale-removal owner, so it cannot race with a newly published record. One goroutine calls `cmd.Wait` and publishes a cached result. `WaitReady` only polls the ready path, accepts `ReadyRecord.Matches`, and selects between timeout, child-exit channel, and match. `Wait` returns the cached result to every caller without another `cmd.Wait`. `Stop` runs `zellij delete-session <request.Session> --force`, terminates the exact launch child if still alive, and waits for both child exit and absence from `list-sessions`.

- [ ] **Step 3: Add OS-backed tests with a process-level fake zellij and temp ready file**

Put a fake `zellij` executable first on `PATH`. It records create/delete/list calls, starts a helper “agent” process for the named session, and removes that process/session only on `delete-session`. Test exact nonce success, wrong/stale nonce ignored until timeout, process exit before ready, and `Stop` issuing the named delete, terminating the fake agent, observing an empty session list, and reaping the launch child. No real daemon is used.

Run: `go test ./cmd/internal/launcher -run 'TestOSSessionLaunch' -count=1`

Expected: PASS.

- [ ] **Step 4: Add fake `StartSession` while retaining `LaunchSession` until Task 7 swaps the production caller**

Update `fakeRuntime` in `createflow_test.go` to expose scripted ready/exit/stop behavior and ordered events. Keep the old create-only `LaunchSession` method and `runCreate` call intact for this commit; Task 7 replaces the caller and removes the old method in one compiling change. Attach remains blocking throughout.

- [ ] **Step 5: Run the launcher suite**

Run: `go test ./cmd/internal/launcher -count=1`

Expected: PASS with existing attach/restart behavior unchanged.

- [ ] **Step 6: Commit the observable launch seam**

```bash
git add cmd/internal/launcher/runtime.go cmd/internal/launcher/osruntime.go cmd/internal/launcher/readiness_os.go cmd/internal/launcher/readiness_os_test.go cmd/internal/launcher/createflow_test.go
git commit -m "#115 M2: observe session readiness before blocking attach" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 7: Wire automatic precedence and readiness-gated defaults

**Files:**
- Modify: `cmd/internal/launcher/runtime.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/createflow.go`
- Modify: `cmd/internal/launcher/createflow_test.go`
- Modify: `cmd/internal/launcher/createlogic.go`
- Modify: `cmd/internal/launcher/createlogic_test.go`
- Modify: `cmd/internal/launcher/runcli.go`
- Modify: `cmd/internal/launcher/runcli_test.go`

- [ ] **Step 1: Write RED orchestration tests for reading defaults and persisting only after readiness**

Named cases:

- `bare-agent-uses-repo-default`: no usable tag config → default args launch;
- `explicit-replaces-after-ready`: default file unchanged before ready, replaced after ready;
- `explicit-empty-clears-after-ready`: writes `args: []` after ready;
- `timeout-does-not-persist`: launch stops and old default remains;
- `tag-config-resumes-automatically`: no fzf config picker, valid native session resumes;
- `stale-session-uses-config-args-fresh`: warning plus no resume token;
- `malformed-config-falls-to-default`: warning plus default args;
- `default-write-failure-after-ready`: target remains current, warning returned, no false launch failure.

Run: `go test ./cmd/internal/launcher -run 'TestRunCreate(AgentDefault|AutomaticConfig|Readiness)' -count=1`

Expected: FAIL because defaults are not on the runtime and create still opens the config picker.

- [ ] **Step 2: Add focused `AgentDefaultOps` to `Runtime` and OS/fake implementations**

```go
type AgentDefaultOps interface {
    ReadAgentDefault(agent string) AgentDefaultCandidate
    WriteAgentDefault(agent string, args []string) error
}
```

OS methods use `ScopedPaths.AgentDefault`, the M1 codec, and atomic writes. The candidate retains malformed/mismatch errors for pure warnings.

- [ ] **Step 3: Replace `runConfigPicker` with M1 `DecideLaunchArgs` in `runCreate`**

Gather saved config (including parse error and native artifact result), repo default, and explicit intent once. Print decision warnings, derive the final args, set a fresh launch nonce in `PAIR_LAUNCH_NONCE`, and call `StartSession`; `StartSession` remains the sole stale-ready cleanup owner.

- [ ] **Step 4: Gate side effects at readiness**

After `WaitReady` succeeds, atomically persist explicit defaults and then call `Wait` to retain the current blocking user experience. On readiness failure, call `Stop`, wait for reaping, and return exit 1. Keep tag ledger/config behavior intact; only the repo-agent default is readiness-gated in M2.

- [ ] **Step 5: Delete the obsolete interactive config-choice surface only after replacement tests pass**

Remove `runConfigPicker`, `configChoice`, `buildConfigChoices`, `selectAction`, `composeTagRestartArgs`, the old `LaunchSession` method, and their choice/blocking-create-only tests. Retain `savedConfig`, JSON codec, config migration, blocking `AttachSession`, `persistedConfigArgs`, and resume composition.

- [ ] **Step 6: Run focused and full suites**

Run: `go test ./cmd/internal/launcher -run 'TestRunCreate|TestDecideLaunchArgs|TestParseLaunchArgs' -count=1`

Expected: PASS.

Run: `go test ./...`

Expected: PASS.

- [ ] **Step 7: Commit production default/readiness wiring**

```bash
git add cmd/internal/launcher
git commit -m "#115 M2: apply agent defaults after ready launch" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 8: Verify and close M2

**Files:**
- Modify: `atlas/session-identity.md`
- Modify: `workshop/issues/000115-resurrect-a-session-across-agents.md`
- Modify if review finds a reusable lesson: `workshop/lessons.md`

- [ ] **Step 1: Update atlas and append the issue log**

Document the ready-record identity, readiness definition, automatic arg precedence, and readiness-gated default persistence in `atlas/session-identity.md`. Append M2 evidence/discoveries to the issue. Leave the pre-existing `- [ ] M2 — Add nonce-bound readiness and wire automatic repo-agent defaults.` row unchecked for the close command.

- [ ] **Step 2: Run verification**

Run: `go test ./cmd/internal/wrapcmd ./cmd/internal/launcher -count=1`

Expected: PASS.

Run: `go test ./...`

Expected: PASS.

Run: `git diff --check`

Expected: no output.

- [ ] **Step 3: Commit pre-boundary docs**

```bash
git add atlas/session-identity.md workshop/issues/000115-resurrect-a-session-across-agents.md
git commit -m "#115 M2: map ready launch defaults" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

- [ ] **Step 4: Run the M2 close without masking its exit**

Run `sdlc milestone-close --issue 115 --milestone M2 --verified 'wrap/launcher suites and go test ./... pass; nonce readiness, automatic resume precedence, and readiness-gated defaults are covered' > /tmp/pair-115-m2-close.out 2>&1`; save `$?` as `close_status`, print the file, and require `close_status == 0`.

- [ ] **Step 5: Resolve the M2 review and require a real verdict**

Apply M1's verdict policy to `/tmp/pair-115-m2-close.out`. Fix Critical/Important `FIX-THEN-SHIP` findings and add lessons; stop on `REWORK`. For `not-run`, rerun `sdlc judge milestone-review --base BASE --head HEAD --issue 115 --milestone M2 --agent codex > /tmp/pair-115-m2-rerun.out 2>&1`, require a real verdict, append an issue-log correction, and write replacement lines to `/tmp/pair-115-m2-rerun.trailers`.

- [ ] **Step 6: Reverify and stage every M2 surface**

Run `go test ./cmd/internal/wrapcmd ./cmd/internal/launcher -count=1`, `go test ./...`, and `git diff --check`. Then run:

```bash
git add cmd/internal/launcher cmd/internal/wrapcmd atlas/session-identity.md workshop/lessons.md workshop/issues/000115-resurrect-a-session-across-agents.md
git diff --cached --check
```

Expected: only M2/review-fix files are staged; unrelated user changes remain unstaged.

- [ ] **Step 7: Commit the M2 boundary**

Commit with subject `#115 M2: close readiness milestone`, a why-focused body, the real original trailers from `/tmp/pair-115-m2-close.out` or replacement trailers from `/tmp/pair-115-m2-rerun.trailers`, and `Co-Authored-By: OpenAI Codex <noreply@openai.com>`. Verify `git log -1 --format=%B` contains the real trailers and `git show HEAD:workshop/issues/000115-resurrect-a-session-across-agents.md | rg '^- \[x\] M2'` matches.

## Chunk 3: Crash-safe state-transfer substrate (M3)

### Core concepts

#### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `HandoffState` / `HandoffJournal` | `cmd/internal/launcher/handoff_state.go` | new |
| `RecoveryPlan` | `cmd/internal/launcher/handoff_state.go` | new |
| `QueuePushFrontPlan` | `cmd/internal/queuecmd/queue.go` | new |
| `TranscriptMetadata` | `cmd/internal/launcher/handoff_transcript.go` | new |

- **`HandoffState` / `HandoffJournal`** — versioned durable transaction with monotonic states and state-specific payload validation.
  - **Relationships:** one journal per transaction; one active transaction per locked repo/tag.
  - **DRY rationale:** live handoff and stale-journal startup recovery interpret the same state contract.
  - **Future extensions:** additional post-ready finalization fields can be versioned without changing old recovery.
- **`RecoveryPlan`** — deterministic actions derived from journal state plus current target/source observations.
  - **Relationships:** one incomplete journal produces one ordered recovery plan.
  - **DRY rationale:** error-path recovery and crash recovery cannot diverge.
  - **Future extensions:** operator-selected recovery modes become inputs, not separate orchestration.
- **`QueuePushFrontPlan`** — stable key allocation and inverse for one logical queue insertion.
  - **Relationships:** one plan inserts at most one exact key and never renames existing keys.
  - **DRY rationale:** Neovim's queue action and handoff both call the same Go primitive.
  - **Future extensions:** reserve multiple front keys while preserving order.
- **`TranscriptMetadata`** — immutable provenance describing one published continuation substrate.
  - **Relationships:** one `snapshot-complete` journal points to one bundle manifest.
  - **DRY rationale:** generated prompt, recovery output, and future tools consume the same metadata.
  - **Future extensions:** checksum/compression fields.

#### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `HandoffStoreOps` | `cmd/internal/launcher/runtime.go`, `cmd/internal/launcher/handoff_store.go` | new | `O_EXCL` lock, atomic journal, backups, restore |
| Queue CLI | `cmd/internal/queuecmd/queuecmd.go`, `cmd/pair-go/main.go`, `cmd/internal/dispatcher/dispatcher.go` | new | stdin body to atomic queue push-front |
| Neovim queue adapter | `nvim/init.lua` | modified | `pair queue push-front` invocation |
| `TranscriptOps` | `cmd/internal/launcher/runtime.go`, `cmd/internal/launcher/handoff_transcript.go` | new | quiescent raw/events copy, plain render, directory publication |

- **`HandoffStoreOps`** — atomic lock acquisition and journal/backup filesystem operations; it does not decide recovery.
  - **Injected into:** M4's state-machine driver.
  - **Future extensions:** `pair handoff inspect/recover` tooling.
- **Queue CLI** — the one writer for logical push-front key allocation and atomic body creation.
  - **Injected into:** Neovim via subprocess and handoff via the underlying Go package.
  - **Future extensions:** push-back/remove may migrate only if another Go consumer needs them.
- **Neovim queue adapter** — passes the current queue directory/body and consumes the returned stable key.
  - **Injected into:** existing draft navigation/send flows through the unchanged `queue_push_front` Lua function.
  - **Future extensions:** none unless queue storage moves wholesale.
- **`TranscriptOps`** — owns immutable bundle IO and delegates rendering to existing `scrollbackcmd`.
  - **Injected into:** M4 after source quiescence.
  - **Future extensions:** alternate renderers behind the same manifest.

### Task 9: Specify journal transitions and recovery plans

**Files:**
- Create: `cmd/internal/launcher/handoff_state.go`
- Create: `cmd/internal/launcher/handoff_state_test.go`
- Modify: `cmd/internal/launcher/scoped_paths.go`
- Modify: `cmd/internal/launcher/scoped_paths_test.go`

- [ ] **Step 1: Write RED tests for every legal and illegal transition**

```go
type HandoffState string

const (
    HandoffPrepared HandoffState = "prepared"
    HandoffSourceStopRequested HandoffState = "source-stop-requested"
    HandoffSourceStopped HandoffState = "source-stopped"
    HandoffSnapshotComplete HandoffState = "snapshot-complete"
    HandoffQueueCommitted HandoffState = "queue-committed"
    HandoffInputCommitted HandoffState = "input-committed"
    HandoffTargetReady HandoffState = "target-ready"
    HandoffComplete HandoffState = "complete"
    HandoffRolledBack HandoffState = "rolled-back"
)

type SnapshotManifest struct {
    Draft      InputBackup
    Instruction InputBackup
    Queue      []QueueItemBackup
    InsertedKey string
    Transcript TranscriptMaterial
}

type TranscriptMaterial struct {
    Available bool
    BundlePath, UnavailableReason string
}

type InputBackup struct {
    Present bool
    Size int64
    Path, BackupPath, Digest string
}

type QueueItemBackup struct {
    Size int64
    Key, Path, BackupPath, Digest string
}

type RecoveryLaunch struct {
    Agent, SessionID, Cwd, PublicSession string
    Args []string
    SessionArtifactVerified bool
    Ready ReadyIdentity
}

type RecoveryMaterial struct {
    Available bool
    UnavailableReason, ManualCommand string
    Launch RecoveryLaunch
}

type StopIdentity struct {
    Tag, Agent, PublicSession, LaunchNonce string
    PairWrapPID, AgentPID, NvimPID, OuterLauncherPID int
}

type HandoffJournal struct {
    Version int
    TxnID, Tag, SourceAgent, TargetAgent, Session string
    SourceStop StopIdentity
    SourceRecovery RecoveryMaterial
    TargetLaunch RecoveryLaunch
    State HandoffState
    Snapshot *SnapshotManifest
    ExplicitDefault *AgentDefault
    DefaultPersisted bool
}
```

Require adjacent forward transitions only; `source-stop-requested` is durable before the first stop/marker effect, `snapshot-complete` requires the authoritative post-quiescence manifest, and `target-ready` is the commit point; unknown version/state and missing state payloads fail validation. `rolled-back` is a separate terminal resolution produced only by the recovery finalizer, not by ordinary `Advance`.

Run: `go test ./cmd/internal/launcher -run 'TestHandoffJournal' -count=1`

Expected: FAIL because the journal does not exist.

- [ ] **Step 2: Implement validation, transition, and JSON round-trip purely**

```go
func (j HandoffJournal) Advance(next HandoffState, snapshot *SnapshotManifest) (HandoffJournal, error)
func ParseHandoffJournal(raw string) (HandoffJournal, error)
func BuildHandoffJournal(j HandoffJournal) (string, error)
```

Defensively copy pointer/slice payloads; never mutate the input journal. `prepared` validation requires a source `StopIdentity` matching journal tag/source agent/public session with at least one authoritative process/session evidence field, a complete target launch, plus exactly one source-recovery variant: either `Available=true` with a complete launch, or `Available=false` with a non-empty unavailability reason and concrete manual recovery command. Thus an unrecoverable source is still exactly stoppable. Reject unavailable records carrying a recovery launch. Keep each target/recovery nonce only in its launch's `ReadyIdentity`; validation requires target and recoverable-source ready identities to have non-empty nonces and exactly match the journal tag, corresponding agent, and public session. It also requires each launch's `Agent` and `PublicSession` to match those same journal fields, so teardown and readiness cannot target different launches. `snapshot-complete` requires the draft presence/path/size/digest/backup, a retained fsynced generated-instruction path/size/digest, an exact sorted queue key/body-size/digest/backup manifest, inserted key iff the present draft's size is greater than zero, and exactly one transcript variant: durable bundle path or non-empty confirmed-unavailable reason. Pin absent, present-empty, and present-nonempty draft plus both transcript variants in the pure tests.

- [ ] **Step 3: Write RED recovery-plan tests on both sides of every crash boundary**

```go
type RecoveryObservation struct {
    TargetPresent, TargetReady bool
    SourceRecoveryAvailable bool
    SourceState SourceProcessState
}

type SourceProcessState string

const (
    SourceIntact SourceProcessState = "intact"
    SourcePartiallyStopping SourceProcessState = "partially-stopping"
    SourceQuiescent SourceProcessState = "quiescent"
)

type RecoveryStepKind string

const (
    RecoveryStopTarget RecoveryStepKind = "stop-target"
    RecoveryWaitTargetQuiescent RecoveryStepKind = "wait-target-quiescent"
    RecoveryStopSource RecoveryStepKind = "stop-source"
    RecoveryWaitSourceQuiescent RecoveryStepKind = "wait-source-quiescent"
    RecoveryReconcileQueueInsert RecoveryStepKind = "reconcile-queue-insert"
    RecoveryReconcileDraftReplace RecoveryStepKind = "reconcile-draft-replace"
    RecoveryRestoreInput RecoveryStepKind = "restore-input"
    RecoveryClearHandoffMarker RecoveryStepKind = "clear-handoff-marker"
    RecoveryStartSource RecoveryStepKind = "start-source"
    RecoveryWaitSourceReady RecoveryStepKind = "wait-source-ready"
    RecoveryPersistExplicitDefault RecoveryStepKind = "persist-explicit-default"
    RecoveryFinalizeRollback RecoveryStepKind = "finalize-rollback"
    RecoveryFinalizeForward RecoveryStepKind = "finalize-forward"
    RecoveryReleaseLock RecoveryStepKind = "release-lock"
)

type RecoveryPlan struct { Steps []RecoveryStepKind }

func PlanRecovery(j HandoffJournal, o RecoveryObservation) (RecoveryPlan, error)
```

Expected matrix:

- `prepared`: `[clear-handoff-marker, finalize-rollback, release-lock]`, never stop source;
- `source-stop-requested` + `intact`: `[clear-handoff-marker, finalize-rollback, release-lock]`; never stop or relaunch the unchanged source, regardless of recovery availability;
- `source-stop-requested` + `partially-stopping`: `[stop-source, wait-source-quiescent, clear-handoff-marker, start-source, wait-source-ready, finalize-rollback, release-lock]` when recovery is available, omitting start/wait when unavailable;
- `source-stop-requested` + `quiescent`: `[clear-handoff-marker, start-source, wait-source-ready, finalize-rollback, release-lock]` when recovery is available, omitting start/wait when unavailable;
- `source-stopped`: `[clear-handoff-marker, start-source, wait-source-ready, finalize-rollback, release-lock]` when recovery is available;
- `snapshot-complete`: `[reconcile-queue-insert, clear-handoff-marker, start-source, wait-source-ready, finalize-rollback, release-lock]`; reconciliation removes only an insertion proven to be this transaction's, and no draft mutation has happened yet;
- `queue-committed`: exactly `[stop-target, wait-target-quiescent, reconcile-draft-replace, restore-input, clear-handoff-marker, start-source, wait-source-ready, finalize-rollback, release-lock]`; reconciliation distinguishes unchanged original draft from this transaction's staged instruction inode and refuses any third-party content;
- `input-committed`: exactly `[stop-target, wait-target-quiescent, restore-input, clear-handoff-marker, start-source, wait-source-ready, finalize-rollback, release-lock]`;
- `target-ready` with a pending explicit default: `[persist-explicit-default, finalize-forward, release-lock]`; without one or after its durable persistence marker: `[finalize-forward, release-lock]`; neither variant restores/recovers source;
- `complete` or `rolled-back`: `[release-lock]`; and
- contradiction `TargetReady=true` before durable journal `target-ready`: the pre-commit stop/quiescence/rollback sequence still applies.

Include source start/wait steps only when both the durable `j.SourceRecovery.Available` and current `RecoveryObservation.SourceRecoveryAvailable` are true; contradiction tests cover either side being false. An `intact` source means the exact public session and every authoritative recorded process remain alive, no teardown helper is live, and the observation is stable across a bounded recheck; any missing/changing evidence is `partially-stopping`, never optimistically intact. Otherwise omit start/wait-source but retain required source/target quiescence, queue reconciliation, input restoration, and marker clearing, then durably `finalize-rollback` before releasing the lock. At `target-ready`, `persist-explicit-default` performs the idempotent write and atomically marks `DefaultPersisted=true`; an effect-before-journal crash safely repeats the same value, while a write failure leaves the state/lock at `target-ready` and never runs `finalize-forward`. M4 executes steps strictly in order and refuses `restore-input` unless the preceding target-quiescence observation succeeded. `finalize-rollback` atomically publishes the terminal tombstone only after every applicable reconcile/restore/recovery step succeeds; future scans treat the transaction as resolved.

- [ ] **Step 4: Implement minimal recovery planning and run tests**

Run: `go test ./cmd/internal/launcher -run 'Test(HandoffJournal|PlanRecovery)' -count=1`

Expected: PASS.

- [ ] **Step 5: Add scoped paths for lock, transaction root, and journal**

Pin `handoff-<tag>.lock`, `handoff-<tag>-<txn>/journal.json`, and transaction-local backup paths in `scoped_paths_test.go`.

- [ ] **Step 6: Commit the transaction state model**

```bash
git add cmd/internal/launcher/handoff_state.go cmd/internal/launcher/handoff_state_test.go cmd/internal/launcher/scoped_paths.go cmd/internal/launcher/scoped_paths_test.go
git commit -m "#115 M3: define durable handoff states" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 10: Implement exclusive lock and durable journal storage

**Files:**
- Create: `cmd/internal/launcher/handoff_store.go`
- Create: `cmd/internal/launcher/handoff_store_test.go`
- Modify: `cmd/internal/launcher/runtime.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/createflow_test.go`

- [ ] **Step 1: Write RED OS-backed store tests in a temp repo scope**

Cover:

- two concurrent `AcquireTagLock` calls with complete records: exactly one succeeds;
- a live owner rejects a competitor and reports PID/transaction;
- a dead-owner lock returns a stale-recovery result rather than silently overwriting;
- two processes racing to claim that same dead-owner record: exactly one `O_EXCL` recovery sidecar wins, the winner re-reads and compare-validates the original tag/transaction/owner before effects, and the loser cannot recover or acquire the ordinary tag lock; a crashed recovery claimant is itself reclaimed by one atomic rename-to-quarantine winner before a new `O_EXCL` claim, preventing a permanent lockout;
- lock JSON contains version, tag, transaction ID, owner PID, source public session, and RFC3339 start time; malformed/partial records refuse automatic takeover and print their path for manual inspection;
- journal writes are sibling-temp + rename and readers see old-or-new complete JSON;
- state regressions/invalid payloads are rejected before write;
- authoritative snapshot backup copies draft plus exact queue key/body/digest manifest;
- idempotent reconciliation/restore uses `os.SameFile` against retained transaction inodes: `snapshot-complete` removes an owned effect-before-journal queue insertion and refuses an unowned collision; `queue-committed` removes that key and restores the draft iff it is the staged instruction inode (leaves an original-digest draft alone, refuses any third state); `input-committed` removes/restores directly, always after target quiescence;
- release removes only a lock with the expected transaction ID.

Run: `go test ./cmd/internal/launcher -run 'TestOSHandOffStore' -count=1`

Expected: FAIL because no store exists.

- [ ] **Step 2: Add a focused runtime seam**

```go
type HandoffStoreOps interface {
    AcquireTagLock(TagLockRecord) (TagLockResult, error)
    ClaimStaleTagLock(TagLockRecord, RecoveryClaim) error
    ReleaseRecoveryClaim(tag, txnID, claimID string) error
    FindUnresolvedHandoff(tag string) (HandoffJournal, bool, error)
    ReadHandoffJournal(tag, txnID string) (HandoffJournal, error)
    WriteHandoffJournal(HandoffJournal) error
    BackupHandoffInput(tag, txnID string, keys []string, replacementBody string) (SnapshotBackup, error)
    CommitHandoffQueue(HandoffJournal) error
    ReconcileHandoffQueue(HandoffJournal) (QueueCommitObservation, error)
    CommitHandoffDraft(HandoffJournal) error
    ReconcileHandoffDraft(HandoffJournal) (DraftCommitObservation, error)
    MarkHandoffDefaultPersisted(HandoffJournal) error
    RestoreHandoffInput(HandoffJournal) error
    FinalizeHandoffRollback(HandoffJournal) error
    ReleaseTagLock(tag, txnID string) error
}

type QueueCommitObservation string

const (
    QueueInsertAbsent QueueCommitObservation = "absent"
    QueueInsertedByTransaction QueueCommitObservation = "inserted-by-transaction"
    QueueInsertConflict QueueCommitObservation = "conflict"
)

type DraftCommitObservation string

const (
    DraftStillOriginal DraftCommitObservation = "original"
    DraftInstalledByTransaction DraftCommitObservation = "installed-by-transaction"
    DraftCommitConflict DraftCommitObservation = "conflict"
)

type RecoveryClaim struct {
    ClaimID string
    OwnerPID int
    StartedAt time.Time
}

type SnapshotBackup struct {
    Draft InputBackup
    Instruction InputBackup
    Queue []QueueItemBackup
}

type TagLockRecord struct {
    Version int
    Tag, TxnID, SourceSession string
    OwnerPID int
    StartedAt time.Time
}
```

Use `procutil.Alive` for owner liveness. A stale recovery claimant first creates `handoff-<tag>.recovery.lock` with `O_EXCL`, then re-reads the ordinary lock and proceeds only if its complete record matches the observed dead owner; ordinary acquisition refuses while that recovery claim exists. If the recovery-claim owner is dead, contenders re-read the complete claim and atomically rename the one canonical claim path to a claim-ID quarantine path; only the rename winner retries `O_EXCL`, and quarantines remain forensic. Release active locks only by matching transaction/claim IDs. `CommitHandoffQueue` accepts only `snapshot-complete` and delegates the exact retained-inode hard link to `queuecmd.PushFrontPlanned`; `ReconcileHandoffQueue` returns absent, `os.SameFile`-proven inserted, or conflict. `CommitHandoffDraft` accepts only `queue-committed`, verifies the current draft is still the snapshot's original absence/digest, hard-links the retained instruction inode to a sibling temp, renames it over the draft, and fsyncs the draft directory. `ReconcileHandoffDraft` returns original, `os.SameFile`-proven installed, or conflict; rollback restores only the installed variant and refuses conflict. `MarkHandoffDefaultPersisted` accepts only `target-ready`, requires an explicit default, and atomically rewrites that same state with `DefaultPersisted=true`; repeat writes are idempotent. `FinalizeHandoffRollback` validates a pre-commit state, atomically writes a `rolled-back` terminal tombstone, and is idempotent for the same transaction; stale-journal scans ignore terminal `complete`/`rolled-back` transactions. For every durable write: fsync file contents before rename/link, fsync the containing transaction directory after publication, and fsync its parent after creating/renaming transaction directories. Keep lock/store mechanics in `handoff_store.go`, not the already-large `osruntime.go`; OSRuntime delegates.

- [ ] **Step 3: Implement storage minimally and run concurrency/race tests**

Before implementation, add RED cases for `CommitHandoffQueue`/`ReconcileHandoffQueue`: wrong-state rejection, exact retained-inode publication, absent/owned/conflict observations, fsync, and crash-after-link reconciliation without real IO in coordinator tests. Add `CommitHandoffDraft`/`ReconcileHandoffDraft` cases for wrong-state rejection, absent/original draft success, changed-original conflict before rename, installed-inode recognition, same-content/different-inode conflict, no torn reader-visible body, fsynced publication, and idempotent recovery after an effect-before-journal crash. Add `MarkHandoffDefaultPersisted` cases for wrong state, no-default rejection, idempotence, and old-or-new durable `target-ready` visibility. Add `FinalizeHandoffRollback` cases for valid finalization from each pre-commit state, rejection at/after `target-ready`, idempotent replay for the same transaction, old-or-new durable tombstone visibility, and `FindUnresolvedHandoff` ignoring `complete`/`rolled-back` while returning a nonterminal transaction. Assert the returned `SnapshotBackup` contains exact draft/instruction/queue size, digest, path, backup, and retained-inode fields. Race `ClaimStaleTagLock` callers and prove exactly one recovery owner plus transaction/owner mismatch rejection; then kill that claimant and prove exactly one contender quarantines/replaces it while all others lose.

Run: `go test ./cmd/internal/launcher -run 'TestOSHandOffStore' -race -count=1`

Expected: PASS; exactly one lock winner and no torn journals.

- [ ] **Step 4: Extend fakeRuntime with deterministic lock/journal/backup behavior**

Record calls and inject failure by state (`failWriteState`) so M4 can test every boundary without filesystem mocks.

- [ ] **Step 5: Run the launcher package and commit**

Run: `go test ./cmd/internal/launcher -count=1`

Expected: PASS.

```bash
git add cmd/internal/launcher/handoff_store.go cmd/internal/launcher/handoff_store_test.go cmd/internal/launcher/runtime.go cmd/internal/launcher/osruntime.go cmd/internal/launcher/createflow_test.go
git commit -m "#115 M3: add exclusive handoff journal store" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 11: Share stable queue push-front between Neovim and handoff

**Files:**
- Create: `cmd/internal/queuecmd/queue.go`
- Create: `cmd/internal/queuecmd/queue_test.go`
- Create: `cmd/internal/queuecmd/queuecmd.go`
- Create: `cmd/internal/queuecmd/queuecmd_test.go`
- Modify: `cmd/internal/dispatcher/dispatcher.go`
- Modify: `cmd/internal/dispatcher/dispatcher_test.go`
- Modify: `cmd/pair-go/main.go`
- Modify: `cmd/pair-go/main_test.go`
- Modify: `nvim/init.lua`
- Modify (generated): `cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua`
- Modify (generated): `cmd/internal/runtimebundle/assets/runtime/manifest.json`
- Modify: `tests/queue-send-test.sh`
- Modify: `Makefile.local`

- [ ] **Step 1: Write RED pure tests for key allocation and inversion**

```go
type QueuePushFrontPlan struct {
    Key, Path, Body, RetainedSourcePath string
}

func PlanPushFront(dir string, sortedKeys []string, body string) (QueuePushFrontPlan, error)
```

Pin empty → `500000`, existing `500000` → `499999`, ordering with gaps, invalid/non-six-digit inputs ignored by scanner, zero-key exhaustion error, and preservation of all existing keys.

Run: `go test ./cmd/internal/queuecmd -run 'TestPlanPushFront' -count=1`

Expected: FAIL because the package does not exist.

- [ ] **Step 2: Implement complete-body atomic `PushFront` with collision retry**

Write the complete body to a sibling temp file, fsync and close it, then publish without replacement by hard-linking the temp inode to the final key and unlinking the temp. If the link reports `EEXIST`, rescan/replan/retry; never overwrite or rekey an existing item. Fsync the queue directory after publication. Return the committed key on stdout. Readers can observe only absent or complete bodies.

Expose `PushFrontPlanned(QueuePushFrontPlan)` for the handoff transaction: it hard-links the retained, fsynced transaction draft-backup inode to exactly the journaled key once and returns a typed collision without replanning or unlinking the retained source. The ordinary `PushFront`/CLI owns its temporary file and rescan-and-retry loop. Tests prove a planned collision creates nothing, a success is `os.SameFile` with the retained source, and a simulated crash can prove/remove only its own published inode.

- [ ] **Step 3: Add `pair queue push-front <dir>` streaming CLI tests**

Read body from stdin, reject empty body and extra args, create the directory lazily, and print exactly `<key>\n`. Register `queue push-front` as implemented/streaming and route it in `cmd/pair-go/main.go`.

Run: `go test ./cmd/internal/queuecmd ./cmd/internal/dispatcher ./cmd/pair-go -count=1`

Expected: PASS after minimal dispatcher wiring.

- [ ] **Step 4: Replace Lua key allocation/write inside `queue_push_front` with the CLI**

Keep the Lua function signature and callers unchanged. Resolve the executable as `$PAIR_HOME/bin/pair`, pass `body` on stdin via `vim.fn.system`, trim/validate the six-digit returned key, and raise a visible error without changing navigation on nonzero exit. Do not retain a Lua allocation fallback: that would recreate two sources of truth.

- [ ] **Step 5: Extend queue integration tests**

Build `bin/pair` before `test-queue`, set `PAIR_HOME` to the checkout in the headless test, and assert Alt+q/front-push still orders items, concurrent CLI pushes produce two distinct ordered files, delimiter/body bytes round-trip, and a CLI error leaves the prior draft/queue untouched. The test must not resolve a stale installed binary.

Run: `make test-queue`

Expected: PASS.

- [ ] **Step 6: Regenerate and verify the embedded runtime**

Run: `make runtimebundle-generate && make test-runtimebundle && make runtimebundle-drift-check`

Expected: PASS; the embedded Neovim file contains the same queue adapter as `nvim/init.lua` and the manifest digest is current.

- [ ] **Step 7: Commit the shared queue primitive**

```bash
git add cmd/internal/queuecmd cmd/internal/dispatcher/dispatcher.go cmd/internal/dispatcher/dispatcher_test.go cmd/pair-go/main.go cmd/pair-go/main_test.go nvim/init.lua cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua cmd/internal/runtimebundle/assets/runtime/manifest.json tests/queue-send-test.sh Makefile.local
git commit -m "#115 M3: share atomic queue push-front" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 12: Publish immutable continuation transcript bundles

**Files:**
- Create: `cmd/internal/launcher/handoff_transcript.go`
- Create: `cmd/internal/launcher/handoff_transcript_test.go`
- Modify: `cmd/internal/launcher/runtime.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/createflow_test.go`

- [ ] **Step 1: Write RED metadata/path tests**

```go
type TranscriptMetadata struct {
    Tag, Agent, SessionID, PublicSession, Cutoff, TransactionID string
}

type TranscriptBundle struct {
    Dir, PlainPath, RawPath, EventsPath, MetadataPath string
    Metadata TranscriptMetadata
}

type TranscriptRequest struct {
    RawPath, EventsPath, ParkedRoot string
    Tag, Agent, NativeSessionID, PublicSession, TransactionID string
    Cutoff time.Time
}
```

Pin `parked/<tag>/<timestamp>-<agent>-<txn>/`, safe normalized components, required source paths/provenance, exact RFC3339 cutoff preservation, and collision resistance from transaction ID rather than timestamp alone.

- [ ] **Step 2: Write RED OS-backed publication tests**

Feed real small raw/events fixtures through `scrollbackcmd.Run --plain`; assert rendering reads only the copied immutable inputs, the published directory contains `transcript.txt`, `scrollback.raw`, `events.jsonl`, and `metadata.json`, metadata preserves the request's exact cutoff, readers see no final directory before rename, an existing final dir is never overwritten, and source files remain unchanged.

Run: `go test ./cmd/internal/launcher -run 'TestTranscriptBundle' -count=1`

Expected: FAIL because transcript bundling does not exist.

- [ ] **Step 3: Add `TranscriptOps` and implement temp-directory publication**

```go
type TranscriptOps interface {
    PublishTranscriptBundle(TranscriptRequest) (TranscriptBundle, error)
}
```

Copy quiescent raw/events into a sibling temp directory, render the copied inputs to plain text through the existing renderer, write metadata atomically, fsync/close every file, fsync the temp directory, rename the directory, then fsync the parked parent directory. Remove temp output on failure; never mutate source scrollback. Do not advance the journal to `snapshot-complete` until this durable publication and the durable input backups both succeed.

- [ ] **Step 4: Extend fakeRuntime and run tests**

Add scripted bundles/errors for each M4 snapshot boundary.

Run: `go test ./cmd/internal/launcher -run 'TestTranscriptBundle' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit transcript publication**

```bash
git add cmd/internal/launcher/handoff_transcript.go cmd/internal/launcher/handoff_transcript_test.go cmd/internal/launcher/runtime.go cmd/internal/launcher/osruntime.go cmd/internal/launcher/createflow_test.go
git commit -m "#115 M3: publish immutable handoff transcripts" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 13: Verify and close M3

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `atlas/session-identity.md`
- Modify: `workshop/issues/000115-resurrect-a-session-across-agents.md`
- Modify if review finds a reusable lesson: `workshop/lessons.md`

- [ ] **Step 1: Document the substrate and append issue evidence**

Map the lock/journal states, queue push-front ownership, and transcript bundle layout. Append M3 discoveries/evidence. Leave `- [ ] M3 — Add the crash-safe lock/journal, shared queue push-front, and immutable transcript bundle.` unchecked for `milestone-close`.

- [ ] **Step 2: Run verification**

Run: `go test ./cmd/internal/launcher ./cmd/internal/queuecmd ./cmd/internal/wrapcmd -race -count=1`

Expected: PASS.

Run: `make test-queue`

Expected: PASS.

Run: `make test-runtimebundle && make runtimebundle-drift-check`

Expected: PASS.

Run: `go test ./...`

Expected: PASS.

Run: `git diff --check`

Expected: no output.

- [ ] **Step 3: Commit pre-boundary docs**

```bash
git add atlas/architecture.md atlas/session-identity.md workshop/issues/000115-resurrect-a-session-across-agents.md
git commit -m "#115 M3: map crash-safe handoff state" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

- [ ] **Step 4: Close and review M3**

Run the fully spelled M2 boundary protocol with milestone `M3`, `/tmp/pair-115-m3-*` files, verification `race/unit suites, make test-queue, runtimebundle checks, and go test ./... pass; lock/journal recovery, shared queue push-front, and immutable transcript publication are covered`, staging `cmd/internal/launcher`, `cmd/internal/queuecmd`, dispatcher/pair-go, `nvim/init.lua`, generated runtimebundle assets, `tests/queue-send-test.sh`, `Makefile.local`, atlas, lessons, and issue #115. Commit with subject `#115 M3: close handoff substrate milestone`, a real review verdict/window, and the co-author trailer.

## Chunk 4: Exclusive handoff orchestration and acceptance (M4)

### Core concepts

#### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ResolvedDriver` / `HandoffPreflight` | `cmd/internal/launcher/handoff_preflight.go` | new |
| `HandoffConfirmation` | `cmd/internal/launcher/handoff_confirm.go` | new |
| `HandoffInstruction` | `cmd/internal/launcher/handoff_instruction.go` | new |
| `TagOperation` / `TagLockPlan` | `cmd/internal/launcher/tag_operation.go` | new |

- **`ResolvedDriver` / `HandoffPreflight`** — immutable revalidated source evidence, transcript inputs, exact source recovery material, and exact target launch selected under the tag lock.
  - **Relationships:** one `ActionHandoff` selection is re-resolved to one preflight; conflicting evidence produces no preflight.
  - **DRY rationale:** confirmation text and transaction execution consume the same post-lock facts rather than re-reading mutable files independently.
  - **Future extensions:** additional driver providers add evidence adapters without changing orchestration.
- **`HandoffConfirmation`** — structured exclusive-switch disclosure with explicit recovery-unavailable, transcript-unavailable, and post-render-failure variants.
  - **Relationships:** one preflight produces the ordinary confirmation; unavailable recovery/transcript requires distinct disclosure, and a new render failure after quiescence requires a new affirmative result before tag-state-only continuation.
  - **DRY rationale:** terminal UI wording and tests share one rendering contract.
  - **Future extensions:** noninteractive policy can consume the same disclosure object.
- **`HandoffInstruction`** — exact generated draft body that points at the immutable source bundle and dead-agent continuation procedure, or explicitly reports a confirmed transcript-unavailable handoff.
  - **Relationships:** one `TranscriptMaterial` produces one instruction; it names source provenance while the target pane keeps the shared tag identity.
  - **DRY rationale:** repeated handoffs cannot drift into agent-specific prompt variants.
  - **Future extensions:** datatype version and extra bundle checksums can be added as fields.
- **`TagOperation` / `TagLockPlan`** — lock lifetime and recovery policy for attach, create, rename, and handoff.
  - **Relationships:** every mutating/attaching entry has one operation policy; rename locks both tags in lexical order.
  - **DRY rationale:** exclusivity is enforced at one launcher boundary instead of scattered handoff-only checks.
  - **Future extensions:** future tag-mutating commands must add one enum case and test.

#### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `DriverOps` | `cmd/internal/launcher/runtime.go`, `cmd/internal/launcher/handoff_driver_os.go` | new | live sessions, agent files, session index, ledgers, transcript/config evidence |
| `HandoffUIOps` | `cmd/internal/launcher/runtime.go`, `cmd/internal/launcher/osruntime.go` | new | confirmation and irreversible second confirmation |
| `SessionControlOps` | `cmd/internal/launcher/runtime.go`, `cmd/internal/launcher/handoff_process_os.go` | new | marker/ack, journal-identified source or target teardown, process quiescence |
| `HandoffCoordinator` | `cmd/internal/launcher/handoff.go` | new | store, queue, transcript, session launch/control, defaults |

- **`DriverOps`** — gathers provenance records and resolves exact config/transcript paths; classification remains in M1's pure core.
  - **Injected into:** picker row construction and locked preflight revalidation.
  - **Future extensions:** another agent's native-session locator remains behind `AgentSessionExists`.
- **`HandoffUIOps`** — renders `[y/N]` prompts on the controlling TTY and distinguishes decline/dismiss from IO failure.
  - **Injected into:** the post-lock preflight before the source is changed.
  - **Future extensions:** a future `--yes` flag would implement this seam explicitly.
- **`SessionControlOps`** — marks the source as handoff-owned, deletes a journal-identified source or target public session, and proves pair-wrap, agent, Neovim, Zellij server, and attached launcher child are gone without requiring an in-memory handle.
  - **Injected into:** live failure handling and dead-owner recovery; both use `RecoveryLaunch` nonce/session plus durable ready/PID evidence.
  - **Future extensions:** backend-specific quiescence probes.
- **`HandoffCoordinator`** — the sole ordered executor of the M3 journal and recovery plans.
  - **Injected into:** `runOnce` for `ActionHandoff` and startup stale-journal recovery.
  - **Future extensions:** an inspect/recover CLI can invoke the same coordinator.

### Task 14: Revalidate drivers and render the user contract

**Files:**
- Create: `cmd/internal/launcher/handoff_preflight.go`
- Create: `cmd/internal/launcher/handoff_preflight_test.go`
- Create: `cmd/internal/launcher/handoff_confirm.go`
- Create: `cmd/internal/launcher/handoff_confirm_test.go`
- Create: `cmd/internal/launcher/handoff_instruction.go`
- Create: `cmd/internal/launcher/handoff_instruction_test.go`
- Create: `cmd/internal/launcher/handoff_driver_os.go`
- Create: `cmd/internal/launcher/handoff_driver_os_test.go`
- Modify: `cmd/internal/launcher/runtime.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/createflow_test.go`

- [ ] **Step 1: Write RED evidence-gathering and locked-preflight tests**

Cover live attached/detached session plus matching agent-file authority, ledger fallback for exited/recent-inactive work, unknown historical driver, missing/partial transcript inputs with exact reasons, config agent mismatch, stale native session, recoverable fresh source launch, and every conflict shape. Require preflight to carry copied evidence, a transcript-availability variant with exact raw/events paths or reason, `RecoveryMaterial`, target `RecoveryLaunch`, and shared tag/public session.

Run: `go test ./cmd/internal/launcher -run 'Test(ResolveHandoffPreflight|OSDriverEvidence)' -count=1`

Expected: FAIL because the gatherer/preflight do not exist.

- [ ] **Step 2: Implement post-lock evidence gathering and preflight without mutation**

Gather session index, live Zellij rows, `agent-<tag>`, and ledger/config records into M1's `DriverEvidence`; do not choose provenance in OS code. Re-run `ClassifyDriver` after the lock, require the selected tag/public session still matches, and compute source recovery through the same M1 launch-arg policy/native-artifact checks used by ordinary create. Unknown historical sources may produce explicit unavailable recovery and transcript records; live conflicting sources always refuse.

- [ ] **Step 3: Write RED confirmation tests**

Pin ordinary text naming tag/source/target and the four effects (preserve tag state, park source transcript, close source session, start target). Pin decline and dismissal as zero-mutation aborts. When `RecoveryMaterial.Available=false`, require a separately rendered irreversible prompt naming the reason/manual command. When transcript input is already unavailable, disclose exactly what is missing and require a separate affirmative tag-state-only choice before stop. Model a third `TranscriptRenderFailed` confirmation which can occur only after quiescence and names the render error; prior confirmation never authorizes it. Any required affirmative missing is an abort/rollback.

Run: `go test ./cmd/internal/launcher -run 'TestHandoffConfirmation' -count=1`

Expected: FAIL until `HandoffConfirmation` and the TTY seam exist.

- [ ] **Step 4: Write RED generated-instruction tests and implement the exact template**

For available transcripts, the body must identify the shared work tag, source agent, source native session ID when present, source public session, and absolute immutable `transcript.txt`. It must say to follow the continuation datatype's dead-agent procedure, draft and preview the continuation for approval, finalize the approved body through `pair continuation --no-restart`, then continue the work in the current target session. For confirmed unavailable transcripts, build a distinct instruction naming the reason and telling the target to continue from tag draft/history/queue without claiming a continuation exists. Assert neither variant tells the target to infer the source from `PAIR_TAG` or constructs an executable shell command from untrusted metadata.

Run: `go test ./cmd/internal/launcher -run 'TestBuildHandoffInstruction' -count=1`

Expected: PASS after minimal pure implementation.

- [ ] **Step 5: Commit preflight and user contract**

```bash
git add cmd/internal/launcher/handoff_preflight.go cmd/internal/launcher/handoff_preflight_test.go cmd/internal/launcher/handoff_confirm.go cmd/internal/launcher/handoff_confirm_test.go cmd/internal/launcher/handoff_instruction.go cmd/internal/launcher/handoff_instruction_test.go cmd/internal/launcher/handoff_driver_os.go cmd/internal/launcher/handoff_driver_os_test.go cmd/internal/launcher/runtime.go cmd/internal/launcher/osruntime.go cmd/internal/launcher/createflow_test.go
git commit -m "#115 M4: define exclusive handoff preflight" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 15: Execute and recover the journaled handoff

**Files:**
- Create: `cmd/internal/launcher/handoff.go`
- Create: `cmd/internal/launcher/handoff_test.go`
- Create: `cmd/internal/launcher/handoff_process_os.go`
- Create: `cmd/internal/launcher/handoff_process_os_test.go`
- Modify: `cmd/internal/launcher/runtime.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/lifecycle.go`
- Modify: `cmd/internal/launcher/lifecycle_test.go`
- Modify: `cmd/internal/launcher/createflow_test.go`

- [ ] **Step 1: Write RED state-machine tests with failure injection at every durable/effect boundary**

For success, assert this exact order: acquire/revalidate/confirm → write `prepared` → write `source-stop-requested` → set transaction handoff marker → stop source → receive marker-consumed cleanup acknowledgement and prove source quiescence → write `source-stopped` → publish bundle (or obtain the new post-render-failure affirmative), build the instruction, and durably retain input backups plus staged instruction → write `snapshot-complete` → hard-link the retained draft-backup inode to the allocated front key → write `queue-committed` → atomically install `*` from the retained instruction inode → write `input-committed` → start exact target → accept matching readiness → write `target-ready` → persist/clear explicit target default when requested → write `complete` → release lock → wait on the attached target child. For an empty draft, advance through `queue-committed` without creating a queue item so the transition sequence stays uniform.

Inject a crash/error before and after every external effect and every journal write, including: immediately after `source-stop-requested` before marker; after marker with a bounded-stable intact source; during partial teardown; after full quiescence before `source-stopped`; queue-link-before-`queue-committed`; draft-replace-before-`input-committed`; target-start-before-ready; ready-before-`target-ready`; default-write-before-`DefaultPersisted`; and default-marker-before-`complete`. `prepared` leaves source/input unchanged. `source-stop-requested` + intact clears marker and rolls back without stop/relaunch—even when recovery is unavailable; partial teardown completes quiescence and relaunches when recoverable; already-quiescent relaunches directly. At `snapshot-complete`, reconcile the exact key through the injected store: absent is clean, `os.SameFile` with the retained backup is this transaction's effect and is removed, and any other inode is a collision requiring operator resolution. At `queue-committed`, similarly distinguish the unchanged original draft from the retained instruction inode; restore the latter and refuse unrelated content. Later pre-commit states stop any partial target through journal-only identity, prove quiescence, restore only journal-owned input, clear the marker, recover source when available, write `rolled-back`, and release. At/after durable `target-ready`, never stop target/restore source: retry the same explicit default until `DefaultPersisted=true`, then finalize forward. Default-write failure warns and leaves a retryable `target-ready` journal/lock with target live; fake-runtime tests inject it without real queue/filesystem IO.

Run: `go test ./cmd/internal/launcher -run 'TestHandoff(Coordinator|Recovery)' -count=1`

Expected: FAIL because orchestration does not exist.

- [ ] **Step 2: Implement exact source handoff mode and OS quiescence**

After durable `source-stop-requested`, atomically publish a transaction-bound handoff marker before deleting the source. Teach `runCleanup` to consume only its matching marker, skip ordinary quit parking/removal for that source return path, and atomically write a transaction cleanup acknowledgement; unrelated quit/restart behavior remains unchanged. `ObserveJournaledSource(StopIdentity)` returns `intact` only after two bounded observations show the exact session and all authoritative PIDs alive, no cleanup acknowledgement, and no teardown helper/process; any missing/changing evidence is `partially-stopping`, while total absence plus acknowledgement/no-outer-launcher proof is `quiescent`. `StopJournaledLaunch` deletes only the journaled public session, discovers matching ready/PID artifacts by nonce/tag/agent/session, and waits with a bound until Zellij no longer lists it and the recorded pair-wrap/agent process tree plus tag Neovim and attached launcher child are dead. Both work after process restart with no `SessionLaunch` handle. Source quiescence requires cleanup acknowledgement or proof no matching outer launcher remains, after which the coordinator invalidates the exact marker. Rollback always clears/invalidates marker+ack before source relaunch/finalization; if stopping never began and the source is intact, rollback clears the marker without deleting it. Preserve diagnostic PIDs/evidence in errors. A timeout leaves the lock/journal for recovery and never restores input while writers may live.

Add OS-backed tests with fake Zellij/pair-wrap/agent/Neovim/outer-launcher processes proving a user-TTY-attached source is torn down, a different session survives, marker mismatch does not suppress cleanup, acknowledgement is transaction-bound, before-marker and marker-only crashes classify bounded-stable live evidence as intact, a disappearing PID/helper classifies partial rather than intact, rollback clears a marker with source still live, journal-only target teardown works after discarding the in-memory handle, and quiescence does not succeed on a Zellij-row-only disappearance.

Run: `go test ./cmd/internal/launcher -run 'Test(OSHandoffProcess|RunCleanupHandoff)' -count=1`

Expected: PASS.

- [ ] **Step 3: Implement the coordinator by interpreting pure plans**

Keep state decisions in `handoff_state.go`; `handoff.go` performs only the next planned effect then durably advances. Invoke injected `CommitHandoffQueue`, `PublishTranscriptBundle`, `CommitHandoffDraft`, `StartSession`, `StopJournaledLaunch`, and `AgentDefaultOps`; do not call `RunLaunch` recursively, call `queuecmd` directly, or perform filesystem mutation directly. Pass the immutable cutoff captured immediately after source quiescence to transcript publication. A render error invokes the new transcript-failure confirmation: decline rolls back, affirm records `TranscriptMaterial{Available:false}` and builds the tag-state-only instruction. Queue insertion must create the exact key already journaled by `snapshot-complete`; retain both draft-backup and instruction inodes through terminal resolution so crash recovery can prove ownership of either effect. A typed pre-link collision aborts without removal; an owned link found after restart is removed during reconciliation. Durable `queue-committed` authorizes the store's owned-key removal and `ReconcileHandoffDraft`; `input-committed` records both mutations complete. After matching target readiness, write `target-ready`, then persist any explicit default and call `MarkHandoffDefaultPersisted` before `complete`; failure in either write stops forward finalization but never rolls back the ready target.

- [ ] **Step 4: Test dead-owner recovery on launcher entry**

For every journal state, simulate a dead lock owner and call the coordinator's `RecoverTag`. First require `ClaimStaleTagLock` to win the atomic recovery sidecar and revalidate the unchanged dead-owner record; a losing racer performs no journal/process/file effect. Assert terminal journals only release matching stale/recovery locks, pre-commit journals follow the ordered rollback/reconciliation plan, ready journals finalize forward, target teardown uses only journal+filesystem evidence, malformed lock/journal files refuse with their exact paths, and a live owner rejects without mutation. Recovery failure keeps the journal/lock/claim and prints the exact remaining processes plus manual command.

Run: `go test ./cmd/internal/launcher -run 'TestRecoverTag' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit transaction orchestration**

```bash
git add cmd/internal/launcher/handoff.go cmd/internal/launcher/handoff_test.go cmd/internal/launcher/handoff_process_os.go cmd/internal/launcher/handoff_process_os_test.go cmd/internal/launcher/runtime.go cmd/internal/launcher/osruntime.go cmd/internal/launcher/lifecycle.go cmd/internal/launcher/lifecycle_test.go cmd/internal/launcher/createflow_test.go
git commit -m "#115 M4: orchestrate crash-safe agent handoff" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 16: Wire the normal picker and lock every tag operation

**Files:**
- Create: `cmd/internal/launcher/tag_operation.go`
- Create: `cmd/internal/launcher/tag_operation_test.go`
- Modify: `cmd/internal/launcher/createflow.go`
- Modify: `cmd/internal/launcher/createflow_test.go`
- Modify: `cmd/internal/launcher/pick.go`
- Modify: `cmd/internal/launcher/pick_test.go`
- Modify: `cmd/internal/launcher/rename.go`
- Modify: `cmd/internal/launcher/rename_test.go`
- Modify: `cmd/internal/launcher/runtime.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/help.go`
- Modify: `cmd/internal/launcher/help_test.go`

- [ ] **Step 1: Write RED normal-picker behavior tests**

Pin bare `pair` excluding attached rows, explicit `pair codex` including attached/recent rows inside the same repo/history cutoff, exact driver/state labels, same-agent attach/resume without confirmation, different-agent `ActionHandoff`, unknown historical driver disclosure, expired history exclusion, and conflict-disabled rows listing every evidence source. Ensure an existing attached session no longer suppresses the explicit-agent picker.

Run: `go test ./cmd/internal/launcher -run 'Test(ExplicitAgentPicker|AgentPickerPolicy|RunHandoffSelection)' -count=1`

Expected: FAIL because the pure rows are not wired to fzf/createflow.

- [ ] **Step 2: Acquire after selection, recover, and revalidate before acting**

For attach/resume, acquire the selected tag lock; if it reports a dead owner, win the atomic recovery claim and compare-revalidate before recovering, then acquire normally, re-snapshot/re-decide, and release before blocking in `AttachSession`. For create, acquire the final normalized tag after the name prompt and hold through matching readiness/default persistence, then release before `SessionLaunch.Wait`. For handoff, pass the already-held lock and revalidated decision into `HandoffCoordinator`, which owns it through target readiness/rollback. If state changed between picker and lock, restart the decision once and show the new result; never act on the stale row.

- [ ] **Step 3: Enforce the same lock protocol for rename and restart-loop re-entry**

Model exact lock lifetimes in `TagLockPlan`. Rename acquires old/new tag locks in lexical order, recovers both, performs the sidecar move, and releases in reverse order; same-tag rename takes one lock. Restart-loop create/resume returns through the ordinary selected-tag gate. Add concurrency tests proving attach vs handoff, create vs handoff, two creates, and crossed renames cannot overlap or deadlock.

Run: `go test ./cmd/internal/launcher -run 'Test(TagOperation|RenameLocking|ConcurrentTagOperations)' -race -count=1`

Expected: PASS.

- [ ] **Step 4: Wire `ActionHandoff`, confirmation, and help text**

Route only explicit-agent different-driver selections to the coordinator. Decline/dismiss returns 0 without journal, marker, transcript, input, session, ledger, config, or default writes. Update usage text to explain that `pair <agent>` includes live work, resumes the same agent, offers an exclusive switch for another driver, and automatically applies repo-scoped last explicit `--` arguments.

- [ ] **Step 5: Run focused and package verification**

Run: `go test ./cmd/internal/launcher -race -count=1`

Expected: PASS.

- [ ] **Step 6: Commit picker/lock integration**

```bash
git add cmd/internal/launcher
git commit -m "#115 M4: switch tag drivers from the normal picker" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 17: Prove the complete handoff at process level

**Files:**
- Create: `tests/pair-agent-handoff-test.sh`
- Modify: `Makefile.local`
- Modify: `README.md`
- Modify: `atlas/architecture.md`
- Modify: `atlas/session-identity.md`
- Modify: `atlas/index.md` only if a new atlas page is introduced
- Modify: `workshop/issues/000115-resurrect-a-session-across-agents.md`
- Modify if review finds a reusable lesson: `workshop/lessons.md`

- [ ] **Step 1: Add a hermetic process-level fake**

Build the checkout's `bin/pair`/`bin/pair-wrap`, put fake `zellij`, `claude`, and `codex` first on `PATH`, and isolate `PAIR_DATA_DIR`, `HOME`, and repo root. The fake must model a live user-TTY-attached source, ready records, native session artifacts, transcript writers, process trees, deletes, and attach blocking; do not stub Go function calls.

Exercise live Claude → Codex and assert: explicit picker includes the attached tag; confirmation is required; source is gone before target starts; tag/public session stay identical; source bundle has exact raw/events/plain/metadata; old nonempty `*` becomes the first queue item; existing queue keys/order/bodies and history checksum are unchanged; generated `*` carries exact provenance/`--no-restart`; and explicit Codex args become the repo Codex default only after ready. Then hand Codex → Claude and assert the previous Claude native session resumes while the new instruction points at Codex's immutable bundle.

Also exercise decline, conflicting live evidence, concurrent launcher rejection, two stale-recovery racers with one winner, target wrong-nonce/timeout rollback after discarding the live handle, source-recovery readiness, unrecoverable-source second confirmation, pre-known missing transcript disclosure, render-failure re-confirm/decline, stale-marker cleanup, crashes in every effect-before-journal window (source stop, queue link, draft replace, target start/readiness), and post-`target-ready` default/finalization failure. Assert no sibling `*-resurrect*` tag is created.

- [ ] **Step 2: Register and run the acceptance target**

Add `test-agent-handoff` to `.PHONY` and the aggregate `test` dependency.

Run: `make test-agent-handoff`

Expected: PASS with no real Zellij daemon or provider process.

- [ ] **Step 3: Update user and architecture documentation**

Document the normal picker switch flow and repo-agent defaults in `README.md`. In atlas, map tag-as-work vs agent-as-driver, agent-scoped conversations/transcripts, tag-scoped input, lock/journal states, readiness commit point, recovery direction, and the process-level acceptance seam. Append M4 decisions/evidence to the issue log. Leave `- [ ] M4 — Wire exclusive handoff into the normal picker and prove end-to-end recovery.` unchecked for `milestone-close`.

- [ ] **Step 4: Run the full verification matrix**

Run: `go test ./cmd/internal/launcher ./cmd/internal/queuecmd ./cmd/internal/wrapcmd -race -count=1`

Expected: PASS.

Run: `make test-agent-handoff && make test-queue && make test-runtimebundle && make runtimebundle-drift-check`

Expected: PASS.

Run: `go test ./...`

Expected: PASS.

Run: `make test`

Expected: PASS.

Run: `git diff --check`

Expected: no output.

- [ ] **Step 5: Commit pre-boundary acceptance/docs**

```bash
git add tests/pair-agent-handoff-test.sh Makefile.local README.md atlas/architecture.md atlas/session-identity.md atlas/index.md workshop/issues/000115-resurrect-a-session-across-agents.md
git commit -m "#115 M4: prove exclusive cross-agent handoff" \
  -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

- [ ] **Step 6: Close and review M4**

Run the fully spelled M2 boundary protocol with milestone `M4`, `/tmp/pair-115-m4-*` files, verification `race/unit suites, process-level handoff and queue/runtimebundle acceptance, go test ./..., and make test pass; live attached switching, rollback, native-session return, defaults, and exclusivity are covered`. Apply the binary-owned verdict exactly, stage only M4/review-fix/docs/lessons/issue files, and commit subject `#115 M4: close exclusive handoff milestone` with the real `Review-Verdict:`/`Review-Window:` and co-author trailers.

### Task 18: Run the whole-issue integration gate

**Files:**
- Modify: `workshop/issues/000115-resurrect-a-session-across-agents.md`
- Modify if the integration review finds a reusable lesson: `workshop/lessons.md`
- Modify any code/docs named by a `FIX-THEN-SHIP` verdict

- [ ] **Step 1: Re-run final verification from the reviewed M4 head**

Run: `go test ./cmd/internal/launcher ./cmd/internal/queuecmd ./cmd/internal/wrapcmd -race -count=1 && make test-agent-handoff && make test && go test ./... && git diff --check`

Expected: PASS with a clean issue-scoped diff.

- [ ] **Step 2: Run `sdlc close` without masking its exit**

Run `sdlc close --issue 115 --verified 'race/unit suites, hermetic live-agent handoff acceptance, full make test, and go test ./... pass; the same tag switches exclusive drivers with durable state, defaults, rollback, and native-session return' > /tmp/pair-115-close.out 2>&1`; save `$?` as `close_status`, print the file, and require `close_status == 0`. Do not supply guessed `--actual`; let SDLC adopt the measured value.

- [ ] **Step 3: Resolve the binary-owned integration verdict**

Require a real `SHIP` or `FIX-THEN-SHIP` verdict. For `FIX-THEN-SHIP`, apply every Critical/Important fix now, add reusable rules to `workshop/lessons.md`, re-run the final matrix, and bundle fixes with the close mutation in one commit; stop and re-plan on `REWORK`. If review is `not-run`, use the exact whole-issue base/head printed by close with `sdlc judge milestone-review --base BASE --head HEAD --issue 115 --agent codex`, require a real verdict, and record replacement trailers.

- [ ] **Step 4: Commit the codecomplete integration anchor**

Stage only issue #115 plus any integration fixes/docs/lessons, run `git diff --cached --check`, and commit with subject `#115: close exclusive agent handoff`, a why-focused body, the real `Review-Verdict:`/`Review-Window:` trailers, and `Co-Authored-By: OpenAI Codex <noreply@openai.com>`. Verify the issue is `codecomplete`, every M1-M4 row is checked, `git log -1 --format=%B` contains the real trailers, and unrelated user changes remain unstaged. Do not run `sdlc push` or merge without a separate user instruction.

## Revisions

### 2026-07-16T15:53:01-07:00 — code-entry gate refinement

The plan-quality gate required the readiness producer and consumer to share one
wire schema rather than maintain parallel JSON definitions. Task 5 now places
the codec in `cmd/internal/readiness`; launcher and pair-wrap both consume it,
with a cross-facing golden round trip. The issue estimate was independently
re-derived to count all 18 tasks and all five review boundaries.

### 2026-07-16T15:55:14-07:00 — forward draft seam and service-scale estimate

The second code-entry review found the coordinator's forward draft install had
no injected filesystem owner. Added `CommitHandoffDraft` and
`ReconcileHandoffDraft` to the store seam with collision, inode-ownership,
durability, and effect-before-journal tests (ARCH-PURE). Reclassified the
coordinated lock/journal/process recovery and crash matrix as a greenfield
service plus two OS integrations rather than hiding them in generic module
counts; the issue estimate now derives to 16.70 hours.

### 2026-07-16T16:00:15-07:00 — queue seam and post-commit default recovery

The third code-entry review found queue IO still crossed the pure coordinator
boundary, so `CommitHandoffQueue`/`ReconcileHandoffQueue` now own that effect and
its fake/OS failure cases. `target-ready` recovery now explicitly persists the
requested repo-agent default and durably marks it before forward finalization;
effect-before-marker replay is idempotent. Pair-wrap readiness publication
reuses `osfs.FS.WriteAtomic`. The estimate now counts M3's durability substrate
and M4's live coordinator as two distinct service-scale items, deriving to
23.35 hours.
