# Pair Session Change Log (`Alt+l`) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an `Alt+l` change log — an LLM-distilled, append-mostly list of session milestones/decisions, opened read-only and full-screen in nvim (the distilled counterpart to `Alt+/`'s raw scrollback).

**Architecture:** A thin shell orchestrator (`bin/pair-changelog-open`) acquires a lock, cleans the captured TTY via the existing `pair-scrollback-render`, runs a Go distiller, and `exec`s the nvim viewer. The Go distiller (`cmd/pair-changelog`) holds all testable logic as pure functions — anchor `locate`, turn-boundary scan, prior-log split, and date-aware assembly — over a thin IO seam that calls the model and writes files atomically. The per-agent model dispatch is extracted out of `pair-slug`'s `package main` into a shared `cmd/internal/model` package (parameterizing the OpenAI output-token cap so multi-entry logs don't truncate on Codex). Spec: `workshop/issues/000053-create-visualization-of-change-log-in-a-pair-session.md`.

**Tech Stack:** Go (distiller + shared model package), POSIX sh (orchestrator), Lua (nvim viewer), KDL (zellij keybind). Tests: Go `testing` with a process-level fake model on `PATH`; headless `nvim -l` Lua test.

**Architecture principles in play:**
- **ARCH-DRY** — reuse `pair-scrollback-render` for cleaning; extract `pair-slug`'s model dispatch into a shared package rather than copy it; carry only a 3-glyph synced copy of `scrollback.lua`'s prompt patterns (a shared cross-language store would be over-engineering for three stable tokens).
- **ARCH-PURE** — `locate`, `scanTurnBoundaries`, `splitFrozenTail`, `assemble`, `lastHeaderDate`, `anchorSnippet` are pure (deterministic, no IO), unit-tested without mocks; the only IO seam is `main.go` (read files, call model, atomic write) + the shell.

---

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `LocateResult` | `cmd/pair-changelog/distill.go` | new |
| `locate` | `cmd/pair-changelog/distill.go` | new |
| `scanTurnBoundaries` | `cmd/pair-changelog/distill.go` | new |
| `splitFrozenTail` | `cmd/pair-changelog/distill.go` | new |
| `splitFirstEntry` | `cmd/pair-changelog/distill.go` | new |
| `lastHeaderDate` | `cmd/pair-changelog/distill.go` | new |
| `assemble` | `cmd/pair-changelog/distill.go` | new |
| `anchorSnippet` | `cmd/pair-changelog/distill.go` | new |
| `promptGlyphByAgent` | `cmd/pair-changelog/distill.go` | new |
| `parseAnchor` | `cmd/pair-changelog/distill.go` | new (M2) |

- **`LocateResult`** — `{Kind LocateKind; Start int}` where `Kind ∈ {Found, NoOp, FullRedistill}`. `Start` is the slice start index (0 for `FullRedistill`); the slice end is always `len(lines)`. `NoOp` carries no slice.
  - **Relationships:** produced by `locate`, consumed by `main.go`'s flow.
  - **DRY rationale:** one result type for the three timing outcomes, so the caller branches once.
- **`locate(lines, anchor, turnBoundaries, lookbackTurns, lineCap) LocateResult`** — the incremental-boundary decision. Searches `lines` from the end for the exact `anchor` block (newest occurrence wins); returns `NoOp` if the anchor sits flush with the end, `FullRedistill` if not found (or only partially), else `Found` with `Start` = walk back from the match past `lookbackTurns` boundaries, clamped by `lineCap`.
  - **Future extensions:** `lookbackTurns`/`lineCap` already parameterized; a future caller could pass different budgets.
- **`scanTurnBoundaries(lines, agent) []int`** — indices of user-prompt-marker lines, via `promptGlyphByAgent`. Pure scan.
- **`splitFrozenTail(log) (frozenPrefix, lastEntry string)`** — splits the prior log into the byte-verbatim frozen prefix (everything up to, not including, the last bullet block) and the last bullet block `Ek` (`""` if none). Date headers are structure, never part of a bullet block.
- **`splitFirstEntry(modelOut) (first, rest string)`** — splits the model's tail output into its first bullet block (`Ek'`) and the remaining new entries, so the assembler can insert a date header between them on a day rollover.
- **`lastHeaderDate(log) string`** — the date of the last `## YYYY-MM-DD` header (`""` if none).
- **`assemble(frozenPrefix, ekPrime, newEntries, today, lastDate string) string`** — deterministic log assembly with date-header ownership (see History contract in the spec). Header invariant: a `## date` is emitted **only** immediately before ≥1 bullet, so the parser never sees a header with no following bullet.
- **`anchorSnippet(lines, k) []string`** — the last `k` lines of the cleaned text (the next anchor).
- **`promptGlyphByAgent`** — `map[string]*regexp.Regexp`: `claude ^❯`, `codex ^›`, `agy ^>` (a **deliberate simplification** of scrollback.lua's box-aware agy pattern — safe under the spec's graceful-degradation guarantee, since turns drive only the lookback). **Sync-commented to `nvim/scrollback.lua` `PROMPT_PATTERN_BY_AGENT`** (ARCH-DRY: accepted minimal duplication).

