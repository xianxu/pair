# Boundary Review — pair#114 (whole-issue close)

| field | value |
|-------|-------|
| issue | 114 — draft completion should skip outside insert mode |
| repo | pair |
| issue file | workshop/issues/000114-draft-complete-insert-mode.md |
| boundary | whole-issue close |
| milestone | — |
| window | d04ca665612c9441492121404c6fb1579cb35364..HEAD |
| command | sdlc close --issue 114 |
| reviewer | codex |
| timestamp | 2026-07-10T08:03:27-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reading additional input from stdin...
OpenAI Codex v0.144.1
--------
workdir: /Users/xianxu/workspace/pair
model: gpt-5.6-sol
provider: openai
approval: never
sandbox: workspace-write [workdir, /tmp, $TMPDIR, /tmp] (network access enabled)
reasoning effort: none
reasoning summaries: none
session id: 019f4c8c-f545-70d2-b66a-5d305cf233ab
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **pair** repository.

- repository: pair   (root: /Users/xianxu/workspace/pair)
- issue:      pair#114   (file: workshop/issues/000114-draft-complete-insert-mode.md)
- window:     Base: d04ca665612c9441492121404c6fb1579cb35364   Head: HEAD

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
  - Docs / atlas updated for new surface (see the Docs update gate).

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

Work through each of ARCH-DRY, ARCH-PURE, ARCH-PURPOSE explicitly, applying its at-review lens. The
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


OUTPUT CONTRACT (machine-read — do not deviate). LEAD your response with the
fenced ```verdict block shown above — that is the authoritative handoff the binary
reads (its `verdict:` value is one of the listed tokens). Everything after the block
is advisory: a non-blocking verdict WITH findings still PASSES the gate. A bare
`VERDICT: <TOKEN>` line is accepted only as a FALLBACK when the block is absent.

Diff:
diff --git a/Makefile.local b/Makefile.local
index f9d8846..f960918 100644
--- a/Makefile.local
+++ b/Makefile.local
@@ -1,7 +1,7 @@
 # Repo-specific Makefile targets.
 # This file is included by Makefile — add your own targets here.
 
-.PHONY: build install pair runtimebundle-generate runtimebundle-drift-check test-runtimebundle test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-pair-embedded-runtime test-pair-restart-quit test-emitter-health test-session-watch test-changelog test-copy-on-select test-pair-title test-zellij-trace test-review
+.PHONY: build install pair runtimebundle-generate runtimebundle-drift-check test-runtimebundle test test-race test-run-headless test-lua test-queue test-autopair test-draft-complete test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-pair-embedded-runtime test-pair-restart-quit test-emitter-health test-session-watch test-changelog test-copy-on-select test-pair-title test-zellij-trace test-review
 
 # Cross-cutting Go-binary targets. Every Go binary lives under cmd/<name>/
 # and ships with the `pair-` prefix so it doesn't collide with anything
@@ -71,7 +71,7 @@ pair: $(BIN_DIR)/pair
 #                     inside the vt.Emulator library API itself, which is
 #                     outside our scope to fix — exclude that package
 #                     here rather than chase a library-level race.
-test: test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-pair-embedded-runtime test-pair-restart-quit test-emitter-health test-session-watch test-changelog test-copy-on-select test-pair-title test-zellij-trace test-review
+test: test-run-headless test-lua test-queue test-autopair test-draft-complete test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-pair-embedded-runtime test-pair-restart-quit test-emitter-health test-session-watch test-changelog test-copy-on-select test-pair-title test-zellij-trace test-review
 	go test ./... -count=1
 
 # pair-title poller (#93 M1, now Go): single-instance guard (identity-checked
@@ -180,6 +180,13 @@ test-zellij-trace:
 test-autopair:
 	bash tests/autopair-test.sh
 
+# Regression test for the draft typeahead completer mode guard (nvim/init.lua,
+# run_completers). Drives the real init.lua headlessly and asserts the shared
+# runner no-ops outside Insert mode before any vim.fn.complete() call can raise
+# E785. See tests/draft-complete-mode-test.sh.
+test-draft-complete:
+	bash tests/draft-complete-mode-test.sh
+
 # Regression test for the insert-mode <CR> decision (nvim/init.lua, cr_keys /
 # _G.PairCRKeys, #65): Return must always insert a newline when no completion
 # is selected, dismissing the popup. Asserts the expr output for all three
diff --git a/nvim/init.lua b/nvim/init.lua
index b8930b0..b2515de 100644
--- a/nvim/init.lua
+++ b/nvim/init.lua
@@ -3834,7 +3834,13 @@ end
 -- run 30ms after the last event.
 local complete_last_fire = 0
 local complete_pending = nil
+function _G.PairDraftCompletionCanRun()
+  local mode = vim.api.nvim_get_mode().mode or ''
+  return mode:sub(1, 1) == 'i'
+end
+
 local function run_completers()
+  if not _G.PairDraftCompletionCanRun() then return end
   -- The explicit z= gesture (spell_suggest_popup) owns the popup while it's
   -- active, so the as-you-type completers must stay out of its way. Without
   -- this guard, startinsert's TextChangedI drives spell_complete to pop its own
@@ -3847,6 +3853,13 @@ local function run_completers()
   if word_complete() then return end
   spell_complete()
 end
+-- Exposed for tests/draft-complete-mode-test.sh. The live path reaches this
+-- runner through TextChangedI/P, but the regression drives the runner directly
+-- from Normal mode to keep the E785 failure deterministic under headless nvim.
+_G.PairDraftCompleteTest = {
+  can_run = _G.PairDraftCompletionCanRun,
+  run_completers = run_completers,
+}
 vim.api.nvim_create_autocmd({ 'TextChangedI', 'TextChangedP' }, {
   group = pair_aug,
   callback = function()
diff --git a/tests/draft-complete-mode-test.sh b/tests/draft-complete-mode-test.sh
new file mode 100644
index 0000000..e87c5d6
--- /dev/null
+++ b/tests/draft-complete-mode-test.sh
@@ -0,0 +1,77 @@
+#!/usr/bin/env bash
+# Regression test for draft typeahead completion mode safety (nvim/init.lua).
+#
+# TextChangedI/P can debounce completion through a scheduled callback. If the
+# user leaves Insert mode before that callback runs, the shared completion
+# runner must no-op; otherwise word_complete/path_complete/spell_complete can
+# call vim.fn.complete() from Normal/Visual mode and raise E785.
+#
+# Run: bash tests/draft-complete-mode-test.sh   (also wired into `make test`)
+set -uo pipefail
+
+ROOT="$(cd "$(dirname "$0")/.." && pwd)"
+INIT="$ROOT/nvim/init.lua"
+. "$ROOT/tests/lib/run-headless.sh"   # run_headless: timeout watchdog (#60)
+RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-draft-complete-test.XXXXXX")"
+trap 'rm -rf "$RT"' EXIT
+
+printf '' > "$RT/draft.md"
+
+cat > "$RT/driver.lua" <<'LUA'
+local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
+local fails = 0
+local function check(cond, msg, detail)
+  if cond then
+    O:write('ok\t' .. msg .. '\n')
+  else
+    fails = fails + 1
+    O:write('FAIL\t' .. msg .. '\t' .. tostring(detail or '') .. '\n')
+  end
+end
+
+check(type(_G.PairDraftCompleteTest) == 'table', 'test seam exported')
+check(type(_G.PairDraftCompleteTest.run_completers) == 'function', 'runner exported')
+
+-- Prefix "complet" has a same-buffer candidate "completion", so without the
+-- shared mode guard run_completers reaches word_complete() and vim.fn.complete().
+vim.api.nvim_buf_set_lines(0, 0, -1, false, { 'complet completion' })
+vim.api.nvim_win_set_cursor(0, { 1, #'complet' })
+vim.cmd('stopinsert')
+local mode = vim.api.nvim_get_mode().mode
+check(mode:sub(1, 1) ~= 'i', 'driver is outside insert mode', mode)
+
+local ok, err = pcall(_G.PairDraftCompleteTest.run_completers)
+check(ok, 'runner skips outside insert mode without E785', err)
+
+O:write('TOTAL_FAILS=' .. fails .. '\n')
+O:close()
+vim.cmd('qall!')
+LUA
+
+if ! run_headless --timeout 30 -- \
+  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
+  nvim --headless -u "$INIT" "$RT/draft.md" \
+  -c "luafile $RT/driver.lua"; then
+  echo "draft-complete-mode-test: nvim driver failed"
+  exit 1
+fi
+
+echo "draft-complete-mode-test:"
+fails=0
+if [ ! -f "$RT/result.txt" ]; then
+  echo "  FAIL driver produced no result (nvim boot/driver error)"
+  exit 1
+fi
+while IFS=$'\t' read -r status label detail; do
+  case "$status" in
+    ok)   printf '  ok   %s\n' "$label" ;;
+    FAIL) printf '  FAIL %s: %s\n' "$label" "$detail"; fails=$((fails + 1)) ;;
+    TOTAL_FAILS=*) ;;
+  esac
+done < "$RT/result.txt"
+
+if [ "$fails" -ne 0 ]; then
+  echo "draft-complete-mode-test: $fails failure(s)"
+  exit 1
+fi
+echo "draft-complete-mode-test: all passed"


codex
I’m using the repository’s “using superpowers” workflow skill to honor its session-start contract, then I’ll inspect the issue, exact diff, surrounding completion code, and run the focused regression test read-only.
exec
/bin/zsh -lc "sed -n '1,240p' /Users/xianxu/workspace/ariadne/construct/adapted/superpowers-using-superpowers/SKILL.md && sdlc --help && sed -n '1,240p' workshop/lessons.md" in /Users/xianxu/workspace/pair
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
  - Publish ONCE at issue close, not per milestone — and do NOT reuse a branch
    name that already has a merged PR. `sdlc merge` refuses (#148) when a branch
    has commits not in main despite a merged PR (a reused name would otherwise
    silently strand the new commits); rename to a fresh branch, `sdlc pr`, retry.

RECOVER
  - After a compaction or session resume, run `sdlc state` to recover where you
    are instead of re-inferring from issue files.

LOCAL REPO TRANSACTION LOCK
  - Mutating verbs take an SDLC-owned repo transaction lock at
    `.git/sdlc.lock` before reading/writing issue state, committing, changing
    branches, or pushing. The lock is local to the Git common dir, so linked
    worktrees of the same repo serialize with each other.
  - Wait messages identify the holder pid and command when metadata is
    available. `close` and `milestone-close` release the lock while the external
    boundary-review subprocess runs, then reacquire before finalization; if HEAD
    or the issue/project file state they prepared changed meanwhile, they refuse
    to finalize and tell you to rerun. `change-code`, `merge`, and `push` can still hold the lock during
    long-running review/ship transactions; wait or retry rather than removing
    the lock while that process is alive.
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
  resolve         Resolve a symbolic artifact ref (ariadne#11, #15 M4) to its current path(s) — read-only
  open            Resolve a ref and open the primary artifact in $EDITOR
  judge           Run an LLM-judge check against the diff (fresh-context)
  arch-principles Print the ARCH-* architecture principles (single source; pull for non-gate work)
  estimate-source Name the shared estimate method + the repo-local calibration source (pull)
  process-manual  Unroll every injection source into a linked process manual (#153)
  propagate-base  Re-weave every recursive dependent of this repo (foundation-first)
  help            Help about any command

Flags:
  -h, --help   help for sdlc

Use "sdlc [command] --help" for more information about a command.
# Lessons

## Async buffer requests need live anchors, not saved coordinates

Pair review definitions originally stored the selected line/column range while
the agent produced an answer. If the user inserted text before the selected term
before the result arrived, the response applied to stale coordinates and inserted
the footnote reference into the wrong text.

**Rule.** Any Neovim request that crosses an async boundary and later mutates the
same buffer must anchor the target with an extmark (or re-locate/validate the
target from content) before applying the result. Raw row/column pairs are only a
snapshot. Add an integration regression that mutates text before the target while
the request is pending, then verifies the result follows the target or aborts
cleanly. Caught in #000112 close review.

## Generated review sidecars must stay bounded

`sdlc close` writes a review sidecar, and that sidecar becomes part of later
diffs. If it stores the full raw prompt/transcript, it can bloat the reviewed
diff and carry whitespace-sensitive embedded patches.

**Rule.** Keep committed review sidecars to the durable facts: verdict, window,
findings, verification, and resolution. Avoid committing full prompt/diff
transcripts unless the generator normalizes them and they remain small enough
for future review prompts.

Caught while closing #000108.

## Path precedence contracts need explicit divergent-env tests

#90's embedded runtime implementation documented extraction under
`$PAIR_DATA_DIR/runtime/<digest>/pair-home`, but the first OS-backed
implementation only used the XDG/home resolver. The copied-binary smoke unset
`PAIR_DATA_DIR`, so the bug survived until boundary review tried
`PAIR_DATA_DIR` and `XDG_DATA_HOME` with different roots.

**Rule.** When a feature promises environment-variable precedence, add a test
where the higher-priority and fallback variables are both set to different
directories, then assert the selected path. Also include every Go source file
that can change build output in Make prerequisites; a generated or embedded
artifact path should have a dependency test or an explicit review checklist
entry. Caught in #000090 boundary review.

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

codex
The SDLC skill applies because this is the whole-issue boundary gate. I’m using it only to understand the review contract; I will not mutate tracker or repository state.
exec
/bin/zsh -lc "sed -n '1,260p' /Users/xianxu/workspace/ariadne/construct/local/sdlc/SKILL.md && git status --short && git diff --stat d04ca665612c9441492121404c6fb1579cb35364..HEAD && git diff --name-status d04ca665612c9441492121404c6fb1579cb35364..HEAD && sed -n '1,260p' workshop/issues/000114-draft-complete-insert-mode.md" in /Users/xianxu/workspace/pair
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
 M workshop/issues/000083-pair-workbench-prescribed-way-of-when-to-use-which-agent.md
 Makefile.local                                     | 11 +++-
 nvim/init.lua                                      | 13 ++++
 tests/draft-complete-mode-test.sh                  | 77 ++++++++++++++++++++++
 .../issues/000114-draft-complete-insert-mode.md    | 18 +++--
 4 files changed, 113 insertions(+), 6 deletions(-)
M	Makefile.local
M	nvim/init.lua
A	tests/draft-complete-mode-test.sh
M	workshop/issues/000114-draft-complete-insert-mode.md
---
id: 000114
status: working
deps: []
github_issue:
created: 2026-07-10
updated: 2026-07-10
estimate_hours: 0.45
started: 2026-07-10T07:54:48-07:00
---

# draft completion should skip outside insert mode

## Problem

Draft-pane typeahead completion is scheduled from `TextChangedI/P`. If the user
leaves Insert mode before a debounced callback runs, the callback can still call
`vim.fn.complete()`. Neovim raises `E785: complete() can only be used in Insert
mode`, currently observed when selecting a word around spell-check/completion
flows.

## Spec

- The draft completion runner must no-op unless the current mode is Insert or
  completion-select mode.
- Delayed/debounced callbacks must re-check mode at execution time, not only at
  scheduling time.
- Existing path, word, and spell completion behavior in Insert mode is
  preserved.

ARCH-PURPOSE: the fix must prevent every `run_completers()` path from reaching
`complete()` outside Insert mode, not only one completer.
ARCH-DRY: put the guard at the shared completion runner, not separately in
`path_complete`, `word_complete`, and `spell_complete`.

## Done when

- A regression simulates the draft completer running in Normal/Visual mode and
  proves it does not call `complete()` or raise `E785`.
- Existing nvim headless tests pass for the draft completion area.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 0.9
item: lua-neovim design=0.10 impl=0.20
item: milestone-review design=0.00 impl=0.15
total: 0.45
```

