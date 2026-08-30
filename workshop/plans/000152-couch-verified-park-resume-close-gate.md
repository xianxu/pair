---
gate: boundary-review
issue: 152
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-30T14:22:58-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: 'ARCH-MOCK/ARCH-PURPOSE: live conformance does not exercise the production park lifecycle'
          detail: cmd/internal/pairlifecycletest/fake.go:251 hard-codes RunConformanceScenario to *Fake, while cmd/internal/launcher/quit_lifecycle_live_test.go:49-89 runs only RunCleanup against real Zellij and compares six cleanup-stage labels. Request publication, duplicate TriggerQuit delivery, the production trigger in couchcore/artifactcollision.go, completion publication, restart recovery, holder-death recovery, and exact child-death proof are not compared fake-vs-real as the Spec and checked Plan require. Generalize the scenario over the production seam, implement a controlled real driver, require complete trace equality, and make a regression fail when the production TriggerQuit path is bypassed. Add artifactcollision*.go to the live-conformance workflow paths.
          family: live-conformance-production-seam
          round: 1
        - id: BR-2
          severity: Critical
          title: 'ARCH-PURE/ARCH-CONSTRAINTS: construction-time recovery performs blocking Zellij teardown'
          detail: cmd/internal/couchcore/couch.go:108 calls ReconcileActiveParks synchronously. park.go:318 enters runActiveAttempt with await=false, but park.go:355-364 calls TriggerQuit before checking await; production TriggerQuit reaches OSRuntime.DeleteSession with a background context and may wait up to five seconds. With multiple pending parks, ordinary Couch startup can block serially despite the documented non-blocking recovery contract. Move teardown/awaiting work to the bounded worker and keep construction to a non-blocking reconciliation pass. Add a barrier test proving a blocked TriggerQuit cannot block NewCouchWith and leaves the transaction occupied.
          family: startup-recovery-must-not-block
          round: 1
        - id: BR-3
          severity: Critical
          title: 'ARCH-PURPOSE: the Core Concepts table does not match the delivered entities'
          detail: workshop/plans/000152-couch-verified-park-resume-plan.md:21-79 names LifecycleIdentity, LifecycleStore, LifecycleLock, ExistingThreadLauncher, ConsoleOperationQueue, and FakePairLifecycle, none of which exist under those names. It also locates VerifiedPark and QuitLifecycleOps in the wrong files; the delivered entities are Identity, Store/OSRuntime.Lock, launchTrackedThread in launch_existing.go, operationQueue in operation_queue.go, Fake, VerifiedPark in parktransaction.go, and QuitLifecycleOps in pairlifecycle/cleanup.go. Append a plan revision with an accurate greppable Pure/Integration table and add a mechanical contract test that resolves every listed symbol and path.
          family: core-concepts-match-code
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-30T14:48:03-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: The real driver later deletes Zellij during cleanup and explicitly kills an unrelated sleep child, so it still passes if production TriggerQuit only writes the intent or is bypassed.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: TriggerQuit moved out of construction, but New still calls PairSession, whose production implementation executes unbounded Zellij list-sessions commands.
          round: 2
        - id: BR-3
          disposition: addressed
          note: The authoritative revision lists the 19 delivered declarations and the Go-AST contract test resolves every symbol and path.
          round: 2
      findings:
        - id: BR-4
          severity: Important
          title: The required post-fix same-tag/native-root smoke remains incomplete
          detail: The plan still leaves the manual Couch Alt+x and exact Resume smoke unchecked, and the issue Log records only the earlier defect-discovering smokes rather than final before/after tag and native-root evidence.
          family: boundary-verification-must-be-complete
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-08-30T15:36:42-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: The live scenario now reaches production TriggerQuit, the blocking attach handoff, typed cleanup/completion publication, exact child-death observation, and finalization; the main scenario structurally fails if TriggerQuit no longer releases the handoff.
          round: 3
        - id: BR-2
          disposition: addressed
          note: New performs no Pair-session query or teardown, and TestParkCoordinatorConstructorDoesNotQueryPairSession proves the external query is reached only by asynchronous recovery.
          round: 3
        - id: BR-4
          disposition: addressed
          note: The smoke item is checked and the issue Log records exact tags and native Claude roots preserved across final Leave and Resume observations.
          round: 3
      findings:
        - id: BR-5
          severity: Critical
          title: The tested park single-flight worker has no production call site
          detail: cmd/internal/couchcore/parkworker.go:54 implements the required per-address coalescing, but newParkWorker is referenced only by parkworker_test.go. Startup recovery calls PairLifecycleController.Recover directly at couch.go:73 while interactive Park, Retry, Recover, Abandon, and Leave call the same controller directly at operationdispatch.go:218-231. Recovery and UI work can therefore run two coordinators for one thread, contrary to the one-coordinator/coalescing contract. Install one controller-owned worker and route every lifecycle entrypoint, including startup recovery and Leave, through it; add a composition-level barrier test proving overlapping recovery and UI retry produce one trigger and one shared result.
          family: single-coordinator-per-thread
          round: 3
        - id: BR-6
          severity: Important
          title: The intent-only live mutation test passes on unrelated trigger errors
          detail: 'This is the 2nd finding in family live-conformance-production-seam. TestParkLifecycleLiveIntentOnlyMutation at park_lifecycle_live_test.go:240 accepts any RunConformanceScenario error, then checks only that the child remains live. In this review it passed because WriteQuitIntent failed with operation not permitted before publishing the intent, so it did not exercise the intended mutation. Do not patch only this error: establish the rule that every negative conformance mutation must prove it crossed its intended precondition and failed at the expected lifecycle stage. Assert the intent is durably observable and require the expected completion-timeout/stage error.'
          family: live-conformance-production-seam
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-08-30T15:51:42-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: The complete lifecycle conformance driver remains wired through the production trigger and shared cleanup boundary.
          round: 4
        - id: BR-2
          disposition: addressed
          note: Construction remains free of active recovery and external Pair/Zellij observation.
          round: 4
        - id: BR-3
          disposition: addressed
          note: The authoritative Core Concepts table is present and its declaration-resolution contract passes.
          round: 4
        - id: BR-4
          disposition: addressed
          note: The documented same-tag/native-root smoke evidence remains recorded; no contrary code drift was found.
          round: 4
        - id: BR-5
          disposition: addressed
          note: One controller-owned worker now serves every production lifecycle entrypoint, with a passing startup-recovery versus interactive-retry barrier regression.
          round: 4
        - id: BR-6
          disposition: addressed
          note: The mutation test now proves trigger delivery and durable matching intent, then requires failure specifically during completion observation.
          round: 4
      blocked: false
