---
id: 000300
status: working
deps: []
github_issue:
created: 2026-09-20
updated: 2026-09-20
estimate_hours: 5.35
started: 2026-09-20T18:58:25-07:00
flow: {kind: full, provenance: inferred}
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

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

**Derivation:**
- **Design hours:** v2 ranges, with the ×0.2 spec-quality discount on every
  code primitive — the durable plan resolves their decisions with files, code
  and failure modes written out.
- **Undiscounted design:** `issue-spec` (the spec + plan + two external review
  rounds already spent since the claim) and `ux-rename-iteration` (live TTY
  adaptation is not pre-resolvable by a plan).
- **Library check (Step 2.5):** no external library applies; the short-circuit
  is internal — qoder is claude-family (claude-shaped JSONL transcripts), so
  the claude scanner core, normalizers and arg plumbing are mirrored rather
  than rebuilt. That is what holds the M2 items in the smaller-module range.
- **Implementation hours:** 40% of the v2 ranges (v3.1). Upper part for the
  items that ride live-capture iteration (M3 tui-screen, both
  `real-api-discovery` budgets) and for the multi-site wiring items.
- **Familiarity 1.0:** the repo's harness bring-up pattern (muse precedent +
  the atlas checklist) is established; the genuinely novel surfaces (qoder's
  TTY bytes, the `qoder -p` invocation) carry their own `real-api-discovery`
  budgets instead.
- **Design buffer +15%:** thorough plan doc (v2.1).
- **Boundaries:** one `milestone-review` per review boundary — M1–M4 closes
  plus the final close.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec              design=1.00 impl=0.08
