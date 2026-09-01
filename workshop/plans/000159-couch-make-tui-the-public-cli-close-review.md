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

---

## Re-review — 2026-09-01T11:20:57-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 159 — couch: make TUI the public CLI |
| repo | pair |
| issue file | workshop/issues/000159-couch-make-tui-the-public-cli.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0088ac064c29e835a46b613712f86980ad072f1d..f9d61ae94a421146a5a4e58643d9e080d4e88917 |
| command | sdlc close --issue 159 |
| reviewer | codex |
| timestamp | 2026-09-01T11:20:57-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The core TUI-first projection is cleanly structured, documentation is updated, both prior findings are addressed, and the full test suite passes. However, the parser accepts flag-shaped values after `--show`, contradicting the Spec’s closed grammar. The installed smoke also does not enforce the exact Pair invocation promised by the plan.

### 1. Strengths

- `ParseCLI` is a small, deterministic IO-free parser, while runtime initialization remains behind classification and terminal validation ([cli.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/cli.go:31), [run.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/run.go:149)).
- Operation presentation is fail-closed through a non-authorizing zero value and exhaustive registry expectations ([ops.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/ops.go:78)).
- Non-terminal launch refusal occurs before namespace, policy, store, lease, or actor effects ([run.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/run.go:159)).
- README and atlas both reflect the new TUI-first public surface.
- The obsolete-argv audit now scans all Go sources under `cmd`, including tests, with line-local negative-fixture allowances ([readme_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/readme_test.go:190)).

### 2. Critical findings

- [cli.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/cli.go:49): `--show` accepts any non-empty second token, including `--list`, `--help`, and other flag-shaped values. The Spec says public flag forms cannot be combined with another flag. Reject flag-shaped references here and add regressions for `--show --list`, `--show --help`, and `--show --unknown`. This is an `ARCH-PURPOSE` contract violation.

### 3. Important findings

- [main_test.go](/Users/xianxu/workspace/pair/cmd/couch/main_test.go:94): the installed smoke waits only for the prefix `pair resume ` and later asserts merely that some `pair ` call exists. It would pass if the tag or required `--layout2` argument disappeared. The plan and its revision promise an exact process-boundary assertion. Parse the recorded call and require exactly `pair resume <generated-tag> --layout2`.

### 4. Minor findings

None.

### 5. Test coverage notes

- Focused package suite passed.
- `go test -p 20 ./... -count=1` passed.
- `git diff --check` passed.
- Parser tests cover most valid and malformed forms but omit flag-shaped `--show` references.
- The installed PTY smoke covers launch, cancellation/reaping, and pre-effect pipe refusal, but its child-argv assertion is too weak.

### 6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass. Registry presentation is the single authority for operation exposure.
- `ARCH-PURE`: Pass. Classification remains pure and runtime/terminal work stays in the shell.
- `ARCH-PURPOSE`: Flagged. The public grammar is not fully enforced for `--show`.
- `ARCH-MOCK`: Pass with a test-strength caveat. Production and test flows share the process seam and stateful call log, but exact child argv must be asserted.
- `ARCH-CONSTRAINTS`: Pass. Parsing is bounded and linear; terminal refusal precedes expensive effects; subprocess teardown is bounded.

### 7. Plan revision recommendations

None. The plan already states the intended exact grammar and installed-call assertion; implementation and tests need to conform to it.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The Core concepts table, explanatory prose, and delivered symbol now consistently name package-private cliInvocation.
  - id: BR-2
    disposition: addressed
    note: |
      Command tests now use typed-operation or explicit launch helpers, and the obsolete-argv audit scans all Go tests with line-local rejection-fixture allowances.
findings:
  - id: new
    severity: Critical
    family: closed-public-argv-grammar
    title: |
      --show accepts another public flag as its reference
    detail: |
      ParseCLI accepts any non-empty second token after --show, so forms such as --show --list and --show --help succeed despite the Spec requiring every public flag form to reject combination with another flag. Reject flag-shaped references and pin the full public-flag class with parser tests. This violates ARCH-PURPOSE.
  - id: new
    severity: Important
    family: integration-smoke-observable-contract
    title: |
      Installed smoke does not enforce the promised exact Pair invocation
    detail: |
      The smoke recognizes only the pair resume prefix and finally checks only for pair followed by a space, so it stays green if the generated tag or required --layout2 argument disappears. Assert the exact recorded pair resume <tag> --layout2 call at the process seam.
```

---

## Re-review — 2026-09-01T11:28:57-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 159 — couch: make TUI the public CLI |
| repo | pair |
| issue file | workshop/issues/000159-couch-make-tui-the-public-cli.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0088ac064c29e835a46b613712f86980ad072f1d..def48f1bf8ebaaf16ae7a6182d63e6aa25441e66 |
| command | sdlc close --issue 159 |
| reviewer | codex |
| timestamp | 2026-09-01T11:28:57-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The boundary fulfills the TUI-first CLI specification, both open prior findings are addressed with reachable regressions, documentation and atlas changes cover the new surface, and focused plus race verification passed. No blocking or advisory findings remain.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      ParseCLI rejects flag-shaped show references, and parser regressions enumerate the public reserved-flag class.
  - id: BR-4
    disposition: addressed
    note: |
      The installed smoke requires exactly one pair resume <16-hex Couch tag> --layout2 invocation.
```

1. Strengths

- The closed parser rejects malformed combinations before runtime initialization ([cli.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/cli.go:31)).
- BR-3 is pinned by explicit `--show --list`, `--show --help`, and unknown-flag cases ([cli_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/cli_test.go:37)); removing the flag-prefix guard makes these cases fail.
- BR-4 is pinned at the real process seam with exact command shape, one-call cardinality, hexadecimal tag validation, and mandatory `--layout2` ([main_test.go](/Users/xianxu/workspace/pair/cmd/couch/main_test.go:142)).
- Operation presentation is registry-owned, exhaustive, and fail-safe at zero ([ops.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/ops.go:78)).
- README and `atlas/couch.md` were updated in the reviewed range.

2. Critical findings

None.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

Passed:

- Focused command, core, TTY, dispatcher, entrypoint, runtime-bundle, and artifact tests.
- Race tests for `couchcmd`, `couchcore`, and `couchtty`.
- Installed-command PTY smoke.
- `git diff --check`.

The Core concepts table matches the delivered symbols and paths. PURE parser and presentation tests require no filesystem, subprocess, terminal, or mocks. Integration behavior uses the existing injected runtime/fake seams.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass — argv classification and operation presentation each have one authority.
- `ARCH-PURE`: Pass — parsing remains pure; terminal/runtime effects stay in the shell.
- `ARCH-PURPOSE`: Pass — public argv, internal protocol, TUI-only operations, tests, README, and atlas all follow the declared projection.
- `ARCH-MOCK`: Pass — the installed smoke exercises production process boundaries with stateful call logs and exact argv assertions.
- `ARCH-CONSTRAINTS`: Pass — parsing is bounded and effect-free; terminal refusal precedes runtime effects, while PTY teardown is bounded.

7. Plan revision recommendations

None.
