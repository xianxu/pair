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

## Plan

- [ ] Investigate and design the smallest ownership fix; approval before code.
- [ ] Implement with regression and loaded before/after evidence.
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
