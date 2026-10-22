# Boundary Review — pair#722 (whole-issue close)

| field | value |
|-------|-------|
| issue | 722 — couch --drain [--fast]: a fleet-wide checkpoint-and-stop before a restart, with a drain board |
| repo | pair |
| issue file | workshop/issues/000722-couch-drain-fast-a-fleet-wide-checkpoint-and-stop-before-a-restart-with-a-drain-board.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7d4e0c3aba6b39fb12d5125ee967dcde9e5be4be..e35f03c8dfbed9e588b532bafd25dc54ab9e3aa8 |
| command | sdlc close --issue 722 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T06:17:36-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The implementation has strong identity fencing, conservative cancellation, bounded generation storage, and substantial sequence coverage. Targeted drain tests and race tests passed. SHIP is blocked by an acknowledgement-ordering defect and missing repeatable real-queue conformance coverage. The plan’s explicit deferral of operator-present acceptance until after review is respected.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: acknowledgement-observation-ordering
    title: |
      Successful acknowledgements can be discarded by activity that preceded them
    detail: |
      cmd/internal/couchcmd/drain_service.go:266-274 reads the current member snapshot but uses only its identity; DrainAcknowledged instead captures the previous reconciliation's Submission at cmd/internal/couchcore/drain.go:255. If operator activity advances the counter before ack, but reconciliation has not consumed it, ack succeeds with the old baseline and the next observation clears it at drain.go:240, despite no activity after acknowledgement. Establish an authoritative activity baseline at acknowledgement and add a deterministic delayed-observation regression while preserving subsequent-activity invalidation (ARCH-ORDER, ARCH-PURPOSE).
  - id: new
    severity: Important
    kind: defect
    family: external-dependency-conformance
    title: |
      Retain a runnable real SDLC queue conformance check and its cadence
    detail: |
      cmd/internal/couchcmd/drain_queue_test.go:13 exercises a generated shell script, while the stateful tests exercise FakeDrainQueue. The issue Log explicitly records removal of the real executable consumer probes; this range contains no retained real-queue conformance test or scheduled check. Preserve an opt-in isolated-root harness through sdlcDrainQueue and document when it runs, covering owner isolation, completeness, interruption and lost-response reconciliation so future dependency drift is detected (ARCH-MOCK).
```

1. **Strengths**

   - Broker admission, reservation revalidation, and batch selection enforce the drain hold; drain notices have a separate control lane.
   - Wrapper generation authority uses two fixed control slots and rechecks fences before input. The 1,000/2,000-cycle tests measure retained outcomes and actual Esc effects.
   - Cancellation preserves uncertain outcomes and releases only the drain’s queue owner.
   - README and atlas updates cover the new commands, states, recovery, and protocol versions.

2. **Critical findings**

   - **Acknowledgement ordering:** [drain_service.go:266](/Volumes/TestScratch/review-checkouts/153ebf1c46781d65/cmd/internal/couchcmd/drain_service.go:266), [drain.go:255](/Volumes/TestScratch/review-checkouts/153ebf1c46781d65/cmd/internal/couchcore/drain.go:255). A sequence of stored counter 10 → pre-ack activity counter 11 → successful ack → observation 11 silently removes the ack. Carry authoritative acknowledgement-time evidence into the transition model; test both pre-ack delayed evidence and genuine post-ack activity.

3. **Important findings**

   - **Real dependency conformance:** [drain_queue_test.go:13](/Volumes/TestScratch/review-checkouts/153ebf1c46781d65/cmd/internal/couchcmd/drain_queue_test.go:13). Retain the isolated consumer harness and an explicit execution cadence. Historical probe results cannot detect later divergence between the fake and SDLC.

4. **Minor findings**

   - None.

5. **Test coverage notes**

   - Passed targeted drain tests across `couchcore`, `couchmessage`, `couchcmd`, `couchtty`, and `wrapcmd`.
   - Passed `-race` drain tests across `couchcmd`, `couchmessage`, and `wrapcmd`.
   - `git diff --check` passed.
   - The service fake returns unchanged activity counters, leaving the acknowledgement ordering above uncovered. No full repository suite or live fleet action was run during this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** centralized notices, board projection, and broker eligibility.
   - **ARCH-PURE — pass:** value-based reducers and injected coordinator boundaries; listed pure entities exist and have direct tests.
   - **ARCH-PURPOSE — flag:** a successful checkpoint acknowledgement must survive evidence of preceding activity.
   - **ARCH-MOCK — flag:** stateful fakes exist, but repeatable real-queue conformance and cadence are missing.
   - **ARCH-CONSTRAINTS — pass:** bounded membership, fan-out, subprocess output, and command duration.
   - **ARCH-SECURE — pass:** strict authority validation and conservative malformed-record handling.
   - **ARCH-ORDER — flag:** acknowledgement-time and observation-time evidence are conflated.
   - **ARCH-FUNERAL — pass:** one bounded intent and fixed current-generation outcomes.
   - **ARCH-EVOLVE — pass:** nested extension preservation, strict known fields, and a documented unsafe-version boundary.

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying acknowledgement-time activity authority and its delayed-observation invariant, plus the retained real-queue conformance harness, isolation requirements, and execution cadence. Keep live acceptance and landing items open as already documented.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ac69fd01-0a4e-4d9c-9c1b-66e0aaaa3e78","repository":"github.com/xianxu/pair","issue_id":"000722","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/153ebf1c46781d65","reviewed_head":"e35f03c8dfbed9e588b532bafd25dc54ab9e3aa8","slot":"pair:5","requested_at":"2026-10-20T13:12:57.477815Z"},"phase":"finished","pid":48497,"started_at":"2026-10-20T13:12:57.672805Z","finished_at":"2026-10-20T13:17:35.348601Z","session_id":"01a15971-93c3-7de2-a7e3-09f0e3336582","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.007093208s
head: "e35f03c8dfbed9e588b532bafd25dc54ab9e3aa8"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T06:35:12-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 722 — couch --drain [--fast]: a fleet-wide checkpoint-and-stop before a restart, with a drain board |
| repo | pair |
| issue file | workshop/issues/000722-couch-drain-fast-a-fleet-wide-checkpoint-and-stop-before-a-restart-with-a-drain-board.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7d4e0c3aba6b39fb12d5125ee967dcde9e5be4be..730bab6aff54eb92132b6d99e0c17c685472e1cf |
| command | sdlc close --issue 722 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T06:35:12-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Both prior findings are addressed, with executable evidence. Drain tests, focused race checks, and the retained real SDLC conformance harness pass. However, that harness confirms a remaining contract mismatch: fast pause leaves merges running, although the issue promises their cancellation and interrupted outcome. The fake models broader cancellation than the real dependency.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Ack samples current activity and fences older observations. Reverting the activity baseline made reducer/service regressions fail; moving the revision capture after Snapshot made both present-member and absent-member ordering regressions fail.
  - id: BR-2
    disposition: addressed
    note: |
      Retained TestPair722RealQueue tests exercise sdlcDrainQueue against isolated real state roots, including owner isolation, completeness, interruption and lost replies. They passed against clean source/build 702b01324fa543038178e23b831d3e3346c1204a; testdata/drainqueue/README.md documents the execution cadence.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: external-dependency-conformance
    title: |
      Fast drain accepts queue behavior that does not cancel merges as specified
    detail: |
      cmd/internal/couchcmd/drain_queue_real_adapter_test.go:164-185 explicitly expects merge/landing work to remain running, merge cancellation_requested=false, and merge retirement interrupted=false; the retained real-queue run confirmed this behavior. This contradicts issue #722 Spec lines 38-39, which promises fast cancellation and interrupted recording for suites and merges. FakeDrainQueue.Pause at drain_queue_fake_test.go:38-40 instead marks every active kind interrupting, masking the difference. This is the 2nd finding in family external-dependency-conformance. State and enforce one cancellation rule across suite, merge and landing in the approved contract, fake, real conformance assertions and coordinator tests; deliver the promised merge behavior or obtain an explicit scope revision before close. ARCH-PURPOSE, ARCH-MOCK.
```

