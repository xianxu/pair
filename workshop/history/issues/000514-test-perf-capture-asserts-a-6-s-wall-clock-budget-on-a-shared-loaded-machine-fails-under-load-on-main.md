---
id: 000514
status: done
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-13
estimate_hours:
card_mirror: '934ac00177733ca94e4e14534bcfd194f8c90c17' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T18:20:35-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "bd444b79", done: "4bedd11b"}
actual_hours: 23.76
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

- [x] Default capture test accepts a controlled overrun, logs timing, and retains all shape/parser/failure assertions.
- [x] The same controlled overrun fails with `PAIR_PERF_BUDGETS=1`; boundary-equal and under-budget captures pass.
- [x] Controlled production-script tests catch lost work shedding, duplicate expensive collection, skipped reserved probes and probes run after exhaustion.
- [x] Privacy guidance tests pin v3 per-note scope, aggregate non-authority and legacy fallback.
- [ ] `make test-perf-budgets` includes the isolated capture check; targeted tests, full `make -k test`, and review/CI evidence are recorded before landing.

## Plan

- [x] Add RED controlled-clock/process tests in `doctor/perf_budget_test.py`, including default versus opt-in gate behavior and production work-shedding invariants.
- [x] Modify `doctor/perf_test.sh` to log timings, validate numeric fields and enforce only in opt-in mode; add `--budget-only` for the single-capture lane. Wire deterministic tests into `test-perf-capture` and the isolated capture into `test-perf-budgets` in `Makefile.local`.
- [ ] Run controlled tests and real capture shape tests; mutation-check budget and shedding assertions, run full `make -k test`, and document the testing contract in `atlas/couch.md`. Close, publish, wait for CI, land on SHIP and report.

## Log

### 2026-10-12
- 2026-10-12: unclaimed: filed for the TL as an unassigned follow-up of #505

### 2026-10-13 — investigation and design
- 2026-10-13: closed — Nine controlled capture tests and all six mutation checks passed; live capture shape and Make environment isolation passed. Full make -k test completed with only the inherited pre-fix #538 privacy assertion failure; complete couchcmd package rerun passed after repair (163.234s), all other full-suite targets passed. Opt-in live ceiling still rejects 9s/6s under load; normal lane logs timing, no quiet-host performance claim. --no-plan-check: remaining row includes this review and subsequent CI/publication/landing, which are not falsely marked complete.; review verdict: SHIP
- Claimed from the hub after #538 landed. Reproduced unchanged main: `sh doctor/perf_test.sh` reports `capture took 13s over a 6s budget` in `/tmp/pair514-baseline.log`. The measured interval includes external top/iostat/ps/probe processes and intentional sample sleep. Existing #436/#481 convention already isolates elapsed-time budgets; this shell check missed it. Production budget is cooperative work shedding, not a hard subprocess deadline, so CPU-time replacement would assert a different property. Quick-flow scope: test scripts, Make target wiring and atlas only; no product behavior change.

- Controlled regression suite: 8 tests pass (17.860 s), exercising default/opt-in overrun policy, equal/under-budget cases, malformed timing, collector counts, reserve-boundary shedding and probe priority/exhaustion. Initial gate test was RED against unchanged `perf_test.sh`; it rejected the controlled 7/6-second report. Live opt-in check correctly remains red under present load: 9/6 seconds (`/tmp/pair514-isolated-live.log`), so no quiet-host performance pass is claimed. Full `make -k test` is queued through the refreshed ordered machine gate; targeted capture/environment tests are running.

- Final controlled suite 9/9 passed (10.955 s, `/tmp/pair514-budget-final.log`). All six temporary mutations were caught by assertions: removed collector shedding, ignored budget exhaustion, duplicate top, lost reserved probes, restored default ceiling and removed opt-in ceiling (`/tmp/pair514-mutations.log`). Real `make test-perf-capture test-make-env` passed: shape checks intact, live elapsed 6/6 s, and all 166 derived environment knobs retained while session identity was scrubbed. Full-suite evidence is still pending at the machine gate.

## Revisions

### 2026-10-13 — inherited #538 privacy assertion

Main was red after #538: `TestSkillBroadcastPrivacyContract/explicit_grant` expected the old exact sentence, while the shipped guidance correctly describes v3 per-note privacy. The test and guidance are identical to origin/main; a focused run reproduces the failure (`/tmp/pair514-inherited-privacy.log`). Hub receipt 47bf5cb5-e163-406b-8ded-3b42ee99ee64 authorizes folding this test-only repair into #514. Replace the stale assertion and additionally pin per-note selection, legacy submission-level fallback and aggregate privacy never granting private-note sharing. No privacy behavior or guidance changes.

- Full `make -k test` completed (`/tmp/pair514-full-suite.log`): all shell/editor targets and Go packages passed except the pre-fix Couch privacy assertion from #538. The complete `go test ./cmd/internal/couchcmd -count=1` rerun passes after the assertion repair (163.234 s, `/tmp/pair514-couchcmd-green.log`); no other full-suite failure. Couch core passed in 439.254 s. This is a completed full run plus the repaired package rerun, not a claim that the original invocation exited zero. Remaining checkboxes include the close/CI/landing gates themselves.