## Plan

- [x] Expose or isolate the draft completer mode guard enough for a headless
      regression.
- [x] Add a failing regression for a scheduled completer execution outside
      Insert mode.
- [x] Guard the shared completion runner before any completer can call
      `vim.fn.complete()`.
- [x] Run focused nvim tests and close #114.

## Log

### 2026-07-10
- Created from reported stack trace:
  `Vim:E785: complete() can only be used in Insert mode` from
  `word_complete()` via scheduled `run_completers()`.
- Added `tests/draft-complete-mode-test.sh`; before the guard it failed with
  `Vim:E785: complete() can only be used in Insert mode`, reproducing the
  reported error through the real `nvim/init.lua` runner.
- Fixed the shared draft completer runner to no-op unless Neovim reports an
  Insert-mode variant, covering path, word, and spell typeahead from the single
  `run_completers()` entry point (`ARCH-DRY`, `ARCH-PURPOSE`).
- Verification passed: `bash tests/draft-complete-mode-test.sh`,
  `bash tests/autopair-test.sh`, `bash tests/cr-newline-test.sh`,
  `make test-lua`, `make test-draft-complete`, `git diff --check`, and
  `make test`.

exec
/bin/zsh -lc "git diff --check d04ca665612c9441492121404c6fb1579cb35364..HEAD && git diff --unified=100 d04ca665612c9441492121404c6fb1579cb35364..HEAD -- nvim/init.lua tests/draft-complete-mode-test.sh Makefile.local workshop/issues/000114-draft-complete-insert-mode.md && rg -n \"function (path_complete|word_complete|spell_complete)|run_completers|complete_pending|PairDraftComplete|PairDraftCompletion\" nvim/init.lua tests -g '"'!tests/draft-complete-mode-test.sh'"'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
diff --git a/Makefile.local b/Makefile.local
index f9d8846..f960918 100644
--- a/Makefile.local
+++ b/Makefile.local
@@ -1,278 +1,285 @@
 # Repo-specific Makefile targets.
 # This file is included by Makefile — add your own targets here.
 
