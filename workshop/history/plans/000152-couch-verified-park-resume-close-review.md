# Boundary Review — pair#152 (whole-issue close)

| field | value |
|-------|-------|
| issue | 152 — couch: verified park and resume lifecycle |
| repo | pair |
| issue file | workshop/issues/000152-couch-verified-park-resume.md |
| boundary | whole-issue close |
| milestone | — |
| window | e9e267d0e9f41bfd5958bdb4872fdfcb79af0dc2..962a36f526cb76ac6700254752324003c81dff9c |
| command | sdlc close --issue 152 |
| reviewer | codex |
| timestamp | 2026-08-30T14:22:58-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The core park/resume implementation is well tested and generally fail-closed, but two runtime contracts are not delivered: construction-time recovery can synchronously block on Zellij teardown, and the purported live conformance test exercises only the cleanup reducer rather than the production lifecycle. The plan’s Core Concepts table also materially contradicts the delivered symbols and locations. These require correction and another boundary review.

```findings
findings:
  - id: new
    severity: Critical
    family: live-conformance-production-seam
    title: |
      ARCH-MOCK/ARCH-PURPOSE: live conformance does not exercise the production park lifecycle
    detail: |
      cmd/internal/pairlifecycletest/fake.go:251 hard-codes RunConformanceScenario to *Fake, while cmd/internal/launcher/quit_lifecycle_live_test.go:49-89 runs only RunCleanup against real Zellij and compares six cleanup-stage labels. Request publication, duplicate TriggerQuit delivery, the production trigger in couchcore/artifactcollision.go, completion publication, restart recovery, holder-death recovery, and exact child-death proof are not compared fake-vs-real as the Spec and checked Plan require. Generalize the scenario over the production seam, implement a controlled real driver, require complete trace equality, and make a regression fail when the production TriggerQuit path is bypassed. Add artifactcollision*.go to the live-conformance workflow paths.
  - id: new
    severity: Critical
    family: startup-recovery-must-not-block
    title: |
      ARCH-PURE/ARCH-CONSTRAINTS: construction-time recovery performs blocking Zellij teardown
    detail: |
      cmd/internal/couchcore/couch.go:108 calls ReconcileActiveParks synchronously. park.go:318 enters runActiveAttempt with await=false, but park.go:355-364 calls TriggerQuit before checking await; production TriggerQuit reaches OSRuntime.DeleteSession with a background context and may wait up to five seconds. With multiple pending parks, ordinary Couch startup can block serially despite the documented non-blocking recovery contract. Move teardown/awaiting work to the bounded worker and keep construction to a non-blocking reconciliation pass. Add a barrier test proving a blocked TriggerQuit cannot block NewCouchWith and leaves the transaction occupied.
  - id: new
    severity: Critical
    family: core-concepts-match-code
    title: |
      ARCH-PURPOSE: the Core Concepts table does not match the delivered entities
    detail: |
      workshop/plans/000152-couch-verified-park-resume-plan.md:21-79 names LifecycleIdentity, LifecycleStore, LifecycleLock, ExistingThreadLauncher, ConsoleOperationQueue, and FakePairLifecycle, none of which exist under those names. It also locates VerifiedPark and QuitLifecycleOps in the wrong files; the delivered entities are Identity, Store/OSRuntime.Lock, launchTrackedThread in launch_existing.go, operationQueue in operation_queue.go, Fake, VerifiedPark in parktransaction.go, and QuitLifecycleOps in pairlifecycle/cleanup.go. Append a plan revision with an accurate greppable Pure/Integration table and add a mechanical contract test that resolves every listed symbol and path.
```

1. Strengths

- Park finalization requires both a matching completion and exact process death before releasing admission ([park.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/park.go:438)).
- Lifecycle publication uses immutable files, stable locking, file sync, rename, and directory sync, with explicit indeterminate outcomes ([store.go](/Users/xianxu/workspace/pair/cmd/internal/pairlifecycle/store.go:93)).
- Resume rechecks the established native binding after durable admission and before child effects ([resume.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/resume.go:218)).
- The terminal path renders progress before bounded asynchronous submission and keeps lifecycle work off the event loop ([console.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:1218)).

