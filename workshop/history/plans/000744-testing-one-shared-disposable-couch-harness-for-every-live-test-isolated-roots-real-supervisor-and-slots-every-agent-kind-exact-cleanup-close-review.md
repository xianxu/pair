# Boundary Review — pair#744 (whole-issue close)

| field | value |
|-------|-------|
| issue | 744 — testing: one shared disposable-Couch harness for every live test (isolated roots, real supervisor and slots, every agent kind, exact cleanup) |
| repo | pair |
| issue file | workshop/issues/000744-testing-one-shared-disposable-couch-harness-for-every-live-test-isolated-roots-real-supervisor-and-slots-every-agent-kind-exact-cleanup.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6df5e89ee15015626ed2e72f99cd26f3b66b1dbe..6d584dc279b5745064cabd5a8e59f01da3c6e8d6 |
| command | sdlc close --issue 744 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T00:41:29-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared harness successfully exercised both native personalities at the pinned head, including messaging, detach/restart, and exact cleanup. Blocking gaps remain: the transport change breaks existing ordinary tests, request cancellation leaves fixture work running, and the Core concepts table misclassifies filesystem-backed correlation. Ownership-manifest write errors also disappear silently.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: isolation-contract-consumer-coverage
    title: |
      The new socket-length requirement breaks existing isolated consumers
    detail: |
      cmd/internal/couchmessage/transport.go:65 rejects existing test roots. The ordinary package run failed TestAttentionServiceIgnoresLegacyListAndDoesNotPersistEvents, TestScratchServiceDrivesMaintenanceAndJoinsOnClose, and TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub with the socket-length error; the first two reproduced independently. ARCH-PURPOSE: sweep isolated consumers and give them owned short roots while preserving private-parent confinement and overflow refusal.
  - id: new
    severity: Critical
    kind: defect
    family: request-cancellation-joins-effects
    title: |
      ActorCLI cancellation leaves published commands and children running
    detail: |
      cmd/internal/testcouch/actor.go:78-94 returns when the caller context ends without canceling or retiring the published request. agentfixture/run.go:369 executes that request under the fixture-wide context, with its relative timeout starting when execution begins. Consequently canceled queued work can start later, active work can continue mutating the fleet, and abandoned results consume mailbox capacity. ARCH-ORDER: add correlated cancellation and acknowledgment, join active children, suppress canceled queued execution, and reclaim terminal request state.
  - id: new
    severity: Critical
    kind: defect
    family: core-concept-classification
    title: |
      The Core concepts table classifies filesystem-backed correlation as PURE
    detail: |
      workshop/plans/000744-shared-native-couch-plan.md:30 lists request correlation as PURE, but its implementation in agentfixture/protocol.go:214 and :241 reads and removes mailbox files, and its coverage uses filesystem fixtures. ARCH-PURE: classify mailbox correlation as INTEGRATION or extract an independently tested pure correlation function. Correct the paint-adapter location to paint.go and align the Command/Result names in the same table revision.
  - id: new
    severity: Important
    kind: defect
    family: recovery-evidence-write-errors
    title: |
      Ownership-manifest failures are silently discarded
    detail: |
      cmd/internal/testcouch/fleet.go:157-163 ignores ownership.json write errors and rewrites the file in place. A failed or partial write can leave cleanup-failure or crash recovery without the promised readable ownership evidence while execution continues normally. ARCH-FUNERAL: publish complete snapshots atomically, surface persistence failures, and preserve cleanup ownership in memory; cover a failed publication.
