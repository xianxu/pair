# Boundary Review — pair#737 (milestone M2)

| field | value |
|-------|-------|
| issue | 737 — couch: scheduler: run each hub's scheduled tasks from its operations repo (due notices, direct script-run, skip-if-missed, --schedule --status) |
| repo | pair |
| issue file | workshop/issues/000737-couch-scheduler-run-each-hub-s-scheduled-tasks-from-its-operations-repo-due-notices-direct-script-run-skip-if-missed-schedule-status.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 369451d79fe2072aa39ee8b3928bfbf9e6f40132..11db21abcaa8e29ad26e7914f8c6ecf86aec6d68 |
| command | sdlc milestone-close --issue 737 --milestone M2 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T00:07:19-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The pinned range delivers automatic dispatch, shared execution, durable occurrence publication, and authenticated notices, with README and atlas updates. Existing tests pass, but four additional regression probes expose three blocking defects: notice admission survives expired or cancelled authority, one damaged task history blocks all scheduler workers, and recovery loses additive summary metadata.

1. **Strengths**

   - Due reservation commits before history publication and worker dispatch; existing fault tests cover both sides of that barrier.
   - `operationalrun.FinalizeNotice` preserves execution/output, rejects conflicting terminal results, and has abrupt-process-death tests.
   - Execution and notice admission have separate bounded capacity; workers join the service lifetime.
   - README and atlas describe the new automatic scheduling and notice surfaces in this range.

2. **Critical findings**

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: effect-authorization-fencing
    title: |
      Notice admission can proceed after its deadline or committed cancellation
    detail: |
      cmd/internal/couchcmd/schedule_automatic.go:650-654 checks authority before calling Admit, while schedule_notice.go:45 then waits for holdHubEffect using the service context. There is no persisted-deadline cancellation or task-authority recheck after that wait. Two deterministic probes using the production scheduler, real broker, held ownership gate, and injected clock reproduced broker admission after NoticeDeadline and after catalogue removal had committed, finalized history, and cleared Active. ARCH-ORDER and ARCH-CONSTRAINTS: enforce one rule across queued and blocked effects—admission must remain authorized until the actual broker boundary. Bound gate/admission work by the persisted deadline and serialize or cancel pending admission against declaration cancellation, preserving uncertainty for already accepted effects. Add both interleavings as permanent regression tests and return this ordering gap to the TL under the approved transaction revision.
  - id: new
    severity: Critical
    kind: defect
    family: task-refusal-isolation
    title: |
      One damaged task history prevents every scheduler worker from starting
    detail: |
      cmd/internal/couchcmd/schedule_automatic.go:151-153 returns a task-local passive-history error from the entire recovery transaction; run starts workers only after recovery succeeds. A probe created offline-skip history for two hubs, truncated one task's record.json, and reopened the scheduler: recover returned the JSON error, preventing all workers from starting. Similar global propagation remains in scheduled completion and notice-history paths. This is the 2nd finding in family task-refusal-isolation. Earlier corrections handled instances; do not patch only recover. State and enforce the rule that task-history failures preserve that task's authority and expose a local refusal, while shared authority-store failures block globally. Enumerate startup recovery, passive repair, completion reconciliation, cancellation finalization, and notice finalization; sweep those consumers and test that a healthy second hub continues. ARCH-PURPOSE.
  - id: new
    severity: Critical
    kind: defect
    family: additive-persisted-format-compatibility
    title: |
      Same-occurrence recovery silently deletes unknown summary fields
    detail: |
      cmd/internal/couchschedule/model.go:235 and :260 replace Last with the recovered summary, while cmd/internal/couchcmd/schedule_records.go:14-15 constructs that summary without the existing Last.Unknown members. A persisted scheduled-delivery summary containing future_summary_field={"value":42} lost that field after automaticScheduler.recover processed the same occurrence. Codec preservation tests do not cover this rewrite. This is the 2nd finding in family additive-persisted-format-compatibility. Do not fix only this caller: enforce preservation of unknown members whenever an existing persisted object is updated for the same identity, keeping distinct schema namespaces separate. Sweep manual, scheduled, passive, and uncertain canonical-adoption paths, with reopen-and-rewrite regressions. ARCH-EVOLVE.
