---
id: 000773
status: done
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-22
estimate_hours:
severity: high
collaboration-mode: collaboration
card_mirror: 'd6b971843461ac4d24a56aa9794788d44a984912' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T08:12:54-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 6.84
---

# startup: terminal readiness truncates the caller to a fixed 10-second bound

## Problem

Production terminal readiness imposes a fixed10s deadline even when its caller has a longer remaining deadline. A cold native agent launch under load can be refused before the caller budget expires. Discovered by #744 final native matrix, independently from #772's attach InputEnded failure. This is a startup-budget design defect; do not classify it away as a flake or increase a numeric timeout.

## Spec

Make terminal readiness honor a coherent caller-owned startup lifetime while preserving cancellation, bounded observation, fail-closed admission and generation-qualified cleanup. Design the production default for callers without deadlines explicitly; do not substitute a larger arbitrary constant. Preserve distinct precreation, readiness and postbirth native-query observation contracts and error evidence.

Approved design: select one absolute startup deadline before the blocked helper is launched; reuse it for helper acknowledgement, pane birth, exact-generation readiness and Couch registration. Preserve an explicit caller deadline; with none, reuse the existing two-minute registration hang ceiling as the shared fallback. Consume the serialized deadline once across exec. Carry startup separately from interactive lifetime and the existing short native-query budgets; startup expiry never ends an admitted interactive client. Full design and adversarial tests: [startup budget plan](../plans/000773-startup-budget-plan.md).

## Done when

- The actual production handoff path no longer truncates a longer caller startup deadline to the fixed birthBound10s; an earlier caller cancellation/deadline still stops promptly.
- A deterministic regression exercises the real production budget-selection boundary with readiness after10s but before a longer caller deadline, plus earlier cancellation/expiry and absent-readiness controls. No retry or blanket timeout bump.
- Cold native launch verification and required recorded checks pass, with original failure evidence retained and no claim that load/AMFI was the measured cause unless proved.

- Existing progress reports the current startup stage and elapsed time; budget-exhaustion failure identifies the consuming stage and duration.

## Plan

- [x] Approve the caller-owned startup/default/transport contract and production-boundary test plan.
- [x] Record deterministic red probes, implement shared deadline selection and propagate it through helper, launcher, readiness and registration without renewal.
- [x] Complete review corrections for public borrowed-client deadline transport, local stage timing and causal warm-adoption fixture controls; requalify the consumer class.
- [x] Use the landed #744 shared disposable harness for cold/warm native acceptance and complete corrected package/race verification.

Remaining gates: a passing corrected recorded suite, mandatory boundary review and normal SDLC merge; no live-fleet/pair:0 operation is authorized by this check.

## Log




- 2026-10-22: closed — Additive atlas/index and artifactpath inventory conflict resolutions preserve both main and #773 entries. Noon main integrated without conflicts, full exact-head recorded suite passed, fast reader inventory passed. Cold and warm shared Claude/Codex native matrices previously passed with exact joined cleanup; later missing-wrapper peer captures retained as unclassified. No stage/live-fleet changes; hub owns project. (fixed-then-ship; full suite: Noon-fresh integrated head be30575e0c8d797c4babc08a478f00cdaebbf089 passed full recorded suite attempt 6053f9073f48f168117b2fa568c4b339, exit 0 in 12m40s, log 89fd95fdf870. Prior integrated-runtime pass 4b42e6ed7c8af4526a196f8a61977aa8 also retained; #685 timeout occurrence and authorized rerun documented.); review verdict: SHIP
- 2026-10-22: closed — BR-1 public warm entry RED0.286s missing deadline/10s cap then corrected public/warm cohort GREEN7.467s; exact30s deadline consumes2s preparation plus1s ack without renewal or interactive cancellation. BR-2 stage-age RED25s instead5s then controlled/race console GREEN. BR-3 delayed-preparation RED then causal warm/retire/retention controls GREEN with virtual budget consumption. Full core PASS308.418s/tty51.055s; race core4.530s/tty1.580s; lint/artifact/inventory pass. New shared native Claude/Codex matrix PASS131.25s on88f7326491b1 including delivery/detach/restart/all4 actors returned/exact joined cleanup, log97403cefee19. Corrected pushed head f3c423c6462058da7433262a1617a9fb42ab3386 is in new recorded affected/full check; result pending and mandatory before landing, review overlap permitted by tl-worker. Prior recorded green predates fixes and is not reused as final proof. Hub owns project; no stage/live-fleet operation.; review verdict: SHIP
- 2026-10-22: flow upgraded quick → full — 481 added lines in code files (limit 100); an earlier round of this close already ran the full review
2026-10-21: filed unclaimed by pair:5 per TL sequencing for #744. Trace: launcher/birthwatch.go birthBound10*time.Second; session_handoff_os.go passes it directly to coordinateTerminalHandoff; session_handoff.go applies context.WithTimeout(original caller,bound). Fleet caller allowed120s; separate LaunchOptions.BirthBound controls outer birth watch only; Couch registration default2m. CLI15s message is progress, not the budget.