```

1. **Strengths**

   - `make test-couch-fleet` passed at head `6d584dc…` in **74.59s**, covering genuine steward/three-worker cohorts for Claude and Codex, public receipts, independent accepted-body evidence, restart, and root removal.
   - Transport tests pin unchanged unisolated addresses and verify foreign socket residue survives isolated listener startup.
   - Mailbox tests exercise concurrent admission, consumed-result reclamation, stale shutdowns, and reserved shutdown capacity.
   - README, atlas, artifact ownership, and reader inventories were updated.

2. **Critical findings**

   - **Socket consumers:** [transport.go:65](cmd/internal/couchmessage/transport.go:65). Migrate the affected root factories and sweep sibling isolated consumers.
   - **Request cancellation:** [actor.go:78](cmd/internal/testcouch/actor.go:78), [run.go:369](cmd/internal/testcouch/agentfixture/run.go:369). Add per-request cancellation with completion acknowledgment and joining.
   - **Core classification:** [plan:30](workshop/plans/000744-shared-native-couch-plan.md:30), [protocol.go:214](cmd/internal/testcouch/agentfixture/protocol.go:214). Separate pure validation from mailbox IO in the table.

3. **Important findings**

   - **Recovery evidence:** [fleet.go:157](cmd/internal/testcouch/fleet.go:157). Make manifest publication atomic and failures visible without abandoning teardown.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed: portable helper tests, helper race tests, `couchmessage`, `wrapcmd`, `artifactpath`, all 14 reader-inventory tests, and the native matrix.

   Failed: the ordinary `couchcmd` and `cmd/couch` packages as detailed above. Existing cancellation coverage cancels the entire fixture; add controlled tests canceling one queued request and one already-started request while the fixture remains alive.

6. **Architectural notes**

   | Principle | Assessment |
   |---|---|
   | ARCH-DRY | Pass: shared path confinement, provisioning runner, and production identity helpers are reused. |
   | ARCH-PURE | Flag: mailbox correlation classification contradicts its IO implementation. |
   | ARCH-PURPOSE | Flag: the new isolation contract leaves existing consumers broken. |
   | ARCH-MOCK | Pass for inspected seams: stateful provisioning/process doubles plus native qualification. Extend ordering coverage for request cancellation. |
   | ARCH-CONSTRAINTS | Pass: worker, output, mailbox, and concurrency bounds are explicit and exercised. |
   | ARCH-SECURE | Pass: empty child environment, private roots, physical confinement, and generation checks are present. |
   | ARCH-ORDER | Flag: caller cancellation has no corresponding fixture request transition. |
   | ARCH-FUNERAL | Flag: abandoned request results and failed ownership-evidence publication need explicit handling. Native successful teardown passed. |
   | ARCH-EVOLVE | Pass: no existing persisted format was changed; inspected native transcript resume preserves prior bytes. |

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying:

   - The isolated-consumer enumeration and short-root migration.
   - Queued/running/completed request cancellation transitions, acknowledgment, joining, and reclamation.
   - Correct PURE/INTEGRATION classifications, entity names, and paint location.
   - Atomic ownership-manifest publication and observable failure behavior.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"8fc913c6-1521-4840-bd9f-dbaa7ee7dd93","repository":"github.com/xianxu/pair","issue_id":"000744","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/029e8c37ac78628c","reviewed_head":"6d584dc279b5745064cabd5a8e59f01da3c6e8d6","slot":"pair:5","requested_at":"2026-10-21T07:36:31.057793Z"},"phase":"finished","pid":1743,"started_at":"2026-10-21T07:36:31.151456Z","finished_at":"2026-10-21T07:41:28.860899Z","session_id":"01a15d63-e9ae-7793-8a4c-0c0d633d2a1f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.56001325s
head: "6d584dc279b5745064cabd5a8e59f01da3c6e8d6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-21T06:51:20-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 744 — testing: one shared disposable-Couch harness for every live test (isolated roots, real supervisor and slots, every agent kind, exact cleanup) |
| repo | pair |
| issue file | workshop/issues/000744-testing-one-shared-disposable-couch-harness-for-every-live-test-isolated-roots-real-supervisor-and-slots-every-agent-kind-exact-cleanup.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7fabbeb93589b7c0c44a43fa699e83b09082bd5e..3a866f60174ee73a004660e363c21f5f05a4d2c1 |
| command | sdlc close --issue 744 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T06:51:20-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared harness passes ordinary tests, race checks, and an independent native Claude/Codex matrix at the pinned head. BR-1, BR-3, and BR-4 are addressed. BR-2 remains blocking: terminal request acknowledgment can occur while a CLI descendant is still running.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The isolated consumers now use owned short roots. All three previously failing tests pass. Restoring the base stale_store_test.go through a temporary Go overlay makes both service regressions fail again with the 103-byte socket error.
  - id: BR-2
    disposition: not-addressed
    note: |
      cmd/internal/testcouch/agentfixture/run.go:459-475 still loses process-group cleanup when the leader exits first. An overlay regression invoked runCLI with a 200ms timeout and a shell that starts sleep 30, records its PID, then exits. runCLI returned a terminal exec.ErrWaitDelay result after 1.01s while the descendant remained alive (kill(pid, 0) succeeded). The cancellation guard refuses group cleanup once the leader identity disappears, and the WaitDelay path performs no reconciliation. Preserve ownership of outstanding descendants and confirm their termination before publishing terminal acknowledgment; add this ordering to the committed regression suite. This remains family request-cancellation-joins-effects (ARCH-ORDER, ARCH-FUNERAL).
  - id: BR-3
    disposition: addressed
    note: |
      The plan's Core concepts tables at lines 25-43 now place Paint in paint.go, use the implemented Command/Result names, and classify ReadResult correlation, retirement, and cancellation as integration. The named pure functions contain no mutable filesystem IO; the dated BR-3 revision explains the correction.
  - id: BR-4
    disposition: addressed
    note: |
      Fleet.record publishes through durablefile.WriteAtomic and retains publication errors without dropping in-memory ownership. The committed snapshot and failure tests pass; overlay reversals of atomic replacement and error retention independently make their corresponding regressions fail. The owned-process cleanup regression also passes.
```

