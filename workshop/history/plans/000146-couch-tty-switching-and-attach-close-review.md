# Boundary Review — pair#146 (whole-issue close)

| field | value |
|-------|-------|
| issue | 146 — couch: tty switching and attach |
| repo | pair |
| issue file | workshop/issues/000146-couch-tty-switching-and-attach.md |
| boundary | whole-issue close |
| milestone | — |
| window | cf735f921fbd57bdd75b6fb082a7fcba22f9647f..df763b73319a3bd2ea6459de61540c1ef4a68883 |
| command | sdlc close --issue 146 |
| reviewer | codex |
| timestamp | 2026-08-25T13:25:54-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The switcher’s core architecture is strong, and the default, focused race, vet, and diff-hygiene checks pass. The boundary remains blocked by two prior Important findings: production’s outer console wiring is still not regression-pinned (BR-24), and the operation-documentation check remains prefix-ambiguous (BR-41). There is also a new Important correctness gap: a live actor hosted by another couch process has no local routing target, but Enter treats it as parked and attempts another start.

## 1. Strengths

- `Screen` now preserves chunk invariance beyond `maxPending` and frames all five string-terminated escape classes (`ptychild/screen.go:134-195,300-327`).
- Console output has one event-loop writer, with ordered teardown covering host closure, input closure, and worker joining (`couchtty/console.go:450-508`).
- Panel state derives from `Couch.Summarize(nil)` and joins transient routing separately (`couchtty/console.go:844-884`).
- The Core concepts contract is bidirectional and correctly classifies `Console` as INTEGRATION.
- README and atlas cover the shipped console, reserved row, controls, cold-start policy, and agent-facing operation.

## 2. Critical findings

None.

## 3. Important findings

### BR-24 remains open — outer console wiring is still unpinned

`couchcmd/run.go:181-205`, `couchcmd/run_test.go:410-466`

Tests exercise the non-terminal branch through `consoleRunner` and the terminal branch through `consoleRunnerFor`. Replacing `consoleRunner` itself with an unconditional `(nil, ExecRunner{})` still bypasses the latter tests. The production entry link remains removable without a default-suite regression.

### BR-41 remains open — operation documentation matching is prefix-ambiguous

`couchcmd/readme_test.go:51-59`

The agent-facing exemption is now correctly redirected to an enforced atlas test, but `strings.Contains(doc, "couch "+op.Name)` still allows `couch stop-all` to satisfy documentation coverage for `couch stop`. Match complete command tokens or lines.

### Live-but-unattached rows are treated as parked

`couchtty/console.go:962-970`, `couchtty/panel.go:10-25`, `couchcore/couch.go:358-389`

`Couch.Summarize(nil)` includes globally registered live actors, while `BindTargets` supplies routing IDs only for children hosted by this console. Therefore a row can have `Live=true` and `Target=""`. Enter checks only `Target` and dispatches `start`, which predictably hits the occupied-tree refusal instead of explaining that attachment requires #147.

Fix sketch: distinguish the three states—local live target, remote live without target, and parked—and test each through the Console. This is an `ARCH-PURPOSE` issue: liveness and local routing capability are separate facts.

## 4. Minor findings

- BR-7: stale `queries.go` and `readPTY` references remain.
- BR-8: `Child.SetSink` still races the pump if called after real-child startup.
- BR-9: `newTab` can replay a chunk and subsequently write the queued live copy.
- BR-20: production now calls `Replay`, but restoring the previous hand-composed equivalent would leave the behavior test green.
- BR-32: `ChildRows(0)` still contradicts its “never returns zero” contract.
- BR-33: `MakeRaw` failure and pre-`runConsole` errors still skip console/host teardown.
- BR-34: several comments still overstate or mislocate behavior.
- BR-35: exit selection can still drop final queued output, and the live conformance predicate remains timing-racy.

## 5. Test coverage notes

Verified:

- `go test ./... -count=1`
- Focused `go test -race` over couch/PTY/host/term packages
- Focused `go vet`
- `git diff --check` for the complete review window

The live suites and operator smoke were not rerun during this read-only review. Missing regression cases are the outer `consoleRunner` production link, live-without-local-target panel rows, exact operation-documentation tokens, zero-row sizing, and queued final output before exit.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass with enforcement caveat.** Replay policy, terminal controls, resolver behavior, summaries, and operations have shared sources. BR-20’s helper adoption is not itself pinned.
- **ARCH-PURE — pass.** Pure entities have direct non-IO tests; `Console` is correctly classified as the integration controller.
- **ARCH-PURPOSE — flag.** BR-24 leaves the central console entry removable, and live/unattached rows collapse two distinct states into the parked action.
- **ARCH-MOCK — pass with test-harness caveat.** Production and tests share the Host, Runner, and concrete Child seams with stateful fakes; BR-35 weakens the live conformance evidence but does not introduce a parallel mock boundary.

