---
id: 000623
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
card_mirror: 'c68a8dc2da8e4ce3ce73f9b164a4d275bd80431e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T18:13:53-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "1474a1e3", done: "640e8a7b"}
---

# Review observation positive control intermittently loses external edit reload

## Problem

tests/review-observation-diagnostics-test.py intermittently fails its positive control, ObservationFixtureDiagnosticsTest.test_positive_control_retains_wanted_fresh_and_request_order. The nested real editor/Git fixture asserts: matching branch external edit did not reload; snapshot lost no-final-newline state.

Observed during #604's recorded rerun on fe4eb6d6e83bd53b9d42fcf27811a383c348b222, first seen here 2026-10-16 at about 16:01 PDT. Eight diagnostic cases ran in 63.905s; one failed. The full local-check stopped before Go tests, exit 2 in 6m59s after 1239s in the machine queue. Retained log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/ffdc28f65f4aad6e6fdf5f09c7feeb01.log, SHA256 prefix 7c703e6585f2.

The preceding complete make -k suite passed this diagnostic command and all other shell/editor/Go checks, with no production/test changes between attempts. Passing log: /tmp/pair604-keepgoing-suite.log, SHA256 79a1e5cb433c82e9af9207831d7deb750f9b8280a826238059423d03fc5b8624. One pass and one failure do not establish a calibrated rate.

## Spec

Use the retained observation ordering and fresh/wanted identity diagnostics to determine why the matching-branch external edit was not reloaded with its exact no-final-newline bytes. Distinguish a delayed observation from lost or rejected state; do not increase timeouts without diagnosing the ordering.

The hub reported load average about 140 and links likely load pressure to #618's Couch Zellij polling. That is a working hypothesis, not a measured cause of this specific assertion. The #604 clipboard diff is not entered by the review fixture, and no clipboard production/test code is to be changed for this failure. Related independent first failure: #620 (missing branch-restore landed.json acknowledgment); that case passed in this failing run.

## Done when

- A reproducer and diagnostics establish the missed reload/byte-preservation ordering.
- A regression defends the identified ordering while preserving exact no-final-newline behavior.
- Repeated runs report before/after failure counts and host conditions; prior failed attempts remain visible.

## Plan

- [x] Check ownership, claim/start-plan, read the retained trace and measure the failing deadline ordering before choosing a fix.
- [x] Replace artificial child-process sleeps/held pipes in the observation fixture with one fixture-owned callback-completion gate around real resolver results; keep native process options and ownership intact.
- [x] Synchronize the withheld-result, slow refresh, different-branch refusal and coalescing phases on explicit start/result/release/application evidence rather than 300ms/1.5s settlement sleeps. Keep exact-byte, undo/redo, branch-fence, nonblocking and coalescing assertions.
- [x] Add deterministic gate lifecycle/late-completion/timeout negative controls; run the intended held-result scenario beyond the native process budget without changing that budget, then report controlled and ordinary repetition counts separately.
- [x] Run targeted observer/trace/editor diagnostics and recorded full suite; enter close review and land on SHIP.

ARCH-DRY: one test-only gate owns callback retention for all delayed phases; reuse existing trace and real resolver boundary. ARCH-PURE: no production policy change, with gate state transitions tested directly and real editor/Git integration retained. ARCH-PURPOSE: prevent artificial latency from exhausting the resolver budget while preserving the byte and identity contract. ARCH-MOCK: the gate controls delivery of real native observations, not invented Git answers. ARCH-CONSTRAINTS: native 2500ms deadline remains unchanged; tests wait on bounded conditions and fail explicitly on actual process timeout. ARCH-ORDER: arm → started → completed/held → released or canceled; settle one phase before the next; a held observation may be released across checkout but never rereads it. ARCH-FUNERAL: gate state belongs to one isolated editor; retained callbacks are released/canceled on teardown and all files stay under its temporary fixture root.

## Log

- 2026-10-16: Filed unclaimed per the hub's explicit decision, delivery efadcd45-6fb6-487e-9392-7fc79782168e. The hub authorized one additional recorded #604 rerun; another failure must stop and be reported, with no indefinite retry loop. No fix is included in #604.

### 2026-10-17 — measurement before design

Confirmed the card was unowned, then claimed and entered planning from current main (092d32a9db3d082d7c413f740f7c2a4bacc14e31). start-plan refreshed the Ariadne tool layer and was rerun as instructed; no implementation edits made. A read-only subagent independently traced the resolver/observer ordering.

The retained #604 failure log proves the positive observation timed out, not a successful snapshot losing bytes: callback call 16 started with timeout 2500ms, completed after 2594.394291ms with code 124, signal 15 and stdout_bytes 0. Call 17 later resolved review/a in 830.720709ms. The test's wrapper inserts a 1s sleep before executing the real resolver, leaving about 1.5s for scheduler/startup/Git. The observer drops this invalid result and has no automatic retry; its 4s text-polling window cannot restore that one-shot observation. Existing recovery_observer guards appropriately require resolved identity-bound bytes. A separate fixed300ms settlement after the withheld observation and 1.5s waits in later phases also guess at completion.

