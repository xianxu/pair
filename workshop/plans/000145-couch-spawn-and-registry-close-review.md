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