1. **Strengths**

   - Acknowledgement regressions cover registry activity, wrapper input, delayed observations, missing evidence and replacement identities.
   - Typed generation fences protect wrapper input; counted 1,000/2,000-cycle tests verify bounded retained outcomes.
   - Real queue conformance uses private roots and the production adapter/decoder. README and atlas updates cover the new CLI and architecture.

2. **Critical findings**

   - **Fast merge cancellation remains undelivered.** At `cmd/internal/couchcmd/drain_queue_real_adapter_test.go:164`, conformance treats continued merge execution as success. Resolve the cancellation policy across the complete operation-kind enumeration, then make fake and real tests enforce the same approved behavior. The board remains conservative about active work, but that does not fulfill the fast-stop promise.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed `TestDrain` across couchcore, couchcmd, couchmessage, couchtty and wrapcmd.
   - Passed drain race tests across couchcore, couchcmd, couchmessage and wrapcmd.
   - Passed all retained `TestPair722RealQueue` checks against revision `702b01324fa5`.
   - Mutation checks confirmed BR-1 regressions fail without the corrections. Disposable checkout removed; reviewed files remain unchanged.
   - Operator-present acceptance remains open under the documented review-first ordering. A full repository suite was not rerun here.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared notice formatter, board projection and broker eligibility policy.
   - **ARCH-PURE — pass:** reducer, projection and decoder logic have direct value-based tests.
   - **ARCH-PURPOSE — flag:** promised fast merge cancellation is missing.
   - **ARCH-MOCK — flag:** fake and real queue cancellation semantics differ.
   - **ARCH-CONSTRAINTS — pass:** bounded membership, fanout, output and polling; counted lifecycle tests.
   - **ARCH-SECURE — pass:** strict input validation and namespace/incarnation authority checks.
   - **ARCH-ORDER — pass:** tested acknowledgement fencing, delayed callbacks and uncertain cancellation.
   - **ARCH-FUNERAL — pass:** one superseding intent record and bounded current-generation outcomes.
   - **ARCH-EVOLVE — pass:** nested extension preservation and explicit unsafe-version refusal.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining fast cancellation for each of suite, merge and landing, identifying the approving authority for any narrowed scope, and requiring matching fake/real conformance expectations. Retain the separate live-acceptance and landing gates.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f655a9fe-940f-4f73-a1d8-d2ff45e2ada1","repository":"github.com/xianxu/pair","issue_id":"000722","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/41aac0e2a59aad91","reviewed_head":"730bab6aff54eb92132b6d99e0c17c685472e1cf","slot":"pair:5","requested_at":"2026-10-20T13:31:46.305124Z"},"phase":"finished","pid":50111,"started_at":"2026-10-20T13:31:46.499567Z","finished_at":"2026-10-20T13:35:11.672408Z","session_id":"01a15982-cd53-7ac3-8677-0517c73cb912","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.805954625s
