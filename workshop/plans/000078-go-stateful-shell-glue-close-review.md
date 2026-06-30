# Boundary Review — pair#78 (whole-issue close)

| field | value |
|-------|-------|
| issue | 78 — pair Go stateful shell glue |
| repo | pair |
| issue file | workshop/issues/000078-go-stateful-shell-glue.md |
| boundary | whole-issue close |
| milestone | — |
| window | 370d43b87ba89fae64a534526cbb51223d88df76..HEAD |
| command | sdlc close --issue 78 |
| reviewer | codex |
| timestamp | 2026-06-30T16:22:05-07:00 |
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
session id: 019f1ad5-b751-7490-a99d-651598e8664a
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **pair** repository.

- repository: pair   (root: /Users/xianxu/workspace/pair)
- issue:      pair#78   (file: workshop/issues/000078-go-stateful-shell-glue.md)
- window:     Base: 370d43b87ba89fae64a534526cbb51223d88df76   Head: HEAD

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
index 7b6c29a..8afd085 100644
--- a/Makefile.local
+++ b/Makefile.local
@@ -1,7 +1,7 @@
 # Repo-specific Makefile targets.
 # This file is included by Makefile — add your own targets here.
 
-.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
+.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-session-watch pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
 
 # Cross-cutting Go-binary targets. Every Go binary lives under cmd/<name>/
 # and ships with the `pair-` prefix so it doesn't collide with anything
@@ -26,7 +26,7 @@
 
 BIN_DIR     := bin
 INSTALL_DIR := $(HOME)/.local/bin
-GO_BINS     := pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-go
+GO_BINS     := pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-session-watch pair-go
 SHELL_BINS  := pair pair-dev
 
 build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
@@ -50,6 +50,7 @@ pair-slug:              $(BIN_DIR)/pair-slug
 pair-continuation:      $(BIN_DIR)/pair-continuation
 pair-changelog:         $(BIN_DIR)/pair-changelog
 pair-context:           $(BIN_DIR)/pair-context
+pair-session-watch:     $(BIN_DIR)/pair-session-watch
 pair-go:                $(BIN_DIR)/pair-go
 
 # Tests
@@ -80,7 +81,7 @@ test-cmux-ownership:
 
 # Session id watcher must ignore stale pidfiles from prior launches and wait
 # for pair-wrap to publish the current agent PID.
-test-session-watch:
+test-session-watch: $(BIN_DIR)/pair-session-watch
 	bash tests/pair-session-watch-test.sh
 
 # Review workbench (#66): the integration shell tests. The pure modules
@@ -239,6 +240,11 @@ $(BIN_DIR)/pair-slug: cmd/pair-slug/main.go cmd/pair-slug/slug.go cmd/internal/m
 $(BIN_DIR)/pair-context: cmd/pair-context/main.go cmd/internal/contextcmd/contextcmd.go cmd/internal/ctxmeter/ctxmeter.go cmd/internal/transcript/transcript.go go.mod
 	go build -o $@ ./cmd/pair-context
 
