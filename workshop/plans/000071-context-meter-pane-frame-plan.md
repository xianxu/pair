# Per-Agent Context Meter in the Zellij Pane Frame — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show each agent pane's live context-window size as an absolute humanized token count in its zellij frame title — `claude (970k) [~/brain]` — read from the agent's real transcript, refreshed on a ~60s activity-gated poll.

**Architecture:** A pure Go core (`ContextTokens` reader + `Humanize`) reusing pair's existing transcript-path resolver (extracted to a shared package); a thin one-shot `pair-context` CLI that wires config/pane-file → resolver → reader → humanizer; the existing `pair-cmux-title.sh` poller generalized into an always-on `pair-title.sh` that owns both the cmux workspace title (when in cmux) and the per-pane zellij frame meter; a dedicated `pane-<tag>-<agent>.json` state file written at pane startup so the external poller can `zellij --session pair-<tag> action rename-pane --pane-id`.

**Tech Stack:** Go (module `github.com/xianxu/pair`, cmds under `cmd/`, built via `Makefile.local`), Bash (`bin/`), zellij KDL layout, `jq`.

**Spec:** `workshop/issues/000071-context-meter-pane-frame.md` (read its `## Spec` + `## Done when` first).

---

## Core Concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ContextTokens(agent string, r io.Reader) (int, bool)` | `cmd/internal/ctxmeter/ctxmeter.go` | new |
| `Humanize(n int) string` | `cmd/internal/ctxmeter/ctxmeter.go` | new |
| `Resolve(agent, sid, cwd, home string) string` | `cmd/internal/transcript/transcript.go` | new (extracted from `cmd/pair-slug/main.go`) |
| `SessionID(dataDir, tag, agent string) string` | `cmd/internal/transcript/transcript.go` | new (extracted) |
| `resolveTranscript` / `sessionID` / `claudePathEncoder` (local copies) | `cmd/pair-slug/main.go` | deleted (replaced by import) |

- **ContextTokens** — given an agent name and a reader over its transcript, returns current-context token occupancy and whether a count was found. Streams JSONL line-by-line and keeps the *last* qualifying record; never loads whole records' content beyond the small header fields it parses.
  - **Relationships:** 1:1 with a transcript stream; dispatches on `agent` to one of three per-agent rules (claude sum-of-three / codex `last_token_usage.input_tokens` / agy none).
  - **DRY rationale:** Single home for "what is this agent's current context size" — `pair-context` (and any future consumer, e.g. a statusline) calls it instead of re-deriving per-agent JSON shapes.
  - **Future extensions:** Returning the model id / `model_context_window` (codex carries it) to enable a true-% later; a fourth agent rule is one `case`.

- **Humanize** — formats a token count: `<1000` exact; `1000≤n<1_000_000` → `Nk` (round, half-up); `≥1_000_000` → `N.NM` (floor to one decimal).
  - **Relationships:** pure int→string; no dependency.
  - **DRY rationale:** one pinned formatting rule, locked by a table test, shared by every display surface.
  - **Future extensions:** a width/padding option if alignment in the frame ever matters.

- **Resolve / SessionID** — extracted verbatim from `cmd/pair-slug/main.go` so both `pair-slug` and the new `pair-context` derive transcript paths from one source (ARCH-DRY). `Resolve` maps `(agent, sid, cwd)`→on-disk path; `SessionID` reads `session_id` from `config-<tag>-<agent>.json`.
  - **Relationships:** `pair-slug` and `pair-context` both import them; `pair-slug` keeps its own `resolveLiveCodexTranscript` (the live-PID special case) and calls `Resolve` for the base path.
  - **DRY rationale:** kills the would-be duplicate of the path-encoding + config-read logic across two binaries.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `pair-context` CLI | `cmd/pair-context/main.go` | new | filesystem (config, pane file, transcript) |
| `pane-<tag>-<agent>.json` writer | `zellij/layouts/main.kdl` (in-pane startup) | new | filesystem write |
| `PAIR_PANE_CWD` export | `bin/pair` | new | env |
| `pair-title.sh` poller | `bin/pair-title.sh` (renamed from `bin/pair-cmux-title.sh`) | modified | zellij IPC + cmux + `pair-context` |
| poller spawn / existence / cleanup refs | `bin/pair` | modified | process lifecycle |
| build rule | `Makefile.local` | modified | `go build` |