head: "730bab6aff54eb92132b6d99e0c17c685472e1cf"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T06:44:42-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 722 — couch --drain [--fast]: a fleet-wide checkpoint-and-stop before a restart, with a drain board |
| repo | pair |
| issue file | workshop/issues/000722-couch-drain-fast-a-fleet-wide-checkpoint-and-stop-before-a-restart-with-a-drain-board.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7d4e0c3aba6b39fb12d5125ee967dcde9e5be4be..b17cc5ce215c1caf6cab8145e60bbaf63513fbb0 |
| command | sdlc close --issue 722 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T06:44:42-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

BR-3 is addressed: the governing revision explicitly permits safe merge/landing retirement, and the fake, coordinator tests, real conformance assertions, README and atlas agree. Focused tests and race checks passed. One Important architectural gap remains: coordinator failure paths bypass the pure transition model. Operator-present live acceptance remains outstanding under the approved gate ordering.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Retained acknowledgement-baseline and delayed present/absent observation regressions pass, including under race; production observations use the revision-fenced observeMember path.
  - id: BR-2
    disposition: addressed
    note: |
      Retained real conformance and documented cadence remain present; reviewer reran all three conformance tests successfully against clean matching SDLC source/build dc76211e27578aca7aa62d434db9ca2bd43eed5b.
  - id: BR-3
    disposition: addressed
    note: |
      Governing plan lines 292–302 explicitly revise fast cancellation policy. Both new regression tests fail with the pre-fix fake and pass on head; real conformance confirms suite interruption and continued merge/landing activity.
findings:
  - id: new
    severity: Important
    kind: defect
    family: state-transition-authority
    title: |
      Coordinator failure paths bypass the authoritative drain transition model
    detail: |
      cmd/internal/couchcmd/drain_service.go:819–820 directly changes cancelled back to cancelling after broker release fails, although ReduceDrain rejects every event in cancelled state. Additional direct Error assignments occur at lines 171, 255 and 552. Consequently the reducer does not enumerate all production transitions, contrary to ARCH-ORDER structural enforcement. Model broker release outcomes and other failure outcomes as explicit events, route these writes through the pure model, and enforce that the coordinator cannot directly mutate authoritative phase/error fields.
```

1. **Strengths**

   - Completion requires checkpoint acknowledgement, subsequent idle evidence and authoritative queue quiescence.
   - BR-3 tests cover suite, merge, landing and combined operation kinds; the coordinator continues waiting through separate retirements.
   - Typed generation fences protect final input effects, with counted 1,000/2,000-cycle lifetime tests.
   - README and atlas document the new commands, recovery behavior and revised fast policy.

2. **Critical findings**

   None.

3. **Important findings**

   The `state-transition-authority` finding above. Add a broker-release outcome event and test failed release followed by retry through the production reducer, including rejection of stale outcomes.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Five-package focused drain/recovery tests: PASS.
   - Four-package drain race tests: PASS.
   - All retained real queue conformance tests: PASS, using fresh private roots.
   - Mutation control: restoring the old fake makes both BR-3 regressions fail. Disposable checkout removed; tracked checkout unchanged.
   - Full repository suite and operator-present live acceptance were not rerun in this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared notice, board and broker eligibility policies.
   - **ARCH-PURE — pass:** listed pure entities use value-based tests without IO.
   - **ARCH-PURPOSE — pass:** revised fast policy covers every queue kind.
   - **ARCH-MOCK — pass:** stateful doubles share production seams; retained real conformance passes.
   - **ARCH-CONSTRAINTS — pass:** bounded membership, fanout, subprocess output and polling.
   - **ARCH-SECURE — pass:** strict decoding and exact incarnation/generation checks.
   - **ARCH-ORDER — flag:** authoritative state mutations bypass the reducer.
   - **ARCH-FUNERAL — pass:** bounded current record and generation outcomes.
   - **ARCH-EVOLVE — pass:** nested unknown-field preservation and strict known-field validation.

7. **Plan revision recommendations**

   Add a `## Revisions` entry specifying broker-release success/failure/retry transitions, their completion invariant and the enforcement preventing coordinator-side phase/error mutation.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7a623b66-2d6a-4da3-be41-e14b781184ba","repository":"github.com/xianxu/pair","issue_id":"000722","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/0e292e1f7a0ac1e3","reviewed_head":"b17cc5ce215c1caf6cab8145e60bbaf63513fbb0","slot":"pair:5","requested_at":"2026-10-20T13:41:19.936664Z"},"phase":"finished","pid":57818,"started_at":"2026-10-20T13:41:20.132552Z","finished_at":"2026-10-20T13:44:41.991386Z","session_id":"01a1598b-8df2-73b0-bff3-2a4ff9ee3e17","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.971019959s
head: "b17cc5ce215c1caf6cab8145e60bbaf63513fbb0"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T07:05:20-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 722 — couch --drain [--fast]: a fleet-wide checkpoint-and-stop before a restart, with a drain board |
| repo | pair |
| issue file | workshop/issues/000722-couch-drain-fast-a-fleet-wide-checkpoint-and-stop-before-a-restart-with-a-drain-board.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7d4e0c3aba6b39fb12d5125ee967dcde9e5be4be..31899be8ee9f96a6869ded680014c974b38fb64f |
| command | sdlc close --issue 722 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T07:05:20-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