---

# Gate ledger — pair#152 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-30T14:22:58-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `live-conformance-production-seam` ARCH-MOCK/ARCH-PURPOSE: live conformance does not exercise the production park lifecycle
  cmd/internal/pairlifecycletest/fake.go:251 hard-codes RunConformanceScenario to *Fake, while cmd/internal/launcher/quit_lifecycle_live_test.go:49-89 runs only RunCleanup against real Zellij and compares six cleanup-stage labels. Request publication, duplicate TriggerQuit delivery, the production trigger in couchcore/artifactcollision.go, completion publication, restart recovery, holder-death recovery, and exact child-death proof are not compared fake-vs-real as the Spec and checked Plan require. Generalize the scenario over the production seam, implement a controlled real driver, require complete trace equality, and make a regression fail when the production TriggerQuit path is bypassed. Add artifactcollision*.go to the live-conformance workflow paths.
- **BR-2** [Critical] `startup-recovery-must-not-block` ARCH-PURE/ARCH-CONSTRAINTS: construction-time recovery performs blocking Zellij teardown
  cmd/internal/couchcore/couch.go:108 calls ReconcileActiveParks synchronously. park.go:318 enters runActiveAttempt with await=false, but park.go:355-364 calls TriggerQuit before checking await; production TriggerQuit reaches OSRuntime.DeleteSession with a background context and may wait up to five seconds. With multiple pending parks, ordinary Couch startup can block serially despite the documented non-blocking recovery contract. Move teardown/awaiting work to the bounded worker and keep construction to a non-blocking reconciliation pass. Add a barrier test proving a blocked TriggerQuit cannot block NewCouchWith and leaves the transaction occupied.