Retained native failure on exact code db791e3b82bb23242ecd5e8fdbebd550c9da8e04: Claude fixture:3 (allocated thread tag1-fixture-4), nonce operation-9923c3b354cb3f27bbbb015f499d4a66; stage registration, Pair before/after exit1. Last output says terminal admission unverified: expected agent-ready-1-fixture-4-claude.json missing, then context deadline exceeded. Operation prepare877ms/operation15.732s; TestDisposableCouchFleet FAIL52.90s; whole command56.322s. Exact cleanup passed; no same-head rerun has occurred. Evidence in #744 PR509 at workshop/evidence/000744/final-outcome-native-fail.log, final-outcome-run.json, final-outcome-result.json; logSHA256355723305828834b91e4c8b91a2374d32d87e3f3c180bc00174f8c9f34058b42. Evidence commit51a07ef3936625a00011008e8d043db10f412efe.

Collision audit found shared scope/tag/agent resolver, unique tag-specific final/tmp paths and exact session/nonce validation; slot address3 versus tag ordinal4 is not a collision. Failed effective wrapper environment and sibling-file census were not retained, so incident inheritance remains unproved. Harness copies executables once per fleet; all workers reuse the same paths. Read-only macOS logs show contemporaneous Gatekeeper assessments with redacted paths; no event attributes this delay to the failed executable. AMFI, machine load, early wrapper failure and missing publication remain possible causes of this particular absent readiness; none is measured as its root cause. #744 is extending pre-cleanup evidence, without changing the production deadline.

### 2026-10-21 — production budget trace and proposed plan

Claimed from clean main after #765 cleanup; bare start-plan refreshed the sibling provider normally (Brewfile check only, no install). Read-only source trace confirms three separate relative startup windows: helper acknowledgement ten seconds, production OS admission birthBound ten seconds, Couch registration two minutes selected only after acknowledgement. Postbirth native observation is a fresh five-second query under interactive lifetime. Existing regression tests intentionally separate precreation-query expiry from readiness and preserve admitted interactive lifetime, so simply reusing queryContext or deadline-bounding LaunchOptions.Context would regress them.

A bounded independent read-only subagent traced Couch/exec/registration callers and confirmed these boundaries; no implementation or test was started. Proposed one absolute startup deadline and explicit phase/lifetime separation in the durable plan. #744 peer reports PR509 still unlanded; its final native matrix passed, recorded suite/close pending. Do not stack, copy that harness or claim its result as #773 acceptance. #739 and #738 remain separately held; #773 does not alter their work.

## Revisions

### 2026-10-21 — one startup owner, not a readiness constant

Reason: the production trace found independently renewed helper/readiness/registration windows, so removing the innermost constant alone cannot fulfill coherent startup ownership. Proposed delta: select one absolute deadline before helper launch, carry it across exec, preserve the existing two-minute fallback only when the caller supplies no deadline, retain shorter observation budgets and separate admitted interactive lifetime. The ten-second birth probe remains a diagnostic threshold. Native acceptance uses #744 after landing; original incident's missing-ready cause remains unknown. Awaiting hub approval before change-code or code.

### 2026-10-21 — hub approval and progress requirement

Hub approved the design and directed change-code. Required UX: existing progress identifies the current stage and elapsed time (waiting for agent readiness, 25 s); final failure identifies the stage that consumed the budget and elapsed time. Added to the durable plan; native acceptance remains on landed #744. No implementation started before the gate.

### 2026-10-21 — first red/green implementation checkpoint

