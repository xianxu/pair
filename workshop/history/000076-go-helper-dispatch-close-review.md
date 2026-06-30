# Boundary Review — pair#76 (whole-issue close)

| field | value |
|-------|-------|
| issue | 76 — pair Go helper dispatch |
| repo | pair |
| issue file | workshop/issues/000076-go-helper-dispatch.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9cc0f57521c3e44508adf3df006ffe3347faa265..HEAD |
| command | sdlc close --issue 76 |
| reviewer | codex |
| timestamp | 2026-06-30T12:24:12-07:00 |
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
session id: 019f19fb-a9eb-7f52-b279-f54b7aae5504
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **pair** repository.

- repository: pair   (root: /Users/xianxu/workspace/pair)
- issue:      pair#76   (file: workshop/issues/000076-go-helper-dispatch.md)
- window:     Base: 9cc0f57521c3e44508adf3df006ffe3347faa265   Head: HEAD

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
index 0618d25..ccdde68 100644
--- a/Makefile.local
+++ b/Makefile.local
@@ -210,7 +210,7 @@ $(BIN_DIR)/pair-scribe: cmd/pair-scribe/main.go go.mod
 # pair-scrollback-render — replays a pair-wrap raw PTY capture through
 # charmbracelet/x/vt and emits one ANSI-styled line per logical scrollback
 # row, for the Alt+/ viewer to display through nvim.
-$(BIN_DIR)/pair-scrollback-render: cmd/pair-scrollback-render/main.go go.mod
+$(BIN_DIR)/pair-scrollback-render: cmd/pair-scrollback-render/main.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
 	go build -o $@ ./cmd/pair-scrollback-render
 
 # pair-wrap — transparent PTY proxy around a TUI coding agent: forwards
@@ -225,7 +225,7 @@ $(BIN_DIR)/pair-slug: cmd/pair-slug/main.go cmd/pair-slug/slug.go cmd/internal/m
 
 # pair-context — one-shot: print a pane's current context size (token count)
 # for the pair-title poller. Pure ctxmeter reader + shared transcript resolver.
-$(BIN_DIR)/pair-context: cmd/pair-context/main.go cmd/internal/ctxmeter/ctxmeter.go cmd/internal/transcript/transcript.go go.mod
+$(BIN_DIR)/pair-context: cmd/pair-context/main.go cmd/internal/contextcmd/contextcmd.go cmd/internal/ctxmeter/ctxmeter.go cmd/internal/transcript/transcript.go go.mod
 	go build -o $@ ./cmd/pair-context
 
 # pair-changelog — distills the pair TTY into the Alt+l change log (#53);
@@ -241,7 +241,7 @@ $(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-con
 
 # pair-go — development-only dispatcher skeleton for the future primary Go CLI.
 # The public launcher remains bin/pair until the entrypoint-switch issue.
-$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/dispatcher/dispatcher.go go.mod
+$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
 	go build -o $@ ./cmd/pair-go
 
 
diff --git a/atlas/architecture.md b/atlas/architecture.md
index a536fc7..5e308ab 100644
--- a/atlas/architecture.md
+++ b/atlas/architecture.md
@@ -55,6 +55,14 @@ prompt/fzf UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation,
 rename, config/session migration, and title-poller behavior remain shell-owned
 until later migration issues.
 
+As of #76, the same dispatcher also has the first implemented helper routes:
+`pair-go context <tag> <agent>` and `pair-go scrollback-render ...`. Both routes
+call shared internal Go runners used by the legacy `pair-context` and
+`pair-scrollback-render` binaries (`ARCH-DRY`), so behavior stays aligned while
+the old command names remain the live integration surface. `bin/pair-title.sh`,
+`bin/pair-scrollback-open`, `bin/pair-changelog-open`, and
+`nvim/scrollback.lua` have not moved to the dispatcher yet.
+
 Native integration layers stay native: `nvim/*.lua` remains the bundled Neovim
 surface and `zellij/*.kdl` remains the zellij layout/config surface. Packaging
 may embed those assets or install them adjacent to the binary, but the migration
diff --git a/atlas/go-migration-inventory.md b/atlas/go-migration-inventory.md
index 3de3bb2..8b404f0 100644
--- a/atlas/go-migration-inventory.md
+++ b/atlas/go-migration-inventory.md
@@ -54,8 +54,8 @@ Priority is packaging impact first, then reliability/testability:
 | `zellij/config.kdl` | zellij native asset | zellij session config from `bin/pair` | Global keybinds, copy command, scrollback buffer, pane frames. | Calls `copy-on-select.sh`, `pair-help`, `pair-scrollback-open`, `pair-changelog-open`; routes quit/restart/compact through nvim functions. | native-asset, packaged adjacent/embedded | P0 |
 | `bin/pair-wrap` / `cmd/pair-wrap` | Go binary | zellij agent pane | `pair-wrap [--scrollback-log PATH] agent [args...]`; transparent PTY proxy; long-running; failure in diagnostics is swallowed. | Reads Pair env and agent command; writes `agent-output-<tag>`, `agent-pid-<tag>`, scrollback `.raw`/`.events.jsonl`, image capture files; may invoke `pair-slug`. | go-subcommand `pair wrap`; keep old binary as compat alias for KDL until caller moves | P0 |
 | `bin/pair-slug` / `cmd/pair-slug` | Go binary | `pair-wrap` turn-end hook, tests | Env-driven, no stdin; resolves native transcript, proposes slug; exits 0 on most failures. | Requires `PAIR_TAG`, `PAIR_DATA_DIR`; reads config/transcripts/git branch; writes `slug-proposed-<tag>`; optional `PAIR_SLUG_*`, `OPENAI_API_KEY`. | go-subcommand `pair slug`; legacy binary retained during #76 | P1 |
-| `bin/pair-context` / `cmd/pair-context` | Go binary | `bin/pair-title.sh` | `pair-context <tag> <agent>`; prints humanized token count or nothing; tolerant exit 0 on failure. | Reads `PAIR_DATA_DIR`, `pane-<tag>-<agent>.json`, config, native transcripts. | go-subcommand `pair context`; legacy binary retained while title poller calls it | P1 |
-| `bin/pair-scrollback-render` / `cmd/pair-scrollback-render` | Go binary | `bin/pair-scrollback-open`, `bin/pair-changelog-open`, `nvim/scrollback.lua` refresh | `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out`; nonzero on render/write failure. | Reads `.raw` and `.events.jsonl`; atomically writes `.ansi` or cleaned text. | go-subcommand `pair scrollback-render`; legacy binary retained for shell/Lua callers | P0 |
+| `bin/pair-context` / `cmd/pair-context` / `cmd/internal/contextcmd` | Go binary plus shared runner | `bin/pair-title.sh`; development-only `pair-go context` | `pair-context <tag> <agent>` and `pair-go context <tag> <agent>` print the same humanized token count or nothing; tolerant exit 0 on failure. | Reads `PAIR_DATA_DIR`, `pane-<tag>-<agent>.json`, config, native transcripts. | implemented helper route in `pair-go context`; legacy binary retained while title poller calls it | P1 |
+| `bin/pair-scrollback-render` / `cmd/pair-scrollback-render` / `cmd/internal/scrollbackcmd` | Go binary plus shared runner | `bin/pair-scrollback-open`, `bin/pair-changelog-open`, `nvim/scrollback.lua` refresh; development-only `pair-go scrollback-render` | `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out` and `pair-go scrollback-render ...`; nonzero on render/write failure. | Reads `.raw` and `.events.jsonl`; atomically writes `.ansi` or cleaned text. | implemented helper route in `pair-go scrollback-render`; legacy binary retained for shell/Lua callers | P0 |
 | `bin/pair-changelog` / `cmd/pair-changelog` | Go binary | `bin/pair-changelog-open` | `pair-changelog --cleaned F --log F --anchor F [--agent A] [--model M]`; exits nonzero on required read/model/write failure. | Reads cleaned scrollback/log/anchor; calls agent model through internal model runner; atomically writes log and anchor. | go-subcommand `pair changelog`; legacy binary retained for opener | P1 |
 | `bin/pair-continuation` / `cmd/pair-continuation` | Go binary | nvim compaction prompt instructions, operator/agent shell | `pair-continuation --slug S --agent A --issues CSV --body-file F [--repo-root R ...]`; writes and commits continuation; nonzero on validation/git failure. | Reads body/stdin, git repo state; writes `workshop/continuation/*.md`; runs git commit/push. | go-subcommand `pair continuation`; legacy binary retained for agent instructions until docs change | P1 |
 | `bin/pair-scribe` / `cmd/pair-scribe` | Go binary | user shell rc outside Pair sessions | `pair-scribe -log PATH -- CMD [ARGS...]`; long-running PTY wrapper; SIGUSR1 pauses log, SIGUSR2 resumes. | Writes typescript log; wraps child PTY; independent of `PAIR_*`. | go-subcommand candidate is low value; may remain separate installed helper or become `pair scribe` with alias | P2 |
@@ -125,9 +125,10 @@ Build/install callers:
 - #75 and #76 can proceed in parallel after #74. The launcher prototype does not
   need helper dispatch to exist first, and helper dispatch does not need the
   launcher prototype.
-- #76 should start with existing Go helpers whose contracts are already clean:
-  `pair context`, `pair slug`, and `pair scrollback-render` are good first
-  candidates. Keep legacy binary names working.
+- #76 established the first helper-dispatch pattern with `pair-go context` and
+  `pair-go scrollback-render`, backed by shared internal runners while legacy
+  binary names remain live for shell/Lua callers. `pair slug` remains a later
+  candidate.
 - #77 is the public entrypoint switch and should wait for both #75 and #76.
 - #78 should prioritize `pair-title.sh` and `pair-session-watch.sh` if stateful
   shell remains a packaging/reliability problem after #77.
diff --git a/cmd/internal/contextcmd/contextcmd.go b/cmd/internal/contextcmd/contextcmd.go
new file mode 100644
index 0000000..ca6debb
--- /dev/null
+++ b/cmd/internal/contextcmd/contextcmd.go
@@ -0,0 +1,78 @@
+// Package contextcmd implements the pair-context command body.
+package contextcmd
+
+import (
+	"encoding/json"
+	"fmt"
+	"io"
+	"os"
+	"path/filepath"
+
+	"github.com/xianxu/pair/cmd/internal/ctxmeter"
+	"github.com/xianxu/pair/cmd/internal/transcript"
+)
+
+type Env struct {
+	Home        string
+	XDGDataHome string
+	PairDataDir string
+}
+
+func EnvFromOS() Env {
+	return Env{
+		Home:        os.Getenv("HOME"),
+		XDGDataHome: os.Getenv("XDG_DATA_HOME"),
+		PairDataDir: os.Getenv("PAIR_DATA_DIR"),
+	}
+}
+
+func Run(args []string, env Env, stdout io.Writer) int {
+	if len(args) < 2 {
+		return 0
+	}
+	tag, agent := args[0], args[1]
+	dataDir := resolveDataDir(env)
+	sid := transcript.SessionID(dataDir, tag, agent)
+	if sid == "" {
+		return 0
+	}
+	cwd := paneCwd(dataDir, tag, agent)
+	path := transcript.Resolve(agent, sid, cwd, env.Home)
+	if path == "" {
+		return 0
+	}
+	f, err := os.Open(path)
+	if err != nil {
+		return 0
+	}
+	defer f.Close()
+	if n, ok := ctxmeter.ContextTokens(agent, f); ok {
+		fmt.Fprintln(stdout, ctxmeter.Humanize(n))
+	}
+	return 0
+}
+
+func resolveDataDir(env Env) string {
+	if env.PairDataDir != "" {
+		return env.PairDataDir
+	}
+	base := env.XDGDataHome
+	if base == "" {
+		base = filepath.Join(env.Home, ".local", "share")
+	}
+	return filepath.Join(base, "pair")
+}
+
+func paneCwd(dataDir, tag, agent string) string {
+	b, err := os.ReadFile(filepath.Join(dataDir, "pane-"+tag+"-"+agent+".json"))
+	if err != nil {
+		return ""
+	}
+	var p struct {
+		Cwd string `json:"cwd"`
+	}
+	if json.Unmarshal(b, &p) != nil {
+		return ""
+	}
+	return p.Cwd
+}
diff --git a/cmd/internal/contextcmd/contextcmd_test.go b/cmd/internal/contextcmd/contextcmd_test.go
new file mode 100644
index 0000000..2e07506
--- /dev/null
+++ b/cmd/internal/contextcmd/contextcmd_test.go
@@ -0,0 +1,59 @@
+package contextcmd
+
+import (
+	"bytes"
+	"os"
+	"path/filepath"
+	"strings"
+	"testing"
+)
+
+func TestRunClaude(t *testing.T) {
+	home := t.TempDir()
+	data := filepath.Join(home, "data")
+	cwd := filepath.Join(home, "repo")
+	enc := strings.NewReplacer(".", "-", "/", "-").Replace(cwd)
+	proj := filepath.Join(home, ".claude", "projects", enc)
+	mustMkdir(t, data)
+	mustMkdir(t, cwd)
+	mustMkdir(t, proj)
+	mustWrite(t, filepath.Join(data, "config-T-claude.json"), `{"session_id":"sid1"}`)
+	mustWrite(t, filepath.Join(data, "pane-T-claude.json"), `{"pane_id":"7","cwd":"`+cwd+`","cwd_display":"~/repo"}`)
+	mustWrite(t, filepath.Join(proj, "sid1.jsonl"),
+		`{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":397556,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`)
+
+	var stdout bytes.Buffer
+	code := Run([]string{"T", "claude"}, Env{Home: home, PairDataDir: data}, &stdout)
+	if code != 0 {
+		t.Fatalf("code = %d, want 0", code)
+	}
+	if got := strings.TrimSpace(stdout.String()); got != "398k" {
+		t.Fatalf("stdout = %q, want 398k", stdout.String())
+	}
+}
+
+func TestRunMissingConfigPrintsNothing(t *testing.T) {
+	home := t.TempDir()
+	var stdout bytes.Buffer
+	code := Run([]string{"T", "claude"}, Env{Home: home, PairDataDir: filepath.Join(home, "empty")}, &stdout)
+	if code != 0 {
+		t.Fatalf("code = %d, want 0", code)
+	}
+	if stdout.String() != "" {
+		t.Fatalf("stdout = %q, want empty", stdout.String())
+	}
+}
+
+func mustMkdir(t *testing.T, d string) {
+	t.Helper()
+	if err := os.MkdirAll(d, 0o755); err != nil {
+		t.Fatal(err)
+	}
+}
+
+func mustWrite(t *testing.T, p, s string) {
+	t.Helper()
+	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
+		t.Fatal(err)
+	}
+}
diff --git a/cmd/internal/dispatcher/dispatcher.go b/cmd/internal/dispatcher/dispatcher.go
index 96d6b3b..aba2165 100644
--- a/cmd/internal/dispatcher/dispatcher.go
+++ b/cmd/internal/dispatcher/dispatcher.go
@@ -1,12 +1,15 @@
 package dispatcher
 
 import (
+	"bytes"
 	"fmt"
 	"os"
 	"strings"
 	"time"
 
+	"github.com/xianxu/pair/cmd/internal/contextcmd"
 	"github.com/xianxu/pair/cmd/internal/launcher"
+	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
 )
 
 const programName = "pair-go"
@@ -29,10 +32,10 @@ type Result struct {
 func Families() []CommandFamily {
 	return []CommandFamily{
 		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "prototype"},
+		{Name: "context", Summary: "agent pane context meter", Status: "implemented"},
+		{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "implemented"},
 		{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
 		{Name: "slug", Summary: "session orientation slug generation", Status: "planned"},
-		{Name: "context", Summary: "agent pane context meter", Status: "planned"},
-		{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "planned"},
 		{Name: "changelog", Summary: "TTY transcript to distilled change log", Status: "planned"},
 		{Name: "continuation", Summary: "continuation datatype writer", Status: "planned"},
 		{Name: "scribe", Summary: "PTY logging wrapper", Status: "planned"},
@@ -55,6 +58,10 @@ func Dispatch(args []string) Result {
 		}
 	case "launch":
 		return DispatchWithLauncherRuntime(args, osLauncherRuntime())
+	case "context":
+		return dispatchContext(args[1:])
+	case "scrollback-render":
+		return dispatchScrollbackRender(args[1:])
 	}
 
 	if family, ok := familyByName(args[0]); ok {
@@ -70,6 +77,18 @@ func Dispatch(args []string) Result {
 	}
 }
 
+func dispatchContext(args []string) Result {
+	var stdout bytes.Buffer
+	code := contextcmd.Run(args, contextcmd.EnvFromOS(), &stdout)
+	return Result{Stdout: stdout.String(), ExitCode: code}
+}
+
+func dispatchScrollbackRender(args []string) Result {
+	var stdout, stderr bytes.Buffer
+	code := scrollbackcmd.Run(args, &stdout, &stderr)
+	return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}
+}
+
 type LauncherRuntime struct {
 	Env      launcher.Env
 	Sessions launcher.SessionSource
@@ -166,11 +185,13 @@ func Help(program string) string {
 	for _, family := range Families() {
 		if family.Status == "prototype" {
 			fmt.Fprintf(&b, "  %-17s %s (prototype; decision-phase only)\n", family.Name, family.Summary)
+		} else if family.Status == "implemented" {
+			fmt.Fprintf(&b, "  %-17s %s (implemented helper route)\n", family.Name, family.Summary)
 		}
 	}
 	b.WriteString("\nPlanned command families (not implemented in this skeleton):\n")
 	for _, family := range Families() {
-		if family.Status != "prototype" {
+		if family.Status == "planned" {
 			fmt.Fprintf(&b, "  %-17s %s (%s; not implemented in this skeleton)\n", family.Name, family.Summary, family.Status)
 		}
 	}
diff --git a/cmd/internal/dispatcher/dispatcher_test.go b/cmd/internal/dispatcher/dispatcher_test.go
index af3a573..d4302b6 100644
--- a/cmd/internal/dispatcher/dispatcher_test.go
+++ b/cmd/internal/dispatcher/dispatcher_test.go
@@ -1,6 +1,8 @@
 package dispatcher
 
 import (
+	"os"
+	"path/filepath"
 	"strings"
 	"testing"
 )
@@ -20,6 +22,8 @@ func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
 				"Implemented prototype commands:",
 				"launch",
 				"decision-phase only",
+				"context",
+				"scrollback-render",
 				"wrap",
 				"slug",
 				"not implemented in this skeleton",
@@ -31,6 +35,14 @@ func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
 			if strings.Contains(res.Stdout, "launch             session lifecycle and public pair launcher flow (planned; not implemented") {
 				t.Fatalf("Stdout still labels launch unimplemented:\n%s", res.Stdout)
 			}
+			for _, stale := range []string{
+				"context           agent pane context meter (planned; not implemented",
+				"scrollback-render raw PTY capture to ANSI scrollback (planned; not implemented",
+			} {
+				if strings.Contains(res.Stdout, stale) {
+					t.Fatalf("Stdout still labels helper unimplemented (%q):\n%s", stale, res.Stdout)
+				}
+			}
 		})
 	}
 }
@@ -122,6 +134,36 @@ func TestDispatchLaunchWithoutArgsReturnsDefaultPrototypeDecision(t *testing.T)
 	}
 }
 
+func TestDispatchContextReturnsHelperOutput(t *testing.T) {
+	home, data := writeContextFixture(t)
+	t.Setenv("HOME", home)
+	t.Setenv("PAIR_DATA_DIR", data)
+
+	res := Dispatch([]string{"context", "T", "claude"})
+	if res.ExitCode != 0 {
+		t.Fatalf("ExitCode = %d, want 0; stderr:\n%s", res.ExitCode, res.Stderr)
+	}
+	if res.Stderr != "" {
+		t.Fatalf("Stderr = %q, want empty", res.Stderr)
+	}
+	if strings.TrimSpace(res.Stdout) != "398k" {
+		t.Fatalf("Stdout = %q, want 398k", res.Stdout)
+	}
+}
+
+func TestDispatchScrollbackRenderUsage(t *testing.T) {
+	res := Dispatch([]string{"scrollback-render"})
+	if res.ExitCode != 2 {
+		t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
+	}
+	if res.Stdout != "" {
+		t.Fatalf("Stdout = %q, want empty", res.Stdout)
+	}
+	if !strings.Contains(res.Stderr, "usage: pair-scrollback-render") {
+		t.Fatalf("Stderr missing usage:\n%s", res.Stderr)
+	}
+}
+
 func TestDispatchUnknownCommandReturnsUsageHint(t *testing.T) {
 	res := Dispatch([]string{"frobnicate"})
 	if res.ExitCode != 2 {
@@ -136,3 +178,34 @@ func TestDispatchUnknownCommandReturnsUsageHint(t *testing.T) {
 		}
 	}
 }
+
+func writeContextFixture(t *testing.T) (home, data string) {
+	t.Helper()
+	home = t.TempDir()
+	data = filepath.Join(home, "data")
+	cwd := filepath.Join(home, "repo")
+	enc := strings.NewReplacer(".", "-", "/", "-").Replace(cwd)
+	proj := filepath.Join(home, ".claude", "projects", enc)
+	mustMkdir(t, data)
+	mustMkdir(t, cwd)
+	mustMkdir(t, proj)
+	mustWrite(t, filepath.Join(data, "config-T-claude.json"), `{"session_id":"sid1"}`)
+	mustWrite(t, filepath.Join(data, "pane-T-claude.json"), `{"pane_id":"7","cwd":"`+cwd+`","cwd_display":"~/repo"}`)
+	mustWrite(t, filepath.Join(proj, "sid1.jsonl"),
+		`{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":397556,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`)
+	return home, data
+}
+
+func mustMkdir(t *testing.T, d string) {
+	t.Helper()
+	if err := os.MkdirAll(d, 0o755); err != nil {
+		t.Fatal(err)
+	}
+}
+
+func mustWrite(t *testing.T, p, s string) {
+	t.Helper()
+	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
+		t.Fatal(err)
+	}
+}
diff --git a/cmd/pair-scrollback-render/events_test.go b/cmd/internal/scrollbackcmd/events_test.go
similarity index 99%
rename from cmd/pair-scrollback-render/events_test.go
rename to cmd/internal/scrollbackcmd/events_test.go
index beec3ee..c932036 100644
--- a/cmd/pair-scrollback-render/events_test.go
+++ b/cmd/internal/scrollbackcmd/events_test.go
@@ -1,4 +1,4 @@
-package main
+package scrollbackcmd
 
 import (
 	"os"
diff --git a/cmd/pair-scrollback-render/render_test.go b/cmd/internal/scrollbackcmd/render_test.go
similarity index 99%
rename from cmd/pair-scrollback-render/render_test.go
rename to cmd/internal/scrollbackcmd/render_test.go
index b49fcf7..32153a0 100644
--- a/cmd/pair-scrollback-render/render_test.go
+++ b/cmd/internal/scrollbackcmd/render_test.go
@@ -1,4 +1,4 @@
-package main
+package scrollbackcmd
 
 import (
 	"fmt"
diff --git a/cmd/internal/scrollbackcmd/scrollbackcmd.go b/cmd/internal/scrollbackcmd/scrollbackcmd.go
new file mode 100644
index 0000000..9525323
--- /dev/null
+++ b/cmd/internal/scrollbackcmd/scrollbackcmd.go
@@ -0,0 +1,413 @@
+// scrollback-render — replay a pair-wrap raw capture through a VT100
+// emulator and write one ANSI-styled output line per logical scrollback row.
+// Drop-in replacement for the Python+pyte renderer at bin/pair-scrollback-render.
+//
+// Why Go: pyte's HistoryScreen dispatches every method call through an
+// __getattribute__ override that ran ~19M times for a 3 MB raw input —
+// ~95% of wall time. Even after the CaptureScreen patch (3.6x speedup),
+// the Python interpreter + pyte vendoring add startup cost and a private
+// venv that the brew formula has to manage. A static Go binary using
+// charmbracelet/x/vt replays the same stream with no runtime deps and
+// stays within the pair repo's existing cmd/ layout.
+//
+// Pipeline:
+//
+//	raw bytes (.raw)              → emulator.Write(...) in segments
+//	resize events (.events.jsonl) → segment boundaries with new (cols,rows)
+//	final emulator state          → scrollback lines + visible buffer
+//	each row                      → SGR-decorated text line written to out
+//
+// CLI is identical to the Python version so bin/pair-scrollback-open can
+// invoke either:
+//
+//	scrollback-render <raw> <events.jsonl> <out.ansi>
+package scrollbackcmd
+
+import (
+	"encoding/json"
+	"flag"
+	"fmt"
+	"io"
+	"math"
+	"os"
+	"strconv"
+	"strings"
+	"time"
+
+	uv "github.com/charmbracelet/ultraviolet"
+	"github.com/charmbracelet/x/vt"
+)
+
+// historyRows caps scrolled-out rows retained for the viewer. Matched
+// to zellij/config.kdl's `scroll_buffer_size 2000` so PageUp inside the
+// agent pane and Alt+/ both reach back the same distance — otherwise
+// the viewer would surface lines that zellij no longer has, leaving
+// the scroll-overlay logic in pair-scrollback-open unable to align
+// against the agent pane's frame.
+const historyRows = 2_000
+
+const (
+	defaultCols = 80
+	defaultRows = 24
+)
+
+type scrollbackEvent struct {
+	Type   string `json:"type"`
+	Offset int64  `json:"offset"`
+	Cols   int    `json:"cols"`
+	Rows   int    `json:"rows"`
+	Ts     string `json:"ts,omitempty"` // RFC3339 wall-clock for "time" events (#59)
+}
+
+// dateOf extracts the YYYY-MM-DD day from an RFC3339 timestamp; "" on a
+// malformed value so a corrupt time event degrades to undated, never panics (#59).
+func dateOf(ts string) string {
+	t, err := time.Parse(time.RFC3339, ts)
+	if err != nil {
+		return ""
+	}
+	return t.Format("2006-01-02")
+}
+
+// parseEvents reads the sidecar JSONL. Empty / missing file → empty slice.
+// Malformed lines are skipped so a corrupted tail doesn't abort the render —
+// imperfect width tracking beats an unusable viewer.
+func parseEvents(path string) ([]scrollbackEvent, error) {
+	data, err := os.ReadFile(path)
+	if err != nil {
+		if os.IsNotExist(err) {
+			return nil, nil
+		}
+		return nil, err
+	}
+	var out []scrollbackEvent
+	for _, line := range strings.Split(string(data), "\n") {
+		line = strings.TrimSpace(line)
+		if line == "" {
+			continue
+		}
+		var e scrollbackEvent
+		if err := json.Unmarshal([]byte(line), &e); err != nil {
+			continue
+		}
+		// Keep both known types: resize boundaries AND time stamps (#59).
+		// Consumers filter by Type at their use sites.
+		if e.Type == "resize" || e.Type == "time" {
+			out = append(out, e)
+		}
+	}
+	return out, nil
+}
+
+// initialSize pulls (cols, rows) from the first usable resize event, or
+// falls back to 80x24 if the sidecar is empty or the first entry is
+// malformed. pair-wrap always emits an initial resize at offset 0, so the
+// fallback only fires on a truncated file.
+func initialSize(events []scrollbackEvent) (int, int) {
+	for _, e := range events {
+		if e.Type == "resize" && e.Cols > 0 && e.Rows > 0 {
+			return e.Cols, e.Rows
+		}
+	}
+	return defaultCols, defaultRows
+}
+
+// dateMark records the emulator's scrollback length at a "time" event's byte
+// offset → the day that applies to committed lines from that index onward (#59).
+// Built during the feed (the only place that knows both byte offsets and the
+// rendered line count); consumed by the pure interleaveDateMarkers.
+type dateMark struct {
+	line int
+	date string
+}
+
+// feedSegments writes raw into the emulator as a single offset-ordered walk over
+// ALL sidecar events: write everything up to event.Offset, then act — Resize on a
+// resize event, or snapshot Scrollback().Len() on a time event. Returns the time
+// snapshots (empty unless time events are present). The caller already set the
+// initial size via initialSize; re-applying the offset-0 resize here is a harmless
+// no-op (resize to the current dimensions). Walking all events — rather than
+// events[1:] — means a time event in any position (incl. first) is captured, and
+// an empty events slice is handled without an out-of-range slice (#59).
+//
+// Clamping Offset to len(raw) defends against a corrupted sidecar that records
+// an offset beyond EOF (saw this once with a half-written events file after a
+// hard kill); without clamping we'd panic on the slice.
+func feedSegments(em *vt.Emulator, raw []byte, events []scrollbackEvent) []dateMark {
+	var cursor int64
+	var marks []dateMark
+	for _, e := range events {
+		off := e.Offset
+		if off > int64(len(raw)) {
+			off = int64(len(raw))
+		}
+		if off > cursor {
+			_, _ = em.Write(raw[cursor:off])
+			cursor = off
+		}
+		switch e.Type {
+		case "resize":
+			em.Resize(e.Cols, e.Rows)
+		case "time":
+			if d := dateOf(e.Ts); d != "" {
+				marks = append(marks, dateMark{line: em.Scrollback().Len(), date: d})
+			}
+		}
+	}
+	if cursor < int64(len(raw)) {
+		_, _ = em.Write(raw[cursor:])
+	}
+	return marks
+}
+
+// tsMarkerLine is the wire format the distiller parses (#59). MUST stay in sync
+// with tsMarkerRe in cmd/pair-changelog/distill.go — the contract is pinned by
+// the render→clean→distill e2e test cmd/pair-changelog/e2e_test.go
+// (TestEndToEndMarkerSurvival), which feeds real time events through both binaries.
+func tsMarkerLine(date string) string {
+	return "⟦pair:ts " + date + "⟧"
+}
+
+// interleaveDateMarkers inserts a tsMarkerLine immediately before the first line
+// of each new date run. marks are (scrollback-line-index, date) snapshots in
+// ascending index; a marker is emitted only when the applicable date *changes*
+// from the running date (consecutive same-date marks collapse). Lines before the
+// first mark stay undated; marks past len(lines) are ignored. Pure (#59).
+func interleaveDateMarkers(lines []string, marks []dateMark) []string {
+	if len(marks) == 0 {
+		return lines
+	}
+	out := make([]string, 0, len(lines)+len(marks))
+	mi := 0
+	prevDate := ""
+	for i := 0; i < len(lines); i++ {
+		curDate := prevDate
+		for mi < len(marks) && marks[mi].line <= i {
+			curDate = marks[mi].date
+			mi++
+		}
+		if curDate != "" && curDate != prevDate {
+			out = append(out, tsMarkerLine(curDate))
+			prevDate = curDate
+		}
+		out = append(out, lines[i])
+	}
+	return out
+}
+
+// serializeRow flattens one row into ANSI-styled text. Trims trailing
+// default-styled blanks (so the viewer doesn't scroll past pad), uses
+// Style.Diff so we only emit SGR codes when the style actually changes
+// between cells, and terminates with \x1b[0m to keep the row's last
+// style from bleeding into the next line if a viewer concatenates without
+// resetting between lines.
+//
+// A non-default background space is treated as visible content (e.g.
+// inverse-video padding). Matches what the Python renderer does.
+//
+// In plain mode (plain=true) no SGR is emitted at all: the row is just its
+// visible content, trimmed to the last non-blank-*content* cell — a cell that
+// is "visible" only via a non-default background (inverse-video padding, box
+// fill) is NOT emitted in plain mode, so it must not extend the row, or a
+// trailing bordered region would become space-padding toward terminal width.
+func serializeRow(line uv.Line, plain bool) string {
+	last := -1
+	for i := range line {
+		c := &line[i]
+		// Continuation cells of a preceding wide grapheme are stored as
+		// zero-value Cell{} per the ultraviolet convention (Width=0,
+		// Content=""). They don't extend the visible row and must not
+		// emit anything in the loop below.
+		if c.IsZero() {
+			continue
+		}
+		content := c.Content
+		if content != "" && content != " " {
+			last = i
+		} else if !plain && c.Style.Bg != nil {
+			last = i
+		}
+	}
+	if last < 0 {
+		return ""
+	}
+	var b strings.Builder
+	var prev uv.Style // zero value = default; Diff vs zero emits a reset
+	first := true
+	for i := 0; i <= last; i++ {
+		c := &line[i]
+		// Skip wide-grapheme continuation cells — the wide cell already
+		// emitted its full glyph; emitting anything here adds a phantom
+		// space after every emoji.
+		if c.IsZero() {
+			continue
+		}
+		if !plain && (first || !c.Style.Equal(&prev)) {
+			b.WriteString(c.Style.Diff(&prev))
+			prev = c.Style
+			first = false
+		}
+		if c.Content == "" {
+			b.WriteByte(' ')
+		} else {
+			b.WriteString(c.Content)
+		}
+	}
+	if !plain {
+		b.WriteString("\x1b[0m")
+	}
+	return b.String()
+}
+
+// resolveMax maps a --max-lines value to a scrollback cap. <=0 means
+// "uncapped" — a continuation wants the whole session, not the viewer's
+// 2000-row window. Represented as a large sentinel; .raw is per-run
+// O_TRUNC'd, so the practical bound is the run length.
+func resolveMax(n int) int {
+	if n <= 0 {
+		return math.MaxInt32
+	}
+	return n
+}
+
+// visibleRow materializes row y of the live screen as a uv.Line. The
+// emulator exposes cells one at a time via CellAt(x,y); there's no
+// "give me the whole row" accessor. A missing cell (CellAt returns nil)
+// becomes a zero-value Cell, which serializeRow treats as a blank.
+func visibleRow(em *vt.Emulator, y, width int) uv.Line {
+	row := make(uv.Line, width)
+	for x := 0; x < width; x++ {
+		if c := em.CellAt(x, y); c != nil {
+			row[x] = *c
+		}
+	}
+	return row
+}
+
+func render(rawPath, eventsPath, outPath string, plain bool, maxLines int, withTimestamps bool) error {
+	events, err := parseEvents(eventsPath)
+	if err != nil {
+		return fmt.Errorf("parse events: %w", err)
+	}
+	cols, rows := initialSize(events)
+	em := vt.NewEmulator(cols, rows)
+	em.Scrollback().SetMaxLines(resolveMax(maxLines))
+
+	// Drain the emulator's input pipe in the background. CSI status
+	// queries (DSR, Device Attributes, etc.) in the captured stream
+	// trigger handlers that *write a reply back* into this pipe — in a
+	// real terminal those bytes go to the controlling app. Offscreen
+	// replay has no reader, so the handler's WriteString blocks
+	// forever and deadlocks the Write goroutine. Discarding the bytes
+	// preserves emulation correctness; we never act on the replies.
+	//
+	// Wait for the drainer to actually exit before letting em.Close()
+	// run, otherwise Close races with the drainer's still-pending
+	// Read() (race detector catches it; in production the window is
+	// usually harmless but it's a real ordering bug).
+	drainDone := make(chan struct{})
+	go func() {
+		defer close(drainDone)
+		_, _ = io.Copy(io.Discard, em)
+	}()
+	defer func() {
+		em.Close()
+		<-drainDone
+	}()
+
+	raw, err := os.ReadFile(rawPath)
+	if err != nil {
+		return fmt.Errorf("read raw: %w", err)
+	}
+	marks := feedSegments(em, raw, events)
+
+	// Scrollback lines (oldest → newest), then visible buffer top → bottom.
+	// Visible buffer iterates by row index rather than dropping trailing
+	// blank rows: an agent that cleared and paused mid-redraw would shift
+	// every subsequent line number otherwise, and `:880` should still land
+	// where zellij showed line 880.
+	sb := em.Scrollback()
+	viewportTop := sb.Len() + 1 // 1-indexed line where the visible buffer starts
+	out := make([]string, 0, sb.Len()+em.Height())
+	for i := 0; i < sb.Len(); i++ {
+		out = append(out, serializeRow(sb.Line(i), plain))
+	}
+	w := em.Width()
+	for y := 0; y < em.Height(); y++ {
+		out = append(out, serializeRow(visibleRow(em, y, w), plain))
+	}
+	// Trim trailing all-blank lines: a half-empty visible buffer otherwise
+	// leaves a tail of empties at EOF.
+	for len(out) > 0 && out[len(out)-1] == "" {
+		out = out[:len(out)-1]
+	}
+
+	// Change-log path only: interleave day markers from the time-event snapshots
+	// so the distiller can date entries by real change-time (#59). Done after the
+	// trailing-blank trim so a marker never dangles past content. The scrollback
+	// viewer never sets this flag → its render is byte-identical to before.
+	if withTimestamps {
+		out = interleaveDateMarkers(out, marks)
+	}
+
+	// Write the viewport sidecar *first*, then atomically rename the
+	// .ansi into place. Order matters: scrollback.lua's BufReadPost
+	// opens the .ansi and immediately reads the sidecar — flipping the
+	// .ansi last guarantees the sidecar is up-to-date by the time
+	// nvim sees the new content. Sidecar is best-effort: on write
+	// failure, scrollback.lua falls back to its prior bottom-alignment.
+	// The viewport sidecar positions the Alt+/ nvim viewer; it's meaningless
+	// for the plain projection (a continuation distills the text, not a
+	// scroll position), so skip it and don't litter a stray <out>.viewport.
+	if !plain {
+		viewportPath := strings.TrimSuffix(outPath, ".ansi") + ".viewport"
+		_ = os.WriteFile(viewportPath, []byte(strconv.Itoa(viewportTop)+"\n"), 0o644)
+	}
+
+	// Atomic write so a double-tap Alt+/ can't race truncate-then-write
+	// on the same path. Reader sees either the old complete file or the
+	// new complete file, never a half-written one.
+	tmp := outPath + ".tmp"
+	f, err := os.Create(tmp)
+	if err != nil {
+		return err
+	}
+	body := strings.Join(out, "\n")
+	if len(out) > 0 {
+		body += "\n"
+	}
+	if _, err := f.WriteString(body); err != nil {
+		f.Close()
+		os.Remove(tmp)
+		return err
+	}
+	if err := f.Close(); err != nil {
+		os.Remove(tmp)
+		return err
+	}
+	return os.Rename(tmp, outPath)
+}
+
+func Run(argv []string, stdout, stderr io.Writer) int {
+	fs := flag.NewFlagSet("pair-scrollback-render", flag.ContinueOnError)
+	fs.SetOutput(stderr)
+	fs.Usage = func() {
+		fmt.Fprintf(stderr, "usage: pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] <raw> <events.jsonl> <out>\n")
+	}
+	plain := fs.Bool("plain", false, "emit plain text (no SGR) for distillation")
+	maxLines := fs.Int("max-lines", historyRows, "scrollback history rows retained; <=0 = uncapped")
+	withTimestamps := fs.Bool("with-timestamps", false, "interleave ⟦pair:ts DATE⟧ day markers from time events (for the change log; #59)")
+	if err := fs.Parse(argv); err != nil {
+		return 2
+	}
+	args := fs.Args()
+	if len(args) != 3 {
+		fs.Usage()
+		return 2
+	}
+	if err := render(args[0], args[1], args[2], *plain, *maxLines, *withTimestamps); err != nil {
+		fmt.Fprintf(stderr, "scrollback-render: %v\n", err)
+		return 1
+	}
+	return 0
+}
diff --git a/cmd/internal/scrollbackcmd/scrollbackcmd_test.go b/cmd/internal/scrollbackcmd/scrollbackcmd_test.go
new file mode 100644
index 0000000..db14e4b
--- /dev/null
+++ b/cmd/internal/scrollbackcmd/scrollbackcmd_test.go
@@ -0,0 +1,48 @@
+package scrollbackcmd
+
+import (
+	"bytes"
+	"io"
+	"os"
+	"path/filepath"
+	"strings"
+	"testing"
+)
+
+func TestRunUsage(t *testing.T) {
+	var stderr bytes.Buffer
+	code := Run([]string{}, io.Discard, &stderr)
+	if code != 2 {
+		t.Fatalf("code = %d, want 2", code)
+	}
+	if !strings.Contains(stderr.String(), "usage: pair-scrollback-render") {
+		t.Fatalf("stderr missing usage:\n%s", stderr.String())
+	}
+}
+
+func TestRunWritesOutput(t *testing.T) {
+	dir := t.TempDir()
+	rawPath := filepath.Join(dir, "in.raw")
+	evPath := filepath.Join(dir, "in.events.jsonl")
+	outPath := filepath.Join(dir, "out.ansi")
+	if err := os.WriteFile(rawPath, []byte("hello\r\n"), 0o644); err != nil {
+		t.Fatal(err)
+	}
+	events := `{"type":"resize","offset":0,"cols":20,"rows":5}` + "\n"
+	if err := os.WriteFile(evPath, []byte(events), 0o644); err != nil {
+		t.Fatal(err)
+	}
+
+	var stderr bytes.Buffer
+	code := Run([]string{rawPath, evPath, outPath}, io.Discard, &stderr)
+	if code != 0 {
+		t.Fatalf("code = %d, want 0; stderr:\n%s", code, stderr.String())
+	}
+	body, err := os.ReadFile(outPath)
+	if err != nil {
+		t.Fatalf("read out: %v", err)
+	}
+	if !strings.Contains(string(body), "hello") {
+		t.Fatalf("output missing rendered text:\n%s", string(body))
+	}
+}
diff --git a/cmd/pair-scrollback-render/serialize_row_test.go b/cmd/internal/scrollbackcmd/serialize_row_test.go
similarity index 99%
rename from cmd/pair-scrollback-render/serialize_row_test.go
rename to cmd/internal/scrollbackcmd/serialize_row_test.go
index 22050ab..1f53365 100644
--- a/cmd/pair-scrollback-render/serialize_row_test.go
+++ b/cmd/internal/scrollbackcmd/serialize_row_test.go
@@ -1,4 +1,4 @@
-package main
+package scrollbackcmd
 
 import (
 	"image/color"
diff --git a/cmd/pair-scrollback-render/timestamps_test.go b/cmd/internal/scrollbackcmd/timestamps_test.go
similarity index 99%
rename from cmd/pair-scrollback-render/timestamps_test.go
rename to cmd/internal/scrollbackcmd/timestamps_test.go
index 71a65aa..798f14e 100644
--- a/cmd/pair-scrollback-render/timestamps_test.go
+++ b/cmd/internal/scrollbackcmd/timestamps_test.go
@@ -1,4 +1,4 @@
-package main
+package scrollbackcmd
 
 import (
 	"fmt"
diff --git a/cmd/pair-context/main.go b/cmd/pair-context/main.go
index 92d1267..7b26f7c 100644
--- a/cmd/pair-context/main.go
+++ b/cmd/pair-context/main.go
@@ -5,57 +5,11 @@
 package main
 
 import (
-	"encoding/json"
-	"fmt"
 	"os"
-	"path/filepath"
 
-	"github.com/xianxu/pair/cmd/internal/ctxmeter"
-	"github.com/xianxu/pair/cmd/internal/transcript"
+	"github.com/xianxu/pair/cmd/internal/contextcmd"
 )
 
 func main() {
-	if len(os.Args) < 3 {
-		return
-	}
-	tag, agent := os.Args[1], os.Args[2]
-	dataDir := os.Getenv("PAIR_DATA_DIR")
-	if dataDir == "" {
-		base := os.Getenv("XDG_DATA_HOME")
-		if base == "" {
-			base = filepath.Join(os.Getenv("HOME"), ".local", "share")
-		}
-		dataDir = filepath.Join(base, "pair")
-	}
-	sid := transcript.SessionID(dataDir, tag, agent)
-	if sid == "" {
-		return
-	}
-	cwd := paneCwd(dataDir, tag, agent) // "" for codex/agy is fine (Resolve ignores it)
-	path := transcript.Resolve(agent, sid, cwd, os.Getenv("HOME"))
-	if path == "" {
-		return
-	}
-	f, err := os.Open(path)
-	if err != nil {
-		return
-	}
-	defer f.Close()
-	if n, ok := ctxmeter.ContextTokens(agent, f); ok {
-		fmt.Println(ctxmeter.Humanize(n))
-	}
-}
-
-func paneCwd(dataDir, tag, agent string) string {
-	b, err := os.ReadFile(filepath.Join(dataDir, "pane-"+tag+"-"+agent+".json"))
-	if err != nil {
-		return ""
-	}
-	var p struct {
-		Cwd string `json:"cwd"`
-	}
-	if json.Unmarshal(b, &p) != nil {
-		return ""
-	}
-	return p.Cwd
+	os.Exit(contextcmd.Run(os.Args[1:], contextcmd.EnvFromOS(), os.Stdout))
 }
diff --git a/cmd/pair-go/helper_equivalence_test.go b/cmd/pair-go/helper_equivalence_test.go
new file mode 100644
index 0000000..6905b86
--- /dev/null
+++ b/cmd/pair-go/helper_equivalence_test.go
@@ -0,0 +1,93 @@
+package main
+
+import (
+	"bytes"
+	"os"
+	"os/exec"
+	"path/filepath"
+	"strings"
+	"testing"
+)
+
+func TestPairGoContextMatchesLegacyPairContext(t *testing.T) {
+	bin := t.TempDir()
+	pairContext := filepath.Join(bin, "pair-context")
+	pairGo := filepath.Join(bin, "pair-go")
+	buildCommand(t, pairContext, "../pair-context")
+	buildCommand(t, pairGo, ".")
+
+	home, data := writeContextFixture(t)
+	env := append(os.Environ(), "HOME="+home, "PAIR_DATA_DIR="+data)
+
+	legacy := runCommand(t, env, pairContext, "T", "claude")
+	dispatch := runCommand(t, env, pairGo, "context", "T", "claude")
+	if dispatch.code != legacy.code || dispatch.stdout != legacy.stdout || dispatch.stderr != legacy.stderr {
+		t.Fatalf("pair-go context mismatch\nlegacy:   code=%d stdout=%q stderr=%q\ndispatch: code=%d stdout=%q stderr=%q",
+			legacy.code, legacy.stdout, legacy.stderr,
+			dispatch.code, dispatch.stdout, dispatch.stderr)
+	}
+}
+
+func buildCommand(t *testing.T, out, pkg string) {
+	t.Helper()
+	cmd := exec.Command("go", "build", "-o", out, pkg)
+	if body, err := cmd.CombinedOutput(); err != nil {
+		t.Fatalf("build %s: %v\n%s", pkg, err, string(body))
+	}
+}
+
+type commandResult struct {
+	code   int
+	stdout string
+	stderr string
+}
+
+func runCommand(t *testing.T, env []string, name string, args ...string) commandResult {
+	t.Helper()
+	cmd := exec.Command(name, args...)
+	cmd.Env = env
+	var stdout, stderr bytes.Buffer
+	cmd.Stdout = &stdout
+	cmd.Stderr = &stderr
+	err := cmd.Run()
+	code := 0
+	if err != nil {
+		exit, ok := err.(*exec.ExitError)
+		if !ok {
+			t.Fatalf("run %s: %v", name, err)
+		}
+		code = exit.ExitCode()
+	}
+	return commandResult{code: code, stdout: stdout.String(), stderr: stderr.String()}
+}
+
+func writeContextFixture(t *testing.T) (home, data string) {
+	t.Helper()
+	home = t.TempDir()
+	data = filepath.Join(home, "data")
+	cwd := filepath.Join(home, "repo")
+	enc := strings.NewReplacer(".", "-", "/", "-").Replace(cwd)
+	proj := filepath.Join(home, ".claude", "projects", enc)
+	mustMkdir(t, data)
+	mustMkdir(t, cwd)
+	mustMkdir(t, proj)
+	mustWrite(t, filepath.Join(data, "config-T-claude.json"), `{"session_id":"sid1"}`)
+	mustWrite(t, filepath.Join(data, "pane-T-claude.json"), `{"pane_id":"7","cwd":"`+cwd+`","cwd_display":"~/repo"}`)
+	mustWrite(t, filepath.Join(proj, "sid1.jsonl"),
+		`{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":397556,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`)
+	return home, data
+}
+
+func mustMkdir(t *testing.T, d string) {
+	t.Helper()
+	if err := os.MkdirAll(d, 0o755); err != nil {
+		t.Fatal(err)
+	}
+}
+
+func mustWrite(t *testing.T, p, s string) {
+	t.Helper()
+	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
+		t.Fatal(err)
+	}
+}
diff --git a/cmd/pair-scrollback-render/main.go b/cmd/pair-scrollback-render/main.go
index 57da8df..6e656be 100644
--- a/cmd/pair-scrollback-render/main.go
+++ b/cmd/pair-scrollback-render/main.go
@@ -1,408 +1,12 @@
-// scrollback-render — replay a pair-wrap raw capture through a VT100
-// emulator and write one ANSI-styled output line per logical scrollback row.
-// Drop-in replacement for the Python+pyte renderer at bin/pair-scrollback-render.
-//
-// Why Go: pyte's HistoryScreen dispatches every method call through an
-// __getattribute__ override that ran ~19M times for a 3 MB raw input —
-// ~95% of wall time. Even after the CaptureScreen patch (3.6x speedup),
-// the Python interpreter + pyte vendoring add startup cost and a private
-// venv that the brew formula has to manage. A static Go binary using
-// charmbracelet/x/vt replays the same stream with no runtime deps and
-// stays within the pair repo's existing cmd/ layout.
-//
-// Pipeline:
-//
-//	raw bytes (.raw)              → emulator.Write(...) in segments
-//	resize events (.events.jsonl) → segment boundaries with new (cols,rows)
-//	final emulator state          → scrollback lines + visible buffer
-//	each row                      → SGR-decorated text line written to out
-//
-// CLI is identical to the Python version so bin/pair-scrollback-open can
-// invoke either:
-//
-//	scrollback-render <raw> <events.jsonl> <out.ansi>
+// scrollback-render — replay a pair-wrap raw capture through a VT100 emulator.
 package main
 
 import (
-	"encoding/json"
-	"flag"
-	"fmt"
-	"io"
-	"math"
 	"os"
-	"strconv"
-	"strings"
-	"time"
 
-	uv "github.com/charmbracelet/ultraviolet"
-	"github.com/charmbracelet/x/vt"
+	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
 )
 
-// historyRows caps scrolled-out rows retained for the viewer. Matched
-// to zellij/config.kdl's `scroll_buffer_size 2000` so PageUp inside the
-// agent pane and Alt+/ both reach back the same distance — otherwise
-// the viewer would surface lines that zellij no longer has, leaving
-// the scroll-overlay logic in pair-scrollback-open unable to align
-// against the agent pane's frame.
-const historyRows = 2_000
-
-const (
-	defaultCols = 80
-	defaultRows = 24
-)
-
-type scrollbackEvent struct {
-	Type   string `json:"type"`
-	Offset int64  `json:"offset"`
-	Cols   int    `json:"cols"`
-	Rows   int    `json:"rows"`
-	Ts     string `json:"ts,omitempty"` // RFC3339 wall-clock for "time" events (#59)
-}
-
-// dateOf extracts the YYYY-MM-DD day from an RFC3339 timestamp; "" on a
-// malformed value so a corrupt time event degrades to undated, never panics (#59).
-func dateOf(ts string) string {
-	t, err := time.Parse(time.RFC3339, ts)
-	if err != nil {
-		return ""
-	}
-	return t.Format("2006-01-02")
-}
-
-// parseEvents reads the sidecar JSONL. Empty / missing file → empty slice.
-// Malformed lines are skipped so a corrupted tail doesn't abort the render —
-// imperfect width tracking beats an unusable viewer.
-func parseEvents(path string) ([]scrollbackEvent, error) {
-	data, err := os.ReadFile(path)
-	if err != nil {
-		if os.IsNotExist(err) {
-			return nil, nil
-		}
-		return nil, err
-	}
-	var out []scrollbackEvent
-	for _, line := range strings.Split(string(data), "\n") {
-		line = strings.TrimSpace(line)
-		if line == "" {
-			continue
-		}
-		var e scrollbackEvent
-		if err := json.Unmarshal([]byte(line), &e); err != nil {
-			continue
-		}
-		// Keep both known types: resize boundaries AND time stamps (#59).
-		// Consumers filter by Type at their use sites.
-		if e.Type == "resize" || e.Type == "time" {
-			out = append(out, e)
-		}
-	}
-	return out, nil
-}
-
-// initialSize pulls (cols, rows) from the first usable resize event, or
-// falls back to 80x24 if the sidecar is empty or the first entry is
-// malformed. pair-wrap always emits an initial resize at offset 0, so the
-// fallback only fires on a truncated file.
-func initialSize(events []scrollbackEvent) (int, int) {
-	for _, e := range events {
-		if e.Type == "resize" && e.Cols > 0 && e.Rows > 0 {
-			return e.Cols, e.Rows
-		}
-	}
-	return defaultCols, defaultRows
-}
-
-// dateMark records the emulator's scrollback length at a "time" event's byte
-// offset → the day that applies to committed lines from that index onward (#59).
-// Built during the feed (the only place that knows both byte offsets and the
-// rendered line count); consumed by the pure interleaveDateMarkers.
-type dateMark struct {
-	line int
-	date string
-}
-
-// feedSegments writes raw into the emulator as a single offset-ordered walk over
-// ALL sidecar events: write everything up to event.Offset, then act — Resize on a
-// resize event, or snapshot Scrollback().Len() on a time event. Returns the time
-// snapshots (empty unless time events are present). The caller already set the
-// initial size via initialSize; re-applying the offset-0 resize here is a harmless
-// no-op (resize to the current dimensions). Walking all events — rather than
-// events[1:] — means a time event in any position (incl. first) is captured, and
-// an empty events slice is handled without an out-of-range slice (#59).
-//
-// Clamping Offset to len(raw) defends against a corrupted sidecar that records
-// an offset beyond EOF (saw this once with a half-written events file after a
-// hard kill); without clamping we'd panic on the slice.
-func feedSegments(em *vt.Emulator, raw []byte, events []scrollbackEvent) []dateMark {
-	var cursor int64
-	var marks []dateMark
-	for _, e := range events {
-		off := e.Offset
-		if off > int64(len(raw)) {
-			off = int64(len(raw))
-		}
-		if off > cursor {
-			_, _ = em.Write(raw[cursor:off])
-			cursor = off
-		}
-		switch e.Type {
-		case "resize":
-			em.Resize(e.Cols, e.Rows)
-		case "time":
-			if d := dateOf(e.Ts); d != "" {
-				marks = append(marks, dateMark{line: em.Scrollback().Len(), date: d})
-			}
-		}
-	}
-	if cursor < int64(len(raw)) {
-		_, _ = em.Write(raw[cursor:])
-	}
-	return marks
-}
-
-// tsMarkerLine is the wire format the distiller parses (#59). MUST stay in sync
-// with tsMarkerRe in cmd/pair-changelog/distill.go — the contract is pinned by
-// the render→clean→distill e2e test cmd/pair-changelog/e2e_test.go
-// (TestEndToEndMarkerSurvival), which feeds real time events through both binaries.
-func tsMarkerLine(date string) string {
-	return "⟦pair:ts " + date + "⟧"
-}
-
-// interleaveDateMarkers inserts a tsMarkerLine immediately before the first line
-// of each new date run. marks are (scrollback-line-index, date) snapshots in
-// ascending index; a marker is emitted only when the applicable date *changes*
-// from the running date (consecutive same-date marks collapse). Lines before the
-// first mark stay undated; marks past len(lines) are ignored. Pure (#59).
-func interleaveDateMarkers(lines []string, marks []dateMark) []string {
-	if len(marks) == 0 {
-		return lines
-	}
-	out := make([]string, 0, len(lines)+len(marks))
-	mi := 0
-	prevDate := ""
-	for i := 0; i < len(lines); i++ {
-		curDate := prevDate
-		for mi < len(marks) && marks[mi].line <= i {
-			curDate = marks[mi].date
-			mi++
-		}
-		if curDate != "" && curDate != prevDate {
-			out = append(out, tsMarkerLine(curDate))
-			prevDate = curDate
-		}
-		out = append(out, lines[i])
-	}
-	return out
-}
-
-// serializeRow flattens one row into ANSI-styled text. Trims trailing
-// default-styled blanks (so the viewer doesn't scroll past pad), uses
-// Style.Diff so we only emit SGR codes when the style actually changes
-// between cells, and terminates with \x1b[0m to keep the row's last
-// style from bleeding into the next line if a viewer concatenates without
-// resetting between lines.
-//
-// A non-default background space is treated as visible content (e.g.
-// inverse-video padding). Matches what the Python renderer does.
-//
-// In plain mode (plain=true) no SGR is emitted at all: the row is just its
-// visible content, trimmed to the last non-blank-*content* cell — a cell that
-// is "visible" only via a non-default background (inverse-video padding, box
-// fill) is NOT emitted in plain mode, so it must not extend the row, or a
-// trailing bordered region would become space-padding toward terminal width.
-func serializeRow(line uv.Line, plain bool) string {
-	last := -1
-	for i := range line {
-		c := &line[i]
-		// Continuation cells of a preceding wide grapheme are stored as
-		// zero-value Cell{} per the ultraviolet convention (Width=0,
-		// Content=""). They don't extend the visible row and must not
-		// emit anything in the loop below.
-		if c.IsZero() {
-			continue
-		}
-		content := c.Content
-		if content != "" && content != " " {
-			last = i
-		} else if !plain && c.Style.Bg != nil {
-			last = i
-		}
-	}
-	if last < 0 {
-		return ""
-	}
-	var b strings.Builder
-	var prev uv.Style // zero value = default; Diff vs zero emits a reset
-	first := true
-	for i := 0; i <= last; i++ {
-		c := &line[i]
-		// Skip wide-grapheme continuation cells — the wide cell already
-		// emitted its full glyph; emitting anything here adds a phantom
-		// space after every emoji.
-		if c.IsZero() {
-			continue
-		}
-		if !plain && (first || !c.Style.Equal(&prev)) {
-			b.WriteString(c.Style.Diff(&prev))
-			prev = c.Style
-			first = false
-		}
-		if c.Content == "" {
-			b.WriteByte(' ')
-		} else {
-			b.WriteString(c.Content)
-		}
-	}
-	if !plain {
-		b.WriteString("\x1b[0m")
-	}
-	return b.String()
-}
-
-// resolveMax maps a --max-lines value to a scrollback cap. <=0 means
-// "uncapped" — a continuation wants the whole session, not the viewer's
-// 2000-row window. Represented as a large sentinel; .raw is per-run
-// O_TRUNC'd, so the practical bound is the run length.
-func resolveMax(n int) int {
-	if n <= 0 {
-		return math.MaxInt32
-	}
-	return n
-}
-
-// visibleRow materializes row y of the live screen as a uv.Line. The
-// emulator exposes cells one at a time via CellAt(x,y); there's no
-// "give me the whole row" accessor. A missing cell (CellAt returns nil)
-// becomes a zero-value Cell, which serializeRow treats as a blank.
-func visibleRow(em *vt.Emulator, y, width int) uv.Line {
-	row := make(uv.Line, width)
-	for x := 0; x < width; x++ {
-		if c := em.CellAt(x, y); c != nil {
-			row[x] = *c
-		}
-	}
-	return row
-}
-
-func render(rawPath, eventsPath, outPath string, plain bool, maxLines int, withTimestamps bool) error {
-	events, err := parseEvents(eventsPath)
-	if err != nil {
-		return fmt.Errorf("parse events: %w", err)
-	}
-	cols, rows := initialSize(events)
-	em := vt.NewEmulator(cols, rows)
-	em.Scrollback().SetMaxLines(resolveMax(maxLines))
-
-	// Drain the emulator's input pipe in the background. CSI status
-	// queries (DSR, Device Attributes, etc.) in the captured stream
-	// trigger handlers that *write a reply back* into this pipe — in a
-	// real terminal those bytes go to the controlling app. Offscreen
-	// replay has no reader, so the handler's WriteString blocks
-	// forever and deadlocks the Write goroutine. Discarding the bytes
-	// preserves emulation correctness; we never act on the replies.
-	//
-	// Wait for the drainer to actually exit before letting em.Close()
-	// run, otherwise Close races with the drainer's still-pending
-	// Read() (race detector catches it; in production the window is
-	// usually harmless but it's a real ordering bug).
-	drainDone := make(chan struct{})
-	go func() {
-		defer close(drainDone)
-		_, _ = io.Copy(io.Discard, em)
-	}()
-	defer func() {
-		em.Close()
-		<-drainDone
-	}()
-
-	raw, err := os.ReadFile(rawPath)
-	if err != nil {
-		return fmt.Errorf("read raw: %w", err)
-	}
-	marks := feedSegments(em, raw, events)
-
-	// Scrollback lines (oldest → newest), then visible buffer top → bottom.
-	// Visible buffer iterates by row index rather than dropping trailing
-	// blank rows: an agent that cleared and paused mid-redraw would shift
-	// every subsequent line number otherwise, and `:880` should still land
-	// where zellij showed line 880.
-	sb := em.Scrollback()
-	viewportTop := sb.Len() + 1 // 1-indexed line where the visible buffer starts
-	out := make([]string, 0, sb.Len()+em.Height())
-	for i := 0; i < sb.Len(); i++ {
-		out = append(out, serializeRow(sb.Line(i), plain))
-	}
-	w := em.Width()
-	for y := 0; y < em.Height(); y++ {
-		out = append(out, serializeRow(visibleRow(em, y, w), plain))
-	}
-	// Trim trailing all-blank lines: a half-empty visible buffer otherwise
-	// leaves a tail of empties at EOF.
-	for len(out) > 0 && out[len(out)-1] == "" {
-		out = out[:len(out)-1]
-	}
-
-	// Change-log path only: interleave day markers from the time-event snapshots
-	// so the distiller can date entries by real change-time (#59). Done after the
-	// trailing-blank trim so a marker never dangles past content. The scrollback
-	// viewer never sets this flag → its render is byte-identical to before.
-	if withTimestamps {
-		out = interleaveDateMarkers(out, marks)
-	}
-
-	// Write the viewport sidecar *first*, then atomically rename the
-	// .ansi into place. Order matters: scrollback.lua's BufReadPost
-	// opens the .ansi and immediately reads the sidecar — flipping the
-	// .ansi last guarantees the sidecar is up-to-date by the time
-	// nvim sees the new content. Sidecar is best-effort: on write
-	// failure, scrollback.lua falls back to its prior bottom-alignment.
-	// The viewport sidecar positions the Alt+/ nvim viewer; it's meaningless
-	// for the plain projection (a continuation distills the text, not a
-	// scroll position), so skip it and don't litter a stray <out>.viewport.
-	if !plain {
-		viewportPath := strings.TrimSuffix(outPath, ".ansi") + ".viewport"
-		_ = os.WriteFile(viewportPath, []byte(strconv.Itoa(viewportTop)+"\n"), 0o644)
-	}
-
-	// Atomic write so a double-tap Alt+/ can't race truncate-then-write
-	// on the same path. Reader sees either the old complete file or the
-	// new complete file, never a half-written one.
-	tmp := outPath + ".tmp"
-	f, err := os.Create(tmp)
-	if err != nil {
-		return err
-	}
-	body := strings.Join(out, "\n")
-	if len(out) > 0 {
-		body += "\n"
-	}
-	if _, err := f.WriteString(body); err != nil {
-		f.Close()
-		os.Remove(tmp)
-		return err
-	}
-	if err := f.Close(); err != nil {
-		os.Remove(tmp)
-		return err
-	}
-	return os.Rename(tmp, outPath)
-}
-
 func main() {
-	flag.Usage = func() {
-		fmt.Fprintf(os.Stderr, "usage: %s [--plain] [--max-lines N] [--with-timestamps] <raw> <events.jsonl> <out>\n", os.Args[0])
-	}
-	plain := flag.Bool("plain", false, "emit plain text (no SGR) for distillation")
-	maxLines := flag.Int("max-lines", historyRows, "scrollback history rows retained; <=0 = uncapped")
-	withTimestamps := flag.Bool("with-timestamps", false, "interleave ⟦pair:ts DATE⟧ day markers from time events (for the change log; #59)")
-	flag.Parse()
-	args := flag.Args()
-	if len(args) != 3 {
-		flag.Usage()
-		os.Exit(2)
-	}
-	if err := render(args[0], args[1], args[2], *plain, *maxLines, *withTimestamps); err != nil {
-		fmt.Fprintf(os.Stderr, "scrollback-render: %v\n", err)
-		os.Exit(1)
-	}
+	os.Exit(scrollbackcmd.Run(os.Args[1:], os.Stdout, os.Stderr))
 }
diff --git a/workshop/plans/000076-go-helper-dispatch-plan.md b/workshop/plans/000076-go-helper-dispatch-plan.md
index 88f9856..2e8ce49 100644
--- a/workshop/plans/000076-go-helper-dispatch-plan.md
+++ b/workshop/plans/000076-go-helper-dispatch-plan.md
@@ -67,7 +67,7 @@
 - Modify: `cmd/pair-context/main_test.go`
 - Modify: `Makefile.local`
 
-- [ ] **Step 1: Add failing runner tests**
+- [x] **Step 1: Add failing runner tests**
 
 Create `cmd/internal/contextcmd/contextcmd_test.go` with tests that call `Run(args []string, env Env, stdout io.Writer) int` directly:
 
@@ -84,13 +84,13 @@ func TestRunMissingConfigPrintsNothing(t *testing.T) {
 }
 ```
 
-- [ ] **Step 2: Run the focused tests and confirm they fail**
+- [x] **Step 2: Run the focused tests and confirm they fail**
 
 Run: `go test ./cmd/internal/contextcmd -count=1`
 
 Expected: FAIL because the package does not exist yet.
 
-- [ ] **Step 3: Extract the runner**
+- [x] **Step 3: Extract the runner**
 
 Move the reusable context behavior into `cmd/internal/contextcmd`. Keep `cmd/pair-context/main.go` as:
 
@@ -106,13 +106,13 @@ The runner must:
 - write the same humanized token count to the injected stdout;
 - never call `os.Exit`.
 
-- [ ] **Step 4: Run the focused tests and existing package tests**
+- [x] **Step 4: Run the focused tests and existing package tests**
 
 Run: `go test ./cmd/internal/contextcmd ./cmd/pair-context -count=1`
 
 Expected: PASS.
 
-- [ ] **Step 5: Commit the context extraction**
+- [x] **Step 5: Commit the context extraction**
 
 Run:
 
@@ -134,7 +134,7 @@ git commit -m "#76: extract pair-context runner"
 - Modify or create tests in: `cmd/pair-scrollback-render/*_test.go`
 - Modify: `Makefile.local`
 
-- [ ] **Step 1: Add failing runner tests**
+- [x] **Step 1: Add failing runner tests**
 
 Create tests that call `scrollbackcmd.Run(args []string, stdout, stderr io.Writer) int`:
 
@@ -152,13 +152,13 @@ func TestRunWritesOutput(t *testing.T) {
 }
 ```
 
-- [ ] **Step 2: Run the focused tests and confirm they fail**
+- [x] **Step 2: Run the focused tests and confirm they fail**
 
 Run: `go test ./cmd/internal/scrollbackcmd -count=1`
 
 Expected: FAIL because the package does not exist yet.
 
-- [ ] **Step 3: Extract the runner**
+- [x] **Step 3: Extract the runner**
 
 Move the renderer command wrapper into `cmd/internal/scrollbackcmd`. If `render(...)` cannot be imported from `package main`, move the rendering core into this internal package too and leave the legacy command as a tiny wrapper:
 
@@ -174,13 +174,13 @@ Use a local `flag.FlagSet` so dispatcher and tests can parse independently. Pres
 - exit 0 for success;
 - same defaults for `--plain`, `--max-lines`, and `--with-timestamps`.
 
-- [ ] **Step 4: Run focused and package tests**
+- [x] **Step 4: Run focused and package tests**
 
 Run: `go test ./cmd/internal/scrollbackcmd ./cmd/pair-scrollback-render -count=1`
 
 Expected: PASS.
 
-- [ ] **Step 5: Commit the renderer extraction**
+- [x] **Step 5: Commit the renderer extraction**
 
 Run:
 
@@ -202,20 +202,20 @@ git commit -m "#76: extract scrollback renderer runner"
 - Create: `cmd/pair-go/helper_equivalence_test.go`
 - Modify: `Makefile.local`
 
-- [ ] **Step 1: Add failing dispatcher tests**
+- [x] **Step 1: Add failing dispatcher tests**
 
 Add tests for:
 - `Dispatch([]string{"context", "T", "claude"})` through a fake or temp fixture returns the same output shape as `contextcmd.Run`;
 - `Dispatch([]string{"scrollback-render"})` returns renderer usage with exit 2;
 - top-level help lists `context` and `scrollback-render` as implemented helper routes, not planned-only commands.
 
-- [ ] **Step 2: Run dispatcher tests and confirm failure**
+- [x] **Step 2: Run dispatcher tests and confirm failure**
 
 Run: `go test ./cmd/internal/dispatcher -run 'TestDispatch(Context|Scrollback|Help)' -count=1`
 
 Expected: FAIL because routes are not implemented yet.
 
-- [ ] **Step 3: Add failing process-level equivalence test**
+- [x] **Step 3: Add failing process-level equivalence test**
 
 Create `cmd/pair-go/helper_equivalence_test.go` with a context helper fixture that builds both commands and runs them against the same temp Pair data/transcript tree:
 
@@ -231,13 +231,13 @@ func TestPairGoContextMatchesLegacyPairContext(t *testing.T) {
 
 This is the representative compatibility proof required by the issue done-when. It should fail before the dispatcher route exists because `pair-go context` is still planned-only.
 
-- [ ] **Step 4: Run the equivalence test and confirm failure**
+- [x] **Step 4: Run the equivalence test and confirm failure**
 
 Run: `go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1`
 
 Expected: FAIL because `pair-go context` returns planned-but-not-implemented.
 
-- [ ] **Step 5: Implement routes**
+- [x] **Step 5: Implement routes**
 
 Update `Families()` statuses for selected helpers and add dispatcher branches:
 
@@ -250,7 +250,7 @@ case "scrollback-render":
 
 The dispatcher should continue returning `dispatcher.Result`. Helper runners should write into buffers so dispatcher can map stdout/stderr/exit code without adding a second process-result abstraction.
 
-- [ ] **Step 6: Run route and process tests**
+- [x] **Step 6: Run route and process tests**
 
 Run:
 
@@ -261,7 +261,7 @@ make pair-context pair-scrollback-render pair-go
 
 Expected: PASS. The `make` command is deliberately not `-B`; it verifies the updated dependency graph can rebuild normally after source changes.
 
-- [ ] **Step 7: Commit dispatcher wiring**
+- [x] **Step 7: Commit dispatcher wiring**
 
 Run:
 
@@ -282,7 +282,7 @@ git commit -m "#76: route selected helpers through pair-go"
 - Modify: `workshop/issues/000076-go-helper-dispatch.md`
 - Modify: `Makefile.local`
 
-- [ ] **Step 1: Verify legacy binaries still build**
+- [x] **Step 1: Verify legacy binaries still build**
 
 Run:
 
@@ -293,7 +293,7 @@ make -B pair-context pair-scrollback-render pair-go
 
 Expected: PASS. The non-`-B` run verifies incremental prerequisites include `cmd/internal/contextcmd`, `cmd/internal/scrollbackcmd`, and dispatcher dependencies; the `-B` run remains the forced clean rebuild check.
 
-- [ ] **Step 2: Verify selected command equivalence**
+- [x] **Step 2: Verify selected command equivalence**
 
 Run focused commands against test fixtures or package tests:
 
@@ -304,13 +304,13 @@ go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1
 
 Expected: PASS; the equivalence test demonstrates the legacy `pair-context` binary and `pair-go context` process path produce identical stdout/stderr/exit code on the same fixture.
 
-- [ ] **Step 3: Verify full Go test suite**
+- [x] **Step 3: Verify full Go test suite**
 
 Run: `go test ./... -count=1`
 
 Expected: PASS.
 
-- [ ] **Step 4: Verify no live call sites moved**
+- [x] **Step 4: Verify no live call sites moved**
 
 Run:
 
@@ -320,17 +320,17 @@ git diff -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollb
 
 Expected: empty diff, unless a test-only or documentation-only change was explicitly made.
 
-- [ ] **Step 5: Update atlas**
+- [x] **Step 5: Update atlas**
 
 Update:
 - `atlas/architecture.md` to say `pair-go context` and `pair-go scrollback-render` are implemented helper routes while public launcher and live zellij/nvim callers remain legacy.
 - `atlas/go-migration-inventory.md` rows for `pair-context` and `pair-scrollback-render` to record dispatcher availability and preserved legacy names.
 
-- [ ] **Step 6: Update issue checklist and log**
+- [x] **Step 6: Update issue checklist and log**
 
 Tick the #76 plan/done items that are complete and add a log entry with verification commands and `ARCH-*` notes.
 
-- [ ] **Step 7: Run final verification before close**
+- [x] **Step 7: Run final verification before close**
 
 Run:
 


codex
I’m using `superpowers-using-superpowers` plus the SDLC/review-specific instructions from the repo: I’ll read the required workflow docs and the issue/plan, then inspect the actual diff from the base commit before issuing the machine-readable verdict.
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
I’ve got the workflow contract. Next I’m reading the issue, durable plan, and the actual base-to-head diff from the working repo so the review is grounded in the tracked contract rather than the prompt excerpt alone.
exec
/bin/zsh -lc "sed -n '1,260p' workshop/issues/000076-go-helper-dispatch.md" in /Users/xianxu/workspace/pair
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

 succeeded in 0ms:
---
id: 000076
status: done
deps: [000074]
github_issue:
created: 2026-06-26
updated: 2026-06-30
estimate_hours: 2.86
started: 2026-06-30T11:58:44-07:00
actual_hours: 0.44
---

# pair Go helper dispatch

## Problem

Pair already has several Go helpers, but packaging still exposes them as separate binaries in `bin/`. A single-primary-binary architecture should route those helpers through `pair` without copying code or breaking existing callers.

## Spec

Fold the first low-risk Go helper commands behind the dispatcher introduced by #74. This milestone proves the shared-runner pattern without moving live zellij/nvim call sites yet.

- `pair context`
- `pair scrollback-render`

The old binary names must continue to build and work during this milestone. Existing zellij/nvim/script callers should not change yet; `bin/pair-title.sh`, `bin/pair-scrollback-open`, `bin/pair-changelog-open`, and `nvim/scrollback.lua` keep invoking the legacy command names.

Implementation should extract shared run functions from the existing `package main` commands so both the legacy binaries and `pair-go <subcommand>` routes call the same behavior (`ARCH-DRY`). The dispatcher remains the only process-facing result abstraction for in-process helper routes; helper packages should write to injected stdout/stderr and return integer exit codes or errors rather than calling `os.Exit` in reusable code (`ARCH-PURE`).

Out of scope for this milestone: `pair wrap`, `pair slug`, `pair changelog`, `pair continuation`, `pair scribe`, public `pair` entrypoint changes, install alias changes, and zellij/nvim call-site rewrites. Those remain candidates for later issues once the low-risk dispatch pattern is proven.

## Done when

- [x] Dispatcher can invoke selected existing Go helpers through `pair-go <subcommand>`.
- [x] Existing helper binary names still build and work.
- [x] Tests prove dispatch and legacy command paths reach the same behavior for at least one representative helper.
- [x] No zellij/nvim keybinding breaks.
- [x] Pair remains usable after merge.

## Plan

- [x] Choose the first helper set based on #73.
- [x] Extract reusable run functions for `pair-context` and `pair-scrollback-render`.
- [x] Add dispatcher routes for `context` and `scrollback-render`.
- [x] Preserve legacy binary names.
- [x] Run helper-specific and full relevant integration tests.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.60 impl=0.45
item: skill-or-dispatcher design=0.30 impl=0.30
item: smaller-go-module design=0.40 impl=0.25
item: atlas-docs design=0.10 impl=0.05
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 2.86
```

## Log

### 2026-06-26

Created from #72. This milestone reduces packaging surface while preserving current command names.

### 2026-06-30
- 2026-06-30: closed — go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1; go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1; make pair-context pair-scrollback-render pair-go; make -B pair-context pair-scrollback-render pair-go; go test ./... -count=1; git diff -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollback-open bin/pair-changelog-open empty; rg atlas helper dispatch; git diff --check

Claimed after #75 landed. Narrowed the first helper dispatch slice to `context` and `scrollback-render`: they are useful enough to prove the dispatcher path, but low-risk enough to avoid long-running PTY, model, git commit/push, or public launcher behavior. Existing zellij/nvim/shell callers stay on legacy binary names for this milestone (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`).

Extracted `cmd/internal/contextcmd` and `cmd/internal/scrollbackcmd` runners so legacy binaries and dispatcher routes share implementation (`ARCH-DRY`, `ARCH-PURE`). Added `pair-go context` and `pair-go scrollback-render` dispatcher routes plus a process-level equivalence test proving `pair-go context` matches `pair-context` stdout/stderr/exit code on the same fixture. Updated atlas to record the current helper-dispatch state and unchanged live shell/Lua callers.

Verification before close: `go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1`; `go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1`; `make pair-context pair-scrollback-render pair-go`; `make -B pair-context pair-scrollback-render pair-go`; `go test ./... -count=1`; `git diff -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollback-open bin/pair-changelog-open` empty; atlas grep found `pair-go context`, `pair-go scrollback-render`, and helper dispatch; `git diff --check`.

exec
/bin/zsh -lc "sed -n '1,380p' workshop/plans/000076-go-helper-dispatch-plan.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
# Go Helper Dispatch Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `pair-go context` and `pair-go scrollback-render` routes that reuse the existing helper implementations while preserving the legacy helper binaries.

**Architecture:** Extract reusable helper runners from the selected `package main` commands and route the dispatcher through those runners instead of reimplementing command behavior (`ARCH-DRY`). Keep helper business logic pure or close to existing pure cores, with IO at thin command boundaries: runners accept argv/stdout/stderr/env-style inputs and return exit status or error, while `main()` remains a tiny process shell (`ARCH-PURE`). The slice intentionally proves the helper-dispatch pattern without moving live zellij/nvim call sites or changing public `pair` behavior (`ARCH-PURPOSE`).

**Tech Stack:** Go standard library, existing `cmd/internal/dispatcher`, existing `cmd/internal/ctxmeter` and `cmd/internal/transcript`, existing scrollback renderer core, `go test`, `make`.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `ContextRunArgs` | `cmd/internal/contextcmd/contextcmd.go` | new |
| `ScrollbackRenderArgs` | `cmd/internal/scrollbackcmd/scrollbackcmd.go` | new |

**ContextRunArgs** — Parsed input for the context helper: tag, agent, home, and Pair data dir.
- **Relationships:** 1:1 with a `pair-context`/`pair-go context` invocation; consumed by the context runner.
- **DRY rationale:** Both legacy binary and dispatcher route need the same env/default resolution and argument validation.
- **Future extensions:** Can widen if more commands need shared Pair environment resolution.

**ScrollbackRenderArgs** — Parsed input for the renderer helper: raw capture path, events path, output path, and render flags.
- **Relationships:** 1:1 with a render invocation; maps directly onto existing render parameters.
- **DRY rationale:** Avoids parallel flag parsing between `pair-scrollback-render` and `pair-go scrollback-render`.
- **Future extensions:** Can become the command-facing shape for future `pair scrollback-render` after the public entrypoint switch.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ContextRunner` | `cmd/internal/contextcmd/contextcmd.go` | new | filesystem, env, transcript files, stdout |
| `ScrollbackRenderRunner` | `cmd/internal/scrollbackcmd/scrollbackcmd.go` | new | filesystem, flag parsing, stdout/stderr |
| `HelperDispatchRoutes` | `cmd/internal/dispatcher/dispatcher.go` | modified | in-process calls to selected helper runners |
| `PairGoMain` | `cmd/pair-go/main.go` | modified | process stdout/stderr exit handling |

**ContextRunner** — Shared command runner for `pair-context <tag> <agent>` behavior. It remains tolerant: missing config/transcript/input prints nothing and exits 0.
- **Injected into:** legacy `cmd/pair-context/main.go` and dispatcher context route.
- **Future extensions:** The title poller can continue using the legacy binary until #77/#78 moves call sites.

**ScrollbackRenderRunner** — Shared command runner for `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out`.
- **Injected into:** legacy `cmd/pair-scrollback-render/main.go` and dispatcher scrollback-render route.
- **Future extensions:** `bin/pair-scrollback-open`, `bin/pair-changelog-open`, and `nvim/scrollback.lua` can move to the dispatcher after the public entrypoint is Go-owned.

**HelperDispatchRoutes** — Dispatcher routes for `context` and `scrollback-render`.
- **Injected into:** `cmd/pair-go`.
- **Future extensions:** Later helper routes should follow the same runner extraction pattern, not duplicate command logic.

**PairGoMain** — Existing process shell that writes dispatcher results.
- **Injected into:** none.
- **Future extensions:** Eventually becomes the public `pair` entrypoint in #77, but not here.

---

## Chunk 1: Extract Context Runner

### Task 1: Make `pair-context` Reusable

**Files:**
- Create: `cmd/internal/contextcmd/contextcmd.go`
- Create: `cmd/internal/contextcmd/contextcmd_test.go`
- Modify: `cmd/pair-context/main.go`
- Modify: `cmd/pair-context/main_test.go`
- Modify: `Makefile.local`

- [x] **Step 1: Add failing runner tests**

Create `cmd/internal/contextcmd/contextcmd_test.go` with tests that call `Run(args []string, env Env, stdout io.Writer) int` directly:

```go
func TestRunClaude(t *testing.T) {
    // Arrange the same config/pane/transcript fixture as TestPairContext_Claude.
    // Call Run([]string{"T", "claude"}, env, &stdout).
    // Assert code == 0 and stdout == "398k\n".
}

func TestRunMissingConfigPrintsNothing(t *testing.T) {
    // Call Run with empty data dir.
    // Assert code == 0 and stdout == "".
}
```

- [x] **Step 2: Run the focused tests and confirm they fail**

Run: `go test ./cmd/internal/contextcmd -count=1`

Expected: FAIL because the package does not exist yet.

- [x] **Step 3: Extract the runner**

Move the reusable context behavior into `cmd/internal/contextcmd`. Keep `cmd/pair-context/main.go` as:

```go
func main() {
    os.Exit(contextcmd.Run(os.Args[1:], contextcmd.EnvFromOS(), os.Stdout))
}
```

The runner must:
- return 0 for missing args, matching the current tolerant behavior;
- resolve `PAIR_DATA_DIR` from env or `$XDG_DATA_HOME/pair` or `$HOME/.local/share/pair`;
- write the same humanized token count to the injected stdout;
- never call `os.Exit`.

- [x] **Step 4: Run the focused tests and existing package tests**

Run: `go test ./cmd/internal/contextcmd ./cmd/pair-context -count=1`

Expected: PASS.

- [x] **Step 5: Commit the context extraction**

Run:

```bash
git add cmd/internal/contextcmd cmd/pair-context Makefile.local
git commit -m "#76: extract pair-context runner"
```

---

## Chunk 2: Extract Scrollback Renderer Runner

### Task 2: Make `pair-scrollback-render` Reusable

**Files:**
- Create: `cmd/internal/scrollbackcmd/scrollbackcmd.go`
- Create: `cmd/internal/scrollbackcmd/scrollbackcmd_test.go`
- Modify: `cmd/pair-scrollback-render/main.go`
- Modify or create tests in: `cmd/pair-scrollback-render/*_test.go`
- Modify: `Makefile.local`

- [x] **Step 1: Add failing runner tests**

Create tests that call `scrollbackcmd.Run(args []string, stdout, stderr io.Writer) int`:

```go
func TestRunUsage(t *testing.T) {
    var stderr bytes.Buffer
    code := Run([]string{}, io.Discard, &stderr)
    // Assert code == 2 and usage is written to stderr.
}

func TestRunWritesOutput(t *testing.T) {
    // Use a tiny raw/events fixture compatible with existing renderer tests.
    // Call Run([]string{raw, events, out}, io.Discard, &stderr).
    // Assert code == 0 and out exists.
}
```

- [x] **Step 2: Run the focused tests and confirm they fail**

Run: `go test ./cmd/internal/scrollbackcmd -count=1`

Expected: FAIL because the package does not exist yet.

- [x] **Step 3: Extract the runner**

Move the renderer command wrapper into `cmd/internal/scrollbackcmd`. If `render(...)` cannot be imported from `package main`, move the rendering core into this internal package too and leave the legacy command as a tiny wrapper:

```go
func main() {
    os.Exit(scrollbackcmd.Run(os.Args[1:], os.Stdout, os.Stderr))
}
```

Use a local `flag.FlagSet` so dispatcher and tests can parse independently. Preserve current behavior:
- usage to stderr and exit 2 for wrong arity;
- `scrollback-render: <err>` to stderr and exit 1 for render errors;
- exit 0 for success;
- same defaults for `--plain`, `--max-lines`, and `--with-timestamps`.

- [x] **Step 4: Run focused and package tests**

Run: `go test ./cmd/internal/scrollbackcmd ./cmd/pair-scrollback-render -count=1`

Expected: PASS.

- [x] **Step 5: Commit the renderer extraction**

Run:

```bash
git add cmd/internal/scrollbackcmd cmd/pair-scrollback-render Makefile.local
git commit -m "#76: extract scrollback renderer runner"
```

---

## Chunk 3: Wire Dispatcher Routes

### Task 3: Route Selected Helpers Through `pair-go`

**Files:**
- Modify: `cmd/internal/dispatcher/dispatcher.go`
- Modify: `cmd/internal/dispatcher/dispatcher_test.go`
- Modify: `cmd/pair-go/main_test.go`
- Create: `cmd/pair-go/helper_equivalence_test.go`
- Modify: `Makefile.local`

- [x] **Step 1: Add failing dispatcher tests**

Add tests for:
- `Dispatch([]string{"context", "T", "claude"})` through a fake or temp fixture returns the same output shape as `contextcmd.Run`;
- `Dispatch([]string{"scrollback-render"})` returns renderer usage with exit 2;
- top-level help lists `context` and `scrollback-render` as implemented helper routes, not planned-only commands.

- [x] **Step 2: Run dispatcher tests and confirm failure**

Run: `go test ./cmd/internal/dispatcher -run 'TestDispatch(Context|Scrollback|Help)' -count=1`

Expected: FAIL because routes are not implemented yet.

- [x] **Step 3: Add failing process-level equivalence test**

Create `cmd/pair-go/helper_equivalence_test.go` with a context helper fixture that builds both commands and runs them against the same temp Pair data/transcript tree:

```go
func TestPairGoContextMatchesLegacyPairContext(t *testing.T) {
    // Build ./cmd/pair-context and ./cmd/pair-go into a temp bin dir.
    // Arrange config-T-claude.json, pane-T-claude.json, and a Claude transcript.
    // Run: pair-context T claude
    // Run: pair-go context T claude
    // Assert stdout, stderr, and exit code match exactly.
}
```

This is the representative compatibility proof required by the issue done-when. It should fail before the dispatcher route exists because `pair-go context` is still planned-only.

- [x] **Step 4: Run the equivalence test and confirm failure**

Run: `go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1`

Expected: FAIL because `pair-go context` returns planned-but-not-implemented.

- [x] **Step 5: Implement routes**

Update `Families()` statuses for selected helpers and add dispatcher branches:

```go
case "context":
    return dispatchContext(args[1:])
case "scrollback-render":
    return dispatchScrollbackRender(args[1:])
```

The dispatcher should continue returning `dispatcher.Result`. Helper runners should write into buffers so dispatcher can map stdout/stderr/exit code without adding a second process-result abstraction.

- [x] **Step 6: Run route and process tests**

Run:

```bash
go test ./cmd/internal/dispatcher ./cmd/pair-go ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render -count=1
make pair-context pair-scrollback-render pair-go
```

Expected: PASS. The `make` command is deliberately not `-B`; it verifies the updated dependency graph can rebuild normally after source changes.

- [x] **Step 7: Commit dispatcher wiring**

Run:

```bash
git add cmd/internal/dispatcher cmd/pair-go cmd/internal/contextcmd cmd/internal/scrollbackcmd cmd/pair-context cmd/pair-scrollback-render Makefile.local
git commit -m "#76: route selected helpers through pair-go"
```

---

## Chunk 4: Verify Legacy Compatibility And Docs

### Task 4: Verify Builds, Callers, And Atlas

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `atlas/go-migration-inventory.md`
- Modify: `workshop/issues/000076-go-helper-dispatch.md`
- Modify: `Makefile.local`

- [x] **Step 1: Verify legacy binaries still build**

Run:

```bash
make pair-context pair-scrollback-render pair-go
make -B pair-context pair-scrollback-render pair-go
```

Expected: PASS. The non-`-B` run verifies incremental prerequisites include `cmd/internal/contextcmd`, `cmd/internal/scrollbackcmd`, and dispatcher dependencies; the `-B` run remains the forced clean rebuild check.

- [x] **Step 2: Verify selected command equivalence**

Run focused commands against test fixtures or package tests:

```bash
go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1
go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1
```

Expected: PASS; the equivalence test demonstrates the legacy `pair-context` binary and `pair-go context` process path produce identical stdout/stderr/exit code on the same fixture.

- [x] **Step 3: Verify full Go test suite**

Run: `go test ./... -count=1`

Expected: PASS.

- [x] **Step 4: Verify no live call sites moved**

Run:

```bash
git diff -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollback-open bin/pair-changelog-open
```

Expected: empty diff, unless a test-only or documentation-only change was explicitly made.

- [x] **Step 5: Update atlas**

Update:
- `atlas/architecture.md` to say `pair-go context` and `pair-go scrollback-render` are implemented helper routes while public launcher and live zellij/nvim callers remain legacy.
- `atlas/go-migration-inventory.md` rows for `pair-context` and `pair-scrollback-render` to record dispatcher availability and preserved legacy names.

- [x] **Step 6: Update issue checklist and log**

Tick the #76 plan/done items that are complete and add a log entry with verification commands and `ARCH-*` notes.

- [x] **Step 7: Run final verification before close**

Run:

```bash
go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1
go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1
make pair-context pair-scrollback-render pair-go
make -B pair-context pair-scrollback-render pair-go
go test ./... -count=1
git diff -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollback-open bin/pair-changelog-open
rg -n "pair-go context|pair-go scrollback-render|helper dispatch" atlas/architecture.md atlas/go-migration-inventory.md
git diff --check
```

Expected: all tests/builds pass, caller diff empty, atlas grep finds the new helper-dispatch documentation, and whitespace check passes.

- [ ] **Step 8: Close through SDLC**

Run:

```bash
sdlc close --issue 76 --verified 'go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1; go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1; make pair-context pair-scrollback-render pair-go; make -B pair-context pair-scrollback-render pair-go; go test ./... -count=1; git diff live callers empty; rg atlas helper dispatch; git diff --check'
```

Expected: close gate runs the boundary review and reports SHIP or actionable findings.

exec
/bin/zsh -lc 'git diff --stat 9cc0f57521c3e44508adf3df006ffe3347faa265..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 Makefile.local                                     |   6 +-
 atlas/architecture.md                              |   8 +
 atlas/go-migration-inventory.md                    |  11 +-
 cmd/internal/contextcmd/contextcmd.go              |  78 ++++
 cmd/internal/contextcmd/contextcmd_test.go         |  59 +++
 cmd/internal/dispatcher/dispatcher.go              |  27 +-
 cmd/internal/dispatcher/dispatcher_test.go         |  73 ++++
 .../scrollbackcmd}/events_test.go                  |   2 +-
 .../scrollbackcmd}/render_test.go                  |   2 +-
 cmd/internal/scrollbackcmd/scrollbackcmd.go        | 413 +++++++++++++++++++++
 cmd/internal/scrollbackcmd/scrollbackcmd_test.go   |  48 +++
 .../scrollbackcmd}/serialize_row_test.go           |   2 +-
 .../scrollbackcmd}/timestamps_test.go              |   2 +-
 cmd/pair-context/main.go                           |  50 +--
 cmd/pair-go/helper_equivalence_test.go             |  93 +++++
 cmd/pair-scrollback-render/main.go                 | 402 +-------------------
 workshop/issues/000076-go-helper-dispatch.md       |  22 +-
 workshop/plans/000076-go-helper-dispatch-plan.md   |  48 +--
 18 files changed, 851 insertions(+), 495 deletions(-)

codex
The issue and plan mostly line up with the diff at a high level: shared runners were extracted and atlas was touched. I’m checking the actual runner surfaces and tests now, especially places where process behavior can drift when moving `flag` parsing and stdout/stderr through the dispatcher.
exec
/bin/zsh -lc "nl -ba cmd/internal/contextcmd/contextcmd.go | sed -n '1,160p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	// Package contextcmd implements the pair-context command body.
     2	package contextcmd
     3	
     4	import (
     5		"encoding/json"
     6		"fmt"
     7		"io"
     8		"os"
     9		"path/filepath"
    10	
    11		"github.com/xianxu/pair/cmd/internal/ctxmeter"
    12		"github.com/xianxu/pair/cmd/internal/transcript"
    13	)
    14	
    15	type Env struct {
    16		Home        string
    17		XDGDataHome string
    18		PairDataDir string
    19	}
    20	
    21	func EnvFromOS() Env {
    22		return Env{
    23			Home:        os.Getenv("HOME"),
    24			XDGDataHome: os.Getenv("XDG_DATA_HOME"),
    25			PairDataDir: os.Getenv("PAIR_DATA_DIR"),
    26		}
    27	}
    28	
    29	func Run(args []string, env Env, stdout io.Writer) int {
    30		if len(args) < 2 {
    31			return 0
    32		}
    33		tag, agent := args[0], args[1]
    34		dataDir := resolveDataDir(env)
    35		sid := transcript.SessionID(dataDir, tag, agent)
    36		if sid == "" {
    37			return 0
    38		}
    39		cwd := paneCwd(dataDir, tag, agent)
    40		path := transcript.Resolve(agent, sid, cwd, env.Home)
    41		if path == "" {
    42			return 0
    43		}
    44		f, err := os.Open(path)
    45		if err != nil {
    46			return 0
    47		}
    48		defer f.Close()
    49		if n, ok := ctxmeter.ContextTokens(agent, f); ok {
    50			fmt.Fprintln(stdout, ctxmeter.Humanize(n))
    51		}
    52		return 0
    53	}
    54	
    55	func resolveDataDir(env Env) string {
    56		if env.PairDataDir != "" {
    57			return env.PairDataDir
    58		}
    59		base := env.XDGDataHome
    60		if base == "" {
    61			base = filepath.Join(env.Home, ".local", "share")
    62		}
    63		return filepath.Join(base, "pair")
    64	}
    65	
    66	func paneCwd(dataDir, tag, agent string) string {
    67		b, err := os.ReadFile(filepath.Join(dataDir, "pane-"+tag+"-"+agent+".json"))
    68		if err != nil {
    69			return ""
    70		}
    71		var p struct {
    72			Cwd string `json:"cwd"`
    73		}
    74		if json.Unmarshal(b, &p) != nil {
    75			return ""
    76		}
    77		return p.Cwd
    78	}

exec
/bin/zsh -lc "nl -ba cmd/internal/dispatcher/dispatcher.go | sed -n '1,240p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package dispatcher
     2	
     3	import (
     4		"bytes"
     5		"fmt"
     6		"os"
     7		"strings"
     8		"time"
     9	
    10		"github.com/xianxu/pair/cmd/internal/contextcmd"
    11		"github.com/xianxu/pair/cmd/internal/launcher"
    12		"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
    13	)
    14	
    15	const programName = "pair-go"
    16	
    17	// CommandFamily names a future Pair CLI surface without claiming it works yet.
    18	type CommandFamily struct {
    19		Name    string
    20		Summary string
    21		Status  string
    22	}
    23	
    24	// Result is the process-facing outcome of a pure dispatch decision.
    25	type Result struct {
    26		Stdout   string
    27		Stderr   string
    28		ExitCode int
    29	}
    30	
    31	// Families returns the planned command families for the Go dispatcher.
    32	func Families() []CommandFamily {
    33		return []CommandFamily{
    34			{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "prototype"},
    35			{Name: "context", Summary: "agent pane context meter", Status: "implemented"},
    36			{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "implemented"},
    37			{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
    38			{Name: "slug", Summary: "session orientation slug generation", Status: "planned"},
    39			{Name: "changelog", Summary: "TTY transcript to distilled change log", Status: "planned"},
    40			{Name: "continuation", Summary: "continuation datatype writer", Status: "planned"},
    41			{Name: "scribe", Summary: "PTY logging wrapper", Status: "planned"},
    42		}
    43	}
    44	
    45	// Dispatch parses argv and returns the skeleton dispatch result.
    46	func Dispatch(args []string) Result {
    47		if len(args) == 0 {
    48			return Result{Stdout: Help(programName), ExitCode: 0}
    49		}
    50	
    51		switch args[0] {
    52		case "help", "--help", "-h":
    53			return Result{Stdout: Help(programName), ExitCode: 0}
    54		case "version", "--version":
    55			return Result{
    56				Stdout:   "pair-go dispatcher skeleton\npublic launcher: bin/pair\n",
    57				ExitCode: 0,
    58			}
    59		case "launch":
    60			return DispatchWithLauncherRuntime(args, osLauncherRuntime())
    61		case "context":
    62			return dispatchContext(args[1:])
    63		case "scrollback-render":
    64			return dispatchScrollbackRender(args[1:])
    65		}
    66	
    67		if family, ok := familyByName(args[0]); ok {
    68			return Result{
    69				Stderr:   fmt.Sprintf("%s: %s is planned but not implemented in this skeleton; run %s help\n", programName, family.Name, programName),
    70				ExitCode: 2,
    71			}
    72		}
    73	
    74		return Result{
    75			Stderr:   fmt.Sprintf("%s: unknown command %q; run %s help\n", programName, args[0], programName),
    76			ExitCode: 2,
    77		}
    78	}
    79	
    80	func dispatchContext(args []string) Result {
    81		var stdout bytes.Buffer
    82		code := contextcmd.Run(args, contextcmd.EnvFromOS(), &stdout)
    83		return Result{Stdout: stdout.String(), ExitCode: code}
    84	}
    85	
    86	func dispatchScrollbackRender(args []string) Result {
    87		var stdout, stderr bytes.Buffer
    88		code := scrollbackcmd.Run(args, &stdout, &stderr)
    89		return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}
    90	}
    91	
    92	type LauncherRuntime struct {
    93		Env      launcher.Env
    94		Sessions launcher.SessionSource
    95		History  launcher.HistoricalScanner
    96	}
    97	
    98	type StaticSessions struct {
    99		Sessions []launcher.Session
   100		Err      error
   101	}
   102	
   103	func (s StaticSessions) Snapshot() ([]launcher.Session, error) {
   104		if s.Err != nil {
   105			return nil, s.Err
   106		}
   107		return s.Sessions, nil
   108	}
   109	
   110	type StaticHistory struct {
   111		Tags []launcher.HistoricalTag
   112		Err  error
   113	}
   114	
   115	func (h StaticHistory) Scan(_ string, _ time.Time) ([]launcher.HistoricalTag, error) {
   116		if h.Err != nil {
   117			return nil, h.Err
   118		}
   119		return h.Tags, nil
   120	}
   121	
   122	func DispatchWithLauncherRuntime(args []string, rt LauncherRuntime) Result {
   123		launchArgs := []string(nil)
   124		if len(args) > 1 {
   125			launchArgs = args[1:]
   126		}
   127		if len(launchArgs) > 0 && (launchArgs[0] == "help" || launchArgs[0] == "--help" || launchArgs[0] == "-h") {
   128			return Result{Stdout: LaunchHelp(programName), ExitCode: 0}
   129		}
   130		outcome, err := launcher.Run(launchArgs, rt.Env, rt.Sessions, rt.History)
   131		if err != nil {
   132			return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
   133		}
   134		decision := outcome.Decision
   135		return Result{
   136			Stderr: fmt.Sprintf(
   137				"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
   138				decision.Action,
   139				decision.Tag,
   140				decision.SessionName,
   141			),
   142			ExitCode: 3,
   143		}
   144	}
   145	
   146	func LaunchHelp(program string) string {
   147		return fmt.Sprintf(`Usage: %s launch [agent] [-- agent-args...]
   148	       %s launch resume <tag>
   149	
   150	Guarded decision-phase prototype. Public sessions still start through bin/pair.
   151	This command parses launch inputs and computes the create/attach/picker decision,
   152	then stops before invoking zellij.
   153	`, program, program)
   154	}
   155	
   156	func LauncherEnv(home, xdgDataHome, cwd string) launcher.Env {
   157		return launcher.Env{
   158			Home:     home,
   159			XDGData:  xdgDataHome,
   160			Cwd:      cwd,
   161			Now:      time.Now(),
   162			HistoryD: 14,
   163		}
   164	}
   165	
   166	func osLauncherRuntime() LauncherRuntime {
   167		home := os.Getenv("HOME")
   168		xdg := os.Getenv("XDG_DATA_HOME")
   169		cwd, _ := os.Getwd()
   170		env := LauncherEnv(home, xdg, cwd)
   171		dataDir := launcher.ResolveDataDir(home, xdg)
   172		return LauncherRuntime{
   173			Env:      env,
   174			Sessions: launcher.ZellijSource{},
   175			History:  launcher.HistorySource{DataDir: dataDir},
   176		}
   177	}
   178	
   179	// Help renders the development-only dispatcher usage text.
   180	func Help(program string) string {
   181		var b strings.Builder
   182		fmt.Fprintf(&b, "Usage: %s <command> [args]\n\n", program)
   183		b.WriteString("Development dispatcher skeleton. Public sessions still start through bin/pair.\n\n")
   184		b.WriteString("Implemented prototype commands:\n")
   185		for _, family := range Families() {
   186			if family.Status == "prototype" {
   187				fmt.Fprintf(&b, "  %-17s %s (prototype; decision-phase only)\n", family.Name, family.Summary)
   188			} else if family.Status == "implemented" {
   189				fmt.Fprintf(&b, "  %-17s %s (implemented helper route)\n", family.Name, family.Summary)
   190			}
   191		}
   192		b.WriteString("\nPlanned command families (not implemented in this skeleton):\n")
   193		for _, family := range Families() {
   194			if family.Status == "planned" {
   195				fmt.Fprintf(&b, "  %-17s %s (%s; not implemented in this skeleton)\n", family.Name, family.Summary, family.Status)
   196			}
   197		}
   198		b.WriteString("\nSupported skeleton commands:\n")
   199		b.WriteString("  help              show this help\n")
   200		b.WriteString("  version           show dispatcher skeleton metadata\n")
   201		return b.String()
   202	}
   203	
   204	func familyByName(name string) (CommandFamily, bool) {
   205		for _, family := range Families() {
   206			if family.Name == name {
   207				return family, true
   208			}
   209		}
   210		return CommandFamily{}, false
   211	}

exec
/bin/zsh -lc "nl -ba cmd/internal/scrollbackcmd/scrollbackcmd.go | sed -n '1,470p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	// scrollback-render — replay a pair-wrap raw capture through a VT100
     2	// emulator and write one ANSI-styled output line per logical scrollback row.
     3	// Drop-in replacement for the Python+pyte renderer at bin/pair-scrollback-render.
     4	//
     5	// Why Go: pyte's HistoryScreen dispatches every method call through an
     6	// __getattribute__ override that ran ~19M times for a 3 MB raw input —
     7	// ~95% of wall time. Even after the CaptureScreen patch (3.6x speedup),
     8	// the Python interpreter + pyte vendoring add startup cost and a private
     9	// venv that the brew formula has to manage. A static Go binary using
    10	// charmbracelet/x/vt replays the same stream with no runtime deps and
    11	// stays within the pair repo's existing cmd/ layout.
    12	//
    13	// Pipeline:
    14	//
    15	//	raw bytes (.raw)              → emulator.Write(...) in segments
    16	//	resize events (.events.jsonl) → segment boundaries with new (cols,rows)
    17	//	final emulator state          → scrollback lines + visible buffer
    18	//	each row                      → SGR-decorated text line written to out
    19	//
    20	// CLI is identical to the Python version so bin/pair-scrollback-open can
    21	// invoke either:
    22	//
    23	//	scrollback-render <raw> <events.jsonl> <out.ansi>
    24	package scrollbackcmd
    25	
    26	import (
    27		"encoding/json"
    28		"flag"
    29		"fmt"
    30		"io"
    31		"math"
    32		"os"
    33		"strconv"
    34		"strings"
    35		"time"
    36	
    37		uv "github.com/charmbracelet/ultraviolet"
    38		"github.com/charmbracelet/x/vt"
    39	)
    40	
    41	// historyRows caps scrolled-out rows retained for the viewer. Matched
    42	// to zellij/config.kdl's `scroll_buffer_size 2000` so PageUp inside the
    43	// agent pane and Alt+/ both reach back the same distance — otherwise
    44	// the viewer would surface lines that zellij no longer has, leaving
    45	// the scroll-overlay logic in pair-scrollback-open unable to align
    46	// against the agent pane's frame.
    47	const historyRows = 2_000
    48	
    49	const (
    50		defaultCols = 80
    51		defaultRows = 24
    52	)
    53	
    54	type scrollbackEvent struct {
    55		Type   string `json:"type"`
    56		Offset int64  `json:"offset"`
    57		Cols   int    `json:"cols"`
    58		Rows   int    `json:"rows"`
    59		Ts     string `json:"ts,omitempty"` // RFC3339 wall-clock for "time" events (#59)
    60	}
    61	
    62	// dateOf extracts the YYYY-MM-DD day from an RFC3339 timestamp; "" on a
    63	// malformed value so a corrupt time event degrades to undated, never panics (#59).
    64	func dateOf(ts string) string {
    65		t, err := time.Parse(time.RFC3339, ts)
    66		if err != nil {
    67			return ""
    68		}
    69		return t.Format("2006-01-02")
    70	}
    71	
    72	// parseEvents reads the sidecar JSONL. Empty / missing file → empty slice.
    73	// Malformed lines are skipped so a corrupted tail doesn't abort the render —
    74	// imperfect width tracking beats an unusable viewer.
    75	func parseEvents(path string) ([]scrollbackEvent, error) {
    76		data, err := os.ReadFile(path)
    77		if err != nil {
    78			if os.IsNotExist(err) {
    79				return nil, nil
    80			}
    81			return nil, err
    82		}
    83		var out []scrollbackEvent
    84		for _, line := range strings.Split(string(data), "\n") {
    85			line = strings.TrimSpace(line)
    86			if line == "" {
    87				continue
    88			}
    89			var e scrollbackEvent
    90			if err := json.Unmarshal([]byte(line), &e); err != nil {
    91				continue
    92			}
    93			// Keep both known types: resize boundaries AND time stamps (#59).
    94			// Consumers filter by Type at their use sites.
    95			if e.Type == "resize" || e.Type == "time" {
    96				out = append(out, e)
    97			}
    98		}
    99		return out, nil
   100	}
   101	
   102	// initialSize pulls (cols, rows) from the first usable resize event, or
   103	// falls back to 80x24 if the sidecar is empty or the first entry is
   104	// malformed. pair-wrap always emits an initial resize at offset 0, so the
   105	// fallback only fires on a truncated file.
   106	func initialSize(events []scrollbackEvent) (int, int) {
   107		for _, e := range events {
   108			if e.Type == "resize" && e.Cols > 0 && e.Rows > 0 {
   109				return e.Cols, e.Rows
   110			}
   111		}
   112		return defaultCols, defaultRows
   113	}
   114	
   115	// dateMark records the emulator's scrollback length at a "time" event's byte
   116	// offset → the day that applies to committed lines from that index onward (#59).
   117	// Built during the feed (the only place that knows both byte offsets and the
   118	// rendered line count); consumed by the pure interleaveDateMarkers.
   119	type dateMark struct {
   120		line int
   121		date string
   122	}
   123	
   124	// feedSegments writes raw into the emulator as a single offset-ordered walk over
   125	// ALL sidecar events: write everything up to event.Offset, then act — Resize on a
   126	// resize event, or snapshot Scrollback().Len() on a time event. Returns the time
   127	// snapshots (empty unless time events are present). The caller already set the
   128	// initial size via initialSize; re-applying the offset-0 resize here is a harmless
   129	// no-op (resize to the current dimensions). Walking all events — rather than
   130	// events[1:] — means a time event in any position (incl. first) is captured, and
   131	// an empty events slice is handled without an out-of-range slice (#59).
   132	//
   133	// Clamping Offset to len(raw) defends against a corrupted sidecar that records
   134	// an offset beyond EOF (saw this once with a half-written events file after a
   135	// hard kill); without clamping we'd panic on the slice.
   136	func feedSegments(em *vt.Emulator, raw []byte, events []scrollbackEvent) []dateMark {
   137		var cursor int64
   138		var marks []dateMark
   139		for _, e := range events {
   140			off := e.Offset
   141			if off > int64(len(raw)) {
   142				off = int64(len(raw))
   143			}
   144			if off > cursor {
   145				_, _ = em.Write(raw[cursor:off])
   146				cursor = off
   147			}
   148			switch e.Type {
   149			case "resize":
   150				em.Resize(e.Cols, e.Rows)
   151			case "time":
   152				if d := dateOf(e.Ts); d != "" {
   153					marks = append(marks, dateMark{line: em.Scrollback().Len(), date: d})
   154				}
   155			}
   156		}
   157		if cursor < int64(len(raw)) {
   158			_, _ = em.Write(raw[cursor:])
   159		}
   160		return marks
   161	}
   162	
   163	// tsMarkerLine is the wire format the distiller parses (#59). MUST stay in sync
   164	// with tsMarkerRe in cmd/pair-changelog/distill.go — the contract is pinned by
   165	// the render→clean→distill e2e test cmd/pair-changelog/e2e_test.go
   166	// (TestEndToEndMarkerSurvival), which feeds real time events through both binaries.
   167	func tsMarkerLine(date string) string {
   168		return "⟦pair:ts " + date + "⟧"
   169	}
   170	
   171	// interleaveDateMarkers inserts a tsMarkerLine immediately before the first line
   172	// of each new date run. marks are (scrollback-line-index, date) snapshots in
   173	// ascending index; a marker is emitted only when the applicable date *changes*
   174	// from the running date (consecutive same-date marks collapse). Lines before the
   175	// first mark stay undated; marks past len(lines) are ignored. Pure (#59).
   176	func interleaveDateMarkers(lines []string, marks []dateMark) []string {
   177		if len(marks) == 0 {
   178			return lines
   179		}
   180		out := make([]string, 0, len(lines)+len(marks))
   181		mi := 0
   182		prevDate := ""
   183		for i := 0; i < len(lines); i++ {
   184			curDate := prevDate
   185			for mi < len(marks) && marks[mi].line <= i {
   186				curDate = marks[mi].date
   187				mi++
   188			}
   189			if curDate != "" && curDate != prevDate {
   190				out = append(out, tsMarkerLine(curDate))
   191				prevDate = curDate
   192			}
   193			out = append(out, lines[i])
   194		}
   195		return out
   196	}
   197	
   198	// serializeRow flattens one row into ANSI-styled text. Trims trailing
   199	// default-styled blanks (so the viewer doesn't scroll past pad), uses
   200	// Style.Diff so we only emit SGR codes when the style actually changes
   201	// between cells, and terminates with \x1b[0m to keep the row's last
   202	// style from bleeding into the next line if a viewer concatenates without
   203	// resetting between lines.
   204	//
   205	// A non-default background space is treated as visible content (e.g.
   206	// inverse-video padding). Matches what the Python renderer does.
   207	//
   208	// In plain mode (plain=true) no SGR is emitted at all: the row is just its
   209	// visible content, trimmed to the last non-blank-*content* cell — a cell that
   210	// is "visible" only via a non-default background (inverse-video padding, box
   211	// fill) is NOT emitted in plain mode, so it must not extend the row, or a
   212	// trailing bordered region would become space-padding toward terminal width.
   213	func serializeRow(line uv.Line, plain bool) string {
   214		last := -1
   215		for i := range line {
   216			c := &line[i]
   217			// Continuation cells of a preceding wide grapheme are stored as
   218			// zero-value Cell{} per the ultraviolet convention (Width=0,
   219			// Content=""). They don't extend the visible row and must not
   220			// emit anything in the loop below.
   221			if c.IsZero() {
   222				continue
   223			}
   224			content := c.Content
   225			if content != "" && content != " " {
   226				last = i
   227			} else if !plain && c.Style.Bg != nil {
   228				last = i
   229			}
   230		}
   231		if last < 0 {
   232			return ""
   233		}
   234		var b strings.Builder
   235		var prev uv.Style // zero value = default; Diff vs zero emits a reset
   236		first := true
   237		for i := 0; i <= last; i++ {
   238			c := &line[i]
   239			// Skip wide-grapheme continuation cells — the wide cell already
   240			// emitted its full glyph; emitting anything here adds a phantom
   241			// space after every emoji.
   242			if c.IsZero() {
   243				continue
   244			}
   245			if !plain && (first || !c.Style.Equal(&prev)) {
   246				b.WriteString(c.Style.Diff(&prev))
   247				prev = c.Style
   248				first = false
   249			}
   250			if c.Content == "" {
   251				b.WriteByte(' ')
   252			} else {
   253				b.WriteString(c.Content)
   254			}
   255		}
   256		if !plain {
   257			b.WriteString("\x1b[0m")
   258		}
   259		return b.String()
   260	}
   261	
   262	// resolveMax maps a --max-lines value to a scrollback cap. <=0 means
   263	// "uncapped" — a continuation wants the whole session, not the viewer's
   264	// 2000-row window. Represented as a large sentinel; .raw is per-run
   265	// O_TRUNC'd, so the practical bound is the run length.
   266	func resolveMax(n int) int {
   267		if n <= 0 {
   268			return math.MaxInt32
   269		}
   270		return n
   271	}
   272	
   273	// visibleRow materializes row y of the live screen as a uv.Line. The
   274	// emulator exposes cells one at a time via CellAt(x,y); there's no
   275	// "give me the whole row" accessor. A missing cell (CellAt returns nil)
   276	// becomes a zero-value Cell, which serializeRow treats as a blank.
   277	func visibleRow(em *vt.Emulator, y, width int) uv.Line {
   278		row := make(uv.Line, width)
   279		for x := 0; x < width; x++ {
   280			if c := em.CellAt(x, y); c != nil {
   281				row[x] = *c
   282			}
   283		}
   284		return row
   285	}
   286	
   287	func render(rawPath, eventsPath, outPath string, plain bool, maxLines int, withTimestamps bool) error {
   288		events, err := parseEvents(eventsPath)
   289		if err != nil {
   290			return fmt.Errorf("parse events: %w", err)
   291		}
   292		cols, rows := initialSize(events)
   293		em := vt.NewEmulator(cols, rows)
   294		em.Scrollback().SetMaxLines(resolveMax(maxLines))
   295	
   296		// Drain the emulator's input pipe in the background. CSI status
   297		// queries (DSR, Device Attributes, etc.) in the captured stream
   298		// trigger handlers that *write a reply back* into this pipe — in a
   299		// real terminal those bytes go to the controlling app. Offscreen
   300		// replay has no reader, so the handler's WriteString blocks
   301		// forever and deadlocks the Write goroutine. Discarding the bytes
   302		// preserves emulation correctness; we never act on the replies.
   303		//
   304		// Wait for the drainer to actually exit before letting em.Close()
   305		// run, otherwise Close races with the drainer's still-pending
   306		// Read() (race detector catches it; in production the window is
   307		// usually harmless but it's a real ordering bug).
   308		drainDone := make(chan struct{})
   309		go func() {
   310			defer close(drainDone)
   311			_, _ = io.Copy(io.Discard, em)
   312		}()
   313		defer func() {
   314			em.Close()
   315			<-drainDone
   316		}()
   317	
   318		raw, err := os.ReadFile(rawPath)
   319		if err != nil {
   320			return fmt.Errorf("read raw: %w", err)
   321		}
   322		marks := feedSegments(em, raw, events)
   323	
   324		// Scrollback lines (oldest → newest), then visible buffer top → bottom.
   325		// Visible buffer iterates by row index rather than dropping trailing
   326		// blank rows: an agent that cleared and paused mid-redraw would shift
   327		// every subsequent line number otherwise, and `:880` should still land
   328		// where zellij showed line 880.
   329		sb := em.Scrollback()
   330		viewportTop := sb.Len() + 1 // 1-indexed line where the visible buffer starts
   331		out := make([]string, 0, sb.Len()+em.Height())
   332		for i := 0; i < sb.Len(); i++ {
   333			out = append(out, serializeRow(sb.Line(i), plain))
   334		}
   335		w := em.Width()
   336		for y := 0; y < em.Height(); y++ {
   337			out = append(out, serializeRow(visibleRow(em, y, w), plain))
   338		}
   339		// Trim trailing all-blank lines: a half-empty visible buffer otherwise
   340		// leaves a tail of empties at EOF.
   341		for len(out) > 0 && out[len(out)-1] == "" {
   342			out = out[:len(out)-1]
   343		}
   344	
   345		// Change-log path only: interleave day markers from the time-event snapshots
   346		// so the distiller can date entries by real change-time (#59). Done after the
   347		// trailing-blank trim so a marker never dangles past content. The scrollback
   348		// viewer never sets this flag → its render is byte-identical to before.
   349		if withTimestamps {
   350			out = interleaveDateMarkers(out, marks)
   351		}
   352	
   353		// Write the viewport sidecar *first*, then atomically rename the
   354		// .ansi into place. Order matters: scrollback.lua's BufReadPost
   355		// opens the .ansi and immediately reads the sidecar — flipping the
   356		// .ansi last guarantees the sidecar is up-to-date by the time
   357		// nvim sees the new content. Sidecar is best-effort: on write
   358		// failure, scrollback.lua falls back to its prior bottom-alignment.
   359		// The viewport sidecar positions the Alt+/ nvim viewer; it's meaningless
   360		// for the plain projection (a continuation distills the text, not a
   361		// scroll position), so skip it and don't litter a stray <out>.viewport.
   362		if !plain {
   363			viewportPath := strings.TrimSuffix(outPath, ".ansi") + ".viewport"
   364			_ = os.WriteFile(viewportPath, []byte(strconv.Itoa(viewportTop)+"\n"), 0o644)
   365		}
   366	
   367		// Atomic write so a double-tap Alt+/ can't race truncate-then-write
   368		// on the same path. Reader sees either the old complete file or the
   369		// new complete file, never a half-written one.
   370		tmp := outPath + ".tmp"
   371		f, err := os.Create(tmp)
   372		if err != nil {
   373			return err
   374		}
   375		body := strings.Join(out, "\n")
   376		if len(out) > 0 {
   377			body += "\n"
   378		}
   379		if _, err := f.WriteString(body); err != nil {
   380			f.Close()
   381			os.Remove(tmp)
   382			return err
   383		}
   384		if err := f.Close(); err != nil {
   385			os.Remove(tmp)
   386			return err
   387		}
   388		return os.Rename(tmp, outPath)
   389	}
   390	
   391	func Run(argv []string, stdout, stderr io.Writer) int {
   392		fs := flag.NewFlagSet("pair-scrollback-render", flag.ContinueOnError)
   393		fs.SetOutput(stderr)
   394		fs.Usage = func() {
   395			fmt.Fprintf(stderr, "usage: pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] <raw> <events.jsonl> <out>\n")
   396		}
   397		plain := fs.Bool("plain", false, "emit plain text (no SGR) for distillation")
   398		maxLines := fs.Int("max-lines", historyRows, "scrollback history rows retained; <=0 = uncapped")
   399		withTimestamps := fs.Bool("with-timestamps", false, "interleave ⟦pair:ts DATE⟧ day markers from time events (for the change log; #59)")
   400		if err := fs.Parse(argv); err != nil {
   401			return 2
   402		}
   403		args := fs.Args()
   404		if len(args) != 3 {
   405			fs.Usage()
   406			return 2
   407		}
   408		if err := render(args[0], args[1], args[2], *plain, *maxLines, *withTimestamps); err != nil {
   409			fmt.Fprintf(stderr, "scrollback-render: %v\n", err)
   410			return 1
   411		}
   412		return 0
   413	}

exec
/bin/zsh -lc "nl -ba cmd/pair-go/main.go | sed -n '1,160p'" in /Users/xianxu/workspace/pair
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
    18		return writeResult(res, stdout, stderr)
    19	}
    20	
    21	func runWithLauncherRuntime(args []string, stdout, stderr io.Writer, rt dispatcher.LauncherRuntime) int {
    22		res := dispatcher.DispatchWithLauncherRuntime(args, rt)
    23		return writeResult(res, stdout, stderr)
    24	}
    25	
    26	func writeResult(res dispatcher.Result, stdout, stderr io.Writer) int {
    27		if res.Stdout != "" {
    28			_, _ = io.WriteString(stdout, res.Stdout)
    29		}
    30		if res.Stderr != "" {
    31			_, _ = io.WriteString(stderr, res.Stderr)
    32		}
    33		return res.ExitCode
    34	}

codex
One contract issue is emerging: the plan’s Core Concepts table names pure entities that the implementation never creates, while the code instead exposes simpler `Run`/`Env` surfaces. I’m still checking whether that is only documentation drift or whether there is also runtime drift in the helper routes.
exec
/bin/zsh -lc 'rg -n "ContextRunArgs|ScrollbackRenderArgs|ContextRunner|ScrollbackRenderRunner|HelperDispatchRoutes|PairGoMain"' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
workshop/plans/000076-go-helper-dispatch-plan.md:19:| `ContextRunArgs` | `cmd/internal/contextcmd/contextcmd.go` | new |
workshop/plans/000076-go-helper-dispatch-plan.md:20:| `ScrollbackRenderArgs` | `cmd/internal/scrollbackcmd/scrollbackcmd.go` | new |
workshop/plans/000076-go-helper-dispatch-plan.md:22:**ContextRunArgs** — Parsed input for the context helper: tag, agent, home, and Pair data dir.
workshop/plans/000076-go-helper-dispatch-plan.md:27:**ScrollbackRenderArgs** — Parsed input for the renderer helper: raw capture path, events path, output path, and render flags.
workshop/plans/000076-go-helper-dispatch-plan.md:36:| `ContextRunner` | `cmd/internal/contextcmd/contextcmd.go` | new | filesystem, env, transcript files, stdout |
workshop/plans/000076-go-helper-dispatch-plan.md:37:| `ScrollbackRenderRunner` | `cmd/internal/scrollbackcmd/scrollbackcmd.go` | new | filesystem, flag parsing, stdout/stderr |
workshop/plans/000076-go-helper-dispatch-plan.md:38:| `HelperDispatchRoutes` | `cmd/internal/dispatcher/dispatcher.go` | modified | in-process calls to selected helper runners |
workshop/plans/000076-go-helper-dispatch-plan.md:39:| `PairGoMain` | `cmd/pair-go/main.go` | modified | process stdout/stderr exit handling |
workshop/plans/000076-go-helper-dispatch-plan.md:41:**ContextRunner** — Shared command runner for `pair-context <tag> <agent>` behavior. It remains tolerant: missing config/transcript/input prints nothing and exits 0.
workshop/plans/000076-go-helper-dispatch-plan.md:45:**ScrollbackRenderRunner** — Shared command runner for `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out`.
workshop/plans/000076-go-helper-dispatch-plan.md:49:**HelperDispatchRoutes** — Dispatcher routes for `context` and `scrollback-render`.
workshop/plans/000076-go-helper-dispatch-plan.md:53:**PairGoMain** — Existing process shell that writes dispatcher results.
workshop/history/000075-go-launcher-prototype-close-review.md:4725:| `PairGoMain` | `cmd/pair-go/main.go` | modified | process stdout/stderr exit handling |
workshop/history/000075-go-launcher-prototype-close-review.md:4740:**PairGoMain** — No business logic; writes dispatcher-returned streams and exits.
workshop/history/000075-go-launcher-prototype-close-review.md:5417:workshop/plans/000075-go-launcher-prototype-plan.md:74:**PairGoMain** — No business logic; writes dispatcher-returned streams and exits.
workshop/history/000075-go-launcher-prototype-close-review.md:5532:    59	| `PairGoMain` | `cmd/pair-go/main.go` | modified | process stdout/stderr exit handling |
workshop/history/000075-go-launcher-prototype-close-review.md:8528:    59	| `PairGoMain` | `cmd/pair-go/main.go` | modified | process stdout/stderr exit handling |
workshop/history/000075-go-launcher-prototype-close-review.md:8543:    74	**PairGoMain** — No business logic; writes dispatcher-returned streams and exits.
workshop/history/000075-go-launcher-prototype-plan.md:59:| `PairGoMain` | `cmd/pair-go/main.go` | modified | process stdout/stderr exit handling |
workshop/history/000075-go-launcher-prototype-plan.md:74:**PairGoMain** — No business logic; writes dispatcher-returned streams and exits.

exec
/bin/zsh -lc 'git diff --name-status 9cc0f57521c3e44508adf3df006ffe3347faa265..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
M	Makefile.local
M	atlas/architecture.md
M	atlas/go-migration-inventory.md
A	cmd/internal/contextcmd/contextcmd.go
A	cmd/internal/contextcmd/contextcmd_test.go
M	cmd/internal/dispatcher/dispatcher.go
M	cmd/internal/dispatcher/dispatcher_test.go
R099	cmd/pair-scrollback-render/events_test.go	cmd/internal/scrollbackcmd/events_test.go
R099	cmd/pair-scrollback-render/render_test.go	cmd/internal/scrollbackcmd/render_test.go
A	cmd/internal/scrollbackcmd/scrollbackcmd.go
A	cmd/internal/scrollbackcmd/scrollbackcmd_test.go
R099	cmd/pair-scrollback-render/serialize_row_test.go	cmd/internal/scrollbackcmd/serialize_row_test.go
R099	cmd/pair-scrollback-render/timestamps_test.go	cmd/internal/scrollbackcmd/timestamps_test.go
M	cmd/pair-context/main.go
A	cmd/pair-go/helper_equivalence_test.go
M	cmd/pair-scrollback-render/main.go
M	workshop/issues/000076-go-helper-dispatch.md
M	workshop/plans/000076-go-helper-dispatch-plan.md

exec
/bin/zsh -lc "nl -ba cmd/pair-go/helper_equivalence_test.go | sed -n '1,140p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package main
     2	
     3	import (
     4		"bytes"
     5		"os"
     6		"os/exec"
     7		"path/filepath"
     8		"strings"
     9		"testing"
    10	)
    11	
    12	func TestPairGoContextMatchesLegacyPairContext(t *testing.T) {
    13		bin := t.TempDir()
    14		pairContext := filepath.Join(bin, "pair-context")
    15		pairGo := filepath.Join(bin, "pair-go")
    16		buildCommand(t, pairContext, "../pair-context")
    17		buildCommand(t, pairGo, ".")
    18	
    19		home, data := writeContextFixture(t)
    20		env := append(os.Environ(), "HOME="+home, "PAIR_DATA_DIR="+data)
    21	
    22		legacy := runCommand(t, env, pairContext, "T", "claude")
    23		dispatch := runCommand(t, env, pairGo, "context", "T", "claude")
    24		if dispatch.code != legacy.code || dispatch.stdout != legacy.stdout || dispatch.stderr != legacy.stderr {
    25			t.Fatalf("pair-go context mismatch\nlegacy:   code=%d stdout=%q stderr=%q\ndispatch: code=%d stdout=%q stderr=%q",
    26				legacy.code, legacy.stdout, legacy.stderr,
    27				dispatch.code, dispatch.stdout, dispatch.stderr)
    28		}
    29	}
    30	
    31	func buildCommand(t *testing.T, out, pkg string) {
    32		t.Helper()
    33		cmd := exec.Command("go", "build", "-o", out, pkg)
    34		if body, err := cmd.CombinedOutput(); err != nil {
    35			t.Fatalf("build %s: %v\n%s", pkg, err, string(body))
    36		}
    37	}
    38	
    39	type commandResult struct {
    40		code   int
    41		stdout string
    42		stderr string
    43	}
    44	
    45	func runCommand(t *testing.T, env []string, name string, args ...string) commandResult {
    46		t.Helper()
    47		cmd := exec.Command(name, args...)
    48		cmd.Env = env
    49		var stdout, stderr bytes.Buffer
    50		cmd.Stdout = &stdout
    51		cmd.Stderr = &stderr
    52		err := cmd.Run()
    53		code := 0
    54		if err != nil {
    55			exit, ok := err.(*exec.ExitError)
    56			if !ok {
    57				t.Fatalf("run %s: %v", name, err)
    58			}
    59			code = exit.ExitCode()
    60		}
    61		return commandResult{code: code, stdout: stdout.String(), stderr: stderr.String()}
    62	}
    63	
    64	func writeContextFixture(t *testing.T) (home, data string) {
    65		t.Helper()
    66		home = t.TempDir()
    67		data = filepath.Join(home, "data")
    68		cwd := filepath.Join(home, "repo")
    69		enc := strings.NewReplacer(".", "-", "/", "-").Replace(cwd)
    70		proj := filepath.Join(home, ".claude", "projects", enc)
    71		mustMkdir(t, data)
    72		mustMkdir(t, cwd)
    73		mustMkdir(t, proj)
    74		mustWrite(t, filepath.Join(data, "config-T-claude.json"), `{"session_id":"sid1"}`)
    75		mustWrite(t, filepath.Join(data, "pane-T-claude.json"), `{"pane_id":"7","cwd":"`+cwd+`","cwd_display":"~/repo"}`)
    76		mustWrite(t, filepath.Join(proj, "sid1.jsonl"),
    77			`{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":397556,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`)
    78		return home, data
    79	}
    80	
    81	func mustMkdir(t *testing.T, d string) {
    82		t.Helper()
    83		if err := os.MkdirAll(d, 0o755); err != nil {
    84			t.Fatal(err)
    85		}
    86	}
    87	
    88	func mustWrite(t *testing.T, p, s string) {
    89		t.Helper()
    90		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
    91			t.Fatal(err)
    92		}
    93	}

exec
/bin/zsh -lc "nl -ba cmd/internal/contextcmd/contextcmd_test.go | sed -n '1,140p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package contextcmd
     2	
     3	import (
     4		"bytes"
     5		"os"
     6		"path/filepath"
     7		"strings"
     8		"testing"
     9	)
    10	
    11	func TestRunClaude(t *testing.T) {
    12		home := t.TempDir()
    13		data := filepath.Join(home, "data")
    14		cwd := filepath.Join(home, "repo")
    15		enc := strings.NewReplacer(".", "-", "/", "-").Replace(cwd)
    16		proj := filepath.Join(home, ".claude", "projects", enc)
    17		mustMkdir(t, data)
    18		mustMkdir(t, cwd)
    19		mustMkdir(t, proj)
    20		mustWrite(t, filepath.Join(data, "config-T-claude.json"), `{"session_id":"sid1"}`)
    21		mustWrite(t, filepath.Join(data, "pane-T-claude.json"), `{"pane_id":"7","cwd":"`+cwd+`","cwd_display":"~/repo"}`)
    22		mustWrite(t, filepath.Join(proj, "sid1.jsonl"),
    23			`{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":397556,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`)
    24	
    25		var stdout bytes.Buffer
    26		code := Run([]string{"T", "claude"}, Env{Home: home, PairDataDir: data}, &stdout)
    27		if code != 0 {
    28			t.Fatalf("code = %d, want 0", code)
    29		}
    30		if got := strings.TrimSpace(stdout.String()); got != "398k" {
    31			t.Fatalf("stdout = %q, want 398k", stdout.String())
    32		}
    33	}
    34	
    35	func TestRunMissingConfigPrintsNothing(t *testing.T) {
    36		home := t.TempDir()
    37		var stdout bytes.Buffer
    38		code := Run([]string{"T", "claude"}, Env{Home: home, PairDataDir: filepath.Join(home, "empty")}, &stdout)
    39		if code != 0 {
    40			t.Fatalf("code = %d, want 0", code)
    41		}
    42		if stdout.String() != "" {
    43			t.Fatalf("stdout = %q, want empty", stdout.String())
    44		}
    45	}
    46	
    47	func mustMkdir(t *testing.T, d string) {
    48		t.Helper()
    49		if err := os.MkdirAll(d, 0o755); err != nil {
    50			t.Fatal(err)
    51		}
    52	}
    53	
    54	func mustWrite(t *testing.T, p, s string) {
    55		t.Helper()
    56		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
    57			t.Fatal(err)
    58		}
    59	}

exec
/bin/zsh -lc "nl -ba cmd/internal/dispatcher/dispatcher_test.go | sed -n '1,240p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package dispatcher
     2	
     3	import (
     4		"os"
     5		"path/filepath"
     6		"strings"
     7		"testing"
     8	)
     9	
    10	func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
    11		for _, args := range [][]string{nil, {"help"}, {"--help"}, {"-h"}} {
    12			t.Run(strings.Join(args, "_"), func(t *testing.T) {
    13				res := Dispatch(args)
    14				if res.ExitCode != 0 {
    15					t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
    16				}
    17				if res.Stderr != "" {
    18					t.Fatalf("Stderr = %q, want empty", res.Stderr)
    19				}
    20				for _, want := range []string{
    21					"Usage: pair-go <command> [args]",
    22					"Implemented prototype commands:",
    23					"launch",
    24					"decision-phase only",
    25					"context",
    26					"scrollback-render",
    27					"wrap",
    28					"slug",
    29					"not implemented in this skeleton",
    30				} {
    31					if !strings.Contains(res.Stdout, want) {
    32						t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
    33					}
    34				}
    35				if strings.Contains(res.Stdout, "launch             session lifecycle and public pair launcher flow (planned; not implemented") {
    36					t.Fatalf("Stdout still labels launch unimplemented:\n%s", res.Stdout)
    37				}
    38				for _, stale := range []string{
    39					"context           agent pane context meter (planned; not implemented",
    40					"scrollback-render raw PTY capture to ANSI scrollback (planned; not implemented",
    41				} {
    42					if strings.Contains(res.Stdout, stale) {
    43						t.Fatalf("Stdout still labels helper unimplemented (%q):\n%s", stale, res.Stdout)
    44					}
    45				}
    46			})
    47		}
    48	}
    49	
    50	func TestDispatchVersionIsDevelopmentSkeletonMetadata(t *testing.T) {
    51		res := Dispatch([]string{"version"})
    52		if res.ExitCode != 0 {
    53			t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
    54		}
    55		if res.Stderr != "" {
    56			t.Fatalf("Stderr = %q, want empty", res.Stderr)
    57		}
    58		for _, want := range []string{"pair-go", "dispatcher skeleton", "public launcher: bin/pair"} {
    59			if !strings.Contains(res.Stdout, want) {
    60				t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
    61			}
    62		}
    63	}
    64	
    65	func TestDispatchPlannedCommandReturnsUnsupported(t *testing.T) {
    66		res := Dispatch([]string{"wrap"})
    67		if res.ExitCode != 2 {
    68			t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
    69		}
    70		if res.Stdout != "" {
    71			t.Fatalf("Stdout = %q, want empty", res.Stdout)
    72		}
    73		for _, want := range []string{"wrap", "planned", "not implemented", "pair-go help"} {
    74			if !strings.Contains(res.Stderr, want) {
    75				t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
    76			}
    77		}
    78	}
    79	
    80	func TestDispatchLaunchHelpRoutesToPrototype(t *testing.T) {
    81		res := Dispatch([]string{"launch", "--help"})
    82		if res.ExitCode != 0 {
    83			t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
    84		}
    85		if res.Stderr != "" {
    86			t.Fatalf("Stderr = %q, want empty", res.Stderr)
    87		}
    88		for _, want := range []string{"Usage: pair-go launch", "decision-phase prototype"} {
    89			if !strings.Contains(res.Stdout, want) {
    90				t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
    91			}
    92		}
    93	}
    94	
    95	func TestDispatchLaunchReturnsPrototypeDecision(t *testing.T) {
    96		res := DispatchWithLauncherRuntime([]string{"launch", "resume", "demo"}, LauncherRuntime{
    97			Env: LauncherEnv("/home/me", "", "/work/pair"),
    98			Sessions: StaticSessions{
    99				Sessions: nil,
   100			},
   101			History: StaticHistory{},
   102		})
   103		if res.ExitCode != 3 {
   104			t.Fatalf("ExitCode = %d, want 3", res.ExitCode)
   105		}
   106		if res.Stdout != "" {
   107			t.Fatalf("Stdout = %q, want empty", res.Stdout)
   108		}
   109		for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=demo", "session=pair-demo"} {
   110			if !strings.Contains(res.Stderr, want) {
   111				t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
   112			}
   113		}
   114	}
   115	
   116	func TestDispatchLaunchWithoutArgsReturnsDefaultPrototypeDecision(t *testing.T) {
   117		res := DispatchWithLauncherRuntime([]string{"launch"}, LauncherRuntime{
   118			Env: LauncherEnv("/home/me", "", "/work/pair"),
   119			Sessions: StaticSessions{
   120				Sessions: nil,
   121			},
   122			History: StaticHistory{},
   123		})
   124		if res.ExitCode != 3 {
   125			t.Fatalf("ExitCode = %d, want 3", res.ExitCode)
   126		}
   127		if res.Stdout != "" {
   128			t.Fatalf("Stdout = %q, want empty", res.Stdout)
   129		}
   130		for _, want := range []string{"pair-go launch: prototype decision", "action=create", "tag=pair", "session=pair-pair"} {
   131			if !strings.Contains(res.Stderr, want) {
   132				t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
   133			}
   134		}
   135	}
   136	
   137	func TestDispatchContextReturnsHelperOutput(t *testing.T) {
   138		home, data := writeContextFixture(t)
   139		t.Setenv("HOME", home)
   140		t.Setenv("PAIR_DATA_DIR", data)
   141	
   142		res := Dispatch([]string{"context", "T", "claude"})
   143		if res.ExitCode != 0 {
   144			t.Fatalf("ExitCode = %d, want 0; stderr:\n%s", res.ExitCode, res.Stderr)
   145		}
   146		if res.Stderr != "" {
   147			t.Fatalf("Stderr = %q, want empty", res.Stderr)
   148		}
   149		if strings.TrimSpace(res.Stdout) != "398k" {
   150			t.Fatalf("Stdout = %q, want 398k", res.Stdout)
   151		}
   152	}
   153	
   154	func TestDispatchScrollbackRenderUsage(t *testing.T) {
   155		res := Dispatch([]string{"scrollback-render"})
   156		if res.ExitCode != 2 {
   157			t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
   158		}
   159		if res.Stdout != "" {
   160			t.Fatalf("Stdout = %q, want empty", res.Stdout)
   161		}
   162		if !strings.Contains(res.Stderr, "usage: pair-scrollback-render") {
   163			t.Fatalf("Stderr missing usage:\n%s", res.Stderr)
   164		}
   165	}
   166	
   167	func TestDispatchUnknownCommandReturnsUsageHint(t *testing.T) {
   168		res := Dispatch([]string{"frobnicate"})
   169		if res.ExitCode != 2 {
   170			t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
   171		}
   172		if res.Stdout != "" {
   173			t.Fatalf("Stdout = %q, want empty", res.Stdout)
   174		}
   175		for _, want := range []string{"unknown command", "frobnicate", "pair-go help"} {
   176			if !strings.Contains(res.Stderr, want) {
   177				t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
   178			}
   179		}
   180	}
   181	
   182	func writeContextFixture(t *testing.T) (home, data string) {
   183		t.Helper()
   184		home = t.TempDir()
   185		data = filepath.Join(home, "data")
   186		cwd := filepath.Join(home, "repo")
   187		enc := strings.NewReplacer(".", "-", "/", "-").Replace(cwd)
   188		proj := filepath.Join(home, ".claude", "projects", enc)
   189		mustMkdir(t, data)
   190		mustMkdir(t, cwd)
   191		mustMkdir(t, proj)
   192		mustWrite(t, filepath.Join(data, "config-T-claude.json"), `{"session_id":"sid1"}`)
   193		mustWrite(t, filepath.Join(data, "pane-T-claude.json"), `{"pane_id":"7","cwd":"`+cwd+`","cwd_display":"~/repo"}`)
   194		mustWrite(t, filepath.Join(proj, "sid1.jsonl"),
   195			`{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":397556,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`)
   196		return home, data
   197	}
   198	
   199	func mustMkdir(t *testing.T, d string) {
   200		t.Helper()
   201		if err := os.MkdirAll(d, 0o755); err != nil {
   202			t.Fatal(err)
   203		}
   204	}
   205	
   206	func mustWrite(t *testing.T, p, s string) {
   207		t.Helper()
   208		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
   209			t.Fatal(err)
   210		}
   211	}

codex
The tests cover the representative `context` process equivalence and the dispatcher route shape. They do not cover process-level equivalence for `scrollback-render`, so I’m doing a quick manual comparison against the base implementation behavior where the wrapper changed.
exec
/bin/zsh -lc "nl -ba workshop/plans/000076-go-helper-dispatch-plan.md | sed -n '1,90p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	# Go Helper Dispatch Implementation Plan
     2	
     3	> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.
     4	
     5	**Goal:** Add `pair-go context` and `pair-go scrollback-render` routes that reuse the existing helper implementations while preserving the legacy helper binaries.
     6	
     7	**Architecture:** Extract reusable helper runners from the selected `package main` commands and route the dispatcher through those runners instead of reimplementing command behavior (`ARCH-DRY`). Keep helper business logic pure or close to existing pure cores, with IO at thin command boundaries: runners accept argv/stdout/stderr/env-style inputs and return exit status or error, while `main()` remains a tiny process shell (`ARCH-PURE`). The slice intentionally proves the helper-dispatch pattern without moving live zellij/nvim call sites or changing public `pair` behavior (`ARCH-PURPOSE`).
     8	
     9	**Tech Stack:** Go standard library, existing `cmd/internal/dispatcher`, existing `cmd/internal/ctxmeter` and `cmd/internal/transcript`, existing scrollback renderer core, `go test`, `make`.
    10	
    11	---
    12	
    13	## Core Concepts
    14	
    15	### Pure Entities
    16	
    17	| Name | Lives in | Status |
    18	|------|----------|--------|
    19	| `ContextRunArgs` | `cmd/internal/contextcmd/contextcmd.go` | new |
    20	| `ScrollbackRenderArgs` | `cmd/internal/scrollbackcmd/scrollbackcmd.go` | new |
    21	
    22	**ContextRunArgs** — Parsed input for the context helper: tag, agent, home, and Pair data dir.
    23	- **Relationships:** 1:1 with a `pair-context`/`pair-go context` invocation; consumed by the context runner.
    24	- **DRY rationale:** Both legacy binary and dispatcher route need the same env/default resolution and argument validation.
    25	- **Future extensions:** Can widen if more commands need shared Pair environment resolution.
    26	
    27	**ScrollbackRenderArgs** — Parsed input for the renderer helper: raw capture path, events path, output path, and render flags.
    28	- **Relationships:** 1:1 with a render invocation; maps directly onto existing render parameters.
    29	- **DRY rationale:** Avoids parallel flag parsing between `pair-scrollback-render` and `pair-go scrollback-render`.
    30	- **Future extensions:** Can become the command-facing shape for future `pair scrollback-render` after the public entrypoint switch.
    31	
    32	### Integration Points
    33	
    34	| Name | Lives in | Status | Wraps |
    35	|------|----------|--------|-------|
    36	| `ContextRunner` | `cmd/internal/contextcmd/contextcmd.go` | new | filesystem, env, transcript files, stdout |
    37	| `ScrollbackRenderRunner` | `cmd/internal/scrollbackcmd/scrollbackcmd.go` | new | filesystem, flag parsing, stdout/stderr |
    38	| `HelperDispatchRoutes` | `cmd/internal/dispatcher/dispatcher.go` | modified | in-process calls to selected helper runners |
    39	| `PairGoMain` | `cmd/pair-go/main.go` | modified | process stdout/stderr exit handling |
    40	
    41	**ContextRunner** — Shared command runner for `pair-context <tag> <agent>` behavior. It remains tolerant: missing config/transcript/input prints nothing and exits 0.
    42	- **Injected into:** legacy `cmd/pair-context/main.go` and dispatcher context route.
    43	- **Future extensions:** The title poller can continue using the legacy binary until #77/#78 moves call sites.
    44	
    45	**ScrollbackRenderRunner** — Shared command runner for `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out`.
    46	- **Injected into:** legacy `cmd/pair-scrollback-render/main.go` and dispatcher scrollback-render route.
    47	- **Future extensions:** `bin/pair-scrollback-open`, `bin/pair-changelog-open`, and `nvim/scrollback.lua` can move to the dispatcher after the public entrypoint is Go-owned.
    48	
    49	**HelperDispatchRoutes** — Dispatcher routes for `context` and `scrollback-render`.
    50	- **Injected into:** `cmd/pair-go`.
    51	- **Future extensions:** Later helper routes should follow the same runner extraction pattern, not duplicate command logic.
    52	
    53	**PairGoMain** — Existing process shell that writes dispatcher results.
    54	- **Injected into:** none.
    55	- **Future extensions:** Eventually becomes the public `pair` entrypoint in #77, but not here.
    56	
    57	---
    58	
    59	## Chunk 1: Extract Context Runner
    60	
    61	### Task 1: Make `pair-context` Reusable
    62	
    63	**Files:**
    64	- Create: `cmd/internal/contextcmd/contextcmd.go`
    65	- Create: `cmd/internal/contextcmd/contextcmd_test.go`
    66	- Modify: `cmd/pair-context/main.go`
    67	- Modify: `cmd/pair-context/main_test.go`
    68	- Modify: `Makefile.local`
    69	
    70	- [x] **Step 1: Add failing runner tests**
    71	
    72	Create `cmd/internal/contextcmd/contextcmd_test.go` with tests that call `Run(args []string, env Env, stdout io.Writer) int` directly:
    73	
    74	```go
    75	func TestRunClaude(t *testing.T) {
    76	    // Arrange the same config/pane/transcript fixture as TestPairContext_Claude.
    77	    // Call Run([]string{"T", "claude"}, env, &stdout).
    78	    // Assert code == 0 and stdout == "398k\n".
    79	}
    80	
    81	func TestRunMissingConfigPrintsNothing(t *testing.T) {
    82	    // Call Run with empty data dir.
    83	    // Assert code == 0 and stdout == "".
    84	}
    85	```
    86	
    87	- [x] **Step 2: Run the focused tests and confirm they fail**
    88	
    89	Run: `go test ./cmd/internal/contextcmd -count=1`
    90	

exec
/bin/zsh -lc 'git show 9cc0f57521c3e44508adf3df006ffe3347faa265:cmd/pair-scrollback-render/main.go | tail -n 40' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
	// new complete file, never a half-written one.
	tmp := outPath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	body := strings.Join(out, "\n")
	if len(out) > 0 {
		body += "\n"
	}
	if _, err := f.WriteString(body); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, outPath)
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [--plain] [--max-lines N] [--with-timestamps] <raw> <events.jsonl> <out>\n", os.Args[0])
	}
	plain := flag.Bool("plain", false, "emit plain text (no SGR) for distillation")
	maxLines := flag.Int("max-lines", historyRows, "scrollback history rows retained; <=0 = uncapped")
	withTimestamps := flag.Bool("with-timestamps", false, "interleave ⟦pair:ts DATE⟧ day markers from time events (for the change log; #59)")
	flag.Parse()
	args := flag.Args()
	if len(args) != 3 {
		flag.Usage()
		os.Exit(2)
	}
	if err := render(args[0], args[1], args[2], *plain, *maxLines, *withTimestamps); err != nil {
		fmt.Fprintf(os.Stderr, "scrollback-render: %v\n", err)
		os.Exit(1)
	}
}

exec
/bin/zsh -lc 'git diff -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollback-open bin/pair-changelog-open' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
exec
/bin/zsh -lc 'go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1' in /Users/xianxu/workspace/pair
 succeeded in 2437ms:
ok  	github.com/xianxu/pair/cmd/internal/contextcmd	0.282s
ok  	github.com/xianxu/pair/cmd/internal/scrollbackcmd	0.496s
ok  	github.com/xianxu/pair/cmd/pair-context	1.830s
?   	github.com/xianxu/pair/cmd/pair-scrollback-render	[no test files]
ok  	github.com/xianxu/pair/cmd/internal/dispatcher	1.158s
ok  	github.com/xianxu/pair/cmd/pair-go	2.263s

codex
Focused tests pass, and the live caller diff is empty. I’m running the full suite once and then I’ll produce the verdict; at this point the only blocking item I see is the plan/Core Concepts contradiction required by the review checklist.
exec
/bin/zsh -lc 'go test ./... -count=1' in /Users/xianxu/workspace/pair
 succeeded in 17136ms:
ok  	github.com/xianxu/pair/cmd/internal/adapt	0.250s
ok  	github.com/xianxu/pair/cmd/internal/contextcmd	0.355s
ok  	github.com/xianxu/pair/cmd/internal/ctxmeter	0.574s
ok  	github.com/xianxu/pair/cmd/internal/dispatcher	0.715s
ok  	github.com/xianxu/pair/cmd/internal/launcher	2.905s
ok  	github.com/xianxu/pair/cmd/internal/model	4.121s
ok  	github.com/xianxu/pair/cmd/internal/scrollbackcmd	1.441s
ok  	github.com/xianxu/pair/cmd/internal/transcript	1.274s
ok  	github.com/xianxu/pair/cmd/pair-changelog	16.777s
ok  	github.com/xianxu/pair/cmd/pair-context	4.259s
ok  	github.com/xianxu/pair/cmd/pair-continuation	6.586s
ok  	github.com/xianxu/pair/cmd/pair-go	5.307s
?   	github.com/xianxu/pair/cmd/pair-scribe	[no test files]
?   	github.com/xianxu/pair/cmd/pair-scrollback-render	[no test files]
ok  	github.com/xianxu/pair/cmd/pair-slug	7.561s
ok  	github.com/xianxu/pair/cmd/pair-wrap	2.286s

codex
VERDICT: REWORK (confidence: high)

The implementation appears to satisfy the runtime purpose of #76: both selected helpers route through `pair-go`, legacy binaries remain as thin wrappers, live zellij/nvim/bin call sites did not move, atlas was updated, and `go test ./... -count=1` passes. The blocker is tracker/plan truthfulness: the plan’s Core Concepts table claims new PURE entities that do not exist in the code, and the boundary checklist explicitly treats a Core Concepts contradiction as Critical until the plan is revised.

1. Strengths:
- `cmd/pair-context/main.go:10` is now a thin process shell over `contextcmd.Run`, preserving legacy command shape while sharing code.
- `cmd/internal/dispatcher/dispatcher.go:61` and `cmd/internal/dispatcher/dispatcher.go:63` route `context` and `scrollback-render` through shared internal runners, which is the right `ARCH-DRY` direction.
- `cmd/pair-go/helper_equivalence_test.go:12` gives process-level proof that `pair-go context` matches legacy `pair-context`.
- Atlas updates in `atlas/architecture.md` and `atlas/go-migration-inventory.md` record the new helper-dispatch surface and unchanged live callers.

2. Critical findings:
- `workshop/plans/000076-go-helper-dispatch-plan.md:17`: Core Concepts lists `ContextRunArgs` and `ScrollbackRenderArgs` as new PURE entities, but `rg` finds those names only in the plan. The actual code exposes `contextcmd.Env`, `contextcmd.Run`, and `scrollbackcmd.Run` instead. Per the boundary’s Core concepts cross-check, this contradiction is Critical. Fix sketch: add a `## Revisions` entry and update the Core Concepts table to match the shipped surface, likely classifying the runners as INTEGRATION rather than claiming nonexistent PURE parsed-args entities. Only add the structs if they are genuinely useful; the current simpler code surface looks acceptable.

3. Important findings:
- None.

4. Minor findings:
- `cmd/internal/dispatcher/dispatcher.go:184`: help still says “Implemented prototype commands” even though it now includes implemented helper routes. Consider renaming the heading to “Implemented commands” or splitting prototype/helper sections.
- `cmd/internal/scrollbackcmd/scrollbackcmd.go:391`: `stdout` is accepted but unused; harmless for this helper, but worth documenting or omitting in a future cleanup if no helper needs stdout.

5. Test coverage notes:
- Verified: `go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1` passes.
- Verified: `go test ./... -count=1` passes.
- Existing coverage proves representative process equivalence for `context`; `scrollback-render` has runner-level tests and moved renderer tests, but no process-level legacy-vs-dispatch equivalence test. That is acceptable under the issue’s “at least one representative helper” done-when.

6. Architectural notes for upcoming work:
- `ARCH-DRY`: Pass. Shared runners remove duplicate command bodies.
- `ARCH-PURE`: Pass for the implemented code shape. IO is still in runner packages, but process shells are thin and pure renderer/context cores remain testable.
- `ARCH-PURPOSE`: Pass for runtime scope. The issue intentionally keeps live call sites on legacy names and proves the dispatcher pattern without switching public `pair`.

7. Plan revision recommendations:
- Add `## Revisions` to `workshop/plans/000076-go-helper-dispatch-plan.md` with an entry like: `2026-06-30: Implementation used direct shared runner APIs instead of parsed args value types; replaced nonexistent ContextRunArgs/ScrollbackRenderArgs PURE rows with the actual Env/Run and scrollbackcmd.Run integration surfaces.`
tokens used
65,295
VERDICT: REWORK (confidence: high)

The implementation appears to satisfy the runtime purpose of #76: both selected helpers route through `pair-go`, legacy binaries remain as thin wrappers, live zellij/nvim/bin call sites did not move, atlas was updated, and `go test ./... -count=1` passes. The blocker is tracker/plan truthfulness: the plan’s Core Concepts table claims new PURE entities that do not exist in the code, and the boundary checklist explicitly treats a Core Concepts contradiction as Critical until the plan is revised.

1. Strengths:
- `cmd/pair-context/main.go:10` is now a thin process shell over `contextcmd.Run`, preserving legacy command shape while sharing code.
- `cmd/internal/dispatcher/dispatcher.go:61` and `cmd/internal/dispatcher/dispatcher.go:63` route `context` and `scrollback-render` through shared internal runners, which is the right `ARCH-DRY` direction.
- `cmd/pair-go/helper_equivalence_test.go:12` gives process-level proof that `pair-go context` matches legacy `pair-context`.
- Atlas updates in `atlas/architecture.md` and `atlas/go-migration-inventory.md` record the new helper-dispatch surface and unchanged live callers.

2. Critical findings:
- `workshop/plans/000076-go-helper-dispatch-plan.md:17`: Core Concepts lists `ContextRunArgs` and `ScrollbackRenderArgs` as new PURE entities, but `rg` finds those names only in the plan. The actual code exposes `contextcmd.Env`, `contextcmd.Run`, and `scrollbackcmd.Run` instead. Per the boundary’s Core concepts cross-check, this contradiction is Critical. Fix sketch: add a `## Revisions` entry and update the Core Concepts table to match the shipped surface, likely classifying the runners as INTEGRATION rather than claiming nonexistent PURE parsed-args entities. Only add the structs if they are genuinely useful; the current simpler code surface looks acceptable.

3. Important findings:
- None.

4. Minor findings:
- `cmd/internal/dispatcher/dispatcher.go:184`: help still says “Implemented prototype commands” even though it now includes implemented helper routes. Consider renaming the heading to “Implemented commands” or splitting prototype/helper sections.
- `cmd/internal/scrollbackcmd/scrollbackcmd.go:391`: `stdout` is accepted but unused; harmless for this helper, but worth documenting or omitting in a future cleanup if no helper needs stdout.

5. Test coverage notes:
- Verified: `go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1` passes.
- Verified: `go test ./... -count=1` passes.
- Existing coverage proves representative process equivalence for `context`; `scrollback-render` has runner-level tests and moved renderer tests, but no process-level legacy-vs-dispatch equivalence test. That is acceptable under the issue’s “at least one representative helper” done-when.

6. Architectural notes for upcoming work:
- `ARCH-DRY`: Pass. Shared runners remove duplicate command bodies.
- `ARCH-PURE`: Pass for the implemented code shape. IO is still in runner packages, but process shells are thin and pure renderer/context cores remain testable.
- `ARCH-PURPOSE`: Pass for runtime scope. The issue intentionally keeps live call sites on legacy names and proves the dispatcher pattern without switching public `pair`.

7. Plan revision recommendations:
- Add `## Revisions` to `workshop/plans/000076-go-helper-dispatch-plan.md` with an entry like: `2026-06-30: Implementation used direct shared runner APIs instead of parsed args value types; replaced nonexistent ContextRunArgs/ScrollbackRenderArgs PURE rows with the actual Env/Run and scrollbackcmd.Run integration surfaces.`

---

## Re-review — 2026-06-30T12:29:15-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 76 — pair Go helper dispatch |
| repo | pair |
| issue file | workshop/issues/000076-go-helper-dispatch.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9cc0f57521c3e44508adf3df006ffe3347faa265..HEAD |
| command | sdlc close --issue 76 |
| reviewer | codex |
| timestamp | 2026-06-30T12:29:15-07:00 |
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
session id: 019f1a00-384c-7580-a5fe-49307f187977
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **pair** repository.

- repository: pair   (root: /Users/xianxu/workspace/pair)
- issue:      pair#76   (file: workshop/issues/000076-go-helper-dispatch.md)
- window:     Base: 9cc0f57521c3e44508adf3df006ffe3347faa265   Head: HEAD

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
index 0618d25..ccdde68 100644
--- a/Makefile.local
+++ b/Makefile.local
@@ -210,7 +210,7 @@ $(BIN_DIR)/pair-scribe: cmd/pair-scribe/main.go go.mod
 # pair-scrollback-render — replays a pair-wrap raw PTY capture through
 # charmbracelet/x/vt and emits one ANSI-styled line per logical scrollback
 # row, for the Alt+/ viewer to display through nvim.
-$(BIN_DIR)/pair-scrollback-render: cmd/pair-scrollback-render/main.go go.mod
+$(BIN_DIR)/pair-scrollback-render: cmd/pair-scrollback-render/main.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
 	go build -o $@ ./cmd/pair-scrollback-render
 
 # pair-wrap — transparent PTY proxy around a TUI coding agent: forwards
@@ -225,7 +225,7 @@ $(BIN_DIR)/pair-slug: cmd/pair-slug/main.go cmd/pair-slug/slug.go cmd/internal/m
 
 # pair-context — one-shot: print a pane's current context size (token count)
 # for the pair-title poller. Pure ctxmeter reader + shared transcript resolver.
-$(BIN_DIR)/pair-context: cmd/pair-context/main.go cmd/internal/ctxmeter/ctxmeter.go cmd/internal/transcript/transcript.go go.mod
+$(BIN_DIR)/pair-context: cmd/pair-context/main.go cmd/internal/contextcmd/contextcmd.go cmd/internal/ctxmeter/ctxmeter.go cmd/internal/transcript/transcript.go go.mod
 	go build -o $@ ./cmd/pair-context
 
 # pair-changelog — distills the pair TTY into the Alt+l change log (#53);
@@ -241,7 +241,7 @@ $(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-con
 
 # pair-go — development-only dispatcher skeleton for the future primary Go CLI.
 # The public launcher remains bin/pair until the entrypoint-switch issue.
-$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/dispatcher/dispatcher.go go.mod
+$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
 	go build -o $@ ./cmd/pair-go
 
 
diff --git a/atlas/architecture.md b/atlas/architecture.md
index a536fc7..5e308ab 100644
--- a/atlas/architecture.md
+++ b/atlas/architecture.md
@@ -55,6 +55,14 @@ prompt/fzf UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation,
 rename, config/session migration, and title-poller behavior remain shell-owned
 until later migration issues.
 
+As of #76, the same dispatcher also has the first implemented helper routes:
+`pair-go context <tag> <agent>` and `pair-go scrollback-render ...`. Both routes
+call shared internal Go runners used by the legacy `pair-context` and
+`pair-scrollback-render` binaries (`ARCH-DRY`), so behavior stays aligned while
+the old command names remain the live integration surface. `bin/pair-title.sh`,
+`bin/pair-scrollback-open`, `bin/pair-changelog-open`, and
+`nvim/scrollback.lua` have not moved to the dispatcher yet.
+
 Native integration layers stay native: `nvim/*.lua` remains the bundled Neovim
 surface and `zellij/*.kdl` remains the zellij layout/config surface. Packaging
 may embed those assets or install them adjacent to the binary, but the migration
diff --git a/atlas/go-migration-inventory.md b/atlas/go-migration-inventory.md
index 3de3bb2..8b404f0 100644
--- a/atlas/go-migration-inventory.md
+++ b/atlas/go-migration-inventory.md
@@ -54,8 +54,8 @@ Priority is packaging impact first, then reliability/testability:
 | `zellij/config.kdl` | zellij native asset | zellij session config from `bin/pair` | Global keybinds, copy command, scrollback buffer, pane frames. | Calls `copy-on-select.sh`, `pair-help`, `pair-scrollback-open`, `pair-changelog-open`; routes quit/restart/compact through nvim functions. | native-asset, packaged adjacent/embedded | P0 |
 | `bin/pair-wrap` / `cmd/pair-wrap` | Go binary | zellij agent pane | `pair-wrap [--scrollback-log PATH] agent [args...]`; transparent PTY proxy; long-running; failure in diagnostics is swallowed. | Reads Pair env and agent command; writes `agent-output-<tag>`, `agent-pid-<tag>`, scrollback `.raw`/`.events.jsonl`, image capture files; may invoke `pair-slug`. | go-subcommand `pair wrap`; keep old binary as compat alias for KDL until caller moves | P0 |
 | `bin/pair-slug` / `cmd/pair-slug` | Go binary | `pair-wrap` turn-end hook, tests | Env-driven, no stdin; resolves native transcript, proposes slug; exits 0 on most failures. | Requires `PAIR_TAG`, `PAIR_DATA_DIR`; reads config/transcripts/git branch; writes `slug-proposed-<tag>`; optional `PAIR_SLUG_*`, `OPENAI_API_KEY`. | go-subcommand `pair slug`; legacy binary retained during #76 | P1 |
-| `bin/pair-context` / `cmd/pair-context` | Go binary | `bin/pair-title.sh` | `pair-context <tag> <agent>`; prints humanized token count or nothing; tolerant exit 0 on failure. | Reads `PAIR_DATA_DIR`, `pane-<tag>-<agent>.json`, config, native transcripts. | go-subcommand `pair context`; legacy binary retained while title poller calls it | P1 |
-| `bin/pair-scrollback-render` / `cmd/pair-scrollback-render` | Go binary | `bin/pair-scrollback-open`, `bin/pair-changelog-open`, `nvim/scrollback.lua` refresh | `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out`; nonzero on render/write failure. | Reads `.raw` and `.events.jsonl`; atomically writes `.ansi` or cleaned text. | go-subcommand `pair scrollback-render`; legacy binary retained for shell/Lua callers | P0 |
+| `bin/pair-context` / `cmd/pair-context` / `cmd/internal/contextcmd` | Go binary plus shared runner | `bin/pair-title.sh`; development-only `pair-go context` | `pair-context <tag> <agent>` and `pair-go context <tag> <agent>` print the same humanized token count or nothing; tolerant exit 0 on failure. | Reads `PAIR_DATA_DIR`, `pane-<tag>-<agent>.json`, config, native transcripts. | implemented helper route in `pair-go context`; legacy binary retained while title poller calls it | P1 |
+| `bin/pair-scrollback-render` / `cmd/pair-scrollback-render` / `cmd/internal/scrollbackcmd` | Go binary plus shared runner | `bin/pair-scrollback-open`, `bin/pair-changelog-open`, `nvim/scrollback.lua` refresh; development-only `pair-go scrollback-render` | `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out` and `pair-go scrollback-render ...`; nonzero on render/write failure. | Reads `.raw` and `.events.jsonl`; atomically writes `.ansi` or cleaned text. | implemented helper route in `pair-go scrollback-render`; legacy binary retained for shell/Lua callers | P0 |
 | `bin/pair-changelog` / `cmd/pair-changelog` | Go binary | `bin/pair-changelog-open` | `pair-changelog --cleaned F --log F --anchor F [--agent A] [--model M]`; exits nonzero on required read/model/write failure. | Reads cleaned scrollback/log/anchor; calls agent model through internal model runner; atomically writes log and anchor. | go-subcommand `pair changelog`; legacy binary retained for opener | P1 |
 | `bin/pair-continuation` / `cmd/pair-continuation` | Go binary | nvim compaction prompt instructions, operator/agent shell | `pair-continuation --slug S --agent A --issues CSV --body-file F [--repo-root R ...]`; writes and commits continuation; nonzero on validation/git failure. | Reads body/stdin, git repo state; writes `workshop/continuation/*.md`; runs git commit/push. | go-subcommand `pair continuation`; legacy binary retained for agent instructions until docs change | P1 |
 | `bin/pair-scribe` / `cmd/pair-scribe` | Go binary | user shell rc outside Pair sessions | `pair-scribe -log PATH -- CMD [ARGS...]`; long-running PTY wrapper; SIGUSR1 pauses log, SIGUSR2 resumes. | Writes typescript log; wraps child PTY; independent of `PAIR_*`. | go-subcommand candidate is low value; may remain separate installed helper or become `pair scribe` with alias | P2 |
@@ -125,9 +125,10 @@ Build/install callers:
 - #75 and #76 can proceed in parallel after #74. The launcher prototype does not
   need helper dispatch to exist first, and helper dispatch does not need the
   launcher prototype.
-- #76 should start with existing Go helpers whose contracts are already clean:
-  `pair context`, `pair slug`, and `pair scrollback-render` are good first
-  candidates. Keep legacy binary names working.
+- #76 established the first helper-dispatch pattern with `pair-go context` and
+  `pair-go scrollback-render`, backed by shared internal runners while legacy
+  binary names remain live for shell/Lua callers. `pair slug` remains a later
+  candidate.
 - #77 is the public entrypoint switch and should wait for both #75 and #76.
 - #78 should prioritize `pair-title.sh` and `pair-session-watch.sh` if stateful
   shell remains a packaging/reliability problem after #77.
diff --git a/cmd/internal/contextcmd/contextcmd.go b/cmd/internal/contextcmd/contextcmd.go
new file mode 100644
index 0000000..ca6debb
--- /dev/null
+++ b/cmd/internal/contextcmd/contextcmd.go
@@ -0,0 +1,78 @@
+// Package contextcmd implements the pair-context command body.
+package contextcmd
+
+import (
+	"encoding/json"
+	"fmt"
+	"io"
+	"os"
+	"path/filepath"
+
+	"github.com/xianxu/pair/cmd/internal/ctxmeter"
+	"github.com/xianxu/pair/cmd/internal/transcript"
+)
+
+type Env struct {
+	Home        string
+	XDGDataHome string
+	PairDataDir string
+}
+
+func EnvFromOS() Env {
+	return Env{
+		Home:        os.Getenv("HOME"),
+		XDGDataHome: os.Getenv("XDG_DATA_HOME"),
+		PairDataDir: os.Getenv("PAIR_DATA_DIR"),
+	}
+}
+
+func Run(args []string, env Env, stdout io.Writer) int {
+	if len(args) < 2 {
+		return 0
+	}
+	tag, agent := args[0], args[1]
+	dataDir := resolveDataDir(env)
+	sid := transcript.SessionID(dataDir, tag, agent)
+	if sid == "" {
+		return 0
+	}
+	cwd := paneCwd(dataDir, tag, agent)
+	path := transcript.Resolve(agent, sid, cwd, env.Home)
+	if path == "" {
+		return 0
+	}
+	f, err := os.Open(path)
+	if err != nil {
+		return 0
+	}
+	defer f.Close()
+	if n, ok := ctxmeter.ContextTokens(agent, f); ok {
+		fmt.Fprintln(stdout, ctxmeter.Humanize(n))
+	}
+	return 0
+}
+
+func resolveDataDir(env Env) string {
+	if env.PairDataDir != "" {
+		return env.PairDataDir
+	}
+	base := env.XDGDataHome
+	if base == "" {
+		base = filepath.Join(env.Home, ".local", "share")
+	}
+	return filepath.Join(base, "pair")
+}
+
+func paneCwd(dataDir, tag, agent string) string {
+	b, err := os.ReadFile(filepath.Join(dataDir, "pane-"+tag+"-"+agent+".json"))
+	if err != nil {
+		return ""
+	}
+	var p struct {
+		Cwd string `json:"cwd"`
+	}
+	if json.Unmarshal(b, &p) != nil {
+		return ""
+	}
+	return p.Cwd
+}
diff --git a/cmd/internal/contextcmd/contextcmd_test.go b/cmd/internal/contextcmd/contextcmd_test.go
new file mode 100644
index 0000000..2e07506
--- /dev/null
+++ b/cmd/internal/contextcmd/contextcmd_test.go
@@ -0,0 +1,59 @@
+package contextcmd
+
+import (
+	"bytes"
+	"os"
+	"path/filepath"
+	"strings"
+	"testing"
+)
+
+func TestRunClaude(t *testing.T) {
+	home := t.TempDir()
+	data := filepath.Join(home, "data")
+	cwd := filepath.Join(home, "repo")
+	enc := strings.NewReplacer(".", "-", "/", "-").Replace(cwd)
+	proj := filepath.Join(home, ".claude", "projects", enc)
+	mustMkdir(t, data)
+	mustMkdir(t, cwd)
+	mustMkdir(t, proj)
+	mustWrite(t, filepath.Join(data, "config-T-claude.json"), `{"session_id":"sid1"}`)
+	mustWrite(t, filepath.Join(data, "pane-T-claude.json"), `{"pane_id":"7","cwd":"`+cwd+`","cwd_display":"~/repo"}`)
+	mustWrite(t, filepath.Join(proj, "sid1.jsonl"),
+		`{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":397556,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`)
+
+	var stdout bytes.Buffer
+	code := Run([]string{"T", "claude"}, Env{Home: home, PairDataDir: data}, &stdout)
+	if code != 0 {
+		t.Fatalf("code = %d, want 0", code)
+	}
+	if got := strings.TrimSpace(stdout.String()); got != "398k" {
+		t.Fatalf("stdout = %q, want 398k", stdout.String())
+	}
+}
+
+func TestRunMissingConfigPrintsNothing(t *testing.T) {
+	home := t.TempDir()
+	var stdout bytes.Buffer
+	code := Run([]string{"T", "claude"}, Env{Home: home, PairDataDir: filepath.Join(home, "empty")}, &stdout)
+	if code != 0 {
+		t.Fatalf("code = %d, want 0", code)
+	}
+	if stdout.String() != "" {
+		t.Fatalf("stdout = %q, want empty", stdout.String())
+	}
+}
+
+func mustMkdir(t *testing.T, d string) {
+	t.Helper()
+	if err := os.MkdirAll(d, 0o755); err != nil {
+		t.Fatal(err)
+	}
+}
+
+func mustWrite(t *testing.T, p, s string) {
+	t.Helper()
+	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
+		t.Fatal(err)
+	}
+}
diff --git a/cmd/internal/dispatcher/dispatcher.go b/cmd/internal/dispatcher/dispatcher.go
index 96d6b3b..3784f1e 100644
--- a/cmd/internal/dispatcher/dispatcher.go
+++ b/cmd/internal/dispatcher/dispatcher.go
@@ -1,12 +1,15 @@
 package dispatcher
 
 import (
+	"bytes"
 	"fmt"
 	"os"
 	"strings"
 	"time"
 
+	"github.com/xianxu/pair/cmd/internal/contextcmd"
 	"github.com/xianxu/pair/cmd/internal/launcher"
+	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
 )
 
 const programName = "pair-go"
@@ -29,10 +32,10 @@ type Result struct {
 func Families() []CommandFamily {
 	return []CommandFamily{
 		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "prototype"},
+		{Name: "context", Summary: "agent pane context meter", Status: "implemented"},
+		{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "implemented"},
 		{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
 		{Name: "slug", Summary: "session orientation slug generation", Status: "planned"},
-		{Name: "context", Summary: "agent pane context meter", Status: "planned"},
-		{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "planned"},
 		{Name: "changelog", Summary: "TTY transcript to distilled change log", Status: "planned"},
 		{Name: "continuation", Summary: "continuation datatype writer", Status: "planned"},
 		{Name: "scribe", Summary: "PTY logging wrapper", Status: "planned"},
@@ -55,6 +58,10 @@ func Dispatch(args []string) Result {
 		}
 	case "launch":
 		return DispatchWithLauncherRuntime(args, osLauncherRuntime())
+	case "context":
+		return dispatchContext(args[1:])
+	case "scrollback-render":
+		return dispatchScrollbackRender(args[1:])
 	}
 
 	if family, ok := familyByName(args[0]); ok {
@@ -70,6 +77,18 @@ func Dispatch(args []string) Result {
 	}
 }
 
+func dispatchContext(args []string) Result {
+	var stdout bytes.Buffer
+	code := contextcmd.Run(args, contextcmd.EnvFromOS(), &stdout)
+	return Result{Stdout: stdout.String(), ExitCode: code}
+}
+
+func dispatchScrollbackRender(args []string) Result {
+	var stdout, stderr bytes.Buffer
+	code := scrollbackcmd.Run(args, &stdout, &stderr)
+	return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}
+}
+
 type LauncherRuntime struct {
 	Env      launcher.Env
 	Sessions launcher.SessionSource
@@ -162,15 +181,17 @@ func Help(program string) string {
 	var b strings.Builder
 	fmt.Fprintf(&b, "Usage: %s <command> [args]\n\n", program)
 	b.WriteString("Development dispatcher skeleton. Public sessions still start through bin/pair.\n\n")
-	b.WriteString("Implemented prototype commands:\n")
+	b.WriteString("Implemented commands:\n")
 	for _, family := range Families() {
 		if family.Status == "prototype" {
 			fmt.Fprintf(&b, "  %-17s %s (prototype; decision-phase only)\n", family.Name, family.Summary)
+		} else if family.Status == "implemented" {
+			fmt.Fprintf(&b, "  %-17s %s (implemented helper route)\n", family.Name, family.Summary)
 		}
 	}
 	b.WriteString("\nPlanned command families (not implemented in this skeleton):\n")
 	for _, family := range Families() {
-		if family.Status != "prototype" {
+		if family.Status == "planned" {
 			fmt.Fprintf(&b, "  %-17s %s (%s; not implemented in this skeleton)\n", family.Name, family.Summary, family.Status)
 		}
 	}
diff --git a/cmd/internal/dispatcher/dispatcher_test.go b/cmd/internal/dispatcher/dispatcher_test.go
index af3a573..f10f32b 100644
--- a/cmd/internal/dispatcher/dispatcher_test.go
+++ b/cmd/internal/dispatcher/dispatcher_test.go
@@ -1,6 +1,8 @@
 package dispatcher
 
 import (
+	"os"
+	"path/filepath"
 	"strings"
 	"testing"
 )
@@ -17,9 +19,11 @@ func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
 			}
 			for _, want := range []string{
 				"Usage: pair-go <command> [args]",
-				"Implemented prototype commands:",
+				"Implemented commands:",
 				"launch",
 				"decision-phase only",
+				"context",
+				"scrollback-render",
 				"wrap",
 				"slug",
 				"not implemented in this skeleton",
@@ -31,6 +35,14 @@ func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
 			if strings.Contains(res.Stdout, "launch             session lifecycle and public pair launcher flow (planned; not implemented") {
 				t.Fatalf("Stdout still labels launch unimplemented:\n%s", res.Stdout)
 			}
+			for _, stale := range []string{
+				"context           agent pane context meter (planned; not implemented",
+				"scrollback-render raw PTY capture to ANSI scrollback (planned; not implemented",
+			} {
+				if strings.Contains(res.Stdout, stale) {
+					t.Fatalf("Stdout still labels helper unimplemented (%q):\n%s", stale, res.Stdout)
+				}
+			}
 		})
 	}
 }
@@ -122,6 +134,36 @@ func TestDispatchLaunchWithoutArgsReturnsDefaultPrototypeDecision(t *testing.T)
 	}
 }
 
+func TestDispatchContextReturnsHelperOutput(t *testing.T) {
+	home, data := writeContextFixture(t)
+	t.Setenv("HOME", home)
+	t.Setenv("PAIR_DATA_DIR", data)
+
+	res := Dispatch([]string{"context", "T", "claude"})
+	if res.ExitCode != 0 {
+		t.Fatalf("ExitCode = %d, want 0; stderr:\n%s", res.ExitCode, res.Stderr)
+	}
+	if res.Stderr != "" {
+		t.Fatalf("Stderr = %q, want empty", res.Stderr)
+	}
+	if strings.TrimSpace(res.Stdout) != "398k" {
+		t.Fatalf("Stdout = %q, want 398k", res.Stdout)
+	}
+}
+
+func TestDispatchScrollbackRenderUsage(t *testing.T) {
+	res := Dispatch([]string{"scrollback-render"})
+	if res.ExitCode != 2 {
+		t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
+	}
+	if res.Stdout != "" {
+		t.Fatalf("Stdout = %q, want empty", res.Stdout)
+	}
+	if !strings.Contains(res.Stderr, "usage: pair-scrollback-render") {
+		t.Fatalf("Stderr missing usage:\n%s", res.Stderr)
+	}
+}
+
 func TestDispatchUnknownCommandReturnsUsageHint(t *testing.T) {
 	res := Dispatch([]string{"frobnicate"})
 	if res.ExitCode != 2 {
@@ -136,3 +178,34 @@ func TestDispatchUnknownCommandReturnsUsageHint(t *testing.T) {
 		}
 	}
 }
+
+func writeContextFixture(t *testing.T) (home, data string) {
+	t.Helper()
+	home = t.TempDir()
+	data = filepath.Join(home, "data")
+	cwd := filepath.Join(home, "repo")
+	enc := strings.NewReplacer(".", "-", "/", "-").Replace(cwd)
+	proj := filepath.Join(home, ".claude", "projects", enc)
+	mustMkdir(t, data)
+	mustMkdir(t, cwd)
+	mustMkdir(t, proj)
+	mustWrite(t, filepath.Join(data, "config-T-claude.json"), `{"session_id":"sid1"}`)
+	mustWrite(t, filepath.Join(data, "pane-T-claude.json"), `{"pane_id":"7","cwd":"`+cwd+`","cwd_display":"~/repo"}`)
+	mustWrite(t, filepath.Join(proj, "sid1.jsonl"),
+		`{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":397556,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`)
+	return home, data
+}
+
+func mustMkdir(t *testing.T, d string) {
+	t.Helper()
+	if err := os.MkdirAll(d, 0o755); err != nil {
+		t.Fatal(err)
+	}
+}
+
+func mustWrite(t *testing.T, p, s string) {
+	t.Helper()
+	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
+		t.Fatal(err)
+	}
+}
diff --git a/cmd/pair-scrollback-render/events_test.go b/cmd/internal/scrollbackcmd/events_test.go
similarity index 99%
rename from cmd/pair-scrollback-render/events_test.go
rename to cmd/internal/scrollbackcmd/events_test.go
index beec3ee..c932036 100644
--- a/cmd/pair-scrollback-render/events_test.go
+++ b/cmd/internal/scrollbackcmd/events_test.go
@@ -1,4 +1,4 @@
-package main
+package scrollbackcmd
 
 import (
 	"os"
diff --git a/cmd/pair-scrollback-render/render_test.go b/cmd/internal/scrollbackcmd/render_test.go
similarity index 99%
rename from cmd/pair-scrollback-render/render_test.go
rename to cmd/internal/scrollbackcmd/render_test.go
index b49fcf7..32153a0 100644
--- a/cmd/pair-scrollback-render/render_test.go
+++ b/cmd/internal/scrollbackcmd/render_test.go
@@ -1,4 +1,4 @@
-package main
+package scrollbackcmd
 
 import (
 	"fmt"
diff --git a/cmd/internal/scrollbackcmd/scrollbackcmd.go b/cmd/internal/scrollbackcmd/scrollbackcmd.go
new file mode 100644
index 0000000..9525323
--- /dev/null
+++ b/cmd/internal/scrollbackcmd/scrollbackcmd.go
@@ -0,0 +1,413 @@
+// scrollback-render — replay a pair-wrap raw capture through a VT100
+// emulator and write one ANSI-styled output line per logical scrollback row.
+// Drop-in replacement for the Python+pyte renderer at bin/pair-scrollback-render.
+//
+// Why Go: pyte's HistoryScreen dispatches every method call through an
+// __getattribute__ override that ran ~19M times for a 3 MB raw input —
+// ~95% of wall time. Even after the CaptureScreen patch (3.6x speedup),
+// the Python interpreter + pyte vendoring add startup cost and a private
+// venv that the brew formula has to manage. A static Go binary using
+// charmbracelet/x/vt replays the same stream with no runtime deps and
+// stays within the pair repo's existing cmd/ layout.
+//
+// Pipeline:
+//
+//	raw bytes (.raw)              → emulator.Write(...) in segments
+//	resize events (.events.jsonl) → segment boundaries with new (cols,rows)
+//	final emulator state          → scrollback lines + visible buffer
+//	each row                      → SGR-decorated text line written to out
+//
+// CLI is identical to the Python version so bin/pair-scrollback-open can
+// invoke either:
+//
+//	scrollback-render <raw> <events.jsonl> <out.ansi>
+package scrollbackcmd
+
+import (
+	"encoding/json"
+	"flag"
+	"fmt"
+	"io"
+	"math"
+	"os"
+	"strconv"
+	"strings"
+	"time"
+
+	uv "github.com/charmbracelet/ultraviolet"
+	"github.com/charmbracelet/x/vt"
+)
+
+// historyRows caps scrolled-out rows retained for the viewer. Matched
+// to zellij/config.kdl's `scroll_buffer_size 2000` so PageUp inside the
+// agent pane and Alt+/ both reach back the same distance — otherwise
+// the viewer would surface lines that zellij no longer has, leaving
+// the scroll-overlay logic in pair-scrollback-open unable to align
+// against the agent pane's frame.
+const historyRows = 2_000
+
+const (
+	defaultCols = 80
+	defaultRows = 24
+)
+
+type scrollbackEvent struct {
+	Type   string `json:"type"`
+	Offset int64  `json:"offset"`
+	Cols   int    `json:"cols"`
+	Rows   int    `json:"rows"`
+	Ts     string `json:"ts,omitempty"` // RFC3339 wall-clock for "time" events (#59)
+}
+
+// dateOf extracts the YYYY-MM-DD day from an RFC3339 timestamp; "" on a
+// malformed value so a corrupt time event degrades to undated, never panics (#59).
+func dateOf(ts string) string {
+	t, err := time.Parse(time.RFC3339, ts)
+	if err != nil {
+		return ""
+	}
+	return t.Format("2006-01-02")
+}
+
+// parseEvents reads the sidecar JSONL. Empty / missing file → empty slice.
+// Malformed lines are skipped so a corrupted tail doesn't abort the render —
+// imperfect width tracking beats an unusable viewer.
+func parseEvents(path string) ([]scrollbackEvent, error) {
+	data, err := os.ReadFile(path)
+	if err != nil {
+		if os.IsNotExist(err) {
+			return nil, nil
+		}
+		return nil, err
+	}
+	var out []scrollbackEvent
+	for _, line := range strings.Split(string(data), "\n") {
+		line = strings.TrimSpace(line)
+		if line == "" {
+			continue
+		}
+		var e scrollbackEvent
+		if err := json.Unmarshal([]byte(line), &e); err != nil {
+			continue
+		}
+		// Keep both known types: resize boundaries AND time stamps (#59).
+		// Consumers filter by Type at their use sites.
+		if e.Type == "resize" || e.Type == "time" {
+			out = append(out, e)
+		}
+	}
+	return out, nil
+}
+
+// initialSize pulls (cols, rows) from the first usable resize event, or
+// falls back to 80x24 if the sidecar is empty or the first entry is
+// malformed. pair-wrap always emits an initial resize at offset 0, so the
+// fallback only fires on a truncated file.
+func initialSize(events []scrollbackEvent) (int, int) {
+	for _, e := range events {
+		if e.Type == "resize" && e.Cols > 0 && e.Rows > 0 {
+			return e.Cols, e.Rows
+		}
+	}
+	return defaultCols, defaultRows
+}
+
+// dateMark records the emulator's scrollback length at a "time" event's byte
+// offset → the day that applies to committed lines from that index onward (#59).
+// Built during the feed (the only place that knows both byte offsets and the
+// rendered line count); consumed by the pure interleaveDateMarkers.
+type dateMark struct {
+	line int
+	date string
+}
+
+// feedSegments writes raw into the emulator as a single offset-ordered walk over
+// ALL sidecar events: write everything up to event.Offset, then act — Resize on a
+// resize event, or snapshot Scrollback().Len() on a time event. Returns the time
+// snapshots (empty unless time events are present). The caller already set the
+// initial size via initialSize; re-applying the offset-0 resize here is a harmless
+// no-op (resize to the current dimensions). Walking all events — rather than
+// events[1:] — means a time event in any position (incl. first) is captured, and
+// an empty events slice is handled without an out-of-range slice (#59).
+//
+// Clamping Offset to len(raw) defends against a corrupted sidecar that records
+// an offset beyond EOF (saw this once with a half-written events file after a
+// hard kill); without clamping we'd panic on the slice.
+func feedSegments(em *vt.Emulator, raw []byte, events []scrollbackEvent) []dateMark {
+	var cursor int64
+	var marks []dateMark
+	for _, e := range events {
+		off := e.Offset
+		if off > int64(len(raw)) {
+			off = int64(len(raw))
+		}
+		if off > cursor {
+			_, _ = em.Write(raw[cursor:off])
+			cursor = off
+		}
+		switch e.Type {
+		case "resize":
+			em.Resize(e.Cols, e.Rows)
+		case "time":
+			if d := dateOf(e.Ts); d != "" {
+				marks = append(marks, dateMark{line: em.Scrollback().Len(), date: d})
+			}
+		}
+	}
+	if cursor < int64(len(raw)) {
+		_, _ = em.Write(raw[cursor:])
+	}
+	return marks
+}
+
+// tsMarkerLine is the wire format the distiller parses (#59). MUST stay in sync
+// with tsMarkerRe in cmd/pair-changelog/distill.go — the contract is pinned by
+// the render→clean→distill e2e test cmd/pair-changelog/e2e_test.go
+// (TestEndToEndMarkerSurvival), which feeds real time events through both binaries.
+func tsMarkerLine(date string) string {
+	return "⟦pair:ts " + date + "⟧"
+}
+
+// interleaveDateMarkers inserts a tsMarkerLine immediately before the first line
+// of each new date run. marks are (scrollback-line-index, date) snapshots in
+// ascending index; a marker is emitted only when the applicable date *changes*
+// from the running date (consecutive same-date marks collapse). Lines before the
+// first mark stay undated; marks past len(lines) are ignored. Pure (#59).
+func interleaveDateMarkers(lines []string, marks []dateMark) []string {
+	if len(marks) == 0 {
+		return lines
+	}
+	out := make([]string, 0, len(lines)+len(marks))
+	mi := 0
+	prevDate := ""
+	for i := 0; i < len(lines); i++ {
+		curDate := prevDate
+		for mi < len(marks) && marks[mi].line <= i {
+			curDate = marks[mi].date
+			mi++
+		}
+		if curDate != "" && curDate != prevDate {
+			out = append(out, tsMarkerLine(curDate))
+			prevDate = curDate
+		}
+		out = append(out, lines[i])
+	}
+	return out
+}
+
+// serializeRow flattens one row into ANSI-styled text. Trims trailing
+// default-styled blanks (so the viewer doesn't scroll past pad), uses
+// Style.Diff so we only emit SGR codes when the style actually changes
+// between cells, and terminates with \x1b[0m to keep the row's last
+// style from bleeding into the next line if a viewer concatenates without
+// resetting between lines.
+//
+// A non-default background space is treated as visible content (e.g.
+// inverse-video padding). Matches what the Python renderer does.
+//
+// In plain mode (plain=true) no SGR is emitted at all: the row is just its
+// visible content, trimmed to the last non-blank-*content* cell — a cell that
+// is "visible" only via a non-default background (inverse-video padding, box
+// fill) is NOT emitted in plain mode, so it must not extend the row, or a
+// trailing bordered region would become space-padding toward terminal width.
+func serializeRow(line uv.Line, plain bool) string {
+	last := -1
+	for i := range line {
+		c := &line[i]
+		// Continuation cells of a preceding wide grapheme are stored as
+		// zero-value Cell{} per the ultraviolet convention (Width=0,
+		// Content=""). They don't extend the visible row and must not
+		// emit anything in the loop below.
+		if c.IsZero() {
+			continue
+		}
+		content := c.Content
+		if content != "" && content != " " {
+			last = i
+		} else if !plain && c.Style.Bg != nil {
+			last = i
+		}
+	}
+	if last < 0 {
+		return ""
+	}
+	var b strings.Builder
+	var prev uv.Style // zero value = default; Diff vs zero emits a reset
+	first := true
+	for i := 0; i <= last; i++ {
+		c := &line[i]
+		// Skip wide-grapheme continuation cells — the wide cell already
+		// emitted its full glyph; emitting anything here adds a phantom
+		// space after every emoji.
+		if c.IsZero() {
+			continue
+		}
+		if !plain && (first || !c.Style.Equal(&prev)) {
+			b.WriteString(c.Style.Diff(&prev))
+			prev = c.Style
+			first = false
+		}
+		if c.Content == "" {
+			b.WriteByte(' ')
+		} else {
+			b.WriteString(c.Content)
+		}
+	}
+	if !plain {
+		b.WriteString("\x1b[0m")
+	}
+	return b.String()
+}
+
+// resolveMax maps a --max-lines value to a scrollback cap. <=0 means
+// "uncapped" — a continuation wants the whole session, not the viewer's
+// 2000-row window. Represented as a large sentinel; .raw is per-run
+// O_TRUNC'd, so the practical bound is the run length.
+func resolveMax(n int) int {
+	if n <= 0 {
+		return math.MaxInt32
+	}
+	return n
+}
+
+// visibleRow materializes row y of the live screen as a uv.Line. The
+// emulator exposes cells one at a time via CellAt(x,y); there's no
+// "give me the whole row" accessor. A missing cell (CellAt returns nil)
+// becomes a zero-value Cell, which serializeRow treats as a blank.
+func visibleRow(em *vt.Emulator, y, width int) uv.Line {
+	row := make(uv.Line, width)
+	for x := 0; x < width; x++ {
+		if c := em.CellAt(x, y); c != nil {
+			row[x] = *c
+		}
+	}
+	return row
+}
+
+func render(rawPath, eventsPath, outPath string, plain bool, maxLines int, withTimestamps bool) error {
+	events, err := parseEvents(eventsPath)
+	if err != nil {
+		return fmt.Errorf("parse events: %w", err)
+	}
+	cols, rows := initialSize(events)
+	em := vt.NewEmulator(cols, rows)
+	em.Scrollback().SetMaxLines(resolveMax(maxLines))
+
+	// Drain the emulator's input pipe in the background. CSI status
+	// queries (DSR, Device Attributes, etc.) in the captured stream
+	// trigger handlers that *write a reply back* into this pipe — in a
+	// real terminal those bytes go to the controlling app. Offscreen
+	// replay has no reader, so the handler's WriteString blocks
+	// forever and deadlocks the Write goroutine. Discarding the bytes
+	// preserves emulation correctness; we never act on the replies.
+	//
+	// Wait for the drainer to actually exit before letting em.Close()
+	// run, otherwise Close races with the drainer's still-pending
+	// Read() (race detector catches it; in production the window is
+	// usually harmless but it's a real ordering bug).
+	drainDone := make(chan struct{})
+	go func() {
+		defer close(drainDone)
+		_, _ = io.Copy(io.Discard, em)
+	}()
+	defer func() {
+		em.Close()
+		<-drainDone
+	}()
+
+	raw, err := os.ReadFile(rawPath)
+	if err != nil {
+		return fmt.Errorf("read raw: %w", err)
+	}
+	marks := feedSegments(em, raw, events)
+
+	// Scrollback lines (oldest → newest), then visible buffer top → bottom.
+	// Visible buffer iterates by row index rather than dropping trailing
+	// blank rows: an agent that cleared and paused mid-redraw would shift
+	// every subsequent line number otherwise, and `:880` should still land
+	// where zellij showed line 880.
+	sb := em.Scrollback()
+	viewportTop := sb.Len() + 1 // 1-indexed line where the visible buffer starts
+	out := make([]string, 0, sb.Len()+em.Height())
+	for i := 0; i < sb.Len(); i++ {
+		out = append(out, serializeRow(sb.Line(i), plain))
+	}
+	w := em.Width()
+	for y := 0; y < em.Height(); y++ {
+		out = append(out, serializeRow(visibleRow(em, y, w), plain))
+	}
+	// Trim trailing all-blank lines: a half-empty visible buffer otherwise
+	// leaves a tail of empties at EOF.
+	for len(out) > 0 && out[len(out)-1] == "" {
+		out = out[:len(out)-1]
+	}
+
+	// Change-log path only: interleave day markers from the time-event snapshots
+	// so the distiller can date entries by real change-time (#59). Done after the
+	// trailing-blank trim so a marker never dangles past content. The scrollback
+	// viewer never sets this flag → its render is byte-identical to before.
+	if withTimestamps {
+		out = interleaveDateMarkers(out, marks)
+	}
+
+	// Write the viewport sidecar *first*, then atomically rename the
+	// .ansi into place. Order matters: scrollback.lua's BufReadPost
+	// opens the .ansi and immediately reads the sidecar — flipping the
+	// .ansi last guarantees the sidecar is up-to-date by the time
+	// nvim sees the new content. Sidecar is best-effort: on write
+	// failure, scrollback.lua falls back to its prior bottom-alignment.
+	// The viewport sidecar positions the Alt+/ nvim viewer; it's meaningless
+	// for the plain projection (a continuation distills the text, not a
+	// scroll position), so skip it and don't litter a stray <out>.viewport.
+	if !plain {
+		viewportPath := strings.TrimSuffix(outPath, ".ansi") + ".viewport"
+		_ = os.WriteFile(viewportPath, []byte(strconv.Itoa(viewportTop)+"\n"), 0o644)
+	}
+
+	// Atomic write so a double-tap Alt+/ can't race truncate-then-write
+	// on the same path. Reader sees either the old complete file or the
+	// new complete file, never a half-written one.
+	tmp := outPath + ".tmp"
+	f, err := os.Create(tmp)
+	if err != nil {
+		return err
+	}
+	body := strings.Join(out, "\n")
+	if len(out) > 0 {
+		body += "\n"
+	}
+	if _, err := f.WriteString(body); err != nil {
+		f.Close()
+		os.Remove(tmp)
+		return err
+	}
+	if err := f.Close(); err != nil {
+		os.Remove(tmp)
+		return err
+	}
+	return os.Rename(tmp, outPath)
+}
+
+func Run(argv []string, stdout, stderr io.Writer) int {
+	fs := flag.NewFlagSet("pair-scrollback-render", flag.ContinueOnError)
+	fs.SetOutput(stderr)
+	fs.Usage = func() {
+		fmt.Fprintf(stderr, "usage: pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] <raw> <events.jsonl> <out>\n")
+	}
+	plain := fs.Bool("plain", false, "emit plain text (no SGR) for distillation")
+	maxLines := fs.Int("max-lines", historyRows, "scrollback history rows retained; <=0 = uncapped")
+	withTimestamps := fs.Bool("with-timestamps", false, "interleave ⟦pair:ts DATE⟧ day markers from time events (for the change log; #59)")
+	if err := fs.Parse(argv); err != nil {
+		return 2
+	}
+	args := fs.Args()
+	if len(args) != 3 {
+		fs.Usage()
+		return 2
+	}
+	if err := render(args[0], args[1], args[2], *plain, *maxLines, *withTimestamps); err != nil {
+		fmt.Fprintf(stderr, "scrollback-render: %v\n", err)
+		return 1
+	}
+	return 0
+}
diff --git a/cmd/internal/scrollbackcmd/scrollbackcmd_test.go b/cmd/internal/scrollbackcmd/scrollbackcmd_test.go
new file mode 100644
index 0000000..db14e4b
--- /dev/null
+++ b/cmd/internal/scrollbackcmd/scrollbackcmd_test.go
@@ -0,0 +1,48 @@
+package scrollbackcmd
+
+import (
+	"bytes"
+	"io"
+	"os"
+	"path/filepath"
+	"strings"
+	"testing"
+)
+
+func TestRunUsage(t *testing.T) {
+	var stderr bytes.Buffer
+	code := Run([]string{}, io.Discard, &stderr)
+	if code != 2 {
+		t.Fatalf("code = %d, want 2", code)
+	}
+	if !strings.Contains(stderr.String(), "usage: pair-scrollback-render") {
+		t.Fatalf("stderr missing usage:\n%s", stderr.String())
+	}
+}
+
+func TestRunWritesOutput(t *testing.T) {
+	dir := t.TempDir()
+	rawPath := filepath.Join(dir, "in.raw")
+	evPath := filepath.Join(dir, "in.events.jsonl")
+	outPath := filepath.Join(dir, "out.ansi")
+	if err := os.WriteFile(rawPath, []byte("hello\r\n"), 0o644); err != nil {
+		t.Fatal(err)
+	}
+	events := `{"type":"resize","offset":0,"cols":20,"rows":5}` + "\n"
+	if err := os.WriteFile(evPath, []byte(events), 0o644); err != nil {
+		t.Fatal(err)
+	}
+
+	var stderr bytes.Buffer
+	code := Run([]string{rawPath, evPath, outPath}, io.Discard, &stderr)
+	if code != 0 {
+		t.Fatalf("code = %d, want 0; stderr:\n%s", code, stderr.String())
+	}
+	body, err := os.ReadFile(outPath)
+	if err != nil {
+		t.Fatalf("read out: %v", err)
+	}
+	if !strings.Contains(string(body), "hello") {
+		t.Fatalf("output missing rendered text:\n%s", string(body))
+	}
+}
diff --git a/cmd/pair-scrollback-render/serialize_row_test.go b/cmd/internal/scrollbackcmd/serialize_row_test.go
similarity index 99%
rename from cmd/pair-scrollback-render/serialize_row_test.go
rename to cmd/internal/scrollbackcmd/serialize_row_test.go
index 22050ab..1f53365 100644
--- a/cmd/pair-scrollback-render/serialize_row_test.go
+++ b/cmd/internal/scrollbackcmd/serialize_row_test.go
@@ -1,4 +1,4 @@
-package main
+package scrollbackcmd
 
 import (
 	"image/color"
diff --git a/cmd/pair-scrollback-render/timestamps_test.go b/cmd/internal/scrollbackcmd/timestamps_test.go
similarity index 99%
rename from cmd/pair-scrollback-render/timestamps_test.go
rename to cmd/internal/scrollbackcmd/timestamps_test.go
index 71a65aa..798f14e 100644
--- a/cmd/pair-scrollback-render/timestamps_test.go
+++ b/cmd/internal/scrollbackcmd/timestamps_test.go
@@ -1,4 +1,4 @@
-package main
+package scrollbackcmd
 
 import (
 	"fmt"
diff --git a/cmd/pair-context/main.go b/cmd/pair-context/main.go
index 92d1267..7b26f7c 100644
--- a/cmd/pair-context/main.go
+++ b/cmd/pair-context/main.go
@@ -5,57 +5,11 @@
 package main
 
 import (
-	"encoding/json"
-	"fmt"
 	"os"
-	"path/filepath"
 
-	"github.com/xianxu/pair/cmd/internal/ctxmeter"
-	"github.com/xianxu/pair/cmd/internal/transcript"
+	"github.com/xianxu/pair/cmd/internal/contextcmd"
 )
 
 func main() {
-	if len(os.Args) < 3 {
-		return
-	}
-	tag, agent := os.Args[1], os.Args[2]
-	dataDir := os.Getenv("PAIR_DATA_DIR")
-	if dataDir == "" {
-		base := os.Getenv("XDG_DATA_HOME")
-		if base == "" {
-			base = filepath.Join(os.Getenv("HOME"), ".local", "share")
-		}
-		dataDir = filepath.Join(base, "pair")
-	}
-	sid := transcript.SessionID(dataDir, tag, agent)
-	if sid == "" {
-		return
-	}
-	cwd := paneCwd(dataDir, tag, agent) // "" for codex/agy is fine (Resolve ignores it)
-	path := transcript.Resolve(agent, sid, cwd, os.Getenv("HOME"))
-	if path == "" {
-		return
-	}
-	f, err := os.Open(path)
-	if err != nil {
-		return
-	}
-	defer f.Close()
-	if n, ok := ctxmeter.ContextTokens(agent, f); ok {
-		fmt.Println(ctxmeter.Humanize(n))
-	}
-}
-
-func paneCwd(dataDir, tag, agent string) string {
-	b, err := os.ReadFile(filepath.Join(dataDir, "pane-"+tag+"-"+agent+".json"))
-	if err != nil {
-		return ""
-	}
-	var p struct {
-		Cwd string `json:"cwd"`
-	}
-	if json.Unmarshal(b, &p) != nil {
-		return ""
-	}
-	return p.Cwd
+	os.Exit(contextcmd.Run(os.Args[1:], contextcmd.EnvFromOS(), os.Stdout))
 }
diff --git a/cmd/pair-go/helper_equivalence_test.go b/cmd/pair-go/helper_equivalence_test.go
new file mode 100644
index 0000000..6905b86
--- /dev/null
+++ b/cmd/pair-go/helper_equivalence_test.go
@@ -0,0 +1,93 @@
+package main
+
+import (
+	"bytes"
+	"os"
+	"os/exec"
+	"path/filepath"
+	"strings"
+	"testing"
+)
+
+func TestPairGoContextMatchesLegacyPairContext(t *testing.T) {
+	bin := t.TempDir()
+	pairContext := filepath.Join(bin, "pair-context")
+	pairGo := filepath.Join(bin, "pair-go")
+	buildCommand(t, pairContext, "../pair-context")
+	buildCommand(t, pairGo, ".")
+
+	home, data := writeContextFixture(t)
+	env := append(os.Environ(), "HOME="+home, "PAIR_DATA_DIR="+data)
+
+	legacy := runCommand(t, env, pairContext, "T", "claude")
+	dispatch := runCommand(t, env, pairGo, "context", "T", "claude")
+	if dispatch.code != legacy.code || dispatch.stdout != legacy.stdout || dispatch.stderr != legacy.stderr {
+		t.Fatalf("pair-go context mismatch\nlegacy:   code=%d stdout=%q stderr=%q\ndispatch: code=%d stdout=%q stderr=%q",
+			legacy.code, legacy.stdout, legacy.stderr,
+			dispatch.code, dispatch.stdout, dispatch.stderr)
+	}
+}
+
+func buildCommand(t *testing.T, out, pkg string) {
+	t.Helper()
+	cmd := exec.Command("go", "build", "-o", out, pkg)
+	if body, err := cmd.CombinedOutput(); err != nil {
+		t.Fatalf("build %s: %v\n%s", pkg, err, string(body))
+	}
+}
+
+type commandResult struct {
+	code   int
+	stdout string
+	stderr string
+}
+
+func runCommand(t *testing.T, env []string, name string, args ...string) commandResult {
+	t.Helper()
+	cmd := exec.Command(name, args...)
+	cmd.Env = env
+	var stdout, stderr bytes.Buffer
+	cmd.Stdout = &stdout
+	cmd.Stderr = &stderr
+	err := cmd.Run()
+	code := 0
+	if err != nil {
+		exit, ok := err.(*exec.ExitError)
+		if !ok {
+			t.Fatalf("run %s: %v", name, err)
+		}
+		code = exit.ExitCode()
+	}
+	return commandResult{code: code, stdout: stdout.String(), stderr: stderr.String()}
+}
+
+func writeContextFixture(t *testing.T) (home, data string) {
+	t.Helper()
+	home = t.TempDir()
+	data = filepath.Join(home, "data")
+	cwd := filepath.Join(home, "repo")
+	enc := strings.NewReplacer(".", "-", "/", "-").Replace(cwd)
+	proj := filepath.Join(home, ".claude", "projects", enc)
+	mustMkdir(t, data)
+	mustMkdir(t, cwd)
+	mustMkdir(t, proj)
+	mustWrite(t, filepath.Join(data, "config-T-claude.json"), `{"session_id":"sid1"}`)
+	mustWrite(t, filepath.Join(data, "pane-T-claude.json"), `{"pane_id":"7","cwd":"`+cwd+`","cwd_display":"~/repo"}`)
+	mustWrite(t, filepath.Join(proj, "sid1.jsonl"),
+		`{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":397556,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`)
+	return home, data
+}
+
+func mustMkdir(t *testing.T, d string) {
+	t.Helper()
+	if err := os.MkdirAll(d, 0o755); err != nil {
+		t.Fatal(err)
+	}
+}
+
+func mustWrite(t *testing.T, p, s string) {
+	t.Helper()
+	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
+		t.Fatal(err)
+	}
+}
diff --git a/cmd/pair-scrollback-render/main.go b/cmd/pair-scrollback-render/main.go
index 57da8df..6e656be 100644
--- a/cmd/pair-scrollback-render/main.go
+++ b/cmd/pair-scrollback-render/main.go
@@ -1,408 +1,12 @@
-// scrollback-render — replay a pair-wrap raw capture through a VT100
-// emulator and write one ANSI-styled output line per logical scrollback row.
-// Drop-in replacement for the Python+pyte renderer at bin/pair-scrollback-render.
-//
-// Why Go: pyte's HistoryScreen dispatches every method call through an
-// __getattribute__ override that ran ~19M times for a 3 MB raw input —
-// ~95% of wall time. Even after the CaptureScreen patch (3.6x speedup),
-// the Python interpreter + pyte vendoring add startup cost and a private
-// venv that the brew formula has to manage. A static Go binary using
-// charmbracelet/x/vt replays the same stream with no runtime deps and
-// stays within the pair repo's existing cmd/ layout.
-//
-// Pipeline:
-//
-//	raw bytes (.raw)              → emulator.Write(...) in segments
-//	resize events (.events.jsonl) → segment boundaries with new (cols,rows)
-//	final emulator state          → scrollback lines + visible buffer
-//	each row                      → SGR-decorated text line written to out
-//
-// CLI is identical to the Python version so bin/pair-scrollback-open can
-// invoke either:
-//
-//	scrollback-render <raw> <events.jsonl> <out.ansi>
+// scrollback-render — replay a pair-wrap raw capture through a VT100 emulator.
 package main
 
 import (
-	"encoding/json"
-	"flag"
-	"fmt"
-	"io"
-	"math"
 	"os"
-	"strconv"
-	"strings"
-	"time"
 
-	uv "github.com/charmbracelet/ultraviolet"
-	"github.com/charmbracelet/x/vt"
+	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
 )
 
-// historyRows caps scrolled-out rows retained for the viewer. Matched
-// to zellij/config.kdl's `scroll_buffer_size 2000` so PageUp inside the
-// agent pane and Alt+/ both reach back the same distance — otherwise
-// the viewer would surface lines that zellij no longer has, leaving
-// the scroll-overlay logic in pair-scrollback-open unable to align
-// against the agent pane's frame.
-const historyRows = 2_000
-
-const (
-	defaultCols = 80
-	defaultRows = 24
-)
-
-type scrollbackEvent struct {
-	Type   string `json:"type"`
-	Offset int64  `json:"offset"`
-	Cols   int    `json:"cols"`
-	Rows   int    `json:"rows"`
-	Ts     string `json:"ts,omitempty"` // RFC3339 wall-clock for "time" events (#59)
-}
-
-// dateOf extracts the YYYY-MM-DD day from an RFC3339 timestamp; "" on a
-// malformed value so a corrupt time event degrades to undated, never panics (#59).
-func dateOf(ts string) string {
-	t, err := time.Parse(time.RFC3339, ts)
-	if err != nil {
-		return ""
-	}
-	return t.Format("2006-01-02")
-}
-
-// parseEvents reads the sidecar JSONL. Empty / missing file → empty slice.
-// Malformed lines are skipped so a corrupted tail doesn't abort the render —
-// imperfect width tracking beats an unusable viewer.
-func parseEvents(path string) ([]scrollbackEvent, error) {
-	data, err := os.ReadFile(path)
-	if err != nil {
-		if os.IsNotExist(err) {
-			return nil, nil
-		}
-		return nil, err
-	}
-	var out []scrollbackEvent
-	for _, line := range strings.Split(string(data), "\n") {
-		line = strings.TrimSpace(line)
-		if line == "" {
-			continue
-		}
-		var e scrollbackEvent
-		if err := json.Unmarshal([]byte(line), &e); err != nil {
-			continue
-		}
-		// Keep both known types: resize boundaries AND time stamps (#59).
-		// Consumers filter by Type at their use sites.
-		if e.Type == "resize" || e.Type == "time" {
-			out = append(out, e)
-		}
-	}
-	return out, nil
-}
-
-// initialSize pulls (cols, rows) from the first usable resize event, or
-// falls back to 80x24 if the sidecar is empty or the first entry is
-// malformed. pair-wrap always emits an initial resize at offset 0, so the
-// fallback only fires on a truncated file.
-func initialSize(events []scrollbackEvent) (int, int) {
-	for _, e := range events {
-		if e.Type == "resize" && e.Cols > 0 && e.Rows > 0 {
-			return e.Cols, e.Rows
-		}
-	}
-	return defaultCols, defaultRows
-}
-
-// dateMark records the emulator's scrollback length at a "time" event's byte
-// offset → the day that applies to committed lines from that index onward (#59).
-// Built during the feed (the only place that knows both byte offsets and the
-// rendered line count); consumed by the pure interleaveDateMarkers.
-type dateMark struct {
-	line int
-	date string
-}
-
-// feedSegments writes raw into the emulator as a single offset-ordered walk over
-// ALL sidecar events: write everything up to event.Offset, then act — Resize on a
-// resize event, or snapshot Scrollback().Len() on a time event. Returns the time
-// snapshots (empty unless time events are present). The caller already set the
-// initial size via initialSize; re-applying the offset-0 resize here is a harmless
-// no-op (resize to the current dimensions). Walking all events — rather than
-// events[1:] — means a time event in any position (incl. first) is captured, and
-// an empty events slice is handled without an out-of-range slice (#59).
-//
-// Clamping Offset to len(raw) defends against a corrupted sidecar that records
-// an offset beyond EOF (saw this once with a half-written events file after a
-// hard kill); without clamping we'd panic on the slice.
-func feedSegments(em *vt.Emulator, raw []byte, events []scrollbackEvent) []dateMark {
-	var cursor int64
-	var marks []dateMark
-	for _, e := range events {
-		off := e.Offset
-		if off > int64(len(raw)) {
-			off = int64(len(raw))
-		}
-		if off > cursor {
-			_, _ = em.Write(raw[cursor:off])
-			cursor = off
-		}
-		switch e.Type {
-		case "resize":
-			em.Resize(e.Cols, e.Rows)
-		case "time":
-			if d := dateOf(e.Ts); d != "" {
-				marks = append(marks, dateMark{line: em.Scrollback().Len(), date: d})
-			}
-		}
-	}
-	if cursor < int64(len(raw)) {
-		_, _ = em.Write(raw[cursor:])
-	}
-	return marks
-}
-
-// tsMarkerLine is the wire format the distiller parses (#59). MUST stay in sync
-// with tsMarkerRe in cmd/pair-changelog/distill.go — the contract is pinned by
-// the render→clean→distill e2e test cmd/pair-changelog/e2e_test.go
-// (TestEndToEndMarkerSurvival), which feeds real time events through both binaries.
-func tsMarkerLine(date string) string {
-	return "⟦pair:ts " + date + "⟧"
-}
-
-// interleaveDateMarkers inserts a tsMarkerLine immediately before the first line
-// of each new date run. marks are (scrollback-line-index, date) snapshots in
-// ascending index; a marker is emitted only when the applicable date *changes*
-// from the running date (consecutive same-date marks collapse). Lines before the
-// first mark stay undated; marks past len(lines) are ignored. Pure (#59).
-func interleaveDateMarkers(lines []string, marks []dateMark) []string {
-	if len(marks) == 0 {
-		return lines
-	}
-	out := make([]string, 0, len(lines)+len(marks))
-	mi := 0
-	prevDate := ""
-	for i := 0; i < len(lines); i++ {
-		curDate := prevDate
-		for mi < len(marks) && marks[mi].line <= i {
-			curDate = marks[mi].date
-			mi++
-		}
-		if curDate != "" && curDate != prevDate {
-			out = append(out, tsMarkerLine(curDate))
-			prevDate = curDate
-		}
-		out = append(out, lines[i])
-	}
-	return out
-}
-
-// serializeRow flattens one row into ANSI-styled text. Trims trailing
-// default-styled blanks (so the viewer doesn't scroll past pad), uses
-// Style.Diff so we only emit SGR codes when the style actually changes
-// between cells, and terminates with \x1b[0m to keep the row's last
-// style from bleeding into the next line if a viewer concatenates without
-// resetting between lines.
-//
-// A non-default background space is treated as visible content (e.g.
-// inverse-video padding). Matches what the Python renderer does.
-//
-// In plain mode (plain=true) no SGR is emitted at all: the row is just its
-// visible content, trimmed to the last non-blank-*content* cell — a cell that
-// is "visible" only via a non-default background (inverse-video padding, box
-// fill) is NOT emitted in plain mode, so it must not extend the row, or a
-// trailing bordered region would become space-padding toward terminal width.
-func serializeRow(line uv.Line, plain bool) string {
-	last := -1
-	for i := range line {
-		c := &line[i]
-		// Continuation cells of a preceding wide grapheme are stored as
-		// zero-value Cell{} per the ultraviolet convention (Width=0,
-		// Content=""). They don't extend the visible row and must not
-		// emit anything in the loop below.
-		if c.IsZero() {
-			continue
-		}
-		content := c.Content
-		if content != "" && content != " " {
-			last = i
-		} else if !plain && c.Style.Bg != nil {
-			last = i
-		}
-	}
-	if last < 0 {
-		return ""
-	}
-	var b strings.Builder
-	var prev uv.Style // zero value = default; Diff vs zero emits a reset
-	first := true
-	for i := 0; i <= last; i++ {
-		c := &line[i]
-		// Skip wide-grapheme continuation cells — the wide cell already
-		// emitted its full glyph; emitting anything here adds a phantom
-		// space after every emoji.
-		if c.IsZero() {
-			continue
-		}
-		if !plain && (first || !c.Style.Equal(&prev)) {
-			b.WriteString(c.Style.Diff(&prev))
-			prev = c.Style
-			first = false
-		}
-		if c.Content == "" {
-			b.WriteByte(' ')
-		} else {
-			b.WriteString(c.Content)
-		}
-	}
-	if !plain {
-		b.WriteString("\x1b[0m")
-	}
-	return b.String()
-}
-
-// resolveMax maps a --max-lines value to a scrollback cap. <=0 means
-// "uncapped" — a continuation wants the whole session, not the viewer's
-// 2000-row window. Represented as a large sentinel; .raw is per-run
-// O_TRUNC'd, so the practical bound is the run length.
-func resolveMax(n int) int {
-	if n <= 0 {
-		return math.MaxInt32
-	}
-	return n
-}
-
-// visibleRow materializes row y of the live screen as a uv.Line. The
-// emulator exposes cells one at a time via CellAt(x,y); there's no
-// "give me the whole row" accessor. A missing cell (CellAt returns nil)
-// becomes a zero-value Cell, which serializeRow treats as a blank.
-func visibleRow(em *vt.Emulator, y, width int) uv.Line {
-	row := make(uv.Line, width)
-	for x := 0; x < width; x++ {
-		if c := em.CellAt(x, y); c != nil {
-			row[x] = *c
-		}
-	}
-	return row
-}
-
-func render(rawPath, eventsPath, outPath string, plain bool, maxLines int, withTimestamps bool) error {
-	events, err := parseEvents(eventsPath)
-	if err != nil {
-		return fmt.Errorf("parse events: %w", err)
-	}
-	cols, rows := initialSize(events)
-	em := vt.NewEmulator(cols, rows)
-	em.Scrollback().SetMaxLines(resolveMax(maxLines))
-
-	// Drain the emulator's input pipe in the background. CSI status
-	// queries (DSR, Device Attributes, etc.) in the captured stream
-	// trigger handlers that *write a reply back* into this pipe — in a
-	// real terminal those bytes go to the controlling app. Offscreen
-	// replay has no reader, so the handler's WriteString blocks
-	// forever and deadlocks the Write goroutine. Discarding the bytes
-	// preserves emulation correctness; we never act on the replies.
-	//
-	// Wait for the drainer to actually exit before letting em.Close()
-	// run, otherwise Close races with the drainer's still-pending
-	// Read() (race detector catches it; in production the window is
-	// usually harmless but it's a real ordering bug).
-	drainDone := make(chan struct{})
-	go func() {
-		defer close(drainDone)
-		_, _ = io.Copy(io.Discard, em)
-	}()
-	defer func() {
-		em.Close()
-		<-drainDone
-	}()
-
-	raw, err := os.ReadFile(rawPath)
-	if err != nil {
-		return fmt.Errorf("read raw: %w", err)
-	}
-	marks := feedSegments(em, raw, events)
-
-	// Scrollback lines (oldest → newest), then visible buffer top → bottom.
-	// Visible buffer iterates by row index rather than dropping trailing
-	// blank rows: an agent that cleared and paused mid-redraw would shift
-	// every subsequent line number otherwise, and `:880` should still land
-	// where zellij showed line 880.
-	sb := em.Scrollback()
-	viewportTop := sb.Len() + 1 // 1-indexed line where the visible buffer starts
-	out := make([]string, 0, sb.Len()+em.Height())
-	for i := 0; i < sb.Len(); i++ {
-		out = append(out, serializeRow(sb.Line(i), plain))
-	}
-	w := em.Width()
-	for y := 0; y < em.Height(); y++ {
-		out = append(out, serializeRow(visibleRow(em, y, w), plain))
-	}
-	// Trim trailing all-blank lines: a half-empty visible buffer otherwise
-	// leaves a tail of empties at EOF.
-	for len(out) > 0 && out[len(out)-1] == "" {
-		out = out[:len(out)-1]
-	}
-
-	// Change-log path only: interleave day markers from the time-event snapshots
-	// so the distiller can date entries by real change-time (#59). Done after the
-	// trailing-blank trim so a marker never dangles past content. The scrollback
-	// viewer never sets this flag → its render is byte-identical to before.
-	if withTimestamps {
-		out = interleaveDateMarkers(out, marks)
-	}
-
-	// Write the viewport sidecar *first*, then atomically rename the
-	// .ansi into place. Order matters: scrollback.lua's BufReadPost
-	// opens the .ansi and immediately reads the sidecar — flipping the
-	// .ansi last guarantees the sidecar is up-to-date by the time
-	// nvim sees the new content. Sidecar is best-effort: on write
-	// failure, scrollback.lua falls back to its prior bottom-alignment.
-	// The viewport sidecar positions the Alt+/ nvim viewer; it's meaningless
-	// for the plain projection (a continuation distills the text, not a
-	// scroll position), so skip it and don't litter a stray <out>.viewport.
-	if !plain {
-		viewportPath := strings.TrimSuffix(outPath, ".ansi") + ".viewport"
-		_ = os.WriteFile(viewportPath, []byte(strconv.Itoa(viewportTop)+"\n"), 0o644)
-	}
-
-	// Atomic write so a double-tap Alt+/ can't race truncate-then-write
-	// on the same path. Reader sees either the old complete file or the
-	// new complete file, never a half-written one.
-	tmp := outPath + ".tmp"
-	f, err := os.Create(tmp)
-	if err != nil {
-		return err
-	}
-	body := strings.Join(out, "\n")
-	if len(out) > 0 {
-		body += "\n"
-	}
-	if _, err := f.WriteString(body); err != nil {
-		f.Close()
-		os.Remove(tmp)
-		return err
-	}
-	if err := f.Close(); err != nil {
-		os.Remove(tmp)
-		return err
-	}
-	return os.Rename(tmp, outPath)
-}
-
 func main() {
-	flag.Usage = func() {
-		fmt.Fprintf(os.Stderr, "usage: %s [--plain] [--max-lines N] [--with-timestamps] <raw> <events.jsonl> <out>\n", os.Args[0])
-	}
-	plain := flag.Bool("plain", false, "emit plain text (no SGR) for distillation")
-	maxLines := flag.Int("max-lines", historyRows, "scrollback history rows retained; <=0 = uncapped")
-	withTimestamps := flag.Bool("with-timestamps", false, "interleave ⟦pair:ts DATE⟧ day markers from time events (for the change log; #59)")
-	flag.Parse()
-	args := flag.Args()
-	if len(args) != 3 {
-		flag.Usage()
-		os.Exit(2)
-	}
-	if err := render(args[0], args[1], args[2], *plain, *maxLines, *withTimestamps); err != nil {
-		fmt.Fprintf(os.Stderr, "scrollback-render: %v\n", err)
-		os.Exit(1)
-	}
+	os.Exit(scrollbackcmd.Run(os.Args[1:], os.Stdout, os.Stderr))
 }
diff --git a/workshop/plans/000076-go-helper-dispatch-plan.md b/workshop/plans/000076-go-helper-dispatch-plan.md
index 88f9856..5a07fe8 100644
--- a/workshop/plans/000076-go-helper-dispatch-plan.md
+++ b/workshop/plans/000076-go-helper-dispatch-plan.md
@@ -12,31 +12,35 @@
 
 ## Core Concepts
 
-### Pure Entities
-
-| Name | Lives in | Status |
-|------|----------|--------|
-| `ContextRunArgs` | `cmd/internal/contextcmd/contextcmd.go` | new |
-| `ScrollbackRenderArgs` | `cmd/internal/scrollbackcmd/scrollbackcmd.go` | new |
-
-**ContextRunArgs** — Parsed input for the context helper: tag, agent, home, and Pair data dir.
-- **Relationships:** 1:1 with a `pair-context`/`pair-go context` invocation; consumed by the context runner.
-- **DRY rationale:** Both legacy binary and dispatcher route need the same env/default resolution and argument validation.
+### Command Surfaces
+
+| Name | Kind | Lives in | Status |
+|------|------|----------|--------|
+| `contextcmd.Env` | INTEGRATION input value | `cmd/internal/contextcmd/contextcmd.go` | new |
+| `contextcmd.Run` | INTEGRATION | `cmd/internal/contextcmd/contextcmd.go` | new |
+| `scrollbackcmd.Run` | INTEGRATION | `cmd/internal/scrollbackcmd/scrollbackcmd.go` | new |
+| `dispatcher.Dispatch` | INTEGRATION | `cmd/internal/dispatcher/dispatcher.go` | modified |
+
+**contextcmd.Env** — Injected environment values for the context helper: home, XDG data home, and explicit Pair data dir.
+- **Relationships:** Passed to `contextcmd.Run`; populated from `EnvFromOS` only at the process boundary.
+- **DRY rationale:** Both legacy binary and dispatcher route need the same env/default resolution without re-reading OS env in tests.
 - **Future extensions:** Can widen if more commands need shared Pair environment resolution.
 
-**ScrollbackRenderArgs** — Parsed input for the renderer helper: raw capture path, events path, output path, and render flags.
-- **Relationships:** 1:1 with a render invocation; maps directly onto existing render parameters.
-- **DRY rationale:** Avoids parallel flag parsing between `pair-scrollback-render` and `pair-go scrollback-render`.
-- **Future extensions:** Can become the command-facing shape for future `pair scrollback-render` after the public entrypoint switch.
+**contextcmd.Run** — Shared command runner for `pair-context <tag> <agent>` behavior. It remains tolerant: missing config/transcript/input prints nothing and exits 0.
+- **Injected into:** legacy `cmd/pair-context/main.go` and dispatcher context route.
+- **Future extensions:** The title poller can continue using the legacy binary until #77/#78 moves call sites.
+
+**scrollbackcmd.Run** — Shared command runner for `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out`.
+- **Injected into:** legacy `cmd/pair-scrollback-render/main.go` and dispatcher scrollback-render route.
+- **Future extensions:** `bin/pair-scrollback-open`, `bin/pair-changelog-open`, and `nvim/scrollback.lua` can move to the dispatcher after the public entrypoint is Go-owned.
 
 ### Integration Points
 
 | Name | Lives in | Status | Wraps |
 |------|----------|--------|-------|
-| `ContextRunner` | `cmd/internal/contextcmd/contextcmd.go` | new | filesystem, env, transcript files, stdout |
-| `ScrollbackRenderRunner` | `cmd/internal/scrollbackcmd/scrollbackcmd.go` | new | filesystem, flag parsing, stdout/stderr |
-| `HelperDispatchRoutes` | `cmd/internal/dispatcher/dispatcher.go` | modified | in-process calls to selected helper runners |
-| `PairGoMain` | `cmd/pair-go/main.go` | modified | process stdout/stderr exit handling |
+| `ContextRunner` (`contextcmd.Run`) | `cmd/internal/contextcmd/contextcmd.go` | new | filesystem, env, transcript files, stdout |
+| `ScrollbackRenderRunner` (`scrollbackcmd.Run`) | `cmd/internal/scrollbackcmd/scrollbackcmd.go` | new | filesystem, flag parsing, stdout/stderr |
+| `dispatcher.Dispatch` | `cmd/internal/dispatcher/dispatcher.go` | modified | in-process calls to selected helper runners |
 
 **ContextRunner** — Shared command runner for `pair-context <tag> <agent>` behavior. It remains tolerant: missing config/transcript/input prints nothing and exits 0.
 - **Injected into:** legacy `cmd/pair-context/main.go` and dispatcher context route.
@@ -46,13 +50,13 @@
 - **Injected into:** legacy `cmd/pair-scrollback-render/main.go` and dispatcher scrollback-render route.
 - **Future extensions:** `bin/pair-scrollback-open`, `bin/pair-changelog-open`, and `nvim/scrollback.lua` can move to the dispatcher after the public entrypoint is Go-owned.
 
-**HelperDispatchRoutes** — Dispatcher routes for `context` and `scrollback-render`.
+**dispatcher.Dispatch** — Dispatcher routes for `context` and `scrollback-render`.
 - **Injected into:** `cmd/pair-go`.
 - **Future extensions:** Later helper routes should follow the same runner extraction pattern, not duplicate command logic.
 
-**PairGoMain** — Existing process shell that writes dispatcher results.
-- **Injected into:** none.
-- **Future extensions:** Eventually becomes the public `pair` entrypoint in #77, but not here.
+## Revisions
+
+- 2026-06-30T12:24:41-0700 — Boundary review found the Core Concepts table still named planned parsed-args structs that the implementation did not need. Replaced `ContextRunArgs`/`ScrollbackRenderArgs` with the shipped `contextcmd.Env`, `contextcmd.Run`, and `scrollbackcmd.Run` surfaces, and classified the runners as INTEGRATION rather than PURE entities.
 
 ---
 
@@ -67,7 +71,7 @@
 - Modify: `cmd/pair-context/main_test.go`
 - Modify: `Makefile.local`
 
-- [ ] **Step 1: Add failing runner tests**
+- [x] **Step 1: Add failing runner tests**
 
 Create `cmd/internal/contextcmd/contextcmd_test.go` with tests that call `Run(args []string, env Env, stdout io.Writer) int` directly:
 
@@ -84,13 +88,13 @@ func TestRunMissingConfigPrintsNothing(t *testing.T) {
 }
 ```
 
-- [ ] **Step 2: Run the focused tests and confirm they fail**
+- [x] **Step 2: Run the focused tests and confirm they fail**
 
 Run: `go test ./cmd/internal/contextcmd -count=1`
 
 Expected: FAIL because the package does not exist yet.
 
-- [ ] **Step 3: Extract the runner**
+- [x] **Step 3: Extract the runner**
 
 Move the reusable context behavior into `cmd/internal/contextcmd`. Keep `cmd/pair-context/main.go` as:
 
@@ -106,13 +110,13 @@ The runner must:
 - write the same humanized token count to the injected stdout;
 - never call `os.Exit`.
 
-- [ ] **Step 4: Run the focused tests and existing package tests**
+- [x] **Step 4: Run the focused tests and existing package tests**
 
 Run: `go test ./cmd/internal/contextcmd ./cmd/pair-context -count=1`
 
 Expected: PASS.
 
-- [ ] **Step 5: Commit the context extraction**
+- [x] **Step 5: Commit the context extraction**
 
 Run:
 
@@ -134,7 +138,7 @@ git commit -m "#76: extract pair-context runner"
 - Modify or create tests in: `cmd/pair-scrollback-render/*_test.go`
 - Modify: `Makefile.local`
 
-- [ ] **Step 1: Add failing runner tests**
+- [x] **Step 1: Add failing runner tests**
 
 Create tests that call `scrollbackcmd.Run(args []string, stdout, stderr io.Writer) int`:
 
@@ -152,13 +156,13 @@ func TestRunWritesOutput(t *testing.T) {
 }
 ```
 
-- [ ] **Step 2: Run the focused tests and confirm they fail**
+- [x] **Step 2: Run the focused tests and confirm they fail**
 
 Run: `go test ./cmd/internal/scrollbackcmd -count=1`
 
 Expected: FAIL because the package does not exist yet.
 
-- [ ] **Step 3: Extract the runner**
+- [x] **Step 3: Extract the runner**
 
 Move the renderer command wrapper into `cmd/internal/scrollbackcmd`. If `render(...)` cannot be imported from `package main`, move the rendering core into this internal package too and leave the legacy command as a tiny wrapper:
 
@@ -174,13 +178,13 @@ Use a local `flag.FlagSet` so dispatcher and tests can parse independently. Pres
 - exit 0 for success;
 - same defaults for `--plain`, `--max-lines`, and `--with-timestamps`.
 
-- [ ] **Step 4: Run focused and package tests**
+- [x] **Step 4: Run focused and package tests**
 
 Run: `go test ./cmd/internal/scrollbackcmd ./cmd/pair-scrollback-render -count=1`
 
 Expected: PASS.
 
-- [ ] **Step 5: Commit the renderer extraction**
+- [x] **Step 5: Commit the renderer extraction**
 
 Run:
 
@@ -202,20 +206,20 @@ git commit -m "#76: extract scrollback renderer runner"
 - Create: `cmd/pair-go/helper_equivalence_test.go`
 - Modify: `Makefile.local`
 
-- [ ] **Step 1: Add failing dispatcher tests**
+- [x] **Step 1: Add failing dispatcher tests**
 
 Add tests for:
 - `Dispatch([]string{"context", "T", "claude"})` through a fake or temp fixture returns the same output shape as `contextcmd.Run`;
 - `Dispatch([]string{"scrollback-render"})` returns renderer usage with exit 2;
 - top-level help lists `context` and `scrollback-render` as implemented helper routes, not planned-only commands.
 
-- [ ] **Step 2: Run dispatcher tests and confirm failure**
+- [x] **Step 2: Run dispatcher tests and confirm failure**
 
 Run: `go test ./cmd/internal/dispatcher -run 'TestDispatch(Context|Scrollback|Help)' -count=1`
 
 Expected: FAIL because routes are not implemented yet.
 
-- [ ] **Step 3: Add failing process-level equivalence test**
+- [x] **Step 3: Add failing process-level equivalence test**
 
 Create `cmd/pair-go/helper_equivalence_test.go` with a context helper fixture that builds both commands and runs them against the same temp Pair data/transcript tree:
 
@@ -231,13 +235,13 @@ func TestPairGoContextMatchesLegacyPairContext(t *testing.T) {
 
 This is the representative compatibility proof required by the issue done-when. It should fail before the dispatcher route exists because `pair-go context` is still planned-only.
 
-- [ ] **Step 4: Run the equivalence test and confirm failure**
+- [x] **Step 4: Run the equivalence test and confirm failure**
 
 Run: `go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1`
 
 Expected: FAIL because `pair-go context` returns planned-but-not-implemented.
 
-- [ ] **Step 5: Implement routes**
+- [x] **Step 5: Implement routes**
 
 Update `Families()` statuses for selected helpers and add dispatcher branches:
 
@@ -250,7 +254,7 @@ case "scrollback-render":
 
 The dispatcher should continue returning `dispatcher.Result`. Helper runners should write into buffers so dispatcher can map stdout/stderr/exit code without adding a second process-result abstraction.
 
-- [ ] **Step 6: Run route and process tests**
+- [x] **Step 6: Run route and process tests**
 
 Run:
 
@@ -261,7 +265,7 @@ make pair-context pair-scrollback-render pair-go
 
 Expected: PASS. The `make` command is deliberately not `-B`; it verifies the updated dependency graph can rebuild normally after source changes.
 
-- [ ] **Step 7: Commit dispatcher wiring**
+- [x] **Step 7: Commit dispatcher wiring**
 
 Run:
 
@@ -282,7 +286,7 @@ git commit -m "#76: route selected helpers through pair-go"
 - Modify: `workshop/issues/000076-go-helper-dispatch.md`
 - Modify: `Makefile.local`
 
-- [ ] **Step 1: Verify legacy binaries still build**
+- [x] **Step 1: Verify legacy binaries still build**
 
 Run:
 
@@ -293,7 +297,7 @@ make -B pair-context pair-scrollback-render pair-go
 
 Expected: PASS. The non-`-B` run verifies incremental prerequisites include `cmd/internal/contextcmd`, `cmd/internal/scrollbackcmd`, and dispatcher dependencies; the `-B` run remains the forced clean rebuild check.
 
-- [ ] **Step 2: Verify selected command equivalence**
+- [x] **Step 2: Verify selected command equivalence**
 
 Run focused commands against test fixtures or package tests:
 
@@ -304,13 +308,13 @@ go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1
 
 Expected: PASS; the equivalence test demonstrates the legacy `pair-context` binary and `pair-go context` process path produce identical stdout/stderr/exit code on the same fixture.
 
-- [ ] **Step 3: Verify full Go test suite**
+- [x] **Step 3: Verify full Go test suite**
 
 Run: `go test ./... -count=1`
 
 Expected: PASS.
 
-- [ ] **Step 4: Verify no live call sites moved**
+- [x] **Step 4: Verify no live call sites moved**
 
 Run:
 
@@ -320,17 +324,17 @@ git diff -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollb
 
 Expected: empty diff, unless a test-only or documentation-only change was explicitly made.
 
-- [ ] **Step 5: Update atlas**
+- [x] **Step 5: Update atlas**
 
 Update:
 - `atlas/architecture.md` to say `pair-go context` and `pair-go scrollback-render` are implemented helper routes while public launcher and live zellij/nvim callers remain legacy.
 - `atlas/go-migration-inventory.md` rows for `pair-context` and `pair-scrollback-render` to record dispatcher availability and preserved legacy names.
 
-- [ ] **Step 6: Update issue checklist and log**
+- [x] **Step 6: Update issue checklist and log**
 
 Tick the #76 plan/done items that are complete and add a log entry with verification commands and `ARCH-*` notes.
 
-- [ ] **Step 7: Run final verification before close**
+- [x] **Step 7: Run final verification before close**
 
 Run:
 


codex
I’ll use `superpowers-using-superpowers` for the required session bootstrap, then treat this as a read-only boundary review and verify the diff against the tracker and code.
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
I’ve got the workflow contract. I’m reading the issue, plan, lessons, and the changed files at `HEAD` now so the review is based on the repository state rather than the prompt diff alone.
exec
/bin/zsh -lc "sed -n '1,260p' workshop/issues/000076-go-helper-dispatch.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
---
id: 000076
status: done
deps: [000074]
github_issue:
created: 2026-06-26
updated: 2026-06-30
estimate_hours: 2.86
started: 2026-06-30T11:58:44-07:00
actual_hours: 0.54
---

# pair Go helper dispatch

## Problem

Pair already has several Go helpers, but packaging still exposes them as separate binaries in `bin/`. A single-primary-binary architecture should route those helpers through `pair` without copying code or breaking existing callers.

## Spec

Fold the first low-risk Go helper commands behind the dispatcher introduced by #74. This milestone proves the shared-runner pattern without moving live zellij/nvim call sites yet.

- `pair context`
- `pair scrollback-render`

The old binary names must continue to build and work during this milestone. Existing zellij/nvim/script callers should not change yet; `bin/pair-title.sh`, `bin/pair-scrollback-open`, `bin/pair-changelog-open`, and `nvim/scrollback.lua` keep invoking the legacy command names.

Implementation should extract shared run functions from the existing `package main` commands so both the legacy binaries and `pair-go <subcommand>` routes call the same behavior (`ARCH-DRY`). The dispatcher remains the only process-facing result abstraction for in-process helper routes; helper packages should write to injected stdout/stderr and return integer exit codes or errors rather than calling `os.Exit` in reusable code (`ARCH-PURE`).

Out of scope for this milestone: `pair wrap`, `pair slug`, `pair changelog`, `pair continuation`, `pair scribe`, public `pair` entrypoint changes, install alias changes, and zellij/nvim call-site rewrites. Those remain candidates for later issues once the low-risk dispatch pattern is proven.

## Done when

- [x] Dispatcher can invoke selected existing Go helpers through `pair-go <subcommand>`.
- [x] Existing helper binary names still build and work.
- [x] Tests prove dispatch and legacy command paths reach the same behavior for at least one representative helper.
- [x] No zellij/nvim keybinding breaks.
- [x] Pair remains usable after merge.

## Plan

- [x] Choose the first helper set based on #73.
- [x] Extract reusable run functions for `pair-context` and `pair-scrollback-render`.
- [x] Add dispatcher routes for `context` and `scrollback-render`.
- [x] Preserve legacy binary names.
- [x] Run helper-specific and full relevant integration tests.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.60 impl=0.45
item: skill-or-dispatcher design=0.30 impl=0.30
item: smaller-go-module design=0.40 impl=0.25
item: atlas-docs design=0.10 impl=0.05
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 2.86
```

## Log

### 2026-06-26

Created from #72. This milestone reduces packaging surface while preserving current command names.

### 2026-06-30
- 2026-06-30: closed — go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1; go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1; make pair-context pair-scrollback-render pair-go; make -B pair-context pair-scrollback-render pair-go; go test ./... -count=1; git diff -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollback-open bin/pair-changelog-open empty; rg atlas helper dispatch; git diff --check

Claimed after #75 landed. Narrowed the first helper dispatch slice to `context` and `scrollback-render`: they are useful enough to prove the dispatcher path, but low-risk enough to avoid long-running PTY, model, git commit/push, or public launcher behavior. Existing zellij/nvim/shell callers stay on legacy binary names for this milestone (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`).

Extracted `cmd/internal/contextcmd` and `cmd/internal/scrollbackcmd` runners so legacy binaries and dispatcher routes share implementation (`ARCH-DRY`, `ARCH-PURE`). Added `pair-go context` and `pair-go scrollback-render` dispatcher routes plus a process-level equivalence test proving `pair-go context` matches `pair-context` stdout/stderr/exit code on the same fixture. Updated atlas to record the current helper-dispatch state and unchanged live shell/Lua callers.

Verification before close: `go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1`; `go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1`; `make pair-context pair-scrollback-render pair-go`; `make -B pair-context pair-scrollback-render pair-go`; `go test ./... -count=1`; `git diff -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollback-open bin/pair-changelog-open` empty; atlas grep found `pair-go context`, `pair-go scrollback-render`, and helper dispatch; `git diff --check`.

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
/bin/zsh -lc "sed -n '1,380p' workshop/plans/000076-go-helper-dispatch-plan.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
# Go Helper Dispatch Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `pair-go context` and `pair-go scrollback-render` routes that reuse the existing helper implementations while preserving the legacy helper binaries.

**Architecture:** Extract reusable helper runners from the selected `package main` commands and route the dispatcher through those runners instead of reimplementing command behavior (`ARCH-DRY`). Keep helper business logic pure or close to existing pure cores, with IO at thin command boundaries: runners accept argv/stdout/stderr/env-style inputs and return exit status or error, while `main()` remains a tiny process shell (`ARCH-PURE`). The slice intentionally proves the helper-dispatch pattern without moving live zellij/nvim call sites or changing public `pair` behavior (`ARCH-PURPOSE`).

**Tech Stack:** Go standard library, existing `cmd/internal/dispatcher`, existing `cmd/internal/ctxmeter` and `cmd/internal/transcript`, existing scrollback renderer core, `go test`, `make`.

---

## Core Concepts

### Command Surfaces

| Name | Kind | Lives in | Status |
|------|------|----------|--------|
| `contextcmd.Env` | INTEGRATION input value | `cmd/internal/contextcmd/contextcmd.go` | new |
| `contextcmd.Run` | INTEGRATION | `cmd/internal/contextcmd/contextcmd.go` | new |
| `scrollbackcmd.Run` | INTEGRATION | `cmd/internal/scrollbackcmd/scrollbackcmd.go` | new |
| `dispatcher.Dispatch` | INTEGRATION | `cmd/internal/dispatcher/dispatcher.go` | modified |

**contextcmd.Env** — Injected environment values for the context helper: home, XDG data home, and explicit Pair data dir.
- **Relationships:** Passed to `contextcmd.Run`; populated from `EnvFromOS` only at the process boundary.
- **DRY rationale:** Both legacy binary and dispatcher route need the same env/default resolution without re-reading OS env in tests.
- **Future extensions:** Can widen if more commands need shared Pair environment resolution.

**contextcmd.Run** — Shared command runner for `pair-context <tag> <agent>` behavior. It remains tolerant: missing config/transcript/input prints nothing and exits 0.
- **Injected into:** legacy `cmd/pair-context/main.go` and dispatcher context route.
- **Future extensions:** The title poller can continue using the legacy binary until #77/#78 moves call sites.

**scrollbackcmd.Run** — Shared command runner for `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out`.
- **Injected into:** legacy `cmd/pair-scrollback-render/main.go` and dispatcher scrollback-render route.
- **Future extensions:** `bin/pair-scrollback-open`, `bin/pair-changelog-open`, and `nvim/scrollback.lua` can move to the dispatcher after the public entrypoint is Go-owned.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ContextRunner` (`contextcmd.Run`) | `cmd/internal/contextcmd/contextcmd.go` | new | filesystem, env, transcript files, stdout |
| `ScrollbackRenderRunner` (`scrollbackcmd.Run`) | `cmd/internal/scrollbackcmd/scrollbackcmd.go` | new | filesystem, flag parsing, stdout/stderr |
| `dispatcher.Dispatch` | `cmd/internal/dispatcher/dispatcher.go` | modified | in-process calls to selected helper runners |

**ContextRunner** — Shared command runner for `pair-context <tag> <agent>` behavior. It remains tolerant: missing config/transcript/input prints nothing and exits 0.
- **Injected into:** legacy `cmd/pair-context/main.go` and dispatcher context route.
- **Future extensions:** The title poller can continue using the legacy binary until #77/#78 moves call sites.

**ScrollbackRenderRunner** — Shared command runner for `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out`.
- **Injected into:** legacy `cmd/pair-scrollback-render/main.go` and dispatcher scrollback-render route.
- **Future extensions:** `bin/pair-scrollback-open`, `bin/pair-changelog-open`, and `nvim/scrollback.lua` can move to the dispatcher after the public entrypoint is Go-owned.

**dispatcher.Dispatch** — Dispatcher routes for `context` and `scrollback-render`.
- **Injected into:** `cmd/pair-go`.
- **Future extensions:** Later helper routes should follow the same runner extraction pattern, not duplicate command logic.

## Revisions

- 2026-06-30T12:24:41-0700 — Boundary review found the Core Concepts table still named planned parsed-args structs that the implementation did not need. Replaced `ContextRunArgs`/`ScrollbackRenderArgs` with the shipped `contextcmd.Env`, `contextcmd.Run`, and `scrollbackcmd.Run` surfaces, and classified the runners as INTEGRATION rather than PURE entities.

---

## Chunk 1: Extract Context Runner

### Task 1: Make `pair-context` Reusable

**Files:**
- Create: `cmd/internal/contextcmd/contextcmd.go`
- Create: `cmd/internal/contextcmd/contextcmd_test.go`
- Modify: `cmd/pair-context/main.go`
- Modify: `cmd/pair-context/main_test.go`
- Modify: `Makefile.local`

- [x] **Step 1: Add failing runner tests**

Create `cmd/internal/contextcmd/contextcmd_test.go` with tests that call `Run(args []string, env Env, stdout io.Writer) int` directly:

```go
func TestRunClaude(t *testing.T) {
    // Arrange the same config/pane/transcript fixture as TestPairContext_Claude.
    // Call Run([]string{"T", "claude"}, env, &stdout).
    // Assert code == 0 and stdout == "398k\n".
}

func TestRunMissingConfigPrintsNothing(t *testing.T) {
    // Call Run with empty data dir.
    // Assert code == 0 and stdout == "".
}
```

- [x] **Step 2: Run the focused tests and confirm they fail**

Run: `go test ./cmd/internal/contextcmd -count=1`

Expected: FAIL because the package does not exist yet.

- [x] **Step 3: Extract the runner**

Move the reusable context behavior into `cmd/internal/contextcmd`. Keep `cmd/pair-context/main.go` as:

```go
func main() {
    os.Exit(contextcmd.Run(os.Args[1:], contextcmd.EnvFromOS(), os.Stdout))
}
```

The runner must:
- return 0 for missing args, matching the current tolerant behavior;
- resolve `PAIR_DATA_DIR` from env or `$XDG_DATA_HOME/pair` or `$HOME/.local/share/pair`;
- write the same humanized token count to the injected stdout;
- never call `os.Exit`.

- [x] **Step 4: Run the focused tests and existing package tests**

Run: `go test ./cmd/internal/contextcmd ./cmd/pair-context -count=1`

Expected: PASS.

- [x] **Step 5: Commit the context extraction**

Run:

```bash
git add cmd/internal/contextcmd cmd/pair-context Makefile.local
git commit -m "#76: extract pair-context runner"
```

---

## Chunk 2: Extract Scrollback Renderer Runner

### Task 2: Make `pair-scrollback-render` Reusable

**Files:**
- Create: `cmd/internal/scrollbackcmd/scrollbackcmd.go`
- Create: `cmd/internal/scrollbackcmd/scrollbackcmd_test.go`
- Modify: `cmd/pair-scrollback-render/main.go`
- Modify or create tests in: `cmd/pair-scrollback-render/*_test.go`
- Modify: `Makefile.local`

- [x] **Step 1: Add failing runner tests**

Create tests that call `scrollbackcmd.Run(args []string, stdout, stderr io.Writer) int`:

```go
func TestRunUsage(t *testing.T) {
    var stderr bytes.Buffer
    code := Run([]string{}, io.Discard, &stderr)
    // Assert code == 2 and usage is written to stderr.
}

func TestRunWritesOutput(t *testing.T) {
    // Use a tiny raw/events fixture compatible with existing renderer tests.
    // Call Run([]string{raw, events, out}, io.Discard, &stderr).
    // Assert code == 0 and out exists.
}
```

- [x] **Step 2: Run the focused tests and confirm they fail**

Run: `go test ./cmd/internal/scrollbackcmd -count=1`

Expected: FAIL because the package does not exist yet.

- [x] **Step 3: Extract the runner**

Move the renderer command wrapper into `cmd/internal/scrollbackcmd`. If `render(...)` cannot be imported from `package main`, move the rendering core into this internal package too and leave the legacy command as a tiny wrapper:

```go
func main() {
    os.Exit(scrollbackcmd.Run(os.Args[1:], os.Stdout, os.Stderr))
}
```

Use a local `flag.FlagSet` so dispatcher and tests can parse independently. Preserve current behavior:
- usage to stderr and exit 2 for wrong arity;
- `scrollback-render: <err>` to stderr and exit 1 for render errors;
- exit 0 for success;
- same defaults for `--plain`, `--max-lines`, and `--with-timestamps`.

- [x] **Step 4: Run focused and package tests**

Run: `go test ./cmd/internal/scrollbackcmd ./cmd/pair-scrollback-render -count=1`

Expected: PASS.

- [x] **Step 5: Commit the renderer extraction**

Run:

```bash
git add cmd/internal/scrollbackcmd cmd/pair-scrollback-render Makefile.local
git commit -m "#76: extract scrollback renderer runner"
```

---

## Chunk 3: Wire Dispatcher Routes

### Task 3: Route Selected Helpers Through `pair-go`

**Files:**
- Modify: `cmd/internal/dispatcher/dispatcher.go`
- Modify: `cmd/internal/dispatcher/dispatcher_test.go`
- Modify: `cmd/pair-go/main_test.go`
- Create: `cmd/pair-go/helper_equivalence_test.go`
- Modify: `Makefile.local`

- [x] **Step 1: Add failing dispatcher tests**

Add tests for:
- `Dispatch([]string{"context", "T", "claude"})` through a fake or temp fixture returns the same output shape as `contextcmd.Run`;
- `Dispatch([]string{"scrollback-render"})` returns renderer usage with exit 2;
- top-level help lists `context` and `scrollback-render` as implemented helper routes, not planned-only commands.

- [x] **Step 2: Run dispatcher tests and confirm failure**

Run: `go test ./cmd/internal/dispatcher -run 'TestDispatch(Context|Scrollback|Help)' -count=1`

Expected: FAIL because routes are not implemented yet.

- [x] **Step 3: Add failing process-level equivalence test**

Create `cmd/pair-go/helper_equivalence_test.go` with a context helper fixture that builds both commands and runs them against the same temp Pair data/transcript tree:

```go
func TestPairGoContextMatchesLegacyPairContext(t *testing.T) {
    // Build ./cmd/pair-context and ./cmd/pair-go into a temp bin dir.
    // Arrange config-T-claude.json, pane-T-claude.json, and a Claude transcript.
    // Run: pair-context T claude
    // Run: pair-go context T claude
    // Assert stdout, stderr, and exit code match exactly.
}
```

This is the representative compatibility proof required by the issue done-when. It should fail before the dispatcher route exists because `pair-go context` is still planned-only.

- [x] **Step 4: Run the equivalence test and confirm failure**

Run: `go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1`

Expected: FAIL because `pair-go context` returns planned-but-not-implemented.

- [x] **Step 5: Implement routes**

Update `Families()` statuses for selected helpers and add dispatcher branches:

```go
case "context":
    return dispatchContext(args[1:])
case "scrollback-render":
    return dispatchScrollbackRender(args[1:])
```

The dispatcher should continue returning `dispatcher.Result`. Helper runners should write into buffers so dispatcher can map stdout/stderr/exit code without adding a second process-result abstraction.

- [x] **Step 6: Run route and process tests**

Run:

```bash
go test ./cmd/internal/dispatcher ./cmd/pair-go ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render -count=1
make pair-context pair-scrollback-render pair-go
```

Expected: PASS. The `make` command is deliberately not `-B`; it verifies the updated dependency graph can rebuild normally after source changes.

- [x] **Step 7: Commit dispatcher wiring**

Run:

```bash
git add cmd/internal/dispatcher cmd/pair-go cmd/internal/contextcmd cmd/internal/scrollbackcmd cmd/pair-context cmd/pair-scrollback-render Makefile.local
git commit -m "#76: route selected helpers through pair-go"
```

---

## Chunk 4: Verify Legacy Compatibility And Docs

### Task 4: Verify Builds, Callers, And Atlas

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `atlas/go-migration-inventory.md`
- Modify: `workshop/issues/000076-go-helper-dispatch.md`
- Modify: `Makefile.local`

- [x] **Step 1: Verify legacy binaries still build**

Run:

```bash
make pair-context pair-scrollback-render pair-go
make -B pair-context pair-scrollback-render pair-go
```

Expected: PASS. The non-`-B` run verifies incremental prerequisites include `cmd/internal/contextcmd`, `cmd/internal/scrollbackcmd`, and dispatcher dependencies; the `-B` run remains the forced clean rebuild check.

- [x] **Step 2: Verify selected command equivalence**

Run focused commands against test fixtures or package tests:

```bash
go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1
go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1
```

Expected: PASS; the equivalence test demonstrates the legacy `pair-context` binary and `pair-go context` process path produce identical stdout/stderr/exit code on the same fixture.

- [x] **Step 3: Verify full Go test suite**

Run: `go test ./... -count=1`

Expected: PASS.

- [x] **Step 4: Verify no live call sites moved**

Run:

```bash
git diff -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollback-open bin/pair-changelog-open
```

Expected: empty diff, unless a test-only or documentation-only change was explicitly made.

- [x] **Step 5: Update atlas**

Update:
- `atlas/architecture.md` to say `pair-go context` and `pair-go scrollback-render` are implemented helper routes while public launcher and live zellij/nvim callers remain legacy.
- `atlas/go-migration-inventory.md` rows for `pair-context` and `pair-scrollback-render` to record dispatcher availability and preserved legacy names.

- [x] **Step 6: Update issue checklist and log**

Tick the #76 plan/done items that are complete and add a log entry with verification commands and `ARCH-*` notes.

- [x] **Step 7: Run final verification before close**

Run:

```bash
go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1
go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1
make pair-context pair-scrollback-render pair-go
make -B pair-context pair-scrollback-render pair-go
go test ./... -count=1
git diff -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollback-open bin/pair-changelog-open
rg -n "pair-go context|pair-go scrollback-render|helper dispatch" atlas/architecture.md atlas/go-migration-inventory.md
git diff --check
```

Expected: all tests/builds pass, caller diff empty, atlas grep finds the new helper-dispatch documentation, and whitespace check passes.

- [ ] **Step 8: Close through SDLC**

Run:

```bash
sdlc close --issue 76 --verified 'go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1; go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1; make pair-context pair-scrollback-render pair-go; make -B pair-context pair-scrollback-render pair-go; go test ./... -count=1; git diff live callers empty; rg atlas helper dispatch; git diff --check'
```

Expected: close gate runs the boundary review and reports SHIP or actionable findings.

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
The tracker and plan line up on scope: only `context` and `scrollback-render`, legacy callers preserved, atlas updated. I’m checking the actual `HEAD` files and base-to-head diff for behavioral drift, especially where command-line semantics can subtly change after moving code out of `package main`.
exec
/bin/zsh -lc "nl -ba cmd/internal/contextcmd/contextcmd.go | sed -n '1,140p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	// Package contextcmd implements the pair-context command body.
     2	package contextcmd
     3	
     4	import (
     5		"encoding/json"
     6		"fmt"
     7		"io"
     8		"os"
     9		"path/filepath"
    10	
    11		"github.com/xianxu/pair/cmd/internal/ctxmeter"
    12		"github.com/xianxu/pair/cmd/internal/transcript"
    13	)
    14	
    15	type Env struct {
    16		Home        string
    17		XDGDataHome string
    18		PairDataDir string
    19	}
    20	
    21	func EnvFromOS() Env {
    22		return Env{
    23			Home:        os.Getenv("HOME"),
    24			XDGDataHome: os.Getenv("XDG_DATA_HOME"),
    25			PairDataDir: os.Getenv("PAIR_DATA_DIR"),
    26		}
    27	}
    28	
    29	func Run(args []string, env Env, stdout io.Writer) int {
    30		if len(args) < 2 {
    31			return 0
    32		}
    33		tag, agent := args[0], args[1]
    34		dataDir := resolveDataDir(env)
    35		sid := transcript.SessionID(dataDir, tag, agent)
    36		if sid == "" {
    37			return 0
    38		}
    39		cwd := paneCwd(dataDir, tag, agent)
    40		path := transcript.Resolve(agent, sid, cwd, env.Home)
    41		if path == "" {
    42			return 0
    43		}
    44		f, err := os.Open(path)
    45		if err != nil {
    46			return 0
    47		}
    48		defer f.Close()
    49		if n, ok := ctxmeter.ContextTokens(agent, f); ok {
    50			fmt.Fprintln(stdout, ctxmeter.Humanize(n))
    51		}
    52		return 0
    53	}
    54	
    55	func resolveDataDir(env Env) string {
    56		if env.PairDataDir != "" {
    57			return env.PairDataDir
    58		}
    59		base := env.XDGDataHome
    60		if base == "" {
    61			base = filepath.Join(env.Home, ".local", "share")
    62		}
    63		return filepath.Join(base, "pair")
    64	}
    65	
    66	func paneCwd(dataDir, tag, agent string) string {
    67		b, err := os.ReadFile(filepath.Join(dataDir, "pane-"+tag+"-"+agent+".json"))
    68		if err != nil {
    69			return ""
    70		}
    71		var p struct {
    72			Cwd string `json:"cwd"`
    73		}
    74		if json.Unmarshal(b, &p) != nil {
    75			return ""
    76		}
    77		return p.Cwd
    78	}

exec
/bin/zsh -lc "nl -ba cmd/internal/dispatcher/dispatcher.go | sed -n '1,240p'" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc 'git diff --stat 9cc0f57521c3e44508adf3df006ffe3347faa265..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 Makefile.local                                     |   6 +-
 atlas/architecture.md                              |   8 +
 atlas/go-migration-inventory.md                    |  11 +-
 cmd/internal/contextcmd/contextcmd.go              |  78 ++++
 cmd/internal/contextcmd/contextcmd_test.go         |  59 +++
 cmd/internal/dispatcher/dispatcher.go              |  29 +-
 cmd/internal/dispatcher/dispatcher_test.go         |  75 +++-
 .../scrollbackcmd}/events_test.go                  |   2 +-
 .../scrollbackcmd}/render_test.go                  |   2 +-
 cmd/internal/scrollbackcmd/scrollbackcmd.go        | 413 +++++++++++++++++++++
 cmd/internal/scrollbackcmd/scrollbackcmd_test.go   |  48 +++
 .../scrollbackcmd}/serialize_row_test.go           |   2 +-
 .../scrollbackcmd}/timestamps_test.go              |   2 +-
 cmd/pair-context/main.go                           |  50 +--
 cmd/pair-go/helper_equivalence_test.go             |  93 +++++
 cmd/pair-scrollback-render/main.go                 | 402 +-------------------
 workshop/issues/000076-go-helper-dispatch.md       |  22 +-
 workshop/plans/000076-go-helper-dispatch-plan.md   |  96 ++---
 18 files changed, 879 insertions(+), 519 deletions(-)

 succeeded in 0ms:
     1	package dispatcher
     2	
     3	import (
     4		"bytes"
     5		"fmt"
     6		"os"
     7		"strings"
     8		"time"
     9	
    10		"github.com/xianxu/pair/cmd/internal/contextcmd"
    11		"github.com/xianxu/pair/cmd/internal/launcher"
    12		"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
    13	)
    14	
    15	const programName = "pair-go"
    16	
    17	// CommandFamily names a future Pair CLI surface without claiming it works yet.
    18	type CommandFamily struct {
    19		Name    string
    20		Summary string
    21		Status  string
    22	}
    23	
    24	// Result is the process-facing outcome of a pure dispatch decision.
    25	type Result struct {
    26		Stdout   string
    27		Stderr   string
    28		ExitCode int
    29	}
    30	
    31	// Families returns the planned command families for the Go dispatcher.
    32	func Families() []CommandFamily {
    33		return []CommandFamily{
    34			{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "prototype"},
    35			{Name: "context", Summary: "agent pane context meter", Status: "implemented"},
    36			{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "implemented"},
    37			{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
    38			{Name: "slug", Summary: "session orientation slug generation", Status: "planned"},
    39			{Name: "changelog", Summary: "TTY transcript to distilled change log", Status: "planned"},
    40			{Name: "continuation", Summary: "continuation datatype writer", Status: "planned"},
    41			{Name: "scribe", Summary: "PTY logging wrapper", Status: "planned"},
    42		}
    43	}
    44	
    45	// Dispatch parses argv and returns the skeleton dispatch result.
    46	func Dispatch(args []string) Result {
    47		if len(args) == 0 {
    48			return Result{Stdout: Help(programName), ExitCode: 0}
    49		}
    50	
    51		switch args[0] {
    52		case "help", "--help", "-h":
    53			return Result{Stdout: Help(programName), ExitCode: 0}
    54		case "version", "--version":
    55			return Result{
    56				Stdout:   "pair-go dispatcher skeleton\npublic launcher: bin/pair\n",
    57				ExitCode: 0,
    58			}
    59		case "launch":
    60			return DispatchWithLauncherRuntime(args, osLauncherRuntime())
    61		case "context":
    62			return dispatchContext(args[1:])
    63		case "scrollback-render":
    64			return dispatchScrollbackRender(args[1:])
    65		}
    66	
    67		if family, ok := familyByName(args[0]); ok {
    68			return Result{
    69				Stderr:   fmt.Sprintf("%s: %s is planned but not implemented in this skeleton; run %s help\n", programName, family.Name, programName),
    70				ExitCode: 2,
    71			}
    72		}
    73	
    74		return Result{
    75			Stderr:   fmt.Sprintf("%s: unknown command %q; run %s help\n", programName, args[0], programName),
    76			ExitCode: 2,
    77		}
    78	}
    79	
    80	func dispatchContext(args []string) Result {
    81		var stdout bytes.Buffer
    82		code := contextcmd.Run(args, contextcmd.EnvFromOS(), &stdout)
    83		return Result{Stdout: stdout.String(), ExitCode: code}
    84	}
    85	
    86	func dispatchScrollbackRender(args []string) Result {
    87		var stdout, stderr bytes.Buffer
    88		code := scrollbackcmd.Run(args, &stdout, &stderr)
    89		return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}
    90	}
    91	
    92	type LauncherRuntime struct {
    93		Env      launcher.Env
    94		Sessions launcher.SessionSource
    95		History  launcher.HistoricalScanner
    96	}
    97	
    98	type StaticSessions struct {
    99		Sessions []launcher.Session
   100		Err      error
   101	}
   102	
   103	func (s StaticSessions) Snapshot() ([]launcher.Session, error) {
   104		if s.Err != nil {
   105			return nil, s.Err
   106		}
   107		return s.Sessions, nil
   108	}
   109	
   110	type StaticHistory struct {
   111		Tags []launcher.HistoricalTag
   112		Err  error
   113	}
   114	
   115	func (h StaticHistory) Scan(_ string, _ time.Time) ([]launcher.HistoricalTag, error) {
   116		if h.Err != nil {
   117			return nil, h.Err
   118		}
   119		return h.Tags, nil
   120	}
   121	
   122	func DispatchWithLauncherRuntime(args []string, rt LauncherRuntime) Result {
   123		launchArgs := []string(nil)
   124		if len(args) > 1 {
   125			launchArgs = args[1:]
   126		}
   127		if len(launchArgs) > 0 && (launchArgs[0] == "help" || launchArgs[0] == "--help" || launchArgs[0] == "-h") {
   128			return Result{Stdout: LaunchHelp(programName), ExitCode: 0}
   129		}
   130		outcome, err := launcher.Run(launchArgs, rt.Env, rt.Sessions, rt.History)
   131		if err != nil {
   132			return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
   133		}
   134		decision := outcome.Decision
   135		return Result{
   136			Stderr: fmt.Sprintf(
   137				"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
   138				decision.Action,
   139				decision.Tag,
   140				decision.SessionName,
   141			),
   142			ExitCode: 3,
   143		}
   144	}
   145	
   146	func LaunchHelp(program string) string {
   147		return fmt.Sprintf(`Usage: %s launch [agent] [-- agent-args...]
   148	       %s launch resume <tag>
   149	
   150	Guarded decision-phase prototype. Public sessions still start through bin/pair.
   151	This command parses launch inputs and computes the create/attach/picker decision,
   152	then stops before invoking zellij.
   153	`, program, program)
   154	}
   155	
   156	func LauncherEnv(home, xdgDataHome, cwd string) launcher.Env {
   157		return launcher.Env{
   158			Home:     home,
   159			XDGData:  xdgDataHome,
   160			Cwd:      cwd,
   161			Now:      time.Now(),
   162			HistoryD: 14,
   163		}
   164	}
   165	
   166	func osLauncherRuntime() LauncherRuntime {
   167		home := os.Getenv("HOME")
   168		xdg := os.Getenv("XDG_DATA_HOME")
   169		cwd, _ := os.Getwd()
   170		env := LauncherEnv(home, xdg, cwd)
   171		dataDir := launcher.ResolveDataDir(home, xdg)
   172		return LauncherRuntime{
   173			Env:      env,
   174			Sessions: launcher.ZellijSource{},
   175			History:  launcher.HistorySource{DataDir: dataDir},
   176		}
   177	}
   178	
   179	// Help renders the development-only dispatcher usage text.
   180	func Help(program string) string {
   181		var b strings.Builder
   182		fmt.Fprintf(&b, "Usage: %s <command> [args]\n\n", program)
   183		b.WriteString("Development dispatcher skeleton. Public sessions still start through bin/pair.\n\n")
   184		b.WriteString("Implemented commands:\n")
   185		for _, family := range Families() {
   186			if family.Status == "prototype" {
   187				fmt.Fprintf(&b, "  %-17s %s (prototype; decision-phase only)\n", family.Name, family.Summary)
   188			} else if family.Status == "implemented" {
   189				fmt.Fprintf(&b, "  %-17s %s (implemented helper route)\n", family.Name, family.Summary)
   190			}
   191		}
   192		b.WriteString("\nPlanned command families (not implemented in this skeleton):\n")
   193		for _, family := range Families() {
   194			if family.Status == "planned" {
   195				fmt.Fprintf(&b, "  %-17s %s (%s; not implemented in this skeleton)\n", family.Name, family.Summary, family.Status)
   196			}
   197		}
   198		b.WriteString("\nSupported skeleton commands:\n")
   199		b.WriteString("  help              show this help\n")
   200		b.WriteString("  version           show dispatcher skeleton metadata\n")
   201		return b.String()
   202	}
   203	
   204	func familyByName(name string) (CommandFamily, bool) {
   205		for _, family := range Families() {
   206			if family.Name == name {
   207				return family, true
   208			}
   209		}
   210		return CommandFamily{}, false
   211	}

exec
/bin/zsh -lc "nl -ba cmd/internal/scrollbackcmd/scrollbackcmd.go | sed -n '1,460p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	// scrollback-render — replay a pair-wrap raw capture through a VT100
     2	// emulator and write one ANSI-styled output line per logical scrollback row.
     3	// Drop-in replacement for the Python+pyte renderer at bin/pair-scrollback-render.
     4	//
     5	// Why Go: pyte's HistoryScreen dispatches every method call through an
     6	// __getattribute__ override that ran ~19M times for a 3 MB raw input —
     7	// ~95% of wall time. Even after the CaptureScreen patch (3.6x speedup),
     8	// the Python interpreter + pyte vendoring add startup cost and a private
     9	// venv that the brew formula has to manage. A static Go binary using
    10	// charmbracelet/x/vt replays the same stream with no runtime deps and
    11	// stays within the pair repo's existing cmd/ layout.
    12	//
    13	// Pipeline:
    14	//
    15	//	raw bytes (.raw)              → emulator.Write(...) in segments
    16	//	resize events (.events.jsonl) → segment boundaries with new (cols,rows)
    17	//	final emulator state          → scrollback lines + visible buffer
    18	//	each row                      → SGR-decorated text line written to out
    19	//
    20	// CLI is identical to the Python version so bin/pair-scrollback-open can
    21	// invoke either:
    22	//
    23	//	scrollback-render <raw> <events.jsonl> <out.ansi>
    24	package scrollbackcmd
    25	
    26	import (
    27		"encoding/json"
    28		"flag"
    29		"fmt"
    30		"io"
    31		"math"
    32		"os"
    33		"strconv"
    34		"strings"
    35		"time"
    36	
    37		uv "github.com/charmbracelet/ultraviolet"
    38		"github.com/charmbracelet/x/vt"
    39	)
    40	
    41	// historyRows caps scrolled-out rows retained for the viewer. Matched
    42	// to zellij/config.kdl's `scroll_buffer_size 2000` so PageUp inside the
    43	// agent pane and Alt+/ both reach back the same distance — otherwise
    44	// the viewer would surface lines that zellij no longer has, leaving
    45	// the scroll-overlay logic in pair-scrollback-open unable to align
    46	// against the agent pane's frame.
    47	const historyRows = 2_000
    48	
    49	const (
    50		defaultCols = 80
    51		defaultRows = 24
    52	)
    53	
    54	type scrollbackEvent struct {
    55		Type   string `json:"type"`
    56		Offset int64  `json:"offset"`
    57		Cols   int    `json:"cols"`
    58		Rows   int    `json:"rows"`
    59		Ts     string `json:"ts,omitempty"` // RFC3339 wall-clock for "time" events (#59)
    60	}
    61	
    62	// dateOf extracts the YYYY-MM-DD day from an RFC3339 timestamp; "" on a
    63	// malformed value so a corrupt time event degrades to undated, never panics (#59).
    64	func dateOf(ts string) string {
    65		t, err := time.Parse(time.RFC3339, ts)
    66		if err != nil {
    67			return ""
    68		}
    69		return t.Format("2006-01-02")
    70	}
    71	
    72	// parseEvents reads the sidecar JSONL. Empty / missing file → empty slice.
    73	// Malformed lines are skipped so a corrupted tail doesn't abort the render —
    74	// imperfect width tracking beats an unusable viewer.
    75	func parseEvents(path string) ([]scrollbackEvent, error) {
    76		data, err := os.ReadFile(path)
    77		if err != nil {
    78			if os.IsNotExist(err) {
    79				return nil, nil
    80			}
    81			return nil, err
    82		}
    83		var out []scrollbackEvent
    84		for _, line := range strings.Split(string(data), "\n") {
    85			line = strings.TrimSpace(line)
    86			if line == "" {
    87				continue
    88			}
    89			var e scrollbackEvent
    90			if err := json.Unmarshal([]byte(line), &e); err != nil {
    91				continue
    92			}
    93			// Keep both known types: resize boundaries AND time stamps (#59).
    94			// Consumers filter by Type at their use sites.
    95			if e.Type == "resize" || e.Type == "time" {
    96				out = append(out, e)
    97			}
    98		}
    99		return out, nil
   100	}
   101	
   102	// initialSize pulls (cols, rows) from the first usable resize event, or
   103	// falls back to 80x24 if the sidecar is empty or the first entry is
   104	// malformed. pair-wrap always emits an initial resize at offset 0, so the
   105	// fallback only fires on a truncated file.
   106	func initialSize(events []scrollbackEvent) (int, int) {
   107		for _, e := range events {
   108			if e.Type == "resize" && e.Cols > 0 && e.Rows > 0 {
   109				return e.Cols, e.Rows
   110			}
   111		}
   112		return defaultCols, defaultRows
   113	}
   114	
   115	// dateMark records the emulator's scrollback length at a "time" event's byte
   116	// offset → the day that applies to committed lines from that index onward (#59).
   117	// Built during the feed (the only place that knows both byte offsets and the
   118	// rendered line count); consumed by the pure interleaveDateMarkers.
   119	type dateMark struct {
   120		line int
   121		date string
   122	}
   123	
   124	// feedSegments writes raw into the emulator as a single offset-ordered walk over
   125	// ALL sidecar events: write everything up to event.Offset, then act — Resize on a
   126	// resize event, or snapshot Scrollback().Len() on a time event. Returns the time
   127	// snapshots (empty unless time events are present). The caller already set the
   128	// initial size via initialSize; re-applying the offset-0 resize here is a harmless
   129	// no-op (resize to the current dimensions). Walking all events — rather than
   130	// events[1:] — means a time event in any position (incl. first) is captured, and
   131	// an empty events slice is handled without an out-of-range slice (#59).
   132	//
   133	// Clamping Offset to len(raw) defends against a corrupted sidecar that records
   134	// an offset beyond EOF (saw this once with a half-written events file after a
   135	// hard kill); without clamping we'd panic on the slice.
   136	func feedSegments(em *vt.Emulator, raw []byte, events []scrollbackEvent) []dateMark {
   137		var cursor int64
   138		var marks []dateMark
   139		for _, e := range events {
   140			off := e.Offset
   141			if off > int64(len(raw)) {
   142				off = int64(len(raw))
   143			}
   144			if off > cursor {
   145				_, _ = em.Write(raw[cursor:off])
   146				cursor = off
   147			}
   148			switch e.Type {
   149			case "resize":
   150				em.Resize(e.Cols, e.Rows)
   151			case "time":
   152				if d := dateOf(e.Ts); d != "" {
   153					marks = append(marks, dateMark{line: em.Scrollback().Len(), date: d})
   154				}
   155			}
   156		}
   157		if cursor < int64(len(raw)) {
   158			_, _ = em.Write(raw[cursor:])
   159		}
   160		return marks
   161	}
   162	
   163	// tsMarkerLine is the wire format the distiller parses (#59). MUST stay in sync
   164	// with tsMarkerRe in cmd/pair-changelog/distill.go — the contract is pinned by
   165	// the render→clean→distill e2e test cmd/pair-changelog/e2e_test.go
   166	// (TestEndToEndMarkerSurvival), which feeds real time events through both binaries.
   167	func tsMarkerLine(date string) string {
   168		return "⟦pair:ts " + date + "⟧"
   169	}
   170	
   171	// interleaveDateMarkers inserts a tsMarkerLine immediately before the first line
   172	// of each new date run. marks are (scrollback-line-index, date) snapshots in
   173	// ascending index; a marker is emitted only when the applicable date *changes*
   174	// from the running date (consecutive same-date marks collapse). Lines before the
   175	// first mark stay undated; marks past len(lines) are ignored. Pure (#59).
   176	func interleaveDateMarkers(lines []string, marks []dateMark) []string {
   177		if len(marks) == 0 {
   178			return lines
   179		}
   180		out := make([]string, 0, len(lines)+len(marks))
   181		mi := 0
   182		prevDate := ""
   183		for i := 0; i < len(lines); i++ {
   184			curDate := prevDate
   185			for mi < len(marks) && marks[mi].line <= i {
   186				curDate = marks[mi].date
   187				mi++
   188			}
   189			if curDate != "" && curDate != prevDate {
   190				out = append(out, tsMarkerLine(curDate))
   191				prevDate = curDate
   192			}
   193			out = append(out, lines[i])
   194		}
   195		return out
   196	}
   197	
   198	// serializeRow flattens one row into ANSI-styled text. Trims trailing
   199	// default-styled blanks (so the viewer doesn't scroll past pad), uses
   200	// Style.Diff so we only emit SGR codes when the style actually changes
   201	// between cells, and terminates with \x1b[0m to keep the row's last
   202	// style from bleeding into the next line if a viewer concatenates without
   203	// resetting between lines.
   204	//
   205	// A non-default background space is treated as visible content (e.g.
   206	// inverse-video padding). Matches what the Python renderer does.
   207	//
   208	// In plain mode (plain=true) no SGR is emitted at all: the row is just its
   209	// visible content, trimmed to the last non-blank-*content* cell — a cell that
   210	// is "visible" only via a non-default background (inverse-video padding, box
   211	// fill) is NOT emitted in plain mode, so it must not extend the row, or a
   212	// trailing bordered region would become space-padding toward terminal width.
   213	func serializeRow(line uv.Line, plain bool) string {
   214		last := -1
   215		for i := range line {
   216			c := &line[i]
   217			// Continuation cells of a preceding wide grapheme are stored as
   218			// zero-value Cell{} per the ultraviolet convention (Width=0,
   219			// Content=""). They don't extend the visible row and must not
   220			// emit anything in the loop below.
   221			if c.IsZero() {
   222				continue
   223			}
   224			content := c.Content
   225			if content != "" && content != " " {
   226				last = i
   227			} else if !plain && c.Style.Bg != nil {
   228				last = i
   229			}
   230		}
   231		if last < 0 {
   232			return ""
   233		}
   234		var b strings.Builder
   235		var prev uv.Style // zero value = default; Diff vs zero emits a reset
   236		first := true
   237		for i := 0; i <= last; i++ {
   238			c := &line[i]
   239			// Skip wide-grapheme continuation cells — the wide cell already
   240			// emitted its full glyph; emitting anything here adds a phantom
   241			// space after every emoji.
   242			if c.IsZero() {
   243				continue
   244			}
   245			if !plain && (first || !c.Style.Equal(&prev)) {
   246				b.WriteString(c.Style.Diff(&prev))
   247				prev = c.Style
   248				first = false
   249			}
   250			if c.Content == "" {
   251				b.WriteByte(' ')
   252			} else {
   253				b.WriteString(c.Content)
   254			}
   255		}
   256		if !plain {
   257			b.WriteString("\x1b[0m")
   258		}
   259		return b.String()
   260	}
   261	
   262	// resolveMax maps a --max-lines value to a scrollback cap. <=0 means
   263	// "uncapped" — a continuation wants the whole session, not the viewer's
   264	// 2000-row window. Represented as a large sentinel; .raw is per-run
   265	// O_TRUNC'd, so the practical bound is the run length.
   266	func resolveMax(n int) int {
   267		if n <= 0 {
   268			return math.MaxInt32
   269		}
   270		return n
   271	}
   272	
   273	// visibleRow materializes row y of the live screen as a uv.Line. The
   274	// emulator exposes cells one at a time via CellAt(x,y); there's no
   275	// "give me the whole row" accessor. A missing cell (CellAt returns nil)
   276	// becomes a zero-value Cell, which serializeRow treats as a blank.
   277	func visibleRow(em *vt.Emulator, y, width int) uv.Line {
   278		row := make(uv.Line, width)
   279		for x := 0; x < width; x++ {
   280			if c := em.CellAt(x, y); c != nil {
   281				row[x] = *c
   282			}
   283		}
   284		return row
   285	}
   286	
   287	func render(rawPath, eventsPath, outPath string, plain bool, maxLines int, withTimestamps bool) error {
   288		events, err := parseEvents(eventsPath)
   289		if err != nil {
   290			return fmt.Errorf("parse events: %w", err)
   291		}
   292		cols, rows := initialSize(events)
   293		em := vt.NewEmulator(cols, rows)
   294		em.Scrollback().SetMaxLines(resolveMax(maxLines))
   295	
   296		// Drain the emulator's input pipe in the background. CSI status
   297		// queries (DSR, Device Attributes, etc.) in the captured stream
   298		// trigger handlers that *write a reply back* into this pipe — in a
   299		// real terminal those bytes go to the controlling app. Offscreen
   300		// replay has no reader, so the handler's WriteString blocks
   301		// forever and deadlocks the Write goroutine. Discarding the bytes
   302		// preserves emulation correctness; we never act on the replies.
   303		//
   304		// Wait for the drainer to actually exit before letting em.Close()
   305		// run, otherwise Close races with the drainer's still-pending
   306		// Read() (race detector catches it; in production the window is
   307		// usually harmless but it's a real ordering bug).
   308		drainDone := make(chan struct{})
   309		go func() {
   310			defer close(drainDone)
   311			_, _ = io.Copy(io.Discard, em)
   312		}()
   313		defer func() {
   314			em.Close()
   315			<-drainDone
   316		}()
   317	
   318		raw, err := os.ReadFile(rawPath)
   319		if err != nil {
   320			return fmt.Errorf("read raw: %w", err)
   321		}
   322		marks := feedSegments(em, raw, events)
   323	
   324		// Scrollback lines (oldest → newest), then visible buffer top → bottom.
   325		// Visible buffer iterates by row index rather than dropping trailing
   326		// blank rows: an agent that cleared and paused mid-redraw would shift
   327		// every subsequent line number otherwise, and `:880` should still land
   328		// where zellij showed line 880.
   329		sb := em.Scrollback()
   330		viewportTop := sb.Len() + 1 // 1-indexed line where the visible buffer starts
   331		out := make([]string, 0, sb.Len()+em.Height())
   332		for i := 0; i < sb.Len(); i++ {
   333			out = append(out, serializeRow(sb.Line(i), plain))
   334		}
   335		w := em.Width()
   336		for y := 0; y < em.Height(); y++ {
   337			out = append(out, serializeRow(visibleRow(em, y, w), plain))
   338		}
   339		// Trim trailing all-blank lines: a half-empty visible buffer otherwise
   340		// leaves a tail of empties at EOF.
   341		for len(out) > 0 && out[len(out)-1] == "" {
   342			out = out[:len(out)-1]
   343		}
   344	
   345		// Change-log path only: interleave day markers from the time-event snapshots
   346		// so the distiller can date entries by real change-time (#59). Done after the
   347		// trailing-blank trim so a marker never dangles past content. The scrollback
   348		// viewer never sets this flag → its render is byte-identical to before.
   349		if withTimestamps {
   350			out = interleaveDateMarkers(out, marks)
   351		}
   352	
   353		// Write the viewport sidecar *first*, then atomically rename the
   354		// .ansi into place. Order matters: scrollback.lua's BufReadPost
   355		// opens the .ansi and immediately reads the sidecar — flipping the
   356		// .ansi last guarantees the sidecar is up-to-date by the time
   357		// nvim sees the new content. Sidecar is best-effort: on write
   358		// failure, scrollback.lua falls back to its prior bottom-alignment.
   359		// The viewport sidecar positions the Alt+/ nvim viewer; it's meaningless
   360		// for the plain projection (a continuation distills the text, not a
   361		// scroll position), so skip it and don't litter a stray <out>.viewport.
   362		if !plain {
   363			viewportPath := strings.TrimSuffix(outPath, ".ansi") + ".viewport"
   364			_ = os.WriteFile(viewportPath, []byte(strconv.Itoa(viewportTop)+"\n"), 0o644)
   365		}
   366	
   367		// Atomic write so a double-tap Alt+/ can't race truncate-then-write
   368		// on the same path. Reader sees either the old complete file or the
   369		// new complete file, never a half-written one.
   370		tmp := outPath + ".tmp"
   371		f, err := os.Create(tmp)
   372		if err != nil {
   373			return err
   374		}
   375		body := strings.Join(out, "\n")
   376		if len(out) > 0 {
   377			body += "\n"
   378		}
   379		if _, err := f.WriteString(body); err != nil {
   380			f.Close()
   381			os.Remove(tmp)
   382			return err
   383		}
   384		if err := f.Close(); err != nil {
   385			os.Remove(tmp)
   386			return err
   387		}
   388		return os.Rename(tmp, outPath)
   389	}
   390	
   391	func Run(argv []string, stdout, stderr io.Writer) int {
   392		fs := flag.NewFlagSet("pair-scrollback-render", flag.ContinueOnError)
   393		fs.SetOutput(stderr)
   394		fs.Usage = func() {
   395			fmt.Fprintf(stderr, "usage: pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] <raw> <events.jsonl> <out>\n")
   396		}
   397		plain := fs.Bool("plain", false, "emit plain text (no SGR) for distillation")
   398		maxLines := fs.Int("max-lines", historyRows, "scrollback history rows retained; <=0 = uncapped")
   399		withTimestamps := fs.Bool("with-timestamps", false, "interleave ⟦pair:ts DATE⟧ day markers from time events (for the change log; #59)")
   400		if err := fs.Parse(argv); err != nil {
   401			return 2
   402		}
   403		args := fs.Args()
   404		if len(args) != 3 {
   405			fs.Usage()
   406			return 2
   407		}
   408		if err := render(args[0], args[1], args[2], *plain, *maxLines, *withTimestamps); err != nil {
   409			fmt.Fprintf(stderr, "scrollback-render: %v\n", err)
   410			return 1
   411		}
   412		return 0
   413	}

exec
/bin/zsh -lc "nl -ba cmd/pair-go/main.go | sed -n '1,120p'" in /Users/xianxu/workspace/pair
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
    18		return writeResult(res, stdout, stderr)
    19	}
    20	
    21	func runWithLauncherRuntime(args []string, stdout, stderr io.Writer, rt dispatcher.LauncherRuntime) int {
    22		res := dispatcher.DispatchWithLauncherRuntime(args, rt)
    23		return writeResult(res, stdout, stderr)
    24	}
    25	
    26	func writeResult(res dispatcher.Result, stdout, stderr io.Writer) int {
    27		if res.Stdout != "" {
    28			_, _ = io.WriteString(stdout, res.Stdout)
    29		}
    30		if res.Stderr != "" {
    31			_, _ = io.WriteString(stderr, res.Stderr)
    32		}
    33		return res.ExitCode
    34	}

codex
One subtle drift candidate is the renderer’s usage text: the old binary used `os.Args[0]`, while the shared runner now prints a fixed command name. I’m checking whether that matters against the stated contract and looking for any higher-impact runtime issues before deciding severity.
exec
/bin/zsh -lc "git show 9cc0f57521c3e44508adf3df006ffe3347faa265:cmd/pair-scrollback-render/main.go | nl -ba | sed -n '380,430p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
   380			f.Close()
   381			os.Remove(tmp)
   382			return err
   383		}
   384		if err := f.Close(); err != nil {
   385			os.Remove(tmp)
   386			return err
   387		}
   388		return os.Rename(tmp, outPath)
   389	}
   390	
   391	func main() {
   392		flag.Usage = func() {
   393			fmt.Fprintf(os.Stderr, "usage: %s [--plain] [--max-lines N] [--with-timestamps] <raw> <events.jsonl> <out>\n", os.Args[0])
   394		}
   395		plain := flag.Bool("plain", false, "emit plain text (no SGR) for distillation")
   396		maxLines := flag.Int("max-lines", historyRows, "scrollback history rows retained; <=0 = uncapped")
   397		withTimestamps := flag.Bool("with-timestamps", false, "interleave ⟦pair:ts DATE⟧ day markers from time events (for the change log; #59)")
   398		flag.Parse()
   399		args := flag.Args()
   400		if len(args) != 3 {
   401			flag.Usage()
   402			os.Exit(2)
   403		}
   404		if err := render(args[0], args[1], args[2], *plain, *maxLines, *withTimestamps); err != nil {
   405			fmt.Fprintf(os.Stderr, "scrollback-render: %v\n", err)
   406			os.Exit(1)
   407		}
   408	}

exec
/bin/zsh -lc "git show 9cc0f57521c3e44508adf3df006ffe3347faa265:cmd/pair-context/main.go | nl -ba | sed -n '1,100p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	// pair-context — print one agent pane's current context size (humanized
     2	// token count), or nothing. Invoked as `pair-context <tag> <agent>` by the
     3	// pair-title poller. Tolerant: any failure prints nothing and exits 0, so a
     4	// hiccup never garbles the pane title.
     5	package main
     6	
     7	import (
     8		"encoding/json"
     9		"fmt"
    10		"os"
    11		"path/filepath"
    12	
    13		"github.com/xianxu/pair/cmd/internal/ctxmeter"
    14		"github.com/xianxu/pair/cmd/internal/transcript"
    15	)
    16	
    17	func main() {
    18		if len(os.Args) < 3 {
    19			return
    20		}
    21		tag, agent := os.Args[1], os.Args[2]
    22		dataDir := os.Getenv("PAIR_DATA_DIR")
    23		if dataDir == "" {
    24			base := os.Getenv("XDG_DATA_HOME")
    25			if base == "" {
    26				base = filepath.Join(os.Getenv("HOME"), ".local", "share")
    27			}
    28			dataDir = filepath.Join(base, "pair")
    29		}
    30		sid := transcript.SessionID(dataDir, tag, agent)
    31		if sid == "" {
    32			return
    33		}
    34		cwd := paneCwd(dataDir, tag, agent) // "" for codex/agy is fine (Resolve ignores it)
    35		path := transcript.Resolve(agent, sid, cwd, os.Getenv("HOME"))
    36		if path == "" {
    37			return
    38		}
    39		f, err := os.Open(path)
    40		if err != nil {
    41			return
    42		}
    43		defer f.Close()
    44		if n, ok := ctxmeter.ContextTokens(agent, f); ok {
    45			fmt.Println(ctxmeter.Humanize(n))
    46		}
    47	}
    48	
    49	func paneCwd(dataDir, tag, agent string) string {
    50		b, err := os.ReadFile(filepath.Join(dataDir, "pane-"+tag+"-"+agent+".json"))
    51		if err != nil {
    52			return ""
    53		}
    54		var p struct {
    55			Cwd string `json:"cwd"`
    56		}
    57		if json.Unmarshal(b, &p) != nil {
    58			return ""
    59		}
    60		return p.Cwd
    61	}

exec
/bin/zsh -lc 'git diff --name-only 9cc0f57521c3e44508adf3df006ffe3347faa265..HEAD -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollback-open bin/pair-changelog-open' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
exec
/bin/zsh -lc 'go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1' in /Users/xianxu/workspace/pair
 succeeded in 2726ms:
ok  	github.com/xianxu/pair/cmd/internal/contextcmd	0.206s
ok  	github.com/xianxu/pair/cmd/internal/scrollbackcmd	0.378s
ok  	github.com/xianxu/pair/cmd/pair-context	1.569s
?   	github.com/xianxu/pair/cmd/pair-scrollback-render	[no test files]
ok  	github.com/xianxu/pair/cmd/internal/dispatcher	0.728s
ok  	github.com/xianxu/pair/cmd/pair-go	2.162s

codex
The focused tests pass and the live zellij/nvim/bin caller paths are untouched in the diff. I’m doing a final verification sweep (`go test ./...`, whitespace check, and atlas grep) so the verdict can separate observed evidence from assumptions.
exec
/bin/zsh -lc 'rg -n "pair-go context|pair-go scrollback-render|helper dispatch" atlas/architecture.md atlas/go-migration-inventory.md workshop/issues/000076-go-helper-dispatch.md workshop/plans/000076-go-helper-dispatch-plan.md' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
workshop/plans/000076-go-helper-dispatch-plan.md:5:**Goal:** Add `pair-go context` and `pair-go scrollback-render` routes that reuse the existing helper implementations while preserving the legacy helper binaries.
workshop/plans/000076-go-helper-dispatch-plan.md:231:    // Run: pair-go context T claude
workshop/plans/000076-go-helper-dispatch-plan.md:236:This is the representative compatibility proof required by the issue done-when. It should fail before the dispatcher route exists because `pair-go context` is still planned-only.
workshop/plans/000076-go-helper-dispatch-plan.md:242:Expected: FAIL because `pair-go context` returns planned-but-not-implemented.
workshop/plans/000076-go-helper-dispatch-plan.md:309:Expected: PASS; the equivalence test demonstrates the legacy `pair-context` binary and `pair-go context` process path produce identical stdout/stderr/exit code on the same fixture.
workshop/plans/000076-go-helper-dispatch-plan.md:330:- `atlas/architecture.md` to say `pair-go context` and `pair-go scrollback-render` are implemented helper routes while public launcher and live zellij/nvim callers remain legacy.
workshop/plans/000076-go-helper-dispatch-plan.md:348:rg -n "pair-go context|pair-go scrollback-render|helper dispatch" atlas/architecture.md atlas/go-migration-inventory.md
workshop/plans/000076-go-helper-dispatch-plan.md:359:sdlc close --issue 76 --verified 'go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1; go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1; make pair-context pair-scrollback-render pair-go; make -B pair-context pair-scrollback-render pair-go; go test ./... -count=1; git diff live callers empty; rg atlas helper dispatch; git diff --check'
workshop/issues/000076-go-helper-dispatch.md:13:# pair Go helper dispatch
workshop/issues/000076-go-helper-dispatch.md:71:- 2026-06-30: closed — go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1; go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1; make pair-context pair-scrollback-render pair-go; make -B pair-context pair-scrollback-render pair-go; go test ./... -count=1; git diff -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollback-open bin/pair-changelog-open empty; rg atlas helper dispatch; git diff --check
workshop/issues/000076-go-helper-dispatch.md:73:Claimed after #75 landed. Narrowed the first helper dispatch slice to `context` and `scrollback-render`: they are useful enough to prove the dispatcher path, but low-risk enough to avoid long-running PTY, model, git commit/push, or public launcher behavior. Existing zellij/nvim/shell callers stay on legacy binary names for this milestone (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`).
workshop/issues/000076-go-helper-dispatch.md:75:Extracted `cmd/internal/contextcmd` and `cmd/internal/scrollbackcmd` runners so legacy binaries and dispatcher routes share implementation (`ARCH-DRY`, `ARCH-PURE`). Added `pair-go context` and `pair-go scrollback-render` dispatcher routes plus a process-level equivalence test proving `pair-go context` matches `pair-context` stdout/stderr/exit code on the same fixture. Updated atlas to record the current helper-dispatch state and unchanged live shell/Lua callers.
workshop/issues/000076-go-helper-dispatch.md:77:Verification before close: `go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1`; `go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1`; `make pair-context pair-scrollback-render pair-go`; `make -B pair-context pair-scrollback-render pair-go`; `go test ./... -count=1`; `git diff -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollback-open bin/pair-changelog-open` empty; atlas grep found `pair-go context`, `pair-go scrollback-render`, and helper dispatch; `git diff --check`.
atlas/go-migration-inventory.md:57:| `bin/pair-context` / `cmd/pair-context` / `cmd/internal/contextcmd` | Go binary plus shared runner | `bin/pair-title.sh`; development-only `pair-go context` | `pair-context <tag> <agent>` and `pair-go context <tag> <agent>` print the same humanized token count or nothing; tolerant exit 0 on failure. | Reads `PAIR_DATA_DIR`, `pane-<tag>-<agent>.json`, config, native transcripts. | implemented helper route in `pair-go context`; legacy binary retained while title poller calls it | P1 |
atlas/go-migration-inventory.md:58:| `bin/pair-scrollback-render` / `cmd/pair-scrollback-render` / `cmd/internal/scrollbackcmd` | Go binary plus shared runner | `bin/pair-scrollback-open`, `bin/pair-changelog-open`, `nvim/scrollback.lua` refresh; development-only `pair-go scrollback-render` | `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out` and `pair-go scrollback-render ...`; nonzero on render/write failure. | Reads `.raw` and `.events.jsonl`; atomically writes `.ansi` or cleaned text. | implemented helper route in `pair-go scrollback-render`; legacy binary retained for shell/Lua callers | P0 |
atlas/go-migration-inventory.md:126:  need helper dispatch to exist first, and helper dispatch does not need the
atlas/go-migration-inventory.md:128:- #76 established the first helper-dispatch pattern with `pair-go context` and
atlas/go-migration-inventory.md:129:  `pair-go scrollback-render`, backed by shared internal runners while legacy
atlas/architecture.md:59:`pair-go context <tag> <agent>` and `pair-go scrollback-render ...`. Both routes

exec
/bin/zsh -lc 'git diff --check 9cc0f57521c3e44508adf3df006ffe3347faa265..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
exec
/bin/zsh -lc 'go test ./... -count=1' in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc 'go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1' in /Users/xianxu/workspace/pair
 succeeded in 4667ms:
ok  	github.com/xianxu/pair/cmd/pair-go	4.323s

 succeeded in 17385ms:
ok  	github.com/xianxu/pair/cmd/internal/adapt	0.181s
ok  	github.com/xianxu/pair/cmd/internal/contextcmd	0.400s
ok  	github.com/xianxu/pair/cmd/internal/ctxmeter	0.547s
ok  	github.com/xianxu/pair/cmd/internal/dispatcher	0.678s
ok  	github.com/xianxu/pair/cmd/internal/launcher	2.769s
ok  	github.com/xianxu/pair/cmd/internal/model	3.612s
ok  	github.com/xianxu/pair/cmd/internal/scrollbackcmd	1.574s
ok  	github.com/xianxu/pair/cmd/internal/transcript	1.313s
ok  	github.com/xianxu/pair/cmd/pair-changelog	16.830s
ok  	github.com/xianxu/pair/cmd/pair-context	4.600s
ok  	github.com/xianxu/pair/cmd/pair-continuation	7.113s
ok  	github.com/xianxu/pair/cmd/pair-go	5.317s
?   	github.com/xianxu/pair/cmd/pair-scribe	[no test files]
?   	github.com/xianxu/pair/cmd/pair-scrollback-render	[no test files]
ok  	github.com/xianxu/pair/cmd/pair-slug	8.138s
ok  	github.com/xianxu/pair/cmd/pair-wrap	2.264s

codex
The full Go suite and diff checks pass. I’m doing the build-target check next because this change touches `Makefile.local` prerequisites; stale incremental rebuilds are the main remaining risk in this boundary.
exec
/bin/zsh -lc 'make pair-context pair-scrollback-render pair-go && make -B pair-context pair-scrollback-render pair-go' in /Users/xianxu/workspace/pair
 succeeded in 1461ms:
go: writing stat cache: open /Users/xianxu/go/pkg/mod/cache/download/github.com/xianxu/pair/@v/v0.0.0-20260630192640-f4da3a98205d.info284224854.tmp: operation not permitted
go build -o bin/pair-context ./cmd/pair-context
go: writing stat cache: open /Users/xianxu/go/pkg/mod/cache/download/github.com/xianxu/pair/@v/v0.0.0-20260630192640-f4da3a98205d.info417925077.tmp: operation not permitted
go build -o bin/pair-scrollback-render ./cmd/pair-scrollback-render
go: writing stat cache: open /Users/xianxu/go/pkg/mod/cache/download/github.com/xianxu/pair/@v/v0.0.0-20260630192640-f4da3a98205d.info193136689.tmp: operation not permitted
go build -o bin/pair-go ./cmd/pair-go
go: writing stat cache: open /Users/xianxu/go/pkg/mod/cache/download/github.com/xianxu/pair/@v/v0.0.0-20260630192640-f4da3a98205d.info498987555.tmp: operation not permitted

codex
The make targets completed with exit 0, though Go emitted sandbox-related stat-cache warnings while trying to write under `/Users/xianxu/go/pkg/mod/cache`; the actual builds continued. I’m checking line references for the final review response now.
exec
/bin/zsh -lc "nl -ba cmd/internal/contextcmd/contextcmd_test.go | sed -n '1,120p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package contextcmd
     2	
     3	import (
     4		"bytes"
     5		"os"
     6		"path/filepath"
     7		"strings"
     8		"testing"
     9	)
    10	
    11	func TestRunClaude(t *testing.T) {
    12		home := t.TempDir()
    13		data := filepath.Join(home, "data")
    14		cwd := filepath.Join(home, "repo")
    15		enc := strings.NewReplacer(".", "-", "/", "-").Replace(cwd)
    16		proj := filepath.Join(home, ".claude", "projects", enc)
    17		mustMkdir(t, data)
    18		mustMkdir(t, cwd)
    19		mustMkdir(t, proj)
    20		mustWrite(t, filepath.Join(data, "config-T-claude.json"), `{"session_id":"sid1"}`)
    21		mustWrite(t, filepath.Join(data, "pane-T-claude.json"), `{"pane_id":"7","cwd":"`+cwd+`","cwd_display":"~/repo"}`)
    22		mustWrite(t, filepath.Join(proj, "sid1.jsonl"),
    23			`{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":397556,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`)
    24	
    25		var stdout bytes.Buffer
    26		code := Run([]string{"T", "claude"}, Env{Home: home, PairDataDir: data}, &stdout)
    27		if code != 0 {
    28			t.Fatalf("code = %d, want 0", code)
    29		}
    30		if got := strings.TrimSpace(stdout.String()); got != "398k" {
    31			t.Fatalf("stdout = %q, want 398k", stdout.String())
    32		}
    33	}
    34	
    35	func TestRunMissingConfigPrintsNothing(t *testing.T) {
    36		home := t.TempDir()
    37		var stdout bytes.Buffer
    38		code := Run([]string{"T", "claude"}, Env{Home: home, PairDataDir: filepath.Join(home, "empty")}, &stdout)
    39		if code != 0 {
    40			t.Fatalf("code = %d, want 0", code)
    41		}
    42		if stdout.String() != "" {
    43			t.Fatalf("stdout = %q, want empty", stdout.String())
    44		}
    45	}
    46	
    47	func mustMkdir(t *testing.T, d string) {
    48		t.Helper()
    49		if err := os.MkdirAll(d, 0o755); err != nil {
    50			t.Fatal(err)
    51		}
    52	}
    53	
    54	func mustWrite(t *testing.T, p, s string) {
    55		t.Helper()
    56		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
    57			t.Fatal(err)
    58		}
    59	}

exec
/bin/zsh -lc "nl -ba cmd/internal/scrollbackcmd/scrollbackcmd_test.go | sed -n '1,100p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package scrollbackcmd
     2	
     3	import (
     4		"bytes"
     5		"io"
     6		"os"
     7		"path/filepath"
     8		"strings"
     9		"testing"
    10	)
    11	
    12	func TestRunUsage(t *testing.T) {
    13		var stderr bytes.Buffer
    14		code := Run([]string{}, io.Discard, &stderr)
    15		if code != 2 {
    16			t.Fatalf("code = %d, want 2", code)
    17		}
    18		if !strings.Contains(stderr.String(), "usage: pair-scrollback-render") {
    19			t.Fatalf("stderr missing usage:\n%s", stderr.String())
    20		}
    21	}
    22	
    23	func TestRunWritesOutput(t *testing.T) {
    24		dir := t.TempDir()
    25		rawPath := filepath.Join(dir, "in.raw")
    26		evPath := filepath.Join(dir, "in.events.jsonl")
    27		outPath := filepath.Join(dir, "out.ansi")
    28		if err := os.WriteFile(rawPath, []byte("hello\r\n"), 0o644); err != nil {
    29			t.Fatal(err)
    30		}
    31		events := `{"type":"resize","offset":0,"cols":20,"rows":5}` + "\n"
    32		if err := os.WriteFile(evPath, []byte(events), 0o644); err != nil {
    33			t.Fatal(err)
    34		}
    35	
    36		var stderr bytes.Buffer
    37		code := Run([]string{rawPath, evPath, outPath}, io.Discard, &stderr)
    38		if code != 0 {
    39			t.Fatalf("code = %d, want 0; stderr:\n%s", code, stderr.String())
    40		}
    41		body, err := os.ReadFile(outPath)
    42		if err != nil {
    43			t.Fatalf("read out: %v", err)
    44		}
    45		if !strings.Contains(string(body), "hello") {
    46			t.Fatalf("output missing rendered text:\n%s", string(body))
    47		}
    48	}

exec
/bin/zsh -lc "nl -ba cmd/pair-go/helper_equivalence_test.go | sed -n '1,130p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package main
     2	
     3	import (
     4		"bytes"
     5		"os"
     6		"os/exec"
     7		"path/filepath"
     8		"strings"
     9		"testing"
    10	)
    11	
    12	func TestPairGoContextMatchesLegacyPairContext(t *testing.T) {
    13		bin := t.TempDir()
    14		pairContext := filepath.Join(bin, "pair-context")
    15		pairGo := filepath.Join(bin, "pair-go")
    16		buildCommand(t, pairContext, "../pair-context")
    17		buildCommand(t, pairGo, ".")
    18	
    19		home, data := writeContextFixture(t)
    20		env := append(os.Environ(), "HOME="+home, "PAIR_DATA_DIR="+data)
    21	
    22		legacy := runCommand(t, env, pairContext, "T", "claude")
    23		dispatch := runCommand(t, env, pairGo, "context", "T", "claude")
    24		if dispatch.code != legacy.code || dispatch.stdout != legacy.stdout || dispatch.stderr != legacy.stderr {
    25			t.Fatalf("pair-go context mismatch\nlegacy:   code=%d stdout=%q stderr=%q\ndispatch: code=%d stdout=%q stderr=%q",
    26				legacy.code, legacy.stdout, legacy.stderr,
    27				dispatch.code, dispatch.stdout, dispatch.stderr)
    28		}
    29	}
    30	
    31	func buildCommand(t *testing.T, out, pkg string) {
    32		t.Helper()
    33		cmd := exec.Command("go", "build", "-o", out, pkg)
    34		if body, err := cmd.CombinedOutput(); err != nil {
    35			t.Fatalf("build %s: %v\n%s", pkg, err, string(body))
    36		}
    37	}
    38	
    39	type commandResult struct {
    40		code   int
    41		stdout string
    42		stderr string
    43	}
    44	
    45	func runCommand(t *testing.T, env []string, name string, args ...string) commandResult {
    46		t.Helper()
    47		cmd := exec.Command(name, args...)
    48		cmd.Env = env
    49		var stdout, stderr bytes.Buffer
    50		cmd.Stdout = &stdout
    51		cmd.Stderr = &stderr
    52		err := cmd.Run()
    53		code := 0
    54		if err != nil {
    55			exit, ok := err.(*exec.ExitError)
    56			if !ok {
    57				t.Fatalf("run %s: %v", name, err)
    58			}
    59			code = exit.ExitCode()
    60		}
    61		return commandResult{code: code, stdout: stdout.String(), stderr: stderr.String()}
    62	}
    63	
    64	func writeContextFixture(t *testing.T) (home, data string) {
    65		t.Helper()
    66		home = t.TempDir()
    67		data = filepath.Join(home, "data")
    68		cwd := filepath.Join(home, "repo")
    69		enc := strings.NewReplacer(".", "-", "/", "-").Replace(cwd)
    70		proj := filepath.Join(home, ".claude", "projects", enc)
    71		mustMkdir(t, data)
    72		mustMkdir(t, cwd)
    73		mustMkdir(t, proj)
    74		mustWrite(t, filepath.Join(data, "config-T-claude.json"), `{"session_id":"sid1"}`)
    75		mustWrite(t, filepath.Join(data, "pane-T-claude.json"), `{"pane_id":"7","cwd":"`+cwd+`","cwd_display":"~/repo"}`)
    76		mustWrite(t, filepath.Join(proj, "sid1.jsonl"),
    77			`{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":397556,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`)
    78		return home, data
    79	}
    80	
    81	func mustMkdir(t *testing.T, d string) {
    82		t.Helper()
    83		if err := os.MkdirAll(d, 0o755); err != nil {
    84			t.Fatal(err)
    85		}
    86	}
    87	
    88	func mustWrite(t *testing.T, p, s string) {
    89		t.Helper()
    90		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
    91			t.Fatal(err)
    92		}
    93	}

exec
/bin/zsh -lc "nl -ba atlas/architecture.md | sed -n '50,72p' && nl -ba atlas/go-migration-inventory.md | sed -n '52,62p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
    50	`bin/pair`; `pair-go launch` parses the launcher subset, resolves the Pair data
    51	dir, snapshots fakeable zellij/history state, and computes create/attach/picker
    52	decisions through the pure `cmd/internal/launcher` core. It then stops with an
    53	explicit prototype message before invoking zellij. Real zellij lifecycle,
    54	prompt/fzf UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation,
    55	rename, config/session migration, and title-poller behavior remain shell-owned
    56	until later migration issues.
    57	
    58	As of #76, the same dispatcher also has the first implemented helper routes:
    59	`pair-go context <tag> <agent>` and `pair-go scrollback-render ...`. Both routes
    60	call shared internal Go runners used by the legacy `pair-context` and
    61	`pair-scrollback-render` binaries (`ARCH-DRY`), so behavior stays aligned while
    62	the old command names remain the live integration surface. `bin/pair-title.sh`,
    63	`bin/pair-scrollback-open`, `bin/pair-changelog-open`, and
    64	`nvim/scrollback.lua` have not moved to the dispatcher yet.
    65	
    66	Native integration layers stay native: `nvim/*.lua` remains the bundled Neovim
    67	surface and `zellij/*.kdl` remains the zellij layout/config surface. Packaging
    68	may embed those assets or install them adjacent to the binary, but the migration
    69	does not force Lua or KDL into Go.
    70	
    71	The migration is deliberately staged through issue #73 onward. Each step must be
    72	merge-safe: after any sub-issue lands, the public `pair` command, `pair-dev`,
    52	| `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
    53	| `zellij/layouts/main.kdl` | zellij native asset | `bin/pair` via `zellij --new-session-with-layout` | Defines agent and draft panes; shell expands Pair env at pane start. | Calls `pair-wrap`; calls `nvim -u "$PAIR_HOME/nvim/init.lua"`; writes `pane-<tag>-<agent>.json`; writes draft nvim pid file. | native-asset, packaged adjacent/embedded | P0 |
    54	| `zellij/config.kdl` | zellij native asset | zellij session config from `bin/pair` | Global keybinds, copy command, scrollback buffer, pane frames. | Calls `copy-on-select.sh`, `pair-help`, `pair-scrollback-open`, `pair-changelog-open`; routes quit/restart/compact through nvim functions. | native-asset, packaged adjacent/embedded | P0 |
    55	| `bin/pair-wrap` / `cmd/pair-wrap` | Go binary | zellij agent pane | `pair-wrap [--scrollback-log PATH] agent [args...]`; transparent PTY proxy; long-running; failure in diagnostics is swallowed. | Reads Pair env and agent command; writes `agent-output-<tag>`, `agent-pid-<tag>`, scrollback `.raw`/`.events.jsonl`, image capture files; may invoke `pair-slug`. | go-subcommand `pair wrap`; keep old binary as compat alias for KDL until caller moves | P0 |
    56	| `bin/pair-slug` / `cmd/pair-slug` | Go binary | `pair-wrap` turn-end hook, tests | Env-driven, no stdin; resolves native transcript, proposes slug; exits 0 on most failures. | Requires `PAIR_TAG`, `PAIR_DATA_DIR`; reads config/transcripts/git branch; writes `slug-proposed-<tag>`; optional `PAIR_SLUG_*`, `OPENAI_API_KEY`. | go-subcommand `pair slug`; legacy binary retained during #76 | P1 |
    57	| `bin/pair-context` / `cmd/pair-context` / `cmd/internal/contextcmd` | Go binary plus shared runner | `bin/pair-title.sh`; development-only `pair-go context` | `pair-context <tag> <agent>` and `pair-go context <tag> <agent>` print the same humanized token count or nothing; tolerant exit 0 on failure. | Reads `PAIR_DATA_DIR`, `pane-<tag>-<agent>.json`, config, native transcripts. | implemented helper route in `pair-go context`; legacy binary retained while title poller calls it | P1 |
    58	| `bin/pair-scrollback-render` / `cmd/pair-scrollback-render` / `cmd/internal/scrollbackcmd` | Go binary plus shared runner | `bin/pair-scrollback-open`, `bin/pair-changelog-open`, `nvim/scrollback.lua` refresh; development-only `pair-go scrollback-render` | `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out` and `pair-go scrollback-render ...`; nonzero on render/write failure. | Reads `.raw` and `.events.jsonl`; atomically writes `.ansi` or cleaned text. | implemented helper route in `pair-go scrollback-render`; legacy binary retained for shell/Lua callers | P0 |
    59	| `bin/pair-changelog` / `cmd/pair-changelog` | Go binary | `bin/pair-changelog-open` | `pair-changelog --cleaned F --log F --anchor F [--agent A] [--model M]`; exits nonzero on required read/model/write failure. | Reads cleaned scrollback/log/anchor; calls agent model through internal model runner; atomically writes log and anchor. | go-subcommand `pair changelog`; legacy binary retained for opener | P1 |
    60	| `bin/pair-continuation` / `cmd/pair-continuation` | Go binary | nvim compaction prompt instructions, operator/agent shell | `pair-continuation --slug S --agent A --issues CSV --body-file F [--repo-root R ...]`; writes and commits continuation; nonzero on validation/git failure. | Reads body/stdin, git repo state; writes `workshop/continuation/*.md`; runs git commit/push. | go-subcommand `pair continuation`; legacy binary retained for agent instructions until docs change | P1 |
    61	| `bin/pair-scribe` / `cmd/pair-scribe` | Go binary | user shell rc outside Pair sessions | `pair-scribe -log PATH -- CMD [ARGS...]`; long-running PTY wrapper; SIGUSR1 pauses log, SIGUSR2 resumes. | Writes typescript log; wraps child PTY; independent of `PAIR_*`. | go-subcommand candidate is low value; may remain separate installed helper or become `pair scribe` with alias | P2 |
    62	| `cmd/internal/adapt` | Go helper package | `pair-wrap`, `pair-slug`, tests | Pure-ish emitter helpers plus file open seam; no command. | Writes `$PAIR_DATA_DIR/adapt-<tag>.jsonl`; schema shared with shell/Lua. | internal package, reuse behind dispatcher | P1 |

exec
/bin/zsh -lc 'nl -ba cmd/pair-context/main.go && nl -ba cmd/pair-scrollback-render/main.go' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	// pair-context — print one agent pane's current context size (humanized
     2	// token count), or nothing. Invoked as `pair-context <tag> <agent>` by the
     3	// pair-title poller. Tolerant: any failure prints nothing and exits 0, so a
     4	// hiccup never garbles the pane title.
     5	package main
     6	
     7	import (
     8		"os"
     9	
    10		"github.com/xianxu/pair/cmd/internal/contextcmd"
    11	)
    12	
    13	func main() {
    14		os.Exit(contextcmd.Run(os.Args[1:], contextcmd.EnvFromOS(), os.Stdout))
    15	}
     1	// scrollback-render — replay a pair-wrap raw capture through a VT100 emulator.
     2	package main
     3	
     4	import (
     5		"os"
     6	
     7		"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
     8	)
     9	
    10	func main() {
    11		os.Exit(scrollbackcmd.Run(os.Args[1:], os.Stdout, os.Stderr))
    12	}

codex
VERDICT: SHIP (confidence: high)

The #76 boundary delivers the stated helper-dispatch slice: `pair-go context` and `pair-go scrollback-render` route through shared runners, legacy binaries remain thin wrappers, live shell/Lua/zellij callers are untouched, and atlas/tracker updates match the new surface. I found no Critical or Important findings blocking close.

1. Strengths:
- `cmd/pair-context/main.go:13` and `cmd/pair-scrollback-render/main.go:10` are now thin process shells over shared internal runners.
- `cmd/internal/dispatcher/dispatcher.go:61` and `cmd/internal/dispatcher/dispatcher.go:63` add only the requested helper routes, with planned commands still rejected.
- `cmd/pair-go/helper_equivalence_test.go:12` proves `pair-go context` and legacy `pair-context` match stdout/stderr/exit code.
- Atlas records the new dispatcher helper surface and unchanged live callers at `atlas/architecture.md:58` and `atlas/go-migration-inventory.md:57`.

2. Critical findings:
- None.

3. Important findings:
- None.

4. Minor findings:
- `cmd/internal/scrollbackcmd/scrollbackcmd.go:395` now prints a fixed `pair-scrollback-render` usage string instead of the old `os.Args[0]`; acceptable for the new shared runner, but worth remembering if invocation-name-sensitive usage text ever matters.

5. Test coverage notes:
- Verified: `go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1`.
- Verified: `go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1`.
- Verified: `go test ./... -count=1`.
- Verified: `git diff --check` and live caller diff empty.
- Verified: `make pair-context pair-scrollback-render pair-go && make -B pair-context pair-scrollback-render pair-go` exited 0; Go emitted sandbox stat-cache warnings under `/Users/xianxu/go/pkg/mod/cache`, but builds completed.

6. Architectural notes for upcoming work:
- ARCH-DRY: Pass. Legacy binaries and dispatcher routes share `contextcmd.Run` / `scrollbackcmd.Run`.
- ARCH-PURE: Pass for this boundary. IO remains in command runners and process shells; pure scrollback rendering helpers remain directly unit-tested in the internal package.
- ARCH-PURPOSE: Pass. The issue promised first helper routes without moving live callers; the diff does exactly that and updates atlas.

7. Plan revision recommendations:
- None. The Core Concepts revision already aligns the plan with the shipped INTEGRATION surfaces.
tokens used
67,493
VERDICT: SHIP (confidence: high)

The #76 boundary delivers the stated helper-dispatch slice: `pair-go context` and `pair-go scrollback-render` route through shared runners, legacy binaries remain thin wrappers, live shell/Lua/zellij callers are untouched, and atlas/tracker updates match the new surface. I found no Critical or Important findings blocking close.

1. Strengths:
- `cmd/pair-context/main.go:13` and `cmd/pair-scrollback-render/main.go:10` are now thin process shells over shared internal runners.
- `cmd/internal/dispatcher/dispatcher.go:61` and `cmd/internal/dispatcher/dispatcher.go:63` add only the requested helper routes, with planned commands still rejected.
- `cmd/pair-go/helper_equivalence_test.go:12` proves `pair-go context` and legacy `pair-context` match stdout/stderr/exit code.
- Atlas records the new dispatcher helper surface and unchanged live callers at `atlas/architecture.md:58` and `atlas/go-migration-inventory.md:57`.

2. Critical findings:
- None.

3. Important findings:
- None.

4. Minor findings:
- `cmd/internal/scrollbackcmd/scrollbackcmd.go:395` now prints a fixed `pair-scrollback-render` usage string instead of the old `os.Args[0]`; acceptable for the new shared runner, but worth remembering if invocation-name-sensitive usage text ever matters.

5. Test coverage notes:
- Verified: `go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1`.
- Verified: `go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1`.
- Verified: `go test ./... -count=1`.
- Verified: `git diff --check` and live caller diff empty.
- Verified: `make pair-context pair-scrollback-render pair-go && make -B pair-context pair-scrollback-render pair-go` exited 0; Go emitted sandbox stat-cache warnings under `/Users/xianxu/go/pkg/mod/cache`, but builds completed.

6. Architectural notes for upcoming work:
- ARCH-DRY: Pass. Legacy binaries and dispatcher routes share `contextcmd.Run` / `scrollbackcmd.Run`.
- ARCH-PURE: Pass for this boundary. IO remains in command runners and process shells; pure scrollback rendering helpers remain directly unit-tested in the internal package.
- ARCH-PURPOSE: Pass. The issue promised first helper routes without moving live callers; the diff does exactly that and updates atlas.

7. Plan revision recommendations:
- None. The Core Concepts revision already aligns the plan with the shipped INTEGRATION surfaces.
