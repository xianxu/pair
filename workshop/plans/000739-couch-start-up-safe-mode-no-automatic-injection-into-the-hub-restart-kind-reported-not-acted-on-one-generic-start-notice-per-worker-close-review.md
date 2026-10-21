# Boundary Review — pair#739 (whole-issue close)

| field | value |
|-------|-------|
| issue | 739 — couch: start-up safe mode: no automatic injection into the hub, restart kind reported not acted on, one generic start notice per worker |
| repo | pair |
| issue file | workshop/issues/000739-couch-start-up-safe-mode-no-automatic-injection-into-the-hub-restart-kind-reported-not-acted-on-one-generic-start-notice-per-worker.md |
| boundary | whole-issue close |
| milestone | — |
| window | 79ea4ca9e5be10c67c5c356b7f7d4067ebd79a92..588e2ce61585d7106f3a1d168982b9b84bbfdcfc |
| command | sdlc close --issue 739 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T01:34:02-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The focused tests support the new automation controls, attachment-only startup, and worker-cue delivery rules. The boundary still needs a corrected Core concepts table and passing ordinary package suites: one startup test retains the old launch expectation, and two packages exceed their enforced write budgets.

```findings
findings:
  - id: new
    severity: Critical
    family: core-concept-traceability
    title: |
      Core concepts table contradicts the delivered implementation
    detail: |
      workshop/plans/000739-startup-safe-mode-plan.md:55 lists startup lifecycle provenance as a modified PURE entity in couchtty/console_lifecycle.go, but that file has no changes in the pinned range; the new suppression implementation is in couchcmd/lifecycle_notices.go. Line 67 also places ObserveStartupContext in startup.go although it is defined in startup_policy.go. Correct the locations, classifications and statuses, and add a Revisions entry. This is the Core concepts contradiction mandated as Critical by this review contract (ARCH-PURE).
  - id: new
    severity: Important
    family: behavioral-test-contract-alignment
    title: |
      Existing startup integration test still requires automatic agent launch
    detail: |
      cmd/internal/couchcmd/startup_observation_test.go:28 requires a non-nil startup handle and runner operations for an empty fixture. The new StartInteractive behavior correctly returns neither, so TestInteractiveHubStartupOpensConsoleWithSelectedChild fails both in the package run and in isolation. Update the test to assert a usable empty switcher without launching, and retain a separately seeded surviving-session attachment case (ARCH-PURPOSE).
  - id: new
    severity: Important
    family: test-operating-envelope
    title: |
      Ordinary couchcore and couchtty suites exceed enforced write budgets
    detail: |
      The five-package go test invocation failed couchcore with 154050560 bytes against its 134217728-byte cap and couchtty with 87949312 bytes against its 67108864-byte cap. The guards are applied at cmd/internal/couchcore/launchhelper_test.go:50 and cmd/internal/couchtty/reattach_testmain_test.go:21. Reduce fixture writes and verify the ordinary suites within their existing envelopes; direct in-memory fixtures for pure startup policy tests are one concrete opportunity (ARCH-CONSTRAINTS, ARCH-PURE).
```

1. **Strengths**

   - Generated broker input has one guarded ingress, with an AST test preventing additional direct `SendSystem` sites.
   - Worker-cue tests exercise supervisor replacement, lost acknowledgments, receipt eviction, and stale launch ordinals while counting native input.
   - Suppression tests cover delayed off/on transitions and verify that consumed notices do not replay.
   - README, atlas, help, and the shipped Couch skill document the new controls and distinct notice lifetimes.

2. **Critical findings**

   - The [Core concepts table](/Volumes/TestScratch/review-checkouts/6849008991c1da7f/workshop/plans/000739-startup-safe-mode-plan.md:55) needs accurate ownership, classification, and paths. This is a traceability blocker; it does not establish a runtime defect.

3. **Important findings**

   - Repair the obsolete [startup integration assertion](/Volumes/TestScratch/review-checkouts/6849008991c1da7f/cmd/internal/couchcmd/startup_observation_test.go:28).
   - Bring the ordinary package runs within their write budgets. The measured failures are concrete; attribution to individual fixtures still needs measurement.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed: affected-test inventory, pinned-range whitespace check, full `couchmessage` and `wrapcmd` package runs.
   - Passed: focused startup, automation, suppression, switcher, and worker-cue tests across four packages.
   - Failed: ordinary `couchcmd` startup assertion; ordinary `couchcore` and `couchtty` write-budget guards.
   - Coordinated real-supervisor acceptance remains unverified. No prior findings required disposition, and no tracked files were changed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared generated-input ingress and negotiation ladder.
   - **ARCH-PURE — flag:** inaccurate concept classification; pure policy tests also unnecessarily construct filesystem-backed fixtures.
   - **ARCH-PURPOSE — pass:** reviewed behavior covers safe startup and both worker-notice lifetimes; scheduler consumer ownership is explicitly assigned to the second lander.
   - **ARCH-MOCK — pass:** private stateful doubles and real private sockets share production seams; live qualification remains outstanding.
   - **ARCH-CONSTRAINTS — flag:** ordinary test write envelopes fail.
   - **ARCH-SECURE — pass:** current caller/owner checks, strict frames, and receiver-side launch validation.
   - **ARCH-ORDER — pass:** committed cue uncertainty stays closed; controlled ordering tests cover delayed admission and replacement.
   - **ARCH-FUNERAL — pass:** new memory owners have bounded capacity and process-scoped lifetimes.
   - **ARCH-EVOLVE — pass:** negotiated session fields preserve old-peer compatibility; no new persisted format.

7. **Plan revision recommendations**

   Add a `## Revisions` entry correcting lifecycle suppression ownership and classification, and locating `ObserveStartupContext` in `startup_policy.go`. Preserve the outstanding live-acceptance requirement until measured evidence is recorded.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"eeff0018-1f9c-4afc-8e8d-26785ef0f9be","repository":"github.com/xianxu/pair","issue_id":"000739","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/6849008991c1da7f","reviewed_head":"588e2ce61585d7106f3a1d168982b9b84bbfdcfc","slot":"pair:7","requested_at":"2026-10-21T08:28:24.707489Z"},"phase":"finished","pid":6743,"started_at":"2026-10-21T08:28:24.801199Z","finished_at":"2026-10-21T08:34:01.54105Z","session_id":"01a15d93-6c5b-7432-8021-b8d0aea3967d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.50205375s
head: "588e2ce61585d7106f3a1d168982b9b84bbfdcfc"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
