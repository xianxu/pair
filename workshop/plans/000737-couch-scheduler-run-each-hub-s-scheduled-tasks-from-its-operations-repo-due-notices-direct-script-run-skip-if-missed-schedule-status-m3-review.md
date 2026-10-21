# Boundary Review — pair#737 (milestone M3)

| field | value |
|-------|-------|
| issue | 737 — couch: scheduler: run each hub's scheduled tasks from its operations repo (due notices, direct script-run, skip-if-missed, --schedule --status) |
| repo | pair |
| issue file | workshop/issues/000737-couch-scheduler-run-each-hub-s-scheduled-tasks-from-its-operations-repo-due-notices-direct-script-run-skip-if-missed-schedule-status.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | e8bfd45c1ce022f5427a69871540a7dccf46fde2..b7b6ebb2335e030df26295135431a5c5bd3b9a48 |
| command | sdlc milestone-close --issue 737 --milestone M3 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T14:10:09-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The single-owner design, durable settlement ordering, bounded workers, and manual RPC integration are substantial improvements. All five reviewed package suites passed, as did the focused race checks. However, an independent probe reproduced a reservation that remains busy indefinitely after a catalogue edit. The replacement core-concepts table also names the wrong implementation path for `StateOwner`.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: persisted-state-invariants
    title: |
      Pending catalogue edits strand undispatched reservations indefinitely
    detail: |
      cmd/internal/couchcmd/schedule_owner_dispatch.go:132-135 skips reserved work whenever o.busy is set, while schedule_owner_batch.go:236-243 provides no deadline settlement for those reservations. With both workers occupied, reserve another task and edit its declaration before dispatch: refresh marks it busy, but neither subsequent ticks nor refreshes can settle it, even after its deadline. A temporary test overlay reproduced this in all six scheduled cases: script/notify-only crossed with revision/removal/owner-policy edits. ARCH-ORDER, ARCH-CONSTRAINTS and ARCH-PURPOSE: this is the 2nd finding in family `persisted-state-invariants`. Do NOT fix only this instance; establish the rule that every undispatched reservation has a bounded settlement path independent of dispatch eligibility, and sweep manual, scripted and notify-only reservations across edit, expiry and cancellation sequences. Preserve canonical-evidence precedence and settle known-unstarted work through the pure transition model before applying the edit.
  - id: new
    severity: Critical
    kind: defect
    family: core-concept-traceability
    title: |
      The replacement core-concepts table mislocates StateOwner
    detail: |
      workshop/plans/000737-couch-scheduler-plan.md:634 lists StateOwner in cmd/internal/couchschedule/{store,store_unix}.go, but neither file declares it; the type and its capability methods are implemented in cmd/internal/couchschedule/owner.go:13 onward. This contradicts the required core-concepts cross-check. ARCH-PURPOSE: this is the 2nd finding in family `core-concept-traceability`. Reconcile every active replacement-table row against its production declaration and mark superseded entries explicitly, rather than repairing only this row. Record the corrected implementation map in a Revisions entry.
```

1. **Strengths**

   - `PlanSettlement` adopts confirmed canonical history before processing edits; production tests cover terminal history followed by failed ledger commits.
   - `StateOwner` enforces lifetime ownership, rejects copied/closed handles, and has exclusion and descriptor-inheritance tests.
   - Manual requests preserve intent identity, bound output pages, and avoid automatic resubmission after lost acknowledgements.
   - README, atlas, CLI help, and the embedded Couch skill document the new behavior.

2. **Critical findings**

   - **Stranded reservations:** `schedule_owner_dispatch.go:132` and `schedule_owner_batch.go:236`. Add an owner-driven settlement path for undispatched work that cannot be disabled by a pending edit.
   - **Concept traceability:** plan line 634. Correct and audit the active implementation table. Severity follows the explicitly required table-consistency gate.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed: `couchschedule`, `operationalrun`, `couchcmd`, `couchmessage`, and `artifactpath`.
   - Passed: focused owner/settlement/manual race checks and selected production settlement/growth tests.
   - `git diff --check` passed; checkout remained unchanged.
   - Temporary overlay probes failed in all six reservation/edit combinations described above. Existing tests cover busy edits and reservation recovery separately, but miss their liveness interaction.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass: shared settlement kernel and owner-only writers. |
   | ARCH-PURE | Pass: new pure models have direct tests without IO. |
   | ARCH-PURPOSE | Flag: pending edits can permanently disable future task progress; concept map is inaccurate. |
   | ARCH-MOCK | Pass: shared runtime/broker seams, stateful doubles, and native qualification. |
   | ARCH-CONSTRAINTS | Flag: undispatched busy reservations exceed their declared timeout without settlement. |
   | ARCH-SECURE | Pass: configured-root validation, bounded requests, argv execution, and strict persisted decoding. |
   | ARCH-ORDER | Flag: reserved → pending edit → deadline has no progressing transition. |
   | ARCH-FUNERAL | Pass: existing retention and ownership lifetimes cover the added surface. |
   | ARCH-EVOLVE | Pass: additive fields and nested unknown-field preservation have regression coverage. |

7. **Plan revision recommendations**

   Add `## Revisions` entries defining bounded settlement for every undispatched reservation, with the edit/expiry/cancellation test matrix, and reconciling the active core-concepts table with production source locations.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a5e8c7b1-7912-45bd-ad5e-a2ca9706713d","repository":"github.com/xianxu/pair","issue_id":"000737","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/014ee292d49e2cff","reviewed_head":"b7b6ebb2335e030df26295135431a5c5bd3b9a48","slot":"pair:1","requested_at":"2026-10-21T21:05:14.151845Z"},"phase":"finished","pid":66135,"started_at":"2026-10-21T21:05:14.321707Z","finished_at":"2026-10-21T21:10:08.785125Z","session_id":"01a16048-51fd-7f62-ad04-cc8dff90fd6f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.17854925s
