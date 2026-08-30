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

## Open findings

- **BR-1** [Critical] `live-conformance-production-seam` ARCH-MOCK/ARCH-PURPOSE: live conformance does not exercise the production park lifecycle
- **BR-2** [Critical] `startup-recovery-must-not-block` ARCH-PURE/ARCH-CONSTRAINTS: construction-time recovery performs blocking Zellij teardown
- **BR-4** [Important] `boundary-verification-must-be-complete` The required post-fix same-tag/native-root smoke remains incomplete