Behavioral RED at the shared production handoff selection boundary: virtual readiness at 12s refused despite a 30s caller, and absent readiness failed at exactly10s. After replacing the admission birthBound with the explicit startup context, both pass in virtual time; default, early cancellation, short-query separation and admitted-client survival controls added. Initial red fixture cleanup had a synctest goroutine-lifetime error; fixed/joined it and repeated RED cleanly before the behavior fix.

Couch transport RED showed no deadline in the helper environment. It now selects before helper preparation, passes the exact timestamp once, and caps registration under that already-selected deadline. Kept registrationDeadline as its registration-only test seam after one existing exit-before-observation regression exposed that moving the seam earlier changed its meaning; no production deadline renewal. The corrected tracked-start and cold late-start/direct-exit/final-look cohort passed1.465s. Added context/error identity protection when caller/client expiry races readiness.

Independent bounded helper work proved expired/invalid timestamps and late acknowledgement incorrectly execed before the fix; now acknowledgement is capped by the absolute deadline, expiry is rechecked before exec, and the read goroutine is joined. Targeted helper controls pass0.274s. Pure value/codec controls pass0.058s, including behavioral ten-second-clipping negative control. CLI consumed-environment RED returned help success and leaked the internal deadline; corrected native entry passes0.191s.

Remote progress RED printed only generic running status. Existing context reporter now reaches an exact-request progress callback and pure receipt transition; same-stage age preserved, late/terminal/reused-ID progress refused. Broad service/authority cohort passed0.170s/9.810s and targeted race cohort1.264s/1.602s. Final failure's consuming-stage/elapsed control and broader core startup/ready/registration/birth/recovery cohort passed22.330s. Full launcher package and final inventory/suite/native acceptance are still pending; no completion or live success claimed.

### 2026-10-21 — pre-recorded-check checkpoint

Full launcher package passed65.149s. After subsequent deadline-race/birth-probe changes, focused production/handoff tests passed0.194s; all relevant core startup/readiness/birth/registration/recovery tests passed22.330s and service/receipt/authority cohort passed0.161s/8.905s. Targeted race controls passed launcher1.236s/core1.392s/message1.195s/command1.433s; that selector did not select pure value tests, whose ordinary package checks passed separately. Birth-probe cancellation RED showed a false dead/abort from the legacy non-contextual query; switched that existing probe to its short contextual observation under startup and the deterministic control passed0.288s. No new probe cadence or timeout bump.

The repository lint disallows even reasoned inline fixed-sleep suppressions. Tests now receive explicit timer events only inside synctest virtual time rather than adding sleep debt; catalog and new narrow admission-budget guard pass2.045s. Fast affected inventory passes; three existing reader certificates re-audited for mechanical callback/context signature changes only (run_test fixed scanner, createflow/workbench fixture-local). No new source reader was waived. Production code exceeds the quick shell, so close must use its full review upgrade; change-code initially inferred quick from the compact design. Native acceptance/close/landing remain pending, no live result claimed.

The actual RunLaunch create/attach boundary control initially assumed the interactive context must be the identical parent pointer. Create correctly owns a cancellable child for its birth watchdog and cancels it after its blocking client returns. Corrected the oracle to require no startup deadline on that child and an uncancelled original parent; control passes0.201s. No production change was made to satisfy the mistaken identity assumption. Short preflight contexts now inherit the original query context directly (synchronous cancellation), are deadline-capped by startup, and recheck startup before Start. OS entry validates startup before constructing the filesystem-backed attempt.

### 2026-10-21 — zero-deadline fix and restart accounting

Ad-hoc focused integration audit found a concrete supplied-zero edge: canonical year0001 decoded to time.Time zero, and Select treated it as absent, renewing the fallback. Behavioral codec RED accepted that value; actual LaunchNative RED continued to repository validation instead of refusing startup. The shared codec now rejects an explicitly supplied zero, while absent caller deadlines still select the fallback. Focused codec/native/helper/production controls pass0.125s/5.792s/1.180s. No source effects occurred in the private invalid-input fixture.

Cancelled queued suite lease-691820565 through supported local-check cancel before fixing that code; it had not acquired capacity and its terminal result is interrupted, not a failed/passing suite. Following the08:44 restart instruction, queue/landing show no own run or merge, and draft PR525 still points at pushedcbf27bb7. Requeue only after the corrected head is pushed. This cancellation was intentional, not a flake retry or presumed restart loss. Hub has a submitted one-line accounting report. #736 M2 is now additionally authorized for interleaving; read-only metadata diagnosis is happening without mixing its code into this branch.