-.PHONY: build install pair runtimebundle-generate runtimebundle-drift-check test-runtimebundle test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-pair-embedded-runtime test-pair-restart-quit test-emitter-health test-session-watch test-changelog test-copy-on-select test-pair-title test-zellij-trace test-review
+.PHONY: build install pair runtimebundle-generate runtimebundle-drift-check test-runtimebundle test test-race test-run-headless test-lua test-queue test-autopair test-draft-complete test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-pair-embedded-runtime test-pair-restart-quit test-emitter-health test-session-watch test-changelog test-copy-on-select test-pair-title test-zellij-trace test-review
 
 # Cross-cutting Go-binary targets. Every Go binary lives under cmd/<name>/
 # and ships with the `pair-` prefix so it doesn't collide with anything
 # else on PATH (pair-scribe replaced an earlier bare-named `scribe`).
 #
 # Adding a new binary: drop main.go under cmd/<name>/, append <name>
 # to GO_BINS, add the per-binary recipe stanza. `make build` and
 # `make install` pick it up automatically.
 #
 #   make build      → compile every binary in GO_BINS into ./bin/
 #   make install    → also copy each into ~/.local/bin/
 #   make <name>     → build just that one (e.g., `make pair-scribe`)
 #
 # Why ./bin/ for the in-tree build: keeps Go output next to the shell
 # scripts pair already ships from bin/, and pair-scrollback-open looks
 # at $PAIR_HOME/bin/pair-scrollback-render to pick up the Go renderer
 # in both dev (source tree) and brew-install layouts.
 #
 # Why ~/.local/bin/ for install: XDG convention, on PATH per the user's
 # zshrc. Each binary the user wires into their shell (e.g. pair-scribe
 # in zshrc) should reference $HOME/.local/bin/<name>.
 
 BIN_DIR     := bin
 INSTALL_DIR := $(HOME)/.local/bin
 # The single binary (#104). Every former helper is a `pair <subcommand>`.
 GO_BINS     := pair
 # BUSYBOX_LINKS: pair invoked under one of these names routes to the matching
 # subcommand (entrypoint busybox dispatch). Only pair-slug survives — the
 # external Claude Stop hook may still call it by bare name. Emitted as symlinks
 # → pair by BOTH `build` and `install`: `build` is load-bearing because pair-dev
 # rebuilds via `make build` (not install) and the session PATH fronts the repo
 # bin/, while `install` covers ~/.local/bin.
 BUSYBOX_LINKS := pair-slug
 SHELL_BINS  := pair-dev
 PAIR_HOME_LDFLAGS := -X main.defaultPairHome=$(CURDIR)
 RUNTIMEBUNDLE_ASSETS := cmd/internal/runtimebundle/assets/runtime/manifest.json
 
 build: $(addprefix $(BIN_DIR)/,$(GO_BINS)) $(addprefix $(BIN_DIR)/,$(BUSYBOX_LINKS))
 
 install: build
 	@mkdir -p $(INSTALL_DIR)
 	@for b in $(GO_BINS); do \
 	    install -m 755 $(BIN_DIR)/$$b $(INSTALL_DIR)/$$b && \
 	    echo "installed: $(INSTALL_DIR)/$$b"; \
 	done
 	@for l in $(BUSYBOX_LINKS); do \
 	    ln -sf pair "$(INSTALL_DIR)/$$l" && \
 	    echo "linked: $(INSTALL_DIR)/$$l -> pair"; \
 	done
 	@for b in $(SHELL_BINS); do \
 	    ln -sf "$(CURDIR)/$(BIN_DIR)/$$b" "$(INSTALL_DIR)/$$b" && \
 	    echo "linked: $(INSTALL_DIR)/$$b -> $(CURDIR)/$(BIN_DIR)/$$b"; \
 	done
 
 # `make pair` builds the single binary (the recipe below also emits the busybox
 # symlinks, so `build`/`install`/`pair` all get them without a recipe on `build`,
 # which would clash with the base layer's `build` target).
 pair: $(BIN_DIR)/pair
 
 # Tests
 #   make test       → go test ./... (cache disabled so reruns actually run)
 #   make test-race  → race-detector run, scoped to packages where the
 #                     suite has actual concurrent code worth checking.
 #                     pair-wrap has translateStdin's goroutine + timer +
 #                     channels; pair-scrollback-render's render() spawns
 #                     a goroutine drainer that races with em.Close()
 #                     inside the vt.Emulator library API itself, which is
 #                     outside our scope to fix — exclude that package
 #                     here rather than chase a library-level race.
-test: test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-pair-embedded-runtime test-pair-restart-quit test-emitter-health test-session-watch test-changelog test-copy-on-select test-pair-title test-zellij-trace test-review
+test: test-run-headless test-lua test-queue test-autopair test-draft-complete test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-pair-embedded-runtime test-pair-restart-quit test-emitter-health test-session-watch test-changelog test-copy-on-select test-pair-title test-zellij-trace test-review
 	go test ./... -count=1
 
 # pair-title poller (#93 M1, now Go): single-instance guard (identity-checked
 # liveness so a recycled PID can't suppress the respawn) + the #71 frame meter
 # (renames each agent pane to "<agent> (<count>) [<cwd>]", with unchanged-skip)
 # + the cmux heat-ramp. The old shell harness (PAIR_TITLE_TEST_CALL hook) retired
 # with the port; its coverage moved to Go unit tests over the Runtime seam.
 test-pair-title:
 	go test ./cmd/internal/titlepoller ./cmd/internal/procutil -count=1
 
 # Session id watcher must ignore stale pidfiles from prior launches and wait
 # for pair-wrap to publish the current agent PID.
 test-session-watch: $(BIN_DIR)/pair
 	bash tests/pair-session-watch-test.sh
 
 # The bundle carries config + shell shims only (#104 M3) — no helper binaries to
 # build first, so no prereq.
 runtimebundle-generate:
 	go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime
 
 $(RUNTIMEBUNDLE_ASSETS): runtimebundle-generate
 
 runtimebundle-drift-check:
 	@d1="$$(mktemp -d)"; d2="$$(mktemp -d)"; \
 	trap 'rm -rf "$$d1" "$$d2"' EXIT; \
 	go run ./cmd/internal/runtimebundle/generatecmd --repo . --out "$$d1/runtime" && \
 	go run ./cmd/internal/runtimebundle/generatecmd --repo . --out "$$d2/runtime" && \
 	diff -qr "$$d1/runtime" "$$d2/runtime" >/dev/null && \
 	find "$$d1/runtime/files" -type f -exec stat -f '%N %OLp' {} + | sed "s#$$d1/runtime/files/##" | sort > "$$d1/modes" && \
 	find "$$d2/runtime/files" -type f -exec stat -f '%N %OLp' {} + | sed "s#$$d2/runtime/files/##" | sort > "$$d2/modes" && \
 	diff -u "$$d1/modes" "$$d2/modes" >/dev/null
 
 test-runtimebundle: runtimebundle-generate
 	go test ./cmd/internal/runtimebundle -count=1
 
 # Review workbench (#66): the integration shell tests. The pure modules
 # (record, reconstruct) run under `make test-lua`.
 # The review helpers are now built Go binaries (#93 M3), not committed shell
 # scripts — declare them as prereqs so a fresh-tree `make test` builds them
 # before the review suite execs `$ROOT/bin/pair-review-*`.
 test-review: $(BIN_DIR)/pair
 	bash tests/review-docflow-test.sh
 	bash tests/review-apply-test.sh
 	bash tests/review-reconcile-test.sh
 	bash tests/review-handoff-test.sh
 	bash tests/review-loop-test.sh
 	bash tests/review-projection-test.sh
 	bash tests/review-poke-test.sh
 	bash tests/review-window-test.sh
 	bash tests/review-toggle-test.sh
 	bash tests/pair-review-target-test.sh
 	bash tests/review-readiness-cli-test.sh
 	bash tests/review-indicator-test.sh
 	bash tests/review-resume-test.sh
 	bash tests/review-definition-test.sh
 
 test-race:
 	go test -count=1 -race ./cmd/pair-wrap/
 
 # Self-test for the headless-nvim timeout watchdog (tests/lib/run-headless.sh,
 # #60). Runs FIRST in `test` so the safety net every headless-nvim sub-target
 # now relies on is proven before those targets run — without it, a driver that
 # fails to quit (bare `qall` on a modified buffer → E37) hangs the whole suite.
 test-run-headless:
 	bash tests/run-headless-test.sh
 
 # Headless Lua tests for nvim/*.lua decision modules (issue #000027). nvim -l
 # runs a Lua script in nvim's interpreter with the full vim.api available, so
 # pure logic AND buffer-manipulation can be asserted without a UI.
 test-lua:
 	nvim -l nvim/slug_test.lua
 	nvim -l nvim/scrollback_test.lua
 	nvim -l nvim/annotate_test.lua
 	nvim -l nvim/changelog_test.lua
 	nvim -l nvim/adapt_test.lua
 	nvim -l nvim/doctor_test.lua
 	nvim -l nvim/review/record_test.lua
 	nvim -l nvim/review/reconstruct_test.lua
 	nvim -l nvim/review/define_test.lua
 	nvim -l nvim/review/markers_test.lua
 	nvim -l nvim/review/seam_test.lua
 	nvim -l nvim/review/mode_test.lua
 	nvim -l nvim/review/poke_bodies_test.lua
 	nvim -l nvim/review/wrap_test.lua
 	nvim -l nvim/review/readiness_test.lua
 	nvim -l nvim/review/spinner_test.lua
 	nvim -l nvim/review/resolve_test.lua
 	nvim -l nvim/review/menu_test.lua
 	nvim -l nvim/review/reconcile_test.lua
 	nvim -l nvim/review/gate_test.lua
 
 # Integration test for nvim/init.lua's send/queue/history state machine.
 # Drives the real init.lua under headless nvim (it can't be unit-tested —
 # monolithic config, all-local fns). Guards the send-from-queue duplication
 # regression. See tests/queue-send-test.sh.
 test-queue:
 	bash tests/queue-send-test.sh
 
 # Diagnostic trace for pair-originated zellij action calls (#68).
 test-zellij-trace:
 	bash tests/zellij-trace-test.sh
 
 # Regression test for the autopair next-char gate (nvim/init.lua,
 # pair_insert_open). Drives the real init.lua headlessly and asserts each
 # opener's expr output. See tests/autopair-test.sh.
 test-autopair:
 	bash tests/autopair-test.sh
 
