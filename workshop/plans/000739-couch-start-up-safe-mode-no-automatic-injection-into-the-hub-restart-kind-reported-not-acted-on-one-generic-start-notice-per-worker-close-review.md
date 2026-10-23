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

---

## Re-review — 2026-10-21T01:51:09-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 739 — couch: start-up safe mode: no automatic injection into the hub, restart kind reported not acted on, one generic start notice per worker |
| repo | pair |
| issue file | workshop/issues/000739-couch-start-up-safe-mode-no-automatic-injection-into-the-hub-restart-kind-reported-not-acted-on-one-generic-start-notice-per-worker.md |
| boundary | whole-issue close |
| milestone | — |
| window | 79ea4ca9e5be10c67c5c356b7f7d4067ebd79a92..ea07b0316494a7370f5f8dcc12688ce516d8a71b |
| command | sdlc close --issue 739 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T01:51:09-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

All three prior findings are addressed: concept ownership is corrected, startup integration expectations match safe mode, and the five ordinary package suites pass within their enforced write budgets. One newly verified failure blocks shipping: the repository’s dead-symbol guard rejects `SelectStartupHub`, which has tests but no production caller.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Plan lines 55, 67–72 now distinguish the pure lifecycle reducer from the IO publisher and correctly locate ObserveStartupContext. The pinned correction matches lifecycle_notice.go, lifecycle_notices.go and startup_policy.go, and adds the requested Revisions entry.
  - id: BR-2
    disposition: addressed
    note: |
      startup_observation_test.go:16 now asserts an empty usable switcher with no launch or retained-state changes; line 53 separately verifies seeded surviving-session attachment. Inspected the before/after correction; the ordinary couchcmd suite passes.
  - id: BR-3
    disposition: addressed
    note: |
      Pure classification fixtures now use direct snapshots, and bulk observation fixtures omit unrelated layout rewrites. The ordinary five-package command passes, including couchcore and couchtty under their unchanged write-budget guards on Darwin with cgo enabled.
findings:
  - id: new
    severity: Important
    kind: defect
    family: production-symbol-reachability
    title: |
      Test-only startup selector fails the repository dead-symbol guard
    detail: |
      cmd/internal/couchcore/startup_policy.go:174 adds SelectStartupHub, but every reference outside its declaration is in startup_policy_test.go; production StartInteractive selects candidates independently. Running go test ./cmd/internal/artifactpath -count=1 fails TestNoProductionSymbolIsReferencedOnlyByTests/cmd/internal/couchcore for this symbol. Connect the intended selector to production, or remove it and update its tests, plan and atlas claims; rerun the guard. The tested policy should have a production consumer (ARCH-PURPOSE).