BR-4 is addressed: production failure paths now use the reducer, regression tests fail when the fixes are removed, and the structural lint passes. No additional behavioral defect was found. However, the pinned head fails existing physical-write checks in two affected packages; both packages pass at the pinned base, leaving validation unresolved.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      drain_service.go routes failure/release outcomes through ReduceDrain. Reducer and service regressions fail with the previous implementations restored in disposable checkouts. Focused race tests, lint fixtures, and the repository catalog scan pass.
findings:
  - id: new
    severity: Important
    kind: unverifiable
    family: test-write-envelope
    title: |
      Pinned-head package validation exceeds enforced physical-write budgets
    detail: |
      ARCH-CONSTRAINTS: go test ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchmessage ./cmd/internal/couchtty ./cmd/internal/wrapcmd ./internal/repolint fails after behavioral assertions pass: couchcore writes 139386880 bytes against 134217728, and couchtty writes 87937024 against 67108864. Enforcement is at cmd/internal/couchcore/launchhelper_test.go:50 and cmd/internal/couchtty/reattach_testmain_test.go:21. Both packages pass in a Git-backed disposable clone of the pinned base. Provide a measured green pinned-head run with an explanation of the discrepancy, or diagnose and correct the excess before crossing the boundary.
```

1. **Strengths**
   - Broker-release failure invalidates stale queue evidence and requires fresh reconciliation (`couchcore/drain.go:174`).
   - Release handling checks the current generation before applying outcomes (`couchcmd/drain_service.go:810`).
   - Tests cover acknowledgement ordering, retained outcomes, transport, and bounded generation retirement.
   - README and atlas document the CLI, recovery behavior, and revised fast-mode policy.

2. **Critical findings:** None.

3. **Important findings:** The write-budget validation blocker above. Its cause has not been attributed to a particular implementation change.

4. **Minor findings:** None.

5. **Test coverage notes**
   - All drain-targeted tests pass across core, messaging, coordinator, and wrapper packages.
   - Console drain tests and focused race tests pass.
   - BR-4 reducer and coordinator mutation controls fail as expected; disposable checkouts were removed.
   - Full couchcmd, couchmessage, wrapcmd, and repolint checks pass.
   - The initial archive-only baseline was inconclusive; the subsequent Git-backed baseline passes both affected packages.

6. **Architectural notes**
   - **ARCH-DRY — pass:** shared notice formatter and dispatch-selection policy.
   - **ARCH-PURE — pass:** listed pure entities have direct tests without IO; integration dependencies are injected.
   - **ARCH-PURPOSE — pass:** checkpoint, acknowledgement, quiescence, and cancellation contracts are implemented.
   - **ARCH-MOCK — pass:** stateful dependency fixtures and retained real-conformance runner exist.
   - **ARCH-CONSTRAINTS — flag:** package write-envelope validation remains unresolved.
   - **ARCH-SECURE — pass:** strict parsing and generation/incarnation fencing fail closed.
   - **ARCH-ORDER — pass:** BR-4 restores reducer authority with regression and lint enforcement.
   - **ARCH-FUNERAL — pass:** bounded current records and control outcomes have explicit retirement.
   - **ARCH-EVOLVE — pass:** persisted extension preservation and strict known-field validation are tested.

7. **Plan revision recommendations:** No design revision required. Record the answering write-budget measurements when resolving the validation finding. Existing live-acceptance and landing holds remain applicable.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"dadb6fbb-7a46-4bc8-9cc3-1d74b697b5fa","repository":"github.com/xianxu/pair","issue_id":"000722","recipe":"milestone-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/c2ccd605232809fa","reviewed_head":"31899be8ee9f96a6869ded680014c974b38fb64f","slot":"pair:5","requested_at":"2026-10-20T13:53:16.820326Z"},"phase":"finished","pid":74321,"started_at":"2026-10-20T13:53:16.91428Z","finished_at":"2026-10-20T14:05:19.65961Z","session_id":"01a15996-7d64-7e11-9124-74e107f7d3b3","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.583554292s
head: "31899be8ee9f96a6869ded680014c974b38fb64f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T07:17:14-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 722 — couch --drain [--fast]: a fleet-wide checkpoint-and-stop before a restart, with a drain board |
| repo | pair |
| issue file | workshop/issues/000722-couch-drain-fast-a-fleet-wide-checkpoint-and-stop-before-a-restart-with-a-drain-board.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7d4e0c3aba6b39fb12d5125ee967dcde9e5be4be..09140407b75d866a912c26cb32e435de2451e4ee |
| command | sdlc close --issue 722 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T07:17:14-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

No new blocking defect found. BR-5’s answering evidence is verified, and fresh pinned-head core/TTY validation passes with unchanged write caps enforced. This review does not waive the recorded full suite or operator-present live acceptance required before merge by the approved review-first plan.

```findings
dispose:
  - id: BR-5
    disposition: withdrawn
    note: |
      Verified the retained JSON log against SHA256 e69040f5cc38dd750769b510d24b3de942dbd2ec9b70409531c34c9af82af45f and its head/environment/argv metadata. Both complete packages passed with unchanged caps enforced; changes since measured head 31899be8 are tracker/review records only. A fresh pinned-head run also passed, with couchcore writing 96641024/134217728 bytes. The original differing totals remain unexplained, but the claimed inability to validate is resolved.
