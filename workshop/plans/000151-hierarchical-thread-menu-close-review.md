# Boundary Review — pair#151 (whole-issue close)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9157998f17a19d3794efe5f5f0cb376752674df4..309a9717aa036740ca71676a244a78c6257aa63d |
| command | sdlc close --issue 151 |
| reviewer | codex |
| timestamp | 2026-08-31T18:17:17-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range fulfills issue #151’s Spec and Plan. The hierarchical switcher, actionable lifecycle projection, asynchronous Console integration, exact completion correlation, bounded scheduling, documentation, and executable architectural ledger are present and well tested. No Critical, Important, or Minor findings remain.

### 1. Strengths

- Lifecycle projection fails closed on malformed, ambiguous, or insufficient live/parked evidence ([actionableinventory.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/actionableinventory.go:65)).
- Reducer state centralizes refresh reconciliation, generation authorization, frame validity, and correlated completions ([menu.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:244)).
- Inventory and preview I/O use bounded, context-cancelable workers outside the Console mutex ([console_menu.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console_menu.go:79)).
- Post-mutation projection remains visibly pending until a later refresh generation succeeds ([menu.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:271), [console.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:1062)).
- README and atlas changes accompany the new user-facing and architectural surfaces; the obsolete panel authority is deleted.

### 2. Critical findings

None.

### 3. Important findings

None.

### 4. Minor findings

None.

### 5. Test coverage notes

- `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchcmd -run 'TestIssue151|TestM3Docs' -count=1` passed.
- Focused Console/menu performance, refresh, operation, and preview tests passed.
- Race tests passed for `couchcore`, `couchtty`, `couchcmd`, and `sessioninventory`.
- `git diff --check` passed.
- The repository-wide suite passed except `cmd/pair-go`, whose tests could not execute sandbox-blocked `/bin/ps`; this is environmental and outside the reviewed change.
- Tests cover exact proof validation, stale result rejection, generation-bound refreshes, cancellation, frame reconciliation, terminal input reachability, rendering bounds, and production Console paths.

### 6. Architectural notes

- `ARCH-DRY`: Pass — lifecycle projection, reducer transitions, operation policy, and schedulers each have one authority; the competing flat panel was removed.
- `ARCH-PURE`: Pass — projection, reducer, layout, and scheduling remain pure; Console owns the thin I/O and concurrency shell.
- `ARCH-PURPOSE`: Pass — the delivered surface covers the full actionable-thread hierarchy, shared operations, lifecycle proof, refresh behavior, and start workflow.
- `ARCH-MOCK`: Pass — production seams are exercised through stateful runner, child, lifecycle, and terminal doubles.
- `ARCH-CONSTRAINTS`: Pass — concurrency is bounded, cancellation is explicit, the 100-row workload is exercised, and target-machine evidence is documented.

### 7. Plan revision recommendations

None; the Core concepts ledger and checklist match the pinned implementation.

```findings
```
