---
id: 000545
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '1bbac479d83e4132a03f67d989ec7dd9cc66e2b5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T23:20:11-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "5c79bc89", done: "7cacdd61"}
---

# couchcore: eliminate spurious workspace creation lock contention

## Problem

#543's complete make -k test run failed six couchcore tests with
`workspace host creation is busy` (one observed loaded suite, six failures):

- TestManagedCreateReuseRefusesProfileDriftDuringSetup
- TestReconcileResultListsWhatItSetAside
- TestSavedWorkIsCollectedPastRetention
- TestSetAsideHoldsMatchThePlan
- TestSetAsideMovesTheWholeCheckoutAndCanBeRestored
- TestSetAsideHostRecordsItsBranch

All six passed unchanged in one isolated rerun (26.819s). Logs:
/tmp/pair543-full-suite.log and /tmp/pair543-core-rerun.log. This is an initial
sample, not a general failure-rate estimate. TL also reports two failures during
#528; the peer has been asked for names/counts. Establish repeated loaded
baseline and fixed counts, following ariadne#360/#361's interleaved method.

## Spec

Identify the actual owner/lifetime of the contended lock. Correct the cause
without increasing a timeout or weakening exclusion while a real workspace
producer remains alive. Distinguish shared test paths from incidental fork
inheritance; use deterministic regression evidence and repeated runs alongside
a complete suite. Preserve parent-death and intentionally inherited lease safety.

## Done when

- Root cause is demonstrated, with the lock path and owner/lifetime identified.
- A deterministic regression fails before the fix and passes after it; genuine
  competing producers still exclude a second creation, including parent death.
- Repeated before/after runs alongside suite load record exact failure counts
  and denominators, at least 50 repetitions of the affected cases per variant.
- Full make -k test completes, failures are explained, close returns SHIP and
  the PR lands through sdlc with validation.

- Active/unknown hints keep the original 100ms retry budget; released hints may
  wait at most 2s from the initial attempt, never renewed by owner turnover.
  Only a successful kernel flock permits creation.

## Plan

- [x] Investigate and design the smallest ownership fix; approval before code.
- [x] Implement with regression and loaded before/after evidence.
- [ ] Complete suite, documentation, review and landing.

## Log

### 2026-10-13

- Filed and claimed via TL receipt 0a70adcf-201a-45b0-9c71-fb6704a2b9c6.
  start-plan refreshed the slot's Ariadne and Pair baselines; rerun started the
  branch. Existing fixture roots are unique temp directories, despite shared
  repo-name basenames. Leading hypothesis is incidental fork inheritance of the
  descriptor flock: Close retains lock while any fork copy exists, and the
  existing 100ms settle is not a lifetime proof. Intentional inheritance must stay.

### 2026-10-13 — additional baseline and investigation

pair:2 receipt da32d999-8b91-47c8-a7df-fe843719cb57 confirms #528 failed
TestOpenRefusesWithTheHandoffWhenSetupCannotConverge and
TestReconcileAgreesWithRealGit/branch=broken/elsewhere once each in
/tmp/pair528-m1-corrected-packages.log (core 296.305s). Both passed an unchanged
isolated rerun (20.384s) and a subsequent complete core rerun (363.825s).

On this branch, the first repeated baseline batch passed 80/80 top-level
executions (eight cases x10; the Reconcile case included all subcases) while
make -k test ran. A second x10 batch with temporary busy-holder tracing and the
single failing Reconcile subcase also passed 80/80. Logs are
/tmp/pair545-before-load.log and /tmp/pair545-trace-load.log. The tracer is a
Go build overlay in /tmp, not a production edit; a full traced core run is next.
Do not interpret this as a zero real-world rate or confirmed causality.

Fixture common directories derive from unique t.TempDir roots, not the shared
repo-name label. The lock is close-only intentionally: producer fd3 must keep it
through parent exit. Go releases syscall.ForkLock before waiting for child exec,
so that mutex alone cannot settle incidental copies. Unconditional LOCK_UN
would violate the existing child/parent-death tests. Investigating an ownership
fix rather than increasing 100ms or serializing tests.

