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
