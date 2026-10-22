---
id: 000778
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '825df4608714fe7adeaf35bb6645bf1f0ab64884' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T18:23:34-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/couch.TestBareCouchInstalledCommand]
flow: {kind: quick, provenance: inferred, spec: "ba2c59cf", done: "855df120"}
actual_hours: 1.11
---

# flake: TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/couch.TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub: Exact source3279e1e9ec6ccbddba0b50193f5c8f72c11f4b22: train238 suite lease-1293856745 failed cmd/couch TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub after15s with live/contested terminal 📁1-1; log .git/sdlc/test-stats/inflight/de3d7e12cd48155a874dbd8db94ac3f6/run.log. On the unchanged source, env -u PAIR_TEST_BINARY go test ./cmd/couch -run ^TestBareCouchInstalledCommand$ -count=1 -timeout=90s -v passed11.305s (PTY2.12s). This fixture has private socket roots but fixed terminal name; SessionServers filters only session basename, a lead not yet a forced reproduction. Production scribe is unchanged. Track this intermittent fixture refusal separately from #768.

## Spec

Fresh private identity authorities each start at C=1; their first terminal is therefore the same 📁1-1 even when socket roots differ. The launcher deliberately uses a host-wide name census. Preserve that refusal and give fixture authorities distinct high-range store IDs instead (ARCH-PURPOSE). Both installedEnv and testcouch.allocateRoots construct this class of authority; initialize both through one testpair helper (ARCH-DRY).

The helper creates a new private registry exclusively, seeds its first C from the live test-process PID plus a bounded atomic fixture sequence, and leaves enrollment and N/M allocation to IdentityStore. Each fixture owns one store. PID supplies disjoint concurrent-process namespaces; sequence distinguishes fixtures in one process. Reject invalid PID, sequence exhaustion and existing registry rather than overwrite. This is concurrent-fixture uniqueness, not permanent identity: crashed residue/PID reuse still encounters the unchanged fail-closed census. Existing high-range fixture precedent stays disjoint. Do not weaken launch admission, add retries or change readiness budgets.

Alternatives: socket-only isolation is insufficient because census is host-wide; serializing suites would hide the shared-name defect and reduce capacity. Decimal temp suffixes alone do not guarantee uniqueness across temp parents. The OS process incarnation owns the namespace for the lifetime of its fixtures, whose existing cleanups join children before releasing their roots (ARCH-ORDER/FUNERAL). The private seed file is one bounded record in the already-owned identity root and is removed with that root; no new shared state or production format changes. Work is constant per fixture: one sequence reservation and one exclusive seed write; normal allocation is unchanged (ARCH-CONSTRAINTS/PURE/EVOLVE). Existing stateful native fixture and shared disposable harness remain the external seam (ARCH-MOCK).

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.
- Two actual independent installed environments allocate distinct C/session/tag through the real allocator; first N/M=1 and later N/M advance under stable C. Equivalent shared-fleet authorities also remain distinct.
- The existing installed PTY launch passes with an expectation derived from its allocated C. Seed validation, no-overwrite and concurrent fixture coverage pass; an AST catalog rule catches omitted seeding at both constructors.
- Recorded full suite and fresh close review pass; retain original failures and subsequent passes without treating a single native pass as proof of absence.

## Plan

- [x] Reproduce independent installed-environment name equality deterministically; retain the saved #775/#757 failure references and source trace.
- [x] Add one exclusive fixture-authority initializer, apply it to installedEnv and testcouch roots, derive the installed launch expectation, and enforce both constructors in Pair's lint catalog with positive/negative tests.
- [x] Run targeted allocator/constructor/PTY/lint checks and inventory preflight, and record a passing full suite. The SDLC close/landing receipts then own closure and retirement of the flake registration.

## Log


