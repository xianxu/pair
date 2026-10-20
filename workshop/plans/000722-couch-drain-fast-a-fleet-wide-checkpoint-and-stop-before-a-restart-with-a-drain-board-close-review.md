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