+# pair-session-watch — async codex/agy session-id discovery. Legacy
+# bin/pair-session-watch.sh remains as a compatibility shim for bin/pair.
+$(BIN_DIR)/pair-session-watch: cmd/pair-session-watch/main.go cmd/internal/sessionwatch/sessionwatch.go cmd/internal/sessionwatch/run.go cmd/internal/sessionwatch/runtime.go cmd/internal/adapt/adapt.go go.mod
+	go build -o $@ ./cmd/pair-session-watch
+
 # pair-changelog — distills the pair TTY into the Alt+l change log (#53);
 # shares cmd/internal/model with pair-slug.
 $(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill.go cmd/pair-changelog/prompt.go cmd/internal/model/model.go go.mod
diff --git a/atlas/architecture.md b/atlas/architecture.md
index c8df19e..bf73929 100644
--- a/atlas/architecture.md
+++ b/atlas/architecture.md
@@ -19,7 +19,7 @@ bin/clipboard-to-pane.sh     # read clipboard, hand off to nvim's PairPasteQuote
 bin/copy-on-select.sh        # invoked by zellij copy_command on mouse-up
 bin/pair-quit.sh             # invoked by Alt+x — marks + kills session
 bin/pair-restart.sh          # invoked by Alt+n / Shift+Alt+N — marks (quit + restart) + kills session
-bin/pair-session-watch.sh    # captures codex/agy session id at create time (#000016, #000020)
+bin/pair-session-watch.sh    # shim to Go watcher that captures codex/agy session id (#000016, #000020, #78)
 bin/pair-wrap                # PTY proxy: OSC translation + scrollback capture
 bin/pair-notify              # hook-driven OSC notifier (e.g. claude Notification)
 bin/pair-scrollback-render   # raw PTY capture → ANSI-colored line dump (#000017)
@@ -564,7 +564,7 @@ A pair *tag* is a durable identity for a coding session: it survives Alt+d (deta
 1. **Pre-write at launch (`bin/pair`).** Two paths:
    - `--resume <id>` / `resume <id>` explicit on argv: pair writes `config-<tag>-<agent>.json` directly with that id, before zellij launch.
    - **Claude fresh launch (issue #000020):** claude supports `--session-id <uuid>`, so on the new-session path pair generates a v4 UUID, injects the flag into the agent argv, and writes the config synchronously *before* spawning the watcher. The id is deterministic from the launcher's perspective, so the watcher is a no-op for claude — and the cross-tag race that existed when two pair sessions shared a cwd is structurally eliminated.
-2. **Watcher (`bin/pair-session-watch.sh`, codex/agy only).** Spawned in the background by `bin/pair` on the create path, right before the zellij launch. Two discovery paths:
+2. **Watcher (`cmd/pair-session-watch` via `bin/pair-session-watch.sh`, codex/agy only).** Spawned in the background by `bin/pair` on the create path, right before the zellij launch. The `.sh` file is now a compatibility shim; the stateful discovery logic lives in Go. Two discovery paths:
    - **PID-bound (preferred).** Reads `$PAIR_DATA_DIR/agent-pid-<tag>` (written by pair-wrap right after `pty.Start`) only when the pidfile's mtime is at-or-after the watcher's start, so a stale pidfile from a prior launch is ignored until pair-wrap overwrites it. Then it inspects open files in that PID's process tree via `lsof -p <pid> -Fn`. Race-free across concurrent pair sessions because lsof output is scoped to specific PIDs. Falls back internally to a birth-time-filtered directory walk if the agent doesn't keep its session file open: candidates are files with `stat -f %B >= agent_start_epoch`, and only a *single* candidate is accepted (multiple = concurrent race, refuse rather than guess).
    - **Legacy snapshot-diff (fallback).** Used when a fresh pidfile doesn't appear within 2s (`PAIR_SESSION_WATCH_PID_WAIT_SECONDS` in tests) — i.e., when the installed pair-wrap binary predates #000020 and doesn't publish the pidfile, or a stale pidfile is never refreshed. Behaves identically to pre-#000020: snapshots the watch dir at start, picks the first new file. Cross-tag races re-emerge in this path, so the proper resolution is to rebuild pair-wrap.
 
@@ -654,9 +654,9 @@ Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/outer-tty-<tag>` — single-lin
 
 Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-<tag>` — single-line file recording which agent binary was launched in the session (`claude`, `codex`, ...). Written once at session create; read by `pair list` to display the agent column, and by `bin/pair`'s tag-restart agent-inference. Removed on full quit. The agent isn't otherwise recoverable post-create — env vars are frozen in pane shells, and custom session names (e.g. `pair-bugfix`) don't carry the agent in the name.
 
-Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/config-<tag>-<agent>.json` — saved restart configuration for `(tag, agent)` (issue #000016, #000020). `{ agent, args, session_id }`. For claude, written synchronously by `bin/pair` before zellij launch (`--session-id` is deterministic). For codex/agy, written by `bin/pair-session-watch.sh` once the agent's session file is discovered via lsof. Read by `bin/pair`'s create-flow prompt and by the post-Alt+x hint. Survives Alt+x (unlike `agent-<tag>`, which is cleared) — that's the whole point: it's the bridge between two pair launches against the same tag.
+Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/config-<tag>-<agent>.json` — saved restart configuration for `(tag, agent)` (issue #000016, #000020). `{ agent, args, session_id }`. For claude, written synchronously by `bin/pair` before zellij launch (`--session-id` is deterministic). For codex/agy, written by the Go `pair-session-watch` command once the agent's session file is discovered via lsof. Read by `bin/pair`'s create-flow prompt and by the post-Alt+x hint. Survives Alt+x (unlike `agent-<tag>`, which is cleared) — that's the whole point: it's the bridge between two pair launches against the same tag.
 
-Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-pid-<tag>` — child agent PID written by `cmd/pair-wrap` immediately after `pty.Start`, removed on shutdown. Consumed by `bin/pair-session-watch.sh` to scope `lsof` discovery to a specific process tree (issue #000020). Mtime is also used as the agent-start epoch in the watcher's birth-time fallback.
+Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-pid-<tag>` — child agent PID written by `cmd/pair-wrap` immediately after `pty.Start`, removed on shutdown. Consumed by `cmd/pair-session-watch` to scope `lsof` discovery to a specific process tree (issue #000020). Mtime is also used as the agent-start epoch in the watcher's birth-time fallback.
 
 Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/nvim-pid-<tag>-{draft,scrollback}` — single-line file containing the pid of an `nvim --embed` server child. Written at VimEnter by `nvim/init.lua` (for the draft pane) and `nvim/scrollback.lua` (for the Alt+/ floating viewer) when `$PAIR_NVIM_PID_FILE` is set; the launch sites (`zellij/layouts/main.kdl` for draft, `bin/pair-scrollback-open` for scrollback) export the env var pointing at a tag-scoped path. Read and removed by `cleanup_quit_marker` on Alt+x to SIGKILL the embed deterministically — without this, the embed sometimes survives zellij's pane teardown and accumulates as a PPID=1 orphan, dragging the host into memory pressure across many quits.
 
@@ -666,7 +666,7 @@ Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/image-capture-<tag>` + `image-c
 
 Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/slug-proposed-<tag>` and `slug-<tag>` — the orientation-slug channel (issue #000027). `pair-slug` (spawned by pair-wrap at turn-end) writes the proposed `=== <branch> | <focus> ===` to `slug-proposed-<tag>` (atomic temp+rename); nvim applies it to draft line 1 and writes the effective line back to `slug-<tag>`, which is the `prev` the proposer reads next turn. For Codex, if `config-<tag>-codex.json` is missing, `pair-slug` can recover the live rollout by reading `agent-pid-<tag>`, walking descendants via `ps`, and checking their `lsof` paths for `~/.codex/sessions/.../rollout-*.jsonl`. Similarly, for agy, the transcript is resolved from `~/.gemini/antigravity-cli/brain/<session_id>/.system_generated/logs/transcript.jsonl`. Codex model auth is API-key first, then Codex CLI subscription auth via `codex exec`. Single writer each, so the channel is race-free.
 
-Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/adapt-<tag>.jsonl` — the adaptation flight recorder (issue #000045). One JSON line per harness-adaptation trigger (`{ts, comp, agent, aspect, signal, outcome, detail}`), appended concurrently by `cmd/pair-wrap`, `cmd/pair-slug`, `bin/pair-session-watch.sh`, and `nvim/adapt.lua` — all writing one shared schema. Truncated once at session launch by `bin/pair` (so multi-process `O_APPEND` never races) and removed on full quit. Read by `doctor/doctor.sh` to surface integration drift (near-miss/fail signals). See `atlas/how-to-bring-up-a-new-harness-cli.md` §3 for the signal registry.
+Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/adapt-<tag>.jsonl` — the adaptation flight recorder (issue #000045). One JSON line per harness-adaptation trigger (`{ts, comp, agent, aspect, signal, outcome, detail}`), appended concurrently by `cmd/pair-wrap`, `cmd/pair-slug`, `cmd/pair-session-watch`, and `nvim/adapt.lua` — all writing one shared schema. Truncated once at session launch by `bin/pair` (so multi-process `O_APPEND` never races) and removed on full quit. Read by `doctor/doctor.sh` to surface integration drift (near-miss/fail signals). See `atlas/how-to-bring-up-a-new-harness-cli.md` §3 for the signal registry.
 
 **Migration from v1:** the launcher detects old `~/scratch/pair-{draft,log}-*.md` files on startup and moves them to the new XDG location, stripping the redundant `pair-` prefix from filenames.
 
diff --git a/atlas/go-migration-inventory.md b/atlas/go-migration-inventory.md
index 7a61e48..bb13dbd 100644
--- a/atlas/go-migration-inventory.md
+++ b/atlas/go-migration-inventory.md
@@ -68,8 +68,8 @@ Priority is packaging impact first, then reliability/testability:
 | `bin/pair-changelog-open` | POSIX shell orchestrator | zellij Alt+l Run | Opens changelog viewer and starts detached render/distill singleton. | Requires Pair env; calls renderer, `pair-changelog`, setsid/perl, nvim; reads/writes `changelog-*` sidecars. | shell-glue now; candidate Go orchestration after entrypoint | P1 |
 | `nvim/changelog.lua` | Neovim native asset | `bin/pair-changelog-open` | Loaded by `nvim -u ... <log>`; read-only watcher/spinner. | Reads `PAIR_CHANGELOG_*` and Pair env. | native-asset, adjacent/embedded | P1 |
 | `bin/pair-title.sh` | Bash stateful poller | `bin/pair` ensure_title_poller | `pair-title.sh <tag> <agent>`; long-running 60s poller; test hook `PAIR_TITLE_TEST_CALL`. | Reads/writes title pid, pane json, cmux owner files; calls `pair-context`, zellij, ps, cmux. | stateful shell-glue; explicit #78 candidate | P1 |
-| `bin/pair-session-watch.sh` | Bash stateful watcher | `bin/pair` create path | `pair-session-watch.sh <agent> <tag> <cwd> [agent-args...]`; background 60s watcher; no-op for claude. | Reads agent pidfile, lsof/ps, native session dirs; writes config JSON atomically; logs adapt events. | stateful shell-glue; explicit #78 candidate | P1 |
-| `bin/lib/adapt-log.sh` | sourced shell helper | `pair-session-watch.sh` | `adapt_log comp agent aspect signal outcome [detail]`; no-op if no `PAIR_TAG` or jq. | Appends JSONL to `$PAIR_DATA_DIR/adapt-<tag>.jsonl`. | keep until shell emitters move; schema must stay DRY with Go/Lua emitters | P1 |
+| `bin/pair-session-watch.sh` / `cmd/pair-session-watch` / `cmd/internal/sessionwatch` | Shell compatibility shim plus Go stateful watcher | `bin/pair` create path | `pair-session-watch.sh <agent> <tag> <cwd> [agent-args...]` execs the Go command; background 60s watcher; no-op for claude. | Reads agent pidfile, lsof/ps, native session dirs; writes config JSON atomically; logs adapt events through `cmd/internal/adapt`. | Go-owned watcher with legacy shim retained while `bin/pair` calls the `.sh` name (#78) | P1 |
+| `bin/lib/adapt-log.sh` | sourced shell helper | remaining shell emitters | `adapt_log comp agent aspect signal outcome [detail]`; no-op if no `PAIR_TAG` or jq. | Appends JSONL to `$PAIR_DATA_DIR/adapt-<tag>.jsonl`. | keep until remaining shell emitters move; schema stays DRY with Go/Lua emitters | P1 |
 | `nvim/adapt.lua` | Lua helper | nvim doctor/adaptation surfaces, tests | Lua adaptation flight recorder emitter. | Writes same JSONL schema as Go/shell. | native-asset; keep schema aligned | P2 |
 | `doctor/README.md` / `doctor/SKILL.md` | docs/skill | operator/agent diagnostics | Documents Pair doctor flow. | Refers to `nvim/doctor.lua` and adaptation logs. | adjacent docs/skill; not Go migration target | P3 |
 | `nvim/doctor.lua` | Lua helper | `:PairDoctor` in nvim | Builds agent instruction payload. | Reads `PAIR_HOME`; sends text through draft/agent flow. | native-asset | P2 |
@@ -134,8 +134,10 @@ Build/install callers:
 - #77 made `pair-go launch ...` a meaningful Go-owned compatibility handoff to
   `bin/pair`, with argv/env preserved and missing-launcher diagnostics. The
   stable public `pair` script remains unchanged for this migration window.
-- #78 should prioritize `pair-title.sh` and `pair-session-watch.sh` if stateful
-  shell remains a packaging/reliability problem after #77.
+- #78 ported the session-id watcher to `cmd/pair-session-watch` with
+  `bin/pair-session-watch.sh` retained as a shim. `pair-title.sh` remains the
+  next stateful shell candidate because it owns a separate UI title-poller
+  surface.
 - #79 owns whether `nvim/` and `zellij/` are embedded or installed adjacent.
 
 ## Coverage Ledger
diff --git a/atlas/how-to-bring-up-a-new-harness-cli.md b/atlas/how-to-bring-up-a-new-harness-cli.md
index 157ed3b..84333fd 100644
--- a/atlas/how-to-bring-up-a-new-harness-cli.md
+++ b/atlas/how-to-bring-up-a-new-harness-cli.md
@@ -56,8 +56,8 @@ If the agent presents blocking overlays, pickers (like file autocompletes), or y
 `pair` features a robust restart-in-place (`Alt+n`) and session reattach (`pair resume <tag>`) mechanism. To make this work, the launcher needs to discover the agent's unique conversation/session ID as soon as it is spawned.
 
 **Discovery & Watcher:**
-- **File:** [bin/pair-session-watch.sh](file:///Users/xianxu/workspace/pair/bin/pair-session-watch.sh)
-- Since TUI agents do not always expose session IDs on stdout, `pair-session-watch.sh` runs in the background. It finds the agent process PID from `$PAIR_DATA_DIR/agent-pid-<tag>` (written by `pair-wrap`), walks its descendants, and inspects files held open by the processes via `lsof -p <pid>`.
+- **Files:** `cmd/pair-session-watch` and `cmd/internal/sessionwatch` (`bin/pair-session-watch.sh` remains a compatibility shim).
+- Since TUI agents do not always expose session IDs on stdout, `pair-session-watch` runs in the background. It finds the agent process PID from `$PAIR_DATA_DIR/agent-pid-<tag>` (written by `pair-wrap`), walks its descendants, and inspects files held open by the processes via `lsof -p <pid>`.
 - Configure the agent's session file criteria:
   ```bash
   agy)
@@ -95,7 +95,7 @@ If the agent presents blocking overlays, pickers (like file autocompletes), or y
       ;;
   ```
 
-**Telemetry Signal** (aspect `3`, see §3): `session-id` from `pair-session-watch.sh` — `fired` when `extract_id` resolves an id and the config is written, **`near-miss`** when a file matching the watch pattern is found but no id can be extracted (filename/format drift), `fail` when the 60s watch window elapses with no id at all (the session file never appeared where expected). The resume mapping in `bin/pair` is the *consumer* of this id; it's static config with no separate signal.
+**Telemetry Signal** (aspect `3`, see §3): `session-id` from `pair-session-watch` — `fired` when `ExtractSessionID` resolves an id and the config is written, **`near-miss`** when a file matching the watch pattern is found but no id can be extracted (filename/format drift), `fail` when the 60s watch window elapses with no id at all (the session file never appeared where expected). The resume mapping in `bin/pair` is the *consumer* of this id; it's static config with no separate signal.
 
 ---
 
@@ -151,7 +151,7 @@ When introducing a new agent `<name>`, ensure you complete each item:
 
 1. [ ] **Verify Return Key remapping** in `sendKeymapByAgent` (Enter = newline, Alt+Enter = send).
 2. [ ] **Check for blocking TUI overlays** and implement a PTY overlay detector in `overlayDetectorByAgent` if needed.
-3. [ ] **Implement Session Watching** in `bin/pair-session-watch.sh` (using `lsof` and target file patterns).
+3. [ ] **Implement Session Watching** in `cmd/internal/sessionwatch` / `cmd/pair-session-watch` (using `lsof` and target file patterns).
 4. [ ] **Configure Launcher Recovery** in `bin/pair` (mapping `--conversation` or `--resume` flags).
 5. [ ] **Add slug generation support** in `pair-slug` (transcript parsing + sandboxed print execution).
 6. [ ] **Confirm mouse scroll and scrollback render** work smoothly without drawing glitch issues.
@@ -200,7 +200,7 @@ write the same line shape directly):
 |---|---|---|---|---|
 | 1 Return remap | `return-remap` | pair-wrap | fired, bypass | zero `fired` / all `bypass` |
 | 2 Overlay suspend | `overlay-detect` | pair-wrap | fired, near-miss | any `near-miss` |
-| 3 Session watch | `session-id` | pair-session-watch.sh | fired, near-miss, fail | `fail` (timeout) / `near-miss` (file found, id unparsed) |
+| 3 Session watch | `session-id` | pair-session-watch | fired, near-miss, fail | `fail` (timeout) / `near-miss` (file found, id unparsed) |
 | 4 Slug gen | `slug-parse` | pair-slug | fired, near-miss, fail | `near-miss` (transcript parsed, 0 turns) / `fail` (resolved a transcript but couldn't read/parse it) |
 | 5 PTY filter | `output-filter` | pair-wrap | fired | a `fired` line that *stops* appearing (its absence is the signal — the sequence was renamed) |
 | 6 Settings | — | — | — | static config; no signal |
diff --git a/bin/pair-session-watch.sh b/bin/pair-session-watch.sh
index 3f45147..9b23879 100755
--- a/bin/pair-session-watch.sh
+++ b/bin/pair-session-watch.sh
@@ -1,311 +1,21 @@
 #!/usr/bin/env bash
-# pair-session-watch.sh — capture the agent's session-id by inspecting
-# files held open by the agent's process tree, then write a per-(tag,agent)
-# restart config under $PAIR_DATA_DIR.
-#
-# Usage:
-#   pair-session-watch.sh <agent> <tag> <cwd> [agent-args...]
-#
-# Spawned in the background by bin/pair right before zellij launch on the
-# new-session path. Issue #000020 replaced the earlier "first new file in
-# the watch dir" snapshot with PID-bound discovery: two pair sessions in
-# the same cwd previously raced to claim whichever agent's session file
-# appeared first, occasionally cross-wiring tags' configs.
-#
-# Per-agent surface:
-#   claude  — no-op. bin/pair pre-injects `--session-id <uuid>` and writes
-#             config-<tag>-claude.json synchronously, so there's nothing
-#             left to discover at runtime.
-#   codex   — open file under ~/.codex/sessions/.../rollout-*-<uuid>.jsonl.
-#             id = trailing UUID in filename.
-#
-#
-# The pidfile ($PAIR_DATA_DIR/agent-pid-<tag>) is dropped by pair-wrap
-# right after pty.Start; we wait briefly for it, then poll
-# `lsof -p <pid>` against that pid + descendants.
-#
-# Config write is atomic (tmp + rename). Watcher self-times-out after 60s
-# so a stuck agent doesn't leak the background process.
+# Compatibility shim for the Go-owned pair-session-watch command.
 
 set -uo pipefail
 
-agent="${1:-}"
-tag="${2:-}"
-cwd="${3:-}"
-[ -z "$agent" ] || [ -z "$tag" ] || [ -z "$cwd" ] && exit 0
-shift 3
-args=( "$@" )
-
-# Claude is fully handled at launch time. Codex and agy need lsof.
-case "$agent" in
-    codex|agy) ;;
-    *) exit 0 ;;
-esac
-
-DATA_DIR="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
-mkdir -p "$DATA_DIR"
-out="$DATA_DIR/config-$tag-$agent.json"
-pid_file="$DATA_DIR/agent-pid-$tag"
-
-# Aspect 3 flight recorder (atlas §3). PAIR_TAG is normally exported by bin/pair;
-# fall back to our positional tag so logging works even if run standalone.
-: "${PAIR_TAG:=$tag}"
-export PAIR_TAG
-_swdir="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
-# shellcheck source=bin/lib/adapt-log.sh
-. "$_swdir/lib/adapt-log.sh" 2>/dev/null || true
-nm_logged=0 # dedup the session-id near-miss across poll iterations
-
-# Per-agent: directory we walk + the find pattern. Used by both the
-# PID-bound primary path (for lsof path matching) and the legacy
-# snapshot-diff fallback (for pair-wrap binaries that don't publish
-# the pidfile yet).
-case "$agent" in
-    codex)
-        watch_dir="$HOME/.codex/sessions"
-        find_args=(-type f -name 'rollout-*.jsonl')
-        ;;
-
-    agy)
-        watch_dir="$HOME/.gemini/antigravity-cli/conversations"
-        find_args=(-type f -name '*.db')
-        ;;
-esac
-mkdir -p "$watch_dir"
-
-# Wait briefly for pair-wrap to publish this launch's agent PID. A pidfile from
-# a prior run can survive until pair-wrap overwrites it; binding to that stale
-# PID makes the watcher exit before the current agent starts.
-watch_start=$(date +%s)
-pid_wait_seconds="${PAIR_SESSION_WATCH_PID_WAIT_SECONDS:-2}"
-case "$pid_wait_seconds" in
-    ''|*[!0-9]*) pid_wait_seconds=2 ;;
-esac
-pid_deadline=$(( watch_start + pid_wait_seconds ))
-pid_fresh() {
-    [ -s "$pid_file" ] || return 1
-    mt=$(stat -f %m "$pid_file" 2>/dev/null || echo 0)
-    [ "$mt" -ge "$watch_start" ]
-}
-while ! pid_fresh && [ "$(date +%s)" -lt "$pid_deadline" ]; do
-    sleep 0.1
+SOURCE="${BASH_SOURCE[0]}"
+while [ -L "$SOURCE" ]; do
+    DIR="$(cd -P "$(dirname "$SOURCE")" && pwd)"
+    SOURCE="$(readlink "$SOURCE")"
+    [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
 done
+PAIR_HOME="$(cd -P "$(dirname "$SOURCE")/.." && pwd)"
+export PAIR_HOME
 
-root_pid=""
-agent_start=0
-if pid_fresh; then
-    root_pid=$(cat "$pid_file" 2>/dev/null)
-    # pair-wrap writes the pidfile right after pty.Start, so its mtime
-    # is a tight upper bound on the agent's start epoch. Used as a
-    # birth-time floor: any session file whose birth predates this was
-    # created by an earlier pair session, not ours.
-    agent_start=$(stat -f %m "$pid_file" 2>/dev/null || echo 0)
+cmd="$PAIR_HOME/bin/pair-session-watch"
+if [ ! -x "$cmd" ]; then
+    echo "pair-session-watch.sh: missing Go watcher at $cmd; run make pair-session-watch or source ../ariadne/construct/dev-aliases.sh in a dev shell" >&2
+    exit 1
 fi
 
-# Legacy fallback state: snapshot the watch dir at start. Only consulted
-# when the PID-bound path can't bind (no pidfile) — preserves the
-# pre-#000020 behavior so old pair-wrap installs still capture sessions
-# in the single-session case. Cross-tag races re-emerge in that path;
-# the proper fix is to rebuild pair-wrap so the pidfile shows up.
-legacy_existing=""
-if [ -z "$root_pid" ]; then
-    legacy_existing=$(find "$watch_dir" "${find_args[@]}" 2>/dev/null | sort)
-fi
-
-# pid + descendants. Codex/agy may keep the session file open in a native
-# child below a JS launcher process. Use ps instead of pgrep -P: on macOS,
-# pgrep can miss children that ps still reports, which makes the PID-bound
-# discovery path inspect only the launcher and miss Codex's rollout fd.
-descendants() {
-    local root="$1"
-    ps -axo pid=,ppid= 2>/dev/null | awk -v root="$root" '
-        { children[$2] = children[$2] " " $1 }
-        END {
-            queue[1] = root
-            seen[root] = 1
-            head = 1
-            tail = 1
-            while (head <= tail) {
-                p = queue[head++]
-                print p
-                n = split(children[p], kids, " ")
-                for (i = 1; i <= n; i++) {
-                    k = kids[i]
-                    if (k != "" && !seen[k]) {
-                        seen[k] = 1
-                        queue[++tail] = k
-                    }
-                }
-            }
-        }'
-}
-
-match_path() {
-    local line="$1"
-    case "$agent" in
-        codex)
-            case "$line" in
-                "$HOME/.codex/sessions/"*"/rollout-"*".jsonl") echo "$line" ;;
-            esac
-            ;;
-
-        agy)
-            case "$line" in
-                "$HOME/.gemini/antigravity-cli/conversations/"*".db") echo "$line" ;;
-            esac
-            ;;
-    esac
-}
-
-extract_id() {
-    case "$agent" in
-        codex)
-            local fn
-            fn=$(basename "$1" .jsonl)
-            if [[ "$fn" =~ ([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$ ]]; then
-                echo "${BASH_REMATCH[1]}"
-            fi
-            ;;
-
-        agy)
-            # Path is like: ~/.gemini/antigravity-cli/conversations/<uuid>.db
-            # The UUID is the basename of the file without the .db extension.
-            local fn
-            fn=$(basename "$1" .db)
-            if [[ "$fn" =~ ^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$ ]]; then
-                echo "$fn"
-            fi
-            ;;
-    esac
-}
-
-deadline=$(( $(date +%s) + 60 ))
-while [ "$(date +%s)" -lt "$deadline" ]; do
-    # If we have a root_pid and the agent's gone, nothing more to do.
-    if [ -n "$root_pid" ]; then
-        kill -0 "$root_pid" 2>/dev/null || exit 0
-    fi
-
-    sid=""
-    matched_file=""
-    nm_candidate="" # a file that matched our pattern but yielded no id this pass
-
-    if [ -n "$root_pid" ]; then
-        # Primary path: lsof against the agent's PID tree. Race-free
-        # because lsof output is scoped to specific PIDs, so a peer pair
-        # session in the same cwd can't masquerade as ours.
-        while IFS= read -r p; do
-            [ -z "$p" ] && continue
-            # `lsof -Fn` emits one record per fd: 'p' header line then 'n'
-            # for the path. We only care about the n-prefixed lines.
-            while IFS= read -r line; do
-                [ "${line:0:1}" = "n" ] || continue
-                path="${line:1}"
-                hit=$(match_path "$path")
-                [ -z "$hit" ] && continue
-                cand=$(extract_id "$hit")
-                if [ -n "$cand" ]; then
-                    sid="$cand"
-                    matched_file="$hit"
-                    break 2
-                fi
-                nm_candidate="$hit"
-            done < <(lsof -p "$p" -Fn 2>/dev/null)
-        done < <(descendants "$root_pid")
-
-        # Birth-time fallback: lsof can miss agents that close the fd
-        # between writes. Walk the watch dir for files born at-or-after
-        # our agent's start epoch (so files from earlier pair sessions
-        # can't match), and accept only when there's exactly one
-        # candidate — multiple = concurrent race, refuse rather than
-        # guess wrong.
-        if [ -z "$sid" ] && [ -n "${watch_dir:-}" ] && [ "$agent_start" -gt 0 ]; then
-            candidates=()
-            while IFS= read -r f; do
-                [ -z "$f" ] && continue
-                bt=$(stat -f %B "$f" 2>/dev/null || echo 0)
-                [ "$bt" -ge "$agent_start" ] && candidates+=("$f")
-            done < <(find "$watch_dir" "${find_args[@]}" 2>/dev/null)
-            if [ "${#candidates[@]}" -eq 1 ]; then
-                cand=$(extract_id "${candidates[0]}")
-                if [ -n "$cand" ]; then
-                    sid="$cand"
-                    matched_file="${candidates[0]}"
-                else
-                    nm_candidate="${candidates[0]}"
-                fi
-            fi
-        fi
-    else
-        # Legacy snapshot-diff path: pair-wrap didn't publish a pidfile
-        # (older binary). Behaves identically to pre-#000020 — first new
-        # file in the watch dir wins. Cross-tag race re-emerges here, but
-        # we'd rather capture in the common single-session case than fail
-        # silently. Rebuild pair-wrap to upgrade to the PID-bound path.
-        current=$(find "$watch_dir" "${find_args[@]}" 2>/dev/null | sort)
-        new=$(comm -13 <(printf '%s\n' "$legacy_existing") <(printf '%s\n' "$current"))
-        if [ -n "$new" ]; then
-            while IFS= read -r f; do
-                [ -z "$f" ] && continue
-                cand=$(extract_id "$f")
-                if [ -n "$cand" ]; then
-                    sid="$cand"
-                    matched_file="$f"
-                    break
-                fi
-                nm_candidate="$f"
-            done <<< "$new"
-        fi
-    fi
-
-    # Drift fingerprint: we found a file matching the agent's session-file
-    # pattern but extract_id couldn't pull an id out of it — likely the
-    # filename/format changed. Log once per session (the poll loop reruns).
-    if [ -z "$sid" ] && [ -n "$nm_candidate" ] && [ "$nm_logged" = 0 ]; then
-        adapt_log session-watch "$agent" 3 session-id near-miss \
-            "matched session file but no id extracted: $(basename "$nm_candidate")"
-        nm_logged=1
-    fi
-
-    if [ -n "$sid" ]; then
-        # Strip --resume <id> / `resume <id>` so saved args don't carry
-        # the resume binding into future relaunches — session_id below is
-        # the canonical store. Same shape as bin/pair's stripping; keep
-        # in sync.
-        stripped=()
-        n=${#args[@]}
-        i=0
-        if [ "$agent" = "codex" ] && [ $n -ge 2 ] \
-            && [ "${args[0]}" = "resume" ]; then
-            i=2
-        fi
-        while [ $i -lt $n ]; do
-            if [ "${args[$i]}" = "--resume" ]; then
-                i=$((i+2))
-            else
-                stripped+=("${args[$i]}")
-                i=$((i+1))
-            fi
-        done
-
-        tmp=$(mktemp "$out.XXXXXX") || exit 0
-        if jq -n \
-              --arg agent "$agent" \
-              --arg sid "$sid" \
-              '{ agent: $agent, args: $ARGS.positional, session_id: $sid }' \
-              --args -- ${stripped[@]+"${stripped[@]}"} > "$tmp"
-        then
-            mv "$tmp" "$out"
-            adapt_log session-watch "$agent" 3 session-id fired "session_id=$sid"
-        else
-            rm -f "$tmp"
-        fi
-        exit 0
-    fi
-    sleep 0.1
-done
-
-# Watched the full window without resolving an id: the session file never
-# appeared where/how we expect — the strongest drift signal for this aspect.
-adapt_log session-watch "$agent" 3 session-id fail "no session id within 60s deadline (agent=$agent)"
-exit 0
+exec "$cmd" "$@"
diff --git a/cmd/internal/sessionwatch/run.go b/cmd/internal/sessionwatch/run.go
new file mode 100644
index 0000000..208def3
--- /dev/null
+++ b/cmd/internal/sessionwatch/run.go
@@ -0,0 +1,175 @@
+package sessionwatch
+
+import (
+	"path/filepath"
+	"strings"
+	"time"
+
+	"github.com/xianxu/pair/cmd/internal/adapt"
+)
+
+// Options are the watcher inputs after CLI/env resolution.
+type Options struct {
+	Agent   string
+	Tag     string
+	Cwd     string
+	Args    []string
+	Home    string
+	DataDir string
+	PIDWait time.Duration
+	Timeout time.Duration
+	Poll    time.Duration
+}
+
+// Runtime is the IO boundary for the session watcher.
+type Runtime interface {
+	Now() time.Time
+	Sleep(time.Duration)
+	ReadFile(path string) ([]byte, error)
+	ModTime(path string) (time.Time, error)
+	BirthTime(path string) (time.Time, error)
+	ListFiles(root string) ([]string, error)
+	Descendants(root string) ([]string, error)
+	LsofPaths(pid string) ([]string, error)
+	ProcessAlive(pid string) bool
+	AtomicWrite(path string, data []byte) error
+	Log(outcome adapt.Outcome, detail string)
+}
+
+// Run discovers the async agent session id and writes config-<tag>-<agent>.json.
+func Run(opts Options, rt Runtime) error {
+	spec, ok := SpecForAgent(opts.Agent, opts.Home)
+	if !ok || opts.Tag == "" || opts.DataDir == "" {
+		return nil
+	}
+	if opts.PIDWait <= 0 {
+		opts.PIDWait = 2 * time.Second
+	}
+	if opts.Timeout <= 0 {
+		opts.Timeout = 60 * time.Second
+	}
+	if opts.Poll <= 0 {
+		opts.Poll = 100 * time.Millisecond
+	}
+
+	watchStart := rt.Now()
+	pidFile := filepath.Join(opts.DataDir, "agent-pid-"+opts.Tag)
+	out := filepath.Join(opts.DataDir, "config-"+opts.Tag+"-"+opts.Agent+".json")
+
+	pidDeadline := watchStart.Add(opts.PIDWait)
+	for {
+		if fresh, _ := freshPID(pidFile, watchStart, rt); fresh {
+			break
+		}
+		if !rt.Now().Before(pidDeadline) {
+			break
+		}
+		rt.Sleep(opts.Poll)
+	}
+
+	rootPID := ""
+	agentStart := time.Time{}
+	if fresh, mod := freshPID(pidFile, watchStart, rt); fresh {
+		if data, err := rt.ReadFile(pidFile); err == nil {
+			rootPID = strings.TrimSpace(string(data))
+			agentStart = mod
+		}
+	}
+
+	legacyExisting := map[string]bool{}
+	if rootPID == "" {
+		files, _ := rt.ListFiles(spec.WatchDir)
+		for _, file := range files {
+			legacyExisting[file] = true
+		}
+	}
+
+	nmLogged := false
+	deadline := watchStart.Add(opts.Timeout)
+	for rt.Now().Before(deadline) {
+		if rootPID != "" && !rt.ProcessAlive(rootPID) {
+			return nil
+		}
+
+		result := discover(spec, rootPID, agentStart, legacyExisting, rt)
+		if result.ID != "" {
+			payload, err := ConfigJSON(ConfigPayload{
+				Agent:     opts.Agent,
+				Args:      StripResumeArgs(opts.Agent, opts.Args),
+				SessionID: result.ID,
+			})
+			if err != nil {
+				return err
+			}
+			if err := rt.AtomicWrite(out, payload); err != nil {
+				return err
+			}
+			rt.Log(adapt.Fired, "session_id="+result.ID)
+			return nil
+		}
+		if result.NearMiss && !nmLogged {
+			rt.Log(adapt.NearMiss, "matched session file but no id extracted: "+filepath.Base(result.Path))
+			nmLogged = true
+		}
+
+		rt.Sleep(opts.Poll)
+	}
+
+	rt.Log(adapt.Fail, "no session id within 60s deadline (agent="+opts.Agent+")")
+	return nil
+}
+
+func freshPID(pidFile string, since time.Time, rt Runtime) (bool, time.Time) {
+	mod, err := rt.ModTime(pidFile)
+	if err != nil {
+		return false, time.Time{}
+	}
+	return !mod.Before(since), mod
+}
+
+func discover(spec AgentSpec, rootPID string, agentStart time.Time, legacyExisting map[string]bool, rt Runtime) SessionID {
+	if rootPID != "" {
+		pids, _ := rt.Descendants(rootPID)
+		for _, pid := range pids {
+			paths, _ := rt.LsofPaths(pid)
+			for _, path := range paths {
+				if result := spec.Match(path); result.ID != "" || result.NearMiss {
+					return result
+				}
+			}
+		}
+		if !agentStart.IsZero() {
+			return discoverByBirth(spec, agentStart, rt)
+		}
+		return SessionID{}
+	}
+	files, _ := rt.ListFiles(spec.WatchDir)
+	for _, file := range files {
+		if legacyExisting[file] {
+			continue
+		}
+		if result := spec.Match(file); result.ID != "" || result.NearMiss {
+			return result
+		}
+	}
+	return SessionID{}
+}
+
+func discoverByBirth(spec AgentSpec, agentStart time.Time, rt Runtime) SessionID {
+	files, _ := rt.ListFiles(spec.WatchDir)
+	candidates := make([]SessionID, 0, 1)
+	for _, file := range files {
+		birth, err := rt.BirthTime(file)
+		if err != nil || birth.Before(agentStart) {
+			continue
+		}
+		result := spec.Match(file)
+		if result.Matched {
+			candidates = append(candidates, result)
+		}
+	}
+	if len(candidates) == 1 {
+		return candidates[0]
+	}
+	return SessionID{}
+}
diff --git a/cmd/internal/sessionwatch/run_test.go b/cmd/internal/sessionwatch/run_test.go
new file mode 100644
index 0000000..1094e4a
--- /dev/null
+++ b/cmd/internal/sessionwatch/run_test.go
@@ -0,0 +1,250 @@
+package sessionwatch
+
+import (
+	"errors"
+	"path/filepath"
+	"strings"
+	"testing"
+	"time"
+
+	"github.com/xianxu/pair/cmd/internal/adapt"
+)
+
+func TestRunUsesFreshPidfileAndWritesConfig(t *testing.T) {
+	home := "/tmp/home"
+	data := "/tmp/data"
+	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
+	sessionFile := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
+	rt := newFakeRuntime(time.Unix(100, 0))
+	rt.files[filepath.Join(data, "agent-pid-test")] = fakeFile{content: []byte("999999\n"), mod: time.Unix(1, 0)}
+	rt.onSleep = func(time.Duration) {
+		rt.files[filepath.Join(data, "agent-pid-test")] = fakeFile{content: []byte("1234\n"), mod: time.Unix(100, 0)}
+	}
+	rt.alive["1234"] = true
+	rt.descendants["1234"] = []string{"1234", "5678"}
+	rt.lsof["5678"] = []string{sessionFile}
+
+	err := Run(Options{
+		Agent:   "codex",
+		Tag:     "test",
+		Cwd:     "/repo",
+		Args:    []string{"resume", "old", `say "hi"`},
+		Home:    home,
+		DataDir: data,
+		PIDWait: 3 * time.Second,
+		Timeout: 5 * time.Second,
+		Poll:    100 * time.Millisecond,
+	}, rt)
+	if err != nil {
+		t.Fatalf("Run error: %v", err)
+	}
+
+	out := filepath.Join(data, "config-test-codex.json")
+	got := string(rt.writes[out])
+	if !strings.Contains(got, `"session_id":"`+sid+`"`) || strings.Contains(got, "old") || !strings.Contains(got, `say \"hi\"`) {
+		t.Fatalf("config write = %s", got)
+	}
+	if !rt.hasLog(adapt.Fired, "session_id="+sid) {
+		t.Fatalf("logs = %+v, want fired session id", rt.logs)
+	}
+}
+
+func TestRunDiscoversAgySessionFromLsof(t *testing.T) {
+	home := "/tmp/home"
+	data := "/tmp/data"
+	sid := "123e4567-e89b-12d3-a456-426614174000"
+	sessionFile := home + "/.gemini/antigravity-cli/conversations/" + sid + ".db"
+	rt := newFakeRuntime(time.Unix(200, 0))
+	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("2000\n"), mod: time.Unix(200, 0)}
+	rt.alive["2000"] = true
+	rt.descendants["2000"] = []string{"2000"}
+	rt.lsof["2000"] = []string{sessionFile}
+
+	err := Run(Options{
+		Agent:   "agy",
+		Tag:     "tag",
+		Cwd:     "/repo",
+		Args:    []string{"--conversation", "keep"},
+		Home:    home,
+		DataDir: data,
+		PIDWait: time.Second,
+		Timeout: time.Second,
+		Poll:    100 * time.Millisecond,
+	}, rt)
+	if err != nil {
+		t.Fatalf("Run error: %v", err)
+	}
+
+	got := string(rt.writes[filepath.Join(data, "config-tag-agy.json")])
+	if !strings.Contains(got, `"session_id":"`+sid+`"`) || !strings.Contains(got, "--conversation") {
+		t.Fatalf("agy config write = %s", got)
+	}
+}
+
+func TestRunLogsNearMissOnce(t *testing.T) {
+	home := "/tmp/home"
+	data := "/tmp/data"
+	bad := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-not-a-uuid.jsonl"
+	rt := newFakeRuntime(time.Unix(300, 0))
+	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("3000\n"), mod: time.Unix(300, 0)}
+	rt.alive["3000"] = true
+	rt.descendants["3000"] = []string{"3000"}
+	rt.lsof["3000"] = []string{bad}
+
+	err := Run(Options{
+		Agent:   "codex",
+		Tag:     "tag",
+		Cwd:     "/repo",
+		Home:    home,
+		DataDir: data,
+		PIDWait: time.Second,
+		Timeout: 350 * time.Millisecond,
+		Poll:    100 * time.Millisecond,
+	}, rt)
+	if err != nil {
+		t.Fatalf("Run error: %v", err)
+	}
+
+	if got := rt.countLogs(adapt.NearMiss); got != 1 {
+		t.Fatalf("near-miss logs = %d, want 1; logs=%+v", got, rt.logs)
+	}
+	if !rt.hasLog(adapt.Fail, "no session id") {
+		t.Fatalf("logs = %+v, want fail after timeout", rt.logs)
+	}
+}
+
+func TestRunLogsFailOnTimeout(t *testing.T) {
+	rt := newFakeRuntime(time.Unix(400, 0))
+	err := Run(Options{
+		Agent:   "codex",
+		Tag:     "tag",
+		Cwd:     "/repo",
+		Home:    "/tmp/home",
+		DataDir: "/tmp/data",
+		PIDWait: 100 * time.Millisecond,
+		Timeout: 300 * time.Millisecond,
+		Poll:    100 * time.Millisecond,
+	}, rt)
+	if err != nil {
+		t.Fatalf("Run error: %v", err)
+	}
+	if !rt.hasLog(adapt.Fail, "no session id") {
+		t.Fatalf("logs = %+v, want fail", rt.logs)
+	}
+}
+
+type fakeFile struct {
+	content []byte
+	mod     time.Time
+	birth   time.Time
+}
+
+type fakeLog struct {
+	outcome adapt.Outcome
+	detail  string
+}
+
+type fakeRuntime struct {
+	now         time.Time
+	files       map[string]fakeFile
+	alive       map[string]bool
+	descendants map[string][]string
+	lsof        map[string][]string
+	writes      map[string][]byte
+	logs        []fakeLog
+	onSleep     func(time.Duration)
+}
+
+func newFakeRuntime(now time.Time) *fakeRuntime {
+	return &fakeRuntime{
+		now:         now,
+		files:       map[string]fakeFile{},
+		alive:       map[string]bool{},
+		descendants: map[string][]string{},
+		lsof:        map[string][]string{},
+		writes:      map[string][]byte{},
+	}
+}
+
+func (f *fakeRuntime) Now() time.Time { return f.now }
+
+func (f *fakeRuntime) Sleep(d time.Duration) {
+	if f.onSleep != nil {
+		f.onSleep(d)
+		f.onSleep = nil
+	}
+	f.now = f.now.Add(d)
+}
+
+func (f *fakeRuntime) ReadFile(path string) ([]byte, error) {
+	file, ok := f.files[path]
+	if !ok {
+		return nil, errors.New("missing")
+	}
+	return file.content, nil
+}
+
+func (f *fakeRuntime) ModTime(path string) (time.Time, error) {
+	file, ok := f.files[path]
+	if !ok {
+		return time.Time{}, errors.New("missing")
+	}
+	return file.mod, nil
+}
+
+func (f *fakeRuntime) BirthTime(path string) (time.Time, error) {
+	file, ok := f.files[path]
+	if !ok {
+		return time.Time{}, errors.New("missing")
+	}
+	if file.birth.IsZero() {
+		return file.mod, nil
+	}
+	return file.birth, nil
+}
+
+func (f *fakeRuntime) ListFiles(root string) ([]string, error) {
+	var out []string
+	for path := range f.files {
+		if strings.HasPrefix(path, root) {
+			out = append(out, path)
+		}
+	}
+	return out, nil
+}
+
+func (f *fakeRuntime) Descendants(root string) ([]string, error) {
+	if out := f.descendants[root]; len(out) > 0 {
+		return out, nil
+	}
+	return []string{root}, nil
+}
+
+func (f *fakeRuntime) LsofPaths(pid string) ([]string, error) { return f.lsof[pid], nil }
+func (f *fakeRuntime) ProcessAlive(pid string) bool           { return f.alive[pid] }
+func (f *fakeRuntime) AtomicWrite(path string, data []byte) error {
+	f.writes[path] = append([]byte(nil), data...)
+	return nil
+}
+func (f *fakeRuntime) Log(outcome adapt.Outcome, detail string) {
+	f.logs = append(f.logs, fakeLog{outcome: outcome, detail: detail})
+}
+
+func (f *fakeRuntime) hasLog(outcome adapt.Outcome, detail string) bool {
+	for _, log := range f.logs {
+		if log.outcome == outcome && strings.Contains(log.detail, detail) {
+			return true
+		}
+	}
+	return false
+}
+
+func (f *fakeRuntime) countLogs(outcome adapt.Outcome) int {
+	var n int
+	for _, log := range f.logs {
+		if log.outcome == outcome {
+			n++
+		}
+	}
+	return n
+}
diff --git a/cmd/internal/sessionwatch/runtime.go b/cmd/internal/sessionwatch/runtime.go
new file mode 100644
index 0000000..0b808ca
--- /dev/null
+++ b/cmd/internal/sessionwatch/runtime.go
@@ -0,0 +1,148 @@
+package sessionwatch
+
+import (
+	"bytes"
+	"os"
+	"os/exec"
+	"path/filepath"
+	"strconv"
+	"strings"
+	"time"
+
+	"github.com/xianxu/pair/cmd/internal/adapt"
+)
+
+// OSRuntime implements Runtime with real process and filesystem calls.
+type OSRuntime struct {
+	logger *adapt.Logger
+}
+
+func NewOSRuntime(logger *adapt.Logger) OSRuntime {
+	return OSRuntime{logger: logger}
+}
+
+func (OSRuntime) Now() time.Time { return time.Now() }
+func (OSRuntime) Sleep(d time.Duration) {
+	time.Sleep(d)
+}
+
+func (OSRuntime) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }
+
+func (OSRuntime) ModTime(path string) (time.Time, error) {
+	info, err := os.Stat(path)
+	if err != nil {
+		return time.Time{}, err
+	}
+	return info.ModTime(), nil
+}
+
+func (OSRuntime) BirthTime(path string) (time.Time, error) {
+	out, err := exec.Command("stat", "-f", "%B", path).Output()
+	if err != nil {
+		return time.Time{}, err
+	}
+	sec, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
+	if err != nil {
+		return time.Time{}, err
+	}
+	return time.Unix(sec, 0), nil
+}
+
+func (OSRuntime) ListFiles(root string) ([]string, error) {
+	var out []string
+	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
+		if err != nil {
+			return nil
+		}
+		if !d.IsDir() {
+			out = append(out, path)
+		}
+		return nil
+	})
+	return out, err
+}
+
+func (OSRuntime) Descendants(root string) ([]string, error) {
+	out, err := exec.Command("ps", "-axo", "pid=,ppid=").Output()
+	if err != nil {
+		return []string{root}, nil
+	}
+	children := map[string][]string{}
+	for _, line := range strings.Split(string(out), "\n") {
+		fields := strings.Fields(line)
+		if len(fields) != 2 {
+			continue
+		}
+		children[fields[1]] = append(children[fields[1]], fields[0])
+	}
+	queue := []string{root}
+	seen := map[string]bool{root: true}
+	for i := 0; i < len(queue); i++ {
+		for _, child := range children[queue[i]] {
+			if child == "" || seen[child] {
+				continue
+			}
+			seen[child] = true
+			queue = append(queue, child)
+		}
+	}
+	return queue, nil
+}
+
+func (OSRuntime) LsofPaths(pid string) ([]string, error) {
+	out, err := exec.Command("lsof", "-p", pid, "-Fn").Output()
+	if err != nil {
+		return nil, nil
+	}
+	var paths []string
+	for _, line := range strings.Split(string(out), "\n") {
+		if strings.HasPrefix(line, "n") {
+			paths = append(paths, strings.TrimPrefix(line, "n"))
+		}
+	}
+	return paths, nil
+}
+
+func (OSRuntime) ProcessAlive(pid string) bool {
+	return exec.Command("kill", "-0", pid).Run() == nil
+}
+
+func (OSRuntime) AtomicWrite(path string, data []byte) error {
+	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
+		return err
+	}
+	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".")
+	if err != nil {
+		return err
+	}
+	tmpPath := tmp.Name()
+	if _, err := tmp.Write(data); err != nil {
+		_ = tmp.Close()
+		_ = os.Remove(tmpPath)
+		return err
+	}
+	if err := tmp.Close(); err != nil {
+		_ = os.Remove(tmpPath)
+		return err
+	}
+	return os.Rename(tmpPath, path)
+}
+
+func (r OSRuntime) Log(outcome adapt.Outcome, detail string) {
+	r.logger.Log(3, "session-id", outcome, detail)
+}
+
+func ParseDurationSeconds(s string, fallback time.Duration) time.Duration {
+	if s == "" {
+		return fallback
+	}
+	n, err := strconv.Atoi(s)
+	if err != nil || n < 0 {
+		return fallback
+	}
+	return time.Duration(n) * time.Second
+}
+
+func trimCommandOutput(b []byte) string {
+	return string(bytes.TrimSpace(b))
+}
diff --git a/cmd/internal/sessionwatch/sessionwatch.go b/cmd/internal/sessionwatch/sessionwatch.go
new file mode 100644
index 0000000..da76d45
--- /dev/null
+++ b/cmd/internal/sessionwatch/sessionwatch.go
@@ -0,0 +1,125 @@
+package sessionwatch
+
+import (
+	"bytes"
+	"encoding/json"
+	"path/filepath"
+	"regexp"
+	"strings"
+)
+
+var (
+	uuidRE    = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
+	endUUIDRE = regexp.MustCompile(`(?i)([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`)
+)
+
+// AgentSpec describes one async session-file discovery contract.
+type AgentSpec struct {
+	Agent    string
+	Home     string
+	WatchDir string
+}
+
+// SessionID is the outcome of matching a candidate session file path.
+type SessionID struct {
+	Matched  bool
+	NearMiss bool
+	ID       string
+	Path     string
+}
+
+// ConfigPayload is the restart config written as config-<tag>-<agent>.json.
+type ConfigPayload struct {
+	Agent     string   `json:"agent"`
+	Args      []string `json:"args"`
+	SessionID string   `json:"session_id"`
+}
+
+// SpecForAgent returns the async watcher contract for agents that need it.
+func SpecForAgent(agent, home string) (AgentSpec, bool) {
+	switch agent {
+	case "codex":
+		return AgentSpec{
+			Agent:    agent,
+			Home:     home,
+			WatchDir: filepath.Join(home, ".codex", "sessions"),
+		}, true
+	case "agy":
+		return AgentSpec{
+			Agent:    agent,
+			Home:     home,
+			WatchDir: filepath.Join(home, ".gemini", "antigravity-cli", "conversations"),
+		}, true
+	default:
+		return AgentSpec{}, false
+	}
+}
+
+// Match checks whether path belongs to the agent's session-file shape and, if
+// so, extracts the session id or reports a near miss.
+func (s AgentSpec) Match(path string) SessionID {
+	switch s.Agent {
+	case "codex":
+		prefix := filepath.Clean(s.WatchDir) + string(filepath.Separator)
+		clean := filepath.Clean(path)
+		if !strings.HasPrefix(clean, prefix) {
+			return SessionID{}
+		}
+		base := filepath.Base(clean)
+		if !strings.HasPrefix(base, "rollout-") || !strings.HasSuffix(base, ".jsonl") {
+			return SessionID{}
+		}
+		stem := strings.TrimSuffix(base, ".jsonl")
+		if match := endUUIDRE.FindStringSubmatch(stem); len(match) == 2 {
+			return SessionID{Matched: true, ID: match[1], Path: path}
+		}
+		return SessionID{Matched: true, NearMiss: true, Path: path}
+	case "agy":
+		prefix := filepath.Clean(s.WatchDir) + string(filepath.Separator)
+		clean := filepath.Clean(path)
+		if !strings.HasPrefix(clean, prefix) {
+			return SessionID{}
+		}
+		base := filepath.Base(clean)
+		if !strings.HasSuffix(base, ".db") {
+			return SessionID{}
+		}
+		id := strings.TrimSuffix(base, ".db")
+		if uuidRE.MatchString(id) {
+			return SessionID{Matched: true, ID: id, Path: path}
+		}
+		return SessionID{Matched: true, NearMiss: true, Path: path}
+	default:
+		return SessionID{}
+	}
+}
+
+// StripResumeArgs removes resume bindings from args before they are persisted;
+// the session_id field is the canonical store for that binding.
+func StripResumeArgs(agent string, args []string) []string {
+	stripped := make([]string, 0, len(args))
+	i := 0
+	if agent == "codex" && len(args) >= 2 && args[0] == "resume" {
+		i = 2
+	}
+	for i < len(args) {
+		if args[i] == "--resume" {
+			i += 2
+			continue
+		}
+		stripped = append(stripped, args[i])
+		i++
+	}
+	return stripped
+}
+
+// ConfigJSON renders the restart config with structured JSON encoding.
+func ConfigJSON(payload ConfigPayload) ([]byte, error) {
+	var buf bytes.Buffer
+	enc := json.NewEncoder(&buf)
+	enc.SetEscapeHTML(false)
+	if err := enc.Encode(payload); err != nil {
+		return nil, err
+	}
+	return buf.Bytes(), nil
+}
diff --git a/cmd/internal/sessionwatch/sessionwatch_test.go b/cmd/internal/sessionwatch/sessionwatch_test.go
new file mode 100644
index 0000000..e63c152
--- /dev/null
+++ b/cmd/internal/sessionwatch/sessionwatch_test.go
@@ -0,0 +1,112 @@
+package sessionwatch
+
+import (
+	"encoding/json"
+	"strings"
+	"testing"
+)
+
+func TestAgentSpecExtractsCodexSessionID(t *testing.T) {
+	home := "/tmp/home"
+	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
+	path := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
+
+	spec, ok := SpecForAgent("codex", home)
+	if !ok {
+		t.Fatalf("codex spec not found")
+	}
+	got := spec.Match(path)
+	if !got.Matched || got.NearMiss || got.ID != sid || got.Path != path {
+		t.Fatalf("codex match = %+v, want id %q", got, sid)
+	}
+}
+
+func TestAgentSpecExtractsAgySessionID(t *testing.T) {
+	home := "/tmp/home"
+	sid := "123e4567-e89b-12d3-a456-426614174000"
+	path := home + "/.gemini/antigravity-cli/conversations/" + sid + ".db"
+
+	spec, ok := SpecForAgent("agy", home)
+	if !ok {
+		t.Fatalf("agy spec not found")
+	}
+	got := spec.Match(path)
+	if !got.Matched || got.NearMiss || got.ID != sid || got.Path != path {
+		t.Fatalf("agy match = %+v, want id %q", got, sid)
+	}
+}
+
+func TestAgentSpecReportsNearMissForPatternWithBadID(t *testing.T) {
+	home := "/tmp/home"
+	path := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-not-a-uuid.jsonl"
+
+	spec, ok := SpecForAgent("codex", home)
+	if !ok {
+		t.Fatalf("codex spec not found")
+	}
+	got := spec.Match(path)
+	if !got.Matched || !got.NearMiss || got.ID != "" || got.Path != path {
+		t.Fatalf("codex near miss = %+v", got)
+	}
+}
+
+func TestAgentSpecRejectsUnsupportedAgent(t *testing.T) {
+	if _, ok := SpecForAgent("claude", "/tmp/home"); ok {
+		t.Fatalf("claude should not use async session watch")
+	}
+}
+
+func TestStripResumeArgsRemovesCanonicalResumeBindings(t *testing.T) {
+	tests := []struct {
+		name  string
+		agent string
+		args  []string
+		want  []string
+	}{
+		{
+			name:  "codex leading resume",
+			agent: "codex",
+			args:  []string{"resume", "abc", "--no-alt-screen"},
+			want:  []string{"--no-alt-screen"},
+		},
+		{
+			name:  "flag resume",
+			agent: "agy",
+			args:  []string{"--model", "x", "--resume", "abc", "--flag"},
+			want:  []string{"--model", "x", "--flag"},
+		},
+		{
+			name:  "unrelated args keep order",
+			agent: "codex",
+			args:  []string{"--foo", "bar", "resume"},
+			want:  []string{"--foo", "bar", "resume"},
+		},
+	}
+
+	for _, tt := range tests {
+		t.Run(tt.name, func(t *testing.T) {
+			got := StripResumeArgs(tt.agent, tt.args)
+			if strings.Join(got, "\x00") != strings.Join(tt.want, "\x00") {
+				t.Fatalf("StripResumeArgs(%q, %#v) = %#v, want %#v", tt.agent, tt.args, got, tt.want)
+			}
+		})
+	}
+}
+
+func TestConfigJSONUsesStructuredEncoding(t *testing.T) {
+	got, err := ConfigJSON(ConfigPayload{
+		Agent:     "codex",
+		Args:      []string{`say "hi"`, "--flag"},
+		SessionID: "019eff64-6ceb-7e72-9d41-a735a97029ac",
+	})
+	if err != nil {
+		t.Fatalf("ConfigJSON error: %v", err)
+	}
+	var decoded ConfigPayload
+	if err := json.Unmarshal(got, &decoded); err != nil {
+		t.Fatalf("ConfigJSON produced invalid JSON %q: %v", got, err)
+	}
+	if decoded.Agent != "codex" || decoded.SessionID == "" || len(decoded.Args) != 2 || decoded.Args[0] != `say "hi"` {
+		t.Fatalf("decoded payload = %+v", decoded)
+	}
+}
diff --git a/cmd/pair-session-watch/main.go b/cmd/pair-session-watch/main.go
new file mode 100644
index 0000000..608bb02
--- /dev/null
+++ b/cmd/pair-session-watch/main.go
@@ -0,0 +1,50 @@
+package main
+
+import (
+	"fmt"
+	"os"
+	"time"
+
+	"github.com/xianxu/pair/cmd/internal/adapt"
+	"github.com/xianxu/pair/cmd/internal/sessionwatch"
+)
+
+func main() {
+	os.Exit(run(os.Args[1:], os.Getenv, os.Stderr))
+}
+
+func run(args []string, getenv func(string) string, stderr *os.File) int {
+	opts, ok := buildOptions(args, getenv)
+	if !ok {
+		return 0
+	}
+	logger := adapt.Open("session-watch", opts.Agent)
+	defer logger.Close()
+	if err := sessionwatch.Run(opts, sessionwatch.NewOSRuntime(logger)); err != nil {
+		fmt.Fprintf(stderr, "pair-session-watch: %v\n", err)
+		return 1
+	}
+	return 0
+}
+
+func buildOptions(args []string, getenv func(string) string) (sessionwatch.Options, bool) {
+	if len(args) < 3 {
+		return sessionwatch.Options{}, false
+	}
+	home := getenv("HOME")
+	dataDir := getenv("PAIR_DATA_DIR")
+	if dataDir == "" {
+		dataDir = adapt.DataDir()
+	}
+	return sessionwatch.Options{
+		Agent:   args[0],
+		Tag:     args[1],
+		Cwd:     args[2],
+		Args:    append([]string(nil), args[3:]...),
+		Home:    home,
+		DataDir: dataDir,
+		PIDWait: sessionwatch.ParseDurationSeconds(getenv("PAIR_SESSION_WATCH_PID_WAIT_SECONDS"), 2*time.Second),
+		Timeout: 60 * time.Second,
+		Poll:    100 * time.Millisecond,
+	}, true
+}
diff --git a/cmd/pair-session-watch/main_test.go b/cmd/pair-session-watch/main_test.go
new file mode 100644
index 0000000..05ca5db
--- /dev/null
+++ b/cmd/pair-session-watch/main_test.go
@@ -0,0 +1,39 @@
+package main
+
+import (
+	"reflect"
+	"testing"
+	"time"
+)
+
+func TestBuildOptionsFromArgsAndEnv(t *testing.T) {
+	env := map[string]string{
+		"HOME":                                "/home/me",
+		"PAIR_DATA_DIR":                       "/tmp/pair-data",
+		"PAIR_SESSION_WATCH_PID_WAIT_SECONDS": "3",
+	}
+	opts, ok := buildOptions([]string{"codex", "tag", "/repo", "resume", "old", "--flag"}, func(k string) string {
+		return env[k]
+	})
+	if !ok {
+		t.Fatalf("buildOptions returned !ok")
+	}
+	if opts.Agent != "codex" || opts.Tag != "tag" || opts.Cwd != "/repo" {
+		t.Fatalf("opts identity = %+v", opts)
+	}
+	if opts.Home != "/home/me" || opts.DataDir != "/tmp/pair-data" {
+		t.Fatalf("opts paths = %+v", opts)
+	}
+	if opts.PIDWait != 3*time.Second || opts.Timeout != 60*time.Second || opts.Poll != 100*time.Millisecond {
+		t.Fatalf("opts durations = %+v", opts)
+	}
+	if !reflect.DeepEqual(opts.Args, []string{"resume", "old", "--flag"}) {
+		t.Fatalf("opts args = %#v", opts.Args)
+	}
+}
+
+func TestBuildOptionsRejectsMissingRequiredArgs(t *testing.T) {
+	if _, ok := buildOptions([]string{"codex", "tag"}, func(string) string { return "" }); ok {
+		t.Fatalf("buildOptions should reject missing cwd")
+	}
+}
diff --git a/tests/pair-session-watch-test.sh b/tests/pair-session-watch-test.sh
index 8112344..c50cd17 100644
--- a/tests/pair-session-watch-test.sh
+++ b/tests/pair-session-watch-test.sh
@@ -33,7 +33,7 @@ HOME="$RT/home" \
 PAIR_DATA_DIR="$RT/data" \
 PAIR_TAG=test \
 PAIR_SESSION_WATCH_PID_WAIT_SECONDS=3 \
