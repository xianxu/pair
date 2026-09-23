# Qoder Harness Integration Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the `qoder` CLI (v1.1.60; the design facts below were verified against 1.1.59 and the harness self-updated before capture — see the M3 execution-deltas Revisions entry) up to parity with `claude`/`codex`/`agy`/`muse` across all surfaces of `atlas/how-to-bring-up-a-new-harness-cli.md` (§0 registry, aspects 1–7), in both hosts (standalone pair and couch).

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
| Claude-family record transition (`validateClaudeFamilyDelta`) | `cmd/internal/sessioninventory/scan_claude.go` | modified |
| `ValidateQoderDelta` | `cmd/internal/sessioninventory/scan_qoder.go` | new |
| `resumeToken`/`composeResumeArgs` qoder case | `cmd/internal/launcher/agentargs.go` | modified |
| `extractExplicitResume` qoder case | `cmd/internal/launcher/createlogic.go` | modified |
| `ValidateFreshAgentArgs`/`freshValueOption` qoder case | `cmd/internal/launcher/fresh_args.go` | modified |
| Qoder composer recognizer | `cmd/internal/wrapcmd/composer_recognizers.go` | new (capture-gated) |
| `DefaultModel` qoder row | `cmd/internal/model/model.go` | modified |
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
| `scanClaudeFamily` / `scanClaudeFamilyFile` and `ScanQoder` | `cmd/internal/sessioninventory/scan_claude.go`, `scan_qoder.go` | modified / new | `Runtime` native roots, files, and record reads |
| `qoder` TTY profile registration | `cmd/internal/wrapcmd/harness_tty.go` | new entry | proxy keymap, recognizer, and overlay dispatch |
| OSRuntime qoder root | `cmd/internal/sessioninventory/runtime_os.go` | new row | `~/.qoder/projects` filesystem |
| Live TTY capture fixtures | `cmd/internal/wrapcmd/testdata/tty/qoder/1.1.59/` | new | real qoder PTY output |
| Scanner conformance fixtures | `cmd/internal/sessioninventory/testdata/native/qoder/v1/qoder-projects/` | new | real qoder transcript shapes (sanitized) |
| `ProviderQoderJSONLV1` contract | `cmd/internal/sessioninventory/provider_contract.go` | new row | reviewed append-only producer contract |
| Watcher/ledger/CLI membership | `sessionwatch/sessionwatch.go`, `sessionledger/record.go`, `sessioninventory/runcli.go` | modified | agent allowlists |
| `detectQoderOverlayOpen` | `cmd/internal/wrapcmd/wrap.go` | new | proxy-owned rolling overlay state |
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
- Possibly modify: `cmd/internal/changelogcmd/distill.go:17-21` (`promptGlyphChar` — see Step 1)

Superseded by the M3 review-fix round: `orientationPromptOK`'s qoder branch landed in M3 Task 9 and now reads the shared `qoderPromptGlyphs` authority (`composer_recognizers.go`), so no glyph is registered here from a second source — any M4 consumer derives from that map.

