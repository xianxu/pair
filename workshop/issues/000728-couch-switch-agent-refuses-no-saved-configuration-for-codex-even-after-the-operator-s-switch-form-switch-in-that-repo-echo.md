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

### 2026-10-21 10:47 — Operator-only unknown-idle force escape approved

Authenticated submitted hub e9c3a9af-422f-4c4e-acfc-ac4e21f612eb approves M2 interrupt-first but notes usage-limit/stuck-turn evidence can be neither working nor idle. Appended governing Revisions: form/console-only Force switch, distinct second confirmation naming loss of the current turn's unsaved progress, source/fingerprint-bound typed local consent, no CLI/hub force path, normal verified ownership/Git/park/cleanup/launch gates retained. Added usage-limit fixture and remote/replay/cancel/adoption acceptance. This closes the product gap without speculative interrupt bytes or a second lifecycle mechanism.

### 2026-10-21 10:53 — Emergency force simplified by the operator

Authenticated submitted hub c6e88045-8036-43b4-abfd-ca18d04d9132 supersedes the earlier two-confirmation unknown-only escape: emergency form/console force has one explicit loss confirmation, stops the stuck agent hard via the existing park owner, then launches the target; no CLI/hub force, no standby hub. Appended the governing Revisions. Read-only lifecycle trace is checking existing hard-stop escalation for reuse. Asked the hub whether force also permits an owned dirty tree or retains the clean-tree refusal; M1 remains independent and proceeds.

### 2026-10-21 10:55 — Dirty-tree force decision and lifecycle reuse

Authenticated submitted hub78fa0734-c341-4e9f-9ec2-9f83b018ebd7 permits owned dirty emergency force; confirmation names N uncommitted files retained for the new agent. Normal switch stays clean-only; no file-changing command is added. Read-only trace confirms normal park already hard-stops the exact session server and wrapper-owned native group (existing SIGTERM1s/SIGKILL), with matching completion/exact helper death required before startup. Governing Revisions now resolves the question and reuses this owner rather than adding another signal path. Real usage-limit fixture and dirty-byte preservation remain M2 acceptance.

### 2026-10-21 — M1 consumer/adoption checkpoint before #743 landing

Implemented bounded read-only Couch legacy collection with actual Git identity/slot containment, aggregate4MiB and256-entry/source refusal, exact source-byte retention and imported-source skipping. Removed enrollment's legacy preference transfer/deletion and all Couch callbacks/readers of Pair defaults. Ordinary starts, managed/fresh slots, recovery preflight, switch preparation and StartResolution now resolve Couch repo arguments; marked borrowing is a service value requiring explicit acceptance, never an implicit CLI launch. Explicit accepted args can proceed when settings are unreadable; final publication warns. Source dependency guard seeds forbidden Pair-reader calls and scans Couch's three production packages. Pair's launcher/store code is unchanged.

Registration now persists exact execution history plus manifest only. A durable explicit remember_configuration bit is independent of resume/fresh shape; launch captures an opaque shared acceptance after successful registry insertion, and the actual shared console attach dispatcher publishes once only on successful attachment. Failure preserves the adopted child with a visible warning carried through StartResult, SwitchAgentResult and noted receipt wrappers. Automatic/warm/recovery paths do not learn. The publication migration has one5s total identity-lookup deadline; capability JSON is excluded, with no durable retry/outbox. Real3-slot/reopen/native-resume/switch/fresh tests prove shared per-agent defaults, historical resume retaining old flags after another slot edits the repo default, and no new folder preference files. Managed creation exposed dropped explicit args during revalidation; StartResolution now preserves/clones requested argv presence/content and revalidates it exactly.

Verification: combined source/resolution/switch/store/migration/recovery/fresh targeted group passed8.362s; full strengthened real-slot integration race passed13.155s; nine drift/resume/separate-Git/switch cases race passed17.420s; adoption/codec tests race passed7.132s; actual console attach plus related startup/wiring tests passed3.347s; affected inventory passes after six narrowly audited reader updates, artifact classification passes3.751s. A larger combined targeted race command is still completing before branch switch; record its actual result rather than infer from partial output. No recorded affected/full suite or M1 boundary review has run on this checkpoint. M1 remains open, M2/M3 unimplemented. Atlas now maps the new repository authority/publication flow.

Hub e085728b-8155-4cb3-8bd0-adeea1303c6b reports operator local-gates-verified now on #743 PR515 and directs landing then the real named conformance-vm check on pinned main; checkpoint #728 clean first, interleave #743 next, then return here for recorded verification and M1 review. Latest M2 emergency force contract is governing10:53/10:55 Revisions; no unanswered product question remains.

- Combined targeted race finished successfully before switching: couchcore30.418s, couchcmd3.136s, threadrecord1.180s for TestRepoArgument/Preference/Configuration, TestRepository, negative preference dependency and remember-claim/codec groups. No recorded check is queued/running. M1 remains unreviewed and open.

### 2026-10-21 — M1 selector-boundary follow-up during the #743 capacity wait

#743 merge remains attached in the original checkout, queued behind #744's integration-capacity wait. A private TestScratch checkout of this existing #728 branch allowed bounded targeted verification without changing that checkout/runner or queuing another full suite. Self-audit reproduced four ordinary-start cases (Claude/Codex × explicit/legacy args) accepting native resume selectors as startup flags, and managed create accepted them too. New-thread spawn now calls the existing ValidateFreshAgentArgs before allocation; managed create does the same before provisioning. Migration/preview still retains old arguments for editing, and dedicated native resume keeps historical invocation authority. Pair code is unchanged.