2. Critical findings

Three, detailed in the machine-readable findings block above.

3. Important findings

None beyond the Critical findings.

4. Minor findings

None.

5. Test coverage notes

- `go test -p 20 ./... -count=1` passed.
- Focused lifecycle, launcher, Couch core, TTY, and command suites passed.
- `go vet` passed for the affected packages.
- The opt-in live probe could not be independently validated in this sandbox: `/bin/ps` was denied, causing `TestQuitLifecycleLive` to time out. Independently of that environmental failure, source inspection proves the probe does not execute the declared full production scenario.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass; cleanup, existing-thread launch, operation declarations, and artifact paths have shared authorities.
- `ARCH-PURE`: Flag; startup composition synchronously enters blocking external teardown.
- `ARCH-PURPOSE`: Flag; full fake-vs-real lifecycle conformance is under-delivered and the concept inventory is stale.
- `ARCH-MOCK`: Flag; the live and fake flows do not share one complete scenario driver.
- `ARCH-CONSTRAINTS`: Flag; pending recovery can add approximately five seconds per thread to startup.

7. Plan revision recommendations

Append a `## Revisions` entry recording the final exact symbol/path map for all Core Concepts. Preserve the existing conformance and non-blocking-startup requirements; the implementation should be brought into agreement rather than weakening those contracts.

---

## Re-review — 2026-08-30T14:48:03-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 152 — couch: verified park and resume lifecycle |
| repo | pair |
| issue file | workshop/issues/000152-couch-verified-park-resume.md |
| boundary | whole-issue close |
| milestone | — |
| window | e9e267d0e9f41bfd5958bdb4872fdfcb79af0dc2..022b5a9698aab40f66a7e4afeac3a4d2ea906022 |
| command | sdlc close --issue 152 |
| reviewer | codex |
| timestamp | 2026-08-30T14:48:03-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The hermetic suite passes and BR-3 is genuinely fixed, but BR-1 and BR-2 remain open. The live driver does not causally exercise production-triggered child death, while constructor recovery still performs a potentially unbounded Zellij query. The final manual same-tag/native-root smoke is also still unchecked.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      The real driver later deletes Zellij during cleanup and explicitly kills an unrelated sleep child, so it still passes if production TriggerQuit only writes the intent or is bypassed.
  - id: BR-2
    disposition: not-addressed
    note: |
      TriggerQuit moved out of construction, but New still calls PairSession, whose production implementation executes unbounded Zellij list-sessions commands.
  - id: BR-3
    disposition: addressed
    note: |
      The authoritative revision lists the 19 delivered declarations and the Go-AST contract test resolves every symbol and path.
findings:
  - id: new
    severity: Important
    family: boundary-verification-must-be-complete
    title: |
      The required post-fix same-tag/native-root smoke remains incomplete
    detail: |
      The plan still leaves the manual Couch Alt+x and exact Resume smoke unchecked, and the issue Log records only the earlier defect-discovering smokes rather than final before/after tag and native-root evidence.