1. **Strengths**

   - Transport tests exercise real listeners, preserve unset-isolation addresses, and verify foreign socket residue remains untouched.
   - Native qualification uses genuine provisioning and registered actors, with independent receipt/body evidence.
   - Ownership publication now preserves complete prior snapshots and exposes persistence failures.
   - README and atlas document the new target, helper interfaces, isolation, and teardown contract.

2. **Critical findings**

   - **BR-2 remains open**, at `cmd/internal/testcouch/agentfixture/run.go:459-475`. Cover leader exit before cancellation and output-pipe completion. Reconcile the owned process group before treating the request as terminal.

3. **Important findings**

   None newly raised.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - Ordinary `testcouch/...`, `testpair`, and `couchmessage` tests.
   - `testcouch/...` race checks.
   - All three previously failing isolated consumers.
   - `make test-affected-inventory`.
   - Independent `make test-couch-fleet`: **73.43s**, both personalities, messaging, detach/restart, and cleanup.
   - Whitespace checks excluding byte-exact terminal captures.

   The temporary leader-exit regression failed against the pinned implementation. Its descendant was explicitly terminated afterward. Repository files remain unchanged.

6. **Architectural notes**

   - **ARCH-DRY — flag:** BR-2 exposes divergence between `runCLI` and the existing `OSProvisionIO` handling of descendants after `ErrWaitDelay`; consolidate lifecycle ownership where practical.
   - **ARCH-PURE — pass:** corrected classifications match the inspected pure functions and IO adapters.
   - **ARCH-PURPOSE — pass:** the narrowed Spec’s shared native fleet and both personalities are delivered.
   - **ARCH-MOCK — pass:** portable stateful fixtures and native qualification exercise the relevant seams.
   - **ARCH-CONSTRAINTS — pass:** worker, mailbox, output, and runtime bounds are explicit and exercised.
   - **ARCH-SECURE — pass:** empty environments, physical confinement, and generation checks protect foreign state.
   - **ARCH-ORDER — flag:** BR-2 collapses leader completion into request completion despite outstanding descendants.
   - **ARCH-FUNERAL — flag:** BR-2 permits terminal retirement before every owned child has ended.
   - **ARCH-EVOLVE — pass:** no cross-version lossy read-modify-write path was introduced; mailbox artifacts are scoped to a disposable run.

