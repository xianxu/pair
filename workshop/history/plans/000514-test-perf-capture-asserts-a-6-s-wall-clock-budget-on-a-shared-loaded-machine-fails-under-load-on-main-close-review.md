# Boundary Review — pair#514 (whole-issue close)

| field | value |
|-------|-------|
| issue | 514 — test-perf-capture asserts a 6 s wall-clock budget on a shared loaded machine; fails under load on main |
| repo | pair |
| issue file | workshop/issues/000514-test-perf-capture-asserts-a-6-s-wall-clock-budget-on-a-shared-loaded-machine-fails-under-load-on-main.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1ef66ff23734f63b866852c147cbe78f4c356704..89d38a9738b732431c815845a2e6b4a3a5933ba5 |
| command | sdlc close --issue 514 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T18:51:42-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The change meets the issue’s purpose: default capture tests retain validation without enforcing a shared-host timing ceiling, while the opt-in lane enforces it. Production capture behavior is unchanged. No blocking findings; broader validation is limited by missing generated runtime assets in this checkout.

1. **Strengths**
   - Controlled fixtures execute the actual capture script and check both output and collector/probe calls.
   - Gate tests cover identical overruns in both modes, equality, under-budget values, and malformed timing.
   - Privacy assertions match the shipped per-note guidance, including legacy fallback and aggregate non-authority.
   - Atlas documentation accurately distinguishes cooperative work shedding from subprocess deadlines.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - All nine Python policy tests passed.
   - Existing shell shape/parser tests passed.
   - Opt-in single-capture check passed: 3 seconds against 6.
   - Privacy test passed when run directly with its embedding source.
   - Package-wide Go testing could not start because generated runtime assets were absent. No full-suite rerun; the missing local binary also leaves the shell’s conditional failing-probe check unexercised here.
   - Pinned diff checks passed.

6. **Architecture**
   - **ARCH-DRY: PASS** — reuses the existing opt-in convention and centralizes fixture helpers.
   - **ARCH-PURE: PASS** — integration behavior is exercised through controlled clock/tool seams; no new production logic mixes concerns.
   - **ARCH-PURPOSE: PASS** — addresses timing flakiness while retaining deterministic scheduling-policy checks.
   - Atlas is updated. The new flag is internal test plumbing, so no product README update is needed.

7. **Plan revision recommendations:** None. Outstanding close/CI checkboxes appropriately remain pending.

```findings
{}
```
