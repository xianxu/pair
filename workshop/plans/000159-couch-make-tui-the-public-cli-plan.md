# Couch TUI-First CLI Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `couch [path]` open the Couch TUI while reducing the public command surface to `--list`, `--show`, and help, with only `publish-description` retained behind a hidden machine protocol.

**Architecture:** A pure argv parser produces one closed invocation variant before any runtime work. The typed operation registry gains an explicit presentation classification, so every operation has exactly one public, hidden, or in-process home; the runtime shell maps accepted variants into the existing dispatcher without creating a second domain path.

**Tech Stack:** Go, `cmd/internal/couchcmd`, `cmd/internal/couchcore`, injected Couch fakes, `creack/pty`, README/atlas contract tests.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| `CLIInvocation` closed variants | `cmd/internal/couchcmd/cli.go` | new |
| `ParseCLI` | `cmd/internal/couchcmd/cli.go` | new |
| `OperationPresentation` | `cmd/internal/couchcore/ops.go` | new |
| `Operation.Presentation` | `cmd/internal/couchcore/ops.go` | modified |

- **`CLIInvocation`** — one of launch, list, show, internal, or help; parse failure is an error rather than a partial invocation.
  - **Relationships:** One argv vector produces at most one invocation. Launch owns one path, show owns one reference, and internal owns one whitelisted typed-operation request.
  - **DRY rationale:** Public precedence lives in one parser instead of being split between execution, generic operation binding, help, and tests.
  - **Future extensions:** A new public mode requires a new closed variant; adding an operation cannot expose it accidentally.

- **`ParseCLI`** — pure exact-grammar parser for the spec's argv table.
  - **Relationships:** Consumed 1:1 by `RunWithRuntime`; it consults operation presentation only to validate the hidden operation.
  - **DRY rationale:** Table-driven tests and production share one precedence decision.
  - **Future extensions:** None anticipated; keep the grammar narrow.

- **`OperationPresentation` / `Operation.Presentation`** — registry-owned classification of each typed operation as TUI-only, public list, public show, or hidden process protocol.
  - **Relationships:** Every `Operation` has one nonzero presentation; only `publish-description` is hidden-process, only `list/show` are public diagnostics, and all remaining operations are in-process.
  - **DRY rationale:** Adding an operation cannot silently expose argv or escape classification.
  - **Future extensions:** A new process boundary must explicitly widen this classification and its closed test.

Pure tests in `cli_test.go` and `ops_declarations_test.go` use no filesystem, subprocess, terminal, or runtime mock (`ARCH-PURE`, `ARCH-DRY`, `ARCH-PURPOSE`).

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| `RunWithRuntime` CLI projection | `cmd/internal/couchcmd/run.go` | modified | argv, terminal ownership, Couch runtime |
| Couch executable smoke | `cmd/couch/main_test.go` | new | compiled binary, PTY, fake `sdlc`/`pair` executables |
| Operator documentation | `README.md`, `atlas/couch.md` | modified | public and architectural contracts |

- **`RunWithRuntime` CLI projection** — parses before namespace/policy/store work, refuses non-terminal launch before effects, and delegates accepted variants into the typed dispatcher.
  - **Injected into:** Existing `testRT`, fake policy/store/runner, and console helpers. Launch orchestration is extracted from the old generic CLI branch so tests can exercise it without re-exposing a command.
  - **Future extensions:** Owner routing may later add transport beneath typed dispatch; it must not widen the public parser.

- **Couch executable smoke** — builds `cmd/couch`, runs bare invocation on a PTY with a temporary Couch store and stateful fake executables, observes the fake Pair marker, then cancels and reaps the process.
  - **Injected into:** The command receives temporary PATH/store/home and fake fleet-policy/Pair behavior; production uses the same OS seams.
  - **Future extensions:** It can later assert packaged installation without changing parser tests.

- **Operator documentation** — describes Couch as a TUI and maps the hidden agent hook in atlas only.
  - **Injected into:** README contract tests derive coverage from `Operation.Presentation`, replacing the rule that every operation becomes a command.
  - **Future extensions:** New presentations fail the audit until assigned a documentation home.

Runtime constraints: `ParseCLI` is O(argv bytes), performs no IO, and allocates only invocation/error data. Launch performs existing work only after terminal validation; diagnostics do not acquire the live-owner lease. Tests never exceed `go test -p 20` (`ARCH-CONSTRAINTS`).

## Chunk 1: Closed CLI projection and migration

### Task 0: Pin the installed bare-command regression before implementation

**Files:**
- Create: `cmd/couch/main_test.go`

- [ ] **Step 1: Write the installed-command smoke and exact teardown**