### 2026-10-22 — resumed after #739 preparation; retained related captures

Hub59dbba43 prioritizes #773 before #736 after #739 morning preparation. #738 landed; #739 main integration and targeted checks are pushed at96c49c8da80e28082cce1058440ed2233396a595, awaiting the hub's explicit stage go. #773 resumes on its own branch, integrating main7702a3dddbfdb5c7d8a021bb8949df98f0a6b282. Additive lint catalog conflicts preserve all rules. No second checkout or live fleet mutation.

Peer #776 evidence (receipt b57b2d30) is retained in workshop/plans/000776-detach-profile.side/native-current-profile.log and current-profile-summary.json on main: eight-total-slot current profile never reached detach; worker4 Pair exited1 with expected agent-ready absent then deadline exceeded. Request18:58:06.588319Z–18:58:29.370249Z; fleet start18:56:56.198242Z; fixtures had no lifespan deadline, caller startup lifetime was longer; exact owned cleanup passed. Original temporary capture root was /Volumes/TestScratch/detach-compare.O0eTeh. Hub classified it as #773 evidence, not a new successful guarded-path acceptance.

Peer #722 captures are durable in PR477 at peer-reported6aada8623f438c3fc6d400faa4e6863d1678cfe0: workshop/evidence/000722/native-shared-f39a3ae7-startup-failed.log/json, full log SHA2560341a32830cf0dbc8bc97b46a11b56eddce1ae8deba385fd3260c4ac84b6d062. Exact tested sourcef39a3ae7bbf0933ebb1a2db038b803479e7678d7: Claude passed; Codex steward failed before any drain effect, unverified admission/agent-ready missing, wrapper startup/failure identities absent and startup_sources0; exact cleanup passed. The separately authorized same-source focused Codex run subsequently registered all3 actors and completed notice/work/ack delivery before a separate screen check/use race. That capture is native-codex-f39a3ae7-frame-failed.log/json, SHA2565012f1b53bc7288c62d5ed2adaced41c73ee4a91e7bf04cc43754f995e0557c7. It is NOT another startup recurrence or evidence of a startup cause. These are canonical peer-reported observations, not this worker's native measurements. #792's capture is still pending; do not infer a root cause from its card title.

The missing-ready incidents remain unclassified as to initial wrapper startup, load or filesystem publication; the proven fixed defect is the production ten-second readiness clipping and renewed phase budgets. A missing wrapper identity is not proof of that same cause. This distinction must survive the close/rollout report.

Integrated deterministic production/handoff/helper/registration/budget cohort passed: startupbudget0.131s, launcher8.886s, couchcore25.222s, couchmessage0.160s, couchcmd1.372s. Merged lint rules passed0.147s. Native conformance uses a focused consumer of landed shared testcouch: genuine cold steward+3 workers and exact cleanup for Claude/Codex, omitting independent message and detach/restart operations. It retains the existing180s setup and120s runtime caller windows; no delay injection or cap increase. Deterministic virtual12s readiness remains the proof against ten-second clipping; native success is conformance only. New opt-in consumer is explicitly certified non_go_full_fallback because Build/New compile and provision outside the Go import graph; fast inventory passes. Native invocation and final recorded checks still pending.


### 2026-10-22 — bounded native cold-start acceptance passed

Exact Git code head b98b3752ec5642e1ecffd2cd238a1c8b7d85e4df passed one shared-harness invocation for Claude and Codex with no retry. Full command: PAIR_NATIVE_COUCH_FLEET=1 go test ./cmd/couch -run ^TestDisposableCouchStartupBudget$ -count=1 -v -timeout=12m. Native test PASS99.93s; whole invocation100.759s, exit0; start09:42:29Z/end09:44:10Z. Measured load1 before34.11/after33.43. Claude genuine steward+3worker registration32.983s, Codex35.798s. Each reported all owned generations absent, direct child and IO joined, and AssertCleaned/root-absence passed. No live fleet, pair:0, visible window, private cache, timeout increase or machine change.