- 2026-10-22: closed — BR-1 addressed: corrected integrated final-code suite PASS12m49s, exit0, on d7198a27791bfac191552e79b19ea45ed6e93f40; completed attemptfa50b4670ad83418945dc02f84c68676 copied to integrated-suite-pass.json and raw SHA25638c6ab4eddde369a20be795981ec0cd8e40f1f4b74ebf1ef89e02eaa4d7e38ba independently matches. Checklist marked complete only after this pass. All packages/non-Go groups passed; no flake rerun/waiver. Operator executable targeted PASS10.027s and installed identity/PTY PASS14.762s after exact compiler RED and minimal caller adaptation. Native Claude/Codex PASS75.199s qualifies unchanged identity/fleet behavior. Original inventory and train composition failures retained separately. No production admission/timeouts/gates changed. Active-time train-wait discrepancy filed ariadne#547; measured hours not hand-edited. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
### 2026-10-21
- 2026-10-21: closed — Deterministic independent-authority RED now GREEN through real installed and fleet constructors; exclusive high-C seed, exhaustion/no-overwrite/race and constructor lint checks pass. Existing native Claude/Codex three-worker matrix PASS75.199s with message, detach/restart and exact cleanup proof. Corrected recorded full suite PASS12m15s on f025dba4, all packages incl installed Couch and artifact inventory, no rerun/waiver. Original failed inventory attempt retained; ordinary source classification fixed. Production census, launch budgets and retries unchanged; atlas maps helper and lint. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

### 2026-10-21 — recovered evidence and bounded plan

#775 landed in train254 through PR534; its retained installed-fixture-778.log and peer757-installed-fixture-778.log under workshop/evidence/000775 preserve the original fixed-name refusal. Peer #757 unchanged-head exact-package rerun passed7.556s (receipt4a5871e0); original failure and pass remain distinct evidence. Read-only tracing confirms private host authorities default NextC=1, first M=1, and native census selects matching basename across all socket roots. The old failure does not identify its competing process; the new deterministic allocation regression will establish the collision precondition without depending on scheduler timing. Bounded independent audit found the same constructor class in testcouch and the cross-parent weakness of decimal suffixes. Quick-flow fixture repair, no public product state or gates change; post plan to hub and proceed unless objected under tl-worker rules.

## Revisions

### 2026-10-21 — concrete fixture-isolation acceptance

Expand the original root-cause Done when into deterministic allocation, preserved admission, both constructor consumers, exclusive seed safety, lint enforcement and recorded verification. Preserve the known-flake history; closing retires its existing parent registration.

### 2026-10-21 — deterministic root-cause RED

Added regressions through the actual installedEnv and testcouch.allocateRoots constructors, then real IdentityStore.Allocate. Both fail on the intended condition: independent authorities each yield C1/N1/M1 and duplicate native name 📁1-1. Installed regression exits1 in0.207s; shared-fleet regression exits1 in0.224s. This reproduces the precondition independently of native scheduler timing. Existing admission checks remain unchanged. Helper implementation and pure lint rule are delegated as separate bounded file-owned tasks; main owns consumers and integration verification.

### 2026-10-21 — constructor and executable GREEN

Shared initializer and its tests now cover exclusive creation, preserved preexisting residue, permissions, directory errors, PID/sequence bounds, permanent exhaustion, 8/16 concurrent fixtures and real allocation with stable C/advancing N/M. Reviewed the pure maximum and corrected one edge: the MaxUint64 seed must refuse because the real allocator cannot enroll from it; the focused regression went RED then race GREEN. Both constructors now seed their private authority. installedEnv returns its seed C so the existing executable PTY assertion derives the exact expected Pair tag via FormatPairTag instead of hardcoding C1. The combined installed allocation/PTY parent tests PASS13.415s; fleet factory tests PASS0.247s. Pure constructor catalog cases PASS, with exact helper-import/alias resolution and shadowing negatives. No production census or timeout changed; full-package checks and inventory remain due.

### 2026-10-21 — package and inventory verification