Build `./cmd/couch` into `t.TempDir()`. Create executable fake `sdlc` and
`pair` programs in a temporary PATH. Resolve the test repo's physical Git
directory and make fake `sdlc fleet policy --path P --json` emit exactly:

```json
{"ok":true,"value":{"policy_version":1,"policy_digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","repo_identity":"<physical-git-dir>","admission_key":"<physical-git-dir>","capacity":{"kind":"unbounded"}}}
```

The fake validates argv, appends one call record, and fails otherwise. Fake
Pair appends argv, prints a unique marker, and exits immediately. Start bare
Couch with `CommandContext` through a PTY and temporary `HOME`,
`XDG_DATA_HOME`, and `COUCH_STORE_DIR`.

Use exactly one buffered `waitResult chan error`; one goroutine calls
`cmd.Wait()` once and sends once. A second goroutine reads the PTY into a
buffer, signals the marker on a buffered channel, and signals reader completion
on another buffered channel. After marker or failure, close the PTY master to
unblock the reader. Select on the one wait-result channel with a deadline; on
timeout call the context cancel function and select on the same channel with a
second deadline; on another timeout call `cmd.Process.Kill()` and select on the
same channel with a final deadline. A final timeout is a teardown failure. Join
the reader with its own bounded select. Never call `Wait` twice or receive
unboundedly. Assert both fake logs and that Couch was reaped; the Pair fake has
already exited and cannot leak.

Add the piped variant: bare invocation must reject non-terminal launch before
either fake records a call.

- [ ] **Step 2: Run the installed smoke and observe RED**

```bash
go test -p 20 ./cmd/couch -run '^TestBareCouchInstalledCommand' -count=1 -v
```

Expected: FAIL because current bare Couch prints help and never invokes fake
`sdlc`/Pair. This proves the smoke distinguishes the original behavior.

- [ ] **Step 3: Commit the red regression with the implementation series only after it turns green**

Leave the failing test uncommitted while Tasks 1–3 implement the behavior; it
must be included in Task 3's green commit, not bypassed or skipped.

### Task 1: Add the registry-owned operation presentation

**Files:**
- Modify: `cmd/internal/couchcore/ops.go`
- Modify: `cmd/internal/couchcore/ops_declarations_test.go`
- Modify: `cmd/internal/couchcore/ops_test.go`
- Modify: `cmd/internal/couchcore/operationdispatch_test.go`
- Modify: `cmd/internal/couchtty/menu.go`
- Modify: `cmd/internal/couchtty/menu_async_test.go`
- Modify: `cmd/internal/couchtty/console_menu_operation_test.go`

- [ ] **Step 1: Write the failing exhaustive projection test**

Add an independent expected table covering all thirteen operations:

```go
want := map[string]couchcore.OperationPresentation{
    "list": couchcore.PresentationPublicList,
    "show": couchcore.PresentationPublicShow,
    "publish-description": couchcore.PresentationInternal,
    "prepare-start": couchcore.PresentationTUI,
    "start": couchcore.PresentationTUI,
    "stop": couchcore.PresentationTUI,
    "name": couchcore.PresentationTUI,
    "describe": couchcore.PresentationTUI,
    "switch": couchcore.PresentationTUI,
    "attach": couchcore.PresentationTUI,
    "park": couchcore.PresentationTUI,
    "leave": couchcore.PresentationTUI,
    "resume": couchcore.PresentationTUI,
}
```

Assert exact key-set equality, nonzero presentation, and exactly one internal operation. Update the independent `start` arity expectation from four arguments to its owner-local `token` only; launch path belongs to `prepare-start`, while `--agent` and `--no-console` are removed. Replace `operationdispatch_test.go`'s stale `start{path,agent}` empty-agent case with the remaining value-required `prepare-start{agent:""}` case, and add a rejection proving `start` refuses undeclared `path`/`agent` arguments before its executor.

- [ ] **Step 2: Run the focused test and observe RED**

```bash
go test -p 20 ./cmd/internal/couchcore -run 'Test(OperationDeclarations|OperationArity|OperationPresentation)' -count=1
```

Expected: FAIL because `OperationPresentation` and the field do not exist.

- [ ] **Step 3: Add the enum and classify every declaration**

Add a non-authorizing zero value plus `PresentationTUI`, `PresentationPublicList`, `PresentationPublicShow`, and `PresentationInternal`. Put `Presentation OperationPresentation` on `Operation`, assign every row, and remove `path`, `agent`, and `no-console` from `start.Args`; retain its implicit required token. Update `startMenuEffect` to send only the accepted preview token: path/agent were inputs to `prepare-start` and are already fingerprinted by that token. Update the async/effect/console-operation tests to prove prepare receives path/agent while start receives token only.

