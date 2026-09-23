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
- [x] M2 — session inventory scanner: claude-family core extraction, `ScanQoder` + roots + wiring, event adapter + watcher/ledger membership + `AgentSessionExists`, live conformance.
- [x] M3 — TTY: bootstrap positive-gated profile + live captures (`composer.raw`/`overlay.raw`) landing atomically (the capture harness requires the gate before bytes), composer recognizer spec, overlay markers, `--session-id` mint decision.
- [x] M4 — slug + glyphs + settings: `runQoder` print invocation, prompt glyph in the consuming registrations (scrollback + orientation; distill only if it applies), permission/trust config.
- [x] M5 — end-to-end: standalone pair live smoke (Enter/Alt+Enter/pickers/Alt+b/Alt+n/resume + doctor telemetry), couch round-trip, docs/atlas roster sweep, close.

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

### 2026-09-21 (M5 live smoke)
- Operator live-verified from inside a pair-hosted qoder session: the AskUserQuestion picker round-tripped correctly — overlay rendered, arrows navigated, Enter confirmed a choice (no newline leak), selection returned to the agent. Exercises the M3-captured overlay path (`detectQoderOverlayOpen` + Return remap) against the running harness, not just the frozen fixtures.

### 2026-09-21
- 2026-09-21: closed M4 — Round-10 findings BR-42..BR-45 + minors E/F/H fixed at rule level: registry-ranging glyph table test (red-first, 4 rows) + exported PromptGlyph accessor with derived parity assertion (qoderPromptCol mutation reddens both parity tests); runQoder passes --no-session-persistence (wantArgs red-first, live conformance PASS 5.57s, TMPDIR project dir measured byte-untouched); Task 17 gained Step 3 (settled-footer capture -> isFooterChrome -> no-op Alt+l); Lua parity escape switched to Vim dialect; allowlist moved to repo-local .qoder/settings.local.json with measured scope (repo allow / tmp ask) and no prefix-rule leak into compound tails; pump OSC test table-driven over claude+codex (bound-first order reddens both rows). Full go test ./... EXIT=0 (74 packages).; review verdict: SHIP
- 2026-09-21: closed M3 — Round-9 findings BR-35..BR-40 fixed with pins (all mutation-verified red-then-restored); full go test ./... EXIT=0; atlas muse-sweep executed and logged; review verdict: SHIP
- 2026-09-21: closed M2 — BR-18/BR-24/BR-25 addressed: SupportedAgents() exported, parity test derives from it and drives production ValidateTargetWork + AdvanceTargetValidation with fixture+append; both unknown-agent default arms mutation-verified red (ValidateTargetWork: schema_near_miss diagnostic via artifactDiagnostic; AdvanceTargetValidation: builds prior from real ValidateTargetWork result, overrides State.Agent, asserts ErrArtifactChanged); default arms harmonized to artifactDiagnostic shape; TestAdvanceTargetValidationPerAgent ranges SupportedAgents(); plan Revisions corrected (field names, slug, narrowed claims). Full go test ./... EXIT=0. No new atlas surface (test coverage + defensive defaults + one accessor export).; review verdict: FIX-THEN-SHIP
- 2026-09-21: closed M1 — Boundary review round 2 findings addressed (BR-8..BR-14): codex --add-dir variadic regression fixed strictly per-agent and pinned (fresh_args.go freshVariadicOption + TestFreshVariadicOptionIsPerAgent, TestValidateFreshAgentArgs codex rows); qoder resume forms enumerated once (explicitResumeForms) and stripped at every persist site (persistedConfigArgs, sessionwatch.StripResumeArgs) with failing-first TestQoderShortResumeRoundTrip over space/short/inline; qoder registry parity is now mechanical (TestAgentInventoryParityWithSessionTables with named known-gap entry M2 must delete) and the intermediate schema_near_miss is pinned in the CLI golden matrix; session-side registry single-sourced (supportedAgents derives validAgent+usage); valueless --resume no longer returns the next flag as the id; --tools reachable in the qoder value list. Full go test ./... green (SUITE-EXIT=0).; review verdict: SHIP
- 2026-09-21: M1 advisories BR-15/BR-16 closed at rule level (daeb781b): resume spellings single-sourced in the new `cmd/internal/resumeform` table (extract, both persist strips and the strictly per-agent validator read it; glued `-r<id>` and inline `-r=` added; valueless `--resume` no longer eats the next flag) with table-ranging round-trip tests; parity test now probes `sessionledger.ParseLedger` and pins gap agents fail-closed. Plan Revisions records the M2 rule for the remaining dispatch sites (derive-or-probe; delete the qoder known-gap entry when M2 wires scanner/watcher/ledger).
- 2026-09-21: M2 implementation landed (800c8375, 3fdf596b, fa89157c) — claude-family core extraction, `ScanQoder` + roots + event/watcher/ledger membership + `AgentSessionExists`; the parity test's qoder known-gap entry deleted (scan/watch/ledger flip together); live conformance against the real `~/.qoder`: ok, 10 nodes, 7 roots, zero diagnostics.
- 2026-09-21: M2 boundary review (round 4, `sdlc milestone-close --milestone M2`) verdict **REWORK** — 4 blocking (BR-17 Critical: `daeb781b` regressed the fresh-arg validator on short-flag clusters `-pr sid`/`-vr sid`/`-hr x`; BR-18: qoder dispatch wiring at `ProviderContractFor`/`AdvanceTargetValidation` pinned by no test + no fail-closed defaults; BR-19: epoch-millis accepted with no range check; BR-20: qoder-only grammar admissions widened claude's scanner/event grammar) + 4 Minor (BR-21..BR-24). Artifacts: `workshop/plans/000300-integrate-qoder-harness-into-pair-m2-review.md`, close-gate ledger round 4 (BLOCKED). M2 close NOT finalized; checkbox stays unticked.
- 2026-09-21: M2 review fix round 1 (56f488dd) — BR-17/19/20/21/22/23 addressed at rule level: cluster letters now derive from `resumeform.ShortLetters` (table-ranged `-p<letter>` test + base rows kept); `Strip(agent, args)` strictly per-agent with the codex/muse subcommand strip gated too, `forms` unexported behind `Forms()`; agent threaded through every production caller (launcher ×6, sessionwatch, wrapcmd, couchcore); `claudeFamilySpec.acceptsMillis` bounds millis to 2000-01-01..9999-12-31 (out-of-range disputes; claude rejects numeric timestamps — negative rows) and `claudeFamilyNoiseTypes` keeps qoder's bookkeeping near-miss for claude (five claude negative rows); `file-history-snapshot` joins qoder's ignore set on measured evidence (all 53 near-misses over 10 real transcripts). Verified: focused packages green, full `go test ./...` EXIT=0. Remaining before re-close: BR-18 (qoder rows in six per-agent test tables, fail-closed defaults in the two incremental switches, parity-ranged dispatch probe) + BR-24 (plan Revisions entry).
- 2026-09-21: M2 review fix round 2 — BR-18/BR-24 addressed: qoder rows added to six per-agent test tables (`TestProviderContractFor`, `TestAppendOnlyProviderConformance`, `provider_live_fake_test.go` validateLiveRecords + agent list, `TestNativeLargeRecordsScanAndEvents`, `FuzzScanQoderV1Records`, `TestQuerySessionCatalogLossProofClassCoversEveryAgentWithoutBodyReads`); fail-closed `default:` arms added to `ValidateTargetWork` (emits `schema_near_miss` diagnostic) and `AdvanceTargetValidation` (returns `ErrArtifactChanged`); `TestEveryAgentDispatchParity` derives from `SupportedAgents()`, drives production `ObserveAgentMetadata` → `ValidateTargetWork` → `NativeEventsFromRecords` per agent; `TestAdvanceTargetValidationPerAgent` drives production `AdvanceTargetValidation` with appended records for all four JSONL agents; unknown-agent tests pin both default arms; plan `## Revisions` M2 entry records claude-family admissions, live conformance evidence, Task 7 filename correction, and the BR-16 rule as delivered.
- 2026-09-21: M2 review fix round 3 (round-5 review feedback) — BR-18/BR-24/BR-25 overclaim corrections: `sessioninventory.SupportedAgents()` exported as the single session-side agent list; parity test derives from it (no hardcoded agent table); `ValidateTargetWork` default arm now emits a diagnostic (not just sets err + continues silently); advance test uses files with proper StableFileID/GenerationToken so fingerprint growth detection works; plan Revisions corrects `acceptsEpochMillis` → `acceptsMillis`, `extraIgnoredTypes` → `claudeFamilyNoiseTypes`, slug typo `untrusted-inputarsed` → `untrusted-input-parsed`; BR-16 bullet rewritten to state only what each test pins.
- 2026-09-21: M2 review fix round 4 (round-6 FIX-THEN-SHIP advisories) — three advisory findings closed: (1) `TestAdvanceTargetValidationRejectsUnknownAgent` rebuilt from a real `ValidateTargetWork` prior with `State.Agent` overridden to `"future"`, now reaches and pins the `AdvanceTargetValidation` default arm (arm-deletion mutation verified red); (2) `TestAdvanceTargetValidationPerAgent` ranges `SupportedAgents()` (skipping agy) instead of a hardcoded four-agent list, adds event non-loss assertion; (3) both default arms harmonized to use `artifactDiagnostic` for consistent diagnostic shape. Plan BR-16 bullet corrected to match.
- 2026-09-21: M3 implementation landed (7255fcc3, 779777d4, a346b607, 133a5b58) — bootstrap profile + live capture atomic (7255fcc3: qoder positively gated on the column-1 ruled-box recognizer, `composer.raw` captured live from 1.1.60); recognizer spec over the frozen capture (779777d4); `--session-id` mint pinned (a346b607: qoder honored a caller-minted id, transcript landed under `~/.qoder/projects/<slug>/<minted id>.jsonl`; `launcher.MintsSessionID` is the single source, both fresh paths); overlay markers (133a5b58).
- 2026-09-21: M3 overlay evidence — both qoder declining families captured live-driven, never hand-authored. Permission picker: measured 2026-09-21 that `--permission-mode default` still auto-ran `ls -la` inside the trusted repo but asked before `ls -ld /tmp` (path outside the workspace), so the scenario drives that command; markers verbatim from the strip ("Permission Required", glued "Rejectandtypesomething", …). Question picker (AskUserQuestion UI): "Asking User" header over a ruled card, footer `↑↓navigate·Enterselect·Esccancel`. Driven captures are now bounded by `trimmedHarnessTTYCapture` to the final synchronized paint block only when that block replays to the same Return decision (verified, pinned by `TestTrimmedHarnessTTYCapture`; 55KB→5.5KB and 104KB→6.6KB; the slash menu kept its full 13KB). `ttyFixtureNegativeGaps["qoder"]` dropped; discrimination/reaction entries rewritten to what the two captures prove (both decline on shape; nobody pressed Return on either screen).
- 2026-09-21: M3 all-splits replay caught a real split-boundary bug — `selection.raw` split at 6426/6616 severed `Enterselect` because per-chunk stripping keeps the chunk-split `\x1b[23` verbatim and the next chunk's `m` completes it ahead of the text; the header marker meanwhile sat beyond the stripped tail's horizon. Fixed in `detectQoderOverlayOpen` with a raw rolling-buffer pre-check (byte-contiguous, cannot be corrupted by a boundary inside an escape; mirrors claude/codex scanning `rolling` for OSC). `go test ./cmd/internal/wrapcmd` green including the full fixture replay (35.7s).

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

- 2026-09-21: M3 boundary review (round 7, `sdlc milestone-close --milestone M3`, window `301c5381..1b1977c4`) verdict **REWORK** — Critical BR-28 (raw rolling-scan re-armed `pickerActive` off consumed picker bytes: paint → Enter → small chunk → next composer Enter submitted a draft), Important BR-29 (`qoderComposerActive` near-copied `ruledBoxComposerActive` instead of a spec), BR-30 (prompt column/glyphs restated in the recognizer and `orientation.go`), BR-31 (the `claudeComposerRule` skip landed for every agent in the shared orientation loop), BR-32 (create-path mint unpinned), plus minors. Artifacts: `workshop/plans/000300-integrate-qoder-harness-into-pair-m3-review.md`; close NOT finalized.
- 2026-09-21: M3 review fix round 1 — BR-28: `overlayRawTail` is proxy-owned and cleared in `emitPlainCR` beside `overlayTextTail`; `TestCheckOverlayOpen_QoderDoesNotRedetectStalePickerText` drives paint→Enter→chunk over both frozen captures. BR-29/30: `ruledBoxComposerSpec` gained `promptCol` + `requireVisibleCursor`; `qoderComposerActive` is a spec registration (the shared loop owns the only ruled-box scan); `qoderPromptCol`/`qoderPromptGlyphs` are the single authority, read by both the recognizer and `orientationPromptOK`. BR-31: the rule-cell skip is gated to qoder; `TestOrientationRuleCellToleranceStaysPerProfile` pins claude/muse/agy flipping true under the un-gated skip (mutation-verified), codex's recognizer barrier, and the qoder positive contrast. BR-32: `TestRunLaunchForcedCreateQoderMintProbesQoderSessions` pins the `qoder|MINTED-1` collision retry to `MINTED-2` and that a claude collision does not block qoder (both fail against a literal `"claude"` probe). Minors: `forfuturesessions` dropped from `qoderPickerMarkers` (generic prose that glues in any word-by-word paint would arm the overlay off agent output) with a glued-prose negative row; fixture-expectation comment corrected; `atlas/architecture.md` `--session-id` no longer claude-only; Task 14 (M4) amended to derive from `qoderPromptGlyphs`, plan Goal at 1.1.60, Revisions entry added. Verified: `go test ./...` EXIT=0 (wrapcmd 31s, launcher 9s included); mutation checks for the three new pins all red-then-restored.
- 2026-09-21: M3 review fix round 2 (round-9 findings) — BR-35 (Important): `detectQoderOverlayOpen` scans `stripTerminalControls(prevTail+data)` BEFORE re-bounding the carry, so a split inside a marker's escape survives kilobytes of trailing bytes (`TestCheckOverlayOpen_QoderSplitFooterSurvivesLongSecondChunk`, 0/600/2000 filler; bound-first order mutation reddens 600/2000) — supersedes the "byte-contiguous window cannot be corrupted" wording. BR-36 (Important): the muse sweep (`grep -n -i muse atlas/*.md`) executed; architecture.md :694/:702/:704/:903, session-identity.md :15/:24/:94, index.md :3 and the how-to (:5, resume-flag bullet, qoder marker bullet) extended; deliberate non-extensions all checked (couch.md:321 M5; how-to PROMPT_PATTERN_BY_AGENT M4 — scrollback.lua has no qoder row; how-to status-line M5; architecture.md:895 claude-only `endOfTurnByAgent`; :912 `model.Run` defaults to runClaude; :1060 no agent list). BR-37: `requireVisibleCursor` inverted to fail-closed `allowHiddenCursor` (only qoder sets it) + `TestOrientationUncoloredAgyRequiresVisibleCursor` (hidden row mutation-verified). BR-38: atlas rule amended — title-case "Permission Required" is a deliberate exemption, boundary pinned by the spaced-prose negative row (marker-widening mutation reddens exactly it). BR-39: `harnessTTYProfile.orientationPromptCol`/`orientationRuleCellTolerant` carry qoder's quirks; orientation.go no longer string-compares (zeroing either field reddens its pin). BR-40: `overlayVisible` + `firstMarker` collapse the four carry blocks and three marker loops (muse's folded loop kept for the vary-case reason spelling). Carried minima re-verified at HEAD for the next round: BR-26/27 present in `14caa844`/`301c5381` (SupportedAgents()-ranged advance test; both default arms use artifactDiagnostic), BR-25's claim backed at HEAD. Verified: full `go test ./...` EXIT=0; all pins mutation-checked red-then-restored.
- 2026-09-21: side-quest (M3 boundary advisory, BR-35 family): the shared chunk pump bounded `rolling` to `rollingTailLen` BEFORE `checkOverlayOpen` and the OSC scan, so a Claude/Codex OSC sitting more than a tail's worth of bytes before the end of one chunk was never scanned at all — the sibling instance of the rule BR-35 fixed for Qoder's raw window ("scan at full carry+chunk length, bound afterwards"). The trim now runs after both scans and after the advance past the last OSC match, so the carry stays bounded on every path. Pinned by `TestHandleChunk_OscScannedBeforeCarryIsBounded` (claude picker OSC + 640 filler bytes in one chunk; red under the bound-first order). Full `go test ./...` EXIT=0 (74 packages ok).
- 2026-09-21: M4 Task 13 (qoder slug generation) — Step 1 live pin: `qoder -p "Reply with exactly: ok"` from `/tmp` prints exactly `ok`, exit 0 (no workspace coupling); `qoder --list-models` lists 17 entries (Auto/Ultimate/Performance/Efficient; Qwen3.8/3.7 Max/Plus/Flash; GLM-5.3(+Flash); DeepSeek-V4-Pro/Flash; Kimi-K3/K2.8-Preview; MiniMax-M3; Sonus/Cantus) with no pricing metadata, so `DefaultQoderModel` pins the cheap/fast tier alias **`Efficient`** — a tier alias rather than a model id so the pin survives qoder's model-generation churn. Invocation shape verified live: `-p --model <name> <prompt>` + stdin is consumed as input (BANANA round-trip), ~4.6s per call. Step 2: failing dispatch test `TestRunQoderDispatchesToQoderCLI` (fake-binary seam as in `TestRunCodexCLIWithoutAPIKey`; the plan's "see how runAgy is tested" was stale — no runAgy test exists) — red before the fix (fell through to runClaude, real claude binary "exit status 1"). Step 3: `runQoder` (+`Run` case, `DefaultModel("qoder")`, `Request.Agent` doc) — argv `-p --model`, stdin, `cmd.Dir = os.TempDir()`, `PAIR_SLUG_NESTED=1`. Step 4: `go test ./cmd/internal/model ./cmd/internal/slugcmd ./cmd/internal/changelogcmd` green; new gated live conformance `TestRunQoderLiveConformance` (`PAIR_LIVE_QODER_MODEL=1`) PASS 4.79s against the installed CLI through the production dispatch; `make pair` rebuilt `bin/pair`; `bin/pair-slug` with `PAIR_AGENT=qoder` + scratch data dir exits 0 logging "native session is unbound; slug waits for an established binding" (tolerant path). The binding-driven end-to-end run (`slug-parse` fired in the adapt log) carries to M5 Task 17's standalone smoke — no qoder pair session exists yet to drive it (first launch is M5's).
- 2026-09-21: M4 Task 14 (qoder prompt glyph — scrollback + distill) — the two glyph consumers the M3 review note named (plan Revisions, line 860). **Sync mechanism named and built:** `nvim/scrollback.lua` gets `qoder  = [=[^ [*>]]=],` (glyph at `qoderPromptCol` 1 ⇒ `^ [*>]`; leveled long string because `[[^ [*>]]]` fuses the class-closing `]` with the `]]` delimiter) and the M3-owed sync story is the red-first parity test `TestScrollbackQoderPatternTracksPromptAuthority` (`cmd/internal/wrapcmd/scrollback_glyph_parity_test.go`): it DERIVES the expected pattern from `qoderPromptGlyphs` (sorted, class-escaped) + `qoderPromptCol` and requires scrollback.lua to carry exactly one matching qoder row — red at "found 0" before the row; mutation-verified (dropping `*` from the map reddens, since the Lua row is the only consumer that cannot read the Go map). `nvim/scrollback_test.lua` test 4 pins the semantics (` > x`/` * x` match; flush-left `>` and extra-indented `  >` reject). **distill registration justified by proven consumption**: `PAIR_AGENT` → `opener.RunChangelogCLI` → `distillerEnv` PCL_AGENT → `--agent "$PCL_AGENT"` → `glyphFor(agent)`, and `model.Run` supports qoder since Task 13 — so `cmd/internal/changelogcmd/distill.go` registers `"qoder": " >"` (leading space = the captured col-1 echo; `scanTurnBoundaries` test pins `" > ..."` matching while flush-left/indented reject and claude doesn't match qoder lines). Yolo `*` deliberately omitted there, like agy's: captured echo evidence covers default mode only and a missed boundary degrades gracefully (extra lookback), never corrupts the log. **Findings (not fixed here):** (a) qoder's live footer rows — last-row status churn (`Model · ctx …%`), `? for shortcuts`, `Shift+Tab to Accept Edits`, the placeholder composer row — match none of `isFooterChrome`'s rows, so `trimLiveTail` strips nothing for qoder (anchor-leak/FullRedistill risk, #58 family); the settled-session cleaned shape must be captured in M5 Task 17 before extending the recognizer. (b) muse is consumed-but-absent: distill has no muse row (falls back to claude `❯`) and scrollback.lua's `muse = [[^>]]` covers only one of `musePromptGlyphs` {⟩, ›, ❯, >} → peer finding for its own issue, out of scope here per plan. Verification: `go test ./cmd/internal/wrapcmd ./cmd/internal/changelogcmd -count=1` green (31.2s/2.9s); `make test-lua` EXIT=0; runtimebundle asset regenerated in sync. The route to a green `make test-lua` surfaced a pre-existing red: `nvim/init.lua`'s load-time `layout_write('small')` opened a nil `PAIR_LAYOUT_MODE_PATH` in three `tests/workbench-route-nvim-test.sh` invocations (red at de56ea58 in a detached baseline worktree) — fixed test-side as side-quest 03abb18a.
- 2026-09-21: M4 Task 15 (settings, aspect 6) — **the allowlist exists**; registered and live-verified. Evidence: `qoder --help` exposes `--permission-mode` (default/accept_edits/bypass_permissions/dont_ask/auto), `--allowed-tools`/`--disallowed-tools`/`--tools`, `--settings` (file or inline JSON), `--setting-sources` (user/project/local); binary inspection of qodercli 1.1.60 shows the settings loader reading `permissions.{allow,deny,ask}` string arrays from every layer (user `~/.qoder/settings.json`, project `<repo>/.qoder/settings.json` team-shared, local `<repo>/.qoder/settings.local.json` gitignored — paths from in-binary help copy), tagged per source; rule syntax per the in-binary rule dialog: "a tool name, optionally followed by a specifier in parentheses … e.g., WebFetch or Bash(ls:*)", with `Bash(cmd:*)` = "Any Bash command starting with cmd" (prefix match ⇒ decision points `shell.rule_exact.allow`/`shell.rule_prefix.allow`); the interactive settings *dialog* schema surfaces only `additionalDirectories`+`trustDirectories` — allow/deny/ask are JSON/flag-level surface. There is also a built-in read-only auto-allow heuristic (`shell.readonly.allow`; kill-switch `QODER_DISABLE_READONLY_SHELL_AUTO_ALLOW=1`) and a `general.defaultPermissionMode` setting. **Live A/B (headless probes, decision points read from `~/.qoder/logs/runs/*/qodercli.log`):** before — `git status --short` → `shell.readonly.allow` (readonly heuristic already covers read git), but `make --version`, `sdlc --help`, `lsof -c zellij`, `zellij --version` → `shell.no_match.ask` ("No rule covers this command"), denied without a TTY; after — registered `Bash(git:*)`, `Bash(make:*)`, `Bash(sdlc:*)`, `Bash(lsof:*)`, `Bash(zellij:*)` in `~/.qoder/settings.json` (`permissions.allow`, backup at `/tmp/qoder-settings-backup-1790033243.json`), and the identical probe resolves all five `shell.rule_prefix.allow`, commands executed. `trustDirectories` already covers `/Users/xianxu/workspace/pair` (verified in-file); the `../ariadne` addition stays an **open operator knob** — no demonstrated cross-repo need yet (plan marks it operator decision; revisit at M5 if cross-repo smoke needs it). Corroboration that the engine persists rules: the repo's `.qoder/settings.local.json` (untracked, local-only) already carries one allow entry qoder itself wrote from an interactive always-allow in an earlier live session. No tests (static config; the A/B probe is the verification).
- 2026-09-21: M4 review fix round (round-10 verdict FIX-THEN-SHIP; four open Importants + minors E/F/H) — all closed at rule level. **BR-42:** `trimLiveTail`'s empty-box check compared `TrimSpace(line)` to the raw glyph, so qoder's space-prefixed `" >"` never matched `">"`; the registry row now drives both readers by construction — `TestPromptGlyphRowsDriveBothReaders` ranges `promptGlyphChar` over `scanTurnBoundaries` AND `trimLiveTail` with each agent's drawn box row (red-first: qoder row only), and `trimLiveTail` normalizes the glyph once (`strings.TrimSpace`; the boundary regex keeps the leading space). **BR-43:** `changelogcmd.PromptGlyph(agent)` is the exported accessor; `TestDistillQoderGlyphTracksPromptAuthority` (wrapcmd) derives the expected value from `qoderPromptCol` + ">" and asserts equality, pins the yolo `*` omission from both ends; mutation `qoderPromptCol 1→0` reddens it and the Lua parity test. **BR-44:** `runQoder` passes `--no-session-persistence` (argv pinned red-first via `wantArgs`); live conformance PASS 5.57s, and the TMPDIR project dir was measured byte-untouched (same 2 files, mtimes 16:00:28; the pre-flag jsonl is the reviewer's measured residue, now not reproducible). **BR-45:** plan Task 17 gained Step 3 — settled-footer capture → `isFooterChrome` extension → no-op Alt+l verification (the carried #58-class item now lives in the executing task, restated in Revisions). **E:** the Lua parity test escapes in the Vim dialect (`\`, `\]`, `\-`, `\^` — consumed by `vim.fn.search`) and the missing-file fatal names drift. **F:** the allowlist moved from user scope to `<repo>/.qoder/settings.local.json` (merged with qoder's own entry); measured — repo cwd `make --version` → `shell.rule_prefix.allow` (ran, GNU Make 3.81), `/tmp` cwd same command → `shell.no_match.ask` (denied; scope is repo-bound), and the chained probe `git rev-parse --show-toplevel && mkdir -p /tmp/pair300-chain-probe` → single `shell.no_match.ask` (denied; no prefix-rule leak into compound tails, dir not created); `~/.qoder/settings.json` restored to pre-M4 (backups `/tmp/qoder-settings-backup-1790033243.json`, `-1790034148.json`). **H:** `TestHandleChunk_OscScannedBeforeCarryIsBounded` is table-driven over every OSC-reading profile (claude 777 + codex `9;Plan mode prompt:`); both rows red under the restored bound-first order, green after. Verified: full `go test ./...` EXIT=0 (74 packages ok).

### 2026-09-22 — M5 continuation

- Operator reports qoder now loads in a Couch-hosted thread. The parked Pair TTY window for `couch-16efd58677ab012e` shows Qoder CLI v1.1.61 answering two prompts; it retains only 40 rendered lines, so it does not prove earlier interactions. Local `agent-ready-couch-16efd58677ab012e-qoder.json` and `config-couch-16efd58677ab012e-qoder.json` corroborate the hosted launch. This accepts Task 18's launch observation only; menu listing, park/cold-resume, switch-agent both ways, and parked/live projection still need direct evidence.
- Task 19 roster sweep landed in 5501f23f: README, Couch and architecture atlas, doctor README and skill, and CHANGELOG. `atlas/index.md` and `atlas/session-identity.md` already name Qoder; no new terminal protocol fact was found for `atlas/terminal.md`. The how-to status line refers specifically to #134's historical telemetry verification, so it remains unchanged until Qoder telemetry is measured.
- Verification at this commit: `go test ./...` exit 0; `make -f Makefile.local test-native-terminal-ci` exit 0 (five Python checks and all six required native tests/subtests); `git diff --cached --check` exit 0 before the docs commit. `doctor/doctor.sh` on the Qoder standalone and Couch adapt-log paths reports `NO-DATA`: the log files are absent, while both emitter-health probes say `[ok]`. M5 Task 17's telemetry criterion remains open.
- Remaining M5 live evidence: standalone Enter/Alt+Enter, permission picker, mouse scroll, Alt+b, resize, Alt+n, and cold `pair resume`; settled-footer Alt+l no-op (the code landed in e8d6621c, but needs a live repeat); Couch park/cold-resume and switch-agent round-trip with correct state projection. The already recorded AskUserQuestion picker confirmation covers that one picker behavior only. Keep the M5 checkbox and close gate open until those observations are recorded.

### 2026-09-22 — M5 acceptance completed

- Operator confirmed the remaining standalone and Couch smoke items after the prior log: composer Enter/Alt+Enter, permission picker, mouse scroll, Alt+b, resize, Alt+n, cold `pair resume`, settled-footer Alt+l no-op, and Couch park/cold-resume plus switch-agent round-trips and state projection. The earlier AskUserQuestion picker observation remains separately recorded. This is operator-reported live evidence; the old TTY renderer retains only a bounded window.
- Fresh doctor probe used an isolated Pair Qoder session (`PAIR_DATA_DIR=/tmp/pair300-doctor.by3RuW`, tag `pairqdoctor300`), sent a prompt through the Neovim draft to establish the native binding, and ran production `pair-slug` on that binding. `doctor/doctor.sh` read four real events: `return-remap/fired: 1`, `session-id/fired: 1`, `slug-parse/fired: 2`; zero `near-miss` and zero `fail`; pair-wrap and pair-slug emitter health both `[ok]`. No bypass key was pressed in this probe, so the observed fired:bypass count is 1:0; picker bypass behavior is covered by the operator's separate smoke, not inferred from that count. The temporary Zellij session was deleted after the check; the log remains at `/tmp/pair300-doctor.by3RuW/adapt-pairqdoctor300.jsonl` for this close review.
- M5 requirements now have evidence. The prior fresh `go test ./...` and `make -f Makefile.local test-native-terminal-ci` runs both exited 0, and the docs sweep is committed as 5501f23f. Proceed to the full-issue `sdlc close` gate; its review owns the remaining acceptance judgment.

### 2026-09-22 — Close review round 13 (REWORK, BR-50)

- `sdlc close --issue 300` measured 4.16 focused hours and dispatched the full-issue review. The gate returned REWORK and left the issue working on one Critical documentation finding: the durable plan's Core concepts table grouped IO scanner entry points and the `runQoder` subprocess under PURE (ARCH-PURE). The other seven carried findings were disposed as addressed; no new code behavior finding was raised.
- Swept every Pure table row for external reads, process launches, and retained state. Split scanner entry points from deterministic record transitions; left `runQoder` only in Integration points and `DefaultModel` in Pure; moved TTY profile registration and its mutable overlay detector to Integration points. Appended the plan Revisions note and a lesson. This is a classification correction; code and tests are unchanged. Re-run the close gate after committing the correction and the review artifacts.

### 2026-09-22 — Close review round 14 (REWORK, BR-51)

- The second `sdlc close` measured 4.29 focused hours and disposed BR-50, then returned REWORK for a Critical plan-location mismatch: Core concepts still pointed at the pre-capture `qoder/1.1.59/` directory and conflated user trust settings with the final repo-local command allowlist.
- Checked the Core concepts locations against delivered files and the final M4 settings decision. Corrected the table, Task 9's capture paths, and Task 15's config path; retained dated discovery notes as history. Added a Revisions entry and a reusable lesson. Production code and test results are unchanged; re-run the close gate.
