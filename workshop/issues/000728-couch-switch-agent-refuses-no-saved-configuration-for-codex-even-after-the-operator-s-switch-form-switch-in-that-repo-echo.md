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

- [x] M1 — Couch-owned repository arguments, bounded migration and explicitly accepted borrowed suggestions; Pair stays independent.
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


- 2026-10-22: closed M1 — BR-4 shares original5s migration/writer context; controlled real store/coordinator contention spends migration budget then refuses with warning, unchanged bytes, live child and no later retry. Sixteen concurrent writers preserved; actual wired publication leaves input/render responsive and remote completion pending. Full couchcore483.600s/couchcmd153.782s/couchtty79.923s/repolint25.054s PASS; targeted core race14.201s and console race PASS; artifact/dead-symbol/inventory PASS. Recorded affected started on pushede7de5ca161079b427e7fe4726d3782ac03d35b95; final-head recorded pass still required before landing. Unchanged PTY teardown flake registered separately as#814 from verified round2 same-head fail/pass.; review verdict: SHIP
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

### 2026-10-22 — M1 review round1 REWORK and corrective checkpoint

Pushed5a0fb9f504e8672b8491d6637b1828fe0161ee73. Recorded affected selected full suite; attempt7d207d47fb07708bd6f621878129fbff, snapshot /Volumes/TestScratch/local-checks/sdlc-local-1472033946 verified at that exact head. It remained queued, never started. M1 review ran concurrently under the permitted overlap rule, measured3.69h, and returned REWORK with BR-1 configuration-acceptance-provenance, BR-2 user-documentation-traceability and BR-3 regression-suite-migration. M1 did not finalize. The queued obsolete suite was cancelled through sdlc local-check cancel --run lease-161268321; retained runner47534 joined exit1 with suite lock interrupted/context canceled, not test evidence. No duplicate run remains.

Verified all findings: eight Claude/Codex own/borrowed/absent/conflict reducer/render cases fail specifically on missing labels before the fix. The active form now renders provenance/donor and explicit acceptance semantics, with compact rows preserving parameters, actions and borrowed donor under40x10 plus two banners. Cancellation/initial prefill dispatch nothing; final acceptance transports exact explicit argv/fingerprint. Complete couchtty package passes48.208s, targeted switch/cursor/copy-banner regressions pass0.322s. This brings the already-approved visible-state contract into M1 because borrowing is already active; the governing plan Revisions records it, without implementing M2/M3 controls.

Both reported old tests reproduced unchanged: stale-fingerprint returned repository-read ENOENT and nested repository expected a retired routed preference. Migrated fixtures to rememberRepoConfiguration and canonical document inspection; retained stale-preview no-effects refusal and independent nested/outer thread/archive/profile checks. Both pass0.608s. Complete couchcore package is still running; do not infer its result. README now describes delivered global Couch authority, marked borrowing/conflict, adoption/publication warning and repair, independent standalone Pair defaults and current CLI limitations. Atlas traces active provenance and clarifies the older path-memory historical note. Reader inventory and diff checks pass with no new certificate repair. Next: join complete couchcore, record actual result, push final head, queue one recorded affected check and rerun mandatory M1 review; retain M2/M3 and #743 fixture repair as outstanding.

Complete package run joined successfully: couchcore243.963s and couchtty50.187s; the final compact-layout/40x10+banners version separately passed complete couchtty48.208s. No failures. Corrective code is pushed at56084079f; replacement recorded check and M1 round2 remain pending. M1 is still open.

### 2026-10-22 — M1 round2: bounded optional publication

Round2 on32ababba112e3519b0009afb904b3df84acaf724 disposed BR-1/2/3, then returned REWORK on new BR-4 optional-console-work-bounds: competing writer admission blocks beyond the migration context. No M1 finalization. Replacement recorded affected again selected a full suite, attempt4513d331eb408414328a1c48f2b23abe, lease-4202133588, snapshot sdlc-local-4151263781; it never started. Cancelled through sdlc and joined runner55859 exit1/context canceled after this new required code correction. No recorded pass/failure is claimed for a cancelled queue wait.