```

1. **Strengths**

   - Generated system input has one guarded broker ingress, enforced by an AST source test.
   - Worker-cue tests count native input across supervisor replacement, lost acknowledgments, receipt eviction and stale ordinals.
   - Suppression tests exercise delayed off/on transitions and verify no replay.
   - README, atlas, help and the shipped Couch skill cover automation controls and both notice lifetimes.

2. **Critical findings**

   None.

3. **Important findings**

   - [SelectStartupHub](/Volumes/TestScratch/review-checkouts/3a8c72b06310f7ff/cmd/internal/couchcore/startup_policy.go:174) is unreachable from production and breaks an existing repository guard. Wire it into startup or remove the unused surface and corresponding claims.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed: ordinary `couchcore`, `couchcmd`, `couchmessage`, `couchtty` and `wrapcmd` suites.
   - Passed: affected-test inventory and pinned-range whitespace check.
   - Failed: `artifactpath` suite’s production-symbol reachability guard.
   - Coordinated real-supervisor acceptance remains outstanding as documented. No tracked files changed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared automatic-input ingress and negotiation ladder.
   - **ARCH-PURE — pass:** corrected classifications; pure policy fixtures avoid IO.
   - **ARCH-PURPOSE — flag:** the tested startup selector has no production consumer.
   - **ARCH-MOCK — pass:** stateful private doubles and sockets share production seams; live qualification remains pending.
   - **ARCH-CONSTRAINTS — pass:** ordinary suites satisfy unchanged write caps; counted tests cover 20/40-worker workloads.
   - **ARCH-SECURE — pass:** current caller/owner checks, strict frames and receiver-side ordinal validation.
   - **ARCH-ORDER — pass:** controlled delayed-admission tests and monotonic cue witnesses preserve uncertainty.
   - **ARCH-FUNERAL — pass:** new memory owners have bounded capacity and process lifetimes.
   - **ARCH-EVOLVE — pass:** negotiated session additions preserve old-peer compatibility; no persisted-format change.

7. **Plan revision recommendations**

   Add a `## Revisions` entry recording whether production startup adopts `SelectStartupHub` or removes it. If removed, update the Core concepts table and atlas passage claiming startup uses it. Retain the outstanding live-acceptance requirement.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1a2c6b9e-4afc-42b4-b1af-f2c74ef61779","repository":"github.com/xianxu/pair","issue_id":"000739","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/3a8c72b06310f7ff","reviewed_head":"ea07b0316494a7370f5f8dcc12688ce516d8a71b","slot":"pair:7","requested_at":"2026-10-21T08:44:30.673562Z"},"phase":"finished","pid":9282,"started_at":"2026-10-21T08:44:30.826356Z","finished_at":"2026-10-21T08:51:08.261918Z","session_id":"01a15da2-2a1c-78c2-820b-b464cee55443","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.244965583s
head: "ea07b0316494a7370f5f8dcc12688ce516d8a71b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T01:59:09-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 739 — couch: start-up safe mode: no automatic injection into the hub, restart kind reported not acted on, one generic start notice per worker |
| repo | pair |
| issue file | workshop/issues/000739-couch-start-up-safe-mode-no-automatic-injection-into-the-hub-restart-kind-reported-not-acted-on-one-generic-start-notice-per-worker.md |
| boundary | whole-issue close |
| milestone | — |
| window | 79ea4ca9e5be10c67c5c356b7f7d4067ebd79a92..eb0befce2a38a7d0d497659fc9073a543373d7b3 |
| command | sdlc close --issue 739 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T01:59:09-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-4 is addressed, and startup attachment, wrapper-owned cue delivery, compatibility, and documentation are well covered. One blocking ordering defect remains: automation checks occur before potentially delayed broker admission, allowing generated hub input after off completes or stale input after off/on.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The corrected Core concepts and IO seams tables match the delivered classifier, lifecycle reducer, publisher, and wrapper reporting locations.
  - id: BR-2
    disposition: addressed
    note: |
      Startup tests now assert attachment-only behavior and zero launches for missing or parked hubs; the full couchcmd rerun passes.
  - id: BR-3
    disposition: addressed
    note: |
      Ordinary couchcore and couchtty suites pass with their existing enforced write budgets.
  - id: BR-4
    disposition: addressed
    note: |
      SelectStartupHub and its false consumer claims are removed. The complete artifactpath suite passes; restoring the helper in a temporary checkout makes TestNoProductionSymbolIsReferencedOnlyByTests fail specifically for SelectStartupHub.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: automation-admission-ordering
    title: |
      Automation authorization expires before generated input actually enters the mailbox
    detail: |
      cmd/internal/couchcmd/automation.go:203 checks policy before broker.SendSystem performs routing and endpoint IO, with no policy check serialized with mailbox admission. A controlled test turned automation off inside the broker's families callback: off completed, yet sendGenerated returned a queued system receipt. Separately, resources_alert.go:51 captures an epoch but does not carry it through send: cycling off/on inside recipient resolution admitted the old alert and marked its episode notified. Both regression tests fail on the pinned head. Carry the occurrence's expected hub/epoch through the shared ingress and enforce it at actual admission, retiring invalidated occurrences. Sweep lifecycle, wait, stale-reminder, resource, stall, and supervision producers under this rule, with controlled delays inside broker admission as well as before send (ARCH-ORDER, ARCH-PURPOSE).
