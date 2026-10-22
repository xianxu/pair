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
- [x] M2 — Editable field/provenance, explicit CLI arguments and guarded live/proven-vacant switching, with typed restart intent.
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



- 2026-10-22: closed M2 — BR5: live guards return verified source token retained across ordinary/cooperative phases; controlled local/remote reconnect regression red then green with zero park/start and unchanged source. BR6: cancellation copies preview; both Escape/Cancel immutability and concurrent render races red then green. Corrected admission fixture installs authority before startup and uses synchronized fault state. Final broader race core2.793s/cmd10.941s/tty1.439s; fixture race count3 PASS4.433s; repolint6.711s and affected inventory PASS. Prior clean native Claude full argv+parked and both force cases pass; Codex third attach false-miss accepted environmental by authenticated hub54a9ef63 and filed#821, raw cb0b2cf75c161c0ba3afa92f2ff8a197d136fe19e03865dd57a15fbebd4b1418, no retry. M1 recorded suite PASS11m39s on e7de5ca1; final-head recorded and M3 operator acceptance remain required before landing.; review verdict: SHIP
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

### 2026-10-22 — M2 cooperative interrupt transaction checkpoint

The socket regression first refused explicit cooperative consent at the ordinary idle gate. Admission now waives only idle for interrupt-first, preserves connected source/authority/workspace/clean Git proof, and queues the consent without sending input on request workers. SwitchAgent acquires its existing lifecycle hold before the optional cooperative phase, re-prepares the accepted fingerprint afterward, then retains the normal final idle/Git guard and park/start owners (ARCH-DRY/ORDER). Held, unsupported, changed-generation and refused phases never park or launch.

The switch service reuses executeAgentDrive with the existing authenticated Interrupt request and semantic instance/sequence. One shared thirty-second phase deadline covers admission, at most one input, and 100ms fresh-idle observations. Already-idle sources send no input. Unknown admission sends none and preserves its exact diagnostic; uncertain reply, cancellation, dirty/source/record change and confirmed interruption without fresh idle all refuse replacement. No retry, raw key, signal owner or durable state is added.

Claude/Codex stateful drive tests and core lifecycle-hold/dispatcher tests pass under race: core3.753s/cmd14.751s. Existing drive interrupt/deadline/caller-loss/queued-change/hub-ownership/lost-admission race cases separately pass1.836s. Repolint21.748s and reader inventory pass. The exhaustive artifact-source preflight caught four omitted new helpers from this and earlier M2 checkpoints; audited their codec/owner-delegation inputs and registered them as non-artifact sources rather than quarantining violations. Final artifact and inventory results are recorded with the commit verification.

The form still needs its cooperative action, and operator-only force, M3 edited resume/fallback, documentation/live acceptance and mandatory M2/M3 reviews remain outstanding. Both recorded checks survive restart: #728 M1 runner22411 and #743 repaired runner45648 remain queued; no recorded pass is claimed. Hub recovery report5f75df14-ddc5-4b88-91ee-5bfdb8ac0b2f is submitted.

Final audited artifact-source regression passes3.023s; repeated reader inventory and whitespace checks pass. No certificate hashes needed repair.

### 2026-10-22 — M2 cooperative form intent checkpoint

The new Claude/Codex form regression first failed for the missing Interrupt then switch action. The compact parameter form now offers it beside Switch without adding a body row, retaining Cancel, provenance and parameter cursor at40x10 with both banners. Focus cycles through the distinct cooperative choice; accepted intent is captured at final-preview submission, so later focus movement cannot add or remove interruption consent. Editing invalidates the generation and consent; cancellation discards it; final dispatch carries the exact accepted argv and the cooperative bit only for that action.

All SwitchAgent-selected UI tests pass0.310s; selected form/Console cases pass race2.348s; repolint18.586s. The operation-arity backstop correctly failed because the added implicit interrupt-first declaration changes switch-agent from seven to eight arguments; updated that explicit contract and reran it. Reader inventory and whitespace pass. This is a checkpoint, not M2 completion: local force capability/confirmation, M3 and live acceptance remain due.

Correction to the preceding checkpoint: the concurrent inventory command returned a stale run_test.go reader certificate, not a pass; its result was read after commit1814e1c55. Audited that file: only repository inputs are its two unchanged README.md checks, while remaining reads use private test stores/trace fixtures; the sole edit changes the in-memory declared-arity expectation. Repaired only its fixed_scanner certificate through the printed real --record CLI, then reran inventory and the README/arity tests before pushing the combined checkpoint. No input/fallback ownership changed.