```

1. Strengths

- The startup regression correctly proves that a blocked `TriggerQuit` no longer blocks construction and that the park remains occupied (`cmd/internal/couchcore/park_test.go:645`).
- `runActiveAttempt` now checks `await` before triggering teardown (`cmd/internal/couchcore/park.go:355`).
- The revised Core Concepts table is accurate and mechanically resolved through Go AST declarations (`workshop/plans/000152-couch-verified-park-resume-plan.md:755`, `cmd/internal/couchcore/plan_contract_test.go:686`).
- README, atlas, and workflow path coverage were updated for the new lifecycle surface.

2. Critical findings

- **BR-1 — not addressed; second review in family `live-conformance-production-seam` (ARCH-MOCK, ARCH-PURPOSE).**
  `realParkConformanceDriver.ObserveChildDeath` kills and reaps an independent `sleep` process itself (`cmd/internal/couchcore/park_lifecycle_live_test.go:149`). Meanwhile, later cleanup deletes the Zellij session (`:66-67`). Consequently, the scenario would still pass if `ScopedThreadArtifactCollisionChecker.TriggerQuit` stopped deleting the session or were replaced with a direct intent write.
  Fix the rule, not this instance: the controlled child must be the actual launcher blocked in the Zellij handoff. `DeliverTrigger` alone must cause that handoff to return, execute production cleanup/completion publication, and terminate the recorded child. The driver should only observe death and committed evidence—not manufacture them. Add a mutation regression proving bypassing/removing production `TriggerQuit` makes the test fail.

- **BR-2 — not addressed; second review in family `startup-recovery-must-not-block` (ARCH-PURE, ARCH-CONSTRAINTS).**
  `New` still synchronously calls `ReconcileActiveParks` (`cmd/internal/couchcore/couch.go:133`). That calls `PairSession` (`cmd/internal/couchcore/park.go:304`), whose production implementation calls `runtime.Sessions()` (`cmd/internal/couchcore/artifactcollision.go:156`). `ZellijSource.run` uses `exec.Command(...).Run()` without a context or timeout (`cmd/internal/launcher/zellij.go:50`). A wedged Zellij query can therefore still block Couch startup indefinitely.
  Move every external session query into the context-bound recovery worker. Constructor work must be local durable inspection only. Extend the barrier regression to block `PairSession`, not merely `TriggerQuit`.

3. Important findings

- The required manual acceptance remains unchecked at `workshop/plans/000152-couch-verified-park-resume-plan.md:522`. Run the corrected lifecycle through Couch, record the exact tag/native root before and after Resume in the issue Log, and check the item.

4. Minor findings

None.

5. Test coverage notes

- `go test -p 20 ./... -count=1`: passed.
- Focused remediation and cross-package lifecycle suites: passed.
- `git diff --check`: passed.
- `make test-couch-zellij-live`: could not complete in this sandbox because `/bin/ps` execution was denied; source inspection independently establishes BR-1.
- BR-3’s contract test would fail if a listed declaration/path were reverted.
- BR-2’s barrier catches the old trigger ordering but does not catch blocking session discovery.
- BR-1 has no regression that fails when the production trigger is bypassed.

6. Architectural notes for upcoming work

- `ARCH-DRY`: pass—shared cleanup and conformance scenario abstractions avoid parallel lifecycle reducers.
- `ARCH-PURE`: flag—constructor recovery still crosses into external Zellij IO.
- `ARCH-PURPOSE`: flag—the live test demonstrates a scripted trace, not causally verified production lifecycle behavior.
- `ARCH-MOCK`: flag—the fake/real comparison does not share the complete production handoff-to-child-death boundary.
- `ARCH-CONSTRAINTS`: flag—startup still contains an unbounded external command despite the declared non-blocking envelope.

7. Plan revision recommendations

Append a round-two `## Revisions` entry stating:

- construction invokes no `PairSession` or other external command;
- recovery’s context-bound worker owns all session observation and teardown;
- live conformance launches a causally connected production launcher child and only observes its death;
- bypassing production `TriggerQuit` must fail the regression.

The currently checked remediation rows for full conformance and non-blocking startup should not be treated as delivered until those class-wide rules hold.

---

