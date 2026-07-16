# alt+shift+c Deterministic Restart + Draft-Fold Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `alt+shift+c` compaction deterministic — the continuation writer itself triggers the session restart (no agent step to forget), and the operator's parked draft WIP is folded into the continuation's NEXT ACTION so it survives the restart.

**Architecture:** Both behaviors move into the one binary that #104 unified. The `pair continuation` writer, when it detects it is running inside its own pair pane (compaction context, read from `PAIR_TAG`/`ZELLIJ_SESSION_NAME`), (a) folds the stripped draft file into the body's `## NEXT ACTION` before writing, and (b) after the write+commit+push, re-invokes `pair continue <slug>` on itself — reusing the existing, tested `runCompaction` → outer-reincarnation-loop path. `COMPACT_PROMPT` collapses to a single "write the continuation" instruction. The pure logic (comment-strip, NEXT-ACTION fold, context gate) is unit-tested without IO; the draft read and the restart exec are injected IO seams.

**Tech Stack:** Go (`cmd/internal/continuationcmd`), Lua (`nvim/init.lua`), the launcher's fake-`Runtime` test harness.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `StripStickyComments` | `cmd/internal/continuationcmd/draft.go` | new |
| `FoldDraftIntoNextAction` | `cmd/internal/continuationcmd/draft.go` | new |
| `InCompactionContext` | `cmd/internal/continuationcmd/draft.go` | new |

- **StripStickyComments(s string) string** — drop the draft's sticky-comment lines (those matching `^\s*===`, the `=== label ===` stickies) and trim leading/trailing blank lines. The comment WIP that remains is what gets folded.
  - **Relationships:** 1:1 with the draft file's content; consumed only by the writer's fold path.
  - **DRY rationale:** A minimal Go mirror of the Lua `strip_comments` (`nvim/init.lua:995`). Pinned to that source with a comment + a fixture unit test. A cross-language drift test (headless-nvim over shared fixtures) is deliberately **not** built: pair has no Lua unit harness, and `^\s*===` is a trivial, stable convention — the [[inline-copy + drift test]] lesson's own test ("would behavior be *wrong* if the guard were missing, or just less ergonomic?") lands on "less ergonomic" here. The pinning comment names the Lua source so a future convention change touches both.
  - **Future extensions:** If the sticky grammar grows beyond `===`, promote to a shared token both sides read.

- **FoldDraftIntoNextAction(body, wip string) string** — insert `wip` into `body`'s `## NEXT ACTION` section (after the section's existing content, before the next `## ` heading or EOF), under a one-line `_Parked draft at compaction:_` label. No-op (returns `body` unchanged) when `wip` is empty or `body` has no `## NEXT ACTION`.
  - **Relationships:** Reuses the section-scan shape of `firstNextActionLine` (`continuation.go:81`); operates on the agent-authored body before `Assemble`.
  - **DRY rationale:** One place that knows "where NEXT ACTION content lives" for insertion, mirroring the existing scan used for validation/preview.
  - **Future extensions:** Could fold into other named sections if a future feature wants "parked links → Artifact map."

- **InCompactionContext(pairTag, zellijSession string) bool** — the gate: `pairTag != "" && zellijSession == "pair-"+pairTag`. True ⇒ the writer is inside its own live pane, so fold + restart apply; false ⇒ a standalone `pair continuation`, write-only.
  - **Relationships:** Mirrors the tag-match half of the launcher's `compactionDecision` (`compaction.go:22`) — the same guard against acting on a sibling session's leaked env.
  - **DRY rationale:** First occurrence of "is this writer a compaction?" — keeps the env-shape decision out of `run()`'s IO body.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| draft-file read | `cmd/internal/continuationcmd/continuationcmd.go` | modified | filesystem (`os.ReadFile`) |
| restart trigger seam | `cmd/internal/continuationcmd/continuationcmd.go` | new | `os/exec` (re-invoke `pair continue`) |
| env inputs (`PAIR_TAG`/`PAIR_DATA_DIR`/`ZELLIJ_SESSION_NAME`) | `cmd/internal/continuationcmd/continuationcmd.go` | modified | process env |
| `PairConfirmCompact` / `COMPACT_PROMPT` | `nvim/init.lua` | modified | operator keybinding |