-"$ROOT/bin/pair-session-watch.sh" codex test "$ROOT" --no-alt-screen &
+"$ROOT/bin/pair-session-watch.sh" codex test "$ROOT" resume old-session 'say "hi"' --no-alt-screen &
 watch_pid=$!
 
 sleep 0.2
@@ -47,4 +47,10 @@ got="$(jq -r '.session_id // empty' "$RT/data/config-test-codex.json")"
   exit 1
 }
 
+args="$(jq -c '.args' "$RT/data/config-test-codex.json")"
+[ "$args" = '["say \"hi\"","--no-alt-screen"]' ] || {
+  echo "args mismatch: got '$args'" >&2
+  exit 1
+}
+
 echo "pair-session-watch stale pidfile test PASS"
diff --git a/workshop/plans/000078-go-stateful-shell-glue-plan.md b/workshop/plans/000078-go-stateful-shell-glue-plan.md
index 54dde9d..a4dcacd 100644
--- a/workshop/plans/000078-go-stateful-shell-glue-plan.md
+++ b/workshop/plans/000078-go-stateful-shell-glue-plan.md
@@ -6,7 +6,7 @@
 
 **Architecture:** Keep the stateful watcher behavior split into pure session-watch decisions and a thin process/filesystem shell. `cmd/internal/sessionwatch` will own agent support, PID-tree/session-file matching, id extraction, resume-arg stripping, and config payload construction; `cmd/pair-session-watch` will own real time, process commands, atomic writes, and adapt-log emission. `bin/pair-session-watch.sh` remains the stable caller surface and execs the built Go binary for this migration window.
 
-**Tech Stack:** Go standard library, existing `cmd/internal/adapt` JSONL emitter shape where practical, shell compatibility shim, process-level shell tests with fake `ps`/`lsof`/filesystem state.
+**Tech Stack:** Go standard library, existing `cmd/internal/adapt.Open` / `adapt.Logger` for flight-recorder events, shell compatibility shim, process-level shell tests with fake `ps`/`lsof`/filesystem state.
 
 ---
 
@@ -53,6 +53,7 @@
 | `PairSessionWatchCommand` | `cmd/pair-session-watch/main.go` | new | CLI, environment, process loop |
 | `PairSessionWatchShim` | `bin/pair-session-watch.sh` | modified | legacy shell command name |
 | `SessionWatchProcessTest` | `tests/pair-session-watch-test.sh` | modified | fake PATH commands and temp HOME/data dirs |
+| `AdaptLogger` | `cmd/internal/adapt/adapt.go` | reused | adaptation flight-recorder JSONL schema |
 
 - **WatcherRuntime** — Boundary used by the command loop for process and filesystem side effects.
   - **Injected into:** `sessionwatch.Run` or equivalent orchestration function; pure helpers stay independent.
@@ -70,6 +71,10 @@
   - **Injected into:** `make test-session-watch` and the repo-wide `make test` target.
   - **Future extensions:** Add agy fixtures as its native session format evolves.
 
+- **AdaptLogger** — Existing Go logger for the shared adaptation flight-recorder schema.
+  - **Injected into:** `cmd/internal/sessionwatch` runtime setup, not pure helper functions.
+  - **Future extensions:** Keeps shell/Lua/Go emitters aligned as the remaining shell emitters move. `ARCH-DRY`: do not hand-maintain a second Go copy of the adapt JSON schema.
+
 ## Task 1: Pure Session Watch Decisions
 
 **Files:**
@@ -161,7 +166,12 @@ Expected: FAIL until diagnostics/fallbacks are complete.
 
 - [ ] **Step 5: Finish orchestration and CLI command**
 
-Create `cmd/pair-session-watch/main.go` as a thin CLI over the runtime. Add `pair-session-watch` to `GO_BINS` and `make build` dependencies in `Makefile.local`.
+Create `cmd/pair-session-watch/main.go` as a thin CLI over the runtime. Update `Makefile.local` explicitly:
+- add `pair-session-watch` to `.PHONY`;
+- add `pair-session-watch` to `GO_BINS`;
+- add a per-binary `pair-session-watch: $(BIN_DIR)/pair-session-watch` alias;
+- add a `$(BIN_DIR)/pair-session-watch` build rule;
+- make `test-session-watch` depend on `$(BIN_DIR)/pair-session-watch` so repo-wide `make test` cannot run the shim process test before the Go binary exists.
 
 - [ ] **Step 6: Verify command package tests pass**
 
@@ -173,7 +183,7 @@ Expected: PASS.
 **Files:**
 - Modify: `bin/pair-session-watch.sh`
 - Modify: `tests/pair-session-watch-test.sh`
-- Modify: `Makefile.local` if dependency wiring needs adjustment
+- Modify: `Makefile.local`
 
 - [ ] **Step 1: Replace shell implementation with a compatibility shim**
 
