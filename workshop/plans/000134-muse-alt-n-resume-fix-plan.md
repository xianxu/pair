# Muse Alt+n Resume Fix Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `Alt+n` (and `Ctrl+Alt+n`) resume the previous `muse` session instead of starting a fresh one.

**Architecture:** Keep the existing 7-aspect harness diff (unstaged on `main` at `c12609d`, issue #000134 M1) as baseline. Harden the **pure** session-matching and resume-arg logic so the watcher captures the **root** `session.jsonl` (`YYYY/MM/DD/<uuid>/session.jsonl`) and ignores `…/subagent/…/session.jsonl`, and the launcher treats `muse resume <id>` identically to `codex` across `discover`/`AgentSessionExists`/`composeResumeArgs`. The thin-filesystem `Runtime` seam stays the only IO boundary; all decisions become unit-testable pure functions with a fake `Runtime`.

**Tech Stack:** Go (`cmd/internal/sessionwatch`, `cmd/internal/transcript`, `cmd/internal/launcher`), `zellij` config `zellij/config.kdl` (Alt+n forwarding), `nvim/init.lua` (`PairConfirmRestart` → `pair restart`), Muse CLI `muse resume <uuid>` + on-disk layout `~/.local/share/muse/sessions`, `lsof`/`stat` via `Runtime`.

---

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `AgentSpec.Match` | `cmd/internal/sessionwatch/sessionwatch.go` | modified |
| `discoverByBirth` | `cmd/internal/sessionwatch/run.go` | modified |
| `MuseTranscriptResolver` | `cmd/internal/transcript/transcript.go` (`Resolve`) | modified |
| `ResumeArgComposer` | `cmd/internal/launcher/agentargs.go` (`resumeToken`, `composeResumeArgs`, `stripCodexResumeSubcommand`) | modified |
| `ExplicitResumeExtractor` | `cmd/internal/launcher/createlogic.go` (`extractExplicitResume`) | modified |
| `ConfigPersistence` | `cmd/internal/launcher/config.go` (`MuseSessionsDir`, `CanonicalConfigPath`) | modified (already) |

- **AgentSpec.Match** — pure path → `SessionID` classifier for one agent. Returns `Matched/NearMiss/ID` or nothing. Must ignore sub-agent sessions.
  - **Relationships:** 1 `AgentSpec` : N `SessionID` candidates (the watcher filters to 1). Owned by `discover`/`discoverByBirth`.
  - **DRY rationale:** Single source for “what counts as a valid session file” — both the `lsof` leg and the birth-time leg call it. Duplicating the filter in `transcript.Resolve` or `launcher` would be ARCH-DRY violation.
  - **Future extensions:** New agent adds one `case` branch; add one table-driven test row. If Muse adds a new session layout, only this classifier changes.

- **discoverByBirth** — pure selection over `(file, birthTime)` list: candidates with `birth.After(agentStart)` and `Match.Matched`, picks newest `Matched` else newest `NearMiss`.
  - **Relationships:** 1 `discoverByBirth` consumes N `AgentSpec.Match` results; called by `discover` as fallback when `lsof` yields nothing (Muse does not hold `session.jsonl` open continuously).
  - **DRY rationale:** The old “exactly 1 candidate” gate dropped captures when multiple sessions shared a birth second; newest-by-birth is the one we just launched (ARCH-PURE: deterministic).
  - **Future extensions:** If Muse adds `subagent/` isolation, extend the filter predicate (exclude segment `subagent`).

- **MuseTranscriptResolver** — pure `Resolve("muse", sid)` → on-disk path via `Glob …/*/*/*/<sid>/session.jsonl`. Used by `OSRuntime.AgentSessionExists("muse")` which gates the restart picker’s `hasResumable`.
  - **Relationships:** 1 sid : 0..1 path. Injected into `launcher` via `Runtime.AgentSessionExists`; never shells out.
  - **DRY rationale:** `transcript` is the single source for “where is this agent’s transcript” — shared by `pair-slug` and `launcher`. Without it launcher would re-derive the glob (ARCH-DRY).
  - **Future extensions:** If Muse moves to flat layout, add branch here; tests cover both glob and flat fallback.

- **ResumeArgComposer** — pure `resumeToken`/`composeResumeArgs`/`stripCodexResumeSubcommand` family. Encodes per-agent syntax: `muse resume <id>` like `codex`, vs `--resume` vs `--conversation`.
  - **Relationships:** N call sites (`createflow` picker, `sessionwatch.StripResumeArgs`, pre-capture in `RunLaunch`) share the same composer so persisted `config-<tag>-muse.json` never stores a stale `resume` token (ARCH-DRY).
  - **Future extensions:** Additional resume flags add one `case` row; ordering invariant (“resume subcommand leads, flags trail”) stays tested.

- **ExplicitResumeExtractor** — pure `extractExplicitResume(agent, args) → sid` used to (a) skip the picker when argv already pins a resume and (b) synchronously write `config` before the watcher (which only catches NEW files).
  - **Relationships:** Composes with `codexResumeCommandIndex` which understands `codex [OPTIONS] resume` position-sensitivity.
  - **Future extensions:** If `muse` gains global options before `resume`, extend the index without touching picker logic.

- **ConfigPersistence** — pure path helpers (`MuseSessionsDir`, `CanonicalConfigPath`). No IO, just `filepath.Join`.
  - **Future extensions:** N/A — narrow path formatter.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `SessionWatcherRun` | `cmd/internal/sessionwatch/run.go` (`Run`, `discover`, `Runtime`) | modified | `lsof` + `stat` + `ListFiles` + `AtomicWrite` + `adapt.Logger` |
| `OSRuntime.AgentSessionExists` | `cmd/internal/launcher/osruntime.go` | modified | filesystem `transcript.Resolve` + `fileExists` |
| `RestartFlow` | `cmd/internal/launcher/restart.go` + `cmd/internal/launcher/createflow.go` + `nvim/init.lua:PairConfirmRestart` + `zellij/config.kdl: Alt n` | modified | `pair restart` marker, `zellij --- kill-session`, `ledger`/`config` writes, user confirm |
| `MuseCLI` | `cmd/internal/model/model.go` (`runMuse`) + live `muse resume/exec` binary | modified (already) | `muse` external binary |

- **SessionWatcherRun** — discovers the async session id and writes `config-<tag>-muse.json` + appends `ledger-<tag>.jsonl`.
  - **Injected into:** `AgentSpec.Match` + `discoverByBirth` (pure). Tests use a fake `Runtime` (in-memory `ListFiles`/`BirthTime`/`LsofPaths`, see `cmd/internal/sessionwatch/run_test.go` pattern).
  - **Future extensions:** Add `adapt` signal assertions in fake to catch drift (`session-id:fired/near-miss/fail`).

- **OSRuntime.AgentSessionExists** — thin wrapper around `transcript.Resolve` + `fileExists`. Gated by `Runtime` seam for tests.
  - **Injected into:** `runConfigPicker` (`hasResumable`) and `readSavedConfigForTag` fallback. Keeps `createlogic` unit-testable with a stub `Runtime`.

- **RestartFlow** — `Alt+n`/`Ctrl+Alt+n` → `WriteChars "\u{1b}[110;3u"` → `nvim PairConfirmRestart` → `pair restart [--new-session]` → `kill-session` → outer `RunLaunch` restart re-entry that replays `config` + ledger.
  - **Wraps:** `~/.cache/pair/restart-<session>` marker, kitty keyboard protocol forwarding, `zellij` session lifecycle.
  - **Future extensions:** `Shift+Alt+N` fresh-session variant (`--new-session`) uses same path but `hasResumable=false` by design.

- **MuseCLI** — external `muse` binary. Integration tests hit a stateful fake that mimics `session.jsonl` creation + `resume` flag handling; live conformance is “launch pair muse, check watcher capture” on the operator’s machine.
  - **Future extensions:** Model invocation already uses `muse exec` via `model.Run`.

**Test surface implied:** Every pure entity gets colocated table-driven unit tests (no IO mocks). Integration points get fake `Runtime` tests. End-to-end is a manual `pair muse` smoke (see Chunk 3) plus an `ls`/`cat` check of the on-disk artifacts — automated zellij E2E is intentionally not in plan per `workshop/lessons.md` (ephemeral client focus bugs).

---

## Chunk 1: Diagnose + harden session watcher (the resume miss)

### Task 1: Reproduce the Alt+n miss on the current tree

**Files:**
- Modify: `tmp/000134-diag.sh` (new, scratch, not committed)
- Read: `cmd/internal/sessionwatch/sessionwatch.go`, `cmd/internal/sessionwatch/run.go`, `cmd/internal/transcript/transcript.go`, `cmd/internal/launcher/osruntime.go:523`

- [ ] **Step 1: Run the targeted harness suites on the dirty tree**

Run: `GOCACHE=/tmp/gocache go test ./cmd/internal/sessionwatch ./cmd/internal/transcript ./cmd/internal/launcher -run "TestAgentSpec|TestResolve|TestResume|TestCompose" -count=1 -v 2>&1 | tail -n 80`
Expected: PASS for existing rows (including `muse` leading `resume` after M1), confirming baseline.

- [ ] **Step 2: Show the subagent leak in the current `Match`**

Run: `go test -run TestMuseMatchIgnoresSubagent ./cmd/internal/sessionwatch -count=1 -v` (expected: no such test → `no test to run`, proving coverage gap).
Then manual probe:
```bash
python3 -c "
import pathlib, re, subprocess, textwrap, json, os
home='/tmp/home'
# Simulate current Match: parent dir of session.jsonl is taken as id without depth/subagent check
# The bug is that .../<root-uuid>/subagent/<sub-uuid>/session.jsonl also matches (parent is sub-uuid)
print('subagent path would incorrectly match as id = sub-uuid')
"
```
Expected: confirms hypothesis that `Match` today extracts `subagent/<subuuid>` id.

- [ ] **Step 3: Inspect live Muse session tree**

Run: `ls -R ~/.local/share/muse/sessions/2026/08/14 | head -n 80; echo "---"; cat cmd/internal/sessionwatch/run.go | grep -n "subagent"`
Expected: many `…/<root-uuid>/subagent/<sub-uuid>/session.jsonl` siblings; `run.go` has no `subagent` filter.

### Task 2: Fix `AgentSpec.Match` for `muse` to exclude `subagent` sessions

**Files:**
- Modify: `cmd/internal/sessionwatch/sessionwatch.go:43-78`
- Test: `cmd/internal/sessionwatch/sessionwatch_test.go`

ARCH: ARCH-PURE (pure classifier), ARCH-DRY (single filter), ARCH-PURPOSE (Alt+n resume is the purpose, not just “watcher writes something”).

- [ ] **Step 1: Write failing test — subagent must not match**

```go
func TestMuseMatchIgnoresSubagentSession(t *testing.T) {
    home := "/tmp/home"
    sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
    subSid := "123e4567-e89b-12d3-a456-426614174000"
    // Root session should match
    spec, _ := SpecForAgent("muse", home)
    rootPath := home + "/.local/share/muse/sessions/2026/08/14/" + sid + "/session.jsonl"
    got := spec.Match(rootPath)
    if !got.Matched || got.ID != sid {
        t.Fatalf("root muse match = %+v, want id %q", got, sid)
    }
    // Subagent session must NOT be considered a valid top-level session
    subPath := home + "/.local/share/muse/sessions/2026/08/14/" + sid + "/subagent/" + subSid + "/session.jsonl"
    got2 := spec.Match(subPath)
    if got2.Matched {
        t.Fatalf("subagent path should not match, got %+v", got2)
    }
    // Near-miss case: malformed uuid under subagent also ignored
    badSub := home + "/.local/share/muse/sessions/2026/08/14/" + sid + "/subagent/not-a-uuid/session.jsonl"
    got3 := spec.Match(badSub)
    if got3.Matched {
        t.Fatalf("bad subagent path should not match, got %+v", got3)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/internal/sessionwatch -run TestMuseMatchIgnoresSubagent -count=1 -v`
Expected: FAIL on `subagent path should not match` (current code matches `subSid`).

- [ ] **Step 3: Implement minimal fix in `Match` for `muse`**

In `cmd/internal/sessionwatch/sessionwatch.go`, `case "muse":` add guard before basename check:

```go
case "muse":
    prefix := filepath.Clean(s.WatchDir) + string(filepath.Separator)
    clean := filepath.Clean(path)
    if !strings.HasPrefix(clean, prefix) {
        return SessionID{}
    }
    // Muse subagent sessions live under …/<root-uuid>/subagent/<sub-uuid>/session.jsonl.
    // Only the root session is resumable via `muse resume <id>`; ignore subagent interior.
    if strings.Contains(clean, string(filepath.Separator)+"subagent"+string(filepath.Separator)) {
        return SessionID{}
    }
    if filepath.Base(clean) != "session.jsonl" {
        return SessionID{}
    }
    id := filepath.Base(filepath.Dir(clean))
    if uuidRE.MatchString(id) {
        return SessionID{Matched: true, ID: id, Path: path}
    }
    return SessionID{Matched: true, NearMiss: true, Path: path}
```

Note: use `strings.Contains` on `clean` (filepath-normalized). Alternative is to split/`filepath.Rel` and reject depth>4, but `subagent` string is the durable marker Muse uses. Keep it explicit.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/internal/sessionwatch -run TestMuseMatchIgnoresSubagent -count=1 -v`
Expected: PASS.

- [ ] **Step 5: Run broader sessionwatch suite**

Run: `go test ./cmd/internal/sessionwatch -count=1 -v 2>&1 | tail -n 40`
Expected: all PASS (existing + new).

### Task 3: Fix `discover`/`discoverByBirth` to never return a subagent id

**Files:**
- Modify: `cmd/internal/sessionwatch/run.go:219-290` (`discover`, `discoverByBirth`)
- Test: `cmd/internal/sessionwatch/run_test.go` (new cases)

ARCH-PURE — pure selection.

- [ ] **Step 1: Write failing test — birth-time tie with mixed root/subagent**

```go
func TestDiscoverByBirthPrefersRootOverSubagent(t *testing.T) {
    home := "/tmp/home"
    spec, _ := SpecForAgent("muse", home)
    rootID := "019eff64-6ceb-7e72-9d41-a735a97029ac"
    subID  := "123e4567-e89b-12d3-a456-426614174000"
    rootPath := home + "/.local/share/muse/sessions/2026/08/14/" + rootID + "/session.jsonl"
    subPath  := home + "/.local/share/muse/sessions/2026/08/14/" + rootID + "/subagent/" + subID + "/session.jsonl"
    // Fake runtime: ListFiles returns both, BirthTime equal (same second)
    // After Task 2 fix, subPath Match returns {} so only root qualifies.
    // This test guards regression even if Match filter were removed — discover must also ignore subagent.
    // Setup uses existing FakeRuntime pattern in run_test.go (stub ListFiles/BirthTime/Match).
    // Assert result.ID == rootID
}
```

Simplify: if test harness is heavy, at minimum add a `discover` test where `LsofPaths` returns a subagent path and a root path, asserting root wins.

- [ ] **Step 2: Run to fail (or to pass after Task 2, then strengthen)**

Run: `go test ./cmd/internal/sessionwatch -run TestDiscoverByBirth -count=1 -v`
Expected: either FAIL (if `discover` still considers subagent via `lsof` leg) or PASS (if `Match` already filtered). Either way, add explicit guard in `discover`: skip any `result` where `strings.Contains(path, "/subagent/")`.

- [ ] **Step 3: Add defense-in-depth in `discover` + `discoverByBirth`**

In `run.go` `discover` loop (lsof leg) and `discoverByBirth` candidate loop, early-continue if `strings.Contains(file, "/subagent/")` or `strings.Contains(path, "/subagent/")`. This makes the watcher correct even if `Match` were changed later (ARCH-DRY: two layers, but the `subagent` fact is one predicate — extract `isMuseSubagentPath(path) bool` helper if duplicated).

```go
func isMuseSubagentPath(p string) bool {
    return strings.Contains(p, string(filepath.Separator)+"subagent"+string(filepath.Separator))
}
```

Apply in:
- `discover` lsof inner loop: `if s.Agent=="muse" && isMuseSubagentPath(path) { continue }`
- `discoverByBirth` file loop: same check before `Match`.

- [ ] **Step 4: Run suite**

Run: `GOCACHE=/tmp/gocache go test ./cmd/internal/sessionwatch -count=1 -v`
Expected: PASS.

### Task 4: Harden `transcript.Resolve` for `muse` against subagent Glob pollution

**Files:**
- Modify: `cmd/internal/transcript/transcript.go:38-56`
- Test: `cmd/internal/transcript/transcript_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestResolveMuseIgnoresSubagent(t *testing.T) {
    // Using a temp home with both root and subagent session.jsonl present,
    // Resolve("muse", sid) must return the root path, not a subagent one.
    // Today Glob `*/*/*/<sid>/session.jsonl` will still find root correctly,
    // but if sid IS a subagent id, it must return "" (subagent not resumable).
}
```

- [ ] **Step 2: Implement if needed**

`Resolve` for `muse` already Globs `sessions/*/*/*/sid/session.jsonl` — that naturally only finds `YYYY/MM/DD/<sid>/session.jsonl`, so a `subagent` sid won’t match as a top-level date dir (it would require `sessions/*/*/*/subAgentSid/session.jsonl` which would be `sessions/2026/08/<subagentSid>/...` — no such file). The Glob is already safe, but add explicit guard: if resolved path contains `/subagent/`, discard it and return `""`. Keep flat fallback (`sessions/<sid>/session.jsonl`) as test aid but also guard.

- [ ] **Step 3: Run**

Run: `go test ./cmd/internal/transcript -count=1 -v`
Expected: PASS.

- [ ] **Step 4: Commit Chunk 1**

```bash
git add cmd/internal/sessionwatch/sessionwatch.go cmd/internal/sessionwatch/run.go cmd/internal/sessionwatch/sessionwatch_test.go cmd/internal/sessionwatch/run_test.go cmd/internal/transcript/transcript.go cmd/internal/transcript/transcript_test.go
git commit -m "#134 M2: fix Muse watcher to ignore subagent sessions

Match/discover now exclude …/subagent/…/session.jsonl so Alt+n
captures the root session id, not a subagent id. Transcript resolve
guards the same path. ARCH-DRY/ARCH-PURE."
```

---

## Chunk 2: Launcher resume parity for `muse` (the picker gate)

### Task 5: Ensure `AgentSessionExists("muse")` is the gate that actually passes

**Files:**
- Modify: `cmd/internal/launcher/osruntime.go:523-540`
- Test: `cmd/internal/launcher/osruntime_test.go` (extend `TestOSRuntimeAgentSessionExists*`)

- [ ] **Step 1: Write failing test if missing**

```go
func TestOSRuntimeAgentSessionExistsFindsMuseRoot(t *testing.T) {
    home := t.TempDir()
    sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
    rootPath := filepath.Join(home, ".local", "share", "muse", "sessions", "2026", "08", "14", sid, "session.jsonl")
    os.MkdirAll(filepath.Dir(rootPath), 0o755)
    os.WriteFile(rootPath, []byte("{}\n"), 0o644)
    if !(OSRuntime{}).AgentSessionExists("muse", sid, "/repo") {
        t.Fatal("AgentSessionExists(muse) should find root session.jsonl")
    }
    subSid := "123e4567-e89b-12d3-a456-426614174000"
    if (OSRuntime{}).AgentSessionExists("muse", subSid, "/repo") {
        t.Fatal("subagent sid must not be considered resumable")
    }
}
```

- [ ] **Step 2: Run to verify**

Run: `go test ./cmd/internal/launcher -run TestOSRuntimeAgentSessionExistsFindsMuseRoot -count=1 -v`
Expected: PASS after M1 (but the negative subagent case is new — should PASS after Chunk 1’s Resolve guard; otherwise FAIL then fix `transcript.Resolve` guard).

### Task 6: Verify `resumeToken`/`composeResumeArgs`/`extractExplicitResume`/`StripResumeArgs` already cover `muse`

**Files:**
- Read: `cmd/internal/launcher/agentargs.go:55-180`
- Test: `cmd/internal/launcher/agentargs_test.go` (add `muse` rows)
- Read: `cmd/internal/launcher/createlogic.go:50-80`

Current M1 already added `muse` to `resumeToken` (`resume <sid>`), `composeResumeArgs` (leading), `extractExplicitResume` (with fallback `args[0]=="resume"`). Chunk 2’s job is to **lock it with tests** and fix `stripCodexResumeSubcommand` naming/cover.

- [ ] **Step 1: Add table rows for muse**

In `agentargs_test.go`:
```go
func TestResumeTokenPerAgent(t *testing.T) {
    // existing plus:
    // {"muse", "s1", []string{"resume", "s1"}},
}
func TestComposeResumeArgsOrdering(t *testing.T) {
    // add: muse leading same as codex
    if got := composeResumeArgs("muse", []string{"--model","x"}, "sid"); !reflect.DeepEqual(got, []string{"resume","sid","--model","x"}) { … }
}
func TestStripResumeArgsRemovesMuseResume(t *testing.T) {
    // StripResumeArgs("muse", ["resume","abc","--model","x"]) == ["--model","x"]
}
func TestExtractExplicitResumeMuse(t *testing.T) {
    // leading resume: ["resume","sid-1", "--model","x"] → "sid-1"
    // with globals before resume: if muse supports globals, ensure index handling
}
```

- [ ] **Step 2: Run**

Run: `go test ./cmd/internal/launcher -run "TestResume|TestCompose|TestStrip|TestExtract" -count=1 -v`
Expected: PASS (if any FAIL, fix `agentargs.go` — e.g., generalize `codexResumeCommandIndex` to also handle `muse` globals, or keep the `args[0]=="resume"` fallback which already covers the canonical `composeResumeArgs` placement).

- [ ] **Step 3: Audit `FreshAgentArgs` / `persistedConfigArgs` don’t strip `muse` resume incorrectly**

Check `cmd/internal/launcher/agentargs.go:FreshAgentArgs` — it calls `stripCodexResumeSubcommand` which now has a `muse` fallback. Ensure it works for `muse` when invoked via `RunLaunch` fresh-session path. Add test.

### Task 7: End-to-end picker path: `runConfigPicker` offers resume when `hasResumable` true for muse

**Files:**
- Modify: `cmd/internal/launcher/createflow.go:550-585` (no change, just test)
- Test: `cmd/internal/launcher/createflow_test.go` or `pick_test.go`

- [ ] **Step 1: Add fake-Runtime test**

Stub `Runtime` where `AgentSessionExists("muse", savedID)=true, saved Args clean. Call `runConfigPicker` with `saved.SessionID=sid`. Assert `composeTagRestartArgs` produced `["resume", sid, …]` and `*agentArgs` mutated to that. Also test negative: when `AgentSessionExists=false`, `choices` lack `saved+resume` and `hasResumable` false.

Run: `go test ./cmd/internal/launcher -run "TestRunConfigPicker|TestBuildConfigChoices" -count=1 -v`

- [ ] **Step 2: Commit Chunk 2**

```bash
git add cmd/internal/launcher/osruntime.go cmd/internal/launcher/agentargs.go cmd/internal/launcher/agentargs_test.go cmd/internal/launcher/osruntime_test.go
git commit -m "#134 M2: launcher resume parity for muse (tests)

Resume token, compose ordering, strip, and explicit-resume extraction
covered for muse == codex. AgentSessionExists(muse) verified against
root session.jsonl. ARCH-DRY: single composer for all resume sites."
```

---

## Chunk 3: Rebuild, smoke Alt+n, and close drift surface

### Task 8: Rebuild and verify harness suites green

**Files:**
- Modify: none (build artifact)
- Read: `Makefile.local`

- [ ] **Step 1: Build**

Run: `GOCACHE=/tmp/gocache go test ./... 2>&1 | tail -n 30`
Expected: same 3 sandbox flakes as baseline, no new failures. If new failure, fix.

Run: `go build -o /tmp/pair ./cmd/pair && /tmp/pair --help 2>&1 | head`

### Task 9: Manual smoke — Alt+n resumes muse

**Files:**
- None (operator procedure, result recorded in issue Log)

ARCH-PURPOSE — the Done-when is Alt+n resume.

- [ ] **Step 1: Fresh muse workbench with a tag**

Run: `pair muse -- --help` (should show Muse help through pair wrap, return-remap active).
Then: `pair muse` with tag e.g. `pair-muse-smoke` (or auto-generated `📁pair-…`).
In the workbench:
- Type a prompt (“explain what pair does in one sentence”), send via `Alt+Enter`, observe response.
- Check watcher captured: in a separate terminal, `cat ~/.cache/pair/config-<tag>-muse.json` and `cat ~/.local/share/pair/ledger-<tag>.jsonl | tail -1` — `session_id` must be the root UUID, not a subagent UUID; `adapt-<tag>.jsonl` should have `session-id:fired`.
- Check `AgentSessionExists` gate: `go run ./cmd/pair -- help` not needed; instead run `pair list` — row shows `muse` agent, tag, and `session_id` suffix.

- [ ] **Step 2: Alt+n reload**

Press `Alt+n` (or `Ctrl+Alt+n` on macOS Tahoe), confirm `Yes` in the nvim modal. Session should reload with same tag, and Muse pane should show conversation history resumed (previous prompt+response still in scrollback / resumed context). Verify:
- `Alt+/` then `Alt+b` jumps to the earlier prompt `^>` line.
- Draft is intact.
- Second `Alt+n` again resumes (not fresh).
- `Shift+Alt+N` (“Restart only the coding agent with a fresh conversation”) starts a genuinely fresh Muse session (new `session.jsonl` under a new UUID) — draft/layout survive.

- [ ] **Step 3: If smoke fails, collect diagnostics**

`cat ~/.cache/pair/adapt-<tag>.jsonl`, `cat ~/.cache/pair/config-<tag>-muse.json`, `ls -lt ~/.local/share/muse/sessions/2026/08/14/*/session.jsonl | head`, `lsof -p <pair-wrap-pid> | grep session.jsonl` (expected: no open file for muse — confirms birth-time fallback is the live path). Paste findings into issue Log.

### Task 10: Docs + sdlc close

**Files:**
- Modify: `workshop/issues/000134-muse-harness-support.md` (Log, Plan ticks)
- Modify: `atlas/how-to-bring-up-a-new-harness-cli.md` if any new invariant discovered (subagent exclusion)
- Modify: `atlas/index.md` if needed

- [ ] **Step 1: Update issue**

Append to `## Log` the smoke evidence (session ids, adapt lines, Alt+n result). Tick remaining plan checkboxes.

- [ ] **Step 2: Atlas note (one line is enough)**

In `atlas/how-to-bring-up-a-new-harness-cli.md` “Session watcher” paragraph, add: “For Muse, exclude `…/subagent/…/session.jsonl` — only the root `YYYY/MM/DD/<uuid>/session.jsonl` is resumable.”

- [ ] **Step 3: sdlc close**

Run: `sdlc close --issue 000134 --verified "go test ./cmd/internal/sessionwatch ./cmd/internal/transcript ./cmd/internal/launcher -count=1 PASS; manual pair muse smoke: Alt+n resumes root session <uuid>, adapt session-id:fired, Alt+b jumps, Shift+Alt+N fresh verified"` (or `sdlc milestone-close` if splitting M2).

---

## Verification summary

- **Unit:** `go test ./cmd/internal/sessionwatch -run TestMuseMatchIgnoresSubagent`, `TestDiscover*`, `go test ./cmd/internal/transcript -run TestResolveMuse`, `go test ./cmd/internal/launcher -run "TestResume|TestCompose|TestExtract|TestOSRuntimeAgentSessionExistsFindsMuseRoot"` — all PASS.
- **Integration (fake Runtime):** `runConfigPicker` offers `saved+resume` when `AgentSessionExists(muse)=true`, suppresses it when false.
- **Manual:** `pair muse` → prompt → `Alt+n` resumes same conversation; `adapt-<tag>.jsonl` shows `session-id:fired`; `config-<tag>-muse.json` holds root UUID; `Shift+Alt+N` starts fresh.
- **No regressions:** `go test ./...` same 3 sandbox flakes as baseline.

## Risks

- Muse could change its session layout (e.g., drop `YYYY/MM/DD` or rename `subagent/`). Mitigated by `adapt` near-miss signal (`session-id:near-miss` with matched file but no id) and `doctor` drift check.
- `lsof` never shows `session.jsonl` for Muse (continuous fallback to birth-time). Current `discoverByBirth` picks newest by birth; if two root sessions share identical birth nanosecond, tie-break is filesystem order — acceptable because tag-scoped watchers rarely race.
- Kit forwarding `Alt+n` on macOS requires `Ctrl+Alt+n` alias — already bound in `zellij/config.kdl:216`.