## Re-review — 2026-08-30T15:36:42-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 152 — couch: verified park and resume lifecycle |
| repo | pair |
| issue file | workshop/issues/000152-couch-verified-park-resume.md |
| boundary | whole-issue close |
| milestone | — |
| window | e9e267d0e9f41bfd5958bdb4872fdfcb79af0dc2..dd5f6e2192c6f327e265ec104f9633aa494b9858 |
| command | sdlc close --issue 152 |
| reviewer | codex |
| timestamp | 2026-08-30T15:36:42-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The lifecycle design is substantially implemented, and BR-1, BR-2, and BR-4 are addressed. However, the production coordinator bypasses its own single-flight worker, allowing startup recovery and interactive operations to coordinate the same thread concurrently. The live mutation regression also accepts unrelated setup failures as success. These block the boundary.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The live scenario now reaches production TriggerQuit, the blocking attach handoff, typed cleanup/completion publication, exact child-death observation, and finalization; the main scenario structurally fails if TriggerQuit no longer releases the handoff.
  - id: BR-2
    disposition: addressed
    note: |
      New performs no Pair-session query or teardown, and TestParkCoordinatorConstructorDoesNotQueryPairSession proves the external query is reached only by asynchronous recovery.
  - id: BR-4
    disposition: addressed
    note: |
      The smoke item is checked and the issue Log records exact tags and native Claude roots preserved across final Leave and Resume observations.
findings:
  - id: new
    severity: Critical
    family: single-coordinator-per-thread
    title: |
      The tested park single-flight worker has no production call site
    detail: |
      cmd/internal/couchcore/parkworker.go:54 implements the required per-address coalescing, but newParkWorker is referenced only by parkworker_test.go. Startup recovery calls PairLifecycleController.Recover directly at couch.go:73 while interactive Park, Retry, Recover, Abandon, and Leave call the same controller directly at operationdispatch.go:218-231. Recovery and UI work can therefore run two coordinators for one thread, contrary to the one-coordinator/coalescing contract. Install one controller-owned worker and route every lifecycle entrypoint, including startup recovery and Leave, through it; add a composition-level barrier test proving overlapping recovery and UI retry produce one trigger and one shared result.
  - id: new
    severity: Important
    family: live-conformance-production-seam
    title: |
      The intent-only live mutation test passes on unrelated trigger errors
    detail: |
      This is the 2nd finding in family live-conformance-production-seam. TestParkLifecycleLiveIntentOnlyMutation at park_lifecycle_live_test.go:240 accepts any RunConformanceScenario error, then checks only that the child remains live. In this review it passed because WriteQuitIntent failed with operation not permitted before publishing the intent, so it did not exercise the intended mutation. Do not patch only this error: establish the rule that every negative conformance mutation must prove it crossed its intended precondition and failed at the expected lifecycle stage. Assert the intent is durably observable and require the expected completion-timeout/stage error.