**Test surface:** all of the above are unit-tested in `cmd/pair-changelog/distill_test.go` with no IO. `locate` is exercised across growth / no-change / shrink / anchor-recurs (newest wins) / not-found / turn-walk (<2 turns → start 0; line cap). `assemble` across same-day+new / rollover+new / revise-only-no-new / first-ever. `splitFrozenTail` asserts the frozen prefix is byte-identical to the input prefix.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `model.Run` | `cmd/internal/model/model.go` | new (extracted) | agent CLI / OpenAI API |
| `cmd/pair-changelog` main | `cmd/pair-changelog/main.go` | new | files + model |
| `bin/pair-changelog-open` | `bin/pair-changelog-open` | new | lock, renderer subprocess, nvim |
| `nvim/changelog.lua` | `nvim/changelog.lua` | new | nvim UI |
| `Alt l` keybind | `zellij/config.kdl` | modified | zellij |
| `pair-slug` model calls | `cmd/pair-slug/main.go` | modified | now calls `model.Run` |
| `writeAnchor` | `cmd/pair-changelog/main.go` | new (M2) | atomic file write |

- **`model.Run(Request) (string, error)`** — per-agent model dispatch, lifted verbatim from `pair-slug`'s unexported helpers, with `MaxOutputTokens` + `Verbosity` now parameters (the OpenAI path hardcoded `64`/`low`).
  - **Injected into:** both `pair-slug` (passes 64/low) and `pair-changelog` (passes a generous budget). Keeps the two binaries DRY over one dispatch.
  - **Future extensions:** add providers, or a streaming variant.
- **`cmd/pair-changelog` main** — reads the cleaned-text file + prior log + anchor, runs the pure pipeline, calls `model.Run`, writes log then anchor atomically. Date injected via `--today` (real date by default) for testability.
  - **Injected into:** the pure functions receive its file contents; it is the thin seam.
- **`bin/pair-changelog-open`** — lock (PID, `kill -0`), clean TTY via `pair-scrollback-render --plain --max-lines 0`, run distiller, `exec` nvim. Models on `bin/pair-scrollback-open`.
- **`nvim/changelog.lua`** — read-only full-screen viewer + token colorization.
- **Fake model (test scaffolding, part of the deliverable):** a `claude` shell script on `PATH` that reads stdin and prints a canned change log — a process-level fake, exercised by the distiller's integration test.

---

## Chunk 1: M1 — shared model package + Go distiller

### Task 1: Extract `cmd/internal/model` and refactor `pair-slug` onto it

**Files:**
- Create: `cmd/internal/model/model.go`
- Create: `cmd/internal/model/model_test.go`
- Modify: `cmd/pair-slug/main.go` (remove the moved helpers; call `model.Run`)
- Modify: `cmd/pair-slug/main_test.go` (drop the moved-helper tests; they move to `model_test.go`)

- [ ] **Step 1: Create the package by moving the dispatch verbatim, then parameterize output length.** Move `runModel`, `runClaudeModel`, `runAgyModel`, `runCodexCLIModel`, `runOpenAIModel`, `responseText`, `defaultModel`, and the `modelTimeout` / `defaultClaudeModel` / `defaultOpenAIModel` constants from `cmd/pair-slug/main.go` into `cmd/internal/model/model.go` as package `model`. Expose one entry point:

```go
package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	Timeout            = 30 * time.Second
	DefaultClaudeModel = "claude-haiku-4-5" // move verbatim from pair-slug main.go:49
	DefaultOpenAIModel = "gpt-5.4-mini"     // move verbatim from pair-slug main.go:50
)

// Request is one model call. MaxOutputTokens and Verbosity are honored by the
// OpenAI Responses path; the CLI paths (claude/agy/codex) ignore them (no cap).
type Request struct {
	Agent           string // "claude" | "codex" | "agy"
	Model           string // "" → DefaultModel(Agent)
	Prompt          string // instructions / system prompt
	Input           string // content on stdin
	MaxOutputTokens int    // OpenAI hard cap; must be > 0
	Verbosity       string // "low"|"medium"|"high"; "" → "low"
}

func DefaultModel(agent string) string {
	if agent == "codex" {
		return DefaultOpenAIModel
	}
	return DefaultClaudeModel
}

func Run(r Request) (string, error) {
	if r.Model == "" {
		r.Model = DefaultModel(r.Agent)
	}
	if r.Verbosity == "" {
		r.Verbosity = "low"
	}
	switch r.Agent {
	case "codex":
		if os.Getenv("OPENAI_API_KEY") != "" {
			return runOpenAI(r)
		}
		return runCodexCLI(r)
	case "agy":
		return runAgy(r)
	default:
		return runClaude(r)
	}
}
```

In `runOpenAI`, replace the hardcoded body with the request's values:

```go
body := map[string]any{
	"model":             r.Model,
	"instructions":      r.Prompt,
	"input":             r.Input,
	"max_output_tokens": r.MaxOutputTokens,
	"text":              map[string]string{"verbosity": r.Verbosity},
}
```

Keep `ResponseText` exported (the test exercises it). The CLI helpers keep `PAIR_SLUG_NESTED=1` (rename later if desired; out of scope here) and `PAIR_SLUG_OPENAI_BASE_URL` override so the existing tests still work.

- [ ] **Step 2: Move the model-specific tests into `model_test.go` and run them (RED→GREEN).** Move the `responseText` / `runOpenAIModel` (now `ResponseText` / `runOpenAI`) test cases from `cmd/pair-slug/main_test.go` into `cmd/internal/model/model_test.go`, updating identifiers and adding an assertion that `MaxOutputTokens` is forwarded (point `PAIR_SLUG_OPENAI_BASE_URL` at an `httptest` server that echoes the received `max_output_tokens` and assert it equals the value passed, e.g. 64 and 2000).

Run: `go test ./cmd/internal/model/...`
Expected: PASS (including the new max-tokens-forwarding assertion).

- [ ] **Step 3: Refactor `pair-slug` to call `model.Run` and delete the moved code.** Replace `pair-slug`'s `runModel(agent, m, prompt, input)` call sites with:

```go
out, err := model.Run(model.Request{
	Agent: agent, Model: m, Prompt: prompt, Input: input,
	MaxOutputTokens: 64, Verbosity: "low",
})
```

Delete the now-moved functions/constants from `main.go`. Update `main_test.go` to remove the relocated tests and keep the slug-behavior tests (which now go through `model.Run`).

- [ ] **Step 4: Run the full pair-slug + model suite (GREEN).**

Run: `go test ./cmd/pair-slug/... ./cmd/internal/model/...`
Expected: PASS. This is the refactor-safety gate the spec promises.

- [ ] **Step 5: Commit.**

```bash
git add cmd/internal/model cmd/pair-slug
git commit -m "#53 M1: extract cmd/internal/model from pair-slug; parameterize output length"
```

### Task 2: Pure turn-boundary scan + `locate`

**Files:**
- Create: `cmd/pair-changelog/distill.go`
- Create: `cmd/pair-changelog/distill_test.go`

- [ ] **Step 1: Write failing tests for `scanTurnBoundaries` and `locate`.**