head: "b7b6ebb2335e030df26295135431a5c5bd3b9a48"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-21T14:27:45-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 737 — couch: scheduler: run each hub's scheduled tasks from its operations repo (due notices, direct script-run, skip-if-missed, --schedule --status) |
| repo | pair |
| issue file | workshop/issues/000737-couch-scheduler-run-each-hub-s-scheduled-tasks-from-its-operations-repo-due-notices-direct-script-run-skip-if-missed-schedule-status.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | e8bfd45c1ce022f5427a69871540a7dccf46fde2..e7044d90e3d1cda81cc0b99b137f6680a792a6c8 |
| command | sdlc milestone-close --issue 737 --milestone M3 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T14:27:45-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Both prior findings are addressed, with regression evidence for BR-15 and corrected production references for BR-16. One new blocker remains: an ordinary timer tick clears an unresolved catalogue-discovery error, making stale scheduler state appear healthy. Focused tests pass; an additional temporary-overlay probe reproduces this failure.

```findings
dispose:
  - id: BR-15
    disposition: addressed
    note: |
      schedule_owner_batch.go:236-255 settles expired undispatched reservations independently of busy/policy/capacity. The reservation matrix passes; removing this branch through a temporary overlay makes manual/revision/expiry fail. Removing the companion timeout-recovery correction also fails TestUnstartedTimeoutSummaryCanBeRecovered.
  - id: BR-16
    disposition: addressed
    note: |
      The plan's BR-16 revision at line 770 explicitly supersedes earlier location tables and supplies the complete current map. StateOwner now points to owner.go, where its declaration and capability methods exist; the production-anchor and plan/atlas equality checks pass.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: unresolved-failure-visibility
    title: |
      Ordinary ticks erase unresolved catalogue-discovery errors
    detail: |
      cmd/internal/couchcmd/schedule_owner_loop.go:474-482 ignores refresh's returned error and immediately runs tick; schedule_owner.go:344-352 then publishes a fresh snapshot with an empty Error. Renaming a configured hub away makes refresh report its discovery failure, but the subsequent successful tick clears that error without rediscovery succeeding. A temporary-overlay regression reproduced this on the pinned head. ARCH-ORDER and ARCH-SECURE: retain each failure until evidence from the corresponding operation resolves it; successful ledger reads do not establish successful catalogue discovery. Preserve settlement progress while retaining the discovery error, and add a failure -> tick -> successful rediscovery regression.
```

1. **Strengths**

   - Undispatched reservation settlement now precedes dispatch eligibility checks, including pending catalogue edits.
   - Canonical history takes precedence over competing expiry/cancellation events; tests cover publication and ledger-failure barriers.
   - Writer capabilities, fact-only workers and bounded command handoffs give the replacement a clear ownership boundary.
   - README and atlas document the manual transport, completion output and scheduler implementation map.

2. **Critical findings**

   - Discovery errors disappear without recovery, as detailed above. Store discovery health separately from successful state publication and clear it only after successful discovery.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - Full `couchschedule`, `operationalrun`, `couchmessage` and `artifactpath` package tests.
   - `couchcmd` tests matching `TestSchedule`, uncached.
   - Pinned-range `git diff --check`.

   Both BR-15 mutation probes failed as expected. The new discovery-error probe also failed as described. Temporary overlays were removed; repository files remain unchanged. Live/day-long acceptance was not performed; the plan explicitly permits milestone review before that acceptance.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** shared settlement logic serves commands and batched observations.
   - **ARCH-PURE — pass:** transitions and settlement decisions are directly testable without IO.
   - **ARCH-PURPOSE — pass:** the replacement routes production scheduling and manual execution through one owner.
   - **ARCH-MOCK — pass:** stateful runtime/broker doubles and native fixtures exercise shared seams.
   - **ARCH-CONSTRAINTS — pass:** bounded workers, transport/output limits and counted growth tests support the declared envelope.
   - **ARCH-SECURE — flag:** unresolved external-input failures lose their visible diagnostic state.
   - **ARCH-ORDER — flag:** snapshot publication incorrectly treats an unrelated successful operation as recovery evidence.
   - **ARCH-FUNERAL — pass:** existing retention and joined worker shutdown cover the artifact and process lifecycles examined.
   - **ARCH-EVOLVE — pass:** additive intent fields and nested unknown-field preservation have round-trip coverage.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining failure ownership: catalogue errors survive timer ticks and ledger/status publication until successful rediscovery; independent completion settlement continues meanwhile. Record the corresponding recovery-sequence test.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"96709ecd-e829-4c93-bd0c-2b2c49841dda","repository":"github.com/xianxu/pair","issue_id":"000737","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/a4851ab06959ac89","reviewed_head":"e7044d90e3d1cda81cc0b99b137f6680a792a6c8","slot":"pair:1","requested_at":"2026-10-21T21:24:41.937989Z"},"phase":"finished","pid":84320,"started_at":"2026-10-21T21:24:42.033184Z","finished_at":"2026-10-21T21:27:44.751636Z","session_id":"01a1605a-22bd-7cc1-86a3-bbcb03574089","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.839922417s
head: "e7044d90e3d1cda81cc0b99b137f6680a792a6c8"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
