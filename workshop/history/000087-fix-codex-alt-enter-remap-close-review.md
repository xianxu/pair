# Boundary Review — ariadne#87 (whole-issue close)

| field | value |
|-------|-------|
| issue | 87 — Fix Codex Alt Enter remap |
| repo | pair |
| issue file | workshop/issues/000087-fix-codex-alt-enter-remap.md |
| boundary | whole-issue close |
| milestone | — |
| window | cc2cb6da1dea4c6c5422267bb74bcff34ba92343..HEAD |
| command | sdlc close --issue 87 |
| reviewer | codex |
| timestamp | 2026-06-29T18:34:00-07:00 |
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
session id: 019f1628-1a30-7ea0-ad69-0a02724b73cb
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary — a
milestone close, or a whole-issue close — for ariadne#87.

Base: cc2cb6da1dea4c6c5422267bb74bcff34ba92343   Head: HEAD

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
index e9e577d..1c5fd8d 100644
--- a/atlas/architecture.md
+++ b/atlas/architecture.md
@@ -178,7 +178,7 @@ Alt+x leaves the draft, queue, and history intact — the next session resumes t
 
      **Stdin raw mode.** The wrapper switches its stdin (zellij's pane PTY) into termios raw mode for the duration. Without this the kernel's line discipline does local echo + canonical buffering on the bytes flowing toward the wrapped TUI, which double-echoes keystrokes and corrupts terminal-response sequences. Saved/restored in a `finally` block.
 
-     **Stdin Enter remap (per-agent).** `sendKeymapByAgent` (`cmd/pair-wrap/main.go`) translates the user's Enter / Alt+Enter to per-agent send/newline bytes so the convention matches pair's nvim draft pane (Enter = newline, Alt+Enter = send). For `claude` the user's plain Enter becomes `\<CR>` (claude's portable "insert newline" sequence); Alt+Enter becomes a bare `\r` (send). For Codex / agy, pair sends LF for plain Enter (their newline gesture) and CR for Alt+Enter (send). The same keymap carries `altBS`: Alt+Backspace (legacy `\x1b\x7f` or KKP `\x1b[127;3u`, the same two-protocol shape as Alt+Enter) rewrites to **Ctrl+U** (`0x15`, kill-to-line-start) for every agent — so Alt+Delete in the agent pane matches the agent's Cmd+Delete and the draft pane's Alt+Delete. A lone `0x7f` (plain Backspace) isn't ESC-prefixed, so it passes through untouched. Opt out of the whole remap with `PAIR_WRAP_REMAP_RETURN=0`.
+     **Stdin Enter remap (per-agent).** `sendKeymapByAgent` (`cmd/pair-wrap/main.go`) translates the user's Enter / Alt+Enter to per-agent send/newline bytes so the convention matches pair's nvim draft pane (Enter = newline, Alt+Enter = send). For `claude` the user's plain Enter becomes `\<CR>` (claude's portable "insert newline" sequence); Alt+Enter becomes a bare `\r` (send). For Codex, pair sends LF for plain Enter (newline) and preserves Alt+Enter as `ESC CR` (submit chord). For agy, pair sends LF for plain Enter and CR for Alt+Enter (send). The same keymap carries `altBS`: Alt+Backspace (legacy `\x1b\x7f` or KKP `\x1b[127;3u`, the same two-protocol shape as Alt+Enter) rewrites to **Ctrl+U** (`0x15`, kill-to-line-start) for every agent — so Alt+Delete in the agent pane matches the agent's Cmd+Delete and the draft pane's Alt+Delete. A lone `0x7f` (plain Backspace) isn't ESC-prefixed, so it passes through untouched. Opt out of the whole remap with `PAIR_WRAP_REMAP_RETURN=0`.
 
      **Stdout filtering and batching (Codex).** Codex inline mode emits DEC synchronized-output markers (`ESC[?2026h` / `ESC[?2026l`) around frequent redraw batches. It can also enable terminal focus-event mode (`ESC[?1004h`) even though pair/zellij do not use focus events for the agent pane. `pair-wrap` strips those markers from the stdout stream sent to zellij, because zellij scrollback/mouse scrolling can behave poorly while a pane is in synchronized-output or extra terminal-event modes during generation. The filtered, user-visible stdout stream is then queued and flushed to zellij on a 100ms cadence (plus EOF) to lower redraw pressure from dense Codex repaint bursts (#85). The raw scrollback log remains immediate and unfiltered so forensic replay still captures the agent's original PTY stream and offset-keyed resize/time events stay aligned.
 
diff --git a/atlas/how-to-bring-up-a-new-harness-cli.md b/atlas/how-to-bring-up-a-new-harness-cli.md
index 157ed3b..639f37f 100644
--- a/atlas/how-to-bring-up-a-new-harness-cli.md
+++ b/atlas/how-to-bring-up-a-new-harness-cli.md
@@ -22,9 +22,13 @@ By default, the bottom Neovim draft pane maps **Enter** to insert a newline, and
           plainCR: []byte{'\n'}, // plain Enter inserts newline
           altCR:   []byte{'\r'}, // Alt+Enter sends query
       },