@@ -215,6 +225,8 @@ Record that `pair-session-watch` is now Go-owned with a shell shim, while `pair-
 
 Check off candidate selection and implementation items that are complete. Log the explicit split: `pair-title.sh` remains a follow-up because it owns UI title state rather than restart config discovery.
 
+Also log that short shell scripts and opener scripts remain intentionally shell-owned in this slice because #78's payoff target is stateful session discovery. This directly satisfies the Done-when item about leaving no-payoff shell glue alone.
+
 - [ ] **Step 3: Run final verification**
 
 Run:


exec
/bin/zsh -lc "sed -n '1,220p' /Users/xianxu/workspace/ariadne/construct/adapted/superpowers-using-superpowers/SKILL.md && sdlc --help && sed -n '1,220p' workshop/lessons.md && sed -n '1,260p' workshop/issues/000078-go-stateful-shell-glue.md && sed -n '1,280p' workshop/plans/000078-go-stateful-shell-glue-plan.md" in /Users/xianxu/workspace/pair
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
id: 000078
status: done
deps: [000077]
github_issue:
created: 2026-06-26
updated: 2026-06-30
estimate_hours: 3.12
started: 2026-06-30T15:58:17-07:00
actual_hours: 0.46
---

# pair Go stateful shell glue

## Problem

After the public entrypoint is Go-owned, remaining stateful shell scripts can keep packaging brittle and hide reliability bugs. The biggest candidates are long-running or session-observing scripts, not short native glue.

## Spec

Port stateful shell glue where the packaging or reliability payoff is clear. Candidates include:

- `pair-title.sh` — long-running poller, pane title/context meter, cmux title ownership.
- `pair-session-watch.sh` — session-id discovery, PID tree/lsof behavior, atomic config write.
- review readiness/target helpers if #73 finds packaging benefit.
- opener scripts only when the Go entrypoint can replace their orchestration cleanly.

This issue may be split further if #73 shows the candidates are too large. Keep native assets native. Do not port Lua or zellij KDL into Go.

Selected slice: port `pair-session-watch.sh` first. It owns session-id discovery, PID tree/lsof behavior, atomic restart-config writes, and adapt-log drift signals; those are high-value correctness and packaging surfaces with clear process-level fake coverage. `pair-title.sh` remains shell-owned for this issue and should become a follow-up because it owns a separate UI/title-poller surface: zellij frame titles, cmux workspace ownership, activity buckets, singleton poller identity, and session liveness. `ARCH-PURPOSE`: #78 is satisfied by porting a prioritized stateful subset and explicitly splitting the other stateful candidate instead of blending two long-running scripts into one review boundary.

## Done when

- [x] A prioritized subset of stateful shell glue is ported or split into smaller issues.
- [x] Ported behavior has process-level tests with fake external commands/files.
- [x] Legacy script callers either route to Go or remain as compatibility shims.
- [x] Short shell scripts with no packaging/reliability payoff are explicitly left alone.
- [x] Pair remains usable after each merge.

## Plan

- [x] Choose the candidate from #73's priority table.
- [x] Capture existing behavior in tests before porting.
- [x] Port pure decision logic and thin IO seams.
- [x] Keep compatibility shims until all callers move.
- [x] Verify live or fake end-to-end behavior.

Detailed implementation plan: `workshop/plans/000078-go-stateful-shell-glue-plan.md`.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.20 impl=0.08
item: greenfield-go-module design=0.45 impl=0.55
item: skill-or-dispatcher design=0.35 impl=0.45
item: smaller-go-module design=0.15 impl=0.12
item: atlas-docs design=0.20 impl=0.10
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.20
total: 3.12
```

## Log

### 2026-06-26

Created from #72. This is intentionally later in the sequence; porting shell before the entrypoint shape is clear risks wasted work.

### 2026-06-30
- 2026-06-30: closed — go test ./cmd/internal/sessionwatch ./cmd/pair-session-watch -count=1; go test ./... -count=1; make pair-session-watch; make test-session-watch; bin/pair --help; bin/pair-dev --help; bin/pair-session-watch claude test /tmp

Claimed after #77 landed. Selected `pair-session-watch.sh` as the #78 slice from the #73 migration inventory because it owns restart-config correctness and brittle PID/lsof/session-file discovery. Split `pair-title.sh` out of this issue: it remains stateful shell glue, but its UI title-poller ownership is a separate risk surface. `ARCH-DRY`: the plan centralizes agent watch patterns, id extraction, resume-arg stripping, and config JSON in Go helpers instead of scattering them across shell conditionals. `ARCH-PURE`: pure parsing and config helpers are tested without process IO; process discovery stays behind a fakeable runtime.

Plan-quality returned INFO. Tightened the durable plan to reuse `cmd/internal/adapt.Open` / `adapt.Logger` directly, spell out `Makefile.local` wiring for `pair-session-watch`, and explicitly log that short shell scripts/opener scripts remain out of scope because this slice targets stateful session discovery.

Implemented `cmd/internal/sessionwatch` and `cmd/pair-session-watch`. The pure helpers own agent watch specs, codex/agy session-id extraction, near-miss classification, resume-arg stripping, and structured config JSON. The runtime owns PID freshness, process-tree/lsof discovery, birth-time and legacy fallbacks, atomic config writes, and adapt-log `fired` / `near-miss` / `fail` events via `cmd/internal/adapt`.

Replaced `bin/pair-session-watch.sh` with a compatibility shim that resolves `PAIR_HOME` and execs `bin/pair-session-watch`, leaving the existing `bin/pair` create-path caller unchanged. Updated `Makefile.local` so `pair-session-watch` is a built Go binary and `make test-session-watch` depends on it. Expanded `tests/pair-session-watch-test.sh` to exercise the shim, fake `lsof`, stale pidfile wait, codex resume stripping, and quote-safe JSON args.

Updated atlas migration docs to show `pair-session-watch` as Go-owned with a shell shim. `pair-title.sh` remains the next stateful shell candidate because it owns UI title-poller state. Short shell scripts and opener scripts remain intentionally shell-owned in this slice because their packaging/reliability payoff is lower than the stateful session-discovery watcher.

Verification passed:

- `go test ./cmd/internal/sessionwatch ./cmd/pair-session-watch -count=1`
- `go test ./... -count=1`
- `make pair-session-watch`
- `make test-session-watch`
- `bin/pair --help`
- `bin/pair-dev --help`
- `bin/pair-session-watch claude test /tmp`
# Go Stateful Shell Glue Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port `bin/pair-session-watch.sh` to a Go-owned command while keeping the legacy script name as a compatibility shim.

**Architecture:** Keep the stateful watcher behavior split into pure session-watch decisions and a thin process/filesystem shell. `cmd/internal/sessionwatch` will own agent support, PID-tree/session-file matching, id extraction, resume-arg stripping, and config payload construction; `cmd/pair-session-watch` will own real time, process commands, atomic writes, and adapt-log emission. `bin/pair-session-watch.sh` remains the stable caller surface and execs the built Go binary for this migration window.

**Tech Stack:** Go standard library, existing `cmd/internal/adapt.Open` / `adapt.Logger` for flight-recorder events, shell compatibility shim, process-level shell tests with fake `ps`/`lsof`/filesystem state.

---

## Scope

#78 ports only `pair-session-watch.sh`. `pair-title.sh` is explicitly deferred because it has a separate long-running UI ownership surface: zellij frame titles, cmux workspace title ownership, activity buckets, singleton poller identity, and session liveness. Keeping these as separate issues reduces review risk and keeps each migrated script meaningfully testable. `ARCH-PURPOSE`: this still satisfies #78 by porting a prioritized stateful shell-glue subset and splitting the rest.

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `AgentSpec` | `cmd/internal/sessionwatch/sessionwatch.go` | new |
| `SessionID` | `cmd/internal/sessionwatch/sessionwatch.go` | new |
| `ResumeArgs` | `cmd/internal/sessionwatch/sessionwatch.go` | new |
| `ConfigPayload` | `cmd/internal/sessionwatch/sessionwatch.go` | new |

- **AgentSpec** — Per-agent watch metadata for `codex` and `agy`: watch directory suffix, filename pattern, and id extractor.
  - **Relationships:** 1:1 with supported agent names; owned by the watcher planner.
  - **DRY rationale:** The current shell duplicates agent conditionals across directory selection, path matching, and extraction. Centralizing the agent contract avoids drift as more harnesses are added.
  - **Future extensions:** Claude is still synchronous in `bin/pair`; if a future agent needs async discovery, add one spec row and tests.

- **SessionID** — Extracted identifier plus matched file path and near-miss state.
  - **Relationships:** 1:1 with a matched session file candidate.
  - **DRY rationale:** `lsof`, birth-time fallback, and legacy fallback all need the same extract-or-near-miss behavior.
  - **Future extensions:** Add richer confidence reasons if diagnostics need to distinguish lsof vs fallback discovery.

- **ResumeArgs** — Agent args with resume bindings removed before persistence.
  - **Relationships:** N:1 from raw agent args to saved config args.
  - **DRY rationale:** Mirrors `bin/pair` stripping semantics in one Go function and tests edge cases that are awkward in shell.
  - **Future extensions:** If `bin/pair` becomes Go-owned, the same stripping function should become the single source for launcher and watcher.

- **ConfigPayload** — `{agent,args,session_id}` JSON payload for `config-<tag>-<agent>.json`.
  - **Relationships:** 1:1 with a discovered `(tag, agent)` restart config.
  - **DRY rationale:** Moves JSON construction out of shell and into Go's structured encoder, matching the `workshop/lessons.md` rule against printf JSON.
  - **Future extensions:** Add schema fields here if restart config widens.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `WatcherRuntime` | `cmd/internal/sessionwatch/runtime.go` | new | process list, lsof, find/stat, sleep/clock, filesystem writes |
| `PairSessionWatchCommand` | `cmd/pair-session-watch/main.go` | new | CLI, environment, process loop |
| `PairSessionWatchShim` | `bin/pair-session-watch.sh` | modified | legacy shell command name |
| `SessionWatchProcessTest` | `tests/pair-session-watch-test.sh` | modified | fake PATH commands and temp HOME/data dirs |
| `AdaptLogger` | `cmd/internal/adapt/adapt.go` | reused | adaptation flight-recorder JSONL schema |

- **WatcherRuntime** — Boundary used by the command loop for process and filesystem side effects.
  - **Injected into:** `sessionwatch.Run` or equivalent orchestration function; pure helpers stay independent.
  - **Future extensions:** Lets tests drive timeout/failure cases without real 60s sleeps.

- **PairSessionWatchCommand** — Parses `pair-session-watch <agent> <tag> <cwd> [agent-args...]`, no-ops unsupported agents, and runs the watcher.
  - **Injected into:** Called by the shell shim and later directly by `bin/pair` or a Go entrypoint.
  - **Future extensions:** Can become `pair-go session-watch` or internal launch subcommand when `bin/pair` is retired.

- **PairSessionWatchShim** — Keeps existing callers stable by execing `$PAIR_HOME/bin/pair-session-watch`.
  - **Injected into:** Existing `bin/pair` create path remains unchanged.
  - **Future extensions:** Delete once callers move to the Go command directly.

- **SessionWatchProcessTest** — Process-level regression coverage with fake commands and temp files.
  - **Injected into:** `make test-session-watch` and the repo-wide `make test` target.
  - **Future extensions:** Add agy fixtures as its native session format evolves.

- **AdaptLogger** — Existing Go logger for the shared adaptation flight-recorder schema.
  - **Injected into:** `cmd/internal/sessionwatch` runtime setup, not pure helper functions.
  - **Future extensions:** Keeps shell/Lua/Go emitters aligned as the remaining shell emitters move. `ARCH-DRY`: do not hand-maintain a second Go copy of the adapt JSON schema.

## Task 1: Pure Session Watch Decisions

**Files:**
- Create: `cmd/internal/sessionwatch/sessionwatch.go`
- Create: `cmd/internal/sessionwatch/sessionwatch_test.go`

- [ ] **Step 1: Write failing tests for supported agent specs and id extraction**

Tests:
- `codex` accepts paths under `~/.codex/sessions/.../rollout-*-<uuid>.jsonl`.
- `agy` accepts paths under `~/.gemini/antigravity-cli/conversations/<uuid>.db`.
- filenames matching the watch pattern but not the id grammar return a near-miss.
- unsupported agents are no-op.

Run: `go test ./cmd/internal/sessionwatch -run 'TestAgentSpec|TestExtract' -count=1`
Expected: FAIL because the package does not exist.

- [ ] **Step 2: Implement minimal `AgentSpec` and `ExtractSessionID`**

Use only deterministic string/path logic. Do not shell out or read files in this package function.

- [ ] **Step 3: Verify pure extraction tests pass**

Run: `go test ./cmd/internal/sessionwatch -run 'TestAgentSpec|TestExtract' -count=1`
Expected: PASS.

- [ ] **Step 4: Write failing tests for resume-arg stripping and config JSON**

Tests:
- codex leading `resume <id>` is removed.
- any `--resume <id>` pair is removed.
- unrelated args keep order.
- JSON payload escapes quotes and preserves arrays through `encoding/json`.

Run: `go test ./cmd/internal/sessionwatch -run 'TestStrip|TestConfig' -count=1`
Expected: FAIL until helpers exist.

- [ ] **Step 5: Implement `StripResumeArgs` and `ConfigJSON`**

Keep behavior byte-compatible in structure with existing shell output: object with `agent`, `args`, and `session_id`.

- [ ] **Step 6: Verify all pure tests pass**

Run: `go test ./cmd/internal/sessionwatch -count=1`
Expected: PASS.

## Task 2: Go Watcher Command

**Files:**
- Create: `cmd/internal/sessionwatch/runtime.go`
- Create: `cmd/internal/sessionwatch/run.go`
- Create: `cmd/internal/sessionwatch/run_test.go`
- Create: `cmd/pair-session-watch/main.go`
- Modify: `Makefile.local`

- [ ] **Step 1: Write failing runtime tests for stale pidfile replacement**

Use a fake runtime:
- initial pidfile mtime predates watcher start and points at a dead/unrelated PID.
- fresh pidfile appears during the wait window.
- lsof on the fresh PID returns a codex rollout file.
- config is written atomically with the discovered id.

Run: `go test ./cmd/internal/sessionwatch -run TestRunUsesFreshPidfile -count=1`
Expected: FAIL because orchestration does not exist.

- [ ] **Step 2: Implement watcher orchestration with injected runtime**

Keep the loop behavior faithful:
- return immediately for unsupported agents.
- wait briefly for a fresh `agent-pid-<tag>` file.
- if bound to a root PID, inspect root plus descendants with `lsof -Fn`.
- if lsof misses, use birth-time fallback for files born at or after pidfile mtime, accepting exactly one candidate.
- if no root PID, use legacy snapshot-diff fallback.
- write config via temp file plus rename.
- emit adapt-log `near-miss`, `fired`, and `fail` outcomes.

- [ ] **Step 3: Verify runtime stale-pidfile test passes**

Run: `go test ./cmd/internal/sessionwatch -run TestRunUsesFreshPidfile -count=1`
Expected: PASS.

- [ ] **Step 4: Add failing tests for near-miss, fail, and agy discovery**

Use fake runtime with a controllable clock so the fail case does not sleep 60s.

Run: `go test ./cmd/internal/sessionwatch -run 'TestRunLogs|TestRunAgy' -count=1`
Expected: FAIL until diagnostics/fallbacks are complete.

- [ ] **Step 5: Finish orchestration and CLI command**

Create `cmd/pair-session-watch/main.go` as a thin CLI over the runtime. Update `Makefile.local` explicitly:
- add `pair-session-watch` to `.PHONY`;
- add `pair-session-watch` to `GO_BINS`;
- add a per-binary `pair-session-watch: $(BIN_DIR)/pair-session-watch` alias;
- add a `$(BIN_DIR)/pair-session-watch` build rule;
- make `test-session-watch` depend on `$(BIN_DIR)/pair-session-watch` so repo-wide `make test` cannot run the shim process test before the Go binary exists.

- [ ] **Step 6: Verify command package tests pass**

Run: `go test ./cmd/internal/sessionwatch ./cmd/pair-session-watch -count=1`
Expected: PASS.

## Task 3: Compatibility Shim And Process Tests

**Files:**
- Modify: `bin/pair-session-watch.sh`
- Modify: `tests/pair-session-watch-test.sh`
- Modify: `Makefile.local`

- [ ] **Step 1: Replace shell implementation with a compatibility shim**

The shim should:
- resolve its real path like other Pair scripts;
- set `PAIR_HOME`;
- exec `$PAIR_HOME/bin/pair-session-watch "$@"`;
- print a clear diagnostic if the Go binary is missing.

- [ ] **Step 2: Expand process-level test coverage**

Update `tests/pair-session-watch-test.sh` to exercise the shim invoking the Go binary with fake `ps`/`lsof` and temp HOME/data dirs. Keep the stale pidfile regression. Add a quoted arg in the saved config to prove JSON escaping is structured.

Run: `make pair-session-watch && make test-session-watch`
Expected: PASS.

- [ ] **Step 3: Verify direct command and shim both work**

Run:
- `bin/pair-session-watch --help` or unsupported-agent smoke if no help is exposed.
- `go test ./... -count=1`
- `make test-session-watch`

Expected: all PASS.

## Task 4: Docs, Issue Split, And Verification

**Files:**
- Modify: `workshop/issues/000078-go-stateful-shell-glue.md`
- Modify: `atlas/go-migration-inventory.md`
- Modify: `atlas/architecture.md`
- Optionally create: follow-up issue for `pair-title.sh` if #78 does not already leave enough trace.

- [ ] **Step 1: Update atlas**

Record that `pair-session-watch` is now Go-owned with a shell shim, while `pair-title.sh` remains stateful shell glue.

- [ ] **Step 2: Update #78 issue**

Check off candidate selection and implementation items that are complete. Log the explicit split: `pair-title.sh` remains a follow-up because it owns UI title state rather than restart config discovery.

Also log that short shell scripts and opener scripts remain intentionally shell-owned in this slice because #78's payoff target is stateful session discovery. This directly satisfies the Done-when item about leaving no-payoff shell glue alone.

- [ ] **Step 3: Run final verification**

Run:
- `go test ./cmd/internal/sessionwatch ./cmd/pair-session-watch -count=1`
- `go test ./... -count=1`
- `make pair-session-watch`
- `make test-session-watch`
- `bin/pair --help`
- `bin/pair-dev --help`

Expected: all PASS.

- [ ] **Step 4: Close through SDLC**

Run `sdlc actual --issue 78`, then `sdlc close --issue 78 --verified '<commands>'` with the verification evidence. Let the boundary review decide whether the title-poller split is sufficiently documented.

codex
Using `superpowers-using-superpowers` for the required session workflow check, then I’m doing the boundary review read-only against the issue, plan, and actual files.
exec
/bin/zsh -lc "nl -ba cmd/internal/sessionwatch/sessionwatch.go | sed -n '1,220p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package sessionwatch
     2	
     3	import (
     4		"bytes"
     5		"encoding/json"
     6		"path/filepath"
     7		"regexp"
     8		"strings"
     9	)
    10	
    11	var (
    12		uuidRE    = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
    13		endUUIDRE = regexp.MustCompile(`(?i)([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`)
    14	)
    15	
    16	// AgentSpec describes one async session-file discovery contract.
    17	type AgentSpec struct {
    18		Agent    string
    19		Home     string
    20		WatchDir string
    21	}
    22	
    23	// SessionID is the outcome of matching a candidate session file path.
    24	type SessionID struct {
    25		Matched  bool
    26		NearMiss bool
    27		ID       string
    28		Path     string
    29	}
    30	
    31	// ConfigPayload is the restart config written as config-<tag>-<agent>.json.
    32	type ConfigPayload struct {
    33		Agent     string   `json:"agent"`
    34		Args      []string `json:"args"`
    35		SessionID string   `json:"session_id"`
    36	}
    37	
    38	// SpecForAgent returns the async watcher contract for agents that need it.
    39	func SpecForAgent(agent, home string) (AgentSpec, bool) {
    40		switch agent {
    41		case "codex":
    42			return AgentSpec{
    43				Agent:    agent,
    44				Home:     home,
    45				WatchDir: filepath.Join(home, ".codex", "sessions"),
    46			}, true
    47		case "agy":
    48			return AgentSpec{
    49				Agent:    agent,
    50				Home:     home,
    51				WatchDir: filepath.Join(home, ".gemini", "antigravity-cli", "conversations"),
    52			}, true
    53		default:
    54			return AgentSpec{}, false
    55		}
    56	}
    57	
    58	// Match checks whether path belongs to the agent's session-file shape and, if
    59	// so, extracts the session id or reports a near miss.
    60	func (s AgentSpec) Match(path string) SessionID {
    61		switch s.Agent {
    62		case "codex":
    63			prefix := filepath.Clean(s.WatchDir) + string(filepath.Separator)
    64			clean := filepath.Clean(path)
    65			if !strings.HasPrefix(clean, prefix) {
    66				return SessionID{}
    67			}
    68			base := filepath.Base(clean)
    69			if !strings.HasPrefix(base, "rollout-") || !strings.HasSuffix(base, ".jsonl") {
    70				return SessionID{}
    71			}
    72			stem := strings.TrimSuffix(base, ".jsonl")
    73			if match := endUUIDRE.FindStringSubmatch(stem); len(match) == 2 {
    74				return SessionID{Matched: true, ID: match[1], Path: path}
    75			}
    76			return SessionID{Matched: true, NearMiss: true, Path: path}
    77		case "agy":
    78			prefix := filepath.Clean(s.WatchDir) + string(filepath.Separator)
    79			clean := filepath.Clean(path)
    80			if !strings.HasPrefix(clean, prefix) {
    81				return SessionID{}
    82			}
    83			base := filepath.Base(clean)
    84			if !strings.HasSuffix(base, ".db") {
    85				return SessionID{}
    86			}
    87			id := strings.TrimSuffix(base, ".db")
    88			if uuidRE.MatchString(id) {
    89				return SessionID{Matched: true, ID: id, Path: path}
    90			}
    91			return SessionID{Matched: true, NearMiss: true, Path: path}
    92		default:
    93			return SessionID{}
    94		}
    95	}
    96	
    97	// StripResumeArgs removes resume bindings from args before they are persisted;
    98	// the session_id field is the canonical store for that binding.
    99	func StripResumeArgs(agent string, args []string) []string {
   100		stripped := make([]string, 0, len(args))
   101		i := 0
   102		if agent == "codex" && len(args) >= 2 && args[0] == "resume" {
   103			i = 2
   104		}
   105		for i < len(args) {
   106			if args[i] == "--resume" {
   107				i += 2
   108				continue
   109			}
   110			stripped = append(stripped, args[i])
   111			i++
   112		}
   113		return stripped
   114	}
   115	
   116	// ConfigJSON renders the restart config with structured JSON encoding.
   117	func ConfigJSON(payload ConfigPayload) ([]byte, error) {
   118		var buf bytes.Buffer
   119		enc := json.NewEncoder(&buf)
   120		enc.SetEscapeHTML(false)
   121		if err := enc.Encode(payload); err != nil {
   122			return nil, err
   123		}
   124		return buf.Bytes(), nil
   125	}

