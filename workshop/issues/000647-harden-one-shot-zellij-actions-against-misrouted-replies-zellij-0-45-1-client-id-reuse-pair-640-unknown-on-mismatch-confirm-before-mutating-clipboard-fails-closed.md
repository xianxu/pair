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
