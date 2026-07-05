# Copy-on-select Detach Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop the copy-on-select paste from being reaped mid-chain by zellij, so the first (cold) copy after a restart reliably inserts into the draft.

**Architecture:** Split `copy-on-select` into a fast `copy_command` **hook** (mirror the selection to the clipboard, then return immediately) and a **detached orchestrator** (`copy-on-select --orchestrate`, spawned `setsid`) that does the slow in_nvim check + flash + hand-off to `clipboard-to-pane`. Because the orchestrator runs in its own session, zellij's reap of the `copy_command` child (uncatchable SIGKILL after ~1s) can no longer truncate the paste — correct on any machine, cold or warm. This is the root-cause fix chosen over prewarm (which only narrows the cold window and stays machine-speed dependent). See issue #100 Spec for the evidence.

**Tech Stack:** Go (`cmd/internal/clipcmd`), zellij CLI, macOS/Linux clipboard tools. Detach idiom already present in `runtime.go`'s `ResetPaneColorAfter` (`setsid` + `/dev/null` stdio) and the launcher's `spawnDetached`.

---

## Core concepts

### Pure entities (orchestration logic behind the injected Runtime)

The clip pipeline's decisions live in `run.go` as functions over the `Runtime` seam; they are unit-tested with a fake Runtime (no real IO). Per ARCH-PURE the decision logic stays here and the process/IO lives behind `Runtime`.

| Name | Lives in | Status |
|------|----------|--------|
| `RunCopyOnSelect` | `cmd/internal/clipcmd/run.go` | modified |
| `RunCopyOnSelectOrchestrate` | `cmd/internal/clipcmd/run.go` | new |

- **RunCopyOnSelect** — the `copy_command` hook. Now does only: read stdin, empty-guard, `ClipboardCopy(sel)`, `SpawnDetached(<pairHome>/bin/copy-on-select, "--orchestrate")`, return 0. No `ListPanes`/flash/`ExecReplace` inline — that's what made the hook slow enough to be reaped.
  - **Relationships:** 1:1 kicks off exactly one detached orchestrator per non-empty selection.
  - **DRY rationale:** Reuses the existing `SpawnDetached` detach idiom rather than inventing a new backgrounding mechanism (ARCH-DRY — same pattern as `ResetPaneColorAfter`, launcher `spawnDetached`).
  - **Future extensions:** If the hook ever needs to pass the selection out-of-band (rather than via the OS clipboard), the orchestrator argv is the seam.