7. **Plan revision recommendations**

   Add a `## Revisions` entry extending BR-2’s ordering table to cover **leader exited / descendants retain pipes / deadline or cancellation arrives**. Require confirmed descendant termination, or explicit retained uncertainty, before terminal acknowledgment and retirement.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"2e54f2b9-1890-4ed8-aa76-b90414813608","repository":"github.com/xianxu/pair","issue_id":"000744","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/1ecee9bece0fcf70","reviewed_head":"3a866f60174ee73a004660e363c21f5f05a4d2c1","slot":"pair:5","requested_at":"2026-10-21T13:47:14.108626Z"},"phase":"finished","pid":93929,"started_at":"2026-10-21T13:47:14.211852Z","finished_at":"2026-10-21T13:51:19.505864Z","session_id":"01a15eb7-50d2-7dc3-a339-696cebf6e11e","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.25150225s
head: "3a866f60174ee73a004660e363c21f5f05a4d2c1"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-21T07:30:07-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 744 — testing: one shared disposable-Couch harness for every live test (isolated roots, real supervisor and slots, every agent kind, exact cleanup) |
| repo | pair |
| issue file | workshop/issues/000744-testing-one-shared-disposable-couch-harness-for-every-live-test-isolated-roots-real-supervisor-and-slots-every-agent-kind-exact-cleanup.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7fabbeb93589b7c0c44a43fa699e83b09082bd5e..23a5b119d111b985de0cc18bb5cedb42c1cf838c |
| command | sdlc close --issue 744 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T07:30:07-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared harness, transport confinement, and leader-exit correction are well supported by tests. BR-2 remains blocking: cancellation-publication and acknowledgment failures bypass the sticky uncertainty state, allowing further fleet work while an abandoned child remains alive. Native qualification also failed once before passing unchanged.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Real isolated consumers now use owned short roots. Short-root, shell-wrapper, transport-confinement and targeted couchcmd tests pass.
  - id: BR-2
    disposition: not-addressed
    note: |
      cmd/internal/testcouch/actor.go:97-109 returns on cancellation-publication or acknowledgment failure without recording unknown ownership or returning typed ErrCleanupUncertain. A disposable-checkout regression made cancellations/ read-only after a CLI child started, then canceled its caller: ActorCLI returned a publication error with the child still alive, and a subsequent ActorCLI executed successfully. ARCH-ORDER/ARCH-PURPOSE: apply the request-cancellation-joins-effects rule to every completion-uncertain exit, latch fleet uncertainty before returning, retain correlated evidence for Close, and cover publication failure, acknowledgment failure and cleanup-budget exhaustion.
  - id: BR-3
    disposition: addressed
    note: |
      The Core concepts tables correctly separate deterministic validation/address selection from filesystem-backed publication, correlation, cancellation and retirement. Listed entities exist at their stated paths.
  - id: BR-4
    disposition: addressed
    note: |
      Fleet.record atomically publishes ownership snapshots, retains the first publication error, and preserves in-memory cleanup authority. Snapshot-reader, real publication-failure and exact-owned-process cleanup regressions pass.
findings:
  - id: new
    severity: Important
    kind: defect
    family: native-qualification-failure-classification
    title: |
      Classify the native worker-startup failure before accepting qualification
    detail: |
      cmd/couch/fleet_acceptance_test.go:43 failed on pinned head 23a5b119 during Codex worker 3 startup: “attach terminal input has ended,” failure nonce operation-730874d8fa66046c985836b933151494; the matrix exited 1 after 103.75s. An unchanged-head rerun passed in 107.43s. Preserve and investigate this failure under the plan’s same-head flake rule rather than treating the successful rerun as its disposition; cleanup succeeded on both attempts.