```

1. **Strengths**

   - Implicit startup preserves explicit lifecycle operations while refusing automatic cold resume or Fresh (`couchcore/startup.go:75`).
   - Wrapper-owned cue witnesses retain committed and uncertain outcomes across supervisor replacement; private-socket tests count native input.
   - README, atlas, help, and shipped Couch skill document automation controls and the two notice lifetimes.
   - BR-4 removes unused surface without an allowlist exemption or dummy production caller.

2. **Critical findings**

   - The automation admission defect above requires an end-to-end occurrence permit. Adding another check before `SendSystem` leaves its internal IO window exposed.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed: artifactpath, couchcore, couchmessage, couchtty, wrapcmd, affected-reader inventory, and range whitespace checks.
   - Couchcmd initially failed `TestBlockedThreadsReachTheConsoleView`; ten focused reruns and a complete package rerun passed.
   - The two temporary ordering regressions failed with actual mailbox admission, demonstrating the uncovered behavior. Temporary copies were removed; the reviewed checkout remains unchanged.
   - Coordinated live acceptance remains explicitly pending in the tracker.

6. **Architectural notes**

   | Principle | Assessment |
   |---|---|
   | ARCH-DRY | Pass: one generated-system broker ingress. |
   | ARCH-PURE | Pass: declared pure policies and reducers have direct tests; IO remains in adapters. |
   | ARCH-PURPOSE | **Flag:** automation suppression must hold through admission for every producer. |
   | ARCH-MOCK | Pass: stateful dependency fixtures and private sockets exercise production seams. |
   | ARCH-CONSTRAINTS | Pass: counted workload tests and ordinary package budgets pass. |
   | ARCH-SECURE | Pass: own-hub controls validate current caller authority; unknown cue capability grants no delivery permission. |
   | ARCH-ORDER | **Flag:** policy transitions and mailbox admission lack a shared ordering boundary. |
   | ARCH-FUNERAL | Pass: new witnesses and tracking structures are bounded and process-owned. |
   | ARCH-EVOLVE | Pass: negotiated session-v5 fields are omitted for older peers; no new persisted control format. |

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining the ordering of automation-off versus mailbox admission, how expected epochs survive broker IO, and the complete producer sweep. Require regression coverage for off and off/on during routing, observation, and reservation—not only before calling send.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"033c4dd3-3f70-40db-9154-fdcfc907c0b1","repository":"github.com/xianxu/pair","issue_id":"000739","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/d89f17a523e83bc7","reviewed_head":"eb0befce2a38a7d0d497659fc9073a543373d7b3","slot":"pair:7","requested_at":"2026-10-21T08:53:48.335096Z"},"phase":"finished","pid":32583,"started_at":"2026-10-21T08:53:48.427145Z","finished_at":"2026-10-21T08:59:09.086513Z","session_id":"01a15daa-ac12-7791-a28c-6eb3a1d60233","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.6540015s
head: "eb0befce2a38a7d0d497659fc9073a543373d7b3"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000410

---

## Re-review — 2026-10-21T02:21:57-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 739 — couch: start-up safe mode: no automatic injection into the hub, restart kind reported not acted on, one generic start notice per worker |
| repo | pair |
| issue file | workshop/issues/000739-couch-start-up-safe-mode-no-automatic-injection-into-the-hub-restart-kind-reported-not-acted-on-one-generic-start-notice-per-worker.md |
| boundary | whole-issue close |
| milestone | — |
| window | 79ea4ca9e5be10c67c5c356b7f7d4067ebd79a92..1b948b3c040f6979eaafa77b1ad2da127f524da4 |
| command | sdlc close --issue 739 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T02:21:57-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-5 is addressed: both regressions pass and fail when their respective fixes are removed in a disposable copy. Startup, automation, and worker-cue behavior match the documented contract. However, BR-3 must be reopened: the ordinary five-package test command still fails both enforced write budgets.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The plan separates pure lifecycle transitions in couchcore/lifecycle_notice.go from the couchcmd publisher and correctly locates ObserveStartupContext in startup_policy.go. The referenced entities and changes exist.
  - id: BR-2
    disposition: addressed
    note: |
      startup_observation_test.go now separately verifies an empty usable switcher without launches and attachment to a seeded surviving session. The couchcmd suite passes.
  - id: BR-3
    disposition: not-addressed
    note: |
      Family test-operating-envelope remains open (Important; ARCH-CONSTRAINTS). Plain go test across couchmessage, couchcmd, couchcore, couchtty and wrapcmd exits 1: couchcore writes 157257728 bytes against 134217728, and couchtty writes 88002560 against 67108864. Both suites pass with -count=1, which disables Go's cache-access logging; that control does not establish the failing ordinary invocation meets its envelope. Define and enforce the supported invocation across test callers, or bring cache-enabled execution within the existing caps.
  - id: BR-4
    disposition: addressed
    note: |
      The unused SelectStartupHub helper is removed. TestNoProductionSymbolIsReferencedOnlyByTests passes on the pinned head.
  - id: BR-5
    disposition: addressed
    note: |
      Broker direct admission and both enqueue paths enforce the original hub/epoch permit under broker then automation locks. Resource dispatch transports its original permit through recipient resolution. Both named regressions pass; removing the guard or recapturing the resource permit independently makes the corresponding regression fail. The six-producer sweep and focused race tests pass.
```