```

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - Full `couchschedule` and `operationalrun` package tests.
   - Focused scheduler, reducer, state-codec, occurrence-publication, and notice-finalization tests across the three affected packages.
   - Pinned-range `git diff --check`.

   Four temporary Go-overlay probes failed at the pinned head: deadline admission, cancellation admission, task-history isolation, and unknown-summary preservation. They exercised production code using existing fixtures; temporary files were removed and the checkout remained unchanged.

   The live day and disposable exit-10/timeout qualification remain outstanding. The latest acceptance-sequencing revision permits this code review before that work; these tests do not establish live acceptance.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** automatic and manual execution share the runner and occurrence publisher.
   - **ARCH-PURE — pass:** recurrence and transition logic remain deterministic; clocks and effects enter through integration seams.
   - **ARCH-PURPOSE — flag:** task-local history failures disable unrelated hubs.
   - **ARCH-MOCK — pass:** stateful execution doubles, portable stores, and real-broker integration exercise shared production boundaries.
   - **ARCH-CONSTRAINTS — flag:** the persisted admission window does not bound blocked notice workers.
   - **ARCH-SECURE — pass:** reviewed paths validate persisted input and frozen bindings; no new credential exposure found.
   - **ARCH-ORDER — flag:** cancellation can commit while an already authorized worker later admits the cancelled notice.
   - **ARCH-FUNERAL — pass:** finalization reuses recoverable staging and existing retention; pending records are protected.
   - **ARCH-EVOLVE — flag:** codec preservation is defeated by canonical-summary replacement.

   Core-concept locations inspected match the listed entities; no PURE entity required IO to test. README and atlas updates are present.

7. **Plan revision recommendations**

   Add `## Revisions` entries that:

   - Correct the transaction inventory’s cancellation “Covered” claim and enumerate gate-wait × deadline/cancellation interleavings, with TL review as already required for another ordering gap.
   - Define and inventory task-local versus shared-store failure handling across every history consumer.
   - Extend additive-format qualification from codec round trips to same-identity recovery rewrites.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"bc0371b0-1975-48a7-b8fc-e851b329c8f4","repository":"github.com/xianxu/pair","issue_id":"000737","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/4cd79521962a6ba7","reviewed_head":"11db21abcaa8e29ad26e7914f8c6ecf86aec6d68","slot":"pair:1","requested_at":"2026-10-21T07:01:06.065439Z"},"phase":"finished","pid":25993,"started_at":"2026-10-21T07:01:06.294104Z","finished_at":"2026-10-21T07:07:18.426758Z","session_id":"01a15d43-7dda-7903-a57d-6d47ed804ece","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.577231s
head: "11db21abcaa8e29ad26e7914f8c6ecf86aec6d68"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409

---

## Re-review — 2026-10-21T08:53:10-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 737 — couch: scheduler: run each hub's scheduled tasks from its operations repo (due notices, direct script-run, skip-if-missed, --schedule --status) |
| repo | pair |
| issue file | workshop/issues/000737-couch-scheduler-run-each-hub-s-scheduled-tasks-from-its-operations-repo-due-notices-direct-script-run-skip-if-missed-schedule-status.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 369451d79fe2072aa39ee8b3928bfbf9e6f40132..8e26782e53cbabcf4eb7c86d81803045582dc4c0 |
| command | sdlc milestone-close --issue 737 --milestone M2 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T08:53:10-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-9, BR-10 and BR-11 are addressed, with regressions that fail when their fixes are removed. One new fault probe exposes a blocking recovery defect: cached canonical history is silently ignored after a failed ledger commit. README and atlas updates are present. Live acceptance remains outstanding under the documented TL sequencing revision.

```findings
dispose:
  - id: BR-9
    disposition: addressed
    note: |
      Held-gate deadline and cancellation regressions pass and fail when the admission fence is removed; additional tests cover deadline-bounded gate and broker waits.
  - id: BR-10
    disposition: addressed
    note: |
      Seven history-consumer isolation cases pass and fail without local quarantine; manual refusal, pending-evidence retry, and shared-store failure tests also pass.
  - id: BR-11
    disposition: addressed
    note: |
      Manual and scheduled recovery regressions pass and lose persisted summary metadata when the central same-identity merge is removed; broader rewrite tests cover passive and uncertain paths.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: durable-artifact-crash-recovery
    title: |
      Cached canonical recovery is ignored after a failed authority commit
    detail: |
      cmd/internal/couchcmd/schedule_automatic.go:250 constructs ResultRecovered without At, so model.go:162-163 silently ignores it. A production-path overlay probe published canonical passive history with different timestamps, failed the following ledger commit, and retried: the ledger retained EndedAt=12:00:00 instead of canonical 11:59:50. ARCH-ORDER: cached and freshly read evidence must produce equivalent recovery transitions, including after failed persistence. This is the 4th finding in family durable-artifact-crash-recovery. Earlier rounds fixed instances; state and enforce that rule across cached/direct adoption and failed-commit retries, rather than patching only this call. Supply the canonical event time and add a permanent regression asserting the reopened ledger matches canonical history.
```