- **pair-context CLI** — `pair-context <tag> <agent>` prints the humanized count (or nothing). Reads `session_id` from config and `cwd` from the pane file, resolves the transcript via `transcript.Resolve`, opens it, calls `ContextTokens`, prints `Humanize`. Any failure → prints nothing, exit 0 (tolerant, like `pair-slug`).
  - **Injected into:** the `pair-title.sh` poller calls it per pane; `ContextTokens`/`Resolve` stay pure and are unit-tested without it.
  - **Future extensions:** a `--json` mode emitting `{tokens, model}` for richer consumers.

- **pane-<tag>-<agent>.json** — `{pane_id, cwd, cwd_display}` written once at pane startup where `$ZELLIJ_PANE_ID` is in scope. Single writer (the in-pane `sh -c`), so it dodges the 3-writer race on `config-<tag>-<agent>.json`.
  - **Injected into:** read by the poller (pane_id, cwd_display) and by `pair-context` (cwd for claude path encoding).

- **pair-title.sh poller** — the existing per-tag poller, generalized: cmux gates become block-local (frame meter runs with or without cmux), and each active tick it loops the tag's panes and renames each zellij frame to `<agent> (<count>) [<cwd>]`.
  - **Injected into:** spawned by `bin/pair`; drives `pair-context` + `zellij` as subprocesses (faked in tests).

**Test surface.** `ContextTokens`/`Humanize`/`Resolve` get colocated table tests with inline/`testdata` fixtures (no mocks). `pair-context` gets a process-level test (build binary, temp `PAIR_DATA_DIR`, fixture transcript + config + pane file, assert stdout) mirroring `cmd/pair-slug/main_test.go`. `pair-title.sh` gets a shell test (fake `zellij` + fake `pair-context` on `PATH`, assert `rename-pane` args, activity-gate, unchanged-skip) mirroring `tests/cmux-title-poller-test.sh`.

---

## Chunk 1: Pure core (transcript resolver + context reader + humanizer)

### Task 1: Extract the transcript resolver into a shared package

**Files:**
- Create: `cmd/internal/transcript/transcript.go`
- Create: `cmd/internal/transcript/transcript_test.go`
- Modify: `cmd/pair-slug/main.go` (delete local `resolveTranscript`, `sessionID`, `claudePathEncoder`; import the package; keep `resolveLiveCodexTranscript`)

- [ ] **Step 1: Write the new package** with the three symbols moved verbatim (exported):

```go
// Package transcript resolves an agent's on-disk session transcript path and
// the session id pair recorded for it. Single source for both pair-slug and
// pair-context (ARCH-DRY) — extracted from cmd/pair-slug/main.go.
package transcript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ClaudePathEncoder mirrors nvim's `cwd:gsub('[./]', '-')` for the
// ~/.claude/projects/<encoded-cwd>/ directory name.
var ClaudePathEncoder = strings.NewReplacer(".", "-", "/", "-")

// SessionID reads the session id pair recorded for (tag, agent) in
// config-<tag>-<agent>.json (written by bin/pair / pair-session-watch.sh).
func SessionID(dataDir, tag, agent string) string {
	b, err := os.ReadFile(filepath.Join(dataDir, "config-"+tag+"-"+agent+".json"))
	if err != nil {
		return ""
	}
	var c struct {
		SessionID string `json:"session_id"`
	}
	if json.Unmarshal(b, &c) != nil {
		return ""
	}
	return c.SessionID
}

// Resolve returns the on-disk transcript path for (agent, sid), or "" if it
// can't be located. cwd is only needed for claude (project-dir encoding).
func Resolve(agent, sid, cwd, home string) string {
	switch agent {
	case "codex":
		matches, _ := filepath.Glob(filepath.Join(home, ".codex", "sessions", "*", "*", "*", "rollout-*"+sid+"*.jsonl"))
		if len(matches) > 0 {
			return matches[0]
		}
		return ""
	case "agy":
		return filepath.Join(home, ".gemini", "antigravity-cli", "brain", sid, ".system_generated", "logs", "transcript.jsonl")
	default: // claude
		return filepath.Join(home, ".claude", "projects", ClaudePathEncoder.Replace(cwd), sid+".jsonl")
	}
}
```

- [ ] **Step 2: Write a small test** pinning the claude path encoding + agy path:

```go
package transcript

import (
	"path/filepath"
	"testing"
)

func TestResolveClaudeEncodesCwd(t *testing.T) {
	got := Resolve("claude", "abc", "/Users/x/work.dir", "/home")
	want := filepath.Join("/home", ".claude", "projects", "-Users-x-work-dir", "abc.jsonl")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveAgy(t *testing.T) {
	got := Resolve("agy", "sid1", "", "/home")
	want := filepath.Join("/home", ".gemini", "antigravity-cli", "brain", "sid1", ".system_generated", "logs", "transcript.jsonl")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
```

