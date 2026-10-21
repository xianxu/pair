---
id: 000775
status: codecomplete
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '2bcb7c45668986a09b18d923ecfbf8fd886029c9' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T10:56:40-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "db8c8a6c", done: "1839b87d"}
actual_hours: 1.75
---

# Standalone Zellij false-miss reproduction

## Problem

The operator needs a small upstream-ready reproduction of Zellij 0.45.1 false session misses from #761, without Pair or Couch.

## Spec

Add scripts/repro/zellij-false-miss.sh and a ten-line README section. Bash plus stock shell utilities and Zellij 0.45.1 only. Create three private background sessions with temporary HOME, XDG, config and socket roots; run N raw reads with configurable parallelism (default1000 reads/four workers), mixing native pane/client/session-list reads as in #761. Count exact exit0 target-not-found plus a listing containing the target, and exit1 no-active-session while target liveness is independently observed. Preserve raw anomalies and print counts; no read retries hide failures. Trap cleanup touches only the private sessions/root. Validate one real run on this machine and report both counts. No product code or global configuration changes. ARCH-PURPOSE keeps this diagnostic separate from product recovery.

## Done when

- Standalone script and ten-line README section committed on an independent main-based quick-flow branch.
- Syntax, invalid-input refusal and one native three-session run verified; report N, parallelism, both counts and cleanup result.

## Plan

- [x] Extract private native stimulus and strict raw signature counts; document invocation and liveness limits.
- [x] Run syntax/refusal checks and one real1000-query/four-worker experiment, retain result in Log, then close and prepare PR independently of #744.

## Log

TL c4915544 dispatches this tiny tooling side task while #744 integration suite waits. Use a separate checkout so the retained #744 merge runner sees no branch or working-tree changes. Quick plan follows the exact dispatched contract; proceeding under tl-worker quick-flow authority.

### 2026-10-21 — standalone native result and slot-only continuation
- 2026-10-21: closed — Test-only producer-contract fix after SHIP: both private executables now use unchanged shared reply serializers/catalog;23 named producer cases include positives and explicit classified faults, with2 generator regressions. Independent15 repro tests,7 catalog tests, targeted Go decoder contracts and fast reader inventory PASS. Standalone script SHA256 unchanged from retained manual native2/0 cleanupPASS, no public CLI or target behavior change. Complete previous suite retained: only registered #778 parent and this now-fixed producer guard failed. Fresh final-code suite queued and required before landing; no scanner/decoder/config waiver. (fixed-then-ship) (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
- 2026-10-21: closed — Full re-close after BR-6: review complete standalone diagnostic plus TL-approved opt-in make zellij-false-miss (Python regressions then native1000/4); no default-suite hook or gate-config changes. BR-7 root README now names both entry points; all4 documentation consumers audited. All12 affected-runner tests PASS and15 repro tests PASS; explicit target native exit0 misses2, exit1/uncertain/other0, cleanupPASS. Pinned raw evidence retained in manual-target files. Final-code suite lease4033892438 queued on d9bc8999, only documentation changed since; require its pass before landing. Prior ENOSPC is infrastructure only, recipe defect fixed without bypass. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
- 2026-10-21: closed — Final standalone native1000/4: exit0 misses2, exit1 confirmed0, uncertain0, other0; cleanupPASS. Raw log and pinned source manifest in workshop/evidence/000775/decimal-*. Fifteen actual-worker/main regressions PASS, N/2N counted requests10/20 with constant lifecycle overhead; bash syntax and diff checks PASS. Whole-stderr mutation RED8 and numeric pre-fix RED17 subcases. All four README/atlas consumers swept; final recorded suite queued and must pass before landing. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

