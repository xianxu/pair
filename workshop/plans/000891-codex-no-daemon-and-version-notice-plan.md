# Codex private app-server launch + version-change notice Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A wrapped Codex 0.164+ start never reaches the shared background server (so the unanswerable
"incompatible feature settings" prompt cannot appear), and Couch reports a Codex version change to the hub once.

**Architecture:** The launcher probes `codex --version` (read-only, bounded), records it in
`PAIR_AGENT_VERSION`, and prepends `--no-daemon` when the version supports it. The wrapper copies the version
into the ready record. Couch, on each successful fresh registration, compares the recorded version against a
durable per-agent baseline (`agent-versions.json`) and publishes one notice when it differs.

**Tech Stack:** Go; zellij; existing `readiness`, `couchcore.OSReadyReader`, `NoticeClass`/`RouteNotice`
notice pipeline; `strictjson`; atomic namespace-file stores.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `parseAgentVersion` | `cmd/internal/launcher/agent_version.go` | new |
| `versionAtLeast` | `cmd/internal/launcher/agent_version.go` | new |
| `AgentVersion` ready-record field | `cmd/internal/readiness/record.go` | modified |
| `AgentVersionStore` + `AgentVersionObserver` | `cmd/internal/couchcore/agent_versions.go` | new |
| `NoticeAgentVersionChanged` | `cmd/internal/couchcore/notice_route.go` | new |

- **`parseAgentVersion` / `versionAtLeast`** — `codex --version` prints `codex-cli 0.164.0`; parse the last
  token only when it matches `X.Y.Z[-…]`, and compare numerically. Unparseable output yields `""` and
  `versionAtLeast` is false, so an unknown CLI never gains a flag it may not support.
  - Relationships: 1:1 with a launch probe.
  - DRY rationale: first occurrence of "which agent CLI versions get which launch policy"; the axis widens
    here if another agent ever needs a version-gated flag.
- **`AgentVersion` ready-record field** — optional `agent_version` the wrapper copies from the launcher env.
  `readiness.Decode` already tolerates unknown members, so old and new binaries interoperate in both
  directions; `Validate` bounds it (≤64 bytes, valid UTF-8, no control characters).
- **`AgentVersionStore` + `AgentVersionObserver`** — the store is `namespace/agent-versions.json`
  (`{"versions":{"codex":"0.164.0"}}`), written atomically; the observer wraps a `FreshRegistration` func,
  reads the version from the ready record, and returns the previous version exactly once per change. The
  first observation for an agent is the silent baseline. Only one Couch process writes a namespace (the
  supervisor lease), so an in-process mutex is the sync; no flock.
  - Relationships: one store per Couch namespace; N:1 observer calls per launch poll (only a successful
    registration touches the store).
- **`NoticeAgentVersionChanged`** — a `NoticeClass`; it has no special route case, so `RouteNotice` gives it
  the default owning-hub behavior (feed + hub when the owner is enabled). Add it to the exhaustive class
  table in `notice_route_test.go`.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Runtime.AgentVersion` | `cmd/internal/launcher/runtime.go`, `osruntime.go` | new | `codex --version` |
| launch argv policy | `cmd/internal/launcher/createflow.go` | modified | agent argv |
| `publishAgentReady` | `cmd/internal/wrapcmd/wrap.go` | modified | ready record |
| `OSReadyReader.RecordedAgentVersion` | `cmd/internal/couchcore/switchcontext.go` | new | ready record |
| `messageService.watchAgentVersions` | `cmd/internal/couchcmd/agent_version_notice.go` | new | `c.FreshRegistration`, `publishSystemNotice` |

- **`Runtime.AgentVersion`** — `OSRuntime` runs `exec.CommandContext(ctx, agent, "--version")` under a 3s
  timeout, codex only; failure returns `""` and leaves the old path untouched. Injected as a new
  `AgentVersionOps` sub-interface of `launcher.Runtime` so `fakeRuntime` controls it (default `""` keeps every
  existing argv pin unchanged).
- **launch argv policy** — after `inlineModeArgs` and the `shell_snapshot` block: probe once, `rt.SetEnv(
  AgentVersionEnv, version)` whenever the probe succeeded (so the version is recorded even below the gate),
  and prepend `--no-daemon` when `versionAtLeast(version, {0,164,0})` and the caller did not already pass it.
  Applies to all Codex launches, not only `couchOwned`: a pair launch must not depend on a machine-global
  server another program owns. It never restarts or contacts the shared server.
- **`publishAgentReady`** — one more field read from `os.Getenv(launcher.AgentVersionEnv)`.
- **`OSReadyReader.RecordedAgentVersion`** — mirrors `RecordedChild`: reads the ready file alone and returns
  the recorded version with `ok=false` when absent.
- **`messageService.watchAgentVersions`** — installed in `startMessageServiceWithStartup` right after
  `installNoticeDelivery` (the first point where the console, the delivery binder and the launch observer all
  exist); wraps `c.FreshRegistration`, and on a version change calls `publishSystemNotice` with
  `Class: NoticeAgentVersionChanged`, `Kind: "status"`, `Thread: address`, `Label: tag`. Notices without a
  hub owner stay in the feed; the store still records the change, so the hub sees it once when it returns.
  Exception, by design: the first-ever `StartupFresh` launch happens before the service exists and is silent
  anyway (empty store = baseline).

---

### Task 1: Ready record carries the agent version

**Files:**
- Modify: `cmd/internal/readiness/record.go`
- Test: `cmd/internal/readiness/record_test.go`
- Modify: `cmd/internal/wrapcmd/wrap.go:627`
- Test: `cmd/internal/wrapcmd/readiness_test.go`

- [ ] **Step 1: Write the failing tests**

`record_test.go` — extend the round-trip test with `AgentVersion: "0.164.0"`, and add:

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
Expected: FAIL — unknown field `AgentVersion`, missing env plumbing.

- [ ] **Step 3: Implement**

`record.go`: add the field after `LaunchOrdinal`, and the bound in `Validate`:

```go
	// AgentVersion is the CLI version the launcher probed at launch (e.g.
	// codex "0.164.0"); empty when the probe is unavailable (pair#891).
	AgentVersion string `json:"agent_version,omitempty"`