- [ ] **Step 3: Update `cmd/pair-slug/main.go`** — delete the local `claudePathEncoder` (line 175), `sessionID` (84-96), `resolveTranscript` (179-194); add `"github.com/xianxu/pair/cmd/internal/transcript"` to imports; replace call sites: `sessionID(...)` → `transcript.SessionID(...)`, `resolveTranscript(...)` → `transcript.Resolve(...)`. Leave `resolveLiveCodexTranscript` and `codexRolloutRE` in place.

- [ ] **Step 4: Run the existing pair-slug suite + the new test** — Run: `cd /Users/xianxu/workspace/pair && go test ./cmd/pair-slug/... ./cmd/internal/transcript/...` — Expected: PASS (refactor preserves behavior). Also `go vet ./cmd/...`.

- [ ] **Step 5: Commit** — `git add cmd/internal/transcript cmd/pair-slug/main.go && git commit -m "#71 M1: extract transcript resolver to shared package (DRY)"`

### Task 2: `ContextTokens` reader (TDD, per agent)

**Files:**
- Create: `cmd/internal/ctxmeter/ctxmeter.go`
- Create: `cmd/internal/ctxmeter/ctxmeter_test.go`

- [ ] **Step 1: Write the failing test** with inline fixtures pinning each agent rule + the sidechain/synthetic filters:

```go
package ctxmeter

import (
	"strings"
	"testing"
)

func TestContextTokensClaude_SumsThreeInputs_SkipsSidechainAndSynthetic(t *testing.T) {
	// real turn (300) → sidechain (small) → synthetic (0). Want the last REAL one: 300.
	jsonl := strings.Join([]string{
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-opus-4-8","usage":{"input_tokens":100,"cache_creation_input_tokens":50,"cache_read_input_tokens":150}}}`,
		`{"type":"assistant","isSidechain":true,"message":{"model":"claude-opus-4-8","usage":{"input_tokens":1,"cache_creation_input_tokens":1,"cache_read_input_tokens":1}}}`,
		`{"type":"assistant","message":{"model":"<synthetic>","usage":{"input_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`,
	}, "\n")
	got, ok := ContextTokens("claude", strings.NewReader(jsonl))
	if !ok || got != 300 {
		t.Fatalf("got (%d,%v) want (300,true)", got, ok)
	}
}

func TestContextTokensCodex_LastTokenUsageNotTotal(t *testing.T) {
	jsonl := strings.Join([]string{
		`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":60287},"total_token_usage":{"input_tokens":38425074}}}}`,
		`{"type":"response_item","payload":{"type":"message"}}`,
	}, "\n")
	got, ok := ContextTokens("codex", strings.NewReader(jsonl))
	if !ok || got != 60287 {
		t.Fatalf("got (%d,%v) want (60287,true)", got, ok)
	}
}

func TestContextTokensAgy_None(t *testing.T) {
	if _, ok := ContextTokens("agy", strings.NewReader(`{"type":"PLANNER_RESPONSE"}`)); ok {
		t.Fatal("agy should yield no count")
	}
}