Full targeted packages PASS: testpair5.666s, testcouch20.105s, repolint6.239s. Staged tracked-source inventory correctly named the two existing changed reader audits; their inputs/ownership are unchanged (main_test remains local/literal fixture; environment_test remains non-Go full fallback). Regenerated only those two per-file certificates using the printed record commands, then fast inventory PASS. No .sdlc or workflow edits. The shared harness now receives longer high-range session identities, so run its existing opt-in Claude/Codex native matrix on the committed source before the full suite and close; this qualifies the actual socket/actor paths as well as pure allocation. No duplicate harness, timeout adjustment or live-fleet operation.

### 2026-10-21 — native two-personality qualification PASS

Committed sourcebf7be2e2e6b1371df6182287c83f3ddd2aa948b1 passed make test-couch-fleet in75.199s (Claude34.43s, Codex31.36s). Each genuine native fleet registered its steward and3workers, delivered/submitted the ordinary message with accepted-body proof, detached/restarted with all4actors returned, and proved owned generations absent plus joined child/IO cleanup. Retained completion tail and pinned manifest are workshop/evidence/000778/native-*; completion-tail SHA256c983b14df2c7a3ccc64e003aa97486480b7b12d67299c3d82af03c151cb6e739. This qualifies the high-C identity/session paths through both personalities without any live-fleet effect or retry. PR536 opened; full recorded suite and close/landing remain required.

### 2026-10-21 — final-source recorded suite queued

Recorded suite lease275330665 is retained on pushed sourceec3c6714 (PR536), attempt0afdfcfe7e70e8b1e2cd8bd438e8dc1e; fresh checkout /Volumes/TestScratch/local-checks/sdlc-local-4115531898. Initial queue position5, no execution yet. Native matrix and affected-package/inventory checks are green. Keep this runner; wait for its complete result, finish the final Plan checkbox, then close and merge without chasing newer main. No continuation/restart while this check is queued or running.

### 2026-10-21 — complete suite failure retained and inventory correction

Lease275330665 acquired after2792s and finished11m57s, exit2. Complete log confirms exactly one failing test: artifactpath.TestProductionArtifactReferencesAreExactlyClassified, because the new non-test testpair/couchidentity.go was absent from its exhaustive source inventory. All other packages and non-Go groups passed, including installed cmd/couch, testpair, testcouch and repolint. The post-run flake-registry fetch separately failed DNS resolution for github.com, so SDLC recorded known-flake-rule-unavailable; this does not explain or waive the deterministic test failure. Do not run resume/register this as a flake. Raw log and pinned manifest are suite-inventory-failure.*; SHA256a662f00e110fac06295153cc8cf469dddfefb51834bbfc5c16ac79aec825decc.

Audit: the helper writes only its private test identity authority, just like the already-classified couchidentity store, not a Pair artifact family. Add it to the ordinary NonArtifactSources inventory; preserve exhaustive scanning and the known-violation quarantine. Verify the failing guard directly, then all relevant packages and inventory before recording a fresh suite on the new pushed head. No behavioral change to the qualified native helper; the native source evidence remains valid for this inventory-only correction.

### 2026-10-21 — ordinary inventory correction verified; DNS blocks publication

After the one-line NonArtifactSources classification, all relevant packages PASS: artifactpath16.042s, testpair11.379s, testcouch27.901s, repolint11.911s. Reader inventory and diff checks PASS. Git diff against the native-qualified source confirms no changes in the helper or either constructor, so the Claude/Codex native evidence still measures the same behavior. The full failed run remains retained; require a new recorded suite on the corrected pushed head. A separate git fetch also failed with Could not resolve host: github.com; an outside-sandbox fetch is checking whether access policy explains it. No passing suite or publication is claimed.

### 2026-10-21 — implementation checklist clarification

The original final Plan row included "close, land, and retire the flake registration through close" after full verification. Those are subsequent SDLC receipts, not pre-close implementation tasks. The active row now ends at the passing full suite and explicitly leaves closure/registration retirement to the binary's close/landing receipts. No acceptance or implementation scope changes; the original text is retained here to preserve the revision.

### 2026-10-21 19:40 — hub-directed DNS hold