- **RunCopyOnSelectOrchestrate** — the detached second half: `ListPanes(command)` → `focusedPane`/`isNvimCommand`; if in nvim, return 0 (don't loop back); else flash the source pane (`RunSubprocess` flash-pane) and `ExecReplace` clipboard-to-pane. This is the *current* `RunCopyOnSelect` tail, extracted verbatim (minus the stdin read + clipboard mirror, which the hook already did). It reads the selection from the OS clipboard via the clipboard-to-pane hand-off, so it needs no stdin.
  - **Relationships:** N/A — a leaf orchestration that terminates in `ExecReplace`.
  - **DRY rationale:** `focusedPane`, `isNvimCommand`, and the flash/hand-off calls are reused unchanged; nothing is duplicated between hook and orchestrator (the two halves are disjoint).
  - **Future extensions:** The orchestrator still issues its own `ListPanes(command)` and clipboard-to-pane issues another — a possible future ARCH-DRY consolidation into one list-panes. Out of scope here: once detached, the extra round-trip is off the reap path and harmless, and merging would change clipboard-to-pane's tested contract (regression risk on a bug fix).

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `SpawnDetached` | `cmd/internal/clipcmd/runtime.go` | new | `exec.Command` + `setsid` |

- **SpawnDetached(path string, args ...string)** — start `path args` in a new session (`Setsid: true`) with `/dev/null` stdio, inheriting the environment, and do not wait. Mirrors `ResetPaneColorAfter`'s detach exactly.
  - **Injected into:** `RunCopyOnSelect` (the hook), so the hook's "return fast, work continues detached" behavior is unit-testable with a fake that records the spawn instead of forking.
  - **Future extensions:** Any other clip helper that must outlive a short-lived hook.

### Removed (temporary diagnostic instrumentation)

The `[trace]` scaffolding added while diagnosing #100 is deleted in this work: the wall-clock timestamp prefix in `writeDebugLog`, the before/after timing around `ClipboardPaste`/`ListPanes`, the `ExecReplace` trace, `installSignalLogger` (+ its calls in `runcli.go`), and the `process entry` trace. `os/signal` import goes with it.

### Test surface

- **Go unit tests** (`cmd/internal/clipcmd/run_test.go`): `fakeRuntime` gains a `SpawnDetached` recorder. The hook test asserts *mirror + detached spawn and nothing else*; new orchestrate tests assert the in_nvim skip and the not-in-nvim flash+hand-off. This is the "hook does not run the slow chain inline" coverage from the issue's Done-when.
- **Shell test** (`tests/copy-on-select-test.sh`): the hand-off is now **asynchronous** (the orchestrator is `setsid`'d), so the test must **poll** for the handoff marker with a timeout rather than reading it once, and assert its **absence** (after a grace wait) for the in_nvim case.

---

## Chunk 1: Detach the orchestration

### Task 1: Add the `SpawnDetached` Runtime seam

**Files:**
- Modify: `cmd/internal/clipcmd/run.go` (Runtime interface: add method)
- Modify: `cmd/internal/clipcmd/runtime.go` (OSRuntime impl)
- Test: `cmd/internal/clipcmd/run_test.go` (fake impl)

- [ ] **Step 1: Add to the `Runtime` interface** in `run.go`, near `RunSubprocess`/`ExecReplace`:

```go
	// SpawnDetached starts path in a new session (setsid) with /dev/null stdio and
	// does NOT wait — the copy-on-select hook uses it to hand the flash+paste to a
	// process that outlives zellij's reap of the copy_command child (#100).
	SpawnDetached(path string, args ...string)
```

- [ ] **Step 2: Implement on `OSRuntime`** in `runtime.go`, mirroring `ResetPaneColorAfter`:

```go
// SpawnDetached starts path in its own session (setsid) with /dev/null stdio,
// inheriting the environment, and returns immediately (the copy-on-select hook's
// escape from zellij's copy_command reap — #100). Same idiom as ResetPaneColorAfter.
func (OSRuntime) SpawnDetached(path string, args ...string) {
	cmd := exec.Command(path, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if devNull, err := os.Open(os.DevNull); err == nil {
		cmd.Stdin, cmd.Stdout, cmd.Stderr = devNull, devNull, devNull
		defer devNull.Close()
	}
	if cmd.Start() == nil {
		go func() { _ = cmd.Wait() }()
	}
}
```

- [ ] **Step 3: Add the fake recorder** in `run_test.go` (`fakeRuntime` struct gets a field + method):

```go
	// spawned records SpawnDetached(path, args) calls (the detached orchestrator).
	spawned []spawnCall
```
```go
func (f *fakeRuntime) SpawnDetached(path string, args ...string) {
	f.spawned = append(f.spawned, spawnCall{path: path, args: args})
}
```
with `type spawnCall struct { path string; args []string }` near the top.

- [ ] **Step 4: Build to verify the interface is satisfied**

Run: `go build ./cmd/internal/clipcmd/`
Expected: PASS (fake + OSRuntime both implement Runtime).

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/clipcmd/run.go cmd/internal/clipcmd/runtime.go cmd/internal/clipcmd/run_test.go
git commit -m "#100: add SpawnDetached seam to the clip Runtime"
```

### Task 2: Split `RunCopyOnSelect` into hook + orchestrator (TDD)

**Files:**
- Modify: `cmd/internal/clipcmd/run.go`
- Test: `cmd/internal/clipcmd/run_test.go`

- [ ] **Step 1: Rewrite the hook tests to the new contract.** Replace `TestCopyOnSelectAgentPaneHandsOff` with a hook test and add orchestrator tests:

```go
func TestCopyOnSelectHookMirrorsThenDetaches(t *testing.T) {
	f := newFake()
	code := RunCopyOnSelect(copyOpts(), strings.NewReader("selected text"), f, io.Discard)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if f.copied != "selected text" {
		t.Errorf("clipboard = %q, want mirrored selection", f.copied)
	}
	if len(f.spawned) != 1 || f.spawned[0].path != "/h/bin/copy-on-select" ||
		len(f.spawned[0].args) != 1 || f.spawned[0].args[0] != "--orchestrate" {
		t.Errorf("spawned = %+v, want one detached /h/bin/copy-on-select --orchestrate", f.spawned)
	}
	// The hook must NOT run the slow chain inline — that is the reap bug (#100).
	if len(f.listed) != 0 || len(f.ran) != 0 || len(f.execd) != 0 {
		t.Errorf("hook ran slow chain inline: listed=%v ran=%v execd=%v", f.listed, f.ran, f.execd)
	}
}

func TestCopyOnSelectOrchestrateHandsOff(t *testing.T) {
	f := newFake()
	f.panes = agentFocusedPanesJSON // focused pane is the agent (not nvim)
	f.executable["/h/bin/flash-pane"] = true
	code := RunCopyOnSelectOrchestrate(copyOpts(), f, io.Discard)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if len(f.ran) != 1 || f.ran[0].path != "/h/bin/flash-pane" {
		t.Errorf("flash not run: %+v", f.ran)
	}
	if len(f.execd) != 1 || f.execd[0].path != "/h/bin/clipboard-to-pane" {
		t.Errorf("hand-off not exec'd: %+v", f.execd)
	}
}

func TestCopyOnSelectOrchestrateInNvimSkips(t *testing.T) {
	f := newFake()
	f.panes = nvimFocusedPanesJSON // focused pane is the nvim draft
	code := RunCopyOnSelectOrchestrate(copyOpts(), f, io.Discard)
	if code != 0 || len(f.ran) != 0 || len(f.execd) != 0 {
		t.Errorf("in_nvim must skip flash+handoff: code=%d ran=%v execd=%v", code, f.ran, f.execd)
	}
}
```

> Adapt field names (`copied`, `listed`, `ran`, `execd`, `panes`) to the existing `fakeRuntime` — Step 0 for the implementer is to read `run_test.go:20-95` and reuse its existing recorders and the panes-JSON fixtures already used by `TestCopyOnSelectInNvimSkips`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./cmd/internal/clipcmd/ -run 'CopyOnSelect' -v`
Expected: FAIL — `RunCopyOnSelectOrchestrate` undefined; hook still runs the chain inline.

- [ ] **Step 3: Rewrite `RunCopyOnSelect` and add `RunCopyOnSelectOrchestrate`** in `run.go`:

```go
// RunCopyOnSelect is zellij's copy_command body (the HOOK): mirror the selection
// to the OS clipboard, then hand the flash + paste to a DETACHED orchestrator and
// return immediately. Keeping the slow zellij round-trips out of the hook is what
// stops zellij reaping the child mid-paste (#100) — the hook must return fast on
// any machine, cold or warm.
func RunCopyOnSelect(opts CopyOnSelectOptions, stdin io.Reader, rt Runtime, stderr io.Writer) int {
	rt.LogFresh("=== copy-on-select invoked ===")
	sel, _ := io.ReadAll(stdin)
	if len(sel) == 0 {
		rt.Log("empty sel, exiting")
		return 0
	}
	// Mirror to the OS clipboard so other apps see it AND so the detached
	// orchestrator can read it back via clipboard-to-pane (best-effort copy).
	if err := rt.ClipboardCopy(string(sel)); err != nil {
		rt.Log("clipboard copy failed: " + err.Error())
	}
	// Detach the rest: the orchestrator (setsid) survives zellij's reap of this
	// copy_command child, so the paste completes even when the chain is slow.
	rt.SpawnDetached(opts.PairHome+"/bin/copy-on-select", "--orchestrate")
	rt.Log("detached orchestrator spawned; hook returning")
	return 0
}

// RunCopyOnSelectOrchestrate is the detached second half (#100): inspect the
// focused pane, and — unless the selection was made in the nvim draft — flash the
// source pane and hand off to clipboard-to-pane for the insert. Runs setsid'd, so
// zellij's reap of the copy_command child can't truncate it. The selection is
// already on the OS clipboard (the hook mirrored it), so this reads no stdin.
func RunCopyOnSelectOrchestrate(opts CopyOnSelectOptions, rt Runtime, stderr io.Writer) int {
	rt.Log("=== copy-on-select --orchestrate ===")
	focusedID := ""
	inNvim := false
	if out, err := rt.ListPanes(true); err == nil {
		if p, ok := focusedPane(zellijpane.Parse([]byte(out))); ok {
			focusedID = p.ID
			inNvim = isNvimCommand(p.TerminalCommand)
		}
	}
	rt.Log(fmt.Sprintf("in_nvim: %v focused_id: %q", inNvim, focusedID))
	if inNvim {
		return 0
	}
	flashScript := opts.PairHome + "/bin/flash-pane"
	if focusedID != "" && rt.Executable(flashScript) {
		if err := rt.RunSubprocess(flashScript, focusedID); err != nil {
			rt.Log("flash-pane failed: " + err.Error())
		}
	}
	clipScript := opts.PairHome + "/bin/clipboard-to-pane"
	if err := rt.ExecReplace(clipScript); err != nil {
		fmt.Fprintf(stderr, "copy-on-select: exec %s: %v\n", clipScript, err)
		return 1
	}
	return 0
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./cmd/internal/clipcmd/ -run 'CopyOnSelect' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/clipcmd/run.go cmd/internal/clipcmd/run_test.go
git commit -m "#100: split copy-on-select into fast hook + detached orchestrator"
```

### Task 3: Dispatch `--orchestrate` and remove temp instrumentation

**Files:**
- Modify: `cmd/copy-on-select/main.go` (pass args)
- Modify: `cmd/internal/clipcmd/runcli.go` (dispatch + drop instrumentation)
- Modify: `cmd/internal/clipcmd/runtime.go` (drop instrumentation)

- [ ] **Step 1: Pass args through** in `cmd/copy-on-select/main.go`:

```go
func main() {
	os.Exit(clipcmd.RunCopyOnSelectCLI(os.Args[1:], os.Stdin, os.Getenv, os.Stderr))
}
```

- [ ] **Step 2: Dispatch `--orchestrate`** in `runcli.go` (and drop the `installSignalLogger` temp call):

```go
func RunCopyOnSelectCLI(args []string, stdin io.Reader, getenv func(string) string, stderr io.Writer) int {
	home := getenv("PAIR_HOME")
	if home == "" {
		home = repoRootFromExe()
	}
	opts := CopyOnSelectOptions{PairHome: home}
	if len(args) > 0 && args[0] == "--orchestrate" {
		return RunCopyOnSelectOrchestrate(opts, NewOSRuntime(), stderr)
	}
	return RunCopyOnSelect(opts, stdin, NewOSRuntime(), stderr)
}
```
Also delete the `installSignalLogger("clipboard-to-pane")` call + `writeDebugLog("[trace] clipboard-to-pane process entry ...")` line from `RunClipboardToPaneCLI`.

- [ ] **Step 3: Remove instrumentation from `runtime.go`:** delete `installSignalLogger`, the `os/signal` import, the `[trace]` timing lines in `ClipboardPaste`/`ListPanes`, the `ExecReplace` `[trace]` line, and restore `writeDebugLog`'s final line to `f.WriteString(line + "\n")` (no timestamp prefix).

- [ ] **Step 4: Verify no `[trace]`/signal residue remains**

Run: `grep -rn "\[trace\]\|installSignalLogger\|os/signal" cmd/internal/clipcmd/ cmd/copy-on-select/`
Expected: no matches.

- [ ] **Step 5: Build + full clip tests**

Run: `go build ./cmd/... && go test ./cmd/internal/clipcmd/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/copy-on-select/main.go cmd/internal/clipcmd/runcli.go cmd/internal/clipcmd/runtime.go
git commit -m "#100: dispatch --orchestrate; remove temporary diagnostic instrumentation"
```

### Task 4: Make the shell regression test async

**Files:**
- Modify: `tests/copy-on-select-test.sh`

- [ ] **Step 1: Read the current assertions** (`tests/copy-on-select-test.sh`) — it checks `$tmp/handoff` immediately after running `copy-on-select`. The hand-off is now via a detached `copy-on-select --orchestrate`, so a synchronous check races.

- [ ] **Step 2: Replace the immediate check with a poll helper** for the not-in-nvim case:

```bash
# Wait up to ~3s for the detached orchestrator to reach the stubbed hand-off.
wait_for() { for _ in $(seq 1 60); do [ -e "$1" ] && return 0; sleep 0.05; done; return 1; }
```
Use `wait_for "$tmp/handoff" && pass || fail` where it asserted the hand-off was reached; for the in_nvim case, `sleep 0.5; [ ! -e "$tmp/handoff" ] && pass || fail` (grace wait then assert absence).

- [ ] **Step 3: Run the shell test**

Run: `make -f Makefile.local test-copy-on-select`
Expected: PASS (rebuilds `bin/copy-on-select`, drives the real detached chain).

- [ ] **Step 4: Commit**

```bash
git add tests/copy-on-select-test.sh
git commit -m "#100: make copy-on-select shell test await the detached hand-off"
```

### Task 5: Full verification + dogfood

- [ ] **Step 1: Full test suite** — `env -u PAIR_SESSION_ID -u PAIR_TAG make -f Makefile.local test` (scrub session env per the known make-test leak). Expected: green except the pre-existing `parley_harness_golden` failure.
- [ ] **Step 2: Rebuild the live binaries** — `go build -o bin/copy-on-select ./cmd/copy-on-select`.
- [ ] **Step 3: Dogfood the cold path** — restart `pair-dev` (forces the fleet rebuild → cold binaries), then the **first** copy from the agent pane into the draft. Confirm it inserts. Confirm `~/.cache/pair/clipboard-debug.log` shows the hook returning fast (`detached orchestrator spawned`) and a separate `=== copy-on-select --orchestrate ===` block completing.
- [ ] **Step 4: Update `atlas/` for the copy pipeline's new hook/orchestrator split; log the outcome in the issue `## Log`.**

---

## Notes / decisions

- **Why self-exec `--orchestrate` and not merge into clipboard-to-pane:** keeps `clipboard-to-pane` and `flash-pane` byte-for-byte unchanged (lower regression risk on a bug fix — Simplicity First), and keeps the orchestration logic co-located in `run.go` where its tests already live. The double `ListPanes` is harmless once off the reap path (see the orchestrator's Future-extensions note).
- **Ordering safety:** `ClipboardCopy` completes in the hook before the orchestrator's `pbpaste`, so no clipboard race is introduced; the focused pane is unchanged in the sub-100ms gap before the orchestrator's `ListPanes`, so the in_nvim gate stays correct.
- **ARCH citations:** ARCH-DRY (reuse `SpawnDetached`/setsid idiom, no new backgrounding mechanism), ARCH-PURE (decision logic in `run.go` behind the `Runtime` seam; `SpawnDetached` is the thin IO wrapper, faked in tests), ARCH-PURPOSE (fully empties the hook so the fix holds on slow/cold machines — the operator's stated concern — not just the fast common case).
