# Codex private app-server launch (Part A) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A wrapped Codex ≥ 0.164 launch runs without the machine's shared background server (so the
unanswerable "incompatible feature settings" prompt cannot appear), and every launch records the CLI version it
ran.

**Architecture:** The launcher probes `codex --version` once (read-only, bounded), exports the parsed version as
`PAIR_AGENT_VERSION`, and prepends `--no-daemon` when the version supports it. The wrapper copies the version
into the ready record. Part B (the once-per-change hub notice and its baseline store) is #894.

**Tech Stack:** Go; zellij; `readiness` ready record; `strictjson`; the existing launcher Runtime seam.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `parseAgentVersion`, `versionAtLeast` | `cmd/internal/launcher/agent_version.go` | new |
| `AgentVersion` ready-record field | `cmd/internal/readiness/record.go` | modified |
| `codexBoolGlobalOption` table entry | `cmd/internal/launcher/agentargs.go` | modified |

- **`parseAgentVersion` / `versionAtLeast`** — `codex --version` prints `codex-cli 0.164.0`; parse the last
  token only when it matches `X.Y.Z[-…]`, compare numerically. Unparseable output yields `""` and
  `versionAtLeast` is false, so an unknown CLI never gains a flag it may not support. The existing `hasFlag`
  keeps a caller-supplied `--no-daemon` from being duplicated (a prompt word after `--` is not a flag).
- **`AgentVersion` ready-record field** — optional `agent_version`, bounded (≤64 bytes, valid UTF-8, no control
  characters). `Decode` already tolerates unknown members, so the record interoperates across an upgrade in
  both directions.
