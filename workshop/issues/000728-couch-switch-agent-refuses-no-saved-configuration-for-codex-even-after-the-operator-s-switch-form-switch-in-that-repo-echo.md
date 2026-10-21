---
id: 000728
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours: 14.58
collaboration-mode: collaboration
card_mirror: '087579056e70ca221fbcd3d8626feabeada1fb57' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T23:31:35-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# couch --switch-agent refuses 'no saved configuration for codex' even after the operator's switch-form switch in that repo (echo)

## Problem

ops-echo:0 (10-20 ~09:20): `couch --switch-agent echo:2 --agent codex --confirm` refuses with "no saved configuration
for codex; the operator must configure it in the switch form first" (requests 8e69e8d8 and 0ad9d2c3), and still
refused after the operator switched echo:0 to Codex by hand in the switch form. So the form's switch doesn't save the
per-repo configuration the CLI reads (or saves it under a key the CLI doesn't look up). In the devinfra repos the
same CLI switch worked (ariadne:0, ariadne:5, parley.nvim:3, :4 on 10-20), so a saved codex configuration exists there.

## Spec

Find where the switch form records its agent configuration and where `--switch-agent` reads it; make a successful
form switch leave the configuration the CLI needs (same key, same repo scope), or have the CLI fall back to the
agent's last successful launch configuration in that repo. The refusal names exactly what's missing and where.

## Done when

- A form-driven switch in a repo without prior CLI config, followed by a CLI switch on another slot of that repo,
  succeeds (test against the stateful fake/store).
- echo:2 and echo:3 switch to Codex by CLI live.

## Plan

The latest governing Revisions and [durable plan](../plans/000728-shared-agent-startup-preferences-plan.md#2026-10-21-0004--governing-couch-only-repository-preference-design) supersede the historical shared-store proposal. The operator approved the governing Couch-only plan and edited-resume/fresh-fallback revision on 2026-10-21 08:54; entering change-code before implementation.

- [ ] M1 — Couch-owned repository arguments, bounded migration and explicitly accepted borrowed suggestions; Pair stays independent.
- [ ] M2 — Editable field/provenance, explicit CLI arguments and guarded live/proven-vacant switching, with typed restart intent.
- [ ] M3 — Edited same-native-conversation restart, visible missing-conversation fresh fallback and real acceptance.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. estimate-source marks the calibration stale; values are provisional.*

Primitive mapping, in block order: approved issue/spec; repository argument decisions; nested raw codec; global store/concurrency; legacy fold/conflict/provenance; all Couch consumer migration; final console adoption/publication and warnings; editable form; typed CLI/socket/receipt intent; live/proven-vacant guard integration; pure edited-restart decision; pinned cold-resume/fallback lifecycle integration; combined warning projection; README/help/atlas; Claude and Codex native conformance discovery; M1/M2/M3 and whole-issue reviews. Stateful fake/regression work belongs to its component, not a separate undifferentiated test buffer.

Library check: existing strictjson codecs/MergeRaw and ThreadStore atomic writer/locks halve design for the first two greenfield model/codec primitives (1.5h ×0.5 library ×0.2 thorough-spec =0.15h each). Migration/conflict and pure fallback rules are new domain policy without a substituting library (1.5h ×0.2 =0.30h each). Other design anchors are2–3h for the integration seams,2h for the TUI,1h for cross-cutting consumers,0.3h for the smaller warning extension and0.2h for docs, all ×0.2 thorough-plan discount. Four review design anchors are0.1h ×0.2. No new third-party dependency or stack is needed.

Each impl value is its v2/v2.1 anchor ×0.40, including two0.6h native discovery budgets and four0.5h review budgets. Familiarity1.5 reflects bounded but unfamiliar global-store routing, late attachment adoption and native resume failure authority; no additional implementation buffer or cross-repo factor. Design4.38h ×1.15 =5.037h; scaled impl6.36h ×1.5 =9.54h; total14.577h, recorded14.58h. A reasonable provisional range is10–18 focused hours; queue waits, operator label/smoke waits and the separate #743 work are excluded. The earlier2–4h marginal edited-resume comparison is not this full scope's total.

```estimate
model: estimate-logic-v3.1
familiarity: 1.5
item: issue-spec design=0.30 impl=0.12
item: greenfield-go-module design=0.15 impl=0.32
item: greenfield-go-module design=0.15 impl=0.32
item: api-integration design=0.40 impl=0.60
item: greenfield-go-module design=0.30 impl=0.32
item: cross-cutting-refactor design=0.20 impl=0.20
item: api-integration design=0.60 impl=0.60
item: tui-screen design=0.40 impl=0.40
item: api-integration design=0.40 impl=0.40
item: api-integration design=0.40 impl=0.60
item: greenfield-go-module design=0.30 impl=0.32
item: api-integration design=0.60 impl=0.60
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: real-api-discovery design=0.00 impl=0.24
item: real-api-discovery design=0.00 impl=0.24
item: milestone-review design=0.02 impl=0.20
item: milestone-review design=0.02 impl=0.20
item: milestone-review design=0.02 impl=0.20
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 14.58
```

## Log

### 2026-10-20

## Log

- 10-20 10:05 (TL): same refusal in a devinfra repo: `couch --switch-agent ariadne:4 --agent codex --confirm` →
  "no saved configuration for codex; the operator must configure it in the switch form first", although the same
  command succeeded on ariadne:0 and ariadne:5 an hour earlier. So the saved configuration looks per slot (or per
  launch history), not per repo: a slot that never ran Codex has none.
- 10-20 14:15 (TL): third case: the operator's switch-form switch of pair:8 (Qoder → Codex) failed with
  "switch-agent: source changed or disconnected; retry switch" (launch-diagnostics failures.json, operation
  2e048ac2508c98354822c8fa342aa0e3). pair:8's Qoder session was a recording only (no live wrapper). The switch should
  handle a parked or disconnected source (start fresh on the new agent) instead of refusing.
- 10-20 21:15 (TL, operator report): there is no way to change a running Codex slot's startup parameters and restart
  it. Choosing the switch form with the **same** agent shows a parameters box, but the box **accepts no keyboard
  input**. Scope for this issue now: (1) the parameters box in the switch form takes input; (2) "restart with new
  parameters" for the same agent is a supported path (form and CLI: `couch --switch-agent repo:N --agent codex
  --agent-args '…' --confirm` on the current agent), saved as that agent's remembered arguments; (3) the earlier cases
  (no saved configuration; a parked or disconnected source). Operator-facing; the operator hits it now.
- 10-20 22:35 (TL): root cause of the operator's latest report ("switching to codex in ops didn't use the recorded
  parameters"): remembered arguments are keyed by **exact physical path**. The ops primary checkout's preference file
  (`threadstore/path-preferences/b96c0dde….json`, physical_path /Users/xianxu/workspace/ops) records codex
  `["--sandbox","danger-full-access"]`, but slot worktrees (e.g. /Users/xianxu/workspace/worktree/ops-slot1/ops) have
  **no preference file at all**, so a switch or fresh start there falls back to the agent's defaults. Fix to include
  in this issue: when a slot path has no remembered arguments for an agent, fall back to the same repository's
  (repo_identity) remembered arguments, then save them for the slot on a successful launch; show which source was used
  in the switch form.
- 10-20 22:45 (TL): the operator ran `pair codex -- --sandbox danger-full-access` in the ops-slot1 worktree and the
  switch still ignored it: Pair's remembered-arguments store and Couch's `threadstore/path-preferences/` are separate,
  and Couch never got an entry for that folder. The fix must give remembered arguments one source of truth shared by
  Pair and Couch (plus the repo-level fallback for slot folders above).

## Revisions

### 2026-10-20 23:36 — Authenticated ops:0 dispatch; four-report scope

Delivery f55d328f-fc86-4a9a-8310-da4caa858833 is authenticated submitted from the owning hub. #743 is parked pending ariadne#509; this workspace now owns #728 on its own main-based branch. In addition to the original reports, unify Pair's repository defaults and Couch's path preferences under one current-preference authority with repository fallback for slot folders. Fix the form field, support same-agent fresh restart and explicit `--agent-args` through the CLI, and admit parked/disconnected sources only when existing core classification proves vacancy. Unknown or surviving native ownership must still refuse. Operator-facing plan goes to hub before code.

Proposed design: [shared startup preferences plan](../plans/000728-shared-agent-startup-preferences-plan.md). Two real review boundaries: M1 shared preference authority/migration; M2 form, CLI and guarded switch/restart with live acceptance. Governing completion additionally requires explicit empty/nonempty argv parity, shared settings from one slot visible to another/Pair, no learning on failed launch or recovery/warm attach, safe refusal races and operator-approved live checks. The plan explicitly requests approval for same-agent fresh-conversation semantics and started-with-warning preference publication after local registration; neither is assumed approved yet. Original reports/Spec remain as historical shaping context.

- [ ] M1 — Shared preference authority and bounded compatibility migration, with all current-default consumers moved and historical invocation evidence preserved.
- [ ] M2 — Editable form, explicit CLI args and safe live/vacant same/different-agent flow; required checks and hub-selected live acceptance before landing.

### Investigation checkpoint

Read-only traces identify launcher/createflow.go skipping default persistence for every Couch-owned/fresh launch; threadstore.go advances only exact-path preferences; switchagent.go reads that path plus a separate primary-folder default. Thus form success cannot seed another slot. switchServiceGuard requires a connected wrapper before core SwitchableState, excluding already-supported proven vacancy. Same-agent preparation explicitly errors; form typing requires SwitchPrepared, coupling preview refusal to a dead field. CLI/protocol omit explicit argv intent. No implementation edits or live switches.

The proposed leaf launchprefs component avoids launcher/couchcore import cycles. Local registration already persists the exact executed profile, and both successful-registration sites can publish preferences afterward via the existing warning channels. A publication error must never enter launch failure handling, which would otherwise stop a successfully registered helper. Plan makes the post-registration crash/write-failure limit visible rather than inventing a cross-root journal. One advisory proposal audit is running before submission; mandatory plan-quality review remains change-code's gate after hub approval.

- Proposal advisory audit completed once, with three refinements accepted in the durable plan's Revisions: distinguish provisional imported fallbacks from confirmed values and handle both discovery orders/same-path conflicts; preserve PreviewPathLaunchPreference's read-only/no-initialization behavior; publish only after final successful actor adoption, not merely StartRegistered, and explicitly bridge Pair defaultReady and SwitchAgentResult's Warning-field/interface mismatch into visible succeeded-with-warning receipts. No source edits. Plan-quality remains the binary gate after hub approval.

- Authenticated hub5f79b09b-db98-4a2c-a060-bb12435c8827 holds code while operator reviews the UI. Requested incremental estimate for same-agent edited same-conversation resume plus explicit Fresh option. Read-only trace finds reusable trusted required-ID/cold-resume transport, but switch currently hardcodes Fresh and resume profile marks sources saved; user edit intent must remain separate from selector authority and automatic-resume learning. Durable plan option analysis derives roughly2–4 additional focused hours (central2.5, provisional stale v3.1 calibration), recommends a separate edited-resume review boundary and actual Claude/Codex native acceptance. This is a marginal comparison only, not a card estimate or approved scope change.

### 2026-10-21 — Governing operator layering revision

Authenticated ops:0 directions64b4b0ac-bf04-425e-857e-8ee9a9c2ef1e andbd0eb7a0-818e-45d5-8436-f4f951ab5183 replace the earlier shared-store proposal. Pair stays unchanged as a thin standalone per-folder launcher; it reads no Couch preferences. Couch owns repository-wide remembered agent args, folds its own path-preferences once and passes accepted args explicitly to Pair. Missing repository config may offer a marked/pre-filled value from another Couch repository on the machine; acceptance saves it locally, cancellation does not. No Pair store migration/read/write changes. Working field, --agent-args and safe proven-vacant switching remain. Started-but-not-remembered warning stands; Resume/Fresh restart row remains with the operator and code is held.

The durable plan's 2026-10-21 00:04 governing revision completely replaces shared-authority M1 and documents entities, all Couch consumers, migration conflicts, read-only previews, marked borrowing, bounds/state/recovery/test strategy and the open noninteractive-CLI borrowing interpretation. Architecture rule is made durable in atlas/architecture.md and AGENTS.local.md as requested. No implementation source edits or live switches.

#### Governing completion contract for the revised scope

- Pair's standalone configuration/store behavior is unchanged and consumes no Couch preferences; the architecture boundary is durable in atlas/local rules and covered by a focused isolation guard.
- A successful explicit Couch configuration for one slot is reused by every slot of that verified repository, including CLI switch/restart; another agent's argv never leaks across harnesses.
- An absent configuration offers a marked borrowed value only from another repository's configuration for the selected agent; cancel does not save, explicit acceptance followed by successful adoption makes it this repo's own value. Empty and absent remain distinct.
- Existing Couch path-preferences fold once without touching Pair data; conflicts remain explicit, migration is discovery-order independent, read-only previews/cancel do not mutate storage, and unknown fields/other records survive bounded concurrent updates.
- Parameter typing/editing works through the real console path despite loading/refused previews, with exact quoted/Unicode/empty argv preserved. CLI --agent-args reaches the same guarded operation and receipt replay rejects changed intent.
- Same-agent restart follows the operator's still-pending Resume/Fresh choice. Proven-vacant sources work; live/unknown/foreign/busy/dirty/replaced sources retain appropriate refusal and recovery, with no duplicate launch.
- Preferences learn only after final adopted success; failed adoption preserves old values. Publication failure keeps the agent running and visibly warns that arguments were not remembered, including remote succeeded-with-warning receipts.
- Required recorded checks, mandatory boundary reviews and hub-authorized real operator acceptance (original echo:2/echo:3 targets if still selected and safe, otherwise hub-selected replacements) precede landing.

### 2026-10-21 08:54 — Operator approval and final restart contract

Authenticated submitted hub76a8370a-46db-4150-a6ee-f0362d42e550 approves the revised Couch-only design and resolves Restart: preserve the same native conversation with edited flags; when no previous conversation is found, start fresh with those flags and report "no previous conversation found; started fresh with the new parameters". Started-but-not-remembered behavior remains accepted. Plan Revisions now includes explicit absence versus unrelated failure/unknown ownership, final adoption learning, exact combined warning transport, native-ID/race/fallback tests and the extra M3 edited-resume review boundary. No Pair preference/store/default changes, no new Fresh selector, no new resume engine. #743 is parked codecomplete for its operator label; its final code suite passed, so it needs no restarted runner. Next is change-code plan-quality, scoped estimate after quality passes, then M1 TDD.

- Plan-quality round1: PQ-1 Important asks for an enforced pure restart-to-fallback transition; Minor asks for function-level strategies rather than case inventories. Added named AdvanceEditedRestart table, exact binder/failed-attempt evidence producers, claim/lifecycle reuse and interruption behavior. Read-only M1 trace also identified final console attach after registry insertion; publication now explicitly follows successful shared attach and warning payload survives StartedChild projection. No code edits; M1 store and M3 decision remain held for plan-quality.

- Plan-quality round2 CLEAN: PQ-1/PQ-2 addressed, no blockers. Derived full scoped estimate only afterward using the named stale v3.1 calibration;14.58h includes all three milestones, four boundaries, stateful tests/native qualification and final adoption handling. Card setter and estimate-quality gate follow; implementation still awaits admission.

### 2026-10-21 — M1 first implementation checkpoint (not milestone completion)

change-code admitted implementation at353ba806f after plan-quality CLEAN/PQ-1+PQ-2 addressed and estimate-quality INFO. Implemented the pure repository model and raw-preserving codec in repopreferences.go/repopreferences_codec.go, plus global ThreadStore read/publication seam in repopreferences_store.go. One document joins slots, preview initializes nothing, concurrent independent store instances preserve other repositories/agents, and publication failure preserves old bytes. Source snapshots preserve original UTF-8 bytes/newlines via a JSON string payload plus digest; unknown envelope/repo/agent/provenance fields remain raw. Red regressions covered missing APIs, newline digest loss, null argv elements and malformed UTF-8 read/write; all now pass. Added counted borrowing scan at8/16 repositories, exactly one record visit each.

Targeted complete TestRepoArgument/TestRepoPreference group passes under race in2.627s; affected-reader inventory and exact production artifact classification pass1.837s. Audited new production files as Couch namespace control metadata/pure policy, not Pair per-tag artifacts. This is an internal unit: no current-default consumers migrated, no legacy-source scanner, no publication after console adoption, no UI/CLI or restart behavior changed yet. M1 remains open. Next implement bounded Couch-only legacy collection/fold through enrolled verified sources without changing legacy bytes, then replace Couch preference consumers and final-adoption learning; Pair remains unchanged. No full recorded check queued yet.

### 2026-10-21 10:44 — Operator's interrupt-then-switch scope addition

Authenticated submitted hub c304f9f9-aca2-4d34-a7c2-fcbbf0415f96 adds explicit confirmed interrupt-then-switch for busy/unknown-idle sources. Read the actual diagnostics entry operation-0547363bcf95d1afe9d438f9da750c80: hub thread 1-ops-71 refused known-idle admission at09:18:38, before any helper started. Appended governing plan Revisions with existing #742 interrupt reuse, distinct confirmed CLI/form intent, one30s total bound, at most one interrupt, fresh idle proof, same-source/cleanliness guards, precise failure diagnostics and state/acceptance table. Unknown working-turn proof never authorizes speculative Ctrl+C. This is approved M2 scope; M1 work continues, M3 retained.
