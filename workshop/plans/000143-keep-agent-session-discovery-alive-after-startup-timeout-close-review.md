# Boundary Review — 000143-keep-agent-session-discovery-alive-after-startup-timeout#143 (whole-issue close)

| field | value |
|-------|-------|
| issue | 143 — Keep agent session discovery alive after startup timeout |
| repo | 000143-keep-agent-session-discovery-alive-after-startup-timeout |
| issue file | workshop/issues/000143-keep-agent-session-discovery-alive-after-startup-timeout.md |
| boundary | whole-issue close |
| milestone | — |
| window | d08c0bec1669719090fdb5bb0f793e9db4272a12..HEAD |
| command | sdlc close --issue 143 |
| reviewer | codex |
| timestamp | 2026-08-18T22:10:30-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reading additional input from stdin...
OpenAI Codex v0.147.0
--------
workdir: /private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout
model: gpt-5.6-sol
provider: openai
approval: never
sandbox: workspace-write [workdir, /tmp, $TMPDIR]
reasoning effort: medium
reasoning summaries: none
session id: 01a0186b-ed9a-7113-908d-f1450f86ba5a
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **000143-keep-agent-session-discovery-alive-after-startup-timeout** repository.

- repository: 000143-keep-agent-session-discovery-alive-after-startup-timeout   (root: /private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout)
- issue:      000143-keep-agent-session-discovery-alive-after-startup-timeout#143   (file: workshop/issues/000143-keep-agent-session-discovery-alive-after-startup-timeout.md)
- window:     Base: d08c0bec1669719090fdb5bb0f793e9db4272a12   Head: HEAD

Review the **000143-keep-agent-session-discovery-alive-after-startup-timeout** repo and its tracker — the ariadne base-layer repo itself (changes here propagate to dependent repos). Do not assume any
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
diff --git a/cmd/internal/sessionwatch/run.go b/cmd/internal/sessionwatch/run.go
index f7e1489..c1a8883 100644
--- a/cmd/internal/sessionwatch/run.go
+++ b/cmd/internal/sessionwatch/run.go
@@ -29,6 +29,7 @@ type Options struct {
 	PIDWait  time.Duration
 	Timeout  time.Duration
 	Poll     time.Duration
+	SlowPoll time.Duration
 }
 
 // Runtime is the IO boundary for the session watcher.
