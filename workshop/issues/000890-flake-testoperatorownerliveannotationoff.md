---
id: 000890
status: codecomplete
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: '4283676f99ee459d99ba9dece13ef3a9d5d667da' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-25T10:31:42-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/webcast.TestOperatorOwnerLiveAnnotationOff]
flow: {kind: quick, provenance: inferred, spec: "b21f6d6f", done: "6ba36e92"}
actual_hours: 0.46
---

# flake: TestOperatorOwnerLiveAnnotationOff

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/webcast.TestOperatorOwnerLiveAnnotationOff: pair#885 M1 live-Chrome proof, 2026-10-25, pair:10 at e769340d4 (WEBCAST_LIVE_CHROME=1, load average 23-35): 2 failures in 20 runs, both operator_owner_live_test.go:84 "waiting for Note selected after expanded Pen: context deadline exceeded" after 60s (the test ctx), i.e. the click on .sketch-label button never left Note aria-pressed=true. Unmodified main (4ae4b15e1) passed 10/10 in the same window. The #885 change to this test only moved its wait from a 10s hand-rolled deadline to testwait.WaitEvery at the same 25ms cadence; the condition, clicks and ctx are unchanged, so the event was lost rather than slow. Cause unclassified (candidate: the label submit click races the toolbar re-layout after resize(400)). Not tuned here.

## Spec

The owner live fixture must observe completion of a requested viewport resize before sampling native click geometry. Emulation metrics change innerHeight before Chrome dispatches resize; the old toolbar can already fit, so its bounds do not prove floating Notes consumed resize. Arm a viewport-matching resize receipt before metrics, await it after existing synchronous UI listeners, and dispose its listener/object on success or error. Same geometry is an immediate no-op. Retain native one-shot press/release, real permissions and existing timeout. ARCH-ORDER separates changed viewport from consumed resize; ARCH-DRY owns the CDP resize sequence in one test helper. No product/API/persisted-state change.

## Done when

- A controlled private-Chrome regression holds resize delivery after metrics change and rejects a helper that reports completion early; releasing the event completes the barrier. Same-geometry and error cleanup are covered.
- The owner scenario passes bounded native runs with the existing timeout, one click per action; no retry or arbitrary sleep.
- The recorded affected check and normal landing suite pass; close unregisters890.

## Plan

- [x] Pin consumed-resize ordering with a held native event and establish RED.
- [x] Await the armed resize receipt in the owner fixture; verify native and lifecycle controls.
- [x] Record evidence, lifetime lesson, inventory and final verification.

## Log

### 2026-10-25
- 2026-10-25: closed — BR-1 supplied: fenced Chrome fails before assertions, so committed host proof at exact requested code head4b0b9f8e9d4d9b0c9d5c137bff879c5432319669 with clean source diff. Both focused native tests count3 PASS18.909s; metrics-only overlay on same head fails held-event assertion1.347s exit1; complete logs, commands and SHA256 in workshop/evidence/000890/host-proof.md. Earlier final owner+held-event controls count10 PASS61.602s, throttled native trace count10 PASS46.813s, race PASS7.223s, catalog PASS8.158s. Reader inventory PASS immediately before re-close. Native one-shot input, existing timeout and product behavior retained; atlas/lessons complete. Affected run87687 is still retained queued on code-identical4b0b9f8e (six Go packages plus shell), permitted review/check overlap; must pass before normal merge and integrated suite. No extra check or manual main integration. Actual omitted. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

Claimed from main after884 landed. Dispatch4f39b992 asks890, then895, then888 one at a time, private headless Chrome only, targeted count<=10, no retry/timeout increase. Root investigation: owner test computes a hit-tested center in Runtime.evaluate, then dispatches separate CDP mousePressed/mouseReleased; final Save-label is inside a feedback panel after short-viewport resize and tool exit. Sketch895 similarly computes Done geometry before native input. Shared cause is unproven; inspect rendered/presented layout, event targets, and model completion before proposing a fix. Existing #858 live-control helper guards hidden/disabled geometry but is specific to the live toggle. No code changes yet. Baseline head11fece5cfb3b7da9ab93e6e592ec4f940919887a.

Root captured with diagnostics and browser-only CPU throttle4:9/10 pass,1 failure46.273s total. Native Save coordinates260.09375 hit feedback-body after button moved248->206; model still label-required, generation0 allowed, host visible. [Capture](../evidence/000890/lost-click.md). Initial trace-only10/10 PASS34.626s. No passing retry used as fix. Hub plan c0cec94d posted before code; quick plan proceeding unless objected. Separate895/888 remain unclaimed until890 lands or shared-cause proof justifies a fold.

Controlled native regression RED with metrics-only helper: resize completed before page handlers, exit1,1.011s. Receipt helper GREEN1.677s. First combined count10 qualification exposed initial-navigation context retirement at the new receipt: initial metrics previously ran before the fixture-document wait. Move that existing wait before the first receipt, so every barrier is armed in the intended document; no navigation retry or longer deadline. This is fixture setup ordering, not a changed browser product policy. Combined run is being allowed to finish; its failures are retained, and the corrected source needs fresh verification.

Corrected document ordering plus resize receipt: controlled native regression and owner count10 PASS61.602s; browser-throttle4/trace owner count10 PASS46.813s with every Save click at218 after panel clamp; focused race PASS7.223s. Existing timeout and one-shot input remain. First combined qualification before document ordering correction failed48.349s (navigation context destruction), retained as failed evidence rather than passed over. Reader inventory PASS: new helper/regression uses only literal JS and private Chrome/t.TempDir, reads no repository artifact; no certificates changed. The ordering is semantic protocol behavior defended by the held-event regression and lesson, not an AST-identifiable Go API misuse.


Catalog TestTrackedGoFilesPassCatalog PASS8.158s, whitespace check PASS. Final recorded pre-close check and normal merge still required before landing.

BR-1 response: first close review REWORK solely because fenced Chrome cannot initialize its sandbox/GPU; static review found no defect. At exact requested head4b0b9f8e9d4d9b0c9d5c137bff879c5432319669 with no source changes, host focused tests count3 PASS18.909s; original metrics-only overlay on the same head fails held-event assertion1.347s exit1. [Pinned commands, environment, exit results, complete logs and hashes](../evidence/000890/host-proof.md). This supplies the requested executable counterexample and positive evidence without weakening Chrome isolation. Existing recorded affected run87687 remains queued on the code-identical head; no second check queued.

Final recorded affected attemptb5f8e0277b2b1c4d38cffd3a19e5e92b on4b0b9f8e9d4d9b0c9d5c137bff879c5432319669 passed six Go packages plus shell in3m24s, log SHA256 prefix728a4f325e6b, after3304s FIFO wait. No failure or retry. Close round2 SHIP/medium with no open findings; BR-1 withdrawn after reviewer verified host logs/hashes and code identity. Close evidencebd9a2cac5539a52f946cc37a2554bca2c048b885, measured0.46h trusted-window (build0.41,converge0.05), no typed actual. Next: normal merge owns main integration and its full suite; no manual main chase. #895, then888 remain the dispatched sequence after890 lands.

## Revisions

### 2026-10-25 — document-owned resize receipt

A resize receipt belongs to the current document. Require the existing fixture-document readiness before the initial resize, while retaining the original viewport and60s context. The captured final-click root cause and remaining scope are unchanged.