Red proof used real private Git fixtures and the normal initialized store lock: a raw-only preference fixture initially refused on missing store.lock, which was an unrelated false green and was corrected before observing all four actual selector failures. Follow-up targeted TestRepositoryArguments race passes3.620s, historical resume/shared real-slot/dependency guard race passes13.814s, affected inventory and diff checks pass. M1 full recorded verification and mandatory review remain pending; this is no completion claim. No new full-suite runner or #743 priority change.

### 2026-10-22 — Resume M1 on its own branch

Authenticated hub63707f21-69c6-4fb3-a207-a4fb6291288b and60aa7f36-8fbe-44b3-a052-636461616204 prioritize#728 M1 now, with#743 repair at its M1 checkpoint. #743 recorded suite has finished, failed only its own native fake-argv interleaving assertion; classification/full raw log hash are committed on its separate branch. No suite, merge or test process remains for#743 and its private snapshot was cleaned, so switching is safe. No stacking on unlanded#743.

Integrated origin/main4a7ab2ca1a4c0b0d4aa34e8cada1d8c2917427bc here. Atlas retains architecture ownership wording plus landed endpoint/mock/reply links; retired aggregate reader inventory is removed. Audited exactly six#728 readers: run_test.go reads README under registered guards plus private trace fixtures; preference_boundary_test.go scans Couch production source under registered TestCouchPreferencesNeverReadPairDefaults; couch_test.go reads only private owned process/session/registry fixtures; recovery_preflight_test.go traverses private ThreadStore and publication files; resume_launch_test.go checks private seeded historical/resume state; threadstore_test.go reads private journal/legacy/configuration bytes. Recorded only those six per-reader certificates through the real CLI, retaining other metadata. Required inventory passes. Integrated targeted M1 verification is pending.

Deferred hub input be776001-341e-4b2a-a9bf-6d478f124a21 is now active M1 acceptance: reported echo:1 per-slot config edit was overwritten by stale running-session args. Add a production-path regression that canonical Couch repo edits survive later session persistence and per-slot config cannot override them; Pair remains unchanged. Live acceptance will seed Echo Codex [--sandbox,danger-full-access] and Claude [--dangerously-skip-permissions] through the Couch store, not manual per-slot edits. No live settings changed.

Earlier private-copy deviation is preserved: commit5f989be76 was authored/pushed in a private TestScratch copy before hub11bf06ac-816e-4562-a91e-1d4955d3597f prohibited edits/commits/SDLC there; disclosed in submitted9d1a1d6f-c9a8-4e7a-8892-b0f613419172 and removed copy/empty parent. All work now uses this slot checkout. Private copies, if needed, are read-only targeted verification only and removed afterward.

Integrated targeted run completed: couchcore28.864s and couchcmd1.777s passed TestRepoArgument/Preference/Configuration, TestRepository, isolation guard and TestResumeLaunchExactProfileMatrix; couchtty had no matching cases in this selection. These are targeted results only, not recorded M1 boundary verification.

### 2026-10-22 — Authority regression exposes trusted-profile picker gap

New Claude/Codex registration regression passes and retains canonical repo bytes plus old historical invocation; removing acceptedConfigurationForStart explicit RememberConfiguration guard makes both cases fail at the exact stale-session-overwrite assertion, then source restored. New RunLaunch regression fails both agents because an ordinary validated Couch profile still opens the per-slot picker. Standalone inverse control is added before the fix. Submitted approval52a2c824-6b62-435e-bca3-b99a1203f316 permits only the two trusted-profile adapter guards, keeping standalone Pair policy/store behavior unchanged; plan Revisions records the wording correction. No live settings changed.

### 2026-10-22 — M1 trusted-profile correction verified

Implemented only the approved two runCreate guards keyed by validated AgentArgsFromCouch: skip standalone default lookup and saved-config picker. Atlas documents the boundary; no store/environment heuristic is added. Standalone inverse fixture uses no native binding, so cancellation proves picker/default behavior without triggering unrelated stale-session cleanup. Both Claude/Codex Couch-launch tests and standalone inverse controls pass; canonical repo-edit/session-registration tests pass for both agents. Earlier negative control removing explicit RememberConfiguration admission made both stale-session regressions fail, then restored source.

Fresh verification: full launcher package passes149.542s. Combined targeted race passes couchcore62.569s/couchcmd5.448s for repository preference/model/migration/adoption/resolution/shared-slot and historical-resume groups (couchtty had no matching tests). Explicit actual console adoption plus initial attach/abort/warm-pass/start wiring selection passes couchtty0.410s/couchcmd3.553s. Required affected-reader inventory passes with no new certificate repairs. Recorded affected check and mandatory M1 boundary review are next; M1 remains open and M2/M3 remain unimplemented. No live settings changed.

Broader preflight caught four APIs now referenced only by tests after this migration: RecordSuccessfulLaunch, GetPathLaunchPreference, PreviewPathLaunchPreference and previewRepoPreferences. Traced all callers; production migration reads legacy bytes through its bounded collector, while these helpers only construct/inspect private legacy fixtures or inspect canonical test state. Moved them and their private legacy getter unchanged into repopreferences_fixture_support_test.go, preserving historical regression coverage without shipping a second reader/writer or extending the dead-symbol allowlist (ARCH-PURPOSE). Full repolint passed25.106s; dead-symbol and exhaustive production-artifact classification now pass5.524s. Reader inventory remains passing; no new source reader was introduced.

Post-move targeted repository/legacy preference/store/preview/shared-slot regressions pass couchcore80.678s. The helpers are test support only; production model/codec and bounded migration remain in place.