exec
/bin/zsh -lc "nl -ba cmd/internal/sessionwatch/runtime.go | sed -n '1,220p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package sessionwatch
     2	
     3	import (
     4		"bytes"
     5		"os"
     6		"os/exec"
     7		"path/filepath"
     8		"strconv"
     9		"strings"
    10		"time"
    11	
    12		"github.com/xianxu/pair/cmd/internal/adapt"
    13	)
    14	
    15	// OSRuntime implements Runtime with real process and filesystem calls.
    16	type OSRuntime struct {
    17		logger *adapt.Logger
    18	}
    19	
    20	func NewOSRuntime(logger *adapt.Logger) OSRuntime {
    21		return OSRuntime{logger: logger}
    22	}
    23	
    24	func (OSRuntime) Now() time.Time { return time.Now() }
    25	func (OSRuntime) Sleep(d time.Duration) {
    26		time.Sleep(d)
    27	}
    28	
    29	func (OSRuntime) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }
    30	
    31	func (OSRuntime) ModTime(path string) (time.Time, error) {
    32		info, err := os.Stat(path)
    33		if err != nil {
    34			return time.Time{}, err
    35		}
    36		return info.ModTime(), nil
    37	}
    38	
    39	func (OSRuntime) BirthTime(path string) (time.Time, error) {
    40		out, err := exec.Command("stat", "-f", "%B", path).Output()
    41		if err != nil {
    42			return time.Time{}, err
    43		}
    44		sec, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
    45		if err != nil {
    46			return time.Time{}, err
    47		}
    48		return time.Unix(sec, 0), nil
    49	}
    50	
    51	func (OSRuntime) ListFiles(root string) ([]string, error) {
    52		var out []string
    53		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
    54			if err != nil {
    55				return nil
    56			}
    57			if !d.IsDir() {
    58				out = append(out, path)
    59			}
    60			return nil
    61		})
    62		return out, err
    63	}
    64	
    65	func (OSRuntime) Descendants(root string) ([]string, error) {
    66		out, err := exec.Command("ps", "-axo", "pid=,ppid=").Output()
    67		if err != nil {
    68			return []string{root}, nil
    69		}
    70		children := map[string][]string{}
    71		for _, line := range strings.Split(string(out), "\n") {
    72			fields := strings.Fields(line)
    73			if len(fields) != 2 {
    74				continue
    75			}
    76			children[fields[1]] = append(children[fields[1]], fields[0])
    77		}
    78		queue := []string{root}
    79		seen := map[string]bool{root: true}
    80		for i := 0; i < len(queue); i++ {
    81			for _, child := range children[queue[i]] {
    82				if child == "" || seen[child] {
    83					continue
    84				}
    85				seen[child] = true
    86				queue = append(queue, child)
    87			}
    88		}
    89		return queue, nil
    90	}
    91	
    92	func (OSRuntime) LsofPaths(pid string) ([]string, error) {
    93		out, err := exec.Command("lsof", "-p", pid, "-Fn").Output()
    94		if err != nil {
    95			return nil, nil
    96		}
    97		var paths []string
    98		for _, line := range strings.Split(string(out), "\n") {
    99			if strings.HasPrefix(line, "n") {
   100				paths = append(paths, strings.TrimPrefix(line, "n"))
   101			}
   102		}
   103		return paths, nil
   104	}
   105	
   106	func (OSRuntime) ProcessAlive(pid string) bool {
   107		return exec.Command("kill", "-0", pid).Run() == nil
   108	}
   109	
   110	func (OSRuntime) AtomicWrite(path string, data []byte) error {
   111		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
   112			return err
   113		}
   114		tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".")
   115		if err != nil {
   116			return err
   117		}
   118		tmpPath := tmp.Name()
   119		if _, err := tmp.Write(data); err != nil {
   120			_ = tmp.Close()
   121			_ = os.Remove(tmpPath)
   122			return err
   123		}
   124		if err := tmp.Close(); err != nil {
   125			_ = os.Remove(tmpPath)
   126			return err
   127		}
   128		return os.Rename(tmpPath, path)
   129	}
   130	
   131	func (r OSRuntime) Log(outcome adapt.Outcome, detail string) {
   132		r.logger.Log(3, "session-id", outcome, detail)
   133	}
   134	
   135	func ParseDurationSeconds(s string, fallback time.Duration) time.Duration {
   136		if s == "" {
   137			return fallback
   138		}
   139		n, err := strconv.Atoi(s)
   140		if err != nil || n < 0 {
   141			return fallback
   142		}
   143		return time.Duration(n) * time.Second
   144	}
   145	
   146	func trimCommandOutput(b []byte) string {
   147		return string(bytes.TrimSpace(b))
   148	}

exec
/bin/zsh -lc "nl -ba cmd/internal/sessionwatch/run.go | sed -n '1,240p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package sessionwatch
     2	
     3	import (
     4		"path/filepath"
     5		"strings"
     6		"time"
     7	
     8		"github.com/xianxu/pair/cmd/internal/adapt"
     9	)
    10	
    11	// Options are the watcher inputs after CLI/env resolution.
    12	type Options struct {
    13		Agent   string
    14		Tag     string
    15		Cwd     string
    16		Args    []string
    17		Home    string
    18		DataDir string
    19		PIDWait time.Duration
    20		Timeout time.Duration
    21		Poll    time.Duration
    22	}
    23	
    24	// Runtime is the IO boundary for the session watcher.
    25	type Runtime interface {
    26		Now() time.Time
    27		Sleep(time.Duration)
    28		ReadFile(path string) ([]byte, error)
    29		ModTime(path string) (time.Time, error)
    30		BirthTime(path string) (time.Time, error)
    31		ListFiles(root string) ([]string, error)
    32		Descendants(root string) ([]string, error)
    33		LsofPaths(pid string) ([]string, error)
    34		ProcessAlive(pid string) bool
    35		AtomicWrite(path string, data []byte) error
    36		Log(outcome adapt.Outcome, detail string)
    37	}
    38	
    39	// Run discovers the async agent session id and writes config-<tag>-<agent>.json.
    40	func Run(opts Options, rt Runtime) error {
    41		spec, ok := SpecForAgent(opts.Agent, opts.Home)
    42		if !ok || opts.Tag == "" || opts.DataDir == "" {
    43			return nil
    44		}
    45		if opts.PIDWait <= 0 {
    46			opts.PIDWait = 2 * time.Second
    47		}
    48		if opts.Timeout <= 0 {
    49			opts.Timeout = 60 * time.Second
    50		}
    51		if opts.Poll <= 0 {
    52			opts.Poll = 100 * time.Millisecond
    53		}
    54	
    55		watchStart := rt.Now()
    56		pidFile := filepath.Join(opts.DataDir, "agent-pid-"+opts.Tag)
    57		out := filepath.Join(opts.DataDir, "config-"+opts.Tag+"-"+opts.Agent+".json")
    58	
    59		pidDeadline := watchStart.Add(opts.PIDWait)
    60		for {
    61			if fresh, _ := freshPID(pidFile, watchStart, rt); fresh {
    62				break
    63			}
    64			if !rt.Now().Before(pidDeadline) {
    65				break
    66			}
    67			rt.Sleep(opts.Poll)
    68		}
    69	
    70		rootPID := ""
    71		agentStart := time.Time{}
    72		if fresh, mod := freshPID(pidFile, watchStart, rt); fresh {
    73			if data, err := rt.ReadFile(pidFile); err == nil {
    74				rootPID = strings.TrimSpace(string(data))
    75				agentStart = mod
    76			}
    77		}
    78	
    79		legacyExisting := map[string]bool{}
    80		if rootPID == "" {
    81			files, _ := rt.ListFiles(spec.WatchDir)
    82			for _, file := range files {
    83				legacyExisting[file] = true
    84			}
    85		}
    86	
    87		nmLogged := false
    88		deadline := watchStart.Add(opts.Timeout)
    89		for rt.Now().Before(deadline) {
    90			if rootPID != "" && !rt.ProcessAlive(rootPID) {
    91				return nil
    92			}
    93	
    94			result := discover(spec, rootPID, agentStart, legacyExisting, rt)
    95			if result.ID != "" {
    96				payload, err := ConfigJSON(ConfigPayload{
    97					Agent:     opts.Agent,
    98					Args:      StripResumeArgs(opts.Agent, opts.Args),
    99					SessionID: result.ID,
   100				})
   101				if err != nil {
   102					return err
   103				}
   104				if err := rt.AtomicWrite(out, payload); err != nil {
   105					return err
   106				}
   107				rt.Log(adapt.Fired, "session_id="+result.ID)
   108				return nil
   109			}
   110			if result.NearMiss && !nmLogged {
   111				rt.Log(adapt.NearMiss, "matched session file but no id extracted: "+filepath.Base(result.Path))
   112				nmLogged = true
   113			}
   114	
   115			rt.Sleep(opts.Poll)
   116		}
   117	
   118		rt.Log(adapt.Fail, "no session id within 60s deadline (agent="+opts.Agent+")")
   119		return nil
   120	}
   121	
   122	func freshPID(pidFile string, since time.Time, rt Runtime) (bool, time.Time) {
   123		mod, err := rt.ModTime(pidFile)
   124		if err != nil {
   125			return false, time.Time{}
   126		}
   127		return !mod.Before(since), mod
   128	}
   129	
   130	func discover(spec AgentSpec, rootPID string, agentStart time.Time, legacyExisting map[string]bool, rt Runtime) SessionID {
   131		if rootPID != "" {
   132			pids, _ := rt.Descendants(rootPID)
   133			for _, pid := range pids {
   134				paths, _ := rt.LsofPaths(pid)
   135				for _, path := range paths {
   136					if result := spec.Match(path); result.ID != "" || result.NearMiss {
   137						return result
   138					}
   139				}
   140			}
   141			if !agentStart.IsZero() {
   142				return discoverByBirth(spec, agentStart, rt)
   143			}
   144			return SessionID{}
   145		}
   146		files, _ := rt.ListFiles(spec.WatchDir)
   147		for _, file := range files {
   148			if legacyExisting[file] {
   149				continue
   150			}
   151			if result := spec.Match(file); result.ID != "" || result.NearMiss {
   152				return result
   153			}
   154		}
   155		return SessionID{}
   156	}
   157	
   158	func discoverByBirth(spec AgentSpec, agentStart time.Time, rt Runtime) SessionID {
   159		files, _ := rt.ListFiles(spec.WatchDir)
   160		candidates := make([]SessionID, 0, 1)
   161		for _, file := range files {
   162			birth, err := rt.BirthTime(file)
   163			if err != nil || birth.Before(agentStart) {
   164				continue
   165			}
   166			result := spec.Match(file)
   167			if result.Matched {
   168				candidates = append(candidates, result)
   169			}
   170		}
   171		if len(candidates) == 1 {
   172			return candidates[0]
   173		}
   174		return SessionID{}
   175	}

exec
/bin/zsh -lc 'git diff --stat 370d43b87ba89fae64a534526cbb51223d88df76..HEAD && git diff --name-only 370d43b87ba89fae64a534526cbb51223d88df76..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 Makefile.local                                     |  12 +-
 atlas/architecture.md                              |  10 +-
 atlas/go-migration-inventory.md                    |  10 +-
 atlas/how-to-bring-up-a-new-harness-cli.md         |  10 +-
 bin/pair-session-watch.sh                          | 316 +--------------------
 cmd/internal/sessionwatch/run.go                   | 175 ++++++++++++
 cmd/internal/sessionwatch/run_test.go              | 250 ++++++++++++++++
 cmd/internal/sessionwatch/runtime.go               | 148 ++++++++++
 cmd/internal/sessionwatch/sessionwatch.go          | 125 ++++++++
 cmd/internal/sessionwatch/sessionwatch_test.go     | 112 ++++++++
 cmd/pair-session-watch/main.go                     |  50 ++++
 cmd/pair-session-watch/main_test.go                |  39 +++
 tests/pair-session-watch-test.sh                   |   8 +-
 workshop/issues/000078-go-stateful-shell-glue.md   |  36 ++-
 .../plans/000078-go-stateful-shell-glue-plan.md    |  18 +-
 15 files changed, 986 insertions(+), 333 deletions(-)
Makefile.local
atlas/architecture.md
atlas/go-migration-inventory.md
atlas/how-to-bring-up-a-new-harness-cli.md
bin/pair-session-watch.sh
cmd/internal/sessionwatch/run.go
cmd/internal/sessionwatch/run_test.go
cmd/internal/sessionwatch/runtime.go
cmd/internal/sessionwatch/sessionwatch.go
cmd/internal/sessionwatch/sessionwatch_test.go
cmd/pair-session-watch/main.go
cmd/pair-session-watch/main_test.go
tests/pair-session-watch-test.sh
workshop/issues/000078-go-stateful-shell-glue.md
workshop/plans/000078-go-stateful-shell-glue-plan.md

exec
/bin/zsh -lc "nl -ba cmd/internal/adapt/adapt.go | sed -n '1,220p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	// Package adapt writes the per-session adaptation flight recorder: one
     2	// JSON line per harness-adaptation event into
     3	// $PAIR_DATA_DIR/adapt-<tag>.jsonl.
     4	//
     5	// pair adapts each agent harness across a handful of integration aspects
     6	// (return-key remap, overlay suspension, slug generation, …; see
     7	// atlas/how-to-bring-up-a-new-harness-cli.md). Harnesses drift — a renamed
     8	// picker string or transcript shape silently breaks an adaptation — and the
     9	// breakage manifests as *silence*, not an error. The flight recorder makes
    10	// that drift observable: every adaptation logs when it fires AND when it
    11	// near-misses (the harness did something we half-recognized but no matcher
    12	// caught), so `pair-doctor` can read the trace and point at the broken aspect.
    13	//
    14	// Multiple components append concurrently (pair-wrap, pair-slug, plus shell
    15	// and Lua emitters writing the same line format from other processes). All
    16	// appends are O_APPEND of a single sub-PIPE_BUF line, which the kernel keeps
    17	// atomic across processes; bin/pair truncates the file once at session launch
    18	// so no writer ever races on truncation.
    19	package adapt
    20	
    21	import (
    22		"bytes"
    23		"encoding/json"
    24		"io"
    25		"os"
    26		"path/filepath"
    27		"sync"
    28		"time"
    29	)
    30	
    31	// maxDetail caps the free-text detail field. detail can carry a snippet of
    32	// agent output (e.g. an unrecognized prompt string), so it is bounded both to
    33	// keep lines small and to limit how much transcript content lands on disk.
    34	const maxDetail = 200
    35	
    36	// Outcome enumerates what happened when an adaptation point was reached.
    37	type Outcome string
    38	
    39	const (
    40		// Fired: the adaptation matched and acted as designed.
    41		Fired Outcome = "fired"
    42		// Bypass: the adaptation deliberately stepped aside (e.g. plain Enter
    43		// passed through as a bare CR because an overlay was active).
    44		Bypass Outcome = "bypass"
    45		// NearMiss: the harness emitted something we half-recognized but no
    46		// specific matcher caught it — the fingerprint of drift.
    47		NearMiss Outcome = "near-miss"
    48		// Fail: the adaptation was expected to work but couldn't (e.g. a session
    49		// id never resolved).
    50		Fail Outcome = "fail"
    51	)
    52	
    53	// event is one line of the flight recorder. Flat by design: detail is a
    54	// single capped string, never a nested object, so the shell and Lua emitters
    55	// can produce the identical shape with a one-line printf.
    56	type event struct {
    57		TS      string `json:"ts"`
    58		Comp    string `json:"comp"`
    59		Agent   string `json:"agent"`
    60		Aspect  int    `json:"aspect"`
    61		Signal  string `json:"signal"`
    62		Outcome string `json:"outcome"`
    63		Detail  string `json:"detail,omitempty"`
    64	}
    65	
    66	// Logger appends adaptation events for one component. A nil *Logger is a safe
    67	// no-op, so callers never have to nil-check — telemetry must never block or
    68	// crash the thing it observes.
    69	type Logger struct {
    70		mu     sync.Mutex
    71		w      io.Writer
    72		closer io.Closer
    73		comp   string
    74		agent  string
    75		now    func() time.Time
    76	}
    77	
    78	// DataDir returns $PAIR_DATA_DIR or the XDG default. This is the canonical
    79	// home for all per-session pair files; callers should use it rather than
    80	// re-deriving the path.
    81	func DataDir() string {
    82		if d := os.Getenv("PAIR_DATA_DIR"); d != "" {
    83			return d
    84		}
    85		if d := os.Getenv("XDG_DATA_HOME"); d != "" {
    86			return filepath.Join(d, "pair")
    87		}
    88		home, _ := os.UserHomeDir()
    89		return filepath.Join(home, ".local", "share", "pair")
    90	}
    91	
    92	// New builds a Logger writing to w. Used directly by tests; production code
    93	// uses Open. comp names the writing component (e.g. "pair-wrap"); agent is
    94	// the active harness basename.
    95	func New(w io.Writer, comp, agent string) *Logger {
    96		return &Logger{w: w, comp: comp, agent: agent, now: time.Now}
    97	}
    98	
    99	// Open opens the session flight recorder for appending and returns a Logger.
   100	// Returns nil (a no-op Logger) when $PAIR_TAG is unset or the file can't be
   101	// opened — telemetry failures are never fatal. The caller owns Close.
   102	func Open(comp, agent string) *Logger {
   103		tag := os.Getenv("PAIR_TAG")
   104		if tag == "" {
   105			return nil
   106		}
   107		path := filepath.Join(DataDir(), "adapt-"+tag+".jsonl")
   108		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
   109		if err != nil {
   110			return nil
   111		}
   112		l := New(f, comp, agent)
   113		l.closer = f
   114		return l
   115	}
   116	
   117	// Log appends one event. Safe on a nil receiver. Errors are swallowed.
   118	func (l *Logger) Log(aspect int, signal string, outcome Outcome, detail string) {
   119		if l == nil || l.w == nil {
   120			return
   121		}
   122		line := marshalEvent(l.now().UTC(), l.comp, l.agent, aspect, signal, outcome, detail)
   123		l.mu.Lock()
   124		_, _ = l.w.Write(line)
   125		l.mu.Unlock()
   126	}
   127	
   128	// Close releases the underlying file, if any. Safe on a nil receiver.
   129	func (l *Logger) Close() error {
   130		if l == nil || l.closer == nil {
   131			return nil
   132		}
   133		return l.closer.Close()
   134	}
   135	
   136	// marshalEvent renders one newline-terminated JSON line. Pure (time passed in)
   137	// so the line format is unit-testable without touching the clock or env.
   138	//
   139	// HTML escaping is disabled: the shell (jq) and Lua (vim.json) emitters that
   140	// write the same schema do NOT escape <, >, & — and detail routinely carries
   141	// agent output containing those (e.g. a "press > to continue" prompt). Keeping
   142	// Go in step makes all three emitters byte-identical and the detail readable.
   143	func marshalEvent(ts time.Time, comp, agent string, aspect int, signal string, outcome Outcome, detail string) []byte {
   144		var buf bytes.Buffer
   145		enc := json.NewEncoder(&buf) // Encoder appends '\n' itself
   146		enc.SetEscapeHTML(false)
   147		_ = enc.Encode(event{
   148			TS:      ts.Format(time.RFC3339),
   149			Comp:    comp,
   150			Agent:   agent,
   151			Aspect:  aspect,
   152			Signal:  signal,
   153			Outcome: string(outcome),
   154			Detail:  truncate(detail, maxDetail),
   155		})
   156		return buf.Bytes()
   157	}
   158	
   159	// truncate caps s to at most n bytes without splitting a multi-byte rune, so
   160	// the result is always valid UTF-8 (and thus valid JSON).
   161	func truncate(s string, n int) string {
   162		if len(s) <= n {
   163			return s
   164		}
   165		// Back up off any continuation bytes (0b10xxxxxx) at the cut point.
   166		for n > 0 && s[n]&0xC0 == 0x80 {
   167			n--
   168		}
   169		return s[:n]
   170	}