```

1. Strengths

- `couch.go:82-134` keeps construction free of active-park external IO, with a reachable barrier regression at `park_test.go:645-733`.
- `park_lifecycle_live_test.go:436-491` places the controlled helper through the production attach and cleanup/completion boundaries.
- The main live scenario compares complete semantic traces through finalization, not merely cleanup labels.
- README and atlas updates cover Park, Leave, Resume, lifecycle authority, recovery, and terminal restoration.
- The authoritative Core Concepts table resolves successfully through its Go-AST contract test.

2. Critical findings

- `parkworker.go:54`, `couch.go:73`, `operationdispatch.go:218-231`: wire the currently dead single-flight abstraction into every production lifecycle entrypoint.

3. Important findings

- `park_lifecycle_live_test.go:240-250`: require the mutation probe to prove intent publication and the expected failure stage.

4. Minor findings

None.

5. Test coverage notes

- Focused lifecycle/launcher/Couch suites passed.
- `git diff --check` passed.
- The positive live scenario could not complete because the sandbox denied writing its production `~/.cache/pair/quit-*` marker.
- The negative live mutation test passed for that same unrelated permission failure, exposing the Important finding.
- `go test -p 20 ./... -count=1` otherwise passed but ended with sandbox-denied `/bin/ps` execution in `cmd/pair-go`.

6. Architectural notes for upcoming work

- `ARCH-DRY`: pass; cleanup, attach handoff, lifecycle vocabulary, and terminal reset are shared.
- `ARCH-PURE`: pass; transition logic remains separated from IO, and construction no longer performs external recovery.
- `ARCH-PURPOSE`: flag; the promised one-coordinator/coalescing behavior is not connected to production.
- `ARCH-MOCK`: flag; the negative live mutation can pass without reaching its modeled precondition.
- `ARCH-CONSTRAINTS`: flag; lifecycle concurrency is not enforced by the declared bounded worker, although construction latency itself is protected.

7. Plan revision recommendations

Append a revision recording:

- One controller-owned single-flight boundary shared by startup recovery, Park/Retry/Recover/Abandon, and Leave.
- A consuming-flow overlap test proving recovery and interactive retry coalesce.
- A live-mutation rule requiring proof of the intended precondition and an expected-stage failure, rather than accepting any error.

---

## Re-review — 2026-08-30T15:51:42-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 152 — couch: verified park and resume lifecycle |
| repo | pair |
| issue file | workshop/issues/000152-couch-verified-park-resume.md |
| boundary | whole-issue close |
| milestone | — |
| window | e9e267d0e9f41bfd5958bdb4872fdfcb79af0dc2..d04744774f2ae1fd8a431af0509d912b5f1ced48 |
| command | sdlc close --issue 152 |
| reviewer | codex |
| timestamp | 2026-08-30T15:51:42-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The reviewed remediation appears correct and no new code defect was found. BR-5 and BR-6 are addressed with reachable regressions. However, required live/full verification could not complete because the review sandbox denied `/bin/ps` execution with `operation not permitted`; per the inspection contract, the boundary must be rerun in an environment that permits the production process-observation seam.

### 1. Strengths

- The production controller now owns the single-flight worker and routes Park, Retry, Recover, Abandon, reconciliation, and Leave through it ([park.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/park.go:157)).
- The composition-level barrier test overlaps startup recovery with dispatched interactive retry and verifies one trigger plus one shared result ([parkworker_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/parkworker_test.go:60)).
- The negative live mutation now pins the exact trace, completion-observation timeout, durable intent contents, and surviving helper ([park_lifecycle_live_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/park_lifecycle_live_test.go:241)).
- README and atlas changes cover the new commands, lifecycle flow, recovery behavior, and coordinator boundary.

### 2. Critical findings

None found in the inspected code.

### 3. Important findings

None found in the inspected code.

### 4. Minor findings

None.

### 5. Test coverage notes

- Focused coordinator and Core Concepts tests passed.
- All internal Go packages passed.
- `go vet ./...` and `git diff --check` passed.
- `make test-couch-zellij-live` failed before reaching Couch tests because launcher probes could not execute `/bin/ps`.
- The full suite likewise failed only in `cmd/pair-go` cases requiring `/bin/ps`.
- The worktree was clean after verification.

### 6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass—one controller-owned worker is the lifecycle coordination source.
- `ARCH-PURE`: Pass—coordination remains separated from durable and external IO seams.
- `ARCH-PURPOSE`: Pass—the production entrypoints, not merely the worker unit test, enforce the stated invariant.
- `ARCH-MOCK`: Structurally passes; stateful fake and production driver share the boundary. Live execution remains unverified because of the sandbox restriction.
- `ARCH-CONSTRAINTS`: Pass—capacity-one admission and bounded behavior are explicit and tested deterministically.

### 7. Plan revision recommendations

None; the latest revision matches the delivered declarations and routing.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The complete lifecycle conformance driver remains wired through the production trigger and shared cleanup boundary.
  - id: BR-2
    disposition: addressed
    note: |
      Construction remains free of active recovery and external Pair/Zellij observation.
  - id: BR-3
    disposition: addressed
    note: |
      The authoritative Core Concepts table is present and its declaration-resolution contract passes.
  - id: BR-4
    disposition: addressed
    note: |
      The documented same-tag/native-root smoke evidence remains recorded; no contrary code drift was found.
  - id: BR-5
    disposition: addressed
    note: |
      One controller-owned worker now serves every production lifecycle entrypoint, with a passing startup-recovery versus interactive-retry barrier regression.
  - id: BR-6
    disposition: addressed
    note: |
      The mutation test now proves trigger delivery and durable matching intent, then requires failure specifically during completion observation.
```