+# Regression test for the draft typeahead completer mode guard (nvim/init.lua,
+# run_completers). Drives the real init.lua headlessly and asserts the shared
+# runner no-ops outside Insert mode before any vim.fn.complete() call can raise
+# E785. See tests/draft-complete-mode-test.sh.
+test-draft-complete:
+	bash tests/draft-complete-mode-test.sh
+
 # Regression test for the insert-mode <CR> decision (nvim/init.lua, cr_keys /
 # _G.PairCRKeys, #65): Return must always insert a newline when no completion
 # is selected, dismissing the popup. Asserts the expr output for all three
 # popup states. See tests/cr-newline-test.sh.
 test-cr:
 	bash tests/cr-newline-test.sh
 
 # Regression tests for the draft statusline (nvim/init.lua, _G.PairStatusline).
 # Drive the real init.lua headlessly: statusline-pos asserts the *, -N, +N nav
 # marker; changelog-notify asserts the ephemeral "change log ready" flash + its
 # fs_event marker watcher (#58).
 test-statusline:
 	bash tests/statusline-pos-test.sh
 	bash tests/changelog-notify-test.sh
 
 # Regression test for doctor/doctor.sh (the adaptation flight-recorder reader):
 # a malformed line must not abort the diagnostic. See doctor/doctor_test.sh.
 test-doctor:
 	bash doctor/doctor_test.sh
 
 # copy-on-select in_nvim detection: the agent overwrites its pane title with
 # "claude [<cwd>]", so a repo path containing "nvim" (e.g. parley.nvim) must not
 # be mistaken for the nvim draft pane (which would skip the paste). Keys on
 # terminal_command, not the cwd-polluted title. See tests/copy-on-select-test.sh.
 # copy-on-select is the Go binary (#93 M4; zellij's copy_command invokes it
 # directly since #94 M2), so a fresh-tree run must build it before the test copies
 # it into the sandboxed $PAIR_HOME/bin (same missing-prereq lesson as M2/M3).
 test-copy-on-select: $(BIN_DIR)/pair
 	bash tests/copy-on-select-test.sh
 
 # Cross-emitter contract for the flight recorder: Go == shell == Lua schema,
 # plus concurrent-append line-atomicity. See tests/adapt-schema-test.sh.
 test-adapt-schema:
 	bash tests/adapt-schema-test.sh
 
 # pair-dev rebuild hook (#000046): deployed mode (PAIR_DEV unset) must invoke
 # no toolchain; dev mode builds; build failure is errexit-safe. See
 # tests/dev-rebuild-test.sh.
 test-dev-rebuild:
 	bash tests/dev-rebuild-test.sh
 
 # Go public entrypoint install layout (#79): installed pair is a regular Go
 # binary, pair-go launch shares its pair-shell handoff, and copied local
 # installs fall back to the build-time source root for adjacent assets.
 test-pair-go-install-layout:
 	bash tests/pair-go-install-layout-test.sh
 
 test-pair-embedded-runtime:
 	bash tests/pair-embedded-runtime-test.sh
 
 # pair restart/quit Go subcommands (#94 M1, ported from bin/pair-{restart,quit}.sh):
 # markers land in ~/.cache/pair with PAIR_KILL_CMD stubbing the terminal kill.
 test-pair-restart-quit: $(BIN_DIR)/pair
 	bash tests/pair-restart-quit-test.sh
 
 # pair-doctor emitter-health probe (#000047): marker check + binary selection
 # (running-via-pidfile vs PATH). See tests/emitter-health-test.sh.
 test-emitter-health:
 	bash tests/emitter-health-test.sh
 
 # changelog + scrollback openers — end-to-end smokes for the #93 M2 Go binaries
 # (bin/pair-changelog-open, bin/pair-scrollback-open): fake the scrollback capture
 # + model + nvim and assert clean → distill → write → open (changelog) and render
 # → open (scrollback). Depend on the binaries so `make test` builds + runs them
 # (the scripts SKIP if absent, which would silently no-op in CI without these).
 test-changelog: $(BIN_DIR)/pair
 	sh tests/changelog-open-test.sh
 	sh tests/changelog-session-key-test.sh
 	sh tests/scrollback-open-test.sh
 
 # All former helper binaries are gone (#104 M3) — each is a `pair <subcommand>`
 # built into the single `pair` below. Their logic still lives in cmd/internal/*.
 
 # Staleness hint for the pair recipe (NOT the compile set — `go build` pulls
 # every dep). A find over cmd/*.go avoids the drift the old hand-maintained list
 # risked; the runtime assets are a prereq so a bundle change rebuilds pair too.
 PAIR_GO_SRCS := $(shell find cmd -name '*.go' -not -name '*_test.go' 2>/dev/null) go.mod $(RUNTIMEBUNDLE_ASSETS) cmd/pair-go/main.go cmd/internal/entrypoint/asset_root.go cmd/internal/entrypoint/mode.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go cmd/internal/slugcmd/slugcmd.go cmd/internal/slugcmd/slug.go cmd/internal/wrapcmd/wrap.go cmd/internal/scribecmd/scribecmd.go cmd/internal/changelogcmd/changelogcmd.go cmd/internal/changelogcmd/distill.go cmd/internal/changelogcmd/prompt.go cmd/internal/continuationcmd/continuationcmd.go cmd/internal/continuationcmd/continuation.go cmd/internal/continuationcmd/git.go cmd/internal/sessionwatch/sessionwatch.go cmd/internal/sessionwatch/run.go cmd/internal/sessionwatch/runcli.go cmd/internal/sessionwatch/runtime.go cmd/internal/procutil/procutil.go cmd/internal/model/model.go cmd/internal/transcript/transcript.go cmd/internal/adapt/adapt.go cmd/internal/runtimebundle/manifest.go cmd/internal/runtimebundle/plan.go cmd/internal/runtimebundle/cleanup.go cmd/internal/runtimebundle/store.go cmd/internal/runtimebundle/embed.go $(RUNTIMEBUNDLE_ASSETS) go.mod
 
 # pair — the Go-owned entrypoint. It drives the native launcher in-process (#99
 # M5c — bin/pair-shell retired); source/Homebrew builds can inject defaultPairHome,
 # while copied binaries without adjacent assets use the embedded runtime fallback.
 # (PAIR_GO_SRCS is a staleness hint, not the compile set — `go build` pulls every
 # dep; it needn't list every cmd/internal/launcher file.)
 $(BIN_DIR)/pair: $(PAIR_GO_SRCS)
 	go build -ldflags "$(PAIR_HOME_LDFLAGS)" -o $@ ./cmd/pair-go
 
 # Busybox symlinks (pair-slug → pair): a real target each, so `build` restores a
 # deleted symlink even when bin/pair is already up-to-date (not only when the pair
 # recipe fires). `install` re-links them into $(INSTALL_DIR) separately.
 $(addprefix $(BIN_DIR)/,$(BUSYBOX_LINKS)): $(BIN_DIR)/pair
 	@ln -sf pair $@
 
 
 # Tart VM targets (tart, tart-gui, tart-mount, tart-stop, tart-clean)
 # live in .tart/Makefile, vendored from ariadne and auto-included by
 # Makefile.workflow. `make help-tart` for usage.
