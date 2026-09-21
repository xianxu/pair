# Qoder Harness Integration Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the `qoder` CLI (v1.1.59) up to parity with `claude`/`codex`/`agy`/`muse` across all surfaces of `atlas/how-to-bring-up-a-new-harness-cli.md` (§0 registry, aspects 1–7), in both hosts (standalone pair and couch).

**Architecture:** Qoder is claude-family on disk (transcript JSONL records decode through the claude record transition; resume is `--resume <uuid>`; session storage is `~/.qoder/projects/<encoded-repo>/<uuid>.jsonl`), so the scanner derives from a parameterized claude-family core rather than a copy (ARCH-DRY). Every other surface is a registration in an existing per-agent seam; the TTY recognizer and prompt glyphs are **capture-first** — no recognizer or glyph is written from imagination, only from bytes recorded through the live PTY seam (atlas aspect 2 discipline).

**Tech Stack:** Go (pair repo, `cmd/internal/*`), Neovim Lua (`nvim/scrollback.lua`), JSONL fixtures, live PTY capture harness (`PAIR_LIVE_HARNESS`).

**Verified ground facts (2026-09-20, qoder 1.1.59):**

- Binary `qoder` on PATH; `~/.qoder/entry/qoder`; `--version` → `1.1.59`.
- Sessions: `~/.qoder/projects/-Users-xianxu-workspace-pair/<uuid>.jsonl` plus `<uuid>/` sidecar dir (`state.json`, `compression-v2/`, `subagents/agent-*.meta.json`). `qoder --list-sessions` prints `[<uuid>]` — the resume identifier is the session UUID.
- Transcript records are claude-shaped: `{"type":"user","uuid":…,"timestamp":"ISO8601","message":{"role":"user","content":"…"},"parentUuid":null,"isSidechain":false,"sessionId":"<uuid>","version":"1.1.59",…}` plus noise records (`workspace-directories`, `runtime-config`, `worktree-state`, `active-leaf`, `attachment`) that carry `sessionId` or nothing.
- CLI flags of interest: `-r/--resume [id]`, `-c/--continue`, `--session-id <id>`, `--fork-session`, `-p/--print` (non-interactive, query positional), `-o/--output-format`, `-m/--model`, `--permission-mode {default,accept_edits,bypass_permissions,dont_ask,auto}`, `--list-sessions`, `--remote*`/`--teleport` (cloud sessions).
- Settings: `~/.qoder/settings.json` with `permissions.trustDirectories` (already lists `/Users/xianxu/workspace/pair`).
- Couch spawns hosted threads as `pair resume <tag>` (`cmd/internal/couchcore/launch_existing.go:58`) and enumerates agents via `launcher.AgentInventory()` — no couch-side table (atlas §0). Unknown TTY profiles fail closed (`profileForHarness` miss → `unknownComposerDecision` → bare CR passthrough), so the registry can light up before the scanner/profile land.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `AgentQoder` (enum value) | `cmd/internal/sessioninventory/model.go` | new |
| `supportedAgents` + qoder row | `cmd/internal/launcher/agent_defaults.go` | modified |
| Claude-family scanner core (`scanClaudeFamily`, `validateClaudeFamilyDelta`) | `cmd/internal/sessioninventory/scan_claude.go` | modified |
| `ScanQoder` / `ValidateQoderDelta` | `cmd/internal/sessioninventory/scan_qoder.go` | new |
| `resumeToken`/`composeResumeArgs` qoder case | `cmd/internal/launcher/agentargs.go` | modified |
| `extractExplicitResume` qoder case | `cmd/internal/launcher/createlogic.go` | modified |
| `ValidateFreshAgentArgs`/`freshValueOption` qoder case | `cmd/internal/launcher/fresh_args.go` | modified |
| `qoder` TTY profile (keymap + recognizer + overlay) | `cmd/internal/wrapcmd/harness_tty.go` | new entry |
| Qoder composer recognizer | `cmd/internal/wrapcmd/composer_recognizers.go` | new (capture-gated) |
| `runQoder` + `DefaultModel` qoder row | `cmd/internal/model/model.go` | new / modified |
| Prompt-glyph registrations | `nvim/scrollback.lua`, `cmd/internal/wrapcmd/orientation.go`, `cmd/internal/changelogcmd/distill.go` | modified (capture-gated) |

- **`AgentQoder` + `supportedAgents` row** — the §0 registry pair. One string in the launcher slice (couch menus, switch-agent validation, storage-GC, rename/migrate derive from it automatically) and one typed value in the sessioninventory enum (scanners, ledger records, CLI). They join in the same commit — a half-joined registry is the drift the atlas §0 warns about.
  - **Relationships:** 1:1 launcher-string ↔ enum-value; N consumers read both.
  - **DRY rationale:** every host derives from these two; no per-host roster (ARCH-DRY, ARCH-PURPOSE — the registry is *enforced*, not restated).
  - **Future extensions:** the next harness joins the same two rows.
- **Claude-family scanner core** — `ScanClaude`'s record transition (`applyClaudeRecord`, `claudePathFact`) is agent-agnostic except for the `Agent` constant and `ScannerSchema` string. Parameterize both; claude and qoder become two thin producers of one transition.
  - **Relationships:** 1 transition : N producers (claude-v1, qoder-v1).
  - **DRY rationale:** qoder transcripts pass the claude decode (`type`/`sessionId`/`isSidechain`/`timestamp`/`message.role` — verified against a real transcript); a copy would fork the transition and drift at the next record-shape change (ARCH-DRY).
  - **Future extensions:** any claude-transcript-derived harness (the family keeps growing).
- **TTY profile / recognizer / glyphs** — capture-first. The profile ships **fail-closed first** (keymap only, no `composerGate`), the recognizer and glyph registers only after fixtures prove the stable signal.
  - **Relationships:** profile 1:1 with the wrap proxy instance; recognizer is a pure function over `terminalSnapshot` (atlas aspect 2).
  - **DRY rationale:** if qoder paints claude's ruled-box composer shape, it shares `ruledBoxComposerActive` via a spec, not a near-copy (atlas aspect 2 explicitly demands this).
  - **Future extensions:** overlay marker families grow from `near-miss` telemetry `detail` strings.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| OSRuntime qoder root | `cmd/internal/sessioninventory/runtime_os.go` | new row | `~/.qoder/projects` filesystem |
| Live TTY capture fixtures | `cmd/internal/wrapcmd/testdata/tty/qoder/1.1.59/` | new | real qoder PTY output |
| Scanner conformance fixtures | `cmd/internal/sessioninventory/testdata/native/qoder/v1/qoder-projects/` | new | real qoder transcript shapes (sanitized) |
| `ProviderQoderJSONLV1` contract | `cmd/internal/sessioninventory/provider_contract.go` | new row | reviewed append-only producer contract |
| Watcher/ledger/CLI membership | `sessionwatch/sessionwatch.go`, `sessionledger/record.go`, `sessioninventory/runcli.go` | modified | agent allowlists |
| Qoder settings (trust) | `~/.qoder/settings.json` | config | qoder permission system (aspect 6, static — no signal) |
| `runQoder` print invocation | `cmd/internal/model/model.go` | new | `qoder -p` subprocess (slug summarize) |

- **OSRuntime qoder root** — name `qoder-projects`, path `~/.qoder/projects`. Feeds `AgentSessionExists`, `QuerySession`, couch's native-binding resume.
  - **Injected into:** every scanner/observation consumer; nothing agent-specific beyond the row (ARCH-PURE).