1. **Strengths**

   - `automation.go:203` provides one guarded generated-input ingress, backed by a source guard.
   - `broker.go:779` checks policy at mailbox commit while preserving already-admitted canonical receipts.
   - Worker-cue tests count native input across supervisor replacement, lost acknowledgments, receipt eviction, and launch-ordinal changes.
   - README, atlas, help, and the shipped skill document the controls and distinct notice lifetimes.

2. **Critical findings**

   None.

3. **Important findings**

   **BR-3 remains open.** The guards at `couchcore/launchhelper_test.go:50` and `couchtty/reattach_testmain_test.go:21` reject the ordinary invocation despite passing assertions. Address the invocation-wide operating-envelope rule rather than another isolated fixture. Go’s source confirms cache-enabled runs add filesystem-access logging; the successful `-count=1` control disables that instrumentation.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed: complete couchmessage, couchcmd, and wrapcmd suites.
   - Passed: couchcore and couchtty with `-count=1`; ordinary invocation fails their write budgets.
   - Passed: focused admission-ordering race tests, dead-symbol guard, affected-reader inventory, and diff whitespace check.
   - Both BR-5 mutation checks failed as expected; the disposable copy was removed.
   - Coordinated live acceptance remains pending under the issue’s explicit review-before-acceptance agreement.

6. **Architectural notes**

   | Principle | Assessment |
   |---|---|
   | ARCH-DRY | Pass: shared guarded ingress; no parallel policy stack found. |
   | ARCH-PURE | Pass: policy reducers have direct pure tests; IO adapters remain separate. |
   | ARCH-PURPOSE | Pass: all six generated-hub producer families transport permits. |
   | ARCH-MOCK | Pass: stateful endpoints, native doubles, and private sockets exercise production seams. |
   | ARCH-CONSTRAINTS | **Flag: BR-3**, ordinary test execution exceeds declared write envelopes. |
   | ARCH-SECURE | Pass: exact bindings and original hub/epoch constrain admission; unknown evidence grants no authority. |
   | ARCH-ORDER | Pass: controlled interleavings cover commit ordering, suppression, and canonical prior receipts. |
   | ARCH-FUNERAL | Pass: new witnesses and policy state are bounded and process-owned. |
   | ARCH-EVOLVE | Pass: no persisted-format migration; negotiated v5 omits new fields for older peers. |

7. **Plan revision recommendations**

   Add a `## Revisions` entry for BR-3 recording both invocation modes, their different outcomes, and the rule that test callers will enforce. Keep live acceptance explicitly pending.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"bb52c787-b7cd-435b-8b1b-56ef28f56dcb","repository":"github.com/xianxu/pair","issue_id":"000739","recipe":"milestone-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/7c7ea792647487b0","reviewed_head":"1b948b3c040f6979eaafa77b1ad2da127f524da4","slot":"pair:7","requested_at":"2026-10-21T09:15:23.991786Z"},"phase":"finished","pid":2175,"started_at":"2026-10-21T09:15:24.08619Z","finished_at":"2026-10-21T09:21:56.901006Z","session_id":"01a15dbe-7149-7880-9411-0eee00f874fc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.668953s