Durable complete log: workshop/evidence/000773/native-cold-b98b3752ec56.log; matching JSON records invocation/head/time/load/exit. Full log SHA256040c9f6d3f1498f1100d624283fc650173672649b092bfe588825bd452ef3cba. This qualifies native cold registration on the changed production budget path, not causation of each earlier missing-wrapper incident. Deterministic virtual12s and phase-deadline tests supply the non-clipping/no-renewal proof; the native cohort's total duration is not a per-agent readiness measurement. Required recorded verification and fresh boundary review remain pending; no landing claim.


### 2026-10-22 — related restart-registration capture, not cold-readiness proof

Canonical peer receipt d5ee8343 reports PR477 at ec88486fe63198753349d1fc12e5ce7b9fed968d now retains workshop/evidence/000773/pair722-6aada862-restart-registration-failed.log/json (SHA256e06556d9e51a859c49f34f5902a20c03486800d2a4aedcb0e92fed98c8a0355d). Label: steward registration deadline after RestartSupervisor. Claude completed drain/notice/ack/undrain/badge/selection/quit/detach, then shared Start/WaitActors hit the caller deadline; current steward registration absent while surviving workers had valid ready files/panes; cleanup passed. Cause is unclassified, not asserted equal to birthBound. Peer says TL stopped further native reruns on #722 and requested its recorded checks/re-close/watched live acceptance; this conveys no change to #773's own authorization. Keep this attachment distinct from the earlier pre-agent-ready capture and this branch's successful cold-start conformance.


### 2026-10-22 — recorded suite completed; bookkeeping and exact-env oracle repair

Recorded full-suite attempt6c97fd46f24e0ceee10c4d23ac849831 on c8af3052487d6b242b6ada4e4ae7383b7b6094b6 completed11m2s, exit2; no lost runner or early stop. Log .git/sdlc/test-stats/failed/6c97fd46f24e0ceee10c4d23ac849831.log, hashd43e244cffdb. Every other package completed. Failures: TestProductionArtifactReferencesAreExactlyClassified named four new production metadata helpers missing from exhaustive inventory; TestResumeLaunchExactProfileMatrix and TestSpawnStartsPairAndRecordsTheActor expected the old exact environment without the intentional PAIR_STARTUP_DEADLINE entry. These are reproducible branch defects, not flakes.

Audited all four sources: couchcore/startup_budget.go, couchcore/startup_progress.go, launcher/startup_budget.go and startupbudget/budget.go carry deadlines/progress via existing owners and construct/name no artifact files. Added exact NonArtifactSources entries; exhaustive artifact test GREEN2.516s after recorded RED. Reproduced both env tests RED5.340s, then require exactly one canonical deadline selected in the shared default interval during the convenience launch call, appending that validated exact entry while retaining ordered equality of the complete environment. Both tests plus the explicit caller/shared-budget control GREEN1.664s. No production timeout/path/authority change. Refreshed only their audited fixture-local reader certificates; fast inventory and diff checks pass. Native conformance from b98b3752 remains applicable because these changes are tests/inventory only. Corrected pushed head still requires a fresh recorded pass before close/landing.


### 2026-10-22 — corrected recorded suite passed; close submitted

Corrected head961a1f58d81c326b50b4726928f0f7fe4f55f779 passed full recorded suite attempt e33828c37ba00eb2c2c83a139f4d0dce / lease-126363276, exit0 in10m48s, log .git/sdlc/local-checks/961a1f58d81c326b50b4726928f0f7fe4f55f779/suite.log, SHA prefix737664b19b64. The runner completed normally, not lost. All previous inventory and env-oracle failures are fixed; no flake registration, status override or unchanged blind retry. Native Claude/Codex cold acceptance and exact cleanup passed as recorded above. Implementation/verification checkboxes now complete; fresh boundary review and normal merge remain required. Hub owns project and #739 staging/live rollout.


### 2026-10-22 — close round1 REWORK; three consumer-class corrections

The close review completed REWORK, so no completion card or merge was started. BR-1: public listed-terminal attachment bypasses launchTrackedThread and still used an untransported relative10s helper budget. BR-2: local menu progress showed operation age next to the current stage, although remote receipt progress used stage age. BR-3: the general warm-adoption fixture's200ms propagation sentinel now armed a real startup timer before preparation, preventing some intended causal failure routes. These are in-scope coverage gaps in the approved warm-attach/progress/failure-fixture sweep; no review bypass or premature #801 branch.