- **Live TTY fixtures** — captured bytes through the bounded PTY seam, never hand-authored (atlas aspect 2). `metadata.json` carries `--version` string, argv, RFC3339 time, per-file SHA-256. Bounded inventory: newest version dir only.
- **`runQoder`** — `qoder -p` with `cmd.Dir = os.TempDir()` (sandbox: no workspace context), `PAIR_SLUG_NESTED=1`, mirroring `runAgy`/`runMuse`.

**Test surface.** Pure entities get colocated unit tests (launcher arg tables, scanner deltas over fixture records, recognizer-over-snapshot). Integration seams get fakes already in-tree (`sessioninventorytest.NewFakeRuntime`, fixture replay at every byte split) plus live conformance (opt-in `PAIR_LIVE_*`) — no new fakes needed; the capture fixtures *are* the stateful double for the real CLI's paint behavior, and live conformance is the drift check (ARCH-MOCK).

---

## Chunk 1: Milestone M1 — registry + launcher arg plumbing

**Boundary:** qoder is accepted by every pure launcher/arg seam (headless; no IO). Everything that activates on the registry (couch menus, storage-GC) behaves fail-closed with no scanner/profile — verified by test, not assumption.

### Task 1: Join the registry (§0, checklist item 1)

**Files:**
- Modify: `cmd/internal/launcher/agent_defaults.go:19`
- Modify: `cmd/internal/sessioninventory/model.go` (enum block, ~line 12-19)
- Test: `cmd/internal/launcher/agent_defaults_test.go:88`

- [ ] **Step 1: Extend the failing inventory test**

In `agent_defaults_test.go`, `TestAgentInventoryIsTheSingleDefensiveHarnessSet` pins the list — update `want`:

```go
want := []string{"claude", "codex", "agy", "muse", "qoder"}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./cmd/internal/launcher -run TestAgentInventoryIsTheSingleDefensiveHarnessSet -v`
Expected: FAIL — `AgentInventory = ["claude" "codex" "agy" "muse"]`

- [ ] **Step 3: Add the enum value and the registry row**

`model.go` (beside `AgentMuse`):

```go
AgentQoder Agent = "qoder"
```

`agent_defaults.go:19`:

```go
var supportedAgents = []string{"claude", "codex", "agy", "muse", "qoder"}
```

- [ ] **Step 4: Run the full launcher package**

Run: `go test ./cmd/internal/launcher`
Expected: PASS (if any other test pins the four-agent set, extend it in this same commit — the failure names it)

- [ ] **Step 5: Verify the fail-closed intermediate (no scanner/profile yet)**

Run: `go test ./cmd/internal/wrapcmd ./cmd/internal/sessioninventory ./cmd/internal/couchcore ./cmd/internal/couchtty`
Expected: PASS. This is the load-bearing check: the registry row activates couch menus and storage-GC *now*, and every consumer must tolerate an agent with no TTY profile (verified: `profileForHarness` miss → bare-CR passthrough) and no scanner root (no observations → empty inventory, `ScannerForAgent` default → unsupported-agent diagnostic only when explicitly queried).

- [ ] **Step 6: Commit**

```bash
git add cmd/internal/launcher/agent_defaults.go cmd/internal/launcher/agent_defaults_test.go cmd/internal/sessioninventory/model.go
git commit -m "#300 M1: qoder joins the agent registry (launcher + sessioninventory enum)"
```

### Task 2: Fresh-args validation

**Files:**
- Modify: `cmd/internal/launcher/fresh_args.go:31-120`
- Test: `cmd/internal/launcher/fresh_launch_test.go`

- [ ] **Step 1: Write the failing table rows**

Extend the fresh-args table tests in `fresh_launch_test.go` (follow the existing claude/agy row shapes):

```go
// qoder: context selectors rejected; value flags consume their value.
{"qoder", []string{"--resume", "abc"}, true},          // selects existing conversation
{"qoder", []string{"-r", "abc"}, true},                // shorthand resume
{"qoder", []string{"--continue"}, true},               // -c/--continue
{"qoder", []string{"-c"}, true},
{"qoder", []string{"--session-id", "x"}, true},        // pair mints ids itself
{"qoder", []string{"--fork-session"}, true},
{"qoder", []string{"--remote"}, true},
{"qoder", []string{"--remote-session", "x"}, true},
{"qoder", []string{"--teleport", "x"}, true},
{"qoder", []string{"--remote-control", "x"}, true},
{"qoder", []string{"--list-sessions"}, true},
{"qoder", []string{"--delete-session", "1"}, true},
{"qoder", []string{"-m", "some-model", "hello"}, false},   // value flag + prompt
{"qoder", []string{"--model", "m", "-p"}, false},          // -p is a bool, not a value
{"qoder", []string{"--worktree", "feat", "hello"}, false}, // optional-value worktree
{"qoder", []string{"--tools", "a", "b", "--", "hi"}, false},
{"qoder", []string{"hello", "world"}, false},
```

(Adjust to the actual table's field order/bool polarity — `true` here means "rejected".)

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/launcher -run TestValidateFreshAgentArgs -v` (or the package's actual fresh-args test name)
Expected: FAIL on the qoder rows (currently: only NUL/unsupported rejected; `--resume` passes).

- [ ] **Step 3: Implement the qoder case**

In `ValidateFreshAgentArgs`, add a `case "qoder":` beside claude/agy:

```go
case "qoder":
	switch flag {
	case "--resume", "--continue", "--session-id", "--fork-session",
		"--remote", "--remote-session", "--teleport", "--remote-control",
		"--list-sessions", "--delete-session":
		forbidden = true
	}
	// Short-flag clusters: -c (continue) and -r (resume) select context.
	// -m -i -w -n -o take values (their value follows as the next argv).
	if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") {
		for _, r := range arg[1:] {
			if r == 'c' || r == 'r' {
				forbidden = true
				break
			}
			if r == 'm' || r == 'i' || r == 'w' || r == 'n' || r == 'o' || r == 'p' || r == 'd' {
				break
			}
		}
	}
```

In `freshValueOption`, add:

```go
case "qoder":
	flags = "--add-dir --allowed-mcp-server-names --allowed-tools --attachment --config-dir --context-window --cwd --delete-session --disallowed-tools --input-format --max-output-tokens --model --name --output-format --permission-mode --plugin-dir --prompt-interactive --reasoning-effort --remote-control --remote-session --teleport --thinking --thinking-budget --tools -i -m -n -o -w"
```

For the variadic `--tools` / repeatable `--allowed-tools` / `--disallowed-tools` / `--add-dir`, generalize the claude-only variadic gate at fresh_args.go:73 — change `if agent == "claude" && freshVariadicOption(flag)` to a per-agent function:

```go
func freshVariadicOption(agent, flag string) bool {
	switch agent {
	case "claude":
		switch flag {
		case "--add-dir", "--allowedTools", "--allowed-tools", "--disallowedTools", "--disallowed-tools", "--betas", "--file", "--mcp-config", "--tools":
			return true
		}
	case "qoder":
		switch flag {
		case "--add-dir", "--allowed-tools", "--disallowed-tools", "--tools":
			return true
		}
	}
	return false
}
```

And guard `--worktree`'s optional value like claude's `-d/-w` guard at line 69 (skip next arg only when it doesn't start with `-`).

- [ ] **Step 4: Run to verify pass**

Run: `go test ./cmd/internal/launcher`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/launcher/fresh_args.go cmd/internal/launcher/fresh_launch_test.go
git commit -m "#300 M1: qoder fresh-args validation (context selectors rejected)"
```