Traced execution ownership, including a bounded independent read-only investigation: normal attachment/publication already runs on operationQueue lifecycle workers through prepareOperationCompletion; publishOperationCompletion settles remote callers afterward, and Run only reduces completed results. Initial dispatchInitialAttach is before Console.Run. The old failed-attach comment claiming console-goroutine execution was stale and is corrected. New actual wireResolver regression holds legacy discovery after successful attachment, proves input and acknowledged output/rendering continue, keeps remote completion pending, and verifies accepted argv is durable before the callback. Its only synthetic admission is returning the pre-spawned accepted child; inventory preview is disabled to isolate the publication barrier. Normal and race targeted runs passed.

New core regression acquires a real competing store/coordinator lock from a legacy-identity barrier after the preview lock is released. Both original cases failed beyond their admission bound (store0.34s/coordinator5.60s). Carry the original five-second context into writer admission and pre-publication checks. Reuse withUpdateLockContext, releasing coordinator authority between busy nested-store attempts; shared context-aware lock polling already used by launch diagnostics/orphan registry now lives under storelock. An intermediate immediate-busy refusal failed the existing sixteen-writer test; corrected it to preserve concurrent serialization within the original deadline, without changing that regression. No worker, queue or outbox was added. After terminal failure, shared sync.Once prevents a retry when locks are released.

Fresh combined targeted verification passes couchcore11.724s, couchcmd1.295s, artifactpath5.452s: contention consumes migration time first, so renewing the writer budget fails; warning/old bytes/child survival/no-retry, concurrent independent writers, actual input/render/receipt ordering, exhaustive artifact and dead-symbol checks pass. Related publication/deadline, launch diagnostics and orphan tests passed21.432s; reader inventory and diff checks pass. Full couchcore/couchcmd/couchtty/repolint command27394 is still running; retain its actual result before queuing/re-review.

Reviewer round2's broad couchcore run reported unchanged TestTeardownProvesARealPtyTree lease-holder probe deadline, then isolated pass; preserve that observation separately from the implementor's earlier complete243.963s pass. It is not a passing full-package reviewer result or recorded suite evidence. M2/M3, #743 fixture repair and live Echo acceptance remain outstanding.

Verified the reviewer transcript's actual failure497.705s/test2.66s at teardown_live_test.go:106 and isolated pass1.332s (same exact32ababba, no diff/status changes). Registered it on newly filed unclaimed pair#814 through sdlc flake register with that evidence; hub report fbdaccd4-ab62-422f-9389-679ed60c63d7 submitted. Authenticated hub3ddefa7d-eb74-4c38-b1b2-0405dd21cd86 notes possible #809 allocator ENXIO linkage; this captured failure has only list-lease-holders deadline, no PTY-start/ptmx/ENXIO signature, so no causal link is claimed. Bounded-publication correction is pushed at4a1d1bf76; complete package verification remains active.

Final corrected-code verification joined exit0: complete couchcore483.600s, couchcmd153.782s, couchtty79.923s and repolint25.054s. Targeted core contention/concurrent-writer/once/publication-warning tests also pass under race14.201s; actual console publication responsiveness passed normal and race in the bounded test investigation. No package failures in this final run. This replaces pending preflight, not the still-required recorded suite. Next push this evidence head, start one recorded affected check and M1 round3 concurrently; recorded final-head pass remains required before landing.

### 2026-10-22 — M1 boundary SHIP; recorded suite retained while repairing #743

Round3 returned SHIP, confidence medium, no new findings and BR-4 addressed. No open blockers after three rounds. SDLC finalized/ticked M1 and committed/pushed boundary f158214cfd8fe721c61ad02f75e91187f57a7950 with Review-Verdict: SHIP; measured M1 actual4.53h. The review independently restored blocking admission in a disposable checkout and both contention regressions failed. Reviewer full couchcore reached600s while retention/archive tests were active; selected active tests passed in isolation. Preserve that timeout as a limitation, not a green full reviewer package run; the implementor complete483.600s result remains separately recorded above.