### 2026-10-22 — M2 local force capability and existing cleanup checkpoint

The missing force API/typed transport regressions were red before implementation. PrepareForcedAgentSwitch now produces a private owner/fingerprint/wrapper-bound offer; the form opens exactly one risk confirmation, defaults to Cancel and mints AgentSwitchForceConsent only on acceptance. The capability has no serialized authority and can be consumed once under the existing lifecycle hold. Changed argv/record/owner, decoded copies and replay refuse before park. MenuEffect.TypedPayload now forwards through the actual Console operation queue; the dispatcher's existing local typed payload reaches SwitchAgent without any force argument or message field. Strict remote Force/ForceConsent/TypedPayload envelopes and CLI force flags refuse.

Source proof is shared with existing service guards. Force alone waives idle and dirty/status-display policy; it repeats record, exact wrapper/process/connection, verified working path/repository and cancellation checks before and after observations. Private SwitchSourceEvidence now carries its admitted canonical repo identity. Both live and core-proved parked source scenarios are pinned for Claude and Codex. An unavailable status yields unknown count even if its errored result contains a bogus count. ChangedFiles derives from the same porcelain image with one Git call, using SlotGitChangedPaths; malformed/truncated and summarized untracked-directory counts remain unknown rather than fabricated zero.

The hard-stop owner is unchanged (ARCH-DRY/SECURE). A stateful force-through-SwitchAgent fixture for both agents proves exact matching park completion plus helper death precedes target launch. Missing completion and a surviving helper keep SwitchParkIncomplete with no Runner launch. Real worktree file bytes remain identical on success and refusal. No stash/reset/commit/checkout, raw keys, alternate signals, native-stop engine or durable consent record is introduced. One-confirmation risk/kept-file wording, Cancel, stale preview, exact accepted argv and 40x10 rendering with both banners are covered. No usage-limit screen was invented: actual captured usage-limit qualification is still pending.

Verification joined: core force/park/count race11.661s; UI/Console force and switch race2.324s; strict message force-negative race1.256s; combined supervisor force/normal/cooperative/CLI race18.400s. Independent bounded implementor checks passed count0.262s, combined force and ordinary/cooperative38.616s and both-agent force16.730s. Artifact-source classification2.822s, repolint12.756s, reader inventory and whitespace pass. New force helpers delegate artifact IO to existing owners and are registered in the exhaustive non-artifact source list.

README, atlas, CLI help and the shipped Couch skill describe delivered literal args, consent, vacancy and force limits. The real skill-command test caught my bracketed synopsis as non-runnable; replaced it with concrete parsed commands, preserving that check. Help/README/skill/arity verification now passes0.370s. The plan Revisions map the delivered capability/transport/count seams without changing approved scope.

This is an implementation checkpoint, not M2 or whole-issue completion. Pending: actual usage-limit capture/disposable/native acceptance, final full-package/recorded verification and mandatory M2 review; M3 edited same-conversation resume and positive-missing fallback, and final live Echo argument acceptance. Both old recorded runners remain retained/queued: #728 M1 runner22411 latest FIFO13 and #743 repaired runner45648 latest FIFO15. #743 hosted pass remains valid; its actual recorded pass and narrow re-close/merge/named-main VM acceptance remain due. No live settings, :0 stage or real slot effects occurred.

### 2026-10-22 — M2 same-agent intent checkpoint

Final M2 contract audit found the typed restart intent still missing: preparation retained the old different-agent refusal. New Claude/Codex tests were red for the missing intent, then the renderer regression caught its fresh-conversation label and finally the shared breadcrumb overriding the new title. PreparedAgentSwitch now carries fingerprint-bound AgentSwitchIntent from pure ResolveAgentSwitchIntent. Same-agent preparation models restart-prefer-resume; execution explicitly refuses before park while M3 is unimplemented, with no temporary silent fresh path. The form and shared breadcrumb show Restart with new parameters/Restart. Different-agent force and switching retain their behavior.