```

1. **Strengths**

   - Completion requires matching acknowledgement, subsequent idle evidence, and authoritative queue quiescence.
   - Generation retirement requires confirmed revocation; stale callbacks retain no input authority (`cmd/internal/couchmessage/drain_generation.go:239`).
   - Broker release outcomes are generation-checked and reducer-owned (`cmd/internal/couchcmd/drain_service.go:810`).
   - README and atlas cover the commands, revised fast policy, timeout, and recovery behavior.

2. **Critical findings:** None.

3. **Important findings:** None remaining.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Full uncached couchcore and couchtty packages passed.
   - Focused drain/queue tests passed across coordinator, messaging, wrapper, and lint packages.
   - Drain race tests passed across couchcmd, couchmessage, and wrapcmd.
   - Full repolint and artifactpath checks passed; `git diff --check` passed.
   - Real queue conformance was inspected but not rerun in this review. The recorded full suite and live acceptance remain outstanding.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass — shared notice formatter, board projection, and control eligibility helpers. |
   | ARCH-PURE | Pass — pure transitions/decoders; declared PURE tests use values without IO. |
   | ARCH-PURPOSE | Pass — both modes, cancellation, timeout, and board delivered under documented revisions. |
   | ARCH-MOCK | Pass — stateful doubles share production seams; retained real conformance runner has an explicit cadence. |
   | ARCH-CONSTRAINTS | Pass — bounded membership, fan-out, subprocesses, output, and counted generation residue. |
   | ARCH-SECURE | Pass — authenticated callers, exact incarnation fences, strict persisted-input validation. |
   | ARCH-ORDER | Pass — explicit transitions and controllable ordering tests preserve uncertain outcomes. |
   | ARCH-FUNERAL | Pass — one replacement record, bounded current outcomes, joined coordinator work. |
   | ARCH-EVOLVE | Pass — nested extensions preserved; unsafe schema downgrade is explicitly refused. |

7. **Plan revision recommendations:** None. Keep existing live-acceptance and landing checkboxes open until their evidence is recorded.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"92c61180-d287-49f0-8593-8a16b30206ab","repository":"github.com/xianxu/pair","issue_id":"000722","recipe":"milestone-review","round":5,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/9dba7c9029955f31","reviewed_head":"09140407b75d866a912c26cb32e435de2451e4ee","slot":"pair:5","requested_at":"2026-10-20T14:13:30.028015Z"},"phase":"finished","pid":12602,"started_at":"2026-10-20T14:13:30.138693Z","finished_at":"2026-10-20T14:17:13.602063Z","session_id":"01a159a9-00b8-7a12-9ef7-e3f55e20bc19","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.138481125s
head: "09140407b75d866a912c26cb32e435de2451e4ee"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T01:40:19-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 722 — couch --drain [--fast]: a fleet-wide checkpoint-and-stop before a restart, with a drain board |
| repo | pair |
| issue file | workshop/issues/000722-couch-drain-fast-a-fleet-wide-checkpoint-and-stop-before-a-restart-with-a-drain-board.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0b605b1ee4ec9bda5a4d569708d25e0a857efd2f..903a134c3603877e6eedc662effa1b56448c5117 |
| command | sdlc close --issue 722 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T01:40:19-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the approved transient-flags Spec and superseding Plan revisions. No blocking defects found. README and atlas cover the new surface. The documented rule rollout and watched live acceptance remain required before landing.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The approved redesign removes machine-parsed acknowledgements and readiness authority; ordinary replies no longer affect drain flags.
  - id: BR-2
    disposition: addressed
    note: |
      No drain queue adapter or automatic SDLC operation remains; the former external dependency and its conformance requirement were retired by the approved redesign.
  - id: BR-3
    disposition: addressed
    note: |
      Current fast drain sends ordinary guidance and performs no queue cancellation; source and current documentation agree.
  - id: BR-4
    disposition: addressed
    note: |
      The coordinator and its phase/error mutation paths are absent. Current flag changes use the pure set helpers under the service mutex.
  - id: BR-5
    disposition: withdrawn
    note: |
      Withdrawal remains supported: this review completed uncached full couchcore and couchtty package runs successfully without changing write budgets.
```

1. **Strengths**

   - Steward authorization, issuing-steward exclusion, and membership limits precede flag publication and notice admission (`cmd/internal/couchcmd/drain_wiring.go:63`).
   - Undrain clears flags before sending notices, preserving the specified behavior when recipients disconnect or delivery fails (`cmd/internal/couchcmd/drain_wiring.go:81`).
   - Idle projection requires positive quiescence evidence and preserves unknown values (`cmd/internal/couchcore/drain_board.go:40`).
   - Native acceptance combines public receipts with independently accepted message bodies, actual wrapper registration, restart checks, and exact cleanup.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Fresh focused drain tests passed across core, service, protocol, console, and CLI packages.
   - Fresh complete runs passed for `couchcore`, `couchmessage`, `couchcmd`, `couchtty`, and `wrapcmd`.
   - Both retained native-acceptance and recorded-suite log hashes match their manifests. Git confirms subsequent changes contain only tracker, plan, and evidence updates.
   - Native acceptance was inspected through retained evidence, not rerun during this review. The older NoticeStarted discrepancy remains explicitly unresolved rather than credited as fixed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Reuses ordinary mailbox admission, receipts, authorization, and console publication.
   - **ARCH-PURE — pass:** Set operations, projection, and formatting are deterministic; IO stays in service wiring.
   - **ARCH-PURPOSE — pass:** Delivers the approved flag/notice/status/lift contract without restoring completion authority.
   - **ARCH-MOCK — pass:** Reuses existing delivery seams and stateful fakes; native acceptance qualifies actual composition.
   - **ARCH-CONSTRAINTS — pass:** Flags are capped, admission has a shared five-second budget, and status performs no new sampling.
   - **ARCH-SECURE — pass:** Mutations require verified steward identity; request shapes and continuation text are validated.
   - **ARCH-ORDER — pass:** Flag updates and publication are mutex-ordered; notice outcomes remain independent, including documented delayed-message limitations.
   - **ARCH-FUNERAL — pass:** Flags end with service lifetime; receipts retain their existing lifecycle.
   - **ARCH-EVOLVE — pass:** No drain persistence format is introduced; legacy bytes remain untouched.