diff --git a/nvim/init.lua b/nvim/init.lua
index b8930b0..b2515de 100644
--- a/nvim/init.lua
+++ b/nvim/init.lua
@@ -3737,213 +3737,226 @@ local function pair_insert_open(open)
   -- so a single `u` undoes the whole opener+closer insertion.
   return open .. close .. '<C-G>U<Left>'
 end
 
 local function pair_insert_close(close)
   local _, next = pair_cursor_chars()
   if next == close then return '<C-G>U<Right>' end
   return close
 end
 
 local function pair_backspace()
   local prev, next = pair_cursor_chars()
   if prev ~= '' and PAIR_OPEN_TO_CLOSE[prev] == next then
     return '<BS><Del>'
   end
   return '<BS>'
 end
 
 for open, close in pairs(PAIR_OPEN_TO_CLOSE) do
   vim.keymap.set('i', open, function() return pair_insert_open(open) end,
     { silent = true, expr = true, desc = 'pair: autopair ' .. open })
   -- Closers for non-quote pairs only — quote keys map to the opener handler
   -- above, which already does the jump-over check.
   if not PAIR_QUOTES[open] then
     vim.keymap.set('i', close, function() return pair_insert_close(close) end,
       { silent = true, expr = true, desc = 'pair: jump over ' .. close })
   end
 end
 vim.keymap.set('i', '<BS>', pair_backspace,
   { silent = true, expr = true, desc = 'pair: smart-delete empty pair' })
 
 -- <M-1>..<M-9>: quick-pick the Nth visible completion item from any popup.
 -- The as-you-type menus (path/word/spell) are unlabelled now (uniform, no
 -- numbering), so this is an unadvertised power-key — and it also swallows Alt+N
 -- so the terminal's Esc+N sequence can't break insert mode. Outside the popup
 -- these keys are no-ops — returning '' from an expr keymap leaves the buffer
 -- unchanged. (The z= popup advertises its own bare-digit picking separately.)
 --
 -- Mechanism: feed `<C-n>` / `<C-p>` to land selection on item N, then
 -- `<C-y>` to accept. We don't replace text manually; vim's accept handler
 -- already knows the span passed to complete() and substitutes correctly.
 local function pair_pick_completion(n)
   if vim.fn.pumvisible() == 0 then return '' end
   local info = vim.fn.complete_info({ 'items', 'selected' })
   if not info.items or not info.items[n] then return '' end
   local cn = vim.api.nvim_replace_termcodes('<C-n>', true, false, true)
   local cp = vim.api.nvim_replace_termcodes('<C-p>', true, false, true)
   local cy = vim.api.nvim_replace_termcodes('<C-y>', true, false, true)
   -- selected is 0-indexed (-1 = nothing selected, the noselect default).
   -- Treat -1 as "before item 0" so steps from -1 to target=0 is +1.
   local current = info.selected
   local target = n - 1
   local steps = (current >= 0) and (target - current) or (target + 1)
   local keys = ''
   if steps > 0 then keys = string.rep(cn, steps)
   elseif steps < 0 then keys = string.rep(cp, -steps) end
   return keys .. cy
 end
 
 for i = 1, 9 do
   vim.keymap.set('i', '<M-' .. i .. '>',
     function() return pair_pick_completion(i) end,
     { silent = true, expr = true, desc = 'pair: pick completion item ' .. i })
 end
 
 -- Bare digits pick a spell suggestion — but ONLY while the z= spell popup is
 -- up (spell_popup_active + a visible menu). Everywhere else a digit is just a
 -- digit, so the expr returns the literal key. CompleteDone then drops us back
 -- to normal mode (the popup's <C-y> accept fires it).
 local function spell_pick_digit(n)
   if spell_popup_active and vim.fn.pumvisible() == 1 then
     return pair_pick_completion(n)
   end
   return tostring(n)
 end
 
 for i = 1, 9 do
   vim.keymap.set('i', tostring(i),
     function() return spell_pick_digit(i) end,
     { silent = true, expr = true, desc = 'pair: pick spell suggestion ' .. i })
 end
 
 -- Fire on both events: TextChangedI when popup is hidden, TextChangedP when
 -- popup is visible — refreshing the menu as the user types more characters.
 -- path_complete handles slash/tilde tokens; word_complete kicks in for plain
 -- alphanumeric tokens >= 6 chars. Their token regexes are mutually exclusive
 -- (path needs `/` or `~`, word excludes both), so at most one calls complete().
 -- spell_complete is the fallback: only when both decline does it offer
 -- spelling fixes for a misspelled word-in-progress. Each completer returns
 -- true once it calls complete(), so run_completers short-circuits and at most
 -- one menu is built per keystroke.
 --
 -- Burst-debounce: paste dumps hundreds of TextChangedI events within a few
 -- ms; running both completion handlers (each scans the buffer + agent file)
 -- on every char stalls nvim and looks like flaky paste. Detection threshold
 -- is 20ms — well above human typing cadence (worst-case ~40ms at 200wpm),
 -- well below paste IO (~1-2ms). Bursts get coalesced into a single deferred
 -- run 30ms after the last event.
 local complete_last_fire = 0
 local complete_pending = nil
+function _G.PairDraftCompletionCanRun()
+  local mode = vim.api.nvim_get_mode().mode or ''
+  return mode:sub(1, 1) == 'i'
+end
+
 local function run_completers()
+  if not _G.PairDraftCompletionCanRun() then return end
   -- The explicit z= gesture (spell_suggest_popup) owns the popup while it's
   -- active, so the as-you-type completers must stay out of its way. Without
   -- this guard, startinsert's TextChangedI drives spell_complete to pop its own
   -- menu before z='s scheduled complete() runs; z= then replaces that menu, and
   -- the replacement's CompleteDone is misread by the teardown as "popup
   -- dismissed" → a scheduled stopinsert closes the just-opened z= menu (the
   -- first-z= menu flash).
   if spell_popup_active then return end
   if path_complete() then return end
   if word_complete() then return end
   spell_complete()
 end
