# Stop Extracting Orchestrator Shell Scripts Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove all seven `#93`/`#99`-owned orchestrator shell shims from the embedded runtime bundle — first by porting the two that still have real shell logic (`pair-restart.sh`, `pair-quit.sh`) into in-process `pair restart` / `pair quit` subcommands (M1), then by repointing the five exec-shim call sites to their Go binaries and deleting the shims (M2).

**Architecture:** M1 reuses the launcher's *existing* marker-write seam — `WriteRestartMarker`/`TouchQuitMarker`/`ExecKillSession`/`InferAgent` already live on `OSRuntime` (built for #55/#99 compaction); `runCompaction` already chains exactly the sequence the two scripts perform. So M1 adds no new Runtime methods — only a pure arg parser (`parseRestart`), two thin runners (`runRestart`/`runQuit`) reusing the seam, a router branch, and two nvim call-site repoints. M2 is a pure "repoint caller → drop shim" sweep: five call sites (launcher spawns ×2, zellij `copy_command`, the Go `copy-on-select`'s internal exec of flash/clipboard ×2) drop the `.sh` suffix to invoke the Go binary directly, then the seven shim source files + their `explicitAssetPaths` entries are deleted and the bundle regenerated.

**Tech Stack:** Go (launcher + clipcmd + runtimebundle generator), Lua (nvim call sites), KDL (zellij config), bash (shim files being deleted + the smoke-test harnesses). Bundle is code-generated via `make runtimebundle-generate` from `explicitAssetPaths` in `cmd/internal/runtimebundlegen/generate.go` — the single packaging source (ARCH-DRY).

**Scope note (read before starting):** #94's stated purpose is "the bundle carries only native assets (`nvim/`,`zellij/`) plus Go-owned pieces" — so all seven orchestrator shims must go, not just the two with dead callers. The five exec-shims (`pair-title.sh`, `pair-session-watch.sh`, `copy-on-select.sh`, `flash-pane.sh`, `clipboard-to-pane.sh`) are NOT dead — they are live `.sh` → `exec $PAIR_HOME/bin/<name>` passthroughs invoked by name; removing them requires repointing their callers first (ARCH-PURPOSE: deliver the purpose, not the two-file subset). The **six** shell files that remain bundled after #94 — `bin/lib/adapt-log.sh`, `bin/lib/dev-rebuild.sh`, `bin/pair-help`, `bin/pair-notify`, `doctor/doctor.sh`, `doctor/emitter-health.sh` — are non-orchestrator utilities never in #93's scope (dev-rebuild source, notify/help leaves, doctor diagnostics); porting them is explicitly OUT of scope. #94's honest endpoint is therefore **shell-reduced, not shell-free**; say so at close.

---

## Core Concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `LaunchArgs` (add `NewSession bool`, `RenameTo string`) | `cmd/internal/launcher/args.go` | modified |
| `parseRestart` | `cmd/internal/launcher/args.go` | new |
| `RestartMarker` | `cmd/internal/launcher/markers.go` | reused (unchanged) |
| `serializeRestartMarker` | `cmd/internal/launcher/compaction.go` | reused (unchanged) |