codex
The main split is as planned: pure matching/config helpers in `cmd/internal/sessionwatch`, side effects behind `Runtime`, and the shell name retained as a shim. I’m checking behavior deltas now, especially anything the old script handled implicitly.
exec
/bin/zsh -lc 'rg -n "pair-session-watch|PAIR_TAG|agent-pid" bin cmd tests Makefile.local atlas workshop/issues/000078-go-stateful-shell-glue.md workshop/plans/000078-go-stateful-shell-glue-plan.md' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
workshop/plans/000078-go-stateful-shell-glue-plan.md:5:**Goal:** Port `bin/pair-session-watch.sh` to a Go-owned command while keeping the legacy script name as a compatibility shim.
workshop/plans/000078-go-stateful-shell-glue-plan.md:7:**Architecture:** Keep the stateful watcher behavior split into pure session-watch decisions and a thin process/filesystem shell. `cmd/internal/sessionwatch` will own agent support, PID-tree/session-file matching, id extraction, resume-arg stripping, and config payload construction; `cmd/pair-session-watch` will own real time, process commands, atomic writes, and adapt-log emission. `bin/pair-session-watch.sh` remains the stable caller surface and execs the built Go binary for this migration window.
workshop/plans/000078-go-stateful-shell-glue-plan.md:15:#78 ports only `pair-session-watch.sh`. `pair-title.sh` is explicitly deferred because it has a separate long-running UI ownership surface: zellij frame titles, cmux workspace title ownership, activity buckets, singleton poller identity, and session liveness. Keeping these as separate issues reduces review risk and keeps each migrated script meaningfully testable. `ARCH-PURPOSE`: this still satisfies #78 by porting a prioritized stateful shell-glue subset and splitting the rest.
workshop/plans/000078-go-stateful-shell-glue-plan.md:53:| `PairSessionWatchCommand` | `cmd/pair-session-watch/main.go` | new | CLI, environment, process loop |
workshop/plans/000078-go-stateful-shell-glue-plan.md:54:| `PairSessionWatchShim` | `bin/pair-session-watch.sh` | modified | legacy shell command name |
workshop/plans/000078-go-stateful-shell-glue-plan.md:55:| `SessionWatchProcessTest` | `tests/pair-session-watch-test.sh` | modified | fake PATH commands and temp HOME/data dirs |
workshop/plans/000078-go-stateful-shell-glue-plan.md:62:- **PairSessionWatchCommand** — Parses `pair-session-watch <agent> <tag> <cwd> [agent-args...]`, no-ops unsupported agents, and runs the watcher.
workshop/plans/000078-go-stateful-shell-glue-plan.md:66:- **PairSessionWatchShim** — Keeps existing callers stable by execing `$PAIR_HOME/bin/pair-session-watch`.
workshop/plans/000078-go-stateful-shell-glue-plan.md:130:- Create: `cmd/pair-session-watch/main.go`
workshop/plans/000078-go-stateful-shell-glue-plan.md:148:- wait briefly for a fresh `agent-pid-<tag>` file.
workshop/plans/000078-go-stateful-shell-glue-plan.md:169:Create `cmd/pair-session-watch/main.go` as a thin CLI over the runtime. Update `Makefile.local` explicitly:
workshop/plans/000078-go-stateful-shell-glue-plan.md:170:- add `pair-session-watch` to `.PHONY`;
workshop/plans/000078-go-stateful-shell-glue-plan.md:171:- add `pair-session-watch` to `GO_BINS`;
workshop/plans/000078-go-stateful-shell-glue-plan.md:172:- add a per-binary `pair-session-watch: $(BIN_DIR)/pair-session-watch` alias;
workshop/plans/000078-go-stateful-shell-glue-plan.md:173:- add a `$(BIN_DIR)/pair-session-watch` build rule;
workshop/plans/000078-go-stateful-shell-glue-plan.md:174:- make `test-session-watch` depend on `$(BIN_DIR)/pair-session-watch` so repo-wide `make test` cannot run the shim process test before the Go binary exists.
workshop/plans/000078-go-stateful-shell-glue-plan.md:178:Run: `go test ./cmd/internal/sessionwatch ./cmd/pair-session-watch -count=1`
workshop/plans/000078-go-stateful-shell-glue-plan.md:184:- Modify: `bin/pair-session-watch.sh`
workshop/plans/000078-go-stateful-shell-glue-plan.md:185:- Modify: `tests/pair-session-watch-test.sh`
workshop/plans/000078-go-stateful-shell-glue-plan.md:193:- exec `$PAIR_HOME/bin/pair-session-watch "$@"`;
workshop/plans/000078-go-stateful-shell-glue-plan.md:198:Update `tests/pair-session-watch-test.sh` to exercise the shim invoking the Go binary with fake `ps`/`lsof` and temp HOME/data dirs. Keep the stale pidfile regression. Add a quoted arg in the saved config to prove JSON escaping is structured.
workshop/plans/000078-go-stateful-shell-glue-plan.md:200:Run: `make pair-session-watch && make test-session-watch`
workshop/plans/000078-go-stateful-shell-glue-plan.md:206:- `bin/pair-session-watch --help` or unsupported-agent smoke if no help is exposed.
workshop/plans/000078-go-stateful-shell-glue-plan.md:222:Record that `pair-session-watch` is now Go-owned with a shell shim, while `pair-title.sh` remains stateful shell glue.
workshop/plans/000078-go-stateful-shell-glue-plan.md:233:- `go test ./cmd/internal/sessionwatch ./cmd/pair-session-watch -count=1`
workshop/plans/000078-go-stateful-shell-glue-plan.md:235:- `make pair-session-watch`
workshop/issues/000078-go-stateful-shell-glue.md:24:- `pair-session-watch.sh` — session-id discovery, PID tree/lsof behavior, atomic config write.
workshop/issues/000078-go-stateful-shell-glue.md:30:Selected slice: port `pair-session-watch.sh` first. It owns session-id discovery, PID tree/lsof behavior, atomic restart-config writes, and adapt-log drift signals; those are high-value correctness and packaging surfaces with clear process-level fake coverage. `pair-title.sh` remains shell-owned for this issue and should become a follow-up because it owns a separate UI/title-poller surface: zellij frame titles, cmux workspace ownership, activity buckets, singleton poller identity, and session liveness. `ARCH-PURPOSE`: #78 is satisfied by porting a prioritized stateful subset and explicitly splitting the other stateful candidate instead of blending two long-running scripts into one review boundary.
workshop/issues/000078-go-stateful-shell-glue.md:74:- 2026-06-30: closed — go test ./cmd/internal/sessionwatch ./cmd/pair-session-watch -count=1; go test ./... -count=1; make pair-session-watch; make test-session-watch; bin/pair --help; bin/pair-dev --help; bin/pair-session-watch claude test /tmp
workshop/issues/000078-go-stateful-shell-glue.md:76:Claimed after #77 landed. Selected `pair-session-watch.sh` as the #78 slice from the #73 migration inventory because it owns restart-config correctness and brittle PID/lsof/session-file discovery. Split `pair-title.sh` out of this issue: it remains stateful shell glue, but its UI title-poller ownership is a separate risk surface. `ARCH-DRY`: the plan centralizes agent watch patterns, id extraction, resume-arg stripping, and config JSON in Go helpers instead of scattering them across shell conditionals. `ARCH-PURE`: pure parsing and config helpers are tested without process IO; process discovery stays behind a fakeable runtime.
workshop/issues/000078-go-stateful-shell-glue.md:78:Plan-quality returned INFO. Tightened the durable plan to reuse `cmd/internal/adapt.Open` / `adapt.Logger` directly, spell out `Makefile.local` wiring for `pair-session-watch`, and explicitly log that short shell scripts/opener scripts remain out of scope because this slice targets stateful session discovery.
workshop/issues/000078-go-stateful-shell-glue.md:80:Implemented `cmd/internal/sessionwatch` and `cmd/pair-session-watch`. The pure helpers own agent watch specs, codex/agy session-id extraction, near-miss classification, resume-arg stripping, and structured config JSON. The runtime owns PID freshness, process-tree/lsof discovery, birth-time and legacy fallbacks, atomic config writes, and adapt-log `fired` / `near-miss` / `fail` events via `cmd/internal/adapt`.
workshop/issues/000078-go-stateful-shell-glue.md:82:Replaced `bin/pair-session-watch.sh` with a compatibility shim that resolves `PAIR_HOME` and execs `bin/pair-session-watch`, leaving the existing `bin/pair` create-path caller unchanged. Updated `Makefile.local` so `pair-session-watch` is a built Go binary and `make test-session-watch` depends on it. Expanded `tests/pair-session-watch-test.sh` to exercise the shim, fake `lsof`, stale pidfile wait, codex resume stripping, and quote-safe JSON args.
workshop/issues/000078-go-stateful-shell-glue.md:84:Updated atlas migration docs to show `pair-session-watch` as Go-owned with a shell shim. `pair-title.sh` remains the next stateful shell candidate because it owns UI title-poller state. Short shell scripts and opener scripts remain intentionally shell-owned in this slice because their packaging/reliability payoff is lower than the stateful session-discovery watcher.
workshop/issues/000078-go-stateful-shell-glue.md:88:- `go test ./cmd/internal/sessionwatch ./cmd/pair-session-watch -count=1`
workshop/issues/000078-go-stateful-shell-glue.md:90:- `make pair-session-watch`
workshop/issues/000078-go-stateful-shell-glue.md:94:- `bin/pair-session-watch claude test /tmp`
Makefile.local:4:.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-session-watch pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
Makefile.local:29:GO_BINS     := pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-session-watch pair-go
Makefile.local:53:pair-session-watch:     $(BIN_DIR)/pair-session-watch
Makefile.local:84:test-session-watch: $(BIN_DIR)/pair-session-watch
Makefile.local:85:	bash tests/pair-session-watch-test.sh
Makefile.local:243:# pair-session-watch — async codex/agy session-id discovery. Legacy
Makefile.local:244:# bin/pair-session-watch.sh remains as a compatibility shim for bin/pair.
Makefile.local:245:$(BIN_DIR)/pair-session-watch: cmd/pair-session-watch/main.go cmd/internal/sessionwatch/sessionwatch.go cmd/internal/sessionwatch/run.go cmd/internal/sessionwatch/runtime.go cmd/internal/adapt/adapt.go go.mod
Makefile.local:246:	go build -o $@ ./cmd/pair-session-watch
bin/pair:248:        # agent-tracking feature shipped. Read PAIR_TAG from each process's
bin/pair:267:                  | awk -F= '$1=="PAIR_TAG"{print $2; exit}')
bin/pair:399:                       agent agent-pid agent-output agent-picks \
bin/pair:448:                agent-$old_tag|agent-pid-$old_tag|agent-output-$old_tag|\
bin/pair:735:    [ -z "$r_tag" ] && r_tag="$PAIR_TAG"
bin/pair:883:    [ "$(cat "$owner_file" 2>/dev/null)" = "$PAIR_TAG" ]
bin/pair:892:    printf '%s\n' "$PAIR_TAG" > "$owner_file"
bin/pair:1001:# under the same tag (handle_restart_marker, hoisted above). PAIR_TAG/PAIR_AGENT
bin/pair:1005:# propagates ZELLIJ / ZELLIJ_SESSION_NAME / PAIR_TAG to sibling non-pair panes
bin/pair:1017:    elif [ "$_in_pane" = "1" ] && [ -n "${PAIR_TAG:-}" ] \
bin/pair:1018:        && [ "${ZELLIJ_SESSION_NAME:-}" = "pair-${PAIR_TAG}" ]; then
bin/pair:1023:        : "${PAIR_TAG:?compaction needs PAIR_TAG}"
bin/pair:1026:            "$PAIR_TAG" >&2
bin/pair:1027:        park_scrollback "$PAIR_TAG" "$_cagent" --copy >/dev/null 2>&1 || true
bin/pair:1030:            printf 'tag=%s\n' "$PAIR_TAG"
bin/pair:1034:        } > "$HOME/.cache/pair/restart-pair-${PAIR_TAG}"
bin/pair:1035:        touch "$HOME/.cache/pair/quit-pair-${PAIR_TAG}"
bin/pair:1037:        exec ${PAIR_KILL_CMD:-zellij kill-session} "pair-${PAIR_TAG}"
bin/pair:1171:# PAIR_TAG is the namespace identifier — used to name the zellij session
bin/pair:1174:export PAIR_TAG="$default_tag"
bin/pair:1194:BASE_TAG="$PAIR_TAG"
bin/pair:1504:        quit_agent=$(cat "$DATA_DIR/agent-${PAIR_TAG}" 2>/dev/null || echo "$AGENT")
bin/pair:1515:        reap_nvim_for_tag "$PAIR_TAG"
bin/pair:1522:        _sb_base="$DATA_DIR/scrollback-${PAIR_TAG}-${quit_agent}"
bin/pair:1549:                    if _pbase="$(park_scrollback "$PAIR_TAG" "$quit_agent")"; then
bin/pair:1560:        rm -f "$DATA_DIR/outer-tty-${PAIR_TAG}" "$DATA_DIR/agent-${PAIR_TAG}" \
bin/pair:1561:              "$DATA_DIR/agent-output-${PAIR_TAG}" \
bin/pair:1563:              "$DATA_DIR/pair-wrap-pid-${PAIR_TAG}" \
bin/pair:1564:              "$DATA_DIR/adapt-${PAIR_TAG}.jsonl" \
bin/pair:1565:              "$DATA_DIR/image-capture-${PAIR_TAG}" \
bin/pair:1566:              "$DATA_DIR/image-capture-${PAIR_TAG}.done"
bin/pair:1575:        # Display SESSION (e.g. pair-2) rather than PAIR_TAG ("2") to match
bin/pair:1580:        resume_cfg="$(resolve_config_file "$PAIR_TAG" "$quit_agent")"
bin/pair:1598:        local poller_pid_file="$DATA_DIR/title-pid-${PAIR_TAG}"
bin/pair:1656:        /dev/*) printf '%s\n' "$outer" > "$DATA_DIR/outer-tty-${PAIR_TAG}" ;;
bin/pair:1657:        *)      rm -f "$DATA_DIR/outer-tty-${PAIR_TAG}" ;;
bin/pair:1671:    "$PAIR_HOME/bin/pair-title.sh" "$PAIR_TAG" "$title_agent" \
bin/pair:1679:    export PAIR_TAG="$chosen_tag"
bin/pair:1680:    SESSION="pair-${PAIR_TAG}"
bin/pair:1681:    DRAFT="$DATA_DIR/draft-${PAIR_TAG}.md"
bin/pair:1692:    title_agent=$(cat "$DATA_DIR/agent-$PAIR_TAG" 2>/dev/null | tr -d '\r\n[:space:]')
bin/pair:1820:    # Defensive strip: pair-session-watch.sh and the inline explicit_resume
bin/pair:2014:export PAIR_TAG="$chosen_tag"
bin/pair:2015:SESSION="pair-${PAIR_TAG}"
bin/pair:2016:DRAFT="$DATA_DIR/draft-${PAIR_TAG}.md"
bin/pair:2032:printf '%s\n' "$AGENT" > "$DATA_DIR/agent-${PAIR_TAG}"
bin/pair:2075:    # bin/pair-session-watch.sh; keep them in sync.
bin/pair:2093:    config_path="$(resolve_config_file "$PAIR_TAG" "$AGENT")"
bin/pair:2143:            config_path="$(resolve_config_file "$PAIR_TAG" "$AGENT")"
bin/pair:2206:# the id async via pair-session-watch.sh, so it is empty here; pair-changelog-open
bin/pair:2212:# components (pair-wrap, pair-session-watch.sh, pair-slug, nvim) append one
bin/pair:2218:: > "$DATA_DIR/adapt-${PAIR_TAG}.jsonl" 2>/dev/null || true
bin/pair:2227:"$PAIR_HOME/bin/pair-session-watch.sh" "$AGENT" "$PAIR_TAG" "$PWD" $agent_extra \
bin/pair:2264:        echo "pair: tag '$PAIR_TAG' makes zellij's session name too long for this" >&2
atlas/how-to-bring-up-a-new-harness-cli.md:59:- **Files:** `cmd/pair-session-watch` and `cmd/internal/sessionwatch` (`bin/pair-session-watch.sh` remains a compatibility shim).
atlas/how-to-bring-up-a-new-harness-cli.md:60:- Since TUI agents do not always expose session IDs on stdout, `pair-session-watch` runs in the background. It finds the agent process PID from `$PAIR_DATA_DIR/agent-pid-<tag>` (written by `pair-wrap`), walks its descendants, and inspects files held open by the processes via `lsof -p <pid>`.
atlas/how-to-bring-up-a-new-harness-cli.md:98:**Telemetry Signal** (aspect `3`, see §3): `session-id` from `pair-session-watch` — `fired` when `ExtractSessionID` resolves an id and the config is written, **`near-miss`** when a file matching the watch pattern is found but no id can be extracted (filename/format drift), `fail` when the 60s watch window elapses with no id at all (the session file never appeared where expected). The resume mapping in `bin/pair` is the *consumer* of this id; it's static config with no separate signal.
atlas/how-to-bring-up-a-new-harness-cli.md:154:3. [ ] **Implement Session Watching** in `cmd/internal/sessionwatch` / `cmd/pair-session-watch` (using `lsof` and target file patterns).
atlas/how-to-bring-up-a-new-harness-cli.md:203:| 3 Session watch | `session-id` | pair-session-watch | fired, near-miss, fail | `fail` (timeout) / `near-miss` (file found, id unparsed) |
cmd/pair-wrap/main.go:195:	// Resolved paths (empty when env didn't provide PAIR_TAG)
cmd/pair-wrap/main.go:434:	tag := os.Getenv("PAIR_TAG")
cmd/pair-wrap/main.go:446:	p.agentPIDPath = filepath.Join(dir, "agent-pid-"+tag)
cmd/pair-wrap/main.go:451:	tag := os.Getenv("PAIR_TAG")
cmd/pair-wrap/main.go:524:// signals don't double-spawn. pair-slug self-gates (no-op without PAIR_TAG)
cmd/pair-wrap/main.go:1789:	// file once per session launch, so we append. nil when PAIR_TAG is unset.
cmd/pair-wrap/main.go:1873:	// Drop the agent's PID so pair-session-watch.sh can bind discovery to
cmd/pair-wrap/main.go:1888:	// to send SIGUSR1; only enabled when PAIR_TAG/PAIR_DATA_DIR resolved
atlas/review-workbench.md:142:  `agent-pid-<tag>`; Codex/agy learn ids asynchronously, so review target handling must
tests/pair-continue-test.sh:134:  env HOME="$CRT" XDG_DATA_HOME="$CRT/xdg" PAIR_TAG=demo PAIR_AGENT=claude PAIR_KILL_CMD=true "$@"
atlas/go-migration-inventory.md:50:| `bin/pair` / `cmd/internal/launcher` / `cmd/internal/entrypoint` | Bash public launcher plus Go-owned launch handoff | user shell, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` (#77) resolves sibling `bin/pair` from the `pair-go` executable and execs it with `pair`-compatible argv/env. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves the compatibility handoff; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Go handoff entrypoint with `bin/pair` retained as stable public launcher and compat target through #77; real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
atlas/go-migration-inventory.md:55:| `bin/pair-wrap` / `cmd/pair-wrap` | Go binary | zellij agent pane | `pair-wrap [--scrollback-log PATH] agent [args...]`; transparent PTY proxy; long-running; failure in diagnostics is swallowed. | Reads Pair env and agent command; writes `agent-output-<tag>`, `agent-pid-<tag>`, scrollback `.raw`/`.events.jsonl`, image capture files; may invoke `pair-slug`. | go-subcommand `pair wrap`; keep old binary as compat alias for KDL until caller moves | P0 |
atlas/go-migration-inventory.md:56:| `bin/pair-slug` / `cmd/pair-slug` | Go binary | `pair-wrap` turn-end hook, tests | Env-driven, no stdin; resolves native transcript, proposes slug; exits 0 on most failures. | Requires `PAIR_TAG`, `PAIR_DATA_DIR`; reads config/transcripts/git branch; writes `slug-proposed-<tag>`; optional `PAIR_SLUG_*`, `OPENAI_API_KEY`. | go-subcommand `pair slug`; legacy binary retained during #76 | P1 |
atlas/go-migration-inventory.md:66:| `bin/pair-scrollback-open` | POSIX shell orchestrator | zellij Alt+/ Run, nvim Alt+b jump | `pair-scrollback-open [--jump prev|next]`; opens read-only nvim viewer; singleton lock. | Requires `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_HOME`; calls renderer, zellij IPC, nvim; writes `.ansi`, `.viewport`, lock. | shell-glue now; candidate Go orchestration after entrypoint, while `nvim/scrollback.lua` remains native | P1 |
atlas/go-migration-inventory.md:71:| `bin/pair-session-watch.sh` / `cmd/pair-session-watch` / `cmd/internal/sessionwatch` | Shell compatibility shim plus Go stateful watcher | `bin/pair` create path | `pair-session-watch.sh <agent> <tag> <cwd> [agent-args...]` execs the Go command; background 60s watcher; no-op for claude. | Reads agent pidfile, lsof/ps, native session dirs; writes config JSON atomically; logs adapt events through `cmd/internal/adapt`. | Go-owned watcher with legacy shim retained while `bin/pair` calls the `.sh` name (#78) | P1 |
atlas/go-migration-inventory.md:72:| `bin/lib/adapt-log.sh` | sourced shell helper | remaining shell emitters | `adapt_log comp agent aspect signal outcome [detail]`; no-op if no `PAIR_TAG` or jq. | Appends JSONL to `$PAIR_DATA_DIR/adapt-<tag>.jsonl`. | keep until remaining shell emitters move; schema stays DRY with Go/Lua emitters | P1 |
atlas/go-migration-inventory.md:76:| `bin/pair-notify` | Bash notification helper | agent hooks/manual shell inside Pair | `pair-notify [--osc 9|777] "message"`; writes OSC to outer tty; nonzero on bad args/missing tty. | Requires `PAIR_TAG`; reads `outer-tty-<tag>`. | small shell-glue; possible Go subcommand but low packaging impact | P2 |
atlas/go-migration-inventory.md:78:| `bin/pair-restart.sh` | Bash keybind helper | nvim restart confirmations | Writes restart marker then kill zellij session; supports `--new-session`. | Uses `PAIR_TAG`, `PAIR_AGENT`, `ZELLIJ_SESSION_NAME`, cache marker files. | small compat shell; can fold after entrypoint | P2 |
atlas/go-migration-inventory.md:80:| `bin/clipboard-to-pane.sh` | Bash copy/paste helper | `copy-on-select.sh`, direct zellij run possible | Reads OS clipboard, stages quote, focuses nvim, triggers Lua paste. | Uses pbpaste/wl-paste/xclip, jq, zellij, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_HOME`; writes quote and debug log. | shell-glue; keep until zellij copy flow has Go owner | P2 |
atlas/go-migration-inventory.md:137:- #78 ported the session-id watcher to `cmd/pair-session-watch` with
atlas/go-migration-inventory.md:138:  `bin/pair-session-watch.sh` retained as a shim. `pair-title.sh` remains the
atlas/go-migration-inventory.md:176:- `bin/pair-session-watch.sh`
tests/pair-rename.sh:37:    : > "$dd/agent-pid-$tag"
tests/pair-rename.sh:97:for f in agent-t1 agent-pid-t1 outer-tty-t1 draft-t1.md log-t1.md \
tests/pair-rename.sh:102:for f in agent-t2 agent-pid-t2 outer-tty-t2 draft-t2.md log-t2.md \
atlas/architecture.md:22:bin/pair-session-watch.sh    # shim to Go watcher that captures codex/agy session id (#000016, #000020, #78)
atlas/architecture.md:83:Resolves `$PAIR_HOME` from its own real path (portable bash, no `readlink -f`), prepends `$PAIR_HOME/bin` to `$PATH` (idempotent across re-launches) so all helper scripts resolve by bare name in zellij configs and keybinds, parses argv — first positional is `$PAIR_AGENT` (default `claude`), everything after `--` is joined into `$PAIR_AGENT_ARGS`, extra positionals before `--` are an error with a usage hint, defaults `$PAIR_TAG` to the cwd basename (the create-flow prompt or `pair resume <tag>` overrides it), resolves `$PAIR_DATA_DIR` to `${XDG_DATA_HOME:-$HOME/.local/share}/pair`, runs a one-time migration of any old `~/scratch/pair-{draft,log}-*` files, and dispatches:
atlas/architecture.md:129:Both panes wrap their command in `sh -c "..."` so the shell expands `$PAIR_AGENT`, `$PAIR_AGENT_ARGS`, `$PAIR_TAG`, and `$PAIR_HOME` at exec time — zellij itself does not interpolate env vars in `command`/`args` fields.
atlas/architecture.md:245:**Failure mode.** Both are designed to never block the agent. `pair-wrap` swallows exceptions in the detection/emission path and keeps proxying. `pair-notify` exits 0 with a stderr warning when `PAIR_TAG` is unset, the file is missing, or the recorded path isn't writable.
atlas/architecture.md:282:**Open (`bin/pair-scrollback-open`, POSIX sh).** Validates `PAIR_DATA_DIR` / `PAIR_TAG` / `PAIR_AGENT`, runs the renderer, then *launches* `nvim -u $PAIR_HOME/nvim/scrollback.lua $ANSI` as a child — deliberately **not** `exec`, so the script stays alive as nvim's parent and an `EXIT`/`INT`/`TERM` trap can clear the re-entrancy lock on quit. Errors print and `sleep` briefly so the message is readable before the floating pane self-closes. Bound in `zellij/config.kdl` to `Alt+/` as a 100% × 100% floating pane with `close_on_exit=true` — the user's `:q` in the viewer dismisses the pane and returns to pair's two-pane layout untouched. **Re-entrancy guard:** `Alt+/` is a global zellij bind, so pressing it again while the viewer is already focused fires another `Run` and would stack a second nvim (one `:q`/Esc per layer to unwind). zellij can't conditionally skip a `Run`, so the script self-guards: before launching nvim it writes its own PID to `$PAIR_DATA_DIR/scrollback-<tag>-<agent>.openlock`, and on entry it exits immediately if that lock already holds a *live* PID — the redundant floating pane then self-dismisses via `close_on_exit` and focus falls back to the open viewer. A stale lock (hard kill) carries a dead PID and is reclaimed by the next open's `kill -0` check. The draft pane's `Alt+b` (`--jump prev`) runs the same launcher, so it's covered too.
atlas/architecture.md:526:  `pair-session-watch.sh` codex/agy async path) → the **legacy unsuffixed base**
atlas/architecture.md:567:2. **Watcher (`cmd/pair-session-watch` via `bin/pair-session-watch.sh`, codex/agy only).** Spawned in the background by `bin/pair` on the create path, right before the zellij launch. The `.sh` file is now a compatibility shim; the stateful discovery logic lives in Go. Two discovery paths:
atlas/architecture.md:568:   - **PID-bound (preferred).** Reads `$PAIR_DATA_DIR/agent-pid-<tag>` (written by pair-wrap right after `pty.Start`) only when the pidfile's mtime is at-or-after the watcher's start, so a stale pidfile from a prior launch is ignored until pair-wrap overwrites it. Then it inspects open files in that PID's process tree via `lsof -p <pid> -Fn`. Race-free across concurrent pair sessions because lsof output is scoped to specific PIDs. Falls back internally to a birth-time-filtered directory walk if the agent doesn't keep its session file open: candidates are files with `stat -f %B >= agent_start_epoch`, and only a *single* candidate is accepted (multiple = concurrent race, refuse rather than guess).
atlas/architecture.md:616:`SESSION` rather than `PAIR_TAG` is shown — that's what the user just saw in the UI tab. `pair resume <tag>` accepts both forms (it strips a leading `pair-`).
atlas/architecture.md:624:1. **Tag-only families** (filename is `<prefix>-<tag>[<ext>]`, no further structure): `agent`, `agent-pid`, `agent-output`, `agent-picks`, `outer-tty`, `pair-wrap-pid`, `title-pid`, `layout-mode`, `queue` (dir), `quote`, `image-capture` + `.done`, `draft-<tag>.md`, `log-<tag>.md`, `nvim-pid-<tag>-{draft,scrollback}`.
atlas/architecture.md:657:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/config-<tag>-<agent>.json` — saved restart configuration for `(tag, agent)` (issue #000016, #000020). `{ agent, args, session_id }`. For claude, written synchronously by `bin/pair` before zellij launch (`--session-id` is deterministic). For codex/agy, written by the Go `pair-session-watch` command once the agent's session file is discovered via lsof. Read by `bin/pair`'s create-flow prompt and by the post-Alt+x hint. Survives Alt+x (unlike `agent-<tag>`, which is cleared) — that's the whole point: it's the bridge between two pair launches against the same tag.
atlas/architecture.md:659:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-pid-<tag>` — child agent PID written by `cmd/pair-wrap` immediately after `pty.Start`, removed on shutdown. Consumed by `cmd/pair-session-watch` to scope `lsof` discovery to a specific process tree (issue #000020). Mtime is also used as the agent-start epoch in the watcher's birth-time fallback.
atlas/architecture.md:663:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/pair-wrap-pid-<tag>` — single-line file containing pair-wrap's pid, written at startup by `bin/pair-wrap` if `PAIR_TAG` is set. Read by nvim's Alt+i (`attach_image`) so it can `kill -USR1 <pid>` to arm an image-capture window. Removed by pair-wrap on exit (the `finally` block in `main()`) and by `cleanup_quit_marker` as belt-and-suspenders on Alt+x.
atlas/architecture.md:667:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/slug-proposed-<tag>` and `slug-<tag>` — the orientation-slug channel (issue #000027). `pair-slug` (spawned by pair-wrap at turn-end) writes the proposed `=== <branch> | <focus> ===` to `slug-proposed-<tag>` (atomic temp+rename); nvim applies it to draft line 1 and writes the effective line back to `slug-<tag>`, which is the `prev` the proposer reads next turn. For Codex, if `config-<tag>-codex.json` is missing, `pair-slug` can recover the live rollout by reading `agent-pid-<tag>`, walking descendants via `ps`, and checking their `lsof` paths for `~/.codex/sessions/.../rollout-*.jsonl`. Similarly, for agy, the transcript is resolved from `~/.gemini/antigravity-cli/brain/<session_id>/.system_generated/logs/transcript.jsonl`. Codex model auth is API-key first, then Codex CLI subscription auth via `codex exec`. Single writer each, so the channel is race-free.
atlas/architecture.md:669:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/adapt-<tag>.jsonl` — the adaptation flight recorder (issue #000045). One JSON line per harness-adaptation trigger (`{ts, comp, agent, aspect, signal, outcome, detail}`), appended concurrently by `cmd/pair-wrap`, `cmd/pair-slug`, `cmd/pair-session-watch`, and `nvim/adapt.lua` — all writing one shared schema. Truncated once at session launch by `bin/pair` (so multi-process `O_APPEND` never races) and removed on full quit. Read by `doctor/doctor.sh` to surface integration drift (near-miss/fail signals). See `atlas/how-to-bring-up-a-new-harness-cli.md` §3 for the signal registry.
tests/review-indicator-test.sh:54:PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" \
tests/pair-review-target-test.sh:14:PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=codex \
tests/pair-review-target-test.sh:20:PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=codex \
tests/cmux-ownership-test.sh:54:        CMUX_WORKSPACE_ID="$WS" PAIR_TAG="$tag" \
tests/cmux-ownership-test.sh:95:    PAIR_TAG=211 PAIR_TEST_CALL=cmux_rename_workspace PAIR_TEST_ARGS="pair-211" \
tests/changelog-notify-test.sh:94:  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/pair-session-watch-test.sh:5:RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-session-watch-test.XXXXXX")"
tests/pair-session-watch-test.sh:22:echo 999999 > "$RT/data/agent-pid-test"
tests/pair-session-watch-test.sh:23:touch -t 200001010000 "$RT/data/agent-pid-test"
tests/pair-session-watch-test.sh:34:PAIR_TAG=test \
tests/pair-session-watch-test.sh:36:"$ROOT/bin/pair-session-watch.sh" codex test "$ROOT" resume old-session 'say "hi"' --no-alt-screen &
tests/pair-session-watch-test.sh:40:printf '%s\n' "$live_pid" > "$RT/data/agent-pid-test"
tests/pair-session-watch-test.sh:56:echo "pair-session-watch stale pidfile test PASS"
bin/pair-session-watch.sh:2:# Compatibility shim for the Go-owned pair-session-watch command.
bin/pair-session-watch.sh:15:cmd="$PAIR_HOME/bin/pair-session-watch"
bin/pair-session-watch.sh:17:    echo "pair-session-watch.sh: missing Go watcher at $cmd; run make pair-session-watch or source ../ariadne/construct/dev-aliases.sh in a dev shell" >&2
tests/changelog-open-test.sh:24:export PAIR_TAG="t"
bin/clipboard-to-pane.sh:77:tag="${PAIR_TAG:-${PAIR_AGENT:-claude}}"
bin/pair-notify:54:tag="${PAIR_TAG:-}"
bin/pair-notify:56:    echo "pair-notify: PAIR_TAG not set — not running inside a pair session" >&2
tests/statusline-pos-test.sh:68:  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
cmd/pair-session-watch/main.go:24:		fmt.Fprintf(stderr, "pair-session-watch: %v\n", err)
tests/adapt-schema-test.sh:26:export PAIR_TAG=golden PAIR_DATA_DIR="$RT" PAIR_AGENT=codex
tests/adapt-schema-test.sh:41:# Emits into the same file (PAIR_TAG=golden); it appends, so read the last line.
tests/adapt-schema-test.sh:60:PAIR_TAG=conc
tests/adapt-schema-test.sh:64:      PAIR_TAG=conc adapt_log golden codex 1 return-remap fired "writer-$i" ) &
bin/pair-changelog-open:18:#   $PAIR_DATA_DIR  $PAIR_TAG  $PAIR_AGENT  $PAIR_HOME
bin/pair-changelog-open:21:if [ -z "${PAIR_DATA_DIR:-}" ] || [ -z "${PAIR_TAG:-}" ] || [ -z "${PAIR_AGENT:-}" ]; then
bin/pair-changelog-open:22:    printf 'pair-changelog-open: missing PAIR_DATA_DIR / PAIR_TAG / PAIR_AGENT\n' >&2
bin/pair-changelog-open:31:# is discovered async by pair-session-watch.sh) fall back to the per-tag config
bin/pair-changelog-open:35:    cfg="$PAIR_DATA_DIR/config-$PAIR_TAG-$PAIR_AGENT.json"
bin/pair-changelog-open:38:base="$PAIR_DATA_DIR/changelog-$PAIR_TAG-$PAIR_AGENT${sid:+-$sid}"
bin/pair-changelog-open:39:RAW="$PAIR_DATA_DIR/scrollback-$PAIR_TAG-$PAIR_AGENT.raw"
bin/pair-changelog-open:40:EVENTS="$PAIR_DATA_DIR/scrollback-$PAIR_TAG-$PAIR_AGENT.events.jsonl"
bin/lib/adapt-log.sh:3:# Sourced by shell components (pair-session-watch.sh, …) that need to record a
bin/lib/adapt-log.sh:12:# No-op (returns 0) when PAIR_TAG is unset or jq is missing — telemetry must
bin/lib/adapt-log.sh:16:    [ -n "${PAIR_TAG:-}" ] || return 0
bin/lib/adapt-log.sh:38:        >> "$dir/adapt-${PAIR_TAG}.jsonl" 2>/dev/null || true
tests/cr-newline-test.sh:76:  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
bin/pair-scrollback-open:11:#   $PAIR_TAG       — session identity
bin/pair-scrollback-open:31:if [ -z "${PAIR_DATA_DIR:-}" ] || [ -z "${PAIR_TAG:-}" ] || [ -z "${PAIR_AGENT:-}" ]; then
bin/pair-scrollback-open:32:    printf 'pair-scrollback-open: missing PAIR_DATA_DIR / PAIR_TAG / PAIR_AGENT\n' >&2
bin/pair-scrollback-open:51:LOCK="$PAIR_DATA_DIR/scrollback-$PAIR_TAG-$PAIR_AGENT.openlock"
bin/pair-scrollback-open:59:RAW="$PAIR_DATA_DIR/scrollback-$PAIR_TAG-$PAIR_AGENT.raw"
bin/pair-scrollback-open:60:EVENTS="$PAIR_DATA_DIR/scrollback-$PAIR_TAG-$PAIR_AGENT.events.jsonl"
bin/pair-scrollback-open:61:ANSI="$PAIR_DATA_DIR/scrollback-$PAIR_TAG-$PAIR_AGENT.ansi"
bin/pair-scrollback-open:64:    printf 'pair-scrollback-open: no scrollback yet for %s/%s\n' "$PAIR_TAG" "$PAIR_AGENT" >&2
bin/pair-scrollback-open:92:VIEWPORT="$PAIR_DATA_DIR/scrollback-$PAIR_TAG-$PAIR_AGENT.viewport"
bin/pair-scrollback-open:186:export PAIR_NVIM_PID_FILE="$PAIR_DATA_DIR/nvim-pid-$PAIR_TAG-scrollback"
tests/review-readiness-cli-test.sh:38:prep_out="$(PAIR_HOME="$ROOT" PAIR_DATA_DIR="$RT" PAIR_TAG=prep PAIR_SESSION_ID=sid \
tests/autopair-test.sh:69:  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/review-toggle-test.sh:64:local sf = vim.env.PAIR_DATA_DIR .. '/review-' .. vim.env.PAIR_TAG .. '.open'
tests/review-toggle-test.sh:89:local target = vim.env.PAIR_DATA_DIR .. '/review-target-' .. vim.env.PAIR_TAG .. '.json'
tests/review-toggle-test.sh:125:os.remove(vim.env.PAIR_DATA_DIR .. '/config-' .. vim.env.PAIR_TAG .. '-claude.json')
tests/review-toggle-test.sh:138:  vim.env.PAIR_DATA_DIR .. '/config-' .. vim.env.PAIR_TAG .. '-' .. vim.env.PAIR_AGENT .. '.json')
tests/review-toggle-test.sh:149:os.remove(vim.env.PAIR_DATA_DIR .. '/config-' .. vim.env.PAIR_TAG .. '-codex.json')
tests/review-toggle-test.sh:150:vim.fn.writefile({ '111' }, vim.env.PAIR_DATA_DIR .. '/agent-pid-' .. vim.env.PAIR_TAG)
tests/review-toggle-test.sh:203:    PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" PAIR_SESSION_ID=testsid \
bin/pair-review-open:9:# Env (exported by bin/pair): $PAIR_DATA_DIR $PAIR_TAG $PAIR_HOME
bin/pair-review-open:21:if [ -z "${PAIR_DATA_DIR:-}" ] || [ -z "${PAIR_TAG:-}" ] || [ -z "${PAIR_HOME:-}" ]; then
bin/pair-review-open:22:    printf 'pair-review-open: missing PAIR_DATA_DIR / PAIR_TAG / PAIR_HOME\n' >&2
bin/pair-review-open:29:# Path == nvim/review/seam.lua's `open_state` formula; PAIR_TAG is guaranteed
bin/pair-review-open:31:STATE="$PAIR_DATA_DIR/review-$PAIR_TAG.open"
bin/pair-review-open:42:export PAIR_NVIM_PID_FILE="$PAIR_DATA_DIR/nvim-pid-$PAIR_TAG-review"
cmd/pair-slug/main_test.go:57:		"PAIR_TAG=testtag",
cmd/pair-slug/main_test.go:129:		"PAIR_TAG=testtag",
cmd/pair-slug/main_test.go:170:	if err := os.WriteFile(filepath.Join(dataDir, "agent-pid-testtag"), []byte("10\n"), 0o644); err != nil {
cmd/pair-slug/main.go:14://	PAIR_TAG, PAIR_DATA_DIR   required; identify the session
cmd/pair-slug/main.go:138:	b, err := os.ReadFile(filepath.Join(dataDir, "agent-pid-"+tag))
cmd/pair-slug/main.go:163:	tag := os.Getenv("PAIR_TAG")
cmd/pair-slug/main.go:166:		logf("no PAIR_TAG/PAIR_DATA_DIR; not inside a pair session")
tests/zellij-trace-test.sh:44:PAIR_TAG=trace \
tests/changelog-session-key-test.sh:11:export PAIR_DATA_DIR="$tmp/data" PAIR_TAG=t PAIR_AGENT=claude
bin/pair-review-target:15:tag="${PAIR_TAG:-default}"
bin/pair-review-target:26:  pid_file="$PAIR_DATA_DIR/agent-pid-$tag"
tests/review-poke-test.sh:49:PATH="$RT/bin:$PATH" PAIR_ROOT="$ROOT" RESULT="$RESULT" PAIR_DATA_DIR="$RT" PAIR_TAG=poke \
tests/queue-send-test.sh:46:    env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/review-window-test.sh:43:( cd "$RT" && PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" RESULT="$RT/r1" \
tests/review-window-test.sh:48:if PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-open" "$RT/nope.md" 2>/dev/null; then
tests/review-window-test.sh:54:PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-open" "$RT/doc.md" || true
tests/review-window-test.sh:81:( cd "$REPO" && PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/review-window-test.sh:323:( cd "$REPO" && PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
cmd/internal/sessionwatch/run.go:56:	pidFile := filepath.Join(opts.DataDir, "agent-pid-"+opts.Tag)
cmd/internal/sessionwatch/run_test.go:19:	rt.files[filepath.Join(data, "agent-pid-test")] = fakeFile{content: []byte("999999\n"), mod: time.Unix(1, 0)}
cmd/internal/sessionwatch/run_test.go:21:		rt.files[filepath.Join(data, "agent-pid-test")] = fakeFile{content: []byte("1234\n"), mod: time.Unix(100, 0)}
cmd/internal/sessionwatch/run_test.go:58:	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("2000\n"), mod: time.Unix(200, 0)}
cmd/internal/sessionwatch/run_test.go:89:	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("3000\n"), mod: time.Unix(300, 0)}
cmd/internal/adapt/adapt_test.go:127:	t.Setenv("PAIR_TAG", "")
cmd/internal/adapt/adapt_test.go:130:		t.Fatalf("Open with empty PAIR_TAG should return nil, got %v", l)
cmd/internal/adapt/adapt_test.go:140:	t.Setenv("PAIR_TAG", "t1")
cmd/internal/adapt/adapt_test.go:145:		t.Fatal("Open returned nil with PAIR_TAG set")
cmd/internal/adapt/adapt.go:100:// Returns nil (a no-op Logger) when $PAIR_TAG is unset or the file can't be
cmd/internal/adapt/adapt.go:103:	tag := os.Getenv("PAIR_TAG")
cmd/internal/transcript/transcript.go:18:// config-<tag>-<agent>.json (written by bin/pair / pair-session-watch.sh).

exec
/bin/zsh -lc "nl -ba tests/pair-session-watch-test.sh | sed -n '1,180p'" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "nl -ba cmd/pair-session-watch/main.go | sed -n '1,120p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package main
     2	
     3	import (
     4		"fmt"
     5		"os"
     6		"time"
     7	
     8		"github.com/xianxu/pair/cmd/internal/adapt"
     9		"github.com/xianxu/pair/cmd/internal/sessionwatch"
    10	)
    11	
    12	func main() {
    13		os.Exit(run(os.Args[1:], os.Getenv, os.Stderr))
    14	}
    15	
    16	func run(args []string, getenv func(string) string, stderr *os.File) int {
    17		opts, ok := buildOptions(args, getenv)
    18		if !ok {
    19			return 0
    20		}
    21		logger := adapt.Open("session-watch", opts.Agent)
    22		defer logger.Close()
    23		if err := sessionwatch.Run(opts, sessionwatch.NewOSRuntime(logger)); err != nil {
    24			fmt.Fprintf(stderr, "pair-session-watch: %v\n", err)
    25			return 1
    26		}
    27		return 0
    28	}
    29	
    30	func buildOptions(args []string, getenv func(string) string) (sessionwatch.Options, bool) {
    31		if len(args) < 3 {
    32			return sessionwatch.Options{}, false
    33		}
    34		home := getenv("HOME")
    35		dataDir := getenv("PAIR_DATA_DIR")
    36		if dataDir == "" {
    37			dataDir = adapt.DataDir()
    38		}
    39		return sessionwatch.Options{
    40			Agent:   args[0],
    41			Tag:     args[1],
    42			Cwd:     args[2],
    43			Args:    append([]string(nil), args[3:]...),
    44			Home:    home,
    45			DataDir: dataDir,
    46			PIDWait: sessionwatch.ParseDurationSeconds(getenv("PAIR_SESSION_WATCH_PID_WAIT_SECONDS"), 2*time.Second),
    47			Timeout: 60 * time.Second,
    48			Poll:    100 * time.Millisecond,
    49		}, true
    50	}

 succeeded in 0ms:
     1	#!/usr/bin/env bash
     2	set -euo pipefail
     3	
     4	ROOT="$(cd "$(dirname "$0")/.." && pwd)"
     5	RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-session-watch-test.XXXXXX")"
     6	trap 'rm -rf "$RT"; [ -z "${live_pid:-}" ] || kill "$live_pid" 2>/dev/null || true' EXIT
     7	
     8	mkdir -p "$RT/bin" "$RT/data" "$RT/home/.codex/sessions/2026/06/25"
     9	
    10	sid="019eff64-6ceb-7e72-9d41-a735a97029ac"
    11	session_file="$RT/home/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-$sid.jsonl"
    12	: > "$session_file"
    13	
    14	cat > "$RT/bin/lsof" <<SH
    15	#!/usr/bin/env bash
    16	if [ "\$1" = "-p" ] && [ "\$2" = "__LIVE_PID__" ]; then
    17	  printf 'p%s\nn%s\n' "__LIVE_PID__" "$session_file"
    18	fi
    19	SH
    20	chmod +x "$RT/bin/lsof"
    21	
    22	echo 999999 > "$RT/data/agent-pid-test"
    23	touch -t 200001010000 "$RT/data/agent-pid-test"
    24	
    25	sleep 10 &
    26	live_pid=$!
    27	sed "s/__LIVE_PID__/$live_pid/g" "$RT/bin/lsof" > "$RT/bin/lsof.tmp"
    28	mv "$RT/bin/lsof.tmp" "$RT/bin/lsof"
    29	chmod +x "$RT/bin/lsof"
    30	
    31	PATH="$RT/bin:$PATH" \
    32	HOME="$RT/home" \
    33	PAIR_DATA_DIR="$RT/data" \
    34	PAIR_TAG=test \
    35	PAIR_SESSION_WATCH_PID_WAIT_SECONDS=3 \
    36	"$ROOT/bin/pair-session-watch.sh" codex test "$ROOT" resume old-session 'say "hi"' --no-alt-screen &
    37	watch_pid=$!
    38	
    39	sleep 0.2
    40	printf '%s\n' "$live_pid" > "$RT/data/agent-pid-test"
    41	
    42	wait "$watch_pid"
    43	
    44	got="$(jq -r '.session_id // empty' "$RT/data/config-test-codex.json")"
    45	[ "$got" = "$sid" ] || {
    46	  echo "session_id mismatch: got '$got', want '$sid'" >&2
    47	  exit 1
    48	}
    49	
    50	args="$(jq -c '.args' "$RT/data/config-test-codex.json")"
    51	[ "$args" = '["say \"hi\"","--no-alt-screen"]' ] || {
    52	  echo "args mismatch: got '$args'" >&2
    53	  exit 1
    54	}
    55	
    56	echo "pair-session-watch stale pidfile test PASS"

exec
/bin/zsh -lc "nl -ba bin/pair-session-watch.sh | sed -n '1,80p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	#!/usr/bin/env bash
     2	# Compatibility shim for the Go-owned pair-session-watch command.
     3	
     4	set -uo pipefail
     5	
     6	SOURCE="${BASH_SOURCE[0]}"
     7	while [ -L "$SOURCE" ]; do
     8	    DIR="$(cd -P "$(dirname "$SOURCE")" && pwd)"
     9	    SOURCE="$(readlink "$SOURCE")"
    10	    [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
    11	done
    12	PAIR_HOME="$(cd -P "$(dirname "$SOURCE")/.." && pwd)"
    13	export PAIR_HOME
    14	
    15	cmd="$PAIR_HOME/bin/pair-session-watch"
    16	if [ ! -x "$cmd" ]; then
    17	    echo "pair-session-watch.sh: missing Go watcher at $cmd; run make pair-session-watch or source ../ariadne/construct/dev-aliases.sh in a dev shell" >&2
    18	    exit 1
    19	fi
    20	
    21	exec "$cmd" "$@"

exec
/bin/zsh -lc "nl -ba Makefile.local | sed -n '1,120p;220,260p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	# Repo-specific Makefile targets.
     2	# This file is included by Makefile — add your own targets here.
     3	
     4	.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-session-watch pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
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
    29	GO_BINS     := pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-session-watch pair-go
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
    53	pair-session-watch:     $(BIN_DIR)/pair-session-watch
    54	pair-go:                $(BIN_DIR)/pair-go
    55	
    56	# Tests
    57	#   make test       → go test ./... (cache disabled so reruns actually run)
    58	#   make test-race  → race-detector run, scoped to packages where the
    59	#                     suite has actual concurrent code worth checking.
    60	#                     pair-wrap has translateStdin's goroutine + timer +
    61	#                     channels; pair-scrollback-render's render() spawns
    62	#                     a goroutine drainer that races with em.Close()
    63	#                     inside the vt.Emulator library API itself, which is
    64	#                     outside our scope to fix — exclude that package
    65	#                     here rather than chase a library-level race.
    66	test: test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace test-review
    67		go test ./... -count=1
    68	
    69	# pair-title poller: single-instance guard (identity-checked liveness so a
    70	# recycled PID can't suppress the respawn) + the #71 frame meter (renames each
    71	# agent pane to "<agent> (<count>) [<cwd>]", with unchanged-skip). See
    72	# tests/pair-title-poller-test.sh.
    73	test-pair-title:
    74		bash tests/pair-title-poller-test.sh
    75	
    76	# cmux workspace ownership claim (presence beats a stale owner flag), so a
    77	# launch/attach/restart reclaims its workspace title. See
    78	# tests/cmux-ownership-test.sh.
    79	test-cmux-ownership:
    80		bash tests/cmux-ownership-test.sh
    81	
    82	# Session id watcher must ignore stale pidfiles from prior launches and wait
    83	# for pair-wrap to publish the current agent PID.
    84	test-session-watch: $(BIN_DIR)/pair-session-watch
    85		bash tests/pair-session-watch-test.sh
    86	
    87	# Review workbench (#66): the integration shell tests. The pure modules
    88	# (record, reconstruct) run under `make test-lua`.
    89	test-review:
    90		bash tests/review-docflow-test.sh
    91		bash tests/review-apply-test.sh
    92		bash tests/review-handoff-test.sh
    93		bash tests/review-loop-test.sh
    94		bash tests/review-projection-test.sh
    95		bash tests/review-poke-test.sh
    96		bash tests/review-window-test.sh
    97		bash tests/review-toggle-test.sh
    98		bash tests/pair-review-target-test.sh
    99		bash tests/review-readiness-cli-test.sh
   100		bash tests/review-indicator-test.sh
   101		bash tests/review-resume-test.sh
   102	
   103	test-race:
   104		go test -count=1 -race ./cmd/pair-wrap/
   105	
   106	# Self-test for the headless-nvim timeout watchdog (tests/lib/run-headless.sh,
   107	# #60). Runs FIRST in `test` so the safety net every headless-nvim sub-target
   108	# now relies on is proven before those targets run — without it, a driver that
   109	# fails to quit (bare `qall` on a modified buffer → E37) hangs the whole suite.
   110	test-run-headless:
   111		bash tests/run-headless-test.sh
   112	
   113	# Headless Lua tests for nvim/*.lua decision modules (issue #000027). nvim -l
   114	# runs a Lua script in nvim's interpreter with the full vim.api available, so
   115	# pure logic AND buffer-manipulation can be asserted without a UI.
   116	test-lua:
   117		nvim -l nvim/slug_test.lua
   118		nvim -l nvim/scrollback_test.lua
   119		nvim -l nvim/annotate_test.lua
   120		nvim -l nvim/changelog_test.lua
   220		go build -o $@ ./cmd/pair-scribe
   221	
   222	# pair-scrollback-render — replays a pair-wrap raw PTY capture through
   223	# charmbracelet/x/vt and emits one ANSI-styled line per logical scrollback
   224	# row, for the Alt+/ viewer to display through nvim.
   225	$(BIN_DIR)/pair-scrollback-render: cmd/pair-scrollback-render/main.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
   226		go build -o $@ ./cmd/pair-scrollback-render
   227	
   228	# pair-wrap — transparent PTY proxy around a TUI coding agent: forwards
   229	# stdio, tees scrollback log, extracts colored spans for nvim autocomplete,
   230	# detects OSC/BEL and forwards as OSC 9 to the recorded outer-TTY, handles
   231	# SIGUSR1 image-paste capture window.
   232	$(BIN_DIR)/pair-wrap: cmd/pair-wrap/main.go go.mod
   233		go build -o $@ ./cmd/pair-wrap
   234	
   235	$(BIN_DIR)/pair-slug: cmd/pair-slug/main.go cmd/pair-slug/slug.go cmd/internal/model/model.go cmd/internal/transcript/transcript.go go.mod
   236		go build -o $@ ./cmd/pair-slug
   237	
   238	# pair-context — one-shot: print a pane's current context size (token count)
   239	# for the pair-title poller. Pure ctxmeter reader + shared transcript resolver.
   240	$(BIN_DIR)/pair-context: cmd/pair-context/main.go cmd/internal/contextcmd/contextcmd.go cmd/internal/ctxmeter/ctxmeter.go cmd/internal/transcript/transcript.go go.mod
   241		go build -o $@ ./cmd/pair-context
   242	
   243	# pair-session-watch — async codex/agy session-id discovery. Legacy
   244	# bin/pair-session-watch.sh remains as a compatibility shim for bin/pair.
   245	$(BIN_DIR)/pair-session-watch: cmd/pair-session-watch/main.go cmd/internal/sessionwatch/sessionwatch.go cmd/internal/sessionwatch/run.go cmd/internal/sessionwatch/runtime.go cmd/internal/adapt/adapt.go go.mod
   246		go build -o $@ ./cmd/pair-session-watch
   247	
   248	# pair-changelog — distills the pair TTY into the Alt+l change log (#53);
   249	# shares cmd/internal/model with pair-slug.
   250	$(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill.go cmd/pair-changelog/prompt.go cmd/internal/model/model.go go.mod
   251		go build -o $@ ./cmd/pair-changelog
   252	
   253	# pair-continuation — the deterministic writer for the `continuation` datatype
   254	# (ariadne#91): renders frontmatter, allocates a collision-safe timestamped
   255	# name under workshop/continuation/, writes, then commits + pushes (recovery).
   256	$(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-continuation/git.go cmd/pair-continuation/main.go go.mod
   257		go build -o $@ ./cmd/pair-continuation
   258	
   259	# pair-go — development dispatcher and Go-owned launch handoff.
   260	# `pair-go launch ...` execs sibling bin/pair for the current migration window;

codex
One possible review target is compatibility with the deleted shell implementation, so I’m comparing the old watcher’s edge semantics rather than only reading the new code in isolation.
exec
/bin/zsh -lc "git show 370d43b87ba89fae64a534526cbb51223d88df76:bin/pair-session-watch.sh | nl -ba | sed -n '1,340p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	#!/usr/bin/env bash
     2	# pair-session-watch.sh — capture the agent's session-id by inspecting
     3	# files held open by the agent's process tree, then write a per-(tag,agent)
     4	# restart config under $PAIR_DATA_DIR.
     5	#
     6	# Usage:
     7	#   pair-session-watch.sh <agent> <tag> <cwd> [agent-args...]
     8	#
     9	# Spawned in the background by bin/pair right before zellij launch on the
    10	# new-session path. Issue #000020 replaced the earlier "first new file in
    11	# the watch dir" snapshot with PID-bound discovery: two pair sessions in
    12	# the same cwd previously raced to claim whichever agent's session file
    13	# appeared first, occasionally cross-wiring tags' configs.
    14	#
    15	# Per-agent surface:
    16	#   claude  — no-op. bin/pair pre-injects `--session-id <uuid>` and writes
    17	#             config-<tag>-claude.json synchronously, so there's nothing
    18	#             left to discover at runtime.
    19	#   codex   — open file under ~/.codex/sessions/.../rollout-*-<uuid>.jsonl.
    20	#             id = trailing UUID in filename.
    21	#
    22	#
    23	# The pidfile ($PAIR_DATA_DIR/agent-pid-<tag>) is dropped by pair-wrap
    24	# right after pty.Start; we wait briefly for it, then poll
    25	# `lsof -p <pid>` against that pid + descendants.
    26	#
    27	# Config write is atomic (tmp + rename). Watcher self-times-out after 60s
    28	# so a stuck agent doesn't leak the background process.
    29	
    30	set -uo pipefail
    31	
    32	agent="${1:-}"
    33	tag="${2:-}"
    34	cwd="${3:-}"
    35	[ -z "$agent" ] || [ -z "$tag" ] || [ -z "$cwd" ] && exit 0
    36	shift 3
    37	args=( "$@" )
    38	
    39	# Claude is fully handled at launch time. Codex and agy need lsof.
    40	case "$agent" in
    41	    codex|agy) ;;
    42	    *) exit 0 ;;
    43	esac
    44	
    45	DATA_DIR="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
    46	mkdir -p "$DATA_DIR"
    47	out="$DATA_DIR/config-$tag-$agent.json"
    48	pid_file="$DATA_DIR/agent-pid-$tag"
    49	
    50	# Aspect 3 flight recorder (atlas §3). PAIR_TAG is normally exported by bin/pair;
    51	# fall back to our positional tag so logging works even if run standalone.
    52	: "${PAIR_TAG:=$tag}"
    53	export PAIR_TAG
    54	_swdir="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    55	# shellcheck source=bin/lib/adapt-log.sh
    56	. "$_swdir/lib/adapt-log.sh" 2>/dev/null || true
    57	nm_logged=0 # dedup the session-id near-miss across poll iterations
    58	
    59	# Per-agent: directory we walk + the find pattern. Used by both the
    60	# PID-bound primary path (for lsof path matching) and the legacy
    61	# snapshot-diff fallback (for pair-wrap binaries that don't publish
    62	# the pidfile yet).
    63	case "$agent" in
    64	    codex)
    65	        watch_dir="$HOME/.codex/sessions"
    66	        find_args=(-type f -name 'rollout-*.jsonl')
    67	        ;;
    68	
    69	    agy)
    70	        watch_dir="$HOME/.gemini/antigravity-cli/conversations"
    71	        find_args=(-type f -name '*.db')
    72	        ;;
    73	esac
    74	mkdir -p "$watch_dir"
    75	
    76	# Wait briefly for pair-wrap to publish this launch's agent PID. A pidfile from
    77	# a prior run can survive until pair-wrap overwrites it; binding to that stale
    78	# PID makes the watcher exit before the current agent starts.
    79	watch_start=$(date +%s)
    80	pid_wait_seconds="${PAIR_SESSION_WATCH_PID_WAIT_SECONDS:-2}"
    81	case "$pid_wait_seconds" in
    82	    ''|*[!0-9]*) pid_wait_seconds=2 ;;
    83	esac
    84	pid_deadline=$(( watch_start + pid_wait_seconds ))
    85	pid_fresh() {
    86	    [ -s "$pid_file" ] || return 1
    87	    mt=$(stat -f %m "$pid_file" 2>/dev/null || echo 0)
    88	    [ "$mt" -ge "$watch_start" ]
    89	}
    90	while ! pid_fresh && [ "$(date +%s)" -lt "$pid_deadline" ]; do
    91	    sleep 0.1
    92	done
    93	
    94	root_pid=""
    95	agent_start=0
    96	if pid_fresh; then
    97	    root_pid=$(cat "$pid_file" 2>/dev/null)
    98	    # pair-wrap writes the pidfile right after pty.Start, so its mtime
    99	    # is a tight upper bound on the agent's start epoch. Used as a
   100	    # birth-time floor: any session file whose birth predates this was
   101	    # created by an earlier pair session, not ours.
   102	    agent_start=$(stat -f %m "$pid_file" 2>/dev/null || echo 0)
   103	fi
   104	
   105	# Legacy fallback state: snapshot the watch dir at start. Only consulted
   106	# when the PID-bound path can't bind (no pidfile) — preserves the
   107	# pre-#000020 behavior so old pair-wrap installs still capture sessions
   108	# in the single-session case. Cross-tag races re-emerge in that path;
   109	# the proper fix is to rebuild pair-wrap so the pidfile shows up.
   110	legacy_existing=""
   111	if [ -z "$root_pid" ]; then
   112	    legacy_existing=$(find "$watch_dir" "${find_args[@]}" 2>/dev/null | sort)
   113	fi
   114	
   115	# pid + descendants. Codex/agy may keep the session file open in a native
   116	# child below a JS launcher process. Use ps instead of pgrep -P: on macOS,
   117	# pgrep can miss children that ps still reports, which makes the PID-bound
   118	# discovery path inspect only the launcher and miss Codex's rollout fd.
   119	descendants() {
   120	    local root="$1"
   121	    ps -axo pid=,ppid= 2>/dev/null | awk -v root="$root" '
   122	        { children[$2] = children[$2] " " $1 }
   123	        END {
   124	            queue[1] = root
   125	            seen[root] = 1
   126	            head = 1
   127	            tail = 1
   128	            while (head <= tail) {
   129	                p = queue[head++]
   130	                print p
   131	                n = split(children[p], kids, " ")
   132	                for (i = 1; i <= n; i++) {
   133	                    k = kids[i]
   134	                    if (k != "" && !seen[k]) {
   135	                        seen[k] = 1
   136	                        queue[++tail] = k
   137	                    }
   138	                }
   139	            }
   140	        }'
   141	}
   142	
   143	match_path() {
   144	    local line="$1"
   145	    case "$agent" in
   146	        codex)
   147	            case "$line" in
   148	                "$HOME/.codex/sessions/"*"/rollout-"*".jsonl") echo "$line" ;;
   149	            esac
   150	            ;;
   151	
   152	        agy)
   153	            case "$line" in
   154	                "$HOME/.gemini/antigravity-cli/conversations/"*".db") echo "$line" ;;
   155	            esac
   156	            ;;
   157	    esac
   158	}
   159	
   160	extract_id() {
   161	    case "$agent" in
   162	        codex)
   163	            local fn
   164	            fn=$(basename "$1" .jsonl)
   165	            if [[ "$fn" =~ ([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$ ]]; then
   166	                echo "${BASH_REMATCH[1]}"
   167	            fi
   168	            ;;
   169	
   170	        agy)
   171	            # Path is like: ~/.gemini/antigravity-cli/conversations/<uuid>.db
   172	            # The UUID is the basename of the file without the .db extension.
   173	            local fn
   174	            fn=$(basename "$1" .db)
   175	            if [[ "$fn" =~ ^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$ ]]; then
   176	                echo "$fn"
   177	            fi
   178	            ;;
   179	    esac
   180	}
   181	
   182	deadline=$(( $(date +%s) + 60 ))
   183	while [ "$(date +%s)" -lt "$deadline" ]; do
   184	    # If we have a root_pid and the agent's gone, nothing more to do.
   185	    if [ -n "$root_pid" ]; then
   186	        kill -0 "$root_pid" 2>/dev/null || exit 0
   187	    fi
   188	
   189	    sid=""
   190	    matched_file=""
   191	    nm_candidate="" # a file that matched our pattern but yielded no id this pass
   192	
   193	    if [ -n "$root_pid" ]; then
   194	        # Primary path: lsof against the agent's PID tree. Race-free
   195	        # because lsof output is scoped to specific PIDs, so a peer pair
   196	        # session in the same cwd can't masquerade as ours.
   197	        while IFS= read -r p; do
   198	            [ -z "$p" ] && continue
   199	            # `lsof -Fn` emits one record per fd: 'p' header line then 'n'
   200	            # for the path. We only care about the n-prefixed lines.
   201	            while IFS= read -r line; do
   202	                [ "${line:0:1}" = "n" ] || continue
   203	                path="${line:1}"
   204	                hit=$(match_path "$path")
   205	                [ -z "$hit" ] && continue
   206	                cand=$(extract_id "$hit")
   207	                if [ -n "$cand" ]; then
   208	                    sid="$cand"
   209	                    matched_file="$hit"
   210	                    break 2
   211	                fi
   212	                nm_candidate="$hit"
   213	            done < <(lsof -p "$p" -Fn 2>/dev/null)
   214	        done < <(descendants "$root_pid")
   215	
   216	        # Birth-time fallback: lsof can miss agents that close the fd
   217	        # between writes. Walk the watch dir for files born at-or-after
   218	        # our agent's start epoch (so files from earlier pair sessions
   219	        # can't match), and accept only when there's exactly one
   220	        # candidate — multiple = concurrent race, refuse rather than
   221	        # guess wrong.
   222	        if [ -z "$sid" ] && [ -n "${watch_dir:-}" ] && [ "$agent_start" -gt 0 ]; then
   223	            candidates=()
   224	            while IFS= read -r f; do
   225	                [ -z "$f" ] && continue
   226	                bt=$(stat -f %B "$f" 2>/dev/null || echo 0)
   227	                [ "$bt" -ge "$agent_start" ] && candidates+=("$f")
   228	            done < <(find "$watch_dir" "${find_args[@]}" 2>/dev/null)
   229	            if [ "${#candidates[@]}" -eq 1 ]; then
   230	                cand=$(extract_id "${candidates[0]}")
   231	                if [ -n "$cand" ]; then
   232	                    sid="$cand"
   233	                    matched_file="${candidates[0]}"
   234	                else
   235	                    nm_candidate="${candidates[0]}"
   236	                fi
   237	            fi
   238	        fi
   239	    else
   240	        # Legacy snapshot-diff path: pair-wrap didn't publish a pidfile
   241	        # (older binary). Behaves identically to pre-#000020 — first new
   242	        # file in the watch dir wins. Cross-tag race re-emerges here, but
   243	        # we'd rather capture in the common single-session case than fail
   244	        # silently. Rebuild pair-wrap to upgrade to the PID-bound path.
   245	        current=$(find "$watch_dir" "${find_args[@]}" 2>/dev/null | sort)
   246	        new=$(comm -13 <(printf '%s\n' "$legacy_existing") <(printf '%s\n' "$current"))
   247	        if [ -n "$new" ]; then
   248	            while IFS= read -r f; do
   249	                [ -z "$f" ] && continue
   250	                cand=$(extract_id "$f")
   251	                if [ -n "$cand" ]; then
   252	                    sid="$cand"
   253	                    matched_file="$f"
   254	                    break
   255	                fi
   256	                nm_candidate="$f"
   257	            done <<< "$new"
   258	        fi
   259	    fi
   260	
   261	    # Drift fingerprint: we found a file matching the agent's session-file
   262	    # pattern but extract_id couldn't pull an id out of it — likely the
   263	    # filename/format changed. Log once per session (the poll loop reruns).
   264	    if [ -z "$sid" ] && [ -n "$nm_candidate" ] && [ "$nm_logged" = 0 ]; then
   265	        adapt_log session-watch "$agent" 3 session-id near-miss \
   266	            "matched session file but no id extracted: $(basename "$nm_candidate")"
   267	        nm_logged=1
   268	    fi
   269	
   270	    if [ -n "$sid" ]; then
   271	        # Strip --resume <id> / `resume <id>` so saved args don't carry
   272	        # the resume binding into future relaunches — session_id below is
   273	        # the canonical store. Same shape as bin/pair's stripping; keep
   274	        # in sync.
   275	        stripped=()
   276	        n=${#args[@]}
   277	        i=0
   278	        if [ "$agent" = "codex" ] && [ $n -ge 2 ] \
   279	            && [ "${args[0]}" = "resume" ]; then
   280	            i=2
   281	        fi
   282	        while [ $i -lt $n ]; do
   283	            if [ "${args[$i]}" = "--resume" ]; then
   284	                i=$((i+2))
   285	            else
   286	                stripped+=("${args[$i]}")
   287	                i=$((i+1))
   288	            fi
   289	        done
   290	
   291	        tmp=$(mktemp "$out.XXXXXX") || exit 0
   292	        if jq -n \
   293	              --arg agent "$agent" \
   294	              --arg sid "$sid" \
   295	              '{ agent: $agent, args: $ARGS.positional, session_id: $sid }' \
   296	              --args -- ${stripped[@]+"${stripped[@]}"} > "$tmp"
   297	        then
   298	            mv "$tmp" "$out"
   299	            adapt_log session-watch "$agent" 3 session-id fired "session_id=$sid"
   300	        else
   301	            rm -f "$tmp"
   302	        fi
   303	        exit 0
   304	    fi
   305	    sleep 0.1
   306	done
   307	
   308	# Watched the full window without resolving an id: the session file never
   309	# appeared where/how we expect — the strongest drift signal for this aspect.
   310	adapt_log session-watch "$agent" 3 session-id fail "no session id within 60s deadline (agent=$agent)"
   311	exit 0

exec
/bin/zsh -lc 'go test ./cmd/internal/sessionwatch ./cmd/pair-session-watch -count=1' in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc 'make test-session-watch' in /Users/xianxu/workspace/pair
 succeeded in 478ms:
ok  	github.com/xianxu/pair/cmd/internal/sessionwatch	0.198s
ok  	github.com/xianxu/pair/cmd/pair-session-watch	0.348s

 succeeded in 629ms:
pair-session-watch stale pidfile test PASS

codex
```verdict
verdict: REWORK
confidence: high
```

The port is structurally solid: the pure/sessionwatch split is real, the shim preserves the old caller name, atlas was updated, and the basic fake process test passes. I would not ship this boundary yet because one matching-path regression can make the watcher time out even when a valid session file is present, and the plan’s Core Concepts table no longer precisely matches the implemented integration entity.

1. Strengths:
- `cmd/internal/sessionwatch/sessionwatch.go:60` keeps agent file matching and ID extraction pure and directly unit-tested.
- `cmd/internal/sessionwatch/run.go:24` cleanly injects process/filesystem/clock behavior behind `Runtime`, matching `ARCH-PURE`.
- `cmd/internal/sessionwatch/sessionwatch.go:116` uses `encoding/json`, avoiding shell string-built JSON.
- `Makefile.local:84` makes `test-session-watch` depend on the Go binary, so the shim test cannot accidentally run without the built command.
- Atlas updates cover the new Go-owned watcher surface.

2. Critical findings:
- `cmd/internal/sessionwatch/run.go:136` and `cmd/internal/sessionwatch/run.go:151`: a near-miss candidate stops discovery before later valid candidates are checked. The old shell saved `nm_candidate` but kept scanning; this code returns immediately on `result.NearMiss`, so one malformed `rollout-*.jsonl` or `.db` can shadow a valid file from a later PID/path/list entry until timeout. Fix by continuing after near-miss, remembering the first near-miss, and returning it only after all candidates are exhausted with no ID. Add tests for lsof and legacy fallback ordering where a bad match appears before a good match. This is a correctness drift and `ARCH-PURPOSE` issue.
- `workshop/plans/000078-go-stateful-shell-glue-plan.md:51`: Core Concepts says `WatcherRuntime` lives in `cmd/internal/sessionwatch/runtime.go`, but the injectable boundary is named `Runtime` and lives in `cmd/internal/sessionwatch/run.go:24`; `runtime.go` contains `OSRuntime`. Per the review contract, revise the plan table or code naming so the concept table is greppable and accurate.

3. Important findings:
- `cmd/pair-session-watch/main.go:21`: the old script defaulted `PAIR_TAG` from the positional tag before opening the adapt log; the new code only logs when `PAIR_TAG` is already set. Production via `bin/pair` likely still sets it, but standalone watcher runs lose `near-miss`/`fail` diagnostics. Either restore the fallback before `adapt.Open`, or document that standalone logging is intentionally dropped.

4. Minor findings:
- `cmd/internal/sessionwatch/runtime.go:146`: `trimCommandOutput` appears unused; remove it if `go vet` or future cleanup wants a tighter package.

5. Test coverage notes:
- I ran `go test ./cmd/internal/sessionwatch ./cmd/pair-session-watch -count=1` and `make test-session-watch`; both passed.
- Missing coverage: near-miss-before-valid ordering for lsof and legacy discovery, plus optional coverage for `PAIR_TAG` fallback logging.

6. Architectural notes:
- `ARCH-DRY`: pass. Agent matching, resume stripping, and config JSON are centralized.
- `ARCH-PURE`: pass for the main design. Pure helpers are tested without IO; process work is behind `Runtime`.
- `ARCH-PURPOSE`: flag. The near-miss early return under-delivers the watcher’s core purpose: reliably capturing the session ID when one is discoverable.

7. Plan revision recommendations:
- Add a `## Revisions` entry noting the Core Concepts correction: either rename `WatcherRuntime` to `Runtime` and move its location to `cmd/internal/sessionwatch/run.go`, or rename/move the implementation to match the plan.
- If retaining the new no-`PAIR_TAG` standalone behavior, add a revision explaining that telemetry now requires launcher-provided `PAIR_TAG`; otherwise restore the old fallback.
tokens used
68,775
```verdict
verdict: REWORK
confidence: high
```

The port is structurally solid: the pure/sessionwatch split is real, the shim preserves the old caller name, atlas was updated, and the basic fake process test passes. I would not ship this boundary yet because one matching-path regression can make the watcher time out even when a valid session file is present, and the plan’s Core Concepts table no longer precisely matches the implemented integration entity.

1. Strengths:
- `cmd/internal/sessionwatch/sessionwatch.go:60` keeps agent file matching and ID extraction pure and directly unit-tested.
- `cmd/internal/sessionwatch/run.go:24` cleanly injects process/filesystem/clock behavior behind `Runtime`, matching `ARCH-PURE`.
- `cmd/internal/sessionwatch/sessionwatch.go:116` uses `encoding/json`, avoiding shell string-built JSON.
- `Makefile.local:84` makes `test-session-watch` depend on the Go binary, so the shim test cannot accidentally run without the built command.
- Atlas updates cover the new Go-owned watcher surface.

2. Critical findings:
- `cmd/internal/sessionwatch/run.go:136` and `cmd/internal/sessionwatch/run.go:151`: a near-miss candidate stops discovery before later valid candidates are checked. The old shell saved `nm_candidate` but kept scanning; this code returns immediately on `result.NearMiss`, so one malformed `rollout-*.jsonl` or `.db` can shadow a valid file from a later PID/path/list entry until timeout. Fix by continuing after near-miss, remembering the first near-miss, and returning it only after all candidates are exhausted with no ID. Add tests for lsof and legacy fallback ordering where a bad match appears before a good match. This is a correctness drift and `ARCH-PURPOSE` issue.
- `workshop/plans/000078-go-stateful-shell-glue-plan.md:51`: Core Concepts says `WatcherRuntime` lives in `cmd/internal/sessionwatch/runtime.go`, but the injectable boundary is named `Runtime` and lives in `cmd/internal/sessionwatch/run.go:24`; `runtime.go` contains `OSRuntime`. Per the review contract, revise the plan table or code naming so the concept table is greppable and accurate.

3. Important findings:
- `cmd/pair-session-watch/main.go:21`: the old script defaulted `PAIR_TAG` from the positional tag before opening the adapt log; the new code only logs when `PAIR_TAG` is already set. Production via `bin/pair` likely still sets it, but standalone watcher runs lose `near-miss`/`fail` diagnostics. Either restore the fallback before `adapt.Open`, or document that standalone logging is intentionally dropped.

4. Minor findings:
- `cmd/internal/sessionwatch/runtime.go:146`: `trimCommandOutput` appears unused; remove it if `go vet` or future cleanup wants a tighter package.

5. Test coverage notes:
- I ran `go test ./cmd/internal/sessionwatch ./cmd/pair-session-watch -count=1` and `make test-session-watch`; both passed.
- Missing coverage: near-miss-before-valid ordering for lsof and legacy discovery, plus optional coverage for `PAIR_TAG` fallback logging.

6. Architectural notes:
- `ARCH-DRY`: pass. Agent matching, resume stripping, and config JSON are centralized.
- `ARCH-PURE`: pass for the main design. Pure helpers are tested without IO; process work is behind `Runtime`.
- `ARCH-PURPOSE`: flag. The near-miss early return under-delivers the watcher’s core purpose: reliably capturing the session ID when one is discoverable.

7. Plan revision recommendations:
- Add a `## Revisions` entry noting the Core Concepts correction: either rename `WatcherRuntime` to `Runtime` and move its location to `cmd/internal/sessionwatch/run.go`, or rename/move the implementation to match the plan.
- If retaining the new no-`PAIR_TAG` standalone behavior, add a revision explaining that telemetry now requires launcher-provided `PAIR_TAG`; otherwise restore the old fallback.