- **draft-file read** — `run()` computes `<PAIR_DATA_DIR>/draft-<PAIR_TAG>.md` (data dir via the canonical `adapt.DataDir()`, `cmd/internal/adapt/adapt.go:78`) and reads it when `InCompactionContext`. Missing/empty file ⇒ no fold.
  - **Injected into:** the fold path; the file content is passed through `StripStickyComments` → `FoldDraftIntoNextAction`.
- **restart trigger seam** — a `restart func(slug string) error` field on the run inputs. Real impl: `exec.Command(os.Executable(), "continue", slug)` with inherited env + stdio (so `PAIR_DEV`, `ZELLIJ_SESSION_NAME`, `PAIR_TAG` ride through → `compactionDecision` fires and pair/pair-dev config is preserved). Called only after a successful write+commit+push and only when `InCompactionContext` (unless `--no-restart`). The kill-session inside `runCompaction` tears the whole session down; the outer loop relaunches.
  - **Injected into:** `run()`, so the fake in tests records the slug without killing anything.
- **PairConfirmCompact / COMPACT_PROMPT** — `PairConfirmCompact()` saves the draft buffer to disk first (so the writer reads fresh content), then sends a **single-step** `COMPACT_PROMPT` ("write the continuation via `pair continuation`") — step 2 (`pair continue`) is removed.
  - **Future extensions:** none needed; the determinism now lives in the writer.

---

## Task 1: `StripStickyComments`

**Files:**
- Create: `cmd/internal/continuationcmd/draft.go`
- Test: `cmd/internal/continuationcmd/draft_test.go`

- [ ] **Step 1: Write the failing test** — fixtures mirroring the Lua rule (drop `^\s*===` lines; trim blank edges; interior blanks kept):

```go
func TestStripStickyComments(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"drops === lines", "=== label ===\nreal WIP\n=== end ===", "real WIP"},
		{"trims blank edges", "\n\n  hi\n\n", "hi"},
		{"keeps interior blanks", "a\n\nb", "a\n\nb"},
		{"indented === also a comment", "   === x ===\nkeep", "keep"},
		{"all comments -> empty", "=== a ===\n=== b ===", ""},
		{"empty -> empty", "", ""},
	}
	for _, c := range cases {
		if got := StripStickyComments(c.in); got != c.want {
			t.Errorf("%s: StripStickyComments(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails** — `go test ./cmd/internal/continuationcmd/ -run TestStripStickyComments` → FAIL (undefined).

- [ ] **Step 3: Write minimal implementation** in `draft.go`:

```go
package continuationcmd

import "strings"