- [ ] **Step 4: Run the package and observe GREEN**

```bash
go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchtty -run 'Test(Operation|Start|MenuPreview)' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/couchcore/ops.go cmd/internal/couchcore/ops_declarations_test.go cmd/internal/couchcore/ops_test.go cmd/internal/couchcore/operationdispatch_test.go cmd/internal/couchtty/menu.go cmd/internal/couchtty/menu_async_test.go cmd/internal/couchtty/console_menu_operation_test.go
git commit -m "#159: classify Couch operation presentation"
```

### Task 2: Implement the pure public argv grammar

**Files:**
- Create: `cmd/internal/couchcmd/cli.go`
- Create: `cmd/internal/couchcmd/cli_test.go`

- [ ] **Step 1: Write table-driven valid-form tests**

Cover `[]`, `["/repo"]`, `["--", "--repo"]`, `--list`, `--show ref`, `--help`, `-h`, and `--internal publish-description text`, including explicit empty text. Assert exact variant and payload.

- [ ] **Step 2: Write table-driven malformed-form tests**

Cover empty path; missing/extra `--show`; combined or repeated flags; flags after a path; help mixed with another token; bare `--`; extra paths; unknown single-dash and double-dash flags; `--internal`; `--internal=...`; unknown, TUI, or public operations behind `--internal`; missing/extra internal args; and `--` within internal args. Assert error and no invocation. Pin that a dash-prefixed path succeeds only through `couch -- <path>`.

- [ ] **Step 3: Run parser tests and observe RED**

```bash
go test -p 20 ./cmd/internal/couchcmd -run '^TestParseCLI' -count=1
```

Expected: FAIL because `ParseCLI` is undefined.

- [ ] **Step 4: Implement the minimal parser**

Use an unexported invocation-kind enum and payload struct. `ParseCLI(args, operations)` performs no IO. For `--internal`, resolve through declarations and require `PresentationInternal`; do not maintain a second whitelist. Return usage-class errors without printing help.

- [ ] **Step 5: Run parser tests and observe GREEN**

```bash
go test -p 20 ./cmd/internal/couchcmd -run '^TestParseCLI' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/internal/couchcmd/cli.go cmd/internal/couchcmd/cli_test.go
git commit -m "#159: parse the Couch TUI-first CLI"
```

### Task 3: Route public invocations without exposing operations

**Files:**
- Modify: `cmd/internal/couchcmd/run.go`
- Modify: `cmd/internal/couchcmd/run_test.go`

- [ ] **Step 1: Replace old surface tests with failing public-contract tests**

Pin all of these behaviors:

- help contains `couch [path]`, `--list`, and `--show`, but no `start`, lifecycle operation, `publish-description`, or `--internal`;
- parser tests prove bare/path argv selects launch; the existing `consoleRunnerFor` seam proves a true terminal selects Console/PtyRunner; the previously red installed PTY smoke from Task 0 proves the composed production path reaches prepare-start/start;
- bare/path launch without a terminal fails before namespace, policy, store, lease, or runner effects;
- `--list` stays global and `--show` derives scope only from CWD;
- `--internal publish-description` uses composite thread environment and supports empty clearing;
- old operation argv spellings are rejected or treated solely as paths;
- parse errors return 2 without `ResolveNamespace`.

Retain domain tests by calling the extracted private launch/operation executor instead of deprecated argv. Remove tests for public `--agent`, `--no-console`, generic operation help, and generic binding where that behavior no longer exists.

- [ ] **Step 2: Run focused tests and observe RED**

```bash
go test -p 20 ./cmd/internal/couchcmd -run 'Test(Public|Bare|Path|Help|List|Show|Internal|ParseError|NonTerminal)' -count=1
```

Expected: FAIL because zero args still select help and the registry is exposed as commands.

- [ ] **Step 3: Split parsing from execution**

Refactor `RunWithRuntime` into: `ParseCLI` before runtime methods; fixed public help; an explicit `terminalFiles(stdin, stdout) (in, out *os.File, ok bool)` decision made before any launch runtime method; `runLaunch(path, console, runner, ...)` dispatching `prepare-start{path}` then implicit token-bound `start`; public list/show and hidden publish-description through `DispatchOperation`; and existing result rendering. Production feeds `terminalFiles` into the existing `consoleRunnerFor` seam. Tests use buffers to prove `ok=false` has zero runtime effects and `pty.Open` plus `consoleRunnerFor(..., true, slave, slave)` to pin Console/PtyRunner wiring; no new ambient Runtime method or package-global override is introduced. Keep the domain executor private for focused orchestration tests. Delete operation-enumerating usage and argv reachability audits. Update capacity guidance to `couch --list`.

