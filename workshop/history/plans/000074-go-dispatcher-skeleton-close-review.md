# Boundary Review — ariadne#74 (whole-issue close)

| field | value |
|-------|-------|
| issue | 74 — pair Go dispatcher skeleton |
| repo | pair |
| issue file | workshop/issues/000074-go-dispatcher-skeleton.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1199bb427dfc0d7c7b87dcc1115ce19907c4d804..HEAD |
| command | sdlc close --issue 74 |
| reviewer | codex |
| timestamp | 2026-06-29T17:10:18-07:00 |
| verdict | SHIP |

## Review

Reading additional input from stdin...
OpenAI Codex v0.142.4
--------
workdir: /Users/xianxu/workspace/pair
model: gpt-5.5
provider: openai
approval: never
sandbox: workspace-write [workdir, /tmp, $TMPDIR, /tmp] (network access enabled)
reasoning effort: none
reasoning summaries: none
session id: 019f15db-6fd7-7583-80f5-21da2e59cd7f
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary — a
milestone close, or a whole-issue close — for ariadne#74.

Base: 1199bb427dfc0d7c7b87dcc1115ce19907c4d804   Head: HEAD

You have no prior session context — that is the anti-collusion property. Verify
behavior against the issue's documented Spec/Plan and the code itself; do NOT
take the implementor's word in commit messages or docs at face value. Tools are
read-only: report findings precisely; the main agent (which has session context)
applies the fixes, commits, and re-runs.

Read the diff against the issue's Spec + Plan, then work the checklist below.
Categorize every finding by severity — not everything is Critical; a nitpick
marked Critical is noise.

  Critical (must fix before crossing the boundary)
    - correctness bugs; crashes / panics on unexpected input
    - behavior drift from stated contracts (for ports of existing code where
      byte-faithfulness was promised, diff against the source)
    - silent error swallowing where the source raised
  Important (fix before the boundary if cheap)
    - API design of newly-introduced internal packages (downstream work will
      consume them; is the surface stable?)
    - missing test coverage that would catch the kind of bug shipped
    - inconsistent error handling across the diff
  Minor (note for future)
    - style nits, naming, comment density; performance only if hot-path

## Review checklist

Code quality
  - Clean separation of concerns; edge cases handled (empty / nil / unexpected).
  - Proper error handling — no silent swallowing where the source raised.
  - No duplicated logic / copy-paste that should be a shared helper.

Testing
  - Tests pin real logic, not mocks reasserting the implementation.
  - The kind of bug this diff could ship is covered.
  - PURE entities tested without IO; INTEGRATION via injected fakes (see below).

Requirements traceability
  - Every Plan checklist item this boundary claims is actually delivered.
  - Implementation matches the Spec; no undeclared scope creep.
  - Breaking changes documented.

Production readiness
  - Migration / backward-compatibility considered where state or formats change.
  - Docs / atlas updated for new surface (see the Atlas update gate).

## Core concepts cross-check (if the plan has a Core concepts table)

The plan should list entities in a greppable table — name, kind
(PURE/INTEGRATION), file location, status (new/modified/deleted). For each row:
  - Verify the entity exists at the stated path (grep the diff or filesystem).
  - PURE: tests run without IO (no exec, net, mutable fs). If tests need mocks
    to run, it isn't really PURE — flag Critical and recommend promoting it to
    INTEGRATION.
  - INTEGRATION: injected into pure callers, not invoked directly from business
    logic.
  - "modified" / "deleted": the diff shows the expected change/removal at the
    stated location.