At load averages 65.56/39.29/28.92 at measurement start (49.28/47.75/36.32 later), the unmodified positive case passed twice (22.749s and 26.45s); a third attempt failed before the reload phase because review.open was absent after the fixture's 6s startup wait. That startup miss is not counted as the reload failure and is not proposed for this change. Current untimed host load is evidence, not a calibrated flake rate.

A private init changed only the first existing artificial slow-wrapper sleep to 2.65s. Three valid controlled runs reproduced BOTH original assertions, exit 1, with code 124/signal 15/zero stdout at 2502.298ms, 2501.761ms and 2502.388ms (24.11s, 23.13s and 21.47s total). Logs and temporary init: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair623-measure.sishQA/controlled-fixed-{1,2,3}.log. Baseline logs are baseline-{1,2}.log; historical metadata extraction is /tmp/pair623-historical-timing.json. Three earlier probe attempts had a Lua pattern-escaping error, did not inject the delay or install the trace, and are excluded rather than counted as successful controls.

## Revisions

### 2026-10-17 — remove fixture-induced deadline exhaustion, keep production policy

Measured failure supports a test-only correction. Recommended: hold an already-completed real resolver callback in a fixture-owned gate, so tests choose completion order without spending the real child's deadline on artificial sleeps. Native invalid/timeout results remain failures and cannot authorize refresh. Alternative: enlarging the production timeout only moves the load threshold; automatic production retry adds a policy/mechanism not requested by this flake. Neither is selected. Before/after reports will distinguish the old in-process-delay reproducer from the new callback-held scenario; a genuinely slow native resolver remains subject to the same 2500ms limit. Plan sent to the hub before code; waiting for design approval because it introduces a test completion-control seam.

### 2026-10-17 — approved test-only implementation and deterministic boundaries

The hub approved 4036a148e with the condition that the gate add no production hook (delivery d29eee45-661e-4095-a9f3-db6e72b6a812). change-code selected quick flow. The gate and its lifecycle tests live only under tests/; production resolver, observer, timeout and admission code are unchanged. The first new diagnostic failed because the old fixture had no phase_evidence; the gate unit failed on its absent helper and later on missing timestamps before those implementations were added. Logs: /tmp/pair623-phase-red.log, /tmp/pair623-gate-red.log, /tmp/pair623-gate-timing-red.log.

Early integration repetitions exposed an implementation race in the new fixture: an observation could start between successful arm and a second RPC reading the current ID. The snapshot then excluded that held request and waited for another request that could not start. Capturing the pre-arm ID and arming inside one editor RPC removes that gap. A forced ordering test sends a real FocusGained request during arm and requires observed in-flight/result evidence before returning. Replacing the atomic boundary with the original two-RPC code fails with controlled resolver did not start (/tmp/pair623-arm-negative-forced.log); the earlier negative attempt did not actually force an observation and is excluded. Gate state is bounded, metadata omits document content, actual SystemObj/options are retained, and close discards callbacks without delivering late results. Pure observer, gate and trace tests pass. The first complete diagnostics pass was 9/9, then 10/10 with the arm regression; final repetitions follow below.

### 2026-10-17 — targeted acceptance and repeated counts

Final diagnostics pass all10 cases in30.741s, including the forced arm boundary and lifecycle/native-timeout tests (/tmp/pair623-diagnostics-forced.log). Pure observer, gate and trace tests pass independently. Five ordinary real-editor positive repetitions pass (0/5 failures;14.43s,13.24s,14.88s,11.48s,9.53s). Three final controlled repetitions both force observation start inside arm and retain the completed positive callback for2700ms; all pass (0/3 failures;17.88s,14.31s,12.34s). Measured completion-to-release holds are2853.08ms,2873.32ms and2860.03ms, while every recorded native timeout remains2500ms and start/result/release/settled/applied events retain one ID. Logs/reports: pair623-measure.sishQA/final-normal-{1..5}.{log,json} and forced-after-{1..3}.{log,json} under the temporary directory recorded above.

Before: the old fixture's first artificial child delay of2650ms failed3/3 with the original paired assertions. After: a2700ms callback-delivery hold succeeds3/3 because the real native process has already completed; genuinely slow native processes still time out. These are deliberately different injection boundaries reflecting the fix, not a claim that native work can exceed its budget. Host load declined during investigation: load averages12.04/32.91/41.38 at final collection, versus65.56/39.29/28.92 at start. Counts establish controlled ordering coverage, not a calibrated natural flake-rate comparison. No production file or hook was added.

### 2026-10-17 — recorded full-suite acceptance

Recorded suite passed on2514a9ec037e240bf2f0270f0862484bc4cf4f2e in12m26s, attempt8fd112334610bd02d84b0727a8079f25, log sha256c05a8dba998a. All10 observation diagnostics passed in28.667s, the standalone observation fixture passed, all other shell/editor checks and every Go package passed. Log: /Users/xianxu/workspace/pair/.git/sdlc/local-checks/2514a9ec037e240bf2f0270f0862484bc4cf4f2e/suite.log. PR#418. No production hook, deadline change or live staging; acceptance is the controlled private headless fixture and normal suite. Entering close on this tested branch without manually chasing main.
