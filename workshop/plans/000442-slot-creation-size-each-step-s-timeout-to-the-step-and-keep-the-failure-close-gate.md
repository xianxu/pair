---
gate: boundary-review
issue: 442
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T10:23:16-07:00"
      dispatched: "2026-10-11T10:20:41-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: TestFailedCreationIsRecordedAndRetryIsClean retry runs real git worktree add under the 200ms WriteTimeout and flakes
          detail: provision_budget_test.go:93-97 keeps p.WriteTimeout=200ms when slow.delay=0 delegates to the fixture's real checkout; reproduced twice (once in a filtered package run, once in -count=5 of the three new tests) with "timed out after 200ms ... Preparing worktree". Restore p.WriteTimeout (e.g. time.Minute) before the retry Ensure.
          family: test-budget-below-real-work
          round: 1
        - id: BR-2
          severity: Important
          title: Failure record silently not written when the cause exceeds the 16 KiB provision record limit
          detail: newSlotFailure stores the full e.Err.Error(), which carries up to a 64 KiB diagnostic tail (provision_io.go:89); ProvisionStore.Write rejects records over 16 KiB (provision_store.go:15,80) and recordSlotFailure discards the error. Long-output failures such as weave compile leave no record. Keep only the last ~8 KiB of Cause before writing, and test it.
          family: best-effort-write-silently-dropped
          round: 1
        - id: BR-3
          severity: Minor
          title: clearSlotFailure runs before verifyHost; a later verifyHost or marker failure erases the record and saves nothing
          family: record-cleared-before-success
          round: 1
        - id: BR-4
          severity: Minor
          title: ReconcileAdvice FailureUnknown and FailureHold branches omit the step name
          family: step-in-advice-coverage
          round: 1
        - id: BR-5
          severity: Minor
          title: slotReport ignores ReadSlotFailure errors, so a corrupt last-failure.json shows nothing in couch --show
          family: untrusted-read-degrades-visibly
          round: 1
        - id: BR-6
          severity: Minor
          title: couch --show "last failure" line has no render test in slotreport_render_test.go
          family: render-surface-untested
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-11T10:33:06-07:00"
      dispatched: "2026-10-11T10:30:51-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: provision_budget_test.go:243 restores WriteTimeout=time.Minute before the retry; 5 runs green at -count=5.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: 'boundCause limits raw bytes but the 16 KiB limit applies after json.Marshal; 64 KiB stderr of "\x1b[0m" or "<" still records nothing (probe-tested). Rule: limit by encoded size, fall back to a smaller record, never drop.'
          round: 2
        - id: BR-3
          disposition: addressed
          note: clearSlotFailure now runs after verifyHost and the marker read (provision.go:139); nothing after it can fail.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: Code fixed in every branch, but no test asserts the step in Unknown/Hold advice; TestReconcileAdviceText unchanged.
          round: 2
        - id: BR-5
          disposition: addressed
          note: 'slotreport.go:530-534 sets LastFailureError; run.go renders "last failure: unreadable"; covered by TestRenderSlotReportLastFailure.'
          round: 2
        - id: BR-6
          disposition: addressed
          note: TestRenderSlotReportLastFailure (slotreport_render_test.go:127) pins both lines (read; not run here because embed assets are missing).
          round: 2
      findings:
        - id: BR-7
          severity: Important
          title: atlas/workspace-provisioning.md not updated for the write budget or last-failure.json
          detail: Line 138 still lists probe 5 s / fetch 120 s / compile 20 min with no 2-min write budget; the couch-workspaces/N/last-failure.json record (writer, clearing, size bound) and the couch --show "last failure" line are undocumented.
          family: atlas-missing-new-surface
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-11T10:43:08-07:00"
      dispatched: "2026-10-11T10:38:11-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: addressed
          note: fitSlotFailure bounds the encoded record to 16 KiB; removing it makes TestSlotFailureRecordFitsTheStore/escaped fail with "provision record too large".
          round: 3
        - id: BR-4
          disposition: addressed
          note: ReconcileAdvice names the step for every class and cause (slotfailure.go:106-128), pinned by the AllFailureClasses loop in TestReconcileAdviceText.
          round: 3
        - id: BR-7
          disposition: addressed
          note: atlas/workspace-provisioning.md:138-150 documents the 2-minute write budget and the last-failure.json writer, clearing, size bound and --show line.
          round: 3
      findings:
        - id: BR-8
          severity: Minor
          title: clearSlotFailure ignores lease and remove errors, so --show can keep showing an old failure after the slot converged
          detail: 'This is the 2nd finding in family best-effort-write-silently-dropped. Rule: any write or remove of the slot-failure record that fails is reported to the caller (RecordErr when the run blocks, a ProvisionResult.Warning after a successful run). Prevalence: two sites, recordSlotFailure (fixed) and clearSlotFailure (slotmemo.go:131-142).'
          family: best-effort-write-silently-dropped
          round: 3
        - id: BR-9
          severity: Minor
          title: boundCause panics with a slice index out of range when limit is smaller than head plus the elision marker
          detail: 'fitSlotFailure halves limit down to 1; once limit is about 6 or less, len(cause)-(limit-head-len(marker)) is past the end of the string. Only reachable if the rest of the record is over about 16 KiB. Guard with: if limit <= len(marker), return "".'
          family: bound-helper-total-over-input
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: 06f754f4e34ff179fbc369e19ea9cafb42327738
      blocked: false
---

# Gate ledger — pair#442 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T10:23:16-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `test-budget-below-real-work` TestFailedCreationIsRecordedAndRetryIsClean retry runs real git worktree add under the 200ms WriteTimeout and flakes
  provision_budget_test.go:93-97 keeps p.WriteTimeout=200ms when slow.delay=0 delegates to the fixture's real checkout; reproduced twice (once in a filtered package run, once in -count=5 of the three new tests) with "timed out after 200ms ... Preparing worktree". Restore p.WriteTimeout (e.g. time.Minute) before the retry Ensure.
- **BR-2** [Important] `best-effort-write-silently-dropped` Failure record silently not written when the cause exceeds the 16 KiB provision record limit
  newSlotFailure stores the full e.Err.Error(), which carries up to a 64 KiB diagnostic tail (provision_io.go:89); ProvisionStore.Write rejects records over 16 KiB (provision_store.go:15,80) and recordSlotFailure discards the error. Long-output failures such as weave compile leave no record. Keep only the last ~8 KiB of Cause before writing, and test it.
- **BR-3** [Minor] `record-cleared-before-success` clearSlotFailure runs before verifyHost; a later verifyHost or marker failure erases the record and saves nothing
- **BR-4** [Minor] `step-in-advice-coverage` ReconcileAdvice FailureUnknown and FailureHold branches omit the step name
- **BR-5** [Minor] `untrusted-read-degrades-visibly` slotReport ignores ReadSlotFailure errors, so a corrupt last-failure.json shows nothing in couch --show
- **BR-6** [Minor] `render-surface-untested` couch --show "last failure" line has no render test in slotreport_render_test.go

## Round 2 — 2026-10-11T10:33:06-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — provision_budget_test.go:243 restores WriteTimeout=time.Minute before the retry; 5 runs green at -count=5.
- BR-2 — not-addressed — boundCause limits raw bytes but the 16 KiB limit applies after json.Marshal; 64 KiB stderr of "\x1b[0m" or "<" still records nothing (probe-tested). Rule: limit by encoded size, fall back to a smaller record, never drop.
- BR-3 — addressed — clearSlotFailure now runs after verifyHost and the marker read (provision.go:139); nothing after it can fail.
- BR-4 — not-addressed — Code fixed in every branch, but no test asserts the step in Unknown/Hold advice; TestReconcileAdviceText unchanged.
- BR-5 — addressed — slotreport.go:530-534 sets LastFailureError; run.go renders "last failure: unreadable"; covered by TestRenderSlotReportLastFailure.
- BR-6 — addressed — TestRenderSlotReportLastFailure (slotreport_render_test.go:127) pins both lines (read; not run here because embed assets are missing).

### Raised

- **BR-7** [Important] `atlas-missing-new-surface` atlas/workspace-provisioning.md not updated for the write budget or last-failure.json
  Line 138 still lists probe 5 s / fetch 120 s / compile 20 min with no 2-min write budget; the couch-workspaces/N/last-failure.json record (writer, clearing, size bound) and the couch --show "last failure" line are undocumented.

## Round 3 — 2026-10-11T10:43:08-07:00 (claude) — passed

### Disposed

- BR-2 — addressed — fitSlotFailure bounds the encoded record to 16 KiB; removing it makes TestSlotFailureRecordFitsTheStore/escaped fail with "provision record too large".
- BR-4 — addressed — ReconcileAdvice names the step for every class and cause (slotfailure.go:106-128), pinned by the AllFailureClasses loop in TestReconcileAdviceText.
- BR-7 — addressed — atlas/workspace-provisioning.md:138-150 documents the 2-minute write budget and the last-failure.json writer, clearing, size bound and --show line.

### Raised

- **BR-8** [Minor] `best-effort-write-silently-dropped` clearSlotFailure ignores lease and remove errors, so --show can keep showing an old failure after the slot converged
  This is the 2nd finding in family best-effort-write-silently-dropped. Rule: any write or remove of the slot-failure record that fails is reported to the caller (RecordErr when the run blocks, a ProvisionResult.Warning after a successful run). Prevalence: two sites, recordSlotFailure (fixed) and clearSlotFailure (slotmemo.go:131-142).
- **BR-9** [Minor] `bound-helper-total-over-input` boundCause panics with a slice index out of range when limit is smaller than head plus the elision marker
  fitSlotFailure halves limit down to 1; once limit is about 6 or less, len(cause)-(limit-head-len(marker)) is past the end of the string. Only reachable if the rest of the record is over about 16 KiB. Guard with: if limit <= len(marker), return "".

## Open findings

- **BR-8** [Minor] `best-effort-write-silently-dropped` clearSlotFailure ignores lease and remove errors, so --show can keep showing an old failure after the slot converged
- **BR-9** [Minor] `bound-helper-total-over-input` boundCause panics with a slice index out of range when limit is smaller than head plus the elision marker
