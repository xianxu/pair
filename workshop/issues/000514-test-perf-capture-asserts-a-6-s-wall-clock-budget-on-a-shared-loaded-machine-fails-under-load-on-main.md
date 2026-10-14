---
id: 000514
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-13
estimate_hours:
card_mirror: 'a5883bfedc2f97bc69b0ad01b42cde036d331c74' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T18:20:35-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# test-perf-capture asserts a 6 s wall-clock budget on a shared loaded machine; fails under load on main

## Problem

`make test-perf-capture` (`doctor/perf_test.sh`) asserts that PairDoctor's machine-diagnostics capture finishes within a 6 s
wall-clock budget. On the shared machine, with other slots' full suites running, it fails on unchanged main:
10-12 18:14, load 64/72/62, main failed twice (7 s, 16 s); at load ~180 it took 11-12 s. It blocked #505's close
(which doesn't touch capture) until the TL waived it. A wall-clock bound on a shared, loaded machine measures the
machine, not the code (see lessons: "A stopwatch ceiling hides the race it was meant to catch", #479).

## Spec

Use the existing `PAIR_PERF_BUDGETS=1` convention: normal capture tests always validate report shape and log elapsed/budget, while only the isolated performance lane fails on wall-clock overrun. Preserve the 6-second default and production collector/probe behavior. CPU accounting misses sleeping or blocked collectors; a relative baseline can normalize away a regression and depends on concurrent load changing similarly. The existing opt-in lane avoids both new measurement policy and a misleading shared-host ceiling (ARCH-DRY, ARCH-CONSTRAINTS).

Default verification must still catch real scheduling-policy regressions: a stateful fake clock and PATH tools run the actual capture script, advance time during named collectors, and record invocations. Pin one top call, skipping low-value work at the reserve boundary, retaining probes while budget remains, and skipping probes at exhaustion. Assert output as well as calls. This tests behavior independently of host scheduling (ARCH-MOCK). Add a single-capture `--budget-only` entry to the test script for the isolated lane and policy regression harness; report/clock validation still runs there. No product API or persisted state changes.

## Done when

- [ ] Default capture test accepts a controlled overrun, logs timing, and retains all shape/parser/failure assertions.
- [ ] The same controlled overrun fails with `PAIR_PERF_BUDGETS=1`; boundary-equal and under-budget captures pass.
- [ ] Controlled production-script tests catch lost work shedding, duplicate expensive collection, skipped reserved probes and probes run after exhaustion.
- [ ] `make test-perf-budgets` includes the isolated capture check; targeted tests, full `make -k test`, and review/CI evidence are recorded before landing.

## Plan

- [ ] Add RED controlled-clock/process tests in `doctor/perf_budget_test.py`, including default versus opt-in gate behavior and production work-shedding invariants.
- [ ] Modify `doctor/perf_test.sh` to log timings, validate numeric fields and enforce only in opt-in mode; add `--budget-only` for the single-capture lane. Wire deterministic tests into `test-perf-capture` and the isolated capture into `test-perf-budgets` in `Makefile.local`.
- [ ] Run controlled tests and real capture shape tests; mutation-check budget and shedding assertions, run full `make -k test`, and document the testing contract in `atlas/couch.md`. Close, publish, wait for CI, land on SHIP and report.

## Log

### 2026-10-12
- 2026-10-12: unclaimed: filed for the TL as an unassigned follow-up of #505

### 2026-10-13 — investigation and design
- Claimed from the hub after #538 landed. Reproduced unchanged main: `sh doctor/perf_test.sh` reports `capture took 13s over a 6s budget` in `/tmp/pair514-baseline.log`. The measured interval includes external top/iostat/ps/probe processes and intentional sample sleep. Existing #436/#481 convention already isolates elapsed-time budgets; this shell check missed it. Production budget is cooperative work shedding, not a hard subprocess deadline, so CPU-time replacement would assert a different property. Quick-flow scope: test scripts, Make target wiring and atlas only; no product behavior change.