```

1. **Strengths**

   - Messaging isolation changes the listener parent and confines its sweep; real-listener tests verify foreign socket inodes survive.
   - `ActorCLI` executes through the registered fixture’s inherited environment, preserving genuine caller identity.
   - The leader-exit regression covers both inherited and closed pipes. Disabling quiescence classification made both cases fail.
   - README and atlas document the new target, ownership model, isolation, and recovery limits.

2. **Critical findings**

   **BR-2 — [actor.go:97](/Volumes/TestScratch/review-checkouts/0e10ade396eb70bd/cmd/internal/testcouch/actor.go:97).** Publication and acknowledgment errors return without closing admission. The reproduction observed `child_alive=true`, followed by successful output `new-work-executed`.

   Consolidate uncertain request outcomes into one transition that records sticky uncertainty and returns `ErrCleanupUncertain`. Sweep every exit lacking joined terminal proof; fixing only the publication branch would leave the acknowledgment branch exposed.

3. **Important findings**

   Native acceptance needs the failure classification described above. The passing rerun establishes successful execution, but does not explain the preceding startup failure.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed portable helper tests, helper races, couchmessage, testpair, targeted couchcmd tests, fixture composer conformance, artifactpath, repolint, and `make test-affected-inventory`.

   Existing tests miss cancellation-publication failure followed by attempted new work. The reviewer regression demonstrated that failure in a disposable checkout. That checkout was removed; the reviewed checkout remains clean at the pinned head.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared path confinement and existing process/provisioning owners are reused.
   - **ARCH-PURE — pass:** documented pure entities match deterministic implementations.
   - **ARCH-PURPOSE — flag:** BR-2 leaves the cancellation contract incomplete.
   - **ARCH-MOCK — pass:** stateful filesystem/process seams accompany native qualification.
   - **ARCH-CONSTRAINTS — pass:** worker, mailbox, concurrency, output and timeout bounds are explicit and exercised.
   - **ARCH-SECURE — pass:** empty environments, private roots and generation-qualified authority are tested.
   - **ARCH-ORDER — flag:** unacknowledged cancellation bypasses the uncertainty transition.
   - **ARCH-FUNERAL — pass:** normal and explicitly uncertain completion paths retain cleanup ownership; BR-2 needs the same handoff.
   - **ARCH-EVOLVE — pass:** no existing persisted-format compatibility regression identified.

7. **Plan revision recommendations**

   Add a `## Revisions` entry covering cancellation-publication failure, acknowledgment-read failure, and acknowledgment timeout: each retains evidence, fences new mutations, and delegates reconciliation to `Close`. Map each outcome to a regression. Record and classify the native startup failure alongside the unchanged-head passing rerun.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6ddb1318-3360-4d98-b36d-85ab7240fd8c","repository":"github.com/xianxu/pair","issue_id":"000744","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/0e10ade396eb70bd","reviewed_head":"23a5b119d111b985de0cc18bb5cedb42c1cf838c","slot":"pair:5","requested_at":"2026-10-21T14:24:07.857862Z"},"phase":"finished","pid":21173,"started_at":"2026-10-21T14:24:08.055642Z","finished_at":"2026-10-21T14:30:06.84975Z","session_id":"01a15ed9-1912-7043-8dcd-cc74e7537546","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.917762417s
