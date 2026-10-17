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

- [ ] M1 — Strict reply contracts, stateful fake and lifecycle/absence admission.
- [ ] M2 — Go one-shot action and clipboard safety, with per-caller fake refusal tests.
- [ ] M3 — Editor admission, native refusal qualification and complete atlas caller map.

## Log

### 2026-10-17

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