head: "1b948b3c040f6979eaafa77b1ad2da127f524da4"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409

---

## Re-review — 2026-10-21T03:07:32-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 739 — couch: start-up safe mode: no automatic injection into the hub, restart kind reported not acted on, one generic start notice per worker |
| repo | pair |
| issue file | workshop/issues/000739-couch-start-up-safe-mode-no-automatic-injection-into-the-hub-restart-kind-reported-not-acted-on-one-generic-start-notice-per-worker.md |
| boundary | whole-issue close |
| milestone | — |
| window | 79ea4ca9e5be10c67c5c356b7f7d4067ebd79a92..6792c49527b9cb76acd7efb4dbe53b717bbd9b33 |
| command | sdlc close --issue 739 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T03:07:32-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The implementation matches the approved startup and automation contract. Focused tests, race checks, the installed-command regression, repository lint and source-reader inventory passed. No new blocking findings. The disputed write-budget failure is pre-existing; verified artifacts do not establish a regression attributable to this change. Private-fleet acceptance and second-lander scheduler integration remain explicitly pending under the documented boundary agreement.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The plan now separates pure lifecycle transitions in couchcore/lifecycle_notice.go from publisher IO in couchcmd/lifecycle_notices.go, and identifies startup classification versus observation correctly. The referenced entities exist at those paths.
  - id: BR-2
    disposition: addressed
    note: |
      TestBareCouchInstalledCommand now asserts an empty switcher and zero automatic launch before explicitly submitting Start. The installed-command test passed in this review.
  - id: BR-4
    disposition: addressed
    note: |
      SelectStartupHub is removed; candidate assertions exercise ClassifyStartup. TestTrackedGoFilesPassCatalog passed.
  - id: BR-5
    disposition: addressed
    note: |
      Original permits reach the shared mailbox commit guard, including resource recipient resolution. Controlled routing, reservation and queue-path regressions exist and passed; focused race checks also passed.
  - id: BR-3
    disposition: withdrawn
    note: |
      Independently verified retained log and trace SHA256 hashes, trace sizes and operation counts. Baseline cold/warm runs already exceed both unchanged caps. The paired full-core control measures 137310208 baseline versus 134885376 branch bytes; neither changed nor unchanged test cohort grows. The traces substantiate Go cache access logging as the bulk source. This supports withdrawing branch attribution, while the existing envelope defect remains tracked by #765. No full measurement rerun or zero-incremental-cost claim is made.