- **`LaunchArgs` (modified)** — the pure parse result the launcher router switches on. Gains two fields used only by the `restart` verb: `NewSession` (from `--new-session`) and `RenameTo` (from `--rename-to <tag>`; the inside-flow tag rename written into the restart marker — distinct from the `rename` command's `RenameOld`/`RenameNew`).
  - **Relationships:** 1:1 with an invocation. `Command == "restart"` reads `NewSession`/`RenameTo`; `Command == "quit"` reads neither.
  - **DRY rationale:** reuses the same `LaunchArgs`/`ParseArgs` switchboard every other verb (list/rename/continue) already flows through — no parallel parser.
- **`parseRestart` (new)** — pure `[]string → (LaunchArgs, error)`, mirroring `pair-restart.sh:29-56`: accepts `--new-session` and `--rename-to <value>` (error on missing value or unknown arg). Sibling of the existing `parseRename`/`parseContinue`.
- **`RestartMarker` / `serializeRestartMarker` (reused, unchanged)** — the marker struct + `key=value` serializer already exist and already round-trip through `parseRestartMarker`/`TakeRestartMarker` (the launcher's restart loop). M1 constructs a `RestartMarker` and hands it to the existing `WriteRestartMarker` — it does NOT touch the format. `serializeRestartMarker` omits `new_session=0` where the shell always wrote it; harmless (only `parseRestartMarker` reads markers, and it treats absent as false).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `runRestart` / `runQuit` | `cmd/internal/launcher/restart.go` | new | the launcher `Runtime` seam |
| `LaunchNative` restart/quit branches | `cmd/internal/launcher/runcli.go` | modified | env (`ZELLIJ_SESSION_NAME`) + router |
| nvim quit/restart call sites | `nvim/init.lua:3185,3288` | modified | the `pair` binary |
| `SpawnSessionWatcher` / `SpawnTitlePoller` | `cmd/internal/launcher/osruntime.go:276,281` | modified | detached Go binary spawn |
| clipcmd flash/clip exec paths | `cmd/internal/clipcmd/run.go:87,97` | modified | Go binary exec |
| zellij `copy_command` | `zellij/config.kdl:41` | modified | zellij → Go binary |
| `explicitAssetPaths` | `cmd/internal/runtimebundlegen/generate.go:19-49` | modified | the bundle manifest source |
| `TestEmbeddedManifestContainsLaunchAssets` | `cmd/internal/runtimebundle/embed_test.go` | modified | bundle-contents assertion |

- **`runRestart` / `runQuit` (new)** — thin runners that read the live session, derive tag/agent, and drive the existing Runtime marker seam. `runRestart` = `InferAgent` → `WriteRestartMarker` → `TouchQuitMarker` → `ExecKillSession`; `runQuit` = `TouchQuitMarker` → `ExecKillSession`. Direct analog of `runCompaction` (`compaction.go:54-68`).
  - **Injected into:** invoked by `LaunchNative` with the `Runtime` interface, so the pure decision (which markers, what payload) is unit-testable against a fake runtime (mirrors `createflow_test.go`'s `fakeRuntime` with `writtenMarkers`/`touchedQuit`/`killed`).
- **nvim call sites (modified)** — fire-and-forget `vim.fn.system(...)` that exec kill-session and never return; exit codes are not checked. Repoint = change `argv[0]` from the `.sh` name to `{ 'pair', '<verb>' }`.
- **`SpawnSessionWatcher`/`SpawnTitlePoller` + clipcmd paths + zellij `copy_command` (modified)** — the five live exec-shim callers. Each drops the `.sh` suffix so the Go binary runs directly (one fewer bash exec). `pair-title.sh`'s own comment confirms the Go poller's single-instance argv guard already expects the `…/pair-title <tag> <agent>` shape, so spawning it directly is the shape the guard recognizes — safe.
- **`explicitAssetPaths` + `embed_test.go` (modified)** — the single source of what gets bundled, and the test asserting bundle contents. Removing a shim = delete its line from `explicitAssetPaths` (regeneration drops the manifest entry + the `assets/runtime/files/bin/*.sh` copy) and add its name to `embed_test.go`'s `excluded` slice so a regression that re-adds it fails.

**Test surface:** `parseRestart` + the runners get colocated unit tests in `cmd/internal/launcher/restart_test.go` (pure parse + fake-Runtime marker assertions, no IO mocks for the parse half). The wiring gets a process-level smoke (`tests/pair-restart-quit-test.sh`) driving the real `pair` binary with `PAIR_KILL_CMD` stubbing the terminal exec, asserting the marker files land. M2's removal is guarded by the existing `embed_test.go` + the copied-binary smoke `tests/pair-embedded-runtime-test.sh` + `tests/copy-on-select-test.sh`.

---

## Chunk 1: M1 — port `pair restart` / `pair quit`

### Task 1: Pure arg parsing for `restart` / `quit`

**Files:**
- Modify: `cmd/internal/launcher/args.go` (add fields to `LaunchArgs`; add `case "restart"`/`case "quit"` to `ParseArgs`; add `parseRestart`)
- Test: `cmd/internal/launcher/args_test.go`

- [ ] **Step 1: Write failing parse tests**

Add to `args_test.go`:

```go
func TestParseRestart(t *testing.T) {
	cases := []struct {
		name    string
		argv    []string
		want    LaunchArgs
		wantErr bool
	}{
		{"bare", []string{"restart"}, LaunchArgs{Command: "restart"}, false},
		{"new-session", []string{"restart", "--new-session"}, LaunchArgs{Command: "restart", NewSession: true}, false},
		{"rename-to", []string{"restart", "--rename-to", "foo"}, LaunchArgs{Command: "restart", RenameTo: "foo"}, false},
		{"new-session+rename", []string{"restart", "--new-session", "--rename-to", "foo"}, LaunchArgs{Command: "restart", NewSession: true, RenameTo: "foo"}, false},
		{"rename-to missing value", []string{"restart", "--rename-to"}, LaunchArgs{}, true},
		{"unknown arg", []string{"restart", "--bogus"}, LaunchArgs{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseArgs(tc.argv)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestParseQuit(t *testing.T) {
	got, err := ParseArgs([]string{"quit"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != (LaunchArgs{Command: "quit"}) {
		t.Fatalf("got %+v, want quit", got)
	}
}
```

Note: `got != tc.want` compares `LaunchArgs` by value — it has no slice/map fields set in these cases (`AgentArgs` stays nil on both sides), so `==` is valid.

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/launcher/ -run 'TestParseRestart|TestParseQuit' -v`
Expected: FAIL — currently `restart`/`quit` fall through to the general loop and set `Agent: "restart"` (the silent-misroute trap), so the equality checks fail.

- [ ] **Step 3: Add fields + parser + cases**

In `args.go`, add to the `LaunchArgs` struct (after `ContinueSlug`):

```go
	// restart (#94 M1): `pair restart [--new-session] [--rename-to <tag>]` —
	// the nvim-keybind lifecycle writer ported from bin/pair-restart.sh. Both
	// fields are written into the restart marker; RenameTo is the inside-flow
	// tag rename (distinct from the `rename` command's RenameOld/RenameNew).
	NewSession bool
	RenameTo   string
```

In `ParseArgs`, add to the `switch argv[0]` (after the `continue` case, before `resume`):

```go
	case "restart":
		return parseRestart(argv[1:]) // #94 M1
	case "quit":
		return LaunchArgs{Command: "quit"}, nil // #94 M1
```

Add the parser (sibling of `parseRename`):

```go
// parseRestart parses `restart [--new-session] [--rename-to <tag>]` (#94 M1,
// ported from bin/pair-restart.sh:29-56). Flags only; no positionals.
func parseRestart(args []string) (LaunchArgs, error) {
	out := LaunchArgs{Command: "restart"}
	i := 0
	for i < len(args) {
		switch args[i] {
		case "--new-session":
			out.NewSession = true
			i++
		case "--rename-to":
			if i+1 >= len(args) || args[i+1] == "" {
				return LaunchArgs{}, UsageError{Message: "pair restart: --rename-to requires a value"}
			}
			out.RenameTo = args[i+1]
			i += 2
		default:
			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair restart: unknown arg %q", args[i])}
		}
	}
	return out, nil
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./cmd/internal/launcher/ -run 'TestParseRestart|TestParseQuit' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/launcher/args.go cmd/internal/launcher/args_test.go
git commit -m "#94 M1: parse pair restart/quit verbs

Add --new-session/--rename-to parsing (ported from pair-restart.sh) + the
quit verb to the launcher router's ParseArgs. Explicit cases prevent the
silent misroute where an unknown verb becomes an agent name.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 2: `runRestart` / `runQuit` runners + router wiring

**Files:**
- Create: `cmd/internal/launcher/restart.go`
- Modify: `cmd/internal/launcher/runcli.go` (dispatch branches)
- Test: `cmd/internal/launcher/restart_test.go`

- [ ] **Step 1: Write failing runner tests**

Create `restart_test.go` mirroring the `fakeRuntime` pattern in `createflow_test.go` (fields `writtenMarkers map[string]RestartMarker`, `touchedQuit []string`, `killed []string`; constructor `newFakeRuntime()`):

```go
package launcher

import (
	"bytes"
	"testing"
)

func TestRunRestartWritesMarkersAndKills(t *testing.T) {
	rt := newFakeRuntime()
	rt.inferAgent = map[string]string{"demo": "codex"} // the fake's InferAgent source (createflow_test.go:27,163)
	var stderr bytes.Buffer
	code := runRestart(rt, LaunchArgs{Command: "restart", NewSession: true, RenameTo: "renamed"}, "pair-demo", &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%q", code, stderr.String())
	}
	m, ok := rt.writtenMarkers["pair-demo"]
	if !ok {
		t.Fatal("no restart marker written")
	}
	if m.Tag != "demo" || m.Agent != "codex" || !m.NewSession || m.RenameTo != "renamed" {
		t.Fatalf("marker = %+v", m)
	}
	if len(rt.touchedQuit) != 1 || rt.touchedQuit[0] != "pair-demo" {
		t.Fatalf("touchedQuit = %v", rt.touchedQuit)
	}
	if len(rt.killed) != 1 || rt.killed[0] != "pair-demo" {
		t.Fatalf("killed = %v", rt.killed)
	}
}

func TestRunQuitTouchesQuitAndKills(t *testing.T) {
	rt := newFakeRuntime()
	var stderr bytes.Buffer
	code := runQuit(rt, "pair-demo", &stderr)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if len(rt.writtenMarkers) != 0 {
		t.Fatalf("quit must not write a restart marker: %v", rt.writtenMarkers)
	}
	if len(rt.touchedQuit) != 1 || rt.touchedQuit[0] != "pair-demo" {
		t.Fatalf("touchedQuit = %v", rt.touchedQuit)
	}
	if len(rt.killed) != 1 {
		t.Fatalf("killed = %v", rt.killed)
	}
}

func TestRunRestartMissingSession(t *testing.T) {
	rt := newFakeRuntime()
	var stderr bytes.Buffer
	if code := runRestart(rt, LaunchArgs{Command: "restart"}, "", &stderr); code != 1 {
		t.Fatalf("want exit 1 on empty session, got %d", code)
	}
	if len(rt.writtenMarkers) != 0 || len(rt.killed) != 0 {
		t.Fatal("must not write markers or kill when session is unset")
	}
}
```

**Before writing the test, read `cmd/internal/launcher/createflow_test.go`.** The `fakeRuntime` already has everything needed: the `writtenMarkers`/`touchedQuit`/`killed` recorders, the `WriteRestartMarker`/`TouchQuitMarker`/`ExecKillSession` stubs, and an `inferAgent map[string]string` field (`createflow_test.go:27`) whose `InferAgent` stub returns `f.inferAgent[tag]` (`createflow_test.go:163`). Reuse it as-is — set `rt.inferAgent`; do NOT add a new field or define a second fake (ARCH-DRY).

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/internal/launcher/ -run 'TestRunRestart|TestRunQuit' -v`
Expected: FAIL — `runRestart`/`runQuit` undefined.

- [ ] **Step 3: Write the runners**

Create `restart.go`:

```go
package launcher

import (
	"io"
	"strings"
)

// runRestart is the in-process port of bin/pair-restart.sh (#94 M1): resolve the
// live session/tag/agent, write the restart marker (carrying new_session +
// optional rename_to), touch the quit marker, then exec kill-session. It reuses
// the exact Runtime seam runCompaction drives (compaction.go:54-68); the effects
// live on OSRuntime, so nothing new is added to the seam. ExecKillSession is
// terminal on the real runtime (syscall.Exec) — the return is reached only when
// the kill binary is missing or under the fake runtime.
func runRestart(rt Runtime, args LaunchArgs, session string, stderr io.Writer) int {
	if session == "" {
		_, _ = io.WriteString(stderr, "pair restart: ZELLIJ_SESSION_NAME unset; cannot restart cleanly.\n")
		return 1
	}
	tag := strings.TrimPrefix(session, "pair-")
	rt.WriteRestartMarker(session, RestartMarker{
		Tag:        tag,
		Agent:      rt.InferAgent(tag),
		NewSession: args.NewSession,
		RenameTo:   args.RenameTo,
	})
	rt.TouchQuitMarker(session)
	rt.ExecKillSession(session)
	return 0
}

// runQuit is the in-process port of bin/pair-quit.sh (#94 M1): touch the quit
// marker so the outer loop's cleanup fires, then exec kill-session. No restart
// marker — Alt+x is a full quit.
func runQuit(rt Runtime, session string, stderr io.Writer) int {
	if session == "" {
		_, _ = io.WriteString(stderr, "pair quit: ZELLIJ_SESSION_NAME unset; cannot quit cleanly.\n")
		return 1
	}
	rt.TouchQuitMarker(session)
	rt.ExecKillSession(session)
	return 0
}
```

In `runcli.go`, add after the bare-`continue` branch (line 58, before the `env :=` block):

```go
	// `restart`/`quit` are the nvim-keybind lifecycle writers (#94 M1, ported
	// from bin/pair-{restart,quit}.sh): write markers, exec kill-session. They
	// need the live ZELLIJ_SESSION_NAME the keybind fires under.
	if args.Command == "restart" {
		return runRestart(rt, args, os.Getenv("ZELLIJ_SESSION_NAME"), stderr), nil
	}
	if args.Command == "quit" {
		return runQuit(rt, os.Getenv("ZELLIJ_SESSION_NAME"), stderr), nil
	}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./cmd/internal/launcher/ -run 'TestRunRestart|TestRunQuit' -v`
Expected: PASS

- [ ] **Step 5: Full package test**

Run: `go test ./cmd/internal/launcher/`
Expected: PASS (confirms the new router branches + fake-runtime changes didn't break existing routing tests in `runcli_test.go`).

- [ ] **Step 6: Commit**

```bash
git add cmd/internal/launcher/restart.go cmd/internal/launcher/restart_test.go cmd/internal/launcher/runcli.go cmd/internal/launcher/createflow_test.go
git commit -m "#94 M1: runRestart/runQuit reuse the compaction marker seam

Port pair-restart.sh/pair-quit.sh into runners that reuse the existing
WriteRestartMarker/TouchQuitMarker/ExecKillSession/InferAgent Runtime seam
(no new methods) and wire them into LaunchNative. Analog of runCompaction.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 3: Repoint nvim; process-level smoke; retire the two shims

**Files:**
- Modify: `nvim/init.lua` (lines 3185, 3288-3294; comment blocks 3201-3217, 3301-3307)
- Modify: `cmd/internal/runtimebundlegen/generate.go` (remove lines 22-23)
- Modify: `cmd/internal/runtimebundle/embed_test.go` (add to `excluded`)
- Delete: `bin/pair-quit.sh`, `bin/pair-restart.sh`
- Create: `tests/pair-restart-quit-test.sh`
- Modify: `Makefile.local` (register the new test target if the suite enumerates targets explicitly)

- [ ] **Step 1: Repoint the nvim call sites**

`nvim/init.lua:3185` (inside `PairConfirmQuit`):
```lua
      vim.fn.system({ 'pair', 'quit' })
```
`nvim/init.lua:3288` (inside `pair_confirm_restart_impl`):
```lua
    local argv = { 'pair', 'restart' }
```
(the existing `--new-session` / `--rename-to` appends below stay verbatim). Update the two comment blocks (3201-3217, 3301-3307) that say "pair-restart.sh"/"shell out to pair-restart.sh directly" to name `pair restart`/`pair quit`. Edit only the source `nvim/init.lua` — the bundled copy `cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua` is regenerated in Step 4.

- [ ] **Step 2: Write the process-level smoke test**

Create `tests/pair-restart-quit-test.sh` — driving the *real* built `pair` binary with `PAIR_KILL_CMD` stubbing the terminal exec so the marker writes are observable (read `tests/pair-embedded-runtime-test.sh` for the harness idioms: `$TMPDIR` HOME, stub PATH, assertion helpers). Skeleton:

```bash
#!/usr/bin/env bash
# Drives `pair restart` / `pair quit` against a stubbed kill-session, asserting
# the marker files land where the launcher's restart loop reads them.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PAIR="$ROOT/bin/pair"           # built by the target prereq
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
export HOME="$TMP"
export PAIR_KILL_CMD="true"     # ExecKillSession runs `true <session>` instead of zellij
export ZELLIJ_SESSION_NAME="pair-smoke"
MARK="$TMP/.cache/pair"
mkdir -p "$MARK"                # WriteAtomic/Touch MkdirAll it too, but be explicit

# (a) restart writes the restart marker (tag/new_session/rename_to) + touches quit.
"$PAIR" restart --new-session --rename-to renamed
grep -qx 'tag=smoke'         "$MARK/restart-pair-smoke"
grep -qx 'new_session=1'     "$MARK/restart-pair-smoke"
grep -qx 'rename_to=renamed'  "$MARK/restart-pair-smoke"
test -f "$MARK/quit-pair-smoke"

# (b) quit touches quit AND writes NO restart marker. Clear both first so the
#     post-quit assertions actually pin quit's behavior (not restart's leftovers).
rm -f "$MARK/quit-pair-smoke" "$MARK/restart-pair-smoke"
"$PAIR" quit
test -f "$MARK/quit-pair-smoke"
test ! -f "$MARK/restart-pair-smoke"   # quit must not write a restart marker
echo "PASS pair-restart-quit"
```

Verify against the real payload: `serializeRestartMarker` writes `tag=`, `agent=`, then `new_session=1` (omitted when false) and `rename_to=`. `agent=` is empty here (no `agent-smoke` file), which is fine. Adjust `grep` lines to the exact emitted format after a first run.

- [ ] **Step 3: Register + run the smoke test**

If `Makefile.local` enumerates test targets explicitly (as it does for `test-copy-on-select`, `test-changelog`), add a `test-pair-restart-quit` target with `$(BIN_DIR)/pair` as a prereq (so a fresh tree builds the binary first — the M1/M3 lesson) and fold it into the aggregate `test` target. Run:

Run: `chmod +x tests/pair-restart-quit-test.sh && make test-pair-restart-quit` (or `bash tests/pair-restart-quit-test.sh` after `make pair`)
Expected: `PASS pair-restart-quit`

- [ ] **Step 4: Retire the two shims + regenerate bundle**

Delete lines 22-23 (`"bin/pair-quit.sh"`, `"bin/pair-restart.sh"`) from `explicitAssetPaths` in `generate.go`. Delete the source files:

```bash
git rm bin/pair-quit.sh bin/pair-restart.sh
```

Regenerate the bundle (rewrites `manifest.json` + removes `assets/runtime/files/bin/pair-{quit,restart}.sh`):

Run: `make runtimebundle-generate`

- [ ] **Step 5: Tighten the bundle-contents test**

In `cmd/internal/runtimebundle/embed_test.go`, add `"bin/pair-quit.sh"` and `"bin/pair-restart.sh"` to the `excluded` slice (next to `bin/pair-dev`). Run:

Run: `make test-runtimebundle`
Expected: PASS (asserts the two `.sh` are no longer in the embedded manifest).

- [ ] **Step 6: Verify no straggler references**

Run: `grep -rn 'pair-restart\.sh\|pair-quit\.sh' . --exclude-dir=.git --exclude-dir=workshop`
Expected: only historical-provenance comments (e.g. in `markers.go`/`compaction.go`/`osruntime.go` describing the ported format) and the regenerated `assets/.../files/nvim/init.lua` should show — NO live invocation. If a live caller remains, repoint it before proceeding.

- [ ] **Step 7: Full test + commit**

Run: `env -u PAIR_SESSION_ID -u PAIR_TAG make test` (scrub the session env leak per the standing lesson; sandbox-off — the launcher tests need real `ps`)
Expected: green.

```bash
git add -A
git commit -m "#94 M1: repoint nvim to pair restart/quit, retire the two shims

nvim keybinds now invoke the Go subcommands; bin/pair-{restart,quit}.sh are
deleted from the source tree and the runtime bundle (the last orchestrator
shell with real logic — the other five are exec-shims removed in M2).

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

- [ ] **Step 8: M1 milestone close**

`sdlc milestone-close --issue 94 --milestone M1 --verified '<evidence>'` (compute `--actual` via `sdlc actual --issue 94` or the omit-then-suggest path). Fix any Critical/Important from the boundary review before crossing. Log the verdict in `## Log`.

---

## Chunk 2: M2 — repoint the five exec-shims and drop them

### Task 4: Repoint launcher spawns + zellij + clipcmd exec chain

**Files:**
- Modify: `cmd/internal/launcher/osruntime.go:276,281`
- Modify: `zellij/config.kdl:41`
- Modify: `cmd/internal/clipcmd/run.go:87,97`
- Modify: `tests/copy-on-select-test.sh` (stubs by `.sh` path)
- Test: existing `cmd/internal/clipcmd/*_test.go`, `cmd/internal/launcher/*_test.go`

- [ ] **Step 1: Find the tests that pin the `.sh` paths (before editing)**

Run: `grep -rn 'pair-title\.sh\|pair-session-watch\.sh\|copy-on-select\.sh\|flash-pane\.sh\|clipboard-to-pane\.sh' cmd/ tests/`
This enumerates every assertion/stub keyed on the `.sh` names — the exact set to update in lockstep with the repoints. (Expect: `clipcmd` fake-runtime expectations on the flash/clip paths, `tests/copy-on-select-test.sh` path stubs, and the `osruntime` spawn strings.)

- [ ] **Step 2: Repoint the five call sites (drop the `.sh` suffix)**

- `osruntime.go:276` → `filepath.Join(r.PairHome, "bin", "pair-session-watch")`
- `osruntime.go:281` → `filepath.Join(r.PairHome, "bin", "pair-title")`
- `run.go:87` → `flashScript := opts.PairHome + "/bin/flash-pane"`
- `run.go:97` → `clipScript := opts.PairHome + "/bin/clipboard-to-pane"`
- `zellij/config.kdl:41` → `copy_command "copy-on-select"`

Update the adjacent `// … flash-pane.sh …` comments in `run.go` (lines 21, 33-34, 84, 94) to name the Go binaries. The Go binaries take the same argv the shims passed through, so behavior is preserved (one fewer bash exec).

- [ ] **Step 3: Update the tests that pinned the `.sh` paths**

Two files hold pinned `.sh` paths — update BOTH now, in lockstep with Step 2, so the chain stays green (and so nothing references `bin/copy-on-select.sh` once Task 5 deletes it):

1. `cmd/internal/clipcmd/run_test.go` (~lines 102, 113, 117 per Step 1's grep) — the fake-runtime expectations assert the flash/clip exec paths end in `.sh`; change them to the suffix-free `…/bin/flash-pane` and `…/bin/clipboard-to-pane` to match the repointed `run.go`.
2. `tests/copy-on-select-test.sh` — this harness references the shims four ways; all must change:
   - **:28** `cp "$REPO/bin/copy-on-select.sh" …` — DROP this line (the shim is deleted in Task 5; keep only `:29`'s `cp "$REPO/bin/copy-on-select"`).
   - **:66** `run() { … | "$PAIR_HOME/bin/copy-on-select.sh"; }` — invoke the Go binary directly: `"$PAIR_HOME/bin/copy-on-select"`.
   - **:32, :37** — the stubs it writes are named `clipboard-to-pane.sh` / `flash-pane.sh`; rename them to `clipboard-to-pane` / `flash-pane` (the suffix-free names the repointed Go `copy-on-select` now execs by absolute path), and update the `chmod +x` at **:38** to match.
   - **:20-25** — update the comment block describing the `.sh` chain to the suffix-free names.

   (These stubs are found by absolute path under `$PAIR_HOME/bin`, so renaming the files is sufficient; the `PATH must NOT include $PAIR_HOME/bin` invariant at :41-42 is unaffected.)

Also update the stale `pair-quit.sh`/`pair-restart.sh` comments at `zellij/config.kdl:74,150,152` (they describe live behavior: "tears down cleanly via pair-quit.sh") to name the `pair quit`/`pair restart` subcommands — a comment cleanup, done here since you're already editing `config.kdl`.

- [ ] **Step 4: Run the affected unit + shell tests**

Run: `go test ./cmd/internal/clipcmd/ ./cmd/internal/launcher/ && make test-copy-on-select`
Expected: PASS — the Go copy-on-select now execs the Go flash/clipboard binaries; the shell test drives the real chain through the stubs.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/launcher/osruntime.go zellij/config.kdl cmd/internal/clipcmd/run.go tests/copy-on-select-test.sh cmd/internal/clipcmd cmd/internal/launcher
git commit -m "#94 M2: repoint the five exec-shim callers to the Go binaries

Launcher spawns (pair-title, pair-session-watch), zellij copy_command, and
the copy-on-select flash/clipboard hand-off now invoke the Go binaries
directly instead of the .sh passthrough shims.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 5: Delete the five shims from the tree + bundle; tighten the guards

**Files:**
- Modify: `cmd/internal/runtimebundlegen/generate.go` (remove lines 29-33)
- Delete: `bin/copy-on-select.sh`, `bin/clipboard-to-pane.sh`, `bin/flash-pane.sh`, `bin/pair-title.sh`, `bin/pair-session-watch.sh`
- Modify: `cmd/internal/runtimebundle/embed_test.go` (add the five to `excluded`)
- Modify: `tests/pair-embedded-runtime-test.sh` (asserts `pair-session-watch.sh`/`pair-title.sh` present)

- [ ] **Step 1: Update the copied-binary smoke assertions first**

`tests/pair-embedded-runtime-test.sh:45-46` asserts the extracted root contains `bin/pair-session-watch.sh` and `bin/pair-title.sh`. Change these to assert the Go binaries (`bin/pair-session-watch`, `bin/pair-title`) are present AND the `.sh` are absent (e.g. `test ! -e "$root/bin/pair-title.sh"`). This makes the smoke fail loudly if a shim reappears in the bundle.

- [ ] **Step 2: Remove the five from the manifest source + delete + regenerate**

Delete lines 29-33 (`copy-on-select.sh`, `clipboard-to-pane.sh`, `flash-pane.sh`, `pair-title.sh`, `pair-session-watch.sh`) from `explicitAssetPaths`. Then:

```bash
git rm bin/copy-on-select.sh bin/clipboard-to-pane.sh bin/flash-pane.sh bin/pair-title.sh bin/pair-session-watch.sh
make runtimebundle-generate
```

- [ ] **Step 3: Tighten `embed_test.go`**

Three edits (the first is mandatory — `pair-title.sh`/`pair-session-watch.sh` are in the required-present `want` list, so the `want` loop would `t.Fatalf` before the `excluded` check is reached):
1. **Remove** `"bin/pair-title.sh"` (line 13) and `"bin/pair-session-watch.sh"` (line 14) from the `want` slice — they're no longer bundled.
2. **Add** `"bin/pair-title"` (the Go binary) to the `want` slice for symmetry — `"bin/pair-session-watch"` (Go) is already asserted present at line 22, but `bin/pair-title` (Go) is asserted nowhere after the `.sh` removal.
3. **Add** the five exec-shims (`bin/copy-on-select.sh`, `bin/clipboard-to-pane.sh`, `bin/flash-pane.sh`, `bin/pair-title.sh`, `bin/pair-session-watch.sh`) to the `excluded` slice.

Run: `make test-runtimebundle`
Expected: PASS.

- [ ] **Step 4: Run the copied-binary smoke + verify no stragglers**

Run: `make test-pair-embedded-runtime`
Expected: PASS — a copied binary extracts a bundle with the Go binaries and none of the seven shims.

Run: `grep -rn 'copy-on-select\.sh\|clipboard-to-pane\.sh\|flash-pane\.sh\|pair-title\.sh\|pair-session-watch\.sh' . --exclude-dir=.git --exclude-dir=workshop`
Expected: only historical-provenance comments remain — NO live invocation and NO manifest/`files/` entry (the regenerated bundle dropped them).

- [ ] **Step 5: Full test + commit**

Run: `env -u PAIR_SESSION_ID -u PAIR_TAG make test` (sandbox-off)
Expected: green — all launch/session/scrollback/review/continuation/clipboard flows pass with the shell-reduced bundle.

```bash
git add -A
git commit -m "#94 M2: delete the five exec-shims from the tree and bundle

With every caller repointed, bin/{copy-on-select,clipboard-to-pane,
flash-pane,pair-title,pair-session-watch}.sh are removed from the source
tree and explicitAssetPaths. The runtime bundle now carries only Go
binaries + native nvim/zellij assets + the six non-orchestrator shell
utilities (lib/*, pair-help, pair-notify, doctor/*) out of #93's scope.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 6: Atlas + M2 close

- [ ] **Step 1: Update the atlas**

Update `atlas/go-migration-inventory.md` to reflect the shell-reduced bundle: the seven orchestrator shims are gone; enumerate the six non-orchestrator shell files that remain and why they're out of scope. Keep `atlas/index.md` linking it. (ARCH-PURPOSE: record the honest "shell-reduced, not shell-free" endpoint.)

- [ ] **Step 2: M2 milestone close**

`sdlc milestone-close --issue 94 --milestone M2 --verified '<evidence>'` (measured `--actual`). Fix any Critical/Important. Then the issue close: `sdlc close --issue 94 --verified '<evidence>'` (the end-of-issue integration review), and publish via `sdlc merge` (or `sdlc push` if landed direct-on-main).

---

## Notes / risks

- **Silent-misroute trap (M1):** without the explicit `case "restart"/"quit"` in `ParseArgs`, the verbs fall through to the general loop and become an *agent name* (`Agent: "restart"`). Task 1's tests pin this.
- **`restart`/`quit` stay in the launcher router, NOT `dispatcher.Families()`** — they need the launcher `Runtime` (`WriteRestartMarker`/`ExecKillSession`/`InferAgent`), which the buffered dispatch path lacks. Do NOT add them to `DispatchNames()`.
- **single-instance argv guard (M2):** `pair-title.sh`'s comment confirms the Go poller's guard already matches the `…/pair-title <tag> <agent>` shape, so spawning the Go binary directly is the shape it recognizes. A transient double-poller is possible only across an upgrade from a still-running pre-repoint process — self-heals when that session ends (same class as the M1 note).
- **Bundle is generated, not hand-edited:** always change `explicitAssetPaths` then `make runtimebundle-generate`; never hand-edit `manifest.json` or `assets/runtime/files/` (ARCH-DRY — the generator is the single packaging source).
- **`PAIR_KILL_CMD` is the test seam** for the terminal `ExecKillSession` — the smoke test relies on it to observe marker writes without a real zellij.
```