Any contradiction between table and code = Critical finding, plus a plan-revision
recommendation (a "## Revisions" entry so the plan stops claiming what the code
doesn't deliver).

## Atlas update gate (per AGENTS.md §8)

The boundary should update atlas/ entries for any new architectural surface,
flow, or terminology introduced. Scan the diff for evidence of new surface — new
entity types, new subcommands, new conventions, new file-tree locations. Any
present without corresponding atlas/ changes in the same range = Important
finding ("atlas update appears missing for <surface>").

## Architecture (the at-review backstop — these matter most long-term)

Work through each of ARCH-DRY, ARCH-PURE, ARCH-PURPOSE explicitly, applying its at-review lens. The
full principle definitions are delivered in the ARCHITECTURE PRINCIPLES block
right after this prompt — for EACH marker, state pass or flag, and cite the
marker (e.g. ARCH-DRY) in any finding. Architecture is where review has the
least training signal and the longest-delayed payoff, so be deliberate here, not
holistic.

## Verdict + output

Your FIRST line is the VERDICT (format in the output contract below). Tokens:
  SHIP          = ready; ship it.
  FIX-THEN-SHIP = ship after addressing the findings (non-blocking at the gate;
                  fix them before the next boundary).
  REWORK        = blocking; needs rework before shipping.

After the VERDICT line: a 1-paragraph summary — what worked, what blocks SHIP if
it isn't — followed by:
  1. Strengths: 2-5 specific things done well (file:line where useful). Affirm
     validated approaches so the operator knows what's confirmed-good ground.
     Empty acceptable for trivial boundaries.
  2. Critical findings (file:line + fix sketch); empty if none.
  3. Important findings (same format).
  4. Minor findings (terse one-liners).
  5. Test coverage notes.
  6. Architectural notes for upcoming work.
  7. Plan revision recommendations: specific "## Revisions" entries the plan
     needs (empty if the plan still matches the code).


ARCHITECTURE PRINCIPLES — work through each of the 3 entries below explicitly, applying its `at-review` lens; cite the marker (e.g. ARCH-DRY) in any finding.

# Architecture principles (ARCH-*)

Injected architectural taste — the structural decisions whose payoff (or cost)
shows up many turns, often months, down the road. Agents are strong at local
tactics and weak here, so these are checked **at-plan** (when the design is being
made — highest leverage) and **at-review** (backstop, on the diff). Cite the
marker (e.g. `ARCH-DRY`) in plans, `## Log` entries, and review findings.

This file is the single source; it is embedded into the planning, plan-quality,
and code-review prompts. The human narrative lives in AGENTS.md "Core Design
Principles"; this is its machine-delivered companion.

## ARCH-DRY — Don't Repeat Yourself

- **principle:** Reuse before adding. One source of truth per fact/behavior; no
  duplicated logic, copy-pasted blocks, or parallel functions that should be one
  shared helper.
- **at-plan:** Flag a plan that re-implements something the codebase already has,
  or that will obviously duplicate logic across the new files instead of
  extracting a shared helper. Name the existing thing it should reuse.
- **at-review:** Flag duplicated logic / copy-pasted blocks / near-identical
  functions in the diff; point at the consolidation (file:line + the shared
  helper they should become).

## ARCH-PURE — Pure core, thin IO shell

- **principle:** The majority of code is pure functions (deterministic, no side
  effects); a thin "glue" layer at the boundary touches IO/UI/network/clock. Pure
  functions are unit-tested directly; the glue is kept small and injected.
- **at-plan:** Flag a design that buries business logic inside IO/handlers, or
  that will only be testable with heavy mocks (a sign logic isn't separated from
  IO). The plan should name what's pure vs the thin IO seam.
- **at-review:** Flag business logic mixed with IO in the diff; logic that should
  be a pure function injected into a thin caller. If a test needs mocks to run a
  "pure" entity, it isn't pure — recommend extracting the IO to the boundary.

## ARCH-PURPOSE — Serve the issue's actual purpose

- **principle:** Deliver the issue's stated purpose, not the easy subset of it. A
  single-source / "compiled to consumers" change is not done until **every
  consumer derives** from the source — the source is *enforced*, not just
  documentation a surface happens to restate; a hand-maintained restatement of the
  model is a deferred consumer, not a finished one. "Follow-up" is for separable
  extensions, never for the thing that is the point. This is the *opposite axis*
  from Simplicity-First/YAGNI: not "build for an imagined future," but "don't
  **under**-deliver the purpose you already committed to."
- **at-plan:** Flag a plan whose scope is a strict subset of the issue's stated
  goal / Done-when where the part deferred as "follow-up" *is* the purpose (e.g.
  wires one consumer + enforcement but leaves the consumers that motivated the
  issue as documentation that doesn't derive). Ask: does the plan fulfill the
  purpose, or just the cheap win? Name the deferred purpose.
- **at-review:** Does the diff *fulfill* the purpose or settle for the easy win?
  For a single-source change, run the **shadow-sweep** — enumerate the consumers,
  confirm each derives from the source, flag any remaining hand-maintained
  restatement of the model. A "follow-up" that is actually the deferred point of
  the issue is a finding, not a deferral.


OUTPUT CONTRACT (machine-read — do not deviate). Your response's FIRST line
MUST be exactly:

    VERDICT: <TOKEN> (confidence: high | medium | low)

The parser reads ONLY this <TOKEN>. Findings, notes, and severity tags below it
are advisory — a non-blocking verdict WITH notes still PASSES the gate. Do not
put a title, heading, or any preamble above the VERDICT line; it must lead.

Diff:
diff --git a/Makefile.local b/Makefile.local
index c6ca50c..0618d25 100644
--- a/Makefile.local
+++ b/Makefile.local
@@ -1,7 +1,7 @@
 # Repo-specific Makefile targets.
 # This file is included by Makefile — add your own targets here.
 
-.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
+.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
 
 # Cross-cutting Go-binary targets. Every Go binary lives under cmd/<name>/
 # and ships with the `pair-` prefix so it doesn't collide with anything
@@ -26,7 +26,7 @@
 
 BIN_DIR     := bin
 INSTALL_DIR := $(HOME)/.local/bin
-GO_BINS     := pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context
+GO_BINS     := pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-go
 
 build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
 
@@ -45,6 +45,7 @@ pair-slug:              $(BIN_DIR)/pair-slug
 pair-continuation:      $(BIN_DIR)/pair-continuation
 pair-changelog:         $(BIN_DIR)/pair-changelog
 pair-context:           $(BIN_DIR)/pair-context
+pair-go:                $(BIN_DIR)/pair-go
 
 # Tests
 #   make test       → go test ./... (cache disabled so reruns actually run)
@@ -238,6 +239,11 @@ $(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill
 $(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-continuation/git.go cmd/pair-continuation/main.go go.mod
 	go build -o $@ ./cmd/pair-continuation
 
+# pair-go — development-only dispatcher skeleton for the future primary Go CLI.
+# The public launcher remains bin/pair until the entrypoint-switch issue.
+$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/dispatcher/dispatcher.go go.mod
+	go build -o $@ ./cmd/pair-go
+
 
 # Tart VM targets (tart, tart-gui, tart-mount, tart-stop, tart-clean)
 # live in .tart/Makefile, vendored from ariadne and auto-included by
diff --git a/atlas/architecture.md b/atlas/architecture.md
index c761760..c55433e 100644
--- a/atlas/architecture.md
+++ b/atlas/architecture.md
@@ -10,6 +10,7 @@ The whole thing is deliberately small — a handful of shell scripts, one nvim i
 
 ```
 bin/pair                     # entry point (launcher)
+bin/pair-go                  # opt-in Go dispatcher skeleton; public launcher remains bin/pair
 bin/clipboard-to-pane.sh     # read clipboard, hand off to nvim's PairPasteQuote
 bin/copy-on-select.sh        # invoked by zellij copy_command on mouse-up
 bin/pair-quit.sh             # invoked by Alt+x — marks + kills session
@@ -40,6 +41,11 @@ internal subcommands or dispatch modes behind that primary binary (`pair wrap`,
 `pair continuation`, `pair scribe`) instead of staying as independently managed
 installed commands forever.
 
+As of #74, `bin/pair-go` is a development-only dispatcher skeleton built from
+`cmd/pair-go` and the pure `cmd/internal/dispatcher` parser. It lists the
+planned command families and returns explicit unsupported-command errors, but it
+does not launch sessions or replace `bin/pair`.
+
 Native integration layers stay native: `nvim/*.lua` remains the bundled Neovim
 surface and `zellij/*.kdl` remains the zellij layout/config surface. Packaging
 may embed those assets or install them adjacent to the binary, but the migration
diff --git a/cmd/internal/dispatcher/dispatcher.go b/cmd/internal/dispatcher/dispatcher.go
new file mode 100644
index 0000000..dddfe46
--- /dev/null
+++ b/cmd/internal/dispatcher/dispatcher.go
@@ -0,0 +1,89 @@
+package dispatcher
+
+import (
+	"fmt"
+	"strings"
+)
+
+const programName = "pair-go"
+
+// CommandFamily names a future Pair CLI surface without claiming it works yet.
+type CommandFamily struct {
+	Name    string
+	Summary string
+	Status  string
+}
+
+// Result is the process-facing outcome of a pure dispatch decision.
+type Result struct {
+	Stdout   string
+	Stderr   string
+	ExitCode int
+}
+
+// Families returns the planned command families for the Go dispatcher.
+func Families() []CommandFamily {
+	return []CommandFamily{
+		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "planned"},
+		{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
+		{Name: "slug", Summary: "session orientation slug generation", Status: "planned"},
+		{Name: "context", Summary: "agent pane context meter", Status: "planned"},
+		{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "planned"},
+		{Name: "changelog", Summary: "TTY transcript to distilled change log", Status: "planned"},
+		{Name: "continuation", Summary: "continuation datatype writer", Status: "planned"},
+		{Name: "scribe", Summary: "PTY logging wrapper", Status: "planned"},
+	}
+}
+
+// Dispatch parses argv and returns the skeleton dispatch result.
+func Dispatch(args []string) Result {
+	if len(args) == 0 {
+		return Result{Stdout: Help(programName), ExitCode: 0}
+	}
+
+	switch args[0] {
+	case "help", "--help", "-h":
+		return Result{Stdout: Help(programName), ExitCode: 0}
+	case "version", "--version":
+		return Result{
+			Stdout:   "pair-go dispatcher skeleton\npublic launcher: bin/pair\n",
+			ExitCode: 0,
+		}
+	}
+
+	if family, ok := familyByName(args[0]); ok {
+		return Result{
+			Stderr:   fmt.Sprintf("%s: %s is planned but not implemented in this skeleton; run %s help\n", programName, family.Name, programName),
+			ExitCode: 2,
+		}
+	}
+
+	return Result{
+		Stderr:   fmt.Sprintf("%s: unknown command %q; run %s help\n", programName, args[0], programName),
+		ExitCode: 2,
+	}
+}
+
+// Help renders the development-only dispatcher usage text.
+func Help(program string) string {
+	var b strings.Builder
+	fmt.Fprintf(&b, "Usage: %s <command> [args]\n\n", program)
+	b.WriteString("Development dispatcher skeleton. Public sessions still start through bin/pair.\n\n")
+	b.WriteString("Planned command families (not implemented in this skeleton):\n")
+	for _, family := range Families() {
+		fmt.Fprintf(&b, "  %-17s %s (%s; not implemented in this skeleton)\n", family.Name, family.Summary, family.Status)
+	}
+	b.WriteString("\nSupported skeleton commands:\n")
+	b.WriteString("  help              show this help\n")
+	b.WriteString("  version           show dispatcher skeleton metadata\n")
+	return b.String()
+}
+
+func familyByName(name string) (CommandFamily, bool) {
+	for _, family := range Families() {
+		if family.Name == name {
+			return family, true
+		}
+	}
+	return CommandFamily{}, false
+}
diff --git a/cmd/internal/dispatcher/dispatcher_test.go b/cmd/internal/dispatcher/dispatcher_test.go
new file mode 100644
index 0000000..c9fed26
--- /dev/null
+++ b/cmd/internal/dispatcher/dispatcher_test.go
@@ -0,0 +1,76 @@
+package dispatcher
+
+import (
+	"strings"
+	"testing"
+)
+
+func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
+	for _, args := range [][]string{nil, {"help"}, {"--help"}, {"-h"}} {
+		t.Run(strings.Join(args, "_"), func(t *testing.T) {
+			res := Dispatch(args)
+			if res.ExitCode != 0 {
+				t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
+			}
+			if res.Stderr != "" {
+				t.Fatalf("Stderr = %q, want empty", res.Stderr)
+			}
+			for _, want := range []string{
+				"Usage: pair-go <command> [args]",
+				"launch",
+				"wrap",
+				"slug",
+				"not implemented in this skeleton",
+			} {
+				if !strings.Contains(res.Stdout, want) {
+					t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
+				}
+			}
+		})
+	}
+}
+
+func TestDispatchVersionIsDevelopmentSkeletonMetadata(t *testing.T) {
+	res := Dispatch([]string{"version"})
+	if res.ExitCode != 0 {
+		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
+	}
+	if res.Stderr != "" {
+		t.Fatalf("Stderr = %q, want empty", res.Stderr)
+	}
+	for _, want := range []string{"pair-go", "dispatcher skeleton", "public launcher: bin/pair"} {
+		if !strings.Contains(res.Stdout, want) {
+			t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
+		}
+	}
+}
+
+func TestDispatchPlannedCommandReturnsUnsupported(t *testing.T) {
+	res := Dispatch([]string{"wrap"})
+	if res.ExitCode != 2 {
+		t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
+	}
+	if res.Stdout != "" {
+		t.Fatalf("Stdout = %q, want empty", res.Stdout)
+	}
+	for _, want := range []string{"wrap", "planned", "not implemented", "pair-go help"} {
+		if !strings.Contains(res.Stderr, want) {
+			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
+		}
+	}
+}
+
+func TestDispatchUnknownCommandReturnsUsageHint(t *testing.T) {
+	res := Dispatch([]string{"frobnicate"})
+	if res.ExitCode != 2 {
+		t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
+	}
+	if res.Stdout != "" {
+		t.Fatalf("Stdout = %q, want empty", res.Stdout)
+	}
+	for _, want := range []string{"unknown command", "frobnicate", "pair-go help"} {
+		if !strings.Contains(res.Stderr, want) {
+			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
+		}
+	}
+}
diff --git a/cmd/pair-go/main.go b/cmd/pair-go/main.go
new file mode 100644
index 0000000..29cafe0
--- /dev/null
+++ b/cmd/pair-go/main.go
@@ -0,0 +1,25 @@
+// pair-go is a development-only dispatcher skeleton for the future primary Go
+// CLI. The public launcher remains bin/pair.
+package main
+
+import (
+	"io"
+	"os"
+
+	"github.com/xianxu/pair/cmd/internal/dispatcher"
+)
+
+func main() {
+	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
+}
+
+func run(args []string, stdout, stderr io.Writer) int {
+	res := dispatcher.Dispatch(args)
+	if res.Stdout != "" {
+		_, _ = io.WriteString(stdout, res.Stdout)
+	}
+	if res.Stderr != "" {
+		_, _ = io.WriteString(stderr, res.Stderr)
+	}
+	return res.ExitCode
+}
diff --git a/cmd/pair-go/main_test.go b/cmd/pair-go/main_test.go
new file mode 100644
index 0000000..1563f04
--- /dev/null
+++ b/cmd/pair-go/main_test.go
@@ -0,0 +1,35 @@
+package main
+
+import (
+	"bytes"
+	"strings"
+	"testing"
+)
+
+func TestRunWritesStdoutAndReturnsDispatcherCode(t *testing.T) {
+	var stdout, stderr bytes.Buffer
+	code := run([]string{"help"}, &stdout, &stderr)
+	if code != 0 {
+		t.Fatalf("code = %d, want 0", code)
+	}
+	if !strings.Contains(stdout.String(), "Usage: pair-go <command> [args]") {
+		t.Fatalf("stdout missing usage:\n%s", stdout.String())
+	}
+	if stderr.String() != "" {
+		t.Fatalf("stderr = %q, want empty", stderr.String())
+	}
+}
+
+func TestRunWritesStderrAndReturnsDispatcherCode(t *testing.T) {
+	var stdout, stderr bytes.Buffer
+	code := run([]string{"wrap"}, &stdout, &stderr)
+	if code != 2 {
+		t.Fatalf("code = %d, want 2", code)
+	}
+	if stdout.String() != "" {
+		t.Fatalf("stdout = %q, want empty", stdout.String())
+	}
+	if !strings.Contains(stderr.String(), "wrap is planned but not implemented") {
+		t.Fatalf("stderr missing unsupported-command message:\n%s", stderr.String())
+	}
+}
diff --git a/workshop/plans/000074-go-dispatcher-skeleton-plan.md b/workshop/plans/000074-go-dispatcher-skeleton-plan.md
new file mode 100644
index 0000000..ad88560
--- /dev/null
+++ b/workshop/plans/000074-go-dispatcher-skeleton-plan.md
@@ -0,0 +1,153 @@
+# Pair Go Dispatcher Skeleton Implementation Plan
+
+> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.
+
+**Goal:** Add an opt-in Go dispatcher skeleton that establishes Pair's future primary CLI shape without changing the public `bin/pair` launcher.
+
+**Architecture:** Create a pure `cmd/internal/dispatcher` package for argv parsing, help rendering, planned command metadata, and unsupported-command errors (`ARCH-PURE`). Add `cmd/pair-go/main.go` as a thin IO shell that calls the dispatcher and writes output/exit codes. Wire `pair-go` into the existing `GO_BINS` build pattern while leaving `bin/pair`, zellij layouts, and helper command names unchanged (`ARCH-DRY`, `ARCH-PURPOSE`).
+
+**Tech Stack:** Go 1.26, standard library only, existing `Makefile.local` build conventions.
+
+---
+
+## Core Concepts
+
+### Pure Entities
+
+| Name | Lives in | Status |
+|------|----------|--------|
+| `CommandFamily` | `cmd/internal/dispatcher/dispatcher.go` | new |
+| `Result` | `cmd/internal/dispatcher/dispatcher.go` | new |
+| `Dispatch` | `cmd/internal/dispatcher/dispatcher.go` | new |
+| `Help` | `cmd/internal/dispatcher/dispatcher.go` | new |
+
+- **CommandFamily** — metadata for planned Pair command families that the skeleton can list without implementing.
+  - **Relationships:** N:1 with `Help`; `Help` renders all families from one slice.
+  - **DRY rationale:** One source for names/descriptions/status text instead of duplicating planned command lists between help and dispatch validation.
+  - **Future extensions:** Add handler wiring when later issues port a family.
+
+- **Result** — pure dispatch outcome: stdout, stderr, and process exit code.
+  - **Relationships:** 1:1 with a dispatcher invocation; consumed by the IO shell.
+  - **DRY rationale:** Keeps exit behavior testable without spawning a subprocess.
+  - **Future extensions:** Can grow a handler enum or execution callback once subcommands become real.
+
+- **Dispatch** — pure argv parser and router for `help`, `--help`, `version`, and unsupported commands.
+  - **Relationships:** Owns one invocation's decision; reads `CommandFamily` metadata; returns `Result`.
+  - **DRY rationale:** Centralizes command parsing before behavior begins migrating out of shell scripts.
+  - **Future extensions:** Replace unsupported-command outcomes with real handlers one family at a time.
+
+- **Help** — pure renderer for the dev CLI usage and planned command families.
+  - **Relationships:** Reads `CommandFamily`; used by `Dispatch`.
+  - **DRY rationale:** Help text and tests derive from the same command-family metadata.
+  - **Future extensions:** Add richer command-specific help when handlers exist.
+
+### Integration Points
+
+| Name | Lives in | Status | Wraps |
+|------|----------|--------|-------|
+| `pair-go main` | `cmd/pair-go/main.go` | new | `os.Args`, stdout/stderr, process exit |
+| `pair-go build target` | `Makefile.local` | modified | `go build` into `bin/pair-go` |
+| `architecture note` | `atlas/architecture.md` | modified | repo map documentation |
+
+- **pair-go main** — minimal binary entrypoint: call `dispatcher.Dispatch(os.Args[1:])`, print streams, exit.
+  - **Injected into:** No pure entity; it consumes the pure `Result`.
+  - **Future extensions:** Later issues can replace result-only routing with handler execution behind the same package.
+
+- **pair-go build target** — follow the existing `GO_BINS` and per-binary recipe convention so `make build` compiles the skeleton.
+  - **Injected into:** Build/install flow only.
+  - **Future extensions:** The public `pair` entrypoint switch happens in a later issue, not here.
+
+- **architecture note** — record that `pair-go` is the opt-in dispatcher skeleton and not the public launcher.
+  - **Injected into:** Atlas readers and future migration issues.
+  - **Future extensions:** Update when the public entrypoint changes.
+
+## Chunk 1: Dispatcher Skeleton
+
+### Task 1: Add pure dispatch tests
+
+**Files:**
+- Create: `cmd/internal/dispatcher/dispatcher_test.go`
+- Later create: `cmd/internal/dispatcher/dispatcher.go`
+
+- [x] **Step 1: Write failing tests for help, version, and unsupported commands**
+
+Cover:
+- empty argv and `help` return exit `0` with usage and planned families.
+- `--help` and `-h` mirror help.
+- `version` returns deterministic dev metadata without pretending to be the shell launcher.
+- a planned-but-unimplemented command like `wrap` returns exit `2` and an unsupported message.
+- an unknown command returns exit `2` and suggests `pair-go help`.
+
+- [x] **Step 2: Run the package test and verify it fails**
+
+Run: `go test ./cmd/internal/dispatcher -count=1`
+
+Expected: fail because `dispatcher.go` does not exist yet.
+
+- [x] **Step 3: Implement the minimal dispatcher**
+
+Create:
+- `CommandFamily` with `Name`, `Summary`, and `Status`.
+- `Families()` returning planned command families: `launch`, `wrap`, `slug`, `context`, `scrollback-render`, `changelog`, `continuation`, `scribe`.
+- `Result` with `Stdout`, `Stderr`, `ExitCode`.
+- `Dispatch(args []string) Result`.
+- `Help(program string) string`.
+
+- [x] **Step 4: Run dispatcher tests and verify they pass**
+
+Run: `go test ./cmd/internal/dispatcher -count=1`
+
+Expected: pass.
+
+### Task 2: Add the opt-in binary wrapper and build target
+
+**Files:**
+- Create: `cmd/pair-go/main.go`
+- Modify: `Makefile.local`
+
+- [x] **Step 1: Write a failing wrapper smoke test if needed**
+
+If the pure tests cover all behavior, keep `main.go` untested directly and rely on `go test ./...` plus build verification. Do not introduce subprocess tests unless pure coverage misses behavior.
+
+- [x] **Step 2: Implement `cmd/pair-go/main.go`**
+
+Main reads `os.Args[1:]`, delegates to `dispatcher.Dispatch`, writes stdout/stderr to the matching streams, and exits with the returned code.
+
+- [x] **Step 3: Wire `pair-go` into `Makefile.local`**
+
+Append `pair-go` to `GO_BINS`, add the `.PHONY` alias, and add:
+
+```make
+$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/dispatcher/dispatcher.go go.mod
+	go build -o $@ ./cmd/pair-go
+```
+
+- [x] **Step 4: Build the new binary**
+
+Run: `make pair-go`
+
+Expected: `bin/pair-go` is created.
+
+### Task 3: Document and verify non-disruption
+
+**Files:**
+- Modify: `atlas/architecture.md`
+- Modify: `workshop/issues/000074-go-dispatcher-skeleton.md`
+
+- [x] **Step 1: Update atlas**
+
+Add `bin/pair-go` to the piece list as the opt-in Go dispatcher skeleton and mention it in the packaging migration target as development-only for this issue.
+
+- [x] **Step 2: Run focused verification**
+
+Run:
+- `go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
+- `make pair-go`
+- `go test ./... -count=1`
+- `bash tests/pair-continue-test.sh`
+
+Expected: all pass. The shell launcher remains the public entrypoint.
+
+- [x] **Step 3: Update issue checkboxes and log**
+
+Mark completed plan and done-when boxes, then close through `sdlc close --issue 74 --verified '<evidence>'` with atlas evidence. This plan expects an atlas update, so prefer satisfying the atlas gate.


codex
I’ll use `superpowers-using-superpowers` first because the repo instructions require skill handling before task work, then I’ll inspect the issue/plan and changed files against the provided diff.
exec
/bin/zsh -lc "sed -n '1,220p' /Users/xianxu/workspace/ariadne/construct/adapted/superpowers-using-superpowers/SKILL.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
---
name: superpowers-using-superpowers
description: Use when starting any conversation - establishes how to find and use skills, requiring Skill tool invocation before ANY response including clarifying questions
---

<SUBAGENT-STOP>
If you were dispatched as a subagent to execute a specific task, skip this skill.
</SUBAGENT-STOP>

<EXTREMELY-IMPORTANT>
If you think there is even a 1% chance a skill might apply to what you are doing, you ABSOLUTELY MUST invoke the skill.

IF A SKILL APPLIES TO YOUR TASK, YOU DO NOT HAVE A CHOICE. YOU MUST USE IT.

This is not negotiable. This is not optional. You cannot rationalize your way out of this.
</EXTREMELY-IMPORTANT>

## Instruction Priority

> **Ariadne note:** AGENTS.md Section 3 governs subagent strategy and overrides skills that mandate subagent-driven-development as the default execution path.

Superpowers skills override default system prompt behavior, but **user instructions always take precedence**:

1. **User's explicit instructions** (CLAUDE.md, GEMINI.md, AGENTS.md, direct requests) — highest priority
2. **Superpowers skills** — override default system behavior where they conflict
3. **Default system prompt** — lowest priority

If CLAUDE.md, GEMINI.md, or AGENTS.md says "don't use TDD" and a skill says "always use TDD," follow the user's instructions. The user is in control.

## How to Access Skills

**In Claude Code:** Use the `Skill` tool. When you invoke a skill, its content is loaded and presented to you—follow it directly. Never use the Read tool on skill files.

**In Gemini CLI:** Skills activate via the `activate_skill` tool. Gemini loads skill metadata at session start and activates the full content on demand.

**In other environments:** Check your platform's documentation for how skills are loaded.

## Platform Adaptation

Skills use Claude Code tool names. Non-CC platforms: see `references/codex-tools.md` (Codex) for tool equivalents. Gemini CLI users get the tool mapping loaded automatically via GEMINI.md.

# Using Skills

## The Rule

**Invoke relevant or requested skills BEFORE any response or action.** Even a 1% chance a skill might apply means that you should invoke the skill to check. If an invoked skill turns out to be wrong for the situation, you don't need to use it.

```dot
digraph skill_flow {
    "User message received" [shape=doublecircle];
    "About to EnterPlanMode?" [shape=doublecircle];
    "Already brainstormed?" [shape=diamond];
    "Invoke brainstorming skill" [shape=box];
    "Might any skill apply?" [shape=diamond];
    "Invoke Skill tool" [shape=box];
    "Announce: 'Using [skill] to [purpose]'" [shape=box];
    "Has checklist?" [shape=diamond];
    "Create TodoWrite todo per item" [shape=box];
    "Follow skill exactly" [shape=box];
    "Respond (including clarifications)" [shape=doublecircle];

    "About to EnterPlanMode?" -> "Already brainstormed?";
    "Already brainstormed?" -> "Invoke brainstorming skill" [label="no"];
    "Already brainstormed?" -> "Might any skill apply?" [label="yes"];
    "Invoke brainstorming skill" -> "Might any skill apply?";

    "User message received" -> "Might any skill apply?";
    "Might any skill apply?" -> "Invoke Skill tool" [label="yes, even 1%"];
    "Might any skill apply?" -> "Respond (including clarifications)" [label="definitely not"];
    "Invoke Skill tool" -> "Announce: 'Using [skill] to [purpose]'";
    "Announce: 'Using [skill] to [purpose]'" -> "Has checklist?";
    "Has checklist?" -> "Create TodoWrite todo per item" [label="yes"];
    "Has checklist?" -> "Follow skill exactly" [label="no"];
    "Create TodoWrite todo per item" -> "Follow skill exactly";
}
```

## Red Flags

These thoughts mean STOP—you're rationalizing:

| Thought | Reality |
|---------|---------|
| "This is just a simple question" | Questions are tasks. Check for skills. |
| "I need more context first" | Skill check comes BEFORE clarifying questions. |
| "Let me explore the codebase first" | Skills tell you HOW to explore. Check first. |
| "I can check git/files quickly" | Files lack conversation context. Check for skills. |
| "Let me gather information first" | Skills tell you HOW to gather information. |
| "This doesn't need a formal skill" | If a skill exists, use it. |
| "I remember this skill" | Skills evolve. Read current version. |
| "This doesn't count as a task" | Action = task. Check for skills. |
| "The skill is overkill" | Simple things become complex. Use it. |
| "I'll just do this one thing first" | Check BEFORE doing anything. |
| "This feels productive" | Undisciplined action wastes time. Skills prevent this. |
| "I know what that means" | Knowing the concept ≠ using the skill. Invoke it. |

## Skill Priority

When multiple skills could apply, use this order:

1. **Process skills first** (brainstorming, debugging) - these determine HOW to approach the task
2. **Implementation skills second** (frontend-design, mcp-builder) - these guide execution

"Let's build X" → brainstorming first, then implementation skills.
"Fix this bug" → debugging first, then domain-specific skills.

## Skill Types

**Rigid** (TDD, debugging): Follow exactly. Don't adapt away discipline.

**Flexible** (patterns): Adapt principles to context.

The skill itself tells you which.

## User Instructions

Instructions say WHAT, not HOW. "Add X" or "Fix Y" doesn't mean skip workflows.

exec
/bin/zsh -lc 'sdlc --help' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
sdlc collects ariadne's SDLC checkpoint guards into one binary. Each subcommand
owns one checkpoint: it requires evidence at the gate, mutates state, logs the
transition, and refuses transitions that lack it. We don't model the SDLC as a
state machine — stages stay prose; we codify the gates between them where drift
recurs. `sdlc` manages the development life cycle; prefer it over `git`/`gh`.

BEFORE WORK
  - `sdlc claim --issue N` — the single start-of-work gesture, a CHEAP LOCK.
    Flips an *open* issue to `working` and publishes the claim to origin/main so
    peer agents see it. No estimate demanded (#113) — claim early, the moment an
    idea crystallizes. `--no-start` suppresses the flip.
  - Do NOT hand-edit an issue's `status:` — let `sdlc claim` or `sdlc issue
    set-status` own that transition (it carries the reopen/`→ done` guards).

ENTER IMPLEMENTATION
  - After plan approval, before editing code, run `sdlc change-code`. It owns the
    branching decision (in-place branch by default; `--worktree=yes` for an
    isolated worktree), the plan-quality check, and the `estimate_hours` gate
    (relocated here from claim, #113). Don't start coding without it.

PUBLISH
  - Publishing goes through a PR: `sdlc pr` → `sdlc merge`. Direct `sdlc push`
    if working directly on main.

RECOVER
  - After a compaction or session resume, run `sdlc state` to recover where you
    are instead of re-inferring from issue files.

LOCAL REPO TRANSACTION LOCK
  - Mutating verbs take an SDLC-owned repo transaction lock at
    `.git/sdlc.lock` before reading/writing issue state, committing, changing
    branches, or pushing. The lock is local to the Git common dir, so linked
    worktrees of the same repo serialize with each other.
  - Wait messages identify the holder pid and command when metadata is
    available. `change-code`, `close`, `milestone-close`, `merge`, and `push`
    can hold the lock during long-running review/ship transactions; wait or
    retry rather than removing the lock while that process is alive.
  - A dead same-host holder is reclaimed automatically; initializing metadata
    is waited through. Other stale/timeout errors tell you how to inspect
    `.git/sdlc.lock`. Remote push/ref races are separate: the local lock
    serializes this checkout, not another machine or clone.

WHEN A VERB ERRORS
  Do NOT route around it with hand-rolled `git`/`gh`. Its errors are next-action
  specs. The fix is one of two things:
    (a) satisfy the precondition it names and re-run the same verb (e.g. `sdlc
        merge` saying "no upstream" → run `sdlc pr` first, then `sdlc merge`); or
    (b) if the error is a genuine gap in `sdlc` itself, fix that edge case in the
        source and re-run. We're still ironing out edge cases.
  Only drop to manual when a verb genuinely cannot express the need — say so.

These gates sit inside a wider prose arc the binary does NOT own: ideation
(parley/pensive) → brainstorm → plan → build → milestone review (`sdlc judge`,
auto-dispatched) → close/ship → postmortem.

CONVENTIONS

  --issue vs --github-issue — `--issue N` always means workshop/issues
  (6-digit ID). `--github-issue N` means a GitHub issue number. Bare `--issue`
  never means a GitHub issue.

  Form vs essence — checkpoint guards (close, milestone-close, push, merge)
  defend against *omission* via required-evidence flags; `sdlc judge` defends
  against *theater* via fresh-context review. Form runs first; judge second.

The verb list + per-verb help (`sdlc <verb> --help`) follow below.

Usage:
  sdlc [flags]
  sdlc [command]

Available Commands:
  claim           Start work: flip an open issue to working + broadcast the claim
  start-plan      Enter planning: deliver the architecture principles to design against (#75)
  change-code     Enter implementation after the structural + plan-quality gates
  issue           Create + manage issues (new / set-status / list / show)
  actual          Compute an issue's focused dev-hours via active-time-v3 (#68)
  active-time     Per-issue active-time attribution table (the v3 engine, standalone)
  close           Close an issue or milestone (ACTUAL + VERIFIED + atlas/project sweep)
  milestone-close Close one milestone + auto-dispatch its review
  pr              Open a pull request from a feature branch
  merge           Merge the PR, archive done issues, clean up
  push            Ship from main (clean tree + pre-merge judges + archive)
  state           Inspect workflow state (branch, working issues, drift)
  judge           Run an LLM-judge check against the diff (fresh-context)
  arch-principles Print the ARCH-* architecture principles (single source; pull for non-gate work)
  estimate-source Name the shared estimate method + the repo-local calibration source (pull)
  propagate-base  Re-weave every recursive dependent of this repo (foundation-first)
  help            Help about any command

Flags:
  -h, --help   help for sdlc

Use "sdlc [command] --help" for more information about a command.

exec
/bin/zsh -lc "sed -n '1,220p' workshop/lessons.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
# Lessons

## Lua patterns: `\0` is empty-position match, not NUL byte

The unescape function in `nvim/scrollback.lua` first attempt used a
placeholder dance: `s:gsub('\\\\', '\0')` to swap `\\` for NUL,
then `gsub('\\(.)', '%1')` to strip remaining `\X`, then
`gsub('\0', '\\')` to restore the NUL → `\`. The result was
absurd: `unescape("plain")` returned `\p\l\a\i\n\` — the NUL pattern
matches between every byte (empty-position match), not the NUL byte
character. Each "match" inserted a `\` between every char.

**Rule.** When you need to match a literal NUL byte in a Lua pattern,
use `%z` or wrap as a character class `[%z]`. But the cleaner answer
is usually to skip patterns entirely for character-by-character
walks: a tiny while-loop with `s:sub(i, i)` is unambiguous and avoids
all the pattern-syntax footguns. Caught in #000018 review.

## Escape on insert, scan-with-parity on extract — for delimited markers

When user-supplied content is embedded in a delimited container
(e.g. `🤖<X>[Y]`), and X or Y can contain the delimiter chars,
the choice is "escape at insert + unescape at extract" vs "find
the closing delimiter cleverly." The first attempt at `🤖<X>[Y]`
parsing tried the latter — find first `>`, peek for `[`, give up
otherwise. Result: any selection with `>` was silently dropped on
extract, since the user couldn't tell the marker had been written
malformed.

**Rule.** Escape the delimiter chars in user-supplied fields at
insert time; have the parser walk byte-by-byte counting backslash
parity to find the *next unescaped* delimiter; unescape the
extracted content. The escape→walk→unescape chain handles every
delimiter-collision case uniformly, including `\\>` (literal `\`
followed by `>`). Don't try to be clever with "find first `>[`
adjacent" patterns — they fail when X contains `>[` literally,
and the failure mode is silent data loss. Caught in #000018 review.

## Shared delimiter codecs beat subsystem-local marker parsing

M4b's review pane added `Alt+q` visual wrapping as `🤖<selection>[]` but initially
embedded the selected text raw, even though annotate already had delimiter escaping for
the same marker family. A selection containing `>` or `]` could truncate the parsed marker
and make accept/reject leave stray syntax in the document.

**Rule.** When a second feature writes the same delimited marker format, reuse or extract
the existing codec before adding parser/writer code. Add tests for delimiter collisions
(`>`, `]`, backslash) at the write path and the consume path. A parser unit test alone is
not enough; the UI wrapper that inserts the marker must also be covered. Caught in #000066
M4b review.

## Shell scripts should use JSON builders, not `printf` JSON

`pair-review-readiness` originally printed JSON with `printf` and unescaped string fields.
A review branch named `review/a"b` produced invalid JSON, even though all the boolean
fields were correct.

**Rule.** In shell seams that emit JSON, use `jq -n --arg/--argjson` (or an equivalent
structured encoder) for every field. Do not hand-build JSON with `printf` unless every
string field is impossible by construction — and then document why. Guard it with a test
using quotes in a branch/path/name. Caught in #000066 M4b review.

## `#table` is 0 on string-keyed tables — never use it for ID generation

Adding nvim/scrollback.lua's hl-group cache: `local name = 'PairScrollback_' .. (#hl_cache + 1)` was meant to give each new (state→hl-group) entry a unique numeric suffix. `hl_cache` is a string-keyed dict (cache key is `(fg|bg|attrs)`), and Lua's `#` on a non-array table returns 0. Result: every group resolved to `PairScrollback_1`, `nvim_set_hl(0, "PairScrollback_1", def)` overwrote on each call, and all extmarks ended up sharing whatever the last-written attrs were. Caught only by an end-to-end test that checked extmark hl_groups against expected fg/bg ints.

**Rule.** When you need monotonic IDs in Lua, use an explicit counter (`local counter = 0; ... counter = counter + 1`). Do not use `#table` unless `table` is provably array-shaped (`{[1]=..., [2]=..., ...}`). The bug is silent — `nvim_set_hl` doesn't error on overwrite, it just wins-last. Filed during #000017 M4.

## Empty fields in delimited parsing — `[^;]+` drops them; semantics may differ

ECMA-48 SGR semantics: an omitted field is `0` (reset). So `\x1b[;1m` = "reset; bold". The first SGR parser pass used `params:gmatch('[^;]+')`, which silently skips empty fields — `[;1m` produced just `1` (bold), and any standing fg/bg/decoration leaked through. Caught in code review of #000017 (no real input from pair-scrollback-render's output would have triggered it, but it's a correctness footgun for any future caller pointing the viewer at non-pair-rendered ANSI).

**Rule.** When the protocol says "empty field has meaning," parse with `([^;]*);` on a `string + ';'` so the trailing-delimiter trick yields every field including empties. Generally true for any delimiter-separated format where omission has semantic value (CSV with empty cells, env-var lists, SGR, etc.).

## Sparse data structures: iterate by index, not by `.keys()`, when count must be exact

pyte's `screen.buffer` is a `StaticDefaultDict` — accessing `buffer[y][x]` lazily creates a default Char, but `buffer.keys()` only contains rows that were *written to*. The renderer originally did `for y in sorted(screen.buffer.keys())`, which silently dropped trailing blank rows when the agent cleared and paused mid-redraw. That shifts every subsequent line number — directly breaking the feature's core promise that `:880` lands where zellij showed line 880. Caught in code review of #000017.

**Rule.** When iterating over a sparse-by-design structure where every slot has a logical existence (even if unwritten), use `range(0, total)` and let the structure's `__getitem__` materialize defaults. `.keys()` is only correct when "absent" really means "doesn't exist." Same shape applies to anything with lazy materialization: defaultdicts, JS Maps with default fallbacks, sparse arrays.

## Atomic write for files a feature can race on its own

`bin/pair-scrollback-render` initially opened `<out.ansi>` with `'w'` (truncate-then-write). Two `Alt+/` presses in quick succession would race on the same path; whichever finished second left a half-interleaved file for nvim to open. Fixed by writing to `<out.ansi>.tmp` and `os.replace()`-ing at the end.

**Rule.** Any output file that a user-triggered keybind (or any concurrently-fireable mechanism) writes to should use the tempfile + atomic rename pattern. The cost is one extra file path; the gain is that readers see only "old complete file" or "new complete file," never "torn file." Apply uniformly even when a race is unlikely — discipline reduces the cognitive load for future readers.

## Verify zellij action and flag names against the installed version

Two bugs in v1 of `bin/pair` and `zellij/config.kdl` came from going off memory of zellij's API:

- Used `TogglePaneFullscreen` for the Alt+u bind. The actual action name in zellij 0.44.1 is `ToggleFocusFullscreen`. Caught by `zellij setup --check --config-dir <pair>/zellij`.
- Used `--layout PATH --session NAME` to "create a new named session with this layout." Zellij's actual semantic: when `--session` is set, `--layout` means "add as tab to that session" and errors if the session doesn't exist. The right flag is `--new-session-with-layout` (`-n`).

**Rule.** Before writing zellij KDL or invoking the zellij CLI:

1. Run `zellij setup --dump-config` to see the canonical action names used in default keybinds.
2. Run `zellij --help`, `zellij attach --help`, `zellij setup --help` against the installed version, and read the flag descriptions in full — they have non-obvious conditional semantics.
3. Always validate config and layout files with `zellij setup --check --config-dir <dir>` and `zellij setup --dump-layout <path>` before committing.

The verification tools are cheap and authoritative. Memory of "I think it's called X" is not.

## Stage content edits before `git mv` when closing an issue

Closing an issue means (a) editing the file (`status: done`, plan checkboxes), then (b) moving it to `workshop/history/`. Done in that order with `Edit` then `git mv`, the rename gets staged but the unstaged content edits do *not* — they stay in the working tree. `make issue-sync` only stages `workshop/issues/`, so the edits silently miss the commit. End state: history file with stale `status: working`.

**Rule.** When closing an issue:
1. Edit the file in place under `workshop/issues/` and `git add` it (or use `git add -u` after editing).
2. Then `git mv` to `workshop/history/` — git carries the staged content into the rename.
3. Or simpler: `git mv` first, edit second, `git add` the new path.

After running `make issue-sync` on a close, verify with `git show HEAD:workshop/history/<file> | grep status:` that the committed file actually has `status: done`. Don't trust the rename alone.

## On cancel, restore the prior visible state

When a confirmation prompt or interactive flow is dismissed, the cancel path must put the UI back exactly how it was — not just "do nothing." Issuing a prompt via `nvim_echo`/`getchar` (or any flow that paints over a region: cmdline, statusline, floating windows, virtual text, highlights) leaves that region in the prompt's state. The proceed branch usually triggers a redraw incidentally (state changes → statusline refresh → cmdline cleared). The cancel branch does not, so the prompt residue lingers until the next user input.

**Rule.** For every interactive surface, the cancel path is responsible for the same restoration the proceed path gets for free:

- Prompts that overdraw the cmdline/statusline → call the same redraw/refresh helper the success path calls (e.g. `refresh_statusline()`), not just `return`.
- Operations that mutated buffer text/cursor/window before asking for confirmation → snapshot first, restore on cancel.
- Highlights, virtual text, floating windows added as part of the flow → tear them down on cancel just like on success.

Treat cancel as an active branch with cleanup duties, not an early return. If you find yourself writing `if ch == 'n' then return end`, ask: what did the proceed branch do that I'm now skipping, and is any of it visual cleanup that cancel also needs?

## Transcript summarization must bias toward USER turns, not a flat tail

`cmd/pair-slug` (#000027) summarized "what is this session about" by feeding
the last N text-bearing transcript turns to a small model. On a tool-heavy
session that window is almost entirely assistant narration: a real Claude
transcript had ~16 genuine user prompts vs ~200 assistant entries (most
`user` entries carry only `tool_result` blocks, correctly dropped as
text-less). Measured: the last 10 text-bearing turns were 10/10 assistant,
0 user. So the slug tracked what the agent was *saying*, not what the user
*asked for* — the orientation signal was pushed out of the window. The unit
tests passed because their fixtures used only text-content messages, never
the dominant `tool_result`-only user shape — green tests masked the bug.
Caught in #000027 M1 review.

**Rule.** When sampling a conversation transcript to infer user intent:
- Don't take a flat tail of turns. Guarantee a minimum number of recent
  *user* turns are in the window (extend backward until satisfied, capped).
- Model test fixtures on the *real* transcript shape, including
  `tool_result`-only user entries and any sidechain/summary types — not the
  clean text-only case. A fixture that can't reproduce the bug can't guard
  against it.

## `gofmt -w <dir>` reformats files you didn't touch

Running `gofmt -w cmd/pair-wrap/` to format M3's edited `main.go` also
rewrote four pre-existing `*_test.go` files (struct-field alignment) that the
milestone never touched, staging unrelated churn into the commit. Caught at
`git status` review before commit; reverted with `git checkout -- <files>`.

**Rule.** Format only the files the change actually touches: `gofmt -w
path/to/file.go` (or `gofmt -w $(git diff --name-only '*.go')`), not the whole
package directory. If a dir-wide gofmt lights up files outside the change,
revert them — don't smuggle repo-wide reformatting into a feature commit.
Caught in #000027 M3.

## Dogfooding a Go-binary change needs `make install`, not just `make build`

M3's pair-wrap trigger "didn't fire" on restart. Trace: pair-slug worked in
isolation, but the running `pair-wrap` (pid via `pair-wrap-pid-<tag>`, binary
via `lsof -p <pid> | awk '$4=="txt"`) was `~/.local/bin/pair-wrap` dated days
earlier — the *installed* copy, with no spawn. I had only `go build -o bin/…`;
the layout (`zellij/layouts/main.kdl`) execs `pair-wrap` by bare name and the
pane's PATH resolved `~/.local/bin` first.

**Rule.** `bin/` is the repo build; `~/.local/bin` (via `make install`) is what
actually runs in a live pair session. To dogfood a change to a Go binary
(pair-wrap/pair-slug/…): `make install`, *then* restart pair. Verifying with
`bin/<binary>` alone proves nothing about the running session. When a "live"
change seems inert, confirm the running binary: `lsof -p $(cat
$PAIR_DATA_DIR/pair-wrap-pid-<tag>) | awk '$4=="txt"{print $NF}'`. Caught in
#000027 M3 dogfood.

## Queue items: resolve by filename key, not display index, across a mutation

Sending from a future-queue slot (`+N`) while the draft `*` was non-empty left
the sent item in BOTH the queue (`+N`) and history (`-1`). Root cause:
`send_and_clear` resolved the item to remove via `queue_key_for_n(nav.pos.n)` —
the *display index* — but the new "park the draft into the queue first"
(`push_front`) step shifts every index by one. Resolving by the stale index
then removed the wrong file (or `nil`), so the actually-sent item was logged to
history but never deleted from the queue → duplication.

**Rule.** A `+N` display index is only valid against the queue snapshot it was
read from. The moment any queue mutation (`push_front`/`push_back`/remove) can
intervene, capture the item's **filename key first** (`queue_key_for_n(n)` →
`NNNNNN`), then mutate, then remove by that stable key. Keys don't move on
insert; indices do. Verified the duplication via a headless driver
(`nvim --headless -u nvim/init.lua` + `maparg().callback`) before fixing, and
guarded it with `tests/queue-send-test.sh` (`make test-queue`).

## strings.ToLower can change byte length — don't cross-index a folded copy

`promptShape` matched against `strings.ToLower(visible)` but then sliced the
**original** `visible` at the match offset. `ToLower` is not length-preserving
(e.g. `Ⱥ` U+023A, 2 bytes → `ⱥ` U+2C65, 3 bytes), so on agent output with such a
rune the offset exceeded `len(visible)` and panicked the slice. The panic was
swallowed by `handleChunk`'s `recover`, but that `recover` wraps the whole
detect block, so OSC-notification + bell handling were silently skipped for that
chunk — a diagnostic-only feature altering proxy behavior. Surfaced in #000045
M1 review (C1).

**Rule.** If you compute a byte offset in one string, slice the *same* string —
never a transformed copy whose length can differ. For case-insensitive matching
where you need offsets back in the original, use a **length-preserving** fold
(ASCII-only `asciiFold`) and clamp slice indices defensively. Add a multibyte
test case (`Ⱥ`/`İ`/`Å`) — ASCII-only tests can't catch this.

## jq slurp (`-s`) over a JSONL file aborts on one bad line

`doctor.sh` read the flight recorder with `jq -rs '…'`, which parses the whole
file as one array — so a single malformed/partial line (a writer crashing
mid-line; O_APPEND only guarantees atomicity below PIPE_BUF) made jq error and,
under `set -euo pipefail`, killed the script. The operator got a jq stack trace
and zero diagnostics exactly when they needed the tool. Surfaced in #000045 M1
review (I1).


codex
I have the repository workflow loaded. Next I’m checking the actual issue artifact and HEAD files so the review is against source state, not just the pasted diff.
exec
/bin/zsh -lc "sed -n '1,220p' /Users/xianxu/workspace/ariadne/construct/local/sdlc/SKILL.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
---
name: sdlc
description: Use when at an SDLC checkpoint — starting work, closing an issue or milestone, opening/merging a PR, or recovering workflow state after compaction. The `sdlc` binary owns the gates between workflow stages and refuses transitions that lack required evidence.
---

# sdlc — SDLC checkpoint binary

`sdlc` owns the gates between SDLC workflow stages (claim → change-code → pr →
merge, plus close, milestone-close, judge). It requires evidence at each gate,
mutates state, logs the transition, and refuses transitions that lack the
evidence — that is the shape of a "checkpoint guard."

The binary is the single source of truth. This skill is a static pointer and
intentionally carries no copy of the contract, so it can never drift:

- **`sdlc --help`** — the workflow contract: the start-of-work runbook,
  conventions, and the verb list.
- **`sdlc <verb> --help`** — one checkpoint's full contract, flags, and examples.

Read those instead of relying on memory; the binary's help is always current.

exec
/bin/zsh -lc 'git diff --stat 1199bb427dfc0d7c7b87dcc1115ce19907c4d804..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 Makefile.local                                     |  10 +-
 atlas/architecture.md                              |   6 +
 cmd/internal/dispatcher/dispatcher.go              |  89 ++++++++++++
 cmd/internal/dispatcher/dispatcher_test.go         |  76 ++++++++++
 cmd/pair-go/main.go                                |  25 ++++
 cmd/pair-go/main_test.go                           |  35 +++++
 workshop/issues/000074-go-dispatcher-skeleton.md   |  43 ++++--
 .../plans/000074-go-dispatcher-skeleton-plan.md    | 153 +++++++++++++++++++++
 8 files changed, 424 insertions(+), 13 deletions(-)

exec
/bin/zsh -lc "rg --files workshop | rg '000074|74|go-dispatcher'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
workshop/issues/000074-go-dispatcher-skeleton.md
workshop/plans/000074-go-dispatcher-skeleton-plan.md

exec
/bin/zsh -lc 'git diff --name-only 1199bb427dfc0d7c7b87dcc1115ce19907c4d804..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
Makefile.local
atlas/architecture.md
cmd/internal/dispatcher/dispatcher.go
cmd/internal/dispatcher/dispatcher_test.go
cmd/pair-go/main.go
cmd/pair-go/main_test.go
workshop/issues/000074-go-dispatcher-skeleton.md
workshop/plans/000074-go-dispatcher-skeleton-plan.md

codex
The diff is small and centered on a new opt-in Go binary plus a pure dispatcher package. I’m now verifying the issue’s done-when items, the plan’s core-concepts table, and whether the Makefile target accurately builds what the tests exercise.
exec
/bin/zsh -lc "nl -ba cmd/pair-go/main.go | sed -n '1,120p'" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "nl -ba workshop/plans/000074-go-dispatcher-skeleton-plan.md | sed -n '1,240p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	// pair-go is a development-only dispatcher skeleton for the future primary Go
     2	// CLI. The public launcher remains bin/pair.
     3	package main
     4	
     5	import (
     6		"io"
     7		"os"
     8	
     9		"github.com/xianxu/pair/cmd/internal/dispatcher"
    10	)
    11	
    12	func main() {
    13		os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
    14	}
    15	
    16	func run(args []string, stdout, stderr io.Writer) int {
    17		res := dispatcher.Dispatch(args)
    18		if res.Stdout != "" {
    19			_, _ = io.WriteString(stdout, res.Stdout)
    20		}
    21		if res.Stderr != "" {
    22			_, _ = io.WriteString(stderr, res.Stderr)
    23		}
    24		return res.ExitCode
    25	}

 succeeded in 0ms:
     1	# Pair Go Dispatcher Skeleton Implementation Plan
     2	
     3	> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.
     4	
     5	**Goal:** Add an opt-in Go dispatcher skeleton that establishes Pair's future primary CLI shape without changing the public `bin/pair` launcher.
     6	
     7	**Architecture:** Create a pure `cmd/internal/dispatcher` package for argv parsing, help rendering, planned command metadata, and unsupported-command errors (`ARCH-PURE`). Add `cmd/pair-go/main.go` as a thin IO shell that calls the dispatcher and writes output/exit codes. Wire `pair-go` into the existing `GO_BINS` build pattern while leaving `bin/pair`, zellij layouts, and helper command names unchanged (`ARCH-DRY`, `ARCH-PURPOSE`).
     8	
     9	**Tech Stack:** Go 1.26, standard library only, existing `Makefile.local` build conventions.
    10	
    11	---
    12	
    13	## Core Concepts
    14	
    15	### Pure Entities
    16	
    17	| Name | Lives in | Status |
    18	|------|----------|--------|
    19	| `CommandFamily` | `cmd/internal/dispatcher/dispatcher.go` | new |
    20	| `Result` | `cmd/internal/dispatcher/dispatcher.go` | new |
    21	| `Dispatch` | `cmd/internal/dispatcher/dispatcher.go` | new |
    22	| `Help` | `cmd/internal/dispatcher/dispatcher.go` | new |
    23	
    24	- **CommandFamily** — metadata for planned Pair command families that the skeleton can list without implementing.
    25	  - **Relationships:** N:1 with `Help`; `Help` renders all families from one slice.
    26	  - **DRY rationale:** One source for names/descriptions/status text instead of duplicating planned command lists between help and dispatch validation.
    27	  - **Future extensions:** Add handler wiring when later issues port a family.
    28	
    29	- **Result** — pure dispatch outcome: stdout, stderr, and process exit code.
    30	  - **Relationships:** 1:1 with a dispatcher invocation; consumed by the IO shell.
    31	  - **DRY rationale:** Keeps exit behavior testable without spawning a subprocess.
    32	  - **Future extensions:** Can grow a handler enum or execution callback once subcommands become real.
    33	
    34	- **Dispatch** — pure argv parser and router for `help`, `--help`, `version`, and unsupported commands.
    35	  - **Relationships:** Owns one invocation's decision; reads `CommandFamily` metadata; returns `Result`.
    36	  - **DRY rationale:** Centralizes command parsing before behavior begins migrating out of shell scripts.
    37	  - **Future extensions:** Replace unsupported-command outcomes with real handlers one family at a time.
    38	
    39	- **Help** — pure renderer for the dev CLI usage and planned command families.
    40	  - **Relationships:** Reads `CommandFamily`; used by `Dispatch`.
    41	  - **DRY rationale:** Help text and tests derive from the same command-family metadata.
    42	  - **Future extensions:** Add richer command-specific help when handlers exist.
    43	
    44	### Integration Points
    45	
    46	| Name | Lives in | Status | Wraps |
    47	|------|----------|--------|-------|
    48	| `pair-go main` | `cmd/pair-go/main.go` | new | `os.Args`, stdout/stderr, process exit |
    49	| `pair-go build target` | `Makefile.local` | modified | `go build` into `bin/pair-go` |
    50	| `architecture note` | `atlas/architecture.md` | modified | repo map documentation |
    51	
    52	- **pair-go main** — minimal binary entrypoint: call `dispatcher.Dispatch(os.Args[1:])`, print streams, exit.
    53	  - **Injected into:** No pure entity; it consumes the pure `Result`.
    54	  - **Future extensions:** Later issues can replace result-only routing with handler execution behind the same package.
    55	
    56	- **pair-go build target** — follow the existing `GO_BINS` and per-binary recipe convention so `make build` compiles the skeleton.
    57	  - **Injected into:** Build/install flow only.
    58	  - **Future extensions:** The public `pair` entrypoint switch happens in a later issue, not here.
    59	
    60	- **architecture note** — record that `pair-go` is the opt-in dispatcher skeleton and not the public launcher.
    61	  - **Injected into:** Atlas readers and future migration issues.
    62	  - **Future extensions:** Update when the public entrypoint changes.
    63	
    64	## Chunk 1: Dispatcher Skeleton
    65	
    66	### Task 1: Add pure dispatch tests
    67	
    68	**Files:**
    69	- Create: `cmd/internal/dispatcher/dispatcher_test.go`
    70	- Later create: `cmd/internal/dispatcher/dispatcher.go`
    71	
    72	- [x] **Step 1: Write failing tests for help, version, and unsupported commands**
    73	
    74	Cover:
    75	- empty argv and `help` return exit `0` with usage and planned families.
    76	- `--help` and `-h` mirror help.
    77	- `version` returns deterministic dev metadata without pretending to be the shell launcher.
    78	- a planned-but-unimplemented command like `wrap` returns exit `2` and an unsupported message.
    79	- an unknown command returns exit `2` and suggests `pair-go help`.
    80	
    81	- [x] **Step 2: Run the package test and verify it fails**
    82	
    83	Run: `go test ./cmd/internal/dispatcher -count=1`
    84	
    85	Expected: fail because `dispatcher.go` does not exist yet.
    86	
    87	- [x] **Step 3: Implement the minimal dispatcher**
    88	
    89	Create:
    90	- `CommandFamily` with `Name`, `Summary`, and `Status`.
    91	- `Families()` returning planned command families: `launch`, `wrap`, `slug`, `context`, `scrollback-render`, `changelog`, `continuation`, `scribe`.
    92	- `Result` with `Stdout`, `Stderr`, `ExitCode`.
    93	- `Dispatch(args []string) Result`.
    94	- `Help(program string) string`.
    95	
    96	- [x] **Step 4: Run dispatcher tests and verify they pass**
    97	
    98	Run: `go test ./cmd/internal/dispatcher -count=1`
    99	
   100	Expected: pass.
   101	
   102	### Task 2: Add the opt-in binary wrapper and build target
   103	
   104	**Files:**
   105	- Create: `cmd/pair-go/main.go`
   106	- Modify: `Makefile.local`
   107	
   108	- [x] **Step 1: Write a failing wrapper smoke test if needed**
   109	
   110	If the pure tests cover all behavior, keep `main.go` untested directly and rely on `go test ./...` plus build verification. Do not introduce subprocess tests unless pure coverage misses behavior.
   111	
   112	- [x] **Step 2: Implement `cmd/pair-go/main.go`**
   113	
   114	Main reads `os.Args[1:]`, delegates to `dispatcher.Dispatch`, writes stdout/stderr to the matching streams, and exits with the returned code.
   115	
   116	- [x] **Step 3: Wire `pair-go` into `Makefile.local`**
   117	
   118	Append `pair-go` to `GO_BINS`, add the `.PHONY` alias, and add:
   119	
   120	```make
   121	$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/dispatcher/dispatcher.go go.mod
   122		go build -o $@ ./cmd/pair-go
   123	```
   124	
   125	- [x] **Step 4: Build the new binary**
   126	
   127	Run: `make pair-go`
   128	
   129	Expected: `bin/pair-go` is created.
   130	
   131	### Task 3: Document and verify non-disruption
   132	
   133	**Files:**
   134	- Modify: `atlas/architecture.md`
   135	- Modify: `workshop/issues/000074-go-dispatcher-skeleton.md`
   136	
   137	- [x] **Step 1: Update atlas**
   138	
   139	Add `bin/pair-go` to the piece list as the opt-in Go dispatcher skeleton and mention it in the packaging migration target as development-only for this issue.
   140	
   141	- [x] **Step 2: Run focused verification**
   142	
   143	Run:
   144	- `go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
   145	- `make pair-go`
   146	- `go test ./... -count=1`
   147	- `bash tests/pair-continue-test.sh`
   148	
   149	Expected: all pass. The shell launcher remains the public entrypoint.
   150	
   151	- [x] **Step 3: Update issue checkboxes and log**
   152	
   153	Mark completed plan and done-when boxes, then close through `sdlc close --issue 74 --verified '<evidence>'` with atlas evidence. This plan expects an atlas update, so prefer satisfying the atlas gate.

exec
/bin/zsh -lc "nl -ba workshop/issues/000074-go-dispatcher-skeleton.md | sed -n '1,240p'" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "nl -ba cmd/internal/dispatcher/dispatcher.go | sed -n '1,180p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package dispatcher
     2	
     3	import (
     4		"fmt"
     5		"strings"
     6	)
     7	
     8	const programName = "pair-go"
     9	
    10	// CommandFamily names a future Pair CLI surface without claiming it works yet.
    11	type CommandFamily struct {
    12		Name    string
    13		Summary string
    14		Status  string
    15	}
    16	
    17	// Result is the process-facing outcome of a pure dispatch decision.
    18	type Result struct {
    19		Stdout   string
    20		Stderr   string
    21		ExitCode int
    22	}
    23	
    24	// Families returns the planned command families for the Go dispatcher.
    25	func Families() []CommandFamily {
    26		return []CommandFamily{
    27			{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "planned"},
    28			{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
    29			{Name: "slug", Summary: "session orientation slug generation", Status: "planned"},
    30			{Name: "context", Summary: "agent pane context meter", Status: "planned"},
    31			{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "planned"},
    32			{Name: "changelog", Summary: "TTY transcript to distilled change log", Status: "planned"},
    33			{Name: "continuation", Summary: "continuation datatype writer", Status: "planned"},
    34			{Name: "scribe", Summary: "PTY logging wrapper", Status: "planned"},
    35		}
    36	}
    37	
    38	// Dispatch parses argv and returns the skeleton dispatch result.
    39	func Dispatch(args []string) Result {
    40		if len(args) == 0 {
    41			return Result{Stdout: Help(programName), ExitCode: 0}
    42		}
    43	
    44		switch args[0] {
    45		case "help", "--help", "-h":
    46			return Result{Stdout: Help(programName), ExitCode: 0}
    47		case "version", "--version":
    48			return Result{
    49				Stdout:   "pair-go dispatcher skeleton\npublic launcher: bin/pair\n",
    50				ExitCode: 0,
    51			}
    52		}
    53	
    54		if family, ok := familyByName(args[0]); ok {
    55			return Result{
    56				Stderr:   fmt.Sprintf("%s: %s is planned but not implemented in this skeleton; run %s help\n", programName, family.Name, programName),
    57				ExitCode: 2,
    58			}
    59		}
    60	
    61		return Result{
    62			Stderr:   fmt.Sprintf("%s: unknown command %q; run %s help\n", programName, args[0], programName),
    63			ExitCode: 2,
    64		}
    65	}
    66	
    67	// Help renders the development-only dispatcher usage text.
    68	func Help(program string) string {
    69		var b strings.Builder
    70		fmt.Fprintf(&b, "Usage: %s <command> [args]\n\n", program)
    71		b.WriteString("Development dispatcher skeleton. Public sessions still start through bin/pair.\n\n")
    72		b.WriteString("Planned command families (not implemented in this skeleton):\n")
    73		for _, family := range Families() {
    74			fmt.Fprintf(&b, "  %-17s %s (%s; not implemented in this skeleton)\n", family.Name, family.Summary, family.Status)
    75		}
    76		b.WriteString("\nSupported skeleton commands:\n")
    77		b.WriteString("  help              show this help\n")
    78		b.WriteString("  version           show dispatcher skeleton metadata\n")
    79		return b.String()
    80	}
    81	
    82	func familyByName(name string) (CommandFamily, bool) {
    83		for _, family := range Families() {
    84			if family.Name == name {
    85				return family, true
    86			}
    87		}
    88		return CommandFamily{}, false
    89	}

 succeeded in 0ms:
     1	---
     2	id: 000074
     3	status: done
     4	deps: [000073]
     5	github_issue:
     6	created: 2026-06-26
     7	updated: 2026-06-29
     8	estimate_hours: 1.39
     9	started: 2026-06-29T17:00:26-07:00
    10	actual_hours: 0.14
    11	---
    12	
    13	# pair Go dispatcher skeleton
    14	
    15	## Problem
    16	
    17	The target architecture needs a primary Go command, but switching the public launcher immediately would be too risky. The first code step should introduce the dispatch shape without changing user-visible behavior.
    18	
    19	## Spec
    20	
    21	Add a Go dispatcher skeleton that can host Pair subcommands behind an explicit development path. It should establish command parsing, help text shape, version/build metadata if needed, and an internal routing pattern for future subcommands.
    22	
    23	The existing `bin/pair` shell launcher remains the public entrypoint for this issue. Any new Go command must be opt-in, for example a new built binary or a hidden/dev invocation, so this can merge without affecting normal sessions.
    24	
    25	Design constraints:
    26	
    27	- Reuse existing package structure where possible (`ARCH-DRY`).
    28	- Keep command parsing and dispatch decision logic pure enough to unit-test (`ARCH-PURE`).
    29	- Do not port launcher behavior yet. This issue is only the skeleton.
    30	
    31	## Done when
    32	
    33	- [x] A Go dispatcher command exists and builds in the normal `make build` flow or an explicitly documented dev target.
    34	- [x] Dispatcher help lists the planned command families without claiming unsupported behavior works.
    35	- [x] Public `bin/pair` behavior is unchanged.
    36	- [x] Tests cover dispatch parsing and unsupported-command errors.
    37	- [x] Pair remains usable after merge through the existing `pair` entrypoint.
    38	
    39	## Estimate
    40	
    41	```estimate
    42	model: estimate-logic-v3.1
    43	familiarity: 1.0
    44	item: skill-or-dispatcher design=0.30 impl=0.25
    45	item: smaller-go-module design=0.20 impl=0.20
    46	item: atlas-docs design=0.10 impl=0.05
    47	item: milestone-review design=0.00 impl=0.20
    48	design-buffer: 0.15
    49	total: 1.39
    50	```
    51	
    52	Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.
    53	
    54	## Plan
    55	
    56	- [x] Choose the command location/name based on #73 inventory.
    57	- [x] Add pure dispatch parsing tests.
    58	- [x] Add the skeleton implementation.
    59	- [x] Wire build/install only in a non-disruptive way.
    60	- [x] Verify existing Pair flows still use the shell launcher.
    61	
    62	## Log
    63	
    64	### 2026-06-26
    65	
    66	Created from #72 as the safe first code-bearing step toward one primary Go command.
    67	
    68	### 2026-06-29
    69	- 2026-06-29: closed — go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; bash tests/pair-continue-test.sh; git diff -- bin/pair empty
    70	
    71	Claimed and entered planning. Design uses a new opt-in `pair-go` binary plus pure `cmd/internal/dispatcher` package so command parsing/help/error behavior is unit-testable (`ARCH-PURE`) and `bin/pair` remains unchanged while the skeleton fulfills the issue purpose (`ARCH-PURPOSE`). Durable plan: `workshop/plans/000074-go-dispatcher-skeleton-plan.md`.
    72	
    73	Implemented `cmd/internal/dispatcher` and `cmd/pair-go` behind a non-public `pair-go` build target. Wrapper behavior is covered by `cmd/pair-go/main_test.go` rather than subprocess tests because the pure dispatcher tests cover parsing/help/error semantics and the wrapper only writes the returned streams/exit code. Verified with `env GOCACHE=/private/tmp/pair-go-cache GOMODCACHE=/private/tmp/pair-gomod-cache go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1`, `env GOCACHE=/private/tmp/pair-go-cache GOMODCACHE=/private/tmp/pair-gomod-cache make -B pair-go`, `env GOCACHE=/private/tmp/pair-go-cache GOMODCACHE=/private/tmp/pair-gomod-cache go test ./... -count=1`, `bash tests/pair-continue-test.sh`, and `git diff -- bin/pair` (empty).

exec
/bin/zsh -lc "nl -ba Makefile.local | sed -n '1,75p;225,255p'" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "nl -ba cmd/internal/dispatcher/dispatcher_test.go | sed -n '1,140p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package dispatcher
     2	
     3	import (
     4		"strings"
     5		"testing"
     6	)
     7	
     8	func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
     9		for _, args := range [][]string{nil, {"help"}, {"--help"}, {"-h"}} {
    10			t.Run(strings.Join(args, "_"), func(t *testing.T) {
    11				res := Dispatch(args)
    12				if res.ExitCode != 0 {
    13					t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
    14				}
    15				if res.Stderr != "" {
    16					t.Fatalf("Stderr = %q, want empty", res.Stderr)
    17				}
    18				for _, want := range []string{
    19					"Usage: pair-go <command> [args]",
    20					"launch",
    21					"wrap",
    22					"slug",
    23					"not implemented in this skeleton",
    24				} {
    25					if !strings.Contains(res.Stdout, want) {
    26						t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
    27					}
    28				}
    29			})
    30		}
    31	}
    32	
    33	func TestDispatchVersionIsDevelopmentSkeletonMetadata(t *testing.T) {
    34		res := Dispatch([]string{"version"})
    35		if res.ExitCode != 0 {
    36			t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
    37		}
    38		if res.Stderr != "" {
    39			t.Fatalf("Stderr = %q, want empty", res.Stderr)
    40		}
    41		for _, want := range []string{"pair-go", "dispatcher skeleton", "public launcher: bin/pair"} {
    42			if !strings.Contains(res.Stdout, want) {
    43				t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
    44			}
    45		}
    46	}
    47	
    48	func TestDispatchPlannedCommandReturnsUnsupported(t *testing.T) {
    49		res := Dispatch([]string{"wrap"})
    50		if res.ExitCode != 2 {
    51			t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
    52		}
    53		if res.Stdout != "" {
    54			t.Fatalf("Stdout = %q, want empty", res.Stdout)
    55		}
    56		for _, want := range []string{"wrap", "planned", "not implemented", "pair-go help"} {
    57			if !strings.Contains(res.Stderr, want) {
    58				t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
    59			}
    60		}
    61	}
    62	
    63	func TestDispatchUnknownCommandReturnsUsageHint(t *testing.T) {
    64		res := Dispatch([]string{"frobnicate"})
    65		if res.ExitCode != 2 {
    66			t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
    67		}
    68		if res.Stdout != "" {
    69			t.Fatalf("Stdout = %q, want empty", res.Stdout)
    70		}
    71		for _, want := range []string{"unknown command", "frobnicate", "pair-go help"} {
    72			if !strings.Contains(res.Stderr, want) {
    73				t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
    74			}
    75		}
    76	}

 succeeded in 0ms:
     1	# Repo-specific Makefile targets.
     2	# This file is included by Makefile — add your own targets here.
     3	
     4	.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
     5	
     6	# Cross-cutting Go-binary targets. Every Go binary lives under cmd/<name>/
     7	# and ships with the `pair-` prefix so it doesn't collide with anything
     8	# else on PATH (pair-scribe replaced an earlier bare-named `scribe`).
     9	#
    10	# Adding a new binary: drop main.go under cmd/<name>/, append <name>
    11	# to GO_BINS, add the per-binary recipe stanza. `make build` and
    12	# `make install` pick it up automatically.
    13	#
    14	#   make build      → compile every binary in GO_BINS into ./bin/
    15	#   make install    → also copy each into ~/.local/bin/
    16	#   make <name>     → build just that one (e.g., `make pair-scribe`)
    17	#
    18	# Why ./bin/ for the in-tree build: keeps Go output next to the shell
    19	# scripts pair already ships from bin/, and pair-scrollback-open looks
    20	# at $PAIR_HOME/bin/pair-scrollback-render to pick up the Go renderer
    21	# in both dev (source tree) and brew-install layouts.
    22	#
    23	# Why ~/.local/bin/ for install: XDG convention, on PATH per the user's
    24	# zshrc. Each binary the user wires into their shell (e.g. pair-scribe
    25	# in zshrc) should reference $HOME/.local/bin/<name>.
    26	
    27	BIN_DIR     := bin
    28	INSTALL_DIR := $(HOME)/.local/bin
    29	GO_BINS     := pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-go
    30	
    31	build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
    32	
    33	install: build
    34		@mkdir -p $(INSTALL_DIR)
    35		@for b in $(GO_BINS); do \
    36		    install -m 755 $(BIN_DIR)/$$b $(INSTALL_DIR)/$$b && \
    37		    echo "installed: $(INSTALL_DIR)/$$b"; \
    38		done
    39	
    40	# Per-binary aliases — `make pair-scribe`, etc.
    41	pair-scribe:            $(BIN_DIR)/pair-scribe
    42	pair-scrollback-render: $(BIN_DIR)/pair-scrollback-render
    43	pair-wrap:              $(BIN_DIR)/pair-wrap
    44	pair-slug:              $(BIN_DIR)/pair-slug
    45	pair-continuation:      $(BIN_DIR)/pair-continuation
    46	pair-changelog:         $(BIN_DIR)/pair-changelog
    47	pair-context:           $(BIN_DIR)/pair-context
    48	pair-go:                $(BIN_DIR)/pair-go
    49	
    50	# Tests
    51	#   make test       → go test ./... (cache disabled so reruns actually run)
    52	#   make test-race  → race-detector run, scoped to packages where the
    53	#                     suite has actual concurrent code worth checking.
    54	#                     pair-wrap has translateStdin's goroutine + timer +
    55	#                     channels; pair-scrollback-render's render() spawns
    56	#                     a goroutine drainer that races with em.Close()
    57	#                     inside the vt.Emulator library API itself, which is
    58	#                     outside our scope to fix — exclude that package
    59	#                     here rather than chase a library-level race.
    60	test: test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace test-review
    61		go test ./... -count=1
    62	
    63	# pair-title poller: single-instance guard (identity-checked liveness so a
    64	# recycled PID can't suppress the respawn) + the #71 frame meter (renames each
    65	# agent pane to "<agent> (<count>) [<cwd>]", with unchanged-skip). See
    66	# tests/pair-title-poller-test.sh.
    67	test-pair-title:
    68		bash tests/pair-title-poller-test.sh
    69	
    70	# cmux workspace ownership claim (presence beats a stale owner flag), so a
    71	# launch/attach/restart reclaims its workspace title. See
    72	# tests/cmux-ownership-test.sh.
    73	test-cmux-ownership:
    74		bash tests/cmux-ownership-test.sh
    75	
   225	
   226	# pair-context — one-shot: print a pane's current context size (token count)
   227	# for the pair-title poller. Pure ctxmeter reader + shared transcript resolver.
   228	$(BIN_DIR)/pair-context: cmd/pair-context/main.go cmd/internal/ctxmeter/ctxmeter.go cmd/internal/transcript/transcript.go go.mod
   229		go build -o $@ ./cmd/pair-context
   230	
   231	# pair-changelog — distills the pair TTY into the Alt+l change log (#53);
   232	# shares cmd/internal/model with pair-slug.
   233	$(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill.go cmd/pair-changelog/prompt.go cmd/internal/model/model.go go.mod
   234		go build -o $@ ./cmd/pair-changelog
   235	
   236	# pair-continuation — the deterministic writer for the `continuation` datatype
   237	# (ariadne#91): renders frontmatter, allocates a collision-safe timestamped
   238	# name under workshop/continuation/, writes, then commits + pushes (recovery).
   239	$(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-continuation/git.go cmd/pair-continuation/main.go go.mod
   240		go build -o $@ ./cmd/pair-continuation
   241	
   242	# pair-go — development-only dispatcher skeleton for the future primary Go CLI.
   243	# The public launcher remains bin/pair until the entrypoint-switch issue.
   244	$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/dispatcher/dispatcher.go go.mod
   245		go build -o $@ ./cmd/pair-go
   246	
   247	
   248	# Tart VM targets (tart, tart-gui, tart-mount, tart-stop, tart-clean)
   249	# live in .tart/Makefile, vendored from ariadne and auto-included by
   250	# Makefile.workflow. `make help-tart` for usage.

exec
/bin/zsh -lc "nl -ba atlas/architecture.md | sed -n '1,80p'" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "nl -ba cmd/pair-go/main_test.go | sed -n '1,90p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package main
     2	
     3	import (
     4		"bytes"
     5		"strings"
     6		"testing"
     7	)
     8	
     9	func TestRunWritesStdoutAndReturnsDispatcherCode(t *testing.T) {
    10		var stdout, stderr bytes.Buffer
    11		code := run([]string{"help"}, &stdout, &stderr)
    12		if code != 0 {
    13			t.Fatalf("code = %d, want 0", code)
    14		}
    15		if !strings.Contains(stdout.String(), "Usage: pair-go <command> [args]") {
    16			t.Fatalf("stdout missing usage:\n%s", stdout.String())
    17		}
    18		if stderr.String() != "" {
    19			t.Fatalf("stderr = %q, want empty", stderr.String())
    20		}
    21	}
    22	
    23	func TestRunWritesStderrAndReturnsDispatcherCode(t *testing.T) {
    24		var stdout, stderr bytes.Buffer
    25		code := run([]string{"wrap"}, &stdout, &stderr)
    26		if code != 2 {
    27			t.Fatalf("code = %d, want 2", code)
    28		}
    29		if stdout.String() != "" {
    30			t.Fatalf("stdout = %q, want empty", stdout.String())
    31		}
    32		if !strings.Contains(stderr.String(), "wrap is planned but not implemented") {
    33			t.Fatalf("stderr missing unsupported-command message:\n%s", stderr.String())
    34		}
    35	}

 succeeded in 0ms:
     1	# Architecture
     2	
     3	## What pair is
     4	
     5	A launcher that starts a zellij session with a fixed two-pane split. The top pane runs a TUI coding agent; the bottom pane runs Neovim on a persistent draft file. Keystrokes — and mouse-up after a selection — drive bidirectional flow between the panes via `zellij action write-chars` and `zellij action focus-pane-id`.
     6	
     7	The whole thing is deliberately small — a handful of shell scripts, one nvim init, and two zellij KDL files. Required deps: `zellij`, `nvim`, `fzf`, `jq`, `par`, plus the agent itself.
     8	
     9	## Pieces
    10	
    11	```
    12	bin/pair                     # entry point (launcher)
    13	bin/pair-go                  # opt-in Go dispatcher skeleton; public launcher remains bin/pair
    14	bin/clipboard-to-pane.sh     # read clipboard, hand off to nvim's PairPasteQuote
    15	bin/copy-on-select.sh        # invoked by zellij copy_command on mouse-up
    16	bin/pair-quit.sh             # invoked by Alt+x — marks + kills session
    17	bin/pair-restart.sh          # invoked by Alt+n / Shift+Alt+N — marks (quit + restart) + kills session
    18	bin/pair-session-watch.sh    # captures codex/agy session id at create time (#000016, #000020)
    19	bin/pair-wrap                # PTY proxy: OSC translation + scrollback capture
    20	bin/pair-notify              # hook-driven OSC notifier (e.g. claude Notification)
    21	bin/pair-scrollback-render   # raw PTY capture → ANSI-colored line dump (#000017)
    22	bin/pair-scrollback-open     # Alt+/ orchestrator: render + open viewer
    23	bin/pair-changelog           # TTY → distilled change log (LLM, incremental) (#53)
    24	bin/pair-changelog-open      # Alt+l orchestrator: open viewer; clean+distill run in the background (#53)
    25	nvim/init.lua                # bundled nvim config (loaded via -u)
    26	nvim/scrollback.lua          # read-only ANSI viewer for the scrollback dump
    27	nvim/changelog.lua           # read-only viewer for the distilled change log (#53)
    28	nvim/annotate.lua            # shared 🤖-marker annotation subsystem (Alt+q) for both viewers (#57)
    29	zellij/config.kdl            # mouse, copy_command, keybinds, pane frames
    30	zellij/layouts/main.kdl      # the split + agent/draft commands + swap layouts
    31	```
    32	
    33	## Packaging migration target (#72)
    34	
    35	Pair is moving toward a single primary Go `pair` binary for packaging and
    36	distribution. The target shape is a Go-owned CLI/dispatcher that owns session
    37	lifecycle, data/config path resolution, asset discovery, restart/quit/continue
    38	flows, and subprocess orchestration. Existing Go command surfaces should become
    39	internal subcommands or dispatch modes behind that primary binary (`pair wrap`,
    40	`pair slug`, `pair context`, `pair scrollback-render`, `pair changelog`,
    41	`pair continuation`, `pair scribe`) instead of staying as independently managed
    42	installed commands forever.
    43	
    44	As of #74, `bin/pair-go` is a development-only dispatcher skeleton built from
    45	`cmd/pair-go` and the pure `cmd/internal/dispatcher` parser. It lists the
    46	planned command families and returns explicit unsupported-command errors, but it
    47	does not launch sessions or replace `bin/pair`.
    48	
    49	Native integration layers stay native: `nvim/*.lua` remains the bundled Neovim
    50	surface and `zellij/*.kdl` remains the zellij layout/config surface. Packaging
    51	may embed those assets or install them adjacent to the binary, but the migration
    52	does not force Lua or KDL into Go.
    53	
    54	The migration is deliberately staged through issue #73 onward. Each step must be
    55	merge-safe: after any sub-issue lands, the public `pair` command, `pair-dev`,
    56	keybindings, scrollback, changelog, continuation, and review flows still work.
    57	
    58	### `bin/pair` — launcher
    59	
    60	Resolves `$PAIR_HOME` from its own real path (portable bash, no `readlink -f`), prepends `$PAIR_HOME/bin` to `$PATH` (idempotent across re-launches) so all helper scripts resolve by bare name in zellij configs and keybinds, parses argv — first positional is `$PAIR_AGENT` (default `claude`), everything after `--` is joined into `$PAIR_AGENT_ARGS`, extra positionals before `--` are an error with a usage hint, defaults `$PAIR_TAG` to the cwd basename (the create-flow prompt or `pair resume <tag>` overrides it), resolves `$PAIR_DATA_DIR` to `${XDG_DATA_HOME:-$HOME/.local/share}/pair`, runs a one-time migration of any old `~/scratch/pair-{draft,log}-*` files, and dispatches:
    61	
    62	A leading `pair resume <tag>` is recognized as a subcommand verb (alongside `list` / `help`): it skips both the picker and the name prompt, attaches if `pair-<tag>` already exists in any state, otherwise creates with that tag. When `resume` is in play, the agent is inferred from saved state on disk (`agent-<tag>` for live/recently-detached sessions; the agent embedded in the `config-<tag>-<agent>.json` filename otherwise) — so a single tag is enough to restart, regardless of which agent was originally paired with it. See "Tag-restart" below.
    63	
    64	**Decision tree.** Finds *all* detached pair-* sessions on the machine (any agent, any naming). It also surfaces **historical tags from this cwd** (#000024): tags named `<cwd-base>` or `<cwd-base>-<subproject>` whose `draft-` / `log-` sidecars in `$DATA_DIR/` were touched within the last `$PAIR_HISTORY_DAYS` (default 14) but no longer have a live session. Convention-only — operators are expected to name sessions `<cwd-base>-<subproject>` so they appear in the right cwd's picker. Then:
    65	
    66	- 0 detached + 0 historical → run create flow directly (validate agent, prompt for name, create).
    67	- ≥1 detached or ≥1 historical → fzf picker over the detached sessions, then historical rows annotated `(Nd ago, no live session)`, then a `+ new <agent> session` sentinel. Pick a detached row → attach. Pick a historical row → create-by-name (same path as `pair resume <tag>`, which re-uses any saved `draft-<tag>.md` / `config-<tag>-<agent>.json`). Pick the sentinel → fall through to create with `free_slot_tag`. `PAIR_DEBUG_HISTORY=1 pair` exits early printing the scan results — useful for sanity-checking the cwd-prefix convention on a given data dir. A historical row also gets an amber `[⏎ N queued]` badge when `queue_count_for` finds N `<digits>.md` items under `$DATA_DIR/queue-<tag>/` (the queue nvim writes), so a forgotten queue is visible before resume. The badge is a separate color segment appended after the age-tinted base row; the plain `historical_rows` copy carries the same visible text so the `fzf --ansi` return still maps back and the `${picked%%  *}` → `pair-<tag>` parse is unaffected.
    68	
    69	The agent argument doesn't filter the picker — reattach is agent-agnostic (the existing session already runs whatever it runs). The agent argument only matters for the create path: it labels the sentinel, drives the auto-suggested default name, and is the binary that gets exec'd in the new session.
    70	
    71	There is **no silent auto-attach**. Every reattach goes through the picker so the user explicitly sees what they're connecting to.
    72	
    73	Detection of attached-vs-detached uses `zellij --session NAME action list-clients`, which prints a header plus one row per connected client. Zero rows = detached.
    74	
    75	**Tag reuse & stale-EXITED residue (#67).** A pair tag *is* a workspace, and a live zellij session is globally `pair-<tag>` regardless of agent. `Alt+x` can leave a resurrect record like `pair-2 (EXITED - attach to resurrect)`; that row still shows in `list-sessions`, so the old open-coded `grep -qx` collision checks treated a fully-quit tag as occupied and blocked reuse. The single helper `session_blocks_reuse <session>` centralizes the decision (`ARCH-DRY`): an `EXITED` row is stale full-quit residue — it deletes the zellij record (`delete-session --force`) and reports the tag reusable; a running/detached row still blocks; an absent session never blocks. It is the shared gate for forced `pair resume <tag>`, the free-slot scan, and the create-flow name-prompt collision. `pair rename` keeps its own offline-only resurrectable-session contract and is *not* routed through it. (As of #69 the cmux-owner path no longer consults it — see the cmux workspace-title note below.)
    76	
    77	**Title poller (`bin/pair-title.sh`) — two surfaces.** A single always-on per-tag 60s background poller, spawned via `ensure_title_poller` on *every* entry (create, attach, restart) so a poller a host sleep/reboot/SIGKILL killed is reliably revived. Single-instance guard is identity-checked (`poller_alive` `ps`-matches the command line for this tag; pidfile `$DATA_DIR/title-pid-<tag>`; not a bare `kill -0`) so a recycled PID can't suppress the respawn. It owns two title surfaces (tested in `tests/pair-title-poller-test.sh`):
    78	
    79	- **Per-pane context meter in the zellij FRAME (#71).** Each agent pane's frame title reads `<agent> (<count>) [<cwd>]`, where `<count>` is the agent's current context-window occupancy — an absolute humanized token count (`970k`), so no model→window catalog is needed. Source of truth is the agent's own session transcript: the pure `cmd/internal/ctxmeter` reader (`ContextTokens` sums the last *real* claude `message.usage`, skipping `isSidechain`/`<synthetic>` records; codex `last_token_usage.input_tokens` of the last `token_count` event; agy none) + `Humanize`, over the path from the shared `cmd/internal/transcript` resolver (extracted from `pair-slug`, ARCH-DRY). The one-shot `cmd/pair-context <tag> <agent>` wires it (tolerant: any failure prints nothing). Each pane records `{pane_id, cwd, cwd_display}` to a single-writer `pane-<tag>-<agent>.json` at startup (`main.kdl`, beside the startup rename — dodges the 3-writer race on `config-*`); the poller loops those, calls `pair-context`, and `zellij --session pair-<tag> action rename-pane`s each frame, gated on recent activity with a per-pane unchanged-skip cache. Always-on (the frame exists with or without cmux). Carried through `pair rename` like `config-*`.
    80	

exec
/bin/zsh -lc "nl -ba go.mod | sed -n '1,40p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	module github.com/xianxu/pair
     2	
     3	go 1.26.3
     4	
     5	require (
     6		github.com/charmbracelet/ultraviolet v0.0.0-20260303162955-0b88c25f3fff
     7		github.com/charmbracelet/x/vt v0.0.0-20260510215043-e3181689be6b
     8		github.com/creack/pty v1.1.24
     9		golang.org/x/sys v0.44.0
    10		golang.org/x/term v0.43.0
    11	)
    12	
    13	require (
    14		github.com/charmbracelet/colorprofile v0.4.2 // indirect
    15		github.com/charmbracelet/x/ansi v0.11.7 // indirect
    16		github.com/charmbracelet/x/exp/ordered v0.1.0 // indirect
    17		github.com/charmbracelet/x/term v0.2.2 // indirect
    18		github.com/charmbracelet/x/termios v0.1.1 // indirect
    19		github.com/charmbracelet/x/windows v0.2.2 // indirect
    20		github.com/clipperhouse/displaywidth v0.11.0 // indirect
    21		github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
    22		github.com/lucasb-eyer/go-colorful v1.4.0 // indirect
    23		github.com/mattn/go-runewidth v0.0.23 // indirect
    24		github.com/muesli/cancelreader v0.2.2 // indirect
    25		github.com/rivo/uniseg v0.4.7 // indirect
    26		github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
    27		golang.org/x/sync v0.19.0 // indirect
    28	)

codex
The issue and plan match the broad shape: a pure `cmd/internal/dispatcher` plus a thin `cmd/pair-go` wrapper, and the atlas has a corresponding new architecture note. I’m doing one more pass for edge cases and verification signals, including whether the public launcher path changed.
exec
/bin/zsh -lc 'git diff -- 1199bb427dfc0d7c7b87dcc1115ce19907c4d804..HEAD -- bin/pair' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
exec
/bin/zsh -lc "rg -n \"pair-go|cmd/internal/dispatcher|dispatcher skeleton|GO_BINS|bin/pair-go\" . --glob '"'!workshop/history/**'"' --glob '"'!bin/pair-go'"'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
./atlas/architecture.md:13:bin/pair-go                  # opt-in Go dispatcher skeleton; public launcher remains bin/pair
./atlas/architecture.md:44:As of #74, `bin/pair-go` is a development-only dispatcher skeleton built from
./atlas/architecture.md:45:`cmd/pair-go` and the pure `cmd/internal/dispatcher` parser. It lists the
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:5:**Goal:** Add an opt-in Go dispatcher skeleton that establishes Pair's future primary CLI shape without changing the public `bin/pair` launcher.
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:7:**Architecture:** Create a pure `cmd/internal/dispatcher` package for argv parsing, help rendering, planned command metadata, and unsupported-command errors (`ARCH-PURE`). Add `cmd/pair-go/main.go` as a thin IO shell that calls the dispatcher and writes output/exit codes. Wire `pair-go` into the existing `GO_BINS` build pattern while leaving `bin/pair`, zellij layouts, and helper command names unchanged (`ARCH-DRY`, `ARCH-PURPOSE`).
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:19:| `CommandFamily` | `cmd/internal/dispatcher/dispatcher.go` | new |
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:20:| `Result` | `cmd/internal/dispatcher/dispatcher.go` | new |
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:21:| `Dispatch` | `cmd/internal/dispatcher/dispatcher.go` | new |
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:22:| `Help` | `cmd/internal/dispatcher/dispatcher.go` | new |
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:48:| `pair-go main` | `cmd/pair-go/main.go` | new | `os.Args`, stdout/stderr, process exit |
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:49:| `pair-go build target` | `Makefile.local` | modified | `go build` into `bin/pair-go` |
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:52:- **pair-go main** — minimal binary entrypoint: call `dispatcher.Dispatch(os.Args[1:])`, print streams, exit.
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:56:- **pair-go build target** — follow the existing `GO_BINS` and per-binary recipe convention so `make build` compiles the skeleton.
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:60:- **architecture note** — record that `pair-go` is the opt-in dispatcher skeleton and not the public launcher.
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:69:- Create: `cmd/internal/dispatcher/dispatcher_test.go`
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:70:- Later create: `cmd/internal/dispatcher/dispatcher.go`
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:79:- an unknown command returns exit `2` and suggests `pair-go help`.
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:83:Run: `go test ./cmd/internal/dispatcher -count=1`
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:98:Run: `go test ./cmd/internal/dispatcher -count=1`
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:105:- Create: `cmd/pair-go/main.go`
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:112:- [x] **Step 2: Implement `cmd/pair-go/main.go`**
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:116:- [x] **Step 3: Wire `pair-go` into `Makefile.local`**
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:118:Append `pair-go` to `GO_BINS`, add the `.PHONY` alias, and add:
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:121:$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/dispatcher/dispatcher.go go.mod
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:122:	go build -o $@ ./cmd/pair-go
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:127:Run: `make pair-go`
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:129:Expected: `bin/pair-go` is created.
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:139:Add `bin/pair-go` to the piece list as the opt-in Go dispatcher skeleton and mention it in the packaging migration target as development-only for this issue.
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:144:- `go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
./workshop/plans/000074-go-dispatcher-skeleton-plan.md:145:- `make pair-go`
./workshop/plans/000071-context-meter-pane-frame-plan.md:493:- Modify: `Makefile.local` (GO_BINS line ~29; alias block ~38-46; build rules ~220)
./workshop/plans/000071-context-meter-pane-frame-plan.md:495:- [ ] **Step 1: Add to `GO_BINS`** (line 29): append ` pair-context`.
./workshop/plans/000050-pair-continue-plan.md:231:- [ ] **Step 5 — wire build/test**: add `pair-continuation` to `GO_BINS` (`Makefile.local:29`) + a build recipe whose target is `./cmd/pair-continuation` (so all `.go` files compile — mirror the **multi-file `pair-slug`** recipe, not the single-file `pair-scrollback-render` one) + an alias. Run `make build && make test`.
./workshop/issues/000074-go-dispatcher-skeleton.md:13:# pair Go dispatcher skeleton
./workshop/issues/000074-go-dispatcher-skeleton.md:21:Add a Go dispatcher skeleton that can host Pair subcommands behind an explicit development path. It should establish command parsing, help text shape, version/build metadata if needed, and an internal routing pattern for future subcommands.
./workshop/issues/000074-go-dispatcher-skeleton.md:69:- 2026-06-29: closed — go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; bash tests/pair-continue-test.sh; git diff -- bin/pair empty
./workshop/issues/000074-go-dispatcher-skeleton.md:71:Claimed and entered planning. Design uses a new opt-in `pair-go` binary plus pure `cmd/internal/dispatcher` package so command parsing/help/error behavior is unit-testable (`ARCH-PURE`) and `bin/pair` remains unchanged while the skeleton fulfills the issue purpose (`ARCH-PURPOSE`). Durable plan: `workshop/plans/000074-go-dispatcher-skeleton-plan.md`.
./workshop/issues/000074-go-dispatcher-skeleton.md:73:Implemented `cmd/internal/dispatcher` and `cmd/pair-go` behind a non-public `pair-go` build target. Wrapper behavior is covered by `cmd/pair-go/main_test.go` rather than subprocess tests because the pure dispatcher tests cover parsing/help/error semantics and the wrapper only writes the returned streams/exit code. Verified with `env GOCACHE=/private/tmp/pair-go-cache GOMODCACHE=/private/tmp/pair-gomod-cache go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1`, `env GOCACHE=/private/tmp/pair-go-cache GOMODCACHE=/private/tmp/pair-gomod-cache make -B pair-go`, `env GOCACHE=/private/tmp/pair-go-cache GOMODCACHE=/private/tmp/pair-gomod-cache go test ./... -count=1`, `bash tests/pair-continue-test.sh`, and `git diff -- bin/pair` (empty).
./Makefile.local:4:.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
./Makefile.local:11:# to GO_BINS, add the per-binary recipe stanza. `make build` and
./Makefile.local:14:#   make build      → compile every binary in GO_BINS into ./bin/
./Makefile.local:29:GO_BINS     := pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-go
./Makefile.local:31:build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
./Makefile.local:35:	@for b in $(GO_BINS); do \
./Makefile.local:48:pair-go:                $(BIN_DIR)/pair-go
./Makefile.local:242:# pair-go — development-only dispatcher skeleton for the future primary Go CLI.
./Makefile.local:244:$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/dispatcher/dispatcher.go go.mod
./Makefile.local:245:	go build -o $@ ./cmd/pair-go
./workshop/plans/000053-changelog-plan.md:505:- [ ] **Step 5: Add the binary to the build.** Edit `Makefile.local` (the top-level `Makefile` is a symlink to ariadne's): (a) append `pair-changelog` to `GO_BINS` (~line 29) and (b) add the per-binary recipe + alias stanza mirroring `pair-slug` (~lines 41–45 and 139–140; the in-file comment at lines 10–12 documents the 2-step recipe). Confirm the pattern first with `grep -n pair-slug Makefile.local`. Run `make pair-changelog` → builds to `bin/`.
./cmd/internal/dispatcher/dispatcher.go:8:const programName = "pair-go"
./cmd/internal/dispatcher/dispatcher.go:49:			Stdout:   "pair-go dispatcher skeleton\npublic launcher: bin/pair\n",
./cmd/internal/dispatcher/dispatcher.go:71:	b.WriteString("Development dispatcher skeleton. Public sessions still start through bin/pair.\n\n")
./cmd/internal/dispatcher/dispatcher.go:78:	b.WriteString("  version           show dispatcher skeleton metadata\n")
./cmd/internal/dispatcher/dispatcher_test.go:19:				"Usage: pair-go <command> [args]",
./cmd/internal/dispatcher/dispatcher_test.go:41:	for _, want := range []string{"pair-go", "dispatcher skeleton", "public launcher: bin/pair"} {
./cmd/internal/dispatcher/dispatcher_test.go:56:	for _, want := range []string{"wrap", "planned", "not implemented", "pair-go help"} {
./cmd/internal/dispatcher/dispatcher_test.go:71:	for _, want := range []string{"unknown command", "frobnicate", "pair-go help"} {
./cmd/pair-go/main_test.go:15:	if !strings.Contains(stdout.String(), "Usage: pair-go <command> [args]") {
./cmd/pair-go/main.go:1:// pair-go is a development-only dispatcher skeleton for the future primary Go
./cmd/pair-go/main.go:9:	"github.com/xianxu/pair/cmd/internal/dispatcher"

exec
/bin/zsh -lc 'git diff --check 1199bb427dfc0d7c7b87dcc1115ce19907c4d804..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
exec
/bin/zsh -lc 'env GOCACHE=/private/tmp/pair-review-gocache GOMODCACHE=/private/tmp/pair-review-gomodcache go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1' in /Users/xianxu/workspace/pair
 succeeded in 2828ms:
ok  	github.com/xianxu/pair/cmd/internal/dispatcher	0.332s
ok  	github.com/xianxu/pair/cmd/pair-go	0.197s

codex
Focused tests for the new packages pass, and `git diff -- bin/pair` is empty. The remaining question is whether any small implementation choices are worth gate findings, especially around future API stability and IO error handling in the wrapper.
exec
/bin/zsh -lc "rg -n \"io\\.WriteString|fmt\\.Fprint|fmt\\.Fprintln|os\\.Exit\\(run|func run\\(\" cmd | head -80" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
cmd/pair-scrollback-render/main.go:393:		fmt.Fprintf(os.Stderr, "usage: %s [--plain] [--max-lines N] [--with-timestamps] <raw> <events.jsonl> <out>\n", os.Args[0])
cmd/pair-scrollback-render/main.go:405:		fmt.Fprintf(os.Stderr, "scrollback-render: %v\n", err)
cmd/pair-scrollback-render/render_test.go:33:			fmt.Fprintf(&raw, "line %02d\r\n", i)
cmd/pair-scrollback-render/render_test.go:35:			fmt.Fprintf(&raw, "line %02d", i) // no trailing CRLF → no extra scroll
cmd/pair-scrollback-render/render_test.go:132:		fmt.Fprintf(&raw, "line %02d\r\n", i)
cmd/pair-scrollback-render/timestamps_test.go:88:		fmt.Fprintf(&raw, "line %02d\r\n", i)
cmd/pair-wrap/main.go:478:	fmt.Fprintf(f, "[%s] %s: %q\n", time.Now().Format("15:04:05"), label, ctx)
cmd/pair-wrap/main.go:835:		fmt.Fprintf(f, "%s\t%d\t%s\n", e.color, e.count, e.text)
cmd/pair-wrap/main.go:1745:		fmt.Fprintf(os.Stderr, "pair-wrap: %v\n", err)
cmd/pair-wrap/main.go:1750:func run() error {
cmd/pair-continuation/main.go:32:		fmt.Fprintf(os.Stderr, "pair-continuation: %v\n", err)
cmd/pair-continuation/main.go:45:func run(a runArgs, now func() time.Time, stdin io.Reader, stdout io.Writer) error {
cmd/pair-continuation/main.go:108:		fmt.Fprintf(os.Stderr, "pair-continuation: push failed (commit kept locally): %v\n%s\n", err, out)
cmd/pair-continuation/main.go:110:	fmt.Fprintln(stdout, abs)
cmd/pair-continuation/continuation.go:36:	fmt.Fprintf(&b, "slug: %s\n", f.Slug)
cmd/pair-continuation/continuation.go:37:	fmt.Fprintf(&b, "agent: %s\n", f.Agent)
cmd/pair-continuation/continuation.go:39:		fmt.Fprintf(&b, "session_id: %s\n", f.SessionID)
cmd/pair-continuation/continuation.go:41:	fmt.Fprintf(&b, "created: %s\n", f.Created.Format("2006-01-02T15:04:05"))
cmd/pair-continuation/continuation.go:43:		fmt.Fprintf(&b, "supersedes: %s\n", f.Supersedes)
cmd/pair-continuation/continuation.go:46:		fmt.Fprintf(&b, "branch: %s\n", f.Branch)
cmd/pair-continuation/continuation.go:49:		fmt.Fprintf(&b, "worktree: %s\n", f.Worktree)
cmd/pair-continuation/continuation.go:51:	fmt.Fprintf(&b, "issues: [%s]\n", strings.Join(f.Issues, ", "))
cmd/internal/dispatcher/dispatcher.go:70:	fmt.Fprintf(&b, "Usage: %s <command> [args]\n\n", program)
cmd/internal/dispatcher/dispatcher.go:74:		fmt.Fprintf(&b, "  %-17s %s (%s; not implemented in this skeleton)\n", family.Name, family.Summary, family.Status)
cmd/pair-slug/main.go:58:	fmt.Fprintf(f, format+"\n", a...)
cmd/pair-scribe/main.go:39:		fmt.Fprintf(os.Stderr, "usage: %s -log PATH -- CMD [ARGS...]\n", os.Args[0])
cmd/pair-scribe/main.go:138:	fmt.Fprintf(os.Stderr, "scribe: "+format+"\n", a...)
cmd/pair-changelog/main_test.go:70:func run(t *testing.T, bin, cleaned, priorLog, priorAnchor string) (log, anchor string) {
cmd/pair-go/main.go:13:	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
cmd/pair-go/main.go:16:func run(args []string, stdout, stderr io.Writer) int {
cmd/pair-go/main.go:19:		_, _ = io.WriteString(stdout, res.Stdout)
cmd/pair-go/main.go:22:		_, _ = io.WriteString(stderr, res.Stderr)
cmd/pair-changelog/main.go:38:	fmt.Fprintf(os.Stderr, "pair-changelog: "+format+"\n", a...)
cmd/pair-changelog/main.go:91:		fmt.Fprintln(os.Stderr, "pair-changelog: up to date (no new turn)")
cmd/pair-changelog/main.go:120:				fmt.Fprintf(os.Stderr, "pair-changelog: distilling batch %d/%d (%d lines)\n", i+1, len(chunks), len(chunk))
cmd/pair-changelog/main.go:122:				fmt.Fprintf(os.Stderr, "pair-changelog: distilling %d lines\n", len(chunk))
cmd/pair-changelog/main.go:211:	fmt.Fprintf(&b, "turns:%d\n", turns)
cmd/pair-changelog/e2e_test.go:51:		fmt.Fprintf(&raw, "line %02d\r\n", i)

exec
/bin/zsh -lc "rg -n \"Families\\(|CommandFamily|Result struct|Dispatch\\(\" cmd/internal cmd/pair-*" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
cmd/pair-go/main.go:17:	res := dispatcher.Dispatch(args)
cmd/pair-slug/slug_test.go:233:func TestParseTranscriptDispatch(t *testing.T) {
cmd/pair-changelog/distill.go:163:type LocateResult struct {
cmd/internal/dispatcher/dispatcher.go:10:// CommandFamily names a future Pair CLI surface without claiming it works yet.
cmd/internal/dispatcher/dispatcher.go:11:type CommandFamily struct {
cmd/internal/dispatcher/dispatcher.go:18:type Result struct {
cmd/internal/dispatcher/dispatcher.go:25:func Families() []CommandFamily {
cmd/internal/dispatcher/dispatcher.go:26:	return []CommandFamily{
cmd/internal/dispatcher/dispatcher.go:39:func Dispatch(args []string) Result {
cmd/internal/dispatcher/dispatcher.go:73:	for _, family := range Families() {
cmd/internal/dispatcher/dispatcher.go:82:func familyByName(name string) (CommandFamily, bool) {
cmd/internal/dispatcher/dispatcher.go:83:	for _, family := range Families() {
cmd/internal/dispatcher/dispatcher.go:88:	return CommandFamily{}, false
cmd/internal/dispatcher/dispatcher_test.go:11:			res := Dispatch(args)
cmd/internal/dispatcher/dispatcher_test.go:34:	res := Dispatch([]string{"version"})
cmd/internal/dispatcher/dispatcher_test.go:49:	res := Dispatch([]string{"wrap"})
cmd/internal/dispatcher/dispatcher_test.go:64:	res := Dispatch([]string{"frobnicate"})

exec
/bin/zsh -lc 'git status --short' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 M workshop/issues/000074-go-dispatcher-skeleton.md

exec
/bin/zsh -lc 'git diff -- workshop/issues/000074-go-dispatcher-skeleton.md' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
diff --git a/workshop/issues/000074-go-dispatcher-skeleton.md b/workshop/issues/000074-go-dispatcher-skeleton.md
index 770423d..8e6320e 100644
--- a/workshop/issues/000074-go-dispatcher-skeleton.md
+++ b/workshop/issues/000074-go-dispatcher-skeleton.md
@@ -1,12 +1,13 @@
 ---
 id: 000074
-status: working
+status: done
 deps: [000073]
 github_issue:
 created: 2026-06-26
 updated: 2026-06-29
 estimate_hours: 1.39
 started: 2026-06-29T17:00:26-07:00
+actual_hours: 0.14
 ---
 
 # pair Go dispatcher skeleton
@@ -65,6 +66,7 @@ Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `
 Created from #72 as the safe first code-bearing step toward one primary Go command.
 
 ### 2026-06-29
+- 2026-06-29: closed — go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; bash tests/pair-continue-test.sh; git diff -- bin/pair empty
 
 Claimed and entered planning. Design uses a new opt-in `pair-go` binary plus pure `cmd/internal/dispatcher` package so command parsing/help/error behavior is unit-testable (`ARCH-PURE`) and `bin/pair` remains unchanged while the skeleton fulfills the issue purpose (`ARCH-PURPOSE`). Durable plan: `workshop/plans/000074-go-dispatcher-skeleton-plan.md`.
 

codex
One local issue file has unstaged close-state edits, so I’m separating that from the actual `HEAD` review range. For the verdict I’ll use `1199bb4..HEAD` plus the issue/plan content committed at `HEAD`, not the uncommitted close bookkeeping.
exec
/bin/zsh -lc "git show HEAD:workshop/issues/000074-go-dispatcher-skeleton.md | nl -ba | sed -n '1,140p'" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "git show HEAD:workshop/plans/000074-go-dispatcher-skeleton-plan.md | nl -ba | sed -n '1,190p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	# Pair Go Dispatcher Skeleton Implementation Plan
     2	
     3	> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.
     4	
     5	**Goal:** Add an opt-in Go dispatcher skeleton that establishes Pair's future primary CLI shape without changing the public `bin/pair` launcher.
     6	
     7	**Architecture:** Create a pure `cmd/internal/dispatcher` package for argv parsing, help rendering, planned command metadata, and unsupported-command errors (`ARCH-PURE`). Add `cmd/pair-go/main.go` as a thin IO shell that calls the dispatcher and writes output/exit codes. Wire `pair-go` into the existing `GO_BINS` build pattern while leaving `bin/pair`, zellij layouts, and helper command names unchanged (`ARCH-DRY`, `ARCH-PURPOSE`).
     8	
     9	**Tech Stack:** Go 1.26, standard library only, existing `Makefile.local` build conventions.
    10	
    11	---
    12	
    13	## Core Concepts
    14	
    15	### Pure Entities
    16	
    17	| Name | Lives in | Status |
    18	|------|----------|--------|
    19	| `CommandFamily` | `cmd/internal/dispatcher/dispatcher.go` | new |
    20	| `Result` | `cmd/internal/dispatcher/dispatcher.go` | new |
    21	| `Dispatch` | `cmd/internal/dispatcher/dispatcher.go` | new |
    22	| `Help` | `cmd/internal/dispatcher/dispatcher.go` | new |
    23	
    24	- **CommandFamily** — metadata for planned Pair command families that the skeleton can list without implementing.
    25	  - **Relationships:** N:1 with `Help`; `Help` renders all families from one slice.
    26	  - **DRY rationale:** One source for names/descriptions/status text instead of duplicating planned command lists between help and dispatch validation.
    27	  - **Future extensions:** Add handler wiring when later issues port a family.
    28	
    29	- **Result** — pure dispatch outcome: stdout, stderr, and process exit code.
    30	  - **Relationships:** 1:1 with a dispatcher invocation; consumed by the IO shell.
    31	  - **DRY rationale:** Keeps exit behavior testable without spawning a subprocess.
    32	  - **Future extensions:** Can grow a handler enum or execution callback once subcommands become real.
    33	
    34	- **Dispatch** — pure argv parser and router for `help`, `--help`, `version`, and unsupported commands.
    35	  - **Relationships:** Owns one invocation's decision; reads `CommandFamily` metadata; returns `Result`.
    36	  - **DRY rationale:** Centralizes command parsing before behavior begins migrating out of shell scripts.
    37	  - **Future extensions:** Replace unsupported-command outcomes with real handlers one family at a time.
    38	
    39	- **Help** — pure renderer for the dev CLI usage and planned command families.
    40	  - **Relationships:** Reads `CommandFamily`; used by `Dispatch`.
    41	  - **DRY rationale:** Help text and tests derive from the same command-family metadata.
    42	  - **Future extensions:** Add richer command-specific help when handlers exist.
    43	
    44	### Integration Points
    45	
    46	| Name | Lives in | Status | Wraps |
    47	|------|----------|--------|-------|
    48	| `pair-go main` | `cmd/pair-go/main.go` | new | `os.Args`, stdout/stderr, process exit |
    49	| `pair-go build target` | `Makefile.local` | modified | `go build` into `bin/pair-go` |
    50	| `architecture note` | `atlas/architecture.md` | modified | repo map documentation |
    51	
    52	- **pair-go main** — minimal binary entrypoint: call `dispatcher.Dispatch(os.Args[1:])`, print streams, exit.
    53	  - **Injected into:** No pure entity; it consumes the pure `Result`.
    54	  - **Future extensions:** Later issues can replace result-only routing with handler execution behind the same package.
    55	
    56	- **pair-go build target** — follow the existing `GO_BINS` and per-binary recipe convention so `make build` compiles the skeleton.
    57	  - **Injected into:** Build/install flow only.
    58	  - **Future extensions:** The public `pair` entrypoint switch happens in a later issue, not here.
    59	
    60	- **architecture note** — record that `pair-go` is the opt-in dispatcher skeleton and not the public launcher.
    61	  - **Injected into:** Atlas readers and future migration issues.
    62	  - **Future extensions:** Update when the public entrypoint changes.
    63	
    64	## Chunk 1: Dispatcher Skeleton
    65	
    66	### Task 1: Add pure dispatch tests
    67	
    68	**Files:**
    69	- Create: `cmd/internal/dispatcher/dispatcher_test.go`
    70	- Later create: `cmd/internal/dispatcher/dispatcher.go`
    71	
    72	- [x] **Step 1: Write failing tests for help, version, and unsupported commands**
    73	
    74	Cover:
    75	- empty argv and `help` return exit `0` with usage and planned families.
    76	- `--help` and `-h` mirror help.
    77	- `version` returns deterministic dev metadata without pretending to be the shell launcher.
    78	- a planned-but-unimplemented command like `wrap` returns exit `2` and an unsupported message.
    79	- an unknown command returns exit `2` and suggests `pair-go help`.
    80	
    81	- [x] **Step 2: Run the package test and verify it fails**
    82	
    83	Run: `go test ./cmd/internal/dispatcher -count=1`
    84	
    85	Expected: fail because `dispatcher.go` does not exist yet.
    86	
    87	- [x] **Step 3: Implement the minimal dispatcher**
    88	
    89	Create:
    90	- `CommandFamily` with `Name`, `Summary`, and `Status`.
    91	- `Families()` returning planned command families: `launch`, `wrap`, `slug`, `context`, `scrollback-render`, `changelog`, `continuation`, `scribe`.
    92	- `Result` with `Stdout`, `Stderr`, `ExitCode`.
    93	- `Dispatch(args []string) Result`.
    94	- `Help(program string) string`.
    95	
    96	- [x] **Step 4: Run dispatcher tests and verify they pass**
    97	
    98	Run: `go test ./cmd/internal/dispatcher -count=1`
    99	
   100	Expected: pass.
   101	
   102	### Task 2: Add the opt-in binary wrapper and build target
   103	
   104	**Files:**
   105	- Create: `cmd/pair-go/main.go`
   106	- Modify: `Makefile.local`
   107	
   108	- [x] **Step 1: Write a failing wrapper smoke test if needed**
   109	
   110	If the pure tests cover all behavior, keep `main.go` untested directly and rely on `go test ./...` plus build verification. Do not introduce subprocess tests unless pure coverage misses behavior.
   111	
   112	- [x] **Step 2: Implement `cmd/pair-go/main.go`**
   113	
   114	Main reads `os.Args[1:]`, delegates to `dispatcher.Dispatch`, writes stdout/stderr to the matching streams, and exits with the returned code.
   115	
   116	- [x] **Step 3: Wire `pair-go` into `Makefile.local`**
   117	
   118	Append `pair-go` to `GO_BINS`, add the `.PHONY` alias, and add:
   119	
   120	```make
   121	$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/dispatcher/dispatcher.go go.mod
   122		go build -o $@ ./cmd/pair-go
   123	```
   124	
   125	- [x] **Step 4: Build the new binary**
   126	
   127	Run: `make pair-go`
   128	
   129	Expected: `bin/pair-go` is created.
   130	
   131	### Task 3: Document and verify non-disruption
   132	
   133	**Files:**
   134	- Modify: `atlas/architecture.md`
   135	- Modify: `workshop/issues/000074-go-dispatcher-skeleton.md`
   136	
   137	- [x] **Step 1: Update atlas**
   138	
   139	Add `bin/pair-go` to the piece list as the opt-in Go dispatcher skeleton and mention it in the packaging migration target as development-only for this issue.
   140	
   141	- [x] **Step 2: Run focused verification**
   142	
   143	Run:
   144	- `go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
   145	- `make pair-go`
   146	- `go test ./... -count=1`
   147	- `bash tests/pair-continue-test.sh`
   148	
   149	Expected: all pass. The shell launcher remains the public entrypoint.
   150	
   151	- [x] **Step 3: Update issue checkboxes and log**
   152	
   153	Mark completed plan and done-when boxes, then close through `sdlc close --issue 74 --verified '<evidence>'` with atlas evidence. This plan expects an atlas update, so prefer satisfying the atlas gate.

 succeeded in 0ms:
     1	---
     2	id: 000074
     3	status: working
     4	deps: [000073]
     5	github_issue:
     6	created: 2026-06-26
     7	updated: 2026-06-29
     8	estimate_hours: 1.39
     9	started: 2026-06-29T17:00:26-07:00
    10	---
    11	
    12	# pair Go dispatcher skeleton
    13	
    14	## Problem
    15	
    16	The target architecture needs a primary Go command, but switching the public launcher immediately would be too risky. The first code step should introduce the dispatch shape without changing user-visible behavior.
    17	
    18	## Spec
    19	
    20	Add a Go dispatcher skeleton that can host Pair subcommands behind an explicit development path. It should establish command parsing, help text shape, version/build metadata if needed, and an internal routing pattern for future subcommands.
    21	
    22	The existing `bin/pair` shell launcher remains the public entrypoint for this issue. Any new Go command must be opt-in, for example a new built binary or a hidden/dev invocation, so this can merge without affecting normal sessions.
    23	
    24	Design constraints:
    25	
    26	- Reuse existing package structure where possible (`ARCH-DRY`).
    27	- Keep command parsing and dispatch decision logic pure enough to unit-test (`ARCH-PURE`).
    28	- Do not port launcher behavior yet. This issue is only the skeleton.
    29	
    30	## Done when
    31	
    32	- [x] A Go dispatcher command exists and builds in the normal `make build` flow or an explicitly documented dev target.
    33	- [x] Dispatcher help lists the planned command families without claiming unsupported behavior works.
    34	- [x] Public `bin/pair` behavior is unchanged.
    35	- [x] Tests cover dispatch parsing and unsupported-command errors.
    36	- [x] Pair remains usable after merge through the existing `pair` entrypoint.
    37	
    38	## Estimate
    39	
    40	```estimate
    41	model: estimate-logic-v3.1
    42	familiarity: 1.0
    43	item: skill-or-dispatcher design=0.30 impl=0.25
    44	item: smaller-go-module design=0.20 impl=0.20
    45	item: atlas-docs design=0.10 impl=0.05
    46	item: milestone-review design=0.00 impl=0.20
    47	design-buffer: 0.15
    48	total: 1.39
    49	```
    50	
    51	Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.
    52	
    53	## Plan
    54	
    55	- [x] Choose the command location/name based on #73 inventory.
    56	- [x] Add pure dispatch parsing tests.
    57	- [x] Add the skeleton implementation.
    58	- [x] Wire build/install only in a non-disruptive way.
    59	- [x] Verify existing Pair flows still use the shell launcher.
    60	
    61	## Log
    62	
    63	### 2026-06-26
    64	
    65	Created from #72 as the safe first code-bearing step toward one primary Go command.
    66	
    67	### 2026-06-29
    68	
    69	Claimed and entered planning. Design uses a new opt-in `pair-go` binary plus pure `cmd/internal/dispatcher` package so command parsing/help/error behavior is unit-testable (`ARCH-PURE`) and `bin/pair` remains unchanged while the skeleton fulfills the issue purpose (`ARCH-PURPOSE`). Durable plan: `workshop/plans/000074-go-dispatcher-skeleton-plan.md`.
    70	
    71	Implemented `cmd/internal/dispatcher` and `cmd/pair-go` behind a non-public `pair-go` build target. Wrapper behavior is covered by `cmd/pair-go/main_test.go` rather than subprocess tests because the pure dispatcher tests cover parsing/help/error semantics and the wrapper only writes the returned streams/exit code. Verified with `env GOCACHE=/private/tmp/pair-go-cache GOMODCACHE=/private/tmp/pair-gomod-cache go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1`, `env GOCACHE=/private/tmp/pair-go-cache GOMODCACHE=/private/tmp/pair-gomod-cache make -B pair-go`, `env GOCACHE=/private/tmp/pair-go-cache GOMODCACHE=/private/tmp/pair-gomod-cache go test ./... -count=1`, `bash tests/pair-continue-test.sh`, and `git diff -- bin/pair` (empty).

exec
/bin/zsh -lc 'git diff --stat 1199bb427dfc0d7c7b87dcc1115ce19907c4d804..HEAD -- workshop/issues/000074-go-dispatcher-skeleton.md workshop/plans/000074-go-dispatcher-skeleton-plan.md' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 workshop/issues/000074-go-dispatcher-skeleton.md   |  43 ++++--
 .../plans/000074-go-dispatcher-skeleton-plan.md    | 153 +++++++++++++++++++++
 2 files changed, 185 insertions(+), 11 deletions(-)

codex
VERDICT: SHIP (confidence: high)

The boundary delivers the stated skeleton without changing `bin/pair`: `pair-go` is opt-in, help/version/unsupported-command behavior is explicit, and the parser is separated from IO. I found no blocking or non-blocking findings.

1. Strengths:
- `cmd/internal/dispatcher/dispatcher.go:39` keeps dispatch pure and returns a simple `Result`, matching the plan’s `ARCH-PURE` design.
- `cmd/pair-go/main.go:16` is a thin IO wrapper around the dispatcher.
- `Makefile.local:29` and `Makefile.local:244` wire `pair-go` into the existing Go binary pattern.
- `atlas/architecture.md:44` documents the new surface and clearly says it does not replace `bin/pair`.

2. Critical findings:
- None.

3. Important findings:
- None.

4. Minor findings:
- None.

5. Test coverage notes:
- Focused tests pass: `go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1`.
- `git diff -- bin/pair` is empty for `1199bb4..HEAD`.
- `git diff --check 1199bb4..HEAD` is clean.

6. Architectural notes for upcoming work:
- `ARCH-DRY`: Pass. Planned families are centralized in `Families()` and reused by help/dispatch lookup.
- `ARCH-PURE`: Pass. Command behavior is unit-tested without subprocesses or filesystem IO.
- `ARCH-PURPOSE`: Pass. The change establishes the future dispatcher shape while deliberately leaving the public shell launcher untouched.

7. Plan revision recommendations:
- None.
tokens used
70,234
VERDICT: SHIP (confidence: high)

The boundary delivers the stated skeleton without changing `bin/pair`: `pair-go` is opt-in, help/version/unsupported-command behavior is explicit, and the parser is separated from IO. I found no blocking or non-blocking findings.

1. Strengths:
- `cmd/internal/dispatcher/dispatcher.go:39` keeps dispatch pure and returns a simple `Result`, matching the plan’s `ARCH-PURE` design.
- `cmd/pair-go/main.go:16` is a thin IO wrapper around the dispatcher.
- `Makefile.local:29` and `Makefile.local:244` wire `pair-go` into the existing Go binary pattern.
- `atlas/architecture.md:44` documents the new surface and clearly says it does not replace `bin/pair`.

2. Critical findings:
- None.

3. Important findings:
- None.

4. Minor findings:
- None.

5. Test coverage notes:
- Focused tests pass: `go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1`.
- `git diff -- bin/pair` is empty for `1199bb4..HEAD`.
- `git diff --check 1199bb4..HEAD` is clean.

6. Architectural notes for upcoming work:
- `ARCH-DRY`: Pass. Planned families are centralized in `Families()` and reused by help/dispatch lookup.
- `ARCH-PURE`: Pass. Command behavior is unit-tested without subprocesses or filesystem IO.
- `ARCH-PURPOSE`: Pass. The change establishes the future dispatcher shape while deliberately leaving the public shell launcher untouched.

7. Plan revision recommendations:
- None.