```go
package main

import (
	"reflect"
	"testing"
)

func TestScanTurnBoundaries(t *testing.T) {
	lines := []string{"hello", "❯ first prompt", "work", "❯ second prompt", "more"}
	got := scanTurnBoundaries(lines, "claude")
	want := []int{1, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestLocateFoundWalksBackTwoTurns(t *testing.T) {
	// boundaries at 1 and 3; anchor is line "more" (index 4, the end).
	// New content after anchor: none → but we want the *found-with-new* case,
	// so put new lines after the anchor.
	lines := []string{"a", "❯ t1", "b", "❯ t2", "c", "anchorX", "new1", "new2"}
	anchor := []string{"anchorX"}
	res := locate(lines, anchor, scanTurnBoundaries(lines, "claude"), 2, 200)
	if res.Kind != Found {
		t.Fatalf("kind=%v want Found", res.Kind)
	}
	// anchor at index 5; walk back past 2 boundaries (3, then 1) → start at 1.
	if res.Start != 1 {
		t.Fatalf("start=%d want 1", res.Start)
	}
}

func TestLocateNoOpWhenAnchorFlushWithEnd(t *testing.T) {
	lines := []string{"a", "❯ t1", "b", "anchorX"}
	res := locate(lines, []string{"anchorX"}, scanTurnBoundaries(lines, "claude"), 2, 200)
	if res.Kind != NoOp {
		t.Fatalf("kind=%v want NoOp", res.Kind)
	}
}

func TestLocateFullRedistillWhenAnchorMissing(t *testing.T) {
	lines := []string{"a", "b", "c"}
	res := locate(lines, []string{"nope"}, nil, 2, 200)
	if res.Kind != FullRedistill || res.Start != 0 {
		t.Fatalf("got %+v want {FullRedistill 0}", res)
	}
}

func TestLocateNewestOccurrenceWins(t *testing.T) {
	lines := []string{"X", "a", "X", "b"} // "X" recurs; newest is index 2
	res := locate(lines, []string{"X"}, nil, 2, 200)
	if res.Kind != Found || res.Start != 2 { // no boundaries → start clamps to match
		t.Fatalf("got %+v want Found start 2", res)
	}
}

func TestLocateLineCap(t *testing.T) {
	lines := make([]string, 500)
	for i := range lines {
		lines[i] = "x"
	}
	lines[450] = "anchorX"
	res := locate(lines, []string{"anchorX"}, nil, 2, 200)
	// no boundaries before 450 → would walk to 0, but cap holds at 450-200=250.
	if res.Kind != Found || res.Start != 250 {
		t.Fatalf("got %+v want Found start 250", res)
	}
}
```

- [ ] **Step 2: Run to verify failure.** `go test ./cmd/pair-changelog/...` → FAIL (undefined symbols).

- [ ] **Step 3: Implement.**

```go
package main

import "regexp"

// promptGlyphByAgent — sync-commented to nvim/scrollback.lua PROMPT_PATTERN_BY_AGENT.
// claude/codex are faithful single-glyph ports. agy is a DELIBERATE
// SIMPLIFICATION: scrollback uses a box-aware, multi-line variant
// (`\(──.*\n\)\zs>` — a `>` only when the prior line starts with `──`); we
// approximate with `^>`. Over-matching is safe because turns drive ONLY the
// lookback — a false boundary just shortens context (graceful degradation, per
// spec), never breaks the anchor. A faithful port (also require lines[i-1] to
// start with `──`) is a future refinement if agy log quality suffers.
var promptGlyphByAgent = map[string]*regexp.Regexp{
	"claude": regexp.MustCompile(`^❯`),
	"codex":  regexp.MustCompile(`^›`),
	"agy":    regexp.MustCompile(`^>`),
}

func glyphFor(agent string) *regexp.Regexp {
	if re, ok := promptGlyphByAgent[agent]; ok {
		return re
	}
	return promptGlyphByAgent["claude"]
}

func scanTurnBoundaries(lines []string, agent string) []int {
	re := glyphFor(agent)
	var out []int
	for i, l := range lines {
		if re.MatchString(l) {
			out = append(out, i)
		}
	}
	return out
}

type LocateKind int

const (
	Found LocateKind = iota
	NoOp
	FullRedistill
)

type LocateResult struct {
	Kind  LocateKind
	Start int
}

// locate finds the newest exact occurrence of anchor (a contiguous block of
// lines) and decides the slice start. anchor is matched as consecutive lines.
func locate(lines, anchor []string, turnBoundaries []int, lookbackTurns, lineCap int) LocateResult {
	if len(anchor) == 0 {
		return LocateResult{Kind: FullRedistill, Start: 0}
	}
	matchAt := -1
	for i := len(lines) - len(anchor); i >= 0; i-- { // newest-first
		if linesEqual(lines[i:i+len(anchor)], anchor) {
			matchAt = i
			break
		}
	}
	if matchAt < 0 {
		return LocateResult{Kind: FullRedistill, Start: 0}
	}
	anchorEnd := matchAt + len(anchor) // index just past the anchor
	if anchorEnd >= len(lines) {
		return LocateResult{Kind: NoOp}
	}
	// Walk back from the match past lookbackTurns boundaries.
	start := matchAt
	count := 0
	for i := len(turnBoundaries) - 1; i >= 0 && count < lookbackTurns; i-- {
		if turnBoundaries[i] < matchAt {
			start = turnBoundaries[i]
			count++
		}
	}
	if count < lookbackTurns {
		// fewer than lookbackTurns boundaries available → from the start,
		// unless the line cap pulls it back.
		start = 0
	}
	if matchAt-start > lineCap {
		start = matchAt - lineCap
	}
	return LocateResult{Kind: Found, Start: start}
}

func linesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Run to verify pass.** `go test ./cmd/pair-changelog/...` → PASS.

- [ ] **Step 5: Commit.**

```bash
git add cmd/pair-changelog/distill.go cmd/pair-changelog/distill_test.go
git commit -m "#53 M1: pure turn-boundary scan + anchor locate (ARCH-PURE)"
```

### Task 3: Pure prior-log split + date-aware assembly

**Files:**
- Modify: `cmd/pair-changelog/distill.go`
- Modify: `cmd/pair-changelog/distill_test.go`

- [ ] **Step 1: Write failing tests** for `splitFrozenTail`, `splitFirstEntry`, `lastHeaderDate`, `assemble`, `anchorSnippet`.

```go
func TestSplitFrozenTailByteExact(t *testing.T) {
	log := "## 2026-06-12\n\n- one\n\n- two\n\n- three\n"
	frozen, ek := splitFrozenTail(log)
	if frozen != "## 2026-06-12\n\n- one\n\n- two\n\n" {
		t.Fatalf("frozen=%q", frozen)
	}
	if ek != "- three\n" {
		t.Fatalf("ek=%q", ek)
	}
	if frozen+ek != log {
		t.Fatalf("frozen+ek must reconstruct the log byte-for-byte")
	}
}