Measured from /tmp with Bash plus installed Zellij0.45.1 and stock utilities only, three fresh background shell sessions,1000 raw reads/four workers: exit0_target_listed=7, exit1_alive_confirmed=0, exit1_liveness_unconfirmed=0, other_nonzero=0. Independent raw-output classification agrees. Exit1 was not reproduced in this sample; zero is not evidence of absence. Cleanup passed (private sockets absent/root removed); a subsequent exact session-prefix process search found no server. LogSHA256a3a178d1c8660b2ce94a2e2e067d5de1b5700ee9795950f1172e0e432b49d9a9. The run manifest pins the measured script hash; source is unchanged from that run. Syntax and invalid-input refusal pass.

Initial draft setup refused before measured queries because the native socket directory is contract_version_1 rather than the release version; initial-setup-failure.log preserves it and its passing cleanup. No query failure was retried or erased. Corrected native run and ten-line README are retained. Temporary clone planning was stopped on TLbee5ced4's correction; its record was copied to scratch and the clone deleted, with no code there. After #744 landed PR509, this issue resumed through start-plan/change-code in canonical pair:5 on landed main. TL reported the draft counts received. Required suite and close/landing gates remain due; no product or local-gates configuration changed.

## Revisions

### 2026-10-21 11:52 — review corrections BR-1/2/3

Require complete stderr-byte equality for both native signatures, preserving final-newline distinctions; grep line membership is insufficient. Keep the same CLI, native query schedule, counters and private roots. Source-safe main/worker separation allows deterministic actual-worker tests with a stateful executable and real sockets, covering exact positives, extra diagnostic negatives and confirmed/unconfirmed exit1 liveness. Add top-level README and atlas pointers for this new standalone surface, and correct the original manifest's setup-failure filename. Preserve the original7/0 native sample and repeat native qualification on the corrected source. The obsolete queued suite lease199251636 was canceled through SDLC before execution; no suite result is claimed.

### 2026-10-21 — whole-stderr correction verification

BR-1's actual-worker/stateful-executable regressions pass10tests2.871s (normal make test-zellij-false-miss target). Replacing only complete-stream comparison with the old grep line membership produces8 failing subcases/0errors across both signatures: extra prefix/suffix/blank line and missing terminating newline. whole-stderr-red.log retains that mutation result. Confirmed/unconfirmed/absent-socket/malformed-layout outcomes and no-effect sourcing are covered. Tests prefer TestScratch and use a short private fallback when Unix socket path limits require it. Both predicates are the enumerated class; no product parser or recovery changed.

Unchanged corrected source b7fc2efbb4c567af641a8a8623c84b77a480166d ran the same1000-query/four-worker native schedule once from /tmp in23.172s: all four counters0, cleanupPASS, no matching zfm server afterward. whole-stderr-native.log SHA25658a703734211867cc2b5553f8177b1052e3589d1b524d2412c9b62487a22117e and whole-stderr-run.json pin that sample. Original7/0 sample remains retained; replay of all7 original raw stderr streams through the corrected exact_error predicate and independent target-token checks accepted7/7 (whole-stderr-retained-replay.json). This is variable native occurrence, not a claim that Zellij is fixed. The original native source/hash and failure sample stay unchanged. README/atlas pointers and initial-failure manifest note address BR-2/3. Queue a fresh recorded suite and re-close; no passing suite is claimed yet.

### 2026-10-21 12:16 — review corrections BR-4/5

Normalize both numeric arguments as bounded decimal strings before Bash arithmetic. Strip leading zeros, reject empty/zero/non-digit/overflow/out-of-range inputs and extra arguments before version lookup or temporary-root creation. Limits remain100000 queries and16 workers. Actual-main CLI tests reproduce the old octal and overflow defects (17 failing numeric subcases before the fix), cover both arguments, normalized headers/count files and no-effect refusal; pure boundary checks cover exact maxima without launching huge fleets. All15 tests now PASS3.872s. Counted actual-main10/20-query tests require exactly10/20 measured requests plus the same5 lifecycle calls; no timing assertion substitutes for the operation count.