Recorded affected selected full suite on pushede7de5ca161079b427e7fe4726d3782ac03d35b95: attempt d21fcedfda5b439e8007eef7615195fc, lease-1992946301, retained terminal22411, snapshot /Volumes/TestScratch/local-checks/sdlc-local-2256962163, raw inflight log under .git/sdlc/test-stats/inflight/d21fcedfda5b439e8007eef7615195fc/run.log. Still queued position15 at last observation, no test result yet. Verified snapshot HEAD equals e7de5ca1; finalized boundary changes only issue/plan/review evidence. Both review and check survived the Couch restart; review runner63666 has now joined exit0. Keep this suite runner/FIFO place and inspect its actual outcome; do not invoke continuation/restart while it is retained.

M1's review checkpoint is closed, but recorded verification remains outstanding before landing. Hub's existing direction now returns this slot checkout to #743 for its bounded fake-argv recording repair, without touching this snapshot or runner. No #743 process is currently retained. After that checkpoint return here for M2/M3; live Echo argument seeding and all edited-resume/interrupt/force acceptance remain outstanding. No live settings or :0 build changed. Go cache is the restored SSD default /Users/xianxu/Library/Caches/go-build, not TestScratch.

### 2026-10-22 — M2 resumes after bounded #743 repair

#743 fixture correction is pushed separately; its recorded full suite and manually dispatched hosted conformance remain pending in independent runners. #728 M1 recorded runner22411 remains retained, last FIFO14; no cancellation/restart or branch stacking. M2 follows the governing approved revisions: editable parameters, optional literal CLI argv/replay identity, core-proved live/vacant source, bounded cooperative interrupt and one-confirmation operator-only force; M3 owns exact-conversation edited resume and positive-missing fallback.

Raw Console input regression for Claude/Codex, held/refused preview, quotes/Unicode/cursor/backspace, late result and cancellation first failed all four cases because SwitchPrepared gated editing. Removed only the two input guards; existing generation invalidation remains the authority (ARCH-ORDER). All SwitchAgent-selected couchtty tests pass0.566s; new raw-console plus cursor/explicit-empty/stale-preview cases pass race2.286s. No switch/start effect is dispatched by editing or cancel. This is an M2 implementation checkpoint, not milestone completion or final recorded verification. Literal CLI/protocol, lifecycle guards/emergency actions, M3 and live acceptance remain outstanding.

### 2026-10-22 — M2 literal argument and receipt checkpoint

CLI tests first rejected explicit empty/nonempty --agent-args and --interrupt-first for both agents. Added optional comparable SwitchArguments with a private canonical literal-array representation; absent remains distinct from explicit empty, no pointer/null ambiguity or loss of Request comparability. New codec rejects null array/elements, nonstrings, invalid UTF-8/NUL, malformed/unknown/duplicate envelopes, and bounded transport/formatted argv. Strict existing fresh-selector validation remains shared. Added the two new fields to reflection-based closed-shape negative fixtures without weakening their other-operation rejection.

A stateful receipt test first proved all changed-argv/presence/interrupt intents were accepted as the same old request. Both existingSwitch and queue admission now use operationIntentDigest, preserving drive digests and rejecting changed switch intent before target probes. Real socket admission initially still queued known-config instead of explicit argv; now explicit values use PrepareAgentSwitch and preserve exact argv/fingerprint in the queued call, while absence uses PrepareKnownAgentSwitch. Both Claude/Codex empty/nonempty cases pass.

Internal dispatcher tests independently reproduced null elements becoming empty strings for both agents. Reused the existing strict persisted-array decoder as pure DecodeArgumentArray for preferences, wire carrier and internal dispatch (ARCH-DRY/SECURE), rather than retaining two grammars. Targeted Switch/codec/drive verification passed couchcore12.672s, couchmessage0.236s and couchcmd16.486s. Final focused race passes core1.998s/message1.285s/cmd1.807s; complete message package25.093s; repolint23.396s; required inventory and whitespace checks pass without refreshing certificates.

This checkpoint implements literal transport and replay identity. Interrupt-first is parsed/bound in the request but cooperative execution is still pending; existing busy/unknown/dirty checks continue to refuse. Core-proved vacant targets, source evidence, cooperative interrupt, operator-only force, same-agent intent/M3 edited resume, final docs/live acceptance and boundary reviews remain outstanding. No live slot or settings changed. #743 hosted exact-head conformance39854919087 passed on1afd8bd6732bcb276137216f58066f583efc8ff4; submitted hub reportb92823ec-a045-4605-8bec-7f9348d8a61c. Both recorded suites remain retained, latest #728 FIFO14/#743 FIFO16.