7. **Plan revision recommendations:** None. The later revisions explicitly supersede the historical coordinator design and preserve the remaining live-acceptance gates.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"3a753312-157e-433d-b234-95325e1e31fb","repository":"github.com/xianxu/pair","issue_id":"000722","recipe":"milestone-review","round":6,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/0fabf8a0da2c91d9","reviewed_head":"903a134c3603877e6eedc662effa1b56448c5117","slot":"pair:5","requested_at":"2026-10-22T08:31:59.268119Z"},"phase":"finished","pid":64226,"started_at":"2026-10-22T08:31:59.48569Z","finished_at":"2026-10-22T08:40:18.993176Z","session_id":"01a162bd-0f99-7040-85fe-ca221bdafb12","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.090192458s
head: "903a134c3603877e6eedc662effa1b56448c5117"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T03:20:43-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 722 — couch --drain [--fast]: a fleet-wide checkpoint-and-stop before a restart, with a drain board |
| repo | pair |
| issue file | workshop/issues/000722-couch-drain-fast-a-fleet-wide-checkpoint-and-stop-before-a-restart-with-a-drain-board.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0b605b1ee4ec9bda5a4d569708d25e0a857efd2f..221af63022ff5fcd7247e04830d6b7c9853ec0f8 |
| command | sdlc close --issue 722 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T03:20:43-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the approved transient-flags Spec and superseding Plan revisions. No blocking defects found. README and atlas document the new commands and their limits. The explicitly retained operator-present acceptance gate remains required before landing; no complete current-source native pass is claimed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The approved redesign removes parsed acknowledgements and readiness authority. Ordinary replies cannot change drain flags; message-independence regressions pass.
  - id: BR-2
    disposition: addressed
    note: |
      The superseding Plan removes automatic queue integration. Current drain wiring uses ordinary Broker.SendSystem delivery and makes no SDLC calls.
  - id: BR-3
    disposition: addressed
    note: |
      Fast drain now supplies worker guidance only. The formatter, README and atlas agree that Couch performs no automatic queue cancellation or interruption.
  - id: BR-4
    disposition: addressed
    note: |
      The former coordinator is absent. Production flag updates use AddDrainFlags and ClearDrainFlags under drainMu; service and race regressions pass.
  - id: BR-5
    disposition: withdrawn
    note: |
      Withdrawal remains supported: fresh uncached runs of all five affected internal packages passed without changing physical-write budgets.
