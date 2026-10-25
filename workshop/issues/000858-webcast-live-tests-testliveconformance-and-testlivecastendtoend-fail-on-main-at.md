---
id: 000858
status: codecomplete
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-25
estimate_hours:
card_mirror: '7ee42352596b18a1d0ab4d3bd2c8e2eca63da488' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-25T00:11:45-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "56328961", done: "dc4bbbee"}
actual_hours: 0.57
---

# Webcast live tests TestLiveConformance and TestLiveCastEndToEnd fail on main at the in-page pause click

## Problem

## Spec

The native fixture clicks the actual in-page Live/Pause button using real CDP mouse input. It must resolve the control inside Couch's closed shadow root rather than assuming the outer UI host's center is the button. Production controls, authority and pause semantics stay unchanged. Headless Chrome uses a private profile and never raises a foreground window. Retain existing readiness and state deadlines.


**Two webcast live tests fail on clean main: TestLiveConformance and TestLiveCastEndToEnd (found by pair:6 while
working pair#849, 10-23).** Both fail at the same step, clicking the in-page pause control, identically on clean
`origin/main`, so they're not caused by any branch. Find whether the pause control moved or changed (pair#850's panel
and controls, or another recent cast change), or the tests' selector is stale; fix the cause, not the test's
expectation, unless the control legitimately moved.

## Done when

- Root cause is recorded against clean main; the reported native pause failure is reproduced before the fix.
- The fixture locates the actual Live/Pause button and checks its usable geometry before real mouse input; a missing/hidden control refuses rather than clicking a guessed host point.
- TestLiveConformance passes pause/resume and TestLiveCastEndToEnd observes paused viewer state. Audit and qualify every shared clickControl consumer, one targeted native run at a time, with no focus-taking Chrome.
- Required recorded pre-close and mandatory close pass; no production behavior or timeout changes.


- Both tests pass on main, with the cause stated in the Log.

## Plan

- [x] Reproduce on main and trace the host/button geometry ownership.
- [x] Replace the stale fixture hit target with the actual closed-shadow control; keep real input and downstream state assertions.
- [ ] Run the named native regressions and affected fixture consumers; inventory, recorded pre-close, close and merge.


- [ ]

## Log

### 2026-10-25 — Cause-first diagnosis
- 2026-10-25: closed — BR-1 fixed: actual closed-shadow control visibility/opacity, viewport and shadow/document hit checks precede all real mouse input; debugger objects released. Real native TestLiveControlReadiness failed hidden/transparent/offscreen/clipped before predicate and passes after, with zero mouse events for every refusal including missing control and real visible/restored transitions. Canonical issue.side/evidence/native pins complete source-bound runs on7b104dd55c2078641b813b72b628e8e769314ec3: conformance, end-to-end and readiness exit0, private-profile Headless:true,count1,sequential; controlled old-fixture reversion exit1 at actual page pause, no timeout or production changes. BR-2 formal dispute names exact log hashes and source metadata; reviewer GPU failure is before page creation. Inventory PASS without refresh; no new atlas surface because fixture-only. Fresh recorded pre-close retained; its pass still precedes landing. Last unchecked Plan row is recorded-check/review/merge workflow, not unimplemented code. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

Started from clean main63d35ac8295d6b4ff04001c683b038eca1d0e3d7. WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run '^TestLiveConformance$' -count=1 -v -timeout=90s fails17.15s at the page pause assertion: Phase remains live. Launch uses Headless:true and a private t.TempDir profile. The shared helper queries only outer data-couchcast-ui geometry; #856's mountCastTools adopts liveButton into a fixed-position column, invalidating that hit-target assumption. The closed shadow's actual cast-button remains the production click/binding owner. Fix that fixture ownership boundary (ARCH-DRY, ARCH-PURPOSE, ARCH-MOCK), not the pause behavior. No new state, wire, timer or persistent resource. Runtime cost is fixture-only bounded debugger queries on an explicit action.

#857 remains on its own pushed branch, codecomplete after documentation-only FIX-THEN-SHIP correction; native revised-source suite passed, required integrated check terminal2123/lease-904989313 is retained on its independent snapshot. It has no merge runner and may safely finish while this canonical checkout handles #858. Pair:0 stays unchanged; #857's operator re-look is tomorrow.


### 2026-10-23

### 2026-10-25 — Native pause fixture corrected

Updated only liveHarness.clickControl in live_test.go. It asks CDP for the host's closed shadow root, queries the actual cast-button and reads that node's box model; missing roots/nodes or non-positive boxes remain unavailable until the existing10s readiness deadline. It dispatches real mouse press/release at the actual button center, preserving production hit testing and toggle binding rather than invoking a JavaScript handler. No production source or timeout changes.

Before fix TestLiveConformance failed17.15s at pause with PhaseLive; after fix it passes1.56s/package1.815s including pause and resume. A separate, non-overlapping TestLiveCastEndToEnd run passes1.36s/package1.532s and sees the paused empty viewer state. Both commands set WEBCAST_LIVE_CHROME=1, use -count=1 and -timeout=90s; existing liveChrome launches Headless:true with private t.TempDir and joined cleanup. Shared-consumer sweep finds exactly these two native callers (conformance invokes pause and resume) and the fakeHarness equivalent. No stale host-center click remains in other fixture callers.