### 2026-10-22 — M2 core-owned source evidence checkpoint

Actual parked-Couch regression first failed because the service demanded a connected wrapper after core admitted vacancy. PreparedAgentSwitch now exposes private-field SwitchSourceEvidence from its existing classification; its mode/reason joins the accepted fingerprint. Serialization cannot recreate vacancy authority. The source guard consumes that evidence, verifies retained address/revision/path, resolves the actual workspace and checks Git, then repeats retained-record/reconnect checks. Missing proof and an old live token remain on the live/refusal path; no disconnected-to-vacant inference or new process census.

SwitchAgent prepares the typed mode before its first guard and re-prepares under the existing lifecycle hold, preserving guard-mutation, stale fingerprint, exact park and start claims (ARCH-ORDER/SECURE). Cost is two fixed preparation passes when a service guard is installed, one otherwise; no per-entry extra census. Service admission also prepares before consuming the guard evidence. Canonical remote parked-target selection and UI parked offers are not wired yet.

Targeted SwitchAgent/guard/parked-producer/effect-ownership tests pass core12.641s/cmd12.491s. Final race covers actual core proof, nonserializability, dirty/unknown Git, missing proof, old token, workspace/unreadable record, replacement before/during Git, reconnect, cancellation, source death/change and effect-time hub ownership: core3.149s/cmd6.145s. Repolint25.418s, inventory and whitespace pass. This is an implementation checkpoint, not M2 close. Next wire passive canonical retained-slot resolution through the same preparation/queue/guard owner, then cooperative interrupt and operator-only force; M3 remains separate. Both recorded runners retained.

### 2026-10-22 — M2 retained target and parked form checkpoint

ResolveAgentSwitchTarget now uses verified WorkspaceReferencePath/slotWorkspace, a context-bound read-only snapshot and the existing projection/SelectSlotRow address rule. It selects retained metadata only; PrepareAgentSwitch remains the lifecycle/vacancy owner. Primary and numbered/alias fixtures pin no-thread/unreadable/ambiguous outcomes, root-only primary selection, invalid identities, cancellation, unchanged global/local bytes and pending journals. Review caught a lossy partial-snapshot case: a readable record plus its backend error was projected without the error. The added behavioral regression failed, then matching slot errors were explicitly retained as a refusal (ARCH-ORDER/SECURE).

Real socket regression first refused a parked repo:N for no wrapper. The handler now keeps its existing connected-source route and uses passive canonical selection only for ErrUnavailable; ambiguous live targets and discovery errors still refuse. The fallback rechecks canonical selection, own-slot exclusion, verified hub ownership and the evidence guard at queued preparation/effect. Stateful fixtures pin foreign hub, newly ambiguous retained selection, dirty Git, queued ownership move and effect-time ownership move. One initial test fixture incorrectly copied old-address park history into a replacement record; corrected that new fixture before final verification. No product refusal was bypassed.

The UI now derives eligible parked/binding-loss/session-gone offers from core SwitchableState, excluding orphan/open-park/unfinished-continuation rows; detached and unknown sources remain excluded. Existing action-table expectations were revised before the implementation and failed as expected. An editable switch draft survives a newly parked source but still requires a fresh accepted preview. Core source proof additionally pins private retained-record fields rather than mutable public preview mirrors; a new public-field mutation test failed before that tightening, and serialization still cannot confer vacancy authority.

Final combined race verification passes core8.818s/cmd4.724s/tty1.801s. The broader UI advice/action sweep separately passed race152.443s; its combined process originally exited1 only on the now-corrected new replacement fixture, and the final command above verifies that case. Repolint13.818s, inventory and whitespace pass without certificate refresh. All targeted commands joined; no boundary review is active. Cooperative interrupt execution, operator-only emergency force, M3 edited resume/fallback, final documentation/live acceptance and M2/M3 reviews remain pending. Keep recorded runners22411 (#728 M1) and45648 (#743 repair); latest positions15/17, no results yet.