head: "23a5b119d111b985de0cc18bb5cedb42c1cf838c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T08:25:39-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 744 — testing: one shared disposable-Couch harness for every live test (isolated roots, real supervisor and slots, every agent kind, exact cleanup) |
| repo | pair |
| issue file | workshop/issues/000744-testing-one-shared-disposable-couch-harness-for-every-live-test-isolated-roots-real-supervisor-and-slots-every-agent-kind-exact-cleanup.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7fabbeb93589b7c0c44a43fa699e83b09082bd5e..f37acbe8b92fe6c1b186b88874e60392722133df |
| command | sdlc close --issue 744 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T08:25:39-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The shared harness and both prior corrections are substantiated by code and tests. Native qualification passed for Claude and Codex on pinned head, including exact cleanup. One Important issue remains: the provisioning helper duplicates canonical slot-layout rules and fails an existing repository audit.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      ActorCLI now fences every unretired post-publication exit. Current helper and race suites pass; replacing actor.go with its pre-fix version through a temporary Go overlay makes all four acknowledgment/retirement regressions fail. Cancellation, queued suppression, descendant uncertainty and owned cleanup have executable coverage.
  - id: BR-5
    disposition: addressed
    note: |
      Independently inspected Pair's issue-tracker card for #772: it registers the exact Codex subtest with the original nonce and same-head failure/pass evidence. Retained log hashes match. Attach diagnostics have reachable preflight/precommit regressions; this classifies and preserves the failure without claiming its root cause is fixed.
findings:
  - id: new
    severity: Important
    kind: defect
    family: slot-layout-single-source
    title: |
      Provisioning helper bypasses canonical SlotLayout and fails the repository audit
    detail: |
      cmd/internal/testcouch/provision.go:175, :176 and :210 independently construct the worker directory, resting branch and setup-marker path. Both the full couchcore run and isolated TestEverySlotStateSiteIsAResource fail on these three sites. ARCH-DRY / ARCH-PURPOSE: derive them through NewSlotLayout(primary, common, n).Host(), RestingBranch() and SetupMarkerPath(admin), then rerun the audit and provisioning tests. Preserve the audit rather than exempting this consumer.
```

1. **Strengths**

   - Request lifetime has one completion guard at `cmd/internal/testcouch/actor.go:78`, covering early failures as well as uncertain results.
   - Attach evidence preserves bounded output and distinguishes observed exits from unknown outcomes.
   - Isolation tests exercise actual listeners and verify that foreign socket residue remains untouched.
   - README and atlas document the new helper, native target, isolation behavior and cleanup limits.

2. **Critical findings:** None.

3. **Important findings:** The canonical-layout violation above needs correction before landing.

4. **Minor findings:** None.

5. **Test coverage**

   - Native matrix: PASS, 154.44 seconds, both personalities and exact cleanup.
   - Helper ordinary and race suites: PASS.
   - Transport, terminal, wrapper, testpair, artifact and repository-lint packages: PASS.
   - Reader inventory and focused operation-evidence regression: PASS.
   - Full couchcore run: FAIL on the three layout violations; isolated rerun reproduces them.
   - Checkout remains unchanged.

6. **Architecture**

   - **ARCH-DRY — flag:** duplicated slot-layout rules.
   - **ARCH-PURE — pass:** listed pure entities use value-based tests; IO remains in adapters.
   - **ARCH-PURPOSE — flag:** the new layout consumer bypasses the existing authoritative source.
   - **ARCH-MOCK — pass:** stateful filesystem/process fixtures and native qualification cover the seams.
   - **ARCH-CONSTRAINTS — pass:** bounded workers, requests, output and diagnostic budgets have coverage.
   - **ARCH-SECURE — pass:** private environments, physical confinement and diagnostic allowlists are tested.
   - **ARCH-ORDER — pass:** cancellation and uncertain outcomes retain ownership and refuse new work.
   - **ARCH-FUNERAL — pass:** joins, generation-qualified cleanup and retained failure evidence are explicit.
   - **ARCH-EVOLVE — pass:** diagnostic changes reuse existing persisted fields without changing their meaning.

7. **Plan revision recommendation**

   Add a `## Revisions` entry requiring provisioning observations to derive slot paths and branches from `SlotLayout`, with `TestEverySlotStateSiteIsAResource` included in focused verification.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7d6c4789-809e-46de-a897-0ba01cc08cb7","repository":"github.com/xianxu/pair","issue_id":"000744","recipe":"milestone-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/c93bd2053417abab","reviewed_head":"f37acbe8b92fe6c1b186b88874e60392722133df","slot":"pair:5","requested_at":"2026-10-21T15:17:29.859383Z"},"phase":"finished","pid":85756,"started_at":"2026-10-21T15:17:29.959483Z","finished_at":"2026-10-21T15:25:39.185745Z","session_id":"01a15f09-f3e1-7160-b7e8-355ad814c494","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.743222959s
