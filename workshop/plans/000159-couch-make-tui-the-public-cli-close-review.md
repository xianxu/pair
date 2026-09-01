# Boundary Review — pair#159 (whole-issue close)

| field | value |
|-------|-------|
| issue | 159 — couch: make TUI the public CLI |
| repo | pair |
| issue file | workshop/issues/000159-couch-make-tui-the-public-cli.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0088ac064c29e835a46b613712f86980ad072f1d..a7061a3c32550293d30f646985ee79e6cd3d3903 |
| command | sdlc close --issue 159 |
| reviewer | codex |
| timestamp | 2026-09-01T11:08:20-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The production CLI implementation largely matches the specification: parsing is pure and precedes runtime effects, public reachability is closed, hidden publication is registry-authorized, and focused plus race tests pass. Two traceability problems block the boundary: the Core concepts table names an entity that does not exist, and the claimed current-source migration excludes tests while `run_test.go` retains a legacy command interpreter and obsolete command contracts.

## Strengths

- `ParseCLI` implements the closed argv grammar without IO and returns no partial invocation on errors ([cli.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/cli.go:26)).
- `RunWithRuntime` parses and validates terminal ownership before namespace, policy, store, lease, or actor initialization ([run.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/run.go:149)).
- Every operation has an explicit, nonzero presentation, independently pinned by the declaration closure test ([ops.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/ops.go:78)).
- The installed-command test exercises the compiled executable through a PTY and uses stateful fake process boundaries with bounded reap and reader joins ([main_test.go](/Users/xianxu/workspace/pair/cmd/couch/main_test.go:18)).
- README and atlas both changed in-range and accurately describe the new public and hidden surfaces.

## Critical findings

- [Plan line 19](/Users/xianxu/workspace/pair/workshop/plans/000159-couch-make-tui-the-public-cli-plan.md:19) — The Core concepts table declares `CLIInvocation`, but the delivered entity is the unexported `cliInvocation` at [cli.go:22](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/cli.go:22). The review contract explicitly requires each table entity to exist under its greppable name. Append a plan revision recording the encapsulation decision and change the table and relationship prose to `cliInvocation`; alternatively export the implemented type if that was truly intended. (`ARCH-PURPOSE`)

## Important findings

- [readme_test.go:185](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/readme_test.go:185), [run_test.go:352](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/run_test.go:352) — The current-source sweep deliberately excludes every `_test.go`, while the plan claims tests were migrated. `runRT` still implements a test-only legacy argv interpreter, reconstructs the removed `start path --no-console --agent` schema, and multiple current tests/comments continue to express removed shell contracts. This leaves the enumerable test-source class unswept and contradicts the checked Plan claim that orchestration tests use the private executor without deprecated argv. Replace legacy argv interpretation with an explicitly typed operation/runtime helper, retarget or rename tests that protect still-valid internal seams, and include test sources in the shadow sweep with narrow allowlists only for parser rejection fixtures. (`ARCH-PURPOSE`)

## Minor findings

None.

## Test coverage notes

Executed successfully at the pinned clean HEAD:

- Focused Couch, dispatcher, entrypoint, runtime-bundle, and artifact suites.
- Race suites for `couchcmd`, `couchcore`, and `couchtty`.
- `git diff --check`.

Parser tests are pure. Runtime tests use injected stateful fakes, and the installed smoke crosses the real compiled-command/PTY seam. No prior findings required disposition.

## Architectural notes for upcoming work

- `ARCH-DRY`: Pass. Registry presentation is the single reachability classification; parser and documentation checks derive from it.
- `ARCH-PURE`: Pass. Argument classification is isolated from runtime IO, and runtime orchestration remains a thin injected shell.
- `ARCH-PURPOSE`: Flagged. Production fulfills the TUI-first purpose, but the plan/entity traceability and test-source migration are incomplete.
- `ARCH-MOCK`: Pass. External `sdlc` and Pair interactions use production seams with stateful call logs in the installed smoke.
- `ARCH-CONSTRAINTS`: Pass. Parsing is bounded and linear; launch performs terminal validation before expensive work; process teardown is bounded.

## Plan revision recommendations

Append a `## Revisions` entry stating:

- `CLIInvocation` was implemented as package-private `cliInvocation`; update the Core concepts table and prose accordingly.
- The migration scope now includes current `_test.go` sources; legacy argv-shaped helpers are replaced by typed in-process helpers, with obsolete forms retained only as explicit parser rejection fixtures.

```findings
findings:
  - id: new
    severity: Critical
    family: plan-code-entity-traceability
    title: |
      Core concepts names CLIInvocation, but only cliInvocation exists
    detail: |
      The greppable entity declared at plan line 19 is absent. Append a plan revision and name the delivered package-private entity, or export the implementation if that was intended.
  - id: new
    severity: Important
    family: current-source-shadow-sweep
    title: |
      The obsolete-argv sweep excludes tests that retain a legacy command interpreter
    detail: |
      TestNoCurrentSourcesAdvertiseObsoleteCouchArgv skips all _test.go files, while runRT reconstructs the removed start argv schema and current tests still express obsolete command contracts. Migrate these tests to a typed private-operation helper and sweep test sources with narrow rejection-fixture allowlists.
```