### 2026-10-14 — fork probe and resumed direction

The baseline full make -k test completed successfully (/tmp/pair545-baseline-suite.log).
The traced full-core binary was accidentally run from the repository root; its
relative-source inventory failures are a probe harness error, not product failures.
It found only expected busy holders in exclusion tests, not a historical failure.
The two targeted batches remain 80/80 passes each; the second used tracing.

A controlled native fork probe (/tmp/pair545-fork-probe.c) set FD_CLOEXEC, locked
the file, forked a child paused before exec, and closed the parent's descriptor.
A fresh open still returned EWOULDBLOCK (errno 35); after the child execed true,
the fresh descriptor acquired the lock. Creation already opens O_CLOEXEC. This
proves the kernel mechanism, but does not yet identify a holder from an actual
loaded-suite failure.

## Revisions

### 2026-10-14 — TL narrows the remedy after the kernel probe

TL receipt 78aedcac-58c2-43ef-8322-b937f8a0f009 approves investigating a short
bounded backoff when a busy acquirer has evidence the previous owner released.
This refines the initial prohibition on increasing a timeout: preserve kernel
exclusion and distinguish a released descriptor still inherited before exec
from active ownership, rather than merely increasing the unconditional 100ms.
Check the atlas's busy/no-queue contract, add a paused-fork regression, and retain
exact before/after load counts. Receipt 6c74a424-9044-48d3-89ad-39909f05d11c
asks to resume this work after model capacity failures. No production changes yet.

### 2026-10-14 — concrete design

Durable plan: workshop/plans/000545-workspace-creation-lock-contention-plan.md.
Read-only peer review supports a held/released advisory byte, with kernel flock
as sole admission and one absolute bounded retry window. Unknown/held retains
100ms; released permits 2s with backoff. Native C paused-fork conformance plus
pure retry boundary tests avoid forking inside Go. TL proposal sent as receipt
27999969-68e9-4b54-9ff4-b7c6c294e132.

The open-site sweep found no missing CLOEXEC on creation. Raw opens lacking the
flag exist in diagnosticlog.openRegular and notifytransport.lockNamespace; both
locks explicitly unlock. Reported separately; no unrelated changes planned.
History confirms #436's 100ms was a measured implementation choice reduced from
500ms because intentional contention paid the full budget, not a requirement
for instantaneous refusal. The new plan preserves fast active-owner refusal.

### 2026-10-14 — implementation regression evidence

TL approved bdaf65c4 in receipt cc82ffc3-c4f5-4c1c-9e9a-5b2fb14f5340;
change-code passed (quick inferred). The real paused-fork regression failed
against unchanged production (50/50 failures, /tmp/pair545-forced-before-50.log),
and the first fixed run passed 50/50 (/tmp/pair545-forced-fixed-50.log).

A broader targeted run exposed a flaw in the old dup fixture: syscall.Dup made
a non-CLOEXEC duplicate, so unrelated execs could inherit it beyond the timer's
close. The test now duplicates atomically with F_DUPFD_CLOEXEC to match the real
lease; this is distinct from the intentional pre-exec fork window. One failure
in /tmp/pair545-targeted-final.log led to this fixture correction. No production
exclusion failure was observed. Follow-up descriptor hygiene issue filed #551;
publishing its body needs a main merge because it was created after this branch.

### 2026-10-14 — stable verification in progress

