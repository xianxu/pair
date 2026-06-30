# Boundary Review — pair#75 (whole-issue close)

| field | value |
|-------|-------|
| issue | 75 — pair Go launcher prototype |
| repo | pair |
| issue file | workshop/issues/000075-go-launcher-prototype.md |
| boundary | whole-issue close |
| milestone | — |
| window | e89e2cea484ab8ca7ec9ca518fd2bf04ba38a8fe..HEAD |
| command | sdlc close --issue 75 |
| reviewer | codex |
| timestamp | 2026-06-29T22:47:06-07:00 |
| verdict | REWORK |

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
session id: 019f170f-014b-7e70-86fd-33ec2c75a26c
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **pair** repository.

- repository: pair   (root: /Users/xianxu/workspace/pair)
- issue:      pair#75   (file: workshop/issues/000075-go-launcher-prototype.md)
- window:     Base: e89e2cea484ab8ca7ec9ca518fd2bf04ba38a8fe   Head: HEAD

Review the **pair** repo and its tracker — the ariadne base-layer repo itself (changes here propagate to dependent repos). Do not assume any
other repository or apply another repo's conventions.

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
diff --git a/atlas/architecture.md b/atlas/architecture.md
index f9d58a3..a536fc7 100644
--- a/atlas/architecture.md
+++ b/atlas/architecture.md
@@ -45,10 +45,15 @@ internal subcommands or dispatch modes behind that primary binary (`pair wrap`,
 `pair continuation`, `pair scribe`) instead of staying as independently managed
 installed commands forever.
 
-As of #74, `bin/pair-go` is a development-only dispatcher skeleton built from
-`cmd/pair-go` and the pure `cmd/internal/dispatcher` parser. It lists the
-planned command families and returns explicit unsupported-command errors, but it
-does not launch sessions or replace `bin/pair`.
+As of #75, `bin/pair-go` is a development-only dispatcher with a guarded
+`pair-go launch` decision-phase prototype. The public launcher remains
+`bin/pair`; `pair-go launch` parses the launcher subset, resolves the Pair data
+dir, snapshots fakeable zellij/history state, and computes create/attach/picker
+decisions through the pure `cmd/internal/launcher` core. It then stops with an
+explicit prototype message before invoking zellij. Real zellij lifecycle,
+prompt/fzf UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation,
+rename, config/session migration, and title-poller behavior remain shell-owned
+until later migration issues.
 
 Native integration layers stay native: `nvim/*.lua` remains the bundled Neovim
 surface and `zellij/*.kdl` remains the zellij layout/config surface. Packaging
diff --git a/atlas/go-migration-inventory.md b/atlas/go-migration-inventory.md
index f671fbf..3de3bb2 100644
--- a/atlas/go-migration-inventory.md
+++ b/atlas/go-migration-inventory.md
@@ -47,7 +47,7 @@ Priority is packaging impact first, then reliability/testability:
 
 | Artifact | Type | Callers | Runtime contract | Files/env | Disposition | Priority |
 |---|---|---|---|---|---|---|
-| `bin/pair` | Bash public launcher | user shell, `bin/pair-dev`, restart re-exec, tests | Parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. | Exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. | go-entrypoint with `bin/pair` retained as compat fallback through #77 | P0 |
+| `bin/pair` / `cmd/internal/launcher` | Bash public launcher plus guarded Go decision prototype | user shell, `bin/pair-dev`, restart re-exec, tests, development-only `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch` (#75) parses a documented subset, snapshots zellij/history state, prints the decision-phase prototype action, and exits before mutating zellij. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/launcher` resolves data-dir inputs and reads zellij/history through fakeable seams. | go-entrypoint with `bin/pair` retained as compat fallback through #77; #75 owns decision-phase only, while real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
 | `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling `pair`. | Resolves symlinks; depends on `bin/pair` and `bin/lib/dev-rebuild.sh`. | compat-shim, likely stays as developer wrapper until Go entrypoint has dev mode | P1 |
 | `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
 | `zellij/layouts/main.kdl` | zellij native asset | `bin/pair` via `zellij --new-session-with-layout` | Defines agent and draft panes; shell expands Pair env at pane start. | Calls `pair-wrap`; calls `nvim -u "$PAIR_HOME/nvim/init.lua"`; writes `pane-<tag>-<agent>.json`; writes draft nvim pid file. | native-asset, packaged adjacent/embedded | P0 |
diff --git a/cmd/internal/dispatcher/dispatcher.go b/cmd/internal/dispatcher/dispatcher.go
index dddfe46..1b9710a 100644
--- a/cmd/internal/dispatcher/dispatcher.go
+++ b/cmd/internal/dispatcher/dispatcher.go
@@ -2,7 +2,11 @@ package dispatcher
 
 import (
 	"fmt"
+	"os"
 	"strings"
+	"time"
+
+	"github.com/xianxu/pair/cmd/internal/launcher"
 )
 
 const programName = "pair-go"
@@ -49,6 +53,8 @@ func Dispatch(args []string) Result {
 			Stdout:   "pair-go dispatcher skeleton\npublic launcher: bin/pair\n",
 			ExitCode: 0,
 		}
+	case "launch":
+		return DispatchWithLauncherRuntime(args, osLauncherRuntime())
 	}
 
 	if family, ok := familyByName(args[0]); ok {
@@ -64,6 +70,93 @@ func Dispatch(args []string) Result {
 	}
 }
 
+type LauncherRuntime struct {
+	Env      launcher.Env
+	Sessions launcher.SessionSource
+	History  launcher.HistoricalScanner
+}
+
+type StaticSessions struct {
+	Sessions []launcher.Session
+	Err      error
+}
+
+func (s StaticSessions) Snapshot() ([]launcher.Session, error) {
+	if s.Err != nil {
+		return nil, s.Err
+	}
+	return s.Sessions, nil
+}
+
+type StaticHistory struct {
+	Tags []launcher.HistoricalTag
+	Err  error
+}
+
+func (h StaticHistory) Scan(_ string, _ time.Time) ([]launcher.HistoricalTag, error) {
+	if h.Err != nil {
+		return nil, h.Err
+	}
+	return h.Tags, nil
+}
+
+func DispatchWithLauncherRuntime(args []string, rt LauncherRuntime) Result {
+	launchArgs := []string(nil)
+	if len(args) > 1 {
+		launchArgs = args[1:]
+	}
+	if len(launchArgs) == 0 || launchArgs[0] == "help" || launchArgs[0] == "--help" || launchArgs[0] == "-h" {
+		return Result{Stdout: LaunchHelp(programName), ExitCode: 0}
+	}
+	outcome, err := launcher.Run(launchArgs, rt.Env, rt.Sessions, rt.History)
+	if err != nil {
+		return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
+	}
+	decision := outcome.Decision
+	return Result{
+		Stderr: fmt.Sprintf(
+			"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
+			decision.Action,
+			decision.Tag,
+			decision.SessionName,
+		),
+		ExitCode: 3,
+	}
+}
+
+func LaunchHelp(program string) string {
+	return fmt.Sprintf(`Usage: %s launch [agent] [-- agent-args...]
+       %s launch resume <tag>
+
+Guarded decision-phase prototype. Public sessions still start through bin/pair.
+This command parses launch inputs and computes the create/attach/picker decision,
+then stops before invoking zellij.
+`, program, program)
+}
+
+func LauncherEnv(home, xdgDataHome, cwd string) launcher.Env {
+	return launcher.Env{
+		Home:     home,
+		XDGData:  xdgDataHome,
+		Cwd:      cwd,
+		Now:      time.Now(),
+		HistoryD: 14,
+	}
+}
+
+func osLauncherRuntime() LauncherRuntime {
+	home := os.Getenv("HOME")
+	xdg := os.Getenv("XDG_DATA_HOME")
+	cwd, _ := os.Getwd()
+	env := LauncherEnv(home, xdg, cwd)
+	dataDir := launcher.ResolveDataDir(home, xdg)
+	return LauncherRuntime{
+		Env:      env,
+		Sessions: launcher.ZellijSource{},
+		History:  launcher.HistorySource{DataDir: dataDir},
+	}
+}
+
 // Help renders the development-only dispatcher usage text.
 func Help(program string) string {
 	var b strings.Builder
diff --git a/cmd/internal/dispatcher/dispatcher_test.go b/cmd/internal/dispatcher/dispatcher_test.go
index c9fed26..1d1358a 100644
--- a/cmd/internal/dispatcher/dispatcher_test.go
+++ b/cmd/internal/dispatcher/dispatcher_test.go
@@ -60,6 +60,42 @@ func TestDispatchPlannedCommandReturnsUnsupported(t *testing.T) {
 	}
 }
 
+func TestDispatchLaunchHelpRoutesToPrototype(t *testing.T) {
+	res := Dispatch([]string{"launch", "--help"})
+	if res.ExitCode != 0 {
+		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
+	}
+	if res.Stderr != "" {
+		t.Fatalf("Stderr = %q, want empty", res.Stderr)
+	}
+	for _, want := range []string{"Usage: pair-go launch", "decision-phase prototype"} {
+		if !strings.Contains(res.Stdout, want) {
+			t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
+		}
+	}
+}
+
+func TestDispatchLaunchReturnsPrototypeDecision(t *testing.T) {
+	res := DispatchWithLauncherRuntime([]string{"launch", "resume", "demo"}, LauncherRuntime{
+		Env: LauncherEnv("/home/me", "", "/work/pair"),
+		Sessions: StaticSessions{
+			Sessions: nil,
+		},
+		History: StaticHistory{},
+	})
+	if res.ExitCode != 3 {
+		t.Fatalf("ExitCode = %d, want 3", res.ExitCode)
+	}
+	if res.Stdout != "" {
+		t.Fatalf("Stdout = %q, want empty", res.Stdout)
+	}
+	for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
+		if !strings.Contains(res.Stderr, want) {
+			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
+		}
+	}
+}
+
 func TestDispatchUnknownCommandReturnsUsageHint(t *testing.T) {
 	res := Dispatch([]string{"frobnicate"})
 	if res.ExitCode != 2 {
diff --git a/cmd/internal/launcher/args.go b/cmd/internal/launcher/args.go
new file mode 100644
index 0000000..06aa8fb
--- /dev/null
+++ b/cmd/internal/launcher/args.go
@@ -0,0 +1,71 @@
+package launcher
+
+import "fmt"
+
+// LaunchArgs is the pure parse result for the guarded pair-go launch prototype.
+type LaunchArgs struct {
+	Agent       string
+	ForcedTag   string
+	SelectedTag string
+	AgentArgs   []string
+}
+
+// UsageError is an operator-facing parse error.
+type UsageError struct {
+	Message string
+}
+
+func (e UsageError) Error() string {
+	return e.Message
+}
+
+// ParseArgs parses pair-go launch args. It intentionally supports only the
+// decision-phase subset for #75; unsupported shell-owned launcher verbs fail
+// explicitly.
+func ParseArgs(argv []string) (LaunchArgs, error) {
+	var out LaunchArgs
+	if len(argv) == 0 {
+		out.Agent = "claude"
+		return out, nil
+	}
+
+	switch argv[0] {
+	case "continue", "rename", "list", "ls":
+		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %s is not implemented by pair-go launch; use bin/pair", argv[0])}
+	case "resume":
+		if len(argv) < 2 {
+			return LaunchArgs{}, UsageError{Message: "pair-go launch: 'resume' requires a tag"}
+		}
+		tag, err := NormalizeTag(argv[1])
+		if err != nil {
+			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: invalid tag: %v", err)}
+		}
+		if len(argv) > 2 {
+			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", argv[2])}
+		}
+		out.ForcedTag = tag
+		return out, nil
+	}
+
+	seenSeparator := false
+	for _, arg := range argv {
+		if seenSeparator {
+			out.AgentArgs = append(out.AgentArgs, arg)
+			continue
+		}
+		if arg == "--" {
+			seenSeparator = true
+			continue
+		}
+		if out.Agent == "" {
+			out.Agent = arg
+			continue
+		}
+		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", arg)}
+	}
+
+	if out.Agent == "" {
+		out.Agent = "claude"
+	}
+	return out, nil
+}
diff --git a/cmd/internal/launcher/args_test.go b/cmd/internal/launcher/args_test.go
new file mode 100644
index 0000000..85bc25e
--- /dev/null
+++ b/cmd/internal/launcher/args_test.go
@@ -0,0 +1,88 @@
+package launcher
+
+import (
+	"strings"
+	"testing"
+)
+
+func TestParseLaunchArgsDefaultsToClaude(t *testing.T) {
+	args, err := ParseArgs(nil)
+	if err != nil {
+		t.Fatalf("ParseArgs returned error: %v", err)
+	}
+	if args.Agent != "claude" {
+		t.Fatalf("Agent = %q, want claude", args.Agent)
+	}
+	if args.ForcedTag != "" {
+		t.Fatalf("ForcedTag = %q, want empty", args.ForcedTag)
+	}
+	if len(args.AgentArgs) != 0 {
+		t.Fatalf("AgentArgs = %#v, want empty", args.AgentArgs)
+	}
+}
+
+func TestParseLaunchArgsAgentAndForwardedArgs(t *testing.T) {
+	args, err := ParseArgs([]string{"codex", "--", "-p", "say hi"})
+	if err != nil {
+		t.Fatalf("ParseArgs returned error: %v", err)
+	}
+	if args.Agent != "codex" {
+		t.Fatalf("Agent = %q, want codex", args.Agent)
+	}
+	if got := strings.Join(args.AgentArgs, " "); got != "-p say hi" {
+		t.Fatalf("AgentArgs = %q, want forwarded args", got)
+	}
+}
+
+func TestParseLaunchArgsDefaultAgentWithForwardedArgs(t *testing.T) {
+	args, err := ParseArgs([]string{"--", "--dangerously-skip-permissions"})
+	if err != nil {
+		t.Fatalf("ParseArgs returned error: %v", err)
+	}
+	if args.Agent != "claude" {
+		t.Fatalf("Agent = %q, want claude", args.Agent)
+	}
+	if got := strings.Join(args.AgentArgs, " "); got != "--dangerously-skip-permissions" {
+		t.Fatalf("AgentArgs = %q, want forwarded args", got)
+	}
+}
+
+func TestParseLaunchArgsResumeNormalizesForcedTag(t *testing.T) {
+	args, err := ParseArgs([]string{"resume", "pair-demo"})
+	if err != nil {
+		t.Fatalf("ParseArgs returned error: %v", err)
+	}
+	if args.Agent != "" {
+		t.Fatalf("Agent = %q, want empty for resume inference", args.Agent)
+	}
+	if args.ForcedTag != "demo" {
+		t.Fatalf("ForcedTag = %q, want demo", args.ForcedTag)
+	}
+}
+
+func TestParseLaunchArgsUnexpectedPositionalGuidesAgentArgs(t *testing.T) {
+	_, err := ParseArgs([]string{"codex", "extra"})
+	if err == nil {
+		t.Fatal("ParseArgs returned nil error")
+	}
+	msg := err.Error()
+	for _, want := range []string{"unexpected positional arg", "use '--' to forward args to the agent"} {
+		if !strings.Contains(msg, want) {
+			t.Fatalf("error missing %q: %s", want, msg)
+		}
+	}
+}
+
+func TestParseLaunchArgsUnsupportedLaunchSubcommandsAreExplicit(t *testing.T) {
+	for _, verb := range []string{"continue", "rename", "list"} {
+		t.Run(verb, func(t *testing.T) {
+			_, err := ParseArgs([]string{verb})
+			if err == nil {
+				t.Fatal("ParseArgs returned nil error")
+			}
+			if !strings.Contains(err.Error(), "not implemented by pair-go launch") {
+				t.Fatalf("error = %q, want explicit unsupported message", err)
+			}
+		})
+	}
+}
diff --git a/cmd/internal/launcher/datadir.go b/cmd/internal/launcher/datadir.go
new file mode 100644
index 0000000..7a57144
--- /dev/null
+++ b/cmd/internal/launcher/datadir.go
@@ -0,0 +1,11 @@
+package launcher
+
+import "path/filepath"
+
+// ResolveDataDir returns Pair's data directory from explicit environment values.
+func ResolveDataDir(home, xdgDataHome string) string {
+	if xdgDataHome != "" {
+		return filepath.Join(xdgDataHome, "pair")
+	}
+	return filepath.Join(home, ".local", "share", "pair")
+}
diff --git a/cmd/internal/launcher/datadir_test.go b/cmd/internal/launcher/datadir_test.go
new file mode 100644
index 0000000..bb604b6
--- /dev/null
+++ b/cmd/internal/launcher/datadir_test.go
@@ -0,0 +1,21 @@
+package launcher
+
+import "testing"
+
+func TestResolveDataDir(t *testing.T) {
+	for _, tc := range []struct {
+		name string
+		home string
+		xdg  string
+		want string
+	}{
+		{name: "xdg", home: "/home/me", xdg: "/tmp/xdg", want: "/tmp/xdg/pair"},
+		{name: "home", home: "/home/me", want: "/home/me/.local/share/pair"},
+	} {
+		t.Run(tc.name, func(t *testing.T) {
+			if got := ResolveDataDir(tc.home, tc.xdg); got != tc.want {
+				t.Fatalf("ResolveDataDir(%q, %q) = %q, want %q", tc.home, tc.xdg, got, tc.want)
+			}
+		})
+	}
+}
diff --git a/cmd/internal/launcher/decision.go b/cmd/internal/launcher/decision.go
new file mode 100644
index 0000000..de49d33
--- /dev/null
+++ b/cmd/internal/launcher/decision.go
@@ -0,0 +1,103 @@
+package launcher
+
+// LaunchAction is the guarded prototype's next launcher action.
+type LaunchAction string
+
+const (
+	ActionAttach LaunchAction = "attach"
+	ActionCreate LaunchAction = "create"
+	ActionPick   LaunchAction = "pick"
+)
+
+// LaunchDecision is a pure create/attach/pick decision. Tag is canonical bare
+// form; SessionName is derived as pair-<tag> when a zellij session is named.
+type LaunchDecision struct {
+	Action      LaunchAction
+	Tag         string
+	SessionName string
+	PromptName  bool
+}
+
+// DecideLaunch decides the launch action without touching zellij, fzf, or disk.
+func DecideLaunch(args LaunchArgs, snap SessionSnapshot) (LaunchDecision, error) {
+	if args.SelectedTag != "" {
+		return createDecision(args.SelectedTag, false), nil
+	}
+	if args.ForcedTag != "" {
+		if sessionBlocksReuse(snap, sessionName(args.ForcedTag)) {
+			return LaunchDecision{Action: ActionAttach, Tag: args.ForcedTag, SessionName: sessionName(args.ForcedTag)}, nil
+		}
+		return createDecision(args.ForcedTag, false), nil
+	}
+	if hasDetached(snap) || len(snap.Historical) > 0 {
+		return LaunchDecision{Action: ActionPick}, nil
+	}
+	tag := snap.BaseTag
+	if tag == "" {
+		tag = "pair"
+	}
+	return createDecision(nextFreeTag(tag, snap), true), nil
+}
+
+func createDecision(tag string, prompt bool) LaunchDecision {
+	return LaunchDecision{Action: ActionCreate, Tag: tag, SessionName: sessionName(tag), PromptName: prompt}
+}
+
+func sessionName(tag string) string {
+	return "pair-" + tag
+}
+
+func hasDetached(snap SessionSnapshot) bool {
+	for _, sess := range snap.Sessions {
+		if sess.State == SessionDetached {
+			return true
+		}
+	}
+	return false
+}
+
+func sessionBlocksReuse(snap SessionSnapshot, name string) bool {
+	for _, sess := range snap.Sessions {
+		if sess.Name != name {
+			continue
+		}
+		return sess.State == SessionAttached || sess.State == SessionDetached
+	}
+	return false
+}
+
+func nextFreeTag(base string, snap SessionSnapshot) string {
+	for i := 1; i <= 100; i++ {
+		tag := base
+		if i > 1 {
+			tag = base + "-" + itoa(i)
+		}
+		if !sessionBlocksReuse(snap, sessionName(tag)) && !isHistorical(snap, tag) {
+			return tag
+		}
+	}
+	return base
+}
+
+func isHistorical(snap SessionSnapshot, tag string) bool {
+	for _, hist := range snap.Historical {
+		if hist.Tag == tag {
+			return true
+		}
+	}
+	return false
+}
+
+func itoa(n int) string {
+	if n == 0 {
+		return "0"
+	}
+	var buf [20]byte
+	i := len(buf)
+	for n > 0 {
+		i--
+		buf[i] = byte('0' + n%10)
+		n /= 10
+	}
+	return string(buf[i:])
+}
diff --git a/cmd/internal/launcher/decision_test.go b/cmd/internal/launcher/decision_test.go
new file mode 100644
index 0000000..8c39d21
--- /dev/null
+++ b/cmd/internal/launcher/decision_test.go
@@ -0,0 +1,67 @@
+package launcher
+
+import "testing"
+
+func TestDecideLaunchForcedResumeAttachesWhenSessionBlocksReuse(t *testing.T) {
+	decision, err := DecideLaunch(LaunchArgs{ForcedTag: "demo"}, SessionSnapshot{
+		Sessions: []Session{{Name: "pair-demo", State: SessionDetached}},
+	})
+	if err != nil {
+		t.Fatalf("DecideLaunch returned error: %v", err)
+	}
+	if decision.Action != ActionAttach || decision.Tag != "demo" || decision.SessionName != "pair-demo" {
+		t.Fatalf("decision = %#v, want attach demo/pair-demo", decision)
+	}
+}
+
+func TestDecideLaunchForcedResumeCreatesWhenSessionDoesNotBlockReuse(t *testing.T) {
+	decision, err := DecideLaunch(LaunchArgs{ForcedTag: "demo"}, SessionSnapshot{
+		Sessions: []Session{{Name: "pair-demo", State: SessionExited}},
+	})
+	if err != nil {
+		t.Fatalf("DecideLaunch returned error: %v", err)
+	}
+	if decision.Action != ActionCreate || decision.Tag != "demo" || decision.SessionName != "pair-demo" || decision.PromptName {
+		t.Fatalf("decision = %#v, want create demo/pair-demo without prompt", decision)
+	}
+}
+
+func TestDecideLaunchEmptyStateCreatesNextFreeTagWithPrompt(t *testing.T) {
+	decision, err := DecideLaunch(LaunchArgs{Agent: "codex"}, SessionSnapshot{
+		BaseTag: "pair",
+	})
+	if err != nil {
+		t.Fatalf("DecideLaunch returned error: %v", err)
+	}
+	if decision.Action != ActionCreate || decision.Tag != "pair" || decision.SessionName != "pair-pair" || !decision.PromptName {
+		t.Fatalf("decision = %#v, want create next free tag with prompt", decision)
+	}
+}
+
+func TestDecideLaunchShowsPickerWhenDetachedOrHistoricalExist(t *testing.T) {
+	for _, snap := range []SessionSnapshot{
+		{BaseTag: "pair", Sessions: []Session{{Name: "pair-other", State: SessionDetached}}},
+		{BaseTag: "pair", Historical: []HistoricalTag{{Tag: "pair-old"}}},
+	} {
+		decision, err := DecideLaunch(LaunchArgs{Agent: "claude"}, snap)
+		if err != nil {
+			t.Fatalf("DecideLaunch returned error: %v", err)
+		}
+		if decision.Action != ActionPick {
+			t.Fatalf("decision = %#v, want picker", decision)
+		}
+	}
+}
+
+func TestDecideLaunchHistoricalSelectionCreatesByTag(t *testing.T) {
+	decision, err := DecideLaunch(LaunchArgs{Agent: "claude", SelectedTag: "pair-old"}, SessionSnapshot{
+		BaseTag:    "pair",
+		Historical: []HistoricalTag{{Tag: "pair-old"}},
+	})
+	if err != nil {
+		t.Fatalf("DecideLaunch returned error: %v", err)
+	}
+	if decision.Action != ActionCreate || decision.Tag != "pair-old" || decision.SessionName != "pair-pair-old" || decision.PromptName {
+		t.Fatalf("decision = %#v, want create historical tag without prompt", decision)
+	}
+}
diff --git a/cmd/internal/launcher/history.go b/cmd/internal/launcher/history.go
new file mode 100644
index 0000000..0e9ef52
--- /dev/null
+++ b/cmd/internal/launcher/history.go
@@ -0,0 +1,65 @@
+package launcher
+
+import (
+	"os"
+	"path/filepath"
+	"sort"
+	"strings"
+	"time"
+)
+
+// HistorySource scans Pair draft/log sidecars under the data directory.
+type HistorySource struct {
+	DataDir string
+}
+
+func (s HistorySource) Scan(base string, cutoff time.Time) ([]HistoricalTag, error) {
+	latest := map[string]time.Time{}
+	for _, pattern := range []string{"draft-*.md", "log-*.md"} {
+		matches, err := filepath.Glob(filepath.Join(s.DataDir, pattern))
+		if err != nil {
+			return nil, err
+		}
+		for _, path := range matches {
+			tag, ok := tagFromSidecar(filepath.Base(path))
+			if !ok || !matchesHistoryBase(tag, base) {
+				continue
+			}
+			info, err := os.Stat(path)
+			if err != nil {
+				continue
+			}
+			if info.ModTime().Before(cutoff) {
+				continue
+			}
+			if info.ModTime().After(latest[tag]) {
+				latest[tag] = info.ModTime()
+			}
+		}
+	}
+	tags := make([]string, 0, len(latest))
+	for tag := range latest {
+		tags = append(tags, tag)
+	}
+	sort.Strings(tags)
+	out := make([]HistoricalTag, 0, len(tags))
+	for _, tag := range tags {
+		out = append(out, HistoricalTag{Tag: tag})
+	}
+	return out, nil
+}
+
+func tagFromSidecar(name string) (string, bool) {
+	switch {
+	case strings.HasPrefix(name, "draft-") && strings.HasSuffix(name, ".md"):
+		return strings.TrimSuffix(strings.TrimPrefix(name, "draft-"), ".md"), true
+	case strings.HasPrefix(name, "log-") && strings.HasSuffix(name, ".md"):
+		return strings.TrimSuffix(strings.TrimPrefix(name, "log-"), ".md"), true
+	default:
+		return "", false
+	}
+}
+
+func matchesHistoryBase(tag, base string) bool {
+	return tag == base || strings.HasPrefix(tag, base+"-")
+}
diff --git a/cmd/internal/launcher/history_test.go b/cmd/internal/launcher/history_test.go
new file mode 100644
index 0000000..5f158fd
--- /dev/null
+++ b/cmd/internal/launcher/history_test.go
@@ -0,0 +1,33 @@
+package launcher
+
+import (
+	"os"
+	"path/filepath"
+	"testing"
+	"time"
+)
+
+func TestHistorySourceScansCwdPrefixedDraftAndLogSidecars(t *testing.T) {
+	dir := t.TempDir()
+	now := time.Unix(2000, 0)
+	for _, name := range []string{"draft-pair.md", "log-pair-old.md", "draft-other.md"} {
+		path := filepath.Join(dir, name)
+		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
+			t.Fatal(err)
+		}
+		if err := os.Chtimes(path, now, now); err != nil {
+			t.Fatal(err)
+		}
+	}
+
+	got, err := HistorySource{DataDir: dir}.Scan("pair", now.Add(-24*time.Hour))
+	if err != nil {
+		t.Fatalf("Scan returned error: %v", err)
+	}
+	if len(got) != 2 {
+		t.Fatalf("Scan returned %#v, want 2 pair-prefixed tags", got)
+	}
+	if got[0].Tag != "pair" || got[1].Tag != "pair-old" {
+		t.Fatalf("Scan returned %#v, want sorted pair tags", got)
+	}
+}
diff --git a/cmd/internal/launcher/run.go b/cmd/internal/launcher/run.go
new file mode 100644
index 0000000..c3d9b2c
--- /dev/null
+++ b/cmd/internal/launcher/run.go
@@ -0,0 +1,70 @@
+package launcher
+
+import (
+	"time"
+)
+
+// Env is the explicit runtime environment for launch decisions.
+type Env struct {
+	Home     string
+	XDGData  string
+	Cwd      string
+	Now      time.Time
+	HistoryD int
+	DataDir  string
+}
+
+// SessionSource supplies zellij session state.
+type SessionSource interface {
+	Snapshot() ([]Session, error)
+}
+
+// HistoricalScanner supplies recently touched Pair tags.
+type HistoricalScanner interface {
+	Scan(base string, cutoff time.Time) ([]HistoricalTag, error)
+}
+
+// LaunchOutcome is the domain outcome from the launcher core.
+type LaunchOutcome struct {
+	Args     LaunchArgs
+	Env      Env
+	Decision LaunchDecision
+}
+
+// Run builds a pure snapshot from injected sources and returns a domain launch
+// outcome. The dispatcher maps this to process stdout/stderr/exit status.
+func Run(argv []string, env Env, sessions SessionSource, history HistoricalScanner) (LaunchOutcome, error) {
+	args, err := ParseArgs(argv)
+	if err != nil {
+		return LaunchOutcome{}, err
+	}
+	if env.DataDir == "" {
+		env.DataDir = ResolveDataDir(env.Home, env.XDGData)
+	}
+	if env.HistoryD == 0 {
+		env.HistoryD = 14
+	}
+	if env.Now.IsZero() {
+		env.Now = time.Now()
+	}
+
+	sessionRows, err := sessions.Snapshot()
+	if err != nil {
+		return LaunchOutcome{}, err
+	}
+	base := DefaultTag(env.Cwd)
+	historical, err := history.Scan(base, env.Now.Add(-time.Duration(env.HistoryD)*24*time.Hour))
+	if err != nil {
+		return LaunchOutcome{}, err
+	}
+
+	decision, err := DecideLaunch(args, SessionSnapshot{
+		BaseTag:    base,
+		Sessions:   sessionRows,
+		Historical: historical,
+	})
+	if err != nil {
+		return LaunchOutcome{}, err
+	}
+	return LaunchOutcome{Args: args, Env: env, Decision: decision}, nil
+}
diff --git a/cmd/internal/launcher/run_test.go b/cmd/internal/launcher/run_test.go
new file mode 100644
index 0000000..fdc7ced
--- /dev/null
+++ b/cmd/internal/launcher/run_test.go
@@ -0,0 +1,90 @@
+package launcher
+
+import (
+	"errors"
+	"testing"
+	"time"
+)
+
+type fakeSessions struct {
+	sessions []Session
+	err      error
+}
+
+func (f fakeSessions) Snapshot() ([]Session, error) {
+	if f.err != nil {
+		return nil, f.err
+	}
+	return f.sessions, nil
+}
+
+type fakeHistory struct {
+	tags []HistoricalTag
+	err  error
+}
+
+func (f fakeHistory) Scan(base string, cutoff time.Time) ([]HistoricalTag, error) {
+	if f.err != nil {
+		return nil, f.err
+	}
+	return f.tags, nil
+}
+
+func TestRunLaunchUsesSuppliedEnvironment(t *testing.T) {
+	outcome, err := Run([]string{"codex"}, Env{
+		Home:     "/home/me",
+		Cwd:      "/work/pair",
+		Now:      time.Unix(1000, 0),
+		HistoryD: 14,
+	}, fakeSessions{}, fakeHistory{})
+	if err != nil {
+		t.Fatalf("Run returned error: %v", err)
+	}
+	if outcome.Decision.Action != ActionCreate || outcome.Decision.Tag != "pair" || !outcome.Decision.PromptName {
+		t.Fatalf("Decision = %#v, want create pair with prompt", outcome.Decision)
+	}
+	if outcome.Env.DataDir != "/home/me/.local/share/pair" {
+		t.Fatalf("DataDir = %q, want home-derived data dir", outcome.Env.DataDir)
+	}
+}
+
+func TestRunLaunchTurnsFakeSessionsIntoPickerDecision(t *testing.T) {
+	outcome, err := Run([]string{"claude"}, Env{
+		Home:     "/home/me",
+		Cwd:      "/work/pair",
+		Now:      time.Unix(1000, 0),
+		HistoryD: 14,
+	}, fakeSessions{sessions: []Session{{Name: "pair-demo", State: SessionDetached}}}, fakeHistory{})
+	if err != nil {
+		t.Fatalf("Run returned error: %v", err)
+	}
+	if outcome.Decision.Action != ActionPick {
+		t.Fatalf("Decision = %#v, want picker", outcome.Decision)
+	}
+}
+
+func TestRunLaunchTurnsFakeHistoryIntoPickerDecision(t *testing.T) {
+	outcome, err := Run([]string{"claude"}, Env{
+		Home:     "/home/me",
+		Cwd:      "/work/pair",
+		Now:      time.Unix(1000, 0),
+		HistoryD: 14,
+	}, fakeSessions{}, fakeHistory{tags: []HistoricalTag{{Tag: "pair-old"}}})
+	if err != nil {
+		t.Fatalf("Run returned error: %v", err)
+	}
+	if outcome.Decision.Action != ActionPick {
+		t.Fatalf("Decision = %#v, want picker", outcome.Decision)
+	}
+}
+
+func TestRunLaunchReturnsTypedUsageError(t *testing.T) {
+	_, err := Run([]string{"codex", "extra"}, Env{Home: "/home/me", Cwd: "/work/pair"}, fakeSessions{}, fakeHistory{})
+	if err == nil {
+		t.Fatal("Run returned nil error")
+	}
+	var usage UsageError
+	if !errors.As(err, &usage) {
+		t.Fatalf("error = %T, want UsageError", err)
+	}
+}
diff --git a/cmd/internal/launcher/session.go b/cmd/internal/launcher/session.go
new file mode 100644
index 0000000..9dc303a
--- /dev/null
+++ b/cmd/internal/launcher/session.go
@@ -0,0 +1,28 @@
+package launcher
+
+// SessionState describes whether a zellij session blocks tag reuse.
+type SessionState string
+
+const (
+	SessionAttached SessionState = "attached"
+	SessionDetached SessionState = "detached"
+	SessionExited   SessionState = "exited"
+)
+
+// Session is a zellij session row projected into launcher decision space.
+type Session struct {
+	Name  string
+	State SessionState
+}
+
+// HistoricalTag is a recently touched Pair tag with no live zellij session.
+type HistoricalTag struct {
+	Tag string
+}
+
+// SessionSnapshot is the pure input to launcher decision-making.
+type SessionSnapshot struct {
+	BaseTag    string
+	Sessions   []Session
+	Historical []HistoricalTag
+}
diff --git a/cmd/internal/launcher/tag.go b/cmd/internal/launcher/tag.go
new file mode 100644
index 0000000..f8bb4c6
--- /dev/null
+++ b/cmd/internal/launcher/tag.go
@@ -0,0 +1,42 @@
+package launcher
+
+import (
+	"fmt"
+	"path/filepath"
+	"strings"
+)
+
+// NormalizeTag returns Pair's canonical bare tag form.
+func NormalizeTag(raw string) (string, error) {
+	tag := strings.TrimPrefix(raw, "pair-")
+	if tag == "" {
+		return "", fmt.Errorf("empty tag")
+	}
+	for _, r := range tag {
+		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
+			continue
+		}
+		return "", fmt.Errorf("tag %q contains invalid character %q", raw, r)
+	}
+	return tag, nil
+}
+
+// DefaultTag derives Pair's create-flow default tag from a cwd path.
+func DefaultTag(cwd string) string {
+	base := filepath.Base(cwd)
+	if base == "." || base == string(filepath.Separator) {
+		return "pair"
+	}
+	var b strings.Builder
+	for _, r := range base {
+		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
+			b.WriteRune(r)
+		} else {
+			b.WriteByte('_')
+		}
+	}
+	if b.Len() == 0 {
+		return "pair"
+	}
+	return b.String()
+}
diff --git a/cmd/internal/launcher/tag_test.go b/cmd/internal/launcher/tag_test.go
new file mode 100644
index 0000000..ff30987
--- /dev/null
+++ b/cmd/internal/launcher/tag_test.go
@@ -0,0 +1,53 @@
+package launcher
+
+import "testing"
+
+func TestNormalizeTag(t *testing.T) {
+	for _, tc := range []struct {
+		name string
+		raw  string
+		want string
+	}{
+		{name: "bare", raw: "demo", want: "demo"},
+		{name: "prefixed", raw: "pair-demo", want: "demo"},
+		{name: "underscore", raw: "pair-demo_2", want: "demo_2"},
+	} {
+		t.Run(tc.name, func(t *testing.T) {
+			got, err := NormalizeTag(tc.raw)
+			if err != nil {
+				t.Fatalf("NormalizeTag returned error: %v", err)
+			}
+			if got != tc.want {
+				t.Fatalf("NormalizeTag(%q) = %q, want %q", tc.raw, got, tc.want)
+			}
+		})
+	}
+}
+
+func TestNormalizeTagRejectsInvalid(t *testing.T) {
+	for _, raw := range []string{"", "pair-", "bad/slug", "has space"} {
+		t.Run(raw, func(t *testing.T) {
+			if _, err := NormalizeTag(raw); err == nil {
+				t.Fatalf("NormalizeTag(%q) returned nil error", raw)
+			}
+		})
+	}
+}
+
+func TestDefaultTag(t *testing.T) {
+	for _, tc := range []struct {
+		cwd  string
+		want string
+	}{
+		{cwd: "/Users/xianxu/workspace/pair", want: "pair"},
+		{cwd: "/tmp/hello world", want: "hello_world"},
+		{cwd: "/tmp/!!!", want: "___"},
+		{cwd: "", want: "pair"},
+	} {
+		t.Run(tc.cwd, func(t *testing.T) {
+			if got := DefaultTag(tc.cwd); got != tc.want {
+				t.Fatalf("DefaultTag(%q) = %q, want %q", tc.cwd, got, tc.want)
+			}
+		})
+	}
+}
diff --git a/cmd/internal/launcher/zellij.go b/cmd/internal/launcher/zellij.go
new file mode 100644
index 0000000..96febe8
--- /dev/null
+++ b/cmd/internal/launcher/zellij.go
@@ -0,0 +1,91 @@
+package launcher
+
+import (
+	"bytes"
+	"os/exec"
+	"sort"
+	"strings"
+)
+
+// ZellijSource reads zellij session state through the zellij CLI.
+type ZellijSource struct {
+	Path string
+}
+
+func (s ZellijSource) Snapshot() ([]Session, error) {
+	short, err := s.run("list-sessions", "--short")
+	if err != nil {
+		short = nil
+	}
+	raw, err := s.run("list-sessions", "--no-formatting")
+	if err != nil {
+		raw = nil
+	}
+	exited := exitedSessions(string(raw))
+	var out []Session
+	for _, name := range lines(string(short)) {
+		if !strings.HasPrefix(name, "pair-") {
+			continue
+		}
+		state := SessionDetached
+		if exited[name] {
+			state = SessionExited
+		} else if s.clientCount(name) > 0 {
+			state = SessionAttached
+		}
+		out = append(out, Session{Name: name, State: state})
+	}
+	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
+	return out, nil
+}
+
+func (s ZellijSource) clientCount(session string) int {
+	out, err := s.run("--session", session, "action", "list-clients")
+	if err != nil {
+		return 0
+	}
+	lines := lines(string(out))
+	if len(lines) <= 1 {
+		return 0
+	}
+	return len(lines) - 1
+}
+
+func (s ZellijSource) run(args ...string) ([]byte, error) {
+	path := s.Path
+	if path == "" {
+		path = "zellij"
+	}
+	cmd := exec.Command(path, args...)
+	var stdout bytes.Buffer
+	cmd.Stdout = &stdout
+	if err := cmd.Run(); err != nil {
+		return nil, err
+	}
+	return stdout.Bytes(), nil
+}
+
+func exitedSessions(raw string) map[string]bool {
+	out := map[string]bool{}
+	for _, line := range lines(raw) {
+		fields := strings.Fields(line)
+		if len(fields) == 0 {
+			continue
+		}
+		if strings.Contains(line, "EXITED") {
+			out[fields[0]] = true
+		}
+	}
+	return out
+}
+
+func lines(s string) []string {
+	var out []string
+	for _, line := range strings.Split(s, "\n") {
+		line = strings.TrimSpace(line)
+		if line != "" {
+			out = append(out, line)
+		}
+	}
+	return out
+}
diff --git a/cmd/internal/launcher/zellij_test.go b/cmd/internal/launcher/zellij_test.go
new file mode 100644
index 0000000..56bc4b4
--- /dev/null
+++ b/cmd/internal/launcher/zellij_test.go
@@ -0,0 +1,44 @@
+package launcher
+
+import (
+	"os"
+	"path/filepath"
+	"testing"
+)
+
+func TestZellijSourceClassifiesSessions(t *testing.T) {
+	dir := t.TempDir()
+	log := filepath.Join(dir, "zellij.log")
+	zellij := filepath.Join(dir, "zellij")
+	script := `#!/usr/bin/env bash
+printf '%s\n' "$*" >> "` + log + `"
+case "$*" in
+  "list-sessions --short") printf 'pair-live\npair-detached\npair-gone\nother\n' ;;
+  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-gone [Created] (EXITED - attach to resurrect)\n' ;;
+  "--session pair-live action list-clients") printf 'CLIENTS\n1\n' ;;
+  "--session pair-detached action list-clients") printf 'CLIENTS\n' ;;
+  *) exit 1 ;;
+esac
+`
+	if err := os.WriteFile(zellij, []byte(script), 0o755); err != nil {
+		t.Fatal(err)
+	}
+
+	got, err := ZellijSource{Path: zellij}.Snapshot()
+	if err != nil {
+		t.Fatalf("Snapshot returned error: %v", err)
+	}
+	want := []Session{
+		{Name: "pair-detached", State: SessionDetached},
+		{Name: "pair-gone", State: SessionExited},
+		{Name: "pair-live", State: SessionAttached},
+	}
+	if len(got) != len(want) {
+		t.Fatalf("Snapshot returned %#v, want %#v", got, want)
+	}
+	for i := range want {
+		if got[i] != want[i] {
+			t.Fatalf("Snapshot[%d] = %#v, want %#v", i, got[i], want[i])
+		}
+	}
+}
diff --git a/cmd/pair-go/launch_process_test.go b/cmd/pair-go/launch_process_test.go
new file mode 100644
index 0000000..9563492
--- /dev/null
+++ b/cmd/pair-go/launch_process_test.go
@@ -0,0 +1,71 @@
+package main
+
+import (
+	"bytes"
+	"os"
+	"path/filepath"
+	"strings"
+	"testing"
+	"time"
+)
+
+func TestRunLaunchWithFakeZellij(t *testing.T) {
+	rt := t.TempDir()
+	bin := filepath.Join(rt, "bin")
+	data := filepath.Join(rt, "data")
+	if err := os.MkdirAll(bin, 0o755); err != nil {
+		t.Fatal(err)
+	}
+	if err := os.MkdirAll(filepath.Join(data, "pair"), 0o755); err != nil {
+		t.Fatal(err)
+	}
+	log := filepath.Join(rt, "zellij.log")
+	zellij := filepath.Join(bin, "zellij")
+	script := `#!/usr/bin/env bash
+printf '%s\n' "$*" >> "` + log + `"
+case "$*" in
+  "list-sessions --short") printf 'pair-live\npair-detached\npair-exited\n' ;;
+  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-exited [Created] (EXITED - attach to resurrect)\n' ;;
+  "--session pair-live action list-clients") printf 'CLIENTS\n1\n' ;;
+  "--session pair-detached action list-clients") printf 'CLIENTS\n' ;;
+  *attach*|*new-session*|*--new-session-with-layout*|*delete-session*) printf 'MUTATING %s\n' "$*" >> "` + log + `"; exit 99 ;;
+  *) exit 1 ;;
+esac
+`
+	if err := os.WriteFile(zellij, []byte(script), 0o755); err != nil {
+		t.Fatal(err)
+	}
+	historical := filepath.Join(data, "pair", "draft-pair-old.md")
+	if err := os.WriteFile(historical, []byte("draft"), 0o644); err != nil {
+		t.Fatal(err)
+	}
+	now := time.Now()
+	if err := os.Chtimes(historical, now, now); err != nil {
+		t.Fatal(err)
+	}
+
+	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
+	t.Setenv("HOME", filepath.Join(rt, "home"))
+	t.Setenv("XDG_DATA_HOME", data)
+
+	var stdout, stderr bytes.Buffer
+	code := run([]string{"launch", "claude"}, &stdout, &stderr)
+	if code != 3 {
+		t.Fatalf("code = %d, want 3; stderr:\n%s", code, stderr.String())
+	}
+	if stdout.String() != "" {
+		t.Fatalf("stdout = %q, want empty", stdout.String())
+	}
+	for _, want := range []string{"prototype decision", "action=pick"} {
+		if !strings.Contains(stderr.String(), want) {
+			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
+		}
+	}
+	logBytes, err := os.ReadFile(log)
+	if err != nil {
+		t.Fatal(err)
+	}
+	if strings.Contains(string(logBytes), "MUTATING") {
+		t.Fatalf("fake zellij recorded mutating invocation:\n%s", string(logBytes))
+	}
+}
diff --git a/cmd/pair-go/main.go b/cmd/pair-go/main.go
index 29cafe0..bf7d28b 100644
--- a/cmd/pair-go/main.go
+++ b/cmd/pair-go/main.go
@@ -15,6 +15,15 @@ func main() {
 
 func run(args []string, stdout, stderr io.Writer) int {
 	res := dispatcher.Dispatch(args)
+	return writeResult(res, stdout, stderr)
+}
+
+func runWithLauncherRuntime(args []string, stdout, stderr io.Writer, rt dispatcher.LauncherRuntime) int {
+	res := dispatcher.DispatchWithLauncherRuntime(args, rt)
+	return writeResult(res, stdout, stderr)
+}
+
+func writeResult(res dispatcher.Result, stdout, stderr io.Writer) int {
 	if res.Stdout != "" {
 		_, _ = io.WriteString(stdout, res.Stdout)
 	}
diff --git a/cmd/pair-go/main_test.go b/cmd/pair-go/main_test.go
index 1563f04..ca89631 100644
--- a/cmd/pair-go/main_test.go
+++ b/cmd/pair-go/main_test.go
@@ -4,6 +4,8 @@ import (
 	"bytes"
 	"strings"
 	"testing"
+
+	"github.com/xianxu/pair/cmd/internal/dispatcher"
 )
 
 func TestRunWritesStdoutAndReturnsDispatcherCode(t *testing.T) {
@@ -33,3 +35,41 @@ func TestRunWritesStderrAndReturnsDispatcherCode(t *testing.T) {
 		t.Fatalf("stderr missing unsupported-command message:\n%s", stderr.String())
 	}
 }
+
+func TestRunLaunchHelp(t *testing.T) {
+	var stdout, stderr bytes.Buffer
+	code := run([]string{"launch", "--help"}, &stdout, &stderr)
+	if code != 0 {
+		t.Fatalf("code = %d, want 0", code)
+	}
+	if !strings.Contains(stdout.String(), "Usage: pair-go launch") {
+		t.Fatalf("stdout missing launch usage:\n%s", stdout.String())
+	}
+	if stderr.String() != "" {
+		t.Fatalf("stderr = %q, want empty", stderr.String())
+	}
+}
+
+func TestRunLaunchResumeReturnsPrototypeDecision(t *testing.T) {
+	var stdout, stderr bytes.Buffer
+	code := runWithLauncherRuntime([]string{"launch", "resume", "demo"}, &stdout, &stderr, testLauncherRuntime("/home/me", "", "/work/pair"))
+	if code != 3 {
+		t.Fatalf("code = %d, want 3", code)
+	}
+	if stdout.String() != "" {
+		t.Fatalf("stdout = %q, want empty", stdout.String())
+	}
+	for _, want := range []string{"prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
+		if !strings.Contains(stderr.String(), want) {
+			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
+		}
+	}
+}
+
+func testLauncherRuntime(home, xdg, cwd string) dispatcher.LauncherRuntime {
+	return dispatcher.LauncherRuntime{
+		Env:      dispatcher.LauncherEnv(home, xdg, cwd),
+		Sessions: dispatcher.StaticSessions{},
+		History:  dispatcher.StaticHistory{},
+	}
+}


codex
I’ll use the required repo workflow context first, then review the boundary diff against the issue file and current code.
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
Using `sdlc` and review-oriented repo instructions because this is the SDLC boundary’s fresh-context review; I’m keeping the pass read-only and evidence-driven.
exec
/bin/zsh -lc "sed -n '1,240p' /Users/xianxu/workspace/ariadne/construct/local/sdlc/SKILL.md" in /Users/xianxu/workspace/pair
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
/bin/zsh -lc "sed -n '1,260p' workshop/issues/000075-go-launcher-prototype.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
---
id: 000075
status: done
deps: [000073, 000074]
github_issue:
created: 2026-06-26
updated: 2026-06-29
estimate_hours: 3.75
started: 2026-06-29T21:55:46-07:00
actual_hours: 0.98
---

# pair Go launcher prototype

## Problem

The launcher is the largest remaining shell surface and the most important packaging target, but it owns many behavioral edges: session picker, tag normalization, resume/continue/rename, zellij lifecycle, quit/restart markers, data-dir migrations, orphan cleanup, cmux title ownership, and dev rebuild behavior. Porting it must not break normal Pair usage.

## Spec

Prototype the launcher core in Go behind an alternate or guarded path. The prototype should implement a coherent vertical slice of `bin/pair` behavior while the shell launcher remains the public default.

The #73 inventory identifies `bin/pair` as the P0 public launcher surface. For this milestone, the guarded path is `pair-go launch`, a development-only launcher prototype that reaches the decision boundary but never starts or attaches a real zellij session. The public `bin/pair` shell launcher remains the only supported user entrypoint.

The vertical slice is:

- argv parsing for `pair-go launch`, including default agent, `resume <tag>`, optional agent positional, and `--` forwarded args;
- tag normalization and default tag derivation using the same bare-or-`pair-` contract as `bin/pair`;
- data-dir resolution from `XDG_DATA_HOME` / `HOME`;
- a session snapshot model that represents live, detached, exited, and historical tag candidates;
- a pure decision model for forced resume attach/create, direct create, picker-required, and historical create;
- a thin command/filesystem shell that can read fake `zellij` output and filesystem sidecars in tests, then print the selected prototype action.

The prototype must fail explicitly after the decision phase, rather than silently launching or diverging from `bin/pair`. Unsupported launcher behavior stays shell-owned and documented as out of scope: `continue`, `rename`, zellij lifecycle execution, quit/restart markers, orphan cleanup, cmux title ownership, dev rebuild, tag-restart prompt, config migration, and real fzf interaction.

The Go model stores canonical tags as bare names such as `demo`. Zellij session names are derived at the boundary as `pair-<tag>`. `LaunchDecision` should carry both `Tag` and derived `SessionName` when the action needs a zellij session so printouts and comparisons cannot mix the two forms.

The implementation should keep business decisions pure (`ARCH-PURE`) and reuse the dispatcher introduced in #74 instead of creating a parallel command parser (`ARCH-DRY`). The slice must still satisfy the issue purpose (`ARCH-PURPOSE`): it is not enough to port helpers; `pair-go launch` has to exercise a coherent launcher decision surface.

## Done when

- [x] A guarded Go launcher path can exercise a documented subset of launcher behavior.
- [x] Existing `bin/pair` remains the default public launcher.
- [x] Tests cover the ported decision logic and at least one process-level fake for external commands.
- [x] Any behavior not yet ported fails explicitly rather than silently diverging.
- [x] Pair remains usable after merge through the existing public command.

## Plan

- [x] Select the launcher slice from #73.
- [x] Extract pure decision models and tests.
- [x] Add fake-command process tests for the selected slice.
- [x] Implement the guarded Go path.
- [x] Document remaining shell-owned launcher behavior.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.60 impl=0.45
item: greenfield-go-module design=1.00 impl=0.45
item: skill-or-dispatcher design=0.30 impl=0.30
item: atlas-docs design=0.10 impl=0.05
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 3.75
```

## Log

### 2026-06-26

Created from #72. This issue is deliberately a prototype/vertical-slice milestone, not the public switch.

### 2026-06-29
- 2026-06-29: closed — go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; git diff -- bin/pair empty; rg atlas boundary check; git diff --check

Claimed #75 after parking #82. Entered planning with `sdlc start-plan --issue 75`; design cites #73's inventory and chooses a guarded `pair-go launch` decision-phase prototype so the shell launcher remains public while the Go path exercises real launcher concepts (`ARCH-PURE`, `ARCH-DRY`, `ARCH-PURPOSE`).

Plan-quality gate returned FAILURE: estimate was low for the visible multi-surface scope, and tag/session naming was ambiguous. Revised the estimate to 3.75 and clarified that canonical tags are bare while zellij session names are derived as `pair-<tag>`.

Second plan-quality gate returned FAILURE: the process-level test was ordered before the dispatcher route it needs, and the plan risked duplicating `dispatcher.Result`. Reordered route before process test and made `dispatcher.Result` the sole process-facing result abstraction (`ARCH-DRY`).

Third plan-quality gate returned FAILURE: Task 4 still named a `LaunchResult` return despite the single-result-abstraction rule. Revised the runner contract so launcher returns domain `LaunchOutcome` values and dispatcher alone maps to `dispatcher.Result`; also named the production IO constructor and test runtime seam.

Implemented `cmd/internal/launcher` as a pure decision-phase core plus fakeable zellij/history seams. `pair-go launch` now routes through the #74 dispatcher, returns explicit prototype decisions, and does not mutate zellij or replace `bin/pair`. Updated atlas architecture and the Go migration inventory to record the shell-owned boundary.

exec
/bin/zsh -lc 'git diff --stat e89e2cea484ab8ca7ec9ca518fd2bf04ba38a8fe..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 atlas/architecture.md                           |  13 ++-
 atlas/go-migration-inventory.md                 |   2 +-
 cmd/internal/dispatcher/dispatcher.go           |  93 +++++++++++++++++++++
 cmd/internal/dispatcher/dispatcher_test.go      |  36 +++++++++
 cmd/internal/launcher/args.go                   |  71 ++++++++++++++++
 cmd/internal/launcher/args_test.go              |  88 ++++++++++++++++++++
 cmd/internal/launcher/datadir.go                |  11 +++
 cmd/internal/launcher/datadir_test.go           |  21 +++++
 cmd/internal/launcher/decision.go               | 103 ++++++++++++++++++++++++
 cmd/internal/launcher/decision_test.go          |  67 +++++++++++++++
 cmd/internal/launcher/history.go                |  65 +++++++++++++++
 cmd/internal/launcher/history_test.go           |  33 ++++++++
 cmd/internal/launcher/run.go                    |  70 ++++++++++++++++
 cmd/internal/launcher/run_test.go               |  90 +++++++++++++++++++++
 cmd/internal/launcher/session.go                |  28 +++++++
 cmd/internal/launcher/tag.go                    |  42 ++++++++++
 cmd/internal/launcher/tag_test.go               |  53 ++++++++++++
 cmd/internal/launcher/zellij.go                 |  91 +++++++++++++++++++++
 cmd/internal/launcher/zellij_test.go            |  44 ++++++++++
 cmd/pair-go/launch_process_test.go              |  71 ++++++++++++++++
 cmd/pair-go/main.go                             |   9 +++
 cmd/pair-go/main_test.go                        |  40 +++++++++
 workshop/issues/000075-go-launcher-prototype.md |  22 ++---
 23 files changed, 1148 insertions(+), 15 deletions(-)

exec
/bin/zsh -lc 'git diff --name-only e89e2cea484ab8ca7ec9ca518fd2bf04ba38a8fe..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
atlas/architecture.md
atlas/go-migration-inventory.md
cmd/internal/dispatcher/dispatcher.go
cmd/internal/dispatcher/dispatcher_test.go
cmd/internal/launcher/args.go
cmd/internal/launcher/args_test.go
cmd/internal/launcher/datadir.go
cmd/internal/launcher/datadir_test.go
cmd/internal/launcher/decision.go
cmd/internal/launcher/decision_test.go
cmd/internal/launcher/history.go
cmd/internal/launcher/history_test.go
cmd/internal/launcher/run.go
cmd/internal/launcher/run_test.go
cmd/internal/launcher/session.go
cmd/internal/launcher/tag.go
cmd/internal/launcher/tag_test.go
cmd/internal/launcher/zellij.go
cmd/internal/launcher/zellij_test.go
cmd/pair-go/launch_process_test.go
cmd/pair-go/main.go
cmd/pair-go/main_test.go
workshop/issues/000075-go-launcher-prototype.md

codex
I’ve got the issue contract now. Next I’m comparing the Go prototype against the existing shell launcher behavior the spec says it is slicing, especially tag/session naming, resume, history scanning, and zellij snapshot handling.
exec
/bin/zsh -lc "sed -n '1,260p' bin/pair" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
#!/usr/bin/env bash
# pair — launch a zellij session split between a TUI coding agent (top pane)
# and an nvim drafting pane (bottom). See README.md.
#
# Usage:
#   pair                       claude in default session
#   pair <agent>               claude / codex / agy
#   pair [<agent>] -- <args>   forward args to agent on create,
#                              e.g. pair claude -- --resume
#                                   pair -- --dangerously-skip-permissions
#                                   pair codex -- -p "say hi"
#   pair -h, --help            show help
#
# When detached pair-* sessions exist, an fzf picker is shown so the user
# explicitly chooses what to do — no silent auto-attach. The picker shows
# ALL detached pair-* sessions regardless of agent argument; the agent
# argument only matters for the create path (sentinel label, default name,
# binary to exec).

set -euo pipefail

# Wrap zellij with a hard timeout for read-only IPC. The daemon socket can
# wedge if an earlier client got stuck (seen 2026-05-06: a `list-sessions`
# from hours prior held the socket and every new pair launch piled up behind
# it). pair makes several of these per launch, so one wedge cascades into
# unkillable bash trees. 5s is generous — healthy responses are <100ms; if
# it doesn't return by then, something's wrong and falling through is better
# than hanging. Long-running zellij invocations (attach, new-session) are
# intentionally NOT wrapped — they're meant to block.
zj() {
    local secs=5
    if command -v gtimeout >/dev/null 2>&1; then
        gtimeout "$secs" zellij "$@"
    elif command -v timeout >/dev/null 2>&1; then
        timeout "$secs" zellij "$@"
    else
        # Pure-bash fallback for systems without coreutils' timeout (default
        # macOS). Run zellij in the background, arm a watchdog, wait.
        #
        # The watchdog's stdio is redirected to /dev/null. Otherwise, in a
        # pipeline like `zj list-sessions | awk ...`, the watchdog subshell
        # inherits stdout = pipe-to-awk; when we SIGTERM it after zellij
        # exits, bash dies but the orphan `sleep` child keeps the pipe FD
        # open, so awk waits for EOF until the full $secs elapses. With N
        # piped zj calls in pair's startup, that compounds (we saw N×5s).
        zellij "$@" &
        local pid=$!
        ( sleep "$secs"; kill -9 "$pid" 2>/dev/null ) </dev/null >/dev/null 2>&1 &
        local watchdog=$!
        local rc=0
        wait "$pid" 2>/dev/null || rc=$?
        kill "$watchdog" 2>/dev/null || true
        wait "$watchdog" 2>/dev/null || true
        return $rc
    fi
}

# normalize_tag <raw> <errprefix> — accept a bare tag or `pair-<tag>`, strip the
# prefix, and validate the charset (letters/digits/dash/underscore). Echoes the
# cleaned tag on stdout; on an empty/invalid value prints "<errprefix> '<raw>'
# (allowed: …)" to stderr and returns 1. Callers: t="$(normalize_tag "$x" "…")" || exit 1.
# Shared by resume/continue/rename + the name prompt (#52 ARCH-DRY). It *returns*
# (never exits) so it composes inside $( ), where an exit would only kill the subshell.
normalize_tag() {
    local _raw="$1" _t="$1"
    case "$_t" in pair-*) _t="${_t#pair-}" ;; esac
    case "$_t" in
        ""|*[!A-Za-z0-9_-]*)
            printf "%s '%s' (allowed: letters, digits, dash, underscore)\n" "$2" "$_raw" >&2
            return 1 ;;
    esac
    printf '%s' "$_t"
}

# --- help -------------------------------------------------------------------
case "${1:-}" in
    -h|--help|help)
        cat <<'EOF'
pair — Neovim-backed input field for any TUI coding agent.

USAGE
  pair                          claude in default session
  pair <agent>                  e.g. pair codex / pair agy
  pair resume <tag>             attach pair-<tag> if it exists, else create
                                with that tag and the saved config; agent
                                is inferred from saved state
  pair continue [slug]          resume from a continuation doc; bare lists them
  pair [<agent>] -- <args>      forward args to the agent on create
                                e.g. pair claude -- --resume
                                     pair -- --dangerously-skip-permissions
                                     pair codex -- -p "say hi"
  pair list                     list pair-* sessions and their attach state
  pair rename <old> <new>       rename every tag-scoped file in
                                $PAIR_DATA_DIR from <old> to <new>.
                                Refuses if `pair-<old>` is still tracked
                                by zellij (use Ctrl+Alt+n's (R)ename
                                option for the in-session equivalent).
  pair -h, --help               this message

  Use `--` to separate pair's args from the agent's args. Without `--`,
  pair only accepts `<agent>` as a positional (or `resume <tag>`).

  DEV MODE
  `pair-dev` is identical to `pair` but rebuilds pair's Go binaries from
  source (`make build`) on launch AND on every Alt+n / Shift+Alt+N restart,
  so the zellij-spawned pair-wrap always matches the working tree. Use it
  when hacking on cmd/*; deployed installs keep running `pair` (prebuilt,
  no toolchain dependency). See bin/pair-dev + bin/lib/dev-rebuild.sh.

  When creating a new session you'll be prompted for a name; type any
  custom name (e.g. "bugfix") to make it independent of other sessions.
  `resume <tag>` skips that prompt — use it to restart a saved tag with
  its captured args (and the agent's session if still on disk).

  Agent args (after `--`) only apply on CREATE. Reattaching to an
  existing session uses whatever args the agent was originally started
  with — the picker doesn't re-launch the agent.

BEHAVIOR
  Picker shows ALL detached pair-* sessions, regardless of the agent
  argument. The agent argument only controls what gets created if you
  pick the "+ new" sentinel: it labels the sentinel, drives the auto-
  suggested name, and is the binary that gets exec'd.

    0 detached + 0 historical → create directly (prompts for name;
                                default is auto-generated like
                                pair-claude-2)
    ≥1 detached or ≥1 historical → fzf picker over: detached pair-*
                                sessions, then historical tags from
                                this cwd within $PAIR_HISTORY_DAYS
                                (default 14) annotated `(Nd ago, no
                                live session)`, then a `+ new <agent>
                                session` sentinel. Detached row →
                                attach. Historical row → create-by-name
                                (reuses saved draft / agent config).
                                Sentinel → fall through to create.

  Historical surface (#24) uses the cwd basename as prefix: tags named
  `<cwd-base>` or `<cwd-base>-<subproject>` whose draft/log sidecars
  were touched within the window. PAIR_DEBUG_HISTORY=1 prints the scan
  results and exits without launching zellij.

  No silent attach. Every reattach is explicit.

KEYBINDINGS (inside a pair session)
  Alt+Return    send nvim buffer to agent + Enter, log, clear, insert mode
  Alt+Shift+Return
                append nvim buffer to agent composer + newline, no submit;
                log + clear (like Alt+Return but unsubmitted)
  Alt+c         open/show/hide the review collaboration pane; if no review
                target exists, starts :PairReview
  Ctrl+C        send ESC (0x1b) to agent pane — interrupts claude's
                in-flight stream without leaving the draft
  Alt+← / Alt+→ walk through prompt history (-N) ↔ draft (*) ↔ queue (+N)
  Shift+Alt+← / Shift+Alt+→
                jump to next region boundary (oldest-history / newest-history /
                * / front-of-queue / back-of-queue)
  Alt+1..Alt+9  (insert mode, completion popup visible) quick-pick the Nth
                visible completion item. Items show their index (e.g.
                `1 bin/pair-wrap`); past 9, use arrows or <C-n>/<C-p>.
  Alt+q         push current buffer to queue front (+1)
                from *: parks the draft. from -N: forks. from +N: move-to-front
  Alt+BS        (at +N) delete the current queued prompt; items shift down,
                so tap-tap deletes a run
  Shift+Alt+BS  erase history + draft + queue ("start anew"); confirms first
  Alt+↑ / Alt+↓ step nvim pane along minimized ↔ 12 lines ↔ 1/2 ladder
                (works from either pane). minimized = statusline only;
                Alt+↑ from there grows the pane back. Confirm-requiring
                keys (Alt+x/d/n/N) auto-grow out of minimized first so
                the prompt is visible.
  Alt+d         detach from session (re-attach later via `pair`)
  Alt+x         full quit — kill session and remove from resurrect list
                (Ctrl+q is unbound; Alt+x is the only quit path)
  Alt+n         reload pair — kill the session and re-launch in place
                with the same tag, agent, args, AND agent session.
                The conversation resumes; pair itself is the only
                thing that restarts (useful after a config or binary
                rebuild). Ctrl+Alt+n is a bound alias — use it on macOS
                terminal combos where Option+n triggers the dead-tilde
                composer instead of forwarding as Meta+n.
  Shift+Alt+N   restart with a fresh agent conversation — same tag,
                agent, and args, but the saved per-(tag,agent) config
                is dropped so the agent starts a brand-new session
  Alt+h         pop up this help in a floating pane (press `q` to dismiss)
  Alt+i         attach clipboard image to agent + insert whatever marker
                the agent renders (e.g. claude `[Image #N]`). Captured
                from pair-wrap's PTY stream so no per-agent format
                knowledge is baked in. Flashes
                "[no image in clipboard]" at cursor for 1s if the OS
                clipboard has no image data.
  Alt+/         open agent pane scrollback in a read-only ANSI-colored
                nvim viewer; line numbers match zellij's frame indicator,
                so e.g. `:880` jumps to the line you saw scrolled to 880.
                Inside the viewer:
                  • Alt+q drops a 🤖[comment] marker — bare in normal
                    mode (the line is the context) or 🤖<selection>
                    [comment] in visual mode.
                  • Alt+b / Alt+B jump to previous / next prompt
                    boundary, so you can hop between turns instead of
                    scrolling line-by-line.
                  • Search is smart-case: `/foo` matches Foo/FOO/foo,
                    `/Foo` only Foo.
                  • <Esc> exits. If markers (or an overall comment) are
                    pending, a Yes/No confirm runs first; on accept,
                    everything is formatted (`> <quote>\n<comment>`)
                    and appended to the draft for review + send.

  Status line shape: `Alt: <- history H < pos[*] [hint] > Q queued ->`
  — H/Q are total counts, pos is `*`/`-N`/`+N`. A trailing `*` on `-N`
  means a pending fork on that history slot; navigating away prompts:
  (S)end, (Q)ueue, (D)iscard, [S]tay. The `[hint]` is contextual:
  `[q=queue]` on */-N, `[⌫=del]` on +N.

DRAFT SYNTAX
  ===  comment  Whole lines starting with `===` are stripped at send time
                but kept in draft/queue/log. Use them to annotate why a
                queued prompt exists. Mid-line `===` is unaffected.

FILES
  ${XDG_DATA_HOME:-~/.local/share}/pair/draft-<tag>.md     persistent draft (*)
  ${XDG_DATA_HOME:-~/.local/share}/pair/log-<tag>.md       prompt history (-N)
  ${XDG_DATA_HOME:-~/.local/share}/pair/queue-<tag>/*.md   queued prompts (+N)

See README.md for image-paste recipes and other details.
EOF
        exit 0
        ;;
    list|ls)
        if ! command -v zellij >/dev/null 2>&1; then
            echo "pair: zellij not found on PATH." >&2
            exit 1
        fi
        # Snapshot session state once. --no-formatting carries the EXITED
        # annotation; --short gives clean newline-separated names.
        raw=$( { zj list-sessions --no-formatting 2>/dev/null || true; } )
        sessions=$( { zj list-sessions --short 2>/dev/null || true; } \
                    | awk '/^pair-/' \
                    | sort )
        if [ -z "$sessions" ]; then
            echo "no pair sessions"
            exit 0
        fi
        list_data_dir="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
        mkdir -p "$list_data_dir"

        # Backfill agent-<tag> for any running pair-wrap process whose tag
        # we don't yet have on disk — covers sessions created before the
        # agent-tracking feature shipped. Read PAIR_TAG from each process's
        # environ (macOS: `ps -E`; Linux: /proc/<pid>/environ) and the
        # agent name from argv (the token after `pair-wrap`).
        pw_pids=$(pgrep -f 'pair-wrap' 2>/dev/null || true)
        for pid in $pw_pids; do
            argv=$(ps -p "$pid" -o command= 2>/dev/null || true)
            [ -z "$argv" ] && continue
            agent=$(printf '%s\n' "$argv" | awk '
                { for (i=1; i<=NF; i++) if ($i ~ /pair-wrap$/) { print $(i+1); exit } }')
            [ -z "$agent" ] && continue
            if [ -r "/proc/$pid/environ" ]; then
                env_blob=$(tr '\0' '\n' < "/proc/$pid/environ" 2>/dev/null || true)
            else

exec
/bin/zsh -lc "sed -n '260,560p' bin/pair" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
            else
                env_blob=$( { ps -wwEp "$pid" -o command= 2>/dev/null \
                              || ps eww -p "$pid" 2>/dev/null \
                              || true; } )
            fi
            tag=$(printf '%s' "$env_blob" \
                  | tr ' \0' '\n\n' \
                  | awk -F= '$1=="PAIR_TAG"{print $2; exit}')
            [ -z "$tag" ] && continue
            if [ ! -f "$list_data_dir/agent-${tag}" ]; then
                printf '%s\n' "$agent" > "$list_data_dir/agent-${tag}"
            fi
        done

        printf '%-30s %-10s %s\n' "SESSION" "AGENT" "STATUS"
        while IFS= read -r sess; do
            [ -z "$sess" ] && continue
            tag="${sess#pair-}"
            agent_file="$list_data_dir/agent-${tag}"
            if [ -f "$agent_file" ]; then
                agent=$(head -n1 "$agent_file" | tr -d '\r\n[:space:]')
                [ -z "$agent" ] && agent="?"
            else
                agent="?"
            fi
            if printf '%s\n' "$raw" \
                 | grep -E "^${sess}[[:space:]].*EXITED" >/dev/null 2>&1; then
                status="exited"
            else
                # `|| true` so pipefail+errexit doesn't abort when zellij
                # can't reach the session (e.g. running outside any zellij
                # client); empty input through tail/wc yields 0.
                clients=$( { zj --session "$sess" action list-clients 2>/dev/null || true; } \
                          | tail -n +2 | wc -l | tr -d ' ')
                clients=${clients:-0}
                if [ "$clients" -eq 0 ]; then
                    status="detached"
                elif [ "$clients" -eq 1 ]; then
                    status="attached (1 client)"
                else
                    status="attached ($clients clients)"
                fi
            fi
            printf '%-30s %-10s %s\n' "$sess" "$agent" "$status"
        done <<< "$sessions"
        exit 0
        ;;
    rename)
        # `pair rename <old> <new>` — rename every tag-scoped file in
        # $PAIR_DATA_DIR from <old> to <new>. Offline-only: refuses if
        # `pair-<old>` is still in zellij's session list (live, detached,
        # or resurrectable). The inside-session rename gesture (M2) takes
        # care of the quit→rename→re-exec choreography on top of this
        # primitive; the only safe live equivalent is to use that.
        #
        # File enumeration is by exact-name construction, never globbing —
        # that way `pair rename brain newname` can't accidentally pick up
        # `*-brain-2-*` files. The set of agent suffixes (claude/codex/
        # agy) is hardcoded below; if a new agent is added to pair,
        # add it there too.
        # Optional flags before the positionals.
        #   --restart-check : validate everything but skip the `pair-<old>`
        #                     live-session refusal AND skip the actual mv.
        #                     Used by the in-session rename gesture
        #                     (Ctrl+Alt+n's (R)ename), where pair-<old> IS
        #                     the current session — about to be killed by
        #                     pair-restart.sh — so its presence in
        #                     `zellij list-sessions` is expected, not a
        #                     reason to refuse.
        rename_check_only=0
        rename_skip_old_live=0
        rename_argi=2
        while :; do
            case "${!rename_argi:-}" in
                --restart-check)
                    rename_check_only=1
                    rename_skip_old_live=1
                    rename_argi=$((rename_argi + 1))
                    ;;
                --) rename_argi=$((rename_argi + 1)); break ;;
                *)  break ;;
            esac
        done
        old_tag="${!rename_argi:-}"
        rename_argi=$((rename_argi + 1))
        new_tag="${!rename_argi:-}"
        rename_argi=$((rename_argi + 1))
        if [ -z "$old_tag" ] || [ -z "$new_tag" ]; then
            echo "usage: pair rename [--restart-check] <old> <new>" >&2
            exit 1
        fi
        if [ -n "${!rename_argi:-}" ]; then
            echo "pair rename: unexpected arg '${!rename_argi}'" >&2
            exit 1
        fi
        # Accept either bare tag or `pair-<tag>`; strip + charset-validate via the
        # shared helper (#52 ARCH-DRY). Length stays a rename-specific guard below.
        old_tag="$(normalize_tag "$old_tag" "pair rename: invalid tag")" || exit 1
        new_tag="$(normalize_tag "$new_tag" "pair rename: invalid tag")" || exit 1
        for t in "$old_tag" "$new_tag"; do
            if [ ${#t} -gt 256 ]; then
                echo "pair rename: tag '$t' is too long (max 256)" >&2
                exit 1
            fi
        done
        if [ "$old_tag" = "$new_tag" ]; then
            echo "pair rename: old and new tag are the same ('$old_tag')" >&2
            exit 1
        fi
        rename_dd="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
        if [ ! -d "$rename_dd" ]; then
            echo "pair rename: data dir not found: $rename_dd" >&2
            exit 1
        fi

        # Zellij session checks. Skipped if zellij isn't on PATH (e.g.,
        # offline housekeeping from a machine that never had it installed)
        # — refusal is for safety, not gatekeeping.
        if command -v zellij >/dev/null 2>&1; then
            rename_sessions=$( { zj list-sessions --short 2>/dev/null || true; } )
            if [ "$rename_skip_old_live" -eq 0 ] \
               && printf '%s\n' "$rename_sessions" | grep -qx "pair-${old_tag}"; then
                echo "pair rename: session 'pair-${old_tag}' is still tracked by zellij." >&2
                echo "             Quit it first (Alt+x), or use the in-session" >&2
                echo "             rename via the Ctrl+Alt+n restart confirm." >&2
                exit 1
            fi
            if printf '%s\n' "$rename_sessions" | grep -qx "pair-${new_tag}"; then
                echo "pair rename: session 'pair-${new_tag}' already exists in zellij." >&2
                exit 1
            fi
        fi

        # Enumerate every candidate path for a given tag. Tag-only families
        # use exact-name construction; per-(tag,agent) families iterate the
        # known agent set. Output: one absolute path per line.
        rename_paths_for() {
            local tag="$1" dd="$2" fam
            for fam in outer-tty pair-wrap-pid title-pid \
                       agent agent-pid agent-output agent-picks \
                       layout-mode queue quote image-capture; do
                printf '%s\n' "$dd/$fam-$tag"
            done
            printf '%s\n' "$dd/image-capture-$tag.done"
            printf '%s\n' "$dd/draft-$tag.md"
            printf '%s\n' "$dd/log-$tag.md"
            printf '%s\n' "$dd/nvim-pid-$tag-draft"
            printf '%s\n' "$dd/nvim-pid-$tag-scrollback"
            for a in claude codex agy; do
                printf '%s\n' "$dd/config-$tag-$a.json"
                printf '%s\n' "$dd/pane-$tag-$a.json"
                printf '%s\n' "$dd/scrollback-$tag-$a.ansi"
                printf '%s\n' "$dd/scrollback-$tag-$a.raw"
                printf '%s\n' "$dd/scrollback-$tag-$a.viewport"
                printf '%s\n' "$dd/scrollback-$tag-$a.events.jsonl"
                printf '%s\n' "$dd/draft-$tag-$a.md"
            done
        }

        # NEW must be empty of any tag-scoped files. If any exist, that
        # tag is in use (live, detached, or stale-after-quit) and a
        # rename onto it would conflict.
        rename_new_occupied=""
        while IFS= read -r p; do
            if [ -e "$p" ]; then
                rename_new_occupied="$p"
                break
            fi
        done < <(rename_paths_for "$new_tag" "$rename_dd")
        if [ -n "$rename_new_occupied" ]; then
            echo "pair rename: tag '$new_tag' is occupied — '$rename_new_occupied' exists." >&2
            exit 1
        fi

        # Build (src TAB dst) plan, in stable enumeration order.
        rename_plan=""
        rename_count=0
        old_paths_tmp="$(mktemp "${TMPDIR:-/tmp}/pair-rename.XXXXXX")"
        rename_paths_for "$old_tag" "$rename_dd" > "$old_paths_tmp"
        while IFS= read -r src; do
            [ -e "$src" ] || continue
            base="${src##*/}"
            # Substitute the tag chunk: the path was constructed by
            # rename_paths_for as "<dir>/<chunk-with-$old_tag>", so the
            # transform is deterministic.
            case "$base" in
                # tag-only families (suffix is "")
                outer-tty-$old_tag|pair-wrap-pid-$old_tag|title-pid-$old_tag|\
                agent-$old_tag|agent-pid-$old_tag|agent-output-$old_tag|\
                agent-picks-$old_tag|layout-mode-$old_tag|queue-$old_tag|\
                quote-$old_tag|image-capture-$old_tag)
                    dst="$rename_dd/${base%-"$old_tag"}-$new_tag"
                    ;;
                image-capture-$old_tag.done)
                    dst="$rename_dd/image-capture-$new_tag.done"
                    ;;
                draft-$old_tag.md)
                    dst="$rename_dd/draft-$new_tag.md"
                    ;;
                log-$old_tag.md)
                    dst="$rename_dd/log-$new_tag.md"
                    ;;
                nvim-pid-$old_tag-draft|nvim-pid-$old_tag-scrollback)
                    dst="$rename_dd/nvim-pid-$new_tag-${base##*-}"
                    ;;
                config-$old_tag-*.json)
                    suf="${base#config-$old_tag-}"
                    dst="$rename_dd/config-$new_tag-$suf"
                    ;;
                pane-$old_tag-*.json)
                    suf="${base#pane-$old_tag-}"
                    dst="$rename_dd/pane-$new_tag-$suf"
                    ;;
                scrollback-$old_tag-*)
                    suf="${base#scrollback-$old_tag-}"
                    dst="$rename_dd/scrollback-$new_tag-$suf"
                    ;;
                draft-$old_tag-*.md)
                    suf="${base#draft-$old_tag-}"
                    dst="$rename_dd/draft-$new_tag-$suf"
                    ;;
                *)
                    echo "pair rename: internal error: unrecognised src '$src'" >&2
                    rm -f "$old_paths_tmp"
                    exit 1
                    ;;
            esac
            if [ -e "$dst" ]; then
                echo "pair rename: destination already exists: $dst" >&2
                rm -f "$old_paths_tmp"
                exit 1
            fi
            rename_plan="${rename_plan}${src}	${dst}
"
            rename_count=$((rename_count + 1))
        done < "$old_paths_tmp"
        rm -f "$old_paths_tmp"

        if [ "$rename_count" -eq 0 ]; then
            echo "pair rename: no files found for tag '$old_tag' in $rename_dd" >&2
            exit 1
        fi

        # --restart-check returns after full validation (incl. the
        # collision-free dst plan) without touching disk. The actual
        # rename runs later, post-kill, from handle_restart_marker.
        if [ "$rename_check_only" -eq 1 ]; then
            echo "pair rename: ok ($rename_count file(s) would move from '$old_tag' to '$new_tag')"
            exit 0
        fi

        # Journal the plan to disk before executing so a crash-mid-rename
        # leaves a forensic breadcrumb for M3's recovery path.
        rename_journal="$rename_dd/.rename-$old_tag-to-$new_tag.journal"
        printf '%s' "$rename_plan" > "$rename_journal"

        echo "pair rename: $rename_count file(s) ${old_tag} → ${new_tag}"
        rename_done=0
        rename_failed=""
        while IFS=$'\t' read -r src dst; do
            [ -z "$src" ] && continue
            if mv "$src" "$dst"; then
                rename_done=$((rename_done + 1))
            else
                rename_failed="$src → $dst"
                break
            fi
        done <<EOF
$rename_plan
EOF

        if [ -n "$rename_failed" ]; then
            echo "pair rename: mv failed: $rename_failed" >&2
            echo "pair rename: rolling back $rename_done completed rename(s)..." >&2
            # Reverse the journal lines we've already executed and mv back.
            head -n "$rename_done" "$rename_journal" \
                | awk -F'\t' 'NF { printf "%s\t%s\n", $2, $1 }' \
                | while IFS=$'\t' read -r r_src r_dst; do
                    mv "$r_src" "$r_dst" 2>/dev/null || true
                done
            # Keep journal on rollback failure so it's diagnostic.
            exit 1
        fi
        rm -f "$rename_journal"
        echo "pair rename: ok"
        exit 0
        ;;
esac

# Resolve this script's real path portably (macOS lacks `readlink -f`).
SOURCE="${BASH_SOURCE[0]}"
while [ -L "$SOURCE" ]; do
    DIR="$(cd -P "$(dirname "$SOURCE")" && pwd)"
    SOURCE="$(readlink "$SOURCE")"
    [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
done
PAIR_HOME="$(cd -P "$(dirname "$SOURCE")/.." && pwd)"
export PAIR_HOME

# Put pair's bin/ on PATH so zellij can resolve our helper scripts by bare
# name (clipboard-to-pane.sh, copy-on-select.sh) without sh-c env-var hacks

exec
/bin/zsh -lc 'nl -ba cmd/internal/launcher/zellij.go' in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc 'nl -ba cmd/internal/launcher/decision.go' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package launcher
     2	
     3	import (
     4		"bytes"
     5		"os/exec"
     6		"sort"
     7		"strings"
     8	)
     9	
    10	// ZellijSource reads zellij session state through the zellij CLI.
    11	type ZellijSource struct {
    12		Path string
    13	}
    14	
    15	func (s ZellijSource) Snapshot() ([]Session, error) {
    16		short, err := s.run("list-sessions", "--short")
    17		if err != nil {
    18			short = nil
    19		}
    20		raw, err := s.run("list-sessions", "--no-formatting")
    21		if err != nil {
    22			raw = nil
    23		}
    24		exited := exitedSessions(string(raw))
    25		var out []Session
    26		for _, name := range lines(string(short)) {
    27			if !strings.HasPrefix(name, "pair-") {
    28				continue
    29			}
    30			state := SessionDetached
    31			if exited[name] {
    32				state = SessionExited
    33			} else if s.clientCount(name) > 0 {
    34				state = SessionAttached
    35			}
    36			out = append(out, Session{Name: name, State: state})
    37		}
    38		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
    39		return out, nil
    40	}
    41	
    42	func (s ZellijSource) clientCount(session string) int {
    43		out, err := s.run("--session", session, "action", "list-clients")
    44		if err != nil {
    45			return 0
    46		}
    47		lines := lines(string(out))
    48		if len(lines) <= 1 {
    49			return 0
    50		}
    51		return len(lines) - 1
    52	}
    53	
    54	func (s ZellijSource) run(args ...string) ([]byte, error) {
    55		path := s.Path
    56		if path == "" {
    57			path = "zellij"
    58		}
    59		cmd := exec.Command(path, args...)
    60		var stdout bytes.Buffer
    61		cmd.Stdout = &stdout
    62		if err := cmd.Run(); err != nil {
    63			return nil, err
    64		}
    65		return stdout.Bytes(), nil
    66	}
    67	
    68	func exitedSessions(raw string) map[string]bool {
    69		out := map[string]bool{}
    70		for _, line := range lines(raw) {
    71			fields := strings.Fields(line)
    72			if len(fields) == 0 {
    73				continue
    74			}
    75			if strings.Contains(line, "EXITED") {
    76				out[fields[0]] = true
    77			}
    78		}
    79		return out
    80	}
    81	
    82	func lines(s string) []string {
    83		var out []string
    84		for _, line := range strings.Split(s, "\n") {
    85			line = strings.TrimSpace(line)
    86			if line != "" {
    87				out = append(out, line)
    88			}
    89		}
    90		return out
    91	}

 succeeded in 0ms:
     1	package launcher
     2	
     3	// LaunchAction is the guarded prototype's next launcher action.
     4	type LaunchAction string
     5	
     6	const (
     7		ActionAttach LaunchAction = "attach"
     8		ActionCreate LaunchAction = "create"
     9		ActionPick   LaunchAction = "pick"
    10	)
    11	
    12	// LaunchDecision is a pure create/attach/pick decision. Tag is canonical bare
    13	// form; SessionName is derived as pair-<tag> when a zellij session is named.
    14	type LaunchDecision struct {
    15		Action      LaunchAction
    16		Tag         string
    17		SessionName string
    18		PromptName  bool
    19	}
    20	
    21	// DecideLaunch decides the launch action without touching zellij, fzf, or disk.
    22	func DecideLaunch(args LaunchArgs, snap SessionSnapshot) (LaunchDecision, error) {
    23		if args.SelectedTag != "" {
    24			return createDecision(args.SelectedTag, false), nil
    25		}
    26		if args.ForcedTag != "" {
    27			if sessionBlocksReuse(snap, sessionName(args.ForcedTag)) {
    28				return LaunchDecision{Action: ActionAttach, Tag: args.ForcedTag, SessionName: sessionName(args.ForcedTag)}, nil
    29			}
    30			return createDecision(args.ForcedTag, false), nil
    31		}
    32		if hasDetached(snap) || len(snap.Historical) > 0 {
    33			return LaunchDecision{Action: ActionPick}, nil
    34		}
    35		tag := snap.BaseTag
    36		if tag == "" {
    37			tag = "pair"
    38		}
    39		return createDecision(nextFreeTag(tag, snap), true), nil
    40	}
    41	
    42	func createDecision(tag string, prompt bool) LaunchDecision {
    43		return LaunchDecision{Action: ActionCreate, Tag: tag, SessionName: sessionName(tag), PromptName: prompt}
    44	}
    45	
    46	func sessionName(tag string) string {
    47		return "pair-" + tag
    48	}
    49	
    50	func hasDetached(snap SessionSnapshot) bool {
    51		for _, sess := range snap.Sessions {
    52			if sess.State == SessionDetached {
    53				return true
    54			}
    55		}
    56		return false
    57	}
    58	
    59	func sessionBlocksReuse(snap SessionSnapshot, name string) bool {
    60		for _, sess := range snap.Sessions {
    61			if sess.Name != name {
    62				continue
    63			}
    64			return sess.State == SessionAttached || sess.State == SessionDetached
    65		}
    66		return false
    67	}
    68	
    69	func nextFreeTag(base string, snap SessionSnapshot) string {
    70		for i := 1; i <= 100; i++ {
    71			tag := base
    72			if i > 1 {
    73				tag = base + "-" + itoa(i)
    74			}
    75			if !sessionBlocksReuse(snap, sessionName(tag)) && !isHistorical(snap, tag) {
    76				return tag
    77			}
    78		}
    79		return base
    80	}
    81	
    82	func isHistorical(snap SessionSnapshot, tag string) bool {
    83		for _, hist := range snap.Historical {
    84			if hist.Tag == tag {
    85				return true
    86			}
    87		}
    88		return false
    89	}
    90	
    91	func itoa(n int) string {
    92		if n == 0 {
    93			return "0"
    94		}
    95		var buf [20]byte
    96		i := len(buf)
    97		for n > 0 {
    98			i--
    99			buf[i] = byte('0' + n%10)
   100			n /= 10
   101		}
   102		return string(buf[i:])
   103	}

exec
/bin/zsh -lc 'nl -ba cmd/internal/launcher/history.go' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package launcher
     2	
     3	import (
     4		"os"
     5		"path/filepath"
     6		"sort"
     7		"strings"
     8		"time"
     9	)
    10	
    11	// HistorySource scans Pair draft/log sidecars under the data directory.
    12	type HistorySource struct {
    13		DataDir string
    14	}
    15	
    16	func (s HistorySource) Scan(base string, cutoff time.Time) ([]HistoricalTag, error) {
    17		latest := map[string]time.Time{}
    18		for _, pattern := range []string{"draft-*.md", "log-*.md"} {
    19			matches, err := filepath.Glob(filepath.Join(s.DataDir, pattern))
    20			if err != nil {
    21				return nil, err
    22			}
    23			for _, path := range matches {
    24				tag, ok := tagFromSidecar(filepath.Base(path))
    25				if !ok || !matchesHistoryBase(tag, base) {
    26					continue
    27				}
    28				info, err := os.Stat(path)
    29				if err != nil {
    30					continue
    31				}
    32				if info.ModTime().Before(cutoff) {
    33					continue
    34				}
    35				if info.ModTime().After(latest[tag]) {
    36					latest[tag] = info.ModTime()
    37				}
    38			}
    39		}
    40		tags := make([]string, 0, len(latest))
    41		for tag := range latest {
    42			tags = append(tags, tag)
    43		}
    44		sort.Strings(tags)
    45		out := make([]HistoricalTag, 0, len(tags))
    46		for _, tag := range tags {
    47			out = append(out, HistoricalTag{Tag: tag})
    48		}
    49		return out, nil
    50	}
    51	
    52	func tagFromSidecar(name string) (string, bool) {
    53		switch {
    54		case strings.HasPrefix(name, "draft-") && strings.HasSuffix(name, ".md"):
    55			return strings.TrimSuffix(strings.TrimPrefix(name, "draft-"), ".md"), true
    56		case strings.HasPrefix(name, "log-") && strings.HasSuffix(name, ".md"):
    57			return strings.TrimSuffix(strings.TrimPrefix(name, "log-"), ".md"), true
    58		default:
    59			return "", false
    60		}
    61	}
    62	
    63	func matchesHistoryBase(tag, base string) bool {
    64		return tag == base || strings.HasPrefix(tag, base+"-")
    65	}

exec
/bin/zsh -lc 'nl -ba cmd/internal/launcher/run.go' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package launcher
     2	
     3	import (
     4		"time"
     5	)
     6	
     7	// Env is the explicit runtime environment for launch decisions.
     8	type Env struct {
     9		Home     string
    10		XDGData  string
    11		Cwd      string
    12		Now      time.Time
    13		HistoryD int
    14		DataDir  string
    15	}
    16	
    17	// SessionSource supplies zellij session state.
    18	type SessionSource interface {
    19		Snapshot() ([]Session, error)
    20	}
    21	
    22	// HistoricalScanner supplies recently touched Pair tags.
    23	type HistoricalScanner interface {
    24		Scan(base string, cutoff time.Time) ([]HistoricalTag, error)
    25	}
    26	
    27	// LaunchOutcome is the domain outcome from the launcher core.
    28	type LaunchOutcome struct {
    29		Args     LaunchArgs
    30		Env      Env
    31		Decision LaunchDecision
    32	}
    33	
    34	// Run builds a pure snapshot from injected sources and returns a domain launch
    35	// outcome. The dispatcher maps this to process stdout/stderr/exit status.
    36	func Run(argv []string, env Env, sessions SessionSource, history HistoricalScanner) (LaunchOutcome, error) {
    37		args, err := ParseArgs(argv)
    38		if err != nil {
    39			return LaunchOutcome{}, err
    40		}
    41		if env.DataDir == "" {
    42			env.DataDir = ResolveDataDir(env.Home, env.XDGData)
    43		}
    44		if env.HistoryD == 0 {
    45			env.HistoryD = 14
    46		}
    47		if env.Now.IsZero() {
    48			env.Now = time.Now()
    49		}
    50	
    51		sessionRows, err := sessions.Snapshot()
    52		if err != nil {
    53			return LaunchOutcome{}, err
    54		}
    55		base := DefaultTag(env.Cwd)
    56		historical, err := history.Scan(base, env.Now.Add(-time.Duration(env.HistoryD)*24*time.Hour))
    57		if err != nil {
    58			return LaunchOutcome{}, err
    59		}
    60	
    61		decision, err := DecideLaunch(args, SessionSnapshot{
    62			BaseTag:    base,
    63			Sessions:   sessionRows,
    64			Historical: historical,
    65		})
    66		if err != nil {
    67			return LaunchOutcome{}, err
    68		}
    69		return LaunchOutcome{Args: args, Env: env, Decision: decision}, nil
    70	}

exec
/bin/zsh -lc 'nl -ba cmd/internal/launcher/args.go' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package launcher
     2	
     3	import "fmt"
     4	
     5	// LaunchArgs is the pure parse result for the guarded pair-go launch prototype.
     6	type LaunchArgs struct {
     7		Agent       string
     8		ForcedTag   string
     9		SelectedTag string
    10		AgentArgs   []string
    11	}
    12	
    13	// UsageError is an operator-facing parse error.
    14	type UsageError struct {
    15		Message string
    16	}
    17	
    18	func (e UsageError) Error() string {
    19		return e.Message
    20	}
    21	
    22	// ParseArgs parses pair-go launch args. It intentionally supports only the
    23	// decision-phase subset for #75; unsupported shell-owned launcher verbs fail
    24	// explicitly.
    25	func ParseArgs(argv []string) (LaunchArgs, error) {
    26		var out LaunchArgs
    27		if len(argv) == 0 {
    28			out.Agent = "claude"
    29			return out, nil
    30		}
    31	
    32		switch argv[0] {
    33		case "continue", "rename", "list", "ls":
    34			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %s is not implemented by pair-go launch; use bin/pair", argv[0])}
    35		case "resume":
    36			if len(argv) < 2 {
    37				return LaunchArgs{}, UsageError{Message: "pair-go launch: 'resume' requires a tag"}
    38			}
    39			tag, err := NormalizeTag(argv[1])
    40			if err != nil {
    41				return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: invalid tag: %v", err)}
    42			}
    43			if len(argv) > 2 {
    44				return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", argv[2])}
    45			}
    46			out.ForcedTag = tag
    47			return out, nil
    48		}
    49	
    50		seenSeparator := false
    51		for _, arg := range argv {
    52			if seenSeparator {
    53				out.AgentArgs = append(out.AgentArgs, arg)
    54				continue
    55			}
    56			if arg == "--" {
    57				seenSeparator = true
    58				continue
    59			}
    60			if out.Agent == "" {
    61				out.Agent = arg
    62				continue
    63			}
    64			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", arg)}
    65		}
    66	
    67		if out.Agent == "" {
    68			out.Agent = "claude"
    69		}
    70		return out, nil
    71	}

exec
/bin/zsh -lc 'rg -n "resume|historical|draft-|list-sessions|PAIR_HISTORY|next|tag|new_session|new session|fzf|session_name" bin/pair' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
9:#                              e.g. pair claude -- --resume
14:# When detached pair-* sessions exist, an fzf picker is shown so the user
23:# wedge if an earlier client got stuck (seen 2026-05-06: a `list-sessions`
41:        # pipeline like `zj list-sessions | awk ...`, the watchdog subshell
58:# normalize_tag <raw> <errprefix> — accept a bare tag or `pair-<tag>`, strip the
60:# cleaned tag on stdout; on an empty/invalid value prints "<errprefix> '<raw>'
61:# (allowed: …)" to stderr and returns 1. Callers: t="$(normalize_tag "$x" "…")" || exit 1.
62:# Shared by resume/continue/rename + the name prompt (#52 ARCH-DRY). It *returns*
64:normalize_tag() {
84:  pair resume <tag>             attach pair-<tag> if it exists, else create
85:                                with that tag and the saved config; agent
87:  pair continue [slug]          resume from a continuation doc; bare lists them
89:                                e.g. pair claude -- --resume
93:  pair rename <old> <new>       rename every tag-scoped file in
101:  pair only accepts `<agent>` as a positional (or `resume <tag>`).
110:  When creating a new session you'll be prompted for a name; type any
112:  `resume <tag>` skips that prompt — use it to restart a saved tag with
125:    0 detached + 0 historical → create directly (prompts for name;
128:    ≥1 detached or ≥1 historical → fzf picker over: detached pair-*
129:                                sessions, then historical tags from
130:                                this cwd within $PAIR_HISTORY_DAYS
138:  Historical surface (#24) uses the cwd basename as prefix: tags named
156:                jump to next region boundary (oldest-history / newest-history /
175:                with the same tag, agent, args, AND agent session.
176:                The conversation resumes; pair itself is the only
181:  Shift+Alt+N   restart with a fresh agent conversation — same tag,
182:                agent, and args, but the saved per-(tag,agent) config
183:                is dropped so the agent starts a brand-new session
198:                  • Alt+b / Alt+B jump to previous / next prompt
220:  ${XDG_DATA_HOME:-~/.local/share}/pair/draft-<tag>.md     persistent draft (*)
221:  ${XDG_DATA_HOME:-~/.local/share}/pair/log-<tag>.md       prompt history (-N)
222:  ${XDG_DATA_HOME:-~/.local/share}/pair/queue-<tag>/*.md   queued prompts (+N)
235:        raw=$( { zj list-sessions --no-formatting 2>/dev/null || true; } )
236:        sessions=$( { zj list-sessions --short 2>/dev/null || true; } \
246:        # Backfill agent-<tag> for any running pair-wrap process whose tag
265:            tag=$(printf '%s' "$env_blob" \
268:            [ -z "$tag" ] && continue
269:            if [ ! -f "$list_data_dir/agent-${tag}" ]; then
270:                printf '%s\n' "$agent" > "$list_data_dir/agent-${tag}"
277:            tag="${sess#pair-}"
278:            agent_file="$list_data_dir/agent-${tag}"
308:        # `pair rename <old> <new>` — rename every tag-scoped file in
327:        #                     `zellij list-sessions` is expected, not a
343:        old_tag="${!rename_argi:-}"
345:        new_tag="${!rename_argi:-}"
347:        if [ -z "$old_tag" ] || [ -z "$new_tag" ]; then
355:        # Accept either bare tag or `pair-<tag>`; strip + charset-validate via the
357:        old_tag="$(normalize_tag "$old_tag" "pair rename: invalid tag")" || exit 1
358:        new_tag="$(normalize_tag "$new_tag" "pair rename: invalid tag")" || exit 1
359:        for t in "$old_tag" "$new_tag"; do
361:                echo "pair rename: tag '$t' is too long (max 256)" >&2
365:        if [ "$old_tag" = "$new_tag" ]; then
366:            echo "pair rename: old and new tag are the same ('$old_tag')" >&2
379:            rename_sessions=$( { zj list-sessions --short 2>/dev/null || true; } )
381:               && printf '%s\n' "$rename_sessions" | grep -qx "pair-${old_tag}"; then
382:                echo "pair rename: session 'pair-${old_tag}' is still tracked by zellij." >&2
387:            if printf '%s\n' "$rename_sessions" | grep -qx "pair-${new_tag}"; then
388:                echo "pair rename: session 'pair-${new_tag}' already exists in zellij." >&2
393:        # Enumerate every candidate path for a given tag. Tag-only families
394:        # use exact-name construction; per-(tag,agent) families iterate the
397:            local tag="$1" dd="$2" fam
401:                printf '%s\n' "$dd/$fam-$tag"
403:            printf '%s\n' "$dd/image-capture-$tag.done"
404:            printf '%s\n' "$dd/draft-$tag.md"
405:            printf '%s\n' "$dd/log-$tag.md"
406:            printf '%s\n' "$dd/nvim-pid-$tag-draft"
407:            printf '%s\n' "$dd/nvim-pid-$tag-scrollback"
409:                printf '%s\n' "$dd/config-$tag-$a.json"
410:                printf '%s\n' "$dd/pane-$tag-$a.json"
411:                printf '%s\n' "$dd/scrollback-$tag-$a.ansi"
412:                printf '%s\n' "$dd/scrollback-$tag-$a.raw"
413:                printf '%s\n' "$dd/scrollback-$tag-$a.viewport"
414:                printf '%s\n' "$dd/scrollback-$tag-$a.events.jsonl"
415:                printf '%s\n' "$dd/draft-$tag-$a.md"
419:        # NEW must be empty of any tag-scoped files. If any exist, that
420:        # tag is in use (live, detached, or stale-after-quit) and a
428:        done < <(rename_paths_for "$new_tag" "$rename_dd")
430:            echo "pair rename: tag '$new_tag' is occupied — '$rename_new_occupied' exists." >&2
438:        rename_paths_for "$old_tag" "$rename_dd" > "$old_paths_tmp"
442:            # Substitute the tag chunk: the path was constructed by
443:            # rename_paths_for as "<dir>/<chunk-with-$old_tag>", so the
446:                # tag-only families (suffix is "")
447:                outer-tty-$old_tag|pair-wrap-pid-$old_tag|title-pid-$old_tag|\
448:                agent-$old_tag|agent-pid-$old_tag|agent-output-$old_tag|\
449:                agent-picks-$old_tag|layout-mode-$old_tag|queue-$old_tag|\
450:                quote-$old_tag|image-capture-$old_tag)
451:                    dst="$rename_dd/${base%-"$old_tag"}-$new_tag"
453:                image-capture-$old_tag.done)
454:                    dst="$rename_dd/image-capture-$new_tag.done"
456:                draft-$old_tag.md)
457:                    dst="$rename_dd/draft-$new_tag.md"
459:                log-$old_tag.md)
460:                    dst="$rename_dd/log-$new_tag.md"
462:                nvim-pid-$old_tag-draft|nvim-pid-$old_tag-scrollback)
463:                    dst="$rename_dd/nvim-pid-$new_tag-${base##*-}"
465:                config-$old_tag-*.json)
466:                    suf="${base#config-$old_tag-}"
467:                    dst="$rename_dd/config-$new_tag-$suf"
469:                pane-$old_tag-*.json)
470:                    suf="${base#pane-$old_tag-}"
471:                    dst="$rename_dd/pane-$new_tag-$suf"
473:                scrollback-$old_tag-*)
474:                    suf="${base#scrollback-$old_tag-}"
475:                    dst="$rename_dd/scrollback-$new_tag-$suf"
477:                draft-$old_tag-*.md)
478:                    suf="${base#draft-$old_tag-}"
479:                    dst="$rename_dd/draft-$new_tag-$suf"
499:            echo "pair rename: no files found for tag '$old_tag' in $rename_dd" >&2
507:            echo "pair rename: ok ($rename_count file(s) would move from '$old_tag' to '$new_tag')"
513:        rename_journal="$rename_dd/.rename-$old_tag-to-$new_tag.journal"
516:        echo "pair rename: $rename_count file(s) ${old_tag} → ${new_tag}"
575:#   pair claude -- --resume               → claude --resume
576:#   pair -- --resume                      → claude --resume
580:forced_tag=""
582:# `pair resume <tag>` is the documented restart path: skip both the picker
583:# and the name prompt, attach if pair-<tag> exists, otherwise create with
584:# that tag. We claim `resume` as a subcommand verb — it can't be a real
587:if [ "${1:-}" = "resume" ]; then
589:        echo "pair: 'resume' requires a tag (e.g. pair resume bugfix)" >&2
592:    forced_tag="$(normalize_tag "$2" "pair: invalid tag")" || exit 1
596:# `pair continue [slug] [agent]` — the human-understanding sibling of `resume`
599:# tag — like `pair <agent> -- <args>` it flows through the normal name prompt
600:# so the operator picks the tag (a long slug must never become an over-long
605:# `continue` never reads session_id — that's `resume`.
606:# Fresh-vs-attach (#52): the seeded session is "fresh" only for a NEW tag. If the
607:# tag picked at the prompt collides with a LIVE `pair-<tag>`, the normal launch
608:# path ATTACHES to it (same as `pair <agent>` / `resume`) rather than reseeding —
609:# intentional, so a continue never clobbers a running session; pick a fresh tag
620:                _cn="$(awk '/^## NEXT ACTION/{f=1;next} f&&NF{print;exit}' "$_cf")"
631:    _cslug="$(normalize_tag "$2" "pair: invalid slug")" || exit 1
637:    # No forced_tag (see header). Consume `continue` + slug; an optional next
660:    elif [ -z "$AGENT" ] && [ -z "$forced_tag" ]; then
664:        echo "      e.g. pair claude -- --resume" >&2
670:# (handle_restart_marker below): it pins the new run to the same tag as
672:# Treated as forced_tag from here on; unset so a child process doesn't
674:if [ -z "$forced_tag" ] && [ -n "${PAIR_FORCE_TAG:-}" ]; then
675:    forced_tag="$PAIR_FORCE_TAG"
696:park_scrollback() {  # <tag> <agent> [--copy]
697:    local tag="$1" agent="$2" mode="${3:-}"
698:    local base="$DATA_DIR/scrollback-${tag}-${agent}"
702:    pbase="$DATA_DIR/parked-scrollback-${tag}-${ts}"
706:    : > "$DATA_DIR/parked-${tag}"
711:# or the #55 in-session compaction branch, and re-exec pair under the same tag.
727:    local r_tag r_agent r_new_session r_rename_to r_continue
728:    r_tag=$(awk -F= '$1=="tag"{print $2; exit}' "$marker")
730:    r_new_session=$(awk -F= '$1=="new_session"{print $2; exit}' "$marker")
735:    [ -z "$r_tag" ] && r_tag="$PAIR_TAG"
738:    # Inside-flow tag rename (#000022 M2). By this point the zellij kill
739:    # has completed and `pair-<r_tag>` is no longer in the sessions list,
741:    # we keep the old tag and continue restarting — the user is not left
744:        if "$0" rename "$r_tag" "$r_rename_to"; then
745:            r_tag="$r_rename_to"
747:            echo "pair: tag rename failed; restarting with original tag '$r_tag'." >&2
754:    cfg="$(resolve_config_file "$r_tag" "$r_agent")"
760:    export PAIR_FORCE_TAG="$r_tag"
762:    if [ "$r_new_session" = "1" ]; then
763:        # Fresh conversation: drop the config so the next launch doesn't
770:            # the new_session arm, which never applies the resume-token reorder.
781:    # Default Alt+n: keep config, append the resume token so the new
784:    # `--resume <id>` flag, codex uses `resume <id>`, and agy uses
786:    local resume_extra=""
789:            claude)        resume_extra="--resume $r_sid" ;;
790:            codex)         resume_extra="resume $r_sid" ;;
791:            agy)           resume_extra="--conversation $r_sid" ;;
794:    # Composition order matters for codex: its `resume` subcommand must
797:    # follow. claude's --resume flag works anywhere, so the
800:    if [ "$r_agent" = "codex" ] && [ -n "$resume_extra" ]; then
801:        combined="${resume_extra}${r_args:+ $r_args}"
803:        combined="${r_args}${r_args:+${resume_extra:+ }}${resume_extra}"
814:# workspace tag. Alt+x can leave a resurrect record like
816:# `list-sessions`, so the old open-coded `grep -qx` collision checks treated a
817:# fully-quit tag as occupied. Treat an EXITED row as stale full-quit residue:
818:# delete the zellij record and report the tag reusable (non-zero). A
820:# Shared by forced resume, the free-slot scan, the name-prompt collision, and
826:    row=$( { zj list-sessions --no-formatting 2>/dev/null || true; } \
837:# #67: resolve the canonical saved-config path for a known (tag, agent), echoing
838:# it on stdout. Current launches write `config-<tag>-<agent>.json`, but older
839:# Codex sessions on disk use the doubled shape `config-<tag>-codex-codex.json`.
841:# declares `"agent": "codex"`, migrate it to the canonical name so the resume
842:# picker can offer native resume. This is a narrow, agent-checked compatibility
844:# (ARCH-DRY, ARCH-PURE). Used only where both tag and agent are already known;
848:    local tag="$1" agent="$2"
849:    local canonical="$DATA_DIR/config-${tag}-${agent}.json"
851:        local legacy="$DATA_DIR/config-${tag}-codex-codex.json"
861:# title mirroring pair's tag. CMUX_WORKSPACE_ID is set by cmux inside every
869:# records which tag last labeled this workspace. cmux_rename_workspace runs from
875:# background poller (bin/pair-title.sh) still defers to a live different-tag
914:# Defer the [-z AGENT] default to after DATA_DIR is set: when `resume <tag>`
915:# is used we infer the agent from saved state on disk so a single tag is
919:# default_tag drives the auto-suggestion in the create-flow name prompt
920:# and the picker's "+ new <tag> session" sentinel. The interactive prompt
921:# and `resume <tag>` both override this.
924:# The tag scopes sessions to a workspace, and per-agent variants of the
926:# files (config-<tag>-<agent>.json, scrollback-<tag>-<agent>.raw, ...).
927:default_tag="$(basename "$PWD")"
928:default_tag="${default_tag//[^A-Za-z0-9_-]/_}"
929:[ -z "$default_tag" ] && default_tag="pair"
943:# to every other pane it spawns, so a standalone ghostty pane next to a
964:# Agent inference for `pair resume <tag>`: look up what was paired with
965:# this tag last time. agent-<tag> covers live (or recently detached)
967:# config-<tag>-<agent>.json (issue #000016) covers the case where the
970:if [ -n "$forced_tag" ] && [ -z "$AGENT" ]; then
971:    if [ -f "$DATA_DIR/agent-${forced_tag}" ]; then
972:        AGENT=$(cat "$DATA_DIR/agent-${forced_tag}")
974:        for cfg in "$DATA_DIR"/config-"${forced_tag}"-*.json; do
987:# agent / forced_tag / forwarded args / continuation doc — dump them and exit
992:        "$AGENT" "$forced_tag" "$agent_extra" "${CONTINUE_DOC:-}"
1001:# under the same tag (handle_restart_marker, hoisted above). PAIR_TAG/PAIR_AGENT
1010:# tag-match; PAIR_KILL_CMD overrides the kill.
1030:            printf 'tag=%s\n' "$PAIR_TAG"
1032:            printf 'new_session=1\n'
1058:# Two callers reap, both via reap_nvim_for_tag:
1059:#   - cleanup_quit_marker (Alt+x path) — reaps the just-quit tag.
1061:#     tag whose pair-<tag> session is no longer alive. Covers external
1066:reap_nvim_for_tag() {
1067:    local tag="$1" kind pf pid
1072:    # var at $DATA_DIR/nvim-pid-<tag>-<kind>.
1074:        pf="$DATA_DIR/nvim-pid-${tag}-${kind}"
1084:    # prompted this change had to be found). Scoped to this tag's
1087:    pkill -9 -f "nvim --embed.*${DATA_DIR}/draft-${tag}\.md\$" >/dev/null 2>&1 || true
1088:    pkill -9 -f "nvim --embed.*${DATA_DIR}/scrollback-${tag}-" >/dev/null 2>&1 || true
1091:# Reap nvim --embed processes whose pair-<tag> session is no longer alive.
1095:    local live tag pf fname argv
1101:            tag=${fname#nvim-pid-}
1102:            tag=${tag%-draft}
1103:            tag=${tag%-scrollback}
1104:            printf '%s\n' "$tag"
1112:                *nvim*--embed*"$DATA_DIR/draft-"*)
1113:                    tag=${argv##*"$DATA_DIR/draft-"}
1114:                    tag=${tag%% *}
1115:                    tag=${tag%.md}
1116:                    printf '%s\n' "$tag"
1119:                    tag=${argv##*"$DATA_DIR/scrollback-"}
1120:                    tag=${tag%% *}
1121:                    tag=${tag%.ansi}
1122:                    # Filename shape: scrollback-<tag>-<agent>.ansi —
1123:                    # strip trailing -<agent> to recover <tag>.
1124:                    tag=${tag%-*}
1125:                    printf '%s\n' "$tag"
1129:    } | sort -u | while IFS= read -r tag; do
1130:        [ -z "$tag" ] && continue
1131:        if ! printf '%s\n' "$live" | grep -qx "$tag"; then
1132:            reap_nvim_for_tag "$tag"
1152:# rename runs only at pane creation, so on resume the title keeps the
1166:# command can record it into pane-<tag>-<agent>.json for pair-title's frame
1173:# prompt and `resume <tag>` override this.
1174:export PAIR_TAG="$default_tag"
1181:    for old in "$HOME/scratch"/pair-draft-*.md "$HOME/scratch"/pair-log-*.md; do
1197:# whether the user explicitly named an agent). Convention: tags named
1202:HISTORY_DAYS="${PAIR_HISTORY_DAYS:-14}"
1208:# `zellij list-sessions` exits 1 when there are no sessions, and with
1212:all_pair=$( { zj list-sessions --short 2>/dev/null || true; } \
1217:# a tag whose pair-<tag> session is gone — external `zellij kill-session`,
1225:    if zj list-sessions --no-formatting 2>/dev/null \
1234:    # list-sessions; the rest of the pipeline still produces a clean "0" for
1244:# ── Historical tags from this cwd (#000024) ──────────────────────────────────
1245:# Surface tags named `$HISTORY_BASE` or `$HISTORY_BASE-*` that were touched
1247:# live zellij session. Lets the operator resume an old session by name from
1249:# tags that don't follow the cwd-prefix rule are not surfaced; the operator
1253:# Scan strategy: walk draft-/log- sidecars (created/touched by nvim's
1254:# autosave and bin/pair's send-log), bucket per tag, take max mtime, filter
1259:    local f name tag mt
1260:    for f in "$DATA_DIR"/draft-*.md "$DATA_DIR"/log-*.md; do
1264:            draft-*.md) tag="${name#draft-}"; tag="${tag%.md}" ;;
1265:            log-*.md)   tag="${name#log-}";   tag="${tag%.md}" ;;
1268:        # Strict prefix: tag is exactly $base, or starts with "$base-".
1269:        case "$tag" in
1275:        printf '%s\t%s\n' "$tag" "$mt"
1292:# Greyscale gradient by age for the historical (no-live-session) rows: the
1293:# more recently a tag was touched, the brighter it reads. 256-color greys
1294:# (xterm 232=near-black .. 255=near-white); fzf --ansi honors these. Newest
1307:# Count the +N draft items parked in a tag's queue. nvim writes each queued
1308:# prompt as queue-<tag>/<6-digit-key>.md (see queue_dir / queue_keys_sorted in
1311:# queue is visible before you resume it.
1313:    local tag="$1" qdir="$DATA_DIR/queue-$tag" n=0 f
1325:# Build the live-tag exclusion set as a newline-delimited string (bash 3.2
1326:# friendly — no assoc arrays). One tag per line, lookup via `grep -qx`.
1327:live_tags_nl=$(printf '%s\n' "$all_pair" | sed 's|^pair-||' | sort -u)
1329:# Capture the historical scan once. It feeds two things: the next-free create
1333:historical_tags_nl=$(printf '%s\n' "$history_scan" | cut -f1 | sed '/^$/d' | sort -u)
1335:# Compute the next-free slot in the agent's family — pair-<base>, then
1337:# Skip any candidate that is live OR has recent history, so "+ new <tag>
1342:free_slot_tag=""
1350:    cand_tag="${cand#pair-}"
1352:       && ! printf '%s\n' "$historical_tags_nl" | grep -qx "$cand_tag"; then
1353:        free_slot_tag="$cand_tag"
1359:historical_rows=()      # plain text — drives the fallback display + matching
1360:historical_colored=()   # age-graded grey — fzf display only
1361:while IFS=$'\t' read -r tag mtime; do
1362:    [ -z "$tag" ] && continue
1363:    # Dedup against live sessions only — free_slot_tag is guaranteed not to be
1364:    # a historical tag (the slot scan above skips history), so an inactive
1365:    # pair-<tag> now shows up here as its own resumable row.
1366:    if printf '%s\n' "$live_tags_nl" | grep -qx "$tag"; then continue; fi
1368:    base_row="$(printf 'pair-%s  (%s, no live session)' "$tag" "$age")"
1372:    # fzf --ansi return still maps back to a historical_rows entry, and the
1373:    # "${picked%%  *}" → pair-<tag> parse below is unaffected (badge sits
1375:    qn=$(queue_count_for "$tag")
1381:    historical_rows+=("${base_row}${badge_plain}")
1383:    historical_colored+=("$(age_color "$diff_days")${base_row}"$'\033[0m'"${badge_colored}")
1390:    echo "pair: historical scan"
1394:    echo "  live tags      = $(printf '%s' "$live_tags_nl" | tr '\n' ' ')"
1395:    echo "  free_slot_tag  = $free_slot_tag"
1396:    if [ "${#historical_rows[@]}" -eq 0 ]; then
1397:        echo "  (no historical tags matched)"
1400:        printf '    %s\n' "${historical_rows[@]}"
1405:# Decide. Picker for any ≥1 detached or ≥1 historical row; create directly
1408:chosen_tag=""
1410:# 1 only when naming a brand-new session (the "+ new" pick, or a direct
1411:# create when there's nothing to surface). Picking an existing historical
1412:# tag, or `pair resume <tag>`, already names the session — and a historical
1413:# tag is known to have no live session (it was deduped against the live set)
1417:# `pair resume <tag>` short-circuits both the picker and the name prompt:
1418:# attach if pair-<tag> already exists in any state, otherwise create with
1419:# that tag. This is the "resume by name" path — pair off the hint printed
1421:if [ -n "$forced_tag" ]; then
1422:    if session_blocks_reuse "pair-${forced_tag}"; then
1423:        chosen_session="pair-${forced_tag}"
1424:        chosen_tag="$forced_tag"
1427:        chosen_tag="$forced_tag"
1430:elif [ ${#detached_list[@]} -eq 0 ] && [ ${#historical_rows[@]} -eq 0 ]; then
1431:    chosen_tag="$free_slot_tag"
1437:    # fzf display only. fzf --ansi strips its color codes from the returned
1441:        if [ ${#historical_rows[@]} -gt 0 ]; then printf '%s\n' "${historical_rows[@]}"; fi
1444:    if command -v fzf >/dev/null 2>&1; then
1445:        # Color-code: green = live session (picking it attaches); historical
1446:        # rows carry an age-graded grey (set per-row in historical_colored,
1452:            if [ ${#historical_colored[@]} -gt 0 ]; then printf '%s\n' "${historical_colored[@]}"; fi
1455:        picked=$(printf '%s' "$options_colored" | fzf --ansi --prompt='pair> ' --height=10 --reverse) || exit 0
1457:        echo "Detached / historical pair sessions:"
1466:        chosen_tag="$free_slot_tag"
1470:        # Historical rows are annotated as "pair-<tag>  (... ago, ...)" — the
1472:        # tags can't contain spaces), so trimming everything from the first
1473:        # double-space onward yields the bare "pair-<tag>" for both shapes.
1475:        chosen_tag="${chosen_session#pair-}"
1476:        # If the picked tag is currently live → attach; otherwise (historical
1477:        # row, or the unlikely "live tag listed as historical" race) → create
1478:        # by name, which goes through the same path as `pair resume <tag>`.
1479:        if printf '%s\n' "$live_tags_nl" | grep -qx "$chosen_tag"; then
1488:if [ -z "$chosen_tag" ]; then
1501:        # Capture the agent BEFORE the file is cleared below, so the resume
1509:        # session record on its next heartbeat), SIGKILL it. Pattern matches
1512:        # Reap this tag's nvim --embed children. See reap_nvim_for_tag and
1515:        reap_nvim_for_tag "$PAIR_TAG"
1518:        # non-recyclable name so the next `pair <same-tag>` O_TRUNC can't clobber
1545:                    # .raw/.events.jsonl rename + parked-<tag> marker (#55 ARCH-DRY).
1570:        # If the watcher captured a session config for this (tag, agent),
1571:        # show the user the one-liner that resumes it. config-<tag>-<agent>
1573:        # signal that the resume path will work next time.
1576:        # what the user saw in the UI tab; the `resume` subcommand accepts
1578:        # the resume command — pair infers it from the saved state.
1579:        local resume_cfg
1580:        resume_cfg="$(resolve_config_file "$PAIR_TAG" "$quit_agent")"
1581:        if [ -f "$resume_cfg" ]; then
1584:                "$resume_cfg" 2>/dev/null || true)
1585:            printf 'pair: saved session config for tag "%s" (%s).\n' \
1587:            printf '      resume with: pair resume %s\n' "$SESSION"
1589:            # wants to drive the agent directly (`claude --resume <id>`,
1590:            # `codex resume <id>`) without going back through pair.
1596:        # Kill the cmux activity-prefix poller for this tag (if running)
1597:        # so its next 60s tick doesn't race with the cwd-rename below.
1609:        # of the just-killed pair-<tag>. On Alt+n / Shift+Alt+N the
1616:        # remaining pair (or the next one) can claim.
1630:#   Alt+n         — pure pair reload. Same tag + agent + args + agent
1632:#                   resumes the prior conversation via --resume <id> (claude),
1633:#                   --conversation <id> (agy), or `resume <id>` (codex).
1634:#   Shift+Alt+N   — same tag + agent + args, fresh agent conversation:
1635:#                   we drop the saved per-(tag,agent) config so the new
1636:#                   run starts a brand-new session.
1638:# Both flavors pin the same tag via PAIR_FORCE_TAG. The agent name comes
1639:# from the marker (captured before cleanup_quit_marker wiped agent-<tag>);
1661:# Ensure the per-tag title poller (bin/pair-title.sh) is running for this tag.
1664:# per-tag singleton AND re-takes over a dead/recycled prior PID, so create,
1679:    export PAIR_TAG="$chosen_tag"
1681:    DRAFT="$DATA_DIR/draft-${PAIR_TAG}.md"
1689:    # Real agent for this tag may differ from the CLI default — prefer
1704:# to a custom-named session whose tag happens to match a non-command).
1711:# Surface any existing sessions in this tag's family (pair-<base> and
1715:# historical pick or `pair resume <tag>` already has its name.
1718:[ "$prompt_name" = 1 ] && family_existing=$( { zj list-sessions --no-formatting 2>/dev/null || true; } \
1728:# suggested next-free tag (the `pair-` prefix is implicit and added back
1738:# Skipped unless we're naming a brand-new session (prompt_name=1). A
1739:# historical pick already chose the name (and is known to have no live
1740:# session, so there's nothing to collide with), and `pair resume <tag>`
1741:# passed a validated forced_tag — both land in `create` with chosen_tag
1748:        # edit the prefix of the prefilled tag (only append). `bindkey
1757:            ' _ "$chosen_tag" </dev/tty 2>/dev/tty
1764:        printf 'Session name [%s]: ' "$chosen_tag" >/dev/tty
1770:    user_input="${user_input:-$chosen_tag}"
1771:    typed_tag="$(normalize_tag "$user_input" "pair: invalid name")" || exit 1
1772:    if session_blocks_reuse "pair-${typed_tag}"; then
1773:        echo "pair: session 'pair-${typed_tag}' already exists." >&2
1776:    chosen_tag="$typed_tag"
1782:# If we have a saved config for (tag, agent) from a prior launch, ask the
1783:# user whether to reuse the saved args (and resume the agent's session if
1784:# the agent's transcript is still on disk). This is the place: the tag has
1788:# fzf is already a hard dep (used for the session picker). Each option line
1791:# If the user (or Alt+n's restart loop) explicitly passed a resume token
1793:# Same per-agent surface as the explicit_resume detection further below;
1795:_argv_has_resume=0
1800:            if [ "$prev" = "--resume" ] || [ "$prev" = "--conversation" ]; then _argv_has_resume=1; break; fi
1801:            if [[ "$tok" =~ ^--conversation= ]]; then _argv_has_resume=1; break; fi
1807:        [ "$first" = "resume" ] && [ -n "$second" ] && _argv_has_resume=1
1811:config_file="$(resolve_config_file "$chosen_tag" "$AGENT")"
1812:if [ -f "$config_file" ] && [ $_argv_has_resume -eq 0 ]; then
1820:    # Defensive strip: pair-session-watch.sh and the inline explicit_resume
1821:    # capture both now scrub --resume <id> (and codex's `resume <id>`)
1824:    # still carry the resume token and we don't want the picker to
1825:    # display "args=[... --resume <id>] / resume=<id>" with the same id
1826:    # duplicated, nor have the "saved (fresh)" option silently resume.
1832:        && [ "${saved_args[0]}" = "resume" ]; then
1836:        if [ "${saved_args[$i]}" = "--resume" ] || [ "${saved_args[$i]}" = "--conversation" ]; then
1846:    # Stale-id check: only offer "resume" if the agent's transcript file
1882:    # truncation. NUL-separated items + `--read0` is fzf's multi-line-render
1883:    # mode (added in 0.46, broadly available in homebrew fzf). Each item
1884:    # spans three lines: label, indented args, indented resume/freshness.
1894:        options+=("$(printf '%d) use saved params + session\n     args=[%s]\n     resume=%s' \
1896:        actions+=("saved+resume")
1905:    # New params + resumed session is only meaningful when there *are* new
1909:        options+=("$(printf '%d) use new params + session\n     args=[%s]\n     resume=%s' \
1911:        actions+=("new+resume")
1924:    fzf_height=$(( ${#options[@]} * 3 + 4 ))
1929:        done | fzf --read0 \
1930:                   --header "saved config for tag '$chosen_tag' ($AGENT)" \
1931:                   --height "$fzf_height" \
1937:    # Map fzf's selection back to the action key. Comparing full multi-line
1948:        saved+resume|new+resume)
1949:            # Compose: (base args stripped of any resume tokens) + the
1950:            # canonical resume binding from saved_session. saved+resume's
1951:            # base is saved_args_clean (already stripped above). new+resume's
1952:            # base is the user-typed agent_extra, which may carry --resume
1954:            # with duplicate resume tokens in the composed argv.
1956:            if [ "$action" = "saved+resume" ]; then
1965:                    && [ "${_new_base[0]}" = "resume" ]; then
1969:                    if [ "${_new_base[$i]}" = "--resume" ] || [ "${_new_base[$i]}" = "--conversation" ]; then
1982:                    # Codex resume is `codex resume <id> [other args]` —
1984:                    agent_extra="resume $saved_session${stripped[*]:+ ${stripped[*]}}"
1987:                    # agy resume is `--conversation <id>`
1992:                    # `--resume <id>` style.
1993:                    agent_extra="${stripped[*]:+${stripped[*]} }--resume $saved_session"
1999:            # --resume baked into the on-disk config must NOT come along
2005:            # spawn below will write a fresh one once the new session
2014:export PAIR_TAG="$chosen_tag"
2016:DRAFT="$DATA_DIR/draft-${PAIR_TAG}.md"
2021:# same-tag session's draft is never clobbered (that path attaches instead).
2034:# Pre-capture the session id when --resume <id> is explicit on the
2036:# creates; an explicit --resume makes the agent APPEND to an existing
2038:# config. Pre-writing here means config-<tag>-<agent>.json is right from
2039:# the start, so a future `pair resume` picker shows the resumed session
2043:#   claude / agy   — flag style: `--resume <id>` / `--conversation <id>` anywhere in argv.
2044:#   codex           — subcommand: `resume <id>` at args[0..1].
2045:explicit_resume=""
2050:            if [ "$prev" = "--resume" ] || [ "$prev" = "--conversation" ]; then
2051:                explicit_resume="$tok"
2055:                explicit_resume="${BASH_REMATCH[1]}"
2063:        if [ "$first" = "resume" ] && [ -n "$second" ]; then
2064:            explicit_resume="$second"
2068:if [ -n "$explicit_resume" ]; then
2069:    # Strip the --resume <id> (or codex's `resume <id>`) we just detected
2071:    # canonical storage for the resume binding. Without this, every
2072:    # --resume launch writes a config whose `args` contains the resume
2073:    # token, the picker then re-appends it on the next launch, and the
2080:    if [ "$AGENT" = "codex" ] && [ $n -ge 2 ] && [ "${_eargs[0]}" = "resume" ]; then
2084:        if [ "${_eargs[$i]}" = "--resume" ] || [ "${_eargs[$i]}" = "--conversation" ]; then
2097:            --arg sid "$explicit_resume" \
2106:# watcher. Two tags in the same cwd previously raced for the next new
2112:#   - explicit_resume already pinned a session above;
2117:if [ "$AGENT" = "claude" ] && [ -z "$explicit_resume" ]; then
2145:            # is the canonical storage for the resume binding, and leaving
2147:            # picker (same logic as the explicit_resume strip above).
2184:# Append (not prepend) keeps the `resume` subcommand at args[0] when
2202:# its file set per-session (#63): a fresh session opens an empty log, a resume
2204:# resume pins it ($explicit_resume) — the Alt+n restart re-execs with --resume,
2205:# so it lands in the explicit_resume arm too. codex/agy fresh sessions discover
2207:# and the draft nvim watcher fall back to reading .session_id from the per-tag
2209:export PAIR_SESSION_ID="${explicit_resume:-${new_sid:-}}"
2231:# Set terminal title — harmless duplicate for new sessions (zellij sets it
2254:# message. Covers every create path — `continue`, `resume`, a manually-typed
2255:# long tag — not just the one that surfaced it (#54).
2264:        echo "pair: tag '$PAIR_TAG' makes zellij's session name too long for this" >&2
2265:        echo "      machine's socket path ($SESSION). Pick a shorter tag." >&2

exec
/bin/zsh -lc 'nl -ba cmd/internal/dispatcher/dispatcher.go' in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc 'nl -ba cmd/internal/launcher/tag.go' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package launcher
     2	
     3	import (
     4		"fmt"
     5		"path/filepath"
     6		"strings"
     7	)
     8	
     9	// NormalizeTag returns Pair's canonical bare tag form.
    10	func NormalizeTag(raw string) (string, error) {
    11		tag := strings.TrimPrefix(raw, "pair-")
    12		if tag == "" {
    13			return "", fmt.Errorf("empty tag")
    14		}
    15		for _, r := range tag {
    16			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
    17				continue
    18			}
    19			return "", fmt.Errorf("tag %q contains invalid character %q", raw, r)
    20		}
    21		return tag, nil
    22	}
    23	
    24	// DefaultTag derives Pair's create-flow default tag from a cwd path.
    25	func DefaultTag(cwd string) string {
    26		base := filepath.Base(cwd)
    27		if base == "." || base == string(filepath.Separator) {
    28			return "pair"
    29		}
    30		var b strings.Builder
    31		for _, r := range base {
    32			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
    33				b.WriteRune(r)
    34			} else {
    35				b.WriteByte('_')
    36			}
    37		}
    38		if b.Len() == 0 {
    39			return "pair"
    40		}
    41		return b.String()
    42	}

 succeeded in 0ms:
     1	package dispatcher
     2	
     3	import (
     4		"fmt"
     5		"os"
     6		"strings"
     7		"time"
     8	
     9		"github.com/xianxu/pair/cmd/internal/launcher"
    10	)
    11	
    12	const programName = "pair-go"
    13	
    14	// CommandFamily names a future Pair CLI surface without claiming it works yet.
    15	type CommandFamily struct {
    16		Name    string
    17		Summary string
    18		Status  string
    19	}
    20	
    21	// Result is the process-facing outcome of a pure dispatch decision.
    22	type Result struct {
    23		Stdout   string
    24		Stderr   string
    25		ExitCode int
    26	}
    27	
    28	// Families returns the planned command families for the Go dispatcher.
    29	func Families() []CommandFamily {
    30		return []CommandFamily{
    31			{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "planned"},
    32			{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
    33			{Name: "slug", Summary: "session orientation slug generation", Status: "planned"},
    34			{Name: "context", Summary: "agent pane context meter", Status: "planned"},
    35			{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "planned"},
    36			{Name: "changelog", Summary: "TTY transcript to distilled change log", Status: "planned"},
    37			{Name: "continuation", Summary: "continuation datatype writer", Status: "planned"},
    38			{Name: "scribe", Summary: "PTY logging wrapper", Status: "planned"},
    39		}
    40	}
    41	
    42	// Dispatch parses argv and returns the skeleton dispatch result.
    43	func Dispatch(args []string) Result {
    44		if len(args) == 0 {
    45			return Result{Stdout: Help(programName), ExitCode: 0}
    46		}
    47	
    48		switch args[0] {
    49		case "help", "--help", "-h":
    50			return Result{Stdout: Help(programName), ExitCode: 0}
    51		case "version", "--version":
    52			return Result{
    53				Stdout:   "pair-go dispatcher skeleton\npublic launcher: bin/pair\n",
    54				ExitCode: 0,
    55			}
    56		case "launch":
    57			return DispatchWithLauncherRuntime(args, osLauncherRuntime())
    58		}
    59	
    60		if family, ok := familyByName(args[0]); ok {
    61			return Result{
    62				Stderr:   fmt.Sprintf("%s: %s is planned but not implemented in this skeleton; run %s help\n", programName, family.Name, programName),
    63				ExitCode: 2,
    64			}
    65		}
    66	
    67		return Result{
    68			Stderr:   fmt.Sprintf("%s: unknown command %q; run %s help\n", programName, args[0], programName),
    69			ExitCode: 2,
    70		}
    71	}
    72	
    73	type LauncherRuntime struct {
    74		Env      launcher.Env
    75		Sessions launcher.SessionSource
    76		History  launcher.HistoricalScanner
    77	}
    78	
    79	type StaticSessions struct {
    80		Sessions []launcher.Session
    81		Err      error
    82	}
    83	
    84	func (s StaticSessions) Snapshot() ([]launcher.Session, error) {
    85		if s.Err != nil {
    86			return nil, s.Err
    87		}
    88		return s.Sessions, nil
    89	}
    90	
    91	type StaticHistory struct {
    92		Tags []launcher.HistoricalTag
    93		Err  error
    94	}
    95	
    96	func (h StaticHistory) Scan(_ string, _ time.Time) ([]launcher.HistoricalTag, error) {
    97		if h.Err != nil {
    98			return nil, h.Err
    99		}
   100		return h.Tags, nil
   101	}
   102	
   103	func DispatchWithLauncherRuntime(args []string, rt LauncherRuntime) Result {
   104		launchArgs := []string(nil)
   105		if len(args) > 1 {
   106			launchArgs = args[1:]
   107		}
   108		if len(launchArgs) == 0 || launchArgs[0] == "help" || launchArgs[0] == "--help" || launchArgs[0] == "-h" {
   109			return Result{Stdout: LaunchHelp(programName), ExitCode: 0}
   110		}
   111		outcome, err := launcher.Run(launchArgs, rt.Env, rt.Sessions, rt.History)
   112		if err != nil {
   113			return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
   114		}
   115		decision := outcome.Decision
   116		return Result{
   117			Stderr: fmt.Sprintf(
   118				"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
   119				decision.Action,
   120				decision.Tag,
   121				decision.SessionName,
   122			),
   123			ExitCode: 3,
   124		}
   125	}
   126	
   127	func LaunchHelp(program string) string {
   128		return fmt.Sprintf(`Usage: %s launch [agent] [-- agent-args...]
   129	       %s launch resume <tag>
   130	
   131	Guarded decision-phase prototype. Public sessions still start through bin/pair.
   132	This command parses launch inputs and computes the create/attach/picker decision,
   133	then stops before invoking zellij.
   134	`, program, program)
   135	}
   136	
   137	func LauncherEnv(home, xdgDataHome, cwd string) launcher.Env {
   138		return launcher.Env{
   139			Home:     home,
   140			XDGData:  xdgDataHome,
   141			Cwd:      cwd,
   142			Now:      time.Now(),
   143			HistoryD: 14,
   144		}
   145	}
   146	
   147	func osLauncherRuntime() LauncherRuntime {
   148		home := os.Getenv("HOME")
   149		xdg := os.Getenv("XDG_DATA_HOME")
   150		cwd, _ := os.Getwd()
   151		env := LauncherEnv(home, xdg, cwd)
   152		dataDir := launcher.ResolveDataDir(home, xdg)
   153		return LauncherRuntime{
   154			Env:      env,
   155			Sessions: launcher.ZellijSource{},
   156			History:  launcher.HistorySource{DataDir: dataDir},
   157		}
   158	}
   159	
   160	// Help renders the development-only dispatcher usage text.
   161	func Help(program string) string {
   162		var b strings.Builder
   163		fmt.Fprintf(&b, "Usage: %s <command> [args]\n\n", program)
   164		b.WriteString("Development dispatcher skeleton. Public sessions still start through bin/pair.\n\n")
   165		b.WriteString("Planned command families (not implemented in this skeleton):\n")
   166		for _, family := range Families() {
   167			fmt.Fprintf(&b, "  %-17s %s (%s; not implemented in this skeleton)\n", family.Name, family.Summary, family.Status)
   168		}
   169		b.WriteString("\nSupported skeleton commands:\n")
   170		b.WriteString("  help              show this help\n")
   171		b.WriteString("  version           show dispatcher skeleton metadata\n")
   172		return b.String()
   173	}
   174	
   175	func familyByName(name string) (CommandFamily, bool) {
   176		for _, family := range Families() {
   177			if family.Name == name {
   178				return family, true
   179			}
   180		}
   181		return CommandFamily{}, false
   182	}

exec
/bin/zsh -lc "find workshop/plans -maxdepth 1 -type f -name '*75*' -o -name '*launcher*' | sort" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
workshop/plans/000075-go-launcher-prototype-plan.md

codex
The shell launcher has a few non-obvious contracts around exited sessions and history. I’m reading those exact regions and the durable plan before deciding whether mismatches are intentional scope or review findings.
exec
/bin/zsh -lc "sed -n '560,690p' bin/pair" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
# name (clipboard-to-pane.sh, copy-on-select.sh) without sh-c env-var hacks
# in zellij/config.kdl. Avoid double-prepending across re-launches.
case ":$PATH:" in
    *":$PAIR_HOME/bin:"*) ;;
    *) export PATH="$PAIR_HOME/bin:$PATH" ;;
esac

# Parse argv. First positional is the agent name (default `claude`).
# Anything after `--` is forwarded to the agent on create. The `--`
# separator is required to disambiguate "agent name" from "flag for the
# agent" — without it, `pair --dangerously-skip-permissions` is ambiguous
# (is `--dangerously-skip-permissions` the agent or a flag?).
#
#   pair                                  → claude
#   pair claude                           → claude
#   pair claude -- --resume               → claude --resume
#   pair -- --resume                      → claude --resume
#   pair codex -- -p "hi"                 → codex -p hi   (word-split caveat)
AGENT=""
agent_extra=""
forced_tag=""

# `pair resume <tag>` is the documented restart path: skip both the picker
# and the name prompt, attach if pair-<tag> exists, otherwise create with
# that tag. We claim `resume` as a subcommand verb — it can't be a real
# agent binary (too generic a name), and we already special-case `list`
# and `help` similarly above.
if [ "${1:-}" = "resume" ]; then
    if [ -z "${2:-}" ]; then
        echo "pair: 'resume' requires a tag (e.g. pair resume bugfix)" >&2
        exit 1
    fi
    forced_tag="$(normalize_tag "$2" "pair: invalid tag")" || exit 1
    shift 2
fi

# `pair continue [slug] [agent]` — the human-understanding sibling of `resume`
# (ariadne#91). Bare: list continuation docs. With a slug: seed a fresh session
# to read the matching doc and execute its NEXT ACTION. It does NOT force the
# tag — like `pair <agent> -- <args>` it flows through the normal name prompt
# so the operator picks the tag (a long slug must never become an over-long
# zellij session name — see the launch-time guard) and any `-- <args>` reach
# the agent. The optional agent is the "port" (continue under a different
# stack); absent, it's read from the doc. We capture slug + agent HERE (the
# positional loop below rejects a bare agent arg) but leave `-- <args>` for it.
# `continue` never reads session_id — that's `resume`.
# Fresh-vs-attach (#52): the seeded session is "fresh" only for a NEW tag. If the
# tag picked at the prompt collides with a LIVE `pair-<tag>`, the normal launch
# path ATTACHES to it (same as `pair <agent>` / `resume`) rather than reseeding —
# intentional, so a continue never clobbers a running session; pick a fresh tag
# to force a clean seed.
CONTINUE_DOC=""
if [ "${1:-}" = "continue" ]; then
    _croot="$(git rev-parse --show-toplevel 2>/dev/null || true)"
    _cdir="${_croot:-$PWD}/workshop/continuation"
    if [ -z "${2:-}" ]; then
        if [ -d "$_cdir" ] && ls "$_cdir"/*.md >/dev/null 2>&1; then
            printf 'continuations in %s:\n' "$_cdir"
            for _cf in "$_cdir"/*.md; do
                _cs="$(basename "$_cf" .md)"; _cs="${_cs#*-}"
                _cn="$(awk '/^## NEXT ACTION/{f=1;next} f&&NF{print;exit}' "$_cf")"
                # cap the preview so a long NEXT ACTION line doesn't flood the row (#52)
                if [ "${#_cn}" -gt 80 ]; then _cn="${_cn:0:79}…"; fi
                _ci="$(awk -F': ' '/^issues:/{print $2; exit}' "$_cf")"
                printf '  %-22s %-18s %s\n' "$_cs" "${_ci:-[]}" "$_cn"
            done
        else
            printf 'pair: no continuations in %s\n' "$_cdir" >&2
        fi
        exit 0
    fi
    _cslug="$(normalize_tag "$2" "pair: invalid slug")" || exit 1
    CONTINUE_DOC="$(ls -1 "$_cdir"/*-"$_cslug".md 2>/dev/null | sort | tail -1)"
    if [ -z "$CONTINUE_DOC" ]; then
        echo "pair: no continuation matching '$_cslug' in $_cdir" >&2
        exit 1
    fi
    # No forced_tag (see header). Consume `continue` + slug; an optional next
    # arg is the agent port UNLESS it's `--` (then no port, only forwarded
    # args). Leave everything from `--` onward for the positional loop to
    # capture as agent_extra.
    shift 2
    if [ -n "${1:-}" ] && [ "${1:-}" != "--" ]; then
        AGENT="$1"   # explicit port: continue under a different agent
        shift
    else
        AGENT="$(awk -F': ' '/^agent:/{print $2; exit}' "$CONTINUE_DOC")"
    fi
fi

seen_dd=false
for arg in "$@"; do
    if $seen_dd; then
        if [ -z "$agent_extra" ]; then
            agent_extra="$arg"
        else
            agent_extra="$agent_extra $arg"
        fi
    elif [ "$arg" = "--" ]; then
        seen_dd=true
    elif [ -z "$AGENT" ] && [ -z "$forced_tag" ]; then
        AGENT="$arg"
    else
        echo "pair: unexpected positional arg '$arg' (use '--' to forward args to the agent)" >&2
        echo "      e.g. pair claude -- --resume" >&2
        exit 1
    fi
done

# PAIR_FORCE_TAG is set by the Alt+n / Shift+Alt+N restart flow
# (handle_restart_marker below): it pins the new run to the same tag as
# the killed session, skipping both the picker and the name prompt.
# Treated as forced_tag from here on; unset so a child process doesn't
# inherit it.
if [ -z "$forced_tag" ] && [ -n "${PAIR_FORCE_TAG:-}" ]; then
    forced_tag="$PAIR_FORCE_TAG"
fi
unset PAIR_FORCE_TAG

# --- #55 compaction support: hoisted DATA_DIR + early helpers ----------------
# DATA_DIR is hoisted above the `command -v zellij` gate, the in-session
# compaction branch, and the in_zellij_pane guard so park_scrollback /
# handle_restart_marker (defined here) and the branch can all use it.
# Honors $XDG_DATA_HOME with the spec-defined fallback. (Original site below
# the guard is removed.)
DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
mkdir -p "$DATA_DIR"
export PAIR_DATA_DIR="$DATA_DIR"

# Park a session's scrollback for later distillation into a continuation
# (ariadne#91). Extracted from cleanup_quit_marker so BOTH the Alt+x quit path

exec
/bin/zsh -lc "sed -n '800,850p' bin/pair" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
    if [ "$r_agent" = "codex" ] && [ -n "$resume_extra" ]; then
        combined="${resume_extra}${r_args:+ $r_args}"
    else
        combined="${r_args}${r_args:+${resume_extra:+ }}${resume_extra}"
    fi
    if [ -n "$combined" ]; then
        # shellcheck disable=SC2086
        _reexec "$0" "$r_agent" -- $combined
    else
        _reexec "$0" "$r_agent"
    fi
}

# #67: decide whether a live zellij session named <session> blocks reuse of its
# workspace tag. Alt+x can leave a resurrect record like
# `pair-2 (EXITED - attach to resurrect)`; that row still shows in
# `list-sessions`, so the old open-coded `grep -qx` collision checks treated a
# fully-quit tag as occupied. Treat an EXITED row as stale full-quit residue:
# delete the zellij record and report the tag reusable (non-zero). A
# running/detached row still blocks (zero); an absent session never blocks.
# Shared by forced resume, the free-slot scan, the name-prompt collision, and
# the cmux-owner liveness check (ARCH-DRY). `pair rename` keeps its own
# offline-only resurrectable-session contract and is intentionally not routed
# through here.
session_blocks_reuse() {
    local session="$1" row
    row=$( { zj list-sessions --no-formatting 2>/dev/null || true; } \
           | awk -v s="$session" '$1==s {print; exit}' )
    [ -z "$row" ] && return 1            # no such session — reuse is free
    case "$row" in
        *EXITED*)
            zj delete-session "$session" --force >/dev/null 2>&1 || true
            return 1 ;;                  # stale residue cleaned — reuse is free
    esac
    return 0                             # running/detached — still occupied
}

# #67: resolve the canonical saved-config path for a known (tag, agent), echoing
# it on stdout. Current launches write `config-<tag>-<agent>.json`, but older
# Codex sessions on disk use the doubled shape `config-<tag>-codex-codex.json`.
# When the canonical file is absent and a legacy Codex file exists whose JSON
# declares `"agent": "codex"`, migrate it to the canonical name so the resume
# picker can offer native resume. This is a narrow, agent-checked compatibility
# path — NOT a glob resolver — so unrelated stale files can never silently win
# (ARCH-DRY, ARCH-PURE). Used only where both tag and agent are already known;
# the agent-inference glob loop is deliberately left alone (it discovers the
# agent and already sees the legacy filename).
resolve_config_file() {
    local tag="$1" agent="$2"
    local canonical="$DATA_DIR/config-${tag}-${agent}.json"
    if [ ! -f "$canonical" ] && [ "$agent" = "codex" ]; then

exec
/bin/zsh -lc "sed -n '1488,1780p' bin/pair" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "sed -n '1188,1490p' bin/pair" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
if [ -z "$chosen_tag" ]; then
    echo "pair: couldn't determine a session to use; bailing." >&2
    exit 1
fi

# Helper: after zellij returns, check for the Alt+x marker. If present, the
# user asked for a full quit — delete the session entry from the resurrect
# list. (Ctrl+q is unbound in pair, so Alt+x is the only quit path; Alt+d
# is detach, which keeps the session alive and leaves no marker.)
cleanup_quit_marker() {
    local marker="$HOME/.cache/pair/quit-$SESSION"
    if [ -f "$marker" ]; then
        rm -f "$marker"
        # Capture the agent BEFORE the file is cleared below, so the resume
        # hint can name the right binary.
        local quit_agent _sb_base _parked _ans _pbase _ptimeout   # #52: keep park-nudge vars function-scoped
        quit_agent=$(cat "$DATA_DIR/agent-${PAIR_TAG}" 2>/dev/null || echo "$AGENT")
        zj delete-session "$SESSION" --force >/dev/null 2>&1 || true
        # Defensive belt-and-suspenders: if a zellij --server process for this
        # session is still alive after delete-session (rare but seen — e.g.
        # the server lingered in a no-clients state and re-registered the
        # session record on its next heartbeat), SIGKILL it. Pattern matches
        # the socket path component, scoped to this session only.
        pkill -9 -f "zellij --server .*/${SESSION}\$" >/dev/null 2>&1 || true
        # Reap this tag's nvim --embed children. See reap_nvim_for_tag and
        # the orphan-reaping block near $DATA_DIR setup for the full story
        # on why these survive pane death and need explicit cleanup.
        reap_nvim_for_tag "$PAIR_TAG"
        # Park-nudge: the scrollback is the only on-disk record of this session,
        # and Alt+x is about to discard it. Offer to PRESERVE it (rename to a
        # non-recyclable name so the next `pair <same-tag>` O_TRUNC can't clobber
        # it) so a live session can later distill it into a continuation
        # (ariadne#91). No live agent exists at quit, so we only preserve — never
        # distill here. `[ -t 0 ]` skips the prompt on a detached/automated quit.
        _sb_base="$DATA_DIR/scrollback-${PAIR_TAG}-${quit_agent}"
        _parked=0
        # Skip the park nudge when a restart is pending: a restart isn't a
        # quit (the relaunch keeps the work), and for #55 compaction the
        # branch has ALREADY copy-parked a recovery net + written a
        # continuation — re-asking "park as a continuation?" here is noise.
        # Also de-noises the inherited Alt+n / Shift+Alt+N restart paths.
        if [ -s "${_sb_base}.raw" ] && [ -t 0 ] \
            && [ ! -f "$HOME/.cache/pair/restart-$SESSION" ]; then
            # Bound the nudge with a timeout (default 5s, PAIR_PARK_PROMPT_TIMEOUT
            # seam) so an unattended quit auto-picks the default (N) instead of
            # blocking forever. read -t exits non-zero on timeout/EOF, so only the
            # `then` branch (read SUCCEEDED) consults the answer — any partial
            # input bash 4+ leaves in $_ans on timeout is never read as a "yes".
            # Guard a non-integer seam value (bash 3.2 `read -t` needs an integer).
            # A 0 value is a valid "don't wait": `read -t 0` returns at once, so
            # the prompt auto-declines (N) immediately rather than blocking.
            _ptimeout="${PAIR_PARK_PROMPT_TIMEOUT:-5}"
            case "$_ptimeout" in ''|*[!0-9]*) _ptimeout=5 ;; esac
            printf 'pair: preserve "%s" scrollback to distill into a continuation later? [y/N] (%ss → N): ' "$SESSION" "$_ptimeout" >/dev/tty
            if read -t "$_ptimeout" -r _ans </dev/tty; then
                if [ "$_ans" = "y" ] || [ "$_ans" = "Y" ]; then
                    # park_scrollback (move mode — session is dying) handles the
                    # .raw/.events.jsonl rename + parked-<tag> marker (#55 ARCH-DRY).
                    # The parked file is RAW VT bytes in the XDG data dir, NOT in the
                    # repo — the durable continuation doc is written later when a
                    # live agent renders + distills it.
                    if _pbase="$(park_scrollback "$PAIR_TAG" "$quit_agent")"; then
                        _parked=1
                        printf 'pair: scrollback preserved at\n        %s.raw\n      open a session and "park %s" to distill it into a continuation.\n' "$_pbase" "$SESSION" >/dev/tty
                    fi
                fi
            else
                # Timed out / EOF: no Enter was consumed, so close the
                # un-terminated prompt line before the cleanup output below.
                printf '\n' >/dev/tty
            fi
        fi
        rm -f "$DATA_DIR/outer-tty-${PAIR_TAG}" "$DATA_DIR/agent-${PAIR_TAG}" \
              "$DATA_DIR/agent-output-${PAIR_TAG}" \
              "${_sb_base}.ansi" \
              "$DATA_DIR/pair-wrap-pid-${PAIR_TAG}" \
              "$DATA_DIR/adapt-${PAIR_TAG}.jsonl" \
              "$DATA_DIR/image-capture-${PAIR_TAG}" \
              "$DATA_DIR/image-capture-${PAIR_TAG}.done"
        # Remove the raw capture only if it wasn't parked (preserved above).
        [ "$_parked" = 1 ] || rm -f "${_sb_base}.raw" "${_sb_base}.events.jsonl"

        # If the watcher captured a session config for this (tag, agent),
        # show the user the one-liner that resumes it. config-<tag>-<agent>
        # is what the create-flow prompt reads — its presence here is the
        # signal that the resume path will work next time.
        #
        # Display SESSION (e.g. pair-2) rather than PAIR_TAG ("2") to match
        # what the user saw in the UI tab; the `resume` subcommand accepts
        # both forms (it strips a leading `pair-`). Agent is omitted from
        # the resume command — pair infers it from the saved state.
        local resume_cfg
        resume_cfg="$(resolve_config_file "$PAIR_TAG" "$quit_agent")"
        if [ -f "$resume_cfg" ]; then
            local saved_sid=""
            saved_sid=$(jq -r '.session_id // empty' \
                "$resume_cfg" 2>/dev/null || true)
            printf 'pair: saved session config for tag "%s" (%s).\n' \
                "$SESSION" "$quit_agent"
            printf '      resume with: pair resume %s\n' "$SESSION"
            # Surface the agent session id too: useful when the user
            # wants to drive the agent directly (`claude --resume <id>`,
            # `codex resume <id>`) without going back through pair.
            if [ -n "$saved_sid" ]; then
                printf '      session id:  %s\n' "$saved_sid"
            fi
        fi

        # Kill the cmux activity-prefix poller for this tag (if running)
        # so its next 60s tick doesn't race with the cwd-rename below.
        local poller_pid_file="$DATA_DIR/title-pid-${PAIR_TAG}"
        if [ -f "$poller_pid_file" ]; then
            local poller_pid
            poller_pid=$(cat "$poller_pid_file" 2>/dev/null)
            [ -n "$poller_pid" ] && kill "$poller_pid" 2>/dev/null || true
            rm -f "$poller_pid_file"
        fi

        # Reset the cmux workspace title to the shell's cwd basename —
        # this pair session is dead and the user is back at a plain
        # shell prompt; the workspace label should reflect that instead
        # of the just-killed pair-<tag>. On Alt+n / Shift+Alt+N the
        # follow-up re-exec immediately re-renames to the new SESSION,
        # so the cwd flash here is invisible.
        #
        # Gated on ownership: when a 2nd pair shares the workspace,
        # the 1st pair owns the title and a 2nd-pair exit must not
        # clobber it. After resetting we release ownership so any
        # remaining pair (or the next one) can claim.
        if pair_owns_cmux_workspace; then
            local reset_title
            reset_title=$(basename "$PWD" 2>/dev/null)
            [ -n "$reset_title" ] || reset_title="shell"
            cmux_rename_workspace "$reset_title"
            rm -f "$DATA_DIR/cmux-owner-${CMUX_WORKSPACE_ID}"
        fi
    fi
}

# After cleanup_quit_marker, check for the restart marker written by
# pair-restart.sh. Two flavors:
#
#   Alt+n         — pure pair reload. Same tag + agent + args + agent
#                   session. The saved config is kept and the new run
#                   resumes the prior conversation via --resume <id> (claude),
#                   --conversation <id> (agy), or `resume <id>` (codex).
#   Shift+Alt+N   — same tag + agent + args, fresh agent conversation:
#                   we drop the saved per-(tag,agent) config so the new
#                   run starts a brand-new session.
#
# Both flavors pin the same tag via PAIR_FORCE_TAG. The agent name comes
# from the marker (captured before cleanup_quit_marker wiped agent-<tag>);
# agent args + session id come from the saved config.
# handle_restart_marker() is defined earlier (hoisted for #55 so the
# PAIR_TEST_CALL dispatcher can drive it before the picker). It is still
# *called* below, after the zellij session exits.

# Capture pair's controlling TTY before zellij takes over — this IS the path
# of the outer terminal/wrapper's PTY. Writing OSC sequences to it bypasses
# zellij's escape filtering, letting outer wrappers (e.g. cmux, which watches
# OSC 9/777 to surface attention indicators) see notifications emitted from
# inside a pane. Refreshed on every attach because the outer PTY changes
# across detach/reattach; pane-shell env doesn't (env is frozen at zellij
# session-creation time). #000011
record_outer_tty() {
    local outer
    outer="$(tty 2>/dev/null || true)"
    case "$outer" in
        /dev/*) printf '%s\n' "$outer" > "$DATA_DIR/outer-tty-${PAIR_TAG}" ;;
        *)      rm -f "$DATA_DIR/outer-tty-${PAIR_TAG}" ;;
    esac
}

# Ensure the per-tag title poller (bin/pair-title.sh) is running for this tag.
# It owns the zellij frame meter (#71, always-on) AND the cmux workspace title
# (when in cmux). Safe to call on every entry: the poller self-guards as a
# per-tag singleton AND re-takes over a dead/recycled prior PID, so create,
# attach, and restart (which re-execs through the create path) all converge on
# exactly one live poller — reviving it if a host sleep/reboot/SIGKILL killed
# the previous one without it ever respawning. Best-effort cosmetics; never
# load-bearing.
ensure_title_poller() {
    local title_agent="$1"
    "$PAIR_HOME/bin/pair-title.sh" "$PAIR_TAG" "$title_agent" \
        </dev/null >/dev/null 2>&1 &
    # disown so cleanup_quit_marker's `kill <pid>` later doesn't spawn a
    # "[1] Terminated:" line into the user's shell on session exit.
    disown $! 2>/dev/null || true
}

if [ "$action" = "attach" ]; then
    export PAIR_TAG="$chosen_tag"
    SESSION="pair-${PAIR_TAG}"
    DRAFT="$DATA_DIR/draft-${PAIR_TAG}.md"
    [ -f "$DRAFT" ] || : > "$DRAFT"
    # Set terminal title — zellij sets this on new-session creation but not
    # on attach, so without this the title stays as whatever the shell's
    # preexec hook set (typically "bin/pair").
    printf '\033]0;%s\007' "$SESSION"
    record_outer_tty
    cmux_rename_workspace "$SESSION"
    # Real agent for this tag may differ from the CLI default — prefer
    # the on-disk record so the activity poller looks at the right
    # session file. Falls back to $AGENT if the record's missing.
    title_agent=$(cat "$DATA_DIR/agent-$PAIR_TAG" 2>/dev/null | tr -d '\r\n[:space:]')
    [ -n "$title_agent" ] || title_agent="$AGENT"
    ensure_title_poller "$title_agent"
    zellij --config-dir "$PAIR_HOME/zellij" attach "$SESSION"
    rc=$?
    cleanup_quit_marker
    handle_restart_marker
    exit $rc
fi

# Create branch. Validate the agent here (deferred past the picker so attach
# paths work even when AGENT isn't a real binary on PATH — e.g. attaching
# to a custom-named session whose tag happens to match a non-command).
if ! command -v "$AGENT" >/dev/null 2>&1; then
    echo "pair: agent '$AGENT' not found on PATH." >&2
    echo "      install it first, then re-run." >&2
    exit 1
fi

# Surface any existing sessions in this tag's family (pair-<base> and
# pair-<base>-*) before prompting, so the user can see what's already
# there — and why the auto-suggested prefill may include a numeric
# suffix. Only when we're about to prompt for a brand-new name; a
# historical pick or `pair resume <tag>` already has its name.
family_prefix="pair-${BASE_TAG}"
family_existing=""
[ "$prompt_name" = 1 ] && family_existing=$( { zj list-sessions --no-formatting 2>/dev/null || true; } \
    | awk -v p="$family_prefix" '$1 == p || index($1, p "-") == 1' )
if [ -n "$family_existing" ]; then
    {
        printf 'Existing %s* sessions:\n' "$family_prefix"
        printf '%s\n' "$family_existing" | sed 's/^/  /'
    } >/dev/tty
fi

# Prompt for a session name. The input line is pre-filled with the auto-
# suggested next-free tag (the `pair-` prefix is implicit and added back
# below) so the user can edit/extend it inline — e.g. with the default
# pre-filled as "pair", typing "1" yields "pair1". Empty input falls back
# to the suggested default.
#
# bash 3.2 (system bash on macOS) lacks `read -i`, so we delegate the
# editable prompt to zsh's `vared`, which is universally available on
# macOS and Linux. The vared editor UI runs on /dev/tty; the captured
# stdout is just the final value.
#
# Skipped unless we're naming a brand-new session (prompt_name=1). A
# historical pick already chose the name (and is known to have no live
# session, so there's nothing to collide with), and `pair resume <tag>`
# passed a validated forced_tag — both land in `create` with chosen_tag
# already set, so re-prompting would be redundant friction.
if [ "$prompt_name" = 1 ]; then
    if command -v zsh >/dev/null 2>&1; then
        # `zsh -c` boots without sourcing .zshrc, so the line editor's
        # keymap defaults to a raw state where Ctrl-A through Ctrl-F are
        # all `self-insert` — meaning the user couldn't navigate left to
        # edit the prefix of the prefilled tag (only append). `bindkey
        # -e` selects the emacs keymap explicitly, giving the expected
        # arrows / backspace / Ctrl-A / Ctrl-E bindings inside vared.
        if user_input=$(
            zsh -c '
                bindkey -e
                v=$1
                vared -p "Session name: " v
                print -r -- "$v"
            ' _ "$chosen_tag" </dev/tty 2>/dev/tty
        ); then :; else
            echo >/dev/tty
            exit 0
        fi
    else
        # No zsh — fall back to a non-editable [default]-style prompt.
        printf 'Session name [%s]: ' "$chosen_tag" >/dev/tty
        if ! read -r user_input </dev/tty; then
            echo >/dev/tty
            exit 0
        fi
    fi
    user_input="${user_input:-$chosen_tag}"
    typed_tag="$(normalize_tag "$user_input" "pair: invalid name")" || exit 1
    if session_blocks_reuse "pair-${typed_tag}"; then
        echo "pair: session 'pair-${typed_tag}' already exists." >&2
        exit 1
    fi
    chosen_tag="$typed_tag"
fi

# ---------------------------------------------------------------------------
# Tag-restart prompt (issue #000016)

 succeeded in 0ms:
    done
    if [ "$moved" -gt 0 ]; then
        echo "pair: migrated $moved file(s) from ~/scratch/ to $DATA_DIR" >&2
    fi
fi

BASE_TAG="$PAIR_TAG"

# Historical scan always anchors on the cwd basename (independent of
# whether the user explicitly named an agent). Convention: tags named
# `<cwd-base>` or `<cwd-base>-<subproject>` belong to this dir; #000024.
HISTORY_BASE="$(basename "$PWD")"
HISTORY_BASE="${HISTORY_BASE//[^A-Za-z0-9_-]/_}"
[ -z "$HISTORY_BASE" ] && HISTORY_BASE="pair"
HISTORY_DAYS="${PAIR_HISTORY_DAYS:-14}"

# Find ALL detached pair-* sessions, regardless of agent. The agent argument
# (and BASE_TAG) is only used for the create path's defaults; the picker
# itself shows everything you might want to reattach to, including custom-
# named sessions like pair-blogging that don't follow any agent prefix.
# `zellij list-sessions` exits 1 when there are no sessions, and with
# `set -euo pipefail` that would silently kill the whole script. Wrap the
# zellij call in `|| true` and use awk (returns 0 even on no matches)
# instead of grep (returns 1 on no matches).
all_pair=$( { zj list-sessions --short 2>/dev/null || true; } \
            | awk '/^pair-/' \
            | sort )

# With the live-session set in hand, reap any nvim --embed left behind by
# a tag whose pair-<tag> session is gone — external `zellij kill-session`,
# host reboot, or pair upgrade leave no Alt+x marker, so cleanup_quit_marker
# never runs and these embeds accumulate. Definition near $DATA_DIR setup.
sweep_orphan_nvim

detached_list=()
while IFS= read -r sess; do
    [ -z "$sess" ] && continue
    if zj list-sessions --no-formatting 2>/dev/null \
        | grep -E "^${sess}[[:space:]].*EXITED" >/dev/null 2>&1; then
        continue   # exited, skip
    fi
    # `zj --session NAME action list-clients` exits 1 with "There is no active
    # session!" when invoked from a process tree inside that same session's
    # panes — e.g. a script in an agent pane that shells out to `pair`. With
    # set -o pipefail, that propagates and errexit silently kills bin/pair at
    # the assignment. Swallow zellij's exit the same way line 804 does for
    # list-sessions; the rest of the pipeline still produces a clean "0" for
    # the empty-stdout case, which is the right semantic (treat unable-to-
    # query as detached).
    clients=$( { zj --session "$sess" action list-clients 2>/dev/null || true; } \
              | tail -n +2 | wc -l | tr -d ' ')
    if [ "${clients:-0}" -eq 0 ]; then
        detached_list+=("$sess")
    fi
done <<< "$all_pair"

# ── Historical tags from this cwd (#000024) ──────────────────────────────────
# Surface tags named `$HISTORY_BASE` or `$HISTORY_BASE-*` that were touched
# (draft / log mtime) within the last $HISTORY_DAYS days but no longer have a
# live zellij session. Lets the operator resume an old session by name from
# the picker without remembering the exact slot number. Convention-only —
# tags that don't follow the cwd-prefix rule are not surfaced; the operator
# is expected to name sessions `<cwd-base>-<subproject>` so they show up in
# the right context (atlas note).
#
# Scan strategy: walk draft-/log- sidecars (created/touched by nvim's
# autosave and bin/pair's send-log), bucket per tag, take max mtime, filter
# by cutoff, then dedup against the live set.

scan_history() {
    local base="$1" cutoff="$2"
    local f name tag mt
    for f in "$DATA_DIR"/draft-*.md "$DATA_DIR"/log-*.md; do
        [ -e "$f" ] || continue
        name="${f##*/}"
        case "$name" in
            draft-*.md) tag="${name#draft-}"; tag="${tag%.md}" ;;
            log-*.md)   tag="${name#log-}";   tag="${tag%.md}" ;;
            *) continue ;;
        esac
        # Strict prefix: tag is exactly $base, or starts with "$base-".
        case "$tag" in
            "$base"|"$base"-*) ;;
            *) continue ;;
        esac
        mt=$(stat -f %m "$f" 2>/dev/null || echo 0)
        [ "$mt" -lt "$cutoff" ] && continue
        printf '%s\t%s\n' "$tag" "$mt"
    done | awk -F'\t' '
        { if ($2 > m[$1]) m[$1] = $2 }
        END { for (t in m) printf "%s\t%s\n", t, m[t] }
    ' | sort -t$'\t' -k2 -nr
}

format_age() {
    local now="$1" then="$2"
    local diff_days=$(( (now - then) / 86400 ))
    case "$diff_days" in
        0) echo "today" ;;
        1) echo "yesterday" ;;
        *) echo "${diff_days}d ago" ;;
    esac
}

# Greyscale gradient by age for the historical (no-live-session) rows: the
# more recently a tag was touched, the brighter it reads. 256-color greys
# (xterm 232=near-black .. 255=near-white); fzf --ansi honors these. Newest
# stays comfortably readable on the dark bg; oldest fades toward the
# background without disappearing.
age_color() {
    local d="$1"
    if   [ "$d" -le 0 ]; then printf '\033[38;5;250m'   # today
    elif [ "$d" -le 1 ]; then printf '\033[38;5;245m'   # yesterday
    elif [ "$d" -le 3 ]; then printf '\033[38;5;242m'   # 2-3d
    elif [ "$d" -le 6 ]; then printf '\033[38;5;240m'   # 4-6d
    else                      printf '\033[38;5;238m'   # a week+
    fi
}

# Count the +N draft items parked in a tag's queue. nvim writes each queued
# prompt as queue-<tag>/<6-digit-key>.md (see queue_dir / queue_keys_sorted in
# nvim/init.lua), so the queued count is just how many <digits>.md files live
# there. Surfaced as a badge on inactive picker rows so a session's forgotten
# queue is visible before you resume it.
queue_count_for() {
    local tag="$1" qdir="$DATA_DIR/queue-$tag" n=0 f
    [ -d "$qdir" ] || { echo 0; return; }
    for f in "$qdir"/[0-9]*.md; do
        [ -e "$f" ] || continue   # no match → glob stays literal; skip
        n=$((n + 1))
    done
    echo "$n"
}

now_epoch=$(date +%s)
history_cutoff=$(( now_epoch - HISTORY_DAYS * 86400 ))

# Build the live-tag exclusion set as a newline-delimited string (bash 3.2
# friendly — no assoc arrays). One tag per line, lookup via `grep -qx`.
live_tags_nl=$(printf '%s\n' "$all_pair" | sed 's|^pair-||' | sort -u)

# Capture the historical scan once. It feeds two things: the next-free create
# slot (below) and the picker's "no live session" rows. Tags here have a recent
# draft/log sidecar but may or may not still have a live zellij session.
history_scan="$(scan_history "$HISTORY_BASE" "$history_cutoff")"
historical_tags_nl=$(printf '%s\n' "$history_scan" | cut -f1 | sed '/^$/d' | sort -u)

# Compute the next-free slot in the agent's family — pair-<base>, then
# pair-<base>-2, ... — used as the "+ new" prompt's default name on create.
# Skip any candidate that is live OR has recent history, so "+ new <tag>
# session" always proposes a genuinely fresh name; an inactive pair-<base>
# then surfaces as its own resumable "no live session" row below instead of
# being folded into "+ new" (#000024 follow-up). Checks against ALL live
# sessions (not just the family) to avoid clobbering a shared slot name.
free_slot_tag=""
i=1
while [ "$i" -le 100 ]; do
    if [ "$i" -eq 1 ]; then
        cand="pair-${BASE_TAG}"
    else
        cand="pair-${BASE_TAG}-${i}"
    fi
    cand_tag="${cand#pair-}"
    if ! session_blocks_reuse "$cand" \
       && ! printf '%s\n' "$historical_tags_nl" | grep -qx "$cand_tag"; then
        free_slot_tag="$cand_tag"
        break
    fi
    i=$((i + 1))
done

historical_rows=()      # plain text — drives the fallback display + matching
historical_colored=()   # age-graded grey — fzf display only
while IFS=$'\t' read -r tag mtime; do
    [ -z "$tag" ] && continue
    # Dedup against live sessions only — free_slot_tag is guaranteed not to be
    # a historical tag (the slot scan above skips history), so an inactive
    # pair-<tag> now shows up here as its own resumable row.
    if printf '%s\n' "$live_tags_nl" | grep -qx "$tag"; then continue; fi
    age=$(format_age "$now_epoch" "$mtime")
    base_row="$(printf 'pair-%s  (%s, no live session)' "$tag" "$age")"
    # Queue badge (only when non-empty): kept as a separate segment so the
    # colored row tints just the badge amber (xterm 214) while age_color
    # greys the base. The plain badge has the identical visible text, so the
    # fzf --ansi return still maps back to a historical_rows entry, and the
    # "${picked%%  *}" → pair-<tag> parse below is unaffected (badge sits
    # after the parenthetical's leading double-space).
    qn=$(queue_count_for "$tag")
    badge_plain=""; badge_colored=""
    if [ "$qn" -gt 0 ]; then
        badge_plain="$(printf '   [\xe2\x8f\x8e %d queued]' "$qn")"
        badge_colored="$(printf '   \033[38;5;214m[\xe2\x8f\x8e %d queued]\033[0m' "$qn")"
    fi
    historical_rows+=("${base_row}${badge_plain}")
    diff_days=$(( (now_epoch - mtime) / 86400 ))
    historical_colored+=("$(age_color "$diff_days")${base_row}"$'\033[0m'"${badge_colored}")
done <<< "$history_scan"

# Debug probe — `PAIR_DEBUG_HISTORY=1 pair` prints what the scan would have
# surfaced and exits without launching zellij. Use this to verify the cwd-
# prefix convention is matching what you expect on your data dir.
if [ "${PAIR_DEBUG_HISTORY:-0}" = "1" ]; then
    echo "pair: historical scan"
    echo "  data_dir       = $DATA_DIR"
    echo "  base (cwd)     = $HISTORY_BASE"
    echo "  window         = ${HISTORY_DAYS}d (cutoff epoch ${history_cutoff})"
    echo "  live tags      = $(printf '%s' "$live_tags_nl" | tr '\n' ' ')"
    echo "  free_slot_tag  = $free_slot_tag"
    if [ "${#historical_rows[@]}" -eq 0 ]; then
        echo "  (no historical tags matched)"
    else
        echo "  matched rows:"
        printf '    %s\n' "${historical_rows[@]}"
    fi
    exit 0
fi

# Decide. Picker for any ≥1 detached or ≥1 historical row; create directly
# only when nothing to surface.
chosen_session=""
chosen_tag=""
action=""        # "attach" | "create"
# 1 only when naming a brand-new session (the "+ new" pick, or a direct
# create when there's nothing to surface). Picking an existing historical
# tag, or `pair resume <tag>`, already names the session — and a historical
# tag is known to have no live session (it was deduped against the live set)
# — so the name prompt + collision warning below are skipped for those.
prompt_name=0

# `pair resume <tag>` short-circuits both the picker and the name prompt:
# attach if pair-<tag> already exists in any state, otherwise create with
# that tag. This is the "resume by name" path — pair off the hint printed
# at Alt+x.
if [ -n "$forced_tag" ]; then
    if session_blocks_reuse "pair-${forced_tag}"; then
        chosen_session="pair-${forced_tag}"
        chosen_tag="$forced_tag"
        action=attach
    else
        chosen_tag="$forced_tag"
        action=create
    fi
elif [ ${#detached_list[@]} -eq 0 ] && [ ${#historical_rows[@]} -eq 0 ]; then
    chosen_tag="$free_slot_tag"
    action=create
    prompt_name=1
else
    new_label="+ new ${BASE_TAG} session"
    # Plain text drives selection matching below; the colored variant is for
    # fzf display only. fzf --ansi strips its color codes from the returned
    # line, so `picked` maps back to one of these exact plain strings.
    options=$({
        if [ ${#detached_list[@]} -gt 0 ]; then printf '%s\n' "${detached_list[@]}"; fi
        if [ ${#historical_rows[@]} -gt 0 ]; then printf '%s\n' "${historical_rows[@]}"; fi
        printf '%s\n' "$new_label"
    })
    if command -v fzf >/dev/null 2>&1; then
        # Color-code: green = live session (picking it attaches); historical
        # rows carry an age-graded grey (set per-row in historical_colored,
        # brighter = more recent) and picking one re-creates by name. The
        # "+ new" row stays default.
        C_ACTIVE=$'\033[32m'; C_RESET=$'\033[0m'
        options_colored=$({
            if [ ${#detached_list[@]} -gt 0 ]; then printf "${C_ACTIVE}%s${C_RESET}\n" "${detached_list[@]}"; fi
            if [ ${#historical_colored[@]} -gt 0 ]; then printf '%s\n' "${historical_colored[@]}"; fi
            printf '%s\n' "$new_label"
        })
        picked=$(printf '%s' "$options_colored" | fzf --ansi --prompt='pair> ' --height=10 --reverse) || exit 0
    else
        echo "Detached / historical pair sessions:"
        printf '%s' "$options" | nl
        printf 'Pick: '
        read -r idx
        picked=$(printf '%s' "$options" | sed -n "${idx}p")
    fi
    [ -z "$picked" ] && exit 0

    if [ "$picked" = "$new_label" ]; then
        chosen_tag="$free_slot_tag"
        action=create
        prompt_name=1
    else
        # Historical rows are annotated as "pair-<tag>  (... ago, ...)" — the
        # double-space separator never appears in detached-row text (sanitized
        # tags can't contain spaces), so trimming everything from the first
        # double-space onward yields the bare "pair-<tag>" for both shapes.
        chosen_session="${picked%%  *}"
        chosen_tag="${chosen_session#pair-}"
        # If the picked tag is currently live → attach; otherwise (historical
        # row, or the unlikely "live tag listed as historical" race) → create
        # by name, which goes through the same path as `pair resume <tag>`.
        if printf '%s\n' "$live_tags_nl" | grep -qx "$chosen_tag"; then
            action=attach
        else
            chosen_session=""
            action=create
        fi
    fi
fi

if [ -z "$chosen_tag" ]; then
    echo "pair: couldn't determine a session to use; bailing." >&2
    exit 1

exec
/bin/zsh -lc "sed -n '1,280p' workshop/plans/000075-go-launcher-prototype-plan.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
# Pair Go Launcher Prototype Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a guarded `pair-go launch` prototype that reaches Pair launcher decision logic without replacing or invoking the public shell launcher.

**Architecture:** Put launcher business logic in a pure `cmd/internal/launcher` package and keep subprocess/filesystem work in thin, injectable seams (`ARCH-PURE`). Extend the existing #74 dispatcher instead of adding a parallel command parser (`ARCH-DRY`). The prototype prints the decision it would take, then exits with an explicit unsupported-after-decision code so the issue delivers a real launcher vertical slice without changing public `bin/pair` behavior (`ARCH-PURPOSE`).

**Tech Stack:** Go standard library, existing `cmd/internal/dispatcher`, fake `zellij` process tests, `go test`, `make pair-go`.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `LaunchArgs` | `cmd/internal/launcher/args.go` | new |
| `Tag` | `cmd/internal/launcher/tag.go` | new |
| `DataDir` | `cmd/internal/launcher/datadir.go` | new |
| `SessionSnapshot` | `cmd/internal/launcher/session.go` | new |
| `LaunchDecision` | `cmd/internal/launcher/decision.go` | new |

**LaunchArgs** — Parsed launch-specific argv: agent, forced tag, forwarded agent args, help flag, and unsupported subcommand errors.
- **Relationships:** 1:1 with a `pair-go launch` invocation; owns a `Tag` only for forced resume.
- **DRY rationale:** Keeps `pair-go launch` parsing behind the existing dispatcher instead of duplicating parser branches in `cmd/pair-go/main.go`.
- **Future extensions:** `continue`, `rename`, and tag-restart prompts widen this parser after the prototype has tests.

**Tag** — Normalized workspace tag, accepting either bare `demo` or `pair-demo` and rejecting empty or non `[A-Za-z0-9_-]` values. The canonical value is always bare (`demo`); zellij session names are derived only at the boundary as `pair-<tag>`.
- **Relationships:** Used by `LaunchArgs`, `SessionSnapshot`, and `LaunchDecision`.
- **DRY rationale:** Mirrors the shell launcher's `normalize_tag` as a named Go concept so later Go launcher work has one validation point.
- **Future extensions:** Length checks can move here when the Go path reaches real zellij session creation.

**DataDir** — Resolved Pair data directory from `XDG_DATA_HOME` or `$HOME/.local/share/pair`.
- **Relationships:** Provides the root for historical sidecars and future config/session files.
- **DRY rationale:** Prevents each command seam from recomputing Pair's data directory.
- **Future extensions:** Asset/data path resolution can join this with future `PAIR_HOME` discovery.

**SessionSnapshot** — In-memory view of active zellij rows and historical tag candidates relevant to the current cwd.
- **Relationships:** 1:N with zellij sessions and historical rows; consumed by `LaunchDecision`.
- **DRY rationale:** Separates "what exists" from "what should we do", matching the shell launcher's implicit stages.
- **Future extensions:** Can add queue badges, age coloring, and config-derived agent inference without changing decision callers.

**LaunchDecision** — Pure create/attach/picker-required decision for forced resume, empty state, detached sessions, and historical tags. It carries the canonical bare `Tag` and, for attach/create decisions that name zellij, the derived `SessionName` (`pair-<tag>`) so comparisons and printouts cannot accidentally mix forms.
- **Relationships:** N:1 from snapshot plus args to one decision.
- **DRY rationale:** Pulls the business rule out of command execution so unit tests do not need zellij/fzf.
- **Future extensions:** Real fzf selection can become another input shape instead of branching inside IO code.

The launcher package must not define a second stdout/stderr/exit-code result type. `cmd/internal/dispatcher.Result` remains the single process-facing result abstraction (`ARCH-DRY`). Launcher functions return domain values (`LaunchDecision`, snapshots, parse errors); the dispatcher route converts those values into `dispatcher.Result`.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ZellijSource` | `cmd/internal/launcher/zellij.go` | new | `zellij list-sessions`, `zellij --session ... action list-clients` |
| `HistorySource` | `cmd/internal/launcher/history.go` | new | filesystem scan of `draft-*.md`, `log-*.md`, `queue-*` under Pair data dir |
| `LaunchCommand` | `cmd/internal/dispatcher/dispatcher.go` | modified | existing dispatcher route for `launch` |
| `PairGoMain` | `cmd/pair-go/main.go` | modified | process stdout/stderr exit handling |

**ZellijSource** — Reads zellij session rows and client counts through `exec.Command`.
- **Injected into:** `launcher.Run`, which converts IO into a pure `SessionSnapshot`.
- **Future extensions:** Timeout behavior can be added when the Go launcher owns real launch flow; do not port the shell `zj` timeout in this prototype unless the process fake exposes the need.

**HistorySource** — Scans cwd-prefixed historical sidecars from the resolved data dir.
- **Injected into:** `launcher.Run`.
- **Future extensions:** Queue badges and age display can be layered onto historical rows when picker UI is implemented.

**LaunchCommand** — Routes `pair-go launch` from the existing dispatcher to the launcher runner.
- **Injected into:** `cmd/pair-go` via the existing `run(args, stdout, stderr)` path.
- **Future extensions:** Other implemented subcommands can follow the same dispatcher pattern.
- **Boundary rule:** production environment reads (`os.Getenv`, `os.Getwd`) and `exec.Command` construction live in a small launcher IO constructor used by the dispatcher route. Tests may call a dispatcher test seam with an explicit launcher runtime. The launcher core returns domain outcomes; only the dispatcher maps those outcomes to stdout/stderr/exit code.

**PairGoMain** — No business logic; writes dispatcher-returned streams and exits.
- **Injected into:** none.
- **Future extensions:** May eventually become the public `pair` binary entrypoint in #77, but not here.

## Chunk 1: Pure Launcher Core

### Task 1: Parse `pair-go launch` Args

**Files:**
- Create: `cmd/internal/launcher/args.go`
- Create: `cmd/internal/launcher/args_test.go`

- [ ] **Step 1: Write failing parse tests**

Cover:
- no args: default agent `claude`;
- `<agent>`: custom agent;
- `<agent> -- <args>` and `-- <args>` forwarding;
- `resume <tag>` strips `pair-` and records forced tag;
- unexpected extra positional includes `unexpected positional arg` and `use '--' to forward args to the agent`;
- unsupported `continue`, `rename`, and `list` under `launch` return explicit prototype errors.

Run: `go test ./cmd/internal/launcher -run 'TestParseLaunchArgs' -count=1`
Expected: FAIL because package/files do not exist.

- [ ] **Step 2: Implement minimal parser**

Create `LaunchArgs`, `ParseArgs(args []string) (LaunchArgs, error)`, and a typed `UsageError`.

- [ ] **Step 3: Verify parse tests pass**

Run: `go test ./cmd/internal/launcher -run 'TestParseLaunchArgs' -count=1`
Expected: PASS.

### Task 2: Add Tag and Data-Dir Pure Helpers

**Files:**
- Create: `cmd/internal/launcher/tag.go`
- Create: `cmd/internal/launcher/tag_test.go`
- Create: `cmd/internal/launcher/datadir.go`
- Create: `cmd/internal/launcher/datadir_test.go`

- [ ] **Step 1: Write failing helper tests**

Cover:
- `NormalizeTag("pair-demo") == "demo"`;
- invalid characters and empty string return errors;
- `DefaultTag("/Users/xianxu/workspace/pair") == "pair"`;
- empty/symbol-only cwd basename falls back to `pair`;
- `ResolveDataDir(home, xdg)` returns `$XDG_DATA_HOME/pair` or `$HOME/.local/share/pair`.

Run: `go test ./cmd/internal/launcher -run 'TestNormalizeTag|TestDefaultTag|TestResolveDataDir' -count=1`
Expected: FAIL.

- [ ] **Step 2: Implement helpers**

Keep these functions pure; do not read environment variables directly inside them.

- [ ] **Step 3: Verify helper tests pass**

Run: `go test ./cmd/internal/launcher -run 'TestNormalizeTag|TestDefaultTag|TestResolveDataDir' -count=1`
Expected: PASS.

### Task 3: Model Sessions and Decisions

**Files:**
- Create: `cmd/internal/launcher/session.go`
- Create: `cmd/internal/launcher/decision.go`
- Create: `cmd/internal/launcher/decision_test.go`

- [ ] **Step 1: Write failing decision tests**

Cover:
- forced resume + blocking session -> attach `pair-<tag>`;
- forced resume + no blocking session -> create canonical bare tag `<tag>` with derived session name `pair-<tag>` and no prompt;
- no detached/no historical -> create next free tag and prompt;
- detached or historical present -> picker required;
- selected historical row -> create canonical bare tag with derived session name;
- exited rows do not block reuse;
- live and detached rows block reuse.

Run: `go test ./cmd/internal/launcher -run 'TestDecideLaunch' -count=1`
Expected: FAIL.

- [ ] **Step 2: Implement models**

Add small structs:

```go
type SessionState string

const (
    SessionAttached SessionState = "attached"
    SessionDetached SessionState = "detached"
    SessionExited   SessionState = "exited"
)

type LaunchAction string

const (
    ActionAttach LaunchAction = "attach"
    ActionCreate LaunchAction = "create"
    ActionPick   LaunchAction = "pick"
)
```

Keep `DecideLaunch(args LaunchArgs, snap SessionSnapshot) (LaunchDecision, error)` pure.

- [ ] **Step 3: Verify decision tests pass**

Run: `go test ./cmd/internal/launcher -run 'TestDecideLaunch' -count=1`
Expected: PASS.

- [ ] **Step 4: Commit pure core**

```bash
git add cmd/internal/launcher
git commit -m "#75: model Go launcher decisions" -m "Add pure launch argument, tag, data-dir, session, and decision models for the guarded Go launcher prototype." -m "Co-Authored-By: GPT-5 Codex <codex@openai.com>"
```

## Chunk 2: Thin IO Runner and Dispatcher Route

### Task 4: Add Fakeable Zellij and History Sources

**Files:**
- Create: `cmd/internal/launcher/zellij.go`
- Create: `cmd/internal/launcher/history.go`
- Create: `cmd/internal/launcher/run.go`
- Create: `cmd/internal/launcher/run_test.go`

- [ ] **Step 1: Write failing runner tests with in-memory fakes**

Cover:
- runner uses supplied environment/cwd/data-dir fields, not global process state;
- fake zellij rows become `SessionSnapshot` rows;
- fake historical files become historical candidates;
- runner returns a domain outcome that identifies help, parse failure, or a valid decision that is intentionally unsupported after the decision phase.

Run: `go test ./cmd/internal/launcher -run 'TestRunLaunch' -count=1`
Expected: FAIL.

- [ ] **Step 2: Implement interfaces and runner**

Use interfaces:

```go
type SessionLister interface {
    ListSessions() ([]ZellijSession, error)
    ClientCount(session string) (int, error)
}

type HistoricalScanner interface {
    Scan(base string, cutoff time.Time) ([]HistoricalTag, error)
}
```

`Run(args []string, env Env, sessions SessionLister, history HistoricalScanner) (LaunchOutcome, error)` should return domain values only:
- help outcome for `launch help`;
- typed parse/usage errors for invalid args;
- decision outcome for valid create/attach/pick decisions.

The dispatcher route, not the launcher package, maps those outcomes to `dispatcher.Result`:
- exit `0` for help;
- exit `2` for parse errors;
- exit `3` for a valid decision that is intentionally unsupported after the decision phase.

- [ ] **Step 3: Verify runner tests pass**

Run: `go test ./cmd/internal/launcher -run 'TestRunLaunch' -count=1`
Expected: PASS.

### Task 5: Route `pair-go launch`

**Files:**
- Modify: `cmd/internal/dispatcher/dispatcher.go`
- Modify: `cmd/internal/dispatcher/dispatcher_test.go`
- Modify: `cmd/pair-go/main.go`
- Modify: `cmd/pair-go/main_test.go`

- [ ] **Step 1: Update dispatcher tests**

Change the existing planned-command test so `wrap` still reports planned/unsupported, but `launch` routes to the implemented prototype and no longer says "planned but not implemented".

Run: `go test ./cmd/internal/dispatcher -run 'TestDispatch' -count=1`
Expected: FAIL until route is wired.

- [ ] **Step 2: Implement the route**

Add a dispatcher branch for `launch` that delegates to the launcher package. Keep all other planned families unchanged. The dispatcher route constructs `dispatcher.Result` from launcher domain values; the launcher package must not define a parallel stdout/stderr/exit-code result type. For production, the dispatcher route uses a small launcher runtime constructor that reads environment/cwd and creates the real zellij/history sources. For tests, expose a package-private or exported test seam that accepts an explicit launcher runtime.

- [ ] **Step 3: Update `pair-go` tests**

Assert:
- `pair-go help` still lists launch as the guarded prototype;
- `pair-go launch --help` prints launch usage;
- `pair-go launch resume demo` returns a prototype decision message, not a real launch.

Run: `go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
Expected: PASS.

- [ ] **Step 4: Commit IO and route**

```bash
git add cmd/internal/launcher cmd/internal/dispatcher cmd/pair-go
git commit -m "#75: add guarded Go launcher prototype" -m "Route pair-go launch through a fakeable launcher runner that reaches create/attach decisions without replacing bin/pair." -m "Co-Authored-By: GPT-5 Codex <codex@openai.com>"
```


codex
One plan item stands out: the plan’s integration table mentions `queue-*`, but the implemented history scanner only reads draft/log sidecars. I’m checking whether that changes the issue’s actual Done-when or is just a stale plan detail.
exec
/bin/zsh -lc "sed -n '280,620p' workshop/plans/000075-go-launcher-prototype-plan.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:

### Task 6: Add Process-Level Fake Test

**Files:**
- Create: `cmd/pair-go/launch_process_test.go`
- Modify: `cmd/pair-go/main.go`

- [ ] **Step 1: Write failing process-level fake test**

The test should:
- create a temp `PATH` with fake `zellij`;
- set temp `HOME`/`XDG_DATA_HOME`;
- create draft/log sidecars for a historical tag;
- invoke the real routed path with `run([]string{"launch", ...}, stdout, stderr)`;
- assert the output names the prototype decision and never invokes real zellij attach/new-session.

Fake `zellij` contract:
- `zellij list-sessions --short` prints newline-separated session names such as `pair-live` and `pair-detached`;
- `zellij list-sessions --no-formatting` prints rows where exited sessions include `EXITED`;
- `zellij --session <name> action list-clients` prints a header plus one client row for attached sessions, and only the header for detached sessions;
- any `attach`, `--new-session-with-layout`, `new-session`, or `delete-session` invocation appends to a log and exits with a test failure marker so the assertion proves the prototype did not launch or mutate zellij state.

Run: `go test ./cmd/pair-go -run 'TestRunLaunchWithFakeZellij' -count=1`
Expected: FAIL until the dispatcher/cmd path has an injectable launcher runtime seam.

- [ ] **Step 2: Make the dispatcher/cmd path support injected launcher runtime for tests**

Keep production `main()` simple. It may call a small `runWithRuntime(args, stdout, stderr, runtime)` helper if needed, but `cmd/pair-go` must still contain no launcher business logic. Environment reads and command lookup belong to the launcher IO constructor or dispatcher route, not the pure launcher core.

- [ ] **Step 3: Verify process test passes**

Run: `go test ./cmd/pair-go -run 'TestRunLaunchWithFakeZellij' -count=1`
Expected: PASS.

## Chunk 3: Documentation and Verification

### Task 7: Document Remaining Shell-Owned Behavior

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `atlas/go-migration-inventory.md`

- [ ] **Step 1: Update atlas**

Record that `pair-go launch` now owns only the decision-phase prototype. Explicitly list shell-owned behavior still out of scope: real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller.

- [ ] **Step 2: Verify docs mention the boundary**

Run: `rg -n "pair-go launch|decision-phase|shell-owned" atlas/architecture.md atlas/go-migration-inventory.md`
Expected: matches in both files.

- [ ] **Step 3: Commit docs**

```bash
git add atlas/architecture.md atlas/go-migration-inventory.md
git commit -m "#75: document launcher prototype boundary" -m "Clarify that pair-go launch is a guarded decision-phase prototype while bin/pair remains the public launcher." -m "Co-Authored-By: GPT-5 Codex <codex@openai.com>"
```

### Task 8: Final Verification

**Files:**
- No planned edits.

- [ ] **Step 1: Run focused Go tests**

Run: `go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
Expected: PASS.

- [ ] **Step 2: Build `pair-go`**

Run: `make -B pair-go`
Expected: builds `bin/pair-go`.

- [ ] **Step 3: Run full Go suite**

Run: `go test ./... -count=1`
Expected: PASS.

- [ ] **Step 4: Verify public launcher unchanged**

Run: `git diff -- bin/pair`
Expected: empty output.

- [ ] **Step 5: Verify docs and whitespace**

Run: `rg -n "pair-go launch|decision-phase|shell-owned" atlas/architecture.md atlas/go-migration-inventory.md && git diff --check`
Expected: atlas matches and no whitespace errors.

- [ ] **Step 6: Close through SDLC**

Run: `sdlc close --issue 75 --verified 'go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; git diff -- bin/pair empty; rg atlas boundary check; git diff --check'`
Expected: close gate runs its mandatory review and records the verdict.

exec
/bin/zsh -lc 'git diff e89e2cea484ab8ca7ec9ca518fd2bf04ba38a8fe..HEAD -- workshop/plans/000075-go-launcher-prototype-plan.md workshop/issues/000075-go-launcher-prototype.md' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
diff --git a/workshop/issues/000075-go-launcher-prototype.md b/workshop/issues/000075-go-launcher-prototype.md
index fc08355..6f15faa 100644
--- a/workshop/issues/000075-go-launcher-prototype.md
+++ b/workshop/issues/000075-go-launcher-prototype.md
@@ -38,19 +38,19 @@ The implementation should keep business decisions pure (`ARCH-PURE`) and reuse t
 
 ## Done when
 
-- [ ] A guarded Go launcher path can exercise a documented subset of launcher behavior.
-- [ ] Existing `bin/pair` remains the default public launcher.
-- [ ] Tests cover the ported decision logic and at least one process-level fake for external commands.
-- [ ] Any behavior not yet ported fails explicitly rather than silently diverging.
-- [ ] Pair remains usable after merge through the existing public command.
+- [x] A guarded Go launcher path can exercise a documented subset of launcher behavior.
+- [x] Existing `bin/pair` remains the default public launcher.
+- [x] Tests cover the ported decision logic and at least one process-level fake for external commands.
+- [x] Any behavior not yet ported fails explicitly rather than silently diverging.
+- [x] Pair remains usable after merge through the existing public command.
 
 ## Plan
 
-- [ ] Select the launcher slice from #73.
-- [ ] Extract pure decision models and tests.
-- [ ] Add fake-command process tests for the selected slice.
-- [ ] Implement the guarded Go path.
-- [ ] Document remaining shell-owned launcher behavior.
+- [x] Select the launcher slice from #73.
+- [x] Extract pure decision models and tests.
+- [x] Add fake-command process tests for the selected slice.
+- [x] Implement the guarded Go path.
+- [x] Document remaining shell-owned launcher behavior.
 
 ## Estimate
 
@@ -83,3 +83,5 @@ Plan-quality gate returned FAILURE: estimate was low for the visible multi-surfa
 Second plan-quality gate returned FAILURE: the process-level test was ordered before the dispatcher route it needs, and the plan risked duplicating `dispatcher.Result`. Reordered route before process test and made `dispatcher.Result` the sole process-facing result abstraction (`ARCH-DRY`).
 
 Third plan-quality gate returned FAILURE: Task 4 still named a `LaunchResult` return despite the single-result-abstraction rule. Revised the runner contract so launcher returns domain `LaunchOutcome` values and dispatcher alone maps to `dispatcher.Result`; also named the production IO constructor and test runtime seam.
+
+Implemented `cmd/internal/launcher` as a pure decision-phase core plus fakeable zellij/history seams. `pair-go launch` now routes through the #74 dispatcher, returns explicit prototype decisions, and does not mutate zellij or replace `bin/pair`. Updated atlas architecture and the Go migration inventory to record the shell-owned boundary.

exec
/bin/zsh -lc "git diff e89e2cea484ab8ca7ec9ca518fd2bf04ba38a8fe..HEAD -- cmd/internal/launcher cmd/internal/dispatcher cmd/pair-go | sed -n '1,260p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
diff --git a/cmd/internal/dispatcher/dispatcher.go b/cmd/internal/dispatcher/dispatcher.go
index dddfe46..1b9710a 100644
--- a/cmd/internal/dispatcher/dispatcher.go
+++ b/cmd/internal/dispatcher/dispatcher.go
@@ -2,7 +2,11 @@ package dispatcher
 
 import (
 	"fmt"
+	"os"
 	"strings"
+	"time"
+
+	"github.com/xianxu/pair/cmd/internal/launcher"
 )
 
 const programName = "pair-go"
@@ -49,6 +53,8 @@ func Dispatch(args []string) Result {
 			Stdout:   "pair-go dispatcher skeleton\npublic launcher: bin/pair\n",
 			ExitCode: 0,
 		}
+	case "launch":
+		return DispatchWithLauncherRuntime(args, osLauncherRuntime())
 	}
 
 	if family, ok := familyByName(args[0]); ok {
@@ -64,6 +70,93 @@ func Dispatch(args []string) Result {
 	}
 }
 
+type LauncherRuntime struct {
+	Env      launcher.Env
+	Sessions launcher.SessionSource
+	History  launcher.HistoricalScanner
+}
+
+type StaticSessions struct {
+	Sessions []launcher.Session
+	Err      error
+}
+
+func (s StaticSessions) Snapshot() ([]launcher.Session, error) {
+	if s.Err != nil {
+		return nil, s.Err
+	}
+	return s.Sessions, nil
+}
+
+type StaticHistory struct {
+	Tags []launcher.HistoricalTag
+	Err  error
+}
+
+func (h StaticHistory) Scan(_ string, _ time.Time) ([]launcher.HistoricalTag, error) {
+	if h.Err != nil {
+		return nil, h.Err
+	}
+	return h.Tags, nil
+}
+
+func DispatchWithLauncherRuntime(args []string, rt LauncherRuntime) Result {
+	launchArgs := []string(nil)
+	if len(args) > 1 {
+		launchArgs = args[1:]
+	}
+	if len(launchArgs) == 0 || launchArgs[0] == "help" || launchArgs[0] == "--help" || launchArgs[0] == "-h" {
+		return Result{Stdout: LaunchHelp(programName), ExitCode: 0}
+	}
+	outcome, err := launcher.Run(launchArgs, rt.Env, rt.Sessions, rt.History)
+	if err != nil {
+		return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
+	}
+	decision := outcome.Decision
+	return Result{
+		Stderr: fmt.Sprintf(
+			"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
+			decision.Action,
+			decision.Tag,
+			decision.SessionName,
+		),
+		ExitCode: 3,
+	}
+}
+
+func LaunchHelp(program string) string {
+	return fmt.Sprintf(`Usage: %s launch [agent] [-- agent-args...]
+       %s launch resume <tag>
+
+Guarded decision-phase prototype. Public sessions still start through bin/pair.
+This command parses launch inputs and computes the create/attach/picker decision,
+then stops before invoking zellij.
+`, program, program)
+}
+
+func LauncherEnv(home, xdgDataHome, cwd string) launcher.Env {
+	return launcher.Env{
+		Home:     home,
+		XDGData:  xdgDataHome,
+		Cwd:      cwd,
+		Now:      time.Now(),
+		HistoryD: 14,
+	}
+}
+
+func osLauncherRuntime() LauncherRuntime {
+	home := os.Getenv("HOME")
+	xdg := os.Getenv("XDG_DATA_HOME")
+	cwd, _ := os.Getwd()
+	env := LauncherEnv(home, xdg, cwd)
+	dataDir := launcher.ResolveDataDir(home, xdg)
+	return LauncherRuntime{
+		Env:      env,
+		Sessions: launcher.ZellijSource{},
+		History:  launcher.HistorySource{DataDir: dataDir},
+	}
+}
+
 // Help renders the development-only dispatcher usage text.
 func Help(program string) string {
 	var b strings.Builder
diff --git a/cmd/internal/dispatcher/dispatcher_test.go b/cmd/internal/dispatcher/dispatcher_test.go
index c9fed26..1d1358a 100644
--- a/cmd/internal/dispatcher/dispatcher_test.go
+++ b/cmd/internal/dispatcher/dispatcher_test.go
@@ -60,6 +60,42 @@ func TestDispatchPlannedCommandReturnsUnsupported(t *testing.T) {
 	}
 }
 
+func TestDispatchLaunchHelpRoutesToPrototype(t *testing.T) {
+	res := Dispatch([]string{"launch", "--help"})
+	if res.ExitCode != 0 {
+		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
+	}
+	if res.Stderr != "" {
+		t.Fatalf("Stderr = %q, want empty", res.Stderr)
+	}
+	for _, want := range []string{"Usage: pair-go launch", "decision-phase prototype"} {
+		if !strings.Contains(res.Stdout, want) {
+			t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
+		}
+	}
+}
+
+func TestDispatchLaunchReturnsPrototypeDecision(t *testing.T) {
+	res := DispatchWithLauncherRuntime([]string{"launch", "resume", "demo"}, LauncherRuntime{
+		Env: LauncherEnv("/home/me", "", "/work/pair"),
+		Sessions: StaticSessions{
+			Sessions: nil,
+		},
+		History: StaticHistory{},
+	})
+	if res.ExitCode != 3 {
+		t.Fatalf("ExitCode = %d, want 3", res.ExitCode)
+	}
+	if res.Stdout != "" {
+		t.Fatalf("Stdout = %q, want empty", res.Stdout)
+	}
+	for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
+		if !strings.Contains(res.Stderr, want) {
+			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
+		}
+	}
+}
+
 func TestDispatchUnknownCommandReturnsUsageHint(t *testing.T) {
 	res := Dispatch([]string{"frobnicate"})
 	if res.ExitCode != 2 {
diff --git a/cmd/internal/launcher/args.go b/cmd/internal/launcher/args.go
new file mode 100644
index 0000000..06aa8fb
--- /dev/null
+++ b/cmd/internal/launcher/args.go
@@ -0,0 +1,71 @@
+package launcher
+
+import "fmt"
+
+// LaunchArgs is the pure parse result for the guarded pair-go launch prototype.
+type LaunchArgs struct {
+	Agent       string
+	ForcedTag   string
+	SelectedTag string
+	AgentArgs   []string
+}
+
+// UsageError is an operator-facing parse error.
+type UsageError struct {
+	Message string
+}
+
+func (e UsageError) Error() string {
+	return e.Message
+}
+
+// ParseArgs parses pair-go launch args. It intentionally supports only the
+// decision-phase subset for #75; unsupported shell-owned launcher verbs fail
+// explicitly.
+func ParseArgs(argv []string) (LaunchArgs, error) {
+	var out LaunchArgs
+	if len(argv) == 0 {
+		out.Agent = "claude"
+		return out, nil
+	}
+
+	switch argv[0] {
+	case "continue", "rename", "list", "ls":
+		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %s is not implemented by pair-go launch; use bin/pair", argv[0])}
+	case "resume":
+		if len(argv) < 2 {
+			return LaunchArgs{}, UsageError{Message: "pair-go launch: 'resume' requires a tag"}
+		}
+		tag, err := NormalizeTag(argv[1])
+		if err != nil {
+			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: invalid tag: %v", err)}
+		}
+		if len(argv) > 2 {
+			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", argv[2])}
+		}
+		out.ForcedTag = tag
+		return out, nil
+	}
+
+	seenSeparator := false
+	for _, arg := range argv {
+		if seenSeparator {
+			out.AgentArgs = append(out.AgentArgs, arg)
+			continue
+		}
+		if arg == "--" {
+			seenSeparator = true
+			continue
+		}
+		if out.Agent == "" {
+			out.Agent = arg
+			continue
+		}
+		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", arg)}
+	}
+
+	if out.Agent == "" {
+		out.Agent = "claude"
+	}
+	return out, nil
+}
diff --git a/cmd/internal/launcher/args_test.go b/cmd/internal/launcher/args_test.go
new file mode 100644
index 0000000..85bc25e
--- /dev/null
+++ b/cmd/internal/launcher/args_test.go
@@ -0,0 +1,88 @@
+package launcher
+
+import (
+	"strings"
+	"testing"
+)
+
+func TestParseLaunchArgsDefaultsToClaude(t *testing.T) {
+	args, err := ParseArgs(nil)
+	if err != nil {
+		t.Fatalf("ParseArgs returned error: %v", err)

exec
/bin/zsh -lc 'git status --short' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 M workshop/issues/000075-go-launcher-prototype.md

exec
/bin/zsh -lc 'go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1' in /Users/xianxu/workspace/pair
 succeeded in 976ms:
ok  	github.com/xianxu/pair/cmd/internal/launcher	0.670s
ok  	github.com/xianxu/pair/cmd/internal/dispatcher	0.200s
ok  	github.com/xianxu/pair/cmd/pair-go	0.851s

codex
The focused Go tests pass locally. I’m also noting the worktree has an unstaged issue-file modification, so I’m treating the supplied base-to-HEAD diff plus current tracker text as review evidence, not making any edits.
exec
/bin/zsh -lc 'git diff -- workshop/issues/000075-go-launcher-prototype.md' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
diff --git a/workshop/issues/000075-go-launcher-prototype.md b/workshop/issues/000075-go-launcher-prototype.md
index 6f15faa..bb689b8 100644
--- a/workshop/issues/000075-go-launcher-prototype.md
+++ b/workshop/issues/000075-go-launcher-prototype.md
@@ -1,12 +1,13 @@
 ---
 id: 000075
-status: working
+status: done
 deps: [000073, 000074]
 github_issue:
 created: 2026-06-26
 updated: 2026-06-29
 estimate_hours: 3.75
 started: 2026-06-29T21:55:46-07:00
+actual_hours: 0.98
 ---
 
 # pair Go launcher prototype
@@ -75,6 +76,7 @@ total: 3.75
 Created from #72. This issue is deliberately a prototype/vertical-slice milestone, not the public switch.
 
 ### 2026-06-29
+- 2026-06-29: closed — go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; git diff -- bin/pair empty; rg atlas boundary check; git diff --check
 
 Claimed #75 after parking #82. Entered planning with `sdlc start-plan --issue 75`; design cites #73's inventory and chooses a guarded `pair-go launch` decision-phase prototype so the shell launcher remains public while the Go path exercises real launcher concepts (`ARCH-PURE`, `ARCH-DRY`, `ARCH-PURPOSE`).
 

exec
/bin/zsh -lc 'rg -n "queue|HistorySource|draft|log|sidecar|SelectedTag|Picker|pick|sessionBlocksReuse|exited|EXITED" cmd/internal/launcher cmd/internal/dispatcher cmd/pair-go workshop/issues/000075-go-launcher-prototype.md workshop/plans/000075-go-launcher-prototype-plan.md' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
workshop/issues/000075-go-launcher-prototype.md:17:The launcher is the largest remaining shell surface and the most important packaging target, but it owns many behavioral edges: session picker, tag normalization, resume/continue/rename, zellij lifecycle, quit/restart markers, data-dir migrations, orphan cleanup, cmux title ownership, and dev rebuild behavior. Porting it must not break normal Pair usage.
workshop/issues/000075-go-launcher-prototype.md:30:- a session snapshot model that represents live, detached, exited, and historical tag candidates;
workshop/issues/000075-go-launcher-prototype.md:31:- a pure decision model for forced resume attach/create, direct create, picker-required, and historical create;
workshop/issues/000075-go-launcher-prototype.md:32:- a thin command/filesystem shell that can read fake `zellij` output and filesystem sidecars in tests, then print the selected prototype action.
workshop/issues/000075-go-launcher-prototype.md:44:- [x] Tests cover the ported decision logic and at least one process-level fake for external commands.
workshop/issues/000075-go-launcher-prototype.md:58:*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*
workshop/issues/000075-go-launcher-prototype.md:61:model: estimate-logic-v3.1
workshop/plans/000075-go-launcher-prototype-plan.md:5:**Goal:** Add a guarded `pair-go launch` prototype that reaches Pair launcher decision logic without replacing or invoking the public shell launcher.
workshop/plans/000075-go-launcher-prototype-plan.md:7:**Architecture:** Put launcher business logic in a pure `cmd/internal/launcher` package and keep subprocess/filesystem work in thin, injectable seams (`ARCH-PURE`). Extend the existing #74 dispatcher instead of adding a parallel command parser (`ARCH-DRY`). The prototype prints the decision it would take, then exits with an explicit unsupported-after-decision code so the issue delivers a real launcher vertical slice without changing public `bin/pair` behavior (`ARCH-PURPOSE`).
workshop/plans/000075-go-launcher-prototype-plan.md:36:- **Relationships:** Provides the root for historical sidecars and future config/session files.
workshop/plans/000075-go-launcher-prototype-plan.md:43:- **Future extensions:** Can add queue badges, age coloring, and config-derived agent inference without changing decision callers.
workshop/plans/000075-go-launcher-prototype-plan.md:45:**LaunchDecision** — Pure create/attach/picker-required decision for forced resume, empty state, detached sessions, and historical tags. It carries the canonical bare `Tag` and, for attach/create decisions that name zellij, the derived `SessionName` (`pair-<tag>`) so comparisons and printouts cannot accidentally mix forms.
workshop/plans/000075-go-launcher-prototype-plan.md:57:| `HistorySource` | `cmd/internal/launcher/history.go` | new | filesystem scan of `draft-*.md`, `log-*.md`, `queue-*` under Pair data dir |
workshop/plans/000075-go-launcher-prototype-plan.md:65:**HistorySource** — Scans cwd-prefixed historical sidecars from the resolved data dir.
workshop/plans/000075-go-launcher-prototype-plan.md:67:- **Future extensions:** Queue badges and age display can be layered onto historical rows when picker UI is implemented.
workshop/plans/000075-go-launcher-prototype-plan.md:74:**PairGoMain** — No business logic; writes dispatcher-returned streams and exits.
workshop/plans/000075-go-launcher-prototype-plan.md:150:- detached or historical present -> picker required;
workshop/plans/000075-go-launcher-prototype-plan.md:152:- exited rows do not block reuse;
workshop/plans/000075-go-launcher-prototype-plan.md:168:    SessionExited   SessionState = "exited"
workshop/plans/000075-go-launcher-prototype-plan.md:176:    ActionPick   LaunchAction = "pick"
workshop/plans/000075-go-launcher-prototype-plan.md:233:- decision outcome for valid create/attach/pick decisions.
workshop/plans/000075-go-launcher-prototype-plan.md:292:- create draft/log sidecars for a historical tag;
workshop/plans/000075-go-launcher-prototype-plan.md:298:- `zellij list-sessions --no-formatting` prints rows where exited sessions include `EXITED`;
workshop/plans/000075-go-launcher-prototype-plan.md:300:- any `attach`, `--new-session-with-layout`, `new-session`, or `delete-session` invocation appends to a log and exits with a test failure marker so the assertion proves the prototype did not launch or mutate zellij state.
workshop/plans/000075-go-launcher-prototype-plan.md:307:Keep production `main()` simple. It may call a small `runWithRuntime(args, stdout, stderr, runtime)` helper if needed, but `cmd/pair-go` must still contain no launcher business logic. Environment reads and command lookup belong to the launcher IO constructor or dispatcher route, not the pure launcher core.
cmd/pair-go/launch_process_test.go:22:	log := filepath.Join(rt, "zellij.log")
cmd/pair-go/launch_process_test.go:25:printf '%s\n' "$*" >> "` + log + `"
cmd/pair-go/launch_process_test.go:27:  "list-sessions --short") printf 'pair-live\npair-detached\npair-exited\n' ;;
cmd/pair-go/launch_process_test.go:28:  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-exited [Created] (EXITED - attach to resurrect)\n' ;;
cmd/pair-go/launch_process_test.go:31:  *attach*|*new-session*|*--new-session-with-layout*|*delete-session*) printf 'MUTATING %s\n' "$*" >> "` + log + `"; exit 99 ;;
cmd/pair-go/launch_process_test.go:38:	historical := filepath.Join(data, "pair", "draft-pair-old.md")
cmd/pair-go/launch_process_test.go:39:	if err := os.WriteFile(historical, []byte("draft"), 0o644); err != nil {
cmd/pair-go/launch_process_test.go:59:	for _, want := range []string{"prototype decision", "action=pick"} {
cmd/pair-go/launch_process_test.go:64:	logBytes, err := os.ReadFile(log)
cmd/pair-go/launch_process_test.go:68:	if strings.Contains(string(logBytes), "MUTATING") {
cmd/pair-go/launch_process_test.go:69:		t.Fatalf("fake zellij recorded mutating invocation:\n%s", string(logBytes))
cmd/internal/dispatcher/dispatcher.go:36:		{Name: "changelog", Summary: "TTY transcript to distilled change log", Status: "planned"},
cmd/internal/dispatcher/dispatcher.go:38:		{Name: "scribe", Summary: "PTY logging wrapper", Status: "planned"},
cmd/internal/dispatcher/dispatcher.go:132:This command parses launch inputs and computes the create/attach/picker decision,
cmd/internal/dispatcher/dispatcher.go:156:		History:  launcher.HistorySource{DataDir: dataDir},
cmd/internal/launcher/zellij_test.go:11:	log := filepath.Join(dir, "zellij.log")
cmd/internal/launcher/zellij_test.go:14:printf '%s\n' "$*" >> "` + log + `"
cmd/internal/launcher/zellij_test.go:17:  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-gone [Created] (EXITED - attach to resurrect)\n' ;;
cmd/internal/launcher/decision_test.go:41:func TestDecideLaunchShowsPickerWhenDetachedOrHistoricalExist(t *testing.T) {
cmd/internal/launcher/decision_test.go:51:			t.Fatalf("decision = %#v, want picker", decision)
cmd/internal/launcher/decision_test.go:57:	decision, err := DecideLaunch(LaunchArgs{Agent: "claude", SelectedTag: "pair-old"}, SessionSnapshot{
cmd/internal/launcher/run_test.go:51:func TestRunLaunchTurnsFakeSessionsIntoPickerDecision(t *testing.T) {
cmd/internal/launcher/run_test.go:62:		t.Fatalf("Decision = %#v, want picker", outcome.Decision)
cmd/internal/launcher/run_test.go:66:func TestRunLaunchTurnsFakeHistoryIntoPickerDecision(t *testing.T) {
cmd/internal/launcher/run_test.go:77:		t.Fatalf("Decision = %#v, want picker", outcome.Decision)
cmd/internal/launcher/args.go:9:	SelectedTag string
cmd/internal/launcher/session.go:9:	SessionExited   SessionState = "exited"
cmd/internal/launcher/history.go:11:// HistorySource scans Pair draft/log sidecars under the data directory.
cmd/internal/launcher/history.go:12:type HistorySource struct {
cmd/internal/launcher/history.go:16:func (s HistorySource) Scan(base string, cutoff time.Time) ([]HistoricalTag, error) {
cmd/internal/launcher/history.go:18:	for _, pattern := range []string{"draft-*.md", "log-*.md"} {
cmd/internal/launcher/history.go:54:	case strings.HasPrefix(name, "draft-") && strings.HasSuffix(name, ".md"):
cmd/internal/launcher/history.go:55:		return strings.TrimSuffix(strings.TrimPrefix(name, "draft-"), ".md"), true
cmd/internal/launcher/history.go:56:	case strings.HasPrefix(name, "log-") && strings.HasSuffix(name, ".md"):
cmd/internal/launcher/history.go:57:		return strings.TrimSuffix(strings.TrimPrefix(name, "log-"), ".md"), true
cmd/internal/launcher/history_test.go:10:func TestHistorySourceScansCwdPrefixedDraftAndLogSidecars(t *testing.T) {
cmd/internal/launcher/history_test.go:13:	for _, name := range []string{"draft-pair.md", "log-pair-old.md", "draft-other.md"} {
cmd/internal/launcher/history_test.go:23:	got, err := HistorySource{DataDir: dir}.Scan("pair", now.Add(-24*time.Hour))
cmd/internal/launcher/zellij.go:24:	exited := exitedSessions(string(raw))
cmd/internal/launcher/zellij.go:31:		if exited[name] {
cmd/internal/launcher/zellij.go:68:func exitedSessions(raw string) map[string]bool {
cmd/internal/launcher/zellij.go:75:		if strings.Contains(line, "EXITED") {
cmd/internal/launcher/decision.go:9:	ActionPick   LaunchAction = "pick"
cmd/internal/launcher/decision.go:12:// LaunchDecision is a pure create/attach/pick decision. Tag is canonical bare
cmd/internal/launcher/decision.go:23:	if args.SelectedTag != "" {
cmd/internal/launcher/decision.go:24:		return createDecision(args.SelectedTag, false), nil
cmd/internal/launcher/decision.go:27:		if sessionBlocksReuse(snap, sessionName(args.ForcedTag)) {
cmd/internal/launcher/decision.go:59:func sessionBlocksReuse(snap SessionSnapshot, name string) bool {
cmd/internal/launcher/decision.go:75:		if !sessionBlocksReuse(snap, sessionName(tag)) && !isHistorical(snap, tag) {

exec
/bin/zsh -lc 'git diff e89e2cea484ab8ca7ec9ca518fd2bf04ba38a8fe..HEAD -- workshop/plans/000075-go-launcher-prototype-plan.md' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
exec
/bin/zsh -lc 'go test ./... -count=1' in /Users/xianxu/workspace/pair
 succeeded in 15949ms:
ok  	github.com/xianxu/pair/cmd/internal/adapt	0.173s
ok  	github.com/xianxu/pair/cmd/internal/ctxmeter	0.303s
ok  	github.com/xianxu/pair/cmd/internal/dispatcher	0.504s
ok  	github.com/xianxu/pair/cmd/internal/launcher	2.215s
ok  	github.com/xianxu/pair/cmd/internal/model	3.505s
ok  	github.com/xianxu/pair/cmd/internal/transcript	0.902s
ok  	github.com/xianxu/pair/cmd/pair-changelog	15.620s
ok  	github.com/xianxu/pair/cmd/pair-context	3.645s
ok  	github.com/xianxu/pair/cmd/pair-continuation	5.626s
ok  	github.com/xianxu/pair/cmd/pair-go	2.716s
?   	github.com/xianxu/pair/cmd/pair-scribe	[no test files]
ok  	github.com/xianxu/pair/cmd/pair-scrollback-render	1.666s
ok  	github.com/xianxu/pair/cmd/pair-slug	6.630s
ok  	github.com/xianxu/pair/cmd/pair-wrap	1.934s

exec
/bin/zsh -lc "nl -ba workshop/issues/000075-go-launcher-prototype.md | sed -n '21,36p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
    21	Prototype the launcher core in Go behind an alternate or guarded path. The prototype should implement a coherent vertical slice of `bin/pair` behavior while the shell launcher remains the public default.
    22	
    23	The #73 inventory identifies `bin/pair` as the P0 public launcher surface. For this milestone, the guarded path is `pair-go launch`, a development-only launcher prototype that reaches the decision boundary but never starts or attaches a real zellij session. The public `bin/pair` shell launcher remains the only supported user entrypoint.
    24	
    25	The vertical slice is:
    26	
    27	- argv parsing for `pair-go launch`, including default agent, `resume <tag>`, optional agent positional, and `--` forwarded args;
    28	- tag normalization and default tag derivation using the same bare-or-`pair-` contract as `bin/pair`;
    29	- data-dir resolution from `XDG_DATA_HOME` / `HOME`;
    30	- a session snapshot model that represents live, detached, exited, and historical tag candidates;
    31	- a pure decision model for forced resume attach/create, direct create, picker-required, and historical create;
    32	- a thin command/filesystem shell that can read fake `zellij` output and filesystem sidecars in tests, then print the selected prototype action.
    33	
    34	The prototype must fail explicitly after the decision phase, rather than silently launching or diverging from `bin/pair`. Unsupported launcher behavior stays shell-owned and documented as out of scope: `continue`, `rename`, zellij lifecycle execution, quit/restart markers, orphan cleanup, cmux title ownership, dev rebuild, tag-restart prompt, config migration, and real fzf interaction.
    35	
    36	The Go model stores canonical tags as bare names such as `demo`. Zellij session names are derived at the boundary as `pair-<tag>`. `LaunchDecision` should carry both `Tag` and derived `SessionName` when the action needs a zellij session so printouts and comparisons cannot mix the two forms.

exec
/bin/zsh -lc "nl -ba workshop/plans/000075-go-launcher-prototype-plan.md | sed -n '48,72p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
    48	- **Future extensions:** Real fzf selection can become another input shape instead of branching inside IO code.
    49	
    50	The launcher package must not define a second stdout/stderr/exit-code result type. `cmd/internal/dispatcher.Result` remains the single process-facing result abstraction (`ARCH-DRY`). Launcher functions return domain values (`LaunchDecision`, snapshots, parse errors); the dispatcher route converts those values into `dispatcher.Result`.
    51	
    52	### Integration Points
    53	
    54	| Name | Lives in | Status | Wraps |
    55	|------|----------|--------|-------|
    56	| `ZellijSource` | `cmd/internal/launcher/zellij.go` | new | `zellij list-sessions`, `zellij --session ... action list-clients` |
    57	| `HistorySource` | `cmd/internal/launcher/history.go` | new | filesystem scan of `draft-*.md`, `log-*.md`, `queue-*` under Pair data dir |
    58	| `LaunchCommand` | `cmd/internal/dispatcher/dispatcher.go` | modified | existing dispatcher route for `launch` |
    59	| `PairGoMain` | `cmd/pair-go/main.go` | modified | process stdout/stderr exit handling |
    60	
    61	**ZellijSource** — Reads zellij session rows and client counts through `exec.Command`.
    62	- **Injected into:** `launcher.Run`, which converts IO into a pure `SessionSnapshot`.
    63	- **Future extensions:** Timeout behavior can be added when the Go launcher owns real launch flow; do not port the shell `zj` timeout in this prototype unless the process fake exposes the need.
    64	
    65	**HistorySource** — Scans cwd-prefixed historical sidecars from the resolved data dir.
    66	- **Injected into:** `launcher.Run`.
    67	- **Future extensions:** Queue badges and age display can be layered onto historical rows when picker UI is implemented.
    68	
    69	**LaunchCommand** — Routes `pair-go launch` from the existing dispatcher to the launcher runner.
    70	- **Injected into:** `cmd/pair-go` via the existing `run(args, stdout, stderr)` path.
    71	- **Future extensions:** Other implemented subcommands can follow the same dispatcher pattern.
    72	- **Boundary rule:** production environment reads (`os.Getenv`, `os.Getwd`) and `exec.Command` construction live in a small launcher IO constructor used by the dispatcher route. Tests may call a dispatcher test seam with an explicit launcher runtime. The launcher core returns domain outcomes; only the dispatcher maps those outcomes to stdout/stderr/exit code.

exec
/bin/zsh -lc "nl -ba cmd/internal/dispatcher/dispatcher.go | sed -n '100,116p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
   100		return h.Tags, nil
   101	}
   102	
   103	func DispatchWithLauncherRuntime(args []string, rt LauncherRuntime) Result {
   104		launchArgs := []string(nil)
   105		if len(args) > 1 {
   106			launchArgs = args[1:]
   107		}
   108		if len(launchArgs) == 0 || launchArgs[0] == "help" || launchArgs[0] == "--help" || launchArgs[0] == "-h" {
   109			return Result{Stdout: LaunchHelp(programName), ExitCode: 0}
   110		}
   111		outcome, err := launcher.Run(launchArgs, rt.Env, rt.Sessions, rt.History)
   112		if err != nil {
   113			return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
   114		}
   115		decision := outcome.Decision
   116		return Result{

exec
/bin/zsh -lc "nl -ba cmd/internal/launcher/history.go | sed -n '11,24p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
    11	// HistorySource scans Pair draft/log sidecars under the data directory.
    12	type HistorySource struct {
    13		DataDir string
    14	}
    15	
    16	func (s HistorySource) Scan(base string, cutoff time.Time) ([]HistoricalTag, error) {
    17		latest := map[string]time.Time{}
    18		for _, pattern := range []string{"draft-*.md", "log-*.md"} {
    19			matches, err := filepath.Glob(filepath.Join(s.DataDir, pattern))
    20			if err != nil {
    21				return nil, err
    22			}
    23			for _, path := range matches {
    24				tag, ok := tagFromSidecar(filepath.Base(path))

exec
/bin/zsh -lc "nl -ba cmd/internal/launcher/args.go | sed -n '25,31p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
    25	func ParseArgs(argv []string) (LaunchArgs, error) {
    26		var out LaunchArgs
    27		if len(argv) == 0 {
    28			out.Agent = "claude"
    29			return out, nil
    30		}
    31	

exec
/bin/zsh -lc 'git diff --check e89e2cea484ab8ca7ec9ca518fd2bf04ba38a8fe..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
exec
/bin/zsh -lc 'go run ./cmd/pair-go launch' in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc 'go run ./cmd/pair-go launch --help' in /Users/xianxu/workspace/pair
 succeeded in 362ms:
Usage: pair-go launch [agent] [-- agent-args...]
       pair-go launch resume <tag>

Guarded decision-phase prototype. Public sessions still start through bin/pair.
This command parses launch inputs and computes the create/attach/picker decision,
then stops before invoking zellij.

 succeeded in 483ms:
Usage: pair-go launch [agent] [-- agent-args...]
       pair-go launch resume <tag>

Guarded decision-phase prototype. Public sessions still start through bin/pair.
This command parses launch inputs and computes the create/attach/picker decision,
then stops before invoking zellij.

exec
/bin/zsh -lc "nl -ba cmd/internal/dispatcher/dispatcher.go | sed -n '160,172p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
   160	// Help renders the development-only dispatcher usage text.
   161	func Help(program string) string {
   162		var b strings.Builder
   163		fmt.Fprintf(&b, "Usage: %s <command> [args]\n\n", program)
   164		b.WriteString("Development dispatcher skeleton. Public sessions still start through bin/pair.\n\n")
   165		b.WriteString("Planned command families (not implemented in this skeleton):\n")
   166		for _, family := range Families() {
   167			fmt.Fprintf(&b, "  %-17s %s (%s; not implemented in this skeleton)\n", family.Name, family.Summary, family.Status)
   168		}
   169		b.WriteString("\nSupported skeleton commands:\n")
   170		b.WriteString("  help              show this help\n")
   171		b.WriteString("  version           show dispatcher skeleton metadata\n")
   172		return b.String()

codex
VERDICT: REWORK (confidence: high)

The implementation has a solid pure launcher core and fakeable IO seams, and the public `bin/pair` path is untouched. The blocker is that the actual guarded command path does not exercise the default-agent launch slice promised by the issue: `pair-go launch` returns help instead of reaching the decision boundary.

1. Strengths:
- `cmd/internal/launcher/decision.go:21` keeps create/attach/pick logic pure and unit-testable (`ARCH-PURE` pass).
- `cmd/internal/dispatcher/dispatcher.go:111` keeps process result mapping in the dispatcher, avoiding a second stdout/stderr/exit abstraction (`ARCH-DRY` pass).
- `cmd/pair-go/launch_process_test.go:22` uses a fake `zellij` process and asserts no mutating invocation, which is the right integration shape.
- Atlas updates describe the prototype boundary and remaining shell-owned behavior.

2. Critical findings:
- `cmd/internal/dispatcher/dispatcher.go:108`: `pair-go launch` with no launch args returns help. The issue explicitly requires argv parsing “including default agent” and a guarded path that reaches the decision boundary (`workshop/issues/000075-go-launcher-prototype.md:27`). `ParseArgs(nil)` supports `claude` (`cmd/internal/launcher/args.go:27`), but the dispatcher bypasses it. Fix: only treat explicit `help|--help|-h` as help, let empty `launchArgs` call `launcher.Run`, and add dispatcher/process tests for `pair-go launch` returning a prototype decision. `ARCH-PURPOSE` flag.
- `workshop/plans/000075-go-launcher-prototype-plan.md:57`: the integration table says `HistorySource` wraps `queue-*`, but implementation scans only `draft-*.md` and `log-*.md` (`cmd/internal/launcher/history.go:18`). Either implement the queued-surface scan or revise the plan with a `## Revisions` entry clarifying queue badges are future scope. As written, the core concepts table contradicts the code.

3. Important findings:
- `cmd/internal/dispatcher/dispatcher.go:165`: top-level help still says all planned command families, including `launch`, are “not implemented in this skeleton.” The plan expected `pair-go help` to list launch as the guarded prototype. Fix the help rendering to distinguish implemented prototype commands from still-planned families, and test for that wording.

4. Minor findings:
- None.

5. Test coverage notes:
- Ran `go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1`: pass.
- Ran `go test ./... -count=1`: pass.
- Ran `go run ./cmd/pair-go launch`: currently prints help with exit 0, confirming the default-agent route bug.
- Missing test: actual `pair-go launch` no-arg route reaches decision phase.

6. Architectural notes for upcoming work:
- `ARCH-DRY`: pass; dispatcher result reuse is clean.
- `ARCH-PURE`: pass; core logic is separated from zellij/filesystem seams.
- `ARCH-PURPOSE`: flag; the no-arg default launch path under-delivers the committed vertical slice.

7. Plan revision recommendations:
- Add `## Revisions`: “HistorySource queue scanning was deferred; current #75 implementation scans draft/log sidecars only, while queue badges remain future picker UI work.” Or implement `queue-*` support and keep the table unchanged.
tokens used
116,852
VERDICT: REWORK (confidence: high)

The implementation has a solid pure launcher core and fakeable IO seams, and the public `bin/pair` path is untouched. The blocker is that the actual guarded command path does not exercise the default-agent launch slice promised by the issue: `pair-go launch` returns help instead of reaching the decision boundary.

1. Strengths:
- `cmd/internal/launcher/decision.go:21` keeps create/attach/pick logic pure and unit-testable (`ARCH-PURE` pass).
- `cmd/internal/dispatcher/dispatcher.go:111` keeps process result mapping in the dispatcher, avoiding a second stdout/stderr/exit abstraction (`ARCH-DRY` pass).
- `cmd/pair-go/launch_process_test.go:22` uses a fake `zellij` process and asserts no mutating invocation, which is the right integration shape.
- Atlas updates describe the prototype boundary and remaining shell-owned behavior.

2. Critical findings:
- `cmd/internal/dispatcher/dispatcher.go:108`: `pair-go launch` with no launch args returns help. The issue explicitly requires argv parsing “including default agent” and a guarded path that reaches the decision boundary (`workshop/issues/000075-go-launcher-prototype.md:27`). `ParseArgs(nil)` supports `claude` (`cmd/internal/launcher/args.go:27`), but the dispatcher bypasses it. Fix: only treat explicit `help|--help|-h` as help, let empty `launchArgs` call `launcher.Run`, and add dispatcher/process tests for `pair-go launch` returning a prototype decision. `ARCH-PURPOSE` flag.
- `workshop/plans/000075-go-launcher-prototype-plan.md:57`: the integration table says `HistorySource` wraps `queue-*`, but implementation scans only `draft-*.md` and `log-*.md` (`cmd/internal/launcher/history.go:18`). Either implement the queued-surface scan or revise the plan with a `## Revisions` entry clarifying queue badges are future scope. As written, the core concepts table contradicts the code.

3. Important findings:
- `cmd/internal/dispatcher/dispatcher.go:165`: top-level help still says all planned command families, including `launch`, are “not implemented in this skeleton.” The plan expected `pair-go help` to list launch as the guarded prototype. Fix the help rendering to distinguish implemented prototype commands from still-planned families, and test for that wording.

4. Minor findings:
- None.

5. Test coverage notes:
- Ran `go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1`: pass.
- Ran `go test ./... -count=1`: pass.
- Ran `go run ./cmd/pair-go launch`: currently prints help with exit 0, confirming the default-agent route bug.
- Missing test: actual `pair-go launch` no-arg route reaches decision phase.

6. Architectural notes for upcoming work:
- `ARCH-DRY`: pass; dispatcher result reuse is clean.
- `ARCH-PURE`: pass; core logic is separated from zellij/filesystem seams.
- `ARCH-PURPOSE`: flag; the no-arg default launch path under-delivers the committed vertical slice.

7. Plan revision recommendations:
- Add `## Revisions`: “HistorySource queue scanning was deferred; current #75 implementation scans draft/log sidecars only, while queue badges remain future picker UI work.” Or implement `queue-*` support and keep the table unchanged.

---

## Re-review — 2026-06-29T22:52:16-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 75 — pair Go launcher prototype |
| repo | pair |
| issue file | workshop/issues/000075-go-launcher-prototype.md |
| boundary | whole-issue close |
| milestone | — |
| window | e89e2cea484ab8ca7ec9ca518fd2bf04ba38a8fe..HEAD |
| command | sdlc close --issue 75 |
| reviewer | codex |
| timestamp | 2026-06-29T22:52:16-07:00 |
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
session id: 019f1714-41ec-7ce3-9a3e-2ff920305c5a
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **pair** repository.

- repository: pair   (root: /Users/xianxu/workspace/pair)
- issue:      pair#75   (file: workshop/issues/000075-go-launcher-prototype.md)
- window:     Base: e89e2cea484ab8ca7ec9ca518fd2bf04ba38a8fe   Head: HEAD

Review the **pair** repo and its tracker — the ariadne base-layer repo itself (changes here propagate to dependent repos). Do not assume any
other repository or apply another repo's conventions.

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
diff --git a/atlas/architecture.md b/atlas/architecture.md
index f9d58a3..a536fc7 100644
--- a/atlas/architecture.md
+++ b/atlas/architecture.md
@@ -45,10 +45,15 @@ internal subcommands or dispatch modes behind that primary binary (`pair wrap`,
 `pair continuation`, `pair scribe`) instead of staying as independently managed
 installed commands forever.
 
-As of #74, `bin/pair-go` is a development-only dispatcher skeleton built from
-`cmd/pair-go` and the pure `cmd/internal/dispatcher` parser. It lists the
-planned command families and returns explicit unsupported-command errors, but it
-does not launch sessions or replace `bin/pair`.
+As of #75, `bin/pair-go` is a development-only dispatcher with a guarded
+`pair-go launch` decision-phase prototype. The public launcher remains
+`bin/pair`; `pair-go launch` parses the launcher subset, resolves the Pair data
+dir, snapshots fakeable zellij/history state, and computes create/attach/picker
+decisions through the pure `cmd/internal/launcher` core. It then stops with an
+explicit prototype message before invoking zellij. Real zellij lifecycle,
+prompt/fzf UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation,
+rename, config/session migration, and title-poller behavior remain shell-owned
+until later migration issues.
 
 Native integration layers stay native: `nvim/*.lua` remains the bundled Neovim
 surface and `zellij/*.kdl` remains the zellij layout/config surface. Packaging
diff --git a/atlas/go-migration-inventory.md b/atlas/go-migration-inventory.md
index f671fbf..3de3bb2 100644
--- a/atlas/go-migration-inventory.md
+++ b/atlas/go-migration-inventory.md
@@ -47,7 +47,7 @@ Priority is packaging impact first, then reliability/testability:
 
 | Artifact | Type | Callers | Runtime contract | Files/env | Disposition | Priority |
 |---|---|---|---|---|---|---|
-| `bin/pair` | Bash public launcher | user shell, `bin/pair-dev`, restart re-exec, tests | Parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. | Exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. | go-entrypoint with `bin/pair` retained as compat fallback through #77 | P0 |
+| `bin/pair` / `cmd/internal/launcher` | Bash public launcher plus guarded Go decision prototype | user shell, `bin/pair-dev`, restart re-exec, tests, development-only `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch` (#75) parses a documented subset, snapshots zellij/history state, prints the decision-phase prototype action, and exits before mutating zellij. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/launcher` resolves data-dir inputs and reads zellij/history through fakeable seams. | go-entrypoint with `bin/pair` retained as compat fallback through #77; #75 owns decision-phase only, while real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
 | `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling `pair`. | Resolves symlinks; depends on `bin/pair` and `bin/lib/dev-rebuild.sh`. | compat-shim, likely stays as developer wrapper until Go entrypoint has dev mode | P1 |
 | `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
 | `zellij/layouts/main.kdl` | zellij native asset | `bin/pair` via `zellij --new-session-with-layout` | Defines agent and draft panes; shell expands Pair env at pane start. | Calls `pair-wrap`; calls `nvim -u "$PAIR_HOME/nvim/init.lua"`; writes `pane-<tag>-<agent>.json`; writes draft nvim pid file. | native-asset, packaged adjacent/embedded | P0 |
diff --git a/cmd/internal/dispatcher/dispatcher.go b/cmd/internal/dispatcher/dispatcher.go
index dddfe46..1b9710a 100644
--- a/cmd/internal/dispatcher/dispatcher.go
+++ b/cmd/internal/dispatcher/dispatcher.go
@@ -2,7 +2,11 @@ package dispatcher
 
 import (
 	"fmt"
+	"os"
 	"strings"
+	"time"
+
+	"github.com/xianxu/pair/cmd/internal/launcher"
 )
 
 const programName = "pair-go"
@@ -49,6 +53,8 @@ func Dispatch(args []string) Result {
 			Stdout:   "pair-go dispatcher skeleton\npublic launcher: bin/pair\n",
 			ExitCode: 0,
 		}
+	case "launch":
+		return DispatchWithLauncherRuntime(args, osLauncherRuntime())
 	}
 
 	if family, ok := familyByName(args[0]); ok {
@@ -64,6 +70,93 @@ func Dispatch(args []string) Result {
 	}
 }
 
+type LauncherRuntime struct {
+	Env      launcher.Env
+	Sessions launcher.SessionSource
+	History  launcher.HistoricalScanner
+}
+
+type StaticSessions struct {
+	Sessions []launcher.Session
+	Err      error
+}
+
+func (s StaticSessions) Snapshot() ([]launcher.Session, error) {
+	if s.Err != nil {
+		return nil, s.Err
+	}
+	return s.Sessions, nil
+}
+
+type StaticHistory struct {
+	Tags []launcher.HistoricalTag
+	Err  error
+}
+
+func (h StaticHistory) Scan(_ string, _ time.Time) ([]launcher.HistoricalTag, error) {
+	if h.Err != nil {
+		return nil, h.Err
+	}
+	return h.Tags, nil
+}
+
+func DispatchWithLauncherRuntime(args []string, rt LauncherRuntime) Result {
+	launchArgs := []string(nil)
+	if len(args) > 1 {
+		launchArgs = args[1:]
+	}
+	if len(launchArgs) == 0 || launchArgs[0] == "help" || launchArgs[0] == "--help" || launchArgs[0] == "-h" {
+		return Result{Stdout: LaunchHelp(programName), ExitCode: 0}
+	}
+	outcome, err := launcher.Run(launchArgs, rt.Env, rt.Sessions, rt.History)
+	if err != nil {
+		return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
+	}
+	decision := outcome.Decision
+	return Result{
+		Stderr: fmt.Sprintf(
+			"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
+			decision.Action,
+			decision.Tag,
+			decision.SessionName,
+		),
+		ExitCode: 3,
+	}
+}
+
+func LaunchHelp(program string) string {
+	return fmt.Sprintf(`Usage: %s launch [agent] [-- agent-args...]
+       %s launch resume <tag>
+
+Guarded decision-phase prototype. Public sessions still start through bin/pair.
+This command parses launch inputs and computes the create/attach/picker decision,
+then stops before invoking zellij.
+`, program, program)
+}
+
+func LauncherEnv(home, xdgDataHome, cwd string) launcher.Env {
+	return launcher.Env{
+		Home:     home,
+		XDGData:  xdgDataHome,
+		Cwd:      cwd,
+		Now:      time.Now(),
+		HistoryD: 14,
+	}
+}
+
+func osLauncherRuntime() LauncherRuntime {
+	home := os.Getenv("HOME")
+	xdg := os.Getenv("XDG_DATA_HOME")
+	cwd, _ := os.Getwd()
+	env := LauncherEnv(home, xdg, cwd)
+	dataDir := launcher.ResolveDataDir(home, xdg)
+	return LauncherRuntime{
+		Env:      env,
+		Sessions: launcher.ZellijSource{},
+		History:  launcher.HistorySource{DataDir: dataDir},
+	}
+}
+
 // Help renders the development-only dispatcher usage text.
 func Help(program string) string {
 	var b strings.Builder
diff --git a/cmd/internal/dispatcher/dispatcher_test.go b/cmd/internal/dispatcher/dispatcher_test.go
index c9fed26..1d1358a 100644
--- a/cmd/internal/dispatcher/dispatcher_test.go
+++ b/cmd/internal/dispatcher/dispatcher_test.go
@@ -60,6 +60,42 @@ func TestDispatchPlannedCommandReturnsUnsupported(t *testing.T) {
 	}
 }
 
+func TestDispatchLaunchHelpRoutesToPrototype(t *testing.T) {
+	res := Dispatch([]string{"launch", "--help"})
+	if res.ExitCode != 0 {
+		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
+	}
+	if res.Stderr != "" {
+		t.Fatalf("Stderr = %q, want empty", res.Stderr)
+	}
+	for _, want := range []string{"Usage: pair-go launch", "decision-phase prototype"} {
+		if !strings.Contains(res.Stdout, want) {
+			t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
+		}
+	}
+}
+
+func TestDispatchLaunchReturnsPrototypeDecision(t *testing.T) {
+	res := DispatchWithLauncherRuntime([]string{"launch", "resume", "demo"}, LauncherRuntime{
+		Env: LauncherEnv("/home/me", "", "/work/pair"),
+		Sessions: StaticSessions{
+			Sessions: nil,
+		},
+		History: StaticHistory{},
+	})
+	if res.ExitCode != 3 {
+		t.Fatalf("ExitCode = %d, want 3", res.ExitCode)
+	}
+	if res.Stdout != "" {
+		t.Fatalf("Stdout = %q, want empty", res.Stdout)
+	}
+	for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
+		if !strings.Contains(res.Stderr, want) {
+			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
+		}
+	}
+}
+
 func TestDispatchUnknownCommandReturnsUsageHint(t *testing.T) {
 	res := Dispatch([]string{"frobnicate"})
 	if res.ExitCode != 2 {
diff --git a/cmd/internal/launcher/args.go b/cmd/internal/launcher/args.go
new file mode 100644
index 0000000..06aa8fb
--- /dev/null
+++ b/cmd/internal/launcher/args.go
@@ -0,0 +1,71 @@
+package launcher
+
+import "fmt"
+
+// LaunchArgs is the pure parse result for the guarded pair-go launch prototype.
+type LaunchArgs struct {
+	Agent       string
+	ForcedTag   string
+	SelectedTag string
+	AgentArgs   []string
+}
+
+// UsageError is an operator-facing parse error.
+type UsageError struct {
+	Message string
+}
+
+func (e UsageError) Error() string {
+	return e.Message
+}
+
+// ParseArgs parses pair-go launch args. It intentionally supports only the
+// decision-phase subset for #75; unsupported shell-owned launcher verbs fail
+// explicitly.
+func ParseArgs(argv []string) (LaunchArgs, error) {
+	var out LaunchArgs
+	if len(argv) == 0 {
+		out.Agent = "claude"
+		return out, nil
+	}
+
+	switch argv[0] {
+	case "continue", "rename", "list", "ls":
+		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %s is not implemented by pair-go launch; use bin/pair", argv[0])}
+	case "resume":
+		if len(argv) < 2 {
+			return LaunchArgs{}, UsageError{Message: "pair-go launch: 'resume' requires a tag"}
+		}
+		tag, err := NormalizeTag(argv[1])
+		if err != nil {
+			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: invalid tag: %v", err)}
+		}
+		if len(argv) > 2 {
+			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", argv[2])}
+		}
+		out.ForcedTag = tag
+		return out, nil
+	}
+
+	seenSeparator := false
+	for _, arg := range argv {
+		if seenSeparator {
+			out.AgentArgs = append(out.AgentArgs, arg)
+			continue
+		}
+		if arg == "--" {
+			seenSeparator = true
+			continue
+		}
+		if out.Agent == "" {
+			out.Agent = arg
+			continue
+		}
+		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", arg)}
+	}
+
+	if out.Agent == "" {
+		out.Agent = "claude"
+	}
+	return out, nil
+}
diff --git a/cmd/internal/launcher/args_test.go b/cmd/internal/launcher/args_test.go
new file mode 100644
index 0000000..85bc25e
--- /dev/null
+++ b/cmd/internal/launcher/args_test.go
@@ -0,0 +1,88 @@
+package launcher
+
+import (
+	"strings"
+	"testing"
+)
+
+func TestParseLaunchArgsDefaultsToClaude(t *testing.T) {
+	args, err := ParseArgs(nil)
+	if err != nil {
+		t.Fatalf("ParseArgs returned error: %v", err)
+	}
+	if args.Agent != "claude" {
+		t.Fatalf("Agent = %q, want claude", args.Agent)
+	}
+	if args.ForcedTag != "" {
+		t.Fatalf("ForcedTag = %q, want empty", args.ForcedTag)
+	}
+	if len(args.AgentArgs) != 0 {
+		t.Fatalf("AgentArgs = %#v, want empty", args.AgentArgs)
+	}
+}
+
+func TestParseLaunchArgsAgentAndForwardedArgs(t *testing.T) {
+	args, err := ParseArgs([]string{"codex", "--", "-p", "say hi"})
+	if err != nil {
+		t.Fatalf("ParseArgs returned error: %v", err)
+	}
+	if args.Agent != "codex" {
+		t.Fatalf("Agent = %q, want codex", args.Agent)
+	}
+	if got := strings.Join(args.AgentArgs, " "); got != "-p say hi" {
+		t.Fatalf("AgentArgs = %q, want forwarded args", got)
+	}
+}
+
+func TestParseLaunchArgsDefaultAgentWithForwardedArgs(t *testing.T) {
+	args, err := ParseArgs([]string{"--", "--dangerously-skip-permissions"})
+	if err != nil {
+		t.Fatalf("ParseArgs returned error: %v", err)
+	}
+	if args.Agent != "claude" {
+		t.Fatalf("Agent = %q, want claude", args.Agent)
+	}
+	if got := strings.Join(args.AgentArgs, " "); got != "--dangerously-skip-permissions" {
+		t.Fatalf("AgentArgs = %q, want forwarded args", got)
+	}
+}
+
+func TestParseLaunchArgsResumeNormalizesForcedTag(t *testing.T) {
+	args, err := ParseArgs([]string{"resume", "pair-demo"})
+	if err != nil {
+		t.Fatalf("ParseArgs returned error: %v", err)
+	}
+	if args.Agent != "" {
+		t.Fatalf("Agent = %q, want empty for resume inference", args.Agent)
+	}
+	if args.ForcedTag != "demo" {
+		t.Fatalf("ForcedTag = %q, want demo", args.ForcedTag)
+	}
+}
+
+func TestParseLaunchArgsUnexpectedPositionalGuidesAgentArgs(t *testing.T) {
+	_, err := ParseArgs([]string{"codex", "extra"})
+	if err == nil {
+		t.Fatal("ParseArgs returned nil error")
+	}
+	msg := err.Error()
+	for _, want := range []string{"unexpected positional arg", "use '--' to forward args to the agent"} {
+		if !strings.Contains(msg, want) {
+			t.Fatalf("error missing %q: %s", want, msg)
+		}
+	}
+}
+
+func TestParseLaunchArgsUnsupportedLaunchSubcommandsAreExplicit(t *testing.T) {
+	for _, verb := range []string{"continue", "rename", "list"} {
+		t.Run(verb, func(t *testing.T) {
+			_, err := ParseArgs([]string{verb})
+			if err == nil {
+				t.Fatal("ParseArgs returned nil error")
+			}
+			if !strings.Contains(err.Error(), "not implemented by pair-go launch") {
+				t.Fatalf("error = %q, want explicit unsupported message", err)
+			}
+		})
+	}
+}
diff --git a/cmd/internal/launcher/datadir.go b/cmd/internal/launcher/datadir.go
new file mode 100644
index 0000000..7a57144
--- /dev/null
+++ b/cmd/internal/launcher/datadir.go
@@ -0,0 +1,11 @@
+package launcher
+
+import "path/filepath"
+
+// ResolveDataDir returns Pair's data directory from explicit environment values.
+func ResolveDataDir(home, xdgDataHome string) string {
+	if xdgDataHome != "" {
+		return filepath.Join(xdgDataHome, "pair")
+	}
+	return filepath.Join(home, ".local", "share", "pair")
+}
diff --git a/cmd/internal/launcher/datadir_test.go b/cmd/internal/launcher/datadir_test.go
new file mode 100644
index 0000000..bb604b6
--- /dev/null
+++ b/cmd/internal/launcher/datadir_test.go
@@ -0,0 +1,21 @@
+package launcher
+
+import "testing"
+
+func TestResolveDataDir(t *testing.T) {
+	for _, tc := range []struct {
+		name string
+		home string
+		xdg  string
+		want string
+	}{
+		{name: "xdg", home: "/home/me", xdg: "/tmp/xdg", want: "/tmp/xdg/pair"},
+		{name: "home", home: "/home/me", want: "/home/me/.local/share/pair"},
+	} {
+		t.Run(tc.name, func(t *testing.T) {
+			if got := ResolveDataDir(tc.home, tc.xdg); got != tc.want {
+				t.Fatalf("ResolveDataDir(%q, %q) = %q, want %q", tc.home, tc.xdg, got, tc.want)
+			}
+		})
+	}
+}
diff --git a/cmd/internal/launcher/decision.go b/cmd/internal/launcher/decision.go
new file mode 100644
index 0000000..de49d33
--- /dev/null
+++ b/cmd/internal/launcher/decision.go
@@ -0,0 +1,103 @@
+package launcher
+
+// LaunchAction is the guarded prototype's next launcher action.
+type LaunchAction string
+
+const (
+	ActionAttach LaunchAction = "attach"
+	ActionCreate LaunchAction = "create"
+	ActionPick   LaunchAction = "pick"
+)
+
+// LaunchDecision is a pure create/attach/pick decision. Tag is canonical bare
+// form; SessionName is derived as pair-<tag> when a zellij session is named.
+type LaunchDecision struct {
+	Action      LaunchAction
+	Tag         string
+	SessionName string
+	PromptName  bool
+}
+
+// DecideLaunch decides the launch action without touching zellij, fzf, or disk.
+func DecideLaunch(args LaunchArgs, snap SessionSnapshot) (LaunchDecision, error) {
+	if args.SelectedTag != "" {
+		return createDecision(args.SelectedTag, false), nil
+	}
+	if args.ForcedTag != "" {
+		if sessionBlocksReuse(snap, sessionName(args.ForcedTag)) {
+			return LaunchDecision{Action: ActionAttach, Tag: args.ForcedTag, SessionName: sessionName(args.ForcedTag)}, nil
+		}
+		return createDecision(args.ForcedTag, false), nil
+	}
+	if hasDetached(snap) || len(snap.Historical) > 0 {
+		return LaunchDecision{Action: ActionPick}, nil
+	}
+	tag := snap.BaseTag
+	if tag == "" {
+		tag = "pair"
+	}
+	return createDecision(nextFreeTag(tag, snap), true), nil
+}
+
+func createDecision(tag string, prompt bool) LaunchDecision {
+	return LaunchDecision{Action: ActionCreate, Tag: tag, SessionName: sessionName(tag), PromptName: prompt}
+}
+
+func sessionName(tag string) string {
+	return "pair-" + tag
+}
+
+func hasDetached(snap SessionSnapshot) bool {
+	for _, sess := range snap.Sessions {
+		if sess.State == SessionDetached {
+			return true
+		}
+	}
+	return false
+}
+
+func sessionBlocksReuse(snap SessionSnapshot, name string) bool {
+	for _, sess := range snap.Sessions {
+		if sess.Name != name {
+			continue
+		}
+		return sess.State == SessionAttached || sess.State == SessionDetached
+	}
+	return false
+}
+
+func nextFreeTag(base string, snap SessionSnapshot) string {
+	for i := 1; i <= 100; i++ {
+		tag := base
+		if i > 1 {
+			tag = base + "-" + itoa(i)
+		}
+		if !sessionBlocksReuse(snap, sessionName(tag)) && !isHistorical(snap, tag) {
+			return tag
+		}
+	}
+	return base
+}
+
+func isHistorical(snap SessionSnapshot, tag string) bool {
+	for _, hist := range snap.Historical {
+		if hist.Tag == tag {
+			return true
+		}
+	}
+	return false
+}
+
+func itoa(n int) string {
+	if n == 0 {
+		return "0"
+	}
+	var buf [20]byte
+	i := len(buf)
+	for n > 0 {
+		i--
+		buf[i] = byte('0' + n%10)
+		n /= 10
+	}
+	return string(buf[i:])
+}
diff --git a/cmd/internal/launcher/decision_test.go b/cmd/internal/launcher/decision_test.go
new file mode 100644
index 0000000..8c39d21
--- /dev/null
+++ b/cmd/internal/launcher/decision_test.go
@@ -0,0 +1,67 @@
+package launcher
+
+import "testing"
+
+func TestDecideLaunchForcedResumeAttachesWhenSessionBlocksReuse(t *testing.T) {
+	decision, err := DecideLaunch(LaunchArgs{ForcedTag: "demo"}, SessionSnapshot{
+		Sessions: []Session{{Name: "pair-demo", State: SessionDetached}},
+	})
+	if err != nil {
+		t.Fatalf("DecideLaunch returned error: %v", err)
+	}
+	if decision.Action != ActionAttach || decision.Tag != "demo" || decision.SessionName != "pair-demo" {
+		t.Fatalf("decision = %#v, want attach demo/pair-demo", decision)
+	}
+}
+
+func TestDecideLaunchForcedResumeCreatesWhenSessionDoesNotBlockReuse(t *testing.T) {
+	decision, err := DecideLaunch(LaunchArgs{ForcedTag: "demo"}, SessionSnapshot{
+		Sessions: []Session{{Name: "pair-demo", State: SessionExited}},
+	})
+	if err != nil {
+		t.Fatalf("DecideLaunch returned error: %v", err)
+	}
+	if decision.Action != ActionCreate || decision.Tag != "demo" || decision.SessionName != "pair-demo" || decision.PromptName {
+		t.Fatalf("decision = %#v, want create demo/pair-demo without prompt", decision)
+	}
+}
+
+func TestDecideLaunchEmptyStateCreatesNextFreeTagWithPrompt(t *testing.T) {
+	decision, err := DecideLaunch(LaunchArgs{Agent: "codex"}, SessionSnapshot{
+		BaseTag: "pair",
+	})
+	if err != nil {
+		t.Fatalf("DecideLaunch returned error: %v", err)
+	}
+	if decision.Action != ActionCreate || decision.Tag != "pair" || decision.SessionName != "pair-pair" || !decision.PromptName {
+		t.Fatalf("decision = %#v, want create next free tag with prompt", decision)
+	}
+}
+
+func TestDecideLaunchShowsPickerWhenDetachedOrHistoricalExist(t *testing.T) {
+	for _, snap := range []SessionSnapshot{
+		{BaseTag: "pair", Sessions: []Session{{Name: "pair-other", State: SessionDetached}}},
+		{BaseTag: "pair", Historical: []HistoricalTag{{Tag: "pair-old"}}},
+	} {
+		decision, err := DecideLaunch(LaunchArgs{Agent: "claude"}, snap)
+		if err != nil {
+			t.Fatalf("DecideLaunch returned error: %v", err)
+		}
+		if decision.Action != ActionPick {
+			t.Fatalf("decision = %#v, want picker", decision)
+		}
+	}
+}
+
+func TestDecideLaunchHistoricalSelectionCreatesByTag(t *testing.T) {
+	decision, err := DecideLaunch(LaunchArgs{Agent: "claude", SelectedTag: "pair-old"}, SessionSnapshot{
+		BaseTag:    "pair",
+		Historical: []HistoricalTag{{Tag: "pair-old"}},
+	})
+	if err != nil {
+		t.Fatalf("DecideLaunch returned error: %v", err)
+	}
+	if decision.Action != ActionCreate || decision.Tag != "pair-old" || decision.SessionName != "pair-pair-old" || decision.PromptName {
+		t.Fatalf("decision = %#v, want create historical tag without prompt", decision)
+	}
+}
diff --git a/cmd/internal/launcher/history.go b/cmd/internal/launcher/history.go
new file mode 100644
index 0000000..0e9ef52
--- /dev/null
+++ b/cmd/internal/launcher/history.go
@@ -0,0 +1,65 @@
+package launcher
+
+import (
+	"os"
+	"path/filepath"
+	"sort"
+	"strings"
+	"time"
+)
+
+// HistorySource scans Pair draft/log sidecars under the data directory.
+type HistorySource struct {
+	DataDir string
+}
+
+func (s HistorySource) Scan(base string, cutoff time.Time) ([]HistoricalTag, error) {
+	latest := map[string]time.Time{}
+	for _, pattern := range []string{"draft-*.md", "log-*.md"} {
+		matches, err := filepath.Glob(filepath.Join(s.DataDir, pattern))
+		if err != nil {
+			return nil, err
+		}
+		for _, path := range matches {
+			tag, ok := tagFromSidecar(filepath.Base(path))
+			if !ok || !matchesHistoryBase(tag, base) {
+				continue
+			}
+			info, err := os.Stat(path)
+			if err != nil {
+				continue
+			}
+			if info.ModTime().Before(cutoff) {
+				continue
+			}
+			if info.ModTime().After(latest[tag]) {
+				latest[tag] = info.ModTime()
+			}
+		}
+	}
+	tags := make([]string, 0, len(latest))
+	for tag := range latest {
+		tags = append(tags, tag)
+	}
+	sort.Strings(tags)
+	out := make([]HistoricalTag, 0, len(tags))
+	for _, tag := range tags {
+		out = append(out, HistoricalTag{Tag: tag})
+	}
+	return out, nil
+}
+
+func tagFromSidecar(name string) (string, bool) {
+	switch {
+	case strings.HasPrefix(name, "draft-") && strings.HasSuffix(name, ".md"):
+		return strings.TrimSuffix(strings.TrimPrefix(name, "draft-"), ".md"), true
+	case strings.HasPrefix(name, "log-") && strings.HasSuffix(name, ".md"):
+		return strings.TrimSuffix(strings.TrimPrefix(name, "log-"), ".md"), true
+	default:
+		return "", false
+	}
+}
+
+func matchesHistoryBase(tag, base string) bool {
+	return tag == base || strings.HasPrefix(tag, base+"-")
+}
diff --git a/cmd/internal/launcher/history_test.go b/cmd/internal/launcher/history_test.go
new file mode 100644
index 0000000..5f158fd
--- /dev/null
+++ b/cmd/internal/launcher/history_test.go
@@ -0,0 +1,33 @@
+package launcher
+
+import (
+	"os"
+	"path/filepath"
+	"testing"
+	"time"
+)
+
+func TestHistorySourceScansCwdPrefixedDraftAndLogSidecars(t *testing.T) {
+	dir := t.TempDir()
+	now := time.Unix(2000, 0)
+	for _, name := range []string{"draft-pair.md", "log-pair-old.md", "draft-other.md"} {
+		path := filepath.Join(dir, name)
+		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
+			t.Fatal(err)
+		}
+		if err := os.Chtimes(path, now, now); err != nil {
+			t.Fatal(err)
+		}
+	}
+
+	got, err := HistorySource{DataDir: dir}.Scan("pair", now.Add(-24*time.Hour))
+	if err != nil {
+		t.Fatalf("Scan returned error: %v", err)
+	}
+	if len(got) != 2 {
+		t.Fatalf("Scan returned %#v, want 2 pair-prefixed tags", got)
+	}
+	if got[0].Tag != "pair" || got[1].Tag != "pair-old" {
+		t.Fatalf("Scan returned %#v, want sorted pair tags", got)
+	}
+}
diff --git a/cmd/internal/launcher/run.go b/cmd/internal/launcher/run.go
new file mode 100644
index 0000000..c3d9b2c
--- /dev/null
+++ b/cmd/internal/launcher/run.go
@@ -0,0 +1,70 @@
+package launcher
+
+import (
+	"time"
+)
+
+// Env is the explicit runtime environment for launch decisions.
+type Env struct {
+	Home     string
+	XDGData  string
+	Cwd      string
+	Now      time.Time
+	HistoryD int
+	DataDir  string
+}
+
+// SessionSource supplies zellij session state.
+type SessionSource interface {
+	Snapshot() ([]Session, error)
+}
+
+// HistoricalScanner supplies recently touched Pair tags.
+type HistoricalScanner interface {
+	Scan(base string, cutoff time.Time) ([]HistoricalTag, error)
+}
+
+// LaunchOutcome is the domain outcome from the launcher core.
+type LaunchOutcome struct {
+	Args     LaunchArgs
+	Env      Env
+	Decision LaunchDecision
+}
+
+// Run builds a pure snapshot from injected sources and returns a domain launch
+// outcome. The dispatcher maps this to process stdout/stderr/exit status.
+func Run(argv []string, env Env, sessions SessionSource, history HistoricalScanner) (LaunchOutcome, error) {
+	args, err := ParseArgs(argv)
+	if err != nil {
+		return LaunchOutcome{}, err
+	}
+	if env.DataDir == "" {
+		env.DataDir = ResolveDataDir(env.Home, env.XDGData)
+	}
+	if env.HistoryD == 0 {
+		env.HistoryD = 14
+	}
+	if env.Now.IsZero() {
+		env.Now = time.Now()
+	}
+
+	sessionRows, err := sessions.Snapshot()
+	if err != nil {
+		return LaunchOutcome{}, err
+	}
+	base := DefaultTag(env.Cwd)
+	historical, err := history.Scan(base, env.Now.Add(-time.Duration(env.HistoryD)*24*time.Hour))
+	if err != nil {
+		return LaunchOutcome{}, err
+	}
+
+	decision, err := DecideLaunch(args, SessionSnapshot{
+		BaseTag:    base,
+		Sessions:   sessionRows,
+		Historical: historical,
+	})
+	if err != nil {
+		return LaunchOutcome{}, err
+	}
+	return LaunchOutcome{Args: args, Env: env, Decision: decision}, nil
+}
diff --git a/cmd/internal/launcher/run_test.go b/cmd/internal/launcher/run_test.go
new file mode 100644
index 0000000..fdc7ced
--- /dev/null
+++ b/cmd/internal/launcher/run_test.go
@@ -0,0 +1,90 @@
+package launcher
+
+import (
+	"errors"
+	"testing"
+	"time"
+)
+
+type fakeSessions struct {
+	sessions []Session
+	err      error
+}
+
+func (f fakeSessions) Snapshot() ([]Session, error) {
+	if f.err != nil {
+		return nil, f.err
+	}
+	return f.sessions, nil
+}
+
+type fakeHistory struct {
+	tags []HistoricalTag
+	err  error
+}
+
+func (f fakeHistory) Scan(base string, cutoff time.Time) ([]HistoricalTag, error) {
+	if f.err != nil {
+		return nil, f.err
+	}
+	return f.tags, nil
+}
+
+func TestRunLaunchUsesSuppliedEnvironment(t *testing.T) {
+	outcome, err := Run([]string{"codex"}, Env{
+		Home:     "/home/me",
+		Cwd:      "/work/pair",
+		Now:      time.Unix(1000, 0),
+		HistoryD: 14,
+	}, fakeSessions{}, fakeHistory{})
+	if err != nil {
+		t.Fatalf("Run returned error: %v", err)
+	}
+	if outcome.Decision.Action != ActionCreate || outcome.Decision.Tag != "pair" || !outcome.Decision.PromptName {
+		t.Fatalf("Decision = %#v, want create pair with prompt", outcome.Decision)
+	}
+	if outcome.Env.DataDir != "/home/me/.local/share/pair" {
+		t.Fatalf("DataDir = %q, want home-derived data dir", outcome.Env.DataDir)
+	}
+}
+
+func TestRunLaunchTurnsFakeSessionsIntoPickerDecision(t *testing.T) {
+	outcome, err := Run([]string{"claude"}, Env{
+		Home:     "/home/me",
+		Cwd:      "/work/pair",
+		Now:      time.Unix(1000, 0),
+		HistoryD: 14,
+	}, fakeSessions{sessions: []Session{{Name: "pair-demo", State: SessionDetached}}}, fakeHistory{})
+	if err != nil {
+		t.Fatalf("Run returned error: %v", err)
+	}
+	if outcome.Decision.Action != ActionPick {
+		t.Fatalf("Decision = %#v, want picker", outcome.Decision)
+	}
+}
+
+func TestRunLaunchTurnsFakeHistoryIntoPickerDecision(t *testing.T) {
+	outcome, err := Run([]string{"claude"}, Env{
+		Home:     "/home/me",
+		Cwd:      "/work/pair",
+		Now:      time.Unix(1000, 0),
+		HistoryD: 14,
+	}, fakeSessions{}, fakeHistory{tags: []HistoricalTag{{Tag: "pair-old"}}})
+	if err != nil {
+		t.Fatalf("Run returned error: %v", err)
+	}
+	if outcome.Decision.Action != ActionPick {
+		t.Fatalf("Decision = %#v, want picker", outcome.Decision)
+	}
+}
+
+func TestRunLaunchReturnsTypedUsageError(t *testing.T) {
+	_, err := Run([]string{"codex", "extra"}, Env{Home: "/home/me", Cwd: "/work/pair"}, fakeSessions{}, fakeHistory{})
+	if err == nil {
+		t.Fatal("Run returned nil error")
+	}
+	var usage UsageError
+	if !errors.As(err, &usage) {
+		t.Fatalf("error = %T, want UsageError", err)
+	}
+}
diff --git a/cmd/internal/launcher/session.go b/cmd/internal/launcher/session.go
new file mode 100644
index 0000000..9dc303a
--- /dev/null
+++ b/cmd/internal/launcher/session.go
@@ -0,0 +1,28 @@
+package launcher
+
+// SessionState describes whether a zellij session blocks tag reuse.
+type SessionState string
+
+const (
+	SessionAttached SessionState = "attached"
+	SessionDetached SessionState = "detached"
+	SessionExited   SessionState = "exited"
+)
+
+// Session is a zellij session row projected into launcher decision space.
+type Session struct {
+	Name  string
+	State SessionState
+}
+
+// HistoricalTag is a recently touched Pair tag with no live zellij session.
+type HistoricalTag struct {
+	Tag string
+}
+
+// SessionSnapshot is the pure input to launcher decision-making.
+type SessionSnapshot struct {
+	BaseTag    string
+	Sessions   []Session
+	Historical []HistoricalTag
+}
diff --git a/cmd/internal/launcher/tag.go b/cmd/internal/launcher/tag.go
new file mode 100644
index 0000000..f8bb4c6
--- /dev/null
+++ b/cmd/internal/launcher/tag.go
@@ -0,0 +1,42 @@
+package launcher
+
+import (
+	"fmt"
+	"path/filepath"
+	"strings"
+)
+
+// NormalizeTag returns Pair's canonical bare tag form.
+func NormalizeTag(raw string) (string, error) {
+	tag := strings.TrimPrefix(raw, "pair-")
+	if tag == "" {
+		return "", fmt.Errorf("empty tag")
+	}
+	for _, r := range tag {
+		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
+			continue
+		}
+		return "", fmt.Errorf("tag %q contains invalid character %q", raw, r)
+	}
+	return tag, nil
+}
+
+// DefaultTag derives Pair's create-flow default tag from a cwd path.
+func DefaultTag(cwd string) string {
+	base := filepath.Base(cwd)
+	if base == "." || base == string(filepath.Separator) {
+		return "pair"
+	}
+	var b strings.Builder
+	for _, r := range base {
+		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
+			b.WriteRune(r)
+		} else {
+			b.WriteByte('_')
+		}
+	}
+	if b.Len() == 0 {
+		return "pair"
+	}
+	return b.String()
+}
diff --git a/cmd/internal/launcher/tag_test.go b/cmd/internal/launcher/tag_test.go
new file mode 100644
index 0000000..ff30987
--- /dev/null
+++ b/cmd/internal/launcher/tag_test.go
@@ -0,0 +1,53 @@
+package launcher
+
+import "testing"
+
+func TestNormalizeTag(t *testing.T) {
+	for _, tc := range []struct {
+		name string
+		raw  string
+		want string
+	}{
+		{name: "bare", raw: "demo", want: "demo"},
+		{name: "prefixed", raw: "pair-demo", want: "demo"},
+		{name: "underscore", raw: "pair-demo_2", want: "demo_2"},
+	} {
+		t.Run(tc.name, func(t *testing.T) {
+			got, err := NormalizeTag(tc.raw)
+			if err != nil {
+				t.Fatalf("NormalizeTag returned error: %v", err)
+			}
+			if got != tc.want {
+				t.Fatalf("NormalizeTag(%q) = %q, want %q", tc.raw, got, tc.want)
+			}
+		})
+	}
+}
+
+func TestNormalizeTagRejectsInvalid(t *testing.T) {
+	for _, raw := range []string{"", "pair-", "bad/slug", "has space"} {
+		t.Run(raw, func(t *testing.T) {
+			if _, err := NormalizeTag(raw); err == nil {
+				t.Fatalf("NormalizeTag(%q) returned nil error", raw)
+			}
+		})
+	}
+}
+
+func TestDefaultTag(t *testing.T) {
+	for _, tc := range []struct {
+		cwd  string
+		want string
+	}{
+		{cwd: "/Users/xianxu/workspace/pair", want: "pair"},
+		{cwd: "/tmp/hello world", want: "hello_world"},
+		{cwd: "/tmp/!!!", want: "___"},
+		{cwd: "", want: "pair"},
+	} {
+		t.Run(tc.cwd, func(t *testing.T) {
+			if got := DefaultTag(tc.cwd); got != tc.want {
+				t.Fatalf("DefaultTag(%q) = %q, want %q", tc.cwd, got, tc.want)
+			}
+		})
+	}
+}
diff --git a/cmd/internal/launcher/zellij.go b/cmd/internal/launcher/zellij.go
new file mode 100644
index 0000000..96febe8
--- /dev/null
+++ b/cmd/internal/launcher/zellij.go
@@ -0,0 +1,91 @@
+package launcher
+
+import (
+	"bytes"
+	"os/exec"
+	"sort"
+	"strings"
+)
+
+// ZellijSource reads zellij session state through the zellij CLI.
+type ZellijSource struct {
+	Path string
+}
+
+func (s ZellijSource) Snapshot() ([]Session, error) {
+	short, err := s.run("list-sessions", "--short")
+	if err != nil {
+		short = nil
+	}
+	raw, err := s.run("list-sessions", "--no-formatting")
+	if err != nil {
+		raw = nil
+	}
+	exited := exitedSessions(string(raw))
+	var out []Session
+	for _, name := range lines(string(short)) {
+		if !strings.HasPrefix(name, "pair-") {
+			continue
+		}
+		state := SessionDetached
+		if exited[name] {
+			state = SessionExited
+		} else if s.clientCount(name) > 0 {
+			state = SessionAttached
+		}
+		out = append(out, Session{Name: name, State: state})
+	}
+	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
+	return out, nil
+}
+
+func (s ZellijSource) clientCount(session string) int {
+	out, err := s.run("--session", session, "action", "list-clients")
+	if err != nil {
+		return 0
+	}
+	lines := lines(string(out))
+	if len(lines) <= 1 {
+		return 0
+	}
+	return len(lines) - 1
+}
+
+func (s ZellijSource) run(args ...string) ([]byte, error) {
+	path := s.Path
+	if path == "" {
+		path = "zellij"
+	}
+	cmd := exec.Command(path, args...)
+	var stdout bytes.Buffer
+	cmd.Stdout = &stdout
+	if err := cmd.Run(); err != nil {
+		return nil, err
+	}
+	return stdout.Bytes(), nil
+}
+
+func exitedSessions(raw string) map[string]bool {
+	out := map[string]bool{}
+	for _, line := range lines(raw) {
+		fields := strings.Fields(line)
+		if len(fields) == 0 {
+			continue
+		}
+		if strings.Contains(line, "EXITED") {
+			out[fields[0]] = true
+		}
+	}
+	return out
+}
+
+func lines(s string) []string {
+	var out []string
+	for _, line := range strings.Split(s, "\n") {
+		line = strings.TrimSpace(line)
+		if line != "" {
+			out = append(out, line)
+		}
+	}
+	return out
+}
diff --git a/cmd/internal/launcher/zellij_test.go b/cmd/internal/launcher/zellij_test.go
new file mode 100644
index 0000000..56bc4b4
--- /dev/null
+++ b/cmd/internal/launcher/zellij_test.go
@@ -0,0 +1,44 @@
+package launcher
+
+import (
+	"os"
+	"path/filepath"
+	"testing"
+)
+
+func TestZellijSourceClassifiesSessions(t *testing.T) {
+	dir := t.TempDir()
+	log := filepath.Join(dir, "zellij.log")
+	zellij := filepath.Join(dir, "zellij")
+	script := `#!/usr/bin/env bash
+printf '%s\n' "$*" >> "` + log + `"
+case "$*" in
+  "list-sessions --short") printf 'pair-live\npair-detached\npair-gone\nother\n' ;;
+  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-gone [Created] (EXITED - attach to resurrect)\n' ;;
+  "--session pair-live action list-clients") printf 'CLIENTS\n1\n' ;;
+  "--session pair-detached action list-clients") printf 'CLIENTS\n' ;;
+  *) exit 1 ;;
+esac
+`
+	if err := os.WriteFile(zellij, []byte(script), 0o755); err != nil {
+		t.Fatal(err)
+	}
+
+	got, err := ZellijSource{Path: zellij}.Snapshot()
+	if err != nil {
+		t.Fatalf("Snapshot returned error: %v", err)
+	}
+	want := []Session{
+		{Name: "pair-detached", State: SessionDetached},
+		{Name: "pair-gone", State: SessionExited},
+		{Name: "pair-live", State: SessionAttached},
+	}
+	if len(got) != len(want) {
+		t.Fatalf("Snapshot returned %#v, want %#v", got, want)
+	}
+	for i := range want {
+		if got[i] != want[i] {
+			t.Fatalf("Snapshot[%d] = %#v, want %#v", i, got[i], want[i])
+		}
+	}
+}
diff --git a/cmd/pair-go/launch_process_test.go b/cmd/pair-go/launch_process_test.go
new file mode 100644
index 0000000..9563492
--- /dev/null
+++ b/cmd/pair-go/launch_process_test.go
@@ -0,0 +1,71 @@
+package main
+
+import (
+	"bytes"
+	"os"
+	"path/filepath"
+	"strings"
+	"testing"
+	"time"
+)
+
+func TestRunLaunchWithFakeZellij(t *testing.T) {
+	rt := t.TempDir()
+	bin := filepath.Join(rt, "bin")
+	data := filepath.Join(rt, "data")
+	if err := os.MkdirAll(bin, 0o755); err != nil {
+		t.Fatal(err)
+	}
+	if err := os.MkdirAll(filepath.Join(data, "pair"), 0o755); err != nil {
+		t.Fatal(err)
+	}
+	log := filepath.Join(rt, "zellij.log")
+	zellij := filepath.Join(bin, "zellij")
+	script := `#!/usr/bin/env bash
+printf '%s\n' "$*" >> "` + log + `"
+case "$*" in
+  "list-sessions --short") printf 'pair-live\npair-detached\npair-exited\n' ;;
+  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-exited [Created] (EXITED - attach to resurrect)\n' ;;
+  "--session pair-live action list-clients") printf 'CLIENTS\n1\n' ;;
+  "--session pair-detached action list-clients") printf 'CLIENTS\n' ;;
+  *attach*|*new-session*|*--new-session-with-layout*|*delete-session*) printf 'MUTATING %s\n' "$*" >> "` + log + `"; exit 99 ;;
+  *) exit 1 ;;
+esac
+`
+	if err := os.WriteFile(zellij, []byte(script), 0o755); err != nil {
+		t.Fatal(err)
+	}
+	historical := filepath.Join(data, "pair", "draft-pair-old.md")
+	if err := os.WriteFile(historical, []byte("draft"), 0o644); err != nil {
+		t.Fatal(err)
+	}
+	now := time.Now()
+	if err := os.Chtimes(historical, now, now); err != nil {
+		t.Fatal(err)
+	}
+
+	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
+	t.Setenv("HOME", filepath.Join(rt, "home"))
+	t.Setenv("XDG_DATA_HOME", data)
+
+	var stdout, stderr bytes.Buffer
+	code := run([]string{"launch", "claude"}, &stdout, &stderr)
+	if code != 3 {
+		t.Fatalf("code = %d, want 3; stderr:\n%s", code, stderr.String())
+	}
+	if stdout.String() != "" {
+		t.Fatalf("stdout = %q, want empty", stdout.String())
+	}
+	for _, want := range []string{"prototype decision", "action=pick"} {
+		if !strings.Contains(stderr.String(), want) {
+			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
+		}
+	}
+	logBytes, err := os.ReadFile(log)
+	if err != nil {
+		t.Fatal(err)
+	}
+	if strings.Contains(string(logBytes), "MUTATING") {
+		t.Fatalf("fake zellij recorded mutating invocation:\n%s", string(logBytes))
+	}
+}
diff --git a/cmd/pair-go/main.go b/cmd/pair-go/main.go
index 29cafe0..bf7d28b 100644
--- a/cmd/pair-go/main.go
+++ b/cmd/pair-go/main.go
@@ -15,6 +15,15 @@ func main() {
 
 func run(args []string, stdout, stderr io.Writer) int {
 	res := dispatcher.Dispatch(args)
+	return writeResult(res, stdout, stderr)
+}
+
+func runWithLauncherRuntime(args []string, stdout, stderr io.Writer, rt dispatcher.LauncherRuntime) int {
+	res := dispatcher.DispatchWithLauncherRuntime(args, rt)
+	return writeResult(res, stdout, stderr)
+}
+
+func writeResult(res dispatcher.Result, stdout, stderr io.Writer) int {
 	if res.Stdout != "" {
 		_, _ = io.WriteString(stdout, res.Stdout)
 	}
diff --git a/cmd/pair-go/main_test.go b/cmd/pair-go/main_test.go
index 1563f04..ca89631 100644
--- a/cmd/pair-go/main_test.go
+++ b/cmd/pair-go/main_test.go
@@ -4,6 +4,8 @@ import (
 	"bytes"
 	"strings"
 	"testing"
+
+	"github.com/xianxu/pair/cmd/internal/dispatcher"
 )
 
 func TestRunWritesStdoutAndReturnsDispatcherCode(t *testing.T) {
@@ -33,3 +35,41 @@ func TestRunWritesStderrAndReturnsDispatcherCode(t *testing.T) {
 		t.Fatalf("stderr missing unsupported-command message:\n%s", stderr.String())
 	}
 }
+
+func TestRunLaunchHelp(t *testing.T) {
+	var stdout, stderr bytes.Buffer
+	code := run([]string{"launch", "--help"}, &stdout, &stderr)
+	if code != 0 {
+		t.Fatalf("code = %d, want 0", code)
+	}
+	if !strings.Contains(stdout.String(), "Usage: pair-go launch") {
+		t.Fatalf("stdout missing launch usage:\n%s", stdout.String())
+	}
+	if stderr.String() != "" {
+		t.Fatalf("stderr = %q, want empty", stderr.String())
+	}
+}
+
+func TestRunLaunchResumeReturnsPrototypeDecision(t *testing.T) {
+	var stdout, stderr bytes.Buffer
+	code := runWithLauncherRuntime([]string{"launch", "resume", "demo"}, &stdout, &stderr, testLauncherRuntime("/home/me", "", "/work/pair"))
+	if code != 3 {
+		t.Fatalf("code = %d, want 3", code)
+	}
+	if stdout.String() != "" {
+		t.Fatalf("stdout = %q, want empty", stdout.String())
+	}
+	for _, want := range []string{"prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
+		if !strings.Contains(stderr.String(), want) {
+			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
+		}
+	}
+}
+
+func testLauncherRuntime(home, xdg, cwd string) dispatcher.LauncherRuntime {
+	return dispatcher.LauncherRuntime{
+		Env:      dispatcher.LauncherEnv(home, xdg, cwd),
+		Sessions: dispatcher.StaticSessions{},
+		History:  dispatcher.StaticHistory{},
+	}
+}


codex
I’ll use `superpowers-using-superpowers` first because the repo instructions require it at conversation start, then I’ll do the boundary review read-only against the issue, plan, and diff.
exec
/bin/zsh -lc "sed -n '1,220p' /Users/xianxu/workspace/ariadne/construct/adapted/superpowers-using-superpowers/SKILL.md && sdlc --help" in /Users/xianxu/workspace/pair
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

codex
I’m gathering the boundary context now: the active issue/plan, lessons, and the actual diff from the requested base so findings are grounded in files rather than the prompt excerpt.
exec
/bin/zsh -lc "sed -n '1,220p' workshop/lessons.md" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "sed -n '1,240p' workshop/issues/000075-go-launcher-prototype.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
---
id: 000075
status: done
deps: [000073, 000074]
github_issue:
created: 2026-06-26
updated: 2026-06-29
estimate_hours: 3.75
started: 2026-06-29T21:55:46-07:00
actual_hours: 0.98
---

# pair Go launcher prototype

## Problem

The launcher is the largest remaining shell surface and the most important packaging target, but it owns many behavioral edges: session picker, tag normalization, resume/continue/rename, zellij lifecycle, quit/restart markers, data-dir migrations, orphan cleanup, cmux title ownership, and dev rebuild behavior. Porting it must not break normal Pair usage.

## Spec

Prototype the launcher core in Go behind an alternate or guarded path. The prototype should implement a coherent vertical slice of `bin/pair` behavior while the shell launcher remains the public default.

The #73 inventory identifies `bin/pair` as the P0 public launcher surface. For this milestone, the guarded path is `pair-go launch`, a development-only launcher prototype that reaches the decision boundary but never starts or attaches a real zellij session. The public `bin/pair` shell launcher remains the only supported user entrypoint.

The vertical slice is:

- argv parsing for `pair-go launch`, including default agent, `resume <tag>`, optional agent positional, and `--` forwarded args;
- tag normalization and default tag derivation using the same bare-or-`pair-` contract as `bin/pair`;
- data-dir resolution from `XDG_DATA_HOME` / `HOME`;
- a session snapshot model that represents live, detached, exited, and historical tag candidates;
- a pure decision model for forced resume attach/create, direct create, picker-required, and historical create;
- a thin command/filesystem shell that can read fake `zellij` output and filesystem sidecars in tests, then print the selected prototype action.

The prototype must fail explicitly after the decision phase, rather than silently launching or diverging from `bin/pair`. Unsupported launcher behavior stays shell-owned and documented as out of scope: `continue`, `rename`, zellij lifecycle execution, quit/restart markers, orphan cleanup, cmux title ownership, dev rebuild, tag-restart prompt, config migration, and real fzf interaction.

The Go model stores canonical tags as bare names such as `demo`. Zellij session names are derived at the boundary as `pair-<tag>`. `LaunchDecision` should carry both `Tag` and derived `SessionName` when the action needs a zellij session so printouts and comparisons cannot mix the two forms.

The implementation should keep business decisions pure (`ARCH-PURE`) and reuse the dispatcher introduced in #74 instead of creating a parallel command parser (`ARCH-DRY`). The slice must still satisfy the issue purpose (`ARCH-PURPOSE`): it is not enough to port helpers; `pair-go launch` has to exercise a coherent launcher decision surface.

## Done when

- [x] A guarded Go launcher path can exercise a documented subset of launcher behavior.
- [x] Existing `bin/pair` remains the default public launcher.
- [x] Tests cover the ported decision logic and at least one process-level fake for external commands.
- [x] Any behavior not yet ported fails explicitly rather than silently diverging.
- [x] Pair remains usable after merge through the existing public command.

## Plan

- [x] Select the launcher slice from #73.
- [x] Extract pure decision models and tests.
- [x] Add fake-command process tests for the selected slice.
- [x] Implement the guarded Go path.
- [x] Document remaining shell-owned launcher behavior.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.60 impl=0.45
item: greenfield-go-module design=1.00 impl=0.45
item: skill-or-dispatcher design=0.30 impl=0.30
item: atlas-docs design=0.10 impl=0.05
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 3.75
```

## Log

### 2026-06-26

Created from #72. This issue is deliberately a prototype/vertical-slice milestone, not the public switch.

### 2026-06-29
- 2026-06-29: closed — go test ./cmd/internal/dispatcher -run 'TestDispatch(Help|Launch)' -count=1; go test ./cmd/pair-go -run 'TestRunLaunch' -count=1; go run ./cmd/pair-go launch reached prototype decision exit 3; go run ./cmd/pair-go help lists launch as prototype; go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; git diff -- bin/pair empty; rg atlas boundary check; git diff --check
- 2026-06-29: closed — go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; git diff -- bin/pair empty; rg atlas boundary check; git diff --check; review verdict: REWORK

Claimed #75 after parking #82. Entered planning with `sdlc start-plan --issue 75`; design cites #73's inventory and chooses a guarded `pair-go launch` decision-phase prototype so the shell launcher remains public while the Go path exercises real launcher concepts (`ARCH-PURE`, `ARCH-DRY`, `ARCH-PURPOSE`).

Plan-quality gate returned FAILURE: estimate was low for the visible multi-surface scope, and tag/session naming was ambiguous. Revised the estimate to 3.75 and clarified that canonical tags are bare while zellij session names are derived as `pair-<tag>`.

Second plan-quality gate returned FAILURE: the process-level test was ordered before the dispatcher route it needs, and the plan risked duplicating `dispatcher.Result`. Reordered route before process test and made `dispatcher.Result` the sole process-facing result abstraction (`ARCH-DRY`).

Third plan-quality gate returned FAILURE: Task 4 still named a `LaunchResult` return despite the single-result-abstraction rule. Revised the runner contract so launcher returns domain `LaunchOutcome` values and dispatcher alone maps to `dispatcher.Result`; also named the production IO constructor and test runtime seam.

Implemented `cmd/internal/launcher` as a pure decision-phase core plus fakeable zellij/history seams. `pair-go launch` now routes through the #74 dispatcher, returns explicit prototype decisions, and does not mutate zellij or replace `bin/pair`. Updated atlas architecture and the Go migration inventory to record the shell-owned boundary.

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


exec
/bin/zsh -lc "rg -n \"Core concepts|cmd/internal/launcher|pair-go launch|M[0-9]|\\[ \\]|\\[x\\]|\\[X\\]|Spec|Plan|Log|Revisions\" workshop/issues/000075-go-launcher-prototype.md workshop/plans atlas cmd -g '"'!workshop/history/**'"'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
workshop/issues/000075-go-launcher-prototype.md:19:## Spec
workshop/issues/000075-go-launcher-prototype.md:23:The #73 inventory identifies `bin/pair` as the P0 public launcher surface. For this milestone, the guarded path is `pair-go launch`, a development-only launcher prototype that reaches the decision boundary but never starts or attaches a real zellij session. The public `bin/pair` shell launcher remains the only supported user entrypoint.
workshop/issues/000075-go-launcher-prototype.md:27:- argv parsing for `pair-go launch`, including default agent, `resume <tag>`, optional agent positional, and `--` forwarded args;
workshop/issues/000075-go-launcher-prototype.md:38:The implementation should keep business decisions pure (`ARCH-PURE`) and reuse the dispatcher introduced in #74 instead of creating a parallel command parser (`ARCH-DRY`). The slice must still satisfy the issue purpose (`ARCH-PURPOSE`): it is not enough to port helpers; `pair-go launch` has to exercise a coherent launcher decision surface.
workshop/issues/000075-go-launcher-prototype.md:42:- [x] A guarded Go launcher path can exercise a documented subset of launcher behavior.
workshop/issues/000075-go-launcher-prototype.md:43:- [x] Existing `bin/pair` remains the default public launcher.
workshop/issues/000075-go-launcher-prototype.md:44:- [x] Tests cover the ported decision logic and at least one process-level fake for external commands.
workshop/issues/000075-go-launcher-prototype.md:45:- [x] Any behavior not yet ported fails explicitly rather than silently diverging.
workshop/issues/000075-go-launcher-prototype.md:46:- [x] Pair remains usable after merge through the existing public command.
workshop/issues/000075-go-launcher-prototype.md:48:## Plan
workshop/issues/000075-go-launcher-prototype.md:50:- [x] Select the launcher slice from #73.
workshop/issues/000075-go-launcher-prototype.md:51:- [x] Extract pure decision models and tests.
workshop/issues/000075-go-launcher-prototype.md:52:- [x] Add fake-command process tests for the selected slice.
workshop/issues/000075-go-launcher-prototype.md:53:- [x] Implement the guarded Go path.
workshop/issues/000075-go-launcher-prototype.md:54:- [x] Document remaining shell-owned launcher behavior.
workshop/issues/000075-go-launcher-prototype.md:72:## Log
workshop/issues/000075-go-launcher-prototype.md:79:- 2026-06-29: closed — go test ./cmd/internal/dispatcher -run 'TestDispatch(Help|Launch)' -count=1; go test ./cmd/pair-go -run 'TestRunLaunch' -count=1; go run ./cmd/pair-go launch reached prototype decision exit 3; go run ./cmd/pair-go help lists launch as prototype; go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; git diff -- bin/pair empty; rg atlas boundary check; git diff --check
workshop/issues/000075-go-launcher-prototype.md:80:- 2026-06-29: closed — go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; git diff -- bin/pair empty; rg atlas boundary check; git diff --check; review verdict: REWORK
workshop/issues/000075-go-launcher-prototype.md:82:Claimed #75 after parking #82. Entered planning with `sdlc start-plan --issue 75`; design cites #73's inventory and chooses a guarded `pair-go launch` decision-phase prototype so the shell launcher remains public while the Go path exercises real launcher concepts (`ARCH-PURE`, `ARCH-DRY`, `ARCH-PURPOSE`).
workshop/issues/000075-go-launcher-prototype.md:84:Plan-quality gate returned FAILURE: estimate was low for the visible multi-surface scope, and tag/session naming was ambiguous. Revised the estimate to 3.75 and clarified that canonical tags are bare while zellij session names are derived as `pair-<tag>`.
workshop/issues/000075-go-launcher-prototype.md:90:Implemented `cmd/internal/launcher` as a pure decision-phase core plus fakeable zellij/history seams. `pair-go launch` now routes through the #74 dispatcher, returns explicit prototype decisions, and does not mutate zellij or replace `bin/pair`. Updated atlas architecture and the Go migration inventory to record the shell-owned boundary.
workshop/plans/000075-go-launcher-prototype-plan.md:1:# Pair Go Launcher Prototype Implementation Plan
workshop/plans/000075-go-launcher-prototype-plan.md:3:> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.
workshop/plans/000075-go-launcher-prototype-plan.md:5:**Goal:** Add a guarded `pair-go launch` prototype that reaches Pair launcher decision logic without replacing or invoking the public shell launcher.
workshop/plans/000075-go-launcher-prototype-plan.md:7:**Architecture:** Put launcher business logic in a pure `cmd/internal/launcher` package and keep subprocess/filesystem work in thin, injectable seams (`ARCH-PURE`). Extend the existing #74 dispatcher instead of adding a parallel command parser (`ARCH-DRY`). The prototype prints the decision it would take, then exits with an explicit unsupported-after-decision code so the issue delivers a real launcher vertical slice without changing public `bin/pair` behavior (`ARCH-PURPOSE`).
workshop/plans/000075-go-launcher-prototype-plan.md:19:| `LaunchArgs` | `cmd/internal/launcher/args.go` | new |
workshop/plans/000075-go-launcher-prototype-plan.md:20:| `Tag` | `cmd/internal/launcher/tag.go` | new |
workshop/plans/000075-go-launcher-prototype-plan.md:21:| `DataDir` | `cmd/internal/launcher/datadir.go` | new |
workshop/plans/000075-go-launcher-prototype-plan.md:22:| `SessionSnapshot` | `cmd/internal/launcher/session.go` | new |
workshop/plans/000075-go-launcher-prototype-plan.md:23:| `LaunchDecision` | `cmd/internal/launcher/decision.go` | new |
workshop/plans/000075-go-launcher-prototype-plan.md:26:- **Relationships:** 1:1 with a `pair-go launch` invocation; owns a `Tag` only for forced resume.
workshop/plans/000075-go-launcher-prototype-plan.md:27:- **DRY rationale:** Keeps `pair-go launch` parsing behind the existing dispatcher instead of duplicating parser branches in `cmd/pair-go/main.go`.
workshop/plans/000075-go-launcher-prototype-plan.md:56:| `ZellijSource` | `cmd/internal/launcher/zellij.go` | new | `zellij list-sessions`, `zellij --session ... action list-clients` |
workshop/plans/000075-go-launcher-prototype-plan.md:57:| `HistorySource` | `cmd/internal/launcher/history.go` | new | filesystem scan of `draft-*.md` and `log-*.md` under Pair data dir |
workshop/plans/000075-go-launcher-prototype-plan.md:69:**LaunchCommand** — Routes `pair-go launch` from the existing dispatcher to the launcher runner.
workshop/plans/000075-go-launcher-prototype-plan.md:78:## Revisions
workshop/plans/000075-go-launcher-prototype-plan.md:88:### Task 1: Parse `pair-go launch` Args
workshop/plans/000075-go-launcher-prototype-plan.md:91:- Create: `cmd/internal/launcher/args.go`
workshop/plans/000075-go-launcher-prototype-plan.md:92:- Create: `cmd/internal/launcher/args_test.go`
workshop/plans/000075-go-launcher-prototype-plan.md:94:- [ ] **Step 1: Write failing parse tests**
workshop/plans/000075-go-launcher-prototype-plan.md:104:Run: `go test ./cmd/internal/launcher -run 'TestParseLaunchArgs' -count=1`
workshop/plans/000075-go-launcher-prototype-plan.md:107:- [ ] **Step 2: Implement minimal parser**
workshop/plans/000075-go-launcher-prototype-plan.md:111:- [ ] **Step 3: Verify parse tests pass**
workshop/plans/000075-go-launcher-prototype-plan.md:113:Run: `go test ./cmd/internal/launcher -run 'TestParseLaunchArgs' -count=1`
workshop/plans/000075-go-launcher-prototype-plan.md:119:- Create: `cmd/internal/launcher/tag.go`
workshop/plans/000075-go-launcher-prototype-plan.md:120:- Create: `cmd/internal/launcher/tag_test.go`
workshop/plans/000075-go-launcher-prototype-plan.md:121:- Create: `cmd/internal/launcher/datadir.go`
workshop/plans/000075-go-launcher-prototype-plan.md:122:- Create: `cmd/internal/launcher/datadir_test.go`
workshop/plans/000075-go-launcher-prototype-plan.md:124:- [ ] **Step 1: Write failing helper tests**
workshop/plans/000075-go-launcher-prototype-plan.md:133:Run: `go test ./cmd/internal/launcher -run 'TestNormalizeTag|TestDefaultTag|TestResolveDataDir' -count=1`
workshop/plans/000075-go-launcher-prototype-plan.md:136:- [ ] **Step 2: Implement helpers**
workshop/plans/000075-go-launcher-prototype-plan.md:140:- [ ] **Step 3: Verify helper tests pass**
workshop/plans/000075-go-launcher-prototype-plan.md:142:Run: `go test ./cmd/internal/launcher -run 'TestNormalizeTag|TestDefaultTag|TestResolveDataDir' -count=1`
workshop/plans/000075-go-launcher-prototype-plan.md:148:- Create: `cmd/internal/launcher/session.go`
workshop/plans/000075-go-launcher-prototype-plan.md:149:- Create: `cmd/internal/launcher/decision.go`
workshop/plans/000075-go-launcher-prototype-plan.md:150:- Create: `cmd/internal/launcher/decision_test.go`
workshop/plans/000075-go-launcher-prototype-plan.md:152:- [ ] **Step 1: Write failing decision tests**
workshop/plans/000075-go-launcher-prototype-plan.md:163:Run: `go test ./cmd/internal/launcher -run 'TestDecideLaunch' -count=1`
workshop/plans/000075-go-launcher-prototype-plan.md:166:- [ ] **Step 2: Implement models**
workshop/plans/000075-go-launcher-prototype-plan.md:190:- [ ] **Step 3: Verify decision tests pass**
workshop/plans/000075-go-launcher-prototype-plan.md:192:Run: `go test ./cmd/internal/launcher -run 'TestDecideLaunch' -count=1`
workshop/plans/000075-go-launcher-prototype-plan.md:195:- [ ] **Step 4: Commit pure core**
workshop/plans/000075-go-launcher-prototype-plan.md:198:git add cmd/internal/launcher
workshop/plans/000075-go-launcher-prototype-plan.md:207:- Create: `cmd/internal/launcher/zellij.go`
workshop/plans/000075-go-launcher-prototype-plan.md:208:- Create: `cmd/internal/launcher/history.go`
workshop/plans/000075-go-launcher-prototype-plan.md:209:- Create: `cmd/internal/launcher/run.go`
workshop/plans/000075-go-launcher-prototype-plan.md:210:- Create: `cmd/internal/launcher/run_test.go`
workshop/plans/000075-go-launcher-prototype-plan.md:212:- [ ] **Step 1: Write failing runner tests with in-memory fakes**
workshop/plans/000075-go-launcher-prototype-plan.md:220:Run: `go test ./cmd/internal/launcher -run 'TestRunLaunch' -count=1`
workshop/plans/000075-go-launcher-prototype-plan.md:223:- [ ] **Step 2: Implement interfaces and runner**
workshop/plans/000075-go-launcher-prototype-plan.md:248:- [ ] **Step 3: Verify runner tests pass**
workshop/plans/000075-go-launcher-prototype-plan.md:250:Run: `go test ./cmd/internal/launcher -run 'TestRunLaunch' -count=1`
workshop/plans/000075-go-launcher-prototype-plan.md:253:### Task 5: Route `pair-go launch`
workshop/plans/000075-go-launcher-prototype-plan.md:261:- [ ] **Step 1: Update dispatcher tests**
workshop/plans/000075-go-launcher-prototype-plan.md:268:- [ ] **Step 2: Implement the route**
workshop/plans/000075-go-launcher-prototype-plan.md:272:- [ ] **Step 3: Update `pair-go` tests**
workshop/plans/000075-go-launcher-prototype-plan.md:276:- `pair-go launch --help` prints launch usage;
workshop/plans/000075-go-launcher-prototype-plan.md:277:- `pair-go launch resume demo` returns a prototype decision message, not a real launch.
workshop/plans/000075-go-launcher-prototype-plan.md:282:- [ ] **Step 4: Commit IO and route**
workshop/plans/000075-go-launcher-prototype-plan.md:285:git add cmd/internal/launcher cmd/internal/dispatcher cmd/pair-go
workshop/plans/000075-go-launcher-prototype-plan.md:286:git commit -m "#75: add guarded Go launcher prototype" -m "Route pair-go launch through a fakeable launcher runner that reaches create/attach decisions without replacing bin/pair." -m "Co-Authored-By: GPT-5 Codex <codex@openai.com>"
workshop/plans/000075-go-launcher-prototype-plan.md:295:- [ ] **Step 1: Write failing process-level fake test**
workshop/plans/000075-go-launcher-prototype-plan.md:313:- [ ] **Step 2: Make the dispatcher/cmd path support injected launcher runtime for tests**
workshop/plans/000075-go-launcher-prototype-plan.md:317:- [ ] **Step 3: Verify process test passes**
workshop/plans/000075-go-launcher-prototype-plan.md:330:- [ ] **Step 1: Update atlas**
workshop/plans/000075-go-launcher-prototype-plan.md:332:Record that `pair-go launch` now owns only the decision-phase prototype. Explicitly list shell-owned behavior still out of scope: real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller.
workshop/plans/000075-go-launcher-prototype-plan.md:334:- [ ] **Step 2: Verify docs mention the boundary**
workshop/plans/000075-go-launcher-prototype-plan.md:336:Run: `rg -n "pair-go launch|decision-phase|shell-owned" atlas/architecture.md atlas/go-migration-inventory.md`
workshop/plans/000075-go-launcher-prototype-plan.md:339:- [ ] **Step 3: Commit docs**
workshop/plans/000075-go-launcher-prototype-plan.md:343:git commit -m "#75: document launcher prototype boundary" -m "Clarify that pair-go launch is a guarded decision-phase prototype while bin/pair remains the public launcher." -m "Co-Authored-By: GPT-5 Codex <codex@openai.com>"
workshop/plans/000075-go-launcher-prototype-plan.md:351:- [ ] **Step 1: Run focused Go tests**
workshop/plans/000075-go-launcher-prototype-plan.md:353:Run: `go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
workshop/plans/000075-go-launcher-prototype-plan.md:356:- [ ] **Step 2: Build `pair-go`**
workshop/plans/000075-go-launcher-prototype-plan.md:361:- [ ] **Step 3: Run full Go suite**
workshop/plans/000075-go-launcher-prototype-plan.md:366:- [ ] **Step 4: Verify public launcher unchanged**
workshop/plans/000075-go-launcher-prototype-plan.md:371:- [ ] **Step 5: Verify docs and whitespace**
workshop/plans/000075-go-launcher-prototype-plan.md:373:Run: `rg -n "pair-go launch|decision-phase|shell-owned" atlas/architecture.md atlas/go-migration-inventory.md && git diff --check`
workshop/plans/000075-go-launcher-prototype-plan.md:376:- [ ] **Step 6: Close through SDLC**
workshop/plans/000075-go-launcher-prototype-plan.md:378:Run: `sdlc close --issue 75 --verified 'go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; git diff -- bin/pair empty; rg atlas boundary check; git diff --check'`
cmd/pair-scrollback-render/main.go:281:			row[x] = *c
atlas/how-to-bring-up-a-new-harness-cli.md:48:- Implement the detector. Detectors can scan the rolling output stream for custom OSC escape sequences (e.g. Claude's permission OSC `OSC 777;notify;...`, or Codex's `OSC 9;Plan mode prompt:...`) or fallback to visible text substring matches (e.g., watching for `"Press enter to confirm"`).
atlas/how-to-bring-up-a-new-harness-cli.md:152:1. [ ] **Verify Return Key remapping** in `sendKeymapByAgent` (Enter = newline, Alt+Enter = send).
atlas/how-to-bring-up-a-new-harness-cli.md:153:2. [ ] **Check for blocking TUI overlays** and implement a PTY overlay detector in `overlayDetectorByAgent` if needed.
atlas/how-to-bring-up-a-new-harness-cli.md:154:3. [ ] **Implement Session Watching** in `bin/pair-session-watch.sh` (using `lsof` and target file patterns).
atlas/how-to-bring-up-a-new-harness-cli.md:155:4. [ ] **Configure Launcher Recovery** in `bin/pair` (mapping `--conversation` or `--resume` flags).
atlas/how-to-bring-up-a-new-harness-cli.md:156:5. [ ] **Add slug generation support** in `pair-slug` (transcript parsing + sandboxed print execution).
atlas/how-to-bring-up-a-new-harness-cli.md:157:6. [ ] **Confirm mouse scroll and scrollback render** work smoothly without drawing glitch issues.
atlas/how-to-bring-up-a-new-harness-cli.md:158:7. [ ] **White-list permissions** in the agent's global or workspace settings directory.
atlas/how-to-bring-up-a-new-harness-cli.md:159:8. [ ] **Register the user-prompt glyph** in `nvim/scrollback.lua` for `Alt+b` jumping.
atlas/how-to-bring-up-a-new-harness-cli.md:209:> Status: all six runtime aspects emit today (#000045 M1: aspects 1 & 2; M2: aspects 3, 4, 5, 7).
atlas/review-workbench.md:26:- `markers.lua` (M2) — pure 🤖 review-request parser (ported from parley):
atlas/review-workbench.md:29:  in-doc review requests; M3 highlights from it, M4's agent reads it.
atlas/review-workbench.md:30:- `mode.lua` (M2/M4d) — pure pair-side UI metadata for the 3 human assistance
atlas/review-workbench.md:32:  their meanings live in ariadne's `xx-fix` skill. `menu.lua` (M4c/M4d) presents
atlas/review-workbench.md:52:- `apply.snapshot`/`apply.apply_snapshot` (M2) — read/restore the decoration
atlas/review-workbench.md:55:- `projection.lua` (M2) — decoration coherence across undo/redo (ported from
atlas/review-workbench.md:97:## The review window (M3)
atlas/review-workbench.md:157:  don't leak in — `🤖 0/0` off a review branch / in M3 render-only; mode from
atlas/review-workbench.md:177:that makes "please review" / "ship it" review-aware is the ariadne #000121 half of M4.
atlas/review-workbench.md:181:M1 (contract + history spine), M2 (consumer-half port), M3 (review window + live
atlas/review-workbench.md:182:smoke), M4a (nvim writes no git; fake-agent commits from landed artifacts), and
atlas/review-workbench.md:183:M4a' pair-side review-start/resume are implemented and headless-tested. M4b adds
atlas/review-workbench.md:185:and ship request. M4c adds pair-side mode display/menu and the awaiting-agent
atlas/review-workbench.md:186:spinner. M4d starts workflow-detail tuning with one-round instruction menu polish,
atlas/review-workbench.md:204:  and `projection` (undo/redo coherence + riding + round-2 idempotence); M3 adds
cmd/pair-continuation/main.go:59:	// The writer is the structural guard (Spec Done-when): continuation.md
cmd/pair-wrap/main.go:189:	scrollbackLog  string
cmd/pair-wrap/main.go:191:	debugLogPath   string
cmd/pair-wrap/main.go:240:	adapt        *adapt.Logger
cmd/pair-wrap/main.go:469:	if p.debugLogPath == "" {
cmd/pair-wrap/main.go:472:	f, err := os.OpenFile(p.debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
cmd/pair-wrap/main.go:523:// slug (#000027 M3). Debounced by slugDebounceS so closely-spaced turn-end
cmd/pair-wrap/main.go:598:const codexQuestionOSC9Prefix = "Plan mode prompt:"
cmd/pair-wrap/main.go:748:						p.adapt.Log(5, "output-filter", adapt.Fired, "stripped "+mk)
cmd/pair-wrap/main.go:1364:			p.adapt.Log(2, "overlay-detect", adapt.Fired, p.agentBasename+": "+reason)
cmd/pair-wrap/main.go:1379:			p.adapt.Log(2, "overlay-detect", adapt.NearMiss, p.agentBasename+": unmatched prompt-shaped output: "+snippet)
cmd/pair-wrap/main.go:1462:		p.adapt.Log(1, "return-remap", adapt.Bypass, "plain Enter → bare CR (overlay active)")
cmd/pair-wrap/main.go:1465:	p.adapt.Log(1, "return-remap", adapt.Fired, "plain Enter → newline remap")
cmd/pair-wrap/main.go:1679:// maybeLogTime drops a minute-debounced wall-clock event into the scrollback
cmd/pair-wrap/main.go:1683:func (p *proxy) maybeLogTime() {
cmd/pair-wrap/main.go:1758:		debugLogPath:  os.Getenv("PAIR_WRAP_LOG"),
cmd/pair-wrap/main.go:1769:			p.scrollbackLog = argv[1]
cmd/pair-wrap/main.go:1806:	if p.scrollbackLog != "" {
cmd/pair-wrap/main.go:1807:		eventsPath := strings.TrimSuffix(p.scrollbackLog, ".raw") + ".events.jsonl"
cmd/pair-wrap/main.go:1808:		if !strings.HasSuffix(p.scrollbackLog, ".raw") {
cmd/pair-wrap/main.go:1809:			eventsPath = p.scrollbackLog + ".events.jsonl"
cmd/pair-wrap/main.go:1811:		f, err := os.OpenFile(p.scrollbackLog, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
cmd/pair-wrap/main.go:1813:			p.debug("SCROLLBACK-open-fail", fmt.Sprintf("%q: %v", p.scrollbackLog, err))
cmd/pair-wrap/main.go:1816:			p.debug("SCROLLBACK-open", p.scrollbackLog)
cmd/pair-wrap/main.go:1965:			if p.debugLogPath != "" {
cmd/pair-wrap/main.go:2191:			p.maybeLogTime() // minute-debounced timestamp for change-log dates (#59)
cmd/pair-changelog/main_test.go:70:func run(t *testing.T, bin, cleaned, priorLog, priorAnchor string) (log, anchor string) {
cmd/pair-changelog/main_test.go:72:	return runIn(t, bin, t.TempDir(), cleaned, priorLog, priorAnchor)
cmd/pair-changelog/main_test.go:78:func runIn(t *testing.T, bin, dir, cleaned, priorLog, priorAnchor string) (log, anchor string) {
cmd/pair-changelog/main_test.go:84:	if priorLog != "" {
cmd/pair-changelog/main_test.go:85:		mustWrite(t, logPath, priorLog)
cmd/pair-changelog/main_test.go:136:	priorLog := "- one\n\n- two\n"
cmd/pair-changelog/main_test.go:138:	log, anchor := run(t, bin, cleaned, priorLog, priorAnchor)
cmd/pair-changelog/main_test.go:158:	priorLog := "- one\n\n- two\n"
cmd/pair-changelog/main_test.go:160:	log, _ := run(t, bin, cleaned, priorLog, priorAnchor)
cmd/pair-changelog/main_test.go:201:	priorLog := "## 2026-06-13\n\n- one\n\n- two\n"
cmd/pair-changelog/main_test.go:203:	log, _ := run(t, bin, cleaned, priorLog, priorAnchor)
cmd/pair-changelog/main_test.go:240:func TestFullRedistillWithPriorLogKeepsFrozenPrefix(t *testing.T) {
cmd/pair-changelog/main_test.go:245:	priorLog := "- one\n\n- two\n"
cmd/pair-changelog/main_test.go:247:	log, anchor := run(t, bin, cleaned, priorLog, priorAnchor)
cmd/pair-changelog/main_test.go:272:	priorLog := "- one\n\n- two\n"
cmd/pair-changelog/main_test.go:277:	log, _ := run(t, bin, cleaned, priorLog, priorAnchor)
cmd/pair-changelog/main_test.go:279:	if log != priorLog {
cmd/pair-changelog/main_test.go:298:	priorLog := "- old one\n\n- old two\n"
cmd/pair-changelog/main_test.go:302:	log, anchor := run(t, bin, cleaned, priorLog, priorAnchor)
cmd/pair-changelog/main_test.go:329:	// unchanged last entry → newLog == priorLog. Anchor must advance to turns:2.
cmd/pair-changelog/main_test.go:392:	priorLog := "- one\n"
cmd/pair-changelog/main_test.go:394:	run(t, bin, b.String(), priorLog, priorAnchor)
cmd/pair-changelog/main_test.go:436:	priorLog := "- one\n\n- two\n"
cmd/pair-changelog/main_test.go:438:	log, _ := run(t, bin, cleaned, priorLog, priorAnchor)
cmd/pair-changelog/main_test.go:442:	if log != priorLog {
cmd/pair-changelog/prompt.go:22:const entryGuidance = `A change-log entry is one of: a milestone started or finished ("M1 started",
cmd/pair-changelog/prompt.go:23:"M1 done"); a design or product decision; a significant change (a feature
cmd/pair-changelog/main.go:71:	priorLog := readFileOr(logPath)
cmd/pair-changelog/main.go:74:	hasPrior := strings.TrimSpace(priorLog) != ""
cmd/pair-changelog/main.go:115:	newLog := priorLog // "" on a first-ever run
cmd/pair-changelog/main.go:124:			nl, err := distillStep(newLog, strings.Join(chunk, "\n"), agent, modelName, seg.date)
cmd/pair-changelog/main.go:128:			if nl == newLog {
cmd/pair-changelog/main.go:131:			newLog = nl
cmd/pair-changelog/main.go:132:			if err := atomicWrite(logPath, newLog); err != nil {
cmd/pair-changelog/main.go:139:	if strings.TrimSpace(newLog) == "" {
cmd/pair-changelog/main.go:151:	if newLog == priorLog {
cmd/pair-changelog/main.go:171:// distillStep runs one model distill — first-run (priorLog empty) or incremental
cmd/pair-changelog/main.go:173:// priorLog unchanged when the model produces nothing; errors on a non-distill
cmd/pair-changelog/main.go:175:func distillStep(priorLog, sliceText, agent, modelName, date string) (string, error) {
cmd/pair-changelog/main.go:176:	firstRun := strings.TrimSpace(priorLog) == ""
cmd/pair-changelog/main.go:182:		frozen, ek = splitFrozenTail(priorLog)
cmd/pair-changelog/main.go:195:		return priorLog, nil // model produced nothing; no change this step
cmd/pair-changelog/main.go:206:	return assemble(frozen, ekPrime, newEntries, date, lastHeaderDate(priorLog)), nil
cmd/pair-wrap/time_event_test.go:26:// maybeLogTime emits a debounced "time" event to the events sidecar: the first
cmd/pair-wrap/time_event_test.go:29:func TestMaybeLogTimeDebounced(t *testing.T) {
cmd/pair-wrap/time_event_test.go:39:	p.maybeLogTime() // first → emit
cmd/pair-wrap/time_event_test.go:40:	p.maybeLogTime() // same instant → skip
cmd/pair-wrap/time_event_test.go:42:	p.maybeLogTime() // >1min → emit
atlas/go-migration-inventory.md:50:| `bin/pair` / `cmd/internal/launcher` | Bash public launcher plus guarded Go decision prototype | user shell, `bin/pair-dev`, restart re-exec, tests, development-only `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch` (#75) parses a documented subset, snapshots zellij/history state, prints the decision-phase prototype action, and exits before mutating zellij. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/launcher` resolves data-dir inputs and reads zellij/history through fakeable seams. | go-entrypoint with `bin/pair` retained as compat fallback through #77; #75 owns decision-phase only, while real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
cmd/pair-wrap/adapt_drift_test.go:99:// TestEmitPlainCR_LogsFiredAndBypass covers aspect 1's telemetry: a normal
cmd/pair-wrap/adapt_drift_test.go:103:func TestEmitPlainCR_LogsFiredAndBypass(t *testing.T) {
cmd/pair-wrap/adapt_drift_test.go:126:// TestOutputFilter_LogsFiredDedupedPerMarker covers aspect 5: stripping a
cmd/pair-wrap/adapt_drift_test.go:129:func TestOutputFilter_LogsFiredDedupedPerMarker(t *testing.T) {
cmd/pair-slug/main.go:4:// notify point — so it works for claude/codex/agy alike (issue #000027 M3,
cmd/pair-slug/main.go:203:		lg.Log(4, "slug-parse", adapt.Fail, "could not resolve transcript for agent "+agent)
cmd/pair-slug/main.go:210:		lg.Log(4, "slug-parse", adapt.Fail, "read transcript: "+err.Error())
cmd/pair-slug/main.go:216:		lg.Log(4, "slug-parse", adapt.NearMiss, "transcript read but 0 turns extracted (agent="+agent+")")
cmd/pair-slug/main.go:219:	lg.Log(4, "slug-parse", adapt.Fired, fmt.Sprintf("%d turns", len(turns)))
cmd/pair-wrap/overlay_test.go:53:			raw:       []byte("\x1b]9;Plan mode prompt: Probe\x07"),
cmd/pair-wrap/overlay_test.go:55:			wantMatch: "Plan mode prompt: Probe",
cmd/pair-go/main_test.go:45:	if !strings.Contains(stdout.String(), "Usage: pair-go launch") {
cmd/pair-slug/slug.go:27:// break the "two segments separated by ' | '" channel contract M2 parses. We
cmd/pair-slug/slug.go:300://   - focus has | or ===     → no write (would confuse M2's line-1 parser)
cmd/pair-slug/slug.go:321:	// written slug and confuse nvim's line-1 parse in M2. Reject it.
atlas/architecture.md:49:`pair-go launch` decision-phase prototype. The public launcher remains
atlas/architecture.md:50:`bin/pair`; `pair-go launch` parses the launcher subset, resolves the Pair data
atlas/architecture.md:52:decisions through the pure `cmd/internal/launcher` core. It then stops with an
atlas/architecture.md:196:     **Overlay-aware suspension (per-agent).** Textarea Enter remaps are wrong while a blocking overlay / picker has focus: the overlay needs a bare `\r` to confirm the highlighted option. pair-wrap registers per-agent overlay detectors in `overlayDetectorByAgent`, sets `pickerActive` when one fires, and emits a bare `\r` for the next plain Enter only. The flag clears after that one Enter, so normal textarea remapping resumes for the following keystroke. Claude uses the stable `OSC 777;notify;Claude Code;Claude needs your permission` body. Codex question prompts use `OSC 9;Plan mode prompt:...`; other Codex pickers fall back to stripped visible output plus a short text carryover watching for labels such as `Use session directory (` / `Use current directory (`, `Press enter to continue`, and `Press enter to confirm or esc to go back`. Codex image attach uses a stronger local signal: Alt+i arms pair-wrap capture immediately before Ctrl+V, and that capture arm also enables the next-Enter overlay bypass. The carryover is cleared when the confirming Enter is consumed so stale picker text cannot re-arm the flag. Known edge inherited from the one-shot design: dismissing an overlay without Enter leaves the flag set until the next plain Enter.
atlas/architecture.md:216:     Log lines:
atlas/architecture.md:329:- A contextual `[key=action]` hint appears inside the brackets — `[q=queue]` on `*`/`-N` (Alt+q parks/forks to queue front), `[⌫=del]` on `+N` (Alt+BS deletes the item). Bracket convention: TUI status-bar "key badge" idiom (`[Esc] cancel` etc.). Distinct from the prompt convention `( ) [ ]` for access-key-vs-default, which only applies to interactive dialogs.
atlas/architecture.md:617:**Atomicity.** The full `(src, dst)` plan is written to `$PAIR_DATA_DIR/.rename-<old>-to-<new>.journal` before any `mv` runs. On mid-flight failure, the renamer reads the first N journal lines, swaps columns, and `mv`s the completed renames back to their original paths. The journal is cleared on success and retained on rollback failure as a forensic breadcrumb (M3 will add crash-recovery: a stale journal on startup gets finished or rolled back automatically).
workshop/plans/000075-go-launcher-prototype-close-review.md:44:behavior against the issue's documented Spec/Plan and the code itself; do NOT
workshop/plans/000075-go-launcher-prototype-close-review.md:49:Read the diff against the issue's Spec + Plan, then work the checklist below.
workshop/plans/000075-go-launcher-prototype-close-review.md:79:  - Every Plan checklist item this boundary claims is actually delivered.
workshop/plans/000075-go-launcher-prototype-close-review.md:80:  - Implementation matches the Spec; no undeclared scope creep.
workshop/plans/000075-go-launcher-prototype-close-review.md:87:## Core concepts cross-check (if the plan has a Core concepts table)
workshop/plans/000075-go-launcher-prototype-close-review.md:100:recommendation (a "## Revisions" entry so the plan stops claiming what the code
workshop/plans/000075-go-launcher-prototype-close-review.md:138:  7. Plan revision recommendations: specific "## Revisions" entries the plan
workshop/plans/000075-go-launcher-prototype-close-review.md:150:marker (e.g. `ARCH-DRY`) in plans, `## Log` entries, and review findings.
workshop/plans/000075-go-launcher-prototype-close-review.md:225:+`pair-go launch` decision-phase prototype. The public launcher remains
workshop/plans/000075-go-launcher-prototype-close-review.md:226:+`bin/pair`; `pair-go launch` parses the launcher subset, resolves the Pair data
workshop/plans/000075-go-launcher-prototype-close-review.md:228:+decisions through the pure `cmd/internal/launcher` core. It then stops with an
workshop/plans/000075-go-launcher-prototype-close-review.md:245:+| `bin/pair` / `cmd/internal/launcher` | Bash public launcher plus guarded Go decision prototype | user shell, `bin/pair-dev`, restart re-exec, tests, development-only `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch` (#75) parses a documented subset, snapshots zellij/history state, prints the decision-phase prototype action, and exits before mutating zellij. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/launcher` resolves data-dir inputs and reads zellij/history through fakeable seams. | go-entrypoint with `bin/pair` retained as compat fallback through #77; #75 owns decision-phase only, while real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
workshop/plans/000075-go-launcher-prototype-close-review.md:261:+	"github.com/xianxu/pair/cmd/internal/launcher"
workshop/plans/000075-go-launcher-prototype-close-review.md:318:+		return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
workshop/plans/000075-go-launcher-prototype-close-review.md:323:+			"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
workshop/plans/000075-go-launcher-prototype-close-review.md:372:@@ -60,6 +60,42 @@ func TestDispatchPlannedCommandReturnsUnsupported(t *testing.T) {
workshop/plans/000075-go-launcher-prototype-close-review.md:384:+	for _, want := range []string{"Usage: pair-go launch", "decision-phase prototype"} {
workshop/plans/000075-go-launcher-prototype-close-review.md:405:+	for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
workshop/plans/000075-go-launcher-prototype-close-review.md:415:diff --git a/cmd/internal/launcher/args.go b/cmd/internal/launcher/args.go
workshop/plans/000075-go-launcher-prototype-close-review.md:419:+++ b/cmd/internal/launcher/args.go
workshop/plans/000075-go-launcher-prototype-close-review.md:425:+// LaunchArgs is the pure parse result for the guarded pair-go launch prototype.
workshop/plans/000075-go-launcher-prototype-close-review.md:442:+// ParseArgs parses pair-go launch args. It intentionally supports only the
workshop/plans/000075-go-launcher-prototype-close-review.md:454:+		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %s is not implemented by pair-go launch; use bin/pair", argv[0])}
workshop/plans/000075-go-launcher-prototype-close-review.md:457:+			return LaunchArgs{}, UsageError{Message: "pair-go launch: 'resume' requires a tag"}
workshop/plans/000075-go-launcher-prototype-close-review.md:461:+			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: invalid tag: %v", err)}
workshop/plans/000075-go-launcher-prototype-close-review.md:464:+			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", argv[2])}
workshop/plans/000075-go-launcher-prototype-close-review.md:484:+		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", arg)}
workshop/plans/000075-go-launcher-prototype-close-review.md:492:diff --git a/cmd/internal/launcher/args_test.go b/cmd/internal/launcher/args_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:496:+++ b/cmd/internal/launcher/args_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:580:+			if !strings.Contains(err.Error(), "not implemented by pair-go launch") {
workshop/plans/000075-go-launcher-prototype-close-review.md:586:diff --git a/cmd/internal/launcher/datadir.go b/cmd/internal/launcher/datadir.go
workshop/plans/000075-go-launcher-prototype-close-review.md:590:+++ b/cmd/internal/launcher/datadir.go
workshop/plans/000075-go-launcher-prototype-close-review.md:603:diff --git a/cmd/internal/launcher/datadir_test.go b/cmd/internal/launcher/datadir_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:607:+++ b/cmd/internal/launcher/datadir_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:630:diff --git a/cmd/internal/launcher/decision.go b/cmd/internal/launcher/decision.go
workshop/plans/000075-go-launcher-prototype-close-review.md:634:+++ b/cmd/internal/launcher/decision.go
workshop/plans/000075-go-launcher-prototype-close-review.md:739:diff --git a/cmd/internal/launcher/decision_test.go b/cmd/internal/launcher/decision_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:743:+++ b/cmd/internal/launcher/decision_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:812:diff --git a/cmd/internal/launcher/history.go b/cmd/internal/launcher/history.go
workshop/plans/000075-go-launcher-prototype-close-review.md:816:+++ b/cmd/internal/launcher/history.go
workshop/plans/000075-go-launcher-prototype-close-review.md:883:diff --git a/cmd/internal/launcher/history_test.go b/cmd/internal/launcher/history_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:887:+++ b/cmd/internal/launcher/history_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:898:+func TestHistorySourceScansCwdPrefixedDraftAndLogSidecars(t *testing.T) {
workshop/plans/000075-go-launcher-prototype-close-review.md:922:diff --git a/cmd/internal/launcher/run.go b/cmd/internal/launcher/run.go
workshop/plans/000075-go-launcher-prototype-close-review.md:926:+++ b/cmd/internal/launcher/run.go
workshop/plans/000075-go-launcher-prototype-close-review.md:998:diff --git a/cmd/internal/launcher/run_test.go b/cmd/internal/launcher/run_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:1002:+++ b/cmd/internal/launcher/run_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:1094:diff --git a/cmd/internal/launcher/session.go b/cmd/internal/launcher/session.go
workshop/plans/000075-go-launcher-prototype-close-review.md:1098:+++ b/cmd/internal/launcher/session.go
workshop/plans/000075-go-launcher-prototype-close-review.md:1128:diff --git a/cmd/internal/launcher/tag.go b/cmd/internal/launcher/tag.go
workshop/plans/000075-go-launcher-prototype-close-review.md:1132:+++ b/cmd/internal/launcher/tag.go
workshop/plans/000075-go-launcher-prototype-close-review.md:1176:diff --git a/cmd/internal/launcher/tag_test.go b/cmd/internal/launcher/tag_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:1180:+++ b/cmd/internal/launcher/tag_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:1235:diff --git a/cmd/internal/launcher/zellij.go b/cmd/internal/launcher/zellij.go
workshop/plans/000075-go-launcher-prototype-close-review.md:1239:+++ b/cmd/internal/launcher/zellij.go
workshop/plans/000075-go-launcher-prototype-close-review.md:1332:diff --git a/cmd/internal/launcher/zellij_test.go b/cmd/internal/launcher/zellij_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:1336:+++ b/cmd/internal/launcher/zellij_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:1503:+	if !strings.Contains(stdout.String(), "Usage: pair-go launch") {
workshop/plans/000075-go-launcher-prototype-close-review.md:1591:    "About to EnterPlanMode?" [shape=doublecircle];
workshop/plans/000075-go-launcher-prototype-close-review.md:1602:    "About to EnterPlanMode?" -> "Already brainstormed?";
workshop/plans/000075-go-launcher-prototype-close-review.md:1800:M4b's review pane added `Alt+q` visual wrapping as `🤖<selection>[]` but initially
workshop/plans/000075-go-launcher-prototype-close-review.md:1809:M4b review.
workshop/plans/000075-go-launcher-prototype-close-review.md:1820:using quotes in a branch/path/name. Caught in #000066 M4b review.
workshop/plans/000075-go-launcher-prototype-close-review.md:1826:**Rule.** When you need monotonic IDs in Lua, use an explicit counter (`local counter = 0; ... counter = counter + 1`). Do not use `#table` unless `table` is provably array-shaped (`{[1]=..., [2]=..., ...}`). The bug is silent — `nvim_set_hl` doesn't error on overwrite, it just wins-last. Filed during #000017 M4.
workshop/plans/000075-go-launcher-prototype-close-review.md:1836:pyte's `screen.buffer` is a `StaticDefaultDict` — accessing `buffer[y][x]` lazily creates a default Char, but `buffer.keys()` only contains rows that were *written to*. The renderer originally did `for y in sorted(screen.buffer.keys())`, which silently dropped trailing blank rows when the agent cleared and paused mid-redraw. That shifts every subsequent line number — directly breaking the feature's core promise that `:880` lands where zellij showed line 880. Caught in code review of #000017.
workshop/plans/000075-go-launcher-prototype-close-review.md:1896:Caught in #000027 M1 review.
workshop/plans/000075-go-launcher-prototype-close-review.md:1908:Running `gofmt -w cmd/pair-wrap/` to format M3's edited `main.go` also
workshop/plans/000075-go-launcher-prototype-close-review.md:1917:Caught in #000027 M3.
workshop/plans/000075-go-launcher-prototype-close-review.md:1921:M3's pair-wrap trigger "didn't fire" on restart. Trace: pair-slug worked in
workshop/plans/000075-go-launcher-prototype-close-review.md:1934:#000027 M3 dogfood.
workshop/plans/000075-go-launcher-prototype-close-review.md:1963:M1 review (C1).
workshop/plans/000075-go-launcher-prototype-close-review.md:1977:and zero diagnostics exactly when they needed the tool. Surfaced in #000045 M1
workshop/plans/000075-go-launcher-prototype-close-review.md:2028:## Spec
workshop/plans/000075-go-launcher-prototype-close-review.md:2032:The #73 inventory identifies `bin/pair` as the P0 public launcher surface. For this milestone, the guarded path is `pair-go launch`, a development-only launcher prototype that reaches the decision boundary but never starts or attaches a real zellij session. The public `bin/pair` shell launcher remains the only supported user entrypoint.
workshop/plans/000075-go-launcher-prototype-close-review.md:2036:- argv parsing for `pair-go launch`, including default agent, `resume <tag>`, optional agent positional, and `--` forwarded args;
workshop/plans/000075-go-launcher-prototype-close-review.md:2047:The implementation should keep business decisions pure (`ARCH-PURE`) and reuse the dispatcher introduced in #74 instead of creating a parallel command parser (`ARCH-DRY`). The slice must still satisfy the issue purpose (`ARCH-PURPOSE`): it is not enough to port helpers; `pair-go launch` has to exercise a coherent launcher decision surface.
workshop/plans/000075-go-launcher-prototype-close-review.md:2051:- [x] A guarded Go launcher path can exercise a documented subset of launcher behavior.
workshop/plans/000075-go-launcher-prototype-close-review.md:2052:- [x] Existing `bin/pair` remains the default public launcher.
workshop/plans/000075-go-launcher-prototype-close-review.md:2053:- [x] Tests cover the ported decision logic and at least one process-level fake for external commands.
workshop/plans/000075-go-launcher-prototype-close-review.md:2054:- [x] Any behavior not yet ported fails explicitly rather than silently diverging.
workshop/plans/000075-go-launcher-prototype-close-review.md:2055:- [x] Pair remains usable after merge through the existing public command.
workshop/plans/000075-go-launcher-prototype-close-review.md:2057:## Plan
workshop/plans/000075-go-launcher-prototype-close-review.md:2059:- [x] Select the launcher slice from #73.
workshop/plans/000075-go-launcher-prototype-close-review.md:2060:- [x] Extract pure decision models and tests.
workshop/plans/000075-go-launcher-prototype-close-review.md:2061:- [x] Add fake-command process tests for the selected slice.
workshop/plans/000075-go-launcher-prototype-close-review.md:2062:- [x] Implement the guarded Go path.
workshop/plans/000075-go-launcher-prototype-close-review.md:2063:- [x] Document remaining shell-owned launcher behavior.
workshop/plans/000075-go-launcher-prototype-close-review.md:2081:## Log
workshop/plans/000075-go-launcher-prototype-close-review.md:2088:- 2026-06-29: closed — go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; git diff -- bin/pair empty; rg atlas boundary check; git diff --check
workshop/plans/000075-go-launcher-prototype-close-review.md:2090:Claimed #75 after parking #82. Entered planning with `sdlc start-plan --issue 75`; design cites #73's inventory and chooses a guarded `pair-go launch` decision-phase prototype so the shell launcher remains public while the Go path exercises real launcher concepts (`ARCH-PURE`, `ARCH-DRY`, `ARCH-PURPOSE`).
workshop/plans/000075-go-launcher-prototype-close-review.md:2092:Plan-quality gate returned FAILURE: estimate was low for the visible multi-surface scope, and tag/session naming was ambiguous. Revised the estimate to 3.75 and clarified that canonical tags are bare while zellij session names are derived as `pair-<tag>`.
workshop/plans/000075-go-launcher-prototype-close-review.md:2098:Implemented `cmd/internal/launcher` as a pure decision-phase core plus fakeable zellij/history seams. `pair-go launch` now routes through the #74 dispatcher, returns explicit prototype decisions, and does not mutate zellij or replace `bin/pair`. Updated atlas architecture and the Go migration inventory to record the shell-owned boundary.
workshop/plans/000075-go-launcher-prototype-close-review.md:2107: cmd/internal/launcher/args.go                   |  71 ++++++++++++++++
workshop/plans/000075-go-launcher-prototype-close-review.md:2108: cmd/internal/launcher/args_test.go              |  88 ++++++++++++++++++++
workshop/plans/000075-go-launcher-prototype-close-review.md:2109: cmd/internal/launcher/datadir.go                |  11 +++
workshop/plans/000075-go-launcher-prototype-close-review.md:2110: cmd/internal/launcher/datadir_test.go           |  21 +++++
workshop/plans/000075-go-launcher-prototype-close-review.md:2111: cmd/internal/launcher/decision.go               | 103 ++++++++++++++++++++++++
workshop/plans/000075-go-launcher-prototype-close-review.md:2112: cmd/internal/launcher/decision_test.go          |  67 +++++++++++++++
workshop/plans/000075-go-launcher-prototype-close-review.md:2113: cmd/internal/launcher/history.go                |  65 +++++++++++++++
workshop/plans/000075-go-launcher-prototype-close-review.md:2114: cmd/internal/launcher/history_test.go           |  33 ++++++++
workshop/plans/000075-go-launcher-prototype-close-review.md:2115: cmd/internal/launcher/run.go                    |  70 ++++++++++++++++
workshop/plans/000075-go-launcher-prototype-close-review.md:2116: cmd/internal/launcher/run_test.go               |  90 +++++++++++++++++++++
workshop/plans/000075-go-launcher-prototype-close-review.md:2117: cmd/internal/launcher/session.go                |  28 +++++++
workshop/plans/000075-go-launcher-prototype-close-review.md:2118: cmd/internal/launcher/tag.go                    |  42 ++++++++++
workshop/plans/000075-go-launcher-prototype-close-review.md:2119: cmd/internal/launcher/tag_test.go               |  53 ++++++++++++
workshop/plans/000075-go-launcher-prototype-close-review.md:2120: cmd/internal/launcher/zellij.go                 |  91 +++++++++++++++++++++
workshop/plans/000075-go-launcher-prototype-close-review.md:2121: cmd/internal/launcher/zellij_test.go            |  44 ++++++++++
workshop/plans/000075-go-launcher-prototype-close-review.md:2135:cmd/internal/launcher/args.go
workshop/plans/000075-go-launcher-prototype-close-review.md:2136:cmd/internal/launcher/args_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:2137:cmd/internal/launcher/datadir.go
workshop/plans/000075-go-launcher-prototype-close-review.md:2138:cmd/internal/launcher/datadir_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:2139:cmd/internal/launcher/decision.go
workshop/plans/000075-go-launcher-prototype-close-review.md:2140:cmd/internal/launcher/decision_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:2141:cmd/internal/launcher/history.go
workshop/plans/000075-go-launcher-prototype-close-review.md:2142:cmd/internal/launcher/history_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:2143:cmd/internal/launcher/run.go
workshop/plans/000075-go-launcher-prototype-close-review.md:2144:cmd/internal/launcher/run_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:2145:cmd/internal/launcher/session.go
workshop/plans/000075-go-launcher-prototype-close-review.md:2146:cmd/internal/launcher/tag.go
workshop/plans/000075-go-launcher-prototype-close-review.md:2147:cmd/internal/launcher/tag_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:2148:cmd/internal/launcher/zellij.go
workshop/plans/000075-go-launcher-prototype-close-review.md:2149:cmd/internal/launcher/zellij_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:2475:        # or resurrectable). The inside-session rename gesture (M2) takes
workshop/plans/000075-go-launcher-prototype-close-review.md:2676:        # leaves a forensic breadcrumb for M3's recovery path.
workshop/plans/000075-go-launcher-prototype-close-review.md:2727:/bin/zsh -lc 'nl -ba cmd/internal/launcher/zellij.go' in /Users/xianxu/workspace/pair
workshop/plans/000075-go-launcher-prototype-close-review.md:2729:/bin/zsh -lc 'nl -ba cmd/internal/launcher/decision.go' in /Users/xianxu/workspace/pair
workshop/plans/000075-go-launcher-prototype-close-review.md:2929:/bin/zsh -lc 'nl -ba cmd/internal/launcher/history.go' in /Users/xianxu/workspace/pair
workshop/plans/000075-go-launcher-prototype-close-review.md:2998:/bin/zsh -lc 'nl -ba cmd/internal/launcher/run.go' in /Users/xianxu/workspace/pair
workshop/plans/000075-go-launcher-prototype-close-review.md:3072:/bin/zsh -lc 'nl -ba cmd/internal/launcher/args.go' in /Users/xianxu/workspace/pair
workshop/plans/000075-go-launcher-prototype-close-review.md:3078:     5	// LaunchArgs is the pure parse result for the guarded pair-go launch prototype.
workshop/plans/000075-go-launcher-prototype-close-review.md:3095:    22	// ParseArgs parses pair-go launch args. It intentionally supports only the
workshop/plans/000075-go-launcher-prototype-close-review.md:3107:    34			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %s is not implemented by pair-go launch; use bin/pair", argv[0])}
workshop/plans/000075-go-launcher-prototype-close-review.md:3110:    37				return LaunchArgs{}, UsageError{Message: "pair-go launch: 'resume' requires a tag"}
workshop/plans/000075-go-launcher-prototype-close-review.md:3114:    41				return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: invalid tag: %v", err)}
workshop/plans/000075-go-launcher-prototype-close-review.md:3117:    44				return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", argv[2])}
workshop/plans/000075-go-launcher-prototype-close-review.md:3137:    64			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", arg)}
workshop/plans/000075-go-launcher-prototype-close-review.md:3296:738:    # Inside-flow tag rename (#000022 M2). By this point the zellij kill
workshop/plans/000075-go-launcher-prototype-close-review.md:3634:/bin/zsh -lc 'nl -ba cmd/internal/launcher/tag.go' in /Users/xianxu/workspace/pair
workshop/plans/000075-go-launcher-prototype-close-review.md:3688:     9		"github.com/xianxu/pair/cmd/internal/launcher"
workshop/plans/000075-go-launcher-prototype-close-review.md:3792:   113			return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
workshop/plans/000075-go-launcher-prototype-close-review.md:3797:   118				"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
workshop/plans/000075-go-launcher-prototype-close-review.md:3844:   165		b.WriteString("Planned command families (not implemented in this skeleton):\n")
workshop/plans/000075-go-launcher-prototype-close-review.md:4667:# Pair Go Launcher Prototype Implementation Plan
workshop/plans/000075-go-launcher-prototype-close-review.md:4669:> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.
workshop/plans/000075-go-launcher-prototype-close-review.md:4671:**Goal:** Add a guarded `pair-go launch` prototype that reaches Pair launcher decision logic without replacing or invoking the public shell launcher.
workshop/plans/000075-go-launcher-prototype-close-review.md:4673:**Architecture:** Put launcher business logic in a pure `cmd/internal/launcher` package and keep subprocess/filesystem work in thin, injectable seams (`ARCH-PURE`). Extend the existing #74 dispatcher instead of adding a parallel command parser (`ARCH-DRY`). The prototype prints the decision it would take, then exits with an explicit unsupported-after-decision code so the issue delivers a real launcher vertical slice without changing public `bin/pair` behavior (`ARCH-PURPOSE`).
workshop/plans/000075-go-launcher-prototype-close-review.md:4685:| `LaunchArgs` | `cmd/internal/launcher/args.go` | new |
workshop/plans/000075-go-launcher-prototype-close-review.md:4686:| `Tag` | `cmd/internal/launcher/tag.go` | new |
workshop/plans/000075-go-launcher-prototype-close-review.md:4687:| `DataDir` | `cmd/internal/launcher/datadir.go` | new |
workshop/plans/000075-go-launcher-prototype-close-review.md:4688:| `SessionSnapshot` | `cmd/internal/launcher/session.go` | new |
workshop/plans/000075-go-launcher-prototype-close-review.md:4689:| `LaunchDecision` | `cmd/internal/launcher/decision.go` | new |
workshop/plans/000075-go-launcher-prototype-close-review.md:4692:- **Relationships:** 1:1 with a `pair-go launch` invocation; owns a `Tag` only for forced resume.
workshop/plans/000075-go-launcher-prototype-close-review.md:4693:- **DRY rationale:** Keeps `pair-go launch` parsing behind the existing dispatcher instead of duplicating parser branches in `cmd/pair-go/main.go`.
workshop/plans/000075-go-launcher-prototype-close-review.md:4722:| `ZellijSource` | `cmd/internal/launcher/zellij.go` | new | `zellij list-sessions`, `zellij --session ... action list-clients` |
workshop/plans/000075-go-launcher-prototype-close-review.md:4723:| `HistorySource` | `cmd/internal/launcher/history.go` | new | filesystem scan of `draft-*.md`, `log-*.md`, `queue-*` under Pair data dir |
workshop/plans/000075-go-launcher-prototype-close-review.md:4735:**LaunchCommand** — Routes `pair-go launch` from the existing dispatcher to the launcher runner.
workshop/plans/000075-go-launcher-prototype-close-review.md:4746:### Task 1: Parse `pair-go launch` Args
workshop/plans/000075-go-launcher-prototype-close-review.md:4749:- Create: `cmd/internal/launcher/args.go`
workshop/plans/000075-go-launcher-prototype-close-review.md:4750:- Create: `cmd/internal/launcher/args_test.go`
workshop/plans/000075-go-launcher-prototype-close-review.md:4752:- [ ] **Step 1: Write failing parse tests**
workshop/plans/000075-go-launcher-prototype-close-review.md:4762:Run: `go test ./cmd/internal/launcher -run 'TestParseLaunchArgs' -count=1`
workshop/plans/000075-go-launcher-prototype-close-review.md:4765:- [ ] **Step 2: Implement minimal parser**
workshop/plans/000075-go-launcher-prototype-close-review.md:4769:- [ ] **Step 3: Verify parse tests pass**
workshop/plans/000075-go-launcher-prototype-close-review.md:4771:Run: `go test ./cmd/internal/launcher -run 'TestParseLaunchArgs' -count=1`
workshop/plans/000075-go-launcher-prototype-close-review.md:4777:- Create: `cmd/internal/launcher/tag.go`
workshop/plans/000075-go-launcher-prototype-close-review.md:4778:- Create: `cmd/internal/launcher/tag_test.go`
workshop/plans/000075-go-launcher-prototype-close-review.md:4779:- Create: `cmd/internal/launcher/datadir.go`
workshop/plans/000075-go-launcher-prototype-close-review.md:4780:- Create: `cmd/internal/launcher/datadir_test.go`
workshop/plans/000075-go-launcher-prototype-close-review.md:4782:- [ ] **Step 1: Write failing helper tests**
workshop/plans/000075-go-launcher-prototype-close-review.md:4791:Run: `go test ./cmd/internal/launcher -run 'TestNormalizeTag|TestDefaultTag|TestResolveDataDir' -count=1`
workshop/plans/000075-go-launcher-prototype-close-review.md:4794:- [ ] **Step 2: Implement helpers**
workshop/plans/000075-go-launcher-prototype-close-review.md:4798:- [ ] **Step 3: Verify helper tests pass**
workshop/plans/000075-go-launcher-prototype-close-review.md:4800:Run: `go test ./cmd/internal/launcher -run 'TestNormalizeTag|TestDefaultTag|TestResolveDataDir' -count=1`
workshop/plans/000075-go-launcher-prototype-close-review.md:4806:- Create: `cmd/internal/launcher/session.go`
workshop/plans/000075-go-launcher-prototype-close-review.md:4807:- Create: `cmd/internal/launcher/decision.go`
workshop/plans/000075-go-launcher-prototype-close-review.md:4808:- Create: `cmd/internal/launcher/decision_test.go`
workshop/plans/000075-go-launcher-prototype-close-review.md:4810:- [ ] **Step 1: Write failing decision tests**
workshop/plans/000075-go-launcher-prototype-close-review.md:4821:Run: `go test ./cmd/internal/launcher -run 'TestDecideLaunch' -count=1`
workshop/plans/000075-go-launcher-prototype-close-review.md:4824:- [ ] **Step 2: Implement models**
workshop/plans/000075-go-launcher-prototype-close-review.md:4848:- [ ] **Step 3: Verify decision tests pass**
workshop/plans/000075-go-launcher-prototype-close-review.md:4850:Run: `go test ./cmd/internal/launcher -run 'TestDecideLaunch' -count=1`
workshop/plans/000075-go-launcher-prototype-close-review.md:4853:- [ ] **Step 4: Commit pure core**
workshop/plans/000075-go-launcher-prototype-close-review.md:4856:git add cmd/internal/launcher
workshop/plans/000075-go-launcher-prototype-close-review.md:4865:- Create: `cmd/internal/launcher/zellij.go`
workshop/plans/000075-go-launcher-prototype-close-review.md:4866:- Create: `cmd/internal/launcher/history.go`
workshop/plans/000075-go-launcher-prototype-close-review.md:4867:- Create: `cmd/internal/launcher/run.go`
workshop/plans/000075-go-launcher-prototype-close-review.md:4868:- Create: `cmd/internal/launcher/run_test.go`
workshop/plans/000075-go-launcher-prototype-close-review.md:4870:- [ ] **Step 1: Write failing runner tests with in-memory fakes**
workshop/plans/000075-go-launcher-prototype-close-review.md:4878:Run: `go test ./cmd/internal/launcher -run 'TestRunLaunch' -count=1`
workshop/plans/000075-go-launcher-prototype-close-review.md:4881:- [ ] **Step 2: Implement interfaces and runner**
workshop/plans/000075-go-launcher-prototype-close-review.md:4906:- [ ] **Step 3: Verify runner tests pass**
workshop/plans/000075-go-launcher-prototype-close-review.md:4908:Run: `go test ./cmd/internal/launcher -run 'TestRunLaunch' -count=1`
workshop/plans/000075-go-launcher-prototype-close-review.md:4911:### Task 5: Route `pair-go launch`
workshop/plans/000075-go-launcher-prototype-close-review.md:4919:- [ ] **Step 1: Update dispatcher tests**
workshop/plans/000075-go-launcher-prototype-close-review.md:4926:- [ ] **Step 2: Implement the route**
workshop/plans/000075-go-launcher-prototype-close-review.md:4930:- [ ] **Step 3: Update `pair-go` tests**
workshop/plans/000075-go-launcher-prototype-close-review.md:4934:- `pair-go launch --help` prints launch usage;
workshop/plans/000075-go-launcher-prototype-close-review.md:4935:- `pair-go launch resume demo` returns a prototype decision message, not a real launch.
workshop/plans/000075-go-launcher-prototype-close-review.md:4940:- [ ] **Step 4: Commit IO and route**
workshop/plans/000075-go-launcher-prototype-close-review.md:4943:git add cmd/internal/launcher cmd/internal/dispatcher cmd/pair-go
workshop/plans/000075-go-launcher-prototype-close-review.md:4944:git commit -m "#75: add guarded Go launcher prototype" -m "Route pair-go launch through a fakeable launcher runner that reaches create/attach decisions without replacing bin/pair." -m "Co-Authored-By: GPT-5 Codex <codex@openai.com>"
workshop/plans/000075-go-launcher-prototype-close-review.md:4960:- [ ] **Step 1: Write failing process-level fake test**
workshop/plans/000075-go-launcher-prototype-close-review.md:4978:- [ ] **Step 2: Make the dispatcher/cmd path support injected launcher runtime for tests**
workshop/plans/000075-go-launcher-prototype-close-review.md:4982:- [ ] **Step 3: Verify process test passes**
workshop/plans/000075-go-launcher-prototype-close-review.md:4995:- [ ] **Step 1: Update atlas**
workshop/plans/000075-go-launcher-prototype-close-review.md:4997:Record that `pair-go launch` now owns only the decision-phase prototype. Explicitly list shell-owned behavior still out of scope: real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller.
workshop/plans/000075-go-launcher-prototype-close-review.md:4999:- [ ] **Step 2: Verify docs mention the boundary**
workshop/plans/000075-go-launcher-prototype-close-review.md:5001:Run: `rg -n "pair-go launch|decision-phase|shell-owned" atlas/architecture.md atlas/go-migration-inventory.md`
workshop/plans/000075-go-launcher-prototype-close-review.md:5004:- [ ] **Step 3: Commit docs**
workshop/plans/000075-go-launcher-prototype-close-review.md:5008:git commit -m "#75: document launcher prototype boundary" -m "Clarify that pair-go launch is a guarded decision-phase prototype while bin/pair remains the public launcher." -m "Co-Authored-By: GPT-5 Codex <codex@openai.com>"
workshop/plans/000075-go-launcher-prototype-close-review.md:5016:- [ ] **Step 1: Run focused Go tests**
workshop/plans/000075-go-launcher-prototype-close-review.md:5018:Run: `go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
workshop/plans/000075-go-launcher-prototype-close-review.md:5021:- [ ] **Step 2: Build `pair-go`**
workshop/plans/000075-go-launcher-prototype-close-review.md:5026:- [ ] **Step 3: Run full Go suite**
workshop/plans/000075-go-launcher-prototype-close-review.md:5031:- [ ] **Step 4: Verify public launcher unchanged**
workshop/plans/000075-go-launcher-prototype-close-review.md:5036:- [ ] **Step 5: Verify docs and whitespace**
workshop/plans/000075-go-launcher-prototype-close-review.md:5038:Run: `rg -n "pair-go launch|decision-phase|shell-owned" atlas/architecture.md atlas/go-migration-inventory.md && git diff --check`
workshop/plans/000075-go-launcher-prototype-close-review.md:5041:- [ ] **Step 6: Close through SDLC**
workshop/plans/000075-go-launcher-prototype-close-review.md:5043:Run: `sdlc close --issue 75 --verified 'go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; git diff -- bin/pair empty; rg atlas boundary check; git diff --check'`
workshop/plans/000075-go-launcher-prototype-close-review.md:5057:-- [ ] A guarded Go launcher path can exercise a documented subset of launcher behavior.
workshop/plans/000075-go-launcher-prototype-close-review.md:5058:-- [ ] Existing `bin/pair` remains the default public launcher.
workshop/plans/000075-go-launcher-prototype-close-review.md:5059:-- [ ] Tests cover the ported decision logic and at least one process-level fake for external commands.
workshop/plans/000075-go-launcher-prototype-close-review.md:5060:-- [ ] Any behavior not yet ported fails explicitly rather than silently diverging.
workshop/plans/000075-go-launcher-prototype-close-review.md:5061:-- [ ] Pair remains usable after merge through the existing public command.
workshop/plans/000075-go-launcher-prototype-close-review.md:5062:+- [x] A guarded Go launcher path can exercise a documented subset of launcher behavior.
workshop/plans/000075-go-launcher-prototype-close-review.md:5063:+- [x] Existing `bin/pair` remains the default public launcher.
workshop/plans/000075-go-launcher-prototype-close-review.md:5064:+- [x] Tests cover the ported decision logic and at least one process-level fake for external commands.
workshop/plans/000075-go-launcher-prototype-close-review.md:5065:+- [x] Any behavior not yet ported fails explicitly rather than silently diverging.
workshop/plans/000075-go-launcher-prototype-close-review.md:5066:+- [x] Pair remains usable after merge through the existing public command.
workshop/plans/000075-go-launcher-prototype-close-review.md:5068: ## Plan
workshop/plans/000075-go-launcher-prototype-close-review.md:5070:-- [ ] Select the launcher slice from #73.
workshop/plans/000075-go-launcher-prototype-close-review.md:5071:-- [ ] Extract pure decision models and tests.
workshop/plans/000075-go-launcher-prototype-close-review.md:5072:-- [ ] Add fake-command process tests for the selected slice.
workshop/plans/000075-go-launcher-prototype-close-review.md:5073:-- [ ] Implement the guarded Go path.
workshop/plans/000075-go-launcher-prototype-close-review.md:5074:-- [ ] Document remaining shell-owned launcher behavior.
workshop/plans/000075-go-launcher-prototype-close-review.md:5075:+- [x] Select the launcher slice from #73.
workshop/plans/000075-go-launcher-prototype-close-review.md:5076:+- [x] Extract pure decision models and tests.
workshop/plans/000075-go-launcher-prototype-close-review.md:5077:+- [x] Add fake-command process tests for the selected slice.
workshop/plans/000075-go-launcher-prototype-close-review.md:5078:+- [x] Implement the guarded Go path.
workshop/plans/000075-go-launcher-prototype-close-review.md:5079:+- [x] Document remaining shell-owned launcher behavior.
workshop/plans/000075-go-launcher-prototype-close-review.md:5083:@@ -83,3 +83,5 @@ Plan-quality gate returned FAILURE: estimate was low for the visible multi-surfa
workshop/plans/000075-go-launcher-prototype-close-review.md:5088:+Implemented `cmd/internal/launcher` as a pure decision-phase core plus fakeable zellij/history seams. `pair-go launch` now routes through the #74 dispatcher, returns explicit prototype decisions, and does not mutate zellij or replace `bin/pair`. Updated atlas architecture and the Go migration inventory to record the shell-owned boundary.
workshop/plans/000075-go-launcher-prototype-close-review.md:5091:/bin/zsh -lc "git diff e89e2cea484ab8ca7ec9ca518fd2bf04ba38a8fe..HEAD -- cmd/internal/launcher cmd/internal/dispatcher cmd/pair-go | sed -n '1,260p'" in /Users/xianxu/workspace/pair
workshop/plans/000075-go-launcher-prototype-close-review.md:5105:+	"github.com/xianxu/pair/cmd/internal/launcher"
workshop/plans/000075-go-launcher-prototype-close-review.md:5162:+		return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
workshop/plans/000075-go-launcher-prototype-close-review.md:5167:+			"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
workshop/plans/000075-go-launcher-prototype-close-review.md:5216:@@ -60,6 +60,42 @@ func TestDispatchPlannedCommandReturnsUnsupported(t *testing.T) {
workshop/plans/000075-go-launcher-prototype-close-review.md:5228:+	for _, want := range []string{"Usage: pair-go launch", "decision-phase prototype"} {
workshop/plans/000075-go-launcher-prototype-close-review.md:5249:+	for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
workshop/plans/000075-go-launcher-prototype-close-review.md:5259:diff --git a/cmd/internal/launcher/args.go b/cmd/internal/launcher/args.go
workshop/plans/000075-go-launcher-prototype-close-review.md:5263:+++ b/cmd/internal/launcher/args.go
workshop/plans/000075-go-launcher-prototype-close-review.md:5269:+// LaunchArgs is the pure parse result for the guarded pair-go launch prototype.
workshop/plans/000075-go-launcher-prototype-close-review.md:5286:+// ParseArgs parses pair-go launch args. It intentionally supports only the
workshop/plans/000075-go-launcher-prototype-close-review.md:5298:+		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %s is not implemented by pair-go launch; use bin/pair", argv[0])}
workshop/plans/000075-go-launcher-prototype-close-review.md:5301:+			return LaunchArgs{}, UsageError{Message: "pair-go launch: 'resume' requires a tag"}
workshop/plans/000075-go-launcher-prototype-close-review.md:5305:+			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: invalid tag: %v", err)}
workshop/plans/000075-go-launcher-prototype-close-review.md:5308:+			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", argv[2])}
workshop/plans/000075-go-launcher-prototype-close-review.md:5328:+		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", arg)}
workshop/plans/000075-go-launcher-prototype-close-review.md:5336:diff --git a/cmd/internal/launcher/args_test.go b/cmd/internal/launcher/args_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:5340:+++ b/cmd/internal/launcher/args_test.go
workshop/plans/000075-go-launcher-prototype-close-review.md:5360:/bin/zsh -lc 'go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1' in /Users/xianxu/workspace/pair
workshop/plans/000075-go-launcher-prototype-close-review.md:5362:ok  	github.com/xianxu/pair/cmd/internal/launcher	0.670s
workshop/plans/000075-go-launcher-prototype-close-review.md:5394:+- 2026-06-29: closed — go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; git diff -- bin/pair empty; rg atlas boundary check; git diff --check
workshop/plans/000075-go-launcher-prototype-close-review.md:5396: Claimed #75 after parking #82. Entered planning with `sdlc start-plan --issue 75`; design cites #73's inventory and chooses a guarded `pair-go launch` decision-phase prototype so the shell launcher remains public while the Go path exercises real launcher concepts (`ARCH-PURE`, `ARCH-DRY`, `ARCH-PURPOSE`).
workshop/plans/000075-go-launcher-prototype-close-review.md:5400:/bin/zsh -lc 'rg -n "queue|HistorySource|draft|log|sidecar|SelectedTag|Picker|pick|sessionBlocksReuse|exited|EXITED" cmd/internal/launcher cmd/internal/dispatcher cmd/pair-go workshop/issues/000075-go-launcher-prototype.md workshop/plans/000075-go-launcher-prototype-plan.md' in /Users/xianxu/workspace/pair
workshop/plans/000075-go-launcher-prototype-close-review.md:5406:workshop/issues/000075-go-launcher-prototype.md:44:- [x] Tests cover the ported decision logic and at least one process-level fake for external commands.
workshop/plans/000075-go-launcher-prototype-close-review.md:5409:workshop/plans/000075-go-launcher-prototype-plan.md:5:**Goal:** Add a guarded `pair-go launch` prototype that reaches Pair launcher decision logic without replacing or invoking the public shell launcher.
workshop/plans/000075-go-launcher-prototype-close-review.md:5410:workshop/plans/000075-go-launcher-prototype-plan.md:7:**Architecture:** Put launcher business logic in a pure `cmd/internal/launcher` package and keep subprocess/filesystem work in thin, injectable seams (`ARCH-PURE`). Extend the existing #74 dispatcher instead of adding a parallel command parser (`ARCH-DRY`). The prototype prints the decision it would take, then exits with an explicit unsupported-after-decision code so the issue delivers a real launcher vertical slice without changing public `bin/pair` behavior (`ARCH-PURPOSE`).
workshop/plans/000075-go-launcher-prototype-close-review.md:5414:workshop/plans/000075-go-launcher-prototype-plan.md:57:| `HistorySource` | `cmd/internal/launcher/history.go` | new | filesystem scan of `draft-*.md`, `log-*.md`, `queue-*` under Pair data dir |
workshop/plans/000075-go-launcher-prototype-close-review.md:5442:cmd/internal/launcher/zellij_test.go:11:	log := filepath.Join(dir, "zellij.log")
workshop/plans/000075-go-launcher-prototype-close-review.md:5443:cmd/internal/launcher/zellij_test.go:14:printf '%s\n' "$*" >> "` + log + `"
workshop/plans/000075-go-launcher-prototype-close-review.md:5444:cmd/internal/launcher/zellij_test.go:17:  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-gone [Created] (EXITED - attach to resurrect)\n' ;;
workshop/plans/000075-go-launcher-prototype-close-review.md:5445:cmd/internal/launcher/decision_test.go:41:func TestDecideLaunchShowsPickerWhenDetachedOrHistoricalExist(t *testing.T) {
workshop/plans/000075-go-launcher-prototype-close-review.md:5446:cmd/internal/launcher/decision_test.go:51:			t.Fatalf("decision = %#v, want picker", decision)
workshop/plans/000075-go-launcher-prototype-close-review.md:5447:cmd/internal/launcher/decision_test.go:57:	decision, err := DecideLaunch(LaunchArgs{Agent: "claude", SelectedTag: "pair-old"}, SessionSnapshot{
workshop/plans/000075-go-launcher-prototype-close-review.md:5448:cmd/internal/launcher/run_test.go:51:func TestRunLaunchTurnsFakeSessionsIntoPickerDecision(t *testing.T) {
workshop/plans/000075-go-launcher-prototype-close-review.md:5449:cmd/internal/launcher/run_test.go:62:		t.Fatalf("Decision = %#v, want picker", outcome.Decision)
workshop/plans/000075-go-launcher-prototype-close-review.md:5450:cmd/internal/launcher/run_test.go:66:func TestRunLaunchTurnsFakeHistoryIntoPickerDecision(t *testing.T) {
workshop/plans/000075-go-launcher-prototype-close-review.md:5451:cmd/internal/launcher/run_test.go:77:		t.Fatalf("Decision = %#v, want picker", outcome.Decision)
workshop/plans/000075-go-launcher-prototype-close-review.md:5452:cmd/internal/launcher/args.go:9:	SelectedTag string
workshop/plans/000075-go-launcher-prototype-close-review.md:5453:cmd/internal/launcher/session.go:9:	SessionExited   SessionState = "exited"
workshop/plans/000075-go-launcher-prototype-close-review.md:5454:cmd/internal/launcher/history.go:11:// HistorySource scans Pair draft/log sidecars under the data directory.
workshop/plans/000075-go-launcher-prototype-close-review.md:5455:cmd/internal/launcher/history.go:12:type HistorySource struct {
workshop/plans/000075-go-launcher-prototype-close-review.md:5456:cmd/internal/launcher/history.go:16:func (s HistorySource) Scan(base string, cutoff time.Time) ([]HistoricalTag, error) {
workshop/plans/000075-go-launcher-prototype-close-review.md:5457:cmd/internal/launcher/history.go:18:	for _, pattern := range []string{"draft-*.md", "log-*.md"} {
workshop/plans/000075-go-launcher-prototype-close-review.md:5458:cmd/internal/launcher/history.go:54:	case strings.HasPrefix(name, "draft-") && strings.HasSuffix(name, ".md"):
workshop/plans/000075-go-launcher-prototype-close-review.md:5459:cmd/internal/launcher/history.go:55:		return strings.TrimSuffix(strings.TrimPrefix(name, "draft-"), ".md"), true
workshop/plans/000075-go-launcher-prototype-close-review.md:5460:cmd/internal/launcher/history.go:56:	case strings.HasPrefix(name, "log-") && strings.HasSuffix(name, ".md"):
workshop/plans/000075-go-launcher-prototype-close-review.md:5461:cmd/internal/launcher/history.go:57:		return strings.TrimSuffix(strings.TrimPrefix(name, "log-"), ".md"), true
workshop/plans/000075-go-launcher-prototype-close-review.md:5462:cmd/internal/launcher/history_test.go:10:func TestHistorySourceScansCwdPrefixedDraftAndLogSidecars(t *testing.T) {
workshop/plans/000075-go-launcher-prototype-close-review.md:5463:cmd/internal/launcher/history_test.go:13:	for _, name := range []string{"draft-pair.md", "log-pair-old.md", "draft-other.md"} {
workshop/plans/000075-go-launcher-prototype-close-review.md:5464:cmd/internal/launcher/history_test.go:23:	got, err := HistorySource{DataDir: dir}.Scan("pair", now.Add(-24*time.Hour))
workshop/plans/000075-go-launcher-prototype-close-review.md:5465:cmd/internal/launcher/zellij.go:24:	exited := exitedSessions(string(raw))
workshop/plans/000075-go-launcher-prototype-close-review.md:5466:cmd/internal/launcher/zellij.go:31:		if exited[name] {
workshop/plans/000075-go-launcher-prototype-close-review.md:5467:cmd/internal/launcher/zellij.go:68:func exitedSessions(raw string) map[string]bool {
workshop/plans/000075-go-launcher-prototype-close-review.md:5468:cmd/internal/launcher/zellij.go:75:		if strings.Contains(line, "EXITED") {
workshop/plans/000075-go-launcher-prototype-close-review.md:5469:cmd/internal/launcher/decision.go:9:	ActionPick   LaunchAction = "pick"
workshop/plans/000075-go-launcher-prototype-close-review.md:5470:cmd/internal/launcher/decision.go:12:// LaunchDecision is a pure create/attach/pick decision. Tag is canonical bare
workshop/plans/000075-go-launcher-prototype-close-review.md:5471:cmd/internal/launcher/decision.go:23:	if args.SelectedTag != "" {
workshop/plans/000075-go-launcher-prototype-close-review.md:5472:cmd/internal/launcher/decision.go:24:		return createDecision(args.SelectedTag, false), nil
workshop/plans/000075-go-launcher-prototype-close-review.md:5473:cmd/internal/launcher/decision.go:27:		if sessionBlocksReuse(snap, sessionName(args.ForcedTag)) {
workshop/plans/000075-go-launcher-prototype-close-review.md:5474:cmd/internal/launcher/decision.go:59:func sessionBlocksReuse(snap SessionSnapshot, name string) bool {
workshop/plans/000075-go-launcher-prototype-close-review.md:5475:cmd/internal/launcher/decision.go:75:		if !sessionBlocksReuse(snap, sessionName(tag)) && !isHistorical(snap, tag) {
workshop/plans/000075-go-launcher-prototype-close-review.md:5486:ok  	github.com/xianxu/pair/cmd/internal/launcher	2.215s
workshop/plans/000075-go-launcher-prototype-close-review.md:5503:    23	The #73 inventory identifies `bin/pair` as the P0 public launcher surface. For this milestone, the guarded path is `pair-go launch`, a development-only launcher prototype that reaches the decision boundary but never starts or attaches a real zellij session. The public `bin/pair` shell launcher remains the only supported user entrypoint.
workshop/plans/000075-go-launcher-prototype-close-review.md:5507:    27	- argv parsing for `pair-go launch`, including default agent, `resume <tag>`, optional agent positional, and `--` forwarded args;
workshop/plans/000075-go-launcher-prototype-close-review.md:5529:    56	| `ZellijSource` | `cmd/internal/launcher/zellij.go` | new | `zellij list-sessions`, `zellij --session ... action list-clients` |
workshop/plans/000075-go-launcher-prototype-close-review.md:5530:    57	| `HistorySource` | `cmd/internal/launcher/history.go` | new | filesystem scan of `draft-*.md`, `log-*.md`, `queue-*` under Pair data dir |
workshop/plans/000075-go-launcher-prototype-close-review.md:5542:    69	**LaunchCommand** — Routes `pair-go launch` from the existing dispatcher to the launcher runner.
workshop/plans/000075-go-launcher-prototype-close-review.md:5563:   113			return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
workshop/plans/000075-go-launcher-prototype-close-review.md:5569:/bin/zsh -lc "nl -ba cmd/internal/launcher/history.go | sed -n '11,24p'" in /Users/xianxu/workspace/pair
workshop/plans/000075-go-launcher-prototype-close-review.md:5587:/bin/zsh -lc "nl -ba cmd/internal/launcher/args.go | sed -n '25,31p'" in /Users/xianxu/workspace/pair
workshop/plans/000075-go-launcher-prototype-close-review.md:5601:/bin/zsh -lc 'go run ./cmd/pair-go launch' in /Users/xianxu/workspace/pair
workshop/plans/000075-go-launcher-prototype-close-review.md:5603:/bin/zsh -lc 'go run ./cmd/pair-go launch --help' in /Users/xianxu/workspace/pair
workshop/plans/000075-go-launcher-prototype-close-review.md:5605:Usage: pair-go launch [agent] [-- agent-args...]
workshop/plans/000075-go-launcher-prototype-close-review.md:5606:       pair-go launch resume <tag>
workshop/plans/000075-go-launcher-prototype-close-review.md:5613:Usage: pair-go launch [agent] [-- agent-args...]
workshop/plans/000075-go-launcher-prototype-close-review.md:5614:       pair-go launch resume <tag>
workshop/plans/000075-go-launcher-prototype-close-review.md:5628:   165		b.WriteString("Planned command families (not implemented in this skeleton):\n")
workshop/plans/000075-go-launcher-prototype-close-review.md:5640:The implementation has a solid pure launcher core and fakeable IO seams, and the public `bin/pair` path is untouched. The blocker is that the actual guarded command path does not exercise the default-agent launch slice promised by the issue: `pair-go launch` returns help instead of reaching the decision boundary.
workshop/plans/000075-go-launcher-prototype-close-review.md:5643:- `cmd/internal/launcher/decision.go:21` keeps create/attach/pick logic pure and unit-testable (`ARCH-PURE` pass).
workshop/plans/000075-go-launcher-prototype-close-review.md:5649:- `cmd/internal/dispatcher/dispatcher.go:108`: `pair-go launch` with no launch args returns help. The issue explicitly requires argv parsing “including default agent” and a guarded path that reaches the decision boundary (`workshop/issues/000075-go-launcher-prototype.md:27`). `ParseArgs(nil)` supports `claude` (`cmd/internal/launcher/args.go:27`), but the dispatcher bypasses it. Fix: only treat explicit `help|--help|-h` as help, let empty `launchArgs` call `launcher.Run`, and add dispatcher/process tests for `pair-go launch` returning a prototype decision. `ARCH-PURPOSE` flag.
workshop/plans/000075-go-launcher-prototype-close-review.md:5650:- `workshop/plans/000075-go-launcher-prototype-plan.md:57`: the integration table says `HistorySource` wraps `queue-*`, but implementation scans only `draft-*.md` and `log-*.md` (`cmd/internal/launcher/history.go:18`). Either implement the queued-surface scan or revise the plan with a `## Revisions` entry clarifying queue badges are future scope. As written, the core concepts table contradicts the code.
workshop/plans/000075-go-launcher-prototype-close-review.md:5659:- Ran `go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1`: pass.
workshop/plans/000075-go-launcher-prototype-close-review.md:5661:- Ran `go run ./cmd/pair-go launch`: currently prints help with exit 0, confirming the default-agent route bug.
workshop/plans/000075-go-launcher-prototype-close-review.md:5662:- Missing test: actual `pair-go launch` no-arg route reaches decision phase.
workshop/plans/000075-go-launcher-prototype-close-review.md:5669:7. Plan revision recommendations:
workshop/plans/000075-go-launcher-prototype-close-review.md:5670:- Add `## Revisions`: “HistorySource queue scanning was deferred; current #75 implementation scans draft/log sidecars only, while queue badges remain future picker UI work.” Or implement `queue-*` support and keep the table unchanged.
workshop/plans/000075-go-launcher-prototype-close-review.md:5675:The implementation has a solid pure launcher core and fakeable IO seams, and the public `bin/pair` path is untouched. The blocker is that the actual guarded command path does not exercise the default-agent launch slice promised by the issue: `pair-go launch` returns help instead of reaching the decision boundary.
workshop/plans/000075-go-launcher-prototype-close-review.md:5678:- `cmd/internal/launcher/decision.go:21` keeps create/attach/pick logic pure and unit-testable (`ARCH-PURE` pass).
workshop/plans/000075-go-launcher-prototype-close-review.md:5684:- `cmd/internal/dispatcher/dispatcher.go:108`: `pair-go launch` with no launch args returns help. The issue explicitly requires argv parsing “including default agent” and a guarded path that reaches the decision boundary (`workshop/issues/000075-go-launcher-prototype.md:27`). `ParseArgs(nil)` supports `claude` (`cmd/internal/launcher/args.go:27`), but the dispatcher bypasses it. Fix: only treat explicit `help|--help|-h` as help, let empty `launchArgs` call `launcher.Run`, and add dispatcher/process tests for `pair-go launch` returning a prototype decision. `ARCH-PURPOSE` flag.
workshop/plans/000075-go-launcher-prototype-close-review.md:5685:- `workshop/plans/000075-go-launcher-prototype-plan.md:57`: the integration table says `HistorySource` wraps `queue-*`, but implementation scans only `draft-*.md` and `log-*.md` (`cmd/internal/launcher/history.go:18`). Either implement the queued-surface scan or revise the plan with a `## Revisions` entry clarifying queue badges are future scope. As written, the core concepts table contradicts the code.
workshop/plans/000075-go-launcher-prototype-close-review.md:5694:- Ran `go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1`: pass.
workshop/plans/000075-go-launcher-prototype-close-review.md:5696:- Ran `go run ./cmd/pair-go launch`: currently prints help with exit 0, confirming the default-agent route bug.
workshop/plans/000075-go-launcher-prototype-close-review.md:5697:- Missing test: actual `pair-go launch` no-arg route reaches decision phase.
workshop/plans/000075-go-launcher-prototype-close-review.md:5704:7. Plan revision recommendations:
workshop/plans/000075-go-launcher-prototype-close-review.md:5705:- Add `## Revisions`: “HistorySource queue scanning was deferred; current #75 implementation scans draft/log sidecars only, while queue badges remain future picker UI work.” Or implement `queue-*` support and keep the table unchanged.
cmd/internal/adapt/adapt_test.go:16:func TestLogLineFormat(t *testing.T) {
cmd/internal/adapt/adapt_test.go:21:	l.Log(2, "overlay-detect", NearMiss, "prompt-shaped, no detector matched: 'Do you want to continue?'")
cmd/internal/adapt/adapt_test.go:46:func TestNilLoggerIsNoOp(t *testing.T) {
cmd/internal/adapt/adapt_test.go:47:	var l *Logger
cmd/internal/adapt/adapt_test.go:49:	l.Log(1, "return-remap", Fired, "x")
cmd/internal/adapt/adapt_test.go:63:	l.Log(2, "overlay-detect", NearMiss, long)
cmd/internal/adapt/adapt_test.go:132:	l.Log(1, "x", Fired, "y") // must not panic
cmd/internal/adapt/adapt_test.go:147:	l1.Log(1, "return-remap", Fired, "first")
cmd/internal/adapt/adapt_test.go:152:	l2.Log(4, "slug-parse", NearMiss, "second")
cmd/internal/adapt/adapt.go:66:// Logger appends adaptation events for one component. A nil *Logger is a safe
cmd/internal/adapt/adapt.go:69:type Logger struct {
cmd/internal/adapt/adapt.go:92:// New builds a Logger writing to w. Used directly by tests; production code
cmd/internal/adapt/adapt.go:95:func New(w io.Writer, comp, agent string) *Logger {
cmd/internal/adapt/adapt.go:96:	return &Logger{w: w, comp: comp, agent: agent, now: time.Now}
cmd/internal/adapt/adapt.go:99:// Open opens the session flight recorder for appending and returns a Logger.
cmd/internal/adapt/adapt.go:100:// Returns nil (a no-op Logger) when $PAIR_TAG is unset or the file can't be
cmd/internal/adapt/adapt.go:102:func Open(comp, agent string) *Logger {
cmd/internal/adapt/adapt.go:117:// Log appends one event. Safe on a nil receiver. Errors are swallowed.
cmd/internal/adapt/adapt.go:118:func (l *Logger) Log(aspect int, signal string, outcome Outcome, detail string) {
cmd/internal/adapt/adapt.go:129:func (l *Logger) Close() error {
cmd/internal/dispatcher/dispatcher.go:9:	"github.com/xianxu/pair/cmd/internal/launcher"
cmd/internal/dispatcher/dispatcher.go:113:		return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
cmd/internal/dispatcher/dispatcher.go:118:			"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
cmd/internal/dispatcher/dispatcher.go:171:	b.WriteString("\nPlanned command families (not implemented in this skeleton):\n")
cmd/internal/launcher/history_test.go:10:func TestHistorySourceScansCwdPrefixedDraftAndLogSidecars(t *testing.T) {
cmd/internal/launcher/args.go:5:// LaunchArgs is the pure parse result for the guarded pair-go launch prototype.
cmd/internal/launcher/args.go:22:// ParseArgs parses pair-go launch args. It intentionally supports only the
cmd/internal/launcher/args.go:34:		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %s is not implemented by pair-go launch; use bin/pair", argv[0])}
cmd/internal/launcher/args.go:37:			return LaunchArgs{}, UsageError{Message: "pair-go launch: 'resume' requires a tag"}
cmd/internal/launcher/args.go:41:			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: invalid tag: %v", err)}
cmd/internal/launcher/args.go:44:			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", argv[2])}
cmd/internal/launcher/args.go:64:		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", arg)}
cmd/internal/launcher/args_test.go:83:			if !strings.Contains(err.Error(), "not implemented by pair-go launch") {
cmd/internal/dispatcher/dispatcher_test.go:8:func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
cmd/internal/dispatcher/dispatcher_test.go:53:func TestDispatchPlannedCommandReturnsUnsupported(t *testing.T) {
cmd/internal/dispatcher/dispatcher_test.go:76:	for _, want := range []string{"Usage: pair-go launch", "decision-phase prototype"} {
cmd/internal/dispatcher/dispatcher_test.go:97:	for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
cmd/internal/dispatcher/dispatcher_test.go:118:	for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=pair", "session=pair-pair"} {

exec
/bin/zsh -lc 'git diff --stat e89e2cea484ab8ca7ec9ca518fd2bf04ba38a8fe..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 atlas/architecture.md                           |  13 ++-
 atlas/go-migration-inventory.md                 |   2 +-
 cmd/internal/dispatcher/dispatcher.go           |  93 +++++++++++++++++++++
 cmd/internal/dispatcher/dispatcher_test.go      |  36 +++++++++
 cmd/internal/launcher/args.go                   |  71 ++++++++++++++++
 cmd/internal/launcher/args_test.go              |  88 ++++++++++++++++++++
 cmd/internal/launcher/datadir.go                |  11 +++
 cmd/internal/launcher/datadir_test.go           |  21 +++++
 cmd/internal/launcher/decision.go               | 103 ++++++++++++++++++++++++
 cmd/internal/launcher/decision_test.go          |  67 +++++++++++++++
 cmd/internal/launcher/history.go                |  65 +++++++++++++++
 cmd/internal/launcher/history_test.go           |  33 ++++++++
 cmd/internal/launcher/run.go                    |  70 ++++++++++++++++
 cmd/internal/launcher/run_test.go               |  90 +++++++++++++++++++++
 cmd/internal/launcher/session.go                |  28 +++++++
 cmd/internal/launcher/tag.go                    |  42 ++++++++++
 cmd/internal/launcher/tag_test.go               |  53 ++++++++++++
 cmd/internal/launcher/zellij.go                 |  91 +++++++++++++++++++++
 cmd/internal/launcher/zellij_test.go            |  44 ++++++++++
 cmd/pair-go/launch_process_test.go              |  71 ++++++++++++++++
 cmd/pair-go/main.go                             |   9 +++
 cmd/pair-go/main_test.go                        |  40 +++++++++
 workshop/issues/000075-go-launcher-prototype.md |  22 ++---
 23 files changed, 1148 insertions(+), 15 deletions(-)

codex
The issue file records a prior REWORK and a later close attempt. I’m checking the current tree rather than relying on that note, especially the default no-arg route and the plan’s core-concepts table.
exec
/bin/zsh -lc "nl -ba cmd/internal/dispatcher/dispatcher.go | sed -n '90,140p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
    90	
    91	type StaticHistory struct {
    92		Tags []launcher.HistoricalTag
    93		Err  error
    94	}
    95	
    96	func (h StaticHistory) Scan(_ string, _ time.Time) ([]launcher.HistoricalTag, error) {
    97		if h.Err != nil {
    98			return nil, h.Err
    99		}
   100		return h.Tags, nil
   101	}
   102	
   103	func DispatchWithLauncherRuntime(args []string, rt LauncherRuntime) Result {
   104		launchArgs := []string(nil)
   105		if len(args) > 1 {
   106			launchArgs = args[1:]
   107		}
   108		if len(launchArgs) > 0 && (launchArgs[0] == "help" || launchArgs[0] == "--help" || launchArgs[0] == "-h") {
   109			return Result{Stdout: LaunchHelp(programName), ExitCode: 0}
   110		}
   111		outcome, err := launcher.Run(launchArgs, rt.Env, rt.Sessions, rt.History)
   112		if err != nil {
   113			return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
   114		}
   115		decision := outcome.Decision
   116		return Result{
   117			Stderr: fmt.Sprintf(
   118				"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
   119				decision.Action,
   120				decision.Tag,
   121				decision.SessionName,
   122			),
   123			ExitCode: 3,
   124		}
   125	}
   126	
   127	func LaunchHelp(program string) string {
   128		return fmt.Sprintf(`Usage: %s launch [agent] [-- agent-args...]
   129	       %s launch resume <tag>
   130	
   131	Guarded decision-phase prototype. Public sessions still start through bin/pair.
   132	This command parses launch inputs and computes the create/attach/picker decision,
   133	then stops before invoking zellij.
   134	`, program, program)
   135	}
   136	
   137	func LauncherEnv(home, xdgDataHome, cwd string) launcher.Env {
   138		return launcher.Env{
   139			Home:     home,
   140			XDGData:  xdgDataHome,

exec
/bin/zsh -lc "nl -ba workshop/plans/000075-go-launcher-prototype-plan.md | sed -n '1,110p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	# Pair Go Launcher Prototype Implementation Plan
     2	
     3	> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.
     4	
     5	**Goal:** Add a guarded `pair-go launch` prototype that reaches Pair launcher decision logic without replacing or invoking the public shell launcher.
     6	
     7	**Architecture:** Put launcher business logic in a pure `cmd/internal/launcher` package and keep subprocess/filesystem work in thin, injectable seams (`ARCH-PURE`). Extend the existing #74 dispatcher instead of adding a parallel command parser (`ARCH-DRY`). The prototype prints the decision it would take, then exits with an explicit unsupported-after-decision code so the issue delivers a real launcher vertical slice without changing public `bin/pair` behavior (`ARCH-PURPOSE`).
     8	
     9	**Tech Stack:** Go standard library, existing `cmd/internal/dispatcher`, fake `zellij` process tests, `go test`, `make pair-go`.
    10	
    11	---
    12	
    13	## Core Concepts
    14	
    15	### Pure Entities
    16	
    17	| Name | Lives in | Status |
    18	|------|----------|--------|
    19	| `LaunchArgs` | `cmd/internal/launcher/args.go` | new |
    20	| `Tag` | `cmd/internal/launcher/tag.go` | new |
    21	| `DataDir` | `cmd/internal/launcher/datadir.go` | new |
    22	| `SessionSnapshot` | `cmd/internal/launcher/session.go` | new |
    23	| `LaunchDecision` | `cmd/internal/launcher/decision.go` | new |
    24	
    25	**LaunchArgs** — Parsed launch-specific argv: agent, forced tag, forwarded agent args, help flag, and unsupported subcommand errors.
    26	- **Relationships:** 1:1 with a `pair-go launch` invocation; owns a `Tag` only for forced resume.
    27	- **DRY rationale:** Keeps `pair-go launch` parsing behind the existing dispatcher instead of duplicating parser branches in `cmd/pair-go/main.go`.
    28	- **Future extensions:** `continue`, `rename`, and tag-restart prompts widen this parser after the prototype has tests.
    29	
    30	**Tag** — Normalized workspace tag, accepting either bare `demo` or `pair-demo` and rejecting empty or non `[A-Za-z0-9_-]` values. The canonical value is always bare (`demo`); zellij session names are derived only at the boundary as `pair-<tag>`.
    31	- **Relationships:** Used by `LaunchArgs`, `SessionSnapshot`, and `LaunchDecision`.
    32	- **DRY rationale:** Mirrors the shell launcher's `normalize_tag` as a named Go concept so later Go launcher work has one validation point.
    33	- **Future extensions:** Length checks can move here when the Go path reaches real zellij session creation.
    34	
    35	**DataDir** — Resolved Pair data directory from `XDG_DATA_HOME` or `$HOME/.local/share/pair`.
    36	- **Relationships:** Provides the root for historical sidecars and future config/session files.
    37	- **DRY rationale:** Prevents each command seam from recomputing Pair's data directory.
    38	- **Future extensions:** Asset/data path resolution can join this with future `PAIR_HOME` discovery.
    39	
    40	**SessionSnapshot** — In-memory view of active zellij rows and historical tag candidates relevant to the current cwd.
    41	- **Relationships:** 1:N with zellij sessions and historical rows; consumed by `LaunchDecision`.
    42	- **DRY rationale:** Separates "what exists" from "what should we do", matching the shell launcher's implicit stages.
    43	- **Future extensions:** Can add queue badges, age coloring, and config-derived agent inference without changing decision callers.
    44	
    45	**LaunchDecision** — Pure create/attach/picker-required decision for forced resume, empty state, detached sessions, and historical tags. It carries the canonical bare `Tag` and, for attach/create decisions that name zellij, the derived `SessionName` (`pair-<tag>`) so comparisons and printouts cannot accidentally mix forms.
    46	- **Relationships:** N:1 from snapshot plus args to one decision.
    47	- **DRY rationale:** Pulls the business rule out of command execution so unit tests do not need zellij/fzf.
    48	- **Future extensions:** Real fzf selection can become another input shape instead of branching inside IO code.
    49	
    50	The launcher package must not define a second stdout/stderr/exit-code result type. `cmd/internal/dispatcher.Result` remains the single process-facing result abstraction (`ARCH-DRY`). Launcher functions return domain values (`LaunchDecision`, snapshots, parse errors); the dispatcher route converts those values into `dispatcher.Result`.
    51	
    52	### Integration Points
    53	
    54	| Name | Lives in | Status | Wraps |
    55	|------|----------|--------|-------|
    56	| `ZellijSource` | `cmd/internal/launcher/zellij.go` | new | `zellij list-sessions`, `zellij --session ... action list-clients` |
    57	| `HistorySource` | `cmd/internal/launcher/history.go` | new | filesystem scan of `draft-*.md` and `log-*.md` under Pair data dir |
    58	| `LaunchCommand` | `cmd/internal/dispatcher/dispatcher.go` | modified | existing dispatcher route for `launch` |
    59	| `PairGoMain` | `cmd/pair-go/main.go` | modified | process stdout/stderr exit handling |
    60	
    61	**ZellijSource** — Reads zellij session rows and client counts through `exec.Command`.
    62	- **Injected into:** `launcher.Run`, which converts IO into a pure `SessionSnapshot`.
    63	- **Future extensions:** Timeout behavior can be added when the Go launcher owns real launch flow; do not port the shell `zj` timeout in this prototype unless the process fake exposes the need.
    64	
    65	**HistorySource** — Scans cwd-prefixed historical sidecars from the resolved data dir.
    66	- **Injected into:** `launcher.Run`.
    67	- **Future extensions:** Queue badges and age display can be layered onto historical rows when picker UI is implemented.
    68	
    69	**LaunchCommand** — Routes `pair-go launch` from the existing dispatcher to the launcher runner.
    70	- **Injected into:** `cmd/pair-go` via the existing `run(args, stdout, stderr)` path.
    71	- **Future extensions:** Other implemented subcommands can follow the same dispatcher pattern.
    72	- **Boundary rule:** production environment reads (`os.Getenv`, `os.Getwd`) and `exec.Command` construction live in a small launcher IO constructor used by the dispatcher route. Tests may call a dispatcher test seam with an explicit launcher runtime. The launcher core returns domain outcomes; only the dispatcher maps those outcomes to stdout/stderr/exit code.
    73	
    74	**PairGoMain** — No business logic; writes dispatcher-returned streams and exits.
    75	- **Injected into:** none.
    76	- **Future extensions:** May eventually become the public `pair` binary entrypoint in #77, but not here.
    77	
    78	## Revisions
    79	
    80	### 2026-06-29 — Close review correction
    81	
    82	Reason: the close review found the integration table claimed `HistorySource` scanned `queue-*`, while the implemented #75 prototype only uses draft/log sidecars as historical tag candidates.
    83	
    84	Delta: revised the `HistorySource` integration row to list `draft-*.md` and `log-*.md` only. Queue badges remain future picker UI scope, as already noted in the `HistorySource` future extensions.
    85	
    86	## Chunk 1: Pure Launcher Core
    87	
    88	### Task 1: Parse `pair-go launch` Args
    89	
    90	**Files:**
    91	- Create: `cmd/internal/launcher/args.go`
    92	- Create: `cmd/internal/launcher/args_test.go`
    93	
    94	- [ ] **Step 1: Write failing parse tests**
    95	
    96	Cover:
    97	- no args: default agent `claude`;
    98	- `<agent>`: custom agent;
    99	- `<agent> -- <args>` and `-- <args>` forwarding;
   100	- `resume <tag>` strips `pair-` and records forced tag;
   101	- unexpected extra positional includes `unexpected positional arg` and `use '--' to forward args to the agent`;
   102	- unsupported `continue`, `rename`, and `list` under `launch` return explicit prototype errors.
   103	
   104	Run: `go test ./cmd/internal/launcher -run 'TestParseLaunchArgs' -count=1`
   105	Expected: FAIL because package/files do not exist.
   106	
   107	- [ ] **Step 2: Implement minimal parser**
   108	
   109	Create `LaunchArgs`, `ParseArgs(args []string) (LaunchArgs, error)`, and a typed `UsageError`.
   110	

exec
/bin/zsh -lc "nl -ba cmd/internal/launcher/history.go | sed -n '1,80p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package launcher
     2	
     3	import (
     4		"os"
     5		"path/filepath"
     6		"sort"
     7		"strings"
     8		"time"
     9	)
    10	
    11	// HistorySource scans Pair draft/log sidecars under the data directory.
    12	type HistorySource struct {
    13		DataDir string
    14	}
    15	
    16	func (s HistorySource) Scan(base string, cutoff time.Time) ([]HistoricalTag, error) {
    17		latest := map[string]time.Time{}
    18		for _, pattern := range []string{"draft-*.md", "log-*.md"} {
    19			matches, err := filepath.Glob(filepath.Join(s.DataDir, pattern))
    20			if err != nil {
    21				return nil, err
    22			}
    23			for _, path := range matches {
    24				tag, ok := tagFromSidecar(filepath.Base(path))
    25				if !ok || !matchesHistoryBase(tag, base) {
    26					continue
    27				}
    28				info, err := os.Stat(path)
    29				if err != nil {
    30					continue
    31				}
    32				if info.ModTime().Before(cutoff) {
    33					continue
    34				}
    35				if info.ModTime().After(latest[tag]) {
    36					latest[tag] = info.ModTime()
    37				}
    38			}
    39		}
    40		tags := make([]string, 0, len(latest))
    41		for tag := range latest {
    42			tags = append(tags, tag)
    43		}
    44		sort.Strings(tags)
    45		out := make([]HistoricalTag, 0, len(tags))
    46		for _, tag := range tags {
    47			out = append(out, HistoricalTag{Tag: tag})
    48		}
    49		return out, nil
    50	}
    51	
    52	func tagFromSidecar(name string) (string, bool) {
    53		switch {
    54		case strings.HasPrefix(name, "draft-") && strings.HasSuffix(name, ".md"):
    55			return strings.TrimSuffix(strings.TrimPrefix(name, "draft-"), ".md"), true
    56		case strings.HasPrefix(name, "log-") && strings.HasSuffix(name, ".md"):
    57			return strings.TrimSuffix(strings.TrimPrefix(name, "log-"), ".md"), true
    58		default:
    59			return "", false
    60		}
    61	}
    62	
    63	func matchesHistoryBase(tag, base string) bool {
    64		return tag == base || strings.HasPrefix(tag, base+"-")
    65	}

exec
/bin/zsh -lc 'git diff --name-only e89e2cea484ab8ca7ec9ca518fd2bf04ba38a8fe..HEAD -- workshop/plans/000075-go-launcher-prototype-plan.md workshop/issues/000075-go-launcher-prototype.md atlas cmd bin/pair' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
atlas/architecture.md
atlas/go-migration-inventory.md
cmd/internal/dispatcher/dispatcher.go
cmd/internal/dispatcher/dispatcher_test.go
cmd/internal/launcher/args.go
cmd/internal/launcher/args_test.go
cmd/internal/launcher/datadir.go
cmd/internal/launcher/datadir_test.go
cmd/internal/launcher/decision.go
cmd/internal/launcher/decision_test.go
cmd/internal/launcher/history.go
cmd/internal/launcher/history_test.go
cmd/internal/launcher/run.go
cmd/internal/launcher/run_test.go
cmd/internal/launcher/session.go
cmd/internal/launcher/tag.go
cmd/internal/launcher/tag_test.go
cmd/internal/launcher/zellij.go
cmd/internal/launcher/zellij_test.go
cmd/pair-go/launch_process_test.go
cmd/pair-go/main.go
cmd/pair-go/main_test.go
workshop/issues/000075-go-launcher-prototype.md

exec
/bin/zsh -lc "go run ./cmd/pair-go launch; printf 'exit=%s\\n' "'$?' in /Users/xianxu/workspace/pair
 succeeded in 2463ms:
pair-go launch: prototype decision action=pick tag= session=; real zellij launch remains shell-owned
exit status 3
exit=1

codex
The current HEAD has already fixed the no-arg `launch` bypass and revised the plan’s HistorySource row. I’m doing a deeper pass now for behavior drift from `bin/pair`, error handling in the IO seams, and whether the tests actually exercise the promised slice.
exec
/bin/zsh -lc "sed -n '1,260p' bin/pair" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
#!/usr/bin/env bash
# pair — launch a zellij session split between a TUI coding agent (top pane)
# and an nvim drafting pane (bottom). See README.md.
#
# Usage:
#   pair                       claude in default session
#   pair <agent>               claude / codex / agy
#   pair [<agent>] -- <args>   forward args to agent on create,
#                              e.g. pair claude -- --resume
#                                   pair -- --dangerously-skip-permissions
#                                   pair codex -- -p "say hi"
#   pair -h, --help            show help
#
# When detached pair-* sessions exist, an fzf picker is shown so the user
# explicitly chooses what to do — no silent auto-attach. The picker shows
# ALL detached pair-* sessions regardless of agent argument; the agent
# argument only matters for the create path (sentinel label, default name,
# binary to exec).

set -euo pipefail

# Wrap zellij with a hard timeout for read-only IPC. The daemon socket can
# wedge if an earlier client got stuck (seen 2026-05-06: a `list-sessions`
# from hours prior held the socket and every new pair launch piled up behind
# it). pair makes several of these per launch, so one wedge cascades into
# unkillable bash trees. 5s is generous — healthy responses are <100ms; if
# it doesn't return by then, something's wrong and falling through is better
# than hanging. Long-running zellij invocations (attach, new-session) are
# intentionally NOT wrapped — they're meant to block.
zj() {
    local secs=5
    if command -v gtimeout >/dev/null 2>&1; then
        gtimeout "$secs" zellij "$@"
    elif command -v timeout >/dev/null 2>&1; then
        timeout "$secs" zellij "$@"
    else
        # Pure-bash fallback for systems without coreutils' timeout (default
        # macOS). Run zellij in the background, arm a watchdog, wait.
        #
        # The watchdog's stdio is redirected to /dev/null. Otherwise, in a
        # pipeline like `zj list-sessions | awk ...`, the watchdog subshell
        # inherits stdout = pipe-to-awk; when we SIGTERM it after zellij
        # exits, bash dies but the orphan `sleep` child keeps the pipe FD
        # open, so awk waits for EOF until the full $secs elapses. With N
        # piped zj calls in pair's startup, that compounds (we saw N×5s).
        zellij "$@" &
        local pid=$!
        ( sleep "$secs"; kill -9 "$pid" 2>/dev/null ) </dev/null >/dev/null 2>&1 &
        local watchdog=$!
        local rc=0
        wait "$pid" 2>/dev/null || rc=$?
        kill "$watchdog" 2>/dev/null || true
        wait "$watchdog" 2>/dev/null || true
        return $rc
    fi
}

# normalize_tag <raw> <errprefix> — accept a bare tag or `pair-<tag>`, strip the
# prefix, and validate the charset (letters/digits/dash/underscore). Echoes the
# cleaned tag on stdout; on an empty/invalid value prints "<errprefix> '<raw>'
# (allowed: …)" to stderr and returns 1. Callers: t="$(normalize_tag "$x" "…")" || exit 1.
# Shared by resume/continue/rename + the name prompt (#52 ARCH-DRY). It *returns*
# (never exits) so it composes inside $( ), where an exit would only kill the subshell.
normalize_tag() {
    local _raw="$1" _t="$1"
    case "$_t" in pair-*) _t="${_t#pair-}" ;; esac
    case "$_t" in
        ""|*[!A-Za-z0-9_-]*)
            printf "%s '%s' (allowed: letters, digits, dash, underscore)\n" "$2" "$_raw" >&2
            return 1 ;;
    esac
    printf '%s' "$_t"
}

# --- help -------------------------------------------------------------------
case "${1:-}" in
    -h|--help|help)
        cat <<'EOF'
pair — Neovim-backed input field for any TUI coding agent.

USAGE
  pair                          claude in default session
  pair <agent>                  e.g. pair codex / pair agy
  pair resume <tag>             attach pair-<tag> if it exists, else create
                                with that tag and the saved config; agent
                                is inferred from saved state
  pair continue [slug]          resume from a continuation doc; bare lists them
  pair [<agent>] -- <args>      forward args to the agent on create
                                e.g. pair claude -- --resume
                                     pair -- --dangerously-skip-permissions
                                     pair codex -- -p "say hi"
  pair list                     list pair-* sessions and their attach state
  pair rename <old> <new>       rename every tag-scoped file in
                                $PAIR_DATA_DIR from <old> to <new>.
                                Refuses if `pair-<old>` is still tracked
                                by zellij (use Ctrl+Alt+n's (R)ename
                                option for the in-session equivalent).
  pair -h, --help               this message

  Use `--` to separate pair's args from the agent's args. Without `--`,
  pair only accepts `<agent>` as a positional (or `resume <tag>`).

  DEV MODE
  `pair-dev` is identical to `pair` but rebuilds pair's Go binaries from
  source (`make build`) on launch AND on every Alt+n / Shift+Alt+N restart,
  so the zellij-spawned pair-wrap always matches the working tree. Use it
  when hacking on cmd/*; deployed installs keep running `pair` (prebuilt,
  no toolchain dependency). See bin/pair-dev + bin/lib/dev-rebuild.sh.

  When creating a new session you'll be prompted for a name; type any
  custom name (e.g. "bugfix") to make it independent of other sessions.
  `resume <tag>` skips that prompt — use it to restart a saved tag with
  its captured args (and the agent's session if still on disk).

  Agent args (after `--`) only apply on CREATE. Reattaching to an
  existing session uses whatever args the agent was originally started
  with — the picker doesn't re-launch the agent.

BEHAVIOR
  Picker shows ALL detached pair-* sessions, regardless of the agent
  argument. The agent argument only controls what gets created if you
  pick the "+ new" sentinel: it labels the sentinel, drives the auto-
  suggested name, and is the binary that gets exec'd.

    0 detached + 0 historical → create directly (prompts for name;
                                default is auto-generated like
                                pair-claude-2)
    ≥1 detached or ≥1 historical → fzf picker over: detached pair-*
                                sessions, then historical tags from
                                this cwd within $PAIR_HISTORY_DAYS
                                (default 14) annotated `(Nd ago, no
                                live session)`, then a `+ new <agent>
                                session` sentinel. Detached row →
                                attach. Historical row → create-by-name
                                (reuses saved draft / agent config).
                                Sentinel → fall through to create.

  Historical surface (#24) uses the cwd basename as prefix: tags named
  `<cwd-base>` or `<cwd-base>-<subproject>` whose draft/log sidecars
  were touched within the window. PAIR_DEBUG_HISTORY=1 prints the scan
  results and exits without launching zellij.

  No silent attach. Every reattach is explicit.

KEYBINDINGS (inside a pair session)
  Alt+Return    send nvim buffer to agent + Enter, log, clear, insert mode
  Alt+Shift+Return
                append nvim buffer to agent composer + newline, no submit;
                log + clear (like Alt+Return but unsubmitted)
  Alt+c         open/show/hide the review collaboration pane; if no review
                target exists, starts :PairReview
  Ctrl+C        send ESC (0x1b) to agent pane — interrupts claude's
                in-flight stream without leaving the draft
  Alt+← / Alt+→ walk through prompt history (-N) ↔ draft (*) ↔ queue (+N)
  Shift+Alt+← / Shift+Alt+→
                jump to next region boundary (oldest-history / newest-history /
                * / front-of-queue / back-of-queue)
  Alt+1..Alt+9  (insert mode, completion popup visible) quick-pick the Nth
                visible completion item. Items show their index (e.g.
                `1 bin/pair-wrap`); past 9, use arrows or <C-n>/<C-p>.
  Alt+q         push current buffer to queue front (+1)
                from *: parks the draft. from -N: forks. from +N: move-to-front
  Alt+BS        (at +N) delete the current queued prompt; items shift down,
                so tap-tap deletes a run
  Shift+Alt+BS  erase history + draft + queue ("start anew"); confirms first
  Alt+↑ / Alt+↓ step nvim pane along minimized ↔ 12 lines ↔ 1/2 ladder
                (works from either pane). minimized = statusline only;
                Alt+↑ from there grows the pane back. Confirm-requiring
                keys (Alt+x/d/n/N) auto-grow out of minimized first so
                the prompt is visible.
  Alt+d         detach from session (re-attach later via `pair`)
  Alt+x         full quit — kill session and remove from resurrect list
                (Ctrl+q is unbound; Alt+x is the only quit path)
  Alt+n         reload pair — kill the session and re-launch in place
                with the same tag, agent, args, AND agent session.
                The conversation resumes; pair itself is the only
                thing that restarts (useful after a config or binary
                rebuild). Ctrl+Alt+n is a bound alias — use it on macOS
                terminal combos where Option+n triggers the dead-tilde
                composer instead of forwarding as Meta+n.
  Shift+Alt+N   restart with a fresh agent conversation — same tag,
                agent, and args, but the saved per-(tag,agent) config
                is dropped so the agent starts a brand-new session
  Alt+h         pop up this help in a floating pane (press `q` to dismiss)
  Alt+i         attach clipboard image to agent + insert whatever marker
                the agent renders (e.g. claude `[Image #N]`). Captured
                from pair-wrap's PTY stream so no per-agent format
                knowledge is baked in. Flashes
                "[no image in clipboard]" at cursor for 1s if the OS
                clipboard has no image data.
  Alt+/         open agent pane scrollback in a read-only ANSI-colored
                nvim viewer; line numbers match zellij's frame indicator,
                so e.g. `:880` jumps to the line you saw scrolled to 880.
                Inside the viewer:
                  • Alt+q drops a 🤖[comment] marker — bare in normal
                    mode (the line is the context) or 🤖<selection>
                    [comment] in visual mode.
                  • Alt+b / Alt+B jump to previous / next prompt
                    boundary, so you can hop between turns instead of
                    scrolling line-by-line.
                  • Search is smart-case: `/foo` matches Foo/FOO/foo,
                    `/Foo` only Foo.
                  • <Esc> exits. If markers (or an overall comment) are
                    pending, a Yes/No confirm runs first; on accept,
                    everything is formatted (`> <quote>\n<comment>`)
                    and appended to the draft for review + send.

  Status line shape: `Alt: <- history H < pos[*] [hint] > Q queued ->`
  — H/Q are total counts, pos is `*`/`-N`/`+N`. A trailing `*` on `-N`
  means a pending fork on that history slot; navigating away prompts:
  (S)end, (Q)ueue, (D)iscard, [S]tay. The `[hint]` is contextual:
  `[q=queue]` on */-N, `[⌫=del]` on +N.

DRAFT SYNTAX
  ===  comment  Whole lines starting with `===` are stripped at send time
                but kept in draft/queue/log. Use them to annotate why a
                queued prompt exists. Mid-line `===` is unaffected.

FILES
  ${XDG_DATA_HOME:-~/.local/share}/pair/draft-<tag>.md     persistent draft (*)
  ${XDG_DATA_HOME:-~/.local/share}/pair/log-<tag>.md       prompt history (-N)
  ${XDG_DATA_HOME:-~/.local/share}/pair/queue-<tag>/*.md   queued prompts (+N)

See README.md for image-paste recipes and other details.
EOF
        exit 0
        ;;
    list|ls)
        if ! command -v zellij >/dev/null 2>&1; then
            echo "pair: zellij not found on PATH." >&2
            exit 1
        fi
        # Snapshot session state once. --no-formatting carries the EXITED
        # annotation; --short gives clean newline-separated names.
        raw=$( { zj list-sessions --no-formatting 2>/dev/null || true; } )
        sessions=$( { zj list-sessions --short 2>/dev/null || true; } \
                    | awk '/^pair-/' \
                    | sort )
        if [ -z "$sessions" ]; then
            echo "no pair sessions"
            exit 0
        fi
        list_data_dir="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
        mkdir -p "$list_data_dir"

        # Backfill agent-<tag> for any running pair-wrap process whose tag
        # we don't yet have on disk — covers sessions created before the
        # agent-tracking feature shipped. Read PAIR_TAG from each process's
        # environ (macOS: `ps -E`; Linux: /proc/<pid>/environ) and the
        # agent name from argv (the token after `pair-wrap`).
        pw_pids=$(pgrep -f 'pair-wrap' 2>/dev/null || true)
        for pid in $pw_pids; do
            argv=$(ps -p "$pid" -o command= 2>/dev/null || true)
            [ -z "$argv" ] && continue
            agent=$(printf '%s\n' "$argv" | awk '
                { for (i=1; i<=NF; i++) if ($i ~ /pair-wrap$/) { print $(i+1); exit } }')
            [ -z "$agent" ] && continue
            if [ -r "/proc/$pid/environ" ]; then
                env_blob=$(tr '\0' '\n' < "/proc/$pid/environ" 2>/dev/null || true)
            else

exec
/bin/zsh -lc "nl -ba cmd/internal/launcher/args.go cmd/internal/launcher/tag.go cmd/internal/launcher/decision.go cmd/internal/launcher/run.go cmd/internal/launcher/zellij.go | sed -n '1,360p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
usage: nl [-p] [-b type] [-d delim] [-f type] [-h type] [-i incr] [-l num]
          [-n format] [-s sep] [-v startnum] [-w width] [file]

exec
/bin/zsh -lc 'rg -n "normalize|PAIR_TAG|TAG|SESSION|fzf|draft-|log-|list-sessions|resume|continue|rename|agent_args|PAIR_AGENT|PAIR_DATA_DIR|XDG_DATA_HOME|local/share/pair" bin/pair tests cmd/internal/launcher cmd/internal/dispatcher cmd/pair-go' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
bin/pair:9:#                              e.g. pair claude -- --resume
bin/pair:14:# When detached pair-* sessions exist, an fzf picker is shown so the user
bin/pair:23:# wedge if an earlier client got stuck (seen 2026-05-06: a `list-sessions`
bin/pair:41:        # pipeline like `zj list-sessions | awk ...`, the watchdog subshell
bin/pair:58:# normalize_tag <raw> <errprefix> — accept a bare tag or `pair-<tag>`, strip the
bin/pair:61:# (allowed: …)" to stderr and returns 1. Callers: t="$(normalize_tag "$x" "…")" || exit 1.
bin/pair:62:# Shared by resume/continue/rename + the name prompt (#52 ARCH-DRY). It *returns*
bin/pair:64:normalize_tag() {
bin/pair:84:  pair resume <tag>             attach pair-<tag> if it exists, else create
bin/pair:87:  pair continue [slug]          resume from a continuation doc; bare lists them
bin/pair:89:                                e.g. pair claude -- --resume
bin/pair:93:  pair rename <old> <new>       rename every tag-scoped file in
bin/pair:94:                                $PAIR_DATA_DIR from <old> to <new>.
bin/pair:101:  pair only accepts `<agent>` as a positional (or `resume <tag>`).
bin/pair:112:  `resume <tag>` skips that prompt — use it to restart a saved tag with
bin/pair:128:    ≥1 detached or ≥1 historical → fzf picker over: detached pair-*
bin/pair:176:                The conversation resumes; pair itself is the only
bin/pair:220:  ${XDG_DATA_HOME:-~/.local/share}/pair/draft-<tag>.md     persistent draft (*)
bin/pair:221:  ${XDG_DATA_HOME:-~/.local/share}/pair/log-<tag>.md       prompt history (-N)
bin/pair:222:  ${XDG_DATA_HOME:-~/.local/share}/pair/queue-<tag>/*.md   queued prompts (+N)
bin/pair:235:        raw=$( { zj list-sessions --no-formatting 2>/dev/null || true; } )
bin/pair:236:        sessions=$( { zj list-sessions --short 2>/dev/null || true; } \
bin/pair:243:        list_data_dir="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
bin/pair:248:        # agent-tracking feature shipped. Read PAIR_TAG from each process's
bin/pair:254:            [ -z "$argv" ] && continue
bin/pair:257:            [ -z "$agent" ] && continue
bin/pair:267:                  | awk -F= '$1=="PAIR_TAG"{print $2; exit}')
bin/pair:268:            [ -z "$tag" ] && continue
bin/pair:274:        printf '%-30s %-10s %s\n' "SESSION" "AGENT" "STATUS"
bin/pair:276:            [ -z "$sess" ] && continue
bin/pair:307:    rename)
bin/pair:308:        # `pair rename <old> <new>` — rename every tag-scoped file in
bin/pair:309:        # $PAIR_DATA_DIR from <old> to <new>. Offline-only: refuses if
bin/pair:311:        # or resurrectable). The inside-session rename gesture (M2) takes
bin/pair:312:        # care of the quit→rename→re-exec choreography on top of this
bin/pair:316:        # that way `pair rename brain newname` can't accidentally pick up
bin/pair:323:        #                     Used by the in-session rename gesture
bin/pair:327:        #                     `zellij list-sessions` is expected, not a
bin/pair:329:        rename_check_only=0
bin/pair:330:        rename_skip_old_live=0
bin/pair:331:        rename_argi=2
bin/pair:333:            case "${!rename_argi:-}" in
bin/pair:335:                    rename_check_only=1
bin/pair:336:                    rename_skip_old_live=1
bin/pair:337:                    rename_argi=$((rename_argi + 1))
bin/pair:339:                --) rename_argi=$((rename_argi + 1)); break ;;
bin/pair:343:        old_tag="${!rename_argi:-}"
bin/pair:344:        rename_argi=$((rename_argi + 1))
bin/pair:345:        new_tag="${!rename_argi:-}"
bin/pair:346:        rename_argi=$((rename_argi + 1))
bin/pair:348:            echo "usage: pair rename [--restart-check] <old> <new>" >&2
bin/pair:351:        if [ -n "${!rename_argi:-}" ]; then
bin/pair:352:            echo "pair rename: unexpected arg '${!rename_argi}'" >&2
bin/pair:356:        # shared helper (#52 ARCH-DRY). Length stays a rename-specific guard below.
bin/pair:357:        old_tag="$(normalize_tag "$old_tag" "pair rename: invalid tag")" || exit 1
bin/pair:358:        new_tag="$(normalize_tag "$new_tag" "pair rename: invalid tag")" || exit 1
bin/pair:361:                echo "pair rename: tag '$t' is too long (max 256)" >&2
bin/pair:366:            echo "pair rename: old and new tag are the same ('$old_tag')" >&2
bin/pair:369:        rename_dd="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
bin/pair:370:        if [ ! -d "$rename_dd" ]; then
bin/pair:371:            echo "pair rename: data dir not found: $rename_dd" >&2
bin/pair:379:            rename_sessions=$( { zj list-sessions --short 2>/dev/null || true; } )
bin/pair:380:            if [ "$rename_skip_old_live" -eq 0 ] \
bin/pair:381:               && printf '%s\n' "$rename_sessions" | grep -qx "pair-${old_tag}"; then
bin/pair:382:                echo "pair rename: session 'pair-${old_tag}' is still tracked by zellij." >&2
bin/pair:384:                echo "             rename via the Ctrl+Alt+n restart confirm." >&2
bin/pair:387:            if printf '%s\n' "$rename_sessions" | grep -qx "pair-${new_tag}"; then
bin/pair:388:                echo "pair rename: session 'pair-${new_tag}' already exists in zellij." >&2
bin/pair:396:        rename_paths_for() {
bin/pair:404:            printf '%s\n' "$dd/draft-$tag.md"
bin/pair:405:            printf '%s\n' "$dd/log-$tag.md"
bin/pair:415:                printf '%s\n' "$dd/draft-$tag-$a.md"
bin/pair:421:        # rename onto it would conflict.
bin/pair:422:        rename_new_occupied=""
bin/pair:425:                rename_new_occupied="$p"
bin/pair:428:        done < <(rename_paths_for "$new_tag" "$rename_dd")
bin/pair:429:        if [ -n "$rename_new_occupied" ]; then
bin/pair:430:            echo "pair rename: tag '$new_tag' is occupied — '$rename_new_occupied' exists." >&2
bin/pair:435:        rename_plan=""
bin/pair:436:        rename_count=0
bin/pair:437:        old_paths_tmp="$(mktemp "${TMPDIR:-/tmp}/pair-rename.XXXXXX")"
bin/pair:438:        rename_paths_for "$old_tag" "$rename_dd" > "$old_paths_tmp"
bin/pair:440:            [ -e "$src" ] || continue
bin/pair:443:            # rename_paths_for as "<dir>/<chunk-with-$old_tag>", so the
bin/pair:451:                    dst="$rename_dd/${base%-"$old_tag"}-$new_tag"
bin/pair:454:                    dst="$rename_dd/image-capture-$new_tag.done"
bin/pair:456:                draft-$old_tag.md)
bin/pair:457:                    dst="$rename_dd/draft-$new_tag.md"
bin/pair:459:                log-$old_tag.md)
bin/pair:460:                    dst="$rename_dd/log-$new_tag.md"
bin/pair:463:                    dst="$rename_dd/nvim-pid-$new_tag-${base##*-}"
bin/pair:467:                    dst="$rename_dd/config-$new_tag-$suf"
bin/pair:471:                    dst="$rename_dd/pane-$new_tag-$suf"
bin/pair:475:                    dst="$rename_dd/scrollback-$new_tag-$suf"
bin/pair:477:                draft-$old_tag-*.md)
bin/pair:478:                    suf="${base#draft-$old_tag-}"
bin/pair:479:                    dst="$rename_dd/draft-$new_tag-$suf"
bin/pair:482:                    echo "pair rename: internal error: unrecognised src '$src'" >&2
bin/pair:488:                echo "pair rename: destination already exists: $dst" >&2
bin/pair:492:            rename_plan="${rename_plan}${src}	${dst}
bin/pair:494:            rename_count=$((rename_count + 1))
bin/pair:498:        if [ "$rename_count" -eq 0 ]; then
bin/pair:499:            echo "pair rename: no files found for tag '$old_tag' in $rename_dd" >&2
bin/pair:505:        # rename runs later, post-kill, from handle_restart_marker.
bin/pair:506:        if [ "$rename_check_only" -eq 1 ]; then
bin/pair:507:            echo "pair rename: ok ($rename_count file(s) would move from '$old_tag' to '$new_tag')"
bin/pair:511:        # Journal the plan to disk before executing so a crash-mid-rename
bin/pair:513:        rename_journal="$rename_dd/.rename-$old_tag-to-$new_tag.journal"
bin/pair:514:        printf '%s' "$rename_plan" > "$rename_journal"
bin/pair:516:        echo "pair rename: $rename_count file(s) ${old_tag} → ${new_tag}"
bin/pair:517:        rename_done=0
bin/pair:518:        rename_failed=""
bin/pair:520:            [ -z "$src" ] && continue
bin/pair:522:                rename_done=$((rename_done + 1))
bin/pair:524:                rename_failed="$src → $dst"
bin/pair:528:$rename_plan
bin/pair:531:        if [ -n "$rename_failed" ]; then
bin/pair:532:            echo "pair rename: mv failed: $rename_failed" >&2
bin/pair:533:            echo "pair rename: rolling back $rename_done completed rename(s)..." >&2
bin/pair:535:            head -n "$rename_done" "$rename_journal" \
bin/pair:543:        rm -f "$rename_journal"
bin/pair:544:        echo "pair rename: ok"
bin/pair:575:#   pair claude -- --resume               → claude --resume
bin/pair:576:#   pair -- --resume                      → claude --resume
bin/pair:582:# `pair resume <tag>` is the documented restart path: skip both the picker
bin/pair:584:# that tag. We claim `resume` as a subcommand verb — it can't be a real
bin/pair:587:if [ "${1:-}" = "resume" ]; then
bin/pair:589:        echo "pair: 'resume' requires a tag (e.g. pair resume bugfix)" >&2
bin/pair:592:    forced_tag="$(normalize_tag "$2" "pair: invalid tag")" || exit 1
bin/pair:596:# `pair continue [slug] [agent]` — the human-understanding sibling of `resume`
bin/pair:602:# the agent. The optional agent is the "port" (continue under a different
bin/pair:605:# `continue` never reads session_id — that's `resume`.
bin/pair:608:# path ATTACHES to it (same as `pair <agent>` / `resume`) rather than reseeding —
bin/pair:609:# intentional, so a continue never clobbers a running session; pick a fresh tag
bin/pair:612:if [ "${1:-}" = "continue" ]; then
bin/pair:631:    _cslug="$(normalize_tag "$2" "pair: invalid slug")" || exit 1
bin/pair:637:    # No forced_tag (see header). Consume `continue` + slug; an optional next
bin/pair:643:        AGENT="$1"   # explicit port: continue under a different agent
bin/pair:664:        echo "      e.g. pair claude -- --resume" >&2
bin/pair:669:# PAIR_FORCE_TAG is set by the Alt+n / Shift+Alt+N restart flow
bin/pair:674:if [ -z "$forced_tag" ] && [ -n "${PAIR_FORCE_TAG:-}" ]; then
bin/pair:675:    forced_tag="$PAIR_FORCE_TAG"
bin/pair:677:unset PAIR_FORCE_TAG
bin/pair:683:# Honors $XDG_DATA_HOME with the spec-defined fallback. (Original site below
bin/pair:685:DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
bin/pair:687:export PAIR_DATA_DIR="$DATA_DIR"
bin/pair:717:    local marker="$HOME/.cache/pair/restart-$SESSION"
bin/pair:727:    local r_tag r_agent r_new_session r_rename_to r_continue
bin/pair:731:    r_rename_to=$(awk -F= '$1=="rename_to"{print $2; exit}' "$marker")
bin/pair:732:    r_continue=$(awk -F= '$1=="continue"{print $2; exit}' "$marker")
bin/pair:735:    [ -z "$r_tag" ] && r_tag="$PAIR_TAG"
bin/pair:738:    # Inside-flow tag rename (#000022 M2). By this point the zellij kill
bin/pair:740:    # so `pair rename` (full check) passes the live-old gate. On failure
bin/pair:741:    # we keep the old tag and continue restarting — the user is not left
bin/pair:742:    # stranded, just without the rename. Loud stderr so they notice.
bin/pair:743:    if [ -n "$r_rename_to" ]; then
bin/pair:744:        if "$0" rename "$r_tag" "$r_rename_to"; then
bin/pair:745:            r_tag="$r_rename_to"
bin/pair:747:            echo "pair: tag rename failed; restarting with original tag '$r_tag'." >&2
bin/pair:760:    export PAIR_FORCE_TAG="$r_tag"
bin/pair:766:        if [ -n "$r_continue" ]; then
bin/pair:767:            # #55 compaction: relaunch seeded from the continuation. `continue`
bin/pair:768:            # + slug come BEFORE the agent (continue's parse order). Args come
bin/pair:770:            # the new_session arm, which never applies the resume-token reorder.
bin/pair:772:            _reexec "$0" continue "$r_continue" "$r_agent" ${r_args:+-- $r_args}
bin/pair:781:    # Default Alt+n: keep config, append the resume token so the new
bin/pair:784:    # `--resume <id>` flag, codex uses `resume <id>`, and agy uses
bin/pair:786:    local resume_extra=""
bin/pair:789:            claude)        resume_extra="--resume $r_sid" ;;
bin/pair:790:            codex)         resume_extra="resume $r_sid" ;;
bin/pair:791:            agy)           resume_extra="--conversation $r_sid" ;;
bin/pair:794:    # Composition order matters for codex: its `resume` subcommand must
bin/pair:797:    # follow. claude's --resume flag works anywhere, so the
bin/pair:800:    if [ "$r_agent" = "codex" ] && [ -n "$resume_extra" ]; then
bin/pair:801:        combined="${resume_extra}${r_args:+ $r_args}"
bin/pair:803:        combined="${r_args}${r_args:+${resume_extra:+ }}${resume_extra}"
bin/pair:816:# `list-sessions`, so the old open-coded `grep -qx` collision checks treated a
bin/pair:820:# Shared by forced resume, the free-slot scan, the name-prompt collision, and
bin/pair:821:# the cmux-owner liveness check (ARCH-DRY). `pair rename` keeps its own
bin/pair:826:    row=$( { zj list-sessions --no-formatting 2>/dev/null || true; } \
bin/pair:841:# declares `"agent": "codex"`, migrate it to the canonical name so the resume
bin/pair:842:# picker can offer native resume. This is a narrow, agent-checked compatibility
bin/pair:863:# no-op outside cmux or if the rename round-trips an error. Hoisted into the
bin/pair:865:# PAIR_TEST_CALL can unit-test cmux_rename_workspace without a live session
bin/pair:869:# records which tag last labeled this workspace. cmux_rename_workspace runs from
bin/pair:883:    [ "$(cat "$owner_file" 2>/dev/null)" = "$PAIR_TAG" ]
bin/pair:886:cmux_rename_workspace() {
bin/pair:892:    printf '%s\n' "$PAIR_TAG" > "$owner_file"
bin/pair:894:    # anywhere in the title. Applied to every cmux rename (launch + the
bin/pair:900:    cmux rename-workspace "$title" >/dev/null 2>&1 || true
bin/pair:914:# Defer the [-z AGENT] default to after DATA_DIR is set: when `resume <tag>`
bin/pair:921:# and `resume <tag>` both override this.
bin/pair:942:# focused pane's env (including ZELLIJ / ZELLIJ_SESSION_NAME / ZELLIJ_PANE_ID)
bin/pair:944:# pair pane inherits stale ZELLIJ_SESSION_NAME and used to trigger this
bin/pair:964:# Agent inference for `pair resume <tag>`: look up what was paired with
bin/pair:975:            [ -f "$cfg" ] || continue
bin/pair:988:# before any session work, so tests/pair-continue-test.sh can assert the
bin/pair:991:    printf 'AGENT=%s\nFORCED_TAG=%s\nAGENT_EXTRA=%s\nCONTINUE_DOC=%s\n' \
bin/pair:996:# In-session compaction (#55): `pair continue <slug>` run from INSIDE its own
bin/pair:1000:# the session — the outer bin/pair then re-execs `pair continue <slug>` fresh
bin/pair:1001:# under the same tag (handle_restart_marker, hoisted above). PAIR_TAG/PAIR_AGENT
bin/pair:1005:# propagates ZELLIJ / ZELLIJ_SESSION_NAME / PAIR_TAG to sibling non-pair panes
bin/pair:1008:# PAIR_FORCE_IN_SESSION forces the branch (bypassing both halves);
bin/pair:1015:    if [ "${PAIR_FORCE_IN_SESSION:-0}" = "1" ]; then
bin/pair:1017:    elif [ "$_in_pane" = "1" ] && [ -n "${PAIR_TAG:-}" ] \
bin/pair:1018:        && [ "${ZELLIJ_SESSION_NAME:-}" = "pair-${PAIR_TAG}" ]; then
bin/pair:1021:    unset PAIR_FORCE_IN_SESSION PAIR_FAKE_IN_ZELLIJ
bin/pair:1023:        : "${PAIR_TAG:?compaction needs PAIR_TAG}"
bin/pair:1024:        _cagent="${PAIR_AGENT:-${AGENT:-claude}}"
bin/pair:1026:            "$PAIR_TAG" >&2
bin/pair:1027:        park_scrollback "$PAIR_TAG" "$_cagent" --copy >/dev/null 2>&1 || true
bin/pair:1030:            printf 'tag=%s\n' "$PAIR_TAG"
bin/pair:1033:            printf 'continue=%s\n' "$_cslug"
bin/pair:1034:        } > "$HOME/.cache/pair/restart-pair-${PAIR_TAG}"
bin/pair:1035:        touch "$HOME/.cache/pair/quit-pair-${PAIR_TAG}"
bin/pair:1037:        exec ${PAIR_KILL_CMD:-zellij kill-session} "pair-${PAIR_TAG}"
bin/pair:1047:# DATA_DIR / PAIR_DATA_DIR are set above (hoisted for #55, just after
bin/pair:1048:# `unset PAIR_FORCE_TAG`) so they precede the zellij gate + compaction branch.
bin/pair:1087:    pkill -9 -f "nvim --embed.*${DATA_DIR}/draft-${tag}\.md\$" >/dev/null 2>&1 || true
bin/pair:1099:            [ -e "$pf" ] || continue
bin/pair:1112:                *nvim*--embed*"$DATA_DIR/draft-"*)
bin/pair:1113:                    tag=${argv##*"$DATA_DIR/draft-"}
bin/pair:1130:        [ -z "$tag" ] && continue
bin/pair:1137:# PAIR_AGENT is the command to exec in the agent pane on create.
bin/pair:1138:# PAIR_AGENT_ARGS is appended on the agent pane's command line at create.
bin/pair:1141:export PAIR_AGENT="$AGENT"
bin/pair:1142:export PAIR_AGENT_ARGS="$agent_extra"
bin/pair:1146:# args block calls `zellij action rename-pane --pane-id $ZELLIJ_PANE_ID
bin/pair:1152:# rename runs only at pane creation, so on resume the title keeps the
bin/pair:1171:# PAIR_TAG is the namespace identifier — used to name the zellij session
bin/pair:1173:# prompt and `resume <tag>` override this.
bin/pair:1174:export PAIR_TAG="$default_tag"
bin/pair:1181:    for old in "$HOME/scratch"/pair-draft-*.md "$HOME/scratch"/pair-log-*.md; do
bin/pair:1182:        [ -e "$old" ] || continue
bin/pair:1194:BASE_TAG="$PAIR_TAG"
bin/pair:1205:# (and BASE_TAG) is only used for the create path's defaults; the picker
bin/pair:1208:# `zellij list-sessions` exits 1 when there are no sessions, and with
bin/pair:1212:all_pair=$( { zj list-sessions --short 2>/dev/null || true; } \
bin/pair:1224:    [ -z "$sess" ] && continue
bin/pair:1225:    if zj list-sessions --no-formatting 2>/dev/null \
bin/pair:1227:        continue   # exited, skip
bin/pair:1234:    # list-sessions; the rest of the pipeline still produces a clean "0" for
bin/pair:1247:# live zellij session. Lets the operator resume an old session by name from
bin/pair:1253:# Scan strategy: walk draft-/log- sidecars (created/touched by nvim's
bin/pair:1260:    for f in "$DATA_DIR"/draft-*.md "$DATA_DIR"/log-*.md; do
bin/pair:1261:        [ -e "$f" ] || continue
bin/pair:1264:            draft-*.md) tag="${name#draft-}"; tag="${tag%.md}" ;;
bin/pair:1265:            log-*.md)   tag="${name#log-}";   tag="${tag%.md}" ;;
bin/pair:1266:            *) continue ;;
bin/pair:1271:            *) continue ;;
bin/pair:1274:        [ "$mt" -lt "$cutoff" ] && continue
bin/pair:1294:# (xterm 232=near-black .. 255=near-white); fzf --ansi honors these. Newest
bin/pair:1311:# queue is visible before you resume it.
bin/pair:1316:        [ -e "$f" ] || continue   # no match → glob stays literal; skip
bin/pair:1346:        cand="pair-${BASE_TAG}"
bin/pair:1348:        cand="pair-${BASE_TAG}-${i}"
bin/pair:1360:historical_colored=()   # age-graded grey — fzf display only
bin/pair:1362:    [ -z "$tag" ] && continue
bin/pair:1366:    if printf '%s\n' "$live_tags_nl" | grep -qx "$tag"; then continue; fi
bin/pair:1372:    # fzf --ansi return still maps back to a historical_rows entry, and the
bin/pair:1412:# tag, or `pair resume <tag>`, already names the session — and a historical
bin/pair:1417:# `pair resume <tag>` short-circuits both the picker and the name prompt:
bin/pair:1419:# that tag. This is the "resume by name" path — pair off the hint printed
bin/pair:1435:    new_label="+ new ${BASE_TAG} session"
bin/pair:1437:    # fzf display only. fzf --ansi strips its color codes from the returned
bin/pair:1444:    if command -v fzf >/dev/null 2>&1; then
bin/pair:1455:        picked=$(printf '%s' "$options_colored" | fzf --ansi --prompt='pair> ' --height=10 --reverse) || exit 0
bin/pair:1478:        # by name, which goes through the same path as `pair resume <tag>`.
bin/pair:1498:    local marker="$HOME/.cache/pair/quit-$SESSION"
bin/pair:1501:        # Capture the agent BEFORE the file is cleared below, so the resume
bin/pair:1504:        quit_agent=$(cat "$DATA_DIR/agent-${PAIR_TAG}" 2>/dev/null || echo "$AGENT")
bin/pair:1505:        zj delete-session "$SESSION" --force >/dev/null 2>&1 || true
bin/pair:1511:        pkill -9 -f "zellij --server .*/${SESSION}\$" >/dev/null 2>&1 || true
bin/pair:1515:        reap_nvim_for_tag "$PAIR_TAG"
bin/pair:1517:        # and Alt+x is about to discard it. Offer to PRESERVE it (rename to a
bin/pair:1522:        _sb_base="$DATA_DIR/scrollback-${PAIR_TAG}-${quit_agent}"
bin/pair:1530:            && [ ! -f "$HOME/.cache/pair/restart-$SESSION" ]; then
bin/pair:1541:            printf 'pair: preserve "%s" scrollback to distill into a continuation later? [y/N] (%ss → N): ' "$SESSION" "$_ptimeout" >/dev/tty
bin/pair:1545:                    # .raw/.events.jsonl rename + parked-<tag> marker (#55 ARCH-DRY).
bin/pair:1549:                    if _pbase="$(park_scrollback "$PAIR_TAG" "$quit_agent")"; then
bin/pair:1551:                        printf 'pair: scrollback preserved at\n        %s.raw\n      open a session and "park %s" to distill it into a continuation.\n' "$_pbase" "$SESSION" >/dev/tty
bin/pair:1560:        rm -f "$DATA_DIR/outer-tty-${PAIR_TAG}" "$DATA_DIR/agent-${PAIR_TAG}" \
bin/pair:1561:              "$DATA_DIR/agent-output-${PAIR_TAG}" \
bin/pair:1563:              "$DATA_DIR/pair-wrap-pid-${PAIR_TAG}" \
bin/pair:1564:              "$DATA_DIR/adapt-${PAIR_TAG}.jsonl" \
bin/pair:1565:              "$DATA_DIR/image-capture-${PAIR_TAG}" \
bin/pair:1566:              "$DATA_DIR/image-capture-${PAIR_TAG}.done"
bin/pair:1571:        # show the user the one-liner that resumes it. config-<tag>-<agent>
bin/pair:1573:        # signal that the resume path will work next time.
bin/pair:1575:        # Display SESSION (e.g. pair-2) rather than PAIR_TAG ("2") to match
bin/pair:1576:        # what the user saw in the UI tab; the `resume` subcommand accepts
bin/pair:1578:        # the resume command — pair infers it from the saved state.
bin/pair:1579:        local resume_cfg
bin/pair:1580:        resume_cfg="$(resolve_config_file "$PAIR_TAG" "$quit_agent")"
bin/pair:1581:        if [ -f "$resume_cfg" ]; then
bin/pair:1584:                "$resume_cfg" 2>/dev/null || true)
bin/pair:1586:                "$SESSION" "$quit_agent"
bin/pair:1587:            printf '      resume with: pair resume %s\n' "$SESSION"
bin/pair:1589:            # wants to drive the agent directly (`claude --resume <id>`,
bin/pair:1590:            # `codex resume <id>`) without going back through pair.
bin/pair:1597:        # so its next 60s tick doesn't race with the cwd-rename below.
bin/pair:1598:        local poller_pid_file="$DATA_DIR/title-pid-${PAIR_TAG}"
bin/pair:1610:        # follow-up re-exec immediately re-renames to the new SESSION,
bin/pair:1621:            cmux_rename_workspace "$reset_title"
bin/pair:1632:#                   resumes the prior conversation via --resume <id> (claude),
bin/pair:1633:#                   --conversation <id> (agy), or `resume <id>` (codex).
bin/pair:1638:# Both flavors pin the same tag via PAIR_FORCE_TAG. The agent name comes
bin/pair:1656:        /dev/*) printf '%s\n' "$outer" > "$DATA_DIR/outer-tty-${PAIR_TAG}" ;;
bin/pair:1657:        *)      rm -f "$DATA_DIR/outer-tty-${PAIR_TAG}" ;;
bin/pair:1671:    "$PAIR_HOME/bin/pair-title.sh" "$PAIR_TAG" "$title_agent" \
bin/pair:1679:    export PAIR_TAG="$chosen_tag"
bin/pair:1680:    SESSION="pair-${PAIR_TAG}"
bin/pair:1681:    DRAFT="$DATA_DIR/draft-${PAIR_TAG}.md"
bin/pair:1686:    printf '\033]0;%s\007' "$SESSION"
bin/pair:1688:    cmux_rename_workspace "$SESSION"
bin/pair:1692:    title_agent=$(cat "$DATA_DIR/agent-$PAIR_TAG" 2>/dev/null | tr -d '\r\n[:space:]')
bin/pair:1695:    zellij --config-dir "$PAIR_HOME/zellij" attach "$SESSION"
bin/pair:1715:# historical pick or `pair resume <tag>` already has its name.
bin/pair:1716:family_prefix="pair-${BASE_TAG}"
bin/pair:1718:[ "$prompt_name" = 1 ] && family_existing=$( { zj list-sessions --no-formatting 2>/dev/null || true; } \
bin/pair:1740:# session, so there's nothing to collide with), and `pair resume <tag>`
bin/pair:1771:    typed_tag="$(normalize_tag "$user_input" "pair: invalid name")" || exit 1
bin/pair:1783:# user whether to reuse the saved args (and resume the agent's session if
bin/pair:1788:# fzf is already a hard dep (used for the session picker). Each option line
bin/pair:1791:# If the user (or Alt+n's restart loop) explicitly passed a resume token
bin/pair:1793:# Same per-agent surface as the explicit_resume detection further below;
bin/pair:1795:_argv_has_resume=0
bin/pair:1800:            if [ "$prev" = "--resume" ] || [ "$prev" = "--conversation" ]; then _argv_has_resume=1; break; fi
bin/pair:1801:            if [[ "$tok" =~ ^--conversation= ]]; then _argv_has_resume=1; break; fi
bin/pair:1807:        [ "$first" = "resume" ] && [ -n "$second" ] && _argv_has_resume=1
bin/pair:1812:if [ -f "$config_file" ] && [ $_argv_has_resume -eq 0 ]; then
bin/pair:1820:    # Defensive strip: pair-session-watch.sh and the inline explicit_resume
bin/pair:1821:    # capture both now scrub --resume <id> (and codex's `resume <id>`)
bin/pair:1824:    # still carry the resume token and we don't want the picker to
bin/pair:1825:    # display "args=[... --resume <id>] / resume=<id>" with the same id
bin/pair:1826:    # duplicated, nor have the "saved (fresh)" option silently resume.
bin/pair:1832:        && [ "${saved_args[0]}" = "resume" ]; then
bin/pair:1836:        if [ "${saved_args[$i]}" = "--resume" ] || [ "${saved_args[$i]}" = "--conversation" ]; then
bin/pair:1846:    # Stale-id check: only offer "resume" if the agent's transcript file
bin/pair:1882:    # truncation. NUL-separated items + `--read0` is fzf's multi-line-render
bin/pair:1883:    # mode (added in 0.46, broadly available in homebrew fzf). Each item
bin/pair:1884:    # spans three lines: label, indented args, indented resume/freshness.
bin/pair:1894:        options+=("$(printf '%d) use saved params + session\n     args=[%s]\n     resume=%s' \
bin/pair:1896:        actions+=("saved+resume")
bin/pair:1905:    # New params + resumed session is only meaningful when there *are* new
bin/pair:1909:        options+=("$(printf '%d) use new params + session\n     args=[%s]\n     resume=%s' \
bin/pair:1911:        actions+=("new+resume")
bin/pair:1924:    fzf_height=$(( ${#options[@]} * 3 + 4 ))
bin/pair:1929:        done | fzf --read0 \
bin/pair:1931:                   --height "$fzf_height" \
bin/pair:1937:    # Map fzf's selection back to the action key. Comparing full multi-line
bin/pair:1948:        saved+resume|new+resume)
bin/pair:1949:            # Compose: (base args stripped of any resume tokens) + the
bin/pair:1950:            # canonical resume binding from saved_session. saved+resume's
bin/pair:1951:            # base is saved_args_clean (already stripped above). new+resume's
bin/pair:1952:            # base is the user-typed agent_extra, which may carry --resume
bin/pair:1954:            # with duplicate resume tokens in the composed argv.
bin/pair:1956:            if [ "$action" = "saved+resume" ]; then
bin/pair:1965:                    && [ "${_new_base[0]}" = "resume" ]; then
bin/pair:1969:                    if [ "${_new_base[$i]}" = "--resume" ] || [ "${_new_base[$i]}" = "--conversation" ]; then
bin/pair:1982:                    # Codex resume is `codex resume <id> [other args]` —
bin/pair:1984:                    agent_extra="resume $saved_session${stripped[*]:+ ${stripped[*]}}"
bin/pair:1987:                    # agy resume is `--conversation <id>`
bin/pair:1992:                    # `--resume <id>` style.
bin/pair:1993:                    agent_extra="${stripped[*]:+${stripped[*]} }--resume $saved_session"
bin/pair:1999:            # --resume baked into the on-disk config must NOT come along
bin/pair:2011:    export PAIR_AGENT_ARGS="$agent_extra"
bin/pair:2014:export PAIR_TAG="$chosen_tag"
bin/pair:2015:SESSION="pair-${PAIR_TAG}"
bin/pair:2016:DRAFT="$DATA_DIR/draft-${PAIR_TAG}.md"
bin/pair:2019:# pair continue <slug>: this is a fresh session — seed the draft to read the
bin/pair:2023:    printf 'Read workshop/continuation/%s and continue from its NEXT ACTION.\n' \
bin/pair:2032:printf '%s\n' "$AGENT" > "$DATA_DIR/agent-${PAIR_TAG}"
bin/pair:2034:# Pre-capture the session id when --resume <id> is explicit on the
bin/pair:2036:# creates; an explicit --resume makes the agent APPEND to an existing
bin/pair:2039:# the start, so a future `pair resume` picker shows the resumed session
bin/pair:2043:#   claude / agy   — flag style: `--resume <id>` / `--conversation <id>` anywhere in argv.
bin/pair:2044:#   codex           — subcommand: `resume <id>` at args[0..1].
bin/pair:2045:explicit_resume=""
bin/pair:2050:            if [ "$prev" = "--resume" ] || [ "$prev" = "--conversation" ]; then
bin/pair:2051:                explicit_resume="$tok"
bin/pair:2055:                explicit_resume="${BASH_REMATCH[1]}"
bin/pair:2063:        if [ "$first" = "resume" ] && [ -n "$second" ]; then
bin/pair:2064:            explicit_resume="$second"
bin/pair:2068:if [ -n "$explicit_resume" ]; then
bin/pair:2069:    # Strip the --resume <id> (or codex's `resume <id>`) we just detected
bin/pair:2071:    # canonical storage for the resume binding. Without this, every
bin/pair:2072:    # --resume launch writes a config whose `args` contains the resume
bin/pair:2080:    if [ "$AGENT" = "codex" ] && [ $n -ge 2 ] && [ "${_eargs[0]}" = "resume" ]; then
bin/pair:2084:        if [ "${_eargs[$i]}" = "--resume" ] || [ "${_eargs[$i]}" = "--conversation" ]; then
bin/pair:2093:    config_path="$(resolve_config_file "$PAIR_TAG" "$AGENT")"
bin/pair:2097:            --arg sid "$explicit_resume" \
bin/pair:2112:#   - explicit_resume already pinned a session above;
bin/pair:2117:if [ "$AGENT" = "claude" ] && [ -z "$explicit_resume" ]; then
bin/pair:2142:            export PAIR_AGENT_ARGS="$agent_extra"
bin/pair:2143:            config_path="$(resolve_config_file "$PAIR_TAG" "$AGENT")"
bin/pair:2145:            # is the canonical storage for the resume binding, and leaving
bin/pair:2147:            # picker (same logic as the explicit_resume strip above).
bin/pair:2184:# Append (not prepend) keeps the `resume` subcommand at args[0] when
bin/pair:2190:        if [ "$tok" = "--no-alt-screen" ]; then continue; fi
bin/pair:2198:    export PAIR_AGENT_ARGS="$agent_extra"
bin/pair:2202:# its file set per-session (#63): a fresh session opens an empty log, a resume
bin/pair:2204:# resume pins it ($explicit_resume) — the Alt+n restart re-execs with --resume,
bin/pair:2205:# so it lands in the explicit_resume arm too. codex/agy fresh sessions discover
bin/pair:2206:# the id async via pair-session-watch.sh, so it is empty here; pair-changelog-open
bin/pair:2209:export PAIR_SESSION_ID="${explicit_resume:-${new_sid:-}}"
bin/pair:2218:: > "$DATA_DIR/adapt-${PAIR_TAG}.jsonl" 2>/dev/null || true
bin/pair:2226:# intentionally word-split, matching the existing PAIR_AGENT_ARGS rule.
bin/pair:2227:"$PAIR_HOME/bin/pair-session-watch.sh" "$AGENT" "$PAIR_TAG" "$PWD" $agent_extra \
bin/pair:2233:printf '\033]0;%s\007' "$SESSION"
bin/pair:2235:cmux_rename_workspace "$SESSION"
bin/pair:2254:# message. Covers every create path — `continue`, `resume`, a manually-typed
bin/pair:2261:_zj_probe="$(zj --session "$SESSION" action list-clients 2>&1 || true)"
bin/pair:2264:        echo "pair: tag '$PAIR_TAG' makes zellij's session name too long for this" >&2
bin/pair:2265:        echo "      machine's socket path ($SESSION). Pick a shorter tag." >&2
bin/pair:2277:    --session                 "$SESSION"
cmd/pair-go/main_test.go:55:	code := runWithLauncherRuntime([]string{"launch", "resume", "demo"}, &stdout, &stderr, testLauncherRuntime("/home/me", "", "/work/pair"))
cmd/internal/dispatcher/dispatcher.go:129:       %s launch resume <tag>
cmd/internal/dispatcher/dispatcher.go:149:	xdg := os.Getenv("XDG_DATA_HOME")
cmd/pair-go/launch_process_test.go:27:  "list-sessions --short") printf 'pair-live\npair-detached\npair-exited\n' ;;
cmd/pair-go/launch_process_test.go:28:  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-exited [Created] (EXITED - attach to resurrect)\n' ;;
cmd/pair-go/launch_process_test.go:38:	historical := filepath.Join(data, "pair", "draft-pair-old.md")
cmd/pair-go/launch_process_test.go:49:	t.Setenv("XDG_DATA_HOME", data)
cmd/internal/dispatcher/dispatcher_test.go:84:	res := DispatchWithLauncherRuntime([]string{"launch", "resume", "demo"}, LauncherRuntime{
cmd/internal/launcher/datadir_test.go:13:		{name: "home", home: "/home/me", want: "/home/me/.local/share/pair"},
cmd/internal/launcher/zellij_test.go:16:  "list-sessions --short") printf 'pair-live\npair-detached\npair-gone\nother\n' ;;
cmd/internal/launcher/zellij_test.go:17:  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-gone [Created] (EXITED - attach to resurrect)\n' ;;
cmd/internal/launcher/args.go:33:	case "continue", "rename", "list", "ls":
cmd/internal/launcher/args.go:35:	case "resume":
cmd/internal/launcher/args.go:37:			return LaunchArgs{}, UsageError{Message: "pair-go launch: 'resume' requires a tag"}
cmd/internal/launcher/args.go:54:			continue
cmd/internal/launcher/args.go:58:			continue
cmd/internal/launcher/args.go:62:			continue
cmd/internal/launcher/tag.go:17:			continue
cmd/internal/launcher/zellij.go:16:	short, err := s.run("list-sessions", "--short")
cmd/internal/launcher/zellij.go:20:	raw, err := s.run("list-sessions", "--no-formatting")
cmd/internal/launcher/zellij.go:28:			continue
cmd/internal/launcher/zellij.go:73:			continue
cmd/internal/launcher/run_test.go:46:	if outcome.Env.DataDir != "/home/me/.local/share/pair" {
cmd/internal/launcher/history.go:18:	for _, pattern := range []string{"draft-*.md", "log-*.md"} {
cmd/internal/launcher/history.go:26:				continue
cmd/internal/launcher/history.go:30:				continue
cmd/internal/launcher/history.go:33:				continue
cmd/internal/launcher/history.go:54:	case strings.HasPrefix(name, "draft-") && strings.HasSuffix(name, ".md"):
cmd/internal/launcher/history.go:55:		return strings.TrimSuffix(strings.TrimPrefix(name, "draft-"), ".md"), true
cmd/internal/launcher/history.go:56:	case strings.HasPrefix(name, "log-") && strings.HasSuffix(name, ".md"):
cmd/internal/launcher/history.go:57:		return strings.TrimSuffix(strings.TrimPrefix(name, "log-"), ".md"), true
cmd/internal/launcher/args_test.go:51:	args, err := ParseArgs([]string{"resume", "pair-demo"})
cmd/internal/launcher/args_test.go:56:		t.Fatalf("Agent = %q, want empty for resume inference", args.Agent)
cmd/internal/launcher/args_test.go:77:	for _, verb := range []string{"continue", "rename", "list"} {
tests/pair-rename.sh:2:# Integration test for `pair rename <old> <new>` (issue #000022, M1).
tests/pair-rename.sh:4:# Builds a fixtured $PAIR_DATA_DIR with representative tag-scoped files,
tests/pair-rename.sh:5:# exercises the rename CLI, and verifies post-state. Exits 0 on pass,
tests/pair-rename.sh:8:# Run:  bash tests/pair-rename.sh
tests/pair-rename.sh:33:# the families enumerated in bin/pair's `rename_paths_for` helper.
tests/pair-rename.sh:43:    : > "$dd/draft-$tag.md"
tests/pair-rename.sh:44:    : > "$dd/log-$tag.md"
tests/pair-rename.sh:56:# returns no sessions. (The real zellij would refuse the rename if a
tests/pair-rename.sh:68:# Run `pair rename` in an isolated $PAIR_DATA_DIR with the zellij shim.
tests/pair-rename.sh:70:run_rename() {
tests/pair-rename.sh:73:    shim="$(mktemp -d "${TMPDIR:-/tmp}/pair-rename-shim.XXXXXX")"
tests/pair-rename.sh:75:    PAIR_DATA_DIR="$dd" PATH="$shim:$PATH" "$PAIR_BIN" rename "$@"
tests/pair-rename.sh:92:# ── T1: clean rename ─────────────────────────────────────────────────────────
tests/pair-rename.sh:93:case_begin "T1 clean rename"
tests/pair-rename.sh:94:DD="$(mktemp -d "${TMPDIR:-/tmp}/pair-rename-t1.XXXXXX")"
tests/pair-rename.sh:96:run_rename "$DD" t1 t2 >/dev/null
tests/pair-rename.sh:97:for f in agent-t1 agent-pid-t1 outer-tty-t1 draft-t1.md log-t1.md \
tests/pair-rename.sh:102:for f in agent-t2 agent-pid-t2 outer-tty-t2 draft-t2.md log-t2.md \
tests/pair-rename.sh:115:DD="$(mktemp -d "${TMPDIR:-/tmp}/pair-rename-t2.XXXXXX")"
tests/pair-rename.sh:119:run_rename "$DD" brain new-brain >/dev/null
tests/pair-rename.sh:121:for f in agent-brain-2 outer-tty-brain-2 draft-brain-2.md \
tests/pair-rename.sh:138:DD="$(mktemp -d "${TMPDIR:-/tmp}/pair-rename-t3.XXXXXX")"
tests/pair-rename.sh:141:assert_exits_nonzero run_rename "$DD" src dst
tests/pair-rename.sh:149:DD="$(mktemp -d "${TMPDIR:-/tmp}/pair-rename-t4.XXXXXX")"
tests/pair-rename.sh:150:assert_exits_nonzero run_rename "$DD" ghost newname
tests/pair-rename.sh:155:DD="$(mktemp -d "${TMPDIR:-/tmp}/pair-rename-t5.XXXXXX")"
tests/pair-rename.sh:157:assert_exits_nonzero run_rename "$DD" t1 t1
tests/pair-rename.sh:162:DD="$(mktemp -d "${TMPDIR:-/tmp}/pair-rename-t6.XXXXXX")"
tests/pair-rename.sh:164:assert_exits_nonzero run_rename "$DD" t1 'bad name'
tests/pair-rename.sh:165:assert_exits_nonzero run_rename "$DD" t1 'bad/slash'
tests/pair-rename.sh:166:assert_exits_nonzero run_rename "$DD" '' newname
tests/pair-rename.sh:173:DD="$(mktemp -d "${TMPDIR:-/tmp}/pair-rename-t7.XXXXXX")"
tests/pair-rename.sh:175:run_rename "$DD" pair-t1 pair-t2 >/dev/null
tests/pair-rename.sh:181:case_begin "T8 renames all agents under one tag"
tests/pair-rename.sh:182:DD="$(mktemp -d "${TMPDIR:-/tmp}/pair-rename-t8.XXXXXX")"
tests/pair-rename.sh:191:run_rename "$DD" multi renamed >/dev/null
tests/pair-rename.sh:194:    assert_exists "$DD/config-renamed-$a.json"
tests/pair-rename.sh:196:    assert_exists "$DD/scrollback-renamed-$a.raw"
tests/pair-rename.sh:202:DD="$(mktemp -d "${TMPDIR:-/tmp}/pair-rename-t9.XXXXXX")"
tests/pair-rename.sh:204:run_rename "$DD" --restart-check src dst >/dev/null
tests/pair-rename.sh:210:assert_exits_nonzero run_rename "$DD" --restart-check src dst
tests/pair-rename.sh:215:printf 'pair rename: %s passed, %s failed\n' "$(green "$pass")" "$([ "$fail" -eq 0 ] && green "$fail" || red "$fail")"
cmd/internal/launcher/decision.go:21:// DecideLaunch decides the launch action without touching zellij, fzf, or disk.
cmd/internal/launcher/decision.go:62:			continue
cmd/internal/launcher/history_test.go:13:	for _, name := range []string{"draft-pair.md", "log-pair-old.md", "draft-other.md"} {
tests/queue-send-test.sh:10:# the item was resolved by a stale display index after the draft-enqueue
tests/queue-send-test.sh:27:# Reset the per-tag fixture. Args: draft-body, then queue item bodies (front→back).
tests/queue-send-test.sh:31:  printf '%s' "$1" > "$RT/draft-test.md"
tests/queue-send-test.sh:32:  : > "$RT/log-test.md"
tests/queue-send-test.sh:46:    env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/queue-send-test.sh:47:    nvim --headless -u "$INIT" "$RT/draft-test.md" \
tests/queue-send-test.sh:55:local dd = os.getenv('PAIR_DATA_DIR')
tests/queue-send-test.sh:69:local lh=io.open(dd..'/log-test.md'); local lc=lh and lh:read('*a') or ''; if lh then lh:close() end
tests/queue-send-test.sh:73:local d=io.open(dd..'/draft-test.md'); O:write('D '..((d and d:read('*a') or ''):gsub('%s+\$',''))..'\n'); if d then d:close() end
tests/cmux-ownership-test.sh:2:# Regression test for bin/pair's cmux_rename_workspace ownership claim.
tests/cmux-ownership-test.sh:12:# fakes (HOME/XDG_DATA_HOME pinned, fake `cmux` + `zellij` on PATH), exactly
tests/cmux-ownership-test.sh:14:# tests/pair-continue-test.sh.
tests/cmux-ownership-test.sh:31:# Fake `cmux`: record every invocation's args so we can assert the rename fired
tests/cmux-ownership-test.sh:32:# (and with what title). `command -v cmux` must also succeed for the rename path.
tests/cmux-ownership-test.sh:40:# would have deferred on this. (cmux_rename_workspace no longer consults it; the
tests/cmux-ownership-test.sh:45:  "list-sessions --short") printf 'pair-other\npair-211\n' ;;
tests/cmux-ownership-test.sh:51:run_rename() {  # <tag> <title-arg>
tests/cmux-ownership-test.sh:53:    env HOME="$RT" XDG_DATA_HOME="$RT/xdg" PATH="$RT/bin:$PATH" \
tests/cmux-ownership-test.sh:54:        CMUX_WORKSPACE_ID="$WS" PAIR_TAG="$tag" \
tests/cmux-ownership-test.sh:55:        PAIR_TEST_CALL=cmux_rename_workspace PAIR_TEST_ARGS="$title" \
tests/cmux-ownership-test.sh:63:#    asked to rename. This is the core regression.
tests/cmux-ownership-test.sh:66:run_rename 211 "pair-211"
tests/cmux-ownership-test.sh:67:if [ "$(cat "$OWNER_FILE")" = "211" ] && grep -q 'rename-workspace' "$RT/cmux.log"; then
tests/cmux-ownership-test.sh:73:# 2. The rename applies the personal display substitution (pair → ♋).
tests/cmux-ownership-test.sh:74:if grep -q 'rename-workspace ♋-211' "$RT/cmux.log"; then
tests/cmux-ownership-test.sh:77:    fail "expected 'rename-workspace ♋-211', got: $(cat "$RT/cmux.log")"
tests/cmux-ownership-test.sh:82:run_rename 211 "pair-211"
tests/cmux-ownership-test.sh:83:if [ "$(cat "$OWNER_FILE")" = "211" ] && grep -q 'rename-workspace' "$RT/cmux.log"; then
tests/cmux-ownership-test.sh:89:# 4. Outside cmux (no CMUX_WORKSPACE_ID) → silent no-op, no rename, no owner file.
tests/cmux-ownership-test.sh:94:env -u CMUX_WORKSPACE_ID HOME="$RT" XDG_DATA_HOME="$RT/xdg" PATH="$RT/bin:$PATH" \
tests/cmux-ownership-test.sh:95:    PAIR_TAG=211 PAIR_TEST_CALL=cmux_rename_workspace PAIR_TEST_ARGS="pair-211" \
tests/review-poke-test.sh:49:PATH="$RT/bin:$PATH" PAIR_ROOT="$ROOT" RESULT="$RESULT" PAIR_DATA_DIR="$RT" PAIR_TAG=poke \
tests/copy-on-select-test.sh:55:  "terminal_command":"sh -c zellij action rename-pane --pane-id \"$ZELLIJ_PANE_ID\" \"${PAIR_PANE_TITLE:-agent}\" 2>/dev/null; exec pair-wrap --scrollback-log \"/data/scrollback-t-claude.raw\" claude"}'
tests/copy-on-select-test.sh:59:  "terminal_command":"sh -c export PAIR_NVIM_PID_FILE=\"/data/nvim-pid-t-draft\" && exec nvim -u \"$PAIR_HOME/nvim/init.lua\" \"/data/draft-t.md\""}'
tests/copy-on-select-test.sh:79:reached && { echo "FAIL (b) draft-pane selection handed off (would self-insert)"; fail=1; }
tests/review-resume-test.sh:2:# tests/review-resume-test.sh — reconstruct-on-open (#66 M4a' resume): opening a
tests/review-resume-test.sh:9:# Run: bash tests/review-resume-test.sh
tests/review-resume-test.sh:13:RT="${TMPDIR:-/tmp}/pair-resume-test.$$"; mkdir -p "$RT"
tests/review-resume-test.sh:32:  new_occurrence = 1, explain = 'kept — the example why for resume' } }
tests/review-resume-test.sh:56:case "$res" in *hl=0*|'') fail "no change-highlights on resume ($res)";; *) pass "change-highlights repainted from commit";; esac
tests/review-resume-test.sh:57:case "$res" in *diag=0*|'') fail "no diagnostics on resume ($res)";; *) pass "diagnosis repainted from commit";; esac
tests/review-resume-test.sh:59:[ "$fails" -eq 0 ] || { printf 'review-resume-test FAILED (%d)\n' "$fails"; exit 1; }
tests/review-resume-test.sh:60:printf 'review-resume-test ok\n'
tests/adapt-golden.expected:1:{"ts":"TS","comp":"golden","agent":"codex","aspect":2,"signal":"overlay-detect","outcome":"near-miss","detail":"press > to continue? (y/n)"}
tests/statusline-pos-test.sh:25:printf 'my draft text' > "$RT/draft-test.md"
tests/statusline-pos-test.sh:26:printf '## 2026-01-01 00:00\n\nfirst entry\n\n---\n\n## 2026-01-02 00:00\n\nsecond entry' > "$RT/log-test.md"
tests/statusline-pos-test.sh:33:local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
tests/statusline-pos-test.sh:68:  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/statusline-pos-test.sh:69:  nvim --headless -u "$INIT" "$RT/draft-test.md" \
tests/review-readiness-cli-test.sh:38:prep_out="$(PAIR_HOME="$ROOT" PAIR_DATA_DIR="$RT" PAIR_TAG=prep PAIR_SESSION_ID=sid \
tests/review-handoff-test.sh:4:# the file. XDG_DATA_HOME is redirected so the handoff lands in a temp dir.
tests/review-handoff-test.sh:53:PAIR_ROOT="$ROOT" RESULT="$RESULT" XDG_DATA_HOME="$RT/xdg" \
tests/review-toggle-test.sh:2:# tests/review-toggle-test.sh — the Alt+c review/collaboration toggle, now a draft-nvim lua
tests/review-toggle-test.sh:64:local sf = vim.env.PAIR_DATA_DIR .. '/review-' .. vim.env.PAIR_TAG .. '.open'
tests/review-toggle-test.sh:89:local target = vim.env.PAIR_DATA_DIR .. '/review-target-' .. vim.env.PAIR_TAG .. '.json'
tests/review-toggle-test.sh:90:local draft = vim.env.PAIR_DATA_DIR .. '/draft.md' -- exists (the test wrote it)
tests/review-toggle-test.sh:93:-- (PAIR_SESSION_ID=oldsid, pre-written below) is ignored by this session (testsid),
tests/review-toggle-test.sh:97:local prepbin = vim.env.PAIR_DATA_DIR .. '/prep-ok'
tests/review-toggle-test.sh:123:vim.env.PAIR_SESSION_ID = ''
tests/review-toggle-test.sh:124:vim.env.PAIR_AGENT = 'claude'
tests/review-toggle-test.sh:125:os.remove(vim.env.PAIR_DATA_DIR .. '/config-' .. vim.env.PAIR_TAG .. '-claude.json')
tests/review-toggle-test.sh:131:vim.env.PAIR_SESSION_ID = 'testsid'
tests/review-toggle-test.sh:135:-- fall back to config-<tag>-<agent>.json when PAIR_SESSION_ID is empty.
tests/review-toggle-test.sh:136:vim.env.PAIR_SESSION_ID = ''
tests/review-toggle-test.sh:138:  vim.env.PAIR_DATA_DIR .. '/config-' .. vim.env.PAIR_TAG .. '-' .. vim.env.PAIR_AGENT .. '.json')
tests/review-toggle-test.sh:144:vim.env.PAIR_SESSION_ID = 'testsid'
tests/review-toggle-test.sh:147:vim.env.PAIR_SESSION_ID = ''
tests/review-toggle-test.sh:148:vim.env.PAIR_AGENT = 'codex'
tests/review-toggle-test.sh:149:os.remove(vim.env.PAIR_DATA_DIR .. '/config-' .. vim.env.PAIR_TAG .. '-codex.json')
tests/review-toggle-test.sh:150:vim.fn.writefile({ '111' }, vim.env.PAIR_DATA_DIR .. '/agent-pid-' .. vim.env.PAIR_TAG)
tests/review-toggle-test.sh:156:vim.env.PAIR_AGENT = 'claude'
tests/review-toggle-test.sh:157:vim.env.PAIR_SESSION_ID = 'testsid'
tests/review-toggle-test.sh:199:# runs as PAIR_SESSION_ID=testsid, so read_target must ignore it (a fresh session
tests/review-toggle-test.sh:200:# prompts; an Alt+n resume — same id — would keep its target). (#66 smoke #6.)
tests/review-toggle-test.sh:203:    PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" PAIR_SESSION_ID=testsid \
tests/dev-rebuild-test.sh:9:# failure is errexit-safe (dev_rebuild returns 0 and the launcher continues —
tests/dev-rebuild-test.sh:57:#    returns 0 and the caller continues to launch with last-good binaries.
tests/dev-rebuild-test.sh:61:    pass "build failure is errexit-safe (launcher continues)"
tests/changelog-notify-test.sh:7:#   (2) the marker-poll timer turns a dropped changelog-<tag>-<agent>.ready
tests/changelog-notify-test.sh:11:# This is the draft-side half of the build-complete signal: the operator
tests/changelog-notify-test.sh:15:# Run: bash tests/changelog-notify-test.sh   (also wired into `make test`)
tests/changelog-notify-test.sh:21:RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-changelog-notify-test.XXXXXX")"
tests/changelog-notify-test.sh:24:printf 'draft text' > "$RT/draft-test.md"
tests/changelog-notify-test.sh:27:local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
tests/changelog-notify-test.sh:54:  -- (PAIR_SESSION_ID -> per-tag config -> none) on each tick and polls the
tests/changelog-notify-test.sh:55:  -- matching changelog-<tag>-<agent>[-<sid>].ready. We drive all three branches
tests/changelog-notify-test.sh:59:  local dd = os.getenv('PAIR_DATA_DIR')
tests/changelog-notify-test.sh:68:  vim.env.PAIR_SESSION_ID = nil
tests/changelog-notify-test.sh:69:  drop('changelog-test-claude.ready')
tests/changelog-notify-test.sh:70:  emit(consumed('changelog-test-claude.ready'), 'legacy (no-id) marker consumed')
tests/changelog-notify-test.sh:73:  -- (2b) PAIR_SESSION_ID set -> keyed marker (the claude-fresh / resume path).
tests/changelog-notify-test.sh:75:  vim.env.PAIR_SESSION_ID = SID
tests/changelog-notify-test.sh:76:  drop('changelog-test-claude-' .. SID .. '.ready')
tests/changelog-notify-test.sh:77:  emit(consumed('changelog-test-claude-' .. SID .. '.ready'), 'env-keyed marker consumed')
tests/changelog-notify-test.sh:81:  vim.env.PAIR_SESSION_ID = nil
tests/changelog-notify-test.sh:85:  drop('changelog-test-claude-' .. CSID .. '.ready')
tests/changelog-notify-test.sh:86:  emit(consumed('changelog-test-claude-' .. CSID .. '.ready'), 'config-resolved keyed marker consumed')
tests/changelog-notify-test.sh:94:  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/changelog-notify-test.sh:95:  nvim --headless -u "$INIT" "$RT/draft-test.md" \
tests/changelog-notify-test.sh:98:echo "changelog-notify-test:"
tests/changelog-notify-test.sh:112:  echo "changelog-notify-test: $fails failure(s)"
tests/changelog-notify-test.sh:115:echo "changelog-notify-test: all passed"
tests/pair-review-target-test.sh:14:PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=codex \
tests/pair-review-target-test.sh:15:  PAIR_SESSION_ID=envsid "$ROOT/bin/pair-review-target" "$doc" ready >/dev/null
tests/pair-review-target-test.sh:17:[ "$got" = envsid ] && pass "uses PAIR_SESSION_ID when set" || fail "env session stamp ($got)"
tests/pair-review-target-test.sh:20:PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=codex \
tests/autopair-test.sh:31:local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
tests/autopair-test.sh:69:  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/adapt-schema-test.sh:9:#   1. Golden: all three emit byte-identical lines (ts normalized) for the same
tests/adapt-schema-test.sh:26:export PAIR_TAG=golden PAIR_DATA_DIR="$RT" PAIR_AGENT=codex
tests/adapt-schema-test.sh:27:DETAIL='press > to continue? (y/n)'
tests/adapt-schema-test.sh:28:norm() { jq -c '.ts="TS"'; } # normalize the only field that legitimately varies
tests/adapt-schema-test.sh:41:# Emits into the same file (PAIR_TAG=golden); it appends, so read the last line.
tests/adapt-schema-test.sh:60:PAIR_TAG=conc
tests/adapt-schema-test.sh:64:      PAIR_TAG=conc adapt_log golden codex 1 return-remap fired "writer-$i" ) &
tests/changelog-open-test.sh:2:# End-to-end smoke test for bin/pair-changelog-open (#53/#58).
tests/changelog-open-test.sh:15:    echo "SKIP changelog-open-test: build the binaries first (make pair-changelog pair-scrollback-render)"
tests/changelog-open-test.sh:20:tmp=$(mktemp -d "${TMPDIR:-/tmp}/pair-changelog-test.XXXXXX")
tests/changelog-open-test.sh:23:export PAIR_DATA_DIR="$tmp/data"
tests/changelog-open-test.sh:24:export PAIR_TAG="t"
tests/changelog-open-test.sh:25:export PAIR_AGENT="claude"
tests/changelog-open-test.sh:26:mkdir -p "$PAIR_DATA_DIR"
tests/changelog-open-test.sh:30:    > "$PAIR_DATA_DIR/scrollback-t-claude.raw"
tests/changelog-open-test.sh:32:    > "$PAIR_DATA_DIR/scrollback-t-claude.events.jsonl"
tests/changelog-open-test.sh:50:"$PAIR_HOME/bin/pair-changelog-open"
tests/changelog-open-test.sh:53:DLOCK="$PAIR_DATA_DIR/changelog-t-claude.distill.lock"
tests/changelog-open-test.sh:54:LOG="$PAIR_DATA_DIR/changelog-t-claude.md"
tests/changelog-open-test.sh:55:ANCHOR="$PAIR_DATA_DIR/changelog-t-claude.anchor"
tests/changelog-open-test.sh:60:        if [ -n "${p:-}" ] && kill -0 "$p" 2>/dev/null; then i=$((i + 1)); sleep 1; continue; fi
tests/changelog-open-test.sh:69:grep -q 'changelog-t-claude.md' "$tmp/nvim-args" 2>/dev/null || { echo "FAIL: nvim not opened on the log; args: $(cat "$tmp/nvim-args" 2>/dev/null)"; fail=1; }
tests/changelog-open-test.sh:70:[ -f "$PAIR_DATA_DIR/changelog-t-claude.openlock" ] && { echo "FAIL: openlock not cleared on viewer exit"; fail=1; }
tests/changelog-open-test.sh:73:    echo "PASS changelog-open-test"
tests/review-indicator-test.sh:54:PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" \
tests/lib/fake-review-agent.sh:14:# Runs in the doc's repo (cwd), with DOCFLOW_BIN + XDG_DATA_HOME from the caller.
tests/lib/fake-review-agent.sh:19:dir="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
tests/pair-session-watch-test.sh:33:PAIR_DATA_DIR="$RT/data" \
tests/pair-session-watch-test.sh:34:PAIR_TAG=test \
tests/pair-session-watch-test.sh:35:PAIR_SESSION_WATCH_PID_WAIT_SECONDS=3 \
tests/changelog-session-key-test.sh:2:# Focused keying test for bin/pair-changelog-open (#63): the change-log base is
tests/changelog-session-key-test.sh:3:# keyed on the resolved session id (PAIR_SESSION_ID -> config -> none). No model
tests/changelog-session-key-test.sh:10:tmp=$(mktemp -d "${TMPDIR:-/tmp}/pair-changelog-key.XXXXXX"); trap 'rm -rf "$tmp"' EXIT
tests/changelog-session-key-test.sh:11:export PAIR_DATA_DIR="$tmp/data" PAIR_TAG=t PAIR_AGENT=claude
tests/changelog-session-key-test.sh:12:mkdir -p "$PAIR_DATA_DIR"
tests/changelog-session-key-test.sh:25:run() { rm -f "$tmp/nvim-arg"; "$PAIR_HOME/bin/pair-changelog-open"; }
tests/changelog-session-key-test.sh:31:# (a) PAIR_SESSION_ID set -> keyed base
tests/changelog-session-key-test.sh:32:PAIR_SESSION_ID="$A" run
tests/changelog-session-key-test.sh:33:case "$(opened)" in *"changelog-t-claude-$A.md") ;;
tests/changelog-session-key-test.sh:37:printf 'old log\n' > "$PAIR_DATA_DIR/changelog-t-claude-$A.md"
tests/changelog-session-key-test.sh:38:PAIR_SESSION_ID="$B" run
tests/changelog-session-key-test.sh:39:case "$(opened)" in *"changelog-t-claude-$B.md") ;;
tests/changelog-session-key-test.sh:41:[ -s "$PAIR_DATA_DIR/changelog-t-claude-$B.md" ] \
tests/changelog-session-key-test.sh:44:# (c) resume = same id -> same file, prior content intact
tests/changelog-session-key-test.sh:45:PAIR_SESSION_ID="$A" run
tests/changelog-session-key-test.sh:46:grep -q 'old log' "$PAIR_DATA_DIR/changelog-t-claude-$A.md" \
tests/changelog-session-key-test.sh:47:  || { echo "FAIL (c) resume lost prior content"; fail=1; }
tests/changelog-session-key-test.sh:50:unset PAIR_SESSION_ID
tests/changelog-session-key-test.sh:52:  > "$PAIR_DATA_DIR/config-t-claude.json"
tests/changelog-session-key-test.sh:54:case "$(opened)" in *"changelog-t-claude-$C.md") ;;
tests/changelog-session-key-test.sh:58:rm -f "$PAIR_DATA_DIR/config-t-claude.json"
tests/changelog-session-key-test.sh:60:case "$(opened)" in *"changelog-t-claude.md") ;;
tests/changelog-session-key-test.sh:64:  echo "PASS changelog-session-key-test"
tests/zellij-trace-test.sh:43:PAIR_DATA_DIR="$RT" \
tests/zellij-trace-test.sh:44:PAIR_TAG=trace \
tests/pair-title-poller-test.sh:12:# Part B (#71) — update_frame_titles(): renames each agent pane's zellij frame
tests/pair-title-poller-test.sh:13:# to "<agent> (<count>) [<cwd>]" and skips redundant renames.
tests/pair-title-poller-test.sh:94:# Fake zellij records rename-pane calls. The poller calls
tests/pair-title-poller-test.sh:95:#   `zellij --session <s> action rename-pane --pane-id <id> <title>`
tests/pair-title-poller-test.sh:96:# so $1=--session $2=<s> $3=action $4=rename-pane.
tests/pair-title-poller-test.sh:99:[ "${4:-}" = "rename-pane" ] && printf '%s\n' "$*" >> "$RENAME_LOG"
tests/pair-title-poller-test.sh:116:    : > "$RT/rename.log"
tests/pair-title-poller-test.sh:117:    PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$DD" RENAME_LOG="$RT/rename.log" FAKE_COUNT="$count" \
tests/pair-title-poller-test.sh:122:# 6. One tick with a count → one rename with "claude (970k) [~/repo]" for pane 7.
tests/pair-title-poller-test.sh:124:if grep -q -- "--pane-id 7 claude (970k) \[~/repo\]" "$RT/rename.log"; then
tests/pair-title-poller-test.sh:127:    fail "frame title missing/wrong: $(cat "$RT/rename.log")"
tests/pair-title-poller-test.sh:132:if grep -q -- "--pane-id 7 claude \[~/repo\]" "$RT/rename.log" && ! grep -q "(" "$RT/rename.log"; then
tests/pair-title-poller-test.sh:135:    fail "no-count fallback wrong: $(cat "$RT/rename.log")"
tests/pair-title-poller-test.sh:138:# 8. Two ticks, same state → unchanged-skip: exactly ONE rename emitted.
tests/pair-title-poller-test.sh:140:n=$(grep -c "rename-pane" "$RT/rename.log")
tests/pair-title-poller-test.sh:142:    pass "unchanged title is renamed once, not twice (skip guard)"
tests/pair-title-poller-test.sh:144:    fail "expected 1 rename across two identical ticks, got $n"
tests/review-loop-test.sh:119:    XDG_DATA_HOME="$RT/xdg" DOCFLOW_BIN="$ROOT/tests/lib/fake-docflow.sh" \
tests/pair-continue-test.sh:2:# Regression test for `pair continue` arg-parsing + the launch-time session-
tests/pair-continue-test.sh:8:#   1. `continue <slug>` does NOT force the tag (forced_tag empty → the
tests/pair-continue-test.sh:18:# Run: bash tests/pair-continue-test.sh   (also wired into `make test`)
tests/pair-continue-test.sh:23:RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-continue-test.XXXXXX")"
tests/pair-continue-test.sh:43:# XDG_DATA_HOME is pinned into the sandbox so the probe never reads/writes the
tests/pair-continue-test.sh:44:# operator's real pair data dir. cwd = RT so `continue` finds the fixture.
tests/pair-continue-test.sh:47:  ( cd "$RT" && XDG_DATA_HOME="$RT/xdg" PAIR_DEBUG_ARGS=1 "$PAIR" "$@" 2>/dev/null ) \
tests/pair-continue-test.sh:52:[ "$(probe FORCED_TAG continue demo)" = "" ] \
tests/pair-continue-test.sh:53:  && pass "continue: tag not forced (prompts like a normal create)" \
tests/pair-continue-test.sh:54:  || fail "continue: tag should not be forced"
tests/pair-continue-test.sh:55:[ "$(probe AGENT continue demo)" = "claude" ] \
tests/pair-continue-test.sh:56:  && pass "continue: agent from doc frontmatter" || fail "continue: agent should come from doc"
tests/pair-continue-test.sh:57:[ -n "$(probe CONTINUE_DOC continue demo)" ] \
tests/pair-continue-test.sh:58:  && pass "continue: slug resolved to the doc" || fail "continue: doc not resolved"
tests/pair-continue-test.sh:61:[ "$(probe AGENT_EXTRA continue demo -- --dangerously-skip-permissions)" = "--dangerously-skip-permissions" ] \
tests/pair-continue-test.sh:62:  && pass "continue: forwards -- args" || fail "continue: -- args dropped"
tests/pair-continue-test.sh:63:[ "$(probe AGENT_EXTRA continue demo claude -- --foo bar)" = "--foo bar" ] \
tests/pair-continue-test.sh:64:  && pass "continue: forwards -- args after an agent port" || fail "continue: -- args after port dropped"
tests/pair-continue-test.sh:67:[ "$(probe AGENT continue demo codex)" = "codex" ] \
tests/pair-continue-test.sh:68:  && pass "continue: [agent] port overrides the doc" || fail "continue: port not honored"
tests/pair-continue-test.sh:71:( cd "$RT" && "$PAIR" continue 2>&1 | grep -q 'demo' ) \
tests/pair-continue-test.sh:73:( cd "$RT" && "$PAIR" continue nope >/dev/null 2>&1 ); [ $? -eq 1 ] \
tests/pair-continue-test.sh:75:( cd "$RT" && "$PAIR" continue 'bad/slug' >/dev/null 2>&1 ); [ $? -eq 1 ] \
tests/pair-continue-test.sh:88:LOUT="$( cd "$RT" && "$PAIR" continue 2>&1 )"
tests/pair-continue-test.sh:116:# PAIR_FORCE_IN_SESSION / PAIR_FAKE_IN_ZELLIJ / PAIR_KILL_CMD / PAIR_TEST_CALL /
tests/pair-continue-test.sh:134:  env HOME="$CRT" XDG_DATA_HOME="$CRT/xdg" PAIR_TAG=demo PAIR_AGENT=claude PAIR_KILL_CMD=true "$@"
tests/pair-continue-test.sh:140:( cd "$CRT" && compact_env PAIR_FORCE_IN_SESSION=1 "$PAIR" continue demo >/dev/null 2>&1 )
tests/pair-continue-test.sh:141:grep -q '^continue=demo$' "$MK" 2>/dev/null && pass "compact: marker continue=slug" || fail "compact: marker missing continue="
tests/pair-continue-test.sh:149:( cd "$CRT" && compact_env PAIR_FORCE_IN_SESSION=1 "$PAIR" continue bogus >/dev/null 2>&1 ); rc=$?
tests/pair-continue-test.sh:154:( cd "$CRT" && compact_env PAIR_FAKE_IN_ZELLIJ=1 ZELLIJ_SESSION_NAME=pair-demo "$PAIR" continue demo >/dev/null 2>&1 )
tests/pair-continue-test.sh:157:( cd "$CRT" && compact_env PAIR_FAKE_IN_ZELLIJ=1 ZELLIJ_SESSION_NAME=pair-other "$PAIR" continue demo >/dev/null 2>&1 ); rc=$?
tests/pair-continue-test.sh:162:( cd "$CRT" && HOME="$CRT" XDG_DATA_HOME="$CRT/xdg" \
tests/pair-continue-test.sh:167:# 5. handle_restart_marker continue= → re-exec argv
tests/pair-continue-test.sh:172:continue=demo
tests/pair-continue-test.sh:176:( cd "$CRT" && HOME="$CRT" XDG_DATA_HOME="$CRT/xdg" SESSION=pair-demo \
tests/pair-continue-test.sh:178:grep -Eq 'continue demo claude -- .*--dangerously-skip-permissions' "$CAP" 2>/dev/null \
tests/pair-continue-test.sh:179:  && pass "restart: re-exec = continue <slug> <agent> -- <args>" || fail "restart: wrong re-exec ($(cat "$CAP" 2>/dev/null))"
tests/pair-continue-test.sh:193:  list-sessions)
tests/pair-continue-test.sh:209:  env HOME="$HRT" XDG_DATA_HOME="$HRT/xdg" PATH="$HRT/bin:$PATH" \
tests/pair-continue-test.sh:277:  printf 'PASS pair-continue-test\n'
tests/pair-continue-test.sh:279:  printf 'FAIL pair-continue-test (%d failure(s))\n' "$fails"
tests/review-apply-test.sh:120:-- (k) live decoration lines EQUAL the resume(reconstruct) lines — the
tests/review-apply-test.sh:121:-- new_occurrence invariant that keeps live-render and resume-render identical.
tests/review-apply-test.sh:133:ok(same, 'live decoration lines == reconstruct (resume) lines')
tests/cr-newline-test.sh:36:local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
tests/cr-newline-test.sh:76:  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/review-window-test.sh:43:( cd "$RT" && PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" RESULT="$RT/r1" \
tests/review-window-test.sh:48:if PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-open" "$RT/nope.md" 2>/dev/null; then
tests/review-window-test.sh:54:PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-open" "$RT/doc.md" || true
tests/review-window-test.sh:81:( cd "$REPO" && PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/review-window-test.sh:323:( cd "$REPO" && PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \

exec
/bin/zsh -lc 'go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1' in /Users/xianxu/workspace/pair
 succeeded in 936ms:
ok  	github.com/xianxu/pair/cmd/internal/launcher	0.649s
ok  	github.com/xianxu/pair/cmd/internal/dispatcher	0.459s
ok  	github.com/xianxu/pair/cmd/pair-go	0.822s

exec
/bin/zsh -lc "sed -n '560,690p' bin/pair && sed -n '1200,1465p' bin/pair" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
# name (clipboard-to-pane.sh, copy-on-select.sh) without sh-c env-var hacks
# in zellij/config.kdl. Avoid double-prepending across re-launches.
case ":$PATH:" in
    *":$PAIR_HOME/bin:"*) ;;
    *) export PATH="$PAIR_HOME/bin:$PATH" ;;
esac

# Parse argv. First positional is the agent name (default `claude`).
# Anything after `--` is forwarded to the agent on create. The `--`
# separator is required to disambiguate "agent name" from "flag for the
# agent" — without it, `pair --dangerously-skip-permissions` is ambiguous
# (is `--dangerously-skip-permissions` the agent or a flag?).
#
#   pair                                  → claude
#   pair claude                           → claude
#   pair claude -- --resume               → claude --resume
#   pair -- --resume                      → claude --resume
#   pair codex -- -p "hi"                 → codex -p hi   (word-split caveat)
AGENT=""
agent_extra=""
forced_tag=""

# `pair resume <tag>` is the documented restart path: skip both the picker
# and the name prompt, attach if pair-<tag> exists, otherwise create with
# that tag. We claim `resume` as a subcommand verb — it can't be a real
# agent binary (too generic a name), and we already special-case `list`
# and `help` similarly above.
if [ "${1:-}" = "resume" ]; then
    if [ -z "${2:-}" ]; then
        echo "pair: 'resume' requires a tag (e.g. pair resume bugfix)" >&2
        exit 1
    fi
    forced_tag="$(normalize_tag "$2" "pair: invalid tag")" || exit 1
    shift 2
fi

# `pair continue [slug] [agent]` — the human-understanding sibling of `resume`
# (ariadne#91). Bare: list continuation docs. With a slug: seed a fresh session
# to read the matching doc and execute its NEXT ACTION. It does NOT force the
# tag — like `pair <agent> -- <args>` it flows through the normal name prompt
# so the operator picks the tag (a long slug must never become an over-long
# zellij session name — see the launch-time guard) and any `-- <args>` reach
# the agent. The optional agent is the "port" (continue under a different
# stack); absent, it's read from the doc. We capture slug + agent HERE (the
# positional loop below rejects a bare agent arg) but leave `-- <args>` for it.
# `continue` never reads session_id — that's `resume`.
# Fresh-vs-attach (#52): the seeded session is "fresh" only for a NEW tag. If the
# tag picked at the prompt collides with a LIVE `pair-<tag>`, the normal launch
# path ATTACHES to it (same as `pair <agent>` / `resume`) rather than reseeding —
# intentional, so a continue never clobbers a running session; pick a fresh tag
# to force a clean seed.
CONTINUE_DOC=""
if [ "${1:-}" = "continue" ]; then
    _croot="$(git rev-parse --show-toplevel 2>/dev/null || true)"
    _cdir="${_croot:-$PWD}/workshop/continuation"
    if [ -z "${2:-}" ]; then
        if [ -d "$_cdir" ] && ls "$_cdir"/*.md >/dev/null 2>&1; then
            printf 'continuations in %s:\n' "$_cdir"
            for _cf in "$_cdir"/*.md; do
                _cs="$(basename "$_cf" .md)"; _cs="${_cs#*-}"
                _cn="$(awk '/^## NEXT ACTION/{f=1;next} f&&NF{print;exit}' "$_cf")"
                # cap the preview so a long NEXT ACTION line doesn't flood the row (#52)
                if [ "${#_cn}" -gt 80 ]; then _cn="${_cn:0:79}…"; fi
                _ci="$(awk -F': ' '/^issues:/{print $2; exit}' "$_cf")"
                printf '  %-22s %-18s %s\n' "$_cs" "${_ci:-[]}" "$_cn"
            done
        else
            printf 'pair: no continuations in %s\n' "$_cdir" >&2
        fi
        exit 0
    fi
    _cslug="$(normalize_tag "$2" "pair: invalid slug")" || exit 1
    CONTINUE_DOC="$(ls -1 "$_cdir"/*-"$_cslug".md 2>/dev/null | sort | tail -1)"
    if [ -z "$CONTINUE_DOC" ]; then
        echo "pair: no continuation matching '$_cslug' in $_cdir" >&2
        exit 1
    fi
    # No forced_tag (see header). Consume `continue` + slug; an optional next
    # arg is the agent port UNLESS it's `--` (then no port, only forwarded
    # args). Leave everything from `--` onward for the positional loop to
    # capture as agent_extra.
    shift 2
    if [ -n "${1:-}" ] && [ "${1:-}" != "--" ]; then
        AGENT="$1"   # explicit port: continue under a different agent
        shift
    else
        AGENT="$(awk -F': ' '/^agent:/{print $2; exit}' "$CONTINUE_DOC")"
    fi
fi

seen_dd=false
for arg in "$@"; do
    if $seen_dd; then
        if [ -z "$agent_extra" ]; then
            agent_extra="$arg"
        else
            agent_extra="$agent_extra $arg"
        fi
    elif [ "$arg" = "--" ]; then
        seen_dd=true
    elif [ -z "$AGENT" ] && [ -z "$forced_tag" ]; then
        AGENT="$arg"
    else
        echo "pair: unexpected positional arg '$arg' (use '--' to forward args to the agent)" >&2
        echo "      e.g. pair claude -- --resume" >&2
        exit 1
    fi
done

# PAIR_FORCE_TAG is set by the Alt+n / Shift+Alt+N restart flow
# (handle_restart_marker below): it pins the new run to the same tag as
# the killed session, skipping both the picker and the name prompt.
# Treated as forced_tag from here on; unset so a child process doesn't
# inherit it.
if [ -z "$forced_tag" ] && [ -n "${PAIR_FORCE_TAG:-}" ]; then
    forced_tag="$PAIR_FORCE_TAG"
fi
unset PAIR_FORCE_TAG

# --- #55 compaction support: hoisted DATA_DIR + early helpers ----------------
# DATA_DIR is hoisted above the `command -v zellij` gate, the in-session
# compaction branch, and the in_zellij_pane guard so park_scrollback /
# handle_restart_marker (defined here) and the branch can all use it.
# Honors $XDG_DATA_HOME with the spec-defined fallback. (Original site below
# the guard is removed.)
DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
mkdir -p "$DATA_DIR"
export PAIR_DATA_DIR="$DATA_DIR"

# Park a session's scrollback for later distillation into a continuation
# (ariadne#91). Extracted from cleanup_quit_marker so BOTH the Alt+x quit path
HISTORY_BASE="${HISTORY_BASE//[^A-Za-z0-9_-]/_}"
[ -z "$HISTORY_BASE" ] && HISTORY_BASE="pair"
HISTORY_DAYS="${PAIR_HISTORY_DAYS:-14}"

# Find ALL detached pair-* sessions, regardless of agent. The agent argument
# (and BASE_TAG) is only used for the create path's defaults; the picker
# itself shows everything you might want to reattach to, including custom-
# named sessions like pair-blogging that don't follow any agent prefix.
# `zellij list-sessions` exits 1 when there are no sessions, and with
# `set -euo pipefail` that would silently kill the whole script. Wrap the
# zellij call in `|| true` and use awk (returns 0 even on no matches)
# instead of grep (returns 1 on no matches).
all_pair=$( { zj list-sessions --short 2>/dev/null || true; } \
            | awk '/^pair-/' \
            | sort )

# With the live-session set in hand, reap any nvim --embed left behind by
# a tag whose pair-<tag> session is gone — external `zellij kill-session`,
# host reboot, or pair upgrade leave no Alt+x marker, so cleanup_quit_marker
# never runs and these embeds accumulate. Definition near $DATA_DIR setup.
sweep_orphan_nvim

detached_list=()
while IFS= read -r sess; do
    [ -z "$sess" ] && continue
    if zj list-sessions --no-formatting 2>/dev/null \
        | grep -E "^${sess}[[:space:]].*EXITED" >/dev/null 2>&1; then
        continue   # exited, skip
    fi
    # `zj --session NAME action list-clients` exits 1 with "There is no active
    # session!" when invoked from a process tree inside that same session's
    # panes — e.g. a script in an agent pane that shells out to `pair`. With
    # set -o pipefail, that propagates and errexit silently kills bin/pair at
    # the assignment. Swallow zellij's exit the same way line 804 does for
    # list-sessions; the rest of the pipeline still produces a clean "0" for
    # the empty-stdout case, which is the right semantic (treat unable-to-
    # query as detached).
    clients=$( { zj --session "$sess" action list-clients 2>/dev/null || true; } \
              | tail -n +2 | wc -l | tr -d ' ')
    if [ "${clients:-0}" -eq 0 ]; then
        detached_list+=("$sess")
    fi
done <<< "$all_pair"

# ── Historical tags from this cwd (#000024) ──────────────────────────────────
# Surface tags named `$HISTORY_BASE` or `$HISTORY_BASE-*` that were touched
# (draft / log mtime) within the last $HISTORY_DAYS days but no longer have a
# live zellij session. Lets the operator resume an old session by name from
# the picker without remembering the exact slot number. Convention-only —
# tags that don't follow the cwd-prefix rule are not surfaced; the operator
# is expected to name sessions `<cwd-base>-<subproject>` so they show up in
# the right context (atlas note).
#
# Scan strategy: walk draft-/log- sidecars (created/touched by nvim's
# autosave and bin/pair's send-log), bucket per tag, take max mtime, filter
# by cutoff, then dedup against the live set.

scan_history() {
    local base="$1" cutoff="$2"
    local f name tag mt
    for f in "$DATA_DIR"/draft-*.md "$DATA_DIR"/log-*.md; do
        [ -e "$f" ] || continue
        name="${f##*/}"
        case "$name" in
            draft-*.md) tag="${name#draft-}"; tag="${tag%.md}" ;;
            log-*.md)   tag="${name#log-}";   tag="${tag%.md}" ;;
            *) continue ;;
        esac
        # Strict prefix: tag is exactly $base, or starts with "$base-".
        case "$tag" in
            "$base"|"$base"-*) ;;
            *) continue ;;
        esac
        mt=$(stat -f %m "$f" 2>/dev/null || echo 0)
        [ "$mt" -lt "$cutoff" ] && continue
        printf '%s\t%s\n' "$tag" "$mt"
    done | awk -F'\t' '
        { if ($2 > m[$1]) m[$1] = $2 }
        END { for (t in m) printf "%s\t%s\n", t, m[t] }
    ' | sort -t$'\t' -k2 -nr
}

format_age() {
    local now="$1" then="$2"
    local diff_days=$(( (now - then) / 86400 ))
    case "$diff_days" in
        0) echo "today" ;;
        1) echo "yesterday" ;;
        *) echo "${diff_days}d ago" ;;
    esac
}

# Greyscale gradient by age for the historical (no-live-session) rows: the
# more recently a tag was touched, the brighter it reads. 256-color greys
# (xterm 232=near-black .. 255=near-white); fzf --ansi honors these. Newest
# stays comfortably readable on the dark bg; oldest fades toward the
# background without disappearing.
age_color() {
    local d="$1"
    if   [ "$d" -le 0 ]; then printf '\033[38;5;250m'   # today
    elif [ "$d" -le 1 ]; then printf '\033[38;5;245m'   # yesterday
    elif [ "$d" -le 3 ]; then printf '\033[38;5;242m'   # 2-3d
    elif [ "$d" -le 6 ]; then printf '\033[38;5;240m'   # 4-6d
    else                      printf '\033[38;5;238m'   # a week+
    fi
}

# Count the +N draft items parked in a tag's queue. nvim writes each queued
# prompt as queue-<tag>/<6-digit-key>.md (see queue_dir / queue_keys_sorted in
# nvim/init.lua), so the queued count is just how many <digits>.md files live
# there. Surfaced as a badge on inactive picker rows so a session's forgotten
# queue is visible before you resume it.
queue_count_for() {
    local tag="$1" qdir="$DATA_DIR/queue-$tag" n=0 f
    [ -d "$qdir" ] || { echo 0; return; }
    for f in "$qdir"/[0-9]*.md; do
        [ -e "$f" ] || continue   # no match → glob stays literal; skip
        n=$((n + 1))
    done
    echo "$n"
}

now_epoch=$(date +%s)
history_cutoff=$(( now_epoch - HISTORY_DAYS * 86400 ))

# Build the live-tag exclusion set as a newline-delimited string (bash 3.2
# friendly — no assoc arrays). One tag per line, lookup via `grep -qx`.
live_tags_nl=$(printf '%s\n' "$all_pair" | sed 's|^pair-||' | sort -u)

# Capture the historical scan once. It feeds two things: the next-free create
# slot (below) and the picker's "no live session" rows. Tags here have a recent
# draft/log sidecar but may or may not still have a live zellij session.
history_scan="$(scan_history "$HISTORY_BASE" "$history_cutoff")"
historical_tags_nl=$(printf '%s\n' "$history_scan" | cut -f1 | sed '/^$/d' | sort -u)

# Compute the next-free slot in the agent's family — pair-<base>, then
# pair-<base>-2, ... — used as the "+ new" prompt's default name on create.
# Skip any candidate that is live OR has recent history, so "+ new <tag>
# session" always proposes a genuinely fresh name; an inactive pair-<base>
# then surfaces as its own resumable "no live session" row below instead of
# being folded into "+ new" (#000024 follow-up). Checks against ALL live
# sessions (not just the family) to avoid clobbering a shared slot name.
free_slot_tag=""
i=1
while [ "$i" -le 100 ]; do
    if [ "$i" -eq 1 ]; then
        cand="pair-${BASE_TAG}"
    else
        cand="pair-${BASE_TAG}-${i}"
    fi
    cand_tag="${cand#pair-}"
    if ! session_blocks_reuse "$cand" \
       && ! printf '%s\n' "$historical_tags_nl" | grep -qx "$cand_tag"; then
        free_slot_tag="$cand_tag"
        break
    fi
    i=$((i + 1))
done

historical_rows=()      # plain text — drives the fallback display + matching
historical_colored=()   # age-graded grey — fzf display only
while IFS=$'\t' read -r tag mtime; do
    [ -z "$tag" ] && continue
    # Dedup against live sessions only — free_slot_tag is guaranteed not to be
    # a historical tag (the slot scan above skips history), so an inactive
    # pair-<tag> now shows up here as its own resumable row.
    if printf '%s\n' "$live_tags_nl" | grep -qx "$tag"; then continue; fi
    age=$(format_age "$now_epoch" "$mtime")
    base_row="$(printf 'pair-%s  (%s, no live session)' "$tag" "$age")"
    # Queue badge (only when non-empty): kept as a separate segment so the
    # colored row tints just the badge amber (xterm 214) while age_color
    # greys the base. The plain badge has the identical visible text, so the
    # fzf --ansi return still maps back to a historical_rows entry, and the
    # "${picked%%  *}" → pair-<tag> parse below is unaffected (badge sits
    # after the parenthetical's leading double-space).
    qn=$(queue_count_for "$tag")
    badge_plain=""; badge_colored=""
    if [ "$qn" -gt 0 ]; then
        badge_plain="$(printf '   [\xe2\x8f\x8e %d queued]' "$qn")"
        badge_colored="$(printf '   \033[38;5;214m[\xe2\x8f\x8e %d queued]\033[0m' "$qn")"
    fi
    historical_rows+=("${base_row}${badge_plain}")
    diff_days=$(( (now_epoch - mtime) / 86400 ))
    historical_colored+=("$(age_color "$diff_days")${base_row}"$'\033[0m'"${badge_colored}")
done <<< "$history_scan"

# Debug probe — `PAIR_DEBUG_HISTORY=1 pair` prints what the scan would have
# surfaced and exits without launching zellij. Use this to verify the cwd-
# prefix convention is matching what you expect on your data dir.
if [ "${PAIR_DEBUG_HISTORY:-0}" = "1" ]; then
    echo "pair: historical scan"
    echo "  data_dir       = $DATA_DIR"
    echo "  base (cwd)     = $HISTORY_BASE"
    echo "  window         = ${HISTORY_DAYS}d (cutoff epoch ${history_cutoff})"
    echo "  live tags      = $(printf '%s' "$live_tags_nl" | tr '\n' ' ')"
    echo "  free_slot_tag  = $free_slot_tag"
    if [ "${#historical_rows[@]}" -eq 0 ]; then
        echo "  (no historical tags matched)"
    else
        echo "  matched rows:"
        printf '    %s\n' "${historical_rows[@]}"
    fi
    exit 0
fi

# Decide. Picker for any ≥1 detached or ≥1 historical row; create directly
# only when nothing to surface.
chosen_session=""
chosen_tag=""
action=""        # "attach" | "create"
# 1 only when naming a brand-new session (the "+ new" pick, or a direct
# create when there's nothing to surface). Picking an existing historical
# tag, or `pair resume <tag>`, already names the session — and a historical
# tag is known to have no live session (it was deduped against the live set)
# — so the name prompt + collision warning below are skipped for those.
prompt_name=0

# `pair resume <tag>` short-circuits both the picker and the name prompt:
# attach if pair-<tag> already exists in any state, otherwise create with
# that tag. This is the "resume by name" path — pair off the hint printed
# at Alt+x.
if [ -n "$forced_tag" ]; then
    if session_blocks_reuse "pair-${forced_tag}"; then
        chosen_session="pair-${forced_tag}"
        chosen_tag="$forced_tag"
        action=attach
    else
        chosen_tag="$forced_tag"
        action=create
    fi
elif [ ${#detached_list[@]} -eq 0 ] && [ ${#historical_rows[@]} -eq 0 ]; then
    chosen_tag="$free_slot_tag"
    action=create
    prompt_name=1
else
    new_label="+ new ${BASE_TAG} session"
    # Plain text drives selection matching below; the colored variant is for
    # fzf display only. fzf --ansi strips its color codes from the returned
    # line, so `picked` maps back to one of these exact plain strings.
    options=$({
        if [ ${#detached_list[@]} -gt 0 ]; then printf '%s\n' "${detached_list[@]}"; fi
        if [ ${#historical_rows[@]} -gt 0 ]; then printf '%s\n' "${historical_rows[@]}"; fi
        printf '%s\n' "$new_label"
    })
    if command -v fzf >/dev/null 2>&1; then
        # Color-code: green = live session (picking it attaches); historical
        # rows carry an age-graded grey (set per-row in historical_colored,
        # brighter = more recent) and picking one re-creates by name. The
        # "+ new" row stays default.
        C_ACTIVE=$'\033[32m'; C_RESET=$'\033[0m'
        options_colored=$({
            if [ ${#detached_list[@]} -gt 0 ]; then printf "${C_ACTIVE}%s${C_RESET}\n" "${detached_list[@]}"; fi
            if [ ${#historical_colored[@]} -gt 0 ]; then printf '%s\n' "${historical_colored[@]}"; fi
            printf '%s\n' "$new_label"
        })
        picked=$(printf '%s' "$options_colored" | fzf --ansi --prompt='pair> ' --height=10 --reverse) || exit 0
    else
        echo "Detached / historical pair sessions:"
        printf '%s' "$options" | nl
        printf 'Pick: '
        read -r idx
        picked=$(printf '%s' "$options" | sed -n "${idx}p")
    fi
    [ -z "$picked" ] && exit 0

    if [ "$picked" = "$new_label" ]; then

codex
The focused Go tests pass. One subtle point: `go run` turns the program’s intentional exit 3 into `go run` exit 1 with an `exit status 3` wrapper, so I’m treating direct `run()`/tests as the cleaner evidence for the prototype status code.
exec
/bin/zsh -lc "sed -n '1465,1545p' bin/pair && nl -ba cmd/internal/launcher/args.go | sed -n '1,90p' && nl -ba cmd/internal/launcher/decision.go | sed -n '1,125p' && nl -ba cmd/internal/launcher/run.go | sed -n '1,90p' && nl -ba cmd/internal/launcher/zellij.go | sed -n '1,115p' && nl -ba cmd/internal/dispatcher/dispatcher.go | sed -n '1,190p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
    if [ "$picked" = "$new_label" ]; then
        chosen_tag="$free_slot_tag"
        action=create
        prompt_name=1
    else
        # Historical rows are annotated as "pair-<tag>  (... ago, ...)" — the
        # double-space separator never appears in detached-row text (sanitized
        # tags can't contain spaces), so trimming everything from the first
        # double-space onward yields the bare "pair-<tag>" for both shapes.
        chosen_session="${picked%%  *}"
        chosen_tag="${chosen_session#pair-}"
        # If the picked tag is currently live → attach; otherwise (historical
        # row, or the unlikely "live tag listed as historical" race) → create
        # by name, which goes through the same path as `pair resume <tag>`.
        if printf '%s\n' "$live_tags_nl" | grep -qx "$chosen_tag"; then
            action=attach
        else
            chosen_session=""
            action=create
        fi
    fi
fi

if [ -z "$chosen_tag" ]; then
    echo "pair: couldn't determine a session to use; bailing." >&2
    exit 1
fi

# Helper: after zellij returns, check for the Alt+x marker. If present, the
# user asked for a full quit — delete the session entry from the resurrect
# list. (Ctrl+q is unbound in pair, so Alt+x is the only quit path; Alt+d
# is detach, which keeps the session alive and leaves no marker.)
cleanup_quit_marker() {
    local marker="$HOME/.cache/pair/quit-$SESSION"
    if [ -f "$marker" ]; then
        rm -f "$marker"
        # Capture the agent BEFORE the file is cleared below, so the resume
        # hint can name the right binary.
        local quit_agent _sb_base _parked _ans _pbase _ptimeout   # #52: keep park-nudge vars function-scoped
        quit_agent=$(cat "$DATA_DIR/agent-${PAIR_TAG}" 2>/dev/null || echo "$AGENT")
        zj delete-session "$SESSION" --force >/dev/null 2>&1 || true
        # Defensive belt-and-suspenders: if a zellij --server process for this
        # session is still alive after delete-session (rare but seen — e.g.
        # the server lingered in a no-clients state and re-registered the
        # session record on its next heartbeat), SIGKILL it. Pattern matches
        # the socket path component, scoped to this session only.
        pkill -9 -f "zellij --server .*/${SESSION}\$" >/dev/null 2>&1 || true
        # Reap this tag's nvim --embed children. See reap_nvim_for_tag and
        # the orphan-reaping block near $DATA_DIR setup for the full story
        # on why these survive pane death and need explicit cleanup.
        reap_nvim_for_tag "$PAIR_TAG"
        # Park-nudge: the scrollback is the only on-disk record of this session,
        # and Alt+x is about to discard it. Offer to PRESERVE it (rename to a
        # non-recyclable name so the next `pair <same-tag>` O_TRUNC can't clobber
        # it) so a live session can later distill it into a continuation
        # (ariadne#91). No live agent exists at quit, so we only preserve — never
        # distill here. `[ -t 0 ]` skips the prompt on a detached/automated quit.
        _sb_base="$DATA_DIR/scrollback-${PAIR_TAG}-${quit_agent}"
        _parked=0
        # Skip the park nudge when a restart is pending: a restart isn't a
        # quit (the relaunch keeps the work), and for #55 compaction the
        # branch has ALREADY copy-parked a recovery net + written a
        # continuation — re-asking "park as a continuation?" here is noise.
        # Also de-noises the inherited Alt+n / Shift+Alt+N restart paths.
        if [ -s "${_sb_base}.raw" ] && [ -t 0 ] \
            && [ ! -f "$HOME/.cache/pair/restart-$SESSION" ]; then
            # Bound the nudge with a timeout (default 5s, PAIR_PARK_PROMPT_TIMEOUT
            # seam) so an unattended quit auto-picks the default (N) instead of
            # blocking forever. read -t exits non-zero on timeout/EOF, so only the
            # `then` branch (read SUCCEEDED) consults the answer — any partial
            # input bash 4+ leaves in $_ans on timeout is never read as a "yes".
            # Guard a non-integer seam value (bash 3.2 `read -t` needs an integer).
            # A 0 value is a valid "don't wait": `read -t 0` returns at once, so
            # the prompt auto-declines (N) immediately rather than blocking.
            _ptimeout="${PAIR_PARK_PROMPT_TIMEOUT:-5}"
            case "$_ptimeout" in ''|*[!0-9]*) _ptimeout=5 ;; esac
            printf 'pair: preserve "%s" scrollback to distill into a continuation later? [y/N] (%ss → N): ' "$SESSION" "$_ptimeout" >/dev/tty
            if read -t "$_ptimeout" -r _ans </dev/tty; then
                if [ "$_ans" = "y" ] || [ "$_ans" = "Y" ]; then
                    # park_scrollback (move mode — session is dying) handles the
                    # .raw/.events.jsonl rename + parked-<tag> marker (#55 ARCH-DRY).
     1	package launcher
     2	
     3	import "fmt"
     4	
     5	// LaunchArgs is the pure parse result for the guarded pair-go launch prototype.
     6	type LaunchArgs struct {
     7		Agent       string
     8		ForcedTag   string
     9		SelectedTag string
    10		AgentArgs   []string
    11	}
    12	
    13	// UsageError is an operator-facing parse error.
    14	type UsageError struct {
    15		Message string
    16	}
    17	
    18	func (e UsageError) Error() string {
    19		return e.Message
    20	}
    21	
    22	// ParseArgs parses pair-go launch args. It intentionally supports only the
    23	// decision-phase subset for #75; unsupported shell-owned launcher verbs fail
    24	// explicitly.
    25	func ParseArgs(argv []string) (LaunchArgs, error) {
    26		var out LaunchArgs
    27		if len(argv) == 0 {
    28			out.Agent = "claude"
    29			return out, nil
    30		}
    31	
    32		switch argv[0] {
    33		case "continue", "rename", "list", "ls":
    34			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %s is not implemented by pair-go launch; use bin/pair", argv[0])}
    35		case "resume":
    36			if len(argv) < 2 {
    37				return LaunchArgs{}, UsageError{Message: "pair-go launch: 'resume' requires a tag"}
    38			}
    39			tag, err := NormalizeTag(argv[1])
    40			if err != nil {
    41				return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: invalid tag: %v", err)}
    42			}
    43			if len(argv) > 2 {
    44				return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", argv[2])}
    45			}
    46			out.ForcedTag = tag
    47			return out, nil
    48		}
    49	
    50		seenSeparator := false
    51		for _, arg := range argv {
    52			if seenSeparator {
    53				out.AgentArgs = append(out.AgentArgs, arg)
    54				continue
    55			}
    56			if arg == "--" {
    57				seenSeparator = true
    58				continue
    59			}
    60			if out.Agent == "" {
    61				out.Agent = arg
    62				continue
    63			}
    64			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", arg)}
    65		}
    66	
    67		if out.Agent == "" {
    68			out.Agent = "claude"
    69		}
    70		return out, nil
    71	}
     1	package launcher
     2	
     3	// LaunchAction is the guarded prototype's next launcher action.
     4	type LaunchAction string
     5	
     6	const (
     7		ActionAttach LaunchAction = "attach"
     8		ActionCreate LaunchAction = "create"
     9		ActionPick   LaunchAction = "pick"
    10	)
    11	
    12	// LaunchDecision is a pure create/attach/pick decision. Tag is canonical bare
    13	// form; SessionName is derived as pair-<tag> when a zellij session is named.
    14	type LaunchDecision struct {
    15		Action      LaunchAction
    16		Tag         string
    17		SessionName string
    18		PromptName  bool
    19	}
    20	
    21	// DecideLaunch decides the launch action without touching zellij, fzf, or disk.
    22	func DecideLaunch(args LaunchArgs, snap SessionSnapshot) (LaunchDecision, error) {
    23		if args.SelectedTag != "" {
    24			return createDecision(args.SelectedTag, false), nil
    25		}
    26		if args.ForcedTag != "" {
    27			if sessionBlocksReuse(snap, sessionName(args.ForcedTag)) {
    28				return LaunchDecision{Action: ActionAttach, Tag: args.ForcedTag, SessionName: sessionName(args.ForcedTag)}, nil
    29			}
    30			return createDecision(args.ForcedTag, false), nil
    31		}
    32		if hasDetached(snap) || len(snap.Historical) > 0 {
    33			return LaunchDecision{Action: ActionPick}, nil
    34		}
    35		tag := snap.BaseTag
    36		if tag == "" {
    37			tag = "pair"
    38		}
    39		return createDecision(nextFreeTag(tag, snap), true), nil
    40	}
    41	
    42	func createDecision(tag string, prompt bool) LaunchDecision {
    43		return LaunchDecision{Action: ActionCreate, Tag: tag, SessionName: sessionName(tag), PromptName: prompt}
    44	}
    45	
    46	func sessionName(tag string) string {
    47		return "pair-" + tag
    48	}
    49	
    50	func hasDetached(snap SessionSnapshot) bool {
    51		for _, sess := range snap.Sessions {
    52			if sess.State == SessionDetached {
    53				return true
    54			}
    55		}
    56		return false
    57	}
    58	
    59	func sessionBlocksReuse(snap SessionSnapshot, name string) bool {
    60		for _, sess := range snap.Sessions {
    61			if sess.Name != name {
    62				continue
    63			}
    64			return sess.State == SessionAttached || sess.State == SessionDetached
    65		}
    66		return false
    67	}
    68	
    69	func nextFreeTag(base string, snap SessionSnapshot) string {
    70		for i := 1; i <= 100; i++ {
    71			tag := base
    72			if i > 1 {
    73				tag = base + "-" + itoa(i)
    74			}
    75			if !sessionBlocksReuse(snap, sessionName(tag)) && !isHistorical(snap, tag) {
    76				return tag
    77			}
    78		}
    79		return base
    80	}
    81	
    82	func isHistorical(snap SessionSnapshot, tag string) bool {
    83		for _, hist := range snap.Historical {
    84			if hist.Tag == tag {
    85				return true
    86			}
    87		}
    88		return false
    89	}
    90	
    91	func itoa(n int) string {
    92		if n == 0 {
    93			return "0"
    94		}
    95		var buf [20]byte
    96		i := len(buf)
    97		for n > 0 {
    98			i--
    99			buf[i] = byte('0' + n%10)
   100			n /= 10
   101		}
   102		return string(buf[i:])
   103	}
     1	package launcher
     2	
     3	import (
     4		"time"
     5	)
     6	
     7	// Env is the explicit runtime environment for launch decisions.
     8	type Env struct {
     9		Home     string
    10		XDGData  string
    11		Cwd      string
    12		Now      time.Time
    13		HistoryD int
    14		DataDir  string
    15	}
    16	
    17	// SessionSource supplies zellij session state.
    18	type SessionSource interface {
    19		Snapshot() ([]Session, error)
    20	}
    21	
    22	// HistoricalScanner supplies recently touched Pair tags.
    23	type HistoricalScanner interface {
    24		Scan(base string, cutoff time.Time) ([]HistoricalTag, error)
    25	}
    26	
    27	// LaunchOutcome is the domain outcome from the launcher core.
    28	type LaunchOutcome struct {
    29		Args     LaunchArgs
    30		Env      Env
    31		Decision LaunchDecision
    32	}
    33	
    34	// Run builds a pure snapshot from injected sources and returns a domain launch
    35	// outcome. The dispatcher maps this to process stdout/stderr/exit status.
    36	func Run(argv []string, env Env, sessions SessionSource, history HistoricalScanner) (LaunchOutcome, error) {
    37		args, err := ParseArgs(argv)
    38		if err != nil {
    39			return LaunchOutcome{}, err
    40		}
    41		if env.DataDir == "" {
    42			env.DataDir = ResolveDataDir(env.Home, env.XDGData)
    43		}
    44		if env.HistoryD == 0 {
    45			env.HistoryD = 14
    46		}
    47		if env.Now.IsZero() {
    48			env.Now = time.Now()
    49		}
    50	
    51		sessionRows, err := sessions.Snapshot()
    52		if err != nil {
    53			return LaunchOutcome{}, err
    54		}
    55		base := DefaultTag(env.Cwd)
    56		historical, err := history.Scan(base, env.Now.Add(-time.Duration(env.HistoryD)*24*time.Hour))
    57		if err != nil {
    58			return LaunchOutcome{}, err
    59		}
    60	
    61		decision, err := DecideLaunch(args, SessionSnapshot{
    62			BaseTag:    base,
    63			Sessions:   sessionRows,
    64			Historical: historical,
    65		})
    66		if err != nil {
    67			return LaunchOutcome{}, err
    68		}
    69		return LaunchOutcome{Args: args, Env: env, Decision: decision}, nil
    70	}
     1	package launcher
     2	
     3	import (
     4		"bytes"
     5		"os/exec"
     6		"sort"
     7		"strings"
     8	)
     9	
    10	// ZellijSource reads zellij session state through the zellij CLI.
    11	type ZellijSource struct {
    12		Path string
    13	}
    14	
    15	func (s ZellijSource) Snapshot() ([]Session, error) {
    16		short, err := s.run("list-sessions", "--short")
    17		if err != nil {
    18			short = nil
    19		}
    20		raw, err := s.run("list-sessions", "--no-formatting")
    21		if err != nil {
    22			raw = nil
    23		}
    24		exited := exitedSessions(string(raw))
    25		var out []Session
    26		for _, name := range lines(string(short)) {
    27			if !strings.HasPrefix(name, "pair-") {
    28				continue
    29			}
    30			state := SessionDetached
    31			if exited[name] {
    32				state = SessionExited
    33			} else if s.clientCount(name) > 0 {
    34				state = SessionAttached
    35			}
    36			out = append(out, Session{Name: name, State: state})
    37		}
    38		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
    39		return out, nil
    40	}
    41	
    42	func (s ZellijSource) clientCount(session string) int {
    43		out, err := s.run("--session", session, "action", "list-clients")
    44		if err != nil {
    45			return 0
    46		}
    47		lines := lines(string(out))
    48		if len(lines) <= 1 {
    49			return 0
    50		}
    51		return len(lines) - 1
    52	}
    53	
    54	func (s ZellijSource) run(args ...string) ([]byte, error) {
    55		path := s.Path
    56		if path == "" {
    57			path = "zellij"
    58		}
    59		cmd := exec.Command(path, args...)
    60		var stdout bytes.Buffer
    61		cmd.Stdout = &stdout
    62		if err := cmd.Run(); err != nil {
    63			return nil, err
    64		}
    65		return stdout.Bytes(), nil
    66	}
    67	
    68	func exitedSessions(raw string) map[string]bool {
    69		out := map[string]bool{}
    70		for _, line := range lines(raw) {
    71			fields := strings.Fields(line)
    72			if len(fields) == 0 {
    73				continue
    74			}
    75			if strings.Contains(line, "EXITED") {
    76				out[fields[0]] = true
    77			}
    78		}
    79		return out
    80	}
    81	
    82	func lines(s string) []string {
    83		var out []string
    84		for _, line := range strings.Split(s, "\n") {
    85			line = strings.TrimSpace(line)
    86			if line != "" {
    87				out = append(out, line)
    88			}
    89		}
    90		return out
    91	}
     1	package dispatcher
     2	
     3	import (
     4		"fmt"
     5		"os"
     6		"strings"
     7		"time"
     8	
     9		"github.com/xianxu/pair/cmd/internal/launcher"
    10	)
    11	
    12	const programName = "pair-go"
    13	
    14	// CommandFamily names a future Pair CLI surface without claiming it works yet.
    15	type CommandFamily struct {
    16		Name    string
    17		Summary string
    18		Status  string
    19	}
    20	
    21	// Result is the process-facing outcome of a pure dispatch decision.
    22	type Result struct {
    23		Stdout   string
    24		Stderr   string
    25		ExitCode int
    26	}
    27	
    28	// Families returns the planned command families for the Go dispatcher.
    29	func Families() []CommandFamily {
    30		return []CommandFamily{
    31			{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "prototype"},
    32			{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
    33			{Name: "slug", Summary: "session orientation slug generation", Status: "planned"},
    34			{Name: "context", Summary: "agent pane context meter", Status: "planned"},
    35			{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "planned"},
    36			{Name: "changelog", Summary: "TTY transcript to distilled change log", Status: "planned"},
    37			{Name: "continuation", Summary: "continuation datatype writer", Status: "planned"},
    38			{Name: "scribe", Summary: "PTY logging wrapper", Status: "planned"},
    39		}
    40	}
    41	
    42	// Dispatch parses argv and returns the skeleton dispatch result.
    43	func Dispatch(args []string) Result {
    44		if len(args) == 0 {
    45			return Result{Stdout: Help(programName), ExitCode: 0}
    46		}
    47	
    48		switch args[0] {
    49		case "help", "--help", "-h":
    50			return Result{Stdout: Help(programName), ExitCode: 0}
    51		case "version", "--version":
    52			return Result{
    53				Stdout:   "pair-go dispatcher skeleton\npublic launcher: bin/pair\n",
    54				ExitCode: 0,
    55			}
    56		case "launch":
    57			return DispatchWithLauncherRuntime(args, osLauncherRuntime())
    58		}
    59	
    60		if family, ok := familyByName(args[0]); ok {
    61			return Result{
    62				Stderr:   fmt.Sprintf("%s: %s is planned but not implemented in this skeleton; run %s help\n", programName, family.Name, programName),
    63				ExitCode: 2,
    64			}
    65		}
    66	
    67		return Result{
    68			Stderr:   fmt.Sprintf("%s: unknown command %q; run %s help\n", programName, args[0], programName),
    69			ExitCode: 2,
    70		}
    71	}
    72	
    73	type LauncherRuntime struct {
    74		Env      launcher.Env
    75		Sessions launcher.SessionSource
    76		History  launcher.HistoricalScanner
    77	}
    78	
    79	type StaticSessions struct {
    80		Sessions []launcher.Session
    81		Err      error
    82	}
    83	
    84	func (s StaticSessions) Snapshot() ([]launcher.Session, error) {
    85		if s.Err != nil {
    86			return nil, s.Err
    87		}
    88		return s.Sessions, nil
    89	}
    90	
    91	type StaticHistory struct {
    92		Tags []launcher.HistoricalTag
    93		Err  error
    94	}
    95	
    96	func (h StaticHistory) Scan(_ string, _ time.Time) ([]launcher.HistoricalTag, error) {
    97		if h.Err != nil {
    98			return nil, h.Err
    99		}
   100		return h.Tags, nil
   101	}
   102	
   103	func DispatchWithLauncherRuntime(args []string, rt LauncherRuntime) Result {
   104		launchArgs := []string(nil)
   105		if len(args) > 1 {
   106			launchArgs = args[1:]
   107		}
   108		if len(launchArgs) > 0 && (launchArgs[0] == "help" || launchArgs[0] == "--help" || launchArgs[0] == "-h") {
   109			return Result{Stdout: LaunchHelp(programName), ExitCode: 0}
   110		}
   111		outcome, err := launcher.Run(launchArgs, rt.Env, rt.Sessions, rt.History)
   112		if err != nil {
   113			return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
   114		}
   115		decision := outcome.Decision
   116		return Result{
   117			Stderr: fmt.Sprintf(
   118				"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
   119				decision.Action,
   120				decision.Tag,
   121				decision.SessionName,
   122			),
   123			ExitCode: 3,
   124		}
   125	}
   126	
   127	func LaunchHelp(program string) string {
   128		return fmt.Sprintf(`Usage: %s launch [agent] [-- agent-args...]
   129	       %s launch resume <tag>
   130	
   131	Guarded decision-phase prototype. Public sessions still start through bin/pair.
   132	This command parses launch inputs and computes the create/attach/picker decision,
   133	then stops before invoking zellij.
   134	`, program, program)
   135	}
   136	
   137	func LauncherEnv(home, xdgDataHome, cwd string) launcher.Env {
   138		return launcher.Env{
   139			Home:     home,
   140			XDGData:  xdgDataHome,
   141			Cwd:      cwd,
   142			Now:      time.Now(),
   143			HistoryD: 14,
   144		}
   145	}
   146	
   147	func osLauncherRuntime() LauncherRuntime {
   148		home := os.Getenv("HOME")
   149		xdg := os.Getenv("XDG_DATA_HOME")
   150		cwd, _ := os.Getwd()
   151		env := LauncherEnv(home, xdg, cwd)
   152		dataDir := launcher.ResolveDataDir(home, xdg)
   153		return LauncherRuntime{
   154			Env:      env,
   155			Sessions: launcher.ZellijSource{},
   156			History:  launcher.HistorySource{DataDir: dataDir},
   157		}
   158	}
   159	
   160	// Help renders the development-only dispatcher usage text.
   161	func Help(program string) string {
   162		var b strings.Builder
   163		fmt.Fprintf(&b, "Usage: %s <command> [args]\n\n", program)
   164		b.WriteString("Development dispatcher skeleton. Public sessions still start through bin/pair.\n\n")
   165		b.WriteString("Implemented prototype commands:\n")
   166		for _, family := range Families() {
   167			if family.Status == "prototype" {
   168				fmt.Fprintf(&b, "  %-17s %s (prototype; decision-phase only)\n", family.Name, family.Summary)
   169			}
   170		}
   171		b.WriteString("\nPlanned command families (not implemented in this skeleton):\n")
   172		for _, family := range Families() {
   173			if family.Status != "prototype" {
   174				fmt.Fprintf(&b, "  %-17s %s (%s; not implemented in this skeleton)\n", family.Name, family.Summary, family.Status)
   175			}
   176		}
   177		b.WriteString("\nSupported skeleton commands:\n")
   178		b.WriteString("  help              show this help\n")
   179		b.WriteString("  version           show dispatcher skeleton metadata\n")
   180		return b.String()
   181	}
   182	
   183	func familyByName(name string) (CommandFamily, bool) {
   184		for _, family := range Families() {
   185			if family.Name == name {
   186				return family, true
   187			}
   188		}
   189		return CommandFamily{}, false
   190	}

codex
No new blocker has surfaced in the implementation pass. I’m checking the exact test additions now so the final verdict can distinguish “covered by tests” from “works by inspection.”
exec
/bin/zsh -lc "nl -ba cmd/pair-go/main_test.go | sed -n '1,130p'" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "nl -ba cmd/pair-go/launch_process_test.go | sed -n '1,110p'" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "nl -ba cmd/internal/dispatcher/dispatcher_test.go | sed -n '1,150p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package main
     2	
     3	import (
     4		"bytes"
     5		"strings"
     6		"testing"
     7	
     8		"github.com/xianxu/pair/cmd/internal/dispatcher"
     9	)
    10	
    11	func TestRunWritesStdoutAndReturnsDispatcherCode(t *testing.T) {
    12		var stdout, stderr bytes.Buffer
    13		code := run([]string{"help"}, &stdout, &stderr)
    14		if code != 0 {
    15			t.Fatalf("code = %d, want 0", code)
    16		}
    17		if !strings.Contains(stdout.String(), "Usage: pair-go <command> [args]") {
    18			t.Fatalf("stdout missing usage:\n%s", stdout.String())
    19		}
    20		if stderr.String() != "" {
    21			t.Fatalf("stderr = %q, want empty", stderr.String())
    22		}
    23	}
    24	
    25	func TestRunWritesStderrAndReturnsDispatcherCode(t *testing.T) {
    26		var stdout, stderr bytes.Buffer
    27		code := run([]string{"wrap"}, &stdout, &stderr)
    28		if code != 2 {
    29			t.Fatalf("code = %d, want 2", code)
    30		}
    31		if stdout.String() != "" {
    32			t.Fatalf("stdout = %q, want empty", stdout.String())
    33		}
    34		if !strings.Contains(stderr.String(), "wrap is planned but not implemented") {
    35			t.Fatalf("stderr missing unsupported-command message:\n%s", stderr.String())
    36		}
    37	}
    38	
    39	func TestRunLaunchHelp(t *testing.T) {
    40		var stdout, stderr bytes.Buffer
    41		code := run([]string{"launch", "--help"}, &stdout, &stderr)
    42		if code != 0 {
    43			t.Fatalf("code = %d, want 0", code)
    44		}
    45		if !strings.Contains(stdout.String(), "Usage: pair-go launch") {
    46			t.Fatalf("stdout missing launch usage:\n%s", stdout.String())
    47		}
    48		if stderr.String() != "" {
    49			t.Fatalf("stderr = %q, want empty", stderr.String())
    50		}
    51	}
    52	
    53	func TestRunLaunchResumeReturnsPrototypeDecision(t *testing.T) {
    54		var stdout, stderr bytes.Buffer
    55		code := runWithLauncherRuntime([]string{"launch", "resume", "demo"}, &stdout, &stderr, testLauncherRuntime("/home/me", "", "/work/pair"))
    56		if code != 3 {
    57			t.Fatalf("code = %d, want 3", code)
    58		}
    59		if stdout.String() != "" {
    60			t.Fatalf("stdout = %q, want empty", stdout.String())
    61		}
    62		for _, want := range []string{"prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
    63			if !strings.Contains(stderr.String(), want) {
    64				t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
    65			}
    66		}
    67	}
    68	
    69	func TestRunLaunchWithoutArgsReturnsDefaultPrototypeDecision(t *testing.T) {
    70		var stdout, stderr bytes.Buffer
    71		code := runWithLauncherRuntime([]string{"launch"}, &stdout, &stderr, testLauncherRuntime("/home/me", "", "/work/pair"))
    72		if code != 3 {
    73			t.Fatalf("code = %d, want 3", code)
    74		}
    75		if stdout.String() != "" {
    76			t.Fatalf("stdout = %q, want empty", stdout.String())
    77		}
    78		for _, want := range []string{"prototype decision", "action=create", "tag=pair", "session=pair-pair"} {
    79			if !strings.Contains(stderr.String(), want) {
    80				t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
    81			}
    82		}
    83	}
    84	
    85	func testLauncherRuntime(home, xdg, cwd string) dispatcher.LauncherRuntime {
    86		return dispatcher.LauncherRuntime{
    87			Env:      dispatcher.LauncherEnv(home, xdg, cwd),
    88			Sessions: dispatcher.StaticSessions{},
    89			History:  dispatcher.StaticHistory{},
    90		}
    91	}

 succeeded in 0ms:
     1	package main
     2	
     3	import (
     4		"bytes"
     5		"os"
     6		"path/filepath"
     7		"strings"
     8		"testing"
     9		"time"
    10	)
    11	
    12	func TestRunLaunchWithFakeZellij(t *testing.T) {
    13		rt := t.TempDir()
    14		bin := filepath.Join(rt, "bin")
    15		data := filepath.Join(rt, "data")
    16		if err := os.MkdirAll(bin, 0o755); err != nil {
    17			t.Fatal(err)
    18		}
    19		if err := os.MkdirAll(filepath.Join(data, "pair"), 0o755); err != nil {
    20			t.Fatal(err)
    21		}
    22		log := filepath.Join(rt, "zellij.log")
    23		zellij := filepath.Join(bin, "zellij")
    24		script := `#!/usr/bin/env bash
    25	printf '%s\n' "$*" >> "` + log + `"
    26	case "$*" in
    27	  "list-sessions --short") printf 'pair-live\npair-detached\npair-exited\n' ;;
    28	  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-exited [Created] (EXITED - attach to resurrect)\n' ;;
    29	  "--session pair-live action list-clients") printf 'CLIENTS\n1\n' ;;
    30	  "--session pair-detached action list-clients") printf 'CLIENTS\n' ;;
    31	  *attach*|*new-session*|*--new-session-with-layout*|*delete-session*) printf 'MUTATING %s\n' "$*" >> "` + log + `"; exit 99 ;;
    32	  *) exit 1 ;;
    33	esac
    34	`
    35		if err := os.WriteFile(zellij, []byte(script), 0o755); err != nil {
    36			t.Fatal(err)
    37		}
    38		historical := filepath.Join(data, "pair", "draft-pair-old.md")
    39		if err := os.WriteFile(historical, []byte("draft"), 0o644); err != nil {
    40			t.Fatal(err)
    41		}
    42		now := time.Now()
    43		if err := os.Chtimes(historical, now, now); err != nil {
    44			t.Fatal(err)
    45		}
    46	
    47		t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
    48		t.Setenv("HOME", filepath.Join(rt, "home"))
    49		t.Setenv("XDG_DATA_HOME", data)
    50	
    51		var stdout, stderr bytes.Buffer
    52		code := run([]string{"launch", "claude"}, &stdout, &stderr)
    53		if code != 3 {
    54			t.Fatalf("code = %d, want 3; stderr:\n%s", code, stderr.String())
    55		}
    56		if stdout.String() != "" {
    57			t.Fatalf("stdout = %q, want empty", stdout.String())
    58		}
    59		for _, want := range []string{"prototype decision", "action=pick"} {
    60			if !strings.Contains(stderr.String(), want) {
    61				t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
    62			}
    63		}
    64		logBytes, err := os.ReadFile(log)
    65		if err != nil {
    66			t.Fatal(err)
    67		}
    68		if strings.Contains(string(logBytes), "MUTATING") {
    69			t.Fatalf("fake zellij recorded mutating invocation:\n%s", string(logBytes))
    70		}
    71	}

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
    20					"Implemented prototype commands:",
    21					"launch",
    22					"decision-phase only",
    23					"wrap",
    24					"slug",
    25					"not implemented in this skeleton",
    26				} {
    27					if !strings.Contains(res.Stdout, want) {
    28						t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
    29					}
    30				}
    31				if strings.Contains(res.Stdout, "launch             session lifecycle and public pair launcher flow (planned; not implemented") {
    32					t.Fatalf("Stdout still labels launch unimplemented:\n%s", res.Stdout)
    33				}
    34			})
    35		}
    36	}
    37	
    38	func TestDispatchVersionIsDevelopmentSkeletonMetadata(t *testing.T) {
    39		res := Dispatch([]string{"version"})
    40		if res.ExitCode != 0 {
    41			t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
    42		}
    43		if res.Stderr != "" {
    44			t.Fatalf("Stderr = %q, want empty", res.Stderr)
    45		}
    46		for _, want := range []string{"pair-go", "dispatcher skeleton", "public launcher: bin/pair"} {
    47			if !strings.Contains(res.Stdout, want) {
    48				t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
    49			}
    50		}
    51	}
    52	
    53	func TestDispatchPlannedCommandReturnsUnsupported(t *testing.T) {
    54		res := Dispatch([]string{"wrap"})
    55		if res.ExitCode != 2 {
    56			t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
    57		}
    58		if res.Stdout != "" {
    59			t.Fatalf("Stdout = %q, want empty", res.Stdout)
    60		}
    61		for _, want := range []string{"wrap", "planned", "not implemented", "pair-go help"} {
    62			if !strings.Contains(res.Stderr, want) {
    63				t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
    64			}
    65		}
    66	}
    67	
    68	func TestDispatchLaunchHelpRoutesToPrototype(t *testing.T) {
    69		res := Dispatch([]string{"launch", "--help"})
    70		if res.ExitCode != 0 {
    71			t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
    72		}
    73		if res.Stderr != "" {
    74			t.Fatalf("Stderr = %q, want empty", res.Stderr)
    75		}
    76		for _, want := range []string{"Usage: pair-go launch", "decision-phase prototype"} {
    77			if !strings.Contains(res.Stdout, want) {
    78				t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
    79			}
    80		}
    81	}
    82	
    83	func TestDispatchLaunchReturnsPrototypeDecision(t *testing.T) {
    84		res := DispatchWithLauncherRuntime([]string{"launch", "resume", "demo"}, LauncherRuntime{
    85			Env: LauncherEnv("/home/me", "", "/work/pair"),
    86			Sessions: StaticSessions{
    87				Sessions: nil,
    88			},
    89			History: StaticHistory{},
    90		})
    91		if res.ExitCode != 3 {
    92			t.Fatalf("ExitCode = %d, want 3", res.ExitCode)
    93		}
    94		if res.Stdout != "" {
    95			t.Fatalf("Stdout = %q, want empty", res.Stdout)
    96		}
    97		for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
    98			if !strings.Contains(res.Stderr, want) {
    99				t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
   100			}
   101		}
   102	}
   103	
   104	func TestDispatchLaunchWithoutArgsReturnsDefaultPrototypeDecision(t *testing.T) {
   105		res := DispatchWithLauncherRuntime([]string{"launch"}, LauncherRuntime{
   106			Env: LauncherEnv("/home/me", "", "/work/pair"),
   107			Sessions: StaticSessions{
   108				Sessions: nil,
   109			},
   110			History: StaticHistory{},
   111		})
   112		if res.ExitCode != 3 {
   113			t.Fatalf("ExitCode = %d, want 3", res.ExitCode)
   114		}
   115		if res.Stdout != "" {
   116			t.Fatalf("Stdout = %q, want empty", res.Stdout)
   117		}
   118		for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=pair", "session=pair-pair"} {
   119			if !strings.Contains(res.Stderr, want) {
   120				t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
   121			}
   122		}
   123	}
   124	
   125	func TestDispatchUnknownCommandReturnsUsageHint(t *testing.T) {
   126		res := Dispatch([]string{"frobnicate"})
   127		if res.ExitCode != 2 {
   128			t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
   129		}
   130		if res.Stdout != "" {
   131			t.Fatalf("Stdout = %q, want empty", res.Stdout)
   132		}
   133		for _, want := range []string{"unknown command", "frobnicate", "pair-go help"} {
   134			if !strings.Contains(res.Stderr, want) {
   135				t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
   136			}
   137		}
   138	}

exec
/bin/zsh -lc "nl -ba cmd/internal/launcher/decision_test.go cmd/internal/launcher/run_test.go cmd/internal/launcher/zellij_test.go | sed -n '1,260p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
usage: nl [-p] [-b type] [-d delim] [-f type] [-h type] [-i incr] [-l num]
          [-n format] [-s sep] [-v startnum] [-w width] [file]

codex
The tests now include the previously missing no-arg `launch` dispatcher and runner paths, plus the fake-zellij process-level non-mutation check. I’m doing one final verification sweep before writing the verdict.
exec
/bin/zsh -lc 'git diff --exit-code e89e2cea484ab8ca7ec9ca518fd2bf04ba38a8fe..HEAD -- bin/pair' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
exec
/bin/zsh -lc 'git diff --check e89e2cea484ab8ca7ec9ca518fd2bf04ba38a8fe..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
exec
/bin/zsh -lc 'rg -n "pair-go launch|decision-phase|shell-owned|cmd/internal/launcher" atlas/architecture.md atlas/go-migration-inventory.md' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
atlas/go-migration-inventory.md:50:| `bin/pair` / `cmd/internal/launcher` | Bash public launcher plus guarded Go decision prototype | user shell, `bin/pair-dev`, restart re-exec, tests, development-only `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch` (#75) parses a documented subset, snapshots zellij/history state, prints the decision-phase prototype action, and exits before mutating zellij. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/launcher` resolves data-dir inputs and reads zellij/history through fakeable seams. | go-entrypoint with `bin/pair` retained as compat fallback through #77; #75 owns decision-phase only, while real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
atlas/architecture.md:49:`pair-go launch` decision-phase prototype. The public launcher remains
atlas/architecture.md:50:`bin/pair`; `pair-go launch` parses the launcher subset, resolves the Pair data
atlas/architecture.md:52:decisions through the pure `cmd/internal/launcher` core. It then stops with an
atlas/architecture.md:55:rename, config/session migration, and title-poller behavior remain shell-owned

exec
/bin/zsh -lc 'go test ./... -count=1' in /Users/xianxu/workspace/pair
 succeeded in 15460ms:
ok  	github.com/xianxu/pair/cmd/internal/adapt	0.332s
ok  	github.com/xianxu/pair/cmd/internal/ctxmeter	0.472s
ok  	github.com/xianxu/pair/cmd/internal/dispatcher	0.191s
ok  	github.com/xianxu/pair/cmd/internal/launcher	2.143s
ok  	github.com/xianxu/pair/cmd/internal/model	3.419s
ok  	github.com/xianxu/pair/cmd/internal/transcript	0.847s
ok  	github.com/xianxu/pair/cmd/pair-changelog	15.131s
ok  	github.com/xianxu/pair/cmd/pair-context	3.561s
ok  	github.com/xianxu/pair/cmd/pair-continuation	5.527s
ok  	github.com/xianxu/pair/cmd/pair-go	2.427s
?   	github.com/xianxu/pair/cmd/pair-scribe	[no test files]
ok  	github.com/xianxu/pair/cmd/pair-scrollback-render	1.000s
ok  	github.com/xianxu/pair/cmd/pair-slug	6.496s
ok  	github.com/xianxu/pair/cmd/pair-wrap	1.820s

codex
VERDICT: SHIP (confidence: high)

The boundary now matches the issue’s Spec/Plan: `pair-go launch` is guarded, reaches the launcher decision phase, keeps `bin/pair` as the public launcher, and has focused unit plus fake-process coverage. I found no Critical or Important blockers.

1. Strengths:
- Pure launcher decision logic is separated in [decision.go](/Users/xianxu/workspace/pair/cmd/internal/launcher/decision.go:21), with IO injected through `Run` (`ARCH-PURE` pass).
- Dispatcher reuses the existing `dispatcher.Result` process boundary instead of introducing a parallel result type (`ARCH-DRY` pass).
- Fake-zellij process test verifies the prototype reads state but does not invoke mutating zellij actions: [launch_process_test.go](/Users/xianxu/workspace/pair/cmd/pair-go/launch_process_test.go:12).
- Atlas updates document the new guarded launcher surface and remaining shell-owned behavior: [architecture.md](/Users/xianxu/workspace/pair/atlas/architecture.md:49).

2. Critical findings: none.

3. Important findings: none.

4. Minor findings: none.

5. Test coverage notes:
- Ran `go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1`: pass.
- Ran `go test ./... -count=1`: pass.
- Ran `git diff --exit-code ... -- bin/pair`: pass, public launcher unchanged.
- Ran `git diff --check ...`: pass.

6. Architectural notes for upcoming work:
- `ARCH-DRY`: pass. The new route extends the existing dispatcher.
- `ARCH-PURE`: pass. Business decisions are pure; zellij/history are injectable seams.
- `ARCH-PURPOSE`: pass. The slice exercises a coherent launcher decision surface rather than only porting helpers.

7. Plan revision recommendations: none.
tokens used
85,547
VERDICT: SHIP (confidence: high)

The boundary now matches the issue’s Spec/Plan: `pair-go launch` is guarded, reaches the launcher decision phase, keeps `bin/pair` as the public launcher, and has focused unit plus fake-process coverage. I found no Critical or Important blockers.

1. Strengths:
- Pure launcher decision logic is separated in [decision.go](/Users/xianxu/workspace/pair/cmd/internal/launcher/decision.go:21), with IO injected through `Run` (`ARCH-PURE` pass).
- Dispatcher reuses the existing `dispatcher.Result` process boundary instead of introducing a parallel result type (`ARCH-DRY` pass).
- Fake-zellij process test verifies the prototype reads state but does not invoke mutating zellij actions: [launch_process_test.go](/Users/xianxu/workspace/pair/cmd/pair-go/launch_process_test.go:12).
- Atlas updates document the new guarded launcher surface and remaining shell-owned behavior: [architecture.md](/Users/xianxu/workspace/pair/atlas/architecture.md:49).

2. Critical findings: none.

3. Important findings: none.

4. Minor findings: none.

5. Test coverage notes:
- Ran `go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1`: pass.
- Ran `go test ./... -count=1`: pass.
- Ran `git diff --exit-code ... -- bin/pair`: pass, public launcher unchanged.
- Ran `git diff --check ...`: pass.

6. Architectural notes for upcoming work:
- `ARCH-DRY`: pass. The new route extends the existing dispatcher.
- `ARCH-PURE`: pass. Business decisions are pure; zellij/history are injectable seams.
- `ARCH-PURPOSE`: pass. The slice exercises a coherent launcher decision surface rather than only porting helpers.

7. Plan revision recommendations: none.
