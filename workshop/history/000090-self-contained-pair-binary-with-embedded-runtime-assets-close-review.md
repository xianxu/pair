# Boundary Review — pair#90 (whole-issue close)

| field | value |
|-------|-------|
| issue | 90 — self-contained pair binary with embedded runtime assets |
| repo | pair |
| issue file | workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md |
| boundary | whole-issue close |
| milestone | — |
| window | e1ae416c6084399c155e0a89c5597712f679460b..HEAD |
| command | sdlc close --issue 90 |
| reviewer | codex |
| timestamp | 2026-07-01T00:50:43-07:00 |
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
session id: 019f1ca6-5cfe-7c21-91ac-a1a1b86fbcbb
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **pair** repository.

- repository: pair   (root: /Users/xianxu/workspace/pair)
- issue:      pair#90   (file: workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md)
- window:     Base: e1ae416c6084399c155e0a89c5597712f679460b   Head: HEAD

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
index 18ff43e..a8ea612 100644
--- a/.gitignore
+++ b/.gitignore
@@ -37,3 +37,4 @@ cache/
 /GEMINI.md
 /.agents/skills/
 /construct/generated/
+/cmd/internal/runtimebundle/assets/
diff --git a/Makefile.local b/Makefile.local
index d2383b8..0ad1df0 100644
--- a/Makefile.local
+++ b/Makefile.local
@@ -1,7 +1,7 @@
 # Repo-specific Makefile targets.
 # This file is included by Makefile — add your own targets here.
 
-.PHONY: build install pair pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-session-watch pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
+.PHONY: build install pair pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-session-watch pair-go runtimebundle-generate runtimebundle-drift-check test-runtimebundle test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-pair-embedded-runtime test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
 
 # Cross-cutting Go-binary targets. Every Go binary lives under cmd/<name>/
 # and ships with the `pair-` prefix so it doesn't collide with anything
@@ -29,6 +29,8 @@ INSTALL_DIR := $(HOME)/.local/bin
 GO_BINS     := pair pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-session-watch pair-go
 SHELL_BINS  := pair-dev
 PAIR_HOME_LDFLAGS := -X main.defaultPairHome=$(CURDIR)
+RUNTIMEBUNDLE_HELPERS := pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-session-watch
+RUNTIMEBUNDLE_ASSETS := cmd/internal/runtimebundle/assets/runtime/manifest.json
 
 build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
 
@@ -65,7 +67,7 @@ pair-go:                $(BIN_DIR)/pair-go
 #                     inside the vt.Emulator library API itself, which is
 #                     outside our scope to fix — exclude that package
 #                     here rather than chase a library-level race.
-test: test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace test-review
+test: test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-pair-embedded-runtime test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace test-review
 	go test ./... -count=1
 
 # pair-title poller: single-instance guard (identity-checked liveness so a
@@ -86,6 +88,21 @@ test-cmux-ownership: $(BIN_DIR)/pair
 test-session-watch: $(BIN_DIR)/pair-session-watch
 	bash tests/pair-session-watch-test.sh
 
+runtimebundle-generate: $(addprefix $(BIN_DIR)/,$(RUNTIMEBUNDLE_HELPERS))
+	go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime
+
+$(RUNTIMEBUNDLE_ASSETS): runtimebundle-generate
+
+runtimebundle-drift-check: $(addprefix $(BIN_DIR)/,$(RUNTIMEBUNDLE_HELPERS))
+	@d1="$$(mktemp -d)"; d2="$$(mktemp -d)"; \
+	trap 'rm -rf "$$d1" "$$d2"' EXIT; \
+	go run ./cmd/internal/runtimebundle/generatecmd --repo . --out "$$d1/runtime" && \
+	go run ./cmd/internal/runtimebundle/generatecmd --repo . --out "$$d2/runtime" && \
+	diff -qr "$$d1/runtime" "$$d2/runtime" >/dev/null
+
+test-runtimebundle: runtimebundle-generate
+	go test ./cmd/internal/runtimebundle -count=1
+
 # Review workbench (#66): the integration shell tests. The pure modules
 # (record, reconstruct) run under `make test-lua`.
 test-review:
@@ -195,6 +212,9 @@ test-dev-rebuild:
 test-pair-go-install-layout:
 	bash tests/pair-go-install-layout-test.sh
 
+test-pair-embedded-runtime:
+	bash tests/pair-embedded-runtime-test.sh
+
 # pair-doctor emitter-health probe (#000047): marker check + binary selection
 # (running-via-pidfile vs PATH). See tests/emitter-health-test.sh.
 test-emitter-health:
@@ -258,11 +278,12 @@ $(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill
 $(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-continuation/git.go cmd/pair-continuation/main.go go.mod
 	go build -o $@ ./cmd/pair-continuation
 
-PAIR_GO_SRCS := cmd/pair-go/main.go cmd/internal/entrypoint/asset_root.go cmd/internal/entrypoint/launch.go cmd/internal/entrypoint/mode.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
+PAIR_GO_SRCS := cmd/pair-go/main.go cmd/internal/entrypoint/asset_root.go cmd/internal/entrypoint/launch.go cmd/internal/entrypoint/mode.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go cmd/internal/runtimebundle/manifest.go cmd/internal/runtimebundle/plan.go cmd/internal/runtimebundle/cleanup.go cmd/internal/runtimebundle/embed.go $(RUNTIMEBUNDLE_ASSETS) go.mod
 
 # pair — public Go-owned entrypoint. It execs bin/pair-shell for the current
-# shell-owned zellij lifecycle and uses the build-time defaultPairHome when a
-# copied local install has no sibling assets.
+# shell-owned zellij lifecycle; source/Homebrew builds can inject
+# defaultPairHome, while copied binaries without adjacent assets use the
+# embedded runtime fallback.
 $(BIN_DIR)/pair: $(PAIR_GO_SRCS)
 	go build -ldflags "$(PAIR_HOME_LDFLAGS)" -o $@ ./cmd/pair-go
 
diff --git a/README.md b/README.md
index eb3af77..1081b19 100644
--- a/README.md
+++ b/README.md
@@ -201,12 +201,19 @@ pair -h, --help                  # show full help
 ```
 
 The installed `pair` command is Go-owned. For this migration window it resolves
-the adjacent Pair asset root and hands off to `bin/pair-shell`, the retained
-shell launcher that still owns the zellij lifecycle. `pair-go launch ...` remains
-the explicit development dispatcher path and accepts the same arguments after
-`launch` that `pair` accepts directly. In a dev shell sourced from
-`../ariadne/construct/dev-aliases.sh`, `pair` and `pair-go` rebuild from
-`cmd/pair-go` automatically before running; no `pair-go-dev` command is needed.
+the Pair asset root and hands off to `bin/pair-shell`, the retained shell
+launcher that still owns the zellij lifecycle. Source and Homebrew installs use
+their adjacent asset roots. A copied standalone `pair` binary with no adjacent
+or build-time source root extracts its embedded Pair-owned runtime assets to
+`${XDG_DATA_HOME:-$HOME/.local/share}/pair/runtime/<digest>/pair-home` and runs
+with `PAIR_HOME` pointed there. External tools such as `zellij`, `nvim`, `fzf`,
+`jq`, clipboard tools, and agent CLIs are still installed separately.
+
+`pair-go launch ...` remains the explicit development dispatcher path and
+accepts the same arguments after `launch` that `pair` accepts directly. In a dev
+shell sourced from `../ariadne/construct/dev-aliases.sh`, `pair` and `pair-go`
+rebuild from `cmd/pair-go` automatically before running; no `pair-go-dev`
+command is needed.
 
 Use `--` to separate pair's positional from agent flags. Without it, pair only takes `<agent>` as a positional and everything else is rejected.
 
diff --git a/atlas/architecture.md b/atlas/architecture.md
index ca01dee..6718202 100644
--- a/atlas/architecture.md
+++ b/atlas/architecture.md
@@ -46,13 +46,21 @@ internal subcommands or dispatch modes behind that primary binary (`pair wrap`,
 `pair continuation`, `pair scribe`) instead of staying as independently managed
 installed commands forever.
 
-As of #79, the public `bin/pair` command is a Go-built entrypoint from
-`cmd/pair-go`. Direct `pair ...` and explicit `pair-go launch ...` share one
-compatibility handoff: resolve the Pair asset root, then exec
-`<asset-root>/bin/pair-shell` with `pair`-compatible argv/env. Asset root
-resolution is ordered: explicit `PAIR_HOME`, executable sibling root, then the
-build-time `defaultPairHome` injected by Make/Homebrew for copied installs.
-Native `nvim/` and `zellij/` assets remain adjacent to that root.
+As of #90, the public `bin/pair` command is a Go-built entrypoint from
+`cmd/pair-go` with an embedded Pair-owned runtime bundle. Direct `pair ...` and
+explicit `pair-go launch ...` share one compatibility handoff: resolve the Pair
+asset root, then exec `<asset-root>/bin/pair-shell` with `pair`-compatible
+argv/env. Asset root resolution is ordered: explicit `PAIR_HOME`, executable
+sibling root, the build-time `defaultPairHome` injected by Make/Homebrew, then
+an extracted embedded runtime under `$PAIR_DATA_DIR/runtime/<digest>/pair-home`
+when no adjacent/source asset root exists. Native `nvim/` and `zellij/` assets
+remain native files inside whichever asset root was selected.
+
+The embedded runtime is generated from a deterministic manifest before builds
+and tests. That manifest is the packaging source of truth for bundled Pair-owned
+shell helpers, helper binaries, `bin/lib/`, `nvim/`, `zellij/`, and doctor
+assets; external programs such as `zellij`, `nvim`, `fzf`, `jq`, clipboard
+tools, and agent CLIs remain system dependencies.
 
 `pair-go` remains the development dispatcher for helper routes and explicit
 launch testing: `pair-go launch claude`, `pair-go launch resume <tag>`,
diff --git a/atlas/go-migration-inventory.md b/atlas/go-migration-inventory.md
index ac87eee..37c5f91 100644
--- a/atlas/go-migration-inventory.md
+++ b/atlas/go-migration-inventory.md
@@ -21,14 +21,15 @@ Out of scope:
 
 - porting behavior;
 - changing public command behavior;
-- deciding final embedded-vs-adjacent asset packaging. #79 owns that decision.
+- removing adjacent asset packaging while Homebrew/source layouts still rely on
+  it.
 
 ## Single-Binary Deployment Path
 
-#79 made the installed public `pair` command Go-owned, but the supported runtime
-layout is still an adjacent asset tree. The next deployment goal is #90:
-produce a self-contained `pair` binary that embeds the Pair-owned runtime assets
-and extracts them to a versioned runtime root on demand.
+#79 made the installed public `pair` command Go-owned. #90 added a
+self-contained deployment mode: the Go binary embeds the Pair-owned runtime
+assets and extracts them to `$PAIR_DATA_DIR/runtime/<digest>/pair-home` on
+demand when no adjacent/source/Homebrew asset root is available.
 
 This is not the same as "no external dependencies." The single-binary target is
 one Pair artifact. System programs such as `zellij`, `nvim`, clipboard tools,
@@ -37,11 +38,13 @@ a later issue explicitly replaces them.
 
 Execution path:
 
-1. **Embedded runtime bundle (#90):** embed the current Pair-owned runtime tree
+1. **Embedded runtime bundle (#90):** the current Pair-owned runtime tree
    (`bin/pair-shell`, shell helpers, helper binaries or dispatcher shims,
-   `nvim/`, `zellij/`, docs/help needed at runtime) into the Go binary. On run,
-   extract to a versioned cache/data root and set `PAIR_HOME` there before the
-   existing launch handoff.
+   `bin/lib/`, `nvim/`, `zellij/`, and doctor/help assets needed at runtime) is
+   generated into a manifest-backed embedded bundle. On run, copied binaries
+   extract to a digest-named Pair data root, write a runtime marker, prune stale
+   older extracted runtimes without deleting the selected digest, and set
+   `PAIR_HOME` there before the existing launch handoff.
 2. **Dispatcher consolidation:** move helper binaries behind `pair <subcommand>`
    routes and leave old command names as generated compatibility shims only
    where native callers still need them.
@@ -84,7 +87,7 @@ Priority is packaging impact first, then reliability/testability:
 
 | Artifact | Type | Callers | Runtime contract | Files/env | Disposition | Priority |
 |---|---|---|---|---|---|---|
-| `bin/pair` / `bin/pair-shell` / `cmd/internal/launcher` / `cmd/internal/entrypoint` | Go public entrypoint plus retained shell launcher | user shell, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` is generated from `cmd/pair-go` and resolves `PAIR_HOME` / sibling root / build-time `defaultPairHome`, then execs `<asset-root>/bin/pair-shell` with `pair`-compatible argv/env. `bin/pair-shell` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` shares the same compatibility handoff. | `bin/pair-shell` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves invocation mode, asset root, and compatibility request; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Public entrypoint is Go-owned as of #79; `bin/pair-shell` is retained because real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
+| `bin/pair` / `bin/pair-shell` / `cmd/internal/launcher` / `cmd/internal/entrypoint` / `cmd/internal/runtimebundle` | Go public entrypoint plus retained shell launcher and embedded runtime fallback | user shell, copied-binary installs, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` is generated from `cmd/pair-go` and resolves `PAIR_HOME` / sibling root / build-time `defaultPairHome`; if none exists, it extracts the embedded runtime to `$PAIR_DATA_DIR/runtime/<digest>/pair-home`; then it execs `<asset-root>/bin/pair-shell` with `pair`-compatible argv/env and `PAIR_HOME` pointed at the selected root. `bin/pair-shell` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` shares the same compatibility handoff. | `bin/pair-shell` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves invocation mode, asset root, and compatibility request; `cmd/internal/runtimebundle` owns manifest hashing, extraction planning, runtime markers, and stale-runtime cleanup; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Public entrypoint is Go-owned as of #79; copied-binary embedded fallback exists as of #90; `bin/pair-shell` is retained because real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
 | `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling Go-built `pair`. | Resolves symlinks; depends on generated `bin/pair`, retained `bin/pair-shell`, and `bin/lib/dev-rebuild.sh`. | retained dev wrapper so developer launches exercise the public Go entrypoint | P1 |
 | `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair-shell` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
 | `zellij/layouts/main.kdl` | zellij native asset | `bin/pair-shell` via `zellij --new-session-with-layout` | Defines agent and draft panes; shell expands Pair env at pane start. | Calls `pair-wrap`; calls `nvim -u "$PAIR_HOME/nvim/init.lua"`; writes `pane-<tag>-<agent>.json`; writes draft nvim pid file. | native-asset, packaged adjacent/embedded | P0 |
@@ -151,14 +154,22 @@ Nvim shell-outs and binary dependencies:
 
 Build/install callers:
 
+- `make runtimebundle-generate` refreshes the gitignored embedded runtime asset
+  tree and manifest; `make runtimebundle-drift-check` verifies the generated
+  bundle is reproducible.
 - `make build` builds `GO_BINS` into `bin/`; `pair` and `pair-go` are both built
-  from `cmd/pair-go` with `defaultPairHome=$(CURDIR)`.
+  from `cmd/pair-go` with `defaultPairHome=$(CURDIR)`, while copied builds with
+  no adjacent/default root use the embedded fallback.
 - `make install` copies `GO_BINS` to `~/.local/bin` and symlinks only retained
   shell wrappers such as `pair-dev`. Installed `pair` is a regular Go binary;
-  if it has no sibling assets, it falls back to the build-time source root.
+  if it has no sibling assets, it falls back to the build-time source root when
+  that exists and otherwise extracts the embedded runtime.
 - Homebrew installs `bin/`, `nvim/`, and `zellij/` under `libexec`, then builds
   Go `pair`, `pair-go`, and required runtime helpers into `libexec/bin` with
   `defaultPairHome=#{libexec}`.
+- `make test-runtimebundle` runs bundle-generation-aware Go tests, and
+  `make test-pair-embedded-runtime` exercises copied-binary launch plus stale
+  runtime cleanup with fake external dependencies.
 - `pair-dev` relies on `make build`, then zellij's PATH lookup resolves fresh
   repo `bin/` binaries.
 
diff --git a/cmd/internal/entrypoint/asset_root.go b/cmd/internal/entrypoint/asset_root.go
index ebdf6df..aa023ff 100644
--- a/cmd/internal/entrypoint/asset_root.go
+++ b/cmd/internal/entrypoint/asset_root.go
@@ -10,6 +10,7 @@ type AssetRootInput struct {
 	PairHome        string
 	Executable      string
 	DefaultPairHome string
+	EmbeddedRoot    string
 	PairShellExists func(root string) bool
 }
 
@@ -38,6 +39,9 @@ func ResolveAssetRoot(input AssetRootInput) (AssetRoot, error) {
 	if input.DefaultPairHome != "" {
 		candidates = append(candidates, assetRootCandidate{root: input.DefaultPairHome, source: "defaultPairHome"})
 	}
+	if input.EmbeddedRoot != "" {
+		candidates = append(candidates, assetRootCandidate{root: input.EmbeddedRoot, source: "embedded runtime"})
+	}
 
 	seen := map[string]bool{}
 	checked := make([]string, 0, len(candidates))
diff --git a/cmd/internal/entrypoint/asset_root_test.go b/cmd/internal/entrypoint/asset_root_test.go
index 6c6843d..5124ffc 100644
--- a/cmd/internal/entrypoint/asset_root_test.go
+++ b/cmd/internal/entrypoint/asset_root_test.go
@@ -57,6 +57,24 @@ func TestResolveAssetRootFallsBackToDefaultPairHome(t *testing.T) {
 	}
 }
 
+func TestResolveAssetRootFallsBackToEmbeddedRootAfterAdjacentRoots(t *testing.T) {
+	root, err := ResolveAssetRoot(AssetRootInput{
+		Executable:      "/home/me/.local/bin/pair",
+		DefaultPairHome: "/default/root",
+		EmbeddedRoot:    "/data/pair/runtime/abc/pair-home",
+		PairShellExists: existingRoots("/data/pair/runtime/abc/pair-home"),
+	})
+	if err != nil {
+		t.Fatalf("ResolveAssetRoot error = %v", err)
+	}
+	if root.Root != "/data/pair/runtime/abc/pair-home" {
+		t.Fatalf("Root = %q, want embedded root", root.Root)
+	}
+	if root.Source != "embedded runtime" {
+		t.Fatalf("Source = %q, want embedded runtime", root.Source)
+	}
+}
+
 func TestResolveAssetRootReportsMissingPairShellAndPairHome(t *testing.T) {
 	_, err := ResolveAssetRoot(AssetRootInput{
 		Executable:      "/home/me/.local/bin/pair",
diff --git a/cmd/internal/runtimebundle/cleanup.go b/cmd/internal/runtimebundle/cleanup.go
new file mode 100644
index 0000000..02f99b0
--- /dev/null
+++ b/cmd/internal/runtimebundle/cleanup.go
@@ -0,0 +1,68 @@
+package runtimebundle
+
+import (
+	"fmt"
+	"sort"
+)
+
+type RuntimeGeneration struct {
+	Digest    string
+	HasMarker bool
+	ModUnix   int64
+}
+
+type CleanupInput struct {
+	SelectedDigest string
+	Keep           int
+	Generations    []RuntimeGeneration
+}
+
+type CleanupPlan struct {
+	DeleteDigests []string
+}
+
+func PlanCleanup(input CleanupInput) (CleanupPlan, error) {
+	if input.SelectedDigest == "" {
+		return CleanupPlan{}, fmt.Errorf("selected digest is required")
+	}
+	if input.Keep < 0 {
+		return CleanupPlan{}, fmt.Errorf("keep must be non-negative")
+	}
+	candidates := make([]RuntimeGeneration, 0, len(input.Generations))
+	for _, gen := range input.Generations {
+		if gen.Digest == input.SelectedDigest || !gen.HasMarker || !isDigestName(gen.Digest) {
+			continue
+		}
+		candidates = append(candidates, gen)
+	}
+	sort.Slice(candidates, func(i, j int) bool {
+		if candidates[i].ModUnix == candidates[j].ModUnix {
+			return candidates[i].Digest > candidates[j].Digest
+		}
+		return candidates[i].ModUnix > candidates[j].ModUnix
+	})
+	if input.Keep >= len(candidates) {
+		return CleanupPlan{}, nil
+	}
+	deleteCandidates := candidates[input.Keep:]
+	sort.Slice(deleteCandidates, func(i, j int) bool {
+		return deleteCandidates[i].Digest < deleteCandidates[j].Digest
+	})
+	plan := CleanupPlan{DeleteDigests: make([]string, 0, len(deleteCandidates))}
+	for _, gen := range deleteCandidates {
+		plan.DeleteDigests = append(plan.DeleteDigests, gen.Digest)
+	}
+	return plan, nil
+}
+
+func isDigestName(s string) bool {
+	if s == "" {
+		return false
+	}
+	for _, r := range s {
+		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
+			return false
+		}
+	}
+	return true
+}
diff --git a/cmd/internal/runtimebundle/cleanup_test.go b/cmd/internal/runtimebundle/cleanup_test.go
new file mode 100644
index 0000000..d7f8375
--- /dev/null
+++ b/cmd/internal/runtimebundle/cleanup_test.go
@@ -0,0 +1,40 @@
+package runtimebundle
+
+import "testing"
+
+func TestPlanCleanupKeepsSelectedRuntime(t *testing.T) {
+	plan, err := PlanCleanup(CleanupInput{
+		SelectedDigest: "bbbb",
+		Keep:           1,
+		Generations: []RuntimeGeneration{
+			{Digest: "aaaa", HasMarker: true, ModUnix: 10},
+			{Digest: "bbbb", HasMarker: true, ModUnix: 1},
+			{Digest: "cccc", HasMarker: true, ModUnix: 20},
+		},
+	})
+	if err != nil {
+		t.Fatalf("PlanCleanup error = %v", err)
+	}
+	if len(plan.DeleteDigests) != 1 || plan.DeleteDigests[0] != "aaaa" {
+		t.Fatalf("DeleteDigests = %#v, want only aaaa", plan.DeleteDigests)
+	}
+}
+
+func TestPlanCleanupIgnoresNonRuntimeDirectories(t *testing.T) {
+	plan, err := PlanCleanup(CleanupInput{
+		SelectedDigest: "bbbb",
+		Keep:           0,
+		Generations: []RuntimeGeneration{
+			{Digest: "not-a-digest", HasMarker: true, ModUnix: 10},
+			{Digest: "aaaa", HasMarker: false, ModUnix: 20},
+			{Digest: "bbbb", HasMarker: true, ModUnix: 1},
+			{Digest: "cccc", HasMarker: true, ModUnix: 30},
+		},
+	})
+	if err != nil {
+		t.Fatalf("PlanCleanup error = %v", err)
+	}
+	if len(plan.DeleteDigests) != 1 || plan.DeleteDigests[0] != "cccc" {
+		t.Fatalf("DeleteDigests = %#v, want only cccc", plan.DeleteDigests)
+	}
+}
diff --git a/cmd/internal/runtimebundle/embed.go b/cmd/internal/runtimebundle/embed.go
new file mode 100644
index 0000000..c08cf3c
--- /dev/null
+++ b/cmd/internal/runtimebundle/embed.go
@@ -0,0 +1,25 @@
+package runtimebundle
+
+import (
+	"embed"
+	"encoding/json"
+)
+
+//go:embed assets/runtime/manifest.json assets/runtime/files
+var embedded embed.FS
+
+func EmbeddedManifest() RuntimeManifest {
+	data, err := embedded.ReadFile("assets/runtime/manifest.json")
+	if err != nil {
+		panic(err)
+	}
+	var manifest RuntimeManifest
+	if err := json.Unmarshal(data, &manifest); err != nil {
+		panic(err)
+	}
+	return manifest
+}
+
+func EmbeddedAsset(path string) ([]byte, error) {
+	return embedded.ReadFile("assets/runtime/files/" + path)
+}
diff --git a/cmd/internal/runtimebundle/embed_test.go b/cmd/internal/runtimebundle/embed_test.go
new file mode 100644
index 0000000..c0494ed
--- /dev/null
+++ b/cmd/internal/runtimebundle/embed_test.go
@@ -0,0 +1,45 @@
+package runtimebundle
+
+import "testing"
+
+func TestEmbeddedManifestContainsLaunchAssets(t *testing.T) {
+	manifest := EmbeddedManifest()
+	paths := map[string]bool{}
+	for _, asset := range manifest.Assets {
+		paths[asset.Path] = true
+	}
+	for _, want := range []string{
+		"bin/pair-shell",
+		"bin/pair-help",
+		"bin/pair-title.sh",
+		"bin/pair-session-watch.sh",
+		"bin/lib/dev-rebuild.sh",
+		"bin/pair-wrap",
+		"bin/pair-slug",
+		"bin/pair-context",
+		"bin/pair-scrollback-render",
+		"bin/pair-changelog",
+		"bin/pair-continuation",
+		"bin/pair-session-watch",
+		"nvim/init.lua",
+		"nvim/review/init.lua",
+		"zellij/config.kdl",
+		"zellij/layouts/main.kdl",
+		"doctor/SKILL.md",
+		"doctor/doctor.sh",
+	} {
+		if !paths[want] {
+			t.Fatalf("EmbeddedManifest missing %q", want)
+		}
+	}
+	for _, excluded := range []string{
+		"bin/pair",
+		"bin/pair-go",
+		"bin/pair-dev",
+		"nvim/init_test.lua",
+	} {
+		if paths[excluded] {
+			t.Fatalf("EmbeddedManifest includes excluded path %q", excluded)
+		}
+	}
+}
diff --git a/cmd/internal/runtimebundle/generatecmd/main.go b/cmd/internal/runtimebundle/generatecmd/main.go
new file mode 100644
index 0000000..ceddad8
--- /dev/null
+++ b/cmd/internal/runtimebundle/generatecmd/main.go
@@ -0,0 +1,19 @@
+package main
+
+import (
+	"flag"
+	"fmt"
+	"os"
+
+	"github.com/xianxu/pair/cmd/internal/runtimebundlegen"
+)
+
+func main() {
+	repo := flag.String("repo", ".", "repository root")
+	out := flag.String("out", "", "output root")
+	flag.Parse()
+	if _, err := runtimebundlegen.Generate(runtimebundlegen.GenerateOptions{RepoRoot: *repo, OutRoot: *out}); err != nil {
+		fmt.Fprintf(os.Stderr, "runtimebundle-generate: %v\n", err)
+		os.Exit(1)
+	}
+}
diff --git a/cmd/internal/runtimebundle/manifest.go b/cmd/internal/runtimebundle/manifest.go
new file mode 100644
index 0000000..434b61d
--- /dev/null
+++ b/cmd/internal/runtimebundle/manifest.go
@@ -0,0 +1,73 @@
+package runtimebundle
+
+import (
+	"crypto/sha256"
+	"encoding/hex"
+	"errors"
+	"fmt"
+	"path"
+	"sort"
+	"strings"
+)
+
+type RuntimeAsset struct {
+	Path   string
+	Mode   uint32
+	Size   int64
+	Digest string
+}
+
+type RuntimeManifest struct {
+	Assets []RuntimeAsset
+}
+
+func (m RuntimeManifest) Validate() error {
+	seen := map[string]bool{}
+	for _, asset := range m.Assets {
+		if err := validateAsset(asset); err != nil {
+			return err
+		}
+		if seen[asset.Path] {
+			return fmt.Errorf("duplicate asset path %q", asset.Path)
+		}
+		seen[asset.Path] = true
+	}
+	return nil
+}
+
+func (m RuntimeManifest) ManifestDigest() (string, error) {
+	if err := m.Validate(); err != nil {
+		return "", err
+	}
+	assets := append([]RuntimeAsset(nil), m.Assets...)
+	sort.Slice(assets, func(i, j int) bool {
+		return assets[i].Path < assets[j].Path
+	})
+	h := sha256.New()
+	for _, asset := range assets {
+		_, _ = fmt.Fprintf(h, "%s\x00%o\x00%d\x00%s\x00", asset.Path, asset.Mode, asset.Size, asset.Digest)
+	}
+	return hex.EncodeToString(h.Sum(nil)), nil
+}
+
+func validateAsset(asset RuntimeAsset) error {
+	if asset.Path == "" {
+		return errors.New("asset path is empty")
+	}
+	if strings.HasPrefix(asset.Path, "/") {
+		return fmt.Errorf("asset path %q is absolute", asset.Path)
+	}
+	clean := path.Clean(asset.Path)
+	if clean != asset.Path || clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
+		return fmt.Errorf("asset path %q is not clean relative path", asset.Path)
+	}
+	if asset.Digest == "" {
+		return fmt.Errorf("asset %q digest is empty", asset.Path)
+	}
+	return nil
+}
+
+func digestFor(s string) string {
+	h := sha256.Sum256([]byte(s))
+	return "sha256:" + hex.EncodeToString(h[:])
+}
diff --git a/cmd/internal/runtimebundle/manifest_test.go b/cmd/internal/runtimebundle/manifest_test.go
new file mode 100644
index 0000000..2aa69ce
--- /dev/null
+++ b/cmd/internal/runtimebundle/manifest_test.go
@@ -0,0 +1,58 @@
+package runtimebundle
+
+import "testing"
+
+func TestManifestDigestIsOrderIndependent(t *testing.T) {
+	a := RuntimeManifest{
+		Assets: []RuntimeAsset{
+			{Path: "bin/pair-shell", Mode: 0o755, Size: 3, Digest: "aaa"},
+			{Path: "nvim/init.lua", Mode: 0o644, Size: 4, Digest: "bbb"},
+		},
+	}
+	b := RuntimeManifest{
+		Assets: []RuntimeAsset{
+			{Path: "nvim/init.lua", Mode: 0o644, Size: 4, Digest: "bbb"},
+			{Path: "bin/pair-shell", Mode: 0o755, Size: 3, Digest: "aaa"},
+		},
+	}
+
+	gotA, err := a.ManifestDigest()
+	if err != nil {
+		t.Fatalf("ManifestDigest(a) error = %v", err)
+	}
+	gotB, err := b.ManifestDigest()
+	if err != nil {
+		t.Fatalf("ManifestDigest(b) error = %v", err)
+	}
+	if gotA == "" {
+		t.Fatal("ManifestDigest() = empty")
+	}
+	if gotA != gotB {
+		t.Fatalf("digest differs by order: %q != %q", gotA, gotB)
+	}
+}
+
+func TestManifestRejectsUnsafePaths(t *testing.T) {
+	tests := []struct {
+		name   string
+		assets []RuntimeAsset
+	}{
+		{name: "empty", assets: []RuntimeAsset{{Path: "", Mode: 0o644, Digest: "a"}}},
+		{name: "absolute", assets: []RuntimeAsset{{Path: "/bin/pair-shell", Mode: 0o755, Digest: "a"}}},
+		{name: "dotdot", assets: []RuntimeAsset{{Path: "bin/../pair-shell", Mode: 0o755, Digest: "a"}}},
+		{name: "duplicate", assets: []RuntimeAsset{
+			{Path: "bin/pair-shell", Mode: 0o755, Digest: "a"},
+			{Path: "bin/pair-shell", Mode: 0o755, Digest: "a"},
+		}},
+		{name: "empty digest", assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755}}},
+	}
+
+	for _, tt := range tests {
+		t.Run(tt.name, func(t *testing.T) {
+			err := RuntimeManifest{Assets: tt.assets}.Validate()
+			if err == nil {
+				t.Fatal("Validate() error = nil, want error")
+			}
+		})
+	}
+}
diff --git a/cmd/internal/runtimebundle/plan.go b/cmd/internal/runtimebundle/plan.go
new file mode 100644
index 0000000..adcb33d
--- /dev/null
+++ b/cmd/internal/runtimebundle/plan.go
@@ -0,0 +1,63 @@
+package runtimebundle
+
+import (
+	"fmt"
+	"path/filepath"
+)
+
+type ExistingAsset struct {
+	Mode   uint32
+	Size   int64
+	Digest string
+}
+
+type ExtractionInput struct {
+	StoreRoot   string
+	RuntimeRoot string
+	Manifest    RuntimeManifest
+	Existing    map[string]ExistingAsset
+}
+
+type ExtractionPlan struct {
+	Writes []RuntimeAsset
+	Skips  []string
+}
+
+func PlanExtraction(input ExtractionInput) (ExtractionPlan, error) {
+	if err := input.Manifest.Validate(); err != nil {
+		return ExtractionPlan{}, err
+	}
+	if err := validateRuntimeRoot(input.StoreRoot, input.RuntimeRoot); err != nil {
+		return ExtractionPlan{}, err
+	}
+	existing := input.Existing
+	if existing == nil {
+		existing = map[string]ExistingAsset{}
+	}
+	plan := ExtractionPlan{}
+	for _, asset := range input.Manifest.Assets {
+		got, ok := existing[asset.Path]
+		if ok && got.Mode == asset.Mode && got.Size == asset.Size && got.Digest == asset.Digest {
+			plan.Skips = append(plan.Skips, asset.Path)
+			continue
+		}
+		plan.Writes = append(plan.Writes, asset)
+	}
+	return plan, nil
+}
+
+func validateRuntimeRoot(storeRoot, runtimeRoot string) error {
+	if storeRoot == "" || runtimeRoot == "" {
+		return fmt.Errorf("store root and runtime root are required")
+	}
+	store := filepath.Clean(storeRoot)
+	root := filepath.Clean(runtimeRoot)
+	rel, err := filepath.Rel(store, root)
+	if err != nil {
+		return err
+	}
+	if rel == "." || rel == ".." || rel == "" || (len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator)) {
+		return fmt.Errorf("runtime root %q is outside store root %q", runtimeRoot, storeRoot)
+	}
+	return nil
+}
diff --git a/cmd/internal/runtimebundle/plan_test.go b/cmd/internal/runtimebundle/plan_test.go
new file mode 100644
index 0000000..ff2a09a
--- /dev/null
+++ b/cmd/internal/runtimebundle/plan_test.go
@@ -0,0 +1,78 @@
+package runtimebundle
+
+import (
+	"path/filepath"
+	"testing"
+)
+
+func TestPlanExtractionWritesMissingAssets(t *testing.T) {
+	manifest := RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755, Size: 10, Digest: "sha256:a"}}}
+
+	plan, err := PlanExtraction(ExtractionInput{
+		StoreRoot:   "/data/pair/runtime",
+		RuntimeRoot: "/data/pair/runtime/abc/pair-home",
+		Manifest:    manifest,
+		Existing:    map[string]ExistingAsset{},
+	})
+	if err != nil {
+		t.Fatalf("PlanExtraction error = %v", err)
+	}
+	if len(plan.Writes) != 1 || plan.Writes[0].Path != "bin/pair-shell" {
+		t.Fatalf("Writes = %#v, want bin/pair-shell", plan.Writes)
+	}
+	if len(plan.Skips) != 0 {
+		t.Fatalf("Skips = %#v, want empty", plan.Skips)
+	}
+}
+
+func TestPlanExtractionSkipsMatchingAssets(t *testing.T) {
+	manifest := RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755, Size: 10, Digest: "sha256:a"}}}
+
+	plan, err := PlanExtraction(ExtractionInput{
+		StoreRoot:   "/data/pair/runtime",
+		RuntimeRoot: "/data/pair/runtime/abc/pair-home",
+		Manifest:    manifest,
+		Existing: map[string]ExistingAsset{
+			"bin/pair-shell": {Mode: 0o755, Size: 10, Digest: "sha256:a"},
+		},
+	})
+	if err != nil {
+		t.Fatalf("PlanExtraction error = %v", err)
+	}
+	if len(plan.Writes) != 0 {
+		t.Fatalf("Writes = %#v, want empty", plan.Writes)
+	}
+	if len(plan.Skips) != 1 || plan.Skips[0] != "bin/pair-shell" {
+		t.Fatalf("Skips = %#v, want bin/pair-shell", plan.Skips)
+	}
+}
+
+func TestPlanExtractionRefreshesMismatchedDigest(t *testing.T) {
+	manifest := RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755, Size: 10, Digest: "sha256:a"}}}
+
+	plan, err := PlanExtraction(ExtractionInput{
+		StoreRoot:   "/data/pair/runtime",
+		RuntimeRoot: "/data/pair/runtime/abc/pair-home",
+		Manifest:    manifest,
+		Existing: map[string]ExistingAsset{
+			"bin/pair-shell": {Mode: 0o755, Size: 10, Digest: "sha256:old"},
+		},
+	})
+	if err != nil {
+		t.Fatalf("PlanExtraction error = %v", err)
+	}
+	if len(plan.Writes) != 1 || plan.Writes[0].Path != "bin/pair-shell" {
+		t.Fatalf("Writes = %#v, want refresh", plan.Writes)
+	}
+}
+
+func TestPlanExtractionRejectsRuntimeRootOutsideStore(t *testing.T) {
+	_, err := PlanExtraction(ExtractionInput{
+		StoreRoot:   "/data/pair/runtime",
+		RuntimeRoot: filepath.Clean("/data/pair/not-runtime/abc/pair-home"),
+		Manifest:    RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755, Digest: "sha256:a"}}},
+	})
+	if err == nil {
+		t.Fatal("PlanExtraction error = nil, want root containment error")
+	}
+}
diff --git a/cmd/internal/runtimebundle/store.go b/cmd/internal/runtimebundle/store.go
new file mode 100644
index 0000000..cd19f22
--- /dev/null
+++ b/cmd/internal/runtimebundle/store.go
@@ -0,0 +1,187 @@
+package runtimebundle
+
+import (
+	"crypto/sha256"
+	"encoding/hex"
+	"encoding/json"
+	"fmt"
+	"os"
+	"path/filepath"
+	"sort"
+	"time"
+)
+
+type StoreInput struct {
+	StoreRoot string
+	Manifest  RuntimeManifest
+	ReadAsset func(string) ([]byte, error)
+	Keep      int
+}
+
+type StoreResult struct {
+	Digest   string
+	PairHome string
+}
+
+type runtimeMarker struct {
+	Digest     string `json:"digest"`
+	AssetCount int    `json:"asset_count"`
+	Generated  string `json:"generated,omitempty"`
+}
+
+func Extract(input StoreInput) (StoreResult, error) {
+	if input.ReadAsset == nil {
+		return StoreResult{}, fmt.Errorf("asset reader is required")
+	}
+	digest, err := input.Manifest.ManifestDigest()
+	if err != nil {
+		return StoreResult{}, err
+	}
+	pairHome := filepath.Join(input.StoreRoot, digest, "pair-home")
+	existing, err := scanExisting(pairHome, input.Manifest)
+	if err != nil {
+		return StoreResult{}, err
+	}
+	plan, err := PlanExtraction(ExtractionInput{
+		StoreRoot:   input.StoreRoot,
+		RuntimeRoot: pairHome,
+		Manifest:    input.Manifest,
+		Existing:    existing,
+	})
+	if err != nil {
+		return StoreResult{}, err
+	}
+	for _, asset := range plan.Writes {
+		data, err := input.ReadAsset(asset.Path)
+		if err != nil {
+			return StoreResult{}, fmt.Errorf("read embedded asset %s: %w", asset.Path, err)
+		}
+		if digestFor(string(data)) != asset.Digest {
+			// Binary assets may not be valid UTF-8 but string preserves bytes.
+			return StoreResult{}, fmt.Errorf("embedded asset %s digest mismatch", asset.Path)
+		}
+		if int64(len(data)) != asset.Size {
+			return StoreResult{}, fmt.Errorf("embedded asset %s size mismatch", asset.Path)
+		}
+		if err := writeFileAtomic(filepath.Join(pairHome, filepath.FromSlash(asset.Path)), data, os.FileMode(asset.Mode)); err != nil {
+			return StoreResult{}, err
+		}
+	}
+	if err := writeMarker(filepath.Join(input.StoreRoot, digest, "manifest.json"), digest, len(input.Manifest.Assets)); err != nil {
+		return StoreResult{}, err
+	}
+	if err := applyCleanup(input.StoreRoot, digest, input.Keep); err != nil {
+		return StoreResult{}, err
+	}
+	return StoreResult{Digest: digest, PairHome: pairHome}, nil
+}
+
+func scanExisting(root string, manifest RuntimeManifest) (map[string]ExistingAsset, error) {
+	existing := map[string]ExistingAsset{}
+	for _, asset := range manifest.Assets {
+		p := filepath.Join(root, filepath.FromSlash(asset.Path))
+		info, err := os.Stat(p)
+		if err != nil {
+			if os.IsNotExist(err) {
+				continue
+			}
+			return nil, err
+		}
+		if info.IsDir() {
+			continue
+		}
+		data, err := os.ReadFile(p)
+		if err != nil {
+			return nil, err
+		}
+		h := sha256.Sum256(data)
+		existing[asset.Path] = ExistingAsset{
+			Mode:   uint32(info.Mode().Perm()),
+			Size:   info.Size(),
+			Digest: "sha256:" + hex.EncodeToString(h[:]),
+		}
+	}
+	return existing, nil
+}
+
+func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
+	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
+		return err
+	}
+	tmp := path + ".tmp"
+	if err := os.WriteFile(tmp, data, mode); err != nil {
+		_ = os.Remove(tmp)
+		return err
+	}
+	if err := os.Chmod(tmp, mode); err != nil {
+		_ = os.Remove(tmp)
+		return err
+	}
+	if err := os.Rename(tmp, path); err != nil {
+		_ = os.Remove(tmp)
+		return err
+	}
+	return nil
+}
+
+func writeMarker(path, digest string, assetCount int) error {
+	data, err := json.MarshalIndent(runtimeMarker{
+		Digest:     digest,
+		AssetCount: assetCount,
+		Generated:  time.Now().UTC().Format(time.RFC3339),
+	}, "", "  ")
+	if err != nil {
+		return err
+	}
+	data = append(data, '\n')
+	return writeFileAtomic(path, data, 0o644)
+}
+
+func applyCleanup(storeRoot, selectedDigest string, keep int) error {
+	entries, err := os.ReadDir(storeRoot)
+	if err != nil {
+		if os.IsNotExist(err) {
+			return nil
+		}
+		return err
+	}
+	gens := make([]RuntimeGeneration, 0, len(entries))
+	for _, entry := range entries {
+		if !entry.IsDir() {
+			continue
+		}
+		name := entry.Name()
+		info, err := entry.Info()
+		if err != nil {
+			return err
+		}
+		gens = append(gens, RuntimeGeneration{
+			Digest:    name,
+			HasMarker: markerValid(filepath.Join(storeRoot, name, "manifest.json")),
+			ModUnix:   info.ModTime().Unix(),
+		})
+	}
+	plan, err := PlanCleanup(CleanupInput{SelectedDigest: selectedDigest, Keep: keep, Generations: gens})
+	if err != nil {
+		return err
+	}
+	sort.Strings(plan.DeleteDigests)
+	for _, digest := range plan.DeleteDigests {
+		if err := os.RemoveAll(filepath.Join(storeRoot, digest)); err != nil {
+			return err
+		}
+	}
+	return nil
+}
+
+func markerValid(path string) bool {
+	data, err := os.ReadFile(path)
+	if err != nil {
+		return false
+	}
+	var marker runtimeMarker
+	if err := json.Unmarshal(data, &marker); err != nil {
+		return false
+	}
+	return marker.Digest != ""
+}
diff --git a/cmd/internal/runtimebundle/store_test.go b/cmd/internal/runtimebundle/store_test.go
new file mode 100644
index 0000000..d9b2db9
--- /dev/null
+++ b/cmd/internal/runtimebundle/store_test.go
@@ -0,0 +1,110 @@
+package runtimebundle
+
+import (
+	"os"
+	"path/filepath"
+	"testing"
+)
+
+func TestStoreExtractsAssetsAndWritesMarker(t *testing.T) {
+	dir := t.TempDir()
+	shellContent := "pair shell\n"
+	initContent := "init\n"
+	manifest := RuntimeManifest{Assets: []RuntimeAsset{
+		{Path: "bin/pair-shell", Mode: 0o755, Size: int64(len(shellContent)), Digest: digestFor(shellContent)},
+		{Path: "nvim/init.lua", Mode: 0o644, Size: int64(len(initContent)), Digest: digestFor(initContent)},
+	}}
+
+	res, err := Extract(StoreInput{
+		StoreRoot: dir,
+		Manifest:  manifest,
+		ReadAsset: fakeAssetReader(map[string]string{
+			"bin/pair-shell": shellContent,
+			"nvim/init.lua":  initContent,
+		}),
+		Keep: 1,
+	})
+	if err != nil {
+		t.Fatalf("Extract error = %v", err)
+	}
+	shell := filepath.Join(res.PairHome, "bin", "pair-shell")
+	got, err := os.ReadFile(shell)
+	if err != nil {
+		t.Fatalf("ReadFile(pair-shell) error = %v", err)
+	}
+	if string(got) != "pair shell\n" {
+		t.Fatalf("pair-shell content = %q", got)
+	}
+	info, err := os.Stat(shell)
+	if err != nil {
+		t.Fatalf("Stat(pair-shell) error = %v", err)
+	}
+	if info.Mode().Perm() != 0o755 {
+		t.Fatalf("pair-shell mode = %o, want 755", info.Mode().Perm())
+	}
+	marker, err := os.ReadFile(filepath.Join(filepath.Dir(res.PairHome), "manifest.json"))
+	if err != nil {
+		t.Fatalf("ReadFile(marker) error = %v", err)
+	}
+	if len(marker) == 0 {
+		t.Fatal("marker is empty")
+	}
+}
+
+func TestStoreSecondExtractIsIdempotent(t *testing.T) {
+	dir := t.TempDir()
+	content := "pair shell\n"
+	manifest := RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755, Size: int64(len(content)), Digest: digestFor(content)}}}
+	input := StoreInput{
+		StoreRoot: dir,
+		Manifest:  manifest,
+		ReadAsset: fakeAssetReader(map[string]string{"bin/pair-shell": content}),
+		Keep:      1,
+	}
+	first, err := Extract(input)
+	if err != nil {
+		t.Fatalf("first Extract error = %v", err)
+	}
+	second, err := Extract(input)
+	if err != nil {
+		t.Fatalf("second Extract error = %v", err)
+	}
+	if first.PairHome != second.PairHome {
+		t.Fatalf("PairHome changed: %q != %q", first.PairHome, second.PairHome)
+	}
+}
+
+func TestStoreCleanupPreservesSelectedRuntime(t *testing.T) {
+	dir := t.TempDir()
+	old := filepath.Join(dir, "aaaaaaaa", "pair-home")
+	if err := os.MkdirAll(old, 0o755); err != nil {
+		t.Fatalf("MkdirAll(old) error = %v", err)
+	}
+	if err := os.WriteFile(filepath.Join(dir, "aaaaaaaa", "manifest.json"), []byte(`{"digest":"aaaaaaaa","asset_count":0}`), 0o644); err != nil {
+		t.Fatalf("WriteFile(old marker) error = %v", err)
+	}
+	content := "pair shell\n"
+	manifest := RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755, Size: int64(len(content)), Digest: digestFor(content)}}}
+
+	res, err := Extract(StoreInput{
+		StoreRoot: dir,
+		Manifest:  manifest,
+		ReadAsset: fakeAssetReader(map[string]string{"bin/pair-shell": content}),
+		Keep:      0,
+	})
+	if err != nil {
+		t.Fatalf("Extract error = %v", err)
+	}
+	if _, err := os.Stat(res.PairHome); err != nil {
+		t.Fatalf("selected runtime was removed: %v", err)
+	}
+	if _, err := os.Stat(filepath.Join(dir, "aaaaaaaa")); !os.IsNotExist(err) {
+		t.Fatalf("old runtime still exists or stat failed unexpectedly: %v", err)
+	}
+}
+
+func fakeAssetReader(files map[string]string) func(string) ([]byte, error) {
+	return func(path string) ([]byte, error) {
+		return []byte(files[path]), nil
+	}
+}
diff --git a/cmd/internal/runtimebundlegen/generate.go b/cmd/internal/runtimebundlegen/generate.go
new file mode 100644
index 0000000..1d0f907
--- /dev/null
+++ b/cmd/internal/runtimebundlegen/generate.go
@@ -0,0 +1,224 @@
+package runtimebundlegen
+
+import (
+	"crypto/sha256"
+	"encoding/hex"
+	"encoding/json"
+	"fmt"
+	"io"
+	"io/fs"
+	"os"
+	"path/filepath"
+	"sort"
+	"strings"
+)
+
+type RuntimeAsset struct {
+	Path   string
+	Mode   uint32
+	Size   int64
+	Digest string
+}
+
+type RuntimeManifest struct {
+	Assets []RuntimeAsset
+}
+
+var explicitAssetPaths = []string{
+	"bin/pair-shell",
+	"bin/pair-help",
+	"bin/pair-notify",
+	"bin/pair-quit.sh",
+	"bin/pair-restart.sh",
+	"bin/pair-scrollback-open",
+	"bin/pair-changelog-open",
+	"bin/pair-review-open",
+	"bin/pair-review-readiness",
+	"bin/pair-review-target",
+	"bin/copy-on-select.sh",
+	"bin/clipboard-to-pane.sh",
+	"bin/flash-pane.sh",
+	"bin/pair-title.sh",
+	"bin/pair-session-watch.sh",
+	"bin/pair-wrap",
+	"bin/pair-slug",
+	"bin/pair-context",
+	"bin/pair-scrollback-render",
+	"bin/pair-changelog",
+	"bin/pair-continuation",
+	"bin/pair-session-watch",
+	"doctor/README.md",
+	"doctor/SKILL.md",
+	"doctor/doctor.sh",
+	"doctor/emitter-health.sh",
+}
+
+var assetDirs = []string{
+	"bin/lib",
+	"nvim",
+	"zellij",
+}
+
+type GenerateOptions struct {
+	RepoRoot string
+	OutRoot  string
+}
+
+func Generate(opts GenerateOptions) (RuntimeManifest, error) {
+	if opts.RepoRoot == "" {
+		opts.RepoRoot = "."
+	}
+	if opts.OutRoot == "" {
+		return RuntimeManifest{}, fmt.Errorf("output root is required")
+	}
+	repoRoot, err := filepath.Abs(opts.RepoRoot)
+	if err != nil {
+		return RuntimeManifest{}, err
+	}
+	outRoot, err := filepath.Abs(opts.OutRoot)
+	if err != nil {
+		return RuntimeManifest{}, err
+	}
+	outParent := filepath.Dir(outRoot)
+	outBase := filepath.Base(outRoot)
+	if err := os.MkdirAll(outParent, 0o755); err != nil {
+		return RuntimeManifest{}, err
+	}
+	stageRoot, err := os.MkdirTemp(outParent, "."+outBase+"-tmp-")
+	if err != nil {
+		return RuntimeManifest{}, err
+	}
+	committed := false
+	defer func() {
+		if !committed {
+			_ = os.RemoveAll(stageRoot)
+		}
+	}()
+	filesRoot := filepath.Join(stageRoot, "files")
+
+	paths := map[string]bool{}
+	for _, p := range explicitAssetPaths {
+		paths[p] = true
+	}
+	for _, dir := range assetDirs {
+		root := filepath.Join(repoRoot, filepath.FromSlash(dir))
+		if err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
+			if err != nil {
+				return err
+			}
+			if d.IsDir() {
+				return nil
+			}
+			rel, err := filepath.Rel(repoRoot, p)
+			if err != nil {
+				return err
+			}
+			logical := filepath.ToSlash(rel)
+			if shouldExclude(logical) {
+				return nil
+			}
+			paths[logical] = true
+			return nil
+		}); err != nil {
+			return RuntimeManifest{}, err
+		}
+	}
+
+	ordered := make([]string, 0, len(paths))
+	for p := range paths {
+		if shouldExclude(p) {
+			continue
+		}
+		ordered = append(ordered, p)
+	}
+	sort.Strings(ordered)
+
+	manifest := RuntimeManifest{Assets: make([]RuntimeAsset, 0, len(ordered))}
+	for _, logical := range ordered {
+		src := filepath.Join(repoRoot, filepath.FromSlash(logical))
+		info, err := os.Stat(src)
+		if err != nil {
+			return RuntimeManifest{}, fmt.Errorf("asset %s: %w", logical, err)
+		}
+		if info.IsDir() {
+			continue
+		}
+		digest, err := copyAsset(src, filepath.Join(filesRoot, filepath.FromSlash(logical)), info.Mode().Perm())
+		if err != nil {
+			return RuntimeManifest{}, err
+		}
+		manifest.Assets = append(manifest.Assets, RuntimeAsset{
+			Path:   logical,
+			Mode:   uint32(info.Mode().Perm()),
+			Size:   info.Size(),
+			Digest: "sha256:" + digest,
+		})
+	}
+	encoded, err := json.MarshalIndent(manifest, "", "  ")
+	if err != nil {
+		return RuntimeManifest{}, err
+	}
+	encoded = append(encoded, '\n')
+	if err := os.WriteFile(filepath.Join(stageRoot, "manifest.json"), encoded, 0o644); err != nil {
+		return RuntimeManifest{}, err
+	}
+	if err := os.RemoveAll(outRoot); err != nil {
+		return RuntimeManifest{}, err
+	}
+	if err := os.Rename(stageRoot, outRoot); err != nil {
+		return RuntimeManifest{}, err
+	}
+	committed = true
+	return manifest, nil
+}
+
+func shouldExclude(logical string) bool {
+	base := filepath.Base(logical)
+	if base == ".DS_Store" || strings.Contains(logical, "__pycache__/") {
+		return true
+	}
+	if strings.HasSuffix(logical, "_test.lua") {
+		return true
+	}
+	switch logical {
+	case "bin/pair", "bin/pair-go", "bin/pair-dev":
+		return true
+	}
+	return false
+}
+
+func copyAsset(src, dst string, mode os.FileMode) (string, error) {
+	in, err := os.Open(src)
+	if err != nil {
+		return "", err
+	}
+	defer in.Close()
+	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
+		return "", err
+	}
+	tmp := dst + ".tmp"
+	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
+	if err != nil {
+		return "", err
+	}
+	h := sha256.New()
+	_, copyErr := io.Copy(io.MultiWriter(out, h), in)
+	closeErr := out.Close()
+	if copyErr != nil {
+		_ = os.Remove(tmp)
+		return "", copyErr
+	}
+	if closeErr != nil {
+		_ = os.Remove(tmp)
+		return "", closeErr
+	}
+	if err := os.Chmod(tmp, mode); err != nil {
+		_ = os.Remove(tmp)
+		return "", err
+	}
+	if err := os.Rename(tmp, dst); err != nil {
+		_ = os.Remove(tmp)
+		return "", err
+	}
+	return hex.EncodeToString(h.Sum(nil)), nil
+}
diff --git a/cmd/internal/runtimebundlegen/generate_test.go b/cmd/internal/runtimebundlegen/generate_test.go
new file mode 100644
index 0000000..c2043d5
--- /dev/null
+++ b/cmd/internal/runtimebundlegen/generate_test.go
@@ -0,0 +1,56 @@
+package runtimebundlegen
+
+import (
+	"os"
+	"path/filepath"
+	"testing"
+)
+
+func TestGeneratePreservesExistingOutputOnFailure(t *testing.T) {
+	repo := t.TempDir()
+	out := filepath.Join(t.TempDir(), "runtime")
+	writeMinimalRuntimeRepo(t, repo)
+
+	if _, err := Generate(GenerateOptions{RepoRoot: repo, OutRoot: out}); err != nil {
+		t.Fatalf("initial Generate error = %v", err)
+	}
+	before, err := os.ReadFile(filepath.Join(out, "manifest.json"))
+	if err != nil {
+		t.Fatalf("ReadFile(initial manifest) error = %v", err)
+	}
+
+	if err := os.Remove(filepath.Join(repo, "bin", "pair-shell")); err != nil {
+		t.Fatalf("Remove(pair-shell) error = %v", err)
+	}
+	if _, err := Generate(GenerateOptions{RepoRoot: repo, OutRoot: out}); err == nil {
+		t.Fatal("Generate error = nil, want missing asset error")
+	}
+
+	after, err := os.ReadFile(filepath.Join(out, "manifest.json"))
+	if err != nil {
+		t.Fatalf("ReadFile(preserved manifest) error = %v", err)
+	}
+	if string(after) != string(before) {
+		t.Fatal("failed generation changed existing output manifest")
+	}
+}
+
+func writeMinimalRuntimeRepo(t *testing.T, repo string) {
+	t.Helper()
+	for _, logical := range explicitAssetPaths {
+		writeFile(t, filepath.Join(repo, filepath.FromSlash(logical)), "#!/bin/sh\n", 0o755)
+	}
+	writeFile(t, filepath.Join(repo, "bin", "lib", "shared.sh"), "shared\n", 0o644)
+	writeFile(t, filepath.Join(repo, "nvim", "init.lua"), "-- init\n", 0o644)
+	writeFile(t, filepath.Join(repo, "zellij", "config.kdl"), "keybinds {}\n", 0o644)
+}
+
+func writeFile(t *testing.T, path, content string, mode os.FileMode) {
+	t.Helper()
+	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
+		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
+	}
+	if err := os.WriteFile(path, []byte(content), mode); err != nil {
+		t.Fatalf("WriteFile(%s) error = %v", path, err)
+	}
+}
diff --git a/cmd/pair-go/main.go b/cmd/pair-go/main.go
index 6834be7..bb638e3 100644
--- a/cmd/pair-go/main.go
+++ b/cmd/pair-go/main.go
@@ -7,10 +7,13 @@ import (
 	"io"
 	"os"
 	"path/filepath"
+	"strings"
 	"syscall"
 
 	"github.com/xianxu/pair/cmd/internal/dispatcher"
 	"github.com/xianxu/pair/cmd/internal/entrypoint"
+	"github.com/xianxu/pair/cmd/internal/launcher"
+	"github.com/xianxu/pair/cmd/internal/runtimebundle"
 )
 
 var defaultPairHome string
@@ -29,6 +32,7 @@ type legacyRuntime interface {
 	DefaultPairHome() string
 	Stat(path string) error
 	Environ() []string
+	EmbeddedAssetRoot() (string, error)
 	Exec(label string, path string, argv []string, env []string) int
 }
 
@@ -64,11 +68,28 @@ func runLegacyLaunch(label string, executable string, args []string, stderr io.W
 		},
 	})
 	if err != nil {
-		_, _ = fmt.Fprintf(stderr, "%s: %v; run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", label, err)
-		return 1
+		embeddedRoot, embeddedErr := rt.EmbeddedAssetRoot()
+		if embeddedErr == nil && embeddedRoot != "" {
+			root, err = entrypoint.ResolveAssetRoot(entrypoint.AssetRootInput{
+				PairHome:        rt.PairHome(),
+				Executable:      executable,
+				DefaultPairHome: rt.DefaultPairHome(),
+				EmbeddedRoot:    embeddedRoot,
+				PairShellExists: func(root string) bool {
+					return rt.Stat(entrypoint.PairShellPath(root)) == nil
+				},
+			})
+		}
+		if err != nil {
+			if embeddedErr != nil {
+				_, _ = fmt.Fprintf(stderr, "%s: embedded runtime extraction failed: %v\n", label, embeddedErr)
+			}
+			_, _ = fmt.Fprintf(stderr, "%s: %v; run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", label, err)
+			return 1
+		}
 	}
 	req := entrypoint.ResolveLegacyLaunch(root, args)
-	return rt.Exec(label, req.Path, req.Argv, rt.Environ())
+	return rt.Exec(label, req.Path, req.Argv, withEnv(rt.Environ(), "PAIR_HOME", root.Root))
 }
 
 type osLegacyRuntime struct{}
@@ -101,6 +122,20 @@ func (osLegacyRuntime) Environ() []string {
 	return os.Environ()
 }
 
+func (osLegacyRuntime) EmbeddedAssetRoot() (string, error) {
+	dataDir := launcher.ResolveDataDir(os.Getenv("HOME"), os.Getenv("XDG_DATA_HOME"))
+	res, err := runtimebundle.Extract(runtimebundle.StoreInput{
+		StoreRoot: filepath.Join(dataDir, "runtime"),
+		Manifest:  runtimebundle.EmbeddedManifest(),
+		ReadAsset: runtimebundle.EmbeddedAsset,
+		Keep:      2,
+	})
+	if err != nil {
+		return "", err
+	}
+	return res.PairHome, nil
+}
+
 func (osLegacyRuntime) Exec(label string, path string, argv []string, env []string) int {
 	if err := syscall.Exec(path, argv, env); err != nil {
 		_, _ = fmt.Fprintf(os.Stderr, "%s: exec %s failed: %v\n", label, path, err)
@@ -118,3 +153,23 @@ func writeResult(res dispatcher.Result, stdout, stderr io.Writer) int {
 	}
 	return res.ExitCode
 }
+
+func withEnv(env []string, key, value string) []string {
+	prefix := key + "="
+	out := make([]string, 0, len(env)+1)
+	replaced := false
+	for _, entry := range env {
+		if strings.HasPrefix(entry, prefix) {
+			if !replaced {
+				out = append(out, prefix+value)
+				replaced = true
+			}
+			continue
+		}
+		out = append(out, entry)
+	}
+	if !replaced {
+		out = append(out, prefix+value)
+	}
+	return out
+}
diff --git a/cmd/pair-go/main_test.go b/cmd/pair-go/main_test.go
index 75dd2a7..c0b08fd 100644
--- a/cmd/pair-go/main_test.go
+++ b/cmd/pair-go/main_test.go
@@ -176,6 +176,32 @@ func TestRunDirectPairFallsBackToDefaultPairHome(t *testing.T) {
 	}
 }
 
+func TestRunDirectPairFallsBackToEmbeddedRuntimeAndSetsPairHome(t *testing.T) {
+	rt := &fakeLegacyRuntime{
+		executable:   "/home/me/.local/bin/pair",
+		embeddedRoot: "/data/pair/runtime/abc/pair-home",
+		roots:        map[string]bool{"/data/pair/runtime/abc/pair-home": true},
+		execCode:     9,
+		environ:      []string{"PATH=/bin", "PAIR_HOME=/old"},
+	}
+
+	var stdout, stderr bytes.Buffer
+	code := runWithLegacyRuntime([]string{"--help"}, &stdout, &stderr, rt)
+
+	if code != 9 {
+		t.Fatalf("code = %d, want 9", code)
+	}
+	if rt.execPath != "/data/pair/runtime/abc/pair-home/bin/pair-shell" {
+		t.Fatalf("execPath = %q, want embedded pair-shell", rt.execPath)
+	}
+	if !containsEnv(rt.execEnv, "PAIR_HOME=/data/pair/runtime/abc/pair-home") {
+		t.Fatalf("execEnv missing embedded PAIR_HOME: %#v", rt.execEnv)
+	}
+	if containsEnv(rt.execEnv, "PAIR_HOME=/old") {
+		t.Fatalf("execEnv kept old PAIR_HOME: %#v", rt.execEnv)
+	}
+}
+
 func TestRunPairGoHelperDoesNotProbeOrExecShellLauncher(t *testing.T) {
 	rt := &fakeLegacyRuntime{
 		executable: "/repo/bin/pair-go",
@@ -209,6 +235,9 @@ type fakeLegacyRuntime struct {
 	statErr         error
 	execCode        int
 	statCalls       int
+	embeddedRoot    string
+	embeddedErr     error
+	environ         []string
 
 	execPath  string
 	execLabel string
@@ -246,9 +275,16 @@ func (f *fakeLegacyRuntime) Stat(path string) error {
 }
 
 func (f *fakeLegacyRuntime) Environ() []string {
+	if f.environ != nil {
+		return f.environ
+	}
 	return os.Environ()
 }
 
+func (f *fakeLegacyRuntime) EmbeddedAssetRoot() (string, error) {
+	return f.embeddedRoot, f.embeddedErr
+}
+
 func (f *fakeLegacyRuntime) Exec(label string, path string, argv []string, env []string) int {
 	f.execLabel = label
 	f.execPath = path
diff --git a/tests/pair-embedded-runtime-test.sh b/tests/pair-embedded-runtime-test.sh
new file mode 100644
index 0000000..a735a4d
--- /dev/null
+++ b/tests/pair-embedded-runtime-test.sh
@@ -0,0 +1,112 @@
+#!/usr/bin/env bash
+set -euo pipefail
+
+repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
+tmp="$(mktemp -d "${TMPDIR:-/tmp}/pair-embedded-runtime.XXXXXX")"
+trap 'rm -rf "$tmp"' EXIT
+
+bin_dir="$tmp/bin"
+home="$tmp/home"
+xdg="$tmp/xdg"
+mkdir -p "$bin_dir" "$home" "$xdg"
+gomodcache="$(go env GOMODCACHE)"
+gocache="$(go env GOCACHE)"
+
+make -C "$repo_root" runtimebundle-generate >/dev/null
+go build -o "$bin_dir/pair" "$repo_root/cmd/pair-go"
+
+cat >"$bin_dir/zellij" <<'SH'
+#!/usr/bin/env bash
+set -eu
+printf '%s\n' "$*" >> "${ZELLIJ_LOG:?}"
+case "$*" in
+  "list-sessions --no-formatting"|"list-sessions --short")
+    exit 0
+    ;;
+  --session*" action list-clients")
+    exit 0
+    ;;
+  --config-dir*)
+    config=""
+    layout=""
+    prev=""
+    for arg in "$@"; do
+      if [ "$prev" = "--config-dir" ]; then config="$arg"; fi
+      if [ "$prev" = "--new-session-with-layout" ]; then layout="$arg"; fi
+      prev="$arg"
+    done
+    test -f "$config/config.kdl"
+    test -f "$layout"
+    case "$config" in */pair/runtime/*/pair-home/zellij) ;; *) printf 'bad config path: %s\n' "$config" >&2; exit 11 ;; esac
+    case "$layout" in */pair/runtime/*/pair-home/zellij/layouts/main.kdl) ;; *) printf 'bad layout path: %s\n' "$layout" >&2; exit 12 ;; esac
+    root="${config%/zellij}"
+    test -x "$root/bin/pair-shell"
+    test -x "$root/bin/pair-wrap"
+    test -x "$root/bin/pair-session-watch.sh"
+    test -x "$root/bin/pair-title.sh"
+    test -f "$root/nvim/init.lua"
+    printf '%s\n' "$root" > "${PAIR_SMOKE_ROOT:?}"
+    exit 0
+    ;;
+  *)
+    exit 0
+    ;;
+esac
+SH
+chmod +x "$bin_dir/zellij"
+
+cat >"$bin_dir/ps" <<'SH'
+#!/usr/bin/env bash
+case "$*" in
+  "-o comm= -p "*)
+    printf 'sh\n'
+    ;;
+  "-o ppid= -p "*)
+    printf '1\n'
+    ;;
+  *)
+    exec /bin/ps "$@"
+    ;;
+esac
+SH
+chmod +x "$bin_dir/ps"
+
+export PATH="$bin_dir:$PATH"
+export HOME="$home"
+export XDG_DATA_HOME="$xdg"
+export GOMODCACHE="$gomodcache"
+export GOCACHE="$gocache"
+export ZELLIJ_LOG="$tmp/zellij.log"
+export PAIR_SMOKE_ROOT="$tmp/root"
+unset PAIR_DEV PAIR_HOME PAIR_TAG PAIR_AGENT PAIR_AGENT_ARGS PAIR_DATA_DIR ZELLIJ_SESSION_NAME ZELLIJ ZELLIJ_PANE_ID
+
+help_out="$("$bin_dir/pair" --help)"
+case "$help_out" in
+  pair\ —*) ;;
+  *)
+    printf 'copied pair --help did not reach embedded pair-shell help; first bytes:\n%s\n' "$help_out" >&2
+    exit 1
+    ;;
+esac
+
+mkdir -p "$xdg/pair/runtime/aaaaaaaa/pair-home" \
+         "$xdg/pair/runtime/bbbbbbbb/pair-home" \
+         "$xdg/pair/runtime/cccccccc/pair-home"
+printf '{"digest":"aaaaaaaa","asset_count":0}\n' > "$xdg/pair/runtime/aaaaaaaa/manifest.json"
+printf '{"digest":"bbbbbbbb","asset_count":0}\n' > "$xdg/pair/runtime/bbbbbbbb/manifest.json"
+printf '{"digest":"cccccccc","asset_count":0}\n' > "$xdg/pair/runtime/cccccccc/manifest.json"
+touch -t 202001010000 "$xdg/pair/runtime/aaaaaaaa"
+touch -t 202001020000 "$xdg/pair/runtime/bbbbbbbb"
+touch -t 202001030000 "$xdg/pair/runtime/cccccccc"
+
+"$bin_dir/pair" resume smoke >/dev/null
+
+test -s "$PAIR_SMOKE_ROOT"
+root="$(cat "$PAIR_SMOKE_ROOT")"
+case "$root" in */pair/runtime/*/pair-home) ;; *) printf 'bad extracted root: %s\n' "$root" >&2; exit 1 ;; esac
+test -d "$root"
+test ! -e "$xdg/pair/runtime/aaaaaaaa"
+test -d "$xdg/pair/runtime/bbbbbbbb"
+test -d "$xdg/pair/runtime/cccccccc"
+
+printf 'pair embedded runtime smoke passed\n'
diff --git a/workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md b/workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md
index 2c30dfc..5ee59dd 100644
--- a/workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md
+++ b/workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md
@@ -1,6 +1,6 @@
 # Self-Contained Pair Binary Implementation Plan
 
-> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.
+> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [x]`) syntax for tracking.
 
 **Goal:** Build a `pair` binary that can launch from a clean copied location by extracting embedded Pair-owned runtime assets into a versioned runtime root.
 
@@ -148,7 +148,7 @@ Runtime root contract:
 - Create: `cmd/internal/runtimebundle/manifest.go`
 - Create: `cmd/internal/runtimebundle/manifest_test.go`
 
-- [ ] **Step 1: Write failing tests for manifest validation and digest stability**
+- [x] **Step 1: Write failing tests for manifest validation and digest stability**
 
 Add tests that construct a small manifest in memory and assert:
 
@@ -162,17 +162,17 @@ func TestManifestRejectsUnsafePaths(t *testing.T) {
 }
 ```
 
-- [ ] **Step 2: Run the red tests**
+- [x] **Step 2: Run the red tests**
 
 Run: `go test ./cmd/internal/runtimebundle -run 'TestManifest' -count=1`
 
 Expected: package or symbols missing.
 
-- [ ] **Step 3: Implement `RuntimeAsset`, `RuntimeManifest`, validation, and digest**
+- [x] **Step 3: Implement `RuntimeAsset`, `RuntimeManifest`, validation, and digest**
 
 Keep this pure: no filesystem or embed access. Normalize paths with slash semantics, sort entries for hashing, and include path, mode, size, and content digest in the manifest digest.
 
-- [ ] **Step 4: Run green tests**
+- [x] **Step 4: Run green tests**
 
 Run: `go test ./cmd/internal/runtimebundle -run 'TestManifest' -count=1`
 
@@ -186,7 +186,7 @@ Expected: pass.
 - Create: `cmd/internal/runtimebundle/cleanup.go`
 - Create: `cmd/internal/runtimebundle/cleanup_test.go`
 
-- [ ] **Step 1: Write failing tests for extraction idempotency**
+- [x] **Step 1: Write failing tests for extraction idempotency**
 
 Cover these behaviors:
 
@@ -197,21 +197,21 @@ func TestPlanExtractionRefreshesMismatchedDigest(t *testing.T) {}
 func TestPlanExtractionRejectsRuntimeRootOutsideStore(t *testing.T) {}
 ```
 
-- [ ] **Step 2: Write failing tests for cleanup safety**
+- [x] **Step 2: Write failing tests for cleanup safety**
 
 Cover keeping the selected `$PAIR_DATA_DIR/runtime/<digest>/` generation, retaining the newest configured count, and deleting only directories that match the runtime root contract above.
 
-- [ ] **Step 3: Run the red tests**
+- [x] **Step 3: Run the red tests**
 
 Run: `go test ./cmd/internal/runtimebundle -run 'TestPlan|TestCleanup' -count=1`
 
 Expected: missing functions or failing behavior.
 
-- [ ] **Step 4: Implement pure planning**
+- [x] **Step 4: Implement pure planning**
 
 Represent write/skip/delete as data. Do not call `os.WriteFile`, `os.RemoveAll`, or `time.Now` from the pure functions; pass observed filesystem state and current runtime identity in.
 
-- [ ] **Step 5: Run green tests**
+- [x] **Step 5: Run green tests**
 
 Run: `go test ./cmd/internal/runtimebundle -count=1`
 
@@ -229,11 +229,11 @@ Expected: pass.
 - Modify: `Makefile.local`
 - Test: `cmd/internal/runtimebundle/embed_test.go`
 
-- [ ] **Step 1: Write failing tests that assert all required runtime assets exist in the embedded manifest**
+- [x] **Step 1: Write failing tests that assert all required runtime assets exist in the embedded manifest**
 
 Assert the exact asset boundary above: launcher shell, shell helper scripts, `bin/lib/**`, required Go helper binaries, `nvim/**`, `zellij/**`, and doctor runtime diagnostics are present; excluded entrypoints/caches/tests are absent. Use `atlas/go-migration-inventory.md` only as human review context; automated tests inspect the generated manifest, not atlas.
 
-- [ ] **Step 2: Run the red test before adding embed code**
+- [x] **Step 2: Run the red test before adding embed code**
 
 Run: `go test ./cmd/internal/runtimebundle -run TestEmbeddedManifestContainsLaunchAssets -count=1`
 
@@ -241,15 +241,15 @@ Expected: missing embedded manifest symbol. This red run happens before
 `embed.go` is added, so it does not trip Go's compile-time `//go:embed` missing
 file rule.
 
-- [ ] **Step 3: Add the deterministic generator and embedding path**
+- [x] **Step 3: Add the deterministic generator and embedding path**
 
 Implement `make runtimebundle-generate` to copy the required inputs into `cmd/internal/runtimebundle/assets/runtime/`, preserving executable bits and writing a manifest JSON with sorted paths. Add `//go:embed assets/runtime/**` and expose the embedded `RuntimeManifest` plus file reader through `RuntimeFS`.
 
-- [ ] **Step 4: Wire build, test, and staleness checks**
+- [x] **Step 4: Wire build, test, and staleness checks**
 
 Update `PAIR_GO_SRCS` so changes to runtimebundle sources rebuild `bin/pair` and `bin/pair-go`. Add `test-runtimebundle` and `runtimebundle-drift-check`; include both in the relevant test path. Ensure `make build` builds helper binaries first, regenerates the asset tree, then builds `pair`/`pair-go` so the single binary embeds current helpers without committing generated binaries.
 
-- [ ] **Step 5: Run green tests**
+- [x] **Step 5: Run green tests**
 
 Run: `make test-runtimebundle`
 
@@ -261,21 +261,21 @@ Expected: pass.
 - Create: `cmd/internal/runtimebundle/store.go`
 - Create: `cmd/internal/runtimebundle/store_test.go`
 
-- [ ] **Step 1: Write failing filesystem tests**
+- [x] **Step 1: Write failing filesystem tests**
 
 Use `t.TempDir()` and a fake embedded asset reader. Assert atomic first-run extraction, executable bits for `bin/*` scripts, idempotent second run, manifest marker write, and stale cleanup that preserves the selected runtime.
 
-- [ ] **Step 2: Run the red tests**
+- [x] **Step 2: Run the red tests**
 
 Run: `go test ./cmd/internal/runtimebundle -run 'TestStore|TestExtract' -count=1`
 
 Expected: missing store implementation.
 
-- [ ] **Step 3: Implement the thin IO shell**
+- [x] **Step 3: Implement the thin IO shell**
 
 Use temp files plus rename for file writes. Write a manifest marker after all assets land. Apply cleanup after successful extraction/selection, not before.
 
-- [ ] **Step 4: Run green tests**
+- [x] **Step 4: Run green tests**
 
 Run: `make test-runtimebundle`
 
@@ -291,7 +291,7 @@ Expected: pass.
 - Modify: `cmd/pair-go/main.go`
 - Modify: `cmd/pair-go/main_test.go`
 
-- [ ] **Step 1: Write failing tests for launch-root precedence**
+- [x] **Step 1: Write failing tests for launch-root precedence**
 
 Assert ordering:
 
@@ -300,21 +300,21 @@ Assert ordering:
 3. `defaultPairHome` wins.
 4. Embedded extracted runtime is used only when the adjacent/default candidates fail.
 
-- [ ] **Step 2: Write failing `cmd/pair-go` fake-runtime tests**
+- [x] **Step 2: Write failing `cmd/pair-go` fake-runtime tests**
 
 Extend the existing `legacyRuntime` fake so launch records `PAIR_HOME=<extracted-root>` in the exec env when embedded fallback is selected.
 
-- [ ] **Step 3: Run red tests**
+- [x] **Step 3: Run red tests**
 
 Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -run 'TestResolveAssetRoot|TestRun' -count=1`
 
 Expected: embedded fallback behavior missing.
 
-- [ ] **Step 4: Implement embedded fallback**
+- [x] **Step 4: Implement embedded fallback**
 
 Keep `ResolveAssetRoot` deterministic and side-effect-free by passing the already-prepared embedded root as an optional candidate. Let `cmd/pair-go` decide whether extraction is needed and then pass the selected root into the existing legacy launch handoff.
 
-- [ ] **Step 5: Run green tests**
+- [x] **Step 5: Run green tests**
 
 Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
 
@@ -326,11 +326,11 @@ Expected: pass.
 - Create: `tests/pair-embedded-runtime-test.sh`
 - Modify: `Makefile.local`
 
-- [ ] **Step 1: Write the failing clean-copy help smoke test**
+- [x] **Step 1: Write the failing clean-copy help smoke test**
 
 Build `bin/pair`, copy only that binary to a temp `PATH`, set `PAIR_DATA_DIR` to a temp dir, and run a harmless launch path such as `--help` through `pair-shell`. Use fake external commands only where needed so the test does not start a real zellij session.
 
-- [ ] **Step 2: Write the failing session-asset smoke test**
+- [x] **Step 2: Write the failing session-asset smoke test**
 
 Run the copied binary through a create-path with fake `zellij`, `nvim`, `fzf`,
 `jq`, `par`, and fake agent commands on `PATH`. The fake `zellij` must record the
@@ -341,21 +341,21 @@ must verify the embedded runtime contains and can resolve `pair-wrap`,
 the copied binary reaches the compatibility launch/session asset path, not only
 the help text.
 
-- [ ] **Step 3: Add upgrade/stale-runtime assertions**
+- [x] **Step 3: Add upgrade/stale-runtime assertions**
 
 Simulate an old extracted runtime under the temp data/cache area and assert it is pruned only after the new runtime is selected; assert the current runtime remains.
 
-- [ ] **Step 4: Run the red smoke**
+- [x] **Step 4: Run the red smoke**
 
 Run: `bash tests/pair-embedded-runtime-test.sh`
 
 Expected: copied binary cannot find `pair-shell` before implementation.
 
-- [ ] **Step 5: Add the `make` target**
+- [x] **Step 5: Add the `make` target**
 
 Add `test-pair-embedded-runtime` and `runtimebundle-drift-check` to `.PHONY`; add the smoke target to the main `test` dependency list after `test-pair-go-install-layout`.
 
-- [ ] **Step 6: Run green smoke**
+- [x] **Step 6: Run green smoke**
 
 Run: `make test-pair-embedded-runtime`
 
@@ -372,15 +372,15 @@ Expected: copied-binary clean and upgrade paths pass.
 - Modify: `README.md`
 - Optional modify: sibling `../homebrew-pair/Formula/pair.rb`
 
-- [ ] **Step 1: Update atlas execution path**
+- [x] **Step 1: Update atlas execution path**
 
 Record that #90 embeds/extracts Pair-owned runtime assets and that adjacent layout remains supported for source/Homebrew compatibility.
 
-- [ ] **Step 2: Update install docs**
+- [x] **Step 2: Update install docs**
 
 Document the copied-binary deployment mode and keep external dependencies explicit.
 
-- [ ] **Step 3: Audit Homebrew notes**
+- [x] **Step 3: Audit Homebrew notes**
 
 This issue does not require moving Homebrew to binary-only packaging. If the formula still packages adjacent assets, leave behavior unchanged and document it as compatibility mode in `README.md` / atlas. Touch `../homebrew-pair/Formula/pair.rb` only if it contains false adjacent-only claims or build rules that conflict with the generated bundle; if touched, verify with `ruby -c ../homebrew-pair/Formula/pair.rb` and record the sibling commit.
 
@@ -393,7 +393,7 @@ a separate sibling commit and record that commit hash in this issue's `## Log`.
 **Files:**
 - Modify: `workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md`
 
-- [ ] **Step 1: Run focused Go tests**
+- [x] **Step 1: Run focused Go tests**
 
 Run: `make test-runtimebundle`
 
@@ -403,7 +403,7 @@ Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
 
 Expected: pass.
 
-- [ ] **Step 2: Run smoke and build tests**
+- [x] **Step 2: Run smoke and build tests**
 
 Run: `make build`
 
@@ -413,13 +413,13 @@ Run: `make test-pair-go-install-layout test-pair-embedded-runtime`
 
 Expected: pass.
 
-- [ ] **Step 3: Run broader Go tests**
+- [x] **Step 3: Run broader Go tests**
 
 Run: `go test ./... -count=1`
 
 Expected: pass.
 
-- [ ] **Step 4: Validate issue and diff hygiene**
+- [x] **Step 4: Validate issue and diff hygiene**
 
 Run: `sdlc issue validate workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md`
 
@@ -429,6 +429,6 @@ Run: `git diff --check`
 
 Expected: no whitespace errors.
 
-- [ ] **Step 5: Update issue plan/log**
+- [x] **Step 5: Update issue plan/log**
 
 Tick completed issue plan items, add verification evidence to `## Log`, and prepare for `sdlc close --issue 90 --verified '<evidence>' --no-atlas` only if atlas truly had no new surface. This issue is expected to require atlas updates, so avoid `--no-atlas` unless the implementation scope changes.


codex
I’m using `superpowers-using-superpowers` for the required session-start skill protocol, then I’ll inspect the issue/plan and changed files directly for the boundary review.
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
I’ve confirmed the SDLC contract. Now I’m reading the issue record, durable plan, and the actual files behind the diff so the review is against repo state, not just the pasted patch.
exec
/bin/zsh -lc "sed -n '1,460p' workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "sed -n '1,260p' workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
---
id: 000090
status: working
deps: []
github_issue:
created: 2026-07-01
updated: 2026-07-01
estimate_hours: 5.44
started: 2026-07-01T00:18:42-07:00
---

# self-contained pair binary with embedded runtime assets

## Problem

After #79, the public `pair` command is Go-owned, but deployment is still an
installed tree: the Go entrypoint must find adjacent Pair-owned runtime assets
such as `bin/pair-shell`, shell helpers, `nvim/`, `zellij/`, and helper
binaries. That is simpler for Homebrew, but it is not the deployment shape we
eventually want: copying one Pair binary around and having it work.

The long-term direction is a true native single binary. Rewriting every
remaining shell and orchestration surface directly into Go is too much risk in
one jump, so the next step should make the current runtime tree derive from one
Go artifact without pretending the shell lifecycle is already gone.

## Spec

Add a self-contained deployment mode for the Go `pair` binary:

- Embed the Pair-owned runtime assets needed by launch/session flows into the Go
  binary.
- On first run, extract those assets to a versioned runtime directory under the
  user's Pair data/cache area, then run the existing launch flow with `PAIR_HOME`
  pointed at that extracted runtime root.
- Preserve the current adjacent-install behavior for source checkout and
  Homebrew layouts unless/until the self-contained mode proves it can replace
  them.
- Keep external programs external: `zellij`, `nvim`, `fzf`, `jq` while the shell
  runtime still needs it, clipboard tools, and agent CLIs are not bundled by this
  issue.
- Make runtime extraction deterministic, idempotent, and upgrade-safe: a new
  binary/runtime version should extract a new directory or refresh only when the
  embedded manifest changes.
- Add a cleanup policy for stale extracted runtimes that cannot delete the
  currently running runtime.

Execution path toward the native single binary:

1. Embed and extract the existing runtime tree. This delivers the "single Pair
   artifact" deployment option while retaining the tested shell/nvim/zellij
   contracts.
2. Route generated internal calls through the Go dispatcher where possible
   (`pair wrap`, `pair slug`, `pair changelog`, `pair continuation`, etc.) while
   keeping compatibility names only as shims.
3. Port stateful shell orchestrators into Go one at a time: launcher/session
   lifecycle, scrollback/changelog openers, title poller, review helpers,
   clipboard helpers.
4. Once shell ownership is gone, stop extracting shell scripts and use embedded
   or generated native assets only for `nvim/` and `zellij/`.
5. Revisit whether `nvim/` and `zellij/` remain extracted native assets or move
   to generated temp files/API-driven startup. The native single binary target
   is one Pair executable, with external platform tools still supplied by the
   system.

Architecture notes:

- `ARCH-PURPOSE`: the copied binary must be enough to provide Pair-owned
  runtime assets; falling back to a source checkout does not satisfy this issue.
- `ARCH-DRY`: the embedded runtime manifest must be the single source of what is
  packaged, installed, and tested. Do not maintain a separate hand-written asset
  list for Homebrew, tests, and extraction.
- `ARCH-PURE`: keep manifest planning, runtime selection, and extraction
  decisions as pure functions with unit tests; keep filesystem writes and
  process exec in thin seams.

## Done when

- [x] A release build can produce one `pair` binary that contains the Pair-owned
      runtime assets needed for launch/session flows.
- [x] Copying only that binary to a clean path works when external dependencies
      are installed.
- [x] First run extracts or refreshes a versioned runtime root and points
      `PAIR_HOME` at it for the compatibility launch handoff.
- [x] Adjacent source/Homebrew layouts still work.
- [x] Upgrade and stale-runtime cleanup behavior is tested.
- [x] The execution path toward the true native single binary is documented in
      atlas.

## Plan

- [x] Define the embedded runtime manifest and generated asset list.
- [x] Implement runtime extraction and version/manifest selection.
- [x] Wire `cmd/pair-go` to prefer extracted embedded runtime when no adjacent
      asset root exists.
- [x] Add install/copy smoke tests for clean and upgrade paths.
- [x] Update README, atlas, and Homebrew packaging notes.

Detailed implementation plan:
`workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md`.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale, so the number is provisional but uses the required method.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.20 impl=0.08
item: greenfield-go-module design=0.60 impl=0.56
item: smaller-go-module design=0.35 impl=0.48
item: cross-cutting-refactor design=0.80 impl=1.12
item: atlas-docs design=0.25 impl=0.20
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 5.44
```

## Log

### 2026-07-01

Created after #79 closed: #79 made `pair` Go-owned but intentionally retained
the adjacent runtime tree. The desired final direction is a true native single
binary; this issue captures the lower-risk next step of embedding/extracting
the current runtime tree first.

Claimed and entered planning. `sdlc start-plan --issue 90` delivered
`ARCH-DRY`, `ARCH-PURE`, and `ARCH-PURPOSE`; the durable plan keeps the runtime
manifest as the packaging source of truth, pure planning/selection functions in
Go, and copied-binary launch as the acceptance path rather than a follow-up.

First `sdlc change-code --issue 90` plan-quality gate returned FAILURE: asset
boundary, generator/staleness contract, and copied-binary smoke were too loose.
Refined the durable plan to name exact runtime asset roots/exclusions, require a
deterministic gitignored generator plus drift check, exercise a fake
launch/session path, and bound Homebrew formula edits to false/conflicting
packaging claims only.

Second `sdlc change-code --issue 90` plan-quality gate returned FAILURE on
remaining precision issues: `bin/pair-title.sh` was referenced by smoke coverage
but missing from the required asset list, extracted runtime naming/version rules
were implicit, and atlas wording could imply a second source. Updated the plan
to include `pair-title.sh`, define `$PAIR_DATA_DIR/runtime/<digest>/pair-home`
plus marker/cleanup rules, and state that automated behavior derives only from
the generated manifest and runtime marker.

Third `sdlc change-code --issue 90` plan-quality gate returned FAILURE because
raw `go test ./cmd/internal/runtimebundle` would fail from a clean checkout once
`//go:embed` references the gitignored generated asset tree. Updated the plan to
add `make test-runtimebundle` as the generated-assets-before-test path after
`embed.go` exists, keep earlier pure tests as raw `go test`, and spell out the
peer-repo `AGENTS.local.md` / `MEMORY.md` requirement before any optional
Homebrew tap edit.

Implemented the embedded runtime path. Added the generated runtime manifest and
bundle generator, pure manifest/extraction/cleanup planning, embedded asset
reader, runtime extraction store, and `cmd/pair-go` fallback that extracts to
`$PAIR_DATA_DIR/runtime/<digest>/pair-home` only after `PAIR_HOME`, executable
sibling assets, and build-time `defaultPairHome` fail. Source/Homebrew adjacent
layouts remain first in the selection order.

Added copied-binary smoke coverage with fake external dependencies for `pair
--help`, `pair resume smoke`, required extracted assets, `PAIR_HOME` handoff,
and stale-runtime cleanup. During verification, parallel smoke runs exposed that
`runtimebundle-generate` rewrote the shared output tree in place; added a
regression test for preserving existing output on failed generation and changed
the generator to stage output in a unique temp directory before replacing the
published bundle (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`).

Updated `README.md`, `atlas/architecture.md`, and
`atlas/go-migration-inventory.md` to document the implemented embedded fallback,
manifest ownership, cleanup behavior, and remaining external dependencies.
No Homebrew tap edit was needed because the adjacent `libexec` packaging path
remains accurate and intentionally precedes embedded fallback.

Verification passed:

- `make test-runtimebundle`
- `make runtimebundle-drift-check`
- `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
- `make build`
- `make test-pair-go-install-layout`
- `make test-pair-embedded-runtime`
- `go test ./... -count=1`
- `sdlc issue validate workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md`
- `git diff --check`

 succeeded in 0ms:
# Self-Contained Pair Binary Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Build a `pair` binary that can launch from a clean copied location by extracting embedded Pair-owned runtime assets into a versioned runtime root.

**Architecture:** Add a generated runtime manifest as the single source for packaged assets (`ARCH-DRY`), a pure `runtimebundle` package for manifest hashing, version selection, extraction planning, and cleanup planning (`ARCH-PURE`), and a thin `cmd/pair-go` integration that only falls back to embedded extraction when `PAIR_HOME`, executable siblings, and `defaultPairHome` do not provide `bin/pair-shell`. This satisfies the issue purpose by proving copied-binary launch without relying on a source checkout (`ARCH-PURPOSE`).

**Tech Stack:** Go 1.26, `embed.FS`, Makefile build rules, POSIX shell smoke tests, existing `cmd/internal/entrypoint` launch handoff.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `RuntimeAsset` | `cmd/internal/runtimebundle/manifest.go` | new |
| `RuntimeManifest` | `cmd/internal/runtimebundle/manifest.go` | new |
| `ExtractionPlan` | `cmd/internal/runtimebundle/plan.go` | new |
| `RuntimeSelection` | `cmd/internal/entrypoint/asset_root.go` | modified |
| `CleanupPlan` | `cmd/internal/runtimebundle/cleanup.go` | new |

- **RuntimeAsset** — one embedded file entry: logical path, mode, size, and digest.
  - **Relationships:** N:1 with `RuntimeManifest`; one manifest owns many assets.
  - **DRY rationale:** Avoid parallel hand-maintained asset lists for embedding, install notes, and tests.
  - **Future extensions:** Can add generated-command shim metadata when helper binaries move behind dispatcher routes.

- **RuntimeManifest** — the deterministic package contract for Pair-owned runtime files.
  - **Relationships:** 1:N with `RuntimeAsset`; 1:1 with an embedded runtime version/digest.
  - **DRY rationale:** Single source for extraction, clean-copy smoke expectations, and atlas/Homebrew docs.
  - **Future extensions:** Can widen from static file assets to generated native assets.

- **ExtractionPlan** — pure decision for which files to write, chmod, leave alone, or reject.
  - **Relationships:** 1:1 with one target runtime root and one manifest.
  - **DRY rationale:** Keeps idempotency/upgrade rules out of filesystem code and makes them directly unit-testable.
  - **Future extensions:** Can add partial refresh or corrupted-file repair without changing callers.

- **RuntimeSelection** — existing launch-root decision extended to include an optional embedded runtime root after adjacent candidates fail.
  - **Relationships:** `cmd/pair-go` asks for one `RuntimeSelection`; it yields the `AssetRoot` passed to `ResolveLegacyLaunch`.
  - **DRY rationale:** Reuse `ResolveAssetRoot` ordering instead of adding a second launch resolver.
  - **Future extensions:** Can remove adjacent/default fallbacks when embedded mode becomes the primary release shape.

- **CleanupPlan** — pure stale-runtime pruning decision that never returns the currently selected runtime.
  - **Relationships:** N old runtime dirs to zero or more delete candidates.
  - **DRY rationale:** Shared cleanup policy for first-run extraction and future maintenance commands.
  - **Future extensions:** Retention count/age can become configurable without touching extraction.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `RuntimeFS` | `cmd/internal/runtimebundle/embed.go` | new | `embed.FS` |
| `RuntimeStore` | `cmd/internal/runtimebundle/store.go` | new | filesystem writes, chmod, rename |
| `PairGoLegacyRuntime` | `cmd/pair-go/main.go` | modified | `os`, `syscall.Exec`, env |
| `BuildManifest` | `cmd/internal/runtimebundle/generate.go`, `cmd/internal/runtimebundle/assets/**`, `Makefile.local` | new | repo file tree at build time |
| `CopiedBinarySmoke` | `tests/pair-embedded-runtime-test.sh` | new | real built binary with fake external commands |

- **RuntimeFS** — exposes embedded asset bytes and manifest data.
  - **Injected into:** `RuntimeStore` extraction runner and `cmd/pair-go` runtime setup.
  - **Future extensions:** Swap generated bytes for generated config content without changing launch.

- **RuntimeStore** — applies `ExtractionPlan` atomically under the user Pair data/cache area.
  - **Injected into:** `cmd/pair-go` through the existing `legacyRuntime` seam.
  - **Future extensions:** Add lock-file serialization if concurrent first-runs race on the same runtime root.

- **PairGoLegacyRuntime** — OS-backed seam that resolves paths, environment, extraction, and final exec.
  - **Injected into:** `runWithLegacyRuntime` tests via fake runtime.
  - **Future extensions:** Native launcher can reuse the same data/cache path helpers.

- **BuildManifest** — Makefile/generator path that keeps embedded assets synchronized with repo runtime files.
  - **Injected into:** `go generate` or `make build` before compiling `cmd/pair-go`.
  - **Future extensions:** Homebrew can audit the manifest rather than restating asset paths.

#### Runtime Asset Boundary

The embedded bundle must be a generated, gitignored package-local asset tree under
`cmd/internal/runtimebundle/assets/`; Go's `embed` package reads that tree at
compile time. The generator is the single source of the file list (`ARCH-DRY`):
tests and docs audit the generated `RuntimeManifest`, not a second list.

Required inputs:

- `bin/pair-shell`, `bin/pair-help`, `bin/pair-notify`, `bin/pair-quit.sh`,
  `bin/pair-restart.sh`, `bin/pair-scrollback-open`,
  `bin/pair-changelog-open`, `bin/pair-review-open`,
  `bin/pair-review-readiness`, `bin/pair-review-target`,
  `bin/copy-on-select.sh`, `bin/clipboard-to-pane.sh`, `bin/flash-pane.sh`,
  `bin/pair-title.sh`, `bin/pair-session-watch.sh`, and `bin/lib/**`.
- Go-built helper binaries used by the retained shell/KDL/Lua runtime:
  `bin/pair-wrap`, `bin/pair-slug`, `bin/pair-context`,
  `bin/pair-scrollback-render`, `bin/pair-changelog`,
  `bin/pair-continuation`, and `bin/pair-session-watch`.
- Native assets `nvim/**`, `zellij/**`, and runtime diagnostic assets
  `doctor/README.md`, `doctor/SKILL.md`, `doctor/doctor.sh`,
  `doctor/emitter-health.sh`.

Excluded inputs:

- Public entrypoints `bin/pair`, `bin/pair-go`, and `bin/pair-dev` to avoid
  embedding the launching binary or dev-only wrapper.
- External programs (`zellij`, `nvim`, `fzf`, `jq`, clipboard tools, agent CLIs),
  generated caches (`__pycache__`, `.DS_Store`), test files, and workflow/docs
  that are not read by launch/session flows.

Generator contract:

- `make runtimebundle-generate` removes and rebuilds the gitignored asset tree
  deterministically after helper binaries are built.
- `make runtimebundle-drift-check` regenerates twice into temp dirs and compares
  manifest JSON plus file bytes/modes; nondeterminism fails.
- `make test-runtimebundle` depends on helper binaries plus
  `runtimebundle-generate`, then runs `go test ./cmd/internal/runtimebundle
  -count=1`. After `embed.go` exists, use this target for package tests from a
  clean checkout; raw `go test ./cmd/internal/runtimebundle` is only valid after
  the asset tree has been generated.
- `make build` depends on `runtimebundle-generate` before building `bin/pair` and
  `bin/pair-go`; helper binaries are built first so the embedded bundle contains
  the current helper executables.

Runtime root contract:

- The extracted root is `$PAIR_DATA_DIR/runtime/<manifest-digest>/pair-home`,
  where `<manifest-digest>` is the `RuntimeManifest` digest. The directory that
  cleanup treats as one runtime generation is `$PAIR_DATA_DIR/runtime/<digest>/`.
- Extraction writes `$PAIR_DATA_DIR/runtime/<digest>/manifest.json` after all
  assets land under `pair-home/`; the marker includes the manifest digest,
  generated-at build metadata if available, and the asset count.
- Cleanup only considers direct children of `$PAIR_DATA_DIR/runtime/` whose names
  are valid manifest digests and whose `manifest.json` parses as a Pair runtime
  marker. It must always exclude the selected `<digest>` generation, even if the
  generation is older than the retention policy.
- Tests and production code derive behavior from the generated `RuntimeManifest`
  and runtime marker only. `atlas/go-migration-inventory.md` remains review
  context and documentation; no code or automated test should parse atlas as an
  input source.

- **CopiedBinarySmoke** — process-level fake test for external dependencies and copied-binary launch.
  - **Injected into:** `make test` via a focused target.
  - **Future extensions:** Can become the release packaging smoke for Homebrew and tarball builds.

## Chunk 1: Manifest And Pure Runtime Planning

### Task 1: Define the manifest contract

**Files:**
- Create: `cmd/internal/runtimebundle/manifest.go`
- Create: `cmd/internal/runtimebundle/manifest_test.go`

- [x] **Step 1: Write failing tests for manifest validation and digest stability**

Add tests that construct a small manifest in memory and assert:

```go
func TestManifestDigestIsOrderIndependent(t *testing.T) {
	// Same logical assets in different input order produce the same digest.
}

func TestManifestRejectsUnsafePaths(t *testing.T) {
	// Absolute paths, "..", empty path, and duplicate paths fail validation.
}
```

- [x] **Step 2: Run the red tests**

Run: `go test ./cmd/internal/runtimebundle -run 'TestManifest' -count=1`

Expected: package or symbols missing.

- [x] **Step 3: Implement `RuntimeAsset`, `RuntimeManifest`, validation, and digest**

Keep this pure: no filesystem or embed access. Normalize paths with slash semantics, sort entries for hashing, and include path, mode, size, and content digest in the manifest digest.

- [x] **Step 4: Run green tests**

Run: `go test ./cmd/internal/runtimebundle -run 'TestManifest' -count=1`

Expected: pass.

### Task 2: Define extraction and cleanup plans

**Files:**
- Create: `cmd/internal/runtimebundle/plan.go`
- Create: `cmd/internal/runtimebundle/plan_test.go`
- Create: `cmd/internal/runtimebundle/cleanup.go`
- Create: `cmd/internal/runtimebundle/cleanup_test.go`

- [x] **Step 1: Write failing tests for extraction idempotency**

Cover these behaviors:

```go
func TestPlanExtractionWritesMissingAssets(t *testing.T) {}
func TestPlanExtractionSkipsMatchingAssets(t *testing.T) {}
func TestPlanExtractionRefreshesMismatchedDigest(t *testing.T) {}
func TestPlanExtractionRejectsRuntimeRootOutsideStore(t *testing.T) {}
```

- [x] **Step 2: Write failing tests for cleanup safety**

Cover keeping the selected `$PAIR_DATA_DIR/runtime/<digest>/` generation, retaining the newest configured count, and deleting only directories that match the runtime root contract above.

- [x] **Step 3: Run the red tests**

Run: `go test ./cmd/internal/runtimebundle -run 'TestPlan|TestCleanup' -count=1`

Expected: missing functions or failing behavior.

- [x] **Step 4: Implement pure planning**

Represent write/skip/delete as data. Do not call `os.WriteFile`, `os.RemoveAll`, or `time.Now` from the pure functions; pass observed filesystem state and current runtime identity in.

- [x] **Step 5: Run green tests**

Run: `go test ./cmd/internal/runtimebundle -count=1`

Expected: pass.

## Chunk 2: Embedded Asset Generation And Extraction IO

### Task 3: Generate or maintain one embedded asset source

**Files:**
- Create: `cmd/internal/runtimebundle/generate.go`
- Create or generate: `cmd/internal/runtimebundle/embed.go`
- Generate, gitignored: `cmd/internal/runtimebundle/assets/**`
- Modify: `.gitignore`
- Modify: `Makefile.local`
- Test: `cmd/internal/runtimebundle/embed_test.go`

- [x] **Step 1: Write failing tests that assert all required runtime assets exist in the embedded manifest**

Assert the exact asset boundary above: launcher shell, shell helper scripts, `bin/lib/**`, required Go helper binaries, `nvim/**`, `zellij/**`, and doctor runtime diagnostics are present; excluded entrypoints/caches/tests are absent. Use `atlas/go-migration-inventory.md` only as human review context; automated tests inspect the generated manifest, not atlas.

- [x] **Step 2: Run the red test before adding embed code**

Run: `go test ./cmd/internal/runtimebundle -run TestEmbeddedManifestContainsLaunchAssets -count=1`

Expected: missing embedded manifest symbol. This red run happens before
`embed.go` is added, so it does not trip Go's compile-time `//go:embed` missing
file rule.

- [x] **Step 3: Add the deterministic generator and embedding path**

Implement `make runtimebundle-generate` to copy the required inputs into `cmd/internal/runtimebundle/assets/runtime/`, preserving executable bits and writing a manifest JSON with sorted paths. Add `//go:embed assets/runtime/**` and expose the embedded `RuntimeManifest` plus file reader through `RuntimeFS`.

- [x] **Step 4: Wire build, test, and staleness checks**

Update `PAIR_GO_SRCS` so changes to runtimebundle sources rebuild `bin/pair` and `bin/pair-go`. Add `test-runtimebundle` and `runtimebundle-drift-check`; include both in the relevant test path. Ensure `make build` builds helper binaries first, regenerates the asset tree, then builds `pair`/`pair-go` so the single binary embeds current helpers without committing generated binaries.

- [x] **Step 5: Run green tests**

Run: `make test-runtimebundle`

Expected: pass.

### Task 4: Implement filesystem extraction

**Files:**
- Create: `cmd/internal/runtimebundle/store.go`
- Create: `cmd/internal/runtimebundle/store_test.go`

- [x] **Step 1: Write failing filesystem tests**

Use `t.TempDir()` and a fake embedded asset reader. Assert atomic first-run extraction, executable bits for `bin/*` scripts, idempotent second run, manifest marker write, and stale cleanup that preserves the selected runtime.

- [x] **Step 2: Run the red tests**

Run: `go test ./cmd/internal/runtimebundle -run 'TestStore|TestExtract' -count=1`

Expected: missing store implementation.

- [x] **Step 3: Implement the thin IO shell**

Use temp files plus rename for file writes. Write a manifest marker after all assets land. Apply cleanup after successful extraction/selection, not before.

- [x] **Step 4: Run green tests**

Run: `make test-runtimebundle`

Expected: pass.

## Chunk 3: Launch Integration And Smoke Coverage

### Task 5: Extend asset-root selection to embedded fallback

**Files:**
- Modify: `cmd/internal/entrypoint/asset_root.go`
- Modify: `cmd/internal/entrypoint/asset_root_test.go`
- Modify: `cmd/pair-go/main.go`
- Modify: `cmd/pair-go/main_test.go`

- [x] **Step 1: Write failing tests for launch-root precedence**

Assert ordering:

1. `PAIR_HOME` with `bin/pair-shell` wins.
2. Executable sibling root wins.
3. `defaultPairHome` wins.
4. Embedded extracted runtime is used only when the adjacent/default candidates fail.

- [x] **Step 2: Write failing `cmd/pair-go` fake-runtime tests**

Extend the existing `legacyRuntime` fake so launch records `PAIR_HOME=<extracted-root>` in the exec env when embedded fallback is selected.

- [x] **Step 3: Run red tests**

Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -run 'TestResolveAssetRoot|TestRun' -count=1`

Expected: embedded fallback behavior missing.

- [x] **Step 4: Implement embedded fallback**

Keep `ResolveAssetRoot` deterministic and side-effect-free by passing the already-prepared embedded root as an optional candidate. Let `cmd/pair-go` decide whether extraction is needed and then pass the selected root into the existing legacy launch handoff.

- [x] **Step 5: Run green tests**

Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`

Expected: pass.

### Task 6: Add copied-binary and upgrade smoke tests

**Files:**
- Create: `tests/pair-embedded-runtime-test.sh`
- Modify: `Makefile.local`

- [x] **Step 1: Write the failing clean-copy help smoke test**

Build `bin/pair`, copy only that binary to a temp `PATH`, set `PAIR_DATA_DIR` to a temp dir, and run a harmless launch path such as `--help` through `pair-shell`. Use fake external commands only where needed so the test does not start a real zellij session.

- [x] **Step 2: Write the failing session-asset smoke test**

Run the copied binary through a create-path with fake `zellij`, `nvim`, `fzf`,
`jq`, `par`, and fake agent commands on `PATH`. The fake `zellij` must record the
`--config-dir` and `--new-session-with-layout` arguments, read both files, and
fail if they do not come from the extracted `PAIR_HOME`. The fake layout check
must verify the embedded runtime contains and can resolve `pair-wrap`,
`pair-session-watch.sh`, `pair-title.sh`, and `nvim/init.lua`. This smoke proves
the copied binary reaches the compatibility launch/session asset path, not only
the help text.

- [x] **Step 3: Add upgrade/stale-runtime assertions**

Simulate an old extracted runtime under the temp data/cache area and assert it is pruned only after the new runtime is selected; assert the current runtime remains.

- [x] **Step 4: Run the red smoke**

Run: `bash tests/pair-embedded-runtime-test.sh`

Expected: copied binary cannot find `pair-shell` before implementation.

- [x] **Step 5: Add the `make` target**

Add `test-pair-embedded-runtime` and `runtimebundle-drift-check` to `.PHONY`; add the smoke target to the main `test` dependency list after `test-pair-go-install-layout`.

- [x] **Step 6: Run green smoke**

Run: `make test-pair-embedded-runtime`

Expected: copied-binary clean and upgrade paths pass.

## Chunk 4: Docs, Packaging Notes, And Final Verification

### Task 7: Update docs and atlas

**Files:**
- Modify: `atlas/go-migration-inventory.md`
- Modify: `atlas/architecture.md`
- Modify: `atlas/index.md` if a new atlas page is added
- Modify: `README.md`
- Optional modify: sibling `../homebrew-pair/Formula/pair.rb`

- [x] **Step 1: Update atlas execution path**

Record that #90 embeds/extracts Pair-owned runtime assets and that adjacent layout remains supported for source/Homebrew compatibility.

- [x] **Step 2: Update install docs**

Document the copied-binary deployment mode and keep external dependencies explicit.

- [x] **Step 3: Audit Homebrew notes**

This issue does not require moving Homebrew to binary-only packaging. If the formula still packages adjacent assets, leave behavior unchanged and document it as compatibility mode in `README.md` / atlas. Touch `../homebrew-pair/Formula/pair.rb` only if it contains false adjacent-only claims or build rules that conflict with the generated bundle; if touched, verify with `ruby -c ../homebrew-pair/Formula/pair.rb` and record the sibling commit.

Before touching `../homebrew-pair`, follow the peer-repo rule from AGENTS.md:
read that repo's `AGENTS.local.md` and `MEMORY.md` first, then keep its change in
a separate sibling commit and record that commit hash in this issue's `## Log`.

### Task 8: Run verification and close-ready checks

**Files:**
- Modify: `workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md`

- [x] **Step 1: Run focused Go tests**

Run: `make test-runtimebundle`

Expected: pass.

Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`

Expected: pass.

- [x] **Step 2: Run smoke and build tests**

Run: `make build`

Expected: pass.

Run: `make test-pair-go-install-layout test-pair-embedded-runtime`

Expected: pass.

- [x] **Step 3: Run broader Go tests**

Run: `go test ./... -count=1`

Expected: pass.

- [x] **Step 4: Validate issue and diff hygiene**

Run: `sdlc issue validate workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md`

Expected: pass.

Run: `git diff --check`

Expected: no whitespace errors.

- [x] **Step 5: Update issue plan/log**

Tick completed issue plan items, add verification evidence to `## Log`, and prepare for `sdlc close --issue 90 --verified '<evidence>' --no-atlas` only if atlas truly had no new surface. This issue is expected to require atlas updates, so avoid `--no-atlas` unless the implementation scope changes.

exec
/bin/zsh -lc 'git diff --stat e1ae416c6084399c155e0a89c5597712f679460b..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 .gitignore                                         |   1 +
 Makefile.local                                     |  31 ++-
 README.md                                          |  19 +-
 atlas/architecture.md                              |  22 +-
 atlas/go-migration-inventory.md                    |  35 ++--
 cmd/internal/entrypoint/asset_root.go              |   4 +
 cmd/internal/entrypoint/asset_root_test.go         |  18 ++
 cmd/internal/runtimebundle/cleanup.go              |  68 +++++++
 cmd/internal/runtimebundle/cleanup_test.go         |  40 ++++
 cmd/internal/runtimebundle/embed.go                |  25 +++
 cmd/internal/runtimebundle/embed_test.go           |  45 +++++
 cmd/internal/runtimebundle/generatecmd/main.go     |  19 ++
 cmd/internal/runtimebundle/manifest.go             |  73 +++++++
 cmd/internal/runtimebundle/manifest_test.go        |  58 ++++++
 cmd/internal/runtimebundle/plan.go                 |  63 ++++++
 cmd/internal/runtimebundle/plan_test.go            |  78 +++++++
 cmd/internal/runtimebundle/store.go                | 187 +++++++++++++++++
 cmd/internal/runtimebundle/store_test.go           | 110 ++++++++++
 cmd/internal/runtimebundlegen/generate.go          | 224 +++++++++++++++++++++
 cmd/internal/runtimebundlegen/generate_test.go     |  56 ++++++
 cmd/pair-go/main.go                                |  61 +++++-
 cmd/pair-go/main_test.go                           |  36 ++++
 tests/pair-embedded-runtime-test.sh                | 112 +++++++++++
 ...ned-pair-binary-with-embedded-runtime-assets.md |  55 ++++-
 ...air-binary-with-embedded-runtime-assets-plan.md |  76 +++----
 25 files changed, 1434 insertions(+), 82 deletions(-)

codex
The issue and plan are explicit about `$PAIR_DATA_DIR/runtime/<digest>/pair-home`. I’m checking the implementation path now, especially environment/data-dir handling, generator coverage, and whether pure/planning boundaries stayed true.
exec
/bin/zsh -lc "nl -ba cmd/pair-go/main.go | sed -n '1,230p'" in /Users/xianxu/workspace/pair
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
    10		"strings"
    11		"syscall"
    12	
    13		"github.com/xianxu/pair/cmd/internal/dispatcher"
    14		"github.com/xianxu/pair/cmd/internal/entrypoint"
    15		"github.com/xianxu/pair/cmd/internal/launcher"
    16		"github.com/xianxu/pair/cmd/internal/runtimebundle"
    17	)
    18	
    19	var defaultPairHome string
    20	
    21	func main() {
    22		os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
    23	}
    24	
    25	func run(args []string, stdout, stderr io.Writer) int {
    26		return runWithLegacyRuntime(args, stdout, stderr, osLegacyRuntime{})
    27	}
    28	
    29	type legacyRuntime interface {
    30		Executable() (string, error)
    31		PairHome() string
    32		DefaultPairHome() string
    33		Stat(path string) error
    34		Environ() []string
    35		EmbeddedAssetRoot() (string, error)
    36		Exec(label string, path string, argv []string, env []string) int
    37	}
    38	
    39	func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
    40		exe, err := rt.Executable()
    41		if err != nil {
    42			if len(args) > 0 && args[0] == "launch" {
    43				_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
    44				return 1
    45			}
    46			res := dispatcher.Dispatch(args)
    47			return writeResult(res, stdout, stderr)
    48		}
    49	
    50		switch entrypoint.ClassifyInvocation(exe, args) {
    51		case entrypoint.ModePublicPair:
    52			return runLegacyLaunch("pair", exe, args, stderr, rt)
    53		case entrypoint.ModePairGoLaunch:
    54			return runLegacyLaunch("pair-go launch", exe, args[1:], stderr, rt)
    55		default:
    56			res := dispatcher.Dispatch(args)
    57			return writeResult(res, stdout, stderr)
    58		}
    59	}
    60	
    61	func runLegacyLaunch(label string, executable string, args []string, stderr io.Writer, rt legacyRuntime) int {
    62		root, err := entrypoint.ResolveAssetRoot(entrypoint.AssetRootInput{
    63			PairHome:        rt.PairHome(),
    64			Executable:      executable,
    65			DefaultPairHome: rt.DefaultPairHome(),
    66			PairShellExists: func(root string) bool {
    67				return rt.Stat(entrypoint.PairShellPath(root)) == nil
    68			},
    69		})
    70		if err != nil {
    71			embeddedRoot, embeddedErr := rt.EmbeddedAssetRoot()
    72			if embeddedErr == nil && embeddedRoot != "" {
    73				root, err = entrypoint.ResolveAssetRoot(entrypoint.AssetRootInput{
    74					PairHome:        rt.PairHome(),
    75					Executable:      executable,
    76					DefaultPairHome: rt.DefaultPairHome(),
    77					EmbeddedRoot:    embeddedRoot,
    78					PairShellExists: func(root string) bool {
    79						return rt.Stat(entrypoint.PairShellPath(root)) == nil
    80					},
    81				})
    82			}
    83			if err != nil {
    84				if embeddedErr != nil {
    85					_, _ = fmt.Fprintf(stderr, "%s: embedded runtime extraction failed: %v\n", label, embeddedErr)
    86				}
    87				_, _ = fmt.Fprintf(stderr, "%s: %v; run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", label, err)
    88				return 1
    89			}
    90		}
    91		req := entrypoint.ResolveLegacyLaunch(root, args)
    92		return rt.Exec(label, req.Path, req.Argv, withEnv(rt.Environ(), "PAIR_HOME", root.Root))
    93	}
    94	
    95	type osLegacyRuntime struct{}
    96	
    97	func (osLegacyRuntime) Executable() (string, error) {
    98		return os.Executable()
    99	}
   100	
   101	func (osLegacyRuntime) PairHome() string {
   102		return os.Getenv("PAIR_HOME")
   103	}
   104	
   105	func (osLegacyRuntime) DefaultPairHome() string {
   106		return defaultPairHome
   107	}
   108	
   109	func (osLegacyRuntime) Stat(path string) error {
   110		path = filepath.Clean(path)
   111		info, err := os.Stat(path)
   112		if err != nil {
   113			return err
   114		}
   115		if info.IsDir() {
   116			return fmt.Errorf("is a directory")
   117		}
   118		return nil
   119	}
   120	
   121	func (osLegacyRuntime) Environ() []string {
   122		return os.Environ()
   123	}
   124	
   125	func (osLegacyRuntime) EmbeddedAssetRoot() (string, error) {
   126		dataDir := launcher.ResolveDataDir(os.Getenv("HOME"), os.Getenv("XDG_DATA_HOME"))
   127		res, err := runtimebundle.Extract(runtimebundle.StoreInput{
   128			StoreRoot: filepath.Join(dataDir, "runtime"),
   129			Manifest:  runtimebundle.EmbeddedManifest(),
   130			ReadAsset: runtimebundle.EmbeddedAsset,
   131			Keep:      2,
   132		})
   133		if err != nil {
   134			return "", err
   135		}
   136		return res.PairHome, nil
   137	}
   138	
   139	func (osLegacyRuntime) Exec(label string, path string, argv []string, env []string) int {
   140		if err := syscall.Exec(path, argv, env); err != nil {
   141			_, _ = fmt.Fprintf(os.Stderr, "%s: exec %s failed: %v\n", label, path, err)
   142			return 1
   143		}
   144		return 0
   145	}
   146	
   147	func writeResult(res dispatcher.Result, stdout, stderr io.Writer) int {
   148		if res.Stdout != "" {
   149			_, _ = io.WriteString(stdout, res.Stdout)
   150		}
   151		if res.Stderr != "" {
   152			_, _ = io.WriteString(stderr, res.Stderr)
   153		}
   154		return res.ExitCode
   155	}
   156	
   157	func withEnv(env []string, key, value string) []string {
   158		prefix := key + "="
   159		out := make([]string, 0, len(env)+1)
   160		replaced := false
   161		for _, entry := range env {
   162			if strings.HasPrefix(entry, prefix) {
   163				if !replaced {
   164					out = append(out, prefix+value)
   165					replaced = true
   166				}
   167				continue
   168			}
   169			out = append(out, entry)
   170		}
   171		if !replaced {
   172			out = append(out, prefix+value)
   173		}
   174		return out
   175	}

exec
/bin/zsh -lc "nl -ba cmd/internal/runtimebundlegen/generate.go | sed -n '1,270p'" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "nl -ba cmd/internal/runtimebundle/store.go | sed -n '1,240p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package runtimebundle
     2	
     3	import (
     4		"crypto/sha256"
     5		"encoding/hex"
     6		"encoding/json"
     7		"fmt"
     8		"os"
     9		"path/filepath"
    10		"sort"
    11		"time"
    12	)
    13	
    14	type StoreInput struct {
    15		StoreRoot string
    16		Manifest  RuntimeManifest
    17		ReadAsset func(string) ([]byte, error)
    18		Keep      int
    19	}
    20	
    21	type StoreResult struct {
    22		Digest   string
    23		PairHome string
    24	}
    25	
    26	type runtimeMarker struct {
    27		Digest     string `json:"digest"`
    28		AssetCount int    `json:"asset_count"`
    29		Generated  string `json:"generated,omitempty"`
    30	}
    31	
    32	func Extract(input StoreInput) (StoreResult, error) {
    33		if input.ReadAsset == nil {
    34			return StoreResult{}, fmt.Errorf("asset reader is required")
    35		}
    36		digest, err := input.Manifest.ManifestDigest()
    37		if err != nil {
    38			return StoreResult{}, err
    39		}
    40		pairHome := filepath.Join(input.StoreRoot, digest, "pair-home")
    41		existing, err := scanExisting(pairHome, input.Manifest)
    42		if err != nil {
    43			return StoreResult{}, err
    44		}
    45		plan, err := PlanExtraction(ExtractionInput{
    46			StoreRoot:   input.StoreRoot,
    47			RuntimeRoot: pairHome,
    48			Manifest:    input.Manifest,
    49			Existing:    existing,
    50		})
    51		if err != nil {
    52			return StoreResult{}, err
    53		}
    54		for _, asset := range plan.Writes {
    55			data, err := input.ReadAsset(asset.Path)
    56			if err != nil {
    57				return StoreResult{}, fmt.Errorf("read embedded asset %s: %w", asset.Path, err)
    58			}
    59			if digestFor(string(data)) != asset.Digest {
    60				// Binary assets may not be valid UTF-8 but string preserves bytes.
    61				return StoreResult{}, fmt.Errorf("embedded asset %s digest mismatch", asset.Path)
    62			}
    63			if int64(len(data)) != asset.Size {
    64				return StoreResult{}, fmt.Errorf("embedded asset %s size mismatch", asset.Path)
    65			}
    66			if err := writeFileAtomic(filepath.Join(pairHome, filepath.FromSlash(asset.Path)), data, os.FileMode(asset.Mode)); err != nil {
    67				return StoreResult{}, err
    68			}
    69		}
    70		if err := writeMarker(filepath.Join(input.StoreRoot, digest, "manifest.json"), digest, len(input.Manifest.Assets)); err != nil {
    71			return StoreResult{}, err
    72		}
    73		if err := applyCleanup(input.StoreRoot, digest, input.Keep); err != nil {
    74			return StoreResult{}, err
    75		}
    76		return StoreResult{Digest: digest, PairHome: pairHome}, nil
    77	}
    78	
    79	func scanExisting(root string, manifest RuntimeManifest) (map[string]ExistingAsset, error) {
    80		existing := map[string]ExistingAsset{}
    81		for _, asset := range manifest.Assets {
    82			p := filepath.Join(root, filepath.FromSlash(asset.Path))
    83			info, err := os.Stat(p)
    84			if err != nil {
    85				if os.IsNotExist(err) {
    86					continue
    87				}
    88				return nil, err
    89			}
    90			if info.IsDir() {
    91				continue
    92			}
    93			data, err := os.ReadFile(p)
    94			if err != nil {
    95				return nil, err
    96			}
    97			h := sha256.Sum256(data)
    98			existing[asset.Path] = ExistingAsset{
    99				Mode:   uint32(info.Mode().Perm()),
   100				Size:   info.Size(),
   101				Digest: "sha256:" + hex.EncodeToString(h[:]),
   102			}
   103		}
   104		return existing, nil
   105	}
   106	
   107	func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
   108		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
   109			return err
   110		}
   111		tmp := path + ".tmp"
   112		if err := os.WriteFile(tmp, data, mode); err != nil {
   113			_ = os.Remove(tmp)
   114			return err
   115		}
   116		if err := os.Chmod(tmp, mode); err != nil {
   117			_ = os.Remove(tmp)
   118			return err
   119		}
   120		if err := os.Rename(tmp, path); err != nil {
   121			_ = os.Remove(tmp)
   122			return err
   123		}
   124		return nil
   125	}
   126	
   127	func writeMarker(path, digest string, assetCount int) error {
   128		data, err := json.MarshalIndent(runtimeMarker{
   129			Digest:     digest,
   130			AssetCount: assetCount,
   131			Generated:  time.Now().UTC().Format(time.RFC3339),
   132		}, "", "  ")
   133		if err != nil {
   134			return err
   135		}
   136		data = append(data, '\n')
   137		return writeFileAtomic(path, data, 0o644)
   138	}
   139	
   140	func applyCleanup(storeRoot, selectedDigest string, keep int) error {
   141		entries, err := os.ReadDir(storeRoot)
   142		if err != nil {
   143			if os.IsNotExist(err) {
   144				return nil
   145			}
   146			return err
   147		}
   148		gens := make([]RuntimeGeneration, 0, len(entries))
   149		for _, entry := range entries {
   150			if !entry.IsDir() {
   151				continue
   152			}
   153			name := entry.Name()
   154			info, err := entry.Info()
   155			if err != nil {
   156				return err
   157			}
   158			gens = append(gens, RuntimeGeneration{
   159				Digest:    name,
   160				HasMarker: markerValid(filepath.Join(storeRoot, name, "manifest.json")),
   161				ModUnix:   info.ModTime().Unix(),
   162			})
   163		}
   164		plan, err := PlanCleanup(CleanupInput{SelectedDigest: selectedDigest, Keep: keep, Generations: gens})
   165		if err != nil {
   166			return err
   167		}
   168		sort.Strings(plan.DeleteDigests)
   169		for _, digest := range plan.DeleteDigests {
   170			if err := os.RemoveAll(filepath.Join(storeRoot, digest)); err != nil {
   171				return err
   172			}
   173		}
   174		return nil
   175	}
   176	
   177	func markerValid(path string) bool {
   178		data, err := os.ReadFile(path)
   179		if err != nil {
   180			return false
   181		}
   182		var marker runtimeMarker
   183		if err := json.Unmarshal(data, &marker); err != nil {
   184			return false
   185		}
   186		return marker.Digest != ""
   187	}

 succeeded in 0ms:
     1	package runtimebundlegen
     2	
     3	import (
     4		"crypto/sha256"
     5		"encoding/hex"
     6		"encoding/json"
     7		"fmt"
     8		"io"
     9		"io/fs"
    10		"os"
    11		"path/filepath"
    12		"sort"
    13		"strings"
    14	)
    15	
    16	type RuntimeAsset struct {
    17		Path   string
    18		Mode   uint32
    19		Size   int64
    20		Digest string
    21	}
    22	
    23	type RuntimeManifest struct {
    24		Assets []RuntimeAsset
    25	}
    26	
    27	var explicitAssetPaths = []string{
    28		"bin/pair-shell",
    29		"bin/pair-help",
    30		"bin/pair-notify",
    31		"bin/pair-quit.sh",
    32		"bin/pair-restart.sh",
    33		"bin/pair-scrollback-open",
    34		"bin/pair-changelog-open",
    35		"bin/pair-review-open",
    36		"bin/pair-review-readiness",
    37		"bin/pair-review-target",
    38		"bin/copy-on-select.sh",
    39		"bin/clipboard-to-pane.sh",
    40		"bin/flash-pane.sh",
    41		"bin/pair-title.sh",
    42		"bin/pair-session-watch.sh",
    43		"bin/pair-wrap",
    44		"bin/pair-slug",
    45		"bin/pair-context",
    46		"bin/pair-scrollback-render",
    47		"bin/pair-changelog",
    48		"bin/pair-continuation",
    49		"bin/pair-session-watch",
    50		"doctor/README.md",
    51		"doctor/SKILL.md",
    52		"doctor/doctor.sh",
    53		"doctor/emitter-health.sh",
    54	}
    55	
    56	var assetDirs = []string{
    57		"bin/lib",
    58		"nvim",
    59		"zellij",
    60	}
    61	
    62	type GenerateOptions struct {
    63		RepoRoot string
    64		OutRoot  string
    65	}
    66	
    67	func Generate(opts GenerateOptions) (RuntimeManifest, error) {
    68		if opts.RepoRoot == "" {
    69			opts.RepoRoot = "."
    70		}
    71		if opts.OutRoot == "" {
    72			return RuntimeManifest{}, fmt.Errorf("output root is required")
    73		}
    74		repoRoot, err := filepath.Abs(opts.RepoRoot)
    75		if err != nil {
    76			return RuntimeManifest{}, err
    77		}
    78		outRoot, err := filepath.Abs(opts.OutRoot)
    79		if err != nil {
    80			return RuntimeManifest{}, err
    81		}
    82		outParent := filepath.Dir(outRoot)
    83		outBase := filepath.Base(outRoot)
    84		if err := os.MkdirAll(outParent, 0o755); err != nil {
    85			return RuntimeManifest{}, err
    86		}
    87		stageRoot, err := os.MkdirTemp(outParent, "."+outBase+"-tmp-")
    88		if err != nil {
    89			return RuntimeManifest{}, err
    90		}
    91		committed := false
    92		defer func() {
    93			if !committed {
    94				_ = os.RemoveAll(stageRoot)
    95			}
    96		}()
    97		filesRoot := filepath.Join(stageRoot, "files")
    98	
    99		paths := map[string]bool{}
   100		for _, p := range explicitAssetPaths {
   101			paths[p] = true
   102		}
   103		for _, dir := range assetDirs {
   104			root := filepath.Join(repoRoot, filepath.FromSlash(dir))
   105			if err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
   106				if err != nil {
   107					return err
   108				}
   109				if d.IsDir() {
   110					return nil
   111				}
   112				rel, err := filepath.Rel(repoRoot, p)
   113				if err != nil {
   114					return err
   115				}
   116				logical := filepath.ToSlash(rel)
   117				if shouldExclude(logical) {
   118					return nil
   119				}
   120				paths[logical] = true
   121				return nil
   122			}); err != nil {
   123				return RuntimeManifest{}, err
   124			}
   125		}
   126	
   127		ordered := make([]string, 0, len(paths))
   128		for p := range paths {
   129			if shouldExclude(p) {
   130				continue
   131			}
   132			ordered = append(ordered, p)
   133		}
   134		sort.Strings(ordered)
   135	
   136		manifest := RuntimeManifest{Assets: make([]RuntimeAsset, 0, len(ordered))}
   137		for _, logical := range ordered {
   138			src := filepath.Join(repoRoot, filepath.FromSlash(logical))
   139			info, err := os.Stat(src)
   140			if err != nil {
   141				return RuntimeManifest{}, fmt.Errorf("asset %s: %w", logical, err)
   142			}
   143			if info.IsDir() {
   144				continue
   145			}
   146			digest, err := copyAsset(src, filepath.Join(filesRoot, filepath.FromSlash(logical)), info.Mode().Perm())
   147			if err != nil {
   148				return RuntimeManifest{}, err
   149			}
   150			manifest.Assets = append(manifest.Assets, RuntimeAsset{
   151				Path:   logical,
   152				Mode:   uint32(info.Mode().Perm()),
   153				Size:   info.Size(),
   154				Digest: "sha256:" + digest,
   155			})
   156		}
   157		encoded, err := json.MarshalIndent(manifest, "", "  ")
   158		if err != nil {
   159			return RuntimeManifest{}, err
   160		}
   161		encoded = append(encoded, '\n')
   162		if err := os.WriteFile(filepath.Join(stageRoot, "manifest.json"), encoded, 0o644); err != nil {
   163			return RuntimeManifest{}, err
   164		}
   165		if err := os.RemoveAll(outRoot); err != nil {
   166			return RuntimeManifest{}, err
   167		}
   168		if err := os.Rename(stageRoot, outRoot); err != nil {
   169			return RuntimeManifest{}, err
   170		}
   171		committed = true
   172		return manifest, nil
   173	}
   174	
   175	func shouldExclude(logical string) bool {
   176		base := filepath.Base(logical)
   177		if base == ".DS_Store" || strings.Contains(logical, "__pycache__/") {
   178			return true
   179		}
   180		if strings.HasSuffix(logical, "_test.lua") {
   181			return true
   182		}
   183		switch logical {
   184		case "bin/pair", "bin/pair-go", "bin/pair-dev":
   185			return true
   186		}
   187		return false
   188	}
   189	
   190	func copyAsset(src, dst string, mode os.FileMode) (string, error) {
   191		in, err := os.Open(src)
   192		if err != nil {
   193			return "", err
   194		}
   195		defer in.Close()
   196		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
   197			return "", err
   198		}
   199		tmp := dst + ".tmp"
   200		out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
   201		if err != nil {
   202			return "", err
   203		}
   204		h := sha256.New()
   205		_, copyErr := io.Copy(io.MultiWriter(out, h), in)
   206		closeErr := out.Close()
   207		if copyErr != nil {
   208			_ = os.Remove(tmp)
   209			return "", copyErr
   210		}
   211		if closeErr != nil {
   212			_ = os.Remove(tmp)
   213			return "", closeErr
   214		}
   215		if err := os.Chmod(tmp, mode); err != nil {
   216			_ = os.Remove(tmp)
   217			return "", err
   218		}
   219		if err := os.Rename(tmp, dst); err != nil {
   220			_ = os.Remove(tmp)
   221			return "", err
   222		}
   223		return hex.EncodeToString(h.Sum(nil)), nil
   224	}

exec
/bin/zsh -lc "rg -n \"func ResolveDataDir|PAIR_DATA_DIR|XDG_DATA_HOME\" cmd/internal tests bin -g'*.go' -g'*.sh'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
bin/copy-on-select.sh:15:#      $PAIR_DATA_DIR/quote-<tag> and triggers nvim's PairPasteQuote() to
tests/pair-rename.sh:4:# Builds a fixtured $PAIR_DATA_DIR with representative tag-scoped files,
tests/pair-rename.sh:68:# Run `pair rename` in an isolated $PAIR_DATA_DIR with the zellij shim.
tests/pair-rename.sh:75:    PAIR_DATA_DIR="$dd" PATH="$shim:$PATH" "$PAIR_BIN" rename "$@"
bin/clipboard-to-pane.sh:15:# temp file at $PAIR_DATA_DIR/quote-<tag> and then triggering
bin/clipboard-to-pane.sh:75:data_dir="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
tests/cmux-ownership-test.sh:12:# fakes (HOME/XDG_DATA_HOME pinned, fake `cmux` + `zellij` on PATH), exactly
tests/cmux-ownership-test.sh:53:    env HOME="$RT" XDG_DATA_HOME="$RT/xdg" PATH="$RT/bin:$PATH" \
tests/cmux-ownership-test.sh:94:env -u CMUX_WORKSPACE_ID HOME="$RT" XDG_DATA_HOME="$RT/xdg" PATH="$RT/bin:$PATH" \
tests/changelog-notify-test.sh:27:local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
tests/changelog-notify-test.sh:59:  local dd = os.getenv('PAIR_DATA_DIR')
tests/changelog-notify-test.sh:94:  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
bin/lib/adapt-log.sh:5:# $PAIR_DATA_DIR/adapt-<tag>.jsonl, byte-identical in schema + field order to
bin/lib/adapt-log.sh:20:    local dir="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
tests/review-loop-test.sh:119:    XDG_DATA_HOME="$RT/xdg" DOCFLOW_BIN="$ROOT/tests/lib/fake-docflow.sh" \
tests/pair-continue-test.sh:43:# XDG_DATA_HOME is pinned into the sandbox so the probe never reads/writes the
tests/pair-continue-test.sh:47:  ( cd "$RT" && XDG_DATA_HOME="$RT/xdg" PAIR_DEBUG_ARGS=1 "$PAIR" "$@" 2>/dev/null ) \
tests/pair-continue-test.sh:134:  env HOME="$CRT" XDG_DATA_HOME="$CRT/xdg" PAIR_TAG=demo PAIR_AGENT=claude PAIR_KILL_CMD=true "$@"
tests/pair-continue-test.sh:162:( cd "$CRT" && HOME="$CRT" XDG_DATA_HOME="$CRT/xdg" \
tests/pair-continue-test.sh:176:( cd "$CRT" && HOME="$CRT" XDG_DATA_HOME="$CRT/xdg" SESSION=pair-demo \
tests/pair-continue-test.sh:209:  env HOME="$HRT" XDG_DATA_HOME="$HRT/xdg" PATH="$HRT/bin:$PATH" \
bin/pair-title.sh:20:#   - nvim draft: $PAIR_DATA_DIR/draft-<tag>.md
bin/pair-title.sh:47:DATA_DIR="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
tests/pair-session-watch-test.sh:33:PAIR_DATA_DIR="$RT/data" \
bin/pair-restart.sh:68:DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
cmd/internal/contextcmd/contextcmd.go:24:		XDGDataHome: os.Getenv("XDG_DATA_HOME"),
cmd/internal/contextcmd/contextcmd.go:25:		PairDataDir: os.Getenv("PAIR_DATA_DIR"),
tests/changelog-open-test.sh:23:export PAIR_DATA_DIR="$tmp/data"
tests/changelog-open-test.sh:26:mkdir -p "$PAIR_DATA_DIR"
tests/changelog-open-test.sh:30:    > "$PAIR_DATA_DIR/scrollback-t-claude.raw"
tests/changelog-open-test.sh:32:    > "$PAIR_DATA_DIR/scrollback-t-claude.events.jsonl"
tests/changelog-open-test.sh:53:DLOCK="$PAIR_DATA_DIR/changelog-t-claude.distill.lock"
tests/changelog-open-test.sh:54:LOG="$PAIR_DATA_DIR/changelog-t-claude.md"
tests/changelog-open-test.sh:55:ANCHOR="$PAIR_DATA_DIR/changelog-t-claude.anchor"
tests/changelog-open-test.sh:70:[ -f "$PAIR_DATA_DIR/changelog-t-claude.openlock" ] && { echo "FAIL: openlock not cleared on viewer exit"; fail=1; }
tests/adapt-schema-test.sh:26:export PAIR_TAG=golden PAIR_DATA_DIR="$RT" PAIR_AGENT=codex
tests/lib/fake-review-agent.sh:14:# Runs in the doc's repo (cwd), with DOCFLOW_BIN + XDG_DATA_HOME from the caller.
tests/lib/fake-review-agent.sh:19:dir="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
tests/review-readiness-cli-test.sh:38:prep_out="$(PAIR_HOME="$ROOT" PAIR_DATA_DIR="$RT" PAIR_TAG=prep PAIR_SESSION_ID=sid \
tests/review-toggle-test.sh:64:local sf = vim.env.PAIR_DATA_DIR .. '/review-' .. vim.env.PAIR_TAG .. '.open'
tests/review-toggle-test.sh:89:local target = vim.env.PAIR_DATA_DIR .. '/review-target-' .. vim.env.PAIR_TAG .. '.json'
tests/review-toggle-test.sh:90:local draft = vim.env.PAIR_DATA_DIR .. '/draft.md' -- exists (the test wrote it)
tests/review-toggle-test.sh:97:local prepbin = vim.env.PAIR_DATA_DIR .. '/prep-ok'
tests/review-toggle-test.sh:125:os.remove(vim.env.PAIR_DATA_DIR .. '/config-' .. vim.env.PAIR_TAG .. '-claude.json')
tests/review-toggle-test.sh:138:  vim.env.PAIR_DATA_DIR .. '/config-' .. vim.env.PAIR_TAG .. '-' .. vim.env.PAIR_AGENT .. '.json')
tests/review-toggle-test.sh:149:os.remove(vim.env.PAIR_DATA_DIR .. '/config-' .. vim.env.PAIR_TAG .. '-codex.json')
tests/review-toggle-test.sh:150:vim.fn.writefile({ '111' }, vim.env.PAIR_DATA_DIR .. '/agent-pid-' .. vim.env.PAIR_TAG)
tests/review-toggle-test.sh:203:    PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" PAIR_SESSION_ID=testsid \
tests/queue-send-test.sh:46:    env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/queue-send-test.sh:55:local dd = os.getenv('PAIR_DATA_DIR')
tests/cr-newline-test.sh:36:local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
tests/cr-newline-test.sh:76:  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/review-window-test.sh:43:( cd "$RT" && PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" RESULT="$RT/r1" \
tests/review-window-test.sh:48:if PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-open" "$RT/nope.md" 2>/dev/null; then
tests/review-window-test.sh:54:PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-open" "$RT/doc.md" || true
tests/review-window-test.sh:81:( cd "$REPO" && PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/review-window-test.sh:323:( cd "$REPO" && PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/statusline-pos-test.sh:33:local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
tests/statusline-pos-test.sh:68:  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/pair-embedded-runtime-test.sh:76:export XDG_DATA_HOME="$xdg"
tests/pair-embedded-runtime-test.sh:81:unset PAIR_DEV PAIR_HOME PAIR_TAG PAIR_AGENT PAIR_AGENT_ARGS PAIR_DATA_DIR ZELLIJ_SESSION_NAME ZELLIJ ZELLIJ_PANE_ID
tests/pair-title-poller-test.sh:117:    PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$DD" RENAME_LOG="$RT/rename.log" FAKE_COUNT="$count" \
tests/autopair-test.sh:31:local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
tests/autopair-test.sh:69:  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/changelog-session-key-test.sh:11:export PAIR_DATA_DIR="$tmp/data" PAIR_TAG=t PAIR_AGENT=claude
tests/changelog-session-key-test.sh:12:mkdir -p "$PAIR_DATA_DIR"
tests/changelog-session-key-test.sh:37:printf 'old log\n' > "$PAIR_DATA_DIR/changelog-t-claude-$A.md"
tests/changelog-session-key-test.sh:41:[ -s "$PAIR_DATA_DIR/changelog-t-claude-$B.md" ] \
tests/changelog-session-key-test.sh:46:grep -q 'old log' "$PAIR_DATA_DIR/changelog-t-claude-$A.md" \
tests/changelog-session-key-test.sh:52:  > "$PAIR_DATA_DIR/config-t-claude.json"
tests/changelog-session-key-test.sh:58:rm -f "$PAIR_DATA_DIR/config-t-claude.json"
tests/review-handoff-test.sh:4:# the file. XDG_DATA_HOME is redirected so the handoff lands in a temp dir.
tests/review-handoff-test.sh:53:PAIR_ROOT="$ROOT" RESULT="$RESULT" XDG_DATA_HOME="$RT/xdg" \
tests/zellij-trace-test.sh:43:PAIR_DATA_DIR="$RT" \
tests/review-indicator-test.sh:54:PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" \
tests/review-poke-test.sh:49:PATH="$RT/bin:$PATH" PAIR_ROOT="$ROOT" RESULT="$RESULT" PAIR_DATA_DIR="$RT" PAIR_TAG=poke \
tests/pair-review-target-test.sh:14:PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=codex \
tests/pair-review-target-test.sh:20:PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=codex \
cmd/internal/dispatcher/dispatcher_test.go:101:	t.Setenv("PAIR_DATA_DIR", data)
cmd/internal/adapt/adapt_test.go:141:	t.Setenv("PAIR_DATA_DIR", dir)
cmd/internal/adapt/adapt.go:3:// $PAIR_DATA_DIR/adapt-<tag>.jsonl.
cmd/internal/adapt/adapt.go:78:// DataDir returns $PAIR_DATA_DIR or the XDG default. This is the canonical
cmd/internal/adapt/adapt.go:82:	if d := os.Getenv("PAIR_DATA_DIR"); d != "" {
cmd/internal/adapt/adapt.go:85:	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
cmd/internal/launcher/datadir.go:6:func ResolveDataDir(home, xdgDataHome string) string {

exec
/bin/zsh -lc "nl -ba cmd/internal/launcher/datadir.go | sed -n '1,80p'; nl -ba bin/pair-shell | sed -n '1,120p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package launcher
     2	
     3	import "path/filepath"
     4	
     5	// ResolveDataDir returns Pair's data directory from explicit environment values.
     6	func ResolveDataDir(home, xdgDataHome string) string {
     7		if xdgDataHome != "" {
     8			return filepath.Join(xdgDataHome, "pair")
     9		}
    10		return filepath.Join(home, ".local", "share", "pair")
    11	}
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

exec
/bin/zsh -lc 'rg -n "PAIR_DATA_DIR|DATA_DIR|XDG_DATA_HOME" bin/pair-shell' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
94:                                $PAIR_DATA_DIR from <old> to <new>.
220:  ${XDG_DATA_HOME:-~/.local/share}/pair/draft-<tag>.md     persistent draft (*)
221:  ${XDG_DATA_HOME:-~/.local/share}/pair/log-<tag>.md       prompt history (-N)
222:  ${XDG_DATA_HOME:-~/.local/share}/pair/queue-<tag>/*.md   queued prompts (+N)
243:        list_data_dir="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
309:        # $PAIR_DATA_DIR from <old> to <new>. Offline-only: refuses if
369:        rename_dd="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
679:# --- #55 compaction support: hoisted DATA_DIR + early helpers ----------------
680:# DATA_DIR is hoisted above the `command -v zellij` gate, the in-session
683:# Honors $XDG_DATA_HOME with the spec-defined fallback. (Original site below
685:DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
686:mkdir -p "$DATA_DIR"
687:export PAIR_DATA_DIR="$DATA_DIR"
698:    local base="$DATA_DIR/scrollback-${tag}-${agent}"
702:    pbase="$DATA_DIR/parked-scrollback-${tag}-${ts}"
706:    : > "$DATA_DIR/parked-${tag}"
849:    local canonical="$DATA_DIR/config-${tag}-${agent}.json"
851:        local legacy="$DATA_DIR/config-${tag}-codex-codex.json"
891:# Ownership model: the owner file at $DATA_DIR/cmux-owner-$CMUX_WORKSPACE_ID
904:    local owner_file="$DATA_DIR/cmux-owner-${CMUX_WORKSPACE_ID}"
912:    local owner_file="$DATA_DIR/cmux-owner-${CMUX_WORKSPACE_ID}"
937:# Defer the [-z AGENT] default to after DATA_DIR is set: when `resume <tag>`
994:    if [ -f "$DATA_DIR/agent-${forced_tag}" ]; then
995:        AGENT=$(cat "$DATA_DIR/agent-${forced_tag}")
997:        for cfg in "$DATA_DIR"/config-"${forced_tag}"-*.json; do
1070:# DATA_DIR / PAIR_DATA_DIR are set above (hoisted for #55, just after
1095:    # var at $DATA_DIR/nvim-pid-<tag>-<kind>.
1097:        pf="$DATA_DIR/nvim-pid-${tag}-${kind}"
1108:    # DATA_DIR file so unrelated nvim --embed instances are untouched.
1110:    pkill -9 -f "nvim --embed.*${DATA_DIR}/draft-${tag}\.md\$" >/dev/null 2>&1 || true
1111:    pkill -9 -f "nvim --embed.*${DATA_DIR}/scrollback-${tag}-" >/dev/null 2>&1 || true
1121:        for pf in "$DATA_DIR"/nvim-pid-*-draft "$DATA_DIR"/nvim-pid-*-scrollback; do
1135:                *nvim*--embed*"$DATA_DIR/draft-"*)
1136:                    tag=${argv##*"$DATA_DIR/draft-"}
1141:                *nvim*--embed*"$DATA_DIR/scrollback-"*)
1142:                    tag=${argv##*"$DATA_DIR/scrollback-"}
1207:        new="$DATA_DIR/${base#pair-}"
1213:        echo "pair: migrated $moved file(s) from ~/scratch/ to $DATA_DIR" >&2
1242:# never runs and these embeds accumulate. Definition near $DATA_DIR setup.
1283:    for f in "$DATA_DIR"/draft-*.md "$DATA_DIR"/log-*.md; do
1336:    local tag="$1" qdir="$DATA_DIR/queue-$tag" n=0 f
1414:    echo "  data_dir       = $DATA_DIR"
1527:        quit_agent=$(cat "$DATA_DIR/agent-${PAIR_TAG}" 2>/dev/null || echo "$AGENT")
1536:        # the orphan-reaping block near $DATA_DIR setup for the full story
1545:        _sb_base="$DATA_DIR/scrollback-${PAIR_TAG}-${quit_agent}"
1583:        rm -f "$DATA_DIR/outer-tty-${PAIR_TAG}" "$DATA_DIR/agent-${PAIR_TAG}" \
1584:              "$DATA_DIR/agent-output-${PAIR_TAG}" \
1586:              "$DATA_DIR/pair-wrap-pid-${PAIR_TAG}" \
1587:              "$DATA_DIR/adapt-${PAIR_TAG}.jsonl" \
1588:              "$DATA_DIR/image-capture-${PAIR_TAG}" \
1589:              "$DATA_DIR/image-capture-${PAIR_TAG}.done"
1621:        local poller_pid_file="$DATA_DIR/title-pid-${PAIR_TAG}"
1645:            rm -f "$DATA_DIR/cmux-owner-${CMUX_WORKSPACE_ID}"
1679:        /dev/*) printf '%s\n' "$outer" > "$DATA_DIR/outer-tty-${PAIR_TAG}" ;;
1680:        *)      rm -f "$DATA_DIR/outer-tty-${PAIR_TAG}" ;;
1704:    DRAFT="$DATA_DIR/draft-${PAIR_TAG}.md"
1715:    title_agent=$(cat "$DATA_DIR/agent-$PAIR_TAG" 2>/dev/null | tr -d '\r\n[:space:]')
2022:DRAFT="$DATA_DIR/draft-${PAIR_TAG}.md"
2038:printf '%s\n' "$AGENT" > "$DATA_DIR/agent-${PAIR_TAG}"
2226:: > "$DATA_DIR/adapt-${PAIR_TAG}.jsonl" 2>/dev/null || true

codex
One concrete mismatch is emerging: the plan names `PAIR_DATA_DIR` as the runtime-store root, but the Go fallback currently derives only from `HOME`/`XDG_DATA_HOME`. I’m continuing the sweep to see whether that is isolated or part of a wider contract drift.
exec
/bin/zsh -lc "nl -ba tests/pair-embedded-runtime-test.sh | sed -n '1,170p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	#!/usr/bin/env bash
     2	set -euo pipefail
     3	
     4	repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
     5	tmp="$(mktemp -d "${TMPDIR:-/tmp}/pair-embedded-runtime.XXXXXX")"
     6	trap 'rm -rf "$tmp"' EXIT
     7	
     8	bin_dir="$tmp/bin"
     9	home="$tmp/home"
    10	xdg="$tmp/xdg"
    11	mkdir -p "$bin_dir" "$home" "$xdg"
    12	gomodcache="$(go env GOMODCACHE)"
    13	gocache="$(go env GOCACHE)"
    14	
    15	make -C "$repo_root" runtimebundle-generate >/dev/null
    16	go build -o "$bin_dir/pair" "$repo_root/cmd/pair-go"
    17	
    18	cat >"$bin_dir/zellij" <<'SH'
    19	#!/usr/bin/env bash
    20	set -eu
    21	printf '%s\n' "$*" >> "${ZELLIJ_LOG:?}"
    22	case "$*" in
    23	  "list-sessions --no-formatting"|"list-sessions --short")
    24	    exit 0
    25	    ;;
    26	  --session*" action list-clients")
    27	    exit 0
    28	    ;;
    29	  --config-dir*)
    30	    config=""
    31	    layout=""
    32	    prev=""
    33	    for arg in "$@"; do
    34	      if [ "$prev" = "--config-dir" ]; then config="$arg"; fi
    35	      if [ "$prev" = "--new-session-with-layout" ]; then layout="$arg"; fi
    36	      prev="$arg"
    37	    done
    38	    test -f "$config/config.kdl"
    39	    test -f "$layout"
    40	    case "$config" in */pair/runtime/*/pair-home/zellij) ;; *) printf 'bad config path: %s\n' "$config" >&2; exit 11 ;; esac
    41	    case "$layout" in */pair/runtime/*/pair-home/zellij/layouts/main.kdl) ;; *) printf 'bad layout path: %s\n' "$layout" >&2; exit 12 ;; esac
    42	    root="${config%/zellij}"
    43	    test -x "$root/bin/pair-shell"
    44	    test -x "$root/bin/pair-wrap"
    45	    test -x "$root/bin/pair-session-watch.sh"
    46	    test -x "$root/bin/pair-title.sh"
    47	    test -f "$root/nvim/init.lua"
    48	    printf '%s\n' "$root" > "${PAIR_SMOKE_ROOT:?}"
    49	    exit 0
    50	    ;;
    51	  *)
    52	    exit 0
    53	    ;;
    54	esac
    55	SH
    56	chmod +x "$bin_dir/zellij"
    57	
    58	cat >"$bin_dir/ps" <<'SH'
    59	#!/usr/bin/env bash
    60	case "$*" in
    61	  "-o comm= -p "*)
    62	    printf 'sh\n'
    63	    ;;
    64	  "-o ppid= -p "*)
    65	    printf '1\n'
    66	    ;;
    67	  *)
    68	    exec /bin/ps "$@"
    69	    ;;
    70	esac
    71	SH
    72	chmod +x "$bin_dir/ps"
    73	
    74	export PATH="$bin_dir:$PATH"
    75	export HOME="$home"
    76	export XDG_DATA_HOME="$xdg"
    77	export GOMODCACHE="$gomodcache"
    78	export GOCACHE="$gocache"
    79	export ZELLIJ_LOG="$tmp/zellij.log"
    80	export PAIR_SMOKE_ROOT="$tmp/root"
    81	unset PAIR_DEV PAIR_HOME PAIR_TAG PAIR_AGENT PAIR_AGENT_ARGS PAIR_DATA_DIR ZELLIJ_SESSION_NAME ZELLIJ ZELLIJ_PANE_ID
    82	
    83	help_out="$("$bin_dir/pair" --help)"
    84	case "$help_out" in
    85	  pair\ —*) ;;
    86	  *)
    87	    printf 'copied pair --help did not reach embedded pair-shell help; first bytes:\n%s\n' "$help_out" >&2
    88	    exit 1
    89	    ;;
    90	esac
    91	
    92	mkdir -p "$xdg/pair/runtime/aaaaaaaa/pair-home" \
    93	         "$xdg/pair/runtime/bbbbbbbb/pair-home" \
    94	         "$xdg/pair/runtime/cccccccc/pair-home"
    95	printf '{"digest":"aaaaaaaa","asset_count":0}\n' > "$xdg/pair/runtime/aaaaaaaa/manifest.json"
    96	printf '{"digest":"bbbbbbbb","asset_count":0}\n' > "$xdg/pair/runtime/bbbbbbbb/manifest.json"
    97	printf '{"digest":"cccccccc","asset_count":0}\n' > "$xdg/pair/runtime/cccccccc/manifest.json"
    98	touch -t 202001010000 "$xdg/pair/runtime/aaaaaaaa"
    99	touch -t 202001020000 "$xdg/pair/runtime/bbbbbbbb"
   100	touch -t 202001030000 "$xdg/pair/runtime/cccccccc"
   101	
   102	"$bin_dir/pair" resume smoke >/dev/null
   103	
   104	test -s "$PAIR_SMOKE_ROOT"
   105	root="$(cat "$PAIR_SMOKE_ROOT")"
   106	case "$root" in */pair/runtime/*/pair-home) ;; *) printf 'bad extracted root: %s\n' "$root" >&2; exit 1 ;; esac
   107	test -d "$root"
   108	test ! -e "$xdg/pair/runtime/aaaaaaaa"
   109	test -d "$xdg/pair/runtime/bbbbbbbb"
   110	test -d "$xdg/pair/runtime/cccccccc"
   111	
   112	printf 'pair embedded runtime smoke passed\n'

exec
/bin/zsh -lc "nl -ba Makefile.local | sed -n '1,330p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	# Repo-specific Makefile targets.
     2	# This file is included by Makefile — add your own targets here.
     3	
     4	.PHONY: build install pair pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-session-watch pair-go runtimebundle-generate runtimebundle-drift-check test-runtimebundle test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-pair-embedded-runtime test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
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
    32	RUNTIMEBUNDLE_HELPERS := pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-session-watch
    33	RUNTIMEBUNDLE_ASSETS := cmd/internal/runtimebundle/assets/runtime/manifest.json
    34	
    35	build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
    36	
    37	install: build
    38		@mkdir -p $(INSTALL_DIR)
    39		@for b in $(GO_BINS); do \
    40		    install -m 755 $(BIN_DIR)/$$b $(INSTALL_DIR)/$$b && \
    41		    echo "installed: $(INSTALL_DIR)/$$b"; \
    42		done
    43		@for b in $(SHELL_BINS); do \
    44		    ln -sf "$(CURDIR)/$(BIN_DIR)/$$b" "$(INSTALL_DIR)/$$b" && \
    45		    echo "linked: $(INSTALL_DIR)/$$b -> $(CURDIR)/$(BIN_DIR)/$$b"; \
    46		done
    47	
    48	# Per-binary aliases — `make pair-scribe`, etc.
    49	pair:                   $(BIN_DIR)/pair
    50	pair-scribe:            $(BIN_DIR)/pair-scribe
    51	pair-scrollback-render: $(BIN_DIR)/pair-scrollback-render
    52	pair-wrap:              $(BIN_DIR)/pair-wrap
    53	pair-slug:              $(BIN_DIR)/pair-slug
    54	pair-continuation:      $(BIN_DIR)/pair-continuation
    55	pair-changelog:         $(BIN_DIR)/pair-changelog
    56	pair-context:           $(BIN_DIR)/pair-context
    57	pair-session-watch:     $(BIN_DIR)/pair-session-watch
    58	pair-go:                $(BIN_DIR)/pair-go
    59	
    60	# Tests
    61	#   make test       → go test ./... (cache disabled so reruns actually run)
    62	#   make test-race  → race-detector run, scoped to packages where the
    63	#                     suite has actual concurrent code worth checking.
    64	#                     pair-wrap has translateStdin's goroutine + timer +
    65	#                     channels; pair-scrollback-render's render() spawns
    66	#                     a goroutine drainer that races with em.Close()
    67	#                     inside the vt.Emulator library API itself, which is
    68	#                     outside our scope to fix — exclude that package
    69	#                     here rather than chase a library-level race.
    70	test: test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-pair-embedded-runtime test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace test-review
    71		go test ./... -count=1
    72	
    73	# pair-title poller: single-instance guard (identity-checked liveness so a
    74	# recycled PID can't suppress the respawn) + the #71 frame meter (renames each
    75	# agent pane to "<agent> (<count>) [<cwd>]", with unchanged-skip). See
    76	# tests/pair-title-poller-test.sh.
    77	test-pair-title:
    78		bash tests/pair-title-poller-test.sh
    79	
    80	# cmux workspace ownership claim (presence beats a stale owner flag), so a
    81	# launch/attach/restart reclaims its workspace title. See
    82	# tests/cmux-ownership-test.sh.
    83	test-cmux-ownership: $(BIN_DIR)/pair
    84		bash tests/cmux-ownership-test.sh
    85	
    86	# Session id watcher must ignore stale pidfiles from prior launches and wait
    87	# for pair-wrap to publish the current agent PID.
    88	test-session-watch: $(BIN_DIR)/pair-session-watch
    89		bash tests/pair-session-watch-test.sh
    90	
    91	runtimebundle-generate: $(addprefix $(BIN_DIR)/,$(RUNTIMEBUNDLE_HELPERS))
    92		go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime
    93	
    94	$(RUNTIMEBUNDLE_ASSETS): runtimebundle-generate
    95	
    96	runtimebundle-drift-check: $(addprefix $(BIN_DIR)/,$(RUNTIMEBUNDLE_HELPERS))
    97		@d1="$$(mktemp -d)"; d2="$$(mktemp -d)"; \
    98		trap 'rm -rf "$$d1" "$$d2"' EXIT; \
    99		go run ./cmd/internal/runtimebundle/generatecmd --repo . --out "$$d1/runtime" && \
   100		go run ./cmd/internal/runtimebundle/generatecmd --repo . --out "$$d2/runtime" && \
   101		diff -qr "$$d1/runtime" "$$d2/runtime" >/dev/null
   102	
   103	test-runtimebundle: runtimebundle-generate
   104		go test ./cmd/internal/runtimebundle -count=1
   105	
   106	# Review workbench (#66): the integration shell tests. The pure modules
   107	# (record, reconstruct) run under `make test-lua`.
   108	test-review:
   109		bash tests/review-docflow-test.sh
   110		bash tests/review-apply-test.sh
   111		bash tests/review-handoff-test.sh
   112		bash tests/review-loop-test.sh
   113		bash tests/review-projection-test.sh
   114		bash tests/review-poke-test.sh
   115		bash tests/review-window-test.sh
   116		bash tests/review-toggle-test.sh
   117		bash tests/pair-review-target-test.sh
   118		bash tests/review-readiness-cli-test.sh
   119		bash tests/review-indicator-test.sh
   120		bash tests/review-resume-test.sh
   121	
   122	test-race:
   123		go test -count=1 -race ./cmd/pair-wrap/
   124	
   125	# Self-test for the headless-nvim timeout watchdog (tests/lib/run-headless.sh,
   126	# #60). Runs FIRST in `test` so the safety net every headless-nvim sub-target
   127	# now relies on is proven before those targets run — without it, a driver that
   128	# fails to quit (bare `qall` on a modified buffer → E37) hangs the whole suite.
   129	test-run-headless:
   130		bash tests/run-headless-test.sh
   131	
   132	# Headless Lua tests for nvim/*.lua decision modules (issue #000027). nvim -l
   133	# runs a Lua script in nvim's interpreter with the full vim.api available, so
   134	# pure logic AND buffer-manipulation can be asserted without a UI.
   135	test-lua:
   136		nvim -l nvim/slug_test.lua
   137		nvim -l nvim/scrollback_test.lua
   138		nvim -l nvim/annotate_test.lua
   139		nvim -l nvim/changelog_test.lua
   140		nvim -l nvim/adapt_test.lua
   141		nvim -l nvim/doctor_test.lua
   142		nvim -l nvim/review/record_test.lua
   143		nvim -l nvim/review/reconstruct_test.lua
   144		nvim -l nvim/review/markers_test.lua
   145		nvim -l nvim/review/seam_test.lua
   146		nvim -l nvim/review/mode_test.lua
   147		nvim -l nvim/review/poke_bodies_test.lua
   148		nvim -l nvim/review/wrap_test.lua
   149		nvim -l nvim/review/readiness_test.lua
   150		nvim -l nvim/review/spinner_test.lua
   151		nvim -l nvim/review/resolve_test.lua
   152		nvim -l nvim/review/menu_test.lua
   153	
   154	# Integration test for nvim/init.lua's send/queue/history state machine.
   155	# Drives the real init.lua under headless nvim (it can't be unit-tested —
   156	# monolithic config, all-local fns). Guards the send-from-queue duplication
   157	# regression. See tests/queue-send-test.sh.
   158	test-queue:
   159		bash tests/queue-send-test.sh
   160	
   161	# Diagnostic trace for pair-originated zellij action calls (#68).
   162	test-zellij-trace:
   163		bash tests/zellij-trace-test.sh
   164	
   165	# Regression test for the autopair next-char gate (nvim/init.lua,
   166	# pair_insert_open). Drives the real init.lua headlessly and asserts each
   167	# opener's expr output. See tests/autopair-test.sh.
   168	test-autopair:
   169		bash tests/autopair-test.sh
   170	
   171	# Regression test for the insert-mode <CR> decision (nvim/init.lua, cr_keys /
   172	# _G.PairCRKeys, #65): Return must always insert a newline when no completion
   173	# is selected, dismissing the popup. Asserts the expr output for all three
   174	# popup states. See tests/cr-newline-test.sh.
   175	test-cr:
   176		bash tests/cr-newline-test.sh
   177	
   178	# Regression tests for the draft statusline (nvim/init.lua, _G.PairStatusline).
   179	# Drive the real init.lua headlessly: statusline-pos asserts the *, -N, +N nav
   180	# marker; changelog-notify asserts the ephemeral "change log ready" flash + its
   181	# fs_event marker watcher (#58).
   182	test-statusline:
   183		bash tests/statusline-pos-test.sh
   184		bash tests/changelog-notify-test.sh
   185	
   186	# Regression test for doctor/doctor.sh (the adaptation flight-recorder reader):
   187	# a malformed line must not abort the diagnostic. See doctor/doctor_test.sh.
   188	test-doctor:
   189		bash doctor/doctor_test.sh
   190	
   191	# copy-on-select in_nvim detection: the agent overwrites its pane title with
   192	# "claude [<cwd>]", so a repo path containing "nvim" (e.g. parley.nvim) must not
   193	# be mistaken for the nvim draft pane (which would skip the paste). Keys on
   194	# terminal_command, not the cwd-polluted title. See tests/copy-on-select-test.sh.
   195	test-copy-on-select:
   196		bash tests/copy-on-select-test.sh
   197	
   198	# Cross-emitter contract for the flight recorder: Go == shell == Lua schema,
   199	# plus concurrent-append line-atomicity. See tests/adapt-schema-test.sh.
   200	test-adapt-schema:
   201		bash tests/adapt-schema-test.sh
   202	
   203	# pair-dev rebuild hook (#000046): deployed mode (PAIR_DEV unset) must invoke
   204	# no toolchain; dev mode builds; build failure is errexit-safe. See
   205	# tests/dev-rebuild-test.sh.
   206	test-dev-rebuild:
   207		bash tests/dev-rebuild-test.sh
   208	
   209	# Go public entrypoint install layout (#79): installed pair is a regular Go
   210	# binary, pair-go launch shares its pair-shell handoff, and copied local
   211	# installs fall back to the build-time source root for adjacent assets.
   212	test-pair-go-install-layout:
   213		bash tests/pair-go-install-layout-test.sh
   214	
   215	test-pair-embedded-runtime:
   216		bash tests/pair-embedded-runtime-test.sh
   217	
   218	# pair-doctor emitter-health probe (#000047): marker check + binary selection
   219	# (running-via-pidfile vs PATH). See tests/emitter-health-test.sh.
   220	test-emitter-health:
   221		bash tests/emitter-health-test.sh
   222	
   223	# `pair continue` arg-parsing + session-name guard (#000054): tag not forced,
   224	# -- args forwarded, [agent] port, bare list / error paths, guard short-vs-long.
   225	# Drives the real bin/pair via its PAIR_DEBUG_ARGS probe. See tests/pair-continue-test.sh.
   226	test-continue: $(BIN_DIR)/pair
   227		bash tests/pair-continue-test.sh
   228	
   229	# changelog — end-to-end smoke for bin/pair-changelog-open (#53 M2): fakes the
   230	# scrollback capture + model + nvim, asserts clean → distill → write → open.
   231	# Depends on the binaries so `make test` builds + runs it (the script SKIPs if
   232	# they're absent, which would silently no-op in CI without these prereqs).
   233	test-changelog: $(BIN_DIR)/pair-changelog $(BIN_DIR)/pair-scrollback-render
   234		sh tests/changelog-open-test.sh
   235		sh tests/changelog-session-key-test.sh
   236	
   237	# pair-scribe — Go-based pty wrapper with SIGUSR1/SIGUSR2 pause/resume
   238	# of the typescript log. Replaces script(1) at the top of the user's
   239	# shell, so preexec can stop the log from filling with TUI redraw bytes
   240	# (claude, nvim, etc). See cmd/pair-scribe/README.md for the full why.
   241	$(BIN_DIR)/pair-scribe: cmd/pair-scribe/main.go go.mod
   242		go build -o $@ ./cmd/pair-scribe
   243	
   244	# pair-scrollback-render — replays a pair-wrap raw PTY capture through
   245	# charmbracelet/x/vt and emits one ANSI-styled line per logical scrollback
   246	# row, for the Alt+/ viewer to display through nvim.
   247	$(BIN_DIR)/pair-scrollback-render: cmd/pair-scrollback-render/main.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
   248		go build -o $@ ./cmd/pair-scrollback-render
   249	
   250	# pair-wrap — transparent PTY proxy around a TUI coding agent: forwards
   251	# stdio, tees scrollback log, extracts colored spans for nvim autocomplete,
   252	# detects OSC/BEL and forwards as OSC 9 to the recorded outer-TTY, handles
   253	# SIGUSR1 image-paste capture window.
   254	$(BIN_DIR)/pair-wrap: cmd/pair-wrap/main.go go.mod
   255		go build -o $@ ./cmd/pair-wrap
   256	
   257	$(BIN_DIR)/pair-slug: cmd/pair-slug/main.go cmd/pair-slug/slug.go cmd/internal/model/model.go cmd/internal/transcript/transcript.go go.mod
   258		go build -o $@ ./cmd/pair-slug
   259	
   260	# pair-context — one-shot: print a pane's current context size (token count)
   261	# for the pair-title poller. Pure ctxmeter reader + shared transcript resolver.
   262	$(BIN_DIR)/pair-context: cmd/pair-context/main.go cmd/internal/contextcmd/contextcmd.go cmd/internal/ctxmeter/ctxmeter.go cmd/internal/transcript/transcript.go go.mod
   263		go build -o $@ ./cmd/pair-context
   264	
   265	# pair-session-watch — async codex/agy session-id discovery. Legacy
   266	# bin/pair-session-watch.sh remains as a compatibility shim for bin/pair.
   267	$(BIN_DIR)/pair-session-watch: cmd/pair-session-watch/main.go cmd/internal/sessionwatch/sessionwatch.go cmd/internal/sessionwatch/run.go cmd/internal/sessionwatch/runtime.go cmd/internal/adapt/adapt.go go.mod
   268		go build -o $@ ./cmd/pair-session-watch
   269	
   270	# pair-changelog — distills the pair TTY into the Alt+l change log (#53);
   271	# shares cmd/internal/model with pair-slug.
   272	$(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill.go cmd/pair-changelog/prompt.go cmd/internal/model/model.go go.mod
   273		go build -o $@ ./cmd/pair-changelog
   274	
   275	# pair-continuation — the deterministic writer for the `continuation` datatype
   276	# (ariadne#91): renders frontmatter, allocates a collision-safe timestamped
   277	# name under workshop/continuation/, writes, then commits + pushes (recovery).
   278	$(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-continuation/git.go cmd/pair-continuation/main.go go.mod
   279		go build -o $@ ./cmd/pair-continuation
   280	
   281	PAIR_GO_SRCS := cmd/pair-go/main.go cmd/internal/entrypoint/asset_root.go cmd/internal/entrypoint/launch.go cmd/internal/entrypoint/mode.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go cmd/internal/runtimebundle/manifest.go cmd/internal/runtimebundle/plan.go cmd/internal/runtimebundle/cleanup.go cmd/internal/runtimebundle/embed.go $(RUNTIMEBUNDLE_ASSETS) go.mod
   282	
   283	# pair — public Go-owned entrypoint. It execs bin/pair-shell for the current
   284	# shell-owned zellij lifecycle; source/Homebrew builds can inject
   285	# defaultPairHome, while copied binaries without adjacent assets use the
   286	# embedded runtime fallback.
   287	$(BIN_DIR)/pair: $(PAIR_GO_SRCS)
   288		go build -ldflags "$(PAIR_HOME_LDFLAGS)" -o $@ ./cmd/pair-go
   289	
   290	# pair-go — development dispatcher and explicit migration surface.
   291	# `pair-go launch ...` shares the same bin/pair-shell handoff as public pair;
   292	# helper routes stay in the dispatcher.
   293	$(BIN_DIR)/pair-go: $(PAIR_GO_SRCS)
   294		go build -ldflags "$(PAIR_HOME_LDFLAGS)" -o $@ ./cmd/pair-go
   295	
   296	
   297	# Tart VM targets (tart, tart-gui, tart-mount, tart-stop, tart-clean)
   298	# live in .tart/Makefile, vendored from ariadne and auto-included by
   299	# Makefile.workflow. `make help-tart` for usage.

exec
/bin/zsh -lc "rg -n \"pair-[a-z0-9-]+(\\.sh)?|bin/|PAIR_HOME|command -v\" bin/pair-shell zellij nvim doctor bin/lib | head -n 250" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "find bin -maxdepth 2 -type f | sort | sed -n '1,220p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
bin/pair-shell:1:#!/usr/bin/env bash
bin/pair-shell:32:    if command -v gtimeout >/dev/null 2>&1; then
bin/pair-shell:34:    elif command -v timeout >/dev/null 2>&1; then
bin/pair-shell:104:  `pair-dev` is identical to `pair` but rebuilds pair's Go binaries from
bin/pair-shell:106:  so the zellij-spawned pair-wrap always matches the working tree. Use it
bin/pair-shell:108:  no toolchain dependency). See bin/pair-dev + bin/lib/dev-rebuild.sh.
bin/pair-shell:127:                                pair-claude-2)
bin/pair-shell:160:                `1 bin/pair-wrap`); past 9, use arrows or <C-n>/<C-p>.
bin/pair-shell:187:                from pair-wrap's PTY stream so no per-agent format
bin/pair-shell:229:        if ! command -v zellij >/dev/null 2>&1; then
bin/pair-shell:246:        # Backfill agent-<tag> for any running pair-wrap process whose tag
bin/pair-shell:250:        # agent name from argv (the token after `pair-wrap`).
bin/pair-shell:251:        pw_pids=$(pgrep -f 'pair-wrap' 2>/dev/null || true)
bin/pair-shell:256:                { for (i=1; i<=NF; i++) if ($i ~ /pair-wrap$/) { print $(i+1); exit } }')
bin/pair-shell:326:        #                     pair-restart.sh — so its presence in
bin/pair-shell:378:        if command -v zellij >/dev/null 2>&1; then
bin/pair-shell:398:            for fam in outer-tty pair-wrap-pid title-pid \
bin/pair-shell:437:        old_paths_tmp="$(mktemp "${TMPDIR:-/tmp}/pair-rename.XXXXXX")"
bin/pair-shell:447:                outer-tty-$old_tag|pair-wrap-pid-$old_tag|title-pid-$old_tag|\
bin/pair-shell:556:PAIR_HOME="$(cd -P "$(dirname "$SOURCE")/.." && pwd)"
bin/pair-shell:557:export PAIR_HOME
bin/pair-shell:559:# Put pair's bin/ on PATH so zellij can resolve our helper scripts by bare
bin/pair-shell:563:    *":$PAIR_HOME/bin:"*) ;;
bin/pair-shell:564:    *) export PATH="$PAIR_HOME/bin:$PATH" ;;
bin/pair-shell:680:# DATA_DIR is hoisted above the `command -v zellij` gate, the in-session
bin/pair-shell:692:# — the live pair-wrap is still appending to .raw) share one implementation
bin/pair-shell:710:# Re-read the restart marker dropped by pair-restart.sh (Alt+n / Shift+Alt+N)
bin/pair-shell:755:    if [ -f "$cfg" ] && command -v jq >/dev/null 2>&1; then
bin/pair-shell:815:# `pair-2 (EXITED - attach to resurrect)`; that row still shows in
bin/pair-shell:852:        if [ -f "$legacy" ] && command -v jq >/dev/null 2>&1 \
bin/pair-shell:898:# background poller (bin/pair-title.sh) still defers to a live different-tag
bin/pair-shell:911:    command -v cmux >/dev/null 2>&1 || return 0
bin/pair-shell:928:# script without a live zellij/agent. Placed BEFORE the `command -v zellij` gate
bin/pair-shell:954:if ! command -v zellij >/dev/null 2>&1; then
bin/pair-shell:1011:# before any session work, so tests/pair-continue-test.sh can assert the
bin/pair-shell:1021:# it; --session would nest). Instead: park the scrollback (copy — pair-wrap is
bin/pair-shell:1023:# the session — the outer bin/pair then re-execs `pair continue <slug>` fresh
bin/pair-shell:1094:    # (zellij/layouts/main.kdl, bin/pair-scrollback-open) point that env
bin/pair-shell:1170:# "$PAIR_PANE_TITLE"` before exec'ing pair-wrap, so the frame shows the
bin/pair-shell:1174:# $PWD here is the dir the user ran `pair` from (bin/pair never cd's); the
bin/pair-shell:1189:# command can record it into pane-<tag>-<agent>.json for pair-title's frame
bin/pair-shell:1204:    for old in "$HOME/scratch"/pair-draft-*.md "$HOME/scratch"/pair-log-*.md; do
bin/pair-shell:1230:# named sessions like pair-blogging that don't follow any agent prefix.
bin/pair-shell:1255:    # set -o pipefail, that propagates and errexit silently kills bin/pair at
bin/pair-shell:1277:# autosave and bin/pair's send-log), bucket per tag, take max mtime, filter
bin/pair-shell:1467:    if command -v fzf >/dev/null 2>&1; then
bin/pair-shell:1586:              "$DATA_DIR/pair-wrap-pid-${PAIR_TAG}" \
bin/pair-shell:1598:        # Display SESSION (e.g. pair-2) rather than PAIR_TAG ("2") to match
bin/pair-shell:1651:# pair-restart.sh. Two flavors:
bin/pair-shell:1684:# Ensure the per-tag title poller (bin/pair-title.sh) is running for this tag.
bin/pair-shell:1694:    "$PAIR_HOME/bin/pair-title.sh" "$PAIR_TAG" "$title_agent" \
bin/pair-shell:1708:    # preexec hook set (typically "bin/pair").
bin/pair-shell:1718:    zellij --config-dir "$PAIR_HOME/zellij" attach "$SESSION"
bin/pair-shell:1728:if ! command -v "$AGENT" >/dev/null 2>&1; then
bin/pair-shell:1767:    if command -v zsh >/dev/null 2>&1; then
bin/pair-shell:1843:    # Defensive strip: pair-session-watch.sh and the inline explicit_resume
bin/pair-shell:2036:# name (e.g. `pair-bugfix`) doesn't always carry the agent. Persists across
bin/pair-shell:2083:    # bin/pair-session-watch.sh; keep them in sync.
bin/pair-shell:2193:# present — several detection points in this script + pair-session-
bin/pair-shell:2214:# the id async via pair-session-watch.sh, so it is empty here; pair-changelog-open
bin/pair-shell:2220:# components (pair-wrap, pair-session-watch.sh, pair-slug, nvim) append one
bin/pair-shell:2222:# pair-doctor reads it to spot integration drift. bin/pair owns its lifecycle:
bin/pair-shell:2235:"$PAIR_HOME/bin/pair-session-watch.sh" "$AGENT" "$PAIR_TAG" "$PWD" $agent_extra \
bin/pair-shell:2246:# Dev mode (pair-dev exports PAIR_DEV): rebuild the Go binaries from source so
bin/pair-shell:2247:# the layout's `exec pair-wrap` below resolves to a fresh repo/bin build — first
bin/pair-shell:2250:# only, so a plain attach skips it; restart re-execs $0=bin/pair with PAIR_DEV
bin/pair-shell:2252:# shellcheck source=bin/lib/dev-rebuild.sh
bin/pair-shell:2253:. "$PAIR_HOME/bin/lib/dev-rebuild.sh"
bin/pair-shell:2283:    --config-dir              "$PAIR_HOME/zellij" \
bin/pair-shell:2284:    --new-session-with-layout "$PAIR_HOME/zellij/layouts/main.kdl" \
bin/lib/adapt-log.sh:3:# Sourced by shell components (pair-session-watch.sh, …) that need to record a
bin/lib/adapt-log.sh:17:    command -v jq >/dev/null 2>&1 || return 0
doctor/emitter-health.sh:5:# Why this exists: a pair-wrap / pair-slug binary built before #000045 has no
doctor/emitter-health.sh:9:# (#000046's `pair-dev` PREVENTS staleness at launch; this DIAGNOSES it.)
doctor/emitter-health.sh:14:#   pair-wrap → cmd/pair-wrap/main.go  ("return-remap" / "overlay-detect" / "output-filter")
doctor/emitter-health.sh:15:#   pair-slug → cmd/pair-slug/main.go  ("slug-parse")
doctor/emitter-health.sh:25:    if [ -z "$exe" ] && command -v lsof >/dev/null 2>&1; then   # macOS/full path
doctor/emitter-health.sh:39:# *actually running* binary (via the pair-wrap-pid-<tag> pidfile) since that's
doctor/emitter-health.sh:42:# bug). Only pair-wrap is long-running with a pidfile; pair-slug is on-demand,
doctor/emitter-health.sh:46:    if [ "$name" = "pair-wrap" ] && [ -n "$tag" ]; then
doctor/emitter-health.sh:47:        pidfile="$datadir/pair-wrap-pid-$tag"
doctor/emitter-health.sh:54:    command -v "$name" 2>/dev/null || true
doctor/emitter-health.sh:62:    command -v strings >/dev/null 2>&1 || return 2
doctor/emitter-health.sh:77:    for spec in "pair-wrap:return-remap" "pair-slug:slug-parse"; do
doctor/emitter-health.sh:91:               echo "          Fix: make install (or launch via pair-dev). See atlas \"Binary freshness\"." ;;
nvim/changelog.lua:4:-- bin/pair-changelog-open. The distilled counterpart to scrollback.lua, but
nvim/changelog.lua:81:-- The orchestrator (bin/pair-changelog-open) launches render+distill as a nohup'd
zellij/layouts/main.kdl:7://   $PAIR_HOME        — pair install location (for nvim init)
zellij/layouts/main.kdl:8:// All four are exported by bin/pair before invoking zellij.
zellij/layouts/main.kdl:13:// The agent runs under pair-wrap, a transparent PTY proxy that translates
zellij/layouts/main.kdl:32:            // are exported by bin/pair before zellij launches, so the
zellij/layouts/main.kdl:45:            args "-c" "_pdd=\"${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}\"; printf '{\"pane_id\":\"%s\",\"cwd\":\"%s\",\"cwd_display\":\"%s\"}\\n' \"$ZELLIJ_PANE_ID\" \"$PWD\" \"${PAIR_PANE_CWD:-$PWD}\" > \"$_pdd/pane-${PAIR_TAG:-${PAIR_AGENT:-claude}}-${PAIR_AGENT:-claude}.json\" 2>/dev/null; zellij action rename-pane --pane-id \"$ZELLIJ_PANE_ID\" \"${PAIR_PANE_TITLE:-agent}\" 2>/dev/null; exec pair-wrap --scrollback-log \"${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}/scrollback-${PAIR_TAG:-${PAIR_AGENT:-claude}}-${PAIR_AGENT:-claude}.raw\" ${PAIR_AGENT:-claude} ${PAIR_AGENT_ARGS:-}"
zellij/layouts/main.kdl:63:            // embed pid here so cleanup_quit_marker in bin/pair can reap
zellij/layouts/main.kdl:67:            args "-c" "DATA_DIR=\"${XDG_DATA_HOME:-$HOME/.local/share}/pair\" && mkdir -p \"$DATA_DIR\" && export PAIR_NVIM_PID_FILE=\"$DATA_DIR/nvim-pid-${PAIR_TAG:-${PAIR_AGENT:-claude}}-draft\" && exec nvim -u \"$PAIR_HOME/nvim/init.lua\" \"$DATA_DIR/draft-${PAIR_TAG:-${PAIR_AGENT:-claude}}.md\""
zellij/layouts/main.kdl:73:    // re-tiling existing panes positionally. Pane processes (pair-wrap,
bin/lib/dev-rebuild.sh:3:# `pair-dev` exports PAIR_DEV=1 then execs `pair`; bin/pair sources this and
bin/lib/dev-rebuild.sh:5:# execs pair-wrap. In dev mode it recompiles the Go binaries from source into
bin/lib/dev-rebuild.sh:6:# $PAIR_HOME/bin (which is first on PATH), so the layout's `exec pair-wrap` — a
bin/lib/dev-rebuild.sh:9:# since bin/ is gitignored) ~/.local/bin copy. See atlas/architecture.md.
bin/lib/dev-rebuild.sh:11:# Restart-safe: Alt+n / Shift+Alt+N re-exec $0=bin/pair, and PAIR_DEV rides
bin/lib/dev-rebuild.sh:17:# Usage:  PAIR_HOME=<repo> dev_rebuild      (no-op unless PAIR_DEV is set)
bin/lib/dev-rebuild.sh:20:# `set -e` (bin/pair:20) — least of all mid-restart, when the old session is
bin/lib/dev-rebuild.sh:26:    if ! command -v make >/dev/null 2>&1; then
bin/lib/dev-rebuild.sh:27:        echo "pair-dev: 'make' not on PATH — launching with existing binaries." >&2
bin/lib/dev-rebuild.sh:30:    echo "pair-dev: rebuilding Go binaries (make build in $PAIR_HOME) …" >&2
bin/lib/dev-rebuild.sh:31:    make -C "$PAIR_HOME" build >&2 \
bin/lib/dev-rebuild.sh:32:        || echo "pair-dev: build FAILED — launching with last-good binaries (fix, then Alt+n)." >&2
doctor/SKILL.md:2:name: xx-pair-doctor
doctor/SKILL.md:6:# pair-doctor — diagnose agent-harness integration drift
doctor/SKILL.md:34:   output leads with an emitter-health line: a `[STALE]` `pair-wrap`/`pair-slug`
doctor/SKILL.md:36:   the fix is `make install` / `pair-dev` (#000046), not a matcher edit. Then the
doctor/SKILL.md:58:- The log truncates at each session launch (`bin/pair`), so it reflects the
zellij/config.kdl:22:// pair-scrollback-render historyRows cap so PageUp inside the agent
zellij/config.kdl:39:// bin/pair prepends $PAIR_HOME/bin to PATH so the script is resolvable by
zellij/config.kdl:40:// bare name — avoids the fragile `sh -c '$PAIR_HOME/...'` quoting dance.
zellij/config.kdl:64:        // reclaimed by pair-specific flows.
zellij/config.kdl:74:        // the only quit path; it tears down cleanly via pair-quit.sh.
zellij/config.kdl:150:        // destructive action — pair-quit.sh kills the zellij session, so an
zellij/config.kdl:152:        // shells out to pair-quit.sh only on Yes.
zellij/config.kdl:162:        // pair-help wraps `less` with a custom keymap so ESC (in addition to
zellij/config.kdl:166:            Run "pair-help" {
zellij/config.kdl:178:        // ANSI-colored nvim viewer (#000017). pair-wrap captures the raw
zellij/config.kdl:179:        // PTY stream alongside resize events; pair-scrollback-open replays
zellij/config.kdl:185:            Run "pair-scrollback-open" {
zellij/config.kdl:205:        // pair-review-toggle pane caused. When Alt+c is pressed from inside the
zellij/config.kdl:217:        // viewer (#53). The distilled counterpart to Alt+/: pair-changelog-open
zellij/config.kdl:218:        // cleans the captured TTY, runs the pair-changelog distiller (LLM
zellij/config.kdl:223:            Run "pair-changelog-open" {
doctor/doctor.sh:1:#!/usr/bin/env bash
doctor/doctor.sh:30:    command -v emitter_health_report >/dev/null 2>&1 || return 0
doctor/doctor.sh:55:if ! command -v jq >/dev/null 2>&1; then
doctor/doctor.sh:153:    printf '%s\n' "$(paint "$BOLD" '══ pair-doctor ══')"
doctor/doctor.sh:155:    echo "== pair-doctor =="
doctor/README.md:1:# pair-doctor — diagnose agent-harness integration drift
doctor/README.md:25:drift but isn't: a `pair-wrap`/`pair-slug` binary built before the flight
doctor/README.md:29:`make install`, or launch via `pair-dev` (#000046). A `[STALE]` line explains an
doctor/README.md:40:| `overlay-detect/near-miss` (aspect 2) | harness renamed its picker; the `detail` holds the new wording | add that string to `codexPickerMarkers` / `agyPickerMarkers` (or the OSC body for claude) in `cmd/pair-wrap/main.go` |
doctor/README.md:42:| `session-id/fail` or `near-miss` (aspect 3) | session file moved or id format changed | update `watch_dir`/`find_args`/`extract_id` in `bin/pair-session-watch.sh` |
doctor/README.md:43:| `slug-parse/near-miss` (aspect 4) | transcript schema changed | update the parser in `cmd/pair-slug/slug.go` |
doctor/README.md:44:| `output-filter` *absent* for codex (aspect 5) | a sync-output sequence was renamed (no `fired` line where you'd expect one) | update `codexSyncOutputMarkers` in `cmd/pair-wrap/main.go` |
doctor/README.md:53:- The log truncates at each session launch (`bin/pair`), so it reflects the
doctor/README.md:64:`$PAIR_HOME`-absolute instruction to run `doctor.sh` and propose fixes — so it
nvim/doctor_test.lua:4:-- $PAIR_HOME-absolute substitution and graceful nil-on-unset.
nvim/doctor_test.lua:19:ok(type(p) == 'string', 'payload returns a string for a real PAIR_HOME')
nvim/doctor_test.lua:23:ok(p:find('$PAIR_HOME', 1, true) == nil, 'payload has no literal $PAIR_HOME (substituted, not deferred)')
nvim/doctor_test.lua:25:-- Trailing slash on PAIR_HOME must not double up in the paths.
doctor/doctor_test.sh:1:#!/usr/bin/env bash
doctor/doctor_test.sh:15:RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-doctor-test.XXXXXX")"
doctor/doctor_test.sh:25:    echo '{"ts":"2026-06-03T10:00:00Z","comp":"pair-wrap","agent":"codex","aspect":1,"signal":"return-remap","outcome":"fired","detail":"a"}'
doctor/doctor_test.sh:26:    echo '{"ts":"2026-06-03T10:00:01Z","comp":"pair-wrap","agent":"codex","aspect":2,"sig'   # truncated, invalid JSON
doctor/doctor_test.sh:27:    echo '{"ts":"2026-06-03T10:00:02Z","comp":"pair-wrap","agent":"codex","aspect":2,"signal":"overlay-detect","outcome":"near-miss","detail":"Do you want to apply this patch? (y/n)"}'
nvim/scrollback.lua:1:-- nvim/scrollback.lua — read-only viewer for pair-scrollback-render output.
nvim/scrollback.lua:3:-- Loaded via `nvim -u $PAIR_HOME/nvim/scrollback.lua <path-to-.ansi>`.
nvim/scrollback.lua:32:-- Stub out the pair-launcher cmdline targets so a stray zellij Alt+Up
nvim/scrollback.lua:56:-- $PAIR_NVIM_PID_FILE so bin/pair's cleanup_quit_marker can reap it on
nvim/scrollback.lua:285:  if vim.env.PAIR_HOME and vim.env.PAIR_HOME ~= '' then
nvim/scrollback.lua:286:    bin = vim.env.PAIR_HOME .. '/bin/pair-scrollback-render'
nvim/scrollback.lua:288:    bin = 'pair-scrollback-render'
nvim/adapt.lua:5:-- the Go (cmd/internal/adapt) and shell (bin/lib/adapt-log.sh) emitters, so
nvim/doctor.lua:4:-- (read $PAIR_HOME, hand the instruction to the agent pane via send_to_agent).
nvim/doctor.lua:10:-- $PAIR_HOME — so it can hand ANY agent a $PAIR_HOME-absolute instruction. The
nvim/doctor.lua:14:-- payload(pair_home) → the instruction string with $PAIR_HOME-absolute paths
nvim/doctor.lua:15:-- substituted in (NOT a literal `$PAIR_HOME` — the agent must not depend on its
nvim/review.lua:2:-- `nvim -u $PAIR_HOME/nvim/review.lua <file>` (from bin/pair-review-open).
nvim/review.lua:539:  -- pid file (reaped by bin/pair's cleanup) + the open-state file.
nvim/scrollback_test.lua:288:  local pend = (os.getenv('TMPDIR') or '/tmp') .. '/pair-sb-annotate-test.md'
nvim/slug.lua:5:-- The proposer (cmd/pair-slug, a Stop hook) writes a candidate slug to
nvim/pair_poke.lua:14:-- Alt+Enter key event so zellij delivers one modified chord to pair-wrap.
nvim/pair_poke.lua:41:-- draft — the same predicate pair-scrollback-open uses to find it.
nvim/changelog_test.lua:20:  '- M1 done for #53 on `cmd/pair-changelog`, branch feature/53-changelog',
nvim/changelog_test.lua:48:  local path = (os.getenv('TMPDIR') or '/tmp') .. '/pair-changelog-reload-test.md'
nvim/changelog_test.lua:73:  local pend = (os.getenv('TMPDIR') or '/tmp') .. '/pair-cl-annotate-test.md'
nvim/changelog_test.lua:119:  vim.api.nvim_buf_set_name(named, (os.getenv('TMPDIR') or '/tmp') .. '/pair-cl-named-test.md')
nvim/zellij_trace.lua:1:-- nvim/zellij_trace.lua — trace pair-originated zellij action calls.
nvim/init.lua:14:-- the embed pid (lua runs inside the embed); bin/pair's cleanup_quit_marker
nvim/init.lua:88:--   1. The pair-specific `===` comment line — faded (Comment) in general,
nvim/init.lua:271:-- (bin/copy-on-select.sh + bin/clipboard-to-pane.sh) lands a
nvim/init.lua:742:  -- pair-wrap's stdin translator rewrites incoming \r into the agent's
nvim/init.lua:750:  -- (write 13) is exactly what pair-wrap rewrites into the agent's
nvim/init.lua:916:    local home = vim.env.PAIR_HOME or ''
nvim/init.lua:918:    if not bin or bin == '' then bin = (home ~= '') and (home .. '/bin/pair-review-readiness') or 'pair-review-readiness' end
nvim/init.lua:947:  -- branch on the review target (seam #6): ready→open via pair-review-open,
nvim/init.lua:965:      local home = vim.env.PAIR_HOME or ''
nvim/init.lua:966:      local bin = (home ~= '') and (home .. '/bin/pair-review-open') or 'pair-review-open'
nvim/init.lua:1063:--   2. Read pair-wrap's pid from $DATA_DIR/pair-wrap-pid-<tag>. Bail with a
nvim/init.lua:1064:--      restart hint if missing or dead — pair-wrap is the whole I/O path
nvim/init.lua:1066:--   3. SIGUSR1 pair-wrap to arm a capture window, then write Ctrl+V to the
nvim/init.lua:1068:--      input area; pair-wrap tees the bytes into image-capture-<tag>.
nvim/init.lua:1282:  local pid_path = dd .. '/pair-wrap-pid-' .. tag
nvim/init.lua:1299:    vim.notify('pair: pair-wrap pid missing — restart the pair session (Alt+n)',
nvim/init.lua:1336:    -- file itself doesn't need clearing — pair-wrap truncates it on write.
nvim/init.lua:1338:    -- Arm capture. If pair-wrap is gone, SIGUSR1 fails — surface a clear
nvim/init.lua:1345:        'pair: pair-wrap (pid ' .. pid .. ') not running — placeholder left in place; restart the pair session (Alt+n)',
nvim/init.lua:1365:-- PairPasteQuote: triggered from bin/clipboard-to-pane.sh after a copy_command
nvim/init.lua:1607:-- Reload via `:luafile $PAIR_HOME/nvim/init.lua` (works because all autocmds
nvim/init.lua:1655:-- `/`, `~`, or `./` / `../` etc. Plain `bin/pair-wrap` is *not* a path here
nvim/init.lua:1695:--      the agent's output by pair-wrap. Each line is
nvim/init.lua:1727:-- `pair-wrap`, `lessons.md`, `bin/pair-wrap`, `$PAIR_HOME`,
nvim/init.lua:2982:-- Resolve the change-log session id (#63): the env var bin/pair exports when the
nvim/init.lua:2985:-- order in bin/pair-changelog-open so the polled .ready path matches the base the
nvim/init.lua:3185:      vim.fn.system('pair-quit.sh')
nvim/init.lua:3202:-- (PairConfirmRestartNewSession). Differs in whether pair-restart.sh
nvim/init.lua:3288:    local argv = { 'pair-restart.sh' }
nvim/init.lua:3302:-- pair-restart.sh directly), creating a continuation needs the agent's
nvim/init.lua:3319:  '   then distill per that procedure and finalize with the pair-continuation',
nvim/init.lua:3322:  '   (or  pair-dev continue <that-slug>  if this is a dev checkout)',
nvim/init.lua:3339:-- path (the VT byte stream pair-wrap --scrollback-log captures; the substrate
nvim/init.lua:3340:-- pair-scrollback-render replays). It lives in the XDG data dir, NOT the repo,
nvim/init.lua:3505:-- in zellij/config.kdl), but passes `--jump prev` so pair-scrollback-open
nvim/init.lua:3515:    '--', 'pair-scrollback-open', '--jump', 'prev',
nvim/init.lua:3668:-- bin/clipboard-to-pane.sh sends (as a single Ctrl-_, ASCII 31) after a
nvim/init.lua:3946:-- The Stop hook's `cmd/pair-slug` proposes a `=== <branch> | <focus> ===`
nvim/init.lua:3960:-- :PairDoctor — agent-agnostic pair-doctor entry (issue #000048). The payload is
nvim/init.lua:3962:-- $PAIR_HOME and hand the absolute-pathed instruction to whatever agent is
nvim/init.lua:3964:-- since the paths are $PAIR_HOME-absolute). Auto-sends via send_to_agent.
nvim/init.lua:3977:    local body = doctor.payload(vim.env.PAIR_HOME)
nvim/init.lua:3979:      vim.notify('PairDoctor: PAIR_HOME unset (run inside a pair session).',
nvim/init.lua:3986:    { desc = 'Ask the agent to run pair-doctor and propose harness-drift fixes' })
nvim/review/readiness.lua:4:-- gathering lives in bin/pair-review-readiness. The 4 cases are
nvim/review/seam.lua:4:-- and bin/pair-review-open — can't diverge on the empty-tag fallback
nvim/review/mode_test.lua:3:-- in pair-side markdown prompt files.
nvim/review/mode_test.lua:14:eq(M.parse, nil, 'mode parser removed; no pair-side markdown prompt parsing')
nvim/review/mode_test.lua:29:eq(generate.body, nil, 'generate has no pair-side prompt body')
nvim/review/mode_test.lua:32:eq(edit.body, nil, 'edit has no pair-side prompt body')
nvim/review/mode_test.lua:35:eq(proofread.body, nil, 'proofread has no pair-side prompt body')
nvim/review/mode.lua:1:-- nvim/review/mode.lua — pair-side review mode UI metadata.

 succeeded in 0ms:
bin/__pycache__/pair-wrapcpython-314.pyc
bin/clipboard-to-pane.sh
bin/copy-on-select.sh
bin/flash-pane.sh
bin/lib/adapt-log.sh
bin/lib/dev-rebuild.sh
bin/pair
bin/pair-changelog
bin/pair-changelog-open
bin/pair-context
bin/pair-continuation
bin/pair-dev
bin/pair-go
bin/pair-help
bin/pair-notify
bin/pair-quit.sh
bin/pair-restart.sh
bin/pair-review-open
bin/pair-review-readiness
bin/pair-review-target
bin/pair-scribe
bin/pair-scrollback-open
bin/pair-scrollback-render
bin/pair-session-watch
bin/pair-session-watch.sh
bin/pair-shell
bin/pair-slug
bin/pair-title.sh
bin/pair-wrap

exec
/bin/zsh -lc 'find bin nvim zellij doctor -type l -ls' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
exec
/bin/zsh -lc "nl -ba cmd/internal/entrypoint/asset_root.go | sed -n '1,140p'; nl -ba cmd/internal/entrypoint/asset_root_test.go | sed -n '1,140p'; nl -ba cmd/pair-go/main_test.go | sed -n '1,330p'" in /Users/xianxu/workspace/pair
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
    13		EmbeddedRoot    string
    14		PairShellExists func(root string) bool
    15	}
    16	
    17	type AssetRoot struct {
    18		Root      string
    19		ShellPath string
    20		Source    string
    21	}
    22	
    23	func ResolveAssetRoot(input AssetRootInput) (AssetRoot, error) {
    24		exists := input.PairShellExists
    25		if exists == nil {
    26			exists = func(string) bool { return false }
    27		}
    28	
    29		candidates := make([]assetRootCandidate, 0, 3)
    30		if input.PairHome != "" {
    31			candidates = append(candidates, assetRootCandidate{root: input.PairHome, source: "PAIR_HOME"})
    32		}
    33		if input.Executable != "" {
    34			candidates = append(candidates, assetRootCandidate{
    35				root:   filepath.Dir(filepath.Dir(input.Executable)),
    36				source: "executable sibling",
    37			})
    38		}
    39		if input.DefaultPairHome != "" {
    40			candidates = append(candidates, assetRootCandidate{root: input.DefaultPairHome, source: "defaultPairHome"})
    41		}
    42		if input.EmbeddedRoot != "" {
    43			candidates = append(candidates, assetRootCandidate{root: input.EmbeddedRoot, source: "embedded runtime"})
    44		}
    45	
    46		seen := map[string]bool{}
    47		checked := make([]string, 0, len(candidates))
    48		for _, candidate := range candidates {
    49			root := filepath.Clean(candidate.root)
    50			if root == "." || seen[root] {
    51				continue
    52			}
    53			seen[root] = true
    54			checked = append(checked, root)
    55			if exists(root) {
    56				return AssetRoot{
    57					Root:      root,
    58					ShellPath: PairShellPath(root),
    59					Source:    candidate.source,
    60				}, nil
    61			}
    62		}
    63	
    64		if len(checked) == 0 {
    65			checked = append(checked, "<none>")
    66		}
    67		return AssetRoot{}, fmt.Errorf("pair-shell not found; set PAIR_HOME to a Pair checkout/install root containing bin/pair-shell (checked: %s)", strings.Join(checked, ", "))
    68	}
    69	
    70	func PairShellPath(root string) string {
    71		return filepath.Join(root, "bin", "pair-shell")
    72	}
    73	
    74	type assetRootCandidate struct {
    75		root   string
    76		source string
    77	}
     1	package entrypoint
     2	
     3	import (
     4		"strings"
     5		"testing"
     6	)
     7	
     8	func TestResolveAssetRootPrefersPairHome(t *testing.T) {
     9		root, err := ResolveAssetRoot(AssetRootInput{
    10			PairHome:        "/env/root",
    11			Executable:      "/repo/bin/pair",
    12			DefaultPairHome: "/default/root",
    13			PairShellExists: existingRoots("/env/root", "/repo", "/default/root"),
    14		})
    15		if err != nil {
    16			t.Fatalf("ResolveAssetRoot error = %v", err)
    17		}
    18		if root.Root != "/env/root" {
    19			t.Fatalf("Root = %q, want /env/root", root.Root)
    20		}
    21		if root.ShellPath != "/env/root/bin/pair-shell" {
    22			t.Fatalf("ShellPath = %q, want /env/root/bin/pair-shell", root.ShellPath)
    23		}
    24	}
    25	
    26	func TestResolveAssetRootUsesExecutableSiblingRoot(t *testing.T) {
    27		root, err := ResolveAssetRoot(AssetRootInput{
    28			Executable:      "/repo/bin/pair",
    29			DefaultPairHome: "/default/root",
    30			PairShellExists: existingRoots("/repo", "/default/root"),
    31		})
    32		if err != nil {
    33			t.Fatalf("ResolveAssetRoot error = %v", err)
    34		}
    35		if root.Root != "/repo" {
    36			t.Fatalf("Root = %q, want /repo", root.Root)
    37		}
    38		if root.ShellPath != "/repo/bin/pair-shell" {
    39			t.Fatalf("ShellPath = %q, want /repo/bin/pair-shell", root.ShellPath)
    40		}
    41	}
    42	
    43	func TestResolveAssetRootFallsBackToDefaultPairHome(t *testing.T) {
    44		root, err := ResolveAssetRoot(AssetRootInput{
    45			Executable:      "/home/me/.local/bin/pair",
    46			DefaultPairHome: "/repo",
    47			PairShellExists: existingRoots("/repo"),
    48		})
    49		if err != nil {
    50			t.Fatalf("ResolveAssetRoot error = %v", err)
    51		}
    52		if root.Root != "/repo" {
    53			t.Fatalf("Root = %q, want /repo", root.Root)
    54		}
    55		if root.ShellPath != "/repo/bin/pair-shell" {
    56			t.Fatalf("ShellPath = %q, want /repo/bin/pair-shell", root.ShellPath)
    57		}
    58	}
    59	
    60	func TestResolveAssetRootFallsBackToEmbeddedRootAfterAdjacentRoots(t *testing.T) {
    61		root, err := ResolveAssetRoot(AssetRootInput{
    62			Executable:      "/home/me/.local/bin/pair",
    63			DefaultPairHome: "/default/root",
    64			EmbeddedRoot:    "/data/pair/runtime/abc/pair-home",
    65			PairShellExists: existingRoots("/data/pair/runtime/abc/pair-home"),
    66		})
    67		if err != nil {
    68			t.Fatalf("ResolveAssetRoot error = %v", err)
    69		}
    70		if root.Root != "/data/pair/runtime/abc/pair-home" {
    71			t.Fatalf("Root = %q, want embedded root", root.Root)
    72		}
    73		if root.Source != "embedded runtime" {
    74			t.Fatalf("Source = %q, want embedded runtime", root.Source)
    75		}
    76	}
    77	
    78	func TestResolveAssetRootReportsMissingPairShellAndPairHome(t *testing.T) {
    79		_, err := ResolveAssetRoot(AssetRootInput{
    80			Executable:      "/home/me/.local/bin/pair",
    81			DefaultPairHome: "/repo",
    82			PairShellExists: existingRoots(),
    83		})
    84		if err == nil {
    85			t.Fatal("ResolveAssetRoot error = nil, want missing-root error")
    86		}
    87		for _, want := range []string{"pair-shell", "PAIR_HOME", "/home/me/.local", "/repo"} {
    88			if !strings.Contains(err.Error(), want) {
    89				t.Fatalf("error missing %q:\n%v", want, err)
    90			}
    91		}
    92	}
    93	
    94	func existingRoots(roots ...string) func(string) bool {
    95		set := make(map[string]bool, len(roots))
    96		for _, root := range roots {
    97			set[root] = true
    98		}
    99		return func(root string) bool {
   100			return set[root]
   101		}
   102	}
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
    43			roots:      map[string]bool{"/repo": true},
    44		}
    45		var stdout, stderr bytes.Buffer
    46		code := runWithLegacyRuntime([]string{"launch", "--help"}, &stdout, &stderr, rt)
    47		if code != 0 {
    48			t.Fatalf("code = %d, want 0", code)
    49		}
    50		if stdout.String() != "" {
    51			t.Fatalf("stdout = %q, want empty", stdout.String())
    52		}
    53		if stderr.String() != "" {
    54			t.Fatalf("stderr = %q, want empty", stderr.String())
    55		}
    56		if rt.execPath != "/repo/bin/pair-shell" {
    57			t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
    58		}
    59		if rt.execLabel != "pair-go launch" {
    60			t.Fatalf("execLabel = %q, want pair-go launch", rt.execLabel)
    61		}
    62		wantArgv := []string{"pair", "--help"}
    63		if !reflect.DeepEqual(rt.execArgv, wantArgv) {
    64			t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
    65		}
    66	}
    67	
    68	func TestRunLaunchExecsLegacyPairWithArgvAndEnv(t *testing.T) {
    69		t.Setenv("PAIR_TEST_ENV", "kept")
    70		rt := &fakeLegacyRuntime{
    71			executable: "/repo/bin/pair-go",
    72			roots:      map[string]bool{"/repo": true},
    73			execCode:   42,
    74		}
    75	
    76		var stdout, stderr bytes.Buffer
    77		code := runWithLegacyRuntime([]string{"launch", "claude", "--", "--resume"}, &stdout, &stderr, rt)
    78	
    79		if code != 42 {
    80			t.Fatalf("code = %d, want 42", code)
    81		}
    82		if stdout.String() != "" {
    83			t.Fatalf("stdout = %q, want empty", stdout.String())
    84		}
    85		if stderr.String() != "" {
    86			t.Fatalf("stderr = %q, want empty", stderr.String())
    87		}
    88		if rt.execPath != "/repo/bin/pair-shell" {
    89			t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
    90		}
    91		if rt.execLabel != "pair-go launch" {
    92			t.Fatalf("execLabel = %q, want pair-go launch", rt.execLabel)
    93		}
    94		wantArgv := []string{"pair", "claude", "--", "--resume"}
    95		if !reflect.DeepEqual(rt.execArgv, wantArgv) {
    96			t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
    97		}
    98		if !containsEnv(rt.execEnv, "PAIR_TEST_ENV=kept") {
    99			t.Fatalf("execEnv missing PAIR_TEST_ENV=kept: %#v", rt.execEnv)
   100		}
   101	}
   102	
   103	func TestRunLaunchReportsMissingLegacyPair(t *testing.T) {
   104		rt := &fakeLegacyRuntime{
   105			executable: "/repo/bin/pair-go",
   106		}
   107	
   108		var stdout, stderr bytes.Buffer
   109		code := runWithLegacyRuntime([]string{"launch", "claude"}, &stdout, &stderr, rt)
   110	
   111		if code != 1 {
   112			t.Fatalf("code = %d, want 1", code)
   113		}
   114		if stdout.String() != "" {
   115			t.Fatalf("stdout = %q, want empty", stdout.String())
   116		}
   117		for _, want := range []string{"pair-go launch", "pair-shell", "PAIR_HOME", "/repo", "make build", "make install", "dev-aliases.sh"} {
   118			if !strings.Contains(stderr.String(), want) {
   119				t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
   120			}
   121		}
   122		if rt.execPath != "" {
   123			t.Fatalf("execPath = %q, want empty", rt.execPath)
   124		}
   125	}
   126	
   127	func TestRunDirectPairExecsLegacyShellWithAllArgs(t *testing.T) {
   128		rt := &fakeLegacyRuntime{
   129			executable: "/repo/bin/pair",
   130			roots:      map[string]bool{"/repo": true},
   131			execCode:   7,
   132		}
   133	
   134		var stdout, stderr bytes.Buffer
   135		code := runWithLegacyRuntime([]string{"claude", "--", "--resume"}, &stdout, &stderr, rt)
   136	
   137		if code != 7 {
   138			t.Fatalf("code = %d, want 7", code)
   139		}
   140		if stdout.String() != "" {
   141			t.Fatalf("stdout = %q, want empty", stdout.String())
   142		}
   143		if stderr.String() != "" {
   144			t.Fatalf("stderr = %q, want empty", stderr.String())
   145		}
   146		if rt.execPath != "/repo/bin/pair-shell" {
   147			t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
   148		}
   149		if rt.execLabel != "pair" {
   150			t.Fatalf("execLabel = %q, want pair", rt.execLabel)
   151		}
   152		wantArgv := []string{"pair", "claude", "--", "--resume"}
   153		if !reflect.DeepEqual(rt.execArgv, wantArgv) {
   154			t.Fatalf("execArgv = %#v, want %#v", rt.execArgv, wantArgv)
   155		}
   156	}
   157	
   158	func TestRunDirectPairFallsBackToDefaultPairHome(t *testing.T) {
   159		rt := &fakeLegacyRuntime{
   160			executable:      "/home/me/.local/bin/pair",
   161			defaultPairHome: "/repo",
   162			roots:           map[string]bool{"/repo": true},
   163		}
   164	
   165		var stdout, stderr bytes.Buffer
   166		code := runWithLegacyRuntime([]string{"--help"}, &stdout, &stderr, rt)
   167	
   168		if code != 0 {
   169			t.Fatalf("code = %d, want 0", code)
   170		}
   171		if stderr.String() != "" {
   172			t.Fatalf("stderr = %q, want empty", stderr.String())
   173		}
   174		if rt.execPath != "/repo/bin/pair-shell" {
   175			t.Fatalf("execPath = %q, want /repo/bin/pair-shell", rt.execPath)
   176		}
   177	}
   178	
   179	func TestRunDirectPairFallsBackToEmbeddedRuntimeAndSetsPairHome(t *testing.T) {
   180		rt := &fakeLegacyRuntime{
   181			executable:   "/home/me/.local/bin/pair",
   182			embeddedRoot: "/data/pair/runtime/abc/pair-home",
   183			roots:        map[string]bool{"/data/pair/runtime/abc/pair-home": true},
   184			execCode:     9,
   185			environ:      []string{"PATH=/bin", "PAIR_HOME=/old"},
   186		}
   187	
   188		var stdout, stderr bytes.Buffer
   189		code := runWithLegacyRuntime([]string{"--help"}, &stdout, &stderr, rt)
   190	
   191		if code != 9 {
   192			t.Fatalf("code = %d, want 9", code)
   193		}
   194		if rt.execPath != "/data/pair/runtime/abc/pair-home/bin/pair-shell" {
   195			t.Fatalf("execPath = %q, want embedded pair-shell", rt.execPath)
   196		}
   197		if !containsEnv(rt.execEnv, "PAIR_HOME=/data/pair/runtime/abc/pair-home") {
   198			t.Fatalf("execEnv missing embedded PAIR_HOME: %#v", rt.execEnv)
   199		}
   200		if containsEnv(rt.execEnv, "PAIR_HOME=/old") {
   201			t.Fatalf("execEnv kept old PAIR_HOME: %#v", rt.execEnv)
   202		}
   203	}
   204	
   205	func TestRunPairGoHelperDoesNotProbeOrExecShellLauncher(t *testing.T) {
   206		rt := &fakeLegacyRuntime{
   207			executable: "/repo/bin/pair-go",
   208		}
   209	
   210		var stdout, stderr bytes.Buffer
   211		code := runWithLegacyRuntime([]string{"help"}, &stdout, &stderr, rt)
   212	
   213		if code != 0 {
   214			t.Fatalf("code = %d, want 0", code)
   215		}
   216		if rt.statCalls != 0 {
   217			t.Fatalf("statCalls = %d, want 0", rt.statCalls)
   218		}
   219		if rt.execPath != "" {
   220			t.Fatalf("execPath = %q, want empty", rt.execPath)
   221		}
   222		if !strings.Contains(stdout.String(), "Usage: pair-go <command> [args]") {
   223			t.Fatalf("stdout missing usage:\n%s", stdout.String())
   224		}
   225		if stderr.String() != "" {
   226			t.Fatalf("stderr = %q, want empty", stderr.String())
   227		}
   228	}
   229	
   230	type fakeLegacyRuntime struct {
   231		executable      string
   232		pairHome        string
   233		defaultPairHome string
   234		roots           map[string]bool
   235		statErr         error
   236		execCode        int
   237		statCalls       int
   238		embeddedRoot    string
   239		embeddedErr     error
   240		environ         []string
   241	
   242		execPath  string
   243		execLabel string
   244		execArgv  []string
   245		execEnv   []string
   246	}
   247	
   248	func (f *fakeLegacyRuntime) Executable() (string, error) {
   249		if f.executable == "" {
   250			return "", errors.New("missing executable")
   251		}
   252		return f.executable, nil
   253	}
   254	
   255	func (f *fakeLegacyRuntime) PairHome() string {
   256		return f.pairHome
   257	}
   258	
   259	func (f *fakeLegacyRuntime) DefaultPairHome() string {
   260		return f.defaultPairHome
   261	}
   262	
   263	func (f *fakeLegacyRuntime) Stat(path string) error {
   264		f.statCalls++
   265		if f.statErr != nil {
   266			return f.statErr
   267		}
   268		if strings.HasSuffix(path, "/bin/pair-shell") && f.roots != nil {
   269			root := strings.TrimSuffix(path, "/bin/pair-shell")
   270			if f.roots[root] {
   271				return nil
   272			}
   273		}
   274		return os.ErrNotExist
   275	}
   276	
   277	func (f *fakeLegacyRuntime) Environ() []string {
   278		if f.environ != nil {
   279			return f.environ
   280		}
   281		return os.Environ()
   282	}
   283	
   284	func (f *fakeLegacyRuntime) EmbeddedAssetRoot() (string, error) {
   285		return f.embeddedRoot, f.embeddedErr
   286	}
   287	
   288	func (f *fakeLegacyRuntime) Exec(label string, path string, argv []string, env []string) int {
   289		f.execLabel = label
   290		f.execPath = path
   291		f.execArgv = append([]string(nil), argv...)
   292		f.execEnv = append([]string(nil), env...)
   293		return f.execCode
   294	}
   295	
   296	func containsEnv(env []string, want string) bool {
   297		for _, got := range env {
   298			if got == want {
   299				return true
   300			}
   301		}
   302		return false
   303	}

codex
I’m going to run a small targeted probe for the `PAIR_DATA_DIR` contract rather than rely on static reading. This should show where a copied binary extracts when `PAIR_DATA_DIR` and `XDG_DATA_HOME` disagree.
exec
/bin/zsh -lc 'tmp=$(mktemp -d "${TMPDIR:-/tmp}/pair-review-data-dir.XXXXXX"); trap '"'rm -rf \""'$tmp"'"' EXIT; make runtimebundle-generate >/dev/null; go build -o \""'$tmp/pair" ./cmd/pair-go; HOME="$tmp/home" XDG_DATA_HOME="$tmp/xdg" PAIR_DATA_DIR="$tmp/custom-data" PAIR_HOME= "$tmp/pair" --help >/dev/null; printf '"'custom runtime dirs:\\n'; find \""'$tmp/custom-data" -maxdepth 3 -type d 2>/dev/null | sort || true; printf '"'xdg runtime dirs:\\n'; find \""'$tmp/xdg" -maxdepth 4 -type d 2>/dev/null | sort || true' in /Users/xianxu/workspace/pair
 succeeded in 1347ms:
go: writing stat cache: open /Users/xianxu/go/pkg/mod/cache/download/github.com/xianxu/pair/@v/v0.0.0-20260701074753-cfb9a21ce58b.info582241610.tmp: operation not permitted
custom runtime dirs:
xdg runtime dirs:
/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T//pair-review-data-dir.9W6ZPe/xdg
/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T//pair-review-data-dir.9W6ZPe/xdg/pair
/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T//pair-review-data-dir.9W6ZPe/xdg/pair/runtime
/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T//pair-review-data-dir.9W6ZPe/xdg/pair/runtime/c45fe52e404662cfe904cee6591fcda4cfcfa9201a5bcb75145a1eb08a674816
/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T//pair-review-data-dir.9W6ZPe/xdg/pair/runtime/c45fe52e404662cfe904cee6591fcda4cfcfa9201a5bcb75145a1eb08a674816/pair-home

exec
/bin/zsh -lc "nl -ba workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md | sed -n '1,150p'" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "nl -ba workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md | sed -n '250,330p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
   250	Update `PAIR_GO_SRCS` so changes to runtimebundle sources rebuild `bin/pair` and `bin/pair-go`. Add `test-runtimebundle` and `runtimebundle-drift-check`; include both in the relevant test path. Ensure `make build` builds helper binaries first, regenerates the asset tree, then builds `pair`/`pair-go` so the single binary embeds current helpers without committing generated binaries.
   251	
   252	- [x] **Step 5: Run green tests**
   253	
   254	Run: `make test-runtimebundle`
   255	
   256	Expected: pass.
   257	
   258	### Task 4: Implement filesystem extraction
   259	
   260	**Files:**
   261	- Create: `cmd/internal/runtimebundle/store.go`
   262	- Create: `cmd/internal/runtimebundle/store_test.go`
   263	
   264	- [x] **Step 1: Write failing filesystem tests**
   265	
   266	Use `t.TempDir()` and a fake embedded asset reader. Assert atomic first-run extraction, executable bits for `bin/*` scripts, idempotent second run, manifest marker write, and stale cleanup that preserves the selected runtime.
   267	
   268	- [x] **Step 2: Run the red tests**
   269	
   270	Run: `go test ./cmd/internal/runtimebundle -run 'TestStore|TestExtract' -count=1`
   271	
   272	Expected: missing store implementation.
   273	
   274	- [x] **Step 3: Implement the thin IO shell**
   275	
   276	Use temp files plus rename for file writes. Write a manifest marker after all assets land. Apply cleanup after successful extraction/selection, not before.
   277	
   278	- [x] **Step 4: Run green tests**
   279	
   280	Run: `make test-runtimebundle`
   281	
   282	Expected: pass.
   283	
   284	## Chunk 3: Launch Integration And Smoke Coverage
   285	
   286	### Task 5: Extend asset-root selection to embedded fallback
   287	
   288	**Files:**
   289	- Modify: `cmd/internal/entrypoint/asset_root.go`
   290	- Modify: `cmd/internal/entrypoint/asset_root_test.go`
   291	- Modify: `cmd/pair-go/main.go`
   292	- Modify: `cmd/pair-go/main_test.go`
   293	
   294	- [x] **Step 1: Write failing tests for launch-root precedence**
   295	
   296	Assert ordering:
   297	
   298	1. `PAIR_HOME` with `bin/pair-shell` wins.
   299	2. Executable sibling root wins.
   300	3. `defaultPairHome` wins.
   301	4. Embedded extracted runtime is used only when the adjacent/default candidates fail.
   302	
   303	- [x] **Step 2: Write failing `cmd/pair-go` fake-runtime tests**
   304	
   305	Extend the existing `legacyRuntime` fake so launch records `PAIR_HOME=<extracted-root>` in the exec env when embedded fallback is selected.
   306	
   307	- [x] **Step 3: Run red tests**
   308	
   309	Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -run 'TestResolveAssetRoot|TestRun' -count=1`
   310	
   311	Expected: embedded fallback behavior missing.
   312	
   313	- [x] **Step 4: Implement embedded fallback**
   314	
   315	Keep `ResolveAssetRoot` deterministic and side-effect-free by passing the already-prepared embedded root as an optional candidate. Let `cmd/pair-go` decide whether extraction is needed and then pass the selected root into the existing legacy launch handoff.
   316	
   317	- [x] **Step 5: Run green tests**
   318	
   319	Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
   320	
   321	Expected: pass.
   322	
   323	### Task 6: Add copied-binary and upgrade smoke tests
   324	
   325	**Files:**
   326	- Create: `tests/pair-embedded-runtime-test.sh`
   327	- Modify: `Makefile.local`
   328	
   329	- [x] **Step 1: Write the failing clean-copy help smoke test**
   330	

 succeeded in 0ms:
     1	# Self-Contained Pair Binary Implementation Plan
     2	
     3	> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [x]`) syntax for tracking.
     4	
     5	**Goal:** Build a `pair` binary that can launch from a clean copied location by extracting embedded Pair-owned runtime assets into a versioned runtime root.
     6	
     7	**Architecture:** Add a generated runtime manifest as the single source for packaged assets (`ARCH-DRY`), a pure `runtimebundle` package for manifest hashing, version selection, extraction planning, and cleanup planning (`ARCH-PURE`), and a thin `cmd/pair-go` integration that only falls back to embedded extraction when `PAIR_HOME`, executable siblings, and `defaultPairHome` do not provide `bin/pair-shell`. This satisfies the issue purpose by proving copied-binary launch without relying on a source checkout (`ARCH-PURPOSE`).
     8	
     9	**Tech Stack:** Go 1.26, `embed.FS`, Makefile build rules, POSIX shell smoke tests, existing `cmd/internal/entrypoint` launch handoff.
    10	
    11	---
    12	
    13	## Core Concepts
    14	
    15	### Pure Entities
    16	
    17	| Name | Lives in | Status |
    18	|------|----------|--------|
    19	| `RuntimeAsset` | `cmd/internal/runtimebundle/manifest.go` | new |
    20	| `RuntimeManifest` | `cmd/internal/runtimebundle/manifest.go` | new |
    21	| `ExtractionPlan` | `cmd/internal/runtimebundle/plan.go` | new |
    22	| `RuntimeSelection` | `cmd/internal/entrypoint/asset_root.go` | modified |
    23	| `CleanupPlan` | `cmd/internal/runtimebundle/cleanup.go` | new |
    24	
    25	- **RuntimeAsset** — one embedded file entry: logical path, mode, size, and digest.
    26	  - **Relationships:** N:1 with `RuntimeManifest`; one manifest owns many assets.
    27	  - **DRY rationale:** Avoid parallel hand-maintained asset lists for embedding, install notes, and tests.
    28	  - **Future extensions:** Can add generated-command shim metadata when helper binaries move behind dispatcher routes.
    29	
    30	- **RuntimeManifest** — the deterministic package contract for Pair-owned runtime files.
    31	  - **Relationships:** 1:N with `RuntimeAsset`; 1:1 with an embedded runtime version/digest.
    32	  - **DRY rationale:** Single source for extraction, clean-copy smoke expectations, and atlas/Homebrew docs.
    33	  - **Future extensions:** Can widen from static file assets to generated native assets.
    34	
    35	- **ExtractionPlan** — pure decision for which files to write, chmod, leave alone, or reject.
    36	  - **Relationships:** 1:1 with one target runtime root and one manifest.
    37	  - **DRY rationale:** Keeps idempotency/upgrade rules out of filesystem code and makes them directly unit-testable.
    38	  - **Future extensions:** Can add partial refresh or corrupted-file repair without changing callers.
    39	
    40	- **RuntimeSelection** — existing launch-root decision extended to include an optional embedded runtime root after adjacent candidates fail.
    41	  - **Relationships:** `cmd/pair-go` asks for one `RuntimeSelection`; it yields the `AssetRoot` passed to `ResolveLegacyLaunch`.
    42	  - **DRY rationale:** Reuse `ResolveAssetRoot` ordering instead of adding a second launch resolver.
    43	  - **Future extensions:** Can remove adjacent/default fallbacks when embedded mode becomes the primary release shape.
    44	
    45	- **CleanupPlan** — pure stale-runtime pruning decision that never returns the currently selected runtime.
    46	  - **Relationships:** N old runtime dirs to zero or more delete candidates.
    47	  - **DRY rationale:** Shared cleanup policy for first-run extraction and future maintenance commands.
    48	  - **Future extensions:** Retention count/age can become configurable without touching extraction.
    49	
    50	### Integration Points
    51	
    52	| Name | Lives in | Status | Wraps |
    53	|------|----------|--------|-------|
    54	| `RuntimeFS` | `cmd/internal/runtimebundle/embed.go` | new | `embed.FS` |
    55	| `RuntimeStore` | `cmd/internal/runtimebundle/store.go` | new | filesystem writes, chmod, rename |
    56	| `PairGoLegacyRuntime` | `cmd/pair-go/main.go` | modified | `os`, `syscall.Exec`, env |
    57	| `BuildManifest` | `cmd/internal/runtimebundle/generate.go`, `cmd/internal/runtimebundle/assets/**`, `Makefile.local` | new | repo file tree at build time |
    58	| `CopiedBinarySmoke` | `tests/pair-embedded-runtime-test.sh` | new | real built binary with fake external commands |
    59	
    60	- **RuntimeFS** — exposes embedded asset bytes and manifest data.
    61	  - **Injected into:** `RuntimeStore` extraction runner and `cmd/pair-go` runtime setup.
    62	  - **Future extensions:** Swap generated bytes for generated config content without changing launch.
    63	
    64	- **RuntimeStore** — applies `ExtractionPlan` atomically under the user Pair data/cache area.
    65	  - **Injected into:** `cmd/pair-go` through the existing `legacyRuntime` seam.
    66	  - **Future extensions:** Add lock-file serialization if concurrent first-runs race on the same runtime root.
    67	
    68	- **PairGoLegacyRuntime** — OS-backed seam that resolves paths, environment, extraction, and final exec.
    69	  - **Injected into:** `runWithLegacyRuntime` tests via fake runtime.
    70	  - **Future extensions:** Native launcher can reuse the same data/cache path helpers.
    71	
    72	- **BuildManifest** — Makefile/generator path that keeps embedded assets synchronized with repo runtime files.
    73	  - **Injected into:** `go generate` or `make build` before compiling `cmd/pair-go`.
    74	  - **Future extensions:** Homebrew can audit the manifest rather than restating asset paths.
    75	
    76	#### Runtime Asset Boundary
    77	
    78	The embedded bundle must be a generated, gitignored package-local asset tree under
    79	`cmd/internal/runtimebundle/assets/`; Go's `embed` package reads that tree at
    80	compile time. The generator is the single source of the file list (`ARCH-DRY`):
    81	tests and docs audit the generated `RuntimeManifest`, not a second list.
    82	
    83	Required inputs:
    84	
    85	- `bin/pair-shell`, `bin/pair-help`, `bin/pair-notify`, `bin/pair-quit.sh`,
    86	  `bin/pair-restart.sh`, `bin/pair-scrollback-open`,
    87	  `bin/pair-changelog-open`, `bin/pair-review-open`,
    88	  `bin/pair-review-readiness`, `bin/pair-review-target`,
    89	  `bin/copy-on-select.sh`, `bin/clipboard-to-pane.sh`, `bin/flash-pane.sh`,
    90	  `bin/pair-title.sh`, `bin/pair-session-watch.sh`, and `bin/lib/**`.
    91	- Go-built helper binaries used by the retained shell/KDL/Lua runtime:
    92	  `bin/pair-wrap`, `bin/pair-slug`, `bin/pair-context`,
    93	  `bin/pair-scrollback-render`, `bin/pair-changelog`,
    94	  `bin/pair-continuation`, and `bin/pair-session-watch`.
    95	- Native assets `nvim/**`, `zellij/**`, and runtime diagnostic assets
    96	  `doctor/README.md`, `doctor/SKILL.md`, `doctor/doctor.sh`,
    97	  `doctor/emitter-health.sh`.
    98	
    99	Excluded inputs:
   100	
   101	- Public entrypoints `bin/pair`, `bin/pair-go`, and `bin/pair-dev` to avoid
   102	  embedding the launching binary or dev-only wrapper.
   103	- External programs (`zellij`, `nvim`, `fzf`, `jq`, clipboard tools, agent CLIs),
   104	  generated caches (`__pycache__`, `.DS_Store`), test files, and workflow/docs
   105	  that are not read by launch/session flows.
   106	
   107	Generator contract:
   108	
   109	- `make runtimebundle-generate` removes and rebuilds the gitignored asset tree
   110	  deterministically after helper binaries are built.
   111	- `make runtimebundle-drift-check` regenerates twice into temp dirs and compares
   112	  manifest JSON plus file bytes/modes; nondeterminism fails.
   113	- `make test-runtimebundle` depends on helper binaries plus
   114	  `runtimebundle-generate`, then runs `go test ./cmd/internal/runtimebundle
   115	  -count=1`. After `embed.go` exists, use this target for package tests from a
   116	  clean checkout; raw `go test ./cmd/internal/runtimebundle` is only valid after
   117	  the asset tree has been generated.
   118	- `make build` depends on `runtimebundle-generate` before building `bin/pair` and
   119	  `bin/pair-go`; helper binaries are built first so the embedded bundle contains
   120	  the current helper executables.
   121	
   122	Runtime root contract:
   123	
   124	- The extracted root is `$PAIR_DATA_DIR/runtime/<manifest-digest>/pair-home`,
   125	  where `<manifest-digest>` is the `RuntimeManifest` digest. The directory that
   126	  cleanup treats as one runtime generation is `$PAIR_DATA_DIR/runtime/<digest>/`.
   127	- Extraction writes `$PAIR_DATA_DIR/runtime/<digest>/manifest.json` after all
   128	  assets land under `pair-home/`; the marker includes the manifest digest,
   129	  generated-at build metadata if available, and the asset count.
   130	- Cleanup only considers direct children of `$PAIR_DATA_DIR/runtime/` whose names
   131	  are valid manifest digests and whose `manifest.json` parses as a Pair runtime
   132	  marker. It must always exclude the selected `<digest>` generation, even if the
   133	  generation is older than the retention policy.
   134	- Tests and production code derive behavior from the generated `RuntimeManifest`
   135	  and runtime marker only. `atlas/go-migration-inventory.md` remains review
   136	  context and documentation; no code or automated test should parse atlas as an
   137	  input source.
   138	
   139	- **CopiedBinarySmoke** — process-level fake test for external dependencies and copied-binary launch.
   140	  - **Injected into:** `make test` via a focused target.
   141	  - **Future extensions:** Can become the release packaging smoke for Homebrew and tarball builds.
   142	
   143	## Chunk 1: Manifest And Pure Runtime Planning
   144	
   145	### Task 1: Define the manifest contract
   146	
   147	**Files:**
   148	- Create: `cmd/internal/runtimebundle/manifest.go`
   149	- Create: `cmd/internal/runtimebundle/manifest_test.go`
   150	

exec
/bin/zsh -lc "nl -ba workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md | sed -n '1,130p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	---
     2	id: 000090
     3	status: working
     4	deps: []
     5	github_issue:
     6	created: 2026-07-01
     7	updated: 2026-07-01
     8	estimate_hours: 5.44
     9	started: 2026-07-01T00:18:42-07:00
    10	---
    11	
    12	# self-contained pair binary with embedded runtime assets
    13	
    14	## Problem
    15	
    16	After #79, the public `pair` command is Go-owned, but deployment is still an
    17	installed tree: the Go entrypoint must find adjacent Pair-owned runtime assets
    18	such as `bin/pair-shell`, shell helpers, `nvim/`, `zellij/`, and helper
    19	binaries. That is simpler for Homebrew, but it is not the deployment shape we
    20	eventually want: copying one Pair binary around and having it work.
    21	
    22	The long-term direction is a true native single binary. Rewriting every
    23	remaining shell and orchestration surface directly into Go is too much risk in
    24	one jump, so the next step should make the current runtime tree derive from one
    25	Go artifact without pretending the shell lifecycle is already gone.
    26	
    27	## Spec
    28	
    29	Add a self-contained deployment mode for the Go `pair` binary:
    30	
    31	- Embed the Pair-owned runtime assets needed by launch/session flows into the Go
    32	  binary.
    33	- On first run, extract those assets to a versioned runtime directory under the
    34	  user's Pair data/cache area, then run the existing launch flow with `PAIR_HOME`
    35	  pointed at that extracted runtime root.
    36	- Preserve the current adjacent-install behavior for source checkout and
    37	  Homebrew layouts unless/until the self-contained mode proves it can replace
    38	  them.
    39	- Keep external programs external: `zellij`, `nvim`, `fzf`, `jq` while the shell
    40	  runtime still needs it, clipboard tools, and agent CLIs are not bundled by this
    41	  issue.
    42	- Make runtime extraction deterministic, idempotent, and upgrade-safe: a new
    43	  binary/runtime version should extract a new directory or refresh only when the
    44	  embedded manifest changes.
    45	- Add a cleanup policy for stale extracted runtimes that cannot delete the
    46	  currently running runtime.
    47	
    48	Execution path toward the native single binary:
    49	
    50	1. Embed and extract the existing runtime tree. This delivers the "single Pair
    51	   artifact" deployment option while retaining the tested shell/nvim/zellij
    52	   contracts.
    53	2. Route generated internal calls through the Go dispatcher where possible
    54	   (`pair wrap`, `pair slug`, `pair changelog`, `pair continuation`, etc.) while
    55	   keeping compatibility names only as shims.
    56	3. Port stateful shell orchestrators into Go one at a time: launcher/session
    57	   lifecycle, scrollback/changelog openers, title poller, review helpers,
    58	   clipboard helpers.
    59	4. Once shell ownership is gone, stop extracting shell scripts and use embedded
    60	   or generated native assets only for `nvim/` and `zellij/`.
    61	5. Revisit whether `nvim/` and `zellij/` remain extracted native assets or move
    62	   to generated temp files/API-driven startup. The native single binary target
    63	   is one Pair executable, with external platform tools still supplied by the
    64	   system.
    65	
    66	Architecture notes:
    67	
    68	- `ARCH-PURPOSE`: the copied binary must be enough to provide Pair-owned
    69	  runtime assets; falling back to a source checkout does not satisfy this issue.
    70	- `ARCH-DRY`: the embedded runtime manifest must be the single source of what is
    71	  packaged, installed, and tested. Do not maintain a separate hand-written asset
    72	  list for Homebrew, tests, and extraction.
    73	- `ARCH-PURE`: keep manifest planning, runtime selection, and extraction
    74	  decisions as pure functions with unit tests; keep filesystem writes and
    75	  process exec in thin seams.
    76	
    77	## Done when
    78	
    79	- [x] A release build can produce one `pair` binary that contains the Pair-owned
    80	      runtime assets needed for launch/session flows.
    81	- [x] Copying only that binary to a clean path works when external dependencies
    82	      are installed.
    83	- [x] First run extracts or refreshes a versioned runtime root and points
    84	      `PAIR_HOME` at it for the compatibility launch handoff.
    85	- [x] Adjacent source/Homebrew layouts still work.
    86	- [x] Upgrade and stale-runtime cleanup behavior is tested.
    87	- [x] The execution path toward the true native single binary is documented in
    88	      atlas.
    89	
    90	## Plan
    91	
    92	- [x] Define the embedded runtime manifest and generated asset list.
    93	- [x] Implement runtime extraction and version/manifest selection.
    94	- [x] Wire `cmd/pair-go` to prefer extracted embedded runtime when no adjacent
    95	      asset root exists.
    96	- [x] Add install/copy smoke tests for clean and upgrade paths.
    97	- [x] Update README, atlas, and Homebrew packaging notes.
    98	
    99	Detailed implementation plan:
   100	`workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md`.
   101	
   102	## Estimate
   103	
   104	Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
   105	`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
   106	source as stale, so the number is provisional but uses the required method.
   107	
   108	```estimate
   109	model: estimate-logic-v3.1
   110	familiarity: 1.0
   111	item: issue-spec design=0.20 impl=0.08
   112	item: greenfield-go-module design=0.60 impl=0.56
   113	item: smaller-go-module design=0.35 impl=0.48
   114	item: cross-cutting-refactor design=0.80 impl=1.12
   115	item: atlas-docs design=0.25 impl=0.20
   116	item: milestone-review design=0.00 impl=0.20
   117	design-buffer: 0.15
   118	total: 5.44
   119	```
   120	
   121	## Log
   122	
   123	### 2026-07-01
   124	
   125	Created after #79 closed: #79 made `pair` Go-owned but intentionally retained
   126	the adjacent runtime tree. The desired final direction is a true native single
   127	binary; this issue captures the lower-risk next step of embedding/extracting
   128	the current runtime tree first.
   129	
   130	Claimed and entered planning. `sdlc start-plan --issue 90` delivered

exec
/bin/zsh -lc "nl -ba cmd/internal/runtimebundle/cleanup.go | sed -n '1,90p'; nl -ba cmd/internal/runtimebundle/cleanup_test.go | sed -n '1,80p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package runtimebundle
     2	
     3	import (
     4		"fmt"
     5		"sort"
     6	)
     7	
     8	type RuntimeGeneration struct {
     9		Digest    string
    10		HasMarker bool
    11		ModUnix   int64
    12	}
    13	
    14	type CleanupInput struct {
    15		SelectedDigest string
    16		Keep           int
    17		Generations    []RuntimeGeneration
    18	}
    19	
    20	type CleanupPlan struct {
    21		DeleteDigests []string
    22	}
    23	
    24	func PlanCleanup(input CleanupInput) (CleanupPlan, error) {
    25		if input.SelectedDigest == "" {
    26			return CleanupPlan{}, fmt.Errorf("selected digest is required")
    27		}
    28		if input.Keep < 0 {
    29			return CleanupPlan{}, fmt.Errorf("keep must be non-negative")
    30		}
    31		candidates := make([]RuntimeGeneration, 0, len(input.Generations))
    32		for _, gen := range input.Generations {
    33			if gen.Digest == input.SelectedDigest || !gen.HasMarker || !isDigestName(gen.Digest) {
    34				continue
    35			}
    36			candidates = append(candidates, gen)
    37		}
    38		sort.Slice(candidates, func(i, j int) bool {
    39			if candidates[i].ModUnix == candidates[j].ModUnix {
    40				return candidates[i].Digest > candidates[j].Digest
    41			}
    42			return candidates[i].ModUnix > candidates[j].ModUnix
    43		})
    44		if input.Keep >= len(candidates) {
    45			return CleanupPlan{}, nil
    46		}
    47		deleteCandidates := candidates[input.Keep:]
    48		sort.Slice(deleteCandidates, func(i, j int) bool {
    49			return deleteCandidates[i].Digest < deleteCandidates[j].Digest
    50		})
    51		plan := CleanupPlan{DeleteDigests: make([]string, 0, len(deleteCandidates))}
    52		for _, gen := range deleteCandidates {
    53			plan.DeleteDigests = append(plan.DeleteDigests, gen.Digest)
    54		}
    55		return plan, nil
    56	}
    57	
    58	func isDigestName(s string) bool {
    59		if s == "" {
    60			return false
    61		}
    62		for _, r := range s {
    63			if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
    64				return false
    65			}
    66		}
    67		return true
    68	}
     1	package runtimebundle
     2	
     3	import "testing"
     4	
     5	func TestPlanCleanupKeepsSelectedRuntime(t *testing.T) {
     6		plan, err := PlanCleanup(CleanupInput{
     7			SelectedDigest: "bbbb",
     8			Keep:           1,
     9			Generations: []RuntimeGeneration{
    10				{Digest: "aaaa", HasMarker: true, ModUnix: 10},
    11				{Digest: "bbbb", HasMarker: true, ModUnix: 1},
    12				{Digest: "cccc", HasMarker: true, ModUnix: 20},
    13			},
    14		})
    15		if err != nil {
    16			t.Fatalf("PlanCleanup error = %v", err)
    17		}
    18		if len(plan.DeleteDigests) != 1 || plan.DeleteDigests[0] != "aaaa" {
    19			t.Fatalf("DeleteDigests = %#v, want only aaaa", plan.DeleteDigests)
    20		}
    21	}
    22	
    23	func TestPlanCleanupIgnoresNonRuntimeDirectories(t *testing.T) {
    24		plan, err := PlanCleanup(CleanupInput{
    25			SelectedDigest: "bbbb",
    26			Keep:           0,
    27			Generations: []RuntimeGeneration{
    28				{Digest: "not-a-digest", HasMarker: true, ModUnix: 10},
    29				{Digest: "aaaa", HasMarker: false, ModUnix: 20},
    30				{Digest: "bbbb", HasMarker: true, ModUnix: 1},
    31				{Digest: "cccc", HasMarker: true, ModUnix: 30},
    32			},
    33		})
    34		if err != nil {
    35			t.Fatalf("PlanCleanup error = %v", err)
    36		}
    37		if len(plan.DeleteDigests) != 1 || plan.DeleteDigests[0] != "cccc" {
    38			t.Fatalf("DeleteDigests = %#v, want only cccc", plan.DeleteDigests)
    39		}
    40	}

codex
```verdict
verdict: REWORK
confidence: high
```

The embedded-runtime path is well structured overall: the pure planning pieces are separated from IO, copied-binary smoke coverage exists, and atlas/README were updated. It should not cross the whole-issue boundary yet because the implementation misses part of the documented runtime-root contract and the plan’s Core Concepts table no longer matches the code.

1. Strengths:
- `cmd/internal/runtimebundle/manifest.go:23` keeps manifest validation/digesting pure and order-independent.
- `cmd/internal/runtimebundle/plan.go:24` models extraction writes/skips as data before touching the filesystem.
- `cmd/pair-go/main.go:61` reuses `entrypoint.ResolveAssetRoot` for embedded fallback instead of adding a parallel resolver.
- `atlas/architecture.md` and `atlas/go-migration-inventory.md` document the new embedded fallback and adjacent-layout compatibility.

2. Critical findings:
- `cmd/pair-go/main.go:125`: embedded extraction ignores `PAIR_DATA_DIR`, but the plan contract says extraction root is `$PAIR_DATA_DIR/runtime/<digest>/pair-home` (`workshop/plans/...plan.md:122`). I verified a copied binary with `PAIR_DATA_DIR=/tmp/custom-data` and `XDG_DATA_HOME=/tmp/xdg` extracts under `XDG_DATA_HOME`, leaving `PAIR_DATA_DIR` empty. Fix by resolving store root from `PAIR_DATA_DIR` first, then XDG/home fallback, and add a fake-runtime or smoke assertion where `PAIR_DATA_DIR` and `XDG_DATA_HOME` differ. (`ARCH-PURPOSE`)
- `workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:57`: Core Concepts says `BuildManifest` lives in `cmd/internal/runtimebundle/generate.go`, but the implementation is `cmd/internal/runtimebundlegen/generate.go` plus `cmd/internal/runtimebundle/generatecmd/main.go`. Per the boundary prompt’s Core Concepts cross-check, this table/code contradiction needs a `## Revisions` entry or the code moved to the stated path.

3. Important findings:
- `Makefile.local:281`: `PAIR_GO_SRCS` omits `cmd/internal/runtimebundle/store.go`, even though `cmd/pair-go` compiles and calls `runtimebundle.Extract`. A store-only change will not rebuild `bin/pair`/`bin/pair-go` via make, violating the plan item at `workshop/plans/...plan.md:250`. Add `store.go` to the prerequisite list.
- `cmd/internal/runtimebundle/cleanup.go:58`: `isDigestName` accepts any non-empty lowercase hex string, and tests use `aaaa`/`bbbb` (`cleanup_test.go:7`). The runtime contract says cleanup only considers valid manifest digests; these are SHA-256 hex strings. Require 64 hex chars, and ideally validate marker digest matches the directory name.
- `cmd/internal/runtimebundlegen/generate.go:16` duplicates the `RuntimeAsset`/`RuntimeManifest` schema from `cmd/internal/runtimebundle/manifest.go:11`. This is a small but real `ARCH-DRY` drift risk for the manifest JSON contract. Prefer reusing the runtimebundle types from the generator package.

4. Minor findings:
- `Makefile.local:96`: `runtimebundle-drift-check` uses `diff -qr`, which compares names/content but not file modes despite the plan saying bytes/modes are checked.
- `tests/pair-embedded-runtime-test.sh:102`: the smoke proves zellij receives extracted config/layout paths, but it does not execute the layout’s `nvim`/agent commands; keep that limitation explicit if not expanding the fake.

5. Test coverage notes:
- Add coverage for `PAIR_DATA_DIR` precedence during embedded extraction.
- Add cleanup tests using real 64-character manifest digests and rejecting short hex names.
- Add a make/build dependency regression if the repo has an existing pattern for generated prerequisite checks.

6. Architectural notes for upcoming work:
- `ARCH-DRY`: flag until manifest schema duplication is removed or intentionally justified.
- `ARCH-PURE`: pass; the main planning decisions are pure and directly tested.
- `ARCH-PURPOSE`: flag until copied-binary extraction honors the documented Pair data-dir contract.

7. Plan revision recommendations:
- Add `## Revisions` noting that `BuildManifest` was implemented at `cmd/internal/runtimebundlegen/generate.go` / `cmd/internal/runtimebundle/generatecmd/main.go`, or move the code to match the table.
- If the intended contract is XDG-only rather than `PAIR_DATA_DIR`, revise the runtime-root contract and issue log explicitly; otherwise fix implementation/tests and leave the plan as-is.
tokens used
91,731
```verdict
verdict: REWORK
confidence: high
```

The embedded-runtime path is well structured overall: the pure planning pieces are separated from IO, copied-binary smoke coverage exists, and atlas/README were updated. It should not cross the whole-issue boundary yet because the implementation misses part of the documented runtime-root contract and the plan’s Core Concepts table no longer matches the code.

1. Strengths:
- `cmd/internal/runtimebundle/manifest.go:23` keeps manifest validation/digesting pure and order-independent.
- `cmd/internal/runtimebundle/plan.go:24` models extraction writes/skips as data before touching the filesystem.
- `cmd/pair-go/main.go:61` reuses `entrypoint.ResolveAssetRoot` for embedded fallback instead of adding a parallel resolver.
- `atlas/architecture.md` and `atlas/go-migration-inventory.md` document the new embedded fallback and adjacent-layout compatibility.

2. Critical findings:
- `cmd/pair-go/main.go:125`: embedded extraction ignores `PAIR_DATA_DIR`, but the plan contract says extraction root is `$PAIR_DATA_DIR/runtime/<digest>/pair-home` (`workshop/plans/...plan.md:122`). I verified a copied binary with `PAIR_DATA_DIR=/tmp/custom-data` and `XDG_DATA_HOME=/tmp/xdg` extracts under `XDG_DATA_HOME`, leaving `PAIR_DATA_DIR` empty. Fix by resolving store root from `PAIR_DATA_DIR` first, then XDG/home fallback, and add a fake-runtime or smoke assertion where `PAIR_DATA_DIR` and `XDG_DATA_HOME` differ. (`ARCH-PURPOSE`)
- `workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:57`: Core Concepts says `BuildManifest` lives in `cmd/internal/runtimebundle/generate.go`, but the implementation is `cmd/internal/runtimebundlegen/generate.go` plus `cmd/internal/runtimebundle/generatecmd/main.go`. Per the boundary prompt’s Core Concepts cross-check, this table/code contradiction needs a `## Revisions` entry or the code moved to the stated path.

3. Important findings:
- `Makefile.local:281`: `PAIR_GO_SRCS` omits `cmd/internal/runtimebundle/store.go`, even though `cmd/pair-go` compiles and calls `runtimebundle.Extract`. A store-only change will not rebuild `bin/pair`/`bin/pair-go` via make, violating the plan item at `workshop/plans/...plan.md:250`. Add `store.go` to the prerequisite list.
- `cmd/internal/runtimebundle/cleanup.go:58`: `isDigestName` accepts any non-empty lowercase hex string, and tests use `aaaa`/`bbbb` (`cleanup_test.go:7`). The runtime contract says cleanup only considers valid manifest digests; these are SHA-256 hex strings. Require 64 hex chars, and ideally validate marker digest matches the directory name.
- `cmd/internal/runtimebundlegen/generate.go:16` duplicates the `RuntimeAsset`/`RuntimeManifest` schema from `cmd/internal/runtimebundle/manifest.go:11`. This is a small but real `ARCH-DRY` drift risk for the manifest JSON contract. Prefer reusing the runtimebundle types from the generator package.

4. Minor findings:
- `Makefile.local:96`: `runtimebundle-drift-check` uses `diff -qr`, which compares names/content but not file modes despite the plan saying bytes/modes are checked.
- `tests/pair-embedded-runtime-test.sh:102`: the smoke proves zellij receives extracted config/layout paths, but it does not execute the layout’s `nvim`/agent commands; keep that limitation explicit if not expanding the fake.

5. Test coverage notes:
- Add coverage for `PAIR_DATA_DIR` precedence during embedded extraction.
- Add cleanup tests using real 64-character manifest digests and rejecting short hex names.
- Add a make/build dependency regression if the repo has an existing pattern for generated prerequisite checks.

6. Architectural notes for upcoming work:
- `ARCH-DRY`: flag until manifest schema duplication is removed or intentionally justified.
- `ARCH-PURE`: pass; the main planning decisions are pure and directly tested.
- `ARCH-PURPOSE`: flag until copied-binary extraction honors the documented Pair data-dir contract.

7. Plan revision recommendations:
- Add `## Revisions` noting that `BuildManifest` was implemented at `cmd/internal/runtimebundlegen/generate.go` / `cmd/internal/runtimebundle/generatecmd/main.go`, or move the code to match the table.
- If the intended contract is XDG-only rather than `PAIR_DATA_DIR`, revise the runtime-root contract and issue log explicitly; otherwise fix implementation/tests and leave the plan as-is.

---

## Re-review — 2026-07-01T00:59:07-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 90 — self-contained pair binary with embedded runtime assets |
| repo | pair |
| issue file | workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md |
| boundary | whole-issue close |
| milestone | — |
| window | e1ae416c6084399c155e0a89c5597712f679460b..HEAD |
| command | sdlc close --issue 90 |
| reviewer | codex |
| timestamp | 2026-07-01T00:59:07-07:00 |
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
session id: 019f1cae-a856-7a42-8104-3acf9ba4b67e
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **pair** repository.

- repository: pair   (root: /Users/xianxu/workspace/pair)
- issue:      pair#90   (file: workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md)
- window:     Base: e1ae416c6084399c155e0a89c5597712f679460b   Head: HEAD

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
index 18ff43e..a8ea612 100644
--- a/.gitignore
+++ b/.gitignore
@@ -37,3 +37,4 @@ cache/
 /GEMINI.md
 /.agents/skills/
 /construct/generated/
+/cmd/internal/runtimebundle/assets/
diff --git a/Makefile.local b/Makefile.local
index d2383b8..5b50439 100644
--- a/Makefile.local
+++ b/Makefile.local
@@ -1,7 +1,7 @@
 # Repo-specific Makefile targets.
 # This file is included by Makefile — add your own targets here.
 
-.PHONY: build install pair pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-session-watch pair-go test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
+.PHONY: build install pair pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-session-watch pair-go runtimebundle-generate runtimebundle-drift-check test-runtimebundle test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-pair-embedded-runtime test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
 
 # Cross-cutting Go-binary targets. Every Go binary lives under cmd/<name>/
 # and ships with the `pair-` prefix so it doesn't collide with anything
@@ -29,6 +29,8 @@ INSTALL_DIR := $(HOME)/.local/bin
 GO_BINS     := pair pair-scribe pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-session-watch pair-go
 SHELL_BINS  := pair-dev
 PAIR_HOME_LDFLAGS := -X main.defaultPairHome=$(CURDIR)
+RUNTIMEBUNDLE_HELPERS := pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-session-watch
+RUNTIMEBUNDLE_ASSETS := cmd/internal/runtimebundle/assets/runtime/manifest.json
 
 build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
 
@@ -65,7 +67,7 @@ pair-go:                $(BIN_DIR)/pair-go
 #                     inside the vt.Emulator library API itself, which is
 #                     outside our scope to fix — exclude that package
 #                     here rather than chase a library-level race.
-test: test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace test-review
+test: test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-pair-embedded-runtime test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace test-review
 	go test ./... -count=1
 
 # pair-title poller: single-instance guard (identity-checked liveness so a
@@ -86,6 +88,24 @@ test-cmux-ownership: $(BIN_DIR)/pair
 test-session-watch: $(BIN_DIR)/pair-session-watch
 	bash tests/pair-session-watch-test.sh
 
+runtimebundle-generate: $(addprefix $(BIN_DIR)/,$(RUNTIMEBUNDLE_HELPERS))
+	go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime
+
+$(RUNTIMEBUNDLE_ASSETS): runtimebundle-generate
+
+runtimebundle-drift-check: $(addprefix $(BIN_DIR)/,$(RUNTIMEBUNDLE_HELPERS))
+	@d1="$$(mktemp -d)"; d2="$$(mktemp -d)"; \
+	trap 'rm -rf "$$d1" "$$d2"' EXIT; \
+	go run ./cmd/internal/runtimebundle/generatecmd --repo . --out "$$d1/runtime" && \
+	go run ./cmd/internal/runtimebundle/generatecmd --repo . --out "$$d2/runtime" && \
+	diff -qr "$$d1/runtime" "$$d2/runtime" >/dev/null && \
+	find "$$d1/runtime/files" -type f -exec stat -f '%N %OLp' {} + | sed "s#$$d1/runtime/files/##" | sort > "$$d1/modes" && \
+	find "$$d2/runtime/files" -type f -exec stat -f '%N %OLp' {} + | sed "s#$$d2/runtime/files/##" | sort > "$$d2/modes" && \
+	diff -u "$$d1/modes" "$$d2/modes" >/dev/null
+
+test-runtimebundle: runtimebundle-generate
+	go test ./cmd/internal/runtimebundle -count=1
+
 # Review workbench (#66): the integration shell tests. The pure modules
 # (record, reconstruct) run under `make test-lua`.
 test-review:
@@ -195,6 +215,9 @@ test-dev-rebuild:
 test-pair-go-install-layout:
 	bash tests/pair-go-install-layout-test.sh
 
+test-pair-embedded-runtime:
+	bash tests/pair-embedded-runtime-test.sh
+
 # pair-doctor emitter-health probe (#000047): marker check + binary selection
 # (running-via-pidfile vs PATH). See tests/emitter-health-test.sh.
 test-emitter-health:
@@ -258,11 +281,12 @@ $(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill
 $(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-continuation/git.go cmd/pair-continuation/main.go go.mod
 	go build -o $@ ./cmd/pair-continuation
 
-PAIR_GO_SRCS := cmd/pair-go/main.go cmd/internal/entrypoint/asset_root.go cmd/internal/entrypoint/launch.go cmd/internal/entrypoint/mode.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
+PAIR_GO_SRCS := cmd/pair-go/main.go cmd/internal/entrypoint/asset_root.go cmd/internal/entrypoint/launch.go cmd/internal/entrypoint/mode.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go cmd/internal/runtimebundle/manifest.go cmd/internal/runtimebundle/plan.go cmd/internal/runtimebundle/cleanup.go cmd/internal/runtimebundle/store.go cmd/internal/runtimebundle/embed.go $(RUNTIMEBUNDLE_ASSETS) go.mod
 
 # pair — public Go-owned entrypoint. It execs bin/pair-shell for the current
-# shell-owned zellij lifecycle and uses the build-time defaultPairHome when a
-# copied local install has no sibling assets.
+# shell-owned zellij lifecycle; source/Homebrew builds can inject
+# defaultPairHome, while copied binaries without adjacent assets use the
+# embedded runtime fallback.
 $(BIN_DIR)/pair: $(PAIR_GO_SRCS)
 	go build -ldflags "$(PAIR_HOME_LDFLAGS)" -o $@ ./cmd/pair-go
 
diff --git a/README.md b/README.md
index eb3af77..1081b19 100644
--- a/README.md
+++ b/README.md
@@ -201,12 +201,19 @@ pair -h, --help                  # show full help
 ```
 
 The installed `pair` command is Go-owned. For this migration window it resolves
-the adjacent Pair asset root and hands off to `bin/pair-shell`, the retained
-shell launcher that still owns the zellij lifecycle. `pair-go launch ...` remains
-the explicit development dispatcher path and accepts the same arguments after
-`launch` that `pair` accepts directly. In a dev shell sourced from
-`../ariadne/construct/dev-aliases.sh`, `pair` and `pair-go` rebuild from
-`cmd/pair-go` automatically before running; no `pair-go-dev` command is needed.
+the Pair asset root and hands off to `bin/pair-shell`, the retained shell
+launcher that still owns the zellij lifecycle. Source and Homebrew installs use
+their adjacent asset roots. A copied standalone `pair` binary with no adjacent
+or build-time source root extracts its embedded Pair-owned runtime assets to
+`${XDG_DATA_HOME:-$HOME/.local/share}/pair/runtime/<digest>/pair-home` and runs
+with `PAIR_HOME` pointed there. External tools such as `zellij`, `nvim`, `fzf`,
+`jq`, clipboard tools, and agent CLIs are still installed separately.
+
+`pair-go launch ...` remains the explicit development dispatcher path and
+accepts the same arguments after `launch` that `pair` accepts directly. In a dev
+shell sourced from `../ariadne/construct/dev-aliases.sh`, `pair` and `pair-go`
+rebuild from `cmd/pair-go` automatically before running; no `pair-go-dev`
+command is needed.
 
 Use `--` to separate pair's positional from agent flags. Without it, pair only takes `<agent>` as a positional and everything else is rejected.
 
diff --git a/atlas/architecture.md b/atlas/architecture.md
index ca01dee..6718202 100644
--- a/atlas/architecture.md
+++ b/atlas/architecture.md
@@ -46,13 +46,21 @@ internal subcommands or dispatch modes behind that primary binary (`pair wrap`,
 `pair continuation`, `pair scribe`) instead of staying as independently managed
 installed commands forever.
 
-As of #79, the public `bin/pair` command is a Go-built entrypoint from
-`cmd/pair-go`. Direct `pair ...` and explicit `pair-go launch ...` share one
-compatibility handoff: resolve the Pair asset root, then exec
-`<asset-root>/bin/pair-shell` with `pair`-compatible argv/env. Asset root
-resolution is ordered: explicit `PAIR_HOME`, executable sibling root, then the
-build-time `defaultPairHome` injected by Make/Homebrew for copied installs.
-Native `nvim/` and `zellij/` assets remain adjacent to that root.
+As of #90, the public `bin/pair` command is a Go-built entrypoint from
+`cmd/pair-go` with an embedded Pair-owned runtime bundle. Direct `pair ...` and
+explicit `pair-go launch ...` share one compatibility handoff: resolve the Pair
+asset root, then exec `<asset-root>/bin/pair-shell` with `pair`-compatible
+argv/env. Asset root resolution is ordered: explicit `PAIR_HOME`, executable
+sibling root, the build-time `defaultPairHome` injected by Make/Homebrew, then
+an extracted embedded runtime under `$PAIR_DATA_DIR/runtime/<digest>/pair-home`
+when no adjacent/source asset root exists. Native `nvim/` and `zellij/` assets
+remain native files inside whichever asset root was selected.
+
+The embedded runtime is generated from a deterministic manifest before builds
+and tests. That manifest is the packaging source of truth for bundled Pair-owned
+shell helpers, helper binaries, `bin/lib/`, `nvim/`, `zellij/`, and doctor
+assets; external programs such as `zellij`, `nvim`, `fzf`, `jq`, clipboard
+tools, and agent CLIs remain system dependencies.
 
 `pair-go` remains the development dispatcher for helper routes and explicit
 launch testing: `pair-go launch claude`, `pair-go launch resume <tag>`,
diff --git a/atlas/go-migration-inventory.md b/atlas/go-migration-inventory.md
index ac87eee..37c5f91 100644
--- a/atlas/go-migration-inventory.md
+++ b/atlas/go-migration-inventory.md
@@ -21,14 +21,15 @@ Out of scope:
 
 - porting behavior;
 - changing public command behavior;
-- deciding final embedded-vs-adjacent asset packaging. #79 owns that decision.
+- removing adjacent asset packaging while Homebrew/source layouts still rely on
+  it.
 
 ## Single-Binary Deployment Path
 
-#79 made the installed public `pair` command Go-owned, but the supported runtime
-layout is still an adjacent asset tree. The next deployment goal is #90:
-produce a self-contained `pair` binary that embeds the Pair-owned runtime assets
-and extracts them to a versioned runtime root on demand.
+#79 made the installed public `pair` command Go-owned. #90 added a
+self-contained deployment mode: the Go binary embeds the Pair-owned runtime
+assets and extracts them to `$PAIR_DATA_DIR/runtime/<digest>/pair-home` on
+demand when no adjacent/source/Homebrew asset root is available.
 
 This is not the same as "no external dependencies." The single-binary target is
 one Pair artifact. System programs such as `zellij`, `nvim`, clipboard tools,
@@ -37,11 +38,13 @@ a later issue explicitly replaces them.
 
 Execution path:
 
-1. **Embedded runtime bundle (#90):** embed the current Pair-owned runtime tree
+1. **Embedded runtime bundle (#90):** the current Pair-owned runtime tree
    (`bin/pair-shell`, shell helpers, helper binaries or dispatcher shims,
-   `nvim/`, `zellij/`, docs/help needed at runtime) into the Go binary. On run,
-   extract to a versioned cache/data root and set `PAIR_HOME` there before the
-   existing launch handoff.
+   `bin/lib/`, `nvim/`, `zellij/`, and doctor/help assets needed at runtime) is
+   generated into a manifest-backed embedded bundle. On run, copied binaries
+   extract to a digest-named Pair data root, write a runtime marker, prune stale
+   older extracted runtimes without deleting the selected digest, and set
+   `PAIR_HOME` there before the existing launch handoff.
 2. **Dispatcher consolidation:** move helper binaries behind `pair <subcommand>`
    routes and leave old command names as generated compatibility shims only
    where native callers still need them.
@@ -84,7 +87,7 @@ Priority is packaging impact first, then reliability/testability:
 
 | Artifact | Type | Callers | Runtime contract | Files/env | Disposition | Priority |
 |---|---|---|---|---|---|---|
-| `bin/pair` / `bin/pair-shell` / `cmd/internal/launcher` / `cmd/internal/entrypoint` | Go public entrypoint plus retained shell launcher | user shell, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` is generated from `cmd/pair-go` and resolves `PAIR_HOME` / sibling root / build-time `defaultPairHome`, then execs `<asset-root>/bin/pair-shell` with `pair`-compatible argv/env. `bin/pair-shell` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` shares the same compatibility handoff. | `bin/pair-shell` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves invocation mode, asset root, and compatibility request; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Public entrypoint is Go-owned as of #79; `bin/pair-shell` is retained because real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
+| `bin/pair` / `bin/pair-shell` / `cmd/internal/launcher` / `cmd/internal/entrypoint` / `cmd/internal/runtimebundle` | Go public entrypoint plus retained shell launcher and embedded runtime fallback | user shell, copied-binary installs, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` is generated from `cmd/pair-go` and resolves `PAIR_HOME` / sibling root / build-time `defaultPairHome`; if none exists, it extracts the embedded runtime to `$PAIR_DATA_DIR/runtime/<digest>/pair-home`; then it execs `<asset-root>/bin/pair-shell` with `pair`-compatible argv/env and `PAIR_HOME` pointed at the selected root. `bin/pair-shell` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` shares the same compatibility handoff. | `bin/pair-shell` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves invocation mode, asset root, and compatibility request; `cmd/internal/runtimebundle` owns manifest hashing, extraction planning, runtime markers, and stale-runtime cleanup; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Public entrypoint is Go-owned as of #79; copied-binary embedded fallback exists as of #90; `bin/pair-shell` is retained because real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
 | `bin/pair-dev` | Bash launcher shim | developer shell | Same argv as `pair`; exports `PAIR_DEV=1` then execs sibling Go-built `pair`. | Resolves symlinks; depends on generated `bin/pair`, retained `bin/pair-shell`, and `bin/lib/dev-rebuild.sh`. | retained dev wrapper so developer launches exercise the public Go entrypoint | P1 |
 | `bin/lib/dev-rebuild.sh` | sourced shell helper | `bin/pair-shell` | Function `dev_rebuild`; no-op unless `PAIR_DEV`; always returns 0. | Reads `PAIR_HOME`; runs `make -C "$PAIR_HOME" build`; stderr warnings. | shell-glue or Go launcher dev-mode helper | P1 |
 | `zellij/layouts/main.kdl` | zellij native asset | `bin/pair-shell` via `zellij --new-session-with-layout` | Defines agent and draft panes; shell expands Pair env at pane start. | Calls `pair-wrap`; calls `nvim -u "$PAIR_HOME/nvim/init.lua"`; writes `pane-<tag>-<agent>.json`; writes draft nvim pid file. | native-asset, packaged adjacent/embedded | P0 |
@@ -151,14 +154,22 @@ Nvim shell-outs and binary dependencies:
 
 Build/install callers:
 
+- `make runtimebundle-generate` refreshes the gitignored embedded runtime asset
+  tree and manifest; `make runtimebundle-drift-check` verifies the generated
+  bundle is reproducible.
 - `make build` builds `GO_BINS` into `bin/`; `pair` and `pair-go` are both built
-  from `cmd/pair-go` with `defaultPairHome=$(CURDIR)`.
+  from `cmd/pair-go` with `defaultPairHome=$(CURDIR)`, while copied builds with
+  no adjacent/default root use the embedded fallback.
 - `make install` copies `GO_BINS` to `~/.local/bin` and symlinks only retained
   shell wrappers such as `pair-dev`. Installed `pair` is a regular Go binary;
-  if it has no sibling assets, it falls back to the build-time source root.
+  if it has no sibling assets, it falls back to the build-time source root when
+  that exists and otherwise extracts the embedded runtime.
 - Homebrew installs `bin/`, `nvim/`, and `zellij/` under `libexec`, then builds
   Go `pair`, `pair-go`, and required runtime helpers into `libexec/bin` with
   `defaultPairHome=#{libexec}`.
+- `make test-runtimebundle` runs bundle-generation-aware Go tests, and
+  `make test-pair-embedded-runtime` exercises copied-binary launch plus stale
+  runtime cleanup with fake external dependencies.
 - `pair-dev` relies on `make build`, then zellij's PATH lookup resolves fresh
   repo `bin/` binaries.
 
diff --git a/cmd/internal/entrypoint/asset_root.go b/cmd/internal/entrypoint/asset_root.go
index ebdf6df..aa023ff 100644
--- a/cmd/internal/entrypoint/asset_root.go
+++ b/cmd/internal/entrypoint/asset_root.go
@@ -10,6 +10,7 @@ type AssetRootInput struct {
 	PairHome        string
 	Executable      string
 	DefaultPairHome string
+	EmbeddedRoot    string
 	PairShellExists func(root string) bool
 }
 
@@ -38,6 +39,9 @@ func ResolveAssetRoot(input AssetRootInput) (AssetRoot, error) {
 	if input.DefaultPairHome != "" {
 		candidates = append(candidates, assetRootCandidate{root: input.DefaultPairHome, source: "defaultPairHome"})
 	}
+	if input.EmbeddedRoot != "" {
+		candidates = append(candidates, assetRootCandidate{root: input.EmbeddedRoot, source: "embedded runtime"})
+	}
 
 	seen := map[string]bool{}
 	checked := make([]string, 0, len(candidates))
diff --git a/cmd/internal/entrypoint/asset_root_test.go b/cmd/internal/entrypoint/asset_root_test.go
index 6c6843d..5124ffc 100644
--- a/cmd/internal/entrypoint/asset_root_test.go
+++ b/cmd/internal/entrypoint/asset_root_test.go
@@ -57,6 +57,24 @@ func TestResolveAssetRootFallsBackToDefaultPairHome(t *testing.T) {
 	}
 }
 
+func TestResolveAssetRootFallsBackToEmbeddedRootAfterAdjacentRoots(t *testing.T) {
+	root, err := ResolveAssetRoot(AssetRootInput{
+		Executable:      "/home/me/.local/bin/pair",
+		DefaultPairHome: "/default/root",
+		EmbeddedRoot:    "/data/pair/runtime/abc/pair-home",
+		PairShellExists: existingRoots("/data/pair/runtime/abc/pair-home"),
+	})
+	if err != nil {
+		t.Fatalf("ResolveAssetRoot error = %v", err)
+	}
+	if root.Root != "/data/pair/runtime/abc/pair-home" {
+		t.Fatalf("Root = %q, want embedded root", root.Root)
+	}
+	if root.Source != "embedded runtime" {
+		t.Fatalf("Source = %q, want embedded runtime", root.Source)
+	}
+}
+
 func TestResolveAssetRootReportsMissingPairShellAndPairHome(t *testing.T) {
 	_, err := ResolveAssetRoot(AssetRootInput{
 		Executable:      "/home/me/.local/bin/pair",
diff --git a/cmd/internal/runtimebundle/cleanup.go b/cmd/internal/runtimebundle/cleanup.go
new file mode 100644
index 0000000..9afe253
--- /dev/null
+++ b/cmd/internal/runtimebundle/cleanup.go
@@ -0,0 +1,68 @@
+package runtimebundle
+
+import (
+	"fmt"
+	"sort"
+)
+
+type RuntimeGeneration struct {
+	Digest    string
+	HasMarker bool
+	ModUnix   int64
+}
+
+type CleanupInput struct {
+	SelectedDigest string
+	Keep           int
+	Generations    []RuntimeGeneration
+}
+
+type CleanupPlan struct {
+	DeleteDigests []string
+}
+
+func PlanCleanup(input CleanupInput) (CleanupPlan, error) {
+	if input.SelectedDigest == "" {
+		return CleanupPlan{}, fmt.Errorf("selected digest is required")
+	}
+	if input.Keep < 0 {
+		return CleanupPlan{}, fmt.Errorf("keep must be non-negative")
+	}
+	candidates := make([]RuntimeGeneration, 0, len(input.Generations))
+	for _, gen := range input.Generations {
+		if gen.Digest == input.SelectedDigest || !gen.HasMarker || !isDigestName(gen.Digest) {
+			continue
+		}
+		candidates = append(candidates, gen)
+	}
+	sort.Slice(candidates, func(i, j int) bool {
+		if candidates[i].ModUnix == candidates[j].ModUnix {
+			return candidates[i].Digest > candidates[j].Digest
+		}
+		return candidates[i].ModUnix > candidates[j].ModUnix
+	})
+	if input.Keep >= len(candidates) {
+		return CleanupPlan{}, nil
+	}
+	deleteCandidates := candidates[input.Keep:]
+	sort.Slice(deleteCandidates, func(i, j int) bool {
+		return deleteCandidates[i].Digest < deleteCandidates[j].Digest
+	})
+	plan := CleanupPlan{DeleteDigests: make([]string, 0, len(deleteCandidates))}
+	for _, gen := range deleteCandidates {
+		plan.DeleteDigests = append(plan.DeleteDigests, gen.Digest)
+	}
+	return plan, nil
+}
+
+func isDigestName(s string) bool {
+	if len(s) != 64 {
+		return false
+	}
+	for _, r := range s {
+		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
+			return false
+		}
+	}
+	return true
+}
diff --git a/cmd/internal/runtimebundle/cleanup_test.go b/cmd/internal/runtimebundle/cleanup_test.go
new file mode 100644
index 0000000..2aea4bc
--- /dev/null
+++ b/cmd/internal/runtimebundle/cleanup_test.go
@@ -0,0 +1,49 @@
+package runtimebundle
+
+import (
+	"strings"
+	"testing"
+)
+
+func TestPlanCleanupKeepsSelectedRuntime(t *testing.T) {
+	a := strings.Repeat("a", 64)
+	b := strings.Repeat("b", 64)
+	c := strings.Repeat("c", 64)
+	plan, err := PlanCleanup(CleanupInput{
+		SelectedDigest: b,
+		Keep:           1,
+		Generations: []RuntimeGeneration{
+			{Digest: a, HasMarker: true, ModUnix: 10},
+			{Digest: b, HasMarker: true, ModUnix: 1},
+			{Digest: c, HasMarker: true, ModUnix: 20},
+		},
+	})
+	if err != nil {
+		t.Fatalf("PlanCleanup error = %v", err)
+	}
+	if len(plan.DeleteDigests) != 1 || plan.DeleteDigests[0] != a {
+		t.Fatalf("DeleteDigests = %#v, want only %s", plan.DeleteDigests, a)
+	}
+}
+
+func TestPlanCleanupIgnoresNonRuntimeDirectories(t *testing.T) {
+	b := strings.Repeat("b", 64)
+	c := strings.Repeat("c", 64)
+	plan, err := PlanCleanup(CleanupInput{
+		SelectedDigest: b,
+		Keep:           0,
+		Generations: []RuntimeGeneration{
+			{Digest: "not-a-digest", HasMarker: true, ModUnix: 10},
+			{Digest: strings.Repeat("a", 64), HasMarker: false, ModUnix: 20},
+			{Digest: "abcd", HasMarker: true, ModUnix: 25},
+			{Digest: b, HasMarker: true, ModUnix: 1},
+			{Digest: c, HasMarker: true, ModUnix: 30},
+		},
+	})
+	if err != nil {
+		t.Fatalf("PlanCleanup error = %v", err)
+	}
+	if len(plan.DeleteDigests) != 1 || plan.DeleteDigests[0] != c {
+		t.Fatalf("DeleteDigests = %#v, want only %s", plan.DeleteDigests, c)
+	}
+}
diff --git a/cmd/internal/runtimebundle/embed.go b/cmd/internal/runtimebundle/embed.go
new file mode 100644
index 0000000..c08cf3c
--- /dev/null
+++ b/cmd/internal/runtimebundle/embed.go
@@ -0,0 +1,25 @@
+package runtimebundle
+
+import (
+	"embed"
+	"encoding/json"
+)
+
+//go:embed assets/runtime/manifest.json assets/runtime/files
+var embedded embed.FS
+
+func EmbeddedManifest() RuntimeManifest {
+	data, err := embedded.ReadFile("assets/runtime/manifest.json")
+	if err != nil {
+		panic(err)
+	}
+	var manifest RuntimeManifest
+	if err := json.Unmarshal(data, &manifest); err != nil {
+		panic(err)
+	}
+	return manifest
+}
+
+func EmbeddedAsset(path string) ([]byte, error) {
+	return embedded.ReadFile("assets/runtime/files/" + path)
+}
diff --git a/cmd/internal/runtimebundle/embed_test.go b/cmd/internal/runtimebundle/embed_test.go
new file mode 100644
index 0000000..c0494ed
--- /dev/null
+++ b/cmd/internal/runtimebundle/embed_test.go
@@ -0,0 +1,45 @@
+package runtimebundle
+
+import "testing"
+
+func TestEmbeddedManifestContainsLaunchAssets(t *testing.T) {
+	manifest := EmbeddedManifest()
+	paths := map[string]bool{}
+	for _, asset := range manifest.Assets {
+		paths[asset.Path] = true
+	}
+	for _, want := range []string{
+		"bin/pair-shell",
+		"bin/pair-help",
+		"bin/pair-title.sh",
+		"bin/pair-session-watch.sh",
+		"bin/lib/dev-rebuild.sh",
+		"bin/pair-wrap",
+		"bin/pair-slug",
+		"bin/pair-context",
+		"bin/pair-scrollback-render",
+		"bin/pair-changelog",
+		"bin/pair-continuation",
+		"bin/pair-session-watch",
+		"nvim/init.lua",
+		"nvim/review/init.lua",
+		"zellij/config.kdl",
+		"zellij/layouts/main.kdl",
+		"doctor/SKILL.md",
+		"doctor/doctor.sh",
+	} {
+		if !paths[want] {
+			t.Fatalf("EmbeddedManifest missing %q", want)
+		}
+	}
+	for _, excluded := range []string{
+		"bin/pair",
+		"bin/pair-go",
+		"bin/pair-dev",
+		"nvim/init_test.lua",
+	} {
+		if paths[excluded] {
+			t.Fatalf("EmbeddedManifest includes excluded path %q", excluded)
+		}
+	}
+}
diff --git a/cmd/internal/runtimebundle/generatecmd/main.go b/cmd/internal/runtimebundle/generatecmd/main.go
new file mode 100644
index 0000000..ceddad8
--- /dev/null
+++ b/cmd/internal/runtimebundle/generatecmd/main.go
@@ -0,0 +1,19 @@
+package main
+
+import (
+	"flag"
+	"fmt"
+	"os"
+
+	"github.com/xianxu/pair/cmd/internal/runtimebundlegen"
+)
+
+func main() {
+	repo := flag.String("repo", ".", "repository root")
+	out := flag.String("out", "", "output root")
+	flag.Parse()
+	if _, err := runtimebundlegen.Generate(runtimebundlegen.GenerateOptions{RepoRoot: *repo, OutRoot: *out}); err != nil {
+		fmt.Fprintf(os.Stderr, "runtimebundle-generate: %v\n", err)
+		os.Exit(1)
+	}
+}
diff --git a/cmd/internal/runtimebundle/manifest.go b/cmd/internal/runtimebundle/manifest.go
new file mode 100644
index 0000000..434b61d
--- /dev/null
+++ b/cmd/internal/runtimebundle/manifest.go
@@ -0,0 +1,73 @@
+package runtimebundle
+
+import (
+	"crypto/sha256"
+	"encoding/hex"
+	"errors"
+	"fmt"
+	"path"
+	"sort"
+	"strings"
+)
+
+type RuntimeAsset struct {
+	Path   string
+	Mode   uint32
+	Size   int64
+	Digest string
+}
+
+type RuntimeManifest struct {
+	Assets []RuntimeAsset
+}
+
+func (m RuntimeManifest) Validate() error {
+	seen := map[string]bool{}
+	for _, asset := range m.Assets {
+		if err := validateAsset(asset); err != nil {
+			return err
+		}
+		if seen[asset.Path] {
+			return fmt.Errorf("duplicate asset path %q", asset.Path)
+		}
+		seen[asset.Path] = true
+	}
+	return nil
+}
+
+func (m RuntimeManifest) ManifestDigest() (string, error) {
+	if err := m.Validate(); err != nil {
+		return "", err
+	}
+	assets := append([]RuntimeAsset(nil), m.Assets...)
+	sort.Slice(assets, func(i, j int) bool {
+		return assets[i].Path < assets[j].Path
+	})
+	h := sha256.New()
+	for _, asset := range assets {
+		_, _ = fmt.Fprintf(h, "%s\x00%o\x00%d\x00%s\x00", asset.Path, asset.Mode, asset.Size, asset.Digest)
+	}
+	return hex.EncodeToString(h.Sum(nil)), nil
+}
+
+func validateAsset(asset RuntimeAsset) error {
+	if asset.Path == "" {
+		return errors.New("asset path is empty")
+	}
+	if strings.HasPrefix(asset.Path, "/") {
+		return fmt.Errorf("asset path %q is absolute", asset.Path)
+	}
+	clean := path.Clean(asset.Path)
+	if clean != asset.Path || clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
+		return fmt.Errorf("asset path %q is not clean relative path", asset.Path)
+	}
+	if asset.Digest == "" {
+		return fmt.Errorf("asset %q digest is empty", asset.Path)
+	}
+	return nil
+}
+
+func digestFor(s string) string {
+	h := sha256.Sum256([]byte(s))
+	return "sha256:" + hex.EncodeToString(h[:])
+}
diff --git a/cmd/internal/runtimebundle/manifest_test.go b/cmd/internal/runtimebundle/manifest_test.go
new file mode 100644
index 0000000..2aa69ce
--- /dev/null
+++ b/cmd/internal/runtimebundle/manifest_test.go
@@ -0,0 +1,58 @@
+package runtimebundle
+
+import "testing"
+
+func TestManifestDigestIsOrderIndependent(t *testing.T) {
+	a := RuntimeManifest{
+		Assets: []RuntimeAsset{
+			{Path: "bin/pair-shell", Mode: 0o755, Size: 3, Digest: "aaa"},
+			{Path: "nvim/init.lua", Mode: 0o644, Size: 4, Digest: "bbb"},
+		},
+	}
+	b := RuntimeManifest{
+		Assets: []RuntimeAsset{
+			{Path: "nvim/init.lua", Mode: 0o644, Size: 4, Digest: "bbb"},
+			{Path: "bin/pair-shell", Mode: 0o755, Size: 3, Digest: "aaa"},
+		},
+	}
+
+	gotA, err := a.ManifestDigest()
+	if err != nil {
+		t.Fatalf("ManifestDigest(a) error = %v", err)
+	}
+	gotB, err := b.ManifestDigest()
+	if err != nil {
+		t.Fatalf("ManifestDigest(b) error = %v", err)
+	}
+	if gotA == "" {
+		t.Fatal("ManifestDigest() = empty")
+	}
+	if gotA != gotB {
+		t.Fatalf("digest differs by order: %q != %q", gotA, gotB)
+	}
+}
+
+func TestManifestRejectsUnsafePaths(t *testing.T) {
+	tests := []struct {
+		name   string
+		assets []RuntimeAsset
+	}{
+		{name: "empty", assets: []RuntimeAsset{{Path: "", Mode: 0o644, Digest: "a"}}},
+		{name: "absolute", assets: []RuntimeAsset{{Path: "/bin/pair-shell", Mode: 0o755, Digest: "a"}}},
+		{name: "dotdot", assets: []RuntimeAsset{{Path: "bin/../pair-shell", Mode: 0o755, Digest: "a"}}},
+		{name: "duplicate", assets: []RuntimeAsset{
+			{Path: "bin/pair-shell", Mode: 0o755, Digest: "a"},
+			{Path: "bin/pair-shell", Mode: 0o755, Digest: "a"},
+		}},
+		{name: "empty digest", assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755}}},
+	}
+
+	for _, tt := range tests {
+		t.Run(tt.name, func(t *testing.T) {
+			err := RuntimeManifest{Assets: tt.assets}.Validate()
+			if err == nil {
+				t.Fatal("Validate() error = nil, want error")
+			}
+		})
+	}
+}
diff --git a/cmd/internal/runtimebundle/plan.go b/cmd/internal/runtimebundle/plan.go
new file mode 100644
index 0000000..adcb33d
--- /dev/null
+++ b/cmd/internal/runtimebundle/plan.go
@@ -0,0 +1,63 @@
+package runtimebundle
+
+import (
+	"fmt"
+	"path/filepath"
+)
+
+type ExistingAsset struct {
+	Mode   uint32
+	Size   int64
+	Digest string
+}
+
+type ExtractionInput struct {
+	StoreRoot   string
+	RuntimeRoot string
+	Manifest    RuntimeManifest
+	Existing    map[string]ExistingAsset
+}
+
+type ExtractionPlan struct {
+	Writes []RuntimeAsset
+	Skips  []string
+}
+
+func PlanExtraction(input ExtractionInput) (ExtractionPlan, error) {
+	if err := input.Manifest.Validate(); err != nil {
+		return ExtractionPlan{}, err
+	}
+	if err := validateRuntimeRoot(input.StoreRoot, input.RuntimeRoot); err != nil {
+		return ExtractionPlan{}, err
+	}
+	existing := input.Existing
+	if existing == nil {
+		existing = map[string]ExistingAsset{}
+	}
+	plan := ExtractionPlan{}
+	for _, asset := range input.Manifest.Assets {
+		got, ok := existing[asset.Path]
+		if ok && got.Mode == asset.Mode && got.Size == asset.Size && got.Digest == asset.Digest {
+			plan.Skips = append(plan.Skips, asset.Path)
+			continue
+		}
+		plan.Writes = append(plan.Writes, asset)
+	}
+	return plan, nil
+}
+
+func validateRuntimeRoot(storeRoot, runtimeRoot string) error {
+	if storeRoot == "" || runtimeRoot == "" {
+		return fmt.Errorf("store root and runtime root are required")
+	}
+	store := filepath.Clean(storeRoot)
+	root := filepath.Clean(runtimeRoot)
+	rel, err := filepath.Rel(store, root)
+	if err != nil {
+		return err
+	}
+	if rel == "." || rel == ".." || rel == "" || (len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator)) {
+		return fmt.Errorf("runtime root %q is outside store root %q", runtimeRoot, storeRoot)
+	}
+	return nil
+}
diff --git a/cmd/internal/runtimebundle/plan_test.go b/cmd/internal/runtimebundle/plan_test.go
new file mode 100644
index 0000000..ff2a09a
--- /dev/null
+++ b/cmd/internal/runtimebundle/plan_test.go
@@ -0,0 +1,78 @@
+package runtimebundle
+
+import (
+	"path/filepath"
+	"testing"
+)
+
+func TestPlanExtractionWritesMissingAssets(t *testing.T) {
+	manifest := RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755, Size: 10, Digest: "sha256:a"}}}
+
+	plan, err := PlanExtraction(ExtractionInput{
+		StoreRoot:   "/data/pair/runtime",
+		RuntimeRoot: "/data/pair/runtime/abc/pair-home",
+		Manifest:    manifest,
+		Existing:    map[string]ExistingAsset{},
+	})
+	if err != nil {
+		t.Fatalf("PlanExtraction error = %v", err)
+	}
+	if len(plan.Writes) != 1 || plan.Writes[0].Path != "bin/pair-shell" {
+		t.Fatalf("Writes = %#v, want bin/pair-shell", plan.Writes)
+	}
+	if len(plan.Skips) != 0 {
+		t.Fatalf("Skips = %#v, want empty", plan.Skips)
+	}
+}
+
+func TestPlanExtractionSkipsMatchingAssets(t *testing.T) {
+	manifest := RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755, Size: 10, Digest: "sha256:a"}}}
+
+	plan, err := PlanExtraction(ExtractionInput{
+		StoreRoot:   "/data/pair/runtime",
+		RuntimeRoot: "/data/pair/runtime/abc/pair-home",
+		Manifest:    manifest,
+		Existing: map[string]ExistingAsset{
+			"bin/pair-shell": {Mode: 0o755, Size: 10, Digest: "sha256:a"},
+		},
+	})
+	if err != nil {
+		t.Fatalf("PlanExtraction error = %v", err)
+	}
+	if len(plan.Writes) != 0 {
+		t.Fatalf("Writes = %#v, want empty", plan.Writes)
+	}
+	if len(plan.Skips) != 1 || plan.Skips[0] != "bin/pair-shell" {
+		t.Fatalf("Skips = %#v, want bin/pair-shell", plan.Skips)
+	}
+}
+
+func TestPlanExtractionRefreshesMismatchedDigest(t *testing.T) {
+	manifest := RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755, Size: 10, Digest: "sha256:a"}}}
+
+	plan, err := PlanExtraction(ExtractionInput{
+		StoreRoot:   "/data/pair/runtime",
+		RuntimeRoot: "/data/pair/runtime/abc/pair-home",
+		Manifest:    manifest,
+		Existing: map[string]ExistingAsset{
+			"bin/pair-shell": {Mode: 0o755, Size: 10, Digest: "sha256:old"},
+		},
+	})
+	if err != nil {
+		t.Fatalf("PlanExtraction error = %v", err)
+	}
+	if len(plan.Writes) != 1 || plan.Writes[0].Path != "bin/pair-shell" {
+		t.Fatalf("Writes = %#v, want refresh", plan.Writes)
+	}
+}
+
+func TestPlanExtractionRejectsRuntimeRootOutsideStore(t *testing.T) {
+	_, err := PlanExtraction(ExtractionInput{
+		StoreRoot:   "/data/pair/runtime",
+		RuntimeRoot: filepath.Clean("/data/pair/not-runtime/abc/pair-home"),
+		Manifest:    RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755, Digest: "sha256:a"}}},
+	})
+	if err == nil {
+		t.Fatal("PlanExtraction error = nil, want root containment error")
+	}
+}
diff --git a/cmd/internal/runtimebundle/store.go b/cmd/internal/runtimebundle/store.go
new file mode 100644
index 0000000..8ddece7
--- /dev/null
+++ b/cmd/internal/runtimebundle/store.go
@@ -0,0 +1,187 @@
+package runtimebundle
+
+import (
+	"crypto/sha256"
+	"encoding/hex"
+	"encoding/json"
+	"fmt"
+	"os"
+	"path/filepath"
+	"sort"
+	"time"
+)
+
+type StoreInput struct {
+	StoreRoot string
+	Manifest  RuntimeManifest
+	ReadAsset func(string) ([]byte, error)
+	Keep      int
+}
+
+type StoreResult struct {
+	Digest   string
+	PairHome string
+}
+
+type runtimeMarker struct {
+	Digest     string `json:"digest"`
+	AssetCount int    `json:"asset_count"`
+	Generated  string `json:"generated,omitempty"`
+}
+
+func Extract(input StoreInput) (StoreResult, error) {
+	if input.ReadAsset == nil {
+		return StoreResult{}, fmt.Errorf("asset reader is required")
+	}
+	digest, err := input.Manifest.ManifestDigest()
+	if err != nil {
+		return StoreResult{}, err
+	}
+	pairHome := filepath.Join(input.StoreRoot, digest, "pair-home")
+	existing, err := scanExisting(pairHome, input.Manifest)
+	if err != nil {
+		return StoreResult{}, err
+	}
+	plan, err := PlanExtraction(ExtractionInput{
+		StoreRoot:   input.StoreRoot,
+		RuntimeRoot: pairHome,
+		Manifest:    input.Manifest,
+		Existing:    existing,
+	})
+	if err != nil {
+		return StoreResult{}, err
+	}
+	for _, asset := range plan.Writes {
+		data, err := input.ReadAsset(asset.Path)
+		if err != nil {
+			return StoreResult{}, fmt.Errorf("read embedded asset %s: %w", asset.Path, err)
+		}
+		if digestFor(string(data)) != asset.Digest {
+			// Binary assets may not be valid UTF-8 but string preserves bytes.
+			return StoreResult{}, fmt.Errorf("embedded asset %s digest mismatch", asset.Path)
+		}
+		if int64(len(data)) != asset.Size {
+			return StoreResult{}, fmt.Errorf("embedded asset %s size mismatch", asset.Path)
+		}
+		if err := writeFileAtomic(filepath.Join(pairHome, filepath.FromSlash(asset.Path)), data, os.FileMode(asset.Mode)); err != nil {
+			return StoreResult{}, err
+		}
+	}
+	if err := writeMarker(filepath.Join(input.StoreRoot, digest, "manifest.json"), digest, len(input.Manifest.Assets)); err != nil {
+		return StoreResult{}, err
+	}
+	if err := applyCleanup(input.StoreRoot, digest, input.Keep); err != nil {
+		return StoreResult{}, err
+	}
+	return StoreResult{Digest: digest, PairHome: pairHome}, nil
+}
+
+func scanExisting(root string, manifest RuntimeManifest) (map[string]ExistingAsset, error) {
+	existing := map[string]ExistingAsset{}
+	for _, asset := range manifest.Assets {
+		p := filepath.Join(root, filepath.FromSlash(asset.Path))
+		info, err := os.Stat(p)
+		if err != nil {
+			if os.IsNotExist(err) {
+				continue
+			}
+			return nil, err
+		}
+		if info.IsDir() {
+			continue
+		}
+		data, err := os.ReadFile(p)
+		if err != nil {
+			return nil, err
+		}
+		h := sha256.Sum256(data)
+		existing[asset.Path] = ExistingAsset{
+			Mode:   uint32(info.Mode().Perm()),
+			Size:   info.Size(),
+			Digest: "sha256:" + hex.EncodeToString(h[:]),
+		}
+	}
+	return existing, nil
+}
+
+func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
+	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
+		return err
+	}
+	tmp := path + ".tmp"
+	if err := os.WriteFile(tmp, data, mode); err != nil {
+		_ = os.Remove(tmp)
+		return err
+	}
+	if err := os.Chmod(tmp, mode); err != nil {
+		_ = os.Remove(tmp)
+		return err
+	}
+	if err := os.Rename(tmp, path); err != nil {
+		_ = os.Remove(tmp)
+		return err
+	}
+	return nil
+}
+
+func writeMarker(path, digest string, assetCount int) error {
+	data, err := json.MarshalIndent(runtimeMarker{
+		Digest:     digest,
+		AssetCount: assetCount,
+		Generated:  time.Now().UTC().Format(time.RFC3339),
+	}, "", "  ")
+	if err != nil {
+		return err
+	}
+	data = append(data, '\n')
+	return writeFileAtomic(path, data, 0o644)
+}
+
+func applyCleanup(storeRoot, selectedDigest string, keep int) error {
+	entries, err := os.ReadDir(storeRoot)
+	if err != nil {
+		if os.IsNotExist(err) {
+			return nil
+		}
+		return err
+	}
+	gens := make([]RuntimeGeneration, 0, len(entries))
+	for _, entry := range entries {
+		if !entry.IsDir() {
+			continue
+		}
+		name := entry.Name()
+		info, err := entry.Info()
+		if err != nil {
+			return err
+		}
+		gens = append(gens, RuntimeGeneration{
+			Digest:    name,
+			HasMarker: markerValid(filepath.Join(storeRoot, name, "manifest.json"), name),
+			ModUnix:   info.ModTime().Unix(),
+		})
+	}
+	plan, err := PlanCleanup(CleanupInput{SelectedDigest: selectedDigest, Keep: keep, Generations: gens})
+	if err != nil {
+		return err
+	}
+	sort.Strings(plan.DeleteDigests)
+	for _, digest := range plan.DeleteDigests {
+		if err := os.RemoveAll(filepath.Join(storeRoot, digest)); err != nil {
+			return err
+		}
+	}
+	return nil
+}
+
+func markerValid(path, digest string) bool {
+	data, err := os.ReadFile(path)
+	if err != nil {
+		return false
+	}
+	var marker runtimeMarker
+	if err := json.Unmarshal(data, &marker); err != nil {
+		return false
+	}
+	return marker.Digest == digest
+}
diff --git a/cmd/internal/runtimebundle/store_test.go b/cmd/internal/runtimebundle/store_test.go
new file mode 100644
index 0000000..21f729f
--- /dev/null
+++ b/cmd/internal/runtimebundle/store_test.go
@@ -0,0 +1,137 @@
+package runtimebundle
+
+import (
+	"os"
+	"path/filepath"
+	"strings"
+	"testing"
+)
+
+func TestStoreExtractsAssetsAndWritesMarker(t *testing.T) {
+	dir := t.TempDir()
+	shellContent := "pair shell\n"
+	initContent := "init\n"
+	manifest := RuntimeManifest{Assets: []RuntimeAsset{
+		{Path: "bin/pair-shell", Mode: 0o755, Size: int64(len(shellContent)), Digest: digestFor(shellContent)},
+		{Path: "nvim/init.lua", Mode: 0o644, Size: int64(len(initContent)), Digest: digestFor(initContent)},
+	}}
+
+	res, err := Extract(StoreInput{
+		StoreRoot: dir,
+		Manifest:  manifest,
+		ReadAsset: fakeAssetReader(map[string]string{
+			"bin/pair-shell": shellContent,
+			"nvim/init.lua":  initContent,
+		}),
+		Keep: 1,
+	})
+	if err != nil {
+		t.Fatalf("Extract error = %v", err)
+	}
+	shell := filepath.Join(res.PairHome, "bin", "pair-shell")
+	got, err := os.ReadFile(shell)
+	if err != nil {
+		t.Fatalf("ReadFile(pair-shell) error = %v", err)
+	}
+	if string(got) != "pair shell\n" {
+		t.Fatalf("pair-shell content = %q", got)
+	}
+	info, err := os.Stat(shell)
+	if err != nil {
+		t.Fatalf("Stat(pair-shell) error = %v", err)
+	}
+	if info.Mode().Perm() != 0o755 {
+		t.Fatalf("pair-shell mode = %o, want 755", info.Mode().Perm())
+	}
+	marker, err := os.ReadFile(filepath.Join(filepath.Dir(res.PairHome), "manifest.json"))
+	if err != nil {
+		t.Fatalf("ReadFile(marker) error = %v", err)
+	}
+	if len(marker) == 0 {
+		t.Fatal("marker is empty")
+	}
+}
+
+func TestStoreSecondExtractIsIdempotent(t *testing.T) {
+	dir := t.TempDir()
+	content := "pair shell\n"
+	manifest := RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755, Size: int64(len(content)), Digest: digestFor(content)}}}
+	input := StoreInput{
+		StoreRoot: dir,
+		Manifest:  manifest,
+		ReadAsset: fakeAssetReader(map[string]string{"bin/pair-shell": content}),
+		Keep:      1,
+	}
+	first, err := Extract(input)
+	if err != nil {
+		t.Fatalf("first Extract error = %v", err)
+	}
+	second, err := Extract(input)
+	if err != nil {
+		t.Fatalf("second Extract error = %v", err)
+	}
+	if first.PairHome != second.PairHome {
+		t.Fatalf("PairHome changed: %q != %q", first.PairHome, second.PairHome)
+	}
+}
+
+func TestStoreCleanupPreservesSelectedRuntime(t *testing.T) {
+	dir := t.TempDir()
+	oldDigest := strings.Repeat("a", 64)
+	old := filepath.Join(dir, oldDigest, "pair-home")
+	if err := os.MkdirAll(old, 0o755); err != nil {
+		t.Fatalf("MkdirAll(old) error = %v", err)
+	}
+	if err := os.WriteFile(filepath.Join(dir, oldDigest, "manifest.json"), []byte(`{"digest":"`+oldDigest+`","asset_count":0}`), 0o644); err != nil {
+		t.Fatalf("WriteFile(old marker) error = %v", err)
+	}
+	content := "pair shell\n"
+	manifest := RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755, Size: int64(len(content)), Digest: digestFor(content)}}}
+
+	res, err := Extract(StoreInput{
+		StoreRoot: dir,
+		Manifest:  manifest,
+		ReadAsset: fakeAssetReader(map[string]string{"bin/pair-shell": content}),
+		Keep:      0,
+	})
+	if err != nil {
+		t.Fatalf("Extract error = %v", err)
+	}
+	if _, err := os.Stat(res.PairHome); err != nil {
+		t.Fatalf("selected runtime was removed: %v", err)
+	}
+	if _, err := os.Stat(filepath.Join(dir, oldDigest)); !os.IsNotExist(err) {
+		t.Fatalf("old runtime still exists or stat failed unexpectedly: %v", err)
+	}
+}
+
+func TestStoreCleanupIgnoresMarkerDigestMismatch(t *testing.T) {
+	dir := t.TempDir()
+	oldDigest := strings.Repeat("b", 64)
+	if err := os.MkdirAll(filepath.Join(dir, oldDigest, "pair-home"), 0o755); err != nil {
+		t.Fatalf("MkdirAll(old) error = %v", err)
+	}
+	if err := os.WriteFile(filepath.Join(dir, oldDigest, "manifest.json"), []byte(`{"digest":"`+strings.Repeat("c", 64)+`","asset_count":0}`), 0o644); err != nil {
+		t.Fatalf("WriteFile(old marker) error = %v", err)
+	}
+	content := "pair shell\n"
+	manifest := RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755, Size: int64(len(content)), Digest: digestFor(content)}}}
+
+	if _, err := Extract(StoreInput{
+		StoreRoot: dir,
+		Manifest:  manifest,
+		ReadAsset: fakeAssetReader(map[string]string{"bin/pair-shell": content}),
+		Keep:      0,
+	}); err != nil {
+		t.Fatalf("Extract error = %v", err)
+	}
+	if _, err := os.Stat(filepath.Join(dir, oldDigest)); err != nil {
+		t.Fatalf("mismatched-marker runtime should be ignored, not deleted: %v", err)
+	}
+}
+
+func fakeAssetReader(files map[string]string) func(string) ([]byte, error) {
+	return func(path string) ([]byte, error) {
+		return []byte(files[path]), nil
+	}
+}
diff --git a/cmd/internal/runtimebundlegen/generate.go b/cmd/internal/runtimebundlegen/generate.go
new file mode 100644
index 0000000..0daeb3b
--- /dev/null
+++ b/cmd/internal/runtimebundlegen/generate.go
@@ -0,0 +1,236 @@
+package runtimebundlegen
+
+import (
+	"crypto/sha256"
+	"encoding/hex"
+	"encoding/json"
+	"fmt"
+	"io"
+	"io/fs"
+	"os"
+	"path/filepath"
+	"sort"
+	"strings"
+	"time"
+
+	"github.com/xianxu/pair/cmd/internal/runtimebundle"
+)
+
+var explicitAssetPaths = []string{
+	"bin/pair-shell",
+	"bin/pair-help",
+	"bin/pair-notify",
+	"bin/pair-quit.sh",
+	"bin/pair-restart.sh",
+	"bin/pair-scrollback-open",
+	"bin/pair-changelog-open",
+	"bin/pair-review-open",
+	"bin/pair-review-readiness",
+	"bin/pair-review-target",
+	"bin/copy-on-select.sh",
+	"bin/clipboard-to-pane.sh",
+	"bin/flash-pane.sh",
+	"bin/pair-title.sh",
+	"bin/pair-session-watch.sh",
+	"bin/pair-wrap",
+	"bin/pair-slug",
+	"bin/pair-context",
+	"bin/pair-scrollback-render",
+	"bin/pair-changelog",
+	"bin/pair-continuation",
+	"bin/pair-session-watch",
+	"doctor/README.md",
+	"doctor/SKILL.md",
+	"doctor/doctor.sh",
+	"doctor/emitter-health.sh",
+}
+
+var assetDirs = []string{
+	"bin/lib",
+	"nvim",
+	"zellij",
+}
+
+type GenerateOptions struct {
+	RepoRoot string
+	OutRoot  string
+}
+
+func Generate(opts GenerateOptions) (runtimebundle.RuntimeManifest, error) {
+	if opts.RepoRoot == "" {
+		opts.RepoRoot = "."
+	}
+	if opts.OutRoot == "" {
+		return runtimebundle.RuntimeManifest{}, fmt.Errorf("output root is required")
+	}
+	repoRoot, err := filepath.Abs(opts.RepoRoot)
+	if err != nil {
+		return runtimebundle.RuntimeManifest{}, err
+	}
+	outRoot, err := filepath.Abs(opts.OutRoot)
+	if err != nil {
+		return runtimebundle.RuntimeManifest{}, err
+	}
+	outParent := filepath.Dir(outRoot)
+	outBase := filepath.Base(outRoot)
+	if err := os.MkdirAll(outParent, 0o755); err != nil {
+		return runtimebundle.RuntimeManifest{}, err
+	}
+	stageRoot, err := os.MkdirTemp(outParent, "."+outBase+"-tmp-")
+	if err != nil {
+		return runtimebundle.RuntimeManifest{}, err
+	}
+	committed := false
+	defer func() {
+		if !committed {
+			_ = os.RemoveAll(stageRoot)
+		}
+	}()
+	filesRoot := filepath.Join(stageRoot, "files")
+
+	paths := map[string]bool{}
+	for _, p := range explicitAssetPaths {
+		paths[p] = true
+	}
+	for _, dir := range assetDirs {
+		root := filepath.Join(repoRoot, filepath.FromSlash(dir))
+		if err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
+			if err != nil {
+				return err
+			}
+			if d.IsDir() {
+				return nil
+			}
+			rel, err := filepath.Rel(repoRoot, p)
+			if err != nil {
+				return err
+			}
+			logical := filepath.ToSlash(rel)
+			if shouldExclude(logical) {
+				return nil
+			}
+			paths[logical] = true
+			return nil
+		}); err != nil {
+			return runtimebundle.RuntimeManifest{}, err
+		}
+	}
+
+	ordered := make([]string, 0, len(paths))
+	for p := range paths {
+		if shouldExclude(p) {
+			continue
+		}
+		ordered = append(ordered, p)
+	}
+	sort.Strings(ordered)
+
+	manifest := runtimebundle.RuntimeManifest{Assets: make([]runtimebundle.RuntimeAsset, 0, len(ordered))}
+	for _, logical := range ordered {
+		src := filepath.Join(repoRoot, filepath.FromSlash(logical))
+		info, err := os.Stat(src)
+		if err != nil {
+			return runtimebundle.RuntimeManifest{}, fmt.Errorf("asset %s: %w", logical, err)
+		}
+		if info.IsDir() {
+			continue
+		}
+		digest, err := copyAsset(src, filepath.Join(filesRoot, filepath.FromSlash(logical)), info.Mode().Perm())
+		if err != nil {
+			return runtimebundle.RuntimeManifest{}, err
+		}
+		manifest.Assets = append(manifest.Assets, runtimebundle.RuntimeAsset{
+			Path:   logical,
+			Mode:   uint32(info.Mode().Perm()),
+			Size:   info.Size(),
+			Digest: "sha256:" + digest,
+		})
+	}
+	encoded, err := json.MarshalIndent(manifest, "", "  ")
+	if err != nil {
+		return runtimebundle.RuntimeManifest{}, err
+	}
+	encoded = append(encoded, '\n')
+	if err := os.WriteFile(filepath.Join(stageRoot, "manifest.json"), encoded, 0o644); err != nil {
+		return runtimebundle.RuntimeManifest{}, err
+	}
+	unlock, err := acquirePublishLock(outRoot + ".lock")
+	if err != nil {
+		return runtimebundle.RuntimeManifest{}, err
+	}
+	defer unlock()
+	if err := os.RemoveAll(outRoot); err != nil {
+		return runtimebundle.RuntimeManifest{}, err
+	}
+	if err := os.Rename(stageRoot, outRoot); err != nil {
+		return runtimebundle.RuntimeManifest{}, err
+	}
+	committed = true
+	return manifest, nil
+}
+
+func acquirePublishLock(path string) (func(), error) {
+	const attempts = 1000
+	for i := 0; i < attempts; i++ {
+		err := os.Mkdir(path, 0o755)
+		if err == nil {
+			return func() { _ = os.Remove(path) }, nil
+		}
+		if !os.IsExist(err) {
+			return nil, err
+		}
+		time.Sleep(10 * time.Millisecond)
+	}
+	return nil, fmt.Errorf("timed out waiting for runtime bundle publish lock %s", path)
+}
+
+func shouldExclude(logical string) bool {
+	base := filepath.Base(logical)
+	if base == ".DS_Store" || strings.Contains(logical, "__pycache__/") {
+		return true
+	}
+	if strings.HasSuffix(logical, "_test.lua") {
+		return true
+	}
+	switch logical {
+	case "bin/pair", "bin/pair-go", "bin/pair-dev":
+		return true
+	}
+	return false
+}
+
+func copyAsset(src, dst string, mode os.FileMode) (string, error) {
+	in, err := os.Open(src)
+	if err != nil {
+		return "", err
+	}
+	defer in.Close()
+	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
+		return "", err
+	}
+	tmp := dst + ".tmp"
+	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
+	if err != nil {
+		return "", err
+	}
+	h := sha256.New()
+	_, copyErr := io.Copy(io.MultiWriter(out, h), in)
+	closeErr := out.Close()
+	if copyErr != nil {
+		_ = os.Remove(tmp)
+		return "", copyErr
+	}
+	if closeErr != nil {
+		_ = os.Remove(tmp)
+		return "", closeErr
+	}
+	if err := os.Chmod(tmp, mode); err != nil {
+		_ = os.Remove(tmp)
+		return "", err
+	}
+	if err := os.Rename(tmp, dst); err != nil {
+		_ = os.Remove(tmp)
+		return "", err
+	}
+	return hex.EncodeToString(h.Sum(nil)), nil
+}
diff --git a/cmd/internal/runtimebundlegen/generate_test.go b/cmd/internal/runtimebundlegen/generate_test.go
new file mode 100644
index 0000000..3b10e18
--- /dev/null
+++ b/cmd/internal/runtimebundlegen/generate_test.go
@@ -0,0 +1,84 @@
+package runtimebundlegen
+
+import (
+	"os"
+	"path/filepath"
+	"sync"
+	"testing"
+)
+
+func TestGeneratePreservesExistingOutputOnFailure(t *testing.T) {
+	repo := t.TempDir()
+	out := filepath.Join(t.TempDir(), "runtime")
+	writeMinimalRuntimeRepo(t, repo)
+
+	if _, err := Generate(GenerateOptions{RepoRoot: repo, OutRoot: out}); err != nil {
+		t.Fatalf("initial Generate error = %v", err)
+	}
+	before, err := os.ReadFile(filepath.Join(out, "manifest.json"))
+	if err != nil {
+		t.Fatalf("ReadFile(initial manifest) error = %v", err)
+	}
+
+	if err := os.Remove(filepath.Join(repo, "bin", "pair-shell")); err != nil {
+		t.Fatalf("Remove(pair-shell) error = %v", err)
+	}
+	if _, err := Generate(GenerateOptions{RepoRoot: repo, OutRoot: out}); err == nil {
+		t.Fatal("Generate error = nil, want missing asset error")
+	}
+
+	after, err := os.ReadFile(filepath.Join(out, "manifest.json"))
+	if err != nil {
+		t.Fatalf("ReadFile(preserved manifest) error = %v", err)
+	}
+	if string(after) != string(before) {
+		t.Fatal("failed generation changed existing output manifest")
+	}
+}
+
+func TestGenerateConcurrentSameOutputSucceeds(t *testing.T) {
+	repo := t.TempDir()
+	out := filepath.Join(t.TempDir(), "runtime")
+	writeMinimalRuntimeRepo(t, repo)
+
+	var wg sync.WaitGroup
+	errs := make(chan error, 8)
+	for i := 0; i < 8; i++ {
+		wg.Add(1)
+		go func() {
+			defer wg.Done()
+			_, err := Generate(GenerateOptions{RepoRoot: repo, OutRoot: out})
+			errs <- err
+		}()
+	}
+	wg.Wait()
+	close(errs)
+	for err := range errs {
+		if err != nil {
+			t.Fatalf("Generate error = %v", err)
+		}
+	}
+	if _, err := os.Stat(filepath.Join(out, "manifest.json")); err != nil {
+		t.Fatalf("Stat(manifest) error = %v", err)
+	}
+}
+
+func writeMinimalRuntimeRepo(t *testing.T, repo string) {
+	t.Helper()
+	for _, logical := range explicitAssetPaths {
+		writeFile(t, filepath.Join(repo, filepath.FromSlash(logical)), "#!/bin/sh\n", 0o755)
+	}
+	writeFile(t, filepath.Join(repo, "bin", "lib", "shared.sh"), "shared\n", 0o644)
+	writeFile(t, filepath.Join(repo, "nvim", "init.lua"), "-- init\n", 0o644)
+	writeFile(t, filepath.Join(repo, "zellij", "config.kdl"), "keybinds {}\n", 0o644)
+}
+
+func writeFile(t *testing.T, path, content string, mode os.FileMode) {
+	t.Helper()
+	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
+		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
+	}
+	if err := os.WriteFile(path, []byte(content), mode); err != nil {
+		t.Fatalf("WriteFile(%s) error = %v", path, err)
+	}
+}
diff --git a/cmd/pair-go/main.go b/cmd/pair-go/main.go
index 6834be7..08c80c1 100644
--- a/cmd/pair-go/main.go
+++ b/cmd/pair-go/main.go
@@ -7,10 +7,13 @@ import (
 	"io"
 	"os"
 	"path/filepath"
+	"strings"
 	"syscall"
 
 	"github.com/xianxu/pair/cmd/internal/dispatcher"
 	"github.com/xianxu/pair/cmd/internal/entrypoint"
+	"github.com/xianxu/pair/cmd/internal/launcher"
+	"github.com/xianxu/pair/cmd/internal/runtimebundle"
 )
 
 var defaultPairHome string
@@ -29,6 +32,7 @@ type legacyRuntime interface {
 	DefaultPairHome() string
 	Stat(path string) error
 	Environ() []string
+	EmbeddedAssetRoot() (string, error)
 	Exec(label string, path string, argv []string, env []string) int
 }
 
@@ -64,11 +68,28 @@ func runLegacyLaunch(label string, executable string, args []string, stderr io.W
 		},
 	})
 	if err != nil {
-		_, _ = fmt.Fprintf(stderr, "%s: %v; run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", label, err)
-		return 1
+		embeddedRoot, embeddedErr := rt.EmbeddedAssetRoot()
+		if embeddedErr == nil && embeddedRoot != "" {
+			root, err = entrypoint.ResolveAssetRoot(entrypoint.AssetRootInput{
+				PairHome:        rt.PairHome(),
+				Executable:      executable,
+				DefaultPairHome: rt.DefaultPairHome(),
+				EmbeddedRoot:    embeddedRoot,
+				PairShellExists: func(root string) bool {
+					return rt.Stat(entrypoint.PairShellPath(root)) == nil
+				},
+			})
+		}
+		if err != nil {
+			if embeddedErr != nil {
+				_, _ = fmt.Fprintf(stderr, "%s: embedded runtime extraction failed: %v\n", label, embeddedErr)
+			}
+			_, _ = fmt.Fprintf(stderr, "%s: %v; run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", label, err)
+			return 1
+		}
 	}
 	req := entrypoint.ResolveLegacyLaunch(root, args)
-	return rt.Exec(label, req.Path, req.Argv, rt.Environ())
+	return rt.Exec(label, req.Path, req.Argv, withEnv(rt.Environ(), "PAIR_HOME", root.Root))
 }
 
 type osLegacyRuntime struct{}
@@ -101,6 +122,27 @@ func (osLegacyRuntime) Environ() []string {
 	return os.Environ()
 }
 
+func (osLegacyRuntime) EmbeddedAssetRoot() (string, error) {
+	dataDir := runtimeDataDir(os.Getenv("PAIR_DATA_DIR"), os.Getenv("HOME"), os.Getenv("XDG_DATA_HOME"))
+	res, err := runtimebundle.Extract(runtimebundle.StoreInput{
+		StoreRoot: filepath.Join(dataDir, "runtime"),
+		Manifest:  runtimebundle.EmbeddedManifest(),
+		ReadAsset: runtimebundle.EmbeddedAsset,
+		Keep:      2,
+	})
+	if err != nil {
+		return "", err
+	}
+	return res.PairHome, nil
+}
+
+func runtimeDataDir(pairDataDir, home, xdgDataHome string) string {
+	if pairDataDir != "" {
+		return pairDataDir
+	}
+	return launcher.ResolveDataDir(home, xdgDataHome)
+}
+
 func (osLegacyRuntime) Exec(label string, path string, argv []string, env []string) int {
 	if err := syscall.Exec(path, argv, env); err != nil {
 		_, _ = fmt.Fprintf(os.Stderr, "%s: exec %s failed: %v\n", label, path, err)
@@ -118,3 +160,23 @@ func writeResult(res dispatcher.Result, stdout, stderr io.Writer) int {
 	}
 	return res.ExitCode
 }
+
+func withEnv(env []string, key, value string) []string {
+	prefix := key + "="
+	out := make([]string, 0, len(env)+1)
+	replaced := false
+	for _, entry := range env {
+		if strings.HasPrefix(entry, prefix) {
+			if !replaced {
+				out = append(out, prefix+value)
+				replaced = true
+			}
+			continue
+		}
+		out = append(out, entry)
+	}
+	if !replaced {
+		out = append(out, prefix+value)
+	}
+	return out
+}
diff --git a/cmd/pair-go/main_test.go b/cmd/pair-go/main_test.go
index 75dd2a7..f87426b 100644
--- a/cmd/pair-go/main_test.go
+++ b/cmd/pair-go/main_test.go
@@ -176,6 +176,46 @@ func TestRunDirectPairFallsBackToDefaultPairHome(t *testing.T) {
 	}
 }
 
+func TestRunDirectPairFallsBackToEmbeddedRuntimeAndSetsPairHome(t *testing.T) {
+	rt := &fakeLegacyRuntime{
+		executable:   "/home/me/.local/bin/pair",
+		embeddedRoot: "/data/pair/runtime/abc/pair-home",
+		roots:        map[string]bool{"/data/pair/runtime/abc/pair-home": true},
+		execCode:     9,
+		environ:      []string{"PATH=/bin", "PAIR_HOME=/old"},
+	}
+
+	var stdout, stderr bytes.Buffer
+	code := runWithLegacyRuntime([]string{"--help"}, &stdout, &stderr, rt)
+
+	if code != 9 {
+		t.Fatalf("code = %d, want 9", code)
+	}
+	if rt.execPath != "/data/pair/runtime/abc/pair-home/bin/pair-shell" {
+		t.Fatalf("execPath = %q, want embedded pair-shell", rt.execPath)
+	}
+	if !containsEnv(rt.execEnv, "PAIR_HOME=/data/pair/runtime/abc/pair-home") {
+		t.Fatalf("execEnv missing embedded PAIR_HOME: %#v", rt.execEnv)
+	}
+	if containsEnv(rt.execEnv, "PAIR_HOME=/old") {
+		t.Fatalf("execEnv kept old PAIR_HOME: %#v", rt.execEnv)
+	}
+}
+
+func TestRuntimeDataDirPrefersPairDataDir(t *testing.T) {
+	got := runtimeDataDir("/pair-data", "/home/me", "/xdg")
+	if got != "/pair-data" {
+		t.Fatalf("runtimeDataDir = %q, want PAIR_DATA_DIR", got)
+	}
+}
+
+func TestRuntimeDataDirFallsBackToXDGPairDir(t *testing.T) {
+	got := runtimeDataDir("", "/home/me", "/xdg")
+	if got != "/xdg/pair" {
+		t.Fatalf("runtimeDataDir = %q, want XDG pair dir", got)
+	}
+}
+
 func TestRunPairGoHelperDoesNotProbeOrExecShellLauncher(t *testing.T) {
 	rt := &fakeLegacyRuntime{
 		executable: "/repo/bin/pair-go",
@@ -209,6 +249,9 @@ type fakeLegacyRuntime struct {
 	statErr         error
 	execCode        int
 	statCalls       int
+	embeddedRoot    string
+	embeddedErr     error
+	environ         []string
 
 	execPath  string
 	execLabel string
@@ -246,9 +289,16 @@ func (f *fakeLegacyRuntime) Stat(path string) error {
 }
 
 func (f *fakeLegacyRuntime) Environ() []string {
+	if f.environ != nil {
+		return f.environ
+	}
 	return os.Environ()
 }
 
+func (f *fakeLegacyRuntime) EmbeddedAssetRoot() (string, error) {
+	return f.embeddedRoot, f.embeddedErr
+}
+
 func (f *fakeLegacyRuntime) Exec(label string, path string, argv []string, env []string) int {
 	f.execLabel = label
 	f.execPath = path
diff --git a/tests/pair-embedded-runtime-test.sh b/tests/pair-embedded-runtime-test.sh
new file mode 100644
index 0000000..a229481
--- /dev/null
+++ b/tests/pair-embedded-runtime-test.sh
@@ -0,0 +1,118 @@
+#!/usr/bin/env bash
+set -euo pipefail
+
+repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
+tmp="$(mktemp -d "${TMPDIR:-/tmp}/pair-embedded-runtime.XXXXXX")"
+trap 'rm -rf "$tmp"' EXIT
+
+bin_dir="$tmp/bin"
+home="$tmp/home"
+xdg="$tmp/xdg"
+pair_data="$tmp/custom-data"
+mkdir -p "$bin_dir" "$home" "$xdg" "$pair_data"
+gomodcache="$(go env GOMODCACHE)"
+gocache="$(go env GOCACHE)"
+
+make -C "$repo_root" runtimebundle-generate >/dev/null
+go build -o "$bin_dir/pair" "$repo_root/cmd/pair-go"
+
+cat >"$bin_dir/zellij" <<'SH'
+#!/usr/bin/env bash
+set -eu
+printf '%s\n' "$*" >> "${ZELLIJ_LOG:?}"
+case "$*" in
+  "list-sessions --no-formatting"|"list-sessions --short")
+    exit 0
+    ;;
+  --session*" action list-clients")
+    exit 0
+    ;;
+  --config-dir*)
+    config=""
+    layout=""
+    prev=""
+    for arg in "$@"; do
+      if [ "$prev" = "--config-dir" ]; then config="$arg"; fi
+      if [ "$prev" = "--new-session-with-layout" ]; then layout="$arg"; fi
+      prev="$arg"
+    done
+    test -f "$config/config.kdl"
+    test -f "$layout"
+    case "$config" in */custom-data/runtime/*/pair-home/zellij) ;; *) printf 'bad config path: %s\n' "$config" >&2; exit 11 ;; esac
+    case "$layout" in */custom-data/runtime/*/pair-home/zellij/layouts/main.kdl) ;; *) printf 'bad layout path: %s\n' "$layout" >&2; exit 12 ;; esac
+    root="${config%/zellij}"
+    test -x "$root/bin/pair-shell"
+    test -x "$root/bin/pair-wrap"
+    test -x "$root/bin/pair-session-watch.sh"
+    test -x "$root/bin/pair-title.sh"
+    test -f "$root/nvim/init.lua"
+    printf '%s\n' "$root" > "${PAIR_SMOKE_ROOT:?}"
+    exit 0
+    ;;
+  *)
+    exit 0
+    ;;
+esac
+SH
+chmod +x "$bin_dir/zellij"
+
+cat >"$bin_dir/ps" <<'SH'
+#!/usr/bin/env bash
+case "$*" in
+  "-o comm= -p "*)
+    printf 'sh\n'
+    ;;
+  "-o ppid= -p "*)
+    printf '1\n'
+    ;;
+  *)
+    exec /bin/ps "$@"
+    ;;
+esac
+SH
+chmod +x "$bin_dir/ps"
+
+export PATH="$bin_dir:$PATH"
+export HOME="$home"
+export XDG_DATA_HOME="$xdg"
+export PAIR_DATA_DIR="$pair_data"
+export GOMODCACHE="$gomodcache"
+export GOCACHE="$gocache"
+export ZELLIJ_LOG="$tmp/zellij.log"
+export PAIR_SMOKE_ROOT="$tmp/root"
+unset PAIR_DEV PAIR_HOME PAIR_TAG PAIR_AGENT PAIR_AGENT_ARGS ZELLIJ_SESSION_NAME ZELLIJ ZELLIJ_PANE_ID
+
+help_out="$("$bin_dir/pair" --help)"
+case "$help_out" in
+  pair\ —*) ;;
+  *)
+    printf 'copied pair --help did not reach embedded pair-shell help; first bytes:\n%s\n' "$help_out" >&2
+    exit 1
+    ;;
+esac
+
+old_a="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
+old_b="bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
+old_c="cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
+mkdir -p "$pair_data/runtime/$old_a/pair-home" \
+         "$pair_data/runtime/$old_b/pair-home" \
+         "$pair_data/runtime/$old_c/pair-home"
+printf '{"digest":"%s","asset_count":0}\n' "$old_a" > "$pair_data/runtime/$old_a/manifest.json"
+printf '{"digest":"%s","asset_count":0}\n' "$old_b" > "$pair_data/runtime/$old_b/manifest.json"
+printf '{"digest":"%s","asset_count":0}\n' "$old_c" > "$pair_data/runtime/$old_c/manifest.json"
+touch -t 202001010000 "$pair_data/runtime/$old_a"
+touch -t 202001020000 "$pair_data/runtime/$old_b"
+touch -t 202001030000 "$pair_data/runtime/$old_c"
+
+"$bin_dir/pair" resume smoke >/dev/null
+
+test -s "$PAIR_SMOKE_ROOT"
+root="$(cat "$PAIR_SMOKE_ROOT")"
+case "$root" in */custom-data/runtime/*/pair-home) ;; *) printf 'bad extracted root: %s\n' "$root" >&2; exit 1 ;; esac
+test -d "$root"
+test ! -e "$pair_data/runtime/$old_a"
+test -d "$pair_data/runtime/$old_b"
+test -d "$pair_data/runtime/$old_c"
+test ! -e "$xdg/pair/runtime"
+
+printf 'pair embedded runtime smoke passed\n'
diff --git a/workshop/lessons.md b/workshop/lessons.md
index 5e78983..1757151 100644
--- a/workshop/lessons.md
+++ b/workshop/lessons.md
@@ -1,5 +1,20 @@
 # Lessons
 
+## Path precedence contracts need explicit divergent-env tests
+
+#90's embedded runtime implementation documented extraction under
+`$PAIR_DATA_DIR/runtime/<digest>/pair-home`, but the first OS-backed
+implementation only used the XDG/home resolver. The copied-binary smoke unset
+`PAIR_DATA_DIR`, so the bug survived until boundary review tried
+`PAIR_DATA_DIR` and `XDG_DATA_HOME` with different roots.
+
+**Rule.** When a feature promises environment-variable precedence, add a test
+where the higher-priority and fallback variables are both set to different
+directories, then assert the selected path. Also include every Go source file
+that can change build output in Make prerequisites; a generated or embedded
+artifact path should have a dependency test or an explicit review checklist
+entry. Caught in #000090 boundary review.
+
 ## Lua patterns: `\0` is empty-position match, not NUL byte
 
 The unescape function in `nvim/scrollback.lua` first attempt used a
diff --git a/workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md b/workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md
index 2c30dfc..45f8bc7 100644
--- a/workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md
+++ b/workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md
@@ -1,6 +1,6 @@
 # Self-Contained Pair Binary Implementation Plan
 
-> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.
+> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [x]`) syntax for tracking.
 
 **Goal:** Build a `pair` binary that can launch from a clean copied location by extracting embedded Pair-owned runtime assets into a versioned runtime root.
 
@@ -54,7 +54,7 @@
 | `RuntimeFS` | `cmd/internal/runtimebundle/embed.go` | new | `embed.FS` |
 | `RuntimeStore` | `cmd/internal/runtimebundle/store.go` | new | filesystem writes, chmod, rename |
 | `PairGoLegacyRuntime` | `cmd/pair-go/main.go` | modified | `os`, `syscall.Exec`, env |
-| `BuildManifest` | `cmd/internal/runtimebundle/generate.go`, `cmd/internal/runtimebundle/assets/**`, `Makefile.local` | new | repo file tree at build time |
+| `BuildManifest` | `cmd/internal/runtimebundlegen/generate.go`, `cmd/internal/runtimebundle/generatecmd/main.go`, `cmd/internal/runtimebundle/assets/**`, `Makefile.local` | new | repo file tree at build time |
 | `CopiedBinarySmoke` | `tests/pair-embedded-runtime-test.sh` | new | real built binary with fake external commands |
 
 - **RuntimeFS** — exposes embedded asset bytes and manifest data.
@@ -148,7 +148,7 @@ Runtime root contract:
 - Create: `cmd/internal/runtimebundle/manifest.go`
 - Create: `cmd/internal/runtimebundle/manifest_test.go`
 
-- [ ] **Step 1: Write failing tests for manifest validation and digest stability**
+- [x] **Step 1: Write failing tests for manifest validation and digest stability**
 
 Add tests that construct a small manifest in memory and assert:
 
@@ -162,17 +162,17 @@ func TestManifestRejectsUnsafePaths(t *testing.T) {
 }
 ```
 
-- [ ] **Step 2: Run the red tests**
+- [x] **Step 2: Run the red tests**
 
 Run: `go test ./cmd/internal/runtimebundle -run 'TestManifest' -count=1`
 
 Expected: package or symbols missing.
 
-- [ ] **Step 3: Implement `RuntimeAsset`, `RuntimeManifest`, validation, and digest**
+- [x] **Step 3: Implement `RuntimeAsset`, `RuntimeManifest`, validation, and digest**
 
 Keep this pure: no filesystem or embed access. Normalize paths with slash semantics, sort entries for hashing, and include path, mode, size, and content digest in the manifest digest.
 
-- [ ] **Step 4: Run green tests**
+- [x] **Step 4: Run green tests**
 
 Run: `go test ./cmd/internal/runtimebundle -run 'TestManifest' -count=1`
 
@@ -186,7 +186,7 @@ Expected: pass.
 - Create: `cmd/internal/runtimebundle/cleanup.go`
 - Create: `cmd/internal/runtimebundle/cleanup_test.go`
 
-- [ ] **Step 1: Write failing tests for extraction idempotency**
+- [x] **Step 1: Write failing tests for extraction idempotency**
 
 Cover these behaviors:
 
@@ -197,21 +197,21 @@ func TestPlanExtractionRefreshesMismatchedDigest(t *testing.T) {}
 func TestPlanExtractionRejectsRuntimeRootOutsideStore(t *testing.T) {}
 ```
 
-- [ ] **Step 2: Write failing tests for cleanup safety**
+- [x] **Step 2: Write failing tests for cleanup safety**
 
 Cover keeping the selected `$PAIR_DATA_DIR/runtime/<digest>/` generation, retaining the newest configured count, and deleting only directories that match the runtime root contract above.
 
-- [ ] **Step 3: Run the red tests**
+- [x] **Step 3: Run the red tests**
 
 Run: `go test ./cmd/internal/runtimebundle -run 'TestPlan|TestCleanup' -count=1`
 
 Expected: missing functions or failing behavior.
 
-- [ ] **Step 4: Implement pure planning**
+- [x] **Step 4: Implement pure planning**
 
 Represent write/skip/delete as data. Do not call `os.WriteFile`, `os.RemoveAll`, or `time.Now` from the pure functions; pass observed filesystem state and current runtime identity in.
 
-- [ ] **Step 5: Run green tests**
+- [x] **Step 5: Run green tests**
 
 Run: `go test ./cmd/internal/runtimebundle -count=1`
 
@@ -229,11 +229,11 @@ Expected: pass.
 - Modify: `Makefile.local`
 - Test: `cmd/internal/runtimebundle/embed_test.go`
 
-- [ ] **Step 1: Write failing tests that assert all required runtime assets exist in the embedded manifest**
+- [x] **Step 1: Write failing tests that assert all required runtime assets exist in the embedded manifest**
 
 Assert the exact asset boundary above: launcher shell, shell helper scripts, `bin/lib/**`, required Go helper binaries, `nvim/**`, `zellij/**`, and doctor runtime diagnostics are present; excluded entrypoints/caches/tests are absent. Use `atlas/go-migration-inventory.md` only as human review context; automated tests inspect the generated manifest, not atlas.
 
-- [ ] **Step 2: Run the red test before adding embed code**
+- [x] **Step 2: Run the red test before adding embed code**
 
 Run: `go test ./cmd/internal/runtimebundle -run TestEmbeddedManifestContainsLaunchAssets -count=1`
 
@@ -241,15 +241,15 @@ Expected: missing embedded manifest symbol. This red run happens before
 `embed.go` is added, so it does not trip Go's compile-time `//go:embed` missing
 file rule.
 
-- [ ] **Step 3: Add the deterministic generator and embedding path**
+- [x] **Step 3: Add the deterministic generator and embedding path**
 
 Implement `make runtimebundle-generate` to copy the required inputs into `cmd/internal/runtimebundle/assets/runtime/`, preserving executable bits and writing a manifest JSON with sorted paths. Add `//go:embed assets/runtime/**` and expose the embedded `RuntimeManifest` plus file reader through `RuntimeFS`.
 
-- [ ] **Step 4: Wire build, test, and staleness checks**
+- [x] **Step 4: Wire build, test, and staleness checks**
 
 Update `PAIR_GO_SRCS` so changes to runtimebundle sources rebuild `bin/pair` and `bin/pair-go`. Add `test-runtimebundle` and `runtimebundle-drift-check`; include both in the relevant test path. Ensure `make build` builds helper binaries first, regenerates the asset tree, then builds `pair`/`pair-go` so the single binary embeds current helpers without committing generated binaries.
 
-- [ ] **Step 5: Run green tests**
+- [x] **Step 5: Run green tests**
 
 Run: `make test-runtimebundle`
 
@@ -261,21 +261,21 @@ Expected: pass.
 - Create: `cmd/internal/runtimebundle/store.go`
 - Create: `cmd/internal/runtimebundle/store_test.go`
 
-- [ ] **Step 1: Write failing filesystem tests**
+- [x] **Step 1: Write failing filesystem tests**
 
 Use `t.TempDir()` and a fake embedded asset reader. Assert atomic first-run extraction, executable bits for `bin/*` scripts, idempotent second run, manifest marker write, and stale cleanup that preserves the selected runtime.
 
-- [ ] **Step 2: Run the red tests**
+- [x] **Step 2: Run the red tests**
 
 Run: `go test ./cmd/internal/runtimebundle -run 'TestStore|TestExtract' -count=1`
 
 Expected: missing store implementation.
 
-- [ ] **Step 3: Implement the thin IO shell**
+- [x] **Step 3: Implement the thin IO shell**
 
 Use temp files plus rename for file writes. Write a manifest marker after all assets land. Apply cleanup after successful extraction/selection, not before.
 
-- [ ] **Step 4: Run green tests**
+- [x] **Step 4: Run green tests**
 
 Run: `make test-runtimebundle`
 
@@ -291,7 +291,7 @@ Expected: pass.
 - Modify: `cmd/pair-go/main.go`
 - Modify: `cmd/pair-go/main_test.go`
 
-- [ ] **Step 1: Write failing tests for launch-root precedence**
+- [x] **Step 1: Write failing tests for launch-root precedence**
 
 Assert ordering:
 
@@ -300,21 +300,21 @@ Assert ordering:
 3. `defaultPairHome` wins.
 4. Embedded extracted runtime is used only when the adjacent/default candidates fail.
 
-- [ ] **Step 2: Write failing `cmd/pair-go` fake-runtime tests**
+- [x] **Step 2: Write failing `cmd/pair-go` fake-runtime tests**
 
 Extend the existing `legacyRuntime` fake so launch records `PAIR_HOME=<extracted-root>` in the exec env when embedded fallback is selected.
 
-- [ ] **Step 3: Run red tests**
+- [x] **Step 3: Run red tests**
 
 Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -run 'TestResolveAssetRoot|TestRun' -count=1`
 
 Expected: embedded fallback behavior missing.
 
-- [ ] **Step 4: Implement embedded fallback**
+- [x] **Step 4: Implement embedded fallback**
 
 Keep `ResolveAssetRoot` deterministic and side-effect-free by passing the already-prepared embedded root as an optional candidate. Let `cmd/pair-go` decide whether extraction is needed and then pass the selected root into the existing legacy launch handoff.
 
-- [ ] **Step 5: Run green tests**
+- [x] **Step 5: Run green tests**
 
 Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
 
@@ -326,11 +326,11 @@ Expected: pass.
 - Create: `tests/pair-embedded-runtime-test.sh`
 - Modify: `Makefile.local`
 
-- [ ] **Step 1: Write the failing clean-copy help smoke test**
+- [x] **Step 1: Write the failing clean-copy help smoke test**
 
 Build `bin/pair`, copy only that binary to a temp `PATH`, set `PAIR_DATA_DIR` to a temp dir, and run a harmless launch path such as `--help` through `pair-shell`. Use fake external commands only where needed so the test does not start a real zellij session.
 
-- [ ] **Step 2: Write the failing session-asset smoke test**
+- [x] **Step 2: Write the failing session-asset smoke test**
 
 Run the copied binary through a create-path with fake `zellij`, `nvim`, `fzf`,
 `jq`, `par`, and fake agent commands on `PATH`. The fake `zellij` must record the
@@ -341,21 +341,21 @@ must verify the embedded runtime contains and can resolve `pair-wrap`,
 the copied binary reaches the compatibility launch/session asset path, not only
 the help text.
 
-- [ ] **Step 3: Add upgrade/stale-runtime assertions**
+- [x] **Step 3: Add upgrade/stale-runtime assertions**
 
 Simulate an old extracted runtime under the temp data/cache area and assert it is pruned only after the new runtime is selected; assert the current runtime remains.
 
-- [ ] **Step 4: Run the red smoke**
+- [x] **Step 4: Run the red smoke**
 
 Run: `bash tests/pair-embedded-runtime-test.sh`
 
 Expected: copied binary cannot find `pair-shell` before implementation.
 
-- [ ] **Step 5: Add the `make` target**
+- [x] **Step 5: Add the `make` target**
 
 Add `test-pair-embedded-runtime` and `runtimebundle-drift-check` to `.PHONY`; add the smoke target to the main `test` dependency list after `test-pair-go-install-layout`.
 
-- [ ] **Step 6: Run green smoke**
+- [x] **Step 6: Run green smoke**
 
 Run: `make test-pair-embedded-runtime`
 
@@ -372,15 +372,15 @@ Expected: copied-binary clean and upgrade paths pass.
 - Modify: `README.md`
 - Optional modify: sibling `../homebrew-pair/Formula/pair.rb`
 
-- [ ] **Step 1: Update atlas execution path**
+- [x] **Step 1: Update atlas execution path**
 
 Record that #90 embeds/extracts Pair-owned runtime assets and that adjacent layout remains supported for source/Homebrew compatibility.
 
-- [ ] **Step 2: Update install docs**
+- [x] **Step 2: Update install docs**
 
 Document the copied-binary deployment mode and keep external dependencies explicit.
 
-- [ ] **Step 3: Audit Homebrew notes**
+- [x] **Step 3: Audit Homebrew notes**
 
 This issue does not require moving Homebrew to binary-only packaging. If the formula still packages adjacent assets, leave behavior unchanged and document it as compatibility mode in `README.md` / atlas. Touch `../homebrew-pair/Formula/pair.rb` only if it contains false adjacent-only claims or build rules that conflict with the generated bundle; if touched, verify with `ruby -c ../homebrew-pair/Formula/pair.rb` and record the sibling commit.
 
@@ -393,7 +393,7 @@ a separate sibling commit and record that commit hash in this issue's `## Log`.
 **Files:**
 - Modify: `workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md`
 
-- [ ] **Step 1: Run focused Go tests**
+- [x] **Step 1: Run focused Go tests**
 
 Run: `make test-runtimebundle`
 
@@ -403,7 +403,7 @@ Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
 
 Expected: pass.
 
-- [ ] **Step 2: Run smoke and build tests**
+- [x] **Step 2: Run smoke and build tests**
 
 Run: `make build`
 
@@ -413,13 +413,13 @@ Run: `make test-pair-go-install-layout test-pair-embedded-runtime`
 
 Expected: pass.
 
-- [ ] **Step 3: Run broader Go tests**
+- [x] **Step 3: Run broader Go tests**
 
 Run: `go test ./... -count=1`
 
 Expected: pass.
 
-- [ ] **Step 4: Validate issue and diff hygiene**
+- [x] **Step 4: Validate issue and diff hygiene**
 
 Run: `sdlc issue validate workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md`
 
@@ -429,6 +429,19 @@ Run: `git diff --check`
 
 Expected: no whitespace errors.
 
-- [ ] **Step 5: Update issue plan/log**
+- [x] **Step 5: Update issue plan/log**
 
 Tick completed issue plan items, add verification evidence to `## Log`, and prepare for `sdlc close --issue 90 --verified '<evidence>' --no-atlas` only if atlas truly had no new surface. This issue is expected to require atlas updates, so avoid `--no-atlas` unless the implementation scope changes.
+
+## Revisions
+
+### 2026-07-01 — Boundary review fixes
+
+Reason: the SDLC boundary review found that the Core Concepts table named the
+generator package path incorrectly after implementation split the generator from
+the embedded runtime package.
+
+Delta: updated `BuildManifest` to point at
+`cmd/internal/runtimebundlegen/generate.go` and
+`cmd/internal/runtimebundle/generatecmd/main.go`, leaving the generated
+gitignored assets and Makefile integration as originally planned.


codex
I’ll do this as a fresh boundary review: first I’m loading the required workflow skill, then I’ll inspect the issue, plan, and current tree against the supplied diff before returning only review findings.
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
Using `superpowers-using-superpowers` for the required session workflow check. I’ve also read the SDLC command contract; next I’m checking the issue/plan and the changed files at `HEAD` for line-precise findings.
exec
/bin/zsh -lc "sed -n '1,260p' workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "sed -n '1,460p' workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
---
id: 000090
status: working
deps: []
github_issue:
created: 2026-07-01
updated: 2026-07-01
estimate_hours: 5.44
started: 2026-07-01T00:18:42-07:00
---

# self-contained pair binary with embedded runtime assets

## Problem

After #79, the public `pair` command is Go-owned, but deployment is still an
installed tree: the Go entrypoint must find adjacent Pair-owned runtime assets
such as `bin/pair-shell`, shell helpers, `nvim/`, `zellij/`, and helper
binaries. That is simpler for Homebrew, but it is not the deployment shape we
eventually want: copying one Pair binary around and having it work.

The long-term direction is a true native single binary. Rewriting every
remaining shell and orchestration surface directly into Go is too much risk in
one jump, so the next step should make the current runtime tree derive from one
Go artifact without pretending the shell lifecycle is already gone.

## Spec

Add a self-contained deployment mode for the Go `pair` binary:

- Embed the Pair-owned runtime assets needed by launch/session flows into the Go
  binary.
- On first run, extract those assets to a versioned runtime directory under the
  user's Pair data/cache area, then run the existing launch flow with `PAIR_HOME`
  pointed at that extracted runtime root.
- Preserve the current adjacent-install behavior for source checkout and
  Homebrew layouts unless/until the self-contained mode proves it can replace
  them.
- Keep external programs external: `zellij`, `nvim`, `fzf`, `jq` while the shell
  runtime still needs it, clipboard tools, and agent CLIs are not bundled by this
  issue.
- Make runtime extraction deterministic, idempotent, and upgrade-safe: a new
  binary/runtime version should extract a new directory or refresh only when the
  embedded manifest changes.
- Add a cleanup policy for stale extracted runtimes that cannot delete the
  currently running runtime.

Execution path toward the native single binary:

1. Embed and extract the existing runtime tree. This delivers the "single Pair
   artifact" deployment option while retaining the tested shell/nvim/zellij
   contracts.
2. Route generated internal calls through the Go dispatcher where possible
   (`pair wrap`, `pair slug`, `pair changelog`, `pair continuation`, etc.) while
   keeping compatibility names only as shims.
3. Port stateful shell orchestrators into Go one at a time: launcher/session
   lifecycle, scrollback/changelog openers, title poller, review helpers,
   clipboard helpers.
4. Once shell ownership is gone, stop extracting shell scripts and use embedded
   or generated native assets only for `nvim/` and `zellij/`.
5. Revisit whether `nvim/` and `zellij/` remain extracted native assets or move
   to generated temp files/API-driven startup. The native single binary target
   is one Pair executable, with external platform tools still supplied by the
   system.

Architecture notes:

- `ARCH-PURPOSE`: the copied binary must be enough to provide Pair-owned
  runtime assets; falling back to a source checkout does not satisfy this issue.
- `ARCH-DRY`: the embedded runtime manifest must be the single source of what is
  packaged, installed, and tested. Do not maintain a separate hand-written asset
  list for Homebrew, tests, and extraction.
- `ARCH-PURE`: keep manifest planning, runtime selection, and extraction
  decisions as pure functions with unit tests; keep filesystem writes and
  process exec in thin seams.

## Done when

- [x] A release build can produce one `pair` binary that contains the Pair-owned
      runtime assets needed for launch/session flows.
- [x] Copying only that binary to a clean path works when external dependencies
      are installed.
- [x] First run extracts or refreshes a versioned runtime root and points
      `PAIR_HOME` at it for the compatibility launch handoff.
- [x] Adjacent source/Homebrew layouts still work.
- [x] Upgrade and stale-runtime cleanup behavior is tested.
- [x] The execution path toward the true native single binary is documented in
      atlas.

## Plan

- [x] Define the embedded runtime manifest and generated asset list.
- [x] Implement runtime extraction and version/manifest selection.
- [x] Wire `cmd/pair-go` to prefer extracted embedded runtime when no adjacent
      asset root exists.
- [x] Add install/copy smoke tests for clean and upgrade paths.
- [x] Update README, atlas, and Homebrew packaging notes.

Detailed implementation plan:
`workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md`.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale, so the number is provisional but uses the required method.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.20 impl=0.08
item: greenfield-go-module design=0.60 impl=0.56
item: smaller-go-module design=0.35 impl=0.48
item: cross-cutting-refactor design=0.80 impl=1.12
item: atlas-docs design=0.25 impl=0.20
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 5.44
```

## Log

### 2026-07-01

Created after #79 closed: #79 made `pair` Go-owned but intentionally retained
the adjacent runtime tree. The desired final direction is a true native single
binary; this issue captures the lower-risk next step of embedding/extracting
the current runtime tree first.

Claimed and entered planning. `sdlc start-plan --issue 90` delivered
`ARCH-DRY`, `ARCH-PURE`, and `ARCH-PURPOSE`; the durable plan keeps the runtime
manifest as the packaging source of truth, pure planning/selection functions in
Go, and copied-binary launch as the acceptance path rather than a follow-up.

First `sdlc change-code --issue 90` plan-quality gate returned FAILURE: asset
boundary, generator/staleness contract, and copied-binary smoke were too loose.
Refined the durable plan to name exact runtime asset roots/exclusions, require a
deterministic gitignored generator plus drift check, exercise a fake
launch/session path, and bound Homebrew formula edits to false/conflicting
packaging claims only.

Second `sdlc change-code --issue 90` plan-quality gate returned FAILURE on
remaining precision issues: `bin/pair-title.sh` was referenced by smoke coverage
but missing from the required asset list, extracted runtime naming/version rules
were implicit, and atlas wording could imply a second source. Updated the plan
to include `pair-title.sh`, define `$PAIR_DATA_DIR/runtime/<digest>/pair-home`
plus marker/cleanup rules, and state that automated behavior derives only from
the generated manifest and runtime marker.

Third `sdlc change-code --issue 90` plan-quality gate returned FAILURE because
raw `go test ./cmd/internal/runtimebundle` would fail from a clean checkout once
`//go:embed` references the gitignored generated asset tree. Updated the plan to
add `make test-runtimebundle` as the generated-assets-before-test path after
`embed.go` exists, keep earlier pure tests as raw `go test`, and spell out the
peer-repo `AGENTS.local.md` / `MEMORY.md` requirement before any optional
Homebrew tap edit.

Implemented the embedded runtime path. Added the generated runtime manifest and
bundle generator, pure manifest/extraction/cleanup planning, embedded asset
reader, runtime extraction store, and `cmd/pair-go` fallback that extracts to
`$PAIR_DATA_DIR/runtime/<digest>/pair-home` only after `PAIR_HOME`, executable
sibling assets, and build-time `defaultPairHome` fail. Source/Homebrew adjacent
layouts remain first in the selection order.

Added copied-binary smoke coverage with fake external dependencies for `pair
--help`, `pair resume smoke`, required extracted assets, `PAIR_HOME` handoff,
and stale-runtime cleanup. During verification, parallel smoke runs exposed that
`runtimebundle-generate` rewrote the shared output tree in place; added a
regression test for preserving existing output on failed generation and changed
the generator to stage output in a unique temp directory before replacing the
published bundle (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`).

Updated `README.md`, `atlas/architecture.md`, and
`atlas/go-migration-inventory.md` to document the implemented embedded fallback,
manifest ownership, cleanup behavior, and remaining external dependencies.
No Homebrew tap edit was needed because the adjacent `libexec` packaging path
remains accurate and intentionally precedes embedded fallback.

Verification passed:

- `make test-runtimebundle`
- `make runtimebundle-drift-check`
- `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
- `make build`
- `make test-pair-go-install-layout`
- `make test-pair-embedded-runtime`
- `go test ./... -count=1`
- `sdlc issue validate workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md`
- `git diff --check`

First `sdlc close --issue 90` boundary review returned REWORK. Fixed the
blocking findings by honoring `PAIR_DATA_DIR` before XDG/home fallback for
embedded extraction, adding divergent-env copied-binary smoke coverage,
updating the plan's `BuildManifest` path with a `## Revisions` entry, adding
`store.go` to `PAIR_GO_SRCS`, tightening stale-runtime cleanup to 64-character
manifest digests with matching markers, and reusing `runtimebundle` manifest
types from the generator. Also strengthened `runtimebundle-drift-check` to
compare generated file modes and recorded a new lesson for path precedence and
build-prerequisite coverage. A parallel verification run also exposed a
generator publish race; added an interprocess publish lock and concurrent
same-output generator regression test.

Review-fix verification passed:

- `go test ./cmd/internal/runtimebundle ./cmd/internal/runtimebundlegen -count=1`
- `go test ./cmd/pair-go -run 'TestRuntimeDataDir|TestRunDirectPairFallsBackToEmbeddedRuntime' -count=1`
- `make runtimebundle-drift-check`
- `make test-pair-embedded-runtime`
- `make test-runtimebundle`
- `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`
- `make build`
- `make test-pair-go-install-layout test-pair-embedded-runtime`
- `go test ./... -count=1`
- `sdlc issue validate workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md`
- `git diff --check`

 succeeded in 0ms:
# Self-Contained Pair Binary Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Build a `pair` binary that can launch from a clean copied location by extracting embedded Pair-owned runtime assets into a versioned runtime root.

**Architecture:** Add a generated runtime manifest as the single source for packaged assets (`ARCH-DRY`), a pure `runtimebundle` package for manifest hashing, version selection, extraction planning, and cleanup planning (`ARCH-PURE`), and a thin `cmd/pair-go` integration that only falls back to embedded extraction when `PAIR_HOME`, executable siblings, and `defaultPairHome` do not provide `bin/pair-shell`. This satisfies the issue purpose by proving copied-binary launch without relying on a source checkout (`ARCH-PURPOSE`).

**Tech Stack:** Go 1.26, `embed.FS`, Makefile build rules, POSIX shell smoke tests, existing `cmd/internal/entrypoint` launch handoff.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `RuntimeAsset` | `cmd/internal/runtimebundle/manifest.go` | new |
| `RuntimeManifest` | `cmd/internal/runtimebundle/manifest.go` | new |
| `ExtractionPlan` | `cmd/internal/runtimebundle/plan.go` | new |
| `RuntimeSelection` | `cmd/internal/entrypoint/asset_root.go` | modified |
| `CleanupPlan` | `cmd/internal/runtimebundle/cleanup.go` | new |

- **RuntimeAsset** — one embedded file entry: logical path, mode, size, and digest.
  - **Relationships:** N:1 with `RuntimeManifest`; one manifest owns many assets.
  - **DRY rationale:** Avoid parallel hand-maintained asset lists for embedding, install notes, and tests.
  - **Future extensions:** Can add generated-command shim metadata when helper binaries move behind dispatcher routes.

- **RuntimeManifest** — the deterministic package contract for Pair-owned runtime files.
  - **Relationships:** 1:N with `RuntimeAsset`; 1:1 with an embedded runtime version/digest.
  - **DRY rationale:** Single source for extraction, clean-copy smoke expectations, and atlas/Homebrew docs.
  - **Future extensions:** Can widen from static file assets to generated native assets.

- **ExtractionPlan** — pure decision for which files to write, chmod, leave alone, or reject.
  - **Relationships:** 1:1 with one target runtime root and one manifest.
  - **DRY rationale:** Keeps idempotency/upgrade rules out of filesystem code and makes them directly unit-testable.
  - **Future extensions:** Can add partial refresh or corrupted-file repair without changing callers.

- **RuntimeSelection** — existing launch-root decision extended to include an optional embedded runtime root after adjacent candidates fail.
  - **Relationships:** `cmd/pair-go` asks for one `RuntimeSelection`; it yields the `AssetRoot` passed to `ResolveLegacyLaunch`.
  - **DRY rationale:** Reuse `ResolveAssetRoot` ordering instead of adding a second launch resolver.
  - **Future extensions:** Can remove adjacent/default fallbacks when embedded mode becomes the primary release shape.

- **CleanupPlan** — pure stale-runtime pruning decision that never returns the currently selected runtime.
  - **Relationships:** N old runtime dirs to zero or more delete candidates.
  - **DRY rationale:** Shared cleanup policy for first-run extraction and future maintenance commands.
  - **Future extensions:** Retention count/age can become configurable without touching extraction.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `RuntimeFS` | `cmd/internal/runtimebundle/embed.go` | new | `embed.FS` |
| `RuntimeStore` | `cmd/internal/runtimebundle/store.go` | new | filesystem writes, chmod, rename |
| `PairGoLegacyRuntime` | `cmd/pair-go/main.go` | modified | `os`, `syscall.Exec`, env |
| `BuildManifest` | `cmd/internal/runtimebundlegen/generate.go`, `cmd/internal/runtimebundle/generatecmd/main.go`, `cmd/internal/runtimebundle/assets/**`, `Makefile.local` | new | repo file tree at build time |
| `CopiedBinarySmoke` | `tests/pair-embedded-runtime-test.sh` | new | real built binary with fake external commands |

- **RuntimeFS** — exposes embedded asset bytes and manifest data.
  - **Injected into:** `RuntimeStore` extraction runner and `cmd/pair-go` runtime setup.
  - **Future extensions:** Swap generated bytes for generated config content without changing launch.

- **RuntimeStore** — applies `ExtractionPlan` atomically under the user Pair data/cache area.
  - **Injected into:** `cmd/pair-go` through the existing `legacyRuntime` seam.
  - **Future extensions:** Add lock-file serialization if concurrent first-runs race on the same runtime root.

- **PairGoLegacyRuntime** — OS-backed seam that resolves paths, environment, extraction, and final exec.
  - **Injected into:** `runWithLegacyRuntime` tests via fake runtime.
  - **Future extensions:** Native launcher can reuse the same data/cache path helpers.

- **BuildManifest** — Makefile/generator path that keeps embedded assets synchronized with repo runtime files.
  - **Injected into:** `go generate` or `make build` before compiling `cmd/pair-go`.
  - **Future extensions:** Homebrew can audit the manifest rather than restating asset paths.

#### Runtime Asset Boundary

The embedded bundle must be a generated, gitignored package-local asset tree under
`cmd/internal/runtimebundle/assets/`; Go's `embed` package reads that tree at
compile time. The generator is the single source of the file list (`ARCH-DRY`):
tests and docs audit the generated `RuntimeManifest`, not a second list.

Required inputs:

- `bin/pair-shell`, `bin/pair-help`, `bin/pair-notify`, `bin/pair-quit.sh`,
  `bin/pair-restart.sh`, `bin/pair-scrollback-open`,
  `bin/pair-changelog-open`, `bin/pair-review-open`,
  `bin/pair-review-readiness`, `bin/pair-review-target`,
  `bin/copy-on-select.sh`, `bin/clipboard-to-pane.sh`, `bin/flash-pane.sh`,
  `bin/pair-title.sh`, `bin/pair-session-watch.sh`, and `bin/lib/**`.
- Go-built helper binaries used by the retained shell/KDL/Lua runtime:
  `bin/pair-wrap`, `bin/pair-slug`, `bin/pair-context`,
  `bin/pair-scrollback-render`, `bin/pair-changelog`,
  `bin/pair-continuation`, and `bin/pair-session-watch`.
- Native assets `nvim/**`, `zellij/**`, and runtime diagnostic assets
  `doctor/README.md`, `doctor/SKILL.md`, `doctor/doctor.sh`,
  `doctor/emitter-health.sh`.

Excluded inputs:

- Public entrypoints `bin/pair`, `bin/pair-go`, and `bin/pair-dev` to avoid
  embedding the launching binary or dev-only wrapper.
- External programs (`zellij`, `nvim`, `fzf`, `jq`, clipboard tools, agent CLIs),
  generated caches (`__pycache__`, `.DS_Store`), test files, and workflow/docs
  that are not read by launch/session flows.

Generator contract:

- `make runtimebundle-generate` removes and rebuilds the gitignored asset tree
  deterministically after helper binaries are built.
- `make runtimebundle-drift-check` regenerates twice into temp dirs and compares
  manifest JSON plus file bytes/modes; nondeterminism fails.
- `make test-runtimebundle` depends on helper binaries plus
  `runtimebundle-generate`, then runs `go test ./cmd/internal/runtimebundle
  -count=1`. After `embed.go` exists, use this target for package tests from a
  clean checkout; raw `go test ./cmd/internal/runtimebundle` is only valid after
  the asset tree has been generated.
- `make build` depends on `runtimebundle-generate` before building `bin/pair` and
  `bin/pair-go`; helper binaries are built first so the embedded bundle contains
  the current helper executables.

Runtime root contract:

- The extracted root is `$PAIR_DATA_DIR/runtime/<manifest-digest>/pair-home`,
  where `<manifest-digest>` is the `RuntimeManifest` digest. The directory that
  cleanup treats as one runtime generation is `$PAIR_DATA_DIR/runtime/<digest>/`.
- Extraction writes `$PAIR_DATA_DIR/runtime/<digest>/manifest.json` after all
  assets land under `pair-home/`; the marker includes the manifest digest,
  generated-at build metadata if available, and the asset count.
- Cleanup only considers direct children of `$PAIR_DATA_DIR/runtime/` whose names
  are valid manifest digests and whose `manifest.json` parses as a Pair runtime
  marker. It must always exclude the selected `<digest>` generation, even if the
  generation is older than the retention policy.
- Tests and production code derive behavior from the generated `RuntimeManifest`
  and runtime marker only. `atlas/go-migration-inventory.md` remains review
  context and documentation; no code or automated test should parse atlas as an
  input source.

- **CopiedBinarySmoke** — process-level fake test for external dependencies and copied-binary launch.
  - **Injected into:** `make test` via a focused target.
  - **Future extensions:** Can become the release packaging smoke for Homebrew and tarball builds.

## Chunk 1: Manifest And Pure Runtime Planning

### Task 1: Define the manifest contract

**Files:**
- Create: `cmd/internal/runtimebundle/manifest.go`
- Create: `cmd/internal/runtimebundle/manifest_test.go`

- [x] **Step 1: Write failing tests for manifest validation and digest stability**

Add tests that construct a small manifest in memory and assert:

```go
func TestManifestDigestIsOrderIndependent(t *testing.T) {
	// Same logical assets in different input order produce the same digest.
}

func TestManifestRejectsUnsafePaths(t *testing.T) {
	// Absolute paths, "..", empty path, and duplicate paths fail validation.
}
```

- [x] **Step 2: Run the red tests**

Run: `go test ./cmd/internal/runtimebundle -run 'TestManifest' -count=1`

Expected: package or symbols missing.

- [x] **Step 3: Implement `RuntimeAsset`, `RuntimeManifest`, validation, and digest**

Keep this pure: no filesystem or embed access. Normalize paths with slash semantics, sort entries for hashing, and include path, mode, size, and content digest in the manifest digest.

- [x] **Step 4: Run green tests**

Run: `go test ./cmd/internal/runtimebundle -run 'TestManifest' -count=1`

Expected: pass.

### Task 2: Define extraction and cleanup plans

**Files:**
- Create: `cmd/internal/runtimebundle/plan.go`
- Create: `cmd/internal/runtimebundle/plan_test.go`
- Create: `cmd/internal/runtimebundle/cleanup.go`
- Create: `cmd/internal/runtimebundle/cleanup_test.go`

- [x] **Step 1: Write failing tests for extraction idempotency**

Cover these behaviors:

```go
func TestPlanExtractionWritesMissingAssets(t *testing.T) {}
func TestPlanExtractionSkipsMatchingAssets(t *testing.T) {}
func TestPlanExtractionRefreshesMismatchedDigest(t *testing.T) {}
func TestPlanExtractionRejectsRuntimeRootOutsideStore(t *testing.T) {}
```

- [x] **Step 2: Write failing tests for cleanup safety**

Cover keeping the selected `$PAIR_DATA_DIR/runtime/<digest>/` generation, retaining the newest configured count, and deleting only directories that match the runtime root contract above.

- [x] **Step 3: Run the red tests**

Run: `go test ./cmd/internal/runtimebundle -run 'TestPlan|TestCleanup' -count=1`

Expected: missing functions or failing behavior.

- [x] **Step 4: Implement pure planning**

Represent write/skip/delete as data. Do not call `os.WriteFile`, `os.RemoveAll`, or `time.Now` from the pure functions; pass observed filesystem state and current runtime identity in.

- [x] **Step 5: Run green tests**

Run: `go test ./cmd/internal/runtimebundle -count=1`

Expected: pass.

## Chunk 2: Embedded Asset Generation And Extraction IO

### Task 3: Generate or maintain one embedded asset source

**Files:**
- Create: `cmd/internal/runtimebundle/generate.go`
- Create or generate: `cmd/internal/runtimebundle/embed.go`
- Generate, gitignored: `cmd/internal/runtimebundle/assets/**`
- Modify: `.gitignore`
- Modify: `Makefile.local`
- Test: `cmd/internal/runtimebundle/embed_test.go`

- [x] **Step 1: Write failing tests that assert all required runtime assets exist in the embedded manifest**

Assert the exact asset boundary above: launcher shell, shell helper scripts, `bin/lib/**`, required Go helper binaries, `nvim/**`, `zellij/**`, and doctor runtime diagnostics are present; excluded entrypoints/caches/tests are absent. Use `atlas/go-migration-inventory.md` only as human review context; automated tests inspect the generated manifest, not atlas.

- [x] **Step 2: Run the red test before adding embed code**

Run: `go test ./cmd/internal/runtimebundle -run TestEmbeddedManifestContainsLaunchAssets -count=1`

Expected: missing embedded manifest symbol. This red run happens before
`embed.go` is added, so it does not trip Go's compile-time `//go:embed` missing
file rule.

- [x] **Step 3: Add the deterministic generator and embedding path**

Implement `make runtimebundle-generate` to copy the required inputs into `cmd/internal/runtimebundle/assets/runtime/`, preserving executable bits and writing a manifest JSON with sorted paths. Add `//go:embed assets/runtime/**` and expose the embedded `RuntimeManifest` plus file reader through `RuntimeFS`.

- [x] **Step 4: Wire build, test, and staleness checks**

Update `PAIR_GO_SRCS` so changes to runtimebundle sources rebuild `bin/pair` and `bin/pair-go`. Add `test-runtimebundle` and `runtimebundle-drift-check`; include both in the relevant test path. Ensure `make build` builds helper binaries first, regenerates the asset tree, then builds `pair`/`pair-go` so the single binary embeds current helpers without committing generated binaries.

- [x] **Step 5: Run green tests**

Run: `make test-runtimebundle`

Expected: pass.

### Task 4: Implement filesystem extraction

**Files:**
- Create: `cmd/internal/runtimebundle/store.go`
- Create: `cmd/internal/runtimebundle/store_test.go`

- [x] **Step 1: Write failing filesystem tests**

Use `t.TempDir()` and a fake embedded asset reader. Assert atomic first-run extraction, executable bits for `bin/*` scripts, idempotent second run, manifest marker write, and stale cleanup that preserves the selected runtime.

- [x] **Step 2: Run the red tests**

Run: `go test ./cmd/internal/runtimebundle -run 'TestStore|TestExtract' -count=1`

Expected: missing store implementation.

- [x] **Step 3: Implement the thin IO shell**

Use temp files plus rename for file writes. Write a manifest marker after all assets land. Apply cleanup after successful extraction/selection, not before.

- [x] **Step 4: Run green tests**

Run: `make test-runtimebundle`

Expected: pass.

## Chunk 3: Launch Integration And Smoke Coverage

### Task 5: Extend asset-root selection to embedded fallback

**Files:**
- Modify: `cmd/internal/entrypoint/asset_root.go`
- Modify: `cmd/internal/entrypoint/asset_root_test.go`
- Modify: `cmd/pair-go/main.go`
- Modify: `cmd/pair-go/main_test.go`

- [x] **Step 1: Write failing tests for launch-root precedence**

Assert ordering:

1. `PAIR_HOME` with `bin/pair-shell` wins.
2. Executable sibling root wins.
3. `defaultPairHome` wins.
4. Embedded extracted runtime is used only when the adjacent/default candidates fail.

- [x] **Step 2: Write failing `cmd/pair-go` fake-runtime tests**

Extend the existing `legacyRuntime` fake so launch records `PAIR_HOME=<extracted-root>` in the exec env when embedded fallback is selected.

- [x] **Step 3: Run red tests**

Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -run 'TestResolveAssetRoot|TestRun' -count=1`

Expected: embedded fallback behavior missing.

- [x] **Step 4: Implement embedded fallback**

Keep `ResolveAssetRoot` deterministic and side-effect-free by passing the already-prepared embedded root as an optional candidate. Let `cmd/pair-go` decide whether extraction is needed and then pass the selected root into the existing legacy launch handoff.

- [x] **Step 5: Run green tests**

Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`

Expected: pass.

### Task 6: Add copied-binary and upgrade smoke tests

**Files:**
- Create: `tests/pair-embedded-runtime-test.sh`
- Modify: `Makefile.local`

- [x] **Step 1: Write the failing clean-copy help smoke test**

Build `bin/pair`, copy only that binary to a temp `PATH`, set `PAIR_DATA_DIR` to a temp dir, and run a harmless launch path such as `--help` through `pair-shell`. Use fake external commands only where needed so the test does not start a real zellij session.

- [x] **Step 2: Write the failing session-asset smoke test**

Run the copied binary through a create-path with fake `zellij`, `nvim`, `fzf`,
`jq`, `par`, and fake agent commands on `PATH`. The fake `zellij` must record the
`--config-dir` and `--new-session-with-layout` arguments, read both files, and
fail if they do not come from the extracted `PAIR_HOME`. The fake layout check
must verify the embedded runtime contains and can resolve `pair-wrap`,
`pair-session-watch.sh`, `pair-title.sh`, and `nvim/init.lua`. This smoke proves
the copied binary reaches the compatibility launch/session asset path, not only
the help text.

- [x] **Step 3: Add upgrade/stale-runtime assertions**

Simulate an old extracted runtime under the temp data/cache area and assert it is pruned only after the new runtime is selected; assert the current runtime remains.

- [x] **Step 4: Run the red smoke**

Run: `bash tests/pair-embedded-runtime-test.sh`

Expected: copied binary cannot find `pair-shell` before implementation.

- [x] **Step 5: Add the `make` target**

Add `test-pair-embedded-runtime` and `runtimebundle-drift-check` to `.PHONY`; add the smoke target to the main `test` dependency list after `test-pair-go-install-layout`.

- [x] **Step 6: Run green smoke**

Run: `make test-pair-embedded-runtime`

Expected: copied-binary clean and upgrade paths pass.

## Chunk 4: Docs, Packaging Notes, And Final Verification

### Task 7: Update docs and atlas

**Files:**
- Modify: `atlas/go-migration-inventory.md`
- Modify: `atlas/architecture.md`
- Modify: `atlas/index.md` if a new atlas page is added
- Modify: `README.md`
- Optional modify: sibling `../homebrew-pair/Formula/pair.rb`

- [x] **Step 1: Update atlas execution path**

Record that #90 embeds/extracts Pair-owned runtime assets and that adjacent layout remains supported for source/Homebrew compatibility.

- [x] **Step 2: Update install docs**

Document the copied-binary deployment mode and keep external dependencies explicit.

- [x] **Step 3: Audit Homebrew notes**

This issue does not require moving Homebrew to binary-only packaging. If the formula still packages adjacent assets, leave behavior unchanged and document it as compatibility mode in `README.md` / atlas. Touch `../homebrew-pair/Formula/pair.rb` only if it contains false adjacent-only claims or build rules that conflict with the generated bundle; if touched, verify with `ruby -c ../homebrew-pair/Formula/pair.rb` and record the sibling commit.

Before touching `../homebrew-pair`, follow the peer-repo rule from AGENTS.md:
read that repo's `AGENTS.local.md` and `MEMORY.md` first, then keep its change in
a separate sibling commit and record that commit hash in this issue's `## Log`.

### Task 8: Run verification and close-ready checks

**Files:**
- Modify: `workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md`

- [x] **Step 1: Run focused Go tests**

Run: `make test-runtimebundle`

Expected: pass.

Run: `go test ./cmd/internal/entrypoint ./cmd/pair-go -count=1`

Expected: pass.

- [x] **Step 2: Run smoke and build tests**

Run: `make build`

Expected: pass.

Run: `make test-pair-go-install-layout test-pair-embedded-runtime`

Expected: pass.

- [x] **Step 3: Run broader Go tests**

Run: `go test ./... -count=1`

Expected: pass.

- [x] **Step 4: Validate issue and diff hygiene**

Run: `sdlc issue validate workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md`

Expected: pass.

Run: `git diff --check`

Expected: no whitespace errors.

- [x] **Step 5: Update issue plan/log**

Tick completed issue plan items, add verification evidence to `## Log`, and prepare for `sdlc close --issue 90 --verified '<evidence>' --no-atlas` only if atlas truly had no new surface. This issue is expected to require atlas updates, so avoid `--no-atlas` unless the implementation scope changes.

## Revisions

### 2026-07-01 — Boundary review fixes

Reason: the SDLC boundary review found that the Core Concepts table named the
generator package path incorrectly after implementation split the generator from
the embedded runtime package.

Delta: updated `BuildManifest` to point at
`cmd/internal/runtimebundlegen/generate.go` and
`cmd/internal/runtimebundle/generatecmd/main.go`, leaving the generated
gitignored assets and Makefile integration as originally planned.

exec
/bin/zsh -lc 'git diff --stat e1ae416c6084399c155e0a89c5597712f679460b..HEAD && git diff --name-only e1ae416c6084399c155e0a89c5597712f679460b..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 .gitignore                                         |   1 +
 Makefile.local                                     |  34 ++-
 README.md                                          |  19 +-
 atlas/architecture.md                              |  22 +-
 atlas/go-migration-inventory.md                    |  35 +--
 cmd/internal/entrypoint/asset_root.go              |   4 +
 cmd/internal/entrypoint/asset_root_test.go         |  18 ++
 cmd/internal/runtimebundle/cleanup.go              |  68 ++++++
 cmd/internal/runtimebundle/cleanup_test.go         |  49 +++++
 cmd/internal/runtimebundle/embed.go                |  25 +++
 cmd/internal/runtimebundle/embed_test.go           |  45 ++++
 cmd/internal/runtimebundle/generatecmd/main.go     |  19 ++
 cmd/internal/runtimebundle/manifest.go             |  73 +++++++
 cmd/internal/runtimebundle/manifest_test.go        |  58 +++++
 cmd/internal/runtimebundle/plan.go                 |  63 ++++++
 cmd/internal/runtimebundle/plan_test.go            |  78 +++++++
 cmd/internal/runtimebundle/store.go                | 187 ++++++++++++++++
 cmd/internal/runtimebundle/store_test.go           | 137 ++++++++++++
 cmd/internal/runtimebundlegen/generate.go          | 236 +++++++++++++++++++++
 cmd/internal/runtimebundlegen/generate_test.go     |  84 ++++++++
 cmd/pair-go/main.go                                |  68 +++++-
 cmd/pair-go/main_test.go                           |  50 +++++
 tests/pair-embedded-runtime-test.sh                | 118 +++++++++++
 ...ned-pair-binary-with-embedded-runtime-assets.md |  81 ++++++-
 workshop/lessons.md                                |  15 ++
 ...air-binary-with-embedded-runtime-assets-plan.md |  91 ++++----
 26 files changed, 1595 insertions(+), 83 deletions(-)
.gitignore
Makefile.local
README.md
atlas/architecture.md
atlas/go-migration-inventory.md
cmd/internal/entrypoint/asset_root.go
cmd/internal/entrypoint/asset_root_test.go
cmd/internal/runtimebundle/cleanup.go
cmd/internal/runtimebundle/cleanup_test.go
cmd/internal/runtimebundle/embed.go
cmd/internal/runtimebundle/embed_test.go
cmd/internal/runtimebundle/generatecmd/main.go
cmd/internal/runtimebundle/manifest.go
cmd/internal/runtimebundle/manifest_test.go
cmd/internal/runtimebundle/plan.go
cmd/internal/runtimebundle/plan_test.go
cmd/internal/runtimebundle/store.go
cmd/internal/runtimebundle/store_test.go
cmd/internal/runtimebundlegen/generate.go
cmd/internal/runtimebundlegen/generate_test.go
cmd/pair-go/main.go
cmd/pair-go/main_test.go
tests/pair-embedded-runtime-test.sh
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md
workshop/lessons.md
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md

codex
I’m cross-checking the implementation now. The first pass shows the core split is sensible: manifest/cleanup planning is pure, extraction is in a store layer, and `cmd/pair-go` remains the only exec/env integration point.
exec
/bin/zsh -lc "nl -ba cmd/internal/runtimebundlegen/generate.go | sed -n '1,280p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package runtimebundlegen
     2	
     3	import (
     4		"crypto/sha256"
     5		"encoding/hex"
     6		"encoding/json"
     7		"fmt"
     8		"io"
     9		"io/fs"
    10		"os"
    11		"path/filepath"
    12		"sort"
    13		"strings"
    14		"time"
    15	
    16		"github.com/xianxu/pair/cmd/internal/runtimebundle"
    17	)
    18	
    19	var explicitAssetPaths = []string{
    20		"bin/pair-shell",
    21		"bin/pair-help",
    22		"bin/pair-notify",
    23		"bin/pair-quit.sh",
    24		"bin/pair-restart.sh",
    25		"bin/pair-scrollback-open",
    26		"bin/pair-changelog-open",
    27		"bin/pair-review-open",
    28		"bin/pair-review-readiness",
    29		"bin/pair-review-target",
    30		"bin/copy-on-select.sh",
    31		"bin/clipboard-to-pane.sh",
    32		"bin/flash-pane.sh",
    33		"bin/pair-title.sh",
    34		"bin/pair-session-watch.sh",
    35		"bin/pair-wrap",
    36		"bin/pair-slug",
    37		"bin/pair-context",
    38		"bin/pair-scrollback-render",
    39		"bin/pair-changelog",
    40		"bin/pair-continuation",
    41		"bin/pair-session-watch",
    42		"doctor/README.md",
    43		"doctor/SKILL.md",
    44		"doctor/doctor.sh",
    45		"doctor/emitter-health.sh",
    46	}
    47	
    48	var assetDirs = []string{
    49		"bin/lib",
    50		"nvim",
    51		"zellij",
    52	}
    53	
    54	type GenerateOptions struct {
    55		RepoRoot string
    56		OutRoot  string
    57	}
    58	
    59	func Generate(opts GenerateOptions) (runtimebundle.RuntimeManifest, error) {
    60		if opts.RepoRoot == "" {
    61			opts.RepoRoot = "."
    62		}
    63		if opts.OutRoot == "" {
    64			return runtimebundle.RuntimeManifest{}, fmt.Errorf("output root is required")
    65		}
    66		repoRoot, err := filepath.Abs(opts.RepoRoot)
    67		if err != nil {
    68			return runtimebundle.RuntimeManifest{}, err
    69		}
    70		outRoot, err := filepath.Abs(opts.OutRoot)
    71		if err != nil {
    72			return runtimebundle.RuntimeManifest{}, err
    73		}
    74		outParent := filepath.Dir(outRoot)
    75		outBase := filepath.Base(outRoot)
    76		if err := os.MkdirAll(outParent, 0o755); err != nil {
    77			return runtimebundle.RuntimeManifest{}, err
    78		}
    79		stageRoot, err := os.MkdirTemp(outParent, "."+outBase+"-tmp-")
    80		if err != nil {
    81			return runtimebundle.RuntimeManifest{}, err
    82		}
    83		committed := false
    84		defer func() {
    85			if !committed {
    86				_ = os.RemoveAll(stageRoot)
    87			}
    88		}()
    89		filesRoot := filepath.Join(stageRoot, "files")
    90	
    91		paths := map[string]bool{}
    92		for _, p := range explicitAssetPaths {
    93			paths[p] = true
    94		}
    95		for _, dir := range assetDirs {
    96			root := filepath.Join(repoRoot, filepath.FromSlash(dir))
    97			if err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
    98				if err != nil {
    99					return err
   100				}
   101				if d.IsDir() {
   102					return nil
   103				}
   104				rel, err := filepath.Rel(repoRoot, p)
   105				if err != nil {
   106					return err
   107				}
   108				logical := filepath.ToSlash(rel)
   109				if shouldExclude(logical) {
   110					return nil
   111				}
   112				paths[logical] = true
   113				return nil
   114			}); err != nil {
   115				return runtimebundle.RuntimeManifest{}, err
   116			}
   117		}
   118	
   119		ordered := make([]string, 0, len(paths))
   120		for p := range paths {
   121			if shouldExclude(p) {
   122				continue
   123			}
   124			ordered = append(ordered, p)
   125		}
   126		sort.Strings(ordered)
   127	
   128		manifest := runtimebundle.RuntimeManifest{Assets: make([]runtimebundle.RuntimeAsset, 0, len(ordered))}
   129		for _, logical := range ordered {
   130			src := filepath.Join(repoRoot, filepath.FromSlash(logical))
   131			info, err := os.Stat(src)
   132			if err != nil {
   133				return runtimebundle.RuntimeManifest{}, fmt.Errorf("asset %s: %w", logical, err)
   134			}
   135			if info.IsDir() {
   136				continue
   137			}
   138			digest, err := copyAsset(src, filepath.Join(filesRoot, filepath.FromSlash(logical)), info.Mode().Perm())
   139			if err != nil {
   140				return runtimebundle.RuntimeManifest{}, err
   141			}
   142			manifest.Assets = append(manifest.Assets, runtimebundle.RuntimeAsset{
   143				Path:   logical,
   144				Mode:   uint32(info.Mode().Perm()),
   145				Size:   info.Size(),
   146				Digest: "sha256:" + digest,
   147			})
   148		}
   149		encoded, err := json.MarshalIndent(manifest, "", "  ")
   150		if err != nil {
   151			return runtimebundle.RuntimeManifest{}, err
   152		}
   153		encoded = append(encoded, '\n')
   154		if err := os.WriteFile(filepath.Join(stageRoot, "manifest.json"), encoded, 0o644); err != nil {
   155			return runtimebundle.RuntimeManifest{}, err
   156		}
   157		unlock, err := acquirePublishLock(outRoot + ".lock")
   158		if err != nil {
   159			return runtimebundle.RuntimeManifest{}, err
   160		}
   161		defer unlock()
   162		if err := os.RemoveAll(outRoot); err != nil {
   163			return runtimebundle.RuntimeManifest{}, err
   164		}
   165		if err := os.Rename(stageRoot, outRoot); err != nil {
   166			return runtimebundle.RuntimeManifest{}, err
   167		}
   168		committed = true
   169		return manifest, nil
   170	}
   171	
   172	func acquirePublishLock(path string) (func(), error) {
   173		const attempts = 1000
   174		for i := 0; i < attempts; i++ {
   175			err := os.Mkdir(path, 0o755)
   176			if err == nil {
   177				return func() { _ = os.Remove(path) }, nil
   178			}
   179			if !os.IsExist(err) {
   180				return nil, err
   181			}
   182			time.Sleep(10 * time.Millisecond)
   183		}
   184		return nil, fmt.Errorf("timed out waiting for runtime bundle publish lock %s", path)
   185	}
   186	
   187	func shouldExclude(logical string) bool {
   188		base := filepath.Base(logical)
   189		if base == ".DS_Store" || strings.Contains(logical, "__pycache__/") {
   190			return true
   191		}
   192		if strings.HasSuffix(logical, "_test.lua") {
   193			return true
   194		}
   195		switch logical {
   196		case "bin/pair", "bin/pair-go", "bin/pair-dev":
   197			return true
   198		}
   199		return false
   200	}
   201	
   202	func copyAsset(src, dst string, mode os.FileMode) (string, error) {
   203		in, err := os.Open(src)
   204		if err != nil {
   205			return "", err
   206		}
   207		defer in.Close()
   208		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
   209			return "", err
   210		}
   211		tmp := dst + ".tmp"
   212		out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
   213		if err != nil {
   214			return "", err
   215		}
   216		h := sha256.New()
   217		_, copyErr := io.Copy(io.MultiWriter(out, h), in)
   218		closeErr := out.Close()
   219		if copyErr != nil {
   220			_ = os.Remove(tmp)
   221			return "", copyErr
   222		}
   223		if closeErr != nil {
   224			_ = os.Remove(tmp)
   225			return "", closeErr
   226		}
   227		if err := os.Chmod(tmp, mode); err != nil {
   228			_ = os.Remove(tmp)
   229			return "", err
   230		}
   231		if err := os.Rename(tmp, dst); err != nil {
   232			_ = os.Remove(tmp)
   233			return "", err
   234		}
   235		return hex.EncodeToString(h.Sum(nil)), nil
   236	}

exec
/bin/zsh -lc "nl -ba Makefile.local | sed -n '1,330p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	# Repo-specific Makefile targets.
     2	# This file is included by Makefile — add your own targets here.
     3	
     4	.PHONY: build install pair pair-scribe pair-scrollback-render pair-wrap pair-slug pair-changelog pair-context pair-session-watch pair-go runtimebundle-generate runtimebundle-drift-check test-runtimebundle test test-race test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-pair-embedded-runtime test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace
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
    32	RUNTIMEBUNDLE_HELPERS := pair-scrollback-render pair-wrap pair-slug pair-continuation pair-changelog pair-context pair-session-watch
    33	RUNTIMEBUNDLE_ASSETS := cmd/internal/runtimebundle/assets/runtime/manifest.json
    34	
    35	build: $(addprefix $(BIN_DIR)/,$(GO_BINS))
    36	
    37	install: build
    38		@mkdir -p $(INSTALL_DIR)
    39		@for b in $(GO_BINS); do \
    40		    install -m 755 $(BIN_DIR)/$$b $(INSTALL_DIR)/$$b && \
    41		    echo "installed: $(INSTALL_DIR)/$$b"; \
    42		done
    43		@for b in $(SHELL_BINS); do \
    44		    ln -sf "$(CURDIR)/$(BIN_DIR)/$$b" "$(INSTALL_DIR)/$$b" && \
    45		    echo "linked: $(INSTALL_DIR)/$$b -> $(CURDIR)/$(BIN_DIR)/$$b"; \
    46		done
    47	
    48	# Per-binary aliases — `make pair-scribe`, etc.
    49	pair:                   $(BIN_DIR)/pair
    50	pair-scribe:            $(BIN_DIR)/pair-scribe
    51	pair-scrollback-render: $(BIN_DIR)/pair-scrollback-render
    52	pair-wrap:              $(BIN_DIR)/pair-wrap
    53	pair-slug:              $(BIN_DIR)/pair-slug
    54	pair-continuation:      $(BIN_DIR)/pair-continuation
    55	pair-changelog:         $(BIN_DIR)/pair-changelog
    56	pair-context:           $(BIN_DIR)/pair-context
    57	pair-session-watch:     $(BIN_DIR)/pair-session-watch
    58	pair-go:                $(BIN_DIR)/pair-go
    59	
    60	# Tests
    61	#   make test       → go test ./... (cache disabled so reruns actually run)
    62	#   make test-race  → race-detector run, scoped to packages where the
    63	#                     suite has actual concurrent code worth checking.
    64	#                     pair-wrap has translateStdin's goroutine + timer +
    65	#                     channels; pair-scrollback-render's render() spawns
    66	#                     a goroutine drainer that races with em.Close()
    67	#                     inside the vt.Emulator library API itself, which is
    68	#                     outside our scope to fix — exclude that package
    69	#                     here rather than chase a library-level race.
    70	test: test-run-headless test-lua test-queue test-autopair test-cr test-statusline test-doctor test-adapt-schema test-dev-rebuild test-pair-go-install-layout test-pair-embedded-runtime test-emitter-health test-session-watch test-continue test-changelog test-copy-on-select test-pair-title test-cmux-ownership test-zellij-trace test-review
    71		go test ./... -count=1
    72	
    73	# pair-title poller: single-instance guard (identity-checked liveness so a
    74	# recycled PID can't suppress the respawn) + the #71 frame meter (renames each
    75	# agent pane to "<agent> (<count>) [<cwd>]", with unchanged-skip). See
    76	# tests/pair-title-poller-test.sh.
    77	test-pair-title:
    78		bash tests/pair-title-poller-test.sh
    79	
    80	# cmux workspace ownership claim (presence beats a stale owner flag), so a
    81	# launch/attach/restart reclaims its workspace title. See
    82	# tests/cmux-ownership-test.sh.
    83	test-cmux-ownership: $(BIN_DIR)/pair
    84		bash tests/cmux-ownership-test.sh
    85	
    86	# Session id watcher must ignore stale pidfiles from prior launches and wait
    87	# for pair-wrap to publish the current agent PID.
    88	test-session-watch: $(BIN_DIR)/pair-session-watch
    89		bash tests/pair-session-watch-test.sh
    90	
    91	runtimebundle-generate: $(addprefix $(BIN_DIR)/,$(RUNTIMEBUNDLE_HELPERS))
    92		go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime
    93	
    94	$(RUNTIMEBUNDLE_ASSETS): runtimebundle-generate
    95	
    96	runtimebundle-drift-check: $(addprefix $(BIN_DIR)/,$(RUNTIMEBUNDLE_HELPERS))
    97		@d1="$$(mktemp -d)"; d2="$$(mktemp -d)"; \
    98		trap 'rm -rf "$$d1" "$$d2"' EXIT; \
    99		go run ./cmd/internal/runtimebundle/generatecmd --repo . --out "$$d1/runtime" && \
   100		go run ./cmd/internal/runtimebundle/generatecmd --repo . --out "$$d2/runtime" && \
   101		diff -qr "$$d1/runtime" "$$d2/runtime" >/dev/null && \
   102		find "$$d1/runtime/files" -type f -exec stat -f '%N %OLp' {} + | sed "s#$$d1/runtime/files/##" | sort > "$$d1/modes" && \
   103		find "$$d2/runtime/files" -type f -exec stat -f '%N %OLp' {} + | sed "s#$$d2/runtime/files/##" | sort > "$$d2/modes" && \
   104		diff -u "$$d1/modes" "$$d2/modes" >/dev/null
   105	
   106	test-runtimebundle: runtimebundle-generate
   107		go test ./cmd/internal/runtimebundle -count=1
   108	
   109	# Review workbench (#66): the integration shell tests. The pure modules
   110	# (record, reconstruct) run under `make test-lua`.
   111	test-review:
   112		bash tests/review-docflow-test.sh
   113		bash tests/review-apply-test.sh
   114		bash tests/review-handoff-test.sh
   115		bash tests/review-loop-test.sh
   116		bash tests/review-projection-test.sh
   117		bash tests/review-poke-test.sh
   118		bash tests/review-window-test.sh
   119		bash tests/review-toggle-test.sh
   120		bash tests/pair-review-target-test.sh
   121		bash tests/review-readiness-cli-test.sh
   122		bash tests/review-indicator-test.sh
   123		bash tests/review-resume-test.sh
   124	
   125	test-race:
   126		go test -count=1 -race ./cmd/pair-wrap/
   127	
   128	# Self-test for the headless-nvim timeout watchdog (tests/lib/run-headless.sh,
   129	# #60). Runs FIRST in `test` so the safety net every headless-nvim sub-target
   130	# now relies on is proven before those targets run — without it, a driver that
   131	# fails to quit (bare `qall` on a modified buffer → E37) hangs the whole suite.
   132	test-run-headless:
   133		bash tests/run-headless-test.sh
   134	
   135	# Headless Lua tests for nvim/*.lua decision modules (issue #000027). nvim -l
   136	# runs a Lua script in nvim's interpreter with the full vim.api available, so
   137	# pure logic AND buffer-manipulation can be asserted without a UI.
   138	test-lua:
   139		nvim -l nvim/slug_test.lua
   140		nvim -l nvim/scrollback_test.lua
   141		nvim -l nvim/annotate_test.lua
   142		nvim -l nvim/changelog_test.lua
   143		nvim -l nvim/adapt_test.lua
   144		nvim -l nvim/doctor_test.lua
   145		nvim -l nvim/review/record_test.lua
   146		nvim -l nvim/review/reconstruct_test.lua
   147		nvim -l nvim/review/markers_test.lua
   148		nvim -l nvim/review/seam_test.lua
   149		nvim -l nvim/review/mode_test.lua
   150		nvim -l nvim/review/poke_bodies_test.lua
   151		nvim -l nvim/review/wrap_test.lua
   152		nvim -l nvim/review/readiness_test.lua
   153		nvim -l nvim/review/spinner_test.lua
   154		nvim -l nvim/review/resolve_test.lua
   155		nvim -l nvim/review/menu_test.lua
   156	
   157	# Integration test for nvim/init.lua's send/queue/history state machine.
   158	# Drives the real init.lua under headless nvim (it can't be unit-tested —
   159	# monolithic config, all-local fns). Guards the send-from-queue duplication
   160	# regression. See tests/queue-send-test.sh.
   161	test-queue:
   162		bash tests/queue-send-test.sh
   163	
   164	# Diagnostic trace for pair-originated zellij action calls (#68).
   165	test-zellij-trace:
   166		bash tests/zellij-trace-test.sh
   167	
   168	# Regression test for the autopair next-char gate (nvim/init.lua,
   169	# pair_insert_open). Drives the real init.lua headlessly and asserts each
   170	# opener's expr output. See tests/autopair-test.sh.
   171	test-autopair:
   172		bash tests/autopair-test.sh
   173	
   174	# Regression test for the insert-mode <CR> decision (nvim/init.lua, cr_keys /
   175	# _G.PairCRKeys, #65): Return must always insert a newline when no completion
   176	# is selected, dismissing the popup. Asserts the expr output for all three
   177	# popup states. See tests/cr-newline-test.sh.
   178	test-cr:
   179		bash tests/cr-newline-test.sh
   180	
   181	# Regression tests for the draft statusline (nvim/init.lua, _G.PairStatusline).
   182	# Drive the real init.lua headlessly: statusline-pos asserts the *, -N, +N nav
   183	# marker; changelog-notify asserts the ephemeral "change log ready" flash + its
   184	# fs_event marker watcher (#58).
   185	test-statusline:
   186		bash tests/statusline-pos-test.sh
   187		bash tests/changelog-notify-test.sh
   188	
   189	# Regression test for doctor/doctor.sh (the adaptation flight-recorder reader):
   190	# a malformed line must not abort the diagnostic. See doctor/doctor_test.sh.
   191	test-doctor:
   192		bash doctor/doctor_test.sh
   193	
   194	# copy-on-select in_nvim detection: the agent overwrites its pane title with
   195	# "claude [<cwd>]", so a repo path containing "nvim" (e.g. parley.nvim) must not
   196	# be mistaken for the nvim draft pane (which would skip the paste). Keys on
   197	# terminal_command, not the cwd-polluted title. See tests/copy-on-select-test.sh.
   198	test-copy-on-select:
   199		bash tests/copy-on-select-test.sh
   200	
   201	# Cross-emitter contract for the flight recorder: Go == shell == Lua schema,
   202	# plus concurrent-append line-atomicity. See tests/adapt-schema-test.sh.
   203	test-adapt-schema:
   204		bash tests/adapt-schema-test.sh
   205	
   206	# pair-dev rebuild hook (#000046): deployed mode (PAIR_DEV unset) must invoke
   207	# no toolchain; dev mode builds; build failure is errexit-safe. See
   208	# tests/dev-rebuild-test.sh.
   209	test-dev-rebuild:
   210		bash tests/dev-rebuild-test.sh
   211	
   212	# Go public entrypoint install layout (#79): installed pair is a regular Go
   213	# binary, pair-go launch shares its pair-shell handoff, and copied local
   214	# installs fall back to the build-time source root for adjacent assets.
   215	test-pair-go-install-layout:
   216		bash tests/pair-go-install-layout-test.sh
   217	
   218	test-pair-embedded-runtime:
   219		bash tests/pair-embedded-runtime-test.sh
   220	
   221	# pair-doctor emitter-health probe (#000047): marker check + binary selection
   222	# (running-via-pidfile vs PATH). See tests/emitter-health-test.sh.
   223	test-emitter-health:
   224		bash tests/emitter-health-test.sh
   225	
   226	# `pair continue` arg-parsing + session-name guard (#000054): tag not forced,
   227	# -- args forwarded, [agent] port, bare list / error paths, guard short-vs-long.
   228	# Drives the real bin/pair via its PAIR_DEBUG_ARGS probe. See tests/pair-continue-test.sh.
   229	test-continue: $(BIN_DIR)/pair
   230		bash tests/pair-continue-test.sh
   231	
   232	# changelog — end-to-end smoke for bin/pair-changelog-open (#53 M2): fakes the
   233	# scrollback capture + model + nvim, asserts clean → distill → write → open.
   234	# Depends on the binaries so `make test` builds + runs it (the script SKIPs if
   235	# they're absent, which would silently no-op in CI without these prereqs).
   236	test-changelog: $(BIN_DIR)/pair-changelog $(BIN_DIR)/pair-scrollback-render
   237		sh tests/changelog-open-test.sh
   238		sh tests/changelog-session-key-test.sh
   239	
   240	# pair-scribe — Go-based pty wrapper with SIGUSR1/SIGUSR2 pause/resume
   241	# of the typescript log. Replaces script(1) at the top of the user's
   242	# shell, so preexec can stop the log from filling with TUI redraw bytes
   243	# (claude, nvim, etc). See cmd/pair-scribe/README.md for the full why.
   244	$(BIN_DIR)/pair-scribe: cmd/pair-scribe/main.go go.mod
   245		go build -o $@ ./cmd/pair-scribe
   246	
   247	# pair-scrollback-render — replays a pair-wrap raw PTY capture through
   248	# charmbracelet/x/vt and emits one ANSI-styled line per logical scrollback
   249	# row, for the Alt+/ viewer to display through nvim.
   250	$(BIN_DIR)/pair-scrollback-render: cmd/pair-scrollback-render/main.go cmd/internal/scrollbackcmd/scrollbackcmd.go go.mod
   251		go build -o $@ ./cmd/pair-scrollback-render
   252	
   253	# pair-wrap — transparent PTY proxy around a TUI coding agent: forwards
   254	# stdio, tees scrollback log, extracts colored spans for nvim autocomplete,
   255	# detects OSC/BEL and forwards as OSC 9 to the recorded outer-TTY, handles
   256	# SIGUSR1 image-paste capture window.
   257	$(BIN_DIR)/pair-wrap: cmd/pair-wrap/main.go go.mod
   258		go build -o $@ ./cmd/pair-wrap
   259	
   260	$(BIN_DIR)/pair-slug: cmd/pair-slug/main.go cmd/pair-slug/slug.go cmd/internal/model/model.go cmd/internal/transcript/transcript.go go.mod
   261		go build -o $@ ./cmd/pair-slug
   262	
   263	# pair-context — one-shot: print a pane's current context size (token count)
   264	# for the pair-title poller. Pure ctxmeter reader + shared transcript resolver.
   265	$(BIN_DIR)/pair-context: cmd/pair-context/main.go cmd/internal/contextcmd/contextcmd.go cmd/internal/ctxmeter/ctxmeter.go cmd/internal/transcript/transcript.go go.mod
   266		go build -o $@ ./cmd/pair-context
   267	
   268	# pair-session-watch — async codex/agy session-id discovery. Legacy
   269	# bin/pair-session-watch.sh remains as a compatibility shim for bin/pair.
   270	$(BIN_DIR)/pair-session-watch: cmd/pair-session-watch/main.go cmd/internal/sessionwatch/sessionwatch.go cmd/internal/sessionwatch/run.go cmd/internal/sessionwatch/runtime.go cmd/internal/adapt/adapt.go go.mod
   271		go build -o $@ ./cmd/pair-session-watch
   272	
   273	# pair-changelog — distills the pair TTY into the Alt+l change log (#53);
   274	# shares cmd/internal/model with pair-slug.
   275	$(BIN_DIR)/pair-changelog: cmd/pair-changelog/main.go cmd/pair-changelog/distill.go cmd/pair-changelog/prompt.go cmd/internal/model/model.go go.mod
   276		go build -o $@ ./cmd/pair-changelog
   277	
   278	# pair-continuation — the deterministic writer for the `continuation` datatype
   279	# (ariadne#91): renders frontmatter, allocates a collision-safe timestamped
   280	# name under workshop/continuation/, writes, then commits + pushes (recovery).
   281	$(BIN_DIR)/pair-continuation: cmd/pair-continuation/continuation.go cmd/pair-continuation/git.go cmd/pair-continuation/main.go go.mod
   282		go build -o $@ ./cmd/pair-continuation
   283	
   284	PAIR_GO_SRCS := cmd/pair-go/main.go cmd/internal/entrypoint/asset_root.go cmd/internal/entrypoint/launch.go cmd/internal/entrypoint/mode.go cmd/internal/dispatcher/dispatcher.go cmd/internal/launcher/args.go cmd/internal/launcher/datadir.go cmd/internal/launcher/decision.go cmd/internal/launcher/history.go cmd/internal/launcher/run.go cmd/internal/launcher/session.go cmd/internal/launcher/tag.go cmd/internal/launcher/zellij.go cmd/internal/contextcmd/contextcmd.go cmd/internal/scrollbackcmd/scrollbackcmd.go cmd/internal/runtimebundle/manifest.go cmd/internal/runtimebundle/plan.go cmd/internal/runtimebundle/cleanup.go cmd/internal/runtimebundle/store.go cmd/internal/runtimebundle/embed.go $(RUNTIMEBUNDLE_ASSETS) go.mod
   285	
   286	# pair — public Go-owned entrypoint. It execs bin/pair-shell for the current
   287	# shell-owned zellij lifecycle; source/Homebrew builds can inject
   288	# defaultPairHome, while copied binaries without adjacent assets use the
   289	# embedded runtime fallback.
   290	$(BIN_DIR)/pair: $(PAIR_GO_SRCS)
   291		go build -ldflags "$(PAIR_HOME_LDFLAGS)" -o $@ ./cmd/pair-go
   292	
   293	# pair-go — development dispatcher and explicit migration surface.
   294	# `pair-go launch ...` shares the same bin/pair-shell handoff as public pair;
   295	# helper routes stay in the dispatcher.
   296	$(BIN_DIR)/pair-go: $(PAIR_GO_SRCS)
   297		go build -ldflags "$(PAIR_HOME_LDFLAGS)" -o $@ ./cmd/pair-go
   298	
   299	
   300	# Tart VM targets (tart, tart-gui, tart-mount, tart-stop, tart-clean)
   301	# live in .tart/Makefile, vendored from ariadne and auto-included by
   302	# Makefile.workflow. `make help-tart` for usage.

exec
/bin/zsh -lc "nl -ba cmd/pair-go/main.go | sed -n '1,240p'" in /Users/xianxu/workspace/pair
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
    10		"strings"
    11		"syscall"
    12	
    13		"github.com/xianxu/pair/cmd/internal/dispatcher"
    14		"github.com/xianxu/pair/cmd/internal/entrypoint"
    15		"github.com/xianxu/pair/cmd/internal/launcher"
    16		"github.com/xianxu/pair/cmd/internal/runtimebundle"
    17	)
    18	
    19	var defaultPairHome string
    20	
    21	func main() {
    22		os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
    23	}
    24	
    25	func run(args []string, stdout, stderr io.Writer) int {
    26		return runWithLegacyRuntime(args, stdout, stderr, osLegacyRuntime{})
    27	}
    28	
    29	type legacyRuntime interface {
    30		Executable() (string, error)
    31		PairHome() string
    32		DefaultPairHome() string
    33		Stat(path string) error
    34		Environ() []string
    35		EmbeddedAssetRoot() (string, error)
    36		Exec(label string, path string, argv []string, env []string) int
    37	}
    38	
    39	func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
    40		exe, err := rt.Executable()
    41		if err != nil {
    42			if len(args) > 0 && args[0] == "launch" {
    43				_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
    44				return 1
    45			}
    46			res := dispatcher.Dispatch(args)
    47			return writeResult(res, stdout, stderr)
    48		}
    49	
    50		switch entrypoint.ClassifyInvocation(exe, args) {
    51		case entrypoint.ModePublicPair:
    52			return runLegacyLaunch("pair", exe, args, stderr, rt)
    53		case entrypoint.ModePairGoLaunch:
    54			return runLegacyLaunch("pair-go launch", exe, args[1:], stderr, rt)
    55		default:
    56			res := dispatcher.Dispatch(args)
    57			return writeResult(res, stdout, stderr)
    58		}
    59	}
    60	
    61	func runLegacyLaunch(label string, executable string, args []string, stderr io.Writer, rt legacyRuntime) int {
    62		root, err := entrypoint.ResolveAssetRoot(entrypoint.AssetRootInput{
    63			PairHome:        rt.PairHome(),
    64			Executable:      executable,
    65			DefaultPairHome: rt.DefaultPairHome(),
    66			PairShellExists: func(root string) bool {
    67				return rt.Stat(entrypoint.PairShellPath(root)) == nil
    68			},
    69		})
    70		if err != nil {
    71			embeddedRoot, embeddedErr := rt.EmbeddedAssetRoot()
    72			if embeddedErr == nil && embeddedRoot != "" {
    73				root, err = entrypoint.ResolveAssetRoot(entrypoint.AssetRootInput{
    74					PairHome:        rt.PairHome(),
    75					Executable:      executable,
    76					DefaultPairHome: rt.DefaultPairHome(),
    77					EmbeddedRoot:    embeddedRoot,
    78					PairShellExists: func(root string) bool {
    79						return rt.Stat(entrypoint.PairShellPath(root)) == nil
    80					},
    81				})
    82			}
    83			if err != nil {
    84				if embeddedErr != nil {
    85					_, _ = fmt.Fprintf(stderr, "%s: embedded runtime extraction failed: %v\n", label, embeddedErr)
    86				}
    87				_, _ = fmt.Fprintf(stderr, "%s: %v; run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", label, err)
    88				return 1
    89			}
    90		}
    91		req := entrypoint.ResolveLegacyLaunch(root, args)
    92		return rt.Exec(label, req.Path, req.Argv, withEnv(rt.Environ(), "PAIR_HOME", root.Root))
    93	}
    94	
    95	type osLegacyRuntime struct{}
    96	
    97	func (osLegacyRuntime) Executable() (string, error) {
    98		return os.Executable()
    99	}
   100	
   101	func (osLegacyRuntime) PairHome() string {
   102		return os.Getenv("PAIR_HOME")
   103	}
   104	
   105	func (osLegacyRuntime) DefaultPairHome() string {
   106		return defaultPairHome
   107	}
   108	
   109	func (osLegacyRuntime) Stat(path string) error {
   110		path = filepath.Clean(path)
   111		info, err := os.Stat(path)
   112		if err != nil {
   113			return err
   114		}
   115		if info.IsDir() {
   116			return fmt.Errorf("is a directory")
   117		}
   118		return nil
   119	}
   120	
   121	func (osLegacyRuntime) Environ() []string {
   122		return os.Environ()
   123	}
   124	
   125	func (osLegacyRuntime) EmbeddedAssetRoot() (string, error) {
   126		dataDir := runtimeDataDir(os.Getenv("PAIR_DATA_DIR"), os.Getenv("HOME"), os.Getenv("XDG_DATA_HOME"))
   127		res, err := runtimebundle.Extract(runtimebundle.StoreInput{
   128			StoreRoot: filepath.Join(dataDir, "runtime"),
   129			Manifest:  runtimebundle.EmbeddedManifest(),
   130			ReadAsset: runtimebundle.EmbeddedAsset,
   131			Keep:      2,
   132		})
   133		if err != nil {
   134			return "", err
   135		}
   136		return res.PairHome, nil
   137	}
   138	
   139	func runtimeDataDir(pairDataDir, home, xdgDataHome string) string {
   140		if pairDataDir != "" {
   141			return pairDataDir
   142		}
   143		return launcher.ResolveDataDir(home, xdgDataHome)
   144	}
   145	
   146	func (osLegacyRuntime) Exec(label string, path string, argv []string, env []string) int {
   147		if err := syscall.Exec(path, argv, env); err != nil {
   148			_, _ = fmt.Fprintf(os.Stderr, "%s: exec %s failed: %v\n", label, path, err)
   149			return 1
   150		}
   151		return 0
   152	}
   153	
   154	func writeResult(res dispatcher.Result, stdout, stderr io.Writer) int {
   155		if res.Stdout != "" {
   156			_, _ = io.WriteString(stdout, res.Stdout)
   157		}
   158		if res.Stderr != "" {
   159			_, _ = io.WriteString(stderr, res.Stderr)
   160		}
   161		return res.ExitCode
   162	}
   163	
   164	func withEnv(env []string, key, value string) []string {
   165		prefix := key + "="
   166		out := make([]string, 0, len(env)+1)
   167		replaced := false
   168		for _, entry := range env {
   169			if strings.HasPrefix(entry, prefix) {
   170				if !replaced {
   171					out = append(out, prefix+value)
   172					replaced = true
   173				}
   174				continue
   175			}
   176			out = append(out, entry)
   177		}
   178		if !replaced {
   179			out = append(out, prefix+value)
   180		}
   181		return out
   182	}

exec
/bin/zsh -lc "nl -ba cmd/internal/runtimebundle/store.go | sed -n '1,240p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package runtimebundle
     2	
     3	import (
     4		"crypto/sha256"
     5		"encoding/hex"
     6		"encoding/json"
     7		"fmt"
     8		"os"
     9		"path/filepath"
    10		"sort"
    11		"time"
    12	)
    13	
    14	type StoreInput struct {
    15		StoreRoot string
    16		Manifest  RuntimeManifest
    17		ReadAsset func(string) ([]byte, error)
    18		Keep      int
    19	}
    20	
    21	type StoreResult struct {
    22		Digest   string
    23		PairHome string
    24	}
    25	
    26	type runtimeMarker struct {
    27		Digest     string `json:"digest"`
    28		AssetCount int    `json:"asset_count"`
    29		Generated  string `json:"generated,omitempty"`
    30	}
    31	
    32	func Extract(input StoreInput) (StoreResult, error) {
    33		if input.ReadAsset == nil {
    34			return StoreResult{}, fmt.Errorf("asset reader is required")
    35		}
    36		digest, err := input.Manifest.ManifestDigest()
    37		if err != nil {
    38			return StoreResult{}, err
    39		}
    40		pairHome := filepath.Join(input.StoreRoot, digest, "pair-home")
    41		existing, err := scanExisting(pairHome, input.Manifest)
    42		if err != nil {
    43			return StoreResult{}, err
    44		}
    45		plan, err := PlanExtraction(ExtractionInput{
    46			StoreRoot:   input.StoreRoot,
    47			RuntimeRoot: pairHome,
    48			Manifest:    input.Manifest,
    49			Existing:    existing,
    50		})
    51		if err != nil {
    52			return StoreResult{}, err
    53		}
    54		for _, asset := range plan.Writes {
    55			data, err := input.ReadAsset(asset.Path)
    56			if err != nil {
    57				return StoreResult{}, fmt.Errorf("read embedded asset %s: %w", asset.Path, err)
    58			}
    59			if digestFor(string(data)) != asset.Digest {
    60				// Binary assets may not be valid UTF-8 but string preserves bytes.
    61				return StoreResult{}, fmt.Errorf("embedded asset %s digest mismatch", asset.Path)
    62			}
    63			if int64(len(data)) != asset.Size {
    64				return StoreResult{}, fmt.Errorf("embedded asset %s size mismatch", asset.Path)
    65			}
    66			if err := writeFileAtomic(filepath.Join(pairHome, filepath.FromSlash(asset.Path)), data, os.FileMode(asset.Mode)); err != nil {
    67				return StoreResult{}, err
    68			}
    69		}
    70		if err := writeMarker(filepath.Join(input.StoreRoot, digest, "manifest.json"), digest, len(input.Manifest.Assets)); err != nil {
    71			return StoreResult{}, err
    72		}
    73		if err := applyCleanup(input.StoreRoot, digest, input.Keep); err != nil {
    74			return StoreResult{}, err
    75		}
    76		return StoreResult{Digest: digest, PairHome: pairHome}, nil
    77	}
    78	
    79	func scanExisting(root string, manifest RuntimeManifest) (map[string]ExistingAsset, error) {
    80		existing := map[string]ExistingAsset{}
    81		for _, asset := range manifest.Assets {
    82			p := filepath.Join(root, filepath.FromSlash(asset.Path))
    83			info, err := os.Stat(p)
    84			if err != nil {
    85				if os.IsNotExist(err) {
    86					continue
    87				}
    88				return nil, err
    89			}
    90			if info.IsDir() {
    91				continue
    92			}
    93			data, err := os.ReadFile(p)
    94			if err != nil {
    95				return nil, err
    96			}
    97			h := sha256.Sum256(data)
    98			existing[asset.Path] = ExistingAsset{
    99				Mode:   uint32(info.Mode().Perm()),
   100				Size:   info.Size(),
   101				Digest: "sha256:" + hex.EncodeToString(h[:]),
   102			}
   103		}
   104		return existing, nil
   105	}
   106	
   107	func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
   108		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
   109			return err
   110		}
   111		tmp := path + ".tmp"
   112		if err := os.WriteFile(tmp, data, mode); err != nil {
   113			_ = os.Remove(tmp)
   114			return err
   115		}
   116		if err := os.Chmod(tmp, mode); err != nil {
   117			_ = os.Remove(tmp)
   118			return err
   119		}
   120		if err := os.Rename(tmp, path); err != nil {
   121			_ = os.Remove(tmp)
   122			return err
   123		}
   124		return nil
   125	}
   126	
   127	func writeMarker(path, digest string, assetCount int) error {
   128		data, err := json.MarshalIndent(runtimeMarker{
   129			Digest:     digest,
   130			AssetCount: assetCount,
   131			Generated:  time.Now().UTC().Format(time.RFC3339),
   132		}, "", "  ")
   133		if err != nil {
   134			return err
   135		}
   136		data = append(data, '\n')
   137		return writeFileAtomic(path, data, 0o644)
   138	}
   139	
   140	func applyCleanup(storeRoot, selectedDigest string, keep int) error {
   141		entries, err := os.ReadDir(storeRoot)
   142		if err != nil {
   143			if os.IsNotExist(err) {
   144				return nil
   145			}
   146			return err
   147		}
   148		gens := make([]RuntimeGeneration, 0, len(entries))
   149		for _, entry := range entries {
   150			if !entry.IsDir() {
   151				continue
   152			}
   153			name := entry.Name()
   154			info, err := entry.Info()
   155			if err != nil {
   156				return err
   157			}
   158			gens = append(gens, RuntimeGeneration{
   159				Digest:    name,
   160				HasMarker: markerValid(filepath.Join(storeRoot, name, "manifest.json"), name),
   161				ModUnix:   info.ModTime().Unix(),
   162			})
   163		}
   164		plan, err := PlanCleanup(CleanupInput{SelectedDigest: selectedDigest, Keep: keep, Generations: gens})
   165		if err != nil {
   166			return err
   167		}
   168		sort.Strings(plan.DeleteDigests)
   169		for _, digest := range plan.DeleteDigests {
   170			if err := os.RemoveAll(filepath.Join(storeRoot, digest)); err != nil {
   171				return err
   172			}
   173		}
   174		return nil
   175	}
   176	
   177	func markerValid(path, digest string) bool {
   178		data, err := os.ReadFile(path)
   179		if err != nil {
   180			return false
   181		}
   182		var marker runtimeMarker
   183		if err := json.Unmarshal(data, &marker); err != nil {
   184			return false
   185		}
   186		return marker.Digest == digest
   187	}

codex
I’ve found one area worth probing harder: first-run extraction has no store-level serialization, while the generator now does. I’m checking whether that is an actual shipped behavior risk or just an unneeded future hardening note.
exec
/bin/zsh -lc "nl -ba tests/pair-embedded-runtime-test.sh | sed -n '1,170p'" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "nl -ba cmd/internal/entrypoint/asset_root.go | sed -n '1,130p' && nl -ba cmd/internal/entrypoint/launch.go | sed -n '1,160p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	#!/usr/bin/env bash
     2	set -euo pipefail
     3	
     4	repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
     5	tmp="$(mktemp -d "${TMPDIR:-/tmp}/pair-embedded-runtime.XXXXXX")"
     6	trap 'rm -rf "$tmp"' EXIT
     7	
     8	bin_dir="$tmp/bin"
     9	home="$tmp/home"
    10	xdg="$tmp/xdg"
    11	pair_data="$tmp/custom-data"
    12	mkdir -p "$bin_dir" "$home" "$xdg" "$pair_data"
    13	gomodcache="$(go env GOMODCACHE)"
    14	gocache="$(go env GOCACHE)"
    15	
    16	make -C "$repo_root" runtimebundle-generate >/dev/null
    17	go build -o "$bin_dir/pair" "$repo_root/cmd/pair-go"
    18	
    19	cat >"$bin_dir/zellij" <<'SH'
    20	#!/usr/bin/env bash
    21	set -eu
    22	printf '%s\n' "$*" >> "${ZELLIJ_LOG:?}"
    23	case "$*" in
    24	  "list-sessions --no-formatting"|"list-sessions --short")
    25	    exit 0
    26	    ;;
    27	  --session*" action list-clients")
    28	    exit 0
    29	    ;;
    30	  --config-dir*)
    31	    config=""
    32	    layout=""
    33	    prev=""
    34	    for arg in "$@"; do
    35	      if [ "$prev" = "--config-dir" ]; then config="$arg"; fi
    36	      if [ "$prev" = "--new-session-with-layout" ]; then layout="$arg"; fi
    37	      prev="$arg"
    38	    done
    39	    test -f "$config/config.kdl"
    40	    test -f "$layout"
    41	    case "$config" in */custom-data/runtime/*/pair-home/zellij) ;; *) printf 'bad config path: %s\n' "$config" >&2; exit 11 ;; esac
    42	    case "$layout" in */custom-data/runtime/*/pair-home/zellij/layouts/main.kdl) ;; *) printf 'bad layout path: %s\n' "$layout" >&2; exit 12 ;; esac
    43	    root="${config%/zellij}"
    44	    test -x "$root/bin/pair-shell"
    45	    test -x "$root/bin/pair-wrap"
    46	    test -x "$root/bin/pair-session-watch.sh"
    47	    test -x "$root/bin/pair-title.sh"
    48	    test -f "$root/nvim/init.lua"
    49	    printf '%s\n' "$root" > "${PAIR_SMOKE_ROOT:?}"
    50	    exit 0
    51	    ;;
    52	  *)
    53	    exit 0
    54	    ;;
    55	esac
    56	SH
    57	chmod +x "$bin_dir/zellij"
    58	
    59	cat >"$bin_dir/ps" <<'SH'
    60	#!/usr/bin/env bash
    61	case "$*" in
    62	  "-o comm= -p "*)
    63	    printf 'sh\n'
    64	    ;;
    65	  "-o ppid= -p "*)
    66	    printf '1\n'
    67	    ;;
    68	  *)
    69	    exec /bin/ps "$@"
    70	    ;;
    71	esac
    72	SH
    73	chmod +x "$bin_dir/ps"
    74	
    75	export PATH="$bin_dir:$PATH"
    76	export HOME="$home"
    77	export XDG_DATA_HOME="$xdg"
    78	export PAIR_DATA_DIR="$pair_data"
    79	export GOMODCACHE="$gomodcache"
    80	export GOCACHE="$gocache"
    81	export ZELLIJ_LOG="$tmp/zellij.log"
    82	export PAIR_SMOKE_ROOT="$tmp/root"
    83	unset PAIR_DEV PAIR_HOME PAIR_TAG PAIR_AGENT PAIR_AGENT_ARGS ZELLIJ_SESSION_NAME ZELLIJ ZELLIJ_PANE_ID
    84	
    85	help_out="$("$bin_dir/pair" --help)"
    86	case "$help_out" in
    87	  pair\ —*) ;;
    88	  *)
    89	    printf 'copied pair --help did not reach embedded pair-shell help; first bytes:\n%s\n' "$help_out" >&2
    90	    exit 1
    91	    ;;
    92	esac
    93	
    94	old_a="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    95	old_b="bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
    96	old_c="cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
    97	mkdir -p "$pair_data/runtime/$old_a/pair-home" \
    98	         "$pair_data/runtime/$old_b/pair-home" \
    99	         "$pair_data/runtime/$old_c/pair-home"
   100	printf '{"digest":"%s","asset_count":0}\n' "$old_a" > "$pair_data/runtime/$old_a/manifest.json"
   101	printf '{"digest":"%s","asset_count":0}\n' "$old_b" > "$pair_data/runtime/$old_b/manifest.json"
   102	printf '{"digest":"%s","asset_count":0}\n' "$old_c" > "$pair_data/runtime/$old_c/manifest.json"
   103	touch -t 202001010000 "$pair_data/runtime/$old_a"
   104	touch -t 202001020000 "$pair_data/runtime/$old_b"
   105	touch -t 202001030000 "$pair_data/runtime/$old_c"
   106	
   107	"$bin_dir/pair" resume smoke >/dev/null
   108	
   109	test -s "$PAIR_SMOKE_ROOT"
   110	root="$(cat "$PAIR_SMOKE_ROOT")"
   111	case "$root" in */custom-data/runtime/*/pair-home) ;; *) printf 'bad extracted root: %s\n' "$root" >&2; exit 1 ;; esac
   112	test -d "$root"
   113	test ! -e "$pair_data/runtime/$old_a"
   114	test -d "$pair_data/runtime/$old_b"
   115	test -d "$pair_data/runtime/$old_c"
   116	test ! -e "$xdg/pair/runtime"
   117	
   118	printf 'pair embedded runtime smoke passed\n'

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
    13		EmbeddedRoot    string
    14		PairShellExists func(root string) bool
    15	}
    16	
    17	type AssetRoot struct {
    18		Root      string
    19		ShellPath string
    20		Source    string
    21	}
    22	
    23	func ResolveAssetRoot(input AssetRootInput) (AssetRoot, error) {
    24		exists := input.PairShellExists
    25		if exists == nil {
    26			exists = func(string) bool { return false }
    27		}
    28	
    29		candidates := make([]assetRootCandidate, 0, 3)
    30		if input.PairHome != "" {
    31			candidates = append(candidates, assetRootCandidate{root: input.PairHome, source: "PAIR_HOME"})
    32		}
    33		if input.Executable != "" {
    34			candidates = append(candidates, assetRootCandidate{
    35				root:   filepath.Dir(filepath.Dir(input.Executable)),
    36				source: "executable sibling",
    37			})
    38		}
    39		if input.DefaultPairHome != "" {
    40			candidates = append(candidates, assetRootCandidate{root: input.DefaultPairHome, source: "defaultPairHome"})
    41		}
    42		if input.EmbeddedRoot != "" {
    43			candidates = append(candidates, assetRootCandidate{root: input.EmbeddedRoot, source: "embedded runtime"})
    44		}
    45	
    46		seen := map[string]bool{}
    47		checked := make([]string, 0, len(candidates))
    48		for _, candidate := range candidates {
    49			root := filepath.Clean(candidate.root)
    50			if root == "." || seen[root] {
    51				continue
    52			}
    53			seen[root] = true
    54			checked = append(checked, root)
    55			if exists(root) {
    56				return AssetRoot{
    57					Root:      root,
    58					ShellPath: PairShellPath(root),
    59					Source:    candidate.source,
    60				}, nil
    61			}
    62		}
    63	
    64		if len(checked) == 0 {
    65			checked = append(checked, "<none>")
    66		}
    67		return AssetRoot{}, fmt.Errorf("pair-shell not found; set PAIR_HOME to a Pair checkout/install root containing bin/pair-shell (checked: %s)", strings.Join(checked, ", "))
    68	}
    69	
    70	func PairShellPath(root string) string {
    71		return filepath.Join(root, "bin", "pair-shell")
    72	}
    73	
    74	type assetRootCandidate struct {
    75		root   string
    76		source string
    77	}
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
/bin/zsh -lc "nl -ba cmd/internal/runtimebundle/manifest.go | sed -n '1,140p' && nl -ba cmd/internal/runtimebundle/plan.go | sed -n '1,120p' && nl -ba cmd/internal/runtimebundle/cleanup.go | sed -n '1,100p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	package runtimebundle
     2	
     3	import (
     4		"crypto/sha256"
     5		"encoding/hex"
     6		"errors"
     7		"fmt"
     8		"path"
     9		"sort"
    10		"strings"
    11	)
    12	
    13	type RuntimeAsset struct {
    14		Path   string
    15		Mode   uint32
    16		Size   int64
    17		Digest string
    18	}
    19	
    20	type RuntimeManifest struct {
    21		Assets []RuntimeAsset
    22	}
    23	
    24	func (m RuntimeManifest) Validate() error {
    25		seen := map[string]bool{}
    26		for _, asset := range m.Assets {
    27			if err := validateAsset(asset); err != nil {
    28				return err
    29			}
    30			if seen[asset.Path] {
    31				return fmt.Errorf("duplicate asset path %q", asset.Path)
    32			}
    33			seen[asset.Path] = true
    34		}
    35		return nil
    36	}
    37	
    38	func (m RuntimeManifest) ManifestDigest() (string, error) {
    39		if err := m.Validate(); err != nil {
    40			return "", err
    41		}
    42		assets := append([]RuntimeAsset(nil), m.Assets...)
    43		sort.Slice(assets, func(i, j int) bool {
    44			return assets[i].Path < assets[j].Path
    45		})
    46		h := sha256.New()
    47		for _, asset := range assets {
    48			_, _ = fmt.Fprintf(h, "%s\x00%o\x00%d\x00%s\x00", asset.Path, asset.Mode, asset.Size, asset.Digest)
    49		}
    50		return hex.EncodeToString(h.Sum(nil)), nil
    51	}
    52	
    53	func validateAsset(asset RuntimeAsset) error {
    54		if asset.Path == "" {
    55			return errors.New("asset path is empty")
    56		}
    57		if strings.HasPrefix(asset.Path, "/") {
    58			return fmt.Errorf("asset path %q is absolute", asset.Path)
    59		}
    60		clean := path.Clean(asset.Path)
    61		if clean != asset.Path || clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
    62			return fmt.Errorf("asset path %q is not clean relative path", asset.Path)
    63		}
    64		if asset.Digest == "" {
    65			return fmt.Errorf("asset %q digest is empty", asset.Path)
    66		}
    67		return nil
    68	}
    69	
    70	func digestFor(s string) string {
    71		h := sha256.Sum256([]byte(s))
    72		return "sha256:" + hex.EncodeToString(h[:])
    73	}
     1	package runtimebundle
     2	
     3	import (
     4		"fmt"
     5		"path/filepath"
     6	)
     7	
     8	type ExistingAsset struct {
     9		Mode   uint32
    10		Size   int64
    11		Digest string
    12	}
    13	
    14	type ExtractionInput struct {
    15		StoreRoot   string
    16		RuntimeRoot string
    17		Manifest    RuntimeManifest
    18		Existing    map[string]ExistingAsset
    19	}
    20	
    21	type ExtractionPlan struct {
    22		Writes []RuntimeAsset
    23		Skips  []string
    24	}
    25	
    26	func PlanExtraction(input ExtractionInput) (ExtractionPlan, error) {
    27		if err := input.Manifest.Validate(); err != nil {
    28			return ExtractionPlan{}, err
    29		}
    30		if err := validateRuntimeRoot(input.StoreRoot, input.RuntimeRoot); err != nil {
    31			return ExtractionPlan{}, err
    32		}
    33		existing := input.Existing
    34		if existing == nil {
    35			existing = map[string]ExistingAsset{}
    36		}
    37		plan := ExtractionPlan{}
    38		for _, asset := range input.Manifest.Assets {
    39			got, ok := existing[asset.Path]
    40			if ok && got.Mode == asset.Mode && got.Size == asset.Size && got.Digest == asset.Digest {
    41				plan.Skips = append(plan.Skips, asset.Path)
    42				continue
    43			}
    44			plan.Writes = append(plan.Writes, asset)
    45		}
    46		return plan, nil
    47	}
    48	
    49	func validateRuntimeRoot(storeRoot, runtimeRoot string) error {
    50		if storeRoot == "" || runtimeRoot == "" {
    51			return fmt.Errorf("store root and runtime root are required")
    52		}
    53		store := filepath.Clean(storeRoot)
    54		root := filepath.Clean(runtimeRoot)
    55		rel, err := filepath.Rel(store, root)
    56		if err != nil {
    57			return err
    58		}
    59		if rel == "." || rel == ".." || rel == "" || (len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator)) {
    60			return fmt.Errorf("runtime root %q is outside store root %q", runtimeRoot, storeRoot)
    61		}
    62		return nil
    63	}
     1	package runtimebundle
     2	
     3	import (
     4		"fmt"
     5		"sort"
     6	)
     7	
     8	type RuntimeGeneration struct {
     9		Digest    string
    10		HasMarker bool
    11		ModUnix   int64
    12	}
    13	
    14	type CleanupInput struct {
    15		SelectedDigest string
    16		Keep           int
    17		Generations    []RuntimeGeneration
    18	}
    19	
    20	type CleanupPlan struct {
    21		DeleteDigests []string
    22	}
    23	
    24	func PlanCleanup(input CleanupInput) (CleanupPlan, error) {
    25		if input.SelectedDigest == "" {
    26			return CleanupPlan{}, fmt.Errorf("selected digest is required")
    27		}
    28		if input.Keep < 0 {
    29			return CleanupPlan{}, fmt.Errorf("keep must be non-negative")
    30		}
    31		candidates := make([]RuntimeGeneration, 0, len(input.Generations))
    32		for _, gen := range input.Generations {
    33			if gen.Digest == input.SelectedDigest || !gen.HasMarker || !isDigestName(gen.Digest) {
    34				continue
    35			}
    36			candidates = append(candidates, gen)
    37		}
    38		sort.Slice(candidates, func(i, j int) bool {
    39			if candidates[i].ModUnix == candidates[j].ModUnix {
    40				return candidates[i].Digest > candidates[j].Digest
    41			}
    42			return candidates[i].ModUnix > candidates[j].ModUnix
    43		})
    44		if input.Keep >= len(candidates) {
    45			return CleanupPlan{}, nil
    46		}
    47		deleteCandidates := candidates[input.Keep:]
    48		sort.Slice(deleteCandidates, func(i, j int) bool {
    49			return deleteCandidates[i].Digest < deleteCandidates[j].Digest
    50		})
    51		plan := CleanupPlan{DeleteDigests: make([]string, 0, len(deleteCandidates))}
    52		for _, gen := range deleteCandidates {
    53			plan.DeleteDigests = append(plan.DeleteDigests, gen.Digest)
    54		}
    55		return plan, nil
    56	}
    57	
    58	func isDigestName(s string) bool {
    59		if len(s) != 64 {
    60			return false
    61		}
    62		for _, r := range s {
    63			if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
    64				return false
    65			}
    66		}
    67		return true
    68	}

exec
/bin/zsh -lc "find cmd/internal/runtimebundle/assets/runtime -maxdepth 2 -type f | sed -n '1,40p'; test -f cmd/internal/runtimebundle/assets/runtime/manifest.json && wc -c cmd/internal/runtimebundle/assets/runtime/manifest.json" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
cmd/internal/runtimebundle/assets/runtime/manifest.json
   10296 cmd/internal/runtimebundle/assets/runtime/manifest.json

codex
The current risk profile looks mostly clean. I’m doing one last traceability pass over docs and tests, especially the paths the issue promises: `$PAIR_DATA_DIR` precedence, adjacent-root fallback, embedded extraction, cleanup, and atlas coverage.
exec
/bin/zsh -lc "nl -ba README.md | sed -n '195,225p' && nl -ba atlas/architecture.md | sed -n '42,72p' && nl -ba atlas/go-migration-inventory.md | sed -n '28,65p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
   195	pair rename <old> <new>          # rename every tag-scoped file in
   196	                                 # $PAIR_DATA_DIR from <old> to <new>;
   197	                                 # refuses if pair-<old> is still tracked
   198	                                 # by zellij (use Ctrl+Alt+n's (R) inside
   199	                                 # a session for the live equivalent)
   200	pair -h, --help                  # show full help
   201	```
   202	
   203	The installed `pair` command is Go-owned. For this migration window it resolves
   204	the Pair asset root and hands off to `bin/pair-shell`, the retained shell
   205	launcher that still owns the zellij lifecycle. Source and Homebrew installs use
   206	their adjacent asset roots. A copied standalone `pair` binary with no adjacent
   207	or build-time source root extracts its embedded Pair-owned runtime assets to
   208	`${XDG_DATA_HOME:-$HOME/.local/share}/pair/runtime/<digest>/pair-home` and runs
   209	with `PAIR_HOME` pointed there. External tools such as `zellij`, `nvim`, `fzf`,
   210	`jq`, clipboard tools, and agent CLIs are still installed separately.
   211	
   212	`pair-go launch ...` remains the explicit development dispatcher path and
   213	accepts the same arguments after `launch` that `pair` accepts directly. In a dev
   214	shell sourced from `../ariadne/construct/dev-aliases.sh`, `pair` and `pair-go`
   215	rebuild from `cmd/pair-go` automatically before running; no `pair-go-dev`
   216	command is needed.
   217	
   218	Use `--` to separate pair's positional from agent flags. Without it, pair only takes `<agent>` as a positional and everything else is rejected.
   219	
   220	Agent args (after `--`) are appended to the agent command line on **create**. Reattaching to an existing session does not re-launch the agent, so the args don't apply on attach. (The picker connects you to whatever's already running.)
   221	
   222	**Hacking on pair?** Use `pair-dev` instead of `pair` — same arguments, but it rebuilds pair's Go binaries from source (`make build`) on launch *and* on every Alt+n / Shift+Alt+N restart, so the zellij-spawned `pair-wrap` always matches your working tree. (Deployed installs run `pair`, which uses the prebuilt binary and needs no Go toolchain.)
   223	
   224	When `pair` runs and there's anything to pick — a detached `pair-*` session **or** a tag from this cwd used within the last 14 days — it shows an `fzf` picker. Detached rows come first, then historical rows annotated `(Nd ago, no live session)`, then a `+ new <agent> session` sentinel. A historical row whose session has prompts parked in its queue also carries an amber `[⏎ N queued]` badge, so you don't resume a session without remembering the work you queued up in it. Picking a historical row reuses the name and any surviving draft / saved agent config (same path as `pair resume <tag>`). Override the 14-day window with `PAIR_HISTORY_DAYS`; `PAIR_DEBUG_HISTORY=1 pair` prints the scan and exits without launching.
   225	
    42	lifecycle, data/config path resolution, asset discovery, restart/quit/continue
    43	flows, and subprocess orchestration. Existing Go command surfaces should become
    44	internal subcommands or dispatch modes behind that primary binary (`pair wrap`,
    45	`pair slug`, `pair context`, `pair scrollback-render`, `pair changelog`,
    46	`pair continuation`, `pair scribe`) instead of staying as independently managed
    47	installed commands forever.
    48	
    49	As of #90, the public `bin/pair` command is a Go-built entrypoint from
    50	`cmd/pair-go` with an embedded Pair-owned runtime bundle. Direct `pair ...` and
    51	explicit `pair-go launch ...` share one compatibility handoff: resolve the Pair
    52	asset root, then exec `<asset-root>/bin/pair-shell` with `pair`-compatible
    53	argv/env. Asset root resolution is ordered: explicit `PAIR_HOME`, executable
    54	sibling root, the build-time `defaultPairHome` injected by Make/Homebrew, then
    55	an extracted embedded runtime under `$PAIR_DATA_DIR/runtime/<digest>/pair-home`
    56	when no adjacent/source asset root exists. Native `nvim/` and `zellij/` assets
    57	remain native files inside whichever asset root was selected.
    58	
    59	The embedded runtime is generated from a deterministic manifest before builds
    60	and tests. That manifest is the packaging source of truth for bundled Pair-owned
    61	shell helpers, helper binaries, `bin/lib/`, `nvim/`, `zellij/`, and doctor
    62	assets; external programs such as `zellij`, `nvim`, `fzf`, `jq`, clipboard
    63	tools, and agent CLIs remain system dependencies.
    64	
    65	`pair-go` remains the development dispatcher for helper routes and explicit
    66	launch testing: `pair-go launch claude`, `pair-go launch resume <tag>`,
    67	`pair-go launch continue ...`, `pair-go launch list`, and
    68	`pair-go launch rename ...` all reach the same shell launcher as direct `pair`.
    69	A developer shell sourced from `../ariadne/construct/dev-aliases.sh` rebuilds
    70	`cmd/pair-go` automatically; no `pair-go-dev` command exists.
    71	
    72	The earlier #75 pure launcher core remains available as internal decision logic,
    28	
    29	#79 made the installed public `pair` command Go-owned. #90 added a
    30	self-contained deployment mode: the Go binary embeds the Pair-owned runtime
    31	assets and extracts them to `$PAIR_DATA_DIR/runtime/<digest>/pair-home` on
    32	demand when no adjacent/source/Homebrew asset root is available.
    33	
    34	This is not the same as "no external dependencies." The single-binary target is
    35	one Pair artifact. System programs such as `zellij`, `nvim`, clipboard tools,
    36	`fzf`, `jq` while retained shell needs it, and agent CLIs remain external unless
    37	a later issue explicitly replaces them.
    38	
    39	Execution path:
    40	
    41	1. **Embedded runtime bundle (#90):** the current Pair-owned runtime tree
    42	   (`bin/pair-shell`, shell helpers, helper binaries or dispatcher shims,
    43	   `bin/lib/`, `nvim/`, `zellij/`, and doctor/help assets needed at runtime) is
    44	   generated into a manifest-backed embedded bundle. On run, copied binaries
    45	   extract to a digest-named Pair data root, write a runtime marker, prune stale
    46	   older extracted runtimes without deleting the selected digest, and set
    47	   `PAIR_HOME` there before the existing launch handoff.
    48	2. **Dispatcher consolidation:** move helper binaries behind `pair <subcommand>`
    49	   routes and leave old command names as generated compatibility shims only
    50	   where native callers still need them.
    51	3. **Go-owned orchestration:** port stateful shell orchestrators into Go in
    52	   dependency order: launch/session lifecycle, scrollback and changelog openers,
    53	   title poller, review helpers, clipboard helpers, then small quit/restart/help
    54	   shims.
    55	4. **Native single binary:** once shell ownership is gone, stop extracting shell
    56	   scripts. Keep only the native assets that external runtimes require, such as
    57	   Neovim Lua and Zellij KDL, either embedded-and-extracted or generated at
    58	   startup.
    59	
    60	`ARCH-PURPOSE`: #90 is only complete if a copied binary can supply all Pair-owned
    61	runtime assets without falling back to a source checkout. `ARCH-DRY`: use one
    62	runtime manifest for embedding, extraction, install verification, and package
    63	metadata. `ARCH-PURE`: keep manifest planning and runtime selection testable as
    64	pure functions, with filesystem writes confined to thin seams.
    65	

exec
/bin/zsh -lc 'rg -n "PAIR_DATA_DIR|runtimeDataDir|EmbeddedAssetRoot|withEnv|runtimebundle|BuildManifest|Revisions|ARCH-" cmd tests README.md atlas workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
README.md:196:                                 # $PAIR_DATA_DIR from <old> to <new>;
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:68:- `ARCH-PURPOSE`: the copied binary must be enough to provide Pair-owned
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:70:- `ARCH-DRY`: the embedded runtime manifest must be the single source of what is
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:73:- `ARCH-PURE`: keep manifest planning, runtime selection, and extraction
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:131:`ARCH-DRY`, `ARCH-PURE`, and `ARCH-PURPOSE`; the durable plan keeps the runtime
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:146:to include `pair-title.sh`, define `$PAIR_DATA_DIR/runtime/<digest>/pair-home`
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:151:raw `go test ./cmd/internal/runtimebundle` would fail from a clean checkout once
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:153:add `make test-runtimebundle` as the generated-assets-before-test path after
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:161:`$PAIR_DATA_DIR/runtime/<digest>/pair-home` only after `PAIR_HOME`, executable
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:168:`runtimebundle-generate` rewrote the shared output tree in place; added a
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:171:published bundle (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`).
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:181:- `make test-runtimebundle`
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:182:- `make runtimebundle-drift-check`
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:192:blocking findings by honoring `PAIR_DATA_DIR` before XDG/home fallback for
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:194:updating the plan's `BuildManifest` path with a `## Revisions` entry, adding
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:196:manifest digests with matching markers, and reusing `runtimebundle` manifest
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:197:types from the generator. Also strengthened `runtimebundle-drift-check` to
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:205:- `go test ./cmd/internal/runtimebundle ./cmd/internal/runtimebundlegen -count=1`
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:207:- `make runtimebundle-drift-check`
workshop/issues/000090-self-contained-pair-binary-with-embedded-runtime-assets.md:209:- `make test-runtimebundle`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:7:**Architecture:** Add a generated runtime manifest as the single source for packaged assets (`ARCH-DRY`), a pure `runtimebundle` package for manifest hashing, version selection, extraction planning, and cleanup planning (`ARCH-PURE`), and a thin `cmd/pair-go` integration that only falls back to embedded extraction when `PAIR_HOME`, executable siblings, and `defaultPairHome` do not provide `bin/pair-shell`. This satisfies the issue purpose by proving copied-binary launch without relying on a source checkout (`ARCH-PURPOSE`).
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:19:| `RuntimeAsset` | `cmd/internal/runtimebundle/manifest.go` | new |
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:20:| `RuntimeManifest` | `cmd/internal/runtimebundle/manifest.go` | new |
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:21:| `ExtractionPlan` | `cmd/internal/runtimebundle/plan.go` | new |
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:23:| `CleanupPlan` | `cmd/internal/runtimebundle/cleanup.go` | new |
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:54:| `RuntimeFS` | `cmd/internal/runtimebundle/embed.go` | new | `embed.FS` |
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:55:| `RuntimeStore` | `cmd/internal/runtimebundle/store.go` | new | filesystem writes, chmod, rename |
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:57:| `BuildManifest` | `cmd/internal/runtimebundlegen/generate.go`, `cmd/internal/runtimebundle/generatecmd/main.go`, `cmd/internal/runtimebundle/assets/**`, `Makefile.local` | new | repo file tree at build time |
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:72:- **BuildManifest** — Makefile/generator path that keeps embedded assets synchronized with repo runtime files.
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:79:`cmd/internal/runtimebundle/assets/`; Go's `embed` package reads that tree at
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:80:compile time. The generator is the single source of the file list (`ARCH-DRY`):
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:109:- `make runtimebundle-generate` removes and rebuilds the gitignored asset tree
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:111:- `make runtimebundle-drift-check` regenerates twice into temp dirs and compares
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:113:- `make test-runtimebundle` depends on helper binaries plus
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:114:  `runtimebundle-generate`, then runs `go test ./cmd/internal/runtimebundle
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:116:  clean checkout; raw `go test ./cmd/internal/runtimebundle` is only valid after
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:118:- `make build` depends on `runtimebundle-generate` before building `bin/pair` and
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:124:- The extracted root is `$PAIR_DATA_DIR/runtime/<manifest-digest>/pair-home`,
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:126:  cleanup treats as one runtime generation is `$PAIR_DATA_DIR/runtime/<digest>/`.
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:127:- Extraction writes `$PAIR_DATA_DIR/runtime/<digest>/manifest.json` after all
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:130:- Cleanup only considers direct children of `$PAIR_DATA_DIR/runtime/` whose names
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:148:- Create: `cmd/internal/runtimebundle/manifest.go`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:149:- Create: `cmd/internal/runtimebundle/manifest_test.go`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:167:Run: `go test ./cmd/internal/runtimebundle -run 'TestManifest' -count=1`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:177:Run: `go test ./cmd/internal/runtimebundle -run 'TestManifest' -count=1`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:184:- Create: `cmd/internal/runtimebundle/plan.go`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:185:- Create: `cmd/internal/runtimebundle/plan_test.go`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:186:- Create: `cmd/internal/runtimebundle/cleanup.go`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:187:- Create: `cmd/internal/runtimebundle/cleanup_test.go`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:202:Cover keeping the selected `$PAIR_DATA_DIR/runtime/<digest>/` generation, retaining the newest configured count, and deleting only directories that match the runtime root contract above.
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:206:Run: `go test ./cmd/internal/runtimebundle -run 'TestPlan|TestCleanup' -count=1`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:216:Run: `go test ./cmd/internal/runtimebundle -count=1`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:225:- Create: `cmd/internal/runtimebundle/generate.go`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:226:- Create or generate: `cmd/internal/runtimebundle/embed.go`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:227:- Generate, gitignored: `cmd/internal/runtimebundle/assets/**`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:230:- Test: `cmd/internal/runtimebundle/embed_test.go`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:238:Run: `go test ./cmd/internal/runtimebundle -run TestEmbeddedManifestContainsLaunchAssets -count=1`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:246:Implement `make runtimebundle-generate` to copy the required inputs into `cmd/internal/runtimebundle/assets/runtime/`, preserving executable bits and writing a manifest JSON with sorted paths. Add `//go:embed assets/runtime/**` and expose the embedded `RuntimeManifest` plus file reader through `RuntimeFS`.
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:250:Update `PAIR_GO_SRCS` so changes to runtimebundle sources rebuild `bin/pair` and `bin/pair-go`. Add `test-runtimebundle` and `runtimebundle-drift-check`; include both in the relevant test path. Ensure `make build` builds helper binaries first, regenerates the asset tree, then builds `pair`/`pair-go` so the single binary embeds current helpers without committing generated binaries.
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:254:Run: `make test-runtimebundle`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:261:- Create: `cmd/internal/runtimebundle/store.go`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:262:- Create: `cmd/internal/runtimebundle/store_test.go`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:270:Run: `go test ./cmd/internal/runtimebundle -run 'TestStore|TestExtract' -count=1`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:280:Run: `make test-runtimebundle`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:331:Build `bin/pair`, copy only that binary to a temp `PATH`, set `PAIR_DATA_DIR` to a temp dir, and run a harmless launch path such as `--help` through `pair-shell`. Use fake external commands only where needed so the test does not start a real zellij session.
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:356:Add `test-pair-embedded-runtime` and `runtimebundle-drift-check` to `.PHONY`; add the smoke target to the main `test` dependency list after `test-pair-go-install-layout`.
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:398:Run: `make test-runtimebundle`
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:436:## Revisions
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:444:Delta: updated `BuildManifest` to point at
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:445:`cmd/internal/runtimebundlegen/generate.go` and
workshop/plans/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md:446:`cmd/internal/runtimebundle/generatecmd/main.go`, leaving the generated
atlas/how-to-bring-up-a-new-harness-cli.md:60:- Since TUI agents do not always expose session IDs on stdout, `pair-session-watch` runs in the background. It finds the agent process PID from `$PAIR_DATA_DIR/agent-pid-<tag>` (written by `pair-wrap`), walks its descendants, and inspects files held open by the processes via `lsof -p <pid>`.
atlas/how-to-bring-up-a-new-harness-cli.md:155:one JSON line per trigger to `$PAIR_DATA_DIR/adapt-<tag>.jsonl` during normal use.
atlas/how-to-bring-up-a-new-harness-cli.md:196:prompt). It is capped at 200 bytes and the file stays local under `$PAIR_DATA_DIR`,
tests/pair-rename.sh:4:# Builds a fixtured $PAIR_DATA_DIR with representative tag-scoped files,
tests/pair-rename.sh:68:# Run `pair rename` in an isolated $PAIR_DATA_DIR with the zellij shim.
tests/pair-rename.sh:75:    PAIR_DATA_DIR="$dd" PATH="$shim:$PATH" "$PAIR_BIN" rename "$@"
atlas/review-workbench.md:158:  `$PAIR_DATA_DIR/review-<tag>.mode`, defaulting to Edit) and triggers a redraw
tests/changelog-notify-test.sh:27:local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
tests/changelog-notify-test.sh:59:  local dd = os.getenv('PAIR_DATA_DIR')
tests/changelog-notify-test.sh:94:  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
cmd/pair-wrap/main.go:1888:	// to send SIGUSR1; only enabled when PAIR_TAG/PAIR_DATA_DIR resolved
atlas/go-migration-inventory.md:31:assets and extracts them to `$PAIR_DATA_DIR/runtime/<digest>/pair-home` on
atlas/go-migration-inventory.md:60:`ARCH-PURPOSE`: #90 is only complete if a copied binary can supply all Pair-owned
atlas/go-migration-inventory.md:61:runtime assets without falling back to a source checkout. `ARCH-DRY`: use one
atlas/go-migration-inventory.md:63:metadata. `ARCH-PURE`: keep manifest planning and runtime selection testable as
atlas/go-migration-inventory.md:90:| `bin/pair` / `bin/pair-shell` / `cmd/internal/launcher` / `cmd/internal/entrypoint` / `cmd/internal/runtimebundle` | Go public entrypoint plus retained shell launcher and embedded runtime fallback | user shell, copied-binary installs, `bin/pair-dev`, restart re-exec, tests, `pair-go launch` | `bin/pair` is generated from `cmd/pair-go` and resolves `PAIR_HOME` / sibling root / build-time `defaultPairHome`; if none exists, it extracts the embedded runtime to `$PAIR_DATA_DIR/runtime/<digest>/pair-home`; then it execs `<asset-root>/bin/pair-shell` with `pair`-compatible argv/env and `PAIR_HOME` pointed at the selected root. `bin/pair-shell` parses `pair [agent]`, `pair resume`, `pair continue`, `pair list`, `pair rename`, `--` agent args; starts/attaches zellij; exits nonzero on invalid create flow; long-running parent of zellij. `pair-go launch ...` shares the same compatibility handoff. | `bin/pair-shell` exports `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_AGENT_ARGS`; reads/writes many tag files under data dir; uses zellij, fzf, jq, nvim, make via dev hook. `cmd/internal/entrypoint` resolves invocation mode, asset root, and compatibility request; `cmd/internal/runtimebundle` owns manifest hashing, extraction planning, runtime markers, and stale-runtime cleanup; `cmd/internal/launcher` keeps the fakeable pure decision core from #75 for later native launch work. | Public entrypoint is Go-owned as of #79; copied-binary embedded fallback exists as of #90; `bin/pair-shell` is retained because real zellij lifecycle, prompt UI, restart/quit cleanup, cmux ownership, dev rebuild, continuation, rename, config/session migration, and title poller remain shell-owned | P0 |
atlas/go-migration-inventory.md:96:| `bin/pair-slug` / `cmd/pair-slug` | Go binary | `pair-wrap` turn-end hook, tests | Env-driven, no stdin; resolves native transcript, proposes slug; exits 0 on most failures. | Requires `PAIR_TAG`, `PAIR_DATA_DIR`; reads config/transcripts/git branch; writes `slug-proposed-<tag>`; optional `PAIR_SLUG_*`, `OPENAI_API_KEY`. | go-subcommand `pair slug`; legacy binary retained during #76 | P1 |
atlas/go-migration-inventory.md:97:| `bin/pair-context` / `cmd/pair-context` / `cmd/internal/contextcmd` | Go binary plus shared runner | `bin/pair-title.sh`; development-only `pair-go context` | `pair-context <tag> <agent>` and `pair-go context <tag> <agent>` print the same humanized token count or nothing; tolerant exit 0 on failure. | Reads `PAIR_DATA_DIR`, `pane-<tag>-<agent>.json`, config, native transcripts. | implemented helper route in `pair-go context`; legacy binary retained while title poller calls it | P1 |
atlas/go-migration-inventory.md:102:| `cmd/internal/adapt` | Go helper package | `pair-wrap`, `pair-slug`, tests | Pure-ish emitter helpers plus file open seam; no command. | Writes `$PAIR_DATA_DIR/adapt-<tag>.jsonl`; schema shared with shell/Lua. | internal package, reuse behind dispatcher | P1 |
atlas/go-migration-inventory.md:106:| `bin/pair-scrollback-open` | POSIX shell orchestrator | zellij Alt+/ Run, nvim Alt+b jump | `pair-scrollback-open [--jump prev|next]`; opens read-only nvim viewer; singleton lock. | Requires `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_AGENT`, `PAIR_HOME`; calls renderer, zellij IPC, nvim; writes `.ansi`, `.viewport`, lock. | shell-glue now; candidate Go orchestration after entrypoint, while `nvim/scrollback.lua` remains native | P1 |
atlas/go-migration-inventory.md:112:| `bin/lib/adapt-log.sh` | sourced shell helper | remaining shell emitters | `adapt_log comp agent aspect signal outcome [detail]`; no-op if no `PAIR_TAG` or jq. | Appends JSONL to `$PAIR_DATA_DIR/adapt-<tag>.jsonl`. | keep until remaining shell emitters move; schema stays DRY with Go/Lua emitters | P1 |
atlas/go-migration-inventory.md:120:| `bin/clipboard-to-pane.sh` | Bash copy/paste helper | `copy-on-select.sh`, direct zellij run possible | Reads OS clipboard, stages quote, focuses nvim, triggers Lua paste. | Uses pbpaste/wl-paste/xclip, jq, zellij, `PAIR_DATA_DIR`, `PAIR_TAG`, `PAIR_HOME`; writes quote and debug log. | shell-glue; keep until zellij copy flow has Go owner | P2 |
atlas/go-migration-inventory.md:125:| `bin/pair-review-target` | Bash review helper | review readiness/open/tests | Emits JSON target metadata under data dir. | Requires `PAIR_DATA_DIR`; reads config/pid files/lsof; writes `review-target-<tag>.json`. | shell-glue; possible #78 candidate if review packaging matters | P2 |
atlas/go-migration-inventory.md:157:- `make runtimebundle-generate` refreshes the gitignored embedded runtime asset
atlas/go-migration-inventory.md:158:  tree and manifest; `make runtimebundle-drift-check` verifies the generated
atlas/go-migration-inventory.md:170:- `make test-runtimebundle` runs bundle-generation-aware Go tests, and
tests/pair-session-watch-test.sh:33:PAIR_DATA_DIR="$RT/data" \
atlas/architecture.md:55:an extracted embedded runtime under `$PAIR_DATA_DIR/runtime/<digest>/pair-home`
atlas/architecture.md:81:`pair-scrollback-render` binaries (`ARCH-DRY`), so behavior stays aligned while
atlas/architecture.md:103:`bin/pair-shell` resolves `$PAIR_HOME` from its own real path (portable bash, no `readlink -f`), prepends `$PAIR_HOME/bin` to `$PATH` (idempotent across re-launches) so all helper scripts resolve by bare name in zellij configs and keybinds, parses argv — first positional is `$PAIR_AGENT` (default `claude`), everything after `--` is joined into `$PAIR_AGENT_ARGS`, extra positionals before `--` are an error with a usage hint, defaults `$PAIR_TAG` to the cwd basename (the create-flow prompt or `pair resume <tag>` overrides it), resolves `$PAIR_DATA_DIR` to `${XDG_DATA_HOME:-$HOME/.local/share}/pair`, runs a one-time migration of any old `~/scratch/pair-{draft,log}-*` files, and dispatches:
atlas/architecture.md:118:**Tag reuse & stale-EXITED residue (#67).** A pair tag *is* a workspace, and a live zellij session is globally `pair-<tag>` regardless of agent. `Alt+x` can leave a resurrect record like `pair-2 (EXITED - attach to resurrect)`; that row still shows in `list-sessions`, so the old open-coded `grep -qx` collision checks treated a fully-quit tag as occupied and blocked reuse. The single helper `session_blocks_reuse <session>` centralizes the decision (`ARCH-DRY`): an `EXITED` row is stale full-quit residue — it deletes the zellij record (`delete-session --force`) and reports the tag reusable; a running/detached row still blocks; an absent session never blocks. It is the shared gate for forced `pair resume <tag>`, the free-slot scan, and the create-flow name-prompt collision. `pair rename` keeps its own offline-only resurrectable-session contract and is *not* routed through it. (As of #69 the cmux-owner path no longer consults it — see the cmux workspace-title note below.)
atlas/architecture.md:122:- **Per-pane context meter in the zellij FRAME (#71).** Each agent pane's frame title reads `<agent> (<count>) [<cwd>]`, where `<count>` is the agent's current context-window occupancy — an absolute humanized token count (`970k`), so no model→window catalog is needed. Source of truth is the agent's own session transcript: the pure `cmd/internal/ctxmeter` reader (`ContextTokens` sums the last *real* claude `message.usage`, skipping `isSidechain`/`<synthetic>` records; codex `last_token_usage.input_tokens` of the last `token_count` event; agy none) + `Humanize`, over the path from the shared `cmd/internal/transcript` resolver (extracted from `pair-slug`, ARCH-DRY). The one-shot `cmd/pair-context <tag> <agent>` wires it (tolerant: any failure prints nothing). Each pane records `{pane_id, cwd, cwd_display}` to a single-writer `pane-<tag>-<agent>.json` at startup (`main.kdl`, beside the startup rename — dodges the 3-writer race on `config-*`); the poller loops those, calls `pair-context`, and `zellij --session pair-<tag> action rename-pane`s each frame, gated on recent activity with a per-pane unchanged-skip cache. Always-on (the frame exists with or without cmux). Carried through `pair rename` like `config-*`.
atlas/architecture.md:126:**Saved-config resolution & legacy Codex migration (#67).** `resolve_config_file <tag> <agent>` resolves the canonical `config-<tag>-<agent>.json`. Older Codex sessions on disk use a doubled shape `config-<tag>-codex-codex.json`; when the canonical file is absent and the agent is `codex`, the helper migrates the legacy file to the canonical name *iff* its JSON declares `"agent":"codex"` — a narrow, agent-checked compatibility path, **not** a glob resolver, so unrelated stale files can't silently win (`ARCH-DRY`, `ARCH-PURE`). It is used only where both tag and agent are known (restart-marker read, cleanup resume hint, the tag-restart picker that surfaces native Codex resume, and the two config writes); the agent-inference glob loop is deliberately left alone, since it is *discovering* the agent and already sees the legacy filename.
atlas/architecture.md:185:Read OS clipboard (`pbpaste` / `wl-paste` / `xclip`). Stage the raw body to `$PAIR_DATA_DIR/quote-<tag>`. All formatting decisions (par reflow, `> ` prefix) live in nvim now, conditional on cursor position — the shell is just a transport.
atlas/architecture.md:278:The `time` events (one generic `logScrollbackEvent` writer, ARCH-DRY; pure `dueForTimeEvent` debounce + a `p.now` clock seam, ARCH-PURE) let the change-log render date entries by real change-time — the raw byte stream stays byte-faithful (the scrollback render replays it), since the timestamp lives in the sidecar, not injected into the TTY (#59).
atlas/architecture.md:282:**Replay (`bin/pair-scrollback-render`, Go).** Reads `<raw>` and `<events.jsonl>`, feeds the bytes to a `charmbracelet/x/vt` emulator in a single offset-ordered walk over all events (`feedSegments`): write up to each offset, then `Resize` on a resize event or snapshot `Scrollback().Len()` on a `time` event (#59). The emulator runs the same VT100 interpretation zellij does live (width-based wrap, alternate-screen flips, scroll regions), so its row count matches what the user saw in zellij's indicator. After feeding, the renderer walks the scrolled-out history followed by the visible buffer, and emits one ANSI-decorated line per row to `<out.ansi>`: full-reset SGR + per-row attrs + the row's characters + `\x1b[0m`. With `--with-timestamps` (the change-log path only — never the Alt+/ viewer) the pure `interleaveDateMarkers` then inserts `⟦pair:ts DATE⟧` lines at each day boundary from the time snapshots (#59). Built into `bin/pair-scrollback-render` via `make pair-scrollback-render`; single static binary, no runtime dep. Its raw inputs live in `$PAIR_DATA_DIR` as `scrollback-<tag>-<agent>.{raw,events.jsonl}` (RAW VT bytes, NOT in the repo); `:PairTTYRawPath` / `_G.PairTTYRawPath()` (nvim, #56) prints the current session's live `.raw` path on demand and copies it to the `+` register — useful for grabbing the byte stream mid-session, since an Alt+x quit deletes it unless preserved.
atlas/architecture.md:286:**Continuation writer (`cmd/pair-continuation`, `continuation` datatype).** A *continuation* is the human-understanding cousin of a native `pair resume`: `resume` restores machine state (the agent's own session id, byte-faithful); a continuation distills the *rendered* session (the plain projection above) into a durable, portable markdown doc — `workshop/continuation/<YYYYMMDDTHHMMSS>-slug.md`, the `continuation` datatype defined in `construct/datatype/continuation.md` (ariadne#91) — so work resumes across time / machines / people / agent stacks. The `xx-datatype` dispatcher does the distillation (judgment); `cmd/pair-continuation` does the *mechanics* deterministically: render conformant frontmatter, allocate a collision-safe timestamped name, write, then `git add` + a **path-scoped** `commit -- <file>` (so a dirty index isn't swept in) + `push origin HEAD` — on the current branch, which lands on main when that branch merges (disaster-recovery — an unpushed recovery doc can't save state; a push failure is non-fatal so a detached park still keeps the local commit). Pure core (frontmatter / name allocation / assemble / validate) is IO-free; a thin clock/fs/git seam (the `git -C` shell-out pattern from `cmd/pair-slug`, no git library) is exercised against a real temp repo with a bare origin. `pair continue [slug] [agent]` resumes *from* a continuation: bare lists them; `<slug>` seeds a fresh session (via `draft-<tag>.md`, create-path only) to read the doc and do its NEXT ACTION; `[agent]` ports to a different stack. Unlike `resume` it does **not** force the tag — it flows through the normal name prompt so the operator picks it (a long slug must never become an over-long zellij `--session` name: zellij caps it at the sun_path socket-budget and rejects overflow with a cryptic "must be less than 0 characters" clap error, so a launch-time guard probes zellij's own validator via a no-op `action list-clients` and fails with a clear message instead — #54) — and forwards `-- <args>` to the agent like a plain `pair <agent> -- <args>`. It never reads `session_id` — that's `resume`'s job. On Alt+x, `cleanup_quit_marker` offers to **park** the session: preserve its scrollback (`.raw` + `.events.jsonl`) under a non-recyclable `parked-scrollback-<tag>-<ts>.*` so a live session can distill it later (no live agent exists at quit, so the nudge only *preserves*). The prompt is timeout-bounded — it auto-defaults to **N** (preserve nothing) after `PAIR_PARK_PROMPT_TIMEOUT` seconds (default 5, integer seam) so an unattended quit never blocks on it (#64). The park mechanics live in a shared `park_scrollback <tag> <agent> [--copy]` helper (#55, extracted from `cleanup_quit_marker` — ARCH-DRY): the quit path *moves* (session dying), the compaction path *copies* (the live `pair-wrap --scrollback-log` is still appending to `.raw`).
atlas/architecture.md:294:**Shape (ARCH-PURE).** A **pure core** — `find_markers_in_line` (byte-walk parser), the `>`/`]` escape-unescape (`esc_x`/`esc_y`/`unescape`, backslash-parity walk so a selection containing the delimiters survives), `strip_markers`, `marker_key`, `collect_markers_by_line` (the load-time baseline so only *newly-added* markers extract), `format_extraction`, `new_marker_count`, width helpers — is exposed on the module table and unit-tested directly in `nvim/annotate_test.lua` (no buffer, no IO, no mocks). A **thin IO/UI seam** — `M.attach{bufnr, pending_path, footer, source_label, quit_noun}` — wires the `Alt+q` keymaps, the floating prompt (`open_marker_prompt`), the read-only unlock→insert→relock rewrite, the `VimLeavePre`→sidecar `M.emit`, and the `M.confirm_quit` gate (confirms only when there are user-added markers / a footer comment to ship).
atlas/architecture.md:302:**Open (`bin/pair-scrollback-open`, POSIX sh).** Validates `PAIR_DATA_DIR` / `PAIR_TAG` / `PAIR_AGENT`, runs the renderer, then *launches* `nvim -u $PAIR_HOME/nvim/scrollback.lua $ANSI` as a child — deliberately **not** `exec`, so the script stays alive as nvim's parent and an `EXIT`/`INT`/`TERM` trap can clear the re-entrancy lock on quit. Errors print and `sleep` briefly so the message is readable before the floating pane self-closes. Bound in `zellij/config.kdl` to `Alt+/` as a 100% × 100% floating pane with `close_on_exit=true` — the user's `:q` in the viewer dismisses the pane and returns to pair's two-pane layout untouched. **Re-entrancy guard:** `Alt+/` is a global zellij bind, so pressing it again while the viewer is already focused fires another `Run` and would stack a second nvim (one `:q`/Esc per layer to unwind). zellij can't conditionally skip a `Run`, so the script self-guards: before launching nvim it writes its own PID to `$PAIR_DATA_DIR/scrollback-<tag>-<agent>.openlock`, and on entry it exits immediately if that lock already holds a *live* PID — the redundant floating pane then self-dismisses via `close_on_exit` and focus falls back to the open viewer. A stale lock (hard kill) carries a dead PID and is reclaimed by the next open's `kill -0` check. The draft pane's `Alt+b` (`--jump prev`) runs the same launcher, so it's covered too.
atlas/architecture.md:306:**Comment markers — `Alt+q` in viewer → draft pickup (#000018).** While reading scrollback, `Alt+q` drops a parley-style `🤖[]` marker at the cursor (or `🤖<selection>[…]` in visual mode). The buffer is read-only, so the keymap lifts `modifiable`/`readonly` for the insert and re-locks immediately. (#57: this whole marker subsystem was extracted to the shared `nvim/annotate.lua` — the change-log viewer uses the identical flow; see "Shared 🤖-marker annotation" above.) On viewer exit (`VimLeavePre`), `nvim/annotate.lua` (`M.emit`) walks every line, parses each `🤖<X>?[Y]` marker by literal-byte scan (Lua patterns aren't UTF-8 aware), and writes a formatted block to `$PAIR_DATA_DIR/scrollback-pending-<tag>.md`:
atlas/architecture.md:330:- `PairPasteQuote()` (global, called from `bin/clipboard-to-pane.sh` via `:lua PairPasteQuote()`): reads the raw selection from `$PAIR_DATA_DIR/quote-<tag>` and dispatches on cursor column.
atlas/architecture.md:537:- **State** (`$PAIR_DATA_DIR`, per `(tag, agent, session)` — the base is
atlas/architecture.md:588:   - **PID-bound (preferred).** Reads `$PAIR_DATA_DIR/agent-pid-<tag>` (written by pair-wrap right after `pty.Start`) only when the pidfile's mtime is at-or-after the watcher's start, so a stale pidfile from a prior launch is ignored until pair-wrap overwrites it. Then it inspects open files in that PID's process tree via `lsof -p <pid> -Fn`. Race-free across concurrent pair sessions because lsof output is scoped to specific PIDs. Falls back internally to a birth-time-filtered directory walk if the agent doesn't keep its session file open: candidates are files with `stat -f %B >= agent_start_epoch`, and only a *single* candidate is accepted (multiple = concurrent race, refuse rather than guess).
atlas/architecture.md:603:**Stored shape.** `$PAIR_DATA_DIR/config-<tag>-<agent>.json`:
atlas/architecture.md:640:A tag is durable but historically frozen-at-create. `pair rename <old> <new>` lifts that: every tag-scoped file in `$PAIR_DATA_DIR` is renamed in one transactional pass, so the agent's saved session, draft buffer, scrollback artefacts, log, queue, and per-pane pidfiles all follow the new name. Renaming is offline-only — zellij has no live-rename for a session, so the inside-session UX wraps quit → rename → re-exec around this primitive: Ctrl+Alt+n's confirm offers `&Yes / &No / &Rename`, and the (R) path prompts for a new tag, pre-validates via `pair rename --restart-check`, then triggers the restart with `--rename-to <new>`. Orthogonal to Shift+Alt+N's `--new-session` — rename + fresh agent is one gesture.
atlas/architecture.md:649:**Atomicity.** The full `(src, dst)` plan is written to `$PAIR_DATA_DIR/.rename-<old>-to-<new>.journal` before any `mv` runs. On mid-flight failure, the renamer reads the first N journal lines, swaps columns, and `mv`s the completed renames back to their original paths. The journal is cleared on success and retained on rollback failure as a forensic breadcrumb (M3 will add crash-recovery: a stale journal on startup gets finished or rolled back automatically).
atlas/architecture.md:665:The launcher exports `$PAIR_DATA_DIR` so `nvim/init.lua` can compute the same path without re-deriving the XDG fallback chain.
tests/adapt-schema-test.sh:26:export PAIR_TAG=golden PAIR_DATA_DIR="$RT" PAIR_AGENT=codex
tests/review-readiness-cli-test.sh:38:prep_out="$(PAIR_HOME="$ROOT" PAIR_DATA_DIR="$RT" PAIR_TAG=prep PAIR_SESSION_ID=sid \
tests/review-toggle-test.sh:64:local sf = vim.env.PAIR_DATA_DIR .. '/review-' .. vim.env.PAIR_TAG .. '.open'
tests/review-toggle-test.sh:89:local target = vim.env.PAIR_DATA_DIR .. '/review-target-' .. vim.env.PAIR_TAG .. '.json'
tests/review-toggle-test.sh:90:local draft = vim.env.PAIR_DATA_DIR .. '/draft.md' -- exists (the test wrote it)
tests/review-toggle-test.sh:97:local prepbin = vim.env.PAIR_DATA_DIR .. '/prep-ok'
tests/review-toggle-test.sh:125:os.remove(vim.env.PAIR_DATA_DIR .. '/config-' .. vim.env.PAIR_TAG .. '-claude.json')
tests/review-toggle-test.sh:138:  vim.env.PAIR_DATA_DIR .. '/config-' .. vim.env.PAIR_TAG .. '-' .. vim.env.PAIR_AGENT .. '.json')
tests/review-toggle-test.sh:149:os.remove(vim.env.PAIR_DATA_DIR .. '/config-' .. vim.env.PAIR_TAG .. '-codex.json')
tests/review-toggle-test.sh:150:vim.fn.writefile({ '111' }, vim.env.PAIR_DATA_DIR .. '/agent-pid-' .. vim.env.PAIR_TAG)
tests/review-toggle-test.sh:203:    PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" PAIR_SESSION_ID=testsid \
tests/cr-newline-test.sh:36:local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
tests/cr-newline-test.sh:76:  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/queue-send-test.sh:46:    env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/queue-send-test.sh:55:local dd = os.getenv('PAIR_DATA_DIR')
tests/review-window-test.sh:43:( cd "$RT" && PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" RESULT="$RT/r1" \
tests/review-window-test.sh:48:if PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-open" "$RT/nope.md" 2>/dev/null; then
tests/review-window-test.sh:54:PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_HOME="$ROOT" "$ROOT/bin/pair-review-open" "$RT/doc.md" || true
tests/review-window-test.sh:81:( cd "$REPO" && PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/review-window-test.sh:323:( cd "$REPO" && PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/pair-embedded-runtime-test.sh:16:make -C "$repo_root" runtimebundle-generate >/dev/null
tests/pair-embedded-runtime-test.sh:78:export PAIR_DATA_DIR="$pair_data"
tests/pair-title-poller-test.sh:117:    PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$DD" RENAME_LOG="$RT/rename.log" FAKE_COUNT="$count" \
tests/statusline-pos-test.sh:33:local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
tests/statusline-pos-test.sh:68:  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/changelog-session-key-test.sh:11:export PAIR_DATA_DIR="$tmp/data" PAIR_TAG=t PAIR_AGENT=claude
tests/changelog-session-key-test.sh:12:mkdir -p "$PAIR_DATA_DIR"
tests/changelog-session-key-test.sh:37:printf 'old log\n' > "$PAIR_DATA_DIR/changelog-t-claude-$A.md"
tests/changelog-session-key-test.sh:41:[ -s "$PAIR_DATA_DIR/changelog-t-claude-$B.md" ] \
tests/changelog-session-key-test.sh:46:grep -q 'old log' "$PAIR_DATA_DIR/changelog-t-claude-$A.md" \
tests/changelog-session-key-test.sh:52:  > "$PAIR_DATA_DIR/config-t-claude.json"
tests/changelog-session-key-test.sh:58:rm -f "$PAIR_DATA_DIR/config-t-claude.json"
tests/zellij-trace-test.sh:43:PAIR_DATA_DIR="$RT" \
tests/review-poke-test.sh:49:PATH="$RT/bin:$PATH" PAIR_ROOT="$ROOT" RESULT="$RESULT" PAIR_DATA_DIR="$RT" PAIR_TAG=poke \
tests/review-indicator-test.sh:54:PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude PAIR_HOME="$ROOT" \
tests/pair-review-target-test.sh:14:PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=codex \
tests/pair-review-target-test.sh:20:PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=codex \
tests/autopair-test.sh:31:local O = assert(io.open(os.getenv('PAIR_DATA_DIR') .. '/result.txt', 'w'))
tests/autopair-test.sh:69:  env PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=claude \
tests/changelog-open-test.sh:23:export PAIR_DATA_DIR="$tmp/data"
tests/changelog-open-test.sh:26:mkdir -p "$PAIR_DATA_DIR"
tests/changelog-open-test.sh:30:    > "$PAIR_DATA_DIR/scrollback-t-claude.raw"
tests/changelog-open-test.sh:32:    > "$PAIR_DATA_DIR/scrollback-t-claude.events.jsonl"
tests/changelog-open-test.sh:53:DLOCK="$PAIR_DATA_DIR/changelog-t-claude.distill.lock"
tests/changelog-open-test.sh:54:LOG="$PAIR_DATA_DIR/changelog-t-claude.md"
tests/changelog-open-test.sh:55:ANCHOR="$PAIR_DATA_DIR/changelog-t-claude.anchor"
tests/changelog-open-test.sh:70:[ -f "$PAIR_DATA_DIR/changelog-t-claude.openlock" ] && { echo "FAIL: openlock not cleared on viewer exit"; fail=1; }
cmd/pair-go/main_test.go:206:	got := runtimeDataDir("/pair-data", "/home/me", "/xdg")
cmd/pair-go/main_test.go:208:		t.Fatalf("runtimeDataDir = %q, want PAIR_DATA_DIR", got)
cmd/pair-go/main_test.go:213:	got := runtimeDataDir("", "/home/me", "/xdg")
cmd/pair-go/main_test.go:215:		t.Fatalf("runtimeDataDir = %q, want XDG pair dir", got)
cmd/pair-go/main_test.go:298:func (f *fakeLegacyRuntime) EmbeddedAssetRoot() (string, error) {
cmd/pair-session-watch/main_test.go:13:		"PAIR_DATA_DIR":                       "/tmp/pair-data",
cmd/pair-slug/main_test.go:37:// transcript, and an isolated PAIR_DATA_DIR. cwd is a non-git temp dir so
cmd/pair-slug/main_test.go:58:		"PAIR_DATA_DIR="+dataDir,
cmd/pair-slug/main_test.go:130:		"PAIR_DATA_DIR="+dataDir,
cmd/pair-go/main.go:16:	"github.com/xianxu/pair/cmd/internal/runtimebundle"
cmd/pair-go/main.go:35:	EmbeddedAssetRoot() (string, error)
cmd/pair-go/main.go:71:		embeddedRoot, embeddedErr := rt.EmbeddedAssetRoot()
cmd/pair-go/main.go:92:	return rt.Exec(label, req.Path, req.Argv, withEnv(rt.Environ(), "PAIR_HOME", root.Root))
cmd/pair-go/main.go:125:func (osLegacyRuntime) EmbeddedAssetRoot() (string, error) {
cmd/pair-go/main.go:126:	dataDir := runtimeDataDir(os.Getenv("PAIR_DATA_DIR"), os.Getenv("HOME"), os.Getenv("XDG_DATA_HOME"))
cmd/pair-go/main.go:127:	res, err := runtimebundle.Extract(runtimebundle.StoreInput{
cmd/pair-go/main.go:129:		Manifest:  runtimebundle.EmbeddedManifest(),
cmd/pair-go/main.go:130:		ReadAsset: runtimebundle.EmbeddedAsset,
cmd/pair-go/main.go:139:func runtimeDataDir(pairDataDir, home, xdgDataHome string) string {
cmd/pair-go/main.go:164:func withEnv(env []string, key, value string) []string {
cmd/pair-session-watch/main.go:45:	dataDir := getenv("PAIR_DATA_DIR")
cmd/pair-changelog/main.go:163:// fs-watches $PAIR_DATA_DIR for it and, on arrival, flashes its statusline then
cmd/internal/runtimebundlegen/generate_test.go:1:package runtimebundlegen
cmd/pair-slug/main.go:6:// from $PAIR_DATA_DIR/config-<tag>-<agent>.json (session_id) + the per-agent
cmd/pair-slug/main.go:10:// $PAIR_DATA_DIR/slug-proposed-<tag>. nvim applies it (see nvim/slug.lua).
cmd/pair-slug/main.go:14://	PAIR_TAG, PAIR_DATA_DIR   required; identify the session
cmd/pair-slug/main.go:164:	dataDir := os.Getenv("PAIR_DATA_DIR")
cmd/pair-slug/main.go:166:		logf("no PAIR_TAG/PAIR_DATA_DIR; not inside a pair session")
cmd/pair-context/main_test.go:37:	cmd.Env = append(os.Environ(), "HOME="+home, "PAIR_DATA_DIR="+data)
cmd/pair-context/main_test.go:51:	cmd.Env = append(os.Environ(), "HOME="+home, "PAIR_DATA_DIR="+filepath.Join(home, "empty"))
cmd/pair-go/helper_equivalence_test.go:20:	env := append(os.Environ(), "HOME="+home, "PAIR_DATA_DIR="+data)
cmd/internal/runtimebundlegen/generate.go:1:package runtimebundlegen
cmd/internal/runtimebundlegen/generate.go:16:	"github.com/xianxu/pair/cmd/internal/runtimebundle"
cmd/internal/runtimebundlegen/generate.go:59:func Generate(opts GenerateOptions) (runtimebundle.RuntimeManifest, error) {
cmd/internal/runtimebundlegen/generate.go:64:		return runtimebundle.RuntimeManifest{}, fmt.Errorf("output root is required")
cmd/internal/runtimebundlegen/generate.go:68:		return runtimebundle.RuntimeManifest{}, err
cmd/internal/runtimebundlegen/generate.go:72:		return runtimebundle.RuntimeManifest{}, err
cmd/internal/runtimebundlegen/generate.go:77:		return runtimebundle.RuntimeManifest{}, err
cmd/internal/runtimebundlegen/generate.go:81:		return runtimebundle.RuntimeManifest{}, err
cmd/internal/runtimebundlegen/generate.go:115:			return runtimebundle.RuntimeManifest{}, err
cmd/internal/runtimebundlegen/generate.go:128:	manifest := runtimebundle.RuntimeManifest{Assets: make([]runtimebundle.RuntimeAsset, 0, len(ordered))}
cmd/internal/runtimebundlegen/generate.go:133:			return runtimebundle.RuntimeManifest{}, fmt.Errorf("asset %s: %w", logical, err)
cmd/internal/runtimebundlegen/generate.go:140:			return runtimebundle.RuntimeManifest{}, err
cmd/internal/runtimebundlegen/generate.go:142:		manifest.Assets = append(manifest.Assets, runtimebundle.RuntimeAsset{
cmd/internal/runtimebundlegen/generate.go:151:		return runtimebundle.RuntimeManifest{}, err
cmd/internal/runtimebundlegen/generate.go:155:		return runtimebundle.RuntimeManifest{}, err
cmd/internal/runtimebundlegen/generate.go:159:		return runtimebundle.RuntimeManifest{}, err
cmd/internal/runtimebundlegen/generate.go:163:		return runtimebundle.RuntimeManifest{}, err
cmd/internal/runtimebundlegen/generate.go:166:		return runtimebundle.RuntimeManifest{}, err
cmd/internal/contextcmd/contextcmd.go:25:		PairDataDir: os.Getenv("PAIR_DATA_DIR"),
cmd/internal/dispatcher/dispatcher_test.go:101:	t.Setenv("PAIR_DATA_DIR", data)
cmd/internal/runtimebundle/store.go:1:package runtimebundle
cmd/internal/runtimebundle/assets/runtime/files/zellij/layouts/main.kdl:45:            args "-c" "_pdd=\"${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}\"; printf '{\"pane_id\":\"%s\",\"cwd\":\"%s\",\"cwd_display\":\"%s\"}\\n' \"$ZELLIJ_PANE_ID\" \"$PWD\" \"${PAIR_PANE_CWD:-$PWD}\" > \"$_pdd/pane-${PAIR_TAG:-${PAIR_AGENT:-claude}}-${PAIR_AGENT:-claude}.json\" 2>/dev/null; zellij action rename-pane --pane-id \"$ZELLIJ_PANE_ID\" \"${PAIR_PANE_TITLE:-agent}\" 2>/dev/null; exec pair-wrap --scrollback-log \"${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}/scrollback-${PAIR_TAG:-${PAIR_AGENT:-claude}}-${PAIR_AGENT:-claude}.raw\" ${PAIR_AGENT:-claude} ${PAIR_AGENT_ARGS:-}"
cmd/internal/runtimebundle/embed.go:1:package runtimebundle
cmd/internal/runtimebundle/manifest.go:1:package runtimebundle
cmd/internal/runtimebundle/manifest_test.go:1:package runtimebundle
cmd/internal/runtimebundle/store_test.go:1:package runtimebundle
cmd/internal/runtimebundle/cleanup_test.go:1:package runtimebundle
cmd/internal/adapt/adapt_test.go:141:	t.Setenv("PAIR_DATA_DIR", dir)
cmd/internal/runtimebundle/plan.go:1:package runtimebundle
cmd/internal/adapt/adapt.go:3:// $PAIR_DATA_DIR/adapt-<tag>.jsonl.
cmd/internal/adapt/adapt.go:78:// DataDir returns $PAIR_DATA_DIR or the XDG default. This is the canonical
cmd/internal/adapt/adapt.go:82:	if d := os.Getenv("PAIR_DATA_DIR"); d != "" {
cmd/internal/runtimebundle/cleanup.go:1:package runtimebundle
cmd/internal/runtimebundle/plan_test.go:1:package runtimebundle
cmd/internal/runtimebundle/embed_test.go:1:package runtimebundle
cmd/internal/runtimebundle/assets/runtime/files/doctor/SKILL.md:12:(`$PAIR_DATA_DIR/adapt-<tag>.jsonl`) captures one line per adaptation trigger,
cmd/internal/runtimebundle/assets/runtime/files/doctor/SKILL.md:60:- `detail` is capped at 200 bytes and stays local under `$PAIR_DATA_DIR`; it can
cmd/internal/runtimebundle/generatecmd/main.go:8:	"github.com/xianxu/pair/cmd/internal/runtimebundlegen"
cmd/internal/runtimebundle/generatecmd/main.go:15:	if _, err := runtimebundlegen.Generate(runtimebundlegen.GenerateOptions{RepoRoot: *repo, OutRoot: *out}); err != nil {
cmd/internal/runtimebundle/generatecmd/main.go:16:		fmt.Fprintf(os.Stderr, "runtimebundle-generate: %v\n", err)
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-title.sh:20:#   - nvim draft: $PAIR_DATA_DIR/draft-<tag>.md
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-title.sh:47:DATA_DIR="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
cmd/internal/runtimebundle/assets/runtime/files/doctor/README.md:7:firing. The **flight recorder** (`$PAIR_DATA_DIR/adapt-<tag>.jsonl`) captures one
cmd/internal/runtimebundle/assets/runtime/files/doctor/README.md:55:- `detail` is capped at 200 bytes and stays local under `$PAIR_DATA_DIR`; it can
cmd/internal/runtimebundle/assets/runtime/files/nvim/review.lua:392:  return seam.open_state(vim.env.PAIR_DATA_DIR, vim.env.PAIR_TAG)
cmd/internal/runtimebundle/assets/runtime/files/nvim/review.lua:399:  return seam.read_mode(vim.env.PAIR_DATA_DIR, vim.env.PAIR_TAG)
cmd/internal/runtimebundle/assets/runtime/files/nvim/review.lua:458:  seam.write_mode(vim.env.PAIR_DATA_DIR, vim.env.PAIR_TAG, m)
cmd/internal/runtimebundle/assets/runtime/files/doctor/doctor.sh:5:# $PAIR_DATA_DIR/adapt-<tag>.jsonl during normal use (see
cmd/internal/runtimebundle/assets/runtime/files/doctor/doctor.sh:13:#   No arg → $PAIR_DATA_DIR/adapt-$PAIR_TAG.jsonl, else newest adapt-*.jsonl.
cmd/internal/runtimebundle/assets/runtime/files/doctor/doctor.sh:21:DATA_DIR="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
cmd/internal/runtimebundle/assets/runtime/files/doctor/doctor.sh:48:    echo "  A log appears once you run a session: \$PAIR_DATA_DIR/adapt-<tag>.jsonl"
cmd/internal/runtimebundle/assets/runtime/files/nvim/adapt.lua:4:-- $PAIR_DATA_DIR/adapt-<tag>.jsonl, byte-identical in schema + field order to
cmd/internal/runtimebundle/assets/runtime/files/nvim/adapt.lua:39:  local dir = vim.env.PAIR_DATA_DIR
cmd/internal/runtimebundle/assets/runtime/files/nvim/zellij_trace.lua:12:  return vim.env.PAIR_DATA_DIR
cmd/internal/runtimebundle/assets/runtime/files/bin/copy-on-select.sh:15:#      $PAIR_DATA_DIR/quote-<tag> and triggers nvim's PairPasteQuote() to
cmd/internal/runtimebundle/assets/runtime/files/nvim/annotate.lua:5:-- marker implementation instead of duplicating ~400 lines (ARCH-DRY).
cmd/internal/runtimebundle/assets/runtime/files/nvim/annotate.lua:7:-- Shape (ARCH-PURE):
cmd/internal/runtimebundle/assets/runtime/files/nvim/annotate.lua:279:-- Expose the pure core for direct unit testing (ARCH-PURE boundary).
cmd/internal/runtimebundle/assets/runtime/files/nvim/annotate.lua:693:-- the resolver lives once here rather than copied per viewer (ARCH-DRY).
cmd/internal/runtimebundle/assets/runtime/files/nvim/annotate.lua:695:  local data_dir = vim.env.PAIR_DATA_DIR
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-changelog-open:18:#   $PAIR_DATA_DIR  $PAIR_TAG  $PAIR_AGENT  $PAIR_HOME
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-changelog-open:21:if [ -z "${PAIR_DATA_DIR:-}" ] || [ -z "${PAIR_TAG:-}" ] || [ -z "${PAIR_AGENT:-}" ]; then
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-changelog-open:22:    printf 'pair-changelog-open: missing PAIR_DATA_DIR / PAIR_TAG / PAIR_AGENT\n' >&2
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-changelog-open:35:    cfg="$PAIR_DATA_DIR/config-$PAIR_TAG-$PAIR_AGENT.json"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-changelog-open:38:base="$PAIR_DATA_DIR/changelog-$PAIR_TAG-$PAIR_AGENT${sid:+-$sid}"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-changelog-open:39:RAW="$PAIR_DATA_DIR/scrollback-$PAIR_TAG-$PAIR_AGENT.raw"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-changelog-open:40:EVENTS="$PAIR_DATA_DIR/scrollback-$PAIR_TAG-$PAIR_AGENT.events.jsonl"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-notify:60:data_dir="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-shell:62:# Shared by resume/continue/rename + the name prompt (#52 ARCH-DRY). It *returns*
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-shell:94:                                $PAIR_DATA_DIR from <old> to <new>.
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-shell:309:        # $PAIR_DATA_DIR from <old> to <new>. Offline-only: refuses if
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-shell:356:        # shared helper (#52 ARCH-DRY). Length stays a rename-specific guard below.
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-shell:369:        rename_dd="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-shell:687:export PAIR_DATA_DIR="$DATA_DIR"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-shell:693:# (ARCH-DRY). Non-interactive; echoes the parked base path on success, rc 1 if
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-shell:821:# the cmux-owner liveness check (ARCH-DRY). `pair rename` keeps its own
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-shell:844:# (ARCH-DRY, ARCH-PURE). Used only where both tag and agent are already known;
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-shell:1070:# DATA_DIR / PAIR_DATA_DIR are set above (hoisted for #55, just after
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-shell:1568:                    # .raw/.events.jsonl rename + parked-<tag> marker (#55 ARCH-DRY).
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:462:  return os.getenv('PAIR_DATA_DIR')
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:792:  -- seam module so this reader and the pane writer can't diverge (ARCH-DRY, I3).
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:862:    local data_dir = vim.env.PAIR_DATA_DIR
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:873:    return seam.open_state(vim.env.PAIR_DATA_DIR, vim.env.PAIR_TAG)
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:890:    local p = seam.target_path(vim.env.PAIR_DATA_DIR, vim.env.PAIR_TAG)
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:906:    local p = seam.target_path(vim.env.PAIR_DATA_DIR, vim.env.PAIR_TAG)
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:1367:-- $PAIR_DATA_DIR/quote-<tag>; we decide the formatting here based on where
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:1389:  local data_dir = os.getenv('PAIR_DATA_DIR')
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:1694:--   2. $PAIR_DATA_DIR/agent-output-<tag> — colored spans extracted from
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:1718:-- ids — inspect $PAIR_DATA_DIR/agent-output-<tag> to see what's emitted).
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:1741:  -- Inspect $PAIR_DATA_DIR/agent-output-<tag> to update.
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:2849:  local data_dir = vim.env.PAIR_DATA_DIR
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:2856:  local data_dir = vim.env.PAIR_DATA_DIR
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:2950:  local data_dir = vim.env.PAIR_DATA_DIR
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:2959:      -- Many other files churn in $PAIR_DATA_DIR (agent-output, draft,
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:2972:-- Watch $PAIR_DATA_DIR for the change-log "build complete" marker (#58). The
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:3003:  local data_dir = vim.env.PAIR_DATA_DIR
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:3140:  local data_dir = vim.env.PAIR_DATA_DIR
cmd/internal/runtimebundle/assets/runtime/files/nvim/init.lua:3508:-- viewer. `zellij run` panes inherit the session env (PAIR_DATA_DIR etc.),
cmd/internal/transcript/transcript.go:3:// pair-context (ARCH-DRY) — extracted from cmd/pair-slug/main.go.
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-review-readiness:7:# the thin git-fact / git-effect shell (ARCH-PURE). JSON mode reports the case:
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-review-target:14:: "${PAIR_DATA_DIR:?pair-review-target: PAIR_DATA_DIR not set}"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-review-target:20:  cfg="$PAIR_DATA_DIR/config-$tag-$agent.json"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-review-target:26:  pid_file="$PAIR_DATA_DIR/agent-pid-$tag"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-review-target:57:out="$PAIR_DATA_DIR/review-target-$tag.json"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-review-open:9:# Env (exported by bin/pair): $PAIR_DATA_DIR $PAIR_TAG $PAIR_HOME
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-review-open:21:if [ -z "${PAIR_DATA_DIR:-}" ] || [ -z "${PAIR_TAG:-}" ] || [ -z "${PAIR_HOME:-}" ]; then
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-review-open:22:    printf 'pair-review-open: missing PAIR_DATA_DIR / PAIR_TAG / PAIR_HOME\n' >&2
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-review-open:31:STATE="$PAIR_DATA_DIR/review-$PAIR_TAG.open"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-review-open:42:export PAIR_NVIM_PID_FILE="$PAIR_DATA_DIR/nvim-pid-$PAIR_TAG-review"
cmd/internal/runtimebundle/assets/runtime/files/bin/lib/adapt-log.sh:5:# $PAIR_DATA_DIR/adapt-<tag>.jsonl, byte-identical in schema + field order to
cmd/internal/runtimebundle/assets/runtime/files/bin/lib/adapt-log.sh:20:    local dir="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-scrollback-open:10:#   $PAIR_DATA_DIR  — where pair stores per-session state files
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-scrollback-open:31:if [ -z "${PAIR_DATA_DIR:-}" ] || [ -z "${PAIR_TAG:-}" ] || [ -z "${PAIR_AGENT:-}" ]; then
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-scrollback-open:32:    printf 'pair-scrollback-open: missing PAIR_DATA_DIR / PAIR_TAG / PAIR_AGENT\n' >&2
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-scrollback-open:51:LOCK="$PAIR_DATA_DIR/scrollback-$PAIR_TAG-$PAIR_AGENT.openlock"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-scrollback-open:59:RAW="$PAIR_DATA_DIR/scrollback-$PAIR_TAG-$PAIR_AGENT.raw"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-scrollback-open:60:EVENTS="$PAIR_DATA_DIR/scrollback-$PAIR_TAG-$PAIR_AGENT.events.jsonl"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-scrollback-open:61:ANSI="$PAIR_DATA_DIR/scrollback-$PAIR_TAG-$PAIR_AGENT.ansi"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-scrollback-open:92:VIEWPORT="$PAIR_DATA_DIR/scrollback-$PAIR_TAG-$PAIR_AGENT.viewport"
cmd/internal/runtimebundle/assets/runtime/files/bin/pair-scrollback-open:186:export PAIR_NVIM_PID_FILE="$PAIR_DATA_DIR/nvim-pid-$PAIR_TAG-scrollback"
cmd/internal/runtimebundle/assets/runtime/files/bin/clipboard-to-pane.sh:15:# temp file at $PAIR_DATA_DIR/quote-<tag> and then triggering
cmd/internal/runtimebundle/assets/runtime/files/bin/clipboard-to-pane.sh:75:data_dir="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
cmd/internal/runtimebundle/assets/runtime/files/nvim/review/seam.lua:5:-- (milestone-review I3, ARCH-DRY: review.lua used `PAIR_TAG or 'default'`, which

exec
/bin/zsh -lc 'go test ./cmd/internal/runtimebundle ./cmd/internal/runtimebundlegen ./cmd/internal/entrypoint ./cmd/pair-go -count=1' in /Users/xianxu/workspace/pair
 succeeded in 2768ms:
ok  	github.com/xianxu/pair/cmd/internal/runtimebundle	0.459s
ok  	github.com/xianxu/pair/cmd/internal/runtimebundlegen	0.846s
ok  	github.com/xianxu/pair/cmd/internal/entrypoint	0.585s
ok  	github.com/xianxu/pair/cmd/pair-go	2.492s

codex
```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The implementation delivers the stated single-binary fallback: asset-root precedence is preserved, embedded extraction is wired after adjacent/default roots fail, docs/atlas were updated, and focused tests pass. I found no Critical issues. There is one production-readiness edge worth fixing cheaply before shipping: concurrent first-run extraction can race on shared `*.tmp` names.

**Strengths**
1. Pure planning is cleanly separated from IO in [manifest.go](/Users/xianxu/workspace/pair/cmd/internal/runtimebundle/manifest.go:24), [plan.go](/Users/xianxu/workspace/pair/cmd/internal/runtimebundle/plan.go:26), and [cleanup.go](/Users/xianxu/workspace/pair/cmd/internal/runtimebundle/cleanup.go:24). ARCH-PURE pass.
2. `cmd/pair-go` reuses `ResolveAssetRoot` and adds embedded runtime only as the last candidate, preserving current layouts in [main.go](/Users/xianxu/workspace/pair/cmd/pair-go/main.go:61). ARCH-DRY pass.
3. The copied-binary smoke test exercises real launch handoff paths and stale cleanup under `PAIR_DATA_DIR` in [pair-embedded-runtime-test.sh](/Users/xianxu/workspace/pair/tests/pair-embedded-runtime-test.sh:75). ARCH-PURPOSE pass.
4. Atlas coverage exists for the new runtime surface in [architecture.md](/Users/xianxu/workspace/pair/atlas/architecture.md:49) and [go-migration-inventory.md](/Users/xianxu/workspace/pair/atlas/go-migration-inventory.md:29).

**Critical Findings**

None.

**Important Findings**
1. [store.go](/Users/xianxu/workspace/pair/cmd/internal/runtimebundle/store.go:107): `writeFileAtomic` uses `path + ".tmp"` as a shared temp path. Two copied `pair` processes doing first-run extraction into the same digest can both write/rename the same temp file; one can fail with `rename ... no such file or directory`, making a valid copied-binary launch fail. Fix by using a per-process/random temp file in the destination directory (`os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")`) and renaming that, or by adding a store-level extraction lock. This is an ARCH-PURPOSE production-readiness gap for copied-binary deployment.

**Minor Findings**
1. [README.md](/Users/xianxu/workspace/pair/README.md:206): copied-runtime docs mention only the XDG/default path, while code prefers `PAIR_DATA_DIR` when set. Atlas is correct; README should say `${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}/runtime/...`.

**Test Coverage Notes**

I independently ran: `go test ./cmd/internal/runtimebundle ./cmd/internal/runtimebundlegen ./cmd/internal/entrypoint ./cmd/pair-go -count=1`, all passing. Existing tests cover manifest validation, extraction planning, cleanup, generation concurrency, `PAIR_DATA_DIR` precedence, and copied-binary smoke. Missing coverage: concurrent `runtimebundle.Extract` calls against the same `StoreRoot`/digest.

**Architectural Notes**

ARCH-DRY: pass. The generator and manifest types are the central packaging source; no blocking duplicate implementation logic found.

ARCH-PURE: pass. Decision logic is unit-testable without filesystem/process IO; extraction and exec stay in integration layers.

ARCH-PURPOSE: pass with the Important concurrency caveat. The copied-binary path is implemented and tested, but simultaneous first-run launches can still fail.

**Plan Revision Recommendations**

None.
tokens used
142,398
```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The implementation delivers the stated single-binary fallback: asset-root precedence is preserved, embedded extraction is wired after adjacent/default roots fail, docs/atlas were updated, and focused tests pass. I found no Critical issues. There is one production-readiness edge worth fixing cheaply before shipping: concurrent first-run extraction can race on shared `*.tmp` names.

**Strengths**
1. Pure planning is cleanly separated from IO in [manifest.go](/Users/xianxu/workspace/pair/cmd/internal/runtimebundle/manifest.go:24), [plan.go](/Users/xianxu/workspace/pair/cmd/internal/runtimebundle/plan.go:26), and [cleanup.go](/Users/xianxu/workspace/pair/cmd/internal/runtimebundle/cleanup.go:24). ARCH-PURE pass.
2. `cmd/pair-go` reuses `ResolveAssetRoot` and adds embedded runtime only as the last candidate, preserving current layouts in [main.go](/Users/xianxu/workspace/pair/cmd/pair-go/main.go:61). ARCH-DRY pass.
3. The copied-binary smoke test exercises real launch handoff paths and stale cleanup under `PAIR_DATA_DIR` in [pair-embedded-runtime-test.sh](/Users/xianxu/workspace/pair/tests/pair-embedded-runtime-test.sh:75). ARCH-PURPOSE pass.
4. Atlas coverage exists for the new runtime surface in [architecture.md](/Users/xianxu/workspace/pair/atlas/architecture.md:49) and [go-migration-inventory.md](/Users/xianxu/workspace/pair/atlas/go-migration-inventory.md:29).

**Critical Findings**

None.

**Important Findings**
1. [store.go](/Users/xianxu/workspace/pair/cmd/internal/runtimebundle/store.go:107): `writeFileAtomic` uses `path + ".tmp"` as a shared temp path. Two copied `pair` processes doing first-run extraction into the same digest can both write/rename the same temp file; one can fail with `rename ... no such file or directory`, making a valid copied-binary launch fail. Fix by using a per-process/random temp file in the destination directory (`os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")`) and renaming that, or by adding a store-level extraction lock. This is an ARCH-PURPOSE production-readiness gap for copied-binary deployment.

**Minor Findings**
1. [README.md](/Users/xianxu/workspace/pair/README.md:206): copied-runtime docs mention only the XDG/default path, while code prefers `PAIR_DATA_DIR` when set. Atlas is correct; README should say `${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}/runtime/...`.

**Test Coverage Notes**

I independently ran: `go test ./cmd/internal/runtimebundle ./cmd/internal/runtimebundlegen ./cmd/internal/entrypoint ./cmd/pair-go -count=1`, all passing. Existing tests cover manifest validation, extraction planning, cleanup, generation concurrency, `PAIR_DATA_DIR` precedence, and copied-binary smoke. Missing coverage: concurrent `runtimebundle.Extract` calls against the same `StoreRoot`/digest.

**Architectural Notes**

ARCH-DRY: pass. The generator and manifest types are the central packaging source; no blocking duplicate implementation logic found.

ARCH-PURE: pass. Decision logic is unit-testable without filesystem/process IO; extraction and exec stay in integration layers.

ARCH-PURPOSE: pass with the Important concurrency caveat. The copied-binary path is implemented and tested, but simultaneous first-run launches can still fail.

**Plan Revision Recommendations**

None.