head: "f37acbe8b92fe6c1b186b88874e60392722133df"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T09:03:47-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 744 — testing: one shared disposable-Couch harness for every live test (isolated roots, real supervisor and slots, every agent kind, exact cleanup) |
| repo | pair |
| issue file | workshop/issues/000744-testing-one-shared-disposable-couch-harness-for-every-live-test-isolated-roots-real-supervisor-and-slots-every-agent-kind-exact-cleanup.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7ad6463ef53930f5e3144f23e23d1ef16f6a116d..86e8804dce281ad1d22f89ffc88b7342cbe15dc1 |
| command | sdlc close --issue 744 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T09:03:47-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

BR-6 is addressed. The interdiff replaces duplicated layout rules with canonical APIs, preserves existing paths and behavior, and introduces no blocking defect. The revision records the correction without changing Spec or Done when.

```findings
dispose:
  - id: BR-6
    disposition: addressed
    note: |
      provision.go now derives worker hosts and resting branches through NewSlotLayout and setup markers through SetupMarkerPath. The unchanged repository audit passes on pinned head and fails at all three original sites with pre-fix provision.go restored in a disposable tree. Provisioning tests pass normally and under race detection.
```

1. **Strengths**

   - Provisioning consumes canonical layout at `cmd/internal/testcouch/provision.go:175` and `:211`, preserving validation of actual provisioning evidence.
   - Transcript creation and resume share the canonical runtime at `cmd/internal/testcouch/agentfixture/conversation.go:71`.
   - Both repository audits remain enforced without exemptions.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Slot-state and native-authority audits pass on pinned head.
   - Restoring pre-fix files makes both audits fail for the expected violations; the disposable tree was removed.
   - Provisioning, failure/cancellation/adoption, and Claude/Codex transcript/resume tests pass, including race checks.
   - Inspected retained native qualification evidence and verified its log checksum; native qualification and the full suite were not rerun during this review.
   - Reviewed checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** both corrections reuse existing authorities.
   - **ARCH-PURE — pass:** path derivation remains separate from filesystem/process adapters.
   - **ARCH-PURPOSE — pass:** all three BR-6 sites are corrected, with repository-wide enforcement preserved.
   - **ARCH-EVOLVE — pass:** no persisted schema or field meaning changes.
   - No new user-facing or architectural surface requires README or atlas changes.

7. **Plan revision recommendations:** None; the added revision accurately records the fix.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ae278350-6091-46ed-9348-e3fc007dc0ec","repository":"github.com/xianxu/pair","issue_id":"000744","recipe":"fix-review","round":5,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/6d4fa1b7d36130c0","reviewed_head":"86e8804dce281ad1d22f89ffc88b7342cbe15dc1","slot":"pair:5","requested_at":"2026-10-21T16:00:52.066699Z"},"phase":"finished","pid":17953,"started_at":"2026-10-21T16:00:52.209129Z","finished_at":"2026-10-21T16:02:43.691845Z","session_id":"01a15f31-a98d-7e12-8210-0aee53bfccb6","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.594949834s
head: "86e8804dce281ad1d22f89ffc88b7342cbe15dc1"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