+-- Exposed for tests/draft-complete-mode-test.sh. The live path reaches this
+-- runner through TextChangedI/P, but the regression drives the runner directly
+-- from Normal mode to keep the E785 failure deterministic under headless nvim.
+_G.PairDraftCompleteTest = {
+  can_run = _G.PairDraftCompletionCanRun,
+  run_completers = run_completers,
+}
 vim.api.nvim_create_autocmd({ 'TextChangedI', 'TextChangedP' }, {
   group = pair_aug,
   callback = function()
     local now = vim.loop.now()
     local delta = now - complete_last_fire
     complete_last_fire = now
     if complete_pending then
       complete_pending:stop()
       complete_pending:close()
       complete_pending = nil
     end
     if delta < 20 then
       complete_pending = vim.loop.new_timer()
       complete_pending:start(30, 0, vim.schedule_wrap(function()
         if complete_pending then
           complete_pending:close(); complete_pending = nil
         end
         run_completers()
       end))
       return
     end
     run_completers()
   end,
 })
 
 vim.keymap.set('n', 'z=', spell_suggest_popup,
   { silent = true, desc = 'pair: spell suggestions in completion popup' })
 
 -- Track which words the user accepts from the completion popup. The pick
 -- count feeds back into word_complete's ranking (PICK_WEIGHT). Fires for
 -- every completion (path/word/spell); only word_complete consults picks,
 -- but path/spell picks don't hurt — they just sit unused in the file.
 -- v.completed_item is `{}` on cancel, so the empty-word guard handles it.
 vim.api.nvim_create_autocmd('CompleteDone', {
   group = pair_aug,
   callback = function()
     local item = vim.v.completed_item
     if item and type(item) == 'table' and item.word then
       picks_bump(item.word)
     end
     -- z= forced insert mode purely to host the popup; once the suggestion is
     -- accepted (or dismissed) hand the user back the normal mode they came
     -- from. stopinsert can't run mid-CompleteDone, so defer it.
     if spell_popup_active then
       spell_popup_active = false
       vim.schedule(function() vim.cmd('stopinsert') end)
     end
   end,
 })
 
 -- Safety net: if the spell popup is torn down without a CompleteDone (e.g.
 -- the popup never materialized), make sure the flag never survives into a
 -- later, ordinary insert session where bare digits must stay literal.
 vim.api.nvim_create_autocmd('InsertLeave', {
   group = pair_aug,
   callback = function() spell_popup_active = false end,
 })
 
 -- "Ghost cursor" while the nvim pane is unfocused. zellij hides the real
 -- terminal cursor on FocusLost, leaving the insertion point invisible.
 -- Mark the position with a glyph chosen by mode so the indicator mirrors
 -- the focused-state cursor:
 --   normal-mode unfocused : ▯ (outline of █, the focused block cursor)
 --   insert-mode unfocused : ¦ (broken version of |, the focused bar cursor)
 local pair_focus_ns = vim.api.nvim_create_namespace('pair_focus_cursor')
 
 local function pair_apply_focus_cursor_hl()
   -- Tie to `Comment` so the glyph picks up the colorscheme's dimmed-text
   -- color — visible but subdued. Reapplied on ColorScheme since :hi clear
   -- (which colorschemes implicitly run) blows highlights away.
   vim.api.nvim_set_hl(0, 'PairFocusCursor', { link = 'Comment' })
 end
 pair_apply_focus_cursor_hl()
 
 local function pair_show_focus_cursor()
   local mode  = vim.api.nvim_get_mode().mode:sub(1, 1)
   local glyph = (mode == 'i') and '¦' or '▯'
   local row1, col = unpack(vim.api.nvim_win_get_cursor(0))
   local row = row1 - 1
   vim.api.nvim_buf_clear_namespace(0, pair_focus_ns, 0, -1)
   pcall(vim.api.nvim_buf_set_extmark, 0, pair_focus_ns, row, col, {
     virt_text     = { { glyph, 'PairFocusCursor' } },
     virt_text_pos = 'overlay',
     priority      = 200,
   })
 end
 
 local function pair_hide_focus_cursor()
   vim.api.nvim_buf_clear_namespace(0, pair_focus_ns, 0, -1)
 end
 
 vim.api.nvim_create_autocmd('FocusLost', {
   group = pair_aug,
   callback = function()
     pair_show_focus_cursor()
     -- A delayed full redraw catches the case where zellij's focus-change
     -- rendering fires after our immediate refresh_statusline (which only
     -- defers one event-loop tick). 80ms is comfortably above one terminal
     -- frame and unobtrusive.
     vim.defer_fn(function() pcall(vim.cmd, 'redraw!') end, 80)
diff --git a/tests/draft-complete-mode-test.sh b/tests/draft-complete-mode-test.sh
new file mode 100644
index 0000000..e87c5d6
--- /dev/null
+++ b/tests/draft-complete-mode-test.sh
@@ -0,0 +1,77 @@
+#!/usr/bin/env bash
+# Regression test for draft typeahead completion mode safety (nvim/init.lua).
+#
+# TextChangedI/P can debounce completion through a scheduled callback. If the
+# user leaves Insert mode before that callback runs, the shared completion
+# runner must no-op; otherwise word_complete/path_complete/spell_complete can
+# call vim.fn.complete() from Normal/Visual mode and raise E785.
+#
+# Run: bash tests/draft-complete-mode-test.sh   (also wired into `make test`)
+set -uo pipefail
+
+ROOT="$(cd "$(dirname "$0")/.." && pwd)"
+INIT="$ROOT/nvim/init.lua"
+. "$ROOT/tests/lib/run-headless.sh"   # run_headless: timeout watchdog (#60)
+RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-draft-complete-test.XXXXXX")"
+trap 'rm -rf "$RT"' EXIT
+
+printf '' > "$RT/draft.md"
+
+cat > "$RT/driver.lua" <<'LUA'
+local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
+local fails = 0
+local function check(cond, msg, detail)
+  if cond then
+    O:write('ok\t' .. msg .. '\n')
+  else
+    fails = fails + 1
+    O:write('FAIL\t' .. msg .. '\t' .. tostring(detail or '') .. '\n')
+  end
+end
+
+check(type(_G.PairDraftCompleteTest) == 'table', 'test seam exported')
+check(type(_G.PairDraftCompleteTest.run_completers) == 'function', 'runner exported')
+
+-- Prefix "complet" has a same-buffer candidate "completion", so without the
+-- shared mode guard run_completers reaches word_complete() and vim.fn.complete().
+vim.api.nvim_buf_set_lines(0, 0, -1, false, { 'complet completion' })
+vim.api.nvim_win_set_cursor(0, { 1, #'complet' })
+vim.cmd('stopinsert')
+local mode = vim.api.nvim_get_mode().mode
+check(mode:sub(1, 1) ~= 'i', 'driver is outside insert mode', mode)
+
+local ok, err = pcall(_G.PairDraftCompleteTest.run_completers)
+check(ok, 'runner skips outside insert mode without E785', err)
+
+O:write('TOTAL_FAILS=' .. fails .. '\n')
+O:close()
+vim.cmd('qall!')
+LUA
+
+if ! run_headless --timeout 30 -- \
+  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
+  nvim --headless -u "$INIT" "$RT/draft.md" \
+  -c "luafile $RT/driver.lua"; then
+  echo "draft-complete-mode-test: nvim driver failed"
+  exit 1
+fi
+
+echo "draft-complete-mode-test:"
+fails=0
+if [ ! -f "$RT/result.txt" ]; then
+  echo "  FAIL driver produced no result (nvim boot/driver error)"
+  exit 1
+fi
+while IFS=$'\t' read -r status label detail; do
+  case "$status" in
+    ok)   printf '  ok   %s\n' "$label" ;;
+    FAIL) printf '  FAIL %s: %s\n' "$label" "$detail"; fails=$((fails + 1)) ;;
+    TOTAL_FAILS=*) ;;
+  esac
+done < "$RT/result.txt"
+
+if [ "$fails" -ne 0 ]; then
+  echo "draft-complete-mode-test: $fails failure(s)"
+  exit 1
+fi
+echo "draft-complete-mode-test: all passed"
diff --git a/workshop/issues/000114-draft-complete-insert-mode.md b/workshop/issues/000114-draft-complete-insert-mode.md
index b930de3..404cf8f 100644
--- a/workshop/issues/000114-draft-complete-insert-mode.md
+++ b/workshop/issues/000114-draft-complete-insert-mode.md
@@ -1,67 +1,77 @@
 ---
 id: 000114
 status: working
 deps: []
 github_issue:
 created: 2026-07-10
 updated: 2026-07-10
 estimate_hours: 0.45
 started: 2026-07-10T07:54:48-07:00
 ---
 
 # draft completion should skip outside insert mode
 
 ## Problem
 
 Draft-pane typeahead completion is scheduled from `TextChangedI/P`. If the user
 leaves Insert mode before a debounced callback runs, the callback can still call
 `vim.fn.complete()`. Neovim raises `E785: complete() can only be used in Insert
 mode`, currently observed when selecting a word around spell-check/completion
 flows.
 
 ## Spec
 
 - The draft completion runner must no-op unless the current mode is Insert or
   completion-select mode.
 - Delayed/debounced callbacks must re-check mode at execution time, not only at
   scheduling time.
 - Existing path, word, and spell completion behavior in Insert mode is
   preserved.
 
 ARCH-PURPOSE: the fix must prevent every `run_completers()` path from reaching
 `complete()` outside Insert mode, not only one completer.
 ARCH-DRY: put the guard at the shared completion runner, not separately in
 `path_complete`, `word_complete`, and `spell_complete`.
 
 ## Done when
 
 - A regression simulates the draft completer running in Normal/Visual mode and
   proves it does not call `complete()` or raise `E785`.
 - Existing nvim headless tests pass for the draft completion area.
 
 ## Estimate
 
 ```estimate
 model: estimate-logic-v3.1
 familiarity: 0.9
 item: lua-neovim design=0.10 impl=0.20
 item: milestone-review design=0.00 impl=0.15
 total: 0.45
 ```
 
 ## Plan
 
-- [ ] Expose or isolate the draft completer mode guard enough for a headless
+- [x] Expose or isolate the draft completer mode guard enough for a headless
       regression.
-- [ ] Add a failing regression for a scheduled completer execution outside
+- [x] Add a failing regression for a scheduled completer execution outside
       Insert mode.
-- [ ] Guard the shared completion runner before any completer can call
+- [x] Guard the shared completion runner before any completer can call
       `vim.fn.complete()`.
-- [ ] Run focused nvim tests and close #114.
+- [x] Run focused nvim tests and close #114.
 
 ## Log
 
 ### 2026-07-10
 - Created from reported stack trace:
   `Vim:E785: complete() can only be used in Insert mode` from
   `word_complete()` via scheduled `run_completers()`.
+- Added `tests/draft-complete-mode-test.sh`; before the guard it failed with
+  `Vim:E785: complete() can only be used in Insert mode`, reproducing the
+  reported error through the real `nvim/init.lua` runner.
+- Fixed the shared draft completer runner to no-op unless Neovim reports an
+  Insert-mode variant, covering path, word, and spell typeahead from the single
+  `run_completers()` entry point (`ARCH-DRY`, `ARCH-PURPOSE`).
+- Verification passed: `bash tests/draft-complete-mode-test.sh`,
+  `bash tests/autopair-test.sh`, `bash tests/cr-newline-test.sh`,
+  `make test-lua`, `make test-draft-complete`, `git diff --check`, and
+  `make test`.
nvim/init.lua:1672:local function path_complete()
nvim/init.lua:1845:local function word_complete()
nvim/init.lua:1960:-- As-you-type spell typeahead. Runs only as a fallback in run_completers,
nvim/init.lua:1977:local function spell_complete()
nvim/init.lua:3826:-- true once it calls complete(), so run_completers short-circuits and at most
nvim/init.lua:3836:local complete_pending = nil
nvim/init.lua:3837:function _G.PairDraftCompletionCanRun()
nvim/init.lua:3842:local function run_completers()
nvim/init.lua:3843:  if not _G.PairDraftCompletionCanRun() then return end
nvim/init.lua:3859:_G.PairDraftCompleteTest = {
nvim/init.lua:3860:  can_run = _G.PairDraftCompletionCanRun,
nvim/init.lua:3861:  run_completers = run_completers,
nvim/init.lua:3869:    if complete_pending then
nvim/init.lua:3870:      complete_pending:stop()
nvim/init.lua:3871:      complete_pending:close()
nvim/init.lua:3872:      complete_pending = nil
nvim/init.lua:3875:      complete_pending = vim.loop.new_timer()
nvim/init.lua:3876:      complete_pending:start(30, 0, vim.schedule_wrap(function()
nvim/init.lua:3877:        if complete_pending then
nvim/init.lua:3878:          complete_pending:close(); complete_pending = nil
nvim/init.lua:3880:        run_completers()
nvim/init.lua:3884:    run_completers()

codex
The implementation places the guard at the correct shared execution point, including the deferred callback path. I’m now checking whether the regression genuinely proves the promised mode matrix and whether it would fail against the pre-fix code, rather than merely passing on the new code.
exec
/bin/zsh -lc "sed -n '1640,2015p' nvim/init.lua && rg -n \"path_complete|word_complete|spell_complete|TextChangedI|TextChangedP|complete\\(\" tests nvim/*_test.lua README.md atlas --glob '"'!tests/draft-complete-mode-test.sh'"' && nvim --version | head -5 && nvim --headless -u NONE '+lua print(vim.inspect(vim.api.nvim_get_mode()))' '+qall' && bash tests/draft-complete-mode-test.sh && bash tests/autopair-test.sh && bash tests/cr-newline-test.sh && make test-lua" in /Users/xianxu/workspace/pair
 succeeded in 3501ms:
draft-complete-mode-test:
  ok   test seam exported
  ok   runner exported
  ok   driver is outside insert mode
  ok   runner skips outside insert mode without E785
draft-complete-mode-test: all passed
autopair-test:
  ok   paren EOL
  ok   bracket EOL
  ok   brace EOL
  ok   paren pre-space
  ok   paren pre-word
  ok   bracket pre-word
  ok   paren pre-close
  ok   dquote EOL
  ok   backtick EOL
  ok   dquote pre-word
  ok   apostrophe dont
  ok   dquote jump-over
autopair-test: all passed
cr-newline-test:
  ok   no popup -> newline
  ok   no popup (sel irrelevant)
  ok   typing: popup + selection -> accept
  ok   typing: nothing picked -> C-e + newline
  ok   z=: nothing picked -> clean dismiss, no newline
  ok   z=: popup + selection -> accept
  ok   live <CR> map routes through cr_keys (no popup)
cr-newline-test: all passed
nvim -l nvim/slug_test.lua
nvim/slug.lua: all tests passed
nvim -l nvim/scrollback_test.lua
scrollback refresh failed: boom
scrollback refresh skipped buffer reload: pending annotations kept
nvim/scrollback.lua: prompt pattern tests passed
nvim -l nvim/annotate_test.lua
ok annotate_test
nvim -l nvim/changelog_test.lua
ok changelog_test
nvim -l nvim/adapt_test.lua
nvim/adapt.lua: emitter tests passed
nvim -l nvim/doctor_test.lua
all doctor.lua tests passed
nvim -l nvim/review/record_test.lua
record_test ok
nvim -l nvim/review/reconstruct_test.lua
reconstruct_test ok
nvim -l nvim/review/define_test.lua
define_test ok
nvim -l nvim/review/markers_test.lua
markers_test ok
nvim -l nvim/review/seam_test.lua
seam_test ok
nvim -l nvim/review/mode_test.lua
mode_test ok
nvim -l nvim/review/poke_bodies_test.lua
poke_bodies_test ok
nvim -l nvim/review/wrap_test.lua
wrap_test ok
nvim -l nvim/review/readiness_test.lua
readiness_test ok
nvim -l nvim/review/spinner_test.lua
spinner_test ok
nvim -l nvim/review/resolve_test.lua
resolve_test ok
nvim -l nvim/review/menu_test.lua
menu_test ok
nvim -l nvim/review/reconcile_test.lua
reconcile_test ok
nvim -l nvim/review/gate_test.lua
gate_test ok

exec
/bin/zsh -lc 'RT=$(mktemp -d /tmp/pair-review-114.XXXXXX)
cp nvim/init.lua "$RT/init.lua"
sed -i '"'' '/if not _G.PairDraftCompletionCanRun() then return end/d' \""'$RT/init.lua"
sed "s#INIT='"\\\"\\"'$ROOT/nvim/init.lua'"\\\"#INIT=\\\""'$RT/init.lua'"\\\"#\" tests/draft-complete-mode-test.sh > \""'$RT/test.sh"
bash "$RT/test.sh"
rm -rf "$RT"
nl -ba nvim/init.lua | sed -n '"'3828,3890p'
nl -ba tests/draft-complete-mode-test.sh | sed -n '20,77p'
nl -ba workshop/issues/000114-draft-complete-insert-mode.md | sed -n '15,80p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
/tmp/pair-review-114.JTqKMC/test.sh: line 14: /tmp/tests/lib/run-headless.sh: No such file or directory
/tmp/pair-review-114.JTqKMC/test.sh: line 51: run_headless: command not found
draft-complete-mode-test: nvim driver failed
  3828	--
  3829	-- Burst-debounce: paste dumps hundreds of TextChangedI events within a few
  3830	-- ms; running both completion handlers (each scans the buffer + agent file)
  3831	-- on every char stalls nvim and looks like flaky paste. Detection threshold
  3832	-- is 20ms — well above human typing cadence (worst-case ~40ms at 200wpm),
  3833	-- well below paste IO (~1-2ms). Bursts get coalesced into a single deferred
  3834	-- run 30ms after the last event.
  3835	local complete_last_fire = 0
  3836	local complete_pending = nil
  3837	function _G.PairDraftCompletionCanRun()
  3838	  local mode = vim.api.nvim_get_mode().mode or ''
  3839	  return mode:sub(1, 1) == 'i'
  3840	end
  3841	
  3842	local function run_completers()
  3843	  if not _G.PairDraftCompletionCanRun() then return end
  3844	  -- The explicit z= gesture (spell_suggest_popup) owns the popup while it's
  3845	  -- active, so the as-you-type completers must stay out of its way. Without
  3846	  -- this guard, startinsert's TextChangedI drives spell_complete to pop its own
  3847	  -- menu before z='s scheduled complete() runs; z= then replaces that menu, and
  3848	  -- the replacement's CompleteDone is misread by the teardown as "popup
  3849	  -- dismissed" → a scheduled stopinsert closes the just-opened z= menu (the
  3850	  -- first-z= menu flash).
  3851	  if spell_popup_active then return end
  3852	  if path_complete() then return end
  3853	  if word_complete() then return end
  3854	  spell_complete()
  3855	end
  3856	-- Exposed for tests/draft-complete-mode-test.sh. The live path reaches this
  3857	-- runner through TextChangedI/P, but the regression drives the runner directly
  3858	-- from Normal mode to keep the E785 failure deterministic under headless nvim.
  3859	_G.PairDraftCompleteTest = {
  3860	  can_run = _G.PairDraftCompletionCanRun,
  3861	  run_completers = run_completers,
  3862	}
  3863	vim.api.nvim_create_autocmd({ 'TextChangedI', 'TextChangedP' }, {
  3864	  group = pair_aug,
  3865	  callback = function()
  3866	    local now = vim.loop.now()
  3867	    local delta = now - complete_last_fire
  3868	    complete_last_fire = now
  3869	    if complete_pending then
  3870	      complete_pending:stop()
  3871	      complete_pending:close()
  3872	      complete_pending = nil
  3873	    end
  3874	    if delta < 20 then
  3875	      complete_pending = vim.loop.new_timer()
  3876	      complete_pending:start(30, 0, vim.schedule_wrap(function()
  3877	        if complete_pending then
  3878	          complete_pending:close(); complete_pending = nil
  3879	        end
  3880	        run_completers()
  3881	      end))
  3882	      return
  3883	    end
  3884	    run_completers()
  3885	  end,
  3886	})
  3887	
  3888	vim.keymap.set('n', 'z=', spell_suggest_popup,
  3889	  { silent = true, desc = 'pair: spell suggestions in completion popup' })
  3890	
    20	cat > "$RT/driver.lua" <<'LUA'
    21	local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
    22	local fails = 0
    23	local function check(cond, msg, detail)
    24	  if cond then
    25	    O:write('ok\t' .. msg .. '\n')
    26	  else
    27	    fails = fails + 1
    28	    O:write('FAIL\t' .. msg .. '\t' .. tostring(detail or '') .. '\n')
    29	  end
    30	end
    31	
    32	check(type(_G.PairDraftCompleteTest) == 'table', 'test seam exported')
    33	check(type(_G.PairDraftCompleteTest.run_completers) == 'function', 'runner exported')
    34	
    35	-- Prefix "complet" has a same-buffer candidate "completion", so without the
    36	-- shared mode guard run_completers reaches word_complete() and vim.fn.complete().
    37	vim.api.nvim_buf_set_lines(0, 0, -1, false, { 'complet completion' })
    38	vim.api.nvim_win_set_cursor(0, { 1, #'complet' })
    39	vim.cmd('stopinsert')
    40	local mode = vim.api.nvim_get_mode().mode
    41	check(mode:sub(1, 1) ~= 'i', 'driver is outside insert mode', mode)
    42	
    43	local ok, err = pcall(_G.PairDraftCompleteTest.run_completers)
    44	check(ok, 'runner skips outside insert mode without E785', err)
    45	
    46	O:write('TOTAL_FAILS=' .. fails .. '\n')
    47	O:close()
    48	vim.cmd('qall!')
    49	LUA
    50	
    51	if ! run_headless --timeout 30 -- \
    52	  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
    53	  nvim --headless -u "$INIT" "$RT/draft.md" \
    54	  -c "luafile $RT/driver.lua"; then
    55	  echo "draft-complete-mode-test: nvim driver failed"
    56	  exit 1
    57	fi
    58	
    59	echo "draft-complete-mode-test:"
    60	fails=0
    61	if [ ! -f "$RT/result.txt" ]; then
    62	  echo "  FAIL driver produced no result (nvim boot/driver error)"
    63	  exit 1
    64	fi
    65	while IFS=$'\t' read -r status label detail; do
    66	  case "$status" in
    67	    ok)   printf '  ok   %s\n' "$label" ;;
    68	    FAIL) printf '  FAIL %s: %s\n' "$label" "$detail"; fails=$((fails + 1)) ;;
    69	    TOTAL_FAILS=*) ;;
    70	  esac
    71	done < "$RT/result.txt"
    72	
    73	if [ "$fails" -ne 0 ]; then
    74	  echo "draft-complete-mode-test: $fails failure(s)"
    75	  exit 1
    76	fi
    77	echo "draft-complete-mode-test: all passed"
    15	
    16	Draft-pane typeahead completion is scheduled from `TextChangedI/P`. If the user
    17	leaves Insert mode before a debounced callback runs, the callback can still call
    18	`vim.fn.complete()`. Neovim raises `E785: complete() can only be used in Insert
    19	mode`, currently observed when selecting a word around spell-check/completion
    20	flows.
    21	
    22	## Spec
    23	
    24	- The draft completion runner must no-op unless the current mode is Insert or
    25	  completion-select mode.
    26	- Delayed/debounced callbacks must re-check mode at execution time, not only at
    27	  scheduling time.
    28	- Existing path, word, and spell completion behavior in Insert mode is
    29	  preserved.
    30	
    31	ARCH-PURPOSE: the fix must prevent every `run_completers()` path from reaching
    32	`complete()` outside Insert mode, not only one completer.
    33	ARCH-DRY: put the guard at the shared completion runner, not separately in
    34	`path_complete`, `word_complete`, and `spell_complete`.
    35	
    36	## Done when
    37	
    38	- A regression simulates the draft completer running in Normal/Visual mode and
    39	  proves it does not call `complete()` or raise `E785`.
    40	- Existing nvim headless tests pass for the draft completion area.
    41	
    42	## Estimate
    43	
    44	```estimate
    45	model: estimate-logic-v3.1
    46	familiarity: 0.9
    47	item: lua-neovim design=0.10 impl=0.20
    48	item: milestone-review design=0.00 impl=0.15
    49	total: 0.45
    50	```
    51	
    52	## Plan
    53	
    54	- [x] Expose or isolate the draft completer mode guard enough for a headless
    55	      regression.
    56	- [x] Add a failing regression for a scheduled completer execution outside
    57	      Insert mode.
    58	- [x] Guard the shared completion runner before any completer can call
    59	      `vim.fn.complete()`.
    60	- [x] Run focused nvim tests and close #114.
    61	
    62	## Log
    63	
    64	### 2026-07-10
    65	- Created from reported stack trace:
    66	  `Vim:E785: complete() can only be used in Insert mode` from
    67	  `word_complete()` via scheduled `run_completers()`.
    68	- Added `tests/draft-complete-mode-test.sh`; before the guard it failed with
    69	  `Vim:E785: complete() can only be used in Insert mode`, reproducing the
    70	  reported error through the real `nvim/init.lua` runner.
    71	- Fixed the shared draft completer runner to no-op unless Neovim reports an
    72	  Insert-mode variant, covering path, word, and spell typeahead from the single
    73	  `run_completers()` entry point (`ARCH-DRY`, `ARCH-PURPOSE`).
    74	- Verification passed: `bash tests/draft-complete-mode-test.sh`,
    75	  `bash tests/autopair-test.sh`, `bash tests/cr-newline-test.sh`,
    76	  `make test-lua`, `make test-draft-complete`, `git diff --check`, and
    77	  `make test`.

codex
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The shared execution-time guard correctly prevents deferred completion from reaching any completer outside Insert-mode variants. The implementation is small and appropriately placed, and focused tests pass. Before closing, cheaply align the regression with the documented Normal/Visual and scheduled-callback coverage; the current test exercises only a direct Normal-mode invocation.

1. Strengths

- [nvim/init.lua:3843](/Users/xianxu/workspace/pair/nvim/init.lua:3843) guards the shared runner before path, word, or spell completion, satisfying ARCH-DRY and preventing partial fixes.
- [nvim/init.lua:3880](/Users/xianxu/workspace/pair/nvim/init.lua:3880) routes the debounced callback through the guarded runner, so mode is checked at execution time.
- `mode:sub(1, 1) == 'i'` admits Neovim’s Insert-mode variants, including completion-related Insert modes.
- The regression is integrated into `make test`, uses the real `init.lua`, and has a timeout watchdog.
- Focused verification passed: draft-completion regression, autopair, CR behavior, all Lua tests, and `git diff --check`.

2. Critical findings

None.

3. Important findings

- [tests/draft-complete-mode-test.sh:39](/Users/xianxu/workspace/pair/tests/draft-complete-mode-test.sh:39): the regression only invokes the runner directly from Normal mode. It does not exercise Visual mode or a scheduled/debounced execution, although both are claimed by the Done-when and checked Plan at [workshop/issues/000114-draft-complete-insert-mode.md:38](/Users/xianxu/workspace/pair/workshop/issues/000114-draft-complete-insert-mode.md:38) and line 56. Add a Visual-mode case and drive a deferred callback after leaving Insert mode, or revise the Plan claim if direct invocation is intentionally the seam.
- [nvim/init.lua:3837](/Users/xianxu/workspace/pair/nvim/init.lua:3837): `PairDraftCompletionCanRun` needlessly exposes the production guard as a mutable global, and `run_completers` looks it up globally on every execution. Keep the function local and export only its reference through `_G.PairDraftCompleteTest`; this prevents unrelated configuration from replacing or clearing a production dependency.

4. Minor findings

None.

5. Test coverage notes

The current test meaningfully catches the reported Normal-mode `E785` path using a real buffer candidate. It does not explicitly demonstrate preserved Insert-mode completion, Visual-mode rejection, or the timer/autocmd transition that caused the original defect. Adding those cases would fully trace the Spec.

6. Architectural notes for upcoming work

- ARCH-DRY: pass — one guard covers the entire completer chain.
- ARCH-PURE: pass, with the global-surface caveat above — mode inspection remains a thin Neovim integration seam; no business logic was buried in IO.
- ARCH-PURPOSE: pass for runtime behavior; the guard covers both immediate and scheduled runner paths. Test traceability remains incomplete.
- No new user-facing command, keybinding, configuration, or architectural surface was introduced, so README and atlas updates are not required.

7. Plan revision recommendations

If the test remains direct-only, append a `## Revisions` entry stating that the regression validates execution-time safety through the shared runner seam rather than driving the debounce timer, and change the checked Plan wording accordingly. Otherwise, add the scheduled and Visual-mode cases and no revision is needed.
tokens used
45,190
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The shared execution-time guard correctly prevents deferred completion from reaching any completer outside Insert-mode variants. The implementation is small and appropriately placed, and focused tests pass. Before closing, cheaply align the regression with the documented Normal/Visual and scheduled-callback coverage; the current test exercises only a direct Normal-mode invocation.

1. Strengths

- [nvim/init.lua:3843](/Users/xianxu/workspace/pair/nvim/init.lua:3843) guards the shared runner before path, word, or spell completion, satisfying ARCH-DRY and preventing partial fixes.
- [nvim/init.lua:3880](/Users/xianxu/workspace/pair/nvim/init.lua:3880) routes the debounced callback through the guarded runner, so mode is checked at execution time.
- `mode:sub(1, 1) == 'i'` admits Neovim’s Insert-mode variants, including completion-related Insert modes.
- The regression is integrated into `make test`, uses the real `init.lua`, and has a timeout watchdog.
- Focused verification passed: draft-completion regression, autopair, CR behavior, all Lua tests, and `git diff --check`.

2. Critical findings

None.

3. Important findings

- [tests/draft-complete-mode-test.sh:39](/Users/xianxu/workspace/pair/tests/draft-complete-mode-test.sh:39): the regression only invokes the runner directly from Normal mode. It does not exercise Visual mode or a scheduled/debounced execution, although both are claimed by the Done-when and checked Plan at [workshop/issues/000114-draft-complete-insert-mode.md:38](/Users/xianxu/workspace/pair/workshop/issues/000114-draft-complete-insert-mode.md:38) and line 56. Add a Visual-mode case and drive a deferred callback after leaving Insert mode, or revise the Plan claim if direct invocation is intentionally the seam.
- [nvim/init.lua:3837](/Users/xianxu/workspace/pair/nvim/init.lua:3837): `PairDraftCompletionCanRun` needlessly exposes the production guard as a mutable global, and `run_completers` looks it up globally on every execution. Keep the function local and export only its reference through `_G.PairDraftCompleteTest`; this prevents unrelated configuration from replacing or clearing a production dependency.

4. Minor findings

None.

5. Test coverage notes

The current test meaningfully catches the reported Normal-mode `E785` path using a real buffer candidate. It does not explicitly demonstrate preserved Insert-mode completion, Visual-mode rejection, or the timer/autocmd transition that caused the original defect. Adding those cases would fully trace the Spec.

6. Architectural notes for upcoming work

- ARCH-DRY: pass — one guard covers the entire completer chain.
- ARCH-PURE: pass, with the global-surface caveat above — mode inspection remains a thin Neovim integration seam; no business logic was buried in IO.
- ARCH-PURPOSE: pass for runtime behavior; the guard covers both immediate and scheduled runner paths. Test traceability remains incomplete.
- No new user-facing command, keybinding, configuration, or architectural surface was introduced, so README and atlas updates are not required.

7. Plan revision recommendations

If the test remains direct-only, append a `## Revisions` entry stating that the regression validates execution-time safety through the shared runner seam rather than driving the debounce timer, and change the checked Plan wording accordingly. Otherwise, add the scheduled and Visual-mode cases and no revision is needed.
