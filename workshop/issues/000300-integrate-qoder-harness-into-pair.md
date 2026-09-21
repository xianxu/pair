---
id: 000300
status: working
deps: []
github_issue:
created: 2026-09-20
updated: 2026-09-20
estimate_hours:
started: 2026-09-20T18:58:25-07:00
flow: {kind: quick, provenance: inferred, spec: "8c42eb10", done: "6d744b6d"}
---

# integrate qoder harness into pair

## Problem

pair hosts four harness CLIs — `claude`, `codex`, `agy`, `muse` — and the
operator's next harness, **qoder**, is registered nowhere: no TTY profile
(Return remap / overlay detection), no session scanner, no resume binding, no
slug/glyph support, invisible to couch. Separately, the bring-up guide
(`atlas/how-to-bring-up-a-new-harness-cli.md`) predates couch: it documents the
eight integration aspects as if Zellij+`pair wrap` were the only hosting path,
and never says what couch requires, what it consumes automatically, or that the
launcher agent registry is the single choke point both hosts share.

## Spec

**A. Atlas doc extension (enablement, do first).** Extend
`atlas/how-to-bring-up-a-new-harness-cli.md` to account for couch:

- Name the **agent registry** — `supportedAgents` in
  `cmd/internal/launcher/agent_defaults.go` (consumed via
  `AgentInventory()`/`IsSupportedAgent()`) — as the one list a new harness
  joins first. Couch, storagegc, rename/migrate, and switch-agent validation
  all derive from it; there is no couch-side agent table.
- Explain couch's hosting model: a hosted thread is spawned as
  `pair resume <tag> [layout-flag]` (`cmd/internal/couchcore/launch_existing.go`),
  so every pair-side adaptation (wrap profile, resume token, session
  inventory) applies under couch unchanged — couch adds no harness-specific
  surface of its own. Couch's start form and switch-agent menu enumerate
  `launcher.AgentInventory()` automatically; `SwitchLaunchCheck` additionally
  requires the agent executable on PATH.
- Note the coupling that matters for a bring-up: the sessioninventory scanner
  (aspect 3) is also what feeds couch's parked/live projection and its
  native-binding resume — no scanner, no couch resume.

**B. Qoder bring-up.** Follow the extended guide's checklist for `qoder`:

1. Registry: `supportedAgents` (launcher), `Agent` enum (sessioninventory), 
   fresh/resume arg validation (`fresh_args.go`, `agentargs.go`,
   `createlogic.go` `extractExplicitResume`).
2. TTY profile in `harnessTTYProfiles`: keymap + captured fixtures under
   `cmd/internal/wrapcmd/testdata/tty/qoder/<version>/` (capture-first
   discipline — no hand-authored bytes), composer recognizer backed by
   captures, overlay/picker markers, fail-closed gate until proven.
3. Session inventory: versioned facts-only scanner (+ roots in
   `NewOSRuntime`), event adapter for slug projection, sessionwatch
   `SupportsAgent`, sessionledger membership.
4. Recovery: `resumeToken`/`composeResumeArgs`, `AgentSessionExists`.
5. Slug + glyphs: `model.go` summarize dispatch + sandbox, prompt glyph in
   `nvim/scrollback.lua`, `wrapcmd/orientation.go` glyph map,
   `changelogcmd/distill.go`.
6. Settings: qoder permission whitelist (aspect 6).
7. Couch + live smoke: qoder appears in start/switch menus, hosted thread
   launch → park → resume round-trips, plain-Enter picker confirmation
   verified live.

## Done when

- `pair` boots qoder in the two-pane layout; plain Enter inserts a newline in
  the composer, Alt+Enter sends, plain Enter confirms pickers — pinned by
  captured fixtures replaying identically at every byte split.
- Restart-in-place and `pair resume <tag>` round-trip a qoder session; the
  session watcher establishes a binding from a completed native round.
- `pair-slug` summarizes a qoder session; Alt+b jumps between user prompts.
- Couch lists qoder in start/switch-agent, and a hosted qoder thread
  launches, parks, and cold-resumes through couch.
- `atlas/how-to-bring-up-a-new-harness-cli.md` reflects the couch
  architecture; `make test` green.

## Plan

Durable plan: `workshop/plans/000300-integrate-qoder-harness-into-pair-plan.md` (Part A done; Part B = M1–M5 below, each its own review boundary).