```

1. **Strengths**

   - Implicit startup removes cold/Fresh fallback while preserving explicit lifecycle operations.
   - Generated input shares one admission boundary. The guard covers direct admission and both queue paths, while preserving previously admitted receipts.
   - Wrapper-owned cue evidence distinguishes launch incarnations and survives ordinary receipt eviction.
   - README, atlas and shipped help explain controls, suppression loss and the two notice lifetimes.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Focused behavioral tests passed across couchcore, couchmessage, couchcmd, couchtty and wrapcmd. Focused race tests passed for automation, admission, resource/lifecycle publication and worker cues. The installed-command test, dead-symbol catalog, reader inventory and range whitespace check passed. This review did not rerun the complete repository suite or private live acceptance.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared generated-send policy and negotiation version gates.
   - **ARCH-PURE — pass:** pure classifiers/reducers are distinct from observation and delivery IO.
   - **ARCH-PURPOSE — pass:** startup and existing producer families are covered; scheduler handoff is explicitly documented.
   - **ARCH-MOCK — pass:** affected behavior uses stateful fakes and private sockets; live qualification remains pending.
   - **ARCH-CONSTRAINTS — pass:** bounded ownership and counted 20/40-worker/startup tests; BR-3 attribution withdrawn.
   - **ARCH-SECURE — pass:** exact bindings, authenticated own-hub control and conservative unknown handling.
   - **ARCH-ORDER — pass:** original epochs survive delayed work; commit ordering and uncertainty have controlled regressions.
   - **ARCH-FUNERAL — pass:** bounded volatile owners and a constant-size wrapper witness introduce no durable artifact family.
   - **ARCH-EVOLVE — pass:** hello-v5 negotiates new fields while older sessions omit them; no persisted-format migration introduced.

7. **Plan revision recommendations:** None. Preserve the recorded pending acceptance and scheduler integration obligations.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"83a8240e-c9d9-4f4c-a5eb-5ccbec48ffa7","repository":"github.com/xianxu/pair","issue_id":"000739","recipe":"milestone-review","round":5,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/ad715e43bd426244","reviewed_head":"6792c49527b9cb76acd7efb4dbe53b717bbd9b33","slot":"pair:7","requested_at":"2026-10-21T10:05:00.947599Z"},"phase":"finished","pid":79916,"started_at":"2026-10-21T10:05:01.041435Z","finished_at":"2026-10-21T10:07:32.34469Z","session_id":"01a15deb-ddf9-7f12-9485-7977cbe8b36f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.51328475s
head: "6792c49527b9cb76acd7efb4dbe53b717bbd9b33"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-23T05:03:51-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 739 — couch: start-up safe mode: no automatic injection into the hub, restart kind reported not acted on, one generic start notice per worker |
| repo | pair |
| issue file | workshop/issues/000739-couch-start-up-safe-mode-no-automatic-injection-into-the-hub-restart-kind-reported-not-acted-on-one-generic-start-notice-per-worker.md |
| boundary | whole-issue close |
| milestone | — |
| window | 146a283c5a47b69f70e47cf0e91ef40e8ab167e7..3b38d4ebdea18c335bdf2540a478574db3783669 |
| command | sdlc close --issue 739 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T05:03:51-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The interdiff preserves both sides of the inspected conflict resolutions, and focused checks pass. One boundary blocker remains: the plan changes acceptance scope after close by replacing the outstanding disposable-fleet qualification with narrower completed acceptance. The narrow-review contract requires a full `sdlc close` for that change. No additional runtime defect was found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Prior disposition retained; the corrected concept table identifies lifecycle suppression ownership and ObserveStartupContext at their implemented paths.
  - id: BR-2
    disposition: addressed
    note: |
      Empty-switcher and seeded surviving-session startup tests pass, including ten repetitions of the selected cohort.
  - id: BR-3
    disposition: withdrawn
    note: |
      Prior withdrawal retained; this interdiff does not alter write budgets or reopen the previously resolved attribution finding.
  - id: BR-4
    disposition: addressed
    note: |
      Prior disposition retained; the automatic-input source guard and production artifact inventory pass.
  - id: BR-5
    disposition: addressed
    note: |
      Broker mailbox-commit and both queue-commit admission-policy tests pass; the interdiff preserves the admission guard.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: boundary-scope-control
    title: |
      scope changed after close — run a full `sdlc close`
    detail: |
      workshop/plans/000739-startup-safe-mode-plan.md:164 replaces the outstanding disposable-fleet acceptance requiring measured input/launch/notice counts and focus with completed full-shutdown acceptance plus existing socket/count tests. The new Revisions passage at lines 319–323 explicitly treats the rollout as complete while acknowledging the unrun matrix. This changes acceptance scope rather than recording a prior finding's fix (ARCH-PURPOSE). Submit the revised acceptance contract through a full sdlc close; this narrow fix review cannot admit that scope change.
```

1. **Strengths**

   - Conflict resolutions retain editor endpoint initialization and cleanup alongside automation control.
   - Protocol validation preserves orphan-payload restrictions and strict automation request shapes.
   - The startup fixture now holds publication explicitly, queues normal host termination, and joins `Run` during cleanup.
   - README and atlas explain recovery controls and distinguish startup history from current liveness.

