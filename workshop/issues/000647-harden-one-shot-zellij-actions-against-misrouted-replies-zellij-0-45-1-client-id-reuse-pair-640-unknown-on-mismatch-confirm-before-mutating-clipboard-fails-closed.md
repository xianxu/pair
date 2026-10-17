---
id: 000647
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours: 7.13
collaboration-mode: collaboration
card_mirror: '0511dc148cbf33fce0806374a6fcc06aafb12477' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T11:09:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# Harden one-shot zellij actions against misrouted replies (zellij 0.45.1 client-ID reuse, pair#640): unknown on mismatch, confirm before mutating, clipboard fails closed

## Problem

## Spec

Operator decision, 2026-10-17: harden Pair and Couch against zellij 0.45.1's reply misrouting (pair#640) on our side. There's no local zellij build. Pick up the upstream fix (zellij PR #5630, "do not reuse client ids", merged but unreleased) when it ships.

Background, from pair#640: a reused client ID lets a late cleanup or reply reach the wrong connection. In a raw-request stress test about 3% of replies went to another connection. Periodic reads already recover on their next tick. One-shot actions don't. pair:1's audit on main 15d84ef7 lists them:
- layout and terminal toggles: focus or geometry from `list-panes` and `current-tab-info`;
- clipboard paste and routing, which doesn't always fail closed;
- warm resume and attach checks: `list-clients` counts;
- restart-key ownership and note anchors;
- discovery, which can silently omit a live session from a well-formed reply;
- the ownership proof's pane content.

Hardening, in this order of value:
1. A reply that doesn't fit its request counts as unknown, never as an answer: the wrong kind, a different session or pane than the one asked about, or malformed.
2. A one-shot mutation confirms with a second independent read before acting. If the reads disagree it refuses, and the user sees a refused action, not a wrong one.
3. Clipboard paths fail closed.
4. Where the protocol allows a request to carry its own correlation, use it.

No blind retries, no sleeps, no transport machinery. Reads that already re-check every tick stay as they are.

## Done when

- Each caller in pair:1's audit is classified as re-checked (no change), hardened (rule 1 and/or 2), or fail-closed (clipboard). The atlas records the table.
- A fake-zellij test per hardened caller shows that a misrouted reply of the same kind and of the wrong kind leads to refusal or unknown, never to the wrong action.
- pair:1's native reproduction, run against the real 0.45.1, shows hardened one-shot actions refusing rather than misacting.
- When zellij ships the fix, a note in the issue says to rerun the reproduction on the new version.

## Plan

Durable full-flow plan: [Zellij reply hardening](../plans/000647-zellij-reply-hardening-plan.md). R1 is proposed for TL approval; no implementation has begun.

- [x] M1 — Strict reply contracts, stateful fake and lifecycle/absence admission.
- [ ] M2 — Go one-shot action and clipboard safety, with per-caller fake refusal tests.
- [ ] M3 — Editor admission, native refusal qualification and complete atlas caller map.

## Log

### 2026-10-17
- 2026-10-17: closed M1 — M1 reviewed from an independent clone while the registered checkout lands #640; clone lacks the primary checkout's queue evidence. TL approval0019655b authenticated. BR-4 collector-error controls RED before and GREEN after both-consumer fix; launcher34.936s, collector4.287s, collector/observer race1.930s. Prior BR-1/2/3 disposed in round2; R3 phase/lifetime controls retained; full-Couch evidence limits in issue.; review verdict: SHIP

### 2026-10-17 — full-flow design R1

Claimed and started on a fresh main-based branch, separately from #640's documentation-only closeout. Applying the writing-plans skill; the durable plan classifies every audited caller, preserves periodic/advisory policies, defines a fixed two-observation admission with no retries/sleeps/transport additions, and enumerates deterministic fake cases plus the native acceptance oracle. Positive local sidecar authority remains a fast path. Clipboard OS mirroring of the selected text remains separate from auto-quote admission; uncertainty prevents staging/focus/trigger effects.

R1 explicitly asks the TL to acknowledge the protocol limit: no native request nonce is available, and two identical valid stale replies cannot be authenticated merely by comparison. Wrong kind/context, required-field absence and observable disagreements refuse; local identity and absence fences remain independent. No claim of universal protection from arbitrary correlated false evidence, and no native mutation replay. ARCH-SECURE/ORDER shaped that boundary. No estimate is derived until plan quality passes.

Upstream follow-through: [zellij#5270](https://github.com/zellij-org/zellij/issues/5270), [#5382](https://github.com/zellij-org/zellij/issues/5382), [#5546](https://github.com/zellij-org/zellij/issues/5546), [PR#5630](https://github.com/zellij-org/zellij/pull/5630). When a release includes retired client IDs, rerun the retained raw reproduction, native refusal/healthy controls and original actual-chord conformance before revising compatibility assumptions. No local native build or upstream posting is authorized.

### 2026-10-17 — R1 approved; entering plan quality

Authenticated TL approval accepts the three milestones, limited guarantee and clipboard boundary. Healthy toggle/paste latency must be measured; visible regression returns to TL rather than raising deadlines. No production implementation yet. #640's independent diagnosis suite continues on its pushed documentation-only head while this branch enters the full-flow gate.


### 2026-10-17 — plan-quality R1 refused; R2 prepared

The first plan-quality gate returned PQ-1/PQ-2/PQ-3 (named function test strategies, enforced total deadline, visible refusal). Source checks confirmed all three: default layout/clipboard/Lua reads do not propagate a shared deadline, and detached/shortcut stderr is discarded. Appended R2 to the durable plan with function-level strategies, one five-second total admission capped by earlier deadlines (restart remains one second), context propagation and joined query lifetime, shared auto-quote preflight before any flash/stage, and existing wrapper notification-broker presentation. Source audit also corrected ordinary warm attach versus explicit continuation adoption and native nullable/enrichment/identity schema facts. Await TL on these concrete refinements, then rerun the remembered gate; no estimate or implementation yet.

Parallel closeout: #640 PR410 documentation suite is running on its unchanged pushed head in its own detached worktree (local runner session 95007); no native fixture is running. Fresh TL guidance authorizes --no-actual only if close names a legacy suite attempt lacking exact queue endpoints, with that attempt ID and ariadne#420 calibration exclusion reason. Do not reconstruct legacy timing evidence.


## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only. Calibration source is flagged stale by estimate-source; values are provisional, not a predetermined total.

Decomposition: one pure contract module; one stateful external API double; eight existing Go consumer packages (launcher, couchcore, layoutcmd, termcmd, draftroute, clipcmd, couchcmd, couchtty); four Lua integrations (workbench route, poke, DOM, restore); three milestone reviews, atlas maintenance and one native qualification. A thorough approved plan gives design x0.2 and a 15% design buffer. Standard-library JSON/context short-circuits contract scaffolding, halving its design before the discount; no external library can supply Zellij-specific decision validation or the fault model. Existing runtime/module seams are familiar (1.0). All implementation values below apply v3.1's x0.4 once.

Contract module uses base design1.0 x0.5 library x0.2=0.10, impl0.6 x0.4=0.24. Fake API integration uses base design2.0 x0.2=0.40, impl1.0 x0.4=0.40. Each of eight Go extensions uses base design0.2 x0.2=0.04, impl0.5 x0.4=0.20. Each of four Lua integrations uses base design1.5 x0.2=0.30, impl1.0 x0.4=0.40. Review values are design0.05 and impl0.30 x0.4=0.12 each; atlas design0.10/impl0.20 x0.4=0.08; native discovery impl0.60 x0.4=0.24. Design sum2.27, buffered2.6105; implementation sum4.52; total7.1305 rounded7.13 hours.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.10 impl=0.24
item: api-integration design=0.40 impl=0.40
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: lua-neovim design=0.30 impl=0.40
item: lua-neovim design=0.30 impl=0.40
item: lua-neovim design=0.30 impl=0.40
item: lua-neovim design=0.30 impl=0.40
item: milestone-review design=0.05 impl=0.12
item: milestone-review design=0.05 impl=0.12
item: milestone-review design=0.05 impl=0.12
item: atlas-docs design=0.10 impl=0.08
item: real-api-discovery design=0 impl=0.24
design-buffer: 0.15
total: 7.13
```


### 2026-10-17 — M1 implementation checkpoint (not a closed milestone)

Change-code passed after CLEAN plan quality (three rounds, PQ-1/PQ-2/PQ-3 addressed) and INFO estimate quality. R2 approval authenticated as 073e88c1-0bf0-4967-8e69-b37cc6758b67. Estimate INFO warned about integration/verification optimism; actuals will remain measured rather than fitted to the estimate.

Added strict pure native contracts and shared 52-case corpus, a stateful fake CLI/model/effect log and a synchronous read-only process owner. Native-positive tests retain nullable templates, absent optional enrichment, plugin/terminal ID namespaces, tab identity and unknown extra fields. Duplicate JSON keys reject through the existing strictjson scanner. Targeted parser fuzz/race and fake transition/race checks passed; process-owner real helper regressions failed before implementation then passed normal/race tests.

Lifecycle RED proof: wrong-kind pane/tab JSON became detached clients; a changing owner was accepted from the first snapshot; listing omission freed a live server name; unknown listing retried into warm authority; a changing live-layout signature was accepted. These regressions also fail under a production-file overlay of Git-verified original base 7ee0fa320e14236d01850ed42ad67b1976752a7c (overlay retained under /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair647-red-control-0qv2tfbm). The current targeted regressions pass. Unknown first listing also now terminates before another native query.

Couch context regressions separately reproduced unknown observations recovering into successful attachment at all four fences. The fix shares query admission while preserving original helper lifetime, uses existing detached-adoption fences, and prevents query refusal entering the unrelated helper-start retry policy. Agent-targeted warm/retention controls passed, including race and unchanged ordinary warm budget (zero client queries/eight listings). All query-side blind retries were removed; ordinary successful read counts remain unchanged. Native-positive abbreviated fixtures were upgraded rather than weakening strict contracts.

Current parent verification: targeted launcher observation/lifecycle set passed before final process-owner wiring; final wired launcher and Couch warm/retention/owner reruns are in progress. Pure modules/fake race passed. M1 remains open pending those results, a coherent checkpoint and mandatory SDLC boundary review. No native conformance run or full #647 suite has started; M2/M3 are untouched.

#640 closeout: docs-only PR410 suite passed on da26fa0a8900192c0c95f0174e9fa34f57116163, exit0 in11m59s, logsha256 prefix df3e66215d0f; saved suite log in Git common-dir local-checks for that head. An auxiliary checkout was rejected as an environment dependency (a linked worktree is not an independent clone there), then removed cleanly. Close #640 from the registered pair:1 checkout after this code checkpoint; keep the successful exact-head suite record. Fresh TL permits no-actual only on a named legacy endpoint refusal, with ariadne#420 exclusion reason.


### 2026-10-17 — M1 broader-check correction and checkpoint

The first whole launcher package run failed an owner-pass scan ratchet and hit an intermittent query-group cleanup EPERM; its cancellation fixture then waited for a readiness file after the query had already failed. Captured the owned test's goroutine dump with SIGQUIT at124s (diagnostic interruption, not a qualifying pass). Fixed the scan regression by preserving the original before/after generation bracket around the native observation pair; retained the 1+2N host-table budget and doubled only intended native pane reads. The cancellation fixture now reports query completion before readiness instead of waiting to the package timeout. Native create/list/resume's abbreviated full-list/pane fixture was upgraded and its isolated control passes.

EPERM was ours to handle, not classified as a native flake: the new query-process collector encountered known Darwin empty-group semantics already handled by Couch #542. Fixed600 control reproduced1/600 (background/group PID68716, shell and Go PID identities agree, exit0); unchanged control after the narrow shared procutil extraction passed600/600 in12.212s. Logs: /tmp/pair647-zellijexec-fast-exit-control.log and -green.log; race log /tmp/pair647-zellijexec-race-green.log. Exact-group membership is read only after EPERM; only known-empty suppresses it, live/zombie/unknown remains failure. Existing Couch semantics and normal-path scan cost stay unchanged. Native kernel evidence: https://raw.githubusercontent.com/apple-oss-distributions/xnu/main/bsd/kern/kern_sig.c (killpg1 filters zombies before its no-signalable-members EPERM).

Latest targeted native-fixture/owner/scan/cancellation controls pass12.663s; broader warm/retention checks passed15.654s before the helper correction, with admission and process-owner race controls already passing. The subsequent full package and Couch reruns failed the now-reproduced EPERM, so they are not qualifying evidence. Next: one fresh full launcher/contracts/fake/process-owner run and Couch targeted run after this fix, then #640 docs closeout and M1's mandatory boundary review. M1 remains unclosed; M2/M3 unchanged.


### 2026-10-17 — M1 qualifying checkpoint verification

On code head 78da3d4c37b97cf80bce3102229d1c5704354986, fresh complete launcher/contracts/fake/process-owner package run passed: launcher55.728s, zellijexec15.996s, zellijreply0.085s, zellijtest2.431s, zellijpane0.080s. Log /tmp/pair647-m1-packages.log. Targeted Couch warm/retention/continuation/owner/pass/detached set passed28.644s, log /tmp/pair647-m1-couchcore.log. No qualifying full #647 suite or native run yet. Next mandatory M1 boundary review follows #640's already-qualified docs closeout; do not tick M1 as closed before its SDLC verdict.

### 2026-10-17 — resumed M1 inventory verification and approved M2 source context

Previous TTY capture is unavailable; recovered from Git, SDLC state, committed artifacts and authenticated current TL receipts. #640 received SHIP without findings and entered the landing line (PR410); its queued checkout remains unchanged. A temporary linked checkout of this branch is used only for Git edits and targeted verification, not SDLC transitions.

The production-artifact coverage guard caught ten new M1 source files absent from its exhaustive inventory. Added their classifications: the owner-reply projection consumes the existing typed storage owner; strict parsers, OS group observation, query collector and private test fake do not construct Pair artifact paths. Initial temporary-checkout failures also included the absent generated embedded runtime bundle; generated it with the existing make target, without tracked changes. After the classification correction, the complete artifactpath package passed20.892s; procutil passed0.553s. A complete couchcore package run under the repository's existing20m allowance is in progress; no new timeout was introduced.

TL approved the native-source-derived auto-quote context rule (39354779-66f0-4d49-9411-f7ad46f22145), now appended to the durable plan: confirm client focus plus source/draft pane role and tab together; differing client focus refuses auto-quote visibly while preserving OS copy. It confirms observed context, never historical selection provenance. M2 implementation still awaits M1's mandatory boundary review; no M2/M3 code is included here.

### 2026-10-17 — M1 simultaneous refusal/helper-exit regression

Inspection of the final warm-attach fence found a concrete error-composition hole: helper exit replaced a simultaneous terminal-observation refusal with a generic helper error. The outer helper retry then lost the refusal marker. New controlled TestWarmAdmissionUnknownSurvivesConcurrentHelperExit exits the helper inside the fourth observation and returns a sentinel refusal; RED showed attempts=2, reads=7 and err=nil. Preserve both causes with errors.Join, so helper exit cannot erase the authority refusal. This implements the approved no-retry-on-unknown rule without a new state, query, timeout or retry mechanism (ARCH-ORDER).

The new regression and all WarmAdmission/WarmAttachment/PublicWarmAttach controls pass together23.073s after the correction. A targeted race pass is in progress. The already running complete couchcore package was compiled before this correction and will be identified as that earlier-code control rather than evidence for the new ordering fix.

### 2026-10-17 — M1 integrated readiness API and qualifying controls

Warm admission/attachment/public-attach race controls passed8.819s. #640's docs-only close is SHIP with no findings, PR410; its merge remained queued third without starting integration. Cancelled only that queued merge (exit130, clean branch) so the registered checkout can review M1; retain #640's close and requeue its landing afterward. Hub report577ffa55-d346-4cf9-a672-cdd6962606f2 is submitted. The temporary #647 verification checkout remains detached at the earlier code while its complete package run finishes.

M1 milestone-close measured1.37h successfully but refused before review because this branch lacked pre-noon main code. Integrated main as the gate required. Resolve the one zellij.go conflict by retaining #645's exported ClientCountContext and #647's strict parser/process owner; remove the unused private error-discarding wrapper. Main's new park unit fixtures used bare pane IDs, so integration controls failed the one/two-client and growth cases on invalid qualified identity. Update their positive replies to native terminal_0/terminal_1 without changing assertions or polling behavior.

After integration, complete launcher passed35.385s. Couch warm/retention/continuation/owner/detached plus park client/growth/helper controls passed16.102s; targeted park/admission race controls passed2.238s. The new concurrent-exit regression, ordinary helper retry controls, no-extra-query warm budget and #645 readiness assertions remain intact. No native or whole-issue suite qualification is claimed. Next: the mandatory M1 boundary review on this combined head; M2/M3 implementation remains pending.

### 2026-10-17 — M1 round1 REWORK and class correction

Boundary review returned BR-1 (launcher verifyCouchSessionAttach still retries unknown), BR-2 (individually valid short/full listings disagree but grant authority), and BR-3 (shared terminal registration fixture violates strict full-row format). M1 is not closed. The durable plan now enumerates the complete lifecycle read/identity class and its no-new-mechanism correction. BR-1's stateful production attach regression reproduced success after wrong-kind evidence at all five launcher/exported-attach fences; removing the retry passes normal/race CouchSession controls. BR-3's existing listed-detached control and new direct listing-form regression were RED; native metadata/is_plugin fixture correction passes all shared fixture consumers4.270s and race33.953s. Main is completing BR-2 with pure typed row facts and native/cached no-authority regressions.

The earlier-code whole couchcore run failed at20m, attributable to our BR-3 fixture, not classified as machine load or a flake: TestSpawnComposesProductionPairRegistrationBoundary/ordinary spent1177.06s before reporting missing creation metadata and failed cleanup; fresh-slot also reported the same malformed listing. The package then timed out1200.277s while later cases/parallel tests remained pending. This is not passing evidence and not an unchanged flake retry. The corrected shared consumer controls above now complete promptly; rerun the full package after all round1 corrections with JSON timing retained so progress/failures remain observable.

Read-only timing overlay of the original M2 layout executor on b4236f4ef378d88e5bee66906b828d0f28f298ce passed one native TestFullscreenZellijConformance attempt12.09s without changing assertions/deadlines. Under concurrent machine load: Focus entry5 samples median320.629ms (308.567–512.710); Normal return5 median775.328ms (591.566–1233.873); native fullscreen exit1 sample161.511ms. No fault observed: healthy baseline only, not refusal qualification. Retained /tmp/p647b.mLsDQq contains overlay/source, baseline-native.log and hash/version/measurement manifest.json. An invalid overlay path and direct invocation of a non-executable environment helper were setup errors before native execution and are excluded, not hidden native retries. The actual attempt used sh plus the shared empty environment owner and a transparent exec-only wrapper for private short socket/temp roots. Fixture cleanup completed and an independent process scan found zero retained native server rows. Native window release to pair:6 is submitted (32ea52f3-e6d6-4290-8011-d7579ac2558f).

### 2026-10-17 — review corrections verified; R3 launcher budget decision pending

BR-2 RED controls showed selected-session replacement/addition granting detached or empty authority, a cached contradiction issuing a client query, and name text being trimmed/misclassified as EXITED. Reconcile exact short/full identity sets before projection or client queries; preserve native short-row query order, and derive full-row state only from validated metadata. Sweep owner presence, quiescence, reuse and handoff onto the same strict full-row facts. Their new controls also failed before wiring (foreign leading-space row admitted as another session; EXITED in a live name misread as exited). They now pass; retained tolerant cosmetic parser is unchanged. A real RunLaunch contradiction regression asserts no attachment, launch or native effects. Reordered compatible rows remain accepted. One old classification fixture omitted its foreign short row from full output; add the actual full foreign row while retaining all assertions.

Qualifying current correction checks: complete launcher33.412s (/tmp/pair647-m1-review-launcher-green.log); authority normal3.743s and race17.080s (/tmp/pair647-m1-review-authority.log and -race.log); after preserving exact native query order, snapshot/authority/CouchSession selection3.596s (/tmp/pair647-m1-review-order.log). Broad Couch warm/retention/continuation/owner/detached/park/shared-fixture/composed-launch selection passes27.136s (/tmp/pair647-m1-review-couch.log). Shared fixture consumer race passed33.953s. The earlier failed complete couchcore package has not been retried as a flake or claimed passing. The detached verification checkout was removed cleanly after its process ended; process scan found no surviving rows for that test binary/failed composed-fixture roots.

The sweep also found R2's launcher total-budget gap: selected Couch attach has three context-free liveness fences (six listing calls with fresh per-call five-second caps), then handoff Check before its bounded Admit. Couch's outer query budget is correct, but a standalone launcher needs explicit selected-operation context plumbing without capping the interactive child or a human picker. R3 in the durable plan states the exact call graph, smallest rejected per-fence-timeout alternative, proposed warm path and unresolved cold-birth phase choice. Request b33692e2-b561-4b99-b22a-b45526ae8611 is queued at ops:0 waiting for image input to clear; it is not approved or submitted yet. Budget implementation and M1 re-review wait for TL's phase decision; M2/M3 remain untouched. #640 remains SHIP PR410, not landed, with its queued merge deliberately canceled and a later requeue still owed.


### 2026-10-17 — R3 query phase decision approved

Authenticated TL receipt 7c0d5560-3f2d-40ec-b9d8-7984c09ead76 resolves the pending launcher budget question. Warm attach shares one selected-operation query context through liveness, attach and Check, with separate interactive-child lifetime. Cold pre-create and post-birth admission each have their own five-second query budget; existing birth/readiness waiting remains independently bounded and unchanged. Approved resolution and regression strategy appended to the durable plan. Proceeding RED first; no M1 close claimed. #640's merge will be requeued when this checkout is free.


### 2026-10-17 — R3 context wiring and deterministic qualification

R3 is implemented through explicit query/lifetime contexts. RunLaunch's selected Couch intent shares one bounded query context across strict liveness, name probing, owner observation and the final OS handoff; exported attach starts an equivalent decision context. Runtime requires context-aware blocking handoffs. Warm Acquire/Check/admission use the shared query context; interactive Start/Wait retain the original lifetime. Cold readiness retains its existing birth bound and post-birth native admission starts its own five-second context under the original lifetime. No retry, sleep, deadline increase or native transport was added. Atlas and function strategies reflect these phases.

RED evidence: /tmp/pair647-r3-launch-red.log showed actual RunLaunch bypassing every context-aware boundary (queries=0, admission/lifetime nil). /tmp/pair647-r3-owner-red.log showed cancellation during foreign-owner observation incorrectly returned nil. Independent handoff controls also reproduced starting after canceled admission and accepting admission canceled just before completion. Corrections check cancellation at each authority boundary; all tests use controlled contexts/channels without sleeps.

GREEN: launcher budget selection passed0.193s; launcher/CouchSession/retention/startup/handoff race selection passed3.171s. Complete launcher + reply/process/fake package control passed48.957s /0.161s /7.747s /2.390s (/tmp/pair647-r3-packages.log). Handoff-focused normal/race controls passed0.182s/1.223s. Earlier complete launcher control before the final owner-cancellation correction passed35.722s; the later complete control above includes it. Full couchcore JSON control is still running with no failure observed so far; /tmp/pair647-m1-couchcore-control.json retains per-test progress. This is a code-corrected qualification after BR-3, not an unchanged flake retry. M1 remains open pending that control and boundary re-review.


### 2026-10-17 — R3 post-birth query ownership edge qualified

Before M1 re-review, cancellation audit found that post-birth admission inherited the original caller but not the specific client attempt's exit cancellation. /tmp/pair647-r3-postbirth-exit-red.log reproduced the coordinator waiting for the query instead of promptly canceling/joining it. The OS Start seam now records its actual attempt context before Admit starts; the coordinator cancels that attempt immediately on child exit before joining admission. This keeps the post-birth query independent of the readiness timer while ending it with its own child attempt. No new API, state machine, retry or deadline was needed. Independent read-only audit confirmed the Start-before-Admit happens-before relation and absence of a field race.

The extended race selection passed1.653s (/tmp/pair647-r3-final-cancel-race.log), including child-exit and original-caller cancellation, query-ended-before-lease-release, preserved readiness independence and established-client lifetime controls. Full launcher control is running again after this narrow production correction. The full Couch JSON control has passed both formerly malformed registration cases (ordinary1.51s, fresh-slot1.73s, parent3.24s), with package completion still pending.


### 2026-10-17 — M1 re-review evidence complete

Full Couch package control passed371.419s with no failing tests (go test -json ./cmd/internal/couchcore -count=1 -timeout 20m; /tmp/pair647-m1-couchcore-control.json). It used the existing20m allowance, not a raised deadline, and ran concurrently with another full suite; this is not an idle-host performance measurement. This control started in the R3 working tree before the final post-birth cancellation correction and qualifies the BR-3 shared fixture across the entire package. That final narrow ownership correction is independently covered by the latest complete launcher package pass74.983s (/tmp/pair647-r3-final-launcher.log) and deterministic race selection1.653s. No exact-head full-suite claim is made here; the required recorded complete suite remains a whole-issue close/landing obligation after M2/M3.

BR-1/BR-2/BR-3 fixes, the R3 phase decision and cancellation ownership controls are committed. Submitting M1 round2 with those evidence limits visible. No milestone completion is claimed before the binary's fresh-context verdict and finding dispositions.


### 2026-10-17 — M1 round2 REWORK: preserve query collector errors

The binary review disposed BR-1/BR-2/BR-3 as addressed and returned REWORK for BR-4: both diagnostic-normalizing consumers can suppress a joined cleanup failure because errors.As still finds the subprocess ExitError. Source inspection confirms the class at ZellijSource.runContext and ProbeSessionNameContext. Appended the class rule, both-consumer regression strategy and minimal direct-exit classification to the durable plan. Candidate actual2.18h was measured by the binary; no milestone close finalized. Review ledger and sidecar are retained unchanged.

The registered checkout is now free of background work. Checkpointing this result and requeuing #640's already closed PR410 landing, per TL R3. BR-4 preparation can continue in a disposable independent clone without changing the landing checkout. M1 remains open; M2/M3 have not started.


### 2026-10-17 — BR-4 RED and targeted GREEN in isolated preparation clone

The clone /Users/xianxu/workspace/worktree/pair-slot1/pair647-query-fix is independent Git storage sharing only immutable objects; the registered checkout remains held by #640's queued merge. /tmp/pair647-br4-red.log reproduced six failures: collection errors were normalized to empty inventory, name acceptance or too-long evidence, and collection+cancel lost EIO. Both process consumers now normalize only direct ExitError, validate successful client tables normally, and join context cancellation with collector errors. Private per-instance runner seams call the real zellijexec.Run before injecting the collector error; no global hook, native process, sleep, retry or deadline increase was introduced. The producer's unchanged raw-exit-versus-joined-failure contract is documented.

Targeted production-boundary/legacy name/observer/attach controls passed17.256s (/tmp/pair647-br4-green.log); corresponding collector/observer race controls passed1.930s (/tmp/pair647-br4-race.log). Pure diagnostic positive controls remain accepted; collector-failed diagnostics prove neither positive nor negative name bounds. Full launcher qualification and M1 round3 remain pending. Hub coordination receipt 0fc4b53a-58bd-4a85-ae28-72e5937f8939 is submitted. #640 remains queued behind #622/#646/#620, without changes to its registered checkout.


### 2026-10-17 — BR-4 complete package qualification

The first broad clone run failed only two shared-Pair-build setup controls because its ignored runtime assets had not been generated (/tmp/pair647-br4-packages.log; TestCreateLayoutWrapperPreservesAgentCommand and TestSelectedRootNativeCreateListAndListedResume). Ran the declared make runtimebundle-generate target in the disposable clone, then the unchanged package command passed: launcher34.936s, zellijexec4.287s (/tmp/pair647-br4-packages-green.log). The targeted collector/observer race control remains pass1.930s. This was environment bootstrap, not a flake retry; both logs are retained. No registered-checkout or operator binary was built or changed.

BR-4's both-consumer class correction is ready for the remaining M1 review round. #640's merge still holds the registered checkout, queued behind #622/#646/#620. Check whether the binary admits a read-only dry-run from this independent ordinary clone; if it requires the registered checkout, wait for landing and import the committed correction as planned. No review or lifecycle transition has yet run in the clone.


### 2026-10-17 — M1 clone review and calibration exclusion authorized

The independent ordinary clone passed read-only milestone preflight; no review dispatched by dry-run. Its initial setup gaps (missing remote tracker ref and main publication upstream) were repaired with fetch/upstream configuration, without changing publication contents. The resulting measurement omitted the primary checkout's queue evidence. Asked the hub rather than adopting a reduced-evidence sample; outgoing receipt38a596f2-4fa0-4495-8780-cd8d2fff682a is submitted.

Authenticated TL receipt0019655b-05f9-460a-aba8-ce05a2d9026f authorizes M1 round3 from the clone with --no-actual and exact reason: “M1 reviewed from an independent clone while the registered checkout lands #640; clone lacks the primary checkout's queue evidence”. Proceeding without waiting for #640. Later boundaries and landing must use the registered checkout after #640 frees it. This is an explicit calibration exclusion, not invented actuals or restored evidence.


### 2026-10-17 — M1 SHIP; R4 approved for M2

M1 round3 returned SHIP with all four findings disposed and no new findings. Binary evidence commit a237be890c228e07b4c7445301a573cdbd8beb8c is published after fetching/inspecting the stale remote ref and a normal fast-forward push; local review pin refs/sdlc/reviewed/000647/M1 is retained in the independent clone. The approved no-actual reason is recorded by the gate. This completes M1 only; native qualification and the recorded whole-issue suite remain outstanding.

Read-only overlay proof /tmp/pair647-validator-stdout-proof.log found a further pre-existing validator-substring gap after the candidate was locked. Stopped and replanned the complete diagnostic class before another code delta. Outgoing receipt25538ce2-4dce-4d17-b2c5-7d11c296a71b and TL approvalc8f8d425-c96f-4305-8e85-b0ec8abe100a are submitted/authenticated. R4's complete status/body/target/error matrix is appended to the durable plan and approved folded into M2, preserving native name-budget search and valid client/absence behavior. Installed-native parse-only manifest is retained; no server connection or operator session was involved.

Proceeding with independent M2 areas in the preparation clone. The registered checkout stays on #640 with its queued merge active (session87105, process73317 at last status). Later boundaries and landing must run there after it is free; transfer the reviewed M1 pin along with the branch. No M2 review or whole-issue completion is claimed.

### 2026-10-17 — M2 diagnostic and action checkpoint

Recovery gap: the requested previous-screen peek returned "previous session descriptor is unavailable"; resumed from Git branch, durable plan, M1 review pin and authenticated TL receipts. No historical screen instruction was used as fresh authorization. Registered checkout remains untouched while #640 waits in the landing line; all M2 work is in the authorized independent clone.

R4 exhaustive diagnostic matrix: RED retained at /tmp/pair647-r4-diagnostic-matrix-red.log (32 authority failures); GREEN /tmp/pair647-r4-diagnostic-matrix-green.log (15.093 s), race /tmp/pair647-r4-diagnostic-race.log (33.470 s), 640 consumer/fault cells. Exact validator decoding requires requested name, pure exit 2, empty stdout, full native diagnostic grammar and canonical uint64 limit; the value is validated only, never converted into downstream arithmetic. Existing monotone budget search passes eight actual probes; zero limit remains accepted. New parse-only native conformance is gated and not run yet.

Layout preflight RED /tmp/pair647-layout-red.log reproduced native mutations after wrong-kind confirmation, changed target and wrong tab. Strict paired pane/tab admission now binds to the independently known invoking terminal, scopes decisions to its tab and addresses supported tab mutations explicitly. Restoring refuses before focus/record-clear on uncertainty, retaining completed native progress and the recovery record (ARCH-PURE/DRY). Whole layout package GREEN /tmp/pair647-layout-package.log (1.267 s); focused race /tmp/pair647-layout-race.log (2.311 s). Stateful external CLI tests cover healthy mutation, no preflight effects, and post-embed refusal without inverse mutation or recovery-record loss. Pure fixture geometry/order assertions remain; positive fixtures now supply actual native fields. Context-aware query methods are mandatory; read-only subprocesses use the shared collecting runner, mutators retain existing lifetime ownership.

Couch restart and note admission controls GREEN couchcmd 1.818 s / couchtty 0.365 s; race 2.766 s / 1.764 s. Restart preserves the existing one-second budget with final active-thread/endpoint fence; notes retain the two-second budget and recheck relevant anchored box text while accepting unrelated output. Actual wrapper presentation oracle passes 0.323 s, race 1.384 s: clipboard refusal adapter through private broker and production master pump to decoded stdout notification envelope, preserving child output/lifetime. Native restart latency remains unmeasured. Clipboard and terminal/draft routing remain in progress; no M2 completion or exact-head suite claim.

### 2026-10-17 — M2 layout latency decision and scoped projection

Authenticated TL approval 683cc32b-b71b-484d-be31-ec48e5318967 retains confirmation correctness, permits one return-to-Normal projection narrowing experiment, prohibits concurrent query reads, and accepts approximately 350 ms if narrowing does not help. Return preflight now omits irrelevant geometry from comparison; it retains identity, role/state and tab, while Normal entry and actual split-order restoration still require geometry. Stateful positive geometry-drift and negative target/tab-drift controls pass. No new retries, parallel reads or deadlines.

Peer pair:6 released the native window (authenticated 3af7fd91-5c36-4e0f-a3e5-4ac96bd2dac2). Three declared serial disposable installed-0.45.1 timing controls passed unchanged conformance assertions: initial hardened candidate 5.62 s (entry median 171.416 ms, return 353.181 ms), original binary control 3.38 s (entry 91.209 ms, return 192.424 ms), narrowed candidate 4.14 s (entry 130.469 ms, return 293.856 ms). Five entry/return samples each; host load not controlled, so no causal narrowing-speedup claim. Root /tmp/p647m2.EeIsbH contains raw logs, binaries, overlay and hashed manifest; no operator environment/build and no surviving native processes. This healthy timing sample observed no misroute and does not qualify M3 refusal. Window released back to pair:6. Atlas/README record the accepted interactive cost. The earlier hub estimate of original entry "about 95 ms" is corrected to the measured median 91.209 ms.

### 2026-10-17 — M2 terminal, clipboard and integration qualification

Terminal/draft actual-CLI RED logs /tmp/pair647-termcmd-admission-red.log and /tmp/pair647-draftroute-admission-red.log reproduced foreign-tab routing and unconfirmed effects. Selector-first RED /tmp/pair647-term-draft-early-exit-red.log reproduced four violations of the first-unknown stop rule. Final full package GREEN /tmp/pair647-term-draft-final-green.log (term 1.996 s, draft 1.487 s), focused race /tmp/pair647-term-draft-final-race.log (2.496/2.479 s), wrapper compatibility /tmp/pair647-wrap-route-compat.log (0.411 s). The same five-second context crosses terminal decision, nested layout and draft delivery; admitted draft is consumed without a third lookup. Existing registered-local/validated-cache paths remain zero-query. Slow fallback scopes caller/role/unique destination and validates required Alt+k target before sidecar writes. Missing unrelated draft remains healthy for inert/split decisions.

Clipboard full package GREEN 13.676 s and full race 23.566 s, including actual detached CLI/private-broker and sleep-free shell cases. Auto-quote confirms combined clients/panes, unanimous attached focus, same-tab unique draft and the full plan before effects; standalone paste requires its own invoking pane/tab. Explicit flash remains zero-query. Actual focus/trigger has one addressed target, no positional fallback/replay; existing exact same-target already-focused success is shared. The independent wrapper presentation oracle proves production broker-to-output delivery.

Planned cross-package M2 selection /tmp/pair647-m2-integration-targeted.log passes all seven packages (layout 8.829 s, draft 9.272 s, clipboard 35.585 s, term 0.303 s, couchcmd 11.349 s, couchtty 7.143 s, wrap 2.793 s). After shared selector validation changed, focused shared race /tmp/pair647-m2-final-shared-race.log passes layout 2.960 s / term 1.248 s / draft 2.072 s. Whole layout+stateful-fake+artifact inventory /tmp/pair647-m2-layout-final.log passes 4.888/3.845/12.650 s. These are targeted checks, not an exact-head full-suite record.

Native R4 qualification in the coordinated private window passed both parse-only validator conformance (0.02 s) and existing complete acceptor-versus-real-native budget check (1.75 s), retained /tmp/p647m2.EeIsbH/native-name-budget.log. Clipboard native attempt failed fixture setup before any server: invalid KDL syntax, then blocking FIFO cleanup. Retained /tmp/p647m2.EeIsbH/native-clipboard.log includes SIGQUIT stacks from the exact owned test process; this is an owned fixture defect, not a flake/retry classification. Fix KDL and add deterministic FIFO teardown controls before a changed-fixture native attempt. No production failure is inferred from that attempt.

### 2026-10-17 — Native clipboard setup diagnosis and editor phase approval

Second changed-fixture clipboard attempt retained /tmp/p647m2.EeIsbH/native-clipboard-fixed.log: exited in 0.33 s with receiver-pipe closed before native action admission. This was not a product failure or passing retry. Darwin Go intentionally excludes FIFO descriptors from runtime polling, so O_NONBLOCK makes read return EAGAIN; a descriptor flag/teardown-only control cannot establish byte transport. Stop further FIFO deltas; fixture-only re-plan (private stream listener, two owned receiver connections and actual-byte plus teardown controls) sent to TL, pending approval. Native window remains reserved with peer:6, no native run active.

Authenticated R5 approval e1ea8c04-2d35-4b70-87c2-5aeb66b4ff25 records DOM/review admission phase boundaries in the durable plan; no M3 code before M2 closes. M2 production and deterministic controls checkpointed; remaining native clipboard fixture qualification and registered-checkout review still outstanding.

Native fixture re-plan approved in authenticated TL receipt f82d7d85-64bf-4632-9d5e-27f99baf9d37; implementation delegated within the isolated clone. Earlier failed attempts remain retained. One private listener holds ownership for both receiver connections and deterministic teardown; real-byte delivery is a mandatory positive control before another native attempt.

### 2026-10-17 — Approved native receiver ownership qualified

The approved single Unix-stream listener fixture passes real readiness/data delivery, preserved EOF errors, and no-client/idle-reader/unread-buffer teardown: normal 0.198 s, race 1.230 s. One changed-fixture installed-native attempt passes /tmp/p647m2.EeIsbH/native-clipboard-stream.log (0.68 s): auto-quote four sequential native queries 53.279 ms, full helper 105.783 ms; standalone two queries 25.701 ms, helper 55.947 ms. Actual bytes reach only the admitted draft, Ctrl31 is observed, private staging content and final client focus match. Clipboard text is a private literal; cosmetic reset scheduling is recorded to avoid unowned delayed work. This does not claim an OS clipboard copy or a native fault observation. Both earlier setup failures remain retained alongside this changed-fixture result. Source, binaries, logs and hashed manifest are under /tmp/p647m2.EeIsbH. Native window released to peer:6, no survivors/operator environment/build. Original-implementation paste baseline remains to reconcile before claiming complete before/after latency.

Final inspection found a presentation wiring gap: terminal toggle discarded the shared executor's refusal exit and its OS runtime lacked the notifier method; wrapper draft routing defaulted to stderr. Completing the already approved R2 contract with a mandatory layout notifier and terminal/wrapper production-path refusal controls before M2 review; no new notification transport or deadline.

### 2026-10-17 — M2 presentation wiring and matched clipboard baseline complete

Terminal silent-refusal RED /tmp/pair647-terminal-refusal-red.log and wrapper missing-notice/delivery-diagnostic RED /tmp/pair647-wrap-draft-refusal-red.log are fixed. Layout Runtime notifier is mandatory; term OS runtime delegates the existing broker and both terminal toggle entry paths return refusal to the mux. The actual terminal CLI with stateful Zellij and private broker proves nonzero refusal, fixed notice, zero effects and one query. Wrapper default draft-route failures send a fixed notice and retain failed delivery, preserving injected reporters. GREEN /tmp/pair647-term-wrap-refusal-green.log (term0.929/wrap0.464 s), full term /tmp/pair647-terminal-refusal-package-green.log (1.809 s), focused race /tmp/pair647-term-wrap-refusal-race.log (1.368/1.458 s). Layout discarded-stderr delivery-loss RED /tmp/pair647-layout-delivery-red.log is fixed by retaining broker delivery errors in the existing fullscreen store; whole layout GREEN /tmp/pair647-layout-delivery-green.log (1.958 s), focused final race /tmp/pair647-layout-presentation-final-race.log (3.855 s).

Final integrated targeted selection /tmp/pair647-m2-final-integration.log passes all eight packages: layout1.466 s, draft1.135 s, clipboard11.163 s, term0.464 s, couchcmd4.362 s, couchtty5.436 s, wrap4.221 s, artifact inventory2.139 s. No exact-head full-suite claim.

Original-clip/native timing baseline uses all nine Git-verified clip files/tests from M1, a shared receiver fixture and explicit removal of only the owned default tab; replies remain unfiltered. Both declared serial controls pass actual source/draft bytes, staging and client-focus assertions: original0.76 s, candidate0.87 s. Original auto-quote two observations25.998 ms, helper102.664 ms; candidate four observations52.882 ms, full helper117.004 ms. Original standalone one observation12.563 ms, helper83.734 ms; candidate two observations25.174 ms, helper63.783 ms. The original runtime invokes original leaves in process and excludes old subprocess/exec overhead; private clipboard/cosmetic-reset boundaries match. One sample each, not a causal speedup claim or full original-process latency comparison. Overlay source hashes, binaries, raw logs and limitation manifest retained /tmp/p647-clip-compare.9iWROT. Native window authenticated release a2525563-6b2f-42ec-ba8b-ab24659b1f6d and released again after controls. No native survivor/operator environment/build.

M2 implementation and targeted qualification are ready for its review. Registered checkout still holds queued #640; no M2 gate run from the clone and no M3 code begun. R5 and native fixture re-plan approvals are durable above.

### 2026-10-17 14:00 PDT — Review-location wait checkpoint

M2 source checkpoint is 787ac7d47127f27960c4976449b51ca9c6c1544b, pushed normally to the owned issue branch; source tree is clean. Review-location/outcome request 20e6b56b-d3e2-4a06-8b77-38f48727a5b2 remains queued at Couch position 3, "waiting for the composer to be in view", as of 14:00 PDT. It has not been submitted or answered; elapsed deadline is not approval. Check its receipt before any resend, and resend only if expired/not-dispatched. No M2 gate has run, no --no-actual exception is assumed, and M3 remains read-only. All delegated tests/edits and native controls are finished; no native process remains.

The registered checkout is still clean on the #640 branch, with its original merge process queued (session 87105, known PID73317); do not switch or edit it while active. Landing line last observed #646 active, then #620, then #640. Preserve that merge. Once #640 frees the checkout, verify and report its landed PR410/hash with a submitted receipt; import this #647 branch plus refs/sdlc/reviewed/000647/M1 from the independent clone and run M2 there unless the TL explicitly approves the requested clone exception. R5 is approved/durable, and M3 starts only after M2 SHIP. Whole-issue suite, M3 qualification, close and landing remain outstanding.

### 2026-10-17 — M2 review-location approval received

Authenticated TL receipt 829d7622-04fb-474b-876c-301c5ae26051 answers the queued request: review and milestone-close M2 in the clone using --no-actual, reason "M2 reviewed from an independent clone while the registered checkout lands #640; clone lacks the primary checkout queue evidence". #646 and #620 have landed. Do not hold M2 for #640; start M3 only after M2 SHIP. No exception is inferred for later whole-issue close/landing.

### 2026-10-17 — M2 REWORK recovered; registered checkout restored

M2 review of bf0524122b46540e2db6d2e0f80d6503e4fbf1d1 returned REWORK: BR-5 displayed-tab/frame provenance, BR-6 cancellation after positive sidecar reads, BR-7 two unmanaged Couch query descendants, BR-8 owed flash reset skipped on cancellation. Sidecar and ledger retained in 75b3f75ccf33b94f0a5397ec4eb16ab82e79399a. No findings disposed yet. Three broader Couch listing-fixture failures also need classification: their pinned review base includes M1, so inherited does not establish a main defect.

#640 landed PR410 at bcf5163bdfac06a3fc51021ddb635ce52fa755a8, verified by Git and hub receipt e803df50-f010-4772-8934-1fca435a33b8. Imported the exact #647 branch and M1 reviewed pin into the clean registered checkout, without modifying the resting branch or running another integration. Previous TTY descriptor unavailable; committed source/review evidence and authenticated receipts supply recovery. All three delegated audits were read-only. No review-fix source or native run started.

## Revisions

### 2026-10-17 — R6 complete M2 mechanism re-plan proposed

Reason: BR-6 is the third finding in unknown-observation-is-terminal; hub explicitly requires a complete plan before code. Appended R6 to the durable plan, covering BR-5/6/7/8 together. One shared pre-effect context gate covers native actions, authoritative sidecar writes and Couch query-derived dispatch, enforced by one exhaustive sink manifest and deterministic per-boundary cancellation tests. Already-issued flash owes reset independent of admission cancellation; apply/reset errors become visible, with no retry or changed mutation lifetime.

Native note frames carry no client/tab provenance. Proposed deliberate fail-closed behavior removes automatic native pane/file enrichment and its now-orphaned async owner, retaining exact screen note text/box/slot and Couch-owned status/tab-bar anchors with visible explanation. This requires hub approval; adding current client queries alone would falsely claim correlation. Removing the note query eliminates one BR-7 bypass; remaining restart client queries adopt the existing collector and real descendant lifetime tests. Native lane is granted by authenticated receipt 0a7eb7b6-1705-49f9-ac54-f2d997a44daf, but no run is needed before plan approval. M2 remains open; M3 stays behind SHIP.

### 2026-10-17 — R6 approved scope implemented; M2 review held for BR-5

Authenticated approval a019943c-f6bf-454a-a5f2-87982ee5aedf covers BR-6/7/8 and classification; BR-5 is explicitly pending operator per 6628e159-5c2b-4ed0-961c-873fafa77e6c. Existing note enrichment is retained. Both existing Couch query entrypoints now collect their process groups. Shared Effect gates cover actual native actions, sidecar writes, clipboard staging and Couch dispatch/publication; the 50-boundary manifest enforces the current owner catalog. Owed flash reset scheduling survives cancellation/apply failure, with visible apply/reset-start errors and no later paste. No new timer, retry, rollback or mutation lifetime.

RED evidence: /tmp/pair647-r6-root-red.log (sidecar focus/tab effects after cancel; note publication after cancel; restart token ended before dispatch); /tmp/pair647-effect-helper-red.log; /tmp/pair647-effect-coverage-guards-red.log; /tmp/pair647-br7-8wqsiycb/red.log (all eight actual query entrypoint descendant leaks); /tmp/p647-br8-red.log and -failures-red.log. Checker adversarial controls also failed before hardening: /tmp/pair647-effect-checker-red.log and -bound-reference-red.log.

Qualifying normal/race: full layout3.631s and Couch TTY50.660s (/tmp/pair647-r6-layout-tty-packages.log); root cancellation/prefix race1.468/1.478s (/tmp/pair647-r6-root-race.log); reply/terminal/draft full0.175/3.385/1.820s and focused race1.233/2.669/2.782s (/tmp/pair647-effect-final-green.log, -race.log); final checker-only reply race1.226s (/tmp/pair647-effect-coverage-final-race.log). Clipboard full10.207s/race14.239s (/tmp/p647-br8-package-green.log, -race.log); Couch query matrix3.366s/race8.758s, prior restart/note controls3.156s (/tmp/pair647-br7-8wqsiycb). Final full couchcmd PASS63.479s in /tmp/pair647-r6-couchcmd-final.jsonl.

The three broader Couch failures were ours: TestMessageAuthorityListedCheckReadsThePolledListing, TestMessageThreadResolverChecksReadOnlyThePoller and TestFleetQueryCollectorScopesConcurrentParentOperations used abbreviated full listings/client rows incompatible with M1 strict contracts. Identical main fixtures passed permissive old consumers, so not classified as pre-existing product flakes. Only fixture forms changed; original assertions retained. RED2.649s, GREEN2.847s/race3.967s: /tmp/pair647-r6-listing-fixtures-{red,green,race}.log. The earlier whole Couch run also compiled before BR-7 and reproduced its expected eight leaks; retained /tmp/pair647-r6-couchcmd-package.log is not qualifying evidence.

Native attempts retained separately: /tmp/pair647-r6-native-layout.log rejected the overlong runtime/socket root before birth. The corrected private-root attempt /tmp/pair647-r6-native-layout-private.log refused malformed list-panes bytes; raw body was not retained, so exact cause remains unknown. Added production/broker malformed empty/truncated/warning/plain-text tests and native fixture bounded separate-stream capture (RED /tmp/pair647-native-capture-red.log, GREEN0.587s /tmp/pair647-native-capture-green.log). Hub pre-granted one diagnostic attempt, which passed10.493s with all geometry/focus/record assertions; 32 pane +50 tab replies valid, stderr empty. Raw log /tmp/pair647-r6-native-layout-diagnostic.log does not explain the prior unobserved malformed body. Clipboard healthy control PASS1.729s /tmp/pair647-r6-native-clipboard.log; four/two observations, actual bytes/staging/focus. All real runs use installed0.45.1, private environment, no operator sessions/clipboard/:0/build. Lane released, socket tree empty.

Hub cb1272f0-c43d-46d4-9e87-e488a48eabb5 directs checkpoint now and holds M2 review for final BR-5 choice (screen-only versus enrichment labeled as looked up after capture). No M2 milestone-close or M3 work until that answer. All test/native/agent jobs finished; no full #647 suite or whole-issue close claimed.