- [x] Extend `atlas/how-to-bring-up-a-new-harness-cli.md` for couch (Part A — 1e912472).
- [x] Author the durable bring-up plan via `superpowers-writing-plans` → `workshop/plans/000300-integrate-qoder-harness-into-pair-plan.md`.
- [ ] M1 — registry + launcher arg plumbing: `supportedAgents`/`Agent` enum join, fresh-args validation, resume token + `extractExplicitResume`; fail-closed intermediate verified.
- [ ] M2 — session inventory scanner: claude-family core extraction, `ScanQoder` + roots + wiring, event adapter + watcher/ledger membership + `AgentSessionExists`, live conformance.
- [ ] M3 — TTY: bootstrap positive-gated profile + live captures (`composer.raw`/`overlay.raw`) landing atomically (the capture harness requires the gate before bytes), composer recognizer spec, overlay markers, `--session-id` mint decision.
- [ ] M4 — slug + glyphs + settings: `runQoder` print invocation, prompt glyph in the consuming registrations (scrollback + orientation; distill only if it applies), permission/trust config.
- [ ] M5 — end-to-end: standalone pair live smoke (Enter/Alt+Enter/pickers/Alt+b/Alt+n/resume + doctor telemetry), couch round-trip, docs/atlas roster sweep, close.

## Log

### 2026-09-20

- Created after an inventory of every agent-registration point (registry,
  TTY profiles, sessioninventory, launcher args, model dispatch, glyphs,
  couch). Couch consumes the launcher registry with no table of its own and
  spawns hosted threads via `pair resume <tag>` — the wrap profile and
  inventory work apply unchanged, which is the fact the atlas doc is missing.
- Ran the full verb sequence late (exploration preceded `claim`/`start-plan`;
  caught up: claim → start-plan → change-code, quick flow inferred, in-place
  branch `000300-integrate-qoder-harness-into-pair`).
- Part A landed (1e912472): guide gains §0 (agent registry + couch hosting
  model), a scanner→couch coupling note in aspect 3, and checklist items 1
  (join the registry) and 10 (verify under couch). ARCH-DRY: §0 points at the
  registry instead of restating an agent roster; ARCH-PURPOSE: the couch
  coverage is the enablement class for the Part B bring-up, not an aside.
- Verified facts behind §0 against code: `launcher/agent_defaults.go`
  `supportedAgents`; `couchtty/menu_switchagent.go` uses
  `launcher.AgentInventory()`; `couchcmd/run.go` `SwitchLaunchCheck` uses
  `IsSupportedAgent` + LookPath pair/agent; `couchcore/launch_existing.go:58`
  spawns `pair resume <tag> [layout flag]`; couchcore resume/binding paths
  consume sessioninventory.

### 2026-09-21

- Plan-quality gate run manually (fresh-context qoder with ariadne's exact
  plan-quality prompt; sdlc's gate dispatch bypassed per operator direction):
  round 1 verdict FAILURE — one Important, one Minor. Important
  (`harness-test-oracle-mismatch`): M3's capture-first sequence contradicted
  the wrapcmd harness at four registration points — no `commands` row
  (`harness_tty_live_test.go:547-556`), the capture predicate itself requires
  an existing positive-gated profile with non-nil `recognize`
  (`:216-218`; `wrap.go:1565-1580`), captures and gate must land atomically
  (`harness_tty_fixture_test.go:93-96,134-136`), and
  `TestComposerReturnExpectationMatchesProfile` needs its own qoder row
  (`:803-819`). Minor (`roster-sweep-pattern-undermatches`): the docs-sweep
  regex missed Title-case/slash-joined rosters and the README line inventory
  was partly stale.
- Plan revised (see `## Revisions` in
  `workshop/plans/000300-integrate-qoder-harness-into-pair-plan.md`): M3 resequenced
  to Task 9 bootstrap+capture atomic, 10 recognizer spec, 11 markers, 12
  session-id + M3 boundary (13-20 renumbered); glyph task registers only where
  each consumer applies (distill.go deliberately lacks muse); docs sweep is
  sweep-driven, not a hand inventory.
- Found while regenerating the round-2 prompt: the binary resolves the durable
  plan as `<issue-basename>-plan.md` (`cmd/sdlc/reviewwindow.go:155`), so the
  off-convention filename meant the plan was never injected inline into either
  review round (round 1's reviewer read it from disk instead). Renamed to
  `workshop/plans/000300-integrate-qoder-harness-into-pair-plan.md`; the real
  gate at `change-code` will now carry it inline. Round 2 dispatched with the
  plan embedded + the round-1 findings as the prior-findings block.
- Round 2 verdict: **INFO** — PQ-1 `addressed`, PQ-2 `addressed`; plan approved
  to start. Two new Minors folded in (plan `## Revisions` round-2 entry):
  wrapcmd's `TestHarnessTTYProfileRegistry` row + the `ttyFixtureReactionGaps`
  oracle surfaced into Task 9 Step 6; `validAgent` (`model.go:444`) and
  `observationNativeID` (`target.go:199`) added to Tasks 6/7 with the wrong
  "agent-agnostic" claim corrected. Implementation continues via the normal
  sdlc flow (`change-code`), whose plan-quality gate now sees this plan inline.