## 7. Plan revision recommendations

Append a revision distinguishing panel rows with a local routing target, live rows hosted elsewhere, and genuinely parked rows. State that only the parked state dispatches `start`; remote attachment remains explicitly deferred to #147.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Over-long sequences now retain framing state instead of rescanning payload as text; long OSC/ST split regressions and latch-aware fuzz coverage pin the false-bell case.
  - id: BR-7
    disposition: not-addressed
    note: |
      termcmd/run.go still cites deleted queries.go, and ptychild/replay.go still says output returns through deleted readPTY.
  - id: BR-8
    disposition: not-addressed
    note: |
      Child.SetSink still writes c.sink without synchronization or a fake-only guard while the real pump reads it.
  - id: BR-9
    disposition: not-addressed
    note: |
      newTab still snapshots after Start launches the pump, leaving a window where one chunk is replayed and later written from the live queue.
  - id: BR-19
    disposition: addressed
    note: |
      frame routes DCS, APC, PM, and SOS through string-terminator framing, with tests covering false bells and nested tmux controls.
  - id: BR-20
    disposition: not-addressed
    note: |
      Both production repaint sites now call Child.Replay, but replacing the helper call with the former StripQueries(Snapshot()) composition leaves the behavioral test green, so helper adoption itself is not pinned.
  - id: BR-24
    disposition: not-addressed
    note: |
      consoleRunnerFor and the path default are pinned, but the production consoleRunner link is not; terminal-path tests bypass it by calling consoleRunnerFor directly.
  - id: BR-32
    disposition: not-addressed
    note: |
      ChildRows(0) still returns zero despite the documented invariant, and the boundary test still begins at one.
  - id: BR-33
    disposition: not-addressed
    note: |
      Normal Run teardown is fixed, but MakeRaw failure returns before teardown and RunWithRuntime constructs OSHost before domain errors that can return without ever running or closing the console.
  - id: BR-34
    disposition: not-addressed
    note: |
      Screen still says ED every form, ops.go says the default is applied at the CLI, and keys.go still describes Console as policy-free glue.
  - id: BR-35
    disposition: not-addressed
    note: |
      Run can still select the exit event before draining already-queued final chunks, and doneBeforeExit is still sampled after the write that may end the live child.
  - id: BR-39
    disposition: addressed
    note: |
      Inactive row damage is retained in pane.rowDirty and a consumer-ordered regression test observes it before switching.
  - id: BR-40
    disposition: addressed
    note: |
      Console and terminal-writing entities are classified as Integration, and the bidirectional Core concepts contract pins missing, extra, relocated, and misclassified rows.
  - id: BR-41
    disposition: not-addressed
    note: |
      The atlas redirection is now enforced, but the operator-operation check remains prefix-ambiguous because it uses strings.Contains on couch plus the operation name.
findings:
  - id: new
    severity: Important
    family: routing-capability-conflated-with-liveness
    title: |
      Enter treats a live actor hosted elsewhere as a parked worktree and dispatches start
    detail: |
      Couch.Summarize(nil) includes globally registered live actors, but BindTargets adds a Target only for children hosted by this Console. console.go:968 checks Target alone, so a Live row with no local Target takes the parked-start branch and reaches the occupied-tree refusal. Model local-live, remote-live, and parked as distinct states; only parked should start, while remote-live should explain that attachment requires pair#147. Add a composed test because existing fixtures cover only local-live and parked rows (ARCH-PURPOSE).