Authenticated TL receipt65adc077 confirms machine-wide github.com DNS failure from a stuck Mac resolver; the operator owns recovery. Hold corrected sourcefa165827a45bacaa8954ea19d8a90d544c8f4c25 locally until the hub says Git works. The outside-sandbox fetch had already completed with the same DNS failure before this direction; no machine settings or resolver changes were made. All own recorded/targeted/native runs are finished; no live runner or landing ticket is retained for #778. PR536 remains at the earlier pushed head.

Resume: confirm the hub's recovery notice, publish the corrected local head with sdlc pr, run a fresh recorded suite (the old suite was a deterministic inventory failure, not a flake), tick the final Plan row only after its pass, close and merge. The native Claude/Codex pass still qualifies unchanged behavior. Do not start #772 on this branch; it follows #778 landing. #775 is done/archived after train254/PR534, reports delivered and grant revoked. Previous recovery screen remained unavailable; issue/Git/tool evidence supplied continuity.

### 2026-10-21 19:58 — DNS recovery and fresh suite

Authenticated TL receipt9b51cb83 reports operator DNS-cache recovery and instructs resumption. sdlc pr successfully published corrected checkpointf025dba400e49af9c6a6d6a4beda4cbc8aba5cf7 to PR536. Fresh recorded suite lease2286202273, attempt54cda992757ffb884f353a200db8f185, is retained on that head; checkout /Volumes/TestScratch/local-checks/sdlc-local-2794522815, initial position2. Old failure remains preserved and no flake resume was used. Wait for this complete result; do not duplicate/cancel or restart while it is queued/running. Close after a full pass; then enter merge promptly.

### 2026-10-21 — next-item priority updated

Authenticated TL receipta5d4365f directs returning immediately to owned #722 (fleet drain, PR477) after #778 lands, and reporting its recovered state. This supersedes the earlier next-item order #772/#740. Keep finishing #778 and its retained fresh suite first; do not switch branches during this check or landing.

### 2026-10-21 — corrected final-code full suite PASS

Fresh lease2286202273 acquired after1200s, then full suite PASS12m15s, exit0 on f025dba400e49af9c6a6d6a4beda4cbc8aba5cf7, recorded sdlc-local/suite. Raw log SHA256a103d6cf241422be1216b2ed9b1b4808b9b4eedf4bd43fbb51c144925a69c9b0. All packages/non-Go checks passed, including cmd/couch7.064s and artifactpath5.017s. No flake rerun or waiver; original failed attempt remains retained. Only issue notes changed since tested source. Together with native Claude/Codex PASS75.199s and targeted race/allocator/lint checks, the implementation verification checklist is complete. Run close now, then merge promptly on SHIP; merge owns current-main integration. Next item is #722 per the latest hub instruction.

### 2026-10-21 23:46 — train261 composition failure; waiter stopped

Train261 composed #777/PR533 and #778/PR536. Shared suite failed10m45s on the deterministic compiler error at cmd/couch/session_sync_operator_test.go:28:41: the new #777 caller uses installedEnv as a single value while this issue returns ([]string,uint64). Read the full raw log directly; retained train261-compile-failure.log/json has SHA25694d3e0b1718097a7832298389d2d02f3a312c13205ffa6c1ecfef51ae47f360c. No flake resume/registration/waiver applies. Main baseline subsequently passed10m5s; the #777-only prefix passed12m13s and is pending train PR539 CI, with neither car landed as of this checkpoint. Those two passes are authenticated conductor reports, not local #778 passes.

Hub ae76eeeb requested dropping #777, but this pre-control conductor refused because train-control.json does not exist. Conductor relayed hub disposition5d1d252c to retain its normal baseline/split path without intervention. Original #778 waiter PID51323/session3122 has now exited1: held unverified, suite log has incomplete failure attribution (cmd/couch build failed). No duplicate runner remains. Wait for #777 landing, integrate only landed main, reproduce compiler RED, adapt the new caller, verify targeted cmd/couch plus inventory, then renew recorded suite/close and merge. No source or gate changes in this checkpoint. #722/PR477 remains next after #778 lands.