// StripStickyComments drops the draft's sticky-comment lines (matching `===`
// after optional leading whitespace) and trims leading/trailing blank lines.
// Minimal Go mirror of nvim/init.lua's strip_comments (the `=== label ===`
// convention) — pinned here so a change to that convention touches both. No
// cross-lang drift test: pair has no Lua unit harness and `^\s*===` is trivial.
func StripStickyComments(s string) string {
	var out []string
	for _, ln := range strings.Split(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(ln), "===") {
			continue
		}
		out = append(out, ln)
	}
	for len(out) > 0 && strings.TrimSpace(out[0]) == "" {
		out = out[1:]
	}
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}
```

- [ ] **Step 4: Run test to verify it passes** — same command → PASS.
- [ ] **Step 5: Commit** — `git add cmd/internal/continuationcmd/draft.go cmd/internal/continuationcmd/draft_test.go && git commit -m "#105: StripStickyComments — Go mirror of nvim draft comment-strip"`

---

## Task 2: `FoldDraftIntoNextAction`

**Files:**
- Modify: `cmd/internal/continuationcmd/draft.go`
- Test: `cmd/internal/continuationcmd/draft_test.go`

- [ ] **Step 1: Write the failing test:**

```go
func TestFoldDraftIntoNextAction(t *testing.T) {
	body := "## NEXT ACTION\n\nrun the thing\n\n## State of play\n\ndone"
	got := FoldDraftIntoNextAction(body, "half-typed idea")
	if !strings.Contains(got, "run the thing") || !strings.Contains(got, "half-typed idea") {
		t.Fatalf("WIP not folded under NEXT ACTION:\n%s", got)
	}
	// folded WIP must sit inside NEXT ACTION, before the next section
	na := strings.Index(got, "## NEXT ACTION")
	sop := strings.Index(got, "## State of play")
	wip := strings.Index(got, "half-typed idea")
	if !(na < wip && wip < sop) {
		t.Fatalf("WIP not positioned inside NEXT ACTION section:\n%s", got)
	}
	// no-ops
	if FoldDraftIntoNextAction(body, "") != body {
		t.Error("empty WIP should be a no-op")
	}
	if got := FoldDraftIntoNextAction("## Other\n\nx", "wip"); strings.Contains(got, "wip") {
		t.Error("no NEXT ACTION -> no fold")
	}
	// section at EOF (no trailing heading)
	eof := FoldDraftIntoNextAction("## NEXT ACTION\n\ndo it", "tail wip")
	if !strings.Contains(eof, "tail wip") {
		t.Error("EOF section: WIP not folded")
	}
}
```

- [ ] **Step 2: Run test to verify it fails** — `go test ./cmd/internal/continuationcmd/ -run TestFoldDraftIntoNextAction` → FAIL (undefined).

- [ ] **Step 3: Write minimal implementation** in `draft.go` (reuse the ATX-heading rule from `continuation.go`'s `isATXHeading`):

```go
// FoldDraftIntoNextAction inserts wip into body's `## NEXT ACTION` section —
// after the section's existing content, before the next `## ` heading or EOF —
// under a labelled block so the operator's parked draft survives the restart.
// No-op if wip is blank or body has no NEXT ACTION section.
func FoldDraftIntoNextAction(body, wip string) string {
	wip = strings.TrimRight(wip, "\n")
	if strings.TrimSpace(wip) == "" {
		return body
	}
	lines := strings.Split(body, "\n")
	start := -1
	for i, ln := range lines {
		if strings.TrimSpace(ln) == "## NEXT ACTION" {
			start = i
			break
		}
	}
	if start == -1 {
		return body
	}
	end := len(lines) // default: section runs to EOF
	for i := start + 1; i < len(lines); i++ {
		if isATXHeading(strings.TrimSpace(lines[i])) {
			end = i
			break
		}
	}
	// trim trailing blanks inside the section so the insert reads cleanly
	insert := end
	for insert > start+1 && strings.TrimSpace(lines[insert-1]) == "" {
		insert--
	}
	block := append([]string{"", "_Parked draft at compaction:_", ""}, strings.Split(wip, "\n")...)
	out := append([]string{}, lines[:insert]...)
	out = append(out, block...)
	out = append(out, lines[insert:]...)
	return strings.Join(out, "\n")
}
```

- [ ] **Step 4: Run test to verify it passes** → PASS.
- [ ] **Step 5: Commit** — `git commit -am "#105: FoldDraftIntoNextAction — insert parked WIP into NEXT ACTION"`

---

## Task 3: `InCompactionContext`

**Files:**
- Modify: `cmd/internal/continuationcmd/draft.go`
- Test: `cmd/internal/continuationcmd/draft_test.go`

- [ ] **Step 1: Write the failing test:**

```go
func TestInCompactionContext(t *testing.T) {
	if !InCompactionContext("mytag", "pair-mytag") {
		t.Error("matching tag+session should be compaction context")
	}
	if InCompactionContext("", "pair-") {
		t.Error("empty tag is never compaction")
	}
	if InCompactionContext("mytag", "pair-other") {
		t.Error("sibling session (leaked env) must not match")
	}
	if InCompactionContext("mytag", "") {
		t.Error("no zellij session -> not in a pane")
	}
}
```

- [ ] **Step 2: Run to verify it fails.**
- [ ] **Step 3: Implement** in `draft.go`:

```go
// InCompactionContext reports whether the writer is running inside its own live
// pair pane — the gate for fold + writer-triggered restart. Mirrors the
// tag-match half of the launcher's compactionDecision (compaction.go), guarding
// against a sibling pane's leaked ZELLIJ_SESSION_NAME.
func InCompactionContext(pairTag, zellijSession string) bool {
	return pairTag != "" && zellijSession == "pair-"+pairTag
}
```

- [ ] **Step 4: Run to verify it passes.**
- [ ] **Step 5: Commit** — `git commit -am "#105: InCompactionContext gate for writer fold+restart"`

---

## Task 4: Wire fold into the writer's `run()`

**Files:**
- Modify: `cmd/internal/continuationcmd/continuationcmd.go`
- Test: `cmd/internal/continuationcmd/continuation_test.go` (drive `Run`/`run` against a temp repo)

- [ ] **Step 0: Build the `run()`-level harness (it does NOT exist yet).** `continuation_test.go` has only pure-unit tests, and `run()` constructs `gitRunner{root}` internally (real git IO, non-injected) — so this is an **integration test built from scratch**. Helper: init a real temp git repo (`git init`, set `user.email`/`user.name`), set `a.repoRoot` = temp repo (skips `rev-parse`), and tolerate the non-fatal `git push origin HEAD` failure (no `origin` remote → prints to `os.Stderr`; that's fine). Add a `readWrittenContinuation(t, repo)` that globs `workshop/continuation/*.md` and reads the single file. This harness is reused by Task 5.

- [ ] **Step 1: Write the failing test** — using the Step 0 harness, drive the writer with a compaction env (`runEnv{PairTag,DataDir,ZellijSession}`) + a draft file + an injected no-op restart seam, and assert the written continuation carries the WIP under NEXT ACTION.

```go
func TestRun_FoldsDraftWhenInCompaction(t *testing.T) {
	repo := t.TempDir()            // git-init helper as existing tests use
	dataDir := t.TempDir()
	os.WriteFile(filepath.Join(dataDir, "draft-mytag.md"),
		[]byte("=== sticky ===\nfinish the parser\n"), 0o644)
	env := runEnv{PairTag: "mytag", DataDir: dataDir, ZellijSession: "pair-mytag"}
	body := "## NEXT ACTION\n\nreview PR\n"
	// ... call run() with a fake restart seam; read the written file ...
	out := readWrittenContinuation(t, repo)
	if !strings.Contains(out, "finish the parser") {
		t.Fatalf("draft WIP not folded:\n%s", out)
	}
	if strings.Contains(out, "=== sticky ===") {
		t.Error("comments should have been stripped")
	}
}
```

- [ ] **Step 2: Run to verify it fails.**
- [ ] **Step 3: Implement.** Add a `runEnv{PairTag, DataDir, ZellijSession string}` struct; `Run` populates it from `os.Getenv` (`PAIR_TAG`, `adapt.DataDir()`, `ZELLIJ_SESSION_NAME`) and passes it to `run`. In `run`, after `readBody` and before the `HasNextAction` guard:

```go
if InCompactionContext(env.PairTag, env.ZellijSession) {
	draft := filepath.Join(env.DataDir, "draft-"+env.PairTag+".md")
	if raw, err := os.ReadFile(draft); err == nil {
		if wip := StripStickyComments(string(raw)); wip != "" {
			body = FoldDraftIntoNextAction(body, wip)
		}
	}
}
```

(Fold *before* `HasNextAction`/write so the persisted, committed, pushed doc already carries the WIP.)

- [ ] **Step 4: Run to verify it passes**; then `go test ./cmd/internal/continuationcmd/` all green.
- [ ] **Step 5: Commit** — `git commit -am "#105: writer folds stripped draft WIP into NEXT ACTION in compaction context"`

---

## Task 5: Writer-triggered restart

**Files:**
- Modify: `cmd/internal/continuationcmd/continuationcmd.go`
- Test: `cmd/internal/continuationcmd/continuation_test.go`

- [ ] **Step 1: Write the failing tests** — (a) in compaction context, the injected restart seam is called once with the written slug, after write; (b) standalone (no PAIR_TAG), the seam is never called; (c) `--no-restart` suppresses it even in compaction.

```go
func TestRun_TriggersRestartInCompaction(t *testing.T) {
	var gotSlug string; called := 0
	restart := func(slug string) error { gotSlug = slug; called++; return nil }
	// run() in compaction env with slug "resume-parser" ...
	if called != 1 || gotSlug != "resume-parser" {
		t.Fatalf("restart not triggered with slug: called=%d slug=%q", called, gotSlug)
	}
}
func TestRun_NoRestartStandalone(t *testing.T) {
	called := 0
	restart := func(string) error { called++; return nil }
	// run() with empty PairTag ...
	if called != 0 { t.Fatal("standalone write must not restart") }
}
```

- [ ] **Step 2: Run to verify they fail.**
- [ ] **Step 3: Implement.** Add `--no-restart` to the flag set and a `restart func(string) error` to the run inputs. `Run`'s real seam:

```go
restart := func(slug string) error {
	exe, err := os.Executable()
	if err != nil { return err }
	c := exec.Command(exe, "continue", slug)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run() // runCompaction inside kills the session; outer loop relaunches
}
```

At the very end of `run` (after the `fmt.Fprintln(stdout, abs)` success print):

```go
if !a.noRestart && InCompactionContext(env.PairTag, env.ZellijSession) {
	if err := restart(f.Slug); err != nil {
		fmt.Fprintf(os.Stderr, "pair-continuation: restart failed (continuation kept): %v\n", err)
	}
}
```

- [ ] **Step 4: Run to verify they pass**; `go test ./cmd/internal/continuationcmd/` green.
- [ ] **Step 5: Commit** — `git commit -am "#105: writer triggers pair continue restart in compaction context (--no-restart escape)"`

---

## Task 6: Simplify the nvim compaction flow

**Files:**
- Modify: `nvim/init.lua` (`COMPACT_PROMPT` ~3324, `PairConfirmCompact` ~3327, and the comment block ~3301)

- [ ] **Step 1: Save the draft before prompting.** In `PairConfirmCompact()`, before `send_to_agent(COMPACT_PROMPT)`, persist the draft buffer to `draft_path_for_tag()` so the writer reads fresh content (use the existing buffer-write path; if the star buffer isn't focused, write the file from `read`+`write` helpers already in init.lua).

- [ ] **Step 2: Collapse `COMPACT_PROMPT` to one step** — remove step 2 (`pair continue <slug>` / `pair-dev continue`); the writer now restarts. New body:

```lua
local COMPACT_PROMPT = table.concat({
  'Compact this session: write a continuation doc for this session NOW by',
  "following this project's continuation DATATYPE procedure — first flush key",
  '   exchanges to pensive, then distill per that procedure and finalize with',
  '   `pair continuation` (workshop/continuation/). Choose a short slug.',
  'The writer restarts this session automatically once the doc is written —',
  '   do not run anything after it.',
}, '\n')
```

- [ ] **Step 3: Update the comment block** (~3301-3314) to describe the new flow (writer-triggered restart; draft WIP folded into NEXT ACTION) and keep the "defer to the datatype procedure; no inline skeleton" note (pair#61).

- [ ] **Step 4: Sync the bundled mirror.** If `nvim/init.lua` is mirrored under `cmd/internal/runtimebundle/assets/...`, regenerate/copy so the two stay byte-identical (the #104 review flagged this drift class). Verify with a diff.

- [ ] **Step 5: Commit** — `git commit -am "#105: single-step COMPACT_PROMPT + save draft before compaction; writer owns restart"`

---

## Task 7: Verification

- [ ] **Step 1: Full Go suite** — `go test ./...` from repo root → all green (esp. `continuationcmd`, `launcher`).
- [ ] **Step 2: Relevant shell tests** — run the compaction/continue-adjacent `tests/*.sh` if any touch this path; note results.
- [ ] **Step 3: Live smoke (manual, the one path unit tests can't drive).** In a real pair session: type WIP into the draft, hit `alt+shift+c`, confirm. Expect: a continuation doc is written under `workshop/continuation/`, its NEXT ACTION contains the parked WIP (sans `===` lines), and the session restarts fresh under the same tag (pair vs pair-dev preserved) seeded from the doc — **without** the agent running any `pair continue` itself. Capture the doc path + a note in `## Log`.
- [ ] **Step 4: Negative check** — run `pair continuation ...` from a non-pane shell (standalone) and confirm it only writes (no restart), preserving the standalone contract.

---

## Notes for the executor

- **One review boundary.** Cohesive single-pass work → plain checkboxes, one `sdlc close` (AGENTS.md §3). No `Mx` tags.
- **ARCH-PURE:** the three `draft.go` entities are pure + unit-tested without IO; the draft read and restart exec are injected seams (fakes in tests) — keep them out of the pure functions.
- **ARCH-DRY:** reuse `adapt.DataDir()`, `isATXHeading`, and the `pair continue` path; do not re-derive the data dir or re-implement the compaction/kill/relaunch.
- **Root cause:** the fix is *removing the dependence on the agent's step 2*, not hardening the prompt to make the agent more reliable.
