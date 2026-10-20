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
