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