@@ -71,6 +72,9 @@ func Run(opts Options, rt Runtime) error {
 	if opts.Poll <= 0 {
 		opts.Poll = 100 * time.Millisecond
 	}
+	if opts.SlowPoll <= 0 {
+		opts.SlowPoll = 60 * time.Second
+	}
 	repoRoot := opts.RepoRoot
 	if repoRoot == "" {
 		repoRoot = opts.Cwd
@@ -114,10 +118,14 @@ func Run(opts Options, rt Runtime) error {
 
 	nmLogged := false
 	deadline := watchStart.Add(opts.Timeout)
-	for rt.Now().Before(deadline) {
+	for {
 		if rootPID != "" && !rt.ProcessAlive(rootPID) {
 			return nil
 		}
+		if rootPID == "" && !rt.Now().Before(deadline) {
+			rt.Log(adapt.Fail, "no session id within startup deadline (agent="+opts.Agent+")")
+			return nil
+		}
 
 		result := discover(spec, rootPID, agentStart, legacyExisting, rt)
 		if result.ID != "" {
@@ -151,11 +159,12 @@ func Run(opts Options, rt Runtime) error {
 			nmLogged = true
 		}
 
-		rt.Sleep(opts.Poll)
+		poll := opts.Poll
+		if !rt.Now().Before(deadline) {
+			poll = opts.SlowPoll
+		}
+		rt.Sleep(poll)
 	}
-
-	rt.Log(adapt.Fail, "no session id within 60s deadline (agent="+opts.Agent+")")
-	return nil
 }
 
 func appendSessionLedger(rt Runtime, path string, entry sessionLedgerEntry) error {
diff --git a/cmd/internal/sessionwatch/run_test.go b/cmd/internal/sessionwatch/run_test.go
index c95daf1..fd9159f 100644
--- a/cmd/internal/sessionwatch/run_test.go
+++ b/cmd/internal/sessionwatch/run_test.go
@@ -192,16 +192,22 @@ func TestRunLogsNearMissOnce(t *testing.T) {
 	rt.alive["3000"] = true
 	rt.descendants["3000"] = []string{"3000"}
 	rt.lsof["3000"] = []string{bad}
+	rt.onSleep = func(d time.Duration) {
+		if d == time.Second {
+			rt.alive["3000"] = false
+		}
+	}
 
 	err := Run(Options{
-		Agent:   "codex",
-		Tag:     "tag",
-		Cwd:     "/repo",
-		Home:    home,
-		DataDir: data,
-		PIDWait: time.Second,
-		Timeout: 350 * time.Millisecond,
-		Poll:    100 * time.Millisecond,
+		Agent:    "codex",
+		Tag:      "tag",
+		Cwd:      "/repo",
+		Home:     home,
+		DataDir:  data,
+		PIDWait:  time.Second,
+		Timeout:  350 * time.Millisecond,
+		Poll:     100 * time.Millisecond,
+		SlowPoll: time.Second,
 	}, rt)
 	if err != nil {
 		t.Fatalf("Run error: %v", err)
@@ -210,8 +216,8 @@ func TestRunLogsNearMissOnce(t *testing.T) {
 	if got := rt.countLogs(adapt.NearMiss); got != 1 {
 		t.Fatalf("near-miss logs = %d, want 1; logs=%+v", got, rt.logs)
 	}
-	if !rt.hasLog(adapt.Fail, "no session id") {
-		t.Fatalf("logs = %+v, want fail after timeout", rt.logs)
+	if rt.hasLog(adapt.Fail, "no session id") {
+		t.Fatalf("logs = %+v, process-bound watch should exit without timeout failure", rt.logs)
 	}
 }
 
@@ -285,6 +291,106 @@ func TestRunContinuesPastLegacyNearMissToValidCandidate(t *testing.T) {
 	}
 }
 
+func TestRunDiscoversSessionAfterStartupTimeoutForEveryAsyncAgent(t *testing.T) {
+	tests := []struct {
+		agent string
+		sid   string
+		path  func(home, sid string) string
+	}{
+		{
+			agent: "codex",
+			sid:   "019eff64-6ceb-7e72-9d41-a735a97029ac",
+			path: func(home, sid string) string {
+				return home + "/.codex/sessions/2026/08/18/rollout-2026-08-18T14-47-32-" + sid + ".jsonl"
+			},
+		},
+		{
+			agent: "agy",
+			sid:   "123e4567-e89b-12d3-a456-426614174000",
+			path: func(home, sid string) string {
+				return home + "/.gemini/antigravity-cli/conversations/" + sid + ".db"
+			},
+		},
+		{
+			agent: "muse",
+			sid:   "223e4567-e89b-12d3-a456-426614174000",
+			path: func(home, sid string) string {
+				return home + "/.local/share/muse/sessions/" + sid + "/session.jsonl"
+			},
+		},
+	}
+
+	for _, tt := range tests {
+		t.Run(tt.agent, func(t *testing.T) {
+			home := "/tmp/home"
+			data := "/tmp/data"
+			rt := newFakeRuntime(time.Unix(500, 0))
+			rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("5000\n"), mod: rt.now}
+			rt.alive["5000"] = true
+			rt.descendants["5000"] = []string{"5000"}
+			rt.onSleep = func(d time.Duration) {
+				if d == time.Minute {
+					rt.lsof["5000"] = []string{tt.path(home, tt.sid)}
+				}
+			}
+
+			err := Run(Options{
+				Agent:   tt.agent,
+				Tag:     "tag",
+				Cwd:     "/repo",
+				Home:    home,
+				DataDir: data,
+				PIDWait: 100 * time.Millisecond,
+				Timeout: 300 * time.Millisecond,
+				Poll:    100 * time.Millisecond,
+			}, rt)
+			if err != nil {
+				t.Fatalf("Run error: %v", err)
+			}
+
+			got := string(rt.writes[filepath.Join(data, "config-tag-"+tt.agent+".json")])
+			if !strings.Contains(got, `"session_id":"`+tt.sid+`"`) {
+				t.Fatalf("config write = %s, want delayed session %s", got, tt.sid)
+			}
+			if got := countDuration(rt.sleeps, time.Minute); got != 1 {
+				t.Fatalf("slow sleeps = %d, want 1; all sleeps=%v", got, rt.sleeps)
+			}
+		})
+	}
+}
+
+func TestRunStopsAtSlowPollWhenBoundProcessExits(t *testing.T) {
+	data := "/tmp/data"
+	rt := newFakeRuntime(time.Unix(600, 0))
+	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("6000\n"), mod: rt.now}
+	rt.alive["6000"] = true
+	rt.onSleep = func(d time.Duration) {
+		if d == time.Minute {
+			rt.alive["6000"] = false
+		}
+	}
+
+	err := Run(Options{
+		Agent:   "codex",
+		Tag:     "tag",
+		Cwd:     "/repo",
+		Home:    "/tmp/home",
+		DataDir: data,
+		PIDWait: 100 * time.Millisecond,
+		Timeout: 300 * time.Millisecond,
+		Poll:    100 * time.Millisecond,
+	}, rt)
+	if err != nil {
+		t.Fatalf("Run error: %v", err)
+	}
+	if got := countDuration(rt.sleeps, time.Minute); got != 1 {
+		t.Fatalf("slow sleeps = %d, want 1; all sleeps=%v", got, rt.sleeps)
+	}
+	if _, ok := rt.writes[filepath.Join(data, "config-tag-codex.json")]; ok {
+		t.Fatal("config should not be written after the bound process exits")
+	}
+}
+
 func TestRunLogsFailOnTimeout(t *testing.T) {
 	rt := newFakeRuntime(time.Unix(400, 0))
 	err := Run(Options{
@@ -305,6 +411,16 @@ func TestRunLogsFailOnTimeout(t *testing.T) {
 	}
 }
 
+func countDuration(ds []time.Duration, want time.Duration) int {
+	var n int
+	for _, d := range ds {
+		if d == want {
+			n++
+		}
+	}
+	return n
+}
+
 type fakeFile struct {
 	content []byte
 	mod     time.Time
@@ -325,6 +441,7 @@ type fakeRuntime struct {
 	writes      map[string][]byte
 	writeErr    map[string]error
 	logs        []fakeLog
+	sleeps      []time.Duration
 	onSleep     func(time.Duration)
 }
 
@@ -343,6 +460,7 @@ func newFakeRuntime(now time.Time) *fakeRuntime {
 func (f *fakeRuntime) Now() time.Time { return f.now }
 
 func (f *fakeRuntime) Sleep(d time.Duration) {
+	f.sleeps = append(f.sleeps, d)
 	if f.onSleep != nil {
 		f.onSleep(d)
 	}


codex
I’ll verify the issue Spec/Plan, carry-forward ledger, repository guidance, full base-to-HEAD diff, and relevant tests. I’ll keep the review read-only and report only evidence-backed findings.
exec
/bin/zsh -lc "pwd && rg --files -g 'AGENTS.md' -g 'workshop/issues/000143*' -g 'workshop/plans/000143*' -g 'cmd/internal/sessionwatch/**' -g 'README.md' -g 'atlas/**' | sort" in /private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout
 succeeded in 0ms:
/private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout
README.md
atlas/.gitkeep
atlas/architecture.md
atlas/go-migration-inventory.md
atlas/how-to-bring-up-a-new-harness-cli.md
atlas/index.md
atlas/review-workbench.md
atlas/session-identity.md
cmd/internal/sessionwatch/run.go
cmd/internal/sessionwatch/run_test.go
cmd/internal/sessionwatch/runcli.go
cmd/internal/sessionwatch/runcli_test.go
cmd/internal/sessionwatch/runtime.go
cmd/internal/sessionwatch/sessionwatch.go
cmd/internal/sessionwatch/sessionwatch_test.go
doctor/README.md
workshop/issues/000143-keep-agent-session-discovery-alive-after-startup-timeout.md

exec
/bin/zsh -lc "find .. -name AGENTS.md -print; git status --short; git diff --stat d08c0bec1669719090fdb5bb0f793e9db4272a12...HEAD; git diff --name-status d08c0bec1669719090fdb5bb0f793e9db4272a12...HEAD; find workshop/plans -maxdepth 1 -type f -name '*000143*' -print" in /private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout
 succeeded in 0ms:
 cmd/internal/sessionwatch/run.go                   |  19 ++-
 cmd/internal/sessionwatch/run_test.go              | 138 +++++++++++++++++++--
 ...ession-discovery-alive-after-startup-timeout.md |  15 ++-
 3 files changed, 154 insertions(+), 18 deletions(-)
M	cmd/internal/sessionwatch/run.go
M	cmd/internal/sessionwatch/run_test.go
M	workshop/issues/000143-keep-agent-session-discovery-alive-after-startup-timeout.md

exec
/bin/zsh -lc "sed -n '1,260p' workshop/issues/000143-keep-agent-session-discovery-alive-after-startup-timeout.md; git diff --find-renames --find-copies d08c0bec1669719090fdb5bb0f793e9db4272a12...HEAD -- cmd/internal/sessionwatch/run.go cmd/internal/sessionwatch/run_test.go workshop/issues/000143-keep-agent-session-discovery-alive-after-startup-timeout.md; sed -n '1,280p' cmd/internal/sessionwatch/run.go; sed -n '1,240p' cmd/internal/sessionwatch/runtime.go; sed -n '1,220p' cmd/internal/sessionwatch/runcli.go" in /private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout
 succeeded in 0ms:
---
id: 000143
status: working
deps: []
github_issue:
created: 2026-08-18
updated: 2026-08-18
estimate_hours:
started: 2026-08-18T22:02:47-07:00
---

# Keep agent session discovery alive after startup timeout

## Problem

The asynchronous session watcher gives up after 60 seconds. Agents such as
Codex can create their transcript only after their first interaction, so an
agent left idle for longer than the startup window never gets a persisted
session ID. The context meter then has no transcript to read and the frame omits
context-window usage for the rest of the session.

## Spec

Treat the existing timeout as the end of fast startup discovery, not the end of
the watcher. When a fresh agent PID is available, continue discovery at a
low-frequency 60-second interval while that process is alive. Stop immediately
when a poll observes that the process has exited (within one slow-poll interval),
and preserve the existing bounded timeout when no fresh PID can be established.

Apply the lifecycle behavior uniformly to every asynchronous agent supported by
`sessionwatch` (Codex, Agy, and Muse). Claude supplies its session ID
synchronously and must remain unaffected.

## Done when

- A live supported agent can acquire and persist a session ID after the initial
  discovery timeout.
- Post-timeout discovery polls no more often than once every 60 seconds by
  default.
- The watcher exits within one slow-poll interval after the bound agent process
  exits, and still times out when no fresh PID exists.
- Automated tests cover delayed discovery for Codex, Agy, and Muse plus both
  exit paths.

## Plan

- [x] Test `Run` with the existing fake-clock/stateful-runtime seam, scheduling
  transcript and process-state transitions to guard the fast-to-slow cadence,
  every `AgentSpec`, PID-bound exit, and PID-less timeout.
- [x] Change the watcher loop to transition from fast polling to lifecycle-bound
  slow polling, using the existing injected runtime seam (ARCH-PURE,
  ARCH-MOCK).
- [x] Run focused and repository-wide verification; confirm the synchronous
  Claude launch path is unchanged (ARCH-PURPOSE, ARCH-DRY).

## Log

### 2026-08-18

- Root cause: the live Codex transcript appeared after the watcher's fixed
  60-second deadline, leaving `config-<tag>-codex.json` absent even though the
  transcript parser supported the current event format.
- Design approved by the operator: retain fast startup polling, then poll every
  60 seconds for the lifetime of the bound agent process. Apply this to all
  asynchronous agent specs rather than special-casing Codex.
- Implemented one shared `Run` schedule for Codex, Agy, and Muse: 100 ms during
  the startup deadline, then 60 seconds while the fresh bound PID remains alive.
  PID-less legacy discovery retains its bounded timeout; Claude remains on its
  existing synchronous session-ID path.
- TDD evidence: the delayed-discovery tests first failed because `SlowPoll` did
  not exist, then passed after the lifecycle loop was implemented. Focused
  package tests, all Go packages (with generated runtime assets and Pair session
  variables cleared), the shell session-watch integration test, and
  `git diff --check` pass.

## Revisions

### 2026-08-18 — Plan-quality review

- Clarified that process death is observed at the next 60-second slow poll,
  rather than promising impossible immediate detection during a blocking sleep.
- Recast the test plan as a function-level fake-clock strategy for `Run`.
diff --git a/cmd/internal/sessionwatch/run.go b/cmd/internal/sessionwatch/run.go
index f7e1489..c1a8883 100644
--- a/cmd/internal/sessionwatch/run.go
+++ b/cmd/internal/sessionwatch/run.go
@@ -29,6 +29,7 @@ type Options struct {
 	PIDWait  time.Duration
 	Timeout  time.Duration
 	Poll     time.Duration
+	SlowPoll time.Duration
 }
 
 // Runtime is the IO boundary for the session watcher.
@@ -71,6 +72,9 @@ func Run(opts Options, rt Runtime) error {
 	if opts.Poll <= 0 {
 		opts.Poll = 100 * time.Millisecond
 	}
+	if opts.SlowPoll <= 0 {
+		opts.SlowPoll = 60 * time.Second
+	}
 	repoRoot := opts.RepoRoot
 	if repoRoot == "" {
 		repoRoot = opts.Cwd
@@ -114,10 +118,14 @@ func Run(opts Options, rt Runtime) error {
 
 	nmLogged := false
 	deadline := watchStart.Add(opts.Timeout)
-	for rt.Now().Before(deadline) {
+	for {
 		if rootPID != "" && !rt.ProcessAlive(rootPID) {
 			return nil
 		}
+		if rootPID == "" && !rt.Now().Before(deadline) {
+			rt.Log(adapt.Fail, "no session id within startup deadline (agent="+opts.Agent+")")
+			return nil
+		}
 
 		result := discover(spec, rootPID, agentStart, legacyExisting, rt)
 		if result.ID != "" {
@@ -151,11 +159,12 @@ func Run(opts Options, rt Runtime) error {
 			nmLogged = true
 		}
 
-		rt.Sleep(opts.Poll)
+		poll := opts.Poll
+		if !rt.Now().Before(deadline) {
+			poll = opts.SlowPoll
+		}
+		rt.Sleep(poll)
 	}
-
-	rt.Log(adapt.Fail, "no session id within 60s deadline (agent="+opts.Agent+")")
-	return nil
 }
 
 func appendSessionLedger(rt Runtime, path string, entry sessionLedgerEntry) error {
diff --git a/cmd/internal/sessionwatch/run_test.go b/cmd/internal/sessionwatch/run_test.go
index c95daf1..fd9159f 100644
--- a/cmd/internal/sessionwatch/run_test.go
+++ b/cmd/internal/sessionwatch/run_test.go
@@ -192,16 +192,22 @@ func TestRunLogsNearMissOnce(t *testing.T) {
 	rt.alive["3000"] = true
 	rt.descendants["3000"] = []string{"3000"}
 	rt.lsof["3000"] = []string{bad}
+	rt.onSleep = func(d time.Duration) {
+		if d == time.Second {
+			rt.alive["3000"] = false
+		}
+	}
 
 	err := Run(Options{
-		Agent:   "codex",
-		Tag:     "tag",
-		Cwd:     "/repo",
-		Home:    home,
-		DataDir: data,
-		PIDWait: time.Second,
-		Timeout: 350 * time.Millisecond,
-		Poll:    100 * time.Millisecond,
+		Agent:    "codex",
+		Tag:      "tag",
+		Cwd:      "/repo",
+		Home:     home,
+		DataDir:  data,
+		PIDWait:  time.Second,
+		Timeout:  350 * time.Millisecond,
+		Poll:     100 * time.Millisecond,
+		SlowPoll: time.Second,
 	}, rt)
 	if err != nil {
 		t.Fatalf("Run error: %v", err)
@@ -210,8 +216,8 @@ func TestRunLogsNearMissOnce(t *testing.T) {
 	if got := rt.countLogs(adapt.NearMiss); got != 1 {
 		t.Fatalf("near-miss logs = %d, want 1; logs=%+v", got, rt.logs)
 	}
-	if !rt.hasLog(adapt.Fail, "no session id") {
-		t.Fatalf("logs = %+v, want fail after timeout", rt.logs)
+	if rt.hasLog(adapt.Fail, "no session id") {
+		t.Fatalf("logs = %+v, process-bound watch should exit without timeout failure", rt.logs)
 	}
 }
 
@@ -285,6 +291,106 @@ func TestRunContinuesPastLegacyNearMissToValidCandidate(t *testing.T) {
 	}
 }
 
+func TestRunDiscoversSessionAfterStartupTimeoutForEveryAsyncAgent(t *testing.T) {
+	tests := []struct {
+		agent string
+		sid   string
+		path  func(home, sid string) string
+	}{
+		{
+			agent: "codex",
+			sid:   "019eff64-6ceb-7e72-9d41-a735a97029ac",
+			path: func(home, sid string) string {
+				return home + "/.codex/sessions/2026/08/18/rollout-2026-08-18T14-47-32-" + sid + ".jsonl"
+			},
+		},
+		{
+			agent: "agy",
+			sid:   "123e4567-e89b-12d3-a456-426614174000",
+			path: func(home, sid string) string {
+				return home + "/.gemini/antigravity-cli/conversations/" + sid + ".db"
+			},
+		},
+		{
+			agent: "muse",
+			sid:   "223e4567-e89b-12d3-a456-426614174000",
+			path: func(home, sid string) string {
+				return home + "/.local/share/muse/sessions/" + sid + "/session.jsonl"
+			},
+		},
+	}
+
+	for _, tt := range tests {
+		t.Run(tt.agent, func(t *testing.T) {
+			home := "/tmp/home"
+			data := "/tmp/data"
+			rt := newFakeRuntime(time.Unix(500, 0))
+			rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("5000\n"), mod: rt.now}
+			rt.alive["5000"] = true
+			rt.descendants["5000"] = []string{"5000"}
+			rt.onSleep = func(d time.Duration) {
+				if d == time.Minute {
+					rt.lsof["5000"] = []string{tt.path(home, tt.sid)}
+				}
+			}
+
+			err := Run(Options{
+				Agent:   tt.agent,
+				Tag:     "tag",
+				Cwd:     "/repo",
+				Home:    home,
+				DataDir: data,
+				PIDWait: 100 * time.Millisecond,
+				Timeout: 300 * time.Millisecond,
+				Poll:    100 * time.Millisecond,
+			}, rt)
+			if err != nil {
+				t.Fatalf("Run error: %v", err)
+			}
+
+			got := string(rt.writes[filepath.Join(data, "config-tag-"+tt.agent+".json")])
+			if !strings.Contains(got, `"session_id":"`+tt.sid+`"`) {
+				t.Fatalf("config write = %s, want delayed session %s", got, tt.sid)
+			}
+			if got := countDuration(rt.sleeps, time.Minute); got != 1 {
+				t.Fatalf("slow sleeps = %d, want 1; all sleeps=%v", got, rt.sleeps)
+			}
+		})
+	}
+}
+
+func TestRunStopsAtSlowPollWhenBoundProcessExits(t *testing.T) {
+	data := "/tmp/data"
+	rt := newFakeRuntime(time.Unix(600, 0))
+	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("6000\n"), mod: rt.now}
+	rt.alive["6000"] = true
+	rt.onSleep = func(d time.Duration) {
+		if d == time.Minute {
+			rt.alive["6000"] = false
+		}
+	}
+
+	err := Run(Options{
+		Agent:   "codex",
+		Tag:     "tag",
+		Cwd:     "/repo",
+		Home:    "/tmp/home",
+		DataDir: data,
+		PIDWait: 100 * time.Millisecond,
+		Timeout: 300 * time.Millisecond,
+		Poll:    100 * time.Millisecond,
+	}, rt)
+	if err != nil {
+		t.Fatalf("Run error: %v", err)
+	}
+	if got := countDuration(rt.sleeps, time.Minute); got != 1 {
+		t.Fatalf("slow sleeps = %d, want 1; all sleeps=%v", got, rt.sleeps)
+	}
+	if _, ok := rt.writes[filepath.Join(data, "config-tag-codex.json")]; ok {
+		t.Fatal("config should not be written after the bound process exits")
+	}
+}
+
 func TestRunLogsFailOnTimeout(t *testing.T) {
 	rt := newFakeRuntime(time.Unix(400, 0))
 	err := Run(Options{
@@ -305,6 +411,16 @@ func TestRunLogsFailOnTimeout(t *testing.T) {
 	}
 }
 
+func countDuration(ds []time.Duration, want time.Duration) int {
+	var n int
+	for _, d := range ds {
+		if d == want {
+			n++
+		}
+	}
+	return n
+}
+
 type fakeFile struct {
 	content []byte
 	mod     time.Time
@@ -325,6 +441,7 @@ type fakeRuntime struct {
 	writes      map[string][]byte
 	writeErr    map[string]error
 	logs        []fakeLog
+	sleeps      []time.Duration
 	onSleep     func(time.Duration)
 }
 
@@ -343,6 +460,7 @@ func newFakeRuntime(now time.Time) *fakeRuntime {
 func (f *fakeRuntime) Now() time.Time { return f.now }
 
 func (f *fakeRuntime) Sleep(d time.Duration) {
+	f.sleeps = append(f.sleeps, d)
 	if f.onSleep != nil {
 		f.onSleep(d)
 	}
diff --git a/workshop/issues/000143-keep-agent-session-discovery-alive-after-startup-timeout.md b/workshop/issues/000143-keep-agent-session-discovery-alive-after-startup-timeout.md
index 85965bd..20356b7 100644
--- a/workshop/issues/000143-keep-agent-session-discovery-alive-after-startup-timeout.md
+++ b/workshop/issues/000143-keep-agent-session-discovery-alive-after-startup-timeout.md
@@ -44,13 +44,13 @@ synchronously and must remain unaffected.
 
 ## Plan
 
-- [ ] Test `Run` with the existing fake-clock/stateful-runtime seam, scheduling
+- [x] Test `Run` with the existing fake-clock/stateful-runtime seam, scheduling
   transcript and process-state transitions to guard the fast-to-slow cadence,
   every `AgentSpec`, PID-bound exit, and PID-less timeout.
-- [ ] Change the watcher loop to transition from fast polling to lifecycle-bound
+- [x] Change the watcher loop to transition from fast polling to lifecycle-bound
   slow polling, using the existing injected runtime seam (ARCH-PURE,
   ARCH-MOCK).
-- [ ] Run focused and repository-wide verification; confirm the synchronous
+- [x] Run focused and repository-wide verification; confirm the synchronous
   Claude launch path is unchanged (ARCH-PURPOSE, ARCH-DRY).
 
 ## Log
@@ -63,6 +63,15 @@ synchronously and must remain unaffected.
 - Design approved by the operator: retain fast startup polling, then poll every
   60 seconds for the lifetime of the bound agent process. Apply this to all
   asynchronous agent specs rather than special-casing Codex.
+- Implemented one shared `Run` schedule for Codex, Agy, and Muse: 100 ms during
+  the startup deadline, then 60 seconds while the fresh bound PID remains alive.
+  PID-less legacy discovery retains its bounded timeout; Claude remains on its
+  existing synchronous session-ID path.
+- TDD evidence: the delayed-discovery tests first failed because `SlowPoll` did
+  not exist, then passed after the lifecycle loop was implemented. Focused
+  package tests, all Go packages (with generated runtime assets and Pair session
+  variables cleared), the shell session-watch integration test, and
+  `git diff --check` pass.
 
 ## Revisions
 
package sessionwatch

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
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
	SlowPoll time.Duration
}

// Runtime is the IO boundary for the session watcher.
type Runtime interface {
	Now() time.Time
	Sleep(time.Duration)
	ReadFile(path string) ([]byte, error)
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
	if opts.SlowPoll <= 0 {
		opts.SlowPoll = 60 * time.Second
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
	for {
		if rootPID != "" && !rt.ProcessAlive(rootPID) {
			return nil
		}
		if rootPID == "" && !rt.Now().Before(deadline) {
			rt.Log(adapt.Fail, "no session id within startup deadline (agent="+opts.Agent+")")
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

		poll := opts.Poll
		if !rt.Now().Before(deadline) {
			poll = opts.SlowPoll
		}
		rt.Sleep(poll)
	}
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
				result := spec.Match(path)
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
		result := spec.Match(file)
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
		result := spec.Match(file)
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
package sessionwatch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

// OSRuntime implements Runtime with real process and filesystem calls.
type OSRuntime struct {
	logger *adapt.Logger
}

func NewOSRuntime(logger *adapt.Logger) OSRuntime {
	return OSRuntime{logger: logger}
}

func (OSRuntime) Now() time.Time { return time.Now() }
func (OSRuntime) Sleep(d time.Duration) {
	time.Sleep(d)
}

func (OSRuntime) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

func (OSRuntime) ModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func (OSRuntime) BirthTime(path string) (time.Time, error) {
	out, err := exec.Command("stat", "-f", "%B", path).Output()
	if err != nil {
		return time.Time{}, err
	}
	sec, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(sec, 0), nil
}

func (OSRuntime) ListFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

func (OSRuntime) Descendants(root string) ([]string, error) {
	out, err := exec.Command("ps", "-axo", "pid=,ppid=").Output()
	if err != nil {
		return []string{root}, nil
	}
	children := map[string][]string{}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		children[fields[1]] = append(children[fields[1]], fields[0])
	}
	queue := []string{root}
	seen := map[string]bool{root: true}
	for i := 0; i < len(queue); i++ {
		for _, child := range children[queue[i]] {
			if child == "" || seen[child] {
				continue
			}
			seen[child] = true
			queue = append(queue, child)
		}
	}
	return queue, nil
}

func (OSRuntime) LsofPaths(pid string) ([]string, error) {
	out, err := exec.Command("lsof", "-p", pid, "-Fn").Output()
	if err != nil {
		return nil, nil
	}
	var paths []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "n") {
			paths = append(paths, strings.TrimPrefix(line, "n"))
		}
	}
	return paths, nil
}

func (OSRuntime) ProcessAlive(pid string) bool {
	return procutil.Alive(pid)
}

func (OSRuntime) AtomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

func (r OSRuntime) Log(outcome adapt.Outcome, detail string) {
	r.logger.Log(3, "session-id", outcome, detail)
}

func ParseDurationSeconds(s string, fallback time.Duration) time.Duration {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return fallback
	}
	return time.Duration(n) * time.Second
}
package sessionwatch

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
)

// RunCLI is the pair-session-watch command body, shared by the
// bin/pair-session-watch shim and the `pair session-watch` dispatcher route. It
// parses argv into Options and drives the watcher; getenv/stderr are injected so
// it is testable, and it no-ops (exit 0) when required args are missing.
func RunCLI(args []string, getenv func(string) string, stderr io.Writer) int {
	opts, ok := buildOptions(args, getenv)
	if !ok {
		return 0
	}
	cleanupPairTag := ensurePairTag(opts.Tag)
	defer cleanupPairTag()
	logger := adapt.Open("session-watch", opts.Agent)
	defer logger.Close()
	if err := Run(opts, NewOSRuntime(logger)); err != nil {
		fmt.Fprintf(stderr, "pair-session-watch: %v\n", err)
		return 1
	}
	return 0
}

func ensurePairTag(tag string) func() {
	if os.Getenv("PAIR_TAG") != "" || tag == "" {
		return func() {}
	}
	_ = os.Setenv("PAIR_TAG", tag)
	return func() { _ = os.Unsetenv("PAIR_TAG") }
}

func buildOptions(args []string, getenv func(string) string) (Options, bool) {
	if len(args) < 3 {
		return Options{}, false
	}
	home := getenv("HOME")
	dataDir := getenv("PAIR_DATA_DIR")
	if dataDir == "" {
		dataDir = adapt.DataDir()
	}
	repoRoot := ""
	repoName := ""
	agentArgs := append([]string(nil), args[3:]...)
	for len(agentArgs) > 0 {
		if agentArgs[0] == "--" {
			agentArgs = append([]string(nil), agentArgs[1:]...)
			break
		}
		if len(agentArgs) >= 2 && agentArgs[0] == "--repo-root" {
			repoRoot = agentArgs[1]
			agentArgs = agentArgs[2:]
			continue
		}
		if len(agentArgs) >= 2 && agentArgs[0] == "--repo-name" {
			repoName = agentArgs[1]
			agentArgs = agentArgs[2:]
			continue
		}
		break
	}
	return Options{
		Agent:    args[0],
		Tag:      args[1],
		Cwd:      args[2],
		RepoRoot: repoRoot,
		RepoName: repoName,
		Args:     agentArgs,
		Home:     home,
		DataDir:  dataDir,
		PIDWait:  ParseDurationSeconds(getenv("PAIR_SESSION_WATCH_PID_WAIT_SECONDS"), 2*time.Second),
		Timeout:  60 * time.Second,
		Poll:     100 * time.Millisecond,
	}, true
}

exec
/bin/zsh -lc "sed -n '1,540p' cmd/internal/sessionwatch/run_test.go; sed -n '220,520p' cmd/internal/sessionwatch/run.go; rg -n \"session-watch|PAIR_SESSION_WATCH|SlowPoll|Timeout|Run\\(\" --glob '*.go' --glob '*.sh' --glob 'README.md' --glob 'atlas/**' . | head -n 300; git log --oneline --decorate d08c0bec1669719090fdb5bb0f793e9db4272a12..HEAD" in /private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout
 succeeded in 0ms:
package sessionwatch

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
)

func TestRunUsesFreshPidfileAndWritesConfig(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	sessionFile := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
	rt := newFakeRuntime(time.Unix(100, 0))
	rt.files[filepath.Join(data, "agent-pid-test")] = fakeFile{content: []byte("999999\n"), mod: time.Unix(1, 0)}
	rt.onSleep = func(time.Duration) {
		rt.files[filepath.Join(data, "agent-pid-test")] = fakeFile{content: []byte("1234\n"), mod: time.Unix(100, 0)}
	}
	rt.alive["1234"] = true
	rt.descendants["1234"] = []string{"1234", "5678"}
	rt.lsof["5678"] = []string{sessionFile}

	err := Run(Options{
		Agent:   "codex",
		Tag:     "test",
		Cwd:     "/repo",
		Args:    []string{"resume", "old", `say "hi"`},
		Home:    home,
		DataDir: data,
		PIDWait: 3 * time.Second,
		Timeout: 5 * time.Second,
		Poll:    100 * time.Millisecond,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	out := filepath.Join(data, "config-test-codex.json")
	got := string(rt.writes[out])
	if !strings.Contains(got, `"session_id":"`+sid+`"`) || strings.Contains(got, "old") || !strings.Contains(got, `say \"hi\"`) {
		t.Fatalf("config write = %s", got)
	}
	if !rt.hasLog(adapt.Fired, "session_id="+sid) {
		t.Fatalf("logs = %+v, want fired session id", rt.logs)
	}
	ledger := string(rt.writes[filepath.Join(data, "ledger-test.jsonl")])
	if !strings.Contains(ledger, `"agent":"codex"`) || !strings.Contains(ledger, `"session_id":"`+sid+`"`) || !strings.Contains(ledger, `"repo_root":"/repo"`) {
		t.Fatalf("ledger write = %s", ledger)
	}
	if strings.Contains(ledger, "old") || !strings.Contains(ledger, `say \"hi\"`) {
		t.Fatalf("ledger args = %s", ledger)
	}
}

func TestRunUsesRepoIdentityForLedgerWhenCwdIsSubdir(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	sessionFile := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
	rt := newFakeRuntime(time.Unix(100, 0))
	rt.files[filepath.Join(data, "agent-pid-test")] = fakeFile{content: []byte("1234\n"), mod: time.Unix(100, 0)}
	rt.alive["1234"] = true
	rt.descendants["1234"] = []string{"1234"}
	rt.lsof["1234"] = []string{sessionFile}

	err := Run(Options{
		Agent:    "codex",
		Tag:      "test",
		Cwd:      "/repo/cmd/pair",
		RepoRoot: "/repo",
		RepoName: "pair",
		Home:     home,
		DataDir:  data,
		PIDWait:  time.Second,
		Timeout:  time.Second,
		Poll:     100 * time.Millisecond,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	ledger := string(rt.writes[filepath.Join(data, "ledger-test.jsonl")])
	if !strings.Contains(ledger, `"repo_root":"/repo"`) || !strings.Contains(ledger, `"repo_name":"pair"`) {
		t.Fatalf("ledger write = %s, want repo identity rather than cwd-derived identity", ledger)
	}
	if strings.Contains(ledger, `/repo/cmd/pair`) || strings.Contains(ledger, `"repo_name":"cmd"`) {
		t.Fatalf("ledger write = %s, should not persist pane cwd as repo identity", ledger)
	}
}

func TestRunDoesNotWriteConfigWhenLedgerAppendFails(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	sessionFile := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
	rt := newFakeRuntime(time.Unix(100, 0))
	rt.files[filepath.Join(data, "agent-pid-test")] = fakeFile{content: []byte("1234\n"), mod: time.Unix(100, 0)}
	rt.alive["1234"] = true
	rt.descendants["1234"] = []string{"1234"}
	rt.lsof["1234"] = []string{sessionFile}
	rt.writeErr[filepath.Join(data, "ledger-test.jsonl")] = errors.New("ledger write failed")

	err := Run(Options{
		Agent:   "codex",
		Tag:     "test",
		Cwd:     "/repo",
		Home:    home,
		DataDir: data,
		PIDWait: time.Second,
		Timeout: time.Second,
		Poll:    100 * time.Millisecond,
	}, rt)
	if err == nil {
		t.Fatalf("Run error = nil, want ledger write error")
	}
	if _, ok := rt.writes[filepath.Join(data, "config-test-codex.json")]; ok {
		t.Fatalf("config should not be written when ledger append fails")
	}
}

func TestRunTreatsSameSecondPidfileAsFresh(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	sessionFile := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
	rt := newFakeRuntime(time.Unix(100, 900_000_000))
	rt.files[filepath.Join(data, "agent-pid-test")] = fakeFile{content: []byte("1234\n"), mod: time.Unix(100, 0)}
	rt.alive["1234"] = true
	rt.descendants["1234"] = []string{"1234"}
	rt.lsof["1234"] = []string{sessionFile}

	err := Run(Options{
		Agent:   "codex",
		Tag:     "test",
		Cwd:     "/repo",
		Home:    home,
		DataDir: data,
		PIDWait: time.Second,
		Timeout: time.Second,
		Poll:    100 * time.Millisecond,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if got := string(rt.writes[filepath.Join(data, "config-test-codex.json")]); !strings.Contains(got, sid) {
		t.Fatalf("config write = %s, want same-second pidfile accepted", got)
	}
}

func TestRunDiscoversAgySessionFromLsof(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	sid := "123e4567-e89b-12d3-a456-426614174000"
	sessionFile := home + "/.gemini/antigravity-cli/conversations/" + sid + ".db"
	rt := newFakeRuntime(time.Unix(200, 0))
	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("2000\n"), mod: time.Unix(200, 0)}
	rt.alive["2000"] = true
	rt.descendants["2000"] = []string{"2000"}
	rt.lsof["2000"] = []string{sessionFile}

	err := Run(Options{
		Agent:   "agy",
		Tag:     "tag",
		Cwd:     "/repo",
		Args:    []string{"--conversation", "keep"},
		Home:    home,
		DataDir: data,
		PIDWait: time.Second,
		Timeout: time.Second,
		Poll:    100 * time.Millisecond,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	got := string(rt.writes[filepath.Join(data, "config-tag-agy.json")])
	if !strings.Contains(got, `"session_id":"`+sid+`"`) || !strings.Contains(got, "--conversation") {
		t.Fatalf("agy config write = %s", got)
	}
}

func TestRunLogsNearMissOnce(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	bad := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-not-a-uuid.jsonl"
	rt := newFakeRuntime(time.Unix(300, 0))
	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("3000\n"), mod: time.Unix(300, 0)}
	rt.alive["3000"] = true
	rt.descendants["3000"] = []string{"3000"}
	rt.lsof["3000"] = []string{bad}
	rt.onSleep = func(d time.Duration) {
		if d == time.Second {
			rt.alive["3000"] = false
		}
	}

	err := Run(Options{
		Agent:    "codex",
		Tag:      "tag",
		Cwd:      "/repo",
		Home:     home,
		DataDir:  data,
		PIDWait:  time.Second,
		Timeout:  350 * time.Millisecond,
		Poll:     100 * time.Millisecond,
		SlowPoll: time.Second,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if got := rt.countLogs(adapt.NearMiss); got != 1 {
		t.Fatalf("near-miss logs = %d, want 1; logs=%+v", got, rt.logs)
	}
	if rt.hasLog(adapt.Fail, "no session id") {
		t.Fatalf("logs = %+v, process-bound watch should exit without timeout failure", rt.logs)
	}
}

func TestRunContinuesPastLsofNearMissToValidCandidate(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	bad := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-not-a-uuid.jsonl"
	good := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
	rt := newFakeRuntime(time.Unix(350, 0))
	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("3500\n"), mod: time.Unix(350, 0)}
	rt.alive["3500"] = true
	rt.descendants["3500"] = []string{"3500", "3501"}
	rt.lsof["3500"] = []string{bad}
	rt.lsof["3501"] = []string{good}

	err := Run(Options{
		Agent:   "codex",
		Tag:     "tag",
		Cwd:     "/repo",
		Home:    home,
		DataDir: data,
		PIDWait: time.Second,
		Timeout: time.Second,
		Poll:    100 * time.Millisecond,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	got := string(rt.writes[filepath.Join(data, "config-tag-codex.json")])
	if !strings.Contains(got, sid) {
		t.Fatalf("config write = %s, want valid sid after near miss", got)
	}
	if rt.countLogs(adapt.NearMiss) != 0 {
		t.Fatalf("near miss should not be logged when a valid candidate is found later: %+v", rt.logs)
	}
}

func TestRunContinuesPastLegacyNearMissToValidCandidate(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	bad := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-not-a-uuid.jsonl"
	good := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
	rt := newFakeRuntime(time.Unix(360, 0))
	var sleeps int
	rt.onSleep = func(time.Duration) {
		sleeps++
		if sleeps == 2 {
			rt.files[bad] = fakeFile{mod: time.Unix(360, 0)}
			rt.files[good] = fakeFile{mod: time.Unix(360, 0)}
		}
	}

	err := Run(Options{
		Agent:   "codex",
		Tag:     "tag",
		Cwd:     "/repo",
		Home:    home,
		DataDir: data,
		PIDWait: 100 * time.Millisecond,
		Timeout: time.Second,
		Poll:    100 * time.Millisecond,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	got := string(rt.writes[filepath.Join(data, "config-tag-codex.json")])
	if !strings.Contains(got, sid) {
		t.Fatalf("config write = %s, want valid sid after legacy near miss", got)
	}
}

func TestRunDiscoversSessionAfterStartupTimeoutForEveryAsyncAgent(t *testing.T) {
	tests := []struct {
		agent string
		sid   string
		path  func(home, sid string) string
	}{
		{
			agent: "codex",
			sid:   "019eff64-6ceb-7e72-9d41-a735a97029ac",
			path: func(home, sid string) string {
				return home + "/.codex/sessions/2026/08/18/rollout-2026-08-18T14-47-32-" + sid + ".jsonl"
			},
		},
		{
			agent: "agy",
			sid:   "123e4567-e89b-12d3-a456-426614174000",
			path: func(home, sid string) string {
				return home + "/.gemini/antigravity-cli/conversations/" + sid + ".db"
			},
		},
		{
			agent: "muse",
			sid:   "223e4567-e89b-12d3-a456-426614174000",
			path: func(home, sid string) string {
				return home + "/.local/share/muse/sessions/" + sid + "/session.jsonl"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			home := "/tmp/home"
			data := "/tmp/data"
			rt := newFakeRuntime(time.Unix(500, 0))
			rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("5000\n"), mod: rt.now}
			rt.alive["5000"] = true
			rt.descendants["5000"] = []string{"5000"}
			rt.onSleep = func(d time.Duration) {
				if d == time.Minute {
					rt.lsof["5000"] = []string{tt.path(home, tt.sid)}
				}
			}

			err := Run(Options{
				Agent:   tt.agent,
				Tag:     "tag",
				Cwd:     "/repo",
				Home:    home,
				DataDir: data,
				PIDWait: 100 * time.Millisecond,
				Timeout: 300 * time.Millisecond,
				Poll:    100 * time.Millisecond,
			}, rt)
			if err != nil {
				t.Fatalf("Run error: %v", err)
			}

			got := string(rt.writes[filepath.Join(data, "config-tag-"+tt.agent+".json")])
			if !strings.Contains(got, `"session_id":"`+tt.sid+`"`) {
				t.Fatalf("config write = %s, want delayed session %s", got, tt.sid)
			}
			if got := countDuration(rt.sleeps, time.Minute); got != 1 {
				t.Fatalf("slow sleeps = %d, want 1; all sleeps=%v", got, rt.sleeps)
			}
		})
	}
}

func TestRunStopsAtSlowPollWhenBoundProcessExits(t *testing.T) {
	data := "/tmp/data"
	rt := newFakeRuntime(time.Unix(600, 0))
	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("6000\n"), mod: rt.now}
	rt.alive["6000"] = true
	rt.onSleep = func(d time.Duration) {
		if d == time.Minute {
			rt.alive["6000"] = false
		}
	}

	err := Run(Options{
		Agent:   "codex",
		Tag:     "tag",
		Cwd:     "/repo",
		Home:    "/tmp/home",
		DataDir: data,
		PIDWait: 100 * time.Millisecond,
		Timeout: 300 * time.Millisecond,
		Poll:    100 * time.Millisecond,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if got := countDuration(rt.sleeps, time.Minute); got != 1 {
		t.Fatalf("slow sleeps = %d, want 1; all sleeps=%v", got, rt.sleeps)
	}
	if _, ok := rt.writes[filepath.Join(data, "config-tag-codex.json")]; ok {
		t.Fatal("config should not be written after the bound process exits")
	}
}

func TestRunLogsFailOnTimeout(t *testing.T) {
	rt := newFakeRuntime(time.Unix(400, 0))
	err := Run(Options{
		Agent:   "codex",
		Tag:     "tag",
		Cwd:     "/repo",
		Home:    "/tmp/home",
		DataDir: "/tmp/data",
		PIDWait: 100 * time.Millisecond,
		Timeout: 300 * time.Millisecond,
		Poll:    100 * time.Millisecond,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !rt.hasLog(adapt.Fail, "no session id") {
		t.Fatalf("logs = %+v, want fail", rt.logs)
	}
}

func countDuration(ds []time.Duration, want time.Duration) int {
	var n int
	for _, d := range ds {
		if d == want {
			n++
		}
	}
	return n
}

type fakeFile struct {
	content []byte
	mod     time.Time
	birth   time.Time
}

type fakeLog struct {
	outcome adapt.Outcome
	detail  string
}

type fakeRuntime struct {
	now         time.Time
	files       map[string]fakeFile
	alive       map[string]bool
	descendants map[string][]string
	lsof        map[string][]string
	writes      map[string][]byte
	writeErr    map[string]error
	logs        []fakeLog
	sleeps      []time.Duration
	onSleep     func(time.Duration)
}

func newFakeRuntime(now time.Time) *fakeRuntime {
	return &fakeRuntime{
		now:         now,
		files:       map[string]fakeFile{},
		alive:       map[string]bool{},
		descendants: map[string][]string{},
		lsof:        map[string][]string{},
		writes:      map[string][]byte{},
		writeErr:    map[string]error{},
	}
}

func (f *fakeRuntime) Now() time.Time { return f.now }

func (f *fakeRuntime) Sleep(d time.Duration) {
	f.sleeps = append(f.sleeps, d)
	if f.onSleep != nil {
		f.onSleep(d)
	}
	f.now = f.now.Add(d)
}

func (f *fakeRuntime) ReadFile(path string) ([]byte, error) {
	file, ok := f.files[path]
	if !ok {
		return nil, errors.New("missing")
	}
	return file.content, nil
}

func (f *fakeRuntime) ModTime(path string) (time.Time, error) {
	file, ok := f.files[path]
	if !ok {
		return time.Time{}, errors.New("missing")
	}
	return file.mod, nil
}

func (f *fakeRuntime) BirthTime(path string) (time.Time, error) {
	file, ok := f.files[path]
	if !ok {
		return time.Time{}, errors.New("missing")
	}
	if file.birth.IsZero() {
		return file.mod, nil
	}
	return file.birth, nil
}

func (f *fakeRuntime) ListFiles(root string) ([]string, error) {
	var out []string
	for path := range f.files {
		if strings.HasPrefix(path, root) {
			out = append(out, path)
		}
	}
	return out, nil
}

func (f *fakeRuntime) Descendants(root string) ([]string, error) {
	if out := f.descendants[root]; len(out) > 0 {
		return out, nil
	}
	return []string{root}, nil
}

func (f *fakeRuntime) LsofPaths(pid string) ([]string, error) { return f.lsof[pid], nil }
func (f *fakeRuntime) ProcessAlive(pid string) bool           { return f.alive[pid] }
func (f *fakeRuntime) AtomicWrite(path string, data []byte) error {
	if err := f.writeErr[path]; err != nil {
		return err
	}
	f.writes[path] = append([]byte(nil), data...)
	return nil
}
func (f *fakeRuntime) Log(outcome adapt.Outcome, detail string) {
	f.logs = append(f.logs, fakeLog{outcome: outcome, detail: detail})
}

func (f *fakeRuntime) hasLog(outcome adapt.Outcome, detail string) bool {
	for _, log := range f.logs {
		if log.outcome == outcome && strings.Contains(log.detail, detail) {
			return true
		}
	}
	return false
}

func (f *fakeRuntime) countLogs(outcome adapt.Outcome) int {
	var n int
	for _, log := range f.logs {
		if log.outcome == outcome {
			n++
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
		result := spec.Match(file)
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
		result := spec.Match(file)
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
./atlas/how-to-bring-up-a-new-harness-cli.md:63:- **Files:** `cmd/pair-session-watch` and `cmd/internal/sessionwatch` (the launcher spawns the Go binary directly since #94 M2 — the `.sh` shim was retired).
./atlas/how-to-bring-up-a-new-harness-cli.md:64:- Since TUI agents do not always expose session IDs on stdout, `pair-session-watch` runs in the background. It finds the agent process PID from `$PAIR_DATA_DIR/agent-pid-<tag>` (written by `pair-wrap`), walks its descendants, and inspects files held open by the processes via `lsof -p <pid>`.
./atlas/how-to-bring-up-a-new-harness-cli.md:86:**Telemetry Signal** (aspect `3`, see §3): `session-id` from `pair-session-watch` — `fired` when `AgentSpec.Match` resolves an id and the config is written, **`near-miss`** when a file matching the watch pattern is found but no id can be extracted (filename/format drift), `fail` when the 60s watch window elapses with no id at all (the session file never appeared where expected). The resume mapping in `bin/pair-shell` is the *consumer* of this id; it's static config with no separate signal.
./atlas/how-to-bring-up-a-new-harness-cli.md:143:3. [ ] **Implement Session Watching** in `cmd/internal/sessionwatch` / `cmd/pair-session-watch` (using `lsof` and target file patterns).
./atlas/how-to-bring-up-a-new-harness-cli.md:192:| 3 Session watch | `session-id` | pair-session-watch | fired, near-miss, fail | `fail` (timeout) / `near-miss` (file found, id unparsed) |
./atlas/go-migration-inventory.md:130:| `bin/pair-session-watch` / `cmd/pair-session-watch` / `cmd/internal/sessionwatch` | Go stateful watcher | launcher `SpawnSessionWatcher` (create path) | `pair-session-watch <agent> <tag> <cwd> [agent-args...]`; background 60s watcher; no-op for claude. | Reads agent pidfile, lsof/ps, native session dirs; writes config JSON atomically; logs adapt events through `cmd/internal/adapt`. | Go-owned watcher with implemented `pair session-watch` route (#92, via `sessionwatch.RunCLI`); the `.sh` passthrough shim was retired in #94 M2 (the launcher spawns `bin/pair-session-watch` directly) | P1 |
./atlas/go-migration-inventory.md:211:- #78 ported the session-id watcher to `cmd/pair-session-watch` with
./atlas/go-migration-inventory.md:212:  `bin/pair-session-watch.sh` retained as a shim (since retired in #94 M2 — the
./atlas/go-migration-inventory.md:213:  launcher spawns the Go `bin/pair-session-watch` directly).
./atlas/go-migration-inventory.md:251:  `continuation`, and `session-watch` gained shared `cmd/internal/<name>cmd`
./atlas/go-migration-inventory.md:258:  `session-watch` (long-running) use a new streaming seam in `cmd/pair-go`
./atlas/go-migration-inventory.md:268:  *shim name*, spawned by the then-shell launcher: `bin/pair-session-watch.sh`
./atlas/go-migration-inventory.md:276:  `Run(args, stdin, stdout, stderr) int` and leaving `cmd/pair-wrap` /
./atlas/go-migration-inventory.md:286:  `bin/clipboard-to-pane.sh`, `bin/pair-title.sh`, `bin/pair-session-watch.sh`.
./atlas/go-migration-inventory.md:292:  `bin/pair-title` / `bin/pair-session-watch` directly. All five were dropped from
./atlas/go-migration-inventory.md:333:  (`wrap`, `scribe`, `session-watch`, `title`, `context`, `slug`, `continuation`).
./atlas/go-migration-inventory.md:351:    `pair session-watch` via `selfPairExe` (os.Executable, works in the copied
./atlas/go-migration-inventory.md:430:- `bin/pair-session-watch.sh` (removed #94 M2 — `.sh` passthrough retired; `cmd/pair-session-watch` / `bin/pair-session-watch` is the owner, still bundled)
./atlas/go-migration-inventory.md:546:- `cmd/pair-session-watch/main.go`
./cmd/pair-go/main_test.go:48:	// session-watch with no args → buildOptions rejects (<3 args) → exit 0,
./cmd/pair-go/main_test.go:51:	code := runStreamingSubcommand("session-watch", nil, strings.NewReader(""), &stdout, &stderr)
./cmd/pair-go/main_test.go:108:		t.Run(tc.name, func(t *testing.T) {
./atlas/session-identity.md:12:  the launcher or `pair-session-watch`.
./cmd/pair-go/helper_equivalence_test.go:75:	err := cmd.Run()
./atlas/architecture.md:22:#     pair wrap · pair scribe · pair session-watch · pair title · pair context ·
./atlas/architecture.md:43:`pair session-watch`) reached inside a session because the launcher fronts pair's
./atlas/architecture.md:99:(#76), then `slug`, `changelog`, `continuation`, and `session-watch` (#92 M1).
./atlas/architecture.md:116:  spinner), `continuation` (reads the body from stdin), `session-watch`
./atlas/architecture.md:134:now invoke the Go binaries `bin/pair-session-watch` / `bin/pair-title` directly —
./atlas/architecture.md:845:  `pair-session-watch` codex/agy async path) → the **legacy unsuffixed base**
./atlas/architecture.md:889:2. **Watcher (`cmd/pair-session-watch` / `bin/pair-session-watch`, codex/agy only).** Spawned in the background by the launcher on the create path, right before the zellij launch — the launcher execs the Go binary directly (the `.sh` passthrough shim was retired in #94 M2). The stateful discovery logic lives in Go. Two discovery paths:
./atlas/architecture.md:891:   - **Legacy snapshot-diff (fallback).** Used when a fresh pidfile doesn't appear within 2s (`PAIR_SESSION_WATCH_PID_WAIT_SECONDS` in tests) — i.e., when the installed pair-wrap binary predates #000020 and doesn't publish the pidfile, or a stale pidfile is never refreshed. It snapshots the watch dir at start, scans new matching files, accepts the first candidate with a valid extracted id, and logs `near-miss` only when matching candidates cannot produce an id. Cross-tag races re-emerge in this path, so the proper resolution is to rebuild pair-wrap.
./atlas/architecture.md:979:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/config-<tag>-<agent>.json` — saved restart configuration for `(tag, agent)` (issue #000016, #000020). `{ agent, args, session_id }`. For claude, written synchronously by the launcher before zellij launch (`--session-id` is deterministic). For codex/agy, written by the Go `pair-session-watch` command once the agent's session file is discovered via lsof. Read by the launcher's create-flow prompt and by the post-Alt+x hint. Survives Alt+x (unlike `agent-<tag>`, which is cleared) — that's the whole point: it's the bridge between two pair launches against the same tag.
./atlas/architecture.md:981:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-pid-<tag>` — child agent PID written by `cmd/pair-wrap` immediately after `pty.Start`, removed on shutdown. Consumed by `cmd/pair-session-watch` to scope `lsof` discovery to a specific process tree (issue #000020). Mtime is also used as the agent-start epoch in the watcher's birth-time fallback.
./atlas/architecture.md:991:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/adapt-<tag>.jsonl` — the adaptation flight recorder (issue #000045). One JSON line per harness-adaptation trigger (`{ts, comp, agent, aspect, signal, outcome, detail}`), appended concurrently by `cmd/pair-wrap`, `cmd/pair-slug`, `cmd/pair-session-watch`, and `nvim/adapt.lua` — all writing one shared schema. Truncated once at session launch by the launcher (so multi-process `O_APPEND` never races) and removed on full quit. Read by `doctor/doctor.sh` to surface integration drift (near-miss/fail signals). See `atlas/how-to-bring-up-a-new-harness-cli.md` §3 for the signal registry.
./tests/pair-embedded-runtime-test.sh:50:    test ! -e "$root/bin/pair-session-watch"
./cmd/pair-go/main.go:76:// or a long lifetime (session-watch, title) — straight to their runner with
./cmd/pair-go/main.go:85:		return wrapcmd.Run(rest, stdin, stdout, stderr)
./cmd/pair-go/main.go:87:		return termcmd.Run(rest, stdin, stdout, stderr)
./cmd/pair-go/main.go:89:		return scribecmd.Run(rest, stdin, stdout, stderr)
./cmd/pair-go/main.go:91:		return changelogcmd.Run(rest, stderr)
./cmd/pair-go/main.go:93:		return continuationcmd.Run(rest, stdin, stdout, stderr, time.Now)
./cmd/pair-go/main.go:94:	case "session-watch":
./cmd/internal/slugcmd/slugcmd.go:105:func Run() int {
./cmd/internal/slugcmd/slugcmd.go:181:	out, err := model.Run(model.Request{
./tests/pair-session-watch-test.sh:5:RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-session-watch-test.XXXXXX")"
./tests/pair-session-watch-test.sh:35:PAIR_SESSION_WATCH_PID_WAIT_SECONDS=3 \
./tests/pair-session-watch-test.sh:36:"$ROOT/bin/pair" session-watch codex test "$ROOT" resume old-session 'say "hi"' --no-alt-screen &
./tests/pair-session-watch-test.sh:56:echo "pair session-watch stale pidfile test PASS"
./cmd/internal/keyscmd/keyscmd.go:22:func Run(args []string, stdout, stderr io.Writer) int {
./cmd/internal/scribecmd/scribecmd.go:21:// Packaging: the logic lives here behind Run(); it is reached two ways with
./cmd/internal/scribecmd/scribecmd.go:47:func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
./cmd/internal/slugcmd/slug_test.go:251:	t.Run("valid new stomps left with branch", func(t *testing.T) {
./cmd/internal/slugcmd/slug_test.go:262:	t.Run("KEEP, branch changed → write (left refreshes, focus kept)", func(t *testing.T) {
./cmd/internal/slugcmd/slug_test.go:269:	t.Run("KEEP, same branch + focus → no write", func(t *testing.T) {
./cmd/internal/slugcmd/slug_test.go:275:	t.Run("invalid → no write (keep last)", func(t *testing.T) {
./cmd/internal/slugcmd/slug_test.go:281:	t.Run("cold start: no prev, valid → write", func(t *testing.T) {
./cmd/internal/slugcmd/slug_test.go:288:	t.Run("cold start: no prev, KEEP → no write", func(t *testing.T) {
./cmd/internal/slugcmd/slug_test.go:294:	t.Run("preamble before slug → uses last line", func(t *testing.T) {
./cmd/internal/slugcmd/slug_test.go:301:	t.Run("value == prev → no write (unchanged)", func(t *testing.T) {
./cmd/internal/slugcmd/slug_test.go:308:	t.Run("focus containing | → no write", func(t *testing.T) {
./cmd/internal/slugcmd/slug_test.go:314:	t.Run("focus containing === → no write", func(t *testing.T) {
./cmd/internal/reviewcmd/run_test.go:162:		t.Run(tc.name, func(t *testing.T) {
./cmd/internal/keyscmd/keyscmd_test.go:12:	if code := Run(nil, &stdout, &stderr); code != 0 {
./cmd/internal/keyscmd/keyscmd_test.go:33:	Run(nil, &plain, &stderr)
./cmd/internal/keyscmd/keyscmd_test.go:34:	Run([]string{"--center", "120"}, &centered, &stderr)
./cmd/internal/keyscmd/keyscmd_test.go:45:	if code := Run([]string{"--center", "not-a-number"}, &stdout, &stderr); code != 0 {
./cmd/internal/keyscmd/keyscmd_test.go:75:	if code := Run([]string{"--bogus"}, &stdout, &stderr); code != 2 {
./cmd/internal/keyscmd/keyscmd_test.go:85:	Run(nil, &plain, &stderr)
./cmd/internal/keyscmd/keyscmd_test.go:86:	if code := Run([]string{"--center=120"}, &eq, &stderr); code != 0 {
./cmd/internal/scribecmd/scribecmd_test.go:27:		t.Run(c.name, func(t *testing.T) {
./cmd/internal/scribecmd/scribecmd_test.go:29:			code := Run(c.args, strings.NewReader(""), &stdout, &stderr)
./cmd/internal/scribecmd/scribecmd_test.go:60:	code := Run(args, tty, &stdout, io.Discard)
./cmd/internal/layoutcmd/resizeplan_test.go:29:		t.Run(tt.name, func(t *testing.T) {
./cmd/internal/scrollbackcmd/scrollbackcmd.go:391:func Run(argv []string, stdout, stderr io.Writer) int {
./cmd/internal/continuationcmd/continuationcmd.go:28:func Run(args []string, stdin io.Reader, stdout, stderr io.Writer, now func() time.Time) int {
./cmd/internal/continuationcmd/continuationcmd.go:64:		return newContinueRestartCmd(exe, slug, stdin, stdout, stderr).Run()
./cmd/internal/launcher/pathenv_test.go:33:		t.Run(tc.name, func(t *testing.T) {
./cmd/internal/reviewcmd/runtime.go:22:func (OSRuntime) Kill(pid string)              { _ = exec.Command("kill", pid).Run() }
./cmd/internal/reviewcmd/runtime.go:96:	return cmd.Run()
./cmd/internal/scrollbackcmd/scrollbackcmd_test.go:14:	code := Run([]string{}, io.Discard, &stderr)
./cmd/internal/scrollbackcmd/scrollbackcmd_test.go:37:	code := Run([]string{rawPath, evPath, outPath}, io.Discard, &stderr)
./cmd/internal/launcher/runcli.go:96:		ParkPromptTimeout:    parkPromptTimeout(),
./cmd/internal/launcher/runcli.go:156:// parkPromptTimeout reads PAIR_PARK_PROMPT_TIMEOUT (default 5, invalid → 5); a
./cmd/internal/launcher/runcli.go:158:func parkPromptTimeout() int {
./cmd/internal/launcher/layoutflow_test.go:81:		t.Run(tc.name, func(t *testing.T) {
./cmd/internal/launcher/layoutflow_test.go:169:		t.Run(tc.name, func(t *testing.T) {
./cmd/internal/termcmd/rename_test.go:87:		t.Run(tt.name, func(t *testing.T) {
./cmd/internal/termcmd/rename_input_test.go:50:		t.Run(tt.name, func(t *testing.T) {
./cmd/internal/termcmd/rename_input_test.go:70:		t.Run(text, func(t *testing.T) {
./cmd/internal/termcmd/rename_input_test.go:84:		t.Run(fmt.Sprintf("%q", seq), func(t *testing.T) {
./cmd/internal/termcmd/rename_input_test.go:94:		t.Run(fmt.Sprintf("%q", seq), func(t *testing.T) {
./cmd/internal/termcmd/rename_input_test.go:119:func TestDecodeRenameInputEscapeTimeoutAndEOF(t *testing.T) {
./cmd/internal/termcmd/rename_input_test.go:159:		t.Run(fmt.Sprintf("%q", seq), func(t *testing.T) {
./cmd/internal/layoutcmd/layoutcmd.go:192:	return exec.Command("zellij", cmdArgs...).Run()
./cmd/internal/launcher/osruntime_test.go:400:// subcommand — #104 M2 folded pair-title/pair-session-watch into `pair title` /
./cmd/internal/launcher/osruntime_test.go:401:// `pair session-watch`. spawnDetached swallows a start error, so a regression in
./cmd/internal/launcher/osruntime_test.go:413:	want := []string{exe, "session-watch", "codex", "work", "/cwd/sub", "--repo-root", "/cwd", "--repo-name", "pair", "--", "--no-alt-screen"}
./cmd/internal/launcher/osruntime_test.go:421:		if strings.HasSuffix(argv[0], ".sh") || strings.HasSuffix(argv[0], "pair-title") || strings.HasSuffix(argv[0], "pair-session-watch") {
./cmd/internal/termcmd/run.go:46:func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
./cmd/internal/termcmd/run.go:1177:	return cmd.Run()
./cmd/internal/launcher/runcli_test.go:17:		t.Run(arg, func(t *testing.T) {
./cmd/internal/launcher/runcli_test.go:32:		t.Run(arg, func(t *testing.T) {
./cmd/internal/launcher/runcli_test.go:135:	if err := exec.Command("git", "-C", repo, "init").Run(); err != nil {
./cmd/internal/launcher/agent_defaults_test.go:36:		t.Run(tc.name, func(t *testing.T) {
./cmd/internal/transcript/transcript.go:31:// config-<tag>-<agent>.json (written by bin/pair / pair-session-watch).
./cmd/internal/termcmd/queries_test.go:32:		t.Run(row.name, func(t *testing.T) {
./cmd/internal/termcmd/queries_test.go:74:		t.Run(k.name, func(t *testing.T) {
./cmd/internal/termcmd/queries_test.go:96:		t.Run(tt.name, func(t *testing.T) {
./cmd/internal/termcmd/queries_test.go:155:		t.Run(reply, func(t *testing.T) {
./cmd/internal/contextcmd/contextcmd.go:29:func Run(args []string, env Env, stdout io.Writer) int {
./cmd/internal/launcher/agentargs.go:165:// pair + pair-session-watch detection assume that position), so its token goes
./cmd/internal/launcher/run.go:37:func Run(argv []string, env Env, sessions SessionSource, history HistoricalScanner) (LaunchOutcome, error) {
./cmd/internal/termcmd/run_test.go:54:		t.Run(tt.name, func(t *testing.T) {
./cmd/internal/termcmd/run_test.go:78:		t.Run(chord, func(t *testing.T) {
./cmd/internal/termcmd/run_test.go:183:		t.Run(tt.name, func(t *testing.T) {
./cmd/internal/termcmd/run_test.go:200:		t.Run(fmt.Sprintf("%q", seq), func(t *testing.T) {
./cmd/internal/termcmd/run_test.go:528:func TestPumpStdinRenameEscapeTimeoutThenNextReadForwards(t *testing.T) {
./cmd/internal/termcmd/run_test.go:653:		t.Run(tt.name, func(t *testing.T) {
./cmd/internal/contextcmd/contextcmd_test.go:26:	code := Run([]string{"T", "claude"}, Env{Home: home, PairDataDir: data}, &stdout)
./cmd/internal/contextcmd/contextcmd_test.go:38:	code := Run([]string{"T", "claude"}, Env{Home: home, PairDataDir: filepath.Join(home, "empty")}, &stdout)
./cmd/internal/opener/runtime.go:32:	if code := scrollbackcmd.Run([]string{raw, events, ansi}, io.Discard, io.Discard); code != 0 {
./cmd/internal/opener/runtime.go:154:	return cmd.Run()
./cmd/internal/sessionwatch/runcli_test.go:14:		"PAIR_SESSION_WATCH_PID_WAIT_SECONDS": "3",
./cmd/internal/sessionwatch/runcli_test.go:28:	if opts.PIDWait != 3*time.Second || opts.Timeout != 60*time.Second || opts.Poll != 100*time.Millisecond {
./cmd/internal/launcher/scope_test.go:58:		t.Run(tc.in, func(t *testing.T) {
./cmd/internal/contextcmd/panejson_kdl_test.go:99:		t.Run(layout, func(t *testing.T) {
./cmd/internal/runtimebundle/manifest_test.go:51:		t.Run(tt.name, func(t *testing.T) {
./cmd/internal/sessionwatch/run_test.go:27:	err := Run(Options{
./cmd/internal/sessionwatch/run_test.go:35:		Timeout: 5 * time.Second,
./cmd/internal/sessionwatch/run_test.go:70:	err := Run(Options{
./cmd/internal/sessionwatch/run_test.go:79:		Timeout:  time.Second,
./cmd/internal/sessionwatch/run_test.go:107:	err := Run(Options{
./cmd/internal/sessionwatch/run_test.go:114:		Timeout: time.Second,
./cmd/internal/sessionwatch/run_test.go:136:	err := Run(Options{
./cmd/internal/sessionwatch/run_test.go:143:		Timeout: time.Second,
./cmd/internal/sessionwatch/run_test.go:165:	err := Run(Options{
./cmd/internal/sessionwatch/run_test.go:173:		Timeout: time.Second,
./cmd/internal/sessionwatch/run_test.go:201:	err := Run(Options{
./cmd/internal/sessionwatch/run_test.go:208:		Timeout:  350 * time.Millisecond,
./cmd/internal/sessionwatch/run_test.go:210:		SlowPoll: time.Second,
./cmd/internal/sessionwatch/run_test.go:237:	err := Run(Options{
./cmd/internal/sessionwatch/run_test.go:244:		Timeout: time.Second,
./cmd/internal/sessionwatch/run_test.go:275:	err := Run(Options{
./cmd/internal/sessionwatch/run_test.go:282:		Timeout: time.Second,
./cmd/internal/sessionwatch/run_test.go:294:func TestRunDiscoversSessionAfterStartupTimeoutForEveryAsyncAgent(t *testing.T) {
./cmd/internal/sessionwatch/run_test.go:324:		t.Run(tt.agent, func(t *testing.T) {
./cmd/internal/sessionwatch/run_test.go:337:			err := Run(Options{
./cmd/internal/sessionwatch/run_test.go:344:				Timeout: 300 * time.Millisecond,
./cmd/internal/sessionwatch/run_test.go:362:func TestRunStopsAtSlowPollWhenBoundProcessExits(t *testing.T) {
./cmd/internal/sessionwatch/run_test.go:373:	err := Run(Options{
./cmd/internal/sessionwatch/run_test.go:380:		Timeout: 300 * time.Millisecond,
./cmd/internal/sessionwatch/run_test.go:394:func TestRunLogsFailOnTimeout(t *testing.T) {
./cmd/internal/sessionwatch/run_test.go:396:	err := Run(Options{
./cmd/internal/sessionwatch/run_test.go:403:		Timeout: 300 * time.Millisecond,
./cmd/internal/launcher/zellij.go:58:	if err := cmd.Run(); err != nil {
./cmd/internal/launcher/createflow.go:77:		runCleanup(env, rt, step, opts.ParkPromptTimeout, stderr)
./cmd/internal/runtimebundle/embed_test.go:41:		"bin/pair-session-watch",
./cmd/internal/sessionwatch/sessionwatch_test.go:142:		t.Run(tt.name, func(t *testing.T) {
./cmd/internal/sessionwatch/run.go:30:	Timeout  time.Duration
./cmd/internal/sessionwatch/run.go:32:	SlowPoll time.Duration
./cmd/internal/sessionwatch/run.go:61:func Run(opts Options, rt Runtime) error {
./cmd/internal/sessionwatch/run.go:69:	if opts.Timeout <= 0 {
./cmd/internal/sessionwatch/run.go:70:		opts.Timeout = 60 * time.Second
./cmd/internal/sessionwatch/run.go:75:	if opts.SlowPoll <= 0 {
./cmd/internal/sessionwatch/run.go:76:		opts.SlowPoll = 60 * time.Second
./cmd/internal/sessionwatch/run.go:120:	deadline := watchStart.Add(opts.Timeout)
./cmd/internal/sessionwatch/run.go:164:			poll = opts.SlowPoll
./cmd/internal/launcher/createlogic.go:17:// session-watcher both write ({agent, args, session_id}).
./cmd/internal/sessionwatch/runcli.go:12:// RunCLI is the pair-session-watch command body, shared by the
./cmd/internal/sessionwatch/runcli.go:13:// bin/pair-session-watch shim and the `pair session-watch` dispatcher route. It
./cmd/internal/sessionwatch/runcli.go:23:	logger := adapt.Open("session-watch", opts.Agent)
./cmd/internal/sessionwatch/runcli.go:25:	if err := Run(opts, NewOSRuntime(logger)); err != nil {
./cmd/internal/sessionwatch/runcli.go:26:		fmt.Fprintf(stderr, "pair-session-watch: %v\n", err)
./cmd/internal/sessionwatch/runcli.go:78:		PIDWait:  ParseDurationSeconds(getenv("PAIR_SESSION_WATCH_PID_WAIT_SECONDS"), 2*time.Second),
./cmd/internal/sessionwatch/runcli.go:79:		Timeout:  60 * time.Second,
./cmd/internal/launcher/osruntime.go:46:const zjTimeout = 5 * time.Second
./cmd/internal/launcher/osruntime.go:52:	ctx, cancel := context.WithTimeout(context.Background(), zjTimeout)
./cmd/internal/launcher/osruntime.go:86:	ctx, cancel := context.WithTimeout(context.Background(), zjTimeout)
./cmd/internal/launcher/osruntime.go:105:	ctx, cancel := context.WithTimeout(context.Background(), zjTimeout)
./cmd/internal/launcher/osruntime.go:125:	err := cmd.Run()
./cmd/internal/launcher/osruntime.go:319:// sidecar spawns (#104 M2 folded pair-title/pair-session-watch into `pair title`
./cmd/internal/launcher/osruntime.go:320:// / `pair session-watch`). The launcher is always `pair`, so os.Executable() is
./cmd/internal/launcher/osruntime.go:341:	return append([]string{exe, "session-watch", agent, tag, cwd, "--repo-root", repoRoot, "--repo-name", repoName, "--"}, agentArgs...)
./cmd/internal/launcher/osruntime.go:355:	cmd.Run()
./cmd/internal/launcher/osruntime.go:427:	exec.Command("cmux", "rename-workspace", title).Run()
./cmd/internal/launcher/osruntime.go:695:	ctx, cancel := context.WithTimeout(context.Background(), zjTimeout)
./cmd/internal/launcher/osruntime.go:705:func pkillF(pattern string) { _ = exec.Command("pkill", "-9", "-f", pattern).Run() }
./cmd/internal/launcher/osruntime.go:714:				_ = exec.Command("kill", "-9", pid).Run()
./cmd/internal/launcher/osruntime.go:841:			_ = exec.Command("kill", pid).Run()
./cmd/internal/model/model.go:28:	// Timeout bounds a single model call so a hung child never leaves the
./cmd/internal/model/model.go:30:	Timeout = 30 * time.Second
./cmd/internal/model/model.go:52:	Timeout         time.Duration // per-call timeout; 0 → package Timeout default
./cmd/internal/model/model.go:55:// timeout returns the per-call timeout, defaulting to the package Timeout.
./cmd/internal/model/model.go:57:	if r.Timeout > 0 {
./cmd/internal/model/model.go:58:		return r.Timeout
./cmd/internal/model/model.go:60:	return Timeout
./cmd/internal/model/model.go:75:func Run(r Request) (string, error) {
./cmd/internal/model/model.go:107:	ctx, cancel := context.WithTimeout(context.Background(), r.timeout())
./cmd/internal/model/model.go:126:	ctx, cancel := context.WithTimeout(context.Background(), r.timeout())
./cmd/internal/model/model.go:139:	ctx, cancel := context.WithTimeout(context.Background(), r.timeout())
./cmd/internal/model/model.go:153:	ctx, cancel := context.WithTimeout(context.Background(), r.timeout())
./cmd/internal/model/model.go:192:	ctx, cancel := context.WithTimeout(context.Background(), r.timeout())
./cmd/internal/launcher/layout_test.go:71:		t.Run(tc.name, func(t *testing.T) {
./cmd/internal/dispatcher/dispatcher.go:61:		{Name: "session-watch", Summary: "async codex/agy session-id discovery", Status: "implemented", Streaming: true},
./cmd/internal/dispatcher/dispatcher.go:166:		// session-watch/title/clip copy-on-select) are intercepted by cmd/pair-go's
./cmd/internal/dispatcher/dispatcher.go:221:	code := contextcmd.Run(args, contextcmd.EnvFromOS(), &stdout)
./cmd/internal/dispatcher/dispatcher.go:227:	code := keyscmd.Run(args, &stdout, &stderr)
./cmd/internal/dispatcher/dispatcher.go:233:	code := scrollbackcmd.Run(args, &stdout, &stderr)
./cmd/internal/dispatcher/dispatcher.go:257:	return Result{ExitCode: slugcmd.Run()}
./cmd/internal/launcher/args_test.go:85:		t.Run(tc.name, func(t *testing.T) {
./cmd/internal/launcher/args_test.go:131:		t.Run(tc.name, func(t *testing.T) {
./cmd/internal/launcher/args_test.go:166:		t.Run(strings.Join(argv, " "), func(t *testing.T) {
./cmd/internal/launcher/args_test.go:226:		t.Run(flag, func(t *testing.T) {
./cmd/internal/launcher/args_test.go:252:		t.Run(verb, func(t *testing.T) {
./cmd/internal/launcher/args_test.go:284:		t.Run(tc.name, func(t *testing.T) {
./cmd/internal/clipcmd/runtime.go:64:		return cmd.Run()
./cmd/internal/clipcmd/runtime.go:97:	_ = exec.Command("zellij", "action", "set-pane-color", "--pane-id", id, "--bg", bg).Run()
./cmd/internal/clipcmd/runtime.go:142:	if err := exec.Command("zellij", "action", "focus-pane-id", id).Run(); err == nil {
./cmd/internal/clipcmd/runtime.go:145:	return exec.Command("zellij", "action", "focus-pane-id", "terminal_"+id).Run()
./cmd/internal/clipcmd/runtime.go:149:	_ = exec.Command("zellij", "action", "move-focus", dir).Run()
./cmd/internal/clipcmd/runtime.go:153:	_ = exec.Command("zellij", "action", "write", strconv.Itoa(int(b))).Run()
./cmd/internal/clipcmd/runtime.go:159:	return exec.Command(path, args...).Run()
./cmd/internal/model/model_test.go:68:	got, err := Run(Request{
./cmd/internal/model/model_test.go:113:	got, err := Run(Request{Agent: "codex", Model: "gpt-test-mini", Prompt: "prompt text", Input: "input text"})
./cmd/internal/model/model_test.go:138:	_, err := Run(Request{Agent: "codex", Model: "gpt-test-mini", Prompt: "p", Input: "i"})
./cmd/internal/launcher/datadir_test.go:18:		t.Run(tc.name, func(t *testing.T) {
./cmd/internal/launcher/runtime.go:83:	// SpawnSessionWatcher backgrounds bin/pair-session-watch (detached) to
./cmd/internal/launcher/runtime.go:257:	ParkPromptTimeout    int    // PAIR_PARK_PROMPT_TIMEOUT (default 5): the quit park-nudge [y/N] bound
./cmd/internal/launcher/readiness_test.go:26:		t.Run(tc.name, func(t *testing.T) {
./cmd/internal/readiness/record_test.go:34:		t.Run(raw, func(t *testing.T) {
./cmd/internal/dispatcher/dispatcher_test.go:16:	for _, want := range []string{"agent", "context", "layout", "scrollback", "wrap", "term", "slug", "changelog", "continuation", "session-watch", "scribe", "review", "clip", "title", "keys"} {
./cmd/internal/dispatcher/dispatcher_test.go:31:	for _, s := range []string{"wrap", "term", "scribe", "changelog render", "continuation", "session-watch", "title", "clip copy-on-select"} {
./cmd/internal/dispatcher/dispatcher_test.go:112:		t.Run(strings.Join(args, "_"), func(t *testing.T) {
./cmd/internal/launcher/tag_test.go:15:		t.Run(tc.name, func(t *testing.T) {
./cmd/internal/launcher/tag_test.go:29:		t.Run(raw, func(t *testing.T) {
./cmd/internal/launcher/tag_test.go:47:		t.Run(tc.cwd, func(t *testing.T) {
./cmd/internal/entrypoint/mode_test.go:6:	names := []string{"slug", "changelog", "continuation", "session-watch", "context", "scrollback-render"}
./cmd/internal/entrypoint/mode_test.go:25:		t.Run(c.name, func(t *testing.T) {
./cmd/internal/entrypoint/mode_test.go:50:		t.Run(c.name, func(t *testing.T) {
./cmd/internal/changelogcmd/run_test.go:84:	code := Run([]string{
./cmd/internal/changelogcmd/run_test.go:109:func TestFirstRun(t *testing.T) {
./cmd/internal/launcher/run_test.go:34:	outcome, err := Run([]string{"codex"}, Env{
./cmd/internal/launcher/run_test.go:52:	outcome, err := Run([]string{"claude"}, Env{
./cmd/internal/launcher/run_test.go:67:	outcome, err := Run([]string{"claude"}, Env{
./cmd/internal/launcher/run_test.go:82:	_, err := Run([]string{"codex", "extra"}, Env{Home: "/home/me", Cwd: "/work/pair"}, fakeSessions{}, fakeHistory{})
./cmd/internal/launcher/lifecycle.go:54:func runCleanup(env Env, rt Runtime, step launchStep, parkTimeout int, out io.Writer) {
./cmd/internal/launcher/lifecycle.go:78:		if rt.ConfirmParkNudge(step.session, parkTimeout) {
./cmd/internal/entrypoint/alias_test.go:9:	valid := []string{"context", "slug", "wrap", "scribe", "session-watch", "title",
./cmd/internal/workbenchshortcut/shortcut_test.go:66:		t.Run(tt.name, func(t *testing.T) {
./cmd/internal/workbenchshortcut/shortcut_test.go:202:		t.Run(tt.name, func(t *testing.T) {
./cmd/internal/workbenchshortcut/shortcut_test.go:238:		t.Run(tt.name, func(t *testing.T) {
./cmd/internal/workbenchshortcut/shortcut_test.go:263:		t.Run(tt.name, func(t *testing.T) {
./cmd/internal/titlepoller/run_test.go:168:	code := Run(fixtureOpts(), rt)
./cmd/internal/titlepoller/run_test.go:180:func TestRunReclaimsStalePidfileThenGraceTimeout(t *testing.T) {
./cmd/internal/titlepoller/run_test.go:191:	code := Run(opts, rt)
./cmd/internal/titlepoller/run_test.go:217:	if code := Run(opts, rt); code != 0 {
./cmd/internal/titlepoller/run_test.go:243:	if code := Run(opts, rt); code != 0 {
./cmd/internal/titlepoller/run_test.go:267:	if code := Run(opts, rt); code != 0 {
./cmd/internal/titlepoller/run_test.go:298:	if code := Run(opts, rt); code != 0 {
./cmd/internal/titlepoller/run_test.go:365:	code := Run(opts, rt)
./cmd/internal/launcher/createlogic_test.go:28:		t.Run(tc.name, func(t *testing.T) {
./cmd/internal/changelogcmd/changelogcmd.go:33:	// changelogTimeout — `claude -p` has a ~28s baseline (CLI startup + model), so
./cmd/internal/changelogcmd/changelogcmd.go:36:	changelogTimeout = 90 * time.Second
./cmd/internal/changelogcmd/changelogcmd.go:42:func Run(args []string, stderr io.Writer) int {
./cmd/internal/changelogcmd/changelogcmd.go:199:	out, err := model.Run(model.Request{
./cmd/internal/changelogcmd/changelogcmd.go:201:		MaxOutputTokens: maxTokens, Verbosity: "medium", Timeout: changelogTimeout,
./cmd/internal/titlepoller/runcli.go:39:	return Run(opts, NewOSRuntime())
./cmd/internal/titlepoller/runtime.go:48:		"rename-pane", "--pane-id", paneID, title).Run()
./cmd/internal/titlepoller/runtime.go:57:	return exec.Command("cmux", "rename-workspace", title).Run()
./cmd/internal/titlepoller/runtime.go:93:	contextcmd.Run([]string{tag, agent}, contextcmd.EnvFromOS(), &buf)
./cmd/internal/procutil/procutil_test.go:60:	return exec.Command("ps", "-p", strconv.Itoa(os.Getpid()), "-o", "command=").Run() == nil
./cmd/internal/titlepoller/run.go:63:func Run(opts Options, rt Runtime) int {
./cmd/internal/procutil/procutil.go:19:	return exec.Command("kill", "-0", pid).Run() == nil
./cmd/internal/wrapcmd/run_test.go:26:		t.Run(c.name, func(t *testing.T) {
./cmd/internal/wrapcmd/run_test.go:28:			code := Run(c.args, strings.NewReader(""), &stdout, &stderr)
./cmd/internal/wrapcmd/run_test.go:60:		code := Run([]string{"sh", "-c", "exit " + strconv.Itoa(want)}, tty, &stdout, io.Discard)
./cmd/internal/wrapcmd/translate_test.go:164:	t.Run("codex keymap", func(t *testing.T) {
./cmd/internal/wrapcmd/translate_test.go:182:		t.Run(tc.name, func(t *testing.T) {
./cmd/internal/wrapcmd/translate_test.go:211:		t.Run(tt.name, func(t *testing.T) {
./cmd/internal/wrapcmd/translate_test.go:305:		t.Run(tt.name, func(t *testing.T) {
./cmd/internal/wrapcmd/extract_fg_test.go:112:		t.Run(c.name, func(t *testing.T) {
./cmd/internal/wrapcmd/extract_fg_test.go:136:		t.Run(c.in, func(t *testing.T) {
./cmd/internal/launcher/createflow_test.go:956:func TestRunLaunchExplicitArgsDoNotPersistRepoDefaultOnReadinessTimeout(t *testing.T) {
./cmd/internal/wrapcmd/agent_restart_test.go:58:		done <- Run([]string{"/bin/sh", "-c", "sleep 30"}, bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{})
./cmd/internal/wrapcmd/translate_stdin_test.go:107:// TestTranslateStdin_HeldBackEscFlushesAfterTimeout is the core
./cmd/internal/wrapcmd/translate_stdin_test.go:114:func TestTranslateStdin_HeldBackEscFlushesAfterTimeout(t *testing.T) {
./cmd/internal/wrapcmd/osc_test.go:43:		t.Run(c.name, func(t *testing.T) {
./cmd/internal/wrapcmd/osc_test.go:110:		t.Run(c.name, func(t *testing.T) {
./cmd/internal/wrapcmd/overlay_test.go:72:		t.Run(c.name, func(t *testing.T) {
./cmd/internal/wrapcmd/wrap.go:29:// Packaging: the logic lives here in cmd/internal/wrapcmd behind Run(); it is
./cmd/internal/wrapcmd/wrap.go:217:	// (stderr is not a field: Run() owns the only stderr write — the fatal
./cmd/internal/wrapcmd/wrap.go:623:	go func() { _ = slugSpawnCmd(p.agentBasename).Run() }()
./cmd/internal/wrapcmd/wrap.go:1581:			"--", "pair", "scrollback", "open").Run()
./cmd/internal/wrapcmd/wrap.go:1606:	return exec.Command("zellij", cmdArgs...).Run()
./cmd/internal/wrapcmd/wrap.go:2118:		watcherArgv = []string{wrapperExecutable, "session-watch", agent, tag, cwd, "--"}
./cmd/internal/wrapcmd/wrap.go:2189:func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
./cmd/internal/wrapcmd/wrap.go:2345:	// Drop the agent's PID so pair-session-watch can bind discovery to
04aab5c (HEAD -> 000143-keep-agent-session-discovery-alive-after-startup-timeout) sessionwatch: #143: keep delayed agent discovery alive

codex
The lifecycle change is concentrated in the shared watcher loop, and the fake runtime advances time and state across calls, so the new delayed-discovery tests exercise real scheduling logic rather than merely asserting mocks. I’m now checking the complete agent-spec inventory, launch paths, and documentation claims for contradictions.
exec
/bin/zsh -lc "sed -n '1,260p' cmd/internal/sessionwatch/sessionwatch.go; sed -n '1,230p' cmd/internal/sessionwatch/sessionwatch_test.go; sed -n '1,130p' cmd/internal/sessionwatch/runcli_test.go; sed -n '40,105p' atlas/how-to-bring-up-a-new-harness-cli.md; sed -n '115,145p' atlas/go-migration-inventory.md; rg -n 'claude|SpawnSessionWatcher|session watcher|session-watch' cmd/internal/launcher cmd/internal/wrapcmd | head -n 240; git diff --check d08c0bec1669719090fdb5bb0f793e9db4272a12...HEAD" in /private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout
 succeeded in 0ms:
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
		if args[i] == "--resume" {
			i += 2
			continue
		}
		stripped = append(stripped, args[i])
		i++
	}
	return stripped
}

// ConfigJSON renders the restart config with structured JSON encoding.
func ConfigJSON(payload ConfigPayload) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
package sessionwatch

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAgentSpecExtractsCodexSessionID(t *testing.T) {
	home := "/tmp/home"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	path := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"

	spec, ok := SpecForAgent("codex", home)
	if !ok {
		t.Fatalf("codex spec not found")
	}
	got := spec.Match(path)
	if !got.Matched || got.NearMiss || got.ID != sid || got.Path != path {
		t.Fatalf("codex match = %+v, want id %q", got, sid)
	}
}

func TestAgentSpecExtractsAgySessionID(t *testing.T) {
	home := "/tmp/home"
	sid := "123e4567-e89b-12d3-a456-426614174000"
	path := home + "/.gemini/antigravity-cli/conversations/" + sid + ".db"

	spec, ok := SpecForAgent("agy", home)
	if !ok {
		t.Fatalf("agy spec not found")
	}
	got := spec.Match(path)
	if !got.Matched || got.NearMiss || got.ID != sid || got.Path != path {
		t.Fatalf("agy match = %+v, want id %q", got, sid)
	}
}

func TestAgentSpecReportsNearMissForPatternWithBadID(t *testing.T) {
	home := "/tmp/home"
	path := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-not-a-uuid.jsonl"

	spec, ok := SpecForAgent("codex", home)
	if !ok {
		t.Fatalf("codex spec not found")
	}
	got := spec.Match(path)
	if !got.Matched || !got.NearMiss || got.ID != "" || got.Path != path {
		t.Fatalf("codex near miss = %+v", got)
	}
}

func TestAgentSpecRejectsUnsupportedAgent(t *testing.T) {
	if _, ok := SpecForAgent("claude", "/tmp/home"); ok {
		t.Fatalf("claude should not use async session watch")
	}
}

func TestMuseMatchExtractsRootSessionID(t *testing.T) {
	home := "/tmp/home"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	path := home + "/.local/share/muse/sessions/2026/08/14/" + sid + "/session.jsonl"
	spec, ok := SpecForAgent("muse", home)
	if !ok {
		t.Fatalf("muse spec not found")
	}
	got := spec.Match(path)
	if !got.Matched || got.NearMiss || got.ID != sid || got.Path != path {
		t.Fatalf("muse root match = %+v, want id %q", got, sid)
	}
}

func TestMuseMatchIgnoresSubagentSession(t *testing.T) {
	home := "/tmp/home"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	subSid := "123e4567-e89b-12d3-a456-426614174000"
	spec, ok := SpecForAgent("muse", home)
	if !ok {
		t.Fatalf("muse spec not found")
	}
	// Root must match
	rootPath := home + "/.local/share/muse/sessions/2026/08/14/" + sid + "/session.jsonl"
	if got := spec.Match(rootPath); !got.Matched || got.ID != sid {
		t.Fatalf("root muse match = %+v, want id %q", got, sid)
	}
	// Subagent session must NOT match
	subPath := home + "/.local/share/muse/sessions/2026/08/14/" + sid + "/subagent/" + subSid + "/session.jsonl"
	if got := spec.Match(subPath); got.Matched {
		t.Fatalf("subagent path should not match, got %+v", got)
	}
	// Bad uuid under subagent also ignored
	badSub := home + "/.local/share/muse/sessions/2026/08/14/" + sid + "/subagent/not-a-uuid/session.jsonl"
	if got := spec.Match(badSub); got.Matched {
		t.Fatalf("bad subagent path should not match, got %+v", got)
	}
}

func TestMuseMatchReportsNearMissForBadID(t *testing.T) {
	home := "/tmp/home"
	spec, _ := SpecForAgent("muse", home)
	path := home + "/.local/share/muse/sessions/2026/08/14/not-a-uuid/session.jsonl"
	got := spec.Match(path)
	if !got.Matched || !got.NearMiss || got.ID != "" {
		t.Fatalf("muse near-miss = %+v", got)
	}
}

func TestStripResumeArgsRemovesCanonicalResumeBindings(t *testing.T) {
	tests := []struct {
		name  string
		agent string
		args  []string
		want  []string
	}{
		{
			name:  "codex leading resume",
			agent: "codex",
			args:  []string{"resume", "abc", "--no-alt-screen"},
			want:  []string{"--no-alt-screen"},
		},
		{
			name:  "muse leading resume",
			agent: "muse",
			args:  []string{"resume", "abc", "--model", "x"},
			want:  []string{"--model", "x"},
		},
		{
			name:  "flag resume",
			agent: "agy",
			args:  []string{"--model", "x", "--resume", "abc", "--flag"},
			want:  []string{"--model", "x", "--flag"},
		},
		{
			name:  "unrelated args keep order",
			agent: "codex",
			args:  []string{"--foo", "bar", "resume"},
			want:  []string{"--foo", "bar", "resume"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripResumeArgs(tt.agent, tt.args)
			if strings.Join(got, "\x00") != strings.Join(tt.want, "\x00") {
				t.Fatalf("StripResumeArgs(%q, %#v) = %#v, want %#v", tt.agent, tt.args, got, tt.want)
			}
		})
	}
}

func TestConfigJSONUsesStructuredEncoding(t *testing.T) {
	got, err := ConfigJSON(ConfigPayload{
		Agent:     "codex",
		Args:      []string{`say "hi"`, "--flag"},
		SessionID: "019eff64-6ceb-7e72-9d41-a735a97029ac",
	})
	if err != nil {
		t.Fatalf("ConfigJSON error: %v", err)
	}
	var decoded ConfigPayload
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("ConfigJSON produced invalid JSON %q: %v", got, err)
	}
	if decoded.Agent != "codex" || decoded.SessionID == "" || len(decoded.Args) != 2 || decoded.Args[0] != `say "hi"` {
		t.Fatalf("decoded payload = %+v", decoded)
	}
}
package sessionwatch

import (
	"os"
	"reflect"
	"testing"
	"time"
)

func TestBuildOptionsFromArgsAndEnv(t *testing.T) {
	env := map[string]string{
		"HOME":                                "/home/me",
		"PAIR_DATA_DIR":                       "/tmp/pair-data",
		"PAIR_SESSION_WATCH_PID_WAIT_SECONDS": "3",
	}
	opts, ok := buildOptions([]string{"codex", "tag", "/repo", "resume", "old", "--flag"}, func(k string) string {
		return env[k]
	})
	if !ok {
		t.Fatalf("buildOptions returned !ok")
	}
	if opts.Agent != "codex" || opts.Tag != "tag" || opts.Cwd != "/repo" {
		t.Fatalf("opts identity = %+v", opts)
	}
	if opts.Home != "/home/me" || opts.DataDir != "/tmp/pair-data" {
		t.Fatalf("opts paths = %+v", opts)
	}
	if opts.PIDWait != 3*time.Second || opts.Timeout != 60*time.Second || opts.Poll != 100*time.Millisecond {
		t.Fatalf("opts durations = %+v", opts)
	}
	if !reflect.DeepEqual(opts.Args, []string{"resume", "old", "--flag"}) {
		t.Fatalf("opts args = %#v", opts.Args)
	}
}

func TestBuildOptionsParsesRepoIdentityBeforeAgentArgs(t *testing.T) {
	opts, ok := buildOptions([]string{"codex", "tag", "/repo/sub", "--repo-root", "/repo", "--repo-name", "pair", "--", "resume", "old", "--repo-root", "agent-value"}, func(k string) string {
		if k == "HOME" {
			return "/home/me"
		}
		return ""
	})
	if !ok {
		t.Fatalf("buildOptions returned !ok")
	}
	if opts.Cwd != "/repo/sub" || opts.RepoRoot != "/repo" || opts.RepoName != "pair" {
		t.Fatalf("opts identity = %+v", opts)
	}
	if !reflect.DeepEqual(opts.Args, []string{"resume", "old", "--repo-root", "agent-value"}) {
		t.Fatalf("opts args = %#v", opts.Args)
	}
}

func TestBuildOptionsRejectsMissingRequiredArgs(t *testing.T) {
	if _, ok := buildOptions([]string{"codex", "tag"}, func(string) string { return "" }); ok {
		t.Fatalf("buildOptions should reject missing cwd")
	}
}

func TestEnsurePairTagFallback(t *testing.T) {
	t.Setenv("PAIR_TAG", "")
	cleanup := ensurePairTag("from-positional")
	defer cleanup()
	if got := os.Getenv("PAIR_TAG"); got != "from-positional" {
		t.Fatalf("PAIR_TAG = %q, want fallback tag", got)
	}
	cleanup()
	if got := os.Getenv("PAIR_TAG"); got != "" {
		t.Fatalf("PAIR_TAG after cleanup = %q, want empty", got)
	}
}
- **File:** [cmd/internal/wrapcmd/wrap.go](file:///Users/xianxu/workspace/pair/cmd/internal/wrapcmd/wrap.go)
- Register the detector in `overlayDetectorByAgent`:
  ```go
  var overlayDetectorByAgent = map[string]overlayDetector{
      "claude": detectClaudeOverlayOpen,
      "codex":  detectCodexOverlayOpen,
      "agy":    detectAgyOverlayOpen,
  }
  ```
- Implement the detector. Detectors can scan the rolling output stream for custom OSC escape sequences (e.g. Claude's permission OSC `OSC 777;notify;...`, or Codex's `OSC 9;Plan mode prompt:...`) or fallback to visible text substring matches (e.g., watching for `"Press enter to confirm"`).
- **For `codex`:** Codex plain Enter rewrites only when `codexComposerTracker` positively detects the active composer: visible cursor on or next to rows painted with the `48;2;57;57;57` background. The detector deliberately avoids Pair outer-screen bottom geometry because Codex can paint an editable composer on a smaller logical screen. Codex also uses OSC 9 plan/question bodies and visible-text picker footers; keep `codexPickerMarkers` current as override/fallback signals for menus, including variants like `"Press enter to confirm or esc to go back"` and `"Press enter to confirm or esc to cancel"`.
- **For `muse`:** Muse plain Enter rewrites only when `museComposerTracker` positively detects the active composer: visible cursor on or next to a row where the prompt glyph `›` (`e2 9f a9`, FG `38;2;90;160;255`) was painted (captured from `scrollback-fix-tty-muse.raw` at `30;1H` empty and `9;1H` filled). The logic is prompt-anchored (≥1 prompt row within `cursor±1` + visible cursor), not BG — do not reuse Codex's `48;2;57;57;57` background unless logs prove stability. Like Codex, the tracker falls back to bare CR when composer is hidden/unknown, so future Muse menus that lack a `musePickerMarkers` entry still select on plain Enter.
- **For `agy`:** Antigravity *does* render its permission picker in the PTY ("Do you want to proceed?", "Yes, and always allow", …), so `detectAgyOverlayOpen` matches those visible-text markers (no OSC) to arm `pickerActive` — without it, the remapped Enter can't confirm the picker and a stray newline leaks into the prompt (#000042).
- **For `muse`:** Muse renders both tool-permission pickers ("Permissions required", "Allow execution", …) **and** user selection menus (AskUserQuestion via `request_user_input` — "Select an option", "Use arrow keys", "Press Enter to select", …). Both families must be in `musePickerMarkers`; a missing selection marker reproduces as "Enter inserts newline, Alt+Enter required to select".

**Telemetry Signal** (aspect `2`, see §3): `overlay-detect` — `fired` when a registered marker arms `pickerActive` (the detail carries the matched marker); **`near-miss`** when the output looks like a confirm/permission prompt (`promptShape` heuristic in `checkOverlayOpen`) but *no* registered marker matched. A `near-miss` is the drift fingerprint: the harness renamed its picker wording, the detector went silent, and the next plain Enter will leak a newline (#000042). The `detail` field carries the unrecognized line verbatim — that's the new string to add to `codexPickerMarkers`/`agyPickerMarkers`/`musePickerMarkers` (or the OSC body for claude).

---

### Aspect 3: Session ID Watcher & Recovery
`pair` features a robust restart-in-place (`Alt+n`) and session reattach (`pair resume <tag>`) mechanism. To make this work, the launcher needs to discover the agent's unique conversation/session ID as soon as it is spawned.

**Discovery & Watcher:**
- **Files:** `cmd/pair-session-watch` and `cmd/internal/sessionwatch` (the launcher spawns the Go binary directly since #94 M2 — the `.sh` shim was retired).
- Since TUI agents do not always expose session IDs on stdout, `pair-session-watch` runs in the background. It finds the agent process PID from `$PAIR_DATA_DIR/agent-pid-<tag>` (written by `pair-wrap`), walks its descendants, and inspects files held open by the processes via `lsof -p <pid>`.
- Configure the agent's session file criteria in `cmd/internal/sessionwatch.SpecForAgent`, then teach `AgentSpec.Match` how to recognize that agent's file shape and return a `SessionID`.
- For example, agy watches `~/.gemini/antigravity-cli/conversations` and extracts the UUID from `<uuid>.db`; codex watches `~/.codex/sessions` and extracts the trailing UUID from `rollout-*.jsonl`; muse watches `~/.local/share/muse/sessions` and extracts the UUID from the parent dir of `session.jsonl` (`YYYY/MM/DD/<uuid>/session.jsonl`) — excluding `…/<uuid>/subagent/<sub-uuid>/session.jsonl` (only the root session is resumable via `muse resume <id>`).
- When captured, the watcher writes `{ "agent": "<agent>", "args": [...], "session_id": "<uuid>" }` into `config-<tag>-<agent>.json`.

**Recovery Flags:**
- **File:** [bin/pair-shell](../bin/pair-shell)
- Integrate the agent-specific resume argument in `bin/pair-shell`:
  ```bash
  case "$r_agent" in
      claude)        resume_extra="--resume $r_sid" ;;
      codex)         resume_extra="resume $r_sid" ;;
      agy)           resume_extra="--conversation $r_sid" ;;
  esac
  ```
- Support checking for active/resumable native session files in `agent_session_exists()`:
  ```bash
      agy)
          [ -f "$HOME/.gemini/antigravity-cli/conversations/$sid.db" ]
          ;;
  ```

**Telemetry Signal** (aspect `3`, see §3): `session-id` from `pair-session-watch` — `fired` when `AgentSpec.Match` resolves an id and the config is written, **`near-miss`** when a file matching the watch pattern is found but no id can be extracted (filename/format drift), `fail` when the 60s watch window elapses with no id at all (the session file never appeared where expected). The resume mapping in `bin/pair-shell` is the *consumer* of this id; it's static config with no separate signal.

---

### Aspect 4: pair-slug Generation
The `pair-slug` script summarizes what the current agent session is about to display in the Zellij list.

**Implementation:**
- **Transcript Parsing:** Register a parser in [cmd/internal/slugcmd/slug.go](file:///Users/xianxu/workspace/pair/cmd/internal/slugcmd/slug.go) under `parseTranscript()`. For JSONL transcripts like `agy`, extract the `content` where `type: "USER_INPUT"`; for `muse`, extract `payload.event.prompt` where `payload_type=="runtime.session"` and `payload.kind=="run"` / `event.kind=="started"`.
- **Model Sandbox Execution:** Ensure that invoking the agent in summarize mode (`agy -p "<prompt>"` / `muse exec "<prompt>"`) runs inside a clean sandbox (e.g. setting `cmd.Dir = os.TempDir()` in [cmd/internal/model/model.go](file:///Users/xianxu/workspace/pair/cmd/internal/model/model.go), the shared model runner). This prevents the agent from triggering expensive workspace exploration tools, speeding up slug generation from 20s to 1s.

**Telemetry Signal** (aspect `4`, see §3): `slug-parse` from `pair-slug` — `fired` when the transcript parses into ≥1 turn, **`near-miss`** when a transcript is read but yields 0 turns (the transcript schema changed and the parser no longer extracts anything), `fail` when no transcript resolves at all. A run of `near-miss` lines points straight at a `parseTranscript` parser that needs updating.

---

### Aspect 5: Mouse Scroll & PTY Output Filtering
Some agents emit DEC synchronized-output markers or other terminal control characters that interfere with Zellij's mouse scrollback.
- **PTY Filter:** If an agent behaves poorly with mouse scrolling, `pair-wrap` can intercept and strip specific sequences (e.g., Codex's `ESC[?2026h` synchronized-output toggles) in `stdoutChunk()` before queueing filtered visible stdout for batched delivery to Zellij. Raw scrollback capture remains immediate and unfiltered.

**Telemetry Signal** (aspect `5`, see §3): `output-filter` from `pair-wrap` (`stripCodexSyncOutput`) — `fired` once per distinct marker stripped per session (deduped; the markers repeat many times per render, so presence is the signal). If a codex update renames a sequence, its `fired` line stops appearing — an *absence* the operator reads against the expected marker set.
| `bin/pair-slug` / `cmd/pair-slug` / `cmd/internal/slugcmd` | Go binary plus shared runner | `pair-wrap` turn-end hook (now `pair slug`), tests | Env-driven, no stdin; resolves native transcript, proposes slug; exits 0 on most failures. | Requires `PAIR_TAG`, `PAIR_DATA_DIR`; reads config/transcripts/git branch; writes `slug-proposed-<tag>`; optional `PAIR_SLUG_*`, `OPENAI_API_KEY`. | implemented `pair slug` route (#92, buffered `slugcmd.Run`); `bin/pair-slug` retained as thin shim | P1 |
| `bin/pair-context` / `cmd/pair-context` / `cmd/internal/contextcmd` | Go binary plus shared runner | `cmd/internal/titlepoller` (in-process, #93 M1); development-only `pair-go context` | `pair-context <tag> <agent>` and `pair-go context <tag> <agent>` print the same humanized token count or nothing; tolerant exit 0 on failure. Exposes `TranscriptPath` for the shared transcript resolution. | Reads `PAIR_DATA_DIR`, `pane-<tag>-<agent>.json`, config, native transcripts. | implemented `pair context` route; the title poller now calls `contextcmd.Run`/`TranscriptPath` **in-process** (#93 M1, no subprocess); `bin/pair-context` retained as thin shim | P1 |
| `bin/pair-scrollback-render` / `cmd/pair-scrollback-render` / `cmd/internal/scrollbackcmd` | Go binary plus shared runner | `cmd/pair-scrollback-open` (in-process, #93 M2), `cmd/pair-changelog-open`'s detached distiller, `nvim/scrollback.lua` refresh; development-only `pair-go scrollback-render` | `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out` and `pair-go scrollback-render ...`; nonzero on render/write failure. | Reads `.raw` and `.events.jsonl`; atomically writes `.ansi` or cleaned text. | implemented `pair scrollback-render` route (#92); the Alt+/ opener now calls `scrollbackcmd.Run` **in-process** (#93 M2, no subprocess); the changelog opener's detached distiller + `nvim/scrollback.lua` still shell `pair scrollback-render`; `bin/pair-scrollback-render` retained as thin shim | P0 |
| `bin/pair-changelog` / `cmd/pair-changelog` / `cmd/internal/changelogcmd` | Go binary plus shared runner | `bin/pair-changelog-open` (now `pair changelog`) | `pair-changelog --cleaned F --log F --anchor F [--agent A] [--model M]`; exits nonzero on required read/model/write failure. | Reads cleaned scrollback/log/anchor; calls agent model through internal model runner; atomically writes log and anchor. | implemented `pair changelog` route (#92, streaming seam — live per-batch stderr spinner); `bin/pair-changelog` retained as thin shim | P1 |
| `bin/pair-continuation` / `cmd/pair-continuation` / `cmd/internal/continuationcmd` | Go binary plus shared runner | nvim compaction prompt instructions, operator/agent shell | `pair-continuation --slug S --agent A --issues CSV --body-file F [--repo-root R ...]`; writes and commits continuation; nonzero on validation/git failure. | Reads body/stdin, git repo state; writes `workshop/continuation/*.md`; runs git commit/push. | implemented `pair continuation` route (#92, streaming seam — reads body from stdin); `bin/pair-continuation` retained as thin shim; no repointed production caller yet (agent-procedure invoked) | P1 |
| `bin/pair-scribe` / `cmd/pair-scribe` / `cmd/internal/scribecmd` | Go binary plus shared runner | user shell rc outside Pair sessions | `pair-scribe -log PATH -- CMD [ARGS...]` and `pair scribe …`; long-running PTY wrapper; SIGUSR1 pauses log, SIGUSR2 resumes. | Writes typescript log; wraps child PTY; independent of `PAIR_*`. | implemented `pair scribe` route (#96, streaming seam — `scribecmd.Run`); `cmd/pair-scribe` is a thin shim so `~/.local/bin/pair-scribe` + the user's `~/.zshrc` wiring keep working; NOT in the runtime bundle (user shell tooling, not runtime) | P2 |
| `cmd/internal/adapt` | Go helper package | `cmd/internal/wrapcmd` (pair-wrap), `pair-slug`, tests | Pure-ish emitter helpers plus file open seam; no command. | Writes `$PAIR_DATA_DIR/adapt-<tag>.jsonl`; schema shared with shell/Lua. | internal package, reuse behind dispatcher | P1 |
| `cmd/internal/ctxmeter` | Go helper package | `pair-context`, tests | Pure transcript token counting and humanization. | No direct IO. | internal package, keep | P1 |
| `cmd/internal/model` | Go helper package | `pair-slug`, `pair-changelog`, tests | Model runner/response parsing. | Calls external agent/model CLIs/APIs at command layer. | internal package, keep | P1 |
| `cmd/internal/transcript` | Go helper package | `pair-slug`, `pair-context`, tests | Resolves native transcript paths and session ids. | Reads Pair config and home paths via callers. | internal package, keep | P1 |
| `cmd/pair-scrollback-open` / `cmd/internal/opener` | Go binary plus shared runner | zellij Alt+/ Run, nvim Alt+b jump | `pair-scrollback-open [--jump prev|next]`; opens read-only nvim viewer; singleton lock. | Requires `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_HOME`; renders in-process (`scrollbackcmd`), zellij IPC (list-panes/dump-screen), nvim; writes `.ansi`, `.viewport`, lock. | ported to Go (#93 M2) on the #78 template — pure viewport scorer in `opener`, IO behind the `Runtime` seam; **replaces** the shell script at the same PATH name (zellij invokes by name → no shim); `nvim/scrollback.lua` stays native | P1 |
| `nvim/scrollback.lua` | Neovim native asset | `cmd/pair-scrollback-open` | Loaded by `nvim -u ... <ansi>`; interactive read-only viewer; refreshes backing render. | Reads Pair env and `.ansi`; may call `pair-scrollback-render`; writes pending marker files. | native-asset, adjacent/embedded | P0 |
| `cmd/pair-changelog-open` / `cmd/internal/opener` | Go binary plus shared runner | zellij Alt+l Run | Opens changelog viewer and starts detached render/distill singleton. | Requires Pair env; launches a `setsid`-detached `pair scrollback-render` / `pair changelog` build (#92), nvim watcher; reads/writes `changelog-*` sidecars. | ported to Go (#93 M2) — shared `opener` package (session keying + detached distiller), IO behind the seam; **replaces** the shell script at the same PATH name (no shim); Go `SysProcAttr.Setsid` replaces setsid/perl | P1 |
| `nvim/changelog.lua` | Neovim native asset | `cmd/pair-changelog-open` | Loaded by `nvim -u ... <log>`; read-only watcher/spinner. | Reads `PAIR_CHANGELOG_*` and Pair env. | native-asset, adjacent/embedded | P1 |
| `bin/pair-title` / `cmd/pair-title` / `cmd/internal/titlepoller` | Go binary plus shared runner | launcher `SpawnTitlePoller` | `pair-title <tag> <agent>`; long-running 60s poller (frame meter + cmux heat-ramp). | Reads/writes title pid, pane json, cmux owner files; calls zellij/cmux/ps + in-process `contextcmd` for the count. | ported to Go (#93 M1) on the #78 sessionwatch template — pure decisions in `titlepoller`, IO behind the `Runtime` seam; the `.sh` re-exec shim was retired in #94 M2 (the launcher spawns `bin/pair-title` directly) | P1 |
| `bin/pair-session-watch` / `cmd/pair-session-watch` / `cmd/internal/sessionwatch` | Go stateful watcher | launcher `SpawnSessionWatcher` (create path) | `pair-session-watch <agent> <tag> <cwd> [agent-args...]`; background 60s watcher; no-op for claude. | Reads agent pidfile, lsof/ps, native session dirs; writes config JSON atomically; logs adapt events through `cmd/internal/adapt`. | Go-owned watcher with implemented `pair session-watch` route (#92, via `sessionwatch.RunCLI`); the `.sh` passthrough shim was retired in #94 M2 (the launcher spawns `bin/pair-session-watch` directly) | P1 |
| `bin/lib/adapt-log.sh` | sourced shell helper | remaining shell emitters | `adapt_log comp agent aspect signal outcome [detail]`; no-op if no `PAIR_TAG` or jq. | Appends JSONL to `$PAIR_DATA_DIR/adapt-<tag>.jsonl`. | keep until remaining shell emitters move; schema stays DRY with Go/Lua emitters | P1 |
| `nvim/adapt.lua` | Lua helper | nvim doctor/adaptation surfaces, tests | Lua adaptation flight recorder emitter. | Writes same JSONL schema as Go/shell. | native-asset; keep schema aligned | P2 |
| `doctor/README.md` / `doctor/SKILL.md` | docs/skill | operator/agent diagnostics | Documents Pair doctor flow. | Refers to `nvim/doctor.lua` and adaptation logs. | adjacent docs/skill; not Go migration target | P3 |
| `nvim/doctor.lua` | Lua helper | `:PairDoctor` in nvim | Builds agent instruction payload. | Reads `PAIR_HOME`; sends text through draft/agent flow. | native-asset | P2 |
| `bin/pair-notify` | Bash notification helper | agent hooks/manual shell inside Pair | `pair-notify [--osc 9|777] "message"`; writes OSC to outer tty; nonzero on bad args/missing tty. | Requires `PAIR_TAG`; reads `outer-tty-<tag>`. | small shell-glue; possible Go subcommand but low packaging impact | P2 |
| `pair quit` (was `bin/pair-quit.sh`; `cmd/internal/launcher/restart.go`) | in-process Go subcommand, ported from a Bash keybind helper | nvim `PairConfirmQuit` (`{ 'pair', 'quit' }`) | Touch quit marker then kill zellij session. | Uses `ZELLIJ_SESSION_NAME`, `PAIR_KILL_CMD`; writes cache marker. | **ported in #94 M1** — `runQuit` reuses the launcher's `TouchQuitMarker`/`ExecKillSession` seam; `.sh` removed from tree + runtime bundle | P2 |
| `pair restart` (was `bin/pair-restart.sh`; `cmd/internal/launcher/restart.go`) | in-process Go subcommand, ported from a Bash keybind helper | nvim restart confirmations (`{ 'pair', 'restart', ... }`) | Writes restart marker then kill zellij session; supports `--new-session` / `--rename-to <tag>`. | Uses `PAIR_TAG`, `PAIR_AGENT`, `ZELLIJ_SESSION_NAME`, cache marker files. | **ported in #94 M1** — `runRestart` reuses `WriteRestartMarker`/`TouchQuitMarker`/`ExecKillSession`/`InferAgent` (marker protocol unchanged); `.sh` removed from tree + runtime bundle | P2 |
| `bin/pair-help` | Bash helper | zellij Alt+h Run | Displays `pair keys` through `less` with escape-to-quit behavior. | Calls `pair`, `less`. | compat-shim; may become `pair help` behavior | P2 |
| `cmd/clipboard-to-pane` / `cmd/internal/clipcmd` (`bin/clipboard-to-pane`) | Go binary + shared runner | `copy-on-select` (execs `bin/clipboard-to-pane` directly), direct zellij run possible | Reads OS clipboard, stages quote at `$PAIR_DATA_DIR/quote-<tag>`, focuses the nvim pane, triggers PairPasteQuote via Ctrl-_. | Uses pbpaste/wl-paste/xclip, zellij, `PAIR_DATA_DIR`, `PAIR_TAG`; nvim-pane pick via `cmd/internal/zellijpane`. | ported to Go (#93 M4); IO behind the `clipcmd.Runtime` seam (embeds `osfs.FS`); the `.sh` re-exec shim was retired in #94 M2 (copy-on-select execs `$PAIR_HOME/bin/clipboard-to-pane` directly) | P2 |
| `cmd/copy-on-select` / `cmd/internal/clipcmd` (`bin/copy-on-select`) | Go binary + shared runner | `zellij/config.kdl` `copy_command "copy-on-select"` | Reads selected text stdin, mirrors OS clipboard, flashes source pane, hands off (execs) to clipboard-to-pane — unless the focused pane was the nvim draft (in_nvim gate on `terminal_command`, not title). | Uses pbcopy/wl-copy/xclip, zellij, `PAIR_HOME`; focused-pane pick via `cmd/internal/zellijpane`; execs the flash/clipboard Go binaries directly. | ported to Go (#93 M4); the `.sh` shim was retired in #94 M2 (zellij's `copy_command "copy-on-select"` invokes `bin/copy-on-select` by name) | P2 |
| `cmd/flash-pane` / `cmd/internal/clipcmd` (`bin/flash-pane`) | Go binary + shared runner | `copy-on-select` (execs `bin/flash-pane` directly); the copy-on-select shell test | `flash-pane [pane-id]`; best-effort pane color flash — synchronous fg set + detached (setsid) bg reset so it doesn't block the caller. | Uses zellij; reads `PAIR_FLASH_*`; focused-pane pick via `cmd/internal/zellijpane`. | ported to Go (#93 M4); the `.sh` re-exec shim was retired in #94 M2 (copy-on-select execs `$PAIR_HOME/bin/flash-pane` directly) | P3 |
| `cmd/pair-review-open` / `cmd/internal/reviewcmd` | Go binary plus shared runner | nvim review flow | Validates target and opens floating `nvim -u nvim/review.lua` (single review pane). | Requires Pair env; calls zellij/nvim; kills the prior review nvim. | ported to Go (#93 M3); IO behind the `Runtime` seam; **replaces** the shell script at the same PATH name (no shim) | P2 |
| `cmd/pair-review-readiness` / `cmd/internal/reviewcmd` | Go binary plus shared runner | `nvim/init.lua` review readiness | Gathers git facts, classifies via `nvim/review/readiness.lua`, emits JSON or performs `--prepare` git effects + marks ready. | Uses `PAIR_HOME`, git, `nvim --headless` classify. | ported to Go (#93 M3); the 4-case decision stays in `readiness.lua` (single source, invoked via `nvim --headless`); replaces the shell script (no shim) | P2 |
| `cmd/pair-review-target` / `cmd/internal/reviewcmd` | Go binary plus shared runner | review readiness/open/tests | Writes JSON target metadata under data dir, session-stamped. | Requires `PAIR_DATA_DIR`; reads config; codex fallback via `cmd/internal/codexsid` (ps/lsof); writes `review-target-<tag>.json`. | ported to Go (#93 M3); session resolution reuses `transcript`-style config read + the extracted `codexsid` walk; replaces the shell script (no shim) | P2 |
| `nvim/init.lua` | Neovim native asset | zellij draft pane | Main draft UI and Pair key handling. | Reads many Pair env vars/data files; shell-outs to zellij, pair quit/restart/open/review helpers. | native-asset; do not port, but audit shell-outs during #77/#78 | P0 |
cmd/internal/wrapcmd/keymap_registry_test.go:11:// is the kind of change that's easy to miss in review — claude /
cmd/internal/wrapcmd/keymap_registry_test.go:21:		// claude reads `\<Enter>` as newline regardless of terminal
cmd/internal/wrapcmd/keymap_registry_test.go:23:		"claude": {[]byte{'\\', '\r'}, []byte{'\r'}, ctrlU},
cmd/internal/wrapcmd/extract_fg_test.go:11:// claude end-of-turn marker → OSC9 notify), so the surface deserves
cmd/internal/wrapcmd/wrap.go:96:	"claude": "marker",
cmd/internal/wrapcmd/wrap.go:111:	"claude": regexp.MustCompile(`^\x{273B}\s*[A-Za-z]+\s+for\s+\d+[hms](?:\s+\d+[hms])*`),
cmd/internal/wrapcmd/wrap.go:119:	"claude": true,
cmd/internal/wrapcmd/wrap.go:152:	"claude": {
cmd/internal/wrapcmd/wrap.go:191:	"claude": detectClaudeOverlayOpen,
cmd/internal/wrapcmd/wrap.go:648:	// here), so it works for claude/codex/agy alike — no claude Stop hook.
cmd/internal/wrapcmd/wrap.go:685:// pickerOpenOSCBody is the OSC 777 body claude emits when a blocking
cmd/internal/wrapcmd/wrap.go:921:// Skip 0/1/2 (title sets — claude updates every second with a spinner),
cmd/internal/wrapcmd/wrap.go:988:// claude marker match, also fires emitOuter("<text>").
cmd/internal/wrapcmd/wrap.go:2076:	if agent == "claude" {
cmd/internal/wrapcmd/wrap.go:2079:			return nil, errors.New("cannot mint fresh claude session id")
cmd/internal/wrapcmd/wrap.go:2091:		if agent == "claude" {
cmd/internal/wrapcmd/wrap.go:2118:		watcherArgv = []string{wrapperExecutable, "session-watch", agent, tag, cwd, "--"}
cmd/internal/wrapcmd/wrap.go:2345:	// Drop the agent's PID so pair-session-watch can bind discovery to
cmd/internal/wrapcmd/wrap.go:2348:	// session-id capture for codex/agy; claude doesn't need it.
cmd/internal/wrapcmd/osc_test.go:10://   - leaks every claude title-update / iTerm progress tick to the
cmd/internal/wrapcmd/osc_test.go:29:		{"OSC 0 title-set swallowed (claude updates every second)", "0", "Title", false},
cmd/internal/launcher/decision_test.go:46:		decision, err := DecideLaunch(LaunchArgs{Agent: "claude"}, snap)
cmd/internal/launcher/decision_test.go:99:	decision, err := DecideLaunch(LaunchArgs{Agent: "claude", SelectedTag: "pair-old"}, SessionSnapshot{
cmd/internal/wrapcmd/stdout_filter_test.go:72:	p := &proxy{agentBasename: "claude"}
cmd/internal/wrapcmd/agent_restart_test.go:98:		[]string{"claude", "--model", "opus", "--resume", "old-session"},
cmd/internal/launcher/list_test.go:36:		{Session: "pair-a", Agent: "claude", State: SessionAttached, Clients: 1},
cmd/internal/launcher/list_test.go:43:		"pair-a", "claude", "attached (1 client)",
cmd/internal/launcher/list_test.go:59:	rt.listRows = []ListRow{{Session: "pair-x", Agent: "claude", State: SessionDetached}}
cmd/internal/wrapcmd/stdout_batch_test.go:76:		agentBasename:    "claude",
cmd/internal/wrapcmd/stdout_batch_test.go:118:		agentBasename:    "claude",
cmd/internal/wrapcmd/adapt_drift_test.go:105:	p := claudeProxy()
cmd/internal/wrapcmd/adapt_drift_test.go:106:	p.adapt = adapt.New(&buf, "pair-wrap", "claude")
cmd/internal/wrapcmd/translate_stdin_test.go:58:// claudeProxy yields a *proxy wired with the claude keymap (plain
cmd/internal/wrapcmd/translate_stdin_test.go:60:// pair-wrap's sendKeymapByAgent[claude] resolves to in production.
cmd/internal/wrapcmd/translate_stdin_test.go:61:func claudeProxy() *proxy {
cmd/internal/wrapcmd/translate_stdin_test.go:73:	p := claudeProxy()
cmd/internal/wrapcmd/translate_stdin_test.go:89:// translateChunk). Plain \r should become \\r per the claude keymap.
cmd/internal/wrapcmd/translate_stdin_test.go:93:	p := claudeProxy()
cmd/internal/wrapcmd/translate_stdin_test.go:117:	p := claudeProxy()
cmd/internal/wrapcmd/translate_stdin_test.go:151:	p := claudeProxy()
cmd/internal/wrapcmd/translate_stdin_test.go:164:	// Should land in output as a single \r (claude altCR), not as
cmd/internal/wrapcmd/translate_stdin_test.go:180:	p := claudeProxy()
cmd/internal/launcher/runcli_test.go:91:	if err := os.WriteFile(filepath.Join(globalDataDir, "agent-work"), []byte("claude\n"), 0o644); err != nil {
cmd/internal/wrapcmd/slug_spawn_test.go:33:	cmd := slugSpawnCmd("claude")
cmd/internal/wrapcmd/picker_overlay_test.go:11:// body that claude emits for AskUserQuestion / tool-permission
cmd/internal/wrapcmd/picker_overlay_test.go:15:	p := &proxy{agentBasename: "claude"}
cmd/internal/wrapcmd/picker_overlay_test.go:27:	p := &proxy{agentBasename: "claude"}
cmd/internal/wrapcmd/picker_overlay_test.go:58:// no-op, even on claude. Guards against accidentally tripping on
cmd/internal/wrapcmd/picker_overlay_test.go:70:		p := &proxy{agentBasename: "claude"}
cmd/internal/wrapcmd/picker_overlay_test.go:88:	p := claudeProxy()
cmd/internal/wrapcmd/picker_overlay_test.go:111:	p := claudeProxy()
cmd/internal/wrapcmd/picker_overlay_test.go:126:	p := claudeProxy()
cmd/internal/wrapcmd/overlay_test.go:17:			name:      "claude permission OSC opens overlay",
cmd/internal/wrapcmd/overlay_test.go:18:			agent:     "claude",
cmd/internal/wrapcmd/overlay_test.go:24:			name:     "claude waiting OSC is not overlay",
cmd/internal/wrapcmd/overlay_test.go:25:			agent:    "claude",
cmd/internal/launcher/createflow_test.go:187:func (f *fakeRuntime) SpawnSessionWatcher(agent, tag, cwd, repoRoot, repoName string, agentArgs []string) {
cmd/internal/launcher/createflow_test.go:424:	if _, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "x"}), rt); err != nil {
cmd/internal/launcher/createflow_test.go:439:// A forced-tag create with no live session: no prompt, claude mints a session id,
cmd/internal/launcher/createflow_test.go:444:	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt)
cmd/internal/launcher/createflow_test.go:457:	if rt.env["PAIR_TAG"] != "bugfix" || rt.env["PAIR_AGENT"] != "claude" || rt.env["PAIR_HOME"] != "/pair" {
cmd/internal/launcher/createflow_test.go:471:	if got := rt.env["PAIR_PANE_TITLE"]; got != "claude" {
cmd/internal/launcher/createflow_test.go:479:	cfg := rt.files["/data/config-bugfix-claude.json"]
cmd/internal/launcher/createflow_test.go:483:	if rt.files["/data/agent-bugfix"] != "claude\n" {
cmd/internal/launcher/createflow_test.go:487:	if len(ledger) != 1 || ledger[0].Agent != "claude" || ledger[0].SessionID != "MINTED-1" {
cmd/internal/launcher/createflow_test.go:488:		t.Fatalf("ledger = %+v, want claude/MINTED-1", ledger)
cmd/internal/launcher/createflow_test.go:490:	if got := rt.watchers; len(got) != 1 || !strings.HasPrefix(got[0], "claude|bugfix|/home/u/work|") {
cmd/internal/launcher/createflow_test.go:493:	if len(rt.pollers) != 1 || rt.pollers[0] != "bugfix|claude" {
cmd/internal/launcher/createflow_test.go:527:	opts := baseOpts(LaunchArgs{Agent: "claude"})
cmd/internal/launcher/createflow_test.go:604:	opts := baseOpts(LaunchArgs{Agent: "claude"})
cmd/internal/launcher/createflow_test.go:663:	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude"}), rt)
cmd/internal/launcher/createflow_test.go:680:	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude"}), rt)
cmd/internal/launcher/createflow_test.go:695:	code, err := RunLaunch(baseOpts(LaunchArgs{Agent: "claude"}), rt, &stderr)
cmd/internal/launcher/createflow_test.go:715:	code, err := RunLaunch(baseOpts(LaunchArgs{Agent: "claude"}), rt, &stderr)
cmd/internal/launcher/createflow_test.go:728:	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt)
cmd/internal/launcher/createflow_test.go:753:	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt)
cmd/internal/launcher/createflow_test.go:775:	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt)
cmd/internal/launcher/createflow_test.go:801:	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude"}), rt)
cmd/internal/launcher/createflow_test.go:820:	// Codex does not mint a claude session id.
cmd/internal/launcher/createflow_test.go:848:	rt.files["/data/config-cx-codex.json"] = `{"agent":"claude","args":["--old"],"session_id":"OLD"}`
cmd/internal/launcher/createflow_test.go:871:	if !strings.Contains(stderr.String(), `saved config agent "claude" does not match requested agent "codex"; ignoring it`) {
cmd/internal/launcher/createflow_test.go:878:	raw, err := BuildAgentDefault("claude", []string{"--model", "opus"})
cmd/internal/launcher/createflow_test.go:882:	rt.files["/data/agent-default-claude.json"] = raw
cmd/internal/launcher/createflow_test.go:883:	rt.historical = []HistoricalTag{{Tag: "old", MTime: time.Unix(1_700_000_000, 0), Agent: "claude"}}
cmd/internal/launcher/createflow_test.go:889:		Agent:         "claude",
cmd/internal/launcher/createflow_test.go:1058:	rt.files["/data/config-work-claude.json"] = `{"agent":"claude","args":["--old"],"session_id":"OLD"}`
cmd/internal/launcher/createflow_test.go:1068:	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work", AgentArgs: []string{"--fresh"}, AgentArgsExplicit: true})
cmd/internal/launcher/createflow_test.go:1073:	if !contains(rt.removed, "/data/config-work-claude.json") {
cmd/internal/launcher/createflow_test.go:1077:	if cfg := rt.files["/data/config-work-claude.json"]; !strings.Contains(cfg, "NEW-SID") {
cmd/internal/launcher/createflow_test.go:1099:	rt.files["/data/config-work-claude.json"] = `{"agent":"claude","args":["--saved"],"session_id":"SAVED"}`
cmd/internal/launcher/createflow_test.go:1102:	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work", AgentArgs: []string{"--resume", "EXPLICIT"}, AgentArgsExplicit: true})
cmd/internal/launcher/createflow_test.go:1111:	cfg := rt.files["/data/config-work-claude.json"]
cmd/internal/launcher/createflow_test.go:1126:	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "x"}), rt)
cmd/internal/launcher/createflow_test.go:1142:	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude"}), rt)
cmd/internal/launcher/createflow_test.go:1157:	rt.commandMissing["claude"] = true
cmd/internal/launcher/createflow_test.go:1158:	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "x"}), rt)
cmd/internal/launcher/createflow_test.go:1213:// With nothing on disk to infer from, the agent defaults to claude.
cmd/internal/launcher/createflow_test.go:1221:	if rt.env["PAIR_AGENT"] != "claude" {
cmd/internal/launcher/createflow_test.go:1222:		t.Fatalf("default agent = %q, want claude", rt.env["PAIR_AGENT"])
cmd/internal/launcher/createflow_test.go:1231:	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "waytoolongtag"}), rt)
cmd/internal/launcher/createflow_test.go:1246:	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt)
cmd/internal/launcher/createflow_test.go:1271:// route through the picker and resume an existing claude tag. Explicit
cmd/internal/launcher/createflow_test.go:1275:	// A historical claude tag (base tag for cwd /home/u/work is "work").
cmd/internal/launcher/createflow_test.go:1277:	rt.inferAgent["work"] = "claude"
cmd/internal/launcher/createflow_test.go:1315:	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude"}), rt)
cmd/internal/wrapcmd/codex_return_test.go:62:	p := claudeProxy()
cmd/internal/wrapcmd/codex_return_test.go:66:		t.Fatalf("got %q, want existing claude newline remap", got)
cmd/internal/launcher/config_test.go:6:	if got := CanonicalConfigPath("/dd", "t", "claude"); got != "/dd/config-t-claude.json" {
cmd/internal/launcher/config_test.go:24:	if ShouldMigrateLegacyCodex(false, "claude", true, "codex") {
cmd/internal/launcher/config_test.go:29:	if ShouldMigrateLegacyCodex(false, "codex", true, "claude") {
cmd/internal/launcher/config_test.go:45:	if got := ClaudeTranscriptPath("/home", "/Users/x/repo", "sid1"); got != "/home/.claude/projects/-Users-x-repo/sid1.jsonl" {
cmd/internal/launcher/config_test.go:46:		t.Errorf("claude transcript = %q", got)
cmd/internal/launcher/createlogic.go:17:// session-watcher both write ({agent, args, session_id}).
cmd/internal/launcher/createlogic.go:51:// pins, or "" if none. Per-agent surface (shell create branch 2053-2075): claude
cmd/internal/launcher/createlogic.go:70:	case "claude", "agy":
cmd/internal/launcher/ledger_test.go:11:		Agent:      "claude",
cmd/internal/launcher/ledger_test.go:49:		{Agent: "claude", SessionID: "old", LastActive: time.Unix(10, 0).UTC()},
cmd/internal/launcher/ledger_test.go:51:		{Agent: "claude", SessionID: "new", LastActive: time.Unix(20, 0).UTC()},
cmd/internal/launcher/ledger_test.go:54:	got, ok := LatestLedgerEntryForAgent(entries, "claude")
cmd/internal/launcher/ledger_test.go:56:		t.Fatalf("latest claude = %#v ok=%v, want new", got, ok)
cmd/internal/launcher/ledger_test.go:65:		{Agent: "claude", SessionID: "c1", LastActive: time.Unix(10, 0).UTC()},
cmd/internal/launcher/ledger_test.go:67:		{Agent: "claude", SessionID: "c2", LastActive: time.Unix(30, 0).UTC()},
cmd/internal/launcher/ledger_test.go:83:		t.Fatalf("CompactLedger ids = %s, should drop old claude", joined)
cmd/internal/launcher/help.go:11:  pair                          claude in the default session
cmd/internal/launcher/agentargs_test.go:69:		{"claude", "s1", []string{"--resume", "s1"}},
cmd/internal/launcher/agentargs_test.go:73:		{"claude", "", nil},
cmd/internal/launcher/agentargs_test.go:93:// Codex's `resume` subcommand must lead (args[0]); claude's --resume can trail.
cmd/internal/launcher/agentargs_test.go:101:	if got := composeResumeArgs("claude", []string{"--search"}, "sid"); !reflect.DeepEqual(got, []string{"--search", "--resume", "sid"}) {
cmd/internal/launcher/agentargs_test.go:102:		t.Errorf("claude resume trails: %v", got)
cmd/internal/launcher/agentargs_test.go:104:	if got := composeResumeArgs("claude", []string{"--search"}, ""); !reflect.DeepEqual(got, []string{"--search"}) {
cmd/internal/launcher/agentargs_test.go:140:// Named case for the claude --session-id mint/skip decision (judge INFO #3).
cmd/internal/launcher/agentargs_test.go:142:	if !shouldMintClaudeSessionID("claude", "", nil) {
cmd/internal/launcher/agentargs_test.go:143:		t.Error("fresh claude with no resume/flags → mint")
cmd/internal/launcher/agentargs_test.go:148:	if shouldMintClaudeSessionID("claude", "resumed-sid", nil) {
cmd/internal/launcher/agentargs_test.go:151:	if shouldMintClaudeSessionID("claude", "", []string{"--session-id", "u"}) {
cmd/internal/launcher/agentargs_test.go:154:	if shouldMintClaudeSessionID("claude", "", []string{"--fork-session"}) {
cmd/internal/launcher/agentargs_test.go:155:		t.Error("--fork-session → claude allocates internally, skip")
cmd/internal/launcher/agentargs_test.go:159:// Every agent's resume binding is stripped before persisting — the claude subset
cmd/internal/launcher/agentargs_test.go:163:	// claude subset.
cmd/internal/launcher/agentargs_test.go:165:		t.Errorf("claude: got %v", got)
cmd/internal/launcher/osruntime.go:310:func (r OSRuntime) SpawnSessionWatcher(agent, tag, cwd, repoRoot, repoName string, agentArgs []string) {
cmd/internal/launcher/osruntime.go:319:// sidecar spawns (#104 M2 folded pair-title/pair-session-watch into `pair title`
cmd/internal/launcher/osruntime.go:320:// / `pair session-watch`). The launcher is always `pair`, so os.Executable() is
cmd/internal/launcher/osruntime.go:341:	return append([]string{exe, "session-watch", agent, tag, cwd, "--repo-root", repoRoot, "--repo-name", repoName, "--"}, agentArgs...)
cmd/internal/launcher/osruntime.go:597:	case "claude":
cmd/internal/launcher/rename_test.go:30:	i = find(old, "/d/config-brain-claude.json")
cmd/internal/launcher/rename_test.go:31:	if i < 0 || nw[i] != "/d/config-mind-claude.json" {
cmd/internal/launcher/rename_test.go:78:		"/d/config-old-claude.json": true,
cmd/internal/launcher/rename_test.go:125:	rt.files["/data/config-old-claude.json"] = "cfg"
cmd/internal/launcher/rename_test.go:144:	if _, ok := rt.files["/data/config-new-claude.json"]; !ok {
cmd/internal/launcher/rename_test.go:294:	// draft-old.md moves first (enumeration order), config-old-claude.json second.
cmd/internal/launcher/rename_test.go:295:	rt.renameFailAt = "/data/config-old-claude.json"
cmd/internal/launcher/agentargs.go:142:// resumeToken is the per-agent surface for resuming a session id: claude uses
cmd/internal/launcher/agentargs.go:151:	case "claude":
cmd/internal/launcher/agentargs.go:165:// pair + pair-session-watch detection assume that position), so its token goes
cmd/internal/launcher/agentargs.go:166:// first; claude's `--resume` flag works anywhere, so saved args keep their leading
cmd/internal/launcher/agentargs.go:192:// deterministic claude session id (via --session-id) instead of leaving it to
cmd/internal/launcher/agentargs.go:194:// their own --session-id, or when --fork-session lets claude allocate internally.
cmd/internal/launcher/agentargs.go:195:// Only claude supports the flag; codex/agy always fall back to the watcher.
cmd/internal/launcher/agentargs.go:197:	return agent == "claude" && explicitResume == "" &&
cmd/internal/launcher/agentargs.go:205:// (claude --resume / --session-id, agy --conversation incl. the inline form, codex
cmd/internal/launcher/continue_test.go:38:	body := "---\nagent: claude\nissues: [#99, #93]\n---\n## NEXT ACTION\ngo\n"
cmd/internal/launcher/continue_test.go:39:	if got := frontmatterField(body, "agent"); got != "claude" {
cmd/internal/launcher/agent_defaults_test.go:34:		{"wrong agent", "codex", `{"agent":"claude","args":[]}`},
cmd/internal/launcher/scoped_paths_test.go:62:		"Config":             scopeDir + "/config-work-claude.json",
cmd/internal/launcher/scoped_paths_test.go:64:		"Pane":               scopeDir + "/pane-work-claude.json",
cmd/internal/launcher/scoped_paths_test.go:65:		"ScrollbackRaw":      scopeDir + "/scrollback-work-claude.raw",
cmd/internal/launcher/scoped_paths_test.go:66:		"ScrollbackANSI":     scopeDir + "/scrollback-work-claude.ansi",
cmd/internal/launcher/scoped_paths_test.go:67:		"ScrollbackEvents":   scopeDir + "/scrollback-work-claude.events.jsonl",
cmd/internal/launcher/scoped_paths_test.go:68:		"ScrollbackViewport": scopeDir + "/scrollback-work-claude.viewport",
cmd/internal/launcher/scoped_paths_test.go:69:		"Changelog":          scopeDir + "/changelog-work-claude.md",
cmd/internal/launcher/scoped_paths_test.go:70:		"AgentDraft":         scopeDir + "/draft-work-claude.md",
cmd/internal/launcher/scoped_paths_test.go:74:		"Config":             paths.Config("claude"),
cmd/internal/launcher/scoped_paths_test.go:76:		"Pane":               paths.Pane("claude"),
cmd/internal/launcher/scoped_paths_test.go:77:		"ScrollbackRaw":      paths.ScrollbackRaw("claude"),
cmd/internal/launcher/scoped_paths_test.go:78:		"ScrollbackANSI":     paths.ScrollbackANSI("claude"),
cmd/internal/launcher/scoped_paths_test.go:79:		"ScrollbackEvents":   paths.ScrollbackEvents("claude"),
cmd/internal/launcher/scoped_paths_test.go:80:		"ScrollbackViewport": paths.ScrollbackViewport("claude"),
cmd/internal/launcher/scoped_paths_test.go:81:		"Changelog":          paths.Changelog("claude"),
cmd/internal/launcher/scoped_paths_test.go:82:		"AgentDraft":         paths.AgentDraft("claude"),
cmd/internal/launcher/markers_test.go:30:	saved := savedConfig{Agent: "claude", Args: []string{"--flag"}, SessionID: "SID-1"}
cmd/internal/launcher/markers_test.go:35:	p := planRestart(RestartMarker{}, "work", "claude", saved)
cmd/internal/launcher/markers_test.go:39:	if p.Args.ForcedTag != "work" || p.Args.Agent != "claude" ||
cmd/internal/launcher/markers_test.go:52:	pn := planRestart(RestartMarker{NewSession: true}, "work", "claude", saved)
cmd/internal/launcher/markers_test.go:62:	pc := planRestart(RestartMarker{NewSession: true, Continue: "demo-slug"}, "work", "claude", saved)
cmd/internal/launcher/runtime.go:83:	// SpawnSessionWatcher backgrounds bin/pair-session-watch (detached) to
cmd/internal/launcher/runtime.go:85:	// claude (whose id is minted synchronously).
cmd/internal/launcher/runtime.go:86:	SpawnSessionWatcher(agent, tag, cwd, repoRoot, repoName string, agentArgs []string)
cmd/internal/launcher/runtime.go:119:	// sid is on disk (claude jsonl / codex sessions glob / agy conversation db).
cmd/internal/launcher/runtime.go:127:	// genuinely fresh tag); the caller then defaults to claude.
cmd/internal/launcher/rename.go:19:var renameAgents = []string{"claude", "codex", "agy", "muse"}
cmd/internal/launcher/lifecycle_test.go:39:		t.Fatalf("attach must not spawn a session watcher: %v", rt.watchers)
cmd/internal/launcher/lifecycle_test.go:64:	// config-bugfix-claude.json (session_id SID) that drives the resume hint.
cmd/internal/launcher/lifecycle_test.go:65:	rt.files["/data/scrollback-bugfix-claude.raw"] = "some captured bytes"
cmd/internal/launcher/lifecycle_test.go:68:	code, err := RunLaunch(baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt, &stderr)
cmd/internal/launcher/lifecycle_test.go:85:	if !reflect.DeepEqual(rt.parked, []string{"bugfix|claude|true"}) {
cmd/internal/launcher/lifecycle_test.go:89:	if contains(rt.removed, "/data/scrollback-bugfix-claude.raw") {
cmd/internal/launcher/lifecycle_test.go:93:	// remove it (quitAgent falls back to step.agent="claude" here) so no stale
cmd/internal/launcher/lifecycle_test.go:95:	for _, want := range []string{"/data/outer-tty-bugfix", "/data/agent-bugfix", "/data/scrollback-bugfix-claude.ansi", "/data/adapt-bugfix.jsonl", "/data/pane-bugfix-claude.json"} {
cmd/internal/launcher/lifecycle_test.go:121:	rt.files["/data/scrollback-bugfix-claude.raw"] = "bytes"
cmd/internal/launcher/lifecycle_test.go:122:	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt)
cmd/internal/launcher/lifecycle_test.go:141:	rt.files["/data/scrollback-work-claude.raw"] = "bytes"
cmd/internal/launcher/lifecycle_test.go:143:	rt.restartMarkers["📁work"] = RestartMarker{Tag: "work", Agent: "claude"}
cmd/internal/launcher/lifecycle_test.go:144:	if _, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"}), rt); err != nil {
cmd/internal/launcher/lifecycle_test.go:158:	rt.restartMarkers["📁work"] = RestartMarker{Tag: "work", Agent: "claude"}
cmd/internal/launcher/lifecycle_test.go:159:	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"}), rt)
cmd/internal/launcher/lifecycle_test.go:201:	rt.restartMarkers["📁work"] = RestartMarker{Tag: "work", Agent: "claude", NewSession: true}
cmd/internal/launcher/lifecycle_test.go:202:	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"}), rt)
cmd/internal/launcher/lifecycle_test.go:209:	if !contains(rt.removed, "/data/config-work-claude.json") {
cmd/internal/launcher/lifecycle_test.go:227:	rt.restartMarkers["📁work"] = RestartMarker{Tag: "work", Agent: "claude", RenameTo: "renamed"}
cmd/internal/launcher/lifecycle_test.go:231:	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"})
cmd/internal/launcher/lifecycle_test.go:256:	rt.restartMarkers["📁work"] = RestartMarker{Tag: "work", Agent: "claude", RenameTo: "renamed"}
cmd/internal/launcher/lifecycle_test.go:260:	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"})
cmd/internal/launcher/lifecycle_test.go:281:	rt.restartMarkers["📁work"] = RestartMarker{Tag: "work", Agent: "claude", NewSession: true, Continue: "demo"}
cmd/internal/launcher/lifecycle_test.go:282:	rt.continuationDocs = map[string][2]string{"demo": {"/repo/workshop/continuation/20260101-demo.md", "claude"}}
cmd/internal/launcher/lifecycle_test.go:284:	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"})
cmd/internal/launcher/lifecycle_test.go:313:	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "fresh"})
cmd/internal/launcher/lifecycle_test.go:343:		"nvim --embed /data/scrollback-work-claude.ansi":          "work",
cmd/internal/launcher/lifecycle_test.go:346:		"nvim --embed /data/scrollback-solo-claude.ansi trailing": "solo",
cmd/internal/launcher/createlogic_test.go:16:		{"claude space form", "claude", []string{"--model", "x", "--resume", "sid-1"}, "sid-1"},
cmd/internal/launcher/createlogic_test.go:17:		{"claude none", "claude", []string{"--model", "x"}, ""},
cmd/internal/launcher/createlogic_test.go:37:	got, err := buildConfigJSON("claude", []string{"--model", "opus"}, "sid-abc")
cmd/internal/launcher/createlogic_test.go:41:	want := `{"agent":"claude","args":["--model","opus"],"session_id":"sid-abc"}` + "\n"
cmd/internal/launcher/createlogic_test.go:50:	if cfg.Agent != "claude" || cfg.SessionID != "sid-abc" || !reflect.DeepEqual(cfg.Args, []string{"--model", "opus"}) {
cmd/internal/launcher/createlogic_test.go:63:	got2, _ := buildConfigJSON("claude", []string{"a<b>&c"}, "")
cmd/internal/launcher/createlogic_test.go:119:	if got := composeTagRestartArgs("saved", "claude", saved, newArgs, "sid"); !reflect.DeepEqual(got, saved) {
cmd/internal/launcher/createlogic_test.go:123:	if got := composeTagRestartArgs("new", "claude", saved, newArgs, "sid"); !reflect.DeepEqual(got, newArgs) {
cmd/internal/launcher/createlogic_test.go:126:	// saved+resume (claude) → saved args + --resume <sid> (flag trails).
cmd/internal/launcher/createlogic_test.go:127:	if got := composeTagRestartArgs("saved+resume", "claude", saved, newArgs, "sid"); !reflect.DeepEqual(got, []string{"--search", "--resume", "sid"}) {
cmd/internal/launcher/createlogic_test.go:128:		t.Fatalf("claude saved+resume = %v", got)
cmd/internal/launcher/createlogic_test.go:135:	if got := composeTagRestartArgs("new+resume", "claude", saved, []string{"--resume", "stale", "--flag"}, "sid"); !reflect.DeepEqual(got, []string{"--flag", "--resume", "sid"}) {
cmd/internal/launcher/launch_args_policy_test.go:30:		Agent: "claude",
cmd/internal/launcher/launch_args_policy_test.go:34:		Default:      AgentDefault{Agent: "claude", Args: []string{"--model", "opus"}},
cmd/internal/launcher/launch_args_policy_test.go:47:		Agent:        "claude",
cmd/internal/launcher/launch_args_policy_test.go:48:		Saved:        savedConfig{Agent: "claude", Args: []string{"--saved"}, SessionID: "SID"},
cmd/internal/launcher/launch_args_policy_test.go:50:		Default:      AgentDefault{Agent: "claude", Args: []string{"--default"}},
cmd/internal/launcher/run_test.go:52:	outcome, err := Run([]string{"claude"}, Env{
cmd/internal/launcher/run_test.go:67:	outcome, err := Run([]string{"claude"}, Env{
cmd/internal/launcher/migrate_test.go:8:		"/global/config-work-claude.json":      true,
cmd/internal/launcher/migrate_test.go:9:		"/global/scrollback-work-claude.raw":   true,

exec
/bin/zsh -lc "sed -n '295,365p' cmd/internal/launcher/osruntime.go; sed -n '2320,2370p' cmd/internal/wrapcmd/wrap.go; rg -n \"agent-pid-\" cmd bin test* .github 2>/dev/null | head -n 100; go test ./cmd/internal/sessionwatch" in /private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout
 succeeded in 661ms:
ok  	github.com/xianxu/pair/cmd/internal/sessionwatch	0.192s

exec
/bin/zsh -lc "sed -n '1,180p' tests/pair-session-watch-test.sh; git show d08c0bec1669719090fdb5bb0f793e9db4272a12:tests/pair-session-watch-test.sh | sed -n '1,180p'; rg -n \"pair-session-watch-test\" Makefile scripts tests .github README.md workshop | head -n 100; nl -ba cmd/internal/sessionwatch/run.go | sed -n '20,180p'; nl -ba cmd/internal/sessionwatch/run_test.go | sed -n '285,420p'; nl -ba atlas/how-to-bring-up-a-new-harness-cli.md | sed -n '58,92p'; nl -ba atlas/go-migration-inventory.md | sed -n '124,136p'" in /private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout
 succeeded in 0ms:
#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-session-watch-test.XXXXXX")"
trap 'rm -rf "$RT"; [ -z "${live_pid:-}" ] || kill "$live_pid" 2>/dev/null || true' EXIT

mkdir -p "$RT/bin" "$RT/data" "$RT/home/.codex/sessions/2026/06/25"

sid="019eff64-6ceb-7e72-9d41-a735a97029ac"
session_file="$RT/home/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-$sid.jsonl"
: > "$session_file"

cat > "$RT/bin/lsof" <<SH
#!/usr/bin/env bash
if [ "\$1" = "-p" ] && [ "\$2" = "__LIVE_PID__" ]; then
  printf 'p%s\nn%s\n' "__LIVE_PID__" "$session_file"
fi
SH
chmod +x "$RT/bin/lsof"

echo 999999 > "$RT/data/agent-pid-test"
touch -t 200001010000 "$RT/data/agent-pid-test"

sleep 10 &
live_pid=$!
sed "s/__LIVE_PID__/$live_pid/g" "$RT/bin/lsof" > "$RT/bin/lsof.tmp"
mv "$RT/bin/lsof.tmp" "$RT/bin/lsof"
chmod +x "$RT/bin/lsof"

PATH="$RT/bin:$PATH" \
HOME="$RT/home" \
PAIR_DATA_DIR="$RT/data" \
PAIR_TAG=test \
PAIR_SESSION_WATCH_PID_WAIT_SECONDS=3 \
"$ROOT/bin/pair" session-watch codex test "$ROOT" resume old-session 'say "hi"' --no-alt-screen &
watch_pid=$!

sleep 0.2
printf '%s\n' "$live_pid" > "$RT/data/agent-pid-test"

wait "$watch_pid"

got="$(jq -r '.session_id // empty' "$RT/data/config-test-codex.json")"
[ "$got" = "$sid" ] || {
  echo "session_id mismatch: got '$got', want '$sid'" >&2
  exit 1
}

args="$(jq -c '.args' "$RT/data/config-test-codex.json")"
[ "$args" = '["say \"hi\"","--no-alt-screen"]' ] || {
  echo "args mismatch: got '$args'" >&2
  exit 1
}

echo "pair session-watch stale pidfile test PASS"
#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-session-watch-test.XXXXXX")"
trap 'rm -rf "$RT"; [ -z "${live_pid:-}" ] || kill "$live_pid" 2>/dev/null || true' EXIT

mkdir -p "$RT/bin" "$RT/data" "$RT/home/.codex/sessions/2026/06/25"

sid="019eff64-6ceb-7e72-9d41-a735a97029ac"
session_file="$RT/home/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-$sid.jsonl"
: > "$session_file"

cat > "$RT/bin/lsof" <<SH
#!/usr/bin/env bash
if [ "\$1" = "-p" ] && [ "\$2" = "__LIVE_PID__" ]; then
  printf 'p%s\nn%s\n' "__LIVE_PID__" "$session_file"
fi
SH
chmod +x "$RT/bin/lsof"

echo 999999 > "$RT/data/agent-pid-test"
touch -t 200001010000 "$RT/data/agent-pid-test"

sleep 10 &
live_pid=$!
sed "s/__LIVE_PID__/$live_pid/g" "$RT/bin/lsof" > "$RT/bin/lsof.tmp"
mv "$RT/bin/lsof.tmp" "$RT/bin/lsof"
chmod +x "$RT/bin/lsof"

PATH="$RT/bin:$PATH" \
HOME="$RT/home" \
PAIR_DATA_DIR="$RT/data" \
PAIR_TAG=test \
PAIR_SESSION_WATCH_PID_WAIT_SECONDS=3 \
"$ROOT/bin/pair" session-watch codex test "$ROOT" resume old-session 'say "hi"' --no-alt-screen &
watch_pid=$!

sleep 0.2
printf '%s\n' "$live_pid" > "$RT/data/agent-pid-test"

wait "$watch_pid"

got="$(jq -r '.session_id // empty' "$RT/data/config-test-codex.json")"
[ "$got" = "$sid" ] || {
  echo "session_id mismatch: got '$got', want '$sid'" >&2
  exit 1
}

args="$(jq -c '.args' "$RT/data/config-test-codex.json")"
[ "$args" = '["say \"hi\"","--no-alt-screen"]' ] || {
  echo "args mismatch: got '$args'" >&2
  exit 1
}

echo "pair session-watch stale pidfile test PASS"
rg: Makefile: No such file or directory (os error 2)
tests/pair-session-watch-test.sh:5:RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-session-watch-test.XXXXXX")"
workshop/history/issues/000070-fix-codex-session-id-capture-with-stale-pidfile.md:39:- 2026-06-25: closed — Fixed stale pidfile race in Codex session watcher. Verified bash -n bin/pair bin/pair-session-watch.sh tests/pair-session-watch-test.sh; make test-session-watch; git diff --check. Full env -u PAIR_SESSION_ID -u PAIR_TAG make test reached review-apply-test and failed with empty result file after headless nvim exited 0, outside the session-watch path. --no-actual because sdlc actual reported no measurable activity for this short same-turn issue.; review verdict: FIX-THEN-SHIP
workshop/history/issues/000070-fix-codex-session-id-capture-with-stale-pidfile.md:48:- Red/green evidence: `bash tests/pair-session-watch-test.sh` failed before the
workshop/history/issues/000070-fix-codex-session-id-capture-with-stale-pidfile.md:52:  tests/pair-session-watch-test.sh` PASS; `make test-session-watch` PASS; `git
workshop/history/issues/000078-go-stateful-shell-glue.md:84:Replaced `bin/pair-session-watch.sh` with a compatibility shim that resolves `PAIR_HOME` and execs `bin/pair-session-watch`, leaving the existing `bin/pair` create-path caller unchanged. Updated `Makefile.local` so `pair-session-watch` is a built Go binary and `make test-session-watch` depends on it. Expanded `tests/pair-session-watch-test.sh` to exercise the shim, fake `lsof`, stale pidfile wait, codex resume stripping, and quote-safe JSON args.
workshop/history/plans/000078-go-stateful-shell-glue-plan.md:56:| `SessionWatchProcessTest` | `tests/pair-session-watch-test.sh` | modified | fake PATH commands and temp HOME/data dirs |
workshop/history/plans/000078-go-stateful-shell-glue-plan.md:190:- Modify: `tests/pair-session-watch-test.sh`
workshop/history/plans/000078-go-stateful-shell-glue-plan.md:203:Update `tests/pair-session-watch-test.sh` to exercise the shim invoking the Go binary with fake `ps`/`lsof` and temp HOME/data dirs. Keep the stale pidfile regression. Add a quoted arg in the saved config to prove JSON escaping is structured.
workshop/history/plans/000077-go-entrypoint-switch-close-review.md:2835:tests/pair-session-watch-test.sh:36:"$ROOT/bin/pair-session-watch.sh" codex test "$ROOT" --no-alt-screen &
workshop/history/plans/000077-go-entrypoint-switch-close-review.md:7447:+tests/pair-session-watch-test.sh:36:"$ROOT/bin/pair-session-watch.sh" codex test "$ROOT" --no-alt-screen &
workshop/history/plans/000113-review-footnote-question-boundary-close-review.md:1217:tests/pair-session-watch-test.sh
workshop/history/plans/000113-review-footnote-question-boundary-close-review.md:1668:bash tests/pair-session-watch-test.sh
workshop/history/plans/000086-fix-alt-return-draft-submit-close-review.md:1591:tests/pair-session-watch-test.sh:11:session_file="$RT/home/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-$sid.jsonl"
workshop/history/plans/000073-go-migration-inventory-plan.md:178:Expected: all pass. If there is a known pre-existing failure, record the exact failing target and rerun the focused no-change verification (`make build`, `bash tests/pair-continue-test.sh`, `bash tests/pair-session-watch-test.sh`) before close.
workshop/history/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-close-review.md:261: 	bash tests/pair-session-watch-test.sh
workshop/history/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-close-review.md:3687:tests/pair-session-watch-test.sh:33:PAIR_DATA_DIR="$RT/data" \
workshop/history/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-close-review.md:4158:    89		bash tests/pair-session-watch-test.sh
workshop/history/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-close-review.md:5965: 	bash tests/pair-session-watch-test.sh
workshop/history/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-close-review.md:9317:    89		bash tests/pair-session-watch-test.sh
workshop/history/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-close-review.md:10552:tests/pair-session-watch-test.sh:33:PAIR_DATA_DIR="$RT/data" \
workshop/history/plans/000079-go-packaging-consolidation-close-review.md:2139:    87		bash tests/pair-session-watch-test.sh
workshop/history/plans/000079-go-packaging-consolidation-close-review.md:2632:./tests/pair-session-watch-test.sh:36:"$ROOT/bin/pair-session-watch.sh" codex test "$ROOT" resume old-session 'say "hi"' --no-alt-screen &
workshop/history/plans/000075-go-launcher-prototype-close-review.md:9589:tests/pair-session-watch-test.sh:33:PAIR_DATA_DIR="$RT/data" \
workshop/history/plans/000075-go-launcher-prototype-close-review.md:9590:tests/pair-session-watch-test.sh:34:PAIR_TAG=test \
workshop/history/plans/000075-go-launcher-prototype-close-review.md:9591:tests/pair-session-watch-test.sh:35:PAIR_SESSION_WATCH_PID_WAIT_SECONDS=3 \
    20	type Options struct {
    21		Agent    string
    22		Tag      string
    23		Cwd      string
    24		RepoRoot string
    25		RepoName string
    26		Args     []string
    27		Home     string
    28		DataDir  string
    29		PIDWait  time.Duration
    30		Timeout  time.Duration
    31		Poll     time.Duration
    32		SlowPoll time.Duration
    33	}
    34	
    35	// Runtime is the IO boundary for the session watcher.
    36	type Runtime interface {
    37		Now() time.Time
    38		Sleep(time.Duration)
    39		ReadFile(path string) ([]byte, error)
    40		ModTime(path string) (time.Time, error)
    41		BirthTime(path string) (time.Time, error)
    42		ListFiles(root string) ([]string, error)
    43		Descendants(root string) ([]string, error)
    44		LsofPaths(pid string) ([]string, error)
    45		ProcessAlive(pid string) bool
    46		AtomicWrite(path string, data []byte) error
    47		Log(outcome adapt.Outcome, detail string)
    48	}
    49	
    50	type sessionLedgerEntry struct {
    51		Agent      string    `json:"agent"`
    52		Args       []string  `json:"args"`
    53		SessionID  string    `json:"session_id"`
    54		Started    time.Time `json:"started"`
    55		LastActive time.Time `json:"last_active"`
    56		RepoRoot   string    `json:"repo_root"`
    57		RepoName   string    `json:"repo_name"`
    58	}
    59	
    60	// Run discovers the async agent session id and writes config-<tag>-<agent>.json.
    61	func Run(opts Options, rt Runtime) error {
    62		spec, ok := SpecForAgent(opts.Agent, opts.Home)
    63		if !ok || opts.Tag == "" || opts.DataDir == "" {
    64			return nil
    65		}
    66		if opts.PIDWait <= 0 {
    67			opts.PIDWait = 2 * time.Second
    68		}
    69		if opts.Timeout <= 0 {
    70			opts.Timeout = 60 * time.Second
    71		}
    72		if opts.Poll <= 0 {
    73			opts.Poll = 100 * time.Millisecond
    74		}
    75		if opts.SlowPoll <= 0 {
    76			opts.SlowPoll = 60 * time.Second
    77		}
    78		repoRoot := opts.RepoRoot
    79		if repoRoot == "" {
    80			repoRoot = opts.Cwd
    81		}
    82		repoName := opts.RepoName
    83		if repoName == "" {
    84			repoName = filepath.Base(filepath.Clean(repoRoot))
    85		}
    86	
    87		watchStart := rt.Now()
    88		pidFile := filepath.Join(opts.DataDir, "agent-pid-"+opts.Tag)
    89		out := filepath.Join(opts.DataDir, "config-"+opts.Tag+"-"+opts.Agent+".json")
    90	
    91		pidDeadline := watchStart.Add(opts.PIDWait)
    92		for {
    93			if fresh, _ := freshPID(pidFile, watchStart, rt); fresh {
    94				break
    95			}
    96			if !rt.Now().Before(pidDeadline) {
    97				break
    98			}
    99			rt.Sleep(opts.Poll)
   100		}
   101	
   102		rootPID := ""
   103		agentStart := time.Time{}
   104		if fresh, mod := freshPID(pidFile, watchStart, rt); fresh {
   105			if data, err := rt.ReadFile(pidFile); err == nil {
   106				rootPID = strings.TrimSpace(string(data))
   107				agentStart = mod
   108			}
   109		}
   110	
   111		legacyExisting := map[string]bool{}
   112		if rootPID == "" {
   113			files, _ := rt.ListFiles(spec.WatchDir)
   114			for _, file := range files {
   115				legacyExisting[file] = true
   116			}
   117		}
   118	
   119		nmLogged := false
   120		deadline := watchStart.Add(opts.Timeout)
   121		for {
   122			if rootPID != "" && !rt.ProcessAlive(rootPID) {
   123				return nil
   124			}
   125			if rootPID == "" && !rt.Now().Before(deadline) {
   126				rt.Log(adapt.Fail, "no session id within startup deadline (agent="+opts.Agent+")")
   127				return nil
   128			}
   129	
   130			result := discover(spec, rootPID, agentStart, legacyExisting, rt)
   131			if result.ID != "" {
   132				payload, err := ConfigJSON(ConfigPayload{
   133					Agent:     opts.Agent,
   134					Args:      StripResumeArgs(opts.Agent, opts.Args),
   135					SessionID: result.ID,
   136				})
   137				if err != nil {
   138					return err
   139				}
   140				if err := appendSessionLedger(rt, filepath.Join(opts.DataDir, "ledger-"+opts.Tag+".jsonl"), sessionLedgerEntry{
   141					Agent:      opts.Agent,
   142					Args:       StripResumeArgs(opts.Agent, opts.Args),
   143					SessionID:  result.ID,
   144					Started:    watchStart,
   145					LastActive: rt.Now(),
   146					RepoRoot:   repoRoot,
   147					RepoName:   repoName,
   148				}); err != nil {
   149					return err
   150				}
   151				if err := rt.AtomicWrite(out, payload); err != nil {
   152					return err
   153				}
   154				rt.Log(adapt.Fired, "session_id="+result.ID)
   155				return nil
   156			}
   157			if result.NearMiss && !nmLogged {
   158				rt.Log(adapt.NearMiss, "matched session file but no id extracted: "+filepath.Base(result.Path))
   159				nmLogged = true
   160			}
   161	
   162			poll := opts.Poll
   163			if !rt.Now().Before(deadline) {
   164				poll = opts.SlowPoll
   165			}
   166			rt.Sleep(poll)
   167		}
   168	}
   169	
   170	func appendSessionLedger(rt Runtime, path string, entry sessionLedgerEntry) error {
   171		raw := ""
   172		if existing, err := rt.ReadFile(path); err == nil {
   173			raw = string(existing)
   174		}
   175		line, err := json.Marshal(entry)
   176		if err != nil {
   177			return err
   178		}
   179		if raw != "" && !strings.HasSuffix(raw, "\n") {
   180			raw += "\n"
   285		if err != nil {
   286			t.Fatalf("Run error: %v", err)
   287		}
   288		got := string(rt.writes[filepath.Join(data, "config-tag-codex.json")])
   289		if !strings.Contains(got, sid) {
   290			t.Fatalf("config write = %s, want valid sid after legacy near miss", got)
   291		}
   292	}
   293	
   294	func TestRunDiscoversSessionAfterStartupTimeoutForEveryAsyncAgent(t *testing.T) {
   295		tests := []struct {
   296			agent string
   297			sid   string
   298			path  func(home, sid string) string
   299		}{
   300			{
   301				agent: "codex",
   302				sid:   "019eff64-6ceb-7e72-9d41-a735a97029ac",
   303				path: func(home, sid string) string {
   304					return home + "/.codex/sessions/2026/08/18/rollout-2026-08-18T14-47-32-" + sid + ".jsonl"
   305				},
   306			},
   307			{
   308				agent: "agy",
   309				sid:   "123e4567-e89b-12d3-a456-426614174000",
   310				path: func(home, sid string) string {
   311					return home + "/.gemini/antigravity-cli/conversations/" + sid + ".db"
   312				},
   313			},
   314			{
   315				agent: "muse",
   316				sid:   "223e4567-e89b-12d3-a456-426614174000",
   317				path: func(home, sid string) string {
   318					return home + "/.local/share/muse/sessions/" + sid + "/session.jsonl"
   319				},
   320			},
   321		}
   322	
   323		for _, tt := range tests {
   324			t.Run(tt.agent, func(t *testing.T) {
   325				home := "/tmp/home"
   326				data := "/tmp/data"
   327				rt := newFakeRuntime(time.Unix(500, 0))
   328				rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("5000\n"), mod: rt.now}
   329				rt.alive["5000"] = true
   330				rt.descendants["5000"] = []string{"5000"}
   331				rt.onSleep = func(d time.Duration) {
   332					if d == time.Minute {
   333						rt.lsof["5000"] = []string{tt.path(home, tt.sid)}
   334					}
   335				}
   336	
   337				err := Run(Options{
   338					Agent:   tt.agent,
   339					Tag:     "tag",
   340					Cwd:     "/repo",
   341					Home:    home,
   342					DataDir: data,
   343					PIDWait: 100 * time.Millisecond,
   344					Timeout: 300 * time.Millisecond,
   345					Poll:    100 * time.Millisecond,
   346				}, rt)
   347				if err != nil {
   348					t.Fatalf("Run error: %v", err)
   349				}
   350	
   351				got := string(rt.writes[filepath.Join(data, "config-tag-"+tt.agent+".json")])
   352				if !strings.Contains(got, `"session_id":"`+tt.sid+`"`) {
   353					t.Fatalf("config write = %s, want delayed session %s", got, tt.sid)
   354				}
   355				if got := countDuration(rt.sleeps, time.Minute); got != 1 {
   356					t.Fatalf("slow sleeps = %d, want 1; all sleeps=%v", got, rt.sleeps)
   357				}
   358			})
   359		}
   360	}
   361	
   362	func TestRunStopsAtSlowPollWhenBoundProcessExits(t *testing.T) {
   363		data := "/tmp/data"
   364		rt := newFakeRuntime(time.Unix(600, 0))
   365		rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("6000\n"), mod: rt.now}
   366		rt.alive["6000"] = true
   367		rt.onSleep = func(d time.Duration) {
   368			if d == time.Minute {
   369				rt.alive["6000"] = false
   370			}
   371		}
   372	
   373		err := Run(Options{
   374			Agent:   "codex",
   375			Tag:     "tag",
   376			Cwd:     "/repo",
   377			Home:    "/tmp/home",
   378			DataDir: data,
   379			PIDWait: 100 * time.Millisecond,
   380			Timeout: 300 * time.Millisecond,
   381			Poll:    100 * time.Millisecond,
   382		}, rt)
   383		if err != nil {
   384			t.Fatalf("Run error: %v", err)
   385		}
   386		if got := countDuration(rt.sleeps, time.Minute); got != 1 {
   387			t.Fatalf("slow sleeps = %d, want 1; all sleeps=%v", got, rt.sleeps)
   388		}
   389		if _, ok := rt.writes[filepath.Join(data, "config-tag-codex.json")]; ok {
   390			t.Fatal("config should not be written after the bound process exits")
   391		}
   392	}
   393	
   394	func TestRunLogsFailOnTimeout(t *testing.T) {
   395		rt := newFakeRuntime(time.Unix(400, 0))
   396		err := Run(Options{
   397			Agent:   "codex",
   398			Tag:     "tag",
   399			Cwd:     "/repo",
   400			Home:    "/tmp/home",
   401			DataDir: "/tmp/data",
   402			PIDWait: 100 * time.Millisecond,
   403			Timeout: 300 * time.Millisecond,
   404			Poll:    100 * time.Millisecond,
   405		}, rt)
   406		if err != nil {
   407			t.Fatalf("Run error: %v", err)
   408		}
   409		if !rt.hasLog(adapt.Fail, "no session id") {
   410			t.Fatalf("logs = %+v, want fail", rt.logs)
   411		}
   412	}
   413	
   414	func countDuration(ds []time.Duration, want time.Duration) int {
   415		var n int
   416		for _, d := range ds {
   417			if d == want {
   418				n++
   419			}
   420		}
    58	
    59	### Aspect 3: Session ID Watcher & Recovery
    60	`pair` features a robust restart-in-place (`Alt+n`) and session reattach (`pair resume <tag>`) mechanism. To make this work, the launcher needs to discover the agent's unique conversation/session ID as soon as it is spawned.
    61	
    62	**Discovery & Watcher:**
    63	- **Files:** `cmd/pair-session-watch` and `cmd/internal/sessionwatch` (the launcher spawns the Go binary directly since #94 M2 — the `.sh` shim was retired).
    64	- Since TUI agents do not always expose session IDs on stdout, `pair-session-watch` runs in the background. It finds the agent process PID from `$PAIR_DATA_DIR/agent-pid-<tag>` (written by `pair-wrap`), walks its descendants, and inspects files held open by the processes via `lsof -p <pid>`.
    65	- Configure the agent's session file criteria in `cmd/internal/sessionwatch.SpecForAgent`, then teach `AgentSpec.Match` how to recognize that agent's file shape and return a `SessionID`.
    66	- For example, agy watches `~/.gemini/antigravity-cli/conversations` and extracts the UUID from `<uuid>.db`; codex watches `~/.codex/sessions` and extracts the trailing UUID from `rollout-*.jsonl`; muse watches `~/.local/share/muse/sessions` and extracts the UUID from the parent dir of `session.jsonl` (`YYYY/MM/DD/<uuid>/session.jsonl`) — excluding `…/<uuid>/subagent/<sub-uuid>/session.jsonl` (only the root session is resumable via `muse resume <id>`).
    67	- When captured, the watcher writes `{ "agent": "<agent>", "args": [...], "session_id": "<uuid>" }` into `config-<tag>-<agent>.json`.
    68	
    69	**Recovery Flags:**
    70	- **File:** [bin/pair-shell](../bin/pair-shell)
    71	- Integrate the agent-specific resume argument in `bin/pair-shell`:
    72	  ```bash
    73	  case "$r_agent" in
    74	      claude)        resume_extra="--resume $r_sid" ;;
    75	      codex)         resume_extra="resume $r_sid" ;;
    76	      agy)           resume_extra="--conversation $r_sid" ;;
    77	  esac
    78	  ```
    79	- Support checking for active/resumable native session files in `agent_session_exists()`:
    80	  ```bash
    81	      agy)
    82	          [ -f "$HOME/.gemini/antigravity-cli/conversations/$sid.db" ]
    83	          ;;
    84	  ```
    85	
    86	**Telemetry Signal** (aspect `3`, see §3): `session-id` from `pair-session-watch` — `fired` when `AgentSpec.Match` resolves an id and the config is written, **`near-miss`** when a file matching the watch pattern is found but no id can be extracted (filename/format drift), `fail` when the 60s watch window elapses with no id at all (the session file never appeared where expected). The resume mapping in `bin/pair-shell` is the *consumer* of this id; it's static config with no separate signal.
    87	
    88	---
    89	
    90	### Aspect 4: pair-slug Generation
    91	The `pair-slug` script summarizes what the current agent session is about to display in the Zellij list.
    92	
   124	| `cmd/internal/transcript` | Go helper package | `pair-slug`, `pair-context`, tests | Resolves native transcript paths and session ids. | Reads Pair config and home paths via callers. | internal package, keep | P1 |
   125	| `cmd/pair-scrollback-open` / `cmd/internal/opener` | Go binary plus shared runner | zellij Alt+/ Run, nvim Alt+b jump | `pair-scrollback-open [--jump prev|next]`; opens read-only nvim viewer; singleton lock. | Requires `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_HOME`; renders in-process (`scrollbackcmd`), zellij IPC (list-panes/dump-screen), nvim; writes `.ansi`, `.viewport`, lock. | ported to Go (#93 M2) on the #78 template — pure viewport scorer in `opener`, IO behind the `Runtime` seam; **replaces** the shell script at the same PATH name (zellij invokes by name → no shim); `nvim/scrollback.lua` stays native | P1 |
   126	| `nvim/scrollback.lua` | Neovim native asset | `cmd/pair-scrollback-open` | Loaded by `nvim -u ... <ansi>`; interactive read-only viewer; refreshes backing render. | Reads Pair env and `.ansi`; may call `pair-scrollback-render`; writes pending marker files. | native-asset, adjacent/embedded | P0 |
   127	| `cmd/pair-changelog-open` / `cmd/internal/opener` | Go binary plus shared runner | zellij Alt+l Run | Opens changelog viewer and starts detached render/distill singleton. | Requires Pair env; launches a `setsid`-detached `pair scrollback-render` / `pair changelog` build (#92), nvim watcher; reads/writes `changelog-*` sidecars. | ported to Go (#93 M2) — shared `opener` package (session keying + detached distiller), IO behind the seam; **replaces** the shell script at the same PATH name (no shim); Go `SysProcAttr.Setsid` replaces setsid/perl | P1 |
   128	| `nvim/changelog.lua` | Neovim native asset | `cmd/pair-changelog-open` | Loaded by `nvim -u ... <log>`; read-only watcher/spinner. | Reads `PAIR_CHANGELOG_*` and Pair env. | native-asset, adjacent/embedded | P1 |
   129	| `bin/pair-title` / `cmd/pair-title` / `cmd/internal/titlepoller` | Go binary plus shared runner | launcher `SpawnTitlePoller` | `pair-title <tag> <agent>`; long-running 60s poller (frame meter + cmux heat-ramp). | Reads/writes title pid, pane json, cmux owner files; calls zellij/cmux/ps + in-process `contextcmd` for the count. | ported to Go (#93 M1) on the #78 sessionwatch template — pure decisions in `titlepoller`, IO behind the `Runtime` seam; the `.sh` re-exec shim was retired in #94 M2 (the launcher spawns `bin/pair-title` directly) | P1 |
   130	| `bin/pair-session-watch` / `cmd/pair-session-watch` / `cmd/internal/sessionwatch` | Go stateful watcher | launcher `SpawnSessionWatcher` (create path) | `pair-session-watch <agent> <tag> <cwd> [agent-args...]`; background 60s watcher; no-op for claude. | Reads agent pidfile, lsof/ps, native session dirs; writes config JSON atomically; logs adapt events through `cmd/internal/adapt`. | Go-owned watcher with implemented `pair session-watch` route (#92, via `sessionwatch.RunCLI`); the `.sh` passthrough shim was retired in #94 M2 (the launcher spawns `bin/pair-session-watch` directly) | P1 |
   131	| `bin/lib/adapt-log.sh` | sourced shell helper | remaining shell emitters | `adapt_log comp agent aspect signal outcome [detail]`; no-op if no `PAIR_TAG` or jq. | Appends JSONL to `$PAIR_DATA_DIR/adapt-<tag>.jsonl`. | keep until remaining shell emitters move; schema stays DRY with Go/Lua emitters | P1 |
   132	| `nvim/adapt.lua` | Lua helper | nvim doctor/adaptation surfaces, tests | Lua adaptation flight recorder emitter. | Writes same JSONL schema as Go/shell. | native-asset; keep schema aligned | P2 |
   133	| `doctor/README.md` / `doctor/SKILL.md` | docs/skill | operator/agent diagnostics | Documents Pair doctor flow. | Refers to `nvim/doctor.lua` and adaptation logs. | adjacent docs/skill; not Go migration target | P3 |
   134	| `nvim/doctor.lua` | Lua helper | `:PairDoctor` in nvim | Builds agent instruction payload. | Reads `PAIR_HOME`; sends text through draft/agent flow. | native-asset | P2 |
   135	| `bin/pair-notify` | Bash notification helper | agent hooks/manual shell inside Pair | `pair-notify [--osc 9|777] "message"`; writes OSC to outer tty; nonzero on bad args/missing tty. | Requires `PAIR_TAG`; reads `outer-tty-<tag>`. | small shell-glue; possible Go subcommand but low packaging impact | P2 |
   136	| `pair quit` (was `bin/pair-quit.sh`; `cmd/internal/launcher/restart.go`) | in-process Go subcommand, ported from a Bash keybind helper | nvim `PairConfirmQuit` (`{ 'pair', 'quit' }`) | Touch quit marker then kill zellij session. | Uses `ZELLIJ_SESSION_NAME`, `PAIR_KILL_CMD`; writes cache marker. | **ported in #94 M1** — `runQuit` reuses the launcher's `TouchQuitMarker`/`ExecKillSession` seam; `.sh` removed from tree + runtime bundle | P2 |

exec
/bin/zsh -lc 'env -u PAIR_SESSION_ID -u PAIR_TAG go test ./...' in /private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout
 exited 1 in 11794ms:
ok  	github.com/xianxu/pair/cmd/internal/adapt	0.178s
ok  	github.com/xianxu/pair/cmd/internal/agentcmd	0.312s
ok  	github.com/xianxu/pair/cmd/internal/ansi	0.446s
ok  	github.com/xianxu/pair/cmd/internal/changelogcmd	10.782s
ok  	github.com/xianxu/pair/cmd/internal/clipcmd	0.424s
ok  	github.com/xianxu/pair/cmd/internal/codexsid	0.560s
ok  	github.com/xianxu/pair/cmd/internal/contextcmd	7.457s
ok  	github.com/xianxu/pair/cmd/internal/continuationcmd	1.341s
ok  	github.com/xianxu/pair/cmd/internal/ctxmeter	0.955s
ok  	github.com/xianxu/pair/cmd/internal/dispatcher	1.128s
ok  	github.com/xianxu/pair/cmd/internal/draftroute	1.268s
ok  	github.com/xianxu/pair/cmd/internal/entrypoint	1.401s
ok  	github.com/xianxu/pair/cmd/internal/keyhelp	1.540s
ok  	github.com/xianxu/pair/cmd/internal/keyscmd	5.510s
ok  	github.com/xianxu/pair/cmd/internal/launcher	8.142s
ok  	github.com/xianxu/pair/cmd/internal/layoutcmd	1.818s
--- FAIL: TestRunForwardsMaxTokensAndVerbosity (0.00s)
panic: httptest: failed to listen on a port: listen tcp6 [::1]:0: bind: operation not permitted [recovered, repanicked]

goroutine 11 [running]:
testing.tRunner.func1.2({0x1050a85c0, 0x298ac6222500})
	/opt/homebrew/Cellar/go/1.26.6/libexec/src/testing/testing.go:1974 +0x1a0
testing.tRunner.func1()
	/opt/homebrew/Cellar/go/1.26.6/libexec/src/testing/testing.go:1977 +0x318
panic({0x1050a85c0?, 0x298ac6222500?})
	/opt/homebrew/Cellar/go/1.26.6/libexec/src/runtime/panic.go:860 +0x12c
net/http/httptest.newLocalListener()
	/opt/homebrew/Cellar/go/1.26.6/libexec/src/net/http/httptest/server.go:72 +0xd4
net/http/httptest.NewUnstartedServer(...)
	/opt/homebrew/Cellar/go/1.26.6/libexec/src/net/http/httptest/server.go:120
net/http/httptest.NewServer({0x105133b80, 0x298ac61f43a8})
	/opt/homebrew/Cellar/go/1.26.6/libexec/src/net/http/httptest/server.go:107 +0x28
github.com/xianxu/pair/cmd/internal/model.TestRunForwardsMaxTokensAndVerbosity(0x298ac633ab48)
	/private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout/cmd/internal/model/model_test.go:51 +0x8c
testing.tRunner(0x298ac633ab48, 0x10512fc00)
	/opt/homebrew/Cellar/go/1.26.6/libexec/src/testing/testing.go:2036 +0xc4
created by testing.(*T).Run in goroutine 1
	/opt/homebrew/Cellar/go/1.26.6/libexec/src/testing/testing.go:2101 +0x3a8
FAIL	github.com/xianxu/pair/cmd/internal/model	1.853s
ok  	github.com/xianxu/pair/cmd/internal/opener	1.727s
ok  	github.com/xianxu/pair/cmd/internal/osfs	1.680s
ok  	github.com/xianxu/pair/cmd/internal/procutil	1.683s
ok  	github.com/xianxu/pair/cmd/internal/readiness	1.762s
ok  	github.com/xianxu/pair/cmd/internal/reviewcmd	1.819s
ok  	github.com/xianxu/pair/cmd/internal/runtimebundle	1.835s
?   	github.com/xianxu/pair/cmd/internal/runtimebundle/generatecmd	[no test files]
?   	github.com/xianxu/pair/cmd/internal/runtimebundle/manifestmodel	[no test files]
ok  	github.com/xianxu/pair/cmd/internal/runtimebundlegen	6.056s
ok  	github.com/xianxu/pair/cmd/internal/scribecmd	1.817s
ok  	github.com/xianxu/pair/cmd/internal/scrollbackcmd	1.773s
ok  	github.com/xianxu/pair/cmd/internal/sessionwatch	1.748s
ok  	github.com/xianxu/pair/cmd/internal/slugcmd	4.632s
ok  	github.com/xianxu/pair/cmd/internal/termcmd	1.785s
ok  	github.com/xianxu/pair/cmd/internal/textwidth	1.772s
ok  	github.com/xianxu/pair/cmd/internal/titlepoller	1.747s
ok  	github.com/xianxu/pair/cmd/internal/transcript	1.810s
ok  	github.com/xianxu/pair/cmd/internal/workbenchshortcut	1.859s
?   	github.com/xianxu/pair/cmd/internal/workbenchshortcut/generatecmd	[no test files]
--- FAIL: TestPublishAgentReadySkipsWhenPairEnvIncomplete (0.00s)
    readiness_test.go:50: ready file stat err = <nil>, want not exist
FAIL
FAIL	github.com/xianxu/pair/cmd/internal/wrapcmd	3.115s
ok  	github.com/xianxu/pair/cmd/internal/zellijpane	1.759s
ok  	github.com/xianxu/pair/cmd/pair-go	5.883s
FAIL

codex
Focused `sessionwatch` tests pass. The implementation also covers exactly the three async specs currently registered, while Claude remains excluded by the existing synchronous path. One documentation inconsistency is confirmed: the atlas still describes a universally terminating 60-second watcher and timeout failure, which is no longer true for PID-bound watches.
exec
/bin/zsh -lc "env | rg '"'^PAIR_'"' | cut -d= -f1 | sort; sed -n '1,80p' cmd/internal/wrapcmd/readiness_test.go; git diff d08c0bec1669719090fdb5bb0f793e9db4272a12...HEAD -- cmd/internal/wrapcmd cmd/internal/model; git status --short" in /private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout
 succeeded in 0ms:
PAIR_AGENT
PAIR_AGENT_ARGS
PAIR_DATA_DIR
PAIR_DEV
PAIR_HOME
PAIR_LAUNCH_NONCE
PAIR_PANE_TITLE
PAIR_SESSION_ID
PAIR_SESSION_NAME
PAIR_TAG
PAIR_WORKBENCH_LAYOUT
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

codex
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation satisfies the lifecycle requirements: PID-bound watches continue at a 60-second cadence, stop after process death, and PID-less watches remain bounded. Focused tests pass and cover all three asynchronous agents. Shipping is held only by stale atlas documentation describing the old universal 60-second timeout.

## 1. Strengths

- [run.go:121](/private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout/cmd/internal/sessionwatch/run.go:121) retains one shared lifecycle loop for Codex, Agy, and Muse.
- [run.go:125](/private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout/cmd/internal/sessionwatch/run.go:125) preserves bounded timeout behavior when no fresh PID is available.
- [run.go:162](/private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout/cmd/internal/sessionwatch/run.go:162) cleanly switches from fast polling to the configurable slow cadence.
- [run_test.go:294](/private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout/cmd/internal/sessionwatch/run_test.go:294) tests delayed discovery for every registered async agent using the stateful runtime seam.
- Process-bound and PID-less exit paths are both tested.

## 2. Critical findings

None.

## 3. Important findings

- [atlas/how-to-bring-up-a-new-harness-cli.md:86](/private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout/atlas/how-to-bring-up-a-new-harness-cli.md:86) still states that `fail` occurs whenever the 60-second watch window expires. PID-bound watches now continue for the process lifetime and do not emit that failure. [atlas/go-migration-inventory.md:130](/private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout/atlas/go-migration-inventory.md:130) similarly calls this a “background 60s watcher.” Update both descriptions to distinguish the 60-second fast-discovery window from subsequent 60-second lifecycle polling, and reserve timeout failure for PID-less discovery.

No README update is required: the change introduces no user-entered command, flag, configuration key, or usage step.

## 4. Minor findings

None.

## 5. Test coverage notes

- `go test ./cmd/internal/sessionwatch` passes.
- The delayed-discovery table covers Codex, Agy, and Muse.
- Tests pin the fast-to-slow transition, session persistence, process-exit path, and PID-less timeout.
- Repository-wide `go test ./...` could not complete cleanly in this sandbox:
  - `cmd/internal/model` cannot bind an `httptest` listener.
  - `cmd/internal/wrapcmd` inherits Pair launch environment variables not cleared by the attempted invocation.
- Neither failure touches this diff.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass:** one shared schedule implements all asynchronous agents; no duplicated per-agent lifecycle logic.
- **ARCH-PURE — pass:** clock, sleeping, process state, filesystem discovery, writes, and logging remain behind the injected `Runtime`.
- **ARCH-PURPOSE — pass:** the complete registered async-agent set derives from the shared behavior; Claude remains excluded through `SpecForAgent`.
- **ARCH-MOCK — pass:** production and tests share the same runtime boundary, and the fake models clock, process, transcript, and write state across calls.

No plan-gate carry-forward file exists, and the plan has no Core concepts table requiring entity reconciliation.

## 7. Plan revision recommendations

None. The Plan and Revisions match the delivered code; the required change is to the atlas documentation rather than the issue plan.
tokens used
67,542
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation satisfies the lifecycle requirements: PID-bound watches continue at a 60-second cadence, stop after process death, and PID-less watches remain bounded. Focused tests pass and cover all three asynchronous agents. Shipping is held only by stale atlas documentation describing the old universal 60-second timeout.

## 1. Strengths

- [run.go:121](/private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout/cmd/internal/sessionwatch/run.go:121) retains one shared lifecycle loop for Codex, Agy, and Muse.
- [run.go:125](/private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout/cmd/internal/sessionwatch/run.go:125) preserves bounded timeout behavior when no fresh PID is available.
- [run.go:162](/private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout/cmd/internal/sessionwatch/run.go:162) cleanly switches from fast polling to the configurable slow cadence.
- [run_test.go:294](/private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout/cmd/internal/sessionwatch/run_test.go:294) tests delayed discovery for every registered async agent using the stateful runtime seam.
- Process-bound and PID-less exit paths are both tested.

## 2. Critical findings

None.

## 3. Important findings

- [atlas/how-to-bring-up-a-new-harness-cli.md:86](/private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout/atlas/how-to-bring-up-a-new-harness-cli.md:86) still states that `fail` occurs whenever the 60-second watch window expires. PID-bound watches now continue for the process lifetime and do not emit that failure. [atlas/go-migration-inventory.md:130](/private/tmp/worktree/pair-context-usage/000143-keep-agent-session-discovery-alive-after-startup-timeout/atlas/go-migration-inventory.md:130) similarly calls this a “background 60s watcher.” Update both descriptions to distinguish the 60-second fast-discovery window from subsequent 60-second lifecycle polling, and reserve timeout failure for PID-less discovery.

No README update is required: the change introduces no user-entered command, flag, configuration key, or usage step.

## 4. Minor findings

None.

## 5. Test coverage notes

- `go test ./cmd/internal/sessionwatch` passes.
- The delayed-discovery table covers Codex, Agy, and Muse.
- Tests pin the fast-to-slow transition, session persistence, process-exit path, and PID-less timeout.
- Repository-wide `go test ./...` could not complete cleanly in this sandbox:
  - `cmd/internal/model` cannot bind an `httptest` listener.
  - `cmd/internal/wrapcmd` inherits Pair launch environment variables not cleared by the attempted invocation.
- Neither failure touches this diff.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass:** one shared schedule implements all asynchronous agents; no duplicated per-agent lifecycle logic.
- **ARCH-PURE — pass:** clock, sleeping, process state, filesystem discovery, writes, and logging remain behind the injected `Runtime`.
- **ARCH-PURPOSE — pass:** the complete registered async-agent set derives from the shared behavior; Claude remains excluded through `SpecForAgent`.
- **ARCH-MOCK — pass:** production and tests share the same runtime boundary, and the fake models clock, process, transcript, and write state across calls.

No plan-gate carry-forward file exists, and the plan has no Core concepts table requiring entity reconciliation.

## 7. Plan revision recommendations

None. The Plan and Revisions match the delivered code; the required change is to the atlas documentation rather than the issue plan.