+      "codex": {
+          plainCR: []byte{'\n'},       // plain Enter inserts newline
+          altCR:   []byte{'\x1b','\r'}, // Alt+Enter submit chord
+      },
   }
   ```
-- **Note:** Claude uses `\<Enter>` (`[]byte{'\\', '\r'}`) as a newline, while Codex and Antigravity (`agy`) use LF (`\n`) for newline and CR (`\r`) for send.
+- **Note:** Claude uses `\<Enter>` (`[]byte{'\\', '\r'}`) as a newline. Codex uses LF (`\n`) for newline and preserves Alt+Enter as `ESC CR` for submit. Antigravity (`agy`) uses LF (`\n`) for newline and CR (`\r`) for send.
 
 **Telemetry Signal** (aspect `1`, see §3): `return-remap` — `fired` each time a plain Enter is remapped to the agent's newline; `bypass` each time it passes through as a bare `\r` while an overlay is active. Emitted from `emitPlainCR`. The `fired:bypass` ratio is the health signal; an all-`bypass` or zero-`fired` session means the remap stopped engaging.
 
diff --git a/cmd/pair-wrap/keymap_registry_test.go b/cmd/pair-wrap/keymap_registry_test.go
index 520c534..9d51739 100644
--- a/cmd/pair-wrap/keymap_registry_test.go
+++ b/cmd/pair-wrap/keymap_registry_test.go
@@ -21,8 +21,8 @@ func TestSendKeymapByAgent_RegistrationTable(t *testing.T) {
 		// claude reads `\<Enter>` as newline regardless of terminal
 		// keyboard-protocol level — the documented portable path.
 		"claude": {[]byte{'\\', '\r'}, []byte{'\r'}, ctrlU},
-		// codex: Enter = send (\r), Shift+Enter = newline (\n).
-		"codex": {[]byte{'\n'}, []byte{'\r'}, ctrlU},
+		// codex: plain Enter inserts newline; Alt+Enter stays a modified submit chord.
+		"codex": {[]byte{'\n'}, []byte{'\x1b', '\r'}, ctrlU},
 		// agy: same as codex.
 		"agy": {[]byte{'\n'}, []byte{'\r'}, ctrlU},
 	}
diff --git a/cmd/pair-wrap/main.go b/cmd/pair-wrap/main.go
index e17c422..ee01ffd 100644
--- a/cmd/pair-wrap/main.go
+++ b/cmd/pair-wrap/main.go
@@ -113,7 +113,7 @@ var spanExtractionAgents = map[string]bool{
 // is jarring when the user moves between panes. When PAIR_WRAP_REMAP_RETURN
 // isn't "0", pair-wrap rewrites stdin so the agent receives the inverted
 // mapping: incoming Enter becomes the agent's "insert newline" sequence,
-// incoming Alt+Enter becomes a plain Enter (send).
+// incoming Alt+Enter becomes the agent's submit sequence.
 //
 //   - plainCR:   bytes emitted when the user hits Enter alone (\r)
 //   - altCR:     bytes emitted when the user hits Alt+Enter (\x1b\r)
@@ -142,13 +142,11 @@ var sendKeymapByAgent = map[string]sendKeymap{
 		altBS:   []byte{0x15}, // Ctrl+U — kill to line start
 	},
 	"codex": {
-		// Codex follows the textbook chat-UI convention: Enter = send,
-		// Shift+Enter = newline. Under Ghostty's KKP level-1
-		// negotiation, Shift+Enter comes through as a literal LF
-		// (\n) and plain Enter stays as \r. Probed via
-		// PAIR_WRAP_LOG=… PAIR_WRAP_REMAP_RETURN=0.
+		// Codex plain Enter inserts a newline when rewritten to LF.
+		// Current Codex submit requires the modified Alt+Enter chord
+		// itself, so preserve that as legacy ESC+CR.
 		plainCR: []byte{'\n'},
-		altCR:   []byte{'\r'},
+		altCR:   []byte{'\x1b', '\r'},
 		altBS:   []byte{0x15}, // Ctrl+U — kill to line start
 	},
 	"agy": {
diff --git a/cmd/pair-wrap/translate_test.go b/cmd/pair-wrap/translate_test.go
index f58acfb..e4b3859 100644
--- a/cmd/pair-wrap/translate_test.go
+++ b/cmd/pair-wrap/translate_test.go
@@ -156,14 +156,12 @@ func TestTranslateChunk(t *testing.T) {
 	}
 
 	t.Run("codex keymap", func(t *testing.T) {
-		px := &proxy{sendKM: sendKeymap{
-			plainCR: []byte{'\n'},
-			altCR:   []byte{'\r'},
-		}}
+		px := &proxy{sendKM: sendKeymapByAgent["codex"]}
 		cases := []struct{ in, want []byte }{
-			{[]byte("hi\r"), []byte("hi\n")},     // Enter → newline
-			{[]byte("hi\x1b\r"), []byte("hi\r")}, // Alt+Enter → send
-			{[]byte("a\rb\x1b\r"), []byte("a\nb\r")},
+			{[]byte("hi\r"), []byte("hi\n")},             // Enter → newline
+			{[]byte("hi\x1b\r"), []byte("hi\x1b\r")},     // legacy Alt+Enter → submit chord
+			{[]byte("hi\x1b[13;3u"), []byte("hi\x1b\r")}, // KKP Alt+Enter → submit chord
+			{[]byte("a\rb\x1b\r"), []byte("a\nb\x1b\r")},
 			{[]byte("\x1b[200~text\rmore\x1b[201~"), []byte("\x1b[200~text\rmore\x1b[201~")}, // paste untouched
 		}
 		for _, c := range cases {
diff --git a/workshop/plans/000087-fix-codex-alt-enter-remap-plan.md b/workshop/plans/000087-fix-codex-alt-enter-remap-plan.md
new file mode 100644
index 0000000..692768f
--- /dev/null
+++ b/workshop/plans/000087-fix-codex-alt-enter-remap-plan.md
@@ -0,0 +1,140 @@
+# Fix Codex Alt Enter Remap Implementation Plan
+
+> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.
+
+**Goal:** Make draft Alt+Return submit in current Codex sessions by forwarding Codex's Alt+Enter submit chord through pair-wrap instead of collapsing it to bare Enter.
+
+**Architecture:** Keep the fix in the existing pure keymap table and translator tests: pair-wrap already owns the user-key to agent-key adaptation, so Codex's changed submit contract belongs there (`ARCH-PURPOSE`). Do not add nvim timing delays or Zellij workarounds because the live trace proves those boundaries delivered the expected event (`ARCH-DRY`, `ARCH-PURE`).
+
+**Tech Stack:** Go pair-wrap translator, existing Go unit tests, Markdown atlas docs.
+
+---
+
+## Core Concepts
+
+### Pure Entities
+
+| Name | Lives in | Status |
+|------|----------|--------|
+| `sendKeymapByAgent["codex"]` | `cmd/pair-wrap/main.go` | modified |
+| `translateChunk` | `cmd/pair-wrap/main.go` | modified contract via keymap data |
+
+- **sendKeymapByAgent["codex"]** — the per-agent byte contract for translating pair's Enter/Alt+Enter convention into the wrapped agent's expected stdin bytes.
+  - **Relationships:** 1:1 with agent basename `codex`; consumed by each `proxy` at startup.
+  - **DRY rationale:** One table row feeds both legacy and KKP input shapes through the existing translator, avoiding duplicate special cases.
+  - **Future extensions:** If Codex exposes a different submit chord later, update this row and the registry tests.
+
+- **translateChunk** — pure byte-stream translator that recognizes plain Enter, legacy Alt+Enter, and KKP Alt+Enter.
+  - **Relationships:** N:1 input chunks to output chunks; receives `sendKM` from the proxy.
+  - **DRY rationale:** The translator already handles protocol shape; the Codex-specific output should remain data-driven.
+  - **Future extensions:** Add new input protocols here only if Zellij/terminals emit a new shape.
+
+### Integration Points
+
+| Name | Lives in | Status | Wraps |
+|------|----------|--------|-------|
+| pair-wrap stdin pump | `cmd/pair-wrap/main.go` | unchanged | stdin and child PTY |
+| draft nvim submit | `nvim/init.lua` | unchanged | `zellij action send-keys "Alt Enter"` |
+
+- **pair-wrap stdin pump** — reads Zellij-delivered bytes, calls `translateChunk`, and writes translated bytes to the Codex PTY.
+  - **Injected into:** `translateChunk` via the `sendKM` field set at startup.
+  - **Future extensions:** If live instrumentation needs richer byte labels, add trace metadata without changing translation semantics.
+
+- **draft nvim submit** — sends the modified key event to pair-wrap. It remains unchanged because live trace proves it now emits the expected Alt+Enter input.
+  - **Injected into:** Zellij/pair-wrap through the existing action path.
+  - **Future extensions:** None for this issue.
+
+## Chunk 1: Codex Submit Chord
+
+### Task 1: Pin the Codex keymap contract
+
+**Files:**
+- Modify: `cmd/pair-wrap/keymap_registry_test.go`
+- Modify: `cmd/pair-wrap/translate_test.go`
+
+- [x] **Step 1: Write failing tests**
+
+Update `TestSendKeymapByAgent_RegistrationTable` so the Codex row expects:
+
+```go
+"codex": {[]byte{'\n'}, []byte{'\x1b', '\r'}, ctrlU},
+```
+
+Update the Codex subtest in `TestTranslateChunk` so:
+
+- `[]byte("hi\x1b\r")` translates to `[]byte("hi\x1b\r")`.
+- `[]byte("hi\x1b[13;3u")` translates to `[]byte("hi\x1b\r")`.
+- plain `[]byte("hi\r")` still translates to `[]byte("hi\n")`.
+
+- [x] **Step 2: Run tests to verify RED**
+
+Run:
+
+```bash
+go test ./cmd/pair-wrap
+```
+
+Expected: FAIL on Codex `altCR` and Codex Alt+Enter translation, because production still emits bare `CR`.
+
+- [x] **Step 3: Implement minimal keymap change**
+
+In `cmd/pair-wrap/main.go`, change only the Codex row:
+
+```go
+"codex": {
+  plainCR: []byte{'\n'},
+  altCR:   []byte{'\x1b', '\r'},
+  altBS:   []byte{0x15},
+},
+```
+
+Do not change Claude or agy.
+
+- [x] **Step 4: Run tests to verify GREEN**
+
+Run:
+
+```bash
+go test ./cmd/pair-wrap
+```
+
+Expected: PASS.
+
+### Task 2: Sync documentation
+
+**Files:**
+- Modify: `atlas/architecture.md`
+- Modify: `atlas/how-to-bring-up-a-new-harness-cli.md`
+- Modify: comments in `cmd/pair-wrap/main.go`, `cmd/pair-wrap/keymap_registry_test.go`, `cmd/pair-wrap/translate_test.go`
+
+- [x] **Step 1: Update stale text**
+
+Replace statements that say Codex Alt+Enter maps to bare `CR` with the current contract: plain Enter maps to LF newline; Alt+Enter is forwarded as the modified submit chord (`ESC CR`).
+
+- [x] **Step 2: Verify no stale Codex keymap docs remain**
+
+Run:
+
+```bash
+rg -n "codex.*Alt\\+Enter|Codex.*Alt\\+Enter|Alt\\+Enter.*codex|bare `?\\\\r|bare CR|plain Enter" cmd/pair-wrap atlas -S
+```
+
+Expected: Any remaining hits should either describe Claude/overlay behavior or the new Codex contract.
+
+### Task 3: Final verification and close
+
+**Files:**
+- Modify: `workshop/issues/000087-fix-codex-alt-enter-remap.md`
+- Modify: `workshop/plans/000087-fix-codex-alt-enter-remap-plan.md`
+
+- [x] Run:
+
+```bash
+go test ./cmd/pair-wrap
+bash tests/queue-send-test.sh
+bash tests/review-poke-test.sh
+git diff --check
+sdlc issue validate workshop/issues/000087-fix-codex-alt-enter-remap.md
+```
+
+- [x] Mark plan and issue checkboxes, log RED/GREEN evidence, close through `sdlc close`, then ship via `sdlc pr` / `sdlc merge`.


codex
I’ll run this as a fresh boundary review: first I’m loading the required workflow skill and local SDLC context, then I’ll verify the diff against the issue/plan and source rather than relying on the pasted summary.
exec
/bin/zsh -lc "sed -n '1,220p' /Users/xianxu/workspace/pair/workshop/lessons.md" in /Users/xianxu/workspace/pair
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

codex
I’ve got the workflow contract. Now I’m locating issue #87 and checking the actual filesystem/diff at `HEAD`, including whether the plan’s entity table and atlas/doc updates line up with the code.
exec
/bin/zsh -lc 'git status --short --branch' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
## 000087-fix-codex-alt-enter-remap
 M workshop/issues/000087-fix-codex-alt-enter-remap.md

exec
/bin/zsh -lc 'rg -n "000087|fix-codex-alt-enter|Codex Alt" workshop atlas cmd tests -S' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
workshop/issues/000087-fix-codex-alt-enter-remap.md:2:id: 000087
workshop/issues/000087-fix-codex-alt-enter-remap.md:13:# Fix Codex Alt Enter remap
workshop/issues/000087-fix-codex-alt-enter-remap.md:33:- [x] Codex Alt+Enter translation emits `ESC CR`.
workshop/issues/000087-fix-codex-alt-enter-remap.md:37:- [x] Atlas/docs no longer claim Codex Alt+Enter maps to bare `CR`.
workshop/issues/000087-fix-codex-alt-enter-remap.md:41:- [x] Update the pure keymap tests first to expect Codex `altCR == ESC CR` and to expect both legacy and KKP Codex Alt+Enter inputs to translate to `ESC CR`.
workshop/issues/000087-fix-codex-alt-enter-remap.md:63:- 2026-06-29: closed — go test ./cmd/pair-wrap; bash tests/queue-send-test.sh; bash tests/review-poke-test.sh; git diff --check; sdlc issue validate workshop/issues/000087-fix-codex-alt-enter-remap.md
workshop/issues/000087-fix-codex-alt-enter-remap.md:67:Implemented the Codex keymap correction with TDD. RED: `go test ./cmd/pair-wrap` failed on `codex.altCR: got "\r", want "\x1b\r"` and on both legacy/KKP Codex Alt+Enter translation cases. GREEN: changed only `sendKeymapByAgent["codex"].altCR` to `ESC CR`; `go test ./cmd/pair-wrap` passed. Synced atlas/docs to distinguish Codex from agy and verified with `go test ./cmd/pair-wrap`, `bash tests/queue-send-test.sh`, `bash tests/review-poke-test.sh`, `git diff --check`, and `sdlc issue validate workshop/issues/000087-fix-codex-alt-enter-remap.md`.
workshop/plans/000087-fix-codex-alt-enter-remap-plan.md:1:# Fix Codex Alt Enter Remap Implementation Plan
workshop/plans/000087-fix-codex-alt-enter-remap-plan.md:77:Expected: FAIL on Codex `altCR` and Codex Alt+Enter translation, because production still emits bare `CR`.
workshop/plans/000087-fix-codex-alt-enter-remap-plan.md:112:Replace statements that say Codex Alt+Enter maps to bare `CR` with the current contract: plain Enter maps to LF newline; Alt+Enter is forwarded as the modified submit chord (`ESC CR`).
workshop/plans/000087-fix-codex-alt-enter-remap-plan.md:127:- Modify: `workshop/issues/000087-fix-codex-alt-enter-remap.md`
workshop/plans/000087-fix-codex-alt-enter-remap-plan.md:128:- Modify: `workshop/plans/000087-fix-codex-alt-enter-remap-plan.md`
workshop/plans/000087-fix-codex-alt-enter-remap-plan.md:137:sdlc issue validate workshop/issues/000087-fix-codex-alt-enter-remap.md

exec
/bin/zsh -lc 'git diff --stat cc2cb6da1dea4c6c5422267bb74bcff34ba92343..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 atlas/architecture.md                              |   2 +-
 atlas/how-to-bring-up-a-new-harness-cli.md         |   6 +-
 cmd/pair-wrap/keymap_registry_test.go              |   4 +-
 cmd/pair-wrap/main.go                              |  12 +-
 cmd/pair-wrap/translate_test.go                    |  12 +-
 .../issues/000087-fix-codex-alt-enter-remap.md     |  43 ++++++-
 .../plans/000087-fix-codex-alt-enter-remap-plan.md | 140 +++++++++++++++++++++
 7 files changed, 198 insertions(+), 21 deletions(-)

exec
/bin/zsh -lc 'git diff --name-only cc2cb6da1dea4c6c5422267bb74bcff34ba92343..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
atlas/architecture.md
atlas/how-to-bring-up-a-new-harness-cli.md
cmd/pair-wrap/keymap_registry_test.go
cmd/pair-wrap/main.go
cmd/pair-wrap/translate_test.go
workshop/issues/000087-fix-codex-alt-enter-remap.md
workshop/plans/000087-fix-codex-alt-enter-remap-plan.md

exec
/bin/zsh -lc "sed -n '1,220p' workshop/issues/000087-fix-codex-alt-enter-remap.md" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "sed -n '1,220p' workshop/plans/000087-fix-codex-alt-enter-remap-plan.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
---
id: 000087
status: done
deps: []
github_issue:
created: 2026-06-29
updated: 2026-06-29
estimate_hours: 0.46
started: 2026-06-29T18:25:25-07:00
actual_hours: 0.13
---

# Fix Codex Alt Enter remap

## Problem

After #86, restarted sessions do invoke `zellij action send-keys "Alt Enter"` from the draft pane, but Codex still leaves the inserted draft text sitting in the composer. Live trace for `PAIR_TAG=2` at 2026-06-29T18:23:10-07:00 shows:

- nvim wrote the body via `draft.send.write-body` (`body_len: 41`).
- nvim invoked `draft.send.submit` as `zellij action send-keys Alt Enter`.
- pair-wrap read the body bytes, then read Alt+Enter as `ESC CR` (`raw_len: 2`, SHA `a6d286d70768`), translated it to bare `CR` (`translated_len: 1`, SHA `9d1e0e2d9459`), and wrote that to the Codex PTY.

So the Zellij action and pair-wrap Alt+Enter recognition work; the stale assumption is the Codex keymap row that collapses Alt+Enter to bare `CR`. Current Codex needs the modified Alt+Enter chord forwarded to the child PTY.

## Spec

For Codex only, pair-wrap should preserve the submit chord when it sees user Alt+Enter: legacy `ESC CR` and KKP `ESC [ 13 ; 3 u` should translate to `ESC CR` for the Codex child PTY, not bare `CR`. Plain Enter should continue translating to LF so the pair convention remains Enter = newline. Claude and agy keymaps should remain unchanged.

Docs that describe Codex's keymap should say Codex plain Enter maps to newline and Alt+Enter is forwarded as the submit chord.

## Done when

- [x] Codex Alt+Enter translation emits `ESC CR`.
- [x] Codex plain Enter still emits LF.
- [x] Claude and agy keymaps are unchanged.
- [x] Tests cover both legacy and KKP Alt+Enter inputs for Codex.
- [x] Atlas/docs no longer claim Codex Alt+Enter maps to bare `CR`.

## Plan

- [x] Update the pure keymap tests first to expect Codex `altCR == ESC CR` and to expect both legacy and KKP Codex Alt+Enter inputs to translate to `ESC CR`.
- [x] Change only `sendKeymapByAgent["codex"].altCR` in `cmd/pair-wrap/main.go`.
- [x] Update Codex keymap docs in `atlas/architecture.md`, `atlas/how-to-bring-up-a-new-harness-cli.md`, and comments/tests that state the old bare-CR contract.
- [x] Verify `go test ./cmd/pair-wrap`, focused nvim submit tests, issue validation, and whitespace.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.10 impl=0.12
item: atlas-docs design=0.00 impl=0.08
item: milestone-review design=0.00 impl=0.15
design-buffer: 0.10
total: 0.46
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.

## Log

### 2026-06-29
- 2026-06-29: closed — go test ./cmd/pair-wrap; bash tests/queue-send-test.sh; bash tests/review-poke-test.sh; git diff --check; sdlc issue validate workshop/issues/000087-fix-codex-alt-enter-remap.md

User restarted after #86 and still saw Alt+Return insert text into Codex without submitting. Investigated live traces instead of guessing. `zellij-actions-2.jsonl` confirmed the restarted draft pane is using `send-keys Alt Enter`; `wrap-events-2.jsonl` confirmed pair-wrap receives that as Alt+Enter and currently translates it to bare `CR`. Because bare `CR` reached Codex and did not submit, the stale contract is the Codex keymap's `altCR` output, not nvim focus, Zellij delivery, or pair-wrap input recognition (`ARCH-PURPOSE`).

Implemented the Codex keymap correction with TDD. RED: `go test ./cmd/pair-wrap` failed on `codex.altCR: got "\r", want "\x1b\r"` and on both legacy/KKP Codex Alt+Enter translation cases. GREEN: changed only `sendKeymapByAgent["codex"].altCR` to `ESC CR`; `go test ./cmd/pair-wrap` passed. Synced atlas/docs to distinguish Codex from agy and verified with `go test ./cmd/pair-wrap`, `bash tests/queue-send-test.sh`, `bash tests/review-poke-test.sh`, `git diff --check`, and `sdlc issue validate workshop/issues/000087-fix-codex-alt-enter-remap.md`.

 succeeded in 0ms:
# Fix Codex Alt Enter Remap Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make draft Alt+Return submit in current Codex sessions by forwarding Codex's Alt+Enter submit chord through pair-wrap instead of collapsing it to bare Enter.

**Architecture:** Keep the fix in the existing pure keymap table and translator tests: pair-wrap already owns the user-key to agent-key adaptation, so Codex's changed submit contract belongs there (`ARCH-PURPOSE`). Do not add nvim timing delays or Zellij workarounds because the live trace proves those boundaries delivered the expected event (`ARCH-DRY`, `ARCH-PURE`).

**Tech Stack:** Go pair-wrap translator, existing Go unit tests, Markdown atlas docs.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `sendKeymapByAgent["codex"]` | `cmd/pair-wrap/main.go` | modified |
| `translateChunk` | `cmd/pair-wrap/main.go` | modified contract via keymap data |

- **sendKeymapByAgent["codex"]** — the per-agent byte contract for translating pair's Enter/Alt+Enter convention into the wrapped agent's expected stdin bytes.
  - **Relationships:** 1:1 with agent basename `codex`; consumed by each `proxy` at startup.
  - **DRY rationale:** One table row feeds both legacy and KKP input shapes through the existing translator, avoiding duplicate special cases.
  - **Future extensions:** If Codex exposes a different submit chord later, update this row and the registry tests.

- **translateChunk** — pure byte-stream translator that recognizes plain Enter, legacy Alt+Enter, and KKP Alt+Enter.
  - **Relationships:** N:1 input chunks to output chunks; receives `sendKM` from the proxy.
  - **DRY rationale:** The translator already handles protocol shape; the Codex-specific output should remain data-driven.
  - **Future extensions:** Add new input protocols here only if Zellij/terminals emit a new shape.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| pair-wrap stdin pump | `cmd/pair-wrap/main.go` | unchanged | stdin and child PTY |
| draft nvim submit | `nvim/init.lua` | unchanged | `zellij action send-keys "Alt Enter"` |

- **pair-wrap stdin pump** — reads Zellij-delivered bytes, calls `translateChunk`, and writes translated bytes to the Codex PTY.
  - **Injected into:** `translateChunk` via the `sendKM` field set at startup.
  - **Future extensions:** If live instrumentation needs richer byte labels, add trace metadata without changing translation semantics.

- **draft nvim submit** — sends the modified key event to pair-wrap. It remains unchanged because live trace proves it now emits the expected Alt+Enter input.
  - **Injected into:** Zellij/pair-wrap through the existing action path.
  - **Future extensions:** None for this issue.

## Chunk 1: Codex Submit Chord

### Task 1: Pin the Codex keymap contract

**Files:**
- Modify: `cmd/pair-wrap/keymap_registry_test.go`
- Modify: `cmd/pair-wrap/translate_test.go`

- [x] **Step 1: Write failing tests**

Update `TestSendKeymapByAgent_RegistrationTable` so the Codex row expects:

```go
"codex": {[]byte{'\n'}, []byte{'\x1b', '\r'}, ctrlU},
```

Update the Codex subtest in `TestTranslateChunk` so:

- `[]byte("hi\x1b\r")` translates to `[]byte("hi\x1b\r")`.
- `[]byte("hi\x1b[13;3u")` translates to `[]byte("hi\x1b\r")`.
- plain `[]byte("hi\r")` still translates to `[]byte("hi\n")`.

- [x] **Step 2: Run tests to verify RED**

Run:

```bash
go test ./cmd/pair-wrap
```

Expected: FAIL on Codex `altCR` and Codex Alt+Enter translation, because production still emits bare `CR`.

- [x] **Step 3: Implement minimal keymap change**

In `cmd/pair-wrap/main.go`, change only the Codex row:

```go
"codex": {
  plainCR: []byte{'\n'},
  altCR:   []byte{'\x1b', '\r'},
  altBS:   []byte{0x15},
},
```

Do not change Claude or agy.

- [x] **Step 4: Run tests to verify GREEN**

Run:

```bash
go test ./cmd/pair-wrap
```

Expected: PASS.

### Task 2: Sync documentation

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `atlas/how-to-bring-up-a-new-harness-cli.md`
- Modify: comments in `cmd/pair-wrap/main.go`, `cmd/pair-wrap/keymap_registry_test.go`, `cmd/pair-wrap/translate_test.go`

- [x] **Step 1: Update stale text**

Replace statements that say Codex Alt+Enter maps to bare `CR` with the current contract: plain Enter maps to LF newline; Alt+Enter is forwarded as the modified submit chord (`ESC CR`).

- [x] **Step 2: Verify no stale Codex keymap docs remain**

Run:

```bash
rg -n "codex.*Alt\\+Enter|Codex.*Alt\\+Enter|Alt\\+Enter.*codex|bare `?\\\\r|bare CR|plain Enter" cmd/pair-wrap atlas -S
```

Expected: Any remaining hits should either describe Claude/overlay behavior or the new Codex contract.

### Task 3: Final verification and close

**Files:**
- Modify: `workshop/issues/000087-fix-codex-alt-enter-remap.md`
- Modify: `workshop/plans/000087-fix-codex-alt-enter-remap-plan.md`

- [x] Run:

```bash
go test ./cmd/pair-wrap
bash tests/queue-send-test.sh
bash tests/review-poke-test.sh
git diff --check
sdlc issue validate workshop/issues/000087-fix-codex-alt-enter-remap.md
```

- [x] Mark plan and issue checkboxes, log RED/GREEN evidence, close through `sdlc close`, then ship via `sdlc pr` / `sdlc merge`.

exec
/bin/zsh -lc "sed -n '1,260p' cmd/pair-wrap/main.go" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
// pair-wrap — transparent PTY proxy around a TUI coding agent.
//
// Installed at bin/pair-wrap and invoked by zellij/layouts/main.kdl on
// pair startup. (Originally ported from a Python prototype, #000011; the
// Python original was retired in #000019.)
//
// What it does:
//   - Spawns the agent in a fresh pty so the wrapper sees the raw output.
//   - Forwards stdin → agent and agent → stdout transparently.
//   - On agent OSC 9 / OSC 777 (and optional bare BEL), writes OSC 9
//     directly to pair's recorded outer-TTY — bypassing zellij, which
//     would otherwise eat the OSC.
//   - Per-agent notify mode: native (forward agent's OSC), idle (after
//     no output for IDLE_S), or marker (on first sighting of an
//     end-of-turn regex over extracted colored spans).
//   - SGR span extraction: per-foreground-color byte-level state machine
//     building an LRU of 1000 unique colored spans, written atomically
//     to agent-output-<tag> for nvim's autocomplete pickup.
//   - Optional --scrollback-log <path> tee with .events.jsonl sidecar
//     recording resize events keyed by byte offset — feeds Alt+/.
//   - Image-paste capture: SIGUSR1 arms a 0.9s window that buffers agent
//     output, writes image-capture-<tag>, touches .done sentinel.
//   - Startup banner; per-feature debug log via PAIR_WRAP_LOG.
//
// Failure mode: any error in detection / emission / capture is logged
// (when PAIR_WRAP_LOG is set) and swallowed. The proxy never blocks the
// agent on a logging hiccup.
package main

import (
	"bytes"
	"container/list"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
	"golang.org/x/term"

	"github.com/xianxu/pair/cmd/internal/adapt"
)

// ----- Tunables ---------------------------------------------------------------

const (
	rateLimitS                 = 500 * time.Millisecond
	slugDebounceS              = 1 * time.Second // min gap between pair-slug spawns (#000027)
	agentOutputSpansMax        = 1000
	agentSpanMax               = 512
	rollingTailLen             = 512
	pendingMax                 = 64 // cap on incomplete-escape carryover
	readBufSize                = 4096
	defaultStdoutFlushInterval = 100 * time.Millisecond
)

var (
	// IDLE_S default — overridable via PAIR_WRAP_IDLE_S. 0 disables.
	defaultIdleS = 60 * time.Second

	// Image capture window — overridable via PAIR_WRAP_CAPTURE_S.
	defaultCaptureWindow = 900 * time.Millisecond
)

// Per-agent notify mode (mode that owns emit_outer for that agent run).
// Anything not listed here uses notifyModeDefault.
var notifyMode = map[string]string{
	"claude": "marker",
}

const notifyModeDefault = "native"

// Per-agent end-of-turn pattern, applied only in "marker" notify mode.
// Matched against finalized colored spans (post-SGR-stripping by the
// span extractor). The Python regex (raw bytes form):
//
//	rb"^\xe2\x9c\xbb\s*[A-Za-z]+\s+for\s+\d+[hms](?:\s+\d+[hms])*"
//
// ✻ = U+273B = 0xE2 0x9C 0xBB in UTF-8. Anchored on ✻ so the
// quoted-history form ("> ✻ Churned for 21s", different color) won't
// double-emit. Durations accept multiple `\d+[hms]` parts: 1m 52s, 2h 13m 4s, etc.
var endOfTurnByAgent = map[string]*regexp.Regexp{
	"claude": regexp.MustCompile(`^\x{273B}\s*[A-Za-z]+\s+for\s+\d+[hms](?:\s+\d+[hms])*`),
}

// Agents we trust the colored-span extractor to handle. Outside this set,
// span extraction is disabled (autocomplete falls back to draft alone,
// and "marker" notify mode becomes a no-op for that agent — a caller
// config error worth logging but not crashing on).
var spanExtractionAgents = map[string]bool{
	"claude": true,
}

// Per-agent stdin keymap. The pair-managed nvim draft uses
//
//	Enter      = insert newline
//	Alt+Enter  = send to agent
//
// but the agent's native TUI typically uses Enter = send. That mismatch
// is jarring when the user moves between panes. When PAIR_WRAP_REMAP_RETURN
// isn't "0", pair-wrap rewrites stdin so the agent receives the inverted
// mapping: incoming Enter becomes the agent's "insert newline" sequence,
// incoming Alt+Enter becomes the agent's submit sequence.
//
//   - plainCR:   bytes emitted when the user hits Enter alone (\r)
//   - altCR:     bytes emitted when the user hits Alt+Enter (\x1b\r)
//   - altBS:     bytes emitted when the user hits Alt+Backspace (\x1b\x7f)
//
// Claude reads "\<Enter>" (backslash + CR) as a newline regardless of
// terminal keyboard-protocol support — this is the documented portable
// path. Other agents need their own probing; leave them out of the table
// to fall through to no-rewrite (today's pass-through behavior).
//
// altBS maps Alt+Backspace to Ctrl+U (kill-to-line-start) so Alt+Delete
// in the agent pane matches the agent's existing Cmd+Delete and the
// draft pane's Alt+Delete. Ctrl+U is the universal readline kill, so it's
// the same byte for every agent.
type sendKeymap struct {
	plainCR, altCR []byte
	altBS          []byte
}

var sendKeymapByAgent = map[string]sendKeymap{
	"claude": {
		// Claude reads `\<Enter>` as newline regardless of terminal
		// keyboard-protocol level — the documented portable path.
		plainCR: []byte{'\\', '\r'},
		altCR:   []byte{'\r'},
		altBS:   []byte{0x15}, // Ctrl+U — kill to line start
	},
	"codex": {
		// Codex plain Enter inserts a newline when rewritten to LF.
		// Current Codex submit requires the modified Alt+Enter chord
		// itself, so preserve that as legacy ESC+CR.
		plainCR: []byte{'\n'},
		altCR:   []byte{'\x1b', '\r'},
		altBS:   []byte{0x15}, // Ctrl+U — kill to line start
	},
	"agy": {
		// Antigravity (agy) follows the same Enter/Shift+Enter convention
		// as codex.
		plainCR: []byte{'\n'},
		altCR:   []byte{'\r'},
		altBS:   []byte{0x15}, // Ctrl+U — kill to line start
	},
}

type overlayDetector func(*proxy, []byte, []byte) (bool, string)

var overlayDetectorByAgent = map[string]overlayDetector{
	"claude": detectClaudeOverlayOpen,
	"codex":  detectCodexOverlayOpen,
	"agy":    detectAgyOverlayOpen,
}

// ----- Compiled regexes (byte-mode) -------------------------------------------

var (
	sgrRe         = regexp.MustCompile(`\x1b\[([0-9;]*)m`)
	otherEscRe    = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[()*+][@-~]|\x1b[@-Z\\-_]`)
	imageMarkerRe = regexp.MustCompile(`\[Image[ #][^\]]+\]`)
	oscRe         = regexp.MustCompile(`\x1b\](\d+);([^\x07\x1b]*)(?:\x07|\x1b\\)`)
)

