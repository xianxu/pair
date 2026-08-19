# Boundary Review — pair-context-usage#144 (whole-issue close)

| field | value |
|-------|-------|
| issue | 144 — Reject Codex subagent sessions during Pair identity discovery |
| repo | pair-context-usage |
| issue file | workshop/issues/000144-reject-codex-subagent-sessions-during-pair-identity-discovery.md |
| boundary | whole-issue close |
| milestone | — |
| window | 356ec01f5aaf6c48d3fcf73a7aff1bd090c8dc79..HEAD |
| command | sdlc close --issue 144 |
| reviewer | codex |
| timestamp | 2026-08-19T07:51:46-07:00 |
| verdict | REWORK |

## Review

Reading additional input from stdin...
OpenAI Codex v0.147.0
--------
workdir: /private/tmp/pair-context-usage
model: gpt-5.6-sol
provider: openai
approval: never
sandbox: workspace-write [workdir, /tmp, $TMPDIR, /tmp] (network access enabled)
reasoning effort: medium
reasoning summaries: none
session id: 01a01a7e-f685-7e93-931b-2b0fd616bfc1
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **pair-context-usage** repository.

- repository: pair-context-usage   (root: /private/tmp/pair-context-usage)
- issue:      pair-context-usage#144   (file: workshop/issues/000144-reject-codex-subagent-sessions-during-pair-identity-discovery.md)
- window:     Base: 356ec01f5aaf6c48d3fcf73a7aff1bd090c8dc79   Head: HEAD

Review the **pair-context-usage** repo and its tracker — the ariadne base-layer repo itself (changes here propagate to dependent repos). Do not assume any
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
  - Docs / atlas updated for new surface (see the Docs update gate).

## Plan-gate carry-forward (ariadne#187)

Read `workshop/plans/<issue-stem>-plan-gate.md` if it exists — the durable ledger of the
pre-implementation plan gate. It holds the findings that gate raised but deliberately did
NOT block on: Minor findings, and blocking ones demoted once the round cap was reached.
They were deferred to THIS boundary by design — that deferral is only safe because you
pick them up.

For each finding still listed under `## Open findings`, confirm the code either addresses
it or that it no longer applies. A still-valid deferred finding is a finding here, at its
original severity.

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

## Docs update gate (atlas + README, per AGENTS.md §8)

The boundary should update user-facing docs for any new surface introduced:

  - **atlas/** — new architectural surface, flow, or terminology. Scan the diff
    for new entity types, subcommands, conventions, file-tree locations. Any
    present without corresponding atlas/ changes in the same range = Important
    finding ("atlas update appears missing for <surface>").
  - **README.md** — new user-facing surface a reader runs or types: subcommands,
    flags, keybindings, config keys, install/usage steps. If the diff adds or
    changes such surface and README.md is not updated in the same range =
    Important finding ("README update appears missing for <surface>"). This is the
    class of gap that used to surface only at the merge-time `specs` judge (#142);
    catch it here, at the earliest gate, before the close verdict is recorded.

## Architecture (the at-review backstop — these matter most long-term)

Work through each of ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK explicitly, applying its at-review lens. The
full principle definitions are delivered in the ARCHITECTURE PRINCIPLES block
right after this prompt — for EACH marker, state pass or flag, and cite the
marker (e.g. ARCH-DRY) in any finding. Architecture is where review has the
least training signal and the longest-delayed payoff, so be deliberate here, not
holistic.

## Verdict + output

Begin your response with this fenced verdict block — the machine-read handoff:

```verdict
verdict: <SHIP | FIX-THEN-SHIP | REWORK>
confidence: <high | medium | low>
```

  SHIP           ready; ship it
  FIX-THEN-SHIP  ship after addressing the findings (non-blocking at the gate)
  REWORK         blocking; needs rework before shipping — fix + re-run

The fenced ```` ```verdict ```` block above is the **authoritative machine-read
handoff** — emit it as the first thing in your response. (A prose
`VERDICT: <TOKEN>` first line still satisfies the legacy contract as a fallback,
but the block is what the binary trusts.)

After the verdict block: a 1-paragraph summary — what worked, what blocks SHIP if
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


ARCHITECTURE PRINCIPLES — work through each of the 4 entries below explicitly, applying its `at-review` lens; cite the marker (e.g. ARCH-DRY) in any finding.

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

## ARCH-MOCK — Stateful external doubles

- **principle:** Every external binary or service dependency the system relies on
  has a stateful fake behind the same seam, modeling our current understanding of
  the dependency's behavior across calls. For libraries, services, and binaries we
  own, the storage/backend layer is backed by a portable folder of files and/or
  database configuration, so the component can be spun up without depending on
  production configuration or production databases. Integration and end-to-end
  tests run against the fake; scheduled/live conformance checks compare the
  fake's modeled behavior with the real binary or service so drift is detected
  and corrected.
- **at-plan:** Flag a design that shells out to, or calls, an external binary or
  service without naming the seam and stateful fake. For owned libraries, services,
  and binaries, also flag any design whose storage/backend depends on production
  configuration or databases instead of a portable file folder and/or database
  configuration. The plan should identify the dependency surface consumed, the
  fake's persisted state model, the owned component's portable backend shape,
  the integration or end-to-end tests that run against it, and the live
  conformance check cadence.
  Examples include `git`, GitHub/`gh`, and Google OAuth.
- **at-review:** Flag direct external calls outside the seam, stateless mocks for
  stateful interactions, tests that cannot run the stack against the fake, owned
  components that cannot boot from portable non-production storage/backend
  configuration, or a missing live conformance check for behavior we depend on. A
  fake satisfies this only when production flow and test flow share the same
  boundary.


OUTPUT CONTRACT (machine-read — do not deviate). LEAD your response with the
fenced ```verdict block shown above — that is the authoritative handoff the binary
reads (its `verdict:` value is one of the listed tokens). Everything after the block
is advisory: a non-blocking verdict WITH findings still PASSES the gate. A bare
`VERDICT: <TOKEN>` line is accepted only as a FALLBACK when the block is absent.

Diff:
diff --git a/atlas/session-identity.md b/atlas/session-identity.md
index 8c1fe72..c8411c5 100644
--- a/atlas/session-identity.md
+++ b/atlas/session-identity.md
@@ -139,6 +139,29 @@ The ledger is the source of truth for agent/config inference. The older
 `agent-<tag>` and `config-<tag>-<agent>.json` files remain as derived caches and
 compatibility surfaces for existing consumers.
 
+### Codex root identity
+
+A Codex rollout filename supplies only a candidate UUID; it does not prove
+which conversation owns the rollout. Pair authorizes an automatic Codex
+identity only when the rollout's first JSONL event is a matching
+`session_meta`, its `parent_thread_id` is absent or null, and its source is the
+observed root source `cli` or `exec`. Subagent, malformed, mismatched, unknown,
+oversized, and incomplete first events fail closed. Candidate scans continue
+past rejected rollouts so an open subagent cannot hide a later root candidate.
+
+The rule lives in `cmd/internal/transcript` and is shared by launcher live
+capture, session watching, context usage, slugging, and review targeting.
+Process-tree and birth-time discovery locate candidates only; neither grants
+identity by itself. Persisted Codex IDs are revalidated at automatic config
+picker and `Alt+n` restart boundaries. An invalid binding is removed from the
+config, its non-resume args are preserved for a fresh launch, and the operator
+is warned. Explicitly typed `codex resume <id>` remains user authority.
+
+Neovim deliberately does not inspect Codex processes or rollouts. Review
+target scoping uses the inherited `PAIR_SESSION_ID`, then Pair's config cache;
+when neither exists it remains unscoped until the Go watcher publishes a
+validated root identity.
+
 `agent-default-<agent>.json` is different from `config-<tag>-<agent>.json`: it
 has only `{agent,args}` and belongs to the repo/agent, not to a work tag or
 native conversation. Fresh `pair <agent>` creates use it as the lowest-priority
diff --git a/cmd/internal/codexsid/codexsid.go b/cmd/internal/codexsid/codexsid.go
index db6c7da..4c178ea 100644
--- a/cmd/internal/codexsid/codexsid.go
+++ b/cmd/internal/codexsid/codexsid.go
@@ -7,15 +7,12 @@ package codexsid
 
 import (
 	"os"
-	"os/exec"
 	"path/filepath"
-	"regexp"
 	"strings"
-)
 
-// rolloutRE matches ~/.codex/sessions/.../rollout-<...>-<uuid>.jsonl and captures
-// the session UUID.
-var rolloutRE = regexp.MustCompile(`/\.codex/sessions/.*/rollout-.*([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\.jsonl$`)
+	"github.com/xianxu/pair/cmd/internal/procutil"
+	"github.com/xianxu/pair/cmd/internal/transcript"
+)
 
 // ResolveSessionID reads the codex agent's root pid from
 // $dataDir/agent-pid-<tag>, BFS-walks its process descendants, and greps each
@@ -30,57 +27,12 @@ func ResolveSessionID(dataDir, tag string) string {
 	if root == "" {
 		return ""
 	}
-	for _, pid := range descendants(root) {
-		for _, name := range lsofNames(pid) {
-			if m := rolloutRE.FindStringSubmatch(name); m != nil {
-				return m[1]
+	for _, pid := range procutil.DescendantPIDs(root, procutil.ProcessChildren()) {
+		for _, name := range procutil.LsofNames(pid) {
+			if sid := transcript.ReadCodexRootSessionID(name); sid != "" {
+				return sid
 			}
 		}
 	}
 	return ""
 }
-
-// descendants returns root plus its transitive child pids (BFS over
-// `ps -axo pid=,ppid=`). On ps failure it degrades to just [root].
-func descendants(root string) []string {
-	out, err := exec.Command("ps", "-axo", "pid=,ppid=").Output()
-	if err != nil {
-		return []string{root}
-	}
-	children := map[string][]string{}
-	for _, line := range strings.Split(string(out), "\n") {
-		f := strings.Fields(line)
-		if len(f) != 2 {
-			continue
-		}
-		children[f[1]] = append(children[f[1]], f[0])
-	}
-	queue := []string{root}
-	seen := map[string]bool{root: true}
-	for i := 0; i < len(queue); i++ {
-		for _, c := range children[queue[i]] {
-			if c == "" || seen[c] {
-				continue
-			}
-			seen[c] = true
-			queue = append(queue, c)
-		}
-	}
-	return queue
-}
-
-// lsofNames returns the file paths a pid has open (`lsof -p <pid> -Fn`, the `n`
-// lines). Empty on any error.
-func lsofNames(pid string) []string {
-	out, err := exec.Command("lsof", "-p", pid, "-Fn").Output()
-	if err != nil {
-		return nil
-	}
-	var names []string
-	for _, line := range strings.Split(string(out), "\n") {
-		if strings.HasPrefix(line, "n") {
-			names = append(names, strings.TrimPrefix(line, "n"))
-		}
-	}
-	return names
-}
diff --git a/cmd/internal/codexsid/codexsid_test.go b/cmd/internal/codexsid/codexsid_test.go
index dd4fb10..039b531 100644
--- a/cmd/internal/codexsid/codexsid_test.go
+++ b/cmd/internal/codexsid/codexsid_test.go
@@ -6,23 +6,6 @@ import (
 	"testing"
 )
 
-func TestRolloutRE(t *testing.T) {
-	cases := []struct{ path, want string }{
-		{"/Users/x/.codex/sessions/2024/01/rollout-2024-01-01T00-00-00-aaaa1111-2222-3333-4444-555566667777.jsonl", "aaaa1111-2222-3333-4444-555566667777"},
-		{"/other/path.jsonl", ""},
-		{"/Users/x/.codex/sessions/rollout-nouuid.jsonl", ""},
-	}
-	for _, c := range cases {
-		got := ""
-		if m := rolloutRE.FindStringSubmatch(c.path); m != nil {
-			got = m[1]
-		}
-		if got != c.want {
-			t.Errorf("%q -> %q, want %q", c.path, got, c.want)
-		}
-	}
-}
-
 func TestResolveNoAgentPid(t *testing.T) {
 	if got := ResolveSessionID(t.TempDir(), "tag"); got != "" {
 		t.Fatalf("no pidfile -> empty, got %q", got)
@@ -35,3 +18,40 @@ func TestResolveNoAgentPid(t *testing.T) {
 		t.Fatalf("empty pidfile -> empty, got %q", got)
 	}
 }
+
+func TestResolveSessionIDSkipsSubagentRollout(t *testing.T) {
+	dataDir := t.TempDir()
+	home := t.TempDir()
+	if err := os.WriteFile(filepath.Join(dataDir, "agent-pid-tag"), []byte("10\n"), 0o644); err != nil {
+		t.Fatal(err)
+	}
+	rootSID := "019e8178-79c2-7862-91db-e8fa1be3b162"
+	subSID := "01a017b6-af00-7c91-a656-0611a3750669"
+	dir := filepath.Join(home, ".codex", "sessions", "2026", "05", "31")
+	if err := os.MkdirAll(dir, 0o755); err != nil {
+		t.Fatal(err)
+	}
+	rootPath := filepath.Join(dir, "rollout-root-"+rootSID+".jsonl")
+	subPath := filepath.Join(dir, "rollout-sub-"+subSID+".jsonl")
+	if err := os.WriteFile(rootPath, []byte(`{"type":"session_meta","payload":{"id":"`+rootSID+`","parent_thread_id":null,"source":"exec"}}`+"\n"), 0o644); err != nil {
+		t.Fatal(err)
+	}
+	if err := os.WriteFile(subPath, []byte(`{"type":"session_meta","payload":{"id":"`+subSID+`","parent_thread_id":"`+rootSID+`","source":{"subagent":{}}}}`+"\n"), 0o644); err != nil {
+		t.Fatal(err)
+	}
+
+	binDir := t.TempDir()
+	ps := "#!/bin/sh\nprintf ' 10 1\\n 11 10\\n'\n"
+	if err := os.WriteFile(filepath.Join(binDir, "ps"), []byte(ps), 0o755); err != nil {
+		t.Fatal(err)
+	}
+	lsof := "#!/bin/sh\nif [ \"$2\" = \"11\" ]; then printf 'p11\\nn" + subPath + "\\nn" + rootPath + "\\n'; else printf 'p%s\\n' \"$2\"; fi\n"
+	if err := os.WriteFile(filepath.Join(binDir, "lsof"), []byte(lsof), 0o755); err != nil {
+		t.Fatal(err)
+	}
+	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
+
+	if got := ResolveSessionID(dataDir, "tag"); got != rootSID {
+		t.Fatalf("ResolveSessionID = %q, want root %q", got, rootSID)
+	}
+}
diff --git a/cmd/internal/contextcmd/contextcmd.go b/cmd/internal/contextcmd/contextcmd.go
index d15101a..f119c02 100644
--- a/cmd/internal/contextcmd/contextcmd.go
+++ b/cmd/internal/contextcmd/contextcmd.go
@@ -52,7 +52,7 @@ func Run(args []string, env Env, stdout io.Writer) int {
 // isn't resolvable yet.
 func TranscriptPath(env Env, tag, agent string) string {
 	dataDir := resolveDataDir(env)
-	sid := transcript.SessionID(dataDir, tag, agent)
+	sid := transcript.SessionID(dataDir, tag, agent, env.Home)
 	if sid == "" {
 		return ""
 	}
diff --git a/cmd/internal/contextcmd/contextcmd_test.go b/cmd/internal/contextcmd/contextcmd_test.go
index 2e07506..f2b63f9 100644
--- a/cmd/internal/contextcmd/contextcmd_test.go
+++ b/cmd/internal/contextcmd/contextcmd_test.go
@@ -44,6 +44,27 @@ func TestRunMissingConfigPrintsNothing(t *testing.T) {
 	}
 }
 
+func TestRunCodexPollutedSubagentConfigPrintsNothing(t *testing.T) {
+	home := t.TempDir()
+	data := filepath.Join(home, "data")
+	sid := "01a017b6-af00-7c91-a656-0611a3750669"
+	parent := "019e8178-79c2-7862-91db-e8fa1be3b162"
+	rollout := filepath.Join(home, ".codex", "sessions", "2026", "08", "18", "rollout-sub-"+sid+".jsonl")
+	mustMkdir(t, data)
+	mustMkdir(t, filepath.Dir(rollout))
+	mustWrite(t, filepath.Join(data, "config-T-codex.json"), `{"session_id":"`+sid+`"}`)
+	mustWrite(t, rollout, `{"type":"session_meta","payload":{"id":"`+sid+`","parent_thread_id":"`+parent+`","source":{"subagent":{}}}}`+"\n"+
+		`{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"total_tokens":398000}}}}`+"\n")
+
+	var stdout bytes.Buffer
+	if code := Run([]string{"T", "codex"}, Env{Home: home, PairDataDir: data}, &stdout); code != 0 {
+		t.Fatalf("code = %d, want 0", code)
+	}
+	if stdout.String() != "" {
+		t.Fatalf("stdout = %q, want empty for subagent config", stdout.String())
+	}
+}
+
 func mustMkdir(t *testing.T, d string) {
 	t.Helper()
 	if err := os.MkdirAll(d, 0o755); err != nil {
diff --git a/cmd/internal/launcher/createflow.go b/cmd/internal/launcher/createflow.go
index 372b6e4..0b10008 100644
--- a/cmd/internal/launcher/createflow.go
+++ b/cmd/internal/launcher/createflow.go
@@ -96,7 +96,15 @@ func RunLaunch(opts LaunchOptions, rt Runtime, stderr io.Writer) (int, error) {
 		}
 
 		configPath := resolveConfigPath(rt, env.DataDir, rTag, rAgent)
-		plan := planRestart(m, rTag, rAgent, readSavedConfig(rt, configPath))
+		saved := readSavedConfig(rt, configPath)
+		savedSessionID := saved.SessionID
+		var quarantine bool
+		saved, quarantine = decideAutomaticResumeConfig(rAgent, saved, rt.AgentSessionExists(rAgent, savedSessionID, env.Cwd))
+		if quarantine {
+			fmt.Fprintf(stderr, "pair: saved session %q for %s is not available; starting fresh\n", savedSessionID, rAgent)
+			rt.Remove(configPath)
+		}
+		plan := planRestart(m, rTag, rAgent, saved)
 		if plan.DropConfig {
 			rt.Remove(configPath) // Shift+Alt+N / compaction: drop the config so create mints fresh.
 		}
@@ -643,11 +651,17 @@ func runConfigPicker(rt Runtime, configPath string, saved savedConfig, agent, ch
 		return 0, true // unusable config — proceed as if none.
 	}
 
-	savedArgsClean := persistedConfigArgs(saved.Args)
-	hasResumable := rt.AgentSessionExists(agent, saved.SessionID, cwd)
-	if saved.SessionID != "" && !hasResumable {
-		fmt.Fprintf(stderr, "pair: saved session %q for %s is not available; starting fresh\n", saved.SessionID, agent)
+	savedSessionID := saved.SessionID
+	hasResumable := rt.AgentSessionExists(agent, savedSessionID, cwd)
+	var quarantine bool
+	saved, quarantine = decideAutomaticResumeConfig(agent, saved, hasResumable)
+	if savedSessionID != "" && !hasResumable {
+		fmt.Fprintf(stderr, "pair: saved session %q for %s is not available; starting fresh\n", savedSessionID, agent)
+	}
+	if quarantine {
+		rt.Remove(configPath)
 	}
+	savedArgsClean := persistedConfigArgs(saved.Args)
 	choices := buildConfigChoices(hasResumable, savedArgsClean, *agentArgs, saved.SessionID)
 
 	labels := make([]string, len(choices))
diff --git a/cmd/internal/launcher/createflow_test.go b/cmd/internal/launcher/createflow_test.go
index e9b4028..4d8dabc 100644
--- a/cmd/internal/launcher/createflow_test.go
+++ b/cmd/internal/launcher/createflow_test.go
@@ -7,6 +7,7 @@ import (
 	"os"
 	"reflect"
 	"regexp"
+	"slices"
 	"strings"
 	"testing"
 	"time"
@@ -1050,6 +1051,9 @@ func TestRunLaunchTagRestartPickerWarnsWhenSavedSessionIsStale(t *testing.T) {
 	if !strings.Contains(stderr.String(), `saved session "CX-9" for codex is not available; starting fresh`) {
 		t.Fatalf("stderr missing stale-session warning: %s", stderr.String())
 	}
+	if !slices.Contains(rt.removed, "/data/config-cx-codex.json") {
+		t.Fatalf("removed = %v, want stale Codex config quarantined", rt.removed)
+	}
 }
 
 // Picking "new" drops the stale config.
@@ -1210,6 +1214,68 @@ func TestRunLaunchResumeUsesLedgerAgentAndArgsWhenConfigMissing(t *testing.T) {
 	}
 }
 
+func TestRunLaunchRejectsInvalidLedgerCodexSession(t *testing.T) {
+	rt := newFakeRuntime()
+	rt.ledger["work"] = []LedgerEntry{{
+		Agent:      "codex",
+		Args:       []string{"--search"},
+		SessionID:  "SUBAGENT",
+		LastActive: time.Unix(1_700_000_010, 0),
+	}}
+	rt.pickFunc = func(header string, options []string) string {
+		for _, o := range options {
+			if strings.Contains(o, "use saved params") {
+				return o
+			}
+		}
+		return ""
+	}
+
+	var stderr bytes.Buffer
+	code, err := RunLaunch(baseOpts(LaunchArgs{ForcedTag: "work"}), rt, &stderr)
+	if err != nil || code != 0 {
+		t.Fatalf("code=%d err=%v stderr=%s", code, err, stderr.String())
+	}
+	if strings.Contains(rt.env["PAIR_AGENT_ARGS"], "SUBAGENT") {
+		t.Fatalf("PAIR_AGENT_ARGS = %q, must not resume rejected session", rt.env["PAIR_AGENT_ARGS"])
+	}
+	if rt.env["PAIR_AGENT_ARGS"] != "--search --no-alt-screen" {
+		t.Fatalf("PAIR_AGENT_ARGS = %q", rt.env["PAIR_AGENT_ARGS"])
+	}
+	if !slices.Contains(rt.removed, "/data/config-work-codex.json") {
+		t.Fatalf("removed = %v, want canonical config quarantine", rt.removed)
+	}
+	if !strings.Contains(stderr.String(), `saved session "SUBAGENT" for codex is not available; starting fresh`) {
+		t.Fatalf("stderr missing stale-session warning: %s", stderr.String())
+	}
+}
+
+func TestRunLaunchAltNRestartRejectsInvalidSavedCodexSession(t *testing.T) {
+	rt := newFakeRuntime()
+	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":["--search"],"session_id":"SUBAGENT"}`
+	rt.restartMarkers["📁work-cx"] = RestartMarker{Tag: "cx", Agent: "codex"}
+
+	opts := baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"})
+	opts.SkipConfigPicker = true
+	var stderr bytes.Buffer
+	code, err := RunLaunch(opts, rt, &stderr)
+	if err != nil || code != 0 {
+		t.Fatalf("code=%d err=%v stderr=%s", code, err, stderr.String())
+	}
+	if rt.launchCount != 2 {
+		t.Fatalf("launchCount = %d, want initial launch plus Alt+n relaunch", rt.launchCount)
+	}
+	if strings.Contains(rt.env["PAIR_AGENT_ARGS"], "SUBAGENT") {
+		t.Fatalf("PAIR_AGENT_ARGS = %q, must not resume rejected session", rt.env["PAIR_AGENT_ARGS"])
+	}
+	if !slices.Contains(rt.removed, "/data/config-cx-codex.json") {
+		t.Fatalf("removed = %v, want stale Codex config quarantined", rt.removed)
+	}
+	if !strings.Contains(stderr.String(), `saved session "SUBAGENT" for codex is not available; starting fresh`) {
+		t.Fatalf("stderr missing stale-session warning: %s", stderr.String())
+	}
+}
+
 // With nothing on disk to infer from, the agent defaults to claude.
 func TestRunLaunchResumeDefaultsClaude(t *testing.T) {
 	rt := newFakeRuntime()
diff --git a/cmd/internal/launcher/markers.go b/cmd/internal/launcher/markers.go
index 13c4e6b..1c7328f 100644
--- a/cmd/internal/launcher/markers.go
+++ b/cmd/internal/launcher/markers.go
@@ -56,6 +56,17 @@ type restartPlan struct {
 	ContinueSlug string // #55 compaction re-entry: re-seed the draft from this slug
 }
 
+// decideAutomaticResumeConfig rejects only persisted Codex bindings that no
+// longer identify a verified root rollout. Keep the saved launch parameters so
+// callers can still offer a fresh launch with the user's prior flags.
+func decideAutomaticResumeConfig(agent string, saved savedConfig, sessionValid bool) (savedConfig, bool) {
+	if agent != "codex" || saved.SessionID == "" || sessionValid {
+		return saved, false
+	}
+	saved.SessionID = ""
+	return saved, true
+}
+
 // planRestart maps a restart marker + the RESOLVED (tag, agent) + saved config
 // into the next launch (#99 M5b makes rename/continue native). The caller has
 // already applied the marker's tag/agent preference AND any rename_to move before
diff --git a/cmd/internal/launcher/markers_test.go b/cmd/internal/launcher/markers_test.go
index 2a70b47..4567ce6 100644
--- a/cmd/internal/launcher/markers_test.go
+++ b/cmd/internal/launcher/markers_test.go
@@ -64,3 +64,27 @@ func TestPlanRestart(t *testing.T) {
 		t.Fatalf("continue re-entry = %+v", pc)
 	}
 }
+
+func TestDecideAutomaticResumeConfig(t *testing.T) {
+	tests := []struct {
+		name         string
+		agent        string
+		saved        savedConfig
+		sessionValid bool
+		want         savedConfig
+		quarantine   bool
+	}{
+		{"valid codex root", "codex", savedConfig{Agent: "codex", Args: []string{"--search"}, SessionID: "ROOT"}, true, savedConfig{Agent: "codex", Args: []string{"--search"}, SessionID: "ROOT"}, false},
+		{"invalid codex candidate", "codex", savedConfig{Agent: "codex", Args: []string{"--search"}, SessionID: "SUB"}, false, savedConfig{Agent: "codex", Args: []string{"--search"}}, true},
+		{"empty codex session", "codex", savedConfig{Agent: "codex", Args: []string{"--search"}}, false, savedConfig{Agent: "codex", Args: []string{"--search"}}, false},
+		{"non-codex unchanged", "claude", savedConfig{Agent: "claude", Args: []string{"--flag"}, SessionID: "STALE"}, false, savedConfig{Agent: "claude", Args: []string{"--flag"}, SessionID: "STALE"}, false},
+	}
+	for _, tt := range tests {
+		t.Run(tt.name, func(t *testing.T) {
+			got, quarantine := decideAutomaticResumeConfig(tt.agent, tt.saved, tt.sessionValid)
+			if !reflect.DeepEqual(got, tt.want) || quarantine != tt.quarantine {
+				t.Fatalf("got (%+v, %t), want (%+v, %t)", got, quarantine, tt.want, tt.quarantine)
+			}
+		})
+	}
+}
diff --git a/cmd/internal/launcher/osruntime.go b/cmd/internal/launcher/osruntime.go
index 471d402..85a3b3b 100644
--- a/cmd/internal/launcher/osruntime.go
+++ b/cmd/internal/launcher/osruntime.go
@@ -599,7 +599,8 @@ func (OSRuntime) AgentSessionExists(agent, sid, cwd string) bool {
 	case "agy":
 		return fileExists(AgyConversationPath(home, sid))
 	case "codex":
-		return transcript.Resolve("codex", sid, cwd, home) != ""
+		path := transcript.Resolve("codex", sid, cwd, home)
+		return path != "" && transcript.ReadCodexRootSessionID(path) == sid
 	case "muse":
 		return transcript.Resolve("muse", sid, cwd, home) != ""
 	}
@@ -624,7 +625,7 @@ func (r OSRuntime) LiveAgentSessionID(agent, tag string) string {
 			if !strings.HasPrefix(name, prefix) {
 				continue
 			}
-			if sid := transcript.CodexSessionIDFromPath(name); sid != "" {
+			if sid := transcript.ReadCodexRootSessionID(name); sid != "" {
 				return sid
 			}
 		}
diff --git a/cmd/internal/launcher/osruntime_test.go b/cmd/internal/launcher/osruntime_test.go
index ae28df4..dc72b00 100644
--- a/cmd/internal/launcher/osruntime_test.go
+++ b/cmd/internal/launcher/osruntime_test.go
@@ -1,6 +1,7 @@
 package launcher
 
 import (
+	"fmt"
 	"os"
 	"path/filepath"
 	"reflect"
@@ -176,16 +177,28 @@ func TestOSRuntimeLiveCodexSessionIDUsesAgentPIDDescendantLsof(t *testing.T) {
 	if err := os.WriteFile(filepath.Join(dataDir, "agent-pid-work"), []byte("10\n"), 0o644); err != nil {
 		t.Fatal(err)
 	}
-	sid := "019e8178-79c2-7862-91db-e8fa1be3b162"
-	path := filepath.Join(home, ".codex", "sessions", "2026", "05", "31",
-		"rollout-2026-05-31T21-36-56-"+sid+".jsonl")
+	rootSID := "019e8178-79c2-7862-91db-e8fa1be3b162"
+	subSID := "01a017b6-af00-7c91-a656-0611a3750669"
+	rootPath := filepath.Join(home, ".codex", "sessions", "2026", "05", "31",
+		"rollout-2026-05-31T21-36-56-"+rootSID+".jsonl")
+	subPath := filepath.Join(home, ".codex", "sessions", "2026", "05", "31",
+		"rollout-2026-05-31T22-00-00-"+subSID+".jsonl")
+	if err := os.MkdirAll(filepath.Dir(rootPath), 0o755); err != nil {
+		t.Fatal(err)
+	}
+	if err := os.WriteFile(rootPath, []byte(`{"type":"session_meta","payload":{"id":"`+rootSID+`","parent_thread_id":null,"source":"cli"}}`+"\n"), 0o644); err != nil {
+		t.Fatal(err)
+	}
+	if err := os.WriteFile(subPath, []byte(`{"type":"session_meta","payload":{"id":"`+subSID+`","parent_thread_id":"`+rootSID+`","source":{"subagent":{}}}}`+"\n"), 0o644); err != nil {
+		t.Fatal(err)
+	}
 
 	binDir := t.TempDir()
 	ps := "#!/bin/sh\nprintf ' 10 1\\n 11 10\\n'\n"
 	if err := os.WriteFile(filepath.Join(binDir, "ps"), []byte(ps), 0o755); err != nil {
 		t.Fatal(err)
 	}
-	lsof := "#!/bin/sh\nif [ \"$2\" = \"11\" ]; then printf 'p11\\nn" + path + "\\n'; else printf 'p%s\\n' \"$2\"; fi\n"
+	lsof := "#!/bin/sh\nif [ \"$2\" = \"11\" ]; then printf 'p11\\nn" + subPath + "\\nn" + rootPath + "\\n'; else printf 'p%s\\n' \"$2\"; fi\n"
 	if err := os.WriteFile(filepath.Join(binDir, "lsof"), []byte(lsof), 0o755); err != nil {
 		t.Fatal(err)
 	}
@@ -195,8 +208,8 @@ func TestOSRuntimeLiveCodexSessionIDUsesAgentPIDDescendantLsof(t *testing.T) {
 	t.Setenv("PATH", binDir+string(os.PathListSeparator)+oldPath)
 
 	rt := NewOSRuntime(dataDir, "/pair")
-	if got := rt.LiveAgentSessionID("codex", "work"); got != sid {
-		t.Fatalf("LiveAgentSessionID = %q, want %q", got, sid)
+	if got := rt.LiveAgentSessionID("codex", "work"); got != rootSID {
+		t.Fatalf("LiveAgentSessionID = %q, want root %q", got, rootSID)
 	}
 	if got := rt.LiveAgentSessionID("claude", "work"); got != "" {
 		t.Fatalf("non-codex LiveAgentSessionID = %q, want empty", got)
@@ -324,7 +337,8 @@ func TestOSRuntimeAgentSessionExistsFindsNestedCodexRollout(t *testing.T) {
 	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
 		t.Fatal(err)
 	}
-	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
+	first := fmt.Sprintf(`{"type":"session_meta","payload":{"id":%q,"parent_thread_id":null,"source":"cli"}}`+"\n", sid)
+	if err := os.WriteFile(path, []byte(first), 0o644); err != nil {
 		t.Fatal(err)
 	}
 
diff --git a/cmd/internal/reviewcmd/reviewcmd_test.go b/cmd/internal/reviewcmd/reviewcmd_test.go
index 62051e1..f81ba0c 100644
--- a/cmd/internal/reviewcmd/reviewcmd_test.go
+++ b/cmd/internal/reviewcmd/reviewcmd_test.go
@@ -2,6 +2,8 @@ package reviewcmd
 
 import (
 	"encoding/json"
+	"os"
+	"path/filepath"
 	"testing"
 )
 
@@ -33,14 +35,26 @@ func TestTargetJSON(t *testing.T) {
 	}
 }
 
-func TestSessionFromConfig(t *testing.T) {
-	if got := sessionFromConfig(`{"agent":"codex","session_id":"cfgsid"}`); got != "cfgsid" {
-		t.Fatalf("got %q", got)
+func TestOSRuntimeConfiguredSessionIDRejectsCodexSubagent(t *testing.T) {
+	home := t.TempDir()
+	data := filepath.Join(home, "data")
+	if err := os.MkdirAll(data, 0o755); err != nil {
+		t.Fatal(err)
 	}
-	if got := sessionFromConfig(`{"agent":"codex"}`); got != "" {
-		t.Fatalf("no session_id → empty, got %q", got)
+	sid := "01a017b6-af00-7c91-a656-0611a3750669"
+	parent := "019e8178-79c2-7862-91db-e8fa1be3b162"
+	rollout := filepath.Join(home, ".codex", "sessions", "2026", "08", "18", "rollout-sub-"+sid+".jsonl")
+	if err := os.MkdirAll(filepath.Dir(rollout), 0o755); err != nil {
+		t.Fatal(err)
 	}
-	if got := sessionFromConfig(`not json`); got != "" {
-		t.Fatalf("bad json → empty, got %q", got)
+	if err := os.WriteFile(rollout, []byte(`{"type":"session_meta","payload":{"id":"`+sid+`","parent_thread_id":"`+parent+`","source":{"subagent":{}}}}`+"\n"), 0o644); err != nil {
+		t.Fatal(err)
+	}
+	if err := os.WriteFile(filepath.Join(data, "config-t-codex.json"), []byte(`{"session_id":"`+sid+`"}`), 0o644); err != nil {
+		t.Fatal(err)
+	}
+	t.Setenv("HOME", home)
+	if got := NewOSRuntime().ConfiguredSessionID(data, "t", "codex"); got != "" {
+		t.Fatalf("ConfiguredSessionID = %q, want empty", got)
 	}
 }
diff --git a/cmd/internal/reviewcmd/run.go b/cmd/internal/reviewcmd/run.go
index 52bde2a..354ab8f 100644
--- a/cmd/internal/reviewcmd/run.go
+++ b/cmd/internal/reviewcmd/run.go
@@ -40,6 +40,8 @@ type Runtime interface {
 	SpawnReviewPane(cwd, lua, absFile, nvimPidFile string) error
 	// ResolveCodexSessionID walks the codex agent's process tree (codexsid).
 	ResolveCodexSessionID(dataDir, tag string) string
+	// ConfiguredSessionID returns a semantically validated persisted identity.
+	ConfiguredSessionID(dataDir, tag, agent string) string
 }
 
 // ── target ────────────────────────────────────────────────────────────────
@@ -105,15 +107,14 @@ func RunDefinition(opts DefinitionOptions, rt Runtime, stdout, stderr io.Writer)
 }
 
 // resolveTargetSession implements the target seam's session priority:
-// PAIR_SESSION_ID → config session_id → (codex only) the live-rollout lsof walk.
+// PAIR_SESSION_ID → validated config session_id → (codex only) validated live
+// root-rollout discovery.
 func resolveTargetSession(rt Runtime, dataDir, tag, agent, envSID string) string {
 	if envSID != "" {
 		return envSID
 	}
-	if cfg, err := rt.ReadFile(filepath.Join(dataDir, "config-"+tag+"-"+agent+".json")); err == nil {
-		if sid := sessionFromConfig(cfg); sid != "" {
-			return sid
-		}
+	if sid := rt.ConfiguredSessionID(dataDir, tag, agent); sid != "" {
+		return sid
 	}
 	if agent == "codex" {
 		return rt.ResolveCodexSessionID(dataDir, tag)
@@ -329,13 +330,3 @@ func firstLine(s string) string {
 	}
 	return strings.TrimSpace(s)
 }
-
-func sessionFromConfig(cfg string) string {
-	var c struct {
-		SessionID string `json:"session_id"`
-	}
-	if json.Unmarshal([]byte(cfg), &c) != nil {
-		return ""
-	}
-	return c.SessionID
-}
diff --git a/cmd/internal/reviewcmd/run_test.go b/cmd/internal/reviewcmd/run_test.go
index 42b7966..9bfb3d6 100644
--- a/cmd/internal/reviewcmd/run_test.go
+++ b/cmd/internal/reviewcmd/run_test.go
@@ -11,19 +11,20 @@ import (
 type spawnCall struct{ cwd, lua, absFile, nvimPid string }
 
 type fakeRuntime struct {
-	files    map[string]string
-	wrote    map[string]string
-	removed  []string
-	sizes    map[string]int64
-	alive    map[string]bool
-	killed   []string
-	gitFn    func(dir string, args []string) (string, error)
-	gitCalls [][]string
-	classify string
-	classErr error
-	spawn    *spawnCall
-	codexSID string
-	writeErr error
+	files     map[string]string
+	wrote     map[string]string
+	removed   []string
+	sizes     map[string]int64
+	alive     map[string]bool
+	killed    []string
+	gitFn     func(dir string, args []string) (string, error)
+	gitCalls  [][]string
+	classify  string
+	classErr  error
+	spawn     *spawnCall
+	configSID string
+	codexSID  string
+	writeErr  error
 }
 
 func newFake() *fakeRuntime {
@@ -80,6 +81,9 @@ func (f *fakeRuntime) SpawnReviewPane(cwd, lua, absFile, nvimPid string) error {
 	return nil
 }
 func (f *fakeRuntime) ResolveCodexSessionID(dataDir, tag string) string { return f.codexSID }
+func (f *fakeRuntime) ConfiguredSessionID(dataDir, tag, agent string) string {
+	return f.configSID
+}
 
 func targetOf(t *testing.T, rt *fakeRuntime, tag string) targetDoc {
 	t.Helper()
@@ -99,7 +103,7 @@ func TestRunTargetSessionPriority(t *testing.T) {
 	}
 	// config fallback
 	rt = newFake()
-	rt.files["/dd/config-t-codex.json"] = `{"session_id":"cfgsid"}`
+	rt.configSID = "cfgsid"
 	RunTarget(TargetOptions{File: "/r/doc.md", Status: "proposed", Tag: "t", Agent: "codex", DataDir: "/dd"}, rt, &bytes.Buffer{}, &bytes.Buffer{})
 	if d := targetOf(t, rt, "t"); d.Session != "cfgsid" {
 		t.Fatalf("config: %+v", d)
@@ -111,6 +115,14 @@ func TestRunTargetSessionPriority(t *testing.T) {
 	if d := targetOf(t, rt, "t"); d.Session != "walksid" {
 		t.Fatalf("codex: %+v", d)
 	}
+	// A rejected configured id falls through to the validated live root.
+	rt = newFake()
+	rt.configSID = ""
+	rt.codexSID = "rootsid"
+	RunTarget(TargetOptions{File: "/r/doc.md", Status: "ready", Tag: "t", Agent: "codex", DataDir: "/dd"}, rt, &bytes.Buffer{}, &bytes.Buffer{})
+	if d := targetOf(t, rt, "t"); d.Session != "rootsid" {
+		t.Fatalf("polluted config fallback: %+v", d)
+	}
 }
 
 func TestRunTargetInvalidStatus(t *testing.T) {
diff --git a/cmd/internal/reviewcmd/runtime.go b/cmd/internal/reviewcmd/runtime.go
index 867f054..ad0403d 100644
--- a/cmd/internal/reviewcmd/runtime.go
+++ b/cmd/internal/reviewcmd/runtime.go
@@ -10,6 +10,7 @@ import (
 	"github.com/xianxu/pair/cmd/internal/codexsid"
 	"github.com/xianxu/pair/cmd/internal/osfs"
 	"github.com/xianxu/pair/cmd/internal/procutil"
+	"github.com/xianxu/pair/cmd/internal/transcript"
 )
 
 // OSRuntime implements Runtime with real git/nvim/zellij/fs calls; the fs
@@ -99,3 +100,8 @@ func (OSRuntime) SpawnReviewPane(cwd, lua, absFile, nvimPidFile string) error {
 func (OSRuntime) ResolveCodexSessionID(dataDir, tag string) string {
 	return codexsid.ResolveSessionID(dataDir, tag)
 }
+
+func (OSRuntime) ConfiguredSessionID(dataDir, tag, agent string) string {
+	home, _ := os.UserHomeDir()
+	return transcript.SessionID(dataDir, tag, agent, home)
+}
diff --git a/cmd/internal/sessionwatch/run.go b/cmd/internal/sessionwatch/run.go
index f7e1489..87ef261 100644
--- a/cmd/internal/sessionwatch/run.go
+++ b/cmd/internal/sessionwatch/run.go
@@ -7,6 +7,7 @@ import (
 	"time"
 
 	"github.com/xianxu/pair/cmd/internal/adapt"
+	"github.com/xianxu/pair/cmd/internal/transcript"
 )
 
 // isMuseSubagentPath reports whether p is inside a Muse subagent directory.
@@ -36,6 +37,7 @@ type Runtime interface {
 	Now() time.Time
 	Sleep(time.Duration)
 	ReadFile(path string) ([]byte, error)
+	ReadFirstLine(path string) ([]byte, error)
 	ModTime(path string) (time.Time, error)
 	BirthTime(path string) (time.Time, error)
 	ListFiles(root string) ([]string, error)
@@ -192,7 +194,7 @@ func discover(spec AgentSpec, rootPID string, agentStart time.Time, legacyExisti
 				if spec.Agent == "muse" && isMuseSubagentPath(path) {
 					continue
 				}
-				result := spec.Match(path)
+				result := authorizeCandidate(spec, spec.Match(path), rt)
 				if result.ID != "" {
 					return result
 				}
@@ -219,7 +221,7 @@ func discover(spec AgentSpec, rootPID string, agentStart time.Time, legacyExisti
 		if spec.Agent == "muse" && isMuseSubagentPath(file) {
 			continue
 		}
-		result := spec.Match(file)
+		result := authorizeCandidate(spec, spec.Match(file), rt)
 		if result.ID != "" {
 			return result
 		}
@@ -246,7 +248,7 @@ func discoverByBirth(spec AgentSpec, agentStart time.Time, rt Runtime) SessionID
 		if err != nil || birth.Before(agentStart) {
 			continue
 		}
-		result := spec.Match(file)
+		result := authorizeCandidate(spec, spec.Match(file), rt)
 		if !result.Matched {
 			continue
 		}
@@ -280,3 +282,14 @@ func discoverByBirth(spec AgentSpec, agentStart time.Time, rt Runtime) SessionID
 	}
 	return SessionID{}
 }
+
+func authorizeCandidate(spec AgentSpec, result SessionID, rt Runtime) SessionID {
+	if spec.Agent != "codex" || result.ID == "" {
+		return result
+	}
+	firstEvent, err := rt.ReadFirstLine(result.Path)
+	if err != nil || transcript.CodexRootSessionID(result.Path, firstEvent) != result.ID {
+		return SessionID{}
+	}
+	return result
+}
diff --git a/cmd/internal/sessionwatch/run_test.go b/cmd/internal/sessionwatch/run_test.go
index c95daf1..24d54b9 100644
--- a/cmd/internal/sessionwatch/run_test.go
+++ b/cmd/internal/sessionwatch/run_test.go
@@ -23,6 +23,7 @@ func TestRunUsesFreshPidfileAndWritesConfig(t *testing.T) {
 	rt.alive["1234"] = true
 	rt.descendants["1234"] = []string{"1234", "5678"}
 	rt.lsof["5678"] = []string{sessionFile}
+	rt.files[sessionFile] = fakeFile{content: rootSessionMeta(sid), birth: rt.now}
 
 	err := Run(Options{
 		Agent:   "codex",
@@ -66,6 +67,7 @@ func TestRunUsesRepoIdentityForLedgerWhenCwdIsSubdir(t *testing.T) {
 	rt.alive["1234"] = true
 	rt.descendants["1234"] = []string{"1234"}
 	rt.lsof["1234"] = []string{sessionFile}
+	rt.files[sessionFile] = fakeFile{content: rootSessionMeta(sid), birth: rt.now}
 
 	err := Run(Options{
 		Agent:    "codex",
@@ -102,6 +104,7 @@ func TestRunDoesNotWriteConfigWhenLedgerAppendFails(t *testing.T) {
 	rt.alive["1234"] = true
 	rt.descendants["1234"] = []string{"1234"}
 	rt.lsof["1234"] = []string{sessionFile}
+	rt.files[sessionFile] = fakeFile{content: rootSessionMeta(sid), birth: rt.now}
 	rt.writeErr[filepath.Join(data, "ledger-test.jsonl")] = errors.New("ledger write failed")
 
 	err := Run(Options{
@@ -132,6 +135,7 @@ func TestRunTreatsSameSecondPidfileAsFresh(t *testing.T) {
 	rt.alive["1234"] = true
 	rt.descendants["1234"] = []string{"1234"}
 	rt.lsof["1234"] = []string{sessionFile}
+	rt.files[sessionFile] = fakeFile{content: rootSessionMeta(sid), birth: rt.now}
 
 	err := Run(Options{
 		Agent:   "codex",
@@ -227,6 +231,7 @@ func TestRunContinuesPastLsofNearMissToValidCandidate(t *testing.T) {
 	rt.descendants["3500"] = []string{"3500", "3501"}
 	rt.lsof["3500"] = []string{bad}
 	rt.lsof["3501"] = []string{good}
+	rt.files[good] = fakeFile{content: rootSessionMeta(sid), birth: rt.now}
 
 	err := Run(Options{
 		Agent:   "codex",
@@ -262,7 +267,7 @@ func TestRunContinuesPastLegacyNearMissToValidCandidate(t *testing.T) {
 		sleeps++
 		if sleeps == 2 {
 			rt.files[bad] = fakeFile{mod: time.Unix(360, 0)}
-			rt.files[good] = fakeFile{mod: time.Unix(360, 0)}
+			rt.files[good] = fakeFile{content: rootSessionMeta(sid), mod: time.Unix(360, 0)}
 		}
 	}
 
@@ -285,6 +290,73 @@ func TestRunContinuesPastLegacyNearMissToValidCandidate(t *testing.T) {
 	}
 }
 
+func TestRunCodexLsofSkipsSubagentForRoot(t *testing.T) {
+	home := "/tmp/home"
+	data := "/tmp/data"
+	rootSID := "019e8178-79c2-7862-91db-e8fa1be3b162"
+	subSID := "01a017b6-af00-7c91-a656-0611a3750669"
+	rootPath := home + "/.codex/sessions/2026/08/18/rollout-root-" + rootSID + ".jsonl"
+	subPath := home + "/.codex/sessions/2026/08/18/rollout-sub-" + subSID + ".jsonl"
+	rt := newFakeRuntime(time.Unix(370, 0))
+	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("3700\n"), mod: rt.now}
+	rt.files[rootPath] = fakeFile{content: []byte(`{"type":"session_meta","payload":{"id":"` + rootSID + `","parent_thread_id":null,"source":"cli"}}` + "\n"), birth: rt.now}
+	rt.files[subPath] = fakeFile{content: []byte(`{"type":"session_meta","payload":{"id":"` + subSID + `","parent_thread_id":"` + rootSID + `","source":{"subagent":{}}}}` + "\n"), birth: rt.now}
+	rt.alive["3700"] = true
+	rt.descendants["3700"] = []string{"3700"}
+	rt.lsof["3700"] = []string{subPath, rootPath}
+
+	if err := Run(Options{Agent: "codex", Tag: "tag", Cwd: "/repo", Home: home, DataDir: data, PIDWait: time.Second, Timeout: time.Second, Poll: 100 * time.Millisecond}, rt); err != nil {
+		t.Fatal(err)
+	}
+	got := string(rt.writes[filepath.Join(data, "config-tag-codex.json")])
+	if !strings.Contains(got, rootSID) || strings.Contains(got, subSID) {
+		t.Fatalf("config = %s, want root %s", got, rootSID)
+	}
+}
+
+func TestRunCodexBirthFallbackSkipsNewerSubagent(t *testing.T) {
+	home := "/tmp/home"
+	data := "/tmp/data"
+	rootSID := "019e8178-79c2-7862-91db-e8fa1be3b162"
+	subSID := "01a017b6-af00-7c91-a656-0611a3750669"
+	rootPath := home + "/.codex/sessions/2026/08/18/rollout-root-" + rootSID + ".jsonl"
+	subPath := home + "/.codex/sessions/2026/08/18/rollout-sub-" + subSID + ".jsonl"
+	rt := newFakeRuntime(time.Unix(380, 0))
+	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("3800\n"), mod: rt.now}
+	rt.files[rootPath] = fakeFile{content: []byte(`{"type":"session_meta","payload":{"id":"` + rootSID + `","parent_thread_id":null,"source":"exec"}}` + "\n"), birth: rt.now.Add(time.Second)}
+	rt.files[subPath] = fakeFile{content: []byte(`{"type":"session_meta","payload":{"id":"` + subSID + `","parent_thread_id":"` + rootSID + `","source":{"subagent":{}}}}` + "\n"), birth: rt.now.Add(2 * time.Second)}
+	rt.alive["3800"] = true
+
+	if err := Run(Options{Agent: "codex", Tag: "tag", Cwd: "/repo", Home: home, DataDir: data, PIDWait: time.Second, Timeout: time.Second, Poll: 100 * time.Millisecond}, rt); err != nil {
+		t.Fatal(err)
+	}
+	got := string(rt.writes[filepath.Join(data, "config-tag-codex.json")])
+	if !strings.Contains(got, rootSID) || strings.Contains(got, subSID) {
+		t.Fatalf("config = %s, want root %s", got, rootSID)
+	}
+}
+
+func TestRunCodexSubagentOnlyWritesNoConfig(t *testing.T) {
+	home := "/tmp/home"
+	data := "/tmp/data"
+	subSID := "01a017b6-af00-7c91-a656-0611a3750669"
+	parent := "019e8178-79c2-7862-91db-e8fa1be3b162"
+	subPath := home + "/.codex/sessions/2026/08/18/rollout-sub-" + subSID + ".jsonl"
+	rt := newFakeRuntime(time.Unix(390, 0))
+	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("3900\n"), mod: rt.now}
+	rt.files[subPath] = fakeFile{content: []byte(`{"type":"session_meta","payload":{"id":"` + subSID + `","parent_thread_id":"` + parent + `","source":{"subagent":{}}}}` + "\n"), birth: rt.now}
+	rt.alive["3900"] = true
+	rt.lsof["3900"] = []string{subPath}
+	rt.onSleep = func(time.Duration) { rt.alive["3900"] = false }
+
+	if err := Run(Options{Agent: "codex", Tag: "tag", Cwd: "/repo", Home: home, DataDir: data, PIDWait: time.Second, Timeout: time.Second, Poll: 100 * time.Millisecond}, rt); err != nil {
+		t.Fatal(err)
+	}
+	if got := rt.writes[filepath.Join(data, "config-tag-codex.json")]; got != nil {
+		t.Fatalf("subagent-only config = %s, want none", got)
+	}
+}
+
 func TestRunLogsFailOnTimeout(t *testing.T) {
 	rt := newFakeRuntime(time.Unix(400, 0))
 	err := Run(Options{
@@ -311,6 +383,10 @@ type fakeFile struct {
 	birth   time.Time
 }
 
+func rootSessionMeta(sid string) []byte {
+	return []byte(`{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":null,"source":"cli"}}` + "\n")
+}
+
 type fakeLog struct {
 	outcome adapt.Outcome
 	detail  string
@@ -357,6 +433,17 @@ func (f *fakeRuntime) ReadFile(path string) ([]byte, error) {
 	return file.content, nil
 }
 
+func (f *fakeRuntime) ReadFirstLine(path string) ([]byte, error) {
+	data, err := f.ReadFile(path)
+	if err != nil {
+		return nil, err
+	}
+	if i := strings.IndexByte(string(data), '\n'); i >= 0 {
+		return data[:i+1], nil
+	}
+	return nil, errors.New("unterminated first line")
+}
+
 func (f *fakeRuntime) ModTime(path string) (time.Time, error) {
 	file, ok := f.files[path]
 	if !ok {
diff --git a/cmd/internal/sessionwatch/runtime.go b/cmd/internal/sessionwatch/runtime.go
index 60e8aa9..ee557ef 100644
--- a/cmd/internal/sessionwatch/runtime.go
+++ b/cmd/internal/sessionwatch/runtime.go
@@ -10,6 +10,7 @@ import (
 
 	"github.com/xianxu/pair/cmd/internal/adapt"
 	"github.com/xianxu/pair/cmd/internal/procutil"
+	"github.com/xianxu/pair/cmd/internal/transcript"
 )
 
 // OSRuntime implements Runtime with real process and filesystem calls.
@@ -28,6 +29,10 @@ func (OSRuntime) Sleep(d time.Duration) {
 
 func (OSRuntime) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }
 
+func (OSRuntime) ReadFirstLine(path string) ([]byte, error) {
+	return transcript.ReadFirstEvent(path)
+}
+
 func (OSRuntime) ModTime(path string) (time.Time, error) {
 	info, err := os.Stat(path)
 	if err != nil {
@@ -63,44 +68,11 @@ func (OSRuntime) ListFiles(root string) ([]string, error) {
 }
 
 func (OSRuntime) Descendants(root string) ([]string, error) {
-	out, err := exec.Command("ps", "-axo", "pid=,ppid=").Output()
-	if err != nil {
-		return []string{root}, nil
-	}
-	children := map[string][]string{}
-	for _, line := range strings.Split(string(out), "\n") {
-		fields := strings.Fields(line)
-		if len(fields) != 2 {
-			continue
-		}
-		children[fields[1]] = append(children[fields[1]], fields[0])
-	}
-	queue := []string{root}
-	seen := map[string]bool{root: true}
-	for i := 0; i < len(queue); i++ {
-		for _, child := range children[queue[i]] {
-			if child == "" || seen[child] {
-				continue
-			}
-			seen[child] = true
-			queue = append(queue, child)
-		}
-	}
-	return queue, nil
+	return procutil.DescendantPIDs(root, procutil.ProcessChildren()), nil
 }
 
 func (OSRuntime) LsofPaths(pid string) ([]string, error) {
-	out, err := exec.Command("lsof", "-p", pid, "-Fn").Output()
-	if err != nil {
-		return nil, nil
-	}
-	var paths []string
-	for _, line := range strings.Split(string(out), "\n") {
-		if strings.HasPrefix(line, "n") {
-			paths = append(paths, strings.TrimPrefix(line, "n"))
-		}
-	}
-	return paths, nil
+	return procutil.LsofNames(pid), nil
 }
 
 func (OSRuntime) ProcessAlive(pid string) bool {
diff --git a/cmd/internal/slugcmd/slugcmd.go b/cmd/internal/slugcmd/slugcmd.go
index 6bb7029..a6cea17 100644
--- a/cmd/internal/slugcmd/slugcmd.go
+++ b/cmd/internal/slugcmd/slugcmd.go
@@ -91,7 +91,7 @@ func resolveLiveCodexTranscript(dataDir, tag, home string) string {
 	prefix := filepath.Join(home, ".codex", "sessions") + string(os.PathSeparator)
 	for _, pid := range procutil.DescendantPIDs(root, procutil.ProcessChildren()) {
 		for _, name := range procutil.LsofNames(pid) {
-			if strings.HasPrefix(name, prefix) && transcriptpkg.CodexSessionIDFromPath(name) != "" {
+			if strings.HasPrefix(name, prefix) && transcriptpkg.ReadCodexRootSessionID(name) != "" {
 				return name
 			}
 		}
@@ -128,7 +128,7 @@ func Run() int {
 
 	transcript := os.Getenv("PAIR_SLUG_TRANSCRIPT")
 	if transcript == "" {
-		sid := transcriptpkg.SessionID(dataDir, tag, agent)
+		sid := transcriptpkg.SessionID(dataDir, tag, agent, home)
 		if sid != "" {
 			transcript = transcriptpkg.Resolve(agent, sid, cwd, home)
 		}
diff --git a/cmd/internal/slugcmd/slugcmd_test.go b/cmd/internal/slugcmd/slugcmd_test.go
index 7b19cfc..b43b208 100644
--- a/cmd/internal/slugcmd/slugcmd_test.go
+++ b/cmd/internal/slugcmd/slugcmd_test.go
@@ -22,14 +22,27 @@ func TestResolveLiveCodexTranscriptUsesDescendantLsof(t *testing.T) {
 		t.Fatal(err)
 	}
 
-	path := filepath.Join(home, ".codex", "sessions", "2026", "05", "31",
-		"rollout-2026-05-31T21-36-56-019e8178-79c2-7862-91db-e8fa1be3b162.jsonl")
+	rootSID := "019e8178-79c2-7862-91db-e8fa1be3b162"
+	subSID := "01a017b6-af00-7c91-a656-0611a3750669"
+	rootPath := filepath.Join(home, ".codex", "sessions", "2026", "05", "31",
+		"rollout-2026-05-31T21-36-56-"+rootSID+".jsonl")
+	subPath := filepath.Join(home, ".codex", "sessions", "2026", "05", "31",
+		"rollout-2026-05-31T22-00-00-"+subSID+".jsonl")
+	if err := os.MkdirAll(filepath.Dir(rootPath), 0o755); err != nil {
+		t.Fatal(err)
+	}
+	if err := os.WriteFile(rootPath, []byte(`{"type":"session_meta","payload":{"id":"`+rootSID+`","parent_thread_id":null,"source":"cli"}}`+"\n"), 0o644); err != nil {
+		t.Fatal(err)
+	}
+	if err := os.WriteFile(subPath, []byte(`{"type":"session_meta","payload":{"id":"`+subSID+`","parent_thread_id":"`+rootSID+`","source":{"subagent":{}}}}`+"\n"), 0o644); err != nil {
+		t.Fatal(err)
+	}
 	binDir := t.TempDir()
 	ps := "#!/bin/sh\nprintf ' 10 1\\n 11 10\\n'\n"
 	if err := os.WriteFile(filepath.Join(binDir, "ps"), []byte(ps), 0o755); err != nil {
 		t.Fatal(err)
 	}
-	lsof := "#!/bin/sh\nif [ \"$2\" = \"11\" ]; then printf 'p11\\nn" + path + "\\n'; else printf 'p%s\\n' \"$2\"; fi\n"
+	lsof := "#!/bin/sh\nif [ \"$2\" = \"11\" ]; then printf 'p11\\nn" + subPath + "\\nn" + rootPath + "\\n'; else printf 'p%s\\n' \"$2\"; fi\n"
 	if err := os.WriteFile(filepath.Join(binDir, "lsof"), []byte(lsof), 0o755); err != nil {
 		t.Fatal(err)
 	}
@@ -37,7 +50,7 @@ func TestResolveLiveCodexTranscriptUsesDescendantLsof(t *testing.T) {
 	oldPath := os.Getenv("PATH")
 	t.Setenv("PATH", binDir+string(os.PathListSeparator)+oldPath)
 	got := resolveLiveCodexTranscript(dataDir, "testtag", home)
-	if got != path {
-		t.Fatalf("resolveLiveCodexTranscript = %q, want %q", got, path)
+	if got != rootPath {
+		t.Fatalf("resolveLiveCodexTranscript = %q, want root %q", got, rootPath)
 	}
 }
diff --git a/cmd/internal/transcript/transcript.go b/cmd/internal/transcript/transcript.go
index 959c9d8..e1d398e 100644
--- a/cmd/internal/transcript/transcript.go
+++ b/cmd/internal/transcript/transcript.go
@@ -4,13 +4,17 @@
 package transcript
 
 import (
+	"bufio"
 	"encoding/json"
+	"io"
 	"os"
 	"path/filepath"
 	"regexp"
 	"strings"
 )
 
+const codexSessionMetaLineLimit = 1 << 20
+
 // ClaudePathEncoder mirrors nvim's `cwd:gsub('[./]', '-')` for the
 // ~/.claude/projects/<encoded-cwd>/ directory name.
 var ClaudePathEncoder = strings.NewReplacer(".", "-", "/", "-")
@@ -27,9 +31,63 @@ func CodexSessionIDFromPath(path string) string {
 	return m[2]
 }
 
+// CodexRootSessionID authorizes a root Codex rollout from its path and first
+// JSONL event. Filename UUIDs identify candidates; session_meta establishes
+// whether the candidate is the operator's resumable root session.
+func CodexRootSessionID(path string, firstEvent []byte) string {
+	sid := CodexSessionIDFromPath(path)
+	if sid == "" {
+		return ""
+	}
+	var event struct {
+		Type    string `json:"type"`
+		Payload struct {
+			ID             string          `json:"id"`
+			ParentThreadID *string         `json:"parent_thread_id"`
+			Source         json.RawMessage `json:"source"`
+		} `json:"payload"`
+	}
+	if err := json.Unmarshal(firstEvent, &event); err != nil || event.Type != "session_meta" || event.Payload.ID != sid || event.Payload.ParentThreadID != nil {
+		return ""
+	}
+	var source string
+	if err := json.Unmarshal(event.Payload.Source, &source); err != nil || (source != "cli" && source != "exec") {
+		return ""
+	}
+	return sid
+}
+
+// ReadCodexRootSessionID reads one bounded, newline-terminated JSONL event and
+// delegates the semantic decision to CodexRootSessionID. It fails closed when
+// the rollout is incomplete, oversized, unreadable, or not a root session.
+func ReadCodexRootSessionID(path string) string {
+	line, err := ReadFirstEvent(path)
+	if err != nil {
+		return ""
+	}
+	return CodexRootSessionID(path, line)
+}
+
+// ReadFirstEvent returns one bounded, newline-terminated JSONL event.
+func ReadFirstEvent(path string) ([]byte, error) {
+	f, err := os.Open(path)
+	if err != nil {
+		return nil, err
+	}
+	defer f.Close()
+	line, err := bufio.NewReader(io.LimitReader(f, codexSessionMetaLineLimit+1)).ReadBytes('\n')
+	if err != nil || len(line) > codexSessionMetaLineLimit {
+		if err == nil {
+			err = io.ErrShortBuffer
+		}
+		return nil, err
+	}
+	return line, nil
+}
+
 // SessionID reads the session id pair recorded for (tag, agent) in
 // config-<tag>-<agent>.json (written by bin/pair / pair-session-watch).
-func SessionID(dataDir, tag, agent string) string {
+func SessionID(dataDir, tag, agent, home string) string {
 	b, err := os.ReadFile(filepath.Join(dataDir, "config-"+tag+"-"+agent+".json"))
 	if err != nil {
 		return ""
@@ -40,6 +98,12 @@ func SessionID(dataDir, tag, agent string) string {
 	if json.Unmarshal(b, &c) != nil {
 		return ""
 	}
+	if agent == "codex" {
+		path := Resolve(agent, c.SessionID, "", home)
+		if path == "" || ReadCodexRootSessionID(path) != c.SessionID {
+			return ""
+		}
+	}
 	return c.SessionID
 }
 
diff --git a/cmd/internal/transcript/transcript_test.go b/cmd/internal/transcript/transcript_test.go
index c567742..69af958 100644
--- a/cmd/internal/transcript/transcript_test.go
+++ b/cmd/internal/transcript/transcript_test.go
@@ -3,9 +3,12 @@ package transcript
 import (
 	"os"
 	"path/filepath"
+	"strings"
 	"testing"
 )
 
+const testCodexSessionMetaLineLimit = 1 << 20
+
 func TestResolveClaudeEncodesCwd(t *testing.T) {
 	got := Resolve("claude", "abc", "/Users/x/work.dir", "/home")
 	want := filepath.Join("/home", ".claude", "projects", "-Users-x-work-dir", "abc.jsonl")
@@ -33,6 +36,139 @@ func TestCodexSessionIDFromPath(t *testing.T) {
 	}
 }
 
+func TestCodexRootSessionID(t *testing.T) {
+	sid := "01a00e37-16c4-7100-89fc-42ce26158f71"
+	path := filepath.Join("/home/u", ".codex", "sessions", "2026", "08", "16", "rollout-2026-08-16T22-34-46-"+sid+".jsonl")
+	tests := []struct {
+		name  string
+		path  string
+		event string
+		want  string
+	}{
+		{name: "cli root", path: path, event: `{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":null,"source":"cli"}}`, want: sid},
+		{name: "exec root with absent parent", path: path, event: `{"type":"session_meta","payload":{"id":"` + sid + `","source":"exec"}}`, want: sid},
+		{name: "subagent", path: path, event: `{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":"parent","source":{"subagent":{"thread_spawn":{"depth":1}}}}}`},
+		{name: "non-null parent", path: path, event: `{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":"parent","source":"cli"}}`},
+		{name: "unknown string source", path: path, event: `{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":null,"source":"future"}}`},
+		{name: "unknown object source", path: path, event: `{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":null,"source":{"other":{}}}}`},
+		{name: "mismatched id", path: path, event: `{"type":"session_meta","payload":{"id":"11a00e37-16c4-7100-89fc-42ce26158f71","parent_thread_id":null,"source":"cli"}}`},
+		{name: "wrong event type", path: path, event: `{"type":"event_msg","payload":{"id":"` + sid + `","source":"cli"}}`},
+		{name: "missing id", path: path, event: `{"type":"session_meta","payload":{"parent_thread_id":null,"source":"cli"}}`},
+		{name: "malformed json", path: path, event: `{"type":`},
+		{name: "malformed filename", path: "/tmp/not-codex.jsonl", event: `{"type":"session_meta","payload":{"id":"` + sid + `","source":"cli"}}`},
+	}
+	for _, tt := range tests {
+		t.Run(tt.name, func(t *testing.T) {
+			if got := CodexRootSessionID(tt.path, []byte(tt.event)); got != tt.want {
+				t.Fatalf("CodexRootSessionID = %q, want %q", got, tt.want)
+			}
+		})
+	}
+}
+
+func TestReadCodexRootSessionIDBoundaries(t *testing.T) {
+	home := t.TempDir()
+	sid := "01a00e37-16c4-7100-89fc-42ce26158f71"
+	path := filepath.Join(home, ".codex", "sessions", "2026", "08", "16", "rollout-2026-08-16T22-34-46-"+sid+".jsonl")
+	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
+		t.Fatal(err)
+	}
+	root := `{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":null,"source":"cli"}}`
+	subagent := `{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":"parent","source":{"subagent":{}}}}`
+
+	write := func(body string) {
+		t.Helper()
+		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
+			t.Fatal(err)
+		}
+	}
+
+	write(root + "\n")
+	if got := ReadCodexRootSessionID(path); got != sid {
+		t.Fatalf("valid root = %q, want %q", got, sid)
+	}
+	write(subagent + "\n")
+	if got := ReadCodexRootSessionID(path); got != "" {
+		t.Fatalf("subagent = %q, want empty", got)
+	}
+	write("{}\n" + root + "\n")
+	if got := ReadCodexRootSessionID(path); got != "" {
+		t.Fatalf("later metadata = %q, want empty", got)
+	}
+	write(root)
+	if got := ReadCodexRootSessionID(path); got != "" {
+		t.Fatalf("unterminated first line = %q, want empty", got)
+	}
+
+	prefix := `{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":null,"source":"cli","padding":"`
+	suffix := `"}}` + "\n"
+	lineOfLength := func(n int) string {
+		t.Helper()
+		padding := n - len(prefix) - len(suffix)
+		if padding < 0 {
+			t.Fatalf("test line length %d too small", n)
+		}
+		return prefix + strings.Repeat("x", padding) + suffix
+	}
+	write(lineOfLength(testCodexSessionMetaLineLimit))
+	if got := ReadCodexRootSessionID(path); got != sid {
+		t.Fatalf("exact-limit root = %q, want %q", got, sid)
+	}
+	write(lineOfLength(testCodexSessionMetaLineLimit + 1))
+	if got := ReadCodexRootSessionID(path); got != "" {
+		t.Fatalf("over-limit root = %q, want empty", got)
+	}
+
+	if got := ReadCodexRootSessionID(filepath.Join(home, "missing.jsonl")); got != "" {
+		t.Fatalf("missing file = %q, want empty", got)
+	}
+	if got := ReadCodexRootSessionID(home); got != "" {
+		t.Fatalf("directory read = %q, want empty", got)
+	}
+}
+
+func TestSessionIDValidatesCodexRootMetadata(t *testing.T) {
+	home := t.TempDir()
+	data := filepath.Join(home, "data")
+	if err := os.MkdirAll(data, 0o755); err != nil {
+		t.Fatal(err)
+	}
+	rootSID := "019e8178-79c2-7862-91db-e8fa1be3b162"
+	subSID := "01a017b6-af00-7c91-a656-0611a3750669"
+	dir := filepath.Join(home, ".codex", "sessions", "2026", "05", "31")
+	if err := os.MkdirAll(dir, 0o755); err != nil {
+		t.Fatal(err)
+	}
+	rootPath := filepath.Join(dir, "rollout-root-"+rootSID+".jsonl")
+	subPath := filepath.Join(dir, "rollout-sub-"+subSID+".jsonl")
+	if err := os.WriteFile(rootPath, []byte(`{"type":"session_meta","payload":{"id":"`+rootSID+`","parent_thread_id":null,"source":"cli"}}`+"\n"), 0o644); err != nil {
+		t.Fatal(err)
+	}
+	if err := os.WriteFile(subPath, []byte(`{"type":"session_meta","payload":{"id":"`+subSID+`","parent_thread_id":"`+rootSID+`","source":{"subagent":{}}}}`+"\n"), 0o644); err != nil {
+		t.Fatal(err)
+	}
+	config := filepath.Join(data, "config-work-codex.json")
+	if err := os.WriteFile(config, []byte(`{"session_id":"`+rootSID+`"}`), 0o644); err != nil {
+		t.Fatal(err)
+	}
+	if got := SessionID(data, "work", "codex", home); got != rootSID {
+		t.Fatalf("root config = %q, want %q", got, rootSID)
+	}
+	if err := os.WriteFile(config, []byte(`{"session_id":"`+subSID+`"}`), 0o644); err != nil {
+		t.Fatal(err)
+	}
+	if got := SessionID(data, "work", "codex", home); got != "" {
+		t.Fatalf("subagent config = %q, want empty", got)
+	}
+
+	if err := os.WriteFile(filepath.Join(data, "config-work-claude.json"), []byte(`{"session_id":"claude-id"}`), 0o644); err != nil {
+		t.Fatal(err)
+	}
+	if got := SessionID(data, "work", "claude", home); got != "claude-id" {
+		t.Fatalf("claude config = %q, want claude-id", got)
+	}
+}
+
 func TestResolveMuseIgnoresSubagent(t *testing.T) {
 	home := t.TempDir()
 	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
diff --git a/nvim/init.lua b/nvim/init.lua
index a6ee0c2..608f455 100644
--- a/nvim/init.lua
+++ b/nvim/init.lua
@@ -839,48 +839,6 @@ do
     return nil
   end
 
-  local function descendant_pids(root)
-    local out = vim.fn.systemlist({ 'ps', '-axo', 'pid=,ppid=' })
-    local children = {}
-    for _, line in ipairs(out) do
-      local pid, ppid = line:match('^%s*(%d+)%s+(%d+)%s*$')
-      if pid and ppid then
-        children[ppid] = children[ppid] or {}
-        table.insert(children[ppid], pid)
-      end
-    end
-    local pids, queue, seen = {}, { root }, { [root] = true }
-    local i = 1
-    while i <= #queue do
-      local pid = queue[i]; i = i + 1
-      table.insert(pids, pid)
-      for _, child in ipairs(children[pid] or {}) do
-        if not seen[child] then
-          seen[child] = true
-          table.insert(queue, child)
-        end
-      end
-    end
-    return pids
-  end
-
-  local function live_codex_session_id(data_dir, tag)
-    local pf = io.open(data_dir .. '/agent-pid-' .. tag, 'r')
-    if not pf then return nil end
-    local root = vim.trim(pf:read('*a') or ''); pf:close()
-    if root == '' then return nil end
-    for _, pid in ipairs(descendant_pids(root)) do
-      for _, line in ipairs(vim.fn.systemlist({ 'lsof', '-p', pid, '-Fn' })) do
-        local path = line:match('^n(.*/%.codex/sessions/.*/rollout%-.*%.jsonl)$')
-        if path then
-          local sid = path:match('([0-9a-fA-F]+%-[0-9a-fA-F]+%-[0-9a-fA-F]+%-[0-9a-fA-F]+%-[0-9a-fA-F]+)%.jsonl$')
-          if sid then return sid end
-        end
-      end
-    end
-    return nil
-  end
-
   local function current_session_id()
     local sid = vim.env.PAIR_SESSION_ID
     if sid and sid ~= '' then return sid end
@@ -890,7 +848,6 @@ do
     local agent = (vim.env.PAIR_AGENT and vim.env.PAIR_AGENT ~= '') and vim.env.PAIR_AGENT or 'claude'
     sid = config_session_id(data_dir, tag, agent)
     if sid then return sid end
-    if agent == 'codex' then return live_codex_session_id(data_dir, tag) end
     return nil
   end
 
diff --git a/tests/pair-review-target-test.sh b/tests/pair-review-target-test.sh
index 3dd8d41..d052170 100644
--- a/tests/pair-review-target-test.sh
+++ b/tests/pair-review-target-test.sh
@@ -16,11 +16,15 @@ PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=codex \
 got="$(jq -r '.session' "$RT/review-target-test.json")"
 [ "$got" = envsid ] && pass "uses PAIR_SESSION_ID when set" || fail "env session stamp ($got)"
 
-printf '{"agent":"codex","args":[],"session_id":"cfgsid"}\n' > "$RT/config-test-codex.json"
-PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=codex PAIR_SESSION_ID="" \
+cfgsid="12345678-1234-1234-1234-123456789abc"
+rollout="$RT/home/.codex/sessions/2026/08/19/rollout-2026-08-19T00-00-00-$cfgsid.jsonl"
+mkdir -p "$(dirname "$rollout")"
+printf '{"type":"session_meta","payload":{"id":"%s","parent_thread_id":null,"source":"cli"}}\n' "$cfgsid" > "$rollout"
+printf '{"agent":"codex","args":[],"session_id":"%s"}\n' "$cfgsid" > "$RT/config-test-codex.json"
+HOME="$RT/home" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=codex PAIR_SESSION_ID="" \
   "$ROOT/bin/pair" review target "$doc" ready >/dev/null
 got="$(jq -r '.session' "$RT/review-target-test.json")"
-[ "$got" = cfgsid ] && pass "falls back to config session_id" || fail "config session stamp ($got)"
+[ "$got" = "$cfgsid" ] && pass "falls back to config session_id" || fail "config session stamp ($got)"
 
 [ "$fails" -eq 0 ] || { printf 'pair-review-target-test FAILED (%d)\n' "$fails"; exit 1; }
 printf 'pair-review-target-test ok\n'
diff --git a/tests/pair-session-watch-test.sh b/tests/pair-session-watch-test.sh
index 6def71e..ce7f369 100644
--- a/tests/pair-session-watch-test.sh
+++ b/tests/pair-session-watch-test.sh
@@ -9,7 +9,7 @@ mkdir -p "$RT/bin" "$RT/data" "$RT/home/.codex/sessions/2026/06/25"
 
 sid="019eff64-6ceb-7e72-9d41-a735a97029ac"
 session_file="$RT/home/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-$sid.jsonl"
-: > "$session_file"
+printf '{"type":"session_meta","payload":{"id":"%s","parent_thread_id":null,"source":"cli"}}\n' "$sid" > "$session_file"
 
 cat > "$RT/bin/lsof" <<SH
 #!/usr/bin/env bash
diff --git a/tests/review-toggle-test.sh b/tests/review-toggle-test.sh
index f4d8599..3aee04e 100755
--- a/tests/review-toggle-test.sh
+++ b/tests/review-toggle-test.sh
@@ -149,10 +149,11 @@ vim.env.PAIR_AGENT = 'codex'
 os.remove(vim.env.PAIR_DATA_DIR .. '/config-' .. vim.env.PAIR_TAG .. '-codex.json')
 vim.fn.writefile({ '111' }, vim.env.PAIR_DATA_DIR .. '/agent-pid-' .. vim.env.PAIR_TAG)
 vim.fn.writefile({ '{"file":"' .. draft .. '","status":"ready","session":"12345678-1234-1234-1234-123456789abc"}' }, target)
-OUT:write((R.read_target() ~= nil) and 'live-codex-session-read ok\n' or 'live-codex-session-read FAIL\n')
+OUT:write((R.current_session_id() == nil) and 'no-live-codex-fallback ok\n' or 'no-live-codex-fallback FAIL\n')
+OUT:write((R.read_target() == nil) and 'unverified-live-target-stale ok\n' or 'unverified-live-target-stale FAIL\n')
 R.write_target(draft, 'ready')
 written = vim.json.decode(table.concat(vim.fn.readfile(target), '\n'))
-OUT:write((written.session == '12345678-1234-1234-1234-123456789abc') and 'live-codex-session-write ok\n' or 'live-codex-session-write FAIL\n')
+OUT:write((written.session == '') and 'unverified-live-target-unstamped ok\n' or 'unverified-live-target-unstamped FAIL\n')
 vim.env.PAIR_AGENT = 'claude'
 vim.env.PAIR_SESSION_ID = 'testsid'
 vim.fn.writefile({ '{"file":"/stale/prev.md","status":"ready","session":"oldsid"}' }, target)
@@ -214,8 +215,9 @@ grep -q 'old-unscoped-target-stale ok' "$RESULT" && pass "old unscoped target re
 grep -q 'fresh-unscoped-target-read ok' "$RESULT" && pass "same-nvim unscoped target remains readable" || fail "same-nvim unscoped target ignored"
 grep -q 'config-session-read ok' "$RESULT" && pass "read_target falls back to config session_id" || fail "read_target config fallback"
 grep -q 'config-session-write ok' "$RESULT" && pass "write_target stamps config session_id" || fail "write_target config fallback"
-grep -q 'live-codex-session-read ok' "$RESULT" && pass "read_target resolves live codex session_id" || fail "read_target live codex fallback"
-grep -q 'live-codex-session-write ok' "$RESULT" && pass "write_target stamps live codex session_id" || fail "write_target live codex fallback"
+grep -q 'no-live-codex-fallback ok' "$RESULT" && pass "current session does not guess from live Codex files" || fail "live Codex fallback remains"
+grep -q 'unverified-live-target-stale ok' "$RESULT" && pass "unverified live target is stale" || fail "unverified live target accepted"
+grep -q 'unverified-live-target-unstamped ok' "$RESULT" && pass "unverified live target remains unstamped" || fail "unverified live target stamped"
 grep -q 'pure-prompt ok'  "$RESULT" && pass "pure: no target → prompt"        || fail "pure prompt"
 grep -q 'pure-open ok'    "$RESULT" && pass "pure: target ready → open"       || fail "pure open"
 grep -q 'pure-wait ok'    "$RESULT" && pass "pure: target proposed → wait"    || fail "pure wait"
diff --git a/workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan-gate.md b/workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan-gate.md
new file mode 100644
index 0000000..4d48942
--- /dev/null
+++ b/workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan-gate.md
@@ -0,0 +1,33 @@
+---
+gate: plan-quality
+issue: 144
+id_prefix: PQ
+rounds:
+    - "n": 1
+      timestamp: "2026-08-19T07:24:49-07:00"
+      agent: codex
+      blocked: true
+      protocol_error: no valid findings block
+    - "n": 2
+      timestamp: "2026-08-19T07:24:57-07:00"
+      agent: claude
+      blocked: true
+      protocol_error: no valid findings block
+---
+
+# Gate ledger — pair-context-usage#144 (plan-quality)
+
+Findings this gate raised, the stable ids the binary assigned them, and how
+later rounds disposed of them. Generated — edit the gate, not this file.
+
+## Round 1 — 2026-08-19T07:24:49-07:00 (codex) — BLOCKED
+
+**Protocol error:** no valid findings block — this round contributed no findings.
+
+## Round 2 — 2026-08-19T07:24:57-07:00 (claude) — BLOCKED
+
+**Protocol error:** no valid findings block — this round contributed no findings.
+
+## Open findings
+
+(none — every finding has been disposed)
diff --git a/workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan.md b/workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan.md
index 1bfccb0..639c8a3 100644
--- a/workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan.md
+++ b/workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan.md
@@ -58,7 +58,7 @@
 - Modify: `cmd/internal/transcript/transcript.go`
 - Modify: `cmd/internal/transcript/transcript_test.go`
 
-- [ ] **Step 1: Write failing pure-classifier tests**
+- [x] **Step 1: Write failing pure-classifier tests**
 
 Add table tests for:
 
@@ -72,13 +72,13 @@ func TestCodexRootSessionIDFromEvent(t *testing.T) {
 
 Add file-adapter tests with a temp rollout tree proving `ReadCodexRootSessionID` reads a valid first event, rejects a subagent first event, and does not authorize a later `session_meta` when the first event is invalid. Define the bound as 1 MiB including the terminating newline, then cover nonexistent/unreadable input, exactly-at-limit acceptance, over-limit rejection, an unterminated first line, and a read-error path (directory or closed/erroring fixture) before implementation.
 
-- [ ] **Step 2: Run the focused tests and verify RED**
+- [x] **Step 2: Run the focused tests and verify RED**
 
 Run: `go test ./cmd/internal/transcript -run 'TestCodexRootSessionID|TestReadCodexRoot' -count=1 -v`
 
 Expected: FAIL because the root classifier/file adapter does not exist.
 
-- [ ] **Step 3: Implement the minimal classifier and bounded reader**
+- [x] **Step 3: Implement the minimal classifier and bounded reader**
 
 Add:
 
@@ -89,13 +89,13 @@ func ReadCodexRootSessionID(path string) string
 
 `CodexRootSessionID` must first extract the filename UUID, decode exactly one `session_meta`, require matching `payload.id`, nil/absent parent, and source `cli` or `exec`. `ReadCodexRootSessionID` reads one bounded line and delegates; it returns `""` for every IO/size/parse failure.
 
-- [ ] **Step 4: Run the focused tests and verify GREEN**
+- [x] **Step 4: Run the focused tests and verify GREEN**
 
 Run: `go test ./cmd/internal/transcript -count=1`
 
 Expected: PASS.
 
-- [ ] **Step 5: Commit the classifier**
+- [x] **Step 5: Commit the classifier**
 
 ```bash
 git add cmd/internal/transcript/transcript.go cmd/internal/transcript/transcript_test.go
@@ -118,27 +118,27 @@ git commit -m "transcript: #144: classify root Codex sessions" -m "Co-Authored-B
 - Modify: `cmd/internal/reviewcmd/reviewcmd_test.go`
 - Modify: `cmd/internal/reviewcmd/runtime.go`
 
-- [ ] **Step 1: Write failing ambiguous-candidate regressions**
+- [x] **Step 1: Write failing ambiguous-candidate regressions**
 
 For launcher, codexsid, and slug, create root and subagent rollout files in a temp Codex session tree. Have the shared process seam report the subagent first and root second. Assert each consumer skips the subagent and returns the root ID/path. Add a subagent-only case returning empty. Add config-backed regressions proving `transcript.SessionID` makes context and slug reject a polluted subagent config, and review targeting rejects polluted config before falling through to a valid live root.
 
-- [ ] **Step 2: Run the consumer tests and verify RED**
+- [x] **Step 2: Run the consumer tests and verify RED**
 
 Run: `go test ./cmd/internal/transcript ./cmd/internal/launcher ./cmd/internal/procutil ./cmd/internal/codexsid ./cmd/internal/slugcmd ./cmd/internal/contextcmd ./cmd/internal/reviewcmd -run 'Codex.*(Root|Subagent)|LiveCodex|PollutedCodex' -count=1 -v`
 
 Expected: FAIL because filename-only scans return the first subagent candidate.
 
-- [ ] **Step 3: Replace filename authorization with the shared adapter**
+- [x] **Step 3: Replace filename authorization with the shared adapter**
 
 Make `procutil` the only Go owner of `ps`/`lsof` parsing/traversal and route launcher, codexsid, slug, and sessionwatch's OS runtime through it. Replace direct regex or `CodexSessionIDFromPath` success checks with `transcript.ReadCodexRootSessionID(path)`, continuing the scan when it returns empty. Delete `codexsid.rolloutRE`; retain `CodexSessionIDFromPath` only as the classifier's low-level path parser. Make `transcript.SessionID` validate Codex config IDs through the same file adapter, and expose that validated config resolution through reviewcmd's injected runtime rather than parsing JSON locally.
 
-- [ ] **Step 4: Run the three consumer packages and verify GREEN**
+- [x] **Step 4: Run the three consumer packages and verify GREEN**
 
 Run: `go test ./cmd/internal/transcript ./cmd/internal/launcher ./cmd/internal/procutil ./cmd/internal/codexsid ./cmd/internal/slugcmd ./cmd/internal/contextcmd ./cmd/internal/reviewcmd -count=1`
 
 Expected: PASS.
 
-- [ ] **Step 5: Commit the consumer sweep**
+- [x] **Step 5: Commit the consumer sweep**
 
 ```bash
 git add cmd/internal/transcript cmd/internal/launcher/osruntime.go cmd/internal/launcher/osruntime_test.go cmd/internal/procutil cmd/internal/codexsid cmd/internal/slugcmd cmd/internal/contextcmd cmd/internal/reviewcmd
@@ -154,7 +154,7 @@ git commit -m "session: #144: reject live Codex subagents" -m "Co-Authored-By: O
 - Modify: `cmd/internal/sessionwatch/sessionwatch.go`
 - Modify: `cmd/internal/sessionwatch/sessionwatch_test.go`
 
-- [ ] **Step 1: Write failing watcher regressions**
+- [x] **Step 1: Write failing watcher regressions**
 
 Extend the fake runtime with first-event data. Add separate tests proving:
 
@@ -163,23 +163,23 @@ Extend the fake runtime with first-event data. Add separate tests proving:
 - subagent-only discovery writes no config and continues until process exit/timeout;
 - a rejected malformed candidate does not hide a later root.
 
-- [ ] **Step 2: Run watcher tests and verify RED**
+- [x] **Step 2: Run watcher tests and verify RED**
 
 Run: `go test ./cmd/internal/sessionwatch -run 'Codex.*(Root|Subagent)|ContinuesPastRejected' -count=1 -v`
 
 Expected: FAIL because `AgentSpec.Match` authorizes filename UUIDs without metadata.
 
-- [ ] **Step 3: Add the thin injected first-event seam and shared authorization**
+- [x] **Step 3: Add the thin injected first-event seam and shared authorization**
 
 Add `ReadFirstLine(path string) ([]byte, error)` to `Runtime` and implement it with the same 1 MiB contract as the transcript adapter. Route OS process traversal/path listing through `procutil`. Keep `AgentSpec.Match` as shape extraction, but before any Codex result becomes returnable, call `transcript.CodexRootSessionID(result.Path, firstEvent)`. Convert explicit subagents/invalid metadata to rejected candidates, not terminal near-misses, and continue scanning.
 
-- [ ] **Step 4: Run watcher tests and verify GREEN**
+- [x] **Step 4: Run watcher tests and verify GREEN**
 
 Run: `go test ./cmd/internal/sessionwatch -count=1`
 
 Expected: PASS on both main and, after integration, the #143 lifecycle branch behavior.
 
-- [ ] **Step 5: Commit watcher authorization**
+- [x] **Step 5: Commit watcher authorization**
 
 ```bash
 git add cmd/internal/sessionwatch
@@ -198,7 +198,7 @@ git commit -m "sessionwatch: #144: persist only root Codex sessions" -m "Co-Auth
 - Modify: `cmd/internal/launcher/osruntime.go`
 - Modify: `cmd/internal/launcher/osruntime_test.go`
 
-- [ ] **Step 1: Write failing config-picker and Alt+n regressions**
+- [x] **Step 1: Write failing config-picker and Alt+n regressions**
 
 First add pure table tests for `DecideAutomaticResumeConfig(agent, saved, sessionValid)` returning sanitized config plus `quarantine` and `warn` intent without IO. Then add integration tests where saved config/ledger contains a real on-disk subagent rollout ID:
 
@@ -207,23 +207,23 @@ First add pure table tests for `DecideAutomaticResumeConfig(agent, saved, sessio
 - a valid root saved ID remains resumable;
 - an explicit user `codex resume <id>` still bypasses automatic saved-state selection.
 
-- [ ] **Step 2: Run launcher tests and verify RED**
+- [x] **Step 2: Run launcher tests and verify RED**
 
 Run: `go test ./cmd/internal/launcher -run 'PollutedCodex|SavedCodexRoot|ExplicitCodexResume' -count=1 -v`
 
 Expected: FAIL because restart fallback currently prefers `saved.SessionID` without root validation and config-picker leaves invalid config on disk.
 
-- [ ] **Step 3: Implement one automatic-resume validation policy**
+- [x] **Step 3: Implement one automatic-resume validation policy**
 
 Update `AgentSessionExists("codex", ...)` to resolve the rollout and require `ReadCodexRootSessionID`. Implement the pure decision over saved state plus `sessionValid`; thin config-picker and restart callers gather validity through `Runtime.AgentSessionExists`, apply returned quarantine/warning intent with `Remove`/stderr, and pass only sanitized state onward. Exercise both config-origin and ledger-fallback saved state. Do not apply this policy to explicit argv resume IDs.
 
-- [ ] **Step 4: Run launcher tests and verify GREEN**
+- [x] **Step 4: Run launcher tests and verify GREEN**
 
 Run: `go test ./cmd/internal/launcher -count=1`
 
 Expected: PASS.
 
-- [ ] **Step 5: Commit persisted-state quarantine**
+- [x] **Step 5: Commit persisted-state quarantine**
 
 ```bash
 git add cmd/internal/launcher
@@ -236,7 +236,7 @@ git commit -m "launcher: #144: quarantine subagent resume state" -m "Co-Authored
 - Modify: `nvim/init.lua`
 - Modify: `tests/review-toggle-test.sh`
 
-- [ ] **Step 1: Add a failing headless derivation regression**
+- [x] **Step 1: Add a failing headless derivation regression**
 
 Extend `tests/review-toggle-test.sh` so `current_session_id`:
 
@@ -245,23 +245,23 @@ Extend `tests/review-toggle-test.sh` so `current_session_id`:
 - returns nil when neither exists;
 - never invokes fake `ps` or `lsof` binaries when config is absent.
 
-- [ ] **Step 2: Run the headless test and verify RED**
+- [x] **Step 2: Run the headless test and verify RED**
 
 Run: `bash tests/review-toggle-test.sh`
 
 Expected: FAIL because the current nil-config Codex path calls `live_codex_session_id`.
 
-- [ ] **Step 3: Delete Lua process/rollout discovery**
+- [x] **Step 3: Delete Lua process/rollout discovery**
 
 Remove `descendant_pids` and `live_codex_session_id`; keep `current_session_id` as `PAIR_SESSION_ID` then config only. Update comments to state that Go authorizes and quarantines automatic Codex identity.
 
-- [ ] **Step 4: Run the headless test and verify GREEN**
+- [x] **Step 4: Run the headless test and verify GREEN**
 
 Run: `bash tests/review-toggle-test.sh`
 
 Expected: PASS.
 
-- [ ] **Step 5: Commit UI derivation**
+- [x] **Step 5: Commit UI derivation**
 
 ```bash
 git add nvim/init.lua tests/review-toggle-test.sh
@@ -274,11 +274,11 @@ git commit -m "nvim: #144: consume validated session identity" -m "Co-Authored-B
 - Modify if needed: `atlas/session-identity.md`
 - Modify: `workshop/issues/000144-reject-codex-subagent-sessions-during-pair-identity-discovery.md`
 
-- [ ] **Step 1: Update the atlas at the implemented boundary**
+- [x] **Step 1: Update the atlas at the implemented boundary**
 
 Document that Codex automatic identity requires matching root `session_meta`, persisted IDs are revalidated/quarantined, and Neovim derives from validated Pair state. Confirm `atlas/index.md` already links `session-identity.md`.
 
-- [ ] **Step 2: Run focused verification**
+- [x] **Step 2: Run focused verification**
 
 Run:
 
@@ -289,7 +289,7 @@ bash tests/review-toggle-test.sh
 
 Expected: PASS.
 
-- [ ] **Step 3: Run repository-wide verification**
+- [x] **Step 3: Run repository-wide verification**
 
 Run the repository's available full suite from the checkout, including generated runtime assets if required by the current Make targets:
 
@@ -303,7 +303,7 @@ git diff --check
 
 Expected: PASS with no warnings attributable to the change. The guarded setup materializes the canonical sibling only when no path exists; it refuses an existing wrong/broken link instead of replacing it. This makes Pair's `Makefile -> ../ariadne/Makefile` and nested plain `make -C "$repo_root"` calls resolve normally. `make -n test` was verified after this setup in the planning checkout. If the canonical repo is unavailable, stop before testing.
 
-- [ ] **Step 4: Perform a shadow-sweep**
+- [x] **Step 4: Perform a shadow-sweep**
 
 Run:
 
@@ -313,11 +313,11 @@ rg -n 'CodexSessionIDFromPath|rolloutRE|endUUIDRE|live_codex_session_id|\.codex/
 
 Expected: every path that authorizes automatic identity reaches the shared root classifier; no Neovim or package-local filename-only authorizer remains. Low-level path-shape tests may remain only in `transcript`.
 
-- [ ] **Step 5: Record evidence and check every issue-plan box**
+- [x] **Step 5: Record evidence and check every issue-plan box**
 
 Append TDD red/green commands, focused/full verification, shadow-sweep result, and atlas disposition to `## Log`; tick all issue and durable-plan checkboxes. Do not hand-edit issue status.
 
-- [ ] **Step 6: Commit documentation and verification record**
+- [x] **Step 6: Commit documentation and verification record**
 
 ```bash
 git add atlas/session-identity.md workshop/issues/000144-reject-codex-subagent-sessions-during-pair-identity-discovery.md workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan.md


codex
I’ll verify the issue, durable plan (including core-concepts and log), repository instructions, the complete base-to-HEAD diff, and run focused/full checks where feasible. Then I’ll report only evidence-backed findings in the required verdict format.
exec
/bin/zsh -lc "pwd && rg --files -g 'AGENTS.md' -g 'workshop/issues/000144*' -g 'workshop/plans/000144*' -g 'go.mod' -g 'Makefile' | sort && git status --short && git diff --stat 356ec01f5aaf6c48d3fcf73a7aff1bd090c8dc79..HEAD" in /private/tmp/pair-context-usage
 succeeded in 0ms:
/private/tmp/pair-context-usage
go.mod
workshop/issues/000144-reject-codex-subagent-sessions-during-pair-identity-discovery.md
workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan-gate.md
workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan.md
?? workshop/plans/000143-keep-agent-session-discovery-alive-after-startup-timeout-plan-gate.md
 atlas/session-identity.md                          |  23 ++++
 cmd/internal/codexsid/codexsid.go                  |  62 ++--------
 cmd/internal/codexsid/codexsid_test.go             |  54 +++++---
 cmd/internal/contextcmd/contextcmd.go              |   2 +-
 cmd/internal/contextcmd/contextcmd_test.go         |  21 ++++
 cmd/internal/launcher/createflow.go                |  24 +++-
 cmd/internal/launcher/createflow_test.go           |  66 ++++++++++
 cmd/internal/launcher/markers.go                   |  11 ++
 cmd/internal/launcher/markers_test.go              |  24 ++++
 cmd/internal/launcher/osruntime.go                 |   5 +-
 cmd/internal/launcher/osruntime_test.go            |  28 +++--
 cmd/internal/reviewcmd/reviewcmd_test.go           |  28 +++--
 cmd/internal/reviewcmd/run.go                      |  21 +---
 cmd/internal/reviewcmd/run_test.go                 |  40 +++---
 cmd/internal/reviewcmd/runtime.go                  |   6 +
 cmd/internal/sessionwatch/run.go                   |  19 ++-
 cmd/internal/sessionwatch/run_test.go              |  89 +++++++++++++-
 cmd/internal/sessionwatch/runtime.go               |  42 ++-----
 cmd/internal/slugcmd/slugcmd.go                    |   4 +-
 cmd/internal/slugcmd/slugcmd_test.go               |  23 +++-
 cmd/internal/transcript/transcript.go              |  66 +++++++++-
 cmd/internal/transcript/transcript_test.go         | 136 +++++++++++++++++++++
 nvim/init.lua                                      |  43 -------
 tests/pair-review-target-test.sh                   |  10 +-
 tests/pair-session-watch-test.sh                   |   2 +-
 tests/review-toggle-test.sh                        |  10 +-
 ...gent-sessions-during-pair-identity-discovery.md |  29 ++++-
 ...ons-during-pair-identity-discovery-plan-gate.md |  33 +++++
 ...sessions-during-pair-identity-discovery-plan.md |  62 +++++-----
 29 files changed, 727 insertions(+), 256 deletions(-)

exec
/bin/zsh -lc "find .. -name AGENTS.md -print 2>/dev/null; sed -n '1,260p' workshop/issues/000144-reject-codex-subagent-sessions-during-pair-identity-discovery.md; sed -n '1,380p' workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan.md; sed -n '1,160p' workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan-gate.md" in /private/tmp/pair-context-usage
 succeeded in 1174ms:
---
id: 000144
status: working
deps: []
github_issue:
created: 2026-08-19
updated: 2026-08-19
estimate_hours: 2.22
started: 2026-08-19T07:09:43-07:00
---

# Reject Codex subagent sessions during Pair identity discovery

## Problem

Pair identifies a live Codex conversation from rollout filenames exposed by
the agent process tree. Codex subagents write rollout files in the same
directory and may be open in the same process tree, so filename-only discovery
can persist a subagent ID. `Alt+n` then gives that ID precedence over the saved
config and resumes the subagent rather than the operator's root conversation.

The same ambiguity exists in the asynchronous session watcher's birth-time
fallback. Issue #143 extends that watcher for the lifetime of an agent, making
it more likely to observe later subagent rollouts when the root session has not
already been captured.

## Spec

Define Codex root-session identity once in `cmd/internal/transcript`: a rollout
is eligible only when its filename contains a valid session UUID and its first
JSONL event is a coherent `session_meta` record for that same UUID. An accepted
record has `type: "session_meta"`, `payload.id` equal to the filename UUID,
`payload.parent_thread_id` absent or null, and the currently observed root
`payload.source` string `"cli"` or `"exec"`. Reject the observed subagent shape
(a non-null parent plus an object-valued source containing `subagent`) and fail
closed on unreadable, malformed, incomplete, mismatched, or unknown-source
metadata. A rejected candidate does not end a scan; later candidates remain
eligible.

Apply the classifier anywhere Pair turns an open or newly-created Codex rollout
into the active conversation identity: launcher `Alt+n` capture, asynchronous
session watching (both `lsof` and birth-time discovery), the shared
`codexsid` resolver used by review targeting, and live slug transcript
resolution. Remove Neovim's independent `ps`/`lsof` filename parser for review
target scoping; it must derive from `PAIR_SESSION_ID` or the Go-authored saved
config rather than restating Codex identity rules in Lua. Path-shape extraction
remains a lower-level helper only; it must not by itself authorize a resumable
session ID. This is the shadow-sweep for ARCH-PURPOSE and keeps one
classification contract under ARCH-DRY.

Treat persisted Codex IDs as untrusted at automatic resume boundaries. Before
the config picker or `Alt+n` fallback composes a saved ID, resolve its rollout
and apply the same root classifier. Warn, clear the invalid ID from automatic
selection, and remove the polluted config so Neovim cannot consume it; preserve
the saved non-resume args for a fresh launch. Ledger fallback remains subject
to the same validation. An explicitly typed Codex `resume <id>` remains user
authority and is outside automatic discovery.

Configuration readers that derive transcript or review identity must validate
Codex IDs through the same transcript contract before returning them. This
includes context usage, slug generation, and review-target scoping; a polluted
config behaves as missing identity and may fall through only to a validated
live root. Neovim is the deliberate thin consumer exception: it does not parse
transcripts, and reads only the environment/config state whose Pair launch
boundary has already quarantined invalid automatic identity.

Keep filesystem and process discovery at the existing integration seams. The
metadata decision is a pure function over a path and first JSONL event; a thin
transcript adapter reads only the first JSONL event, and candidate scanners
continue until that classifier accepts a root (ARCH-PURE). Tests pass root and
subagent files through the shared selector using temporary rollout trees, then
exercise each consumer with its existing stateful runtime/process seam
(ARCH-MOCK).

Alternatives rejected:

- PID ancestry cannot distinguish root and subagent sessions because Codex may
  host both inside the same process tree.
- Choosing the oldest or newest rollout merely changes which concurrent
  session wins and cannot establish semantic identity.

## Done when

- Given an open root rollout and an open subagent rollout, `Alt+n` captures the
  root session regardless of rollout order.
- Session watcher `lsof` and birth-time discovery ignore Codex subagent
  rollouts and persist the root ID when it becomes available.
- Review-target and slug live-session resolution use the same root-only
  classifier; context usage rejects a polluted config; Neovim no longer parses
  live rollout filenames independently.
- A saved config containing a subagent ID is quarantined before config-picker
  or `Alt+n` automatic resume. If no valid live root is available, Pair starts a
  fresh session with the saved non-resume args instead of resuming the
  subagent.
- Malformed, incomplete, mismatched, or explicitly nested `session_meta`
  records, plus unknown source shapes, do not authorize a session ID; scanners
  still find a later valid root candidate.
- Focused and repository-wide automated tests pass.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec                design=0.25 impl=0.10
item: smaller-go-module        design=0.05 impl=0.18
item: cross-cutting-refactor   design=0.15 impl=0.20
item: smaller-go-module        design=0.05 impl=0.18
item: smaller-go-module        design=0.05 impl=0.18
item: lua-neovim               design=0.25 impl=0.24
item: atlas-docs               design=0.02 impl=0.04
item: milestone-review         design=0.00 impl=0.16
design-buffer: 0.15
total: 2.22
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md`
against `baseline-v3.1.md`. Method A only.*

## Plan

- [x] Add the pure Codex root-session metadata classifier and exhaustive unit
  tests for root, subagent, malformed, incomplete, and mismatched events.
- [x] Route every live Codex identity consumer through the shared classifier,
  with regressions for ambiguous root/subagent candidates.
- [x] Validate and quarantine persisted Codex IDs at automatic resume
  boundaries, and remove Neovim's independent live filename parser.
- [x] Verify focused packages and the full repository; update the session
  identity atlas if its current map omits root-vs-subagent semantics.

## Log

### 2026-08-19

- Root cause evidence: the saved config contained session
  `01a017b6-af00-7c91-a656-0611a3750669`; that rollout's first event declares
  parent thread `01a016d8-0a53-72e2-a62a-456c0c72f1a2` and a depth-1 subagent
  source. The live root rollout instead had a null parent and CLI source.
- ARCH-DRY/ARCH-PURPOSE: filename-to-ID logic currently exists in transcript,
  sessionwatch, and codexsid, while launcher and slug consume the filename-only
  result. The fix must centralize semantic authorization and cover every live
  identity consumer rather than patching only `Alt+n`.
- TDD implementation: added a bounded first-event root classifier; routed
  launcher, watcher, context, slug, and review consumers through it; added
  ambiguous subagent/root ordering regressions for process and birth scans.
- Automatic resume now revalidates saved Codex identity in both the config
  picker and the in-process `Alt+n` loop. Invalid IDs quarantine the canonical
  config while saved non-resume flags survive the fresh launch; ledger fallback
  is covered by the same decision.
- Removed Neovim's duplicate process/rollout parser. The review-target headless
  test proves an available live rollout cannot authorize or stamp identity
  without environment/config state.
- Verification: the eight focused identity packages passed; the headless
  review-toggle regression passed; environment-cleared `go test ./... -count=1`
  passed; and the full environment-cleared `make test` passed after updating
  two shell integration fixtures to provide the now-required root metadata.
- Shadow sweep found no package-local or Neovim filename-only authorizer. The
  remaining `CodexSessionIDFromPath` is the transcript classifier's low-level
  path parser; sessionwatch's UUID regex is candidate-shape extraction followed
  by metadata authorization; Neovim's remaining Codex filesystem lookup is an
  age hint for a separately established ID, not identity discovery.
- Updated `atlas/session-identity.md` with root metadata, quarantine, and thin
  Neovim consumer semantics (ARCH-DRY, ARCH-PURPOSE).

## Revisions

### 2026-08-19 07:18 PDT — Fresh-context spec review

- Added Neovim's review-target `ps`/`lsof` scanner to the consumer sweep and
  specified removing that duplicate in favor of validated Go-authored state.
- Added quarantine semantics for already-polluted config/ledger IDs so a failed
  live lookup cannot fall back to the same subagent.
- Defined accepted root metadata (`cli`/`exec`, null parent, matching ID),
  fail-closed unknown shapes, scan continuation, and the pure classifier/thin
  first-event IO seam.

### 2026-08-19 07:29 PDT — Fresh-context plan review

- Expanded persisted-ID validation to context, slug, and review-target config
  readers; polluted configuration is missing identity, never authority.
- Required all Go process discovery implementations to reuse `procutil` for
  `ps`/`lsof` parsing while preserving their injected/stateful test seams.

### 2026-08-19 07:38 PDT — Post-plan estimate

- After the plan cleared fresh-context review, derived 2.22 ship-wall-clock
  hours from eight v3.1 Method A primitives with the thorough-plan 15% design
  buffer and already-scaled implementation values.
# Reject Codex Subagent Sessions Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ensure every automatic Codex identity path accepts only the operator's root session and never persists or resumes a subagent rollout.

**Architecture:** Put semantic rollout classification in `cmd/internal/transcript` as a pure path-plus-first-event decision with a thin bounded file reader. Existing process/filesystem seams gather candidates; every consumer scans past rejected candidates and delegates authorization to the shared classifier. Automatic resume validates persisted IDs through the same contract, while Neovim stops reimplementing rollout discovery and consumes only validated Pair state.

**Tech Stack:** Go, JSONL, existing `procutil` and runtime fakes, Neovim Lua, shell-driven headless Neovim tests.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `CodexRootSessionID` | `cmd/internal/transcript/transcript.go` | new |
| `DecideAutomaticResumeConfig` | `cmd/internal/launcher/createflow.go` | new |

- **`CodexRootSessionID`** — authorizes a session UUID from a rollout path plus its first JSONL event.
  - **Relationships:** N:1 with a Codex process tree: many rollout candidates may be visible, exactly one accepted root ID is selected by each consumer.
  - **DRY rationale:** replaces filename-only authorization duplicated by sessionwatch, launcher, codexsid, slug, and Neovim (ARCH-DRY).
  - **Future extensions:** widen the accepted root source enum only when a captured upstream `session_meta` fixture proves a new root shape.
- **`DecideAutomaticResumeConfig`** — projects a saved config plus already-observed validation facts into safe automatic-resume intent: preserve args, retain a validated ID, or clear an invalid ID and request quarantine/warning.
  - **Relationships:** 1:1 with a loaded saved config; consumed by both config-picker and restart-marker flows.
  - **DRY rationale:** prevents two automatic-resume boundaries from independently deciding whether persisted Codex identity is trustworthy.
  - **Future extensions:** agent-specific persisted-identity validators can join without weakening explicit user-supplied resume authority.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ReadCodexRootSessionID` | `cmd/internal/transcript/transcript.go` | new | bounded first-line filesystem read |
| `sessionwatch.Runtime.ReadFirstLine` | `cmd/internal/sessionwatch/run.go`, `runtime.go` | new | watcher filesystem seam |
| Shared process candidate seam | `cmd/internal/procutil/procutil.go` | modified | `ps`/`lsof` parsing and traversal |
| Validated Pair config | `nvim/init.lua` | modified | asynchronous Go-authored session identity |

- **`ReadCodexRootSessionID`** — reads only the first JSONL event, then calls the pure classifier; unreadable, oversized, or unterminated/incomplete metadata fails closed.
  - **Injected into:** process candidate scans call this adapter after their existing `ps`/`lsof` seam returns paths; sessionwatch uses its own injected first-line reader and calls the pure classifier directly.
  - **Future extensions:** expose classification diagnostics if adaptation telemetry needs to distinguish malformed metadata from explicit subagents.
- **`sessionwatch.Runtime.ReadFirstLine`** — keeps watcher scheduling and candidate ordering deterministic under its stateful fake without reading whole, potentially large transcripts.
  - **Injected into:** `discover` and `discoverByBirth` candidate authorization.
  - **Future extensions:** none planned; deliberately narrower than general transcript reads.
- **Shared process candidate seam** — one implementation owns external `ps`/`lsof` parsing and traversal; tests provide temp rollout trees plus the existing fake command/state model (ARCH-MOCK).
  - **Injected into:** launcher `LiveAgentSessionID`, `codexsid.ResolveSessionID`, slug transcript resolution, and sessionwatch's OS runtime while sessionwatch retains its injected runtime interface.
  - **Future extensions:** command execution injection can widen here without duplicating parsers in consumers.
- **Validated Pair config** — Neovim reads `PAIR_SESSION_ID` or `config-<tag>-<agent>.json`; it no longer shells out or parses rollout filenames.
  - **Injected into:** review-target scoping.
  - **Future extensions:** a typed Pair identity sidecar could replace config reads if more UI consumers emerge.

## Chunk 1: Root classifier and consumers

### Task 1: Add the canonical Codex root-session classifier

**Files:**
- Modify: `cmd/internal/transcript/transcript.go`
- Modify: `cmd/internal/transcript/transcript_test.go`

- [x] **Step 1: Write failing pure-classifier tests**

Add table tests for:

```go
func TestCodexRootSessionIDFromEvent(t *testing.T) {
    // accept matching session_meta with parent_thread_id null/absent and source "cli" or "exec"
    // reject subagent object source + parent, mismatched payload.id, non-session_meta,
    // malformed/incomplete JSON, unknown string/object source, and malformed filename
}
```

Add file-adapter tests with a temp rollout tree proving `ReadCodexRootSessionID` reads a valid first event, rejects a subagent first event, and does not authorize a later `session_meta` when the first event is invalid. Define the bound as 1 MiB including the terminating newline, then cover nonexistent/unreadable input, exactly-at-limit acceptance, over-limit rejection, an unterminated first line, and a read-error path (directory or closed/erroring fixture) before implementation.

- [x] **Step 2: Run the focused tests and verify RED**

Run: `go test ./cmd/internal/transcript -run 'TestCodexRootSessionID|TestReadCodexRoot' -count=1 -v`

Expected: FAIL because the root classifier/file adapter does not exist.

- [x] **Step 3: Implement the minimal classifier and bounded reader**

Add:

```go
func CodexRootSessionID(path string, firstEvent []byte) string
func ReadCodexRootSessionID(path string) string
```

`CodexRootSessionID` must first extract the filename UUID, decode exactly one `session_meta`, require matching `payload.id`, nil/absent parent, and source `cli` or `exec`. `ReadCodexRootSessionID` reads one bounded line and delegates; it returns `""` for every IO/size/parse failure.

- [x] **Step 4: Run the focused tests and verify GREEN**

Run: `go test ./cmd/internal/transcript -count=1`

Expected: PASS.

- [x] **Step 5: Commit the classifier**

```bash
git add cmd/internal/transcript/transcript.go cmd/internal/transcript/transcript_test.go
git commit -m "transcript: #144: classify root Codex sessions" -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
```

### Task 2: Route process-based identity consumers through the classifier

**Files:**
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/osruntime_test.go`
- Modify: `cmd/internal/procutil/procutil.go`
- Modify: `cmd/internal/procutil/procutil_test.go`
- Modify: `cmd/internal/codexsid/codexsid.go`
- Modify: `cmd/internal/codexsid/codexsid_test.go`
- Modify: `cmd/internal/slugcmd/slugcmd.go`
- Modify: `cmd/internal/slugcmd/slugcmd_test.go`
- Modify: `cmd/internal/contextcmd/contextcmd_test.go`
- Modify: `cmd/internal/reviewcmd/run.go`
- Modify: `cmd/internal/reviewcmd/reviewcmd_test.go`
- Modify: `cmd/internal/reviewcmd/runtime.go`

- [x] **Step 1: Write failing ambiguous-candidate regressions**

For launcher, codexsid, and slug, create root and subagent rollout files in a temp Codex session tree. Have the shared process seam report the subagent first and root second. Assert each consumer skips the subagent and returns the root ID/path. Add a subagent-only case returning empty. Add config-backed regressions proving `transcript.SessionID` makes context and slug reject a polluted subagent config, and review targeting rejects polluted config before falling through to a valid live root.

- [x] **Step 2: Run the consumer tests and verify RED**

Run: `go test ./cmd/internal/transcript ./cmd/internal/launcher ./cmd/internal/procutil ./cmd/internal/codexsid ./cmd/internal/slugcmd ./cmd/internal/contextcmd ./cmd/internal/reviewcmd -run 'Codex.*(Root|Subagent)|LiveCodex|PollutedCodex' -count=1 -v`

Expected: FAIL because filename-only scans return the first subagent candidate.

- [x] **Step 3: Replace filename authorization with the shared adapter**

Make `procutil` the only Go owner of `ps`/`lsof` parsing/traversal and route launcher, codexsid, slug, and sessionwatch's OS runtime through it. Replace direct regex or `CodexSessionIDFromPath` success checks with `transcript.ReadCodexRootSessionID(path)`, continuing the scan when it returns empty. Delete `codexsid.rolloutRE`; retain `CodexSessionIDFromPath` only as the classifier's low-level path parser. Make `transcript.SessionID` validate Codex config IDs through the same file adapter, and expose that validated config resolution through reviewcmd's injected runtime rather than parsing JSON locally.

- [x] **Step 4: Run the three consumer packages and verify GREEN**

Run: `go test ./cmd/internal/transcript ./cmd/internal/launcher ./cmd/internal/procutil ./cmd/internal/codexsid ./cmd/internal/slugcmd ./cmd/internal/contextcmd ./cmd/internal/reviewcmd -count=1`

Expected: PASS.

- [x] **Step 5: Commit the consumer sweep**

```bash
git add cmd/internal/transcript cmd/internal/launcher/osruntime.go cmd/internal/launcher/osruntime_test.go cmd/internal/procutil cmd/internal/codexsid cmd/internal/slugcmd cmd/internal/contextcmd cmd/internal/reviewcmd
git commit -m "session: #144: reject live Codex subagents" -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
```

### Task 3: Make sessionwatch authorize Codex metadata

**Files:**
- Modify: `cmd/internal/sessionwatch/run.go`
- Modify: `cmd/internal/sessionwatch/runtime.go`
- Modify: `cmd/internal/sessionwatch/run_test.go`
- Modify: `cmd/internal/sessionwatch/sessionwatch.go`
- Modify: `cmd/internal/sessionwatch/sessionwatch_test.go`

- [x] **Step 1: Write failing watcher regressions**

Extend the fake runtime with first-event data. Add separate tests proving:

- `lsof` reports a subagent before a root and the root ID wins;
- birth-time discovery sees a newer subagent and an older eligible root and the root wins;
- subagent-only discovery writes no config and continues until process exit/timeout;
- a rejected malformed candidate does not hide a later root.

- [x] **Step 2: Run watcher tests and verify RED**

Run: `go test ./cmd/internal/sessionwatch -run 'Codex.*(Root|Subagent)|ContinuesPastRejected' -count=1 -v`

Expected: FAIL because `AgentSpec.Match` authorizes filename UUIDs without metadata.

- [x] **Step 3: Add the thin injected first-event seam and shared authorization**

Add `ReadFirstLine(path string) ([]byte, error)` to `Runtime` and implement it with the same 1 MiB contract as the transcript adapter. Route OS process traversal/path listing through `procutil`. Keep `AgentSpec.Match` as shape extraction, but before any Codex result becomes returnable, call `transcript.CodexRootSessionID(result.Path, firstEvent)`. Convert explicit subagents/invalid metadata to rejected candidates, not terminal near-misses, and continue scanning.

- [x] **Step 4: Run watcher tests and verify GREEN**

Run: `go test ./cmd/internal/sessionwatch -count=1`

Expected: PASS on both main and, after integration, the #143 lifecycle branch behavior.

- [x] **Step 5: Commit watcher authorization**

```bash
git add cmd/internal/sessionwatch
git commit -m "sessionwatch: #144: persist only root Codex sessions" -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
```

## Chunk 2: Persisted-state safety and UI derivation

### Task 4: Quarantine polluted automatic-resume state

**Files:**
- Modify: `cmd/internal/launcher/createflow.go`
- Modify: `cmd/internal/launcher/createflow_test.go`
- Modify: `cmd/internal/launcher/lifecycle.go`
- Modify: `cmd/internal/launcher/markers_test.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/osruntime_test.go`

- [x] **Step 1: Write failing config-picker and Alt+n regressions**

First add pure table tests for `DecideAutomaticResumeConfig(agent, saved, sessionValid)` returning sanitized config plus `quarantine` and `warn` intent without IO. Then add integration tests where saved config/ledger contains a real on-disk subagent rollout ID:

- config picker warns, removes the polluted config, preserves saved args, and offers no resume action;
- `Alt+n` has no valid live ID, rejects the saved subagent, removes the config, and relaunches fresh with saved non-resume args;
- a valid root saved ID remains resumable;
- an explicit user `codex resume <id>` still bypasses automatic saved-state selection.

- [x] **Step 2: Run launcher tests and verify RED**

Run: `go test ./cmd/internal/launcher -run 'PollutedCodex|SavedCodexRoot|ExplicitCodexResume' -count=1 -v`

Expected: FAIL because restart fallback currently prefers `saved.SessionID` without root validation and config-picker leaves invalid config on disk.

- [x] **Step 3: Implement one automatic-resume validation policy**

Update `AgentSessionExists("codex", ...)` to resolve the rollout and require `ReadCodexRootSessionID`. Implement the pure decision over saved state plus `sessionValid`; thin config-picker and restart callers gather validity through `Runtime.AgentSessionExists`, apply returned quarantine/warning intent with `Remove`/stderr, and pass only sanitized state onward. Exercise both config-origin and ledger-fallback saved state. Do not apply this policy to explicit argv resume IDs.

- [x] **Step 4: Run launcher tests and verify GREEN**

Run: `go test ./cmd/internal/launcher -count=1`

Expected: PASS.

- [x] **Step 5: Commit persisted-state quarantine**

```bash
git add cmd/internal/launcher
git commit -m "launcher: #144: quarantine subagent resume state" -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
```

### Task 5: Remove Neovim's duplicate live rollout scanner

**Files:**
- Modify: `nvim/init.lua`
- Modify: `tests/review-toggle-test.sh`

- [x] **Step 1: Add a failing headless derivation regression**

Extend `tests/review-toggle-test.sh` so `current_session_id`:

- prefers non-empty `PAIR_SESSION_ID`;
- otherwise reads the validated config ID;
- returns nil when neither exists;
- never invokes fake `ps` or `lsof` binaries when config is absent.

- [x] **Step 2: Run the headless test and verify RED**

Run: `bash tests/review-toggle-test.sh`

Expected: FAIL because the current nil-config Codex path calls `live_codex_session_id`.

- [x] **Step 3: Delete Lua process/rollout discovery**

Remove `descendant_pids` and `live_codex_session_id`; keep `current_session_id` as `PAIR_SESSION_ID` then config only. Update comments to state that Go authorizes and quarantines automatic Codex identity.

- [x] **Step 4: Run the headless test and verify GREEN**

Run: `bash tests/review-toggle-test.sh`

Expected: PASS.

- [x] **Step 5: Commit UI derivation**

```bash
git add nvim/init.lua tests/review-toggle-test.sh
git commit -m "nvim: #144: consume validated session identity" -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
```

### Task 6: Update maps and verify the complete change

**Files:**
- Modify if needed: `atlas/session-identity.md`
- Modify: `workshop/issues/000144-reject-codex-subagent-sessions-during-pair-identity-discovery.md`

- [x] **Step 1: Update the atlas at the implemented boundary**

Document that Codex automatic identity requires matching root `session_meta`, persisted IDs are revalidated/quarantined, and Neovim derives from validated Pair state. Confirm `atlas/index.md` already links `session-identity.md`.

- [x] **Step 2: Run focused verification**

Run:

```bash
go test ./cmd/internal/transcript ./cmd/internal/sessionwatch ./cmd/internal/launcher ./cmd/internal/procutil ./cmd/internal/codexsid ./cmd/internal/slugcmd ./cmd/internal/contextcmd ./cmd/internal/reviewcmd -count=1
bash tests/review-toggle-test.sh
```

Expected: PASS.

- [x] **Step 3: Run repository-wide verification**

Run the repository's available full suite from the checkout, including generated runtime assets if required by the current Make targets:

```bash
go test ./... -count=1
if test ! -e ../ariadne && test ! -L ../ariadne; then ln -s /Users/xianxu/workspace/ariadne ../ariadne; fi
test "$(cd ../ariadne && pwd -P)" = /Users/xianxu/workspace/ariadne
make test
git diff --check
```

Expected: PASS with no warnings attributable to the change. The guarded setup materializes the canonical sibling only when no path exists; it refuses an existing wrong/broken link instead of replacing it. This makes Pair's `Makefile -> ../ariadne/Makefile` and nested plain `make -C "$repo_root"` calls resolve normally. `make -n test` was verified after this setup in the planning checkout. If the canonical repo is unavailable, stop before testing.

- [x] **Step 4: Perform a shadow-sweep**

Run:

```bash
rg -n 'CodexSessionIDFromPath|rolloutRE|endUUIDRE|live_codex_session_id|\.codex/sessions|transcript(pkg)?\.SessionID|transcript(pkg)?\.Resolve|sessionFromConfig|session_id' cmd nvim --glob '*.go' --glob '*.lua'
```

Expected: every path that authorizes automatic identity reaches the shared root classifier; no Neovim or package-local filename-only authorizer remains. Low-level path-shape tests may remain only in `transcript`.

- [x] **Step 5: Record evidence and check every issue-plan box**

Append TDD red/green commands, focused/full verification, shadow-sweep result, and atlas disposition to `## Log`; tick all issue and durable-plan checkboxes. Do not hand-edit issue status.

- [x] **Step 6: Commit documentation and verification record**

```bash
git add atlas/session-identity.md workshop/issues/000144-reject-codex-subagent-sessions-during-pair-identity-discovery.md workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan.md
git commit -m "workshop: #144: record root session verification" -m "Co-Authored-By: OpenAI Codex <codex@openai.com>"
```

- [ ] **Step 7: Close through the SDLC boundary**

Run:

```bash
sdlc close --issue 144 --verified '<focused tests, full suite, headless Neovim regression, shadow-sweep, and atlas evidence>'
```

Expected: mandatory fresh-context review passes after any Critical/Important findings are fixed; close records measured actual time and moves the issue to `codecomplete`.

## Revisions

### 2026-08-19 07:29 PDT — Fresh-context plan review

- Expanded the consumer sweep to config-backed context, slug, and review-target
  identity, including validated `transcript.SessionID` and a broader shadow
  sweep.
- Consolidated `ps`/`lsof` parsing and traversal in `procutil` instead of
  deferring known duplication.
- Added exact 1 MiB reader-boundary/error tests and separated pure automatic
  resume decisions from caller-owned validation/removal/warning IO.
- Replaced the broken worktree-relative `make test` path with the canonical
  Makefile invocation and added required co-author trailers to every commit.

### 2026-08-19 07:34 PDT — Worktree Makefile validation

- Replaced the still-insufficient absolute `make -f` invocation with a guarded
  sibling-repo symlink setup. Verified `make -n test` resolves the full suite,
  including nested plain `make` calls, from the temporary Pair checkout.
---
gate: plan-quality
issue: 144
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-19T07:24:49-07:00"
      agent: codex
      blocked: true
      protocol_error: no valid findings block
    - "n": 2
      timestamp: "2026-08-19T07:24:57-07:00"
      agent: claude
      blocked: true
      protocol_error: no valid findings block
---

# Gate ledger — pair-context-usage#144 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-19T07:24:49-07:00 (codex) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 2 — 2026-08-19T07:24:57-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Open findings

(none — every finding has been disposed)

codex
The durable gate has no carried findings. One plan-table mismatch is already visible—the planned pure entity is named/exported and located differently from the implementation—so I’m checking whether it is merely ledger drift or accompanies a functional gap before assigning severity.
exec
/bin/zsh -lc "git diff --check 356ec01f5aaf6c48d3fcf73a7aff1bd090c8dc79..HEAD; git diff --name-status 356ec01f5aaf6c48d3fcf73a7aff1bd090c8dc79..HEAD; sed -n '1,180p' cmd/internal/transcript/transcript.go; sed -n '1,180p' cmd/internal/procutil/procutil.go; sed -n '1,360p' cmd/internal/sessionwatch/run.go; sed -n '1,160p' cmd/internal/sessionwatch/spec.go 2>/dev/null || true; rg -n \"func \\(.*AgentSessionExists|AgentSessionExists\\(|decideAutomaticResumeConfig|planRestart|RunLaunch|savedConfig|readSavedConfig\" cmd/internal/launcher" in /private/tmp/pair-context-usage
 succeeded in 0ms:
M	atlas/session-identity.md
M	cmd/internal/codexsid/codexsid.go
M	cmd/internal/codexsid/codexsid_test.go
M	cmd/internal/contextcmd/contextcmd.go
M	cmd/internal/contextcmd/contextcmd_test.go
M	cmd/internal/launcher/createflow.go
M	cmd/internal/launcher/createflow_test.go
M	cmd/internal/launcher/markers.go
M	cmd/internal/launcher/markers_test.go
M	cmd/internal/launcher/osruntime.go
M	cmd/internal/launcher/osruntime_test.go
M	cmd/internal/reviewcmd/reviewcmd_test.go
M	cmd/internal/reviewcmd/run.go
M	cmd/internal/reviewcmd/run_test.go
M	cmd/internal/reviewcmd/runtime.go
M	cmd/internal/sessionwatch/run.go
M	cmd/internal/sessionwatch/run_test.go
M	cmd/internal/sessionwatch/runtime.go
M	cmd/internal/slugcmd/slugcmd.go
M	cmd/internal/slugcmd/slugcmd_test.go
M	cmd/internal/transcript/transcript.go
M	cmd/internal/transcript/transcript_test.go
M	nvim/init.lua
M	tests/pair-review-target-test.sh
M	tests/pair-session-watch-test.sh
M	tests/review-toggle-test.sh
M	workshop/issues/000144-reject-codex-subagent-sessions-during-pair-identity-discovery.md
A	workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan-gate.md
M	workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan.md
// Package transcript resolves an agent's on-disk session transcript path and
// the session id pair recorded for it. Single source for both pair-slug and
// pair-context (ARCH-DRY) — extracted from cmd/pair-slug/main.go.
package transcript

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const codexSessionMetaLineLimit = 1 << 20

// ClaudePathEncoder mirrors nvim's `cwd:gsub('[./]', '-')` for the
// ~/.claude/projects/<encoded-cwd>/ directory name.
var ClaudePathEncoder = strings.NewReplacer(".", "-", "/", "-")

var codexRolloutRE = regexp.MustCompile(`^(.*/\.codex/sessions/.*/rollout-.*([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\.jsonl)$`)

// CodexSessionIDFromPath extracts the native session id from a Codex rollout
// transcript path. It returns "" for non-Codex or malformed paths.
func CodexSessionIDFromPath(path string) string {
	m := codexRolloutRE.FindStringSubmatch(path)
	if len(m) < 3 {
		return ""
	}
	return m[2]
}

// CodexRootSessionID authorizes a root Codex rollout from its path and first
// JSONL event. Filename UUIDs identify candidates; session_meta establishes
// whether the candidate is the operator's resumable root session.
func CodexRootSessionID(path string, firstEvent []byte) string {
	sid := CodexSessionIDFromPath(path)
	if sid == "" {
		return ""
	}
	var event struct {
		Type    string `json:"type"`
		Payload struct {
			ID             string          `json:"id"`
			ParentThreadID *string         `json:"parent_thread_id"`
			Source         json.RawMessage `json:"source"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(firstEvent, &event); err != nil || event.Type != "session_meta" || event.Payload.ID != sid || event.Payload.ParentThreadID != nil {
		return ""
	}
	var source string
	if err := json.Unmarshal(event.Payload.Source, &source); err != nil || (source != "cli" && source != "exec") {
		return ""
	}
	return sid
}

// ReadCodexRootSessionID reads one bounded, newline-terminated JSONL event and
// delegates the semantic decision to CodexRootSessionID. It fails closed when
// the rollout is incomplete, oversized, unreadable, or not a root session.
func ReadCodexRootSessionID(path string) string {
	line, err := ReadFirstEvent(path)
	if err != nil {
		return ""
	}
	return CodexRootSessionID(path, line)
}

// ReadFirstEvent returns one bounded, newline-terminated JSONL event.
func ReadFirstEvent(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	line, err := bufio.NewReader(io.LimitReader(f, codexSessionMetaLineLimit+1)).ReadBytes('\n')
	if err != nil || len(line) > codexSessionMetaLineLimit {
		if err == nil {
			err = io.ErrShortBuffer
		}
		return nil, err
	}
	return line, nil
}

// SessionID reads the session id pair recorded for (tag, agent) in
// config-<tag>-<agent>.json (written by bin/pair / pair-session-watch).
func SessionID(dataDir, tag, agent, home string) string {
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
	if agent == "codex" {
		path := Resolve(agent, c.SessionID, "", home)
		if path == "" || ReadCodexRootSessionID(path) != c.SessionID {
			return ""
		}
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
	case "muse":
		matches, _ := filepath.Glob(filepath.Join(home, ".local", "share", "muse", "sessions", "*", "*", "*", sid, "session.jsonl"))
		for _, m := range matches {
			if !strings.Contains(m, string(filepath.Separator)+"subagent"+string(filepath.Separator)) {
				return m
			}
		}
		// Flat fallback: direct parent dir without date hierarchy (tests / future layout)
		candidate := filepath.Join(home, ".local", "share", "muse", "sessions", sid, "session.jsonl")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		return ""
	default: // claude
		return filepath.Join(home, ".claude", "projects", ClaudePathEncoder.Replace(cwd), sid+".jsonl")
	}
}
// Package procutil holds the tiny cross-runtime process primitives that every
// pair OSRuntime needs: liveness (kill -0) and the process command line
// (ps -p <pid> -o command=). Extracting them here keeps one source of truth as
// the Go-migration ports (#93) each grow an OSRuntime — sessionwatch and
// titlepoller today, the leaf orchestrators next.
package procutil

import (
	"os/exec"
	"strings"
)

// Alive reports whether pid names a live process (kill -0). An empty pid is
// never alive.
func Alive(pid string) bool {
	if pid == "" {
		return false
	}
	return exec.Command("kill", "-0", pid).Run() == nil
}

// Command returns pid's full command line via `ps -p <pid> -o command=`, trimmed
// of the trailing newline. Empty string on any error (dead pid, no ps, etc.) —
// callers treat "no argv" as "not our process".
func Command(pid string) string {
	if pid == "" {
		return ""
	}
	out, err := exec.Command("ps", "-p", pid, "-o", "command=").Output()
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(out), "\n")
}

func ProcessChildren() map[string][]string {
	out, err := exec.Command("ps", "-axo", "pid=,ppid=").Output()
	if err != nil {
		return nil
	}
	children := make(map[string][]string)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		children[fields[1]] = append(children[fields[1]], fields[0])
	}
	return children
}

func DescendantPIDs(root string, children map[string][]string) []string {
	if root == "" {
		return nil
	}
	var out []string
	seen := map[string]bool{root: true}
	queue := []string{root}
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		out = append(out, pid)
		for _, child := range children[pid] {
			if seen[child] {
				continue
			}
			seen[child] = true
			queue = append(queue, child)
		}
	}
	return out
}

func LsofNames(pid string) []string {
	out, err := exec.Command("lsof", "-p", pid, "-Fn").Output()
	if err != nil {
		return nil
	}
	var names []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "n") {
			names = append(names, line[1:])
		}
	}
	return names
}
package sessionwatch

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/transcript"
)

// isMuseSubagentPath reports whether p is inside a Muse subagent directory.
// Muse nests subagent sessions as …/<root-uuid>/subagent/<sub-uuid>/session.jsonl;
// only the root session is resumable via `muse resume <id>` (ARCH-DRY).
func isMuseSubagentPath(p string) bool {
	return strings.Contains(p, string(filepath.Separator)+"subagent"+string(filepath.Separator))
}

// Options are the watcher inputs after CLI/env resolution.
type Options struct {
	Agent    string
	Tag      string
	Cwd      string
	RepoRoot string
	RepoName string
	Args     []string
	Home     string
	DataDir  string
	PIDWait  time.Duration
	Timeout  time.Duration
	Poll     time.Duration
}

// Runtime is the IO boundary for the session watcher.
type Runtime interface {
	Now() time.Time
	Sleep(time.Duration)
	ReadFile(path string) ([]byte, error)
	ReadFirstLine(path string) ([]byte, error)
	ModTime(path string) (time.Time, error)
	BirthTime(path string) (time.Time, error)
	ListFiles(root string) ([]string, error)
	Descendants(root string) ([]string, error)
	LsofPaths(pid string) ([]string, error)
	ProcessAlive(pid string) bool
	AtomicWrite(path string, data []byte) error
	Log(outcome adapt.Outcome, detail string)
}

type sessionLedgerEntry struct {
	Agent      string    `json:"agent"`
	Args       []string  `json:"args"`
	SessionID  string    `json:"session_id"`
	Started    time.Time `json:"started"`
	LastActive time.Time `json:"last_active"`
	RepoRoot   string    `json:"repo_root"`
	RepoName   string    `json:"repo_name"`
}

// Run discovers the async agent session id and writes config-<tag>-<agent>.json.
func Run(opts Options, rt Runtime) error {
	spec, ok := SpecForAgent(opts.Agent, opts.Home)
	if !ok || opts.Tag == "" || opts.DataDir == "" {
		return nil
	}
	if opts.PIDWait <= 0 {
		opts.PIDWait = 2 * time.Second
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 60 * time.Second
	}
	if opts.Poll <= 0 {
		opts.Poll = 100 * time.Millisecond
	}
	repoRoot := opts.RepoRoot
	if repoRoot == "" {
		repoRoot = opts.Cwd
	}
	repoName := opts.RepoName
	if repoName == "" {
		repoName = filepath.Base(filepath.Clean(repoRoot))
	}

	watchStart := rt.Now()
	pidFile := filepath.Join(opts.DataDir, "agent-pid-"+opts.Tag)
	out := filepath.Join(opts.DataDir, "config-"+opts.Tag+"-"+opts.Agent+".json")

	pidDeadline := watchStart.Add(opts.PIDWait)
	for {
		if fresh, _ := freshPID(pidFile, watchStart, rt); fresh {
			break
		}
		if !rt.Now().Before(pidDeadline) {
			break
		}
		rt.Sleep(opts.Poll)
	}

	rootPID := ""
	agentStart := time.Time{}
	if fresh, mod := freshPID(pidFile, watchStart, rt); fresh {
		if data, err := rt.ReadFile(pidFile); err == nil {
			rootPID = strings.TrimSpace(string(data))
			agentStart = mod
		}
	}

	legacyExisting := map[string]bool{}
	if rootPID == "" {
		files, _ := rt.ListFiles(spec.WatchDir)
		for _, file := range files {
			legacyExisting[file] = true
		}
	}

	nmLogged := false
	deadline := watchStart.Add(opts.Timeout)
	for rt.Now().Before(deadline) {
		if rootPID != "" && !rt.ProcessAlive(rootPID) {
			return nil
		}

		result := discover(spec, rootPID, agentStart, legacyExisting, rt)
		if result.ID != "" {
			payload, err := ConfigJSON(ConfigPayload{
				Agent:     opts.Agent,
				Args:      StripResumeArgs(opts.Agent, opts.Args),
				SessionID: result.ID,
			})
			if err != nil {
				return err
			}
			if err := appendSessionLedger(rt, filepath.Join(opts.DataDir, "ledger-"+opts.Tag+".jsonl"), sessionLedgerEntry{
				Agent:      opts.Agent,
				Args:       StripResumeArgs(opts.Agent, opts.Args),
				SessionID:  result.ID,
				Started:    watchStart,
				LastActive: rt.Now(),
				RepoRoot:   repoRoot,
				RepoName:   repoName,
			}); err != nil {
				return err
			}
			if err := rt.AtomicWrite(out, payload); err != nil {
				return err
			}
			rt.Log(adapt.Fired, "session_id="+result.ID)
			return nil
		}
		if result.NearMiss && !nmLogged {
			rt.Log(adapt.NearMiss, "matched session file but no id extracted: "+filepath.Base(result.Path))
			nmLogged = true
		}

		rt.Sleep(opts.Poll)
	}

	rt.Log(adapt.Fail, "no session id within 60s deadline (agent="+opts.Agent+")")
	return nil
}

func appendSessionLedger(rt Runtime, path string, entry sessionLedgerEntry) error {
	raw := ""
	if existing, err := rt.ReadFile(path); err == nil {
		raw = string(existing)
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if raw != "" && !strings.HasSuffix(raw, "\n") {
		raw += "\n"
	}
	raw += string(line) + "\n"
	return rt.AtomicWrite(path, []byte(raw))
}

func freshPID(pidFile string, since time.Time, rt Runtime) (bool, time.Time) {
	mod, err := rt.ModTime(pidFile)
	if err != nil {
		return false, time.Time{}
	}
	return mod.Unix() >= since.Unix(), mod
}

func discover(spec AgentSpec, rootPID string, agentStart time.Time, legacyExisting map[string]bool, rt Runtime) SessionID {
	if rootPID != "" {
		nearMiss := SessionID{}
		pids, _ := rt.Descendants(rootPID)
		for _, pid := range pids {
			paths, _ := rt.LsofPaths(pid)
			for _, path := range paths {
				if spec.Agent == "muse" && isMuseSubagentPath(path) {
					continue
				}
				result := authorizeCandidate(spec, spec.Match(path), rt)
				if result.ID != "" {
					return result
				}
				if result.NearMiss && !nearMiss.NearMiss {
					nearMiss = result
				}
			}
		}
		if !agentStart.IsZero() {
			if result := discoverByBirth(spec, agentStart, rt); result.ID != "" {
				return result
			} else if result.NearMiss && !nearMiss.NearMiss {
				nearMiss = result
			}
		}
		return nearMiss
	}
	nearMiss := SessionID{}
	files, _ := rt.ListFiles(spec.WatchDir)
	for _, file := range files {
		if legacyExisting[file] {
			continue
		}
		if spec.Agent == "muse" && isMuseSubagentPath(file) {
			continue
		}
		result := authorizeCandidate(spec, spec.Match(file), rt)
		if result.ID != "" {
			return result
		}
		if result.NearMiss && !nearMiss.NearMiss {
			nearMiss = result
		}
	}
	return nearMiss
}

func discoverByBirth(spec AgentSpec, agentStart time.Time, rt Runtime) SessionID {
	files, _ := rt.ListFiles(spec.WatchDir)
	type cand struct {
		id    SessionID
		birth time.Time
	}
	var matched []cand
	var nearMiss *cand
	for _, file := range files {
		if spec.Agent == "muse" && isMuseSubagentPath(file) {
			continue
		}
		birth, err := rt.BirthTime(file)
		if err != nil || birth.Before(agentStart) {
			continue
		}
		result := authorizeCandidate(spec, spec.Match(file), rt)
		if !result.Matched {
			continue
		}
		c := cand{id: result, birth: birth}
		if result.NearMiss {
			if nearMiss == nil || birth.After(nearMiss.birth) {
				// Keep newest near-miss for drift signal, but don't return it
				// if a real ID exists — real IDs outrank near-misses.
				cp := c
				nearMiss = &cp
			}
			continue
		}
		matched = append(matched, c)
	}
	if len(matched) > 0 {
		// Pick newest by birth time — with concurrent sessions the birth
		// filter may yield >1 candidate; the freshest is the one we just
		// launched. The old "exactly 1" gate dropped the capture for muse
		// when multiple sessions shared the same birth second.
		best := matched[0]
		for _, c := range matched[1:] {
			if c.birth.After(best.birth) {
				best = c
			}
		}
		return best.id
	}
	if nearMiss != nil {
		return nearMiss.id
	}
	return SessionID{}
}

func authorizeCandidate(spec AgentSpec, result SessionID, rt Runtime) SessionID {
	if spec.Agent != "codex" || result.ID == "" {
		return result
	}
	firstEvent, err := rt.ReadFirstLine(result.Path)
	if err != nil || transcript.CodexRootSessionID(result.Path, firstEvent) != result.ID {
		return SessionID{}
	}
	return result
}
cmd/internal/launcher/markers_test.go:30:	saved := savedConfig{Agent: "claude", Args: []string{"--flag"}, SessionID: "SID-1"}
cmd/internal/launcher/markers_test.go:33:	// calling, so planRestart takes the FINAL tag/agent. Default Alt+n: resume the
cmd/internal/launcher/markers_test.go:35:	p := planRestart(RestartMarker{}, "work", "claude", saved)
cmd/internal/launcher/markers_test.go:46:	pm := planRestart(RestartMarker{SessionID: "SID-LIVE"}, "work", "codex", savedConfig{Agent: "codex", Args: []string{"--flag"}})
cmd/internal/launcher/markers_test.go:52:	pn := planRestart(RestartMarker{NewSession: true}, "work", "claude", saved)
cmd/internal/launcher/markers_test.go:62:	pc := planRestart(RestartMarker{NewSession: true, Continue: "demo-slug"}, "work", "claude", saved)
cmd/internal/launcher/markers_test.go:72:		saved        savedConfig
cmd/internal/launcher/markers_test.go:74:		want         savedConfig
cmd/internal/launcher/markers_test.go:77:		{"valid codex root", "codex", savedConfig{Agent: "codex", Args: []string{"--search"}, SessionID: "ROOT"}, true, savedConfig{Agent: "codex", Args: []string{"--search"}, SessionID: "ROOT"}, false},
cmd/internal/launcher/markers_test.go:78:		{"invalid codex candidate", "codex", savedConfig{Agent: "codex", Args: []string{"--search"}, SessionID: "SUB"}, false, savedConfig{Agent: "codex", Args: []string{"--search"}}, true},
cmd/internal/launcher/markers_test.go:79:		{"empty codex session", "codex", savedConfig{Agent: "codex", Args: []string{"--search"}}, false, savedConfig{Agent: "codex", Args: []string{"--search"}}, false},
cmd/internal/launcher/markers_test.go:80:		{"non-codex unchanged", "claude", savedConfig{Agent: "claude", Args: []string{"--flag"}, SessionID: "STALE"}, false, savedConfig{Agent: "claude", Args: []string{"--flag"}, SessionID: "STALE"}, false},
cmd/internal/launcher/markers_test.go:84:			got, quarantine := decideAutomaticResumeConfig(tt.agent, tt.saved, tt.sessionValid)
cmd/internal/launcher/layoutflow_test.go:10:func TestRunLaunchUnrecordedTagDefaultsToLayout2(t *testing.T) {
cmd/internal/launcher/layoutflow_test.go:27:func TestRunLaunchImplicitlyReusesRecordedLayout3(t *testing.T) {
cmd/internal/launcher/layoutflow_test.go:39:func TestRunLaunchExplicitLayoutWinsOnCreate(t *testing.T) {
cmd/internal/launcher/layoutflow_test.go:59:func TestRunLaunchLayoutRecordWriteFailureAbortsBeforeHandoff(t *testing.T) {
cmd/internal/launcher/layoutflow_test.go:71:func TestRunLaunchImmediateErrorRestoresPriorLayoutRecord(t *testing.T) {
cmd/internal/launcher/layoutflow_test.go:104:func TestRunLaunchFailedPreflightDoesNotWriteLayoutRecord(t *testing.T) {
cmd/internal/launcher/layoutflow_test.go:197:func TestRunLaunchImplicitLiveLayoutAttachesWithoutPrompt(t *testing.T) {
cmd/internal/launcher/layoutflow_test.go:208:func TestRunLaunchExplicitSameLiveLayoutAttachesWithoutPrompt(t *testing.T) {
cmd/internal/launcher/layoutflow_test.go:220:func TestRunLaunchDeclinedLiveLayoutChangeIsInert(t *testing.T) {
cmd/internal/launcher/layoutflow_test.go:235:func TestRunLaunchConfirmedLiveLayoutChangeRelaunchesSameTag(t *testing.T) {
cmd/internal/launcher/layoutflow_test.go:254:func TestRunLaunchPreRecordLiveSessionUsesProbe(t *testing.T) {
cmd/internal/launcher/layoutflow_test.go:267:func TestRunLaunchImplicitPreRecordSessionPersistsDetectedLayout(t *testing.T) {
cmd/internal/launcher/layoutflow_test.go:282:func TestRunLaunchLiveProbeFailureAbortsExplicitOverride(t *testing.T) {
cmd/internal/launcher/layoutflow_test.go:295:func TestRunLaunchLiveLayoutDeleteFailureAborts(t *testing.T) {
cmd/internal/launcher/layoutflow_test.go:309:func TestRunLaunchLiveLayoutStillPresentAborts(t *testing.T) {
cmd/internal/launcher/runcli.go:17:// and drives RunLaunch (create / attach / pick / restart loop / compaction). Every
cmd/internal/launcher/runcli.go:125:	return RunLaunch(opts, rt, stderr)
cmd/internal/launcher/createflow.go:12:// RunLaunch is the native launcher's in-process driver (#99 M2 create + M3
cmd/internal/launcher/createflow.go:23:func RunLaunch(opts LaunchOptions, rt Runtime, stderr io.Writer) (int, error) {
cmd/internal/launcher/createflow.go:31:	// RunLaunch is the sole zellij-spawning path (create/attach/resurrect/restart-
cmd/internal/launcher/createflow.go:99:		saved := readSavedConfig(rt, configPath)
cmd/internal/launcher/createflow.go:102:		saved, quarantine = decideAutomaticResumeConfig(rAgent, saved, rt.AgentSessionExists(rAgent, savedSessionID, env.Cwd))
cmd/internal/launcher/createflow.go:107:		plan := planRestart(m, rTag, rAgent, saved)
cmd/internal/launcher/createflow.go:378:	savedForPicker, savedWarnings := readSavedConfigForTag(rt, configPath, chosenTag, agent)
cmd/internal/launcher/createflow.go:413:			if cand != "" && !rt.AgentSessionExists("claude", cand, env.Cwd) {
cmd/internal/launcher/createflow.go:646:func runConfigPicker(rt Runtime, configPath string, saved savedConfig, agent, chosenTag string, agentArgs *[]string, cwd string, stderr io.Writer) (code int, ok bool) {
cmd/internal/launcher/createflow.go:655:	hasResumable := rt.AgentSessionExists(agent, savedSessionID, cwd)
cmd/internal/launcher/createflow.go:657:	saved, quarantine = decideAutomaticResumeConfig(agent, saved, hasResumable)
cmd/internal/launcher/createflow.go:685:func readSavedConfigForTag(rt Runtime, configPath, tag, agent string) (savedConfig, []string) {
cmd/internal/launcher/createflow.go:698:		return savedConfig{}, warnings
cmd/internal/launcher/createflow.go:701:		return savedConfig{Agent: latest.Agent, Args: latest.Args, SessionID: latest.SessionID}, warnings
cmd/internal/launcher/createflow.go:703:	return savedConfig{}, warnings
cmd/internal/launcher/pick_test.go:155:func TestRunLaunchPickLegacyImportsFlatFiles(t *testing.T) {
cmd/internal/launcher/pick_test.go:190:func TestRunLaunchPickAttachInfersAgent(t *testing.T) {
cmd/internal/launcher/pick_test.go:228:func TestRunLaunchPickNewCreates(t *testing.T) {
cmd/internal/launcher/pick_test.go:248:func TestRunLaunchPickHistoricalCreatesByName(t *testing.T) {
cmd/internal/launcher/pick_test.go:272:func TestRunLaunchExplicitAgentDifferentHistoricalUsesContinuation(t *testing.T) {
cmd/internal/launcher/pick_test.go:296:func TestRunLaunchExplicitAgentDifferentHistoricalSynthesizesContinuationWhenDocMissing(t *testing.T) {
cmd/internal/launcher/pick_test.go:306:	code, err := RunLaunch(opts, rt, &stderr)
cmd/internal/launcher/pick_test.go:334:func TestRunLaunchPickAbort(t *testing.T) {
cmd/internal/launcher/osruntime_test.go:345:	if !(OSRuntime{}).AgentSessionExists("codex", sid, "/repo") {
cmd/internal/launcher/osruntime_test.go:346:		t.Fatal("AgentSessionExists(codex) did not find nested rollout file")
cmd/internal/launcher/launch_args_policy_test.go:15:		Saved:        savedConfig{Agent: "codex", Args: []string{"--saved"}, SessionID: "SID"},
cmd/internal/launcher/launch_args_policy_test.go:48:		Saved:        savedConfig{Agent: "claude", Args: []string{"--saved"}, SessionID: "SID"},
cmd/internal/launcher/launch_args_policy_test.go:75:		Saved:        savedConfig{Agent: "codex", Args: []string{"resume", "OLD", "--search"}, SessionID: "OLD"},
cmd/internal/launcher/osruntime.go:268:	return line, true // an empty line falls through to the default in RunLaunch.
cmd/internal/launcher/osruntime.go:591:func (OSRuntime) AgentSessionExists(agent, sid, cwd string) bool {
cmd/internal/launcher/runtime.go:12:// RunLaunch is driven by a fake in tests and the pure deciders (createlogic.go +
cmd/internal/launcher/runtime.go:120:	AgentSessionExists(agent, sid, cwd string) bool
cmd/internal/launcher/runtime.go:246:// LaunchOptions are RunLaunch's post-parse inputs — the resolved argv (Args),
cmd/internal/launcher/markers.go:59:// decideAutomaticResumeConfig rejects only persisted Codex bindings that no
cmd/internal/launcher/markers.go:62:func decideAutomaticResumeConfig(agent string, saved savedConfig, sessionValid bool) (savedConfig, bool) {
cmd/internal/launcher/markers.go:70:// planRestart maps a restart marker + the RESOLVED (tag, agent) + saved config
cmd/internal/launcher/markers.go:77:func planRestart(m RestartMarker, tag, agent string, saved savedConfig) restartPlan {
cmd/internal/launcher/compaction_test.go:62:func TestRunLaunchCompactionForced(t *testing.T) {
cmd/internal/launcher/compaction_test.go:87:func TestRunLaunchCompactionUsesScopedPublicSession(t *testing.T) {
cmd/internal/launcher/compaction_test.go:111:func TestRunLaunchCompactionTagMatch(t *testing.T) {
cmd/internal/launcher/compaction_test.go:124:func TestRunLaunchCompactionTagMismatch(t *testing.T) {
cmd/internal/launcher/compaction.go:14:// RunLaunch loop then re-launches fresh under the same tag. The decision + marker
cmd/internal/launcher/run_test.go:33:func TestRunLaunchUsesSuppliedEnvironment(t *testing.T) {
cmd/internal/launcher/run_test.go:51:func TestRunLaunchTurnsFakeSessionsIntoPickerDecision(t *testing.T) {
cmd/internal/launcher/run_test.go:66:func TestRunLaunchTurnsFakeHistoryIntoPickerDecision(t *testing.T) {
cmd/internal/launcher/run_test.go:81:func TestRunLaunchReturnsTypedUsageError(t *testing.T) {
cmd/internal/launcher/launch_args_policy.go:8:	Saved        savedConfig
cmd/internal/launcher/launch_args_policy.go:39:	} else if savedConfigUsable(in.Agent, in.Saved) {
cmd/internal/launcher/launch_args_policy.go:54:	if savedConfigUsable(in.Agent, in.Saved) && in.Saved.SessionID != "" && len(resumeToken(in.Agent, in.Saved.SessionID)) > 0 {
cmd/internal/launcher/launch_args_policy.go:67:func savedConfigUsable(agent string, saved savedConfig) bool {
cmd/internal/launcher/createflow_test.go:18:// fakeRuntime is the in-memory create-flow seam for the RunLaunch loop tests.
cmd/internal/launcher/createflow_test.go:212:func (f *fakeRuntime) AgentSessionExists(agent, sid, cwd string) bool {
cmd/internal/launcher/createflow_test.go:409:	code, err := RunLaunch(opts, rt, &stderr)
cmd/internal/launcher/createflow_test.go:416:// RunLaunch must front the resolved asset root's bin/ on PATH at entry (#95),
cmd/internal/launcher/createflow_test.go:422:func TestRunLaunchPrependsBinToPath(t *testing.T) {
cmd/internal/launcher/createflow_test.go:436:		t.Fatalf("RunLaunch did not front the asset-root bin/ on PATH: %q", got)
cmd/internal/launcher/createflow_test.go:442:func TestRunLaunchForcedCreateClaude(t *testing.T) {
cmd/internal/launcher/createflow_test.go:502:func TestRunLaunchForcedCreateUsesScopedSessionName(t *testing.T) {
cmd/internal/launcher/createflow_test.go:524:func TestRunLaunchPromptCreate(t *testing.T) {
cmd/internal/launcher/createflow_test.go:548:func TestRunLaunchBareIgnoresOtherRepoIndexedSessions(t *testing.T) {
cmd/internal/launcher/createflow_test.go:569:func TestRunLaunchBareIgnoresUnindexedLiveSessions(t *testing.T) {
cmd/internal/launcher/createflow_test.go:589:func TestRunLaunchBareAttachesLegacyLiveSessionWithCurrentRepoPaneEvidence(t *testing.T) {
cmd/internal/launcher/createflow_test.go:619:func TestRunLaunchResumePublicSessionNameResolvesThroughIndex(t *testing.T) {
cmd/internal/launcher/createflow_test.go:645:func TestRunLaunchResumeUnindexedPublicSessionNameRefuses(t *testing.T) {
cmd/internal/launcher/createflow_test.go:648:	code, err := RunLaunch(baseOpts(LaunchArgs{ForcedTag: "📁work-demo"}), rt, &stderr)
cmd/internal/launcher/createflow_test.go:661:func TestRunLaunchPromptAbort(t *testing.T) {
cmd/internal/launcher/createflow_test.go:677:func TestRunLaunchPromptCollision(t *testing.T) {
cmd/internal/launcher/createflow_test.go:690:func TestRunLaunchPromptRefusesNameOverDiscoveredBudget(t *testing.T) {
cmd/internal/launcher/createflow_test.go:696:	code, err := RunLaunch(baseOpts(LaunchArgs{Agent: "claude"}), rt, &stderr)
cmd/internal/launcher/createflow_test.go:710:func TestRunLaunchPromptAcceptsNameUnderDiscoveredBudget(t *testing.T) {
cmd/internal/launcher/createflow_test.go:716:	code, err := RunLaunch(baseOpts(LaunchArgs{Agent: "claude"}), rt, &stderr)
cmd/internal/launcher/createflow_test.go:725:func TestRunLaunchFailedPreflightDoesNotAppendLedgerOrSessionIndex(t *testing.T) {
cmd/internal/launcher/createflow_test.go:745:	if len(rt.env) != 1 { // PATH is set at RunLaunch entry.
cmd/internal/launcher/createflow_test.go:750:func TestRunLaunchLedgerAppendFailureAbortsBeforeHandoff(t *testing.T) {
cmd/internal/launcher/createflow_test.go:772:func TestRunLaunchSessionIndexAppendFailureAbortsBeforeHandoff(t *testing.T) {
cmd/internal/launcher/createflow_test.go:797:func TestRunLaunchPromptedTagIgnoresUnrelatedLegacySessionName(t *testing.T) {
cmd/internal/launcher/createflow_test.go:812:func TestRunLaunchCodexAltScreen(t *testing.T) {
cmd/internal/launcher/createflow_test.go:830:func TestRunLaunchUsesRepoAgentDefaultWhenNoTagConfig(t *testing.T) {
cmd/internal/launcher/createflow_test.go:847:func TestRunLaunchIgnoresMismatchedTagConfigWithWarning(t *testing.T) {
cmd/internal/launcher/createflow_test.go:862:	code, err := RunLaunch(baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"}), rt, &stderr)
cmd/internal/launcher/createflow_test.go:877:func TestRunLaunchLayoutOnlyNewPickUsesRepoAgentDefault(t *testing.T) {
cmd/internal/launcher/createflow_test.go:906:func TestRunLaunchExplicitArgsPersistRepoAgentDefaultAfterReadiness(t *testing.T) {
cmd/internal/launcher/createflow_test.go:931:func TestRunLaunchEmptyExplicitArgsPersistEmptyRepoAgentDefaultAfterReadiness(t *testing.T) {
cmd/internal/launcher/createflow_test.go:957:func TestRunLaunchExplicitArgsDoNotPersistRepoDefaultOnReadinessTimeout(t *testing.T) {
cmd/internal/launcher/createflow_test.go:976:func TestRunLaunchExplicitArgsDoNotPersistRepoDefaultOnPreLaunchAbort(t *testing.T) {
cmd/internal/launcher/createflow_test.go:997:func TestRunLaunchTagRestartPickerResume(t *testing.T) {
cmd/internal/launcher/createflow_test.go:1014:func TestRunLaunchTagRestartPickerResumeStripsCodexResumeAfterGlobals(t *testing.T) {
cmd/internal/launcher/createflow_test.go:1031:func TestRunLaunchTagRestartPickerWarnsWhenSavedSessionIsStale(t *testing.T) {
cmd/internal/launcher/createflow_test.go:1044:	code, err := RunLaunch(baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"}), rt, &stderr)
cmd/internal/launcher/createflow_test.go:1060:func TestRunLaunchTagRestartPickerNew(t *testing.T) {
cmd/internal/launcher/createflow_test.go:1087:func TestRunLaunchTagRestartPickerAbort(t *testing.T) {
cmd/internal/launcher/createflow_test.go:1101:func TestRunLaunchExplicitResumeSkipsPicker(t *testing.T) {
cmd/internal/launcher/createflow_test.go:1127:func TestRunLaunchSessionsErrorExits(t *testing.T) {
cmd/internal/launcher/createflow_test.go:1143:func TestRunLaunchInPaneRejected(t *testing.T) {
cmd/internal/launcher/createflow_test.go:1159:func TestRunLaunchAgentMissing(t *testing.T) {
cmd/internal/launcher/createflow_test.go:1172:func TestRunLaunchResumeInfersAgent(t *testing.T) {
cmd/internal/launcher/createflow_test.go:1187:func TestRunLaunchResumeUsesLedgerAgentAndArgsWhenConfigMissing(t *testing.T) {
cmd/internal/launcher/createflow_test.go:1217:func TestRunLaunchRejectsInvalidLedgerCodexSession(t *testing.T) {
cmd/internal/launcher/createflow_test.go:1235:	code, err := RunLaunch(baseOpts(LaunchArgs{ForcedTag: "work"}), rt, &stderr)
cmd/internal/launcher/createflow_test.go:1253:func TestRunLaunchAltNRestartRejectsInvalidSavedCodexSession(t *testing.T) {
cmd/internal/launcher/createflow_test.go:1261:	code, err := RunLaunch(opts, rt, &stderr)
cmd/internal/launcher/createflow_test.go:1280:func TestRunLaunchResumeDefaultsClaude(t *testing.T) {
cmd/internal/launcher/createflow_test.go:1293:func TestRunLaunchProbeTooLong(t *testing.T) {
cmd/internal/launcher/createflow_test.go:1308:func TestRunLaunchPreHandoffCollision(t *testing.T) {
cmd/internal/launcher/createflow_test.go:1339:func TestRunLaunchPickInferredAgentMustNotInheritCliArgs(t *testing.T) {
cmd/internal/launcher/createflow_test.go:1365:func TestRunLaunchPickNewDefaultUsesScopedNextFreeSessionName(t *testing.T) {
cmd/internal/launcher/lifecycle_test.go:11:func TestRunLaunchAttach(t *testing.T) {
cmd/internal/launcher/lifecycle_test.go:55:func TestRunLaunchQuitCleanup(t *testing.T) {
cmd/internal/launcher/lifecycle_test.go:68:	code, err := RunLaunch(baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt, &stderr)
cmd/internal/launcher/lifecycle_test.go:117:func TestRunLaunchDetachNoCleanup(t *testing.T) {
cmd/internal/launcher/lifecycle_test.go:134:func TestRunLaunchParkSkippedOnRestart(t *testing.T) {
cmd/internal/launcher/lifecycle_test.go:154:func TestRunLaunchRestartLoopAltN(t *testing.T) {
cmd/internal/launcher/lifecycle_test.go:175:func TestRunLaunchRestartLoopAltNCodexUsesMarkerSessionID(t *testing.T) {
cmd/internal/launcher/lifecycle_test.go:197:func TestRunLaunchRestartLoopNewSession(t *testing.T) {
cmd/internal/launcher/lifecycle_test.go:223:func TestRunLaunchRenameReentry(t *testing.T) {
cmd/internal/launcher/lifecycle_test.go:248:func TestRunLaunchRenameReentryIgnoresOtherScopeTargetTag(t *testing.T) {
cmd/internal/launcher/lifecycle_test.go:277:func TestRunLaunchContinueReentry(t *testing.T) {
cmd/internal/launcher/lifecycle_test.go:300:func TestRunLaunchSweepsOnce(t *testing.T) {
cmd/internal/launcher/createlogic.go:10:// Pure create-flow logic behind RunLaunch's create path (#99 M2, ported from
cmd/internal/launcher/createlogic.go:13:// uuid mint, agent-session stat) sits on the Runtime seam. RunLaunch stays a thin
cmd/internal/launcher/createlogic.go:16:// savedConfig is the on-disk config-<tag>-<agent>.json shape the launcher and the
cmd/internal/launcher/createlogic.go:18:type savedConfig struct {
cmd/internal/launcher/createlogic.go:26:func parseConfig(raw string) (savedConfig, error) {
cmd/internal/launcher/createlogic.go:27:	var c savedConfig
cmd/internal/launcher/createlogic.go:44:	if err := enc.Encode(savedConfig{Agent: agent, Args: args, SessionID: sid}); err != nil {
cmd/internal/launcher/lifecycle.go:10:// The attach + quit-cleanup orchestrators behind RunLaunch's in-process restart
cmd/internal/launcher/lifecycle.go:137:// readSavedConfig loads config-<tag>-<agent>.json for the restart plan; a
cmd/internal/launcher/lifecycle.go:138:// missing/unusable file yields the zero savedConfig (no resume, no saved args).
cmd/internal/launcher/lifecycle.go:139:func readSavedConfig(rt Runtime, configPath string) savedConfig {
cmd/internal/launcher/lifecycle.go:142:		return savedConfig{}

exec
/bin/zsh -lc "rg -n \"SubagentOnly|subagent.only|Polluted|SavedCodexRoot|ExplicitCodexResume|ReadFirstLine|ReadFirstEvent|ReadCodexRoot|CodexSessionIDFromPath|\\.codex/sessions|session_id\" cmd nvim --glob '*.go' --glob '*.lua'; git diff 356ec01f5aaf6c48d3fcf73a7aff1bd090c8dc79..HEAD -- cmd/internal/launcher/createflow.go cmd/internal/launcher/lifecycle.go cmd/internal/launcher/markers.go cmd/internal/launcher/osruntime.go; sed -n '1,130p' cmd/internal/launcher/markers.go; sed -n '580,635p' cmd/internal/launcher/osruntime.go; sed -n '1,130p' cmd/internal/sessionwatch/sessionwatch.go" in /private/tmp/pair-context-usage
 succeeded in 0ms:
cmd/pair-go/helper_equivalence_test.go:39:	// Empty data dir → no config → slugcmd.Run resolves no session_id and no-ops
cmd/pair-go/helper_equivalence_test.go:97:	mustWrite(t, filepath.Join(data, "config-T-claude.json"), `{"session_id":"sid1"}`)
cmd/internal/codexsid/codexsid.go:32:			if sid := transcript.ReadCodexRootSessionID(name); sid != "" {
nvim/init.lua:831:  local function config_session_id(data_dir, tag, agent)
nvim/init.lua:836:    if ok and type(parsed) == 'table' and parsed.session_id and parsed.session_id ~= '' then
nvim/init.lua:837:      return parsed.session_id
nvim/init.lua:842:  local function current_session_id()
nvim/init.lua:849:    sid = config_session_id(data_dir, tag, agent)
nvim/init.lua:877:    if target_stale(t, current_session_id()) then
nvim/init.lua:890:      { vim.json.encode({ file = file, status = status, session = current_session_id() or '' }) }, p) end
nvim/init.lua:931:    current_session_id = current_session_id }
nvim/init.lua:2980:local function pair_changelog_session_id(data_dir, tag, agent)
nvim/init.lua:2987:  if ok and type(parsed) == 'table' and parsed.session_id and parsed.session_id ~= '' then
nvim/init.lua:2988:    return parsed.session_id
nvim/init.lua:3000:    local sid = pair_changelog_session_id(data_dir, tag, agent)
nvim/init.lua:3104:    local cmd = 'find ' .. vim.fn.shellescape(home .. '/.codex/sessions')
nvim/init.lua:3148:      cfg.session_id = parsed.session_id
nvim/init.lua:3165:      local sid_line = cfg.session_id and cfg.session_id ~= '' and cfg.session_id or '<not captured>'
nvim/init.lua:3166:      local age = session_age_hint(cfg.agent, cfg.session_id)
nvim/init.lua:3263:      if not new_session and cfg.session_id and cfg.session_id ~= '' then
nvim/init.lua:3264:        local resume_line = cfg.session_id
nvim/init.lua:3265:        local age = session_age_hint(cfg.agent, cfg.session_id)
cmd/internal/opener/opener_test.go:65:	if got := resolveSessionID(A, []byte(`{"session_id":"`+C+`"}`)); got != A {
cmd/internal/opener/opener_test.go:69:	if got := resolveSessionID("", []byte(`{"agent":"claude","session_id":"`+C+`"}`)); got != C {
cmd/internal/opener/opener_test.go:77:		t.Fatalf("config without session_id → empty: %q", got)
cmd/internal/opener/run_test.go:203:	rt.files["/dd/config-t-claude.json"] = `{"agent":"claude","session_id":"cfgsid"}`
cmd/internal/opener/opener.go:103:// PAIR_SESSION_ID wins; else the per-tag config's session_id; else "" (the
cmd/internal/opener/opener.go:114:		SessionID string `json:"session_id"`
cmd/internal/reviewcmd/run.go:110:// PAIR_SESSION_ID → validated config session_id → (codex only) validated live
cmd/internal/slugcmd/slugcmd.go:6:// from $PAIR_DATA_DIR/config-<tag>-<agent>.json (session_id) + the per-agent
cmd/internal/slugcmd/slugcmd.go:94:			if strings.HasPrefix(name, prefix) && transcriptpkg.ReadCodexRootSessionID(name) != "" {
cmd/internal/slugcmd/slugcmd.go:145:			logf("no session_id in config-%s-%s.json", tag, agent)
cmd/internal/reviewcmd/reviewcmd_test.go:53:	if err := os.WriteFile(filepath.Join(data, "config-t-codex.json"), []byte(`{"session_id":"`+sid+`"}`), 0o644); err != nil {
cmd/internal/continuationcmd/continuation.go:39:		fmt.Fprintf(&b, "session_id: %s\n", f.SessionID)
cmd/internal/slugcmd/slugcmd_test.go:12:	path := "/Users/x/.codex/sessions/2026/05/31/rollout-2026-05-31T21-36-56-019e8178-79c2-7862-91db-e8fa1be3b162.jsonl"
cmd/internal/slugcmd/slugcmd_test.go:13:	if got := transcript.CodexSessionIDFromPath(path); got == "" {
cmd/internal/slugcmd/slugcmd_test.go:14:		t.Fatalf("CodexSessionIDFromPath did not match %q", path)
cmd/internal/contextcmd/contextcmd_test.go:20:	mustWrite(t, filepath.Join(data, "config-T-claude.json"), `{"session_id":"sid1"}`)
cmd/internal/contextcmd/contextcmd_test.go:47:func TestRunCodexPollutedSubagentConfigPrintsNothing(t *testing.T) {
cmd/internal/contextcmd/contextcmd_test.go:55:	mustWrite(t, filepath.Join(data, "config-T-codex.json"), `{"session_id":"`+sid+`"}`)
cmd/internal/continuationcmd/continuation_test.go:17:		"session_id: 7f3a\ncreated: 2026-06-11T14:20:00\nbranch: main\n" +
cmd/internal/continuationcmd/continuation_test.go:30:	for _, k := range []string{"session_id:", "supersedes:", "branch:", "worktree:"} {
cmd/internal/launcher/createlogic.go:17:// session-watcher both write ({agent, args, session_id}).
cmd/internal/launcher/createlogic.go:21:	SessionID string   `json:"session_id"`
cmd/internal/launcher/createlogic.go:32:// buildConfigJSON renders the {agent, args, session_id} config the shell wrote
cmd/internal/launcher/createlogic.go:36:// order (agent, args, session_id) mirrors the jq object literal.
cmd/internal/sessionwatch/sessionwatch.go:35:	SessionID string   `json:"session_id"`
cmd/internal/sessionwatch/sessionwatch.go:123:// the session_id field is the canonical store for that binding.
cmd/internal/launcher/agentargs.go:202:// they are saved to config-<tag>-<agent>.json: session_id is the canonical
cmd/internal/launcher/restart.go:9:// optional rename_to/session_id), touch the quit marker, then exec kill-session.
cmd/internal/launcher/createflow.go:735:// writeConfig persists {agent, args, session_id}; a serialization failure leaves
cmd/internal/launcher/markers_test.go:9:	got := parseRestartMarker("tag=work\nagent=codex\nnew_session=1\nsession_id=SID-LIVE\n")
cmd/internal/launcher/ledger.go:13:	SessionID    string    `json:"session_id"`
cmd/internal/launcher/lifecycle_test.go:64:	// config-bugfix-claude.json (session_id SID) that drives the resume hint.
cmd/internal/sessionwatch/run_test.go:17:	sessionFile := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
cmd/internal/sessionwatch/run_test.go:45:	if !strings.Contains(got, `"session_id":"`+sid+`"`) || strings.Contains(got, "old") || !strings.Contains(got, `say \"hi\"`) {
cmd/internal/sessionwatch/run_test.go:48:	if !rt.hasLog(adapt.Fired, "session_id="+sid) {
cmd/internal/sessionwatch/run_test.go:52:	if !strings.Contains(ledger, `"agent":"codex"`) || !strings.Contains(ledger, `"session_id":"`+sid+`"`) || !strings.Contains(ledger, `"repo_root":"/repo"`) {
cmd/internal/sessionwatch/run_test.go:64:	sessionFile := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
cmd/internal/sessionwatch/run_test.go:101:	sessionFile := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
cmd/internal/sessionwatch/run_test.go:132:	sessionFile := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
cmd/internal/sessionwatch/run_test.go:185:	if !strings.Contains(got, `"session_id":"`+sid+`"`) || !strings.Contains(got, "--conversation") {
cmd/internal/sessionwatch/run_test.go:193:	bad := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-not-a-uuid.jsonl"
cmd/internal/sessionwatch/run_test.go:226:	bad := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-not-a-uuid.jsonl"
cmd/internal/sessionwatch/run_test.go:227:	good := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
cmd/internal/sessionwatch/run_test.go:262:	bad := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-not-a-uuid.jsonl"
cmd/internal/sessionwatch/run_test.go:263:	good := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
cmd/internal/sessionwatch/run_test.go:298:	rootPath := home + "/.codex/sessions/2026/08/18/rollout-root-" + rootSID + ".jsonl"
cmd/internal/sessionwatch/run_test.go:299:	subPath := home + "/.codex/sessions/2026/08/18/rollout-sub-" + subSID + ".jsonl"
cmd/internal/sessionwatch/run_test.go:322:	rootPath := home + "/.codex/sessions/2026/08/18/rollout-root-" + rootSID + ".jsonl"
cmd/internal/sessionwatch/run_test.go:323:	subPath := home + "/.codex/sessions/2026/08/18/rollout-sub-" + subSID + ".jsonl"
cmd/internal/sessionwatch/run_test.go:339:func TestRunCodexSubagentOnlyWritesNoConfig(t *testing.T) {
cmd/internal/sessionwatch/run_test.go:344:	subPath := home + "/.codex/sessions/2026/08/18/rollout-sub-" + subSID + ".jsonl"
cmd/internal/sessionwatch/run_test.go:356:		t.Fatalf("subagent-only config = %s, want none", got)
cmd/internal/sessionwatch/run_test.go:436:func (f *fakeRuntime) ReadFirstLine(path string) ([]byte, error) {
cmd/internal/launcher/osruntime.go:603:		return path != "" && transcript.ReadCodexRootSessionID(path) == sid
cmd/internal/launcher/osruntime.go:628:			if sid := transcript.ReadCodexRootSessionID(name); sid != "" {
cmd/internal/launcher/markers.go:37:		case "session_id":
cmd/internal/launcher/compaction.go:54:		fmt.Fprintf(&b, "session_id=%s\n", m.SessionID)
cmd/internal/sessionwatch/sessionwatch_test.go:12:	path := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
cmd/internal/sessionwatch/sessionwatch_test.go:41:	path := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-not-a-uuid.jsonl"
cmd/internal/sessionwatch/runtime.go:32:func (OSRuntime) ReadFirstLine(path string) ([]byte, error) {
cmd/internal/sessionwatch/runtime.go:33:	return transcript.ReadFirstEvent(path)
cmd/internal/launcher/createlogic_test.go:41:	want := `{"agent":"claude","args":["--model","opus"],"session_id":"sid-abc"}` + "\n"
cmd/internal/launcher/config_test.go:51:	if got := CodexSessionsDir("/home"); got != "/home/.codex/sessions" {
cmd/internal/sessionwatch/run.go:40:	ReadFirstLine(path string) ([]byte, error)
cmd/internal/sessionwatch/run.go:54:	SessionID  string    `json:"session_id"`
cmd/internal/sessionwatch/run.go:148:			rt.Log(adapt.Fired, "session_id="+result.ID)
cmd/internal/sessionwatch/run.go:290:	firstEvent, err := rt.ReadFirstLine(result.Path)
cmd/internal/launcher/createflow_test.go:479:	// Config written WITHOUT the resume binding (session_id is canonical storage).
cmd/internal/launcher/createflow_test.go:481:	if !strings.Contains(cfg, `"session_id":"MINTED-1"`) || strings.Contains(cfg, "--session-id") {
cmd/internal/launcher/createflow_test.go:849:	rt.files["/data/config-cx-codex.json"] = `{"agent":"claude","args":["--old"],"session_id":"OLD"}`
cmd/internal/launcher/createflow_test.go:999:	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":["--search"],"session_id":"CX-9"}`
cmd/internal/launcher/createflow_test.go:1016:	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":["--sandbox","danger-full-access","resume","CX-9","--no-alt-screen"],"session_id":"CX-9"}`
cmd/internal/launcher/createflow_test.go:1033:	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":["--search"],"session_id":"CX-9"}`
cmd/internal/launcher/createflow_test.go:1062:	rt.files["/data/config-work-claude.json"] = `{"agent":"claude","args":["--old"],"session_id":"OLD"}`
cmd/internal/launcher/createflow_test.go:1089:	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":[],"session_id":""}`
cmd/internal/launcher/createflow_test.go:1103:	rt.files["/data/config-work-claude.json"] = `{"agent":"claude","args":["--saved"],"session_id":"SAVED"}`
cmd/internal/launcher/createflow_test.go:1116:	if !strings.Contains(cfg, `"session_id":"EXPLICIT"`) || strings.Contains(cfg, "--resume") {
cmd/internal/launcher/createflow_test.go:1255:	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":["--search"],"session_id":"SUBAGENT"}`
cmd/internal/transcript/transcript.go:22:var codexRolloutRE = regexp.MustCompile(`^(.*/\.codex/sessions/.*/rollout-.*([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\.jsonl)$`)
cmd/internal/transcript/transcript.go:24:// CodexSessionIDFromPath extracts the native session id from a Codex rollout
cmd/internal/transcript/transcript.go:26:func CodexSessionIDFromPath(path string) string {
cmd/internal/transcript/transcript.go:38:	sid := CodexSessionIDFromPath(path)
cmd/internal/transcript/transcript.go:60:// ReadCodexRootSessionID reads one bounded, newline-terminated JSONL event and
cmd/internal/transcript/transcript.go:63:func ReadCodexRootSessionID(path string) string {
cmd/internal/transcript/transcript.go:64:	line, err := ReadFirstEvent(path)
cmd/internal/transcript/transcript.go:71:// ReadFirstEvent returns one bounded, newline-terminated JSONL event.
cmd/internal/transcript/transcript.go:72:func ReadFirstEvent(path string) ([]byte, error) {
cmd/internal/transcript/transcript.go:96:		SessionID string `json:"session_id"`
cmd/internal/transcript/transcript.go:103:		if path == "" || ReadCodexRootSessionID(path) != c.SessionID {
cmd/internal/transcript/transcript_test.go:28:func TestCodexSessionIDFromPath(t *testing.T) {
cmd/internal/transcript/transcript_test.go:31:	if got := CodexSessionIDFromPath(path); got != sid {
cmd/internal/transcript/transcript_test.go:32:		t.Fatalf("CodexSessionIDFromPath = %q, want %q", got, sid)
cmd/internal/transcript/transcript_test.go:34:	if got := CodexSessionIDFromPath("/tmp/not-codex.jsonl"); got != "" {
cmd/internal/transcript/transcript_test.go:69:func TestReadCodexRootSessionIDBoundaries(t *testing.T) {
cmd/internal/transcript/transcript_test.go:87:	if got := ReadCodexRootSessionID(path); got != sid {
cmd/internal/transcript/transcript_test.go:91:	if got := ReadCodexRootSessionID(path); got != "" {
cmd/internal/transcript/transcript_test.go:95:	if got := ReadCodexRootSessionID(path); got != "" {
cmd/internal/transcript/transcript_test.go:99:	if got := ReadCodexRootSessionID(path); got != "" {
cmd/internal/transcript/transcript_test.go:114:	if got := ReadCodexRootSessionID(path); got != sid {
cmd/internal/transcript/transcript_test.go:118:	if got := ReadCodexRootSessionID(path); got != "" {
cmd/internal/transcript/transcript_test.go:122:	if got := ReadCodexRootSessionID(filepath.Join(home, "missing.jsonl")); got != "" {
cmd/internal/transcript/transcript_test.go:125:	if got := ReadCodexRootSessionID(home); got != "" {
cmd/internal/transcript/transcript_test.go:151:	if err := os.WriteFile(config, []byte(`{"session_id":"`+rootSID+`"}`), 0o644); err != nil {
cmd/internal/transcript/transcript_test.go:157:	if err := os.WriteFile(config, []byte(`{"session_id":"`+subSID+`"}`), 0o644); err != nil {
cmd/internal/transcript/transcript_test.go:164:	if err := os.WriteFile(filepath.Join(data, "config-work-claude.json"), []byte(`{"session_id":"claude-id"}`), 0o644); err != nil {
cmd/internal/dispatcher/dispatcher_test.go:272:	mustWrite(t, filepath.Join(data, "config-T-claude.json"), `{"session_id":"sid1"}`)
diff --git a/cmd/internal/launcher/createflow.go b/cmd/internal/launcher/createflow.go
index 372b6e4..0b10008 100644
--- a/cmd/internal/launcher/createflow.go
+++ b/cmd/internal/launcher/createflow.go
@@ -96,7 +96,15 @@ func RunLaunch(opts LaunchOptions, rt Runtime, stderr io.Writer) (int, error) {
 		}
 
 		configPath := resolveConfigPath(rt, env.DataDir, rTag, rAgent)
-		plan := planRestart(m, rTag, rAgent, readSavedConfig(rt, configPath))
+		saved := readSavedConfig(rt, configPath)
+		savedSessionID := saved.SessionID
+		var quarantine bool
+		saved, quarantine = decideAutomaticResumeConfig(rAgent, saved, rt.AgentSessionExists(rAgent, savedSessionID, env.Cwd))
+		if quarantine {
+			fmt.Fprintf(stderr, "pair: saved session %q for %s is not available; starting fresh\n", savedSessionID, rAgent)
+			rt.Remove(configPath)
+		}
+		plan := planRestart(m, rTag, rAgent, saved)
 		if plan.DropConfig {
 			rt.Remove(configPath) // Shift+Alt+N / compaction: drop the config so create mints fresh.
 		}
@@ -643,11 +651,17 @@ func runConfigPicker(rt Runtime, configPath string, saved savedConfig, agent, ch
 		return 0, true // unusable config — proceed as if none.
 	}
 
-	savedArgsClean := persistedConfigArgs(saved.Args)
-	hasResumable := rt.AgentSessionExists(agent, saved.SessionID, cwd)
-	if saved.SessionID != "" && !hasResumable {
-		fmt.Fprintf(stderr, "pair: saved session %q for %s is not available; starting fresh\n", saved.SessionID, agent)
+	savedSessionID := saved.SessionID
+	hasResumable := rt.AgentSessionExists(agent, savedSessionID, cwd)
+	var quarantine bool
+	saved, quarantine = decideAutomaticResumeConfig(agent, saved, hasResumable)
+	if savedSessionID != "" && !hasResumable {
+		fmt.Fprintf(stderr, "pair: saved session %q for %s is not available; starting fresh\n", savedSessionID, agent)
+	}
+	if quarantine {
+		rt.Remove(configPath)
 	}
+	savedArgsClean := persistedConfigArgs(saved.Args)
 	choices := buildConfigChoices(hasResumable, savedArgsClean, *agentArgs, saved.SessionID)
 
 	labels := make([]string, len(choices))
diff --git a/cmd/internal/launcher/markers.go b/cmd/internal/launcher/markers.go
index 13c4e6b..1c7328f 100644
--- a/cmd/internal/launcher/markers.go
+++ b/cmd/internal/launcher/markers.go
@@ -56,6 +56,17 @@ type restartPlan struct {
 	ContinueSlug string // #55 compaction re-entry: re-seed the draft from this slug
 }
 
+// decideAutomaticResumeConfig rejects only persisted Codex bindings that no
+// longer identify a verified root rollout. Keep the saved launch parameters so
+// callers can still offer a fresh launch with the user's prior flags.
+func decideAutomaticResumeConfig(agent string, saved savedConfig, sessionValid bool) (savedConfig, bool) {
+	if agent != "codex" || saved.SessionID == "" || sessionValid {
+		return saved, false
+	}
+	saved.SessionID = ""
+	return saved, true
+}
+
 // planRestart maps a restart marker + the RESOLVED (tag, agent) + saved config
 // into the next launch (#99 M5b makes rename/continue native). The caller has
 // already applied the marker's tag/agent preference AND any rename_to move before
diff --git a/cmd/internal/launcher/osruntime.go b/cmd/internal/launcher/osruntime.go
index 471d402..85a3b3b 100644
--- a/cmd/internal/launcher/osruntime.go
+++ b/cmd/internal/launcher/osruntime.go
@@ -599,7 +599,8 @@ func (OSRuntime) AgentSessionExists(agent, sid, cwd string) bool {
 	case "agy":
 		return fileExists(AgyConversationPath(home, sid))
 	case "codex":
-		return transcript.Resolve("codex", sid, cwd, home) != ""
+		path := transcript.Resolve("codex", sid, cwd, home)
+		return path != "" && transcript.ReadCodexRootSessionID(path) == sid
 	case "muse":
 		return transcript.Resolve("muse", sid, cwd, home) != ""
 	}
@@ -624,7 +625,7 @@ func (r OSRuntime) LiveAgentSessionID(agent, tag string) string {
 			if !strings.HasPrefix(name, prefix) {
 				continue
 			}
-			if sid := transcript.CodexSessionIDFromPath(name); sid != "" {
+			if sid := transcript.ReadCodexRootSessionID(name); sid != "" {
 				return sid
 			}
 		}
package launcher

import "strings"

// Restart/quit marker logic (#99 M3, ported from bin/pair-shell's
// handle_restart_marker + pair-restart.sh handshake). The markers live under
// ~/.cache/pair/{quit,restart}-<session>; parsing + the re-launch decision are
// pure here, the read/clear IO sits on the Runtime seam.

// RestartMarker is the parsed ~/.cache/pair/restart-<session> handshake dropped
// by `pair restart` (Alt+n / Shift+Alt+N, #94 M1) or the #55 compaction branch.
type RestartMarker struct {
	Tag        string
	Agent      string
	SessionID  string // plain restart: live native session id captured before kill
	NewSession bool   // Shift+Alt+N / compaction: fresh agent conversation
	RenameTo   string // #22 inside-flow tag rename (native re-entry as of M5b)
	Continue   string // #55 compaction slug (native continue re-entry as of M5b)
}

// parseRestartMarker reads the `key=value` lines `pair restart` writes. Unknown
// keys are ignored; a missing marker is the caller's concern (empty content →
// zero value).
func parseRestartMarker(content string) RestartMarker {
	var m RestartMarker
	for _, line := range strings.Split(content, "\n") {
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		switch strings.TrimSpace(key) {
		case "tag":
			m.Tag = val
		case "agent":
			m.Agent = val
		case "session_id":
			m.SessionID = val
		case "new_session":
			m.NewSession = val == "1"
		case "rename_to":
			m.RenameTo = val
		case "continue":
			m.Continue = val
		}
	}
	return m
}

// restartPlan is the decision the in-process restart loop acts on: the next
// launch (Args), whether to drop the saved config first, and — for a #55
// compaction re-entry — the continuation slug to re-seed the draft from.
type restartPlan struct {
	Args         LaunchArgs
	DropConfig   bool   // Shift+Alt+N / compaction: drop the saved config first
	ContinueSlug string // #55 compaction re-entry: re-seed the draft from this slug
}

// decideAutomaticResumeConfig rejects only persisted Codex bindings that no
// longer identify a verified root rollout. Keep the saved launch parameters so
// callers can still offer a fresh launch with the user's prior flags.
func decideAutomaticResumeConfig(agent string, saved savedConfig, sessionValid bool) (savedConfig, bool) {
	if agent != "codex" || saved.SessionID == "" || sessionValid {
		return saved, false
	}
	saved.SessionID = ""
	return saved, true
}

// planRestart maps a restart marker + the RESOLVED (tag, agent) + saved config
// into the next launch (#99 M5b makes rename/continue native). The caller has
// already applied the marker's tag/agent preference AND any rename_to move before
// calling this, so tag/agent here are final. Mirrors handle_restart_marker (shell
// 762-810): Shift+Alt+N / compaction drop the config and re-launch fresh; a plain
// Alt+n composes the canonical resume binding onto the saved args (codex's
// `resume` subcommand leads via composeResumeArgs).
func planRestart(m RestartMarker, tag, agent string, saved savedConfig) restartPlan {
	base := LaunchArgs{Agent: agent, ForcedTag: tag}
	if m.NewSession {
		// Fresh conversation: keep the saved args, drop the config so the create
		// path mints a new session id rather than resuming the prior one. A
		// Continue slug only ever rides new_session (shell 1055-1056), so re-seed
		// here (the loop resolves the slug → draft).
		base.AgentArgs = append([]string(nil), saved.Args...)
		return restartPlan{Args: base, DropConfig: true, ContinueSlug: m.Continue}
	}
	// Default Alt+n: resume the prior conversation. A marker session id was read
	// from the live agent immediately before kill, so it is fresher than config.
	base.AgentArgs = composeResumeArgs(agent, saved.Args, firstNonEmpty(m.SessionID, saved.SessionID))
	return restartPlan{Args: base}
}
	raw += line + "\n"
	return r.WriteAtomic(path, raw)
}

func (r OSRuntime) globalDataDir() string {
	if r.GlobalDataDir != "" {
		return r.GlobalDataDir
	}
	return r.DataDir
}

func (OSRuntime) AgentSessionExists(agent, sid, cwd string) bool {
	if sid == "" {
		return false
	}
	home := os.Getenv("HOME")
	switch agent {
	case "claude":
		return fileExists(ClaudeTranscriptPath(home, cwd, sid))
	case "agy":
		return fileExists(AgyConversationPath(home, sid))
	case "codex":
		path := transcript.Resolve("codex", sid, cwd, home)
		return path != "" && transcript.ReadCodexRootSessionID(path) == sid
	case "muse":
		return transcript.Resolve("muse", sid, cwd, home) != ""
	}
	return false
}

func (r OSRuntime) LiveAgentSessionID(agent, tag string) string {
	if agent != "codex" || tag == "" {
		return ""
	}
	raw, err := r.ReadFile(filepath.Join(r.DataDir, "agent-pid-"+tag))
	if err != nil {
		return ""
	}
	root := strings.TrimSpace(raw)
	if root == "" {
		return ""
	}
	prefix := filepath.Join(os.Getenv("HOME"), ".codex", "sessions") + string(os.PathSeparator)
	for _, pid := range procutil.DescendantPIDs(root, procutil.ProcessChildren()) {
		for _, name := range procutil.LsofNames(pid) {
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			if sid := transcript.ReadCodexRootSessionID(name); sid != "" {
				return sid
			}
		}
	}
	return ""
}

package sessionwatch

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	uuidRE    = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	endUUIDRE = regexp.MustCompile(`(?i)([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`)
)

// AgentSpec describes one async session-file discovery contract.
type AgentSpec struct {
	Agent    string
	Home     string
	WatchDir string
}

// SessionID is the outcome of matching a candidate session file path.
type SessionID struct {
	Matched  bool
	NearMiss bool
	ID       string
	Path     string
}

// ConfigPayload is the restart config written as config-<tag>-<agent>.json.
type ConfigPayload struct {
	Agent     string   `json:"agent"`
	Args      []string `json:"args"`
	SessionID string   `json:"session_id"`
}

// SpecForAgent returns the async watcher contract for agents that need it.
func SpecForAgent(agent, home string) (AgentSpec, bool) {
	switch agent {
	case "codex":
		return AgentSpec{
			Agent:    agent,
			Home:     home,
			WatchDir: filepath.Join(home, ".codex", "sessions"),
		}, true
	case "agy":
		return AgentSpec{
			Agent:    agent,
			Home:     home,
			WatchDir: filepath.Join(home, ".gemini", "antigravity-cli", "conversations"),
		}, true
	case "muse":
		return AgentSpec{
			Agent:    agent,
			Home:     home,
			WatchDir: filepath.Join(home, ".local", "share", "muse", "sessions"),
		}, true
	default:
		return AgentSpec{}, false
	}
}

// Match checks whether path belongs to the agent's session-file shape and, if
// so, extracts the session id or reports a near miss.
func (s AgentSpec) Match(path string) SessionID {
	switch s.Agent {
	case "codex":
		prefix := filepath.Clean(s.WatchDir) + string(filepath.Separator)
		clean := filepath.Clean(path)
		if !strings.HasPrefix(clean, prefix) {
			return SessionID{}
		}
		base := filepath.Base(clean)
		if !strings.HasPrefix(base, "rollout-") || !strings.HasSuffix(base, ".jsonl") {
			return SessionID{}
		}
		stem := strings.TrimSuffix(base, ".jsonl")
		if match := endUUIDRE.FindStringSubmatch(stem); len(match) == 2 {
			return SessionID{Matched: true, ID: match[1], Path: path}
		}
		return SessionID{Matched: true, NearMiss: true, Path: path}
	case "agy":
		prefix := filepath.Clean(s.WatchDir) + string(filepath.Separator)
		clean := filepath.Clean(path)
		if !strings.HasPrefix(clean, prefix) {
			return SessionID{}
		}
		base := filepath.Base(clean)
		if !strings.HasSuffix(base, ".db") {
			return SessionID{}
		}
		id := strings.TrimSuffix(base, ".db")
		if uuidRE.MatchString(id) {
			return SessionID{Matched: true, ID: id, Path: path}
		}
		return SessionID{Matched: true, NearMiss: true, Path: path}
	case "muse":
		prefix := filepath.Clean(s.WatchDir) + string(filepath.Separator)
		clean := filepath.Clean(path)
		if !strings.HasPrefix(clean, prefix) {
			return SessionID{}
		}
		// Muse subagent sessions live under …/<root-uuid>/subagent/<sub-uuid>/session.jsonl.
		// Only the root session is resumable via `muse resume <id>`; ignore subagent interior (ARCH-PURE).
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
	default:
		return SessionID{}
	}
}

// StripResumeArgs removes resume bindings from args before they are persisted;
// the session_id field is the canonical store for that binding.
func StripResumeArgs(agent string, args []string) []string {
	stripped := make([]string, 0, len(args))
	i := 0
	if (agent == "codex" || agent == "muse") && len(args) >= 2 && args[0] == "resume" {
		i = 2
	}
	for i < len(args) {

exec
/bin/zsh -lc "sed -n '1,190p' cmd/internal/opener/opener.go; sed -n '1,280p' cmd/internal/opener/run.go; rg -n \"opener|pair open|PAIR_SESSION_ID\" README.md atlas cmd/pair-go bin nvim | head -120; sed -n '2960,3290p' nvim/init.lua; git log --oneline --decorate 356ec01f5aaf6c48d3fcf73a7aff1bd090c8dc79..HEAD" in /private/tmp/pair-context-usage
 succeeded in 0ms:
// Package opener is the Go owner of pair's two floating-pane viewer launchers
// (#93 M2, ported from bin/pair-scrollback-open + bin/pair-changelog-open):
//
//   - Alt+/ scrollback: render the agent pane's captured PTY to ANSI and open a
//     read-only nvim viewer, positioned to the user's current scroll offset.
//   - Alt+l changelog: launch a DETACHED render+distill build (survives the
//     viewer closing) and open an nvim watcher that tails the distilled log.
//
// The nvim viewers (nvim/scrollback.lua, nvim/changelog.lua) stay native (#95
// boundary); this package owns the orchestration. Following the #78 sessionwatch
// template, the pure decisions live here and IO (zellij/nvim/exec/detach/fs) sits
// behind the Runtime seam.
package opener

import (
	"encoding/json"
	"regexp"
)

// sgrEscape matches an ANSI CSI SGR (and other CSI) escape so a rendered .ansi
// line compares byte-for-byte against a zellij dump-screen plain-text line.
var sgrEscape = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

func stripSGR(s string) string { return sgrEscape.ReplaceAllString(s, "") }

// matchViewport finds the .ansi line (1-based) the user is currently looking at,
// by matching a zellij dump-screen (the pane's actual visible content, including
// zellij's scroll offset) against the rendered scrollback. Ported from
// pair-scrollback-open's awk scorer: index ansi lines (≥ 8 chars) → line numbers,
// derive candidate start offsets from each matching dump line, score each start
// by consecutive matches, and accept the best only if ≥ 50% of non-blank dump
// lines match. On ties it prefers the smaller start (deterministic — the shell's
// map iteration order was not). Returns (line, true) on a high-confidence match;
// the caller falls back to the renderer's own .viewport otherwise.
func matchViewport(dump, ansi []string) (int, bool) {
	an := len(ansi)

	// ansi line → 1-based line numbers (only substantial lines, to avoid short
	// -line false positives).
	idx := map[string][]int{}
	for j, line := range ansi {
		if line != "" && len(line) >= 8 {
			idx[line] = append(idx[line], j+1)
		}
	}

	// Candidate starts: for each substantial dump line that matches an ansi
	// line at 1-based position p (dump index i), "dump line 0 lands at ansi
	// line p-i" is a hypothesis.
	seen := map[int]bool{}
	for i, d := range dump {
		if d == "" || len(d) < 8 {
			continue
		}
		for _, p := range idx[d] {
			s := p - i
			if s < -an {
				continue
			}
			seen[s] = true
		}
	}

	// Non-blank dump lines (the match-fraction denominator).
	nb := 0
	for _, d := range dump {
		if d != "" {
			nb++
		}
	}

	bestScore, bestStart := -1, 0
	for s := range seen {
		score := 0
		for i, d := range dump {
			if d == "" {
				continue
			}
			j := s - 1 + i // 0-based ansi index
			if j < 0 || j >= an {
				continue
			}
			if ansi[j] == d {
				score++
			}
		}
		if score > bestScore || (score == bestScore && s < bestStart) {
			bestScore, bestStart = score, s
		}
	}

	if nb > 0 && bestScore*2 >= nb {
		s := bestStart
		if s < 1 {
			s = 1 // dump starts before .ansi line 1 → user is at the very top
		}
		return s, true
	}
	return 0, false
}

// resolveSessionID implements the #63 change-log keying: an explicit
// PAIR_SESSION_ID wins; else the per-tag config's session_id; else "" (the
// legacy unsuffixed base). configJSON is the raw config-<tag>-<agent>.json bytes
// (nil/empty when absent).
func resolveSessionID(envSID string, configJSON []byte) string {
	if envSID != "" {
		return envSID
	}
	if len(configJSON) == 0 {
		return ""
	}
	var c struct {
		SessionID string `json:"session_id"`
	}
	if json.Unmarshal(configJSON, &c) != nil {
		return ""
	}
	return c.SessionID
}

// changelogBase is the per-session change-log path stem: the sid suffix is
// appended only when resolved (fresh sessions branch; a resume reuses it).
func changelogBase(dataDir, tag, agent, sid string) string {
	base := dataDir + "/changelog-" + tag + "-" + agent
	if sid != "" {
		base += "-" + sid
	}
	return base
}

// distillerInner is the detached build pipeline: render the cleaned scrollback,
// then distill it into the change log + anchor. It references PCL_* env (set by
// distillerEnv) so the paths need no shell quoting — mirrors the shell exactly.
const distillerInner = `"$PCL_BIN" scrollback render --plain --max-lines 0 --with-timestamps "$PCL_RAW" "$PCL_EVENTS" "$PCL_CLEANED" && "$PCL_BIN" changelog render --cleaned "$PCL_CLEANED" --log "$PCL_LOG" --anchor "$PCL_ANCHOR" --agent "$PCL_AGENT"`

// distillerEnv builds the PCL_* KEY=VALUE environment the detached distiller
// reads (paths passed via env, never interpolated into the sh -c string).
func distillerEnv(binPath, raw, events, cleaned, log, anchor, agent string) []string {
	return []string{
		"PCL_BIN=" + binPath,
		"PCL_RAW=" + raw,
		"PCL_EVENTS=" + events,
		"PCL_CLEANED=" + cleaned,
		"PCL_LOG=" + log,
		"PCL_ANCHOR=" + anchor,
		"PCL_AGENT=" + agent,
	}
}
package opener

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Options are a viewer launcher's inputs after CLI/env resolution.
type Options struct {
	Tag       string
	Agent     string
	DataDir   string
	PairHome  string // nvim lua root + the pair binary the detached distiller execs
	SessionID string // PAIR_SESSION_ID (changelog per-session keying)
	Jump      string // --jump prev|next (scrollback)
}

// Runtime is the IO/process boundary for the viewer launchers. Pure decisions
// live in opener.go; everything here that touches zellij/nvim/exec/detach/fs sits
// behind this seam so the orchestration is unit-testable with a fake.
type Runtime interface {
	Sleep(time.Duration)
	Getpid() string
	ProcessAlive(pid string) bool

	ReadFile(path string) (string, error)
	WriteFile(path, data string) error
	WriteAtomic(path, data string) error // temp + rename (for the .viewport a live viewer may re-read)
	Remove(path string)
	FileSize(path string) (int64, bool) // for `[ -s FILE ]` guards
	Touch(path string) error            // `[ -f LOG ] || : > LOG`
	Executable(path string) bool        // `[ -x FILE ]`

	RenderScrollback(raw, events, ansi string) error // in-process scrollbackcmd.Run (sync)
	AgentPaneID() string                             // zellij list-panes → agent pane id, "" if none
	DumpScreen(paneID string) (string, error)        // zellij dump-screen
	// StartDetached launches `sh -c script` in its own session (setsid) with
	// extraEnv, stderr → statusPath, detached from this process; returns its pid.
	StartDetached(script string, extraEnv []string, statusPath string) (string, error)
	// RunViewer execs nvim (-u luaPath file) with extraEnv as a HELD child,
	// returning when the user quits.
	RunViewer(luaPath, file string, extraEnv []string) error
}

func missingEnv(opts Options) bool {
	return opts.DataDir == "" || opts.Tag == "" || opts.Agent == ""
}

// RunScrollback renders the agent pane's captured scrollback to ANSI, overlays
// the user's scroll position, and opens the read-only nvim viewer (Alt+/).
func RunScrollback(opts Options, rt Runtime, stderr io.Writer) int {
	if missingEnv(opts) {
		fmt.Fprintf(stderr, "pair-scrollback-open: missing PAIR_DATA_DIR / PAIR_TAG / PAIR_AGENT\n")
		fmt.Fprintf(stderr, "  This is meant to run inside a pair session.\n")
		rt.Sleep(3 * time.Second)
		return 1
	}
	sb := opts.DataDir + "/scrollback-" + opts.Tag + "-" + opts.Agent
	lock := sb + ".openlock"

	// Re-entrancy: a second Alt+/ while a viewer is up exits (focus falls back).
	if raw, err := rt.ReadFile(lock); err == nil {
		if other := strings.TrimSpace(raw); rt.ProcessAlive(other) {
			return 0
		}
	}

	raw, events, ansi := sb+".raw", sb+".events.jsonl", sb+".ansi"
	if sz, ok := rt.FileSize(raw); !ok || sz == 0 {
		fmt.Fprintf(stderr, "pair-scrollback-open: no scrollback yet for %s/%s\n", opts.Tag, opts.Agent)
		fmt.Fprintf(stderr, "  (capture starts when the agent pane begins emitting output.)\n")
		rt.Sleep(3 * time.Second)
		return 0
	}
	if err := rt.RenderScrollback(raw, events, ansi); err != nil {
		fmt.Fprintf(stderr, "pair-scrollback-open: scrollback-render failed: %v\n", err)
		rt.Sleep(5 * time.Second)
		return 1
	}

	overlayViewport(opts, rt, sb, ansi)

	env := []string{
		"PAIR_NVIM_PID_FILE=" + opts.DataDir + "/nvim-pid-" + opts.Tag + "-scrollback",
		"PAIR_SCROLLBACK_JUMP=" + opts.Jump,
	}
	_ = rt.WriteFile(lock, rt.Getpid()+"\n")
	defer rt.Remove(lock)
	_ = rt.RunViewer(opts.PairHome+"/nvim/scrollback.lua", ansi, env)
	return 0
}

// overlayViewport matches zellij's actual visible content onto the rendered
// .ansi to record the line the user is looking at. Best-effort: any seam failure
// leaves the renderer's own .viewport in place.
func overlayViewport(opts Options, rt Runtime, sb, ansi string) {
	paneID := rt.AgentPaneID()
	if paneID == "" {
		return
	}
	dump, err := rt.DumpScreen(paneID)
	if err != nil || dump == "" {
		return
	}
	content, err := rt.ReadFile(ansi)
	if err != nil {
		return
	}
	var ansiLines []string
	for _, l := range strings.Split(content, "\n") {
		ansiLines = append(ansiLines, stripSGR(l))
	}
	if line, ok := matchViewport(strings.Split(dump, "\n"), ansiLines); ok {
		// Atomic (temp + rename), like the shell's `> .tmp && mv -f`: a live
		// viewer's `G` refresh may re-read .viewport concurrently.
		_ = rt.WriteAtomic(sb+".viewport", strconv.Itoa(line)+"\n")
	}
}

// RunChangelog resolves the per-session change log, launches a DETACHED
// render+distill build (survives the viewer closing), and opens the nvim watcher
// (Alt+l).
func RunChangelog(opts Options, rt Runtime, stderr io.Writer) int {
	if missingEnv(opts) {
		fmt.Fprintf(stderr, "pair-changelog-open: missing PAIR_DATA_DIR / PAIR_TAG / PAIR_AGENT\n")
		fmt.Fprintf(stderr, "  This is meant to run inside a pair session.\n")
		rt.Sleep(3 * time.Second)
		return 1
	}

	sid := opts.SessionID
	if sid == "" {
		if cfg, err := rt.ReadFile(opts.DataDir + "/config-" + opts.Tag + "-" + opts.Agent + ".json"); err == nil {
			sid = resolveSessionID("", []byte(cfg))
		}
	}
	base := changelogBase(opts.DataDir, opts.Tag, opts.Agent, sid)
	sb := opts.DataDir + "/scrollback-" + opts.Tag + "-" + opts.Agent
	raw, events := sb+".raw", sb+".events.jsonl"
	log, anchor, cleaned := base+".md", base+".anchor", base+".cleaned"
	openlock, dlock, status := base+".openlock", base+".distill.lock", base+".status"

	// Viewer re-entrancy.
	if r, err := rt.ReadFile(openlock); err == nil {
		if other := strings.TrimSpace(r); rt.ProcessAlive(other) {
			return 0
		}
	}
	_ = rt.WriteFile(openlock, rt.Getpid()+"\n")
	defer rt.Remove(openlock)

	_ = rt.Touch(log)

	// Launch the detached distiller unless one is already running, RAW has
	// content, and the pair binary is built.
	distillerRunning := false
	if r, err := rt.ReadFile(dlock); err == nil {
		if p := strings.TrimSpace(r); rt.ProcessAlive(p) {
			distillerRunning = true
		}
	}
	bin := opts.PairHome + "/bin/pair"
	if sz, ok := rt.FileSize(raw); !distillerRunning && ok && sz > 0 && rt.Executable(bin) {
		_ = rt.WriteFile(status, "")
		env := distillerEnv(bin, raw, events, cleaned, log, anchor, opts.Agent)
		if pid, err := rt.StartDetached(distillerInner, env, status); err == nil {
			_ = rt.WriteFile(dlock, pid+"\n")
		}
	}

	env := []string{
		"PAIR_CHANGELOG_LOG=" + log,
		"PAIR_CHANGELOG_DLOCK=" + dlock,
		"PAIR_CHANGELOG_STATUS=" + status,
	}
	_ = rt.RunViewer(opts.PairHome+"/nvim/changelog.lua", log, env)
	return 0
}
atlas/session-identity.md:161:target scoping uses the inherited `PAIR_SESSION_ID`, then Pair's config cache;
atlas/go-migration-inventory.md:52:   dependency order: launch/session lifecycle, scrollback and changelog openers,
atlas/go-migration-inventory.md:117:| `bin/pair-scrollback-render` / `cmd/pair-scrollback-render` / `cmd/internal/scrollbackcmd` | Go binary plus shared runner | `cmd/pair-scrollback-open` (in-process, #93 M2), `cmd/pair-changelog-open`'s detached distiller, `nvim/scrollback.lua` refresh; development-only `pair-go scrollback-render` | `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out` and `pair-go scrollback-render ...`; nonzero on render/write failure. | Reads `.raw` and `.events.jsonl`; atomically writes `.ansi` or cleaned text. | implemented `pair scrollback-render` route (#92); the Alt+/ opener now calls `scrollbackcmd.Run` **in-process** (#93 M2, no subprocess); the changelog opener's detached distiller + `nvim/scrollback.lua` still shell `pair scrollback-render`; `bin/pair-scrollback-render` retained as thin shim | P0 |
atlas/go-migration-inventory.md:125:| `cmd/pair-scrollback-open` / `cmd/internal/opener` | Go binary plus shared runner | zellij Alt+/ Run, nvim Alt+b jump | `pair-scrollback-open [--jump prev|next]`; opens read-only nvim viewer; singleton lock. | Requires `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_HOME`; renders in-process (`scrollbackcmd`), zellij IPC (list-panes/dump-screen), nvim; writes `.ansi`, `.viewport`, lock. | ported to Go (#93 M2) on the #78 template — pure viewport scorer in `opener`, IO behind the `Runtime` seam; **replaces** the shell script at the same PATH name (zellij invokes by name → no shim); `nvim/scrollback.lua` stays native | P1 |
atlas/go-migration-inventory.md:127:| `cmd/pair-changelog-open` / `cmd/internal/opener` | Go binary plus shared runner | zellij Alt+l Run | Opens changelog viewer and starts detached render/distill singleton. | Requires Pair env; launches a `setsid`-detached `pair scrollback-render` / `pair changelog` build (#92), nvim watcher; reads/writes `changelog-*` sidecars. | ported to Go (#93 M2) — shared `opener` package (session keying + detached distiller), IO behind the seam; **replaces** the shell script at the same PATH name (no shim); Go `SysProcAttr.Setsid` replaces setsid/perl | P1 |
atlas/go-migration-inventory.md:222:  surfaces (scrollback/changelog openers, review helpers, clipboard helpers, the
atlas/go-migration-inventory.md:225:  + `cmd/pair-changelog-open`, sharing one `cmd/internal/opener` package: the pure
atlas/go-migration-inventory.md:229:  The Alt+/ opener renders in-process via `scrollbackcmd` (no subprocess); the
atlas/go-migration-inventory.md:230:  Alt+l opener keeps a detached `pair scrollback-render`/`pair changelog` build so
atlas/go-migration-inventory.md:241:  (the shared string-based fs primitives, now embedded by opener + titlepoller +
atlas/go-migration-inventory.md:312:  (`pair-wrap`, `copy_command "copy-on-select"`, `Run "pair-help"`/openers, the
atlas/go-migration-inventory.md:358:  - the changelog distiller (`opener.distillerInner`) calls
atlas/go-migration-inventory.md:469:- `cmd/internal/opener/opener.go`
atlas/go-migration-inventory.md:470:- `cmd/internal/opener/opener_test.go`
atlas/go-migration-inventory.md:471:- `cmd/internal/opener/run.go`
atlas/go-migration-inventory.md:472:- `cmd/internal/opener/run_test.go`
atlas/go-migration-inventory.md:473:- `cmd/internal/opener/runcli.go`
atlas/go-migration-inventory.md:474:- `cmd/internal/opener/runtime.go`
nvim/review/markers_test.lua:68:-- a stray opener beyond MULTILINE_LINE_BUDGET (200) yields no section
atlas/review-workbench.md:165:  `PAIR_SESSION_ID` → `config-<tag>-<agent>.json` → live Codex rollout via
nvim/init.lua:822:  -- PAIR_SESSION_ID it was written under, and a reader IGNORES a target from a
nvim/init.lua:823:  -- different session. So a fresh pair session (new PAIR_SESSION_ID) prompts
nvim/init.lua:843:    local sid = vim.env.PAIR_SESSION_ID
nvim/init.lua:2977:-- opener builds. A focused reader, not pair_read_saved_config() -- that one is
nvim/init.lua:2981:  local sid = vim.env.PAIR_SESSION_ID
nvim/init.lua:3769:-- Hand-rolled minimal autopair: each opener inserts its closer and parks
nvim/init.lua:3778:-- trailing closer would just be in the way — so drop in the bare opener.
nvim/init.lua:3807:  -- follows, insert the bare opener so we don't strand a closer mid-word.
nvim/init.lua:3810:  -- so a single `u` undoes the whole opener+closer insertion.
nvim/init.lua:3831:  -- Closers for non-quote pairs only — quote keys map to the opener handler
atlas/architecture.md:125:opener's detached distiller and `nvim/scrollback.lua` (`pair scrollback-render` /
atlas/architecture.md:129:opener no longer shells `pair scrollback-render` — #93 M2 renders via
atlas/architecture.md:130:`scrollbackcmd` in-process. The changelog opener's *detached* distiller still
atlas/architecture.md:534:     **Why bare BEL is opt-in.** When an OSC sequence's terminating `\x07` arrives in a read whose preceding bytes (the `\x1b]<ps>;` opener) were already consumed by a prior match, `OSC_RE` can't reconstruct the boundary, and the trailing `\x07` looks like a standalone BEL. Live data from a single 2hr Claude Code session showed 76 emits, only 8 legitimate (all OSC 777); the other 68 were BEL fallback firing on tails of OSC 8 hyperlinks (claude renders file references as clickable links) and OSC 0 spinner title sets. Modern TUI agents signal attention via OSC 9/777 explicitly — the BEL fallback's defensive value never materialized. The detection branch still runs (so `PAIR_WRAP_LOG` shows `BEL-skip` lines), it just doesn't write to the outer TTY unless the env flag is set.
atlas/architecture.md:599:**Open (`cmd/pair-scrollback-open` / `cmd/internal/opener`, Go — #93 M2).** Validates `PAIR_DATA_DIR` / `PAIR_TAG` / `PAIR_AGENT`, renders **in-process** via `scrollbackcmd` (no `pair scrollback-render` subprocess), then *launches* `nvim -u $PAIR_HOME/nvim/scrollback.lua $ANSI` as a held child (`RunViewer`) — deliberately **not** an exec-replace, so the launcher stays alive as nvim's parent and a `defer` clears the re-entrancy lock on quit (the Go analog of the old shell `EXIT`/`INT`/`TERM` trap). Errors print and `Sleep` briefly so the message is readable before the floating pane self-closes. The Go binary **replaced** the old POSIX-sh script at the same PATH name (zellij invokes it by name, so no shim is kept); the viewport scorer + session keying are pure and unit-tested in `opener`, with zellij/nvim/fs behind its `Runtime` seam. `Alt+/` is now a left-stack pane-local shortcut: `pair wrap` handles it from the agent pane and `nvim/init.lua` handles it from the draft pane, each launching the same 100% × 100% floating viewer with `close_on_exit=true`. The user's `:q` in the viewer dismisses the pane and returns to the selected workbench topology untouched. **Re-entrancy guard:** pressing `Alt+/` again while the viewer is already focused should not stack another nvim (one `:q`/Esc per layer to unwind). zellij can't conditionally skip a `Run`, so the launcher self-guards: before launching nvim it writes its own PID to `$PAIR_DATA_DIR/scrollback-<tag>-<agent>.openlock`, and on entry it returns immediately if that lock already holds a *live* PID — the redundant floating pane then self-dismisses via `close_on_exit` and focus falls back to the open viewer. A stale lock (hard kill) carries a dead PID and is reclaimed by the next open's liveness check (`procutil.Alive`, i.e. `kill -0`). The draft pane's `Alt+b` (`--jump prev`) runs the same launcher, so it's covered too.
atlas/architecture.md:762:  `pair-changelog-open` (`cmd/pair-changelog-open` / `cmd/internal/opener`, Go
atlas/architecture.md:763:  since #93 M2 — the shared `opener` package with the scrollback launcher) in a
atlas/architecture.md:842:  (the opener `bin/pair-changelog-open` and the draft-nvim `.ready` watcher):
atlas/architecture.md:843:  the exported `PAIR_SESSION_ID` (set by the launcher at launch for claude-fresh /
atlas/architecture.md:858:`PAIR_SESSION_ID` / config-resolved, #63); an end-to-end orchestrator smoke
atlas/architecture.md:997:The Go launcher's `RunLaunch` prepends `$PAIR_HOME/bin` to `$PATH` once at entry (via the pure `prependBinToPath`, `cmd/internal/launcher/pathenv.go` + `createflow.go`), before exec'ing zellij. zellij and all its child processes (panes — `pair-wrap`, `copy_command "copy-on-select"`, `Run "pair-help"`/openers, the nvim viewers) inherit the PATH and resolve `clipboard-to-pane`, `copy-on-select`, and the `pair` binary (e.g. nvim's `pair quit` / `pair restart` keybinds) by bare name. This lets the zellij KDL configs reference these helpers without `sh -c` env-var quoting hacks. The retired shell `bin/pair` did this prepend; the Go launcher that replaced it dropped it in #99 M5c — a real regression for a copied/Homebrew install whose `bin/` isn't already on the user's PATH — and #95 restored it (guarded by a `prependBinToPath` unit test + a copied-binary smoke asserting bare-name helper resolution).
nvim/review/markers.lua:18:-- Per-section newline budget — a stray opener absorbs at most this many lines.
end
pair_start_pending_fs_watch()

-- Watch $PAIR_DATA_DIR for the change-log "build complete" marker (#58). The
-- detached distiller drops "changelog-<tag>-<agent>.ready" only when a triggered
-- build actually changed the log; we flash the statusline + delete the marker
-- (one-shot). A low-frequency timer poll, NOT fs_event: macOS FSEvents is
-- unreliable from nvim here (it surfaces EMFILE with a nil filename), and the
-- scrollback-pending fs_event watcher only gets away with it because a
-- FocusGained fallback covers the miss. This signal has no such fallback — its
-- whole job is to flash while the operator works in the *agent* pane (the draft
-- statusline stays on screen), so it can't depend on focus. One fs_stat every
-- 2s is negligible; the ≤2s latency is invisible against a slow background build.
-- Resolve the change-log session id (#63): the env var bin/pair exports when the
-- id is known at launch (claude-fresh / any resume), else the per-tag config the
-- session watcher writes (codex/agy discover it async). Mirrors the env->config
-- order in bin/pair-changelog-open so the polled .ready path matches the base the
-- opener builds. A focused reader, not pair_read_saved_config() -- that one is
-- defined later in this file (Lua local-function ordering) and also reads the
-- agent-<tag> file, which is overkill here.
local function pair_changelog_session_id(data_dir, tag, agent)
  local sid = vim.env.PAIR_SESSION_ID
  if sid and sid ~= '' then return sid end
  local cf = io.open(data_dir .. '/config-' .. tag .. '-' .. agent .. '.json', 'r')
  if not cf then return nil end
  local body = cf:read('*a'); cf:close()
  local ok, parsed = pcall(vim.json.decode, body)
  if ok and type(parsed) == 'table' and parsed.session_id and parsed.session_id ~= '' then
    return parsed.session_id
  end
  return nil
end

local function pair_start_changelog_ready_watch()
  local data_dir = vim.env.PAIR_DATA_DIR
    or ((vim.env.XDG_DATA_HOME or (vim.env.HOME .. '/.local/share')) .. '/pair')
  local tag = vim.env.PAIR_TAG or vim.env.PAIR_AGENT or 'claude'
  local agent = vim.env.PAIR_AGENT or 'claude'
  vim.fn.timer_start(2000, function()
    -- Re-resolve each tick: a codex/agy id may land in the config mid-session.
    local sid = pair_changelog_session_id(data_dir, tag, agent)
    local base = data_dir .. '/changelog-' .. tag .. '-' .. agent
    if sid then base = base .. '-' .. sid end
    local marker = base .. '.ready'
    if not vim.loop.fs_stat(marker) then return end
    os.remove(marker) -- one-shot: consume the marker so the flash fires once
    pair_flash_notify('✓ change log ready · Alt+l')
  end, { ['repeat'] = -1 })
end
pair_start_changelog_ready_watch()

pair_apply_mode_bg(vim.fn.mode():sub(1, 1))

-- ---------------------------------------------------------------------------
-- quit-blocker — fat-finger guard for muscle-memory :wq / :q / ZZ etc.
-- ---------------------------------------------------------------------------
-- This nvim instance is the pair draft pane, not a standalone editor. The
-- correct exits are Alt+x (full quit) or Alt+d (detach); a stray :wq would
-- kill the draft pane mid-session and orphan zellij's layout. We rewrite
-- the common quit verbs as a no-op that echoes the right path. Saves still
-- happen via autosave, so swallowing the `:w` part of `:wq` costs nothing.
function _G.PairQuitWarn()
  vim.api.nvim_echo({
    { 'pair: ', 'Question' },
    { 'use Alt+x to quit, or Alt+d to detach', 'WarningMsg' },
  }, false, {})
end

-- Match the WHOLE typed command exactly (cmdline ==# 'q' etc.) so this only
-- fires for bare quits, not e.g. `:qall` typed character-by-character or a
-- substitute pattern that happens to contain 'q'. The `<expr>` form lets us
-- branch on getcmdtype() so command-mode-only triggers fire.
local quit_verbs = {
  'q', 'q!', 'wq', 'wq!', 'quit', 'quit!',
  'qa', 'qa!', 'qall', 'qall!',
  'wqa', 'wqa!', 'wqall', 'wqall!',
  'x', 'x!', 'xa', 'xa!', 'xall', 'xall!',
  'exit', 'exit!',
}
for _, v in ipairs(quit_verbs) do
  vim.cmd(string.format(
    [[cnoreabbrev <expr> %s getcmdtype() == ':' && getcmdline() ==# %q ? 'lua PairQuitWarn()' : %q]],
    v, v, v
  ))
end

-- Normal-mode shortcuts that bypass the cmdline (and thus the abbreviations).
vim.keymap.set('n', 'ZZ', function() PairQuitWarn() end, { silent = true, desc = 'pair: quit blocked' })
vim.keymap.set('n', 'ZQ', function() PairQuitWarn() end, { silent = true, desc = 'pair: quit blocked' })

-- ---------------------------------------------------------------------------
-- Alt+x / Alt+d confirm prompts
-- ---------------------------------------------------------------------------
-- The zellij keybindings for Alt+x (full quit) and Alt+d (detach) route
-- here instead of running the action directly. Both are easy to fat-finger
-- — Alt+x is unrecoverable (kills the zellij session and its processes)
-- and Alt+d drops the user out of a long-running attached session. The
-- zellij side, pair wrap, or pair term routes the request into draft nvim;
-- vim.fn.confirm pops a modal Y/N (default No), and the
-- action only fires on Yes. Y is shelled out via vim.fn.system because
-- nvim has no direct zellij IPC and re-binding zellij keybindings to first
-- check a flag is more state than this is worth.
-- If the user fires a confirm-requiring keybind while the rung is
-- minimized, the modal prompt would land on a 1-row pane where nothing
-- is visible. Step up to small first so the prompt renders, then defer
-- the actual prompt one event-loop tick — zellij's resize after
-- swap-layout reaches nvim asynchronously, and vim.fn.confirm reads
-- window dimensions when it's called.
local function pair_ensure_visible_then(fn)
  if pair_layout_state == 'minimized' and _G.PairLayoutBigger then
    _G.PairLayoutBigger()
    vim.defer_fn(fn, 100)
  else
    fn()
  end
end

-- Compact "Nu" duration: `45s` `12m` `3.2h` `5d`. Used in the confirm
-- modals so the session-id line carries a "this session is X old, last
-- touched Y ago" hint without ballooning into a sentence.
local function humanize_dur(secs)
  if secs < 0 then secs = 0 end
  if secs < 60 then return string.format('%ds', secs) end
  if secs < 3600 then return string.format('%dm', math.floor(secs / 60)) end
  if secs < 86400 then return string.format('%.1fh', secs / 3600) end
  return string.format('%.1fd', secs / 86400)
end

-- Resolve the on-disk session file for (agent, sid) and return
-- "(<age> old, <idle> idle)" — or nil if the file can't be found
-- (uncaptured id, agent we don't have a path resolver for, etc.).
-- Only called from the confirm modals, so the cost (one stat for
-- claude; a find for codex) is paid at most once
-- per Alt+x / Alt+n press.
local function session_age_hint(agent, sid)
  if not sid or sid == '' then return nil end
  local home = vim.env.HOME or ''
  local path
  if agent == 'claude' then
    local cwd = vim.env.PWD or vim.fn.getcwd()
    local enc = cwd:gsub('[./]', '-')
    path = home .. '/.claude/projects/' .. enc .. '/' .. sid .. '.jsonl'
    if vim.fn.filereadable(path) ~= 1 then path = nil end
  elseif agent == 'codex' then
    local cmd = 'find ' .. vim.fn.shellescape(home .. '/.codex/sessions')
      .. " -type f -name '*" .. sid .. "*.jsonl' 2>/dev/null | head -1"
    local h = io.popen(cmd)
    if h then path = h:read('*l'); h:close() end
  end
  if not path or path == '' then return nil end
  local h = io.popen('stat -f "%B %m" ' .. vim.fn.shellescape(path) .. ' 2>/dev/null')
  if not h then return nil end
  local out = h:read('*l')
  h:close()
  if not out then return nil end
  local birth, mtime = out:match('^(%d+) (%d+)$')
  if not birth then return nil end
  local now = os.time()
  return string.format('(%s old, %s idle)',
                       humanize_dur(now - tonumber(birth)),
                       humanize_dur(now - tonumber(mtime)))
end

-- Read the per-(tag,agent) saved config so the Alt+x prompt can show the
-- user what they're about to detach from for the future `pair resume
-- <tag>` path. Returns nil when the tag isn't set, the agent file is
-- missing, or the JSON parse fails — in which case the prompt falls back
-- to the bare confirmation.
local function pair_read_saved_config()
  local tag = vim.env.PAIR_TAG
  if not tag or tag == '' then return nil end
  local data_dir = vim.env.PAIR_DATA_DIR
    or ((vim.env.XDG_DATA_HOME or (vim.env.HOME .. '/.local/share')) .. '/pair')

  local af = io.open(data_dir .. '/agent-' .. tag, 'r')
  if not af then return nil end
  local agent = af:read('*l')
  af:close()
  if not agent or agent == '' then return nil end

  local cfg = { tag = tag, agent = agent }
  local cf = io.open(data_dir .. '/config-' .. tag .. '-' .. agent .. '.json', 'r')
  if cf then
    local body = cf:read('*a')
    cf:close()
    local ok, parsed = pcall(vim.json.decode, body)
    if ok and type(parsed) == 'table' then
      cfg.args       = parsed.args
      cfg.session_id = parsed.session_id
    end
  end
  return cfg
end

function _G.PairConfirmQuit()
  pair_ensure_visible_then(function()
    local prompt = 'Quit pair session? This kills the session and all its processes.'
    local cfg = pair_read_saved_config()
    if cfg then
      local args_line
      if type(cfg.args) == 'table' and #cfg.args > 0 then
        args_line = table.concat(cfg.args, ' ')
      else
        args_line = '<none>'
      end
      local sid_line = cfg.session_id and cfg.session_id ~= '' and cfg.session_id or '<not captured>'
      local age = session_age_hint(cfg.agent, cfg.session_id)
      if age then sid_line = sid_line .. '  ' .. age end
      prompt = prompt
        .. '\n\nResumable later via `pair resume ' .. cfg.tag .. '`:'
        .. '\n  agent:      ' .. cfg.agent
        .. '\n  args:       ' .. args_line
        .. '\n  session id: ' .. sid_line
    end
    local ans = vim.fn.confirm(prompt, '&Yes\n&No', 2)
    if ans == 1 then
      vim.fn.system({ 'pair', 'quit' })
    end
  end)
end

function _G.PairConfirmDetach()
  pair_ensure_visible_then(function()
    local ans = vim.fn.confirm('Detach from this pair session?', '&Yes\n&No', 2)
    if ans == 1 then
      if has_ui() then
        vim.fn.system({ 'zellij', 'action', 'detach' })
      end
    end
  end)
end

-- Shared between Alt+n (PairConfirmRestart) and Shift+Alt+N
-- (PairConfirmRestartNewSession). Differs in whether `pair restart`
-- is invoked with --new-session and what the prompt says.
--
--   Alt+n         — pure pair reload; agent session is preserved
--                   (resumed via --resume <id> / resume <id>).
--   Shift+Alt+N   — same tag + agent + args, but a fresh agent
--                   conversation (saved config is dropped).
--
-- Both confirms offer an extra (R)ename option (#000022 M2): the
-- restart choreography becomes kill → `pair rename <old> <new>` →
-- re-exec with PAIR_FORCE_TAG=<new>, so the agent conversation
-- (resume) or fresh session (new-session) is preserved under the
-- new tag. The rename runs in handle_restart_marker after the kill;
-- pre-validation here via `pair rename --restart-check` so the user
-- gets immediate feedback on bad tags / collisions instead of
-- discovering them post-kill.
local function pair_rename_prompt(current_tag)
  -- Loop until the user enters a valid new tag (returned) or cancels
  -- with empty input / Esc (returns nil).
  while true do
    -- Pre-fill with current_tag so the user can edit in place rather
    -- than retype the prefix of a related name.
    local input = vim.fn.input({
      prompt = 'New tag: ',
      default = current_tag,
      cancelreturn = '',
    })
    if input == nil or input == '' then return nil end
    -- No prefix-stripping and no charset check here (#130). Both used to be Lua
    -- twins of rules the binary already owns, and the 📁 scheme made keeping them
    -- in sync actively harmful: the tab title now reads `📁repo-tag`, so pasting
    -- it is the natural move, and a local `^[%w_-]+$` test rejects it before
    -- `pair rename` ever gets the chance to resolve it through the ledger.
    -- One implementation of the rule, in the binary.
    local new_tag = input
    if new_tag == current_tag then
      vim.api.nvim_echo({ { '\nnew tag matches current tag; nothing to do', 'WarningMsg' } }, false, {})
      return nil
    end
    local out = vim.fn.system({ 'pair', 'rename', '--restart-check', current_tag, new_tag })
    if vim.v.shell_error == 0 then
      return new_tag
    end
    vim.api.nvim_echo({ { '\n' .. (out:gsub('%s+$', '')) .. ' — try again', 'WarningMsg' } }, false, {})
  end
end

local function pair_confirm_restart_impl(new_session)
  pair_ensure_visible_then(function()
    local prompt
    if new_session then
      prompt = 'Continue work with a brand new session?'
    else
      prompt = 'Reload pair?'
    end
    local cfg = pair_read_saved_config()
    if cfg then
      local args_line
      if type(cfg.args) == 'table' and #cfg.args > 0 then
        args_line = table.concat(cfg.args, ' ')
      else
        args_line = '<none>'
      end
      prompt = prompt
        .. '\n\nRe-launching with:'
        .. '\n  agent: ' .. cfg.agent
        .. '\n  args:  ' .. args_line
      -- Show the session id only on the resume path — it's the load-
      -- bearing detail there. Hiding it on the new-session path avoids
      -- confusing the user into thinking the prior id will carry over.
      if not new_session and cfg.session_id and cfg.session_id ~= '' then
        local resume_line = cfg.session_id
        local age = session_age_hint(cfg.agent, cfg.session_id)
        if age then resume_line = resume_line .. '  ' .. age end
        prompt = prompt .. '\n  resume: ' .. resume_line
      end
    end
    local ans = vim.fn.confirm(prompt, '&Yes\n&No\n&Rename', 2)
    if ans ~= 1 and ans ~= 3 then return end

    local rename_to
    if ans == 3 then
      rename_to = pair_rename_prompt(pair_tag())
      if not rename_to then return end
    end

    local argv = { 'pair', 'restart' }
    if new_session then table.insert(argv, '--new-session') end
    if rename_to then
      table.insert(argv, '--rename-to')
      table.insert(argv, rename_to)
    end
    vim.fn.system(argv)
  end)
end

function _G.PairConfirmRestart()           pair_confirm_restart_impl(false) end
function _G.PairConfirmRestartNewSession() pair_confirm_restart_impl(true)  end
7e49622 (HEAD -> 000144-reject-codex-subagent-sessions-during-pair-identity-discovery) workshop: #144: record root session verification
f16d9af nvim: #144: stop guessing live Codex sessions
4e44e2e launcher: #144: quarantine invalid Codex resumes
71dd5c8 sessionwatch: #144: persist only root Codex sessions
6b23e57 session: #144: reject live Codex subagents
dab0bfb transcript: #144: classify root Codex sessions

exec
/bin/zsh -lc "sed -n '145,180p' atlas/review-workbench.md; rg -n \"live Codex rollout|live_codex|config-<tag>-<agent>|Codex.*rollout|filename\" atlas README.md | head -200; git diff 356ec01f5aaf6c48d3fcf73a7aff1bd090c8dc79..HEAD -- README.md atlas; rg -n \"SessionID\\(\" --glob '*.go'; rg -n \"Resolve\\(\\\"codex\\\"|Resolve\\(agent|ReadCodexRootSessionID|CodexSessionIDFromPath\" cmd --glob '*.go'" in /private/tmp/pair-context-usage
 succeeded in 0ms:
- `bin/pair-review-open <file>` — validates + spawns the **full-screen** floating pane
  (`zellij run --floating --close-on-exit --name review --width 100% --height 100%`;
  percentage dims, not `tput`, which measured the wrong pane), replacing any live
  review (single pane).
- `:PairReview <file>` (in draft `nvim/init.lua`, `complete=file`) — proposes the
  review target. It writes `review-target-<tag>.json` with `status=proposed`,
  runs `pair-review-readiness --prepare <file>` locally for deterministic
  start-up work (track file / create or resume `review/<slug>` / mark target
  `ready`), then sends the agent only a concise "review prepared; ack" message.
  It does **not** open the pane; Alt+c opens it once ready.
- **Alt+c** (`zellij/config.kdl`) — routed through the draft nvim like Alt+d
  (`MoveFocus Down` → `<C-\><C-n>` → `:lua PairReviewToggle()`), **not** a spawned
  shell pane. The draft's `PairReviewToggle()` (`nvim/init.lua`) branches on the
  state-file liveness and review-target status: live review → flip visibility from
  this *tiled* draft (`are-floating-panes-visible` → `show`/`hide-floating-panes`;
  **never** `toggle-floating-panes`); no live review + ready target → open;
  proposed target → "prep in progress"; no target → drop into `:PairReview `
  (file-select). Pure decision `_pair_review_toggle_action(alive, visible, status)`.
  Review-targets are scoped to the current conversation id so fresh sessions ignore
  stale targets while resumed sessions keep their in-progress target. Resolution is
  `PAIR_SESSION_ID` → `config-<tag>-<agent>.json` → live Codex rollout via
  `agent-pid-<tag>`; Codex/agy learn ids asynchronously, so review target handling must
  not rely on the launch-time env alone. `Alt+r` is deliberately free inside the review
  pane for reject.
- `nvim/pair_poke.lua` — id-based agent poke: relative `move-focus` does NOT escape a
  floating pane, so it resolves the agent pane from `list-panes --json` and writes
  directly with `write-chars --pane-id <agent>` + `send-keys --pane-id <agent> "Alt Enter"`.
  The review pane keeps focus while the agent receives the poke.
- **review-mode bar** (`nvim/init.lua`, `do`-block; `_pair_review_bar` count source +
  `_pair_review_segment` cached segment) — while a review is open, the draft's
  **statusline** carries `-H < pos > +Q • 🪄 <Mode> • <file> •     🤖 A/H`: `H` is
  prompt history count, `Q` is future queue count, `A` is agent/robot review rounds,
  and `H` after the slash is human review rounds. `pair_compose_statusline` swaps the
  cached segment in for the rightmost cheatsheet and right-aligns the review-round count,
  so review mode is visible even when the pane is hidden. A 1.5s timer recomputes the segment (counts parsed from `git log` round
  subjects, **branch-scoped** to the active `review/<slug>` so other docs' shipped reviews
README.md:365:`config-<tag>-<agent>.json` files keep priority over repo-agent defaults, so
atlas/how-to-bring-up-a-new-harness-cli.md:67:- When captured, the watcher writes `{ "agent": "<agent>", "args": [...], "session_id": "<uuid>" }` into `config-<tag>-<agent>.json`.
atlas/how-to-bring-up-a-new-harness-cli.md:86:**Telemetry Signal** (aspect `3`, see §3): `session-id` from `pair-session-watch` — `fired` when `AgentSpec.Match` resolves an id and the config is written, **`near-miss`** when a file matching the watch pattern is found but no id can be extracted (filename/format drift), `fail` when the 60s watch window elapses with no id at all (the session file never appeared where expected). The resume mapping in `bin/pair-shell` is the *consumer* of this id; it's static config with no separate signal.
atlas/review-workbench.md:165:  `PAIR_SESSION_ID` → `config-<tag>-<agent>.json` → live Codex rollout via
atlas/architecture.md:223:`config-<tag>-<agent>.json` remains the tag/native-session cache and keeps
atlas/architecture.md:271:A leading `pair resume <tag>` is recognized as a subcommand verb (alongside `list` / `help`): it skips both the picker and the name prompt, attaches if the tag's current-scope public zellij session still exists in any state, otherwise creates with that repo-local tag. When `resume` is in play, the agent is inferred from the scoped tag ledger, with `agent-<tag>` and config filenames kept as compatibility caches — so a single tag is enough to restart, regardless of which agent was originally paired with it. See "Tag-restart" below.
atlas/architecture.md:304:**Saved-config resolution & legacy Codex migration (#67).** `resolve_config_file <tag> <agent>` resolves the canonical `config-<tag>-<agent>.json`. Older Codex sessions on disk use a doubled shape `config-<tag>-codex-codex.json`; when the canonical file is absent and the agent is `codex`, the helper migrates the legacy file to the canonical name *iff* its JSON declares `"agent":"codex"` — a narrow, agent-checked compatibility path, **not** a glob resolver, so unrelated stale files can't silently win (`ARCH-DRY`, `ARCH-PURE`). It is used only where both tag and agent are known (restart-marker read, cleanup resume hint, the tag-restart picker that surfaces native Codex resume, and the two config writes); the agent-inference glob loop is deliberately left alone, since it is *discovering* the agent and already sees the legacy filename.
atlas/architecture.md:312:**Cleanup on quit.** zellij is run as a child (not `exec`) so the launcher resumes when zellij exits. On resume it checks for `~/.cache/pair/quit-<session>` (the marker that `pair quit` writes when Alt+x fires) and, if present, runs `zellij delete-session --force <session>` to clear the resurrect entry. It then SIGKILLs any leftover children that didn't follow the session down: a lingering `zellij --server` (rare but seen), and `nvim --embed` orphans (every `nvim FILE` is internally TUI parent + embed child; the embed sometimes survives RPC-pipe EOF and gets reparented to launchd). The embed reap is two-layered — primary path reads `nvim-pid-<tag>-{draft,scrollback}` files written by VimEnter autocmds inside `nvim/init.lua` and `nvim/scrollback.lua` (so the embed pid is known deterministically); fallback is a tag-scoped `pkill -f`. If a `config-<tag>-<agent>.json` was captured during the session, it also prints a one-liner naming the resume command (`pair resume <session>`) so the user can pick the work back up later. No marker → leave the session as zellij left it (running if Alt+d detached).
atlas/architecture.md:681:`*` is preserved across navigation: when leaving `*`, its content is autosaved, so navigating into history/queue and back never destroys the draft. Sending from `-N` preserves `*` (the "clear the draft" semantic of `Alt+Return` only fires when the source slot was `*`). **Sending from `+N` while `*` holds an in-progress draft parks that draft as a queue item (`push_front`) before shipping the selected item** — so `*` ends up empty (sent item's stickies + a fresh line) and the WIP survives as the new `+1`, rather than dangling at `*`. The selected item is resolved by its filename **key captured before** the enqueue, never by the display index: the `push_front` shifts every index by one, and removing by a stale index is what previously left the sent item in *both* `+N` and `-1` (duplication). Regression-guarded by `tests/queue-send-test.sh` (`make test-queue`). Empty / comment-only drafts have nothing to park, so that case is unchanged.
atlas/architecture.md:709:  `config-<tag>-<agent>.json` (session_id) + the per-agent path, and parses each
atlas/architecture.md:829:  fs_event (macOS FSEvents from nvim is unreliable: EMFILE/nil-filename; the
atlas/architecture.md:844:  any resume) → the per-tag `config-<tag>-<agent>.json` `session_id` (the
atlas/architecture.md:867:- **Alt+x** — full quit. Kills the session AND removes the resurrect entry. After Alt+x, the session is fully gone (but the `config-<tag>-<agent>.json` survives, so `pair resume <tag>` later replays the saved launch args + agent session id).
atlas/architecture.md:868:- **Alt+n** — reload pair. Kills the session AND keeps the saved `config-<tag>-<agent>.json` AND re-launches pair on the same tag with the same agent + args + agent session: the conversation resumes via `--resume <id>` (claude) or `resume <id>` (codex) or `--conversation <id>` (agy). Pair itself is the only thing that restarts — useful after a binary or config rebuild. For Codex, `pair restart` also makes one pre-kill live recovery pass through `agent-pid-<tag>` + descendant `lsof` and carries the recovered session id in the restart marker, so a missed watcher/config write does not force a fresh conversation.
atlas/architecture.md:887:   - `--resume <id>` / `resume <id>` / `--conversation <id>` explicit on argv: pair writes `config-<tag>-<agent>.json` directly with that id, before zellij launch.
atlas/architecture.md:901:| claude | `~/.claude/projects/<encoded-cwd>/<id>.jsonl` | filename | `--session-id` pre-injected by the launcher (deterministic) |
atlas/architecture.md:902:| codex | `~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<id>.jsonl` | trailing UUID in filename (regex) | `lsof -p <pid>` against agent PID + `ps`-discovered descendants, birth-time fallback |
atlas/architecture.md:903:| agy | `~/.gemini/antigravity-cli/conversations/<id>.db` | UUID database filename | `lsof -p <pid>` against agent PID + `ps`-discovered descendants, birth-time fallback |
atlas/architecture.md:905:**Stored shape.** `$PAIR_DATA_DIR/config-<tag>-<agent>.json`:
atlas/architecture.md:913:**Create-flow prompt (the launcher).** When the create path commits a tag, the launcher reads `config-<tag>-<agent>.json`. If present, it runs the per-agent stale-id check (claude: `[ -f .../<id>.jsonl ]`; codex: `find ~/.codex/sessions -name "*<id>*"`; agy: check conversation DB) and fzf-prompts the user with up to three options:
atlas/architecture.md:946:1. **Tag-only families** (filename is `<prefix>-<tag>[<ext>]`, no further structure): `agent`, `agent-pid`, `agent-output`, `agent-picks`, `outer-tty`, `pair-wrap-pid`, `title-pid`, `layout-mode`, `workbench-layout`, `queue` (dir), `quote`, `image-capture` + `.done`, `draft-<tag>.md`, `log-<tag>.md`, `nvim-pid-<tag>-{draft,scrollback}`.
atlas/architecture.md:947:2. **Per-(tag, agent) families** anchored on `config-<tag>-<agent>.json` — also `pane-<tag>-<agent>.json` (#71 frame-meter pane id + cwd), `scrollback-<tag>-<agent>.{ansi,raw,viewport,events.jsonl}` and the per-agent draft `draft-<tag>-<agent>.md`. The set of agent suffixes is hardcoded (`claude codex agy`) — adding a new agent to pair requires updating that list in lockstep.
atlas/architecture.md:949:**Substring safety is enforced by construction**, never by filtering. The enumerator computes exact filenames like `$DD/config-$old-claude.json`; it never globs `$DD/config-$old-*.json`. This is why `pair rename brain newname` cannot accidentally pick up `brain-2`'s files — the `brain-2`'s filenames are never constructed.
atlas/architecture.md:973:Internal: `~/.cache/pair/restart-<session>` — marker written alongside `quit-` by `pair restart` (Alt+n, plus the independent compaction flow). Holds `tag`, `agent`, optional `session_id`, and restart metadata as `key=value` lines so the launcher can reconstruct the relaunch params after cleanup has wiped `agent-<tag>`. Plain Codex restarts can fill `session_id` from the live rollout transcript before the pane is killed; the restart planner prefers that marker id over saved config because it is the freshest source. Removed when the in-process restart loop consumes it.
atlas/architecture.md:979:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/config-<tag>-<agent>.json` — saved restart configuration for `(tag, agent)` (issue #000016, #000020). `{ agent, args, session_id }`. For claude, written synchronously by the launcher before zellij launch (`--session-id` is deterministic). For codex/agy, written by the Go `pair-session-watch` command once the agent's session file is discovered via lsof. Read by the launcher's create-flow prompt and by the post-Alt+x hint. Survives Alt+x (unlike `agent-<tag>`, which is cleared) — that's the whole point: it's the bridge between two pair launches against the same tag.
atlas/architecture.md:989:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/slug-proposed-<tag>` and `slug-<tag>` — the orientation-slug channel (issue #000027). `pair-slug` (spawned by pair-wrap at turn-end) writes the proposed `=== <branch> | <focus> ===` to `slug-proposed-<tag>` (atomic temp+rename); nvim applies it to draft line 1 and writes the effective line back to `slug-<tag>`, which is the `prev` the proposer reads next turn. For Codex, if `config-<tag>-codex.json` is missing, `pair-slug` can recover the live rollout by reading `agent-pid-<tag>`, walking descendants via `ps`, and checking their `lsof` paths for `~/.codex/sessions/.../rollout-*.jsonl`. Agy has two artifacts: restart/session discovery uses `~/.gemini/antigravity-cli/conversations/<session_id>.db`, while transcript summarization still reads `~/.gemini/antigravity-cli/brain/<session_id>/.system_generated/logs/transcript.jsonl`. Codex model auth is API-key first, then Codex CLI subscription auth via `codex exec`. Single writer each, so the channel is race-free.
atlas/architecture.md:993:**Migration from v1:** the launcher detects old `~/scratch/pair-{draft,log}-*.md` files on startup and moves them to the new XDG location, stripping the redundant `pair-` prefix from filenames.
atlas/session-identity.md:30:config-<tag>-<agent>.json
atlas/session-identity.md:78:A session name is a **socket filename**. On the machine this was measured on,
atlas/session-identity.md:126:### Session names are also filename components
atlas/session-identity.md:139:`agent-<tag>` and `config-<tag>-<agent>.json` files remain as derived caches and
atlas/session-identity.md:144:A Codex rollout filename supplies only a candidate UUID; it does not prove
atlas/session-identity.md:160:Neovim deliberately does not inspect Codex processes or rollouts. Review
atlas/session-identity.md:165:`agent-default-<agent>.json` is different from `config-<tag>-<agent>.json`: it
diff --git a/atlas/session-identity.md b/atlas/session-identity.md
index 8c1fe72..c8411c5 100644
--- a/atlas/session-identity.md
+++ b/atlas/session-identity.md
@@ -139,6 +139,29 @@ The ledger is the source of truth for agent/config inference. The older
 `agent-<tag>` and `config-<tag>-<agent>.json` files remain as derived caches and
 compatibility surfaces for existing consumers.
 
+### Codex root identity
+
+A Codex rollout filename supplies only a candidate UUID; it does not prove
+which conversation owns the rollout. Pair authorizes an automatic Codex
+identity only when the rollout's first JSONL event is a matching
+`session_meta`, its `parent_thread_id` is absent or null, and its source is the
+observed root source `cli` or `exec`. Subagent, malformed, mismatched, unknown,
+oversized, and incomplete first events fail closed. Candidate scans continue
+past rejected rollouts so an open subagent cannot hide a later root candidate.
+
+The rule lives in `cmd/internal/transcript` and is shared by launcher live
+capture, session watching, context usage, slugging, and review targeting.
+Process-tree and birth-time discovery locate candidates only; neither grants
+identity by itself. Persisted Codex IDs are revalidated at automatic config
+picker and `Alt+n` restart boundaries. An invalid binding is removed from the
+config, its non-resume args are preserved for a fresh launch, and the operator
+is warned. Explicitly typed `codex resume <id>` remains user authority.
+
+Neovim deliberately does not inspect Codex processes or rollouts. Review
+target scoping uses the inherited `PAIR_SESSION_ID`, then Pair's config cache;
+when neither exists it remains unscoped until the Go watcher publishes a
+validated root identity.
+
 `agent-default-<agent>.json` is different from `config-<tag>-<agent>.json`: it
 has only `{agent,args}` and belongs to the repo/agent, not to a work tag or
 native conversation. Fresh `pair <agent>` creates use it as the lowest-priority
cmd/internal/codexsid/codexsid_test.go:10:	if got := ResolveSessionID(t.TempDir(), "tag"); got != "" {
cmd/internal/codexsid/codexsid_test.go:17:	if got := ResolveSessionID(dir, "tag"); got != "" {
cmd/internal/codexsid/codexsid_test.go:54:	if got := ResolveSessionID(dataDir, "tag"); got != rootSID {
cmd/internal/codexsid/codexsid.go:21:func ResolveSessionID(dataDir, tag string) string {
cmd/internal/codexsid/codexsid.go:32:			if sid := transcript.ReadCodexRootSessionID(name); sid != "" {
cmd/internal/opener/opener_test.go:61:func TestResolveSessionID(t *testing.T) {
cmd/internal/opener/opener_test.go:65:	if got := resolveSessionID(A, []byte(`{"session_id":"`+C+`"}`)); got != A {
cmd/internal/opener/opener_test.go:69:	if got := resolveSessionID("", []byte(`{"agent":"claude","session_id":"`+C+`"}`)); got != C {
cmd/internal/opener/opener_test.go:73:	if got := resolveSessionID("", nil); got != "" {
cmd/internal/opener/opener_test.go:76:	if got := resolveSessionID("", []byte(`{"agent":"claude"}`)); got != "" {
cmd/internal/opener/opener_test.go:79:	if got := resolveSessionID("", []byte(`not json`)); got != "" {
cmd/internal/opener/opener.go:106:func resolveSessionID(envSID string, configJSON []byte) string {
cmd/internal/contextcmd/contextcmd.go:55:	sid := transcript.SessionID(dataDir, tag, agent, env.Home)
cmd/internal/opener/run.go:137:			sid = resolveSessionID("", []byte(cfg))
cmd/internal/sessionwatch/sessionwatch_test.go:9:func TestAgentSpecExtractsCodexSessionID(t *testing.T) {
cmd/internal/sessionwatch/sessionwatch_test.go:24:func TestAgentSpecExtractsAgySessionID(t *testing.T) {
cmd/internal/sessionwatch/sessionwatch_test.go:59:func TestMuseMatchExtractsRootSessionID(t *testing.T) {
cmd/internal/sessionwatch/run.go:291:	if err != nil || transcript.CodexRootSessionID(result.Path, firstEvent) != result.ID {
cmd/internal/transcript/transcript_test.go:39:func TestCodexRootSessionID(t *testing.T) {
cmd/internal/transcript/transcript_test.go:62:			if got := CodexRootSessionID(tt.path, []byte(tt.event)); got != tt.want {
cmd/internal/transcript/transcript_test.go:87:	if got := ReadCodexRootSessionID(path); got != sid {
cmd/internal/transcript/transcript_test.go:91:	if got := ReadCodexRootSessionID(path); got != "" {
cmd/internal/transcript/transcript_test.go:95:	if got := ReadCodexRootSessionID(path); got != "" {
cmd/internal/transcript/transcript_test.go:99:	if got := ReadCodexRootSessionID(path); got != "" {
cmd/internal/transcript/transcript_test.go:114:	if got := ReadCodexRootSessionID(path); got != sid {
cmd/internal/transcript/transcript_test.go:118:	if got := ReadCodexRootSessionID(path); got != "" {
cmd/internal/transcript/transcript_test.go:122:	if got := ReadCodexRootSessionID(filepath.Join(home, "missing.jsonl")); got != "" {
cmd/internal/transcript/transcript_test.go:125:	if got := ReadCodexRootSessionID(home); got != "" {
cmd/internal/transcript/transcript_test.go:154:	if got := SessionID(data, "work", "codex", home); got != rootSID {
cmd/internal/transcript/transcript_test.go:160:	if got := SessionID(data, "work", "codex", home); got != "" {
cmd/internal/transcript/transcript_test.go:167:	if got := SessionID(data, "work", "claude", home); got != "claude-id" {
cmd/internal/transcript/transcript.go:37:func CodexRootSessionID(path string, firstEvent []byte) string {
cmd/internal/transcript/transcript.go:63:func ReadCodexRootSessionID(path string) string {
cmd/internal/transcript/transcript.go:68:	return CodexRootSessionID(path, line)
cmd/internal/transcript/transcript.go:90:func SessionID(dataDir, tag, agent, home string) string {
cmd/internal/transcript/transcript.go:103:		if path == "" || ReadCodexRootSessionID(path) != c.SessionID {
cmd/internal/launcher/agentargs.go:196:func shouldMintClaudeSessionID(agent, explicitResume string, agentExtra []string) bool {
cmd/internal/slugcmd/slugcmd.go:94:			if strings.HasPrefix(name, prefix) && transcriptpkg.ReadCodexRootSessionID(name) != "" {
cmd/internal/slugcmd/slugcmd.go:131:		sid := transcriptpkg.SessionID(dataDir, tag, agent, home)
cmd/internal/launcher/osruntime.go:603:		return path != "" && transcript.ReadCodexRootSessionID(path) == sid
cmd/internal/launcher/osruntime.go:610:func (r OSRuntime) LiveAgentSessionID(agent, tag string) string {
cmd/internal/launcher/osruntime.go:628:			if sid := transcript.ReadCodexRootSessionID(name); sid != "" {
cmd/internal/launcher/agentargs_test.go:141:func TestShouldMintClaudeSessionID(t *testing.T) {
cmd/internal/launcher/agentargs_test.go:142:	if !shouldMintClaudeSessionID("claude", "", nil) {
cmd/internal/launcher/agentargs_test.go:145:	if shouldMintClaudeSessionID("codex", "", nil) {
cmd/internal/launcher/agentargs_test.go:148:	if shouldMintClaudeSessionID("claude", "resumed-sid", nil) {
cmd/internal/launcher/agentargs_test.go:151:	if shouldMintClaudeSessionID("claude", "", []string{"--session-id", "u"}) {
cmd/internal/launcher/agentargs_test.go:154:	if shouldMintClaudeSessionID("claude", "", []string{"--fork-session"}) {
cmd/internal/launcher/lifecycle_test.go:175:func TestRunLaunchRestartLoopAltNCodexUsesMarkerSessionID(t *testing.T) {
cmd/internal/launcher/osruntime_test.go:211:	if got := rt.LiveAgentSessionID("codex", "work"); got != rootSID {
cmd/internal/launcher/osruntime_test.go:214:	if got := rt.LiveAgentSessionID("claude", "work"); got != "" {
cmd/internal/launcher/runtime.go:123:	LiveAgentSessionID(agent, tag string) string
cmd/internal/reviewcmd/run_test.go:83:func (f *fakeRuntime) ResolveCodexSessionID(dataDir, tag string) string { return f.codexSID }
cmd/internal/reviewcmd/run_test.go:84:func (f *fakeRuntime) ConfiguredSessionID(dataDir, tag, agent string) string {
cmd/internal/launcher/markers_test.go:21:func TestSerializeRestartMarkerCarriesSessionID(t *testing.T) {
cmd/internal/launcher/createflow.go:410:	if shouldMintClaudeSessionID(agent, explicitResume, agentArgs) {
cmd/internal/reviewcmd/runtime.go:100:func (OSRuntime) ResolveCodexSessionID(dataDir, tag string) string {
cmd/internal/reviewcmd/runtime.go:101:	return codexsid.ResolveSessionID(dataDir, tag)
cmd/internal/reviewcmd/runtime.go:104:func (OSRuntime) ConfiguredSessionID(dataDir, tag, agent string) string {
cmd/internal/reviewcmd/runtime.go:106:	return transcript.SessionID(dataDir, tag, agent, home)
cmd/internal/launcher/restart.go:34:		sessionID = rt.LiveAgentSessionID(agent, tag)
cmd/internal/reviewcmd/run.go:42:	ResolveCodexSessionID(dataDir, tag string) string
cmd/internal/reviewcmd/run.go:44:	ConfiguredSessionID(dataDir, tag, agent string) string
cmd/internal/reviewcmd/run.go:116:	if sid := rt.ConfiguredSessionID(dataDir, tag, agent); sid != "" {
cmd/internal/reviewcmd/run.go:120:		return rt.ResolveCodexSessionID(dataDir, tag)
cmd/internal/reviewcmd/reviewcmd_test.go:57:	if got := NewOSRuntime().ConfiguredSessionID(data, "t", "codex"); got != "" {
cmd/internal/launcher/createflow_test.go:215:func (f *fakeRuntime) LiveAgentSessionID(agent, tag string) string {
cmd/internal/launcher/restart_test.go:34:func TestRunRestartCapturesLiveCodexSessionID(t *testing.T) {
cmd/internal/contextcmd/contextcmd.go:60:	return transcript.Resolve(agent, sid, cwd, env.Home)
cmd/internal/transcript/transcript_test.go:28:func TestCodexSessionIDFromPath(t *testing.T) {
cmd/internal/transcript/transcript_test.go:31:	if got := CodexSessionIDFromPath(path); got != sid {
cmd/internal/transcript/transcript_test.go:32:		t.Fatalf("CodexSessionIDFromPath = %q, want %q", got, sid)
cmd/internal/transcript/transcript_test.go:34:	if got := CodexSessionIDFromPath("/tmp/not-codex.jsonl"); got != "" {
cmd/internal/transcript/transcript_test.go:69:func TestReadCodexRootSessionIDBoundaries(t *testing.T) {
cmd/internal/transcript/transcript_test.go:87:	if got := ReadCodexRootSessionID(path); got != sid {
cmd/internal/transcript/transcript_test.go:91:	if got := ReadCodexRootSessionID(path); got != "" {
cmd/internal/transcript/transcript_test.go:95:	if got := ReadCodexRootSessionID(path); got != "" {
cmd/internal/transcript/transcript_test.go:99:	if got := ReadCodexRootSessionID(path); got != "" {
cmd/internal/transcript/transcript_test.go:114:	if got := ReadCodexRootSessionID(path); got != sid {
cmd/internal/transcript/transcript_test.go:118:	if got := ReadCodexRootSessionID(path); got != "" {
cmd/internal/transcript/transcript_test.go:122:	if got := ReadCodexRootSessionID(filepath.Join(home, "missing.jsonl")); got != "" {
cmd/internal/transcript/transcript_test.go:125:	if got := ReadCodexRootSessionID(home); got != "" {
cmd/internal/transcript/transcript.go:24:// CodexSessionIDFromPath extracts the native session id from a Codex rollout
cmd/internal/transcript/transcript.go:26:func CodexSessionIDFromPath(path string) string {
cmd/internal/transcript/transcript.go:38:	sid := CodexSessionIDFromPath(path)
cmd/internal/transcript/transcript.go:60:// ReadCodexRootSessionID reads one bounded, newline-terminated JSONL event and
cmd/internal/transcript/transcript.go:63:func ReadCodexRootSessionID(path string) string {
cmd/internal/transcript/transcript.go:102:		path := Resolve(agent, c.SessionID, "", home)
cmd/internal/transcript/transcript.go:103:		if path == "" || ReadCodexRootSessionID(path) != c.SessionID {
cmd/internal/transcript/transcript.go:112:func Resolve(agent, sid, cwd, home string) string {
cmd/internal/codexsid/codexsid.go:32:			if sid := transcript.ReadCodexRootSessionID(name); sid != "" {
cmd/internal/slugcmd/slugcmd.go:94:			if strings.HasPrefix(name, prefix) && transcriptpkg.ReadCodexRootSessionID(name) != "" {
cmd/internal/slugcmd/slugcmd.go:133:			transcript = transcriptpkg.Resolve(agent, sid, cwd, home)
cmd/internal/launcher/osruntime.go:602:		path := transcript.Resolve("codex", sid, cwd, home)
cmd/internal/launcher/osruntime.go:603:		return path != "" && transcript.ReadCodexRootSessionID(path) == sid
cmd/internal/launcher/osruntime.go:628:			if sid := transcript.ReadCodexRootSessionID(name); sid != "" {
cmd/internal/slugcmd/slugcmd_test.go:13:	if got := transcript.CodexSessionIDFromPath(path); got == "" {
cmd/internal/slugcmd/slugcmd_test.go:14:		t.Fatalf("CodexSessionIDFromPath did not match %q", path)

codex
The implementation centralizes the root classifier correctly, but the shadow sweep is incomplete: stale filename-only semantics remain documented in multiple atlas sections, and at least one Go config consumer (`opener` changelog keying) still accepts a Codex config ID without the shared classifier. I’m running tests and tracing whether that consumer falls within the issue’s “every automatic identity/config reader” contract.
exec
/bin/zsh -lc 'env -u PAIR_SESSION_ID -u PAIR_AGENT -u PAIR_TAG -u PAIR_DATA_DIR go test ./... -count=1' in /private/tmp/pair-context-usage
exec
/bin/zsh -lc 'env -u PAIR_SESSION_ID -u PAIR_AGENT -u PAIR_TAG -u PAIR_DATA_DIR GOCACHE=/tmp/pair-review-gocache go test ./... -count=1' in /private/tmp/pair-context-usage
 exited 1 in 24647ms:
FAIL	github.com/xianxu/pair/cmd/internal/dispatcher [setup failed]
# github.com/xianxu/pair/cmd/internal/dispatcher
cmd/internal/dispatcher/dispatcher.go:15:2: open /Users/xianxu/Library/Caches/go-build/06/063926919f912e03d4103c8d0ac6dc3f2dfb04bcc0bb42e01110528a29af3d13-a: operation not permitted
# github.com/xianxu/pair/cmd/pair-go
cmd/internal/dispatcher/dispatcher.go:15:2: open /Users/xianxu/Library/Caches/go-build/06/063926919f912e03d4103c8d0ac6dc3f2dfb04bcc0bb42e01110528a29af3d13-a: operation not permitted
# ./...
pattern ./...: open /Users/xianxu/Library/Caches/go-build/06/063926919f912e03d4103c8d0ac6dc3f2dfb04bcc0bb42e01110528a29af3d13-a: operation not permitted
FAIL	github.com/xianxu/pair/cmd/pair-go [setup failed]
FAIL	./... [setup failed]
ok  	github.com/xianxu/pair/cmd/internal/adapt	0.297s
ok  	github.com/xianxu/pair/cmd/internal/agentcmd	0.502s
ok  	github.com/xianxu/pair/cmd/internal/ansi	0.726s
ok  	github.com/xianxu/pair/cmd/internal/changelogcmd	23.944s
ok  	github.com/xianxu/pair/cmd/internal/clipcmd	0.807s
ok  	github.com/xianxu/pair/cmd/internal/codexsid	6.178s
ok  	github.com/xianxu/pair/cmd/internal/contextcmd	15.011s
ok  	github.com/xianxu/pair/cmd/internal/continuationcmd	2.192s
ok  	github.com/xianxu/pair/cmd/internal/ctxmeter	1.850s
ok  	github.com/xianxu/pair/cmd/internal/draftroute	2.053s
ok  	github.com/xianxu/pair/cmd/internal/entrypoint	2.260s
ok  	github.com/xianxu/pair/cmd/internal/keyhelp	2.473s
ok  	github.com/xianxu/pair/cmd/internal/keyscmd	9.680s
ok  	github.com/xianxu/pair/cmd/internal/launcher	17.160s
ok  	github.com/xianxu/pair/cmd/internal/layoutcmd	2.633s
ok  	github.com/xianxu/pair/cmd/internal/model	9.329s
ok  	github.com/xianxu/pair/cmd/internal/opener	2.399s
ok  	github.com/xianxu/pair/cmd/internal/osfs	2.395s
ok  	github.com/xianxu/pair/cmd/internal/procutil	2.559s
ok  	github.com/xianxu/pair/cmd/internal/readiness	2.623s
ok  	github.com/xianxu/pair/cmd/internal/runtimebundle	2.639s
?   	github.com/xianxu/pair/cmd/internal/runtimebundle/generatecmd	[no test files]
?   	github.com/xianxu/pair/cmd/internal/runtimebundle/manifestmodel	[no test files]
ok  	github.com/xianxu/pair/cmd/internal/runtimebundlegen	9.416s
ok  	github.com/xianxu/pair/cmd/internal/scribecmd	3.643s
ok  	github.com/xianxu/pair/cmd/internal/scrollbackcmd	3.629s
ok  	github.com/xianxu/pair/cmd/internal/sessionwatch	3.557s
ok  	github.com/xianxu/pair/cmd/internal/slugcmd	11.092s
ok  	github.com/xianxu/pair/cmd/internal/termcmd	4.156s
ok  	github.com/xianxu/pair/cmd/internal/textwidth	4.060s
ok  	github.com/xianxu/pair/cmd/internal/titlepoller	4.040s
ok  	github.com/xianxu/pair/cmd/internal/transcript	4.431s
ok  	github.com/xianxu/pair/cmd/internal/workbenchshortcut	4.434s
?   	github.com/xianxu/pair/cmd/internal/workbenchshortcut/generatecmd	[no test files]
--- FAIL: TestPublishAgentReadySkipsWhenPairEnvIncomplete (0.00s)
    readiness_test.go:50: ready file stat err = <nil>, want not exist
FAIL
FAIL	github.com/xianxu/pair/cmd/internal/wrapcmd	7.277s
ok  	github.com/xianxu/pair/cmd/internal/zellijpane	4.053s
FAIL

 exited 1 in 22047ms:
ok  	github.com/xianxu/pair/cmd/internal/adapt	2.492s
ok  	github.com/xianxu/pair/cmd/internal/agentcmd	2.628s
ok  	github.com/xianxu/pair/cmd/internal/ansi	2.769s
ok  	github.com/xianxu/pair/cmd/internal/changelogcmd	20.612s
ok  	github.com/xianxu/pair/cmd/internal/clipcmd	2.876s
ok  	github.com/xianxu/pair/cmd/internal/codexsid	11.037s
ok  	github.com/xianxu/pair/cmd/internal/contextcmd	15.587s
ok  	github.com/xianxu/pair/cmd/internal/continuationcmd	3.966s
ok  	github.com/xianxu/pair/cmd/internal/ctxmeter	3.417s
ok  	github.com/xianxu/pair/cmd/internal/dispatcher	3.783s
ok  	github.com/xianxu/pair/cmd/internal/draftroute	3.868s
ok  	github.com/xianxu/pair/cmd/internal/entrypoint	3.985s
ok  	github.com/xianxu/pair/cmd/internal/keyhelp	4.185s
ok  	github.com/xianxu/pair/cmd/internal/keyscmd	10.699s
ok  	github.com/xianxu/pair/cmd/internal/launcher	14.762s
ok  	github.com/xianxu/pair/cmd/internal/layoutcmd	4.273s
ok  	github.com/xianxu/pair/cmd/internal/model	10.768s
ok  	github.com/xianxu/pair/cmd/internal/opener	4.203s
ok  	github.com/xianxu/pair/cmd/internal/osfs	4.257s
ok  	github.com/xianxu/pair/cmd/internal/procutil	4.150s
ok  	github.com/xianxu/pair/cmd/internal/readiness	4.187s
ok  	github.com/xianxu/pair/cmd/internal/reviewcmd	3.843s
ok  	github.com/xianxu/pair/cmd/internal/runtimebundle	3.907s
?   	github.com/xianxu/pair/cmd/internal/runtimebundle/generatecmd	[no test files]
?   	github.com/xianxu/pair/cmd/internal/runtimebundle/manifestmodel	[no test files]
ok  	github.com/xianxu/pair/cmd/internal/runtimebundlegen	7.790s
ok  	github.com/xianxu/pair/cmd/internal/scribecmd	2.992s
ok  	github.com/xianxu/pair/cmd/internal/scrollbackcmd	2.942s
ok  	github.com/xianxu/pair/cmd/internal/sessionwatch	2.882s
ok  	github.com/xianxu/pair/cmd/internal/slugcmd	6.074s
ok  	github.com/xianxu/pair/cmd/internal/termcmd	2.642s
ok  	github.com/xianxu/pair/cmd/internal/textwidth	2.723s
ok  	github.com/xianxu/pair/cmd/internal/titlepoller	2.145s
ok  	github.com/xianxu/pair/cmd/internal/transcript	2.217s
ok  	github.com/xianxu/pair/cmd/internal/workbenchshortcut	2.200s
?   	github.com/xianxu/pair/cmd/internal/workbenchshortcut/generatecmd	[no test files]
--- FAIL: TestPublishAgentReadySkipsWhenPairEnvIncomplete (0.00s)
    readiness_test.go:50: ready file stat err = <nil>, want not exist
FAIL
FAIL	github.com/xianxu/pair/cmd/internal/wrapcmd	3.227s
ok  	github.com/xianxu/pair/cmd/internal/zellijpane	1.622s
ok  	github.com/xianxu/pair/cmd/pair-go	6.614s
FAIL

exec
/bin/zsh -lc "sed -n '1,245p' tests/review-toggle-test.sh; sed -n '180,235p' cmd/internal/launcher/createflow_test.go; sed -n '990,1130p' cmd/internal/launcher/createflow_test.go; sed -n '1190,1280p' cmd/internal/launcher/createflow_test.go; nl -ba workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan.md | sed -n '12,58p'; nl -ba cmd/internal/launcher/markers.go | sed -n '55,72p'; nl -ba cmd/internal/opener/opener.go | sed -n '95,125p'; nl -ba cmd/internal/opener/run.go | sed -n '128,145p'" in /private/tmp/pair-context-usage
 succeeded in 0ms:
#!/usr/bin/env bash
# tests/review-toggle-test.sh — the Alt+c review/collaboration toggle, now a draft-nvim lua
# fn (#66 M3 rework; the old transient pair-review-toggle floating pane caused
# the open delay / auto-hide / half-size / mis-fire smoke bugs).
#
#   _pair_review_toggle_action(alive, visible) (pure):
#     not alive          → 'open'   (file-select)
#     alive  + visible   → 'hide'
#     alive  + hidden    → 'show'
#   PairReviewToggle() (integration, zellij stubbed on $PATH):
#     live state file + are-floating-panes-visible=true  → hide-floating-panes
#     live state file + are-floating-panes-visible=false → show-floating-panes
#     no state file → file-select (no visibility query, no show/hide)
#   and NEVER toggle-floating-panes (the footgun).
#
# Live zellij pane/focus behaviour is the manual smoke (M3 plan Task 5). Here
# zellij is a $PATH stub that records argv and answers are-floating-panes-visible
# from a file the driver rewrites between branches.
#
# Run: bash tests/review-toggle-test.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-toggle-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
RESULT="$RT/result.txt"; ZLOG="$RT/zlog.txt"; FLOATVIS="$RT/floatvis"; : > "$ZLOG"
fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

# zellij stub: record every action; answer are-floating-panes-visible from a file.
mkdir -p "$RT/bin"
cat > "$RT/bin/zellij" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$ZLOG"
if [ "\$1" = action ] && [ "\$2" = are-floating-panes-visible ]; then
  cat "$FLOATVIS" 2>/dev/null || echo false
fi
exit 0
EOF
cat > "$RT/bin/ps" <<'EOF'
#!/usr/bin/env bash
if [ "$1" = "-axo" ]; then
  printf '111 1\n222 111\n'
  exit 0
fi
exec /bin/ps "$@"
EOF
cat > "$RT/bin/lsof" <<'EOF'
#!/usr/bin/env bash
if [ "$1" = "-p" ] && [ "$2" = "222" ]; then
  printf 'p222\nn%s/.codex/sessions/2026/06/21/rollout-2026-06-21T00-00-00-12345678-1234-1234-1234-123456789abc.jsonl\n' "$HOME"
  exit 0
fi
printf 'p%s\n' "${2:-}"
EOF
chmod +x "$RT/bin/zellij" "$RT/bin/ps" "$RT/bin/lsof"

printf 'draft\n' > "$RT/draft.md"
cat > "$RT/driver.lua" <<'LUA'
local OUT = io.open(os.getenv('RESULT'), 'w')
local ZLOG = os.getenv('ZLOG')
local FLOATVIS = os.getenv('FLOATVIS')
local sf = vim.env.PAIR_DATA_DIR .. '/review-' .. vim.env.PAIR_TAG .. '.open'

local function read_zlog()
  local f = io.open(ZLOG, 'r'); if not f then return {} end
  local t = {}; for l in f:lines() do t[#t + 1] = l end; f:close(); return t
end
local function new_since(n)
  local all = read_zlog(); local out = {}
  for i = n + 1, #all do out[#out + 1] = all[i] end; return out
end
local function has(lines, pat)
  for _, l in ipairs(lines) do if l:find(pat, 1, true) then return true end end
  return false
end
local function setfloat(v) local f = io.open(FLOATVIS, 'w'); f:write(v); f:close() end

-- pure decision (5 cases: a live pane → hide/show; else target-driven prompt/open/wait)
local A = _G._pair_review_toggle_action
OUT:write((A(false, false, nil) == 'prompt') and 'pure-prompt ok\n' or 'pure-prompt FAIL\n')
OUT:write((A(false, false, 'ready') == 'open') and 'pure-open ok\n' or 'pure-open FAIL\n')
OUT:write((A(false, false, 'proposed') == 'wait') and 'pure-wait ok\n' or 'pure-wait FAIL\n')
OUT:write((A(true, true) == 'hide') and 'pure-hide ok\n' or 'pure-hide FAIL\n')
OUT:write((A(true, false) == 'show') and 'pure-show ok\n' or 'pure-show FAIL\n')

local R = _G._pair_review
local target = vim.env.PAIR_DATA_DIR .. '/review-target-' .. vim.env.PAIR_TAG .. '.json'
local draft = vim.env.PAIR_DATA_DIR .. '/draft.md' -- exists (the test wrote it)

-- conversation-scope (#66 smoke #6): a target written under a DIFFERENT session
-- (PAIR_SESSION_ID=oldsid, pre-written below) is ignored by this session (testsid),
-- so a fresh session prompts instead of reopening the previous review.
OUT:write((R.read_target() == nil) and 'session-scope ok\n' or 'session-scope FAIL\n')

local prepbin = vim.env.PAIR_DATA_DIR .. '/prep-ok'
vim.fn.writefile({
  '#!/usr/bin/env bash',
  'set -eu',
  '"' .. vim.env.PAIR_HOME .. '/bin/pair" review target "$2" ready >/dev/null',
  'printf "%s\\n" "review prepared: $2 on review/draft. Reply \\"ready\\"."',
}, prepbin)
vim.fn.system({ 'chmod', '+x', prepbin })
vim.env.PAIR_REVIEW_READINESS_BIN = prepbin
R.propose(draft)
local proposed = R.read_target()
OUT:write((proposed and proposed.status == 'ready') and 'propose-prepares-ready ok\n' or 'propose-prepares-ready FAIL\n')
vim.env.PAIR_REVIEW_READINESS_BIN = nil
vim.fn.writefile({ '{"file":"/stale/prev.md","status":"ready","session":"oldsid"}' }, target)

-- pure target_stale: same id → fresh; different / empty-current / no-id → stale.
local TS = R.target_stale
OUT:write((TS({ session = 'testsid' }, 'testsid') == false) and 'ts-same ok\n' or 'ts-same FAIL\n')
OUT:write((TS({ session = 'oldsid' }, 'testsid') == true) and 'ts-diff ok\n' or 'ts-diff FAIL\n')
OUT:write((TS({ session = 'x' }, '') == true) and 'ts-nocur ok\n' or 'ts-nocur FAIL\n')
OUT:write((TS({}, 'testsid') == true) and 'ts-noid ok\n' or 'ts-noid FAIL\n')

-- Codex fresh-start race: :PairReview may prepare before the async session
-- watcher has discovered a session id. The target was created by this same
-- draft nvim, so it must remain readable even though it is not yet session
-- stamped; otherwise the second Alt+c falls back to :PairReview again.
vim.env.PAIR_SESSION_ID = ''
vim.env.PAIR_AGENT = 'claude'
os.remove(vim.env.PAIR_DATA_DIR .. '/config-' .. vim.env.PAIR_TAG .. '-claude.json')
vim.fn.writefile({ '{"file":"' .. draft .. '","status":"ready","session":""}' }, target)
vim.fn.system({ 'touch', '-t', '202001010000', target })
OUT:write((R.read_target() == nil) and 'old-unscoped-target-stale ok\n' or 'old-unscoped-target-stale FAIL\n')
vim.fn.writefile({ '{"file":"' .. draft .. '","status":"ready","session":""}' }, target)
OUT:write((R.read_target() ~= nil) and 'fresh-unscoped-target-read ok\n' or 'fresh-unscoped-target-read FAIL\n')
vim.env.PAIR_SESSION_ID = 'testsid'
vim.fn.writefile({ '{"file":"/stale/prev.md","status":"ready","session":"oldsid"}' }, target)

-- codex/agy fresh sessions learn their id after nvim starts; review-target must
-- fall back to config-<tag>-<agent>.json when PAIR_SESSION_ID is empty.
vim.env.PAIR_SESSION_ID = ''
vim.fn.writefile({ '{"agent":"claude","args":[],"session_id":"cfgsid"}' },
  vim.env.PAIR_DATA_DIR .. '/config-' .. vim.env.PAIR_TAG .. '-' .. vim.env.PAIR_AGENT .. '.json')
vim.fn.writefile({ '{"file":"' .. draft .. '","status":"ready","session":"cfgsid"}' }, target)
OUT:write((R.read_target() ~= nil) and 'config-session-read ok\n' or 'config-session-read FAIL\n')
R.write_target(draft, 'ready')
local written = vim.json.decode(table.concat(vim.fn.readfile(target), '\n'))
OUT:write((written.session == 'cfgsid') and 'config-session-write ok\n' or 'config-session-write FAIL\n')
vim.env.PAIR_SESSION_ID = 'testsid'
vim.fn.writefile({ '{"file":"/stale/prev.md","status":"ready","session":"oldsid"}' }, target)

vim.env.PAIR_SESSION_ID = ''
vim.env.PAIR_AGENT = 'codex'
os.remove(vim.env.PAIR_DATA_DIR .. '/config-' .. vim.env.PAIR_TAG .. '-codex.json')
vim.fn.writefile({ '111' }, vim.env.PAIR_DATA_DIR .. '/agent-pid-' .. vim.env.PAIR_TAG)
vim.fn.writefile({ '{"file":"' .. draft .. '","status":"ready","session":"12345678-1234-1234-1234-123456789abc"}' }, target)
OUT:write((R.current_session_id() == nil) and 'no-live-codex-fallback ok\n' or 'no-live-codex-fallback FAIL\n')
OUT:write((R.read_target() == nil) and 'unverified-live-target-stale ok\n' or 'unverified-live-target-stale FAIL\n')
R.write_target(draft, 'ready')
written = vim.json.decode(table.concat(vim.fn.readfile(target), '\n'))
OUT:write((written.session == '') and 'unverified-live-target-unstamped ok\n' or 'unverified-live-target-unstamped FAIL\n')
vim.env.PAIR_AGENT = 'claude'
vim.env.PAIR_SESSION_ID = 'testsid'
vim.fn.writefile({ '{"file":"/stale/prev.md","status":"ready","session":"oldsid"}' }, target)

-- live + visible → hide  (state file holds OUR pid, so kill -0 says alive)
vim.fn.writefile({ tostring(vim.fn.getpid()) }, sf); setfloat('true')
local n = #read_zlog(); _G.PairReviewToggle()
local d = new_since(n)
OUT:write((has(d, 'action are-floating-panes-visible') and has(d, 'action hide-floating-panes'))
  and 'hide ok\n' or 'hide FAIL\n')

-- live + hidden → show
vim.fn.writefile({ tostring(vim.fn.getpid()) }, sf); setfloat('false')
n = #read_zlog(); _G.PairReviewToggle()
d = new_since(n)
OUT:write(has(d, 'action show-floating-panes') and 'show ok\n' or 'show FAIL\n')

-- no live pane, NO target → prompt: no open (zellij run), no show/hide
os.remove(sf); os.remove(target)
n = #read_zlog(); _G.PairReviewToggle()
d = new_since(n)
OUT:write((not has(d, 'run --floating') and not has(d, 'hide-floating-panes')
  and not has(d, 'show-floating-panes')) and 'prompt ok\n' or 'prompt FAIL\n')

-- no live pane, target READY → open the pane (pair review open → zellij run)
R.write_target(draft, 'ready')
n = #read_zlog(); _G.PairReviewToggle()
d = new_since(n)
OUT:write(has(d, 'run --floating') and 'targetopen ok\n' or 'targetopen FAIL\n')

-- no live pane, target PROPOSED → wait: do NOT open
R.write_target(draft, 'proposed')
n = #read_zlog(); _G.PairReviewToggle()
d = new_since(n)
OUT:write((not has(d, 'run --floating')) and 'wait ok\n' or 'wait FAIL\n')

-- footgun: never toggle-floating-panes anywhere
OUT:write(has(read_zlog(), 'toggle-floating-panes') and 'footgun FAIL\n' or 'footgun ok\n')
OUT:close()
vim.cmd('qa!')
LUA

# a STALE review-target from a DIFFERENT conversation (session=oldsid). This session
# runs as PAIR_SESSION_ID=testsid, so read_target must ignore it (a fresh session
# prompts; an Alt+n resume — same id — would keep its target). (#66 smoke #6.)
printf '{"file":"/stale/prev.md","status":"ready","session":"oldsid"}\n' > "$RT/review-target-test.json"
( cd "$RT" && PATH="$RT/bin:$PATH" \
    PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" PAIR_SESSION_ID=testsid \
    RESULT="$RESULT" ZLOG="$ZLOG" FLOATVIS="$FLOATVIS" \
    run_headless --timeout 30 -- nvim --headless -u "$ROOT/nvim/init.lua" "$RT/draft.md" \
      -c "luafile $RT/driver.lua" )

grep -q 'session-scope ok' "$RESULT" && pass "other-session target ignored (smoke #6)" || fail "stale (other-session) target not ignored"
grep -q 'propose-prepares-ready ok' "$RESULT" && pass ":PairReview prepares target locally" || fail ":PairReview local prepare"
for c in ts-same ts-diff ts-nocur ts-noid; do
  grep -q "$c ok" "$RESULT" && pass "pure target_stale: $c" || fail "target_stale $c"
done
grep -q 'old-unscoped-target-stale ok' "$RESULT" && pass "old unscoped target remains stale" || fail "old unscoped target accepted"
grep -q 'fresh-unscoped-target-read ok' "$RESULT" && pass "same-nvim unscoped target remains readable" || fail "same-nvim unscoped target ignored"
grep -q 'config-session-read ok' "$RESULT" && pass "read_target falls back to config session_id" || fail "read_target config fallback"
grep -q 'config-session-write ok' "$RESULT" && pass "write_target stamps config session_id" || fail "write_target config fallback"
grep -q 'no-live-codex-fallback ok' "$RESULT" && pass "current session does not guess from live Codex files" || fail "live Codex fallback remains"
grep -q 'unverified-live-target-stale ok' "$RESULT" && pass "unverified live target is stale" || fail "unverified live target accepted"
grep -q 'unverified-live-target-unstamped ok' "$RESULT" && pass "unverified live target remains unstamped" || fail "unverified live target stamped"
grep -q 'pure-prompt ok'  "$RESULT" && pass "pure: no target → prompt"        || fail "pure prompt"
grep -q 'pure-open ok'    "$RESULT" && pass "pure: target ready → open"       || fail "pure open"
grep -q 'pure-wait ok'    "$RESULT" && pass "pure: target proposed → wait"    || fail "pure wait"
grep -q 'pure-hide ok'    "$RESULT" && pass "pure: alive+visible → hide"      || fail "pure hide"
grep -q 'pure-show ok'    "$RESULT" && pass "pure: alive+hidden → show"       || fail "pure show"
grep -q '^hide ok$'       "$RESULT" && pass "live+visible → hide-floating-panes" || fail "hide branch"
grep -q '^show ok$'       "$RESULT" && pass "live+hidden → show-floating-panes" || fail "show branch"
grep -q '^prompt ok$'     "$RESULT" && pass "no target → :PairReview prompt (no open/show/hide)" || fail "prompt branch"
grep -q '^targetopen ok$' "$RESULT" && pass "target ready → opens the pane (pair review open)" || fail "open branch"
grep -q '^wait ok$'       "$RESULT" && pass "target proposed → wait (no open)" || fail "wait branch"
grep -q '^footgun ok$'    "$RESULT" && pass "never toggle-floating-panes" || fail "footgun (toggle-floating-panes used)"

# ── config lint ───────────────────────────────────────────────────────────────
grep -q 'bind "Alt c"' "$ROOT/zellij/config.kdl" && pass "Alt+c bound in config.kdl" || fail "no Alt+c bind"
grep -Fq 'bind "Alt c" { WriteChars "\u{1b}[99;3u"; }' "$ROOT/zellij/config.kdl" \
  && pass "Alt+c forwards to focused process for draft routing" \
  || fail "Alt+c target wrong"
grep -Fq 'bind "Alt r" { Write 27; Write 114; }' "$ROOT/zellij/config.kdl" \
  && pass "Alt+r forwards ESC+r for review-pane reject" \
  || fail "Alt+r does not forward to focused pane"
grep -Eq '^[[:space:]]*bind "Alt r"[^{]*\\{.*(RenameTab|TabNameInput|NewTab)' "$ROOT/zellij/config.kdl" \
  && fail "Alt+r still globally owns tab behavior" \
  || pass "Alt+r has no global tab action"
grep -q 'unbind "Alt o"' "$ROOT/zellij/config.kdl" && pass "Alt+o default zellij tab-move disabled" || fail "Alt+o still captured by zellij"
grep -q 'Run "pair-review-toggle"' "$ROOT/zellij/config.kdl" && fail "Alt+c still spawns the old toggle pane" || pass "old pair-review-toggle pane gone"
	}
	if mode, ok := f.liveLayouts[session]; ok {
		return mode, nil
	}
	return Layout2, nil
}

// ProcOps
func (f *fakeRuntime) SpawnSessionWatcher(agent, tag, cwd, repoRoot, repoName string, agentArgs []string) {
	f.watchers = append(f.watchers, agent+"|"+tag+"|"+cwd+"|"+repoRoot+"|"+repoName+"|"+strings.Join(agentArgs, " "))
}
func (f *fakeRuntime) SpawnTitlePoller(tag, agent, session string) {
	f.pollers = append(f.pollers, tag+"|"+agent)
}
func (f *fakeRuntime) DevRebuild(pairHome string) { f.devRebuilt = true }

// EnvOps
func (f *fakeRuntime) SetEnv(key, value string)       { f.env[key] = value }
func (f *fakeRuntime) InZellijPane() bool             { return f.inPane }
func (f *fakeRuntime) CommandExists(name string) bool { return !f.commandMissing[name] }
func (f *fakeRuntime) RecordOuterTTY(tag string)      { f.ttyRecorded = append(f.ttyRecorded, tag) }
func (f *fakeRuntime) CmuxRename(tag, title string)   { f.cmux = append(f.cmux, tag+"|"+title) }

// IDOps
func (f *fakeRuntime) MintUUID() string {
	if len(f.uuids) == 0 {
		return ""
	}
	u := f.uuids[0]
	f.uuids = f.uuids[1:]
	return u
}
func (f *fakeRuntime) AgentSessionExists(agent, sid, cwd string) bool {
	return f.agentSessions[agent+"|"+sid]
}
func (f *fakeRuntime) LiveAgentSessionID(agent, tag string) string {
	return f.liveAgentSessions[agent+"|"+tag]
}
func (f *fakeRuntime) InferAgent(tag string) string {
	if latest, ok := LatestLedgerEntry(f.ledger[tag]); ok && latest.Agent != "" {
		return latest.Agent
	}
	return f.inferAgent[tag]
}
func (f *fakeRuntime) ReadLedger(tag string) ([]LedgerEntry, error) {
	return append([]LedgerEntry(nil), f.ledger[tag]...), nil
}
func (f *fakeRuntime) AppendLedger(tag string, entry LedgerEntry) error {
	if f.appendLedgerErr != nil {
		return f.appendLedgerErr
	}
	f.ledger[tag] = append(f.ledger[tag], entry)
	return nil
}
func (f *fakeRuntime) ReadSessionNameIndex() (SessionNameIndex, error) {
	return f.sessionIndex, nil
	if _, ok := rt.files["/data/agent-default-codex.json"]; ok {
		t.Fatalf("default persisted despite abort: %q", rt.files["/data/agent-default-codex.json"])
	}
}

// The tag-restart config picker: a saved config offers reuse; picking "saved
// params + session" composes the resume binding.
func TestRunLaunchTagRestartPickerResume(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":["--search"],"session_id":"CX-9"}`
	rt.agentSessions["codex|CX-9"] = true // native session artifact exists → resumable
	rt.pickFunc = func(header string, options []string) string {
		return options[0] // "use saved params + session"
	}
	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	// codex resume subcommand LEADS, --no-alt-screen appended idempotently.
	if rt.env["PAIR_AGENT_ARGS"] != "resume CX-9 --search --no-alt-screen" {
		t.Fatalf("PAIR_AGENT_ARGS = %q", rt.env["PAIR_AGENT_ARGS"])
	}
}

func TestRunLaunchTagRestartPickerResumeStripsCodexResumeAfterGlobals(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":["--sandbox","danger-full-access","resume","CX-9","--no-alt-screen"],"session_id":"CX-9"}`
	rt.agentSessions["codex|CX-9"] = true
	rt.pickFunc = func(header string, options []string) string {
		return options[0] // "use saved params + session"
	}

	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.env["PAIR_AGENT_ARGS"] != "resume CX-9 --sandbox danger-full-access --no-alt-screen" {
		t.Fatalf("PAIR_AGENT_ARGS = %q", rt.env["PAIR_AGENT_ARGS"])
	}
}

func TestRunLaunchTagRestartPickerWarnsWhenSavedSessionIsStale(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":["--search"],"session_id":"CX-9"}`
	rt.pickFunc = func(header string, options []string) string {
		for _, o := range options {
			if strings.Contains(o, "use saved params") {
				return o
			}
		}
		return ""
	}

	var stderr bytes.Buffer
	code, err := RunLaunch(baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"}), rt, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if rt.env["PAIR_AGENT_ARGS"] != "--search --no-alt-screen" {
		t.Fatalf("PAIR_AGENT_ARGS = %q", rt.env["PAIR_AGENT_ARGS"])
	}
	if !strings.Contains(stderr.String(), `saved session "CX-9" for codex is not available; starting fresh`) {
		t.Fatalf("stderr missing stale-session warning: %s", stderr.String())
	}
	if !slices.Contains(rt.removed, "/data/config-cx-codex.json") {
		t.Fatalf("removed = %v, want stale Codex config quarantined", rt.removed)
	}
}

// Picking "new" drops the stale config.
func TestRunLaunchTagRestartPickerNew(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-work-claude.json"] = `{"agent":"claude","args":["--old"],"session_id":"OLD"}`
	rt.uuids = []string{"NEW-SID"}
	rt.pickFunc = func(header string, options []string) string {
		for _, o := range options {
			if strings.Contains(o, "use new params passed in") {
				return o
			}
		}
		return ""
	}
	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work", AgentArgs: []string{"--fresh"}, AgentArgsExplicit: true})
	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !contains(rt.removed, "/data/config-work-claude.json") {
		t.Fatalf("new should remove stale config; removed=%v", rt.removed)
	}
	// The freshly-minted config replaces it (mint runs after the picker).
	if cfg := rt.files["/data/config-work-claude.json"]; !strings.Contains(cfg, "NEW-SID") {
		t.Fatalf("expected fresh minted config, got %q", cfg)
	}
}

// Aborting the config picker exits 1.
func TestRunLaunchTagRestartPickerAbort(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":[],"session_id":""}`
	rt.pickFunc = func(header string, options []string) string { return "" }
	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"}), rt)
	if err != nil || code != 1 {
		t.Fatalf("picker abort should exit 1: code=%d err=%v", code, err)
	}
	if rt.launched != "" {
		t.Fatalf("must not launch on picker abort")
	}
}

// An explicit --resume on argv skips the picker and pre-writes the config.
func TestRunLaunchExplicitResumeSkipsPicker(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-work-claude.json"] = `{"agent":"claude","args":["--saved"],"session_id":"SAVED"}`
	pickerCalled := false
	rt.pickFunc = func(header string, options []string) string { pickerCalled = true; return options[0] }
	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work", AgentArgs: []string{"--resume", "EXPLICIT"}, AgentArgsExplicit: true})
	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if pickerCalled {
		t.Fatalf("explicit resume must skip the picker")
	}
	// Config pre-written with the explicit id, args stripped of the resume token.
	cfg := rt.files["/data/config-work-claude.json"]
	if !strings.Contains(cfg, `"session_id":"EXPLICIT"`) || strings.Contains(cfg, "--resume") {
		t.Fatalf("config = %q", cfg)
	}
	if rt.env["PAIR_SESSION_ID"] != "EXPLICIT" {
		t.Fatalf("PAIR_SESSION_ID = %q", rt.env["PAIR_SESSION_ID"])
	}
}

// A Runtime query failure (Sessions) exits 1 with a message — no shell to fall
// back to as of M5c (the path is unreachable via OSRuntime, which swallows zellij
// errors, but this pins the defensive branch).
func TestRunLaunchSessionsErrorExits(t *testing.T) {
	rt := newFakeRuntime()
	rt.sessionsErr = errors.New("zellij unreachable")
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "x"}), rt)
		Agent:      "codex",
		Args:       []string{"--search"},
		SessionID:  "CX-9",
		LastActive: time.Unix(1_700_000_010, 0),
	}}
	rt.agentSessions["codex|CX-9"] = true
	rt.pickFunc = func(header string, options []string) string {
		for _, o := range options {
			if strings.Contains(o, "use saved params + session") {
				return o
			}
		}
		return ""
	}

	code, err := run(t, baseOpts(LaunchArgs{ForcedTag: "work"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.env["PAIR_AGENT"] != "codex" {
		t.Fatalf("PAIR_AGENT = %q, want codex", rt.env["PAIR_AGENT"])
	}
	if rt.env["PAIR_AGENT_ARGS"] != "resume CX-9 --search --no-alt-screen" {
		t.Fatalf("PAIR_AGENT_ARGS = %q", rt.env["PAIR_AGENT_ARGS"])
	}
}

func TestRunLaunchRejectsInvalidLedgerCodexSession(t *testing.T) {
	rt := newFakeRuntime()
	rt.ledger["work"] = []LedgerEntry{{
		Agent:      "codex",
		Args:       []string{"--search"},
		SessionID:  "SUBAGENT",
		LastActive: time.Unix(1_700_000_010, 0),
	}}
	rt.pickFunc = func(header string, options []string) string {
		for _, o := range options {
			if strings.Contains(o, "use saved params") {
				return o
			}
		}
		return ""
	}

	var stderr bytes.Buffer
	code, err := RunLaunch(baseOpts(LaunchArgs{ForcedTag: "work"}), rt, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if strings.Contains(rt.env["PAIR_AGENT_ARGS"], "SUBAGENT") {
		t.Fatalf("PAIR_AGENT_ARGS = %q, must not resume rejected session", rt.env["PAIR_AGENT_ARGS"])
	}
	if rt.env["PAIR_AGENT_ARGS"] != "--search --no-alt-screen" {
		t.Fatalf("PAIR_AGENT_ARGS = %q", rt.env["PAIR_AGENT_ARGS"])
	}
	if !slices.Contains(rt.removed, "/data/config-work-codex.json") {
		t.Fatalf("removed = %v, want canonical config quarantine", rt.removed)
	}
	if !strings.Contains(stderr.String(), `saved session "SUBAGENT" for codex is not available; starting fresh`) {
		t.Fatalf("stderr missing stale-session warning: %s", stderr.String())
	}
}

func TestRunLaunchAltNRestartRejectsInvalidSavedCodexSession(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":["--search"],"session_id":"SUBAGENT"}`
	rt.restartMarkers["📁work-cx"] = RestartMarker{Tag: "cx", Agent: "codex"}

	opts := baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"})
	opts.SkipConfigPicker = true
	var stderr bytes.Buffer
	code, err := RunLaunch(opts, rt, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if rt.launchCount != 2 {
		t.Fatalf("launchCount = %d, want initial launch plus Alt+n relaunch", rt.launchCount)
	}
	if strings.Contains(rt.env["PAIR_AGENT_ARGS"], "SUBAGENT") {
		t.Fatalf("PAIR_AGENT_ARGS = %q, must not resume rejected session", rt.env["PAIR_AGENT_ARGS"])
	}
	if !slices.Contains(rt.removed, "/data/config-cx-codex.json") {
		t.Fatalf("removed = %v, want stale Codex config quarantined", rt.removed)
	}
	if !strings.Contains(stderr.String(), `saved session "SUBAGENT" for codex is not available; starting fresh`) {
		t.Fatalf("stderr missing stale-session warning: %s", stderr.String())
	}
}

// With nothing on disk to infer from, the agent defaults to claude.
func TestRunLaunchResumeDefaultsClaude(t *testing.T) {
    12	
    13	## Core concepts
    14	
    15	### Pure entities
    16	
    17	| Name | Lives in | Status |
    18	|------|----------|--------|
    19	| `CodexRootSessionID` | `cmd/internal/transcript/transcript.go` | new |
    20	| `DecideAutomaticResumeConfig` | `cmd/internal/launcher/createflow.go` | new |
    21	
    22	- **`CodexRootSessionID`** — authorizes a session UUID from a rollout path plus its first JSONL event.
    23	  - **Relationships:** N:1 with a Codex process tree: many rollout candidates may be visible, exactly one accepted root ID is selected by each consumer.
    24	  - **DRY rationale:** replaces filename-only authorization duplicated by sessionwatch, launcher, codexsid, slug, and Neovim (ARCH-DRY).
    25	  - **Future extensions:** widen the accepted root source enum only when a captured upstream `session_meta` fixture proves a new root shape.
    26	- **`DecideAutomaticResumeConfig`** — projects a saved config plus already-observed validation facts into safe automatic-resume intent: preserve args, retain a validated ID, or clear an invalid ID and request quarantine/warning.
    27	  - **Relationships:** 1:1 with a loaded saved config; consumed by both config-picker and restart-marker flows.
    28	  - **DRY rationale:** prevents two automatic-resume boundaries from independently deciding whether persisted Codex identity is trustworthy.
    29	  - **Future extensions:** agent-specific persisted-identity validators can join without weakening explicit user-supplied resume authority.
    30	
    31	### Integration points
    32	
    33	| Name | Lives in | Status | Wraps |
    34	|------|----------|--------|-------|
    35	| `ReadCodexRootSessionID` | `cmd/internal/transcript/transcript.go` | new | bounded first-line filesystem read |
    36	| `sessionwatch.Runtime.ReadFirstLine` | `cmd/internal/sessionwatch/run.go`, `runtime.go` | new | watcher filesystem seam |
    37	| Shared process candidate seam | `cmd/internal/procutil/procutil.go` | modified | `ps`/`lsof` parsing and traversal |
    38	| Validated Pair config | `nvim/init.lua` | modified | asynchronous Go-authored session identity |
    39	
    40	- **`ReadCodexRootSessionID`** — reads only the first JSONL event, then calls the pure classifier; unreadable, oversized, or unterminated/incomplete metadata fails closed.
    41	  - **Injected into:** process candidate scans call this adapter after their existing `ps`/`lsof` seam returns paths; sessionwatch uses its own injected first-line reader and calls the pure classifier directly.
    42	  - **Future extensions:** expose classification diagnostics if adaptation telemetry needs to distinguish malformed metadata from explicit subagents.
    43	- **`sessionwatch.Runtime.ReadFirstLine`** — keeps watcher scheduling and candidate ordering deterministic under its stateful fake without reading whole, potentially large transcripts.
    44	  - **Injected into:** `discover` and `discoverByBirth` candidate authorization.
    45	  - **Future extensions:** none planned; deliberately narrower than general transcript reads.
    46	- **Shared process candidate seam** — one implementation owns external `ps`/`lsof` parsing and traversal; tests provide temp rollout trees plus the existing fake command/state model (ARCH-MOCK).
    47	  - **Injected into:** launcher `LiveAgentSessionID`, `codexsid.ResolveSessionID`, slug transcript resolution, and sessionwatch's OS runtime while sessionwatch retains its injected runtime interface.
    48	  - **Future extensions:** command execution injection can widen here without duplicating parsers in consumers.
    49	- **Validated Pair config** — Neovim reads `PAIR_SESSION_ID` or `config-<tag>-<agent>.json`; it no longer shells out or parses rollout filenames.
    50	  - **Injected into:** review-target scoping.
    51	  - **Future extensions:** a typed Pair identity sidecar could replace config reads if more UI consumers emerge.
    52	
    53	## Chunk 1: Root classifier and consumers
    54	
    55	### Task 1: Add the canonical Codex root-session classifier
    56	
    57	**Files:**
    58	- Modify: `cmd/internal/transcript/transcript.go`
    55		DropConfig   bool   // Shift+Alt+N / compaction: drop the saved config first
    56		ContinueSlug string // #55 compaction re-entry: re-seed the draft from this slug
    57	}
    58	
    59	// decideAutomaticResumeConfig rejects only persisted Codex bindings that no
    60	// longer identify a verified root rollout. Keep the saved launch parameters so
    61	// callers can still offer a fresh launch with the user's prior flags.
    62	func decideAutomaticResumeConfig(agent string, saved savedConfig, sessionValid bool) (savedConfig, bool) {
    63		if agent != "codex" || saved.SessionID == "" || sessionValid {
    64			return saved, false
    65		}
    66		saved.SessionID = ""
    67		return saved, true
    68	}
    69	
    70	// planRestart maps a restart marker + the RESOLVED (tag, agent) + saved config
    71	// into the next launch (#99 M5b makes rename/continue native). The caller has
    72	// already applied the marker's tag/agent preference AND any rename_to move before
    95				s = 1 // dump starts before .ansi line 1 → user is at the very top
    96			}
    97			return s, true
    98		}
    99		return 0, false
   100	}
   101	
   102	// resolveSessionID implements the #63 change-log keying: an explicit
   103	// PAIR_SESSION_ID wins; else the per-tag config's session_id; else "" (the
   104	// legacy unsuffixed base). configJSON is the raw config-<tag>-<agent>.json bytes
   105	// (nil/empty when absent).
   106	func resolveSessionID(envSID string, configJSON []byte) string {
   107		if envSID != "" {
   108			return envSID
   109		}
   110		if len(configJSON) == 0 {
   111			return ""
   112		}
   113		var c struct {
   114			SessionID string `json:"session_id"`
   115		}
   116		if json.Unmarshal(configJSON, &c) != nil {
   117			return ""
   118		}
   119		return c.SessionID
   120	}
   121	
   122	// changelogBase is the per-session change-log path stem: the sid suffix is
   123	// appended only when resolved (fresh sessions branch; a resume reuses it).
   124	func changelogBase(dataDir, tag, agent, sid string) string {
   125		base := dataDir + "/changelog-" + tag + "-" + agent
   128			fmt.Fprintf(stderr, "pair-changelog-open: missing PAIR_DATA_DIR / PAIR_TAG / PAIR_AGENT\n")
   129			fmt.Fprintf(stderr, "  This is meant to run inside a pair session.\n")
   130			rt.Sleep(3 * time.Second)
   131			return 1
   132		}
   133	
   134		sid := opts.SessionID
   135		if sid == "" {
   136			if cfg, err := rt.ReadFile(opts.DataDir + "/config-" + opts.Tag + "-" + opts.Agent + ".json"); err == nil {
   137				sid = resolveSessionID("", []byte(cfg))
   138			}
   139		}
   140		base := changelogBase(opts.DataDir, opts.Tag, opts.Agent, sid)
   141		sb := opts.DataDir + "/scrollback-" + opts.Tag + "-" + opts.Agent
   142		raw, events := sb+".raw", sb+".events.jsonl"
   143		log, anchor, cleaned := base+".md", base+".anchor", base+".cleaned"
   144		openlock, dlock, status := base+".openlock", base+".distill.lock", base+".status"
   145	

exec
/bin/zsh -lc 'bash tests/review-toggle-test.sh' in /private/tmp/pair-context-usage
exec
/bin/zsh -lc 'env -u PAIR_SESSION_ID -u PAIR_AGENT -u PAIR_TAG -u PAIR_DATA_DIR GOCACHE=/tmp/pair-review-gocache go test ./cmd/internal/transcript ./cmd/internal/sessionwatch ./cmd/internal/launcher ./cmd/internal/procutil ./cmd/internal/codexsid ./cmd/internal/slugcmd ./cmd/internal/contextcmd ./cmd/internal/reviewcmd -count=1' in /private/tmp/pair-context-usage
 succeeded in 2023ms:
  ok   other-session target ignored (smoke #6)
  ok   :PairReview prepares target locally
  ok   pure target_stale: ts-same
  ok   pure target_stale: ts-diff
  ok   pure target_stale: ts-nocur
  ok   pure target_stale: ts-noid
  ok   old unscoped target remains stale
  ok   same-nvim unscoped target remains readable
  ok   read_target falls back to config session_id
  ok   write_target stamps config session_id
  ok   current session does not guess from live Codex files
  ok   unverified live target is stale
  ok   unverified live target remains unstamped
  ok   pure: no target → prompt
  ok   pure: target ready → open
  ok   pure: target proposed → wait
  ok   pure: alive+visible → hide
  ok   pure: alive+hidden → show
  ok   live+visible → hide-floating-panes
  ok   live+hidden → show-floating-panes
  ok   no target → :PairReview prompt (no open/show/hide)
  ok   target ready → opens the pane (pair review open)
  ok   target proposed → wait (no open)
  ok   never toggle-floating-panes
  ok   Alt+c bound in config.kdl
  ok   Alt+c forwards to focused process for draft routing
  ok   Alt+r forwards ESC+r for review-pane reject
  ok   Alt+r has no global tab action
  ok   Alt+o default zellij tab-move disabled
  ok   old pair-review-toggle pane gone
  ok   Alt+x forwards to focused process for local routing
  ok   Alt+x no longer injects quit command from zellij
review-toggle-test ok

 succeeded in 3800ms:
ok  	github.com/xianxu/pair/cmd/internal/transcript	0.456s
ok  	github.com/xianxu/pair/cmd/internal/sessionwatch	0.301s
ok  	github.com/xianxu/pair/cmd/internal/launcher	3.638s
ok  	github.com/xianxu/pair/cmd/internal/procutil	0.565s
ok  	github.com/xianxu/pair/cmd/internal/codexsid	2.278s
ok  	github.com/xianxu/pair/cmd/internal/slugcmd	2.695s
ok  	github.com/xianxu/pair/cmd/internal/contextcmd	3.185s
ok  	github.com/xianxu/pair/cmd/internal/reviewcmd	1.279s

exec
/bin/zsh -lc "sed -n '1,90p' cmd/internal/wrapcmd/readiness_test.go; sed -n '1,120p' cmd/internal/wrapcmd/readiness.go; env | rg '"'^PAIR_|''^XDG_'"' | sort; nl -ba atlas/review-workbench.md | sed -n '158,171p'; nl -ba atlas/architecture.md | sed -n '895,920p'; nl -ba atlas/architecture.md | sed -n '970,992p'; nl -ba cmd/internal/launcher/osruntime_test.go | sed -n '320,355p'; nl -ba cmd/internal/launcher/createflow_test.go | sed -n '1214,1276p'; nl -ba tests/review-toggle-test.sh | sed -n '32,62p;145,165p'" in /private/tmp/pair-context-usage
 succeeded in 0ms:
package wrapcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/readiness"
)

func TestPublishAgentReadyWritesRecordFromPairEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PAIR_DATA_DIR", dir)
	t.Setenv("PAIR_TAG", "work")
	t.Setenv("PAIR_SESSION_NAME", "pair-work")
	t.Setenv("PAIR_LAUNCH_NONCE", "nonce-1")

	p := &proxy{agentBasename: "codex"}
	p.resolvePaths()
	if p.agentReadyPath == "" {
		t.Fatal("agentReadyPath is empty")
	}
	if err := p.publishAgentReady(321); err != nil {
		t.Fatalf("publishAgentReady returned error: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "agent-ready-work-codex.json"))
	if err != nil {
		t.Fatalf("read ready record: %v", err)
	}
	got, err := readiness.Decode(string(raw))
	if err != nil {
		t.Fatalf("Decode ready record: %v", err)
	}
	if got.Tag != "work" || got.Agent != "codex" || got.Session != "pair-work" || got.Nonce != "nonce-1" || got.PID != 321 {
		t.Fatalf("ready record = %+v", got)
	}
}

func TestPublishAgentReadySkipsWhenPairEnvIncomplete(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PAIR_DATA_DIR", dir)
	t.Setenv("PAIR_TAG", "work")

	p := &proxy{agentBasename: "codex"}
	p.resolvePaths()
	if err := p.publishAgentReady(321); err != nil {
		t.Fatalf("publishAgentReady returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "agent-ready-work-codex.json")); !os.IsNotExist(err) {
		t.Fatalf("ready file stat err = %v, want not exist", err)
	}
}
sed: cmd/internal/wrapcmd/readiness.go: No such file or directory
PAIR_AGENT=codex
PAIR_AGENT_ARGS=--sandbox danger-full-access --no-alt-screen
PAIR_DATA_DIR=/Users/xianxu/.local/share/pair/repos/53998a195009a6ec
PAIR_DEV=1
PAIR_HOME=/Users/xianxu/workspace/pair
PAIR_LAUNCH_NONCE=ea16d0bad7b5344a10410745290a4459
PAIR_PANE_TITLE=codex
PAIR_SESSION_ID=
PAIR_SESSION_NAME=📁parley-nvim
PAIR_TAG=parley_nvim
PAIR_WORKBENCH_LAYOUT=layout2
XDG_DATA_DIRS=/usr/local/share:/usr/share:/Applications/Ghostty.app/Contents/Resources/ghostty/..
   158	  state-file liveness and review-target status: live review → flip visibility from
   159	  this *tiled* draft (`are-floating-panes-visible` → `show`/`hide-floating-panes`;
   160	  **never** `toggle-floating-panes`); no live review + ready target → open;
   161	  proposed target → "prep in progress"; no target → drop into `:PairReview `
   162	  (file-select). Pure decision `_pair_review_toggle_action(alive, visible, status)`.
   163	  Review-targets are scoped to the current conversation id so fresh sessions ignore
   164	  stale targets while resumed sessions keep their in-progress target. Resolution is
   165	  `PAIR_SESSION_ID` → `config-<tag>-<agent>.json` → live Codex rollout via
   166	  `agent-pid-<tag>`; Codex/agy learn ids asynchronously, so review target handling must
   167	  not rely on the launch-time env alone. `Alt+r` is deliberately free inside the review
   168	  pane for reject.
   169	- `nvim/pair_poke.lua` — id-based agent poke: relative `move-focus` does NOT escape a
   170	  floating pane, so it resolves the agent pane from `list-panes --json` and writes
   171	  directly with `write-chars --pane-id <agent>` + `send-keys --pane-id <agent> "Alt Enter"`.
   895	Known gap: `/clear` rotates claude's session id mid-session, allocating a new jsonl that neither layer above sees. The launch-time `--session-id` is captured at create time, the watcher's 60s window is long gone by then, and there is no Alt+x trigger anymore. After a `/clear` + Alt+x, `pair resume <tag>` will replay the pre-clear conversation. (Pair previously sent a `bye\n` to the agent on Alt+x specifically to refresh the saved id past a `/clear`; that layer was retired because it polluted the conversation log and the rotation case is rare in practice. `/compact` doesn't rotate.)
   896	
   897	Per-agent surface:
   898	
   899	| Agent | Path | Id source | Capture mechanism |
   900	|---|---|---|---|
   901	| claude | `~/.claude/projects/<encoded-cwd>/<id>.jsonl` | filename | `--session-id` pre-injected by the launcher (deterministic) |
   902	| codex | `~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<id>.jsonl` | trailing UUID in filename (regex) | `lsof -p <pid>` against agent PID + `ps`-discovered descendants, birth-time fallback |
   903	| agy | `~/.gemini/antigravity-cli/conversations/<id>.db` | UUID database filename | `lsof -p <pid>` against agent PID + `ps`-discovered descendants, birth-time fallback |
   904	
   905	**Stored shape.** `$PAIR_DATA_DIR/config-<tag>-<agent>.json`:
   906	
   907	```json
   908	{ "agent": "claude", "args": ["--dangerously-skip-permissions"], "session_id": "8d745d08-..." }
   909	```
   910	
   911	Single write posture: structured JSON plus temp-file rename, only after the id is in hand. The launcher writes the synchronous claude/explicit-resume prewrites, and the Go watcher writes the codex/agy config once the id is discovered — both via `encoding/json` plus `os.CreateTemp`/rename. So a concurrent reader either sees a complete prior config or a complete new one — never a partial. Keyed by `(tag, agent)` because the same tag can hold separate configs for different agents.
   912	
   913	**Create-flow prompt (the launcher).** When the create path commits a tag, the launcher reads `config-<tag>-<agent>.json`. If present, it runs the per-agent stale-id check (claude: `[ -f .../<id>.jsonl ]`; codex: `find ~/.codex/sessions -name "*<id>*"`; agy: check conversation DB) and fzf-prompts the user with up to three options:
   914	
   915	```
   916	1) use params + session   args=[...]   resume=<id>
   917	2) use params             args=[...]   fresh session
   918	3) use none               args=[<current>]   fresh session
   919	```
   920	
   970	
   971	Internal: `~/.cache/pair/quit-<session>` — marker file used to communicate "user asked for full quit" between `pair quit` (or `pair restart`) and the launcher. Touched on Alt+x and Alt+n; removed by the launcher after delete-session.
   972	
   973	Internal: `~/.cache/pair/restart-<session>` — marker written alongside `quit-` by `pair restart` (Alt+n, plus the independent compaction flow). Holds `tag`, `agent`, optional `session_id`, and restart metadata as `key=value` lines so the launcher can reconstruct the relaunch params after cleanup has wiped `agent-<tag>`. Plain Codex restarts can fill `session_id` from the live rollout transcript before the pane is killed; the restart planner prefers that marker id over saved config because it is the freshest source. Removed when the in-process restart loop consumes it.
   974	
   975	Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/outer-tty-<tag>` — single-line file containing the path to pair's controlling TTY at attach time. Read by `pair-notify` to emit OSC escapes that reach the outer terminal/wrapper. Rewritten on every attach (create or reattach); removed on full quit.
   976	
   977	Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-<tag>` — single-line file recording which agent binary was launched in the session (`claude`, `codex`, ...). Written once at session create; read by `pair list` to display the agent column, and by the launcher's tag-restart agent-inference. Removed on full quit. The agent isn't otherwise recoverable post-create — env vars are frozen in pane shells, and custom session names (e.g. `pair-bugfix`) don't carry the agent in the name.
   978	
   979	Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/config-<tag>-<agent>.json` — saved restart configuration for `(tag, agent)` (issue #000016, #000020). `{ agent, args, session_id }`. For claude, written synchronously by the launcher before zellij launch (`--session-id` is deterministic). For codex/agy, written by the Go `pair-session-watch` command once the agent's session file is discovered via lsof. Read by the launcher's create-flow prompt and by the post-Alt+x hint. Survives Alt+x (unlike `agent-<tag>`, which is cleared) — that's the whole point: it's the bridge between two pair launches against the same tag.
   980	
   981	Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-pid-<tag>` — child agent PID written by `cmd/pair-wrap` immediately after `pty.Start`, removed on shutdown. Consumed by `cmd/pair-session-watch` to scope `lsof` discovery to a specific process tree (issue #000020). Mtime is also used as the agent-start epoch in the watcher's birth-time fallback.
   982	
   983	Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/nvim-pid-<tag>-{draft,scrollback}` — single-line file containing the pid of an `nvim --embed` server child. Written at VimEnter by `nvim/init.lua` (for the draft pane) and `nvim/scrollback.lua` (for the Alt+/ floating viewer) when `$PAIR_NVIM_PID_FILE` is set; the launch sites (`zellij/layouts/main-{2,3}.kdl` for draft, `bin/pair-scrollback-open` for scrollback) export the env var pointing at a tag-scoped path. Read and removed by `cleanup_quit_marker` on Alt+x to SIGKILL the embed deterministically — without this, the embed sometimes survives zellij's pane teardown and accumulates as a PPID=1 orphan, dragging the host into memory pressure across many quits.
   984	
   985	Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/pair-wrap-pid-<tag>` — single-line file containing pair-wrap's pid, written at startup by `bin/pair-wrap` if `PAIR_TAG` is set. Read by nvim's Alt+i (`attach_image`) so it can `kill -USR1 <pid>` to arm an image-capture window. Removed by pair-wrap on exit (the `finally` block in `main()`) and by `cleanup_quit_marker` as belt-and-suspenders on Alt+x.
   986	
   987	Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/image-capture-<tag>` + `image-capture-<tag>.done` — paired files driving the Alt+i image-marker pickup. On SIGUSR1, pair-wrap buffers bytes from the agent's PTY for `PAIR_WRAP_CAPTURE_S` seconds (default 0.2), then writes the buffer to the first file and touches the `.done` sentinel. nvim polls the sentinel (20ms cadence, 600ms cap), reads the buffer, strips ANSI, regex-matches the agent's image marker (claude `[Image #N]`, agy `[Image N-M]`), and inserts it at cursor. Both files are removed by nvim after the pickup and by `cleanup_quit_marker` on Alt+x.
   988	
   989	Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/slug-proposed-<tag>` and `slug-<tag>` — the orientation-slug channel (issue #000027). `pair-slug` (spawned by pair-wrap at turn-end) writes the proposed `=== <branch> | <focus> ===` to `slug-proposed-<tag>` (atomic temp+rename); nvim applies it to draft line 1 and writes the effective line back to `slug-<tag>`, which is the `prev` the proposer reads next turn. For Codex, if `config-<tag>-codex.json` is missing, `pair-slug` can recover the live rollout by reading `agent-pid-<tag>`, walking descendants via `ps`, and checking their `lsof` paths for `~/.codex/sessions/.../rollout-*.jsonl`. Agy has two artifacts: restart/session discovery uses `~/.gemini/antigravity-cli/conversations/<session_id>.db`, while transcript summarization still reads `~/.gemini/antigravity-cli/brain/<session_id>/.system_generated/logs/transcript.jsonl`. Codex model auth is API-key first, then Codex CLI subscription auth via `codex exec`. Single writer each, so the channel is race-free.
   990	
   991	Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/adapt-<tag>.jsonl` — the adaptation flight recorder (issue #000045). One JSON line per harness-adaptation trigger (`{ts, comp, agent, aspect, signal, outcome, detail}`), appended concurrently by `cmd/pair-wrap`, `cmd/pair-slug`, `cmd/pair-session-watch`, and `nvim/adapt.lua` — all writing one shared schema. Truncated once at session launch by the launcher (so multi-process `O_APPEND` never races) and removed on full quit. Read by `doctor/doctor.sh` to surface integration drift (near-miss/fail signals). See `atlas/how-to-bring-up-a-new-harness-cli.md` §3 for the signal registry.
   992	
   320		}
   321		if err := os.WriteFile(filepath.Join(dataDir, "ledger-work.jsonl"), []byte(line+"\n"), 0o644); err != nil {
   322			t.Fatal(err)
   323		}
   324		if err := os.WriteFile(filepath.Join(dataDir, "agent-work"), []byte("claude\n"), 0o644); err != nil {
   325			t.Fatal(err)
   326		}
   327		if got := rt.InferAgent("work"); got != "codex" {
   328			t.Fatalf("InferAgent = %q, want ledger codex", got)
   329		}
   330	}
   331	
   332	func TestOSRuntimeAgentSessionExistsFindsNestedCodexRollout(t *testing.T) {
   333		home := t.TempDir()
   334		t.Setenv("HOME", home)
   335		sid := "12345678-1234-1234-1234-123456789abc"
   336		path := filepath.Join(home, ".codex", "sessions", "2026", "07", "07", "rollout-2026-07-07T21-00-00-"+sid+".jsonl")
   337		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
   338			t.Fatal(err)
   339		}
   340		first := fmt.Sprintf(`{"type":"session_meta","payload":{"id":%q,"parent_thread_id":null,"source":"cli"}}`+"\n", sid)
   341		if err := os.WriteFile(path, []byte(first), 0o644); err != nil {
   342			t.Fatal(err)
   343		}
   344	
   345		if !(OSRuntime{}).AgentSessionExists("codex", sid, "/repo") {
   346			t.Fatal("AgentSessionExists(codex) did not find nested rollout file")
   347		}
   348	}
   349	
   350	func TestOSRuntimeSessionNameIndexStore(t *testing.T) {
   351		dataDir := t.TempDir()
   352		rt := NewOSRuntime(dataDir, "/pair")
   353		entry := SessionNameEntry{
   354			SessionName: "📁pair-work",
   355			ScopeKey:    "scope1",
  1214		}
  1215	}
  1216	
  1217	func TestRunLaunchRejectsInvalidLedgerCodexSession(t *testing.T) {
  1218		rt := newFakeRuntime()
  1219		rt.ledger["work"] = []LedgerEntry{{
  1220			Agent:      "codex",
  1221			Args:       []string{"--search"},
  1222			SessionID:  "SUBAGENT",
  1223			LastActive: time.Unix(1_700_000_010, 0),
  1224		}}
  1225		rt.pickFunc = func(header string, options []string) string {
  1226			for _, o := range options {
  1227				if strings.Contains(o, "use saved params") {
  1228					return o
  1229				}
  1230			}
  1231			return ""
  1232		}
  1233	
  1234		var stderr bytes.Buffer
  1235		code, err := RunLaunch(baseOpts(LaunchArgs{ForcedTag: "work"}), rt, &stderr)
  1236		if err != nil || code != 0 {
  1237			t.Fatalf("code=%d err=%v stderr=%s", code, err, stderr.String())
  1238		}
  1239		if strings.Contains(rt.env["PAIR_AGENT_ARGS"], "SUBAGENT") {
  1240			t.Fatalf("PAIR_AGENT_ARGS = %q, must not resume rejected session", rt.env["PAIR_AGENT_ARGS"])
  1241		}
  1242		if rt.env["PAIR_AGENT_ARGS"] != "--search --no-alt-screen" {
  1243			t.Fatalf("PAIR_AGENT_ARGS = %q", rt.env["PAIR_AGENT_ARGS"])
  1244		}
  1245		if !slices.Contains(rt.removed, "/data/config-work-codex.json") {
  1246			t.Fatalf("removed = %v, want canonical config quarantine", rt.removed)
  1247		}
  1248		if !strings.Contains(stderr.String(), `saved session "SUBAGENT" for codex is not available; starting fresh`) {
  1249			t.Fatalf("stderr missing stale-session warning: %s", stderr.String())
  1250		}
  1251	}
  1252	
  1253	func TestRunLaunchAltNRestartRejectsInvalidSavedCodexSession(t *testing.T) {
  1254		rt := newFakeRuntime()
  1255		rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":["--search"],"session_id":"SUBAGENT"}`
  1256		rt.restartMarkers["📁work-cx"] = RestartMarker{Tag: "cx", Agent: "codex"}
  1257	
  1258		opts := baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"})
  1259		opts.SkipConfigPicker = true
  1260		var stderr bytes.Buffer
  1261		code, err := RunLaunch(opts, rt, &stderr)
  1262		if err != nil || code != 0 {
  1263			t.Fatalf("code=%d err=%v stderr=%s", code, err, stderr.String())
  1264		}
  1265		if rt.launchCount != 2 {
  1266			t.Fatalf("launchCount = %d, want initial launch plus Alt+n relaunch", rt.launchCount)
  1267		}
  1268		if strings.Contains(rt.env["PAIR_AGENT_ARGS"], "SUBAGENT") {
  1269			t.Fatalf("PAIR_AGENT_ARGS = %q, must not resume rejected session", rt.env["PAIR_AGENT_ARGS"])
  1270		}
  1271		if !slices.Contains(rt.removed, "/data/config-cx-codex.json") {
  1272			t.Fatalf("removed = %v, want stale Codex config quarantined", rt.removed)
  1273		}
  1274		if !strings.Contains(stderr.String(), `saved session "SUBAGENT" for codex is not available; starting fresh`) {
  1275			t.Fatalf("stderr missing stale-session warning: %s", stderr.String())
  1276		}
    32	mkdir -p "$RT/bin"
    33	cat > "$RT/bin/zellij" <<EOF
    34	#!/usr/bin/env bash
    35	printf '%s\n' "\$*" >> "$ZLOG"
    36	if [ "\$1" = action ] && [ "\$2" = are-floating-panes-visible ]; then
    37	  cat "$FLOATVIS" 2>/dev/null || echo false
    38	fi
    39	exit 0
    40	EOF
    41	cat > "$RT/bin/ps" <<'EOF'
    42	#!/usr/bin/env bash
    43	if [ "$1" = "-axo" ]; then
    44	  printf '111 1\n222 111\n'
    45	  exit 0
    46	fi
    47	exec /bin/ps "$@"
    48	EOF
    49	cat > "$RT/bin/lsof" <<'EOF'
    50	#!/usr/bin/env bash
    51	if [ "$1" = "-p" ] && [ "$2" = "222" ]; then
    52	  printf 'p222\nn%s/.codex/sessions/2026/06/21/rollout-2026-06-21T00-00-00-12345678-1234-1234-1234-123456789abc.jsonl\n' "$HOME"
    53	  exit 0
    54	fi
    55	printf 'p%s\n' "${2:-}"
    56	EOF
    57	chmod +x "$RT/bin/zellij" "$RT/bin/ps" "$RT/bin/lsof"
    58	
    59	printf 'draft\n' > "$RT/draft.md"
    60	cat > "$RT/driver.lua" <<'LUA'
    61	local OUT = io.open(os.getenv('RESULT'), 'w')
    62	local ZLOG = os.getenv('ZLOG')
   145	vim.fn.writefile({ '{"file":"/stale/prev.md","status":"ready","session":"oldsid"}' }, target)
   146	
   147	vim.env.PAIR_SESSION_ID = ''
   148	vim.env.PAIR_AGENT = 'codex'
   149	os.remove(vim.env.PAIR_DATA_DIR .. '/config-' .. vim.env.PAIR_TAG .. '-codex.json')
   150	vim.fn.writefile({ '111' }, vim.env.PAIR_DATA_DIR .. '/agent-pid-' .. vim.env.PAIR_TAG)
   151	vim.fn.writefile({ '{"file":"' .. draft .. '","status":"ready","session":"12345678-1234-1234-1234-123456789abc"}' }, target)
   152	OUT:write((R.current_session_id() == nil) and 'no-live-codex-fallback ok\n' or 'no-live-codex-fallback FAIL\n')
   153	OUT:write((R.read_target() == nil) and 'unverified-live-target-stale ok\n' or 'unverified-live-target-stale FAIL\n')
   154	R.write_target(draft, 'ready')
   155	written = vim.json.decode(table.concat(vim.fn.readfile(target), '\n'))
   156	OUT:write((written.session == '') and 'unverified-live-target-unstamped ok\n' or 'unverified-live-target-unstamped FAIL\n')
   157	vim.env.PAIR_AGENT = 'claude'
   158	vim.env.PAIR_SESSION_ID = 'testsid'
   159	vim.fn.writefile({ '{"file":"/stale/prev.md","status":"ready","session":"oldsid"}' }, target)
   160	
   161	-- live + visible → hide  (state file holds OUR pid, so kill -0 says alive)
   162	vim.fn.writefile({ tostring(vim.fn.getpid()) }, sf); setfloat('true')
   163	local n = #read_zlog(); _G.PairReviewToggle()
   164	local d = new_since(n)
   165	OUT:write((has(d, 'action are-floating-panes-visible') and has(d, 'action hide-floating-panes'))

exec
/bin/zsh -lc 'env -u PAIR_SESSION_ID -u PAIR_AGENT -u PAIR_AGENT_ARGS -u PAIR_TAG -u PAIR_DATA_DIR -u PAIR_SESSION_NAME -u PAIR_LAUNCH_NONCE -u PAIR_HOME -u PAIR_DEV GOCACHE=/tmp/pair-review-gocache go test ./... -count=1' in /private/tmp/pair-context-usage
 succeeded in 12629ms:
ok  	github.com/xianxu/pair/cmd/internal/adapt	0.316s
ok  	github.com/xianxu/pair/cmd/internal/agentcmd	0.170s
ok  	github.com/xianxu/pair/cmd/internal/ansi	0.445s
ok  	github.com/xianxu/pair/cmd/internal/changelogcmd	11.515s
ok  	github.com/xianxu/pair/cmd/internal/clipcmd	0.658s
ok  	github.com/xianxu/pair/cmd/internal/codexsid	4.550s
ok  	github.com/xianxu/pair/cmd/internal/contextcmd	8.161s
ok  	github.com/xianxu/pair/cmd/internal/continuationcmd	1.583s
ok  	github.com/xianxu/pair/cmd/internal/ctxmeter	1.195s
ok  	github.com/xianxu/pair/cmd/internal/dispatcher	1.384s
ok  	github.com/xianxu/pair/cmd/internal/draftroute	1.509s
ok  	github.com/xianxu/pair/cmd/internal/entrypoint	1.716s
ok  	github.com/xianxu/pair/cmd/internal/keyhelp	1.828s
ok  	github.com/xianxu/pair/cmd/internal/keyscmd	5.745s
ok  	github.com/xianxu/pair/cmd/internal/launcher	9.012s
ok  	github.com/xianxu/pair/cmd/internal/layoutcmd	1.887s
ok  	github.com/xianxu/pair/cmd/internal/model	5.617s
ok  	github.com/xianxu/pair/cmd/internal/opener	1.744s
ok  	github.com/xianxu/pair/cmd/internal/osfs	1.741s
ok  	github.com/xianxu/pair/cmd/internal/procutil	1.833s
ok  	github.com/xianxu/pair/cmd/internal/readiness	1.732s
ok  	github.com/xianxu/pair/cmd/internal/reviewcmd	1.736s
ok  	github.com/xianxu/pair/cmd/internal/runtimebundle	1.772s
?   	github.com/xianxu/pair/cmd/internal/runtimebundle/generatecmd	[no test files]
?   	github.com/xianxu/pair/cmd/internal/runtimebundle/manifestmodel	[no test files]
ok  	github.com/xianxu/pair/cmd/internal/runtimebundlegen	5.773s
ok  	github.com/xianxu/pair/cmd/internal/scribecmd	1.829s
ok  	github.com/xianxu/pair/cmd/internal/scrollbackcmd	1.853s
ok  	github.com/xianxu/pair/cmd/internal/sessionwatch	1.794s
ok  	github.com/xianxu/pair/cmd/internal/slugcmd	4.727s
ok  	github.com/xianxu/pair/cmd/internal/termcmd	1.914s
ok  	github.com/xianxu/pair/cmd/internal/textwidth	1.886s
ok  	github.com/xianxu/pair/cmd/internal/titlepoller	1.696s
ok  	github.com/xianxu/pair/cmd/internal/transcript	1.671s
ok  	github.com/xianxu/pair/cmd/internal/workbenchshortcut	1.754s
?   	github.com/xianxu/pair/cmd/internal/workbenchshortcut/generatecmd	[no test files]
ok  	github.com/xianxu/pair/cmd/internal/wrapcmd	2.941s
ok  	github.com/xianxu/pair/cmd/internal/zellijpane	1.598s
ok  	github.com/xianxu/pair/cmd/pair-go	5.540s

codex
```verdict
verdict: REWORK
confidence: high
```

The root-session classifier and consumer routing appear functionally sound, and all focused tests plus `go test ./...` pass with Pair environment variables cleared. The boundary is blocked by the required Core concepts consistency check: the durable plan claims entities and locations the diff does not deliver. Atlas documentation and several explicitly planned integration regressions also remain incomplete.

### 1. Strengths

- `CodexRootSessionID` cleanly separates pure metadata authorization from bounded filesystem reading and fails closed on malformed, mismatched, nested, unknown-source, oversized, and incomplete events (`cmd/internal/transcript/transcript.go:37`).
- Process-based consumers continue past rejected rollouts and reuse the shared classifier in launcher, codexsid, slug, and sessionwatch.
- Persisted Codex IDs are sanitized through one pure decision while preserving non-resume arguments (`cmd/internal/launcher/markers.go:59`).
- Neovim’s duplicate `ps`/`lsof` identity implementation was removed.
- The plan-gate ledger has no open carry-forward findings.

### 2. Critical findings

- `workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan.md:17-38` — The Core concepts table contradicts the implementation:
  - It claims `DecideAutomaticResumeConfig` at `createflow.go`; the implementation is unexported `decideAutomaticResumeConfig` in `markers.go:62`.
  - It marks `cmd/internal/procutil/procutil.go` as modified, but that file is unchanged in the reviewed range; existing helpers were reused.

  The review contract explicitly makes any Core concepts contradiction Critical. Fix by either aligning code with the table or adding a `## Revisions` entry and correcting the entity name, location, and process-seam status to “existing/reused.”

### 3. Important findings

- Atlas updates are incomplete. Existing authoritative sections still describe the removed filename-only behavior:
  - `atlas/architecture.md:902` says Codex identity comes from a trailing filename UUID.
  - `atlas/architecture.md:913` describes the stale-ID check as a filename `find`.
  - `atlas/review-workbench.md:165` still describes Neovim review-target resolution falling through to live `agent-pid` rollout discovery.
  - `atlas/architecture.md:989` describes slug discovery without the metadata authorization requirement.

  Update these sections to match the shared first-event classifier and thin Neovim behavior.

- The planned launcher integration coverage was not delivered. `createflow_test.go:1217` and `:1253` use a fake boolean lookup with sentinel `"SUBAGENT"` values, while the plan required real on-disk subagent rollouts. `osruntime_test.go:332` only covers acceptance of a valid root. Add an OS-runtime rejection test using actual subagent metadata so classifier-to-quarantine wiring is pinned.

- Several promised regressions are absent: subagent-only cases for launcher/codexsid/slug, malformed-metadata-before-root watcher continuation, and an assertion that the headless Neovim test never invokes its `ps`/`lsof` stubs. The current headless stubs do not record or fail on invocation (`tests/review-toggle-test.sh:41-57`).

### 4. Minor findings

None.

### 5. Test coverage notes

Passed:

- Focused eight-package identity suite.
- `bash tests/review-toggle-test.sh`.
- `go test ./... -count=1` with inherited Pair variables cleared.
- `git diff --check`.

The classifier’s pure boundary and candidate-order behavior are well covered. The main gap is real filesystem integration at the launcher quarantine boundary.

### 6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass. Semantic authorization is centralized in `transcript`; process parsing is reused from `procutil`.
- `ARCH-PURE`: Pass. Classification and resume-state projection are pure; IO remains in thin adapters/runtimes.
- `ARCH-PURPOSE`: Pass for the implemented identity consumers. Root/subagent ordering, persisted-state quarantine, and removal of Neovim live discovery are delivered.
- `ARCH-MOCK`: Pass. Production and tests share the existing runtime/process seams; no new direct external call escaped those seams. Strengthening the launcher’s real-rollout test remains advisable.
- The shadow sweep also finds changelog config consumers in `cmd/internal/opener` and `nvim/init.lua`. They key changelog files rather than transcript/review identity, so they do not clearly violate this issue’s stated scope, but a future identity API would avoid further hand-parsed config consumers.

### 7. Plan revision recommendations

Add a `## Revisions` entry recording:

- `DecideAutomaticResumeConfig` became unexported `decideAutomaticResumeConfig` and lives in `cmd/internal/launcher/markers.go`.
- The shared `procutil` process seam was pre-existing and reused, not modified.
- Correct the verification record after adding the missing integration cases, or explicitly document any intentionally dropped cases.

tokens used
114,644
```verdict
verdict: REWORK
confidence: high
```

The root-session classifier and consumer routing appear functionally sound, and all focused tests plus `go test ./...` pass with Pair environment variables cleared. The boundary is blocked by the required Core concepts consistency check: the durable plan claims entities and locations the diff does not deliver. Atlas documentation and several explicitly planned integration regressions also remain incomplete.

### 1. Strengths

- `CodexRootSessionID` cleanly separates pure metadata authorization from bounded filesystem reading and fails closed on malformed, mismatched, nested, unknown-source, oversized, and incomplete events (`cmd/internal/transcript/transcript.go:37`).
- Process-based consumers continue past rejected rollouts and reuse the shared classifier in launcher, codexsid, slug, and sessionwatch.
- Persisted Codex IDs are sanitized through one pure decision while preserving non-resume arguments (`cmd/internal/launcher/markers.go:59`).
- Neovim’s duplicate `ps`/`lsof` identity implementation was removed.
- The plan-gate ledger has no open carry-forward findings.

### 2. Critical findings

- `workshop/plans/000144-reject-codex-subagent-sessions-during-pair-identity-discovery-plan.md:17-38` — The Core concepts table contradicts the implementation:
  - It claims `DecideAutomaticResumeConfig` at `createflow.go`; the implementation is unexported `decideAutomaticResumeConfig` in `markers.go:62`.
  - It marks `cmd/internal/procutil/procutil.go` as modified, but that file is unchanged in the reviewed range; existing helpers were reused.

  The review contract explicitly makes any Core concepts contradiction Critical. Fix by either aligning code with the table or adding a `## Revisions` entry and correcting the entity name, location, and process-seam status to “existing/reused.”

### 3. Important findings

- Atlas updates are incomplete. Existing authoritative sections still describe the removed filename-only behavior:
  - `atlas/architecture.md:902` says Codex identity comes from a trailing filename UUID.
  - `atlas/architecture.md:913` describes the stale-ID check as a filename `find`.
  - `atlas/review-workbench.md:165` still describes Neovim review-target resolution falling through to live `agent-pid` rollout discovery.
  - `atlas/architecture.md:989` describes slug discovery without the metadata authorization requirement.

  Update these sections to match the shared first-event classifier and thin Neovim behavior.

- The planned launcher integration coverage was not delivered. `createflow_test.go:1217` and `:1253` use a fake boolean lookup with sentinel `"SUBAGENT"` values, while the plan required real on-disk subagent rollouts. `osruntime_test.go:332` only covers acceptance of a valid root. Add an OS-runtime rejection test using actual subagent metadata so classifier-to-quarantine wiring is pinned.

- Several promised regressions are absent: subagent-only cases for launcher/codexsid/slug, malformed-metadata-before-root watcher continuation, and an assertion that the headless Neovim test never invokes its `ps`/`lsof` stubs. The current headless stubs do not record or fail on invocation (`tests/review-toggle-test.sh:41-57`).

### 4. Minor findings

None.

### 5. Test coverage notes

Passed:

- Focused eight-package identity suite.
- `bash tests/review-toggle-test.sh`.
- `go test ./... -count=1` with inherited Pair variables cleared.
- `git diff --check`.

The classifier’s pure boundary and candidate-order behavior are well covered. The main gap is real filesystem integration at the launcher quarantine boundary.

### 6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass. Semantic authorization is centralized in `transcript`; process parsing is reused from `procutil`.
- `ARCH-PURE`: Pass. Classification and resume-state projection are pure; IO remains in thin adapters/runtimes.
- `ARCH-PURPOSE`: Pass for the implemented identity consumers. Root/subagent ordering, persisted-state quarantine, and removal of Neovim live discovery are delivered.
- `ARCH-MOCK`: Pass. Production and tests share the existing runtime/process seams; no new direct external call escaped those seams. Strengthening the launcher’s real-rollout test remains advisable.
- The shadow sweep also finds changelog config consumers in `cmd/internal/opener` and `nvim/init.lua`. They key changelog files rather than transcript/review identity, so they do not clearly violate this issue’s stated scope, but a future identity API would avoid further hand-parsed config consumers.

### 7. Plan revision recommendations

Add a `## Revisions` entry recording:

- `DecideAutomaticResumeConfig` became unexported `decideAutomaticResumeConfig` and lives in `cmd/internal/launcher/markers.go`.
- The shared `procutil` process seam was pre-existing and reused, not modified.
- Correct the verification record after adding the missing integration cases, or explicitly document any intentionally dropped cases.