---

## Re-review — 2026-08-30T15:58:01-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 152 — couch: verified park and resume lifecycle |
| repo | pair |
| issue file | workshop/issues/000152-couch-verified-park-resume.md |
| boundary | whole-issue close |
| milestone | — |
| window | e9e267d0e9f41bfd5958bdb4872fdfcb79af0dc2..dd18a9aaa7aa15e8af2bb9af6e0c04f36cade3bb |
| command | sdlc close --issue 152 |
| reviewer | codex |
| timestamp | 2026-08-30T15:58:01-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The implementation matches the authoritative Spec and Plan, all prior findings remain addressed, and no new defects were found. Hermetic lifecycle suites and mutation checks passed; confidence is medium only because the sandbox denied `/bin/ps` and writes to the resolved user cache, preventing independent live-Zellij execution.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The shared scenario crosses request publication, production TriggerQuit, Pair cleanup/completion, exact child death, and finalization; the intent-only driver proves the trigger is causal.
  - id: BR-2
    disposition: addressed
    note: |
      Construction performs no Pair/Zellij observation; reintroducing constructor recovery makes TestParkCoordinatorConstructorDoesNotQueryPairSession fail.
  - id: BR-3
    disposition: addressed
    note: |
      All 21 authoritative Core Concepts resolve to declarations; mutating a listed symbol makes the contract test fail.
  - id: BR-4
    disposition: addressed
    note: |
      The checked acceptance item and issue Log record exact same-tag/native-root observations from final operator smokes.
  - id: BR-5
    disposition: addressed
    note: |
      Every production lifecycle entrypoint uses the controller-owned worker; bypassing it makes the startup-recovery/interactive-retry overlap regression fail.
  - id: BR-6
    disposition: addressed
    note: |
      The negative live test requires both trigger deliveries, exact durable intent, and a completion-observation deadline; an earlier environmental failure is explicitly rejected.
```

1. Strengths

- Park releases capacity only after matching completion and exact process death.
- Resume rechecks the established native binding after durable admission and before child effects.
- Crash-safe lifecycle publication uses immutable records, file and directory synchronization, and stable advisory locking.
- README, atlas, workflow filters, and the authoritative concept inventory cover the new surface.

2. Critical findings

None.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

- Focused lifecycle, launcher, Couch core, TTY, and command suites passed.
- All internal packages passed; `go vet` and `git diff --check` passed.
- Lua, terminal shortcut, review-toggle, and Zellij configuration/layout checks passed.
- Three scratch-copy mutation checks failed as intended for BR-2, BR-3, and BR-5.
- Repository-wide execution reached `cmd/pair-go` but its `/bin/ps` cases were sandbox-denied. Live conformance likewise stopped on sandbox-denied cache writes, before its asserted precondition.
- Worktree was left clean.

6. Architectural notes

- `ARCH-DRY`: Pass—one cleanup authority, lifecycle protocol, worker, and existing-thread launch path.
- `ARCH-PURE`: Pass—pure transition/eligibility logic is separated from injected IO.
- `ARCH-PURPOSE`: Pass—the delivered flow covers verified park, exact resume, recovery, and production conformance.
- `ARCH-MOCK`: Pass—stateful fake and live driver share the complete scenario boundary.
- `ARCH-CONSTRAINTS`: Pass—bounded queueing, single coordination, deadlines, overload behavior, and latency evidence are explicit.

7. Plan revision recommendations

None.
