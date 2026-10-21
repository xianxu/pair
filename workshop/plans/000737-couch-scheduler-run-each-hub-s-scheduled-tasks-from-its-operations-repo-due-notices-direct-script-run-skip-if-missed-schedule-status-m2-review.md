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
