# Boundary Review — ariadne#86 (whole-issue close)

| field | value |
|-------|-------|
| issue | 86 — Fix Alt Return draft submit |
| repo | pair |
| issue file | workshop/issues/000086-fix-alt-return-draft-submit.md |
| boundary | whole-issue close |
| milestone | — |
| window | 62e38b3b7cd38f2ba9e25bda4d9332180c13aa71..HEAD |
| command | sdlc close --issue 86 |
| reviewer | codex |
| timestamp | 2026-06-29T17:45:34-07:00 |
| verdict | FIX-THEN-SHIP |

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
session id: 019f15fb-b2f9-7a22-a2a6-05c58e682704
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary — a
milestone close, or a whole-issue close — for ariadne#86.

Base: 62e38b3b7cd38f2ba9e25bda4d9332180c13aa71   Head: HEAD

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
diff --git a/nvim/init.lua b/nvim/init.lua
index 1a88366..e4ff0da 100644
--- a/nvim/init.lua
+++ b/nvim/init.lua
@@ -699,6 +699,26 @@ local function send_esc_to_agent()
   PairZellijTrace.action('draft.interrupt.focus-draft', { 'zellij', 'action', 'move-focus', 'down' })
 end
 
+local function draftSendCommands(body, no_submit)
+  local cmds = {
+    { label = 'draft.send.focus-agent', argv = { 'zellij', 'action', 'move-focus', 'up' } },
+    {
+      label = 'draft.send.write-body',
+      argv = { 'zellij', 'action', 'write-chars', body },
+      opts = { redact = { [4] = body } },
+    },
+  }
+  if no_submit then
+    cmds[#cmds + 1] = { label = 'draft.send.newline', argv = { 'zellij', 'action', 'write', '13' } }
+  else
+    cmds[#cmds + 1] = { label = 'draft.send.submit', argv = { 'zellij', 'action', 'send-keys', 'Alt Enter' } }
+  end
+  cmds[#cmds + 1] = { label = 'draft.send.focus-draft', argv = { 'zellij', 'action', 'move-focus', 'down' } }
+  return cmds
+end
+
+_G.PairDraftSendCommands = draftSendCommands
+
 local function send_to_agent(body, no_submit)
   -- focus up to agent pane, type body, press Enter, focus back down.
   --
@@ -718,13 +738,12 @@ local function send_to_agent(body, no_submit)
   -- agent has time to ingest the paste and return to the input prompt
   -- before we hit submit. Single-line sends skip the wait.
   --
-  -- Submit is Alt+Enter (ESC 0x1b, then CR 0x0d), not plain Enter:
+  -- Submit is Alt+Enter, not plain Enter:
   -- pair-wrap's stdin translator rewrites incoming \r into the agent's
   -- "insert newline" sequence (claude: `\<Enter>`, codex/agy: \n),
-  -- so a bare CR here would insert a newline rather than submit.
-  -- Alt+Enter is what pair-wrap rewrites into the agent's actual
-  -- submit byte. Mirrors the keyboard convention we set up for the
-  -- user (Enter = newline, Alt+Enter = send).
+  -- so a bare CR here would insert a newline rather than submit. Use
+  -- zellij's semantic send-keys action for the modified chord instead
+  -- of synthesizing it as raw ESC+CR bytes.
   --
   -- no_submit (Alt+Shift+Enter path): land the body in the agent's
   -- composer followed by a literal newline but DON'T submit. A bare CR
@@ -733,19 +752,15 @@ local function send_to_agent(body, no_submit)
   -- *not* a submit — so it leaves the cursor on a fresh line in the
   -- composer, ready for more input.
   if not has_ui() then return end
-  PairZellijTrace.action('draft.send.focus-agent', { 'zellij', 'action', 'move-focus', 'up' })
-  PairZellijTrace.action('draft.send.write-body', { 'zellij', 'action', 'write-chars', body }, {
-    redact = { [4] = body },
-  })
+  local cmds = draftSendCommands(body, no_submit)
+  PairZellijTrace.action(cmds[1].label, cmds[1].argv, cmds[1].opts)
+  PairZellijTrace.action(cmds[2].label, cmds[2].argv, cmds[2].opts)
   if body:find('\n') or #body > 200 then
     vim.cmd('sleep 100m')
   end
-  if no_submit then
-    PairZellijTrace.action('draft.send.newline', { 'zellij', 'action', 'write', '13' })
-  else
-    PairZellijTrace.action('draft.send.submit', { 'zellij', 'action', 'write', '27', '13' })
+  for i = 3, #cmds do
+    PairZellijTrace.action(cmds[i].label, cmds[i].argv, cmds[i].opts)
   end
-  PairZellijTrace.action('draft.send.focus-draft', { 'zellij', 'action', 'move-focus', 'down' })
 end
 
 -- :PairReview <file> — PROPOSE a file for review (#66 M4a'). It does NOT open the
diff --git a/nvim/pair_poke.lua b/nvim/pair_poke.lua
index fb6a170..f825b85 100644
--- a/nvim/pair_poke.lua
+++ b/nvim/pair_poke.lua
@@ -10,12 +10,12 @@ local M = {}
 local pair_nvim_dir = vim.fn.fnamemodify(debug.getinfo(1, 'S').source:sub(2), ':p:h')
 local zellij_trace = dofile(pair_nvim_dir .. '/zellij_trace.lua')
 
--- Pure: the ordered zellij argv list for one poke. Submit is Alt+Enter
--- (write 27 13) — pair-wrap rewrites that into the agent's real submit byte.
+-- Pure: the ordered zellij argv list for one poke. Submit is a semantic
+-- Alt+Enter key event so zellij delivers one modified chord to pair-wrap.
 function M._cmds(body, agent_id, review_id)
   return {
     { 'zellij', 'action', 'write-chars', '--pane-id', tostring(agent_id), body },
-    { 'zellij', 'action', 'write', '--pane-id', tostring(agent_id), '27', '13' },
+    { 'zellij', 'action', 'send-keys', '--pane-id', tostring(agent_id), 'Alt Enter' },
   }
 end
 
diff --git a/tests/queue-send-test.sh b/tests/queue-send-test.sh
index 51ab2ef..c5be13e 100755
--- a/tests/queue-send-test.sh
+++ b/tests/queue-send-test.sh
@@ -71,6 +71,17 @@ for body in (lc..'\n---\n'):gmatch('## %S+ %S+\n\n(.-)\n\n%-%-%-') do
   O:write('L '..body..'\n')
 end
 local d=io.open(dd..'/draft-test.md'); O:write('D '..((d and d:read('*a') or ''):gsub('%s+\$',''))..'\n'); if d then d:close() end
+local function dump_cmds(prefix, no_submit)
+  if type(_G.PairDraftSendCommands) ~= 'function' then
+    O:write('C '..prefix..' missing\n')
+    return
+  end
+  for _,cmd in ipairs(_G.PairDraftSendCommands('HELLO', no_submit)) do
+    O:write('C '..prefix..' '..cmd.label..' '..table.concat(cmd.argv, ' ')..'\n')
+  end
+end
+dump_cmds('submit', false)
+dump_cmds('append', true)
 O:close(); vim.cmd('qall!')  -- force-quit the dirtied throwaway buffer (#60)
 LUA
 }
@@ -115,6 +126,17 @@ if has "L HELLO" && has "Q 500000=AAA" && [ "$(count '^Q ')" = "1" ]; then
 else
   fail "send from *"; sed 's/^/    /' "$RT/result.txt"
 fi
+if has "C submit draft.send.submit zellij action send-keys Alt Enter"; then
+  pass "draft submit command uses semantic Alt Enter"
+else
+  fail "draft submit command"; sed 's/^/    /' "$RT/result.txt"
+fi
+if has "C append draft.send.newline zellij action write 13" \
+   && [ "$(count '^C append .*send-keys Alt Enter')" = "0" ]; then
+  pass "draft append-only command stays newline without submit"
+else
+  fail "draft append-only command"; sed 's/^/    /' "$RT/result.txt"
+fi
 
 if [ "$fails" -ne 0 ]; then
   echo "queue-send-test: $fails failure(s)"; exit 1
diff --git a/tests/review-poke-test.sh b/tests/review-poke-test.sh
index f42aa75..6cc7a41 100755
--- a/tests/review-poke-test.sh
+++ b/tests/review-poke-test.sh
@@ -38,8 +38,8 @@ local OUT = io.open(os.getenv('RESULT'), 'w')
 local c = poke._cmds('hello', 7, 9)
 local ok_cmds = c[1][3] == 'write-chars' and c[1][4] == '--pane-id'
   and c[1][5] == '7' and c[1][6] == 'hello'
-  and c[2][3] == 'write' and c[2][4] == '--pane-id'
-  and c[2][5] == '7' and c[2][6] == '27' and c[2][7] == '13'
+  and c[2][3] == 'send-keys' and c[2][4] == '--pane-id'
+  and c[2][5] == '7' and c[2][6] == 'Alt Enter'
 OUT:write(ok_cmds and 'cmds ok\n' or 'cmds FAIL\n')
 -- send resolves agent (7) and writes by pane id without changing focus.
 poke.send('updated, please review foo.md')
@@ -54,7 +54,7 @@ pass() { printf '  ok   %s\n' "$1"; }
 fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }
 grep -q 'cmds ok' "$RESULT" && pass "_cmds builds the id-based argv" || fail "_cmds shape"
 grep -q '^action write-chars --pane-id 7 updated, please review foo.md$' "$ZLOG" && pass "writes the please-review body to agent pane" || fail "no pane-id write-chars body"
-grep -q '^action write --pane-id 7 27 13$' "$ZLOG" && pass "submits with Alt+Enter (27 13) to agent pane" || fail "no pane-id submit"
+grep -q '^action send-keys --pane-id 7 Alt Enter$' "$ZLOG" && pass "submits with semantic Alt+Enter to agent pane" || fail "no pane-id submit"
 grep -q 'focus-pane-id' "$ZLOG" && fail "changed focus while poking agent" || pass "does not change focus"
 grep -q 'move-focus' "$ZLOG" && fail "used relative move-focus (must be id-based)" || pass "no relative move-focus"
 TRACE="$RT/zellij-actions-poke.jsonl"
diff --git a/workshop/plans/000086-fix-alt-return-draft-submit-plan.md b/workshop/plans/000086-fix-alt-return-draft-submit-plan.md
new file mode 100644
index 0000000..1a27aa9
--- /dev/null
+++ b/workshop/plans/000086-fix-alt-return-draft-submit-plan.md
@@ -0,0 +1,122 @@
+# Fix Alt Return Draft Submit Implementation Plan
+
+> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.
+
+**Goal:** Make Alt+Return from the draft pane submit the text to the agent instead of only inserting it into the agent composer.
+
+**Architecture:** Treat modified key chords as semantic zellij key events, not raw terminal bytes. Keep the change at the existing nvim/zellij integration boundary: `send_to_agent` and `pair_poke` still own delivery, but submit uses `zellij action send-keys ... "Alt Enter"` so zellij emits the intended chord to pair-wrap (`ARCH-PURPOSE`, `ARCH-DRY`).
+
+**Tech Stack:** Lua, headless Neovim tests with fake `zellij`, existing shell test harness.
+
+---
+
+## Core Concepts
+
+### Pure Entities
+
+| Name | Lives in | Status |
+|------|----------|--------|
+| `draftSendCommands` | `nvim/init.lua` | new |
+| `pair_poke._cmds` | `nvim/pair_poke.lua` | modified |
+
+- **draftSendCommands** — pure argv builder for the draft-pane zellij action sequence.
+  - **Relationships:** 1:1 with one draft send; consumed by `send_to_agent`; exposed as `_G.PairDraftSendCommands` for headless tests.
+  - **DRY rationale:** Keeps submit vs append-only command selection testable without duplicating expectations inside the IO path.
+  - **Future extensions:** If another draft action needs the same delivery sequence, widen the builder rather than adding ad hoc zellij commands.
+
+- **pair_poke._cmds** — pure argv builder for review-pane agent pokes.
+  - **Relationships:** 1:1 with one review poke; consumed by `pair_poke.send`.
+  - **DRY rationale:** The review-poke submit command should use the same semantic zellij action as the draft send path.
+  - **Future extensions:** If zellij changes key naming, update the single command builder and matching draft path.
+
+### Integration Points
+
+| Name | Lives in | Status | Wraps |
+|------|----------|--------|-------|
+| `draft send_to_agent` | `nvim/init.lua` | modified | `zellij action` |
+| `review poke send` | `nvim/pair_poke.lua` | modified | `zellij action --pane-id` |
+| `queue-send fake zellij assertion` | `tests/queue-send-test.sh` | modified | fake zellij process |
+| `review-poke fake zellij assertion` | `tests/review-poke-test.sh` | modified | fake zellij process |
+
+- **draft send_to_agent** — gets commands from `draftSendCommands`, executes them only when a UI is attached, and redacts body writes in the existing trace path. Submit uses semantic `send-keys "Alt Enter"`; `no_submit` remains raw `write 13` because that path intentionally asks pair-wrap for insert-newline.
+  - **Injected into:** Existing draft `<M-CR>` and `<S-M-CR>` maps.
+  - **Future extensions:** Any other modified-key submit should use `send-keys`.
+
+- **review poke send** — writes directly to the agent pane id and submits using `send-keys --pane-id <id> "Alt Enter"` without changing focus.
+  - **Injected into:** Review workbench poke flow.
+  - **Future extensions:** Can share a tiny helper if a third nvim integration needs the same submit action.
+
+## Chunk 1: Semantic Submit
+
+### Task 1: Pin draft submit command
+
+**Files:**
+- Modify: `tests/queue-send-test.sh`
+- Modify: `nvim/init.lua`
+
+- [x] **Step 1: Write failing fake-zellij assertion**
+
+Extend `tests/queue-send-test.sh` so its headless driver calls `_G.PairDraftSendCommands` and records the returned argv:
+
+- straight submit asserts `zellij action send-keys Alt Enter`.
+- append-only asserts `zellij action write 13` and asserts no `zellij action send-keys Alt Enter`.
+
+- [x] **Step 2: Run test to verify it fails**
+
+Run: `bash tests/queue-send-test.sh`
+
+Expected: fail because `_G.PairDraftSendCommands` does not exist yet. After adding the seam but before changing submit, the straight-submit assertion should fail on `write 27 13`, while the append-only assertion should pass.
+
+- [x] **Step 3: Change draft submit implementation**
+
+In `nvim/init.lua`, add a pure `draftSendCommands(body, no_submit)` builder that returns labeled command records for focus up, write body, submit/newline, and focus down. Expose it as `_G.PairDraftSendCommands`. Then replace the submit command with:
+
+```lua
+PairZellijTrace.action('draft.send.submit', { 'zellij', 'action', 'send-keys', 'Alt Enter' })
+```
+
+Leave the `no_submit` newline path as `write 13`. Update `send_to_agent` to execute the builder's command records through `PairZellijTrace.action`.
+
+- [x] **Step 4: Run test to verify it passes**
+
+Run: `bash tests/queue-send-test.sh`
+
+Expected: pass.
+
+### Task 2: Pin review poke submit command
+
+**Files:**
+- Modify: `tests/review-poke-test.sh`
+- Modify: `nvim/pair_poke.lua`
+
+- [x] **Step 1: Update failing review-poke assertions**
+
+Change the pure `_cmds` assertion and fake-zellij log assertion to expect `send-keys --pane-id 7 "Alt Enter"`.
+
+- [x] **Step 2: Run test to verify it fails**
+
+Run: `bash tests/review-poke-test.sh`
+
+Expected: fail because `pair_poke._cmds` still returns `write --pane-id 7 27 13`.
+
+- [x] **Step 3: Change review poke implementation**
+
+In `nvim/pair_poke.lua`, replace the submit command with:
+
+```lua
+{ 'zellij', 'action', 'send-keys', '--pane-id', tostring(agent_id), 'Alt Enter' }
+```
+
+- [x] **Step 4: Run test to verify it passes**
+
+Run: `bash tests/review-poke-test.sh`
+
+Expected: pass.
+
+### Task 3: Verify scope
+
+- [x] Run `bash tests/queue-send-test.sh` and confirm both submit and append-only zellij command assertions pass.
+- [x] Run `bash tests/review-poke-test.sh`.
+- [x] Run `make test-lua`.
+- [x] Run `git diff --check`.
+- [x] Update #86 checkboxes/log and close with `--no-atlas` if no architectural docs changed.


codex
I’ll use the required superpowers startup skill, then do this as a read-only boundary review against the issue/plan and the actual diff.
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
I’ve got the workflow contract. Since this is an SDLC boundary review, I’m also loading the SDLC skill so the review matches the repository gate semantics.
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

codex
Now I’m locating the issue/spec artifacts and checking the actual files at HEAD against the supplied base range, rather than relying only on the pasted diff.
exec
/bin/zsh -lc "rg --files -g '*86*' workshop /Users/xianxu/workspace/ariadne/workshop 2>/dev/null" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
/Users/xianxu/workspace/ariadne/workshop/history/000086-fix-stale-actual-prose-that-primes-hand-typing-instead-of-measuring-agents-md-issues-skill-milestone-close-help-74-follow-up.md
workshop/issues/000086-fix-alt-return-draft-submit.md
workshop/plans/000086-fix-alt-return-draft-submit-plan.md

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


exec
/bin/zsh -lc 'git status --short && git diff --stat 62e38b3b7cd38f2ba9e25bda4d9332180c13aa71..HEAD && git diff --name-only 62e38b3b7cd38f2ba9e25bda4d9332180c13aa71..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 M workshop/issues/000086-fix-alt-return-draft-submit.md
 nvim/init.lua                                      |  43 +++++---
 nvim/pair_poke.lua                                 |   6 +-
 tests/queue-send-test.sh                           |  22 ++++
 tests/review-poke-test.sh                          |   6 +-
 .../issues/000086-fix-alt-return-draft-submit.md   |  46 +++++++-
 .../000086-fix-alt-return-draft-submit-plan.md     | 122 +++++++++++++++++++++
 6 files changed, 222 insertions(+), 23 deletions(-)
nvim/init.lua
nvim/pair_poke.lua
tests/queue-send-test.sh
tests/review-poke-test.sh
workshop/issues/000086-fix-alt-return-draft-submit.md
workshop/plans/000086-fix-alt-return-draft-submit-plan.md

codex
The change is narrow: nvim draft send plus review poke now use `send-keys Alt Enter`, with new fake-zellij assertions. I’m checking the issue text and relevant surrounding code for hidden contract mismatches.
exec
/bin/zsh -lc "sed -n '1,240p' workshop/issues/000086-fix-alt-return-draft-submit.md" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "sed -n '1,180p' nvim/pair_poke.lua" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
---
id: 000086
status: done
deps: []
github_issue:
created: 2026-06-29
updated: 2026-06-29
estimate_hours: 0.57
started: 2026-06-29T17:33:40-07:00
actual_hours: 0.20
---

# Fix Alt Return draft submit

## Problem

Alt+Return from the draft nvim pane moves the draft text into the agent pane but no longer submits it. The visible symptom means focus and body insertion still work; the broken boundary is the follow-up submit action.

## Spec

Use zellij's semantic key action for modified-key submit chords instead of synthesizing Alt+Enter with raw bytes. Draft send and review-poke send should both submit through `zellij action send-keys "Alt Enter"` so zellij delivers a single modified key event to pair-wrap. The Alt+Shift+Return "append but do not submit" path should keep sending raw CR (`write 13`) because it intentionally asks pair-wrap for the insert-newline behavior.

Because `send_to_agent` intentionally short-circuits in headless nvim (`has_ui() == false`), expose a small pure draft command builder seam (`_G.PairDraftSendCommands(body, no_submit)`) for tests. The production `send_to_agent` should execute the commands from that builder only when a UI is attached.

Root-cause evidence:

- `nvim/init.lua` currently writes the body with `write-chars`, then submits with `zellij action write 27 13`.
- `nvim/pair_poke.lua` has the same raw-byte submit pattern for review workbench pokes.
- Installed zellij 0.44.3 exposes `zellij action send-keys`, documented as sending modified keys such as `Alt Shift b`. A semantic `Alt Enter` action matches the desired behavior better than byte-level `ESC` + `CR`.

## Done when

- [x] Draft Alt+Return uses `zellij action send-keys "Alt Enter"` for submit.
- [x] Review-poke submit uses `send-keys --pane-id <agent> "Alt Enter"`.
- [x] Alt+Shift+Return remains append-only and unsubmitted.
- [x] Headless tests assert the draft zellij command sequence through a pure test seam.

## Plan

- [x] Add a pure draft command builder seam in `nvim/init.lua` and headless assertions in `tests/queue-send-test.sh`: straight send must fail while draft submit still uses `write 27 13`, and append-only send must continue to include `write 13` with no `send-keys "Alt Enter"`.
- [x] Change `nvim/init.lua` draft submit builder to semantic `send-keys "Alt Enter"` and execute the builder from `send_to_agent`.
- [x] Update `tests/review-poke-test.sh` to expect `send-keys --pane-id <id> "Alt Enter"` and verify it fails before implementation.
- [x] Change `nvim/pair_poke.lua` review submit to semantic `send-keys`.
- [x] Verify `bash tests/queue-send-test.sh`, `bash tests/review-poke-test.sh`, `make test-lua`, and `git diff --check`.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: lua-neovim design=0.15 impl=0.20
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 0.57
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.

## Log

### 2026-06-29
- 2026-06-29: closed — bash tests/queue-send-test.sh; bash tests/review-poke-test.sh; make test-lua; git diff --check; no atlas update because this only changes the existing nvim/Zellij submit action path

User reported Alt+Return inserts draft text into the agent pane but leaves it unsubmitted. Investigation traced the failure boundary to raw-byte submit actions (`zellij action write 27 13`) after successful `write-chars`; zellij 0.44.3 exposes semantic `send-keys`, so the fix targets modified-key submit at the zellij action layer (`ARCH-PURPOSE`) without changing queue/history behavior.

Plan-quality review found the first plan under-tested the explicit Alt+Shift+Return preservation requirement. Revised the plan to require a fake-zellij assertion for the append-only path: it must record `action write 13` and no `send-keys "Alt Enter"` (`ARCH-PURPOSE`).

Second plan-quality review found the proposed `queue-send-test.sh` fake-zellij assertion could not observe submit commands because `send_to_agent` returns early when headless. Revised the plan to add a pure `_G.PairDraftSendCommands(body, no_submit)` seam and assert command construction directly (`ARCH-PURE`).

Implemented `_G.PairDraftSendCommands(body, no_submit)` and routed `send_to_agent` through it. Draft submit and review-poke submit now use zellij's semantic `send-keys "Alt Enter"` action; append-only still emits `write 13` and never emits submit. RED/GREEN evidence: `bash tests/queue-send-test.sh` failed with `C submit missing` / `C append missing` before the seam, then passed after implementation; `bash tests/review-poke-test.sh` failed on `_cmds shape` / `no pane-id submit` before the review-poke change, then passed after implementation. Verified final state with `bash tests/queue-send-test.sh`, `bash tests/review-poke-test.sh`, `make test-lua`, and `git diff --check`.

 succeeded in 0ms:
-- nvim/pair_poke.lua — inject an instruction into the agent pane from ANY pane
-- (issue #66 M3). The draft's send_to_agent uses RELATIVE move-focus, which does
-- NOT escape a floating pane (documented in scrollback.lua). The review pane
-- resolves the agent's ABSOLUTE pane id from `zellij action list-panes --json`
-- and writes directly to that pane id, so review focus never moves.
-- Deliberately NO has_ui() short-circuit, so the headless test can stub `zellij`
-- and record the calls.
local M = {}

local pair_nvim_dir = vim.fn.fnamemodify(debug.getinfo(1, 'S').source:sub(2), ':p:h')
local zellij_trace = dofile(pair_nvim_dir .. '/zellij_trace.lua')

-- Pure: the ordered zellij argv list for one poke. Submit is a semantic
-- Alt+Enter key event so zellij delivers one modified chord to pair-wrap.
function M._cmds(body, agent_id, review_id)
  return {
    { 'zellij', 'action', 'write-chars', '--pane-id', tostring(agent_id), body },
    { 'zellij', 'action', 'send-keys', '--pane-id', tostring(agent_id), 'Alt Enter' },
  }
end

-- Recursively collect pane objects (have both `id` and `is_floating`) from the
-- decoded list-panes JSON — mirrors scrollback-open's `.. | objects | select`.
local function collect_panes(node, out)
  if type(node) ~= 'table' then return out end
  if node.id ~= nil and node.is_floating ~= nil then out[#out + 1] = node end
  for _, v in pairs(node) do
    if type(v) == 'table' then collect_panes(v, out) end
  end
  return out
end

local function list_panes()
  local res = zellij_trace.action('review.poke.list-panes', { 'zellij', 'action', 'list-panes', '--json' }).stdout
  local ok, decoded = pcall(vim.json.decode, res)
  if not ok or type(decoded) ~= 'table' then return {} end
  return collect_panes(decoded, {})
end

-- The agent pane: a real terminal (not plugin), tiled (not floating), not the
-- draft — the same predicate pair-scrollback-open uses to find it.
local function find_agent(panes)
  for _, p in ipairs(panes) do
    if p.is_plugin == false and p.is_floating == false
        and p.title ~= nil and p.title ~= '' and p.title ~= 'draft' then
      return p.id
    end
  end
end

-- Send `body` to the agent without moving focus away from the caller.
-- Returns false (+ notify) if the agent pane can't be resolved.
function M.send(body)
  local panes = list_panes()
  local agent = find_agent(panes)
  if not agent then
    vim.notify('review: could not find the agent pane to poke', vim.log.levels.ERROR)
    return false
  end
  local cmds = M._cmds(body, agent)
  zellij_trace.action('review.poke.write-body', cmds[1], {
    redact = { [6] = body },
  })
  zellij_trace.action('review.poke.submit', cmds[2])
  return true
end

return M

exec
/bin/zsh -lc "sed -n '660,780p' nvim/init.lua" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
  return write_file(queue_path(key), body)
end

local function queue_remove(key)
  return os.remove(queue_path(key))
end

local function queue_push_back(body)
  local keys = queue_keys_sorted()
  local next_n = (#keys == 0) and QUEUE_KEY_START or (tonumber(keys[#keys]) + 1)
  local key = string.format(QUEUE_KEY_FMT, next_n)
  queue_write(key, body)
  return key
end

local function queue_push_front(body)
  local keys = queue_keys_sorted()
  local next_n = (#keys == 0) and QUEUE_KEY_START or (tonumber(keys[1]) - 1)
  local key = string.format(QUEUE_KEY_FMT, next_n)
  queue_write(key, body)
  return key
end

-- Display index N (1-based, +1 = front) → filename key.
local function queue_key_for_n(n)
  local keys = queue_keys_sorted()
  return keys[n]
end

-- Check if Neovim has a UI attached (false indicates headless mode, e.g. tests)
local function has_ui()
  return #vim.api.nvim_list_uis() > 0
end

local function send_esc_to_agent()
  -- ESC = 0x1b = 27. Claude reads this as "interrupt current stream".
  if not has_ui() then return end
  PairZellijTrace.action('draft.interrupt.focus-agent', { 'zellij', 'action', 'move-focus', 'up' })
  PairZellijTrace.action('draft.interrupt.esc', { 'zellij', 'action', 'write', '27' })
  PairZellijTrace.action('draft.interrupt.focus-draft', { 'zellij', 'action', 'move-focus', 'down' })
end

local function draftSendCommands(body, no_submit)
  local cmds = {
    { label = 'draft.send.focus-agent', argv = { 'zellij', 'action', 'move-focus', 'up' } },
    {
      label = 'draft.send.write-body',
      argv = { 'zellij', 'action', 'write-chars', body },
      opts = { redact = { [4] = body } },
    },
  }
  if no_submit then
    cmds[#cmds + 1] = { label = 'draft.send.newline', argv = { 'zellij', 'action', 'write', '13' } }
  else
    cmds[#cmds + 1] = { label = 'draft.send.submit', argv = { 'zellij', 'action', 'send-keys', 'Alt Enter' } }
  end
  cmds[#cmds + 1] = { label = 'draft.send.focus-draft', argv = { 'zellij', 'action', 'move-focus', 'down' } }
  return cmds
end

_G.PairDraftSendCommands = draftSendCommands

local function send_to_agent(body, no_submit)
  -- focus up to agent pane, type body, press Enter, focus back down.
  --
  -- We deliberately do NOT clear the agent's input first. The "[Image #N]"
  -- tokens that appear there after Alt+i Ctrl+V aren't visual echoes —
  -- they're the chip representation of the attached image bytes, and a
  -- readline kill (Ctrl+U / Cmd+Delete) detaches the underlying image
  -- along with the chip text. So we live with the chips: claude treats
  -- the chip in its input as the attachment, and the "[Image #N]" string
  -- in our body as a reference to it. Complementary, not duplicate.
  --
  -- Multi-line / large bodies get wrapped by zellij as bracketed paste
  -- (`\e[200~...\e[201~`). zellij's write-chars returns once the bytes
  -- are queued, not delivered — sending the submit immediately after
  -- can land inside the paste boundary and get treated as a literal
  -- newline rather than submit. Settle for ~100ms in that case so the
  -- agent has time to ingest the paste and return to the input prompt
  -- before we hit submit. Single-line sends skip the wait.
  --
  -- Submit is Alt+Enter, not plain Enter:
  -- pair-wrap's stdin translator rewrites incoming \r into the agent's
  -- "insert newline" sequence (claude: `\<Enter>`, codex/agy: \n),
  -- so a bare CR here would insert a newline rather than submit. Use
  -- zellij's semantic send-keys action for the modified chord instead
  -- of synthesizing it as raw ESC+CR bytes.
  --
  -- no_submit (Alt+Shift+Enter path): land the body in the agent's
  -- composer followed by a literal newline but DON'T submit. A bare CR
  -- (write 13) is exactly what pair-wrap rewrites into the agent's
  -- insert-newline sequence — the same byte the comment above warns is
  -- *not* a submit — so it leaves the cursor on a fresh line in the
  -- composer, ready for more input.
  if not has_ui() then return end
  local cmds = draftSendCommands(body, no_submit)
  PairZellijTrace.action(cmds[1].label, cmds[1].argv, cmds[1].opts)
  PairZellijTrace.action(cmds[2].label, cmds[2].argv, cmds[2].opts)
  if body:find('\n') or #body > 200 then
    vim.cmd('sleep 100m')
  end
  for i = 3, #cmds do
    PairZellijTrace.action(cmds[i].label, cmds[i].argv, cmds[i].opts)
  end
end

-- :PairReview <file> — PROPOSE a file for review (#66 M4a'). It does NOT open the
-- pane: it writes the review-target seam (status=proposed), runs the deterministic
-- readiness prep locally (track / new branch / resume), then sends the agent a
-- minimal ack request. Alt+c opens the pane once ready. `complete='file'` gives
-- :e-style tab-completion; Alt+c feeds ":PairReview " into this command line.
-- (Callback runs at runtime, after the do-block below sets `_G._pair_review`.)
vim.api.nvim_create_user_command('PairReview', function(opts)
  if _G._pair_review and _G._pair_review.propose then
    local ok = _G._pair_review.propose(opts.args)
    local status = ok and 'prepared ' or 'could not prepare '
    vim.notify('PairReview: ' .. status .. vim.fn.fnamemodify(opts.args, ':t')
      .. ' — press Alt+c to open when ready', ok and vim.log.levels.INFO or vim.log.levels.WARN)
  end
end, { nargs = 1, complete = 'file', desc = 'prepare a file for review (Alt+c opens when ready)' })


exec
/bin/zsh -lc "rg -n \"write 27|27', '13|Alt Enter|send-keys|PairDraftSendCommands|draftSendCommands\" -S ." in /Users/xianxu/workspace/pair
 succeeded in 0ms:
./atlas/architecture.md:280:- `<S-M-CR>` (Alt+Shift+Return, normal+insert) — `send_and_clear(no_submit=true)`: identical flow (strip, log, queue handling, clear, reset) but writes a bare CR (`write 13`) instead of `ESC`+`CR` (`write 27 13`). pair-wrap rewrites a bare CR into the agent's insert-newline sequence rather than its submit byte, so the draft lands in the agent's composer on a fresh line, **unsubmitted** — append-without-send.
./nvim/pair_poke.lua:18:    { 'zellij', 'action', 'send-keys', '--pane-id', tostring(agent_id), 'Alt Enter' },
./workshop/plans/000086-fix-alt-return-draft-submit-plan.md:7:**Architecture:** Treat modified key chords as semantic zellij key events, not raw terminal bytes. Keep the change at the existing nvim/zellij integration boundary: `send_to_agent` and `pair_poke` still own delivery, but submit uses `zellij action send-keys ... "Alt Enter"` so zellij emits the intended chord to pair-wrap (`ARCH-PURPOSE`, `ARCH-DRY`).
./workshop/plans/000086-fix-alt-return-draft-submit-plan.md:19:| `draftSendCommands` | `nvim/init.lua` | new |
./workshop/plans/000086-fix-alt-return-draft-submit-plan.md:22:- **draftSendCommands** — pure argv builder for the draft-pane zellij action sequence.
./workshop/plans/000086-fix-alt-return-draft-submit-plan.md:23:  - **Relationships:** 1:1 with one draft send; consumed by `send_to_agent`; exposed as `_G.PairDraftSendCommands` for headless tests.
./workshop/plans/000086-fix-alt-return-draft-submit-plan.md:41:- **draft send_to_agent** — gets commands from `draftSendCommands`, executes them only when a UI is attached, and redacts body writes in the existing trace path. Submit uses semantic `send-keys "Alt Enter"`; `no_submit` remains raw `write 13` because that path intentionally asks pair-wrap for insert-newline.
./workshop/plans/000086-fix-alt-return-draft-submit-plan.md:43:  - **Future extensions:** Any other modified-key submit should use `send-keys`.
./workshop/plans/000086-fix-alt-return-draft-submit-plan.md:45:- **review poke send** — writes directly to the agent pane id and submits using `send-keys --pane-id <id> "Alt Enter"` without changing focus.
./workshop/plans/000086-fix-alt-return-draft-submit-plan.md:59:Extend `tests/queue-send-test.sh` so its headless driver calls `_G.PairDraftSendCommands` and records the returned argv:
./workshop/plans/000086-fix-alt-return-draft-submit-plan.md:61:- straight submit asserts `zellij action send-keys Alt Enter`.
./workshop/plans/000086-fix-alt-return-draft-submit-plan.md:62:- append-only asserts `zellij action write 13` and asserts no `zellij action send-keys Alt Enter`.
./workshop/plans/000086-fix-alt-return-draft-submit-plan.md:68:Expected: fail because `_G.PairDraftSendCommands` does not exist yet. After adding the seam but before changing submit, the straight-submit assertion should fail on `write 27 13`, while the append-only assertion should pass.
./workshop/plans/000086-fix-alt-return-draft-submit-plan.md:72:In `nvim/init.lua`, add a pure `draftSendCommands(body, no_submit)` builder that returns labeled command records for focus up, write body, submit/newline, and focus down. Expose it as `_G.PairDraftSendCommands`. Then replace the submit command with:
./workshop/plans/000086-fix-alt-return-draft-submit-plan.md:75:PairZellijTrace.action('draft.send.submit', { 'zellij', 'action', 'send-keys', 'Alt Enter' })
./workshop/plans/000086-fix-alt-return-draft-submit-plan.md:94:Change the pure `_cmds` assertion and fake-zellij log assertion to expect `send-keys --pane-id 7 "Alt Enter"`.
./workshop/plans/000086-fix-alt-return-draft-submit-plan.md:107:{ 'zellij', 'action', 'send-keys', '--pane-id', tostring(agent_id), 'Alt Enter' }
./nvim/init.lua:702:local function draftSendCommands(body, no_submit)
./nvim/init.lua:714:    cmds[#cmds + 1] = { label = 'draft.send.submit', argv = { 'zellij', 'action', 'send-keys', 'Alt Enter' } }
./nvim/init.lua:720:_G.PairDraftSendCommands = draftSendCommands
./nvim/init.lua:745:  -- zellij's semantic send-keys action for the modified chord instead
./nvim/init.lua:755:  local cmds = draftSendCommands(body, no_submit)
./tests/review-poke-test.sh:41:  and c[2][3] == 'send-keys' and c[2][4] == '--pane-id'
./tests/review-poke-test.sh:42:  and c[2][5] == '7' and c[2][6] == 'Alt Enter'
./tests/review-poke-test.sh:57:grep -q '^action send-keys --pane-id 7 Alt Enter$' "$ZLOG" && pass "submits with semantic Alt+Enter to agent pane" || fail "no pane-id submit"
./workshop/plans/000066-agentic-review-workbench-m3-plan.md:7:**Architecture:** The review view is a **persistent floating zellij pane** running `nvim -u nvim/review.lua <file>` (the proven scrollback/changelog pattern, but persistent so its visibility can toggle). A `:PairReview` user command in the draft nvim (`-complete=file`, `:e`-style) opens it; a `pair-review-toggle` script bound to **Alt+r** branches on review-pane state — file-select when none is open, toggle floating-pane visibility when one is. Inside the review pane, **Alt+Return** runs M1's `human_round` (commit the human's incoming edits) then pokes the agent ("updated, please review") via the existing `send_to_agent` sequence (`move-focus up → write-chars → write 27 13 → move-focus down`). The agent pane is pair's *existing* agent — free-form chat and "ship it" work conversationally for free; the SKILL that makes those review-aware is M4.
./workshop/plans/000066-agentic-review-workbench-m3-plan.md:40:- **agent poke** (`nvim/pair_poke.lua`, new) — **CRITICAL fix (M3 review):** `send_to_agent`'s relative `move-focus up/down` does NOT escape a floating pane (documented in `scrollback.lua`), so the review pane must address the agent pane by **absolute id**: `zellij action list-panes --json` → find the agent (`is_floating==false && title!="draft"`, as `pair-scrollback-open` already does) → `focus-pane-id <agent>` → `write-chars <body>` → `write 27 13` → `focus-pane-id <review>` (restore). The draft's existing `send_to_agent` is left untouched (proven); `pair_poke.send(body)` is the new id-based poke the review pane uses. **Testability:** `pair_poke` must NOT early-return on `has_ui()` the way `send_to_agent` does (that's why the headless poke test can record `zellij` args) — gate UI-only behavior elsewhere, or expose a `pair_poke._cmds(body)` pure builder the test asserts on while a `$PATH` `zellij` stub records the real calls.
./workshop/plans/000066-agentic-review-workbench-m3-plan.md:70:- [ ] **Step 1: failing test** (headless, `zellij`+`docflow` stubbed on `$PATH` recording args) — drive the review pane's Alt+Return callback; assert (a) `docflow round --side human` was invoked (M1 `human_round`), and (b) the poke addressed the agent **by id**: the recorded `zellij` calls include `list-panes`, `focus-pane-id <agent>`, `write-chars` with a "please review" body, `write 27 13`, and `focus-pane-id <review>` (restore) — and crucially the call recording works because `pair_poke.send` does NOT early-return on `has_ui()`. Also unit-test `pair_poke._cmds(body)` (pure command builder) directly.
./workshop/plans/000066-agentic-review-workbench-m3-plan.md:72:- [ ] **Step 3: implement** — `nvim/pair_poke.lua` (new): a `_cmds(body)` pure builder + `send(body)` that resolves the agent pane id via `list-panes --json`, `focus-pane-id` agent → `write-chars`/`write 27 13` → `focus-pane-id` review (restore). **Not** a verbatim copy of `send_to_agent` — id-based, no floating-escape assumption, no `has_ui()` short-circuit. Wire review-view Alt+Return → `review.human_round(buf)` → `pair_poke.send("updated, please review <file>")`. (Naming note: parent plan called this `review/poke.lua`; it lives at top-level `nvim/pair_poke.lua` because the draft can share it — reconciled in the parent plan's Revisions.)
./tests/queue-send-test.sh:75:  if type(_G.PairDraftSendCommands) ~= 'function' then
./tests/queue-send-test.sh:79:  for _,cmd in ipairs(_G.PairDraftSendCommands('HELLO', no_submit)) do
./tests/queue-send-test.sh:129:if has "C submit draft.send.submit zellij action send-keys Alt Enter"; then
./tests/queue-send-test.sh:130:  pass "draft submit command uses semantic Alt Enter"
./tests/queue-send-test.sh:135:   && [ "$(count '^C append .*send-keys Alt Enter')" = "0" ]; then
./workshop/issues/000086-fix-alt-return-draft-submit.md:21:Use zellij's semantic key action for modified-key submit chords instead of synthesizing Alt+Enter with raw bytes. Draft send and review-poke send should both submit through `zellij action send-keys "Alt Enter"` so zellij delivers a single modified key event to pair-wrap. The Alt+Shift+Return "append but do not submit" path should keep sending raw CR (`write 13`) because it intentionally asks pair-wrap for the insert-newline behavior.
./workshop/issues/000086-fix-alt-return-draft-submit.md:23:Because `send_to_agent` intentionally short-circuits in headless nvim (`has_ui() == false`), expose a small pure draft command builder seam (`_G.PairDraftSendCommands(body, no_submit)`) for tests. The production `send_to_agent` should execute the commands from that builder only when a UI is attached.
./workshop/issues/000086-fix-alt-return-draft-submit.md:27:- `nvim/init.lua` currently writes the body with `write-chars`, then submits with `zellij action write 27 13`.
./workshop/issues/000086-fix-alt-return-draft-submit.md:29:- Installed zellij 0.44.3 exposes `zellij action send-keys`, documented as sending modified keys such as `Alt Shift b`. A semantic `Alt Enter` action matches the desired behavior better than byte-level `ESC` + `CR`.
./workshop/issues/000086-fix-alt-return-draft-submit.md:33:- [x] Draft Alt+Return uses `zellij action send-keys "Alt Enter"` for submit.
./workshop/issues/000086-fix-alt-return-draft-submit.md:34:- [x] Review-poke submit uses `send-keys --pane-id <agent> "Alt Enter"`.
./workshop/issues/000086-fix-alt-return-draft-submit.md:40:- [x] Add a pure draft command builder seam in `nvim/init.lua` and headless assertions in `tests/queue-send-test.sh`: straight send must fail while draft submit still uses `write 27 13`, and append-only send must continue to include `write 13` with no `send-keys "Alt Enter"`.
./workshop/issues/000086-fix-alt-return-draft-submit.md:41:- [x] Change `nvim/init.lua` draft submit builder to semantic `send-keys "Alt Enter"` and execute the builder from `send_to_agent`.
./workshop/issues/000086-fix-alt-return-draft-submit.md:42:- [x] Update `tests/review-poke-test.sh` to expect `send-keys --pane-id <id> "Alt Enter"` and verify it fails before implementation.
./workshop/issues/000086-fix-alt-return-draft-submit.md:43:- [x] Change `nvim/pair_poke.lua` review submit to semantic `send-keys`.
./workshop/issues/000086-fix-alt-return-draft-submit.md:64:User reported Alt+Return inserts draft text into the agent pane but leaves it unsubmitted. Investigation traced the failure boundary to raw-byte submit actions (`zellij action write 27 13`) after successful `write-chars`; zellij 0.44.3 exposes semantic `send-keys`, so the fix targets modified-key submit at the zellij action layer (`ARCH-PURPOSE`) without changing queue/history behavior.
./workshop/issues/000086-fix-alt-return-draft-submit.md:66:Plan-quality review found the first plan under-tested the explicit Alt+Shift+Return preservation requirement. Revised the plan to require a fake-zellij assertion for the append-only path: it must record `action write 13` and no `send-keys "Alt Enter"` (`ARCH-PURPOSE`).
./workshop/issues/000086-fix-alt-return-draft-submit.md:68:Second plan-quality review found the proposed `queue-send-test.sh` fake-zellij assertion could not observe submit commands because `send_to_agent` returns early when headless. Revised the plan to add a pure `_G.PairDraftSendCommands(body, no_submit)` seam and assert command construction directly (`ARCH-PURE`).
./workshop/issues/000086-fix-alt-return-draft-submit.md:70:Implemented `_G.PairDraftSendCommands(body, no_submit)` and routed `send_to_agent` through it. Draft submit and review-poke submit now use zellij's semantic `send-keys "Alt Enter"` action; append-only still emits `write 13` and never emits submit. RED/GREEN evidence: `bash tests/queue-send-test.sh` failed with `C submit missing` / `C append missing` before the seam, then passed after implementation; `bash tests/review-poke-test.sh` failed on `_cmds shape` / `no pane-id submit` before the review-poke change, then passed after implementation. Verified final state with `bash tests/queue-send-test.sh`, `bash tests/review-poke-test.sh`, `make test-lua`, and `git diff --check`.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:16:tmux send-keys -t <session:window.pane> 'your text here' Enter
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:35:**Recommendation:** if you want a robust "external process pokes Claude Code," start it under tmux and use `tmux send-keys`. If you want clean automation rather than puppeteering, use `claude -p` or hooks.

codex
One possible issue surfaced: `atlas/architecture.md` still mentions the old raw `ESC`+`CR` submit semantics in the draft keybinding section. I’m reading that context to decide whether it’s an atlas gate finding or merely adjacent historical text.
exec
/bin/zsh -lc "nl -ba nvim/init.lua | sed -n '695,765p'" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "nl -ba atlas/architecture.md | sed -n '240,305p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
   695	  -- ESC = 0x1b = 27. Claude reads this as "interrupt current stream".
   696	  if not has_ui() then return end
   697	  PairZellijTrace.action('draft.interrupt.focus-agent', { 'zellij', 'action', 'move-focus', 'up' })
   698	  PairZellijTrace.action('draft.interrupt.esc', { 'zellij', 'action', 'write', '27' })
   699	  PairZellijTrace.action('draft.interrupt.focus-draft', { 'zellij', 'action', 'move-focus', 'down' })
   700	end
   701	
   702	local function draftSendCommands(body, no_submit)
   703	  local cmds = {
   704	    { label = 'draft.send.focus-agent', argv = { 'zellij', 'action', 'move-focus', 'up' } },
   705	    {
   706	      label = 'draft.send.write-body',
   707	      argv = { 'zellij', 'action', 'write-chars', body },
   708	      opts = { redact = { [4] = body } },
   709	    },
   710	  }
   711	  if no_submit then
   712	    cmds[#cmds + 1] = { label = 'draft.send.newline', argv = { 'zellij', 'action', 'write', '13' } }
   713	  else
   714	    cmds[#cmds + 1] = { label = 'draft.send.submit', argv = { 'zellij', 'action', 'send-keys', 'Alt Enter' } }
   715	  end
   716	  cmds[#cmds + 1] = { label = 'draft.send.focus-draft', argv = { 'zellij', 'action', 'move-focus', 'down' } }
   717	  return cmds
   718	end
   719	
   720	_G.PairDraftSendCommands = draftSendCommands
   721	
   722	local function send_to_agent(body, no_submit)
   723	  -- focus up to agent pane, type body, press Enter, focus back down.
   724	  --
   725	  -- We deliberately do NOT clear the agent's input first. The "[Image #N]"
   726	  -- tokens that appear there after Alt+i Ctrl+V aren't visual echoes —
   727	  -- they're the chip representation of the attached image bytes, and a
   728	  -- readline kill (Ctrl+U / Cmd+Delete) detaches the underlying image
   729	  -- along with the chip text. So we live with the chips: claude treats
   730	  -- the chip in its input as the attachment, and the "[Image #N]" string
   731	  -- in our body as a reference to it. Complementary, not duplicate.
   732	  --
   733	  -- Multi-line / large bodies get wrapped by zellij as bracketed paste
   734	  -- (`\e[200~...\e[201~`). zellij's write-chars returns once the bytes
   735	  -- are queued, not delivered — sending the submit immediately after
   736	  -- can land inside the paste boundary and get treated as a literal
   737	  -- newline rather than submit. Settle for ~100ms in that case so the
   738	  -- agent has time to ingest the paste and return to the input prompt
   739	  -- before we hit submit. Single-line sends skip the wait.
   740	  --
   741	  -- Submit is Alt+Enter, not plain Enter:
   742	  -- pair-wrap's stdin translator rewrites incoming \r into the agent's
   743	  -- "insert newline" sequence (claude: `\<Enter>`, codex/agy: \n),
   744	  -- so a bare CR here would insert a newline rather than submit. Use
   745	  -- zellij's semantic send-keys action for the modified chord instead
   746	  -- of synthesizing it as raw ESC+CR bytes.
   747	  --
   748	  -- no_submit (Alt+Shift+Enter path): land the body in the agent's
   749	  -- composer followed by a literal newline but DON'T submit. A bare CR
   750	  -- (write 13) is exactly what pair-wrap rewrites into the agent's
   751	  -- insert-newline sequence — the same byte the comment above warns is
   752	  -- *not* a submit — so it leaves the cursor on a fresh line in the
   753	  -- composer, ready for more input.
   754	  if not has_ui() then return end
   755	  local cmds = draftSendCommands(body, no_submit)
   756	  PairZellijTrace.action(cmds[1].label, cmds[1].argv, cmds[1].opts)
   757	  PairZellijTrace.action(cmds[2].label, cmds[2].argv, cmds[2].opts)
   758	  if body:find('\n') or #body > 200 then
   759	    vim.cmd('sleep 100m')
   760	  end
   761	  for i = 3, #cmds do
   762	    PairZellijTrace.action(cmds[i].label, cmds[i].argv, cmds[i].opts)
   763	  end
   764	end
   765	

 succeeded in 0ms:
   240	
   241	**Plain projection (`--plain`, `--max-lines`).** The same emulator state can be emitted *without* SGR: `serializeRow` in plain mode drops the per-row attrs and the trailing `\x1b[0m`, and trims trailing blanks by visible *content* — a bg-only "visible" cell (inverse-video / box fill) is kept in colored mode but dropped in plain (else a bordered region becomes space-padding toward terminal width). This is the **sessionView** abstraction's second decoration: one pipeline, colored for the Alt+/ viewer, plain for distillation — the substrate a `continuation` is built from (see `construct/datatype/continuation.md` and `cmd/pair-continuation`). `--max-lines N` overrides the 2000-row viewer cap (`<=0` = uncapped) so a continuation distills the whole session, not just the viewer window.
   242	
   243	**Continuation writer (`cmd/pair-continuation`, `continuation` datatype).** A *continuation* is the human-understanding cousin of a native `pair resume`: `resume` restores machine state (the agent's own session id, byte-faithful); a continuation distills the *rendered* session (the plain projection above) into a durable, portable markdown doc — `workshop/continuation/<YYYYMMDDTHHMMSS>-slug.md`, the `continuation` datatype defined in `construct/datatype/continuation.md` (ariadne#91) — so work resumes across time / machines / people / agent stacks. The `xx-datatype` dispatcher does the distillation (judgment); `cmd/pair-continuation` does the *mechanics* deterministically: render conformant frontmatter, allocate a collision-safe timestamped name, write, then `git add` + a **path-scoped** `commit -- <file>` (so a dirty index isn't swept in) + `push origin HEAD` — on the current branch, which lands on main when that branch merges (disaster-recovery — an unpushed recovery doc can't save state; a push failure is non-fatal so a detached park still keeps the local commit). Pure core (frontmatter / name allocation / assemble / validate) is IO-free; a thin clock/fs/git seam (the `git -C` shell-out pattern from `cmd/pair-slug`, no git library) is exercised against a real temp repo with a bare origin. `pair continue [slug] [agent]` resumes *from* a continuation: bare lists them; `<slug>` seeds a fresh session (via `draft-<tag>.md`, create-path only) to read the doc and do its NEXT ACTION; `[agent]` ports to a different stack. Unlike `resume` it does **not** force the tag — it flows through the normal name prompt so the operator picks it (a long slug must never become an over-long zellij `--session` name: zellij caps it at the sun_path socket-budget and rejects overflow with a cryptic "must be less than 0 characters" clap error, so a launch-time guard probes zellij's own validator via a no-op `action list-clients` and fails with a clear message instead — #54) — and forwards `-- <args>` to the agent like a plain `pair <agent> -- <args>`. It never reads `session_id` — that's `resume`'s job. On Alt+x, `cleanup_quit_marker` offers to **park** the session: preserve its scrollback (`.raw` + `.events.jsonl`) under a non-recyclable `parked-scrollback-<tag>-<ts>.*` so a live session can distill it later (no live agent exists at quit, so the nudge only *preserves*). The prompt is timeout-bounded — it auto-defaults to **N** (preserve nothing) after `PAIR_PARK_PROMPT_TIMEOUT` seconds (default 5, integer seam) so an unattended quit never blocks on it (#64). The park mechanics live in a shared `park_scrollback <tag> <agent> [--copy]` helper (#55, extracted from `cleanup_quit_marker` — ARCH-DRY): the quit path *moves* (session dying), the compaction path *copies* (the live `pair-wrap --scrollback-log` is still appending to `.raw`).
   244	
   245	**In-session compaction (#55, `Alt+Shift+C`).** `pair continue <slug>` is *context-aware*: run from a normal shell it fresh-starts (above); run from INSIDE its own live pane it **compacts** — copy-parks the scrollback as a recovery net, writes a restart marker carrying a new `continue=<slug>` field (same tag, `new_session=1`), and kills the session. The outer `bin/pair` then `handle_restart_marker`-re-execs `pair continue <slug> <agent> -- <args>` (now outside zellij → the fresh-start branch), so the session reincarnates under the same tag with a clean conversation seeded from the continuation. Detection is **ancestry-based** (`in_zellij_pane`) plus a `ZELLIJ_SESSION_NAME == pair-<tag>` confirmation — never `$ZELLIJ*` env alone, since cmux propagates those to sibling non-pair panes (a false positive would park+kill the wrong session). The branch sits *before* the `in_zellij_pane` guard (which otherwise rejects any in-pane `pair`), with `DATA_DIR` + `handle_restart_marker` hoisted above the guard so test seams can drive them. Seams (test-only): `PAIR_FORCE_IN_SESSION`, `PAIR_FAKE_IN_ZELLIJ`, `PAIR_KILL_CMD`, `PAIR_TEST_CALL`, `PAIR_REEXEC_CAPTURE`. The trigger is the `Alt+Shift+C` keybind (`Alt C` / `Ctrl Alt c` → `PairConfirmCompact` → an agent-agnostic prompt that **defers to the `continuation` datatype procedure** — flush-first, then write the continuation and run `pair continue <slug>` — rather than enumerating a section skeleton inline, so the prompt can't drift out of sync with the datatype; that drift was the bug pair#61 fixed); the outer process suppresses the Alt+x park nudge whenever a restart marker is pending (a restart isn't a quit).
   246	
   247	### Shared 🤖-marker annotation — `nvim/annotate.lua`
   248	
   249	**Why.** Both read-only viewers — the scrollback viewer (`Alt+/`, `nvim/scrollback.lua`) and the change-log viewer (`Alt+l`, `nvim/changelog.lua`) — want the same `Alt+q` affordance: drop a 🤖 comment/question on a line (`🤖[Y]`) or a selection (`🤖<X>[Y]`), and on quit ship the user-added markers to the draft pane (→ the agent) via the `scrollback-pending-<tag>.md` sidecar the draft picks up on `FocusGained`. Rather than duplicate ~400 lines, the marker subsystem lives once in `nvim/annotate.lua`, `dofile`'d by both viewers (same dir-relative load as `adapt.lua`, since each viewer launches with `nvim -u <viewer>.lua`). Filed #57 (split from #53).
   250	
   251	**Shape (ARCH-PURE).** A **pure core** — `find_markers_in_line` (byte-walk parser), the `>`/`]` escape-unescape (`esc_x`/`esc_y`/`unescape`, backslash-parity walk so a selection containing the delimiters survives), `strip_markers`, `marker_key`, `collect_markers_by_line` (the load-time baseline so only *newly-added* markers extract), `format_extraction`, `new_marker_count`, width helpers — is exposed on the module table and unit-tested directly in `nvim/annotate_test.lua` (no buffer, no IO, no mocks). A **thin IO/UI seam** — `M.attach{bufnr, pending_path, footer, source_label, quit_noun}` — wires the `Alt+q` keymaps, the floating prompt (`open_marker_prompt`), the read-only unlock→insert→relock rewrite, the `VimLeavePre`→sidecar `M.emit`, and the `M.confirm_quit` gate (confirms only when there are user-added markers / a footer comment to ship).
   252	
   253	**Per-viewer parameters.** `footer=true` adds the scrollback-only overall-comment affordance line (gated entirely by `footer_row` being nil otherwise); `source_label` tags each emitted quote `> [<label>] <quote>` so the agent can tell a change-log question from a raw-scrollback one — a *per-quote* prefix (not a header line) so the draft pickup's `\n> ` marker count (`init.lua` `pair_pickup_scrollback_pending`) stays faithful. `M.has_new_markers`/`M.on_reloaded` let a viewer with an async refresh (changelog) skip a destructive reload when the user has annotated since open. Both viewers consume annotate: `nvim/scrollback.lua` with `footer=true` + no `source_label` (byte-identical pre-#57 UX), and `nvim/changelog.lua` with `footer=false` + `source_label='change log'` + the `start_refresh` reload guard (`safe_reload` skips the distiller's line-replace when `has_new_markers`, so a marker added during the spinner survives — annotations win; the fresh log is on disk for the next `Alt+l`).
   254	
   255	**Viewer (`nvim/scrollback.lua`).** Plugin-free init loaded via `nvim -u`. On `BufReadPost`, an SGR state machine walks each line: peels every `\x1b[...m` escape, mutates a running state (fg/bg/bold/italic/underline/reverse/strike/blink), and emits an extmark span for each contiguous run of visible bytes under a single state. Color resolution: 30-37/90-97 fg + 40-47/100-107 bg map through an xterm-default palette; `38;5;n` indexed maps via the standard 256-color formula (16 anchored to the same palette, 16-231 = 6×6×6 cube, 232-255 = greyscale ramp); `38;2;r;g;b` uses RGB directly. State→hl-group cache is keyed by stringified attrs and uses an explicit counter (not `#hl_cache` — that's 0 on string-keyed tables, a bug caught during the test pass). Buffer is locked read-only (`modifiable = false`, `buftype = nofile`, no swapfile); only `<Esc>` quits via `<cmd>qa<CR>` — `q`, `ZZ`, `ZQ` are deliberately shadowed so a fat-fingered `q` (instead of `Alt+q` for the marker comment) can't slam the viewer shut and drop pending markers.
   256	
   257	`G` is a semi-live refresh affordance (#84): before jumping to EOF, the viewer derives sibling `.raw` / `.events.jsonl` paths from the current `.ansi`, reruns `pair-scrollback-render`, reloads the same buffer in place, reapplies ANSI extmarks, relocks the read-only options, and then lands at the refreshed bottom. If the user has pending `Alt+q` annotations or an overall footer comment, the render still updates the backing `.ansi` but the visible buffer is not destructively replaced; the next clean refresh or reopen will show the new snapshot after the comment is shipped. Render/read failures warn and keep the existing snapshot visible, so refresh never replaces usable scrollback with a broken buffer. This deliberately reuses the existing floating viewer instead of stacking another `pair-scrollback-open` pane.
   258	
   259	**Open (`bin/pair-scrollback-open`, POSIX sh).** Validates `PAIR_DATA_DIR` / `PAIR_TAG` / `PAIR_AGENT`, runs the renderer, then *launches* `nvim -u $PAIR_HOME/nvim/scrollback.lua $ANSI` as a child — deliberately **not** `exec`, so the script stays alive as nvim's parent and an `EXIT`/`INT`/`TERM` trap can clear the re-entrancy lock on quit. Errors print and `sleep` briefly so the message is readable before the floating pane self-closes. Bound in `zellij/config.kdl` to `Alt+/` as a 100% × 100% floating pane with `close_on_exit=true` — the user's `:q` in the viewer dismisses the pane and returns to pair's two-pane layout untouched. **Re-entrancy guard:** `Alt+/` is a global zellij bind, so pressing it again while the viewer is already focused fires another `Run` and would stack a second nvim (one `:q`/Esc per layer to unwind). zellij can't conditionally skip a `Run`, so the script self-guards: before launching nvim it writes its own PID to `$PAIR_DATA_DIR/scrollback-<tag>-<agent>.openlock`, and on entry it exits immediately if that lock already holds a *live* PID — the redundant floating pane then self-dismisses via `close_on_exit` and focus falls back to the open viewer. A stale lock (hard kill) carries a dead PID and is reclaimed by the next open's `kill -0` check. The draft pane's `Alt+b` (`--jump prev`) runs the same launcher, so it's covered too.
   260	
   261	**Jump-on-open shortcut — draft `Alt+b` = "Alt+/ then Alt+b".** `pair-scrollback-open` takes an optional `--jump prev|next`; it exports `PAIR_SCROLLBACK_JUMP` before launching nvim, and `scrollback.lua` calls `jump_to_prompt()` right after its normal viewport positioning — so the viewer opens already sitting on the previous (or next) user prompt, behaviourally identical to opening with Alt+/ and then pressing Alt+b. The draft pane's `Alt+b` (`nvim/init.lua`, `pair_scrollback_prev_prompt`) is the one-key trigger: it opens the same floating pane via `zellij run --floating … -- pair-scrollback-open --jump prev` (geometry mirrored from the `Alt+/` bind). Env-scoped rather than a sentinel file, so there's no staleness across plain `Alt+/` opens.
   262	
   263	**Comment markers — `Alt+q` in viewer → draft pickup (#000018).** While reading scrollback, `Alt+q` drops a parley-style `🤖[]` marker at the cursor (or `🤖<selection>[…]` in visual mode). The buffer is read-only, so the keymap lifts `modifiable`/`readonly` for the insert and re-locks immediately. (#57: this whole marker subsystem was extracted to the shared `nvim/annotate.lua` — the change-log viewer uses the identical flow; see "Shared 🤖-marker annotation" above.) On viewer exit (`VimLeavePre`), `nvim/annotate.lua` (`M.emit`) walks every line, parses each `🤖<X>?[Y]` marker by literal-byte scan (Lua patterns aren't UTF-8 aware), and writes a formatted block to `$PAIR_DATA_DIR/scrollback-pending-<tag>.md`:
   264	
   265	```
   266	> <X | line stripped of all markers>
   267	<Y>
   268	```
   269	
   270	The draft pane's `nvim/init.lua` registers a `FocusGained` autocmd that picks up the sidecar: on the `*` slot, it appends the block directly into the buffer and `:write`s (going through nvim_buf_set_lines, not an autoread + checktime dance, sidesteps the sub-second mtime resolution issue). Off-slot (`-N` / `+N`), it appends to `draft-<tag>.md` so the next nav-to-`*` reads it from disk. Sidecar is removed in both cases, and a `vim.notify` flashes "🤖 picked up N scrollback comment(s)". Round-trip: read scrollback → `Alt+q` to mark → `:q` → focus the draft → see the formatted block ready to send via `Alt+Return`.
   271	
   272	**Overall comment affordance (#000021).** After inline annotations, users often want a standalone summary not tied to any line. `BufReadPost` appends one trailing row — `For overall comment, Alt+q on this line.` — rendered in default Normal color (not dimmed; the affordance is positional, not visual). `nvim`'s `virt_lines` aren't cursor-navigable, so this is a real line. `Alt+q` on that row routes to `add_footer_comment` (no inline-quote context) and stores the text in `state[bufnr].footer_text`; the visible row becomes `Overall comment: <text>` and edits via the same chord. Empty submit clears, restoring the hint. `M.emit` strips the affordance row from the marker scan and appends the stored text as a trailing standalone block (no `> quote` prefix) in the sidecar. The Esc exit-confirm folds it into the prompt ("3 🤖[] markers + overall comment will be sent"). (#57: this footer flow now lives in `nvim/annotate.lua`, gated behind `attach{footer=true}` — so only the scrollback viewer shows the affordance; the change-log viewer attaches with `footer=false`.)
   273	
   274	### `nvim/init.lua` — drafting buffer config
   275	
   276	Loaded via `nvim -u`, fully isolated from the user's main nvim config. Provides:
   277	
   278	- Drafting-friendly defaults: no line numbers, wrap, linebreak, breakindent, spell, persistent undo under `~/.local/share/pair/undo/`, `cmdheight=0` to keep the cmdline out of the way, custom statusline (see "prompt history & queue" below).
   279	- `<M-CR>` (Alt+Return, normal+insert) — `send_and_clear`: append buffer to log, send to agent pane via `zellij action focus-pane-id` + `write-chars` + Enter, clear `*` (when source was `*`, or when a send from `+N` parked a non-empty draft into the queue — see "Prompt history & queue"), save, drop into insert mode.
   280	- `<S-M-CR>` (Alt+Shift+Return, normal+insert) — `send_and_clear(no_submit=true)`: identical flow (strip, log, queue handling, clear, reset) but writes a bare CR (`write 13`) instead of `ESC`+`CR` (`write 27 13`). pair-wrap rewrites a bare CR into the agent's insert-newline sequence rather than its submit byte, so the draft lands in the agent's composer on a fresh line, **unsubmitted** — append-without-send.
   281	- `<M-Left>` / `<M-Right>` — navigate the prompt-history / queue position one slot at a time (see below).
   282	- `<S-M-Left>` / `<S-M-Right>` — jump to the next region boundary (oldest history, newest history, `*`, front-of-queue, back-of-queue). Lets the user skip over long histories or queues without N taps.
   283	- `<M-b>` — `pair_scrollback_prev_prompt`: open the scrollback viewer already positioned on the previous agent-conversation prompt — a one-key shortcut for `Alt+/` then `Alt+b`. Shells out `zellij run --floating … -- pair-scrollback-open --jump prev`. See the scrollback section's "Jump-on-open shortcut".
   284	- `<M-q>` — push the current buffer to the front of the queue. From `*` also clears `*`; from `+N` it's move-to-front (removes the source queue file).
   285	- `<M-BS>` — delete the current `+N` queue item without sending, in **both normal and insert mode** (#62 — the gesture doesn't change meaning mid-edit); "stay-near" behavior (items behind shift down, position label keeps its number, so the next item is now under the cursor for repeat-delete). Off the queue (`*`/`-N`) it's a no-op in normal mode and a kill-to-line-start (`<C-U>`) in insert mode, so the line-kill editing convenience stays on the draft.
   286	- `<M-i>` (Alt+i, normal+insert) — `attach_image`: capture-driven image attach. 1) Verify the OS clipboard holds image data (macOS: AppleScript `clipboard info` enumerates `PNGf`/`TIFF`/etc.; Linux: `wl-paste --list-types` or `xclip -t TARGETS`) — if not, flash `[no image in clipboard]` as inline virt_text for 1s and bail. 2) Read pair-wrap's pid from `$DATA_DIR/pair-wrap-pid-<tag>` (notify+abort if missing/dead, since pair-wrap is the whole agent I/O path). 3) `kill -USR1 <pid>` to arm a ~200ms capture window in pair-wrap, then `zellij action write 22` to send Ctrl+V to the agent pane. 4) Poll `image-capture-<tag>.done` (20ms cadence, 600ms cap); on hit, read `image-capture-<tag>`, strip ANSI, regex `%[Image[ #][^%]]+%]` (matches both claude's `[Image #N]` and agy's `[Image N-M]`) and insert the captured marker verbatim at cursor. The agent is the source of truth for the marker text — no local counter, no per-agent format hardcoded.
   287	- `PairPasteQuote()` (global, called from `bin/clipboard-to-pane.sh` via `:lua PairPasteQuote()`): reads the raw selection from `$PAIR_DATA_DIR/quote-<tag>` and dispatches on cursor column.
   288	  - **col == 0 (`paste_as_quote`)**: par-reflow with width 1000, prefix every line with `> `; if the cursor's line is empty, replace it, else insert above (existing line slides down); scroll first inserted line to top via `zt`; cursor on a single empty line directly below the block in insert mode; flash the quoted lines with `IncSearch` (full-line, per-line `nvim_buf_add_highlight`).
   289	  - **col > 0 (`paste_inline`)**: par-reflow (so hard-wrapped sources collapse to one continuous run, paragraph breaks preserved), insert at the cursor via `nvim_buf_set_text` (handles multi-line splits); cursor at the end of the inserted span in insert mode; no scroll; flash the inserted span with a single multi-line extmark.
   290	  - In both modes the highlight is cleared 500ms later via `vim.defer_fn`. Selection-finalize visual cue (issue #12).
   291	- Autosave on `BufLeave`, `FocusLost`, `InsertLeave` so disk and buffer agree.
   292	- As-you-type fuzzy path completion (issue #13). `TextChangedI`/`TextChangedP` autocmd splits the trailing path token on the last `/` into `<dir>` + `<filter>`, lists `<dir>` via `getcompletion`, fuzzy-filters with built-in `matchfuzzy`, hands the result to `vim.fn.complete()`. Triggers only when the token contains `/` or starts with `~` (plain words stay quiet). `<Tab>`/`<S-Tab>` cycle; `<CR>` routes through the pure `cr_keys(visible, has_selection, momentary)` decision (#65): accepts the highlighted item (`<C-y>`) when one is selected, else — the common case under `completeopt=noselect` — dismisses the popup AND inserts a newline (`<C-e><CR>`; a bare `<CR>` while the menu is up is swallowed without a newline, so the explicit `<C-e>` cancel is required). The shared `<CR>` map passes `momentary=spell_popup_active`, so the transient normal-mode `z=` spell popup keeps its clean-dismiss contract (bare `<CR>`, no spurious newline) while as-you-type completions get the newline. Mouse clicks on menu items are intercepted via insert-mode `<LeftMouse>` mapping which calculates popup bounds (`pum_getpos()`) and mouse coordinates (`getmousepos()`) to select and confirm the clicked item instantly. Plugin-free.
   293	- **Completer chain (`run_completers`).** The `TextChangedI`/`TextChangedP` autocmd runs three completers in priority order, short-circuiting on the first that calls `complete()` (each returns `true` when it does): `path_complete` (slash/tilde tokens) → `word_complete` (draft-buffer words + agent-output spans) → `spell_complete` (spelling fixes). `spell_complete` is the as-you-type counterpart to the `z=` popup: at end-of-word, if the alphabetic word being typed (≥ `SPELL_TRIGGER_MIN`=4 chars) is flagged by `spellbadword`, it offers `spellsuggest` results as a plain (unlabelled) `complete()` menu — picked via CR/Tab/arrows. It deliberately skips the `indexed_items` ⌥N labels: the quick-pick keys don't function in this mid-type fallback (Alt+N is bound for path/word menus, bare digits stay literal), so advertising them would mislead. Being last in the chain, it fires only when path/word completion found nothing, so real completions are never crowded out. Unlike `spell_suggest_popup` it does **not** set `spell_popup_active` (we're mid-type — bare digits stay literal, and `CompleteDone` must not bounce to normal mode); it also bails when the cursor sits inside a word so `complete()`'s replace span can't strand a tail.
   294	- **Completion quick-pick + `z=` spell popup.** The first 9 completion items are abbr-tagged with a pick key (`indexed_items`, optional `label_prefix`). Insert-mode path/word completions tag `⌥1`…`⌥9` and pick via `<M-i>` (`pair_pick_completion` feeds `<C-n>`/`<C-p>` to land on item N, then `<C-y>`) — bare digits there stay literal text. The normal-mode `z=` spell popup (`spell_suggest_popup`) reuses the same menu but is a momentary "pick a fix" gesture, not a typing session: it tags plain `1`…`9` and lets **bare digits** pick (`spell_pick_digit`, gated on a `spell_popup_active` flag so the behavior never leaks into ordinary insert-mode typing). `z=` enters insert mode only to host the popup; `CompleteDone` `stopinsert`s back to normal mode on accept *or* dismiss, and `InsertLeave` clears the flag as a safety net.
   295	- All autocmds live in the `pair` augroup (`clear=true`), so iterating via `:luafile $PAIR_HOME/nvim/init.lua` reloads cleanly without duplicating handlers.
   296	- **Layout ladder** — `PairLayoutBigger` / `PairLayoutSmaller` derive the current rung from `vim.o.lines` (the kdl pins each rung to an exact size — 1 / 12 / 33% — so nvim's pane height is ground truth) and call `zellij action next-swap-layout` / `previous-swap-layout` accordingly. Reading actual height makes drift self-correcting: a silently-rejected swap can't desync state, since the next press recomputes from reality rather than a counter that was incremented optimistically. `pair_layout_state` mirrors the rung in-memory for callers like `pair_spinner_start` and `pair_ensure_visible_then` to check without re-reading; an on-disk copy at `${XDG_DATA_HOME:-~/.local/share}/pair/layout-mode-<tag>` is purely diagnostic. Landing in `minimized` also `MoveFocus`es up to the agent pane (the draft is unusable at 1 row) and the focus-grab spinner suppresses itself when `pair_layout_state == 'minimized'`.
   297	- **Statusline cheatsheet (right-aligned, progressive disclosure).** `PAIR_CHEATS` lists `Alt+h help`, `Alt+⏎ send`, `Alt+q queue`, `Alt+x quit`, `Alt+d detach` in priority order. `pair_compose_statusline` measures the variable left segment (history/queue/position cluster), reserves a 6-cell minimum gap, and accumulates as many cheat entries as fit in the remaining columns — Alt+h is always the last entry to drop. Spinner takes the right slot when active (vim only honors a single `%=` per statusline). The minimized rung shows a standalone "Alt+↑ for pair input box" hint instead, with 4 leading spaces so the terminal cursor (which lands on the statusline row when the buffer has zero visible lines) sits on whitespace rather than the hint text.
   298	- **Alt+x / Alt+d / Alt+n / Shift+Alt+N confirm modals.** `PairConfirmQuit` / `PairConfirmDetach` / `PairConfirmRestart` / `PairConfirmRestartNewSession` shell out to `pair-quit.sh` / `zellij action detach` / `pair-restart.sh` / `pair-restart.sh --new-session` after a Y/N modal that defaults to No. All four are wrapped in `pair_ensure_visible_then`, which auto-grows out of `minimized` (calls `PairLayoutBigger` and defers the modal 100ms) so the prompt renders on visible rows. The two restart modals share a single `pair_confirm_restart_impl(new_session)` helper.
   299	
   300	### Prompt history & queue (issue #000015)
   301	
   302	The nvim buffer is a virtual cursor over a sequence of slots:
   303	
   304	```
   305	[ -N ... -2  -1 ]   *   [ +1  +2 ... +M ]

exec
/bin/zsh -lc "nl -ba nvim/pair_poke.lua | sed -n '1,80p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	-- nvim/pair_poke.lua — inject an instruction into the agent pane from ANY pane
     2	-- (issue #66 M3). The draft's send_to_agent uses RELATIVE move-focus, which does
     3	-- NOT escape a floating pane (documented in scrollback.lua). The review pane
     4	-- resolves the agent's ABSOLUTE pane id from `zellij action list-panes --json`
     5	-- and writes directly to that pane id, so review focus never moves.
     6	-- Deliberately NO has_ui() short-circuit, so the headless test can stub `zellij`
     7	-- and record the calls.
     8	local M = {}
     9	
    10	local pair_nvim_dir = vim.fn.fnamemodify(debug.getinfo(1, 'S').source:sub(2), ':p:h')
    11	local zellij_trace = dofile(pair_nvim_dir .. '/zellij_trace.lua')
    12	
    13	-- Pure: the ordered zellij argv list for one poke. Submit is a semantic
    14	-- Alt+Enter key event so zellij delivers one modified chord to pair-wrap.
    15	function M._cmds(body, agent_id, review_id)
    16	  return {
    17	    { 'zellij', 'action', 'write-chars', '--pane-id', tostring(agent_id), body },
    18	    { 'zellij', 'action', 'send-keys', '--pane-id', tostring(agent_id), 'Alt Enter' },
    19	  }
    20	end
    21	
    22	-- Recursively collect pane objects (have both `id` and `is_floating`) from the
    23	-- decoded list-panes JSON — mirrors scrollback-open's `.. | objects | select`.
    24	local function collect_panes(node, out)
    25	  if type(node) ~= 'table' then return out end
    26	  if node.id ~= nil and node.is_floating ~= nil then out[#out + 1] = node end
    27	  for _, v in pairs(node) do
    28	    if type(v) == 'table' then collect_panes(v, out) end
    29	  end
    30	  return out
    31	end
    32	
    33	local function list_panes()
    34	  local res = zellij_trace.action('review.poke.list-panes', { 'zellij', 'action', 'list-panes', '--json' }).stdout
    35	  local ok, decoded = pcall(vim.json.decode, res)
    36	  if not ok or type(decoded) ~= 'table' then return {} end
    37	  return collect_panes(decoded, {})
    38	end
    39	
    40	-- The agent pane: a real terminal (not plugin), tiled (not floating), not the
    41	-- draft — the same predicate pair-scrollback-open uses to find it.
    42	local function find_agent(panes)
    43	  for _, p in ipairs(panes) do
    44	    if p.is_plugin == false and p.is_floating == false
    45	        and p.title ~= nil and p.title ~= '' and p.title ~= 'draft' then
    46	      return p.id
    47	    end
    48	  end
    49	end
    50	
    51	-- Send `body` to the agent without moving focus away from the caller.
    52	-- Returns false (+ notify) if the agent pane can't be resolved.
    53	function M.send(body)
    54	  local panes = list_panes()
    55	  local agent = find_agent(panes)
    56	  if not agent then
    57	    vim.notify('review: could not find the agent pane to poke', vim.log.levels.ERROR)
    58	    return false
    59	  end
    60	  local cmds = M._cmds(body, agent)
    61	  zellij_trace.action('review.poke.write-body', cmds[1], {
    62	    redact = { [6] = body },
    63	  })
    64	  zellij_trace.action('review.poke.submit', cmds[2])
    65	  return true
    66	end
    67	
    68	return M

exec
/bin/zsh -lc 'git diff --check 62e38b3b7cd38f2ba9e25bda4d9332180c13aa71..HEAD && bash tests/queue-send-test.sh && bash tests/review-poke-test.sh' in /Users/xianxu/workspace/pair
 succeeded in 4139ms:
  ok   send +3 w/ draft: CCC→history, draft parked at +1, no duplicate
  ok   send +2 w/ empty draft: BBB→history, nothing parked, no duplicate
  ok   send from *: HELLO→history, queue untouched
  ok   draft submit command uses semantic Alt Enter
  ok   draft append-only command stays newline without submit
queue-send-test: all passed
  ok   _cmds builds the id-based argv
  ok   writes the please-review body to agent pane
  ok   submits with semantic Alt+Enter to agent pane
  ok   does not change focus
  ok   no relative move-focus
  ok   writes zellij action trace
  ok   traces pane lookup
  ok   traces body write
  ok   records redacted body length
  ok   redacts review poke body
review-poke-test ok

codex
The focused tests pass locally. I’m doing one more API sanity check against the installed zellij help and the Lua test suite the issue claims, then I’ll write the verdict.
exec
/bin/zsh -lc 'zellij action send-keys --help' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
zellij-action-send-keys 
Send one or more keys to the terminal (e.g., "Ctrl a", "F1", "Alt Shift b")

USAGE:
    zellij action send-keys [OPTIONS] <KEYS>...

ARGS:
    <KEYS>...    Keys to send as space-separated strings

OPTIONS:
    -h, --help                 Print help information
    -p, --pane-id <PANE_ID>    The pane_id of the pane, eg. terminal_1, plugin_2 or 3 (equivalent
                               to terminal_3)

exec
/bin/zsh -lc "rg -n \"Alt Enter|\\x1b|27|send-keys|insert newline|submit\" cmd bin nvim tests zellij -S" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
tests/review-poke-test.sh:41:  and c[2][3] == 'send-keys' and c[2][4] == '--pane-id'
tests/review-poke-test.sh:42:  and c[2][5] == '7' and c[2][6] == 'Alt Enter'
tests/review-poke-test.sh:57:grep -q '^action send-keys --pane-id 7 Alt Enter$' "$ZLOG" && pass "submits with semantic Alt+Enter to agent pane" || fail "no pane-id submit"
bin/pair:148:                append nvim buffer to agent composer + newline, no submit;
bin/pair:149:                log + clear (like Alt+Return but unsubmitted)
nvim/scrollback_test.lua:131:        vim.fn.writefile({ '\27[31mnew red\27[0m', 'new tail' }, ansi)
nvim/annotate.lua:610:-- standalone footer, not tied to a line. Empty submit clears the comment;
nvim/annotate.lua:665:    vim.cmd('normal! \27')
nvim/annotate.lua:671:    vim.cmd('normal! \27')
nvim/annotate.lua:674:  vim.cmd('normal! \27')  -- exit visual so the upcoming prompt has focus
cmd/pair-wrap/translate_test.go:137:			in:      []byte("hi\x1b[127;3u"),
cmd/pair-wrap/translate_test.go:152:			in:       []byte("hi\x1b[127;3"),
cmd/pair-wrap/translate_test.go:154:			wantHold: []byte("\x1b[127;3"),
tests/pair-session-watch-test.sh:11:session_file="$RT/home/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-$sid.jsonl"
tests/queue-send-test.sh:74:local function dump_cmds(prefix, no_submit)
tests/queue-send-test.sh:79:  for _,cmd in ipairs(_G.PairDraftSendCommands('HELLO', no_submit)) do
tests/queue-send-test.sh:83:dump_cmds('submit', false)
tests/queue-send-test.sh:129:if has "C submit draft.send.submit zellij action send-keys Alt Enter"; then
tests/queue-send-test.sh:130:  pass "draft submit command uses semantic Alt Enter"
tests/queue-send-test.sh:132:  fail "draft submit command"; sed 's/^/    /' "$RT/result.txt"
tests/queue-send-test.sh:135:   && [ "$(count '^C append .*send-keys Alt Enter')" = "0" ]; then
tests/queue-send-test.sh:136:  pass "draft append-only command stays newline without submit"
nvim/slug.lua:2:-- (issue #000027). No vim API here, so it runs under `nvim -l` for tests
nvim/slug.lua:20:-- Soft policy (issue #000027): nvim trusts the proposer. The proposer only
cmd/pair-wrap/main.go:61:	slugDebounceS              = 1 * time.Second // min gap between pair-slug spawns (#000027)
cmd/pair-wrap/main.go:92:// ✻ = U+273B = 0xE2 0x9C 0xBB in UTF-8. Anchored on ✻ so the
cmd/pair-wrap/main.go:96:	"claude": regexp.MustCompile(`^\x{273B}\s*[A-Za-z]+\s+for\s+\d+[hms](?:\s+\d+[hms])*`),
cmd/pair-wrap/main.go:109://	Enter      = insert newline
cmd/pair-wrap/main.go:115:// mapping: incoming Enter becomes the agent's "insert newline" sequence,
cmd/pair-wrap/main.go:255:	// pair-slug spawn debounce (#000027)
cmd/pair-wrap/main.go:521:// slug (#000027 M3). Debounced by slugDebounceS so closely-spaced turn-end
cmd/pair-wrap/main.go:550:	// Turn-end is also when the orientation slug should refresh (#000027).
cmd/pair-wrap/main.go:1163://   - KKP: \x1b[127;3u (127 = backspace keycode, modifier param 3 = alt).
cmd/pair-wrap/main.go:1170:	altBSKKP    = []byte("\x1b[127;3u")
cmd/pair-wrap/main.go:1528:			// KKP Alt+Backspace: \x1b[127;3u → kill to line start (Ctrl+U).
cmd/pair-wrap/main.go:1738:	os.Stdout.WriteString("\x1b[7m" + text + "\x1b[27m\r\n\r\n")
tests/review-docflow-test.sh:78:OUT:write((r.unavailable == true and r.code == 127) and 'unavailable ok\n' or 'unavailable FAIL\n')
tests/review-docflow-test.sh:93:grep -q 'unavailable ok'     "$RT/unavail.out" && pass "docflow.lua flags unavailable for a missing binary (code 127)" || { fail "no unavailable flag"; cat "$RT/unavail.out"; }
tests/review-window-test.sh:291:  local post_submit_marks = vim.api.nvim_buf_get_extmarks(buf, apply.HL, 0, -1, {})
tests/review-window-test.sh:292:  local post_submit_diags = vim.diagnostic.get(buf, { namespace = apply.DIAG })
tests/review-window-test.sh:295:  OUT2:write((#post_submit_marks == 0 and #post_submit_diags == 0 and 'human-submit-clears-style\n')
tests/review-window-test.sh:296:    or ('NO-human-submit-clears-style marks=' .. #post_submit_marks .. ' diags=' .. #post_submit_diags .. '\n'))
tests/review-window-test.sh:313:  vim.api.nvim_buf_set_lines(buf, -1, -1, false, { 'menu submit edit' })
tests/review-window-test.sh:315:  if h then h.submit() end
tests/review-window-test.sh:361:grep -q '^human-submit-clears-style$' "$RT/r3" && pass "Alt+Return clears stale agent styling" || fail "human submit styling clear"
tests/review-window-test.sh:364:grep -q '^awaiting-statusline$' "$RT/r3" && pass "human submit switches statusline to awaiting spinner" || fail "awaiting statusline"
tests/review-window-test.sh:372:grep -q 'menu submit edit' "$REPO/doc.md" && pass "send menu submit saves the reviewed document buffer" || fail "send menu submit did not save reviewed document"
nvim/scrollback.lua:143:    elseif n == 27 then state.reverse = false
nvim/scrollback.lua:207:local SGR = '\27%[[%d;]*m'
nvim/scrollback.lua:377:--   claude — ❯  (U+276F, HEAVY RIGHT-POINTING ANGLE QUOTATION MARK)
nvim/review/menu.lua:51:  local on_submit = opts.on_submit or function() end
nvim/review/menu.lua:96:    title = ' One-round instruction - optional (M-CR/C-s submit · Tab/Esc→list) ',
nvim/review/menu.lua:128:  local function submit()
nvim/review/menu.lua:134:    on_submit({ mode = mode.name, instruction = instruction })
nvim/review/menu.lua:152:  lmap('<CR>', submit)
nvim/review/menu.lua:153:  lmap('<M-CR>', submit)
nvim/review/menu.lua:154:  lmap('<C-s>', submit)
nvim/review/menu.lua:166:  imap({ 'n', 'i' }, '<M-CR>', submit)
nvim/review/menu.lua:167:  imap({ 'n', 'i' }, '<C-s>', submit)
nvim/review/menu.lua:175:    submit = submit,
bin/pair-scrollback-open:116:            # positives, which scored 1/27 vs 24/27 in dev probe).
nvim/review/menu_test.lua:35:  on_submit = function() end,
nvim/pair_poke.lua:18:    { 'zellij', 'action', 'send-keys', '--pane-id', tostring(agent_id), 'Alt Enter' },
nvim/pair_poke.lua:64:  zellij_trace.action('review.poke.submit', cmds[2])
nvim/init.lua:695:  -- ESC = 0x1b = 27. Claude reads this as "interrupt current stream".
nvim/init.lua:698:  PairZellijTrace.action('draft.interrupt.esc', { 'zellij', 'action', 'write', '27' })
nvim/init.lua:702:local function draftSendCommands(body, no_submit)
nvim/init.lua:711:  if no_submit then
nvim/init.lua:714:    cmds[#cmds + 1] = { label = 'draft.send.submit', argv = { 'zellij', 'action', 'send-keys', 'Alt Enter' } }
nvim/init.lua:722:local function send_to_agent(body, no_submit)
nvim/init.lua:735:  -- are queued, not delivered — sending the submit immediately after
nvim/init.lua:737:  -- newline rather than submit. Settle for ~100ms in that case so the
nvim/init.lua:739:  -- before we hit submit. Single-line sends skip the wait.
nvim/init.lua:743:  -- "insert newline" sequence (claude: `\<Enter>`, codex/agy: \n),
nvim/init.lua:744:  -- so a bare CR here would insert a newline rather than submit. Use
nvim/init.lua:745:  -- zellij's semantic send-keys action for the modified chord instead
nvim/init.lua:748:  -- no_submit (Alt+Shift+Enter path): land the body in the agent's
nvim/init.lua:749:  -- composer followed by a literal newline but DON'T submit. A bare CR
nvim/init.lua:752:  -- *not* a submit — so it leaves the cursor on a fresh line in the
nvim/init.lua:755:  local cmds = draftSendCommands(body, no_submit)
nvim/init.lua:1143:  s = s:gsub('\27%[[%d;?]*[%a@-~]', '')
nvim/init.lua:1145:  s = s:gsub('\27%][^\7\27]*\7', '')
nvim/init.lua:1146:  s = s:gsub('\27%][^\7\27]*\27\\', '')
nvim/init.lua:1527:-- With no_submit=true (Alt+Shift+Return) the body lands in the agent's
nvim/init.lua:1528:-- composer followed by a literal newline but is NOT submitted; everything
nvim/init.lua:1532:local function send_and_clear(no_submit)
nvim/init.lua:1576:  send_to_agent(stripped, no_submit)
nvim/init.lua:2499:  if c == 27 then return cancel() end -- Esc
nvim/init.lua:3944:-- Auto-orientation slug — dispose side (issue #000027)
nvim/review/docflow.lua:22:    return { code = 127, unavailable = true, stdout = '',
nvim/slug_test.lua:22:local PROPOSED = '=== #27 auto | new focus ==='
nvim/slug_test.lua:34:  local a, p = M.decide('=== #27 auto | old focus ===', PROPOSED)
nvim/slug_test.lua:38:  local a = M.decide('=== #27 auto | my own words ===', PROPOSED)
nvim/slug_test.lua:68:    local b = mkbuf({ '=== #27 auto | old ===', 'my prompt line', 'second line' })
nvim/slug_test.lua:80:    local b = mkbuf({ '=== #27 auto | old ===', 'prompt', 'more here' })
nvim/review.lua:478:    on_submit = function(choice)
cmd/pair-slug/main.go:4:// notify point — so it works for claude/codex/agy alike (issue #000027 M3,
cmd/pair-changelog/main_test.go:14:const idleFooter = "\u276f \n\u2500\u2500\u2500\u2500\n\u23f5\u23f5 bypass permissions\n"
cmd/pair-slug/slug_test.go:20:		{"000027-auto-orientation-slug", "pair", "#000027 auto-orientation-slug"},

exec
/bin/zsh -lc 'make test-lua' in /Users/xianxu/workspace/pair
 succeeded in 251ms:
ok changelog_test
nvim -l nvim/adapt_test.lua
nvim/adapt.lua: emitter tests passed
nvim -l nvim/doctor_test.lua
all doctor.lua tests passed
nvim -l nvim/review/record_test.lua
record_test ok
nvim -l nvim/review/reconstruct_test.lua
reconstruct_test ok
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

codex
The implementation matches the issue’s functional contract and the focused tests plus `make test-lua` pass. The only material gap I see is documentation drift in `atlas/architecture.md`, which still describes the old submit mechanics for this existing architectural surface.
exec
/bin/zsh -lc "nl -ba tests/queue-send-test.sh | sed -n '68,142p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
    68	end
    69	local lh=io.open(dd..'/log-test.md'); local lc=lh and lh:read('*a') or ''; if lh then lh:close() end
    70	for body in (lc..'\n---\n'):gmatch('## %S+ %S+\n\n(.-)\n\n%-%-%-') do
    71	  O:write('L '..body..'\n')
    72	end
    73	local d=io.open(dd..'/draft-test.md'); O:write('D '..((d and d:read('*a') or ''):gsub('%s+\$',''))..'\n'); if d then d:close() end
    74	local function dump_cmds(prefix, no_submit)
    75	  if type(_G.PairDraftSendCommands) ~= 'function' then
    76	    O:write('C '..prefix..' missing\n')
    77	    return
    78	  end
    79	  for _,cmd in ipairs(_G.PairDraftSendCommands('HELLO', no_submit)) do
    80	    O:write('C '..prefix..' '..cmd.label..' '..table.concat(cmd.argv, ' ')..'\n')
    81	  end
    82	end
    83	dump_cmds('submit', false)
    84	dump_cmds('append', true)
    85	O:close(); vim.cmd('qall!')  -- force-quit the dirtied throwaway buffer (#60)
    86	LUA
    87	}
    88	
    89	# assert that result.txt contains an exact line
    90	has() { grep -qxF "$1" "$RT/result.txt"; }
    91	# assert NO line starting with prefix matches value (duplication guard)
    92	count() { grep -cE "$1" "$RT/result.txt" || true; }
    93	
    94	echo "queue-send-test:"
    95	
    96	# --- 1. THE BUG: send +3 with a non-empty draft. ----------------------------
    97	# Expect: draft parked at front (+1), CCC removed from queue and now sole
    98	# history entry, NO CCC left in the queue.
    99	setup "DRAFT-WIP" "AAA" "BBB" "CCC"
   100	write_driver 3
   101	run
   102	if has "L CCC" && [ "$(count '^Q .*=CCC')" = "0" ] && has "Q 499999=DRAFT-WIP" \
   103	   && has "Q 500000=AAA" && has "Q 500001=BBB"; then
   104	  pass "send +3 w/ draft: CCC→history, draft parked at +1, no duplicate"
   105	else
   106	  fail "send +3 w/ draft (see result below)"; sed 's/^/    /' "$RT/result.txt"
   107	fi
   108	
   109	# --- 2. send from a queue slot with an EMPTY draft: no spurious enqueue. -----
   110	setup "" "AAA" "BBB" "CCC"
   111	write_driver 2   # → +2 (BBB)
   112	run
   113	if has "L BBB" && [ "$(count '^Q .*=BBB')" = "0" ] && has "Q 500000=AAA" \
   114	   && has "Q 500002=CCC" && [ "$(count '^Q ')" = "2" ]; then
   115	  pass "send +2 w/ empty draft: BBB→history, nothing parked, no duplicate"
   116	else
   117	  fail "send +2 w/ empty draft"; sed 's/^/    /' "$RT/result.txt"
   118	fi
   119	
   120	# --- 3. send straight from * (no queue interaction). ------------------------
   121	setup "HELLO" "AAA"
   122	write_driver 0
   123	run
   124	if has "L HELLO" && has "Q 500000=AAA" && [ "$(count '^Q ')" = "1" ]; then
   125	  pass "send from *: HELLO→history, queue untouched"
   126	else
   127	  fail "send from *"; sed 's/^/    /' "$RT/result.txt"
   128	fi
   129	if has "C submit draft.send.submit zellij action send-keys Alt Enter"; then
   130	  pass "draft submit command uses semantic Alt Enter"
   131	else
   132	  fail "draft submit command"; sed 's/^/    /' "$RT/result.txt"
   133	fi
   134	if has "C append draft.send.newline zellij action write 13" \
   135	   && [ "$(count '^C append .*send-keys Alt Enter')" = "0" ]; then
   136	  pass "draft append-only command stays newline without submit"
   137	else
   138	  fail "draft append-only command"; sed 's/^/    /' "$RT/result.txt"
   139	fi
   140	
   141	if [ "$fails" -ne 0 ]; then
   142	  echo "queue-send-test: $fails failure(s)"; exit 1

exec
/bin/zsh -lc "nl -ba tests/review-poke-test.sh | sed -n '35,65p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
    35	local poke = dofile(ROOT .. '/nvim/pair_poke.lua')
    36	local OUT = io.open(os.getenv('RESULT'), 'w')
    37	-- unit: pure _cmds shape
    38	local c = poke._cmds('hello', 7, 9)
    39	local ok_cmds = c[1][3] == 'write-chars' and c[1][4] == '--pane-id'
    40	  and c[1][5] == '7' and c[1][6] == 'hello'
    41	  and c[2][3] == 'send-keys' and c[2][4] == '--pane-id'
    42	  and c[2][5] == '7' and c[2][6] == 'Alt Enter'
    43	OUT:write(ok_cmds and 'cmds ok\n' or 'cmds FAIL\n')
    44	-- send resolves agent (7) and writes by pane id without changing focus.
    45	poke.send('updated, please review foo.md')
    46	OUT:write('sent\n'); OUT:close()
    47	LUA
    48	
    49	PATH="$RT/bin:$PATH" PAIR_ROOT="$ROOT" RESULT="$RESULT" PAIR_DATA_DIR="$RT" PAIR_TAG=poke \
    50	  run_headless --timeout 30 -- nvim --headless -u NONE -c "luafile $RT/driver.lua" -c 'qa!'
    51	
    52	fails=0
    53	pass() { printf '  ok   %s\n' "$1"; }
    54	fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }
    55	grep -q 'cmds ok' "$RESULT" && pass "_cmds builds the id-based argv" || fail "_cmds shape"
    56	grep -q '^action write-chars --pane-id 7 updated, please review foo.md$' "$ZLOG" && pass "writes the please-review body to agent pane" || fail "no pane-id write-chars body"
    57	grep -q '^action send-keys --pane-id 7 Alt Enter$' "$ZLOG" && pass "submits with semantic Alt+Enter to agent pane" || fail "no pane-id submit"
    58	grep -q 'focus-pane-id' "$ZLOG" && fail "changed focus while poking agent" || pass "does not change focus"
    59	grep -q 'move-focus' "$ZLOG" && fail "used relative move-focus (must be id-based)" || pass "no relative move-focus"
    60	TRACE="$RT/zellij-actions-poke.jsonl"
    61	test -s "$TRACE" && pass "writes zellij action trace" || fail "missing zellij action trace"
    62	grep -q '"label":"review.poke.list-panes"' "$TRACE" && pass "traces pane lookup" || fail "missing list-panes trace"
    63	grep -q '"label":"review.poke.write-body"' "$TRACE" && pass "traces body write" || fail "missing write-body trace"
    64	grep -q '"body_len":29' "$TRACE" && pass "records redacted body length" || fail "missing body length"
    65	if grep -q 'updated, please review foo.md' "$TRACE"; then

exec
/bin/zsh -lc "nl -ba workshop/plans/000086-fix-alt-return-draft-submit-plan.md | sed -n '15,48p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
    15	### Pure Entities
    16	
    17	| Name | Lives in | Status |
    18	|------|----------|--------|
    19	| `draftSendCommands` | `nvim/init.lua` | new |
    20	| `pair_poke._cmds` | `nvim/pair_poke.lua` | modified |
    21	
    22	- **draftSendCommands** — pure argv builder for the draft-pane zellij action sequence.
    23	  - **Relationships:** 1:1 with one draft send; consumed by `send_to_agent`; exposed as `_G.PairDraftSendCommands` for headless tests.
    24	  - **DRY rationale:** Keeps submit vs append-only command selection testable without duplicating expectations inside the IO path.
    25	  - **Future extensions:** If another draft action needs the same delivery sequence, widen the builder rather than adding ad hoc zellij commands.
    26	
    27	- **pair_poke._cmds** — pure argv builder for review-pane agent pokes.
    28	  - **Relationships:** 1:1 with one review poke; consumed by `pair_poke.send`.
    29	  - **DRY rationale:** The review-poke submit command should use the same semantic zellij action as the draft send path.
    30	  - **Future extensions:** If zellij changes key naming, update the single command builder and matching draft path.
    31	
    32	### Integration Points
    33	
    34	| Name | Lives in | Status | Wraps |
    35	|------|----------|--------|-------|
    36	| `draft send_to_agent` | `nvim/init.lua` | modified | `zellij action` |
    37	| `review poke send` | `nvim/pair_poke.lua` | modified | `zellij action --pane-id` |
    38	| `queue-send fake zellij assertion` | `tests/queue-send-test.sh` | modified | fake zellij process |
    39	| `review-poke fake zellij assertion` | `tests/review-poke-test.sh` | modified | fake zellij process |
    40	
    41	- **draft send_to_agent** — gets commands from `draftSendCommands`, executes them only when a UI is attached, and redacts body writes in the existing trace path. Submit uses semantic `send-keys "Alt Enter"`; `no_submit` remains raw `write 13` because that path intentionally asks pair-wrap for insert-newline.
    42	  - **Injected into:** Existing draft `<M-CR>` and `<S-M-CR>` maps.
    43	  - **Future extensions:** Any other modified-key submit should use `send-keys`.
    44	
    45	- **review poke send** — writes directly to the agent pane id and submits using `send-keys --pane-id <id> "Alt Enter"` without changing focus.
    46	  - **Injected into:** Review workbench poke flow.
    47	  - **Future extensions:** Can share a tiny helper if a third nvim integration needs the same submit action.
    48	

codex
VERDICT: FIX-THEN-SHIP (confidence: high)

The runtime change fulfills ariadne#86: draft submit and review-poke submit now use zellij’s semantic `send-keys "Alt Enter"` path, append-only still uses raw `write 13`, and the new pure seams are covered by focused headless tests. No Critical issues found. The one gap is atlas drift: `atlas/architecture.md` still documents the old raw `ESC`+`CR` submit behavior for this existing draft surface, so the atlas update gate is not satisfied.

1. Strengths:
- `nvim/init.lua:702` adds `draftSendCommands` as a pure command builder, and `send_to_agent` stays a thin UI/zellij shell at `nvim/init.lua:755`.
- `nvim/init.lua:714` correctly changes draft submit to `zellij action send-keys "Alt Enter"` while `nvim/init.lua:712` preserves append-only `write 13`.
- `nvim/pair_poke.lua:15` keeps review poke command construction pure and id-based; `nvim/pair_poke.lua:18` uses `send-keys --pane-id <agent> "Alt Enter"` without focus movement.
- `tests/queue-send-test.sh:129` and `tests/queue-send-test.sh:134` pin both submit and append-only behavior, including the “no accidental submit” case.
- `tests/review-poke-test.sh:55` through `tests/review-poke-test.sh:59` cover command shape, pane-id write, semantic submit, and no focus movement.

2. Critical findings:
- None.

3. Important findings:
- `atlas/architecture.md:279` and `atlas/architecture.md:280` still describe draft submit/append behavior in terms of old raw Enter / `ESC`+`CR` mechanics. Update this section to say `<M-CR>` submits via `zellij action send-keys "Alt Enter"` and `<S-M-CR>` remains raw `write 13`. This is an atlas update gate miss for the changed nvim/Zellij submit surface.

4. Minor findings:
- None.

5. Test coverage notes:
- Verified locally: `git diff --check 62e38b3b7cd38f2ba9e25bda4d9332180c13aa71..HEAD`, `bash tests/queue-send-test.sh`, `bash tests/review-poke-test.sh`, and `make test-lua`.
- `zellij action send-keys --help` confirms the installed CLI supports modified key strings such as `"Alt Shift b"` and accepts `<KEYS>...`.

6. Architectural notes for upcoming work:
- `ARCH-DRY`: Pass. The two command builders serve distinct delivery paths; no harmful copy-paste beyond the shared key string.
- `ARCH-PURE`: Pass. Command selection is pure and tested without real zellij; IO remains in `PairZellijTrace.action` / `zellij_trace.action`.
- `ARCH-PURPOSE`: Pass for runtime behavior. The shadow sweep found both active consumers of raw submit (`draft send_to_agent`, `review poke`) converted; only atlas documentation remains stale.

7. Plan revision recommendations:
- None. The plan’s Core Concepts table matches the implemented code.
tokens used
50,315
VERDICT: FIX-THEN-SHIP (confidence: high)

The runtime change fulfills ariadne#86: draft submit and review-poke submit now use zellij’s semantic `send-keys "Alt Enter"` path, append-only still uses raw `write 13`, and the new pure seams are covered by focused headless tests. No Critical issues found. The one gap is atlas drift: `atlas/architecture.md` still documents the old raw `ESC`+`CR` submit behavior for this existing draft surface, so the atlas update gate is not satisfied.

1. Strengths:
- `nvim/init.lua:702` adds `draftSendCommands` as a pure command builder, and `send_to_agent` stays a thin UI/zellij shell at `nvim/init.lua:755`.
- `nvim/init.lua:714` correctly changes draft submit to `zellij action send-keys "Alt Enter"` while `nvim/init.lua:712` preserves append-only `write 13`.
- `nvim/pair_poke.lua:15` keeps review poke command construction pure and id-based; `nvim/pair_poke.lua:18` uses `send-keys --pane-id <agent> "Alt Enter"` without focus movement.
- `tests/queue-send-test.sh:129` and `tests/queue-send-test.sh:134` pin both submit and append-only behavior, including the “no accidental submit” case.
- `tests/review-poke-test.sh:55` through `tests/review-poke-test.sh:59` cover command shape, pane-id write, semantic submit, and no focus movement.

2. Critical findings:
- None.

3. Important findings:
- `atlas/architecture.md:279` and `atlas/architecture.md:280` still describe draft submit/append behavior in terms of old raw Enter / `ESC`+`CR` mechanics. Update this section to say `<M-CR>` submits via `zellij action send-keys "Alt Enter"` and `<S-M-CR>` remains raw `write 13`. This is an atlas update gate miss for the changed nvim/Zellij submit surface.

4. Minor findings:
- None.

5. Test coverage notes:
- Verified locally: `git diff --check 62e38b3b7cd38f2ba9e25bda4d9332180c13aa71..HEAD`, `bash tests/queue-send-test.sh`, `bash tests/review-poke-test.sh`, and `make test-lua`.
- `zellij action send-keys --help` confirms the installed CLI supports modified key strings such as `"Alt Shift b"` and accepts `<KEYS>...`.

6. Architectural notes for upcoming work:
- `ARCH-DRY`: Pass. The two command builders serve distinct delivery paths; no harmful copy-paste beyond the shared key string.
- `ARCH-PURE`: Pass. Command selection is pure and tested without real zellij; IO remains in `PairZellijTrace.action` / `zellij_trace.action`.
- `ARCH-PURPOSE`: Pass for runtime behavior. The shadow sweep found both active consumers of raw submit (`draft send_to_agent`, `review poke`) converted; only atlas documentation remains stale.

7. Plan revision recommendations:
- None. The plan’s Core Concepts table matches the implemented code.
