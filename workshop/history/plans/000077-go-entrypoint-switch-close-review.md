# Boundary Review — pair#77 (whole-issue close)

| field | value |
|-------|-------|
| issue | 77 — pair Go entrypoint switch |
| repo | pair |
| issue file | workshop/issues/000077-go-entrypoint-switch.md |
| boundary | whole-issue close |
| milestone | — |
| window | b829ef00346738e3c3d7e40db38c2c03f51e615b..HEAD |
| command | sdlc close --issue 77 |
| reviewer | codex |
| timestamp | 2026-06-30T15:32:46-07:00 |
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
session id: 019f1aa7-cafd-7b23-acc6-61ab6438b466
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **pair** repository.

- repository: pair   (root: /Users/xianxu/workspace/pair)
- issue:      pair#77   (file: workshop/issues/000077-go-entrypoint-switch.md)
- window:     Base: b829ef00346738e3c3d7e40db38c2c03f51e615b   Head: HEAD

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
diff --git a/Makefile.local b/Makefile.local
index ccdde68..6bbc2eb 100644
--- a/Makefile.local
+++ b/Makefile.local
@@ -239,9 +239,10 @@ $(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill
 $(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-continuation/git.go cmd/pair-continuation/main.go go.mod
 	go build -o $@ ./cmd/pair-continuation
 
-# pair-go — development-only dispatcher skeleton for the future primary Go CLI.
-# The public launcher remains bin/pair until the entrypoint-switch issue.
-$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
+# pair-go — development dispatcher and Go-owned launch handoff.
+# `pair-go launch ...` execs sibling bin/pair for the current migration window;
+# helper routes stay in the dispatcher.
+$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/entrypoint/launch.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
 	go build -o $@ ./cmd/pair-go
 
 
diff --git a/README.md b/README.md
index c4681a3..dcde60a 100644
--- a/README.md
+++ b/README.md
@@ -200,6 +200,13 @@ pair rename <old> <new>          # rename every tag-scoped file in
 pair -h, --help                  # show full help
 ```
 
+`pair-go launch ...` is the Go-owned migration entrypoint for testing the
+launcher path. It accepts the same arguments after `launch` that `pair` accepts
+directly, then hands off to the current `pair` launcher for one migration
+window. In a dev shell sourced from `../ariadne/construct/dev-aliases.sh`,
+`pair-go` rebuilds from `cmd/pair-go` automatically before running; no
+`pair-go-dev` command is needed.
+
 Use `--` to separate pair's positional from agent flags. Without it, pair only takes `<agent>` as a positional and everything else is rejected.
 
 Agent args (after `--`) are appended to the agent command line on **create**. Reattaching to an existing session does not re-launch the agent, so the args don't apply on attach. (The picker connects you to whatever's already running.)
diff --git a/atlas/architecture.md b/atlas/architecture.md
index 5e308ab..c8df19e 100644
--- a/atlas/architecture.md
+++ b/atlas/architecture.md
@@ -14,7 +14,7 @@ contract for the Go packaging migration lives in
 
 ```
 bin/pair                     # entry point (launcher)
-bin/pair-go                  # opt-in Go dispatcher skeleton; public launcher remains bin/pair
+bin/pair-go                  # opt-in Go dispatcher + launch handoff; public launcher remains bin/pair
 bin/clipboard-to-pane.sh     # read clipboard, hand off to nvim's PairPasteQuote
 bin/copy-on-select.sh        # invoked by zellij copy_command on mouse-up
 bin/pair-quit.sh             # invoked by Alt+x — marks + kills session
@@ -45,15 +45,19 @@ internal subcommands or dispatch modes behind that primary binary (`pair wrap`,
 `pair continuation`, `pair scribe`) instead of staying as independently managed
 installed commands forever.
 
-As of #75, `bin/pair-go` is a development-only dispatcher with a guarded
-`pair-go launch` decision-phase prototype. The public launcher remains
-`bin/pair`; `pair-go launch` parses the launcher subset, resolves the Pair data
-dir, snapshots fakeable zellij/history state, and computes create/attach/picker
-decisions through the pure `cmd/internal/launcher` core. It then stops with an
-explicit prototype message before invoking zellij. Real zellij lifecycle,
-prompt/fzf UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation,
-rename, config/session migration, and title-poller behavior remain shell-owned
-until later migration issues.
+As of #77, `bin/pair-go` is the Go-owned launch entrypoint under test:
+`pair-go launch ...` resolves its own executable, finds sibling `bin/pair`, and
+execs it with the same argv/env that `pair` would have received directly. This
+makes `pair-go launch claude`, `pair-go launch resume <tag>`, `pair-go launch
+continue ...`, `pair-go launch list`, and `pair-go launch rename ...`
+meaningful dogfood commands without replacing the stable public `pair` command
+yet. A developer shell sourced from `../ariadne/construct/dev-aliases.sh`
+rebuilds `cmd/pair-go` automatically; no `pair-go-dev` command exists.
+
+The earlier #75 pure launcher core remains available as internal decision logic,
+but real zellij lifecycle, prompt/fzf UI, restart/quit cleanup, cmux ownership,
+dev rebuild, continuation, rename, config/session migration, and title-poller
+behavior remain shell-owned through `bin/pair` until later migration issues.
 
 As of #76, the same dispatcher also has the first implemented helper routes:
 `pair-go context <tag> <agent>` and `pair-go scrollback-render ...`. Both routes
diff --git a/atlas/go-migration-inventory.md b/atlas/go-migration-inventory.md
index 4a691ee..5c4c6bc 100644
--- a/atlas/go-migration-inventory.md
+++ b/atlas/go-migration-inventory.md
@@ -47,7 +47,7 @@ Priority is packaging impact first, then reliability/testability:
 
 | Artifact | Type | Callers | Runtime contract | Files/env | Disposition | Priority |
 |---|---|---|---|---|---|---|
-| `bin/pair` / `cmd/internal/launcher` | Bash public launcher plus guarded Go decision prototype | user shell, `bin/pair-dev`, restart re-exec, tests, development-only `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch` (#75) parses a documented subset, snapshots zellij/history state, prints the decision-phase prototype action, and exits before mutating zellij. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/launcher` resolves data-dir inputs and reads zellij/history through fakeable seams. | go-entrypoint with `bin/pair` retained as compat fallback through #77; #75 owns decision-phase only, while real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
+| `bin/pair` / `cmd/internal/launcher` / `cmd/internal/entrypoint` | Bash public launcher plus Go-owned launch handoff | user shell, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` (#77) resolves sibling `bin/pair` from the `pair-go` executable and execs it with `pair`-compatible argv/env. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves the compatibility handoff; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Go handoff entrypoint with `bin/pair` retained as stable public launcher and compat target through #77; real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
 | `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling `pair`. | Resolves symlinks; depends on `bin/pair` and `bin/lib/dev-rebuild.sh`. | compat-shim, likely stays as developer wrapper until Go entrypoint has dev mode | P1 |
 | `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
 | `zellij/layouts/main.kdl` | zellij native asset | `bin/pair` via `zellij --new-session-with-layout` | Defines agent and draft panes; shell expands Pair env at pane start. | Calls `pair-wrap`; calls `nvim -u "$PAIR_HOME/nvim/init.lua"`; writes `pane-<tag>-<agent>.json`; writes draft nvim pid file. | native-asset, packaged adjacent/embedded | P0 |
@@ -129,7 +129,9 @@ Build/install callers:
   `pair-go scrollback-render`, backed by shared internal runners while legacy
   binary names remain live for shell/Lua callers. `pair slug` remains a later
   candidate.
-- #77 is the public entrypoint switch and should wait for both #75 and #76.
+- #77 made `pair-go launch ...` a meaningful Go-owned compatibility handoff to
+  `bin/pair`, with argv/env preserved and missing-launcher diagnostics. The
+  stable public `pair` script remains unchanged for this migration window.
 - #78 should prioritize `pair-title.sh` and `pair-session-watch.sh` if stateful
   shell remains a packaging/reliability problem after #77.
 - #79 owns whether `nvim/` and `zellij/` are embedded or installed adjacent.
diff --git a/cmd/internal/dispatcher/dispatcher.go b/cmd/internal/dispatcher/dispatcher.go
index 3784f1e..fe1fbfd 100644
--- a/cmd/internal/dispatcher/dispatcher.go
+++ b/cmd/internal/dispatcher/dispatcher.go
@@ -3,12 +3,9 @@ package dispatcher
 import (
 	"bytes"
 	"fmt"
-	"os"
 	"strings"
-	"time"
 
 	"github.com/xianxu/pair/cmd/internal/contextcmd"
-	"github.com/xianxu/pair/cmd/internal/launcher"
 	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
 )
 
@@ -31,7 +28,7 @@ type Result struct {
 // Families returns the planned command families for the Go dispatcher.
 func Families() []CommandFamily {
 	return []CommandFamily{
-		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "prototype"},
+		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "handoff"},
 		{Name: "context", Summary: "agent pane context meter", Status: "implemented"},
 		{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "implemented"},
 		{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
@@ -53,11 +50,11 @@ func Dispatch(args []string) Result {
 		return Result{Stdout: Help(programName), ExitCode: 0}
 	case "version", "--version":
 		return Result{
-			Stdout:   "pair-go dispatcher skeleton\npublic launcher: bin/pair\n",
+			Stdout:   "pair-go dispatcher skeleton\nlaunch handoff: bin/pair\n",
 			ExitCode: 0,
 		}
 	case "launch":
-		return DispatchWithLauncherRuntime(args, osLauncherRuntime())
+		return launchHandoffResult()
 	case "context":
 		return dispatchContext(args[1:])
 	case "scrollback-render":
@@ -89,90 +86,10 @@ func dispatchScrollbackRender(args []string) Result {
 	return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}
 }
 
-type LauncherRuntime struct {
-	Env      launcher.Env
-	Sessions launcher.SessionSource
-	History  launcher.HistoricalScanner
-}
-
-type StaticSessions struct {
-	Sessions []launcher.Session
-	Err      error
-}
-
-func (s StaticSessions) Snapshot() ([]launcher.Session, error) {
-	if s.Err != nil {
-		return nil, s.Err
-	}
-	return s.Sessions, nil
-}
-
-type StaticHistory struct {
-	Tags []launcher.HistoricalTag
-	Err  error
-}
-
-func (h StaticHistory) Scan(_ string, _ time.Time) ([]launcher.HistoricalTag, error) {
-	if h.Err != nil {
-		return nil, h.Err
-	}
-	return h.Tags, nil
-}
-
-func DispatchWithLauncherRuntime(args []string, rt LauncherRuntime) Result {
-	launchArgs := []string(nil)
-	if len(args) > 1 {
-		launchArgs = args[1:]
-	}
-	if len(launchArgs) > 0 && (launchArgs[0] == "help" || launchArgs[0] == "--help" || launchArgs[0] == "-h") {
-		return Result{Stdout: LaunchHelp(programName), ExitCode: 0}
-	}
-	outcome, err := launcher.Run(launchArgs, rt.Env, rt.Sessions, rt.History)
-	if err != nil {
-		return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
-	}
-	decision := outcome.Decision
+func launchHandoffResult() Result {
 	return Result{
-		Stderr: fmt.Sprintf(
-			"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
-			decision.Action,
-			decision.Tag,
-			decision.SessionName,
-		),
-		ExitCode: 3,
-	}
-}
-
-func LaunchHelp(program string) string {
-	return fmt.Sprintf(`Usage: %s launch [agent] [-- agent-args...]
-       %s launch resume <tag>
-
-Guarded decision-phase prototype. Public sessions still start through bin/pair.
-This command parses launch inputs and computes the create/attach/picker decision,
-then stops before invoking zellij.
-`, program, program)
-}
-
-func LauncherEnv(home, xdgDataHome, cwd string) launcher.Env {
-	return launcher.Env{
-		Home:     home,
-		XDGData:  xdgDataHome,
-		Cwd:      cwd,
-		Now:      time.Now(),
-		HistoryD: 14,
-	}
-}
-
-func osLauncherRuntime() LauncherRuntime {
-	home := os.Getenv("HOME")
-	xdg := os.Getenv("XDG_DATA_HOME")
-	cwd, _ := os.Getwd()
-	env := LauncherEnv(home, xdg, cwd)
-	dataDir := launcher.ResolveDataDir(home, xdg)
-	return LauncherRuntime{
-		Env:      env,
-		Sessions: launcher.ZellijSource{},
-		History:  launcher.HistorySource{DataDir: dataDir},
+		Stderr:   "pair-go launch is a process handoff implemented by cmd/pair-go; call pair-go launch ... instead of dispatcher.Dispatch\n",
+		ExitCode: 2,
 	}
 }
 
@@ -185,6 +102,8 @@ func Help(program string) string {
 	for _, family := range Families() {
 		if family.Status == "prototype" {
 			fmt.Fprintf(&b, "  %-17s %s (prototype; decision-phase only)\n", family.Name, family.Summary)
+		} else if family.Status == "handoff" {
+			fmt.Fprintf(&b, "  %-17s %s (compatibility handoff to bin/pair)\n", family.Name, family.Summary)
 		} else if family.Status == "implemented" {
 			fmt.Fprintf(&b, "  %-17s %s (implemented helper route)\n", family.Name, family.Summary)
 		}
diff --git a/cmd/internal/dispatcher/dispatcher_test.go b/cmd/internal/dispatcher/dispatcher_test.go
index f10f32b..7dc3021 100644
--- a/cmd/internal/dispatcher/dispatcher_test.go
+++ b/cmd/internal/dispatcher/dispatcher_test.go
@@ -21,7 +21,7 @@ func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
 				"Usage: pair-go <command> [args]",
 				"Implemented commands:",
 				"launch",
-				"decision-phase only",
+				"compatibility handoff",
 				"context",
 				"scrollback-render",
 				"wrap",
@@ -35,6 +35,9 @@ func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
 			if strings.Contains(res.Stdout, "launch             session lifecycle and public pair launcher flow (planned; not implemented") {
 				t.Fatalf("Stdout still labels launch unimplemented:\n%s", res.Stdout)
 			}
+			if strings.Contains(res.Stdout, "decision-phase only") {
+				t.Fatalf("Stdout still labels launch decision-phase only:\n%s", res.Stdout)
+			}
 			for _, stale := range []string{
 				"context           agent pane context meter (planned; not implemented",
 				"scrollback-render raw PTY capture to ANSI scrollback (planned; not implemented",
@@ -55,7 +58,7 @@ func TestDispatchVersionIsDevelopmentSkeletonMetadata(t *testing.T) {
 	if res.Stderr != "" {
 		t.Fatalf("Stderr = %q, want empty", res.Stderr)
 	}
-	for _, want := range []string{"pair-go", "dispatcher skeleton", "public launcher: bin/pair"} {
+	for _, want := range []string{"pair-go", "dispatcher skeleton", "launch handoff: bin/pair"} {
 		if !strings.Contains(res.Stdout, want) {
 			t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
 		}
@@ -77,57 +80,15 @@ func TestDispatchPlannedCommandReturnsUnsupported(t *testing.T) {
 	}
 }
 
-func TestDispatchLaunchHelpRoutesToPrototype(t *testing.T) {
+func TestDispatchLaunchReportsProcessHandoff(t *testing.T) {
 	res := Dispatch([]string{"launch", "--help"})
-	if res.ExitCode != 0 {
-		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
-	}
-	if res.Stderr != "" {
-		t.Fatalf("Stderr = %q, want empty", res.Stderr)
-	}
-	for _, want := range []string{"Usage: pair-go launch", "decision-phase prototype"} {
-		if !strings.Contains(res.Stdout, want) {
-			t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
-		}
-	}
-}
-
-func TestDispatchLaunchReturnsPrototypeDecision(t *testing.T) {
-	res := DispatchWithLauncherRuntime([]string{"launch", "resume", "demo"}, LauncherRuntime{
-		Env: LauncherEnv("/home/me", "", "/work/pair"),
-		Sessions: StaticSessions{
-			Sessions: nil,
-		},
-		History: StaticHistory{},
-	})
-	if res.ExitCode != 3 {
-		t.Fatalf("ExitCode = %d, want 3", res.ExitCode)
-	}
-	if res.Stdout != "" {
-		t.Fatalf("Stdout = %q, want empty", res.Stdout)
-	}
-	for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
-		if !strings.Contains(res.Stderr, want) {
-			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
-		}
-	}
-}
-
-func TestDispatchLaunchWithoutArgsReturnsDefaultPrototypeDecision(t *testing.T) {
-	res := DispatchWithLauncherRuntime([]string{"launch"}, LauncherRuntime{
-		Env: LauncherEnv("/home/me", "", "/work/pair"),
-		Sessions: StaticSessions{
-			Sessions: nil,
-		},
-		History: StaticHistory{},
-	})
-	if res.ExitCode != 3 {
-		t.Fatalf("ExitCode = %d, want 3", res.ExitCode)
+	if res.ExitCode != 2 {
+		t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
 	}
 	if res.Stdout != "" {
 		t.Fatalf("Stdout = %q, want empty", res.Stdout)
 	}
-	for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=pair", "session=pair-pair"} {
+	for _, want := range []string{"pair-go launch", "process handoff", "cmd/pair-go"} {
 		if !strings.Contains(res.Stderr, want) {
 			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
 		}
diff --git a/cmd/internal/entrypoint/launch.go b/cmd/internal/entrypoint/launch.go
new file mode 100644
index 0000000..1a40970
--- /dev/null
+++ b/cmd/internal/entrypoint/launch.go
@@ -0,0 +1,20 @@
+package entrypoint
+
+import "path/filepath"
+
+// LegacyLaunchRequest describes the current compatibility handoff from
+// pair-go launch to the shell-owned pair launcher.
+type LegacyLaunchRequest struct {
+	Path string
+	Argv []string
+}
+
+func ResolveLegacyLaunch(executable string, launchArgs []string) LegacyLaunchRequest {
+	argv := make([]string, 0, len(launchArgs)+1)
+	argv = append(argv, "pair")
+	argv = append(argv, launchArgs...)
+	return LegacyLaunchRequest{
+		Path: filepath.Join(filepath.Dir(executable), "pair"),
+		Argv: argv,
+	}
+}
diff --git a/cmd/internal/entrypoint/launch_test.go b/cmd/internal/entrypoint/launch_test.go
new file mode 100644
index 0000000..cdeec2c
--- /dev/null
+++ b/cmd/internal/entrypoint/launch_test.go
@@ -0,0 +1,28 @@
+package entrypoint
+
+import (
+	"reflect"
+	"testing"
+)
+
+func TestResolveLegacyLaunchDropsLaunchVerb(t *testing.T) {
+	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"claude", "--", "--resume"})
+	if req.Path != "/repo/bin/pair" {
+		t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
+	}
+	want := []string{"pair", "claude", "--", "--resume"}
+	if !reflect.DeepEqual(req.Argv, want) {
+		t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
+	}
+}
+
+func TestResolveLegacyLaunchPreservesSubcommands(t *testing.T) {
+	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"resume", "demo"})
+	if req.Path != "/repo/bin/pair" {
+		t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
+	}
+	want := []string{"pair", "resume", "demo"}
+	if !reflect.DeepEqual(req.Argv, want) {
+		t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
+	}
+}
diff --git a/cmd/pair-go/launch_process_test.go b/cmd/pair-go/launch_process_test.go
deleted file mode 100644
index 9563492..0000000
--- a/cmd/pair-go/launch_process_test.go
+++ /dev/null
@@ -1,71 +0,0 @@
-package main
-
-import (
-	"bytes"
-	"os"
-	"path/filepath"
-	"strings"
-	"testing"
-	"time"
-)
-
-func TestRunLaunchWithFakeZellij(t *testing.T) {
-	rt := t.TempDir()
-	bin := filepath.Join(rt, "bin")
-	data := filepath.Join(rt, "data")
-	if err := os.MkdirAll(bin, 0o755); err != nil {
-		t.Fatal(err)
-	}
-	if err := os.MkdirAll(filepath.Join(data, "pair"), 0o755); err != nil {
-		t.Fatal(err)
-	}
-	log := filepath.Join(rt, "zellij.log")
-	zellij := filepath.Join(bin, "zellij")
-	script := `#!/usr/bin/env bash
-printf '%s\n' "$*" >> "` + log + `"
-case "$*" in
-  "list-sessions --short") printf 'pair-live\npair-detached\npair-exited\n' ;;
-  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-exited [Created] (EXITED - attach to resurrect)\n' ;;
-  "--session pair-live action list-clients") printf 'CLIENTS\n1\n' ;;
-  "--session pair-detached action list-clients") printf 'CLIENTS\n' ;;
-  *attach*|*new-session*|*--new-session-with-layout*|*delete-session*) printf 'MUTATING %s\n' "$*" >> "` + log + `"; exit 99 ;;
-  *) exit 1 ;;
-esac
-`
-	if err := os.WriteFile(zellij, []byte(script), 0o755); err != nil {
-		t.Fatal(err)
-	}
-	historical := filepath.Join(data, "pair", "draft-pair-old.md")
-	if err := os.WriteFile(historical, []byte("draft"), 0o644); err != nil {
-		t.Fatal(err)
-	}
-	now := time.Now()
-	if err := os.Chtimes(historical, now, now); err != nil {
-		t.Fatal(err)
-	}
-
-	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
-	t.Setenv("HOME", filepath.Join(rt, "home"))
-	t.Setenv("XDG_DATA_HOME", data)
-
-	var stdout, stderr bytes.Buffer
-	code := run([]string{"launch", "claude"}, &stdout, &stderr)
-	if code != 3 {
-		t.Fatalf("code = %d, want 3; stderr:\n%s", code, stderr.String())
-	}
-	if stdout.String() != "" {
-		t.Fatalf("stdout = %q, want empty", stdout.String())
-	}
-	for _, want := range []string{"prototype decision", "action=pick"} {
-		if !strings.Contains(stderr.String(), want) {
-			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
-		}
-	}
-	logBytes, err := os.ReadFile(log)
-	if err != nil {
-		t.Fatal(err)
-	}
-	if strings.Contains(string(logBytes), "MUTATING") {
-		t.Fatalf("fake zellij recorded mutating invocation:\n%s", string(logBytes))
-	}
-}
diff --git a/cmd/pair-go/main.go b/cmd/pair-go/main.go
index bf7d28b..760f876 100644
--- a/cmd/pair-go/main.go
+++ b/cmd/pair-go/main.go
@@ -1,12 +1,15 @@
-// pair-go is a development-only dispatcher skeleton for the future primary Go
-// CLI. The public launcher remains bin/pair.
+// pair-go is the development dispatcher for the future primary Go CLI. Its
+// launch route is a compatibility handoff to the current shell launcher.
 package main
 
 import (
+	"fmt"
 	"io"
 	"os"
+	"syscall"
 
 	"github.com/xianxu/pair/cmd/internal/dispatcher"
+	"github.com/xianxu/pair/cmd/internal/entrypoint"
 )
 
 func main() {
@@ -14,13 +17,65 @@ func main() {
 }
 
 func run(args []string, stdout, stderr io.Writer) int {
+	return runWithLegacyRuntime(args, stdout, stderr, osLegacyRuntime{})
+}
+
+type legacyRuntime interface {
+	Executable() (string, error)
+	Stat(path string) error
+	Environ() []string
+	Exec(path string, argv []string, env []string) int
+}
+
+func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
+	if len(args) > 0 && args[0] == "launch" {
+		return runLegacyLaunch(args[1:], stderr, rt)
+	}
 	res := dispatcher.Dispatch(args)
 	return writeResult(res, stdout, stderr)
 }
 
-func runWithLauncherRuntime(args []string, stdout, stderr io.Writer, rt dispatcher.LauncherRuntime) int {
-	res := dispatcher.DispatchWithLauncherRuntime(args, rt)
-	return writeResult(res, stdout, stderr)
+func runLegacyLaunch(args []string, stderr io.Writer, rt legacyRuntime) int {
+	exe, err := rt.Executable()
+	if err != nil {
+		_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
+		return 1
+	}
+	req := entrypoint.ResolveLegacyLaunch(exe, args)
+	if err := rt.Stat(req.Path); err != nil {
+		_, _ = fmt.Fprintf(stderr, "pair-go launch: pair launcher not found at %s (%v); run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", req.Path, err)
+		return 1
+	}
+	return rt.Exec(req.Path, req.Argv, rt.Environ())
+}
+
+type osLegacyRuntime struct{}
+
+func (osLegacyRuntime) Executable() (string, error) {
+	return os.Executable()
+}
+
+func (osLegacyRuntime) Stat(path string) error {
+	info, err := os.Stat(path)
+	if err != nil {
+		return err
+	}
+	if info.IsDir() {
+		return fmt.Errorf("is a directory")
+	}
+	return nil
+}
+
+func (osLegacyRuntime) Environ() []string {
+	return os.Environ()
+}
+
+func (osLegacyRuntime) Exec(path string, argv []string, env []string) int {
+	if err := syscall.Exec(path, argv, env); err != nil {
+		_, _ = fmt.Fprintf(os.Stderr, "pair-go launch: exec %s failed: %v\n", path, err)
+		return 1
+	}
+	return 0
 }
 
 func writeResult(res dispatcher.Result, stdout, stderr io.Writer) int {
diff --git a/cmd/pair-go/main_test.go b/cmd/pair-go/main_test.go
index b0d6f7f..d137a6b 100644
--- a/cmd/pair-go/main_test.go
+++ b/cmd/pair-go/main_test.go
@@ -2,10 +2,11 @@ package main
 
 import (
 	"bytes"
+	"errors"
+	"os"
+	"reflect"
 	"strings"
 	"testing"
-
-	"github.com/xianxu/pair/cmd/internal/dispatcher"
 )
 
 func TestRunWritesStdoutAndReturnsDispatcherCode(t *testing.T) {
@@ -37,55 +38,122 @@ func TestRunWritesStderrAndReturnsDispatcherCode(t *testing.T) {
 }
 
 func TestRunLaunchHelp(t *testing.T) {
+	rt := &fakeLegacyRuntime{
+		executable: "/repo/bin/pair-go",
+	}
 	var stdout, stderr bytes.Buffer
-	code := run([]string{"launch", "--help"}, &stdout, &stderr)
+	code := runWithLegacyRuntime([]string{"launch", "--help"}, &stdout, &stderr, rt)
 	if code != 0 {
 		t.Fatalf("code = %d, want 0", code)
 	}
-	if !strings.Contains(stdout.String(), "Usage: pair-go launch") {
-		t.Fatalf("stdout missing launch usage:\n%s", stdout.String())
+	if stdout.String() != "" {
+		t.Fatalf("stdout = %q, want empty", stdout.String())
 	}
 	if stderr.String() != "" {
 		t.Fatalf("stderr = %q, want empty", stderr.String())
 	}
+	if rt.execPath != "/repo/bin/pair" {
+		t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
+	}
+	wantArgv := []string{"pair", "--help"}
+	if !reflect.DeepEqual(rt.execArgv, wantArgv) {
+		t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
+	}
 }
 
-func TestRunLaunchResumeReturnsPrototypeDecision(t *testing.T) {
+func TestRunLaunchExecsLegacyPairWithArgvAndEnv(t *testing.T) {
+	t.Setenv("PAIR_TEST_ENV", "kept")
+	rt := &fakeLegacyRuntime{
+		executable: "/repo/bin/pair-go",
+		execCode:   42,
+	}
+
 	var stdout, stderr bytes.Buffer
-	code := runWithLauncherRuntime([]string{"launch", "resume", "demo"}, &stdout, &stderr, testLauncherRuntime("/home/me", "", "/work/pair"))
-	if code != 3 {
-		t.Fatalf("code = %d, want 3", code)
+	code := runWithLegacyRuntime([]string{"launch", "claude", "--", "--resume"}, &stdout, &stderr, rt)
+
+	if code != 42 {
+		t.Fatalf("code = %d, want 42", code)
 	}
 	if stdout.String() != "" {
 		t.Fatalf("stdout = %q, want empty", stdout.String())
 	}
-	for _, want := range []string{"prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
-		if !strings.Contains(stderr.String(), want) {
-			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
-		}
+	if stderr.String() != "" {
+		t.Fatalf("stderr = %q, want empty", stderr.String())
+	}
+	if rt.execPath != "/repo/bin/pair" {
+		t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
+	}
+	wantArgv := []string{"pair", "claude", "--", "--resume"}
+	if !reflect.DeepEqual(rt.execArgv, wantArgv) {
+		t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
+	}
+	if !containsEnv(rt.execEnv, "PAIR_TEST_ENV=kept") {
+		t.Fatalf("execEnv missing PAIR_TEST_ENV=kept: %#v", rt.execEnv)
 	}
 }
 
-func TestRunLaunchWithoutArgsReturnsDefaultPrototypeDecision(t *testing.T) {
+func TestRunLaunchReportsMissingLegacyPair(t *testing.T) {
+	rt := &fakeLegacyRuntime{
+		executable: "/repo/bin/pair-go",
+		statErr:    os.ErrNotExist,
+	}
+
 	var stdout, stderr bytes.Buffer
-	code := runWithLauncherRuntime([]string{"launch"}, &stdout, &stderr, testLauncherRuntime("/home/me", "", "/work/pair"))
-	if code != 3 {
-		t.Fatalf("code = %d, want 3", code)
+	code := runWithLegacyRuntime([]string{"launch", "claude"}, &stdout, &stderr, rt)
+
+	if code != 1 {
+		t.Fatalf("code = %d, want 1", code)
 	}
 	if stdout.String() != "" {
 		t.Fatalf("stdout = %q, want empty", stdout.String())
 	}
-	for _, want := range []string{"prototype decision", "action=create", "tag=pair", "session=pair-pair"} {
+	for _, want := range []string{"pair-go launch", "/repo/bin/pair", "make build", "make install", "dev-aliases.sh"} {
 		if !strings.Contains(stderr.String(), want) {
 			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
 		}
 	}
+	if rt.execPath != "" {
+		t.Fatalf("execPath = %q, want empty", rt.execPath)
+	}
 }
 
-func testLauncherRuntime(home, xdg, cwd string) dispatcher.LauncherRuntime {
-	return dispatcher.LauncherRuntime{
-		Env:      dispatcher.LauncherEnv(home, xdg, cwd),
-		Sessions: dispatcher.StaticSessions{},
-		History:  dispatcher.StaticHistory{},
+type fakeLegacyRuntime struct {
+	executable string
+	statErr    error
+	execCode   int
+
+	execPath string
+	execArgv []string
+	execEnv  []string
+}
+
+func (f *fakeLegacyRuntime) Executable() (string, error) {
+	if f.executable == "" {
+		return "", errors.New("missing executable")
+	}
+	return f.executable, nil
+}
+
+func (f *fakeLegacyRuntime) Stat(_ string) error {
+	return f.statErr
+}
+
+func (f *fakeLegacyRuntime) Environ() []string {
+	return os.Environ()
+}
+
+func (f *fakeLegacyRuntime) Exec(path string, argv []string, env []string) int {
+	f.execPath = path
+	f.execArgv = append([]string(nil), argv...)
+	f.execEnv = append([]string(nil), env...)
+	return f.execCode
+}
+
+func containsEnv(env []string, want string) bool {
+	for _, got := range env {
+		if got == want {
+			return true
+		}
 	}
+	return false
 }


exec
/bin/zsh -lc "sed -n '1,220p' /Users/xianxu/workspace/ariadne/construct/adapted/superpowers-using-superpowers/SKILL.md && sdlc --help && sed -n '1,220p' workshop/lessons.md && sed -n '1,240p' workshop/issues/000077-go-entrypoint-switch.md" in /Users/xianxu/workspace/pair
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

---
id: 000077
status: done
deps: [000074, 000075, 000076]
github_issue:
created: 2026-06-26
updated: 2026-06-30
estimate_hours: 2.6
started: 2026-06-30T12:42:11-07:00
actual_hours: 0.59
---

# pair Go entrypoint switch

## Problem

At some point the public `pair` command must become Go-owned. The next safe step is to make the Go-owned `pair-go launch ...` path exercise the real launcher contract while leaving the existing `pair` and `pair-dev` entrypoints stable.

## Spec

Make `pair-go launch ...` a meaningful Go entrypoint by having it hand off to the existing `bin/pair` launcher with `pair`-compatible arguments. `pair-go launch claude` should behave like `pair claude`; `pair-go launch resume <tag>`, `pair-go launch continue ...`, `pair-go launch list`, and `pair-go launch rename ...` should all pass through to the same shell-owned implementation for this migration window.

Keep `pair` and `pair-dev` working exactly as they do today. The Go command has no separate `-dev` variant: a developer shell sourced from `../ariadne/construct/dev-aliases.sh` already discovers `cmd/pair-go`, rebuilds `bin/pair-go` on every invocation, and then runs it from the caller's cwd. If `pair-go launch` cannot find the launcher beside the built binary, its diagnostic should point to `make build` / `make install` and the dev-alias path instead of failing with a bare exec error.

This deliberately keeps real zellij lifecycle, prompt/fzf UI, restart/quit cleanup, continuation, rename/list, and dev rebuild behavior shell-owned until later migration issues. `ARCH-PURPOSE`: #77's purpose is now the first meaningful Go-owned launch entrypoint without destabilizing the existing public command; full public `pair` replacement remains a later cutover once shell stateful glue is reduced.

## Done when

- [x] `pair-go launch ...` uses Go process code first and then invokes the real launcher with `pair`-compatible argv.
- [x] Existing `pair` remains the stable public entrypoint for one migration window.
- [x] `pair-dev` still rebuilds and launches the working tree behavior.
- [x] Existing create, attach, resume, continue, rename/list, quit, and restart flows are preserved through the `bin/pair` fallback.
- [x] The dev workflow is documented: `cmd/pair-go` is rebuilt by `../ariadne/construct/dev-aliases.sh`; no `pair-go-dev` command is needed.
- [x] Pair remains usable after merge; no keybinding workflow regresses.

## Plan

- [x] Confirm prerequisites from earlier Go migration issues.
- [x] Add tests for `pair-go launch` argv/env handoff to `bin/pair`.
- [x] Add stale/missing launcher diagnostics.
- [x] Implement the thin Go handoff while keeping dispatcher helper routes intact.
- [x] Verify `pair`, `pair-dev`, and `pair-go launch` behavior with process fakes and targeted builds.
- [x] Update README/atlas packaging notes.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.25 impl=0.15
item: greenfield-go-module design=0.35 impl=0.40
item: skill-or-dispatcher design=0.30 impl=0.35
item: atlas-docs design=0.20 impl=0.25
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 2.62
```

## Log

### 2026-06-26

Created from #72 as the public switch milestone. This should not be claimed until the earlier dispatcher/helper/launcher milestones have landed.

### 2026-06-30
- 2026-06-30: closed — go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1; go test ./... -count=1; make pair-go; make test-dev-rebuild; bin/pair-go help; bin/pair-go launch --help; bin/pair --help; bin/pair-dev --help

Re-scoped after operator guidance: keep `cmd/pair-go` as the Go entrypoint under test, leave `pair` / `pair-dev` stable, and rely on `../ariadne/construct/dev-aliases.sh` to rebuild `cmd/pair-go` in developer shells. `ARCH-DRY`: reuse the existing launcher for real zellij behavior instead of duplicating shell-owned lifecycle paths in Go. `ARCH-PURE`: keep launch path selection testable with a pure path/argv decision plus a thin exec boundary.

Plan-quality gate returned FAILURE because the plan promised argv/env handoff but did not explicitly test env propagation, and because `pair-dev --help` under-proved the dev rebuild acceptance criterion. Updated the durable plan to require an inherited-env fake-runner assertion and `make test-dev-rebuild` verification. `ARCH-PURPOSE`: compatibility claims must be pinned by tests, not implied by the shell fallback.

Implemented `cmd/internal/entrypoint` and the `cmd/pair-go` launch handoff. `pair-go launch ...` now resolves sibling `bin/pair`, validates it, and execs it with preserved argv/env; missing launcher diagnostics point to `make build`, `make install`, and `../ariadne/construct/dev-aliases.sh`. Removed the stale decision-phase `pair-go launch` dispatcher surface while keeping `context` and `scrollback-render` helper routes intact. Updated README and atlas migration notes.

Verification passed:

- `go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1`
- `go test ./... -count=1`
- `make pair-go`
- `make test-dev-rebuild`
- `bin/pair-go help`
- `bin/pair-go launch --help`
- `bin/pair --help`
- `bin/pair-dev --help`

codex
Using `superpowers-using-superpowers` for the mandatory session skill check, and treating this as the requested fresh-context SDLC boundary review. I’ve read the issue, SDLC contract, and lessons; now I’m checking the actual repository state and behavior against the diff.
exec
/bin/zsh -lc "nl -ba cmd/pair-go/main.go | sed -n '1,180p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	// pair-go is the development dispatcher for the future primary Go CLI. Its
     2	// launch route is a compatibility handoff to the current shell launcher.
     3	package main
     4	
     5	import (
     6		"fmt"
     7		"io"
     8		"os"
     9		"syscall"
    10	
    11		"github.com/xianxu/pair/cmd/internal/dispatcher"
    12		"github.com/xianxu/pair/cmd/internal/entrypoint"
    13	)
    14	
    15	func main() {
    16		os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
    17	}
    18	
    19	func run(args []string, stdout, stderr io.Writer) int {
    20		return runWithLegacyRuntime(args, stdout, stderr, osLegacyRuntime{})
    21	}
    22	
    23	type legacyRuntime interface {
    24		Executable() (string, error)
    25		Stat(path string) error
    26		Environ() []string
    27		Exec(path string, argv []string, env []string) int
    28	}
    29	
    30	func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
    31		if len(args) > 0 && args[0] == "launch" {
    32			return runLegacyLaunch(args[1:], stderr, rt)
    33		}
    34		res := dispatcher.Dispatch(args)
    35		return writeResult(res, stdout, stderr)
    36	}
    37	
    38	func runLegacyLaunch(args []string, stderr io.Writer, rt legacyRuntime) int {
    39		exe, err := rt.Executable()
    40		if err != nil {
    41			_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
    42			return 1
    43		}
    44		req := entrypoint.ResolveLegacyLaunch(exe, args)
    45		if err := rt.Stat(req.Path); err != nil {
    46			_, _ = fmt.Fprintf(stderr, "pair-go launch: pair launcher not found at %s (%v); run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", req.Path, err)
    47			return 1
    48		}
    49		return rt.Exec(req.Path, req.Argv, rt.Environ())
    50	}
    51	
    52	type osLegacyRuntime struct{}
    53	
    54	func (osLegacyRuntime) Executable() (string, error) {
    55		return os.Executable()
    56	}
    57	
    58	func (osLegacyRuntime) Stat(path string) error {
    59		info, err := os.Stat(path)
    60		if err != nil {
    61			return err
    62		}
    63		if info.IsDir() {
    64			return fmt.Errorf("is a directory")
    65		}
    66		return nil
    67	}
    68	
    69	func (osLegacyRuntime) Environ() []string {
    70		return os.Environ()
    71	}
    72	
    73	func (osLegacyRuntime) Exec(path string, argv []string, env []string) int {
    74		if err := syscall.Exec(path, argv, env); err != nil {
    75			_, _ = fmt.Fprintf(os.Stderr, "pair-go launch: exec %s failed: %v\n", path, err)
    76			return 1
    77		}
    78		return 0
    79	}
    80	
    81	func writeResult(res dispatcher.Result, stdout, stderr io.Writer) int {
    82		if res.Stdout != "" {
    83			_, _ = io.WriteString(stdout, res.Stdout)
    84		}
    85		if res.Stderr != "" {
    86			_, _ = io.WriteString(stderr, res.Stderr)
    87		}
    88		return res.ExitCode
    89	}

exec
/bin/zsh -lc "nl -ba cmd/internal/entrypoint/launch.go && nl -ba cmd/internal/dispatcher/dispatcher.go | sed -n '1,150p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package entrypoint
     2	
     3	import "path/filepath"
     4	
     5	// LegacyLaunchRequest describes the current compatibility handoff from
     6	// pair-go launch to the shell-owned pair launcher.
     7	type LegacyLaunchRequest struct {
     8		Path string
     9		Argv []string
    10	}
    11	
    12	func ResolveLegacyLaunch(executable string, launchArgs []string) LegacyLaunchRequest {
    13		argv := make([]string, 0, len(launchArgs)+1)
    14		argv = append(argv, "pair")
    15		argv = append(argv, launchArgs...)
    16		return LegacyLaunchRequest{
    17			Path: filepath.Join(filepath.Dir(executable), "pair"),
    18			Argv: argv,
    19		}
    20	}
     1	package dispatcher
     2	
     3	import (
     4		"bytes"
     5		"fmt"
     6		"strings"
     7	
     8		"github.com/xianxu/pair/cmd/internal/contextcmd"
     9		"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
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
    31			{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "handoff"},
    32			{Name: "context", Summary: "agent pane context meter", Status: "implemented"},
    33			{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "implemented"},
    34			{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
    35			{Name: "slug", Summary: "session orientation slug generation", Status: "planned"},
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
    53				Stdout:   "pair-go dispatcher skeleton\nlaunch handoff: bin/pair\n",
    54				ExitCode: 0,
    55			}
    56		case "launch":
    57			return launchHandoffResult()
    58		case "context":
    59			return dispatchContext(args[1:])
    60		case "scrollback-render":
    61			return dispatchScrollbackRender(args[1:])
    62		}
    63	
    64		if family, ok := familyByName(args[0]); ok {
    65			return Result{
    66				Stderr:   fmt.Sprintf("%s: %s is planned but not implemented in this skeleton; run %s help\n", programName, family.Name, programName),
    67				ExitCode: 2,
    68			}
    69		}
    70	
    71		return Result{
    72			Stderr:   fmt.Sprintf("%s: unknown command %q; run %s help\n", programName, args[0], programName),
    73			ExitCode: 2,
    74		}
    75	}
    76	
    77	func dispatchContext(args []string) Result {
    78		var stdout bytes.Buffer
    79		code := contextcmd.Run(args, contextcmd.EnvFromOS(), &stdout)
    80		return Result{Stdout: stdout.String(), ExitCode: code}
    81	}
    82	
    83	func dispatchScrollbackRender(args []string) Result {
    84		var stdout, stderr bytes.Buffer
    85		code := scrollbackcmd.Run(args, &stdout, &stderr)
    86		return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}
    87	}
    88	
    89	func launchHandoffResult() Result {
    90		return Result{
    91			Stderr:   "pair-go launch is a process handoff implemented by cmd/pair-go; call pair-go launch ... instead of dispatcher.Dispatch\n",
    92			ExitCode: 2,
    93		}
    94	}
    95	
    96	// Help renders the development-only dispatcher usage text.
    97	func Help(program string) string {
    98		var b strings.Builder
    99		fmt.Fprintf(&b, "Usage: %s <command> [args]\n\n", program)
   100		b.WriteString("Development dispatcher skeleton. Public sessions still start through bin/pair.\n\n")
   101		b.WriteString("Implemented commands:\n")
   102		for _, family := range Families() {
   103			if family.Status == "prototype" {
   104				fmt.Fprintf(&b, "  %-17s %s (prototype; decision-phase only)\n", family.Name, family.Summary)
   105			} else if family.Status == "handoff" {
   106				fmt.Fprintf(&b, "  %-17s %s (compatibility handoff to bin/pair)\n", family.Name, family.Summary)
   107			} else if family.Status == "implemented" {
   108				fmt.Fprintf(&b, "  %-17s %s (implemented helper route)\n", family.Name, family.Summary)
   109			}
   110		}
   111		b.WriteString("\nPlanned command families (not implemented in this skeleton):\n")
   112		for _, family := range Families() {
   113			if family.Status == "planned" {
   114				fmt.Fprintf(&b, "  %-17s %s (%s; not implemented in this skeleton)\n", family.Name, family.Summary, family.Status)
   115			}
   116		}
   117		b.WriteString("\nSupported skeleton commands:\n")
   118		b.WriteString("  help              show this help\n")
   119		b.WriteString("  version           show dispatcher skeleton metadata\n")
   120		return b.String()
   121	}
   122	
   123	func familyByName(name string) (CommandFamily, bool) {
   124		for _, family := range Families() {
   125			if family.Name == name {
   126				return family, true
   127			}
   128		}
   129		return CommandFamily{}, false
   130	}

exec
/bin/zsh -lc "nl -ba cmd/pair-go/main_test.go | sed -n '1,220p' && nl -ba cmd/internal/entrypoint/launch_test.go" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package main
     2	
     3	import (
     4		"bytes"
     5		"errors"
     6		"os"
     7		"reflect"
     8		"strings"
     9		"testing"
    10	)
    11	
    12	func TestRunWritesStdoutAndReturnsDispatcherCode(t *testing.T) {
    13		var stdout, stderr bytes.Buffer
    14		code := run([]string{"help"}, &stdout, &stderr)
    15		if code != 0 {
    16			t.Fatalf("code = %d, want 0", code)
    17		}
    18		if !strings.Contains(stdout.String(), "Usage: pair-go <command> [args]") {
    19			t.Fatalf("stdout missing usage:\n%s", stdout.String())
    20		}
    21		if stderr.String() != "" {
    22			t.Fatalf("stderr = %q, want empty", stderr.String())
    23		}
    24	}
    25	
    26	func TestRunWritesStderrAndReturnsDispatcherCode(t *testing.T) {
    27		var stdout, stderr bytes.Buffer
    28		code := run([]string{"wrap"}, &stdout, &stderr)
    29		if code != 2 {
    30			t.Fatalf("code = %d, want 2", code)
    31		}
    32		if stdout.String() != "" {
    33			t.Fatalf("stdout = %q, want empty", stdout.String())
    34		}
    35		if !strings.Contains(stderr.String(), "wrap is planned but not implemented") {
    36			t.Fatalf("stderr missing unsupported-command message:\n%s", stderr.String())
    37		}
    38	}
    39	
    40	func TestRunLaunchHelp(t *testing.T) {
    41		rt := &fakeLegacyRuntime{
    42			executable: "/repo/bin/pair-go",
    43		}
    44		var stdout, stderr bytes.Buffer
    45		code := runWithLegacyRuntime([]string{"launch", "--help"}, &stdout, &stderr, rt)
    46		if code != 0 {
    47			t.Fatalf("code = %d, want 0", code)
    48		}
    49		if stdout.String() != "" {
    50			t.Fatalf("stdout = %q, want empty", stdout.String())
    51		}
    52		if stderr.String() != "" {
    53			t.Fatalf("stderr = %q, want empty", stderr.String())
    54		}
    55		if rt.execPath != "/repo/bin/pair" {
    56			t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
    57		}
    58		wantArgv := []string{"pair", "--help"}
    59		if !reflect.DeepEqual(rt.execArgv, wantArgv) {
    60			t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
    61		}
    62	}
    63	
    64	func TestRunLaunchExecsLegacyPairWithArgvAndEnv(t *testing.T) {
    65		t.Setenv("PAIR_TEST_ENV", "kept")
    66		rt := &fakeLegacyRuntime{
    67			executable: "/repo/bin/pair-go",
    68			execCode:   42,
    69		}
    70	
    71		var stdout, stderr bytes.Buffer
    72		code := runWithLegacyRuntime([]string{"launch", "claude", "--", "--resume"}, &stdout, &stderr, rt)
    73	
    74		if code != 42 {
    75			t.Fatalf("code = %d, want 42", code)
    76		}
    77		if stdout.String() != "" {
    78			t.Fatalf("stdout = %q, want empty", stdout.String())
    79		}
    80		if stderr.String() != "" {
    81			t.Fatalf("stderr = %q, want empty", stderr.String())
    82		}
    83		if rt.execPath != "/repo/bin/pair" {
    84			t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
    85		}
    86		wantArgv := []string{"pair", "claude", "--", "--resume"}
    87		if !reflect.DeepEqual(rt.execArgv, wantArgv) {
    88			t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
    89		}
    90		if !containsEnv(rt.execEnv, "PAIR_TEST_ENV=kept") {
    91			t.Fatalf("execEnv missing PAIR_TEST_ENV=kept: %#v", rt.execEnv)
    92		}
    93	}
    94	
    95	func TestRunLaunchReportsMissingLegacyPair(t *testing.T) {
    96		rt := &fakeLegacyRuntime{
    97			executable: "/repo/bin/pair-go",
    98			statErr:    os.ErrNotExist,
    99		}
   100	
   101		var stdout, stderr bytes.Buffer
   102		code := runWithLegacyRuntime([]string{"launch", "claude"}, &stdout, &stderr, rt)
   103	
   104		if code != 1 {
   105			t.Fatalf("code = %d, want 1", code)
   106		}
   107		if stdout.String() != "" {
   108			t.Fatalf("stdout = %q, want empty", stdout.String())
   109		}
   110		for _, want := range []string{"pair-go launch", "/repo/bin/pair", "make build", "make install", "dev-aliases.sh"} {
   111			if !strings.Contains(stderr.String(), want) {
   112				t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
   113			}
   114		}
   115		if rt.execPath != "" {
   116			t.Fatalf("execPath = %q, want empty", rt.execPath)
   117		}
   118	}
   119	
   120	type fakeLegacyRuntime struct {
   121		executable string
   122		statErr    error
   123		execCode   int
   124	
   125		execPath string
   126		execArgv []string
   127		execEnv  []string
   128	}
   129	
   130	func (f *fakeLegacyRuntime) Executable() (string, error) {
   131		if f.executable == "" {
   132			return "", errors.New("missing executable")
   133		}
   134		return f.executable, nil
   135	}
   136	
   137	func (f *fakeLegacyRuntime) Stat(_ string) error {
   138		return f.statErr
   139	}
   140	
   141	func (f *fakeLegacyRuntime) Environ() []string {
   142		return os.Environ()
   143	}
   144	
   145	func (f *fakeLegacyRuntime) Exec(path string, argv []string, env []string) int {
   146		f.execPath = path
   147		f.execArgv = append([]string(nil), argv...)
   148		f.execEnv = append([]string(nil), env...)
   149		return f.execCode
   150	}
   151	
   152	func containsEnv(env []string, want string) bool {
   153		for _, got := range env {
   154			if got == want {
   155				return true
   156			}
   157		}
   158		return false
   159	}
     1	package entrypoint
     2	
     3	import (
     4		"reflect"
     5		"testing"
     6	)
     7	
     8	func TestResolveLegacyLaunchDropsLaunchVerb(t *testing.T) {
     9		req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"claude", "--", "--resume"})
    10		if req.Path != "/repo/bin/pair" {
    11			t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
    12		}
    13		want := []string{"pair", "claude", "--", "--resume"}
    14		if !reflect.DeepEqual(req.Argv, want) {
    15			t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
    16		}
    17	}
    18	
    19	func TestResolveLegacyLaunchPreservesSubcommands(t *testing.T) {
    20		req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"resume", "demo"})
    21		if req.Path != "/repo/bin/pair" {
    22			t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
    23		}
    24		want := []string{"pair", "resume", "demo"}
    25		if !reflect.DeepEqual(req.Argv, want) {
    26			t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
    27		}
    28	}

exec
/bin/zsh -lc 'git diff --stat b829ef00346738e3c3d7e40db38c2c03f51e615b..HEAD && git diff --name-only b829ef00346738e3c3d7e40db38c2c03f51e615b..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 Makefile.local                                 |   7 +-
 README.md                                      |   7 ++
 atlas/architecture.md                          |  24 +++---
 atlas/go-migration-inventory.md                |   6 +-
 cmd/internal/dispatcher/dispatcher.go          |  97 ++-------------------
 cmd/internal/dispatcher/dispatcher_test.go     |  57 ++-----------
 cmd/internal/entrypoint/launch.go              |  20 +++++
 cmd/internal/entrypoint/launch_test.go         |  28 ++++++
 cmd/pair-go/launch_process_test.go             |  71 ---------------
 cmd/pair-go/main.go                            |  65 ++++++++++++--
 cmd/pair-go/main_test.go                       | 114 ++++++++++++++++++++-----
 workshop/issues/000077-go-entrypoint-switch.md |  37 +++++---
 12 files changed, 270 insertions(+), 263 deletions(-)
Makefile.local
README.md
atlas/architecture.md
atlas/go-migration-inventory.md
cmd/internal/dispatcher/dispatcher.go
cmd/internal/dispatcher/dispatcher_test.go
cmd/internal/entrypoint/launch.go
cmd/internal/entrypoint/launch_test.go
cmd/pair-go/launch_process_test.go
cmd/pair-go/main.go
cmd/pair-go/main_test.go
workshop/issues/000077-go-entrypoint-switch.md

codex
The core handoff is small: `cmd/pair-go` special-cases `launch`, computes a sibling `pair`, and `exec`s it; non-launch helper routes stay in the dispatcher. I’m checking whether that matches how this repo actually installs and invokes binaries, because that’s the main place this kind of entrypoint change can break despite green unit tests.
exec
/bin/zsh -lc "nl -ba Makefile.local | sed -n '210,270p' && nl -ba Makefile | sed -n '1,220p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
   210	# pair-scrollback-render — replays a pair-wrap raw PTY capture through
   211	# charmbracelet/x/vt and emits one ANSI-styled line per logical scrollback
   212	# row, for the Alt+/ viewer to display through nvim.
   213	$(BIN_DIR)/pair-scrollback-render: cmd/pair-scrollback-render/main.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
   214		go build -o $@ ./cmd/pair-scrollback-render
   215	
   216	# pair-wrap — transparent PTY proxy around a TUI coding agent: forwards
   217	# stdio, tees scrollback log, extracts colored spans for nvim autocomplete,
   218	# detects OSC/BEL and forwards as OSC 9 to the recorded outer-TTY, handles
   219	# SIGUSR1 image-paste capture window.
   220	$(BIN_DIR)/pair-wrap: cmd/pair-wrap/main.go go.mod
   221		go build -o $@ ./cmd/pair-wrap
   222	
   223	$(BIN_DIR)/pair-slug: cmd/pair-slug/main.go cmd/pair-slug/slug.go cmd/internal/model/model.go cmd/internal/transcript/transcript.go go.mod
   224		go build -o $@ ./cmd/pair-slug
   225	
   226	# pair-context — one-shot: print a pane's current context size (token count)
   227	# for the pair-title poller. Pure ctxmeter reader + shared transcript resolver.
   228	$(BIN_DIR)/pair-context: cmd/pair-context/main.go cmd/internal/contextcmd/contextcmd.go cmd/internal/ctxmeter/ctxmeter.go cmd/internal/transcript/transcript.go go.mod
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
   242	# pair-go — development dispatcher and Go-owned launch handoff.
   243	# `pair-go launch ...` execs sibling bin/pair for the current migration window;
   244	# helper routes stay in the dispatcher.
   245	$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/entrypoint/launch.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
   246		go build -o $@ ./cmd/pair-go
   247	
   248	
   249	# Tart VM targets (tart, tart-gui, tart-mount, tart-stop, tart-clean)
   250	# live in .tart/Makefile, vendored from ariadne and auto-included by
   251	# Makefile.workflow. `make help-tart` for usage.
     1	# Canonical repo name from git remote (portable across worktrees and containers)
     2	REPO_NAME := $(shell git remote get-url origin 2>/dev/null | sed 's|.*/||; s|\.git$$||')
     3	
     4	# This project nests issues and history under workshop/
     5	WF_ISSUES_DIR = workshop/issues
     6	WF_HISTORY_DIR = workshop/history
     7	
     8	# Assemble sub-Makefiles (Makefile.workflow already includes .openshell/Makefile)
     9	include Makefile.workflow
    10	-include Makefile.local
    11	
    12	.PHONY: help
    13	
    14	# help-sandbox, help-tart, and help-colima are defined by .openshell/Makefile,
    15	# .tart/Makefile, and .colima/Makefile respectively, all included via
    16	# Makefile.workflow's -include lines. Every consumer that vendors the ariadne
    17	# base layer ships all three fragments (see construct/base.manifest), so these
    18	# targets always resolve. If a consumer ever drops .openshell, .tart, or
    19	# .colima from its manifest, the corresponding help-X line would need to come
    20	# out. Transient window: a consumer that pulls this updated Makefile before
    21	# running setup.sh to materialize the new .colima/Makefile symlink will get a
    22	# "No rule to make target 'help-colima'" until setup runs — same accepted
    23	# fragility the other two already carry.
    24	help: help-workflow help-sandbox help-tart help-colima
    25		@true

exec
/bin/zsh -lc 'rg -n "pair-go|pair-dev|install|BIN_DIR|dev-aliases|PAIR_DEV|cmd/pair-go" Makefile Makefile.local bin cmd README.md atlas workshop/issues/000077-go-entrypoint-switch.md' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
README.md:141:Automatically installed with `homebrew`.
README.md:171:# install
README.md:172:brew tap xianxu/pair && brew install pair
README.md:178:That installs `zellij`, `neovim`, `fzf`, `jq`, and `par` if they aren't already present. The agent (`claude`, `codex`, `agy`) you install separately. Then:
README.md:203:`pair-go launch ...` is the Go-owned migration entrypoint for testing the
README.md:206:window. In a dev shell sourced from `../ariadne/construct/dev-aliases.sh`,
README.md:207:`pair-go` rebuilds from `cmd/pair-go` automatically before running; no
README.md:208:`pair-go-dev` command is needed.
README.md:214:**Hacking on pair?** Use `pair-dev` instead of `pair` — same arguments, but it rebuilds pair's Go binaries from source (`make build`) on launch *and* on every Alt+n / Shift+Alt+N restart, so the zellij-spawned `pair-wrap` always matches your working tree. (Deployed installs run `pair`, which uses the prebuilt binary and needs no Go toolchain.)
Makefile.local:4:.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
Makefile.local:12:# `make install` pick it up automatically.
Makefile.local:15:#   make install    → also copy each into ~/.local/bin/
Makefile.local:21:# in both dev (source tree) and brew-install layouts.
Makefile.local:23:# Why ~/.local/bin/ for install: XDG convention, on PATH per the user's
Makefile.local:27:BIN_DIR     := bin
Makefile.local:29:GO_BINS     := pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-go
Makefile.local:31:build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
Makefile.local:33:install: build
Makefile.local:36:	    install -m 755 $(BIN_DIR)/$$b $(INSTALL_DIR)/$$b && \
Makefile.local:37:	    echo "installed: $(INSTALL_DIR)/$$b"; \
Makefile.local:41:pair-scribe:            $(BIN_DIR)/pair-scribe
Makefile.local:42:pair-scrollback-render: $(BIN_DIR)/pair-scrollback-render
Makefile.local:43:pair-wrap:              $(BIN_DIR)/pair-wrap
Makefile.local:44:pair-slug:              $(BIN_DIR)/pair-slug
Makefile.local:45:pair-continuation:      $(BIN_DIR)/pair-continuation
Makefile.local:46:pair-changelog:         $(BIN_DIR)/pair-changelog
Makefile.local:47:pair-context:           $(BIN_DIR)/pair-context
Makefile.local:48:pair-go:                $(BIN_DIR)/pair-go
Makefile.local:178:# pair-dev rebuild hook (#000046): deployed mode (PAIR_DEV unset) must invoke
Makefile.local:199:test-changelog: $(BIN_DIR)/pair-changelog $(BIN_DIR)/pair-scrollback-render
Makefile.local:207:$(BIN_DIR)/pair-scribe: cmd/pair-scribe/main.go go.mod
Makefile.local:213:$(BIN_DIR)/pair-scrollback-render: cmd/pair-scrollback-render/main.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
Makefile.local:220:$(BIN_DIR)/pair-wrap: cmd/pair-wrap/main.go go.mod
Makefile.local:223:$(BIN_DIR)/pair-slug: cmd/pair-slug/main.go cmd/pair-slug/slug.go cmd/internal/model/model.go cmd/internal/transcript/transcript.go go.mod
Makefile.local:228:$(BIN_DIR)/pair-context: cmd/pair-context/main.go cmd/internal/contextcmd/contextcmd.go cmd/internal/ctxmeter/ctxmeter.go cmd/internal/transcript/transcript.go go.mod
Makefile.local:233:$(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill.go cmd/pair-changelog/prompt.go cmd/internal/model/model.go go.mod
Makefile.local:239:$(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-continuation/git.go cmd/pair-continuation/main.go go.mod
Makefile.local:242:# pair-go — development dispatcher and Go-owned launch handoff.
Makefile.local:243:# `pair-go launch ...` execs sibling bin/pair for the current migration window;
Makefile.local:245:$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/entrypoint/launch.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
Makefile.local:246:	go build -o $@ ./cmd/pair-go
workshop/issues/000077-go-entrypoint-switch.md:17:At some point the public `pair` command must become Go-owned. The next safe step is to make the Go-owned `pair-go launch ...` path exercise the real launcher contract while leaving the existing `pair` and `pair-dev` entrypoints stable.
workshop/issues/000077-go-entrypoint-switch.md:21:Make `pair-go launch ...` a meaningful Go entrypoint by having it hand off to the existing `bin/pair` launcher with `pair`-compatible arguments. `pair-go launch claude` should behave like `pair claude`; `pair-go launch resume <tag>`, `pair-go launch continue ...`, `pair-go launch list`, and `pair-go launch rename ...` should all pass through to the same shell-owned implementation for this migration window.
workshop/issues/000077-go-entrypoint-switch.md:23:Keep `pair` and `pair-dev` working exactly as they do today. The Go command has no separate `-dev` variant: a developer shell sourced from `../ariadne/construct/dev-aliases.sh` already discovers `cmd/pair-go`, rebuilds `bin/pair-go` on every invocation, and then runs it from the caller's cwd. If `pair-go launch` cannot find the launcher beside the built binary, its diagnostic should point to `make build` / `make install` and the dev-alias path instead of failing with a bare exec error.
workshop/issues/000077-go-entrypoint-switch.md:29:- [x] `pair-go launch ...` uses Go process code first and then invokes the real launcher with `pair`-compatible argv.
workshop/issues/000077-go-entrypoint-switch.md:31:- [x] `pair-dev` still rebuilds and launches the working tree behavior.
workshop/issues/000077-go-entrypoint-switch.md:33:- [x] The dev workflow is documented: `cmd/pair-go` is rebuilt by `../ariadne/construct/dev-aliases.sh`; no `pair-go-dev` command is needed.
workshop/issues/000077-go-entrypoint-switch.md:39:- [x] Add tests for `pair-go launch` argv/env handoff to `bin/pair`.
workshop/issues/000077-go-entrypoint-switch.md:42:- [x] Verify `pair`, `pair-dev`, and `pair-go launch` behavior with process fakes and targeted builds.
workshop/issues/000077-go-entrypoint-switch.md:68:- 2026-06-30: closed — go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1; go test ./... -count=1; make pair-go; make test-dev-rebuild; bin/pair-go help; bin/pair-go launch --help; bin/pair --help; bin/pair-dev --help
workshop/issues/000077-go-entrypoint-switch.md:70:Re-scoped after operator guidance: keep `cmd/pair-go` as the Go entrypoint under test, leave `pair` / `pair-dev` stable, and rely on `../ariadne/construct/dev-aliases.sh` to rebuild `cmd/pair-go` in developer shells. `ARCH-DRY`: reuse the existing launcher for real zellij behavior instead of duplicating shell-owned lifecycle paths in Go. `ARCH-PURE`: keep launch path selection testable with a pure path/argv decision plus a thin exec boundary.
workshop/issues/000077-go-entrypoint-switch.md:72:Plan-quality gate returned FAILURE because the plan promised argv/env handoff but did not explicitly test env propagation, and because `pair-dev --help` under-proved the dev rebuild acceptance criterion. Updated the durable plan to require an inherited-env fake-runner assertion and `make test-dev-rebuild` verification. `ARCH-PURPOSE`: compatibility claims must be pinned by tests, not implied by the shell fallback.
workshop/issues/000077-go-entrypoint-switch.md:74:Implemented `cmd/internal/entrypoint` and the `cmd/pair-go` launch handoff. `pair-go launch ...` now resolves sibling `bin/pair`, validates it, and execs it with preserved argv/env; missing launcher diagnostics point to `make build`, `make install`, and `../ariadne/construct/dev-aliases.sh`. Removed the stale decision-phase `pair-go launch` dispatcher surface while keeping `context` and `scrollback-render` helper routes intact. Updated README and atlas migration notes.
workshop/issues/000077-go-entrypoint-switch.md:78:- `go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1`
workshop/issues/000077-go-entrypoint-switch.md:80:- `make pair-go`
workshop/issues/000077-go-entrypoint-switch.md:82:- `bin/pair-go help`
workshop/issues/000077-go-entrypoint-switch.md:83:- `bin/pair-go launch --help`
workshop/issues/000077-go-entrypoint-switch.md:85:- `bin/pair-dev --help`
bin/pair:104:  `pair-dev` is identical to `pair` but rebuilds pair's Go binaries from
bin/pair:107:  when hacking on cmd/*; deployed installs keep running `pair` (prebuilt,
bin/pair:108:  no toolchain dependency). See bin/pair-dev + bin/lib/dev-rebuild.sh.
bin/pair:376:        # offline housekeeping from a machine that never had it installed)
bin/pair:933:    echo "      brew install zellij  (or your package manager equivalent)" >&2
bin/pair:1707:    echo "      install it first, then re-run." >&2
bin/pair:2238:# Dev mode (pair-dev exports PAIR_DEV): rebuild the Go binaries from source so
bin/pair:2241:# with no toolchain dependency unless PAIR_DEV is set. This is the create path
bin/pair:2242:# only, so a plain attach skips it; restart re-execs $0=bin/pair with PAIR_DEV
cmd/pair-wrap/main.go:980:				// `make nous-install` into the unusable autocomplete
cmd/pair-wrap/main.go:981:				// candidate `makenous-install`. Drop in a single-space
atlas/architecture.md:17:bin/pair-go                  # opt-in Go dispatcher + launch handoff; public launcher remains bin/pair
atlas/architecture.md:46:installed commands forever.
atlas/architecture.md:48:As of #77, `bin/pair-go` is the Go-owned launch entrypoint under test:
atlas/architecture.md:49:`pair-go launch ...` resolves its own executable, finds sibling `bin/pair`, and
atlas/architecture.md:51:makes `pair-go launch claude`, `pair-go launch resume <tag>`, `pair-go launch
atlas/architecture.md:52:continue ...`, `pair-go launch list`, and `pair-go launch rename ...`
atlas/architecture.md:54:yet. A developer shell sourced from `../ariadne/construct/dev-aliases.sh`
atlas/architecture.md:55:rebuilds `cmd/pair-go` automatically; no `pair-go-dev` command exists.
atlas/architecture.md:63:`pair-go context <tag> <agent>` and `pair-go scrollback-render ...`. Both routes
atlas/architecture.md:72:may embed those assets or install them adjacent to the binary, but the migration
atlas/architecture.md:76:merge-safe: after any sub-issue lands, the public `pair` command, `pair-dev`,
atlas/architecture.md:569:   - **Legacy snapshot-diff (fallback).** Used when a fresh pidfile doesn't appear within 2s (`PAIR_SESSION_WATCH_PID_WAIT_SECONDS` in tests) — i.e., when the installed pair-wrap binary predates #000020 and doesn't publish the pidfile, or a stale pidfile is never refreshed. Behaves identically to pre-#000020: snapshots the watch dir at start, picks the first new file. Cross-tag races re-emerge in this path, so the proper resolution is to rebuild pair-wrap.
atlas/architecture.md:677:## Binary freshness: deployed vs dev (`pair-dev`)
atlas/architecture.md:679:The Go binaries (`pair-wrap`, `pair-slug`, …) live in `$PAIR_HOME/bin` (first on PATH per *Path resolution* above) and, after `make install`, in `~/.local/bin`. `bin/` is **gitignored** — built on demand, absent in a fresh tree. Because the agent pane launches as `sh -c '… exec pair-wrap …'`, the wrapper is resolved by a **PATH lookup**: no shell function or `.zshenv` can intercept it (`exec` bypasses functions, and `sh` ≠ zsh), so `construct/dev-aliases.sh`'s rebuild-on-call freshness does **not** reach it. When `$PAIR_HOME/bin/pair-wrap` is stale or absent, PATH silently falls through to an old `~/.local/bin` copy and the running wrapper drifts from source — the failure mode is *silence*, not an error (diagnosed once via the #000045 flight recorder going quiet for every Go-emitted aspect while only nvim's Lua emitter still logged).
atlas/architecture.md:682:- **Deployed** — `pair`. Runs whatever prebuilt binary PATH finds; zero toolchain dependency. Keep `~/.local/bin` current with `make install`.
atlas/architecture.md:683:- **Dev** — `pair-dev` (#000046). Exports `PAIR_DEV=1` and execs `pair`; `bin/pair` then runs `make build` (via `bin/lib/dev-rebuild.sh`'s `dev_rebuild`) on the **create path**, before the layout execs pair-wrap, so `$PAIR_HOME/bin` holds a fresh build. Restart-safe: `PAIR_DEV` survives `handle_restart_marker`'s `exec "$0"`, so Alt+n / Shift+Alt+N rebuild too; a plain attach (no new wrapper spawned) correctly skips it. Deployed launches (`PAIR_DEV` unset) invoke no toolchain.
atlas/architecture.md:685:`pair-doctor` *diagnoses* the same staleness `pair-dev` prevents: its emitter-health probe (`doctor/emitter-health.sh`, #000047) greps the *running* `pair-wrap`/`pair-slug` (resolved via the `pair-wrap-pid-<tag>` pidfile, else PATH) for its adapt signal strings and flags `[STALE]` when a binary has no logging code — turning the silent-emitter failure into a named finding.
atlas/architecture.md:689:`cmd/pair-scribe` is a `script(1)` replacement that lives in the pair repo for build-system convenience but is not part of pair's runtime — it's user shell tooling, typically wired at the top of `~/.zshrc` to swap for `script -q -F`. The user's preexec/precmd hooks send `SIGUSR1`/`SIGUSR2` to pause/resume the on-disk typescript around commands whose output (e.g. TUI redraws) shouldn't be captured, enabling a clean "capture last command output" flow that pair can read back from `$_ZSH_SCRIPT_LOG`. Lives at `~/.local/bin/pair-scribe` after `make install`. Full design notes and the zshrc snippet: `cmd/pair-scribe/README.md`.
atlas/architecture.md:695:- **Self-contained.** Uses `--config-dir` and `nvim -u` to fully isolate from the user's normal configs. No invasive install.
atlas/go-migration-inventory.md:14:- installed or runtime-called artifacts under `bin/`, `bin/lib/`, `cmd/`,
atlas/go-migration-inventory.md:16:- `Makefile`, `Makefile.local`, install/packaging docs, and process-level
atlas/go-migration-inventory.md:42:- **P1**: reduces installed binary/script surface or stateful shell risk.
atlas/go-migration-inventory.md:50:| `bin/pair` / `cmd/internal/launcher` / `cmd/internal/entrypoint` | Bash public launcher plus Go-owned launch handoff | user shell, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` (#77) resolves sibling `bin/pair` from the `pair-go` executable and execs it with `pair`-compatible argv/env. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves the compatibility handoff; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Go handoff entrypoint with `bin/pair` retained as stable public launcher and compat target through #77; real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
atlas/go-migration-inventory.md:51:| `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling `pair`. | Resolves symlinks; depends on `bin/pair` and `bin/lib/dev-rebuild.sh`. | compat-shim, likely stays as developer wrapper until Go entrypoint has dev mode | P1 |
atlas/go-migration-inventory.md:52:| `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
atlas/go-migration-inventory.md:57:| `bin/pair-context` / `cmd/pair-context` / `cmd/internal/contextcmd` | Go binary plus shared runner | `bin/pair-title.sh`; development-only `pair-go context` | `pair-context <tag> <agent>` and `pair-go context <tag> <agent>` print the same humanized token count or nothing; tolerant exit 0 on failure. | Reads `PAIR_DATA_DIR`, `pane-<tag>-<agent>.json`, config, native transcripts. | implemented helper route in `pair-go context`; legacy binary retained while title poller calls it | P1 |
atlas/go-migration-inventory.md:58:| `bin/pair-scrollback-render` / `cmd/pair-scrollback-render` / `cmd/internal/scrollbackcmd` | Go binary plus shared runner | `bin/pair-scrollback-open`, `bin/pair-changelog-open`, `nvim/scrollback.lua` refresh; development-only `pair-go scrollback-render` | `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out` and `pair-go scrollback-render ...`; nonzero on render/write failure. | Reads `.raw` and `.events.jsonl`; atomically writes `.ansi` or cleaned text. | implemented helper route in `pair-go scrollback-render`; legacy binary retained for shell/Lua callers | P0 |
atlas/go-migration-inventory.md:61:| `bin/pair-scribe` / `cmd/pair-scribe` | Go binary | user shell rc outside Pair sessions | `pair-scribe -log PATH -- CMD [ARGS...]`; long-running PTY wrapper; SIGUSR1 pauses log, SIGUSR2 resumes. | Writes typescript log; wraps child PTY; independent of `PAIR_*`. | go-subcommand candidate is low value; may remain separate installed helper or become `pair scribe` with alias | P2 |
atlas/go-migration-inventory.md:89:| `Makefile` | build/workflow entry | developer/CI/SDLC | Includes workflow and local makefiles; `help` target. | Uses git remote; includes vendored base fragments. | packaging metadata; keep, update in #79 if install layout changes | P1 |
atlas/go-migration-inventory.md:90:| `Makefile.local` | build/install/test metadata | developer/CI/`pair-dev` | Builds and installs Go binaries, runs test matrix. | Writes `bin/` and `~/.local/bin`; invokes Go, nvim, shell tests. | build contract; #74/#76/#79 must update as dispatcher changes | P0 |
atlas/go-migration-inventory.md:91:| `README.md` | install/usage docs | users/package consumers | Homebrew install, CLI usage, keybindings, dev mode. | Documents dependencies and public commands. | docs to update at #77/#79 | P1 |
atlas/go-migration-inventory.md:92:| `cmd/pair-scribe/README.md` | helper docs | users configuring shell logging | Documents `pair-scribe` install/usage. | No runtime behavior. | docs; update if `pair scribe` route added | P3 |
atlas/go-migration-inventory.md:115:Build/install callers:
atlas/go-migration-inventory.md:118:- `make install` copies those binaries to `~/.local/bin`.
atlas/go-migration-inventory.md:119:- `pair-dev` relies on `make build`, then zellij's PATH lookup resolves fresh
atlas/go-migration-inventory.md:128:- #76 established the first helper-dispatch pattern with `pair-go context` and
atlas/go-migration-inventory.md:129:  `pair-go scrollback-render`, backed by shared internal runners while legacy
atlas/go-migration-inventory.md:132:- #77 made `pair-go launch ...` a meaningful Go-owned compatibility handoff to
atlas/go-migration-inventory.md:137:- #79 owns whether `nvim/` and `zellij/` are embedded or installed adjacent.
atlas/go-migration-inventory.md:161:- `bin/pair-dev`
atlas/go-migration-inventory.md:208:- `cmd/pair-go/helper_equivalence_test.go`
atlas/go-migration-inventory.md:209:- `cmd/pair-go/main.go`
atlas/go-migration-inventory.md:280:timeout wrapper. They are migration evidence, not installed artifacts.
atlas/index.md:16:- `README.md` (repo root) — install and usage.
cmd/internal/dispatcher/dispatcher.go:12:const programName = "pair-go"
cmd/internal/dispatcher/dispatcher.go:53:			Stdout:   "pair-go dispatcher skeleton\nlaunch handoff: bin/pair\n",
cmd/internal/dispatcher/dispatcher.go:91:		Stderr:   "pair-go launch is a process handoff implemented by cmd/pair-go; call pair-go launch ... instead of dispatcher.Dispatch\n",
cmd/pair-wrap/update_agent_output_test.go:87:	// non-SGR escape match, "make nous-install" merges into the
cmd/pair-wrap/update_agent_output_test.go:88:	// unusable autocomplete candidate "makenous-install".
cmd/pair-wrap/update_agent_output_test.go:91:	// "make" and "nous-install" are spaces in the agent pane; the
cmd/pair-wrap/update_agent_output_test.go:93:	p.updateAgentOutput([]byte(sgr("31") + "make" + "\x1b[1C" + "nous-install" + sgr("0")))
cmd/pair-wrap/update_agent_output_test.go:95:	if len(got) != 1 || got[0] != "make nous-install" {
cmd/pair-wrap/update_agent_output_test.go:96:		t.Errorf("got %q, want [\"make nous-install\"]", got)
cmd/internal/dispatcher/dispatcher_test.go:21:				"Usage: pair-go <command> [args]",
cmd/internal/dispatcher/dispatcher_test.go:61:	for _, want := range []string{"pair-go", "dispatcher skeleton", "launch handoff: bin/pair"} {
cmd/internal/dispatcher/dispatcher_test.go:76:	for _, want := range []string{"wrap", "planned", "not implemented", "pair-go help"} {
cmd/internal/dispatcher/dispatcher_test.go:91:	for _, want := range []string{"pair-go launch", "process handoff", "cmd/pair-go"} {
cmd/internal/dispatcher/dispatcher_test.go:136:	for _, want := range []string{"unknown command", "frobnicate", "pair-go help"} {
cmd/pair-scribe/README.md:26:    make install
cmd/internal/entrypoint/launch_test.go:9:	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"claude", "--", "--resume"})
cmd/internal/entrypoint/launch_test.go:20:	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"resume", "demo"})
cmd/pair-go/main_test.go:18:	if !strings.Contains(stdout.String(), "Usage: pair-go <command> [args]") {
cmd/pair-go/main_test.go:42:		executable: "/repo/bin/pair-go",
cmd/pair-go/main_test.go:67:		executable: "/repo/bin/pair-go",
cmd/pair-go/main_test.go:97:		executable: "/repo/bin/pair-go",
cmd/pair-go/main_test.go:110:	for _, want := range []string{"pair-go launch", "/repo/bin/pair", "make build", "make install", "dev-aliases.sh"} {
bin/lib/dev-rebuild.sh:3:# `pair-dev` exports PAIR_DEV=1 then execs `pair`; bin/pair sources this and
bin/lib/dev-rebuild.sh:7:# PATH lookup that neither .zshenv nor construct/dev-aliases.sh's rebuild
bin/lib/dev-rebuild.sh:11:# Restart-safe: Alt+n / Shift+Alt+N re-exec $0=bin/pair, and PAIR_DEV rides
bin/lib/dev-rebuild.sh:15:# Deployed launches leave PAIR_DEV unset → no-op → zero toolchain dependency.
bin/lib/dev-rebuild.sh:17:# Usage:  PAIR_HOME=<repo> dev_rebuild      (no-op unless PAIR_DEV is set)
bin/lib/dev-rebuild.sh:25:    [ -n "${PAIR_DEV:-}" ] || return 0
bin/lib/dev-rebuild.sh:27:        echo "pair-dev: 'make' not on PATH — launching with existing binaries." >&2
bin/lib/dev-rebuild.sh:30:    echo "pair-dev: rebuilding Go binaries (make build in $PAIR_HOME) …" >&2
bin/lib/dev-rebuild.sh:32:        || echo "pair-dev: build FAILED — launching with last-good binaries (fix, then Alt+n)." >&2
cmd/internal/entrypoint/launch.go:6:// pair-go launch to the shell-owned pair launcher.
cmd/pair-go/main.go:1:// pair-go is the development dispatcher for the future primary Go CLI. Its
cmd/pair-go/main.go:41:		_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
cmd/pair-go/main.go:46:		_, _ = fmt.Fprintf(stderr, "pair-go launch: pair launcher not found at %s (%v); run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", req.Path, err)
cmd/pair-go/main.go:75:		_, _ = fmt.Fprintf(os.Stderr, "pair-go launch: exec %s failed: %v\n", path, err)
bin/pair-session-watch.sh:107:# pre-#000020 behavior so old pair-wrap installs still capture sessions
cmd/pair-go/helper_equivalence_test.go:15:	pairGo := filepath.Join(bin, "pair-go")
cmd/pair-go/helper_equivalence_test.go:25:		t.Fatalf("pair-go context mismatch\nlegacy:   code=%d stdout=%q stderr=%q\ndispatch: code=%d stdout=%q stderr=%q",
bin/pair-dev:2:# pair-dev — dev-mode launcher: identical to `pair`, but rebuilds pair's Go
bin/pair-dev:10:# pair-dev exports PAIR_DEV=1 and hands off to `pair`, which (and only which)
bin/pair-dev:11:# owns the gated `make build` — see bin/lib/dev-rebuild.sh. PAIR_DEV survives
bin/pair-dev:17:#   pair-dev            pair-dev codex      pair-dev -- --resume      etc.
bin/pair-dev:21:# or other PATH symlink to pair-dev still finds its sibling `pair` in the
bin/pair-dev:34:export PAIR_DEV=1
bin/pair-scrollback-open:13:#   $PAIR_HOME      — pair install root (for nvim/scrollback.lua)
cmd/internal/launcher/args.go:5:// LaunchArgs is the pure parse result for the guarded pair-go launch prototype.
cmd/internal/launcher/args.go:22:// ParseArgs parses pair-go launch args. It intentionally supports only the
cmd/internal/launcher/args.go:34:		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %s is not implemented by pair-go launch; use bin/pair", argv[0])}
cmd/internal/launcher/args.go:37:			return LaunchArgs{}, UsageError{Message: "pair-go launch: 'resume' requires a tag"}
cmd/internal/launcher/args.go:41:			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: invalid tag: %v", err)}
cmd/internal/launcher/args.go:44:			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", argv[2])}
cmd/internal/launcher/args.go:64:		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", arg)}
cmd/internal/launcher/args_test.go:83:			if !strings.Contains(err.Error(), "not implemented by pair-go launch") {
cmd/internal/launcher/zellij_test.go:16:  "list-sessions --short") printf 'pair-live\npair-detached\npair-gone\nother\n' ;;
cmd/internal/launcher/zellij_test.go:17:  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-gone [Created] (EXITED - attach to resurrect)\n' ;;
cmd/internal/launcher/zellij_test.go:33:		{Name: "pair-gone", State: SessionExited},

exec
/bin/zsh -lc 'ls -l bin/pair bin/pair-dev bin/pair-go 2>/dev/null || true && file bin/pair bin/pair-dev bin/pair-go 2>/dev/null || true' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
-rwxr-xr-x@ 1 xianxu  staff   107426 Jun 26 09:55 bin/pair
-rwxr-xr-x@ 1 xianxu  staff     1720 Jun 16 23:38 bin/pair-dev
-rwxr-xr-x@ 1 xianxu  staff  4626754 Jun 30 15:27 bin/pair-go
bin/pair:     Bourne-Again shell script text executable, Unicode text, UTF-8 text
bin/pair-dev: Bourne-Again shell script text executable, Unicode text, UTF-8 text
bin/pair-go:  Mach-O 64-bit executable arm64

exec
/bin/zsh -lc "nl -ba bin/pair | sed -n '1,260p' && nl -ba bin/pair-dev | sed -n '1,120p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	#!/usr/bin/env bash
     2	# pair — launch a zellij session split between a TUI coding agent (top pane)
     3	# and an nvim drafting pane (bottom). See README.md.
     4	#
     5	# Usage:
     6	#   pair                       claude in default session
     7	#   pair <agent>               claude / codex / agy
     8	#   pair [<agent>] -- <args>   forward args to agent on create,
     9	#                              e.g. pair claude -- --resume
    10	#                                   pair -- --dangerously-skip-permissions
    11	#                                   pair codex -- -p "say hi"
    12	#   pair -h, --help            show help
    13	#
    14	# When detached pair-* sessions exist, an fzf picker is shown so the user
    15	# explicitly chooses what to do — no silent auto-attach. The picker shows
    16	# ALL detached pair-* sessions regardless of agent argument; the agent
    17	# argument only matters for the create path (sentinel label, default name,
    18	# binary to exec).
    19	
    20	set -euo pipefail
    21	
    22	# Wrap zellij with a hard timeout for read-only IPC. The daemon socket can
    23	# wedge if an earlier client got stuck (seen 2026-05-06: a `list-sessions`
    24	# from hours prior held the socket and every new pair launch piled up behind
    25	# it). pair makes several of these per launch, so one wedge cascades into
    26	# unkillable bash trees. 5s is generous — healthy responses are <100ms; if
    27	# it doesn't return by then, something's wrong and falling through is better
    28	# than hanging. Long-running zellij invocations (attach, new-session) are
    29	# intentionally NOT wrapped — they're meant to block.
    30	zj() {
    31	    local secs=5
    32	    if command -v gtimeout >/dev/null 2>&1; then
    33	        gtimeout "$secs" zellij "$@"
    34	    elif command -v timeout >/dev/null 2>&1; then
    35	        timeout "$secs" zellij "$@"
    36	    else
    37	        # Pure-bash fallback for systems without coreutils' timeout (default
    38	        # macOS). Run zellij in the background, arm a watchdog, wait.
    39	        #
    40	        # The watchdog's stdio is redirected to /dev/null. Otherwise, in a
    41	        # pipeline like `zj list-sessions | awk ...`, the watchdog subshell
    42	        # inherits stdout = pipe-to-awk; when we SIGTERM it after zellij
    43	        # exits, bash dies but the orphan `sleep` child keeps the pipe FD
    44	        # open, so awk waits for EOF until the full $secs elapses. With N
    45	        # piped zj calls in pair's startup, that compounds (we saw N×5s).
    46	        zellij "$@" &
    47	        local pid=$!
    48	        ( sleep "$secs"; kill -9 "$pid" 2>/dev/null ) </dev/null >/dev/null 2>&1 &
    49	        local watchdog=$!
    50	        local rc=0
    51	        wait "$pid" 2>/dev/null || rc=$?
    52	        kill "$watchdog" 2>/dev/null || true
    53	        wait "$watchdog" 2>/dev/null || true
    54	        return $rc
    55	    fi
    56	}
    57	
    58	# normalize_tag <raw> <errprefix> — accept a bare tag or `pair-<tag>`, strip the
    59	# prefix, and validate the charset (letters/digits/dash/underscore). Echoes the
    60	# cleaned tag on stdout; on an empty/invalid value prints "<errprefix> '<raw>'
    61	# (allowed: …)" to stderr and returns 1. Callers: t="$(normalize_tag "$x" "…")" || exit 1.
    62	# Shared by resume/continue/rename + the name prompt (#52 ARCH-DRY). It *returns*
    63	# (never exits) so it composes inside $( ), where an exit would only kill the subshell.
    64	normalize_tag() {
    65	    local _raw="$1" _t="$1"
    66	    case "$_t" in pair-*) _t="${_t#pair-}" ;; esac
    67	    case "$_t" in
    68	        ""|*[!A-Za-z0-9_-]*)
    69	            printf "%s '%s' (allowed: letters, digits, dash, underscore)\n" "$2" "$_raw" >&2
    70	            return 1 ;;
    71	    esac
    72	    printf '%s' "$_t"
    73	}
    74	
    75	# --- help -------------------------------------------------------------------
    76	case "${1:-}" in
    77	    -h|--help|help)
    78	        cat <<'EOF'
    79	pair — Neovim-backed input field for any TUI coding agent.
    80	
    81	USAGE
    82	  pair                          claude in default session
    83	  pair <agent>                  e.g. pair codex / pair agy
    84	  pair resume <tag>             attach pair-<tag> if it exists, else create
    85	                                with that tag and the saved config; agent
    86	                                is inferred from saved state
    87	  pair continue [slug]          resume from a continuation doc; bare lists them
    88	  pair [<agent>] -- <args>      forward args to the agent on create
    89	                                e.g. pair claude -- --resume
    90	                                     pair -- --dangerously-skip-permissions
    91	                                     pair codex -- -p "say hi"
    92	  pair list                     list pair-* sessions and their attach state
    93	  pair rename <old> <new>       rename every tag-scoped file in
    94	                                $PAIR_DATA_DIR from <old> to <new>.
    95	                                Refuses if `pair-<old>` is still tracked
    96	                                by zellij (use Ctrl+Alt+n's (R)ename
    97	                                option for the in-session equivalent).
    98	  pair -h, --help               this message
    99	
   100	  Use `--` to separate pair's args from the agent's args. Without `--`,
   101	  pair only accepts `<agent>` as a positional (or `resume <tag>`).
   102	
   103	  DEV MODE
   104	  `pair-dev` is identical to `pair` but rebuilds pair's Go binaries from
   105	  source (`make build`) on launch AND on every Alt+n / Shift+Alt+N restart,
   106	  so the zellij-spawned pair-wrap always matches the working tree. Use it
   107	  when hacking on cmd/*; deployed installs keep running `pair` (prebuilt,
   108	  no toolchain dependency). See bin/pair-dev + bin/lib/dev-rebuild.sh.
   109	
   110	  When creating a new session you'll be prompted for a name; type any
   111	  custom name (e.g. "bugfix") to make it independent of other sessions.
   112	  `resume <tag>` skips that prompt — use it to restart a saved tag with
   113	  its captured args (and the agent's session if still on disk).
   114	
   115	  Agent args (after `--`) only apply on CREATE. Reattaching to an
   116	  existing session uses whatever args the agent was originally started
   117	  with — the picker doesn't re-launch the agent.
   118	
   119	BEHAVIOR
   120	  Picker shows ALL detached pair-* sessions, regardless of the agent
   121	  argument. The agent argument only controls what gets created if you
   122	  pick the "+ new" sentinel: it labels the sentinel, drives the auto-
   123	  suggested name, and is the binary that gets exec'd.
   124	
   125	    0 detached + 0 historical → create directly (prompts for name;
   126	                                default is auto-generated like
   127	                                pair-claude-2)
   128	    ≥1 detached or ≥1 historical → fzf picker over: detached pair-*
   129	                                sessions, then historical tags from
   130	                                this cwd within $PAIR_HISTORY_DAYS
   131	                                (default 14) annotated `(Nd ago, no
   132	                                live session)`, then a `+ new <agent>
   133	                                session` sentinel. Detached row →
   134	                                attach. Historical row → create-by-name
   135	                                (reuses saved draft / agent config).
   136	                                Sentinel → fall through to create.
   137	
   138	  Historical surface (#24) uses the cwd basename as prefix: tags named
   139	  `<cwd-base>` or `<cwd-base>-<subproject>` whose draft/log sidecars
   140	  were touched within the window. PAIR_DEBUG_HISTORY=1 prints the scan
   141	  results and exits without launching zellij.
   142	
   143	  No silent attach. Every reattach is explicit.
   144	
   145	KEYBINDINGS (inside a pair session)
   146	  Alt+Return    send nvim buffer to agent + Enter, log, clear, insert mode
   147	  Alt+Shift+Return
   148	                append nvim buffer to agent composer + newline, no submit;
   149	                log + clear (like Alt+Return but unsubmitted)
   150	  Alt+c         open/show/hide the review collaboration pane; if no review
   151	                target exists, starts :PairReview
   152	  Ctrl+C        send ESC (0x1b) to agent pane — interrupts claude's
   153	                in-flight stream without leaving the draft
   154	  Alt+← / Alt+→ walk through prompt history (-N) ↔ draft (*) ↔ queue (+N)
   155	  Shift+Alt+← / Shift+Alt+→
   156	                jump to next region boundary (oldest-history / newest-history /
   157	                * / front-of-queue / back-of-queue)
   158	  Alt+1..Alt+9  (insert mode, completion popup visible) quick-pick the Nth
   159	                visible completion item. Items show their index (e.g.
   160	                `1 bin/pair-wrap`); past 9, use arrows or <C-n>/<C-p>.
   161	  Alt+q         push current buffer to queue front (+1)
   162	                from *: parks the draft. from -N: forks. from +N: move-to-front
   163	  Alt+BS        (at +N) delete the current queued prompt; items shift down,
   164	                so tap-tap deletes a run
   165	  Shift+Alt+BS  erase history + draft + queue ("start anew"); confirms first
   166	  Alt+↑ / Alt+↓ step nvim pane along minimized ↔ 12 lines ↔ 1/2 ladder
   167	                (works from either pane). minimized = statusline only;
   168	                Alt+↑ from there grows the pane back. Confirm-requiring
   169	                keys (Alt+x/d/n/N) auto-grow out of minimized first so
   170	                the prompt is visible.
   171	  Alt+d         detach from session (re-attach later via `pair`)
   172	  Alt+x         full quit — kill session and remove from resurrect list
   173	                (Ctrl+q is unbound; Alt+x is the only quit path)
   174	  Alt+n         reload pair — kill the session and re-launch in place
   175	                with the same tag, agent, args, AND agent session.
   176	                The conversation resumes; pair itself is the only
   177	                thing that restarts (useful after a config or binary
   178	                rebuild). Ctrl+Alt+n is a bound alias — use it on macOS
   179	                terminal combos where Option+n triggers the dead-tilde
   180	                composer instead of forwarding as Meta+n.
   181	  Shift+Alt+N   restart with a fresh agent conversation — same tag,
   182	                agent, and args, but the saved per-(tag,agent) config
   183	                is dropped so the agent starts a brand-new session
   184	  Alt+h         pop up this help in a floating pane (press `q` to dismiss)
   185	  Alt+i         attach clipboard image to agent + insert whatever marker
   186	                the agent renders (e.g. claude `[Image #N]`). Captured
   187	                from pair-wrap's PTY stream so no per-agent format
   188	                knowledge is baked in. Flashes
   189	                "[no image in clipboard]" at cursor for 1s if the OS
   190	                clipboard has no image data.
   191	  Alt+/         open agent pane scrollback in a read-only ANSI-colored
   192	                nvim viewer; line numbers match zellij's frame indicator,
   193	                so e.g. `:880` jumps to the line you saw scrolled to 880.
   194	                Inside the viewer:
   195	                  • Alt+q drops a 🤖[comment] marker — bare in normal
   196	                    mode (the line is the context) or 🤖<selection>
   197	                    [comment] in visual mode.
   198	                  • Alt+b / Alt+B jump to previous / next prompt
   199	                    boundary, so you can hop between turns instead of
   200	                    scrolling line-by-line.
   201	                  • Search is smart-case: `/foo` matches Foo/FOO/foo,
   202	                    `/Foo` only Foo.
   203	                  • <Esc> exits. If markers (or an overall comment) are
   204	                    pending, a Yes/No confirm runs first; on accept,
   205	                    everything is formatted (`> <quote>\n<comment>`)
   206	                    and appended to the draft for review + send.
   207	
   208	  Status line shape: `Alt: <- history H < pos[*] [hint] > Q queued ->`
   209	  — H/Q are total counts, pos is `*`/`-N`/`+N`. A trailing `*` on `-N`
   210	  means a pending fork on that history slot; navigating away prompts:
   211	  (S)end, (Q)ueue, (D)iscard, [S]tay. The `[hint]` is contextual:
   212	  `[q=queue]` on */-N, `[⌫=del]` on +N.
   213	
   214	DRAFT SYNTAX
   215	  ===  comment  Whole lines starting with `===` are stripped at send time
   216	                but kept in draft/queue/log. Use them to annotate why a
   217	                queued prompt exists. Mid-line `===` is unaffected.
   218	
   219	FILES
   220	  ${XDG_DATA_HOME:-~/.local/share}/pair/draft-<tag>.md     persistent draft (*)
   221	  ${XDG_DATA_HOME:-~/.local/share}/pair/log-<tag>.md       prompt history (-N)
   222	  ${XDG_DATA_HOME:-~/.local/share}/pair/queue-<tag>/*.md   queued prompts (+N)
   223	
   224	See README.md for image-paste recipes and other details.
   225	EOF
   226	        exit 0
   227	        ;;
   228	    list|ls)
   229	        if ! command -v zellij >/dev/null 2>&1; then
   230	            echo "pair: zellij not found on PATH." >&2
   231	            exit 1
   232	        fi
   233	        # Snapshot session state once. --no-formatting carries the EXITED
   234	        # annotation; --short gives clean newline-separated names.
   235	        raw=$( { zj list-sessions --no-formatting 2>/dev/null || true; } )
   236	        sessions=$( { zj list-sessions --short 2>/dev/null || true; } \
   237	                    | awk '/^pair-/' \
   238	                    | sort )
   239	        if [ -z "$sessions" ]; then
   240	            echo "no pair sessions"
   241	            exit 0
   242	        fi
   243	        list_data_dir="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
   244	        mkdir -p "$list_data_dir"
   245	
   246	        # Backfill agent-<tag> for any running pair-wrap process whose tag
   247	        # we don't yet have on disk — covers sessions created before the
   248	        # agent-tracking feature shipped. Read PAIR_TAG from each process's
   249	        # environ (macOS: `ps -E`; Linux: /proc/<pid>/environ) and the
   250	        # agent name from argv (the token after `pair-wrap`).
   251	        pw_pids=$(pgrep -f 'pair-wrap' 2>/dev/null || true)
   252	        for pid in $pw_pids; do
   253	            argv=$(ps -p "$pid" -o command= 2>/dev/null || true)
   254	            [ -z "$argv" ] && continue
   255	            agent=$(printf '%s\n' "$argv" | awk '
   256	                { for (i=1; i<=NF; i++) if ($i ~ /pair-wrap$/) { print $(i+1); exit } }')
   257	            [ -z "$agent" ] && continue
   258	            if [ -r "/proc/$pid/environ" ]; then
   259	                env_blob=$(tr '\0' '\n' < "/proc/$pid/environ" 2>/dev/null || true)
   260	            else
     1	#!/usr/bin/env bash
     2	# pair-dev — dev-mode launcher: identical to `pair`, but rebuilds pair's Go
     3	# binaries from source on every launch AND every in-session restart, so the
     4	# zellij-spawned pair-wrap always matches the working tree (#000046).
     5	#
     6	# Why a separate entrypoint: the agent pane runs pair-wrap via a PATH lookup
     7	# (`sh -c '... exec pair-wrap ...'`) that no shell function or .zshenv can
     8	# intercept — so a stale, or gitignored-absent, repo/bin binary silently falls
     9	# through to an old ~/.local/bin copy and the wrapper drifts from source.
    10	# pair-dev exports PAIR_DEV=1 and hands off to `pair`, which (and only which)
    11	# owns the gated `make build` — see bin/lib/dev-rebuild.sh. PAIR_DEV survives
    12	# the `exec "$0"` that restart performs, so the rebuild re-fires on Alt+n /
    13	# Shift+Alt+N too. Deployed users keep running `pair` directly, with no
    14	# toolchain dependency.
    15	#
    16	# Usage: exactly `pair`'s — all args forward through.
    17	#   pair-dev            pair-dev codex      pair-dev -- --resume      etc.
    18	set -euo pipefail
    19	
    20	# Resolve this script's real directory (following symlinks) so a ~/.local/bin
    21	# or other PATH symlink to pair-dev still finds its sibling `pair` in the
    22	# source tree rather than next to the symlink.
    23	src="${BASH_SOURCE[0]}"
    24	while [ -L "$src" ]; do
    25	    dir="$(cd -P "$(dirname "$src")" && pwd)"
    26	    src="$(readlink "$src")"
    27	    case "$src" in /*) ;; *) src="$dir/$src" ;; esac
    28	done
    29	here="$(cd -P "$(dirname "$src")" && pwd)"
    30	
    31	# Exported (not a bare assignment) so it survives the exec chain — pair → the
    32	# zellij/sh launch → and crucially the `exec "$0"` that handle_restart_marker
    33	# performs on restart. dev_rebuild keys off exactly this.
    34	export PAIR_DEV=1
    35	exec "$here/pair" "$@"

exec
/bin/zsh -lc "rg -n \"install .*pair|pair-dev|bin/pair|GO_BINS|INSTALL_DIR|\\.local/bin.*pair|homebrew|brew\" -g '"'!workshop/history/**'"' -g '"'!bin/pair'"'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
atlas/how-to-bring-up-a-new-harness-cli.md:59:- **File:** [bin/pair-session-watch.sh](file:///Users/xianxu/workspace/pair/bin/pair-session-watch.sh)
atlas/how-to-bring-up-a-new-harness-cli.md:82:- **File:** [bin/pair](file:///Users/xianxu/workspace/pair/bin/pair)
atlas/how-to-bring-up-a-new-harness-cli.md:83:- Integrate the agent-specific resume argument in `bin/pair`:
atlas/how-to-bring-up-a-new-harness-cli.md:98:**Telemetry Signal** (aspect `3`, see §3): `session-id` from `pair-session-watch.sh` — `fired` when `extract_id` resolves an id and the config is written, **`near-miss`** when a file matching the watch pattern is found but no id can be extracted (filename/format drift), `fail` when the 60s watch window elapses with no id at all (the session file never appeared where expected). The resume mapping in `bin/pair` is the *consumer* of this id; it's static config with no separate signal.
atlas/how-to-bring-up-a-new-harness-cli.md:154:3. [ ] **Implement Session Watching** in `bin/pair-session-watch.sh` (using `lsof` and target file patterns).
atlas/how-to-bring-up-a-new-harness-cli.md:155:4. [ ] **Configure Launcher Recovery** in `bin/pair` (mapping `--conversation` or `--resume` flags).
atlas/how-to-bring-up-a-new-harness-cli.md:172:`bin/pair` truncates the file once at session launch; all components then append
atlas/how-to-bring-up-a-new-harness-cli.md:213:the same trust level as the existing scrollback logs. `bin/pair` removes it on quit.
atlas/review-workbench.md:62:- `readiness.lua` + `bin/pair-review-readiness` — pure/classified git readiness
atlas/review-workbench.md:121:- `bin/pair-review-open <file>` — validates + spawns the **full-screen** floating pane
atlas/go-migration-inventory.md:50:| `bin/pair` / `cmd/internal/launcher` / `cmd/internal/entrypoint` | Bash public launcher plus Go-owned launch handoff | user shell, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` (#77) resolves sibling `bin/pair` from the `pair-go` executable and execs it with `pair`-compatible argv/env. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves the compatibility handoff; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Go handoff entrypoint with `bin/pair` retained as stable public launcher and compat target through #77; real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
atlas/go-migration-inventory.md:51:| `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling `pair`. | Resolves symlinks; depends on `bin/pair` and `bin/lib/dev-rebuild.sh`. | compat-shim, likely stays as developer wrapper until Go entrypoint has dev mode | P1 |
atlas/go-migration-inventory.md:52:| `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
atlas/go-migration-inventory.md:53:| `zellij/layouts/main.kdl` | zellij native asset | `bin/pair` via `zellij --new-session-with-layout` | Defines agent and draft panes; shell expands Pair env at pane start. | Calls `pair-wrap`; calls `nvim -u "$PAIR_HOME/nvim/init.lua"`; writes `pane-<tag>-<agent>.json`; writes draft nvim pid file. | native-asset, packaged adjacent/embedded | P0 |
atlas/go-migration-inventory.md:54:| `zellij/config.kdl` | zellij native asset | zellij session config from `bin/pair` | Global keybinds, copy command, scrollback buffer, pane frames. | Calls `copy-on-select.sh`, `pair-help`, `pair-scrollback-open`, `pair-changelog-open`; routes quit/restart/compact through nvim functions. | native-asset, packaged adjacent/embedded | P0 |
atlas/go-migration-inventory.md:55:| `bin/pair-wrap` / `cmd/pair-wrap` | Go binary | zellij agent pane | `pair-wrap [--scrollback-log PATH] agent [args...]`; transparent PTY proxy; long-running; failure in diagnostics is swallowed. | Reads Pair env and agent command; writes `agent-output-<tag>`, `agent-pid-<tag>`, scrollback `.raw`/`.events.jsonl`, image capture files; may invoke `pair-slug`. | go-subcommand `pair wrap`; keep old binary as compat alias for KDL until caller moves | P0 |
atlas/go-migration-inventory.md:56:| `bin/pair-slug` / `cmd/pair-slug` | Go binary | `pair-wrap` turn-end hook, tests | Env-driven, no stdin; resolves native transcript, proposes slug; exits 0 on most failures. | Requires `PAIR_TAG`, `PAIR_DATA_DIR`; reads config/transcripts/git branch; writes `slug-proposed-<tag>`; optional `PAIR_SLUG_*`, `OPENAI_API_KEY`. | go-subcommand `pair slug`; legacy binary retained during #76 | P1 |
atlas/go-migration-inventory.md:57:| `bin/pair-context` / `cmd/pair-context` / `cmd/internal/contextcmd` | Go binary plus shared runner | `bin/pair-title.sh`; development-only `pair-go context` | `pair-context <tag> <agent>` and `pair-go context <tag> <agent>` print the same humanized token count or nothing; tolerant exit 0 on failure. | Reads `PAIR_DATA_DIR`, `pane-<tag>-<agent>.json`, config, native transcripts. | implemented helper route in `pair-go context`; legacy binary retained while title poller calls it | P1 |
atlas/go-migration-inventory.md:58:| `bin/pair-scrollback-render` / `cmd/pair-scrollback-render` / `cmd/internal/scrollbackcmd` | Go binary plus shared runner | `bin/pair-scrollback-open`, `bin/pair-changelog-open`, `nvim/scrollback.lua` refresh; development-only `pair-go scrollback-render` | `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out` and `pair-go scrollback-render ...`; nonzero on render/write failure. | Reads `.raw` and `.events.jsonl`; atomically writes `.ansi` or cleaned text. | implemented helper route in `pair-go scrollback-render`; legacy binary retained for shell/Lua callers | P0 |
atlas/go-migration-inventory.md:59:| `bin/pair-changelog` / `cmd/pair-changelog` | Go binary | `bin/pair-changelog-open` | `pair-changelog --cleaned F --log F --anchor F [--agent A] [--model M]`; exits nonzero on required read/model/write failure. | Reads cleaned scrollback/log/anchor; calls agent model through internal model runner; atomically writes log and anchor. | go-subcommand `pair changelog`; legacy binary retained for opener | P1 |
atlas/go-migration-inventory.md:60:| `bin/pair-continuation` / `cmd/pair-continuation` | Go binary | nvim compaction prompt instructions, operator/agent shell | `pair-continuation --slug S --agent A --issues CSV --body-file F [--repo-root R ...]`; writes and commits continuation; nonzero on validation/git failure. | Reads body/stdin, git repo state; writes `workshop/continuation/*.md`; runs git commit/push. | go-subcommand `pair continuation`; legacy binary retained for agent instructions until docs change | P1 |
atlas/go-migration-inventory.md:61:| `bin/pair-scribe` / `cmd/pair-scribe` | Go binary | user shell rc outside Pair sessions | `pair-scribe -log PATH -- CMD [ARGS...]`; long-running PTY wrapper; SIGUSR1 pauses log, SIGUSR2 resumes. | Writes typescript log; wraps child PTY; independent of `PAIR_*`. | go-subcommand candidate is low value; may remain separate installed helper or become `pair scribe` with alias | P2 |
atlas/go-migration-inventory.md:66:| `bin/pair-scrollback-open` | POSIX shell orchestrator | zellij Alt+/ Run, nvim Alt+b jump | `pair-scrollback-open [--jump prev|next]`; opens read-only nvim viewer; singleton lock. | Requires `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_HOME`; calls renderer, zellij IPC, nvim; writes `.ansi`, `.viewport`, lock. | shell-glue now; candidate Go orchestration after entrypoint, while `nvim/scrollback.lua` remains native | P1 |
atlas/go-migration-inventory.md:67:| `nvim/scrollback.lua` | Neovim native asset | `bin/pair-scrollback-open` | Loaded by `nvim -u ... <ansi>`; interactive read-only viewer; refreshes backing render. | Reads Pair env and `.ansi`; may call `pair-scrollback-render`; writes pending marker files. | native-asset, adjacent/embedded | P0 |
atlas/go-migration-inventory.md:68:| `bin/pair-changelog-open` | POSIX shell orchestrator | zellij Alt+l Run | Opens changelog viewer and starts detached render/distill singleton. | Requires Pair env; calls renderer, `pair-changelog`, setsid/perl, nvim; reads/writes `changelog-*` sidecars. | shell-glue now; candidate Go orchestration after entrypoint | P1 |
atlas/go-migration-inventory.md:69:| `nvim/changelog.lua` | Neovim native asset | `bin/pair-changelog-open` | Loaded by `nvim -u ... <log>`; read-only watcher/spinner. | Reads `PAIR_CHANGELOG_*` and Pair env. | native-asset, adjacent/embedded | P1 |
atlas/go-migration-inventory.md:70:| `bin/pair-title.sh` | Bash stateful poller | `bin/pair` ensure_title_poller | `pair-title.sh <tag> <agent>`; long-running 60s poller; test hook `PAIR_TITLE_TEST_CALL`. | Reads/writes title pid, pane json, cmux owner files; calls `pair-context`, zellij, ps, cmux. | stateful shell-glue; explicit #78 candidate | P1 |
atlas/go-migration-inventory.md:71:| `bin/pair-session-watch.sh` | Bash stateful watcher | `bin/pair` create path | `pair-session-watch.sh <agent> <tag> <cwd> [agent-args...]`; background 60s watcher; no-op for claude. | Reads agent pidfile, lsof/ps, native session dirs; writes config JSON atomically; logs adapt events. | stateful shell-glue; explicit #78 candidate | P1 |
atlas/go-migration-inventory.md:76:| `bin/pair-notify` | Bash notification helper | agent hooks/manual shell inside Pair | `pair-notify [--osc 9|777] "message"`; writes OSC to outer tty; nonzero on bad args/missing tty. | Requires `PAIR_TAG`; reads `outer-tty-<tag>`. | small shell-glue; possible Go subcommand but low packaging impact | P2 |
atlas/go-migration-inventory.md:77:| `bin/pair-quit.sh` | Bash keybind helper | nvim `PairConfirmQuit` | Touch quit marker then kill zellij session. | Uses `ZELLIJ_SESSION_NAME`, `PAIR_KILL_CMD`; writes cache marker. | small compat shell; can fold into Go/nvim flow after entrypoint | P2 |
atlas/go-migration-inventory.md:78:| `bin/pair-restart.sh` | Bash keybind helper | nvim restart confirmations | Writes restart marker then kill zellij session; supports `--new-session`. | Uses `PAIR_TAG`, `PAIR_AGENT`, `ZELLIJ_SESSION_NAME`, cache marker files. | small compat shell; can fold after entrypoint | P2 |
atlas/go-migration-inventory.md:79:| `bin/pair-help` | Bash helper | zellij Alt+h Run | Displays `pair -h` through `less` with escape-to-quit behavior. | Calls `pair`, `less`. | compat-shim; may become `pair help` behavior | P2 |
atlas/go-migration-inventory.md:83:| `bin/pair-review-open` | POSIX shell review helper | nvim review flow | Validates target and opens floating `nvim -u nvim/review.lua`. | Requires Pair env; calls zellij/nvim. | shell-glue, review workbench can move later if packaging needs it | P2 |
atlas/go-migration-inventory.md:84:| `bin/pair-review-readiness` | POSIX shell review helper | `nvim/init.lua` review readiness | Emits readiness data from git and target helper. | Uses `PAIR_HOME`, `PAIR_REVIEW_TARGET_BIN`, git/jq. | shell-glue; possible later Go helper | P2 |
atlas/go-migration-inventory.md:85:| `bin/pair-review-target` | Bash review helper | review readiness/open/tests | Emits JSON target metadata under data dir. | Requires `PAIR_DATA_DIR`; reads config/pid files/lsof; writes `review-target-<tag>.json`. | shell-glue; possible #78 candidate if review packaging matters | P2 |
atlas/go-migration-inventory.md:90:| `Makefile.local` | build/install/test metadata | developer/CI/`pair-dev` | Builds and installs Go binaries, runs test matrix. | Writes `bin/` and `~/.local/bin`; invokes Go, nvim, shell tests. | build contract; #74/#76/#79 must update as dispatcher changes | P0 |
atlas/go-migration-inventory.md:91:| `README.md` | install/usage docs | users/package consumers | Homebrew install, CLI usage, keybindings, dev mode. | Documents dependencies and public commands. | docs to update at #77/#79 | P1 |
atlas/go-migration-inventory.md:117:- `make build` builds `GO_BINS` into `bin/`.
atlas/go-migration-inventory.md:119:- `pair-dev` relies on `make build`, then zellij's PATH lookup resolves fresh
atlas/go-migration-inventory.md:124:- #74 should add a Go dispatcher without changing `bin/pair`.
atlas/go-migration-inventory.md:133:  `bin/pair`, with argv/env preserved and missing-launcher diagnostics. The
atlas/go-migration-inventory.md:156:- `bin/pair`
atlas/go-migration-inventory.md:157:- `bin/pair-changelog`
atlas/go-migration-inventory.md:158:- `bin/pair-changelog-open`
atlas/go-migration-inventory.md:159:- `bin/pair-context`
atlas/go-migration-inventory.md:160:- `bin/pair-continuation`
atlas/go-migration-inventory.md:161:- `bin/pair-dev`
atlas/go-migration-inventory.md:162:- `bin/pair-help`
atlas/go-migration-inventory.md:163:- `bin/pair-notify`
atlas/go-migration-inventory.md:164:- `bin/pair-quit.sh`
atlas/go-migration-inventory.md:165:- `bin/pair-restart.sh`
atlas/go-migration-inventory.md:166:- `bin/pair-review-open`
atlas/go-migration-inventory.md:167:- `bin/pair-review-readiness`
atlas/go-migration-inventory.md:168:- `bin/pair-review-target`
atlas/go-migration-inventory.md:169:- `bin/pair-scribe`
atlas/go-migration-inventory.md:170:- `bin/pair-scrollback-open`
atlas/go-migration-inventory.md:171:- `bin/pair-scrollback-render`
atlas/go-migration-inventory.md:172:- `bin/pair-session-watch.sh`
atlas/go-migration-inventory.md:173:- `bin/pair-slug`
atlas/go-migration-inventory.md:174:- `bin/pair-title.sh`
atlas/go-migration-inventory.md:175:- `bin/pair-wrap`
README.md:66:| **Alt+1**…**Alt+9** | nvim (insert, popup visible) | Quick-pick the Nth visible completion item. The popup tags its first nine items with indices (e.g. `⌥1 bin/pair-wrap`); past 9, use arrows or `<C-n>` / `<C-p>` |
README.md:141:Automatically installed with `homebrew`.
README.md:168:**Homebrew (recommended).** 
README.md:172:brew tap xianxu/pair && brew install pair
README.md:175:brew update; brew upgrade pair
README.md:214:**Hacking on pair?** Use `pair-dev` instead of `pair` — same arguments, but it rebuilds pair's Go binaries from source (`make build`) on launch *and* on every Alt+n / Shift+Alt+N restart, so the zellij-spawned `pair-wrap` always matches your working tree. (Deployed installs run `pair`, which uses the prebuilt binary and needs no Go toolchain.)
Makefile.local:4:.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
Makefile.local:11:# to GO_BINS, add the per-binary recipe stanza. `make build` and
Makefile.local:14:#   make build      → compile every binary in GO_BINS into ./bin/
Makefile.local:20:# at $PAIR_HOME/bin/pair-scrollback-render to pick up the Go renderer
Makefile.local:21:# in both dev (source tree) and brew-install layouts.
Makefile.local:28:INSTALL_DIR := $(HOME)/.local/bin
Makefile.local:29:GO_BINS     := pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-go
Makefile.local:31:build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
Makefile.local:34:	@mkdir -p $(INSTALL_DIR)
Makefile.local:35:	@for b in $(GO_BINS); do \
Makefile.local:36:	    install -m 755 $(BIN_DIR)/$$b $(INSTALL_DIR)/$$b && \
Makefile.local:37:	    echo "installed: $(INSTALL_DIR)/$$b"; \
Makefile.local:178:# pair-dev rebuild hook (#000046): deployed mode (PAIR_DEV unset) must invoke
Makefile.local:191:# Drives the real bin/pair via its PAIR_DEBUG_ARGS probe. See tests/pair-continue-test.sh.
Makefile.local:195:# changelog — end-to-end smoke for bin/pair-changelog-open (#53 M2): fakes the
Makefile.local:243:# `pair-go launch ...` execs sibling bin/pair for the current migration window;
cmd/pair-go/main_test.go:42:		executable: "/repo/bin/pair-go",
cmd/pair-go/main_test.go:55:	if rt.execPath != "/repo/bin/pair" {
cmd/pair-go/main_test.go:56:		t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
cmd/pair-go/main_test.go:67:		executable: "/repo/bin/pair-go",
cmd/pair-go/main_test.go:83:	if rt.execPath != "/repo/bin/pair" {
cmd/pair-go/main_test.go:84:		t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
cmd/pair-go/main_test.go:97:		executable: "/repo/bin/pair-go",
cmd/pair-go/main_test.go:110:	for _, want := range []string{"pair-go launch", "/repo/bin/pair", "make build", "make install", "dev-aliases.sh"} {
atlas/architecture.md:16:bin/pair                     # entry point (launcher)
atlas/architecture.md:17:bin/pair-go                  # opt-in Go dispatcher + launch handoff; public launcher remains bin/pair
atlas/architecture.md:20:bin/pair-quit.sh             # invoked by Alt+x — marks + kills session
atlas/architecture.md:21:bin/pair-restart.sh          # invoked by Alt+n / Shift+Alt+N — marks (quit + restart) + kills session
atlas/architecture.md:22:bin/pair-session-watch.sh    # captures codex/agy session id at create time (#000016, #000020)
atlas/architecture.md:23:bin/pair-wrap                # PTY proxy: OSC translation + scrollback capture
atlas/architecture.md:24:bin/pair-notify              # hook-driven OSC notifier (e.g. claude Notification)
atlas/architecture.md:25:bin/pair-scrollback-render   # raw PTY capture → ANSI-colored line dump (#000017)
atlas/architecture.md:26:bin/pair-scrollback-open     # Alt+/ orchestrator: render + open viewer
atlas/architecture.md:27:bin/pair-changelog           # TTY → distilled change log (LLM, incremental) (#53)
atlas/architecture.md:28:bin/pair-changelog-open      # Alt+l orchestrator: open viewer; clean+distill run in the background (#53)
atlas/architecture.md:48:As of #77, `bin/pair-go` is the Go-owned launch entrypoint under test:
atlas/architecture.md:49:`pair-go launch ...` resolves its own executable, finds sibling `bin/pair`, and
atlas/architecture.md:60:behavior remain shell-owned through `bin/pair` until later migration issues.
atlas/architecture.md:66:the old command names remain the live integration surface. `bin/pair-title.sh`,
atlas/architecture.md:67:`bin/pair-scrollback-open`, `bin/pair-changelog-open`, and
atlas/architecture.md:76:merge-safe: after any sub-issue lands, the public `pair` command, `pair-dev`,
atlas/architecture.md:81:### `bin/pair` — launcher
atlas/architecture.md:100:**Title poller (`bin/pair-title.sh`) — two surfaces.** A single always-on per-tag 60s background poller, spawned via `ensure_title_poller` on *every* entry (create, attach, restart) so a poller a host sleep/reboot/SIGKILL killed is reliably revived. Single-instance guard is identity-checked (`poller_alive` `ps`-matches the command line for this tag; pidfile `$DATA_DIR/title-pid-<tag>`; not a bare `kill -0`) so a recycled PID can't suppress the respawn. It owns two title surfaces (tested in `tests/pair-title-poller-test.sh`):
atlas/architecture.md:118:**Reload / restart in place (Alt+n, Shift+Alt+N).** A second marker, `~/.cache/pair/restart-<session>`, is written alongside `quit-` by `bin/pair-restart.sh`, carrying the agent name + a `new_session` flag. After cleanup_quit_marker tears the session down, `handle_restart_marker` reads the marker and `exec`s pair on itself with `PAIR_FORCE_TAG=<same-tag>` set in the env (pins the new run to the killed session's tag, skipping both the picker and the name prompt). The flag controls what happens to the saved config:
atlas/architecture.md:123:The picker is bypassed in either flavor — Alt+n's argv carries an explicit resume token, and Shift+Alt+N has no saved config to pick against. A third marker field, `continue=<slug>` (#55, written by the in-session compaction branch in `bin/pair`, not `pair-restart.sh`), rides the `new_session=1` path but re-execs `pair continue <slug> <agent> -- <args>` instead of a plain restart — see "In-session compaction" below.
atlas/architecture.md:144:- `copy_command "copy-on-select.sh"` — on every selection finalize (mouse-up after drag), zellij pipes the selected text to this script. `copy_command` replaces zellij's default OS-clipboard write, so the script does that part too. Resolved by PATH (which `bin/pair` populated).
atlas/architecture.md:155:- `Shift+Alt+N` — routed through nvim to `:lua PairConfirmRestartNewSession()` — Y/N modal then `pair-restart.sh --new-session` (restart with a fresh agent conversation). See "Reload / restart in place" under `bin/pair`.
atlas/architecture.md:156:- `Alt+Shift+C` (`Alt C` / `Ctrl Alt c`) — routed through nvim to `:lua PairConfirmCompact()` — Y/N modal then `send_to_agent(<compaction prompt>)` (#55). Unlike the restart binds it does NOT shell out directly: distilling a continuation needs the agent's judgment, so it asks the agent to write a continuation + run `pair continue <slug>` (context-aware → compaction). See "In-session compaction" under `bin/pair`.
atlas/architecture.md:184:### `bin/pair-quit.sh` — Alt+x handler
atlas/architecture.md:186:Touches the marker file `~/.cache/pair/quit-$ZELLIJ_SESSION_NAME`, then `exec zellij kill-session $ZELLIJ_SESSION_NAME`. The kill terminates the session including the script itself; on the launcher side, `bin/pair` resumes, sees the marker, and runs `delete-session --force` to clean up the resurrect entry.
atlas/architecture.md:190:### Outer-TTY capture and notification routing — `bin/pair-wrap`, `bin/pair-notify`
atlas/architecture.md:196:1. **Outer-TTY capture (in `bin/pair`).** Before invoking zellij, on every attach (both create and reattach branches), pair calls `tty(1)`. The result is the path of pair's controlling TTY — which is precisely the outer PTY (the one allocated by whatever wraps pair: cmux, a terminal emulator, etc.). That path gets written to `$DATA_DIR/outer-tty-<tag>`. Refreshed on every attach because the outer PTY changes across detach/reattach, while pane-shell env stays frozen at zellij session-creation time (env-var approaches would go stale).
atlas/architecture.md:200:   - **`bin/pair-wrap`** (Go, `cmd/pair-wrap`). Transparent PTY proxy. The zellij agent pane runs `pair-wrap $PAIR_AGENT $PAIR_AGENT_ARGS` instead of the agent directly (wired in `zellij/layouts/main.kdl`). The wrapper allocates a fresh PTY for the agent, forwards stdin/stdout transparently with SIGWINCH propagation, and watches the agent's output stream for OSC notifications. On detection it writes OSC 9 directly to the recorded outer-TTY path — bypassing zellij.
atlas/architecture.md:243:   - **`bin/pair-notify`** (bash). Hook-driven helper for richer signals. `pair-notify [--osc 9|777] "msg"` reads the same outer-TTY file and writes the OSC. Intended for Claude Code `Notification`/`Stop` hooks where you want semantic events with custom message text rather than relying on the agent's native OSC stream.
atlas/architecture.md:262:**Replay (`bin/pair-scrollback-render`, Go).** Reads `<raw>` and `<events.jsonl>`, feeds the bytes to a `charmbracelet/x/vt` emulator in a single offset-ordered walk over all events (`feedSegments`): write up to each offset, then `Resize` on a resize event or snapshot `Scrollback().Len()` on a `time` event (#59). The emulator runs the same VT100 interpretation zellij does live (width-based wrap, alternate-screen flips, scroll regions), so its row count matches what the user saw in zellij's indicator. After feeding, the renderer walks the scrolled-out history followed by the visible buffer, and emits one ANSI-decorated line per row to `<out.ansi>`: full-reset SGR + per-row attrs + the row's characters + `\x1b[0m`. With `--with-timestamps` (the change-log path only — never the Alt+/ viewer) the pure `interleaveDateMarkers` then inserts `⟦pair:ts DATE⟧` lines at each day boundary from the time snapshots (#59). Built into `bin/pair-scrollback-render` via `make pair-scrollback-render`; single static binary, no runtime dep. Its raw inputs live in `$PAIR_DATA_DIR` as `scrollback-<tag>-<agent>.{raw,events.jsonl}` (RAW VT bytes, NOT in the repo); `:PairTTYRawPath` / `_G.PairTTYRawPath()` (nvim, #56) prints the current session's live `.raw` path on demand and copies it to the `+` register — useful for grabbing the byte stream mid-session, since an Alt+x quit deletes it unless preserved.
atlas/architecture.md:268:**In-session compaction (#55, `Alt+Shift+C`).** `pair continue <slug>` is *context-aware*: run from a normal shell it fresh-starts (above); run from INSIDE its own live pane it **compacts** — copy-parks the scrollback as a recovery net, writes a restart marker carrying a new `continue=<slug>` field (same tag, `new_session=1`), and kills the session. The outer `bin/pair` then `handle_restart_marker`-re-execs `pair continue <slug> <agent> -- <args>` (now outside zellij → the fresh-start branch), so the session reincarnates under the same tag with a clean conversation seeded from the continuation. Detection is **ancestry-based** (`in_zellij_pane`) plus a `ZELLIJ_SESSION_NAME == pair-<tag>` confirmation — never `$ZELLIJ*` env alone, since cmux propagates those to sibling non-pair panes (a false positive would park+kill the wrong session). The branch sits *before* the `in_zellij_pane` guard (which otherwise rejects any in-pane `pair`), with `DATA_DIR` + `handle_restart_marker` hoisted above the guard so test seams can drive them. Seams (test-only): `PAIR_FORCE_IN_SESSION`, `PAIR_FAKE_IN_ZELLIJ`, `PAIR_KILL_CMD`, `PAIR_TEST_CALL`, `PAIR_REEXEC_CAPTURE`. The trigger is the `Alt+Shift+C` keybind (`Alt C` / `Ctrl Alt c` → `PairConfirmCompact` → an agent-agnostic prompt that **defers to the `continuation` datatype procedure** — flush-first, then write the continuation and run `pair continue <slug>` — rather than enumerating a section skeleton inline, so the prompt can't drift out of sync with the datatype; that drift was the bug pair#61 fixed); the outer process suppresses the Alt+x park nudge whenever a restart marker is pending (a restart isn't a quit).
atlas/architecture.md:282:**Open (`bin/pair-scrollback-open`, POSIX sh).** Validates `PAIR_DATA_DIR` / `PAIR_TAG` / `PAIR_AGENT`, runs the renderer, then *launches* `nvim -u $PAIR_HOME/nvim/scrollback.lua $ANSI` as a child — deliberately **not** `exec`, so the script stays alive as nvim's parent and an `EXIT`/`INT`/`TERM` trap can clear the re-entrancy lock on quit. Errors print and `sleep` briefly so the message is readable before the floating pane self-closes. Bound in `zellij/config.kdl` to `Alt+/` as a 100% × 100% floating pane with `close_on_exit=true` — the user's `:q` in the viewer dismisses the pane and returns to pair's two-pane layout untouched. **Re-entrancy guard:** `Alt+/` is a global zellij bind, so pressing it again while the viewer is already focused fires another `Run` and would stack a second nvim (one `:q`/Esc per layer to unwind). zellij can't conditionally skip a `Run`, so the script self-guards: before launching nvim it writes its own PID to `$PAIR_DATA_DIR/scrollback-<tag>-<agent>.openlock`, and on entry it exits immediately if that lock already holds a *live* PID — the redundant floating pane then self-dismisses via `close_on_exit` and focus falls back to the open viewer. A stale lock (hard kill) carries a dead PID and is reclaimed by the next open's `kill -0` check. The draft pane's `Alt+b` (`--jump prev`) runs the same launcher, so it's covered too.
atlas/architecture.md:445:  `bin/pair-changelog-open` in a floating pane. It opens `nvim -u
atlas/architecture.md:523:  (the opener `bin/pair-changelog-open` and the draft-nvim `.ready` watcher):
atlas/architecture.md:524:  the exported `PAIR_SESSION_ID` (set by `bin/pair` at launch for claude-fresh /
atlas/architecture.md:564:1. **Pre-write at launch (`bin/pair`).** Two paths:
atlas/architecture.md:567:2. **Watcher (`bin/pair-session-watch.sh`, codex/agy only).** Spawned in the background by `bin/pair` on the create path, right before the zellij launch. Two discovery paths:
atlas/architecture.md:579:| claude | `~/.claude/projects/<encoded-cwd>/<id>.jsonl` | filename | `--session-id` pre-injected by `bin/pair` (deterministic) |
atlas/architecture.md:591:**Create-flow prompt (`bin/pair`).** When the create path commits a tag, pair reads `config-<tag>-<agent>.json`. If present, it runs the per-agent stale-id check (claude: `[ -f .../<id>.jsonl ]`; codex: `find ~/.codex/sessions -name "*<id>*"`; agy: check transcript file) and fzf-prompts the user with up to three options:
atlas/architecture.md:633:**Inside-flow choreography.** `nvim/init.lua`'s `pair_confirm_restart_impl` shells out `pair rename --restart-check` after the user enters a new tag, re-prompting on each rejection. On accept it execs `pair-restart.sh --rename-to <new>`. `pair-restart.sh` writes `rename_to=<new>` into the restart marker (`~/.cache/pair/restart-<SESSION>`) alongside the existing `tag`, `agent`, `new_session` fields. `handle_restart_marker` in `bin/pair` runs after `cleanup_quit_marker` (so the zj delete-session has cleared the live-old gate) and if `rename_to` is set, invokes `"$0" rename <old> <new>` — full check. On success, the working tag for the re-exec is swapped to `<new>` (so `config-<new>-<agent>.json`, the just-renamed file, is what gets resumed). On failure, a 2-second visible stderr warning is printed and the restart continues with the original tag — the user is never stranded.
atlas/architecture.md:655:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-<tag>` — single-line file recording which agent binary was launched in the session (`claude`, `codex`, ...). Written once at session create; read by `pair list` to display the agent column, and by `bin/pair`'s tag-restart agent-inference. Removed on full quit. The agent isn't otherwise recoverable post-create — env vars are frozen in pane shells, and custom session names (e.g. `pair-bugfix`) don't carry the agent in the name.
atlas/architecture.md:657:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/config-<tag>-<agent>.json` — saved restart configuration for `(tag, agent)` (issue #000016, #000020). `{ agent, args, session_id }`. For claude, written synchronously by `bin/pair` before zellij launch (`--session-id` is deterministic). For codex/agy, written by `bin/pair-session-watch.sh` once the agent's session file is discovered via lsof. Read by `bin/pair`'s create-flow prompt and by the post-Alt+x hint. Survives Alt+x (unlike `agent-<tag>`, which is cleared) — that's the whole point: it's the bridge between two pair launches against the same tag.
atlas/architecture.md:659:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-pid-<tag>` — child agent PID written by `cmd/pair-wrap` immediately after `pty.Start`, removed on shutdown. Consumed by `bin/pair-session-watch.sh` to scope `lsof` discovery to a specific process tree (issue #000020). Mtime is also used as the agent-start epoch in the watcher's birth-time fallback.
atlas/architecture.md:661:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/nvim-pid-<tag>-{draft,scrollback}` — single-line file containing the pid of an `nvim --embed` server child. Written at VimEnter by `nvim/init.lua` (for the draft pane) and `nvim/scrollback.lua` (for the Alt+/ floating viewer) when `$PAIR_NVIM_PID_FILE` is set; the launch sites (`zellij/layouts/main.kdl` for draft, `bin/pair-scrollback-open` for scrollback) export the env var pointing at a tag-scoped path. Read and removed by `cleanup_quit_marker` on Alt+x to SIGKILL the embed deterministically — without this, the embed sometimes survives zellij's pane teardown and accumulates as a PPID=1 orphan, dragging the host into memory pressure across many quits.
atlas/architecture.md:663:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/pair-wrap-pid-<tag>` — single-line file containing pair-wrap's pid, written at startup by `bin/pair-wrap` if `PAIR_TAG` is set. Read by nvim's Alt+i (`attach_image`) so it can `kill -USR1 <pid>` to arm an image-capture window. Removed by pair-wrap on exit (the `finally` block in `main()`) and by `cleanup_quit_marker` as belt-and-suspenders on Alt+x.
atlas/architecture.md:669:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/adapt-<tag>.jsonl` — the adaptation flight recorder (issue #000045). One JSON line per harness-adaptation trigger (`{ts, comp, agent, aspect, signal, outcome, detail}`), appended concurrently by `cmd/pair-wrap`, `cmd/pair-slug`, `bin/pair-session-watch.sh`, and `nvim/adapt.lua` — all writing one shared schema. Truncated once at session launch by `bin/pair` (so multi-process `O_APPEND` never races) and removed on full quit. Read by `doctor/doctor.sh` to surface integration drift (near-miss/fail signals). See `atlas/how-to-bring-up-a-new-harness-cli.md` §3 for the signal registry.
atlas/architecture.md:675:`bin/pair` prepends `$PAIR_HOME/bin` to `$PATH` before exec'ing zellij. zellij and all its child processes (panes, copy_command, Run actions) inherit the PATH and can resolve `clipboard-to-pane.sh`, `copy-on-select.sh`, `pair-quit.sh` by bare name. This lets the zellij KDL configs reference scripts without `sh -c` env-var quoting hacks.
atlas/architecture.md:677:## Binary freshness: deployed vs dev (`pair-dev`)
atlas/architecture.md:679:The Go binaries (`pair-wrap`, `pair-slug`, …) live in `$PAIR_HOME/bin` (first on PATH per *Path resolution* above) and, after `make install`, in `~/.local/bin`. `bin/` is **gitignored** — built on demand, absent in a fresh tree. Because the agent pane launches as `sh -c '… exec pair-wrap …'`, the wrapper is resolved by a **PATH lookup**: no shell function or `.zshenv` can intercept it (`exec` bypasses functions, and `sh` ≠ zsh), so `construct/dev-aliases.sh`'s rebuild-on-call freshness does **not** reach it. When `$PAIR_HOME/bin/pair-wrap` is stale or absent, PATH silently falls through to an old `~/.local/bin` copy and the running wrapper drifts from source — the failure mode is *silence*, not an error (diagnosed once via the #000045 flight recorder going quiet for every Go-emitted aspect while only nvim's Lua emitter still logged).
atlas/architecture.md:683:- **Dev** — `pair-dev` (#000046). Exports `PAIR_DEV=1` and execs `pair`; `bin/pair` then runs `make build` (via `bin/lib/dev-rebuild.sh`'s `dev_rebuild`) on the **create path**, before the layout execs pair-wrap, so `$PAIR_HOME/bin` holds a fresh build. Restart-safe: `PAIR_DEV` survives `handle_restart_marker`'s `exec "$0"`, so Alt+n / Shift+Alt+N rebuild too; a plain attach (no new wrapper spawned) correctly skips it. Deployed launches (`PAIR_DEV` unset) invoke no toolchain.
atlas/architecture.md:685:`pair-doctor` *diagnoses* the same staleness `pair-dev` prevents: its emitter-health probe (`doctor/emitter-health.sh`, #000047) greps the *running* `pair-wrap`/`pair-slug` (resolved via the `pair-wrap-pid-<tag>` pidfile, else PATH) for its adapt signal strings and flags `[STALE]` when a binary has no logging code — turning the silent-emitter failure into a named finding.
atlas/architecture.md:689:`cmd/pair-scribe` is a `script(1)` replacement that lives in the pair repo for build-system convenience but is not part of pair's runtime — it's user shell tooling, typically wired at the top of `~/.zshrc` to swap for `script -q -F`. The user's preexec/precmd hooks send `SIGUSR1`/`SIGUSR2` to pause/resume the on-disk typescript around commands whose output (e.g. TUI redraws) shouldn't be captured, enabling a clean "capture last command output" flow that pair can read back from `$_ZSH_SCRIPT_LOG`. Lives at `~/.local/bin/pair-scribe` after `make install`. Full design notes and the zshrc snippet: `cmd/pair-scribe/README.md`.
bin/pair-quit.sh:2:# Triggered by Alt+x via zellij keybind. Writes a marker file so bin/pair
bin/pair-quit.sh:8:# bin/pair sees the marker on resume and runs `zellij delete-session --force`
cmd/pair-scribe/README.md:28:Produces `~/.local/bin/pair-scribe` (and the other Go binaries). Static
cmd/pair-scribe/README.md:43:    exec ~/.local/bin/pair-scribe -log "$_ZSH_SCRIPT_LOG" -- /bin/zsh
nvim/changelog.lua:4:-- bin/pair-changelog-open. The distilled counterpart to scrollback.lua, but
nvim/changelog.lua:81:-- The orchestrator (bin/pair-changelog-open) launches render+distill as a nohup'd
cmd/pair-wrap/main.go:3:// Installed at bin/pair-wrap and invoked by zellij/layouts/main.kdl on
cmd/pair-wrap/main.go:983:				// fix in bin/pair-wrap.py (949aeec).
cmd/pair-wrap/main.go:1788:	// Open the always-on adaptation flight recorder. bin/pair truncates the
cmd/pair-wrap/main.go:1930:			// bin/pair's cleanup_quit_marker handles them with the rest
nvim/scrollback.lua:56:-- $PAIR_NVIM_PID_FILE so bin/pair's cleanup_quit_marker can reap it on
nvim/scrollback.lua:286:    bin = vim.env.PAIR_HOME .. '/bin/pair-scrollback-render'
workshop/issues/000079-go-packaging-consolidation.md:19:Consolidate release/install packaging around the primary Go `pair` binary and its required assets.
workshop/issues/000079-go-packaging-consolidation.md:24:- Update Homebrew/install/build docs and `Makefile.local` accordingly.
workshop/issues/000079-go-packaging-consolidation.md:42:- [ ] Update build/install/Homebrew wiring.
zellij/layouts/main.kdl:8:// All four are exported by bin/pair before invoking zellij.
zellij/layouts/main.kdl:32:            // are exported by bin/pair before zellij launches, so the
zellij/layouts/main.kdl:63:            // embed pid here so cleanup_quit_marker in bin/pair can reap
cmd/pair-changelog/main.go:2:// log (issue #53). Invoked on demand by bin/pair-changelog-open (Alt+l): it
workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:185:[Inventoried six artifacts: zellij layout, nvim plugin/config, reverse-direction shell helper, zellij keybind config, launcher, README. Discussed packaging options — self-contained launcher (recommended for v1), modular install with nvim plugin, layered approach. Flagged two design choices: agent-agnostic naming (`pair`/`compose`/`prompt-pane` rather than `claude+`) and where draft files live. Estimated three days of work for shippable v1.]
workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:248:bin/pair                      # launcher
workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:281:[Set issue 000001 status to `working`, ran `make issue-sync`. Created directory structure `bin/`, `nvim/`, `zellij/layouts/`. Wrote all six files: `bin/pair`, `bin/clipboard-to-pane.sh`, `nvim/init.lua`, `zellij/config.kdl`, `zellij/layouts/main.kdl`, `README.md`. Plus `atlas/index.md` and `atlas/architecture.md`. Made scripts executable.]
workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:286:- `bin/pair nonexistent` correctly exits 1 with "agent not found."
workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:311:💬: ~/workspace/pair (main*) » bin/pair
workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:486:💬: now, we also need to support multiple bin/pair being issues, potentially on different agnet (claude, codex, gemini). is this supported today
workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:508:[Updated `bin/pair` to accept `VARIANT="${2:-}"`. PAIR_TAG = agent-variant. Session, draft, log all derive from PAIR_TAG. Layout uses `${PAIR_TAG:-${PAIR_AGENT:-claude}}` for the draft path.]
workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:512:💬: when we ctrl-q quit, can you in bin/pair, to run zellij kill-session on that session? or, you should extend the bin/pair to ask your to select which existing sessions to connect to, in an TUI type of thing.
workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:596:💬: ok, when I created a new session with bin/pair, the title of my terminal changed to the slug name used, e.g. pair-claude-5. however, when reattached to pair-claude-5, that didn't happen, the title of the terminal stayed as bin/pair. why?
workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:636:💬: add help text with bin/pair --help
workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:644:💬: bin/pair didn't show the session I created with name blogging, I think it's intentional from you, can you explain how this worked
workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:657:💬: i see, the auto picking behavior is not that useful, because in the bin/pair world, claude sessions are long lived. and auto attach to some long running session that user don't know, is not a good mental model. we can remove auto attach behavior, and always show the picker.
workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:659:now with this change, it seems basically everything would go to bin/pair pick, right? if that's the case we can just remove pick as a subcommand, as it is the default and only thing that makes sense,
workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:669:💬: bin/pair only shows the names pair-claude* still. in the new world, you should show all. right?
workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:708:bin/pair only shows the names pair-claude* still. in the new world, you should show all. right?
workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:714:[Confirmed file still had old code. Rewrote bin/pair from scratch using Write tool, then verified with grep. This time it stuck:]
bin/pair-session-watch.sh:9:# Spawned in the background by bin/pair right before zellij launch on the
bin/pair-session-watch.sh:16:#   claude  — no-op. bin/pair pre-injects `--session-id <uuid>` and writes
bin/pair-session-watch.sh:50:# Aspect 3 flight recorder (atlas §3). PAIR_TAG is normally exported by bin/pair;
bin/pair-session-watch.sh:273:        # the canonical store. Same shape as bin/pair's stripping; keep
zellij/config.kdl:39:// bin/pair prepends $PAIR_HOME/bin to PATH so the script is resolvable by
workshop/issues/000077-go-entrypoint-switch.md:17:At some point the public `pair` command must become Go-owned. The next safe step is to make the Go-owned `pair-go launch ...` path exercise the real launcher contract while leaving the existing `pair` and `pair-dev` entrypoints stable.
workshop/issues/000077-go-entrypoint-switch.md:21:Make `pair-go launch ...` a meaningful Go entrypoint by having it hand off to the existing `bin/pair` launcher with `pair`-compatible arguments. `pair-go launch claude` should behave like `pair claude`; `pair-go launch resume <tag>`, `pair-go launch continue ...`, `pair-go launch list`, and `pair-go launch rename ...` should all pass through to the same shell-owned implementation for this migration window.
workshop/issues/000077-go-entrypoint-switch.md:23:Keep `pair` and `pair-dev` working exactly as they do today. The Go command has no separate `-dev` variant: a developer shell sourced from `../ariadne/construct/dev-aliases.sh` already discovers `cmd/pair-go`, rebuilds `bin/pair-go` on every invocation, and then runs it from the caller's cwd. If `pair-go launch` cannot find the launcher beside the built binary, its diagnostic should point to `make build` / `make install` and the dev-alias path instead of failing with a bare exec error.
workshop/issues/000077-go-entrypoint-switch.md:31:- [x] `pair-dev` still rebuilds and launches the working tree behavior.
workshop/issues/000077-go-entrypoint-switch.md:32:- [x] Existing create, attach, resume, continue, rename/list, quit, and restart flows are preserved through the `bin/pair` fallback.
workshop/issues/000077-go-entrypoint-switch.md:39:- [x] Add tests for `pair-go launch` argv/env handoff to `bin/pair`.
workshop/issues/000077-go-entrypoint-switch.md:42:- [x] Verify `pair`, `pair-dev`, and `pair-go launch` behavior with process fakes and targeted builds.
workshop/issues/000077-go-entrypoint-switch.md:68:- 2026-06-30: closed — go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1; go test ./... -count=1; make pair-go; make test-dev-rebuild; bin/pair-go help; bin/pair-go launch --help; bin/pair --help; bin/pair-dev --help
workshop/issues/000077-go-entrypoint-switch.md:70:Re-scoped after operator guidance: keep `cmd/pair-go` as the Go entrypoint under test, leave `pair` / `pair-dev` stable, and rely on `../ariadne/construct/dev-aliases.sh` to rebuild `cmd/pair-go` in developer shells. `ARCH-DRY`: reuse the existing launcher for real zellij behavior instead of duplicating shell-owned lifecycle paths in Go. `ARCH-PURE`: keep launch path selection testable with a pure path/argv decision plus a thin exec boundary.
workshop/issues/000077-go-entrypoint-switch.md:72:Plan-quality gate returned FAILURE because the plan promised argv/env handoff but did not explicitly test env propagation, and because `pair-dev --help` under-proved the dev rebuild acceptance criterion. Updated the durable plan to require an inherited-env fake-runner assertion and `make test-dev-rebuild` verification. `ARCH-PURPOSE`: compatibility claims must be pinned by tests, not implied by the shell fallback.
workshop/issues/000077-go-entrypoint-switch.md:74:Implemented `cmd/internal/entrypoint` and the `cmd/pair-go` launch handoff. `pair-go launch ...` now resolves sibling `bin/pair`, validates it, and execs it with preserved argv/env; missing launcher diagnostics point to `make build`, `make install`, and `../ariadne/construct/dev-aliases.sh`. Removed the stale decision-phase `pair-go launch` dispatcher surface while keeping `context` and `scrollback-render` helper routes intact. Updated README and atlas migration notes.
workshop/issues/000077-go-entrypoint-switch.md:82:- `bin/pair-go help`
workshop/issues/000077-go-entrypoint-switch.md:83:- `bin/pair-go launch --help`
workshop/issues/000077-go-entrypoint-switch.md:84:- `bin/pair --help`
workshop/issues/000077-go-entrypoint-switch.md:85:- `bin/pair-dev --help`
bin/pair-review-open:9:# Env (exported by bin/pair): $PAIR_DATA_DIR $PAIR_TAG $PAIR_HOME
bin/pair-review-open:40:# nvim/review.lua's VimEnter writes its pid here so bin/pair's cleanup can reap
cmd/pair-changelog/e2e_test.go:59:	// Real render step (as bin/pair-changelog-open invokes it).
bin/pair-notify:8:# the PTY stream for attention markers. bin/pair records its controlling
bin/pair-scrollback-open:9:# Inputs come from pair's env (exported by bin/pair before zellij starts):
bin/pair-scrollback-open:70:if [ ! -x "$PAIR_HOME/bin/pair-scrollback-render" ]; then
bin/pair-scrollback-open:71:    printf 'pair-scrollback-open: bin/pair-scrollback-render not built.\n' >&2
bin/pair-scrollback-open:76:if ! "$PAIR_HOME/bin/pair-scrollback-render" "$RAW" "$EVENTS" "$ANSI"; then
bin/pair-scrollback-open:182:# embed pid here so cleanup_quit_marker in bin/pair can reap it on Alt+x.
cmd/pair-wrap/update_agent_output_test.go:83:	// The 949aeec bug fix in bin/pair-wrap.py, now ported to the Go
bin/pair-review-target:2:# bin/pair-review-target — write the review-target seam (#6) stamped with the
workshop/lessons.md:83:`bin/pair-scrollback-render` initially opened `<out.ansi>` with `'w'` (truncate-then-write). Two `Alt+/` presses in quick succession would race on the same path; whichever finished second left a half-interleaved file for nvim to open. Fixed by writing to `<out.ansi>.tmp` and `os.replace()`-ing at the end.
workshop/lessons.md:89:Two bugs in v1 of `bin/pair` and `zellij/config.kdl` came from going off memory of zellij's API:
workshop/lessons.md:164:via `lsof -p <pid> | awk '$4=="txt"`) was `~/.local/bin/pair-wrap` dated days
workshop/lessons.md:336:   `bin/pair`, but `bin/pair` turned out to be gitignored AND untracked: a normal
workshop/lessons.md:454:`bin/pair` parser what it resolved (`AGENT`, `FORCED_TAG`, forwarded args,
workshop/lessons.md:457:pane, `in_zellij_pane` returned true and `bin/pair` exited with "already running
bin/pair-review-readiness:34:    target_bin="${PAIR_REVIEW_TARGET_BIN:-$HOME_DIR/bin/pair-review-target}"
bin/pair-changelog-open:17:# Inputs from pair's env (exported by bin/pair before zellij starts):
bin/pair-changelog-open:29:# resume reuses it. PAIR_SESSION_ID is exported by bin/pair when the id is known
bin/pair-changelog-open:77:   && [ -x "$PAIR_HOME/bin/pair-scrollback-render" ] && [ -x "$PAIR_HOME/bin/pair-changelog" ]; then
bin/pair-changelog-open:79:    export PCL_RENDER="$PAIR_HOME/bin/pair-scrollback-render"
bin/pair-changelog-open:80:    export PCL_DISTILL="$PAIR_HOME/bin/pair-changelog"
bin/pair-dev:2:# pair-dev — dev-mode launcher: identical to `pair`, but rebuilds pair's Go
bin/pair-dev:10:# pair-dev exports PAIR_DEV=1 and hands off to `pair`, which (and only which)
bin/pair-dev:17:#   pair-dev            pair-dev codex      pair-dev -- --resume      etc.
bin/pair-dev:21:# or other PATH symlink to pair-dev still finds its sibling `pair` in the
tests/pair-rename.sh:17:PAIR_BIN="$SELF_DIR/../bin/pair"
tests/pair-rename.sh:33:# the families enumerated in bin/pair's `rename_paths_for` helper.
bin/pair-restart.sh:3:# current pair session (like Alt+x) and signals bin/pair to re-launch:
bin/pair-restart.sh:9:#                    conversation: bin/pair drops the saved
bin/pair-restart.sh:14:# bin/pair runs `zellij delete-session` as usual) AND a `restart-<session>`
bin/pair-restart.sh:16:# kill-session returns, bin/pair sees the restart marker and execs itself
bin/pair-restart.sh:22:# cleanup_quit_marker deletes that file before bin/pair gets the chance to
bin/pair-title.sh:13:# Spawned in the background by bin/pair on both the create and attach paths.
bin/pair-title.sh:36:# the cmux sidebar are overridden on the next poll — matches bin/pair's
bin/pair-title.sh:59:# matching "211"; the agent arg always follows the tag. Mirrors bin/pair's
bin/pair-title.sh:76:# Abbreviate a raw cwd to ~ on a path boundary (mirrors bin/pair:1154).
bin/pair-title.sh:125:# Test hook (mirrors bin/pair's PAIR_TEST_CALL): invoke a single helper
bin/pair-title.sh:138:# create path in bin/pair, which spawns this poller right BEFORE calling
bin/pair-title.sh:152:# Ignore SIGHUP. bin/pair spawns this with `& disown`, which only
bin/pair-title.sh:245:# Wait for the zellij session to appear (create-path race: bin/pair
bin/pair-title.sh:312:            # Workspace-title ownership (matches bin/pair's cmux_rename_workspace):
bin/pair-title.sh:327:            # Personal display convention (matches bin/pair's cmux_rename_workspace):
cmd/internal/scrollbackcmd/scrollbackcmd.go:3:// Drop-in replacement for the Python+pyte renderer at bin/pair-scrollback-render.
cmd/internal/scrollbackcmd/scrollbackcmd.go:9:// venv that the brew formula has to manage. A static Go binary using
cmd/internal/scrollbackcmd/scrollbackcmd.go:20:// CLI is identical to the Python version so bin/pair-scrollback-open can
tests/cmux-ownership-test.sh:2:# Regression test for bin/pair's cmux_rename_workspace ownership claim.
tests/cmux-ownership-test.sh:11:# Drives the REAL bin/pair through its PAIR_TEST_CALL seam with process-level
tests/cmux-ownership-test.sh:20:PAIR="$ROOT/bin/pair"
nvim/review.lua:2:-- `nvim -u $PAIR_HOME/nvim/review.lua <file>` (from bin/pair-review-open).
nvim/review.lua:539:  -- pid file (reaped by bin/pair's cleanup) + the open-state file.
workshop/continuation/20260619T133818-review-m3.md:19:Read the M3 smoke diagnosis in the session/issue first; the exact bin scripts to rewrite are `bin/pair-review-toggle`, `bin/pair-review-open`, the `Alt r` block in `zellij/config.kdl`, and a new `PairReviewToggle()` in `nvim/init.lua` (model on the `Alt d`/`PairConfirmDetach` pattern). This is live-zellij iteration — implement, then the user re-runs the smoke checklist (in the M3 plan, Task 5). Keep it main-thread (a fork can't verify live zellij).
workshop/continuation/20260619T133818-review-m3.md:51:- Code (branch `000066-agentic-review-workbench`, worktree `/Users/xianxu/workspace/pair`): `nvim/review/{record,reconstruct,apply,projection,markers,mode,docflow,handoff,init}.lua` (the consumer-half), `nvim/review.lua` (pane init), `nvim/pair_poke.lua` (id-based agent poke), `bin/pair-review-{open,toggle}`, `zellij/config.kdl` (the `Alt r` bind). Tests: `nvim/review/*_test.lua` (`make test-lua`) + `tests/review-*-test.sh` (`make test-review`).
cmd/internal/entrypoint/launch_test.go:9:	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"claude", "--", "--resume"})
cmd/internal/entrypoint/launch_test.go:10:	if req.Path != "/repo/bin/pair" {
cmd/internal/entrypoint/launch_test.go:11:		t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
cmd/internal/entrypoint/launch_test.go:20:	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"resume", "demo"})
cmd/internal/entrypoint/launch_test.go:21:	if req.Path != "/repo/bin/pair" {
cmd/internal/entrypoint/launch_test.go:22:		t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
workshop/continuation/20260616T214136-make-test-hang.md:117:  is `pair-dev`. No cross-repo work this session.
workshop/plans/000077-go-entrypoint-switch-plan.md:5:**Goal:** Make `pair-go launch ...` a meaningful Go-owned entrypoint that runs the existing `pair` launcher with compatible arguments while leaving `pair` and `pair-dev` stable.
workshop/plans/000077-go-entrypoint-switch-plan.md:9:**Tech Stack:** Go 1.x, `os.Executable`, `syscall.Exec` or an injected process runner for tests, existing Bash launcher `bin/pair`, existing Makefile build target for `cmd/pair-go`.
workshop/plans/000077-go-entrypoint-switch-plan.md:24:- **Relationships:** 1:1 with a `pair-go launch ...` invocation; owns the resolved launcher path and argv passed to `bin/pair`.
workshop/plans/000077-go-entrypoint-switch-plan.md:31:- **DRY rationale:** Reuses the existing `bin/pair` launcher as the only real session lifecycle owner instead of duplicating create/attach/resume/list/rename behavior in Go.
workshop/plans/000077-go-entrypoint-switch-plan.md:63:	req := entrypoint.ResolveLegacyLaunch("/repo/bin/pair-go", []string{"claude", "--", "--resume"})
workshop/plans/000077-go-entrypoint-switch-plan.md:64:	if req.Path != "/repo/bin/pair" {
workshop/plans/000077-go-entrypoint-switch-plan.md:74:	req := entrypoint.ResolveLegacyLaunch("/repo/bin/pair-go", []string{"resume", "demo"})
workshop/plans/000077-go-entrypoint-switch-plan.md:112:- runner receives sibling `bin/pair`;
workshop/plans/000077-go-entrypoint-switch-plan.md:116:Also add a missing-launcher test that returns a not-found stat result and asserts stderr mentions `pair-go launch`, `bin/pair`, `make build`, `make install`, and `dev-aliases.sh`.
workshop/plans/000077-go-entrypoint-switch-plan.md:160:Expected: `bin/pair-go` builds successfully.
workshop/plans/000077-go-entrypoint-switch-plan.md:180:Update `atlas/architecture.md` and `atlas/go-migration-inventory.md` so they no longer describe `pair-go launch` as decision-phase only. State that #77 makes it a Go-owned compatibility handoff to `bin/pair`, while `bin/pair` remains the public stable entrypoint and the real zellij lifecycle remains shell-owned.
workshop/plans/000077-go-entrypoint-switch-plan.md:209:Expected: PASS and `bin/pair-go` exists.
workshop/plans/000077-go-entrypoint-switch-plan.md:217:bin/pair-go help
workshop/plans/000077-go-entrypoint-switch-plan.md:218:bin/pair-go launch --help
workshop/plans/000077-go-entrypoint-switch-plan.md:219:bin/pair --help
workshop/plans/000077-go-entrypoint-switch-plan.md:220:bin/pair-dev --help
workshop/plans/000077-go-entrypoint-switch-plan.md:223:Expected: `make test-dev-rebuild` passes, proving the existing `PAIR_DEV` rebuild hook still works. Help output succeeds. `bin/pair-go launch --help` should print the existing `pair` help because it hands off to `bin/pair --help`.
bin/lib/dev-rebuild.sh:3:# `pair-dev` exports PAIR_DEV=1 then execs `pair`; bin/pair sources this and
bin/lib/dev-rebuild.sh:11:# Restart-safe: Alt+n / Shift+Alt+N re-exec $0=bin/pair, and PAIR_DEV rides
bin/lib/dev-rebuild.sh:20:# `set -e` (bin/pair:20) — least of all mid-restart, when the old session is
bin/lib/dev-rebuild.sh:27:        echo "pair-dev: 'make' not on PATH — launching with existing binaries." >&2
bin/lib/dev-rebuild.sh:30:    echo "pair-dev: rebuilding Go binaries (make build in $PAIR_HOME) …" >&2
bin/lib/dev-rebuild.sh:32:        || echo "pair-dev: build FAILED — launching with last-good binaries (fix, then Alt+n)." >&2
workshop/continuation/20260612T002626-compact.md:17:2. Record the e2e PASS in `workshop/issues/000055-compact-keybind.md` `## Log` (M2 step 4 done for `claude`; step 5 — repeat once under `pair-dev codex` — is optional, the prompt is agent-agnostic).
workshop/continuation/20260612T002626-compact.md:22:  - **M1** (`bin/pair` mechanics: `park_scrollback` copy|move, ancestry-gated in-session branch, `handle_restart_marker` `continue=` re-exec) — **landed** (9b9e0a1), 21/21 `make test-continue`, boundary review **FIX-THEN-SHIP** (the one Important fixed: park-nudge suppressed when a restart marker is pending).
workshop/continuation/20260612T002626-compact.md:36:- Key files: `bin/pair` — in-session compaction branch (~863–905, placed **before** the `in_zellij_pane` guard), `park_scrollback` (~687), `handle_restart_marker` (~707); `zellij/config.kdl` (`bind "Alt C" "Ctrl Alt c"`); `nvim/init.lua` (`PairConfirmCompact`, ~696–741). Tests: `tests/pair-continue-test.sh` (seams `PAIR_FORCE_IN_SESSION` / `PAIR_FAKE_IN_ZELLIJ` / `PAIR_KILL_CMD`), `make test-continue`.
workshop/continuation/20260612T002626-compact.md:38:- Branch `000055-compact-keybind`, worktree `/Users/xianxu/workspace/pair`. This is a `PAIR_DEV=1` checkout → relaunch verb is `pair-dev`.
CHANGELOG.md:4:tagged in git (`vN.M`) and tracked in the homebrew formula at
CHANGELOG.md:5:[xianxu/homebrew-pair](https://github.com/xianxu/homebrew-pair).
CHANGELOG.md:72:### Homebrew
CHANGELOG.md:75:  runtime is Go-only (`pair-scrollback-render`). `brew install pair` no
CHANGELOG.md:144:- **`pair-dev`** — dev-mode entrypoint that rebuilds the Go binaries from
CHANGELOG.md:147:  the static Go `pair-scrollback-render`. (The brew formula still vendors
CHANGELOG.md:210:  (the primary path shipped via brew) and the Python fallback.
workshop/continuation/20260611T163027-port.md:33:- Key files: `cmd/pair-continuation/` (writer), `cmd/pair-scrollback-render/main.go` (`--plain`/`resolveMax`), `bin/pair` (`continue` verb just after the `resume` block; park-nudge inside `cleanup_quit_marker`).
workshop/continuation/20260615T225403-cont-improve.md:42:- **Key files:** `../ariadne/construct/datatype/continuation.md` — the rewritten datatype, resolved into pair via the `construct/datatype` → `../../ariadne/construct/datatype` **symlink** (so pair gets it for free, no recompose); `nvim/init.lua` `COMPACT_PROMPT` (~2990) — now defers to that datatype, with a comment guarding against re-inlining a skeleton; `bin/pair-continuation` — the writer (enforces only `## NEXT ACTION`; auto-commits+pushes).
workshop/continuation/20260615T225403-cont-improve.md:43:- **Repos:** pair = `/Users/xianxu/workspace/pair` (this; `PAIR_DEV=1` → relaunch verb is `pair-dev`). ariadne = `/Users/xianxu/workspace/ariadne` (substrate). Both on `main`.
workshop/continuation/20260615T225403-cont-improve.md:49:- **No `bin/pair` seed-prompt change** — the resume directive is embedded in the generated file instead. (decision 5)
cmd/internal/dispatcher/dispatcher.go:53:			Stdout:   "pair-go dispatcher skeleton\nlaunch handoff: bin/pair\n",
cmd/internal/dispatcher/dispatcher.go:100:	b.WriteString("Development dispatcher skeleton. Public sessions still start through bin/pair.\n\n")
cmd/internal/dispatcher/dispatcher.go:106:			fmt.Fprintf(&b, "  %-17s %s (compatibility handoff to bin/pair)\n", family.Name, family.Summary)
cmd/internal/dispatcher/dispatcher_test.go:61:	for _, want := range []string{"pair-go", "dispatcher skeleton", "launch handoff: bin/pair"} {
workshop/continuation/20260621T180406-pair-pair.md:54:- `nvim/review/readiness.lua`, `bin/pair-review-readiness`, `nvim/review/seam.lua`, `nvim/init.lua` around `:PairReview` and `PairReviewToggle`, and `tests/review-toggle-test.sh` / `tests/review-resume-test.sh` for M4a'.
cmd/internal/adapt/adapt.go:17:// atomic across processes; bin/pair truncates the file once at session launch
nvim/init.lua:14:-- the embed pid (lua runs inside the embed); bin/pair's cleanup_quit_marker
nvim/init.lua:918:    if not bin or bin == '' then bin = (home ~= '') and (home .. '/bin/pair-review-readiness') or 'pair-review-readiness' end
nvim/init.lua:966:      local bin = (home ~= '') and (home .. '/bin/pair-review-open') or 'pair-review-open'
nvim/init.lua:1655:-- `/`, `~`, or `./` / `../` etc. Plain `bin/pair-wrap` is *not* a path here
nvim/init.lua:1727:-- `pair-wrap`, `lessons.md`, `bin/pair-wrap`, `$PAIR_HOME`,
nvim/init.lua:2982:-- Resolve the change-log session id (#63): the env var bin/pair exports when the
nvim/init.lua:2985:-- order in bin/pair-changelog-open so the polled .ready path matches the base the
nvim/init.lua:3322:  '   (or  pair-dev continue <that-slug>  if this is a dev checkout)',
tests/pair-review-target-test.sh:15:  PAIR_SESSION_ID=envsid "$ROOT/bin/pair-review-target" "$doc" ready >/dev/null
tests/pair-review-target-test.sh:21:  "$ROOT/bin/pair-review-target" "$doc" ready >/dev/null
cmd/internal/transcript/transcript.go:18:// config-<tag>-<agent>.json (written by bin/pair / pair-session-watch.sh).
doctor/emitter-health.sh:9:# (#000046's `pair-dev` PREVENTS staleness at launch; this DIAGNOSES it.)
doctor/emitter-health.sh:91:               echo "          Fix: make install (or launch via pair-dev). See atlas \"Binary freshness\"." ;;
cmd/internal/launcher/args.go:34:		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %s is not implemented by pair-go launch; use bin/pair", argv[0])}
tests/pair-continue-test.sh:3:# name guard (bin/pair, #54). Drives the REAL bin/pair through its
tests/pair-continue-test.sh:22:PAIR="$ROOT/bin/pair"
tests/pair-continue-test.sh:115:# #55 in-session compaction (Alt+Shift+C). Drives the REAL bin/pair via the
tests/pair-continue-test.sh:183:# REAL bin/pair helper seams through PAIR_TEST_CALL with process-level fakes.
nvim/review/readiness.lua:4:-- gathering lives in bin/pair-review-readiness. The 4 cases are
tests/changelog-open-test.sh:2:# End-to-end smoke test for bin/pair-changelog-open (#53/#58).
tests/changelog-open-test.sh:14:if [ ! -x "$PAIR_HOME/bin/pair-changelog" ] || [ ! -x "$PAIR_HOME/bin/pair-scrollback-render" ]; then
tests/changelog-open-test.sh:50:"$PAIR_HOME/bin/pair-changelog-open"
tests/review-window-test.sh:48:if PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-open" "$RT/nope.md" 2>/dev/null; then
tests/review-window-test.sh:54:PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-open" "$RT/doc.md" || true
doctor/SKILL.md:36:   the fix is `make install` / `pair-dev` (#000046), not a matcher edit. Then the
doctor/SKILL.md:58:- The log truncates at each session launch (`bin/pair`), so it reflects the
tests/pair-title-poller-test.sh:2:# Regression + behavior test for bin/pair-title.sh.
tests/pair-title-poller-test.sh:23:POLLER="$ROOT/bin/pair-title.sh"
tests/pair-title-poller-test.sh:103:cat > "$RT/bin/pair-context" <<'EOF'
tests/pair-title-poller-test.sh:108:chmod +x "$RT/bin/zellij" "$RT/bin/pair-context"
tests/changelog-session-key-test.sh:2:# Focused keying test for bin/pair-changelog-open (#63): the change-log base is
tests/changelog-session-key-test.sh:25:run() { rm -f "$tmp/nvim-arg"; "$PAIR_HOME/bin/pair-changelog-open"; }
tests/dev-rebuild-test.sh:10:# critical because bin/pair runs under `set -e` and a failed restart-time build
tests/dev-rebuild-test.sh:18:RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-devrebuild-test.XXXXXX")"
tests/review-readiness-cli-test.sh:22:out="$(PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-readiness" "$REPO/doc \"quoted\".md")"
tests/review-readiness-cli-test.sh:39:  "$ROOT/bin/pair-review-readiness" --prepare "$PREP/doc.md" 2>&1 || true)"
doctor/README.md:29:`make install`, or launch via `pair-dev` (#000046). A `[STALE]` line explains an
doctor/README.md:42:| `session-id/fail` or `near-miss` (aspect 3) | session file moved or id format changed | update `watch_dir`/`find_args`/`extract_id` in `bin/pair-session-watch.sh` |
doctor/README.md:53:- The log truncates at each session launch (`bin/pair`), so it reflects the
nvim/review/seam.lua:4:-- and bin/pair-review-open — can't diverge on the empty-tag fallback
tests/pair-session-watch-test.sh:36:"$ROOT/bin/pair-session-watch.sh" codex test "$ROOT" --no-alt-screen &
tests/emitter-health-test.sh:48:mkdir -p "$RT/pathbin"; cp "$fresh_pw" "$RT/pathbin/pair-wrap"; chmod +x "$RT/pathbin/pair-wrap"
tests/emitter-health-test.sh:50:[ "$got" = "$RT/pathbin/pair-wrap" ] && pass "no pidfile ⇒ resolves via PATH" || fail "PATH fallback wrong: $got"
tests/lib/run-headless.sh:24:# `gtimeout` on macOS via `brew install coreutils`. When neither is on PATH
tests/review-toggle-test.sh:101:  '"' .. vim.env.PAIR_HOME .. '/bin/pair-review-target" "$2" ready >/dev/null',

exec
/bin/zsh -lc "git diff b829ef00346738e3c3d7e40db38c2c03f51e615b..HEAD -- cmd/internal/launcher cmd/pair-go cmd/internal/dispatcher cmd/internal/entrypoint | sed -n '1,260p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
diff --git a/cmd/internal/dispatcher/dispatcher.go b/cmd/internal/dispatcher/dispatcher.go
index 3784f1e..fe1fbfd 100644
--- a/cmd/internal/dispatcher/dispatcher.go
+++ b/cmd/internal/dispatcher/dispatcher.go
@@ -3,12 +3,9 @@ package dispatcher
 import (
 	"bytes"
 	"fmt"
-	"os"
 	"strings"
-	"time"
 
 	"github.com/xianxu/pair/cmd/internal/contextcmd"
-	"github.com/xianxu/pair/cmd/internal/launcher"
 	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
 )
 
@@ -31,7 +28,7 @@ type Result struct {
 // Families returns the planned command families for the Go dispatcher.
 func Families() []CommandFamily {
 	return []CommandFamily{
-		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "prototype"},
+		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "handoff"},
 		{Name: "context", Summary: "agent pane context meter", Status: "implemented"},
 		{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "implemented"},
 		{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
@@ -53,11 +50,11 @@ func Dispatch(args []string) Result {
 		return Result{Stdout: Help(programName), ExitCode: 0}
 	case "version", "--version":
 		return Result{
-			Stdout:   "pair-go dispatcher skeleton\npublic launcher: bin/pair\n",
+			Stdout:   "pair-go dispatcher skeleton\nlaunch handoff: bin/pair\n",
 			ExitCode: 0,
 		}
 	case "launch":
-		return DispatchWithLauncherRuntime(args, osLauncherRuntime())
+		return launchHandoffResult()
 	case "context":
 		return dispatchContext(args[1:])
 	case "scrollback-render":
@@ -89,90 +86,10 @@ func dispatchScrollbackRender(args []string) Result {
 	return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}
 }
 
-type LauncherRuntime struct {
-	Env      launcher.Env
-	Sessions launcher.SessionSource
-	History  launcher.HistoricalScanner
-}
-
-type StaticSessions struct {
-	Sessions []launcher.Session
-	Err      error
-}
-
-func (s StaticSessions) Snapshot() ([]launcher.Session, error) {
-	if s.Err != nil {
-		return nil, s.Err
-	}
-	return s.Sessions, nil
-}
-
-type StaticHistory struct {
-	Tags []launcher.HistoricalTag
-	Err  error
-}
-
-func (h StaticHistory) Scan(_ string, _ time.Time) ([]launcher.HistoricalTag, error) {
-	if h.Err != nil {
-		return nil, h.Err
-	}
-	return h.Tags, nil
-}
-
-func DispatchWithLauncherRuntime(args []string, rt LauncherRuntime) Result {
-	launchArgs := []string(nil)
-	if len(args) > 1 {
-		launchArgs = args[1:]
-	}
-	if len(launchArgs) > 0 && (launchArgs[0] == "help" || launchArgs[0] == "--help" || launchArgs[0] == "-h") {
-		return Result{Stdout: LaunchHelp(programName), ExitCode: 0}
-	}
-	outcome, err := launcher.Run(launchArgs, rt.Env, rt.Sessions, rt.History)
-	if err != nil {
-		return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
-	}
-	decision := outcome.Decision
+func launchHandoffResult() Result {
 	return Result{
-		Stderr: fmt.Sprintf(
-			"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
-			decision.Action,
-			decision.Tag,
-			decision.SessionName,
-		),
-		ExitCode: 3,
-	}
-}
-
-func LaunchHelp(program string) string {
-	return fmt.Sprintf(`Usage: %s launch [agent] [-- agent-args...]
-       %s launch resume <tag>
-
-Guarded decision-phase prototype. Public sessions still start through bin/pair.
-This command parses launch inputs and computes the create/attach/picker decision,
-then stops before invoking zellij.
-`, program, program)
-}
-
-func LauncherEnv(home, xdgDataHome, cwd string) launcher.Env {
-	return launcher.Env{
-		Home:     home,
-		XDGData:  xdgDataHome,
-		Cwd:      cwd,
-		Now:      time.Now(),
-		HistoryD: 14,
-	}
-}
-
-func osLauncherRuntime() LauncherRuntime {
-	home := os.Getenv("HOME")
-	xdg := os.Getenv("XDG_DATA_HOME")
-	cwd, _ := os.Getwd()
-	env := LauncherEnv(home, xdg, cwd)
-	dataDir := launcher.ResolveDataDir(home, xdg)
-	return LauncherRuntime{
-		Env:      env,
-		Sessions: launcher.ZellijSource{},
-		History:  launcher.HistorySource{DataDir: dataDir},
+		Stderr:   "pair-go launch is a process handoff implemented by cmd/pair-go; call pair-go launch ... instead of dispatcher.Dispatch\n",
+		ExitCode: 2,
 	}
 }
 
@@ -185,6 +102,8 @@ func Help(program string) string {
 	for _, family := range Families() {
 		if family.Status == "prototype" {
 			fmt.Fprintf(&b, "  %-17s %s (prototype; decision-phase only)\n", family.Name, family.Summary)
+		} else if family.Status == "handoff" {
+			fmt.Fprintf(&b, "  %-17s %s (compatibility handoff to bin/pair)\n", family.Name, family.Summary)
 		} else if family.Status == "implemented" {
 			fmt.Fprintf(&b, "  %-17s %s (implemented helper route)\n", family.Name, family.Summary)
 		}
diff --git a/cmd/internal/dispatcher/dispatcher_test.go b/cmd/internal/dispatcher/dispatcher_test.go
index f10f32b..7dc3021 100644
--- a/cmd/internal/dispatcher/dispatcher_test.go
+++ b/cmd/internal/dispatcher/dispatcher_test.go
@@ -21,7 +21,7 @@ func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
 				"Usage: pair-go <command> [args]",
 				"Implemented commands:",
 				"launch",
-				"decision-phase only",
+				"compatibility handoff",
 				"context",
 				"scrollback-render",
 				"wrap",
@@ -35,6 +35,9 @@ func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
 			if strings.Contains(res.Stdout, "launch             session lifecycle and public pair launcher flow (planned; not implemented") {
 				t.Fatalf("Stdout still labels launch unimplemented:\n%s", res.Stdout)
 			}
+			if strings.Contains(res.Stdout, "decision-phase only") {
+				t.Fatalf("Stdout still labels launch decision-phase only:\n%s", res.Stdout)
+			}
 			for _, stale := range []string{
 				"context           agent pane context meter (planned; not implemented",
 				"scrollback-render raw PTY capture to ANSI scrollback (planned; not implemented",
@@ -55,7 +58,7 @@ func TestDispatchVersionIsDevelopmentSkeletonMetadata(t *testing.T) {
 	if res.Stderr != "" {
 		t.Fatalf("Stderr = %q, want empty", res.Stderr)
 	}
-	for _, want := range []string{"pair-go", "dispatcher skeleton", "public launcher: bin/pair"} {
+	for _, want := range []string{"pair-go", "dispatcher skeleton", "launch handoff: bin/pair"} {
 		if !strings.Contains(res.Stdout, want) {
 			t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
 		}
@@ -77,57 +80,15 @@ func TestDispatchPlannedCommandReturnsUnsupported(t *testing.T) {
 	}
 }
 
-func TestDispatchLaunchHelpRoutesToPrototype(t *testing.T) {
+func TestDispatchLaunchReportsProcessHandoff(t *testing.T) {
 	res := Dispatch([]string{"launch", "--help"})
-	if res.ExitCode != 0 {
-		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
-	}
-	if res.Stderr != "" {
-		t.Fatalf("Stderr = %q, want empty", res.Stderr)
-	}
-	for _, want := range []string{"Usage: pair-go launch", "decision-phase prototype"} {
-		if !strings.Contains(res.Stdout, want) {
-			t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
-		}
-	}
-}
-
-func TestDispatchLaunchReturnsPrototypeDecision(t *testing.T) {
-	res := DispatchWithLauncherRuntime([]string{"launch", "resume", "demo"}, LauncherRuntime{
-		Env: LauncherEnv("/home/me", "", "/work/pair"),
-		Sessions: StaticSessions{
-			Sessions: nil,
-		},
-		History: StaticHistory{},
-	})
-	if res.ExitCode != 3 {
-		t.Fatalf("ExitCode = %d, want 3", res.ExitCode)
-	}
-	if res.Stdout != "" {
-		t.Fatalf("Stdout = %q, want empty", res.Stdout)
-	}
-	for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
-		if !strings.Contains(res.Stderr, want) {
-			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
-		}
-	}
-}
-
-func TestDispatchLaunchWithoutArgsReturnsDefaultPrototypeDecision(t *testing.T) {
-	res := DispatchWithLauncherRuntime([]string{"launch"}, LauncherRuntime{
-		Env: LauncherEnv("/home/me", "", "/work/pair"),
-		Sessions: StaticSessions{
-			Sessions: nil,
-		},
-		History: StaticHistory{},
-	})
-	if res.ExitCode != 3 {
-		t.Fatalf("ExitCode = %d, want 3", res.ExitCode)
+	if res.ExitCode != 2 {
+		t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
 	}
 	if res.Stdout != "" {
 		t.Fatalf("Stdout = %q, want empty", res.Stdout)
 	}
-	for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=pair", "session=pair-pair"} {
+	for _, want := range []string{"pair-go launch", "process handoff", "cmd/pair-go"} {
 		if !strings.Contains(res.Stderr, want) {
 			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
 		}
diff --git a/cmd/internal/entrypoint/launch.go b/cmd/internal/entrypoint/launch.go
new file mode 100644
index 0000000..1a40970
--- /dev/null
+++ b/cmd/internal/entrypoint/launch.go
@@ -0,0 +1,20 @@
+package entrypoint
+
+import "path/filepath"
+
+// LegacyLaunchRequest describes the current compatibility handoff from
+// pair-go launch to the shell-owned pair launcher.
+type LegacyLaunchRequest struct {
+	Path string
+	Argv []string
+}
+
+func ResolveLegacyLaunch(executable string, launchArgs []string) LegacyLaunchRequest {
+	argv := make([]string, 0, len(launchArgs)+1)
+	argv = append(argv, "pair")
+	argv = append(argv, launchArgs...)
+	return LegacyLaunchRequest{
+		Path: filepath.Join(filepath.Dir(executable), "pair"),

exec
/bin/zsh -lc 'git ls-files bin/pair bin/pair-dev Makefile.local README.md | xargs -n1 git ls-files --stage --' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
100644 6bbc2eb341e3116728aa206ab0d74f7b67b2a730 0	Makefile.local
100644 dcde60a223e7127147f15b2f637d322401945557 0	README.md
100755 b31ed22f2d2fd602ecf1eb32eb22231e08272c5f 0	bin/pair
100755 82a73daa4002496c2eda2f78246c4a4bd612132b 0	bin/pair-dev

exec
/bin/zsh -lc "nl -ba workshop/plans/000077-go-entrypoint-switch-plan.md | sed -n '1,260p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	# Go Entrypoint Switch Implementation Plan
     2	
     3	> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.
     4	
     5	**Goal:** Make `pair-go launch ...` a meaningful Go-owned entrypoint that runs the existing `pair` launcher with compatible arguments while leaving `pair` and `pair-dev` stable.
     6	
     7	**Architecture:** Keep the shell launcher as the behavioral source of truth for this migration window (`ARCH-DRY`, `ARCH-PURPOSE`). Add a small pure decision layer that resolves the sibling launcher path and argv, plus a thin process boundary that execs it (`ARCH-PURE`). The existing dispatcher remains the owner for `pair-go help`, `pair-go context`, and `pair-go scrollback-render`.
     8	
     9	**Tech Stack:** Go 1.x, `os.Executable`, `syscall.Exec` or an injected process runner for tests, existing Bash launcher `bin/pair`, existing Makefile build target for `cmd/pair-go`.
    10	
    11	---
    12	
    13	## Chunk 1: Concepts
    14	
    15	### Pure Entities
    16	
    17	| Name | Lives in | Status |
    18	|------|----------|--------|
    19	| `LegacyLaunchRequest` | `cmd/internal/entrypoint/launch.go` | new |
    20	| `ResolveLegacyLaunch` | `cmd/internal/entrypoint/launch.go` | new |
    21	
    22	**LegacyLaunchRequest** — the process-independent request to invoke the current shell launcher.
    23	
    24	- **Relationships:** 1:1 with a `pair-go launch ...` invocation; owns the resolved launcher path and argv passed to `bin/pair`.
    25	- **DRY rationale:** Centralizes the compatibility mapping from `pair-go launch <pair-args>` to `pair <pair-args>` so tests and the process boundary do not each reconstruct argv handling.
    26	- **Future extensions:** This is the place to widen from shell handoff to native Go launch when #78/#79 remove enough shell-owned behavior.
    27	
    28	**ResolveLegacyLaunch** — pure function that converts the running executable path plus `pair-go launch` args into a `LegacyLaunchRequest`.
    29	
    30	- **Relationships:** Used by `cmd/pair-go` before the IO boundary; independent of zellij, fzf, or the filesystem except for caller-provided existence checks.
    31	- **DRY rationale:** Reuses the existing `bin/pair` launcher as the only real session lifecycle owner instead of duplicating create/attach/resume/list/rename behavior in Go.
    32	- **Future extensions:** Can accept a `PAIR_GO_LAUNCH_NATIVE=1` mode or replacement implementation later without changing top-level command parsing.
    33	
    34	### Integration Points
    35	
    36	| Name | Lives in | Status | Wraps |
    37	|------|----------|--------|-------|
    38	| `LegacyLauncherRunner` | `cmd/pair-go/main.go` | new | `os.Executable`, `os.Stat`, `syscall.Exec` |
    39	| `pair-go launch` docs | `README.md`, `atlas/architecture.md`, `atlas/go-migration-inventory.md` | modified | operator-facing migration contract |
    40	
    41	**LegacyLauncherRunner** — the thin IO shell that resolves the current binary path, validates sibling `pair`, and replaces the current process with it.
    42	
    43	- **Injected into:** `runWithLegacyRunner` in tests, so process behavior is asserted with fakes rather than actually starting zellij.
    44	- **Future extensions:** Native launch can replace only this runner once the pure launcher core owns side effects.
    45	
    46	**pair-go launch docs** — documentation that explains the current migration boundary.
    47	
    48	- **Injected into:** README command usage and atlas migration notes.
    49	- **Future extensions:** #78/#79 can update the same docs when stateful shell glue or packaging changes.
    50	
    51	## Chunk 2: Spec And Handoff Tests
    52	
    53	### Task 1: Add pure launch request tests
    54	
    55	**Files:**
    56	- Create: `cmd/internal/entrypoint/launch_test.go`
    57	- Create: `cmd/internal/entrypoint/launch.go`
    58	
    59	- [ ] **Step 1: Write the failing tests**
    60	
    61	```go
    62	func TestResolveLegacyLaunchDropsLaunchVerb(t *testing.T) {
    63		req := entrypoint.ResolveLegacyLaunch("/repo/bin/pair-go", []string{"claude", "--", "--resume"})
    64		if req.Path != "/repo/bin/pair" {
    65			t.Fatalf("Path = %q", req.Path)
    66		}
    67		want := []string{"pair", "claude", "--", "--resume"}
    68		if !reflect.DeepEqual(req.Argv, want) {
    69			t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
    70		}
    71	}
    72	
    73	func TestResolveLegacyLaunchPreservesSubcommands(t *testing.T) {
    74		req := entrypoint.ResolveLegacyLaunch("/repo/bin/pair-go", []string{"resume", "demo"})
    75		want := []string{"pair", "resume", "demo"}
    76		if !reflect.DeepEqual(req.Argv, want) {
    77			t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
    78		}
    79	}
    80	```
    81	
    82	- [ ] **Step 2: Run the tests to verify RED**
    83	
    84	Run: `go test ./cmd/internal/entrypoint -run TestResolveLegacyLaunch -count=1`
    85	
    86	Expected: FAIL because `cmd/internal/entrypoint` does not exist yet.
    87	
    88	- [ ] **Step 3: Implement minimal pure resolver**
    89	
    90	Create `LegacyLaunchRequest` with `Path string` and `Argv []string`. Implement `ResolveLegacyLaunch(executable string, launchArgs []string)` using `filepath.Dir(executable)` and sibling `pair`. The argv must start with `"pair"` and append all launch args unchanged.
    91	
    92	- [ ] **Step 4: Run tests to verify GREEN**
    93	
    94	Run: `go test ./cmd/internal/entrypoint -run TestResolveLegacyLaunch -count=1`
    95	
    96	Expected: PASS.
    97	
    98	### Task 2: Add `pair-go launch` process-boundary tests
    99	
   100	**Files:**
   101	- Modify: `cmd/pair-go/main.go`
   102	- Modify: `cmd/pair-go/main_test.go`
   103	- Modify: `cmd/pair-go/launch_process_test.go`
   104	- Modify: `cmd/internal/dispatcher/dispatcher_test.go`
   105	
   106	- [ ] **Step 1: Write failing main tests for handoff**
   107	
   108	Add a fake runner test that calls the test seam with args `[]string{"launch", "claude", "--", "--resume"}` and asserts:
   109	
   110	- exit code is the fake runner's code;
   111	- no dispatcher prototype text is printed;
   112	- runner receives sibling `bin/pair`;
   113	- runner receives argv `["pair", "claude", "--", "--resume"]`.
   114	- runner receives inherited environment entries, including a sentinel such as `PAIR_TEST_ENV=kept`, so the shell launcher sees the same env that `pair-go` received (`ARCH-PURPOSE` compatibility surface).
   115	
   116	Also add a missing-launcher test that returns a not-found stat result and asserts stderr mentions `pair-go launch`, `bin/pair`, `make build`, `make install`, and `dev-aliases.sh`.
   117	
   118	- [ ] **Step 2: Run targeted tests to verify RED**
   119	
   120	Run: `go test ./cmd/pair-go -run 'TestRunLaunch' -count=1`
   121	
   122	Expected: FAIL because current `pair-go launch` returns prototype exit code 3 and never invokes the fake runner.
   123	
   124	- [ ] **Step 3: Update main seam**
   125	
   126	Refactor `cmd/pair-go/main.go` so `run` delegates to `runWithRuntime`. For `args[0] == "launch"`, use the injected `LegacyLauncherRunner` and `entrypoint.ResolveLegacyLaunch`; otherwise preserve existing dispatcher behavior. The real runner should:
   127	
   128	- get `os.Executable()`;
   129	- resolve sibling `pair`;
   130	- `os.Stat` it and require a non-directory executable path;
   131	- call `syscall.Exec(path, argv, os.Environ())`.
   132	
   133	In tests, fake these methods and return deterministic codes.
   134	
   135	- [ ] **Step 4: Update stale prototype expectations**
   136	
   137	Change launch tests that currently expect `"prototype decision"` and exit code `3` to expect legacy handoff behavior or delete them if covered by the new fake-runner tests. Keep dispatcher tests for `context`, `scrollback-render`, help, version, and planned command errors.
   138	
   139	- [ ] **Step 5: Run targeted tests to verify GREEN**
   140	
   141	Run: `go test ./cmd/pair-go ./cmd/internal/dispatcher ./cmd/internal/entrypoint -count=1`
   142	
   143	Expected: PASS.
   144	
   145	## Chunk 3: Build Wiring And Documentation
   146	
   147	### Task 3: Keep Makefile wiring accurate
   148	
   149	**Files:**
   150	- Modify: `Makefile.local`
   151	
   152	- [ ] **Step 1: Write or update dependency expectation**
   153	
   154	Inspect the `$(BIN_DIR)/pair-go` dependency list and add `cmd/internal/entrypoint/launch.go` so `make pair-go` rebuilds when the new resolver changes.
   155	
   156	- [ ] **Step 2: Run build**
   157	
   158	Run: `make pair-go`
   159	
   160	Expected: `bin/pair-go` builds successfully.
   161	
   162	### Task 4: Document the migration boundary
   163	
   164	**Files:**
   165	- Modify: `README.md`
   166	- Modify: `atlas/architecture.md`
   167	- Modify: `atlas/go-migration-inventory.md`
   168	- Modify: `workshop/issues/000077-go-entrypoint-switch.md`
   169	
   170	- [ ] **Step 1: Update README**
   171	
   172	Add a short development note near Command Usage:
   173	
   174	```markdown
   175	`pair-go launch ...` is the Go-owned migration entrypoint for testing the launcher path. It accepts the same arguments after `launch` that `pair` accepts directly, then hands off to the current `pair` launcher for one migration window. In a dev shell sourced from `../ariadne/construct/dev-aliases.sh`, `pair-go` rebuilds from `cmd/pair-go` automatically before running; no `pair-go-dev` command is needed.
   176	```
   177	
   178	- [ ] **Step 2: Update atlas**
   179	
   180	Update `atlas/architecture.md` and `atlas/go-migration-inventory.md` so they no longer describe `pair-go launch` as decision-phase only. State that #77 makes it a Go-owned compatibility handoff to `bin/pair`, while `bin/pair` remains the public stable entrypoint and the real zellij lifecycle remains shell-owned.
   181	
   182	- [ ] **Step 3: Tick issue plan items and log verification intent**
   183	
   184	In #77, tick completed plan rows as they land and add a log entry with the exact commands run.
   185	
   186	## Chunk 4: Verification And Close
   187	
   188	### Task 5: Full verification
   189	
   190	**Files:**
   191	- No new code files.
   192	
   193	- [ ] **Step 1: Run focused Go tests**
   194	
   195	Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1`
   196	
   197	Expected: PASS.
   198	
   199	- [ ] **Step 2: Run full Go suite**
   200	
   201	Run: `go test ./... -count=1`
   202	
   203	Expected: PASS.
   204	
   205	- [ ] **Step 3: Run build**
   206	
   207	Run: `make pair-go`
   208	
   209	Expected: PASS and `bin/pair-go` exists.
   210	
   211	- [ ] **Step 4: Smoke test non-zellij routes**
   212	
   213	Run:
   214	
   215	```bash
   216	make test-dev-rebuild
   217	bin/pair-go help
   218	bin/pair-go launch --help
   219	bin/pair --help
   220	bin/pair-dev --help
   221	```
   222	
   223	Expected: `make test-dev-rebuild` passes, proving the existing `PAIR_DEV` rebuild hook still works. Help output succeeds. `bin/pair-go launch --help` should print the existing `pair` help because it hands off to `bin/pair --help`.
   224	
   225	- [ ] **Step 5: SDLC close**
   226	
   227	Run: `sdlc close --issue 77 --verified '<focused tests, full go test, build, and help smoke evidence>'`.
   228	
   229	Expected: SDLC close runs its review gate; fix any Critical/Important findings before merge.

exec
/bin/zsh -lc 'git status --short && git diff --name-only b829ef00346738e3c3d7e40db38c2c03f51e615b..HEAD -- workshop/plans workshop/issues atlas README.md Makefile.local' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 M workshop/issues/000077-go-entrypoint-switch.md
?? workshop/issues/000089-review-mode-should-disable-edit-while-agent-update-the-doc.md
Makefile.local
README.md
atlas/architecture.md
atlas/go-migration-inventory.md
workshop/issues/000077-go-entrypoint-switch.md

exec
/bin/zsh -lc 'bin/pair-go launch --help >/tmp/pair-go-launch-help.out 2>/tmp/pair-go-launch-help.err; code=$?; printf '"'code=%s\\nstdout_first=%s\\nstderr=%s\\n' \""'$code" "$(head -n 1 /tmp/pair-go-launch-help.out)" "$(cat /tmp/pair-go-launch-help.err)"' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
code=0
stdout_first=pair — Neovim-backed input field for any TUI coding agent.
stderr=

exec
/bin/zsh -lc 'go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1' in /Users/xianxu/workspace/pair
 succeeded in 1401ms:
ok  	github.com/xianxu/pair/cmd/internal/entrypoint	0.205s
ok  	github.com/xianxu/pair/cmd/pair-go	1.242s
ok  	github.com/xianxu/pair/cmd/internal/dispatcher	0.488s

codex
I found one tracker mismatch in the durable plan artifact: it names an integration entity that does not exist under that name in the code. I’m also checking the install layout because `pair-go` resolves a sibling `pair`, while this repo’s `make install` currently copies only Go binaries.
exec
/bin/zsh -lc 'rg -n "LegacyLauncherRunner|runWithLegacyRunner|legacyRuntime|runWithLegacyRuntime" cmd workshop/plans/000077-go-entrypoint-switch-plan.md' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
workshop/plans/000077-go-entrypoint-switch-plan.md:38:| `LegacyLauncherRunner` | `cmd/pair-go/main.go` | new | `os.Executable`, `os.Stat`, `syscall.Exec` |
workshop/plans/000077-go-entrypoint-switch-plan.md:41:**LegacyLauncherRunner** — the thin IO shell that resolves the current binary path, validates sibling `pair`, and replaces the current process with it.
workshop/plans/000077-go-entrypoint-switch-plan.md:43:- **Injected into:** `runWithLegacyRunner` in tests, so process behavior is asserted with fakes rather than actually starting zellij.
workshop/plans/000077-go-entrypoint-switch-plan.md:126:Refactor `cmd/pair-go/main.go` so `run` delegates to `runWithRuntime`. For `args[0] == "launch"`, use the injected `LegacyLauncherRunner` and `entrypoint.ResolveLegacyLaunch`; otherwise preserve existing dispatcher behavior. The real runner should:
cmd/pair-go/main_test.go:45:	code := runWithLegacyRuntime([]string{"launch", "--help"}, &stdout, &stderr, rt)
cmd/pair-go/main_test.go:72:	code := runWithLegacyRuntime([]string{"launch", "claude", "--", "--resume"}, &stdout, &stderr, rt)
cmd/pair-go/main_test.go:102:	code := runWithLegacyRuntime([]string{"launch", "claude"}, &stdout, &stderr, rt)
cmd/pair-go/main.go:20:	return runWithLegacyRuntime(args, stdout, stderr, osLegacyRuntime{})
cmd/pair-go/main.go:23:type legacyRuntime interface {
cmd/pair-go/main.go:30:func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
cmd/pair-go/main.go:38:func runLegacyLaunch(args []string, stderr io.Writer, rt legacyRuntime) int {

exec
/bin/zsh -lc "nl -ba Makefile.local | sed -n '24,40p' && nl -ba workshop/plans/000077-go-entrypoint-switch-plan.md | sed -n '34,45p' && nl -ba cmd/pair-go/main.go | sed -n '23,38p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
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
    34	### Integration Points
    35	
    36	| Name | Lives in | Status | Wraps |
    37	|------|----------|--------|-------|
    38	| `LegacyLauncherRunner` | `cmd/pair-go/main.go` | new | `os.Executable`, `os.Stat`, `syscall.Exec` |
    39	| `pair-go launch` docs | `README.md`, `atlas/architecture.md`, `atlas/go-migration-inventory.md` | modified | operator-facing migration contract |
    40	
    41	**LegacyLauncherRunner** — the thin IO shell that resolves the current binary path, validates sibling `pair`, and replaces the current process with it.
    42	
    43	- **Injected into:** `runWithLegacyRunner` in tests, so process behavior is asserted with fakes rather than actually starting zellij.
    44	- **Future extensions:** Native launch can replace only this runner once the pure launcher core owns side effects.
    45	
    23	type legacyRuntime interface {
    24		Executable() (string, error)
    25		Stat(path string) error
    26		Environ() []string
    27		Exec(path string, argv []string, env []string) int
    28	}
    29	
    30	func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
    31		if len(args) > 0 && args[0] == "launch" {
    32			return runLegacyLaunch(args[1:], stderr, rt)
    33		}
    34		res := dispatcher.Dispatch(args)
    35		return writeResult(res, stdout, stderr)
    36	}
    37	
    38	func runLegacyLaunch(args []string, stderr io.Writer, rt legacyRuntime) int {

exec
/bin/zsh -lc "make -n install | sed -n '1,80p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
if command -v go >/dev/null 2>&1; then \
	    :; \
	elif command -v brew >/dev/null 2>&1; then \
	    echo "==> go not found — installing via Homebrew (brew install go)"; \
	    brew install go; \
	else \
	    echo "Error: ariadne ships cmd/sdlc and needs the Go toolchain to build it," >&2; \
	    echo "  but 'go' is not on PATH and Homebrew isn't available to install it." >&2; \
	    echo "  Install Go 1.26+ from https://go.dev/dl/ and re-run." >&2; \
	    exit 1; \
	fi
go build -o bin/pair-scrollback-render ./cmd/pair-scrollback-render
go build -o bin/pair-context ./cmd/pair-context
if [ -f go.mod ]; then \
	    found=0; \
	    skipped=0; \
	    for d in cmd/*/; do \
	        name=$(basename "$d"); \
	        if [ -f "$d/.skip-make-build" ]; then \
	            echo "  (skipping $name — .skip-make-build sentinel present)"; \
	            skipped=$((skipped + 1)); \
	            continue; \
	        fi; \
	        if [ -f "$d/main.go" ]; then \
	            mkdir -p bin; \
	            echo "==> Building $name..."; \
	            go build -o "bin/$name" "./$d" || exit 1; \
	            found=1; \
	        fi; \
	    done; \
	    if [ "$found" = "0" ] && [ "$skipped" = "0" ]; then \
	        echo "  (no cmd/*/main.go to build)"; \
	    fi; \
	fi
/Library/Developer/CommandLineTools/usr/bin/make --no-print-directory local-build
:
mkdir -p /Users/xianxu/.local/bin
for b in pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-go; do \
	    install -m 755 bin/$b /Users/xianxu/.local/bin/$b && \
	    echo "installed: /Users/xianxu/.local/bin/$b"; \
	done

exec
/bin/zsh -lc 'go test ./... -count=1' in /Users/xianxu/workspace/pair
 succeeded in 16085ms:
ok  	github.com/xianxu/pair/cmd/internal/adapt	0.179s
ok  	github.com/xianxu/pair/cmd/internal/contextcmd	0.315s
ok  	github.com/xianxu/pair/cmd/internal/ctxmeter	0.454s
ok  	github.com/xianxu/pair/cmd/internal/dispatcher	0.733s
ok  	github.com/xianxu/pair/cmd/internal/entrypoint	0.578s
ok  	github.com/xianxu/pair/cmd/internal/launcher	2.496s
ok  	github.com/xianxu/pair/cmd/internal/model	3.474s
ok  	github.com/xianxu/pair/cmd/internal/scrollbackcmd	0.976s
ok  	github.com/xianxu/pair/cmd/internal/transcript	1.105s
ok  	github.com/xianxu/pair/cmd/pair-changelog	15.715s
ok  	github.com/xianxu/pair/cmd/pair-context	3.799s
ok  	github.com/xianxu/pair/cmd/pair-continuation	5.800s
ok  	github.com/xianxu/pair/cmd/pair-go	4.009s
?   	github.com/xianxu/pair/cmd/pair-scribe	[no test files]
?   	github.com/xianxu/pair/cmd/pair-scrollback-render	[no test files]
ok  	github.com/xianxu/pair/cmd/pair-slug	6.466s
ok  	github.com/xianxu/pair/cmd/pair-wrap	1.913s

codex
The full Go suite passes locally. One concrete behavioral gap remains around the documented `make install` recovery path: `make install` installs `pair-go` but not its sibling `pair`, so the installed binary cannot satisfy the new sibling resolution unless `pair` came from another installer.
exec
/bin/zsh -lc 'RT=$(mktemp -d "${TMPDIR:-/tmp}/pair-install-review.XXXXXX"); HOME="$RT" make install >/tmp/pair-install-review-install.out 2>/tmp/pair-install-review-install.err; code=$?; printf '"'install_code=%s\\n' \""'$code"; if [ "$code" -eq 0 ]; then "$RT/.local/bin/pair-go" launch --help >/tmp/pair-install-review-launch.out 2>/tmp/pair-install-review-launch.err; printf '"'launch_code=%s\\nlaunch_stderr=%s\\ninstalled_pair=%s\\n' \""'$?" "$(cat /tmp/pair-install-review-launch.err)" "$(test -e "$RT/.local/bin/pair" && echo yes || echo no)"; fi' in /Users/xianxu/workspace/pair
 succeeded in 9138ms:
install_code=0
launch_code=1
launch_stderr=pair-go launch: pair launcher not found at /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-install-review.LYUpPs/.local/bin/pair (stat /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-install-review.LYUpPs/.local/bin/pair: no such file or directory); run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell
installed_pair=no

codex
VERDICT: REWORK (confidence: high)

The handoff shape is mostly right: `pair-go launch` now bypasses the pure dispatcher, computes `pair` argv, preserves env, and execs the shell launcher. The blocking issue is install-layout correctness: the new code requires a sibling `pair`, but this repo’s `make install` installs `pair-go` without installing `pair`, and the failure message recommends `make install` even though that does not fix the missing sibling.

1. Strengths:
- [cmd/internal/entrypoint/launch.go:12](/Users/xianxu/workspace/pair/cmd/internal/entrypoint/launch.go:12) keeps argv mapping pure and tiny. ARCH-PURE pass.
- [cmd/pair-go/main.go:30](/Users/xianxu/workspace/pair/cmd/pair-go/main.go:30) keeps launch as the only process-boundary special case; helper routes still use the dispatcher.
- [cmd/pair-go/main_test.go:64](/Users/xianxu/workspace/pair/cmd/pair-go/main_test.go:64) pins argv and inherited env, which is the right compatibility surface.

2. Critical findings:
- [Makefile.local:29](/Users/xianxu/workspace/pair/Makefile.local:29) / [cmd/internal/entrypoint/launch.go:17](/Users/xianxu/workspace/pair/cmd/internal/entrypoint/launch.go:17): `make install` installs `pair-go` but not `pair`, while `pair-go launch` only looks for sibling `pair`. I verified with a temp `HOME`: `make install` succeeded, then `$HOME/.local/bin/pair-go launch --help` failed with “pair launcher not found …/.local/bin/pair”; `installed_pair=no`. Fix by making the install/package layout install `bin/pair` alongside `pair-go`, or change the resolver/diagnostic to match the actual supported installed layout. ARCH-PURPOSE.
- [workshop/plans/000077-go-entrypoint-switch-plan.md:38](/Users/xianxu/workspace/pair/workshop/plans/000077-go-entrypoint-switch-plan.md:38): Core concepts table claims new entity `LegacyLauncherRunner` and [line 43](/Users/xianxu/workspace/pair/workshop/plans/000077-go-entrypoint-switch-plan.md:43) claims `runWithLegacyRunner`, but the code has `legacyRuntime` and `runWithLegacyRuntime` at [cmd/pair-go/main.go:23](/Users/xianxu/workspace/pair/cmd/pair-go/main.go:23). Per the boundary checklist, plan/code entity contradiction is Critical. Fix by revising the plan artifact or aligning names.

3. Important findings:
- Missing test coverage for the installed sibling layout. Add a process-level test or script seam that simulates `make install` output and asserts installed `pair-go launch --help` can find installed `pair`, or explicitly documents/tests that installed `pair-go` is unsupported outside the repo `bin/` layout.

4. Minor findings:
- None.

5. Test coverage notes:
- Ran `go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1`: pass.
- Ran `go test ./... -count=1`: pass.
- Ran `bin/pair-go launch --help`: pass from repo `bin/`.
- Temp `HOME` install smoke exposed the blocking installed-layout failure.

6. Architectural notes for upcoming work:
- ARCH-DRY: pass, real lifecycle remains single-sourced in `bin/pair`.
- ARCH-PURE: pass for resolver plus injected exec boundary.
- ARCH-PURPOSE: fail until every documented supported invocation layout for `pair-go launch` can actually derive the sibling launcher it requires.

7. Plan revision recommendations:
- Append `## Revisions` to `workshop/plans/000077-go-entrypoint-switch-plan.md`: `2026-06-30 — Boundary review found the integration entity was implemented as legacyRuntime/runWithLegacyRuntime, not LegacyLauncherRunner/runWithLegacyRunner; update the Core concepts table and implementation references to match the code, or rename the code to the planned entity names.`
- Add a revision noting the install-layout discovery: `make install installs pair-go without sibling pair, so the handoff contract needs install/package wiring or a narrower documented support boundary.`
tokens used
92,218
VERDICT: REWORK (confidence: high)

The handoff shape is mostly right: `pair-go launch` now bypasses the pure dispatcher, computes `pair` argv, preserves env, and execs the shell launcher. The blocking issue is install-layout correctness: the new code requires a sibling `pair`, but this repo’s `make install` installs `pair-go` without installing `pair`, and the failure message recommends `make install` even though that does not fix the missing sibling.

1. Strengths:
- [cmd/internal/entrypoint/launch.go:12](/Users/xianxu/workspace/pair/cmd/internal/entrypoint/launch.go:12) keeps argv mapping pure and tiny. ARCH-PURE pass.
- [cmd/pair-go/main.go:30](/Users/xianxu/workspace/pair/cmd/pair-go/main.go:30) keeps launch as the only process-boundary special case; helper routes still use the dispatcher.
- [cmd/pair-go/main_test.go:64](/Users/xianxu/workspace/pair/cmd/pair-go/main_test.go:64) pins argv and inherited env, which is the right compatibility surface.

2. Critical findings:
- [Makefile.local:29](/Users/xianxu/workspace/pair/Makefile.local:29) / [cmd/internal/entrypoint/launch.go:17](/Users/xianxu/workspace/pair/cmd/internal/entrypoint/launch.go:17): `make install` installs `pair-go` but not `pair`, while `pair-go launch` only looks for sibling `pair`. I verified with a temp `HOME`: `make install` succeeded, then `$HOME/.local/bin/pair-go launch --help` failed with “pair launcher not found …/.local/bin/pair”; `installed_pair=no`. Fix by making the install/package layout install `bin/pair` alongside `pair-go`, or change the resolver/diagnostic to match the actual supported installed layout. ARCH-PURPOSE.
- [workshop/plans/000077-go-entrypoint-switch-plan.md:38](/Users/xianxu/workspace/pair/workshop/plans/000077-go-entrypoint-switch-plan.md:38): Core concepts table claims new entity `LegacyLauncherRunner` and [line 43](/Users/xianxu/workspace/pair/workshop/plans/000077-go-entrypoint-switch-plan.md:43) claims `runWithLegacyRunner`, but the code has `legacyRuntime` and `runWithLegacyRuntime` at [cmd/pair-go/main.go:23](/Users/xianxu/workspace/pair/cmd/pair-go/main.go:23). Per the boundary checklist, plan/code entity contradiction is Critical. Fix by revising the plan artifact or aligning names.

3. Important findings:
- Missing test coverage for the installed sibling layout. Add a process-level test or script seam that simulates `make install` output and asserts installed `pair-go launch --help` can find installed `pair`, or explicitly documents/tests that installed `pair-go` is unsupported outside the repo `bin/` layout.

4. Minor findings:
- None.

5. Test coverage notes:
- Ran `go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1`: pass.
- Ran `go test ./... -count=1`: pass.
- Ran `bin/pair-go launch --help`: pass from repo `bin/`.
- Temp `HOME` install smoke exposed the blocking installed-layout failure.

6. Architectural notes for upcoming work:
- ARCH-DRY: pass, real lifecycle remains single-sourced in `bin/pair`.
- ARCH-PURE: pass for resolver plus injected exec boundary.
- ARCH-PURPOSE: fail until every documented supported invocation layout for `pair-go launch` can actually derive the sibling launcher it requires.

7. Plan revision recommendations:
- Append `## Revisions` to `workshop/plans/000077-go-entrypoint-switch-plan.md`: `2026-06-30 — Boundary review found the integration entity was implemented as legacyRuntime/runWithLegacyRuntime, not LegacyLauncherRunner/runWithLegacyRunner; update the Core concepts table and implementation references to match the code, or rename the code to the planned entity names.`
- Add a revision noting the install-layout discovery: `make install installs pair-go without sibling pair, so the handoff contract needs install/package wiring or a narrower documented support boundary.`

---

## Re-review — 2026-06-30T15:39:20-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 77 — pair Go entrypoint switch |
| repo | pair |
| issue file | workshop/issues/000077-go-entrypoint-switch.md |
| boundary | whole-issue close |
| milestone | — |
| window | b829ef00346738e3c3d7e40db38c2c03f51e615b..HEAD |
| command | sdlc close --issue 77 |
| reviewer | codex |
| timestamp | 2026-06-30T15:39:20-07:00 |
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
session id: 019f1aae-3522-78d3-b458-57a12f6eed60
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **pair** repository.

- repository: pair   (root: /Users/xianxu/workspace/pair)
- issue:      pair#77   (file: workshop/issues/000077-go-entrypoint-switch.md)
- window:     Base: b829ef00346738e3c3d7e40db38c2c03f51e615b   Head: HEAD

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
diff --git a/Makefile.local b/Makefile.local
index ccdde68..7b6c29a 100644
--- a/Makefile.local
+++ b/Makefile.local
@@ -1,7 +1,7 @@
 # Repo-specific Makefile targets.
 # This file is included by Makefile — add your own targets here.
 
-.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
+.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
 
 # Cross-cutting Go-binary targets. Every Go binary lives under cmd/<name>/
 # and ships with the `pair-` prefix so it doesn't collide with anything
@@ -27,6 +27,7 @@
 BIN_DIR     := bin
 INSTALL_DIR := $(HOME)/.local/bin
 GO_BINS     := pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-go
+SHELL_BINS  := pair pair-dev
 
 build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
 
@@ -36,6 +37,10 @@ install: build
 	    install -m 755 $(BIN_DIR)/$$b $(INSTALL_DIR)/$$b && \
 	    echo "installed: $(INSTALL_DIR)/$$b"; \
 	done
+	@for b in $(SHELL_BINS); do \
+	    ln -sf "$(CURDIR)/$(BIN_DIR)/$$b" "$(INSTALL_DIR)/$$b" && \
+	    echo "linked: $(INSTALL_DIR)/$$b -> $(CURDIR)/$(BIN_DIR)/$$b"; \
+	done
 
 # Per-binary aliases — `make pair-scribe`, etc.
 pair-scribe:            $(BIN_DIR)/pair-scribe
@@ -57,7 +62,7 @@ pair-go:                $(BIN_DIR)/pair-go
 #                     inside the vt.Emulator library API itself, which is
 #                     outside our scope to fix — exclude that package
 #                     here rather than chase a library-level race.
-test: test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace test-review
+test: test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace test-review
 	go test ./... -count=1
 
 # pair-title poller: single-instance guard (identity-checked liveness so a
@@ -181,6 +186,12 @@ test-adapt-schema:
 test-dev-rebuild:
 	bash tests/dev-rebuild-test.sh
 
+# pair-go launch handoff (#77): installed pair-go requires sibling pair. Local
+# installs link shell launchers back to the source tree so pair keeps its
+# PAIR_HOME asset resolution.
+test-pair-go-install-layout:
+	bash tests/pair-go-install-layout-test.sh
+
 # pair-doctor emitter-health probe (#000047): marker check + binary selection
 # (running-via-pidfile vs PATH). See tests/emitter-health-test.sh.
 test-emitter-health:
@@ -239,9 +250,10 @@ $(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill
 $(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-continuation/git.go cmd/pair-continuation/main.go go.mod
 	go build -o $@ ./cmd/pair-continuation
 
-# pair-go — development-only dispatcher skeleton for the future primary Go CLI.
-# The public launcher remains bin/pair until the entrypoint-switch issue.
-$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
+# pair-go — development dispatcher and Go-owned launch handoff.
+# `pair-go launch ...` execs sibling bin/pair for the current migration window;
+# helper routes stay in the dispatcher.
+$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/entrypoint/launch.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
 	go build -o $@ ./cmd/pair-go
 
 
diff --git a/README.md b/README.md
index c4681a3..dcde60a 100644
--- a/README.md
+++ b/README.md
@@ -200,6 +200,13 @@ pair rename <old> <new>          # rename every tag-scoped file in
 pair -h, --help                  # show full help
 ```
 
+`pair-go launch ...` is the Go-owned migration entrypoint for testing the
+launcher path. It accepts the same arguments after `launch` that `pair` accepts
+directly, then hands off to the current `pair` launcher for one migration
+window. In a dev shell sourced from `../ariadne/construct/dev-aliases.sh`,
+`pair-go` rebuilds from `cmd/pair-go` automatically before running; no
+`pair-go-dev` command is needed.
+
 Use `--` to separate pair's positional from agent flags. Without it, pair only takes `<agent>` as a positional and everything else is rejected.
 
 Agent args (after `--`) are appended to the agent command line on **create**. Reattaching to an existing session does not re-launch the agent, so the args don't apply on attach. (The picker connects you to whatever's already running.)
diff --git a/atlas/architecture.md b/atlas/architecture.md
index 5e308ab..c8df19e 100644
--- a/atlas/architecture.md
+++ b/atlas/architecture.md
@@ -14,7 +14,7 @@ contract for the Go packaging migration lives in
 
 ```
 bin/pair                     # entry point (launcher)
-bin/pair-go                  # opt-in Go dispatcher skeleton; public launcher remains bin/pair
+bin/pair-go                  # opt-in Go dispatcher + launch handoff; public launcher remains bin/pair
 bin/clipboard-to-pane.sh     # read clipboard, hand off to nvim's PairPasteQuote
 bin/copy-on-select.sh        # invoked by zellij copy_command on mouse-up
 bin/pair-quit.sh             # invoked by Alt+x — marks + kills session
@@ -45,15 +45,19 @@ internal subcommands or dispatch modes behind that primary binary (`pair wrap`,
 `pair continuation`, `pair scribe`) instead of staying as independently managed
 installed commands forever.
 
-As of #75, `bin/pair-go` is a development-only dispatcher with a guarded
-`pair-go launch` decision-phase prototype. The public launcher remains
-`bin/pair`; `pair-go launch` parses the launcher subset, resolves the Pair data
-dir, snapshots fakeable zellij/history state, and computes create/attach/picker
-decisions through the pure `cmd/internal/launcher` core. It then stops with an
-explicit prototype message before invoking zellij. Real zellij lifecycle,
-prompt/fzf UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation,
-rename, config/session migration, and title-poller behavior remain shell-owned
-until later migration issues.
+As of #77, `bin/pair-go` is the Go-owned launch entrypoint under test:
+`pair-go launch ...` resolves its own executable, finds sibling `bin/pair`, and
+execs it with the same argv/env that `pair` would have received directly. This
+makes `pair-go launch claude`, `pair-go launch resume <tag>`, `pair-go launch
+continue ...`, `pair-go launch list`, and `pair-go launch rename ...`
+meaningful dogfood commands without replacing the stable public `pair` command
+yet. A developer shell sourced from `../ariadne/construct/dev-aliases.sh`
+rebuilds `cmd/pair-go` automatically; no `pair-go-dev` command exists.
+
+The earlier #75 pure launcher core remains available as internal decision logic,
+but real zellij lifecycle, prompt/fzf UI, restart/quit cleanup, cmux ownership,
+dev rebuild, continuation, rename, config/session migration, and title-poller
+behavior remain shell-owned through `bin/pair` until later migration issues.
 
 As of #76, the same dispatcher also has the first implemented helper routes:
 `pair-go context <tag> <agent>` and `pair-go scrollback-render ...`. Both routes
diff --git a/atlas/go-migration-inventory.md b/atlas/go-migration-inventory.md
index 4a691ee..5c4c6bc 100644
--- a/atlas/go-migration-inventory.md
+++ b/atlas/go-migration-inventory.md
@@ -47,7 +47,7 @@ Priority is packaging impact first, then reliability/testability:
 
 | Artifact | Type | Callers | Runtime contract | Files/env | Disposition | Priority |
 |---|---|---|---|---|---|---|
-| `bin/pair` / `cmd/internal/launcher` | Bash public launcher plus guarded Go decision prototype | user shell, `bin/pair-dev`, restart re-exec, tests, development-only `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch` (#75) parses a documented subset, snapshots zellij/history state, prints the decision-phase prototype action, and exits before mutating zellij. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/launcher` resolves data-dir inputs and reads zellij/history through fakeable seams. | go-entrypoint with `bin/pair` retained as compat fallback through #77; #75 owns decision-phase only, while real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
+| `bin/pair` / `cmd/internal/launcher` / `cmd/internal/entrypoint` | Bash public launcher plus Go-owned launch handoff | user shell, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` (#77) resolves sibling `bin/pair` from the `pair-go` executable and execs it with `pair`-compatible argv/env. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves the compatibility handoff; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Go handoff entrypoint with `bin/pair` retained as stable public launcher and compat target through #77; real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
 | `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling `pair`. | Resolves symlinks; depends on `bin/pair` and `bin/lib/dev-rebuild.sh`. | compat-shim, likely stays as developer wrapper until Go entrypoint has dev mode | P1 |
 | `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
 | `zellij/layouts/main.kdl` | zellij native asset | `bin/pair` via `zellij --new-session-with-layout` | Defines agent and draft panes; shell expands Pair env at pane start. | Calls `pair-wrap`; calls `nvim -u "$PAIR_HOME/nvim/init.lua"`; writes `pane-<tag>-<agent>.json`; writes draft nvim pid file. | native-asset, packaged adjacent/embedded | P0 |
@@ -129,7 +129,9 @@ Build/install callers:
   `pair-go scrollback-render`, backed by shared internal runners while legacy
   binary names remain live for shell/Lua callers. `pair slug` remains a later
   candidate.
-- #77 is the public entrypoint switch and should wait for both #75 and #76.
+- #77 made `pair-go launch ...` a meaningful Go-owned compatibility handoff to
+  `bin/pair`, with argv/env preserved and missing-launcher diagnostics. The
+  stable public `pair` script remains unchanged for this migration window.
 - #78 should prioritize `pair-title.sh` and `pair-session-watch.sh` if stateful
   shell remains a packaging/reliability problem after #77.
 - #79 owns whether `nvim/` and `zellij/` are embedded or installed adjacent.
diff --git a/cmd/internal/dispatcher/dispatcher.go b/cmd/internal/dispatcher/dispatcher.go
index 3784f1e..fe1fbfd 100644
--- a/cmd/internal/dispatcher/dispatcher.go
+++ b/cmd/internal/dispatcher/dispatcher.go
@@ -3,12 +3,9 @@ package dispatcher
 import (
 	"bytes"
 	"fmt"
-	"os"
 	"strings"
-	"time"
 
 	"github.com/xianxu/pair/cmd/internal/contextcmd"
-	"github.com/xianxu/pair/cmd/internal/launcher"
 	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
 )
 
@@ -31,7 +28,7 @@ type Result struct {
 // Families returns the planned command families for the Go dispatcher.
 func Families() []CommandFamily {
 	return []CommandFamily{
-		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "prototype"},
+		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "handoff"},
 		{Name: "context", Summary: "agent pane context meter", Status: "implemented"},
 		{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "implemented"},
 		{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
@@ -53,11 +50,11 @@ func Dispatch(args []string) Result {
 		return Result{Stdout: Help(programName), ExitCode: 0}
 	case "version", "--version":
 		return Result{
-			Stdout:   "pair-go dispatcher skeleton\npublic launcher: bin/pair\n",
+			Stdout:   "pair-go dispatcher skeleton\nlaunch handoff: bin/pair\n",
 			ExitCode: 0,
 		}
 	case "launch":
-		return DispatchWithLauncherRuntime(args, osLauncherRuntime())
+		return launchHandoffResult()
 	case "context":
 		return dispatchContext(args[1:])
 	case "scrollback-render":
@@ -89,90 +86,10 @@ func dispatchScrollbackRender(args []string) Result {
 	return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}
 }
 
-type LauncherRuntime struct {
-	Env      launcher.Env
-	Sessions launcher.SessionSource
-	History  launcher.HistoricalScanner
-}
-
-type StaticSessions struct {
-	Sessions []launcher.Session
-	Err      error
-}
-
-func (s StaticSessions) Snapshot() ([]launcher.Session, error) {
-	if s.Err != nil {
-		return nil, s.Err
-	}
-	return s.Sessions, nil
-}
-
-type StaticHistory struct {
-	Tags []launcher.HistoricalTag
-	Err  error
-}
-
-func (h StaticHistory) Scan(_ string, _ time.Time) ([]launcher.HistoricalTag, error) {
-	if h.Err != nil {
-		return nil, h.Err
-	}
-	return h.Tags, nil
-}
-
-func DispatchWithLauncherRuntime(args []string, rt LauncherRuntime) Result {
-	launchArgs := []string(nil)
-	if len(args) > 1 {
-		launchArgs = args[1:]
-	}
-	if len(launchArgs) > 0 && (launchArgs[0] == "help" || launchArgs[0] == "--help" || launchArgs[0] == "-h") {
-		return Result{Stdout: LaunchHelp(programName), ExitCode: 0}
-	}
-	outcome, err := launcher.Run(launchArgs, rt.Env, rt.Sessions, rt.History)
-	if err != nil {
-		return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
-	}
-	decision := outcome.Decision
+func launchHandoffResult() Result {
 	return Result{
-		Stderr: fmt.Sprintf(
-			"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
-			decision.Action,
-			decision.Tag,
-			decision.SessionName,
-		),
-		ExitCode: 3,
-	}
-}
-
-func LaunchHelp(program string) string {
-	return fmt.Sprintf(`Usage: %s launch [agent] [-- agent-args...]
-       %s launch resume <tag>
-
-Guarded decision-phase prototype. Public sessions still start through bin/pair.
-This command parses launch inputs and computes the create/attach/picker decision,
-then stops before invoking zellij.
-`, program, program)
-}
-
-func LauncherEnv(home, xdgDataHome, cwd string) launcher.Env {
-	return launcher.Env{
-		Home:     home,
-		XDGData:  xdgDataHome,
-		Cwd:      cwd,
-		Now:      time.Now(),
-		HistoryD: 14,
-	}
-}
-
-func osLauncherRuntime() LauncherRuntime {
-	home := os.Getenv("HOME")
-	xdg := os.Getenv("XDG_DATA_HOME")
-	cwd, _ := os.Getwd()
-	env := LauncherEnv(home, xdg, cwd)
-	dataDir := launcher.ResolveDataDir(home, xdg)
-	return LauncherRuntime{
-		Env:      env,
-		Sessions: launcher.ZellijSource{},
-		History:  launcher.HistorySource{DataDir: dataDir},
+		Stderr:   "pair-go launch is a process handoff implemented by cmd/pair-go; call pair-go launch ... instead of dispatcher.Dispatch\n",
+		ExitCode: 2,
 	}
 }
 
@@ -185,6 +102,8 @@ func Help(program string) string {
 	for _, family := range Families() {
 		if family.Status == "prototype" {
 			fmt.Fprintf(&b, "  %-17s %s (prototype; decision-phase only)\n", family.Name, family.Summary)
+		} else if family.Status == "handoff" {
+			fmt.Fprintf(&b, "  %-17s %s (compatibility handoff to bin/pair)\n", family.Name, family.Summary)
 		} else if family.Status == "implemented" {
 			fmt.Fprintf(&b, "  %-17s %s (implemented helper route)\n", family.Name, family.Summary)
 		}
diff --git a/cmd/internal/dispatcher/dispatcher_test.go b/cmd/internal/dispatcher/dispatcher_test.go
index f10f32b..7dc3021 100644
--- a/cmd/internal/dispatcher/dispatcher_test.go
+++ b/cmd/internal/dispatcher/dispatcher_test.go
@@ -21,7 +21,7 @@ func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
 				"Usage: pair-go <command> [args]",
 				"Implemented commands:",
 				"launch",
-				"decision-phase only",
+				"compatibility handoff",
 				"context",
 				"scrollback-render",
 				"wrap",
@@ -35,6 +35,9 @@ func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
 			if strings.Contains(res.Stdout, "launch             session lifecycle and public pair launcher flow (planned; not implemented") {
 				t.Fatalf("Stdout still labels launch unimplemented:\n%s", res.Stdout)
 			}
+			if strings.Contains(res.Stdout, "decision-phase only") {
+				t.Fatalf("Stdout still labels launch decision-phase only:\n%s", res.Stdout)
+			}
 			for _, stale := range []string{
 				"context           agent pane context meter (planned; not implemented",
 				"scrollback-render raw PTY capture to ANSI scrollback (planned; not implemented",
@@ -55,7 +58,7 @@ func TestDispatchVersionIsDevelopmentSkeletonMetadata(t *testing.T) {
 	if res.Stderr != "" {
 		t.Fatalf("Stderr = %q, want empty", res.Stderr)
 	}
-	for _, want := range []string{"pair-go", "dispatcher skeleton", "public launcher: bin/pair"} {
+	for _, want := range []string{"pair-go", "dispatcher skeleton", "launch handoff: bin/pair"} {
 		if !strings.Contains(res.Stdout, want) {
 			t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
 		}
@@ -77,57 +80,15 @@ func TestDispatchPlannedCommandReturnsUnsupported(t *testing.T) {
 	}
 }
 
-func TestDispatchLaunchHelpRoutesToPrototype(t *testing.T) {
+func TestDispatchLaunchReportsProcessHandoff(t *testing.T) {
 	res := Dispatch([]string{"launch", "--help"})
-	if res.ExitCode != 0 {
-		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
-	}
-	if res.Stderr != "" {
-		t.Fatalf("Stderr = %q, want empty", res.Stderr)
-	}
-	for _, want := range []string{"Usage: pair-go launch", "decision-phase prototype"} {
-		if !strings.Contains(res.Stdout, want) {
-			t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
-		}
-	}
-}
-
-func TestDispatchLaunchReturnsPrototypeDecision(t *testing.T) {
-	res := DispatchWithLauncherRuntime([]string{"launch", "resume", "demo"}, LauncherRuntime{
-		Env: LauncherEnv("/home/me", "", "/work/pair"),
-		Sessions: StaticSessions{
-			Sessions: nil,
-		},
-		History: StaticHistory{},
-	})
-	if res.ExitCode != 3 {
-		t.Fatalf("ExitCode = %d, want 3", res.ExitCode)
-	}
-	if res.Stdout != "" {
-		t.Fatalf("Stdout = %q, want empty", res.Stdout)
-	}
-	for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
-		if !strings.Contains(res.Stderr, want) {
-			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
-		}
-	}
-}
-
-func TestDispatchLaunchWithoutArgsReturnsDefaultPrototypeDecision(t *testing.T) {
-	res := DispatchWithLauncherRuntime([]string{"launch"}, LauncherRuntime{
-		Env: LauncherEnv("/home/me", "", "/work/pair"),
-		Sessions: StaticSessions{
-			Sessions: nil,
-		},
-		History: StaticHistory{},
-	})
-	if res.ExitCode != 3 {
-		t.Fatalf("ExitCode = %d, want 3", res.ExitCode)
+	if res.ExitCode != 2 {
+		t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
 	}
 	if res.Stdout != "" {
 		t.Fatalf("Stdout = %q, want empty", res.Stdout)
 	}
-	for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=pair", "session=pair-pair"} {
+	for _, want := range []string{"pair-go launch", "process handoff", "cmd/pair-go"} {
 		if !strings.Contains(res.Stderr, want) {
 			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
 		}
diff --git a/cmd/internal/entrypoint/launch.go b/cmd/internal/entrypoint/launch.go
new file mode 100644
index 0000000..1a40970
--- /dev/null
+++ b/cmd/internal/entrypoint/launch.go
@@ -0,0 +1,20 @@
+package entrypoint
+
+import "path/filepath"
+
+// LegacyLaunchRequest describes the current compatibility handoff from
+// pair-go launch to the shell-owned pair launcher.
+type LegacyLaunchRequest struct {
+	Path string
+	Argv []string
+}
+
+func ResolveLegacyLaunch(executable string, launchArgs []string) LegacyLaunchRequest {
+	argv := make([]string, 0, len(launchArgs)+1)
+	argv = append(argv, "pair")
+	argv = append(argv, launchArgs...)
+	return LegacyLaunchRequest{
+		Path: filepath.Join(filepath.Dir(executable), "pair"),
+		Argv: argv,
+	}
+}
diff --git a/cmd/internal/entrypoint/launch_test.go b/cmd/internal/entrypoint/launch_test.go
new file mode 100644
index 0000000..cdeec2c
--- /dev/null
+++ b/cmd/internal/entrypoint/launch_test.go
@@ -0,0 +1,28 @@
+package entrypoint
+
+import (
+	"reflect"
+	"testing"
+)
+
+func TestResolveLegacyLaunchDropsLaunchVerb(t *testing.T) {
+	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"claude", "--", "--resume"})
+	if req.Path != "/repo/bin/pair" {
+		t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
+	}
+	want := []string{"pair", "claude", "--", "--resume"}
+	if !reflect.DeepEqual(req.Argv, want) {
+		t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
+	}
+}
+
+func TestResolveLegacyLaunchPreservesSubcommands(t *testing.T) {
+	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"resume", "demo"})
+	if req.Path != "/repo/bin/pair" {
+		t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
+	}
+	want := []string{"pair", "resume", "demo"}
+	if !reflect.DeepEqual(req.Argv, want) {
+		t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
+	}
+}
diff --git a/cmd/pair-go/launch_process_test.go b/cmd/pair-go/launch_process_test.go
deleted file mode 100644
index 9563492..0000000
--- a/cmd/pair-go/launch_process_test.go
+++ /dev/null
@@ -1,71 +0,0 @@
-package main
-
-import (
-	"bytes"
-	"os"
-	"path/filepath"
-	"strings"
-	"testing"
-	"time"
-)
-
-func TestRunLaunchWithFakeZellij(t *testing.T) {
-	rt := t.TempDir()
-	bin := filepath.Join(rt, "bin")
-	data := filepath.Join(rt, "data")
-	if err := os.MkdirAll(bin, 0o755); err != nil {
-		t.Fatal(err)
-	}
-	if err := os.MkdirAll(filepath.Join(data, "pair"), 0o755); err != nil {
-		t.Fatal(err)
-	}
-	log := filepath.Join(rt, "zellij.log")
-	zellij := filepath.Join(bin, "zellij")
-	script := `#!/usr/bin/env bash
-printf '%s\n' "$*" >> "` + log + `"
-case "$*" in
-  "list-sessions --short") printf 'pair-live\npair-detached\npair-exited\n' ;;
-  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-exited [Created] (EXITED - attach to resurrect)\n' ;;
-  "--session pair-live action list-clients") printf 'CLIENTS\n1\n' ;;
-  "--session pair-detached action list-clients") printf 'CLIENTS\n' ;;
-  *attach*|*new-session*|*--new-session-with-layout*|*delete-session*) printf 'MUTATING %s\n' "$*" >> "` + log + `"; exit 99 ;;
-  *) exit 1 ;;
-esac
-`
-	if err := os.WriteFile(zellij, []byte(script), 0o755); err != nil {
-		t.Fatal(err)
-	}
-	historical := filepath.Join(data, "pair", "draft-pair-old.md")
-	if err := os.WriteFile(historical, []byte("draft"), 0o644); err != nil {
-		t.Fatal(err)
-	}
-	now := time.Now()
-	if err := os.Chtimes(historical, now, now); err != nil {
-		t.Fatal(err)
-	}
-
-	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
-	t.Setenv("HOME", filepath.Join(rt, "home"))
-	t.Setenv("XDG_DATA_HOME", data)
-
-	var stdout, stderr bytes.Buffer
-	code := run([]string{"launch", "claude"}, &stdout, &stderr)
-	if code != 3 {
-		t.Fatalf("code = %d, want 3; stderr:\n%s", code, stderr.String())
-	}
-	if stdout.String() != "" {
-		t.Fatalf("stdout = %q, want empty", stdout.String())
-	}
-	for _, want := range []string{"prototype decision", "action=pick"} {
-		if !strings.Contains(stderr.String(), want) {
-			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
-		}
-	}
-	logBytes, err := os.ReadFile(log)
-	if err != nil {
-		t.Fatal(err)
-	}
-	if strings.Contains(string(logBytes), "MUTATING") {
-		t.Fatalf("fake zellij recorded mutating invocation:\n%s", string(logBytes))
-	}
-}
diff --git a/cmd/pair-go/main.go b/cmd/pair-go/main.go
index bf7d28b..760f876 100644
--- a/cmd/pair-go/main.go
+++ b/cmd/pair-go/main.go
@@ -1,12 +1,15 @@
-// pair-go is a development-only dispatcher skeleton for the future primary Go
-// CLI. The public launcher remains bin/pair.
+// pair-go is the development dispatcher for the future primary Go CLI. Its
+// launch route is a compatibility handoff to the current shell launcher.
 package main
 
 import (
+	"fmt"
 	"io"
 	"os"
+	"syscall"
 
 	"github.com/xianxu/pair/cmd/internal/dispatcher"
+	"github.com/xianxu/pair/cmd/internal/entrypoint"
 )
 
 func main() {
@@ -14,13 +17,65 @@ func main() {
 }
 
 func run(args []string, stdout, stderr io.Writer) int {
+	return runWithLegacyRuntime(args, stdout, stderr, osLegacyRuntime{})
+}
+
+type legacyRuntime interface {
+	Executable() (string, error)
+	Stat(path string) error
+	Environ() []string
+	Exec(path string, argv []string, env []string) int
+}
+
+func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
+	if len(args) > 0 && args[0] == "launch" {
+		return runLegacyLaunch(args[1:], stderr, rt)
+	}
 	res := dispatcher.Dispatch(args)
 	return writeResult(res, stdout, stderr)
 }
 
-func runWithLauncherRuntime(args []string, stdout, stderr io.Writer, rt dispatcher.LauncherRuntime) int {
-	res := dispatcher.DispatchWithLauncherRuntime(args, rt)
-	return writeResult(res, stdout, stderr)
+func runLegacyLaunch(args []string, stderr io.Writer, rt legacyRuntime) int {
+	exe, err := rt.Executable()
+	if err != nil {
+		_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
+		return 1
+	}
+	req := entrypoint.ResolveLegacyLaunch(exe, args)
+	if err := rt.Stat(req.Path); err != nil {
+		_, _ = fmt.Fprintf(stderr, "pair-go launch: pair launcher not found at %s (%v); run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", req.Path, err)
+		return 1
+	}
+	return rt.Exec(req.Path, req.Argv, rt.Environ())
+}
+
+type osLegacyRuntime struct{}
+
+func (osLegacyRuntime) Executable() (string, error) {
+	return os.Executable()
+}
+
+func (osLegacyRuntime) Stat(path string) error {
+	info, err := os.Stat(path)
+	if err != nil {
+		return err
+	}
+	if info.IsDir() {
+		return fmt.Errorf("is a directory")
+	}
+	return nil
+}
+
+func (osLegacyRuntime) Environ() []string {
+	return os.Environ()
+}
+
+func (osLegacyRuntime) Exec(path string, argv []string, env []string) int {
+	if err := syscall.Exec(path, argv, env); err != nil {
+		_, _ = fmt.Fprintf(os.Stderr, "pair-go launch: exec %s failed: %v\n", path, err)
+		return 1
+	}
+	return 0
 }
 
 func writeResult(res dispatcher.Result, stdout, stderr io.Writer) int {
diff --git a/cmd/pair-go/main_test.go b/cmd/pair-go/main_test.go
index b0d6f7f..d137a6b 100644
--- a/cmd/pair-go/main_test.go
+++ b/cmd/pair-go/main_test.go
@@ -2,10 +2,11 @@ package main
 
 import (
 	"bytes"
+	"errors"
+	"os"
+	"reflect"
 	"strings"
 	"testing"
-
-	"github.com/xianxu/pair/cmd/internal/dispatcher"
 )
 
 func TestRunWritesStdoutAndReturnsDispatcherCode(t *testing.T) {
@@ -37,55 +38,122 @@ func TestRunWritesStderrAndReturnsDispatcherCode(t *testing.T) {
 }
 
 func TestRunLaunchHelp(t *testing.T) {
+	rt := &fakeLegacyRuntime{
+		executable: "/repo/bin/pair-go",
+	}
 	var stdout, stderr bytes.Buffer
-	code := run([]string{"launch", "--help"}, &stdout, &stderr)
+	code := runWithLegacyRuntime([]string{"launch", "--help"}, &stdout, &stderr, rt)
 	if code != 0 {
 		t.Fatalf("code = %d, want 0", code)
 	}
-	if !strings.Contains(stdout.String(), "Usage: pair-go launch") {
-		t.Fatalf("stdout missing launch usage:\n%s", stdout.String())
+	if stdout.String() != "" {
+		t.Fatalf("stdout = %q, want empty", stdout.String())
 	}
 	if stderr.String() != "" {
 		t.Fatalf("stderr = %q, want empty", stderr.String())
 	}
+	if rt.execPath != "/repo/bin/pair" {
+		t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
+	}
+	wantArgv := []string{"pair", "--help"}
+	if !reflect.DeepEqual(rt.execArgv, wantArgv) {
+		t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
+	}
 }
 
-func TestRunLaunchResumeReturnsPrototypeDecision(t *testing.T) {
+func TestRunLaunchExecsLegacyPairWithArgvAndEnv(t *testing.T) {
+	t.Setenv("PAIR_TEST_ENV", "kept")
+	rt := &fakeLegacyRuntime{
+		executable: "/repo/bin/pair-go",
+		execCode:   42,
+	}
+
 	var stdout, stderr bytes.Buffer
-	code := runWithLauncherRuntime([]string{"launch", "resume", "demo"}, &stdout, &stderr, testLauncherRuntime("/home/me", "", "/work/pair"))
-	if code != 3 {
-		t.Fatalf("code = %d, want 3", code)
+	code := runWithLegacyRuntime([]string{"launch", "claude", "--", "--resume"}, &stdout, &stderr, rt)
+
+	if code != 42 {
+		t.Fatalf("code = %d, want 42", code)
 	}
 	if stdout.String() != "" {
 		t.Fatalf("stdout = %q, want empty", stdout.String())
 	}
-	for _, want := range []string{"prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
-		if !strings.Contains(stderr.String(), want) {
-			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
-		}
+	if stderr.String() != "" {
+		t.Fatalf("stderr = %q, want empty", stderr.String())
+	}
+	if rt.execPath != "/repo/bin/pair" {
+		t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
+	}
+	wantArgv := []string{"pair", "claude", "--", "--resume"}
+	if !reflect.DeepEqual(rt.execArgv, wantArgv) {
+		t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
+	}
+	if !containsEnv(rt.execEnv, "PAIR_TEST_ENV=kept") {
+		t.Fatalf("execEnv missing PAIR_TEST_ENV=kept: %#v", rt.execEnv)
 	}
 }
 
-func TestRunLaunchWithoutArgsReturnsDefaultPrototypeDecision(t *testing.T) {
+func TestRunLaunchReportsMissingLegacyPair(t *testing.T) {
+	rt := &fakeLegacyRuntime{
+		executable: "/repo/bin/pair-go",
+		statErr:    os.ErrNotExist,
+	}
+
 	var stdout, stderr bytes.Buffer
-	code := runWithLauncherRuntime([]string{"launch"}, &stdout, &stderr, testLauncherRuntime("/home/me", "", "/work/pair"))
-	if code != 3 {
-		t.Fatalf("code = %d, want 3", code)
+	code := runWithLegacyRuntime([]string{"launch", "claude"}, &stdout, &stderr, rt)
+
+	if code != 1 {
+		t.Fatalf("code = %d, want 1", code)
 	}
 	if stdout.String() != "" {
 		t.Fatalf("stdout = %q, want empty", stdout.String())
 	}
-	for _, want := range []string{"prototype decision", "action=create", "tag=pair", "session=pair-pair"} {
+	for _, want := range []string{"pair-go launch", "/repo/bin/pair", "make build", "make install", "dev-aliases.sh"} {
 		if !strings.Contains(stderr.String(), want) {
 			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
 		}
 	}
+	if rt.execPath != "" {
+		t.Fatalf("execPath = %q, want empty", rt.execPath)
+	}
 }
 
-func testLauncherRuntime(home, xdg, cwd string) dispatcher.LauncherRuntime {
-	return dispatcher.LauncherRuntime{
-		Env:      dispatcher.LauncherEnv(home, xdg, cwd),
-		Sessions: dispatcher.StaticSessions{},
-		History:  dispatcher.StaticHistory{},
+type fakeLegacyRuntime struct {
+	executable string
+	statErr    error
+	execCode   int
+
+	execPath string
+	execArgv []string
+	execEnv  []string
+}
+
+func (f *fakeLegacyRuntime) Executable() (string, error) {
+	if f.executable == "" {
+		return "", errors.New("missing executable")
+	}
+	return f.executable, nil
+}
+
+func (f *fakeLegacyRuntime) Stat(_ string) error {
+	return f.statErr
+}
+
+func (f *fakeLegacyRuntime) Environ() []string {
+	return os.Environ()
+}
+
+func (f *fakeLegacyRuntime) Exec(path string, argv []string, env []string) int {
+	f.execPath = path
+	f.execArgv = append([]string(nil), argv...)
+	f.execEnv = append([]string(nil), env...)
+	return f.execCode
+}
+
+func containsEnv(env []string, want string) bool {
+	for _, got := range env {
+		if got == want {
+			return true
+		}
 	}
+	return false
 }
diff --git a/tests/pair-go-install-layout-test.sh b/tests/pair-go-install-layout-test.sh
new file mode 100644
index 0000000..086cb5b
--- /dev/null
+++ b/tests/pair-go-install-layout-test.sh
@@ -0,0 +1,26 @@
+#!/usr/bin/env bash
+set -euo pipefail
+
+repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
+tmp_home="$(mktemp -d "${TMPDIR:-/tmp}/pair-go-install-layout.XXXXXX")"
+trap 'rm -rf "$tmp_home"' EXIT
+gomodcache="$(go env GOMODCACHE)"
+gocache="$(go env GOCACHE)"
+
+HOME="$tmp_home" GOMODCACHE="$gomodcache" GOCACHE="$gocache" make -C "$repo_root" install >/dev/null
+
+install_bin="$tmp_home/.local/bin"
+test -x "$install_bin/pair-go"
+test -L "$install_bin/pair"
+test -L "$install_bin/pair-dev"
+
+out="$("$install_bin/pair-go" launch --help)"
+case "$out" in
+    pair\ —*) ;;
+    *)
+        printf 'pair-go launch --help did not reach pair help; first bytes:\n%s\n' "$out" >&2
+        exit 1
+        ;;
+esac
+
+printf 'pair-go install layout test passed\n'
diff --git a/workshop/plans/000077-go-entrypoint-switch-close-review.md b/workshop/plans/000077-go-entrypoint-switch-close-review.md
new file mode 100644
index 0000000..b1a58b1
--- /dev/null
+++ b/workshop/plans/000077-go-entrypoint-switch-close-review.md
@@ -0,0 +1,3581 @@
+# Boundary Review — pair#77 (whole-issue close)
+
+| field | value |
+|-------|-------|
+| issue | 77 — pair Go entrypoint switch |
+| repo | pair |
+| issue file | workshop/issues/000077-go-entrypoint-switch.md |
+| boundary | whole-issue close |
+| milestone | — |
+| window | b829ef00346738e3c3d7e40db38c2c03f51e615b..HEAD |
+| command | sdlc close --issue 77 |
+| reviewer | codex |
+| timestamp | 2026-06-30T15:32:46-07:00 |
+| verdict | REWORK |
+
+## Review
+
+Reading additional input from stdin...
+OpenAI Codex v0.142.4
+--------
+workdir: /Users/xianxu/workspace/pair
+model: gpt-5.5
+provider: openai
+approval: never
+sandbox: workspace-write [workdir, /tmp, $TMPDIR, /tmp] (network access enabled)
+reasoning effort: none
+reasoning summaries: none
+session id: 019f1aa7-cafd-7b23-acc6-61ab6438b466
+--------
+user
+# Code review — the one SDLC boundary review
+
+You are conducting a fresh-context code review at a development boundary —
+whole-issue close — in the **pair** repository.
+
+- repository: pair   (root: /Users/xianxu/workspace/pair)
+- issue:      pair#77   (file: workshop/issues/000077-go-entrypoint-switch.md)
+- window:     Base: b829ef00346738e3c3d7e40db38c2c03f51e615b   Head: HEAD
+
+Review the **pair** repo and its tracker — the ariadne base-layer repo itself (changes here propagate to dependent repos). Do not assume any
+other repository or apply another repo's conventions.
+
+You have no prior session context — that is the anti-collusion property. Verify
+behavior against the issue's documented Spec/Plan and the code itself; do NOT
+take the implementor's word in commit messages or docs at face value. Tools are
+read-only: report findings precisely; the main agent (which has session context)
+applies the fixes, commits, and re-runs.
+
+Read the diff against the issue's Spec + Plan, then work the checklist below.
+Categorize every finding by severity — not everything is Critical; a nitpick
+marked Critical is noise.
+
+  Critical (must fix before crossing the boundary)
+    - correctness bugs; crashes / panics on unexpected input
+    - behavior drift from stated contracts (for ports of existing code where
+      byte-faithfulness was promised, diff against the source)
+    - silent error swallowing where the source raised
+  Important (fix before the boundary if cheap)
+    - API design of newly-introduced internal packages (downstream work will
+      consume them; is the surface stable?)
+    - missing test coverage that would catch the kind of bug shipped
+    - inconsistent error handling across the diff
+  Minor (note for future)
+    - style nits, naming, comment density; performance only if hot-path
+
+## Review checklist
+
+Code quality
+  - Clean separation of concerns; edge cases handled (empty / nil / unexpected).
+  - Proper error handling — no silent swallowing where the source raised.
+  - No duplicated logic / copy-paste that should be a shared helper.
+
+Testing
+  - Tests pin real logic, not mocks reasserting the implementation.
+  - The kind of bug this diff could ship is covered.
+  - PURE entities tested without IO; INTEGRATION via injected fakes (see below).
+
+Requirements traceability
+  - Every Plan checklist item this boundary claims is actually delivered.
+  - Implementation matches the Spec; no undeclared scope creep.
+  - Breaking changes documented.
+
+Production readiness
+  - Migration / backward-compatibility considered where state or formats change.
+  - Docs / atlas updated for new surface (see the Atlas update gate).
+
+## Core concepts cross-check (if the plan has a Core concepts table)
+
+The plan should list entities in a greppable table — name, kind
+(PURE/INTEGRATION), file location, status (new/modified/deleted). For each row:
+  - Verify the entity exists at the stated path (grep the diff or filesystem).
+  - PURE: tests run without IO (no exec, net, mutable fs). If tests need mocks
+    to run, it isn't really PURE — flag Critical and recommend promoting it to
+    INTEGRATION.
+  - INTEGRATION: injected into pure callers, not invoked directly from business
+    logic.
+  - "modified" / "deleted": the diff shows the expected change/removal at the
+    stated location.
+Any contradiction between table and code = Critical finding, plus a plan-revision
+recommendation (a "## Revisions" entry so the plan stops claiming what the code
+doesn't deliver).
+
+## Atlas update gate (per AGENTS.md §8)
+
+The boundary should update atlas/ entries for any new architectural surface,
+flow, or terminology introduced. Scan the diff for evidence of new surface — new
+entity types, new subcommands, new conventions, new file-tree locations. Any
+present without corresponding atlas/ changes in the same range = Important
+finding ("atlas update appears missing for <surface>").
+
+## Architecture (the at-review backstop — these matter most long-term)
+
+Work through each of ARCH-DRY, ARCH-PURE, ARCH-PURPOSE explicitly, applying its at-review lens. The
+full principle definitions are delivered in the ARCHITECTURE PRINCIPLES block
+right after this prompt — for EACH marker, state pass or flag, and cite the
+marker (e.g. ARCH-DRY) in any finding. Architecture is where review has the
+least training signal and the longest-delayed payoff, so be deliberate here, not
+holistic.
+
+## Verdict + output
+
+Your FIRST line is the VERDICT (format in the output contract below). Tokens:
+  SHIP          = ready; ship it.
+  FIX-THEN-SHIP = ship after addressing the findings (non-blocking at the gate;
+                  fix them before the next boundary).
+  REWORK        = blocking; needs rework before shipping.
+
+After the VERDICT line: a 1-paragraph summary — what worked, what blocks SHIP if
+it isn't — followed by:
+  1. Strengths: 2-5 specific things done well (file:line where useful). Affirm
+     validated approaches so the operator knows what's confirmed-good ground.
+     Empty acceptable for trivial boundaries.
+  2. Critical findings (file:line + fix sketch); empty if none.
+  3. Important findings (same format).
+  4. Minor findings (terse one-liners).
+  5. Test coverage notes.
+  6. Architectural notes for upcoming work.
+  7. Plan revision recommendations: specific "## Revisions" entries the plan
+     needs (empty if the plan still matches the code).
+
+
+ARCHITECTURE PRINCIPLES — work through each of the 3 entries below explicitly, applying its `at-review` lens; cite the marker (e.g. ARCH-DRY) in any finding.
+
+# Architecture principles (ARCH-*)
+
+Injected architectural taste — the structural decisions whose payoff (or cost)
+shows up many turns, often months, down the road. Agents are strong at local
+tactics and weak here, so these are checked **at-plan** (when the design is being
+made — highest leverage) and **at-review** (backstop, on the diff). Cite the
+marker (e.g. `ARCH-DRY`) in plans, `## Log` entries, and review findings.
+
+This file is the single source; it is embedded into the planning, plan-quality,
+and code-review prompts. The human narrative lives in AGENTS.md "Core Design
+Principles"; this is its machine-delivered companion.
+
+## ARCH-DRY — Don't Repeat Yourself
+
+- **principle:** Reuse before adding. One source of truth per fact/behavior; no
+  duplicated logic, copy-pasted blocks, or parallel functions that should be one
+  shared helper.
+- **at-plan:** Flag a plan that re-implements something the codebase already has,
+  or that will obviously duplicate logic across the new files instead of
+  extracting a shared helper. Name the existing thing it should reuse.
+- **at-review:** Flag duplicated logic / copy-pasted blocks / near-identical
+  functions in the diff; point at the consolidation (file:line + the shared
+  helper they should become).
+
+## ARCH-PURE — Pure core, thin IO shell
+
+- **principle:** The majority of code is pure functions (deterministic, no side
+  effects); a thin "glue" layer at the boundary touches IO/UI/network/clock. Pure
+  functions are unit-tested directly; the glue is kept small and injected.
+- **at-plan:** Flag a design that buries business logic inside IO/handlers, or
+  that will only be testable with heavy mocks (a sign logic isn't separated from
+  IO). The plan should name what's pure vs the thin IO seam.
+- **at-review:** Flag business logic mixed with IO in the diff; logic that should
+  be a pure function injected into a thin caller. If a test needs mocks to run a
+  "pure" entity, it isn't pure — recommend extracting the IO to the boundary.
+
+## ARCH-PURPOSE — Serve the issue's actual purpose
+
+- **principle:** Deliver the issue's stated purpose, not the easy subset of it. A
+  single-source / "compiled to consumers" change is not done until **every
+  consumer derives** from the source — the source is *enforced*, not just
+  documentation a surface happens to restate; a hand-maintained restatement of the
+  model is a deferred consumer, not a finished one. "Follow-up" is for separable
+  extensions, never for the thing that is the point. This is the *opposite axis*
+  from Simplicity-First/YAGNI: not "build for an imagined future," but "don't
+  **under**-deliver the purpose you already committed to."
+- **at-plan:** Flag a plan whose scope is a strict subset of the issue's stated
+  goal / Done-when where the part deferred as "follow-up" *is* the purpose (e.g.
+  wires one consumer + enforcement but leaves the consumers that motivated the
+  issue as documentation that doesn't derive). Ask: does the plan fulfill the
+  purpose, or just the cheap win? Name the deferred purpose.
+- **at-review:** Does the diff *fulfill* the purpose or settle for the easy win?
+  For a single-source change, run the **shadow-sweep** — enumerate the consumers,
+  confirm each derives from the source, flag any remaining hand-maintained
+  restatement of the model. A "follow-up" that is actually the deferred point of
+  the issue is a finding, not a deferral.
+
+
+OUTPUT CONTRACT (machine-read — do not deviate). Your response's FIRST line
+MUST be exactly:
+
+    VERDICT: <TOKEN> (confidence: high | medium | low)
+
+The parser reads ONLY this <TOKEN>. Findings, notes, and severity tags below it
+are advisory — a non-blocking verdict WITH notes still PASSES the gate. Do not
+put a title, heading, or any preamble above the VERDICT line; it must lead.
+
+Diff:
+diff --git a/Makefile.local b/Makefile.local
+index ccdde68..6bbc2eb 100644
+--- a/Makefile.local
++++ b/Makefile.local
+@@ -239,9 +239,10 @@ $(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill
+ $(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-continuation/git.go cmd/pair-continuation/main.go go.mod
+ 	go build -o $@ ./cmd/pair-continuation
+ 
+-# pair-go — development-only dispatcher skeleton for the future primary Go CLI.
+-# The public launcher remains bin/pair until the entrypoint-switch issue.
+-$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
++# pair-go — development dispatcher and Go-owned launch handoff.
++# `pair-go launch ...` execs sibling bin/pair for the current migration window;
++# helper routes stay in the dispatcher.
++$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/entrypoint/launch.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
+ 	go build -o $@ ./cmd/pair-go
+ 
+ 
+diff --git a/README.md b/README.md
+index c4681a3..dcde60a 100644
+--- a/README.md
++++ b/README.md
+@@ -200,6 +200,13 @@ pair rename <old> <new>          # rename every tag-scoped file in
+ pair -h, --help                  # show full help
+ ```
+ 
++`pair-go launch ...` is the Go-owned migration entrypoint for testing the
++launcher path. It accepts the same arguments after `launch` that `pair` accepts
++directly, then hands off to the current `pair` launcher for one migration
++window. In a dev shell sourced from `../ariadne/construct/dev-aliases.sh`,
++`pair-go` rebuilds from `cmd/pair-go` automatically before running; no
++`pair-go-dev` command is needed.
++
+ Use `--` to separate pair's positional from agent flags. Without it, pair only takes `<agent>` as a positional and everything else is rejected.
+ 
+ Agent args (after `--`) are appended to the agent command line on **create**. Reattaching to an existing session does not re-launch the agent, so the args don't apply on attach. (The picker connects you to whatever's already running.)
+diff --git a/atlas/architecture.md b/atlas/architecture.md
+index 5e308ab..c8df19e 100644
+--- a/atlas/architecture.md
++++ b/atlas/architecture.md
+@@ -14,7 +14,7 @@ contract for the Go packaging migration lives in
+ 
+ ```
+ bin/pair                     # entry point (launcher)
+-bin/pair-go                  # opt-in Go dispatcher skeleton; public launcher remains bin/pair
++bin/pair-go                  # opt-in Go dispatcher + launch handoff; public launcher remains bin/pair
+ bin/clipboard-to-pane.sh     # read clipboard, hand off to nvim's PairPasteQuote
+ bin/copy-on-select.sh        # invoked by zellij copy_command on mouse-up
+ bin/pair-quit.sh             # invoked by Alt+x — marks + kills session
+@@ -45,15 +45,19 @@ internal subcommands or dispatch modes behind that primary binary (`pair wrap`,
+ `pair continuation`, `pair scribe`) instead of staying as independently managed
+ installed commands forever.
+ 
+-As of #75, `bin/pair-go` is a development-only dispatcher with a guarded
+-`pair-go launch` decision-phase prototype. The public launcher remains
+-`bin/pair`; `pair-go launch` parses the launcher subset, resolves the Pair data
+-dir, snapshots fakeable zellij/history state, and computes create/attach/picker
+-decisions through the pure `cmd/internal/launcher` core. It then stops with an
+-explicit prototype message before invoking zellij. Real zellij lifecycle,
+-prompt/fzf UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation,
+-rename, config/session migration, and title-poller behavior remain shell-owned
+-until later migration issues.
++As of #77, `bin/pair-go` is the Go-owned launch entrypoint under test:
++`pair-go launch ...` resolves its own executable, finds sibling `bin/pair`, and
++execs it with the same argv/env that `pair` would have received directly. This
++makes `pair-go launch claude`, `pair-go launch resume <tag>`, `pair-go launch
++continue ...`, `pair-go launch list`, and `pair-go launch rename ...`
++meaningful dogfood commands without replacing the stable public `pair` command
++yet. A developer shell sourced from `../ariadne/construct/dev-aliases.sh`
++rebuilds `cmd/pair-go` automatically; no `pair-go-dev` command exists.
++
++The earlier #75 pure launcher core remains available as internal decision logic,
++but real zellij lifecycle, prompt/fzf UI, restart/quit cleanup, cmux ownership,
++dev rebuild, continuation, rename, config/session migration, and title-poller
++behavior remain shell-owned through `bin/pair` until later migration issues.
+ 
+ As of #76, the same dispatcher also has the first implemented helper routes:
+ `pair-go context <tag> <agent>` and `pair-go scrollback-render ...`. Both routes
+diff --git a/atlas/go-migration-inventory.md b/atlas/go-migration-inventory.md
+index 4a691ee..5c4c6bc 100644
+--- a/atlas/go-migration-inventory.md
++++ b/atlas/go-migration-inventory.md
+@@ -47,7 +47,7 @@ Priority is packaging impact first, then reliability/testability:
+ 
+ | Artifact | Type | Callers | Runtime contract | Files/env | Disposition | Priority |
+ |---|---|---|---|---|---|---|
+-| `bin/pair` / `cmd/internal/launcher` | Bash public launcher plus guarded Go decision prototype | user shell, `bin/pair-dev`, restart re-exec, tests, development-only `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch` (#75) parses a documented subset, snapshots zellij/history state, prints the decision-phase prototype action, and exits before mutating zellij. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/launcher` resolves data-dir inputs and reads zellij/history through fakeable seams. | go-entrypoint with `bin/pair` retained as compat fallback through #77; #75 owns decision-phase only, while real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
++| `bin/pair` / `cmd/internal/launcher` / `cmd/internal/entrypoint` | Bash public launcher plus Go-owned launch handoff | user shell, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` (#77) resolves sibling `bin/pair` from the `pair-go` executable and execs it with `pair`-compatible argv/env. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves the compatibility handoff; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Go handoff entrypoint with `bin/pair` retained as stable public launcher and compat target through #77; real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
+ | `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling `pair`. | Resolves symlinks; depends on `bin/pair` and `bin/lib/dev-rebuild.sh`. | compat-shim, likely stays as developer wrapper until Go entrypoint has dev mode | P1 |
+ | `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
+ | `zellij/layouts/main.kdl` | zellij native asset | `bin/pair` via `zellij --new-session-with-layout` | Defines agent and draft panes; shell expands Pair env at pane start. | Calls `pair-wrap`; calls `nvim -u "$PAIR_HOME/nvim/init.lua"`; writes `pane-<tag>-<agent>.json`; writes draft nvim pid file. | native-asset, packaged adjacent/embedded | P0 |
+@@ -129,7 +129,9 @@ Build/install callers:
+   `pair-go scrollback-render`, backed by shared internal runners while legacy
+   binary names remain live for shell/Lua callers. `pair slug` remains a later
+   candidate.
+-- #77 is the public entrypoint switch and should wait for both #75 and #76.
++- #77 made `pair-go launch ...` a meaningful Go-owned compatibility handoff to
++  `bin/pair`, with argv/env preserved and missing-launcher diagnostics. The
++  stable public `pair` script remains unchanged for this migration window.
+ - #78 should prioritize `pair-title.sh` and `pair-session-watch.sh` if stateful
+   shell remains a packaging/reliability problem after #77.
+ - #79 owns whether `nvim/` and `zellij/` are embedded or installed adjacent.
+diff --git a/cmd/internal/dispatcher/dispatcher.go b/cmd/internal/dispatcher/dispatcher.go
+index 3784f1e..fe1fbfd 100644
+--- a/cmd/internal/dispatcher/dispatcher.go
++++ b/cmd/internal/dispatcher/dispatcher.go
+@@ -3,12 +3,9 @@ package dispatcher
+ import (
+ 	"bytes"
+ 	"fmt"
+-	"os"
+ 	"strings"
+-	"time"
+ 
+ 	"github.com/xianxu/pair/cmd/internal/contextcmd"
+-	"github.com/xianxu/pair/cmd/internal/launcher"
+ 	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
+ )
+ 
+@@ -31,7 +28,7 @@ type Result struct {
+ // Families returns the planned command families for the Go dispatcher.
+ func Families() []CommandFamily {
+ 	return []CommandFamily{
+-		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "prototype"},
++		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "handoff"},
+ 		{Name: "context", Summary: "agent pane context meter", Status: "implemented"},
+ 		{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "implemented"},
+ 		{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
+@@ -53,11 +50,11 @@ func Dispatch(args []string) Result {
+ 		return Result{Stdout: Help(programName), ExitCode: 0}
+ 	case "version", "--version":
+ 		return Result{
+-			Stdout:   "pair-go dispatcher skeleton\npublic launcher: bin/pair\n",
++			Stdout:   "pair-go dispatcher skeleton\nlaunch handoff: bin/pair\n",
+ 			ExitCode: 0,
+ 		}
+ 	case "launch":
+-		return DispatchWithLauncherRuntime(args, osLauncherRuntime())
++		return launchHandoffResult()
+ 	case "context":
+ 		return dispatchContext(args[1:])
+ 	case "scrollback-render":
+@@ -89,90 +86,10 @@ func dispatchScrollbackRender(args []string) Result {
+ 	return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}
+ }
+ 
+-type LauncherRuntime struct {
+-	Env      launcher.Env
+-	Sessions launcher.SessionSource
+-	History  launcher.HistoricalScanner
+-}
+-
+-type StaticSessions struct {
+-	Sessions []launcher.Session
+-	Err      error
+-}
+-
+-func (s StaticSessions) Snapshot() ([]launcher.Session, error) {
+-	if s.Err != nil {
+-		return nil, s.Err
+-	}
+-	return s.Sessions, nil
+-}
+-
+-type StaticHistory struct {
+-	Tags []launcher.HistoricalTag
+-	Err  error
+-}
+-
+-func (h StaticHistory) Scan(_ string, _ time.Time) ([]launcher.HistoricalTag, error) {
+-	if h.Err != nil {
+-		return nil, h.Err
+-	}
+-	return h.Tags, nil
+-}
+-
+-func DispatchWithLauncherRuntime(args []string, rt LauncherRuntime) Result {
+-	launchArgs := []string(nil)
+-	if len(args) > 1 {
+-		launchArgs = args[1:]
+-	}
+-	if len(launchArgs) > 0 && (launchArgs[0] == "help" || launchArgs[0] == "--help" || launchArgs[0] == "-h") {
+-		return Result{Stdout: LaunchHelp(programName), ExitCode: 0}
+-	}
+-	outcome, err := launcher.Run(launchArgs, rt.Env, rt.Sessions, rt.History)
+-	if err != nil {
+-		return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
+-	}
+-	decision := outcome.Decision
++func launchHandoffResult() Result {
+ 	return Result{
+-		Stderr: fmt.Sprintf(
+-			"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
+-			decision.Action,
+-			decision.Tag,
+-			decision.SessionName,
+-		),
+-		ExitCode: 3,
+-	}
+-}
+-
+-func LaunchHelp(program string) string {
+-	return fmt.Sprintf(`Usage: %s launch [agent] [-- agent-args...]
+-       %s launch resume <tag>
+-
+-Guarded decision-phase prototype. Public sessions still start through bin/pair.
+-This command parses launch inputs and computes the create/attach/picker decision,
+-then stops before invoking zellij.
+-`, program, program)
+-}
+-
+-func LauncherEnv(home, xdgDataHome, cwd string) launcher.Env {
+-	return launcher.Env{
+-		Home:     home,
+-		XDGData:  xdgDataHome,
+-		Cwd:      cwd,
+-		Now:      time.Now(),
+-		HistoryD: 14,
+-	}
+-}
+-
+-func osLauncherRuntime() LauncherRuntime {
+-	home := os.Getenv("HOME")
+-	xdg := os.Getenv("XDG_DATA_HOME")
+-	cwd, _ := os.Getwd()
+-	env := LauncherEnv(home, xdg, cwd)
+-	dataDir := launcher.ResolveDataDir(home, xdg)
+-	return LauncherRuntime{
+-		Env:      env,
+-		Sessions: launcher.ZellijSource{},
+-		History:  launcher.HistorySource{DataDir: dataDir},
++		Stderr:   "pair-go launch is a process handoff implemented by cmd/pair-go; call pair-go launch ... instead of dispatcher.Dispatch\n",
++		ExitCode: 2,
+ 	}
+ }
+ 
+@@ -185,6 +102,8 @@ func Help(program string) string {
+ 	for _, family := range Families() {
+ 		if family.Status == "prototype" {
+ 			fmt.Fprintf(&b, "  %-17s %s (prototype; decision-phase only)\n", family.Name, family.Summary)
++		} else if family.Status == "handoff" {
++			fmt.Fprintf(&b, "  %-17s %s (compatibility handoff to bin/pair)\n", family.Name, family.Summary)
+ 		} else if family.Status == "implemented" {
+ 			fmt.Fprintf(&b, "  %-17s %s (implemented helper route)\n", family.Name, family.Summary)
+ 		}
+diff --git a/cmd/internal/dispatcher/dispatcher_test.go b/cmd/internal/dispatcher/dispatcher_test.go
+index f10f32b..7dc3021 100644
+--- a/cmd/internal/dispatcher/dispatcher_test.go
++++ b/cmd/internal/dispatcher/dispatcher_test.go
+@@ -21,7 +21,7 @@ func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
+ 				"Usage: pair-go <command> [args]",
+ 				"Implemented commands:",
+ 				"launch",
+-				"decision-phase only",
++				"compatibility handoff",
+ 				"context",
+ 				"scrollback-render",
+ 				"wrap",
+@@ -35,6 +35,9 @@ func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
+ 			if strings.Contains(res.Stdout, "launch             session lifecycle and public pair launcher flow (planned; not implemented") {
+ 				t.Fatalf("Stdout still labels launch unimplemented:\n%s", res.Stdout)
+ 			}
++			if strings.Contains(res.Stdout, "decision-phase only") {
++				t.Fatalf("Stdout still labels launch decision-phase only:\n%s", res.Stdout)
++			}
+ 			for _, stale := range []string{
+ 				"context           agent pane context meter (planned; not implemented",
+ 				"scrollback-render raw PTY capture to ANSI scrollback (planned; not implemented",
+@@ -55,7 +58,7 @@ func TestDispatchVersionIsDevelopmentSkeletonMetadata(t *testing.T) {
+ 	if res.Stderr != "" {
+ 		t.Fatalf("Stderr = %q, want empty", res.Stderr)
+ 	}
+-	for _, want := range []string{"pair-go", "dispatcher skeleton", "public launcher: bin/pair"} {
++	for _, want := range []string{"pair-go", "dispatcher skeleton", "launch handoff: bin/pair"} {
+ 		if !strings.Contains(res.Stdout, want) {
+ 			t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
+ 		}
+@@ -77,57 +80,15 @@ func TestDispatchPlannedCommandReturnsUnsupported(t *testing.T) {
+ 	}
+ }
+ 
+-func TestDispatchLaunchHelpRoutesToPrototype(t *testing.T) {
++func TestDispatchLaunchReportsProcessHandoff(t *testing.T) {
+ 	res := Dispatch([]string{"launch", "--help"})
+-	if res.ExitCode != 0 {
+-		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
+-	}
+-	if res.Stderr != "" {
+-		t.Fatalf("Stderr = %q, want empty", res.Stderr)
+-	}
+-	for _, want := range []string{"Usage: pair-go launch", "decision-phase prototype"} {
+-		if !strings.Contains(res.Stdout, want) {
+-			t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
+-		}
+-	}
+-}
+-
+-func TestDispatchLaunchReturnsPrototypeDecision(t *testing.T) {
+-	res := DispatchWithLauncherRuntime([]string{"launch", "resume", "demo"}, LauncherRuntime{
+-		Env: LauncherEnv("/home/me", "", "/work/pair"),
+-		Sessions: StaticSessions{
+-			Sessions: nil,
+-		},
+-		History: StaticHistory{},
+-	})
+-	if res.ExitCode != 3 {
+-		t.Fatalf("ExitCode = %d, want 3", res.ExitCode)
+-	}
+-	if res.Stdout != "" {
+-		t.Fatalf("Stdout = %q, want empty", res.Stdout)
+-	}
+-	for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
+-		if !strings.Contains(res.Stderr, want) {
+-			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
+-		}
+-	}
+-}
+-
+-func TestDispatchLaunchWithoutArgsReturnsDefaultPrototypeDecision(t *testing.T) {
+-	res := DispatchWithLauncherRuntime([]string{"launch"}, LauncherRuntime{
+-		Env: LauncherEnv("/home/me", "", "/work/pair"),
+-		Sessions: StaticSessions{
+-			Sessions: nil,
+-		},
+-		History: StaticHistory{},
+-	})
+-	if res.ExitCode != 3 {
+-		t.Fatalf("ExitCode = %d, want 3", res.ExitCode)
++	if res.ExitCode != 2 {
++		t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
+ 	}
+ 	if res.Stdout != "" {
+ 		t.Fatalf("Stdout = %q, want empty", res.Stdout)
+ 	}
+-	for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=pair", "session=pair-pair"} {
++	for _, want := range []string{"pair-go launch", "process handoff", "cmd/pair-go"} {
+ 		if !strings.Contains(res.Stderr, want) {
+ 			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
+ 		}
+diff --git a/cmd/internal/entrypoint/launch.go b/cmd/internal/entrypoint/launch.go
+new file mode 100644
+index 0000000..1a40970
+--- /dev/null
++++ b/cmd/internal/entrypoint/launch.go
+@@ -0,0 +1,20 @@
++package entrypoint
++
++import "path/filepath"
++
++// LegacyLaunchRequest describes the current compatibility handoff from
++// pair-go launch to the shell-owned pair launcher.
++type LegacyLaunchRequest struct {
++	Path string
++	Argv []string
++}
++
++func ResolveLegacyLaunch(executable string, launchArgs []string) LegacyLaunchRequest {
++	argv := make([]string, 0, len(launchArgs)+1)
++	argv = append(argv, "pair")
++	argv = append(argv, launchArgs...)
++	return LegacyLaunchRequest{
++		Path: filepath.Join(filepath.Dir(executable), "pair"),
++		Argv: argv,
++	}
++}
+diff --git a/cmd/internal/entrypoint/launch_test.go b/cmd/internal/entrypoint/launch_test.go
+new file mode 100644
+index 0000000..cdeec2c
+--- /dev/null
++++ b/cmd/internal/entrypoint/launch_test.go
+@@ -0,0 +1,28 @@
++package entrypoint
++
++import (
++	"reflect"
++	"testing"
++)
++
++func TestResolveLegacyLaunchDropsLaunchVerb(t *testing.T) {
++	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"claude", "--", "--resume"})
++	if req.Path != "/repo/bin/pair" {
++		t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
++	}
++	want := []string{"pair", "claude", "--", "--resume"}
++	if !reflect.DeepEqual(req.Argv, want) {
++		t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
++	}
++}
++
++func TestResolveLegacyLaunchPreservesSubcommands(t *testing.T) {
++	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"resume", "demo"})
++	if req.Path != "/repo/bin/pair" {
++		t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
++	}
++	want := []string{"pair", "resume", "demo"}
++	if !reflect.DeepEqual(req.Argv, want) {
++		t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
++	}
++}
+diff --git a/cmd/pair-go/launch_process_test.go b/cmd/pair-go/launch_process_test.go
+deleted file mode 100644
+index 9563492..0000000
+--- a/cmd/pair-go/launch_process_test.go
++++ /dev/null
+@@ -1,71 +0,0 @@
+-package main
+-
+-import (
+-	"bytes"
+-	"os"
+-	"path/filepath"
+-	"strings"
+-	"testing"
+-	"time"
+-)
+-
+-func TestRunLaunchWithFakeZellij(t *testing.T) {
+-	rt := t.TempDir()
+-	bin := filepath.Join(rt, "bin")
+-	data := filepath.Join(rt, "data")
+-	if err := os.MkdirAll(bin, 0o755); err != nil {
+-		t.Fatal(err)
+-	}
+-	if err := os.MkdirAll(filepath.Join(data, "pair"), 0o755); err != nil {
+-		t.Fatal(err)
+-	}
+-	log := filepath.Join(rt, "zellij.log")
+-	zellij := filepath.Join(bin, "zellij")
+-	script := `#!/usr/bin/env bash
+-printf '%s\n' "$*" >> "` + log + `"
+-case "$*" in
+-  "list-sessions --short") printf 'pair-live\npair-detached\npair-exited\n' ;;
+-  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-exited [Created] (EXITED - attach to resurrect)\n' ;;
+-  "--session pair-live action list-clients") printf 'CLIENTS\n1\n' ;;
+-  "--session pair-detached action list-clients") printf 'CLIENTS\n' ;;
+-  *attach*|*new-session*|*--new-session-with-layout*|*delete-session*) printf 'MUTATING %s\n' "$*" >> "` + log + `"; exit 99 ;;
+-  *) exit 1 ;;
+-esac
+-`
+-	if err := os.WriteFile(zellij, []byte(script), 0o755); err != nil {
+-		t.Fatal(err)
+-	}
+-	historical := filepath.Join(data, "pair", "draft-pair-old.md")
+-	if err := os.WriteFile(historical, []byte("draft"), 0o644); err != nil {
+-		t.Fatal(err)
+-	}
+-	now := time.Now()
+-	if err := os.Chtimes(historical, now, now); err != nil {
+-		t.Fatal(err)
+-	}
+-
+-	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
+-	t.Setenv("HOME", filepath.Join(rt, "home"))
+-	t.Setenv("XDG_DATA_HOME", data)
+-
+-	var stdout, stderr bytes.Buffer
+-	code := run([]string{"launch", "claude"}, &stdout, &stderr)
+-	if code != 3 {
+-		t.Fatalf("code = %d, want 3; stderr:\n%s", code, stderr.String())
+-	}
+-	if stdout.String() != "" {
+-		t.Fatalf("stdout = %q, want empty", stdout.String())
+-	}
+-	for _, want := range []string{"prototype decision", "action=pick"} {
+-		if !strings.Contains(stderr.String(), want) {
+-			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
+-		}
+-	}
+-	logBytes, err := os.ReadFile(log)
+-	if err != nil {
+-		t.Fatal(err)
+-	}
+-	if strings.Contains(string(logBytes), "MUTATING") {
+-		t.Fatalf("fake zellij recorded mutating invocation:\n%s", string(logBytes))
+-	}
+-}
+diff --git a/cmd/pair-go/main.go b/cmd/pair-go/main.go
+index bf7d28b..760f876 100644
+--- a/cmd/pair-go/main.go
++++ b/cmd/pair-go/main.go
+@@ -1,12 +1,15 @@
+-// pair-go is a development-only dispatcher skeleton for the future primary Go
+-// CLI. The public launcher remains bin/pair.
++// pair-go is the development dispatcher for the future primary Go CLI. Its
++// launch route is a compatibility handoff to the current shell launcher.
+ package main
+ 
+ import (
++	"fmt"
+ 	"io"
+ 	"os"
++	"syscall"
+ 
+ 	"github.com/xianxu/pair/cmd/internal/dispatcher"
++	"github.com/xianxu/pair/cmd/internal/entrypoint"
+ )
+ 
+ func main() {
+@@ -14,13 +17,65 @@ func main() {
+ }
+ 
+ func run(args []string, stdout, stderr io.Writer) int {
++	return runWithLegacyRuntime(args, stdout, stderr, osLegacyRuntime{})
++}
++
++type legacyRuntime interface {
++	Executable() (string, error)
++	Stat(path string) error
++	Environ() []string
++	Exec(path string, argv []string, env []string) int
++}
++
++func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
++	if len(args) > 0 && args[0] == "launch" {
++		return runLegacyLaunch(args[1:], stderr, rt)
++	}
+ 	res := dispatcher.Dispatch(args)
+ 	return writeResult(res, stdout, stderr)
+ }
+ 
+-func runWithLauncherRuntime(args []string, stdout, stderr io.Writer, rt dispatcher.LauncherRuntime) int {
+-	res := dispatcher.DispatchWithLauncherRuntime(args, rt)
+-	return writeResult(res, stdout, stderr)
++func runLegacyLaunch(args []string, stderr io.Writer, rt legacyRuntime) int {
++	exe, err := rt.Executable()
++	if err != nil {
++		_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
++		return 1
++	}
++	req := entrypoint.ResolveLegacyLaunch(exe, args)
++	if err := rt.Stat(req.Path); err != nil {
++		_, _ = fmt.Fprintf(stderr, "pair-go launch: pair launcher not found at %s (%v); run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", req.Path, err)
++		return 1
++	}
++	return rt.Exec(req.Path, req.Argv, rt.Environ())
++}
++
++type osLegacyRuntime struct{}
++
++func (osLegacyRuntime) Executable() (string, error) {
++	return os.Executable()
++}
++
++func (osLegacyRuntime) Stat(path string) error {
++	info, err := os.Stat(path)
++	if err != nil {
++		return err
++	}
++	if info.IsDir() {
++		return fmt.Errorf("is a directory")
++	}
++	return nil
++}
++
++func (osLegacyRuntime) Environ() []string {
++	return os.Environ()
++}
++
++func (osLegacyRuntime) Exec(path string, argv []string, env []string) int {
++	if err := syscall.Exec(path, argv, env); err != nil {
++		_, _ = fmt.Fprintf(os.Stderr, "pair-go launch: exec %s failed: %v\n", path, err)
++		return 1
++	}
++	return 0
+ }
+ 
+ func writeResult(res dispatcher.Result, stdout, stderr io.Writer) int {
+diff --git a/cmd/pair-go/main_test.go b/cmd/pair-go/main_test.go
+index b0d6f7f..d137a6b 100644
+--- a/cmd/pair-go/main_test.go
++++ b/cmd/pair-go/main_test.go
+@@ -2,10 +2,11 @@ package main
+ 
+ import (
+ 	"bytes"
++	"errors"
++	"os"
++	"reflect"
+ 	"strings"
+ 	"testing"
+-
+-	"github.com/xianxu/pair/cmd/internal/dispatcher"
+ )
+ 
+ func TestRunWritesStdoutAndReturnsDispatcherCode(t *testing.T) {
+@@ -37,55 +38,122 @@ func TestRunWritesStderrAndReturnsDispatcherCode(t *testing.T) {
+ }
+ 
+ func TestRunLaunchHelp(t *testing.T) {
++	rt := &fakeLegacyRuntime{
++		executable: "/repo/bin/pair-go",
++	}
+ 	var stdout, stderr bytes.Buffer
+-	code := run([]string{"launch", "--help"}, &stdout, &stderr)
++	code := runWithLegacyRuntime([]string{"launch", "--help"}, &stdout, &stderr, rt)
+ 	if code != 0 {
+ 		t.Fatalf("code = %d, want 0", code)
+ 	}
+-	if !strings.Contains(stdout.String(), "Usage: pair-go launch") {
+-		t.Fatalf("stdout missing launch usage:\n%s", stdout.String())
++	if stdout.String() != "" {
++		t.Fatalf("stdout = %q, want empty", stdout.String())
+ 	}
+ 	if stderr.String() != "" {
+ 		t.Fatalf("stderr = %q, want empty", stderr.String())
+ 	}
++	if rt.execPath != "/repo/bin/pair" {
++		t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
++	}
++	wantArgv := []string{"pair", "--help"}
++	if !reflect.DeepEqual(rt.execArgv, wantArgv) {
++		t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
++	}
+ }
+ 
+-func TestRunLaunchResumeReturnsPrototypeDecision(t *testing.T) {
++func TestRunLaunchExecsLegacyPairWithArgvAndEnv(t *testing.T) {
++	t.Setenv("PAIR_TEST_ENV", "kept")
++	rt := &fakeLegacyRuntime{
++		executable: "/repo/bin/pair-go",
++		execCode:   42,
++	}
++
+ 	var stdout, stderr bytes.Buffer
+-	code := runWithLauncherRuntime([]string{"launch", "resume", "demo"}, &stdout, &stderr, testLauncherRuntime("/home/me", "", "/work/pair"))
+-	if code != 3 {
+-		t.Fatalf("code = %d, want 3", code)
++	code := runWithLegacyRuntime([]string{"launch", "claude", "--", "--resume"}, &stdout, &stderr, rt)
++
++	if code != 42 {
++		t.Fatalf("code = %d, want 42", code)
+ 	}
+ 	if stdout.String() != "" {
+ 		t.Fatalf("stdout = %q, want empty", stdout.String())
+ 	}
+-	for _, want := range []string{"prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
+-		if !strings.Contains(stderr.String(), want) {
+-			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
+-		}
++	if stderr.String() != "" {
++		t.Fatalf("stderr = %q, want empty", stderr.String())
++	}
++	if rt.execPath != "/repo/bin/pair" {
++		t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
++	}
++	wantArgv := []string{"pair", "claude", "--", "--resume"}
++	if !reflect.DeepEqual(rt.execArgv, wantArgv) {
++		t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
++	}
++	if !containsEnv(rt.execEnv, "PAIR_TEST_ENV=kept") {
++		t.Fatalf("execEnv missing PAIR_TEST_ENV=kept: %#v", rt.execEnv)
+ 	}
+ }
+ 
+-func TestRunLaunchWithoutArgsReturnsDefaultPrototypeDecision(t *testing.T) {
++func TestRunLaunchReportsMissingLegacyPair(t *testing.T) {
++	rt := &fakeLegacyRuntime{
++		executable: "/repo/bin/pair-go",
++		statErr:    os.ErrNotExist,
++	}
++
+ 	var stdout, stderr bytes.Buffer
+-	code := runWithLauncherRuntime([]string{"launch"}, &stdout, &stderr, testLauncherRuntime("/home/me", "", "/work/pair"))
+-	if code != 3 {
+-		t.Fatalf("code = %d, want 3", code)
++	code := runWithLegacyRuntime([]string{"launch", "claude"}, &stdout, &stderr, rt)
++
++	if code != 1 {
++		t.Fatalf("code = %d, want 1", code)
+ 	}
+ 	if stdout.String() != "" {
+ 		t.Fatalf("stdout = %q, want empty", stdout.String())
+ 	}
+-	for _, want := range []string{"prototype decision", "action=create", "tag=pair", "session=pair-pair"} {
++	for _, want := range []string{"pair-go launch", "/repo/bin/pair", "make build", "make install", "dev-aliases.sh"} {
+ 		if !strings.Contains(stderr.String(), want) {
+ 			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
+ 		}
+ 	}
++	if rt.execPath != "" {
++		t.Fatalf("execPath = %q, want empty", rt.execPath)
++	}
+ }
+ 
+-func testLauncherRuntime(home, xdg, cwd string) dispatcher.LauncherRuntime {
+-	return dispatcher.LauncherRuntime{
+-		Env:      dispatcher.LauncherEnv(home, xdg, cwd),
+-		Sessions: dispatcher.StaticSessions{},
+-		History:  dispatcher.StaticHistory{},
++type fakeLegacyRuntime struct {
++	executable string
++	statErr    error
++	execCode   int
++
++	execPath string
++	execArgv []string
++	execEnv  []string
++}
++
++func (f *fakeLegacyRuntime) Executable() (string, error) {
++	if f.executable == "" {
++		return "", errors.New("missing executable")
++	}
++	return f.executable, nil
++}
++
++func (f *fakeLegacyRuntime) Stat(_ string) error {
++	return f.statErr
++}
++
++func (f *fakeLegacyRuntime) Environ() []string {
++	return os.Environ()
++}
++
++func (f *fakeLegacyRuntime) Exec(path string, argv []string, env []string) int {
++	f.execPath = path
++	f.execArgv = append([]string(nil), argv...)
++	f.execEnv = append([]string(nil), env...)
++	return f.execCode
++}
++
++func containsEnv(env []string, want string) bool {
++	for _, got := range env {
++		if got == want {
++			return true
++		}
+ 	}
++	return false
+ }
+
+
+exec
+/bin/zsh -lc "sed -n '1,220p' /Users/xianxu/workspace/ariadne/construct/adapted/superpowers-using-superpowers/SKILL.md && sdlc --help && sed -n '1,220p' workshop/lessons.md && sed -n '1,240p' workshop/issues/000077-go-entrypoint-switch.md" in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+---
+name: superpowers-using-superpowers
+description: Use when starting any conversation - establishes how to find and use skills, requiring Skill tool invocation before ANY response including clarifying questions
+---
+
+<SUBAGENT-STOP>
+If you were dispatched as a subagent to execute a specific task, skip this skill.
+</SUBAGENT-STOP>
+
+<EXTREMELY-IMPORTANT>
+If you think there is even a 1% chance a skill might apply to what you are doing, you ABSOLUTELY MUST invoke the skill.
+
+IF A SKILL APPLIES TO YOUR TASK, YOU DO NOT HAVE A CHOICE. YOU MUST USE IT.
+
+This is not negotiable. This is not optional. You cannot rationalize your way out of this.
+</EXTREMELY-IMPORTANT>
+
+## Instruction Priority
+
+> **Ariadne note:** AGENTS.md Section 3 governs subagent strategy and overrides skills that mandate subagent-driven-development as the default execution path.
+
+Superpowers skills override default system prompt behavior, but **user instructions always take precedence**:
+
+1. **User's explicit instructions** (CLAUDE.md, GEMINI.md, AGENTS.md, direct requests) — highest priority
+2. **Superpowers skills** — override default system behavior where they conflict
+3. **Default system prompt** — lowest priority
+
+If CLAUDE.md, GEMINI.md, or AGENTS.md says "don't use TDD" and a skill says "always use TDD," follow the user's instructions. The user is in control.
+
+## How to Access Skills
+
+**In Claude Code:** Use the `Skill` tool. When you invoke a skill, its content is loaded and presented to you—follow it directly. Never use the Read tool on skill files.
+
+**In Gemini CLI:** Skills activate via the `activate_skill` tool. Gemini loads skill metadata at session start and activates the full content on demand.
+
+**In other environments:** Check your platform's documentation for how skills are loaded.
+
+## Platform Adaptation
+
+Skills use Claude Code tool names. Non-CC platforms: see `references/codex-tools.md` (Codex) for tool equivalents. Gemini CLI users get the tool mapping loaded automatically via GEMINI.md.
+
+# Using Skills
+
+## The Rule
+
+**Invoke relevant or requested skills BEFORE any response or action.** Even a 1% chance a skill might apply means that you should invoke the skill to check. If an invoked skill turns out to be wrong for the situation, you don't need to use it.
+
+```dot
+digraph skill_flow {
+    "User message received" [shape=doublecircle];
+    "About to EnterPlanMode?" [shape=doublecircle];
+    "Already brainstormed?" [shape=diamond];
+    "Invoke brainstorming skill" [shape=box];
+    "Might any skill apply?" [shape=diamond];
+    "Invoke Skill tool" [shape=box];
+    "Announce: 'Using [skill] to [purpose]'" [shape=box];
+    "Has checklist?" [shape=diamond];
+    "Create TodoWrite todo per item" [shape=box];
+    "Follow skill exactly" [shape=box];
+    "Respond (including clarifications)" [shape=doublecircle];
+
+    "About to EnterPlanMode?" -> "Already brainstormed?";
+    "Already brainstormed?" -> "Invoke brainstorming skill" [label="no"];
+    "Already brainstormed?" -> "Might any skill apply?" [label="yes"];
+    "Invoke brainstorming skill" -> "Might any skill apply?";
+
+    "User message received" -> "Might any skill apply?";
+    "Might any skill apply?" -> "Invoke Skill tool" [label="yes, even 1%"];
+    "Might any skill apply?" -> "Respond (including clarifications)" [label="definitely not"];
+    "Invoke Skill tool" -> "Announce: 'Using [skill] to [purpose]'";
+    "Announce: 'Using [skill] to [purpose]'" -> "Has checklist?";
+    "Has checklist?" -> "Create TodoWrite todo per item" [label="yes"];
+    "Has checklist?" -> "Follow skill exactly" [label="no"];
+    "Create TodoWrite todo per item" -> "Follow skill exactly";
+}
+```
+
+## Red Flags
+
+These thoughts mean STOP—you're rationalizing:
+
+| Thought | Reality |
+|---------|---------|
+| "This is just a simple question" | Questions are tasks. Check for skills. |
+| "I need more context first" | Skill check comes BEFORE clarifying questions. |
+| "Let me explore the codebase first" | Skills tell you HOW to explore. Check first. |
+| "I can check git/files quickly" | Files lack conversation context. Check for skills. |
+| "Let me gather information first" | Skills tell you HOW to gather information. |
+| "This doesn't need a formal skill" | If a skill exists, use it. |
+| "I remember this skill" | Skills evolve. Read current version. |
+| "This doesn't count as a task" | Action = task. Check for skills. |
+| "The skill is overkill" | Simple things become complex. Use it. |
+| "I'll just do this one thing first" | Check BEFORE doing anything. |
+| "This feels productive" | Undisciplined action wastes time. Skills prevent this. |
+| "I know what that means" | Knowing the concept ≠ using the skill. Invoke it. |
+
+## Skill Priority
+
+When multiple skills could apply, use this order:
+
+1. **Process skills first** (brainstorming, debugging) - these determine HOW to approach the task
+2. **Implementation skills second** (frontend-design, mcp-builder) - these guide execution
+
+"Let's build X" → brainstorming first, then implementation skills.
+"Fix this bug" → debugging first, then domain-specific skills.
+
+## Skill Types
+
+**Rigid** (TDD, debugging): Follow exactly. Don't adapt away discipline.
+
+**Flexible** (patterns): Adapt principles to context.
+
+The skill itself tells you which.
+
+## User Instructions
+
+Instructions say WHAT, not HOW. "Add X" or "Fix Y" doesn't mean skip workflows.
+sdlc collects ariadne's SDLC checkpoint guards into one binary. Each subcommand
+owns one checkpoint: it requires evidence at the gate, mutates state, logs the
+transition, and refuses transitions that lack it. We don't model the SDLC as a
+state machine — stages stay prose; we codify the gates between them where drift
+recurs. `sdlc` manages the development life cycle; prefer it over `git`/`gh`.
+
+BEFORE WORK
+  - `sdlc claim --issue N` — the single start-of-work gesture, a CHEAP LOCK.
+    Flips an *open* issue to `working` and publishes the claim to origin/main so
+    peer agents see it. No estimate demanded (#113) — claim early, the moment an
+    idea crystallizes. `--no-start` suppresses the flip.
+  - Do NOT hand-edit an issue's `status:` — let `sdlc claim` or `sdlc issue
+    set-status` own that transition (it carries the reopen/`→ done` guards).
+
+ENTER IMPLEMENTATION
+  - After plan approval, before editing code, run `sdlc change-code`. It owns the
+    branching decision (in-place branch by default; `--worktree=yes` for an
+    isolated worktree), the plan-quality check, and the `estimate_hours` gate
+    (relocated here from claim, #113). Don't start coding without it.
+
+PUBLISH
+  - Publishing goes through a PR: `sdlc pr` → `sdlc merge`. Direct `sdlc push`
+    if working directly on main.
+
+RECOVER
+  - After a compaction or session resume, run `sdlc state` to recover where you
+    are instead of re-inferring from issue files.
+
+LOCAL REPO TRANSACTION LOCK
+  - Mutating verbs take an SDLC-owned repo transaction lock at
+    `.git/sdlc.lock` before reading/writing issue state, committing, changing
+    branches, or pushing. The lock is local to the Git common dir, so linked
+    worktrees of the same repo serialize with each other.
+  - Wait messages identify the holder pid and command when metadata is
+    available. `change-code`, `close`, `milestone-close`, `merge`, and `push`
+    can hold the lock during long-running review/ship transactions; wait or
+    retry rather than removing the lock while that process is alive.
+  - A dead same-host holder is reclaimed automatically; initializing metadata
+    is waited through. Other stale/timeout errors tell you how to inspect
+    `.git/sdlc.lock`. Remote push/ref races are separate: the local lock
+    serializes this checkout, not another machine or clone.
+
+WHEN A VERB ERRORS
+  Do NOT route around it with hand-rolled `git`/`gh`. Its errors are next-action
+  specs. The fix is one of two things:
+    (a) satisfy the precondition it names and re-run the same verb (e.g. `sdlc
+        merge` saying "no upstream" → run `sdlc pr` first, then `sdlc merge`); or
+    (b) if the error is a genuine gap in `sdlc` itself, fix that edge case in the
+        source and re-run. We're still ironing out edge cases.
+  Only drop to manual when a verb genuinely cannot express the need — say so.
+
+These gates sit inside a wider prose arc the binary does NOT own: ideation
+(parley/pensive) → brainstorm → plan → build → milestone review (`sdlc judge`,
+auto-dispatched) → close/ship → postmortem.
+
+CONVENTIONS
+
+  --issue vs --github-issue — `--issue N` always means workshop/issues
+  (6-digit ID). `--github-issue N` means a GitHub issue number. Bare `--issue`
+  never means a GitHub issue.
+
+  Form vs essence — checkpoint guards (close, milestone-close, push, merge)
+  defend against *omission* via required-evidence flags; `sdlc judge` defends
+  against *theater* via fresh-context review. Form runs first; judge second.
+
+The verb list + per-verb help (`sdlc <verb> --help`) follow below.
+
+Usage:
+  sdlc [flags]
+  sdlc [command]
+
+Available Commands:
+  claim           Start work: flip an open issue to working + broadcast the claim
+  start-plan      Enter planning: deliver the architecture principles to design against (#75)
+  change-code     Enter implementation after the structural + plan-quality gates
+  issue           Create + manage issues (new / set-status / list / show)
+  actual          Compute an issue's focused dev-hours via active-time-v3 (#68)
+  active-time     Per-issue active-time attribution table (the v3 engine, standalone)
+  close           Close an issue or milestone (ACTUAL + VERIFIED + atlas/project sweep)
+  milestone-close Close one milestone + auto-dispatch its review
+  pr              Open a pull request from a feature branch
+  merge           Merge the PR, archive done issues, clean up
+  push            Ship from main (clean tree + pre-merge judges + archive)
+  state           Inspect workflow state (branch, working issues, drift)
+  judge           Run an LLM-judge check against the diff (fresh-context)
+  arch-principles Print the ARCH-* architecture principles (single source; pull for non-gate work)
+  estimate-source Name the shared estimate method + the repo-local calibration source (pull)
+  propagate-base  Re-weave every recursive dependent of this repo (foundation-first)
+  help            Help about any command
+
+Flags:
+  -h, --help   help for sdlc
+
+Use "sdlc [command] --help" for more information about a command.
+# Lessons
+
+## Lua patterns: `\0` is empty-position match, not NUL byte
+
+The unescape function in `nvim/scrollback.lua` first attempt used a
+placeholder dance: `s:gsub('\\\\', '\0')` to swap `\\` for NUL,
+then `gsub('\\(.)', '%1')` to strip remaining `\X`, then
+`gsub('\0', '\\')` to restore the NUL → `\`. The result was
+absurd: `unescape("plain")` returned `\p\l\a\i\n\` — the NUL pattern
+matches between every byte (empty-position match), not the NUL byte
+character. Each "match" inserted a `\` between every char.
+
+**Rule.** When you need to match a literal NUL byte in a Lua pattern,
+use `%z` or wrap as a character class `[%z]`. But the cleaner answer
+is usually to skip patterns entirely for character-by-character
+walks: a tiny while-loop with `s:sub(i, i)` is unambiguous and avoids
+all the pattern-syntax footguns. Caught in #000018 review.
+
+## Escape on insert, scan-with-parity on extract — for delimited markers
+
+When user-supplied content is embedded in a delimited container
+(e.g. `🤖<X>[Y]`), and X or Y can contain the delimiter chars,
+the choice is "escape at insert + unescape at extract" vs "find
+the closing delimiter cleverly." The first attempt at `🤖<X>[Y]`
+parsing tried the latter — find first `>`, peek for `[`, give up
+otherwise. Result: any selection with `>` was silently dropped on
+extract, since the user couldn't tell the marker had been written
+malformed.
+
+**Rule.** Escape the delimiter chars in user-supplied fields at
+insert time; have the parser walk byte-by-byte counting backslash
+parity to find the *next unescaped* delimiter; unescape the
+extracted content. The escape→walk→unescape chain handles every
+delimiter-collision case uniformly, including `\\>` (literal `\`
+followed by `>`). Don't try to be clever with "find first `>[`
+adjacent" patterns — they fail when X contains `>[` literally,
+and the failure mode is silent data loss. Caught in #000018 review.
+
+## Shared delimiter codecs beat subsystem-local marker parsing
+
+M4b's review pane added `Alt+q` visual wrapping as `🤖<selection>[]` but initially
+embedded the selected text raw, even though annotate already had delimiter escaping for
+the same marker family. A selection containing `>` or `]` could truncate the parsed marker
+and make accept/reject leave stray syntax in the document.
+
+**Rule.** When a second feature writes the same delimited marker format, reuse or extract
+the existing codec before adding parser/writer code. Add tests for delimiter collisions
+(`>`, `]`, backslash) at the write path and the consume path. A parser unit test alone is
+not enough; the UI wrapper that inserts the marker must also be covered. Caught in #000066
+M4b review.
+
+## Shell scripts should use JSON builders, not `printf` JSON
+
+`pair-review-readiness` originally printed JSON with `printf` and unescaped string fields.
+A review branch named `review/a"b` produced invalid JSON, even though all the boolean
+fields were correct.
+
+**Rule.** In shell seams that emit JSON, use `jq -n --arg/--argjson` (or an equivalent
+structured encoder) for every field. Do not hand-build JSON with `printf` unless every
+string field is impossible by construction — and then document why. Guard it with a test
+using quotes in a branch/path/name. Caught in #000066 M4b review.
+
+## `#table` is 0 on string-keyed tables — never use it for ID generation
+
+Adding nvim/scrollback.lua's hl-group cache: `local name = 'PairScrollback_' .. (#hl_cache + 1)` was meant to give each new (state→hl-group) entry a unique numeric suffix. `hl_cache` is a string-keyed dict (cache key is `(fg|bg|attrs)`), and Lua's `#` on a non-array table returns 0. Result: every group resolved to `PairScrollback_1`, `nvim_set_hl(0, "PairScrollback_1", def)` overwrote on each call, and all extmarks ended up sharing whatever the last-written attrs were. Caught only by an end-to-end test that checked extmark hl_groups against expected fg/bg ints.
+
+**Rule.** When you need monotonic IDs in Lua, use an explicit counter (`local counter = 0; ... counter = counter + 1`). Do not use `#table` unless `table` is provably array-shaped (`{[1]=..., [2]=..., ...}`). The bug is silent — `nvim_set_hl` doesn't error on overwrite, it just wins-last. Filed during #000017 M4.
+
+## Empty fields in delimited parsing — `[^;]+` drops them; semantics may differ
+
+ECMA-48 SGR semantics: an omitted field is `0` (reset). So `\x1b[;1m` = "reset; bold". The first SGR parser pass used `params:gmatch('[^;]+')`, which silently skips empty fields — `[;1m` produced just `1` (bold), and any standing fg/bg/decoration leaked through. Caught in code review of #000017 (no real input from pair-scrollback-render's output would have triggered it, but it's a correctness footgun for any future caller pointing the viewer at non-pair-rendered ANSI).
+
+**Rule.** When the protocol says "empty field has meaning," parse with `([^;]*);` on a `string + ';'` so the trailing-delimiter trick yields every field including empties. Generally true for any delimiter-separated format where omission has semantic value (CSV with empty cells, env-var lists, SGR, etc.).
+
+## Sparse data structures: iterate by index, not by `.keys()`, when count must be exact
+
+pyte's `screen.buffer` is a `StaticDefaultDict` — accessing `buffer[y][x]` lazily creates a default Char, but `buffer.keys()` only contains rows that were *written to*. The renderer originally did `for y in sorted(screen.buffer.keys())`, which silently dropped trailing blank rows when the agent cleared and paused mid-redraw. That shifts every subsequent line number — directly breaking the feature's core promise that `:880` lands where zellij showed line 880. Caught in code review of #000017.
+
+**Rule.** When iterating over a sparse-by-design structure where every slot has a logical existence (even if unwritten), use `range(0, total)` and let the structure's `__getitem__` materialize defaults. `.keys()` is only correct when "absent" really means "doesn't exist." Same shape applies to anything with lazy materialization: defaultdicts, JS Maps with default fallbacks, sparse arrays.
+
+## Atomic write for files a feature can race on its own
+
+`bin/pair-scrollback-render` initially opened `<out.ansi>` with `'w'` (truncate-then-write). Two `Alt+/` presses in quick succession would race on the same path; whichever finished second left a half-interleaved file for nvim to open. Fixed by writing to `<out.ansi>.tmp` and `os.replace()`-ing at the end.
+
+**Rule.** Any output file that a user-triggered keybind (or any concurrently-fireable mechanism) writes to should use the tempfile + atomic rename pattern. The cost is one extra file path; the gain is that readers see only "old complete file" or "new complete file," never "torn file." Apply uniformly even when a race is unlikely — discipline reduces the cognitive load for future readers.
+
+## Verify zellij action and flag names against the installed version
+
+Two bugs in v1 of `bin/pair` and `zellij/config.kdl` came from going off memory of zellij's API:
+
+- Used `TogglePaneFullscreen` for the Alt+u bind. The actual action name in zellij 0.44.1 is `ToggleFocusFullscreen`. Caught by `zellij setup --check --config-dir <pair>/zellij`.
+- Used `--layout PATH --session NAME` to "create a new named session with this layout." Zellij's actual semantic: when `--session` is set, `--layout` means "add as tab to that session" and errors if the session doesn't exist. The right flag is `--new-session-with-layout` (`-n`).
+
+**Rule.** Before writing zellij KDL or invoking the zellij CLI:
+
+1. Run `zellij setup --dump-config` to see the canonical action names used in default keybinds.
+2. Run `zellij --help`, `zellij attach --help`, `zellij setup --help` against the installed version, and read the flag descriptions in full — they have non-obvious conditional semantics.
+3. Always validate config and layout files with `zellij setup --check --config-dir <dir>` and `zellij setup --dump-layout <path>` before committing.
+
+The verification tools are cheap and authoritative. Memory of "I think it's called X" is not.
+
+## Stage content edits before `git mv` when closing an issue
+
+Closing an issue means (a) editing the file (`status: done`, plan checkboxes), then (b) moving it to `workshop/history/`. Done in that order with `Edit` then `git mv`, the rename gets staged but the unstaged content edits do *not* — they stay in the working tree. `make issue-sync` only stages `workshop/issues/`, so the edits silently miss the commit. End state: history file with stale `status: working`.
+
+**Rule.** When closing an issue:
+1. Edit the file in place under `workshop/issues/` and `git add` it (or use `git add -u` after editing).
+2. Then `git mv` to `workshop/history/` — git carries the staged content into the rename.
+3. Or simpler: `git mv` first, edit second, `git add` the new path.
+
+After running `make issue-sync` on a close, verify with `git show HEAD:workshop/history/<file> | grep status:` that the committed file actually has `status: done`. Don't trust the rename alone.
+
+## On cancel, restore the prior visible state
+
+When a confirmation prompt or interactive flow is dismissed, the cancel path must put the UI back exactly how it was — not just "do nothing." Issuing a prompt via `nvim_echo`/`getchar` (or any flow that paints over a region: cmdline, statusline, floating windows, virtual text, highlights) leaves that region in the prompt's state. The proceed branch usually triggers a redraw incidentally (state changes → statusline refresh → cmdline cleared). The cancel branch does not, so the prompt residue lingers until the next user input.
+
+**Rule.** For every interactive surface, the cancel path is responsible for the same restoration the proceed path gets for free:
+
+- Prompts that overdraw the cmdline/statusline → call the same redraw/refresh helper the success path calls (e.g. `refresh_statusline()`), not just `return`.
+- Operations that mutated buffer text/cursor/window before asking for confirmation → snapshot first, restore on cancel.
+- Highlights, virtual text, floating windows added as part of the flow → tear them down on cancel just like on success.
+
+Treat cancel as an active branch with cleanup duties, not an early return. If you find yourself writing `if ch == 'n' then return end`, ask: what did the proceed branch do that I'm now skipping, and is any of it visual cleanup that cancel also needs?
+
+## Transcript summarization must bias toward USER turns, not a flat tail
+
+`cmd/pair-slug` (#000027) summarized "what is this session about" by feeding
+the last N text-bearing transcript turns to a small model. On a tool-heavy
+session that window is almost entirely assistant narration: a real Claude
+transcript had ~16 genuine user prompts vs ~200 assistant entries (most
+`user` entries carry only `tool_result` blocks, correctly dropped as
+text-less). Measured: the last 10 text-bearing turns were 10/10 assistant,
+0 user. So the slug tracked what the agent was *saying*, not what the user
+*asked for* — the orientation signal was pushed out of the window. The unit
+tests passed because their fixtures used only text-content messages, never
+the dominant `tool_result`-only user shape — green tests masked the bug.
+Caught in #000027 M1 review.
+
+**Rule.** When sampling a conversation transcript to infer user intent:
+- Don't take a flat tail of turns. Guarantee a minimum number of recent
+  *user* turns are in the window (extend backward until satisfied, capped).
+- Model test fixtures on the *real* transcript shape, including
+  `tool_result`-only user entries and any sidechain/summary types — not the
+  clean text-only case. A fixture that can't reproduce the bug can't guard
+  against it.
+
+## `gofmt -w <dir>` reformats files you didn't touch
+
+Running `gofmt -w cmd/pair-wrap/` to format M3's edited `main.go` also
+rewrote four pre-existing `*_test.go` files (struct-field alignment) that the
+milestone never touched, staging unrelated churn into the commit. Caught at
+`git status` review before commit; reverted with `git checkout -- <files>`.
+
+**Rule.** Format only the files the change actually touches: `gofmt -w
+path/to/file.go` (or `gofmt -w $(git diff --name-only '*.go')`), not the whole
+package directory. If a dir-wide gofmt lights up files outside the change,
+revert them — don't smuggle repo-wide reformatting into a feature commit.
+Caught in #000027 M3.
+
+## Dogfooding a Go-binary change needs `make install`, not just `make build`
+
+M3's pair-wrap trigger "didn't fire" on restart. Trace: pair-slug worked in
+isolation, but the running `pair-wrap` (pid via `pair-wrap-pid-<tag>`, binary
+via `lsof -p <pid> | awk '$4=="txt"`) was `~/.local/bin/pair-wrap` dated days
+earlier — the *installed* copy, with no spawn. I had only `go build -o bin/…`;
+the layout (`zellij/layouts/main.kdl`) execs `pair-wrap` by bare name and the
+pane's PATH resolved `~/.local/bin` first.
+
+**Rule.** `bin/` is the repo build; `~/.local/bin` (via `make install`) is what
+actually runs in a live pair session. To dogfood a change to a Go binary
+(pair-wrap/pair-slug/…): `make install`, *then* restart pair. Verifying with
+`bin/<binary>` alone proves nothing about the running session. When a "live"
+change seems inert, confirm the running binary: `lsof -p $(cat
+$PAIR_DATA_DIR/pair-wrap-pid-<tag>) | awk '$4=="txt"{print $NF}'`. Caught in
+#000027 M3 dogfood.
+
+## Queue items: resolve by filename key, not display index, across a mutation
+
+Sending from a future-queue slot (`+N`) while the draft `*` was non-empty left
+the sent item in BOTH the queue (`+N`) and history (`-1`). Root cause:
+`send_and_clear` resolved the item to remove via `queue_key_for_n(nav.pos.n)` —
+the *display index* — but the new "park the draft into the queue first"
+(`push_front`) step shifts every index by one. Resolving by the stale index
+then removed the wrong file (or `nil`), so the actually-sent item was logged to
+history but never deleted from the queue → duplication.
+
+**Rule.** A `+N` display index is only valid against the queue snapshot it was
+read from. The moment any queue mutation (`push_front`/`push_back`/remove) can
+intervene, capture the item's **filename key first** (`queue_key_for_n(n)` →
+`NNNNNN`), then mutate, then remove by that stable key. Keys don't move on
+insert; indices do. Verified the duplication via a headless driver
+(`nvim --headless -u nvim/init.lua` + `maparg().callback`) before fixing, and
+guarded it with `tests/queue-send-test.sh` (`make test-queue`).
+
+## strings.ToLower can change byte length — don't cross-index a folded copy
+
+`promptShape` matched against `strings.ToLower(visible)` but then sliced the
+**original** `visible` at the match offset. `ToLower` is not length-preserving
+(e.g. `Ⱥ` U+023A, 2 bytes → `ⱥ` U+2C65, 3 bytes), so on agent output with such a
+rune the offset exceeded `len(visible)` and panicked the slice. The panic was
+swallowed by `handleChunk`'s `recover`, but that `recover` wraps the whole
+detect block, so OSC-notification + bell handling were silently skipped for that
+chunk — a diagnostic-only feature altering proxy behavior. Surfaced in #000045
+M1 review (C1).
+
+**Rule.** If you compute a byte offset in one string, slice the *same* string —
+never a transformed copy whose length can differ. For case-insensitive matching
+where you need offsets back in the original, use a **length-preserving** fold
+(ASCII-only `asciiFold`) and clamp slice indices defensively. Add a multibyte
+test case (`Ⱥ`/`İ`/`Å`) — ASCII-only tests can't catch this.
+
+## jq slurp (`-s`) over a JSONL file aborts on one bad line
+
+`doctor.sh` read the flight recorder with `jq -rs '…'`, which parses the whole
+file as one array — so a single malformed/partial line (a writer crashing
+mid-line; O_APPEND only guarantees atomicity below PIPE_BUF) made jq error and,
+under `set -euo pipefail`, killed the script. The operator got a jq stack trace
+and zero diagnostics exactly when they needed the tool. Surfaced in #000045 M1
+review (I1).
+
+---
+id: 000077
+status: done
+deps: [000074, 000075, 000076]
+github_issue:
+created: 2026-06-26
+updated: 2026-06-30
+estimate_hours: 2.6
+started: 2026-06-30T12:42:11-07:00
+actual_hours: 0.59
+---
+
+# pair Go entrypoint switch
+
+## Problem
+
+At some point the public `pair` command must become Go-owned. The next safe step is to make the Go-owned `pair-go launch ...` path exercise the real launcher contract while leaving the existing `pair` and `pair-dev` entrypoints stable.
+
+## Spec
+
+Make `pair-go launch ...` a meaningful Go entrypoint by having it hand off to the existing `bin/pair` launcher with `pair`-compatible arguments. `pair-go launch claude` should behave like `pair claude`; `pair-go launch resume <tag>`, `pair-go launch continue ...`, `pair-go launch list`, and `pair-go launch rename ...` should all pass through to the same shell-owned implementation for this migration window.
+
+Keep `pair` and `pair-dev` working exactly as they do today. The Go command has no separate `-dev` variant: a developer shell sourced from `../ariadne/construct/dev-aliases.sh` already discovers `cmd/pair-go`, rebuilds `bin/pair-go` on every invocation, and then runs it from the caller's cwd. If `pair-go launch` cannot find the launcher beside the built binary, its diagnostic should point to `make build` / `make install` and the dev-alias path instead of failing with a bare exec error.
+
+This deliberately keeps real zellij lifecycle, prompt/fzf UI, restart/quit cleanup, continuation, rename/list, and dev rebuild behavior shell-owned until later migration issues. `ARCH-PURPOSE`: #77's purpose is now the first meaningful Go-owned launch entrypoint without destabilizing the existing public command; full public `pair` replacement remains a later cutover once shell stateful glue is reduced.
+
+## Done when
+
+- [x] `pair-go launch ...` uses Go process code first and then invokes the real launcher with `pair`-compatible argv.
+- [x] Existing `pair` remains the stable public entrypoint for one migration window.
+- [x] `pair-dev` still rebuilds and launches the working tree behavior.
+- [x] Existing create, attach, resume, continue, rename/list, quit, and restart flows are preserved through the `bin/pair` fallback.
+- [x] The dev workflow is documented: `cmd/pair-go` is rebuilt by `../ariadne/construct/dev-aliases.sh`; no `pair-go-dev` command is needed.
+- [x] Pair remains usable after merge; no keybinding workflow regresses.
+
+## Plan
+
+- [x] Confirm prerequisites from earlier Go migration issues.
+- [x] Add tests for `pair-go launch` argv/env handoff to `bin/pair`.
+- [x] Add stale/missing launcher diagnostics.
+- [x] Implement the thin Go handoff while keeping dispatcher helper routes intact.
+- [x] Verify `pair`, `pair-dev`, and `pair-go launch` behavior with process fakes and targeted builds.
+- [x] Update README/atlas packaging notes.
+
+## Estimate
+
+Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.
+
+```estimate
+model: estimate-logic-v3.1
+familiarity: 1.0
+item: issue-spec design=0.25 impl=0.15
+item: greenfield-go-module design=0.35 impl=0.40
+item: skill-or-dispatcher design=0.30 impl=0.35
+item: atlas-docs design=0.20 impl=0.25
+item: milestone-review design=0.00 impl=0.20
+design-buffer: 0.15
+total: 2.62
+```
+
+## Log
+
+### 2026-06-26
+
+Created from #72 as the public switch milestone. This should not be claimed until the earlier dispatcher/helper/launcher milestones have landed.
+
+### 2026-06-30
+- 2026-06-30: closed — go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1; go test ./... -count=1; make pair-go; make test-dev-rebuild; bin/pair-go help; bin/pair-go launch --help; bin/pair --help; bin/pair-dev --help
+
+Re-scoped after operator guidance: keep `cmd/pair-go` as the Go entrypoint under test, leave `pair` / `pair-dev` stable, and rely on `../ariadne/construct/dev-aliases.sh` to rebuild `cmd/pair-go` in developer shells. `ARCH-DRY`: reuse the existing launcher for real zellij behavior instead of duplicating shell-owned lifecycle paths in Go. `ARCH-PURE`: keep launch path selection testable with a pure path/argv decision plus a thin exec boundary.
+
+Plan-quality gate returned FAILURE because the plan promised argv/env handoff but did not explicitly test env propagation, and because `pair-dev --help` under-proved the dev rebuild acceptance criterion. Updated the durable plan to require an inherited-env fake-runner assertion and `make test-dev-rebuild` verification. `ARCH-PURPOSE`: compatibility claims must be pinned by tests, not implied by the shell fallback.
+
+Implemented `cmd/internal/entrypoint` and the `cmd/pair-go` launch handoff. `pair-go launch ...` now resolves sibling `bin/pair`, validates it, and execs it with preserved argv/env; missing launcher diagnostics point to `make build`, `make install`, and `../ariadne/construct/dev-aliases.sh`. Removed the stale decision-phase `pair-go launch` dispatcher surface while keeping `context` and `scrollback-render` helper routes intact. Updated README and atlas migration notes.
+
+Verification passed:
+
+- `go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1`
+- `go test ./... -count=1`
+- `make pair-go`
+- `make test-dev-rebuild`
+- `bin/pair-go help`
+- `bin/pair-go launch --help`
+- `bin/pair --help`
+- `bin/pair-dev --help`
+
+codex
+Using `superpowers-using-superpowers` for the mandatory session skill check, and treating this as the requested fresh-context SDLC boundary review. I’ve read the issue, SDLC contract, and lessons; now I’m checking the actual repository state and behavior against the diff.
+exec
+/bin/zsh -lc "nl -ba cmd/pair-go/main.go | sed -n '1,180p'" in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+     1	// pair-go is the development dispatcher for the future primary Go CLI. Its
+     2	// launch route is a compatibility handoff to the current shell launcher.
+     3	package main
+     4	
+     5	import (
+     6		"fmt"
+     7		"io"
+     8		"os"
+     9		"syscall"
+    10	
+    11		"github.com/xianxu/pair/cmd/internal/dispatcher"
+    12		"github.com/xianxu/pair/cmd/internal/entrypoint"
+    13	)
+    14	
+    15	func main() {
+    16		os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
+    17	}
+    18	
+    19	func run(args []string, stdout, stderr io.Writer) int {
+    20		return runWithLegacyRuntime(args, stdout, stderr, osLegacyRuntime{})
+    21	}
+    22	
+    23	type legacyRuntime interface {
+    24		Executable() (string, error)
+    25		Stat(path string) error
+    26		Environ() []string
+    27		Exec(path string, argv []string, env []string) int
+    28	}
+    29	
+    30	func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
+    31		if len(args) > 0 && args[0] == "launch" {
+    32			return runLegacyLaunch(args[1:], stderr, rt)
+    33		}
+    34		res := dispatcher.Dispatch(args)
+    35		return writeResult(res, stdout, stderr)
+    36	}
+    37	
+    38	func runLegacyLaunch(args []string, stderr io.Writer, rt legacyRuntime) int {
+    39		exe, err := rt.Executable()
+    40		if err != nil {
+    41			_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
+    42			return 1
+    43		}
+    44		req := entrypoint.ResolveLegacyLaunch(exe, args)
+    45		if err := rt.Stat(req.Path); err != nil {
+    46			_, _ = fmt.Fprintf(stderr, "pair-go launch: pair launcher not found at %s (%v); run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", req.Path, err)
+    47			return 1
+    48		}
+    49		return rt.Exec(req.Path, req.Argv, rt.Environ())
+    50	}
+    51	
+    52	type osLegacyRuntime struct{}
+    53	
+    54	func (osLegacyRuntime) Executable() (string, error) {
+    55		return os.Executable()
+    56	}
+    57	
+    58	func (osLegacyRuntime) Stat(path string) error {
+    59		info, err := os.Stat(path)
+    60		if err != nil {
+    61			return err
+    62		}
+    63		if info.IsDir() {
+    64			return fmt.Errorf("is a directory")
+    65		}
+    66		return nil
+    67	}
+    68	
+    69	func (osLegacyRuntime) Environ() []string {
+    70		return os.Environ()
+    71	}
+    72	
+    73	func (osLegacyRuntime) Exec(path string, argv []string, env []string) int {
+    74		if err := syscall.Exec(path, argv, env); err != nil {
+    75			_, _ = fmt.Fprintf(os.Stderr, "pair-go launch: exec %s failed: %v\n", path, err)
+    76			return 1
+    77		}
+    78		return 0
+    79	}
+    80	
+    81	func writeResult(res dispatcher.Result, stdout, stderr io.Writer) int {
+    82		if res.Stdout != "" {
+    83			_, _ = io.WriteString(stdout, res.Stdout)
+    84		}
+    85		if res.Stderr != "" {
+    86			_, _ = io.WriteString(stderr, res.Stderr)
+    87		}
+    88		return res.ExitCode
+    89	}
+
+exec
+/bin/zsh -lc "nl -ba cmd/internal/entrypoint/launch.go && nl -ba cmd/internal/dispatcher/dispatcher.go | sed -n '1,150p'" in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+     1	package entrypoint
+     2	
+     3	import "path/filepath"
+     4	
+     5	// LegacyLaunchRequest describes the current compatibility handoff from
+     6	// pair-go launch to the shell-owned pair launcher.
+     7	type LegacyLaunchRequest struct {
+     8		Path string
+     9		Argv []string
+    10	}
+    11	
+    12	func ResolveLegacyLaunch(executable string, launchArgs []string) LegacyLaunchRequest {
+    13		argv := make([]string, 0, len(launchArgs)+1)
+    14		argv = append(argv, "pair")
+    15		argv = append(argv, launchArgs...)
+    16		return LegacyLaunchRequest{
+    17			Path: filepath.Join(filepath.Dir(executable), "pair"),
+    18			Argv: argv,
+    19		}
+    20	}
+     1	package dispatcher
+     2	
+     3	import (
+     4		"bytes"
+     5		"fmt"
+     6		"strings"
+     7	
+     8		"github.com/xianxu/pair/cmd/internal/contextcmd"
+     9		"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
+    10	)
+    11	
+    12	const programName = "pair-go"
+    13	
+    14	// CommandFamily names a future Pair CLI surface without claiming it works yet.
+    15	type CommandFamily struct {
+    16		Name    string
+    17		Summary string
+    18		Status  string
+    19	}
+    20	
+    21	// Result is the process-facing outcome of a pure dispatch decision.
+    22	type Result struct {
+    23		Stdout   string
+    24		Stderr   string
+    25		ExitCode int
+    26	}
+    27	
+    28	// Families returns the planned command families for the Go dispatcher.
+    29	func Families() []CommandFamily {
+    30		return []CommandFamily{
+    31			{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "handoff"},
+    32			{Name: "context", Summary: "agent pane context meter", Status: "implemented"},
+    33			{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "implemented"},
+    34			{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
+    35			{Name: "slug", Summary: "session orientation slug generation", Status: "planned"},
+    36			{Name: "changelog", Summary: "TTY transcript to distilled change log", Status: "planned"},
+    37			{Name: "continuation", Summary: "continuation datatype writer", Status: "planned"},
+    38			{Name: "scribe", Summary: "PTY logging wrapper", Status: "planned"},
+    39		}
+    40	}
+    41	
+    42	// Dispatch parses argv and returns the skeleton dispatch result.
+    43	func Dispatch(args []string) Result {
+    44		if len(args) == 0 {
+    45			return Result{Stdout: Help(programName), ExitCode: 0}
+    46		}
+    47	
+    48		switch args[0] {
+    49		case "help", "--help", "-h":
+    50			return Result{Stdout: Help(programName), ExitCode: 0}
+    51		case "version", "--version":
+    52			return Result{
+    53				Stdout:   "pair-go dispatcher skeleton\nlaunch handoff: bin/pair\n",
+    54				ExitCode: 0,
+    55			}
+    56		case "launch":
+    57			return launchHandoffResult()
+    58		case "context":
+    59			return dispatchContext(args[1:])
+    60		case "scrollback-render":
+    61			return dispatchScrollbackRender(args[1:])
+    62		}
+    63	
+    64		if family, ok := familyByName(args[0]); ok {
+    65			return Result{
+    66				Stderr:   fmt.Sprintf("%s: %s is planned but not implemented in this skeleton; run %s help\n", programName, family.Name, programName),
+    67				ExitCode: 2,
+    68			}
+    69		}
+    70	
+    71		return Result{
+    72			Stderr:   fmt.Sprintf("%s: unknown command %q; run %s help\n", programName, args[0], programName),
+    73			ExitCode: 2,
+    74		}
+    75	}
+    76	
+    77	func dispatchContext(args []string) Result {
+    78		var stdout bytes.Buffer
+    79		code := contextcmd.Run(args, contextcmd.EnvFromOS(), &stdout)
+    80		return Result{Stdout: stdout.String(), ExitCode: code}
+    81	}
+    82	
+    83	func dispatchScrollbackRender(args []string) Result {
+    84		var stdout, stderr bytes.Buffer
+    85		code := scrollbackcmd.Run(args, &stdout, &stderr)
+    86		return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}
+    87	}
+    88	
+    89	func launchHandoffResult() Result {
+    90		return Result{
+    91			Stderr:   "pair-go launch is a process handoff implemented by cmd/pair-go; call pair-go launch ... instead of dispatcher.Dispatch\n",
+    92			ExitCode: 2,
+    93		}
+    94	}
+    95	
+    96	// Help renders the development-only dispatcher usage text.
+    97	func Help(program string) string {
+    98		var b strings.Builder
+    99		fmt.Fprintf(&b, "Usage: %s <command> [args]\n\n", program)
+   100		b.WriteString("Development dispatcher skeleton. Public sessions still start through bin/pair.\n\n")
+   101		b.WriteString("Implemented commands:\n")
+   102		for _, family := range Families() {
+   103			if family.Status == "prototype" {
+   104				fmt.Fprintf(&b, "  %-17s %s (prototype; decision-phase only)\n", family.Name, family.Summary)
+   105			} else if family.Status == "handoff" {
+   106				fmt.Fprintf(&b, "  %-17s %s (compatibility handoff to bin/pair)\n", family.Name, family.Summary)
+   107			} else if family.Status == "implemented" {
+   108				fmt.Fprintf(&b, "  %-17s %s (implemented helper route)\n", family.Name, family.Summary)
+   109			}
+   110		}
+   111		b.WriteString("\nPlanned command families (not implemented in this skeleton):\n")
+   112		for _, family := range Families() {
+   113			if family.Status == "planned" {
+   114				fmt.Fprintf(&b, "  %-17s %s (%s; not implemented in this skeleton)\n", family.Name, family.Summary, family.Status)
+   115			}
+   116		}
+   117		b.WriteString("\nSupported skeleton commands:\n")
+   118		b.WriteString("  help              show this help\n")
+   119		b.WriteString("  version           show dispatcher skeleton metadata\n")
+   120		return b.String()
+   121	}
+   122	
+   123	func familyByName(name string) (CommandFamily, bool) {
+   124		for _, family := range Families() {
+   125			if family.Name == name {
+   126				return family, true
+   127			}
+   128		}
+   129		return CommandFamily{}, false
+   130	}
+
+exec
+/bin/zsh -lc "nl -ba cmd/pair-go/main_test.go | sed -n '1,220p' && nl -ba cmd/internal/entrypoint/launch_test.go" in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+     1	package main
+     2	
+     3	import (
+     4		"bytes"
+     5		"errors"
+     6		"os"
+     7		"reflect"
+     8		"strings"
+     9		"testing"
+    10	)
+    11	
+    12	func TestRunWritesStdoutAndReturnsDispatcherCode(t *testing.T) {
+    13		var stdout, stderr bytes.Buffer
+    14		code := run([]string{"help"}, &stdout, &stderr)
+    15		if code != 0 {
+    16			t.Fatalf("code = %d, want 0", code)
+    17		}
+    18		if !strings.Contains(stdout.String(), "Usage: pair-go <command> [args]") {
+    19			t.Fatalf("stdout missing usage:\n%s", stdout.String())
+    20		}
+    21		if stderr.String() != "" {
+    22			t.Fatalf("stderr = %q, want empty", stderr.String())
+    23		}
+    24	}
+    25	
+    26	func TestRunWritesStderrAndReturnsDispatcherCode(t *testing.T) {
+    27		var stdout, stderr bytes.Buffer
+    28		code := run([]string{"wrap"}, &stdout, &stderr)
+    29		if code != 2 {
+    30			t.Fatalf("code = %d, want 2", code)
+    31		}
+    32		if stdout.String() != "" {
+    33			t.Fatalf("stdout = %q, want empty", stdout.String())
+    34		}
+    35		if !strings.Contains(stderr.String(), "wrap is planned but not implemented") {
+    36			t.Fatalf("stderr missing unsupported-command message:\n%s", stderr.String())
+    37		}
+    38	}
+    39	
+    40	func TestRunLaunchHelp(t *testing.T) {
+    41		rt := &fakeLegacyRuntime{
+    42			executable: "/repo/bin/pair-go",
+    43		}
+    44		var stdout, stderr bytes.Buffer
+    45		code := runWithLegacyRuntime([]string{"launch", "--help"}, &stdout, &stderr, rt)
+    46		if code != 0 {
+    47			t.Fatalf("code = %d, want 0", code)
+    48		}
+    49		if stdout.String() != "" {
+    50			t.Fatalf("stdout = %q, want empty", stdout.String())
+    51		}
+    52		if stderr.String() != "" {
+    53			t.Fatalf("stderr = %q, want empty", stderr.String())
+    54		}
+    55		if rt.execPath != "/repo/bin/pair" {
+    56			t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
+    57		}
+    58		wantArgv := []string{"pair", "--help"}
+    59		if !reflect.DeepEqual(rt.execArgv, wantArgv) {
+    60			t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
+    61		}
+    62	}
+    63	
+    64	func TestRunLaunchExecsLegacyPairWithArgvAndEnv(t *testing.T) {
+    65		t.Setenv("PAIR_TEST_ENV", "kept")
+    66		rt := &fakeLegacyRuntime{
+    67			executable: "/repo/bin/pair-go",
+    68			execCode:   42,
+    69		}
+    70	
+    71		var stdout, stderr bytes.Buffer
+    72		code := runWithLegacyRuntime([]string{"launch", "claude", "--", "--resume"}, &stdout, &stderr, rt)
+    73	
+    74		if code != 42 {
+    75			t.Fatalf("code = %d, want 42", code)
+    76		}
+    77		if stdout.String() != "" {
+    78			t.Fatalf("stdout = %q, want empty", stdout.String())
+    79		}
+    80		if stderr.String() != "" {
+    81			t.Fatalf("stderr = %q, want empty", stderr.String())
+    82		}
+    83		if rt.execPath != "/repo/bin/pair" {
+    84			t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
+    85		}
+    86		wantArgv := []string{"pair", "claude", "--", "--resume"}
+    87		if !reflect.DeepEqual(rt.execArgv, wantArgv) {
+    88			t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
+    89		}
+    90		if !containsEnv(rt.execEnv, "PAIR_TEST_ENV=kept") {
+    91			t.Fatalf("execEnv missing PAIR_TEST_ENV=kept: %#v", rt.execEnv)
+    92		}
+    93	}
+    94	
+    95	func TestRunLaunchReportsMissingLegacyPair(t *testing.T) {
+    96		rt := &fakeLegacyRuntime{
+    97			executable: "/repo/bin/pair-go",
+    98			statErr:    os.ErrNotExist,
+    99		}
+   100	
+   101		var stdout, stderr bytes.Buffer
+   102		code := runWithLegacyRuntime([]string{"launch", "claude"}, &stdout, &stderr, rt)
+   103	
+   104		if code != 1 {
+   105			t.Fatalf("code = %d, want 1", code)
+   106		}
+   107		if stdout.String() != "" {
+   108			t.Fatalf("stdout = %q, want empty", stdout.String())
+   109		}
+   110		for _, want := range []string{"pair-go launch", "/repo/bin/pair", "make build", "make install", "dev-aliases.sh"} {
+   111			if !strings.Contains(stderr.String(), want) {
+   112				t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
+   113			}
+   114		}
+   115		if rt.execPath != "" {
+   116			t.Fatalf("execPath = %q, want empty", rt.execPath)
+   117		}
+   118	}
+   119	
+   120	type fakeLegacyRuntime struct {
+   121		executable string
+   122		statErr    error
+   123		execCode   int
+   124	
+   125		execPath string
+   126		execArgv []string
+   127		execEnv  []string
+   128	}
+   129	
+   130	func (f *fakeLegacyRuntime) Executable() (string, error) {
+   131		if f.executable == "" {
+   132			return "", errors.New("missing executable")
+   133		}
+   134		return f.executable, nil
+   135	}
+   136	
+   137	func (f *fakeLegacyRuntime) Stat(_ string) error {
+   138		return f.statErr
+   139	}
+   140	
+   141	func (f *fakeLegacyRuntime) Environ() []string {
+   142		return os.Environ()
+   143	}
+   144	
+   145	func (f *fakeLegacyRuntime) Exec(path string, argv []string, env []string) int {
+   146		f.execPath = path
+   147		f.execArgv = append([]string(nil), argv...)
+   148		f.execEnv = append([]string(nil), env...)
+   149		return f.execCode
+   150	}
+   151	
+   152	func containsEnv(env []string, want string) bool {
+   153		for _, got := range env {
+   154			if got == want {
+   155				return true
+   156			}
+   157		}
+   158		return false
+   159	}
+     1	package entrypoint
+     2	
+     3	import (
+     4		"reflect"
+     5		"testing"
+     6	)
+     7	
+     8	func TestResolveLegacyLaunchDropsLaunchVerb(t *testing.T) {
+     9		req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"claude", "--", "--resume"})
+    10		if req.Path != "/repo/bin/pair" {
+    11			t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
+    12		}
+    13		want := []string{"pair", "claude", "--", "--resume"}
+    14		if !reflect.DeepEqual(req.Argv, want) {
+    15			t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
+    16		}
+    17	}
+    18	
+    19	func TestResolveLegacyLaunchPreservesSubcommands(t *testing.T) {
+    20		req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"resume", "demo"})
+    21		if req.Path != "/repo/bin/pair" {
+    22			t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
+    23		}
+    24		want := []string{"pair", "resume", "demo"}
+    25		if !reflect.DeepEqual(req.Argv, want) {
+    26			t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
+    27		}
+    28	}
+
+exec
+/bin/zsh -lc 'git diff --stat b829ef00346738e3c3d7e40db38c2c03f51e615b..HEAD && git diff --name-only b829ef00346738e3c3d7e40db38c2c03f51e615b..HEAD' in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+ Makefile.local                                 |   7 +-
+ README.md                                      |   7 ++
+ atlas/architecture.md                          |  24 +++---
+ atlas/go-migration-inventory.md                |   6 +-
+ cmd/internal/dispatcher/dispatcher.go          |  97 ++-------------------
+ cmd/internal/dispatcher/dispatcher_test.go     |  57 ++-----------
+ cmd/internal/entrypoint/launch.go              |  20 +++++
+ cmd/internal/entrypoint/launch_test.go         |  28 ++++++
+ cmd/pair-go/launch_process_test.go             |  71 ---------------
+ cmd/pair-go/main.go                            |  65 ++++++++++++--
+ cmd/pair-go/main_test.go                       | 114 ++++++++++++++++++++-----
+ workshop/issues/000077-go-entrypoint-switch.md |  37 +++++---
+ 12 files changed, 270 insertions(+), 263 deletions(-)
+Makefile.local
+README.md
+atlas/architecture.md
+atlas/go-migration-inventory.md
+cmd/internal/dispatcher/dispatcher.go
+cmd/internal/dispatcher/dispatcher_test.go
+cmd/internal/entrypoint/launch.go
+cmd/internal/entrypoint/launch_test.go
+cmd/pair-go/launch_process_test.go
+cmd/pair-go/main.go
+cmd/pair-go/main_test.go
+workshop/issues/000077-go-entrypoint-switch.md
+
+codex
+The core handoff is small: `cmd/pair-go` special-cases `launch`, computes a sibling `pair`, and `exec`s it; non-launch helper routes stay in the dispatcher. I’m checking whether that matches how this repo actually installs and invokes binaries, because that’s the main place this kind of entrypoint change can break despite green unit tests.
+exec
+/bin/zsh -lc "nl -ba Makefile.local | sed -n '210,270p' && nl -ba Makefile | sed -n '1,220p'" in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+   210	# pair-scrollback-render — replays a pair-wrap raw PTY capture through
+   211	# charmbracelet/x/vt and emits one ANSI-styled line per logical scrollback
+   212	# row, for the Alt+/ viewer to display through nvim.
+   213	$(BIN_DIR)/pair-scrollback-render: cmd/pair-scrollback-render/main.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
+   214		go build -o $@ ./cmd/pair-scrollback-render
+   215	
+   216	# pair-wrap — transparent PTY proxy around a TUI coding agent: forwards
+   217	# stdio, tees scrollback log, extracts colored spans for nvim autocomplete,
+   218	# detects OSC/BEL and forwards as OSC 9 to the recorded outer-TTY, handles
+   219	# SIGUSR1 image-paste capture window.
+   220	$(BIN_DIR)/pair-wrap: cmd/pair-wrap/main.go go.mod
+   221		go build -o $@ ./cmd/pair-wrap
+   222	
+   223	$(BIN_DIR)/pair-slug: cmd/pair-slug/main.go cmd/pair-slug/slug.go cmd/internal/model/model.go cmd/internal/transcript/transcript.go go.mod
+   224		go build -o $@ ./cmd/pair-slug
+   225	
+   226	# pair-context — one-shot: print a pane's current context size (token count)
+   227	# for the pair-title poller. Pure ctxmeter reader + shared transcript resolver.
+   228	$(BIN_DIR)/pair-context: cmd/pair-context/main.go cmd/internal/contextcmd/contextcmd.go cmd/internal/ctxmeter/ctxmeter.go cmd/internal/transcript/transcript.go go.mod
+   229		go build -o $@ ./cmd/pair-context
+   230	
+   231	# pair-changelog — distills the pair TTY into the Alt+l change log (#53);
+   232	# shares cmd/internal/model with pair-slug.
+   233	$(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill.go cmd/pair-changelog/prompt.go cmd/internal/model/model.go go.mod
+   234		go build -o $@ ./cmd/pair-changelog
+   235	
+   236	# pair-continuation — the deterministic writer for the `continuation` datatype
+   237	# (ariadne#91): renders frontmatter, allocates a collision-safe timestamped
+   238	# name under workshop/continuation/, writes, then commits + pushes (recovery).
+   239	$(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-continuation/git.go cmd/pair-continuation/main.go go.mod
+   240		go build -o $@ ./cmd/pair-continuation
+   241	
+   242	# pair-go — development dispatcher and Go-owned launch handoff.
+   243	# `pair-go launch ...` execs sibling bin/pair for the current migration window;
+   244	# helper routes stay in the dispatcher.
+   245	$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/entrypoint/launch.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
+   246		go build -o $@ ./cmd/pair-go
+   247	
+   248	
+   249	# Tart VM targets (tart, tart-gui, tart-mount, tart-stop, tart-clean)
+   250	# live in .tart/Makefile, vendored from ariadne and auto-included by
+   251	# Makefile.workflow. `make help-tart` for usage.
+     1	# Canonical repo name from git remote (portable across worktrees and containers)
+     2	REPO_NAME := $(shell git remote get-url origin 2>/dev/null | sed 's|.*/||; s|\.git$$||')
+     3	
+     4	# This project nests issues and history under workshop/
+     5	WF_ISSUES_DIR = workshop/issues
+     6	WF_HISTORY_DIR = workshop/history
+     7	
+     8	# Assemble sub-Makefiles (Makefile.workflow already includes .openshell/Makefile)
+     9	include Makefile.workflow
+    10	-include Makefile.local
+    11	
+    12	.PHONY: help
+    13	
+    14	# help-sandbox, help-tart, and help-colima are defined by .openshell/Makefile,
+    15	# .tart/Makefile, and .colima/Makefile respectively, all included via
+    16	# Makefile.workflow's -include lines. Every consumer that vendors the ariadne
+    17	# base layer ships all three fragments (see construct/base.manifest), so these
+    18	# targets always resolve. If a consumer ever drops .openshell, .tart, or
+    19	# .colima from its manifest, the corresponding help-X line would need to come
+    20	# out. Transient window: a consumer that pulls this updated Makefile before
+    21	# running setup.sh to materialize the new .colima/Makefile symlink will get a
+    22	# "No rule to make target 'help-colima'" until setup runs — same accepted
+    23	# fragility the other two already carry.
+    24	help: help-workflow help-sandbox help-tart help-colima
+    25		@true
+
+exec
+/bin/zsh -lc 'rg -n "pair-go|pair-dev|install|BIN_DIR|dev-aliases|PAIR_DEV|cmd/pair-go" Makefile Makefile.local bin cmd README.md atlas workshop/issues/000077-go-entrypoint-switch.md' in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+README.md:141:Automatically installed with `homebrew`.
+README.md:171:# install
+README.md:172:brew tap xianxu/pair && brew install pair
+README.md:178:That installs `zellij`, `neovim`, `fzf`, `jq`, and `par` if they aren't already present. The agent (`claude`, `codex`, `agy`) you install separately. Then:
+README.md:203:`pair-go launch ...` is the Go-owned migration entrypoint for testing the
+README.md:206:window. In a dev shell sourced from `../ariadne/construct/dev-aliases.sh`,
+README.md:207:`pair-go` rebuilds from `cmd/pair-go` automatically before running; no
+README.md:208:`pair-go-dev` command is needed.
+README.md:214:**Hacking on pair?** Use `pair-dev` instead of `pair` — same arguments, but it rebuilds pair's Go binaries from source (`make build`) on launch *and* on every Alt+n / Shift+Alt+N restart, so the zellij-spawned `pair-wrap` always matches your working tree. (Deployed installs run `pair`, which uses the prebuilt binary and needs no Go toolchain.)
+Makefile.local:4:.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
+Makefile.local:12:# `make install` pick it up automatically.
+Makefile.local:15:#   make install    → also copy each into ~/.local/bin/
+Makefile.local:21:# in both dev (source tree) and brew-install layouts.
+Makefile.local:23:# Why ~/.local/bin/ for install: XDG convention, on PATH per the user's
+Makefile.local:27:BIN_DIR     := bin
+Makefile.local:29:GO_BINS     := pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-go
+Makefile.local:31:build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
+Makefile.local:33:install: build
+Makefile.local:36:	    install -m 755 $(BIN_DIR)/$$b $(INSTALL_DIR)/$$b && \
+Makefile.local:37:	    echo "installed: $(INSTALL_DIR)/$$b"; \
+Makefile.local:41:pair-scribe:            $(BIN_DIR)/pair-scribe
+Makefile.local:42:pair-scrollback-render: $(BIN_DIR)/pair-scrollback-render
+Makefile.local:43:pair-wrap:              $(BIN_DIR)/pair-wrap
+Makefile.local:44:pair-slug:              $(BIN_DIR)/pair-slug
+Makefile.local:45:pair-continuation:      $(BIN_DIR)/pair-continuation
+Makefile.local:46:pair-changelog:         $(BIN_DIR)/pair-changelog
+Makefile.local:47:pair-context:           $(BIN_DIR)/pair-context
+Makefile.local:48:pair-go:                $(BIN_DIR)/pair-go
+Makefile.local:178:# pair-dev rebuild hook (#000046): deployed mode (PAIR_DEV unset) must invoke
+Makefile.local:199:test-changelog: $(BIN_DIR)/pair-changelog $(BIN_DIR)/pair-scrollback-render
+Makefile.local:207:$(BIN_DIR)/pair-scribe: cmd/pair-scribe/main.go go.mod
+Makefile.local:213:$(BIN_DIR)/pair-scrollback-render: cmd/pair-scrollback-render/main.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
+Makefile.local:220:$(BIN_DIR)/pair-wrap: cmd/pair-wrap/main.go go.mod
+Makefile.local:223:$(BIN_DIR)/pair-slug: cmd/pair-slug/main.go cmd/pair-slug/slug.go cmd/internal/model/model.go cmd/internal/transcript/transcript.go go.mod
+Makefile.local:228:$(BIN_DIR)/pair-context: cmd/pair-context/main.go cmd/internal/contextcmd/contextcmd.go cmd/internal/ctxmeter/ctxmeter.go cmd/internal/transcript/transcript.go go.mod
+Makefile.local:233:$(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill.go cmd/pair-changelog/prompt.go cmd/internal/model/model.go go.mod
+Makefile.local:239:$(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-continuation/git.go cmd/pair-continuation/main.go go.mod
+Makefile.local:242:# pair-go — development dispatcher and Go-owned launch handoff.
+Makefile.local:243:# `pair-go launch ...` execs sibling bin/pair for the current migration window;
+Makefile.local:245:$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/entrypoint/launch.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
+Makefile.local:246:	go build -o $@ ./cmd/pair-go
+workshop/issues/000077-go-entrypoint-switch.md:17:At some point the public `pair` command must become Go-owned. The next safe step is to make the Go-owned `pair-go launch ...` path exercise the real launcher contract while leaving the existing `pair` and `pair-dev` entrypoints stable.
+workshop/issues/000077-go-entrypoint-switch.md:21:Make `pair-go launch ...` a meaningful Go entrypoint by having it hand off to the existing `bin/pair` launcher with `pair`-compatible arguments. `pair-go launch claude` should behave like `pair claude`; `pair-go launch resume <tag>`, `pair-go launch continue ...`, `pair-go launch list`, and `pair-go launch rename ...` should all pass through to the same shell-owned implementation for this migration window.
+workshop/issues/000077-go-entrypoint-switch.md:23:Keep `pair` and `pair-dev` working exactly as they do today. The Go command has no separate `-dev` variant: a developer shell sourced from `../ariadne/construct/dev-aliases.sh` already discovers `cmd/pair-go`, rebuilds `bin/pair-go` on every invocation, and then runs it from the caller's cwd. If `pair-go launch` cannot find the launcher beside the built binary, its diagnostic should point to `make build` / `make install` and the dev-alias path instead of failing with a bare exec error.
+workshop/issues/000077-go-entrypoint-switch.md:29:- [x] `pair-go launch ...` uses Go process code first and then invokes the real launcher with `pair`-compatible argv.
+workshop/issues/000077-go-entrypoint-switch.md:31:- [x] `pair-dev` still rebuilds and launches the working tree behavior.
+workshop/issues/000077-go-entrypoint-switch.md:33:- [x] The dev workflow is documented: `cmd/pair-go` is rebuilt by `../ariadne/construct/dev-aliases.sh`; no `pair-go-dev` command is needed.
+workshop/issues/000077-go-entrypoint-switch.md:39:- [x] Add tests for `pair-go launch` argv/env handoff to `bin/pair`.
+workshop/issues/000077-go-entrypoint-switch.md:42:- [x] Verify `pair`, `pair-dev`, and `pair-go launch` behavior with process fakes and targeted builds.
+workshop/issues/000077-go-entrypoint-switch.md:68:- 2026-06-30: closed — go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1; go test ./... -count=1; make pair-go; make test-dev-rebuild; bin/pair-go help; bin/pair-go launch --help; bin/pair --help; bin/pair-dev --help
+workshop/issues/000077-go-entrypoint-switch.md:70:Re-scoped after operator guidance: keep `cmd/pair-go` as the Go entrypoint under test, leave `pair` / `pair-dev` stable, and rely on `../ariadne/construct/dev-aliases.sh` to rebuild `cmd/pair-go` in developer shells. `ARCH-DRY`: reuse the existing launcher for real zellij behavior instead of duplicating shell-owned lifecycle paths in Go. `ARCH-PURE`: keep launch path selection testable with a pure path/argv decision plus a thin exec boundary.
+workshop/issues/000077-go-entrypoint-switch.md:72:Plan-quality gate returned FAILURE because the plan promised argv/env handoff but did not explicitly test env propagation, and because `pair-dev --help` under-proved the dev rebuild acceptance criterion. Updated the durable plan to require an inherited-env fake-runner assertion and `make test-dev-rebuild` verification. `ARCH-PURPOSE`: compatibility claims must be pinned by tests, not implied by the shell fallback.
+workshop/issues/000077-go-entrypoint-switch.md:74:Implemented `cmd/internal/entrypoint` and the `cmd/pair-go` launch handoff. `pair-go launch ...` now resolves sibling `bin/pair`, validates it, and execs it with preserved argv/env; missing launcher diagnostics point to `make build`, `make install`, and `../ariadne/construct/dev-aliases.sh`. Removed the stale decision-phase `pair-go launch` dispatcher surface while keeping `context` and `scrollback-render` helper routes intact. Updated README and atlas migration notes.
+workshop/issues/000077-go-entrypoint-switch.md:78:- `go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1`
+workshop/issues/000077-go-entrypoint-switch.md:80:- `make pair-go`
+workshop/issues/000077-go-entrypoint-switch.md:82:- `bin/pair-go help`
+workshop/issues/000077-go-entrypoint-switch.md:83:- `bin/pair-go launch --help`
+workshop/issues/000077-go-entrypoint-switch.md:85:- `bin/pair-dev --help`
+bin/pair:104:  `pair-dev` is identical to `pair` but rebuilds pair's Go binaries from
+bin/pair:107:  when hacking on cmd/*; deployed installs keep running `pair` (prebuilt,
+bin/pair:108:  no toolchain dependency). See bin/pair-dev + bin/lib/dev-rebuild.sh.
+bin/pair:376:        # offline housekeeping from a machine that never had it installed)
+bin/pair:933:    echo "      brew install zellij  (or your package manager equivalent)" >&2
+bin/pair:1707:    echo "      install it first, then re-run." >&2
+bin/pair:2238:# Dev mode (pair-dev exports PAIR_DEV): rebuild the Go binaries from source so
+bin/pair:2241:# with no toolchain dependency unless PAIR_DEV is set. This is the create path
+bin/pair:2242:# only, so a plain attach skips it; restart re-execs $0=bin/pair with PAIR_DEV
+cmd/pair-wrap/main.go:980:				// `make nous-install` into the unusable autocomplete
+cmd/pair-wrap/main.go:981:				// candidate `makenous-install`. Drop in a single-space
+atlas/architecture.md:17:bin/pair-go                  # opt-in Go dispatcher + launch handoff; public launcher remains bin/pair
+atlas/architecture.md:46:installed commands forever.
+atlas/architecture.md:48:As of #77, `bin/pair-go` is the Go-owned launch entrypoint under test:
+atlas/architecture.md:49:`pair-go launch ...` resolves its own executable, finds sibling `bin/pair`, and
+atlas/architecture.md:51:makes `pair-go launch claude`, `pair-go launch resume <tag>`, `pair-go launch
+atlas/architecture.md:52:continue ...`, `pair-go launch list`, and `pair-go launch rename ...`
+atlas/architecture.md:54:yet. A developer shell sourced from `../ariadne/construct/dev-aliases.sh`
+atlas/architecture.md:55:rebuilds `cmd/pair-go` automatically; no `pair-go-dev` command exists.
+atlas/architecture.md:63:`pair-go context <tag> <agent>` and `pair-go scrollback-render ...`. Both routes
+atlas/architecture.md:72:may embed those assets or install them adjacent to the binary, but the migration
+atlas/architecture.md:76:merge-safe: after any sub-issue lands, the public `pair` command, `pair-dev`,
+atlas/architecture.md:569:   - **Legacy snapshot-diff (fallback).** Used when a fresh pidfile doesn't appear within 2s (`PAIR_SESSION_WATCH_PID_WAIT_SECONDS` in tests) — i.e., when the installed pair-wrap binary predates #000020 and doesn't publish the pidfile, or a stale pidfile is never refreshed. Behaves identically to pre-#000020: snapshots the watch dir at start, picks the first new file. Cross-tag races re-emerge in this path, so the proper resolution is to rebuild pair-wrap.
+atlas/architecture.md:677:## Binary freshness: deployed vs dev (`pair-dev`)
+atlas/architecture.md:679:The Go binaries (`pair-wrap`, `pair-slug`, …) live in `$PAIR_HOME/bin` (first on PATH per *Path resolution* above) and, after `make install`, in `~/.local/bin`. `bin/` is **gitignored** — built on demand, absent in a fresh tree. Because the agent pane launches as `sh -c '… exec pair-wrap …'`, the wrapper is resolved by a **PATH lookup**: no shell function or `.zshenv` can intercept it (`exec` bypasses functions, and `sh` ≠ zsh), so `construct/dev-aliases.sh`'s rebuild-on-call freshness does **not** reach it. When `$PAIR_HOME/bin/pair-wrap` is stale or absent, PATH silently falls through to an old `~/.local/bin` copy and the running wrapper drifts from source — the failure mode is *silence*, not an error (diagnosed once via the #000045 flight recorder going quiet for every Go-emitted aspect while only nvim's Lua emitter still logged).
+atlas/architecture.md:682:- **Deployed** — `pair`. Runs whatever prebuilt binary PATH finds; zero toolchain dependency. Keep `~/.local/bin` current with `make install`.
+atlas/architecture.md:683:- **Dev** — `pair-dev` (#000046). Exports `PAIR_DEV=1` and execs `pair`; `bin/pair` then runs `make build` (via `bin/lib/dev-rebuild.sh`'s `dev_rebuild`) on the **create path**, before the layout execs pair-wrap, so `$PAIR_HOME/bin` holds a fresh build. Restart-safe: `PAIR_DEV` survives `handle_restart_marker`'s `exec "$0"`, so Alt+n / Shift+Alt+N rebuild too; a plain attach (no new wrapper spawned) correctly skips it. Deployed launches (`PAIR_DEV` unset) invoke no toolchain.
+atlas/architecture.md:685:`pair-doctor` *diagnoses* the same staleness `pair-dev` prevents: its emitter-health probe (`doctor/emitter-health.sh`, #000047) greps the *running* `pair-wrap`/`pair-slug` (resolved via the `pair-wrap-pid-<tag>` pidfile, else PATH) for its adapt signal strings and flags `[STALE]` when a binary has no logging code — turning the silent-emitter failure into a named finding.
+atlas/architecture.md:689:`cmd/pair-scribe` is a `script(1)` replacement that lives in the pair repo for build-system convenience but is not part of pair's runtime — it's user shell tooling, typically wired at the top of `~/.zshrc` to swap for `script -q -F`. The user's preexec/precmd hooks send `SIGUSR1`/`SIGUSR2` to pause/resume the on-disk typescript around commands whose output (e.g. TUI redraws) shouldn't be captured, enabling a clean "capture last command output" flow that pair can read back from `$_ZSH_SCRIPT_LOG`. Lives at `~/.local/bin/pair-scribe` after `make install`. Full design notes and the zshrc snippet: `cmd/pair-scribe/README.md`.
+atlas/architecture.md:695:- **Self-contained.** Uses `--config-dir` and `nvim -u` to fully isolate from the user's normal configs. No invasive install.
+atlas/go-migration-inventory.md:14:- installed or runtime-called artifacts under `bin/`, `bin/lib/`, `cmd/`,
+atlas/go-migration-inventory.md:16:- `Makefile`, `Makefile.local`, install/packaging docs, and process-level
+atlas/go-migration-inventory.md:42:- **P1**: reduces installed binary/script surface or stateful shell risk.
+atlas/go-migration-inventory.md:50:| `bin/pair` / `cmd/internal/launcher` / `cmd/internal/entrypoint` | Bash public launcher plus Go-owned launch handoff | user shell, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` (#77) resolves sibling `bin/pair` from the `pair-go` executable and execs it with `pair`-compatible argv/env. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves the compatibility handoff; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Go handoff entrypoint with `bin/pair` retained as stable public launcher and compat target through #77; real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
+atlas/go-migration-inventory.md:51:| `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling `pair`. | Resolves symlinks; depends on `bin/pair` and `bin/lib/dev-rebuild.sh`. | compat-shim, likely stays as developer wrapper until Go entrypoint has dev mode | P1 |
+atlas/go-migration-inventory.md:52:| `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
+atlas/go-migration-inventory.md:57:| `bin/pair-context` / `cmd/pair-context` / `cmd/internal/contextcmd` | Go binary plus shared runner | `bin/pair-title.sh`; development-only `pair-go context` | `pair-context <tag> <agent>` and `pair-go context <tag> <agent>` print the same humanized token count or nothing; tolerant exit 0 on failure. | Reads `PAIR_DATA_DIR`, `pane-<tag>-<agent>.json`, config, native transcripts. | implemented helper route in `pair-go context`; legacy binary retained while title poller calls it | P1 |
+atlas/go-migration-inventory.md:58:| `bin/pair-scrollback-render` / `cmd/pair-scrollback-render` / `cmd/internal/scrollbackcmd` | Go binary plus shared runner | `bin/pair-scrollback-open`, `bin/pair-changelog-open`, `nvim/scrollback.lua` refresh; development-only `pair-go scrollback-render` | `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out` and `pair-go scrollback-render ...`; nonzero on render/write failure. | Reads `.raw` and `.events.jsonl`; atomically writes `.ansi` or cleaned text. | implemented helper route in `pair-go scrollback-render`; legacy binary retained for shell/Lua callers | P0 |
+atlas/go-migration-inventory.md:61:| `bin/pair-scribe` / `cmd/pair-scribe` | Go binary | user shell rc outside Pair sessions | `pair-scribe -log PATH -- CMD [ARGS...]`; long-running PTY wrapper; SIGUSR1 pauses log, SIGUSR2 resumes. | Writes typescript log; wraps child PTY; independent of `PAIR_*`. | go-subcommand candidate is low value; may remain separate installed helper or become `pair scribe` with alias | P2 |
+atlas/go-migration-inventory.md:89:| `Makefile` | build/workflow entry | developer/CI/SDLC | Includes workflow and local makefiles; `help` target. | Uses git remote; includes vendored base fragments. | packaging metadata; keep, update in #79 if install layout changes | P1 |
+atlas/go-migration-inventory.md:90:| `Makefile.local` | build/install/test metadata | developer/CI/`pair-dev` | Builds and installs Go binaries, runs test matrix. | Writes `bin/` and `~/.local/bin`; invokes Go, nvim, shell tests. | build contract; #74/#76/#79 must update as dispatcher changes | P0 |
+atlas/go-migration-inventory.md:91:| `README.md` | install/usage docs | users/package consumers | Homebrew install, CLI usage, keybindings, dev mode. | Documents dependencies and public commands. | docs to update at #77/#79 | P1 |
+atlas/go-migration-inventory.md:92:| `cmd/pair-scribe/README.md` | helper docs | users configuring shell logging | Documents `pair-scribe` install/usage. | No runtime behavior. | docs; update if `pair scribe` route added | P3 |
+atlas/go-migration-inventory.md:115:Build/install callers:
+atlas/go-migration-inventory.md:118:- `make install` copies those binaries to `~/.local/bin`.
+atlas/go-migration-inventory.md:119:- `pair-dev` relies on `make build`, then zellij's PATH lookup resolves fresh
+atlas/go-migration-inventory.md:128:- #76 established the first helper-dispatch pattern with `pair-go context` and
+atlas/go-migration-inventory.md:129:  `pair-go scrollback-render`, backed by shared internal runners while legacy
+atlas/go-migration-inventory.md:132:- #77 made `pair-go launch ...` a meaningful Go-owned compatibility handoff to
+atlas/go-migration-inventory.md:137:- #79 owns whether `nvim/` and `zellij/` are embedded or installed adjacent.
+atlas/go-migration-inventory.md:161:- `bin/pair-dev`
+atlas/go-migration-inventory.md:208:- `cmd/pair-go/helper_equivalence_test.go`
+atlas/go-migration-inventory.md:209:- `cmd/pair-go/main.go`
+atlas/go-migration-inventory.md:280:timeout wrapper. They are migration evidence, not installed artifacts.
+atlas/index.md:16:- `README.md` (repo root) — install and usage.
+cmd/internal/dispatcher/dispatcher.go:12:const programName = "pair-go"
+cmd/internal/dispatcher/dispatcher.go:53:			Stdout:   "pair-go dispatcher skeleton\nlaunch handoff: bin/pair\n",
+cmd/internal/dispatcher/dispatcher.go:91:		Stderr:   "pair-go launch is a process handoff implemented by cmd/pair-go; call pair-go launch ... instead of dispatcher.Dispatch\n",
+cmd/pair-wrap/update_agent_output_test.go:87:	// non-SGR escape match, "make nous-install" merges into the
+cmd/pair-wrap/update_agent_output_test.go:88:	// unusable autocomplete candidate "makenous-install".
+cmd/pair-wrap/update_agent_output_test.go:91:	// "make" and "nous-install" are spaces in the agent pane; the
+cmd/pair-wrap/update_agent_output_test.go:93:	p.updateAgentOutput([]byte(sgr("31") + "make" + "\x1b[1C" + "nous-install" + sgr("0")))
+cmd/pair-wrap/update_agent_output_test.go:95:	if len(got) != 1 || got[0] != "make nous-install" {
+cmd/pair-wrap/update_agent_output_test.go:96:		t.Errorf("got %q, want [\"make nous-install\"]", got)
+cmd/internal/dispatcher/dispatcher_test.go:21:				"Usage: pair-go <command> [args]",
+cmd/internal/dispatcher/dispatcher_test.go:61:	for _, want := range []string{"pair-go", "dispatcher skeleton", "launch handoff: bin/pair"} {
+cmd/internal/dispatcher/dispatcher_test.go:76:	for _, want := range []string{"wrap", "planned", "not implemented", "pair-go help"} {
+cmd/internal/dispatcher/dispatcher_test.go:91:	for _, want := range []string{"pair-go launch", "process handoff", "cmd/pair-go"} {
+cmd/internal/dispatcher/dispatcher_test.go:136:	for _, want := range []string{"unknown command", "frobnicate", "pair-go help"} {
+cmd/pair-scribe/README.md:26:    make install
+cmd/internal/entrypoint/launch_test.go:9:	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"claude", "--", "--resume"})
+cmd/internal/entrypoint/launch_test.go:20:	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"resume", "demo"})
+cmd/pair-go/main_test.go:18:	if !strings.Contains(stdout.String(), "Usage: pair-go <command> [args]") {
+cmd/pair-go/main_test.go:42:		executable: "/repo/bin/pair-go",
+cmd/pair-go/main_test.go:67:		executable: "/repo/bin/pair-go",
+cmd/pair-go/main_test.go:97:		executable: "/repo/bin/pair-go",
+cmd/pair-go/main_test.go:110:	for _, want := range []string{"pair-go launch", "/repo/bin/pair", "make build", "make install", "dev-aliases.sh"} {
+bin/lib/dev-rebuild.sh:3:# `pair-dev` exports PAIR_DEV=1 then execs `pair`; bin/pair sources this and
+bin/lib/dev-rebuild.sh:7:# PATH lookup that neither .zshenv nor construct/dev-aliases.sh's rebuild
+bin/lib/dev-rebuild.sh:11:# Restart-safe: Alt+n / Shift+Alt+N re-exec $0=bin/pair, and PAIR_DEV rides
+bin/lib/dev-rebuild.sh:15:# Deployed launches leave PAIR_DEV unset → no-op → zero toolchain dependency.
+bin/lib/dev-rebuild.sh:17:# Usage:  PAIR_HOME=<repo> dev_rebuild      (no-op unless PAIR_DEV is set)
+bin/lib/dev-rebuild.sh:25:    [ -n "${PAIR_DEV:-}" ] || return 0
+bin/lib/dev-rebuild.sh:27:        echo "pair-dev: 'make' not on PATH — launching with existing binaries." >&2
+bin/lib/dev-rebuild.sh:30:    echo "pair-dev: rebuilding Go binaries (make build in $PAIR_HOME) …" >&2
+bin/lib/dev-rebuild.sh:32:        || echo "pair-dev: build FAILED — launching with last-good binaries (fix, then Alt+n)." >&2
+cmd/internal/entrypoint/launch.go:6:// pair-go launch to the shell-owned pair launcher.
+cmd/pair-go/main.go:1:// pair-go is the development dispatcher for the future primary Go CLI. Its
+cmd/pair-go/main.go:41:		_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
+cmd/pair-go/main.go:46:		_, _ = fmt.Fprintf(stderr, "pair-go launch: pair launcher not found at %s (%v); run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", req.Path, err)
+cmd/pair-go/main.go:75:		_, _ = fmt.Fprintf(os.Stderr, "pair-go launch: exec %s failed: %v\n", path, err)
+bin/pair-session-watch.sh:107:# pre-#000020 behavior so old pair-wrap installs still capture sessions
+cmd/pair-go/helper_equivalence_test.go:15:	pairGo := filepath.Join(bin, "pair-go")
+cmd/pair-go/helper_equivalence_test.go:25:		t.Fatalf("pair-go context mismatch\nlegacy:   code=%d stdout=%q stderr=%q\ndispatch: code=%d stdout=%q stderr=%q",
+bin/pair-dev:2:# pair-dev — dev-mode launcher: identical to `pair`, but rebuilds pair's Go
+bin/pair-dev:10:# pair-dev exports PAIR_DEV=1 and hands off to `pair`, which (and only which)
+bin/pair-dev:11:# owns the gated `make build` — see bin/lib/dev-rebuild.sh. PAIR_DEV survives
+bin/pair-dev:17:#   pair-dev            pair-dev codex      pair-dev -- --resume      etc.
+bin/pair-dev:21:# or other PATH symlink to pair-dev still finds its sibling `pair` in the
+bin/pair-dev:34:export PAIR_DEV=1
+bin/pair-scrollback-open:13:#   $PAIR_HOME      — pair install root (for nvim/scrollback.lua)
+cmd/internal/launcher/args.go:5:// LaunchArgs is the pure parse result for the guarded pair-go launch prototype.
+cmd/internal/launcher/args.go:22:// ParseArgs parses pair-go launch args. It intentionally supports only the
+cmd/internal/launcher/args.go:34:		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %s is not implemented by pair-go launch; use bin/pair", argv[0])}
+cmd/internal/launcher/args.go:37:			return LaunchArgs{}, UsageError{Message: "pair-go launch: 'resume' requires a tag"}
+cmd/internal/launcher/args.go:41:			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: invalid tag: %v", err)}
+cmd/internal/launcher/args.go:44:			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", argv[2])}
+cmd/internal/launcher/args.go:64:		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", arg)}
+cmd/internal/launcher/args_test.go:83:			if !strings.Contains(err.Error(), "not implemented by pair-go launch") {
+cmd/internal/launcher/zellij_test.go:16:  "list-sessions --short") printf 'pair-live\npair-detached\npair-gone\nother\n' ;;
+cmd/internal/launcher/zellij_test.go:17:  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-gone [Created] (EXITED - attach to resurrect)\n' ;;
+cmd/internal/launcher/zellij_test.go:33:		{Name: "pair-gone", State: SessionExited},
+
+exec
+/bin/zsh -lc 'ls -l bin/pair bin/pair-dev bin/pair-go 2>/dev/null || true && file bin/pair bin/pair-dev bin/pair-go 2>/dev/null || true' in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+-rwxr-xr-x@ 1 xianxu  staff   107426 Jun 26 09:55 bin/pair
+-rwxr-xr-x@ 1 xianxu  staff     1720 Jun 16 23:38 bin/pair-dev
+-rwxr-xr-x@ 1 xianxu  staff  4626754 Jun 30 15:27 bin/pair-go
+bin/pair:     Bourne-Again shell script text executable, Unicode text, UTF-8 text
+bin/pair-dev: Bourne-Again shell script text executable, Unicode text, UTF-8 text
+bin/pair-go:  Mach-O 64-bit executable arm64
+
+exec
+/bin/zsh -lc "nl -ba bin/pair | sed -n '1,260p' && nl -ba bin/pair-dev | sed -n '1,120p'" in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+     1	#!/usr/bin/env bash
+     2	# pair — launch a zellij session split between a TUI coding agent (top pane)
+     3	# and an nvim drafting pane (bottom). See README.md.
+     4	#
+     5	# Usage:
+     6	#   pair                       claude in default session
+     7	#   pair <agent>               claude / codex / agy
+     8	#   pair [<agent>] -- <args>   forward args to agent on create,
+     9	#                              e.g. pair claude -- --resume
+    10	#                                   pair -- --dangerously-skip-permissions
+    11	#                                   pair codex -- -p "say hi"
+    12	#   pair -h, --help            show help
+    13	#
+    14	# When detached pair-* sessions exist, an fzf picker is shown so the user
+    15	# explicitly chooses what to do — no silent auto-attach. The picker shows
+    16	# ALL detached pair-* sessions regardless of agent argument; the agent
+    17	# argument only matters for the create path (sentinel label, default name,
+    18	# binary to exec).
+    19	
+    20	set -euo pipefail
+    21	
+    22	# Wrap zellij with a hard timeout for read-only IPC. The daemon socket can
+    23	# wedge if an earlier client got stuck (seen 2026-05-06: a `list-sessions`
+    24	# from hours prior held the socket and every new pair launch piled up behind
+    25	# it). pair makes several of these per launch, so one wedge cascades into
+    26	# unkillable bash trees. 5s is generous — healthy responses are <100ms; if
+    27	# it doesn't return by then, something's wrong and falling through is better
+    28	# than hanging. Long-running zellij invocations (attach, new-session) are
+    29	# intentionally NOT wrapped — they're meant to block.
+    30	zj() {
+    31	    local secs=5
+    32	    if command -v gtimeout >/dev/null 2>&1; then
+    33	        gtimeout "$secs" zellij "$@"
+    34	    elif command -v timeout >/dev/null 2>&1; then
+    35	        timeout "$secs" zellij "$@"
+    36	    else
+    37	        # Pure-bash fallback for systems without coreutils' timeout (default
+    38	        # macOS). Run zellij in the background, arm a watchdog, wait.
+    39	        #
+    40	        # The watchdog's stdio is redirected to /dev/null. Otherwise, in a
+    41	        # pipeline like `zj list-sessions | awk ...`, the watchdog subshell
+    42	        # inherits stdout = pipe-to-awk; when we SIGTERM it after zellij
+    43	        # exits, bash dies but the orphan `sleep` child keeps the pipe FD
+    44	        # open, so awk waits for EOF until the full $secs elapses. With N
+    45	        # piped zj calls in pair's startup, that compounds (we saw N×5s).
+    46	        zellij "$@" &
+    47	        local pid=$!
+    48	        ( sleep "$secs"; kill -9 "$pid" 2>/dev/null ) </dev/null >/dev/null 2>&1 &
+    49	        local watchdog=$!
+    50	        local rc=0
+    51	        wait "$pid" 2>/dev/null || rc=$?
+    52	        kill "$watchdog" 2>/dev/null || true
+    53	        wait "$watchdog" 2>/dev/null || true
+    54	        return $rc
+    55	    fi
+    56	}
+    57	
+    58	# normalize_tag <raw> <errprefix> — accept a bare tag or `pair-<tag>`, strip the
+    59	# prefix, and validate the charset (letters/digits/dash/underscore). Echoes the
+    60	# cleaned tag on stdout; on an empty/invalid value prints "<errprefix> '<raw>'
+    61	# (allowed: …)" to stderr and returns 1. Callers: t="$(normalize_tag "$x" "…")" || exit 1.
+    62	# Shared by resume/continue/rename + the name prompt (#52 ARCH-DRY). It *returns*
+    63	# (never exits) so it composes inside $( ), where an exit would only kill the subshell.
+    64	normalize_tag() {
+    65	    local _raw="$1" _t="$1"
+    66	    case "$_t" in pair-*) _t="${_t#pair-}" ;; esac
+    67	    case "$_t" in
+    68	        ""|*[!A-Za-z0-9_-]*)
+    69	            printf "%s '%s' (allowed: letters, digits, dash, underscore)\n" "$2" "$_raw" >&2
+    70	            return 1 ;;
+    71	    esac
+    72	    printf '%s' "$_t"
+    73	}
+    74	
+    75	# --- help -------------------------------------------------------------------
+    76	case "${1:-}" in
+    77	    -h|--help|help)
+    78	        cat <<'EOF'
+    79	pair — Neovim-backed input field for any TUI coding agent.
+    80	
+    81	USAGE
+    82	  pair                          claude in default session
+    83	  pair <agent>                  e.g. pair codex / pair agy
+    84	  pair resume <tag>             attach pair-<tag> if it exists, else create
+    85	                                with that tag and the saved config; agent
+    86	                                is inferred from saved state
+    87	  pair continue [slug]          resume from a continuation doc; bare lists them
+    88	  pair [<agent>] -- <args>      forward args to the agent on create
+    89	                                e.g. pair claude -- --resume
+    90	                                     pair -- --dangerously-skip-permissions
+    91	                                     pair codex -- -p "say hi"
+    92	  pair list                     list pair-* sessions and their attach state
+    93	  pair rename <old> <new>       rename every tag-scoped file in
+    94	                                $PAIR_DATA_DIR from <old> to <new>.
+    95	                                Refuses if `pair-<old>` is still tracked
+    96	                                by zellij (use Ctrl+Alt+n's (R)ename
+    97	                                option for the in-session equivalent).
+    98	  pair -h, --help               this message
+    99	
+   100	  Use `--` to separate pair's args from the agent's args. Without `--`,
+   101	  pair only accepts `<agent>` as a positional (or `resume <tag>`).
+   102	
+   103	  DEV MODE
+   104	  `pair-dev` is identical to `pair` but rebuilds pair's Go binaries from
+   105	  source (`make build`) on launch AND on every Alt+n / Shift+Alt+N restart,
+   106	  so the zellij-spawned pair-wrap always matches the working tree. Use it
+   107	  when hacking on cmd/*; deployed installs keep running `pair` (prebuilt,
+   108	  no toolchain dependency). See bin/pair-dev + bin/lib/dev-rebuild.sh.
+   109	
+   110	  When creating a new session you'll be prompted for a name; type any
+   111	  custom name (e.g. "bugfix") to make it independent of other sessions.
+   112	  `resume <tag>` skips that prompt — use it to restart a saved tag with
+   113	  its captured args (and the agent's session if still on disk).
+   114	
+   115	  Agent args (after `--`) only apply on CREATE. Reattaching to an
+   116	  existing session uses whatever args the agent was originally started
+   117	  with — the picker doesn't re-launch the agent.
+   118	
+   119	BEHAVIOR
+   120	  Picker shows ALL detached pair-* sessions, regardless of the agent
+   121	  argument. The agent argument only controls what gets created if you
+   122	  pick the "+ new" sentinel: it labels the sentinel, drives the auto-
+   123	  suggested name, and is the binary that gets exec'd.
+   124	
+   125	    0 detached + 0 historical → create directly (prompts for name;
+   126	                                default is auto-generated like
+   127	                                pair-claude-2)
+   128	    ≥1 detached or ≥1 historical → fzf picker over: detached pair-*
+   129	                                sessions, then historical tags from
+   130	                                this cwd within $PAIR_HISTORY_DAYS
+   131	                                (default 14) annotated `(Nd ago, no
+   132	                                live session)`, then a `+ new <agent>
+   133	                                session` sentinel. Detached row →
+   134	                                attach. Historical row → create-by-name
+   135	                                (reuses saved draft / agent config).
+   136	                                Sentinel → fall through to create.
+   137	
+   138	  Historical surface (#24) uses the cwd basename as prefix: tags named
+   139	  `<cwd-base>` or `<cwd-base>-<subproject>` whose draft/log sidecars
+   140	  were touched within the window. PAIR_DEBUG_HISTORY=1 prints the scan
+   141	  results and exits without launching zellij.
+   142	
+   143	  No silent attach. Every reattach is explicit.
+   144	
+   145	KEYBINDINGS (inside a pair session)
+   146	  Alt+Return    send nvim buffer to agent + Enter, log, clear, insert mode
+   147	  Alt+Shift+Return
+   148	                append nvim buffer to agent composer + newline, no submit;
+   149	                log + clear (like Alt+Return but unsubmitted)
+   150	  Alt+c         open/show/hide the review collaboration pane; if no review
+   151	                target exists, starts :PairReview
+   152	  Ctrl+C        send ESC (0x1b) to agent pane — interrupts claude's
+   153	                in-flight stream without leaving the draft
+   154	  Alt+← / Alt+→ walk through prompt history (-N) ↔ draft (*) ↔ queue (+N)
+   155	  Shift+Alt+← / Shift+Alt+→
+   156	                jump to next region boundary (oldest-history / newest-history /
+   157	                * / front-of-queue / back-of-queue)
+   158	  Alt+1..Alt+9  (insert mode, completion popup visible) quick-pick the Nth
+   159	                visible completion item. Items show their index (e.g.
+   160	                `1 bin/pair-wrap`); past 9, use arrows or <C-n>/<C-p>.
+   161	  Alt+q         push current buffer to queue front (+1)
+   162	                from *: parks the draft. from -N: forks. from +N: move-to-front
+   163	  Alt+BS        (at +N) delete the current queued prompt; items shift down,
+   164	                so tap-tap deletes a run
+   165	  Shift+Alt+BS  erase history + draft + queue ("start anew"); confirms first
+   166	  Alt+↑ / Alt+↓ step nvim pane along minimized ↔ 12 lines ↔ 1/2 ladder
+   167	                (works from either pane). minimized = statusline only;
+   168	                Alt+↑ from there grows the pane back. Confirm-requiring
+   169	                keys (Alt+x/d/n/N) auto-grow out of minimized first so
+   170	                the prompt is visible.
+   171	  Alt+d         detach from session (re-attach later via `pair`)
+   172	  Alt+x         full quit — kill session and remove from resurrect list
+   173	                (Ctrl+q is unbound; Alt+x is the only quit path)
+   174	  Alt+n         reload pair — kill the session and re-launch in place
+   175	                with the same tag, agent, args, AND agent session.
+   176	                The conversation resumes; pair itself is the only
+   177	                thing that restarts (useful after a config or binary
+   178	                rebuild). Ctrl+Alt+n is a bound alias — use it on macOS
+   179	                terminal combos where Option+n triggers the dead-tilde
+   180	                composer instead of forwarding as Meta+n.
+   181	  Shift+Alt+N   restart with a fresh agent conversation — same tag,
+   182	                agent, and args, but the saved per-(tag,agent) config
+   183	                is dropped so the agent starts a brand-new session
+   184	  Alt+h         pop up this help in a floating pane (press `q` to dismiss)
+   185	  Alt+i         attach clipboard image to agent + insert whatever marker
+   186	                the agent renders (e.g. claude `[Image #N]`). Captured
+   187	                from pair-wrap's PTY stream so no per-agent format
+   188	                knowledge is baked in. Flashes
+   189	                "[no image in clipboard]" at cursor for 1s if the OS
+   190	                clipboard has no image data.
+   191	  Alt+/         open agent pane scrollback in a read-only ANSI-colored
+   192	                nvim viewer; line numbers match zellij's frame indicator,
+   193	                so e.g. `:880` jumps to the line you saw scrolled to 880.
+   194	                Inside the viewer:
+   195	                  • Alt+q drops a 🤖[comment] marker — bare in normal
+   196	                    mode (the line is the context) or 🤖<selection>
+   197	                    [comment] in visual mode.
+   198	                  • Alt+b / Alt+B jump to previous / next prompt
+   199	                    boundary, so you can hop between turns instead of
+   200	                    scrolling line-by-line.
+   201	                  • Search is smart-case: `/foo` matches Foo/FOO/foo,
+   202	                    `/Foo` only Foo.
+   203	                  • <Esc> exits. If markers (or an overall comment) are
+   204	                    pending, a Yes/No confirm runs first; on accept,
+   205	                    everything is formatted (`> <quote>\n<comment>`)
+   206	                    and appended to the draft for review + send.
+   207	
+   208	  Status line shape: `Alt: <- history H < pos[*] [hint] > Q queued ->`
+   209	  — H/Q are total counts, pos is `*`/`-N`/`+N`. A trailing `*` on `-N`
+   210	  means a pending fork on that history slot; navigating away prompts:
+   211	  (S)end, (Q)ueue, (D)iscard, [S]tay. The `[hint]` is contextual:
+   212	  `[q=queue]` on */-N, `[⌫=del]` on +N.
+   213	
+   214	DRAFT SYNTAX
+   215	  ===  comment  Whole lines starting with `===` are stripped at send time
+   216	                but kept in draft/queue/log. Use them to annotate why a
+   217	                queued prompt exists. Mid-line `===` is unaffected.
+   218	
+   219	FILES
+   220	  ${XDG_DATA_HOME:-~/.local/share}/pair/draft-<tag>.md     persistent draft (*)
+   221	  ${XDG_DATA_HOME:-~/.local/share}/pair/log-<tag>.md       prompt history (-N)
+   222	  ${XDG_DATA_HOME:-~/.local/share}/pair/queue-<tag>/*.md   queued prompts (+N)
+   223	
+   224	See README.md for image-paste recipes and other details.
+   225	EOF
+   226	        exit 0
+   227	        ;;
+   228	    list|ls)
+   229	        if ! command -v zellij >/dev/null 2>&1; then
+   230	            echo "pair: zellij not found on PATH." >&2
+   231	            exit 1
+   232	        fi
+   233	        # Snapshot session state once. --no-formatting carries the EXITED
+   234	        # annotation; --short gives clean newline-separated names.
+   235	        raw=$( { zj list-sessions --no-formatting 2>/dev/null || true; } )
+   236	        sessions=$( { zj list-sessions --short 2>/dev/null || true; } \
+   237	                    | awk '/^pair-/' \
+   238	                    | sort )
+   239	        if [ -z "$sessions" ]; then
+   240	            echo "no pair sessions"
+   241	            exit 0
+   242	        fi
+   243	        list_data_dir="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
+   244	        mkdir -p "$list_data_dir"
+   245	
+   246	        # Backfill agent-<tag> for any running pair-wrap process whose tag
+   247	        # we don't yet have on disk — covers sessions created before the
+   248	        # agent-tracking feature shipped. Read PAIR_TAG from each process's
+   249	        # environ (macOS: `ps -E`; Linux: /proc/<pid>/environ) and the
+   250	        # agent name from argv (the token after `pair-wrap`).
+   251	        pw_pids=$(pgrep -f 'pair-wrap' 2>/dev/null || true)
+   252	        for pid in $pw_pids; do
+   253	            argv=$(ps -p "$pid" -o command= 2>/dev/null || true)
+   254	            [ -z "$argv" ] && continue
+   255	            agent=$(printf '%s\n' "$argv" | awk '
+   256	                { for (i=1; i<=NF; i++) if ($i ~ /pair-wrap$/) { print $(i+1); exit } }')
+   257	            [ -z "$agent" ] && continue
+   258	            if [ -r "/proc/$pid/environ" ]; then
+   259	                env_blob=$(tr '\0' '\n' < "/proc/$pid/environ" 2>/dev/null || true)
+   260	            else
+     1	#!/usr/bin/env bash
+     2	# pair-dev — dev-mode launcher: identical to `pair`, but rebuilds pair's Go
+     3	# binaries from source on every launch AND every in-session restart, so the
+     4	# zellij-spawned pair-wrap always matches the working tree (#000046).
+     5	#
+     6	# Why a separate entrypoint: the agent pane runs pair-wrap via a PATH lookup
+     7	# (`sh -c '... exec pair-wrap ...'`) that no shell function or .zshenv can
+     8	# intercept — so a stale, or gitignored-absent, repo/bin binary silently falls
+     9	# through to an old ~/.local/bin copy and the wrapper drifts from source.
+    10	# pair-dev exports PAIR_DEV=1 and hands off to `pair`, which (and only which)
+    11	# owns the gated `make build` — see bin/lib/dev-rebuild.sh. PAIR_DEV survives
+    12	# the `exec "$0"` that restart performs, so the rebuild re-fires on Alt+n /
+    13	# Shift+Alt+N too. Deployed users keep running `pair` directly, with no
+    14	# toolchain dependency.
+    15	#
+    16	# Usage: exactly `pair`'s — all args forward through.
+    17	#   pair-dev            pair-dev codex      pair-dev -- --resume      etc.
+    18	set -euo pipefail
+    19	
+    20	# Resolve this script's real directory (following symlinks) so a ~/.local/bin
+    21	# or other PATH symlink to pair-dev still finds its sibling `pair` in the
+    22	# source tree rather than next to the symlink.
+    23	src="${BASH_SOURCE[0]}"
+    24	while [ -L "$src" ]; do
+    25	    dir="$(cd -P "$(dirname "$src")" && pwd)"
+    26	    src="$(readlink "$src")"
+    27	    case "$src" in /*) ;; *) src="$dir/$src" ;; esac
+    28	done
+    29	here="$(cd -P "$(dirname "$src")" && pwd)"
+    30	
+    31	# Exported (not a bare assignment) so it survives the exec chain — pair → the
+    32	# zellij/sh launch → and crucially the `exec "$0"` that handle_restart_marker
+    33	# performs on restart. dev_rebuild keys off exactly this.
+    34	export PAIR_DEV=1
+    35	exec "$here/pair" "$@"
+
+exec
+/bin/zsh -lc "rg -n \"install .*pair|pair-dev|bin/pair|GO_BINS|INSTALL_DIR|\\.local/bin.*pair|homebrew|brew\" -g '"'!workshop/history/**'"' -g '"'!bin/pair'"'" in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+atlas/how-to-bring-up-a-new-harness-cli.md:59:- **File:** [bin/pair-session-watch.sh](file:///Users/xianxu/workspace/pair/bin/pair-session-watch.sh)
+atlas/how-to-bring-up-a-new-harness-cli.md:82:- **File:** [bin/pair](file:///Users/xianxu/workspace/pair/bin/pair)
+atlas/how-to-bring-up-a-new-harness-cli.md:83:- Integrate the agent-specific resume argument in `bin/pair`:
+atlas/how-to-bring-up-a-new-harness-cli.md:98:**Telemetry Signal** (aspect `3`, see §3): `session-id` from `pair-session-watch.sh` — `fired` when `extract_id` resolves an id and the config is written, **`near-miss`** when a file matching the watch pattern is found but no id can be extracted (filename/format drift), `fail` when the 60s watch window elapses with no id at all (the session file never appeared where expected). The resume mapping in `bin/pair` is the *consumer* of this id; it's static config with no separate signal.
+atlas/how-to-bring-up-a-new-harness-cli.md:154:3. [ ] **Implement Session Watching** in `bin/pair-session-watch.sh` (using `lsof` and target file patterns).
+atlas/how-to-bring-up-a-new-harness-cli.md:155:4. [ ] **Configure Launcher Recovery** in `bin/pair` (mapping `--conversation` or `--resume` flags).
+atlas/how-to-bring-up-a-new-harness-cli.md:172:`bin/pair` truncates the file once at session launch; all components then append
+atlas/how-to-bring-up-a-new-harness-cli.md:213:the same trust level as the existing scrollback logs. `bin/pair` removes it on quit.
+atlas/review-workbench.md:62:- `readiness.lua` + `bin/pair-review-readiness` — pure/classified git readiness
+atlas/review-workbench.md:121:- `bin/pair-review-open <file>` — validates + spawns the **full-screen** floating pane
+atlas/go-migration-inventory.md:50:| `bin/pair` / `cmd/internal/launcher` / `cmd/internal/entrypoint` | Bash public launcher plus Go-owned launch handoff | user shell, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` (#77) resolves sibling `bin/pair` from the `pair-go` executable and execs it with `pair`-compatible argv/env. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves the compatibility handoff; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Go handoff entrypoint with `bin/pair` retained as stable public launcher and compat target through #77; real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
+atlas/go-migration-inventory.md:51:| `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling `pair`. | Resolves symlinks; depends on `bin/pair` and `bin/lib/dev-rebuild.sh`. | compat-shim, likely stays as developer wrapper until Go entrypoint has dev mode | P1 |
+atlas/go-migration-inventory.md:52:| `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
+atlas/go-migration-inventory.md:53:| `zellij/layouts/main.kdl` | zellij native asset | `bin/pair` via `zellij --new-session-with-layout` | Defines agent and draft panes; shell expands Pair env at pane start. | Calls `pair-wrap`; calls `nvim -u "$PAIR_HOME/nvim/init.lua"`; writes `pane-<tag>-<agent>.json`; writes draft nvim pid file. | native-asset, packaged adjacent/embedded | P0 |
+atlas/go-migration-inventory.md:54:| `zellij/config.kdl` | zellij native asset | zellij session config from `bin/pair` | Global keybinds, copy command, scrollback buffer, pane frames. | Calls `copy-on-select.sh`, `pair-help`, `pair-scrollback-open`, `pair-changelog-open`; routes quit/restart/compact through nvim functions. | native-asset, packaged adjacent/embedded | P0 |
+atlas/go-migration-inventory.md:55:| `bin/pair-wrap` / `cmd/pair-wrap` | Go binary | zellij agent pane | `pair-wrap [--scrollback-log PATH] agent [args...]`; transparent PTY proxy; long-running; failure in diagnostics is swallowed. | Reads Pair env and agent command; writes `agent-output-<tag>`, `agent-pid-<tag>`, scrollback `.raw`/`.events.jsonl`, image capture files; may invoke `pair-slug`. | go-subcommand `pair wrap`; keep old binary as compat alias for KDL until caller moves | P0 |
+atlas/go-migration-inventory.md:56:| `bin/pair-slug` / `cmd/pair-slug` | Go binary | `pair-wrap` turn-end hook, tests | Env-driven, no stdin; resolves native transcript, proposes slug; exits 0 on most failures. | Requires `PAIR_TAG`, `PAIR_DATA_DIR`; reads config/transcripts/git branch; writes `slug-proposed-<tag>`; optional `PAIR_SLUG_*`, `OPENAI_API_KEY`. | go-subcommand `pair slug`; legacy binary retained during #76 | P1 |
+atlas/go-migration-inventory.md:57:| `bin/pair-context` / `cmd/pair-context` / `cmd/internal/contextcmd` | Go binary plus shared runner | `bin/pair-title.sh`; development-only `pair-go context` | `pair-context <tag> <agent>` and `pair-go context <tag> <agent>` print the same humanized token count or nothing; tolerant exit 0 on failure. | Reads `PAIR_DATA_DIR`, `pane-<tag>-<agent>.json`, config, native transcripts. | implemented helper route in `pair-go context`; legacy binary retained while title poller calls it | P1 |
+atlas/go-migration-inventory.md:58:| `bin/pair-scrollback-render` / `cmd/pair-scrollback-render` / `cmd/internal/scrollbackcmd` | Go binary plus shared runner | `bin/pair-scrollback-open`, `bin/pair-changelog-open`, `nvim/scrollback.lua` refresh; development-only `pair-go scrollback-render` | `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out` and `pair-go scrollback-render ...`; nonzero on render/write failure. | Reads `.raw` and `.events.jsonl`; atomically writes `.ansi` or cleaned text. | implemented helper route in `pair-go scrollback-render`; legacy binary retained for shell/Lua callers | P0 |
+atlas/go-migration-inventory.md:59:| `bin/pair-changelog` / `cmd/pair-changelog` | Go binary | `bin/pair-changelog-open` | `pair-changelog --cleaned F --log F --anchor F [--agent A] [--model M]`; exits nonzero on required read/model/write failure. | Reads cleaned scrollback/log/anchor; calls agent model through internal model runner; atomically writes log and anchor. | go-subcommand `pair changelog`; legacy binary retained for opener | P1 |
+atlas/go-migration-inventory.md:60:| `bin/pair-continuation` / `cmd/pair-continuation` | Go binary | nvim compaction prompt instructions, operator/agent shell | `pair-continuation --slug S --agent A --issues CSV --body-file F [--repo-root R ...]`; writes and commits continuation; nonzero on validation/git failure. | Reads body/stdin, git repo state; writes `workshop/continuation/*.md`; runs git commit/push. | go-subcommand `pair continuation`; legacy binary retained for agent instructions until docs change | P1 |
+atlas/go-migration-inventory.md:61:| `bin/pair-scribe` / `cmd/pair-scribe` | Go binary | user shell rc outside Pair sessions | `pair-scribe -log PATH -- CMD [ARGS...]`; long-running PTY wrapper; SIGUSR1 pauses log, SIGUSR2 resumes. | Writes typescript log; wraps child PTY; independent of `PAIR_*`. | go-subcommand candidate is low value; may remain separate installed helper or become `pair scribe` with alias | P2 |
+atlas/go-migration-inventory.md:66:| `bin/pair-scrollback-open` | POSIX shell orchestrator | zellij Alt+/ Run, nvim Alt+b jump | `pair-scrollback-open [--jump prev|next]`; opens read-only nvim viewer; singleton lock. | Requires `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_HOME`; calls renderer, zellij IPC, nvim; writes `.ansi`, `.viewport`, lock. | shell-glue now; candidate Go orchestration after entrypoint, while `nvim/scrollback.lua` remains native | P1 |
+atlas/go-migration-inventory.md:67:| `nvim/scrollback.lua` | Neovim native asset | `bin/pair-scrollback-open` | Loaded by `nvim -u ... <ansi>`; interactive read-only viewer; refreshes backing render. | Reads Pair env and `.ansi`; may call `pair-scrollback-render`; writes pending marker files. | native-asset, adjacent/embedded | P0 |
+atlas/go-migration-inventory.md:68:| `bin/pair-changelog-open` | POSIX shell orchestrator | zellij Alt+l Run | Opens changelog viewer and starts detached render/distill singleton. | Requires Pair env; calls renderer, `pair-changelog`, setsid/perl, nvim; reads/writes `changelog-*` sidecars. | shell-glue now; candidate Go orchestration after entrypoint | P1 |
+atlas/go-migration-inventory.md:69:| `nvim/changelog.lua` | Neovim native asset | `bin/pair-changelog-open` | Loaded by `nvim -u ... <log>`; read-only watcher/spinner. | Reads `PAIR_CHANGELOG_*` and Pair env. | native-asset, adjacent/embedded | P1 |
+atlas/go-migration-inventory.md:70:| `bin/pair-title.sh` | Bash stateful poller | `bin/pair` ensure_title_poller | `pair-title.sh <tag> <agent>`; long-running 60s poller; test hook `PAIR_TITLE_TEST_CALL`. | Reads/writes title pid, pane json, cmux owner files; calls `pair-context`, zellij, ps, cmux. | stateful shell-glue; explicit #78 candidate | P1 |
+atlas/go-migration-inventory.md:71:| `bin/pair-session-watch.sh` | Bash stateful watcher | `bin/pair` create path | `pair-session-watch.sh <agent> <tag> <cwd> [agent-args...]`; background 60s watcher; no-op for claude. | Reads agent pidfile, lsof/ps, native session dirs; writes config JSON atomically; logs adapt events. | stateful shell-glue; explicit #78 candidate | P1 |
+atlas/go-migration-inventory.md:76:| `bin/pair-notify` | Bash notification helper | agent hooks/manual shell inside Pair | `pair-notify [--osc 9|777] "message"`; writes OSC to outer tty; nonzero on bad args/missing tty. | Requires `PAIR_TAG`; reads `outer-tty-<tag>`. | small shell-glue; possible Go subcommand but low packaging impact | P2 |
+atlas/go-migration-inventory.md:77:| `bin/pair-quit.sh` | Bash keybind helper | nvim `PairConfirmQuit` | Touch quit marker then kill zellij session. | Uses `ZELLIJ_SESSION_NAME`, `PAIR_KILL_CMD`; writes cache marker. | small compat shell; can fold into Go/nvim flow after entrypoint | P2 |
+atlas/go-migration-inventory.md:78:| `bin/pair-restart.sh` | Bash keybind helper | nvim restart confirmations | Writes restart marker then kill zellij session; supports `--new-session`. | Uses `PAIR_TAG`, `PAIR_AGENT`, `ZELLIJ_SESSION_NAME`, cache marker files. | small compat shell; can fold after entrypoint | P2 |
+atlas/go-migration-inventory.md:79:| `bin/pair-help` | Bash helper | zellij Alt+h Run | Displays `pair -h` through `less` with escape-to-quit behavior. | Calls `pair`, `less`. | compat-shim; may become `pair help` behavior | P2 |
+atlas/go-migration-inventory.md:83:| `bin/pair-review-open` | POSIX shell review helper | nvim review flow | Validates target and opens floating `nvim -u nvim/review.lua`. | Requires Pair env; calls zellij/nvim. | shell-glue, review workbench can move later if packaging needs it | P2 |
+atlas/go-migration-inventory.md:84:| `bin/pair-review-readiness` | POSIX shell review helper | `nvim/init.lua` review readiness | Emits readiness data from git and target helper. | Uses `PAIR_HOME`, `PAIR_REVIEW_TARGET_BIN`, git/jq. | shell-glue; possible later Go helper | P2 |
+atlas/go-migration-inventory.md:85:| `bin/pair-review-target` | Bash review helper | review readiness/open/tests | Emits JSON target metadata under data dir. | Requires `PAIR_DATA_DIR`; reads config/pid files/lsof; writes `review-target-<tag>.json`. | shell-glue; possible #78 candidate if review packaging matters | P2 |
+atlas/go-migration-inventory.md:90:| `Makefile.local` | build/install/test metadata | developer/CI/`pair-dev` | Builds and installs Go binaries, runs test matrix. | Writes `bin/` and `~/.local/bin`; invokes Go, nvim, shell tests. | build contract; #74/#76/#79 must update as dispatcher changes | P0 |
+atlas/go-migration-inventory.md:91:| `README.md` | install/usage docs | users/package consumers | Homebrew install, CLI usage, keybindings, dev mode. | Documents dependencies and public commands. | docs to update at #77/#79 | P1 |
+atlas/go-migration-inventory.md:117:- `make build` builds `GO_BINS` into `bin/`.
+atlas/go-migration-inventory.md:119:- `pair-dev` relies on `make build`, then zellij's PATH lookup resolves fresh
+atlas/go-migration-inventory.md:124:- #74 should add a Go dispatcher without changing `bin/pair`.
+atlas/go-migration-inventory.md:133:  `bin/pair`, with argv/env preserved and missing-launcher diagnostics. The
+atlas/go-migration-inventory.md:156:- `bin/pair`
+atlas/go-migration-inventory.md:157:- `bin/pair-changelog`
+atlas/go-migration-inventory.md:158:- `bin/pair-changelog-open`
+atlas/go-migration-inventory.md:159:- `bin/pair-context`
+atlas/go-migration-inventory.md:160:- `bin/pair-continuation`
+atlas/go-migration-inventory.md:161:- `bin/pair-dev`
+atlas/go-migration-inventory.md:162:- `bin/pair-help`
+atlas/go-migration-inventory.md:163:- `bin/pair-notify`
+atlas/go-migration-inventory.md:164:- `bin/pair-quit.sh`
+atlas/go-migration-inventory.md:165:- `bin/pair-restart.sh`
+atlas/go-migration-inventory.md:166:- `bin/pair-review-open`
+atlas/go-migration-inventory.md:167:- `bin/pair-review-readiness`
+atlas/go-migration-inventory.md:168:- `bin/pair-review-target`
+atlas/go-migration-inventory.md:169:- `bin/pair-scribe`
+atlas/go-migration-inventory.md:170:- `bin/pair-scrollback-open`
+atlas/go-migration-inventory.md:171:- `bin/pair-scrollback-render`
+atlas/go-migration-inventory.md:172:- `bin/pair-session-watch.sh`
+atlas/go-migration-inventory.md:173:- `bin/pair-slug`
+atlas/go-migration-inventory.md:174:- `bin/pair-title.sh`
+atlas/go-migration-inventory.md:175:- `bin/pair-wrap`
+README.md:66:| **Alt+1**…**Alt+9** | nvim (insert, popup visible) | Quick-pick the Nth visible completion item. The popup tags its first nine items with indices (e.g. `⌥1 bin/pair-wrap`); past 9, use arrows or `<C-n>` / `<C-p>` |
+README.md:141:Automatically installed with `homebrew`.
+README.md:168:**Homebrew (recommended).** 
+README.md:172:brew tap xianxu/pair && brew install pair
+README.md:175:brew update; brew upgrade pair
+README.md:214:**Hacking on pair?** Use `pair-dev` instead of `pair` — same arguments, but it rebuilds pair's Go binaries from source (`make build`) on launch *and* on every Alt+n / Shift+Alt+N restart, so the zellij-spawned `pair-wrap` always matches your working tree. (Deployed installs run `pair`, which uses the prebuilt binary and needs no Go toolchain.)
+Makefile.local:4:.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
+Makefile.local:11:# to GO_BINS, add the per-binary recipe stanza. `make build` and
+Makefile.local:14:#   make build      → compile every binary in GO_BINS into ./bin/
+Makefile.local:20:# at $PAIR_HOME/bin/pair-scrollback-render to pick up the Go renderer
+Makefile.local:21:# in both dev (source tree) and brew-install layouts.
+Makefile.local:28:INSTALL_DIR := $(HOME)/.local/bin
+Makefile.local:29:GO_BINS     := pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-go
+Makefile.local:31:build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
+Makefile.local:34:	@mkdir -p $(INSTALL_DIR)
+Makefile.local:35:	@for b in $(GO_BINS); do \
+Makefile.local:36:	    install -m 755 $(BIN_DIR)/$$b $(INSTALL_DIR)/$$b && \
+Makefile.local:37:	    echo "installed: $(INSTALL_DIR)/$$b"; \
+Makefile.local:178:# pair-dev rebuild hook (#000046): deployed mode (PAIR_DEV unset) must invoke
+Makefile.local:191:# Drives the real bin/pair via its PAIR_DEBUG_ARGS probe. See tests/pair-continue-test.sh.
+Makefile.local:195:# changelog — end-to-end smoke for bin/pair-changelog-open (#53 M2): fakes the
+Makefile.local:243:# `pair-go launch ...` execs sibling bin/pair for the current migration window;
+cmd/pair-go/main_test.go:42:		executable: "/repo/bin/pair-go",
+cmd/pair-go/main_test.go:55:	if rt.execPath != "/repo/bin/pair" {
+cmd/pair-go/main_test.go:56:		t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
+cmd/pair-go/main_test.go:67:		executable: "/repo/bin/pair-go",
+cmd/pair-go/main_test.go:83:	if rt.execPath != "/repo/bin/pair" {
+cmd/pair-go/main_test.go:84:		t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
+cmd/pair-go/main_test.go:97:		executable: "/repo/bin/pair-go",
+cmd/pair-go/main_test.go:110:	for _, want := range []string{"pair-go launch", "/repo/bin/pair", "make build", "make install", "dev-aliases.sh"} {
+atlas/architecture.md:16:bin/pair                     # entry point (launcher)
+atlas/architecture.md:17:bin/pair-go                  # opt-in Go dispatcher + launch handoff; public launcher remains bin/pair
+atlas/architecture.md:20:bin/pair-quit.sh             # invoked by Alt+x — marks + kills session
+atlas/architecture.md:21:bin/pair-restart.sh          # invoked by Alt+n / Shift+Alt+N — marks (quit + restart) + kills session
+atlas/architecture.md:22:bin/pair-session-watch.sh    # captures codex/agy session id at create time (#000016, #000020)
+atlas/architecture.md:23:bin/pair-wrap                # PTY proxy: OSC translation + scrollback capture
+atlas/architecture.md:24:bin/pair-notify              # hook-driven OSC notifier (e.g. claude Notification)
+atlas/architecture.md:25:bin/pair-scrollback-render   # raw PTY capture → ANSI-colored line dump (#000017)
+atlas/architecture.md:26:bin/pair-scrollback-open     # Alt+/ orchestrator: render + open viewer
+atlas/architecture.md:27:bin/pair-changelog           # TTY → distilled change log (LLM, incremental) (#53)
+atlas/architecture.md:28:bin/pair-changelog-open      # Alt+l orchestrator: open viewer; clean+distill run in the background (#53)
+atlas/architecture.md:48:As of #77, `bin/pair-go` is the Go-owned launch entrypoint under test:
+atlas/architecture.md:49:`pair-go launch ...` resolves its own executable, finds sibling `bin/pair`, and
+atlas/architecture.md:60:behavior remain shell-owned through `bin/pair` until later migration issues.
+atlas/architecture.md:66:the old command names remain the live integration surface. `bin/pair-title.sh`,
+atlas/architecture.md:67:`bin/pair-scrollback-open`, `bin/pair-changelog-open`, and
+atlas/architecture.md:76:merge-safe: after any sub-issue lands, the public `pair` command, `pair-dev`,
+atlas/architecture.md:81:### `bin/pair` — launcher
+atlas/architecture.md:100:**Title poller (`bin/pair-title.sh`) — two surfaces.** A single always-on per-tag 60s background poller, spawned via `ensure_title_poller` on *every* entry (create, attach, restart) so a poller a host sleep/reboot/SIGKILL killed is reliably revived. Single-instance guard is identity-checked (`poller_alive` `ps`-matches the command line for this tag; pidfile `$DATA_DIR/title-pid-<tag>`; not a bare `kill -0`) so a recycled PID can't suppress the respawn. It owns two title surfaces (tested in `tests/pair-title-poller-test.sh`):
+atlas/architecture.md:118:**Reload / restart in place (Alt+n, Shift+Alt+N).** A second marker, `~/.cache/pair/restart-<session>`, is written alongside `quit-` by `bin/pair-restart.sh`, carrying the agent name + a `new_session` flag. After cleanup_quit_marker tears the session down, `handle_restart_marker` reads the marker and `exec`s pair on itself with `PAIR_FORCE_TAG=<same-tag>` set in the env (pins the new run to the killed session's tag, skipping both the picker and the name prompt). The flag controls what happens to the saved config:
+atlas/architecture.md:123:The picker is bypassed in either flavor — Alt+n's argv carries an explicit resume token, and Shift+Alt+N has no saved config to pick against. A third marker field, `continue=<slug>` (#55, written by the in-session compaction branch in `bin/pair`, not `pair-restart.sh`), rides the `new_session=1` path but re-execs `pair continue <slug> <agent> -- <args>` instead of a plain restart — see "In-session compaction" below.
+atlas/architecture.md:144:- `copy_command "copy-on-select.sh"` — on every selection finalize (mouse-up after drag), zellij pipes the selected text to this script. `copy_command` replaces zellij's default OS-clipboard write, so the script does that part too. Resolved by PATH (which `bin/pair` populated).
+atlas/architecture.md:155:- `Shift+Alt+N` — routed through nvim to `:lua PairConfirmRestartNewSession()` — Y/N modal then `pair-restart.sh --new-session` (restart with a fresh agent conversation). See "Reload / restart in place" under `bin/pair`.
+atlas/architecture.md:156:- `Alt+Shift+C` (`Alt C` / `Ctrl Alt c`) — routed through nvim to `:lua PairConfirmCompact()` — Y/N modal then `send_to_agent(<compaction prompt>)` (#55). Unlike the restart binds it does NOT shell out directly: distilling a continuation needs the agent's judgment, so it asks the agent to write a continuation + run `pair continue <slug>` (context-aware → compaction). See "In-session compaction" under `bin/pair`.
+atlas/architecture.md:184:### `bin/pair-quit.sh` — Alt+x handler
+atlas/architecture.md:186:Touches the marker file `~/.cache/pair/quit-$ZELLIJ_SESSION_NAME`, then `exec zellij kill-session $ZELLIJ_SESSION_NAME`. The kill terminates the session including the script itself; on the launcher side, `bin/pair` resumes, sees the marker, and runs `delete-session --force` to clean up the resurrect entry.
+atlas/architecture.md:190:### Outer-TTY capture and notification routing — `bin/pair-wrap`, `bin/pair-notify`
+atlas/architecture.md:196:1. **Outer-TTY capture (in `bin/pair`).** Before invoking zellij, on every attach (both create and reattach branches), pair calls `tty(1)`. The result is the path of pair's controlling TTY — which is precisely the outer PTY (the one allocated by whatever wraps pair: cmux, a terminal emulator, etc.). That path gets written to `$DATA_DIR/outer-tty-<tag>`. Refreshed on every attach because the outer PTY changes across detach/reattach, while pane-shell env stays frozen at zellij session-creation time (env-var approaches would go stale).
+atlas/architecture.md:200:   - **`bin/pair-wrap`** (Go, `cmd/pair-wrap`). Transparent PTY proxy. The zellij agent pane runs `pair-wrap $PAIR_AGENT $PAIR_AGENT_ARGS` instead of the agent directly (wired in `zellij/layouts/main.kdl`). The wrapper allocates a fresh PTY for the agent, forwards stdin/stdout transparently with SIGWINCH propagation, and watches the agent's output stream for OSC notifications. On detection it writes OSC 9 directly to the recorded outer-TTY path — bypassing zellij.
+atlas/architecture.md:243:   - **`bin/pair-notify`** (bash). Hook-driven helper for richer signals. `pair-notify [--osc 9|777] "msg"` reads the same outer-TTY file and writes the OSC. Intended for Claude Code `Notification`/`Stop` hooks where you want semantic events with custom message text rather than relying on the agent's native OSC stream.
+atlas/architecture.md:262:**Replay (`bin/pair-scrollback-render`, Go).** Reads `<raw>` and `<events.jsonl>`, feeds the bytes to a `charmbracelet/x/vt` emulator in a single offset-ordered walk over all events (`feedSegments`): write up to each offset, then `Resize` on a resize event or snapshot `Scrollback().Len()` on a `time` event (#59). The emulator runs the same VT100 interpretation zellij does live (width-based wrap, alternate-screen flips, scroll regions), so its row count matches what the user saw in zellij's indicator. After feeding, the renderer walks the scrolled-out history followed by the visible buffer, and emits one ANSI-decorated line per row to `<out.ansi>`: full-reset SGR + per-row attrs + the row's characters + `\x1b[0m`. With `--with-timestamps` (the change-log path only — never the Alt+/ viewer) the pure `interleaveDateMarkers` then inserts `⟦pair:ts DATE⟧` lines at each day boundary from the time snapshots (#59). Built into `bin/pair-scrollback-render` via `make pair-scrollback-render`; single static binary, no runtime dep. Its raw inputs live in `$PAIR_DATA_DIR` as `scrollback-<tag>-<agent>.{raw,events.jsonl}` (RAW VT bytes, NOT in the repo); `:PairTTYRawPath` / `_G.PairTTYRawPath()` (nvim, #56) prints the current session's live `.raw` path on demand and copies it to the `+` register — useful for grabbing the byte stream mid-session, since an Alt+x quit deletes it unless preserved.
+atlas/architecture.md:268:**In-session compaction (#55, `Alt+Shift+C`).** `pair continue <slug>` is *context-aware*: run from a normal shell it fresh-starts (above); run from INSIDE its own live pane it **compacts** — copy-parks the scrollback as a recovery net, writes a restart marker carrying a new `continue=<slug>` field (same tag, `new_session=1`), and kills the session. The outer `bin/pair` then `handle_restart_marker`-re-execs `pair continue <slug> <agent> -- <args>` (now outside zellij → the fresh-start branch), so the session reincarnates under the same tag with a clean conversation seeded from the continuation. Detection is **ancestry-based** (`in_zellij_pane`) plus a `ZELLIJ_SESSION_NAME == pair-<tag>` confirmation — never `$ZELLIJ*` env alone, since cmux propagates those to sibling non-pair panes (a false positive would park+kill the wrong session). The branch sits *before* the `in_zellij_pane` guard (which otherwise rejects any in-pane `pair`), with `DATA_DIR` + `handle_restart_marker` hoisted above the guard so test seams can drive them. Seams (test-only): `PAIR_FORCE_IN_SESSION`, `PAIR_FAKE_IN_ZELLIJ`, `PAIR_KILL_CMD`, `PAIR_TEST_CALL`, `PAIR_REEXEC_CAPTURE`. The trigger is the `Alt+Shift+C` keybind (`Alt C` / `Ctrl Alt c` → `PairConfirmCompact` → an agent-agnostic prompt that **defers to the `continuation` datatype procedure** — flush-first, then write the continuation and run `pair continue <slug>` — rather than enumerating a section skeleton inline, so the prompt can't drift out of sync with the datatype; that drift was the bug pair#61 fixed); the outer process suppresses the Alt+x park nudge whenever a restart marker is pending (a restart isn't a quit).
+atlas/architecture.md:282:**Open (`bin/pair-scrollback-open`, POSIX sh).** Validates `PAIR_DATA_DIR` / `PAIR_TAG` / `PAIR_AGENT`, runs the renderer, then *launches* `nvim -u $PAIR_HOME/nvim/scrollback.lua $ANSI` as a child — deliberately **not** `exec`, so the script stays alive as nvim's parent and an `EXIT`/`INT`/`TERM` trap can clear the re-entrancy lock on quit. Errors print and `sleep` briefly so the message is readable before the floating pane self-closes. Bound in `zellij/config.kdl` to `Alt+/` as a 100% × 100% floating pane with `close_on_exit=true` — the user's `:q` in the viewer dismisses the pane and returns to pair's two-pane layout untouched. **Re-entrancy guard:** `Alt+/` is a global zellij bind, so pressing it again while the viewer is already focused fires another `Run` and would stack a second nvim (one `:q`/Esc per layer to unwind). zellij can't conditionally skip a `Run`, so the script self-guards: before launching nvim it writes its own PID to `$PAIR_DATA_DIR/scrollback-<tag>-<agent>.openlock`, and on entry it exits immediately if that lock already holds a *live* PID — the redundant floating pane then self-dismisses via `close_on_exit` and focus falls back to the open viewer. A stale lock (hard kill) carries a dead PID and is reclaimed by the next open's `kill -0` check. The draft pane's `Alt+b` (`--jump prev`) runs the same launcher, so it's covered too.
+atlas/architecture.md:445:  `bin/pair-changelog-open` in a floating pane. It opens `nvim -u
+atlas/architecture.md:523:  (the opener `bin/pair-changelog-open` and the draft-nvim `.ready` watcher):
+atlas/architecture.md:524:  the exported `PAIR_SESSION_ID` (set by `bin/pair` at launch for claude-fresh /
+atlas/architecture.md:564:1. **Pre-write at launch (`bin/pair`).** Two paths:
+atlas/architecture.md:567:2. **Watcher (`bin/pair-session-watch.sh`, codex/agy only).** Spawned in the background by `bin/pair` on the create path, right before the zellij launch. Two discovery paths:
+atlas/architecture.md:579:| claude | `~/.claude/projects/<encoded-cwd>/<id>.jsonl` | filename | `--session-id` pre-injected by `bin/pair` (deterministic) |
+atlas/architecture.md:591:**Create-flow prompt (`bin/pair`).** When the create path commits a tag, pair reads `config-<tag>-<agent>.json`. If present, it runs the per-agent stale-id check (claude: `[ -f .../<id>.jsonl ]`; codex: `find ~/.codex/sessions -name "*<id>*"`; agy: check transcript file) and fzf-prompts the user with up to three options:
+atlas/architecture.md:633:**Inside-flow choreography.** `nvim/init.lua`'s `pair_confirm_restart_impl` shells out `pair rename --restart-check` after the user enters a new tag, re-prompting on each rejection. On accept it execs `pair-restart.sh --rename-to <new>`. `pair-restart.sh` writes `rename_to=<new>` into the restart marker (`~/.cache/pair/restart-<SESSION>`) alongside the existing `tag`, `agent`, `new_session` fields. `handle_restart_marker` in `bin/pair` runs after `cleanup_quit_marker` (so the zj delete-session has cleared the live-old gate) and if `rename_to` is set, invokes `"$0" rename <old> <new>` — full check. On success, the working tag for the re-exec is swapped to `<new>` (so `config-<new>-<agent>.json`, the just-renamed file, is what gets resumed). On failure, a 2-second visible stderr warning is printed and the restart continues with the original tag — the user is never stranded.
+atlas/architecture.md:655:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-<tag>` — single-line file recording which agent binary was launched in the session (`claude`, `codex`, ...). Written once at session create; read by `pair list` to display the agent column, and by `bin/pair`'s tag-restart agent-inference. Removed on full quit. The agent isn't otherwise recoverable post-create — env vars are frozen in pane shells, and custom session names (e.g. `pair-bugfix`) don't carry the agent in the name.
+atlas/architecture.md:657:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/config-<tag>-<agent>.json` — saved restart configuration for `(tag, agent)` (issue #000016, #000020). `{ agent, args, session_id }`. For claude, written synchronously by `bin/pair` before zellij launch (`--session-id` is deterministic). For codex/agy, written by `bin/pair-session-watch.sh` once the agent's session file is discovered via lsof. Read by `bin/pair`'s create-flow prompt and by the post-Alt+x hint. Survives Alt+x (unlike `agent-<tag>`, which is cleared) — that's the whole point: it's the bridge between two pair launches against the same tag.
+atlas/architecture.md:659:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-pid-<tag>` — child agent PID written by `cmd/pair-wrap` immediately after `pty.Start`, removed on shutdown. Consumed by `bin/pair-session-watch.sh` to scope `lsof` discovery to a specific process tree (issue #000020). Mtime is also used as the agent-start epoch in the watcher's birth-time fallback.
+atlas/architecture.md:661:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/nvim-pid-<tag>-{draft,scrollback}` — single-line file containing the pid of an `nvim --embed` server child. Written at VimEnter by `nvim/init.lua` (for the draft pane) and `nvim/scrollback.lua` (for the Alt+/ floating viewer) when `$PAIR_NVIM_PID_FILE` is set; the launch sites (`zellij/layouts/main.kdl` for draft, `bin/pair-scrollback-open` for scrollback) export the env var pointing at a tag-scoped path. Read and removed by `cleanup_quit_marker` on Alt+x to SIGKILL the embed deterministically — without this, the embed sometimes survives zellij's pane teardown and accumulates as a PPID=1 orphan, dragging the host into memory pressure across many quits.
+atlas/architecture.md:663:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/pair-wrap-pid-<tag>` — single-line file containing pair-wrap's pid, written at startup by `bin/pair-wrap` if `PAIR_TAG` is set. Read by nvim's Alt+i (`attach_image`) so it can `kill -USR1 <pid>` to arm an image-capture window. Removed by pair-wrap on exit (the `finally` block in `main()`) and by `cleanup_quit_marker` as belt-and-suspenders on Alt+x.
+atlas/architecture.md:669:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/adapt-<tag>.jsonl` — the adaptation flight recorder (issue #000045). One JSON line per harness-adaptation trigger (`{ts, comp, agent, aspect, signal, outcome, detail}`), appended concurrently by `cmd/pair-wrap`, `cmd/pair-slug`, `bin/pair-session-watch.sh`, and `nvim/adapt.lua` — all writing one shared schema. Truncated once at session launch by `bin/pair` (so multi-process `O_APPEND` never races) and removed on full quit. Read by `doctor/doctor.sh` to surface integration drift (near-miss/fail signals). See `atlas/how-to-bring-up-a-new-harness-cli.md` §3 for the signal registry.
+atlas/architecture.md:675:`bin/pair` prepends `$PAIR_HOME/bin` to `$PATH` before exec'ing zellij. zellij and all its child processes (panes, copy_command, Run actions) inherit the PATH and can resolve `clipboard-to-pane.sh`, `copy-on-select.sh`, `pair-quit.sh` by bare name. This lets the zellij KDL configs reference scripts without `sh -c` env-var quoting hacks.
+atlas/architecture.md:677:## Binary freshness: deployed vs dev (`pair-dev`)
+atlas/architecture.md:679:The Go binaries (`pair-wrap`, `pair-slug`, …) live in `$PAIR_HOME/bin` (first on PATH per *Path resolution* above) and, after `make install`, in `~/.local/bin`. `bin/` is **gitignored** — built on demand, absent in a fresh tree. Because the agent pane launches as `sh -c '… exec pair-wrap …'`, the wrapper is resolved by a **PATH lookup**: no shell function or `.zshenv` can intercept it (`exec` bypasses functions, and `sh` ≠ zsh), so `construct/dev-aliases.sh`'s rebuild-on-call freshness does **not** reach it. When `$PAIR_HOME/bin/pair-wrap` is stale or absent, PATH silently falls through to an old `~/.local/bin` copy and the running wrapper drifts from source — the failure mode is *silence*, not an error (diagnosed once via the #000045 flight recorder going quiet for every Go-emitted aspect while only nvim's Lua emitter still logged).
+atlas/architecture.md:683:- **Dev** — `pair-dev` (#000046). Exports `PAIR_DEV=1` and execs `pair`; `bin/pair` then runs `make build` (via `bin/lib/dev-rebuild.sh`'s `dev_rebuild`) on the **create path**, before the layout execs pair-wrap, so `$PAIR_HOME/bin` holds a fresh build. Restart-safe: `PAIR_DEV` survives `handle_restart_marker`'s `exec "$0"`, so Alt+n / Shift+Alt+N rebuild too; a plain attach (no new wrapper spawned) correctly skips it. Deployed launches (`PAIR_DEV` unset) invoke no toolchain.
+atlas/architecture.md:685:`pair-doctor` *diagnoses* the same staleness `pair-dev` prevents: its emitter-health probe (`doctor/emitter-health.sh`, #000047) greps the *running* `pair-wrap`/`pair-slug` (resolved via the `pair-wrap-pid-<tag>` pidfile, else PATH) for its adapt signal strings and flags `[STALE]` when a binary has no logging code — turning the silent-emitter failure into a named finding.
+atlas/architecture.md:689:`cmd/pair-scribe` is a `script(1)` replacement that lives in the pair repo for build-system convenience but is not part of pair's runtime — it's user shell tooling, typically wired at the top of `~/.zshrc` to swap for `script -q -F`. The user's preexec/precmd hooks send `SIGUSR1`/`SIGUSR2` to pause/resume the on-disk typescript around commands whose output (e.g. TUI redraws) shouldn't be captured, enabling a clean "capture last command output" flow that pair can read back from `$_ZSH_SCRIPT_LOG`. Lives at `~/.local/bin/pair-scribe` after `make install`. Full design notes and the zshrc snippet: `cmd/pair-scribe/README.md`.
+bin/pair-quit.sh:2:# Triggered by Alt+x via zellij keybind. Writes a marker file so bin/pair
+bin/pair-quit.sh:8:# bin/pair sees the marker on resume and runs `zellij delete-session --force`
+cmd/pair-scribe/README.md:28:Produces `~/.local/bin/pair-scribe` (and the other Go binaries). Static
+cmd/pair-scribe/README.md:43:    exec ~/.local/bin/pair-scribe -log "$_ZSH_SCRIPT_LOG" -- /bin/zsh
+nvim/changelog.lua:4:-- bin/pair-changelog-open. The distilled counterpart to scrollback.lua, but
+nvim/changelog.lua:81:-- The orchestrator (bin/pair-changelog-open) launches render+distill as a nohup'd
+cmd/pair-wrap/main.go:3:// Installed at bin/pair-wrap and invoked by zellij/layouts/main.kdl on
+cmd/pair-wrap/main.go:983:				// fix in bin/pair-wrap.py (949aeec).
+cmd/pair-wrap/main.go:1788:	// Open the always-on adaptation flight recorder. bin/pair truncates the
+cmd/pair-wrap/main.go:1930:			// bin/pair's cleanup_quit_marker handles them with the rest
+nvim/scrollback.lua:56:-- $PAIR_NVIM_PID_FILE so bin/pair's cleanup_quit_marker can reap it on
+nvim/scrollback.lua:286:    bin = vim.env.PAIR_HOME .. '/bin/pair-scrollback-render'
+workshop/issues/000079-go-packaging-consolidation.md:19:Consolidate release/install packaging around the primary Go `pair` binary and its required assets.
+workshop/issues/000079-go-packaging-consolidation.md:24:- Update Homebrew/install/build docs and `Makefile.local` accordingly.
+workshop/issues/000079-go-packaging-consolidation.md:42:- [ ] Update build/install/Homebrew wiring.
+zellij/layouts/main.kdl:8:// All four are exported by bin/pair before invoking zellij.
+zellij/layouts/main.kdl:32:            // are exported by bin/pair before zellij launches, so the
+zellij/layouts/main.kdl:63:            // embed pid here so cleanup_quit_marker in bin/pair can reap
+cmd/pair-changelog/main.go:2:// log (issue #53). Invoked on demand by bin/pair-changelog-open (Alt+l): it
+workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:185:[Inventoried six artifacts: zellij layout, nvim plugin/config, reverse-direction shell helper, zellij keybind config, launcher, README. Discussed packaging options — self-contained launcher (recommended for v1), modular install with nvim plugin, layered approach. Flagged two design choices: agent-agnostic naming (`pair`/`compose`/`prompt-pane` rather than `claude+`) and where draft files live. Estimated three days of work for shippable v1.]
+workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:248:bin/pair                      # launcher
+workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:281:[Set issue 000001 status to `working`, ran `make issue-sync`. Created directory structure `bin/`, `nvim/`, `zellij/layouts/`. Wrote all six files: `bin/pair`, `bin/clipboard-to-pane.sh`, `nvim/init.lua`, `zellij/config.kdl`, `zellij/layouts/main.kdl`, `README.md`. Plus `atlas/index.md` and `atlas/architecture.md`. Made scripts executable.]
+workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:286:- `bin/pair nonexistent` correctly exits 1 with "agent not found."
+workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:311:💬: ~/workspace/pair (main*) » bin/pair
+workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:486:💬: now, we also need to support multiple bin/pair being issues, potentially on different agnet (claude, codex, gemini). is this supported today
+workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:508:[Updated `bin/pair` to accept `VARIANT="${2:-}"`. PAIR_TAG = agent-variant. Session, draft, log all derive from PAIR_TAG. Layout uses `${PAIR_TAG:-${PAIR_AGENT:-claude}}` for the draft path.]
+workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:512:💬: when we ctrl-q quit, can you in bin/pair, to run zellij kill-session on that session? or, you should extend the bin/pair to ask your to select which existing sessions to connect to, in an TUI type of thing.
+workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:596:💬: ok, when I created a new session with bin/pair, the title of my terminal changed to the slug name used, e.g. pair-claude-5. however, when reattached to pair-claude-5, that didn't happen, the title of the terminal stayed as bin/pair. why?
+workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:636:💬: add help text with bin/pair --help
+workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:644:💬: bin/pair didn't show the session I created with name blogging, I think it's intentional from you, can you explain how this worked
+workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:657:💬: i see, the auto picking behavior is not that useful, because in the bin/pair world, claude sessions are long lived. and auto attach to some long running session that user don't know, is not a good mental model. we can remove auto attach behavior, and always show the picker.
+workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:659:now with this change, it seems basically everything would go to bin/pair pick, right? if that's the case we can just remove pick as a subcommand, as it is the default and only thing that makes sense,
+workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:669:💬: bin/pair only shows the names pair-claude* still. in the new world, you should show all. right?
+workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:708:bin/pair only shows the names pair-claude* still. in the new world, you should show all. right?
+workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:714:[Confirmed file still had old code. Rewrote bin/pair from scratch using Write tool, then verified with grep. This time it stuck:]
+bin/pair-session-watch.sh:9:# Spawned in the background by bin/pair right before zellij launch on the
+bin/pair-session-watch.sh:16:#   claude  — no-op. bin/pair pre-injects `--session-id <uuid>` and writes
+bin/pair-session-watch.sh:50:# Aspect 3 flight recorder (atlas §3). PAIR_TAG is normally exported by bin/pair;
+bin/pair-session-watch.sh:273:        # the canonical store. Same shape as bin/pair's stripping; keep
+zellij/config.kdl:39:// bin/pair prepends $PAIR_HOME/bin to PATH so the script is resolvable by
+workshop/issues/000077-go-entrypoint-switch.md:17:At some point the public `pair` command must become Go-owned. The next safe step is to make the Go-owned `pair-go launch ...` path exercise the real launcher contract while leaving the existing `pair` and `pair-dev` entrypoints stable.
+workshop/issues/000077-go-entrypoint-switch.md:21:Make `pair-go launch ...` a meaningful Go entrypoint by having it hand off to the existing `bin/pair` launcher with `pair`-compatible arguments. `pair-go launch claude` should behave like `pair claude`; `pair-go launch resume <tag>`, `pair-go launch continue ...`, `pair-go launch list`, and `pair-go launch rename ...` should all pass through to the same shell-owned implementation for this migration window.
+workshop/issues/000077-go-entrypoint-switch.md:23:Keep `pair` and `pair-dev` working exactly as they do today. The Go command has no separate `-dev` variant: a developer shell sourced from `../ariadne/construct/dev-aliases.sh` already discovers `cmd/pair-go`, rebuilds `bin/pair-go` on every invocation, and then runs it from the caller's cwd. If `pair-go launch` cannot find the launcher beside the built binary, its diagnostic should point to `make build` / `make install` and the dev-alias path instead of failing with a bare exec error.
+workshop/issues/000077-go-entrypoint-switch.md:31:- [x] `pair-dev` still rebuilds and launches the working tree behavior.
+workshop/issues/000077-go-entrypoint-switch.md:32:- [x] Existing create, attach, resume, continue, rename/list, quit, and restart flows are preserved through the `bin/pair` fallback.
+workshop/issues/000077-go-entrypoint-switch.md:39:- [x] Add tests for `pair-go launch` argv/env handoff to `bin/pair`.
+workshop/issues/000077-go-entrypoint-switch.md:42:- [x] Verify `pair`, `pair-dev`, and `pair-go launch` behavior with process fakes and targeted builds.
+workshop/issues/000077-go-entrypoint-switch.md:68:- 2026-06-30: closed — go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1; go test ./... -count=1; make pair-go; make test-dev-rebuild; bin/pair-go help; bin/pair-go launch --help; bin/pair --help; bin/pair-dev --help
+workshop/issues/000077-go-entrypoint-switch.md:70:Re-scoped after operator guidance: keep `cmd/pair-go` as the Go entrypoint under test, leave `pair` / `pair-dev` stable, and rely on `../ariadne/construct/dev-aliases.sh` to rebuild `cmd/pair-go` in developer shells. `ARCH-DRY`: reuse the existing launcher for real zellij behavior instead of duplicating shell-owned lifecycle paths in Go. `ARCH-PURE`: keep launch path selection testable with a pure path/argv decision plus a thin exec boundary.
+workshop/issues/000077-go-entrypoint-switch.md:72:Plan-quality gate returned FAILURE because the plan promised argv/env handoff but did not explicitly test env propagation, and because `pair-dev --help` under-proved the dev rebuild acceptance criterion. Updated the durable plan to require an inherited-env fake-runner assertion and `make test-dev-rebuild` verification. `ARCH-PURPOSE`: compatibility claims must be pinned by tests, not implied by the shell fallback.
+workshop/issues/000077-go-entrypoint-switch.md:74:Implemented `cmd/internal/entrypoint` and the `cmd/pair-go` launch handoff. `pair-go launch ...` now resolves sibling `bin/pair`, validates it, and execs it with preserved argv/env; missing launcher diagnostics point to `make build`, `make install`, and `../ariadne/construct/dev-aliases.sh`. Removed the stale decision-phase `pair-go launch` dispatcher surface while keeping `context` and `scrollback-render` helper routes intact. Updated README and atlas migration notes.
+workshop/issues/000077-go-entrypoint-switch.md:82:- `bin/pair-go help`
+workshop/issues/000077-go-entrypoint-switch.md:83:- `bin/pair-go launch --help`
+workshop/issues/000077-go-entrypoint-switch.md:84:- `bin/pair --help`
+workshop/issues/000077-go-entrypoint-switch.md:85:- `bin/pair-dev --help`
+bin/pair-review-open:9:# Env (exported by bin/pair): $PAIR_DATA_DIR $PAIR_TAG $PAIR_HOME
+bin/pair-review-open:40:# nvim/review.lua's VimEnter writes its pid here so bin/pair's cleanup can reap
+cmd/pair-changelog/e2e_test.go:59:	// Real render step (as bin/pair-changelog-open invokes it).
+bin/pair-notify:8:# the PTY stream for attention markers. bin/pair records its controlling
+bin/pair-scrollback-open:9:# Inputs come from pair's env (exported by bin/pair before zellij starts):
+bin/pair-scrollback-open:70:if [ ! -x "$PAIR_HOME/bin/pair-scrollback-render" ]; then
+bin/pair-scrollback-open:71:    printf 'pair-scrollback-open: bin/pair-scrollback-render not built.\n' >&2
+bin/pair-scrollback-open:76:if ! "$PAIR_HOME/bin/pair-scrollback-render" "$RAW" "$EVENTS" "$ANSI"; then
+bin/pair-scrollback-open:182:# embed pid here so cleanup_quit_marker in bin/pair can reap it on Alt+x.
+cmd/pair-wrap/update_agent_output_test.go:83:	// The 949aeec bug fix in bin/pair-wrap.py, now ported to the Go
+bin/pair-review-target:2:# bin/pair-review-target — write the review-target seam (#6) stamped with the
+workshop/lessons.md:83:`bin/pair-scrollback-render` initially opened `<out.ansi>` with `'w'` (truncate-then-write). Two `Alt+/` presses in quick succession would race on the same path; whichever finished second left a half-interleaved file for nvim to open. Fixed by writing to `<out.ansi>.tmp` and `os.replace()`-ing at the end.
+workshop/lessons.md:89:Two bugs in v1 of `bin/pair` and `zellij/config.kdl` came from going off memory of zellij's API:
+workshop/lessons.md:164:via `lsof -p <pid> | awk '$4=="txt"`) was `~/.local/bin/pair-wrap` dated days
+workshop/lessons.md:336:   `bin/pair`, but `bin/pair` turned out to be gitignored AND untracked: a normal
+workshop/lessons.md:454:`bin/pair` parser what it resolved (`AGENT`, `FORCED_TAG`, forwarded args,
+workshop/lessons.md:457:pane, `in_zellij_pane` returned true and `bin/pair` exited with "already running
+bin/pair-review-readiness:34:    target_bin="${PAIR_REVIEW_TARGET_BIN:-$HOME_DIR/bin/pair-review-target}"
+bin/pair-changelog-open:17:# Inputs from pair's env (exported by bin/pair before zellij starts):
+bin/pair-changelog-open:29:# resume reuses it. PAIR_SESSION_ID is exported by bin/pair when the id is known
+bin/pair-changelog-open:77:   && [ -x "$PAIR_HOME/bin/pair-scrollback-render" ] && [ -x "$PAIR_HOME/bin/pair-changelog" ]; then
+bin/pair-changelog-open:79:    export PCL_RENDER="$PAIR_HOME/bin/pair-scrollback-render"
+bin/pair-changelog-open:80:    export PCL_DISTILL="$PAIR_HOME/bin/pair-changelog"
+bin/pair-dev:2:# pair-dev — dev-mode launcher: identical to `pair`, but rebuilds pair's Go
+bin/pair-dev:10:# pair-dev exports PAIR_DEV=1 and hands off to `pair`, which (and only which)
+bin/pair-dev:17:#   pair-dev            pair-dev codex      pair-dev -- --resume      etc.
+bin/pair-dev:21:# or other PATH symlink to pair-dev still finds its sibling `pair` in the
+tests/pair-rename.sh:17:PAIR_BIN="$SELF_DIR/../bin/pair"
+tests/pair-rename.sh:33:# the families enumerated in bin/pair's `rename_paths_for` helper.
+bin/pair-restart.sh:3:# current pair session (like Alt+x) and signals bin/pair to re-launch:
+bin/pair-restart.sh:9:#                    conversation: bin/pair drops the saved
+bin/pair-restart.sh:14:# bin/pair runs `zellij delete-session` as usual) AND a `restart-<session>`
+bin/pair-restart.sh:16:# kill-session returns, bin/pair sees the restart marker and execs itself
+bin/pair-restart.sh:22:# cleanup_quit_marker deletes that file before bin/pair gets the chance to
+bin/pair-title.sh:13:# Spawned in the background by bin/pair on both the create and attach paths.
+bin/pair-title.sh:36:# the cmux sidebar are overridden on the next poll — matches bin/pair's
+bin/pair-title.sh:59:# matching "211"; the agent arg always follows the tag. Mirrors bin/pair's
+bin/pair-title.sh:76:# Abbreviate a raw cwd to ~ on a path boundary (mirrors bin/pair:1154).
+bin/pair-title.sh:125:# Test hook (mirrors bin/pair's PAIR_TEST_CALL): invoke a single helper
+bin/pair-title.sh:138:# create path in bin/pair, which spawns this poller right BEFORE calling
+bin/pair-title.sh:152:# Ignore SIGHUP. bin/pair spawns this with `& disown`, which only
+bin/pair-title.sh:245:# Wait for the zellij session to appear (create-path race: bin/pair
+bin/pair-title.sh:312:            # Workspace-title ownership (matches bin/pair's cmux_rename_workspace):
+bin/pair-title.sh:327:            # Personal display convention (matches bin/pair's cmux_rename_workspace):
+cmd/internal/scrollbackcmd/scrollbackcmd.go:3:// Drop-in replacement for the Python+pyte renderer at bin/pair-scrollback-render.
+cmd/internal/scrollbackcmd/scrollbackcmd.go:9:// venv that the brew formula has to manage. A static Go binary using
+cmd/internal/scrollbackcmd/scrollbackcmd.go:20:// CLI is identical to the Python version so bin/pair-scrollback-open can
+tests/cmux-ownership-test.sh:2:# Regression test for bin/pair's cmux_rename_workspace ownership claim.
+tests/cmux-ownership-test.sh:11:# Drives the REAL bin/pair through its PAIR_TEST_CALL seam with process-level
+tests/cmux-ownership-test.sh:20:PAIR="$ROOT/bin/pair"
+nvim/review.lua:2:-- `nvim -u $PAIR_HOME/nvim/review.lua <file>` (from bin/pair-review-open).
+nvim/review.lua:539:  -- pid file (reaped by bin/pair's cleanup) + the open-state file.
+workshop/continuation/20260619T133818-review-m3.md:19:Read the M3 smoke diagnosis in the session/issue first; the exact bin scripts to rewrite are `bin/pair-review-toggle`, `bin/pair-review-open`, the `Alt r` block in `zellij/config.kdl`, and a new `PairReviewToggle()` in `nvim/init.lua` (model on the `Alt d`/`PairConfirmDetach` pattern). This is live-zellij iteration — implement, then the user re-runs the smoke checklist (in the M3 plan, Task 5). Keep it main-thread (a fork can't verify live zellij).
+workshop/continuation/20260619T133818-review-m3.md:51:- Code (branch `000066-agentic-review-workbench`, worktree `/Users/xianxu/workspace/pair`): `nvim/review/{record,reconstruct,apply,projection,markers,mode,docflow,handoff,init}.lua` (the consumer-half), `nvim/review.lua` (pane init), `nvim/pair_poke.lua` (id-based agent poke), `bin/pair-review-{open,toggle}`, `zellij/config.kdl` (the `Alt r` bind). Tests: `nvim/review/*_test.lua` (`make test-lua`) + `tests/review-*-test.sh` (`make test-review`).
+cmd/internal/entrypoint/launch_test.go:9:	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"claude", "--", "--resume"})
+cmd/internal/entrypoint/launch_test.go:10:	if req.Path != "/repo/bin/pair" {
+cmd/internal/entrypoint/launch_test.go:11:		t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
+cmd/internal/entrypoint/launch_test.go:20:	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"resume", "demo"})
+cmd/internal/entrypoint/launch_test.go:21:	if req.Path != "/repo/bin/pair" {
+cmd/internal/entrypoint/launch_test.go:22:		t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
+workshop/continuation/20260616T214136-make-test-hang.md:117:  is `pair-dev`. No cross-repo work this session.
+workshop/plans/000077-go-entrypoint-switch-plan.md:5:**Goal:** Make `pair-go launch ...` a meaningful Go-owned entrypoint that runs the existing `pair` launcher with compatible arguments while leaving `pair` and `pair-dev` stable.
+workshop/plans/000077-go-entrypoint-switch-plan.md:9:**Tech Stack:** Go 1.x, `os.Executable`, `syscall.Exec` or an injected process runner for tests, existing Bash launcher `bin/pair`, existing Makefile build target for `cmd/pair-go`.
+workshop/plans/000077-go-entrypoint-switch-plan.md:24:- **Relationships:** 1:1 with a `pair-go launch ...` invocation; owns the resolved launcher path and argv passed to `bin/pair`.
+workshop/plans/000077-go-entrypoint-switch-plan.md:31:- **DRY rationale:** Reuses the existing `bin/pair` launcher as the only real session lifecycle owner instead of duplicating create/attach/resume/list/rename behavior in Go.
+workshop/plans/000077-go-entrypoint-switch-plan.md:63:	req := entrypoint.ResolveLegacyLaunch("/repo/bin/pair-go", []string{"claude", "--", "--resume"})
+workshop/plans/000077-go-entrypoint-switch-plan.md:64:	if req.Path != "/repo/bin/pair" {
+workshop/plans/000077-go-entrypoint-switch-plan.md:74:	req := entrypoint.ResolveLegacyLaunch("/repo/bin/pair-go", []string{"resume", "demo"})
+workshop/plans/000077-go-entrypoint-switch-plan.md:112:- runner receives sibling `bin/pair`;
+workshop/plans/000077-go-entrypoint-switch-plan.md:116:Also add a missing-launcher test that returns a not-found stat result and asserts stderr mentions `pair-go launch`, `bin/pair`, `make build`, `make install`, and `dev-aliases.sh`.
+workshop/plans/000077-go-entrypoint-switch-plan.md:160:Expected: `bin/pair-go` builds successfully.
+workshop/plans/000077-go-entrypoint-switch-plan.md:180:Update `atlas/architecture.md` and `atlas/go-migration-inventory.md` so they no longer describe `pair-go launch` as decision-phase only. State that #77 makes it a Go-owned compatibility handoff to `bin/pair`, while `bin/pair` remains the public stable entrypoint and the real zellij lifecycle remains shell-owned.
+workshop/plans/000077-go-entrypoint-switch-plan.md:209:Expected: PASS and `bin/pair-go` exists.
+workshop/plans/000077-go-entrypoint-switch-plan.md:217:bin/pair-go help
+workshop/plans/000077-go-entrypoint-switch-plan.md:218:bin/pair-go launch --help
+workshop/plans/000077-go-entrypoint-switch-plan.md:219:bin/pair --help
+workshop/plans/000077-go-entrypoint-switch-plan.md:220:bin/pair-dev --help
+workshop/plans/000077-go-entrypoint-switch-plan.md:223:Expected: `make test-dev-rebuild` passes, proving the existing `PAIR_DEV` rebuild hook still works. Help output succeeds. `bin/pair-go launch --help` should print the existing `pair` help because it hands off to `bin/pair --help`.
+bin/lib/dev-rebuild.sh:3:# `pair-dev` exports PAIR_DEV=1 then execs `pair`; bin/pair sources this and
+bin/lib/dev-rebuild.sh:11:# Restart-safe: Alt+n / Shift+Alt+N re-exec $0=bin/pair, and PAIR_DEV rides
+bin/lib/dev-rebuild.sh:20:# `set -e` (bin/pair:20) — least of all mid-restart, when the old session is
+bin/lib/dev-rebuild.sh:27:        echo "pair-dev: 'make' not on PATH — launching with existing binaries." >&2
+bin/lib/dev-rebuild.sh:30:    echo "pair-dev: rebuilding Go binaries (make build in $PAIR_HOME) …" >&2
+bin/lib/dev-rebuild.sh:32:        || echo "pair-dev: build FAILED — launching with last-good binaries (fix, then Alt+n)." >&2
+workshop/continuation/20260612T002626-compact.md:17:2. Record the e2e PASS in `workshop/issues/000055-compact-keybind.md` `## Log` (M2 step 4 done for `claude`; step 5 — repeat once under `pair-dev codex` — is optional, the prompt is agent-agnostic).
+workshop/continuation/20260612T002626-compact.md:22:  - **M1** (`bin/pair` mechanics: `park_scrollback` copy|move, ancestry-gated in-session branch, `handle_restart_marker` `continue=` re-exec) — **landed** (9b9e0a1), 21/21 `make test-continue`, boundary review **FIX-THEN-SHIP** (the one Important fixed: park-nudge suppressed when a restart marker is pending).
+workshop/continuation/20260612T002626-compact.md:36:- Key files: `bin/pair` — in-session compaction branch (~863–905, placed **before** the `in_zellij_pane` guard), `park_scrollback` (~687), `handle_restart_marker` (~707); `zellij/config.kdl` (`bind "Alt C" "Ctrl Alt c"`); `nvim/init.lua` (`PairConfirmCompact`, ~696–741). Tests: `tests/pair-continue-test.sh` (seams `PAIR_FORCE_IN_SESSION` / `PAIR_FAKE_IN_ZELLIJ` / `PAIR_KILL_CMD`), `make test-continue`.
+workshop/continuation/20260612T002626-compact.md:38:- Branch `000055-compact-keybind`, worktree `/Users/xianxu/workspace/pair`. This is a `PAIR_DEV=1` checkout → relaunch verb is `pair-dev`.
+CHANGELOG.md:4:tagged in git (`vN.M`) and tracked in the homebrew formula at
+CHANGELOG.md:5:[xianxu/homebrew-pair](https://github.com/xianxu/homebrew-pair).
+CHANGELOG.md:72:### Homebrew
+CHANGELOG.md:75:  runtime is Go-only (`pair-scrollback-render`). `brew install pair` no
+CHANGELOG.md:144:- **`pair-dev`** — dev-mode entrypoint that rebuilds the Go binaries from
+CHANGELOG.md:147:  the static Go `pair-scrollback-render`. (The brew formula still vendors
+CHANGELOG.md:210:  (the primary path shipped via brew) and the Python fallback.
+workshop/continuation/20260611T163027-port.md:33:- Key files: `cmd/pair-continuation/` (writer), `cmd/pair-scrollback-render/main.go` (`--plain`/`resolveMax`), `bin/pair` (`continue` verb just after the `resume` block; park-nudge inside `cleanup_quit_marker`).
+workshop/continuation/20260615T225403-cont-improve.md:42:- **Key files:** `../ariadne/construct/datatype/continuation.md` — the rewritten datatype, resolved into pair via the `construct/datatype` → `../../ariadne/construct/datatype` **symlink** (so pair gets it for free, no recompose); `nvim/init.lua` `COMPACT_PROMPT` (~2990) — now defers to that datatype, with a comment guarding against re-inlining a skeleton; `bin/pair-continuation` — the writer (enforces only `## NEXT ACTION`; auto-commits+pushes).
+workshop/continuation/20260615T225403-cont-improve.md:43:- **Repos:** pair = `/Users/xianxu/workspace/pair` (this; `PAIR_DEV=1` → relaunch verb is `pair-dev`). ariadne = `/Users/xianxu/workspace/ariadne` (substrate). Both on `main`.
+workshop/continuation/20260615T225403-cont-improve.md:49:- **No `bin/pair` seed-prompt change** — the resume directive is embedded in the generated file instead. (decision 5)
+cmd/internal/dispatcher/dispatcher.go:53:			Stdout:   "pair-go dispatcher skeleton\nlaunch handoff: bin/pair\n",
+cmd/internal/dispatcher/dispatcher.go:100:	b.WriteString("Development dispatcher skeleton. Public sessions still start through bin/pair.\n\n")
+cmd/internal/dispatcher/dispatcher.go:106:			fmt.Fprintf(&b, "  %-17s %s (compatibility handoff to bin/pair)\n", family.Name, family.Summary)
+cmd/internal/dispatcher/dispatcher_test.go:61:	for _, want := range []string{"pair-go", "dispatcher skeleton", "launch handoff: bin/pair"} {
+workshop/continuation/20260621T180406-pair-pair.md:54:- `nvim/review/readiness.lua`, `bin/pair-review-readiness`, `nvim/review/seam.lua`, `nvim/init.lua` around `:PairReview` and `PairReviewToggle`, and `tests/review-toggle-test.sh` / `tests/review-resume-test.sh` for M4a'.
+cmd/internal/adapt/adapt.go:17:// atomic across processes; bin/pair truncates the file once at session launch
+nvim/init.lua:14:-- the embed pid (lua runs inside the embed); bin/pair's cleanup_quit_marker
+nvim/init.lua:918:    if not bin or bin == '' then bin = (home ~= '') and (home .. '/bin/pair-review-readiness') or 'pair-review-readiness' end
+nvim/init.lua:966:      local bin = (home ~= '') and (home .. '/bin/pair-review-open') or 'pair-review-open'
+nvim/init.lua:1655:-- `/`, `~`, or `./` / `../` etc. Plain `bin/pair-wrap` is *not* a path here
+nvim/init.lua:1727:-- `pair-wrap`, `lessons.md`, `bin/pair-wrap`, `$PAIR_HOME`,
+nvim/init.lua:2982:-- Resolve the change-log session id (#63): the env var bin/pair exports when the
+nvim/init.lua:2985:-- order in bin/pair-changelog-open so the polled .ready path matches the base the
+nvim/init.lua:3322:  '   (or  pair-dev continue <that-slug>  if this is a dev checkout)',
+tests/pair-review-target-test.sh:15:  PAIR_SESSION_ID=envsid "$ROOT/bin/pair-review-target" "$doc" ready >/dev/null
+tests/pair-review-target-test.sh:21:  "$ROOT/bin/pair-review-target" "$doc" ready >/dev/null
+cmd/internal/transcript/transcript.go:18:// config-<tag>-<agent>.json (written by bin/pair / pair-session-watch.sh).
+doctor/emitter-health.sh:9:# (#000046's `pair-dev` PREVENTS staleness at launch; this DIAGNOSES it.)
+doctor/emitter-health.sh:91:               echo "          Fix: make install (or launch via pair-dev). See atlas \"Binary freshness\"." ;;
+cmd/internal/launcher/args.go:34:		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %s is not implemented by pair-go launch; use bin/pair", argv[0])}
+tests/pair-continue-test.sh:3:# name guard (bin/pair, #54). Drives the REAL bin/pair through its
+tests/pair-continue-test.sh:22:PAIR="$ROOT/bin/pair"
+tests/pair-continue-test.sh:115:# #55 in-session compaction (Alt+Shift+C). Drives the REAL bin/pair via the
+tests/pair-continue-test.sh:183:# REAL bin/pair helper seams through PAIR_TEST_CALL with process-level fakes.
+nvim/review/readiness.lua:4:-- gathering lives in bin/pair-review-readiness. The 4 cases are
+tests/changelog-open-test.sh:2:# End-to-end smoke test for bin/pair-changelog-open (#53/#58).
+tests/changelog-open-test.sh:14:if [ ! -x "$PAIR_HOME/bin/pair-changelog" ] || [ ! -x "$PAIR_HOME/bin/pair-scrollback-render" ]; then
+tests/changelog-open-test.sh:50:"$PAIR_HOME/bin/pair-changelog-open"
+tests/review-window-test.sh:48:if PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-open" "$RT/nope.md" 2>/dev/null; then
+tests/review-window-test.sh:54:PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-open" "$RT/doc.md" || true
+doctor/SKILL.md:36:   the fix is `make install` / `pair-dev` (#000046), not a matcher edit. Then the
+doctor/SKILL.md:58:- The log truncates at each session launch (`bin/pair`), so it reflects the
+tests/pair-title-poller-test.sh:2:# Regression + behavior test for bin/pair-title.sh.
+tests/pair-title-poller-test.sh:23:POLLER="$ROOT/bin/pair-title.sh"
+tests/pair-title-poller-test.sh:103:cat > "$RT/bin/pair-context" <<'EOF'
+tests/pair-title-poller-test.sh:108:chmod +x "$RT/bin/zellij" "$RT/bin/pair-context"
+tests/changelog-session-key-test.sh:2:# Focused keying test for bin/pair-changelog-open (#63): the change-log base is
+tests/changelog-session-key-test.sh:25:run() { rm -f "$tmp/nvim-arg"; "$PAIR_HOME/bin/pair-changelog-open"; }
+tests/dev-rebuild-test.sh:10:# critical because bin/pair runs under `set -e` and a failed restart-time build
+tests/dev-rebuild-test.sh:18:RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-devrebuild-test.XXXXXX")"
+tests/review-readiness-cli-test.sh:22:out="$(PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-readiness" "$REPO/doc \"quoted\".md")"
+tests/review-readiness-cli-test.sh:39:  "$ROOT/bin/pair-review-readiness" --prepare "$PREP/doc.md" 2>&1 || true)"
+doctor/README.md:29:`make install`, or launch via `pair-dev` (#000046). A `[STALE]` line explains an
+doctor/README.md:42:| `session-id/fail` or `near-miss` (aspect 3) | session file moved or id format changed | update `watch_dir`/`find_args`/`extract_id` in `bin/pair-session-watch.sh` |
+doctor/README.md:53:- The log truncates at each session launch (`bin/pair`), so it reflects the
+nvim/review/seam.lua:4:-- and bin/pair-review-open — can't diverge on the empty-tag fallback
+tests/pair-session-watch-test.sh:36:"$ROOT/bin/pair-session-watch.sh" codex test "$ROOT" --no-alt-screen &
+tests/emitter-health-test.sh:48:mkdir -p "$RT/pathbin"; cp "$fresh_pw" "$RT/pathbin/pair-wrap"; chmod +x "$RT/pathbin/pair-wrap"
+tests/emitter-health-test.sh:50:[ "$got" = "$RT/pathbin/pair-wrap" ] && pass "no pidfile ⇒ resolves via PATH" || fail "PATH fallback wrong: $got"
+tests/lib/run-headless.sh:24:# `gtimeout` on macOS via `brew install coreutils`. When neither is on PATH
+tests/review-toggle-test.sh:101:  '"' .. vim.env.PAIR_HOME .. '/bin/pair-review-target" "$2" ready >/dev/null',
+
+exec
+/bin/zsh -lc "git diff b829ef00346738e3c3d7e40db38c2c03f51e615b..HEAD -- cmd/internal/launcher cmd/pair-go cmd/internal/dispatcher cmd/internal/entrypoint | sed -n '1,260p'" in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+diff --git a/cmd/internal/dispatcher/dispatcher.go b/cmd/internal/dispatcher/dispatcher.go
+index 3784f1e..fe1fbfd 100644
+--- a/cmd/internal/dispatcher/dispatcher.go
++++ b/cmd/internal/dispatcher/dispatcher.go
+@@ -3,12 +3,9 @@ package dispatcher
+ import (
+ 	"bytes"
+ 	"fmt"
+-	"os"
+ 	"strings"
+-	"time"
+ 
+ 	"github.com/xianxu/pair/cmd/internal/contextcmd"
+-	"github.com/xianxu/pair/cmd/internal/launcher"
+ 	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
+ )
+ 
+@@ -31,7 +28,7 @@ type Result struct {
+ // Families returns the planned command families for the Go dispatcher.
+ func Families() []CommandFamily {
+ 	return []CommandFamily{
+-		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "prototype"},
++		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "handoff"},
+ 		{Name: "context", Summary: "agent pane context meter", Status: "implemented"},
+ 		{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "implemented"},
+ 		{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
+@@ -53,11 +50,11 @@ func Dispatch(args []string) Result {
+ 		return Result{Stdout: Help(programName), ExitCode: 0}
+ 	case "version", "--version":
+ 		return Result{
+-			Stdout:   "pair-go dispatcher skeleton\npublic launcher: bin/pair\n",
++			Stdout:   "pair-go dispatcher skeleton\nlaunch handoff: bin/pair\n",
+ 			ExitCode: 0,
+ 		}
+ 	case "launch":
+-		return DispatchWithLauncherRuntime(args, osLauncherRuntime())
++		return launchHandoffResult()
+ 	case "context":
+ 		return dispatchContext(args[1:])
+ 	case "scrollback-render":
+@@ -89,90 +86,10 @@ func dispatchScrollbackRender(args []string) Result {
+ 	return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}
+ }
+ 
+-type LauncherRuntime struct {
+-	Env      launcher.Env
+-	Sessions launcher.SessionSource
+-	History  launcher.HistoricalScanner
+-}
+-
+-type StaticSessions struct {
+-	Sessions []launcher.Session
+-	Err      error
+-}
+-
+-func (s StaticSessions) Snapshot() ([]launcher.Session, error) {
+-	if s.Err != nil {
+-		return nil, s.Err
+-	}
+-	return s.Sessions, nil
+-}
+-
+-type StaticHistory struct {
+-	Tags []launcher.HistoricalTag
+-	Err  error
+-}
+-
+-func (h StaticHistory) Scan(_ string, _ time.Time) ([]launcher.HistoricalTag, error) {
+-	if h.Err != nil {
+-		return nil, h.Err
+-	}
+-	return h.Tags, nil
+-}
+-
+-func DispatchWithLauncherRuntime(args []string, rt LauncherRuntime) Result {
+-	launchArgs := []string(nil)
+-	if len(args) > 1 {
+-		launchArgs = args[1:]
+-	}
+-	if len(launchArgs) > 0 && (launchArgs[0] == "help" || launchArgs[0] == "--help" || launchArgs[0] == "-h") {
+-		return Result{Stdout: LaunchHelp(programName), ExitCode: 0}
+-	}
+-	outcome, err := launcher.Run(launchArgs, rt.Env, rt.Sessions, rt.History)
+-	if err != nil {
+-		return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
+-	}
+-	decision := outcome.Decision
++func launchHandoffResult() Result {
+ 	return Result{
+-		Stderr: fmt.Sprintf(
+-			"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
+-			decision.Action,
+-			decision.Tag,
+-			decision.SessionName,
+-		),
+-		ExitCode: 3,
+-	}
+-}
+-
+-func LaunchHelp(program string) string {
+-	return fmt.Sprintf(`Usage: %s launch [agent] [-- agent-args...]
+-       %s launch resume <tag>
+-
+-Guarded decision-phase prototype. Public sessions still start through bin/pair.
+-This command parses launch inputs and computes the create/attach/picker decision,
+-then stops before invoking zellij.
+-`, program, program)
+-}
+-
+-func LauncherEnv(home, xdgDataHome, cwd string) launcher.Env {
+-	return launcher.Env{
+-		Home:     home,
+-		XDGData:  xdgDataHome,
+-		Cwd:      cwd,
+-		Now:      time.Now(),
+-		HistoryD: 14,
+-	}
+-}
+-
+-func osLauncherRuntime() LauncherRuntime {
+-	home := os.Getenv("HOME")
+-	xdg := os.Getenv("XDG_DATA_HOME")
+-	cwd, _ := os.Getwd()
+-	env := LauncherEnv(home, xdg, cwd)
+-	dataDir := launcher.ResolveDataDir(home, xdg)
+-	return LauncherRuntime{
+-		Env:      env,
+-		Sessions: launcher.ZellijSource{},
+-		History:  launcher.HistorySource{DataDir: dataDir},
++		Stderr:   "pair-go launch is a process handoff implemented by cmd/pair-go; call pair-go launch ... instead of dispatcher.Dispatch\n",
++		ExitCode: 2,
+ 	}
+ }
+ 
+@@ -185,6 +102,8 @@ func Help(program string) string {
+ 	for _, family := range Families() {
+ 		if family.Status == "prototype" {
+ 			fmt.Fprintf(&b, "  %-17s %s (prototype; decision-phase only)\n", family.Name, family.Summary)
++		} else if family.Status == "handoff" {
++			fmt.Fprintf(&b, "  %-17s %s (compatibility handoff to bin/pair)\n", family.Name, family.Summary)
+ 		} else if family.Status == "implemented" {
+ 			fmt.Fprintf(&b, "  %-17s %s (implemented helper route)\n", family.Name, family.Summary)
+ 		}
+diff --git a/cmd/internal/dispatcher/dispatcher_test.go b/cmd/internal/dispatcher/dispatcher_test.go
+index f10f32b..7dc3021 100644
+--- a/cmd/internal/dispatcher/dispatcher_test.go
++++ b/cmd/internal/dispatcher/dispatcher_test.go
+@@ -21,7 +21,7 @@ func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
+ 				"Usage: pair-go <command> [args]",
+ 				"Implemented commands:",
+ 				"launch",
+-				"decision-phase only",
++				"compatibility handoff",
+ 				"context",
+ 				"scrollback-render",
+ 				"wrap",
+@@ -35,6 +35,9 @@ func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
+ 			if strings.Contains(res.Stdout, "launch             session lifecycle and public pair launcher flow (planned; not implemented") {
+ 				t.Fatalf("Stdout still labels launch unimplemented:\n%s", res.Stdout)
+ 			}
++			if strings.Contains(res.Stdout, "decision-phase only") {
++				t.Fatalf("Stdout still labels launch decision-phase only:\n%s", res.Stdout)
++			}
+ 			for _, stale := range []string{
+ 				"context           agent pane context meter (planned; not implemented",
+ 				"scrollback-render raw PTY capture to ANSI scrollback (planned; not implemented",
+@@ -55,7 +58,7 @@ func TestDispatchVersionIsDevelopmentSkeletonMetadata(t *testing.T) {
+ 	if res.Stderr != "" {
+ 		t.Fatalf("Stderr = %q, want empty", res.Stderr)
+ 	}
+-	for _, want := range []string{"pair-go", "dispatcher skeleton", "public launcher: bin/pair"} {
++	for _, want := range []string{"pair-go", "dispatcher skeleton", "launch handoff: bin/pair"} {
+ 		if !strings.Contains(res.Stdout, want) {
+ 			t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
+ 		}
+@@ -77,57 +80,15 @@ func TestDispatchPlannedCommandReturnsUnsupported(t *testing.T) {
+ 	}
+ }
+ 
+-func TestDispatchLaunchHelpRoutesToPrototype(t *testing.T) {
++func TestDispatchLaunchReportsProcessHandoff(t *testing.T) {
+ 	res := Dispatch([]string{"launch", "--help"})
+-	if res.ExitCode != 0 {
+-		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
+-	}
+-	if res.Stderr != "" {
+-		t.Fatalf("Stderr = %q, want empty", res.Stderr)
+-	}
+-	for _, want := range []string{"Usage: pair-go launch", "decision-phase prototype"} {
+-		if !strings.Contains(res.Stdout, want) {
+-			t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
+-		}
+-	}
+-}
+-
+-func TestDispatchLaunchReturnsPrototypeDecision(t *testing.T) {
+-	res := DispatchWithLauncherRuntime([]string{"launch", "resume", "demo"}, LauncherRuntime{
+-		Env: LauncherEnv("/home/me", "", "/work/pair"),
+-		Sessions: StaticSessions{
+-			Sessions: nil,
+-		},
+-		History: StaticHistory{},
+-	})
+-	if res.ExitCode != 3 {
+-		t.Fatalf("ExitCode = %d, want 3", res.ExitCode)
+-	}
+-	if res.Stdout != "" {
+-		t.Fatalf("Stdout = %q, want empty", res.Stdout)
+-	}
+-	for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
+-		if !strings.Contains(res.Stderr, want) {
+-			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
+-		}
+-	}
+-}
+-
+-func TestDispatchLaunchWithoutArgsReturnsDefaultPrototypeDecision(t *testing.T) {
+-	res := DispatchWithLauncherRuntime([]string{"launch"}, LauncherRuntime{
+-		Env: LauncherEnv("/home/me", "", "/work/pair"),
+-		Sessions: StaticSessions{
+-			Sessions: nil,
+-		},
+-		History: StaticHistory{},
+-	})
+-	if res.ExitCode != 3 {
+-		t.Fatalf("ExitCode = %d, want 3", res.ExitCode)
++	if res.ExitCode != 2 {
++		t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
+ 	}
+ 	if res.Stdout != "" {
+ 		t.Fatalf("Stdout = %q, want empty", res.Stdout)
+ 	}
+-	for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=pair", "session=pair-pair"} {
++	for _, want := range []string{"pair-go launch", "process handoff", "cmd/pair-go"} {
+ 		if !strings.Contains(res.Stderr, want) {
+ 			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
+ 		}
+diff --git a/cmd/internal/entrypoint/launch.go b/cmd/internal/entrypoint/launch.go
+new file mode 100644
+index 0000000..1a40970
+--- /dev/null
++++ b/cmd/internal/entrypoint/launch.go
+@@ -0,0 +1,20 @@
++package entrypoint
++
++import "path/filepath"
++
++// LegacyLaunchRequest describes the current compatibility handoff from
++// pair-go launch to the shell-owned pair launcher.
++type LegacyLaunchRequest struct {
++	Path string
++	Argv []string
++}
++
++func ResolveLegacyLaunch(executable string, launchArgs []string) LegacyLaunchRequest {
++	argv := make([]string, 0, len(launchArgs)+1)
++	argv = append(argv, "pair")
++	argv = append(argv, launchArgs...)
++	return LegacyLaunchRequest{
++		Path: filepath.Join(filepath.Dir(executable), "pair"),
+
+exec
+/bin/zsh -lc 'git ls-files bin/pair bin/pair-dev Makefile.local README.md | xargs -n1 git ls-files --stage --' in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+100644 6bbc2eb341e3116728aa206ab0d74f7b67b2a730 0	Makefile.local
+100644 dcde60a223e7127147f15b2f637d322401945557 0	README.md
+100755 b31ed22f2d2fd602ecf1eb32eb22231e08272c5f 0	bin/pair
+100755 82a73daa4002496c2eda2f78246c4a4bd612132b 0	bin/pair-dev
+
+exec
+/bin/zsh -lc "nl -ba workshop/plans/000077-go-entrypoint-switch-plan.md | sed -n '1,260p'" in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+     1	# Go Entrypoint Switch Implementation Plan
+     2	
+     3	> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.
+     4	
+     5	**Goal:** Make `pair-go launch ...` a meaningful Go-owned entrypoint that runs the existing `pair` launcher with compatible arguments while leaving `pair` and `pair-dev` stable.
+     6	
+     7	**Architecture:** Keep the shell launcher as the behavioral source of truth for this migration window (`ARCH-DRY`, `ARCH-PURPOSE`). Add a small pure decision layer that resolves the sibling launcher path and argv, plus a thin process boundary that execs it (`ARCH-PURE`). The existing dispatcher remains the owner for `pair-go help`, `pair-go context`, and `pair-go scrollback-render`.
+     8	
+     9	**Tech Stack:** Go 1.x, `os.Executable`, `syscall.Exec` or an injected process runner for tests, existing Bash launcher `bin/pair`, existing Makefile build target for `cmd/pair-go`.
+    10	
+    11	---
+    12	
+    13	## Chunk 1: Concepts
+    14	
+    15	### Pure Entities
+    16	
+    17	| Name | Lives in | Status |
+    18	|------|----------|--------|
+    19	| `LegacyLaunchRequest` | `cmd/internal/entrypoint/launch.go` | new |
+    20	| `ResolveLegacyLaunch` | `cmd/internal/entrypoint/launch.go` | new |
+    21	
+    22	**LegacyLaunchRequest** — the process-independent request to invoke the current shell launcher.
+    23	
+    24	- **Relationships:** 1:1 with a `pair-go launch ...` invocation; owns the resolved launcher path and argv passed to `bin/pair`.
+    25	- **DRY rationale:** Centralizes the compatibility mapping from `pair-go launch <pair-args>` to `pair <pair-args>` so tests and the process boundary do not each reconstruct argv handling.
+    26	- **Future extensions:** This is the place to widen from shell handoff to native Go launch when #78/#79 remove enough shell-owned behavior.
+    27	
+    28	**ResolveLegacyLaunch** — pure function that converts the running executable path plus `pair-go launch` args into a `LegacyLaunchRequest`.
+    29	
+    30	- **Relationships:** Used by `cmd/pair-go` before the IO boundary; independent of zellij, fzf, or the filesystem except for caller-provided existence checks.
+    31	- **DRY rationale:** Reuses the existing `bin/pair` launcher as the only real session lifecycle owner instead of duplicating create/attach/resume/list/rename behavior in Go.
+    32	- **Future extensions:** Can accept a `PAIR_GO_LAUNCH_NATIVE=1` mode or replacement implementation later without changing top-level command parsing.
+    33	
+    34	### Integration Points
+    35	
+    36	| Name | Lives in | Status | Wraps |
+    37	|------|----------|--------|-------|
+    38	| `LegacyLauncherRunner` | `cmd/pair-go/main.go` | new | `os.Executable`, `os.Stat`, `syscall.Exec` |
+    39	| `pair-go launch` docs | `README.md`, `atlas/architecture.md`, `atlas/go-migration-inventory.md` | modified | operator-facing migration contract |
+    40	
+    41	**LegacyLauncherRunner** — the thin IO shell that resolves the current binary path, validates sibling `pair`, and replaces the current process with it.
+    42	
+    43	- **Injected into:** `runWithLegacyRunner` in tests, so process behavior is asserted with fakes rather than actually starting zellij.
+    44	- **Future extensions:** Native launch can replace only this runner once the pure launcher core owns side effects.
+    45	
+    46	**pair-go launch docs** — documentation that explains the current migration boundary.
+    47	
+    48	- **Injected into:** README command usage and atlas migration notes.
+    49	- **Future extensions:** #78/#79 can update the same docs when stateful shell glue or packaging changes.
+    50	
+    51	## Chunk 2: Spec And Handoff Tests
+    52	
+    53	### Task 1: Add pure launch request tests
+    54	
+    55	**Files:**
+    56	- Create: `cmd/internal/entrypoint/launch_test.go`
+    57	- Create: `cmd/internal/entrypoint/launch.go`
+    58	
+    59	- [ ] **Step 1: Write the failing tests**
+    60	
+    61	```go
+    62	func TestResolveLegacyLaunchDropsLaunchVerb(t *testing.T) {
+    63		req := entrypoint.ResolveLegacyLaunch("/repo/bin/pair-go", []string{"claude", "--", "--resume"})
+    64		if req.Path != "/repo/bin/pair" {
+    65			t.Fatalf("Path = %q", req.Path)
+    66		}
+    67		want := []string{"pair", "claude", "--", "--resume"}
+    68		if !reflect.DeepEqual(req.Argv, want) {
+    69			t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
+    70		}
+    71	}
+    72	
+    73	func TestResolveLegacyLaunchPreservesSubcommands(t *testing.T) {
+    74		req := entrypoint.ResolveLegacyLaunch("/repo/bin/pair-go", []string{"resume", "demo"})
+    75		want := []string{"pair", "resume", "demo"}
+    76		if !reflect.DeepEqual(req.Argv, want) {
+    77			t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
+    78		}
+    79	}
+    80	```
+    81	
+    82	- [ ] **Step 2: Run the tests to verify RED**
+    83	
+    84	Run: `go test ./cmd/internal/entrypoint -run TestResolveLegacyLaunch -count=1`
+    85	
+    86	Expected: FAIL because `cmd/internal/entrypoint` does not exist yet.
+    87	
+    88	- [ ] **Step 3: Implement minimal pure resolver**
+    89	
+    90	Create `LegacyLaunchRequest` with `Path string` and `Argv []string`. Implement `ResolveLegacyLaunch(executable string, launchArgs []string)` using `filepath.Dir(executable)` and sibling `pair`. The argv must start with `"pair"` and append all launch args unchanged.
+    91	
+    92	- [ ] **Step 4: Run tests to verify GREEN**
+    93	
+    94	Run: `go test ./cmd/internal/entrypoint -run TestResolveLegacyLaunch -count=1`
+    95	
+    96	Expected: PASS.
+    97	
+    98	### Task 2: Add `pair-go launch` process-boundary tests
+    99	
+   100	**Files:**
+   101	- Modify: `cmd/pair-go/main.go`
+   102	- Modify: `cmd/pair-go/main_test.go`
+   103	- Modify: `cmd/pair-go/launch_process_test.go`
+   104	- Modify: `cmd/internal/dispatcher/dispatcher_test.go`
+   105	
+   106	- [ ] **Step 1: Write failing main tests for handoff**
+   107	
+   108	Add a fake runner test that calls the test seam with args `[]string{"launch", "claude", "--", "--resume"}` and asserts:
+   109	
+   110	- exit code is the fake runner's code;
+   111	- no dispatcher prototype text is printed;
+   112	- runner receives sibling `bin/pair`;
+   113	- runner receives argv `["pair", "claude", "--", "--resume"]`.
+   114	- runner receives inherited environment entries, including a sentinel such as `PAIR_TEST_ENV=kept`, so the shell launcher sees the same env that `pair-go` received (`ARCH-PURPOSE` compatibility surface).
+   115	
+   116	Also add a missing-launcher test that returns a not-found stat result and asserts stderr mentions `pair-go launch`, `bin/pair`, `make build`, `make install`, and `dev-aliases.sh`.
+   117	
+   118	- [ ] **Step 2: Run targeted tests to verify RED**
+   119	
+   120	Run: `go test ./cmd/pair-go -run 'TestRunLaunch' -count=1`
+   121	
+   122	Expected: FAIL because current `pair-go launch` returns prototype exit code 3 and never invokes the fake runner.
+   123	
+   124	- [ ] **Step 3: Update main seam**
+   125	
+   126	Refactor `cmd/pair-go/main.go` so `run` delegates to `runWithRuntime`. For `args[0] == "launch"`, use the injected `LegacyLauncherRunner` and `entrypoint.ResolveLegacyLaunch`; otherwise preserve existing dispatcher behavior. The real runner should:
+   127	
+   128	- get `os.Executable()`;
+   129	- resolve sibling `pair`;
+   130	- `os.Stat` it and require a non-directory executable path;
+   131	- call `syscall.Exec(path, argv, os.Environ())`.
+   132	
+   133	In tests, fake these methods and return deterministic codes.
+   134	
+   135	- [ ] **Step 4: Update stale prototype expectations**
+   136	
+   137	Change launch tests that currently expect `"prototype decision"` and exit code `3` to expect legacy handoff behavior or delete them if covered by the new fake-runner tests. Keep dispatcher tests for `context`, `scrollback-render`, help, version, and planned command errors.
+   138	
+   139	- [ ] **Step 5: Run targeted tests to verify GREEN**
+   140	
+   141	Run: `go test ./cmd/pair-go ./cmd/internal/dispatcher ./cmd/internal/entrypoint -count=1`
+   142	
+   143	Expected: PASS.
+   144	
+   145	## Chunk 3: Build Wiring And Documentation
+   146	
+   147	### Task 3: Keep Makefile wiring accurate
+   148	
+   149	**Files:**
+   150	- Modify: `Makefile.local`
+   151	
+   152	- [ ] **Step 1: Write or update dependency expectation**
+   153	
+   154	Inspect the `$(BIN_DIR)/pair-go` dependency list and add `cmd/internal/entrypoint/launch.go` so `make pair-go` rebuilds when the new resolver changes.
+   155	
+   156	- [ ] **Step 2: Run build**
+   157	
+   158	Run: `make pair-go`
+   159	
+   160	Expected: `bin/pair-go` builds successfully.
+   161	
+   162	### Task 4: Document the migration boundary
+   163	
+   164	**Files:**
+   165	- Modify: `README.md`
+   166	- Modify: `atlas/architecture.md`
+   167	- Modify: `atlas/go-migration-inventory.md`
+   168	- Modify: `workshop/issues/000077-go-entrypoint-switch.md`
+   169	
+   170	- [ ] **Step 1: Update README**
+   171	
+   172	Add a short development note near Command Usage:
+   173	
+   174	```markdown
+   175	`pair-go launch ...` is the Go-owned migration entrypoint for testing the launcher path. It accepts the same arguments after `launch` that `pair` accepts directly, then hands off to the current `pair` launcher for one migration window. In a dev shell sourced from `../ariadne/construct/dev-aliases.sh`, `pair-go` rebuilds from `cmd/pair-go` automatically before running; no `pair-go-dev` command is needed.
+   176	```
+   177	
+   178	- [ ] **Step 2: Update atlas**
+   179	
+   180	Update `atlas/architecture.md` and `atlas/go-migration-inventory.md` so they no longer describe `pair-go launch` as decision-phase only. State that #77 makes it a Go-owned compatibility handoff to `bin/pair`, while `bin/pair` remains the public stable entrypoint and the real zellij lifecycle remains shell-owned.
+   181	
+   182	- [ ] **Step 3: Tick issue plan items and log verification intent**
+   183	
+   184	In #77, tick completed plan rows as they land and add a log entry with the exact commands run.
+   185	
+   186	## Chunk 4: Verification And Close
+   187	
+   188	### Task 5: Full verification
+   189	
+   190	**Files:**
+   191	- No new code files.
+   192	
+   193	- [ ] **Step 1: Run focused Go tests**
+   194	
+   195	Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1`
+   196	
+   197	Expected: PASS.
+   198	
+   199	- [ ] **Step 2: Run full Go suite**
+   200	
+   201	Run: `go test ./... -count=1`
+   202	
+   203	Expected: PASS.
+   204	
+   205	- [ ] **Step 3: Run build**
+   206	
+   207	Run: `make pair-go`
+   208	
+   209	Expected: PASS and `bin/pair-go` exists.
+   210	
+   211	- [ ] **Step 4: Smoke test non-zellij routes**
+   212	
+   213	Run:
+   214	
+   215	```bash
+   216	make test-dev-rebuild
+   217	bin/pair-go help
+   218	bin/pair-go launch --help
+   219	bin/pair --help
+   220	bin/pair-dev --help
+   221	```
+   222	
+   223	Expected: `make test-dev-rebuild` passes, proving the existing `PAIR_DEV` rebuild hook still works. Help output succeeds. `bin/pair-go launch --help` should print the existing `pair` help because it hands off to `bin/pair --help`.
+   224	
+   225	- [ ] **Step 5: SDLC close**
+   226	
+   227	Run: `sdlc close --issue 77 --verified '<focused tests, full go test, build, and help smoke evidence>'`.
+   228	
+   229	Expected: SDLC close runs its review gate; fix any Critical/Important findings before merge.
+
+exec
+/bin/zsh -lc 'git status --short && git diff --name-only b829ef00346738e3c3d7e40db38c2c03f51e615b..HEAD -- workshop/plans workshop/issues atlas README.md Makefile.local' in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+ M workshop/issues/000077-go-entrypoint-switch.md
+?? workshop/issues/000089-review-mode-should-disable-edit-while-agent-update-the-doc.md
+Makefile.local
+README.md
+atlas/architecture.md
+atlas/go-migration-inventory.md
+workshop/issues/000077-go-entrypoint-switch.md
+
+exec
+/bin/zsh -lc 'bin/pair-go launch --help >/tmp/pair-go-launch-help.out 2>/tmp/pair-go-launch-help.err; code=$?; printf '"'code=%s\\nstdout_first=%s\\nstderr=%s\\n' \""'$code" "$(head -n 1 /tmp/pair-go-launch-help.out)" "$(cat /tmp/pair-go-launch-help.err)"' in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+code=0
+stdout_first=pair — Neovim-backed input field for any TUI coding agent.
+stderr=
+
+exec
+/bin/zsh -lc 'go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1' in /Users/xianxu/workspace/pair
+ succeeded in 1401ms:
+ok  	github.com/xianxu/pair/cmd/internal/entrypoint	0.205s
+ok  	github.com/xianxu/pair/cmd/pair-go	1.242s
+ok  	github.com/xianxu/pair/cmd/internal/dispatcher	0.488s
+
+codex
+I found one tracker mismatch in the durable plan artifact: it names an integration entity that does not exist under that name in the code. I’m also checking the install layout because `pair-go` resolves a sibling `pair`, while this repo’s `make install` currently copies only Go binaries.
+exec
+/bin/zsh -lc 'rg -n "LegacyLauncherRunner|runWithLegacyRunner|legacyRuntime|runWithLegacyRuntime" cmd workshop/plans/000077-go-entrypoint-switch-plan.md' in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+workshop/plans/000077-go-entrypoint-switch-plan.md:38:| `LegacyLauncherRunner` | `cmd/pair-go/main.go` | new | `os.Executable`, `os.Stat`, `syscall.Exec` |
+workshop/plans/000077-go-entrypoint-switch-plan.md:41:**LegacyLauncherRunner** — the thin IO shell that resolves the current binary path, validates sibling `pair`, and replaces the current process with it.
+workshop/plans/000077-go-entrypoint-switch-plan.md:43:- **Injected into:** `runWithLegacyRunner` in tests, so process behavior is asserted with fakes rather than actually starting zellij.
+workshop/plans/000077-go-entrypoint-switch-plan.md:126:Refactor `cmd/pair-go/main.go` so `run` delegates to `runWithRuntime`. For `args[0] == "launch"`, use the injected `LegacyLauncherRunner` and `entrypoint.ResolveLegacyLaunch`; otherwise preserve existing dispatcher behavior. The real runner should:
+cmd/pair-go/main_test.go:45:	code := runWithLegacyRuntime([]string{"launch", "--help"}, &stdout, &stderr, rt)
+cmd/pair-go/main_test.go:72:	code := runWithLegacyRuntime([]string{"launch", "claude", "--", "--resume"}, &stdout, &stderr, rt)
+cmd/pair-go/main_test.go:102:	code := runWithLegacyRuntime([]string{"launch", "claude"}, &stdout, &stderr, rt)
+cmd/pair-go/main.go:20:	return runWithLegacyRuntime(args, stdout, stderr, osLegacyRuntime{})
+cmd/pair-go/main.go:23:type legacyRuntime interface {
+cmd/pair-go/main.go:30:func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
+cmd/pair-go/main.go:38:func runLegacyLaunch(args []string, stderr io.Writer, rt legacyRuntime) int {
+
+exec
+/bin/zsh -lc "nl -ba Makefile.local | sed -n '24,40p' && nl -ba workshop/plans/000077-go-entrypoint-switch-plan.md | sed -n '34,45p' && nl -ba cmd/pair-go/main.go | sed -n '23,38p'" in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+    24	# zshrc. Each binary the user wires into their shell (e.g. pair-scribe
+    25	# in zshrc) should reference $HOME/.local/bin/<name>.
+    26	
+    27	BIN_DIR     := bin
+    28	INSTALL_DIR := $(HOME)/.local/bin
+    29	GO_BINS     := pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-go
+    30	
+    31	build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
+    32	
+    33	install: build
+    34		@mkdir -p $(INSTALL_DIR)
+    35		@for b in $(GO_BINS); do \
+    36		    install -m 755 $(BIN_DIR)/$$b $(INSTALL_DIR)/$$b && \
+    37		    echo "installed: $(INSTALL_DIR)/$$b"; \
+    38		done
+    39	
+    40	# Per-binary aliases — `make pair-scribe`, etc.
+    34	### Integration Points
+    35	
+    36	| Name | Lives in | Status | Wraps |
+    37	|------|----------|--------|-------|
+    38	| `LegacyLauncherRunner` | `cmd/pair-go/main.go` | new | `os.Executable`, `os.Stat`, `syscall.Exec` |
+    39	| `pair-go launch` docs | `README.md`, `atlas/architecture.md`, `atlas/go-migration-inventory.md` | modified | operator-facing migration contract |
+    40	
+    41	**LegacyLauncherRunner** — the thin IO shell that resolves the current binary path, validates sibling `pair`, and replaces the current process with it.
+    42	
+    43	- **Injected into:** `runWithLegacyRunner` in tests, so process behavior is asserted with fakes rather than actually starting zellij.
+    44	- **Future extensions:** Native launch can replace only this runner once the pure launcher core owns side effects.
+    45	
+    23	type legacyRuntime interface {
+    24		Executable() (string, error)
+    25		Stat(path string) error
+    26		Environ() []string
+    27		Exec(path string, argv []string, env []string) int
+    28	}
+    29	
+    30	func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
+    31		if len(args) > 0 && args[0] == "launch" {
+    32			return runLegacyLaunch(args[1:], stderr, rt)
+    33		}
+    34		res := dispatcher.Dispatch(args)
+    35		return writeResult(res, stdout, stderr)
+    36	}
+    37	
+    38	func runLegacyLaunch(args []string, stderr io.Writer, rt legacyRuntime) int {
+
+exec
+/bin/zsh -lc "make -n install | sed -n '1,80p'" in /Users/xianxu/workspace/pair
+ succeeded in 0ms:
+if command -v go >/dev/null 2>&1; then \
+	    :; \
+	elif command -v brew >/dev/null 2>&1; then \
+	    echo "==> go not found — installing via Homebrew (brew install go)"; \
+	    brew install go; \
+	else \
+	    echo "Error: ariadne ships cmd/sdlc and needs the Go toolchain to build it," >&2; \
+	    echo "  but 'go' is not on PATH and Homebrew isn't available to install it." >&2; \
+	    echo "  Install Go 1.26+ from https://go.dev/dl/ and re-run." >&2; \
+	    exit 1; \
+	fi
+go build -o bin/pair-scrollback-render ./cmd/pair-scrollback-render
+go build -o bin/pair-context ./cmd/pair-context
+if [ -f go.mod ]; then \
+	    found=0; \
+	    skipped=0; \
+	    for d in cmd/*/; do \
+	        name=$(basename "$d"); \
+	        if [ -f "$d/.skip-make-build" ]; then \
+	            echo "  (skipping $name — .skip-make-build sentinel present)"; \
+	            skipped=$((skipped + 1)); \
+	            continue; \
+	        fi; \
+	        if [ -f "$d/main.go" ]; then \
+	            mkdir -p bin; \
+	            echo "==> Building $name..."; \
+	            go build -o "bin/$name" "./$d" || exit 1; \
+	            found=1; \
+	        fi; \
+	    done; \
+	    if [ "$found" = "0" ] && [ "$skipped" = "0" ]; then \
+	        echo "  (no cmd/*/main.go to build)"; \
+	    fi; \
+	fi
+/Library/Developer/CommandLineTools/usr/bin/make --no-print-directory local-build
+:
+mkdir -p /Users/xianxu/.local/bin
+for b in pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-go; do \
+	    install -m 755 bin/$b /Users/xianxu/.local/bin/$b && \
+	    echo "installed: /Users/xianxu/.local/bin/$b"; \
+	done
+
+exec
+/bin/zsh -lc 'go test ./... -count=1' in /Users/xianxu/workspace/pair
+ succeeded in 16085ms:
+ok  	github.com/xianxu/pair/cmd/internal/adapt	0.179s
+ok  	github.com/xianxu/pair/cmd/internal/contextcmd	0.315s
+ok  	github.com/xianxu/pair/cmd/internal/ctxmeter	0.454s
+ok  	github.com/xianxu/pair/cmd/internal/dispatcher	0.733s
+ok  	github.com/xianxu/pair/cmd/internal/entrypoint	0.578s
+ok  	github.com/xianxu/pair/cmd/internal/launcher	2.496s
+ok  	github.com/xianxu/pair/cmd/internal/model	3.474s
+ok  	github.com/xianxu/pair/cmd/internal/scrollbackcmd	0.976s
+ok  	github.com/xianxu/pair/cmd/internal/transcript	1.105s
+ok  	github.com/xianxu/pair/cmd/pair-changelog	15.715s
+ok  	github.com/xianxu/pair/cmd/pair-context	3.799s
+ok  	github.com/xianxu/pair/cmd/pair-continuation	5.800s
+ok  	github.com/xianxu/pair/cmd/pair-go	4.009s
+?   	github.com/xianxu/pair/cmd/pair-scribe	[no test files]
+?   	github.com/xianxu/pair/cmd/pair-scrollback-render	[no test files]
+ok  	github.com/xianxu/pair/cmd/pair-slug	6.466s
+ok  	github.com/xianxu/pair/cmd/pair-wrap	1.913s
+
+codex
+The full Go suite passes locally. One concrete behavioral gap remains around the documented `make install` recovery path: `make install` installs `pair-go` but not its sibling `pair`, so the installed binary cannot satisfy the new sibling resolution unless `pair` came from another installer.
+exec
+/bin/zsh -lc 'RT=$(mktemp -d "${TMPDIR:-/tmp}/pair-install-review.XXXXXX"); HOME="$RT" make install >/tmp/pair-install-review-install.out 2>/tmp/pair-install-review-install.err; code=$?; printf '"'install_code=%s\\n' \""'$code"; if [ "$code" -eq 0 ]; then "$RT/.local/bin/pair-go" launch --help >/tmp/pair-install-review-launch.out 2>/tmp/pair-install-review-launch.err; printf '"'launch_code=%s\\nlaunch_stderr=%s\\ninstalled_pair=%s\\n' \""'$?" "$(cat /tmp/pair-install-review-launch.err)" "$(test -e "$RT/.local/bin/pair" && echo yes || echo no)"; fi' in /Users/xianxu/workspace/pair
+ succeeded in 9138ms:
+install_code=0
+launch_code=1
+launch_stderr=pair-go launch: pair launcher not found at /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-install-review.LYUpPs/.local/bin/pair (stat /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-install-review.LYUpPs/.local/bin/pair: no such file or directory); run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell
+installed_pair=no
+
+codex
+VERDICT: REWORK (confidence: high)
+
+The handoff shape is mostly right: `pair-go launch` now bypasses the pure dispatcher, computes `pair` argv, preserves env, and execs the shell launcher. The blocking issue is install-layout correctness: the new code requires a sibling `pair`, but this repo’s `make install` installs `pair-go` without installing `pair`, and the failure message recommends `make install` even though that does not fix the missing sibling.
+
+1. Strengths:
+- [cmd/internal/entrypoint/launch.go:12](/Users/xianxu/workspace/pair/cmd/internal/entrypoint/launch.go:12) keeps argv mapping pure and tiny. ARCH-PURE pass.
+- [cmd/pair-go/main.go:30](/Users/xianxu/workspace/pair/cmd/pair-go/main.go:30) keeps launch as the only process-boundary special case; helper routes still use the dispatcher.
+- [cmd/pair-go/main_test.go:64](/Users/xianxu/workspace/pair/cmd/pair-go/main_test.go:64) pins argv and inherited env, which is the right compatibility surface.
+
+2. Critical findings:
+- [Makefile.local:29](/Users/xianxu/workspace/pair/Makefile.local:29) / [cmd/internal/entrypoint/launch.go:17](/Users/xianxu/workspace/pair/cmd/internal/entrypoint/launch.go:17): `make install` installs `pair-go` but not `pair`, while `pair-go launch` only looks for sibling `pair`. I verified with a temp `HOME`: `make install` succeeded, then `$HOME/.local/bin/pair-go launch --help` failed with “pair launcher not found …/.local/bin/pair”; `installed_pair=no`. Fix by making the install/package layout install `bin/pair` alongside `pair-go`, or change the resolver/diagnostic to match the actual supported installed layout. ARCH-PURPOSE.
+- [workshop/plans/000077-go-entrypoint-switch-plan.md:38](/Users/xianxu/workspace/pair/workshop/plans/000077-go-entrypoint-switch-plan.md:38): Core concepts table claims new entity `LegacyLauncherRunner` and [line 43](/Users/xianxu/workspace/pair/workshop/plans/000077-go-entrypoint-switch-plan.md:43) claims `runWithLegacyRunner`, but the code has `legacyRuntime` and `runWithLegacyRuntime` at [cmd/pair-go/main.go:23](/Users/xianxu/workspace/pair/cmd/pair-go/main.go:23). Per the boundary checklist, plan/code entity contradiction is Critical. Fix by revising the plan artifact or aligning names.
+
+3. Important findings:
+- Missing test coverage for the installed sibling layout. Add a process-level test or script seam that simulates `make install` output and asserts installed `pair-go launch --help` can find installed `pair`, or explicitly documents/tests that installed `pair-go` is unsupported outside the repo `bin/` layout.
+
+4. Minor findings:
+- None.
+
+5. Test coverage notes:
+- Ran `go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1`: pass.
+- Ran `go test ./... -count=1`: pass.
+- Ran `bin/pair-go launch --help`: pass from repo `bin/`.
+- Temp `HOME` install smoke exposed the blocking installed-layout failure.
+
+6. Architectural notes for upcoming work:
+- ARCH-DRY: pass, real lifecycle remains single-sourced in `bin/pair`.
+- ARCH-PURE: pass for resolver plus injected exec boundary.
+- ARCH-PURPOSE: fail until every documented supported invocation layout for `pair-go launch` can actually derive the sibling launcher it requires.
+
+7. Plan revision recommendations:
+- Append `## Revisions` to `workshop/plans/000077-go-entrypoint-switch-plan.md`: `2026-06-30 — Boundary review found the integration entity was implemented as legacyRuntime/runWithLegacyRuntime, not LegacyLauncherRunner/runWithLegacyRunner; update the Core concepts table and implementation references to match the code, or rename the code to the planned entity names.`
+- Add a revision noting the install-layout discovery: `make install installs pair-go without sibling pair, so the handoff contract needs install/package wiring or a narrower documented support boundary.`
+tokens used
+92,218
+VERDICT: REWORK (confidence: high)
+
+The handoff shape is mostly right: `pair-go launch` now bypasses the pure dispatcher, computes `pair` argv, preserves env, and execs the shell launcher. The blocking issue is install-layout correctness: the new code requires a sibling `pair`, but this repo’s `make install` installs `pair-go` without installing `pair`, and the failure message recommends `make install` even though that does not fix the missing sibling.
+
+1. Strengths:
+- [cmd/internal/entrypoint/launch.go:12](/Users/xianxu/workspace/pair/cmd/internal/entrypoint/launch.go:12) keeps argv mapping pure and tiny. ARCH-PURE pass.
+- [cmd/pair-go/main.go:30](/Users/xianxu/workspace/pair/cmd/pair-go/main.go:30) keeps launch as the only process-boundary special case; helper routes still use the dispatcher.
+- [cmd/pair-go/main_test.go:64](/Users/xianxu/workspace/pair/cmd/pair-go/main_test.go:64) pins argv and inherited env, which is the right compatibility surface.
+
+2. Critical findings:
+- [Makefile.local:29](/Users/xianxu/workspace/pair/Makefile.local:29) / [cmd/internal/entrypoint/launch.go:17](/Users/xianxu/workspace/pair/cmd/internal/entrypoint/launch.go:17): `make install` installs `pair-go` but not `pair`, while `pair-go launch` only looks for sibling `pair`. I verified with a temp `HOME`: `make install` succeeded, then `$HOME/.local/bin/pair-go launch --help` failed with “pair launcher not found …/.local/bin/pair”; `installed_pair=no`. Fix by making the install/package layout install `bin/pair` alongside `pair-go`, or change the resolver/diagnostic to match the actual supported installed layout. ARCH-PURPOSE.
+- [workshop/plans/000077-go-entrypoint-switch-plan.md:38](/Users/xianxu/workspace/pair/workshop/plans/000077-go-entrypoint-switch-plan.md:38): Core concepts table claims new entity `LegacyLauncherRunner` and [line 43](/Users/xianxu/workspace/pair/workshop/plans/000077-go-entrypoint-switch-plan.md:43) claims `runWithLegacyRunner`, but the code has `legacyRuntime` and `runWithLegacyRuntime` at [cmd/pair-go/main.go:23](/Users/xianxu/workspace/pair/cmd/pair-go/main.go:23). Per the boundary checklist, plan/code entity contradiction is Critical. Fix by revising the plan artifact or aligning names.
+
+3. Important findings:
+- Missing test coverage for the installed sibling layout. Add a process-level test or script seam that simulates `make install` output and asserts installed `pair-go launch --help` can find installed `pair`, or explicitly documents/tests that installed `pair-go` is unsupported outside the repo `bin/` layout.
+
+4. Minor findings:
+- None.
+
+5. Test coverage notes:
+- Ran `go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1`: pass.
+- Ran `go test ./... -count=1`: pass.
+- Ran `bin/pair-go launch --help`: pass from repo `bin/`.
+- Temp `HOME` install smoke exposed the blocking installed-layout failure.
+
+6. Architectural notes for upcoming work:
+- ARCH-DRY: pass, real lifecycle remains single-sourced in `bin/pair`.
+- ARCH-PURE: pass for resolver plus injected exec boundary.
+- ARCH-PURPOSE: fail until every documented supported invocation layout for `pair-go launch` can actually derive the sibling launcher it requires.
+
+7. Plan revision recommendations:
+- Append `## Revisions` to `workshop/plans/000077-go-entrypoint-switch-plan.md`: `2026-06-30 — Boundary review found the integration entity was implemented as legacyRuntime/runWithLegacyRuntime, not LegacyLauncherRunner/runWithLegacyRunner; update the Core concepts table and implementation references to match the code, or rename the code to the planned entity names.`
+- Add a revision noting the install-layout discovery: `make install installs pair-go without sibling pair, so the handoff contract needs install/package wiring or a narrower documented support boundary.`
diff --git a/workshop/plans/000077-go-entrypoint-switch-plan.md b/workshop/plans/000077-go-entrypoint-switch-plan.md
index 591e9bb..b6d65b2 100644
--- a/workshop/plans/000077-go-entrypoint-switch-plan.md
+++ b/workshop/plans/000077-go-entrypoint-switch-plan.md
@@ -35,12 +35,12 @@
 
 | Name | Lives in | Status | Wraps |
 |------|----------|--------|-------|
-| `LegacyLauncherRunner` | `cmd/pair-go/main.go` | new | `os.Executable`, `os.Stat`, `syscall.Exec` |
+| `legacyRuntime` | `cmd/pair-go/main.go` | new | `os.Executable`, `os.Stat`, `syscall.Exec` |
 | `pair-go launch` docs | `README.md`, `atlas/architecture.md`, `atlas/go-migration-inventory.md` | modified | operator-facing migration contract |
 
-**LegacyLauncherRunner** — the thin IO shell that resolves the current binary path, validates sibling `pair`, and replaces the current process with it.
+**legacyRuntime** — the thin IO shell that resolves the current binary path, validates sibling `pair`, and replaces the current process with it.
 
-- **Injected into:** `runWithLegacyRunner` in tests, so process behavior is asserted with fakes rather than actually starting zellij.
+- **Injected into:** `runWithLegacyRuntime` in tests, so process behavior is asserted with fakes rather than actually starting zellij.
 - **Future extensions:** Native launch can replace only this runner once the pure launcher core owns side effects.
 
 **pair-go launch docs** — documentation that explains the current migration boundary.
@@ -123,7 +123,7 @@ Expected: FAIL because current `pair-go launch` returns prototype exit code 3 an
 
 - [ ] **Step 3: Update main seam**
 
-Refactor `cmd/pair-go/main.go` so `run` delegates to `runWithRuntime`. For `args[0] == "launch"`, use the injected `LegacyLauncherRunner` and `entrypoint.ResolveLegacyLaunch`; otherwise preserve existing dispatcher behavior. The real runner should:
+Refactor `cmd/pair-go/main.go` so `run` delegates to `runWithLegacyRuntime`. For `args[0] == "launch"`, use the injected `legacyRuntime` and `entrypoint.ResolveLegacyLaunch`; otherwise preserve existing dispatcher behavior. The real runner should:
 
 - get `os.Executable()`;
 - resolve sibling `pair`;
@@ -148,12 +148,17 @@ Expected: PASS.
 
 **Files:**
 - Modify: `Makefile.local`
+- Create: `tests/pair-go-install-layout-test.sh`
 
 - [ ] **Step 1: Write or update dependency expectation**
 
-Inspect the `$(BIN_DIR)/pair-go` dependency list and add `cmd/internal/entrypoint/launch.go` so `make pair-go` rebuilds when the new resolver changes.
+Inspect the `$(BIN_DIR)/pair-go` dependency list and add `cmd/internal/entrypoint/launch.go` so `make pair-go` rebuilds when the new resolver changes. Install `pair` and `pair-dev` as symlinks beside installed Go binaries so copied `pair-go` can resolve sibling `pair` while the shell launcher still resolves `PAIR_HOME` back to the source tree.
 
-- [ ] **Step 2: Run build**
+- [ ] **Step 2: Add installed-layout regression test**
+
+Create `tests/pair-go-install-layout-test.sh` that runs `HOME=<tmp> make install`, asserts `$HOME/.local/bin/pair` and `pair-dev` are symlinks, and runs `$HOME/.local/bin/pair-go launch --help` to prove the installed handoff reaches the existing `pair` help.
+
+- [ ] **Step 3: Run build**
 
 Run: `make pair-go`
 
@@ -214,16 +219,23 @@ Run:
 
 ```bash
 make test-dev-rebuild
+make test-pair-go-install-layout
 bin/pair-go help
 bin/pair-go launch --help
 bin/pair --help
 bin/pair-dev --help
 ```
 
-Expected: `make test-dev-rebuild` passes, proving the existing `PAIR_DEV` rebuild hook still works. Help output succeeds. `bin/pair-go launch --help` should print the existing `pair` help because it hands off to `bin/pair --help`.
+Expected: `make test-dev-rebuild` passes, proving the existing `PAIR_DEV` rebuild hook still works. `make test-pair-go-install-layout` passes, proving `make install` creates the sibling launcher layout that `pair-go launch` requires. Help output succeeds. `bin/pair-go launch --help` should print the existing `pair` help because it hands off to `bin/pair --help`.
 
 - [ ] **Step 5: SDLC close**
 
 Run: `sdlc close --issue 77 --verified '<focused tests, full go test, build, and help smoke evidence>'`.
 
 Expected: SDLC close runs its review gate; fix any Critical/Important findings before merge.
+
+## Revisions
+
+### 2026-06-30
+
+Boundary review found two plan/code mismatches. First, the implemented integration seam is `legacyRuntime` / `runWithLegacyRuntime`, not the original `LegacyLauncherRunner` / `runWithLegacyRunner` names; the Core concepts table and implementation steps now match the code. Second, `make install` originally installed `pair-go` without sibling `pair`, making the documented recovery path false; the plan now includes shell-launcher symlink install wiring plus an installed-layout regression test.


codex
Using `superpowers-using-superpowers` for the mandatory skill check, and treating this strictly as a fresh-context boundary review. I’ll verify the updated diff against the issue, durable plan, and actual repo behavior before giving the verdict.
exec
/bin/zsh -lc "sed -n '1,220p' /Users/xianxu/workspace/ariadne/construct/adapted/superpowers-using-superpowers/SKILL.md && sdlc --help && sed -n '1,220p' workshop/issues/000077-go-entrypoint-switch.md && sed -n '1,280p' workshop/plans/000077-go-entrypoint-switch-plan.md" in /Users/xianxu/workspace/pair
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
---
id: 000077
status: done
deps: [000074, 000075, 000076]
github_issue:
created: 2026-06-26
updated: 2026-06-30
estimate_hours: 2.6
started: 2026-06-30T12:42:11-07:00
actual_hours: 0.75
---

# pair Go entrypoint switch

## Problem

At some point the public `pair` command must become Go-owned. The next safe step is to make the Go-owned `pair-go launch ...` path exercise the real launcher contract while leaving the existing `pair` and `pair-dev` entrypoints stable.

## Spec

Make `pair-go launch ...` a meaningful Go entrypoint by having it hand off to the existing `bin/pair` launcher with `pair`-compatible arguments. `pair-go launch claude` should behave like `pair claude`; `pair-go launch resume <tag>`, `pair-go launch continue ...`, `pair-go launch list`, and `pair-go launch rename ...` should all pass through to the same shell-owned implementation for this migration window.

Keep `pair` and `pair-dev` working exactly as they do today. The Go command has no separate `-dev` variant: a developer shell sourced from `../ariadne/construct/dev-aliases.sh` already discovers `cmd/pair-go`, rebuilds `bin/pair-go` on every invocation, and then runs it from the caller's cwd. If `pair-go launch` cannot find the launcher beside the built binary, its diagnostic should point to `make build` / `make install` and the dev-alias path instead of failing with a bare exec error.

This deliberately keeps real zellij lifecycle, prompt/fzf UI, restart/quit cleanup, continuation, rename/list, and dev rebuild behavior shell-owned until later migration issues. `ARCH-PURPOSE`: #77's purpose is now the first meaningful Go-owned launch entrypoint without destabilizing the existing public command; full public `pair` replacement remains a later cutover once shell stateful glue is reduced.

## Done when

- [x] `pair-go launch ...` uses Go process code first and then invokes the real launcher with `pair`-compatible argv.
- [x] Existing `pair` remains the stable public entrypoint for one migration window.
- [x] `pair-dev` still rebuilds and launches the working tree behavior.
- [x] Existing create, attach, resume, continue, rename/list, quit, and restart flows are preserved through the `bin/pair` fallback.
- [x] The dev workflow is documented: `cmd/pair-go` is rebuilt by `../ariadne/construct/dev-aliases.sh`; no `pair-go-dev` command is needed.
- [x] Pair remains usable after merge; no keybinding workflow regresses.

## Plan

- [x] Confirm prerequisites from earlier Go migration issues.
- [x] Add tests for `pair-go launch` argv/env handoff to `bin/pair`.
- [x] Add stale/missing launcher diagnostics.
- [x] Implement the thin Go handoff while keeping dispatcher helper routes intact.
- [x] Verify `pair`, `pair-dev`, and `pair-go launch` behavior with process fakes and targeted builds.
- [x] Update README/atlas packaging notes.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.25 impl=0.15
item: greenfield-go-module design=0.35 impl=0.40
item: skill-or-dispatcher design=0.30 impl=0.35
item: atlas-docs design=0.20 impl=0.25
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 2.62
```

## Log

### 2026-06-26

Created from #72 as the public switch milestone. This should not be claimed until the earlier dispatcher/helper/launcher milestones have landed.

### 2026-06-30
- 2026-06-30: closed — go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1; go test ./... -count=1; make pair-go; make test-dev-rebuild; make test-pair-go-install-layout; bin/pair-go help; bin/pair-go launch --help; bin/pair --help; bin/pair-dev --help
- 2026-06-30: closed — go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1; go test ./... -count=1; make pair-go; make test-dev-rebuild; bin/pair-go help; bin/pair-go launch --help; bin/pair --help; bin/pair-dev --help; review verdict: REWORK

Re-scoped after operator guidance: keep `cmd/pair-go` as the Go entrypoint under test, leave `pair` / `pair-dev` stable, and rely on `../ariadne/construct/dev-aliases.sh` to rebuild `cmd/pair-go` in developer shells. `ARCH-DRY`: reuse the existing launcher for real zellij behavior instead of duplicating shell-owned lifecycle paths in Go. `ARCH-PURE`: keep launch path selection testable with a pure path/argv decision plus a thin exec boundary.

Plan-quality gate returned FAILURE because the plan promised argv/env handoff but did not explicitly test env propagation, and because `pair-dev --help` under-proved the dev rebuild acceptance criterion. Updated the durable plan to require an inherited-env fake-runner assertion and `make test-dev-rebuild` verification. `ARCH-PURPOSE`: compatibility claims must be pinned by tests, not implied by the shell fallback.

Implemented `cmd/internal/entrypoint` and the `cmd/pair-go` launch handoff. `pair-go launch ...` now resolves sibling `bin/pair`, validates it, and execs it with preserved argv/env; missing launcher diagnostics point to `make build`, `make install`, and `../ariadne/construct/dev-aliases.sh`. Removed the stale decision-phase `pair-go launch` dispatcher surface while keeping `context` and `scrollback-render` helper routes intact. Updated README and atlas migration notes.

Boundary review returned REWORK. Fixed both blockers: `make install` now links `pair` and `pair-dev` beside installed Go binaries so installed `pair-go launch ...` has the sibling launcher it requires, and `tests/pair-go-install-layout-test.sh` covers that temp-install layout. Revised the durable plan's Core concepts table to match the implemented `legacyRuntime` / `runWithLegacyRuntime` names.

Verification passed:

- `go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1`
- `go test ./... -count=1`
- `make pair-go`
- `make test-dev-rebuild`
- `make test-pair-go-install-layout`
- `bin/pair-go help`
- `bin/pair-go launch --help`
- `bin/pair --help`
- `bin/pair-dev --help`
# Go Entrypoint Switch Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `pair-go launch ...` a meaningful Go-owned entrypoint that runs the existing `pair` launcher with compatible arguments while leaving `pair` and `pair-dev` stable.

**Architecture:** Keep the shell launcher as the behavioral source of truth for this migration window (`ARCH-DRY`, `ARCH-PURPOSE`). Add a small pure decision layer that resolves the sibling launcher path and argv, plus a thin process boundary that execs it (`ARCH-PURE`). The existing dispatcher remains the owner for `pair-go help`, `pair-go context`, and `pair-go scrollback-render`.

**Tech Stack:** Go 1.x, `os.Executable`, `syscall.Exec` or an injected process runner for tests, existing Bash launcher `bin/pair`, existing Makefile build target for `cmd/pair-go`.

---

## Chunk 1: Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `LegacyLaunchRequest` | `cmd/internal/entrypoint/launch.go` | new |
| `ResolveLegacyLaunch` | `cmd/internal/entrypoint/launch.go` | new |

**LegacyLaunchRequest** — the process-independent request to invoke the current shell launcher.

- **Relationships:** 1:1 with a `pair-go launch ...` invocation; owns the resolved launcher path and argv passed to `bin/pair`.
- **DRY rationale:** Centralizes the compatibility mapping from `pair-go launch <pair-args>` to `pair <pair-args>` so tests and the process boundary do not each reconstruct argv handling.
- **Future extensions:** This is the place to widen from shell handoff to native Go launch when #78/#79 remove enough shell-owned behavior.

**ResolveLegacyLaunch** — pure function that converts the running executable path plus `pair-go launch` args into a `LegacyLaunchRequest`.

- **Relationships:** Used by `cmd/pair-go` before the IO boundary; independent of zellij, fzf, or the filesystem except for caller-provided existence checks.
- **DRY rationale:** Reuses the existing `bin/pair` launcher as the only real session lifecycle owner instead of duplicating create/attach/resume/list/rename behavior in Go.
- **Future extensions:** Can accept a `PAIR_GO_LAUNCH_NATIVE=1` mode or replacement implementation later without changing top-level command parsing.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `legacyRuntime` | `cmd/pair-go/main.go` | new | `os.Executable`, `os.Stat`, `syscall.Exec` |
| `pair-go launch` docs | `README.md`, `atlas/architecture.md`, `atlas/go-migration-inventory.md` | modified | operator-facing migration contract |

**legacyRuntime** — the thin IO shell that resolves the current binary path, validates sibling `pair`, and replaces the current process with it.

- **Injected into:** `runWithLegacyRuntime` in tests, so process behavior is asserted with fakes rather than actually starting zellij.
- **Future extensions:** Native launch can replace only this runner once the pure launcher core owns side effects.

**pair-go launch docs** — documentation that explains the current migration boundary.

- **Injected into:** README command usage and atlas migration notes.
- **Future extensions:** #78/#79 can update the same docs when stateful shell glue or packaging changes.

## Chunk 2: Spec And Handoff Tests

### Task 1: Add pure launch request tests

**Files:**
- Create: `cmd/internal/entrypoint/launch_test.go`
- Create: `cmd/internal/entrypoint/launch.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestResolveLegacyLaunchDropsLaunchVerb(t *testing.T) {
	req := entrypoint.ResolveLegacyLaunch("/repo/bin/pair-go", []string{"claude", "--", "--resume"})
	if req.Path != "/repo/bin/pair" {
		t.Fatalf("Path = %q", req.Path)
	}
	want := []string{"pair", "claude", "--", "--resume"}
	if !reflect.DeepEqual(req.Argv, want) {
		t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
	}
}

func TestResolveLegacyLaunchPreservesSubcommands(t *testing.T) {
	req := entrypoint.ResolveLegacyLaunch("/repo/bin/pair-go", []string{"resume", "demo"})
	want := []string{"pair", "resume", "demo"}
	if !reflect.DeepEqual(req.Argv, want) {
		t.Fatalf("Argv = %#v, want %#v", req.Argv, want)
	}
}
```

- [ ] **Step 2: Run the tests to verify RED**

Run: `go test ./cmd/internal/entrypoint -run TestResolveLegacyLaunch -count=1`

Expected: FAIL because `cmd/internal/entrypoint` does not exist yet.

- [ ] **Step 3: Implement minimal pure resolver**

Create `LegacyLaunchRequest` with `Path string` and `Argv []string`. Implement `ResolveLegacyLaunch(executable string, launchArgs []string)` using `filepath.Dir(executable)` and sibling `pair`. The argv must start with `"pair"` and append all launch args unchanged.

- [ ] **Step 4: Run tests to verify GREEN**

Run: `go test ./cmd/internal/entrypoint -run TestResolveLegacyLaunch -count=1`

Expected: PASS.

### Task 2: Add `pair-go launch` process-boundary tests

**Files:**
- Modify: `cmd/pair-go/main.go`
- Modify: `cmd/pair-go/main_test.go`
- Modify: `cmd/pair-go/launch_process_test.go`
- Modify: `cmd/internal/dispatcher/dispatcher_test.go`

- [ ] **Step 1: Write failing main tests for handoff**

Add a fake runner test that calls the test seam with args `[]string{"launch", "claude", "--", "--resume"}` and asserts:

- exit code is the fake runner's code;
- no dispatcher prototype text is printed;
- runner receives sibling `bin/pair`;
- runner receives argv `["pair", "claude", "--", "--resume"]`.
- runner receives inherited environment entries, including a sentinel such as `PAIR_TEST_ENV=kept`, so the shell launcher sees the same env that `pair-go` received (`ARCH-PURPOSE` compatibility surface).

Also add a missing-launcher test that returns a not-found stat result and asserts stderr mentions `pair-go launch`, `bin/pair`, `make build`, `make install`, and `dev-aliases.sh`.

- [ ] **Step 2: Run targeted tests to verify RED**

Run: `go test ./cmd/pair-go -run 'TestRunLaunch' -count=1`

Expected: FAIL because current `pair-go launch` returns prototype exit code 3 and never invokes the fake runner.

- [ ] **Step 3: Update main seam**

Refactor `cmd/pair-go/main.go` so `run` delegates to `runWithLegacyRuntime`. For `args[0] == "launch"`, use the injected `legacyRuntime` and `entrypoint.ResolveLegacyLaunch`; otherwise preserve existing dispatcher behavior. The real runner should:

- get `os.Executable()`;
- resolve sibling `pair`;
- `os.Stat` it and require a non-directory executable path;
- call `syscall.Exec(path, argv, os.Environ())`.

In tests, fake these methods and return deterministic codes.

- [ ] **Step 4: Update stale prototype expectations**

Change launch tests that currently expect `"prototype decision"` and exit code `3` to expect legacy handoff behavior or delete them if covered by the new fake-runner tests. Keep dispatcher tests for `context`, `scrollback-render`, help, version, and planned command errors.

- [ ] **Step 5: Run targeted tests to verify GREEN**

Run: `go test ./cmd/pair-go ./cmd/internal/dispatcher ./cmd/internal/entrypoint -count=1`

Expected: PASS.

## Chunk 3: Build Wiring And Documentation

### Task 3: Keep Makefile wiring accurate

**Files:**
- Modify: `Makefile.local`
- Create: `tests/pair-go-install-layout-test.sh`

- [ ] **Step 1: Write or update dependency expectation**

Inspect the `$(BIN_DIR)/pair-go` dependency list and add `cmd/internal/entrypoint/launch.go` so `make pair-go` rebuilds when the new resolver changes. Install `pair` and `pair-dev` as symlinks beside installed Go binaries so copied `pair-go` can resolve sibling `pair` while the shell launcher still resolves `PAIR_HOME` back to the source tree.

- [ ] **Step 2: Add installed-layout regression test**

Create `tests/pair-go-install-layout-test.sh` that runs `HOME=<tmp> make install`, asserts `$HOME/.local/bin/pair` and `pair-dev` are symlinks, and runs `$HOME/.local/bin/pair-go launch --help` to prove the installed handoff reaches the existing `pair` help.

- [ ] **Step 3: Run build**

Run: `make pair-go`

Expected: `bin/pair-go` builds successfully.

### Task 4: Document the migration boundary

**Files:**
- Modify: `README.md`
- Modify: `atlas/architecture.md`
- Modify: `atlas/go-migration-inventory.md`
- Modify: `workshop/issues/000077-go-entrypoint-switch.md`

- [ ] **Step 1: Update README**

Add a short development note near Command Usage:

```markdown
`pair-go launch ...` is the Go-owned migration entrypoint for testing the launcher path. It accepts the same arguments after `launch` that `pair` accepts directly, then hands off to the current `pair` launcher for one migration window. In a dev shell sourced from `../ariadne/construct/dev-aliases.sh`, `pair-go` rebuilds from `cmd/pair-go` automatically before running; no `pair-go-dev` command is needed.
```

- [ ] **Step 2: Update atlas**

Update `atlas/architecture.md` and `atlas/go-migration-inventory.md` so they no longer describe `pair-go launch` as decision-phase only. State that #77 makes it a Go-owned compatibility handoff to `bin/pair`, while `bin/pair` remains the public stable entrypoint and the real zellij lifecycle remains shell-owned.

- [ ] **Step 3: Tick issue plan items and log verification intent**

In #77, tick completed plan rows as they land and add a log entry with the exact commands run.

## Chunk 4: Verification And Close

### Task 5: Full verification

**Files:**
- No new code files.

- [ ] **Step 1: Run focused Go tests**

Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1`

Expected: PASS.

- [ ] **Step 2: Run full Go suite**

Run: `go test ./... -count=1`

Expected: PASS.

- [ ] **Step 3: Run build**

Run: `make pair-go`

Expected: PASS and `bin/pair-go` exists.

- [ ] **Step 4: Smoke test non-zellij routes**

Run:

```bash
make test-dev-rebuild
make test-pair-go-install-layout
bin/pair-go help
bin/pair-go launch --help
bin/pair --help
bin/pair-dev --help
```

Expected: `make test-dev-rebuild` passes, proving the existing `PAIR_DEV` rebuild hook still works. `make test-pair-go-install-layout` passes, proving `make install` creates the sibling launcher layout that `pair-go launch` requires. Help output succeeds. `bin/pair-go launch --help` should print the existing `pair` help because it hands off to `bin/pair --help`.

- [ ] **Step 5: SDLC close**

Run: `sdlc close --issue 77 --verified '<focused tests, full go test, build, and help smoke evidence>'`.

Expected: SDLC close runs its review gate; fix any Critical/Important findings before merge.

## Revisions

### 2026-06-30

Boundary review found two plan/code mismatches. First, the implemented integration seam is `legacyRuntime` / `runWithLegacyRuntime`, not the original `LegacyLauncherRunner` / `runWithLegacyRunner` names; the Core concepts table and implementation steps now match the code. Second, `make install` originally installed `pair-go` without sibling `pair`, making the documented recovery path false; the plan now includes shell-launcher symlink install wiring plus an installed-layout regression test.

codex
The tracker now records the prior REWORK and the plan has a revision for both earlier blockers. I’m checking the actual Makefile/test wiring and running the new install-layout path, since that was the concrete behavioral failure.
exec
/bin/zsh -lc "nl -ba Makefile.local | sed -n '1,80p;170,205p;238,252p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	# Repo-specific Makefile targets.
     2	# This file is included by Makefile — add your own targets here.
     3	
     4	.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
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
    30	SHELL_BINS  := pair pair-dev
    31	
    32	build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
    33	
    34	install: build
    35		@mkdir -p $(INSTALL_DIR)
    36		@for b in $(GO_BINS); do \
    37		    install -m 755 $(BIN_DIR)/$$b $(INSTALL_DIR)/$$b && \
    38		    echo "installed: $(INSTALL_DIR)/$$b"; \
    39		done
    40		@for b in $(SHELL_BINS); do \
    41		    ln -sf "$(CURDIR)/$(BIN_DIR)/$$b" "$(INSTALL_DIR)/$$b" && \
    42		    echo "linked: $(INSTALL_DIR)/$$b -> $(CURDIR)/$(BIN_DIR)/$$b"; \
    43		done
    44	
    45	# Per-binary aliases — `make pair-scribe`, etc.
    46	pair-scribe:            $(BIN_DIR)/pair-scribe
    47	pair-scrollback-render: $(BIN_DIR)/pair-scrollback-render
    48	pair-wrap:              $(BIN_DIR)/pair-wrap
    49	pair-slug:              $(BIN_DIR)/pair-slug
    50	pair-continuation:      $(BIN_DIR)/pair-continuation
    51	pair-changelog:         $(BIN_DIR)/pair-changelog
    52	pair-context:           $(BIN_DIR)/pair-context
    53	pair-go:                $(BIN_DIR)/pair-go
    54	
    55	# Tests
    56	#   make test       → go test ./... (cache disabled so reruns actually run)
    57	#   make test-race  → race-detector run, scoped to packages where the
    58	#                     suite has actual concurrent code worth checking.
    59	#                     pair-wrap has translateStdin's goroutine + timer +
    60	#                     channels; pair-scrollback-render's render() spawns
    61	#                     a goroutine drainer that races with em.Close()
    62	#                     inside the vt.Emulator library API itself, which is
    63	#                     outside our scope to fix — exclude that package
    64	#                     here rather than chase a library-level race.
    65	test: test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace test-review
    66		go test ./... -count=1
    67	
    68	# pair-title poller: single-instance guard (identity-checked liveness so a
    69	# recycled PID can't suppress the respawn) + the #71 frame meter (renames each
    70	# agent pane to "<agent> (<count>) [<cwd>]", with unchanged-skip). See
    71	# tests/pair-title-poller-test.sh.
    72	test-pair-title:
    73		bash tests/pair-title-poller-test.sh
    74	
    75	# cmux workspace ownership claim (presence beats a stale owner flag), so a
    76	# launch/attach/restart reclaims its workspace title. See
    77	# tests/cmux-ownership-test.sh.
    78	test-cmux-ownership:
    79		bash tests/cmux-ownership-test.sh
    80	
   170	
   171	# copy-on-select in_nvim detection: the agent overwrites its pane title with
   172	# "claude [<cwd>]", so a repo path containing "nvim" (e.g. parley.nvim) must not
   173	# be mistaken for the nvim draft pane (which would skip the paste). Keys on
   174	# terminal_command, not the cwd-polluted title. See tests/copy-on-select-test.sh.
   175	test-copy-on-select:
   176		bash tests/copy-on-select-test.sh
   177	
   178	# Cross-emitter contract for the flight recorder: Go == shell == Lua schema,
   179	# plus concurrent-append line-atomicity. See tests/adapt-schema-test.sh.
   180	test-adapt-schema:
   181		bash tests/adapt-schema-test.sh
   182	
   183	# pair-dev rebuild hook (#000046): deployed mode (PAIR_DEV unset) must invoke
   184	# no toolchain; dev mode builds; build failure is errexit-safe. See
   185	# tests/dev-rebuild-test.sh.
   186	test-dev-rebuild:
   187		bash tests/dev-rebuild-test.sh
   188	
   189	# pair-go launch handoff (#77): installed pair-go requires sibling pair. Local
   190	# installs link shell launchers back to the source tree so pair keeps its
   191	# PAIR_HOME asset resolution.
   192	test-pair-go-install-layout:
   193		bash tests/pair-go-install-layout-test.sh
   194	
   195	# pair-doctor emitter-health probe (#000047): marker check + binary selection
   196	# (running-via-pidfile vs PATH). See tests/emitter-health-test.sh.
   197	test-emitter-health:
   198		bash tests/emitter-health-test.sh
   199	
   200	# `pair continue` arg-parsing + session-name guard (#000054): tag not forced,
   201	# -- args forwarded, [agent] port, bare list / error paths, guard short-vs-long.
   202	# Drives the real bin/pair via its PAIR_DEBUG_ARGS probe. See tests/pair-continue-test.sh.
   203	test-continue:
   204		bash tests/pair-continue-test.sh
   205	
   238	# for the pair-title poller. Pure ctxmeter reader + shared transcript resolver.
   239	$(BIN_DIR)/pair-context: cmd/pair-context/main.go cmd/internal/contextcmd/contextcmd.go cmd/internal/ctxmeter/ctxmeter.go cmd/internal/transcript/transcript.go go.mod
   240		go build -o $@ ./cmd/pair-context
   241	
   242	# pair-changelog — distills the pair TTY into the Alt+l change log (#53);
   243	# shares cmd/internal/model with pair-slug.
   244	$(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill.go cmd/pair-changelog/prompt.go cmd/internal/model/model.go go.mod
   245		go build -o $@ ./cmd/pair-changelog
   246	
   247	# pair-continuation — the deterministic writer for the `continuation` datatype
   248	# (ariadne#91): renders frontmatter, allocates a collision-safe timestamped
   249	# name under workshop/continuation/, writes, then commits + pushes (recovery).
   250	$(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-continuation/git.go cmd/pair-continuation/main.go go.mod
   251		go build -o $@ ./cmd/pair-continuation
   252	

exec
/bin/zsh -lc "nl -ba tests/pair-go-install-layout-test.sh && nl -ba cmd/pair-go/main.go | sed -n '1,100p' && nl -ba cmd/internal/entrypoint/launch.go" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	#!/usr/bin/env bash
     2	set -euo pipefail
     3	
     4	repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
     5	tmp_home="$(mktemp -d "${TMPDIR:-/tmp}/pair-go-install-layout.XXXXXX")"
     6	trap 'rm -rf "$tmp_home"' EXIT
     7	gomodcache="$(go env GOMODCACHE)"
     8	gocache="$(go env GOCACHE)"
     9	
    10	HOME="$tmp_home" GOMODCACHE="$gomodcache" GOCACHE="$gocache" make -C "$repo_root" install >/dev/null
    11	
    12	install_bin="$tmp_home/.local/bin"
    13	test -x "$install_bin/pair-go"
    14	test -L "$install_bin/pair"
    15	test -L "$install_bin/pair-dev"
    16	
    17	out="$("$install_bin/pair-go" launch --help)"
    18	case "$out" in
    19	    pair\ —*) ;;
    20	    *)
    21	        printf 'pair-go launch --help did not reach pair help; first bytes:\n%s\n' "$out" >&2
    22	        exit 1
    23	        ;;
    24	esac
    25	
    26	printf 'pair-go install layout test passed\n'
     1	// pair-go is the development dispatcher for the future primary Go CLI. Its
     2	// launch route is a compatibility handoff to the current shell launcher.
     3	package main
     4	
     5	import (
     6		"fmt"
     7		"io"
     8		"os"
     9		"syscall"
    10	
    11		"github.com/xianxu/pair/cmd/internal/dispatcher"
    12		"github.com/xianxu/pair/cmd/internal/entrypoint"
    13	)
    14	
    15	func main() {
    16		os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
    17	}
    18	
    19	func run(args []string, stdout, stderr io.Writer) int {
    20		return runWithLegacyRuntime(args, stdout, stderr, osLegacyRuntime{})
    21	}
    22	
    23	type legacyRuntime interface {
    24		Executable() (string, error)
    25		Stat(path string) error
    26		Environ() []string
    27		Exec(path string, argv []string, env []string) int
    28	}
    29	
    30	func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
    31		if len(args) > 0 && args[0] == "launch" {
    32			return runLegacyLaunch(args[1:], stderr, rt)
    33		}
    34		res := dispatcher.Dispatch(args)
    35		return writeResult(res, stdout, stderr)
    36	}
    37	
    38	func runLegacyLaunch(args []string, stderr io.Writer, rt legacyRuntime) int {
    39		exe, err := rt.Executable()
    40		if err != nil {
    41			_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
    42			return 1
    43		}
    44		req := entrypoint.ResolveLegacyLaunch(exe, args)
    45		if err := rt.Stat(req.Path); err != nil {
    46			_, _ = fmt.Fprintf(stderr, "pair-go launch: pair launcher not found at %s (%v); run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", req.Path, err)
    47			return 1
    48		}
    49		return rt.Exec(req.Path, req.Argv, rt.Environ())
    50	}
    51	
    52	type osLegacyRuntime struct{}
    53	
    54	func (osLegacyRuntime) Executable() (string, error) {
    55		return os.Executable()
    56	}
    57	
    58	func (osLegacyRuntime) Stat(path string) error {
    59		info, err := os.Stat(path)
    60		if err != nil {
    61			return err
    62		}
    63		if info.IsDir() {
    64			return fmt.Errorf("is a directory")
    65		}
    66		return nil
    67	}
    68	
    69	func (osLegacyRuntime) Environ() []string {
    70		return os.Environ()
    71	}
    72	
    73	func (osLegacyRuntime) Exec(path string, argv []string, env []string) int {
    74		if err := syscall.Exec(path, argv, env); err != nil {
    75			_, _ = fmt.Fprintf(os.Stderr, "pair-go launch: exec %s failed: %v\n", path, err)
    76			return 1
    77		}
    78		return 0
    79	}
    80	
    81	func writeResult(res dispatcher.Result, stdout, stderr io.Writer) int {
    82		if res.Stdout != "" {
    83			_, _ = io.WriteString(stdout, res.Stdout)
    84		}
    85		if res.Stderr != "" {
    86			_, _ = io.WriteString(stderr, res.Stderr)
    87		}
    88		return res.ExitCode
    89	}
     1	package entrypoint
     2	
     3	import "path/filepath"
     4	
     5	// LegacyLaunchRequest describes the current compatibility handoff from
     6	// pair-go launch to the shell-owned pair launcher.
     7	type LegacyLaunchRequest struct {
     8		Path string
     9		Argv []string
    10	}
    11	
    12	func ResolveLegacyLaunch(executable string, launchArgs []string) LegacyLaunchRequest {
    13		argv := make([]string, 0, len(launchArgs)+1)
    14		argv = append(argv, "pair")
    15		argv = append(argv, launchArgs...)
    16		return LegacyLaunchRequest{
    17			Path: filepath.Join(filepath.Dir(executable), "pair"),
    18			Argv: argv,
    19		}
    20	}

exec
/bin/zsh -lc 'git diff --stat b829ef00346738e3c3d7e40db38c2c03f51e615b..HEAD && git diff --name-only b829ef00346738e3c3d7e40db38c2c03f51e615b..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 Makefile.local                                     |   22 +-
 README.md                                          |    7 +
 atlas/architecture.md                              |   24 +-
 atlas/go-migration-inventory.md                    |    6 +-
 cmd/internal/dispatcher/dispatcher.go              |   97 +-
 cmd/internal/dispatcher/dispatcher_test.go         |   57 +-
 cmd/internal/entrypoint/launch.go                  |   20 +
 cmd/internal/entrypoint/launch_test.go             |   28 +
 cmd/pair-go/launch_process_test.go                 |   71 -
 cmd/pair-go/main.go                                |   65 +-
 cmd/pair-go/main_test.go                           |  114 +-
 tests/pair-go-install-layout-test.sh               |   26 +
 workshop/issues/000077-go-entrypoint-switch.md     |   44 +-
 .../000077-go-entrypoint-switch-close-review.md    | 3581 ++++++++++++++++++++
 workshop/plans/000077-go-entrypoint-switch-plan.md |   26 +-
 15 files changed, 3915 insertions(+), 273 deletions(-)
Makefile.local
README.md
atlas/architecture.md
atlas/go-migration-inventory.md
cmd/internal/dispatcher/dispatcher.go
cmd/internal/dispatcher/dispatcher_test.go
cmd/internal/entrypoint/launch.go
cmd/internal/entrypoint/launch_test.go
cmd/pair-go/launch_process_test.go
cmd/pair-go/main.go
cmd/pair-go/main_test.go
tests/pair-go-install-layout-test.sh
workshop/issues/000077-go-entrypoint-switch.md
workshop/plans/000077-go-entrypoint-switch-close-review.md
workshop/plans/000077-go-entrypoint-switch-plan.md

exec
/bin/zsh -lc 'rg -n "LegacyLauncherRunner|runWithLegacyRunner|legacyRuntime|runWithLegacyRuntime|test-pair-go-install-layout|SHELL_BINS" workshop/plans/000077-go-entrypoint-switch-plan.md workshop/issues/000077-go-entrypoint-switch.md Makefile.local cmd tests' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
workshop/issues/000077-go-entrypoint-switch.md:68:- 2026-06-30: closed — go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1; go test ./... -count=1; make pair-go; make test-dev-rebuild; make test-pair-go-install-layout; bin/pair-go help; bin/pair-go launch --help; bin/pair --help; bin/pair-dev --help
workshop/issues/000077-go-entrypoint-switch.md:77:Boundary review returned REWORK. Fixed both blockers: `make install` now links `pair` and `pair-dev` beside installed Go binaries so installed `pair-go launch ...` has the sibling launcher it requires, and `tests/pair-go-install-layout-test.sh` covers that temp-install layout. Revised the durable plan's Core concepts table to match the implemented `legacyRuntime` / `runWithLegacyRuntime` names.
workshop/issues/000077-go-entrypoint-switch.md:85:- `make test-pair-go-install-layout`
workshop/plans/000077-go-entrypoint-switch-plan.md:38:| `legacyRuntime` | `cmd/pair-go/main.go` | new | `os.Executable`, `os.Stat`, `syscall.Exec` |
workshop/plans/000077-go-entrypoint-switch-plan.md:41:**legacyRuntime** — the thin IO shell that resolves the current binary path, validates sibling `pair`, and replaces the current process with it.
workshop/plans/000077-go-entrypoint-switch-plan.md:43:- **Injected into:** `runWithLegacyRuntime` in tests, so process behavior is asserted with fakes rather than actually starting zellij.
workshop/plans/000077-go-entrypoint-switch-plan.md:126:Refactor `cmd/pair-go/main.go` so `run` delegates to `runWithLegacyRuntime`. For `args[0] == "launch"`, use the injected `legacyRuntime` and `entrypoint.ResolveLegacyLaunch`; otherwise preserve existing dispatcher behavior. The real runner should:
workshop/plans/000077-go-entrypoint-switch-plan.md:222:make test-pair-go-install-layout
workshop/plans/000077-go-entrypoint-switch-plan.md:229:Expected: `make test-dev-rebuild` passes, proving the existing `PAIR_DEV` rebuild hook still works. `make test-pair-go-install-layout` passes, proving `make install` creates the sibling launcher layout that `pair-go launch` requires. Help output succeeds. `bin/pair-go launch --help` should print the existing `pair` help because it hands off to `bin/pair --help`.
workshop/plans/000077-go-entrypoint-switch-plan.md:241:Boundary review found two plan/code mismatches. First, the implemented integration seam is `legacyRuntime` / `runWithLegacyRuntime`, not the original `LegacyLauncherRunner` / `runWithLegacyRunner` names; the Core concepts table and implementation steps now match the code. Second, `make install` originally installed `pair-go` without sibling `pair`, making the documented recovery path false; the plan now includes shell-launcher symlink install wiring plus an installed-layout regression test.
Makefile.local:4:.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
Makefile.local:30:SHELL_BINS  := pair pair-dev
Makefile.local:40:	@for b in $(SHELL_BINS); do \
Makefile.local:65:test: test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace test-review
Makefile.local:192:test-pair-go-install-layout:
cmd/pair-go/main_test.go:45:	code := runWithLegacyRuntime([]string{"launch", "--help"}, &stdout, &stderr, rt)
cmd/pair-go/main_test.go:72:	code := runWithLegacyRuntime([]string{"launch", "claude", "--", "--resume"}, &stdout, &stderr, rt)
cmd/pair-go/main_test.go:102:	code := runWithLegacyRuntime([]string{"launch", "claude"}, &stdout, &stderr, rt)
cmd/pair-go/main.go:20:	return runWithLegacyRuntime(args, stdout, stderr, osLegacyRuntime{})
cmd/pair-go/main.go:23:type legacyRuntime interface {
cmd/pair-go/main.go:30:func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
cmd/pair-go/main.go:38:func runLegacyLaunch(args []string, stderr io.Writer, rt legacyRuntime) int {

codex
The new install wiring is present: `make install` links `pair` and `pair-dev` beside copied Go binaries, and the regression test asserts that installed `pair-go launch --help` reaches `pair` help. I’m running the focused tests and the new install smoke now, then I’ll decide whether remaining tracker inconsistencies are blocking.
exec
/bin/zsh -lc 'bin/pair-go launch --help >/tmp/pair-go-review-help.out 2>/tmp/pair-go-review-help.err; code=$?; printf '"'code=%s\\nstdout_first=%s\\nstderr=%s\\n' \""'$code" "$(head -n 1 /tmp/pair-go-review-help.out)" "$(cat /tmp/pair-go-review-help.err)"' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
code=0
stdout_first=pair — Neovim-backed input field for any TUI coding agent.
stderr=

exec
/bin/zsh -lc 'make test-pair-go-install-layout' in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc 'go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1' in /Users/xianxu/workspace/pair
 succeeded in 1629ms:
ok  	github.com/xianxu/pair/cmd/internal/entrypoint	0.191s
ok  	github.com/xianxu/pair/cmd/pair-go	1.454s
ok  	github.com/xianxu/pair/cmd/internal/dispatcher	0.466s

 succeeded in 2941ms:
go: writing stat cache: open /Users/xianxu/go/pkg/mod/cache/download/github.com/xianxu/pair/@v/v0.0.0-20260630223646-c1e28031788a.info676788394.tmp: operation not permitted
go: writing stat cache: open /Users/xianxu/go/pkg/mod/cache/download/github.com/xianxu/pair/@v/v0.0.0-20260630223646-c1e28031788a.info523836955.tmp: operation not permitted
go: writing stat cache: open /Users/xianxu/go/pkg/mod/cache/download/github.com/xianxu/pair/@v/v0.0.0-20260630223646-c1e28031788a.info477324646.tmp: operation not permitted
go: writing stat cache: open /Users/xianxu/go/pkg/mod/cache/download/github.com/xianxu/pair/@v/v0.0.0-20260630223646-c1e28031788a.info244103414.tmp: operation not permitted
go: writing stat cache: open /Users/xianxu/go/pkg/mod/cache/download/github.com/xianxu/pair/@v/v0.0.0-20260630223646-c1e28031788a.info772849168.tmp: operation not permitted
go: writing stat cache: open /Users/xianxu/go/pkg/mod/cache/download/github.com/xianxu/pair/@v/v0.0.0-20260630223646-c1e28031788a.info525718179.tmp: operation not permitted
go: writing stat cache: open /Users/xianxu/go/pkg/mod/cache/download/github.com/xianxu/pair/@v/v0.0.0-20260630223646-c1e28031788a.info975862139.tmp: operation not permitted
go: writing stat cache: open /Users/xianxu/go/pkg/mod/cache/download/github.com/xianxu/pair/@v/v0.0.0-20260630223646-c1e28031788a.info973812713.tmp: operation not permitted
pair-go install layout test passed

codex
The focused Go tests and the installed-layout regression both pass. The install test emits Go stat-cache permission warnings from the sandboxed module cache, but the command exits 0 and proves the sibling `pair` layout works; I’m doing one final broader verification pass.
exec
/bin/zsh -lc 'git diff --check b829ef00346738e3c3d7e40db38c2c03f51e615b..HEAD' in /Users/xianxu/workspace/pair
 exited 2 in 0ms:
workshop/plans/000077-go-entrypoint-switch-close-review.md:218: space before tab in indent.
+ 	go build -o $@ ./cmd/pair-continuation
workshop/plans/000077-go-entrypoint-switch-close-review.md:219: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:227: space before tab in indent.
+ 	go build -o $@ ./cmd/pair-go
workshop/plans/000077-go-entrypoint-switch-close-review.md:228: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:229: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:237: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:246: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:253: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:264: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:287: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:295: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:320: space before tab in indent.
+ 	"bytes"
workshop/plans/000077-go-entrypoint-switch-close-review.md:321: space before tab in indent.
+ 	"fmt"
workshop/plans/000077-go-entrypoint-switch-close-review.md:323: space before tab in indent.
+ 	"strings"
workshop/plans/000077-go-entrypoint-switch-close-review.md:325: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:326: space before tab in indent.
+ 	"github.com/xianxu/pair/cmd/internal/contextcmd"
workshop/plans/000077-go-entrypoint-switch-close-review.md:328: space before tab in indent.
+ 	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
workshop/plans/000077-go-entrypoint-switch-close-review.md:330: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:334: space before tab in indent.
+ 	return []CommandFamily{
workshop/plans/000077-go-entrypoint-switch-close-review.md:337: space before tab in indent.
+ 		{Name: "context", Summary: "agent pane context meter", Status: "implemented"},
workshop/plans/000077-go-entrypoint-switch-close-review.md:338: space before tab in indent.
+ 		{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "implemented"},
workshop/plans/000077-go-entrypoint-switch-close-review.md:339: space before tab in indent.
+ 		{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
workshop/plans/000077-go-entrypoint-switch-close-review.md:341: space before tab in indent.
+ 		return Result{Stdout: Help(programName), ExitCode: 0}
workshop/plans/000077-go-entrypoint-switch-close-review.md:342: space before tab in indent.
+ 	case "version", "--version":
workshop/plans/000077-go-entrypoint-switch-close-review.md:343: space before tab in indent.
+ 		return Result{
workshop/plans/000077-go-entrypoint-switch-close-review.md:346: space before tab in indent.
+ 			ExitCode: 0,
workshop/plans/000077-go-entrypoint-switch-close-review.md:347: space before tab in indent.
+ 		}
workshop/plans/000077-go-entrypoint-switch-close-review.md:348: space before tab in indent.
+ 	case "launch":
workshop/plans/000077-go-entrypoint-switch-close-review.md:351: space before tab in indent.
+ 	case "context":
workshop/plans/000077-go-entrypoint-switch-close-review.md:352: space before tab in indent.
+ 		return dispatchContext(args[1:])
workshop/plans/000077-go-entrypoint-switch-close-review.md:353: space before tab in indent.
+ 	case "scrollback-render":
workshop/plans/000077-go-entrypoint-switch-close-review.md:355: space before tab in indent.
+ 	return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}
workshop/plans/000077-go-entrypoint-switch-close-review.md:357: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:402: space before tab in indent.
+ 	return Result{
workshop/plans/000077-go-entrypoint-switch-close-review.md:445: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:447: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:449: space before tab in indent.
+ 	for _, family := range Families() {
workshop/plans/000077-go-entrypoint-switch-close-review.md:450: space before tab in indent.
+ 		if family.Status == "prototype" {
workshop/plans/000077-go-entrypoint-switch-close-review.md:451: space before tab in indent.
+ 			fmt.Fprintf(&b, "  %-17s %s (prototype; decision-phase only)\n", family.Name, family.Summary)
workshop/plans/000077-go-entrypoint-switch-close-review.md:454: space before tab in indent.
+ 		} else if family.Status == "implemented" {
workshop/plans/000077-go-entrypoint-switch-close-review.md:455: space before tab in indent.
+ 			fmt.Fprintf(&b, "  %-17s %s (implemented helper route)\n", family.Name, family.Summary)
workshop/plans/000077-go-entrypoint-switch-close-review.md:456: space before tab in indent.
+ 		}
workshop/plans/000077-go-entrypoint-switch-close-review.md:462: space before tab in indent.
+ 				"Usage: pair-go <command> [args]",
workshop/plans/000077-go-entrypoint-switch-close-review.md:463: space before tab in indent.
+ 				"Implemented commands:",
workshop/plans/000077-go-entrypoint-switch-close-review.md:464: space before tab in indent.
+ 				"launch",
workshop/plans/000077-go-entrypoint-switch-close-review.md:467: space before tab in indent.
+ 				"context",
workshop/plans/000077-go-entrypoint-switch-close-review.md:468: space before tab in indent.
+ 				"scrollback-render",
workshop/plans/000077-go-entrypoint-switch-close-review.md:469: space before tab in indent.
+ 				"wrap",
workshop/plans/000077-go-entrypoint-switch-close-review.md:471: space before tab in indent.
+ 			if strings.Contains(res.Stdout, "launch             session lifecycle and public pair launcher flow (planned; not implemented") {
workshop/plans/000077-go-entrypoint-switch-close-review.md:472: space before tab in indent.
+ 				t.Fatalf("Stdout still labels launch unimplemented:\n%s", res.Stdout)
workshop/plans/000077-go-entrypoint-switch-close-review.md:473: space before tab in indent.
+ 			}
workshop/plans/000077-go-entrypoint-switch-close-review.md:477: space before tab in indent.
+ 			for _, stale := range []string{
workshop/plans/000077-go-entrypoint-switch-close-review.md:478: space before tab in indent.
+ 				"context           agent pane context meter (planned; not implemented",
workshop/plans/000077-go-entrypoint-switch-close-review.md:479: space before tab in indent.
+ 				"scrollback-render raw PTY capture to ANSI scrollback (planned; not implemented",
workshop/plans/000077-go-entrypoint-switch-close-review.md:481: space before tab in indent.
+ 	if res.Stderr != "" {
workshop/plans/000077-go-entrypoint-switch-close-review.md:482: space before tab in indent.
+ 		t.Fatalf("Stderr = %q, want empty", res.Stderr)
workshop/plans/000077-go-entrypoint-switch-close-review.md:483: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:486: space before tab in indent.
+ 		if !strings.Contains(res.Stdout, want) {
workshop/plans/000077-go-entrypoint-switch-close-review.md:487: space before tab in indent.
+ 			t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
workshop/plans/000077-go-entrypoint-switch-close-review.md:488: space before tab in indent.
+ 		}
workshop/plans/000077-go-entrypoint-switch-close-review.md:490: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:492: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:495: space before tab in indent.
+ 	res := Dispatch([]string{"launch", "--help"})
workshop/plans/000077-go-entrypoint-switch-close-review.md:542: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:543: space before tab in indent.
+ 	if res.Stdout != "" {
workshop/plans/000077-go-entrypoint-switch-close-review.md:544: space before tab in indent.
+ 		t.Fatalf("Stdout = %q, want empty", res.Stdout)
workshop/plans/000077-go-entrypoint-switch-close-review.md:545: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:548: space before tab in indent.
+ 		if !strings.Contains(res.Stderr, want) {
workshop/plans/000077-go-entrypoint-switch-close-review.md:549: space before tab in indent.
+ 			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
workshop/plans/000077-go-entrypoint-switch-close-review.md:550: space before tab in indent.
+ 		}
workshop/plans/000077-go-entrypoint-switch-close-review.md:698: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:701: space before tab in indent.
+ 	"io"
workshop/plans/000077-go-entrypoint-switch-close-review.md:702: space before tab in indent.
+ 	"os"
workshop/plans/000077-go-entrypoint-switch-close-review.md:704: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:705: space before tab in indent.
+ 	"github.com/xianxu/pair/cmd/internal/dispatcher"
workshop/plans/000077-go-entrypoint-switch-close-review.md:708: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:712: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:728: space before tab in indent.
+ 	res := dispatcher.Dispatch(args)
workshop/plans/000077-go-entrypoint-switch-close-review.md:729: space before tab in indent.
+ 	return writeResult(res, stdout, stderr)
workshop/plans/000077-go-entrypoint-switch-close-review.md:731: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:777: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:784: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:786: space before tab in indent.
+ 	"bytes"
workshop/plans/000077-go-entrypoint-switch-close-review.md:790: space before tab in indent.
+ 	"strings"
workshop/plans/000077-go-entrypoint-switch-close-review.md:791: space before tab in indent.
+ 	"testing"
workshop/plans/000077-go-entrypoint-switch-close-review.md:795: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:799: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:804: space before tab in indent.
+ 	var stdout, stderr bytes.Buffer
workshop/plans/000077-go-entrypoint-switch-close-review.md:807: space before tab in indent.
+ 	if code != 0 {
workshop/plans/000077-go-entrypoint-switch-close-review.md:808: space before tab in indent.
+ 		t.Fatalf("code = %d, want 0", code)
workshop/plans/000077-go-entrypoint-switch-close-review.md:809: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:814: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:815: space before tab in indent.
+ 	if stderr.String() != "" {
workshop/plans/000077-go-entrypoint-switch-close-review.md:816: space before tab in indent.
+ 		t.Fatalf("stderr = %q, want empty", stderr.String())
workshop/plans/000077-go-entrypoint-switch-close-review.md:817: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:826: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:835: space before tab in indent.
+ 	var stdout, stderr bytes.Buffer
workshop/plans/000077-go-entrypoint-switch-close-review.md:843: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:844: space before tab in indent.
+ 	if stdout.String() != "" {
workshop/plans/000077-go-entrypoint-switch-close-review.md:845: space before tab in indent.
+ 		t.Fatalf("stdout = %q, want empty", stdout.String())
workshop/plans/000077-go-entrypoint-switch-close-review.md:846: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:863: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:865: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:873: space before tab in indent.
+ 	var stdout, stderr bytes.Buffer
workshop/plans/000077-go-entrypoint-switch-close-review.md:881: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:882: space before tab in indent.
+ 	if stdout.String() != "" {
workshop/plans/000077-go-entrypoint-switch-close-review.md:883: space before tab in indent.
+ 		t.Fatalf("stdout = %q, want empty", stdout.String())
workshop/plans/000077-go-entrypoint-switch-close-review.md:884: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:887: space before tab in indent.
+ 		if !strings.Contains(stderr.String(), want) {
workshop/plans/000077-go-entrypoint-switch-close-review.md:888: space before tab in indent.
+ 			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
workshop/plans/000077-go-entrypoint-switch-close-review.md:889: space before tab in indent.
+ 		}
workshop/plans/000077-go-entrypoint-switch-close-review.md:890: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:895: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:938: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:1471: trailing whitespace.
+     4	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1477: trailing whitespace.
+    10	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1481: trailing whitespace.
+    14	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1485: trailing whitespace.
+    18	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1489: trailing whitespace.
+    22	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1496: trailing whitespace.
+    29	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1504: trailing whitespace.
+    37	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1518: trailing whitespace.
+    51	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1520: trailing whitespace.
+    53	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1524: trailing whitespace.
+    57	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1535: trailing whitespace.
+    68	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1539: trailing whitespace.
+    72	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1547: trailing whitespace.
+    80	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1562: trailing whitespace.
+     2	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1564: trailing whitespace.
+     4	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1571: trailing whitespace.
+    11	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1582: trailing whitespace.
+     2	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1587: trailing whitespace.
+     7	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1591: trailing whitespace.
+    11	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1593: trailing whitespace.
+    13	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1600: trailing whitespace.
+    20	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1607: trailing whitespace.
+    27	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1621: trailing whitespace.
+    41	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1627: trailing whitespace.
+    47	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1643: trailing whitespace.
+    63	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1650: trailing whitespace.
+    70	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1656: trailing whitespace.
+    76	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1662: trailing whitespace.
+    82	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1668: trailing whitespace.
+    88	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1675: trailing whitespace.
+    95	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1702: trailing whitespace.
+   122	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1716: trailing whitespace.
+     2	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1725: trailing whitespace.
+    11	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1739: trailing whitespace.
+    25	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1753: trailing whitespace.
+    39	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1777: trailing whitespace.
+    63	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1784: trailing whitespace.
+    70	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1787: trailing whitespace.
+    73	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1808: trailing whitespace.
+    94	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1814: trailing whitespace.
+   100	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1817: trailing whitespace.
+   103	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1833: trailing whitespace.
+   119	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1838: trailing whitespace.
+   124	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1843: trailing whitespace.
+   129	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1850: trailing whitespace.
+   136	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1854: trailing whitespace.
+   140	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1858: trailing whitespace.
+   144	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1865: trailing whitespace.
+   151	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1875: trailing whitespace.
+     2	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1880: trailing whitespace.
+     7	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1891: trailing whitespace.
+    18	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1942: trailing whitespace.
+   215	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1949: trailing whitespace.
+   222	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1952: trailing whitespace.
+   225	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1957: trailing whitespace.
+   230	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1962: trailing whitespace.
+   235	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1968: trailing whitespace.
+   241	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1974: trailing whitespace.
+   247	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1975: trailing whitespace.
+   248	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1981: trailing whitespace.
+     3	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1985: trailing whitespace.
+     7	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1989: trailing whitespace.
+    11	
workshop/plans/000077-go-entrypoint-switch-close-review.md:1991: trailing whitespace.
+    13	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2210: trailing whitespace.
+    19	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2212: trailing whitespace.
+    21	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2248: trailing whitespace.
+    57	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2265: trailing whitespace.
+    74	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2271: trailing whitespace.
+    80	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2290: trailing whitespace.
+    99	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2293: trailing whitespace.
+   102	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2300: trailing whitespace.
+   109	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2305: trailing whitespace.
+   114	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2309: trailing whitespace.
+   118	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2315: trailing whitespace.
+   124	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2328: trailing whitespace.
+   137	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2333: trailing whitespace.
+   142	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2335: trailing whitespace.
+   144	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2398: trailing whitespace.
+   207	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2404: trailing whitespace.
+   213	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2409: trailing whitespace.
+   218	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2414: trailing whitespace.
+   223	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2436: trailing whitespace.
+   245	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2470: trailing whitespace.
+    19	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2481: trailing whitespace.
+    30	
workshop/plans/000077-go-entrypoint-switch-close-review.md:2554: trailing whitespace.
+README.md:168:**Homebrew (recommended).** 
workshop/plans/000077-go-entrypoint-switch-close-review.md:2850: space before tab in indent.
+ 	"bytes"
workshop/plans/000077-go-entrypoint-switch-close-review.md:2851: space before tab in indent.
+ 	"fmt"
workshop/plans/000077-go-entrypoint-switch-close-review.md:2853: space before tab in indent.
+ 	"strings"
workshop/plans/000077-go-entrypoint-switch-close-review.md:2855: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:2856: space before tab in indent.
+ 	"github.com/xianxu/pair/cmd/internal/contextcmd"
workshop/plans/000077-go-entrypoint-switch-close-review.md:2858: space before tab in indent.
+ 	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
workshop/plans/000077-go-entrypoint-switch-close-review.md:2860: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:2864: space before tab in indent.
+ 	return []CommandFamily{
workshop/plans/000077-go-entrypoint-switch-close-review.md:2867: space before tab in indent.
+ 		{Name: "context", Summary: "agent pane context meter", Status: "implemented"},
workshop/plans/000077-go-entrypoint-switch-close-review.md:2868: space before tab in indent.
+ 		{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "implemented"},
workshop/plans/000077-go-entrypoint-switch-close-review.md:2869: space before tab in indent.
+ 		{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
workshop/plans/000077-go-entrypoint-switch-close-review.md:2871: space before tab in indent.
+ 		return Result{Stdout: Help(programName), ExitCode: 0}
workshop/plans/000077-go-entrypoint-switch-close-review.md:2872: space before tab in indent.
+ 	case "version", "--version":
workshop/plans/000077-go-entrypoint-switch-close-review.md:2873: space before tab in indent.
+ 		return Result{
workshop/plans/000077-go-entrypoint-switch-close-review.md:2876: space before tab in indent.
+ 			ExitCode: 0,
workshop/plans/000077-go-entrypoint-switch-close-review.md:2877: space before tab in indent.
+ 		}
workshop/plans/000077-go-entrypoint-switch-close-review.md:2878: space before tab in indent.
+ 	case "launch":
workshop/plans/000077-go-entrypoint-switch-close-review.md:2881: space before tab in indent.
+ 	case "context":
workshop/plans/000077-go-entrypoint-switch-close-review.md:2882: space before tab in indent.
+ 		return dispatchContext(args[1:])
workshop/plans/000077-go-entrypoint-switch-close-review.md:2883: space before tab in indent.
+ 	case "scrollback-render":
workshop/plans/000077-go-entrypoint-switch-close-review.md:2885: space before tab in indent.
+ 	return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}
workshop/plans/000077-go-entrypoint-switch-close-review.md:2887: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:2932: space before tab in indent.
+ 	return Result{
workshop/plans/000077-go-entrypoint-switch-close-review.md:2975: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:2977: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:2979: space before tab in indent.
+ 	for _, family := range Families() {
workshop/plans/000077-go-entrypoint-switch-close-review.md:2980: space before tab in indent.
+ 		if family.Status == "prototype" {
workshop/plans/000077-go-entrypoint-switch-close-review.md:2981: space before tab in indent.
+ 			fmt.Fprintf(&b, "  %-17s %s (prototype; decision-phase only)\n", family.Name, family.Summary)
workshop/plans/000077-go-entrypoint-switch-close-review.md:2984: space before tab in indent.
+ 		} else if family.Status == "implemented" {
workshop/plans/000077-go-entrypoint-switch-close-review.md:2985: space before tab in indent.
+ 			fmt.Fprintf(&b, "  %-17s %s (implemented helper route)\n", family.Name, family.Summary)
workshop/plans/000077-go-entrypoint-switch-close-review.md:2986: space before tab in indent.
+ 		}
workshop/plans/000077-go-entrypoint-switch-close-review.md:2992: space before tab in indent.
+ 				"Usage: pair-go <command> [args]",
workshop/plans/000077-go-entrypoint-switch-close-review.md:2993: space before tab in indent.
+ 				"Implemented commands:",
workshop/plans/000077-go-entrypoint-switch-close-review.md:2994: space before tab in indent.
+ 				"launch",
workshop/plans/000077-go-entrypoint-switch-close-review.md:2997: space before tab in indent.
+ 				"context",
workshop/plans/000077-go-entrypoint-switch-close-review.md:2998: space before tab in indent.
+ 				"scrollback-render",
workshop/plans/000077-go-entrypoint-switch-close-review.md:2999: space before tab in indent.
+ 				"wrap",
workshop/plans/000077-go-entrypoint-switch-close-review.md:3001: space before tab in indent.
+ 			if strings.Contains(res.Stdout, "launch             session lifecycle and public pair launcher flow (planned; not implemented") {
workshop/plans/000077-go-entrypoint-switch-close-review.md:3002: space before tab in indent.
+ 				t.Fatalf("Stdout still labels launch unimplemented:\n%s", res.Stdout)
workshop/plans/000077-go-entrypoint-switch-close-review.md:3003: space before tab in indent.
+ 			}
workshop/plans/000077-go-entrypoint-switch-close-review.md:3007: space before tab in indent.
+ 			for _, stale := range []string{
workshop/plans/000077-go-entrypoint-switch-close-review.md:3008: space before tab in indent.
+ 				"context           agent pane context meter (planned; not implemented",
workshop/plans/000077-go-entrypoint-switch-close-review.md:3009: space before tab in indent.
+ 				"scrollback-render raw PTY capture to ANSI scrollback (planned; not implemented",
workshop/plans/000077-go-entrypoint-switch-close-review.md:3011: space before tab in indent.
+ 	if res.Stderr != "" {
workshop/plans/000077-go-entrypoint-switch-close-review.md:3012: space before tab in indent.
+ 		t.Fatalf("Stderr = %q, want empty", res.Stderr)
workshop/plans/000077-go-entrypoint-switch-close-review.md:3013: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:3016: space before tab in indent.
+ 		if !strings.Contains(res.Stdout, want) {
workshop/plans/000077-go-entrypoint-switch-close-review.md:3017: space before tab in indent.
+ 			t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
workshop/plans/000077-go-entrypoint-switch-close-review.md:3018: space before tab in indent.
+ 		}
workshop/plans/000077-go-entrypoint-switch-close-review.md:3020: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:3022: trailing whitespace.
+ 
workshop/plans/000077-go-entrypoint-switch-close-review.md:3025: space before tab in indent.
+ 	res := Dispatch([]string{"launch", "--help"})
workshop/plans/000077-go-entrypoint-switch-close-review.md:3072: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:3073: space before tab in indent.
+ 	if res.Stdout != "" {
workshop/plans/000077-go-entrypoint-switch-close-review.md:3074: space before tab in indent.
+ 		t.Fatalf("Stdout = %q, want empty", res.Stdout)
workshop/plans/000077-go-entrypoint-switch-close-review.md:3075: space before tab in indent.
+ 	}
workshop/plans/000077-go-entrypoint-switch-close-review.md:3078: space before tab in indent.
+ 		if !strings.Contains(res.Stderr, want) {
workshop/plans/000077-go-entrypoint-switch-close-review.md:3079: space before tab in indent.
+ 			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
workshop/plans/000077-go-entrypoint-switch-close-review.md:3080: space before tab in indent.
+ 		}
workshop/plans/000077-go-entrypoint-switch-close-review.md:3117: trailing whitespace.
+     2	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3119: trailing whitespace.
+     4	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3121: trailing whitespace.
+     6	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3123: trailing whitespace.
+     8	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3125: trailing whitespace.
+    10	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3127: trailing whitespace.
+    12	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3129: trailing whitespace.
+    14	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3131: trailing whitespace.
+    16	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3136: trailing whitespace.
+    21	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3138: trailing whitespace.
+    23	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3142: trailing whitespace.
+    27	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3144: trailing whitespace.
+    29	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3148: trailing whitespace.
+    33	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3150: trailing whitespace.
+    35	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3155: trailing whitespace.
+    40	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3157: trailing whitespace.
+    42	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3160: trailing whitespace.
+    45	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3162: trailing whitespace.
+    47	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3165: trailing whitespace.
+    50	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3167: trailing whitespace.
+    52	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3169: trailing whitespace.
+    54	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3173: trailing whitespace.
+    58	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3175: trailing whitespace.
+    60	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3187: trailing whitespace.
+    72	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3196: trailing whitespace.
+    81	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3198: trailing whitespace.
+    83	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3200: trailing whitespace.
+    85	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3202: trailing whitespace.
+    87	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3204: trailing whitespace.
+    89	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3206: trailing whitespace.
+    91	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3208: trailing whitespace.
+    93	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3210: trailing whitespace.
+    95	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3212: trailing whitespace.
+    97	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3214: trailing whitespace.
+    99	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3220: trailing whitespace.
+   105	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3222: trailing whitespace.
+   107	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3224: trailing whitespace.
+   109	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3230: trailing whitespace.
+   115	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3232: trailing whitespace.
+   117	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3234: trailing whitespace.
+   119	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3236: trailing whitespace.
+   121	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3238: trailing whitespace.
+   123	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3240: trailing whitespace.
+   125	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3242: trailing whitespace.
+   127	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3247: trailing whitespace.
+   132	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3249: trailing whitespace.
+   134	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3251: trailing whitespace.
+   136	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3253: trailing whitespace.
+   138	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3255: trailing whitespace.
+   140	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3257: trailing whitespace.
+   142	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3259: trailing whitespace.
+   144	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3261: trailing whitespace.
+   146	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3263: trailing whitespace.
+   148	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3266: trailing whitespace.
+   151	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3268: trailing whitespace.
+   153	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3270: trailing whitespace.
+   155	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3272: trailing whitespace.
+   157	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3274: trailing whitespace.
+   159	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3276: trailing whitespace.
+   161	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3278: trailing whitespace.
+   163	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3284: trailing whitespace.
+   169	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3286: trailing whitespace.
+   171	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3288: trailing whitespace.
+   173	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3292: trailing whitespace.
+   177	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3294: trailing whitespace.
+   179	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3296: trailing whitespace.
+   181	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3298: trailing whitespace.
+   183	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3300: trailing whitespace.
+   185	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3302: trailing whitespace.
+   187	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3304: trailing whitespace.
+   189	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3307: trailing whitespace.
+   192	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3309: trailing whitespace.
+   194	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3311: trailing whitespace.
+   196	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3313: trailing whitespace.
+   198	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3315: trailing whitespace.
+   200	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3317: trailing whitespace.
+   202	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3319: trailing whitespace.
+   204	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3321: trailing whitespace.
+   206	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3323: trailing whitespace.
+   208	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3325: trailing whitespace.
+   210	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3327: trailing whitespace.
+   212	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3329: trailing whitespace.
+   214	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3337: trailing whitespace.
+   222	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3339: trailing whitespace.
+   224	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3341: trailing whitespace.
+   226	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3343: trailing whitespace.
+   228	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3393: trailing whitespace.
+    26	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3397: trailing whitespace.
+    30	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3399: trailing whitespace.
+    32	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3406: trailing whitespace.
+    39	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3409: trailing whitespace.
+    35	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3414: trailing whitespace.
+    40	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3416: trailing whitespace.
+    42	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3419: trailing whitespace.
+    45	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3426: trailing whitespace.
+    29	
workshop/plans/000077-go-entrypoint-switch-close-review.md:3434: trailing whitespace.
+    37	

exec
/bin/zsh -lc 'make test-dev-rebuild' in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc 'go test ./... -count=1' in /Users/xianxu/workspace/pair
 succeeded in 299ms:
  ok   dev (PAIR_DEV=1) runs make build
  ok   build failure is errexit-safe (launcher continues)

all dev-rebuild tests passed

 succeeded in 16753ms:
ok  	github.com/xianxu/pair/cmd/internal/adapt	0.158s
ok  	github.com/xianxu/pair/cmd/internal/contextcmd	0.303s
ok  	github.com/xianxu/pair/cmd/internal/ctxmeter	0.448s
ok  	github.com/xianxu/pair/cmd/internal/dispatcher	0.600s
ok  	github.com/xianxu/pair/cmd/internal/entrypoint	0.888s
ok  	github.com/xianxu/pair/cmd/internal/launcher	2.516s
ok  	github.com/xianxu/pair/cmd/internal/model	3.730s
ok  	github.com/xianxu/pair/cmd/internal/scrollbackcmd	1.172s
ok  	github.com/xianxu/pair/cmd/internal/transcript	1.308s
ok  	github.com/xianxu/pair/cmd/pair-changelog	16.383s
ok  	github.com/xianxu/pair/cmd/pair-context	4.132s
ok  	github.com/xianxu/pair/cmd/pair-continuation	5.960s
ok  	github.com/xianxu/pair/cmd/pair-go	4.327s
?   	github.com/xianxu/pair/cmd/pair-scribe	[no test files]
?   	github.com/xianxu/pair/cmd/pair-scrollback-render	[no test files]
ok  	github.com/xianxu/pair/cmd/pair-slug	6.795s
ok  	github.com/xianxu/pair/cmd/pair-wrap	1.938s

codex
The broader Go suite and `make test-dev-rebuild` pass. One hygiene check, `git diff --check`, fails only on the generated close-review transcript file because it embeds raw diffs/transcript lines with tabs and trailing spaces; I’m treating that separately from runtime correctness rather than conflating it with the launch change.
exec
/bin/zsh -lc "nl -ba bin/pair | sed -n '300,390p;440,500p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
   300	                    status="attached ($clients clients)"
   301	                fi
   302	            fi
   303	            printf '%-30s %-10s %s\n' "$sess" "$agent" "$status"
   304	        done <<< "$sessions"
   305	        exit 0
   306	        ;;
   307	    rename)
   308	        # `pair rename <old> <new>` — rename every tag-scoped file in
   309	        # $PAIR_DATA_DIR from <old> to <new>. Offline-only: refuses if
   310	        # `pair-<old>` is still in zellij's session list (live, detached,
   311	        # or resurrectable). The inside-session rename gesture (M2) takes
   312	        # care of the quit→rename→re-exec choreography on top of this
   313	        # primitive; the only safe live equivalent is to use that.
   314	        #
   315	        # File enumeration is by exact-name construction, never globbing —
   316	        # that way `pair rename brain newname` can't accidentally pick up
   317	        # `*-brain-2-*` files. The set of agent suffixes (claude/codex/
   318	        # agy) is hardcoded below; if a new agent is added to pair,
   319	        # add it there too.
   320	        # Optional flags before the positionals.
   321	        #   --restart-check : validate everything but skip the `pair-<old>`
   322	        #                     live-session refusal AND skip the actual mv.
   323	        #                     Used by the in-session rename gesture
   324	        #                     (Ctrl+Alt+n's (R)ename), where pair-<old> IS
   325	        #                     the current session — about to be killed by
   326	        #                     pair-restart.sh — so its presence in
   327	        #                     `zellij list-sessions` is expected, not a
   328	        #                     reason to refuse.
   329	        rename_check_only=0
   330	        rename_skip_old_live=0
   331	        rename_argi=2
   332	        while :; do
   333	            case "${!rename_argi:-}" in
   334	                --restart-check)
   335	                    rename_check_only=1
   336	                    rename_skip_old_live=1
   337	                    rename_argi=$((rename_argi + 1))
   338	                    ;;
   339	                --) rename_argi=$((rename_argi + 1)); break ;;
   340	                *)  break ;;
   341	            esac
   342	        done
   343	        old_tag="${!rename_argi:-}"
   344	        rename_argi=$((rename_argi + 1))
   345	        new_tag="${!rename_argi:-}"
   346	        rename_argi=$((rename_argi + 1))
   347	        if [ -z "$old_tag" ] || [ -z "$new_tag" ]; then
   348	            echo "usage: pair rename [--restart-check] <old> <new>" >&2
   349	            exit 1
   350	        fi
   351	        if [ -n "${!rename_argi:-}" ]; then
   352	            echo "pair rename: unexpected arg '${!rename_argi}'" >&2
   353	            exit 1
   354	        fi
   355	        # Accept either bare tag or `pair-<tag>`; strip + charset-validate via the
   356	        # shared helper (#52 ARCH-DRY). Length stays a rename-specific guard below.
   357	        old_tag="$(normalize_tag "$old_tag" "pair rename: invalid tag")" || exit 1
   358	        new_tag="$(normalize_tag "$new_tag" "pair rename: invalid tag")" || exit 1
   359	        for t in "$old_tag" "$new_tag"; do
   360	            if [ ${#t} -gt 256 ]; then
   361	                echo "pair rename: tag '$t' is too long (max 256)" >&2
   362	                exit 1
   363	            fi
   364	        done
   365	        if [ "$old_tag" = "$new_tag" ]; then
   366	            echo "pair rename: old and new tag are the same ('$old_tag')" >&2
   367	            exit 1
   368	        fi
   369	        rename_dd="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
   370	        if [ ! -d "$rename_dd" ]; then
   371	            echo "pair rename: data dir not found: $rename_dd" >&2
   372	            exit 1
   373	        fi
   374	
   375	        # Zellij session checks. Skipped if zellij isn't on PATH (e.g.,
   376	        # offline housekeeping from a machine that never had it installed)
   377	        # — refusal is for safety, not gatekeeping.
   378	        if command -v zellij >/dev/null 2>&1; then
   379	            rename_sessions=$( { zj list-sessions --short 2>/dev/null || true; } )
   380	            if [ "$rename_skip_old_live" -eq 0 ] \
   381	               && printf '%s\n' "$rename_sessions" | grep -qx "pair-${old_tag}"; then
   382	                echo "pair rename: session 'pair-${old_tag}' is still tracked by zellij." >&2
   383	                echo "             Quit it first (Alt+x), or use the in-session" >&2
   384	                echo "             rename via the Ctrl+Alt+n restart confirm." >&2
   385	                exit 1
   386	            fi
   387	            if printf '%s\n' "$rename_sessions" | grep -qx "pair-${new_tag}"; then
   388	                echo "pair rename: session 'pair-${new_tag}' already exists in zellij." >&2
   389	                exit 1
   390	            fi
   440	            [ -e "$src" ] || continue
   441	            base="${src##*/}"
   442	            # Substitute the tag chunk: the path was constructed by
   443	            # rename_paths_for as "<dir>/<chunk-with-$old_tag>", so the
   444	            # transform is deterministic.
   445	            case "$base" in
   446	                # tag-only families (suffix is "")
   447	                outer-tty-$old_tag|pair-wrap-pid-$old_tag|title-pid-$old_tag|\
   448	                agent-$old_tag|agent-pid-$old_tag|agent-output-$old_tag|\
   449	                agent-picks-$old_tag|layout-mode-$old_tag|queue-$old_tag|\
   450	                quote-$old_tag|image-capture-$old_tag)
   451	                    dst="$rename_dd/${base%-"$old_tag"}-$new_tag"
   452	                    ;;
   453	                image-capture-$old_tag.done)
   454	                    dst="$rename_dd/image-capture-$new_tag.done"
   455	                    ;;
   456	                draft-$old_tag.md)
   457	                    dst="$rename_dd/draft-$new_tag.md"
   458	                    ;;
   459	                log-$old_tag.md)
   460	                    dst="$rename_dd/log-$new_tag.md"
   461	                    ;;
   462	                nvim-pid-$old_tag-draft|nvim-pid-$old_tag-scrollback)
   463	                    dst="$rename_dd/nvim-pid-$new_tag-${base##*-}"
   464	                    ;;
   465	                config-$old_tag-*.json)
   466	                    suf="${base#config-$old_tag-}"
   467	                    dst="$rename_dd/config-$new_tag-$suf"
   468	                    ;;
   469	                pane-$old_tag-*.json)
   470	                    suf="${base#pane-$old_tag-}"
   471	                    dst="$rename_dd/pane-$new_tag-$suf"
   472	                    ;;
   473	                scrollback-$old_tag-*)
   474	                    suf="${base#scrollback-$old_tag-}"
   475	                    dst="$rename_dd/scrollback-$new_tag-$suf"
   476	                    ;;
   477	                draft-$old_tag-*.md)
   478	                    suf="${base#draft-$old_tag-}"
   479	                    dst="$rename_dd/draft-$new_tag-$suf"
   480	                    ;;
   481	                *)
   482	                    echo "pair rename: internal error: unrecognised src '$src'" >&2
   483	                    rm -f "$old_paths_tmp"
   484	                    exit 1
   485	                    ;;
   486	            esac
   487	            if [ -e "$dst" ]; then
   488	                echo "pair rename: destination already exists: $dst" >&2
   489	                rm -f "$old_paths_tmp"
   490	                exit 1
   491	            fi
   492	            rename_plan="${rename_plan}${src}	${dst}
   493	"
   494	            rename_count=$((rename_count + 1))
   495	        done < "$old_paths_tmp"
   496	        rm -f "$old_paths_tmp"
   497	
   498	        if [ "$rename_count" -eq 0 ]; then
   499	            echo "pair rename: no files found for tag '$old_tag' in $rename_dd" >&2
   500	            exit 1

exec
/bin/zsh -lc 'rg -n "PAIR_HOME|BASH_SOURCE|readlink|dirname|realpath" bin/pair' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
549:# Resolve this script's real path portably (macOS lacks `readlink -f`).
550:SOURCE="${BASH_SOURCE[0]}"
552:    DIR="$(cd -P "$(dirname "$SOURCE")" && pwd)"
553:    SOURCE="$(readlink "$SOURCE")"
556:PAIR_HOME="$(cd -P "$(dirname "$SOURCE")/.." && pwd)"
557:export PAIR_HOME
563:    *":$PAIR_HOME/bin:"*) ;;
564:    *) export PATH="$PAIR_HOME/bin:$PATH" ;;
1671:    "$PAIR_HOME/bin/pair-title.sh" "$PAIR_TAG" "$title_agent" \
1695:    zellij --config-dir "$PAIR_HOME/zellij" attach "$SESSION"
2227:"$PAIR_HOME/bin/pair-session-watch.sh" "$AGENT" "$PAIR_TAG" "$PWD" $agent_extra \
2245:. "$PAIR_HOME/bin/lib/dev-rebuild.sh"
2275:    --config-dir              "$PAIR_HOME/zellij" \
2276:    --new-session-with-layout "$PAIR_HOME/zellij/layouts/main.kdl" \

exec
/bin/zsh -lc 'git status --short' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 M workshop/issues/000077-go-entrypoint-switch.md
?? workshop/issues/000089-review-mode-should-disable-edit-while-agent-update-the-doc.md

codex
The symlink install approach is compatible with `bin/pair`: the script resolves `BASH_SOURCE[0]` through symlinks before setting `PAIR_HOME`, so an installed symlink back to the source tree keeps asset lookup anchored in the repo. I have enough to produce the boundary verdict.
exec
/bin/zsh -lc "nl -ba bin/pair | sed -n '545,566p' && git show HEAD:Makefile.local | nl -ba | sed -n '27,43p' && git show HEAD:tests/pair-go-install-layout-test.sh | nl -ba" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
   545	        exit 0
   546	        ;;
   547	esac
   548	
   549	# Resolve this script's real path portably (macOS lacks `readlink -f`).
   550	SOURCE="${BASH_SOURCE[0]}"
   551	while [ -L "$SOURCE" ]; do
   552	    DIR="$(cd -P "$(dirname "$SOURCE")" && pwd)"
   553	    SOURCE="$(readlink "$SOURCE")"
   554	    [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
   555	done
   556	PAIR_HOME="$(cd -P "$(dirname "$SOURCE")/.." && pwd)"
   557	export PAIR_HOME
   558	
   559	# Put pair's bin/ on PATH so zellij can resolve our helper scripts by bare
   560	# name (clipboard-to-pane.sh, copy-on-select.sh) without sh-c env-var hacks
   561	# in zellij/config.kdl. Avoid double-prepending across re-launches.
   562	case ":$PATH:" in
   563	    *":$PAIR_HOME/bin:"*) ;;
   564	    *) export PATH="$PAIR_HOME/bin:$PATH" ;;
   565	esac
   566	
    27	BIN_DIR     := bin
    28	INSTALL_DIR := $(HOME)/.local/bin
    29	GO_BINS     := pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-go
    30	SHELL_BINS  := pair pair-dev
    31	
    32	build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
    33	
    34	install: build
    35		@mkdir -p $(INSTALL_DIR)
    36		@for b in $(GO_BINS); do \
    37		    install -m 755 $(BIN_DIR)/$$b $(INSTALL_DIR)/$$b && \
    38		    echo "installed: $(INSTALL_DIR)/$$b"; \
    39		done
    40		@for b in $(SHELL_BINS); do \
    41		    ln -sf "$(CURDIR)/$(BIN_DIR)/$$b" "$(INSTALL_DIR)/$$b" && \
    42		    echo "linked: $(INSTALL_DIR)/$$b -> $(CURDIR)/$(BIN_DIR)/$$b"; \
    43		done
     1	#!/usr/bin/env bash
     2	set -euo pipefail
     3	
     4	repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
     5	tmp_home="$(mktemp -d "${TMPDIR:-/tmp}/pair-go-install-layout.XXXXXX")"
     6	trap 'rm -rf "$tmp_home"' EXIT
     7	gomodcache="$(go env GOMODCACHE)"
     8	gocache="$(go env GOCACHE)"
     9	
    10	HOME="$tmp_home" GOMODCACHE="$gomodcache" GOCACHE="$gocache" make -C "$repo_root" install >/dev/null
    11	
    12	install_bin="$tmp_home/.local/bin"
    13	test -x "$install_bin/pair-go"
    14	test -L "$install_bin/pair"
    15	test -L "$install_bin/pair-dev"
    16	
    17	out="$("$install_bin/pair-go" launch --help)"
    18	case "$out" in
    19	    pair\ —*) ;;
    20	    *)
    21	        printf 'pair-go launch --help did not reach pair help; first bytes:\n%s\n' "$out" >&2
    22	        exit 1
    23	        ;;
    24	esac
    25	
    26	printf 'pair-go install layout test passed\n'

codex
VERDICT: SHIP (confidence: high)

The prior blockers are fixed: installed `pair-go` now has sibling `pair`/`pair-dev` links, the install-layout regression passes, and the durable plan’s Core concepts table now matches the implemented `legacyRuntime` seam. The launch handoff remains small, testable, and aligned with the issue’s scoped purpose.

1. Strengths:
- [Makefile.local:40](/Users/xianxu/workspace/pair/Makefile.local:40) links `pair` and `pair-dev` beside installed Go binaries, fixing the installed handoff layout.
- [tests/pair-go-install-layout-test.sh:10](/Users/xianxu/workspace/pair/tests/pair-go-install-layout-test.sh:10) covers the temp `make install` path and verifies installed `pair-go launch --help` reaches `pair` help.
- [bin/pair:549](/Users/xianxu/workspace/pair/bin/pair:549) resolves symlinks before setting `PAIR_HOME`, so the new installed symlink preserves source-tree asset resolution.
- [workshop/plans/000077-go-entrypoint-switch-plan.md:38](/Users/xianxu/workspace/pair/workshop/plans/000077-go-entrypoint-switch-plan.md:38) now matches the implemented `legacyRuntime` entity.

2. Critical findings:
- None.

3. Important findings:
- None.

4. Minor findings:
- `git diff --check b829ef..HEAD` fails on trailing whitespace embedded in the generated `workshop/plans/000077-go-entrypoint-switch-close-review.md` transcript. This is not runtime-impacting, but it will annoy any whitespace gate that scans generated review artifacts.

5. Test coverage notes:
- Passed: `go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1`
- Passed: `go test ./... -count=1`
- Passed: `make test-dev-rebuild`
- Passed: `make test-pair-go-install-layout`
- Passed: `bin/pair-go launch --help`

6. Architectural notes for upcoming work:
- ARCH-DRY: pass, lifecycle behavior remains single-sourced in `bin/pair`.
- ARCH-PURE: pass, pure argv/path resolution is separate from the exec boundary.
- ARCH-PURPOSE: pass, the documented migration entrypoint works in repo and installed layouts without replacing stable `pair`.

7. Plan revision recommendations:
- None.
tokens used
125,618
VERDICT: SHIP (confidence: high)

The prior blockers are fixed: installed `pair-go` now has sibling `pair`/`pair-dev` links, the install-layout regression passes, and the durable plan’s Core concepts table now matches the implemented `legacyRuntime` seam. The launch handoff remains small, testable, and aligned with the issue’s scoped purpose.

1. Strengths:
- [Makefile.local:40](/Users/xianxu/workspace/pair/Makefile.local:40) links `pair` and `pair-dev` beside installed Go binaries, fixing the installed handoff layout.
- [tests/pair-go-install-layout-test.sh:10](/Users/xianxu/workspace/pair/tests/pair-go-install-layout-test.sh:10) covers the temp `make install` path and verifies installed `pair-go launch --help` reaches `pair` help.
- [bin/pair:549](/Users/xianxu/workspace/pair/bin/pair:549) resolves symlinks before setting `PAIR_HOME`, so the new installed symlink preserves source-tree asset resolution.
- [workshop/plans/000077-go-entrypoint-switch-plan.md:38](/Users/xianxu/workspace/pair/workshop/plans/000077-go-entrypoint-switch-plan.md:38) now matches the implemented `legacyRuntime` entity.

2. Critical findings:
- None.

3. Important findings:
- None.

4. Minor findings:
- `git diff --check b829ef..HEAD` fails on trailing whitespace embedded in the generated `workshop/plans/000077-go-entrypoint-switch-close-review.md` transcript. This is not runtime-impacting, but it will annoy any whitespace gate that scans generated review artifacts.

5. Test coverage notes:
- Passed: `go test ./cmd/internal/entrypoint ./cmd/pair-go ./cmd/internal/dispatcher -count=1`
- Passed: `go test ./... -count=1`
- Passed: `make test-dev-rebuild`
- Passed: `make test-pair-go-install-layout`
- Passed: `bin/pair-go launch --help`

6. Architectural notes for upcoming work:
- ARCH-DRY: pass, lifecycle behavior remains single-sourced in `bin/pair`.
- ARCH-PURE: pass, pure argv/path resolution is separate from the exec boundary.
- ARCH-PURPOSE: pass, the documented migration entrypoint works in repo and installed layouts without replacing stable `pair`.

7. Plan revision recommendations:
- None.