item: smaller-go-module       design=0.06 impl=0.12
item: smaller-go-module       design=0.06 impl=0.16
item: cross-cutting-refactor  design=0.10 impl=0.16
item: greenfield-go-module    design=0.30 impl=0.24
item: smaller-go-module       design=0.06 impl=0.16
item: tui-screen              design=0.30 impl=0.28
item: real-api-discovery      design=0.00 impl=0.24
item: smaller-go-module       design=0.06 impl=0.12
item: smaller-go-module       design=0.06 impl=0.16
item: smaller-go-module       design=0.05 impl=0.08
item: real-api-discovery      design=0.00 impl=0.16
item: atlas-docs              design=0.04 impl=0.06
item: ux-rename-iteration     design=0.30 impl=0.08
item: milestone-review        design=0.00 impl=0.10
item: milestone-review        design=0.00 impl=0.10
item: milestone-review        design=0.00 impl=0.10
item: milestone-review        design=0.00 impl=0.10
item: milestone-review        design=0.00 impl=0.10
design-buffer: 0.15
total: 5.35
```

**Item order**, top to bottom:
1. Issue spec + durable plan + two review rounds (already spent).
2. M1: registry + enum join; fresh-args + resume-token/`extractExplicitResume` plumbing.
3. M2: claude-family scanner core extraction; `ScanQoder` + roots + dispatch wiring; event adapter + watcher/ledger membership + `AgentSessionExists`.
4. M3: bootstrap profile + recognizer + captures + registrations; live TTY discovery; overlay markers + session-id mint.
5. M4: slug via `qoder -p`; glyphs + settings.
6. M5: standalone smoke + couch round-trip; docs/atlas sweep; one operator iteration round.
7. Five boundary reviews (M1–M4 closes + final close).

## Plan

Durable plan: `workshop/plans/000300-integrate-qoder-harness-into-pair-plan.md` (Part A done; Part B = M1–M5 below, each its own review boundary).

- [x] Extend `atlas/how-to-bring-up-a-new-harness-cli.md` for couch (Part A — 1e912472).
- [x] Author the durable bring-up plan via `superpowers-writing-plans` → `workshop/plans/000300-integrate-qoder-harness-into-pair-plan.md`.
- [x] M1 — registry + launcher arg plumbing: `supportedAgents`/`Agent` enum join, fresh-args validation, resume token + `extractExplicitResume`; fail-closed intermediate verified.
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
- 2026-09-21: closed M1 — Boundary review round 2 findings addressed (BR-8..BR-14): codex --add-dir variadic regression fixed strictly per-agent and pinned (fresh_args.go freshVariadicOption + TestFreshVariadicOptionIsPerAgent, TestValidateFreshAgentArgs codex rows); qoder resume forms enumerated once (explicitResumeForms) and stripped at every persist site (persistedConfigArgs, sessionwatch.StripResumeArgs) with failing-first TestQoderShortResumeRoundTrip over space/short/inline; qoder registry parity is now mechanical (TestAgentInventoryParityWithSessionTables with named known-gap entry M2 must delete) and the intermediate schema_near_miss is pinned in the CLI golden matrix; session-side registry single-sourced (supportedAgents derives validAgent+usage); valueless --resume no longer returns the next flag as the id; --tools reachable in the qoder value list. Full go test ./... green (SUITE-EXIT=0).; review verdict: SHIP
- 2026-09-21: M1 advisories BR-15/BR-16 closed at rule level (daeb781b): resume spellings single-sourced in the new `cmd/internal/resumeform` table (extract, both persist strips and the strictly per-agent validator read it; glued `-r<id>` and inline `-r=` added; valueless `--resume` no longer eats the next flag) with table-ranging round-trip tests; parity test now probes `sessionledger.ParseLedger` and pins gap agents fail-closed. Plan Revisions records the M2 rule for the remaining dispatch sites (derive-or-probe; delete the qoder known-gap entry when M2 wires scanner/watcher/ledger).
- 2026-09-21: M2 implementation landed (800c8375, 3fdf596b, fa89157c) — claude-family core extraction, `ScanQoder` + roots + event/watcher/ledger membership + `AgentSessionExists`; the parity test's qoder known-gap entry deleted (scan/watch/ledger flip together); live conformance against the real `~/.qoder`: ok, 10 nodes, 7 roots, zero diagnostics.
- 2026-09-21: M2 boundary review (round 4, `sdlc milestone-close --milestone M2`) verdict **REWORK** — 4 blocking (BR-17 Critical: `daeb781b` regressed the fresh-arg validator on short-flag clusters `-pr sid`/`-vr sid`/`-hr x`; BR-18: qoder dispatch wiring at `ProviderContractFor`/`AdvanceTargetValidation` pinned by no test + no fail-closed defaults; BR-19: epoch-millis accepted with no range check; BR-20: qoder-only grammar admissions widened claude's scanner/event grammar) + 4 Minor (BR-21..BR-24). Artifacts: `workshop/plans/000300-integrate-qoder-harness-into-pair-m2-review.md`, close-gate ledger round 4 (BLOCKED). M2 close NOT finalized; checkbox stays unticked.
- 2026-09-21: M2 review fix round 1 (56f488dd) — BR-17/19/20/21/22/23 addressed at rule level: cluster letters now derive from `resumeform.ShortLetters` (table-ranged `-p<letter>` test + base rows kept); `Strip(agent, args)` strictly per-agent with the codex/muse subcommand strip gated too, `forms` unexported behind `Forms()`; agent threaded through every production caller (launcher ×6, sessionwatch, wrapcmd, couchcore); `claudeFamilySpec.acceptsMillis` bounds millis to 2000-01-01..9999-12-31 (out-of-range disputes; claude rejects numeric timestamps — negative rows) and `claudeFamilyNoiseTypes` keeps qoder's bookkeeping near-miss for claude (five claude negative rows); `file-history-snapshot` joins qoder's ignore set on measured evidence (all 53 near-misses over 10 real transcripts). Verified: focused packages green, full `go test ./...` EXIT=0. Remaining before re-close: BR-18 (qoder rows in six per-agent test tables, fail-closed defaults in the two incremental switches, parity-ranged dispatch probe) + BR-24 (plan Revisions entry).
- 2026-09-21: M2 review fix round 2 — BR-18/BR-24 addressed: qoder rows added to six per-agent test tables (`TestProviderContractFor`, `TestAppendOnlyProviderConformance`, `provider_live_fake_test.go` validateLiveRecords + agent list, `TestNativeLargeRecordsScanAndEvents`, `FuzzScanQoderV1Records`, `TestQuerySessionCatalogLossProofClassCoversEveryAgentWithoutBodyReads`); fail-closed `default:` arms added to `ValidateTargetWork` and `AdvanceTargetValidation` (unknown agents error instead of silently skipping); `TestEveryAgentDispatchParity` (new, `dispatch_parity_test.go`) ranges scanner → ProviderContractFor → ObserveStableArtifact → ValidateDelta → NativeEventsFromRecords for every JSONL agent; plan `## Revisions` M2 entry records claude-family admissions, live conformance evidence, Task 7 filename correction, and the BR-16 rule as delivered.

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
- `sdlc change-code` inference: **full** ("a design of 837 lines (limit 500)";
  M1–M5 rows). Plan-quality gate (binary dispatch, claude) round 1:
  **INFO/passed** — no blocking findings; 7 advisory Minors recorded in
  `workshop/plans/000300-integrate-qoder-harness-into-pair-plan-gate.md` for
  the close review (registry half-joins in M1 + no parity test; `usage.go:60`
  context meter + create-path `shouldMintClaudeSessionID` missed by the hand
  inventory; Task 9 Step 6 omits `assertFixtureIsMachineNeutral`; Tasks
  2/3/5/6/7/13/14 embed stale implementation bodies; Task 7 Step 5 names a
  non-existent test — the real one is
  `TestLiveNativeSessionShapeConformance` gated by
  `PAIR_LIVE_NATIVE_SESSIONS=1`; M5 has no own milestone-close; no non-goals).
  Carry each into implementation as it becomes live (use the correct live-test
  name in Task 7; extend `usage.go` or list as non-goal; neutral-cwd capture
  policy in Task 9).
- Estimate derived per `estimate-logic-v3.1` (Method A, familiarity 1.0,
  +15% design buffer): **total 5.35h** — 19 items, item order in
  `## Estimate`. Reconciliation verified (recomputed 5.3485, tol 0.2675).