func TestContextTokensTolerant_GarbageLines(t *testing.T) {
	jsonl := "not json\n" + `{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":10,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\nalso not json"
	got, ok := ContextTokens("claude", strings.NewReader(jsonl))
	if !ok || got != 10 {
		t.Fatalf("got (%d,%v) want (10,true)", got, ok)
	}
}
```

- [ ] **Step 2: Run to verify it fails** — Run: `go test ./cmd/internal/ctxmeter/...` — Expected: FAIL (undefined `ContextTokens`).

- [ ] **Step 3: Implement** (note the large-line-safe reader — claude records can exceed `bufio.Scanner`'s default 64KB token):

```go
// Package ctxmeter reads an agent's current context-window occupancy (in
// tokens) from its transcript, and humanizes a token count for display.
package ctxmeter

import (
	"bufio"
	"encoding/json"
	"io"
)

// ContextTokens streams a transcript and returns the current-context token
// occupancy from the LAST qualifying record, plus whether one was found.
// Tolerant: unparseable lines are skipped.
func ContextTokens(agent string, r io.Reader) (int, bool) {
	br := bufio.NewReader(r)
	last, found := 0, false
	for {
		line, err := br.ReadBytes('\n') // ReadBytes, not Scanner — records can be MB-sized
		if len(line) > 0 {
			if n, ok := lineTokens(agent, line); ok {
				last, found = n, true
			}
		}
		if err != nil {
			break // io.EOF or read error → stop
		}
	}
	return last, found
}

func lineTokens(agent string, line []byte) (int, bool) {
	switch agent {
	case "codex":
		var r struct {
			Type    string `json:"type"`
			Payload struct {
				Type string `json:"type"`
				Info struct {
					Last struct {
						InputTokens int `json:"input_tokens"`
					} `json:"last_token_usage"`
				} `json:"info"`
			} `json:"payload"`
		}
		if json.Unmarshal(line, &r) != nil || r.Type != "event_msg" || r.Payload.Type != "token_count" {
			return 0, false
		}
		return r.Payload.Info.Last.InputTokens, true
	case "agy":
		return 0, false // no usable token source
	default: // claude
		var r struct {
			Type       string `json:"type"`
			IsSidechain bool  `json:"isSidechain"`
			Message    struct {
				Model string `json:"model"`
				Usage struct {
					Input    int `json:"input_tokens"`
					CacheCreate int `json:"cache_creation_input_tokens"`
					CacheRead   int `json:"cache_read_input_tokens"`
				} `json:"usage"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &r) != nil || r.Type != "assistant" || r.IsSidechain || r.Message.Model == "<synthetic>" {
			return 0, false
		}
		return r.Message.Usage.Input + r.Message.Usage.CacheCreate + r.Message.Usage.CacheRead, true
	}
}
```

- [ ] **Step 4: Run to verify pass** — Run: `go test ./cmd/internal/ctxmeter/...` — Expected: PASS.

- [ ] **Step 5: Commit** — `git add cmd/internal/ctxmeter && git commit -m "#71 M1: ContextTokens per-agent transcript reader"`

### Task 3: `Humanize` (TDD, pinned rule)

**Files:**
- Modify: `cmd/internal/ctxmeter/ctxmeter.go`
- Modify: `cmd/internal/ctxmeter/ctxmeter_test.go`

- [ ] **Step 1: Write the failing table test** (pins the spec's boundary cases):

```go
func TestHumanize(t *testing.T) {
	cases := []struct{ n int; want string }{
		{0, "0"}, {999, "999"},
		{1000, "1k"}, {397556, "398k"}, // round half-up
		{999999, "1000k"},              // k-branch can emit 4 digits
		{1000000, "1.0M"}, {1490000, "1.4M"}, // M-branch floors
	}
	for _, c := range cases {
		if got := Humanize(c.n); got != c.want {
			t.Errorf("Humanize(%d)=%q want %q", c.n, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails** — Run: `go test ./cmd/internal/ctxmeter/... -run Humanize` — Expected: FAIL (undefined `Humanize`).

- [ ] **Step 3: Implement** (append to `ctxmeter.go`, add `"math"` + `"strconv"` + `"fmt"` imports):

```go
// Humanize formats a token count per the spec's pinned rule.
func Humanize(n int) string {
	switch {
	case n < 1000:
		return strconv.Itoa(n)
	case n < 1_000_000:
		return strconv.Itoa(int(math.Round(float64(n)/1000))) + "k"
	default:
		return fmt.Sprintf("%.1fM", math.Floor(float64(n)/100_000)/10)
	}
}
```

- [ ] **Step 4: Run to verify pass** — Run: `go test ./cmd/internal/ctxmeter/...` — Expected: PASS.

- [ ] **Step 5: Commit** — `git add cmd/internal/ctxmeter && git commit -m "#71 M1: Humanize token count (pinned k/M rule)"`

> **Chunk 1 review:** dispatch plan/code review per AGENTS.md before crossing into Chunk 2.

---

## Chunk 2: The `pair-context` CLI + build wiring

### Task 4: `pair-context` one-shot

**Files:**
- Create: `cmd/pair-context/main.go`
- Create: `cmd/pair-context/main_test.go`

- [ ] **Step 1: Write the failing process-level test** (mirrors `cmd/pair-slug/main_test.go`: build binary into TempDir, set up `PAIR_DATA_DIR` with config + pane file + a fixture claude transcript, run, assert stdout `398k`):

```go
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildPairContext(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "pair-context")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

func TestPairContext_Claude(t *testing.T) {
	bin := buildPairContext(t)
	home := t.TempDir()
	data := filepath.Join(home, "data")
	cwd := filepath.Join(home, "repo")
	// config (sid) + pane file (cwd) + transcript
	enc := strings.NewReplacer(".", "-", "/", "-").Replace(cwd)
	proj := filepath.Join(home, ".claude", "projects", enc)
	mustMkdir(t, data); mustMkdir(t, cwd); mustMkdir(t, proj)
	mustWrite(t, filepath.Join(data, "config-T-claude.json"), `{"session_id":"sid1"}`)
	mustWrite(t, filepath.Join(data, "pane-T-claude.json"), `{"pane_id":"7","cwd":"`+cwd+`","cwd_display":"~/repo"}`)
	mustWrite(t, filepath.Join(proj, "sid1.jsonl"),
		`{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":397556,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`)

	cmd := exec.Command(bin, "T", "claude")
	cmd.Env = append(os.Environ(), "HOME="+home, "PAIR_DATA_DIR="+data)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if strings.TrimSpace(string(out)) != "398k" {
		t.Fatalf("got %q want 398k", out)
	}
}

func TestPairContext_NoConfig_PrintsNothing(t *testing.T) {
	bin := buildPairContext(t)
	home := t.TempDir()
	cmd := exec.Command(bin, "T", "claude")
	cmd.Env = append(os.Environ(), "HOME="+home, "PAIR_DATA_DIR="+filepath.Join(home, "empty"))
	out, _ := cmd.Output()
	if strings.TrimSpace(string(out)) != "" {
		t.Fatalf("want empty, got %q", out)
	}
}

func mustMkdir(t *testing.T, d string) { t.Helper(); if err := os.MkdirAll(d, 0o755); err != nil { t.Fatal(err) } }
func mustWrite(t *testing.T, p, s string) { t.Helper(); if err := os.WriteFile(p, []byte(s), 0o644); err != nil { t.Fatal(err) } }
```

- [ ] **Step 2: Run to verify it fails** — Run: `go test ./cmd/pair-context/...` — Expected: FAIL (no package / build error).

- [ ] **Step 3: Implement** `cmd/pair-context/main.go`:

```go
// pair-context — print one agent pane's current context size (humanized
// token count), or nothing. Invoked as `pair-context <tag> <agent>` by the
// pair-title poller. Tolerant: any failure prints nothing and exits 0, so a
// hiccup never garbles the pane title.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/ctxmeter"
	"github.com/xianxu/pair/cmd/internal/transcript"
)

func main() {
	if len(os.Args) < 3 {
		return
	}
	tag, agent := os.Args[1], os.Args[2]
	dataDir := os.Getenv("PAIR_DATA_DIR")
	if dataDir == "" {
		base := os.Getenv("XDG_DATA_HOME")
		if base == "" {
			base = filepath.Join(os.Getenv("HOME"), ".local", "share")
		}
		dataDir = filepath.Join(base, "pair")
	}
	sid := transcript.SessionID(dataDir, tag, agent)
	if sid == "" {
		return
	}
	cwd := paneCwd(dataDir, tag, agent) // "" for codex/agy is fine (Resolve ignores it)
	path := transcript.Resolve(agent, sid, cwd, os.Getenv("HOME"))
	if path == "" {
		return
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	if n, ok := ctxmeter.ContextTokens(agent, f); ok {
		fmt.Println(ctxmeter.Humanize(n))
	}
}

func paneCwd(dataDir, tag, agent string) string {
	b, err := os.ReadFile(filepath.Join(dataDir, "pane-"+tag+"-"+agent+".json"))
	if err != nil {
		return ""
	}
	var p struct {
		Cwd string `json:"cwd"`
	}
	if json.Unmarshal(b, &p) != nil {
		return ""
	}
	return p.Cwd
}
```

- [ ] **Step 4: Run to verify pass** — Run: `go test ./cmd/pair-context/...` — Expected: PASS (both cases).

- [ ] **Step 5: Commit** — `git add cmd/pair-context && git commit -m "#71 M2: pair-context one-shot CLI"`

### Task 5: Wire `pair-context` into the build

**Files:**
- Modify: `Makefile.local` (GO_BINS line ~29; alias block ~38-46; build rules ~220)

- [ ] **Step 1: Add to `GO_BINS`** (line 29): append ` pair-context`.
- [ ] **Step 2: Add the per-binary alias** (in the alias block): `pair-context: $(BIN_DIR)/pair-context`
- [ ] **Step 3: Add the build rule** (near the other `$(BIN_DIR)/...` rules):

```makefile
$(BIN_DIR)/pair-context: cmd/pair-context/main.go cmd/internal/ctxmeter/ctxmeter.go cmd/internal/transcript/transcript.go go.mod
	go build -o $@ ./cmd/pair-context
```

- [ ] **Step 4: Build + install** — Run: `cd /Users/xianxu/workspace/pair && make build && make install` — Expected: `installed: ~/.local/bin/pair-context`. Sanity: `pair-context nonexistent claude` prints nothing, exit 0.
- [ ] **Step 5: Commit** — `git add Makefile.local && git commit -m "#71 M2: build+install pair-context"`

> **Chunk 2 review** before Chunk 3.

---

## Chunk 3: Pane-id capture + the unified `pair-title` poller

### Task 6: Write `pane-<tag>-<agent>.json` at pane startup

**Files:**
- Modify: `bin/pair` (after line 1159 — export `PAIR_PANE_CWD`)
- Modify: `zellij/layouts/main.kdl` (the agent pane `args "-c" "..."` — write the pane file before the rename)

- [ ] **Step 1: Export the display cwd** — in `bin/pair` after line 1159 (`export PAIR_PANE_TITLE=...`), add:

```bash
export PAIR_PANE_CWD="$pane_cwd"
```

- [ ] **Step 2: Write the pane file in the layout** — in `zellij/layouts/main.kdl`, prepend a pane-file write to the agent pane's `sh -c` (before the `zellij action rename-pane`), so `$ZELLIJ_PANE_ID`/`$PWD` are in scope:

```kdl
            args "-c" "_pdd=\"${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}\"; printf '{\"pane_id\":\"%s\",\"cwd\":\"%s\",\"cwd_display\":\"%s\"}\\n' \"$ZELLIJ_PANE_ID\" \"$PWD\" \"${PAIR_PANE_CWD:-$PWD}\" > \"$_pdd/pane-${PAIR_TAG:-${PAIR_AGENT:-claude}}-${PAIR_AGENT:-claude}.json\" 2>/dev/null; zellij action rename-pane --pane-id \"$ZELLIJ_PANE_ID\" \"${PAIR_PANE_TITLE:-agent}\" 2>/dev/null; exec pair-wrap --scrollback-log \"${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}/scrollback-${PAIR_TAG:-${PAIR_AGENT:-claude}}-${PAIR_AGENT:-claude}.raw\" ${PAIR_AGENT:-claude} ${PAIR_AGENT_ARGS:-}"
```

- [ ] **Step 3: Manual verify the write** — launch a pair session (`pair <tag>`), then: `cat ~/.local/share/pair/pane-<tag>-claude.json` — Expected: `{"pane_id":"<n>","cwd":"/Users/.../repo","cwd_display":"~/repo"}`.
- [ ] **Step 4: Commit** — `git add bin/pair zellij/layouts/main.kdl && git commit -m "#71 M3: record pane id+cwd to dedicated pane-<tag>-<agent>.json at startup"`

### Task 7: Generalize the poller → `pair-title.sh`

**Files:**
- Rename+modify: `bin/pair-cmux-title.sh` → `bin/pair-title.sh`
- Modify: `bin/pair` (spawn 1659; existence check 1588; cleanup sweeps 398, 446; pidfile name; `poller_alive` references inside the script)
- Create/modify test: `tests/pair-title-poller-test.sh` (rename from `tests/cmux-title-poller-test.sh`)

> This is the highest-risk task: it preserves all existing cmux behavior while adding the frame meter. The `git mv` keeps history. The rename must move **every** reference site in lockstep (per the spec's plan-obligation list) or you get a double-spawn / orphan.

- [ ] **Step 1: Rename the file + the test** — `git mv bin/pair-cmux-title.sh bin/pair-title.sh && git mv tests/cmux-title-poller-test.sh tests/pair-title-poller-test.sh`

- [ ] **Step 2: De-cmux-gate (make the two whole-script gates block-local).** In `bin/pair-title.sh`, DELETE the two early `exit 0` gates:

```bash
# DELETE these two lines (currently ~line 72-73):
[ -n "${CMUX_WORKSPACE_ID:-}" ] || exit 0
command -v cmux >/dev/null 2>&1 || exit 0
```

The poller is now always-on. (The cmux rename below becomes conditional in Step 5.)

- [ ] **Step 3: Rename the pidfile + identity probe.** Change `PIDFILE="$DATA_DIR/cmux-title-pid-$TAG"` → `PIDFILE="$DATA_DIR/title-pid-$TAG"`, the `poller_alive` argv match `*"pair-cmux-title.sh $TAG "*` → `*"pair-title.sh $TAG "*`, and the test-hook env `PAIR_CMUX_TITLE_TEST_CALL` → `PAIR_TITLE_TEST_CALL`. Update the header comment.

- [ ] **Step 4: Add the per-pane frame-meter loop** — inside the main `while true` loop, after the activity computation (`latest=$(latest_activity)`), before/after the cmux block, add a loop over the tag's pane files. Define a helper near the top:

```bash
# Abbreviate a raw cwd to ~ on a path boundary (mirrors bin/pair).
abbrev_cwd() {
    case "$1" in
        "$HOME")   printf '~' ;;
        "$HOME"/*) printf '~%s' "${1#"$HOME"}" ;;
        *)         printf '%s' "$1" ;;
    esac
}

# Per-pane last-title cache (skip redundant renames). Bash 3.2 (macOS) has no
# associative arrays in older shells — use a flat "pane_id=title" newline list.
frame_titles=""
frame_title_cached() { printf '%s\n' "$frame_titles" | sed -n "s/^$1=//p" | head -1; }
frame_title_store()  { frame_titles=$(printf '%s\n' "$frame_titles" | grep -v "^$1=" ; printf '%s=%s\n' "$1" "$2"); }

# Rename every agent pane's zellij frame to "<agent> (<count>) [<cwd>]".
update_frame_titles() {
    local pf agent pane_id cwd_disp count title cached
    for pf in "$DATA_DIR"/pane-"$TAG"-*.json; do
        [ -f "$pf" ] || continue
        agent=$(basename "$pf" .json); agent=${agent#pane-"$TAG"-}
        pane_id=$(jq -r '.pane_id // empty' "$pf" 2>/dev/null); [ -n "$pane_id" ] || continue
        cwd_disp=$(jq -r '.cwd_display // empty' "$pf" 2>/dev/null)
        [ -n "$cwd_disp" ] || cwd_disp=$(abbrev_cwd "$(jq -r '.cwd // empty' "$pf" 2>/dev/null)")
        count=$(pair-context "$TAG" "$agent" 2>/dev/null)
        if [ -n "$count" ]; then
            title="$agent ($count) [$cwd_disp]"
        else
            title="$agent [$cwd_disp]"
        fi
        cached=$(frame_title_cached "$pane_id")
        [ "$cached" = "$title" ] && continue   # unchanged → skip IPC
        zellij --session "$SESSION" action rename-pane --pane-id "$pane_id" "$title" >/dev/null 2>&1 || true
        frame_title_store "$pane_id" "$title"
    done
}
```

Call `update_frame_titles` inside the loop, gated on activity (reusing `latest`/`age` already computed — only when `age` is within a refresh window, e.g. the existing "activity resolved" path). Keep it on the active branch so idle sessions don't churn.

- [ ] **Step 5: Make the cmux rename block-local** — wrap the existing cmux ownership + `cmux rename-workspace` block in `if [ -n "${CMUX_WORKSPACE_ID:-}" ] && command -v cmux >/dev/null 2>&1; then ... fi` so it only runs in cmux. The frame-meter loop (Step 4) runs unconditionally.

- [ ] **Step 6: Update `bin/pair` reference sites** (all in lockstep):
  - Spawn (1659): `"$PAIR_HOME/bin/pair-cmux-title.sh"` → `"$PAIR_HOME/bin/pair-title.sh"`
  - Existence check (1588): `"$DATA_DIR/cmux-title-pid-${PAIR_TAG}"` → `"$DATA_DIR/title-pid-${PAIR_TAG}"`
  - Cleanup sweep #1 (398-401): the `for fam in ... cmux-title-pid ...` list → `title-pid`
  - Cleanup sweep #2 (446): the case pattern `cmux-title-pid-$old_tag` → `title-pid-$old_tag`
  - Grep to be sure none missed: `grep -rn "cmux-title-pid\|pair-cmux-title" bin/ tests/` → Expected: no hits after edits.

- [ ] **Step 7: Add `pane-<tag>-*.json` to cleanup** — in cleanup sweep #1's `for fam in ...` add `pane` (so `pane-$tag-*` files are reaped on session end). Note: `pane-` is `<tag>-<agent>` keyed, so the per-fam `$dd/$fam-$tag` glob covers `pane-$tag-*` only if the sweep globs; verify it matches the multi-suffix pattern and adjust if the sweep is exact-match (add an explicit `rm -f "$dd"/pane-"$tag"-*.json`).

- [ ] **Step 8: Update the shell test** `tests/pair-title-poller-test.sh` — rename the `PAIR_CMUX_TITLE_TEST_CALL` hook usages → `PAIR_TITLE_TEST_CALL`; add a new test driving `update_frame_titles` with a fake `zellij` and fake `pair-context` on `PATH`:

```bash
# Fake zellij that records rename-pane calls; fake pair-context that prints a count.
mk_fakes() {
    cat > "$BIN/zellij" <<'EOF'
#!/usr/bin/env bash
[ "$3" = "rename-pane" ] && printf '%s\n' "$*" >> "$RENAME_LOG"
exit 0
EOF
    cat > "$BIN/pair-context" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "${FAKE_COUNT:-970k}"
EOF
    chmod +x "$BIN/zellij" "$BIN/pair-context"
}
# Assert: a pane file → one rename-pane with `claude (970k) [~/repo]`; a second
# identical tick → NO new rename (unchanged-skip).
```

- [ ] **Step 9: Run the shell tests** — Run: `cd /Users/xianxu/workspace/pair && bash tests/pair-title-poller-test.sh` — Expected: PASS (poller_alive identity + the new frame-title + unchanged-skip).

- [ ] **Step 10: Commit** — `git add -A && git commit -m "#71 M3: generalize pair-cmux-title.sh -> always-on pair-title (cmux title + zellij frame meter)"`

> **Chunk 3 review** before Chunk 4.

---

## Chunk 4: End-to-end verification + atlas

### Task 8: Manual smoke test (the deterministic shell can't fully fake zellij IPC)

**Files:** none (verification)

- [ ] **Step 1: Build+install** — `make install`.
- [ ] **Step 2: Fresh claude pane** — `pair smoke71` in a repo; after the agent produces a turn, within ~60s the frame reads `claude (<count>) [~/<repo>]`. Confirm `<count>` ≈ `/context` (sanity, not exact).
- [ ] **Step 3: Multiple same-cwd sessions** — start a 2nd pair session in the SAME cwd; confirm each pane shows its OWN count (the pinned-sid disambiguation — the load-bearing case).
- [ ] **Step 4: `/clear` smoke test** (the spec's open item) — in a claude pane, `/clear`, do a turn; confirm the count DROPS to a small value in the same pane (proves in-place, not a frozen stale count). If it stays frozen, `/clear` rotated the file after all → file a follow-up to add newest-sid-in-project-dir-modified-since-pane-start lineage following (do NOT switch to plain newest-by-mtime — it aliases same-cwd).
- [ ] **Step 5: codex pane** — confirm `codex (<count>)`; **agy pane** — confirms `agy [~/cwd]` (no number, identical to today).
- [ ] **Step 6: Idle** — leave a session untouched > a few minutes; confirm titles don't churn (no flicker), and `cmux` workspace title still heat-ramps when in cmux.
- [ ] **Step 7:** Record results (counts seen, /clear behavior) in the issue `## Log`.

### Task 9: Atlas + close

**Files:**
- Modify: `atlas/` (the file documenting pair's poller / session-state surface; + `atlas/index.md` if a new file)

- [ ] **Step 1: Update atlas** — document: the `pair-title` poller (renamed, now always-on, two title surfaces), the `pane-<tag>-<agent>.json` state file, `pair-context` + the `ctxmeter`/`transcript` packages. Reconcile any atlas mention of `pair-cmux-title.sh` → `pair-title.sh`.
- [ ] **Step 2: Full test sweep** — `go test ./... && bash tests/pair-title-poller-test.sh && go vet ./cmd/...` — Expected: all PASS.
- [ ] **Step 3: Set actuals + close** — `sdlc close --issue 71 --verified '<evidence: tests + manual smoke incl. same-cwd + /clear>'` (let close compute `--actual`).

---

## Pre-implementation: set the estimate

Before `sdlc change-code`, set `estimate_hours:` in the issue frontmatter (required by change-code, #113). Estimate after this plan is reviewed.

## Risks & mitigations (carried from spec)
- **Bash 3.2 on macOS** has no associative arrays — the per-pane title cache uses a flat list (Task 7 Step 4). Verify the poller's shebang/runtime.
- **Large transcripts** (14MB+) are re-read each active tick — acceptable for v1 (≤~once/60s when active); `ReadBytes` avoids the Scanner 64KB cap. Backward/tail-read optimization is out of scope.
- **`/clear` rotation** — proven in-place by data, but Task 8 Step 4 smoke-tests it for real; the fallback is lineage-following, NOT newest-by-mtime.
- **Rename-site lockstep** — Task 7 Step 6 includes a `grep` gate to catch a missed reference.