```

---

## Re-review — 2026-08-25T13:43:46-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 146 — couch: tty switching and attach |
| repo | pair |
| issue file | workshop/issues/000146-couch-tty-switching-and-attach.md |
| boundary | whole-issue close |
| milestone | — |
| window | cf735f921fbd57bdd75b6fb082a7fcba22f9647f..da9e17e6ace47e5837b05096fa8bed2d1363e01e |
| command | sdlc close --issue 146 |
| reviewer | codex |
| timestamp | 2026-08-25T13:43:46-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The issue’s core behavior is ready to ship. Both Important findings are addressed with reachable default-suite regressions: real-terminal console wiring is exercised through `consoleRunner`, and remote-live actors are distinguished from parked work. Full, race, vet, and live-conformance suites pass. Six prior Minor findings remain partially open because their fixes are unpinned or incomplete; none blocks the boundary.

## 1. Strengths

- Remote-live, local-live, and parked states are modeled independently, with a composed test proving remote actors never dispatch `start` ([console.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:978), [console_panel_regression_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console_panel_regression_test.go:138)).
- The production terminal-detection link is exercised using a real PTY, closing BR-24’s recurring central-wiring gap ([run_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/run_test.go:460)).
- Final child output is drained before pane removal, with a composed exit regression ([console.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:478), [console_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console_test.go:107)).
- Startup output for a new `pair term` tab is rendered exactly once ([run.go](/Users/xianxu/workspace/pair/cmd/internal/termcmd/run.go:707), [run_test.go](/Users/xianxu/workspace/pair/cmd/internal/termcmd/run_test.go:707)).
- README and atlas coverage now derives operation/control inventories and enforces the agent-facing documentation redirection.

## 2. Critical findings

None.

## 3. Important findings

None.

## 4. Minor findings

- BR-7: the stale references are gone, but no static regression prevents deleted identifiers returning in comments.
- BR-8: `SetSink` and the pump now synchronize through `Child.mu`, but no concurrent `SetSink`/pump test would fail if the lock were removed.
- BR-20: both production repaint paths now use `Child.Replay`, but the helper-adoption rule remains structurally unpinned.
- BR-33: normal, signal, and mid-stream teardown are fixed; a `MakeRaw` failure or spawn refusal still bypasses `host.Close` ([run.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/run.go:139), [console.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:364)).
- BR-34: most comments were corrected, but atlas still says “hands the child its own stdio and block” ([couch.md](/Users/xianxu/workspace/pair/atlas/couch.md:32)).
- BR-35: the two races are fixed, but the separate `couchcore.waitUntilTrue` poller remains ([ptyrunner_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/ptyrunner_test.go:13)).

## 5. Test coverage notes

Passed:

- `go test ./... -count=1`
- `make test-race`
- Focused `go vet` for all changed terminal/couch packages
- `make test-live`
- `git diff --check`

Live operator smoke was not repeated during this read-only review.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** Replay policy, operations, panel controls, and terminal constants have shared owners.
- **ARCH-PURE — pass.** Panel/focus/framing decisions are directly unit-tested; `Console` is honestly classified as the integration controller.
- **ARCH-PURPOSE — pass.** The switcher delivers the complete local/remote/parked state distinction and production console path.
- **ARCH-MOCK — pass.** Production and tests share the concrete child/runner/host boundaries, backed by stateful fakes and live conformance.

## 7. Plan revision recommendations

None. The Core concepts table and implementation agree, and its bidirectional contract passed.

```findings
dispose:
  - id: BR-7
    disposition: not-addressed
    note: |
      The four stale references are gone, but no test or static contract fails if a deleted identifier is reintroduced into these comments.
  - id: BR-8
    disposition: not-addressed
    note: |
      SetSink and both sink reads now use Child.mu, but no concurrent regression exercises SetSink against the real pump; reverting the synchronization is not test-pinned.
  - id: BR-9
    disposition: addressed
    note: |
      newTab clears with a nil replay before releasing startup output, and TestTerminalMuxNewTabPrintsStartupOutputOnce pins the duplicate-output behavior.
  - id: BR-20
    disposition: not-addressed
    note: |
      Both production repaint paths now call Child.Replay, but existing tests only pin equivalent query-stripping behavior and remain green if the decision is hand-composed again.
  - id: BR-24
    disposition: addressed
    note: |
      TestConsoleRunnerDetectsARealPTY drives the actual consoleRunner terminal-detection link, while TestStartDefaultsItsPathToCwd pins the operation-level dot default through a spawn.
  - id: BR-32
    disposition: addressed
    note: |
      ChildRows(0) now returns 1 and reserve_test.go covers the exact zero-row boundary.
  - id: BR-33
    disposition: not-addressed
    note: |
      Run now owns ordered normal/signal teardown with regressions, but MakeRaw failure and pre-Run spawn refusal still return after OSHost construction without calling host.Close.
  - id: BR-34
    disposition: not-addressed
    note: |
      Most named comments were corrected, but atlas/couch.md still says the console no longer “hands the child its own stdio and block”; the comment sweep is also unpinned.
  - id: BR-35
    disposition: not-addressed
    note: |
      Exit ordering and doneBeforeExit are fixed and tested, but the separate couchcore waitUntilTrue polling implementation remains alongside couchtty's shared waitUpTo helper.
  - id: BR-41
    disposition: addressed
    note: |
      The agent-facing exemption is enforced against atlas/couch.md, and token-aware README matching has a negative longer-operation-prefix regression.
  - id: BR-50
    disposition: addressed
    note: |
      Panel rows retain global Live independently of local Target, and the composed Console regression proves Enter on a remote-live row reports deferred attachment without dispatching start.
```