func TestAssembleSameDayAppend(t *testing.T) {
	got := assemble("## 2026-06-12\n\n- one\n\n", "- two\n", "- three\n", "2026-06-12", "2026-06-12")
	want := "## 2026-06-12\n\n- one\n\n- two\n\n- three\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAssembleRolloverInsertsHeader(t *testing.T) {
	got := assemble("## 2026-06-12\n\n- one\n\n", "- two\n", "- three\n", "2026-06-13", "2026-06-12")
	want := "## 2026-06-12\n\n- one\n\n- two\n\n## 2026-06-13\n\n- three\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAssembleReviseOnlyNoNew(t *testing.T) {
	got := assemble("## 2026-06-12\n\n- one\n\n", "- two-revised\n", "", "2026-06-13", "2026-06-12")
	want := "## 2026-06-12\n\n- one\n\n- two-revised\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAssembleFirstEver(t *testing.T) {
	got := assemble("", "", "- a\n\n- b\n", "2026-06-12", "")
	want := "## 2026-06-12\n\n- a\n\n- b\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
```

- [ ] **Step 2: Run to verify failure.** → FAIL.

- [ ] **Step 3: Implement** `splitFrozenTail`, `splitFirstEntry` (same block-parse, first instead of last), `lastHeaderDate` (regex `(?m)^## (\d{4}-\d{2}-\d{2})\s*$`, last match), `assemble`, `anchorSnippet`. Assembly rule (matches the spec, and the example's `## D\n\n-` spacing — normalize each block to end in exactly one `\n`, separate blocks with one blank line):

```go
// assemble builds the new log. frozenPrefix is byte-verbatim. ekPrime is the
// revised last entry ("" on first-ever). newEntries is the model's new bullets
// ("" if none). A "## today" header is inserted before newEntries iff there
// are new entries AND today != lastDate. Invariant: a header is only ever
// emitted immediately before ≥1 bullet.
func assemble(frozenPrefix, ekPrime, newEntries, today, lastDate string) string {
	var b strings.Builder
	b.WriteString(frozenPrefix)
	if ekPrime != "" {
		b.WriteString(ensureBlock(ekPrime))
	}
	if newEntries != "" {
		if today != lastDate {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString("## " + today + "\n\n")
		} else if ekPrime != "" {
			b.WriteString("\n")
		}
		b.WriteString(ensureBlock(newEntries))
	}
	return b.String()
}
```

(`ensureBlock` trims trailing whitespace and appends a single `\n`; the byte-exact normalization is pinned by the tests above — spec-review round-3 advisory #1.) Add the derivable invariant as a doc comment (advisory #2): "a `## date` header is never written without ≥1 following bullet."

- [ ] **Step 4: Run to verify pass.** → PASS.

- [ ] **Step 5: Commit.** `git commit -m "#53 M1: pure prior-log split + date-aware assembly (ARCH-PURE)"`

### Task 4: Distiller `main.go` + Go integration test with a fake model

**Files:**
- Create: `cmd/pair-changelog/main.go`
- Create: `cmd/pair-changelog/main_test.go`
- Create: `cmd/pair-changelog/prompt.go` (the model prompt builder)

- [ ] **Step 1: Write the prompt builder** (`prompt.go`) — first-run vs incremental, with the category vocabulary and the freeze/revise/format rules (no headers, no timestamps, 1–2 sentences, blank line between, revise-last-never-drop). Keep it a pure function `buildPrompt(firstRun bool) (system string)` plus a pure `buildInput(frozen, ek, slice string) string` so it is unit-testable.

- [ ] **Step 2: Write the integration test first (RED).** Build the binary, put a fake `claude` on `PATH` that prints a canned tail, run the binary against fixtures, assert outputs. Cover: first-run; incremental append (frozen prefix byte-identical); last-entry revision; Ek-never-dropped; date-rollover (`--today`); no-op (anchor flush → log unchanged, no model call — fake writes a sentinel file when invoked; assert it is absent).

```go
func TestDistillerIncrementalFreezesPrefix(t *testing.T) {
	dir := t.TempDir()
	bin := buildBinary(t, dir)
	fakeClaude(t, dir, "- M1 done — tests green\n") // canned tail (just Ek', no new)
	// ... write cleaned.txt (with the anchor lines present + new content),
	//     prior changelog.md, changelog.anchor ...
	runDistiller(t, bin, dir, "--today", "2026-06-12")
	got := readFile(t, logPath)
	if !strings.HasPrefix(got, frozenPrefix) {
		t.Fatalf("frozen prefix must be byte-identical; got %q", got)
	}
}
```

- [ ] **Step 3: Implement `main.go`** — flags `--cleaned --log --anchor --agent --today`; read files; `scanTurnBoundaries` → `locate`; on `NoOp` exit 0; build prompt+input; `model.Run` with a generous `MaxOutputTokens` (e.g. 2000); `splitFirstEntry(modelOut)` → `ekPrime`, `newEntries` (on first-run, all of `modelOut` is `newEntries`, `ekPrime=""`); `assemble`; write log (temp+rename) **then** anchor (temp+rename) — log-first crash-safety (spec). Atomic write helper mirrors the lessons.md temp+rename rule.

- [ ] **Step 4: Run to verify pass.** `go test ./cmd/pair-changelog/...` → PASS (all cases).

- [ ] **Step 5: Add the binary to the build.** Edit `Makefile.local` (the top-level `Makefile` is a symlink to ariadne's): (a) append `pair-changelog` to `GO_BINS` (~line 29) and (b) add the per-binary recipe + alias stanza mirroring `pair-slug` (~lines 41–45 and 139–140; the in-file comment at lines 10–12 documents the 2-step recipe). Confirm the pattern first with `grep -n pair-slug Makefile.local`. Run `make pair-changelog` → builds to `bin/`.

- [ ] **Step 6: Commit.** `git commit -m "#53 M1: pair-changelog distiller main + integration test"`

### M1 milestone close

- [ ] Run `go test ./...` (whole module green).
- [ ] `sdlc milestone-close --issue 53 --milestone M1` (auto-dispatches the fresh-context boundary review; fix Critical/Important before crossing; log the `Review-Verdict:` outcome).

---

## Chunk 2: M2 — orchestrator + viewer + keybind

### Task 5: `bin/pair-changelog-open` orchestrator

**Files:**
- Create: `bin/pair-changelog-open`

- [ ] **Step 1: Write the script** modeled on `bin/pair-scrollback-open` (lines 21–193): same env guard (`PAIR_DATA_DIR`/`PAIR_TAG`/`PAIR_AGENT`), same PID lock (`changelog-$PAIR_TAG-$PAIR_AGENT.openlock`, `kill -0` re-entrancy). Then: clean the TTY to a temp file via `"$PAIR_HOME/bin/pair-scrollback-render" --plain --max-lines 0 "$RAW" "$EVENTS" "$CLEANED"`; run `"$PAIR_HOME/bin/pair-changelog" --cleaned "$CLEANED" --log "$LOG" --anchor "$ANCHOR" --agent "$PAIR_AGENT" --today "$(date +%F)"`; then `nvim -u "$PAIR_HOME/nvim/changelog.lua" "$LOG"` (non-exec, so the EXIT trap clears the lock — matching scrollback). Handle the empty-`$RAW` and missing-binary cases with the same friendly messages. Create `$LOG` empty if absent so nvim always has a file.

- [ ] **Step 2: `chmod +x bin/pair-changelog-open`.**

- [ ] **Step 3: Smoke test the no-session guard.**

Run: `env -u PAIR_TAG bin/pair-changelog-open`
Expected: prints the "meant to run inside a pair session" message, exits non-zero.

- [ ] **Step 4: Commit.** `git commit -m "#53 M2: pair-changelog-open orchestrator"`

### Task 6: `nvim/changelog.lua` read-only viewer + colorization

**Files:**
- Create: `nvim/changelog.lua`
- Create: `nvim/changelog_test.lua`

- [ ] **Step 1: Write the headless test (RED).** Model on `nvim/slug_test.lua` / `scrollback_test.lua`: load a fixture log into a buffer, source the viewer's setup, assert `vim.bo.modifiable == false` and that a `#53` token resolves to the `ChangelogTicket` syntax group via `synIDattr(synID(l,c,1),'name')`.

- [ ] **Step 2: Run to verify failure.** `nvim -l nvim/changelog_test.lua` → non-zero.

- [ ] **Step 3: Implement `changelog.lua`.** Read-only/full-screen setup (`modifiable=false`, `readonly=true`, `buftype=nofile`, `number=false`, `signcolumn=no`, `fillchars eob=' '`); `<Esc>`→`qa!`; cursor to bottom (`normal! G`); colorization:

```lua
vim.cmd([[
  syntax match ChangelogTicket    /#\d\+/
  syntax match ChangelogMilestone /\<M\d\+\>/
  syntax match ChangelogCode      /`[^`]\+`/
  syntax match ChangelogBranch    /\<feature\/\S\+/
  highlight default link ChangelogTicket    Identifier
  highlight default link ChangelogMilestone Type
  highlight default link ChangelogCode      String
  highlight default link ChangelogBranch    Constant
]])
```

Structure the file so the test can source the setup without a real `nvim -u` launch (a `M.setup(bufnr)` function, called from a `VimEnter`/`BufReadPost` autocmd when run as `-u`, and directly from the test) — mirrors how `scrollback.lua` exposes testable pieces.

- [ ] **Step 4: Run to verify pass.** `nvim -l nvim/changelog_test.lua` → exit 0.

- [ ] **Step 5: Commit.** `git commit -m "#53 M2: changelog.lua read-only viewer + colorization"`

### Task 7: `Alt l` zellij keybind

**Files:**
- Modify: `zellij/config.kdl` (next to the `Alt /` binding)

- [ ] **Step 1: Add the binding**, mirroring the `Alt /` `Run` block (floating, `close_on_exit true`, full-size), running `pair-changelog-open`. Verify the exact shape against the `Alt /` block already in the file.

- [ ] **Step 2: Validate the config.** `zellij setup --check --config-dir zellij` → no errors (per lessons.md: validate KDL against the installed zellij).

- [ ] **Step 3: Commit.** `git commit -m "#53 M2: bind Alt+l to the change log viewer"`

### Task 8: End-to-end dogfood

- [ ] **Step 1: Live dogfood in a real pair session** (per the dogfood-live value). Start a session, drive a couple of turns, press `Alt+l`: verify the viewer opens read-only/full-screen with distilled entries and colorized tokens. Press `Alt+l` again after more activity: verify new entries append, earlier entries are unchanged, and a no-op press (no new activity) opens instantly without a model call. Capture the observed behavior in `## Log`.
- [ ] **Step 1b: Record the concurrency-safety basis** (judge INFO #2 — make the "verified" evidence explicit, not implicit). The "concurrent presses cannot corrupt the log" guarantee is satisfied by: (i) reusing `pair-scrollback-open`'s proven lifetime PID lock + `kill -0` re-entrancy (a second press while open is a no-op refocus, observed in dogfood), and (ii) the distiller's log-first-then-anchor **atomic temp+rename** ordering (a torn/partial file is never observable; a crash leaves the anchor one-behind → safe re-process). There is **no dedicated shell race test** (a true two-press race is impractical to unit-test in `sh`); state exactly this in the close `--verified` evidence so the gate isn't ambiguous.
- [ ] **Step 2: Update `atlas/`** for the new surface (the `Alt+l` change log: orchestrator + distiller + viewer + state files), and link it from `atlas/index.md`.
- [ ] **Step 3: Commit.** `git commit -m "#53 M2: dogfood + atlas"`

### M2 milestone close / issue close

- [ ] `go test ./...` + the Lua test green; KDL validated.
- [ ] `sdlc close --issue 53 --milestone M2 --verified '<evidence: test output + dogfood observations>'` (auto-dispatches the final fresh-context review; `--actual` measured, not typed).

---

## Notes carried from spec review (advisory)

1. **Inter-block whitespace** is pinned at the byte level by Task 3's `assemble` tests + Task 4's frozen-prefix assertion — the literal `## D\n\n-` spacing and the trailing newline of `Ek'` are nailed by `ensureBlock` + the golden strings.
2. **Header invariant** ("a `## date` is never written without ≥1 following bullet") is encoded in `assemble` and documented there, keeping `splitFrozenTail`'s "last bullet block" well-defined for every reachable state.
3. **Full-redistill** regenerates earlier entries (could reword them) — it trades the structural freeze for that one rare redraw-recovery pass. Only the `Found` happy path guarantees byte-stability; this is acceptable (redraw-mangled anchors are rare) and noted so a reviewer isn't surprised.

## Revisions

### 2026-06-12 — M1 boundary-review (FIX-THEN-SHIP) corrections

- **Note #3 above corrected.** The implemented `main.go` keeps the **frozen
  prefix even on full-redistill** (only `Ek` is revised + new appended; the whole
  transcript is fed as "new activity" with the prior log as read-only dedup
  memory). So byte-stability of the frozen prefix holds there too — the residual
  risk on full-redistill is **duplicate** entries (dedup rests on the prompt),
  not reworded earlier entries. Pinned by `TestFullRedistillWithPriorLogKeepsFrozenPrefix`.
- **Task 1 side-effect (now landed):** the extraction made `pair-slug` import
  `cmd/internal/model`, so its `Makefile.local` recipe prerequisite list gained
  `cmd/internal/model/model.go` (was missing → `make pair-slug` wouldn't rebuild
  on a model change). Caught by the M1 boundary review.
- Minor hardening from the same review: `model.Run` floors `MaxOutputTokens` at
  256 when ≤ 0; `splitLines` now trims all trailing newlines; package doc says
  "near-verbatim" (the codex temp prefix was renamed).

### 2026-06-12 — M2 dogfood UX rework (Tasks 5–6 revised)

Live `Alt+l` dogfood surfaced two issues; both fixed fix-forward before the M2
close:

- **No-op was brittle → turn-count.** The byte-flush no-op (`locate` "anchor
  flush with end") almost never fired in a live session because the anchor (last
  K cleaned lines) is the *volatile prompt/status area* → a model call on every
  press. The distiller now records the **completed-turn count** in the anchor
  (`turns:<N>` header, via `parseAnchor`/`writeAnchor`) and **skips the model
  unless the count increased**. `locate` is retained for the slice. Integration
  tests reworked to drive turns via `❯` boundaries; `TestNoOpWhenNoNewTurn`
  replaces the old byte-flush no-op test.
- **Blocking open → async + spinner.** `bin/pair-changelog-open` is now thin
  (lock + open nvim immediately on the existing log; export `PAIR_CHANGELOG_*`).
  `nvim/changelog.lua` runs render+distill as a background `jobstart`, animating
  a spinner as a bottom virtual line ("Computing change log…" first / "Refreshing
  change log (N new lines)…" after — N parsed from the distiller's `distilling N
  lines` stderr),
  and reloads the buffer on completion. The smoke test's fake nvim simulates the
  job from the exported env. So perceived latency is ~zero on every press;
  unchanged sessions clear the spinner near-instantly (no model call).

### 2026-06-12 — M2 boundary-review (FIX-THEN-SHIP) corrections

- **Graceful-degradation guarantee narrowed.** The §Lookback claim "a missed/false
  marker degrades gracefully — never breaks the boundary" held when turns drove
  *only* the lookback. The turn-count no-op now also gates the **model call**, so
  a dropped prompt glyph can *delay* a distill (or a spurious one trigger an extra
  model call). It still **self-heals within ~1 press** and never corrupts the log
  — but it's a delay/extra-call, not pure lookback-context jitter.
- **CI gaps closed** (both Important): `make test-changelog` now depends on the
  binaries so it runs in `make test` instead of silently SKIPping on a clean
  tree; and the async viewer path got a headless test (`changelog_test.lua` now
  exercises `M.reload`: content replaced, cursor at newest, buffer still readonly).
- **`parseAnchor` moved** from `main.go` to `distill.go` (with its pure siblings)
  and directly unit-tested (header / legacy-no-header / malformed-count / empty);
  added to the Core-concepts tables (`parseAnchor` PURE, `writeAnchor` INTEGRATION).