### Task 3: Resume token + explicit-resume extraction

**Files:**
- Modify: `cmd/internal/launcher/agentargs.go:146-177`
- Modify: `cmd/internal/launcher/createlogic.go:57-85`
- Test: `cmd/internal/launcher/agentargs_test.go`

- [ ] **Step 1: Extend the failing table tests**

`agentargs_test.go` — add rows (follow existing shape):

```go
{"qoder", "SID", []string{"--resume", "SID"}},          // resumeToken
{"qoder", []string{"--model", "m"}, "SID", []string{"--model", "m", "--resume", "SID"}}, // composeResumeArgs — append-style (global flag, any position)
```

`createlogic.go`'s `extractExplicitResume` tests (find its table; likely in `createlogic_test.go`):

```go
{"qoder", []string{"--resume", "abc"}, "abc"},
{"qoder", []string{"-r", "abc"}, "abc"},
{"qoder", []string{"--resume=abc"}, "abc"},
{"qoder", []string{"hello"}, ""},
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/launcher -run 'Resume|ExplicitResume' -v`
Expected: FAIL on qoder rows (empty token today).

- [ ] **Step 3: Implement**

`agentargs.go` `resumeToken`:

```go
case "qoder":
	return []string{"--resume", sid}
```

`composeResumeArgs` needs **no change** — qoder falls through to the append branch (verified: `--resume` is a global option in qoder, valid in any position, unlike codex/muse's leading subcommand).

`createlogic.go` `extractExplicitResume` — add a qoder case beside the claude/agy one:

```go
case "qoder":
	prev := ""
	for _, tok := range args {
		if prev == "--resume" || prev == "-r" {
			return tok
		}
		if v, ok := strings.CutPrefix(tok, "--resume="); ok && v != "" {
			return v
		}
		prev = tok
	}
```

- [ ] **Step 4: Run to verify pass + verify no sessionwatch change needed**

Run: `go test ./cmd/internal/launcher`
Expected: PASS.

Also verify (read-only): `sessionwatch.StripResumeArgs` already strips `--resume <id>` pairs generically for every agent (sessionwatch.go:51) — qoder needs **no** change there. `SupportsAgent` lands in M2 with the scanner (adding it before the scanner exists would let `pair session-watch` run against an agent with no roots; keep the two together).

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/launcher/agentargs.go cmd/internal/launcher/agentargs_test.go cmd/internal/launcher/createlogic.go cmd/internal/launcher/createlogic_test.go
git commit -m "#300 M1: qoder resume token (--resume <uuid>) + explicit-resume extraction"
```

### Task 4: M1 boundary — milestone close

- [ ] **Step 1: Full suite**

Run: `make -f Makefile.local test 2>/dev/null || go test ./...`
Expected: PASS (or pre-existing failures only — record them in `## Log` if any).

- [ ] **Step 2: Close the milestone**

Run: `sdlc milestone-close --issue 300 --milestone M1`
The binary auto-dispatches the fresh-eyes review over the M1 diff window; fix Critical/Important findings before proceeding; log the verdict in `## Log`.

- [ ] **Step 3: Live sanity (operator or agent, 2 min)**

`qoder --version` → 1.1.59; `qoder --list-sessions` shows uuids. No pair boot yet (no scanner — resume can't resolve).

---

## Chunk 2: Milestone M2 — session inventory scanner

**Boundary:** `pair session-inventory --agent qoder` works; a real qoder transcript becomes an established root with a native binding; couch's parked/live projection and cold resume work off it.

### Task 5: Extract the claude-family scanner core

**Files:**
- Modify: `cmd/internal/sessioninventory/scan_claude.go`
- Test: `cmd/internal/sessioninventory/scan_claude_test.go` (must stay green, unchanged)

- [ ] **Step 1: Refactor to a parameterized core (behavior-preserving)**

In `scan_claude.go`, extract:

```go
// scanClaudeFamily runs the claude-family record transition for one agent and
// scanner schema. Claude and qoder transcripts share the record shape
// (type/sessionId/isSidechain/timestamp/message.role) and the path layout
// (<project>/<uuid>.jsonl, <uuid>/subagents/agent-*.jsonl).
func scanClaudeFamily(runtime Runtime, agent Agent, schema string) ScanResult {
	var result ScanResult
	for _, root := range runtime.NativeRoots(agent) {
		files, diagnostics, ok := scannerFiles(runtime, agent, root)
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		if !ok {
			continue
		}
		for _, entry := range files {
			fact, diagnostics, ok := scanClaudeFamilyFile(runtime, entry, agent, schema)
			result.Diagnostics = append(result.Diagnostics, diagnostics...)
			if ok {
				result.Facts = append(result.Facts, fact)
			}
		}
	}
	return result
}

func ScanClaude(runtime Runtime) ScanResult { return scanClaudeFamily(runtime, AgentClaude, "claude-v1") }
```

Thread `agent`/`schema` through `scanClaudeFile` → `scanClaudeFamilyFile`, `applyClaudeRecord` (parameterize the `AgentClaude` constant in its diagnostics and the `state.Agent` field), and `ValidateClaudeDelta` → `validateClaudeFamilyDelta(entry, prior, records, agent, schema)`. Keep the exported `ValidateClaudeDelta(entry, prior, records)` as a one-line delegate with claude's constants (its consumers in `incremental_inventory.go` keep compiling untouched). Parameterize the "unrecognized Claude v1 path" message with the schema string.

- [ ] **Step 2: Verify behavior preservation**

Run: `go test ./cmd/internal/sessioninventory`
Expected: PASS with **zero test edits** — the refactor must be invisible to the claude suite. (If a test hardcodes the diagnostic message, that's the one allowed edit; note it in the commit body.)

- [ ] **Step 3: Commit**

```bash
git add cmd/internal/sessioninventory/scan_claude.go
git commit -m "#300 M2: extract claude-family scanner core (claude + qoder share the record transition)"
```

### Task 6: `ScanQoder` + roots + wiring

**Files:**
- Create: `cmd/internal/sessioninventory/scan_qoder.go`
- Modify: `cmd/internal/sessioninventory/model.go:444-451` (`validAgent` case)
- Modify: `cmd/internal/sessioninventory/runtime_os.go:33-46`
- Modify: `cmd/internal/sessioninventory/conformance.go:131-145`
- Modify: `cmd/internal/sessioninventory/incremental_inventory.go` (two switches, lines ~135-142 and ~193-200)
- Modify: `cmd/internal/sessioninventory/incremental_inventory.go:91-99` (`artifactScannerShape`)
- Modify: `cmd/internal/sessioninventory/provider_contract.go`
- Modify: `cmd/internal/sessioninventory/runcli.go:18,85`
- Test: `cmd/internal/sessioninventory/scan_qoder_test.go` (new)

- [ ] **Step 1: Write the failing scanner test**

Model on `scan_muse_test.go` / `scan_claude_test.go`; fixture tree mirrors the claude layout (encoded dir `-repo`, placeholder UUIDs `1111…`-style — copy the sanitization convention, never real paths/UUIDs):

`cmd/internal/sessioninventory/testdata/native/qoder/v1/qoder-projects/-repo/11111111-1111-4111-8111-111111111111.jsonl` — records **copied from a real transcript and sanitized** (the `workspace-directories`, `runtime-config`, `user` sequence from the ground facts, with `sessionId` rewritten to the placeholder and paths to `-repo`).

```go
func TestScanQoderV1(t *testing.T) {
	t.Parallel()
	runtime := sessioninventorytest.NewFakeRuntime()
	loadNativeFixture(t, runtime, sessioninventory.AgentQoder, "qoder-projects", filepath.Join("testdata", "native", "qoder", "v1", "qoder-projects"))
	got := inventoryFromScan(sessioninventory.ScanQoder(runtime))

	if len(got.Forests) != 1 || len(got.Forests[0].Roots) != 1 {
		t.Fatalf("forests = %#v", got.Forests)
	}
	root := got.Forests[0].Roots[0]
	if root.NativeID != "11111111-1111-4111-8111-111111111111" || !root.Resumable {
		t.Fatalf("root = %#v", root)
	}
	if root.Time == nil {
		t.Fatalf("root has no chronology")
	}
}

func TestValidateQoderDelta(t *testing.T) {
	t.Parallel()
	entry := sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: "qoder-projects", RelativePath: "-repo/11111111-1111-4111-8111-111111111111.jsonl"}}
	first := []sessioninventory.FramedJSONLRecord{{Bytes: []byte(`{"type":"workspace-directories","sessionId":"11111111-1111-4111-8111-111111111111","directories":["/repo"]}`)}}
	state, _, err := sessioninventory.ValidateQoderDelta(entry, nil, first)
	if err != nil || state.Disputed || state.FirstRecordValidated == false {
		t.Fatalf("state=%#v err=%v", state, err)
	}
	// A record naming a different session disputes the file.
	state, diags, err := sessioninventory.ValidateQoderDelta(entry, &state, []sessioninventory.FramedJSONLRecord{{Bytes: []byte(`{"type":"user","sessionId":"99999999-9999-4999-8999-999999999999","message":{"role":"user","content":"x"}}`)}})
	if !state.Disputed || !diagnosticPresent(diags, sessioninventory.DiagnosticNodeMalformed) {
		t.Fatalf("state=%#v diagnostics=%#v", state, diags)
	}
}
```

Also add a subagent-transcript case to the fixture **iff** a real qoder run shows subagent `.jsonl` transcripts exist (check a real `<uuid>/subagents/` dir; if qoder only writes `.meta.json` sidecars, the path fact simply never matches them and the fixture documents that with a `.meta.json` file that must produce no diagnostic).

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/sessioninventory -run Qoder -v`
Expected: FAIL — `ScanQoder` undefined.

- [ ] **Step 3: Implement**

`scan_qoder.go`:

```go
package sessioninventory

// ScanQoder scans qoder's claude-family transcripts under ~/.qoder/projects.
// The record transition and path shape are shared with claude (scan_claude.go);
// only the agent constant, schema id, and storage root differ.
func ScanQoder(runtime Runtime) ScanResult {
	return scanClaudeFamily(runtime, AgentQoder, "qoder-v1")
}

// ValidateQoderDelta applies complete records to a cloned scanner state.
func ValidateQoderDelta(entry FileEntry, prior *ScannerState, records []FramedJSONLRecord) (ScannerState, []Diagnostic, error) {
	return validateClaudeFamilyDelta(entry, prior, records, AgentQoder, "qoder-v1")
}
```

`runtime_os.go` roots map:

```go
AgentQoder: {{Agent: AgentQoder, Name: "qoder-projects", Path: filepath.Join(homeDir, ".qoder", "projects")}},
```

`conformance.go` `ScannerForAgent`: `case AgentQoder: return ScanQoder`.

`incremental_inventory.go` — add `case AgentQoder: state, found, err = ValidateQoderDelta(observation.Entry, nil, observed.Records)` (first switch) and the prior-state variant (second switch); `artifactScannerShape`:

```go
case AgentQoder:
	_, _, _, ok := claudePathFact(artifact.RelativePath)
	return "qoder-v1", ArtifactTranscript, ok && artifact.StorageRoot == "qoder-projects"
```

`provider_contract.go`: `ProviderQoderJSONLV1 ProviderContract = "qoder-jsonl-v1"` and the `ProviderContractFor` case mapping (qoder-projects + qoder-v1 → that contract).

`model.go` `validAgent` (`:444-451`): add `AgentQoder` to the case list — it gates `ValidateScannerState`/`ScannerStateFact`, catalog entries, and the CLI's `--agent qoder`; without it the scanner's own first test fails, not just a later surface.

`runcli.go`: `var supportedAgents = []Agent{AgentAgy, AgentClaude, AgentCodex, AgentMuse, AgentQoder}` and the usage string gains `|qoder`.

- [ ] **Step 4: Run to verify pass**

Run: `go test ./cmd/internal/sessioninventory`
Expected: PASS. Then the sweep tests that pin per-agent inventories:
Run: `go test ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/sessionwatch`
Expected: PASS or failures naming the remaining pinning tests (e.g. `shadow_test.go` native-path guard regexes, `couchcore/plan_contract_test.go` scan-file inventory) — extend each named test with the qoder row in this same commit.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/sessioninventory/ cmd/internal/couchcore/plan_contract_test.go
git commit -m "#300 M2: qoder session scanner (claude-family core, qoder-v1 schema)"
```

### Task 7: Event adapter + membership + AgentSessionExists

**Files:**
- Modify: `cmd/internal/sessioninventory/event.go:53-62`
- Modify: `cmd/internal/sessioninventory/target.go:199-227` (`observationNativeID` case)
- Modify: `cmd/internal/sessionwatch/sessionwatch.go:33-40`
- Modify: `cmd/internal/sessionledger/record.go:478-491`
- Test: `cmd/internal/sessioninventory/events_test.go` (extend), `cmd/internal/launcher/osruntime_test.go` (extend)

- [ ] **Step 1: Failing tests**

Event adapter — a qoder `user` record must project one operator text turn (the slug seam), and qoder noise records (`runtime-config`, `attachment`) must not:

```go
func TestNormalizeQoderUserEvent(t *testing.T) {
	events, disposition := sessioninventory.NormalizeNativeEvent(sessioninventory.AgentQoder,
		[]byte(`{"type":"user","message":{"role":"user","content":"tell me about this repo"},"isSidechain":false,"sessionId":"11111111-1111-4111-8111-111111111111"}`))
	if disposition == sessioninventory.EventNearMiss || len(events) == 0 {
		t.Fatalf("events=%#v disposition=%v", events, disposition)
	}
}
```

`sessionwatch`/`sessionledger` membership tests: extend each package's per-agent membership test with `"qoder"` rows (the existing claude/muse rows show the shape).

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/sessioninventory ./cmd/internal/sessionwatch ./cmd/internal/sessionledger -run 'Qoder|Supports|Supported' -v`
Expected: FAIL.

- [ ] **Step 3: Implement**

`event.go` `NormalizeNativeEvent`: `case AgentQoder: return normalizeClaudeEvent(record)` — qoder records are claude-shaped (ground facts); the fixture test pins it. If the noise records produce `EventNearMiss` dispositions (a telemetry storm), add the qoder noise `type` values to the normalizer's ignore set with a comment naming them — evidence decides, not speculation.

`target.go` `observationNativeID` (`:199-227`): this is a per-agent switch, **not** agent-agnostic — add `case AgentQoder:` using `claudePathFact` over `qoder-projects` (claude's own arm is the shape). Without it `AgentSessionExists("qoder", …)` returns false (`NativeSessionCandidateExistsFromObservations` → `selectNamedArtifacts` → `""`), and the watcher's `TargetNewLaunch` discovery (`target.go:133`) finds no new qoder sessions — the binding round-trip the Done-when rests on.

`sessionwatch.go` `SupportsAgent`: add `"qoder"` to the case list.
`sessionledger/record.go` `isSupportedAgent`: add the qoder case (match the file's existing shape).

`osruntime_test.go`: add a present/absent pair for qoder — fake runtime root `qoder-projects` with a `<uuid>.jsonl` transcript → `AgentSessionExists("qoder", uuid, cwd)` true; empty root → false. The test proves the root row reaches the (now completed) per-agent delegation.

- [ ] **Step 4: Run to verify pass**

Run: `go test ./cmd/internal/sessioninventory ./cmd/internal/sessionwatch ./cmd/internal/sessionledger ./cmd/internal/launcher`
Expected: PASS.

- [ ] **Step 5: Live conformance (opt-in, uses the real ~/.qoder)**

Run: `go test ./cmd/internal/sessioninventory -run TestConformanceLive -count=1 -v` (or the package's live conformance entry point — check `conformance_live_test.go` for the opt-in env)
Expected: qoder row reports a real established root from the existing local sessions (the two `--list-sessions` uuids).

- [ ] **Step 6: Commit**

```bash
git add cmd/internal/sessioninventory/ cmd/internal/sessionwatch/ cmd/internal/sessionledger/ cmd/internal/launcher/
git commit -m "#300 M2: qoder events, watcher/ledger membership, AgentSessionExists"
```

### Task 8: M2 boundary

- [ ] **Step 1:** `go test ./...` green.
- [ ] **Step 2:** `sdlc milestone-close --issue 300 --milestone M2`; fix Critical/Important findings; log the verdict.
- [ ] **Step 3:** Manual: `pair session-inventory --agent qoder --json` lists the real local sessions with resumable roots.

---

## Chunk 3: Milestones M3 (TTY) and M4 (slug/glyphs/settings)

### Task 9: Bootstrap profile + live capture — one atomic commit

**Files:**
- Modify: `cmd/internal/wrapcmd/harness_tty_live_test.go` (`commands` map `:547-556`)
- Modify: `cmd/internal/wrapcmd/harness_tty.go` (`harnessTTYProfiles`: keymap + `composerGatePositive` + `recognize`)
- Modify: `cmd/internal/wrapcmd/harness_tty_fixture_test.go` (`TestComposerReturnExpectationMatchesProfile` `:803-819`; gap ledgers only where the oracle reads them)
- Create: `cmd/internal/wrapcmd/testdata/tty/qoder/1.1.59/composer.raw` (capture)
- Create: `cmd/internal/wrapcmd/testdata/tty/qoder/1.1.59/overlay.raw` (capture, driven scenario — optional, Step 4)
- Create: `cmd/internal/wrapcmd/testdata/tty/qoder/1.1.59/metadata.json`

**Why bootstrap-first (verified against the harness, not assumed):** a capture cannot run before a positive gate exists, so the intermediate "keymap-only fail-closed profile" (earlier draft of this plan) is not a capturable — nor even committable — state. It is replaced by this atomic landing:

- `TestHarnessTTYLiveConformance` hard-fatals for any harness with no `commands` row — `PAIR_LIVE_HARNESS=%q, want agy, codex, or muse` (`harness_tty_live_test.go:547-556`). Row first.
- The capture's startup predicate IS the recognizer: `newHarnessTTYLiveClassifier` fatals `%s has no positive-gated live profile` unless a `composerGatePositive` profile with non-nil `recognize` already exists (`harness_tty_live_test.go:216-218`), and `configureHarnessTTY` releases the terminal for every non-positive gate (`wrap.go:1565-1580`).
- The fixture oracle makes captures and gate atomic from the other direction too: fixtures for a non-positive-gated agent error (`harness_tty_fixture_test.go:93-96`), and a positive gate with no fixtures fatals (`:134-136`). No split commit is green.
- Capture-first survives in substance: no `.raw` byte is ever hand-authored, and the recognizer must be validated by the classifier against live qoder bytes (`harnessTTYRecognized`) before any capture is written. The capture then freezes both.

- [ ] **Step 1: Register the command row** — `"qoder": {"qoder"}`, plus any disable/onboarding flag qoder demonstrably needs from a spawned PTY, mirroring how the codex/agy rows carry theirs.

- [ ] **Step 2: Author the bootstrap profile.** Hand-run qoder in a real terminal; read the idle composer (prompt chrome, cursor state, any OSC). Register in `harnessTTYProfiles`:

```go
"qoder": {
	keymap: sendKeymap{
		plainCR: ..., // observed Return semantics; final values pinned by the capture
		altCR:   ...,
		altBS:   ...,
	},
	composerGate: composerGatePositive,
	recognize:    ..., // first approximation from the live screen
},
```

Recognizer decision tree, in order of preference (atlas aspect 1): (a) qoder emits a native composer-availability OSC → wrap it; (b) qoder paints claude's ruled-box shape (`─` rules flanking the prompt row) → add a spec to `ruledBoxComposerActive`, not a fourth near-copy; (c) a novel glyph/shape → new recognizer function.

- [ ] **Step 3: Iterate the live capture.** `PAIR_LIVE_HARNESS=qoder go test ./cmd/internal/wrapcmd -run TestHarnessTTYLiveConformance -count=1 -v` until the classifier reports `recognized` (what it reports instead, naming the blocker: `harnessTTYUnauthenticated`, `harnessTTYWorkspaceTrust`, `harnessTTYWaiting`). Then add `PAIR_LIVE_CAPTURE_OUT=cmd/internal/wrapcmd/testdata/tty/qoder/1.1.59/composer.raw` and capture. A recognizer that never fires times out at startup with `state=waiting` (`harness_tty_live_test.go:592-594`); `reported recognition but no recognized prefix` (`:595-598`) is the byte-replay path — the live stream looked recognized but replaying the captured bytes cannot reproduce it, and a recognizer that fires too early also truncates the capture (`firstRecognizedHarnessTTYPrefix` `:853-867` cuts at the first recognized byte). Either way: confirm the captured screen is the settled composer, and refine the recognizer, never the fixture.

- [ ] **Step 4: Capture one blocking overlay (preferred, not required).** Add a `harnessTTYDrivenScenarios["qoder"]` row (name/`send`/`until`/`wantComposer: false`/`file: "overlay.raw"`; set `discriminating: true` only if the screen truly is composer-shaped — it is honor-system and is what retires the discrimination ledger). Run:

```bash
PAIR_LIVE_HARNESS=qoder PAIR_LIVE_SCENARIO=<name> \
PAIR_LIVE_CAPTURE_OUT=cmd/internal/wrapcmd/testdata/tty/qoder/1.1.59/overlay.raw \
  go test ./cmd/internal/wrapcmd -run TestHarnessTTYLiveDrivenConformance -count=1 -v
```

An unreachable screen skips (`did not reach … may not be reproducible`). If nothing declines today, skip the capture and take the honest-gap path in Step 6.

- [ ] **Step 5: metadata.json** — exact `qoder --version` output, argv (the `commands` row), RFC3339 capture time, SHA-256 per raw file (copy `testdata/tty/muse/*/metadata.json`).

- [ ] **Step 6: Per-harness registrations + ledgers — every one the oracle reads** (`harness_tty_fixture_test.go:73-84` seeds the positive-gated set; `:141-150` and `:160-176` are the expiry oracles):
  - **Profile registry**: `TestHarnessTTYProfileRegistry` (`harness_tty_test.go:20-31`) iterates its own profile + recognizer tables — add the qoder rows or the profile lands unpinned (silent staleness, same class as the expectation table).
  - **Return expectation**: the qoder row in `TestComposerReturnExpectationMatchesProfile` (`:803-819`) — its table iterates itself, so omission is silent staleness, not a failure.
  - **Negative/discrimination ledgers**: overlay.raw captured and the scenario is `discriminating` → **no** entry in either ledger (a `ttyFixtureNegativeGaps` entry plus a captured overlay.raw errors: "now has a captured declining state; drop its entry"); overlay.raw captured but it declines on incidental state (cursor/size), not composer-vs-picker → `ttyFixtureDiscriminationGaps["qoder"]` naming the unproven separation; no overlay capture → both `ttyFixtureNegativeGaps["qoder"]` and `ttyFixtureDiscriminationGaps["qoder"]`, each with the measured reason.
  - **Reaction gap**: every positively gated harness must either press Return in a driven scenario (`pressesReturn`) or carry a `ttyFixtureReactionGaps` entry (`:160-176`). Prefer a `pressesReturn` scenario — it directly evidences Done-when's core claim (plain Enter inserts a newline, Alt+Enter sends). If no Return can be driven before landing, record the entry with the honest undriven reason; the entry expires the moment a `pressesReturn` scenario lands (the oracle errors on both stale presence and missing absence).

- [ ] **Step 7: Run and commit as ONE tree state.** `go test ./cmd/internal/wrapcmd` green (the fixture oracle now replays composer.raw through the profile), then a single commit — commands row + profile (keymap/gate/recognizer) + captures + metadata + expectation row + ledger entries:

```bash
git commit -m "#300 M3: qoder 1.1.59 live TTY capture — profile, recognizer, fixtures (atomic)"
```

### Task 10: Recognizer hardening from the frozen captures

**Files:**
- Modify: `cmd/internal/wrapcmd/composer_recognizers.go` (or the qoder profile's inline `recognize`)
- Test: `cmd/internal/wrapcmd/composer_recognizers_test.go` (spec over snapshots)

- [ ] **Step 1: Failing spec first** — snapshot cases built from `composer.raw` (active) and `overlay.raw` (inactive) via the same construction helpers the claude/muse recognizer tests use. If the spec exposes brittleness (recognizer fires on the overlay, or depends on state the snapshot can't pin), refine the recognizer and re-run Task 9 Step 3's live conformance to confirm it still fires on real bytes.
- [ ] **Step 2:** `go test ./cmd/internal/wrapcmd` green; if the spec now proves discrimination that Task 9 Step 6 had to record as a gap, retire that entry in the same commit. Commit `#300 M3: qoder composer recognizer spec (from captured <signal>)`.

### Task 11: Overlay markers (evidence-gated)

**Files:**
- Modify: `cmd/internal/wrapcmd/harness_tty.go` (or a new `qoder` detector block following `detectAgyOverlayOpen`)
- Test: extend the marker table tests (`agyPickerMarkers`/`musePickerMarkers` test shapes)

- [ ] **Step 1:** From `overlay.raw`, extract the verbatim picker strings (permission prompts **and** any selection/AskUserQuestion menu — atlas aspect 2's muse lesson: a missing selection marker reproduces as "Enter inserts newline"). Add a `qoderPickerMarkers` set + `detectQoderOverlayOpen` wiring `pickerActive` exactly as agy/muse do; register `overlay` on the profile. OSC-based detection preferred if the capture shows one. If Task 9's Step 4 found no reachable declining screen, this task starts by driving one (same `PAIR_LIVE_SCENARIO` path).
- [ ] **Step 2:** Failing test first (marker table rows), implement, `go test ./cmd/internal/wrapcmd` green. If Task 9 took the no-overlay path and recorded a `ttyFixtureNegativeGaps["qoder"]` entry, drop it now that a declining state is captured (an entry plus a captured overlay.raw is an oracle error — `harness_tty_fixture_test.go:145-146`).
- [ ] **Step 3:** Commit `#300 M3: qoder overlay markers (permission + selection pickers)`.

### Task 12: `--session-id` mint decision (evidence-gated) + M3 boundary

**Files (if honored):** `cmd/internal/wrapcmd/wrap.go:2280` (`freshAgentInvocation`), test beside it.

- [ ] **Step 1:** Live check: `qoder --session-id 00000000-0000-4000-8000-000000000000 -p "hi"` then `ls ~/.qoder/projects/*/00000000-0000-4000-8000-000000000000.jsonl`. If qoder honors a caller-minted id, extend the claude-only mint:

```go
if agent == "claude" || agent == "qoder" {
	sessionID = freshUUID()
	...
	freshArgs = append(freshArgs, "--session-id", sessionID)
}
```

giving the watcher deterministic identity from launch. If not honored, leave it: the watcher establishes identity from the completed round (aspect 3 default). Record the finding in `## Log` either way.

- [ ] **Step 2:** M3 boundary: `go test ./...` green; `sdlc milestone-close --issue 300 --milestone M3`; log the verdict.

### Task 13: Slug generation (M4)

**Files:**
- Modify: `cmd/internal/model/model.go:64-101` (`DefaultModel`, `Run`)
- Test: `cmd/internal/model/model_test.go` (extend)

- [ ] **Step 1: Pin the invocation live**

Run: `qoder -p "Reply with exactly: ok"` (from `/tmp`, to confirm no workspace coupling and see the output shape). Also `qoder --list-models` → pick the cheapest fast model as the default. Record both in `## Log`.

- [ ] **Step 2: Failing test** — dispatch table: `Run(Request{Agent: "qoder", …})` routes to `runQoder` (test via the package's existing command-capture seam; see how `runAgy` is tested).

- [ ] **Step 3: Implement**

```go
// runQoder invokes `qoder -p` for headless summarization. TempDir avoids the
// workspace's agent context; PAIR_SLUG_NESTED=1 guards recursion.
func runQoder(r Request) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout())
	defer cancel()
	cmd := exec.CommandContext(ctx, "qoder", "-p", "--model", r.Model, r.Prompt)
	cmd.Stdin = strings.NewReader(r.Input)
	cmd.Dir = os.TempDir()
	cmd.Env = append(os.Environ(), "PAIR_SLUG_NESTED=1")
	out, err := cmd.Output()
	return string(out), err
}
```

`Run`: `case "qoder": return runQoder(r)`. `DefaultModel`: `if agent == "qoder" { return DefaultQoderModel }` with the const pinned from `--list-models`.

- [ ] **Step 4:** `go test ./cmd/internal/model` green; live: `pair-slug` on a real qoder session produces a slug (manual smoke; slug-parse telemetry `fired` in the adapt log). Commit `#300 M4: qoder slug generation via qoder -p`.

### Task 14: Prompt glyphs (capture-gated, where each consumer applies)

**Files:**
- Modify: `nvim/scrollback.lua:370-375` (`PROMPT_PATTERN_BY_AGENT`)
- Modify: `cmd/internal/wrapcmd/orientation.go:202-212` (`orientationPromptOK` map)
- Possibly modify: `cmd/internal/changelogcmd/distill.go:17-21` (`promptGlyphChar` — see Step 1)

- [ ] **Step 1:** From `composer.raw` (and a scrollback capture if needed), take qoder's user-prompt glyph. Register in `scrollback.lua` and `orientationPromptOK` unconditionally — both tables cover every harness (orientation has a dedicated muse branch, so qoder joins the map). For `distill.go`: the map deliberately has **no muse row** and `glyphFor` falls back to claude's glyph (`distill.go:33-38`); establish whether that omission is deliberate (distill doesn't consume muse sessions) or lagging, and apply the same test to qoder — register only if the consumer actually reads qoder sessions. If the check reveals muse is consumed-but-absent, record it in `## Log` as a peer finding for its own issue; don't fix it here.

```lua
qoder  = [[^<glyph>]],
```

```go
prompt := map[string]string{"claude": "❯", "codex": "›", "agy": ">", "qoder": "<glyph>"}[agent]
```

```go
"qoder": "<glyph>",
```

- [ ] **Step 2:** Tests: scrollback glyph has no Go test (Lua) — smoke via Alt+b in M5; `orientationPromptOK` and (if registered) the distill glyph rows extend their existing tests. `go test ./cmd/internal/wrapcmd ./cmd/internal/changelogcmd` green.
- [ ] **Step 3:** Commit `#300 M4: qoder prompt glyph (scrollback + orientation[, distill])`.

### Task 15: Settings (aspect 6, static)

**Files:**
- Modify (config, outside repo): `~/.qoder/settings.json`
- Possibly create: workspace-local qoder settings for `../ariadne` alignment (evidence first)

- [ ] **Step 1:** Inspect qoder's settings schema for a command-allowlist surface (`qoder --help` full dump; check whether `--permission-mode`/settings support per-tool allowlists like `.claude/settings.json`). If an allowlist exists, register the standard set: `git`, `make`, `sdlc`, `lsof`, `zellij`. If only `trustDirectories` + `--permission-mode` exist, that *is* qoder's permission surface — document that in `## Log` and stop (no invented config).
- [ ] **Step 2:** Verify `trustDirectories` covers the pair workspace (already true) and add `../ariadne` if continuous cross-repo testing needs it (operator decision).
- [ ] **Step 3:** No signal, no test — this is static config (atlas aspect 6). Log the outcome.

### Task 16: M4 boundary

- [ ] `go test ./...` green; `sdlc milestone-close --issue 300 --milestone M4`; log the verdict.

---

## Chunk 4: Milestone M5 — end-to-end, couch, docs

### Task 17: Standalone pair live smoke

- [ ] **Step 1:** `pair` boots qoder in the two-pane layout (Zellij + nvim draft). Verify: plain Enter inserts a newline in the composer; Alt+Enter sends; a permission picker confirms on plain Enter; mouse scroll is smooth; Alt+b jumps between user prompts; the composer survives a resize.
- [ ] **Step 2:** Restart-in-place (Alt+n) round-trips the session; `pair resume <tag>` cold-resumes it (established binding from the completed round — aspect 3).
- [ ] **Step 3:** `doctor/doctor.sh` on the session's adapt log: `return-remap` has a healthy fired:bypass ratio, zero `overlay-detect` near-misses, `session-id` fired, `slug-parse` fired. Any near-miss `detail` is a new marker string — add it (Task 11's set) and note it.
- [ ] **Step 4:** Log the smoke evidence in `## Log` (the close gate's `--verified` cites it).

### Task 18: Couch round-trip

- [ ] **Step 1:** `couch` start form lists qoder (registry-derived — should have worked since M1); start a hosted qoder thread on a repo path.
- [ ] **Step 2:** Live → park → cold resume through the couch switcher; switch-agent from an existing claude/muse thread to qoder and back. Verify the parked/live projection shows correct states (scanner-fed).
- [ ] **Step 3:** Log evidence; any failure is a finding against §0's claims — investigate before patching couch (couch is agent-agnostic by design; a qoder-specific couch fix is a red flag).

### Task 19: Docs + atlas sweep

**Files:**
- Modify: `README.md` (agent rosters — locate via the Step 1 sweep, not a hand inventory)
- Modify: `atlas/index.md`, `atlas/couch.md:321`, `atlas/session-identity.md:15`, `atlas/how-to-bring-up-a-new-harness-cli.md` (parity note if anything was learned), `doctor/README.md` + `doctor/SKILL.md` (agent lists)
- Modify: `CHANGELOG.md`

- [ ] **Step 1:** Sweep every agent-roster restatement case-insensitively and separator-agnostically — `rg -n -i 'claude|codex|agy|muse' README.md atlas/ doctor/ CHANGELOG.md` — keeping only roster-shaped hits (prose enumerations, tables, help text). README's rosters are Title-case and slash-joined (e.g. "(claude/codex/agy)"), so a lowercase list pattern misses them. The sweep output, not a hand-copied inventory, is the worklist; update all hits in one commit.
- [ ] **Step 2:** Atlas: no new architecture expected, but if capture revealed a new terminal shape/protocol fact, extend `atlas/terminal.md`; note qoder in `atlas/session-identity.md`'s storage-root table if one exists.
- [ ] **Step 3:** Commit `#300 M5: docs sweep — qoder joins every agent roster`.

### Task 20: Close

- [ ] **Step 1:** `go test ./...` + `make -f Makefile.local test-native-terminal-ci` if terminal-adjacent code changed (M3 did).
- [ ] **Step 2:** Update the issue `## Log` with per-milestone evidence; `sdlc close --issue 300 --verified '<smoke + suite evidence>'` (omit `--actual`; close measures it).

---

## Notes for the executor

- **Capture-first is absolute, and it fixes the order it permits** (Tasks 9-11, 14): no recognizer, marker, or glyph without a capture beside it — but the capture harness requires a positive-gated profile with a live-validated recognizer before it records anything (Task 9). The bootstrap recognizer is authored from the live screen and must make the classifier report `recognized` on real bytes before the capture is written; hand-authored fixture bytes are how Codex's dead `48;2;57;57;57` gate happened (atlas aspect 2).
- **Don't touch couch** (except its pinning tests): every couch behavior derives from the launcher registry and sessioninventory (§0). A qoder-specific couch change means the registry contract broke — fix the contract.
- **Fail-closed ordering**: registry (M1) → scanner (M2) → profile (M3) is the safe sequence; each milestone leaves main working with the later surfaces failing closed (verified in Task 1 Step 5). Within M3 there is no keymap-only fail-closed intermediate — the wrapcmd oracle admits no such state (`harness_tty_fixture_test.go:93-96,134-136`), so the profile lands positive-gated in the same commit as its captures.
- **Fixture hygiene**: newest qoder version dir only under `testdata/tty/`; when qoder self-updates and a recapture lands, delete the old dir (one capture set ≈ 45% of the wrapcmd suite's runtime — atlas aspect 2).
- When Part B implementation begins, re-run `sdlc change-code --issue 300` — this plan's M1-M5 boundaries mean the full flow (plan-quality + estimate at entry, per-milestone reviews).

---

## Revisions

### 2026-09-21 — plan-quality review round 1 (fresh-context qoder, ariadne plan-quality prompt): Important + Minor addressed

- **Important (harness-test-oracle-mismatch):** M3 Tasks 9-12 resequenced. The keymap-only fail-closed profile was not a capturable or committable state: `TestHarnessTTYLiveConformance` hard-fatals without a `commands` row (`harness_tty_live_test.go:547-556`), the capture's startup predicate requires an existing positive-gated profile with non-nil `recognize` (`:216-218`; `configureHarnessTTY` drops the terminal for non-positive gates, `wrap.go:1565-1580`), and the fixture oracle errors on fixtures-for-non-gated and fatals on gate-without-fixtures (`harness_tty_fixture_test.go:93-96,134-136`). Task 9 now lands the commands row + bootstrap profile/recognizer (authored from the live screen, validated by the classifier before any capture) + captures + metadata + the `TestComposerReturnExpectationMatchesProfile` row (`:803-819`) + oracle-read ledger entries in ONE commit; Task 10 is the recognizer spec over the frozen captures; Task 11 keeps the overlay markers; Task 12 is session-id + M3 boundary.
- **Minor (roster-sweep-pattern-undermatches):** The docs sweep's regex replaced by a case-insensitive, separator-agnostic sweep (README rosters are Title-case and slash-joined; the old pattern matched zero README lines and the line inventory was stale). The glyph task now registers qoder only where each consumer applies — `distill.go` deliberately lacks muse and falls back to claude (`distill.go:33-38`) — instead of unconditionally in all three maps.

### 2026-09-21 — plan-quality review round 2 (fresh-context qoder): both prior findings addressed; two new Minors folded in

- Verdict: INFO. PQ-1 disposed `addressed` (M3 resequencing verified at all four registration points against the code); PQ-2 disposed `addressed` (sweep and per-consumer glyph verified). Approved to start.
- **Minor A folded (harness-test-oracle-mismatch, 2nd in family):** Task 9 Step 6 now names two more wrapcmd registration points — `TestHarnessTTYProfileRegistry` (`harness_tty_test.go:20-31`, silent staleness by omission) and the `ttyFixtureReactionGaps` oracle (`harness_tty_fixture_test.go:160-176`: every positively gated harness presses Return in a driven scenario or carries an entry; a `pressesReturn` scenario is preferred since it evidences Done-when's Enter/Alt+Enter claim directly). Step 3's failure-mode labels corrected per PQ-1's residual note: a never-firing recognizer times out `state=waiting` (`:592-594`); the `no recognized prefix` fatal is the byte-replay path (`:595-598`).
- **Minor B folded (agent-dispatch-registration-gap):** Task 6 adds `validAgent` (`model.go:444-451` — it gates `ValidateScannerState`/`ScannerStateFact`/catalog/CLI, so the scanner's own first test needs it); Task 7 adds `observationNativeID` (`target.go:199-227`) and replaces the factually wrong "(Delegation is already agent-agnostic)" parenthetical — without the qoder case, `AgentSessionExists("qoder", …)` stays false and the watcher's `TargetNewLaunch` discovery (`target.go:133`) finds no new sessions.

### 2026-09-21 — M1 boundary review (fresh-context claude): REWORK — fixes landed as M1 review-fix commits

Verdict REWORK on window `08e9ec02..d1f36450` (Critical BR-8, Important BR-9/BR-10, Minors BR-11–BR-14; BR-1–BR-7 carried from plan-quality). Code fixes live in the `#300 M1:` review-fix commits; the plan-side deltas supersede stale body text below.

- **BR-8 (Critical, refactor-changes-sibling-agent-behavior):** `freshVariadicOption` made strictly per-agent (claude's list stays claude-only; qoder gets `--tools`; default false) and the codex regression the review measured (`codex --add-dir /x resume abc`) is pinned in `TestValidateFreshAgentArgs` plus a direct `TestFreshVariadicOptionIsPerAgent` table.
- **BR-9 (Important, resume-form-recognized-but-not-stripped):** qoder's accepted resume forms are enumerated once (`explicitResumeForms` in `createlogic.go`) and stripped at both persist sites (`persistedConfigArgs` now strips `-r` in both forms; `sessionwatch.StripResumeArgs` now strips `-r` and inline `--resume=`), with a failing-first `TestQoderShortResumeRoundTrip` over all three forms.
- **BR-10 (Important, agent-dispatch-registration-gap):** `TestAgentInventoryParityWithSessionTables` (launcher) ranges `AgentInventory()` over the sessioninventory scanner/CLI and sessionwatch tables, with a named known-gap entry for qoder that M2 must delete; the intermediate `schema_near_miss` diagnostic is pinned in the CLI result matrix (`qoder known gap` row).
- **BR-11 (Minor, hand-restated-registry):** `supportedAgents` in `sessioninventory/runcli.go` is now the session-side single list — `validAgent` (`slices.Contains`) and the CLI usage line derive from it; atlas §0 and checklist item 1 corrected to name the actual sites.
- **BR-12 (Minor, qoder-branch-copies-claude):** the extractor loop and the fresh-args cluster scan are table-parameterized per agent (`explicitResumeForms`, `freshAgentSpecs`); a valueless `--resume` followed by a flag no longer returns the flag as the id (claude pinned too).
- **BR-13 (Minor, unreachable-branch-unbacked-claim):** `--tools` added to qoder's value-flag list so the variadic branch is reachable and its test row exercises it.
- **BR-14 (Minor, plan-prose-restates-diff):** Task 1 now also covers `validAgent`, `runcli.go`'s `supportedAgents`/usage derivation and the golden JSON; **Task 6 drops its stale `validAgent`/`runcli.go` rows** (plan lines 383, 389, 478, 480), and **Task 2's implemented value-flag set follows `qoder --help`** as amended by BR-13 (`-p` and `-d` are bools, not value-taking).
- **BR-2 fold (carried, agent-dispatch-registration-gap):** context-meter usage for qoder (`usage.go:60` is claude-only) is a stated non-goal for #300 (see non-goals below); the create-path twin `shouldMintClaudeSessionID` (`agentargs.go:198`) is Task 12 scope alongside the restart path.
- **BR-3 fold (carried, harness-test-oracle-mismatch):** Task 9's capture policy is a neutral cwd with a stateable account-identity scrub, and the machine-neutral oracle (`assertFixtureIsMachineNeutral`) is satisfied by its failure message, not by a hand-listed oracle set.
- **BR-4 fold (carried, plan-prose-restates-diff):** the pre-M1 task bodies below stay as the record; the divergences that matter are enumerated in this entry.
- **BR-5 fold (carried, unbacked-existing-behavior-claim):** the live conformance entry point is `TestLiveNativeSessionShapeConformance` (`PAIR_LIVE_NATIVE_SESSIONS=1`), not the plan's `TestConformanceLive`.
- **BR-6 fold (carried, milestone-boundary-granularity):** M5's boundary is the final `sdlc close` — no separate `milestone-close` for Task 20.
- **Non-goals for #300 (BR-7 fold):** qoder `--remote`/`--teleport` cloud sessions, subagent transcript resume, the context-meter usage surface (BR-2), claude-only progress-OSC lifecycle authority, and a per-tool allowlist for qoder.