- **BR-3** [Critical] `core-concepts-match-code` ARCH-PURPOSE: the Core Concepts table does not match the delivered entities
  workshop/plans/000152-couch-verified-park-resume-plan.md:21-79 names LifecycleIdentity, LifecycleStore, LifecycleLock, ExistingThreadLauncher, ConsoleOperationQueue, and FakePairLifecycle, none of which exist under those names. It also locates VerifiedPark and QuitLifecycleOps in the wrong files; the delivered entities are Identity, Store/OSRuntime.Lock, launchTrackedThread in launch_existing.go, operationQueue in operation_queue.go, Fake, VerifiedPark in parktransaction.go, and QuitLifecycleOps in pairlifecycle/cleanup.go. Append a plan revision with an accurate greppable Pure/Integration table and add a mechanical contract test that resolves every listed symbol and path.

## Round 2 — 2026-08-30T14:48:03-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — not-addressed — The real driver later deletes Zellij during cleanup and explicitly kills an unrelated sleep child, so it still passes if production TriggerQuit only writes the intent or is bypassed.
- BR-2 — not-addressed — TriggerQuit moved out of construction, but New still calls PairSession, whose production implementation executes unbounded Zellij list-sessions commands.
- BR-3 — addressed — The authoritative revision lists the 19 delivered declarations and the Go-AST contract test resolves every symbol and path.

### Raised

- **BR-4** [Important] `boundary-verification-must-be-complete` The required post-fix same-tag/native-root smoke remains incomplete
  The plan still leaves the manual Couch Alt+x and exact Resume smoke unchecked, and the issue Log records only the earlier defect-discovering smokes rather than final before/after tag and native-root evidence.

## Round 3 — 2026-08-30T15:36:42-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The live scenario now reaches production TriggerQuit, the blocking attach handoff, typed cleanup/completion publication, exact child-death observation, and finalization; the main scenario structurally fails if TriggerQuit no longer releases the handoff.
- BR-2 — addressed — New performs no Pair-session query or teardown, and TestParkCoordinatorConstructorDoesNotQueryPairSession proves the external query is reached only by asynchronous recovery.
- BR-4 — addressed — The smoke item is checked and the issue Log records exact tags and native Claude roots preserved across final Leave and Resume observations.

### Raised

- **BR-5** [Critical] `single-coordinator-per-thread` The tested park single-flight worker has no production call site
  cmd/internal/couchcore/parkworker.go:54 implements the required per-address coalescing, but newParkWorker is referenced only by parkworker_test.go. Startup recovery calls PairLifecycleController.Recover directly at couch.go:73 while interactive Park, Retry, Recover, Abandon, and Leave call the same controller directly at operationdispatch.go:218-231. Recovery and UI work can therefore run two coordinators for one thread, contrary to the one-coordinator/coalescing contract. Install one controller-owned worker and route every lifecycle entrypoint, including startup recovery and Leave, through it; add a composition-level barrier test proving overlapping recovery and UI retry produce one trigger and one shared result.
- **BR-6** [Important] `live-conformance-production-seam` The intent-only live mutation test passes on unrelated trigger errors
  This is the 2nd finding in family live-conformance-production-seam. TestParkLifecycleLiveIntentOnlyMutation at park_lifecycle_live_test.go:240 accepts any RunConformanceScenario error, then checks only that the child remains live. In this review it passed because WriteQuitIntent failed with operation not permitted before publishing the intent, so it did not exercise the intended mutation. Do not patch only this error: establish the rule that every negative conformance mutation must prove it crossed its intended precondition and failed at the expected lifecycle stage. Assert the intent is durably observable and require the expected completion-timeout/stage error.

## Round 4 — 2026-08-30T15:51:42-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — The complete lifecycle conformance driver remains wired through the production trigger and shared cleanup boundary.
- BR-2 — addressed — Construction remains free of active recovery and external Pair/Zellij observation.
- BR-3 — addressed — The authoritative Core Concepts table is present and its declaration-resolution contract passes.
- BR-4 — addressed — The documented same-tag/native-root smoke evidence remains recorded; no contrary code drift was found.
- BR-5 — addressed — One controller-owned worker now serves every production lifecycle entrypoint, with a passing startup-recovery versus interactive-retry barrier regression.
- BR-6 — addressed — The mutation test now proves trigger delivery and durable matching intent, then requires failure specifically during completion observation.

## Open findings

(none — every finding has been disposed)