- [ ] **Step 4: Run Couch command tests and observe GREEN**

```bash
go test -p 20 ./cmd/internal/couchcmd -count=1
```

Expected: PASS.

- [ ] **Step 5: Run affected integration tests**

```bash
go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/internal/couchcmd/run.go cmd/internal/couchcmd/run_test.go
git add cmd/couch/main_test.go
git commit -m "#159: make Couch launch the default mode"
```

### Task 4: Migrate documentation and enforce the projection

**Files:**
- Modify: `README.md`
- Modify: `atlas/couch.md`
- Modify: `workshop/projects/couch.md`
- Modify: `workshop/issues/000153-couch-managed-worktree-lifecycle.md`
- Modify: `probes/zellijpark/main.go`
- Modify: `cmd/internal/couchcmd/readme_test.go`
- Modify: `cmd/internal/couchcore/couch.go`
- Modify: `cmd/internal/couchcore/couch_test.go`
- Modify: `cmd/internal/couchcore/procops.go`
- Modify: `cmd/internal/couchcore/ptyrunner.go`
- Modify: `cmd/internal/couchcore/ptyrunner_test.go`
- Modify: `cmd/internal/couchcore/registry.go`
- Modify: `cmd/internal/couchcore/runner_fake.go`
- Modify: `cmd/internal/couchcore/store.go`
- Modify: `workshop/issues/000159-couch-make-tui-the-public-cli.md`

- [ ] **Step 1: Write failing projection/documentation tests**

Replace `TestREADMEDocumentsEveryOperation` with an exhaustive presentation audit: public diagnostics appear in README as flags; TUI-only operations never appear as commands and operator-visible ones have menu documentation; internal operations have an atlas protocol home but no README/help entry; and every enum value is handled. Update checks requiring `--no-console`, direct resume, `couch list/show`, or operation enumeration. Add `TestNoCurrentSourcesAdvertiseObsoleteCouchArgv`, scanning README, atlas, active issues/projects, probes, and tracked `cmd/**/*.go` production/test sources for old command spellings. Explicit removal discussions in #159 Revisions, parser rejection fixtures, and dated historical project scope events are the only allowlisted contexts; every current instruction, code comment, and caller must use the new forms.

- [ ] **Step 2: Run docs tests and observe RED**

```bash
go test -p 20 ./cmd/internal/couchcmd -run 'Test(README|M3Docs|OperationPresentationDocs|NoCurrentSources)' -count=1
```

Expected: FAIL against command-oriented docs.

- [ ] **Step 3: Rewrite user and architecture surfaces**

README documents only:

```text
couch                 open Couch in the current directory
couch <path>          open Couch for another repository/subdirectory
couch --list          diagnostic inventory
couch --show <ref>    diagnostic detail
```

Describe lifecycle actions as TUI operations and remove `--agent`, `--no-console`, direct lifecycle commands, and public `publish-description`. Atlas records hidden `couch --internal publish-description <text>`, presentation classification, and root launch's internal prepare-start/start composition. Sweep the active `workshop/projects/couch.md` portfolio: revise current guidance and append a dated scope event without falsifying historical milestone records. Update #153's ordinary-start wording and the zellij park probe's direct `couch stop` comments/labels. Rewrite current `couchcore`/`couchcmd` comments and test descriptions listed above to say Couch launch, `--list`, or TUI Park rather than teaching removed argv. Append the issue Log with the repository-wide `ARCH-PURPOSE` migration audit.

- [ ] **Step 4: Run docs and affected tests**

```bash
go test -p 20 ./cmd/internal/couchcmd ./cmd/internal/couchcore ./cmd/internal/couchtty -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add README.md atlas/couch.md workshop/projects/couch.md workshop/issues/000153-couch-managed-worktree-lifecycle.md probes/zellijpark/main.go cmd/internal/couchcmd/readme_test.go cmd/internal/couchcore/couch.go cmd/internal/couchcore/couch_test.go cmd/internal/couchcore/procops.go cmd/internal/couchcore/ptyrunner.go cmd/internal/couchcore/ptyrunner_test.go cmd/internal/couchcore/registry.go cmd/internal/couchcore/runner_fake.go cmd/internal/couchcore/store.go workshop/issues/000159-couch-make-tui-the-public-cli.md
git commit -m "#159: document Couch as a TUI"
```

### Task 5: Prove the installed command and close the boundary