BR-5 aligns the atlas and ten-line repro guide with the actual liveness predicate: a successful dump-layout containing a line matching the layout heading expression, not a complete KDL parse or a beginning-of-stream check. Swept all four new documentation consumers (root README, repro README, atlas guide, atlas index); the two pointers make no stronger claim. Source remains92 lines plus5 Makefile additions, within the quick-flow code shell. Earlier queued suite on6515126f was cancelled via SDLC after REWORK; no passing suite is claimed. Preserve both previous native samples; qualify the final numeric fix once before re-close.

### 2026-10-21 12:18 — final decimal-source native qualification

Committed source2f9215ad916f2573686e247119c16579f0997da2 ran1000 measured queries/four workers against three private Zellij0.45.1 sessions from /tmp: exit0_target_listed=2, exit1_alive_confirmed=0, exit1_liveness_unconfirmed=0, other_nonzero=0; cleanupPASS,11.832s. decimal-native.log SHA25669405cfe110d400e6bfd1d2c7d874366d746f41e4ff25d0c173008977f7a9088; decimal-run.json pins unchanged head, argv and script hash. Both observed target-listing misses retain their exact stderr/stdout. All prior samples remain preserved. This sample did not reproduce exit1; no absence claim. Final recorded suite and fresh close remain required.

### 2026-10-21 12:45 — SHIP; suite blocked by shared cache capacity

Close round3 returned SHIP with0 new findings and all5 prior findings disposed. The retained suite lease-2825741136 on pushed headb88131bf4093ef81422cd19f71f18d328863c2c0 acquired capacity after1761s. All15 repro regressions PASS2.752s inside the attempt. Before Go tests, runtimebundle generation failed writing /Volumes/AgentScratch/go-build: no space left on device. Suite exited2 after3s, unattributable-failure; no passing suite claimed. Retained complete log: workshop/evidence/000775/suite-enospc.log SHA2563f6aa0c2b6cdafdbe249937f0e730e92fe75787bb68169ee4417cbe9eced81e3 (attempt6157b34a00dabe9beffd6806f952a6df). Read-only storage inspection: AgentScratch186Gi used/100Mi available, shared go-build182Gi; TestScratch11Gi and system279Gi available.

Reported machine-capacity blocker to hub (receipt9eb51502-218b-4926-bff2-177e0a46d991), requesting shared-cache recovery or authority for isolated per-run cache. No shared data removed, no blind retry, no suite waiver. Next: once storage is recovered, rerun recorded suite on the unchanged code, then sdlc merge --yes for PR530. After its landing, fresh TL dispatch7e24351d assigns #772 (Codex InputEnded attach failure), then #740; #722 remains parked. The requested previous recovery screen is unavailable; workflow recovery was verified through sdlc state, Git and current issue/PR evidence instead.

### 2026-10-21 13:07 — infrastructure recovery and manual diagnostic recipe fix

TL5a7e4ebc confirmed shared Go cache recovery. go env GOCACHE now reports /Users/xianxu/Library/Caches/go-build; no private override/cache was created. The earlier ENOSPC attempt remains infrastructure evidence only, not evidence that tests passed. SDLC admitted the rerun without a same-head refusal (lease3886659439, headfb73ae14b7bead49e8571d4aaca77de99f280800). It ran all15 repro tests PASS5.566s, then failed deterministically in MakeEntrypoint.test_recipe_ownership: the default test-zellij-false-miss recipe lacked an affected group. The complete second failure is retained as suite-recipe-ownership.log; no passing suite claimed. Targeted recipe-ownership reproduction failed with the same missing recipe, before the fix.

TLb8fa192a approved correcting the default hook rather than changing gate configuration: this is an explicit manual native diagnostic. Replace test-zellij-false-miss and its default-suite dependency with make zellij-false-miss, which runs the Python regressions and the default standalone native sample only when explicitly requested. Keep the standalone Bash CLI unchanged and the guide at ten lines; document the manual target. No .sdlc files, affected ownership groups, consumer audit or protected configuration changed. Verify the recipe inventory, run the explicit target, then record a fresh suite and narrow re-close for this post-SHIP code change before landing.

