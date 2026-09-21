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

- [x] Extend `atlas/how-to-bring-up-a-new-harness-cli.md` for couch (Part A — this session).
- [ ] Author the durable bring-up plan via `superpowers-writing-plans` → `workshop/plans/` when implementation starts (Part B is a #134-muse-scale effort: full flow, milestones to be defined there, not here).

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