// ----- State ------------------------------------------------------------------

// proxy holds all mutable wrapper state. Fields touched only from the main
// loop don't need locking; the few touched from signal goroutines (capture
// window, notify-mode flags) are guarded explicitly.
type proxy struct {
	// CLI / config
	scrollbackLog  string
	agentBasename  string
	debugLogPath   string
	wrapEventsPath string
	bellFallback   bool

	// Resolved paths (empty when env didn't provide PAIR_TAG)
	outerTTYFile    string
	agentOutputFile string
	captureOutPath  string
	captureDonePath string
	capturePIDPath  string
	agentPIDPath    string

	// PTY
	ptmx *os.File
	cmd  *exec.Cmd

	// Notify
	notifyModeActive string
	endOfTurnRe      *regexp.Regexp
	idleS            time.Duration

	// Stdin Return-key remap. Zero-value (empty plainCR + altCR) means
	// pass-through. Populated from sendKeymapByAgent unless the user
	// opts out via PAIR_WRAP_REMAP_RETURN=0.
	sendKM sendKeymap

	// pickerActive is set when the active agent's output stream signals
	// that a blocking overlay / picker opened. While set,
	// translateChunk emits a bare \r for the user's plain Enter
	// instead of the textarea-aware remap, so the overlay confirms.
	// The flag clears after the first plain Enter is consumed —
	// restoring normal remap for the next Enter, which is back in the
	// textarea. Set from masterPump (handleChunk), read+cleared from
	// translateChunk → atomic.
	pickerActive atomic.Bool

	// Codex does not expose a dedicated overlay OSC today, so its
	// detector watches newly arrived visible text plus this carryover for
	// split picker labels. Keeping it separate from the OSC rolling tail
	// avoids re-detecting stale picker text after Enter clears
	// pickerActive.
	overlayMu       sync.Mutex
	overlayTextTail string

	// Adaptation flight recorder: always-on, appends one JSON line per
	// adaptation trigger to adapt-<tag>.jsonl so pair-doctor can spot drift.
	// nil is a safe no-op (telemetry never blocks the proxy). lastNearMiss
	// dedups repeated overlay near-misses across rerenders — touched only
	// from the handleChunk pump goroutine, so it needs no lock.
	adapt        *adapt.Logger
	lastNearMiss string
	// filterSeen dedups the aspect-5 output-filter signal: we log `fired`
	// once per distinct stripped marker (presence is the signal, and the
	// markers fire many times per turn). Touched only from the stdout pump
	// goroutine (stripCodexOutputMarkers), so no lock.
	filterSeen map[string]bool

	// Scrollback log (-1 / nil when disabled)
	scrollbackFD    *os.File
	eventsFD        *os.File
	scrollbackBytes int64
	lastTimeEvent   time.Time        // last logged scrollback "time" event (#59)
	now             func() time.Time // clock seam; defaults to time.Now (#59)

	// OSC rate limiting
	lastEmit time.Time
	// pair-slug spawn debounce (#000027)
	lastSlug time.Time

	// Span LRU. spans maps key="<color>\t<text>" → *spanEntry; order keeps
	// insertion order, oldest at Front, newest at Back. Move-to-back on
	// re-emission. Cap by popping from Front when size > limit.
	spans     map[string]*spanEntry
	spanOrder *list.List

exec
/bin/zsh -lc "sed -n '1,260p' cmd/pair-wrap/translate_test.go" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
package main

import (
	"bytes"
	"testing"
)

func TestTranslateChunk(t *testing.T) {
	p := &proxy{sendKM: sendKeymap{
		plainCR: []byte{'\\', '\r'},
		altCR:   []byte{'\r'},
		altBS:   []byte{0x15},
	}}

	tests := []struct {
		name      string
		in        []byte
		startPase bool
		wantOut   []byte
		wantHold  []byte
		wantPaste bool
	}{
		{
			name:    "plain text passes through",
			in:      []byte("hello world"),
			wantOut: []byte("hello world"),
		},
		{
			name:    "Enter becomes backslash-Enter",
			in:      []byte("hi\r"),
			wantOut: []byte("hi\\\r"),
		},
		{
			name:    "Alt+Enter becomes plain Enter",
			in:      []byte("hi\x1b\r"),
			wantOut: []byte("hi\r"),
		},
		{
			name:    "mixed: Enter and Alt+Enter in same chunk",
			in:      []byte("a\rb\x1b\rc\r"),
			wantOut: []byte("a\\\rb\rc\\\r"),
		},
		{
			name:      "bracketed paste preserves embedded \\r",
			in:        []byte("\x1b[200~line1\rline2\r\x1b[201~"),
			wantOut:   []byte("\x1b[200~line1\rline2\r\x1b[201~"),
			wantPaste: false, // ends out of paste mode
		},
		{
			name:      "Enter after paste end gets rewritten",
			in:        []byte("\x1b[200~x\r\x1b[201~\r"),
			wantOut:   []byte("\x1b[200~x\r\x1b[201~\\\r"),
			wantPaste: false,
		},
		{
			name:      "paste start, mid-paste chunk",
			in:        []byte("\x1b[200~pasted text\r"),
			wantOut:   []byte("\x1b[200~pasted text\r"),
			wantPaste: true,
		},
		{
			name:      "paste continues into chunk, ends",
			startPase: true,
			in:        []byte("more\rstuff\x1b[201~Enter\r"),
			wantOut:   []byte("more\rstuff\x1b[201~Enter\\\r"),
			wantPaste: false,
		},
		{
			name:     "trailing ESC alone is held back",
			in:       []byte("hi\x1b"),
			wantOut:  []byte("hi"),
			wantHold: []byte("\x1b"),
		},
		{
			name:     "trailing partial bpStart held back",
			in:       []byte("hi\x1b[20"),
			wantOut:  []byte("hi"),
			wantHold: []byte("\x1b[20"),
		},
		{
			name:      "trailing partial bpEnd inside paste held back",
			startPase: true,
			in:        []byte("data\x1b[20"),
			wantOut:   []byte("data"),
			wantHold:  []byte("\x1b[20"),
			wantPaste: true,
		},
		{
			name:    "ESC followed by non-CR non-[200 is passed through ESC",
			in:      []byte("hi\x1b[A"), // arrow up
			wantOut: []byte("hi\x1b[A"),
		},
		{
			name:    "KKP plain Enter becomes backslash-Enter",
			in:      []byte("hi\x1b[13u"),
			wantOut: []byte("hi\\\r"),
		},
		{
			name:    "KKP plain Enter (explicit no-modifier) becomes backslash-Enter",
			in:      []byte("hi\x1b[13;1u"),
			wantOut: []byte("hi\\\r"),
		},
		{
			name:    "KKP Alt+Enter becomes plain Enter",
			in:      []byte("hi\x1b[13;3u"),
			wantOut: []byte("hi\r"),
		},
		{
			name:    "mixed KKP and legacy in one chunk",
			in:      []byte("a\rb\x1b[13;3uc\x1b[13u"),
			wantOut: []byte("a\\\rb\rc\\\r"),
		},
		{
			name:    "KKP arrow key still passes through (\\x1b[A)",
			in:      []byte("a\x1b[Ab"),
			wantOut: []byte("a\x1b[Ab"),
		},
		{
			name:     "partial KKP Alt+Enter held back at chunk end",
			in:       []byte("hi\x1b[13;3"),
			wantOut:  []byte("hi"),
			wantHold: []byte("\x1b[13;3"),
		},
		{
			name:     "partial KKP plain Enter held back at chunk end",
			in:       []byte("hi\x1b[13"),
			wantOut:  []byte("hi"),
			wantHold: []byte("\x1b[13"),
		},
		{
			name:    "legacy Alt+Backspace becomes Ctrl+U",
			in:      []byte("hi\x1b\x7f"),
			wantOut: []byte("hi\x15"),
		},
		{
			name:    "KKP Alt+Backspace becomes Ctrl+U",
			in:      []byte("hi\x1b[127;3u"),
			wantOut: []byte("hi\x15"),
		},
		{
			name:    "plain Backspace (lone DEL) passes through",
			in:      []byte("hi\x7f"),
			wantOut: []byte("hi\x7f"),
		},
		{
			name:    "mixed: Alt+Backspace and Alt+Enter in one chunk",
			in:      []byte("a\x1b\x7fb\x1b\rc"),
			wantOut: []byte("a\x15b\rc"),
		},
		{
			name:     "partial KKP Alt+Backspace held back at chunk end",
			in:       []byte("hi\x1b[127;3"),
			wantOut:  []byte("hi"),
			wantHold: []byte("\x1b[127;3"),
		},
	}

	t.Run("codex keymap", func(t *testing.T) {
		px := &proxy{sendKM: sendKeymapByAgent["codex"]}
		cases := []struct{ in, want []byte }{
			{[]byte("hi\r"), []byte("hi\n")},             // Enter → newline
			{[]byte("hi\x1b\r"), []byte("hi\x1b\r")},     // legacy Alt+Enter → submit chord
			{[]byte("hi\x1b[13;3u"), []byte("hi\x1b\r")}, // KKP Alt+Enter → submit chord
			{[]byte("a\rb\x1b\r"), []byte("a\nb\x1b\r")},
			{[]byte("\x1b[200~text\rmore\x1b[201~"), []byte("\x1b[200~text\rmore\x1b[201~")}, // paste untouched
		}
		for _, c := range cases {
			got, _, _ := px.translateChunk(c.in, false)
			if !bytes.Equal(got, c.want) {
				t.Errorf("in=%q: got %q, want %q", c.in, got, c.want)
			}
		}
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotOut, gotHold, gotPaste := p.translateChunk(tc.in, tc.startPase)
			if !bytes.Equal(gotOut, tc.wantOut) {
				t.Errorf("out: got %q, want %q", gotOut, tc.wantOut)
			}
			if !bytes.Equal(gotHold, tc.wantHold) {
				t.Errorf("hold: got %q, want %q", gotHold, tc.wantHold)
			}
			if gotPaste != tc.wantPaste {
				t.Errorf("paste: got %v, want %v", gotPaste, tc.wantPaste)
			}
		})
	}
}

