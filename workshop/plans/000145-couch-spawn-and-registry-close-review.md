# Boundary Review — pair#145 (whole-issue close)

| field | value |
|-------|-------|
| issue | 145 — couch: spawn and registry |
| repo | pair |
| issue file | workshop/issues/000145-couch-spawn-and-registry.md |
| boundary | whole-issue close |
| milestone | — |
| window | 78a3a4b72dc1a8f7f6f308d4f8fe4591d63dd3bc^..f82e458d95353e6548d5d7f682a0b03038ad7954 |
| command | sdlc close --issue 145 |
| reviewer | claude |
| timestamp | 2026-08-21T16:52:24-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The domain layer is the best part of this diff and it is genuinely good: seams are injected, the pure core is real, and I independently confirmed four of the plan's claimed deletion checks go red when the seam is removed, plus that the live conformance check actually fails against the pre-fix `ExecRunner` shape. What blocks SHIP is the shell above that domain: the collision guard — the issue's central invariant — never consults the liveness model the design is built around, so `couch start` on a tree is refused *forever* after the first session ends (reproduced against the built binary), `couch show <ref>` prints every tree in the registry rather than the one asked for (also reproduced), and `couch stop` does not signal anything despite its declared summary, which removes the record while the child keeps running — the guard failing open. All three live in code paths with no test coverage, which is the same gap: `couchcmd` hardwires production seams, so `start`/`stop`/refusal cannot be driven against the fakes at all. Fixes are small; they need a re-run before the close.

## 1. Strengths

- **The deletion-check discipline is real, not claimed.** I reverted four seams in a scratch working copy: dropping the input-side `Physical` in `Resolve` reddens `TestResolveAppliesPhysicalToTheInputBeforeCallingGit`; dropping the output-side one reddens `TestResolveAppliesPhysicalToGitsAnswer`; replacing `copyMap()` with a shared map reddens `TestRegisterSucceedsWithoutMutatingReceiver`; swapping `Enqueue` for `append` in `Actor.Send` reddens both the collapse and drop tests. These tests are load-bearing.
- **The live conformance check earns its description.** I restored the full pre-fix `ExecRunner` shape (no reaper, `Alive` = `kill -0`, `Wait` = `cmd.Wait`) and `PAIR_LIVE_COUCH=1` fails `TestRunnerConformance_SignalFatal` in 5s. Comparing two implementations against one scenario (`conformance_live_test.go:110-155`) is the right shape and it works.
- **The `scribecmd` fix is right in its subtlest part** (`scribecmd.go:83` vs `:105`): the cleanup defer is registered *after* `defer ptmx.Close()`, so LIFO runs it first. `scribecmd` and `scrollbackcmd` are green under `-race` here.
- **Original-case path vs folded lookup key** (`naming.go:9-13`, `registry.go:53-58`, `path.go:38-46`) is a subtle invariant, correctly implemented, and pinned by `TestLookupReturnsTheOriginalCasePath` / `TestRecordsCarryOriginalCaseWorktree`. Splitting `foldWith` out so both platform branches assert on any host (`path_test.go:41`) is the right move.
- `Enqueue` as a pure function with full-slice-expression copies (`mailbox.go:32-55`) — the mailbox policy is testable without goroutines, and `TestEnqueueDoesNotMutateTheInputSlice` proves it.

## 2. Critical findings

**C1 — the one-agent-per-tree guard never consults liveness; a dead actor blocks its tree forever.** `registry.go:70` and `:85` refuse on `len(existing) > 0`, and `Couch.IsLive` (`couch.go:94`) is called only from `Views`/`Summarize`, never from the guard. Reproduced end-to-end with the built binary against a registry holding one dead PID:

```
$ couch start /Users/xianxu/workspace/pair
/Users/xianxu/workspace/pair already has an agent:
  couch-dead (pid 999999)
  -> switch to it, or --same-tree (this repo runs one agent at a time)
```

`couch start` blocks for the child's lifetime and nothing unregisters on exit, so the *normal* end of a session leaves this state. The offered remedies don't apply: "switch to it" is `#146`, and `--same-tree` records a false co-tenancy. `couch stop` is the only way out and the message doesn't mention it. This falsifies Done-when 1 and 2 for every second invocation. Fix: have `CheckAvailable` (and `RegisterWithPolicy`) take a liveness predicate and treat non-live incumbents as absent — pruning them from the returned registry — and, at minimum, have `TreeOccupiedError` carry per-incumbent liveness so the refusal can say "the incumbent is dead, run `couch stop`". Note the occupancy test + error construction is copy-pasted across `:70-82` and `:85-101`, so the fix has to be written twice unless they're consolidated first (ARCH-DRY).

**C2 — `couch stop` never signals its child, and by forgetting the record it defeats the collision guard.** `ops.go:84` declares `"Signal an actor's child and forget it"`; `ops.go:94` only calls `c.Forget`. `grep` confirms no production caller of `Handle.Signal` or any kill path. Failure scenario: shell A runs `couch start ../pair` (blocking, agent live); shell B runs `couch stop pairtree` → registry entry deleted, agent still running; shell C runs `couch start ../pair` → **allowed**, two agents on one tree with one index lock and one branch — exactly the hazard the issue exists to close. Fix: either add `Signal(pid int, sig os.Signal) error` to `ProcOps` (identity-checked before signalling) and use it, or rename the operation and reword the summary to "Forget an actor's registry entry (does not stop the child)". The summary string is the machine-facing contract for `#148`, so it cannot be aspirational.

**C3 — `couch show <ref>` returns every tree with a registered actor, not the requested one.** `Summarize` (`couch.go:213`) adds the filter argument, then unconditionally adds a summary for every record in the registry (`couch.go:231`), so the filter only ever *adds* to the result. Verified in-package (`Summarize([]Worktree{"/repo"})` → `[/other /repo]`) and with the built binary — `couch show <pair tree>` printed byte-identical output to `couch list`. Contradicts the declared summary "Show the actors on one tree, by path or by name". Fix: when `trees` is non-empty, restrict the record loop to those keys; keep the current fold-in-all behaviour only for the `len(trees)==0` (`list`) case. `TestShowResolvesANameToItsTreePath` passes today because the fixture has exactly one tree and no actors — extend it to two trees with actors.

## 3. Important findings

**I1 — `Store.Load` fabricates `same_tree: true` on every record and persists the lie.** `store.go:69` sets `a.Args.SameTree = true` on replay to dodge its own re-register refusal. Verified: save a record with `SameTree=false`, `Load`, `Save` → the on-disk snapshot now reads `"same_tree": true`. The field is documented as the single record of the escape hatch (`startargs.go:17-19`), so after one couch invocation no reader can tell which actors actually used it. Fix: give `Registry` an unchecked `restore`/`insert` used only by `Load`, and stop mutating the record.