After merging main (including #546), the final matched native-fork harness was
built twice: current sources, and a scratch Go overlay replacing only
provision_lock_unix.go with its unchanged baseline. Final counts: baseline
50/50 failures (/tmp/pair545-forced-baseline-final-50.log), fixed 50/50 passes
(/tmp/pair545-forced-final-50.log). Both run alongside the natural-load batches
and suite. The fixture now uses t.TempDir per build and process-group cancellation.

Targeted lease/retry/recovery/cancellation tests passed in 4.966s after correcting
the dup fixture (/tmp/pair545-targeted-corrected.log). A load-only make -k test
was allowed to complete but overlapped source construction and failed compiling
the not-yet-added creationLeaseRetry symbol; it is not final verification. A fresh
full suite runs on committed stable sources (/tmp/pair545-final-suite.log).
The natural before/fixed batches each target 50 repetitions of eight historical
cases; do not infer a natural failure-rate reduction from forced-fork results.

#551's body is published and its claim released after merging the publication
back into this branch. It remains independent and unstarted.

### 2026-10-14 — source inventory correction

The stable full suite found that provision_retry_unix.go was missing from
artifactpath's exhaustive production-source inventory. Added it beside the
other provisioning non-artifact sources; the complete artifactpath package then
passed (13.724s, /tmp/pair545-inventory-rerun.log). The running full suite is
continuing to completion, as required. This is a manifest correction, not a
retry behavior change.

### 2026-10-14 — baseline complete and timing reruns

The unchanged natural-load binary completed 400/400 top-level passes: exactly
50 each of the eight listed tests, including 50 passes of the historically
failing branch=broken/elsewhere subcase. /tmp/pair545-before-50.log. Fixed
counts are still pending. Observed load averages reached 25.97 (one-minute),
with before/fixed binaries overlapping full suites. This sample does not
reproduce the original natural failure rate.

The full suite /tmp/pair545-final-suite.log completed (exit 2): inventory
registration failure already fixed; TestRetentionExplicitResumeTouchesButBackgroundReattachDoesNot/false
exceeded its unchanged 200ms registration budget;
TestSpawnTimerMeasuresTheCommandNotTheHarness rejected a 19.723ms median under
load. Both unchanged timing tests passed targeted reruns (2.101s and 2.116s;
/tmp/pair545-retention-rerun.log, /tmp/pair545-hoprtt-rerun.log). A second
full suite with the inventory fix continues at /tmp/pair545-complete-suite.log.

Added the approved 100ms/2s admission limits explicitly to Done when (the
Revisions design already specified them), so acceptance tracks the final scope.

### 2026-10-14 — full suite green

The complete make -k test with the inventory correction passed, exit 0:
/tmp/pair545-complete-suite.log; couchcore 333.434s. No suite bypass or early
stop. The fixed natural batch is finishing its remaining repetitions.

### 2026-10-14 — final comparison counts

Both natural batches completed exit 0. Each of the eight historical top-level
cases passed exactly 50 times; branch=broken/elsewhere passed 50 times within
the Reconcile parent. Before and fixed binaries were built from the same base
before main's #546 integration; their lease behavior differs by #545. The
final constructor extraction only factors existing initialization/cleanup and
is covered by targeted tests (7.035s, /tmp/pair545-final-error-coverage.log).

| Scenario | Baseline failures/attempts | Fixed failures/attempts |
|---|---:|---:|
| Eight historical cases, 50 each | 0/400 | 0/400 |
| Historical Reconcile subcase (included above) | 0/50 | 0/50 |
| Controlled real paused fork, matched final harness | 50/50 | 0/50 |

Logs: /tmp/pair545-before-50.log, /tmp/pair545-fixed-50.log,
/tmp/pair545-forced-baseline-final-50.log, /tmp/pair545-forced-final-50.log.
The natural samples overlapped full suites, with observed one-minute load up
to 25.97; the remaining fixed tail ran after a suite ended. No natural-rate
reduction can be estimated because the unchanged baseline was green. The
controlled fixture proves the release/fork/exec failure and correction.

The final full suite after the initialization error-path test/refactoring is
queued by the existing weighted test gate (/tmp/pair545-final-coverage-suite.log);
no bypass, manual lock change or custom wait loop. The previous complete suite
passed, and final targeted lease/retry/recovery/cancellation tests passed.