2. **Critical findings**

   - The acceptance-scope change at `workshop/plans/000739-startup-safe-mode-plan.md:164` requires full-close review, as detailed above.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed focused startup, CLI, admission-guard, orphan-protocol, switcher and artifact-inventory tests. The selected startup/automation cohort also passed ten repetitions. `make test-affected-inventory` and pinned-range `git diff --check` passed. I did not independently rerun the full suite or live acceptance. The checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** no duplicated policy logic introduced.
   - **ARCH-PURE — pass:** wording changes preserve existing policy and IO boundaries; test synchronization stays in the fake host.
   - **ARCH-PURPOSE — flag:** acceptance coverage changed after close and needs the broader review.
   - **ARCH-EVOLVE — pass:** response fields are combined additively; no persisted format changes appear in this window.

7. **Plan revision recommendations**

   Add a `## Revisions` entry explicitly identifying the acceptance cases removed or replaced, their evidence, and any remaining obligations. Run a full `sdlc close` against that revised contract.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"28c6f05d-7fb4-4b81-8a5a-4cee479eded8","repository":"github.com/xianxu/pair","issue_id":"000739","recipe":"fix-review","round":6,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/fcdc0533bb27099f","reviewed_head":"3b38d4ebdea18c335bdf2540a478574db3783669","slot":"pair:7","requested_at":"2026-10-23T12:01:53.756269Z"},"phase":"finished","pid":21999,"started_at":"2026-10-23T12:01:53.920418Z","finished_at":"2026-10-23T12:03:50.538622Z","session_id":"01a168a3-989d-79a3-879d-3a9cd48d9bcf","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.928415709s
head: "3b38d4ebdea18c335bdf2540a478574db3783669"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-23T05:09:13-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 739 — couch: start-up safe mode: no automatic injection into the hub, restart kind reported not acted on, one generic start notice per worker |
| repo | pair |
| issue file | workshop/issues/000739-couch-start-up-safe-mode-no-automatic-injection-into-the-hub-restart-kind-reported-not-acted-on-one-generic-start-notice-per-worker.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5f64ce54412afec1982b296b641cd515034482a0..d2da63f3df609ea04cea240524d9c083d8c5c4db |
| command | sdlc close --issue 739 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T05:09:13-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The reviewed implementation matches the revised startup and automation contract. No new blocking defects found. BR-6 is addressed by submitting the acceptance substitution to this full close review; the removed native matrix is explicitly unclaimed. Targeted runtime, race, installed-command and source-guard checks passed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The Core concepts tables distinguish pure reducers from IO adapters and identify the delivered paths.
  - id: BR-2
    disposition: addressed
    note: |
      TestBareCouchInstalledCommand verifies zero automatic launch before explicit Start; it passed in this review.
  - id: BR-3
    disposition: withdrawn
    note: |
      Retain the prior withdrawal; no new operating-envelope finding is raised.
  - id: BR-4
    disposition: addressed
    note: |
      The unused selector remains removed; TestNoProductionSymbolIsReferencedOnlyByTests passed.
  - id: BR-5
    disposition: addressed
    note: |
      Original permits reach broker commit guards on direct and queued admission paths; controlled admission and cue tests passed with race detection.
  - id: BR-6
    disposition: addressed
    note: |
      Plan lines 326–337 explicitly submit the acceptance substitution to full close, enumerate replacement evidence and remove the unrun native matrix without claiming it passed. This review evaluates that revised contract.