No new global lint rule: the outer-host-center operation is valid when a host owns a single button and is not a generally forbidden API; the regression is the existing real column/native control path. Quick plan sent to hub (receipt7656d516 submitted), proceeding unless objection. Next: fake conformance, required inventory, commit/push, recorded pre-close and mandatory small-diff review. Pair:0 untouched and857 check retained independently.

### 2026-10-25 — Source-pinned native qualification retained

On committed code7b104dd55c2078641b813b72b628e8e769314ec3, all three targeted native commands pass separately, count1, zero skips: TestLiveConformance (log SHA256e8d6b9bd703aff3364be21c5827cc81984ebc9a637290eedcff517362b54533b), TestLiveCastEndToEnd (dfd26b5fb0ed56d16dcaaa919cc89b89234b790af78991eeda8f029b5f08ff32), and TestLiveControlReadiness (f954e304a4a44378c05f1ef619e97f533b08e01141310857137a2fadf68ba66e). Complete logs/metadata are in this issue's canonical .side/evidence/native directory, pinning argv, environment, head, exit and source cleanliness. A controlled overlay reverts only live_test.go to clean-main63d35ac8 and omits the new helper-dependent readiness test: TestLiveConformance again fails at the actual pause assertion (exit1, wall20.370s), log SHA2564b370ec1521573f4b5be18112000157c79fca7fe305c86957b4360095d716fa2. Metadata pins the exact reverted source bytes; no source or timeout was changed for the passing measurements. The temporary overlay directory was removed after use.

The stale858 check was still queued, not running, and was cancelled through sdlc local-check cancel --run lease-1361220752; its retained terminal80615 exited1 with context canceled, not a test failure. Queue the fixed source once after pushing the evidence/dispute. The independent857 suite finished PASS12m57s on70f5246cf (log prefixe80a0c145c3b), reported to hub; it remains waiting for tomorrow's stage look. Pair:0 untouched. BR-2 receives the measured host evidence via formal dispute; BR-1 receives the actual readiness fix and native refusal regression, not a waiver.

### 2026-10-25 — Post-close catalog failure fixed at the original wait owner

Round2 close finalized SHIP, BR-1 addressed and BR-2 withdrawn, evidence d0a83511ccdf1f34868433123fb9eff979e6370c; measured0.57h, no measurement waiver. The fixed-head affected attemptc66517e9f36331041b04c0835f978f67 then completed exit2 in56s, solely internal/repolint.TestTrackedGoFilesPassCatalog: unused sleep debt for liveHarness.clickControl's100ms sleep. Failure log .git/sdlc/test-stats/failed/c66517e9f36331041b04c0835f978f67.log, SHA256 prefix32974cb8f530. This is deterministic branch debt drift, not a flake; no retry or registration.

The catalog deliberately admits only shrinking debt, so renaming its entry would be wrong. Retained the existing10s/100ms readiness wait in liveHarness.clickControl, its original exact admitted owner. The shared input helper now makes one attempt and distinguishes readiness misses from input errors so uncertain input is never retried. Native refusal coverage uses the one-attempt seam; added an injected input-error assertion requiring exactly one attempt. No catalog/gate/debt-file modification, production change or deadline change. Focused catalog PASS5.16s/package5.297s with all100 admitted sites, and native readiness PASS1.43s/package1.667s. Next: source-pinned native evidence, fresh pushed affected check and narrow re-close of this post-SHIP fix. Standing measurement waivers are retired by TL09ee0c34; omit actual flags and let close measure normally.

## Revisions

### 2026-10-25 — BR-1 readiness boundary and native refusal qualification

Make the existing hidden-control refusal explicit before input: resolve the actual button, inspect computed visibility/opacity through its ancestors, require the whole box in the viewport, and verify both shadow-root and document hit targets. Readiness is one shared gate for pause/resume and end-to-end pause; the existing10s deadline stays unchanged. Debugger objects are released before dispatch. No production state or new capability is added. Test the gate through real Chrome for visible/restored, hidden, transparent, offscreen, clipped and missing controls; a refused attempt must send zero mouse events.

### 2026-10-25 — Review findings and red/green correction

Close round1 REWORK: BR-1 correctly notes positive dimensions do not establish usability; BR-2 reviewer Chrome failed before opening a page (GPU/sandbox startup). The measured actual succeeded normally at0.07h; no waiver applied. Added TestLiveControlReadiness using the real closed-shadow DOM and a forwarding CDP call counter, not fake geometry. Before the readiness predicate it fails all4 unusable cases in1.04s. After the predicate it passes all cases in1.09s/package1.421s, with zero input on refusal and real state transitions when visible/restored; missing node also refuses without input. The shared fixture now returns errors instead of making the unit caller fatal, allowing exact pre-dispatch assertions with a zero-wait test deadline. Production callers retain the original10s readiness deadline.

Next: commit the reviewed-class fix, cancel only the stale queued858 affected lease-1361220752 through sdlc (never857), retain source-pinned native pass and controlled old-fixture RED logs, push and queue a fresh pre-close. Full re-close answers BR-1 and supplies BR-2 counter-measurement. No pair:0 action.