1. **Strengths**

   - Admission now acquires the hub gate before authority and retains authority through bounded broker admission.
   - History quarantine preserves task authority while healthy hubs continue.
   - Same-occurrence rewrites preserve summary metadata without importing operational-record metadata.
   - Notice finalization preserves execution/output and uses existing retention and recovery mechanisms.

2. **Critical findings**

   - Cached recovery event lacks its required timestamp; see the finding above.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed scheduler race tests, full `couchschedule`, `operationalrun`, and `artifactpath` tests, plus pinned-range whitespace checks. Removing the prior fixes made their regressions fail. The new failed-commit probe fails at the pinned head. Temporary overlays were removed; the checkout is unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared execution and occurrence publication.
   - **ARCH-PURE — pass:** deterministic reducer; IO remains behind integration seams.
   - **ARCH-PURPOSE — pass:** automatic scheduling is delivered; live acceptance remains explicitly open.
   - **ARCH-MOCK — pass:** stateful doubles and portable stores exercise production boundaries.
   - **ARCH-CONSTRAINTS — pass:** bounded worker capacity and admission deadlines.
   - **ARCH-SECURE — pass:** persisted input and frozen bindings are validated.
   - **ARCH-ORDER — flag:** cached recovery silently produces no transition.
   - **ARCH-FUNERAL — pass:** new artifacts reuse existing cleanup and retention.
   - **ARCH-EVOLVE — pass:** reviewed same-identity metadata preservation is enforced and tested.

   Core-concept locations match the listed entities; inspected PURE tests require no IO.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining equivalent cached/direct canonical adoption, enumerating failed-commit retry paths, and requiring ledger-to-history equality assertions alongside history immutability checks.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"d5538866-60e8-4bb3-84a5-4ffa0b7360fe","repository":"github.com/xianxu/pair","issue_id":"000737","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/f4da20a12c709819","reviewed_head":"8e26782e53cbabcf4eb7c86d81803045582dc4c0","slot":"pair:1","requested_at":"2026-10-21T15:48:31.12272Z"},"phase":"finished","pid":68627,"started_at":"2026-10-21T15:48:31.224053Z","finished_at":"2026-10-21T15:53:10.190743Z","session_id":"01a15f26-5a7b-7ac3-aea5-4a432178e00d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.58914175s
head: "8e26782e53cbabcf4eb7c86d81803045582dc4c0"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-21T09:06:35-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 737 — couch: scheduler: run each hub's scheduled tasks from its operations repo (due notices, direct script-run, skip-if-missed, --schedule --status) |
| repo | pair |
| issue file | workshop/issues/000737-couch-scheduler-run-each-hub-s-scheduled-tasks-from-its-operations-repo-due-notices-direct-script-run-skip-if-missed-schedule-status.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 369451d79fe2072aa39ee8b3928bfbf9e6f40132..e72bfd54f67bed8dd8b2947281286bfa61fb008e |
| command | sdlc milestone-close --issue 737 --milestone M2 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T09:06:35-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-12 is addressed: its regression fails with the old cached producer restored and passes at the pinned head. The normal race suites pass. One new blocking defect remains: several recovery consumers treat readable history as durable evidence even after publication directory sync fails.