Public entry regression first reproduced missing transported deadline and helper10s versus remaining28s after virtual preparation (RED0.286s). An initial cleanup-oracle error incorrectly called Cancel after acknowledgement and was corrected/repeated before production changes. Public attachment now selects one deadline before its helper preparation and existing retry loop, caps the existing short query without renewing it, transports the same absolute deadline, and passes remaining time to acknowledgement/confirmation. The helper's original interactive context remains separate. Retention preparation is startup-bound; its existing finalizer remains independently cleanup-owned. Added virtual delayed acknowledgement and exact final-deadline checks; focused warm/public admission cohort initially GREEN4.525s. Final cross-class verification still pending.

Controlled console regression RED0.245s:20s preparation plus5s readiness rendered25s. Pure progress now retains stage-start time, preserves it on same-stage reports and resets only on a stage change; row and notice render the same age, with operation-age fallback and stale/terminal fencing. Targeted/race console cohorts GREEN0.326s/1.846s.

Warm-adoption delayed-preparation controls reproduced the old200ms fixture causing acknowledge-time expiry rather than the named injected failure (RED2.093s). General fixture no longer sets the propagation sentinel; dedicated budget controls own their own clock/injection. Original warm/retire/ownership cohort GREEN6.650s and delayed-control GREEN1.586s before the public-attach correction. Public attachment's now-correct remaining duration additionally exposed an old exact200ms assertion (observed197.948ms); update that dedicated control to virtual time and deadline consumption rather than a renewed relative budget. No production cap increase or sleep. Corrected recorded verification and fresh review are required.


Retention sentinel follow-up RED0.221s observed199.111ms instead of renewed200ms. Both dedicated controls now use synctest: consume virtual50ms preparation from the original200ms selection, assert identical absolute deadline and exact remaining duration, then causally expire before attachment/ownership observation. All retention controls GREEN0.928s. These are controlled fixture budgets, not production cap increases. Final full core/tty packages and race controls are running; no pass claimed until returned.


### 2026-10-22 — round1 corrections fully qualified; new recorded check/review due

Corrected full Couch core package PASS308.418s and tty PASS51.055s, including the warm-adoption cases the reviewer found. Focused race controls PASS core4.530s/tty1.580s; full repolint15.432s/artifactpath13.395s, fast reader inventory and diff checks pass. No hidden short timer, test skip, cap increase or flake classification. Public warm helper now transports the exact original deadline; virtual2s preflight plus1s delayed ack consumes the original30s budget, leaving27s with no client-context cancellation.

One new native matrix on runtime head88f7326491b1daf012aebee461df8f39f1b9184f passed131.25s (whole134.961s), exit0. Existing shared TestDisposableCouchFleet covered Claude/Codex steward+3workers, native ordinary delivery, detach, supervisor restart and all4 actors returned, then exact owned cleanup and joined child/IO for each. Claude returned59.652s, Codex52.307s; load1 before23.63/after24.02. Capture workshop/evidence/000773/native-warm-88f7326491b1.log/json; log SHA25697403cefee196bdb7533a924d162a14562d0feb405f2e640bf4cb8093f48f679. Runtime matched the committed head; only atlas documentation and the in-progress evidence file were uncommitted during the invocation. No native retry, live fleet or pair:0 change.

BR-1/2/3 are submitted as addressed for fresh gate verification, not self-disposed in the ledger. The old passing recorded suite predates these runtime fixes; a new recorded pass remains mandatory before landing. Queue it on this corrected pushed head, with review allowed to overlap per tl-worker. On any review code correction, cancel only a superseded queued check through SDLC and verify the final corrected head.


### 2026-10-22 — final branch suite passed; landing integration conflicts

Final corrected branch check attempt4c8e890e44b09f6a184c006d4f3b5c55 on f3c423c6462058da7433262a1617a9fb42ab3386 completed the full suite, exit0 in11m22s, log SHA prefixe06e56c2e7f4. Close round2 was SHIP with BR-1/2/3 disposed. The original merge runner survived the Couch restart and entered its landing turn; automatic integration then stopped on atlas/index.md and artifactpath/manifest.go conflicts. Manual integration preserves both the main atlas additions and #773 startup mapping, and unions #758 endpoint metadata exclusions with the four #773 deadline/progress exclusions. No runtime policy change in either resolution. Integrated-head full recorded verification and narrow conflict re-close remain required; no landing claim. #801 stays claimed pending this checkout release.