```

1. **Strengths**

   - Implicit startup attaches survivors without cold/Fresh fallback (`cmd/internal/couchcore/startup.go:75`).
   - Automation checks remain locked through mailbox admission (`cmd/internal/couchmessage/broker.go:782`, `:804`).
   - Wrapper-owned cue evidence survives ordinary receipt eviction and rejects stale launch ordinals (`cmd/internal/wrapcmd/peer_delivery.go:773`).
   - README and atlas explain controls, recovery guidance and distinct notice lifetimes.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed targeted tests across couchcore, couchmessage, wrapcmd, couchcmd and couchtty; focused race tests; installed Couch PTY flow; artifact inventory and dead-symbol guards; and `git diff --check`. No full-suite rerun, native fleet matrix or mutation testing was performed during this review. Recorded live acceptance remains operator-reported evidence.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** generated input shares one policy ingress.
   - **ARCH-PURE — pass:** classification and reducers have direct tests; IO adapters remain separate.
   - **ARCH-PURPOSE — pass:** startup and producer families are covered under the revised contract.
   - **ARCH-MOCK — pass:** existing stateful fakes and private sockets exercise production seams.
   - **ARCH-CONSTRAINTS — pass:** counted 20/40-worker tests cover bounded proof work.
   - **ARCH-SECURE — pass:** exact bindings, ownership and launch ordinals fence generated input.
   - **ARCH-ORDER — pass:** epoch permits, commit guards and monotonic cue witnesses address controlled interleavings.
   - **ARCH-FUNERAL — pass:** new state is bounded and dies with its wrapper or supervisor.
   - **ARCH-EVOLVE — pass:** cue fields negotiate through v5; legacy messaging remains covered.

   #737 still owns combined scheduler wiring and qualification, as explicitly documented.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e3b77125-aea4-4e2c-a1c2-60c659a60eb3","repository":"github.com/xianxu/pair","issue_id":"000739","recipe":"milestone-review","round":7,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/8dde9a1c602f53d0","reviewed_head":"d2da63f3df609ea04cea240524d9c083d8c5c4db","slot":"pair:7","requested_at":"2026-10-23T12:07:10.341785Z"},"phase":"finished","pid":98254,"started_at":"2026-10-23T12:07:10.496077Z","finished_at":"2026-10-23T12:09:12.212366Z","session_id":"01a168a8-6d26-7543-8816-1e089c634d2c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.088610209s
head: "d2da63f3df609ea04cea240524d9c083d8c5c4db"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-23T09:31:42-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 739 — couch: start-up safe mode: no automatic injection into the hub, restart kind reported not acted on, one generic start notice per worker |
| repo | pair |
| issue file | workshop/issues/000739-couch-start-up-safe-mode-no-automatic-injection-into-the-hub-restart-kind-reported-not-acted-on-one-generic-start-notice-per-worker.md |
| boundary | whole-issue close |
| milestone | — |
| window | dd37e0ad135c4a91fb4de345a455f8ce829df488..9bc066d843f6345aa6aab8cf72b37849b15307ef |
| command | sdlc close --issue 739 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T09:31:42-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff resolves the three inventory conflicts without changing startup behavior, the Spec, or the implementation plan. Both sides’ manifest entries survive, reader certificates match the merged sources, and no new blocking defects were found. Previously disposed findings remain closed.

```findings
{}
```

1. **Strengths**
   - `cmd/internal/artifactpath/manifest.go:679` preserves the complete conflict union without lost or duplicated entries.
   - Both reader certificates match their source hashes. All 62 `run_test.go` tests remain registered in `SOURCE_GUARDS`.
   - Reader classifications remain consistent with their inputs: repository README scanning versus isolated fixtures.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** Passed `make test-affected-inventory`, targeted artifact classification/inventory tests, and range `git diff --check`. No full-suite rerun or mutation testing was performed; this window introduces no behavior-changing fix.

6. **Architectural notes**
   - **ARCH-DRY — pass:** inventory entries remain unique.
   - **ARCH-PURE — pass:** resolutions add no IO or business logic.
   - **ARCH-PURPOSE — pass:** all three resolutions preserve both sides’ intent.
   - **ARCH-EVOLVE — pass:** certificate schemas and dispositions remain unchanged.

   No new architectural or user-facing surface requires atlas or README updates.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7db49f05-0ccb-463d-93bd-0fb4a0c6c58f","repository":"github.com/xianxu/pair","issue_id":"000739","recipe":"fix-review","round":8,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/9d45e95d397c4561","reviewed_head":"9bc066d843f6345aa6aab8cf72b37849b15307ef","slot":"pair:7","requested_at":"2026-10-23T16:30:33.647878Z"},"phase":"finished","pid":60639,"started_at":"2026-10-23T16:30:33.8196Z","finished_at":"2026-10-23T16:31:41.13943Z","session_id":"01a16999-9117-7f51-8415-0b1ca8a5c2d7","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.200843375s
head: "9bc066d843f6345aa6aab8cf72b37849b15307ef"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