```

1. **Strengths**

   - Steward authentication, issuer exclusion and cohort bounds precede publication and delivery (`cmd/internal/couchcmd/drain_wiring.go:59`).
   - Undrain clears flags before notice attempts, preserving the contract when delivery fails (`cmd/internal/couchcmd/drain_wiring.go:81`).
   - Status preserves unknown observations and requires positive quiescence evidence before displaying idle age (`cmd/internal/couchcore/drain_board.go:28`).
   - Native acceptance checks submitted receipts against independently captured worker input (`cmd/couch/drain_acceptance_test.go:148`).

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Fresh uncached tests passed for `couchcore`, `couchmessage`, `couchcmd`, `couchtty` and `wrapcmd`.
   - Focused drain tests, including CLI and acceptance-oracle tests, passed across six packages.
   - Focused race checks passed for service, broker snapshots and console publication.
   - `make test-affected-inventory` and patch whitespace checks passed.
   - In a disposable copy, the graceful-notice regression passed with current wording and failed after restoring the ambiguous wording.
   - Retained suite/native log hashes matched their manifests. The recorded suite covers unchanged executable source. Native reruns were not performed, consistent with the final approved verification sequence.

6. **Architectural notes**

   - **ARCH-DRY — pass:** reuses ordinary system delivery, receipts and cached activity.
   - **ARCH-PURE — pass:** flag operations, notices and board projection remain deterministic; IO stays in service wiring.
   - **ARCH-PURPOSE — pass:** delivers steward-requested posture without introducing machine readiness authority.
   - **ARCH-MOCK — pass:** retains existing injected messaging seams and shared disposable native harness; adds no queue dependency.
   - **ARCH-CONSTRAINTS — pass:** actor limits, total admission deadline and 8/16-member operation-count tests bound the new work.
   - **ARCH-SECURE — pass:** closed request validation, verified steward identity and quoted, validated continuation notes protect the boundary.
   - **ARCH-ORDER — pass:** serialized flag updates precede delivery; delivery outcomes never implicitly alter posture.
   - **ARCH-FUNERAL — pass:** flags die with the service; no new runtime persistence or background coordinator is introduced.
   - **ARCH-EVOLVE — pass:** no persisted format changes; legacy drain bytes remain untouched.

7. **Plan revision recommendations:** None. The later revisions explicitly supersede the historical coordinator design and identify the remaining watched-acceptance requirement.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a7b903bb-6fac-4a31-ace6-412aba4e4568","repository":"github.com/xianxu/pair","issue_id":"000722","recipe":"milestone-review","round":7,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/1462a9a8be46b80c","reviewed_head":"221af63022ff5fcd7247e04830d6b7c9853ec0f8","slot":"pair:5","requested_at":"2026-10-22T10:14:47.633248Z"},"phase":"finished","pid":23135,"started_at":"2026-10-22T10:14:47.735277Z","finished_at":"2026-10-22T10:20:43.039609Z","session_id":"01a1631b-2dd8-77e3-bc8e-bec09253fae4","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.955565958s
head: "221af63022ff5fcd7247e04830d6b7c9853ec0f8"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T12:01:42-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 722 — couch --drain [--fast]: a fleet-wide checkpoint-and-stop before a restart, with a drain board |
| repo | pair |
| issue file | workshop/issues/000722-couch-drain-fast-a-fleet-wide-checkpoint-and-stop-before-a-restart-with-a-drain-board.md |
| boundary | whole-issue close |
| milestone | — |
| window | fea66e36b0d25b3e37f2dc70a084226a8ed1697a..a78eb50c243892e06ddde91147726edb0f0a75c4 |
| command | sdlc close --issue 722 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T12:01:42-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The asynchronous implementation matches the latest approved scope, and focused behavioral and race checks passed. SHIP is blocked by incomplete final-source verification: the broad package run timed out in `couchcore`, and the retained recorded suite predates this implementation. No drain correctness defect was established.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Retain disposition: the superseding contract removes formal acknowledgements. Current grammar rejects --ack; ordinary acknowledgement text supplies no readiness authority.
  - id: BR-2
    disposition: addressed
    note: |
      Retain disposition: the approved transient redesign removes the SDLC queue adapter and its dependency. Production drain code performs no queue operations.
  - id: BR-3
    disposition: addressed
    note: |
      Retain disposition: current fast drain is worker guidance through ordinary notices. The formatter, tests, README and atlas consistently describe supported worker-owned stopping rather than automatic merge cancellation.
  - id: BR-4
    disposition: addressed
    note: |
      Retain disposition: the obsolete coordinator is absent. Current parent phases use ApplyReceiptEvent and child transitions use ApplyDrainChildren, exercised by socket and ordering regressions.
  - id: BR-5
    disposition: withdrawn
    note: |
      Retain withdrawal: this review observed no physical-write budget violation. The separate package timeout below does not establish that previously withdrawn claim.
findings:
  - id: new
    severity: Important
    kind: unverifiable
    family: boundary-verification-evidence
    title: |
      Final-source package verification remains unconfirmed
    detail: |
      workshop/plans/000722-fleet-drain-plan.md:597 requires a recorded declared affected/full-suite result before re-close, while the pinned issue at line 380 still records that check as pending. The attempted command go test ./cmd/internal/couchcore ./cmd/internal/couchmessage ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/wrapcmd exited 1: couchcore panicked at its existing 10-minute timeout (600.316s); the other four packages passed. No causal attribution to this diff is established. Supply a successful recorded declared check covering a78eb50c's executable sources, with complete output, environment, duration and log hash; investigate the timeout if it reproduces. Historical ec88486f evidence predates the new executor and operation model.
  - id: new
    severity: Minor
    kind: defect
    family: documentation-structure
    title: |
      Embedded skill prose breaks the command table
    detail: |
      cmd/internal/couchcmd/skills/couch/SKILL.md:22 inserts a multiline operation explanation into a table cell, followed by a blank line and an orphaned command at line 32. Restore the status command as one complete row and move the explanatory paragraph below the complete table.
```

1. **Strengths**

   - Parent and child identities are registered before flag publication; capacity refusal precedes posture changes.
   - Controlled socket tests verify healthy-target progress while one target blocks, lost-reply recovery, caller isolation and ordered drain/lift notices.
   - Four service-owned admission workers have independent deadlines and cancellation/join coverage.
   - README and atlas document the new surface and distinguish notice delivery from checkpoint completion.

2. **Critical findings**

   None established.

3. **Important findings**

   Final-source verification remains unconfirmed, as detailed above. Resolve with recorded evidence before crossing the boundary.

4. **Minor findings**

   Repair the embedded skill table at `cmd/internal/couchcmd/skills/couch/SKILL.md:22`.

5. **Test coverage notes**

   - Passed focused drain/activity tests across core, messaging, CLI and console packages.
   - Passed focused executor/operation/flag race checks.
   - Passed affected-reader inventory, repository lint, artifact checks and acceptance-oracle tests.
   - Complete messaging, CLI, console and wrapper packages passed; complete core package timed out.
   - Historical native/suite log hashes matched metadata. No current native matrix rerun was attempted, respecting the latest plan.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared receipt reservation and ordinary broker delivery.
   - **ARCH-PURE — pass:** current concept-map entities exist; reducers and projections have direct value-based tests.
   - **ARCH-PURPOSE — pass:** implementation delivers transient posture and independent notice admission.
   - **ARCH-MOCK — pass:** controlled endpoint seams exercise production transport and scheduling.
   - **ARCH-CONSTRAINTS — pass:** worker count, pending capacity, child count and frame sizes are bounded.
   - **ARCH-SECURE — pass:** authenticated mutations, strict request shapes and explicit uncertain outcomes.
   - **ARCH-ORDER — pass:** reducer-owned child transitions and controlled ordering tests.
   - **ARCH-FUNERAL — pass:** transient state, bounded receipt retention and joined workers.
   - **ARCH-EVOLVE — pass:** no new persisted drain format; legacy drain bytes remain untouched.

7. **Plan revision recommendations**

   No design revision needed. Complete the latest verification checklist with exact-source evidence; retain authorized staging and watched acceptance as subsequent gates.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"724b5222-c276-4926-a27a-bd02577a93e2","repository":"github.com/xianxu/pair","issue_id":"000722","recipe":"milestone-review","round":8,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/504d8b887770143b","reviewed_head":"a78eb50c243892e06ddde91147726edb0f0a75c4","slot":"pair:5","requested_at":"2026-10-22T18:49:34.698925Z"},"phase":"finished","pid":13618,"started_at":"2026-10-22T18:49:34.820173Z","finished_at":"2026-10-22T19:01:39.294369Z","session_id":"01a164f2-7abd-7a53-9ca0-a243116b3474","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.209785083s