### 2026-10-21 13:09 — manual target verified

After the default-hook fix, all12 affected-runner tests PASS8.644s, including the pre-fix RED recipe-ownership check. Explicit make zellij-false-miss on source5c77bd2ee8ae0b6b423ed22be7e669d7e5de26a7 PASS9.739s: all15 regressions PASS3.244s, then1000 native queries/four workers/three private sessions, exit0_target_listed=2, confirmed/unconfirmed exit1=0/0, other_nonzero=0, cleanupPASS. manual-target.log SHA2568f1b1967c44984972f0d5b65be2240b2b66ebf68f93c2795324db75e179ee06d and manual-target-run.json retain exact head and the unchanged script hash. Full suite and the required narrow post-SHIP re-close are still pending; no passing suite claim.

### 2026-10-21 13:12 — full-close requirement and complete entry-point documentation

The post-SHIP narrow review returned REWORK: BR-6 requires a full close because the explicit combined regression/native target expands the earlier reviewed entry point, even though TL approved it; BR-7 requires naming the new target in the top-level README. The earlier 13:07/13:09 instructions to narrow re-close are superseded: run full sdlc close on the final branch patch. The CLI contract and script are unchanged. Both runnable entry points now have complete consumer coverage: root README names standalone Bash and manual Make target, ten-line repro guide names both, atlas guide explains the target's two commands and exclusion from default tests, and atlas index links that owner guide. No other entry point is introduced. The existing suite lease4033892438 remains queued on d9bc8999768f3d49bad8cb003b6e7bf51672845c; these follow-up changes are documentation/review evidence only, so retain its FIFO place.

### 2026-10-21 14:51 — complete suite result and producer contract correction