Final targeted race passes core3.458s/tty1.618s for typed intent, known configuration, parameter focus and force; repolint15.913s, inventory and whitespace pass. Complete four-domain preflight started on the preceding force checkpoint and remains running; it cannot certify these newer edits, so final M2 checks remain due. No review has started. Shared disposable-Couch harness acceptance support is in progress (keyboard-aware press, existing state control, actual argv observation), using its current owner instead of a private harness. Hub requestb2ba75ed-8d49-456b-8c48-977c6feae4eb for the real historical usage-limit capture is submitted. Both recorded suite runners remain retained.

### 2026-10-22 — Complete-domain preflight classification and fixture repair

The four-package run begun before typed-intent edits joined: core247.206s and message27.143s pass; cmd66.606s fails only TestReapOrphansConfirmationIsAProtocolGate/AgentArgs and TestResourceQueryRejectsEveryNonOperationField/AgentArgs because their exhaustive reflection fixtures lack the new comparable struct sample. UI44.890s fails only TestUnusableRowOffersReboot because proved ReasonSessionGone now also offers switch-agent. These are deterministic M2 integration fixture gaps, not flakes, infrastructure, or final-head verification.

Both reflection fixtures now construct explicit empty SwitchArguments and still demand rejection of that non-operation field; unknown future struct types still fail closed. Updated only the old menu action expectation for core-proved session loss. Targeted original failures pass cmd0.362s/tty0.315s. Audited reap_orphans_test.go's inputs: process command/snapshot reads are the literal stateful fakeOrphanTable and supplied RPC stub, with no repository traversal or external fixture read; repaired only its local_or_literal_fixture certificate through --record. Full final-domain verification remains outstanding.

Hub8f80e938-12f1-4047-937d-583c57db6360 is authenticated submitted and replaces the missing-capture blocker: a clearly SYNTHETIC Claude limit fixture plus existing generic unknown-idle/cleanup tests is acceptable before M3's real operator force check. Sourced the short limit line from a firsthand Claude2.1.12 report in anthropics/claude-code#19673; the synthetic layout will not claim original terminal geometry or modern-version capture. The installed official2.1.308 binary corroborates its limit-line prefix and upgrade hint. Shared harness support and production no-composer/no-working/no-input synthetic conformance remain in progress; the new wrapper regression is currently red for the fixture phase that is not implemented yet.

### 2026-10-22 — Shared native harness and synthetic conformance checkpoint