head: "a78eb50c243892e06ddde91147726edb0f0a75c4"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T15:50:35-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 722 — couch --drain [--fast]: a fleet-wide checkpoint-and-stop before a restart, with a drain board |
| repo | pair |
| issue file | workshop/issues/000722-couch-drain-fast-a-fleet-wide-checkpoint-and-stop-before-a-restart-with-a-drain-board.md |
| boundary | whole-issue close |
| milestone | — |
| window | d6cb2ca88ae20435ebff7f48628dcd8f3ca8b1ad..b2c541b239de1db1c2feace42e731b16d8cd1069 |
| command | sdlc close --issue 722 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T15:50:35-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the approved transient-drain contract and registered asynchronous notice design. Both open findings are addressed; no new blocking defects were found. This clears the code-review boundary. The plan’s operator-watched acceptance remains required before landing.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Remains disposed. The approved replacement removes formal acknowledgements and completion authority; ordinary replies do not mutate drain state.
  - id: BR-2
    disposition: addressed
    note: |
      Remains disposed under the approved replacement: Couch no longer integrates with or controls the SDLC queue.
  - id: BR-3
    disposition: addressed
    note: |
      Remains disposed. Fast drain now sends worker guidance through ordinary authenticated messaging; Couch performs no queue cancellation.
  - id: BR-4
    disposition: addressed
    note: |
      Remains disposed. The obsolete coordinator is absent; current child outcomes pass through ApplyDrainChildren and parent lifecycle transitions through ApplyReceiptEvent.
  - id: BR-5
    disposition: withdrawn
    note: |
      Prior withdrawal retained; no new physical-write-budget finding is raised.
  - id: BR-6
    disposition: addressed
    note: |
      Verified the complete declared make test log and metadata for e3c7e6badda23964b0116310fff8cdb8c94291b1: exit 0, 715.607 seconds, managed Mac.localdomain environment, SHA256 a75339213f2f019545eb95405abb4980d455df88993bd173e27c2cd6e72cc44b matching the actual log bytes. All five named packages passed, including couchcore in 163.798 seconds. Git comparison confirms only tracker/evidence changes between that source and the pinned head. The earlier timeout did not recur; its cause remains unattributed.
  - id: BR-7
    disposition: addressed
    note: |
      Inspected the before/after repair: SKILL.md:22 now contains the complete status-command row, the command table continues through line 45, and the operation explanation starts below it at line 47. The explanation agrees with the registered-operation implementation. TestSkillCommandIndexIsOneCompleteTable passes.
```

1. **Strengths**

   - Registration precedes flag publication and endpoint work, preserving parent/child correlation after a lost reply (`drain_operations.go:36`, `drain_wiring.go:98`).
   - Four service-owned workers enforce per-target ordering, independent admission budgets, bounded capacity and joined shutdown (`drain_executor.go:31`).
   - Status preserves unknown activity evidence and distinguishes notice delivery from checkpoint completion (`drain_board.go:29`).
   - README, atlas and embedded skill explain the new commands, transient lifetime and operation-status semantics.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed focused drain/undrain/skill tests across couchcore, couchmessage, couchcmd and couchtty; focused race tests across couchcmd and couchmessage; acceptance-oracle/catalog tests; and `make test-affected-inventory`.

   Inspected controlled tests for slow-target progress, lost replies, replay conflicts, capacity refusal before mutation, stale completions, per-target FIFO and cancellation/join. Verified the recorded full-suite evidence rather than rerunning that suite. No fresh native fleet or watched acceptance was performed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Reuses ordinary broker delivery, receipt reservation and lifecycle reduction.
   - **ARCH-PURE — pass:** Flag operations, board projection, notice formatting and child transitions have direct tests without IO.
   - **ARCH-PURPOSE — pass:** Delivers the approved posture/notice/status/lift contract.
   - **ARCH-MOCK — pass:** Production and controlled endpoint tests share the broker seam; retained native evidence has explicit limitations.
   - **ARCH-CONSTRAINTS — pass:** Actor/operation/task caps, four workers and per-dispatch deadlines are enforced.
   - **ARCH-SECURE — pass:** Mutations authenticate the steward; request shapes and optional text are validated.
   - **ARCH-ORDER — pass:** Explicit reducers govern outcomes; controlled tests exercise delayed and reordered work.
   - **ARCH-FUNERAL — pass:** Workers are cancelled and joined; bounded transient records expire or disappear with the supervisor.
   - **ARCH-EVOLVE — pass:** This implementation introduces no persisted drain format or rewrite path.

7. **Plan revision recommendations:** None required. Retain the documented pre-landing acceptance gates.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"8f214fb1-3d56-4d2b-b958-fc540ab28532","repository":"github.com/xianxu/pair","issue_id":"000722","recipe":"milestone-review","round":9,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/ff8fde5c36f248d5","reviewed_head":"b2c541b239de1db1c2feace42e731b16d8cd1069","slot":"pair:5","requested_at":"2026-10-22T22:48:26.168144Z"},"phase":"finished","pid":2534,"started_at":"2026-10-22T22:48:26.347827Z","finished_at":"2026-10-22T22:50:34.061621Z","session_id":"01a165cd-2990-7c61-91a6-2a7886ca0738","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.041949041s
head: "b2c541b239de1db1c2feace42e731b16d8cd1069"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