### 2026-10-22 — #753 related Codex steward capture preserved

Authenticated pair:9 receipt60e2009a supplies the complete disposable-fleet capture for source011cdcbf62d9169e63ce509ee976bbf98bb72548. Copied byte-for-byte to workshop/evidence/000773/pair753-011cdcbf6-steward-startup-failed.log, with adjacent provenance JSON; verified SHA25629dec23f55757b4c79970db1033143a6bbf21053763b394d919d82c2b27c43b0. Claude cohort, ordinary delivery and detach/restart passed first. Codex steward then exceeded its context; agent-ready, wrapper-startup log and startup records were absent (startup_sources0). Both cohorts proved all owned generations absent and direct child/IO joined. Full native test failed235.72s; Codex subtest131.61s. Peer made no timeout increase or retry and reports hub classification as #773 evidence. This is a related missing-ready observation, not proof that the fixed10s birth clipping caused it or that this branch resolves its unknown cause. The original source log remains untouched; integrated recorded check lease-1865554150 keeps its existing queue place.


### 2026-10-22 — #773 integrated suite occurrence

Recorded attempt1d8fae6745226f7d80414d02eccb7c87 on6d654228d39eb7157597a2a7dab85f1a30cab026 completed exit2 in2m58s after the Lua test target reached tests/nvim-route-poke-admission-test.sh. run_headless reported TIMEOUT after30s with no nvim output. Complete failed log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/1d8fae6745226f7d80414d02eccb7c87.log; SHA256bb89a55b7ae5c3f2727a8bed8b76873685c9c553fb6ec4de2a9a992f0e5a0573. This is another occurrence of #685, not a new issue. The route/poke Lua, shell driver, fake-zellij and run-headless sources are unchanged by #773 compared with integrated main. One unchanged focused command bash tests/nvim-route-poke-admission-test.sh returned exit0 and nvim-route-poke-admission-test ok. Observed load1 at that repeat end34.61 (not the original failure's load). No timeout increase, test skip or production fix. The recorded classification is unattributable-failure because the shell timeout has no Go test identity; it is not eligible for exact-package flake resume.

Hub725b1995 authorized one fresh recorded full suite based on the existing #685 failure and unchanged focused pass, not a blind retry. Exact failed head6d654228 was refused because it is no longer an origin branch tip; no rewrite or fabricated ref. Hube962edf2 authorized the current pushed tip0959aace92f7ea9b3b69b327cbe6a20a23455cd0 instead: that equals6d654228 plus only the #753 log/provenance and #773 Log, with identical runtime and tests. Fresh suite started on that tip; its result is pending. The previous failed Make stopped at test-lua before Go packages, so no completed full integrated-suite pass is claimed.


### 2026-10-22 — authorized integrated-runtime rerun passed

The single hub-authorized fresh recorded suite on0959aace92f7ea9b3b69b327cbe6a20a23455cd0 completed exit0 in12m6s: attempt4b42e6ed7c8af4526a196f8a61977aa8, lease-3589796458, saved .git/sdlc/local-checks/0959aace92f7ea9b3b69b327cbe6a20a23455cd0/suite.log, SHA prefix4b8660d9f349. This head equals integrated6d654228 plus only evidence/provenance/Log, and the only later branch change is this issue Log. The route-poke shell passed in the full run and every Go package completed; no skip, timeout change, manual status, flake identity invention or rerun beyond the one authorization. The earlier #685 failed attempt remains retained and linked above. Both additive conflict resolutions now have full integrated-runtime verification; narrow re-close and normal merge follow.


### 2026-10-22 — noon freshness gate required newer main

Narrow close refused before review because this branch lacked code landed by noon (main fb79d27bd), including the shared zellij-query consolidation across launcher and other callers. No verdict or completion was produced. Followed the integrate-gate recovery without a waiver: git merge origin/main completed without conflicts; main changed74 paths. Previous full-suite pass remains valid only for the previous integrated runtime. The newly integrated runtime requires a fresh pushed-head recorded suite, followed by narrow conflict re-close. No #773 design delta or failure reclassification.