exec
/bin/zsh -lc "sed -n '1,120p' cmd/pair-wrap/keymap_registry_test.go" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
package main

import (
	"bytes"
	"testing"
)

// TestSendKeymapByAgent_RegistrationTable pins the per-agent stdin
// rewrite table. Adding the row for a new agent or accidentally
// editing an existing one (typo in the byte literal, swapped fields)
// is the kind of change that's easy to miss in review — claude /
// codex / agy each have their own ergonomics expectations and
// the wrong bytes silently breaks Enter / Shift+Enter for that
// agent in production. Treat the table as a contract.
func TestSendKeymapByAgent_RegistrationTable(t *testing.T) {
	type row struct {
		plain, alt, altBS []byte
	}
	ctrlU := []byte{0x15} // Alt+Backspace → kill to line start (all agents)
	want := map[string]row{
		// claude reads `\<Enter>` as newline regardless of terminal
		// keyboard-protocol level — the documented portable path.
		"claude": {[]byte{'\\', '\r'}, []byte{'\r'}, ctrlU},
		// codex: plain Enter inserts newline; Alt+Enter stays a modified submit chord.
		"codex": {[]byte{'\n'}, []byte{'\x1b', '\r'}, ctrlU},
		// agy: same as codex.
		"agy": {[]byte{'\n'}, []byte{'\r'}, ctrlU},
	}
	if len(sendKeymapByAgent) != len(want) {
		t.Fatalf("sendKeymapByAgent has %d agents, want %d (%v)",
			len(sendKeymapByAgent), len(want), agentNames())
	}
	for agent, w := range want {
		got, ok := sendKeymapByAgent[agent]
		if !ok {
			t.Errorf("missing agent %q in sendKeymapByAgent", agent)
			continue
		}
		if !bytes.Equal(got.plainCR, w.plain) {
			t.Errorf("%s.plainCR: got %q, want %q", agent, got.plainCR, w.plain)
		}
		if !bytes.Equal(got.altCR, w.alt) {
			t.Errorf("%s.altCR: got %q, want %q", agent, got.altCR, w.alt)
		}
		if !bytes.Equal(got.altBS, w.altBS) {
			t.Errorf("%s.altBS: got %q, want %q", agent, got.altBS, w.altBS)
		}
	}
}

// TestTranslateChunk_AgyKeymap exercises the agy row through
// translateChunk so a typo in the registration table that happens to
// pass the registry test (e.g. swapped fields) also gets caught at
// the translation layer.
func TestTranslateChunk_AgyKeymap(t *testing.T) {
	p := &proxy{sendKM: sendKeymapByAgent["agy"]}
	cases := []struct{ in, want []byte }{
		{[]byte("hi\r"), []byte("hi\n")},                                                 // Enter → newline
		{[]byte("hi\x1b\r"), []byte("hi\r")},                                             // Alt+Enter → send
		{[]byte("a\rb\x1b\r"), []byte("a\nb\r")},                                         // both, same chunk
		{[]byte("hi\x1b\x7f"), []byte("hi\x15")},                                         // Alt+Backspace → Ctrl+U
		{[]byte("\x1b[200~text\rmore\x1b[201~"), []byte("\x1b[200~text\rmore\x1b[201~")}, // paste untouched
	}
	for _, c := range cases {
		got, _, _ := p.translateChunk(c.in, false)
		if !bytes.Equal(got, c.want) {
			t.Errorf("in=%q: got %q, want %q", c.in, got, c.want)
		}
	}
}

func agentNames() []string {
	var out []string
	for k := range sendKeymapByAgent {
		out = append(out, k)
	}
	return out
}

exec
/bin/zsh -lc "sed -n '1,80p' atlas/how-to-bring-up-a-new-harness-cli.md" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "rg -n \"func \\(p \\*proxy\\) translateChunk|translateChunk\" cmd/pair-wrap/main.go cmd/pair-wrap/*.go" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
# How to Bring Up a New Harness CLI in pair

`pair` is an agent-agnostic, Neovim-backed launcher environment. While the horizontal two-pane design is generic, delivering a premium, seamless pair-programming experience requires integrating the agent across seven critical integration surfaces.

This guide outlines how to bring up a new agent harness CLI (e.g., `agy`) and achieve parity with existing agents (`claude`, `codex`).

---

## 1. Key Integration Aspects