- [ ] **Step 1:** Take qoder's user-prompt glyph from the shared `qoderPromptGlyphs` authority (`composer_recognizers.go`; captured bytes are `>` in default mode, `*` in yolo). Register in `scrollback.lua`. For `distill.go`: the map deliberately has **no muse row** and `glyphFor` falls back to claude's glyph (`distill.go:33-38`); establish whether that omission is deliberate (distill doesn't consume muse sessions) or lagging, and apply the same test to qoder — register only if the consumer actually reads qoder sessions. If the check reveals muse is consumed-but-absent, record it in `## Log` as a peer finding for its own issue; don't fix it here.

```lua
qoder  = [[^<glyph>]],
```

```go
"qoder": "<glyph>",
```

- [ ] **Step 2:** Tests: scrollback glyph has no Go test (Lua) — smoke via Alt+b in M5; (if registered) the distill glyph rows extend their existing tests. `go test ./cmd/internal/wrapcmd ./cmd/internal/changelogcmd` green.
- [ ] **Step 3:** Commit `#300 M4: qoder prompt glyph (scrollback[, distill])`.

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
- [ ] **Step 3 (added by the M4 review fix round, BR-45):** Capture the settled-session footer into the distill pipeline: with the session idle, Alt+l must distill to a **no-op** — a live footer that `isFooterChrome` doesn't recognise leaks into the anchor and forces `FullRedistill` on every press (the #58 class; qoder's footer matched no row as of M4, so the degraded state shipped). Extend `isFooterChrome` + the `TestTrimLiveTail` qoder rows with the captured shapes, then re-run the no-op press. If the smoke session can't produce the footer shapes, record what was observed in `## Log` and log the residual gap.
- [ ] **Step 4:** `doctor/doctor.sh` on the session's adapt log: `return-remap` has a healthy fired:bypass ratio, zero `overlay-detect` near-misses, `session-id` fired, `slug-parse` fired. Any near-miss `detail` is a new marker string — add it (Task 11's set) and note it.
- [ ] **Step 5:** Log the smoke evidence in `## Log` (the close gate's `--verified` cites it).

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

### 2026-09-21 — M1 advisories BR-15/BR-16 fixed at rule level (M2-prep commit daeb781b)

The M1 boundary shipped (round 3, SHIP) with two advisory Minors recorded for the close review; both stated rules rather than instances, so they are closed structurally before M2 starts.

- **BR-15 (resume-form-recognized-but-not-stripped, 3rd instance) — fixed.** The per-agent resume spellings now live in ONE table, `resumeform.Forms` (`cmd/internal/resumeform` — new leaf package; the shared table could not live in launcher because launcher already imports sessionwatch). `extractExplicitResume`, `persistedConfigArgs`, `sessionwatch.StripResumeArgs` and `ValidateFreshAgentArgs` (via `resumeform.Selector`, strictly per-agent so no sibling's spelling leaks — the BR-8 failure mode) all read it. The table adds claude/qoder glued short (`-r<id>`) and inline short (`-r=`) spellings, and a valueless space form now keeps the flag that follows it (`--resume --model m` no longer orphans `m`). `TestResumeFormTableRoundTrip` (launcher) and `TestEveryTableSpellingRoundTrips` (resumeform) range over the table, so extract/strip/validate cannot drift again; atlas checklist item 5 now names the table as a bring-up step.
- **BR-16 (agent-dispatch-registration-gap, 4th instance) — rule recorded, named site probed.** The parity test (`agent_parity_test.go`) now probes `sessionledger.ParseLedger` (the `isSupportedAgent` dispatch, record.go:478) and asserts a gap agent's records fail closed as malformed; the known-gap message names only probed sites. **Rule binding M2's Tasks 5–8:** every per-agent dispatch reachable from `AgentInventory()` must either derive from one exported list or be probed by the parity test; M2 must delete the qoder known-gap entry and flip `scansReal`/`watchable`/ledger acceptance together (NormalizeNativeEvent, ProviderContractFor, `observationNativeID` in target.go, the incremental_inventory switches, and the runtime_os native roots are the sites to wire and probe).

### 2026-09-21 — M2 boundary review (fresh-context claude): REWORK round 4 — fixes landed as M2 review-fix commits

Verdict REWORK on window `367610e7..fa89157c` (Critical BR-17, Important BR-18/BR-19/BR-20, Minors BR-21–BR-24). Code fixes in the `#300 M2:` review-fix commits; the plan-side deltas supersede stale body text below.

- **Claude-family core admissions as parameters (BR-20):** the extracted `scanClaudeFamily` / `validateClaudeFamilyDelta` carry qoder-only grammar admissions as parameters of the family spec — `acceptsMillis` (bounded 2000-01-01..9999-12-31) and `claudeFamilyNoiseTypes` (`workspace-directories`, `runtime-config`, `worktree-state`, `active-leaf`, `file-history-snapshot`). Claude's behavior is pinned unchanged: `TestIncrementalClaudeRejectsNumericTimestamp` asserts epoch-millis records still dispute on claude, and `TestNormalizeNativeEvent` carries claude negative rows for the four shared ignored types.
- **Task 7 filename correction (BR-24):** Task 7's Step 5 names `event_test.go` (not `events_test.go` as originally written).
- **Task 7 Step 5 evidence (live conformance):** qoder-only `pair session-inventory --agent qoder` against real `~/.qoder`: ok, 10 nodes, 7 roots, zero diagnostics. Whole-suite `TestLiveNativeSessionShapeConformance` fails on this machine because of pre-existing agy schema drift, unrelated to M2.
- **Task 8 Step 3 evidence (incremental conformance):** `TestValidateQoderDelta` and `TestIncrementalQoderMalformedSuffixFailsClosed` pin the cold and incremental paths; `TestQoderMillisTimestampBounds` pins the BR-19 epoch-millis window.
- **BR-16 rule as delivered (BR-18):** `sessioninventory.SupportedAgents()` is the exported single list. `TestEveryAgentDispatchParity` ranges it through `ObserveAgentMetadata` → `ProviderContractFor` → `ValidateTargetWork` → `NativeEventsFromRecords` for every agent (agy skips the incremental chain); `TestAdvanceTargetValidationPerAgent` ranges `SupportedAgents()` (skipping agy) through the production `AdvanceTargetValidation` with an appended record; `TestValidateTargetWorkRejectsUnknownAgent` pins the `ValidateTargetWork` fail-closed `default:` arm (asserts a `schema_near_miss` diagnostic via `artifactDiagnostic`); `TestAdvanceTargetValidationRejectsUnknownAgent` pins the `AdvanceTargetValidation` fail-closed `default:` arm (builds a prior from a real `ValidateTargetWork` result, overrides `State.Agent` to `"future"`, asserts `errors.Is(err, ErrArtifactChanged)` — arm-deletion mutation verified red). Both default arms use `artifactDiagnostic` for consistent diagnostic shape. `TestProviderContractFor`, `TestAppendOnlyProviderConformance`, `provider_live_fake_test.go`, `native_large_record_test.go`, `scan_fuzz_test.go`, and `TestQuerySessionCatalogLossProof` carry qoder rows.
- **BR-17 (Critical, resume-form-recognized-but-not-stripped):** `forbidsCluster` now consults `resumeform.ShortLetters(agent)` alongside `'c'`, so clusters like `-pr sid` are rejected for both claude and qoder. `TestResumeFormTableRoundTrip` ranges glued letters after a bool short (`-p<letter>`) and requires rejection.
- **BR-19 (Important, untrusted-input-parsed-without-bounds):** `claudeFamilySpec.acceptsMillis` bounds the epoch-millis path to 2000-01-01..9999-12-31; out-of-range values dispute the record visibly. `TestQoderMillisTimestampBounds` pins the window.
- **BR-20 (Important, refactor-changes-sibling-agent-behavior):** `claudeFamilyNoiseTypes` keeps qoder's bookkeeping near-miss set for claude; `file-history-snapshot` joins qoder's ignore set. `TestIncrementalClaudeRejectsNumericTimestamp` and the claude negative rows in `TestNormalizeNativeEvent` pin claude's prior behavior.
- **BR-21–BR-23 (Minors):** `file-history-snapshot` joins qoder's ignore set with evidence (53 near-miss records); `Strip(agent, args)` is strictly per-agent; `resumeform.Forms` stays exported for round-trip tests but behind the accessor surface.

### 2026-09-21 — M3 execution deltas (Tasks 9–12 as landed)

- **Task 9 version label:** the plan's example commit message names qoder 1.1.59; the installed harness reports 1.1.60, so the fixture directory is `qoder/1.1.60/` (per the plan's own newest-version-directory rule).
- **Task 9 capture path (addition):** driven captures are bounded by the new `trimmedHarnessTTYCapture` (`harness_tty_live_test.go`) to the capture's final synchronized paint block, and only when that block replays to the same Return decision as the whole stream (verified, never assumed; a harness with no synchronized updates or a differently-deciding final block keeps its full capture). Evidence: qoder's permission picker is the last 5.5KB of a 55KB capture and the question picker the last 6.6KB of 104KB; the slash menu correctly kept its full 13KB. Pinned by `TestTrimmedHarnessTTYCapture`.
- **Task 11 detector (addition beyond "exactly as agy/muse do"):** `detectQoderOverlayOpen` also scans the raw rolling buffer before its stripped-tail path. Evidence: the all-splits fixture replay went red at split 6426/6616 of `selection.raw` — a chunk boundary cut `\x1b[23m` between `Enter` and `select`, and per-chunk stripping keeps the truncated escape verbatim, severing the marker in the tail; the byte-contiguous rolling window cannot be corrupted that way (same haystack claude/codex scan for OSC).
- **Task 11 second family:** the question picker (`selection.raw`, AskUserQuestion UI) lands beside the permission picker (`overlay.raw`); `qoderPickerMarkers` carries both families' verbatim stripped strings (`AskingUser`, `Enterselect` added to the permission four). `ttyFixtureNegativeGaps["qoder"]` dropped; `ttyFixtureDiscriminationGaps`/`ttyFixtureReactionGaps` rewritten to what the captures prove and leave unproven.

### 2026-09-21 — M3 boundary review (fresh-context claude): REWORK round 7 — fixes landed as M3 review-fix commits

Verdict REWORK on window `301c5381..1b1977c4` (Critical BR-28, Important BR-29–BR-32, plus minors). Code fixes in the `#300 M3:` review-fix commits; the plan-side deltas below supersede stale body text (Task 14 amended in place above, orientation landing moved into M3).

- **BR-28 (Critical, overlay-flag-rearmed-from-stale-input):** the raw window `detectQoderOverlayOpen` scans is now proxy-owned (`proxy.overlayRawTail`, bounded to `rollingTailLen`), not the chunk pump's loop-local `rolling`: `emitPlainCR` clears it beside `overlayTextTail` when the confirming Enter consumes `pickerActive`, so consumed picker bytes cannot re-arm the flag on a later chunk (previously: paint → Enter → one small chunk re-armed from the stale slice, and the next composer Enter submitted a draft). Pinned by `TestCheckOverlayOpen_QoderDoesNotRedetectStalePickerText` over both frozen captures (paint, `emitPlainCR`, small chunk, assert false), the qoder counterpart of the codex stale-text test.
- **BR-29 (Important, qoder-branch-copies-claude):** `ruledBoxComposerSpec` gained `promptCol` and `requireVisibleCursor`; `qoderComposerActive` is now a spec registration and `ruledBoxComposerActive` owns the only ruled-box loop (qoder: `promptCol` 1, `requireVisibleCursor` false, unbounded height; claude/muse/agy specs carry `requireVisibleCursor: true`). All qoder composer differential rows unchanged and green.
- **BR-30 (Important, hand-restated-registry):** `qoderPromptCol` and `qoderPromptGlyphs` (`{">", "*"}`) in `composer_recognizers.go` are the single authority; `orientationPromptOK`'s qoder branch reads the map rather than restating column and glyphs. **M4 glyph consumers (`scrollback.lua`, `distill.go`) derive from this authority — Task 14 amended above.**
- **BR-31 (Important, refactor-changes-sibling-agent-behavior):** the `claudeComposerRule` skip in `orientationComposerActive` is gated to qoder (which parks its hidden cursor on the closing rule), and the prompt column derives from `qoderPromptCol`. `TestOrientationRuleCellToleranceStaysPerProfile` pins the sibling negatives (claude/muse/agy flip true without the gate — mutation-verified — and codex stays false via its recognizer barrier) plus the qoder positive contrast.
- **BR-32 (Important, agent-dispatch-registration-gap, 7th in family):** `TestRunLaunchForcedCreateQoderMintProbesQoderSessions` pins the create-path mint: a `qoder|MINTED-1` collision retries to `MINTED-2` and appends `--session-id MINTED-2`; a `claude|MINTED-1` collision does not block qoder. Both subtests fail against a literal `"claude"` probe (mutation-verified).
- **Minors:** the `ttyFixtureExpectation` comment now says only `overlay.raw` takes the shared declining default (`selection.raw`'s row exists because the shared map does not name that file); the generic `forfuturesessions` marker was **dropped** from `qoderPickerMarkers` (a word-by-word paint glues that prose anywhere, arming `pickerActive` off agent output — the permission family keeps three specific strings) with the glued-prose negative row in `TestOverlayDetectorByAgent`; `atlas/architecture.md` no longer calls `--session-id` claude-only (the `MintsSessionID` set); `atlas/how-to-bring-up-a-new-harness-cli.md` records the prose-marker rule and `overlayRawTail`.

### 2026-09-21 — M3 review fix round 2 (round-9 findings BR-35..BR-40) — fixes landed as the M3 review-fix-round-2 commit

Verdict on round 9 was BLOCKED with two open Importants (BR-35, BR-36) plus four Minors; all six are fixed in the working tree this entry records. The round-8-bloc findings BR-28..BR-34 were already disposed `addressed` in round 9 and are not restated here.

- **BR-35 (Important, detector-carry-bounded-before-scan):** `detectQoderOverlayOpen` now scans `stripTerminalControls(prevTail+data)` **before** the carry is re-bounded to `rollingTailLen`, so a marker that straddles a chunk boundary inside an escape is found even when the second chunk carries kilobytes of trailing bytes. Pinned by `TestCheckOverlayOpen_QoderSplitFooterSurvivesLongSecondChunk`: the split lands inside `\x1b[23m` between `Enter` and `select`, and the second chunk is padded with 0/600/2000 filler bytes (the first chunk must not arm; the second must). Mutation-checked: restoring the bound-before-scan order reddens the 600 and 2000 rows. This **supersedes** the earlier atlas/plan wording that "the byte-contiguous rolling window cannot be corrupted" by a split — true of the bytes, false of the pre-fix scan order.
- **BR-36 (Important, hand-restated-registry, 5th instance):** the recorded rule ("when a harness registers, grep the previous newest harness — `grep -n -i muse atlas/*.md` — and extend every hit that enumerates sibling harnesses in the same commit") was executed over the whole `atlas/` tree. Extended: `atlas/architecture.md` :694 (keymaps, incl. Qoder's `\`-CR/bare-CR), :702 (three specs on one `ruledBoxComposerActive`, promptCol 1, `allowHiddenCursor`), :704 (conformance expectation), :903 (record parsing); `atlas/session-identity.md` :15/:24/:94; `atlas/index.md` :3; `atlas/how-to-bring-up-a-new-harness-cli.md` :5 + the resume-flag bullet (Claude/Qoder share `--resume`/`-r` incl. glued `-r<id>`, via `resumeform`) + the qoder marker bullet (BR-38 amendment). Sweep hits deliberately **not** extended, each with a checked reason: `couch.md:321` (M5 couch scope), how-to `PROMPT_PATTERN_BY_AGENT` snippet (M4 — `nvim/scrollback.lua` has no qoder row today, so the snippet is accurate), how-to's status-line mention (M5 verification sweep), `architecture.md:895` (`endOfTurnByAgent` lists only claude and the sentence is not a sibling enumeration; qoder is in the idle-floor group), `architecture.md:912` (`model.Run` defaults to `runClaude` for qoder — covered, not stale), `architecture.md:1060` (no agent list at that paragraph).
- **BR-37 (Minor, refactor-changes-sibling-agent-behavior, 5th instance):** the ruled-box spec field `requireVisibleCursor` was inverted to `allowHiddenCursor` (zero value = the prior behaviour of every existing spec: the visible cursor is required), and only Qoder's spec sets it. A spec that forgets the field now fails closed instead of failing open. The agy uncolored fallback's hidden-cursor decline is pinned by `TestOrientationUncoloredAgyRequiresVisibleCursor` (visible-cursor control row + hidden-cursor decline row over the captured box shape). Mutation-checked: a permissive guard reddens the hidden row.
- **BR-38 (Minor, overlay-marker-matches-agent-prose):** resolved by the atlas-amendment option: the title-case header `Permission Required` is a **deliberate exemption** to the "markers must not be ordinary English" rule — it titles *every* permission picker while the body strings pin only the one captured question, and the exposure is bounded because `pickerActive` is consumed by exactly one Enter (BR-28's reset covers the re-arm). The exemption's boundary is pinned live: `qoder spaced prose about permissions does not open overlay` (spaced prose must not arm). Mutation-checked: widening `qoderPickerMarkers` with a spaced marker reddens exactly that row.
- **BR-39 (Minor, agent-dispatch-registration-gap):** `harnessTTYProfile` gained `orientationPromptCol int` and `orientationRuleCellTolerant bool`; qoder's row sets `qoderPromptCol` / `true`, and `orientationComposerActive` reads the profile instead of comparing `p.agentBasename == "qoder"`. Both halves are pinned: zeroing `orientationRuleCellTolerant` reddens the qoder rule-cell row of `TestOrientationRuleCellToleranceStaysPerProfile`; zeroing `orientationPromptCol` reddens that row **and** the live-capture row of `TestOrientationCapturedComposersWithoutReturnRemap` (qoder `composer.raw` stops auto-submitting). `orientationPromptOK` keeps its agent-keyed glyph read — the glyph map is the shared authority (BR-30), not a dispatch.
- **BR-40 (Minor, hand-restated-registry / ARCH-DRY):** the four copies of the tail-carry block collapsed into `(*proxy).overlayVisible(data)` (nil-receiver safe: callers without a proxy get the plain strip); the three identical marker loops collapsed into `firstMarker(visible, markers)` (slice-order probing keeps the reason string deterministic). Muse deliberately keeps its own folded loop — it vary-cases markers across versions and the reason string must preserve the declared spelling.
- **Carried minima re-verified at HEAD** (for the next review round's disposition): BR-26/BR-27 fixes are present in commits (`14caa844`, `301c5381`) — `TestAdvanceTargetValidationPerAgent` ranges `sessioninventory.SupportedAgents()` (skipping agy), and both fail-closed default arms use `artifactDiagnostic`; BR-25's overclaim is now backed at HEAD by those same commits.
- **Verification:** full `go test ./...` EXIT=0 (wrapcmd 32.5s incl. both frozen-capture replays); every pin above mutation-checked red-then-restored; `git status --short` after the mutations shows only the intended files.

### 2026-09-21 — M3 close follow-up: M4 design note from the boundary review

The round-10 boundary review (SHIP) left one architectural note for M4 beyond the two M5-owned roster items (`atlas/couch.md:321`, `README.md` Return row + rosters): Task 14 owes qoder's glyphs to `nvim/scrollback.lua`'s `PROMPT_PATTERN_BY_AGENT` and to `distill.go`, and Lua cannot read `qoderPromptGlyphs` (the single authority from BR-30). Task 14 must name how that consumer stays in sync — a checked-in generated table or a parity test against the Go map — rather than restating `>`/`*` by hand.

### 2026-09-21 — M4 execution deltas (Task 13 as landed)

- **`DefaultQoderModel` pin:** `qoder --list-models` publishes no pricing metadata, so the default is the cheap/fast tier alias `Efficient` (verified live end-to-end), chosen over any specific `*-Flash` id so the pin survives model-generation churn.
- **Task 13 Step 2 reference corrected:** the plan said "see how `runAgy` is tested"; no `runAgy` test exists. The dispatch test uses the package's existing fake-binary seam (`TestRunCodexCLIWithoutAPIKey` pattern: script on PATH capturing argv/stdin/cwd).
- **Task 13 Step 4 live evidence:** the invocation shape is proven live through the production dispatch by the new gated test `TestRunQoderLiveConformance` (`PAIR_LIVE_QODER_MODEL=1`). The binding-driven `pair-slug` run on a real session (`slug-parse` fired in the adapt log) is exercised by M5 Task 17's standalone smoke — the first qoder pair session is created there; no binding can exist before it.

### 2026-09-21 — M4 execution deltas (Task 14 as landed)

- **The sync story M3 owed is a parity test, and it is named:** `TestScrollbackQoderPatternTracksPromptAuthority` (`cmd/internal/wrapcmd/scrollback_glyph_parity_test.go`) derives the expected Lua pattern from `qoderPromptGlyphs` (sorted, char-class-escaped) + `qoderPromptCol` and requires `nvim/scrollback.lua`'s `PROMPT_PATTERN_BY_AGENT` to carry exactly one matching qoder row. The Lua row is `qoder  = [=[^ [*>]]=],` — a leveled long string, because the pattern's own class-closing `]` fuses with a `]]` delimiter. Red-first (row absent ⇒ "found 0"), mutation-verified (removing `*` from the Go map reddens until the row follows). `nvim/scrollback_test.lua` gained the behavior block (col-1 `>`/`*` match; flush-left and extra-indented reject).
- **distill's qoder row was earned by a proven consumption chain, not assumed:** `PAIR_AGENT` → `RunChangelogCLI` → `distillerEnv` PCL_AGENT → `--agent` → `glyphFor`; `model.Run` supports qoder since Task 13. `"qoder": " >"` (leading space = captured col-1 echo; the submitted echo serializes with its leading space in `--plain` mode). Yolo `*` deliberately unregistered there (agy precedent: captured evidence covers default mode; a miss only widens lookback).
- **Findings carried, not fixed:** qoder's live footer matches none of `isFooterChrome` rows (`trimLiveTail` strips nothing → anchor-leak risk); the cleaned render must be captured from M5 Task 17's settled session before extending the recognizer — this is now an M5 Task 17 scope item alongside the Alt+b glyph smoke. Muse's distill absence + partial scrollback pattern is a peer finding for muse's own issue.
- **Side-quest 03abb18a (pre-existing red, not Task 14):** `tests/workbench-route-nvim-test.sh` left `PAIR_LAYOUT_MODE_PATH` unset in three init-loading invocations, so `nvim/init.lua`'s load-time `layout_write('small')` hit `io.open(nil)`; proven red at `de56ea58` in a detached baseline worktree, fixed test-side.

### 2026-09-21 — M4 execution deltas (Task 15 as landed)

- **The allowlist branch is the one that obtained** — Task 15's "if an allowlist exists" gate resolved YES, so Step 1's register-the-standard-set applies. Surface: `permissions.{allow,deny,ask}` (claude-style rule strings, `Bash(cmd:*)` prefix form) in `~/.qoder/settings.json` (user), `<repo>/.qoder/settings.json` (project), `<repo>/.qoder/settings.local.json` (local); the `--allowed-tools`/`--disallowed-tools` flags feed the same engine as `flagSettings`. The interactive settings dialog does not surface allow/deny/ask — JSON/flag-level only.
- **Decision points are the observable** (task's "no signal" note refined): every Bash permission decision logs `[permission-check] … point=<decisionPoint>` to `~/.qoder/logs/runs/*/qodercli.log`, which made the A/B evidence mechanical. Pre-state: only `git` was covered (by the heuristic `shell.readonly.allow`, not a rule); `make`/`sdlc`/`lsof`/`zellij` hit `shell.no_match.ask`. Post-state: all five `shell.rule_prefix.allow`.
- **`../ariadne` trust deferred, not decided**: no demonstrated cross-repo need in M1–M4; the plan marks it an operator decision, recorded as an open knob in the issue Log (revisit at M5 if the smoke needs it).

### 2026-09-21 — M4 boundary review round 10 (FIX-THEN-SHIP, BR-42..BR-45 + minors) — fixes landed as the M4 review-fix commit

The gate refused finalization with four open Importants; all are fixed before the re-run. Plan-side deltas:

- **BR-42 (agent-dispatch-registration-gap, 9th):** a registry row is not landed until a table test ranges the registry and drives every reader — `TestPromptGlyphRowsDriveBothReaders` ranges `promptGlyphChar` over `scanTurnBoundaries` **and** `trimLiveTail` with each agent's drawn box row; `trimLiveTail` now normalizes the glyph once (`strings.TrimSpace`), because the box row is matched on its trimmed form while the boundary regex keeps qoder's column-1 leading space. Red-first over the four rows (qoder only, pre-fix).
- **BR-43 (hand-restated-registry, 7th):** `changelogcmd.PromptGlyph(agent)` is the exported accessor for the registry; `TestDistillQoderGlyphTracksPromptAuthority` (wrapcmd) derives the expected value from `qoderPromptCol` + the default-mode glyph and asserts equality, and pins the yolo `*` omission from both ends (authority still carries it; distill must not). Mutation-verified: `qoderPromptCol 1→0` reddens both this and the Lua parity test.
- **BR-44 (headless-call-leaves-durable-residue):** `runQoder` passes `--no-session-persistence` (argv pinned red-first in `wantArgs`); live conformance re-run measured no new files in the TMPDIR project dir (the 16:00 pre-flag jsonl was not touched).
- **BR-45 (deferred-work-not-in-executing-task):** Task 17 gained Step 3 (settled-footer capture → `isFooterChrome` extension → no-op Alt+l verification), restated above in place; the M5 checklist now owns the carried #58-class risk instead of a Revisions paragraph only.
- **Minors:** (E) the Lua parity test's class escape now uses the Vim dialect (`\`, `\]`, `\-`, `\^` — the row is consumed by `vim.fn.search`; the Lua-pattern dialect was wrong for future glyphs) and the missing-file fatal names drift; (F) the allowlist **moved from user scope to `<repo>/.qoder/settings.local.json`** — measured: from the repo cwd `make` resolves `shell.rule_prefix.allow` (local scope applies), from `/tmp` the same command hits `shell.no_match.ask` (repo-bound), and the chained probe `git rev-parse … && mkdir …` hits `shell.no_match.ask` (no prefix-rule leak into compound tails, denied headless); `~/.qoder/settings.json` is back to its pre-M4 content (backups: `/tmp/qoder-settings-backup-1790033243.json`, `-1790034148.json`); (H) `TestHandleChunk_OscScannedBeforeCarryIsBounded` is table-driven over every OSC-reading profile (claude 777 + codex `9;Plan mode prompt:`); both rows mutation-verified red under the restored bound-first order.

### 2026-09-22 — Close review BR-50: correct pure/integration classification

The close review found the Core concepts table classified scanner IO and a subprocess as PURE. The classification now follows the function boundary (ARCH-PURE): `validateClaudeFamilyDelta` and `ValidateQoderDelta` are deterministic transitions over supplied records; `scanClaudeFamily`, `scanClaudeFamilyFile`, and `ScanQoder` read native roots and records through `Runtime` and are INTEGRATION. `DefaultModel` is a pure model-name choice; `runQoder` launches `qoder -p` and remains only in Integration points. The Qoder TTY profile registration and `detectQoderOverlayOpen` are also listed as INTEGRATION because they connect the pure snapshot recognizer to the proxy and its mutable overlay carry. Other Pure table rows were checked for external reads, process launches, and retained state; no further IO entry point remains there. This corrects plan taxonomy only; implementation and test behavior are unchanged.