Shared testcouch now exposes keyboard-aware Press through its lifetime gate, generation-bound SetState through the existing command/cancellation/retirement owner, and actual literal Ready.Argv copied at launch. Readiness readers share one32KiB bound rather than the prior4/16KiB caps, accommodating bounded startup argv. No private Couch harness or alternate input owner was added. Fixture phases retain their old behavior; the new SyntheticClaudeLimit is Claude-only, paints two sourced short lines, and accepts no automatic submission. Its comments explicitly distinguish a synthetic layout from real captures. Text provenance: [firsthand historical Claude2.1.12 limit report](https://github.com/anthropics/claude-code/issues/19673); installed official2.1.308 only corroborates the prefix/upgrade hint. The real original screen remains unavailable, as the authenticated hub confirmed.

The new TestSyntheticClaudeUsageLimitDoesNotAuthorizeInterrupt was red for the missing synthetic phase. It now paints the fixture through production terminal/composer recognition and authenticated drive admission, with stale working-turn flags, and proves unknown composer, neither idle nor interruptible, exact InterruptRefusal and zero input. Wrapper synthetic plus existing composer/interrupt refusal race passes2.314s. Shared harness controls/readiness/state/generation/synthetic tests pass race testcouch1.582s/agentfixture1.345s; broader fixture cancellation/mailbox regressions pass8.406s.

Audited changed actor_cancel_test.go: all readiness, PID markers and request-retirement directory reads are generated beneath its owned EvalSymlinks(t.TempDir()) root; no repository source/external capture is read. Repaired only its local_or_literal_fixture certificate. Registered controls.go in the exhaustive non-artifact source inventory. The restart-prefer-resume enum tag initially collided with the reserved restart- artifact-family prefix; its non-path wire value is now prefer-resume, retaining the same typed intent/constant. No artifact-coverage exception was added. Artifact classification2.244s and exact-source intent/force race4.007s pass after that naming correction.

Final domain reruns after the earlier fixture fixes pass: core317.851s (before the enum spelling correction, covered afterward by the exact targeted race), cmd93.610s and UI66.423s; message27.143s passed the preceding complete run with unchanged production message code. Repolint24.537s, reader inventory and whitespace pass. Disposable native CLI literal/empty/saved-argument and retained-parked-source tests, plus a synthetic-Claude-limit/Codex-busy local force/dirty-file test, now compile but have only SKIPPED under their ordinary opt-in gate (0.170s; parked extension0.199s). They are not native acceptance yet. Next commit/push this harness checkpoint, then run those exact opt-in tests on the committed source and classify any actual failure.

M2 boundary review and final recorded verification remain pending; M3 edited resume/fallback and the hub-selected real operator force/Echo parameter check remain mandatory before landing. Both retained recorded suite runners remain queued and must be preserved.

### 2026-10-22 — First disposable run classified; readiness and refusal assertions corrected

First opt-in run on71aa7ebb5698ce281645c6de2895bbcba09c42eb exited1 after80.031s. TestDisposableStartupArguments/claude refused its first actual switch with known-idle requirement (receipt request38656875-d414-4af7-a540-46b349790742); the worker was registered/ready with a captured empty composer, but that is not broker Settled proof. Static trace confirms StartWorker waits Ready+actor count, while production waits its three-second quiet interval; --actors.Resting is branch status, not idle. No product guard was weakened and no same-head blind retry occurred.

Before each live-source request the test now observes existing --resources --json with supported COUCH_RESOURCE_CADENCE=1s scoped only to the disposable fleet. It requires SampledAt newer than wait start, exact current actor/thread correlation, idle_known&&idle and unchanged native PID/generation/agent before/after the read. Observation is bounded to45s/30 iterations at1s; the effectful switch is still submitted once. The proved parked-source call skips live idle waiting only after its native process is absent. Twelve freshness/address/duplicate/idle predicate cases pass; default-gated compile/skip verification passes0.188s.

The force test's Claude and Codex cases both stopped before force because my negative assertion incorrectly failed on ActorCLI's expected nonzero exit/error for a structured refused response. It now requires decoded Code=refused plus a nonzero exit and no cleanup uncertainty, preserving failure on transport/retirement uncertainty. First-run cleanup confirmed all owned generations absent and direct-child/IO joined for roots pc-1748817799,pc-2065607438,pc-3563974639. Neither case exercised confirmed force in that run; no native pass is claimed.

Audited both new native readers independently: they call shared testcouch.Build/New, which compile current checkout/runtime assets and read installed tool facts; direct file checks use only private generated fleet paths. They inherit the same non-Go full-suite fallback obligation as fleet_acceptance_test.go rather than pretending to be local-only fixture readers. Recorded each new reader certificate through the real --record CLI. Required inventory and whitespace pass before the next committed-source native run. M2 close dry-run exited0, would dispatch its mandatory review and wrote nothing; it establishes gate readiness only, not a review or acceptance result.


### 2026-10-22 — Native force passes; normal fixture setup classified

Committed-source native run on76902af1dde046313124c0e3ce6e4e978bc24bd9 joined exit1 after96.933s. Both emergency force cases PASS (58.68s total): synthetic Claude quota and Codex working screen each refuse ordinary switching, retain the original process before the explicit local confirmation, then replace it and preserve exact binary dirty-file bytes. Shared cleanup confirms every owned generation absent and direct children/IO joined. This is synthetic/native fixture acceptance, not the pending real operator M3 check.

Ordinary literal-argument acceptance passed its new idle observation but refused its first switch because setup generated untracked .gitignore and .gitattributes. Inspection of the concurrent identically provisioned private worker proves precisely those two untracked files; their bytes are weave-generated ignore/merge policies. Before any agent starts, this test now commits only those two generated files in its private disposable repository and requires empty porcelain. Production Git/idle guards stay unchanged; no live repository or user files are committed. The switch remains one effectful attempt, with no refusal retry. Reader re-audit retains non_go_full_fallback for real checkout compilation/tool reads; added Git commands access only the private provisioned worker. Compile/default-gated and freshness predicate checks pass0.190s; final native rerun remains due.


### 2026-10-22 — Native literal argv proved; replacement registration wait corrected

Run on739e5f5b57f99b7b6f00309a108081f3397b18d2 joined exit1 after35.923s. Its first Claude-to-Codex switch succeeded (receipt7b72088f-c2dc-4473-a986-155b5d0df6c5), observing exact --model, spaced/empty/shell-like literal arguments and a new native ID. The next read-only idle wait failed immediately because native Ready preceded replacement wrapper registration: public actors contained only the hub. This is a test observation-order gap, not evidence of a failed switch or permission to replay it. Cleanup proved all owned processes absent.

The existing bounded idle observer now waits for missing/old-agent registration within its same45s/30-probe budget, retaining exact native PID/generation checks, ambiguity refusal, fresh resource correlation and the one effectful submission. No product delay, guard weakening or effect retry was added. Re-audited the sole reader: input ownership is unchanged, so refreshed only its existing non_go_full_fallback certificate. Ordinary native acceptance remains incomplete pending the next committed-source run.


### 2026-10-22 — Native live argument paths pass; parked admission uncertainty retained

Run on28da828d87f437df09a060af8b232312ca362b6f joined exit1 after57.733s. Three live switches succeeded with exact literal flags, explicit-empty parameters and reuse of saved literal parameters (receipts3ac53cd7-3058-4789-93b9-246c946c22c9,81a5859e-45ba-46e2-a7f0-0fb20eef1bd5,47a28511-cbdd-4abb-8f2d-8cff66207b96). The final retained-source call returned Code uncertain/context deadline exceeded with receiptdbc37ba8-9a09-4864-b1f4-e42a145ef178. It was not resent; shared teardown confirmed every owned process absent. Its actual admission outcome remains unknown, not success or a flake.

Static trace confirms the test treated native process disappearance as completed park, even though ParkExpected still has transaction finalization and console completion afterward. The public menu emits park complete only from the successful correlated operation result, so the test now waits for that barrier before its sole retained-source call. This repairs a known test sequencing gap; whether it explains the timeout is still unproved. Also corrected receipt handling to inspect an uncertain admission through its exact public read-only operation-status ID for at most45s, never resubmit. Unknown remaining at the bound fails. Reader audit remains non_go_full_fallback; the new observations use existing private fleet/CLI ownership. Native retained-source and second-agent acceptance remain due.


### 2026-10-22 — Park completed; retained admission timeout remains under diagnosis

Run on8400afdbc7c895b880680dcfd4cb5a58c4aa0b3d joined exit1 after60.307s. All three live argv paths succeeded again. The test observed actual park complete before the retained switch, which still returned an uncertain admission. Its exact receipt1bfec60e-1f14-4954-a57f-51dac2e88c30 was queried once, returning structured Code ok/Status unknown; the helper incorrectly rejected the CLI's expected exit1 before interpreting that status. Corrected that observer-only error so unknown can remain bounded/unknown, never success or an effect retry. No existing uncertainty was resubmitted. All owned generations were cleaned.

Independent read-only trace agrees: process absence is earlier than park finalization, but overlapping park alone cannot explain the timeout because lifecycle admission/queue guards refuse promptly. Retained admission synchronously resolves workspace/selection/authority/classification/preferences/Git before receipt creation under the CLI two-second budget. Exact failing phase is still unproved. Added failure-only timing diagnostics around the real installed Git and SDLC commands in the disposable fleet's private PATH; each invocation owns a unique private log file, stdout/stderr/exit are preserved, and fixture cleanup precedes log collection. This does not replace either actual tool with a fake. Raised only this multi-switch fixture's total runtime budget from default120s to its existing240s context, accommodating bounded receipt inspection. Default-gated compile/predicate check passes0.176s; next run is diagnostic until its actual outcome is classified. Reader re-audit retains full fallback because Build reads the checkout/runtime assets; timing reads are private generated files.


### 2026-10-22 — Diagnostic observer effect classified

The Git+SDLC timing run on5ef9e6b90e785afcdebcb09e398d57e97d5651ea joined exit1 after255.899s, before any switch: StartWorker.WaitActors exhausted its240s context. Native readiness existed but broker registration did not qualify within the bound; cleanup confirmed all owned generations absent. This is not switch acceptance or an established product failure. Mid-run inspection counted over830 Git invocations, each expanded by the observer into a shell, mktemp and two date subprocesses. Independent static review confirms unchanged selected tools/argv/cwd/environment/normal output, but significant process overhead and changed cancellation ancestry. One-second timestamps cannot establish subsecond native costs.

Removed all Git interposition. The next diagnostic delegates directly to actual Git and wraps only the much smaller number of SDLC workspace probes. This preserves the test's single effectful request and bounded unknown-receipt observation. Default-gated compile/predicate check passes0.181s; audited only the changed reader certificate. Existing force acceptance and uninstrumented three-live-switch successes remain historical evidence, while retained-source acceptance is still pending.


### 2026-10-22 — Retained admission cost isolated structurally; focused revision proposed

SDLC-only diagnostic sourceaf413c85fa10968cdbfe793daf3eabc2c6c24081 joined exit1 after86.255s. All three live argv switches succeeded; after actual park complete, retained admission again timed out (receipt5864ecc5-56fa-458b-9e72-681ee8e69b67). Exact-ID read-only observation stayed unknown for its45s bound, then failed on context expiry; no effectful resend. Shared cleanup proved all owned generations absent. Logs show84 real SDLC workspace probes across setup and the sequence; second-resolution timing is insufficient to name one slow probe. Independent static tracing proves eight probes in the retained admission before preparation/guard work for a single numbered slot, growing with unrelated slot count.

Appended a concrete proposed plan revision: direct verification of the named candidate via existing SlotWorkspaceResolver instead of whole-repository Discover, preserving repository-name/sibling semantics, passive retained selection and all fresh authorization/effect checks. No new user-visible state or retry/timeout/process owner. This production mechanism refinement awaits hub approval under tl-worker's full-flow rule; no production change has begun. Hub report870268b1-b83b-437b-89be-ff2e7eee7a48 was queued (not resent). Both old recorded runners remain retained; M2 review and M3 remain pending.


### 2026-10-22 — Approved direct-target correction implemented and preflighted

Authenticated submitted hub55a89732-6199-4358-a9f5-4b1c41e798f1 approves fixed two-probe target resolution with all fresh selection/identity/hub refusals retained. Counted8/16-slot regression was red on whole-catalog discovery; direct slotWorkspace implementation passes fixed two probes per resolution, including a repeated check, with no unrelated discovery. The additional sibling/alias recheck test proves a newly present exact sibling cannot inherit the previous alias selection. Existing target cases pass5.559s; final expanded target race passes3.304s. Service guard/queued/effect race verification is still running and must be joined before claiming its result.

Production change is confined to switch_target.go, reusing repositoryPrimary, NewSlotLayout, WorkspaceIdentity validation and SlotIdentityFromWorkspace. Removed the diagnostic shell interposition from native acceptance; Git and SDLC now execute directly. Re-audited the changed native reader, which still compiles real checkout/runtime assets and therefore keeps non_go_full_fallback; no other reader hash was refreshed. Repolint8.180s, artifact-source classification1.886s and default-gated native compile/predicate0.181s pass. The committed-source native rerun remains mandatory before M2 review.

Service guard/queued/effect race run subsequently joined PASS18.565s before checkpoint; inventory and whitespace also pass.


### 2026-10-22 — Direct-target native results and measured admission overlap

Uninstrumented source8e90ac6d03f15d5e82faaadf08a0e2b30e802543 combined acceptance joined exit1 after111.721s. Both force cases PASS52.89s (Claude21.06/Codex27.29), exact dirty bytes preserved, owned cleanup proved. Ordinary Claude sequence failed at its third live switch, receipt6c0bbd04-44ba-4eb3-9fd2-32fe852837f0, park preparation diagnostic operation-6744889532cd244c86919e84ac29ce3f: lock thread store: resource temporarily unavailable (EWOULDBLOCK). No target launched over uncertain park. Same-head diagnostic repeat joined exit1 after104.428s: all three live switches passed unchanged, then parked admission timed out and exact receipt28c74658-ef4d-49a3-b857-7208d39cd3b2 stayed unknown. This clears that phase but is not a passing whole-test rerun, so no flake registration or passing acceptance claim.

A temporary local phase trace (24 service lines +8 test lines, explicitly outside committed acceptance source) completed both agent sequences and owned cleanup, exit0/121.642s. It measured retained initial target0.60s, immediate authorization0.65s, preparation0.08s and vacant guard0.48s, enqueue~1.86s against2s. All temporary edits were byte-restored from owned scratch originals and git status verified clean afterward. This diagnostic pass is not final source acceptance. Original scratch backups remain /Volumes/TestScratch/tmp/pair-slot3/pair728-admission-trace.p49orz4j; no trace is in production.

Clean source8e90ac6d03f15d5e82faaadf08a0e2b30e802543 final repeat joined exit1/141.266s: Claude sequence PASS51.21s including retained parked switch receipt a19872d4-47c4-40be-b791-e5be9658f80d; Codex live literal/empty/saved paths pass but retained switch e4559ccf-2193-41c3-9dbf-af2abb84a958 timed out and remained unknown through bounded read-only inspection. Codex80.56s fails; whole test has not passed. Every owned generation/child/IO was cleaned. No unknown operation was resent.

Authenticated submitted hub808220ef-45c7-4c34-bc07-96ffde1a7b28 requests checking duplicate observations first, then a measured/re-measurable budget proposal only if necessary; no timeout change is authorized. Read-only comparison confirms five repeated SDLC probes across the three phases and existing hub_guards.go admission-versus-queued-source precedent. Appended a concrete proposed one-snapshot admission revision preserving fresh selection/record/Git/hub checks and full queued/effect verification; awaiting hub decision before implementation. Both retained recorded runners remain alive, latest M1 FIFO4/#743 FIFO6. M2 review, M3, final recorded head and actual operator acceptance remain pending.


### 2026-10-22 — One-snapshot admission approved, implemented and counted

Authenticated submitted hub22be8d8d-5539-4286-9f49-839b12c7dfd5 approves one request-local observation, cheap fresh selection/caller/hub checks and full queued/effect proofs, keeping2s. Implemented exactly that split (ARCH-DRY/CONSTRAINTS/SECURE/ORDER), following hubSlotSources rather than adding a new cache/worker. Private observation never substitutes in queue/effect guards. Core's new passive recheck was red for its missing API, then passed expanded target race3.311s.

Independent bounded service-test worker observed behavioral red2 resolutions/1 workspace probe, expected1/0. Implementation passes those counts for both target agents, with fresh record reads and Git retained; queued preparation and effect checks independently increase full resolver/workspace calls. All ten refusal scenarios pass, including new selection drift during Git and cancellation. Main inspected the test/code diff and ran final focused race2.261s, broader switch/guard/force/queued/effect race18.349s, repolint6.749s, artifact-source classification1.584s, inventory and whitespace. New service reader uses the existing cold/lean stateful fixtures and private generated stores; no real checkout source traversal or external fixture/capture is introduced, and the inventory requires no new reader certificate.

Atlas and plan map the delivered private admission lifetime. No temporary phase trace or tool wrapper remains. Next checkpoint/push, then exact native Claude/Codex argv/parked/force acceptance; do not claim the earlier traced diagnostic pass as native final evidence. Both old recorded runners remain retained; M2 review and M3 remain pending.


### 2026-10-22 — Nested working directory preserved; native startup classified

The request-local observation incorrectly equated a saved working directory with its verified Git root. The regression first failed on a legitimate nested directory. Correction preserves the single resolver/zero extra workspace-probe count, while physically resolving and checking directory containment before and after Git. Nested paths pass; symlink escape, non-directory, disappearance and symlink replacement during Git refuse without queueing. Final expanded focused race passed2.979s; broader switch/guard/force/queued/effect race20.814s, repolint8.514s and inventory pass. No budget/state/retry change (ARCH-SECURE/DRY).

Clean48a1dd8b native run34397 failed ordinary Codex third live park and force Codex park with EWOULDBLOCK; no uncertain cleanup was bypassed. Classification run26762 joined exit1/332.340s: Claude argv sequence passed42.43s; Codex initial fleet.Start exhausted240s before any switch, without agent-ready or wrapper-startup.log. Authenticated ops:0 f8be970b-5e4b-42ee-b226-a7f3734ef66d classifies this as #773/#806 startup family and requests the occurrence on #820. Its transcribed evidence (not raw terminal/server log), retained at /Users/xianxu/.local/share/pair/evidence/000820/20261022-pair3-codex-startup-transcribed.txt, has SHA2561092a8d887cd7e792fa31917ffd6ef86f49e953ca83dc5ef14951afae3d3cac4. #806 is codecomplete but not landed, so server-tail retention was unavailable; owned cleanup removed the private root.

The later force cases passed41.11s but Build recompiles the checkout for each top-level test; I edited this nested-directory fix before that Build. Those cases are diagnostic, not clean same-head evidence. Commit and push this correction first, then exactly one clean same-head native run, retaining raw output from the start and making no source edits while it runs. If the same pre-switch Codex startup signature repeats, record it once and stop retries; the authorized M3 operator check covers the real path. M2 review, M3 implementation and final recorded/live acceptance remain outstanding. Existing M1 and #743 suite runners retain their FIFO places.


### 2026-10-22 — Final clean M2 native run and accepted environmental refusal

Source cf4a3aba8197c7ec961c32c62b33f3229eeba073 stayed clean through both top-level builds. Runner40889 joined exit1/152.127s. Ordinary Claude full live literal/empty/saved and parked-source sequence PASS56.88s. Codex first two switches passed; third switch receipt9cfb3b1a-f1da-4f1e-b88e-69d1b1ac0fa4 failed attach-preflight, operation-b221e2fcaffd34145b3defd0c7bee28e. Raw PTY output names terminal 📁9223725645807222786-5: terminal listing omitted a live server; attach unverified. This is launcher requireListedLiveTerminal's existing ErrListingOmittedLiveServer (#692), not the earlier initial no-ready startup signature. Both source sequences and cleanup completed; no retry.

Force acceptance PASS51.41s: synthetic Claude quota/unknown-idle24.70s and Codex busy21.04s; actual local confirmation, unchanged dirty bytes and all owned generations absent, direct children/IO joined. The fixture is explicitly synthetic and does not claim real Claude quota capture. Retained raw combined output: /Users/xianxu/.local/share/pair/evidence/000728/native-final-20261022/run.log, SHA256 cb0b2cf75c161c0ba3afa92f2ff8a197d136fe19e03865dd57a15fbebd4b1418; source-head.txt beside it records the Git head.

Authenticated submitted ops:0 54a9ef63-f2de-45dd-9a05-a390818006de classifies the third-switch refusal as the known Zellij false-miss family (upstream#5751), environmental rather than #728 switch logic, and accepts M2 evidence. Operator M3 live acceptance still exercises the real path. Add this occurrence to the open false-miss issue (or file unclaimed if none), then mandatory M2 milestone-close. Final focused observation race rerun PASS2.590s and inventory PASS. Neither the whole native test nor the pending recorded suite is claimed green; final-head recorded and operator checks remain landing requirements.


### 2026-10-22 — M2 review REWORK; exact token and preview corrections

Mandatory fresh review returned REWORK, BR-5 Critical source-generation fencing and BR-6 Important immutable reducer state; M2 was not finalized. BR-5 reproduced behaviorally: local ordinary and cooperative operations admitted a reconnected wrapper and actually reached park/start (red0.426s). Both live guard phases now return their existing verified token to SwitchAgent, which retains it for the final guard; remote supplied tokens remain checked, force keeps its already consent-bound token and vacancy retains its empty token/full proof. Local/remote ordinary/cooperative controlled reconnect tests assert unchanged revision, no park/start and old process live. Actual service tests assert both local empty-token and remote phases return the exact observed wrapper token.

BR-6 reproduced both Escape/Cancel input/snapshot mutations and races against rendering (red0.407s). Copying the prepared preview before clearing its Force field preserves shared snapshots and original one-shot offer. Independent bounded worker swept the analogous reducer preview sites; no other writes to shared switch preview fields found. Main inspected its diff. Focused source/interrupt/force/preview race checks pass core1.975s, cmd5.415s, tty1.463s; repolint18.004s and inventory pass. Broader race run exposed a separate test-fixture initialization race: admission rig rewrites authority callbacks after the socket service starts; fixing construction/callback fake state before claiming the broader check green. No additional native retry is authorized or run. Plan/atlas/lessons record both correction classes.

The retained M1 recorded suite finally joined PASS11m39s, exact e7de5ca161079b427e7fe4726d3782ac03d35b95, recorded sdlc-local/suite. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/e7de5ca161079b427e7fe4726d3782ac03d35b95/suite.log, SHA25644a3e9a8d0c3a47a2f8b96aedd10e63ba2d0cbcc2a92878a8960e987f2009cc5. This proves M1, not M2/final head. #743's retained runner has now started its own snapshot; no restart or duplicate check. M2 re-review precedes M3.


Fixture correction completed before re-review: authority callbacks/hub state install through existing decorateAuthority before service start; cold admission no longer connects an unused source wrapper. Branch faults use an atomic callback fake, preserving actual caller authorization and all counted/refusal/nested cases. Independent fixture race repeated three times PASS4.433s. Final broader switch/guard/admission/force/queued race PASS core2.793s, cmd10.941s, tty1.439s; final repolint6.711s, affected inventory and whitespace PASS. No production synchronization or new owner was added for a fixture-only race. Commit these corrections with the REWORK evidence, push, then rerun M2 milestone-close as the gate directs.