### Aspect 1: Return Key Remapping
By default, the bottom Neovim draft pane maps **Enter** to insert a newline, and **Alt+Enter** to send the buffer. To provide visual and interactive consistency, the top agent pane (which runs inside the transparent PTY proxy `pair-wrap`) should map keys similarly:
- **Plain Enter** inside textareas/prompts should insert a newline (preventing accidental premature sends).
- **Alt+Enter** should submit the input.

**Implementation:**
- **File:** [cmd/pair-wrap/main.go](file:///Users/xianxu/workspace/pair/cmd/pair-wrap/main.go)
- Add the agent to `sendKeymapByAgent` defining `plainCR` and `altCR`:
  ```go
  var sendKeymapByAgent = map[string]sendKeymap{
      "agy": {
          plainCR: []byte{'\n'}, // plain Enter inserts newline
          altCR:   []byte{'\r'}, // Alt+Enter sends query
      },
      "codex": {
          plainCR: []byte{'\n'},       // plain Enter inserts newline
          altCR:   []byte{'\x1b','\r'}, // Alt+Enter submit chord
      },
  }
  ```
- **Note:** Claude uses `\<Enter>` (`[]byte{'\\', '\r'}`) as a newline. Codex uses LF (`\n`) for newline and preserves Alt+Enter as `ESC CR` for submit. Antigravity (`agy`) uses LF (`\n`) for newline and CR (`\r`) for send.

**Telemetry Signal** (aspect `1`, see §3): `return-remap` — `fired` each time a plain Enter is remapped to the agent's newline; `bypass` each time it passes through as a bare `\r` while an overlay is active. Emitted from `emitPlainCR`. The `fired:bypass` ratio is the health signal; an all-`bypass` or zero-`fired` session means the remap stopped engaging.

---

### Aspect 2: Overlay-Aware Return Suspension
If the agent presents blocking overlays, pickers (like file autocompletes), or yes/no confirmation modals, text-area Enter remapping will break the interaction. Inside an overlay, a plain **Enter** must send a bare carriage return (`\r`) to select/confirm.

`pair-wrap` suspends remapping by registering an overlay detector function which arms a temporary `pickerActive` flag. The next plain Enter is bypass-translated to a bare `\r`, and the flag is immediately cleared.

**Implementation:**
- **File:** [cmd/pair-wrap/main.go](file:///Users/xianxu/workspace/pair/cmd/pair-wrap/main.go)
- Register the detector in `overlayDetectorByAgent`:
  ```go
  var overlayDetectorByAgent = map[string]overlayDetector{
      "claude": detectClaudeOverlayOpen,
      "codex":  detectCodexOverlayOpen,
      "agy":    detectAgyOverlayOpen,
  }
  ```
- Implement the detector. Detectors can scan the rolling output stream for custom OSC escape sequences (e.g. Claude's permission OSC `OSC 777;notify;...`, or Codex's `OSC 9;Plan mode prompt:...`) or fallback to visible text substring matches (e.g., watching for `"Press enter to confirm"`).
- **For `agy`:** Antigravity *does* render its permission picker in the PTY ("Do you want to proceed?", "Yes, and always allow", …), so `detectAgyOverlayOpen` matches those visible-text markers (no OSC) to arm `pickerActive` — without it, the remapped Enter can't confirm the picker and a stray newline leaks into the prompt (#000042).

**Telemetry Signal** (aspect `2`, see §3): `overlay-detect` — `fired` when a registered marker arms `pickerActive` (the detail carries the matched marker); **`near-miss`** when the output looks like a confirm/permission prompt (`promptShape` heuristic in `checkOverlayOpen`) but *no* registered marker matched. A `near-miss` is the drift fingerprint: the harness renamed its picker wording, the detector went silent, and the next plain Enter will leak a newline (#000042). The `detail` field carries the unrecognized line verbatim — that's the new string to add to `codexPickerMarkers`/`agyPickerMarkers` (or the OSC body for claude).

---

### Aspect 3: Session ID Watcher & Recovery
`pair` features a robust restart-in-place (`Alt+n`) and session reattach (`pair resume <tag>`) mechanism. To make this work, the launcher needs to discover the agent's unique conversation/session ID as soon as it is spawned.

**Discovery & Watcher:**
- **File:** [bin/pair-session-watch.sh](file:///Users/xianxu/workspace/pair/bin/pair-session-watch.sh)
- Since TUI agents do not always expose session IDs on stdout, `pair-session-watch.sh` runs in the background. It finds the agent process PID from `$PAIR_DATA_DIR/agent-pid-<tag>` (written by `pair-wrap`), walks its descendants, and inspects files held open by the processes via `lsof -p <pid>`.
- Configure the agent's session file criteria:
  ```bash
  agy)
      watch_dir="$HOME/.gemini/antigravity-cli/conversations"
      find_args=(-type f -name '*.db')
      ;;
  ```
- Extract the ID from the file path or contents in `extract_id()`:
  ```bash
  agy)
      # Extract UUID from SQLite DB name: ~/.gemini/antigravity-cli/conversations/<uuid>.db
      local fn
      fn=$(basename "$1" .db)
      if [[ "$fn" =~ ^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$ ]]; then
          echo "$fn"
      fi

 succeeded in 0ms:
cmd/pair-wrap/translate_test.go:168:			got, _, _ := px.translateChunk(c.in, false)
cmd/pair-wrap/translate_test.go:177:			gotOut, gotHold, gotPaste := p.translateChunk(tc.in, tc.startPase)
cmd/pair-wrap/overlay_test.go:85:	got, leftover, inPaste := p.translateChunk([]byte("\r\r"), false)
cmd/pair-wrap/keymap_registry_test.go:52:// translateChunk so a typo in the registration table that happens to
cmd/pair-wrap/keymap_registry_test.go:65:		got, _, _ := p.translateChunk(c.in, false)
cmd/pair-wrap/main.go:215:	// translateChunk emits a bare \r for the user's plain Enter
cmd/pair-wrap/main.go:220:	// translateChunk → atomic.
cmd/pair-wrap/main.go:1199://     translateChunk over the combined slice, writes the output to
cmd/pair-wrap/main.go:1205:// State machine (per translateChunk):
cmd/pair-wrap/main.go:1317:			outBytes, leftover, newInPaste := p.translateChunk(data, inPaste)
cmd/pair-wrap/main.go:1465:// translateChunk walks `data` and returns (rewritten bytes, leftover to
cmd/pair-wrap/main.go:1469:func (p *proxy) translateChunk(data []byte, inPaste bool) ([]byte, []byte, bool) {
cmd/pair-wrap/main.go:215:	// translateChunk emits a bare \r for the user's plain Enter
cmd/pair-wrap/main.go:220:	// translateChunk → atomic.
cmd/pair-wrap/main.go:1199://     translateChunk over the combined slice, writes the output to
cmd/pair-wrap/main.go:1205:// State machine (per translateChunk):
cmd/pair-wrap/main.go:1317:			outBytes, leftover, newInPaste := p.translateChunk(data, inPaste)
cmd/pair-wrap/main.go:1465:// translateChunk walks `data` and returns (rewritten bytes, leftover to
cmd/pair-wrap/main.go:1469:func (p *proxy) translateChunk(data []byte, inPaste bool) ([]byte, []byte, bool) {
cmd/pair-wrap/translate_stdin_test.go:89:// translateChunk). Plain \r should become \\r per the claude keymap.
cmd/pair-wrap/translate_stdin_test.go:124:	// Send a lone ESC. translateChunk holds it back (could be the

exec
/bin/zsh -lc "sed -n '165,190p' atlas/architecture.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
Alt+x leaves the draft, queue, and history intact — the next session resumes them. Use Shift+Alt+Backspace (`forget_all`) for the destructive "start anew" path.

### Outer-TTY capture and notification routing — `bin/pair-wrap`, `bin/pair-notify`

**Why.** Zellij parses every escape on the way out for its virtual-screen reconstruction and drops sequences it doesn't recognize. OSC 9 and OSC 777 (the notification escapes outer wrappers like cmux watch for) fall in that bucket and never reach the host terminal. BEL is forwarded since zellij 0.44, but cmux specifically watches OSC, not BEL — so BEL forwarding doesn't help that integration. Filed as #000011.

**Mechanism, in two layers:**

1. **Outer-TTY capture (in `bin/pair`).** Before invoking zellij, on every attach (both create and reattach branches), pair calls `tty(1)`. The result is the path of pair's controlling TTY — which is precisely the outer PTY (the one allocated by whatever wraps pair: cmux, a terminal emulator, etc.). That path gets written to `$DATA_DIR/outer-tty-<tag>`. Refreshed on every attach because the outer PTY changes across detach/reattach, while pane-shell env stays frozen at zellij session-creation time (env-var approaches would go stale).

2. **Two consumers** of the captured path:

   - **`bin/pair-wrap`** (Go, `cmd/pair-wrap`). Transparent PTY proxy. The zellij agent pane runs `pair-wrap $PAIR_AGENT $PAIR_AGENT_ARGS` instead of the agent directly (wired in `zellij/layouts/main.kdl`). The wrapper allocates a fresh PTY for the agent, forwards stdin/stdout transparently with SIGWINCH propagation, and watches the agent's output stream for OSC notifications. On detection it writes OSC 9 directly to the recorded outer-TTY path — bypassing zellij.

     **Stdin raw mode.** The wrapper switches its stdin (zellij's pane PTY) into termios raw mode for the duration. Without this the kernel's line discipline does local echo + canonical buffering on the bytes flowing toward the wrapped TUI, which double-echoes keystrokes and corrupts terminal-response sequences. Saved/restored in a `finally` block.

     **Stdin Enter remap (per-agent).** `sendKeymapByAgent` (`cmd/pair-wrap/main.go`) translates the user's Enter / Alt+Enter to per-agent send/newline bytes so the convention matches pair's nvim draft pane (Enter = newline, Alt+Enter = send). For `claude` the user's plain Enter becomes `\<CR>` (claude's portable "insert newline" sequence); Alt+Enter becomes a bare `\r` (send). For Codex, pair sends LF for plain Enter (newline) and preserves Alt+Enter as `ESC CR` (submit chord). For agy, pair sends LF for plain Enter and CR for Alt+Enter (send). The same keymap carries `altBS`: Alt+Backspace (legacy `\x1b\x7f` or KKP `\x1b[127;3u`, the same two-protocol shape as Alt+Enter) rewrites to **Ctrl+U** (`0x15`, kill-to-line-start) for every agent — so Alt+Delete in the agent pane matches the agent's Cmd+Delete and the draft pane's Alt+Delete. A lone `0x7f` (plain Backspace) isn't ESC-prefixed, so it passes through untouched. Opt out of the whole remap with `PAIR_WRAP_REMAP_RETURN=0`.

     **Stdout filtering and batching (Codex).** Codex inline mode emits DEC synchronized-output markers (`ESC[?2026h` / `ESC[?2026l`) around frequent redraw batches. It can also enable terminal focus-event mode (`ESC[?1004h`) even though pair/zellij do not use focus events for the agent pane. `pair-wrap` strips those markers from the stdout stream sent to zellij, because zellij scrollback/mouse scrolling can behave poorly while a pane is in synchronized-output or extra terminal-event modes during generation. The filtered, user-visible stdout stream is then queued and flushed to zellij on a 100ms cadence (plus EOF) to lower redraw pressure from dense Codex repaint bursts (#85). The raw scrollback log remains immediate and unfiltered so forensic replay still captures the agent's original PTY stream and offset-keyed resize/time events stay aligned.

     **Overlay-aware suspension (per-agent).** Textarea Enter remaps are wrong while a blocking overlay / picker has focus: the overlay needs a bare `\r` to confirm the highlighted option. pair-wrap registers per-agent overlay detectors in `overlayDetectorByAgent`, sets `pickerActive` when one fires, and emits a bare `\r` for the next plain Enter only. The flag clears after that one Enter, so normal textarea remapping resumes for the following keystroke. Claude uses the stable `OSC 777;notify;Claude Code;Claude needs your permission` body. Codex question prompts use `OSC 9;Plan mode prompt:...`; other Codex pickers fall back to stripped visible output plus a short text carryover watching for labels such as `Use session directory (` / `Use current directory (`, `Press enter to continue`, and `Press enter to confirm or esc to go back`. Codex image attach uses a stronger local signal: Alt+i arms pair-wrap capture immediately before Ctrl+V, and that capture arm also enables the next-Enter overlay bypass. The carryover is cleared when the confirming Enter is consumed so stale picker text cannot re-arm the flag. Known edge inherited from the one-shot design: dismissing an overlay without Enter leaves the flag set until the next plain Enter.

     **OSC filter (`is_actionable_osc`).** Parsing every OSC `<Ps>;<body>` and discriminating is essential — naive "any BEL → emit" over-fires constantly because claude (and similar agents) update OSC 0 (window title) every second with a spinner, and every title set's BEL terminator looks like a "lone bell." The filter:
     - **Skip** OSC 0/1/2 (title sets), OSC 9;4;... (iTerm progress codes — fire on every tool-call cycle).
     - **Forward** OSC 777;... (urxvt-style `Notify`) and OSC 9;`<text>` (iTerm-style notification with content).
     - Bare BEL (no OSC framing in the rolling buffer) → **logged but not forwarded by default**; set `PAIR_WRAP_BELL_FALLBACK=1` to re-enable forwarding (issue #000014).

exec
/bin/zsh -lc 'git diff cc2cb6da1dea4c6c5422267bb74bcff34ba92343..HEAD -- workshop/issues/000087-fix-codex-alt-enter-remap.md' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
diff --git a/workshop/issues/000087-fix-codex-alt-enter-remap.md b/workshop/issues/000087-fix-codex-alt-enter-remap.md
index 5c850a9..3cff7f9 100644
--- a/workshop/issues/000087-fix-codex-alt-enter-remap.md
+++ b/workshop/issues/000087-fix-codex-alt-enter-remap.md
@@ -5,7 +5,7 @@ deps: []
 github_issue:
 created: 2026-06-29
 updated: 2026-06-29
-estimate_hours:
+estimate_hours: 0.46
 started: 2026-06-29T18:25:25-07:00
 ---
 
@@ -13,16 +13,53 @@ started: 2026-06-29T18:25:25-07:00
 
 ## Problem
 
+After #86, restarted sessions do invoke `zellij action send-keys "Alt Enter"` from the draft pane, but Codex still leaves the inserted draft text sitting in the composer. Live trace for `PAIR_TAG=2` at 2026-06-29T18:23:10-07:00 shows:
+
+- nvim wrote the body via `draft.send.write-body` (`body_len: 41`).
+- nvim invoked `draft.send.submit` as `zellij action send-keys Alt Enter`.
+- pair-wrap read the body bytes, then read Alt+Enter as `ESC CR` (`raw_len: 2`, SHA `a6d286d70768`), translated it to bare `CR` (`translated_len: 1`, SHA `9d1e0e2d9459`), and wrote that to the Codex PTY.
+
+So the Zellij action and pair-wrap Alt+Enter recognition work; the stale assumption is the Codex keymap row that collapses Alt+Enter to bare `CR`. Current Codex needs the modified Alt+Enter chord forwarded to the child PTY.
+
 ## Spec
 
+For Codex only, pair-wrap should preserve the submit chord when it sees user Alt+Enter: legacy `ESC CR` and KKP `ESC [ 13 ; 3 u` should translate to `ESC CR` for the Codex child PTY, not bare `CR`. Plain Enter should continue translating to LF so the pair convention remains Enter = newline. Claude and agy keymaps should remain unchanged.
+
+Docs that describe Codex's keymap should say Codex plain Enter maps to newline and Alt+Enter is forwarded as the submit chord.
+
 ## Done when
 
--
+- [x] Codex Alt+Enter translation emits `ESC CR`.
+- [x] Codex plain Enter still emits LF.
+- [x] Claude and agy keymaps are unchanged.
+- [x] Tests cover both legacy and KKP Alt+Enter inputs for Codex.
+- [x] Atlas/docs no longer claim Codex Alt+Enter maps to bare `CR`.
 
 ## Plan
 
-- [ ]
+- [x] Update the pure keymap tests first to expect Codex `altCR == ESC CR` and to expect both legacy and KKP Codex Alt+Enter inputs to translate to `ESC CR`.
+- [x] Change only `sendKeymapByAgent["codex"].altCR` in `cmd/pair-wrap/main.go`.
+- [x] Update Codex keymap docs in `atlas/architecture.md`, `atlas/how-to-bring-up-a-new-harness-cli.md`, and comments/tests that state the old bare-CR contract.
+- [x] Verify `go test ./cmd/pair-wrap`, focused nvim submit tests, issue validation, and whitespace.
+
+## Estimate
+
+```estimate
+model: estimate-logic-v3.1
+familiarity: 1.0
+item: smaller-go-module design=0.10 impl=0.12
+item: atlas-docs design=0.00 impl=0.08
+item: milestone-review design=0.00 impl=0.15
+design-buffer: 0.10
+total: 0.46
+```
+
+Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.
 
 ## Log
 
 ### 2026-06-29
+
+User restarted after #86 and still saw Alt+Return insert text into Codex without submitting. Investigated live traces instead of guessing. `zellij-actions-2.jsonl` confirmed the restarted draft pane is using `send-keys Alt Enter`; `wrap-events-2.jsonl` confirmed pair-wrap receives that as Alt+Enter and currently translates it to bare `CR`. Because bare `CR` reached Codex and did not submit, the stale contract is the Codex keymap's `altCR` output, not nvim focus, Zellij delivery, or pair-wrap input recognition (`ARCH-PURPOSE`).
+
+Implemented the Codex keymap correction with TDD. RED: `go test ./cmd/pair-wrap` failed on `codex.altCR: got "\r", want "\x1b\r"` and on both legacy/KKP Codex Alt+Enter translation cases. GREEN: changed only `sendKeymapByAgent["codex"].altCR` to `ESC CR`; `go test ./cmd/pair-wrap` passed. Synced atlas/docs to distinguish Codex from agy and verified with `go test ./cmd/pair-wrap`, `bash tests/queue-send-test.sh`, `bash tests/review-poke-test.sh`, `git diff --check`, and `sdlc issue validate workshop/issues/000087-fix-codex-alt-enter-remap.md`.

exec
/bin/zsh -lc 'git diff -- workshop/issues/000087-fix-codex-alt-enter-remap.md' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
diff --git a/workshop/issues/000087-fix-codex-alt-enter-remap.md b/workshop/issues/000087-fix-codex-alt-enter-remap.md
index 3cff7f9..96ba2ca 100644
--- a/workshop/issues/000087-fix-codex-alt-enter-remap.md
+++ b/workshop/issues/000087-fix-codex-alt-enter-remap.md
@@ -1,12 +1,13 @@
 ---
 id: 000087
-status: working
+status: done
 deps: []
 github_issue:
 created: 2026-06-29
 updated: 2026-06-29
 estimate_hours: 0.46
 started: 2026-06-29T18:25:25-07:00
+actual_hours: 0.13
 ---
 
 # Fix Codex Alt Enter remap
@@ -59,6 +60,7 @@ Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `
 ## Log
 
 ### 2026-06-29
+- 2026-06-29: closed — go test ./cmd/pair-wrap; bash tests/queue-send-test.sh; bash tests/review-poke-test.sh; git diff --check; sdlc issue validate workshop/issues/000087-fix-codex-alt-enter-remap.md
 
 User restarted after #86 and still saw Alt+Return insert text into Codex without submitting. Investigated live traces instead of guessing. `zellij-actions-2.jsonl` confirmed the restarted draft pane is using `send-keys Alt Enter`; `wrap-events-2.jsonl` confirmed pair-wrap receives that as Alt+Enter and currently translates it to bare `CR`. Because bare `CR` reached Codex and did not submit, the stale contract is the Codex keymap's `altCR` output, not nvim focus, Zellij delivery, or pair-wrap input recognition (`ARCH-PURPOSE`).
 

codex
The main code path looks narrowly scoped: the Codex row now emits `ESC CR`, and the existing translator treats both legacy and KKP Alt+Enter as data-driven `altCR`. I’m checking for stale consumers and metadata drift now, because that is where this kind of boundary can still fail.
exec
/bin/zsh -lc "nl -ba cmd/pair-wrap/translate_test.go | sed -n '145,175p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
   145			{
   146				name:    "mixed: Alt+Backspace and Alt+Enter in one chunk",
   147				in:      []byte("a\x1b\x7fb\x1b\rc"),
   148				wantOut: []byte("a\x15b\rc"),
   149			},
   150			{
   151				name:     "partial KKP Alt+Backspace held back at chunk end",
   152				in:       []byte("hi\x1b[127;3"),
   153				wantOut:  []byte("hi"),
   154				wantHold: []byte("\x1b[127;3"),
   155			},
   156		}
   157	
   158		t.Run("codex keymap", func(t *testing.T) {
   159			px := &proxy{sendKM: sendKeymapByAgent["codex"]}
   160			cases := []struct{ in, want []byte }{
   161				{[]byte("hi\r"), []byte("hi\n")},             // Enter → newline
   162				{[]byte("hi\x1b\r"), []byte("hi\x1b\r")},     // legacy Alt+Enter → submit chord
   163				{[]byte("hi\x1b[13;3u"), []byte("hi\x1b\r")}, // KKP Alt+Enter → submit chord
   164				{[]byte("a\rb\x1b\r"), []byte("a\nb\x1b\r")},
   165				{[]byte("\x1b[200~text\rmore\x1b[201~"), []byte("\x1b[200~text\rmore\x1b[201~")}, // paste untouched
   166			}
   167			for _, c := range cases {
   168				got, _, _ := px.translateChunk(c.in, false)
   169				if !bytes.Equal(got, c.want) {
   170					t.Errorf("in=%q: got %q, want %q", c.in, got, c.want)
   171				}
   172			}
   173		})
   174	
   175		for _, tc := range tests {

exec
/bin/zsh -lc "nl -ba cmd/pair-wrap/main.go | sed -n '108,165p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
   108	//
   109	//	Enter      = insert newline
   110	//	Alt+Enter  = send to agent
   111	//
   112	// but the agent's native TUI typically uses Enter = send. That mismatch
   113	// is jarring when the user moves between panes. When PAIR_WRAP_REMAP_RETURN
   114	// isn't "0", pair-wrap rewrites stdin so the agent receives the inverted
   115	// mapping: incoming Enter becomes the agent's "insert newline" sequence,
   116	// incoming Alt+Enter becomes the agent's submit sequence.
   117	//
   118	//   - plainCR:   bytes emitted when the user hits Enter alone (\r)
   119	//   - altCR:     bytes emitted when the user hits Alt+Enter (\x1b\r)
   120	//   - altBS:     bytes emitted when the user hits Alt+Backspace (\x1b\x7f)
   121	//
   122	// Claude reads "\<Enter>" (backslash + CR) as a newline regardless of
   123	// terminal keyboard-protocol support — this is the documented portable
   124	// path. Other agents need their own probing; leave them out of the table
   125	// to fall through to no-rewrite (today's pass-through behavior).
   126	//
   127	// altBS maps Alt+Backspace to Ctrl+U (kill-to-line-start) so Alt+Delete
   128	// in the agent pane matches the agent's existing Cmd+Delete and the
   129	// draft pane's Alt+Delete. Ctrl+U is the universal readline kill, so it's
   130	// the same byte for every agent.
   131	type sendKeymap struct {
   132		plainCR, altCR []byte
   133		altBS          []byte
   134	}
   135	
   136	var sendKeymapByAgent = map[string]sendKeymap{
   137		"claude": {
   138			// Claude reads `\<Enter>` as newline regardless of terminal
   139			// keyboard-protocol level — the documented portable path.
   140			plainCR: []byte{'\\', '\r'},
   141			altCR:   []byte{'\r'},
   142			altBS:   []byte{0x15}, // Ctrl+U — kill to line start
   143		},
   144		"codex": {
   145			// Codex plain Enter inserts a newline when rewritten to LF.
   146			// Current Codex submit requires the modified Alt+Enter chord
   147			// itself, so preserve that as legacy ESC+CR.
   148			plainCR: []byte{'\n'},
   149			altCR:   []byte{'\x1b', '\r'},
   150			altBS:   []byte{0x15}, // Ctrl+U — kill to line start
   151		},
   152		"agy": {
   153			// Antigravity (agy) follows the same Enter/Shift+Enter convention
   154			// as codex.
   155			plainCR: []byte{'\n'},
   156			altCR:   []byte{'\r'},
   157			altBS:   []byte{0x15}, // Ctrl+U — kill to line start
   158		},
   159	}
   160	
   161	type overlayDetector func(*proxy, []byte, []byte) (bool, string)
   162	
   163	var overlayDetectorByAgent = map[string]overlayDetector{
   164		"claude": detectClaudeOverlayOpen,
   165		"codex":  detectCodexOverlayOpen,

exec
/bin/zsh -lc "nl -ba workshop/issues/000087-fix-codex-alt-enter-remap.md | sed -n '1,75p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	---
     2	id: 000087
     3	status: done
     4	deps: []
     5	github_issue:
     6	created: 2026-06-29
     7	updated: 2026-06-29
     8	estimate_hours: 0.46
     9	started: 2026-06-29T18:25:25-07:00
    10	actual_hours: 0.13
    11	---
    12	
    13	# Fix Codex Alt Enter remap
    14	
    15	## Problem
    16	
    17	After #86, restarted sessions do invoke `zellij action send-keys "Alt Enter"` from the draft pane, but Codex still leaves the inserted draft text sitting in the composer. Live trace for `PAIR_TAG=2` at 2026-06-29T18:23:10-07:00 shows:
    18	
    19	- nvim wrote the body via `draft.send.write-body` (`body_len: 41`).
    20	- nvim invoked `draft.send.submit` as `zellij action send-keys Alt Enter`.
    21	- pair-wrap read the body bytes, then read Alt+Enter as `ESC CR` (`raw_len: 2`, SHA `a6d286d70768`), translated it to bare `CR` (`translated_len: 1`, SHA `9d1e0e2d9459`), and wrote that to the Codex PTY.
    22	
    23	So the Zellij action and pair-wrap Alt+Enter recognition work; the stale assumption is the Codex keymap row that collapses Alt+Enter to bare `CR`. Current Codex needs the modified Alt+Enter chord forwarded to the child PTY.
    24	
    25	## Spec
    26	
    27	For Codex only, pair-wrap should preserve the submit chord when it sees user Alt+Enter: legacy `ESC CR` and KKP `ESC [ 13 ; 3 u` should translate to `ESC CR` for the Codex child PTY, not bare `CR`. Plain Enter should continue translating to LF so the pair convention remains Enter = newline. Claude and agy keymaps should remain unchanged.
    28	
    29	Docs that describe Codex's keymap should say Codex plain Enter maps to newline and Alt+Enter is forwarded as the submit chord.
    30	
    31	## Done when
    32	
    33	- [x] Codex Alt+Enter translation emits `ESC CR`.
    34	- [x] Codex plain Enter still emits LF.
    35	- [x] Claude and agy keymaps are unchanged.
    36	- [x] Tests cover both legacy and KKP Alt+Enter inputs for Codex.
    37	- [x] Atlas/docs no longer claim Codex Alt+Enter maps to bare `CR`.
    38	
    39	## Plan
    40	
    41	- [x] Update the pure keymap tests first to expect Codex `altCR == ESC CR` and to expect both legacy and KKP Codex Alt+Enter inputs to translate to `ESC CR`.
    42	- [x] Change only `sendKeymapByAgent["codex"].altCR` in `cmd/pair-wrap/main.go`.
    43	- [x] Update Codex keymap docs in `atlas/architecture.md`, `atlas/how-to-bring-up-a-new-harness-cli.md`, and comments/tests that state the old bare-CR contract.
    44	- [x] Verify `go test ./cmd/pair-wrap`, focused nvim submit tests, issue validation, and whitespace.
    45	
    46	## Estimate
    47	
    48	```estimate
    49	model: estimate-logic-v3.1
    50	familiarity: 1.0
    51	item: smaller-go-module design=0.10 impl=0.12
    52	item: atlas-docs design=0.00 impl=0.08
    53	item: milestone-review design=0.00 impl=0.15
    54	design-buffer: 0.10
    55	total: 0.46
    56	```
    57	
    58	Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.
    59	
    60	## Log
    61	
    62	### 2026-06-29
    63	- 2026-06-29: closed — go test ./cmd/pair-wrap; bash tests/queue-send-test.sh; bash tests/review-poke-test.sh; git diff --check; sdlc issue validate workshop/issues/000087-fix-codex-alt-enter-remap.md
    64	
    65	User restarted after #86 and still saw Alt+Return insert text into Codex without submitting. Investigated live traces instead of guessing. `zellij-actions-2.jsonl` confirmed the restarted draft pane is using `send-keys Alt Enter`; `wrap-events-2.jsonl` confirmed pair-wrap receives that as Alt+Enter and currently translates it to bare `CR`. Because bare `CR` reached Codex and did not submit, the stale contract is the Codex keymap's `altCR` output, not nvim focus, Zellij delivery, or pair-wrap input recognition (`ARCH-PURPOSE`).
    66	
    67	Implemented the Codex keymap correction with TDD. RED: `go test ./cmd/pair-wrap` failed on `codex.altCR: got "\r", want "\x1b\r"` and on both legacy/KKP Codex Alt+Enter translation cases. GREEN: changed only `sendKeymapByAgent["codex"].altCR` to `ESC CR`; `go test ./cmd/pair-wrap` passed. Synced atlas/docs to distinguish Codex from agy and verified with `go test ./cmd/pair-wrap`, `bash tests/queue-send-test.sh`, `bash tests/review-poke-test.sh`, `git diff --check`, and `sdlc issue validate workshop/issues/000087-fix-codex-alt-enter-remap.md`.

exec
/bin/zsh -lc "nl -ba cmd/pair-wrap/keymap_registry_test.go | sed -n '15,35p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
    15	func TestSendKeymapByAgent_RegistrationTable(t *testing.T) {
    16		type row struct {
    17			plain, alt, altBS []byte
    18		}
    19		ctrlU := []byte{0x15} // Alt+Backspace → kill to line start (all agents)
    20		want := map[string]row{
    21			// claude reads `\<Enter>` as newline regardless of terminal
    22			// keyboard-protocol level — the documented portable path.
    23			"claude": {[]byte{'\\', '\r'}, []byte{'\r'}, ctrlU},
    24			// codex: plain Enter inserts newline; Alt+Enter stays a modified submit chord.
    25			"codex": {[]byte{'\n'}, []byte{'\x1b', '\r'}, ctrlU},
    26			// agy: same as codex.
    27			"agy": {[]byte{'\n'}, []byte{'\r'}, ctrlU},
    28		}
    29		if len(sendKeymapByAgent) != len(want) {
    30			t.Fatalf("sendKeymapByAgent has %d agents, want %d (%v)",
    31				len(sendKeymapByAgent), len(want), agentNames())
    32		}
    33		for agent, w := range want {
    34			got, ok := sendKeymapByAgent[agent]
    35			if !ok {

exec
/bin/zsh -lc "rg -n \"Codex.*bare|codex.*bare|Codex.*CR|codex.*CR|same as codex|same as Codex|Alt\\+Enter.*bare|bare.*Alt\\+Enter\" cmd/pair-wrap atlas workshop -S" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
atlas/how-to-bring-up-a-new-harness-cli.md:31:- **Note:** Claude uses `\<Enter>` (`[]byte{'\\', '\r'}`) as a newline. Codex uses LF (`\n`) for newline and preserves Alt+Enter as `ESC CR` for submit. Antigravity (`agy`) uses LF (`\n`) for newline and CR (`\r`) for send.
workshop/issues/000087-fix-codex-alt-enter-remap.md:21:- pair-wrap read the body bytes, then read Alt+Enter as `ESC CR` (`raw_len: 2`, SHA `a6d286d70768`), translated it to bare `CR` (`translated_len: 1`, SHA `9d1e0e2d9459`), and wrote that to the Codex PTY.
workshop/issues/000087-fix-codex-alt-enter-remap.md:23:So the Zellij action and pair-wrap Alt+Enter recognition work; the stale assumption is the Codex keymap row that collapses Alt+Enter to bare `CR`. Current Codex needs the modified Alt+Enter chord forwarded to the child PTY.
workshop/issues/000087-fix-codex-alt-enter-remap.md:27:For Codex only, pair-wrap should preserve the submit chord when it sees user Alt+Enter: legacy `ESC CR` and KKP `ESC [ 13 ; 3 u` should translate to `ESC CR` for the Codex child PTY, not bare `CR`. Plain Enter should continue translating to LF so the pair convention remains Enter = newline. Claude and agy keymaps should remain unchanged.
workshop/issues/000087-fix-codex-alt-enter-remap.md:33:- [x] Codex Alt+Enter translation emits `ESC CR`.
workshop/issues/000087-fix-codex-alt-enter-remap.md:37:- [x] Atlas/docs no longer claim Codex Alt+Enter maps to bare `CR`.
workshop/issues/000087-fix-codex-alt-enter-remap.md:41:- [x] Update the pure keymap tests first to expect Codex `altCR == ESC CR` and to expect both legacy and KKP Codex Alt+Enter inputs to translate to `ESC CR`.
workshop/issues/000087-fix-codex-alt-enter-remap.md:42:- [x] Change only `sendKeymapByAgent["codex"].altCR` in `cmd/pair-wrap/main.go`.
workshop/issues/000087-fix-codex-alt-enter-remap.md:43:- [x] Update Codex keymap docs in `atlas/architecture.md`, `atlas/how-to-bring-up-a-new-harness-cli.md`, and comments/tests that state the old bare-CR contract.
workshop/issues/000087-fix-codex-alt-enter-remap.md:65:User restarted after #86 and still saw Alt+Return insert text into Codex without submitting. Investigated live traces instead of guessing. `zellij-actions-2.jsonl` confirmed the restarted draft pane is using `send-keys Alt Enter`; `wrap-events-2.jsonl` confirmed pair-wrap receives that as Alt+Enter and currently translates it to bare `CR`. Because bare `CR` reached Codex and did not submit, the stale contract is the Codex keymap's `altCR` output, not nvim focus, Zellij delivery, or pair-wrap input recognition (`ARCH-PURPOSE`).
workshop/issues/000087-fix-codex-alt-enter-remap.md:67:Implemented the Codex keymap correction with TDD. RED: `go test ./cmd/pair-wrap` failed on `codex.altCR: got "\r", want "\x1b\r"` and on both legacy/KKP Codex Alt+Enter translation cases. GREEN: changed only `sendKeymapByAgent["codex"].altCR` to `ESC CR`; `go test ./cmd/pair-wrap` passed. Synced atlas/docs to distinguish Codex from agy and verified with `go test ./cmd/pair-wrap`, `bash tests/queue-send-test.sh`, `bash tests/review-poke-test.sh`, `git diff --check`, and `sdlc issue validate workshop/issues/000087-fix-codex-alt-enter-remap.md`.
cmd/pair-wrap/main.go:1043:	if p.agentBasename == "codex" && p.sendKM.plainCR != nil {
cmd/pair-wrap/keymap_registry_test.go:26:		// agy: same as codex.
atlas/architecture.md:181:     **Stdin Enter remap (per-agent).** `sendKeymapByAgent` (`cmd/pair-wrap/main.go`) translates the user's Enter / Alt+Enter to per-agent send/newline bytes so the convention matches pair's nvim draft pane (Enter = newline, Alt+Enter = send). For `claude` the user's plain Enter becomes `\<CR>` (claude's portable "insert newline" sequence); Alt+Enter becomes a bare `\r` (send). For Codex, pair sends LF for plain Enter (newline) and preserves Alt+Enter as `ESC CR` (submit chord). For agy, pair sends LF for plain Enter and CR for Alt+Enter (send). The same keymap carries `altBS`: Alt+Backspace (legacy `\x1b\x7f` or KKP `\x1b[127;3u`, the same two-protocol shape as Alt+Enter) rewrites to **Ctrl+U** (`0x15`, kill-to-line-start) for every agent — so Alt+Delete in the agent pane matches the agent's Cmd+Delete and the draft pane's Alt+Delete. A lone `0x7f` (plain Backspace) isn't ESC-prefixed, so it passes through untouched. Opt out of the whole remap with `PAIR_WRAP_REMAP_RETURN=0`.
atlas/architecture.md:280:- `<S-M-CR>` (Alt+Shift+Return, normal+insert) — `send_and_clear(no_submit=true)`: identical flow (strip, log, queue handling, clear, reset) but writes a bare CR (`write 13`) instead of the semantic Alt+Enter submit event. pair-wrap rewrites a bare CR into the agent's insert-newline sequence rather than its submit byte, so the draft lands in the agent's composer on a fresh line, **unsubmitted** — append-without-send.
workshop/history/000026-mouse-scroll-in-codex.md:68:  Codex unless `PAIR_CODEX_ALT_SCREEN=1` is set.
workshop/plans/000087-fix-codex-alt-enter-remap-plan.md:5:**Goal:** Make draft Alt+Return submit in current Codex sessions by forwarding Codex's Alt+Enter submit chord through pair-wrap instead of collapsing it to bare Enter.
workshop/plans/000087-fix-codex-alt-enter-remap-plan.md:77:Expected: FAIL on Codex `altCR` and Codex Alt+Enter translation, because production still emits bare `CR`.
workshop/plans/000087-fix-codex-alt-enter-remap-plan.md:112:Replace statements that say Codex Alt+Enter maps to bare `CR` with the current contract: plain Enter maps to LF newline; Alt+Enter is forwarded as the modified submit chord (`ESC CR`).
workshop/plans/000087-fix-codex-alt-enter-remap-plan.md:119:rg -n "codex.*Alt\\+Enter|Codex.*Alt\\+Enter|Alt\\+Enter.*codex|bare `?\\\\r|bare CR|plain Enter" cmd/pair-wrap atlas -S

codex
One discrepancy turned up in comments: agy is still described as “same as codex” in two places even though Codex and agy now intentionally differ on Alt+Enter. I’m checking whether that is just stale prose or evidence of a deeper consumer mismatch.
exec
/bin/zsh -lc "sed -n '1460,1545p' cmd/pair-wrap/main.go" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
	}
	p.adapt.Log(1, "return-remap", adapt.Fired, "plain Enter → newline remap")
	return append(out, p.sendKM.plainCR...)
}

// translateChunk walks `data` and returns (rewritten bytes, leftover to
// carry over, new bracketed-paste state). `leftover` is non-nil only
// when the chunk ends mid-escape that could still resolve into bpStart,
// bpEnd, or an Alt+Enter — the caller prepends it to the next read.
func (p *proxy) translateChunk(data []byte, inPaste bool) ([]byte, []byte, bool) {
	out := make([]byte, 0, len(data))
	i := 0
	for i < len(data) {
		if inPaste {
			// Scan for end-of-paste marker. Anything before it is
			// literal pasted content — forward verbatim.
			if idx := indexOfSubseq(data[i:], bpEnd); idx >= 0 {
				out = append(out, data[i:i+idx+len(bpEnd)]...)
				i += idx + len(bpEnd)
				inPaste = false
				continue
			}
			// Marker not in this chunk. Forward everything but hold back
			// a trailing partial ESC[201~ in case it splits the boundary.
			tail := trailingPartial(data[i:], bpEnd)
			out = append(out, data[i:len(data)-tail]...)
			leftover := append([]byte(nil), data[len(data)-tail:]...)
			return out, leftover, true
		}

		b := data[i]
		// Outside paste: scan for the multi-byte markers and the
		// single-byte plain Enter. Longer KKP forms come first so a
		// 7-byte \x1b[13;3u doesn't get partially matched as the
		// 5-byte \x1b[13u.
		if b == 0x1b {
			if startsWith(data[i:], bpStart) {
				out = append(out, bpStart...)
				i += len(bpStart)
				inPaste = true
				continue
			}
			// KKP Alt+Enter: \x1b[13;3u → send.
			if startsWith(data[i:], enterKKPAlt) {
				out = append(out, p.sendKM.altCR...)
				i += len(enterKKPAlt)
				continue
			}
			// KKP plain Enter, explicit-no-modifier form: \x1b[13;1u.
			if startsWith(data[i:], enterKKPPlainExp) {
				out = p.emitPlainCR(out)
				i += len(enterKKPPlainExp)
				continue
			}
			// KKP plain Enter: \x1b[13u.
			if startsWith(data[i:], enterKKPPlain) {
				out = p.emitPlainCR(out)
				i += len(enterKKPPlain)
				continue
			}
			// Legacy Alt+Enter: \x1b\r.
			if startsWith(data[i:], enterLegacyAlt) {
				out = append(out, p.sendKM.altCR...)
				i += len(enterLegacyAlt)
				continue
			}
			// KKP Alt+Backspace: \x1b[127;3u → kill to line start (Ctrl+U).
			if startsWith(data[i:], altBSKKP) {
				out = append(out, p.sendKM.altBS...)
				i += len(altBSKKP)
				continue
			}
			// Legacy Alt+Backspace: \x1b\x7f (ESC + DEL) → kill to line start.
			if startsWith(data[i:], altBSLegacy) {
				out = append(out, p.sendKM.altBS...)
				i += len(altBSLegacy)
				continue
			}
			// Could the chunk-tail still grow into one of our markers
			// on the next read? Hold back only if data[i:] is a strict
			// prefix of *some* known pattern — unrelated escapes (arrow
			// keys, CSI sequences, etc.) pass through.
			held := false
			for _, pat := range holdbackPatterns {
				if isPrefixOf(data[i:], pat) {
					held = true

exec
/bin/zsh -lc "git show HEAD:workshop/issues/000087-fix-codex-alt-enter-remap.md | nl -ba | sed -n '1,75p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	---
     2	id: 000087
     3	status: working
     4	deps: []
     5	github_issue:
     6	created: 2026-06-29
     7	updated: 2026-06-29
     8	estimate_hours: 0.46
     9	started: 2026-06-29T18:25:25-07:00
    10	---
    11	
    12	# Fix Codex Alt Enter remap
    13	
    14	## Problem
    15	
    16	After #86, restarted sessions do invoke `zellij action send-keys "Alt Enter"` from the draft pane, but Codex still leaves the inserted draft text sitting in the composer. Live trace for `PAIR_TAG=2` at 2026-06-29T18:23:10-07:00 shows:
    17	
    18	- nvim wrote the body via `draft.send.write-body` (`body_len: 41`).
    19	- nvim invoked `draft.send.submit` as `zellij action send-keys Alt Enter`.
    20	- pair-wrap read the body bytes, then read Alt+Enter as `ESC CR` (`raw_len: 2`, SHA `a6d286d70768`), translated it to bare `CR` (`translated_len: 1`, SHA `9d1e0e2d9459`), and wrote that to the Codex PTY.
    21	
    22	So the Zellij action and pair-wrap Alt+Enter recognition work; the stale assumption is the Codex keymap row that collapses Alt+Enter to bare `CR`. Current Codex needs the modified Alt+Enter chord forwarded to the child PTY.
    23	
    24	## Spec
    25	
    26	For Codex only, pair-wrap should preserve the submit chord when it sees user Alt+Enter: legacy `ESC CR` and KKP `ESC [ 13 ; 3 u` should translate to `ESC CR` for the Codex child PTY, not bare `CR`. Plain Enter should continue translating to LF so the pair convention remains Enter = newline. Claude and agy keymaps should remain unchanged.
    27	
    28	Docs that describe Codex's keymap should say Codex plain Enter maps to newline and Alt+Enter is forwarded as the submit chord.
    29	
    30	## Done when
    31	
    32	- [x] Codex Alt+Enter translation emits `ESC CR`.
    33	- [x] Codex plain Enter still emits LF.
    34	- [x] Claude and agy keymaps are unchanged.
    35	- [x] Tests cover both legacy and KKP Alt+Enter inputs for Codex.
    36	- [x] Atlas/docs no longer claim Codex Alt+Enter maps to bare `CR`.
    37	
    38	## Plan
    39	
    40	- [x] Update the pure keymap tests first to expect Codex `altCR == ESC CR` and to expect both legacy and KKP Codex Alt+Enter inputs to translate to `ESC CR`.
    41	- [x] Change only `sendKeymapByAgent["codex"].altCR` in `cmd/pair-wrap/main.go`.
    42	- [x] Update Codex keymap docs in `atlas/architecture.md`, `atlas/how-to-bring-up-a-new-harness-cli.md`, and comments/tests that state the old bare-CR contract.
    43	- [x] Verify `go test ./cmd/pair-wrap`, focused nvim submit tests, issue validation, and whitespace.
    44	
    45	## Estimate
    46	
    47	```estimate
    48	model: estimate-logic-v3.1
    49	familiarity: 1.0
    50	item: smaller-go-module design=0.10 impl=0.12
    51	item: atlas-docs design=0.00 impl=0.08
    52	item: milestone-review design=0.00 impl=0.15
    53	design-buffer: 0.10
    54	total: 0.46
    55	```
    56	
    57	Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.
    58	
    59	## Log
    60	
    61	### 2026-06-29
    62	
    63	User restarted after #86 and still saw Alt+Return insert text into Codex without submitting. Investigated live traces instead of guessing. `zellij-actions-2.jsonl` confirmed the restarted draft pane is using `send-keys Alt Enter`; `wrap-events-2.jsonl` confirmed pair-wrap receives that as Alt+Enter and currently translates it to bare `CR`. Because bare `CR` reached Codex and did not submit, the stale contract is the Codex keymap's `altCR` output, not nvim focus, Zellij delivery, or pair-wrap input recognition (`ARCH-PURPOSE`).
    64	
    65	Implemented the Codex keymap correction with TDD. RED: `go test ./cmd/pair-wrap` failed on `codex.altCR: got "\r", want "\x1b\r"` and on both legacy/KKP Codex Alt+Enter translation cases. GREEN: changed only `sendKeymapByAgent["codex"].altCR` to `ESC CR`; `go test ./cmd/pair-wrap` passed. Synced atlas/docs to distinguish Codex from agy and verified with `go test ./cmd/pair-wrap`, `bash tests/queue-send-test.sh`, `bash tests/review-poke-test.sh`, `git diff --check`, and `sdlc issue validate workshop/issues/000087-fix-codex-alt-enter-remap.md`.

exec
/bin/zsh -lc "git show HEAD:cmd/pair-wrap/main.go | nl -ba | sed -n '144,158p'" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "git show HEAD:cmd/pair-wrap/keymap_registry_test.go | nl -ba | sed -n '20,29p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
   144		"codex": {
   145			// Codex plain Enter inserts a newline when rewritten to LF.
   146			// Current Codex submit requires the modified Alt+Enter chord
   147			// itself, so preserve that as legacy ESC+CR.
   148			plainCR: []byte{'\n'},
   149			altCR:   []byte{'\x1b', '\r'},
   150			altBS:   []byte{0x15}, // Ctrl+U — kill to line start
   151		},
   152		"agy": {
   153			// Antigravity (agy) follows the same Enter/Shift+Enter convention
   154			// as codex.
   155			plainCR: []byte{'\n'},
   156			altCR:   []byte{'\r'},
   157			altBS:   []byte{0x15}, // Ctrl+U — kill to line start
   158		},

 succeeded in 0ms:
    20		want := map[string]row{
    21			// claude reads `\<Enter>` as newline regardless of terminal
    22			// keyboard-protocol level — the documented portable path.
    23			"claude": {[]byte{'\\', '\r'}, []byte{'\r'}, ctrlU},
    24			// codex: plain Enter inserts newline; Alt+Enter stays a modified submit chord.
    25			"codex": {[]byte{'\n'}, []byte{'\x1b', '\r'}, ctrlU},
    26			// agy: same as codex.
    27			"agy": {[]byte{'\n'}, []byte{'\r'}, ctrlU},
    28		}
    29		if len(sendKeymapByAgent) != len(want) {

exec
/bin/zsh -lc "git show HEAD:atlas/how-to-bring-up-a-new-harness-cli.md | nl -ba | sed -n '20,35p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
    20	  var sendKeymapByAgent = map[string]sendKeymap{
    21	      "agy": {
    22	          plainCR: []byte{'\n'}, // plain Enter inserts newline
    23	          altCR:   []byte{'\r'}, // Alt+Enter sends query
    24	      },
    25	      "codex": {
    26	          plainCR: []byte{'\n'},       // plain Enter inserts newline
    27	          altCR:   []byte{'\x1b','\r'}, // Alt+Enter submit chord
    28	      },
    29	  }
    30	  ```
    31	- **Note:** Claude uses `\<Enter>` (`[]byte{'\\', '\r'}`) as a newline. Codex uses LF (`\n`) for newline and preserves Alt+Enter as `ESC CR` for submit. Antigravity (`agy`) uses LF (`\n`) for newline and CR (`\r`) for send.
    32	
    33	**Telemetry Signal** (aspect `1`, see §3): `return-remap` — `fired` each time a plain Enter is remapped to the agent's newline; `bypass` each time it passes through as a bare `\r` while an overlay is active. Emitted from `emitPlainCR`. The `fired:bypass` ratio is the health signal; an all-`bypass` or zero-`fired` session means the remap stopped engaging.
    34	
    35	---

codex
VERDICT: SHIP (confidence: high)

The change fulfills the issue purpose: Codex-specific Alt+Enter now remains `ESC CR`, plain Enter still maps to LF, and Claude/agy byte contracts remain unchanged. The implementation stays data-driven through the existing keymap table and translator, with focused tests covering both legacy and KKP Alt+Enter inputs. I found only stale comments that should be cleaned up when convenient.

1. Strengths:
- `cmd/pair-wrap/main.go:144` keeps the behavior change scoped to `sendKeymapByAgent["codex"].altCR`.
- `cmd/pair-wrap/translate_test.go:158` covers Codex plain Enter, legacy Alt+Enter, KKP Alt+Enter, mixed chunks, and paste preservation.
- `atlas/architecture.md:181` and `atlas/how-to-bring-up-a-new-harness-cli.md:31` document the new Codex-vs-agy distinction.
- The plan’s Core Concepts table matches the changed code paths: pure keymap data plus existing pure translation.

2. Critical findings:
- None.

3. Important findings:
- None.

4. Minor findings:
- `cmd/pair-wrap/main.go:153` says agy follows the same convention “as codex”; that is now misleading for the pair-wrap `altCR` contract.
- `cmd/pair-wrap/keymap_registry_test.go:26` says `agy: same as codex` while the expected `alt` bytes intentionally differ.

5. Test coverage notes:
- I did not run tests because this review prompt constrained tools to read-only review. Static inspection confirms the new tests cover the stated regression shape: Codex legacy `ESC CR`, KKP `ESC [ 13 ; 3 u`, and plain Enter LF.

6. Architectural notes for upcoming work:
- ARCH-DRY: Pass. The change reuses the existing keymap table and translator instead of adding Codex-specific branches.
- ARCH-PURE: Pass. Translation remains pure and unit-tested; IO stays in the stdin pump.
- ARCH-PURPOSE: Pass. The diff addresses the actual failure mode: Codex needs the modified submit chord forwarded, not bare CR.

7. Plan revision recommendations:
- None required.
tokens used
113,889
VERDICT: SHIP (confidence: high)

The change fulfills the issue purpose: Codex-specific Alt+Enter now remains `ESC CR`, plain Enter still maps to LF, and Claude/agy byte contracts remain unchanged. The implementation stays data-driven through the existing keymap table and translator, with focused tests covering both legacy and KKP Alt+Enter inputs. I found only stale comments that should be cleaned up when convenient.

1. Strengths:
- `cmd/pair-wrap/main.go:144` keeps the behavior change scoped to `sendKeymapByAgent["codex"].altCR`.
- `cmd/pair-wrap/translate_test.go:158` covers Codex plain Enter, legacy Alt+Enter, KKP Alt+Enter, mixed chunks, and paste preservation.
- `atlas/architecture.md:181` and `atlas/how-to-bring-up-a-new-harness-cli.md:31` document the new Codex-vs-agy distinction.
- The plan’s Core Concepts table matches the changed code paths: pure keymap data plus existing pure translation.

2. Critical findings:
- None.

3. Important findings:
- None.

4. Minor findings:
- `cmd/pair-wrap/main.go:153` says agy follows the same convention “as codex”; that is now misleading for the pair-wrap `altCR` contract.
- `cmd/pair-wrap/keymap_registry_test.go:26` says `agy: same as codex` while the expected `alt` bytes intentionally differ.

5. Test coverage notes:
- I did not run tests because this review prompt constrained tools to read-only review. Static inspection confirms the new tests cover the stated regression shape: Codex legacy `ESC CR`, KKP `ESC [ 13 ; 3 u`, and plain Enter LF.

6. Architectural notes for upcoming work:
- ARCH-DRY: Pass. The change reuses the existing keymap table and translator instead of adding Codex-specific branches.
- ARCH-PURE: Pass. Translation remains pure and unit-tested; IO stays in the stdin pump.
- ARCH-PURPOSE: Pass. The diff addresses the actual failure mode: Codex needs the modified submit chord forwarded, not bare CR.

7. Plan revision recommendations:
- None required.