### 2026-10-21 23:52 — landed caller compatibility RED to GREEN

Authenticated TL67ee3c60 explicitly directs landed-main integration, caller adaptation, targeted checks, fresh recorded suite, re-close and merge. Reopened through set-status working; change-code retained quick flow. Integrated fetched main with #777 and its archive; no unlanded commits imported. On this integrated branch, go test ./cmd/couch -run ^TestSessionSyncOperatorExecutable -count=1 fails at the exact retained single-value compiler error. Adapted newOperatorExecutable to env, _ := installedEnv followed by its ordinary struct construction. Every current installedEnv caller now handles both returns; no allocator/admission/native-harness behavior changed.

Existing operator unavailable/lifecycle executable tests then PASS10.027s, including all four unavailable subcases and moved/retried lock lifecycle. Installed identity isolation plus actual PTY parent PASS14.762s; fast affected-reader inventory and git diff checks PASS. The first combined targeted expression selected only the installed/identity cases because the operator names carry suffixes; the explicit prefix command above ran both operator top-level tests with verbose evidence. No additional fake or redundant compiler test added: these existing executable tests are the regression. Fresh recorded full suite and renewed close remain required; former branch pass and native behavioral evidence remain historical, not a substitute for the integrated suite.

### 2026-10-21 23:54 — corrected integrated suite retained; renewed review

Published final code d7198a27791bfac191552e79b19ea45ed6e93f40 to PR536. Fresh recorded suite lease3582875921, attemptfa50b4670ad83418945dc02f84c68676, checkout /Volumes/TestScratch/local-checks/sdlc-local-2671769656, is queued position1 behind ariadne:1. Keep session92771; no duplicate or restart while queued/running. Renew close review during the queue wait under tl-worker explicit overlap rule; the recorded pass on this final code still must precede landing, and any review code fix requires re-verification. No final suite pass or landing claimed yet.

### 2026-10-21 23:57 — BR-1: integrated verification explicitly pending

Renewed review independently passed affected packages, installed PTY/pipe refusal, constructor isolation, operator executables, focused seed races and reader inventory, with no code finding. It returned REWORK solely because final integrated source d7198a27 still has an inflight suite, while the Plan verification row retained its earlier checked state. Uncheck that row until lease3582875921 completes; the older f025dba4 full-suite pass remains historical evidence only. This is the completion-requires-verification family: both the active checklist and final-source acceptance clause remain outstanding. No new implementation behavior or test-head change. Record the completed matching receipt/log hash before claiming addressed and rerunning close.

The close timing report adopted4.41h, including a dominant213.9m segment containing about187m of retained train waiting and periodic status polls. Filed ariadne#547 unclaimed to investigate conductor-owned wait attribution; do not hand-edit measured actuals or infer that all213.9m was pure waiting.

### 2026-10-22 00:11 — BR-1 evidence complete: corrected integrated suite PASS

Retained lease3582875921 completed exit0 in12m49s (receipt768.631s) on final source d7198a27791bfac191552e79b19ea45ed6e93f40, attemptfa50b4670ad83418945dc02f84c68676. Admission output reported358s wait; completed receipt records359.209s queue time. All packages/non-Go groups passed, including cmd/couch7.676s, artifactpath5.751s, testcouch22.596s, testpair6.708s and repolint10.343s; no flake rerun/waiver. Completed receipt copied to workshop/evidence/000778/integrated-suite-pass.json; raw-log SHA25638c6ab4eddde369a20be795981ec0cd8e40f1f4b74ebf1ef89e02eaa4d7e38ba independently recomputed and equals the receipt. Only process/docs artifacts changed since that tested code.

BR-1 completion-requires-verification is now addressed: the active checklist is checked only after this matching final-source pass, the final-source acceptance clause has its successful receipt/hash, and prior failed/older passes remain distinct history. No behavioral source changes in this review round. Re-close now, then merge promptly on SHIP; do not chase newer main manually. #722 remains next.