Retained suite lease4033892438 acquired after5536s and completed10m58s on unchanged coded9bc8999, exit2. Non-Go groups passed; exactly two Go failures: parent cmd/couch.TestBareCouchInstalledCommand (known flake#778), and zellijtest.TestEveryZellijFixtureReplyUsesProductionDecoder (our new Python test executable is an unregistered native-reply producer). All other packages passed. Full log retained as suite-fixture-contract.log SHA2567d0c143675766058d28e70d5cc9cf4f513a11ee2d18d7d8e2053b545c9968d73; suite-fixture-contract.json pins the exact source/lease/attempt. The #778 raw parent extract is installed-fixture-778.log SHA256a256bcb9140ddef21d6e03f9cbdf5ecb21f6ef6831c545269004c29c9b106897, for attachment when that issue starts. Its message is terminal name still has a live or contested server for the fixture's fixed 📁1-1; the namespace/name-only census remains a lead, not a proved cause. SDLC correctly refused the unknown producer-guard failure and did not rerun #778 over it.

TLfe6fe0de confirmed #778's parent registration and assigned it after #775, ahead of #772; then #740. Pair:2 b9ac0c22 confirmed same-source fail/two targeted passes and requested the retained diagnostic on #778. Preserve the known registration; no manual blind retry. The current correction will make both Python executable consumers use the existing shared reply serializer/catalog with producer-specific generated positive and explicit fault cases, not exclude them from the guard. A bounded test implementation task owns only that Python test and serializer/catalog integration. No product script, Make behavior or protected gate configuration changes. After targeted RED/GREEN, a fresh recorded suite and close are required. The earlier queue position changes were confirmed expected priority grants by TLf6dddbbb; no grant or reorder was requested for #775. The suite survived the later Couch restart under its original lease.

### 2026-10-21 14:56 — producer catalog correction verified

Targeted production-decoder guard reproduced RED with the same unregistered producer. Both fake executables now consume the existing shared serializer/catalog: scheduled worker replies select named cases, main's healthy pane/client/session reads use semantic serializers, and23 producer-specific catalog cases retain positive and explicit fault envelopes (including version, mutation, sentinel and dump-layout output). No scanner exclusion, Go guard, production decoder or fixed-reader certificate changed. Two generator regressions pair diagnostic bytes with classifications and check healthy semantic case ownership. Production name classification accepts two trailing-newline variants that the standalone exact-byte counter intentionally rejects; the catalog records that distinction without changing bytes or weakening either consumer.

Independent verification: all15 repro tests PASS3.148s; all7 generator tests PASS; targeted TestFixtureReplyGeneratorUnitContracts and TestEveryZellijFixtureReplyUsesProductionDecoder PASS5.181s; fast reader inventory and git diff checks PASS. The standalone script SHA256 remains identical to the retained successful manual native run, so no redundant native sample is required for this test-only correction. Atlas maps the shared serializer/catalog consumer. Queue a fresh final-code suite and narrow post-SHIP fix review for this unchanged-public-contract test integration before landing. No passing full suite is claimed yet. #778 remains registered and is next after #775.

### 2026-10-21 15:03 — shared fixture evidence retained while verification queues

Authenticated pair:4 receipt6df2ba31 supplied #757 integration attempt19d704370692e1c8cbf859c2424630d7. Captured its complete #778 parent failure block before automatic-rerun retention moves it: peer757-installed-fixture-778.log SHA25614a6b19a6082893785c61309e5b240496119a5aa9bba6b0d4f318eefdcc1a4a6; accompanying JSON pins Git-verified head and identical main_test.go fixture blob across that integration and our failed suite. Both report fixed 📁1-1 live/contested refusal; neither extract identifies the competing process. The peer reported SDLC recognized the parent registration and queued its same-head targeted rerun; no final peer result is claimed.

Read-only source tracing for the next issue confirms independent private identity authorities each start NextC=1 and allocate M=1, yielding the same native name through the normal formatter; global census matches basename across socket roots. Existing fixture-private high-range C precedent is in couchcore/couch_test.go. After #775 lands, validate the current main and bring #778 a deterministic two-authority allocation regression plus a private fixture-name fix; do not weaken the global census. No #778 code or claim was started on this branch. Current #775 source correction is fix-review SHIP with all7 prior findings disposed; its final-code suite lease3734069217 remains retained on dd995d96, queued. No passing full suite claimed.

### 2026-10-21 15:18 — recovery and peer rerun outcome

Recovery screen explicitly unavailable; current Git, SDLC state and retained terminal confirm the clean issue branch, SHIP close and original final-code suite lease3734069217 still queued. No runner was restarted. Authenticated pair:4 receipt4a5871e0 reports the automatic exact-package rerun f3952ef66894eb968b05efef57a82371 on unchanged integration source passed7.556s (PTY1.72s); its original attempt19d704370692e1c8cbf859c2424630d7 completed10m16s with only the registered #778 parent failure. SDLC classified that peer suite as pass with known flake #778. This supplements the previously retained failure extract; it is peer evidence, not a passing result for this branch. Keep the current final-code suite and enter merge after it passes.

### 2026-10-21 — final-code full suite PASS

Retained lease3734069217 acquired capacity after4165s and completed the full declared suite in9m52s, exit0, recorded as sdlc-local/suite on dd995d96feffc62e3984bebaaeae9b8278ec887c. All non-Go checks and Go packages passed, including cmd/couch (5.895s, no #778 rerun needed) and zellijtest (16.474s, including the producer contract). Log SHA256bd2e218cb9f528b7861d43e06d3044615052b982cac61e089a05b779dc0edf77. No failure or waiver in this attempt. Final-code fix review remains SHIP; follow-up commits contain only close records, notes and retained evidence. Enter sdlc merge --yes promptly; it owns current-main integration and any required integration checks.