```findings
dispose:
  - id: BR-12
    disposition: addressed
    note: |
      TestScheduleCachedCanonicalRecoveryAfterFailedCommit passes at head and fails for skipped, refused, and completed outcomes with the old missing-At producer restored through a disposable overlay. All production recovery producers use NewRecoveryEvent; direct and cached scheduled adoption share adoptScheduledSummary.
  - id: BR-9
    disposition: addressed
    note: |
      Admission retains authority through bounded broker admission after acquiring the hub gate. Deadline/cancellation regression tests pass under race.
  - id: BR-10
    disposition: addressed
    note: |
      Shared quarantine handling preserves task authority and isolates history failures. Startup, consumer-isolation, and pending-evidence regressions pass under race.
  - id: BR-11
    disposition: addressed
    note: |
      Same-ID reducer rewrites preserve summary unknown members separately from operational-record metadata. Codec, consumer, and detached-snapshot regressions pass under race.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: durable-artifact-crash-recovery
    title: |
      Recovery clears authority using history whose durability remains unconfirmed
    detail: |
      schedule_automatic.go:190 reads history through Store.Read and adopts it at line 213 without completing publication durability. A disposable regression injected directory-sync failure after bundle rename: Publish returned an error, the bundle remained readable, and startup recovery nevertheless cleared Active while the same sync still failed. The same regression failed through schedule_manual.go:173 and schedule_service.go:102. The notify-only adoption path at schedule_automatic.go:468 also bypasses durable confirmation. This violates the history-before-authority-release contract and can leave committed completion authority referring to output whose publication was never confirmed durable (ARCH-ORDER, ARCH-PURPOSE). This is the 5th finding in family durable-artifact-crash-recovery. Earlier rounds fixed instances; do NOT fix only this instance. Enforce one rule across every adoption consumer: readable history becomes authoritative only after its publication/finalization durability is confirmed under the task lock. Reuse a shared durable adoption boundary, retain authority on failure, and add permanent regressions across the consumer inventory.
```

1. **Strengths**

   - Cached and direct canonical recovery now share `adoptScheduledSummary`, with checks covering canonical timestamps and other known facts.
   - Notice admission explicitly orders the hub gate before scheduler authority and preserves the original deadline.
   - Same-occurrence unknown metadata survives recovery without mixing record and summary namespaces.
   - README, atlas, and artifact-source inventory cover the new automatic scheduling surface.

2. **Critical findings**

   The finding above blocks shipment. `Store.Read` establishes readability, while `PublishOccurrence` explicitly completes directory durability. Recovery currently bypasses that distinction.

   In the disposable probe, routing startup adoption through `PublishOccurrence` made the regression pass: failed sync preserved authority. Sweep startup/reconciliation, reserved execution, manual refresh, and notify-only adoption together.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Full `couchschedule` and `operationalrun` packages passed under race.
   - `Test(Schedule|Skill|HelpAndSkill|MessageSkill)` passed under race.
   - BR-12 mutation failed all three regression cases as required.
   - New durability probes failed in three production consumers; the startup correction probe passed.
   - Diff whitespace check passed; checkout remains unchanged.
   - Day-long live acceptance remains outstanding under the plan’s TL acceptance sequence.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared executor, reducer, and broker seams.
   - **ARCH-PURE — pass:** core transitions remain deterministic; IO stays in integration adapters.
   - **ARCH-PURPOSE — flag:** recovery must enforce durability across the complete consumer class.
   - **ARCH-MOCK — pass:** stateful runtime and clock doubles support controlled ordering; native tests qualify boundaries.
   - **ARCH-CONSTRAINTS — pass:** bounded workers, deadlines, output, and counted hub lookups.
   - **ARCH-SECURE — pass:** private storage, validated routing, argv execution, and explicit trusted-task envelope.
   - **ARCH-ORDER — flag:** readable evidence is incorrectly promoted to durable completion.
   - **ARCH-FUNERAL — pass:** retention, pending protection, and cache pruning cover new residue.
   - **ARCH-EVOLVE — pass:** additive codecs and same-ID metadata preservation are tested.

   Core-concept locations and PURE/INTEGRATION classifications match the inspected implementation.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining the durable-adoption rule and enumerating every history-to-authority consumer. Record regression evidence that sync failure preserves authority in each consumer, alongside successful canonical adoption.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a0fddd69-28c0-484d-95d6-6e386f230e12","repository":"github.com/xianxu/pair","issue_id":"000737","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/9f972f0a3a987cb9","reviewed_head":"e72bfd54f67bed8dd8b2947281286bfa61fb008e","slot":"pair:1","requested_at":"2026-10-21T16:01:26.215877Z"},"phase":"finished","pid":48930,"started_at":"2026-10-21T16:01:26.350844Z","finished_at":"2026-10-21T16:06:34.248954Z","session_id":"01a15f32-2ea4-72d3-a489-82499f39ec34","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.23731125s
head: "e72bfd54f67bed8dd8b2947281286bfa61fb008e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
