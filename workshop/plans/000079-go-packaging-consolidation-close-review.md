# Boundary Review — pair#79 (whole-issue close)

| field | value |
|-------|-------|
| issue | 79 — pair Go packaging consolidation |
| repo | pair |
| issue file | workshop/issues/000079-go-packaging-consolidation.md |
| boundary | whole-issue close |
| milestone | — |
| window | 776ecce27bc3643bc5598df559c52a3bb5424efe..HEAD |
| command | sdlc close --issue 79 |
| reviewer | codex |
| timestamp | 2026-06-30T17:57:14-07:00 |
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
session id: 019f1b2c-0689-7963-aea3-5f82d45ae4d3
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **pair** repository.

- repository: pair   (root: /Users/xianxu/workspace/pair)
- issue:      pair#79   (file: workshop/issues/000079-go-packaging-consolidation.md)
- window:     Base: 776ecce27bc3643bc5598df559c52a3bb5424efe   Head: HEAD

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
diff --git a/.gitignore b/.gitignore
index 2670f85..18ff43e 100644
--- a/.gitignore
+++ b/.gitignore
@@ -14,7 +14,7 @@ __pycache__/
 # made them look disposable. See workshop/issues #64.)
 bin/*
 !bin/*.sh
-!bin/pair
+!bin/pair-shell
 !bin/pair-changelog-open
 !bin/pair-dev
 !bin/pair-help
diff --git a/Makefile.local b/Makefile.local
index 8afd085..d2383b8 100644
--- a/Makefile.local
+++ b/Makefile.local
@@ -1,7 +1,7 @@
 # Repo-specific Makefile targets.
 # This file is included by Makefile — add your own targets here.
 
-.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-session-watch pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
+.PHONY: build install pair pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-session-watch pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
 
 # Cross-cutting Go-binary targets. Every Go binary lives under cmd/<name>/
 # and ships with the `pair-` prefix so it doesn't collide with anything
@@ -26,8 +26,9 @@
 
 BIN_DIR     := bin
 INSTALL_DIR := $(HOME)/.local/bin
-GO_BINS     := pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-session-watch pair-go
-SHELL_BINS  := pair pair-dev
+GO_BINS     := pair pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-session-watch pair-go
+SHELL_BINS  := pair-dev
+PAIR_HOME_LDFLAGS := -X main.defaultPairHome=$(CURDIR)
 
 build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
 
@@ -43,6 +44,7 @@ install: build
 	done
 
 # Per-binary aliases — `make pair-scribe`, etc.
+pair:                   $(BIN_DIR)/pair
 pair-scribe:            $(BIN_DIR)/pair-scribe
 pair-scrollback-render: $(BIN_DIR)/pair-scrollback-render
 pair-wrap:              $(BIN_DIR)/pair-wrap
@@ -76,7 +78,7 @@ test-pair-title:
 # cmux workspace ownership claim (presence beats a stale owner flag), so a
 # launch/attach/restart reclaims its workspace title. See
 # tests/cmux-ownership-test.sh.
-test-cmux-ownership:
+test-cmux-ownership: $(BIN_DIR)/pair
 	bash tests/cmux-ownership-test.sh
 
 # Session id watcher must ignore stale pidfiles from prior launches and wait
@@ -187,9 +189,9 @@ test-adapt-schema:
 test-dev-rebuild:
 	bash tests/dev-rebuild-test.sh
 
-# pair-go launch handoff (#77): installed pair-go requires sibling pair. Local
-# installs link shell launchers back to the source tree so pair keeps its
-# PAIR_HOME asset resolution.
+# Go public entrypoint install layout (#79): installed pair is a regular Go
+# binary, pair-go launch shares its pair-shell handoff, and copied local
+# installs fall back to the build-time source root for adjacent assets.
 test-pair-go-install-layout:
 	bash tests/pair-go-install-layout-test.sh
 
@@ -201,7 +203,7 @@ test-emitter-health:
 # `pair continue` arg-parsing + session-name guard (#000054): tag not forced,
 # -- args forwarded, [agent] port, bare list / error paths, guard short-vs-long.
 # Drives the real bin/pair via its PAIR_DEBUG_ARGS probe. See tests/pair-continue-test.sh.
-test-continue:
+test-continue: $(BIN_DIR)/pair
 	bash tests/pair-continue-test.sh
 
 # changelog — end-to-end smoke for bin/pair-changelog-open (#53 M2): fakes the
@@ -256,11 +258,19 @@ $(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill
 $(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-continuation/git.go cmd/pair-continuation/main.go go.mod
 	go build -o $@ ./cmd/pair-continuation
 
-# pair-go — development dispatcher and Go-owned launch handoff.
-# `pair-go launch ...` execs sibling bin/pair for the current migration window;
+PAIR_GO_SRCS := cmd/pair-go/main.go cmd/internal/entrypoint/asset_root.go cmd/internal/entrypoint/launch.go cmd/internal/entrypoint/mode.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
+
+# pair — public Go-owned entrypoint. It execs bin/pair-shell for the current
+# shell-owned zellij lifecycle and uses the build-time defaultPairHome when a
+# copied local install has no sibling assets.
+$(BIN_DIR)/pair: $(PAIR_GO_SRCS)
+	go build -ldflags "$(PAIR_HOME_LDFLAGS)" -o $@ ./cmd/pair-go
+
+# pair-go — development dispatcher and explicit migration surface.
+# `pair-go launch ...` shares the same bin/pair-shell handoff as public pair;
 # helper routes stay in the dispatcher.
-$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/entrypoint/launch.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
-	go build -o $@ ./cmd/pair-go
+$(BIN_DIR)/pair-go: $(PAIR_GO_SRCS)
+	go build -ldflags "$(PAIR_HOME_LDFLAGS)" -o $@ ./cmd/pair-go
 
 
 # Tart VM targets (tart, tart-gui, tart-mount, tart-stop, tart-clean)
diff --git a/README.md b/README.md
index eb841b8..eb3af77 100644
--- a/README.md
+++ b/README.md
@@ -200,12 +200,13 @@ pair rename <old> <new>          # rename every tag-scoped file in
 pair -h, --help                  # show full help
 ```
 
-`pair-go launch ...` is the Go-owned migration entrypoint for testing the
-launcher path. It accepts the same arguments after `launch` that `pair` accepts
-directly, then hands off to the current `pair` launcher for one migration
-window. In a dev shell sourced from `../ariadne/construct/dev-aliases.sh`,
-`pair-go` rebuilds from `cmd/pair-go` automatically before running; no
-`pair-go-dev` command is needed.
+The installed `pair` command is Go-owned. For this migration window it resolves
+the adjacent Pair asset root and hands off to `bin/pair-shell`, the retained
+shell launcher that still owns the zellij lifecycle. `pair-go launch ...` remains
+the explicit development dispatcher path and accepts the same arguments after
+`launch` that `pair` accepts directly. In a dev shell sourced from
+`../ariadne/construct/dev-aliases.sh`, `pair` and `pair-go` rebuild from
+`cmd/pair-go` automatically before running; no `pair-go-dev` command is needed.
 
 Use `--` to separate pair's positional from agent flags. Without it, pair only takes `<agent>` as a positional and everything else is rejected.
 
diff --git a/atlas/architecture.md b/atlas/architecture.md
index 65456bd..17fce6a 100644
--- a/atlas/architecture.md
+++ b/atlas/architecture.md
@@ -13,8 +13,9 @@ contract for the Go packaging migration lives in
 [Go migration inventory](go-migration-inventory.md).
 
 ```
-bin/pair                     # entry point (launcher)
-bin/pair-go                  # opt-in Go dispatcher + launch handoff; public launcher remains bin/pair
+bin/pair                     # Go public entrypoint; execs bin/pair-shell during migration
+bin/pair-shell               # retained shell launcher: zellij lifecycle + prompt UI
+bin/pair-go                  # explicit Go dispatcher + launch handoff
 bin/clipboard-to-pane.sh     # read clipboard, hand off to nvim's PairPasteQuote
 bin/copy-on-select.sh        # invoked by zellij copy_command on mouse-up
 bin/pair-quit.sh             # invoked by Alt+x — marks + kills session
@@ -45,19 +46,26 @@ internal subcommands or dispatch modes behind that primary binary (`pair wrap`,
 `pair continuation`, `pair scribe`) instead of staying as independently managed
 installed commands forever.
 
-As of #77, `bin/pair-go` is the Go-owned launch entrypoint under test:
-`pair-go launch ...` resolves its own executable, finds sibling `bin/pair`, and
-execs it with the same argv/env that `pair` would have received directly. This
-makes `pair-go launch claude`, `pair-go launch resume <tag>`, `pair-go launch
-continue ...`, `pair-go launch list`, and `pair-go launch rename ...`
-meaningful dogfood commands without replacing the stable public `pair` command
-yet. A developer shell sourced from `../ariadne/construct/dev-aliases.sh`
-rebuilds `cmd/pair-go` automatically; no `pair-go-dev` command exists.
+As of #79, the public `bin/pair` command is a Go-built entrypoint from
+`cmd/pair-go`. Direct `pair ...` and explicit `pair-go launch ...` share one
+compatibility handoff: resolve the Pair asset root, then exec
+`<asset-root>/bin/pair-shell` with `pair`-compatible argv/env. Asset root
+resolution is ordered: explicit `PAIR_HOME`, executable sibling root, then the
+build-time `defaultPairHome` injected by Make/Homebrew for copied installs.
+Native `nvim/` and `zellij/` assets remain adjacent to that root.
+
+`pair-go` remains the development dispatcher for helper routes and explicit
+launch testing: `pair-go launch claude`, `pair-go launch resume <tag>`,
+`pair-go launch continue ...`, `pair-go launch list`, and
+`pair-go launch rename ...` all reach the same shell launcher as direct `pair`.
+A developer shell sourced from `../ariadne/construct/dev-aliases.sh` rebuilds
+`cmd/pair-go` automatically; no `pair-go-dev` command exists.
 
 The earlier #75 pure launcher core remains available as internal decision logic,
 but real zellij lifecycle, prompt/fzf UI, restart/quit cleanup, cmux ownership,
 dev rebuild, continuation, rename, config/session migration, and title-poller
-behavior remain shell-owned through `bin/pair` until later migration issues.
+behavior remain shell-owned through `bin/pair-shell` until later migration
+issues.
 
 As of #76, the same dispatcher also has the first implemented helper routes:
 `pair-go context <tag> <agent>` and `pair-go scrollback-render ...`. Both routes
@@ -78,9 +86,13 @@ keybindings, scrollback, changelog, continuation, and review flows still work.
 The detailed disposition table is maintained in
 [Go migration inventory](go-migration-inventory.md), not duplicated here.
 
-### `bin/pair` — launcher
+### `bin/pair` / `bin/pair-shell` — launcher
 
-Resolves `$PAIR_HOME` from its own real path (portable bash, no `readlink -f`), prepends `$PAIR_HOME/bin` to `$PATH` (idempotent across re-launches) so all helper scripts resolve by bare name in zellij configs and keybinds, parses argv — first positional is `$PAIR_AGENT` (default `claude`), everything after `--` is joined into `$PAIR_AGENT_ARGS`, extra positionals before `--` are an error with a usage hint, defaults `$PAIR_TAG` to the cwd basename (the create-flow prompt or `pair resume <tag>` overrides it), resolves `$PAIR_DATA_DIR` to `${XDG_DATA_HOME:-$HOME/.local/share}/pair`, runs a one-time migration of any old `~/scratch/pair-{draft,log}-*` files, and dispatches:
+`bin/pair` is the Go public entrypoint. It resolves the asset root, then execs
+`bin/pair-shell` with argv[0] presented as `pair`; the shell script below still
+owns the launcher lifecycle during this migration window.
+
+`bin/pair-shell` resolves `$PAIR_HOME` from its own real path (portable bash, no `readlink -f`), prepends `$PAIR_HOME/bin` to `$PATH` (idempotent across re-launches) so all helper scripts resolve by bare name in zellij configs and keybinds, parses argv — first positional is `$PAIR_AGENT` (default `claude`), everything after `--` is joined into `$PAIR_AGENT_ARGS`, extra positionals before `--` are an error with a usage hint, defaults `$PAIR_TAG` to the cwd basename (the create-flow prompt or `pair resume <tag>` overrides it), resolves `$PAIR_DATA_DIR` to `${XDG_DATA_HOME:-$HOME/.local/share}/pair`, runs a one-time migration of any old `~/scratch/pair-{draft,log}-*` files, and dispatches:
 
 A leading `pair resume <tag>` is recognized as a subcommand verb (alongside `list` / `help`): it skips both the picker and the name prompt, attaches if `pair-<tag>` already exists in any state, otherwise creates with that tag. When `resume` is in play, the agent is inferred from saved state on disk (`agent-<tag>` for live/recently-detached sessions; the agent embedded in the `config-<tag>-<agent>.json` filename otherwise) — so a single tag is enough to restart, regardless of which agent was originally paired with it. See "Tag-restart" below.
 
diff --git a/atlas/go-migration-inventory.md b/atlas/go-migration-inventory.md
index b03e89d..e095add 100644
--- a/atlas/go-migration-inventory.md
+++ b/atlas/go-migration-inventory.md
@@ -47,8 +47,8 @@ Priority is packaging impact first, then reliability/testability:
 
 | Artifact | Type | Callers | Runtime contract | Files/env | Disposition | Priority |
 |---|---|---|---|---|---|---|
-| `bin/pair` / `cmd/internal/launcher` / `cmd/internal/entrypoint` | Bash public launcher plus Go-owned launch handoff | user shell, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` (#77) resolves sibling `bin/pair` from the `pair-go` executable and execs it with `pair`-compatible argv/env. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves the compatibility handoff; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Go handoff entrypoint with `bin/pair` retained as stable public launcher and compat target through #77; real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
-| `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling `pair`. | Resolves symlinks; depends on `bin/pair` and `bin/lib/dev-rebuild.sh`. | compat-shim, likely stays as developer wrapper until Go entrypoint has dev mode | P1 |
+| `bin/pair` / `bin/pair-shell` / `cmd/internal/launcher` / `cmd/internal/entrypoint` | Go public entrypoint plus retained shell launcher | user shell, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` is generated from `cmd/pair-go` and resolves `PAIR_HOME` / sibling root / build-time `defaultPairHome`, then execs `<asset-root>/bin/pair-shell` with `pair`-compatible argv/env. `bin/pair-shell` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` shares the same compatibility handoff. | `bin/pair-shell` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves invocation mode, asset root, and compatibility request; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Public entrypoint is Go-owned as of #79; `bin/pair-shell` is retained because real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
+| `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling Go-built `pair`. | Resolves symlinks; depends on generated `bin/pair`, retained `bin/pair-shell`, and `bin/lib/dev-rebuild.sh`. | retained dev wrapper so developer launches exercise the public Go entrypoint | P1 |
 | `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
 | `zellij/layouts/main.kdl` | zellij native asset | `bin/pair` via `zellij --new-session-with-layout` | Defines agent and draft panes; shell expands Pair env at pane start. | Calls `pair-wrap`; calls `nvim -u "$PAIR_HOME/nvim/init.lua"`; writes `pane-<tag>-<agent>.json`; writes draft nvim pid file. | native-asset, packaged adjacent/embedded | P0 |
 | `zellij/config.kdl` | zellij native asset | zellij session config from `bin/pair` | Global keybinds, copy command, scrollback buffer, pane frames. | Calls `copy-on-select.sh`, `pair-help`, `pair-scrollback-open`, `pair-changelog-open`; routes quit/restart/compact through nvim functions. | native-asset, packaged adjacent/embedded | P0 |
@@ -114,10 +114,14 @@ Nvim shell-outs and binary dependencies:
 
 Build/install callers:
 
-- `make build` builds `GO_BINS` into `bin/`.
-- `make install` copies `GO_BINS` to `~/.local/bin` and symlinks `SHELL_BINS`
-  (`pair`, `pair-dev`) beside them so installed `pair-go launch ...` can resolve
-  sibling `pair`.
+- `make build` builds `GO_BINS` into `bin/`; `pair` and `pair-go` are both built
+  from `cmd/pair-go` with `defaultPairHome=$(CURDIR)`.
+- `make install` copies `GO_BINS` to `~/.local/bin` and symlinks only retained
+  shell wrappers such as `pair-dev`. Installed `pair` is a regular Go binary;
+  if it has no sibling assets, it falls back to the build-time source root.
+- Homebrew installs `bin/`, `nvim/`, and `zellij/` under `libexec`, then builds
+  Go `pair`, `pair-go`, and required runtime helpers into `libexec/bin` with
+  `defaultPairHome=#{libexec}`.
 - `pair-dev` relies on `make build`, then zellij's PATH lookup resolves fresh
   repo `bin/` binaries.
 
@@ -132,13 +136,14 @@ Build/install callers:
   binary names remain live for shell/Lua callers. `pair slug` remains a later
   candidate.
 - #77 made `pair-go launch ...` a meaningful Go-owned compatibility handoff to
-  `bin/pair`, with argv/env preserved and missing-launcher diagnostics. The
-  stable public `pair` script remains unchanged for this migration window.
+  `bin/pair`, with argv/env preserved and missing-launcher diagnostics.
 - #78 ported the session-id watcher to `cmd/pair-session-watch` with
   `bin/pair-session-watch.sh` retained as a shim. `pair-title.sh` remains the
   next stateful shell candidate because it owns a separate UI title-poller
   surface.
-- #79 owns whether `nvim/` and `zellij/` are embedded or installed adjacent.
+- #79 made public `pair` a Go-built entrypoint, renamed the shell launcher to
+  `bin/pair-shell`, and chose adjacent `nvim/` / `zellij/` assets for local and
+  Homebrew installs.
 
 ## Coverage Ledger
 
diff --git a/bin/pair b/bin/pair-shell
similarity index 100%
rename from bin/pair
rename to bin/pair-shell
diff --git a/cmd/internal/entrypoint/asset_root.go b/cmd/internal/entrypoint/asset_root.go
new file mode 100644
index 0000000..ebdf6df
--- /dev/null
+++ b/cmd/internal/entrypoint/asset_root.go
@@ -0,0 +1,73 @@
+package entrypoint
+
+import (
+	"fmt"
+	"path/filepath"
+	"strings"
+)
+
+type AssetRootInput struct {
+	PairHome        string
+	Executable      string
+	DefaultPairHome string
+	PairShellExists func(root string) bool
+}
+
+type AssetRoot struct {
+	Root      string
+	ShellPath string
+	Source    string
+}
+
+func ResolveAssetRoot(input AssetRootInput) (AssetRoot, error) {
+	exists := input.PairShellExists
+	if exists == nil {
+		exists = func(string) bool { return false }
+	}
+
+	candidates := make([]assetRootCandidate, 0, 3)
+	if input.PairHome != "" {
+		candidates = append(candidates, assetRootCandidate{root: input.PairHome, source: "PAIR_HOME"})
+	}
+	if input.Executable != "" {
+		candidates = append(candidates, assetRootCandidate{
+			root:   filepath.Dir(filepath.Dir(input.Executable)),
+			source: "executable sibling",
+		})
+	}
+	if input.DefaultPairHome != "" {
+		candidates = append(candidates, assetRootCandidate{root: input.DefaultPairHome, source: "defaultPairHome"})
+	}
+
+	seen := map[string]bool{}
+	checked := make([]string, 0, len(candidates))
+	for _, candidate := range candidates {
+		root := filepath.Clean(candidate.root)
+		if root == "." || seen[root] {
+			continue
+		}
+		seen[root] = true
+		checked = append(checked, root)
+		if exists(root) {
+			return AssetRoot{
+				Root:      root,
+				ShellPath: PairShellPath(root),
+				Source:    candidate.source,
+			}, nil
+		}
+	}
+
+	if len(checked) == 0 {
+		checked = append(checked, "<none>")
+	}
+	return AssetRoot{}, fmt.Errorf("pair-shell not found; set PAIR_HOME to a Pair checkout/install root containing bin/pair-shell (checked: %s)", strings.Join(checked, ", "))
+}
+
+func PairShellPath(root string) string {
+	return filepath.Join(root, "bin", "pair-shell")
+}
+
+type assetRootCandidate struct {
+	root   string
+	source string
+}
diff --git a/cmd/internal/entrypoint/asset_root_test.go b/cmd/internal/entrypoint/asset_root_test.go
new file mode 100644
index 0000000..6c6843d
--- /dev/null
+++ b/cmd/internal/entrypoint/asset_root_test.go
@@ -0,0 +1,84 @@
+package entrypoint
+
+import (
+	"strings"
+	"testing"
+)
+
+func TestResolveAssetRootPrefersPairHome(t *testing.T) {
+	root, err := ResolveAssetRoot(AssetRootInput{
+		PairHome:        "/env/root",
+		Executable:      "/repo/bin/pair",
+		DefaultPairHome: "/default/root",
+		PairShellExists: existingRoots("/env/root", "/repo", "/default/root"),
+	})
+	if err != nil {
+		t.Fatalf("ResolveAssetRoot error = %v", err)
+	}
+	if root.Root != "/env/root" {
+		t.Fatalf("Root = %q, want /env/root", root.Root)
+	}
+	if root.ShellPath != "/env/root/bin/pair-shell" {
+		t.Fatalf("ShellPath = %q, want /env/root/bin/pair-shell", root.ShellPath)
+	}
+}
+
+func TestResolveAssetRootUsesExecutableSiblingRoot(t *testing.T) {
+	root, err := ResolveAssetRoot(AssetRootInput{
+		Executable:      "/repo/bin/pair",
+		DefaultPairHome: "/default/root",
+		PairShellExists: existingRoots("/repo", "/default/root"),
+	})
+	if err != nil {
+		t.Fatalf("ResolveAssetRoot error = %v", err)
+	}
+	if root.Root != "/repo" {
+		t.Fatalf("Root = %q, want /repo", root.Root)
+	}
+	if root.ShellPath != "/repo/bin/pair-shell" {
+		t.Fatalf("ShellPath = %q, want /repo/bin/pair-shell", root.ShellPath)
+	}
+}
+
+func TestResolveAssetRootFallsBackToDefaultPairHome(t *testing.T) {
+	root, err := ResolveAssetRoot(AssetRootInput{
+		Executable:      "/home/me/.local/bin/pair",
+		DefaultPairHome: "/repo",
+		PairShellExists: existingRoots("/repo"),
+	})
+	if err != nil {
+		t.Fatalf("ResolveAssetRoot error = %v", err)
+	}
+	if root.Root != "/repo" {
+		t.Fatalf("Root = %q, want /repo", root.Root)
+	}
+	if root.ShellPath != "/repo/bin/pair-shell" {
+		t.Fatalf("ShellPath = %q, want /repo/bin/pair-shell", root.ShellPath)
+	}
+}
+
+func TestResolveAssetRootReportsMissingPairShellAndPairHome(t *testing.T) {
+	_, err := ResolveAssetRoot(AssetRootInput{
+		Executable:      "/home/me/.local/bin/pair",
+		DefaultPairHome: "/repo",
+		PairShellExists: existingRoots(),
+	})
+	if err == nil {
+		t.Fatal("ResolveAssetRoot error = nil, want missing-root error")
+	}
+	for _, want := range []string{"pair-shell", "PAIR_HOME", "/home/me/.local", "/repo"} {
+		if !strings.Contains(err.Error(), want) {
+			t.Fatalf("error missing %q:\n%v", want, err)
+		}
+	}
+}
+
+func existingRoots(roots ...string) func(string) bool {
+	set := make(map[string]bool, len(roots))
+	for _, root := range roots {
+		set[root] = true
+	}
+	return func(root string) bool {
+		return set[root]
+	}
+}
diff --git a/cmd/internal/entrypoint/launch.go b/cmd/internal/entrypoint/launch.go
index 1a40970..04e97f8 100644
--- a/cmd/internal/entrypoint/launch.go
+++ b/cmd/internal/entrypoint/launch.go
@@ -1,20 +1,18 @@
 package entrypoint
 
-import "path/filepath"
-
-// LegacyLaunchRequest describes the current compatibility handoff from
-// pair-go launch to the shell-owned pair launcher.
+// LegacyLaunchRequest describes the current compatibility handoff from the Go
+// entrypoint to the shell-owned pair launcher.
 type LegacyLaunchRequest struct {
 	Path string
 	Argv []string
 }
 
-func ResolveLegacyLaunch(executable string, launchArgs []string) LegacyLaunchRequest {
+func ResolveLegacyLaunch(root AssetRoot, launchArgs []string) LegacyLaunchRequest {
 	argv := make([]string, 0, len(launchArgs)+1)
 	argv = append(argv, "pair")
 	argv = append(argv, launchArgs...)
 	return LegacyLaunchRequest{
-		Path: filepath.Join(filepath.Dir(executable), "pair"),
+		Path: root.ShellPath,
 		Argv: argv,
 	}
 }
diff --git a/cmd/internal/entrypoint/launch_test.go b/cmd/internal/entrypoint/launch_test.go
index cdeec2c..c6a50cc 100644
--- a/cmd/internal/entrypoint/launch_test.go
+++ b/cmd/internal/entrypoint/launch_test.go
@@ -6,9 +6,9 @@ import (
 )
 
 func TestResolveLegacyLaunchDropsLaunchVerb(t *testing.T) {
-	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"claude", "--", "--resume"})
-	if req.Path != "/repo/bin/pair" {
-		t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
+	req := ResolveLegacyLaunch(AssetRoot{Root: "/repo", ShellPath: "/repo/bin/pair-shell"}, []string{"claude", "--", "--resume"})
+	if req.Path != "/repo/bin/pair-shell" {
+		t.Fatalf("Path = %q, want /repo/bin/pair-shell", req.Path)
 	}
 	want := []string{"pair", "claude", "--", "--resume"}
 	if !reflect.DeepEqual(req.Argv, want) {
@@ -17,9 +17,9 @@ func TestResolveLegacyLaunchDropsLaunchVerb(t *testing.T) {
 }
 
 func TestResolveLegacyLaunchPreservesSubcommands(t *testing.T) {
-	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"resume", "demo"})
-	if req.Path != "/repo/bin/pair" {
-		t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
+	req := ResolveLegacyLaunch(AssetRoot{Root: "/repo", ShellPath: "/repo/bin/pair-shell"}, []string{"resume", "demo"})
+	if req.Path != "/repo/bin/pair-shell" {
+		t.Fatalf("Path = %q, want /repo/bin/pair-shell", req.Path)
 	}
 	want := []string{"pair", "resume", "demo"}
 	if !reflect.DeepEqual(req.Argv, want) {
diff --git a/cmd/internal/entrypoint/mode.go b/cmd/internal/entrypoint/mode.go
new file mode 100644
index 0000000..23942a2
--- /dev/null
+++ b/cmd/internal/entrypoint/mode.go
@@ -0,0 +1,21 @@
+package entrypoint
+
+import "path/filepath"
+
+type EntrypointMode int
+
+const (
+	ModeDispatch EntrypointMode = iota
+	ModePublicPair
+	ModePairGoLaunch
+)
+
+func ClassifyInvocation(executable string, args []string) EntrypointMode {
+	if filepath.Base(executable) == "pair" {
+		return ModePublicPair
+	}
+	if len(args) > 0 && args[0] == "launch" {
+		return ModePairGoLaunch
+	}
+	return ModeDispatch
+}
diff --git a/cmd/pair-go/main.go b/cmd/pair-go/main.go
index 760f876..6834be7 100644
--- a/cmd/pair-go/main.go
+++ b/cmd/pair-go/main.go
@@ -6,12 +6,15 @@ import (
 	"fmt"
 	"io"
 	"os"
+	"path/filepath"
 	"syscall"
 
 	"github.com/xianxu/pair/cmd/internal/dispatcher"
 	"github.com/xianxu/pair/cmd/internal/entrypoint"
 )
 
+var defaultPairHome string
+
 func main() {
 	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
 }
@@ -22,31 +25,50 @@ func run(args []string, stdout, stderr io.Writer) int {
 
 type legacyRuntime interface {
 	Executable() (string, error)
+	PairHome() string
+	DefaultPairHome() string
 	Stat(path string) error
 	Environ() []string
-	Exec(path string, argv []string, env []string) int
+	Exec(label string, path string, argv []string, env []string) int
 }
 
 func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
-	if len(args) > 0 && args[0] == "launch" {
-		return runLegacyLaunch(args[1:], stderr, rt)
+	exe, err := rt.Executable()
+	if err != nil {
+		if len(args) > 0 && args[0] == "launch" {
+			_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
+			return 1
+		}
+		res := dispatcher.Dispatch(args)
+		return writeResult(res, stdout, stderr)
+	}
+
+	switch entrypoint.ClassifyInvocation(exe, args) {
+	case entrypoint.ModePublicPair:
+		return runLegacyLaunch("pair", exe, args, stderr, rt)
+	case entrypoint.ModePairGoLaunch:
+		return runLegacyLaunch("pair-go launch", exe, args[1:], stderr, rt)
+	default:
+		res := dispatcher.Dispatch(args)
+		return writeResult(res, stdout, stderr)
 	}
-	res := dispatcher.Dispatch(args)
-	return writeResult(res, stdout, stderr)
 }
 
-func runLegacyLaunch(args []string, stderr io.Writer, rt legacyRuntime) int {
-	exe, err := rt.Executable()
+func runLegacyLaunch(label string, executable string, args []string, stderr io.Writer, rt legacyRuntime) int {
+	root, err := entrypoint.ResolveAssetRoot(entrypoint.AssetRootInput{
+		PairHome:        rt.PairHome(),
+		Executable:      executable,
+		DefaultPairHome: rt.DefaultPairHome(),
+		PairShellExists: func(root string) bool {
+			return rt.Stat(entrypoint.PairShellPath(root)) == nil
+		},
+	})
 	if err != nil {
-		_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
+		_, _ = fmt.Fprintf(stderr, "%s: %v; run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", label, err)
 		return 1
 	}
-	req := entrypoint.ResolveLegacyLaunch(exe, args)
-	if err := rt.Stat(req.Path); err != nil {
-		_, _ = fmt.Fprintf(stderr, "pair-go launch: pair launcher not found at %s (%v); run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", req.Path, err)
-		return 1
-	}
-	return rt.Exec(req.Path, req.Argv, rt.Environ())
+	req := entrypoint.ResolveLegacyLaunch(root, args)
+	return rt.Exec(label, req.Path, req.Argv, rt.Environ())
 }
 
 type osLegacyRuntime struct{}
@@ -55,7 +77,16 @@ func (osLegacyRuntime) Executable() (string, error) {
 	return os.Executable()
 }
 
+func (osLegacyRuntime) PairHome() string {
+	return os.Getenv("PAIR_HOME")
+}
+
+func (osLegacyRuntime) DefaultPairHome() string {
+	return defaultPairHome
+}
+
 func (osLegacyRuntime) Stat(path string) error {
+	path = filepath.Clean(path)
 	info, err := os.Stat(path)
 	if err != nil {
 		return err
@@ -70,9 +101,9 @@ func (osLegacyRuntime) Environ() []string {
 	return os.Environ()
 }
 
-func (osLegacyRuntime) Exec(path string, argv []string, env []string) int {
+func (osLegacyRuntime) Exec(label string, path string, argv []string, env []string) int {
 	if err := syscall.Exec(path, argv, env); err != nil {
-		_, _ = fmt.Fprintf(os.Stderr, "pair-go launch: exec %s failed: %v\n", path, err)
+		_, _ = fmt.Fprintf(os.Stderr, "%s: exec %s failed: %v\n", label, path, err)
 		return 1
 	}
 	return 0
diff --git a/cmd/pair-go/main_test.go b/cmd/pair-go/main_test.go
index d137a6b..75dd2a7 100644
--- a/cmd/pair-go/main_test.go
+++ b/cmd/pair-go/main_test.go
@@ -40,6 +40,7 @@ func TestRunWritesStderrAndReturnsDispatcherCode(t *testing.T) {
 func TestRunLaunchHelp(t *testing.T) {
 	rt := &fakeLegacyRuntime{
 		executable: "/repo/bin/pair-go",
+		roots:      map[string]bool{"/repo": true},
 	}
 	var stdout, stderr bytes.Buffer
 	code := runWithLegacyRuntime([]string{"launch", "--help"}, &stdout, &stderr, rt)
@@ -52,8 +53,11 @@ func TestRunLaunchHelp(t *testing.T) {
 	if stderr.String() != "" {
 		t.Fatalf("stderr = %q, want empty", stderr.String())
 	}
-	if rt.execPath != "/repo/bin/pair" {
-		t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
+	if rt.execPath != "/repo/bin/pair-shell" {
+		t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
+	}
+	if rt.execLabel != "pair-go launch" {
+		t.Fatalf("execLabel = %q, want pair-go launch", rt.execLabel)
 	}
 	wantArgv := []string{"pair", "--help"}
 	if !reflect.DeepEqual(rt.execArgv, wantArgv) {
@@ -65,6 +69,7 @@ func TestRunLaunchExecsLegacyPairWithArgvAndEnv(t *testing.T) {
 	t.Setenv("PAIR_TEST_ENV", "kept")
 	rt := &fakeLegacyRuntime{
 		executable: "/repo/bin/pair-go",
+		roots:      map[string]bool{"/repo": true},
 		execCode:   42,
 	}
 
@@ -80,8 +85,11 @@ func TestRunLaunchExecsLegacyPairWithArgvAndEnv(t *testing.T) {
 	if stderr.String() != "" {
 		t.Fatalf("stderr = %q, want empty", stderr.String())
 	}
-	if rt.execPath != "/repo/bin/pair" {
-		t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
+	if rt.execPath != "/repo/bin/pair-shell" {
+		t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
+	}
+	if rt.execLabel != "pair-go launch" {
+		t.Fatalf("execLabel = %q, want pair-go launch", rt.execLabel)
 	}
 	wantArgv := []string{"pair", "claude", "--", "--resume"}
 	if !reflect.DeepEqual(rt.execArgv, wantArgv) {
@@ -95,7 +103,6 @@ func TestRunLaunchExecsLegacyPairWithArgvAndEnv(t *testing.T) {
 func TestRunLaunchReportsMissingLegacyPair(t *testing.T) {
 	rt := &fakeLegacyRuntime{
 		executable: "/repo/bin/pair-go",
-		statErr:    os.ErrNotExist,
 	}
 
 	var stdout, stderr bytes.Buffer
@@ -107,7 +114,7 @@ func TestRunLaunchReportsMissingLegacyPair(t *testing.T) {
 	if stdout.String() != "" {
 		t.Fatalf("stdout = %q, want empty", stdout.String())
 	}
-	for _, want := range []string{"pair-go launch", "/repo/bin/pair", "make build", "make install", "dev-aliases.sh"} {
+	for _, want := range []string{"pair-go launch", "pair-shell", "PAIR_HOME", "/repo", "make build", "make install", "dev-aliases.sh"} {
 		if !strings.Contains(stderr.String(), want) {
 			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
 		}
@@ -117,14 +124,96 @@ func TestRunLaunchReportsMissingLegacyPair(t *testing.T) {
 	}
 }
 
+func TestRunDirectPairExecsLegacyShellWithAllArgs(t *testing.T) {
+	rt := &fakeLegacyRuntime{
+		executable: "/repo/bin/pair",
+		roots:      map[string]bool{"/repo": true},
+		execCode:   7,
+	}
+
+	var stdout, stderr bytes.Buffer
+	code := runWithLegacyRuntime([]string{"claude", "--", "--resume"}, &stdout, &stderr, rt)
+
+	if code != 7 {
+		t.Fatalf("code = %d, want 7", code)
+	}
+	if stdout.String() != "" {
+		t.Fatalf("stdout = %q, want empty", stdout.String())
+	}
+	if stderr.String() != "" {
+		t.Fatalf("stderr = %q, want empty", stderr.String())
+	}
+	if rt.execPath != "/repo/bin/pair-shell" {
+		t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
+	}
+	if rt.execLabel != "pair" {
+		t.Fatalf("execLabel = %q, want pair", rt.execLabel)
+	}
+	wantArgv := []string{"pair", "claude", "--", "--resume"}
+	if !reflect.DeepEqual(rt.execArgv, wantArgv) {
+		t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
+	}
+}
+
+func TestRunDirectPairFallsBackToDefaultPairHome(t *testing.T) {
+	rt := &fakeLegacyRuntime{
+		executable:      "/home/me/.local/bin/pair",
+		defaultPairHome: "/repo",
+		roots:           map[string]bool{"/repo": true},
+	}
+
+	var stdout, stderr bytes.Buffer
+	code := runWithLegacyRuntime([]string{"--help"}, &stdout, &stderr, rt)
+
+	if code != 0 {
+		t.Fatalf("code = %d, want 0", code)
+	}
+	if stderr.String() != "" {
+		t.Fatalf("stderr = %q, want empty", stderr.String())
+	}
+	if rt.execPath != "/repo/bin/pair-shell" {
+		t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
+	}
+}
+
+func TestRunPairGoHelperDoesNotProbeOrExecShellLauncher(t *testing.T) {
+	rt := &fakeLegacyRuntime{
+		executable: "/repo/bin/pair-go",
+	}
+
+	var stdout, stderr bytes.Buffer
+	code := runWithLegacyRuntime([]string{"help"}, &stdout, &stderr, rt)
+
+	if code != 0 {
+		t.Fatalf("code = %d, want 0", code)
+	}
+	if rt.statCalls != 0 {
+		t.Fatalf("statCalls = %d, want 0", rt.statCalls)
+	}
+	if rt.execPath != "" {
+		t.Fatalf("execPath = %q, want empty", rt.execPath)
+	}
+	if !strings.Contains(stdout.String(), "Usage: pair-go <command> [args]") {
+		t.Fatalf("stdout missing usage:\n%s", stdout.String())
+	}
+	if stderr.String() != "" {
+		t.Fatalf("stderr = %q, want empty", stderr.String())
+	}
+}
+
 type fakeLegacyRuntime struct {
-	executable string
-	statErr    error
-	execCode   int
+	executable      string
+	pairHome        string
+	defaultPairHome string
+	roots           map[string]bool
+	statErr         error
+	execCode        int
+	statCalls       int
 
-	execPath string
-	execArgv []string
-	execEnv  []string
+	execPath  string
+	execLabel string
+	execArgv  []string
+	execEnv   []string
 }
 
 func (f *fakeLegacyRuntime) Executable() (string, error) {
@@ -134,15 +223,34 @@ func (f *fakeLegacyRuntime) Executable() (string, error) {
 	return f.executable, nil
 }
 
-func (f *fakeLegacyRuntime) Stat(_ string) error {
-	return f.statErr
+func (f *fakeLegacyRuntime) PairHome() string {
+	return f.pairHome
+}
+
+func (f *fakeLegacyRuntime) DefaultPairHome() string {
+	return f.defaultPairHome
+}
+
+func (f *fakeLegacyRuntime) Stat(path string) error {
+	f.statCalls++
+	if f.statErr != nil {
+		return f.statErr
+	}
+	if strings.HasSuffix(path, "/bin/pair-shell") && f.roots != nil {
+		root := strings.TrimSuffix(path, "/bin/pair-shell")
+		if f.roots[root] {
+			return nil
+		}
+	}
+	return os.ErrNotExist
 }
 
 func (f *fakeLegacyRuntime) Environ() []string {
 	return os.Environ()
 }
 
-func (f *fakeLegacyRuntime) Exec(path string, argv []string, env []string) int {
+func (f *fakeLegacyRuntime) Exec(label string, path string, argv []string, env []string) int {
+	f.execLabel = label
 	f.execPath = path
 	f.execArgv = append([]string(nil), argv...)
 	f.execEnv = append([]string(nil), env...)
diff --git a/tests/pair-go-install-layout-test.sh b/tests/pair-go-install-layout-test.sh
index 086cb5b..d276cfc 100644
--- a/tests/pair-go-install-layout-test.sh
+++ b/tests/pair-go-install-layout-test.sh
@@ -7,12 +7,29 @@ trap 'rm -rf "$tmp_home"' EXIT
 gomodcache="$(go env GOMODCACHE)"
 gocache="$(go env GOCACHE)"
 
+install_bin="$tmp_home/.local/bin"
+old_bin="$tmp_home/old-bin"
+mkdir -p "$install_bin" "$old_bin"
+printf '#!/usr/bin/env bash\nprintf old-pair-shell\\n\n' > "$old_bin/pair"
+chmod +x "$old_bin/pair"
+ln -s "$old_bin/pair" "$install_bin/pair"
+
 HOME="$tmp_home" GOMODCACHE="$gomodcache" GOCACHE="$gocache" make -C "$repo_root" install >/dev/null
 
-install_bin="$tmp_home/.local/bin"
+test -x "$install_bin/pair"
+test ! -L "$install_bin/pair"
 test -x "$install_bin/pair-go"
-test -L "$install_bin/pair"
 test -L "$install_bin/pair-dev"
+test -x "$repo_root/bin/pair-shell"
+
+out="$("$install_bin/pair" --help)"
+case "$out" in
+    pair\ —*) ;;
+    *)
+        printf 'pair --help did not reach pair help; first bytes:\n%s\n' "$out" >&2
+        exit 1
+        ;;
+esac
 
 out="$("$install_bin/pair-go" launch --help)"
 case "$out" in
@@ -23,4 +40,13 @@ case "$out" in
         ;;
 esac
 
+out="$(PAIR_HOME="$repo_root" "$install_bin/pair" --help)"
+case "$out" in
+    pair\ —*) ;;
+    *)
+        printf 'PAIR_HOME pair --help did not reach pair help; first bytes:\n%s\n' "$out" >&2
+        exit 1
+        ;;
+esac
+
 printf 'pair-go install layout test passed\n'
diff --git a/workshop/plans/000079-go-packaging-consolidation-plan.md b/workshop/plans/000079-go-packaging-consolidation-plan.md
index d21b2b0..136b707 100644
--- a/workshop/plans/000079-go-packaging-consolidation-plan.md
+++ b/workshop/plans/000079-go-packaging-consolidation-plan.md
@@ -18,7 +18,7 @@
 |------|----------|--------|
 | `EntrypointMode` | `cmd/internal/entrypoint/mode.go` | new |
 | `AssetRoot` | `cmd/internal/entrypoint/asset_root.go` | new |
-| `LegacyPairRequest` | `cmd/internal/entrypoint/launch.go` | modified |
+| `LegacyLaunchRequest` | `cmd/internal/entrypoint/launch.go` | modified |
 
 **EntrypointMode** — Determines whether one executable invocation should behave as public `pair` or development `pair-go`.
 - **Relationships:** 1:1 with the executable basename; `cmd/pair-go/main.go` owns the argv/env IO and calls this pure classifier.
@@ -26,11 +26,11 @@
 - **Future extensions:** If the shell launcher is later fully ported, this mode decision becomes the dispatch point for native launch instead of compatibility exec.
 
 **AssetRoot** — Pure policy for choosing the root that owns adjacent runtime assets (`bin/pair-shell`, `nvim/`, `zellij/`).
-- **Relationships:** N:1 from local source build, local copied install, and Homebrew `libexec` install into one root decision; `LegacyPairRequest` consumes the resolved root.
+- **Relationships:** N:1 from local source build, local copied install, and Homebrew `libexec` install into one root decision; `LegacyLaunchRequest` consumes the resolved root.
 - **DRY rationale:** Prevents local install and Homebrew install from inventing separate path rules for the same asset layout (ARCH-DRY).
 - **Future extensions:** Can add an extracted-embedded asset dir later without changing launcher request construction.
 
-**LegacyPairRequest** — Describes the compatibility exec into the shell launcher.
+**LegacyLaunchRequest** — Describes the compatibility exec into the shell launcher.
 - **Relationships:** N:1 from `pair` direct mode and `pair-go launch` mode into one request builder; each mode only changes display/diagnostic wording and argv shape. Carries the selected `AssetRoot` and computes `<asset-root>/bin/pair-shell`.
 - **DRY rationale:** Keeps legacy shell handoff rules single-sourced while the actual zellij lifecycle remains shell-owned in this issue (ARCH-PURE).
 - **Future extensions:** Can be deleted once shell launch is replaced by native Go launch.
@@ -76,17 +76,17 @@
 - Create: `cmd/internal/entrypoint/mode.go`
 - Modify: `cmd/pair-go/main_test.go`
 
-- [ ] Add tests showing executable basename `pair` resolves to direct public launcher mode.
-- [ ] Add tests showing executable basename `pair-go` with `launch` still resolves to explicit launch handoff.
-- [ ] Add tests showing `pair-go` helper routes still dispatch without touching the shell launcher.
-- [ ] Add pure tests for asset-root resolution:
+- [x] Add tests showing executable basename `pair` resolves to direct public launcher mode.
+- [x] Add tests showing executable basename `pair-go` with `launch` still resolves to explicit launch handoff.
+- [x] Add tests showing `pair-go` helper routes still dispatch without touching the shell launcher.
+- [x] Add pure tests for asset-root resolution:
   - `PAIR_HOME=/repo` wins when `/repo/bin/pair-shell` exists.
   - executable `/repo/bin/pair` resolves sibling root `/repo` when `/repo/bin/pair-shell` exists.
   - copied executable `/home/me/.local/bin/pair` falls back to build-time default root `/repo` when sibling shell is absent and `/repo/bin/pair-shell` exists.
   - missing sibling and missing build-time root produces a diagnostic naming `pair-shell` and `PAIR_HOME`.
-- [ ] Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
-- [ ] Implement `EntrypointMode`, `AssetRoot`, and shared legacy request construction.
-- [ ] Re-run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
+- [x] Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
+- [x] Implement `EntrypointMode`, `AssetRoot`, and shared legacy request construction.
+- [x] Re-run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
 
 ### Task 2: Move shell launcher behind an internal compatibility name
 
@@ -97,13 +97,13 @@
 - Modify: `cmd/pair-go/main.go`
 - Modify: `bin/pair-dev`
 
-- [ ] Move the existing Bash launcher body to `bin/pair-shell`.
-- [ ] Update `.gitignore`: remove the `!bin/pair` tracked-script exception and add `!bin/pair-shell`; `bin/pair` stays ignored as generated Go build output.
-- [ ] Update Go direct `pair` mode to exec sibling `pair-shell` with argv[0] presented as `pair`.
-- [ ] Update `pair-go launch ...` to exec sibling `pair-shell` with the same argv compatibility as before.
-- [ ] Update `pair-dev` to export `PAIR_DEV=1` and exec sibling `pair` (the Go binary), not `pair-shell`, so dev mode exercises the public entrypoint.
-- [ ] Run: `bin/pair-go launch --help` after build and confirm it reaches the launcher help.
-- [ ] Run: `bin/pair --help` after build and confirm it reaches the same launcher help.
+- [x] Move the existing Bash launcher body to `bin/pair-shell`.
+- [x] Update `.gitignore`: remove the `!bin/pair` tracked-script exception and add `!bin/pair-shell`; `bin/pair` stays ignored as generated Go build output.
+- [x] Update Go direct `pair` mode to exec sibling `pair-shell` with argv[0] presented as `pair`.
+- [x] Update `pair-go launch ...` to exec sibling `pair-shell` with the same argv compatibility as before.
+- [x] Update `pair-dev` to export `PAIR_DEV=1` and exec sibling `pair` (the Go binary), not `pair-shell`, so dev mode exercises the public entrypoint.
+- [x] Run: `bin/pair-go launch --help` after build and confirm it reaches the launcher help.
+- [x] Run: `bin/pair --help` after build and confirm it reaches the same launcher help.
 
 ## Chunk 2: Build And Install Layout
 
@@ -113,13 +113,13 @@
 - Modify: `Makefile.local`
 - Modify: `tests/pair-go-install-layout-test.sh`
 
-- [ ] Update `GO_BINS` so `pair` is a Go-built binary and `pair-go` remains built from the same package.
-- [ ] Remove `pair` from `SHELL_BINS`; keep or explicitly drop `pair-dev` based on install behavior.
-- [ ] Add a specific `$(BIN_DIR)/pair` build rule using `go build -ldflags "-X main.defaultPairHome=$(CURDIR)" -o $@ ./cmd/pair-go`.
-- [ ] Keep `$(BIN_DIR)/pair-go` building from `./cmd/pair-go`; it may use the same `defaultPairHome` ldflag for copied local installs.
-- [ ] Update install-layout test: installed `pair` must be executable and not a symlink; installed `pair-go` remains executable; `pair-dev` may remain a symlink if still a dev wrapper.
-- [ ] Run: `make build`
-- [ ] Run: `make test-pair-go-install-layout`
+- [x] Update `GO_BINS` so `pair` is a Go-built binary and `pair-go` remains built from the same package.
+- [x] Remove `pair` from `SHELL_BINS`; keep or explicitly drop `pair-dev` based on install behavior.
+- [x] Add a specific `$(BIN_DIR)/pair` build rule using `go build -ldflags "-X main.defaultPairHome=$(CURDIR)" -o $@ ./cmd/pair-go`.
+- [x] Keep `$(BIN_DIR)/pair-go` building from `./cmd/pair-go`; it may use the same `defaultPairHome` ldflag for copied local installs.
+- [x] Update install-layout test: installed `pair` must be executable and not a symlink; installed `pair-go` remains executable; `pair-dev` may remain a symlink if still a dev wrapper.
+- [x] Run: `make build`
+- [x] Run: `make test-pair-go-install-layout`
 
 ### Task 4: Adjacent native asset install layout
 
@@ -128,12 +128,12 @@
 - Modify: `tests/pair-go-install-layout-test.sh`
 - Modify: `../homebrew-pair/Formula/pair.rb`
 
-- [ ] Keep `nvim/` and `zellij/` adjacent to `PAIR_HOME` for this issue; do not embed.
-- [ ] Local `make install` remains source-tree based for native assets: installed `pair` is copied to `~/.local/bin`, and when it has no sibling `pair-shell`, `AssetRoot` falls back to build-time `defaultPairHome=$(CURDIR)` to find the repo checkout assets.
-- [ ] Homebrew install remains `libexec`-adjacent: formula installs `bin/`, `nvim/`, and `zellij/` under `libexec`, then builds Go `pair`, `pair-go`, and required helper binaries into `libexec/bin` with `defaultPairHome=#{libexec}`.
-- [ ] Update formula comments and built-binary list so Homebrew surfaces `bin/pair` as the Go-built public command and retains `bin/pair-shell` only as an internal compatibility launcher.
-- [ ] Test that local installed `pair --help` reaches the shell help through the Go entrypoint.
-- [ ] Run: `make test-pair-go-install-layout`
+- [x] Keep `nvim/` and `zellij/` adjacent to `PAIR_HOME` for this issue; do not embed.
+- [x] Local `make install` remains source-tree based for native assets: installed `pair` is copied to `~/.local/bin`, and when it has no sibling `pair-shell`, `AssetRoot` falls back to build-time `defaultPairHome=$(CURDIR)` to find the repo checkout assets.
+- [x] Homebrew install remains `libexec`-adjacent: formula installs `bin/`, `nvim/`, and `zellij/` under `libexec`, then builds Go `pair`, `pair-go`, and required helper binaries into `libexec/bin` with `defaultPairHome=#{libexec}`.
+- [x] Update formula comments and built-binary list so Homebrew surfaces `bin/pair` as the Go-built public command and retains `bin/pair-shell` only as an internal compatibility launcher.
+- [x] Test that local installed `pair --help` reaches the shell help through the Go entrypoint.
+- [x] Run: `make test-pair-go-install-layout`
 
 ## Chunk 3: Compatibility Shim Inventory
 
@@ -146,11 +146,11 @@
 - Modify: `CHANGELOG.md` if the Homebrew/release note wording needs a packaging entry.
 - Modify: `../homebrew-pair/Formula/pair.rb`
 
-- [ ] Document `bin/pair-shell` as a retained compatibility launcher and explain why it is not obsolete yet.
-- [ ] Document `pair-dev` as retained dev-mode wrapper that runs the Go public `pair`.
-- [ ] Document legacy helper binaries retained because native zellij/nvim/shell callers still reference them.
-- [ ] Remove stale wording that says `pair-go launch` is the only Go-owned launch test surface; installed `pair` is now the public Go-owned entrypoint.
-- [ ] Update Homebrew wording: formula comments and README/CHANGELOG must say Homebrew installs a Go-built `pair` plus adjacent native assets under `libexec`.
+- [x] Document `bin/pair-shell` as a retained compatibility launcher and explain why it is not obsolete yet.
+- [x] Document `pair-dev` as retained dev-mode wrapper that runs the Go public `pair`.
+- [x] Document legacy helper binaries retained because native zellij/nvim/shell callers still reference them.
+- [x] Remove stale wording that says `pair-go launch` is the only Go-owned launch test surface; installed `pair` is now the public Go-owned entrypoint.
+- [x] Update Homebrew wording: formula comments and README/CHANGELOG must say Homebrew installs a Go-built `pair` plus adjacent native assets under `libexec`.
 
 ## Chunk 4: Verification And Closure
 
@@ -160,15 +160,29 @@
 - Modify: `workshop/issues/000079-go-packaging-consolidation.md`
 - Modify: `workshop/plans/000079-go-packaging-consolidation-plan.md`
 
-- [ ] Run focused Go tests: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`.
-- [ ] Run packaging tests: `make build && make test-pair-go-install-layout`.
-- [ ] Run an upgrade-layout test in `tests/pair-go-install-layout-test.sh`: seed `~/.local/bin/pair` as the old symlink-to-source-shell layout, run `make install`, then assert `~/.local/bin/pair` is now a regular executable Go binary, `bin/pair-shell` remains tracked/executable under the source root, `PAIR_HOME` override works, and default-root fallback lets installed `pair --help` reach the shell help.
-- [ ] Run a Homebrew formula dry-run/smoke if available: `brew test --formula ../homebrew-pair/Formula/pair.rb` or record the exact local blocker; at minimum run `ruby -c ../homebrew-pair/Formula/pair.rb`.
-- [ ] Run launcher smoke: `bin/pair --help`, `bin/pair-go launch --help`, `bin/pair-dev --help`.
-- [ ] Run broader impacted tests: `make test-dev-rebuild test-session-watch test-continue`.
-- [ ] If practical, run Linux smoke with the available local toolchain; otherwise record why it was not available.
-- [ ] Update issue checklist/log with verification evidence.
-- [ ] Update atlas/README and run stale-doc grep for old packaging statements.
+- [x] Run focused Go tests: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`.
+- [x] Run packaging tests: `make build && make test-pair-go-install-layout`.
+- [x] Run an upgrade-layout test in `tests/pair-go-install-layout-test.sh`: seed `~/.local/bin/pair` as the old symlink-to-source-shell layout, run `make install`, then assert `~/.local/bin/pair` is now a regular executable Go binary, `bin/pair-shell` remains tracked/executable under the source root, `PAIR_HOME` override works, and default-root fallback lets installed `pair --help` reach the shell help.
+- [x] Run a Homebrew formula dry-run/smoke if available: `brew test --formula ../homebrew-pair/Formula/pair.rb` or record the exact local blocker; at minimum run `ruby -c ../homebrew-pair/Formula/pair.rb`.
+- [x] Run launcher smoke: `bin/pair --help`, `bin/pair-go launch --help`, `bin/pair-dev --help`.
+- [x] Run broader impacted tests: `make test-dev-rebuild test-session-watch test-continue`.
+- [x] If practical, run Linux smoke with the available local toolchain; otherwise record why it was not available.
+- [x] Update issue checklist/log with verification evidence.
+- [x] Update atlas/README and run stale-doc grep for old packaging statements.
+
+Verification evidence captured 2026-06-30:
+`go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`; `make build`;
+`make test-pair-go-install-layout`; `bin/pair --help`; `bin/pair-go launch
+--help`; `bin/pair-dev --help`; `make test-dev-rebuild test-session-watch
+test-continue`; `go test ./... -count=1`; `ruby -c
+../homebrew-pair/Formula/pair.rb`; stale-doc grep for old #77 packaging
+wording. Homebrew tap evidence: sibling repo `../homebrew-pair` commit
+`3aeb2a6 pair: build Go public entrypoint` updates `Formula/pair.rb`.
+`brew test --formula ../homebrew-pair/Formula/pair.rb` is not
+supported by this Homebrew (`invalid option: --formula`), so formula validation
+is syntax-only unless tested through an installed tap. Linux smoke was not run
+because this local workspace is Darwin-only (`uname -s` => `Darwin`) and no
+Linux runner is configured.
 
 ## Implementation Notes
 
@@ -178,3 +192,11 @@
 - `ARCH-PURPOSE`: #79 is not complete if only docs change; installed `pair` must become the Go-owned public command.
 - `ARCH-DRY`: direct `pair` and `pair-go launch` must share one compatibility request builder.
 - `ARCH-PURE`: mode selection and request construction stay pure; filesystem/exec behavior stays in the `cmd/pair-go/main.go` runtime seam.
+
+## Revisions
+
+### 2026-06-30 — close-review REWORK traceability fixes
+
+- Corrected the Core Concepts name from planned `LegacyPairRequest` to implemented `LegacyLaunchRequest`.
+- Added exact Homebrew tap evidence: sibling repo `../homebrew-pair` commit `3aeb2a6 pair: build Go public entrypoint`.
+- Added standalone Make prerequisites for generated `bin/pair` on `test-cmux-ownership` and `test-continue`.


codex
I’ll use the required startup skill first, then I’ll inspect the issue/plan and the actual working tree diff before giving the boundary verdict.
exec
/bin/zsh -lc "sed -n '1,220p' /Users/xianxu/workspace/ariadne/construct/adapted/superpowers-using-superpowers/SKILL.md && sdlc --help && sed -n '1,220p' workshop/lessons.md" in /Users/xianxu/workspace/pair
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


codex
I’ve loaded the repo workflow contract. Now I’m checking the actual issue, the changed files, and the diff window so the review is tied to the stated close boundary rather than the summary in the prompt.
exec
/bin/zsh -lc "sed -n '1,260p' workshop/issues/000079-go-packaging-consolidation.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
---
id: 000079
status: working
deps: [000077, 000078]
github_issue:
created: 2026-06-26
updated: 2026-06-30
estimate_hours: 3.13
started: 2026-06-30T16:59:55-07:00
---

# pair Go packaging consolidation

## Problem

The migration is not complete until packaging actually becomes simpler. A Go entrypoint with a pile of stale aliases, duplicated binaries, and unclear asset handling would miss the purpose of #72.

## Spec

Consolidate release/install packaging around the primary Go `pair` binary and its required assets.

Expected work:

- Decide whether `nvim/` and `zellij/` assets are embedded in the binary or installed adjacent to it.
- Update Homebrew/install/build docs and `Makefile.local` accordingly.
- Remove obsolete compatibility shims only after all callers derive from the Go command or documented native assets.
- Verify clean install/upgrade behavior on the supported platform and, if practical, a Linux smoke path.
- Update atlas/README so the packaging architecture is discoverable.

This is where the migration proves its value: fewer installed moving parts and a clearer release story (`ARCH-PURPOSE`).

Approved design: make the installed public `pair` command Go-owned while keeping native assets adjacent. Build `cmd/pair-go` as both `bin/pair` and `bin/pair-go`: direct `pair ...` invokes the compatibility launch handoff, while `pair-go ...` keeps the explicit development dispatcher surface (`pair-go launch`, helper routes). Move the existing Bash launcher behind an internal compatibility name (`bin/pair-shell`) for this issue rather than pretending the full zellij lifecycle is already native Go. Keep `nvim/` and `zellij/` adjacent assets because they are loaded by Neovim/Zellij directly and are heavily tested as files; embedding would require extraction/path rewrites across many surfaces with little packaging payoff right now. `ARCH-PURPOSE`: #79 is only done if installed `pair` is the Go public entrypoint, not just if docs say it will be later. `ARCH-DRY` and `ARCH-PURE`: direct `pair` and `pair-go launch` share one pure compatibility request builder; filesystem/exec behavior stays in a thin runtime seam.

## Done when

- [x] Packaging installs the primary Go `pair` command and required assets coherently.
- [x] Obsolete compatibility shims are removed or explicitly retained with a reason.
- [x] README and atlas describe the new install/runtime layout.
- [x] Clean install and upgrade paths are verified.
- [x] Pair remains usable after merge.

## Plan

- [x] Choose embedded vs adjacent asset strategy.
- [x] Update build/install/Homebrew wiring.
- [x] Remove or document remaining shims.
- [x] Run clean install/upgrade verification.
- [x] Update docs and atlas.

Detailed implementation plan: `workshop/plans/000079-go-packaging-consolidation-plan.md`.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.20 impl=0.08
item: cross-cutting-refactor design=0.60 impl=0.64
item: smaller-go-module design=0.35 impl=0.40
item: atlas-docs design=0.25 impl=0.20
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 3.13
```

## Log

### 2026-06-26

Created from #72 as the final consolidation milestone. This should land only after the command migration has already made packaging simpler in practice.

### 2026-06-30

Claimed after #78 landed. Chose adjacent native assets and Go public entrypoint: build `cmd/pair-go` as installed `pair`, retain `pair-go` as the development dispatcher alias, and keep the current shell launcher as an internal compatibility handoff while the zellij lifecycle remains shell-owned. Wrote durable plan at `workshop/plans/000079-go-packaging-consolidation-plan.md`. Plan-quality found missing Homebrew and upgrade specificity; tightened the plan to include sibling formula `../homebrew-pair/Formula/pair.rb`, a concrete old-symlink-to-Go-binary upgrade test, and a single decided asset strategy: local installs stay source-tree adjacent, Homebrew installs an adjacent `libexec` tree. Second plan-quality pass found asset-root and tracked-file ambiguity; tightened the plan again so pure `AssetRoot` resolves `PAIR_HOME` / sibling root / build-time `defaultPairHome`, and so `bin/pair-shell` is tracked while generated `bin/pair` is ignored. Estimate derived with v3.1 calibration; calibration source is marked stale by `sdlc estimate-source`, so the number is provisional but uses the required method.

Implemented #79 packaging consolidation. `bin/pair` is now generated from `cmd/pair-go`, `bin/pair-shell` is the tracked compatibility launcher, local install copies a regular Go `pair` binary, and Homebrew builds Go `pair` / `pair-go` plus required runtime helpers into `libexec/bin` with adjacent native assets. Homebrew tap evidence: sibling repo `../homebrew-pair` commit `3aeb2a6 pair: build Go public entrypoint` updates `Formula/pair.rb`. Verification: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`; `make build`; `make test-pair-go-install-layout`; `bin/pair --help`; `bin/pair-go launch --help`; `bin/pair-dev --help`; `make test-dev-rebuild test-session-watch test-continue`; `go test ./... -count=1`; `ruby -c ../homebrew-pair/Formula/pair.rb`; stale-doc grep for old #77 packaging wording. `brew test --formula ../homebrew-pair/Formula/pair.rb` was not available on this Homebrew (`invalid option: --formula`), so the formula was syntax-checked locally rather than installed over the operator environment. Linux smoke was not run because this workspace is Darwin-only (`uname -s` => `Darwin`) and no Linux runner is configured.

exec
/bin/zsh -lc "sed -n '1,260p' workshop/plans/000079-go-packaging-consolidation-plan.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
# Go Packaging Consolidation Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the installed public `pair` command Go-owned while keeping native assets adjacent and preserving existing launcher behavior.

**Architecture:** Build `cmd/pair-go` into both `bin/pair` and `bin/pair-go`. When invoked as `pair`, the Go binary resolves an asset root and executes `<asset-root>/bin/pair-shell`; when invoked as `pair-go`, it keeps the explicit dispatcher surface and `pair-go launch` handoff. Asset root resolution is pure and ordered: explicit `PAIR_HOME`, then executable sibling root when `<dir>/pair-shell` exists, then build-time `defaultPairHome` injected by Make/Homebrew. `nvim/` and `zellij/` stay adjacent assets because they are native runtime files loaded by Neovim/Zellij and heavily tested in place.

**Tech Stack:** Go command entrypoint, Bash compatibility launcher, GNU/BSD Make install wiring, shell process-level tests, existing zellij/nvim adjacent assets.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `EntrypointMode` | `cmd/internal/entrypoint/mode.go` | new |
| `AssetRoot` | `cmd/internal/entrypoint/asset_root.go` | new |
| `LegacyLaunchRequest` | `cmd/internal/entrypoint/launch.go` | modified |

**EntrypointMode** — Determines whether one executable invocation should behave as public `pair` or development `pair-go`.
- **Relationships:** 1:1 with the executable basename; `cmd/pair-go/main.go` owns the argv/env IO and calls this pure classifier.
- **DRY rationale:** One classifier prevents `pair` and `pair-go launch` from growing parallel path-resolution rules (ARCH-DRY).
- **Future extensions:** If the shell launcher is later fully ported, this mode decision becomes the dispatch point for native launch instead of compatibility exec.

**AssetRoot** — Pure policy for choosing the root that owns adjacent runtime assets (`bin/pair-shell`, `nvim/`, `zellij/`).
- **Relationships:** N:1 from local source build, local copied install, and Homebrew `libexec` install into one root decision; `LegacyLaunchRequest` consumes the resolved root.
- **DRY rationale:** Prevents local install and Homebrew install from inventing separate path rules for the same asset layout (ARCH-DRY).
- **Future extensions:** Can add an extracted-embedded asset dir later without changing launcher request construction.

**LegacyLaunchRequest** — Describes the compatibility exec into the shell launcher.
- **Relationships:** N:1 from `pair` direct mode and `pair-go launch` mode into one request builder; each mode only changes display/diagnostic wording and argv shape. Carries the selected `AssetRoot` and computes `<asset-root>/bin/pair-shell`.
- **DRY rationale:** Keeps legacy shell handoff rules single-sourced while the actual zellij lifecycle remains shell-owned in this issue (ARCH-PURE).
- **Future extensions:** Can be deleted once shell launch is replaced by native Go launch.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `PairEntrypointRuntime` | `cmd/pair-go/main.go` | modified | `os.Executable`, `os.Stat`, `syscall.Exec`, environment |
| `ShellLauncherShim` | `bin/pair-shell` | new | existing Bash `bin/pair` launcher behavior |
| `InstallLayout` | `Makefile.local` | modified | build/install filesystem layout |
| `HomebrewFormula` | `../homebrew-pair/Formula/pair.rb` | modified | Homebrew libexec/bin install layout |
| `InstallLayoutTest` | `tests/pair-go-install-layout-test.sh` | modified | temp HOME install and process execution |

**PairEntrypointRuntime** — Thin IO shell around the pure entrypoint mode/request logic.
- **Injected into:** `runWithLegacyRuntime` tests via the existing fake runtime shape; runtime supplies `os.Executable`, `PAIR_HOME`, build-time `defaultPairHome`, and stat probes for candidate roots.
- **Future extensions:** Can exec a native Go launch path instead of `bin/pair-shell` without changing tests for mode classification.

**ShellLauncherShim** — The existing Bash launcher, renamed to an internal compatibility target.
- **Injected into:** Go `pair` direct mode and `pair-go launch` handoff.
- **Future extensions:** Shrinks as launch behavior moves to Go; retained explicitly because zellij lifecycle, prompt UI, restart cleanup, title poller, and shell helper orchestration remain shell-owned after #78.

**InstallLayout** — Builds installed `pair` as a Go binary, keeps `pair-go` as the dev dispatcher alias, and keeps native assets source-tree-adjacent for local installs.
- **Injected into:** `make install`, local development and Homebrew-style source checkout installs.
- **Future extensions:** Asset embedding can replace adjacent asset install only after zellij/nvim callers derive from a virtual/extracted asset root.

**HomebrewFormula** — Installs the release layout under Homebrew `libexec` with `bin/`, `nvim/`, and `zellij/` adjacent.
- **Injected into:** `brew install pair` and `brew upgrade pair` through the sibling `xianxu/homebrew-pair` tap.
- **Future extensions:** If Pair later embeds assets, this formula stops installing `nvim/` and `zellij/` trees.

**InstallLayoutTest** — Process-level fake install under temp HOME.
- **Injected into:** `make test-pair-go-install-layout`.
- **Future extensions:** Add Linux smoke assertions if a Linux CI path becomes available.

## Chunk 1: Public Go Entrypoint Compatibility

### Task 1: Protect invocation-mode behavior

**Files:**
- Modify: `cmd/internal/entrypoint/launch.go`
- Create: `cmd/internal/entrypoint/asset_root.go`
- Create: `cmd/internal/entrypoint/asset_root_test.go`
- Create: `cmd/internal/entrypoint/mode.go`
- Modify: `cmd/pair-go/main_test.go`

- [x] Add tests showing executable basename `pair` resolves to direct public launcher mode.
- [x] Add tests showing executable basename `pair-go` with `launch` still resolves to explicit launch handoff.
- [x] Add tests showing `pair-go` helper routes still dispatch without touching the shell launcher.
- [x] Add pure tests for asset-root resolution:
  - `PAIR_HOME=/repo` wins when `/repo/bin/pair-shell` exists.
  - executable `/repo/bin/pair` resolves sibling root `/repo` when `/repo/bin/pair-shell` exists.
  - copied executable `/home/me/.local/bin/pair` falls back to build-time default root `/repo` when sibling shell is absent and `/repo/bin/pair-shell` exists.
  - missing sibling and missing build-time root produces a diagnostic naming `pair-shell` and `PAIR_HOME`.
- [x] Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
- [x] Implement `EntrypointMode`, `AssetRoot`, and shared legacy request construction.
- [x] Re-run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`

### Task 2: Move shell launcher behind an internal compatibility name

**Files:**
- Move: `bin/pair` -> `bin/pair-shell`
- Delete tracked source: `bin/pair` (after `git mv`; future `bin/pair` is generated build output only)
- Modify: `.gitignore`
- Modify: `cmd/pair-go/main.go`
- Modify: `bin/pair-dev`

- [x] Move the existing Bash launcher body to `bin/pair-shell`.
- [x] Update `.gitignore`: remove the `!bin/pair` tracked-script exception and add `!bin/pair-shell`; `bin/pair` stays ignored as generated Go build output.
- [x] Update Go direct `pair` mode to exec sibling `pair-shell` with argv[0] presented as `pair`.
- [x] Update `pair-go launch ...` to exec sibling `pair-shell` with the same argv compatibility as before.
- [x] Update `pair-dev` to export `PAIR_DEV=1` and exec sibling `pair` (the Go binary), not `pair-shell`, so dev mode exercises the public entrypoint.
- [x] Run: `bin/pair-go launch --help` after build and confirm it reaches the launcher help.
- [x] Run: `bin/pair --help` after build and confirm it reaches the same launcher help.

## Chunk 2: Build And Install Layout

### Task 3: Build `pair` from `cmd/pair-go`

**Files:**
- Modify: `Makefile.local`
- Modify: `tests/pair-go-install-layout-test.sh`

- [x] Update `GO_BINS` so `pair` is a Go-built binary and `pair-go` remains built from the same package.
- [x] Remove `pair` from `SHELL_BINS`; keep or explicitly drop `pair-dev` based on install behavior.
- [x] Add a specific `$(BIN_DIR)/pair` build rule using `go build -ldflags "-X main.defaultPairHome=$(CURDIR)" -o $@ ./cmd/pair-go`.
- [x] Keep `$(BIN_DIR)/pair-go` building from `./cmd/pair-go`; it may use the same `defaultPairHome` ldflag for copied local installs.
- [x] Update install-layout test: installed `pair` must be executable and not a symlink; installed `pair-go` remains executable; `pair-dev` may remain a symlink if still a dev wrapper.
- [x] Run: `make build`
- [x] Run: `make test-pair-go-install-layout`

### Task 4: Adjacent native asset install layout

**Files:**
- Modify: `Makefile.local`
- Modify: `tests/pair-go-install-layout-test.sh`
- Modify: `../homebrew-pair/Formula/pair.rb`

- [x] Keep `nvim/` and `zellij/` adjacent to `PAIR_HOME` for this issue; do not embed.
- [x] Local `make install` remains source-tree based for native assets: installed `pair` is copied to `~/.local/bin`, and when it has no sibling `pair-shell`, `AssetRoot` falls back to build-time `defaultPairHome=$(CURDIR)` to find the repo checkout assets.
- [x] Homebrew install remains `libexec`-adjacent: formula installs `bin/`, `nvim/`, and `zellij/` under `libexec`, then builds Go `pair`, `pair-go`, and required helper binaries into `libexec/bin` with `defaultPairHome=#{libexec}`.
- [x] Update formula comments and built-binary list so Homebrew surfaces `bin/pair` as the Go-built public command and retains `bin/pair-shell` only as an internal compatibility launcher.
- [x] Test that local installed `pair --help` reaches the shell help through the Go entrypoint.
- [x] Run: `make test-pair-go-install-layout`

## Chunk 3: Compatibility Shim Inventory

### Task 5: Retain/remove shims intentionally

**Files:**
- Modify: `atlas/go-migration-inventory.md`
- Modify: `README.md`
- Modify: `atlas/architecture.md`
- Modify: `CHANGELOG.md` if the Homebrew/release note wording needs a packaging entry.
- Modify: `../homebrew-pair/Formula/pair.rb`

- [x] Document `bin/pair-shell` as a retained compatibility launcher and explain why it is not obsolete yet.
- [x] Document `pair-dev` as retained dev-mode wrapper that runs the Go public `pair`.
- [x] Document legacy helper binaries retained because native zellij/nvim/shell callers still reference them.
- [x] Remove stale wording that says `pair-go launch` is the only Go-owned launch test surface; installed `pair` is now the public Go-owned entrypoint.
- [x] Update Homebrew wording: formula comments and README/CHANGELOG must say Homebrew installs a Go-built `pair` plus adjacent native assets under `libexec`.

## Chunk 4: Verification And Closure

### Task 6: End-to-end verification

**Files:**
- Modify: `workshop/issues/000079-go-packaging-consolidation.md`
- Modify: `workshop/plans/000079-go-packaging-consolidation-plan.md`

- [x] Run focused Go tests: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`.
- [x] Run packaging tests: `make build && make test-pair-go-install-layout`.
- [x] Run an upgrade-layout test in `tests/pair-go-install-layout-test.sh`: seed `~/.local/bin/pair` as the old symlink-to-source-shell layout, run `make install`, then assert `~/.local/bin/pair` is now a regular executable Go binary, `bin/pair-shell` remains tracked/executable under the source root, `PAIR_HOME` override works, and default-root fallback lets installed `pair --help` reach the shell help.
- [x] Run a Homebrew formula dry-run/smoke if available: `brew test --formula ../homebrew-pair/Formula/pair.rb` or record the exact local blocker; at minimum run `ruby -c ../homebrew-pair/Formula/pair.rb`.
- [x] Run launcher smoke: `bin/pair --help`, `bin/pair-go launch --help`, `bin/pair-dev --help`.
- [x] Run broader impacted tests: `make test-dev-rebuild test-session-watch test-continue`.
- [x] If practical, run Linux smoke with the available local toolchain; otherwise record why it was not available.
- [x] Update issue checklist/log with verification evidence.
- [x] Update atlas/README and run stale-doc grep for old packaging statements.

Verification evidence captured 2026-06-30:
`go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`; `make build`;
`make test-pair-go-install-layout`; `bin/pair --help`; `bin/pair-go launch
--help`; `bin/pair-dev --help`; `make test-dev-rebuild test-session-watch
test-continue`; `go test ./... -count=1`; `ruby -c
../homebrew-pair/Formula/pair.rb`; stale-doc grep for old #77 packaging
wording. Homebrew tap evidence: sibling repo `../homebrew-pair` commit
`3aeb2a6 pair: build Go public entrypoint` updates `Formula/pair.rb`.
`brew test --formula ../homebrew-pair/Formula/pair.rb` is not
supported by this Homebrew (`invalid option: --formula`), so formula validation
is syntax-only unless tested through an installed tap. Linux smoke was not run
because this local workspace is Darwin-only (`uname -s` => `Darwin`) and no
Linux runner is configured.

## Implementation Notes

- This plan deliberately chooses adjacent assets over embedding. `nvim/` and `zellij/` are native runtime surfaces loaded by their own tools; embedding would require extraction or virtual path rewrites across many tested seams. Adjacent assets satisfy #79 now with lower risk and preserve direct edit/test loops.
- Git outcome is explicit: `bin/pair-shell` is tracked source, `bin/pair` is generated Go build output and ignored. The existing blanket `bin/*` ignore remains; the tracked-script exception changes from `!bin/pair` to `!bin/pair-shell`.
- Asset-root outcome is explicit: pure `AssetRoot` chooses `PAIR_HOME`, sibling executable root, or build-time `defaultPairHome`; runtime only probes filesystem existence and execs the resulting shell path.
- `ARCH-PURPOSE`: #79 is not complete if only docs change; installed `pair` must become the Go-owned public command.
- `ARCH-DRY`: direct `pair` and `pair-go launch` must share one compatibility request builder.
- `ARCH-PURE`: mode selection and request construction stay pure; filesystem/exec behavior stays in the `cmd/pair-go/main.go` runtime seam.

## Revisions

### 2026-06-30 — close-review REWORK traceability fixes

- Corrected the Core Concepts name from planned `LegacyPairRequest` to implemented `LegacyLaunchRequest`.
- Added exact Homebrew tap evidence: sibling repo `../homebrew-pair` commit `3aeb2a6 pair: build Go public entrypoint`.
- Added standalone Make prerequisites for generated `bin/pair` on `test-cmux-ownership` and `test-continue`.

exec
/bin/zsh -lc 'git diff --stat 776ecce27bc3643bc5598df559c52a3bb5424efe..HEAD && git diff --name-status 776ecce27bc3643bc5598df559c52a3bb5424efe..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 .gitignore                                         |   2 +-
 Makefile.local                                     |  34 +++--
 README.md                                          |  13 +-
 atlas/architecture.md                              |  38 ++++--
 atlas/go-migration-inventory.md                    |  23 ++--
 bin/{pair => pair-shell}                           |   0
 cmd/internal/entrypoint/asset_root.go              |  73 +++++++++++
 cmd/internal/entrypoint/asset_root_test.go         |  84 +++++++++++++
 cmd/internal/entrypoint/launch.go                  |  10 +-
 cmd/internal/entrypoint/launch_test.go             |  12 +-
 cmd/internal/entrypoint/mode.go                    |  21 ++++
 cmd/pair-go/main.go                                |  63 +++++++---
 cmd/pair-go/main_test.go                           | 138 ++++++++++++++++++---
 tests/pair-go-install-layout-test.sh               |  30 ++++-
 .../issues/000079-go-packaging-consolidation.md    |  22 ++--
 .../000079-go-packaging-consolidation-plan.md      | 110 +++++++++-------
 16 files changed, 533 insertions(+), 140 deletions(-)
M	.gitignore
M	Makefile.local
M	README.md
M	atlas/architecture.md
M	atlas/go-migration-inventory.md
R100	bin/pair	bin/pair-shell
A	cmd/internal/entrypoint/asset_root.go
A	cmd/internal/entrypoint/asset_root_test.go
M	cmd/internal/entrypoint/launch.go
M	cmd/internal/entrypoint/launch_test.go
A	cmd/internal/entrypoint/mode.go
M	cmd/pair-go/main.go
M	cmd/pair-go/main_test.go
M	tests/pair-go-install-layout-test.sh
M	workshop/issues/000079-go-packaging-consolidation.md
M	workshop/plans/000079-go-packaging-consolidation-plan.md

exec
/bin/zsh -lc 'git diff --check 776ecce27bc3643bc5598df559c52a3bb5424efe..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
codex
The issue and durable plan are internally aligned after the recorded revision. I’m doing the shadow-sweep now: all remaining references to the old `bin/pair` script, install behavior, and the new asset-root path rules.
exec
/bin/zsh -lc 'rg -n "bin/pair|pair-shell|pair-go launch|public `?pair|SHELL_BINS|GO_BINS|defaultPairHome|PAIR_HOME" . --glob '"'"'!workshop/history/**'"' --glob '"'!bin/pair'"'" in /Users/xianxu/workspace/pair
 exited 1 in 0ms:
zsh:1: unmatched "

exec
/bin/zsh -lc "nl -ba Makefile.local | sed -n '1,120p;240,285p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	# Repo-specific Makefile targets.
     2	# This file is included by Makefile — add your own targets here.
     3	
     4	.PHONY: build install pair pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-session-watch pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
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
    29	GO_BINS     := pair pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-session-watch pair-go
    30	SHELL_BINS  := pair-dev
    31	PAIR_HOME_LDFLAGS := -X main.defaultPairHome=$(CURDIR)
    32	
    33	build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
    34	
    35	install: build
    36		@mkdir -p $(INSTALL_DIR)
    37		@for b in $(GO_BINS); do \
    38		    install -m 755 $(BIN_DIR)/$$b $(INSTALL_DIR)/$$b && \
    39		    echo "installed: $(INSTALL_DIR)/$$b"; \
    40		done
    41		@for b in $(SHELL_BINS); do \
    42		    ln -sf "$(CURDIR)/$(BIN_DIR)/$$b" "$(INSTALL_DIR)/$$b" && \
    43		    echo "linked: $(INSTALL_DIR)/$$b -> $(CURDIR)/$(BIN_DIR)/$$b"; \
    44		done
    45	
    46	# Per-binary aliases — `make pair-scribe`, etc.
    47	pair:                   $(BIN_DIR)/pair
    48	pair-scribe:            $(BIN_DIR)/pair-scribe
    49	pair-scrollback-render: $(BIN_DIR)/pair-scrollback-render
    50	pair-wrap:              $(BIN_DIR)/pair-wrap
    51	pair-slug:              $(BIN_DIR)/pair-slug
    52	pair-continuation:      $(BIN_DIR)/pair-continuation
    53	pair-changelog:         $(BIN_DIR)/pair-changelog
    54	pair-context:           $(BIN_DIR)/pair-context
    55	pair-session-watch:     $(BIN_DIR)/pair-session-watch
    56	pair-go:                $(BIN_DIR)/pair-go
    57	
    58	# Tests
    59	#   make test       → go test ./... (cache disabled so reruns actually run)
    60	#   make test-race  → race-detector run, scoped to packages where the
    61	#                     suite has actual concurrent code worth checking.
    62	#                     pair-wrap has translateStdin's goroutine + timer +
    63	#                     channels; pair-scrollback-render's render() spawns
    64	#                     a goroutine drainer that races with em.Close()
    65	#                     inside the vt.Emulator library API itself, which is
    66	#                     outside our scope to fix — exclude that package
    67	#                     here rather than chase a library-level race.
    68	test: test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace test-review
    69		go test ./... -count=1
    70	
    71	# pair-title poller: single-instance guard (identity-checked liveness so a
    72	# recycled PID can't suppress the respawn) + the #71 frame meter (renames each
    73	# agent pane to "<agent> (<count>) [<cwd>]", with unchanged-skip). See
    74	# tests/pair-title-poller-test.sh.
    75	test-pair-title:
    76		bash tests/pair-title-poller-test.sh
    77	
    78	# cmux workspace ownership claim (presence beats a stale owner flag), so a
    79	# launch/attach/restart reclaims its workspace title. See
    80	# tests/cmux-ownership-test.sh.
    81	test-cmux-ownership: $(BIN_DIR)/pair
    82		bash tests/cmux-ownership-test.sh
    83	
    84	# Session id watcher must ignore stale pidfiles from prior launches and wait
    85	# for pair-wrap to publish the current agent PID.
    86	test-session-watch: $(BIN_DIR)/pair-session-watch
    87		bash tests/pair-session-watch-test.sh
    88	
    89	# Review workbench (#66): the integration shell tests. The pure modules
    90	# (record, reconstruct) run under `make test-lua`.
    91	test-review:
    92		bash tests/review-docflow-test.sh
    93		bash tests/review-apply-test.sh
    94		bash tests/review-handoff-test.sh
    95		bash tests/review-loop-test.sh
    96		bash tests/review-projection-test.sh
    97		bash tests/review-poke-test.sh
    98		bash tests/review-window-test.sh
    99		bash tests/review-toggle-test.sh
   100		bash tests/pair-review-target-test.sh
   101		bash tests/review-readiness-cli-test.sh
   102		bash tests/review-indicator-test.sh
   103		bash tests/review-resume-test.sh
   104	
   105	test-race:
   106		go test -count=1 -race ./cmd/pair-wrap/
   107	
   108	# Self-test for the headless-nvim timeout watchdog (tests/lib/run-headless.sh,
   109	# #60). Runs FIRST in `test` so the safety net every headless-nvim sub-target
   110	# now relies on is proven before those targets run — without it, a driver that
   111	# fails to quit (bare `qall` on a modified buffer → E37) hangs the whole suite.
   112	test-run-headless:
   113		bash tests/run-headless-test.sh
   114	
   115	# Headless Lua tests for nvim/*.lua decision modules (issue #000027). nvim -l
   116	# runs a Lua script in nvim's interpreter with the full vim.api available, so
   117	# pure logic AND buffer-manipulation can be asserted without a UI.
   118	test-lua:
   119		nvim -l nvim/slug_test.lua
   120		nvim -l nvim/scrollback_test.lua
   240	# pair-context — one-shot: print a pane's current context size (token count)
   241	# for the pair-title poller. Pure ctxmeter reader + shared transcript resolver.
   242	$(BIN_DIR)/pair-context: cmd/pair-context/main.go cmd/internal/contextcmd/contextcmd.go cmd/internal/ctxmeter/ctxmeter.go cmd/internal/transcript/transcript.go go.mod
   243		go build -o $@ ./cmd/pair-context
   244	
   245	# pair-session-watch — async codex/agy session-id discovery. Legacy
   246	# bin/pair-session-watch.sh remains as a compatibility shim for bin/pair.
   247	$(BIN_DIR)/pair-session-watch: cmd/pair-session-watch/main.go cmd/internal/sessionwatch/sessionwatch.go cmd/internal/sessionwatch/run.go cmd/internal/sessionwatch/runtime.go cmd/internal/adapt/adapt.go go.mod
   248		go build -o $@ ./cmd/pair-session-watch
   249	
   250	# pair-changelog — distills the pair TTY into the Alt+l change log (#53);
   251	# shares cmd/internal/model with pair-slug.
   252	$(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill.go cmd/pair-changelog/prompt.go cmd/internal/model/model.go go.mod
   253		go build -o $@ ./cmd/pair-changelog
   254	
   255	# pair-continuation — the deterministic writer for the `continuation` datatype
   256	# (ariadne#91): renders frontmatter, allocates a collision-safe timestamped
   257	# name under workshop/continuation/, writes, then commits + pushes (recovery).
   258	$(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-continuation/git.go cmd/pair-continuation/main.go go.mod
   259		go build -o $@ ./cmd/pair-continuation
   260	
   261	PAIR_GO_SRCS := cmd/pair-go/main.go cmd/internal/entrypoint/asset_root.go cmd/internal/entrypoint/launch.go cmd/internal/entrypoint/mode.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
   262	
   263	# pair — public Go-owned entrypoint. It execs bin/pair-shell for the current
   264	# shell-owned zellij lifecycle and uses the build-time defaultPairHome when a
   265	# copied local install has no sibling assets.
   266	$(BIN_DIR)/pair: $(PAIR_GO_SRCS)
   267		go build -ldflags "$(PAIR_HOME_LDFLAGS)" -o $@ ./cmd/pair-go
   268	
   269	# pair-go — development dispatcher and explicit migration surface.
   270	# `pair-go launch ...` shares the same bin/pair-shell handoff as public pair;
   271	# helper routes stay in the dispatcher.
   272	$(BIN_DIR)/pair-go: $(PAIR_GO_SRCS)
   273		go build -ldflags "$(PAIR_HOME_LDFLAGS)" -o $@ ./cmd/pair-go
   274	
   275	
   276	# Tart VM targets (tart, tart-gui, tart-mount, tart-stop, tart-clean)
   277	# live in .tart/Makefile, vendored from ariadne and auto-included by
   278	# Makefile.workflow. `make help-tart` for usage.

exec
/bin/zsh -lc "nl -ba cmd/pair-go/main.go | sed -n '1,150p'" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "nl -ba cmd/internal/entrypoint/asset_root.go cmd/internal/entrypoint/mode.go cmd/internal/entrypoint/launch.go | sed -n '1,180p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
usage: nl [-p] [-b type] [-d delim] [-f type] [-h type] [-i incr] [-l num]
          [-n format] [-s sep] [-v startnum] [-w width] [file]

 succeeded in 0ms:
     1	// pair-go is the development dispatcher for the future primary Go CLI. Its
     2	// launch route is a compatibility handoff to the current shell launcher.
     3	package main
     4	
     5	import (
     6		"fmt"
     7		"io"
     8		"os"
     9		"path/filepath"
    10		"syscall"
    11	
    12		"github.com/xianxu/pair/cmd/internal/dispatcher"
    13		"github.com/xianxu/pair/cmd/internal/entrypoint"
    14	)
    15	
    16	var defaultPairHome string
    17	
    18	func main() {
    19		os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
    20	}
    21	
    22	func run(args []string, stdout, stderr io.Writer) int {
    23		return runWithLegacyRuntime(args, stdout, stderr, osLegacyRuntime{})
    24	}
    25	
    26	type legacyRuntime interface {
    27		Executable() (string, error)
    28		PairHome() string
    29		DefaultPairHome() string
    30		Stat(path string) error
    31		Environ() []string
    32		Exec(label string, path string, argv []string, env []string) int
    33	}
    34	
    35	func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
    36		exe, err := rt.Executable()
    37		if err != nil {
    38			if len(args) > 0 && args[0] == "launch" {
    39				_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
    40				return 1
    41			}
    42			res := dispatcher.Dispatch(args)
    43			return writeResult(res, stdout, stderr)
    44		}
    45	
    46		switch entrypoint.ClassifyInvocation(exe, args) {
    47		case entrypoint.ModePublicPair:
    48			return runLegacyLaunch("pair", exe, args, stderr, rt)
    49		case entrypoint.ModePairGoLaunch:
    50			return runLegacyLaunch("pair-go launch", exe, args[1:], stderr, rt)
    51		default:
    52			res := dispatcher.Dispatch(args)
    53			return writeResult(res, stdout, stderr)
    54		}
    55	}
    56	
    57	func runLegacyLaunch(label string, executable string, args []string, stderr io.Writer, rt legacyRuntime) int {
    58		root, err := entrypoint.ResolveAssetRoot(entrypoint.AssetRootInput{
    59			PairHome:        rt.PairHome(),
    60			Executable:      executable,
    61			DefaultPairHome: rt.DefaultPairHome(),
    62			PairShellExists: func(root string) bool {
    63				return rt.Stat(entrypoint.PairShellPath(root)) == nil
    64			},
    65		})
    66		if err != nil {
    67			_, _ = fmt.Fprintf(stderr, "%s: %v; run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", label, err)
    68			return 1
    69		}
    70		req := entrypoint.ResolveLegacyLaunch(root, args)
    71		return rt.Exec(label, req.Path, req.Argv, rt.Environ())
    72	}
    73	
    74	type osLegacyRuntime struct{}
    75	
    76	func (osLegacyRuntime) Executable() (string, error) {
    77		return os.Executable()
    78	}
    79	
    80	func (osLegacyRuntime) PairHome() string {
    81		return os.Getenv("PAIR_HOME")
    82	}
    83	
    84	func (osLegacyRuntime) DefaultPairHome() string {
    85		return defaultPairHome
    86	}
    87	
    88	func (osLegacyRuntime) Stat(path string) error {
    89		path = filepath.Clean(path)
    90		info, err := os.Stat(path)
    91		if err != nil {
    92			return err
    93		}
    94		if info.IsDir() {
    95			return fmt.Errorf("is a directory")
    96		}
    97		return nil
    98	}
    99	
   100	func (osLegacyRuntime) Environ() []string {
   101		return os.Environ()
   102	}
   103	
   104	func (osLegacyRuntime) Exec(label string, path string, argv []string, env []string) int {
   105		if err := syscall.Exec(path, argv, env); err != nil {
   106			_, _ = fmt.Fprintf(os.Stderr, "%s: exec %s failed: %v\n", label, path, err)
   107			return 1
   108		}
   109		return 0
   110	}
   111	
   112	func writeResult(res dispatcher.Result, stdout, stderr io.Writer) int {
   113		if res.Stdout != "" {
   114			_, _ = io.WriteString(stdout, res.Stdout)
   115		}
   116		if res.Stderr != "" {
   117			_, _ = io.WriteString(stderr, res.Stderr)
   118		}
   119		return res.ExitCode
   120	}

exec
/bin/zsh -lc "nl -ba tests/pair-go-install-layout-test.sh | sed -n '1,120p'" in /Users/xianxu/workspace/pair
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
    10	install_bin="$tmp_home/.local/bin"
    11	old_bin="$tmp_home/old-bin"
    12	mkdir -p "$install_bin" "$old_bin"
    13	printf '#!/usr/bin/env bash\nprintf old-pair-shell\\n\n' > "$old_bin/pair"
    14	chmod +x "$old_bin/pair"
    15	ln -s "$old_bin/pair" "$install_bin/pair"
    16	
    17	HOME="$tmp_home" GOMODCACHE="$gomodcache" GOCACHE="$gocache" make -C "$repo_root" install >/dev/null
    18	
    19	test -x "$install_bin/pair"
    20	test ! -L "$install_bin/pair"
    21	test -x "$install_bin/pair-go"
    22	test -L "$install_bin/pair-dev"
    23	test -x "$repo_root/bin/pair-shell"
    24	
    25	out="$("$install_bin/pair" --help)"
    26	case "$out" in
    27	    pair\ —*) ;;
    28	    *)
    29	        printf 'pair --help did not reach pair help; first bytes:\n%s\n' "$out" >&2
    30	        exit 1
    31	        ;;
    32	esac
    33	
    34	out="$("$install_bin/pair-go" launch --help)"
    35	case "$out" in
    36	    pair\ —*) ;;
    37	    *)
    38	        printf 'pair-go launch --help did not reach pair help; first bytes:\n%s\n' "$out" >&2
    39	        exit 1
    40	        ;;
    41	esac
    42	
    43	out="$(PAIR_HOME="$repo_root" "$install_bin/pair" --help)"
    44	case "$out" in
    45	    pair\ —*) ;;
    46	    *)
    47	        printf 'PAIR_HOME pair --help did not reach pair help; first bytes:\n%s\n' "$out" >&2
    48	        exit 1
    49	        ;;
    50	esac
    51	
    52	printf 'pair-go install layout test passed\n'

exec
/bin/zsh -lc "nl -ba bin/pair-dev | sed -n '1,80p'; nl -ba bin/pair-shell | sed -n '1,80p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
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

codex
I found one asset-root edge case worth weighing: the pure resolver skips a cleaned root of `"."`, which makes `PAIR_HOME=.` not behave like an explicit asset root even when `./bin/pair-shell` exists. I’m checking whether there are any broader instances of old launcher naming or duplicated path rules before I settle severity.
exec
/bin/zsh -lc "nl -ba cmd/internal/entrypoint/asset_root.go | sed -n '1,120p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package entrypoint
     2	
     3	import (
     4		"fmt"
     5		"path/filepath"
     6		"strings"
     7	)
     8	
     9	type AssetRootInput struct {
    10		PairHome        string
    11		Executable      string
    12		DefaultPairHome string
    13		PairShellExists func(root string) bool
    14	}
    15	
    16	type AssetRoot struct {
    17		Root      string
    18		ShellPath string
    19		Source    string
    20	}
    21	
    22	func ResolveAssetRoot(input AssetRootInput) (AssetRoot, error) {
    23		exists := input.PairShellExists
    24		if exists == nil {
    25			exists = func(string) bool { return false }
    26		}
    27	
    28		candidates := make([]assetRootCandidate, 0, 3)
    29		if input.PairHome != "" {
    30			candidates = append(candidates, assetRootCandidate{root: input.PairHome, source: "PAIR_HOME"})
    31		}
    32		if input.Executable != "" {
    33			candidates = append(candidates, assetRootCandidate{
    34				root:   filepath.Dir(filepath.Dir(input.Executable)),
    35				source: "executable sibling",
    36			})
    37		}
    38		if input.DefaultPairHome != "" {
    39			candidates = append(candidates, assetRootCandidate{root: input.DefaultPairHome, source: "defaultPairHome"})
    40		}
    41	
    42		seen := map[string]bool{}
    43		checked := make([]string, 0, len(candidates))
    44		for _, candidate := range candidates {
    45			root := filepath.Clean(candidate.root)
    46			if root == "." || seen[root] {
    47				continue
    48			}
    49			seen[root] = true
    50			checked = append(checked, root)
    51			if exists(root) {
    52				return AssetRoot{
    53					Root:      root,
    54					ShellPath: PairShellPath(root),
    55					Source:    candidate.source,
    56				}, nil
    57			}
    58		}
    59	
    60		if len(checked) == 0 {
    61			checked = append(checked, "<none>")
    62		}
    63		return AssetRoot{}, fmt.Errorf("pair-shell not found; set PAIR_HOME to a Pair checkout/install root containing bin/pair-shell (checked: %s)", strings.Join(checked, ", "))
    64	}
    65	
    66	func PairShellPath(root string) string {
    67		return filepath.Join(root, "bin", "pair-shell")
    68	}
    69	
    70	type assetRootCandidate struct {
    71		root   string
    72		source string
    73	}

exec
/bin/zsh -lc "rg -n 'bin/pair|pair-shell|pair-go launch|public pair|public "'`pair|SHELL_BINS|GO_BINS|defaultPairHome|PAIR_HOME'"' . --glob '"'!workshop/history/**'"' --glob '"'!bin/pair'"'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
./tests/pair-rename.sh:17:PAIR_BIN="$SELF_DIR/../bin/pair"
./tests/pair-rename.sh:33:# the families enumerated in bin/pair's `rename_paths_for` helper.
./atlas/how-to-bring-up-a-new-harness-cli.md:59:- **Files:** `cmd/pair-session-watch` and `cmd/internal/sessionwatch` (`bin/pair-session-watch.sh` remains a compatibility shim).
./atlas/how-to-bring-up-a-new-harness-cli.md:66:- **File:** [bin/pair](file:///Users/xianxu/workspace/pair/bin/pair)
./atlas/how-to-bring-up-a-new-harness-cli.md:67:- Integrate the agent-specific resume argument in `bin/pair`:
./atlas/how-to-bring-up-a-new-harness-cli.md:82:**Telemetry Signal** (aspect `3`, see §3): `session-id` from `pair-session-watch` — `fired` when `AgentSpec.Match` resolves an id and the config is written, **`near-miss`** when a file matching the watch pattern is found but no id can be extracted (filename/format drift), `fail` when the 60s watch window elapses with no id at all (the session file never appeared where expected). The resume mapping in `bin/pair` is the *consumer* of this id; it's static config with no separate signal.
./atlas/how-to-bring-up-a-new-harness-cli.md:139:4. [ ] **Configure Launcher Recovery** in `bin/pair` (mapping `--conversation` or `--resume` flags).
./atlas/how-to-bring-up-a-new-harness-cli.md:156:`bin/pair` truncates the file once at session launch; all components then append
./atlas/how-to-bring-up-a-new-harness-cli.md:197:the same trust level as the existing scrollback logs. `bin/pair` removes it on quit.
./tests/cmux-ownership-test.sh:2:# Regression test for bin/pair's cmux_rename_workspace ownership claim.
./tests/cmux-ownership-test.sh:11:# Drives the REAL bin/pair through its PAIR_TEST_CALL seam with process-level
./tests/cmux-ownership-test.sh:20:PAIR="$ROOT/bin/pair"
./workshop/issues/000079-go-packaging-consolidation.md:32:Approved design: make the installed public `pair` command Go-owned while keeping native assets adjacent. Build `cmd/pair-go` as both `bin/pair` and `bin/pair-go`: direct `pair ...` invokes the compatibility launch handoff, while `pair-go ...` keeps the explicit development dispatcher surface (`pair-go launch`, helper routes). Move the existing Bash launcher behind an internal compatibility name (`bin/pair-shell`) for this issue rather than pretending the full zellij lifecycle is already native Go. Keep `nvim/` and `zellij/` adjacent assets because they are loaded by Neovim/Zellij directly and are heavily tested as files; embedding would require extraction/path rewrites across many surfaces with little packaging payoff right now. `ARCH-PURPOSE`: #79 is only done if installed `pair` is the Go public entrypoint, not just if docs say it will be later. `ARCH-DRY` and `ARCH-PURE`: direct `pair` and `pair-go launch` share one pure compatibility request builder; filesystem/exec behavior stays in a thin runtime seam.
./workshop/issues/000079-go-packaging-consolidation.md:76:Claimed after #78 landed. Chose adjacent native assets and Go public entrypoint: build `cmd/pair-go` as installed `pair`, retain `pair-go` as the development dispatcher alias, and keep the current shell launcher as an internal compatibility handoff while the zellij lifecycle remains shell-owned. Wrote durable plan at `workshop/plans/000079-go-packaging-consolidation-plan.md`. Plan-quality found missing Homebrew and upgrade specificity; tightened the plan to include sibling formula `../homebrew-pair/Formula/pair.rb`, a concrete old-symlink-to-Go-binary upgrade test, and a single decided asset strategy: local installs stay source-tree adjacent, Homebrew installs an adjacent `libexec` tree. Second plan-quality pass found asset-root and tracked-file ambiguity; tightened the plan again so pure `AssetRoot` resolves `PAIR_HOME` / sibling root / build-time `defaultPairHome`, and so `bin/pair-shell` is tracked while generated `bin/pair` is ignored. Estimate derived with v3.1 calibration; calibration source is marked stale by `sdlc estimate-source`, so the number is provisional but uses the required method.
./workshop/issues/000079-go-packaging-consolidation.md:78:Implemented #79 packaging consolidation. `bin/pair` is now generated from `cmd/pair-go`, `bin/pair-shell` is the tracked compatibility launcher, local install copies a regular Go `pair` binary, and Homebrew builds Go `pair` / `pair-go` plus required runtime helpers into `libexec/bin` with adjacent native assets. Homebrew tap evidence: sibling repo `../homebrew-pair` commit `3aeb2a6 pair: build Go public entrypoint` updates `Formula/pair.rb`. Verification: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`; `make build`; `make test-pair-go-install-layout`; `bin/pair --help`; `bin/pair-go launch --help`; `bin/pair-dev --help`; `make test-dev-rebuild test-session-watch test-continue`; `go test ./... -count=1`; `ruby -c ../homebrew-pair/Formula/pair.rb`; stale-doc grep for old #77 packaging wording. `brew test --formula ../homebrew-pair/Formula/pair.rb` was not available on this Homebrew (`invalid option: --formula`), so the formula was syntax-checked locally rather than installed over the operator environment. Linux smoke was not run because this workspace is Darwin-only (`uname -s` => `Darwin`) and no Linux runner is configured.
./nvim/changelog.lua:4:-- bin/pair-changelog-open. The distilled counterpart to scrollback.lua, but
./nvim/changelog.lua:81:-- The orchestrator (bin/pair-changelog-open) launches render+distill as a nohup'd
./zellij/layouts/main.kdl:7://   $PAIR_HOME        — pair install location (for nvim init)
./zellij/layouts/main.kdl:8:// All four are exported by bin/pair before invoking zellij.
./zellij/layouts/main.kdl:32:            // are exported by bin/pair before zellij launches, so the
./zellij/layouts/main.kdl:63:            // embed pid here so cleanup_quit_marker in bin/pair can reap
./zellij/layouts/main.kdl:67:            args "-c" "DATA_DIR=\"${XDG_DATA_HOME:-$HOME/.local/share}/pair\" && mkdir -p \"$DATA_DIR\" && export PAIR_NVIM_PID_FILE=\"$DATA_DIR/nvim-pid-${PAIR_TAG:-${PAIR_AGENT:-claude}}-draft\" && exec nvim -u \"$PAIR_HOME/nvim/init.lua\" \"$DATA_DIR/draft-${PAIR_TAG:-${PAIR_AGENT:-claude}}.md\""
./tests/pair-continue-test.sh:3:# name guard (bin/pair, #54). Drives the REAL bin/pair through its
./tests/pair-continue-test.sh:22:PAIR="$ROOT/bin/pair"
./tests/pair-continue-test.sh:115:# #55 in-session compaction (Alt+Shift+C). Drives the REAL bin/pair via the
./tests/pair-continue-test.sh:183:# REAL bin/pair helper seams through PAIR_TEST_CALL with process-level fakes.
./doctor/README.md:42:| `session-id/fail` or `near-miss` (aspect 3) | session file moved or id format changed | update `watch_dir`/`find_args`/`extract_id` in `bin/pair-session-watch.sh` |
./doctor/README.md:53:- The log truncates at each session launch (`bin/pair`), so it reflects the
./doctor/README.md:64:`$PAIR_HOME`-absolute instruction to run `doctor.sh` and propose fixes — so it
./cmd/pair-changelog/main.go:2:// log (issue #53). Invoked on demand by bin/pair-changelog-open (Alt+l): it
./atlas/review-workbench.md:62:- `readiness.lua` + `bin/pair-review-readiness` — pure/classified git readiness
./atlas/review-workbench.md:121:- `bin/pair-review-open <file>` — validates + spawns the **full-screen** floating pane
./tests/pair-session-watch-test.sh:36:"$ROOT/bin/pair-session-watch.sh" codex test "$ROOT" resume old-session 'say "hi"' --no-alt-screen &
./README.md:66:| **Alt+1**…**Alt+9** | nvim (insert, popup visible) | Quick-pick the Nth visible completion item. The popup tags its first nine items with indices (e.g. `⌥1 bin/pair-wrap`); past 9, use arrows or `<C-n>` / `<C-p>` |
./README.md:204:the adjacent Pair asset root and hands off to `bin/pair-shell`, the retained
./README.md:205:shell launcher that still owns the zellij lifecycle. `pair-go launch ...` remains
./zellij/config.kdl:39:// bin/pair prepends $PAIR_HOME/bin to PATH so the script is resolvable by
./zellij/config.kdl:40:// bare name — avoids the fragile `sh -c '$PAIR_HOME/...'` quoting dance.
./tests/emitter-health-test.sh:48:mkdir -p "$RT/pathbin"; cp "$fresh_pw" "$RT/pathbin/pair-wrap"; chmod +x "$RT/pathbin/pair-wrap"
./tests/emitter-health-test.sh:50:[ "$got" = "$RT/pathbin/pair-wrap" ] && pass "no pidfile ⇒ resolves via PATH" || fail "PATH fallback wrong: $got"
./cmd/pair-wrap/main.go:3:// Installed at bin/pair-wrap and invoked by zellij/layouts/main.kdl on
./cmd/pair-wrap/main.go:983:				// fix in bin/pair-wrap.py (949aeec).
./cmd/pair-wrap/main.go:1788:	// Open the always-on adaptation flight recorder. bin/pair truncates the
./cmd/pair-wrap/main.go:1930:			// bin/pair's cleanup_quit_marker handles them with the rest
./workshop/continuation/20260619T133818-review-m3.md:19:Read the M3 smoke diagnosis in the session/issue first; the exact bin scripts to rewrite are `bin/pair-review-toggle`, `bin/pair-review-open`, the `Alt r` block in `zellij/config.kdl`, and a new `PairReviewToggle()` in `nvim/init.lua` (model on the `Alt d`/`PairConfirmDetach` pattern). This is live-zellij iteration — implement, then the user re-runs the smoke checklist (in the M3 plan, Task 5). Keep it main-thread (a fork can't verify live zellij).
./workshop/continuation/20260619T133818-review-m3.md:51:- Code (branch `000066-agentic-review-workbench`, worktree `/Users/xianxu/workspace/pair`): `nvim/review/{record,reconstruct,apply,projection,markers,mode,docflow,handoff,init}.lua` (the consumer-half), `nvim/review.lua` (pane init), `nvim/pair_poke.lua` (id-based agent poke), `bin/pair-review-{open,toggle}`, `zellij/config.kdl` (the `Alt r` bind). Tests: `nvim/review/*_test.lua` (`make test-lua`) + `tests/review-*-test.sh` (`make test-review`).
./workshop/plans/000079-go-packaging-consolidation-plan.md:5:**Goal:** Make the installed public `pair` command Go-owned while keeping native assets adjacent and preserving existing launcher behavior.
./workshop/plans/000079-go-packaging-consolidation-plan.md:7:**Architecture:** Build `cmd/pair-go` into both `bin/pair` and `bin/pair-go`. When invoked as `pair`, the Go binary resolves an asset root and executes `<asset-root>/bin/pair-shell`; when invoked as `pair-go`, it keeps the explicit dispatcher surface and `pair-go launch` handoff. Asset root resolution is pure and ordered: explicit `PAIR_HOME`, then executable sibling root when `<dir>/pair-shell` exists, then build-time `defaultPairHome` injected by Make/Homebrew. `nvim/` and `zellij/` stay adjacent assets because they are native runtime files loaded by Neovim/Zellij and heavily tested in place.
./workshop/plans/000079-go-packaging-consolidation-plan.md:23:**EntrypointMode** — Determines whether one executable invocation should behave as public `pair` or development `pair-go`.
./workshop/plans/000079-go-packaging-consolidation-plan.md:25:- **DRY rationale:** One classifier prevents `pair` and `pair-go launch` from growing parallel path-resolution rules (ARCH-DRY).
./workshop/plans/000079-go-packaging-consolidation-plan.md:28:**AssetRoot** — Pure policy for choosing the root that owns adjacent runtime assets (`bin/pair-shell`, `nvim/`, `zellij/`).
./workshop/plans/000079-go-packaging-consolidation-plan.md:34:- **Relationships:** N:1 from `pair` direct mode and `pair-go launch` mode into one request builder; each mode only changes display/diagnostic wording and argv shape. Carries the selected `AssetRoot` and computes `<asset-root>/bin/pair-shell`.
./workshop/plans/000079-go-packaging-consolidation-plan.md:43:| `ShellLauncherShim` | `bin/pair-shell` | new | existing Bash `bin/pair` launcher behavior |
./workshop/plans/000079-go-packaging-consolidation-plan.md:49:- **Injected into:** `runWithLegacyRuntime` tests via the existing fake runtime shape; runtime supplies `os.Executable`, `PAIR_HOME`, build-time `defaultPairHome`, and stat probes for candidate roots.
./workshop/plans/000079-go-packaging-consolidation-plan.md:50:- **Future extensions:** Can exec a native Go launch path instead of `bin/pair-shell` without changing tests for mode classification.
./workshop/plans/000079-go-packaging-consolidation-plan.md:53:- **Injected into:** Go `pair` direct mode and `pair-go launch` handoff.
./workshop/plans/000079-go-packaging-consolidation-plan.md:83:  - `PAIR_HOME=/repo` wins when `/repo/bin/pair-shell` exists.
./workshop/plans/000079-go-packaging-consolidation-plan.md:84:  - executable `/repo/bin/pair` resolves sibling root `/repo` when `/repo/bin/pair-shell` exists.
./workshop/plans/000079-go-packaging-consolidation-plan.md:85:  - copied executable `/home/me/.local/bin/pair` falls back to build-time default root `/repo` when sibling shell is absent and `/repo/bin/pair-shell` exists.
./workshop/plans/000079-go-packaging-consolidation-plan.md:86:  - missing sibling and missing build-time root produces a diagnostic naming `pair-shell` and `PAIR_HOME`.
./workshop/plans/000079-go-packaging-consolidation-plan.md:94:- Move: `bin/pair` -> `bin/pair-shell`
./workshop/plans/000079-go-packaging-consolidation-plan.md:95:- Delete tracked source: `bin/pair` (after `git mv`; future `bin/pair` is generated build output only)
./workshop/plans/000079-go-packaging-consolidation-plan.md:98:- Modify: `bin/pair-dev`
./workshop/plans/000079-go-packaging-consolidation-plan.md:100:- [x] Move the existing Bash launcher body to `bin/pair-shell`.
./workshop/plans/000079-go-packaging-consolidation-plan.md:101:- [x] Update `.gitignore`: remove the `!bin/pair` tracked-script exception and add `!bin/pair-shell`; `bin/pair` stays ignored as generated Go build output.
./workshop/plans/000079-go-packaging-consolidation-plan.md:102:- [x] Update Go direct `pair` mode to exec sibling `pair-shell` with argv[0] presented as `pair`.
./workshop/plans/000079-go-packaging-consolidation-plan.md:103:- [x] Update `pair-go launch ...` to exec sibling `pair-shell` with the same argv compatibility as before.
./workshop/plans/000079-go-packaging-consolidation-plan.md:104:- [x] Update `pair-dev` to export `PAIR_DEV=1` and exec sibling `pair` (the Go binary), not `pair-shell`, so dev mode exercises the public entrypoint.
./workshop/plans/000079-go-packaging-consolidation-plan.md:105:- [x] Run: `bin/pair-go launch --help` after build and confirm it reaches the launcher help.
./workshop/plans/000079-go-packaging-consolidation-plan.md:106:- [x] Run: `bin/pair --help` after build and confirm it reaches the same launcher help.
./workshop/plans/000079-go-packaging-consolidation-plan.md:116:- [x] Update `GO_BINS` so `pair` is a Go-built binary and `pair-go` remains built from the same package.
./workshop/plans/000079-go-packaging-consolidation-plan.md:117:- [x] Remove `pair` from `SHELL_BINS`; keep or explicitly drop `pair-dev` based on install behavior.
./workshop/plans/000079-go-packaging-consolidation-plan.md:118:- [x] Add a specific `$(BIN_DIR)/pair` build rule using `go build -ldflags "-X main.defaultPairHome=$(CURDIR)" -o $@ ./cmd/pair-go`.
./workshop/plans/000079-go-packaging-consolidation-plan.md:119:- [x] Keep `$(BIN_DIR)/pair-go` building from `./cmd/pair-go`; it may use the same `defaultPairHome` ldflag for copied local installs.
./workshop/plans/000079-go-packaging-consolidation-plan.md:131:- [x] Keep `nvim/` and `zellij/` adjacent to `PAIR_HOME` for this issue; do not embed.
./workshop/plans/000079-go-packaging-consolidation-plan.md:132:- [x] Local `make install` remains source-tree based for native assets: installed `pair` is copied to `~/.local/bin`, and when it has no sibling `pair-shell`, `AssetRoot` falls back to build-time `defaultPairHome=$(CURDIR)` to find the repo checkout assets.
./workshop/plans/000079-go-packaging-consolidation-plan.md:133:- [x] Homebrew install remains `libexec`-adjacent: formula installs `bin/`, `nvim/`, and `zellij/` under `libexec`, then builds Go `pair`, `pair-go`, and required helper binaries into `libexec/bin` with `defaultPairHome=#{libexec}`.
./workshop/plans/000079-go-packaging-consolidation-plan.md:134:- [x] Update formula comments and built-binary list so Homebrew surfaces `bin/pair` as the Go-built public command and retains `bin/pair-shell` only as an internal compatibility launcher.
./workshop/plans/000079-go-packaging-consolidation-plan.md:149:- [x] Document `bin/pair-shell` as a retained compatibility launcher and explain why it is not obsolete yet.
./workshop/plans/000079-go-packaging-consolidation-plan.md:150:- [x] Document `pair-dev` as retained dev-mode wrapper that runs the Go public `pair`.
./workshop/plans/000079-go-packaging-consolidation-plan.md:152:- [x] Remove stale wording that says `pair-go launch` is the only Go-owned launch test surface; installed `pair` is now the public Go-owned entrypoint.
./workshop/plans/000079-go-packaging-consolidation-plan.md:165:- [x] Run an upgrade-layout test in `tests/pair-go-install-layout-test.sh`: seed `~/.local/bin/pair` as the old symlink-to-source-shell layout, run `make install`, then assert `~/.local/bin/pair` is now a regular executable Go binary, `bin/pair-shell` remains tracked/executable under the source root, `PAIR_HOME` override works, and default-root fallback lets installed `pair --help` reach the shell help.
./workshop/plans/000079-go-packaging-consolidation-plan.md:167:- [x] Run launcher smoke: `bin/pair --help`, `bin/pair-go launch --help`, `bin/pair-dev --help`.
./workshop/plans/000079-go-packaging-consolidation-plan.md:175:`make test-pair-go-install-layout`; `bin/pair --help`; `bin/pair-go launch
./workshop/plans/000079-go-packaging-consolidation-plan.md:176:--help`; `bin/pair-dev --help`; `make test-dev-rebuild test-session-watch
./workshop/plans/000079-go-packaging-consolidation-plan.md:190:- Git outcome is explicit: `bin/pair-shell` is tracked source, `bin/pair` is generated Go build output and ignored. The existing blanket `bin/*` ignore remains; the tracked-script exception changes from `!bin/pair` to `!bin/pair-shell`.
./workshop/plans/000079-go-packaging-consolidation-plan.md:191:- Asset-root outcome is explicit: pure `AssetRoot` chooses `PAIR_HOME`, sibling executable root, or build-time `defaultPairHome`; runtime only probes filesystem existence and execs the resulting shell path.
./workshop/plans/000079-go-packaging-consolidation-plan.md:193:- `ARCH-DRY`: direct `pair` and `pair-go launch` must share one compatibility request builder.
./workshop/plans/000079-go-packaging-consolidation-plan.md:202:- Added standalone Make prerequisites for generated `bin/pair` on `test-cmux-ownership` and `test-continue`.
./tests/dev-rebuild-test.sh:10:# critical because bin/pair runs under `set -e` and a failed restart-time build
./tests/dev-rebuild-test.sh:36:export PAIR_HOME="$ROOT"
./atlas/go-migration-inventory.md:30:- **go-entrypoint**: should become Go-owned public `pair` behavior.
./atlas/go-migration-inventory.md:50:| `bin/pair` / `bin/pair-shell` / `cmd/internal/launcher` / `cmd/internal/entrypoint` | Go public entrypoint plus retained shell launcher | user shell, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` is generated from `cmd/pair-go` and resolves `PAIR_HOME` / sibling root / build-time `defaultPairHome`, then execs `<asset-root>/bin/pair-shell` with `pair`-compatible argv/env. `bin/pair-shell` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` shares the same compatibility handoff. | `bin/pair-shell` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves invocation mode, asset root, and compatibility request; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Public entrypoint is Go-owned as of #79; `bin/pair-shell` is retained because real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
./atlas/go-migration-inventory.md:51:| `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling Go-built `pair`. | Resolves symlinks; depends on generated `bin/pair`, retained `bin/pair-shell`, and `bin/lib/dev-rebuild.sh`. | retained dev wrapper so developer launches exercise the public Go entrypoint | P1 |
./atlas/go-migration-inventory.md:52:| `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
./atlas/go-migration-inventory.md:53:| `zellij/layouts/main.kdl` | zellij native asset | `bin/pair` via `zellij --new-session-with-layout` | Defines agent and draft panes; shell expands Pair env at pane start. | Calls `pair-wrap`; calls `nvim -u "$PAIR_HOME/nvim/init.lua"`; writes `pane-<tag>-<agent>.json`; writes draft nvim pid file. | native-asset, packaged adjacent/embedded | P0 |
./atlas/go-migration-inventory.md:54:| `zellij/config.kdl` | zellij native asset | zellij session config from `bin/pair` | Global keybinds, copy command, scrollback buffer, pane frames. | Calls `copy-on-select.sh`, `pair-help`, `pair-scrollback-open`, `pair-changelog-open`; routes quit/restart/compact through nvim functions. | native-asset, packaged adjacent/embedded | P0 |
./atlas/go-migration-inventory.md:55:| `bin/pair-wrap` / `cmd/pair-wrap` | Go binary | zellij agent pane | `pair-wrap [--scrollback-log PATH] agent [args...]`; transparent PTY proxy; long-running; failure in diagnostics is swallowed. | Reads Pair env and agent command; writes `agent-output-<tag>`, `agent-pid-<tag>`, scrollback `.raw`/`.events.jsonl`, image capture files; may invoke `pair-slug`. | go-subcommand `pair wrap`; keep old binary as compat alias for KDL until caller moves | P0 |
./atlas/go-migration-inventory.md:56:| `bin/pair-slug` / `cmd/pair-slug` | Go binary | `pair-wrap` turn-end hook, tests | Env-driven, no stdin; resolves native transcript, proposes slug; exits 0 on most failures. | Requires `PAIR_TAG`, `PAIR_DATA_DIR`; reads config/transcripts/git branch; writes `slug-proposed-<tag>`; optional `PAIR_SLUG_*`, `OPENAI_API_KEY`. | go-subcommand `pair slug`; legacy binary retained during #76 | P1 |
./atlas/go-migration-inventory.md:57:| `bin/pair-context` / `cmd/pair-context` / `cmd/internal/contextcmd` | Go binary plus shared runner | `bin/pair-title.sh`; development-only `pair-go context` | `pair-context <tag> <agent>` and `pair-go context <tag> <agent>` print the same humanized token count or nothing; tolerant exit 0 on failure. | Reads `PAIR_DATA_DIR`, `pane-<tag>-<agent>.json`, config, native transcripts. | implemented helper route in `pair-go context`; legacy binary retained while title poller calls it | P1 |
./atlas/go-migration-inventory.md:58:| `bin/pair-scrollback-render` / `cmd/pair-scrollback-render` / `cmd/internal/scrollbackcmd` | Go binary plus shared runner | `bin/pair-scrollback-open`, `bin/pair-changelog-open`, `nvim/scrollback.lua` refresh; development-only `pair-go scrollback-render` | `pair-scrollback-render [--plain] [--max-lines N] [--with-timestamps] raw events out` and `pair-go scrollback-render ...`; nonzero on render/write failure. | Reads `.raw` and `.events.jsonl`; atomically writes `.ansi` or cleaned text. | implemented helper route in `pair-go scrollback-render`; legacy binary retained for shell/Lua callers | P0 |
./atlas/go-migration-inventory.md:59:| `bin/pair-changelog` / `cmd/pair-changelog` | Go binary | `bin/pair-changelog-open` | `pair-changelog --cleaned F --log F --anchor F [--agent A] [--model M]`; exits nonzero on required read/model/write failure. | Reads cleaned scrollback/log/anchor; calls agent model through internal model runner; atomically writes log and anchor. | go-subcommand `pair changelog`; legacy binary retained for opener | P1 |
./atlas/go-migration-inventory.md:60:| `bin/pair-continuation` / `cmd/pair-continuation` | Go binary | nvim compaction prompt instructions, operator/agent shell | `pair-continuation --slug S --agent A --issues CSV --body-file F [--repo-root R ...]`; writes and commits continuation; nonzero on validation/git failure. | Reads body/stdin, git repo state; writes `workshop/continuation/*.md`; runs git commit/push. | go-subcommand `pair continuation`; legacy binary retained for agent instructions until docs change | P1 |
./atlas/go-migration-inventory.md:61:| `bin/pair-scribe` / `cmd/pair-scribe` | Go binary | user shell rc outside Pair sessions | `pair-scribe -log PATH -- CMD [ARGS...]`; long-running PTY wrapper; SIGUSR1 pauses log, SIGUSR2 resumes. | Writes typescript log; wraps child PTY; independent of `PAIR_*`. | go-subcommand candidate is low value; may remain separate installed helper or become `pair scribe` with alias | P2 |
./atlas/go-migration-inventory.md:66:| `bin/pair-scrollback-open` | POSIX shell orchestrator | zellij Alt+/ Run, nvim Alt+b jump | `pair-scrollback-open [--jump prev|next]`; opens read-only nvim viewer; singleton lock. | Requires `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_HOME`; calls renderer, zellij IPC, nvim; writes `.ansi`, `.viewport`, lock. | shell-glue now; candidate Go orchestration after entrypoint, while `nvim/scrollback.lua` remains native | P1 |
./atlas/go-migration-inventory.md:67:| `nvim/scrollback.lua` | Neovim native asset | `bin/pair-scrollback-open` | Loaded by `nvim -u ... <ansi>`; interactive read-only viewer; refreshes backing render. | Reads Pair env and `.ansi`; may call `pair-scrollback-render`; writes pending marker files. | native-asset, adjacent/embedded | P0 |
./atlas/go-migration-inventory.md:68:| `bin/pair-changelog-open` | POSIX shell orchestrator | zellij Alt+l Run | Opens changelog viewer and starts detached render/distill singleton. | Requires Pair env; calls renderer, `pair-changelog`, setsid/perl, nvim; reads/writes `changelog-*` sidecars. | shell-glue now; candidate Go orchestration after entrypoint | P1 |
./atlas/go-migration-inventory.md:69:| `nvim/changelog.lua` | Neovim native asset | `bin/pair-changelog-open` | Loaded by `nvim -u ... <log>`; read-only watcher/spinner. | Reads `PAIR_CHANGELOG_*` and Pair env. | native-asset, adjacent/embedded | P1 |
./atlas/go-migration-inventory.md:70:| `bin/pair-title.sh` | Bash stateful poller | `bin/pair` ensure_title_poller | `pair-title.sh <tag> <agent>`; long-running 60s poller; test hook `PAIR_TITLE_TEST_CALL`. | Reads/writes title pid, pane json, cmux owner files; calls `pair-context`, zellij, ps, cmux. | stateful shell-glue; explicit #78 candidate | P1 |
./atlas/go-migration-inventory.md:71:| `bin/pair-session-watch.sh` / `cmd/pair-session-watch` / `cmd/internal/sessionwatch` | Shell compatibility shim plus Go stateful watcher | `bin/pair` create path | `pair-session-watch.sh <agent> <tag> <cwd> [agent-args...]` execs the Go command; background 60s watcher; no-op for claude. | Reads agent pidfile, lsof/ps, native session dirs; writes config JSON atomically; logs adapt events through `cmd/internal/adapt`. | Go-owned watcher with legacy shim retained while `bin/pair` calls the `.sh` name (#78) | P1 |
./atlas/go-migration-inventory.md:75:| `nvim/doctor.lua` | Lua helper | `:PairDoctor` in nvim | Builds agent instruction payload. | Reads `PAIR_HOME`; sends text through draft/agent flow. | native-asset | P2 |
./atlas/go-migration-inventory.md:76:| `bin/pair-notify` | Bash notification helper | agent hooks/manual shell inside Pair | `pair-notify [--osc 9|777] "message"`; writes OSC to outer tty; nonzero on bad args/missing tty. | Requires `PAIR_TAG`; reads `outer-tty-<tag>`. | small shell-glue; possible Go subcommand but low packaging impact | P2 |
./atlas/go-migration-inventory.md:77:| `bin/pair-quit.sh` | Bash keybind helper | nvim `PairConfirmQuit` | Touch quit marker then kill zellij session. | Uses `ZELLIJ_SESSION_NAME`, `PAIR_KILL_CMD`; writes cache marker. | small compat shell; can fold into Go/nvim flow after entrypoint | P2 |
./atlas/go-migration-inventory.md:78:| `bin/pair-restart.sh` | Bash keybind helper | nvim restart confirmations | Writes restart marker then kill zellij session; supports `--new-session`. | Uses `PAIR_TAG`, `PAIR_AGENT`, `ZELLIJ_SESSION_NAME`, cache marker files. | small compat shell; can fold after entrypoint | P2 |
./atlas/go-migration-inventory.md:79:| `bin/pair-help` | Bash helper | zellij Alt+h Run | Displays `pair -h` through `less` with escape-to-quit behavior. | Calls `pair`, `less`. | compat-shim; may become `pair help` behavior | P2 |
./atlas/go-migration-inventory.md:80:| `bin/clipboard-to-pane.sh` | Bash copy/paste helper | `copy-on-select.sh`, direct zellij run possible | Reads OS clipboard, stages quote, focuses nvim, triggers Lua paste. | Uses pbpaste/wl-paste/xclip, jq, zellij, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_HOME`; writes quote and debug log. | shell-glue; keep until zellij copy flow has Go owner | P2 |
./atlas/go-migration-inventory.md:81:| `bin/copy-on-select.sh` | Bash copy_command helper | `zellij/config.kdl` `copy_command` | Reads selected text stdin, mirrors OS clipboard, flashes source, delegates paste unless selection was in nvim. | Uses pbcopy/wl-copy/xclip, jq, zellij, `PAIR_HOME`; calls flash and clipboard scripts. | shell-glue tied to zellij native surface | P2 |
./atlas/go-migration-inventory.md:83:| `bin/pair-review-open` | POSIX shell review helper | nvim review flow | Validates target and opens floating `nvim -u nvim/review.lua`. | Requires Pair env; calls zellij/nvim. | shell-glue, review workbench can move later if packaging needs it | P2 |
./atlas/go-migration-inventory.md:84:| `bin/pair-review-readiness` | POSIX shell review helper | `nvim/init.lua` review readiness | Emits readiness data from git and target helper. | Uses `PAIR_HOME`, `PAIR_REVIEW_TARGET_BIN`, git/jq. | shell-glue; possible later Go helper | P2 |
./atlas/go-migration-inventory.md:85:| `bin/pair-review-target` | Bash review helper | review readiness/open/tests | Emits JSON target metadata under data dir. | Requires `PAIR_DATA_DIR`; reads config/pid files/lsof; writes `review-target-<tag>.json`. | shell-glue; possible #78 candidate if review packaging matters | P2 |
./atlas/go-migration-inventory.md:100:  "$PAIR_HOME/nvim/init.lua"` by absolute env path.
./atlas/go-migration-inventory.md:117:- `make build` builds `GO_BINS` into `bin/`; `pair` and `pair-go` are both built
./atlas/go-migration-inventory.md:118:  from `cmd/pair-go` with `defaultPairHome=$(CURDIR)`.
./atlas/go-migration-inventory.md:119:- `make install` copies `GO_BINS` to `~/.local/bin` and symlinks only retained
./atlas/go-migration-inventory.md:124:  `defaultPairHome=#{libexec}`.
./atlas/go-migration-inventory.md:130:- #74 should add a Go dispatcher without changing `bin/pair`.
./atlas/go-migration-inventory.md:138:- #77 made `pair-go launch ...` a meaningful Go-owned compatibility handoff to
./atlas/go-migration-inventory.md:139:  `bin/pair`, with argv/env preserved and missing-launcher diagnostics.
./atlas/go-migration-inventory.md:141:  `bin/pair-session-watch.sh` retained as a shim. `pair-title.sh` remains the
./atlas/go-migration-inventory.md:144:- #79 made public `pair` a Go-built entrypoint, renamed the shell launcher to
./atlas/go-migration-inventory.md:145:  `bin/pair-shell`, and chose adjacent `nvim/` / `zellij/` assets for local and
./atlas/go-migration-inventory.md:165:- `bin/pair`
./atlas/go-migration-inventory.md:166:- `bin/pair-changelog`
./atlas/go-migration-inventory.md:167:- `bin/pair-changelog-open`
./atlas/go-migration-inventory.md:168:- `bin/pair-context`
./atlas/go-migration-inventory.md:169:- `bin/pair-continuation`
./atlas/go-migration-inventory.md:170:- `bin/pair-dev`
./atlas/go-migration-inventory.md:171:- `bin/pair-help`
./atlas/go-migration-inventory.md:172:- `bin/pair-notify`
./atlas/go-migration-inventory.md:173:- `bin/pair-quit.sh`
./atlas/go-migration-inventory.md:174:- `bin/pair-restart.sh`
./atlas/go-migration-inventory.md:175:- `bin/pair-review-open`
./atlas/go-migration-inventory.md:176:- `bin/pair-review-readiness`
./atlas/go-migration-inventory.md:177:- `bin/pair-review-target`
./atlas/go-migration-inventory.md:178:- `bin/pair-scribe`
./atlas/go-migration-inventory.md:179:- `bin/pair-scrollback-open`
./atlas/go-migration-inventory.md:180:- `bin/pair-scrollback-render`
./atlas/go-migration-inventory.md:181:- `bin/pair-session-watch.sh`
./atlas/go-migration-inventory.md:182:- `bin/pair-slug`
./atlas/go-migration-inventory.md:183:- `bin/pair-title.sh`
./atlas/go-migration-inventory.md:184:- `bin/pair-wrap`
./atlas/architecture.md:16:bin/pair                     # Go public entrypoint; execs bin/pair-shell during migration
./atlas/architecture.md:17:bin/pair-shell               # retained shell launcher: zellij lifecycle + prompt UI
./atlas/architecture.md:18:bin/pair-go                  # explicit Go dispatcher + launch handoff
./atlas/architecture.md:21:bin/pair-quit.sh             # invoked by Alt+x — marks + kills session
./atlas/architecture.md:22:bin/pair-restart.sh          # invoked by Alt+n / Shift+Alt+N — marks (quit + restart) + kills session
./atlas/architecture.md:23:bin/pair-session-watch.sh    # shim to Go watcher that captures codex/agy session id (#000016, #000020, #78)
./atlas/architecture.md:24:bin/pair-wrap                # PTY proxy: OSC translation + scrollback capture
./atlas/architecture.md:25:bin/pair-notify              # hook-driven OSC notifier (e.g. claude Notification)
./atlas/architecture.md:26:bin/pair-scrollback-render   # raw PTY capture → ANSI-colored line dump (#000017)
./atlas/architecture.md:27:bin/pair-scrollback-open     # Alt+/ orchestrator: render + open viewer
./atlas/architecture.md:28:bin/pair-changelog           # TTY → distilled change log (LLM, incremental) (#53)
./atlas/architecture.md:29:bin/pair-changelog-open      # Alt+l orchestrator: open viewer; clean+distill run in the background (#53)
./atlas/architecture.md:49:As of #79, the public `bin/pair` command is a Go-built entrypoint from
./atlas/architecture.md:50:`cmd/pair-go`. Direct `pair ...` and explicit `pair-go launch ...` share one
./atlas/architecture.md:52:`<asset-root>/bin/pair-shell` with `pair`-compatible argv/env. Asset root
./atlas/architecture.md:53:resolution is ordered: explicit `PAIR_HOME`, executable sibling root, then the
./atlas/architecture.md:54:build-time `defaultPairHome` injected by Make/Homebrew for copied installs.
./atlas/architecture.md:58:launch testing: `pair-go launch claude`, `pair-go launch resume <tag>`,
./atlas/architecture.md:59:`pair-go launch continue ...`, `pair-go launch list`, and
./atlas/architecture.md:60:`pair-go launch rename ...` all reach the same shell launcher as direct `pair`.
./atlas/architecture.md:67:behavior remain shell-owned through `bin/pair-shell` until later migration
./atlas/architecture.md:74:the old command names remain the live integration surface. `bin/pair-title.sh`,
./atlas/architecture.md:75:`bin/pair-scrollback-open`, `bin/pair-changelog-open`, and
./atlas/architecture.md:84:merge-safe: after any sub-issue lands, the public `pair` command, `pair-dev`,
./atlas/architecture.md:89:### `bin/pair` / `bin/pair-shell` — launcher
./atlas/architecture.md:91:`bin/pair` is the Go public entrypoint. It resolves the asset root, then execs
./atlas/architecture.md:92:`bin/pair-shell` with argv[0] presented as `pair`; the shell script below still
./atlas/architecture.md:95:`bin/pair-shell` resolves `$PAIR_HOME` from its own real path (portable bash, no `readlink -f`), prepends `$PAIR_HOME/bin` to `$PATH` (idempotent across re-launches) so all helper scripts resolve by bare name in zellij configs and keybinds, parses argv — first positional is `$PAIR_AGENT` (default `claude`), everything after `--` is joined into `$PAIR_AGENT_ARGS`, extra positionals before `--` are an error with a usage hint, defaults `$PAIR_TAG` to the cwd basename (the create-flow prompt or `pair resume <tag>` overrides it), resolves `$PAIR_DATA_DIR` to `${XDG_DATA_HOME:-$HOME/.local/share}/pair`, runs a one-time migration of any old `~/scratch/pair-{draft,log}-*` files, and dispatches:
./atlas/architecture.md:112:**Title poller (`bin/pair-title.sh`) — two surfaces.** A single always-on per-tag 60s background poller, spawned via `ensure_title_poller` on *every* entry (create, attach, restart) so a poller a host sleep/reboot/SIGKILL killed is reliably revived. Single-instance guard is identity-checked (`poller_alive` `ps`-matches the command line for this tag; pidfile `$DATA_DIR/title-pid-<tag>`; not a bare `kill -0`) so a recycled PID can't suppress the respawn. It owns two title surfaces (tested in `tests/pair-title-poller-test.sh`):
./atlas/architecture.md:130:**Reload / restart in place (Alt+n, Shift+Alt+N).** A second marker, `~/.cache/pair/restart-<session>`, is written alongside `quit-` by `bin/pair-restart.sh`, carrying the agent name + a `new_session` flag. After cleanup_quit_marker tears the session down, `handle_restart_marker` reads the marker and `exec`s pair on itself with `PAIR_FORCE_TAG=<same-tag>` set in the env (pins the new run to the killed session's tag, skipping both the picker and the name prompt). The flag controls what happens to the saved config:
./atlas/architecture.md:135:The picker is bypassed in either flavor — Alt+n's argv carries an explicit resume token, and Shift+Alt+N has no saved config to pick against. A third marker field, `continue=<slug>` (#55, written by the in-session compaction branch in `bin/pair`, not `pair-restart.sh`), rides the `new_session=1` path but re-execs `pair continue <slug> <agent> -- <args>` instead of a plain restart — see "In-session compaction" below.
./atlas/architecture.md:139:Horizontal split. Top pane runs `$PAIR_AGENT $PAIR_AGENT_ARGS` (auto-fills remaining height). Bottom pane is `size=12` (fixed 12 rows) running `nvim -u $PAIR_HOME/nvim/init.lua` on the per-tag draft file. Integer sizes are FIXED in zellij (refusing the `resize` action), but pair drives all rung changes through swap layouts, not resize, so FIXED is harmless.
./atlas/architecture.md:141:Both panes wrap their command in `sh -c "..."` so the shell expands `$PAIR_AGENT`, `$PAIR_AGENT_ARGS`, `$PAIR_TAG`, and `$PAIR_HOME` at exec time — zellij itself does not interpolate env vars in `command`/`args` fields.
./atlas/architecture.md:156:- `copy_command "copy-on-select.sh"` — on every selection finalize (mouse-up after drag), zellij pipes the selected text to this script. `copy_command` replaces zellij's default OS-clipboard write, so the script does that part too. Resolved by PATH (which `bin/pair` populated).
./atlas/architecture.md:167:- `Shift+Alt+N` — routed through nvim to `:lua PairConfirmRestartNewSession()` — Y/N modal then `pair-restart.sh --new-session` (restart with a fresh agent conversation). See "Reload / restart in place" under `bin/pair`.
./atlas/architecture.md:168:- `Alt+Shift+C` (`Alt C` / `Ctrl Alt c`) — routed through nvim to `:lua PairConfirmCompact()` — Y/N modal then `send_to_agent(<compaction prompt>)` (#55). Unlike the restart binds it does NOT shell out directly: distilling a continuation needs the agent's judgment, so it asks the agent to write a continuation + run `pair continue <slug>` (context-aware → compaction). See "In-session compaction" under `bin/pair`.
./atlas/architecture.md:196:### `bin/pair-quit.sh` — Alt+x handler
./atlas/architecture.md:198:Touches the marker file `~/.cache/pair/quit-$ZELLIJ_SESSION_NAME`, then `exec zellij kill-session $ZELLIJ_SESSION_NAME`. The kill terminates the session including the script itself; on the launcher side, `bin/pair` resumes, sees the marker, and runs `delete-session --force` to clean up the resurrect entry.
./atlas/architecture.md:202:### Outer-TTY capture and notification routing — `bin/pair-wrap`, `bin/pair-notify`
./atlas/architecture.md:208:1. **Outer-TTY capture (in `bin/pair`).** Before invoking zellij, on every attach (both create and reattach branches), pair calls `tty(1)`. The result is the path of pair's controlling TTY — which is precisely the outer PTY (the one allocated by whatever wraps pair: cmux, a terminal emulator, etc.). That path gets written to `$DATA_DIR/outer-tty-<tag>`. Refreshed on every attach because the outer PTY changes across detach/reattach, while pane-shell env stays frozen at zellij session-creation time (env-var approaches would go stale).
./atlas/architecture.md:212:   - **`bin/pair-wrap`** (Go, `cmd/pair-wrap`). Transparent PTY proxy. The zellij agent pane runs `pair-wrap $PAIR_AGENT $PAIR_AGENT_ARGS` instead of the agent directly (wired in `zellij/layouts/main.kdl`). The wrapper allocates a fresh PTY for the agent, forwards stdin/stdout transparently with SIGWINCH propagation, and watches the agent's output stream for OSC notifications. On detection it writes OSC 9 directly to the recorded outer-TTY path — bypassing zellij.
./atlas/architecture.md:255:   - **`bin/pair-notify`** (bash). Hook-driven helper for richer signals. `pair-notify [--osc 9|777] "msg"` reads the same outer-TTY file and writes the OSC. Intended for Claude Code `Notification`/`Stop` hooks where you want semantic events with custom message text rather than relying on the agent's native OSC stream.
./atlas/architecture.md:274:**Replay (`bin/pair-scrollback-render`, Go).** Reads `<raw>` and `<events.jsonl>`, feeds the bytes to a `charmbracelet/x/vt` emulator in a single offset-ordered walk over all events (`feedSegments`): write up to each offset, then `Resize` on a resize event or snapshot `Scrollback().Len()` on a `time` event (#59). The emulator runs the same VT100 interpretation zellij does live (width-based wrap, alternate-screen flips, scroll regions), so its row count matches what the user saw in zellij's indicator. After feeding, the renderer walks the scrolled-out history followed by the visible buffer, and emits one ANSI-decorated line per row to `<out.ansi>`: full-reset SGR + per-row attrs + the row's characters + `\x1b[0m`. With `--with-timestamps` (the change-log path only — never the Alt+/ viewer) the pure `interleaveDateMarkers` then inserts `⟦pair:ts DATE⟧` lines at each day boundary from the time snapshots (#59). Built into `bin/pair-scrollback-render` via `make pair-scrollback-render`; single static binary, no runtime dep. Its raw inputs live in `$PAIR_DATA_DIR` as `scrollback-<tag>-<agent>.{raw,events.jsonl}` (RAW VT bytes, NOT in the repo); `:PairTTYRawPath` / `_G.PairTTYRawPath()` (nvim, #56) prints the current session's live `.raw` path on demand and copies it to the `+` register — useful for grabbing the byte stream mid-session, since an Alt+x quit deletes it unless preserved.
./atlas/architecture.md:280:**In-session compaction (#55, `Alt+Shift+C`).** `pair continue <slug>` is *context-aware*: run from a normal shell it fresh-starts (above); run from INSIDE its own live pane it **compacts** — copy-parks the scrollback as a recovery net, writes a restart marker carrying a new `continue=<slug>` field (same tag, `new_session=1`), and kills the session. The outer `bin/pair` then `handle_restart_marker`-re-execs `pair continue <slug> <agent> -- <args>` (now outside zellij → the fresh-start branch), so the session reincarnates under the same tag with a clean conversation seeded from the continuation. Detection is **ancestry-based** (`in_zellij_pane`) plus a `ZELLIJ_SESSION_NAME == pair-<tag>` confirmation — never `$ZELLIJ*` env alone, since cmux propagates those to sibling non-pair panes (a false positive would park+kill the wrong session). The branch sits *before* the `in_zellij_pane` guard (which otherwise rejects any in-pane `pair`), with `DATA_DIR` + `handle_restart_marker` hoisted above the guard so test seams can drive them. Seams (test-only): `PAIR_FORCE_IN_SESSION`, `PAIR_FAKE_IN_ZELLIJ`, `PAIR_KILL_CMD`, `PAIR_TEST_CALL`, `PAIR_REEXEC_CAPTURE`. The trigger is the `Alt+Shift+C` keybind (`Alt C` / `Ctrl Alt c` → `PairConfirmCompact` → an agent-agnostic prompt that **defers to the `continuation` datatype procedure** — flush-first, then write the continuation and run `pair continue <slug>` — rather than enumerating a section skeleton inline, so the prompt can't drift out of sync with the datatype; that drift was the bug pair#61 fixed); the outer process suppresses the Alt+x park nudge whenever a restart marker is pending (a restart isn't a quit).
./atlas/architecture.md:294:**Open (`bin/pair-scrollback-open`, POSIX sh).** Validates `PAIR_DATA_DIR` / `PAIR_TAG` / `PAIR_AGENT`, runs the renderer, then *launches* `nvim -u $PAIR_HOME/nvim/scrollback.lua $ANSI` as a child — deliberately **not** `exec`, so the script stays alive as nvim's parent and an `EXIT`/`INT`/`TERM` trap can clear the re-entrancy lock on quit. Errors print and `sleep` briefly so the message is readable before the floating pane self-closes. Bound in `zellij/config.kdl` to `Alt+/` as a 100% × 100% floating pane with `close_on_exit=true` — the user's `:q` in the viewer dismisses the pane and returns to pair's two-pane layout untouched. **Re-entrancy guard:** `Alt+/` is a global zellij bind, so pressing it again while the viewer is already focused fires another `Run` and would stack a second nvim (one `:q`/Esc per layer to unwind). zellij can't conditionally skip a `Run`, so the script self-guards: before launching nvim it writes its own PID to `$PAIR_DATA_DIR/scrollback-<tag>-<agent>.openlock`, and on entry it exits immediately if that lock already holds a *live* PID — the redundant floating pane then self-dismisses via `close_on_exit` and focus falls back to the open viewer. A stale lock (hard kill) carries a dead PID and is reclaimed by the next open's `kill -0` check. The draft pane's `Alt+b` (`--jump prev`) runs the same launcher, so it's covered too.
./atlas/architecture.md:330:- All autocmds live in the `pair` augroup (`clear=true`), so iterating via `:luafile $PAIR_HOME/nvim/init.lua` reloads cleanly without duplicating handlers.
./atlas/architecture.md:457:  `bin/pair-changelog-open` in a floating pane. It opens `nvim -u
./atlas/architecture.md:535:  (the opener `bin/pair-changelog-open` and the draft-nvim `.ready` watcher):
./atlas/architecture.md:536:  the exported `PAIR_SESSION_ID` (set by `bin/pair` at launch for claude-fresh /
./atlas/architecture.md:576:1. **Pre-write at launch (`bin/pair`).** Two paths:
./atlas/architecture.md:579:2. **Watcher (`cmd/pair-session-watch` via `bin/pair-session-watch.sh`, codex/agy only).** Spawned in the background by `bin/pair` on the create path, right before the zellij launch. The `.sh` file is now a compatibility shim; the stateful discovery logic lives in Go. Two discovery paths:
./atlas/architecture.md:591:| claude | `~/.claude/projects/<encoded-cwd>/<id>.jsonl` | filename | `--session-id` pre-injected by `bin/pair` (deterministic) |
./atlas/architecture.md:603:**Create-flow prompt (`bin/pair`).** When the create path commits a tag, pair reads `config-<tag>-<agent>.json`. If present, it runs the per-agent stale-id check (claude: `[ -f .../<id>.jsonl ]`; codex: `find ~/.codex/sessions -name "*<id>*"`; agy: check conversation DB) and fzf-prompts the user with up to three options:
./atlas/architecture.md:645:**Inside-flow choreography.** `nvim/init.lua`'s `pair_confirm_restart_impl` shells out `pair rename --restart-check` after the user enters a new tag, re-prompting on each rejection. On accept it execs `pair-restart.sh --rename-to <new>`. `pair-restart.sh` writes `rename_to=<new>` into the restart marker (`~/.cache/pair/restart-<SESSION>`) alongside the existing `tag`, `agent`, `new_session` fields. `handle_restart_marker` in `bin/pair` runs after `cleanup_quit_marker` (so the zj delete-session has cleared the live-old gate) and if `rename_to` is set, invokes `"$0" rename <old> <new>` — full check. On success, the working tag for the re-exec is swapped to `<new>` (so `config-<new>-<agent>.json`, the just-renamed file, is what gets resumed). On failure, a 2-second visible stderr warning is printed and the restart continues with the original tag — the user is never stranded.
./atlas/architecture.md:667:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-<tag>` — single-line file recording which agent binary was launched in the session (`claude`, `codex`, ...). Written once at session create; read by `pair list` to display the agent column, and by `bin/pair`'s tag-restart agent-inference. Removed on full quit. The agent isn't otherwise recoverable post-create — env vars are frozen in pane shells, and custom session names (e.g. `pair-bugfix`) don't carry the agent in the name.
./atlas/architecture.md:669:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/config-<tag>-<agent>.json` — saved restart configuration for `(tag, agent)` (issue #000016, #000020). `{ agent, args, session_id }`. For claude, written synchronously by `bin/pair` before zellij launch (`--session-id` is deterministic). For codex/agy, written by the Go `pair-session-watch` command once the agent's session file is discovered via lsof. Read by `bin/pair`'s create-flow prompt and by the post-Alt+x hint. Survives Alt+x (unlike `agent-<tag>`, which is cleared) — that's the whole point: it's the bridge between two pair launches against the same tag.
./atlas/architecture.md:673:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/nvim-pid-<tag>-{draft,scrollback}` — single-line file containing the pid of an `nvim --embed` server child. Written at VimEnter by `nvim/init.lua` (for the draft pane) and `nvim/scrollback.lua` (for the Alt+/ floating viewer) when `$PAIR_NVIM_PID_FILE` is set; the launch sites (`zellij/layouts/main.kdl` for draft, `bin/pair-scrollback-open` for scrollback) export the env var pointing at a tag-scoped path. Read and removed by `cleanup_quit_marker` on Alt+x to SIGKILL the embed deterministically — without this, the embed sometimes survives zellij's pane teardown and accumulates as a PPID=1 orphan, dragging the host into memory pressure across many quits.
./atlas/architecture.md:675:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/pair-wrap-pid-<tag>` — single-line file containing pair-wrap's pid, written at startup by `bin/pair-wrap` if `PAIR_TAG` is set. Read by nvim's Alt+i (`attach_image`) so it can `kill -USR1 <pid>` to arm an image-capture window. Removed by pair-wrap on exit (the `finally` block in `main()`) and by `cleanup_quit_marker` as belt-and-suspenders on Alt+x.
./atlas/architecture.md:681:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/adapt-<tag>.jsonl` — the adaptation flight recorder (issue #000045). One JSON line per harness-adaptation trigger (`{ts, comp, agent, aspect, signal, outcome, detail}`), appended concurrently by `cmd/pair-wrap`, `cmd/pair-slug`, `cmd/pair-session-watch`, and `nvim/adapt.lua` — all writing one shared schema. Truncated once at session launch by `bin/pair` (so multi-process `O_APPEND` never races) and removed on full quit. Read by `doctor/doctor.sh` to surface integration drift (near-miss/fail signals). See `atlas/how-to-bring-up-a-new-harness-cli.md` §3 for the signal registry.
./atlas/architecture.md:687:`bin/pair` prepends `$PAIR_HOME/bin` to `$PATH` before exec'ing zellij. zellij and all its child processes (panes, copy_command, Run actions) inherit the PATH and can resolve `clipboard-to-pane.sh`, `copy-on-select.sh`, `pair-quit.sh` by bare name. This lets the zellij KDL configs reference scripts without `sh -c` env-var quoting hacks.
./atlas/architecture.md:691:The Go binaries (`pair-wrap`, `pair-slug`, …) live in `$PAIR_HOME/bin` (first on PATH per *Path resolution* above) and, after `make install`, in `~/.local/bin`. `bin/` is **gitignored** — built on demand, absent in a fresh tree. Because the agent pane launches as `sh -c '… exec pair-wrap …'`, the wrapper is resolved by a **PATH lookup**: no shell function or `.zshenv` can intercept it (`exec` bypasses functions, and `sh` ≠ zsh), so `construct/dev-aliases.sh`'s rebuild-on-call freshness does **not** reach it. When `$PAIR_HOME/bin/pair-wrap` is stale or absent, PATH silently falls through to an old `~/.local/bin` copy and the running wrapper drifts from source — the failure mode is *silence*, not an error (diagnosed once via the #000045 flight recorder going quiet for every Go-emitted aspect while only nvim's Lua emitter still logged).
./atlas/architecture.md:695:- **Dev** — `pair-dev` (#000046). Exports `PAIR_DEV=1` and execs `pair`; `bin/pair` then runs `make build` (via `bin/lib/dev-rebuild.sh`'s `dev_rebuild`) on the **create path**, before the layout execs pair-wrap, so `$PAIR_HOME/bin` holds a fresh build. Restart-safe: `PAIR_DEV` survives `handle_restart_marker`'s `exec "$0"`, so Alt+n / Shift+Alt+N rebuild too; a plain attach (no new wrapper spawned) correctly skips it. Deployed launches (`PAIR_DEV` unset) invoke no toolchain.
./atlas/architecture.md:701:`cmd/pair-scribe` is a `script(1)` replacement that lives in the pair repo for build-system convenience but is not part of pair's runtime — it's user shell tooling, typically wired at the top of `~/.zshrc` to swap for `script -q -F`. The user's preexec/precmd hooks send `SIGUSR1`/`SIGUSR2` to pause/resume the on-disk typescript around commands whose output (e.g. TUI redraws) shouldn't be captured, enabling a clean "capture last command output" flow that pair can read back from `$_ZSH_SCRIPT_LOG`. Lives at `~/.local/bin/pair-scribe` after `make install`. Full design notes and the zshrc snippet: `cmd/pair-scribe/README.md`.
./cmd/pair-changelog/e2e_test.go:59:	// Real render step (as bin/pair-changelog-open invokes it).
./workshop/lessons.md:83:`bin/pair-scrollback-render` initially opened `<out.ansi>` with `'w'` (truncate-then-write). Two `Alt+/` presses in quick succession would race on the same path; whichever finished second left a half-interleaved file for nvim to open. Fixed by writing to `<out.ansi>.tmp` and `os.replace()`-ing at the end.
./workshop/lessons.md:89:Two bugs in v1 of `bin/pair` and `zellij/config.kdl` came from going off memory of zellij's API:
./workshop/lessons.md:164:via `lsof -p <pid> | awk '$4=="txt"`) was `~/.local/bin/pair-wrap` dated days
./workshop/lessons.md:336:   `bin/pair`, but `bin/pair` turned out to be gitignored AND untracked: a normal
./workshop/lessons.md:454:`bin/pair` parser what it resolved (`AGENT`, `FORCED_TAG`, forwarded args,
./workshop/lessons.md:457:pane, `in_zellij_pane` returned true and `bin/pair` exited with "already running
./workshop/lessons.md:485:agent in `launcher.ParseArgs`, but the dispatcher intercepted `pair-go launch`
./workshop/continuation/20260612T002626-compact.md:22:  - **M1** (`bin/pair` mechanics: `park_scrollback` copy|move, ancestry-gated in-session branch, `handle_restart_marker` `continue=` re-exec) — **landed** (9b9e0a1), 21/21 `make test-continue`, boundary review **FIX-THEN-SHIP** (the one Important fixed: park-nudge suppressed when a restart marker is pending).
./workshop/continuation/20260612T002626-compact.md:36:- Key files: `bin/pair` — in-session compaction branch (~863–905, placed **before** the `in_zellij_pane` guard), `park_scrollback` (~687), `handle_restart_marker` (~707); `zellij/config.kdl` (`bind "Alt C" "Ctrl Alt c"`); `nvim/init.lua` (`PairConfirmCompact`, ~696–741). Tests: `tests/pair-continue-test.sh` (seams `PAIR_FORCE_IN_SESSION` / `PAIR_FAKE_IN_ZELLIJ` / `PAIR_KILL_CMD`), `make test-continue`.
./doctor/SKILL.md:58:- The log truncates at each session launch (`bin/pair`), so it reflects the
./workshop/continuation/20260621T180406-pair-pair.md:54:- `nvim/review/readiness.lua`, `bin/pair-review-readiness`, `nvim/review/seam.lua`, `nvim/init.lua` around `:PairReview` and `PairReviewToggle`, and `tests/review-toggle-test.sh` / `tests/review-resume-test.sh` for M4a'.
./nvim/doctor_test.lua:4:-- $PAIR_HOME-absolute substitution and graceful nil-on-unset.
./nvim/doctor_test.lua:19:ok(type(p) == 'string', 'payload returns a string for a real PAIR_HOME')
./nvim/doctor_test.lua:23:ok(p:find('$PAIR_HOME', 1, true) == nil, 'payload has no literal $PAIR_HOME (substituted, not deferred)')
./nvim/doctor_test.lua:25:-- Trailing slash on PAIR_HOME must not double up in the paths.
./tests/review-window-test.sh:43:( cd "$RT" && PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" RESULT="$RT/r1" \
./tests/review-window-test.sh:48:if PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-open" "$RT/nope.md" 2>/dev/null; then
./tests/review-window-test.sh:54:PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-open" "$RT/doc.md" || true
./tests/review-window-test.sh:82:    PAIR_HOME="$ROOT" DOCFLOW_BIN="$ROOT/tests/lib/fake-docflow.sh" DOCFLOW_ARGLOG="$RT/doclog" \
./tests/review-window-test.sh:106:  local apply = dofile(vim.env.PAIR_HOME .. '/nvim/review/apply.lua')
./tests/review-window-test.sh:324:    PAIR_HOME="$ROOT" DOCFLOW_BIN="$ROOT/tests/lib/fake-docflow.sh" DOCFLOW_ARGLOG="$RT/doclog" RESULT2="$RT/r3" \
./cmd/internal/dispatcher/dispatcher.go:31:		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "handoff"},
./cmd/internal/dispatcher/dispatcher.go:53:			Stdout:   "pair-go dispatcher skeleton\nlaunch handoff: bin/pair\n",
./cmd/internal/dispatcher/dispatcher.go:91:		Stderr:   "pair-go launch is a process handoff implemented by cmd/pair-go; call pair-go launch ... instead of dispatcher.Dispatch\n",
./cmd/internal/dispatcher/dispatcher.go:100:	b.WriteString("Development dispatcher skeleton. Public sessions still start through bin/pair.\n\n")
./cmd/internal/dispatcher/dispatcher.go:106:			fmt.Fprintf(&b, "  %-17s %s (compatibility handoff to bin/pair)\n", family.Name, family.Summary)
./tests/pair-title-poller-test.sh:2:# Regression + behavior test for bin/pair-title.sh.
./tests/pair-title-poller-test.sh:23:POLLER="$ROOT/bin/pair-title.sh"
./tests/pair-title-poller-test.sh:103:cat > "$RT/bin/pair-context" <<'EOF'
./tests/pair-title-poller-test.sh:108:chmod +x "$RT/bin/zellij" "$RT/bin/pair-context"
./workshop/continuation/20260615T225403-cont-improve.md:42:- **Key files:** `../ariadne/construct/datatype/continuation.md` — the rewritten datatype, resolved into pair via the `construct/datatype` → `../../ariadne/construct/datatype` **symlink** (so pair gets it for free, no recompose); `nvim/init.lua` `COMPACT_PROMPT` (~2990) — now defers to that datatype, with a comment guarding against re-inlining a skeleton; `bin/pair-continuation` — the writer (enforces only `## NEXT ACTION`; auto-commits+pushes).
./workshop/continuation/20260615T225403-cont-improve.md:49:- **No `bin/pair` seed-prompt change** — the resume directive is embedded in the generated file instead. (decision 5)
./cmd/pair-wrap/update_agent_output_test.go:83:	// The 949aeec bug fix in bin/pair-wrap.py, now ported to the Go
./Makefile.local:11:# to GO_BINS, add the per-binary recipe stanza. `make build` and
./Makefile.local:14:#   make build      → compile every binary in GO_BINS into ./bin/
./Makefile.local:20:# at $PAIR_HOME/bin/pair-scrollback-render to pick up the Go renderer
./Makefile.local:29:GO_BINS     := pair pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-session-watch pair-go
./Makefile.local:30:SHELL_BINS  := pair-dev
./Makefile.local:31:PAIR_HOME_LDFLAGS := -X main.defaultPairHome=$(CURDIR)
./Makefile.local:33:build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
./Makefile.local:37:	@for b in $(GO_BINS); do \
./Makefile.local:41:	@for b in $(SHELL_BINS); do \
./Makefile.local:193:# binary, pair-go launch shares its pair-shell handoff, and copied local
./Makefile.local:205:# Drives the real bin/pair via its PAIR_DEBUG_ARGS probe. See tests/pair-continue-test.sh.
./Makefile.local:209:# changelog — end-to-end smoke for bin/pair-changelog-open (#53 M2): fakes the
./Makefile.local:246:# bin/pair-session-watch.sh remains as a compatibility shim for bin/pair.
./Makefile.local:263:# pair — public Go-owned entrypoint. It execs bin/pair-shell for the current
./Makefile.local:264:# shell-owned zellij lifecycle and uses the build-time defaultPairHome when a
./Makefile.local:267:	go build -ldflags "$(PAIR_HOME_LDFLAGS)" -o $@ ./cmd/pair-go
./Makefile.local:270:# `pair-go launch ...` shares the same bin/pair-shell handoff as public pair;
./Makefile.local:273:	go build -ldflags "$(PAIR_HOME_LDFLAGS)" -o $@ ./cmd/pair-go
./tests/changelog-session-key-test.sh:2:# Focused keying test for bin/pair-changelog-open (#63): the change-log base is
./tests/changelog-session-key-test.sh:9:PAIR_HOME=$(cd "$(dirname "$0")/.." && pwd); export PAIR_HOME
./tests/changelog-session-key-test.sh:25:run() { rm -f "$tmp/nvim-arg"; "$PAIR_HOME/bin/pair-changelog-open"; }
./cmd/internal/dispatcher/dispatcher_test.go:35:			if strings.Contains(res.Stdout, "launch             session lifecycle and public pair launcher flow (planned; not implemented") {
./cmd/internal/dispatcher/dispatcher_test.go:61:	for _, want := range []string{"pair-go", "dispatcher skeleton", "launch handoff: bin/pair"} {
./cmd/internal/dispatcher/dispatcher_test.go:91:	for _, want := range []string{"pair-go launch", "process handoff", "cmd/pair-go"} {
./tests/review-indicator-test.sh:54:PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" \
./cmd/internal/entrypoint/asset_root_test.go:11:		Executable:      "/repo/bin/pair",
./cmd/internal/entrypoint/asset_root_test.go:21:	if root.ShellPath != "/env/root/bin/pair-shell" {
./cmd/internal/entrypoint/asset_root_test.go:22:		t.Fatalf("ShellPath = %q, want /env/root/bin/pair-shell", root.ShellPath)
./cmd/internal/entrypoint/asset_root_test.go:28:		Executable:      "/repo/bin/pair",
./cmd/internal/entrypoint/asset_root_test.go:38:	if root.ShellPath != "/repo/bin/pair-shell" {
./cmd/internal/entrypoint/asset_root_test.go:39:		t.Fatalf("ShellPath = %q, want /repo/bin/pair-shell", root.ShellPath)
./cmd/internal/entrypoint/asset_root_test.go:45:		Executable:      "/home/me/.local/bin/pair",
./cmd/internal/entrypoint/asset_root_test.go:55:	if root.ShellPath != "/repo/bin/pair-shell" {
./cmd/internal/entrypoint/asset_root_test.go:56:		t.Fatalf("ShellPath = %q, want /repo/bin/pair-shell", root.ShellPath)
./cmd/internal/entrypoint/asset_root_test.go:62:		Executable:      "/home/me/.local/bin/pair",
./cmd/internal/entrypoint/asset_root_test.go:69:	for _, want := range []string{"pair-shell", "PAIR_HOME", "/home/me/.local", "/repo"} {
./tests/pair-review-target-test.sh:15:  PAIR_SESSION_ID=envsid "$ROOT/bin/pair-review-target" "$doc" ready >/dev/null
./tests/pair-review-target-test.sh:21:  "$ROOT/bin/pair-review-target" "$doc" ready >/dev/null
./cmd/internal/entrypoint/launch_test.go:9:	req := ResolveLegacyLaunch(AssetRoot{Root: "/repo", ShellPath: "/repo/bin/pair-shell"}, []string{"claude", "--", "--resume"})
./cmd/internal/entrypoint/launch_test.go:10:	if req.Path != "/repo/bin/pair-shell" {
./cmd/internal/entrypoint/launch_test.go:11:		t.Fatalf("Path = %q, want /repo/bin/pair-shell", req.Path)
./cmd/internal/entrypoint/launch_test.go:20:	req := ResolveLegacyLaunch(AssetRoot{Root: "/repo", ShellPath: "/repo/bin/pair-shell"}, []string{"resume", "demo"})
./cmd/internal/entrypoint/launch_test.go:21:	if req.Path != "/repo/bin/pair-shell" {
./cmd/internal/entrypoint/launch_test.go:22:		t.Fatalf("Path = %q, want /repo/bin/pair-shell", req.Path)
./cmd/internal/adapt/adapt.go:17:// atomic across processes; bin/pair truncates the file once at session launch
./tests/changelog-open-test.sh:2:# End-to-end smoke test for bin/pair-changelog-open (#53/#58).
./tests/changelog-open-test.sh:11:PAIR_HOME=$(cd "$(dirname "$0")/.." && pwd)
./tests/changelog-open-test.sh:12:export PAIR_HOME
./tests/changelog-open-test.sh:14:if [ ! -x "$PAIR_HOME/bin/pair-changelog" ] || [ ! -x "$PAIR_HOME/bin/pair-scrollback-render" ]; then
./tests/changelog-open-test.sh:50:"$PAIR_HOME/bin/pair-changelog-open"
./cmd/internal/transcript/transcript.go:18:// config-<tag>-<agent>.json (written by bin/pair / pair-session-watch.sh).
./tests/review-readiness-cli-test.sh:22:out="$(PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-readiness" "$REPO/doc \"quoted\".md")"
./tests/review-readiness-cli-test.sh:38:prep_out="$(PAIR_HOME="$ROOT" PAIR_DATA_DIR="$RT" PAIR_TAG=prep PAIR_SESSION_ID=sid \
./tests/review-readiness-cli-test.sh:39:  "$ROOT/bin/pair-review-readiness" --prepare "$PREP/doc.md" 2>&1 || true)"
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:248:bin/pair                      # launcher
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:281:[Set issue 000001 status to `working`, ran `make issue-sync`. Created directory structure `bin/`, `nvim/`, `zellij/layouts/`. Wrote all six files: `bin/pair`, `bin/clipboard-to-pane.sh`, `nvim/init.lua`, `zellij/config.kdl`, `zellij/layouts/main.kdl`, `README.md`. Plus `atlas/index.md` and `atlas/architecture.md`. Made scripts executable.]
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:286:- `bin/pair nonexistent` correctly exits 1 with "agent not found."
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:311:💬: ~/workspace/pair (main*) » bin/pair
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:486:💬: now, we also need to support multiple bin/pair being issues, potentially on different agnet (claude, codex, gemini). is this supported today
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:508:[Updated `bin/pair` to accept `VARIANT="${2:-}"`. PAIR_TAG = agent-variant. Session, draft, log all derive from PAIR_TAG. Layout uses `${PAIR_TAG:-${PAIR_AGENT:-claude}}` for the draft path.]
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:512:💬: when we ctrl-q quit, can you in bin/pair, to run zellij kill-session on that session? or, you should extend the bin/pair to ask your to select which existing sessions to connect to, in an TUI type of thing.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:596:💬: ok, when I created a new session with bin/pair, the title of my terminal changed to the slug name used, e.g. pair-claude-5. however, when reattached to pair-claude-5, that didn't happen, the title of the terminal stayed as bin/pair. why?
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:636:💬: add help text with bin/pair --help
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:644:💬: bin/pair didn't show the session I created with name blogging, I think it's intentional from you, can you explain how this worked
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:657:💬: i see, the auto picking behavior is not that useful, because in the bin/pair world, claude sessions are long lived. and auto attach to some long running session that user don't know, is not a good mental model. we can remove auto attach behavior, and always show the picker.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:659:now with this change, it seems basically everything would go to bin/pair pick, right? if that's the case we can just remove pick as a subcommand, as it is the default and only thing that makes sense,
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:669:💬: bin/pair only shows the names pair-claude* still. in the new world, you should show all. right?
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:708:bin/pair only shows the names pair-claude* still. in the new world, you should show all. right?
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:714:[Confirmed file still had old code. Rewrote bin/pair from scratch using Write tool, then verified with grep. This time it stuck:]
./cmd/pair-scribe/README.md:28:Produces `~/.local/bin/pair-scribe` (and the other Go binaries). Static
./cmd/pair-scribe/README.md:43:    exec ~/.local/bin/pair-scribe -log "$_ZSH_SCRIPT_LOG" -- /bin/zsh
./cmd/internal/scrollbackcmd/scrollbackcmd.go:3:// Drop-in replacement for the Python+pyte renderer at bin/pair-scrollback-render.
./cmd/internal/scrollbackcmd/scrollbackcmd.go:20:// CLI is identical to the Python version so bin/pair-scrollback-open can
./workshop/continuation/20260611T163027-port.md:33:- Key files: `cmd/pair-continuation/` (writer), `cmd/pair-scrollback-render/main.go` (`--plain`/`resolveMax`), `bin/pair` (`continue` verb just after the `resume` block; park-nudge inside `cleanup_quit_marker`).
./bin/pair-quit.sh:2:# Triggered by Alt+x via zellij keybind. Writes a marker file so bin/pair
./bin/pair-quit.sh:8:# bin/pair sees the marker on resume and runs `zellij delete-session --force`
./cmd/internal/entrypoint/asset_root.go:30:		candidates = append(candidates, assetRootCandidate{root: input.PairHome, source: "PAIR_HOME"})
./cmd/internal/entrypoint/asset_root.go:39:		candidates = append(candidates, assetRootCandidate{root: input.DefaultPairHome, source: "defaultPairHome"})
./cmd/internal/entrypoint/asset_root.go:63:	return AssetRoot{}, fmt.Errorf("pair-shell not found; set PAIR_HOME to a Pair checkout/install root containing bin/pair-shell (checked: %s)", strings.Join(checked, ", "))
./cmd/internal/entrypoint/asset_root.go:67:	return filepath.Join(root, "bin", "pair-shell")
./cmd/pair-go/main_test.go:42:		executable: "/repo/bin/pair-go",
./cmd/pair-go/main_test.go:56:	if rt.execPath != "/repo/bin/pair-shell" {
./cmd/pair-go/main_test.go:57:		t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
./cmd/pair-go/main_test.go:59:	if rt.execLabel != "pair-go launch" {
./cmd/pair-go/main_test.go:60:		t.Fatalf("execLabel = %q, want pair-go launch", rt.execLabel)
./cmd/pair-go/main_test.go:71:		executable: "/repo/bin/pair-go",
./cmd/pair-go/main_test.go:88:	if rt.execPath != "/repo/bin/pair-shell" {
./cmd/pair-go/main_test.go:89:		t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
./cmd/pair-go/main_test.go:91:	if rt.execLabel != "pair-go launch" {
./cmd/pair-go/main_test.go:92:		t.Fatalf("execLabel = %q, want pair-go launch", rt.execLabel)
./cmd/pair-go/main_test.go:105:		executable: "/repo/bin/pair-go",
./cmd/pair-go/main_test.go:117:	for _, want := range []string{"pair-go launch", "pair-shell", "PAIR_HOME", "/repo", "make build", "make install", "dev-aliases.sh"} {
./cmd/pair-go/main_test.go:129:		executable: "/repo/bin/pair",
./cmd/pair-go/main_test.go:146:	if rt.execPath != "/repo/bin/pair-shell" {
./cmd/pair-go/main_test.go:147:		t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
./cmd/pair-go/main_test.go:160:		executable:      "/home/me/.local/bin/pair",
./cmd/pair-go/main_test.go:161:		defaultPairHome: "/repo",
./cmd/pair-go/main_test.go:174:	if rt.execPath != "/repo/bin/pair-shell" {
./cmd/pair-go/main_test.go:175:		t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
./cmd/pair-go/main_test.go:181:		executable: "/repo/bin/pair-go",
./cmd/pair-go/main_test.go:207:	defaultPairHome string
./cmd/pair-go/main_test.go:231:	return f.defaultPairHome
./cmd/pair-go/main_test.go:239:	if strings.HasSuffix(path, "/bin/pair-shell") && f.roots != nil {
./cmd/pair-go/main_test.go:240:		root := strings.TrimSuffix(path, "/bin/pair-shell")
./bin/copy-on-select.sh:27:    echo "PAIR_HOME=${PAIR_HOME:-unset}"
./bin/copy-on-select.sh:104:if [ -n "$focused_id" ] && [ -x "$PAIR_HOME/bin/flash-pane.sh" ]; then
./bin/copy-on-select.sh:105:    "$PAIR_HOME/bin/flash-pane.sh" "$focused_id"
./bin/copy-on-select.sh:110:exec "$PAIR_HOME/bin/clipboard-to-pane.sh"
./cmd/pair-go/main.go:16:var defaultPairHome string
./cmd/pair-go/main.go:39:			_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
./cmd/pair-go/main.go:50:		return runLegacyLaunch("pair-go launch", exe, args[1:], stderr, rt)
./cmd/pair-go/main.go:81:	return os.Getenv("PAIR_HOME")
./cmd/pair-go/main.go:85:	return defaultPairHome
./bin/pair-title.sh:13:# Spawned in the background by bin/pair on both the create and attach paths.
./bin/pair-title.sh:36:# the cmux sidebar are overridden on the next poll — matches bin/pair's
./bin/pair-title.sh:59:# matching "211"; the agent arg always follows the tag. Mirrors bin/pair's
./bin/pair-title.sh:76:# Abbreviate a raw cwd to ~ on a path boundary (mirrors bin/pair:1154).
./bin/pair-title.sh:125:# Test hook (mirrors bin/pair's PAIR_TEST_CALL): invoke a single helper
./bin/pair-title.sh:138:# create path in bin/pair, which spawns this poller right BEFORE calling
./bin/pair-title.sh:152:# Ignore SIGHUP. bin/pair spawns this with `& disown`, which only
./bin/pair-title.sh:245:# Wait for the zellij session to appear (create-path race: bin/pair
./bin/pair-title.sh:312:            # Workspace-title ownership (matches bin/pair's cmux_rename_workspace):
./bin/pair-title.sh:327:            # Personal display convention (matches bin/pair's cmux_rename_workspace):
./cmd/internal/launcher/args.go:5:// LaunchArgs is the pure parse result for the guarded pair-go launch prototype.
./cmd/internal/launcher/args.go:22:// ParseArgs parses pair-go launch args. It intentionally supports only the
./cmd/internal/launcher/args.go:34:		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %s is not implemented by pair-go launch; use bin/pair", argv[0])}
./cmd/internal/launcher/args.go:37:			return LaunchArgs{}, UsageError{Message: "pair-go launch: 'resume' requires a tag"}
./cmd/internal/launcher/args.go:41:			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: invalid tag: %v", err)}
./cmd/internal/launcher/args.go:44:			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", argv[2])}
./cmd/internal/launcher/args.go:64:		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", arg)}
./bin/lib/dev-rebuild.sh:3:# `pair-dev` exports PAIR_DEV=1 then execs `pair`; bin/pair sources this and
./bin/lib/dev-rebuild.sh:6:# $PAIR_HOME/bin (which is first on PATH), so the layout's `exec pair-wrap` — a
./bin/lib/dev-rebuild.sh:11:# Restart-safe: Alt+n / Shift+Alt+N re-exec $0=bin/pair, and PAIR_DEV rides
./bin/lib/dev-rebuild.sh:17:# Usage:  PAIR_HOME=<repo> dev_rebuild      (no-op unless PAIR_DEV is set)
./bin/lib/dev-rebuild.sh:20:# `set -e` (bin/pair:20) — least of all mid-restart, when the old session is
./bin/lib/dev-rebuild.sh:30:    echo "pair-dev: rebuilding Go binaries (make build in $PAIR_HOME) …" >&2
./bin/lib/dev-rebuild.sh:31:    make -C "$PAIR_HOME" build >&2 \
./bin/pair-session-watch.sh:12:PAIR_HOME="$(cd -P "$(dirname "$SOURCE")/.." && pwd)"
./bin/pair-session-watch.sh:13:export PAIR_HOME
./bin/pair-session-watch.sh:15:cmd="$PAIR_HOME/bin/pair-session-watch"
./tests/review-toggle-test.sh:101:  '"' .. vim.env.PAIR_HOME .. '/bin/pair-review-target" "$2" ready >/dev/null',
./tests/review-toggle-test.sh:203:    PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" PAIR_SESSION_ID=testsid \
./bin/pair-review-open:9:# Env (exported by bin/pair): $PAIR_DATA_DIR $PAIR_TAG $PAIR_HOME
./bin/pair-review-open:21:if [ -z "${PAIR_DATA_DIR:-}" ] || [ -z "${PAIR_TAG:-}" ] || [ -z "${PAIR_HOME:-}" ]; then
./bin/pair-review-open:22:    printf 'pair-review-open: missing PAIR_DATA_DIR / PAIR_TAG / PAIR_HOME\n' >&2
./bin/pair-review-open:40:# nvim/review.lua's VimEnter writes its pid here so bin/pair's cleanup can reap
./bin/pair-review-open:54:    -- nvim -u "$PAIR_HOME/nvim/review.lua" "$abs"
./cmd/internal/launcher/args_test.go:83:			if !strings.Contains(err.Error(), "not implemented by pair-go launch") {
./bin/pair-scrollback-open:9:# Inputs come from pair's env (exported by bin/pair before zellij starts):
./bin/pair-scrollback-open:13:#   $PAIR_HOME      — pair install root (for nvim/scrollback.lua)
./bin/pair-scrollback-open:70:if [ ! -x "$PAIR_HOME/bin/pair-scrollback-render" ]; then
./bin/pair-scrollback-open:71:    printf 'pair-scrollback-open: bin/pair-scrollback-render not built.\n' >&2
./bin/pair-scrollback-open:76:if ! "$PAIR_HOME/bin/pair-scrollback-render" "$RAW" "$EVENTS" "$ANSI"; then
./bin/pair-scrollback-open:182:# embed pid here so cleanup_quit_marker in bin/pair can reap it on Alt+x.
./bin/pair-scrollback-open:193:nvim -u "$PAIR_HOME/nvim/scrollback.lua" "$ANSI"
./bin/clipboard-to-pane.sh:32:    echo "PAIR_HOME=${PAIR_HOME:-unset}"
./bin/pair-notify:8:# the PTY stream for attention markers. bin/pair records its controlling
./bin/pair-shell:108:  no toolchain dependency). See bin/pair-dev + bin/lib/dev-rebuild.sh.
./bin/pair-shell:160:                `1 bin/pair-wrap`); past 9, use arrows or <C-n>/<C-p>.
./bin/pair-shell:556:PAIR_HOME="$(cd -P "$(dirname "$SOURCE")/.." && pwd)"
./bin/pair-shell:557:export PAIR_HOME
./bin/pair-shell:563:    *":$PAIR_HOME/bin:"*) ;;
./bin/pair-shell:564:    *) export PATH="$PAIR_HOME/bin:$PATH" ;;
./bin/pair-shell:898:# background poller (bin/pair-title.sh) still defers to a live different-tag
./bin/pair-shell:1023:# the session — the outer bin/pair then re-execs `pair continue <slug>` fresh
./bin/pair-shell:1094:    # (zellij/layouts/main.kdl, bin/pair-scrollback-open) point that env
./bin/pair-shell:1174:# $PWD here is the dir the user ran `pair` from (bin/pair never cd's); the
./bin/pair-shell:1255:    # set -o pipefail, that propagates and errexit silently kills bin/pair at
./bin/pair-shell:1277:# autosave and bin/pair's send-log), bucket per tag, take max mtime, filter
./bin/pair-shell:1684:# Ensure the per-tag title poller (bin/pair-title.sh) is running for this tag.
./bin/pair-shell:1694:    "$PAIR_HOME/bin/pair-title.sh" "$PAIR_TAG" "$title_agent" \
./bin/pair-shell:1708:    # preexec hook set (typically "bin/pair").
./bin/pair-shell:1718:    zellij --config-dir "$PAIR_HOME/zellij" attach "$SESSION"
./bin/pair-shell:2083:    # bin/pair-session-watch.sh; keep them in sync.
./bin/pair-shell:2222:# pair-doctor reads it to spot integration drift. bin/pair owns its lifecycle:
./bin/pair-shell:2235:"$PAIR_HOME/bin/pair-session-watch.sh" "$AGENT" "$PAIR_TAG" "$PWD" $agent_extra \
./bin/pair-shell:2250:# only, so a plain attach skips it; restart re-execs $0=bin/pair with PAIR_DEV
./bin/pair-shell:2253:. "$PAIR_HOME/bin/lib/dev-rebuild.sh"
./bin/pair-shell:2283:    --config-dir              "$PAIR_HOME/zellij" \
./bin/pair-shell:2284:    --new-session-with-layout "$PAIR_HOME/zellij/layouts/main.kdl" \
./nvim/scrollback.lua:3:-- Loaded via `nvim -u $PAIR_HOME/nvim/scrollback.lua <path-to-.ansi>`.
./nvim/scrollback.lua:56:-- $PAIR_NVIM_PID_FILE so bin/pair's cleanup_quit_marker can reap it on
./nvim/scrollback.lua:285:  if vim.env.PAIR_HOME and vim.env.PAIR_HOME ~= '' then
./nvim/scrollback.lua:286:    bin = vim.env.PAIR_HOME .. '/bin/pair-scrollback-render'
./bin/pair-review-readiness:9:# Env: $PAIR_HOME (locates the lua module; defaults to this script's repo root).
./bin/pair-review-readiness:20:HOME_DIR="${PAIR_HOME:-$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)}"
./bin/pair-review-readiness:34:    target_bin="${PAIR_REVIEW_TARGET_BIN:-$HOME_DIR/bin/pair-review-target}"
./bin/pair-restart.sh:3:# current pair session (like Alt+x) and signals bin/pair to re-launch:
./bin/pair-restart.sh:9:#                    conversation: bin/pair drops the saved
./bin/pair-restart.sh:14:# bin/pair runs `zellij delete-session` as usual) AND a `restart-<session>`
./bin/pair-restart.sh:16:# kill-session returns, bin/pair sees the restart marker and execs itself
./bin/pair-restart.sh:22:# cleanup_quit_marker deletes that file before bin/pair gets the chance to
./nvim/review.lua:2:-- `nvim -u $PAIR_HOME/nvim/review.lua <file>` (from bin/pair-review-open).
./nvim/review.lua:539:  -- pid file (reaped by bin/pair's cleanup) + the open-state file.
./tests/copy-on-select-test.sh:12:# JSON, stub the downstream handoff ($PAIR_HOME/bin/clipboard-to-pane.sh), and
./tests/copy-on-select-test.sh:20:# Sandbox PAIR_HOME so we can stub the scripts copy-on-select.sh execs by
./tests/copy-on-select-test.sh:21:# absolute path ($PAIR_HOME/bin/{flash-pane,clipboard-to-pane}.sh).
./tests/copy-on-select-test.sh:22:export PAIR_HOME="$tmp/home"
./tests/copy-on-select-test.sh:23:mkdir -p "$PAIR_HOME/bin"
./tests/copy-on-select-test.sh:24:cp "$REPO/bin/copy-on-select.sh" "$PAIR_HOME/bin/"
./tests/copy-on-select-test.sh:27:cat > "$PAIR_HOME/bin/clipboard-to-pane.sh" <<EOF
./tests/copy-on-select-test.sh:32:printf '#!/bin/sh\nexit 0\n' > "$PAIR_HOME/bin/flash-pane.sh"
./tests/copy-on-select-test.sh:33:chmod +x "$PAIR_HOME/bin/clipboard-to-pane.sh" "$PAIR_HOME/bin/flash-pane.sh"
./tests/copy-on-select-test.sh:36:# `list-panes` (jq is the real one). PATH must NOT include $PAIR_HOME/bin so
./tests/copy-on-select-test.sh:59:  "terminal_command":"sh -c export PAIR_NVIM_PID_FILE=\"/data/nvim-pid-t-draft\" && exec nvim -u \"$PAIR_HOME/nvim/init.lua\" \"/data/draft-t.md\""}'
./tests/copy-on-select-test.sh:61:run() { rm -f "$tmp/handoff"; printf '%s' 'selected text' | "$PAIR_HOME/bin/copy-on-select.sh"; }
./bin/pair-changelog-open:17:# Inputs from pair's env (exported by bin/pair before zellij starts):
./bin/pair-changelog-open:18:#   $PAIR_DATA_DIR  $PAIR_TAG  $PAIR_AGENT  $PAIR_HOME
./bin/pair-changelog-open:29:# resume reuses it. PAIR_SESSION_ID is exported by bin/pair when the id is known
./bin/pair-changelog-open:77:   && [ -x "$PAIR_HOME/bin/pair-scrollback-render" ] && [ -x "$PAIR_HOME/bin/pair-changelog" ]; then
./bin/pair-changelog-open:79:    export PCL_RENDER="$PAIR_HOME/bin/pair-scrollback-render"
./bin/pair-changelog-open:80:    export PCL_DISTILL="$PAIR_HOME/bin/pair-changelog"
./bin/pair-changelog-open:98:nvim -u "$PAIR_HOME/nvim/changelog.lua" "$LOG"
./tests/pair-go-install-layout-test.sh:13:printf '#!/usr/bin/env bash\nprintf old-pair-shell\\n\n' > "$old_bin/pair"
./tests/pair-go-install-layout-test.sh:14:chmod +x "$old_bin/pair"
./tests/pair-go-install-layout-test.sh:15:ln -s "$old_bin/pair" "$install_bin/pair"
./tests/pair-go-install-layout-test.sh:19:test -x "$install_bin/pair"
./tests/pair-go-install-layout-test.sh:20:test ! -L "$install_bin/pair"
./tests/pair-go-install-layout-test.sh:21:test -x "$install_bin/pair-go"
./tests/pair-go-install-layout-test.sh:22:test -L "$install_bin/pair-dev"
./tests/pair-go-install-layout-test.sh:23:test -x "$repo_root/bin/pair-shell"
./tests/pair-go-install-layout-test.sh:25:out="$("$install_bin/pair" --help)"
./tests/pair-go-install-layout-test.sh:34:out="$("$install_bin/pair-go" launch --help)"
./tests/pair-go-install-layout-test.sh:38:        printf 'pair-go launch --help did not reach pair help; first bytes:\n%s\n' "$out" >&2
./tests/pair-go-install-layout-test.sh:43:out="$(PAIR_HOME="$repo_root" "$install_bin/pair" --help)"
./tests/pair-go-install-layout-test.sh:47:        printf 'PAIR_HOME pair --help did not reach pair help; first bytes:\n%s\n' "$out" >&2
./bin/pair-review-target:2:# bin/pair-review-target — write the review-target seam (#6) stamped with the
./nvim/init.lua:14:-- the embed pid (lua runs inside the embed); bin/pair's cleanup_quit_marker
./nvim/init.lua:916:    local home = vim.env.PAIR_HOME or ''
./nvim/init.lua:918:    if not bin or bin == '' then bin = (home ~= '') and (home .. '/bin/pair-review-readiness') or 'pair-review-readiness' end
./nvim/init.lua:965:      local home = vim.env.PAIR_HOME or ''
./nvim/init.lua:966:      local bin = (home ~= '') and (home .. '/bin/pair-review-open') or 'pair-review-open'
./nvim/init.lua:1607:-- Reload via `:luafile $PAIR_HOME/nvim/init.lua` (works because all autocmds
./nvim/init.lua:1655:-- `/`, `~`, or `./` / `../` etc. Plain `bin/pair-wrap` is *not* a path here
./nvim/init.lua:1727:-- `pair-wrap`, `lessons.md`, `bin/pair-wrap`, `$PAIR_HOME`,
./nvim/init.lua:2982:-- Resolve the change-log session id (#63): the env var bin/pair exports when the
./nvim/init.lua:2985:-- order in bin/pair-changelog-open so the polled .ready path matches the base the
./nvim/init.lua:3962:-- $PAIR_HOME and hand the absolute-pathed instruction to whatever agent is
./nvim/init.lua:3964:-- since the paths are $PAIR_HOME-absolute). Auto-sends via send_to_agent.
./nvim/init.lua:3977:    local body = doctor.payload(vim.env.PAIR_HOME)
./nvim/init.lua:3979:      vim.notify('PairDoctor: PAIR_HOME unset (run inside a pair session).',
./nvim/doctor.lua:4:-- (read $PAIR_HOME, hand the instruction to the agent pane via send_to_agent).
./nvim/doctor.lua:10:-- $PAIR_HOME — so it can hand ANY agent a $PAIR_HOME-absolute instruction. The
./nvim/doctor.lua:14:-- payload(pair_home) → the instruction string with $PAIR_HOME-absolute paths
./nvim/doctor.lua:15:-- substituted in (NOT a literal `$PAIR_HOME` — the agent must not depend on its
./nvim/review/seam.lua:4:-- and bin/pair-review-open — can't diverge on the empty-tag fallback
./nvim/review/readiness.lua:4:-- gathering lives in bin/pair-review-readiness. The 4 cases are

exec
/bin/zsh -lc "nl -ba cmd/internal/entrypoint/launch.go | sed -n '1,80p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package entrypoint
     2	
     3	// LegacyLaunchRequest describes the current compatibility handoff from the Go
     4	// entrypoint to the shell-owned pair launcher.
     5	type LegacyLaunchRequest struct {
     6		Path string
     7		Argv []string
     8	}
     9	
    10	func ResolveLegacyLaunch(root AssetRoot, launchArgs []string) LegacyLaunchRequest {
    11		argv := make([]string, 0, len(launchArgs)+1)
    12		argv = append(argv, "pair")
    13		argv = append(argv, launchArgs...)
    14		return LegacyLaunchRequest{
    15			Path: root.ShellPath,
    16			Argv: argv,
    17		}
    18	}

exec
/bin/zsh -lc "nl -ba cmd/internal/entrypoint/mode.go | sed -n '1,80p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package entrypoint
     2	
     3	import "path/filepath"
     4	
     5	type EntrypointMode int
     6	
     7	const (
     8		ModeDispatch EntrypointMode = iota
     9		ModePublicPair
    10		ModePairGoLaunch
    11	)
    12	
    13	func ClassifyInvocation(executable string, args []string) EntrypointMode {
    14		if filepath.Base(executable) == "pair" {
    15			return ModePublicPair
    16		}
    17		if len(args) > 0 && args[0] == "launch" {
    18			return ModePairGoLaunch
    19		}
    20		return ModeDispatch
    21	}

exec
/bin/zsh -lc 'git diff 776ecce27bc3643bc5598df559c52a3bb5424efe..HEAD -- bin/pair-dev Makefile.local cmd/internal/entrypoint cmd/pair-go tests/pair-go-install-layout-test.sh' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
diff --git a/Makefile.local b/Makefile.local
index 8afd085..d2383b8 100644
--- a/Makefile.local
+++ b/Makefile.local
@@ -1,7 +1,7 @@
 # Repo-specific Makefile targets.
 # This file is included by Makefile — add your own targets here.
 
-.PHONY: build install pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-session-watch pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
+.PHONY: build install pair pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-session-watch pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
 
 # Cross-cutting Go-binary targets. Every Go binary lives under cmd/<name>/
 # and ships with the `pair-` prefix so it doesn't collide with anything
@@ -26,8 +26,9 @@
 
 BIN_DIR     := bin
 INSTALL_DIR := $(HOME)/.local/bin
-GO_BINS     := pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-session-watch pair-go
-SHELL_BINS  := pair pair-dev
+GO_BINS     := pair pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-session-watch pair-go
+SHELL_BINS  := pair-dev
+PAIR_HOME_LDFLAGS := -X main.defaultPairHome=$(CURDIR)
 
 build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
 
@@ -43,6 +44,7 @@ install: build
 	done
 
 # Per-binary aliases — `make pair-scribe`, etc.
+pair:                   $(BIN_DIR)/pair
 pair-scribe:            $(BIN_DIR)/pair-scribe
 pair-scrollback-render: $(BIN_DIR)/pair-scrollback-render
 pair-wrap:              $(BIN_DIR)/pair-wrap
@@ -76,7 +78,7 @@ test-pair-title:
 # cmux workspace ownership claim (presence beats a stale owner flag), so a
 # launch/attach/restart reclaims its workspace title. See
 # tests/cmux-ownership-test.sh.
-test-cmux-ownership:
+test-cmux-ownership: $(BIN_DIR)/pair
 	bash tests/cmux-ownership-test.sh
 
 # Session id watcher must ignore stale pidfiles from prior launches and wait
@@ -187,9 +189,9 @@ test-adapt-schema:
 test-dev-rebuild:
 	bash tests/dev-rebuild-test.sh
 
-# pair-go launch handoff (#77): installed pair-go requires sibling pair. Local
-# installs link shell launchers back to the source tree so pair keeps its
-# PAIR_HOME asset resolution.
+# Go public entrypoint install layout (#79): installed pair is a regular Go
+# binary, pair-go launch shares its pair-shell handoff, and copied local
+# installs fall back to the build-time source root for adjacent assets.
 test-pair-go-install-layout:
 	bash tests/pair-go-install-layout-test.sh
 
@@ -201,7 +203,7 @@ test-emitter-health:
 # `pair continue` arg-parsing + session-name guard (#000054): tag not forced,
 # -- args forwarded, [agent] port, bare list / error paths, guard short-vs-long.
 # Drives the real bin/pair via its PAIR_DEBUG_ARGS probe. See tests/pair-continue-test.sh.
-test-continue:
+test-continue: $(BIN_DIR)/pair
 	bash tests/pair-continue-test.sh
 
 # changelog — end-to-end smoke for bin/pair-changelog-open (#53 M2): fakes the
@@ -256,11 +258,19 @@ $(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill
 $(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-continuation/git.go cmd/pair-continuation/main.go go.mod
 	go build -o $@ ./cmd/pair-continuation
 
-# pair-go — development dispatcher and Go-owned launch handoff.
-# `pair-go launch ...` execs sibling bin/pair for the current migration window;
+PAIR_GO_SRCS := cmd/pair-go/main.go cmd/internal/entrypoint/asset_root.go cmd/internal/entrypoint/launch.go cmd/internal/entrypoint/mode.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
+
+# pair — public Go-owned entrypoint. It execs bin/pair-shell for the current
+# shell-owned zellij lifecycle and uses the build-time defaultPairHome when a
+# copied local install has no sibling assets.
+$(BIN_DIR)/pair: $(PAIR_GO_SRCS)
+	go build -ldflags "$(PAIR_HOME_LDFLAGS)" -o $@ ./cmd/pair-go
+
+# pair-go — development dispatcher and explicit migration surface.
+# `pair-go launch ...` shares the same bin/pair-shell handoff as public pair;
 # helper routes stay in the dispatcher.
-$(BIN_DIR)/pair-go: cmd/pair-go/main.go cmd/internal/entrypoint/launch.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
-	go build -o $@ ./cmd/pair-go
+$(BIN_DIR)/pair-go: $(PAIR_GO_SRCS)
+	go build -ldflags "$(PAIR_HOME_LDFLAGS)" -o $@ ./cmd/pair-go
 
 
 # Tart VM targets (tart, tart-gui, tart-mount, tart-stop, tart-clean)
diff --git a/cmd/internal/entrypoint/asset_root.go b/cmd/internal/entrypoint/asset_root.go
new file mode 100644
index 0000000..ebdf6df
--- /dev/null
+++ b/cmd/internal/entrypoint/asset_root.go
@@ -0,0 +1,73 @@
+package entrypoint
+
+import (
+	"fmt"
+	"path/filepath"
+	"strings"
+)
+
+type AssetRootInput struct {
+	PairHome        string
+	Executable      string
+	DefaultPairHome string
+	PairShellExists func(root string) bool
+}
+
+type AssetRoot struct {
+	Root      string
+	ShellPath string
+	Source    string
+}
+
+func ResolveAssetRoot(input AssetRootInput) (AssetRoot, error) {
+	exists := input.PairShellExists
+	if exists == nil {
+		exists = func(string) bool { return false }
+	}
+
+	candidates := make([]assetRootCandidate, 0, 3)
+	if input.PairHome != "" {
+		candidates = append(candidates, assetRootCandidate{root: input.PairHome, source: "PAIR_HOME"})
+	}
+	if input.Executable != "" {
+		candidates = append(candidates, assetRootCandidate{
+			root:   filepath.Dir(filepath.Dir(input.Executable)),
+			source: "executable sibling",
+		})
+	}
+	if input.DefaultPairHome != "" {
+		candidates = append(candidates, assetRootCandidate{root: input.DefaultPairHome, source: "defaultPairHome"})
+	}
+
+	seen := map[string]bool{}
+	checked := make([]string, 0, len(candidates))
+	for _, candidate := range candidates {
+		root := filepath.Clean(candidate.root)
+		if root == "." || seen[root] {
+			continue
+		}
+		seen[root] = true
+		checked = append(checked, root)
+		if exists(root) {
+			return AssetRoot{
+				Root:      root,
+				ShellPath: PairShellPath(root),
+				Source:    candidate.source,
+			}, nil
+		}
+	}
+
+	if len(checked) == 0 {
+		checked = append(checked, "<none>")
+	}
+	return AssetRoot{}, fmt.Errorf("pair-shell not found; set PAIR_HOME to a Pair checkout/install root containing bin/pair-shell (checked: %s)", strings.Join(checked, ", "))
+}
+
+func PairShellPath(root string) string {
+	return filepath.Join(root, "bin", "pair-shell")
+}
+
+type assetRootCandidate struct {
+	root   string
+	source string
+}
diff --git a/cmd/internal/entrypoint/asset_root_test.go b/cmd/internal/entrypoint/asset_root_test.go
new file mode 100644
index 0000000..6c6843d
--- /dev/null
+++ b/cmd/internal/entrypoint/asset_root_test.go
@@ -0,0 +1,84 @@
+package entrypoint
+
+import (
+	"strings"
+	"testing"
+)
+
+func TestResolveAssetRootPrefersPairHome(t *testing.T) {
+	root, err := ResolveAssetRoot(AssetRootInput{
+		PairHome:        "/env/root",
+		Executable:      "/repo/bin/pair",
+		DefaultPairHome: "/default/root",
+		PairShellExists: existingRoots("/env/root", "/repo", "/default/root"),
+	})
+	if err != nil {
+		t.Fatalf("ResolveAssetRoot error = %v", err)
+	}
+	if root.Root != "/env/root" {
+		t.Fatalf("Root = %q, want /env/root", root.Root)
+	}
+	if root.ShellPath != "/env/root/bin/pair-shell" {
+		t.Fatalf("ShellPath = %q, want /env/root/bin/pair-shell", root.ShellPath)
+	}
+}
+
+func TestResolveAssetRootUsesExecutableSiblingRoot(t *testing.T) {
+	root, err := ResolveAssetRoot(AssetRootInput{
+		Executable:      "/repo/bin/pair",
+		DefaultPairHome: "/default/root",
+		PairShellExists: existingRoots("/repo", "/default/root"),
+	})
+	if err != nil {
+		t.Fatalf("ResolveAssetRoot error = %v", err)
+	}
+	if root.Root != "/repo" {
+		t.Fatalf("Root = %q, want /repo", root.Root)
+	}
+	if root.ShellPath != "/repo/bin/pair-shell" {
+		t.Fatalf("ShellPath = %q, want /repo/bin/pair-shell", root.ShellPath)
+	}
+}
+
+func TestResolveAssetRootFallsBackToDefaultPairHome(t *testing.T) {
+	root, err := ResolveAssetRoot(AssetRootInput{
+		Executable:      "/home/me/.local/bin/pair",
+		DefaultPairHome: "/repo",
+		PairShellExists: existingRoots("/repo"),
+	})
+	if err != nil {
+		t.Fatalf("ResolveAssetRoot error = %v", err)
+	}
+	if root.Root != "/repo" {
+		t.Fatalf("Root = %q, want /repo", root.Root)
+	}
+	if root.ShellPath != "/repo/bin/pair-shell" {
+		t.Fatalf("ShellPath = %q, want /repo/bin/pair-shell", root.ShellPath)
+	}
+}
+
+func TestResolveAssetRootReportsMissingPairShellAndPairHome(t *testing.T) {
+	_, err := ResolveAssetRoot(AssetRootInput{
+		Executable:      "/home/me/.local/bin/pair",
+		DefaultPairHome: "/repo",
+		PairShellExists: existingRoots(),
+	})
+	if err == nil {
+		t.Fatal("ResolveAssetRoot error = nil, want missing-root error")
+	}
+	for _, want := range []string{"pair-shell", "PAIR_HOME", "/home/me/.local", "/repo"} {
+		if !strings.Contains(err.Error(), want) {
+			t.Fatalf("error missing %q:\n%v", want, err)
+		}
+	}
+}
+
+func existingRoots(roots ...string) func(string) bool {
+	set := make(map[string]bool, len(roots))
+	for _, root := range roots {
+		set[root] = true
+	}
+	return func(root string) bool {
+		return set[root]
+	}
+}
diff --git a/cmd/internal/entrypoint/launch.go b/cmd/internal/entrypoint/launch.go
index 1a40970..04e97f8 100644
--- a/cmd/internal/entrypoint/launch.go
+++ b/cmd/internal/entrypoint/launch.go
@@ -1,20 +1,18 @@
 package entrypoint
 
-import "path/filepath"
-
-// LegacyLaunchRequest describes the current compatibility handoff from
-// pair-go launch to the shell-owned pair launcher.
+// LegacyLaunchRequest describes the current compatibility handoff from the Go
+// entrypoint to the shell-owned pair launcher.
 type LegacyLaunchRequest struct {
 	Path string
 	Argv []string
 }
 
-func ResolveLegacyLaunch(executable string, launchArgs []string) LegacyLaunchRequest {
+func ResolveLegacyLaunch(root AssetRoot, launchArgs []string) LegacyLaunchRequest {
 	argv := make([]string, 0, len(launchArgs)+1)
 	argv = append(argv, "pair")
 	argv = append(argv, launchArgs...)
 	return LegacyLaunchRequest{
-		Path: filepath.Join(filepath.Dir(executable), "pair"),
+		Path: root.ShellPath,
 		Argv: argv,
 	}
 }
diff --git a/cmd/internal/entrypoint/launch_test.go b/cmd/internal/entrypoint/launch_test.go
index cdeec2c..c6a50cc 100644
--- a/cmd/internal/entrypoint/launch_test.go
+++ b/cmd/internal/entrypoint/launch_test.go
@@ -6,9 +6,9 @@ import (
 )
 
 func TestResolveLegacyLaunchDropsLaunchVerb(t *testing.T) {
-	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"claude", "--", "--resume"})
-	if req.Path != "/repo/bin/pair" {
-		t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
+	req := ResolveLegacyLaunch(AssetRoot{Root: "/repo", ShellPath: "/repo/bin/pair-shell"}, []string{"claude", "--", "--resume"})
+	if req.Path != "/repo/bin/pair-shell" {
+		t.Fatalf("Path = %q, want /repo/bin/pair-shell", req.Path)
 	}
 	want := []string{"pair", "claude", "--", "--resume"}
 	if !reflect.DeepEqual(req.Argv, want) {
@@ -17,9 +17,9 @@ func TestResolveLegacyLaunchDropsLaunchVerb(t *testing.T) {
 }
 
 func TestResolveLegacyLaunchPreservesSubcommands(t *testing.T) {
-	req := ResolveLegacyLaunch("/repo/bin/pair-go", []string{"resume", "demo"})
-	if req.Path != "/repo/bin/pair" {
-		t.Fatalf("Path = %q, want /repo/bin/pair", req.Path)
+	req := ResolveLegacyLaunch(AssetRoot{Root: "/repo", ShellPath: "/repo/bin/pair-shell"}, []string{"resume", "demo"})
+	if req.Path != "/repo/bin/pair-shell" {
+		t.Fatalf("Path = %q, want /repo/bin/pair-shell", req.Path)
 	}
 	want := []string{"pair", "resume", "demo"}
 	if !reflect.DeepEqual(req.Argv, want) {
diff --git a/cmd/internal/entrypoint/mode.go b/cmd/internal/entrypoint/mode.go
new file mode 100644
index 0000000..23942a2
--- /dev/null
+++ b/cmd/internal/entrypoint/mode.go
@@ -0,0 +1,21 @@
+package entrypoint
+
+import "path/filepath"
+
+type EntrypointMode int
+
+const (
+	ModeDispatch EntrypointMode = iota
+	ModePublicPair
+	ModePairGoLaunch
+)
+
+func ClassifyInvocation(executable string, args []string) EntrypointMode {
+	if filepath.Base(executable) == "pair" {
+		return ModePublicPair
+	}
+	if len(args) > 0 && args[0] == "launch" {
+		return ModePairGoLaunch
+	}
+	return ModeDispatch
+}
diff --git a/cmd/pair-go/main.go b/cmd/pair-go/main.go
index 760f876..6834be7 100644
--- a/cmd/pair-go/main.go
+++ b/cmd/pair-go/main.go
@@ -6,12 +6,15 @@ import (
 	"fmt"
 	"io"
 	"os"
+	"path/filepath"
 	"syscall"
 
 	"github.com/xianxu/pair/cmd/internal/dispatcher"
 	"github.com/xianxu/pair/cmd/internal/entrypoint"
 )
 
+var defaultPairHome string
+
 func main() {
 	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
 }
@@ -22,31 +25,50 @@ func run(args []string, stdout, stderr io.Writer) int {
 
 type legacyRuntime interface {
 	Executable() (string, error)
+	PairHome() string
+	DefaultPairHome() string
 	Stat(path string) error
 	Environ() []string
-	Exec(path string, argv []string, env []string) int
+	Exec(label string, path string, argv []string, env []string) int
 }
 
 func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
-	if len(args) > 0 && args[0] == "launch" {
-		return runLegacyLaunch(args[1:], stderr, rt)
+	exe, err := rt.Executable()
+	if err != nil {
+		if len(args) > 0 && args[0] == "launch" {
+			_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
+			return 1
+		}
+		res := dispatcher.Dispatch(args)
+		return writeResult(res, stdout, stderr)
+	}
+
+	switch entrypoint.ClassifyInvocation(exe, args) {
+	case entrypoint.ModePublicPair:
+		return runLegacyLaunch("pair", exe, args, stderr, rt)
+	case entrypoint.ModePairGoLaunch:
+		return runLegacyLaunch("pair-go launch", exe, args[1:], stderr, rt)
+	default:
+		res := dispatcher.Dispatch(args)
+		return writeResult(res, stdout, stderr)
 	}
-	res := dispatcher.Dispatch(args)
-	return writeResult(res, stdout, stderr)
 }
 
-func runLegacyLaunch(args []string, stderr io.Writer, rt legacyRuntime) int {
-	exe, err := rt.Executable()
+func runLegacyLaunch(label string, executable string, args []string, stderr io.Writer, rt legacyRuntime) int {
+	root, err := entrypoint.ResolveAssetRoot(entrypoint.AssetRootInput{
+		PairHome:        rt.PairHome(),
+		Executable:      executable,
+		DefaultPairHome: rt.DefaultPairHome(),
+		PairShellExists: func(root string) bool {
+			return rt.Stat(entrypoint.PairShellPath(root)) == nil
+		},
+	})
 	if err != nil {
-		_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
+		_, _ = fmt.Fprintf(stderr, "%s: %v; run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", label, err)
 		return 1
 	}
-	req := entrypoint.ResolveLegacyLaunch(exe, args)
-	if err := rt.Stat(req.Path); err != nil {
-		_, _ = fmt.Fprintf(stderr, "pair-go launch: pair launcher not found at %s (%v); run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", req.Path, err)
-		return 1
-	}
-	return rt.Exec(req.Path, req.Argv, rt.Environ())
+	req := entrypoint.ResolveLegacyLaunch(root, args)
+	return rt.Exec(label, req.Path, req.Argv, rt.Environ())
 }
 
 type osLegacyRuntime struct{}
@@ -55,7 +77,16 @@ func (osLegacyRuntime) Executable() (string, error) {
 	return os.Executable()
 }
 
+func (osLegacyRuntime) PairHome() string {
+	return os.Getenv("PAIR_HOME")
+}
+
+func (osLegacyRuntime) DefaultPairHome() string {
+	return defaultPairHome
+}
+
 func (osLegacyRuntime) Stat(path string) error {
+	path = filepath.Clean(path)
 	info, err := os.Stat(path)
 	if err != nil {
 		return err
@@ -70,9 +101,9 @@ func (osLegacyRuntime) Environ() []string {
 	return os.Environ()
 }
 
-func (osLegacyRuntime) Exec(path string, argv []string, env []string) int {
+func (osLegacyRuntime) Exec(label string, path string, argv []string, env []string) int {
 	if err := syscall.Exec(path, argv, env); err != nil {
-		_, _ = fmt.Fprintf(os.Stderr, "pair-go launch: exec %s failed: %v\n", path, err)
+		_, _ = fmt.Fprintf(os.Stderr, "%s: exec %s failed: %v\n", label, path, err)
 		return 1
 	}
 	return 0
diff --git a/cmd/pair-go/main_test.go b/cmd/pair-go/main_test.go
index d137a6b..75dd2a7 100644
--- a/cmd/pair-go/main_test.go
+++ b/cmd/pair-go/main_test.go
@@ -40,6 +40,7 @@ func TestRunWritesStderrAndReturnsDispatcherCode(t *testing.T) {
 func TestRunLaunchHelp(t *testing.T) {
 	rt := &fakeLegacyRuntime{
 		executable: "/repo/bin/pair-go",
+		roots:      map[string]bool{"/repo": true},
 	}
 	var stdout, stderr bytes.Buffer
 	code := runWithLegacyRuntime([]string{"launch", "--help"}, &stdout, &stderr, rt)
@@ -52,8 +53,11 @@ func TestRunLaunchHelp(t *testing.T) {
 	if stderr.String() != "" {
 		t.Fatalf("stderr = %q, want empty", stderr.String())
 	}
-	if rt.execPath != "/repo/bin/pair" {
-		t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
+	if rt.execPath != "/repo/bin/pair-shell" {
+		t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
+	}
+	if rt.execLabel != "pair-go launch" {
+		t.Fatalf("execLabel = %q, want pair-go launch", rt.execLabel)
 	}
 	wantArgv := []string{"pair", "--help"}
 	if !reflect.DeepEqual(rt.execArgv, wantArgv) {
@@ -65,6 +69,7 @@ func TestRunLaunchExecsLegacyPairWithArgvAndEnv(t *testing.T) {
 	t.Setenv("PAIR_TEST_ENV", "kept")
 	rt := &fakeLegacyRuntime{
 		executable: "/repo/bin/pair-go",
+		roots:      map[string]bool{"/repo": true},
 		execCode:   42,
 	}
 
@@ -80,8 +85,11 @@ func TestRunLaunchExecsLegacyPairWithArgvAndEnv(t *testing.T) {
 	if stderr.String() != "" {
 		t.Fatalf("stderr = %q, want empty", stderr.String())
 	}
-	if rt.execPath != "/repo/bin/pair" {
-		t.Fatalf("execPath = %q, want /repo/bin/pair", rt.execPath)
+	if rt.execPath != "/repo/bin/pair-shell" {
+		t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
+	}
+	if rt.execLabel != "pair-go launch" {
+		t.Fatalf("execLabel = %q, want pair-go launch", rt.execLabel)
 	}
 	wantArgv := []string{"pair", "claude", "--", "--resume"}
 	if !reflect.DeepEqual(rt.execArgv, wantArgv) {
@@ -95,7 +103,6 @@ func TestRunLaunchExecsLegacyPairWithArgvAndEnv(t *testing.T) {
 func TestRunLaunchReportsMissingLegacyPair(t *testing.T) {
 	rt := &fakeLegacyRuntime{
 		executable: "/repo/bin/pair-go",
-		statErr:    os.ErrNotExist,
 	}
 
 	var stdout, stderr bytes.Buffer
@@ -107,7 +114,7 @@ func TestRunLaunchReportsMissingLegacyPair(t *testing.T) {
 	if stdout.String() != "" {
 		t.Fatalf("stdout = %q, want empty", stdout.String())
 	}
-	for _, want := range []string{"pair-go launch", "/repo/bin/pair", "make build", "make install", "dev-aliases.sh"} {
+	for _, want := range []string{"pair-go launch", "pair-shell", "PAIR_HOME", "/repo", "make build", "make install", "dev-aliases.sh"} {
 		if !strings.Contains(stderr.String(), want) {
 			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
 		}
@@ -117,14 +124,96 @@ func TestRunLaunchReportsMissingLegacyPair(t *testing.T) {
 	}
 }
 
+func TestRunDirectPairExecsLegacyShellWithAllArgs(t *testing.T) {
+	rt := &fakeLegacyRuntime{
+		executable: "/repo/bin/pair",
+		roots:      map[string]bool{"/repo": true},
+		execCode:   7,
+	}
+
+	var stdout, stderr bytes.Buffer
+	code := runWithLegacyRuntime([]string{"claude", "--", "--resume"}, &stdout, &stderr, rt)
+
+	if code != 7 {
+		t.Fatalf("code = %d, want 7", code)
+	}
+	if stdout.String() != "" {
+		t.Fatalf("stdout = %q, want empty", stdout.String())
+	}
+	if stderr.String() != "" {
+		t.Fatalf("stderr = %q, want empty", stderr.String())
+	}
+	if rt.execPath != "/repo/bin/pair-shell" {
+		t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
+	}
+	if rt.execLabel != "pair" {
+		t.Fatalf("execLabel = %q, want pair", rt.execLabel)
+	}
+	wantArgv := []string{"pair", "claude", "--", "--resume"}
+	if !reflect.DeepEqual(rt.execArgv, wantArgv) {
+		t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
+	}
+}
+
+func TestRunDirectPairFallsBackToDefaultPairHome(t *testing.T) {
+	rt := &fakeLegacyRuntime{
+		executable:      "/home/me/.local/bin/pair",
+		defaultPairHome: "/repo",
+		roots:           map[string]bool{"/repo": true},
+	}
+
+	var stdout, stderr bytes.Buffer
+	code := runWithLegacyRuntime([]string{"--help"}, &stdout, &stderr, rt)
+
+	if code != 0 {
+		t.Fatalf("code = %d, want 0", code)
+	}
+	if stderr.String() != "" {
+		t.Fatalf("stderr = %q, want empty", stderr.String())
+	}
+	if rt.execPath != "/repo/bin/pair-shell" {
+		t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
+	}
+}
+
+func TestRunPairGoHelperDoesNotProbeOrExecShellLauncher(t *testing.T) {
+	rt := &fakeLegacyRuntime{
+		executable: "/repo/bin/pair-go",
+	}
+
+	var stdout, stderr bytes.Buffer
+	code := runWithLegacyRuntime([]string{"help"}, &stdout, &stderr, rt)
+
+	if code != 0 {
+		t.Fatalf("code = %d, want 0", code)
+	}
+	if rt.statCalls != 0 {
+		t.Fatalf("statCalls = %d, want 0", rt.statCalls)
+	}
+	if rt.execPath != "" {
+		t.Fatalf("execPath = %q, want empty", rt.execPath)
+	}
+	if !strings.Contains(stdout.String(), "Usage: pair-go <command> [args]") {
+		t.Fatalf("stdout missing usage:\n%s", stdout.String())
+	}
+	if stderr.String() != "" {
+		t.Fatalf("stderr = %q, want empty", stderr.String())
+	}
+}
+
 type fakeLegacyRuntime struct {
-	executable string
-	statErr    error
-	execCode   int
+	executable      string
+	pairHome        string
+	defaultPairHome string
+	roots           map[string]bool
+	statErr         error
+	execCode        int
+	statCalls       int
 
-	execPath string
-	execArgv []string
-	execEnv  []string
+	execPath  string
+	execLabel string
+	execArgv  []string
+	execEnv   []string
 }
 
 func (f *fakeLegacyRuntime) Executable() (string, error) {
@@ -134,15 +223,34 @@ func (f *fakeLegacyRuntime) Executable() (string, error) {
 	return f.executable, nil
 }
 
-func (f *fakeLegacyRuntime) Stat(_ string) error {
-	return f.statErr
+func (f *fakeLegacyRuntime) PairHome() string {
+	return f.pairHome
+}
+
+func (f *fakeLegacyRuntime) DefaultPairHome() string {
+	return f.defaultPairHome
+}
+
+func (f *fakeLegacyRuntime) Stat(path string) error {
+	f.statCalls++
+	if f.statErr != nil {
+		return f.statErr
+	}
+	if strings.HasSuffix(path, "/bin/pair-shell") && f.roots != nil {
+		root := strings.TrimSuffix(path, "/bin/pair-shell")
+		if f.roots[root] {
+			return nil
+		}
+	}
+	return os.ErrNotExist
 }
 
 func (f *fakeLegacyRuntime) Environ() []string {
 	return os.Environ()
 }
 
-func (f *fakeLegacyRuntime) Exec(path string, argv []string, env []string) int {
+func (f *fakeLegacyRuntime) Exec(label string, path string, argv []string, env []string) int {
+	f.execLabel = label
 	f.execPath = path
 	f.execArgv = append([]string(nil), argv...)
 	f.execEnv = append([]string(nil), env...)
diff --git a/tests/pair-go-install-layout-test.sh b/tests/pair-go-install-layout-test.sh
index 086cb5b..d276cfc 100644
--- a/tests/pair-go-install-layout-test.sh
+++ b/tests/pair-go-install-layout-test.sh
@@ -7,12 +7,29 @@ trap 'rm -rf "$tmp_home"' EXIT
 gomodcache="$(go env GOMODCACHE)"
 gocache="$(go env GOCACHE)"
 
+install_bin="$tmp_home/.local/bin"
+old_bin="$tmp_home/old-bin"
+mkdir -p "$install_bin" "$old_bin"
+printf '#!/usr/bin/env bash\nprintf old-pair-shell\\n\n' > "$old_bin/pair"
+chmod +x "$old_bin/pair"
+ln -s "$old_bin/pair" "$install_bin/pair"
+
 HOME="$tmp_home" GOMODCACHE="$gomodcache" GOCACHE="$gocache" make -C "$repo_root" install >/dev/null
 
-install_bin="$tmp_home/.local/bin"
+test -x "$install_bin/pair"
+test ! -L "$install_bin/pair"
 test -x "$install_bin/pair-go"
-test -L "$install_bin/pair"
 test -L "$install_bin/pair-dev"
+test -x "$repo_root/bin/pair-shell"
+
+out="$("$install_bin/pair" --help)"
+case "$out" in
+    pair\ —*) ;;
+    *)
+        printf 'pair --help did not reach pair help; first bytes:\n%s\n' "$out" >&2
+        exit 1
+        ;;
+esac
 
 out="$("$install_bin/pair-go" launch --help)"
 case "$out" in
@@ -23,4 +40,13 @@ case "$out" in
         ;;
 esac
 
+out="$(PAIR_HOME="$repo_root" "$install_bin/pair" --help)"
+case "$out" in
+    pair\ —*) ;;
+    *)
+        printf 'PAIR_HOME pair --help did not reach pair help; first bytes:\n%s\n' "$out" >&2
+        exit 1
+        ;;
+esac
+
 printf 'pair-go install layout test passed\n'

exec
/bin/zsh -lc "nl -ba atlas/how-to-bring-up-a-new-harness-cli.md | sed -n '55,72p;134,160p;190,200p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
    55	### Aspect 3: Session ID Watcher & Recovery
    56	`pair` features a robust restart-in-place (`Alt+n`) and session reattach (`pair resume <tag>`) mechanism. To make this work, the launcher needs to discover the agent's unique conversation/session ID as soon as it is spawned.
    57	
    58	**Discovery & Watcher:**
    59	- **Files:** `cmd/pair-session-watch` and `cmd/internal/sessionwatch` (`bin/pair-session-watch.sh` remains a compatibility shim).
    60	- Since TUI agents do not always expose session IDs on stdout, `pair-session-watch` runs in the background. It finds the agent process PID from `$PAIR_DATA_DIR/agent-pid-<tag>` (written by `pair-wrap`), walks its descendants, and inspects files held open by the processes via `lsof -p <pid>`.
    61	- Configure the agent's session file criteria in `cmd/internal/sessionwatch.SpecForAgent`, then teach `AgentSpec.Match` how to recognize that agent's file shape and return a `SessionID`.
    62	- For example, agy watches `~/.gemini/antigravity-cli/conversations` and extracts the UUID from `<uuid>.db`; codex watches `~/.codex/sessions` and extracts the trailing UUID from `rollout-*.jsonl`.
    63	- When captured, the watcher writes `{ "agent": "<agent>", "args": [...], "session_id": "<uuid>" }` into `config-<tag>-<agent>.json`.
    64	
    65	**Recovery Flags:**
    66	- **File:** [bin/pair](file:///Users/xianxu/workspace/pair/bin/pair)
    67	- Integrate the agent-specific resume argument in `bin/pair`:
    68	  ```bash
    69	  case "$r_agent" in
    70	      claude)        resume_extra="--resume $r_sid" ;;
    71	      codex)         resume_extra="resume $r_sid" ;;
    72	      agy)           resume_extra="--conversation $r_sid" ;;
   134	When introducing a new agent `<name>`, ensure you complete each item:
   135	
   136	1. [ ] **Verify Return Key remapping** in `sendKeymapByAgent` (Enter = newline, Alt+Enter = send).
   137	2. [ ] **Check for blocking TUI overlays** and implement a PTY overlay detector in `overlayDetectorByAgent` if needed.
   138	3. [ ] **Implement Session Watching** in `cmd/internal/sessionwatch` / `cmd/pair-session-watch` (using `lsof` and target file patterns).
   139	4. [ ] **Configure Launcher Recovery** in `bin/pair` (mapping `--conversation` or `--resume` flags).
   140	5. [ ] **Add slug generation support** in `pair-slug` (transcript parsing + sandboxed print execution).
   141	6. [ ] **Confirm mouse scroll and scrollback render** work smoothly without drawing glitch issues.
   142	7. [ ] **White-list permissions** in the agent's global or workspace settings directory.
   143	8. [ ] **Register the user-prompt glyph** in `nvim/scrollback.lua` for `Alt+b` jumping.
   144	
   145	---
   146	
   147	## 3. Drift Telemetry
   148	
   149	Harnesses update constantly and break the adaptations above *silently* — a renamed
   150	picker string or a changed transcript shape doesn't error, the adaptation just
   151	stops firing. Unit tests can't catch this: they validate our matchers against
   152	strings we froze, so they pass forever even after the live harness moves.
   153	
   154	The **adaptation flight recorder** makes drift observable. Every adaptation appends
   155	one JSON line per trigger to `$PAIR_DATA_DIR/adapt-<tag>.jsonl` during normal use.
   156	`bin/pair` truncates the file once at session launch; all components then append
   157	(`O_APPEND`, atomic per-line across processes). A user runs `pair` normally; when
   158	something feels off they run **`doctor/doctor.sh`** (see [`doctor/README.md`](file:///Users/xianxu/workspace/pair/doctor/README.md)),
   159	which reads the trace and points at the broken aspect — no need to describe the
   160	symptom. The same procedure is packaged as the `doctor/SKILL.md` skill, so an
   190	| 6 Settings | — | — | — | static config; no signal |
   191	| 7 Prompt search | `prompt-search` | nvim/scrollback.lua | fired, near-miss | `near-miss` (0 matches in non-empty scrollback) |
   192	
   193	> Status: all six runtime aspects emit today (#000045 M1: aspects 1 & 2; M2: aspects 3, 4, 5, 7).
   194	
   195	**Privacy:** `detail` can carry a snippet of agent output (e.g. an unrecognized
   196	prompt). It is capped at 200 bytes and the file stays local under `$PAIR_DATA_DIR`,
   197	the same trust level as the existing scrollback logs. `bin/pair` removes it on quit.

exec
/bin/zsh -lc "nl -ba tests/cmux-ownership-test.sh | sed -n '1,50p'; nl -ba tests/pair-continue-test.sh | sed -n '1,35p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	#!/usr/bin/env bash
     2	# Regression test for bin/pair's cmux_rename_workspace ownership claim.
     3	#
     4	# Guards the fix for the "🔴 ♋-♋" stuck-title bug: a launch/attach/restart must
     5	# claim the cmux workspace it is provably running in, EVEN IF the owner file
     6	# names a different tag whose `pair-<owner>` zellij session is still alive
     7	# somewhere else. The old code deferred to any live different-tag owner, which
     8	# froze the title forever when that owner had moved to a different cmux
     9	# workspace and left a stale owner file behind ("presence beats a stale flag").
    10	#
    11	# Drives the REAL bin/pair through its PAIR_TEST_CALL seam with process-level
    12	# fakes (HOME/XDG_DATA_HOME pinned, fake `cmux` + `zellij` on PATH), exactly
    13	# like the session_blocks_reuse / resolve_config_file cases in
    14	# tests/pair-continue-test.sh.
    15	#
    16	# Run: bash tests/cmux-ownership-test.sh   (also wired into `make test`)
    17	set -uo pipefail
    18	
    19	ROOT="$(cd "$(dirname "$0")/.." && pwd)"
    20	PAIR="$ROOT/bin/pair"
    21	RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-cmuxown-test.XXXXXX")"
    22	trap 'rm -rf "$RT"' EXIT
    23	
    24	fails=0
    25	pass() { printf '  ok   %s\n' "$1"; }
    26	fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }
    27	
    28	WS="testws-1"
    29	mkdir -p "$RT/bin" "$RT/xdg/pair" "$RT/.cache/pair"
    30	
    31	# Fake `cmux`: record every invocation's args so we can assert the rename fired
    32	# (and with what title). `command -v cmux` must also succeed for the rename path.
    33	cat > "$RT/bin/cmux" <<EOF
    34	#!/usr/bin/env bash
    35	printf '%s\n' "\$*" >> "$RT/cmux.log"
    36	EOF
    37	chmod +x "$RT/bin/cmux"
    38	
    39	# Fake `zellij`: report that pair-<owner> sessions are alive — the old code
    40	# would have deferred on this. (cmux_rename_workspace no longer consults it; the
    41	# fake exists to prove the claim happens despite a live different-tag owner.)
    42	cat > "$RT/bin/zellij" <<'EOF'
    43	#!/usr/bin/env bash
    44	case "$1 $2" in
    45	  "list-sessions --short") printf 'pair-other\npair-211\n' ;;
    46	  *) exit 0 ;;
    47	esac
    48	EOF
    49	chmod +x "$RT/bin/zellij"
    50	
     1	#!/usr/bin/env bash
     2	# Regression test for `pair continue` arg-parsing + the launch-time session-
     3	# name guard (bin/pair, #54). Drives the REAL bin/pair through its
     4	# PAIR_DEBUG_ARGS probe (which resolves argv, then exits before any zellij
     5	# work) so the contract is pinned against the actual script, not a mirror.
     6	#
     7	# Covers the two #54 defects + the reshape:
     8	#   1. `continue <slug>` does NOT force the tag (forced_tag empty → the
     9	#      operator names the session at the normal prompt) — the fix for the
    10	#      "session name must be less than 0 characters" zellij crash that the
    11	#      forced long slug triggered.
    12	#   2. `-- <args>` forward to the agent (the old `shift "$#"` dropped them).
    13	#   3. agent comes from the doc frontmatter, or an explicit `[agent]` port.
    14	#   4. bare list + error paths (unknown / invalid slug) still hold.
    15	#   5. the guard's grep discriminates a short (ok) vs over-long (reject) name
    16	#      against the REAL zellij --session validator.
    17	#
    18	# Run: bash tests/pair-continue-test.sh   (also wired into `make test`)
    19	set -uo pipefail
    20	
    21	ROOT="$(cd "$(dirname "$0")/.." && pwd)"
    22	PAIR="$ROOT/bin/pair"
    23	RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-continue-test.XXXXXX")"
    24	trap 'rm -rf "$RT"' EXIT
    25	
    26	# Fixture: a continuation doc whose frontmatter agent is `claude`.
    27	CDIR="$RT/workshop/continuation"
    28	mkdir -p "$CDIR"
    29	cat > "$CDIR/20260101T000000-demo.md" <<'DOC'
    30	---
    31	agent: claude
    32	issues: [#99]
    33	---
    34	## NEXT ACTION
    35	Do the demo next action.

exec
/bin/zsh -lc "sed -n '540,570p;2240,2260p' bin/pair-shell" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
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
# in zellij/config.kdl. Avoid double-prepending across re-launches.
case ":$PATH:" in
    *":$PAIR_HOME/bin:"*) ;;
    *) export PATH="$PAIR_HOME/bin:$PATH" ;;
esac

# Parse argv. First positional is the agent name (default `claude`).
# Anything after `--` is forwarded to the agent on create. The `--`
# separator is required to disambiguate "agent name" from "flag for the
# agent" — without it, `pair --dangerously-skip-permissions` is ambiguous
# itself), useful as a uniform path for both create and attach branches.
printf '\033]0;%s\007' "$SESSION"
record_outer_tty
cmux_rename_workspace "$SESSION"
ensure_title_poller "$AGENT"

# Dev mode (pair-dev exports PAIR_DEV): rebuild the Go binaries from source so
# the layout's `exec pair-wrap` below resolves to a fresh repo/bin build — first
# on PATH — instead of a stale (or gitignored-absent) ~/.local/bin copy. No-op
# with no toolchain dependency unless PAIR_DEV is set. This is the create path
# only, so a plain attach skips it; restart re-execs $0=bin/pair with PAIR_DEV
# surviving, so Alt+n / Shift+Alt+N re-fire the rebuild. See #000046.
# shellcheck source=bin/lib/dev-rebuild.sh
. "$PAIR_HOME/bin/lib/dev-rebuild.sh"
dev_rebuild

# zellij caps a --session name at its unix-socket sun_path budget (capacity
# minus the socket dir, which is long under macOS $TMPDIR) and rejects an
# over-long name with a cryptic clap error ("session name must be less than 0
# characters") that aborts the launch. The budget is machine-specific, so we
# probe zellij's OWN validator with a harmless no-op (`action list-clients`

exec
/bin/zsh -lc 'git ls-files bin/pair bin/pair-shell && test -e bin/pair && file bin/pair || true && test -e bin/pair-shell && file bin/pair-shell || true' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
bin/pair-shell
bin/pair: Mach-O 64-bit executable arm64
bin/pair-shell: Bourne-Again shell script text executable, Unicode text, UTF-8 text

codex
The shadow sweep shows two doc/test surfaces still saying “bin/pair” as the shell implementation. The tests intentionally drive the generated public binary, but at least one atlas page now links to a non-source generated artifact for launcher recovery work, which is stale under the new layout.
exec
/bin/zsh -lc 'PAIR_HOME=. bin/pair --help >/tmp/pair-dot.out 2>/tmp/pair-dot.err; rc=$?; printf '"'rc=%s\\n' \""'$rc"; printf '"'stderr='; sed -n '1,3p' /tmp/pair-dot.err; printf 'stdout='; sed -n '1,1p' /tmp/pair-dot.out" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
rc=0
stderr=stdout=pair — Neovim-backed input field for any TUI coding agent.

exec
/bin/zsh -lc 'git diff --name-only 776ecce27bc3643bc5598df559c52a3bb5424efe..HEAD -- atlas README.md workshop/issues/000079-go-packaging-consolidation.md workshop/plans/000079-go-packaging-consolidation-plan.md' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
README.md
atlas/architecture.md
atlas/go-migration-inventory.md
workshop/issues/000079-go-packaging-consolidation.md
workshop/plans/000079-go-packaging-consolidation-plan.md

exec
/bin/zsh -lc 'go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1' in /Users/xianxu/workspace/pair
 succeeded in 1622ms:
ok  	github.com/xianxu/pair/cmd/internal/entrypoint	0.200s
ok  	github.com/xianxu/pair/cmd/pair-go	1.446s

codex
Focused Go tests pass locally. The only behavior concern I can substantiate is not a runtime blocker; the stronger issue is stale atlas guidance that now points future agent-port work at the generated `bin/pair` instead of the retained shell implementation.
exec
/bin/zsh -lc "rg -n 'ParseArgs|cmd/internal/launcher|use bin/pair|bin/pair"'`|File.*bin/pair|Configure Launcher Recovery.*bin/pair|Integrate.*bin/pair'"' cmd atlas README.md tests --glob '"'!bin/pair'"'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
atlas/how-to-bring-up-a-new-harness-cli.md:59:- **Files:** `cmd/pair-session-watch` and `cmd/internal/sessionwatch` (`bin/pair-session-watch.sh` remains a compatibility shim).
atlas/how-to-bring-up-a-new-harness-cli.md:66:- **File:** [bin/pair](file:///Users/xianxu/workspace/pair/bin/pair)
atlas/how-to-bring-up-a-new-harness-cli.md:67:- Integrate the agent-specific resume argument in `bin/pair`:
atlas/how-to-bring-up-a-new-harness-cli.md:82:**Telemetry Signal** (aspect `3`, see §3): `session-id` from `pair-session-watch` — `fired` when `AgentSpec.Match` resolves an id and the config is written, **`near-miss`** when a file matching the watch pattern is found but no id can be extracted (filename/format drift), `fail` when the 60s watch window elapses with no id at all (the session file never appeared where expected). The resume mapping in `bin/pair` is the *consumer* of this id; it's static config with no separate signal.
atlas/how-to-bring-up-a-new-harness-cli.md:139:4. [ ] **Configure Launcher Recovery** in `bin/pair` (mapping `--conversation` or `--resume` flags).
atlas/how-to-bring-up-a-new-harness-cli.md:156:`bin/pair` truncates the file once at session launch; all components then append
atlas/how-to-bring-up-a-new-harness-cli.md:197:the same trust level as the existing scrollback logs. `bin/pair` removes it on quit.
atlas/go-migration-inventory.md:50:| `bin/pair` / `bin/pair-shell` / `cmd/internal/launcher` / `cmd/internal/entrypoint` | Go public entrypoint plus retained shell launcher | user shell, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` is generated from `cmd/pair-go` and resolves `PAIR_HOME` / sibling root / build-time `defaultPairHome`, then execs `<asset-root>/bin/pair-shell` with `pair`-compatible argv/env. `bin/pair-shell` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` shares the same compatibility handoff. | `bin/pair-shell` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves invocation mode, asset root, and compatibility request; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Public entrypoint is Go-owned as of #79; `bin/pair-shell` is retained because real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
atlas/go-migration-inventory.md:51:| `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling Go-built `pair`. | Resolves symlinks; depends on generated `bin/pair`, retained `bin/pair-shell`, and `bin/lib/dev-rebuild.sh`. | retained dev wrapper so developer launches exercise the public Go entrypoint | P1 |
atlas/go-migration-inventory.md:52:| `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
atlas/go-migration-inventory.md:53:| `zellij/layouts/main.kdl` | zellij native asset | `bin/pair` via `zellij --new-session-with-layout` | Defines agent and draft panes; shell expands Pair env at pane start. | Calls `pair-wrap`; calls `nvim -u "$PAIR_HOME/nvim/init.lua"`; writes `pane-<tag>-<agent>.json`; writes draft nvim pid file. | native-asset, packaged adjacent/embedded | P0 |
atlas/go-migration-inventory.md:54:| `zellij/config.kdl` | zellij native asset | zellij session config from `bin/pair` | Global keybinds, copy command, scrollback buffer, pane frames. | Calls `copy-on-select.sh`, `pair-help`, `pair-scrollback-open`, `pair-changelog-open`; routes quit/restart/compact through nvim functions. | native-asset, packaged adjacent/embedded | P0 |
atlas/go-migration-inventory.md:70:| `bin/pair-title.sh` | Bash stateful poller | `bin/pair` ensure_title_poller | `pair-title.sh <tag> <agent>`; long-running 60s poller; test hook `PAIR_TITLE_TEST_CALL`. | Reads/writes title pid, pane json, cmux owner files; calls `pair-context`, zellij, ps, cmux. | stateful shell-glue; explicit #78 candidate | P1 |
atlas/go-migration-inventory.md:71:| `bin/pair-session-watch.sh` / `cmd/pair-session-watch` / `cmd/internal/sessionwatch` | Shell compatibility shim plus Go stateful watcher | `bin/pair` create path | `pair-session-watch.sh <agent> <tag> <cwd> [agent-args...]` execs the Go command; background 60s watcher; no-op for claude. | Reads agent pidfile, lsof/ps, native session dirs; writes config JSON atomically; logs adapt events through `cmd/internal/adapt`. | Go-owned watcher with legacy shim retained while `bin/pair` calls the `.sh` name (#78) | P1 |
atlas/go-migration-inventory.md:130:- #74 should add a Go dispatcher without changing `bin/pair`.
atlas/go-migration-inventory.md:139:  `bin/pair`, with argv/env preserved and missing-launcher diagnostics.
atlas/go-migration-inventory.md:165:- `bin/pair`
atlas/architecture.md:49:As of #79, the public `bin/pair` command is a Go-built entrypoint from
atlas/architecture.md:89:### `bin/pair` / `bin/pair-shell` — launcher
atlas/architecture.md:91:`bin/pair` is the Go public entrypoint. It resolves the asset root, then execs
atlas/architecture.md:135:The picker is bypassed in either flavor — Alt+n's argv carries an explicit resume token, and Shift+Alt+N has no saved config to pick against. A third marker field, `continue=<slug>` (#55, written by the in-session compaction branch in `bin/pair`, not `pair-restart.sh`), rides the `new_session=1` path but re-execs `pair continue <slug> <agent> -- <args>` instead of a plain restart — see "In-session compaction" below.
atlas/architecture.md:156:- `copy_command "copy-on-select.sh"` — on every selection finalize (mouse-up after drag), zellij pipes the selected text to this script. `copy_command` replaces zellij's default OS-clipboard write, so the script does that part too. Resolved by PATH (which `bin/pair` populated).
atlas/architecture.md:167:- `Shift+Alt+N` — routed through nvim to `:lua PairConfirmRestartNewSession()` — Y/N modal then `pair-restart.sh --new-session` (restart with a fresh agent conversation). See "Reload / restart in place" under `bin/pair`.
atlas/architecture.md:168:- `Alt+Shift+C` (`Alt C` / `Ctrl Alt c`) — routed through nvim to `:lua PairConfirmCompact()` — Y/N modal then `send_to_agent(<compaction prompt>)` (#55). Unlike the restart binds it does NOT shell out directly: distilling a continuation needs the agent's judgment, so it asks the agent to write a continuation + run `pair continue <slug>` (context-aware → compaction). See "In-session compaction" under `bin/pair`.
atlas/architecture.md:198:Touches the marker file `~/.cache/pair/quit-$ZELLIJ_SESSION_NAME`, then `exec zellij kill-session $ZELLIJ_SESSION_NAME`. The kill terminates the session including the script itself; on the launcher side, `bin/pair` resumes, sees the marker, and runs `delete-session --force` to clean up the resurrect entry.
atlas/architecture.md:208:1. **Outer-TTY capture (in `bin/pair`).** Before invoking zellij, on every attach (both create and reattach branches), pair calls `tty(1)`. The result is the path of pair's controlling TTY — which is precisely the outer PTY (the one allocated by whatever wraps pair: cmux, a terminal emulator, etc.). That path gets written to `$DATA_DIR/outer-tty-<tag>`. Refreshed on every attach because the outer PTY changes across detach/reattach, while pane-shell env stays frozen at zellij session-creation time (env-var approaches would go stale).
atlas/architecture.md:280:**In-session compaction (#55, `Alt+Shift+C`).** `pair continue <slug>` is *context-aware*: run from a normal shell it fresh-starts (above); run from INSIDE its own live pane it **compacts** — copy-parks the scrollback as a recovery net, writes a restart marker carrying a new `continue=<slug>` field (same tag, `new_session=1`), and kills the session. The outer `bin/pair` then `handle_restart_marker`-re-execs `pair continue <slug> <agent> -- <args>` (now outside zellij → the fresh-start branch), so the session reincarnates under the same tag with a clean conversation seeded from the continuation. Detection is **ancestry-based** (`in_zellij_pane`) plus a `ZELLIJ_SESSION_NAME == pair-<tag>` confirmation — never `$ZELLIJ*` env alone, since cmux propagates those to sibling non-pair panes (a false positive would park+kill the wrong session). The branch sits *before* the `in_zellij_pane` guard (which otherwise rejects any in-pane `pair`), with `DATA_DIR` + `handle_restart_marker` hoisted above the guard so test seams can drive them. Seams (test-only): `PAIR_FORCE_IN_SESSION`, `PAIR_FAKE_IN_ZELLIJ`, `PAIR_KILL_CMD`, `PAIR_TEST_CALL`, `PAIR_REEXEC_CAPTURE`. The trigger is the `Alt+Shift+C` keybind (`Alt C` / `Ctrl Alt c` → `PairConfirmCompact` → an agent-agnostic prompt that **defers to the `continuation` datatype procedure** — flush-first, then write the continuation and run `pair continue <slug>` — rather than enumerating a section skeleton inline, so the prompt can't drift out of sync with the datatype; that drift was the bug pair#61 fixed); the outer process suppresses the Alt+x park nudge whenever a restart marker is pending (a restart isn't a quit).
atlas/architecture.md:536:  the exported `PAIR_SESSION_ID` (set by `bin/pair` at launch for claude-fresh /
atlas/architecture.md:576:1. **Pre-write at launch (`bin/pair`).** Two paths:
atlas/architecture.md:579:2. **Watcher (`cmd/pair-session-watch` via `bin/pair-session-watch.sh`, codex/agy only).** Spawned in the background by `bin/pair` on the create path, right before the zellij launch. The `.sh` file is now a compatibility shim; the stateful discovery logic lives in Go. Two discovery paths:
atlas/architecture.md:591:| claude | `~/.claude/projects/<encoded-cwd>/<id>.jsonl` | filename | `--session-id` pre-injected by `bin/pair` (deterministic) |
atlas/architecture.md:603:**Create-flow prompt (`bin/pair`).** When the create path commits a tag, pair reads `config-<tag>-<agent>.json`. If present, it runs the per-agent stale-id check (claude: `[ -f .../<id>.jsonl ]`; codex: `find ~/.codex/sessions -name "*<id>*"`; agy: check conversation DB) and fzf-prompts the user with up to three options:
atlas/architecture.md:645:**Inside-flow choreography.** `nvim/init.lua`'s `pair_confirm_restart_impl` shells out `pair rename --restart-check` after the user enters a new tag, re-prompting on each rejection. On accept it execs `pair-restart.sh --rename-to <new>`. `pair-restart.sh` writes `rename_to=<new>` into the restart marker (`~/.cache/pair/restart-<SESSION>`) alongside the existing `tag`, `agent`, `new_session` fields. `handle_restart_marker` in `bin/pair` runs after `cleanup_quit_marker` (so the zj delete-session has cleared the live-old gate) and if `rename_to` is set, invokes `"$0" rename <old> <new>` — full check. On success, the working tag for the re-exec is swapped to `<new>` (so `config-<new>-<agent>.json`, the just-renamed file, is what gets resumed). On failure, a 2-second visible stderr warning is printed and the restart continues with the original tag — the user is never stranded.
atlas/architecture.md:667:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-<tag>` — single-line file recording which agent binary was launched in the session (`claude`, `codex`, ...). Written once at session create; read by `pair list` to display the agent column, and by `bin/pair`'s tag-restart agent-inference. Removed on full quit. The agent isn't otherwise recoverable post-create — env vars are frozen in pane shells, and custom session names (e.g. `pair-bugfix`) don't carry the agent in the name.
atlas/architecture.md:669:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/config-<tag>-<agent>.json` — saved restart configuration for `(tag, agent)` (issue #000016, #000020). `{ agent, args, session_id }`. For claude, written synchronously by `bin/pair` before zellij launch (`--session-id` is deterministic). For codex/agy, written by the Go `pair-session-watch` command once the agent's session file is discovered via lsof. Read by `bin/pair`'s create-flow prompt and by the post-Alt+x hint. Survives Alt+x (unlike `agent-<tag>`, which is cleared) — that's the whole point: it's the bridge between two pair launches against the same tag.
atlas/architecture.md:681:Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/adapt-<tag>.jsonl` — the adaptation flight recorder (issue #000045). One JSON line per harness-adaptation trigger (`{ts, comp, agent, aspect, signal, outcome, detail}`), appended concurrently by `cmd/pair-wrap`, `cmd/pair-slug`, `cmd/pair-session-watch`, and `nvim/adapt.lua` — all writing one shared schema. Truncated once at session launch by `bin/pair` (so multi-process `O_APPEND` never races) and removed on full quit. Read by `doctor/doctor.sh` to surface integration drift (near-miss/fail signals). See `atlas/how-to-bring-up-a-new-harness-cli.md` §3 for the signal registry.
atlas/architecture.md:687:`bin/pair` prepends `$PAIR_HOME/bin` to `$PATH` before exec'ing zellij. zellij and all its child processes (panes, copy_command, Run actions) inherit the PATH and can resolve `clipboard-to-pane.sh`, `copy-on-select.sh`, `pair-quit.sh` by bare name. This lets the zellij KDL configs reference scripts without `sh -c` env-var quoting hacks.
atlas/architecture.md:695:- **Dev** — `pair-dev` (#000046). Exports `PAIR_DEV=1` and execs `pair`; `bin/pair` then runs `make build` (via `bin/lib/dev-rebuild.sh`'s `dev_rebuild`) on the **create path**, before the layout execs pair-wrap, so `$PAIR_HOME/bin` holds a fresh build. Restart-safe: `PAIR_DEV` survives `handle_restart_marker`'s `exec "$0"`, so Alt+n / Shift+Alt+N rebuild too; a plain attach (no new wrapper spawned) correctly skips it. Deployed launches (`PAIR_DEV` unset) invoke no toolchain.
tests/dev-rebuild-test.sh:10:# critical because bin/pair runs under `set -e` and a failed restart-time build
cmd/internal/launcher/args.go:22:// ParseArgs parses pair-go launch args. It intentionally supports only the
cmd/internal/launcher/args.go:25:func ParseArgs(argv []string) (LaunchArgs, error) {
cmd/internal/launcher/args.go:34:		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %s is not implemented by pair-go launch; use bin/pair", argv[0])}
cmd/internal/launcher/run.go:37:	args, err := ParseArgs(argv)
cmd/internal/launcher/args_test.go:9:	args, err := ParseArgs(nil)
cmd/internal/launcher/args_test.go:11:		t.Fatalf("ParseArgs returned error: %v", err)
cmd/internal/launcher/args_test.go:25:	args, err := ParseArgs([]string{"codex", "--", "-p", "say hi"})
cmd/internal/launcher/args_test.go:27:		t.Fatalf("ParseArgs returned error: %v", err)
cmd/internal/launcher/args_test.go:38:	args, err := ParseArgs([]string{"--", "--dangerously-skip-permissions"})
cmd/internal/launcher/args_test.go:40:		t.Fatalf("ParseArgs returned error: %v", err)
cmd/internal/launcher/args_test.go:51:	args, err := ParseArgs([]string{"resume", "pair-demo"})
cmd/internal/launcher/args_test.go:53:		t.Fatalf("ParseArgs returned error: %v", err)
cmd/internal/launcher/args_test.go:64:	_, err := ParseArgs([]string{"codex", "extra"})
cmd/internal/launcher/args_test.go:66:		t.Fatal("ParseArgs returned nil error")
cmd/internal/launcher/args_test.go:79:			_, err := ParseArgs([]string{verb})
cmd/internal/launcher/args_test.go:81:				t.Fatal("ParseArgs returned nil error")

exec
/bin/zsh -lc "rg -n 'PAIR_TEST_CALL|PAIR_DEBUG_ARGS|exec \"\\"'$0"|restart marker|handle_restart|cleanup_quit_marker|rename_paths_for|resume_extra|config-.*agent'"' bin/pair-shell tests/*.sh" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
tests/pair-rename.sh:33:# the families enumerated in bin/pair's `rename_paths_for` helper.
tests/pair-rename.sh:48:        > "$dd/config-$tag-$agent.json"
tests/review-toggle-test.sh:135:-- fall back to config-<tag>-<agent>.json when PAIR_SESSION_ID is empty.
bin/pair-shell:396:        rename_paths_for() {
bin/pair-shell:428:        done < <(rename_paths_for "$new_tag" "$rename_dd")
bin/pair-shell:438:        rename_paths_for "$old_tag" "$rename_dd" > "$old_paths_tmp"
bin/pair-shell:443:            # rename_paths_for as "<dir>/<chunk-with-$old_tag>", so the
bin/pair-shell:505:        # rename runs later, post-kill, from handle_restart_marker.
bin/pair-shell:670:# (handle_restart_marker below): it pins the new run to the same tag as
bin/pair-shell:682:# handle_restart_marker (defined here) and the branch can all use it.
bin/pair-shell:690:# (ariadne#91). Extracted from cleanup_quit_marker so BOTH the Alt+x quit path
bin/pair-shell:710:# Re-read the restart marker dropped by pair-restart.sh (Alt+n / Shift+Alt+N)
bin/pair-shell:712:# Moved up from its original site (~1323) so the PAIR_TEST_CALL dispatcher below
bin/pair-shell:716:handle_restart_marker() {
bin/pair-shell:772:            _reexec "$0" continue "$r_continue" "$r_agent" ${r_args:+-- $r_args}
bin/pair-shell:775:            _reexec "$0" "$r_agent" -- $r_args
bin/pair-shell:777:            _reexec "$0" "$r_agent"
bin/pair-shell:786:    local resume_extra=""
bin/pair-shell:789:            claude)        resume_extra="--resume $r_sid" ;;
bin/pair-shell:790:            codex)         resume_extra="resume $r_sid" ;;
bin/pair-shell:791:            agy)           resume_extra="--conversation $r_sid" ;;
bin/pair-shell:800:    if [ "$r_agent" = "codex" ] && [ -n "$resume_extra" ]; then
bin/pair-shell:801:        combined="${resume_extra}${r_args:+ $r_args}"
bin/pair-shell:803:        combined="${r_args}${r_args:+${resume_extra:+ }}${resume_extra}"
bin/pair-shell:807:        _reexec "$0" "$r_agent" -- $combined
bin/pair-shell:809:        _reexec "$0" "$r_agent"
bin/pair-shell:838:# it on stdout. Current launches write `config-<tag>-<agent>.json`, but older
bin/pair-shell:849:    local canonical="$DATA_DIR/config-${tag}-${agent}.json"
bin/pair-shell:888:# PAIR_TEST_CALL can unit-test cmux_rename_workspace without a live session
bin/pair-shell:927:# harness can unit-test park_scrollback / handle_restart_marker against the REAL
bin/pair-shell:930:if [ -n "${PAIR_TEST_CALL:-}" ]; then
bin/pair-shell:932:    "$PAIR_TEST_CALL" ${PAIR_TEST_ARGS:-}
bin/pair-shell:949:# files (config-<tag>-<agent>.json, scrollback-<tag>-<agent>.raw, ...).
bin/pair-shell:990:# config-<tag>-<agent>.json (issue #000016) covers the case where the
bin/pair-shell:1013:if [ "${PAIR_DEBUG_ARGS:-0}" = "1" ]; then
bin/pair-shell:1022:# still appending to .raw), drop a restart marker carrying the slug, and kill
bin/pair-shell:1024:# under the same tag (handle_restart_marker, hoisted above). PAIR_TAG/PAIR_AGENT
bin/pair-shell:1082:#   - cleanup_quit_marker (Alt+x path) — reaps the just-quit tag.
bin/pair-shell:1086:#     mid-session) that left no quit marker, so cleanup_quit_marker
bin/pair-shell:1241:# host reboot, or pair upgrade leave no Alt+x marker, so cleanup_quit_marker
bin/pair-shell:1520:cleanup_quit_marker() {
bin/pair-shell:1594:        # show the user the one-liner that resumes it. config-<tag>-<agent>
bin/pair-shell:1650:# After cleanup_quit_marker, check for the restart marker written by
bin/pair-shell:1662:# from the marker (captured before cleanup_quit_marker wiped agent-<tag>);
bin/pair-shell:1664:# handle_restart_marker() is defined earlier (hoisted for #55 so the
bin/pair-shell:1665:# PAIR_TEST_CALL dispatcher can drive it before the picker). It is still
bin/pair-shell:1696:    # disown so cleanup_quit_marker's `kill <pid>` later doesn't spawn a
bin/pair-shell:1720:    cleanup_quit_marker
bin/pair-shell:1721:    handle_restart_marker
bin/pair-shell:2044:# config. Pre-writing here means config-<tag>-<agent>.json is right from
bin/pair-shell:2286:cleanup_quit_marker
bin/pair-shell:2287:handle_restart_marker
tests/cmux-ownership-test.sh:11:# Drives the REAL bin/pair through its PAIR_TEST_CALL seam with process-level
tests/cmux-ownership-test.sh:55:        PAIR_TEST_CALL=cmux_rename_workspace PAIR_TEST_ARGS="$title" \
tests/cmux-ownership-test.sh:95:    PAIR_TAG=211 PAIR_TEST_CALL=cmux_rename_workspace PAIR_TEST_ARGS="pair-211" \
tests/pair-continue-test.sh:4:# PAIR_DEBUG_ARGS probe (which resolves argv, then exits before any zellij
tests/pair-continue-test.sh:47:  ( cd "$RT" && XDG_DATA_HOME="$RT/xdg" PAIR_DEBUG_ARGS=1 "$PAIR" "$@" 2>/dev/null ) \
tests/pair-continue-test.sh:116:# PAIR_FORCE_IN_SESSION / PAIR_FAKE_IN_ZELLIJ / PAIR_KILL_CMD / PAIR_TEST_CALL /
tests/pair-continue-test.sh:163:  PAIR_TEST_CALL=park_scrollback PAIR_TEST_ARGS="demo claude" "$PAIR" >/dev/null 2>&1 )
tests/pair-continue-test.sh:167:# 5. handle_restart_marker continue= → re-exec argv
tests/pair-continue-test.sh:177:  PAIR_TEST_CALL=handle_restart_marker PAIR_REEXEC_CAPTURE="$CAP" "$PAIR" >/dev/null 2>&1 )
tests/pair-continue-test.sh:183:# REAL bin/pair helper seams through PAIR_TEST_CALL with process-level fakes.
tests/pair-continue-test.sh:215:if ( cd "$HRT" && helper_env PAIR_TEST_CALL=session_blocks_reuse PAIR_TEST_ARGS="pair-demo" "$PAIR" >/dev/null 2>&1 ); then
tests/pair-continue-test.sh:226:if ( cd "$HRT" && helper_env PAIR_TEST_CALL=session_blocks_reuse PAIR_TEST_ARGS="pair-demo" "$PAIR" >/dev/null 2>&1 ); then
tests/pair-continue-test.sh:236:if ( cd "$HRT" && helper_env PAIR_TEST_CALL=session_blocks_reuse PAIR_TEST_ARGS="pair-demo" "$PAIR" >/dev/null 2>&1 ); then
tests/pair-continue-test.sh:246:if out="$( cd "$HRT" && helper_env PAIR_TEST_CALL=resolve_config_file PAIR_TEST_ARGS="demo codex" "$PAIR" 2>/dev/null )" \
tests/pair-continue-test.sh:256:if out="$( cd "$HRT" && helper_env PAIR_TEST_CALL=resolve_config_file PAIR_TEST_ARGS="bad codex" "$PAIR" 2>/dev/null )" \
tests/pair-continue-test.sh:267:if out="$( cd "$HRT" && helper_env PAIR_TEST_CALL=resolve_config_file PAIR_TEST_ARGS="keep codex" "$PAIR" 2>/dev/null )" \
tests/pair-continue-test.sh:278:if ( cd "$HRT" && helper_env PAIR_TEST_CALL=agent_session_exists PAIR_TEST_ARGS="agy agy-session ''" "$PAIR" >/dev/null 2>&1 ); then
tests/pair-continue-test.sh:286:if ( cd "$HRT" && helper_env PAIR_TEST_CALL=agent_session_exists PAIR_TEST_ARGS="agy old-session ''" "$PAIR" >/dev/null 2>&1 ); then

exec
/bin/zsh -lc 'git diff 776ecce27bc3643bc5598df559c52a3bb5424efe..HEAD -- workshop/issues/000079-go-packaging-consolidation.md workshop/plans/000079-go-packaging-consolidation-plan.md atlas/architecture.md atlas/go-migration-inventory.md README.md' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
diff --git a/README.md b/README.md
index eb841b8..eb3af77 100644
--- a/README.md
+++ b/README.md
@@ -200,12 +200,13 @@ pair rename <old> <new>          # rename every tag-scoped file in
 pair -h, --help                  # show full help
 ```
 
-`pair-go launch ...` is the Go-owned migration entrypoint for testing the
-launcher path. It accepts the same arguments after `launch` that `pair` accepts
-directly, then hands off to the current `pair` launcher for one migration
-window. In a dev shell sourced from `../ariadne/construct/dev-aliases.sh`,
-`pair-go` rebuilds from `cmd/pair-go` automatically before running; no
-`pair-go-dev` command is needed.
+The installed `pair` command is Go-owned. For this migration window it resolves
+the adjacent Pair asset root and hands off to `bin/pair-shell`, the retained
+shell launcher that still owns the zellij lifecycle. `pair-go launch ...` remains
+the explicit development dispatcher path and accepts the same arguments after
+`launch` that `pair` accepts directly. In a dev shell sourced from
+`../ariadne/construct/dev-aliases.sh`, `pair` and `pair-go` rebuild from
+`cmd/pair-go` automatically before running; no `pair-go-dev` command is needed.
 
 Use `--` to separate pair's positional from agent flags. Without it, pair only takes `<agent>` as a positional and everything else is rejected.
 
diff --git a/atlas/architecture.md b/atlas/architecture.md
index 65456bd..17fce6a 100644
--- a/atlas/architecture.md
+++ b/atlas/architecture.md
@@ -13,8 +13,9 @@ contract for the Go packaging migration lives in
 [Go migration inventory](go-migration-inventory.md).
 
 ```
-bin/pair                     # entry point (launcher)
-bin/pair-go                  # opt-in Go dispatcher + launch handoff; public launcher remains bin/pair
+bin/pair                     # Go public entrypoint; execs bin/pair-shell during migration
+bin/pair-shell               # retained shell launcher: zellij lifecycle + prompt UI
+bin/pair-go                  # explicit Go dispatcher + launch handoff
 bin/clipboard-to-pane.sh     # read clipboard, hand off to nvim's PairPasteQuote
 bin/copy-on-select.sh        # invoked by zellij copy_command on mouse-up
 bin/pair-quit.sh             # invoked by Alt+x — marks + kills session
@@ -45,19 +46,26 @@ internal subcommands or dispatch modes behind that primary binary (`pair wrap`,
 `pair continuation`, `pair scribe`) instead of staying as independently managed
 installed commands forever.
 
-As of #77, `bin/pair-go` is the Go-owned launch entrypoint under test:
-`pair-go launch ...` resolves its own executable, finds sibling `bin/pair`, and
-execs it with the same argv/env that `pair` would have received directly. This
-makes `pair-go launch claude`, `pair-go launch resume <tag>`, `pair-go launch
-continue ...`, `pair-go launch list`, and `pair-go launch rename ...`
-meaningful dogfood commands without replacing the stable public `pair` command
-yet. A developer shell sourced from `../ariadne/construct/dev-aliases.sh`
-rebuilds `cmd/pair-go` automatically; no `pair-go-dev` command exists.
+As of #79, the public `bin/pair` command is a Go-built entrypoint from
+`cmd/pair-go`. Direct `pair ...` and explicit `pair-go launch ...` share one
+compatibility handoff: resolve the Pair asset root, then exec
+`<asset-root>/bin/pair-shell` with `pair`-compatible argv/env. Asset root
+resolution is ordered: explicit `PAIR_HOME`, executable sibling root, then the
+build-time `defaultPairHome` injected by Make/Homebrew for copied installs.
+Native `nvim/` and `zellij/` assets remain adjacent to that root.
+
+`pair-go` remains the development dispatcher for helper routes and explicit
+launch testing: `pair-go launch claude`, `pair-go launch resume <tag>`,
+`pair-go launch continue ...`, `pair-go launch list`, and
+`pair-go launch rename ...` all reach the same shell launcher as direct `pair`.
+A developer shell sourced from `../ariadne/construct/dev-aliases.sh` rebuilds
+`cmd/pair-go` automatically; no `pair-go-dev` command exists.
 
 The earlier #75 pure launcher core remains available as internal decision logic,
 but real zellij lifecycle, prompt/fzf UI, restart/quit cleanup, cmux ownership,
 dev rebuild, continuation, rename, config/session migration, and title-poller
-behavior remain shell-owned through `bin/pair` until later migration issues.
+behavior remain shell-owned through `bin/pair-shell` until later migration
+issues.
 
 As of #76, the same dispatcher also has the first implemented helper routes:
 `pair-go context <tag> <agent>` and `pair-go scrollback-render ...`. Both routes
@@ -78,9 +86,13 @@ keybindings, scrollback, changelog, continuation, and review flows still work.
 The detailed disposition table is maintained in
 [Go migration inventory](go-migration-inventory.md), not duplicated here.
 
-### `bin/pair` — launcher
+### `bin/pair` / `bin/pair-shell` — launcher
 
-Resolves `$PAIR_HOME` from its own real path (portable bash, no `readlink -f`), prepends `$PAIR_HOME/bin` to `$PATH` (idempotent across re-launches) so all helper scripts resolve by bare name in zellij configs and keybinds, parses argv — first positional is `$PAIR_AGENT` (default `claude`), everything after `--` is joined into `$PAIR_AGENT_ARGS`, extra positionals before `--` are an error with a usage hint, defaults `$PAIR_TAG` to the cwd basename (the create-flow prompt or `pair resume <tag>` overrides it), resolves `$PAIR_DATA_DIR` to `${XDG_DATA_HOME:-$HOME/.local/share}/pair`, runs a one-time migration of any old `~/scratch/pair-{draft,log}-*` files, and dispatches:
+`bin/pair` is the Go public entrypoint. It resolves the asset root, then execs
+`bin/pair-shell` with argv[0] presented as `pair`; the shell script below still
+owns the launcher lifecycle during this migration window.
+
+`bin/pair-shell` resolves `$PAIR_HOME` from its own real path (portable bash, no `readlink -f`), prepends `$PAIR_HOME/bin` to `$PATH` (idempotent across re-launches) so all helper scripts resolve by bare name in zellij configs and keybinds, parses argv — first positional is `$PAIR_AGENT` (default `claude`), everything after `--` is joined into `$PAIR_AGENT_ARGS`, extra positionals before `--` are an error with a usage hint, defaults `$PAIR_TAG` to the cwd basename (the create-flow prompt or `pair resume <tag>` overrides it), resolves `$PAIR_DATA_DIR` to `${XDG_DATA_HOME:-$HOME/.local/share}/pair`, runs a one-time migration of any old `~/scratch/pair-{draft,log}-*` files, and dispatches:
 
 A leading `pair resume <tag>` is recognized as a subcommand verb (alongside `list` / `help`): it skips both the picker and the name prompt, attaches if `pair-<tag>` already exists in any state, otherwise creates with that tag. When `resume` is in play, the agent is inferred from saved state on disk (`agent-<tag>` for live/recently-detached sessions; the agent embedded in the `config-<tag>-<agent>.json` filename otherwise) — so a single tag is enough to restart, regardless of which agent was originally paired with it. See "Tag-restart" below.
 
diff --git a/atlas/go-migration-inventory.md b/atlas/go-migration-inventory.md
index b03e89d..e095add 100644
--- a/atlas/go-migration-inventory.md
+++ b/atlas/go-migration-inventory.md
@@ -47,8 +47,8 @@ Priority is packaging impact first, then reliability/testability:
 
 | Artifact | Type | Callers | Runtime contract | Files/env | Disposition | Priority |
 |---|---|---|---|---|---|---|
-| `bin/pair` / `cmd/internal/launcher` / `cmd/internal/entrypoint` | Bash public launcher plus Go-owned launch handoff | user shell, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` (#77) resolves sibling `bin/pair` from the `pair-go` executable and execs it with `pair`-compatible argv/env. | `bin/pair` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves the compatibility handoff; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Go handoff entrypoint with `bin/pair` retained as stable public launcher and compat target through #77; real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
-| `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling `pair`. | Resolves symlinks; depends on `bin/pair` and `bin/lib/dev-rebuild.sh`. | compat-shim, likely stays as developer wrapper until Go entrypoint has dev mode | P1 |
+| `bin/pair` / `bin/pair-shell` / `cmd/internal/launcher` / `cmd/internal/entrypoint` | Go public entrypoint plus retained shell launcher | user shell, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` is generated from `cmd/pair-go` and resolves `PAIR_HOME` / sibling root / build-time `defaultPairHome`, then execs `<asset-root>/bin/pair-shell` with `pair`-compatible argv/env. `bin/pair-shell` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` shares the same compatibility handoff. | `bin/pair-shell` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves invocation mode, asset root, and compatibility request; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Public entrypoint is Go-owned as of #79; `bin/pair-shell` is retained because real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
+| `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling Go-built `pair`. | Resolves symlinks; depends on generated `bin/pair`, retained `bin/pair-shell`, and `bin/lib/dev-rebuild.sh`. | retained dev wrapper so developer launches exercise the public Go entrypoint | P1 |
 | `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
 | `zellij/layouts/main.kdl` | zellij native asset | `bin/pair` via `zellij --new-session-with-layout` | Defines agent and draft panes; shell expands Pair env at pane start. | Calls `pair-wrap`; calls `nvim -u "$PAIR_HOME/nvim/init.lua"`; writes `pane-<tag>-<agent>.json`; writes draft nvim pid file. | native-asset, packaged adjacent/embedded | P0 |
 | `zellij/config.kdl` | zellij native asset | zellij session config from `bin/pair` | Global keybinds, copy command, scrollback buffer, pane frames. | Calls `copy-on-select.sh`, `pair-help`, `pair-scrollback-open`, `pair-changelog-open`; routes quit/restart/compact through nvim functions. | native-asset, packaged adjacent/embedded | P0 |
@@ -114,10 +114,14 @@ Nvim shell-outs and binary dependencies:
 
 Build/install callers:
 
-- `make build` builds `GO_BINS` into `bin/`.
-- `make install` copies `GO_BINS` to `~/.local/bin` and symlinks `SHELL_BINS`
-  (`pair`, `pair-dev`) beside them so installed `pair-go launch ...` can resolve
-  sibling `pair`.
+- `make build` builds `GO_BINS` into `bin/`; `pair` and `pair-go` are both built
+  from `cmd/pair-go` with `defaultPairHome=$(CURDIR)`.
+- `make install` copies `GO_BINS` to `~/.local/bin` and symlinks only retained
+  shell wrappers such as `pair-dev`. Installed `pair` is a regular Go binary;
+  if it has no sibling assets, it falls back to the build-time source root.
+- Homebrew installs `bin/`, `nvim/`, and `zellij/` under `libexec`, then builds
+  Go `pair`, `pair-go`, and required runtime helpers into `libexec/bin` with
+  `defaultPairHome=#{libexec}`.
 - `pair-dev` relies on `make build`, then zellij's PATH lookup resolves fresh
   repo `bin/` binaries.
 
@@ -132,13 +136,14 @@ Build/install callers:
   binary names remain live for shell/Lua callers. `pair slug` remains a later
   candidate.
 - #77 made `pair-go launch ...` a meaningful Go-owned compatibility handoff to
-  `bin/pair`, with argv/env preserved and missing-launcher diagnostics. The
-  stable public `pair` script remains unchanged for this migration window.
+  `bin/pair`, with argv/env preserved and missing-launcher diagnostics.
 - #78 ported the session-id watcher to `cmd/pair-session-watch` with
   `bin/pair-session-watch.sh` retained as a shim. `pair-title.sh` remains the
   next stateful shell candidate because it owns a separate UI title-poller
   surface.
-- #79 owns whether `nvim/` and `zellij/` are embedded or installed adjacent.
+- #79 made public `pair` a Go-built entrypoint, renamed the shell launcher to
+  `bin/pair-shell`, and chose adjacent `nvim/` / `zellij/` assets for local and
+  Homebrew installs.
 
 ## Coverage Ledger
 
diff --git a/workshop/issues/000079-go-packaging-consolidation.md b/workshop/issues/000079-go-packaging-consolidation.md
index db5a607..6f57a45 100644
--- a/workshop/issues/000079-go-packaging-consolidation.md
+++ b/workshop/issues/000079-go-packaging-consolidation.md
@@ -33,19 +33,19 @@ Approved design: make the installed public `pair` command Go-owned while keeping
 
 ## Done when
 
-- [ ] Packaging installs the primary Go `pair` command and required assets coherently.
-- [ ] Obsolete compatibility shims are removed or explicitly retained with a reason.
-- [ ] README and atlas describe the new install/runtime layout.
-- [ ] Clean install and upgrade paths are verified.
-- [ ] Pair remains usable after merge.
+- [x] Packaging installs the primary Go `pair` command and required assets coherently.
+- [x] Obsolete compatibility shims are removed or explicitly retained with a reason.
+- [x] README and atlas describe the new install/runtime layout.
+- [x] Clean install and upgrade paths are verified.
+- [x] Pair remains usable after merge.
 
 ## Plan
 
-- [ ] Choose embedded vs adjacent asset strategy.
-- [ ] Update build/install/Homebrew wiring.
-- [ ] Remove or document remaining shims.
-- [ ] Run clean install/upgrade verification.
-- [ ] Update docs and atlas.
+- [x] Choose embedded vs adjacent asset strategy.
+- [x] Update build/install/Homebrew wiring.
+- [x] Remove or document remaining shims.
+- [x] Run clean install/upgrade verification.
+- [x] Update docs and atlas.
 
 Detailed implementation plan: `workshop/plans/000079-go-packaging-consolidation-plan.md`.
 
@@ -74,3 +74,5 @@ Created from #72 as the final consolidation milestone. This should land only aft
 ### 2026-06-30
 
 Claimed after #78 landed. Chose adjacent native assets and Go public entrypoint: build `cmd/pair-go` as installed `pair`, retain `pair-go` as the development dispatcher alias, and keep the current shell launcher as an internal compatibility handoff while the zellij lifecycle remains shell-owned. Wrote durable plan at `workshop/plans/000079-go-packaging-consolidation-plan.md`. Plan-quality found missing Homebrew and upgrade specificity; tightened the plan to include sibling formula `../homebrew-pair/Formula/pair.rb`, a concrete old-symlink-to-Go-binary upgrade test, and a single decided asset strategy: local installs stay source-tree adjacent, Homebrew installs an adjacent `libexec` tree. Second plan-quality pass found asset-root and tracked-file ambiguity; tightened the plan again so pure `AssetRoot` resolves `PAIR_HOME` / sibling root / build-time `defaultPairHome`, and so `bin/pair-shell` is tracked while generated `bin/pair` is ignored. Estimate derived with v3.1 calibration; calibration source is marked stale by `sdlc estimate-source`, so the number is provisional but uses the required method.
+
+Implemented #79 packaging consolidation. `bin/pair` is now generated from `cmd/pair-go`, `bin/pair-shell` is the tracked compatibility launcher, local install copies a regular Go `pair` binary, and Homebrew builds Go `pair` / `pair-go` plus required runtime helpers into `libexec/bin` with adjacent native assets. Homebrew tap evidence: sibling repo `../homebrew-pair` commit `3aeb2a6 pair: build Go public entrypoint` updates `Formula/pair.rb`. Verification: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`; `make build`; `make test-pair-go-install-layout`; `bin/pair --help`; `bin/pair-go launch --help`; `bin/pair-dev --help`; `make test-dev-rebuild test-session-watch test-continue`; `go test ./... -count=1`; `ruby -c ../homebrew-pair/Formula/pair.rb`; stale-doc grep for old #77 packaging wording. `brew test --formula ../homebrew-pair/Formula/pair.rb` was not available on this Homebrew (`invalid option: --formula`), so the formula was syntax-checked locally rather than installed over the operator environment. Linux smoke was not run because this workspace is Darwin-only (`uname -s` => `Darwin`) and no Linux runner is configured.
diff --git a/workshop/plans/000079-go-packaging-consolidation-plan.md b/workshop/plans/000079-go-packaging-consolidation-plan.md
index d21b2b0..136b707 100644
--- a/workshop/plans/000079-go-packaging-consolidation-plan.md
+++ b/workshop/plans/000079-go-packaging-consolidation-plan.md
@@ -18,7 +18,7 @@
 |------|----------|--------|
 | `EntrypointMode` | `cmd/internal/entrypoint/mode.go` | new |
 | `AssetRoot` | `cmd/internal/entrypoint/asset_root.go` | new |
-| `LegacyPairRequest` | `cmd/internal/entrypoint/launch.go` | modified |
+| `LegacyLaunchRequest` | `cmd/internal/entrypoint/launch.go` | modified |
 
 **EntrypointMode** — Determines whether one executable invocation should behave as public `pair` or development `pair-go`.
 - **Relationships:** 1:1 with the executable basename; `cmd/pair-go/main.go` owns the argv/env IO and calls this pure classifier.
@@ -26,11 +26,11 @@
 - **Future extensions:** If the shell launcher is later fully ported, this mode decision becomes the dispatch point for native launch instead of compatibility exec.
 
 **AssetRoot** — Pure policy for choosing the root that owns adjacent runtime assets (`bin/pair-shell`, `nvim/`, `zellij/`).
-- **Relationships:** N:1 from local source build, local copied install, and Homebrew `libexec` install into one root decision; `LegacyPairRequest` consumes the resolved root.
+- **Relationships:** N:1 from local source build, local copied install, and Homebrew `libexec` install into one root decision; `LegacyLaunchRequest` consumes the resolved root.
 - **DRY rationale:** Prevents local install and Homebrew install from inventing separate path rules for the same asset layout (ARCH-DRY).
 - **Future extensions:** Can add an extracted-embedded asset dir later without changing launcher request construction.
 
-**LegacyPairRequest** — Describes the compatibility exec into the shell launcher.
+**LegacyLaunchRequest** — Describes the compatibility exec into the shell launcher.
 - **Relationships:** N:1 from `pair` direct mode and `pair-go launch` mode into one request builder; each mode only changes display/diagnostic wording and argv shape. Carries the selected `AssetRoot` and computes `<asset-root>/bin/pair-shell`.
 - **DRY rationale:** Keeps legacy shell handoff rules single-sourced while the actual zellij lifecycle remains shell-owned in this issue (ARCH-PURE).
 - **Future extensions:** Can be deleted once shell launch is replaced by native Go launch.
@@ -76,17 +76,17 @@
 - Create: `cmd/internal/entrypoint/mode.go`
 - Modify: `cmd/pair-go/main_test.go`
 
-- [ ] Add tests showing executable basename `pair` resolves to direct public launcher mode.
-- [ ] Add tests showing executable basename `pair-go` with `launch` still resolves to explicit launch handoff.
-- [ ] Add tests showing `pair-go` helper routes still dispatch without touching the shell launcher.
-- [ ] Add pure tests for asset-root resolution:
+- [x] Add tests showing executable basename `pair` resolves to direct public launcher mode.
+- [x] Add tests showing executable basename `pair-go` with `launch` still resolves to explicit launch handoff.
+- [x] Add tests showing `pair-go` helper routes still dispatch without touching the shell launcher.
+- [x] Add pure tests for asset-root resolution:
   - `PAIR_HOME=/repo` wins when `/repo/bin/pair-shell` exists.
   - executable `/repo/bin/pair` resolves sibling root `/repo` when `/repo/bin/pair-shell` exists.
   - copied executable `/home/me/.local/bin/pair` falls back to build-time default root `/repo` when sibling shell is absent and `/repo/bin/pair-shell` exists.
   - missing sibling and missing build-time root produces a diagnostic naming `pair-shell` and `PAIR_HOME`.
-- [ ] Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
-- [ ] Implement `EntrypointMode`, `AssetRoot`, and shared legacy request construction.
-- [ ] Re-run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
+- [x] Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
+- [x] Implement `EntrypointMode`, `AssetRoot`, and shared legacy request construction.
+- [x] Re-run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
 
 ### Task 2: Move shell launcher behind an internal compatibility name
 
@@ -97,13 +97,13 @@
 - Modify: `cmd/pair-go/main.go`
 - Modify: `bin/pair-dev`
 
-- [ ] Move the existing Bash launcher body to `bin/pair-shell`.
-- [ ] Update `.gitignore`: remove the `!bin/pair` tracked-script exception and add `!bin/pair-shell`; `bin/pair` stays ignored as generated Go build output.
-- [ ] Update Go direct `pair` mode to exec sibling `pair-shell` with argv[0] presented as `pair`.
-- [ ] Update `pair-go launch ...` to exec sibling `pair-shell` with the same argv compatibility as before.
-- [ ] Update `pair-dev` to export `PAIR_DEV=1` and exec sibling `pair` (the Go binary), not `pair-shell`, so dev mode exercises the public entrypoint.
-- [ ] Run: `bin/pair-go launch --help` after build and confirm it reaches the launcher help.
-- [ ] Run: `bin/pair --help` after build and confirm it reaches the same launcher help.
+- [x] Move the existing Bash launcher body to `bin/pair-shell`.
+- [x] Update `.gitignore`: remove the `!bin/pair` tracked-script exception and add `!bin/pair-shell`; `bin/pair` stays ignored as generated Go build output.
+- [x] Update Go direct `pair` mode to exec sibling `pair-shell` with argv[0] presented as `pair`.
+- [x] Update `pair-go launch ...` to exec sibling `pair-shell` with the same argv compatibility as before.
+- [x] Update `pair-dev` to export `PAIR_DEV=1` and exec sibling `pair` (the Go binary), not `pair-shell`, so dev mode exercises the public entrypoint.
+- [x] Run: `bin/pair-go launch --help` after build and confirm it reaches the launcher help.
+- [x] Run: `bin/pair --help` after build and confirm it reaches the same launcher help.
 
 ## Chunk 2: Build And Install Layout
 
@@ -113,13 +113,13 @@
 - Modify: `Makefile.local`
 - Modify: `tests/pair-go-install-layout-test.sh`
 
-- [ ] Update `GO_BINS` so `pair` is a Go-built binary and `pair-go` remains built from the same package.
-- [ ] Remove `pair` from `SHELL_BINS`; keep or explicitly drop `pair-dev` based on install behavior.
-- [ ] Add a specific `$(BIN_DIR)/pair` build rule using `go build -ldflags "-X main.defaultPairHome=$(CURDIR)" -o $@ ./cmd/pair-go`.
-- [ ] Keep `$(BIN_DIR)/pair-go` building from `./cmd/pair-go`; it may use the same `defaultPairHome` ldflag for copied local installs.
-- [ ] Update install-layout test: installed `pair` must be executable and not a symlink; installed `pair-go` remains executable; `pair-dev` may remain a symlink if still a dev wrapper.
-- [ ] Run: `make build`
-- [ ] Run: `make test-pair-go-install-layout`
+- [x] Update `GO_BINS` so `pair` is a Go-built binary and `pair-go` remains built from the same package.
+- [x] Remove `pair` from `SHELL_BINS`; keep or explicitly drop `pair-dev` based on install behavior.
+- [x] Add a specific `$(BIN_DIR)/pair` build rule using `go build -ldflags "-X main.defaultPairHome=$(CURDIR)" -o $@ ./cmd/pair-go`.
+- [x] Keep `$(BIN_DIR)/pair-go` building from `./cmd/pair-go`; it may use the same `defaultPairHome` ldflag for copied local installs.
+- [x] Update install-layout test: installed `pair` must be executable and not a symlink; installed `pair-go` remains executable; `pair-dev` may remain a symlink if still a dev wrapper.
+- [x] Run: `make build`
+- [x] Run: `make test-pair-go-install-layout`
 
 ### Task 4: Adjacent native asset install layout
 
@@ -128,12 +128,12 @@
 - Modify: `tests/pair-go-install-layout-test.sh`
 - Modify: `../homebrew-pair/Formula/pair.rb`
 
-- [ ] Keep `nvim/` and `zellij/` adjacent to `PAIR_HOME` for this issue; do not embed.
-- [ ] Local `make install` remains source-tree based for native assets: installed `pair` is copied to `~/.local/bin`, and when it has no sibling `pair-shell`, `AssetRoot` falls back to build-time `defaultPairHome=$(CURDIR)` to find the repo checkout assets.
-- [ ] Homebrew install remains `libexec`-adjacent: formula installs `bin/`, `nvim/`, and `zellij/` under `libexec`, then builds Go `pair`, `pair-go`, and required helper binaries into `libexec/bin` with `defaultPairHome=#{libexec}`.
-- [ ] Update formula comments and built-binary list so Homebrew surfaces `bin/pair` as the Go-built public command and retains `bin/pair-shell` only as an internal compatibility launcher.
-- [ ] Test that local installed `pair --help` reaches the shell help through the Go entrypoint.
-- [ ] Run: `make test-pair-go-install-layout`
+- [x] Keep `nvim/` and `zellij/` adjacent to `PAIR_HOME` for this issue; do not embed.
+- [x] Local `make install` remains source-tree based for native assets: installed `pair` is copied to `~/.local/bin`, and when it has no sibling `pair-shell`, `AssetRoot` falls back to build-time `defaultPairHome=$(CURDIR)` to find the repo checkout assets.
+- [x] Homebrew install remains `libexec`-adjacent: formula installs `bin/`, `nvim/`, and `zellij/` under `libexec`, then builds Go `pair`, `pair-go`, and required helper binaries into `libexec/bin` with `defaultPairHome=#{libexec}`.
+- [x] Update formula comments and built-binary list so Homebrew surfaces `bin/pair` as the Go-built public command and retains `bin/pair-shell` only as an internal compatibility launcher.
+- [x] Test that local installed `pair --help` reaches the shell help through the Go entrypoint.
+- [x] Run: `make test-pair-go-install-layout`
 
 ## Chunk 3: Compatibility Shim Inventory
 
@@ -146,11 +146,11 @@
 - Modify: `CHANGELOG.md` if the Homebrew/release note wording needs a packaging entry.
 - Modify: `../homebrew-pair/Formula/pair.rb`
 
-- [ ] Document `bin/pair-shell` as a retained compatibility launcher and explain why it is not obsolete yet.
-- [ ] Document `pair-dev` as retained dev-mode wrapper that runs the Go public `pair`.
-- [ ] Document legacy helper binaries retained because native zellij/nvim/shell callers still reference them.
-- [ ] Remove stale wording that says `pair-go launch` is the only Go-owned launch test surface; installed `pair` is now the public Go-owned entrypoint.
-- [ ] Update Homebrew wording: formula comments and README/CHANGELOG must say Homebrew installs a Go-built `pair` plus adjacent native assets under `libexec`.
+- [x] Document `bin/pair-shell` as a retained compatibility launcher and explain why it is not obsolete yet.
+- [x] Document `pair-dev` as retained dev-mode wrapper that runs the Go public `pair`.
+- [x] Document legacy helper binaries retained because native zellij/nvim/shell callers still reference them.
+- [x] Remove stale wording that says `pair-go launch` is the only Go-owned launch test surface; installed `pair` is now the public Go-owned entrypoint.
+- [x] Update Homebrew wording: formula comments and README/CHANGELOG must say Homebrew installs a Go-built `pair` plus adjacent native assets under `libexec`.
 
 ## Chunk 4: Verification And Closure
 
@@ -160,15 +160,29 @@
 - Modify: `workshop/issues/000079-go-packaging-consolidation.md`
 - Modify: `workshop/plans/000079-go-packaging-consolidation-plan.md`
 
-- [ ] Run focused Go tests: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`.
-- [ ] Run packaging tests: `make build && make test-pair-go-install-layout`.
-- [ ] Run an upgrade-layout test in `tests/pair-go-install-layout-test.sh`: seed `~/.local/bin/pair` as the old symlink-to-source-shell layout, run `make install`, then assert `~/.local/bin/pair` is now a regular executable Go binary, `bin/pair-shell` remains tracked/executable under the source root, `PAIR_HOME` override works, and default-root fallback lets installed `pair --help` reach the shell help.
-- [ ] Run a Homebrew formula dry-run/smoke if available: `brew test --formula ../homebrew-pair/Formula/pair.rb` or record the exact local blocker; at minimum run `ruby -c ../homebrew-pair/Formula/pair.rb`.
-- [ ] Run launcher smoke: `bin/pair --help`, `bin/pair-go launch --help`, `bin/pair-dev --help`.
-- [ ] Run broader impacted tests: `make test-dev-rebuild test-session-watch test-continue`.
-- [ ] If practical, run Linux smoke with the available local toolchain; otherwise record why it was not available.
-- [ ] Update issue checklist/log with verification evidence.
-- [ ] Update atlas/README and run stale-doc grep for old packaging statements.
+- [x] Run focused Go tests: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`.
+- [x] Run packaging tests: `make build && make test-pair-go-install-layout`.
+- [x] Run an upgrade-layout test in `tests/pair-go-install-layout-test.sh`: seed `~/.local/bin/pair` as the old symlink-to-source-shell layout, run `make install`, then assert `~/.local/bin/pair` is now a regular executable Go binary, `bin/pair-shell` remains tracked/executable under the source root, `PAIR_HOME` override works, and default-root fallback lets installed `pair --help` reach the shell help.
+- [x] Run a Homebrew formula dry-run/smoke if available: `brew test --formula ../homebrew-pair/Formula/pair.rb` or record the exact local blocker; at minimum run `ruby -c ../homebrew-pair/Formula/pair.rb`.
+- [x] Run launcher smoke: `bin/pair --help`, `bin/pair-go launch --help`, `bin/pair-dev --help`.
+- [x] Run broader impacted tests: `make test-dev-rebuild test-session-watch test-continue`.
+- [x] If practical, run Linux smoke with the available local toolchain; otherwise record why it was not available.
+- [x] Update issue checklist/log with verification evidence.
+- [x] Update atlas/README and run stale-doc grep for old packaging statements.
+
+Verification evidence captured 2026-06-30:
+`go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`; `make build`;
+`make test-pair-go-install-layout`; `bin/pair --help`; `bin/pair-go launch
+--help`; `bin/pair-dev --help`; `make test-dev-rebuild test-session-watch
+test-continue`; `go test ./... -count=1`; `ruby -c
+../homebrew-pair/Formula/pair.rb`; stale-doc grep for old #77 packaging
+wording. Homebrew tap evidence: sibling repo `../homebrew-pair` commit
+`3aeb2a6 pair: build Go public entrypoint` updates `Formula/pair.rb`.
+`brew test --formula ../homebrew-pair/Formula/pair.rb` is not
+supported by this Homebrew (`invalid option: --formula`), so formula validation
+is syntax-only unless tested through an installed tap. Linux smoke was not run
+because this local workspace is Darwin-only (`uname -s` => `Darwin`) and no
+Linux runner is configured.
 
 ## Implementation Notes
 
@@ -178,3 +192,11 @@
 - `ARCH-PURPOSE`: #79 is not complete if only docs change; installed `pair` must become the Go-owned public command.
 - `ARCH-DRY`: direct `pair` and `pair-go launch` must share one compatibility request builder.
 - `ARCH-PURE`: mode selection and request construction stay pure; filesystem/exec behavior stays in the `cmd/pair-go/main.go` runtime seam.
+
+## Revisions
+
+### 2026-06-30 — close-review REWORK traceability fixes
+
+- Corrected the Core Concepts name from planned `LegacyPairRequest` to implemented `LegacyLaunchRequest`.
+- Added exact Homebrew tap evidence: sibling repo `../homebrew-pair` commit `3aeb2a6 pair: build Go public entrypoint`.
+- Added standalone Make prerequisites for generated `bin/pair` on `test-cmux-ownership` and `test-continue`.

exec
/bin/zsh -lc "rg -n 'launcher\\.Run|internal/launcher|LaunchArgs|ParseArgs|Run\\(' cmd/internal cmd/pair-go" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
cmd/pair-go/helper_equivalence_test.go:52:	err := cmd.Run()
cmd/internal/scrollbackcmd/scrollbackcmd.go:391:func Run(argv []string, stdout, stderr io.Writer) int {
cmd/internal/scrollbackcmd/scrollbackcmd_test.go:14:	code := Run([]string{}, io.Discard, &stderr)
cmd/internal/scrollbackcmd/scrollbackcmd_test.go:37:	code := Run([]string{rawPath, evPath, outPath}, io.Discard, &stderr)
cmd/internal/contextcmd/contextcmd.go:29:func Run(args []string, env Env, stdout io.Writer) int {
cmd/internal/contextcmd/contextcmd_test.go:26:	code := Run([]string{"T", "claude"}, Env{Home: home, PairDataDir: data}, &stdout)
cmd/internal/contextcmd/contextcmd_test.go:38:	code := Run([]string{"T", "claude"}, Env{Home: home, PairDataDir: filepath.Join(home, "empty")}, &stdout)
cmd/internal/sessionwatch/run_test.go:27:	err := Run(Options{
cmd/internal/sessionwatch/run_test.go:63:	err := Run(Options{
cmd/internal/sessionwatch/run_test.go:92:	err := Run(Options{
cmd/internal/sessionwatch/run_test.go:123:	err := Run(Options{
cmd/internal/sessionwatch/run_test.go:158:	err := Run(Options{
cmd/internal/sessionwatch/run_test.go:196:	err := Run(Options{
cmd/internal/sessionwatch/run_test.go:217:	err := Run(Options{
cmd/internal/sessionwatch/sessionwatch_test.go:87:		t.Run(tt.name, func(t *testing.T) {
cmd/internal/model/model.go:72:func Run(r Request) (string, error) {
cmd/internal/sessionwatch/runtime.go:106:	return exec.Command("kill", "-0", pid).Run() == nil
cmd/internal/model/model_test.go:68:	got, err := Run(Request{
cmd/internal/model/model_test.go:113:	got, err := Run(Request{Agent: "codex", Model: "gpt-test-mini", Prompt: "prompt text", Input: "input text"})
cmd/internal/model/model_test.go:138:	_, err := Run(Request{Agent: "codex", Model: "gpt-test-mini", Prompt: "p", Input: "i"})
cmd/internal/sessionwatch/run.go:40:func Run(opts Options, rt Runtime) error {
cmd/internal/dispatcher/dispatcher.go:79:	code := contextcmd.Run(args, contextcmd.EnvFromOS(), &stdout)
cmd/internal/dispatcher/dispatcher.go:85:	code := scrollbackcmd.Run(args, &stdout, &stderr)
cmd/internal/dispatcher/dispatcher_test.go:12:		t.Run(strings.Join(args, "_"), func(t *testing.T) {
cmd/internal/launcher/decision_test.go:6:	decision, err := DecideLaunch(LaunchArgs{ForcedTag: "demo"}, SessionSnapshot{
cmd/internal/launcher/decision_test.go:18:	decision, err := DecideLaunch(LaunchArgs{ForcedTag: "demo"}, SessionSnapshot{
cmd/internal/launcher/decision_test.go:30:	decision, err := DecideLaunch(LaunchArgs{Agent: "codex"}, SessionSnapshot{
cmd/internal/launcher/decision_test.go:46:		decision, err := DecideLaunch(LaunchArgs{Agent: "claude"}, snap)
cmd/internal/launcher/decision_test.go:57:	decision, err := DecideLaunch(LaunchArgs{Agent: "claude", SelectedTag: "pair-old"}, SessionSnapshot{
cmd/internal/launcher/decision.go:22:func DecideLaunch(args LaunchArgs, snap SessionSnapshot) (LaunchDecision, error) {
cmd/internal/launcher/zellij.go:62:	if err := cmd.Run(); err != nil {
cmd/internal/launcher/datadir_test.go:15:		t.Run(tc.name, func(t *testing.T) {
cmd/internal/launcher/run.go:29:	Args     LaunchArgs
cmd/internal/launcher/run.go:36:func Run(argv []string, env Env, sessions SessionSource, history HistoricalScanner) (LaunchOutcome, error) {
cmd/internal/launcher/run.go:37:	args, err := ParseArgs(argv)
cmd/internal/launcher/args.go:5:// LaunchArgs is the pure parse result for the guarded pair-go launch prototype.
cmd/internal/launcher/args.go:6:type LaunchArgs struct {
cmd/internal/launcher/args.go:22:// ParseArgs parses pair-go launch args. It intentionally supports only the
cmd/internal/launcher/args.go:25:func ParseArgs(argv []string) (LaunchArgs, error) {
cmd/internal/launcher/args.go:26:	var out LaunchArgs
cmd/internal/launcher/args.go:34:		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %s is not implemented by pair-go launch; use bin/pair", argv[0])}
cmd/internal/launcher/args.go:37:			return LaunchArgs{}, UsageError{Message: "pair-go launch: 'resume' requires a tag"}
cmd/internal/launcher/args.go:41:			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: invalid tag: %v", err)}
cmd/internal/launcher/args.go:44:			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", argv[2])}
cmd/internal/launcher/args.go:64:		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", arg)}
cmd/internal/launcher/tag_test.go:15:		t.Run(tc.name, func(t *testing.T) {
cmd/internal/launcher/tag_test.go:29:		t.Run(raw, func(t *testing.T) {
cmd/internal/launcher/tag_test.go:47:		t.Run(tc.cwd, func(t *testing.T) {
cmd/internal/launcher/run_test.go:34:	outcome, err := Run([]string{"codex"}, Env{
cmd/internal/launcher/run_test.go:52:	outcome, err := Run([]string{"claude"}, Env{
cmd/internal/launcher/run_test.go:67:	outcome, err := Run([]string{"claude"}, Env{
cmd/internal/launcher/run_test.go:82:	_, err := Run([]string{"codex", "extra"}, Env{Home: "/home/me", Cwd: "/work/pair"}, fakeSessions{}, fakeHistory{})
cmd/internal/launcher/args_test.go:8:func TestParseLaunchArgsDefaultsToClaude(t *testing.T) {
cmd/internal/launcher/args_test.go:9:	args, err := ParseArgs(nil)
cmd/internal/launcher/args_test.go:11:		t.Fatalf("ParseArgs returned error: %v", err)
cmd/internal/launcher/args_test.go:24:func TestParseLaunchArgsAgentAndForwardedArgs(t *testing.T) {
cmd/internal/launcher/args_test.go:25:	args, err := ParseArgs([]string{"codex", "--", "-p", "say hi"})
cmd/internal/launcher/args_test.go:27:		t.Fatalf("ParseArgs returned error: %v", err)
cmd/internal/launcher/args_test.go:37:func TestParseLaunchArgsDefaultAgentWithForwardedArgs(t *testing.T) {
cmd/internal/launcher/args_test.go:38:	args, err := ParseArgs([]string{"--", "--dangerously-skip-permissions"})
cmd/internal/launcher/args_test.go:40:		t.Fatalf("ParseArgs returned error: %v", err)
cmd/internal/launcher/args_test.go:50:func TestParseLaunchArgsResumeNormalizesForcedTag(t *testing.T) {
cmd/internal/launcher/args_test.go:51:	args, err := ParseArgs([]string{"resume", "pair-demo"})
cmd/internal/launcher/args_test.go:53:		t.Fatalf("ParseArgs returned error: %v", err)
cmd/internal/launcher/args_test.go:63:func TestParseLaunchArgsUnexpectedPositionalGuidesAgentArgs(t *testing.T) {
cmd/internal/launcher/args_test.go:64:	_, err := ParseArgs([]string{"codex", "extra"})
cmd/internal/launcher/args_test.go:66:		t.Fatal("ParseArgs returned nil error")
cmd/internal/launcher/args_test.go:76:func TestParseLaunchArgsUnsupportedLaunchSubcommandsAreExplicit(t *testing.T) {
cmd/internal/launcher/args_test.go:78:		t.Run(verb, func(t *testing.T) {
cmd/internal/launcher/args_test.go:79:			_, err := ParseArgs([]string{verb})
cmd/internal/launcher/args_test.go:81:				t.Fatal("ParseArgs returned nil error")

codex
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The main packaging change is structurally sound: `pair` and `pair-go` share the Go entrypoint, the shell lifecycle is retained behind `bin/pair-shell`, asset-root resolution is pure and tested, and the local install/upgrade path is covered. I found no Critical blockers. The one Important issue is stale atlas guidance that still points future launcher-recovery edits at generated `bin/pair` instead of the retained shell implementation.

1. Strengths

- `cmd/internal/entrypoint/asset_root.go:22` keeps asset selection pure and injects filesystem probing, matching ARCH-PURE.
- `cmd/pair-go/main.go:46` funnels direct `pair` and `pair-go launch` through one mode switch and one legacy launch path, matching ARCH-DRY.
- `tests/pair-go-install-layout-test.sh:17` exercises upgrade from old symlinked `pair` to copied Go binary and verifies both direct `pair` and `pair-go launch`.
- `atlas/architecture.md:49` and `atlas/go-migration-inventory.md:50` document the new public Go entrypoint plus retained shell launcher.

2. Critical findings

None.

3. Important findings

- `atlas/how-to-bring-up-a-new-harness-cli.md:66`, `atlas/how-to-bring-up-a-new-harness-cli.md:67`, `atlas/how-to-bring-up-a-new-harness-cli.md:139`: atlas still tells future agent-port work to edit `bin/pair`, but #79 makes `bin/pair` generated/ignored and moves the shell lifecycle to `bin/pair-shell`. This is an atlas update gap and can send future work to the wrong file. Fix by pointing launcher recovery/configuration guidance at `bin/pair-shell` while reserving `pair`/`bin/pair` for the public command. Cite ARCH-PURPOSE because the new source-of-truth layout is only useful if consumers stop restating the old one.

4. Minor findings

- `cmd/internal/launcher/args.go:34`: stale user-facing text says `use bin/pair`; prefer `use pair` or update to the current public entrypoint wording.

5. Test coverage notes

I ran `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`; it passed. `git diff --check` also passed. The plan’s Homebrew formula work is recorded in the tracker as a sibling repo commit, but this review scope was the pair repo only, so I did not independently validate that external repo.

6. Architectural notes for upcoming work

- ARCH-DRY: Pass. Direct `pair` and `pair-go launch` share `ResolveLegacyLaunch`.
- ARCH-PURE: Pass. Mode/request/root selection are pure; IO stays in `cmd/pair-go/main.go`.
- ARCH-PURPOSE: Mostly pass. Installed `pair` is now Go-owned, but stale atlas consumer guidance should be corrected before closing.

7. Plan revision recommendations

None required for the plan itself; it already names `LegacyLaunchRequest` and records the previous traceability fixes.
tokens used
121,028
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The main packaging change is structurally sound: `pair` and `pair-go` share the Go entrypoint, the shell lifecycle is retained behind `bin/pair-shell`, asset-root resolution is pure and tested, and the local install/upgrade path is covered. I found no Critical blockers. The one Important issue is stale atlas guidance that still points future launcher-recovery edits at generated `bin/pair` instead of the retained shell implementation.

1. Strengths

- `cmd/internal/entrypoint/asset_root.go:22` keeps asset selection pure and injects filesystem probing, matching ARCH-PURE.
- `cmd/pair-go/main.go:46` funnels direct `pair` and `pair-go launch` through one mode switch and one legacy launch path, matching ARCH-DRY.
- `tests/pair-go-install-layout-test.sh:17` exercises upgrade from old symlinked `pair` to copied Go binary and verifies both direct `pair` and `pair-go launch`.
- `atlas/architecture.md:49` and `atlas/go-migration-inventory.md:50` document the new public Go entrypoint plus retained shell launcher.

2. Critical findings

None.

3. Important findings

- `atlas/how-to-bring-up-a-new-harness-cli.md:66`, `atlas/how-to-bring-up-a-new-harness-cli.md:67`, `atlas/how-to-bring-up-a-new-harness-cli.md:139`: atlas still tells future agent-port work to edit `bin/pair`, but #79 makes `bin/pair` generated/ignored and moves the shell lifecycle to `bin/pair-shell`. This is an atlas update gap and can send future work to the wrong file. Fix by pointing launcher recovery/configuration guidance at `bin/pair-shell` while reserving `pair`/`bin/pair` for the public command. Cite ARCH-PURPOSE because the new source-of-truth layout is only useful if consumers stop restating the old one.

4. Minor findings

- `cmd/internal/launcher/args.go:34`: stale user-facing text says `use bin/pair`; prefer `use pair` or update to the current public entrypoint wording.

5. Test coverage notes

I ran `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`; it passed. `git diff --check` also passed. The plan’s Homebrew formula work is recorded in the tracker as a sibling repo commit, but this review scope was the pair repo only, so I did not independently validate that external repo.

6. Architectural notes for upcoming work

- ARCH-DRY: Pass. Direct `pair` and `pair-go launch` share `ResolveLegacyLaunch`.
- ARCH-PURE: Pass. Mode/request/root selection are pure; IO stays in `cmd/pair-go/main.go`.
- ARCH-PURPOSE: Mostly pass. Installed `pair` is now Go-owned, but stale atlas consumer guidance should be corrected before closing.

7. Plan revision recommendations

None required for the plan itself; it already names `LegacyLaunchRequest` and records the previous traceability fixes.