**I2 — an unreadable registry is silently indistinguishable from a first run, and the next `Save` destroys it.** `store.go:61` discards every `ReadFile` error, not just not-exist. Verified: with `registry.json` at mode `000`, `Load` returns `err=nil, records=0`; the next spawn writes a one-actor snapshot over the old file. Fix: `if errors.Is(err, fs.ErrNotExist)` → empty state; any other error → return it (the checklist's "silent error swallowing where the source raised").

**I3 — the operation-set audit cannot fail for the hazard it names.** `run_test.go:31` compares `Dispatch()`'s keys with `OperationNames()`; both are derived from `Operations()`, so the identity is structural. I added an undeclared `couch nuke` branch ahead of the table lookup in `RunWithRuntime` and the suite stayed green. Its own comment claims it catches "an operation reachable from the CLI but never declared", and Done-when 6 asks for an audit rather than a spot-check. Fix: assert over the *externally observable* surface — e.g. drive `RunWithRuntime` with each declared name and with a sample of undeclared ones, asserting every undeclared argv returns the exit-2 unknown-operation path; that is a test a hidden branch fails.

**I4 — the CLI shell hardwires production seams, so `start`, `stop` and the refusal renderer have zero coverage and cannot get any.** `run.go:87-91` constructs `ExecRunner{}`, `OSPathOps{}`, `ExecGit{}`, `OSProcOps{}` inline; `Runtime` injects only env + store dir. Consequences: no test exercises `render`'s `StartResult` path or `renderError`'s worktree-or-switch offer (Done-when 2's user-visible half), and the existing CLI tests shell out to real `git` in the repo checkout instead of `FakeGit`. This is the ARCH-MOCK at-review flag ("tests that cannot run the stack against the fake") and it's why C1/C2/C3 all shipped. Fix: move construction behind the `Runtime` seam (`rt.NewCouch() (*couchcore.Couch, error)`, or seam fields) so a test can drive the whole CLI against `FakeRunner`/`FakeGit`. This is also the API-stability question for the package `#146`/`#147`/`#148` will consume.

**I5 — the one production bug this milestone found is pinned only by an opt-in test that nothing runs.** With the full pre-fix `ExecRunner` shape restored, `go test ./cmd/internal/couchcore/` is **green in 0.35s**; only `PAIR_LIVE_COUCH=1` fails. No Makefile target sets any `PAIR_LIVE_*` gate (`test-race` is `go test -race ./cmd/pair-wrap/`, a directory that no longer exists). Separately, `execHandle.Alive`'s comment at `runner.go:120` says it "deliberately does NOT consult `procutil.Alive`" — swapping it back to `procutil.Alive` while keeping the reaper leaves *both* suites green, so that specific claim is undefended. Fix: add a default-suite test that starts `sh -c 'exit 0'`, polls `Alive()` without ever calling `Wait()`, and requires it to flip false — that fails against the pre-fix shape and needs no gate.

**I6 — the agent half of "agent-supplied description" was not built.** `Store.WriteDescription` (`store.go:101`) has zero callers and zero tests; `Couch.Describe` prefers a sidecar nothing ever writes. What ships is an operator typing `couch describe`, which lands in the naming table. The Spec is explicit — "Descriptors come from the agent, with a cached fallback ... because a cold agent cannot answer and cold is exactly where forgetting lives" — so the cached-fallback mechanism exists with no source to cache from (ARCH-PURPOSE: the deferred part is the point). Either wire a way for the child to publish (a `couch describe --self` invoked from the session, or a documented file contract) or drop the sidecar read and say so in the atlas.

**I7 — `atlas/couch.md:67-77` describes an actor loop the binary never runs.** "One goroutine per actor, holding a bounded mailbox" is written in the present tense; `NewActor` (`actor.go:36`) has no production call site — `Couch.Spawn` starts a child and returns. The code is fine as `#147` groundwork, but the atlas is the map and currently claims a flow that doesn't exist. Fix: move that section under "Planned, not built", or say plainly that the loop is implemented and unit-tested but not yet instantiated by any command.

**I8 — README update missing for a new installed binary.** `GO_BINS := pair couch` means `make install` now drops a second executable on `$PATH`, while README's Install section still states "`pair` is a **single Go binary**" and Command Usage lists only `pair …`. Per the docs gate this is a README finding even for experimental surface — one line under Install/Command Usage pointing at `atlas/couch.md` clears it.

**I9 — the issue's Plan ticks the operator smoke beyond the evidence.** `- [x] Operator smoke: host one real pair child` corresponds to plan Task 17, whose five checkboxes are all still `- [ ]`, and the issue Log itself records "Not yet exercised by the operator: the second-shell `couch list` read path, the refusal offer, and the kbench-subdirectory case." Two of the unrun steps are precisely where C1 and C3 live. Either run steps 2–4 (they take a minute) or narrow the Plan bullet to what was actually verified.

## 4. Minor findings

- `mailbox.go:35` — collapse matches on `Kind` alone, so a *non*-control message silently replaces a queued Control one of the same kind (verified: `stop{Control:true}` + `stop{}` → one entry with `Control:false`, `ok=true`), contradicting "never drop a Control message". Unreachable today only because `Actor` is unwired.
- `registry.go:70` / `:85` — the occupancy test and `TreeOccupiedError` construction are duplicated verbatim; make `CheckAvailable` the single source and have `RegisterWithPolicy` call it (ARCH-DRY).
- `Makefile.local:6-8` states every Go binary "ships with the `pair-` prefix so it doesn't collide with anything else on PATH (pair-scribe replaced an earlier bare-named `scribe`)"; `couch` is bare-named and the comment wasn't amended.
- `couchcmd/errors.go`, `couchcore/errors.go` — one-line wrappers around `errors.As` at a single call site each; the rest of the repo calls `errors.As` directly (`wrap.go:2206`).
- `git.go:9-17` — the ARCH-DRY note says "revisit at the third consumer", but that threshold has already passed: `reviewcmd.Runtime.Git`, `continuationcmd/git.go`, `launcher/runcli.go` and `slugcmd` all shell to git today.
- Dead exported API: `Couch.List`, `Couch.Policy`, `Registry.Unregister`, `FakeRunner.Signals`, `StartArgs.AgentStack`/`Stack`/`Issue`/`ExtraArgs` (the CLI never populates any of them, so the "structured start-args" record is currently `{Worktree, Cwd, SameTree}`).
- `bindArgs` (`run.go:110`) accepts any `--flag` silently; `--same-tre` typo leaves the guard in force with no diagnostic.
- `make test-race` targets `./cmd/pair-wrap/`, which no longer exists — while this diff's own lesson says to run the whole tree under `-race`. Pointing it at `./cmd/...` would encode the lesson.
- `strings.go:7` — `trimTrailingNewline` is `strings.TrimSpace`; the name understates it. `sanitizeKey` maps `/a/b` and `/a_b` to the same sidecar file.
- `store.go:47` — `Save` writes `reg.Records()` in Go map order, so `registry.json` churns between otherwise identical saves.
- No locking on `registry.json`; two couch processes are last-writer-wins (narrow today because `start` saves once and then blocks).

## 5. Test coverage notes

- 78 test functions across the two new packages; all green, and green under `-race` (`couchcore` 2.3s, `couchcmd` 2.1s). `go vet` clean. `go build ./cmd/pair-go` + `--version` still works (Done-when 7 holds).
- I could not independently confirm the "full tree green under `-race`" claim: `./cmd/internal/wrapcmd` fails here with `operation not permitted` re-executing its own test binary — an environment restriction in my shell, not a code defect. Everything else I ran was green.
- Coverage gaps, in priority order: the CLI `start` path and `renderError` (I4), `Summarize` with a filter (C3), `stop` (C2), guard-vs-liveness (C1), `Store.WriteDescription`/`ReadDescription` (I6), `execHandle.Alive` in the default suite (I5).
- Where tests do exist they mostly pin real logic rather than restate the implementation — `TestRefusedSpawnStartsNoProcess`, `TestSnapshotIsOnDiskWhileTheChildIsStillAlive` and `TestSpawnFailureLeavesTheTreeFree` are all genuinely load-bearing orderings.

## 6. Architecture

- **ARCH-DRY — flag (Minor).** `CheckAvailable`/`RegisterWithPolicy` duplicate the occupancy block (`registry.go:70`,`:85`); `GitRunner` duplicates `reviewcmd.Runtime.Git` with a documented "third consumer" trigger that has already fired; two single-use `errors.As` wrappers.
- **ARCH-PURE — pass on the domain, flag on the shell (I4).** `couchcore` is a genuinely pure core with a thin injected shell — `Enqueue`, `foldWith`, `PolicyTable` and `Registry` all test without IO, and `NormalizePath`'s one impurity is documented rather than hidden. The flag is `couchcmd`: business decisions (which seams, what to render on refusal, whether to `Wait`) sit in a function that constructs its own IO.
- **ARCH-PURPOSE — flag (I6, I7, C2).** Shadow-sweep on the single-source claim: `Operations()` is the source and the CLI *does* derive from it (no argv switch) — that part is delivered. But the audit that was supposed to enforce it is tautological (I3), so the derivation is convention rather than enforcement; the agent-supplied descriptor is a read path with no writer; and `stop`'s declared summary is a hand-maintained claim the code doesn't satisfy.
- **ARCH-MOCK — pass on the fakes, flag on reach (I4, I5).** `FakeRunner` is properly stateful with a scriptable signal disposition, `FakeGit` keys on `dir`, `Store` boots from any directory (portable, non-production). The flags: production flow and test flow do **not** share the boundary at the CLI layer, and the conformance check has no cadence — no target runs `PAIR_LIVE_COUCH=1`, so drift is detected only when someone remembers.

## 7. Plan revision recommendations

Add a `## Revisions` section to `workshop/plans/000145-couch-spawn-and-registry-plan.md` recording:

1. **`NormalizePath` is `filepath.Abs` only.** The Core-concepts bullet still says "`filepath.Abs` + `filepath.Clean`" and Task 1's deletion check still prescribes removing `Clean`; `Abs` already cleans, so the explicit call was dead on the success path (the code and `lessons.md` both record this — the plan doesn't).
2. **Task 15's audit mechanism.** "Identity, not overlap" between two views of the same `Operations()` slice is structurally guaranteed and cannot fail; the plan should specify an audit over the CLI's observable behaviour instead.
3. **Task 14 was not completed.** The sidecar reader shipped; nothing writes it. Record what remains and whether it moves to `#146`.
4. **Task 5's `Runner`/`Handle` contract now includes background reaping** — liveness is a closed channel, not a syscall, and `SetDiesOn` was added to `FakeRunner` for conformance. That contract changed after the plan was written.
5. **Task 17's smoke is 1-of-5 run**, and the issue's Plan bullet is ticked for it; reconcile the two (steps 2–4 are the ones that would have caught C1 and C3).

```findings
findings:
  - id: new
    severity: Critical
    family: guard-ignores-recomputed-liveness
    title: |
      the one-agent-per-tree guard never consults IsLive, so a dead actor blocks its tree forever
    detail: |
      registry.go:70 and :85 refuse on registry membership alone; Couch.IsLive
      (couch.go:94) is called only from Views/Summarize. Since `couch start`
      blocks and nothing unregisters on child exit, the normal end of a session
      leaves a dead record that permanently refuses the tree. Reproduced with
      the built binary: the refusal names a dead PID and offers "switch to it"
      (unimplemented, #146) or --same-tree (records a false co-tenancy).
  - id: new
    severity: Critical
    family: declared-contract-not-implemented
    title: |
      couch stop never signals its child, and forgetting the record opens the collision hazard
    detail: |
      ops.go:84 declares "Signal an actor's child and forget it"; ops.go:94 only
      calls c.Forget, and no production code calls Handle.Signal or any kill
      path. Stopping a live actor leaves the agent running with no registry
      entry, so a subsequent `couch start` on that tree is allowed -- two agents
      on one index lock and one branch. The summary is also the machine-facing
      contract for the advisor in pair#148.
  - id: new
    severity: Critical
    family: filter-argument-only-adds
    title: |
      couch show <ref> prints every tree with a registered actor, not the requested one
    detail: |
      Summarize (couch.go:213) takes a tree filter, then unconditionally folds in
      every record in the registry (couch.go:231), so the filter can only add.
      Verified in-package (Summarize([/repo]) -> [/other /repo]) and with the
      built binary, where `couch show <pair tree>` printed output identical to
      `couch list`. Contradicts the declared summary "Show the actors on one
      tree". The existing test passes only because its fixture has one tree.
  - id: new
    severity: Important
    family: replay-mutates-persisted-record
    title: |
      Store.Load fabricates same_tree=true on every record and the next Save persists it
    detail: |
      store.go:69 sets a.Args.SameTree = true on replay to dodge its own
      re-register refusal. Verified: save with SameTree=false, Load, Save, and
      the on-disk snapshot reads "same_tree": true. StartArgs documents this
      field as the single record of the escape hatch, so afterwards no reader can
      tell which actors actually used it. Give Registry an unchecked insert for
      Load instead.
  - id: new
    severity: Important
    family: silent-error-swallowing
    title: |
      an unreadable registry reads as a first run, and the next Save destroys it
    detail: |
      store.go:61 discards every ReadFile error, not just not-exist. Verified
      with registry.json at mode 000: Load returns err=nil with zero records, and
      the next spawn writes a one-actor snapshot over the old file. Distinguish
      fs.ErrNotExist from real IO/permission failures and return the latter.
  - id: new
    severity: Important
    family: test-cannot-fail-for-claimed-reason
    title: |
      the operation-set audit compares two views of the same source and cannot fail
    detail: |
      run_test.go:31 asserts Dispatch()'s keys equal OperationNames(); both derive
      from Operations(), so the identity is structural. I added an undeclared
      `couch nuke` branch ahead of the table lookup in RunWithRuntime and the
      suite stayed green -- the exact hazard the comment claims it catches, and
      what Done-when 6 asks to be audited rather than spot-checked.
  - id: new
    severity: Important
    family: cli-shell-not-injectable
    title: |
      couchcmd constructs production seams inline, so start/stop/refusal have no reachable tests
    detail: |
      run.go:87-91 builds ExecRunner/OSPathOps/ExecGit/OSProcOps directly;
      Runtime injects only env and store dir. No test exercises render's
      StartResult path or renderError's worktree-or-switch offer (Done-when 2),
      and existing CLI tests shell out to real git instead of FakeGit. ARCH-MOCK:
      production flow and test flow do not share the boundary at this layer,
      which is why the three Critical findings shipped.
  - id: new
    severity: Important
    family: fix-pinned-only-by-opt-in-test
    title: |
      the ExecRunner liveness fix is pinned only by a gate nothing runs
    detail: |
      Restoring the full pre-fix shape (no reaper, Alive = kill -0, Wait =
      cmd.Wait) leaves `go test ./cmd/internal/couchcore/` green in 0.35s; only
      PAIR_LIVE_COUCH=1 fails. No Makefile target sets any PAIR_LIVE_* gate, and
      test-race still points at ./cmd/pair-wrap/, which no longer exists.
      Separately, swapping Alive() back to procutil.Alive while keeping the
      reaper leaves BOTH suites green, so runner.go:120's "deliberately does NOT
      consult procutil.Alive" is undefended. A default-suite test that polls
      Alive() after `sh -c 'exit 0'` without calling Wait() would pin it.
  - id: new
    severity: Important
    family: deferred-purpose
    title: |
      the agent half of the agent-supplied description was not built
    detail: |
      Store.WriteDescription (store.go:101) has zero callers and zero tests;
      Couch.Describe prefers a sidecar nothing writes. What ships is an operator
      typing `couch describe`, landing in the naming table. The Spec is explicit
      that descriptors come from the agent with a cached fallback -- the cache
      exists with no source to cache from (ARCH-PURPOSE).
  - id: new
    severity: Important
    family: docs-claim-unbuilt-behavior
    title: |
      atlas/couch.md describes an actor loop that no command ever starts
    detail: |
      atlas/couch.md:67-77 says "One goroutine per actor, holding a bounded
      mailbox" in the present tense, but NewActor (actor.go:36) has no production
      call site -- Couch.Spawn starts a child and returns. Fine as pair#147
      groundwork; wrong as a map of what exists. Move it under "Planned, not
      built" or state that the loop is unit-tested but not instantiated.
  - id: new
    severity: Important
    family: readme-gate
    title: |
      README not updated for a second installed binary
    detail: |
      GO_BINS := pair couch means `make install` now puts a second executable on
      PATH, while README's Install section still says pair is "a single Go
      binary" and Command Usage lists only `pair ...`. One line pointing at
      atlas/couch.md clears the gate.
  - id: new
    severity: Important
    family: checklist-ticked-beyond-evidence
    title: |
      the issue ticks the operator smoke while four of its five steps are recorded unrun
    detail: |
      The issue Plan has "[x] Operator smoke: host one real pair child"; plan
      Task 17's five checkboxes are all unchecked, and the issue Log says the
      second-shell read path, the refusal offer and the kbench-subdirectory case
      were not exercised. Two of those unrun steps are exactly where the
      dead-actor refusal and the show-leak live.
  - id: new
    severity: Minor
    family: control-message-invariant
    title: |
      Enqueue's collapse can silently downgrade a queued Control message
    detail: |
      mailbox.go:35 matches on Kind alone, so a non-control message replaces a
      queued Control one of the same kind: Enqueue([stop{Control:true}],
      stop{}, 8) yields one entry with Control=false and ok=true. Contradicts
      "never drop a Control message"; unreachable today only because Actor has no
      production caller.
  - id: new
    severity: Minor
    family: duplicated-guard-block
    title: |
      CheckAvailable and RegisterWithPolicy duplicate the occupancy test verbatim
    detail: |
      registry.go:70-82 and :85-101 are the same block; ARCH-DRY, and it means
      the liveness fix for the Critical finding has to be written twice.
  - id: new
    severity: Minor
    family: binary-naming-convention
    title: |
      the bare-named couch binary contradicts the Makefile's own stated pair- prefix rule
    detail: |
      Makefile.local:6-8 says every Go binary ships with the pair- prefix to avoid
      PATH collisions (citing the bare `scribe` that was renamed). `couch` is
      bare and the comment was not amended either way.
  - id: new
    severity: Minor
    family: needless-indirection
    title: |
      two new errors.go files wrap errors.As at a single call site each
    detail: |
      couchcmd/errors.go and couchcore/errors.go add a one-line helper where the
      rest of the repo calls errors.As directly (wrap.go:2206, launcher tests).
  - id: new
    severity: Minor
    family: unused-public-surface
    title: |
      dead exported API and never-populated StartArgs fields
    detail: |
      Couch.List, Couch.Policy, Registry.Unregister, FakeRunner.Signals and
      StartArgs.AgentStack have zero non-test callers; the CLI never populates
      Stack, Issue or ExtraArgs, so the "structured start-args" record is
      effectively {Worktree, Cwd, SameTree}.
  - id: new
    severity: Minor
    family: silent-flag-acceptance
    title: |
      bindArgs accepts any --flag silently
    detail: |
      run.go:110 stores every --flag it sees, so `--same-tre` leaves the guard in
      force with no diagnostic -- unhelpful for the one loud escape hatch.
  - id: new
    severity: Minor
    family: stale-build-target
    title: |
      make test-race targets ./cmd/pair-wrap/, a directory that no longer exists
    detail: |
      Makefile.local:131. This diff's own lesson prescribes running the whole tree
      under -race; pointing the target at ./cmd/... would encode it instead of
      leaving it as a habit.
  - id: new
    severity: Minor
    family: misleading-helper-names
    title: |
      trimTrailingNewline is TrimSpace, and sanitizeKey can collide two trees
    detail: |
      strings.go:7 trims all surrounding whitespace, not a trailing newline;
      sanitizeKey maps /a/b and /a_b onto the same description sidecar file.
  - id: new
    severity: Minor
    family: nondeterministic-snapshot
    title: |
      Save writes actors in Go map order, so registry.json churns between identical saves
    detail: |
      store.go:47 uses reg.Records(), which iterates the map. Sorting by
      (worktree, id) would make the snapshot diffable.
  - id: new
    severity: Minor
    family: unsynchronised-shared-state
    title: |
      no locking on registry.json across couch processes
    detail: |
      Concurrent invocations are last-writer-wins over the whole snapshot. Narrow
      today because `couch start` saves once and then blocks, but it widens as
      soon as any command saves more than once.
```

---

## Re-review — 2026-08-22T09:51:24-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 145 — couch: spawn and registry |
| repo | pair |
| issue file | workshop/issues/000145-couch-spawn-and-registry.md |
| boundary | whole-issue close |
| milestone | — |
| window | 78a3a4b72dc1a8f7f6f308d4f8fe4591d63dd3bc^..5c9619b0ab23dfe598b6cda0992a44ebe66d161b |
| command | sdlc close --issue 145 |
| reviewer | claude |
| timestamp | 2026-08-22T09:51:24-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

All twelve blocking findings from round 1 are genuinely fixed, and — unusually — every one is pinned by a test I confirmed goes red on revert, not just asserted in a commit message: removing the `PruneDead` call, removing `Stop`'s signal, making `Summarize` additive again, restoring `SameTree=true` on replay, swallowing `ReadFile` errors, injecting an undeclared `couch nuke` branch, and restoring the full pre-fix `ExecRunner` each turn a specific named test red. The three-valued `Liveness` follow-up (pruning fails closed, `syscall.Kill(pid,0)` instead of forking) is the right fix at the right layer. What blocks SHIP is requirements traceability, not stability: **Done-when 3 is only half delivered** — an agent-published description is *displayed* but does not *resolve*, because `NamingTable.Lookup` searches the operator-typed description while `PublishDescription` writes a sidecar the lookup never reads (verified: `ResolveRef("composer")` → `no actor matches`). Alongside it, the `--same-tree` hatch that Done-when 2's refusal offers produces two co-tenants that no ref can address, so `couch stop` refuses both with `"be specific"` — an instruction that cannot be followed (verified at the CLI). Both fixes are small; both need a test and a re-run before the close records these bullets as met.

## 1. Strengths

- **The fixes are pinned, and I checked rather than trusted.** Seven independent reverts in a scratch worktree, seven red tests: `TestDeadActorDoesNotBlockItsTreeForever`, `TestStopSignalsTheChildBeforeForgettingIt` (+ the couchcmd half), `TestShowFilterRestrictsRatherThanAdds`, `TestReplayPreservesSameTreeExactly`, `TestUnreadableRegistryErrorsRatherThanReadingAsFirstRun`, `TestCLIAcceptsExactlyTheDeclaredOperations` (caught my `couch nuke` branch verbatim, `run_test.go:248`), `TestAliveIsFalseForAnExitedChildWithoutCallingWait`. This is the standard the round-1 note asked for and it was met.
- **Three-valued liveness is the correct repair for the repair** (`procops.go:23-33`, `couch.go:92-118`). Prune only on `Dead`, signal on `Live`-or-`Unknown`, and both directions are pinned — `TestPruneKeepsRecordsWhoseLivenessIsUnknown` and `TestKnownDeadIsStillPruned` fail in opposite directions, so neither the fail-open nor the fail-shut regression can return quietly.
- **`OSProcOps.Exists` fixed the probe at the root** (`procops.go:78-95`): `syscall.Kill(pid, 0)` with ESRCH→Dead / EPERM→Live / anything-else→Unknown removes the fork dependency entirely, rather than tolerating it and compensating downstream. `TestGuardRefusesAgainstARealLiveProcess` then checks the real probe answers, which is the gap the unit tests structurally could not cover.
- **`Runtime.NewCouch` is the right seam** (`run.go:26-37`), and `FakeRunner.AutoExit` (`runner_fake.go:85-93`) is an honest model of a blocking `Wait` rather than a workaround for one — the comment says so, which is why the fake stays trustworthy.
- **The audit's new comment states its own limits** (`run_test.go:222-229`): a corpus of undeclared names plus "the real guarantee comes from there being one table-only `Resolve` and no switch". That is the difference between a test and a claim.

## 2. Critical findings

None. Every round-1 Critical is fixed and defended by a failing-on-revert test.

## 3. Important findings

**I1 — an agent-published description does not resolve, so Done-when 3 is half delivered.** `Couch.PublishDescription` (`couch.go:339`) writes `<store>/desc/<key>`; `Couch.Describe` (`couch.go:184`) prefers it; but `ResolveRef` (`couch.go:139`) goes through `NamingTable.Lookup` (`naming.go:44`), which searches only `NameEntry.Name` and `NameEntry.Description` — the operator-typed pair. Verified in-package: publish `"reworking the composer gate"`, then `ResolveRef("composer")` → `no actor matches "composer"`. The issue's Done-when 3 says the operator name *and the agent-supplied description* both resolve; the Spec makes descriptions a resolution input ("fuzzy-in/exact-out, so a duplicate label asks which"), which is exactly why `Lookup` already searches `Description`. The existing test `TestLookupMatchesDescriptionNotJustName` passes only because it uses the operator path (`SetDescription`). Fix: have `PublishDescription` also fold the line into the naming table, or have `Lookup` consult `Store.ReadDescription`; pin it with a test that publishes and then resolves.

> **This is the 2nd finding in family `deferred-purpose`.** Earlier rounds fixed instances. Do NOT fix this instance alone — the rule is: **for the agent-supplied descriptor, "shipped" means every consumer of a description derives from the agent's source, not just the display path.** Round 1 (BR-9) found the sidecar had no writer and a writer was added; the *reader* that matters for the Done-when — resolution — still derives from the hand-maintained naming table. Measured prevalence on this issue: 2 of the 2 consumers of a description (display, resolution); display now derives, resolution does not. Sweep both, and state in `## Log` which consumer derives from which source.

**I2 — `--same-tree` produces co-tenants that no ref can address, and `couch stop` tells the operator to do the impossible.** `ops.go:96-99` requires `len(recs) == 1`; `ResolveRef` (`couch.go:139-157`) matches a name or a path and returns *every* actor on the resolved trees — it has no `ActorID` branch. Verified through `RunWithRuntime` with two live co-tenants:

```
stop "/repo"           -> code=1  err='couch: "/repo" matches 2 actors; be specific'
stop "couch-ah8d"      -> code=1  err='couch: no actor matches "couch-ah8d"'
```

So the escape hatch Done-when 2's refusal explicitly offers creates a state where neither agent can be stopped through couch, and the error names a remedy that does not exist. The same message also fires with *zero* actors (a parked named tree → `matches 0 actors; be specific`), which reads as ambiguity when it is absence. Fix: let `ResolveRef` match an `ActorID` exactly (it is already the "pid" half of the identity model, per `actorid.go:7-11`), and split the 0-actor case into its own message. Add a CLI test that starts two co-tenants and stops each by id.

**I3 — three CLI tests drive the production git seam against the ambient checkout, and one asserts on the checkout's directory name.** `run_test.go:60-65`'s `run()` helper uses `testRT{dir: dir}` with `fakes: false`, so `NewCouch` builds `ExecGit{}`/`OSPathOps{}` and shells to real git in whatever directory the tests happen to run from. `TestShowResolvesANameToItsTreePath` (`run_test.go:200-210`) then asserts `strings.Contains(out, "/pair")`. Verified: in a pristine `git worktree` of this same commit it fails —

```
run_test.go:208: out = "pairtree  /Users/xianxu/.cache/couchrev\n  (no agent running)\n"
```

— and it fails identically under `-race`. Any checkout not named `pair` (a worktree, a CI clone, `/tmp/build`) reddens the suite for a reason unrelated to the code.

> **This is the 2nd finding in family `cli-shell-not-injectable`.** Do NOT just rename the assertion. The rule: **every `couchcmd` test drives the CLI through `Runtime`'s fakes; none reaches a production seam.** Round 1 (BR-7) added `Runtime.NewCouch` and routed `start`/`stop`/refusal through `fakeRT`, but left the older tests on the production path. Measured prevalence: 7 of 9 test functions use the non-fake `run()`; 3 of those (`TestListShowsANamedTreeWithNoAgent`, `TestShowResolvesANameToItsTreePath`, `TestRenderedOutputHasNoANSIWhenNotATerminal`) actually invoke git via `name ../..`. Move them onto `fakeRT` and delete the `fakes bool` fork so the production path is unreachable from a test at all.

**I4 — the launcher fake's new mutex covers 2 of 9 accessors of the map it protects.** `createflow_test.go:294`/`:302` lock `WriteAtomic` and `Remove`, but `ReadAgentDefault` (`:270`), `ReadFile` (`:287`), `FileSize` (`:308`), `Touch` (`:312-313`), `Rename` (`:321-323`) and `ReadDir` (`:331`) all touch `f.files` unguarded — and `Touch` and `Rename` *write* to it. The comment justifying `mu` says the fake "is genuinely concurrent" because `startAgentDefaultPersistence` writes through the seam from its own goroutine; if that goroutine's `WriteAtomic` overlaps a main-flow `Touch` or `Rename`, that is a concurrent map write, which the Go runtime turns into an unrecoverable `fatal error`, not a test failure. It passes today only because no exercised path overlaps those two.

> **This is the 2nd finding in family `unsynchronised-shared-state`.** The rule: **when synchronisation is added to a shared structure, the unit of work is an audit of every accessor of that structure, not the one accessor the detector happened to flag.** Round 1 raised the cross-process case (BR-22, registry.json). Measured prevalence here: 2 of 9 accessors locked. Lock the remaining seven (or embed the map behind a small guarded type so an unguarded accessor cannot be written), and apply the same audit to `Store`'s snapshot when BR-22 is taken up.

**I5 — the docs and the plan hand-restate the operation set, and both have drifted from it.** `Operations()` is genuinely single-sourced and the CLI now derives from it under an enforced audit — that half is done. But three places restate the list by hand and two are now wrong: `atlas/couch.md:13` says `couch start|list|show|stop|name|describe` (six) and plan Task 15 (`…-plan.md:204`) says "returns all six", while seven ship — `publish-description` landed in the same window and is documented nowhere a reader would find it. The plan also states `NormalizePath` is "`filepath.Abs` + `filepath.Clean`" (`:59`) and prescribes a deletion check on the `Clean` call (`:130`) that `path.go:24-28` deliberately does not contain; declares `Couch` as `{Runner, Path, Git, Store, Clock, IDs}` (`:118`) where the code has `Proc` too; omits `ProcOps`/`Liveness` from the seam table entirely, though they are the centre of the final design; and Task 17's five checkboxes are all `- [ ]` with step 1 annotated `<- operator, unrun`, contradicting the issue's own record that steps 1–4 ran. The plan carries **no `## Revisions` section**, so none of round 1's five plan-revision recommendations landed.

> **This is the 2nd finding in family `docs-claim-unbuilt-behavior`.** The rule: **any prose that restates the operation set or a seam list is a consumer of `Operations()`/the code, and a consumer that does not derive must be re-derived at every boundary** (ARCH-PURPOSE's shadow-sweep; ARCH-DRY on the restatement). Round 1 fixed the atlas's actor-loop tense; the same file's operation list drifted one commit later. Measured prevalence: 3 hand-maintained restatements (atlas `:13`, plan `:204`, plan seam table `:96-104`), 3 now inconsistent with the code. Either generate the atlas list from `couch --help`/`OperationNames()`, or add a test that every declared operation name appears in `atlas/couch.md` — then append the `## Revisions` entries listed in §7 below.

## 4. Minor findings

- `.gitignore` covers `bin/*` but not a root-level binary; a built `couch` Mach-O is sitting untracked in the working tree right now (`git check-ignore couch` → not ignored) and would be swept up by `git add -A`.
- `couch.go:196-203` — `Views` calls `c.Liveness(r)` twice per record (once for `Live`, once for `State`), so every `list` issues two probe round-trips per actor where one would do.
- `run.go:104-112` — `bindArgs`'s error is checked *after* its result is read for the `publish-description`/`$COUCH_TREE` default; safe today only because of the `parsed != nil` guard. Check `err` first.
- `couch.go:322-331` — the `Stop` doc comment says the identity token "is re-checked immediately before signalling", but the `Unknown` branch signals precisely when identity could *not* be confirmed. The tradeoff is right and argued in the inline comment; the doc comment above it should say so.
- `run.go:148` — `open, close := dim, reset` shadows the `close` builtin inside `renderTrees`.

## 5. Test coverage notes

- `go test ./cmd/internal/couchcore/ ./cmd/internal/couchcmd/` green; `go vet ./cmd/...` clean; `couchcore` green under `-race`. `couchcmd` under `-race` fails only on I3's directory-name assertion.
- I could not confirm a fully green tree: `keyscmd`, `termcmd`, `wrapcmd` and one `launcher` test fail in my shell with `operation not permitted` on `mktemp`/`pty.Start`/re-exec, plus a zellij probe timeout. Those are agent-shell restrictions, not code defects — but I am reporting them as unverified rather than assumed-green.
- `make test-race` runs `go test -race ./cmd/pair-wrap/`, a directory that does not exist; the target fails with `[setup failed]` (BR-19, still open). No target or CI job sets `PAIR_LIVE_COUCH`, so the live conformance suite has zero cadence.
- Gaps the diff could still ship a bug through: agent-description resolution (I1, none); `stop` with `--same-tree` co-tenants (I2, none); `publish-description` end-to-end through the CLI including the `$COUCH_TREE` fallback (`testRT.Getenv` returns `""`, so the fallback branch at `run.go:106` is never entered); `renderError`'s `WorktreeParallel` and `HeavyLocalState` branches (`run.go:194-199`) — `TestStartRendersTheRefusalWithThePolicyShapedOffer` runs with an empty `PolicyTable`, so only the `default:` arm is exercised, and Done-when 2 is specifically about the *policy-shaped* offer.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag.** `CheckAvailable` (`registry.go:70-80`) and `RegisterWithPolicy` (`registry.go:85-94`) still hold the identical occupancy block; `GitRunner` still duplicates `reviewcmd.Runtime.Git` behind its own "revisit at the third consumer" note (that threshold has passed — `reviewcmd`, `continuationcmd`, `launcher/runcli`, `slugcmd` all shell to git); two one-line `errors.As` wrappers remain. All previously raised as Minors and all still open.
- **ARCH-PURE — pass.** `couchcore` remains a real pure core with a thin injected shell, and `Runtime.NewCouch` moved the last construction decision out of `RunWithRuntime`. Nothing new to flag.
- **ARCH-PURPOSE — flag (I1, I5).** Shadow-sweep on the two single-source claims: the *operation set* is enforced (table-only `Resolve` + a behavioural audit) but its three prose restatements have drifted; the *description source* has a writer now but only one of its two consumers derives from it.
- **ARCH-MOCK — pass on the fakes, flag on reach and cadence (I3).** `FakeRunner` is stateful with a scriptable disposition, `FakeGit` keys on `dir`, `FakeProcOps` models the `Unknown` case that the real bug lived in, and `Store` boots from any directory. Two gaps: production and test flow still share the boundary only for *some* CLI operations, and the conformance check that found the zombie bug is behind a gate nothing runs — drift is detected only when someone remembers. For `#146`/`#147`, wiring `PAIR_LIVE_COUCH=1` into a target is the cheap half; the expensive half is that `Handle`'s contract will change when a pty arrives, and `FakeRunner`'s state model should be extended in the same commit rather than after.
- For `#148`'s advisor: `ArgSpec` + `Operations()` is a good machine surface, but `stop`'s "be specific" (I2) shows the resolution layer is the weak point — an advisor will hit the same dead end. Give `ResolveRef` an exact-`ActorID` branch before building on it.

## 7. Plan revision recommendations

`workshop/plans/000145-couch-spawn-and-registry-plan.md` has no `## Revisions` section; add one with these entries (items 1–5 restate round 1's unactioned recommendations, 6–8 are new):

1. **`NormalizePath` is `filepath.Abs` only.** Core-concepts `:59` and Task 1's deletion check `:130` both name a `filepath.Clean` the code deliberately omits (`path.go:24-28`).
2. **Task 15's audit mechanism changed.** The `reflect.DeepEqual(Dispatch(), OperationNames())` identity it prescribes is structurally unable to fail; the shipped audit drives `RunWithRuntime` over declared and undeclared names.
3. **Task 14 gained a writer.** `publish-description` + `COUCH_TREE`/`COUCH_STORE_DIR` were added after the plan was written; record that the sidecar's *resolution* consumer is still outstanding (I1).
4. **Task 5's `Runner` contract now includes background reaping** — liveness is a closed channel, not a syscall — plus `SetDiesOn` and `AutoExit` on the fake.
5. **Task 17 is 4-of-5 run.** Tick steps 1–4 (hosting, second-shell read, `show` filtering, refusal against a live incumbent) and drop the `<- operator, unrun` annotation from step 1; leave the kbench-subdirectory step unticked with its test-coverage note.
6. **The seam inventory is missing `ProcOps`.** Add `ProcOps`/`OSProcOps`/`FakeProcOps` (`couchcore/procops.go`) to the integration table and `Liveness` to the pure table, and correct `Couch`'s composition at `:118` to `{Runner, Path, Git, Proc, Store, Clock, IDs}`.
7. **Task 15 declares six operations; seven ship.** Add `publish-description` to the contract line at `:204` and to `atlas/couch.md:13`.
8. **Record the fail-closed liveness redesign as a design change, not a bugfix.** Decision 6 still describes a two-valued "equal ⇒ alive; different or absent ⇒ gone" model; the shipped model is three-valued and prunes only on `Dead`, which is what the smoke test forced.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Verified: deleting Spawn's PruneDead call reddens TestDeadActorDoesNotBlockItsTreeForever and TestKnownDeadIsStillPruned.
  - id: BR-2
    disposition: addressed
    note: |
      Verified: deleting Stop's signal block reddens 2 couchcore tests and 1 couchcmd test.
  - id: BR-3
    disposition: addressed
    note: |
      Verified: making Summarize additive again reddens TestShowFilterRestrictsRatherThanAdds ([/other /repo]).
  - id: BR-4
    disposition: addressed
    note: |
      Verified: restoring SameTree=true + Register on replay reddens TestReplayPreservesSameTreeExactly.
  - id: BR-5
    disposition: addressed
    note: |
      Verified: swallowing every ReadFile error reddens TestUnreadableRegistryErrorsRatherThanReadingAsFirstRun.
  - id: BR-6
    disposition: addressed
    note: |
      Verified: an injected `couch nuke` branch ahead of Resolve now reddens TestCLIAcceptsExactlyTheDeclaredOperations.
  - id: BR-7
    disposition: addressed
    note: |
      Runtime.NewCouch makes start/stop/refusal reachable; the residual real-git tests are raised separately as a family repeat.
  - id: BR-8
    disposition: addressed
    note: |
      Verified: the full pre-fix ExecRunner shape now reddens TestAliveIsFalseForAnExitedChildWithoutCallingWait in the default suite. Residual, not re-raised - no target or CI sets PAIR_LIVE_COUCH, so conformance still has no cadence.
  - id: BR-9
    disposition: addressed
    note: |
      WriteDescription now has a caller, a CLI operation and tests; the unresolvable-description half is raised separately as a family repeat.
  - id: BR-10
    disposition: addressed
    note: |
      atlas section is now titled "built, unit-tested, not yet instantiated" and says no command starts one.
  - id: BR-11
    disposition: addressed
    note: |
      README.md:260-264 names the second binary and points at atlas/couch.md.
  - id: BR-12
    disposition: addressed
    note: |
      The issue Plan bullet now enumerates which smoke steps ran and states the kbench case is unrun; the plan file's own Task 17 checkboxes are stale in the opposite direction (see plan revisions).
  - id: BR-13
    disposition: not-addressed
    note: |
      Still reproduces - Enqueue([stop{Control:true}], stop{}, 8) yields one entry with Control=false, ok=true.
  - id: BR-14
    disposition: not-addressed
    note: |
      registry.go:70-80 and :85-94 still hold the identical occupancy block.
  - id: BR-15
    disposition: not-addressed
    note: |
      Makefile.local:6-8 comment unamended; the binary is still bare-named `couch`.
  - id: BR-16
    disposition: not-addressed
    note: |
      Both couchcmd/errors.go and couchcore/errors.go still wrap errors.As at one call site each.
  - id: BR-17
    disposition: not-addressed
    note: |
      Couch.Policy, Registry.Unregister, FakeRunner.Signals and StartArgs.AgentStack still have zero non-test callers.
  - id: BR-18
    disposition: not-addressed
    note: |
      bindArgs still stores every --flag without validating it against the operation's ArgSpecs.
  - id: BR-19
    disposition: not-addressed
    note: |
      Makefile.local:131-132 still targets ./cmd/pair-wrap/; the target now fails outright with "directory not found / setup failed".
  - id: BR-20
    disposition: not-addressed
    note: |
      strings.go unchanged - trimTrailingNewline is still TrimSpace and sanitizeKey still collides /a/b with /a_b.
  - id: BR-21
    disposition: not-addressed
    note: |
      store.go Save still marshals reg.Records() in Go map order.
  - id: BR-22
    disposition: not-addressed
    note: |
      No locking on registry.json; see the new partial-mutex finding for the same rule inside the process.
findings:
  - id: new
    severity: Important
    family: deferred-purpose
    title: |
      an agent-published description is displayed but does not resolve, so Done-when 3 is half delivered
    detail: |
      PublishDescription writes the sidecar and Describe prefers it, but ResolveRef goes
      through NamingTable.Lookup, which searches only the operator-typed Name and
      Description. Verified in-package - publish "reworking the composer gate", then
      ResolveRef("composer") returns `no actor matches "composer"`. Done-when 3 requires
      the agent-supplied description to resolve to the right actor; only the operator's
      does. 2nd in this family - the rule is that every consumer of a description must
      derive from the agent's source, and display now derives while resolution does not.
  - id: new
    severity: Important
    family: unaddressable-state
    title: |
      --same-tree co-tenants cannot be stopped, and the error names a remedy that does not exist
    detail: |
      ops.go's stop requires ResolveRef to return exactly one actor, but ResolveRef matches
      a name or a path and returns every actor on the tree; it has no ActorID branch.
      Verified through RunWithRuntime with two live co-tenants: stop "/repo" fails with
      `"/repo" matches 2 actors; be specific`, and stop "couch-ah8d" fails with
      `no actor matches "couch-ah8d"`. The escape hatch Done-when 2's refusal offers thus
      creates a state couch cannot exit. The same message also fires for a parked tree
      with zero actors, reading as ambiguity when it is absence.
  - id: new
    severity: Important
    family: cli-shell-not-injectable
    title: |
      three couchcmd tests drive real git against the ambient checkout, and one asserts on the checkout's directory name
    detail: |
      run_test.go's run() helper uses testRT{fakes:false}, so NewCouch builds ExecGit and
      OSPathOps. TestShowResolvesANameToItsTreePath then asserts strings.Contains(out,
      "/pair"). Verified: in a pristine git worktree of the same commit it fails with
      out = "pairtree  /Users/xianxu/.cache/couchrev...", and identically under -race.
      2nd in this family - the rule is that every couchcmd test drives the CLI through
      Runtime's fakes. Measured prevalence: 7 of 9 test functions use the non-fake run();
      3 of those actually invoke git. Remove the fakes bool fork so the production path is
      unreachable from a test.
  - id: new
    severity: Important
    family: unsynchronised-shared-state
    title: |
      the launcher fake's new mutex guards 2 of 9 accessors of the map it protects
    detail: |
      createflow_test.go locks WriteAtomic and Remove, but ReadAgentDefault, ReadFile,
      FileSize, Touch, Rename and ReadDir all touch f.files unguarded - and Touch and
      Rename write to it. A concurrent WriteAtomic and Touch is a concurrent map write,
      which Go turns into an unrecoverable fatal error rather than a test failure. 2nd in
      this family - the rule is that adding synchronisation to a shared structure means
      auditing every accessor, not only the one the race detector flagged. Measured
      prevalence: 2 of 9 locked.
  - id: new
    severity: Important
    family: docs-claim-unbuilt-behavior
    title: |
      atlas and plan hand-restate the operation set and seam list, and three restatements have drifted from the code
    detail: |
      atlas/couch.md:13 and plan Task 15 both list six operations; seven ship
      (publish-description is documented nowhere a reader would look). The plan also
      states NormalizePath is Abs+Clean and prescribes a deletion check on a Clean the
      code deliberately omits, declares Couch without its Proc field, omits ProcOps and
      Liveness from the seam tables entirely, and leaves Task 17's checkboxes unticked
      with "operator, unrun" while the issue records steps 1-4 as run. No Revisions
      section exists, so round 1's five plan-revision recommendations never landed. 2nd in
      this family - the rule is that any prose restating the operation set or seam list is
      a consumer that must be re-derived at every boundary. Measured prevalence: 3
      restatements, 3 inconsistent.
  - id: new
    severity: Minor
    family: untracked-build-artifact
    title: |
      the new binary name is not gitignored and a built couch executable is sitting in the working tree
    detail: |
      .gitignore covers bin/* but not a root-level binary; `git check-ignore couch`
      reports it unignored and a Mach-O executable is untracked at the repo root right
      now, so a `git add -A` would commit it.
  - id: new
    severity: Minor
    family: redundant-recomputation
    title: |
      Views computes Liveness twice per record, so every list issues two probes per actor
    detail: |
      couch.go's Views sets Live from c.Liveness(r) and State from a second c.Liveness(r)
      call. Each is a syscall plus a kernel-token read.
  - id: new
    severity: Minor
    family: error-checked-after-use
    title: |
      bindArgs's error is checked after its result is read
    detail: |
      run.go reads parsed["tree"] and writes into parsed for the $COUCH_TREE default
      before the `if err != nil` check below it. Safe today only because of the
      `parsed != nil` guard; check err first.
```

---

## Re-review — 2026-08-22T10:10:37-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 145 — couch: spawn and registry |
| repo | pair |
| issue file | workshop/issues/000145-couch-spawn-and-registry.md |
| boundary | whole-issue close |
| milestone | — |
| window | 78a3a4b72dc1a8f7f6f308d4f8fe4591d63dd3bc^..d96bfd0743855447d8d22fabee0374e7770e4663 |
| command | sdlc close --issue 145 |
| reviewer | claude |
| timestamp | 2026-08-22T10:10:37-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I have everything I need. Writing the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All five round-2 Important findings are genuinely disposed, and the two with testable claims are pinned by tests I confirmed go red on revert rather than taken on the commit message: deleting `LookupTrees`' published-line branch reddens `TestAgentPublishedDescriptionResolvesNotJustDisplays`, and deleting `ResolveRef`'s `ActorID` branch reddens `TestCoTenantsAreAddressableByActorID`. The two structural fixes hold under measurement — `couchcmd` is green under `-race` in a pristine tree named `couchrev4` (BR-25's exact reproduction), and all 8 `fakeRuntime` methods that touch `f.files` now lock, enumerated rather than detector-driven (BR-26). The docs fix is the strongest available form: the atlas deletes the restatement rather than trying to keep a copy in sync. Nothing new is Critical and nothing on the shipped happy path is wrong. What I would fix before crossing: `couch start /repo true` **silently disables the one-agent-per-tree guard** — `same-tree` is declared as an optional `ArgSpec` and `bindArgs` binds every optional spec positionally (reproduced at the CLI: second start accepted, two records on one tree); the persisted `cwd` is the operator's relative path (`"cwd": "../pair"` in the live `registry.json`) in a record whose stated purpose is replay; the real-probe guard pin added by `c094baf` runs only under `PAIR_LIVE_COUCH=1`, which no target sets; and `testRT` mints a fresh `FixedIDGen` per CLI invocation, so every CLI-started actor is `couch-ah8d` and the BR-24 remedy cannot be tested where it lives. Eighteen prior Minors remain open and unaddressed.

## 1. Strengths

- **The two testable round-2 fixes are load-bearing, verified by revert.** I rebuilt HEAD in a scratch tree twice. Removing the `knownTrees()`/`Describe` loop from `LookupTrees` (`couch.go:186-194`) → `couch_test.go:493: ResolveRef: no actor or tree matches "composer"`. Removing the `ActorID` loop from `ResolveRef` (`couch.go:222-226`) → `couch_test.go:515: ResolveRef by id: no actor or tree matches "couch-b2c1"`. Neither is a test written to agree with its fix.
- **BR-25 is fixed at the rule, not the instance.** `testRT` has no `fakes bool` fork left, and `grep` confirms no `couchcmd` test can name `ExecGit`/`OSPathOps`/`OSProcOps`/`ExecRunner` at all. I ran the suite from `/tmp/claude-501/couchrev4` — a checkout whose basename is not `pair` — under `-race`: green. That is the precise reproduction that failed last round.
- **BR-26 was audited by enumeration.** `grep -n 'f\.files'` finds 10 sites across 8 methods; all 8 lock (`WriteAgentDefault` delegates through `WriteAtomic`). I traced the actual concurrent producer — `startAgentDefaultPersistence` (`createflow.go:556-563`) calls only `WaitReadyRecord` (reads immutable `readyErr`) and `WriteAgentDefault` — so the guarded set is complete, not merely larger. `launcher` green under `-race -count=2`.
- **The atlas fix removes the consumer instead of syncing it** (`atlas/couch.md:16-20`): "The operation set is deliberately not listed here… Run `couch --help`." A restatement that no longer exists cannot drift, which is a stronger answer than a drift test. Same treatment for the seam list (`:56-60`).
- **The co-tenant remedy works end-to-end when ids differ.** Driving `RunWithRuntime` with a shared id generator: `stop /repo` → `"/repo" matches 2 actors; stop one by id: couch-ah8d couch-b2c1`; `stop couch-b2c1` → `signalled couch-b2c1 on /repo (pid 1001)`, leaving the other. The error now names a remedy that exists.
- Done-when 7 still holds: `go build ./cmd/pair-go` + `./cmd/couch` both succeed; `pair --version` and `couch --help` both answer.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `couch start <path> true` silently disables the one-agent-per-tree guard.** `ops.go:60-67` declares `same-tree` as an `ArgSpec{Required: false}`, and `bindArgs` (`run.go:144-157`) assigns positionals to *every* declared spec in order, requiredness only affecting whether a missing one errors. So the second positional binds to `same-tree`. Reproduced through `RunWithRuntime` against a tree with a live incumbent:

```
start "/repo" "true"  -> code=0  out="started couch-ah8d on /repo (pid 1001)"
list                  -> repo  /repo
                           couch-ah8d  live  pid 1000
                           couch-ah8d  dead  pid 1001
```

No `--same-tree`, no diagnostic, guard bypassed — the one invariant the issue exists to defend. It matters more than the typo case because `ArgSpec` is the machine-facing contract `pair#148`'s advisor constructs calls from: a caller reading the descriptor sees two args and can legitimately emit `["<path>", "true"]`.

> **This is the 2nd finding in family `silent-flag-acceptance`.** Do NOT patch the `start` operation. The rule: **`bindArgs` must validate argv against the declared `ArgSpecs` — reject an unknown `--name`, and never bind a flag-shaped spec positionally.** BR-18 is the "accepts any `--flag`" half; this is the "binds a flag as a positional" half, and both are the same missing validation. The structural fix is a kind on `ArgSpec` (`Flag` vs `Positional`) so the descriptor states which is which and `bindArgs` enforces it — which also makes the advisor's contract unambiguous. Measured prevalence: of the 7 declared operations, 1 (`start`) has a boolean flag among its specs and is bypassable this way; 7 of 7 accept arbitrary unknown `--flags`.

**I2 — the persisted `cwd` is the operator's relative path, in a record whose documented purpose is replay.** `StartArgs`' doc (`startargs.go:3-5`) says "It is persisted, so a revival reproduces the launch without the operator restating it", and `WorkingDir()` (`:24-29`) is what `Runner.Start` receives (`couch.go:79`). `Spawn` canonicalises `Worktree` but never `Cwd` — `ops.go:64` sets `Cwd: a["path"]` verbatim. Confirmed against the operator's live snapshot, not a fixture:

```json
{"id":"couch-7f9860cc","args":{"worktree":"/Users/xianxu/workspace/pair","cwd":"../pair"}, ...}
```

`../pair` reproduces nothing from any other directory, and the `kbench/competition/arc-agi-3` case — the one the identity model is built around — persists a relative subdirectory that cannot be re-derived. Latent today (nothing replays a record; `grep` shows `Cwd`/`WorkingDir` have no reader outside `Spawn`), which is exactly why it should be fixed before `#146`/`#147` read the format. Fix: in `Spawn`, before building the record, `if args.Cwd != "" { if p, err := c.Path.Physical(NormalizePath(args.Cwd)); err == nil { args.Cwd = p } }`. Note `TestSpawnStartsInASubdirectoryButRegistersTheTree` uses absolute fixture paths, so no existing test distinguishes the two — a case with a relative `Cwd` is the missing one.

**I3 — the real-probe pin for the guard runs only behind a gate nothing sets.** `c094baf` added `TestGuardRefusesAgainstARealLiveProcess` (`conformance_live_test.go:240`) precisely because "the unit tests use FakeProcOps, so they prove the logic but not that OSProcOps can actually answer" — the gap the fail-open pruning bug lived in. It opens with `liveOnly(t)`, i.e. `PAIR_LIVE_COUCH=1`. No Makefile target, no CI job and no script sets that variable anywhere in the tree; `make test-race` (`Makefile.local:131-132`) still targets `./cmd/pair-wrap/`, which does not exist, so it fails outright. I ran the gated suite by hand and it passes — but "passes when a reviewer remembers" is the state BR-8 was raised about.

> **This is the 2nd finding in family `fix-pinned-only-by-opt-in-test`.** Do NOT just move this one test. The rule: **a fix is pinned by a test in the suite that actually runs; a gated conformance check is a supplement to that pin, never the pin itself — and a gate with no invocation site is not a check.** BR-8 was disposed by adding a default-suite pin for the zombie fix while its own note recorded the residual ("no target or CI sets PAIR_LIVE_COUCH"); the next fix went straight back behind the same gate. The rule-level fix is one target — `make test-live: PAIR_LIVE_COUCH=1 go test ./cmd/...` — plus repointing `test-race` at `./cmd/...`, which retires BR-19 in the same edit and encodes this window's own lesson ("Run the whole tree under -race"). Measured prevalence: 5 of 5 live-gated tests in `couchcore` have no invocation site.

**I4 — `testRT` mints a fresh id generator per CLI invocation, so no `couchcmd` test can hold two distinguishable actors.** `run_test.go:31` builds `couchcore.NewFixedIDGen("ah8d", "b2c1")` inside `NewCouch()`, which the harness calls once per `RunWithRuntime`. Production also gets a fresh generator per process — but a *random* one, so ids differ; the fake's restarts, so they don't. `"b2c1"` is dead in every `couchcmd` test. Consequence, reproduced:

```
start /repo; start /repo --same-tree   -> two records, both id couch-ah8d
stop /repo    -> `"/repo" matches 2 actors; stop one by id: couch-ah8d couch-ah8d`
stop couch-ah8d -> "signalled couch-ah8d (pid 1000)"; list -> "no trees"
```

One child signalled, **both** records forgotten — `Registry.RemoveActor` matches by id across the tree — which is BR-2's hazard reopened. That specific outcome needs colliding ids, so it is not a production bug (crypto/rand on Go 1.26 does not fail; the `couch-00000000` fallback in `actorid.go:23` is dead code). The finding is that the harness makes the state BR-24 is *about* unrepresentable, so the CLI-facing remedy shipped with no CLI-facing test, and `ops.go`'s three new `stop` branches (`:102-117`) have none either — I had to write throwaway tests to exercise them. Fix: hoist the generator onto `testRT` so one instance spans invocations, then add the co-tenant stop-by-id and parked-tree-absence cases.

## 4. Minor findings

- `conformance_live_test.go:244-251` — the same test reaches two production seams it is not measuring. Its "not a repo" fallback is `Resolve(".")`, i.e. the ambient checkout: it fails outside a git tree (`resolve worktree for …/cmd/internal/couchcore: exit status 128` in my extracted copy, passes in the checkout), where `TestGitConformance_LinkedWorktree` two functions up does the right thing and `git init`s a temp repo. It also uses the real `ExecRunner` for the spawn under test, so on the regression it exists to detect it would fork `pair --layout2` into the operator's checkout with the test binary's stdio. Only `OSProcOps` needs to be real here. **3rd in family `cli-shell-not-injectable`** — the rule is *a test uses the production seam only for the thing it measures; every other seam is a fake*; rounds 1 and 2 applied it to `couchcmd`, and `couchcore`'s live tests were never swept. Measured prevalence: 1 of 5 live tests, and the only non-portable one.
- `COUCH_TREE`, `COUCH_STORE_DIR` and the agent-side publish contract appear in no document a reader or an agent would find — `grep` over `*.md`/`*.lua`/`*.kdl`/`*.sh` outside `workshop/plans` hits only the issue Log. The operation itself is discoverable via `couch --help`, but nothing tells a session inside a couch-spawned tree that it should publish, or what the env contract is. **2nd in family `readme-gate`**; the rule is *new operator- or agent-facing surface is documented where its reader looks*.
- `d96bfd0`'s commit body has a full `couch --help` dump spliced mid-sentence ("prose points at couch - supervise agent actors, one per working tree / describe … stop …"), mangling the paragraph. The branch is 4 commits ahead of `origin/main`, so it is still rewordable.
- `couch.go:159-176` / `:186-194` / `:317-331` each build the same "dedup trees by `Key()`" fold with a near-identical `add` closure; `Summarize`'s `len(trees)==0` branch re-walks `c.names.All()` where `knownTrees()` already returns the union.
- `couch.go:186-193` — `LookupTrees` substring-matches the needle against every tree's `Describe(w)`, including when the needle is a path. An agent whose published line happens to contain another tree's path makes that path ambiguous, and `treeFor` (`:277-284`) has no path-exact escape the way `ResolveRef` now has an id-exact one.

## 5. Test coverage notes

- `couchcore` and `couchcmd` green, and green under `-race`, both in this checkout and in a pristine differently-named extraction. `go vet ./cmd/...` clean. `launcher`, `scribecmd`, `scrollbackcmd` green under `-race -count=2`.
- Full-tree green remains unverified from my shell: `keyscmd`, `termcmd` and `wrapcmd` fail on `mktemp`/`pty.Start`/test-binary re-exec with `operation not permitted`. Sandbox restrictions, not code — same as prior rounds, reported rather than assumed.
- The gated live suite passes by hand (`PAIR_LIVE_COUCH=1`), except `TestGuardRefusesAgainstARealLiveProcess` outside a git checkout (Minor above).
- Gaps that could still ship a bug: `bindArgs` positional binding of a flag (I1); a relative `Cwd` through `Spawn` (I2); all three `stop` disambiguation branches — `ops.go:102-117` has no test at any layer; `publish-description`'s `$COUCH_TREE` fallback (`run.go:105-108`) — `testRT.Getenv` returns `""`, so the branch is never entered; `renderError`'s `WorktreeParallel`/`HeavyLocalState` arms (`run.go:194-199`) — the refusal test runs with an empty `PolicyTable`, so only `default:` is exercised, and Done-when 2 is specifically about the *policy-shaped* offer.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass with a note.** No new duplicated logic of substance; the three tree-dedup folds in `couch.go` are the only repeat and are small. BR-14 (`CheckAvailable`/`RegisterWithPolicy`) and the `GitRunner`/`reviewcmd.Runtime.Git` overlap remain open from earlier rounds, not re-raised.
- **ARCH-PURE — pass with a note.** The core stays pure and the shell stays thin. One drift worth watching: `LookupTrees` is resolution *policy* that now performs a per-tree disk read through `Describe`, so the union rule cannot be tested without a `Store`. Tests use `t.TempDir()` so it is portable, but extracting a pure `labels(entry NameEntry, published string) []string` and letting the caller supply the published line would put the policy back in the core before `#147` grows more label sources.
- **ARCH-PURPOSE — pass on the shadow-sweep.** Descriptions: three consumers — display (`renderTrees` via `TreeSummary.Desc`), resolution (`LookupTrees`), and read-back (`describe <ref>`) — all now derive from `Couch.Describe`, so the round-2 half-delivery is closed. Operation set: consumers are CLI dispatch, `usage`, atlas and README; the first two derive, the last two now point rather than restate, and the plan's restatement is recorded as drifted in `## Revisions`. The residual is on the producer side (nothing instructs an agent to publish), noted Minor.
- **ARCH-MOCK — flag (I3, I4, Minor 1).** The fakes themselves are good: stateful, scriptable disposition, portable non-production storage. Three gaps at the boundary rather than in the doubles. (a) The conformance check that justifies the whole seam design has no invocation site, so drift is detected only by hand. (b) `testRT`'s per-invocation id generator makes the fake *diverge from the behaviour it stands in for* — production yields distinct ids across processes, the fake does not. (c) One live test spawns the real thing under test rather than the seam. For `#146`, `Handle`'s contract changes when a pty arrives; extend `FakeRunner`'s state model in the same commit, and give `PAIR_LIVE_COUCH` a target first so the conformance pair is actually compared.
- For `#148`: `ArgSpec` is the advisor's contract and it currently cannot express "this is a boolean flag, not a positional" (I1). Fix that before anything constructs calls from it.

## 7. Plan revision recommendations

The `## Revisions` section landed and correctly records rewrites 1–2 plus the five superseded statements — that closes round 1's outstanding recommendations. Two entries to append:

1. **`StartArgs.Cwd` is not canonicalised on the way into the record.** Task 11's contract says `Spawn` resolves then records; state that only `Worktree` is canonical today and that `Cwd` persists the operator's raw argument, so the "persisted, so a revival reproduces the launch" claim in `startargs.go` is not yet true (I2).
2. **Task 17's inline `<- operator, unrun` on step 1 is now contradicted by the Revisions note itself.** The note defers to the issue `## Log` as authority, which is right, but the annotation still reads as fact to anyone scanning the task list. Either tick steps 1–4 in place or drop the annotation and let the Revisions entry carry it.

```findings
dispose:
  - id: BR-13
    disposition: not-addressed
    note: |
      mailbox.go:35 unchanged; collapse still matches on Kind alone.
  - id: BR-14
    disposition: not-addressed
    note: |
      registry.go:70-80 and :85-94 still hold the identical occupancy block.
  - id: BR-15
    disposition: not-addressed
    note: |
      Makefile.local:6-8 comment unamended; the binary is still bare-named couch.
  - id: BR-16
    disposition: not-addressed
    note: |
      Both couchcmd/errors.go and couchcore/errors.go still wrap errors.As at one call site each.
  - id: BR-17
    disposition: not-addressed
    note: |
      Verified by grep: c.List(), Registry.Unregister, StartArgs.AgentStack and Couch.Policy still have zero non-test callers.
  - id: BR-18
    disposition: not-addressed
    note: |
      run.go:130-159 unchanged; see the new positional-binding finding, which is the same missing validation.
  - id: BR-19
    disposition: not-addressed
    note: |
      Makefile.local:131-132 still targets ./cmd/pair-wrap/; folded into the new gated-pin finding's rule fix.
  - id: BR-20
    disposition: not-addressed
    note: |
      strings.go unchanged - trimTrailingNewline is still TrimSpace and sanitizeKey still collides /a/b with /a_b.
  - id: BR-21
    disposition: not-addressed
    note: |
      store.go:52 still marshals reg.Records() in Go map order.
  - id: BR-22
    disposition: not-addressed
    note: |
      No locking on registry.json, and the stated narrowness has widened - Spawn now saves twice per start (PruneDead, then register).
  - id: BR-23
    disposition: addressed
    note: |
      Verified by revert - deleting LookupTrees' published-line loop reddens TestAgentPublishedDescriptionResolvesNotJustDisplays.
  - id: BR-24
    disposition: addressed
    note: |
      Verified by revert - deleting ResolveRef's ActorID loop reddens TestCoTenantsAreAddressableByActorID; CLI path confirmed working end-to-end with distinct ids.
  - id: BR-25
    disposition: addressed
    note: |
      The fakes bool fork is gone and no couchcmd test names a production seam; suite green under -race from a checkout named couchrev4.
  - id: BR-26
    disposition: addressed
    note: |
      All 8 fakeRuntime methods touching f.files lock; the only concurrent producer (startAgentDefaultPersistence) reaches the map solely via WriteAtomic.
  - id: BR-27
    disposition: addressed
    note: |
      atlas deletes the operation and seam restatements rather than syncing them; the plan gains the ## Revisions section. Residual - Task 17's inline "operator, unrun" annotation still reads as fact.
  - id: BR-28
    disposition: not-addressed
    note: |
      git check-ignore couch still reports it unignored and the Mach-O is still untracked at the repo root.
  - id: BR-29
    disposition: not-addressed
    note: |
      couch.go:252-253 still calls c.Liveness(r) twice per record.
  - id: BR-30
    disposition: not-addressed
    note: |
      run.go:103-110 still reads and writes parsed before the err check below it.
findings:
  - id: new
    severity: Important
    family: silent-flag-acceptance
    title: |
      couch start <path> true silently disables the one-agent-per-tree guard via positional binding
    detail: |
      ops.go:60-67 declares same-tree as an optional ArgSpec and bindArgs (run.go:144-157)
      binds every declared spec positionally, so the second positional lands on same-tree.
      Reproduced through RunWithRuntime against a live incumbent - `start /repo true` exits 0
      and list shows two records on one tree, with no --same-tree and no diagnostic. ArgSpec is
      also pair#148's machine contract, so an advisor emitting ["<path>","true"] disables the
      guard legitimately. 2nd in this family - the rule is that bindArgs must validate argv
      against the declared ArgSpecs, rejecting unknown --flags AND never binding a flag-shaped
      spec positionally; the structural fix is a kind field on ArgSpec. Measured prevalence:
      1 of 7 operations bypassable positionally, 7 of 7 accepting arbitrary unknown --flags.
  - id: new
    severity: Important
    family: persisted-record-not-canonical
    title: |
      the persisted cwd is the operator's relative path, in a record whose stated purpose is replay
    detail: |
      StartArgs' doc (startargs.go:3-5) says the record is persisted so a revival reproduces the
      launch, and WorkingDir() feeds Runner.Start directly. Spawn canonicalises Worktree but
      leaves Cwd verbatim from ops.go:64. Confirmed in the operator's live registry.json, not a
      fixture - {"worktree":"/Users/xianxu/workspace/pair","cwd":"../pair"}. Latent today (no
      reader outside Spawn) which is why it should be fixed before pair#146 reads the format.
      Fix - Physical(NormalizePath(args.Cwd)) before building the record. No existing test
      distinguishes the two because every fixture uses absolute paths.
  - id: new
    severity: Important
    family: fix-pinned-only-by-opt-in-test
    title: |
      the real-probe guard pin added by c094baf runs only under PAIR_LIVE_COUCH, which nothing sets
    detail: |
      conformance_live_test.go:240 opens with liveOnly(t). No Makefile target, CI job or script
      sets PAIR_LIVE_COUCH anywhere in the tree, and make test-race still points at the
      nonexistent ./cmd/pair-wrap/. 2nd in this family - the rule is that a fix is pinned by a
      test in the suite that actually runs, and a gate with no invocation site is not a check.
      BR-8's own dispose note recorded this residual and the next fix went straight back behind
      the same gate. Rule-level fix - one `make test-live` target plus repointing test-race at
      ./cmd/..., which also retires BR-19. Measured prevalence: 5 of 5 live-gated tests have no
      invocation site.
  - id: new
    severity: Important
    family: fake-diverges-from-production
    title: |
      testRT mints a fresh id generator per CLI invocation, so no couchcmd test can hold two distinguishable actors
    detail: |
      run_test.go:31 constructs NewFixedIDGen("ah8d","b2c1") inside NewCouch(), which the harness
      calls once per RunWithRuntime, so every CLI-started actor is couch-ah8d and "b2c1" is dead.
      Production also gets a fresh generator per process but a random one, so ids differ. Effect:
      with the fixture as-is, `stop couch-ah8d` on two co-tenants signals pid 1000 and forgets
      BOTH records (RemoveActor matches by id across the tree), leaving a running agent with no
      registration - BR-2's hazard. Not reachable in production (crypto/rand does not fail on Go
      1.26), but it makes the state BR-24 is about unrepresentable, so the CLI-facing remedy
      shipped with no CLI-facing test and ops.go:102-117's three stop branches have none either.
  - id: new
    severity: Minor
    family: cli-shell-not-injectable
    title: |
      the live guard test resolves the ambient checkout and forks the real pair on the regression it detects
    detail: |
      conformance_live_test.go:244-251 falls back to Resolve(".") when a temp dir is not a repo,
      so it fails outside a git tree (exit status 128 in an extracted copy, passes in the
      checkout) where TestGitConformance_LinkedWorktree two functions up git-inits a temp repo
      instead. It also uses the real ExecRunner for the spawn under test, so if the guard
      regressed it would fork `pair --layout2` into the operator's checkout with the test
      binary's stdio; only OSProcOps needs to be real here. 3rd in this family - the rule is that
      a test uses the production seam only for the thing it measures. Measured prevalence: 1 of
      5 live tests, the only non-portable one; rounds 1 and 2 swept couchcmd, couchcore never.
  - id: new
    severity: Minor
    family: readme-gate
    title: |
      COUCH_TREE, COUCH_STORE_DIR and the agent-side publish contract are documented nowhere a reader looks
    detail: |
      A grep over md/lua/kdl/sh outside workshop/plans hits only the issue Log. couch --help
      makes publish-description discoverable to a human at a shell, but nothing tells a session
      inside a couch-spawned tree that it should publish, or what the env contract is. 2nd in
      this family - the rule is that new operator- or agent-facing surface is documented where
      its reader looks, which for an agent-facing contract is not the same place as for an
      operator-facing one.
  - id: new
    severity: Minor
    family: docs-claim-unbuilt-behavior
    title: |
      d96bfd0's commit body has a couch --help dump spliced mid-sentence
    detail: |
      The paragraph reads "prose points at couch - supervise agent actors, one per working tree"
      followed by the whole rendered operation table, then resumes with "Same for the seam list."
      The branch is 4 commits ahead of origin/main, so it is still rewordable.
  - id: new
    severity: Minor
    family: duplicated-guard-block
    title: |
      three near-identical tree-dedup folds in couch.go, and Summarize re-walks what knownTrees already unions
    detail: |
      couch.go:159-176 (knownTrees), :186-194 (LookupTrees) and :317-331 (Summarize) each build
      the same seen-by-Key fold with a near-identical add closure, and Summarize's len(trees)==0
      branch re-iterates c.names.All() where knownTrees() already returns exactly that union.
```