- **`codexBoolGlobalOption` entry** — the single table the wrapper's fresh-start strip walks. Without
  `--no-daemon` in it, `stripCodexResumeSubcommand` stops at the new leading flag and a fresh start inside a
  resumed Codex pane would silently keep `resume <id>`. The flag and the table entry land together.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Runtime.AgentVersion` | `cmd/internal/launcher/runtime.go`, `osruntime.go` | new | `codex --version` |
| launch argv policy | `cmd/internal/launcher/createflow.go` | modified | agent argv |
| `publishAgentReady` | `cmd/internal/wrapcmd/wrap.go` | modified | ready record |

- **`Runtime.AgentVersion`** — `OSRuntime` runs `exec.CommandContext(ctx, agent, "--version")` under a 3s
  timeout, codex only; failure returns `""` and leaves the old path untouched. Injected as a new
  `AgentVersionOps` sub-interface of `launcher.Runtime`; the fake's default `""` keeps every existing argv pin
  unchanged.
- **launch argv policy** — after `inlineModeArgs` and the `shell_snapshot` block: probe once, `rt.SetEnv(
  AgentVersionEnv, version)` whenever the probe succeeded (the version is recorded even below the gate), and
  prepend `--no-daemon` when `versionAtLeast(version, {0,164,0})` and the caller did not already pass it.
  Applies to all Codex launches, not only `couchOwned`: a pair launch must not depend on a machine-global
  server another program owns. It never contacts or restarts the shared server.
- **`publishAgentReady`** — one more field read from `os.Getenv(launcher.AgentVersionEnv)`.

### Deferred: Part B (#894)

`OSReadyReader.RecordedAgentVersion`, a durable `namespace/agent-versions.json` baseline, `AgentVersionObserver`
on `c.FreshRegistration`, and `NoticeAgentVersionChanged` were designed with Part A and seam-verified; they are
filed as unclaimed #894 with the design summary in its body. Nothing in this plan depends on them.

---

### Task 1: Ready record carries the agent version

**Files:**
- Modify: `cmd/internal/readiness/record.go`
- Test: `cmd/internal/readiness/record_test.go`
- Modify: `cmd/internal/wrapcmd/wrap.go:661`
- Test: `cmd/internal/wrapcmd/readiness_test.go`

- [ ] **Step 1: Write the failing tests**

`record_test.go` — add `AgentVersion: "0.164.0"` to the round-trip fixture, and:

```go
func TestReadyRecordBoundsAgentVersion(t *testing.T) {
	base := ReadyRecord{Tag: "t", Agent: "codex", Session: "s", Nonce: "n", PID: 1}
	for _, bad := range []string{strings.Repeat("9", 65), "0.164\x00.0", "0.164\n.0"} {
		record := base
		record.AgentVersion = bad
		if _, err := Encode(record); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	ok := base
	ok.AgentVersion = "0.164.0"
	if _, err := Encode(ok); err != nil {
		t.Fatal(err)
	}
}
```

`readiness_test.go` — in `TestPublishAgentReadyWritesRecordFromPairEnv` set
`t.Setenv("PAIR_AGENT_VERSION", "0.164.0")` and assert `got.AgentVersion == "0.164.0"`.

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/readiness ./cmd/internal/wrapcmd -run 'ReadyRecord|PublishAgentReady' -count=1`
Expected: FAIL — unknown field `AgentVersion`; env not plumbed.

- [ ] **Step 3: Implement**

`record.go`, after `LaunchOrdinal`:

```go
	// AgentVersion is the CLI version the launcher probed at launch (e.g.
	// codex "0.164.0"); empty when the probe is unavailable (pair#891).
	AgentVersion string `json:"agent_version,omitempty"`
```

and in `Validate`'s switch:

```go
	case len(record.AgentVersion) > 64 || !utf8.ValidString(record.AgentVersion) ||
		strings.IndexFunc(record.AgentVersion, unicode.IsControl) >= 0:
		return fmt.Errorf("ready record: invalid agent version")
```

`wrap.go`, in the `readiness.Encode(readiness.ReadyRecord{…})` literal:
`AgentVersion: os.Getenv(launcher.AgentVersionEnv),`.

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/internal/readiness ./cmd/internal/wrapcmd -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/readiness cmd/internal/wrapcmd
git commit -m '#891: ready record carries the probed agent version' --trailer 'Agent: qoder/-/-'
```

### Task 2: Launcher probe, `--no-daemon`, and the Codex flag table

**Files:**
- Create: `cmd/internal/launcher/agent_version.go`
- Test: `cmd/internal/launcher/agent_version_test.go`
- Modify: `cmd/internal/launcher/agent_command.go` (env const)
- Modify: `cmd/internal/launcher/runtime.go` (interface + composition)
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/createflow.go:822`
- Modify/Test: `cmd/internal/launcher/createflow_test.go` (fake), new `createflow_agent_version_test.go`
- Modify/Test: `cmd/internal/launcher/agentargs.go:106`, `cmd/internal/launcher/agentargs_test.go`

- [ ] **Step 1: Write the failing pure tests**

`agent_version_test.go`:

```go
func TestParseAgentVersion(t *testing.T) {
	for in, want := range map[string]string{
		"codex-cli 0.164.0\n":         "0.164.0",
		"0.164.0":                     "0.164.0",
		"codex-cli 0.164.0-alpha.1\n": "0.164.0-alpha.1",
		"":                            "",
		"codex-cli":                   "",
		"codex-cli 0.164\n":           "",
	} {
		if got := parseAgentVersion(in); got != want {
			t.Fatalf("parse(%q) = %q want %q", in, got, want)
		}
	}
}

func TestVersionAtLeast(t *testing.T) {
	min := [3]int{0, 164, 0}
	for version, want := range map[string]bool{
		"0.164.0": true, "0.164.1": true, "0.165.0": true, "1.0.0": true,
		"0.163.9": false, "0.163.0": false, "": false, "junk": false, "0.164.0-alpha.1": true,
	} {
		if got := versionAtLeast(version, min); got != want {
			t.Fatalf("versionAtLeast(%q) = %t want %t", version, got, want)
		}
	}
}
```

`agentargs_test.go`, beside the existing `stripCodexResumeSubcommand` cases:

```go
if got := FreshAgentArgs("codex", []string{"--no-daemon", "--disable", "shell_snapshot", "resume", "id1"}); !reflect.DeepEqual(got, []string{"--no-daemon", "--disable", "shell_snapshot"}) {
	t.Fatalf("fresh start kept the resume binding: %v", got)
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/launcher -run 'ParseAgentVersion|VersionAtLeast|FreshAgentArgs|CodexResume' -count=1`
Expected: FAIL — helpers undefined; the fresh-start test keeps `resume id1`.

- [ ] **Step 3: Implement the pure helpers**

`agent_version.go`:

```go
package launcher

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// agentVersionProbeTimeout bounds the read-only `<agent> --version` probe; a
// wedged install must never delay a launch past it.
const agentVersionProbeTimeout = 3 * time.Second

// codexNoDaemonVersion is the first Codex CLI that runs a shared background
// server and offers --no-daemon (pair#891).
var codexNoDaemonVersion = [3]int{0, 164, 0}

var agentVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)

// parseAgentVersion reads the version token from a `<agent> --version` line
// (`codex-cli 0.164.0`); anything else is "".
func parseAgentVersion(out string) string {
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return ""
	}
	last := fields[len(fields)-1]
	if !agentVersionPattern.MatchString(last) {
		return ""
	}
	return last
}

// versionAtLeast compares numeric release components; a pre-release suffix
// sorts with its release. Unparseable input is never at least anything, so an
// unknown CLI does not gain a flag it may not support.
func versionAtLeast(version string, want [3]int) bool {
	release, _, _ := strings.Cut(version, "-")
	release, _, _ = strings.Cut(release, "+")
	parts := strings.Split(release, ".")
	if len(parts) != 3 {
		return false
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return false
		}
		if n != want[i] {
			return n > want[i]
		}
	}
	return true
}

func argumentPresent(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
```

`agentargs.go` `codexBoolGlobalOption`: add `"--no-daemon"` to the case list. Swept readers of the same walk:
`freshValueOption` handles value options only (a boolean needs no entry), and the native-resume selector does
not parse leading flags — `codexResumeCommandIndex` is the only reader of the bool table.

- [ ] **Step 4: Run the pure + parser tests**

Run: `go test ./cmd/internal/launcher -run 'ParseAgentVersion|VersionAtLeast|FreshAgentArgs|CodexResume' -count=1`
Expected: PASS.

- [ ] **Step 5: Write the failing launch tests**

In `createflow_test.go`: add `agentVersions map[string]string` to `fakeRuntime`, initialize it in
`newFakeRuntime`, and implement `func (f *fakeRuntime) AgentVersion(agent string) string { return
f.agentVersions[agent] }`.

New `createflow_agent_version_test.go` reusing the resume setup from
`TestRequiredNativeResumeBindingLaunchesExactRootWithoutDefaults` (`bindingStatuses["work|codex"] =
BindingEstablished`, `establishedSessions["work|codex"] = "native-root-1"`, Couch thread scope/tag,
`ResumeRequired`/`RequiredSessionID`):

```go
func TestCodexLaunchOptsOutOfTheSharedServer(t *testing.T) {
	// rt.agentVersions["codex"] = "0.164.0"
	// want launchArgsText == "--no-daemon --disable shell_snapshot resume native-root-1 --sandbox workspace-write --no-alt-screen"
	// and rt.env["PAIR_AGENT_VERSION"] == "0.164.0"
}

func TestCodexLaunchBeforeTheGateIsUnchanged(t *testing.T) {
	// "0.163.0": no "--no-daemon" token; PAIR_AGENT_VERSION still "0.163.0"
}

func TestCodexLaunchHonorsAnExplicitNoDaemon(t *testing.T) {
	// "0.164.0" with AgentArgs beginning "--no-daemon": exactly one occurrence
}

func TestCodexLaunchWithoutAVersionProbeIsUnchanged(t *testing.T) {
	// "": exact pre-change argv and no PAIR_AGENT_VERSION in rt.env
}
```

Assert token occurrence with `argumentPresent(strings.Fields(launchArgsText(t, rt.env)), "--no-daemon")`.

- [ ] **Step 6: Run to verify failure**

Run: `go test ./cmd/internal/launcher -run 'CodexLaunch' -count=1`
Expected: FAIL — `fakeRuntime` lacks the method; argv lacks `--no-daemon`.

- [ ] **Step 7: Implement the seam**

`runtime.go`:

```go
// AgentVersionOps probes the installed agent CLI's version, read-only.
type AgentVersionOps interface {
	// AgentVersion returns the installed CLI version ("0.164.0"), or "" when
	// the probe is unavailable or the agent has no version probe.
	AgentVersion(agent string) string
}
```

and embed `AgentVersionOps` in `Runtime`.

`osruntime.go`:

```go
func (r OSRuntime) AgentVersion(agent string) string {
	if agent != "codex" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), agentVersionProbeTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, agent, "--version").Output()
	if err != nil {
		return ""
	}
	return parseAgentVersion(string(out))
}
```

`agent_command.go`:

```go
// AgentVersionEnv carries the probed agent CLI version from the launcher to the
// wrapper's ready record (pair#891).
const AgentVersionEnv = "PAIR_AGENT_VERSION"
```

`createflow.go`, after the `commandArgs` `shell_snapshot` block and before `EncodeAgentCommand`:

```go
	agentVersion := ""
	if agent == "codex" {
		agentVersion = rt.AgentVersion(agent)
		if agentVersion != "" {
			rt.SetEnv(AgentVersionEnv, agentVersion)
		}
		// A shared Codex background server started with other feature settings
		// stops the launch at a prompt a Couch slot cannot answer; a private
		// server cannot mismatch. Never restart or contact the shared one.
		if versionAtLeast(agentVersion, codexNoDaemonVersion) && !argumentPresent(agentArgs, "--no-daemon") {
			commandArgs = append([]string{"--no-daemon"}, commandArgs...)
		}
	}
```

- [ ] **Step 8: Run the launcher package**

Run: `go test ./cmd/internal/launcher ./cmd/internal/readiness ./cmd/internal/wrapcmd -count=1`
Expected: PASS, including every pre-existing argv pin (default fake version is "").

- [ ] **Step 9: Commit**

```bash
git add cmd/internal/launcher
git commit -m '#891: launch Codex against a private app server at 0.164+' --trailer 'Agent: qoder/-/-'
```

### Task 3: Docs, manual check, close

**Files:**
- Modify: `atlas/codex-resume.md`
- Modify: `workshop/issues/000891-…md` (`## Log`)

- [ ] **Step 1: Atlas**

`atlas/codex-resume.md`: the launch policy — codex ≥ 0.164.0 gets `--no-daemon` (private app server) and
`PAIR_AGENT_VERSION` rides into the ready record; below the gate or an unparseable probe → unchanged argv. One
line: an in-pane fresh start reuses the launch-time env/argv, so a mid-session Codex upgrade keeps the old
version record until the next launch.

- [ ] **Step 2: Measure the probe cost**

Run: `time codex --version` (3 runs) and record the typical latency in `## Log` — it sits on the launch path.

- [ ] **Step 3: Focused + package tests**

Run: `go test ./cmd/internal/launcher ./cmd/internal/readiness ./cmd/internal/wrapcmd -count=1`
Expected: PASS.

- [ ] **Step 4: Manual PTY check (real Codex 0.164, shared server left running)**

This machine's shared Codex server is already running with other settings (the 10-25 stall). Do **not**
restart it and do not send it any keys. In a scratch dir:

```bash
d=$(mktemp -d "${TMPDIR:-/tmp}/nodaemon.XXXXXX") || exit 1
script -q "$d/fixed.log" codex --no-daemon --sandbox read-only -C "${d:?}" &
sleep 8; kill %1 2>/dev/null
script -q "$d/control.log" codex --sandbox read-only -C "${d:?}" &
sleep 8; kill %1 2>/dev/null
grep -c 'Background server has incompatible feature settings' "$d/fixed.log" "$d/control.log"
```

Expected: fixed → 0, control → 1 (proves the mismatch is present and the flag avoids it). Record both counts
and the Codex version in `## Log`. If the TUI cannot be driven headlessly, fall back to the launcher argv test
plus the `--help` evidence and say so in the Log.

- [ ] **Step 5: Affected-reader inventory and suite**

Run: `make test-affected-inventory`; audit every named reader before using its printed per-file repair command
(never refresh hashes in bulk). On the pushed head: `sdlc local-check run --name suite`.

- [ ] **Step 6: Close, PR, merge, report**

`sdlc close --issue 891 --verified '<tests + manual check evidence>'`, `sdlc pr`, `sdlc merge --yes`; report
the outcome to ops:0.

## Estimate

Part A is scoped inside the quick-flow shell (no estimate on that flow). If the close measures it outside,
derive an itemized `## Estimate` before the estimate gates.

## Revisions

### 2026-10-25 — split Part B out (TL steer)

Reason: land the stall fix first, in the cheap window; the notice is independent follow-up work.
Delta: #891 keeps Task 1 (ready-record carrier), Task 2 (launcher probe + `--no-daemon`) and the manual check;
the Couch-side design moved to unclaimed #894 (which cites this plan). Tasks 3–5 of the first draft were
condensed into the deferred summary above.

### 2026-10-25 — plan-quality finding folded in

Finding PQ-1 (Important): `--no-daemon` was missing from `codexBoolGlobalOption`, so the wrapper's fresh-start
strip would stop at the new leading flag and keep `resume <id>`. Delta: Task 2 now changes that table and tests
`FreshAgentArgs` with the flag leading; the plan names the table as the single source for Codex global flags.
Minors folded in: snippet imports, measured probe latency, stale version after a mid-session upgrade.

### 2026-10-25 — `argumentPresent` replaced by `hasFlag`

Reason: the new helper ignored the `--` boundary, so a prompt word `--no-daemon` would have suppressed the
opt-out (and it duplicated the existing flag predicate, ARCH-DRY). Delta: the seam calls
`hasFlag(agentArgs, "--no-daemon")`; the dedicated helper is gone, the test notes read `hasFlag`, and a new
launch test pins a post-`--` prompt word. The snippets above keep the original shape.

### 2026-10-25 — boundary-review findings folded in

BR-4 (Important): `PAIR_AGENT_VERSION` was set only after a successful codex probe, so a value inherited from
an outer pane could reach the ready record for a failed probe or for another agent. Delta: the seam sets it on
every launch (empty when unknown); a test pre-seeds a stale value for both a failed codex probe and a claude
launch. Minors: the probe command sets `WaitDelay` (a forked CLI shim keeping stdout open cannot hold the 3s
bound), and a standalone (non-Couch) codex launch test pins the `--no-daemon` gate. BR-1 (claimed missing
table entry) is factually wrong: `--no-daemon` is in `codexBoolGlobalOption` and
`TestFreshAgentArgsWalksPastNoDaemon` passes; disputed with that run.