```

```go
	case len(record.AgentVersion) > 64 || !utf8.ValidString(record.AgentVersion) ||
		strings.IndexFunc(record.AgentVersion, unicode.IsControl) >= 0:
		return fmt.Errorf("ready record: invalid agent version")
```

`wrap.go`, in the `readiness.Encode(readiness.ReadyRecord{…})` literal: `AgentVersion: os.Getenv(launcher.AgentVersionEnv),`.

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/internal/readiness ./cmd/internal/wrapcmd -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/readiness cmd/internal/wrapcmd
git commit -m '#891: ready record carries the probed agent version' --trailer 'Agent: qoder/-/-'
```

### Task 2: Launcher probes the version and opts Codex out of the shared server

**Files:**
- Create: `cmd/internal/launcher/agent_version.go`
- Test: `cmd/internal/launcher/agent_version_test.go`
- Modify: `cmd/internal/launcher/agent_command.go` (env const)
- Modify: `cmd/internal/launcher/runtime.go` (interface + composition)
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/createflow.go:822`
- Modify/Test: `cmd/internal/launcher/createflow_test.go` (fake), new `createflow_agent_version_test.go`

- [ ] **Step 1: Write the failing pure tests**

`agent_version_test.go`:

```go
func TestParseAgentVersion(t *testing.T) {
	for in, want := range map[string]string{
		"codex-cli 0.164.0\n": "0.164.0",
		"0.164.0":             "0.164.0",
		"codex-cli 0.164.0-alpha.1\n": "0.164.0-alpha.1",
		"":                    "",
		"codex-cli":           "",
		"codex-cli 0.164\n":   "",
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

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/launcher -run 'ParseAgentVersion|VersionAtLeast' -count=1`
Expected: FAIL — functions undefined.

- [ ] **Step 3: Implement the pure helpers**

`agent_version.go`:

```go
package launcher

import "regexp"

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

// argumentPresent reports an exact argv element.
func argumentPresent(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run pure tests**

Run: `go test ./cmd/internal/launcher -run 'ParseAgentVersion|VersionAtLeast' -count=1`
Expected: PASS.

- [ ] **Step 5: Write the failing launch tests**

In `createflow_test.go`: add `agentVersions map[string]string` to `fakeRuntime`, initialize it in
`newFakeRuntime`, and implement `func (f *fakeRuntime) AgentVersion(agent string) string { return
f.agentVersions[agent] }`.

New `createflow_agent_version_test.go`, mirroring `TestRequiredNativeResumeBindingLaunchesExactRootWithoutDefaults`
for setup:

```go
func codexResumeWithVersion(t *testing.T, version string, agentArgs []string) (LaunchOptions, *fakeRuntime, int, error) {
	// bindingStatuses["work|codex"]=BindingEstablished, establishedSessions["work|codex"]="native-root-1",
	// CouchThreadScope/Tag, ResumeRequired/RequiredSessionID as in the existing resume test;
	// rt.agentVersions["codex"] = version when version != "".
}

func TestCodexLaunchOptsOutOfTheSharedServer(t *testing.T) {
	// version "0.164.0": want launchArgsText == "--no-daemon --disable shell_snapshot resume native-root-1 --sandbox workspace-write --no-alt-screen"
	// and rt.env["PAIR_AGENT_VERSION"] == "0.164.0"
}

func TestCodexLaunchBeforeTheGateIsUnchanged(t *testing.T) {
	// version "0.163.0": want no "--no-daemon" token, and PAIR_AGENT_VERSION still "0.163.0"
}

func TestCodexLaunchHonorsAnExplicitNoDaemon(t *testing.T) {
	// version "0.164.0", AgentArgs begins with "--no-daemon": exactly one occurrence
}

func TestCodexLaunchWithoutAVersionProbeIsUnchanged(t *testing.T) {
	// version "": --disable shell_snapshot resume ... and no PAIR_AGENT_VERSION in rt.env
}
```

Use `argumentPresent` (or `strings.Fields`) on `launchArgsText(t, rt.env)` to assert occurrences.

- [ ] **Step 6: Run to verify failure**

Run: `go test ./cmd/internal/launcher -run 'CodexLaunch' -count=1`
Expected: FAIL — `fakeRuntime` lacks `AgentVersion`/`agentVersions`; argv lacks `--no-daemon`.

- [ ] **Step 7: Implement the seam**

`runtime.go`: add

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

Run: `go test ./cmd/internal/launcher -count=1`
Expected: PASS, including every pre-existing argv pin (default fake version is "").

- [ ] **Step 9: Commit**

```bash
git add cmd/internal/launcher
git commit -m '#891: launch Codex against a private app server at 0.164+' --trailer 'Agent: qoder/-/-'
```

### Task 3: Couch reads the recorded version

**Files:**
- Modify: `cmd/internal/couchcore/switchcontext.go` (after `RecordedChild`)
- Test: `cmd/internal/couchcore/switchcontext_test.go`

- [ ] **Step 1: Write the failing test**

Next to `TestFreshRegistrationAcceptsTheExactReadyRecord`, write the same fixture with
`AgentVersion: "0.164.0"`, then assert `reader.RecordedAgentVersion(ctx, record.Address, "codex", "attempt")`
returns `("0.164.0", true, nil)`; a record without the field returns `ok=false`; a stale nonce returns
`ok=false, err=nil`.

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/couchcore -run RecordedAgentVersion -count=1`
Expected: FAIL — method undefined.

- [ ] **Step 3: Implement**

```go
// RecordedAgentVersion names the agent CLI version the launch's ready file
// records (pair#891). ok is false when the record carries none.
func (r OSReadyReader) RecordedAgentVersion(ctx context.Context, address ThreadAddress, agent, attempt string) (string, bool, error) {
	ready, err := r.readReadyFile(ctx, address, agent, attempt)
	if errors.Is(err, errObsoleteReady) || (err == nil && ready == nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return ready.AgentVersion, ready.AgentVersion != "", nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/internal/couchcore -run 'RecordedAgentVersion|FreshRegistration' -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/couchcore/switchcontext.go cmd/internal/couchcore/switchcontext_test.go
git commit -m '#891: read the recorded agent version from the ready record' --trailer 'Agent: qoder/-/-'
```

### Task 4: Durable store + once-per-change observer

**Files:**
- Create: `cmd/internal/couchcore/agent_versions.go`
- Test: `cmd/internal/couchcore/agent_versions_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestAgentVersionStoreReportsEachChangeOnce(t *testing.T) {
	store := NewAgentVersionStore(t.TempDir())
	step := func(version string) (string, bool) {
		previous, changed, err := store.Observe("codex", version)
		if err != nil {
			t.Fatal(err)
		}
		return previous, changed
	}
	if previous, changed := step("0.163.0"); changed || previous != "" {
		t.Fatalf("first observation reported %q %t", previous, changed)
	}
	if _, changed := step("0.163.0"); changed {
		t.Fatal("same version reported")
	}
	if previous, changed := step("0.164.0"); !changed || previous != "0.163.0" {
		t.Fatalf("change = %q %t", previous, changed)
	}
	if _, changed := step("0.164.0"); changed {
		t.Fatal("change reported twice")
	}
	// A fresh instance reads the same baseline from disk.
	reopened := NewAgentVersionStore(store.path)
	if previous, changed := reopened.Observe("codex", "0.163.0"); !changed || previous != "0.164.0" {
		t.Fatalf("reopen change = %q %t", previous, changed)
	}
}

func TestAgentVersionObserverOnlyObservesSuccessfulRegistrations(t *testing.T) {
	// Inner returns (false,nil): Read must not be called.
	// Inner returns an error: Read must not be called.
	// Inner true but Read ok=false: store untouched (Observe on the next launch
	//     with a version still reports from the old baseline).
	// Inner true and Read "0.164.0": baseline silent, second call quiet, then
	//     Read "0.165.0" reports once with previous "0.164.0".
}

func TestAgentVersionStoreRefusesACorruptRecord(t *testing.T) {
	// write "{" to the path; Observe returns an error and does not overwrite.
}
```

Also a `-race`-friendly test: N goroutines observe the same changed version after a baseline and exactly one
report fires.

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/couchcore -run 'AgentVersion' -count=1`
Expected: FAIL — undefined.

- [ ] **Step 3: Implement `agent_versions.go`**

Follow `launchfailure_store.go`'s read shape (`provisionSafePath`, `unix.O_NOFOLLOW|O_NONBLOCK`, regular-file
check, 64 KiB limit, `strictjson.Decode`) and `writeAtomicBytesWithPattern(s.path, raw, ".agent-versions-*")`
after `provisionMkdirAll`. Public surface:

```go
type AgentVersionStore struct {
	path string
	mu   sync.Mutex
}

func NewAgentVersionStore(namespaceDir string) *AgentVersionStore
func (s *AgentVersionStore) Path() string

// Observe records version as agent's latest, returning the previous version
// with changed=true exactly once per change. The first observation is the
// silent baseline.
func (s *AgentVersionStore) Observe(agent, version string) (previous string, changed bool, err error)

// AgentVersionObserver wraps a fresh-registration observer: after each
// successful registration it reads the launch's recorded agent version and
// reports a change once.
type AgentVersionObserver struct {
	Inner  func(context.Context, ThreadAddress, string, string) (bool, error)
	Read   func(context.Context, ThreadAddress, string, string) (string, bool, error)
	Store  *AgentVersionStore
	Report func(address ThreadAddress, agent, previous, current string)
}
func (o AgentVersionObserver) Registered(ctx context.Context, address ThreadAddress, agent, attempt string) (bool, error)
```

`Registered` returns the inner result unchanged; telemetry failures never fail a registration.

File shape: `{"versions":{"codex":"0.164.0"}}`, `agent_versions` JSON tags, `Versions` normalized to an
empty map on read.

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/internal/couchcore -run 'AgentVersion' -count=1 -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/couchcore/agent_versions.go cmd/internal/couchcore/agent_versions_test.go
git commit -m '#891: durable agent-version baseline reports each change once' --trailer 'Agent: qoder/-/-'
```

### Task 5: One hub notice per version change

**Files:**
- Modify: `cmd/internal/couchcore/notice_route.go`
- Test: `cmd/internal/couchcore/notice_route_test.go`
- Create: `cmd/internal/couchcmd/agent_version_notice.go`
- Test: `cmd/internal/couchcmd/agent_version_notice_test.go`
- Modify: `cmd/internal/couchcmd/message_service.go:580`

- [ ] **Step 1: Write the failing tests**

`notice_route_test.go`: add `NoticeAgentVersionChanged` to the `classes` table (the default-route expectation
already covers it).

`agent_version_notice_test.go`:

```go
func TestWatchAgentVersionsPublishesOneNoticePerChange(t *testing.T) {
	s := &messageService{}
	var got []couchtty.Notice
	publish := func(n couchtty.Notice) { got = append(got, n) }
	s.noticeOutput.Store(&publish)
	c := &couchcore.Couch{FreshRegistration: func(context.Context, couchcore.ThreadAddress, string, string) (bool, error) {
		return true, nil
	}}
	version := "0.163.0"
	read := func(context.Context, couchcore.ThreadAddress, string, string) (string, bool, error) {
		return version, version != "", nil
	}
	s.watchAgentVersions(c, read, couchcore.NewAgentVersionStore(t.TempDir()))
	ctx, address := context.Background(), couchcore.ThreadAddress{RepoScope: "/repo", Tag: "work"}
	for i := 0; i < 3; i++ { _, _ = c.FreshRegistration(ctx, address, "codex", "n") }
	version = "0.164.0"
	for i := 0; i < 3; i++ { _, _ = c.FreshRegistration(ctx, address, "codex", "n") }
	if len(got) != 1 || got[0].Class != couchcore.NoticeAgentVersionChanged || got[0].Kind != "status" ||
		got[0].Thread != address || !strings.Contains(got[0].Body, "0.163.0") || !strings.Contains(got[0].Body, "0.164.0") {
		t.Fatalf("notices = %+v", got)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/couchcmd ./cmd/internal/couchcore -run 'WatchAgentVersions|NoticeRoute' -count=1`
Expected: FAIL — undefined class and method.

- [ ] **Step 3: Implement**

`notice_route.go`: `NoticeAgentVersionChanged NoticeClass = "agent-version-changed"`.

`agent_version_notice.go`:

```go
// watchAgentVersions reports an agent CLI version change once per change, from
// the ready record of each launch Couch observes (pair#891).
func (s *messageService) watchAgentVersions(c *couchcore.Couch, read func(context.Context, couchcore.ThreadAddress, string, string) (string, bool, error), store *couchcore.AgentVersionStore) {
	if c.FreshRegistration == nil {
		return
	}
	observer := couchcore.AgentVersionObserver{
		Inner: c.FreshRegistration,
		Read:  read,
		Store: store,
		Report: func(address couchcore.ThreadAddress, agent, previous, current string) {
			s.publishSystemNotice(couchtty.Notice{
				Class:  couchcore.NoticeAgentVersionChanged,
				Kind:   "status",
				Thread: address,
				Label:  string(address.Tag),
				Body: fmt.Sprintf("%s version changed from %s to %s since its last launch; check that wrapped launches still start", agent, previous, current),
			})
		},
	}
	c.FreshRegistration = observer.Registered
}
```

`message_service.go`, immediately after `service.installNoticeDelivery(console, startup)`:

```go
	service.watchAgentVersions(c, reader.RecordedAgentVersion, couchcore.NewAgentVersionStore(namespace))
```

- [ ] **Step 4: Run the packages**

Run: `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/couchcore/notice_route.go cmd/internal/couchcore/notice_route_test.go cmd/internal/couchcmd
git commit -m '#891: notify the hub once when the agent version changes' --trailer 'Agent: qoder/-/-'
```

### Task 6: Docs, manual check, close

**Files:**
- Modify: `atlas/codex-resume.md` (Codex launch flags), `atlas/couch.md` (namespace store + notice)
- Modify: `workshop/issues/000891-…md` (`## Log`)

- [ ] **Step 1: Atlas**

`atlas/codex-resume.md`: state the launch policy — codex ≥ 0.164.0 gets `--no-daemon` (private app server)
and `PAIR_AGENT_VERSION` rides into the ready record; below the gate / unparseable probe → unchanged argv.
`atlas/couch.md`: `agent-versions.json` in the store layout and the one-notice-per-change rule.

- [ ] **Step 2: Focused + package tests**

Run: `go test ./cmd/internal/launcher ./cmd/internal/readiness ./cmd/internal/wrapcmd ./cmd/internal/couchcore ./cmd/internal/couchcmd -count=1`
Expected: PASS.

- [ ] **Step 3: Manual check (real Codex 0.164, shared server left running)**

This machine's shared Codex server is already running with other settings (the 10-25 stall). Do **not**
restart it and do not send it any keys. In a scratch dir:

```bash
d=$(mktemp -d "${TMPDIR:-/tmp}/nodaemon.XXXXXX") || exit 1
# fixed path (wrapped launch):
script -q "$d/fixed.log" codex --no-daemon --sandbox read-only -C "${d:?}" &
sleep 8; kill %1 2>/dev/null
# control (current wrapped argv):
script -q "$d/control.log" codex --sandbox read-only -C "${d:?}" &
sleep 8; kill %1 2>/dev/null
grep -c 'Background server has incompatible feature settings' "$d/fixed.log" "$d/control.log"
```

Expected: fixed → 0, control → 1 (proving the mismatch is present and the flag avoids it). Record both
counts and the Codex version in `## Log`. If the TUI cannot be driven headlessly, fall back to the launcher
argv test plus the `--help` evidence and say so in the Log.

- [ ] **Step 4: Affected-reader inventory and suite**

Run: `make test-affected-inventory`; audit every named reader before using its printed per-file repair
command (never refresh hashes in bulk). Then, on the pushed head: `sdlc local-check run --name suite`.

- [ ] **Step 5: Close, PR, merge, report**

`sdlc close --issue 891 --verified '<tests + manual check evidence>'`, `sdlc pr`, `sdlc merge --yes`; report
the outcome to ops:0.

## Estimate

Proposed `estimate_hours: 3` (two packages, a durable store with tests, one manual check). Derived after the
plan-quality gate, as the flow requires.