**Files:**
- Modify: `cmd/couch/main_test.go`
- Modify: `cmd/internal/artifactpath/manifest.go`
- Modify: `cmd/internal/artifactpath/coverage_test.go`
- Modify: `workshop/issues/000159-couch-make-tui-the-public-cli.md`

- [ ] **Step 1: Re-run the previously red installed smoke and observe GREEN**

```bash
go test -p 20 ./cmd/couch -run '^TestBareCouchInstalledCommand' -count=1 -v
```

Expected: PASS. The same test failed in Task 0 before implementation.

- [ ] **Step 2: Update exhaustive artifact inventory**

Classify only the new production file `cmd/internal/couchcmd/cli.go` in `NonArtifactSources` or `SourceClassifications`, according to whether its literals intersect artifact vocabulary. Do not add `cli_test.go` or `main_test.go`: `productionSourceFile` deliberately excludes `_test.go` files. Preserve the existing single-source inventory rules.

- [ ] **Step 3: Run bounded verification**

```bash
go test -p 20 ./cmd/couch ./cmd/internal/couchcmd ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/dispatcher ./cmd/internal/entrypoint ./cmd/internal/runtimebundle ./cmd/internal/artifactpath -count=1
go test -p 20 -race ./cmd/internal/couchcmd ./cmd/internal/couchcore ./cmd/internal/couchtty -count=1
git diff --check
```

Expected: all tests PASS, installed smoke observes the fake Pair marker, race suite passes, and diff check is silent. Never exceed 20 Go test workers.

- [ ] **Step 4: Record evidence and stop at close**

Update issue Plan/Log with exact evidence, commit the smoke/inventory/evidence, then run `sdlc close --issue 159 --verified '<observed commands and results>'`. Close owns the fresh-context boundary review.

```bash
git add cmd/couch/main_test.go cmd/internal/artifactpath/manifest.go cmd/internal/artifactpath/coverage_test.go workshop/issues/000159-couch-make-tui-the-public-cli.md
git commit -m "#159: verify the Couch public entrypoint"
```

## Revisions

### 2026-09-01T10:13:00-07:00 — execute the current-source audit in RED

**Reason:** fourth plan review found that Task 4's focused command omitted the
new repository-wide obsolete-argv test by name.

**Delta:** the RED command now includes `TestNoCurrentSources...`, so the full
shadow sweep demonstrably fails before docs/comments migrate (`ARCH-PURPOSE`).

### 2026-09-01T10:07:00-07:00 — complete schema and current-source shadow sweeps

**Reason:** third plan review found one stale start-schema test, obsolete argv
restatements in current Go sources, an unbounded post-cancel receive, and an
unstaged manifest owner.

**Delta:** Task 1 migrates `operationdispatch_test.go`; Task 4 scans and updates
all current docs, probes, production comments, and tests with narrow historical
allowlists; Task 0 uses three bounded stages on one Wait channel ending in
explicit Kill; and Task 5 stages `manifest.go`. The plan now states the same
closed-presentation rule for code, tests, and docs (`ARCH-PURPOSE`,
`ARCH-CONSTRAINTS`).

### 2026-09-01T09:55:00-07:00 — put installed composition under RED and close the current-doc sweep

**Reason:** second plan review found two live nonhistorical consumers outside
the planned docs, underspecified exactly-once process teardown, a smoke written
only after implementation, and the wrong artifact declaration owner.

**Delta:** Task 0 now writes and runs the full installed PTY smoke red before
production work, with one Wait channel and bounded reader join; Task 4 adds #153,
the park probe, and a repository-wide obsolete-argv audit; Task 5 names
`artifactpath/manifest.go` and treats the smoke as previously red. Malformed
parser coverage also pins unknown flags and the `--` escape for dash paths
(`ARCH-PURPOSE`, `ARCH-MOCK`, `ARCH-CONSTRAINTS`).

### 2026-09-01T09:42:00-07:00 — close TUI, terminal, portfolio, and smoke migrations

**Reason:** fresh-context plan review found that removing start arguments would
leave the TUI effect invalid, terminal presence lacked a concrete test seam,
the active Couch project was omitted, the PTY fake/teardown was underspecified,
and test files were incorrectly included in the production artifact inventory.

**Delta:** Task 1 now migrates the menu's accepted-token effect and tests;
Task 3 names `terminalFiles` plus existing `consoleRunnerFor`/PTY seams; Task 4
sweeps the active project; Task 5 specifies the exact policy response,
non-leaking Pair fake, EOF/cancellation/reap sequence, and production-only
artifact classification (`ARCH-PURPOSE`, `ARCH-MOCK`, `ARCH-CONSTRAINTS`).
