# Boundary Review — pair#151 (milestone M3)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 0c40a8d1880b49a9cac1a7f4d8cd24a2c713dba7..542c5d58411e6db126048360676c479cad841df6 |
| command | sdlc milestone-close --issue 151 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-31T15:42:38-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The hierarchical switcher is reachable, well-separated, documented, and broadly tested, but M3 cannot close yet. Production never supplies the operation-result inventory that the reducer tests assume, leaving successful mutations temporarily—or after refresh failure, indefinitely—represented by stale rows. The superseded flat-panel authority also remains despite the plan declaring its deletion. Finally, the latency harness passes but bypasses the real run-loop boundary it claims to measure.

```findings
findings:
  - id: new
    severity: Critical
    family: shared-operation-consumer-sweep
    title: |
      Successful operation projections are exercised only in reducer tests and never supplied by production
    detail: |
      This is the 3rd finding in family `shared-operation-consumer-sweep`. Earlier rounds fixed instances. Do NOT fix this instance alone: enumerate every operation result and state the shared rule for projecting or visibly deferring its committed state. `finishOperation` extracts only addresses from park/start results and never sets `InventorySet`; every production search hit for `InventorySet: true` is in tests. Start, park, name, and describe therefore repaint stale inventory before refresh, and a failed refresh can leave that stale view indefinitely rather than using returned state or remaining visibly refresh-pending.
  - id: new
    severity: Critical
    family: superseded-ui-authority-retirement
    title: |
      The planned deleted flat-panel authority remains as a parallel implementation
    detail: |
      The Core concepts row declares `PanelModel` deleted in M3 and its narrative says `MenuState` replaces it, but `panel.go`, legacy Console state, resolver/summaries callbacks, prompt state, and the old panel/action controller remain. The executable contract was changed to require this compatibility adapter rather than proving its removal. Delete the superseded authority and make the contract fail while any declared legacy symbols or production fields remain.
  - id: new
    severity: Important
    family: lifecycle-evidence-validation
    title: |
      Target latency evidence bypasses the Console run-loop boundary it claims to measure
    detail: |
      This is the 2nd finding in family `lifecycle-evidence-validation`. Earlier rounds fixed instances. Do NOT patch one timing label: state the evidence rule and apply it to every measured path. The harness directly calls `showMenu`, `onMenuInput`, and `finishMenuRefresh`, then times function return; it never sends raw bytes through the host input channel, runs `Console.Run`, or correlates a generation-specific repaint. Delays or misrouting in the actual select loop would not fail these measurements.
```

### Strengths

- `ProjectActionableThreads` fails closed on exact live-process evidence and exact resumable parked evidence ([actionableinventory.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/actionableinventory.go:67)).
- Refresh and preview controllers enforce bounded single-flight/coalescing behavior with cancellation and joined workers ([console_menu.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console_menu.go:79)).
- Operation results are correlated to exact attempt, address, frame instance, and depth before reducer mutation ([menu.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:1017)).
- Transactional attach failure routes through exact-actor cleanup rather than leaking a started process ([run.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/run.go:442)).
- README and atlas changes accurately describe the reachable hierarchical controls and actionable/raw inventory distinction.

### Critical findings

1. ARCH-PURPOSE: Production does not consume operation-returned state.

   At [console.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:1449), `finishOperation` derives addresses but constructs no projected inventory. The reducer’s `InventorySet` path is tested extensively, yet has zero production setters. Establish one exhaustive result-consumption rule covering start, park, resume, name, describe, switch, and leave. Add Console-level tests proving each returned mutation is reflected without waiting for refresh and that refresh failure cannot leave an unmarked stale view.

2. ARCH-DRY / ARCH-PURPOSE: The compatibility panel was not retired.

   The plan declares deletion, while [console.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:77), [panel.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/panel.go:60), and the legacy controller beginning at [console.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:1059) retain the competing state and behavior. Remove the full old surface and invert the concept contract so retention fails tests.

### Important findings

1. ARCH-CONSTRAINTS: The target performance harness does not exercise its declared boundary.

   [menu_perf_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu_perf_test.go:143) constructs a Console but directly invokes internal methods; [measureMenuPath](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu_perf_test.go:208) merely times those calls. Drive the running Console through raw fake-host input/result channels and stop only on the matching generation-tagged repaint.

### Minor findings

None.

### Test coverage notes

- `go test -p 20 ./... -count=1`: PASS.
- `git diff --check <base> <head>`: PASS.
- `BenchmarkMenu100`: PASS; approximately 51–61 µs/op across paths on the current M2 Max.
- Opt-in target test: PASS, but its boundary is invalid for the reason above.
- No prior findings required disposition in this first M3 review round.

### Architectural notes

- ARCH-DRY: Flagged—the legacy panel duplicates the new menu authority.
- ARCH-PURE: Pass—the reducer, renderer, and schedulers remain pure; Console is the thin asynchronous shell.
- ARCH-PURPOSE: Flagged—production omits the reducer’s operation-result projection contract, and planned compatibility retirement is incomplete.
- ARCH-MOCK: Pass—the runner, terminal host, lifecycle, and resolver paths use shared injected/stateful seams.
- ARCH-CONSTRAINTS: Flagged—the implementation is bounded, but the target measurement does not validate the declared real interaction path.

### Plan revision recommendations

Append revisions that:

- Record complete retirement of `PanelModel`, legacy Console fields/controllers, and compatibility contract expectations.
- Enumerate every operation-result consumer and define the shared projection-or-visible-pending rule, including refresh failure.
- Replace the latency evidence with raw-host, running-loop, generation-correlated measurements and update the issue evidence after rerunning it.
