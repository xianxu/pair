# Boundary Review — pair#182 (milestone M1)

| field | value |
|-------|-------|
| issue | 182 — Relaunch an actor: restart Pair in place, keeping the agent conversation |
| repo | pair |
| issue file | workshop/issues/000182-relaunch-an-actor-restart-pair-in-place-keeping-the-agent-conversation.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 88fe1de011b4c6be58e5a8b20eed89dfa4000f5d..9acfd8e509d0d8e8b58ce6b73f77ec0f9b537b3b |
| command | sdlc milestone-close --issue 182 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-04T10:47:03-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers what it promised: `CheckResumePreconditions` is a genuine extraction (`DecideResume` calls it, so there is one copy), `Couch.Relaunch` raises every visible refusal before the park, all four outcomes exist with a test each, and the operation is declared and dispatched through `resolveOperationThread`. I verified the load-bearing guard by mutation in a scratch worktree: moving `CheckResumePreconditions` after `c.PairLifecycle.Park` turns `TestRelaunchRefusesBeforeParkingWhenTheResumeCouldNotSucceed` red with `a refused relaunch performed lifecycle work: [publish-request observe-completion observe-completion cleanup-attempt]` — the guard is real, not decorative. The `DecideResume` refactor preserves behavior on both the cold and warm paths (the equivalence test plus the pre-existing `TestDetachedResumeDoesNotRequireAnEstablishedBinding` pin the `isBindingDiagnostic` skip). Nothing here is Critical. What holds it back from SHIP is four Important items, all cheap: a resolver IO failure is replaced by a fabricated binding diagnostic (and the branch meant to catch it is dead code — I proved it by deleting it and watching the suite stay green), a non-live thread refuses with `resume-live` and a message naming no next action, ~15 lines of resume's evidence-gathering preamble are re-derived in `relaunch.go` with a divergence already in the first copy, and the declared operation reaches no operator surface while the plan asserts it does — which leaves a Done-when bullet with no owning task in either milestone.

---

## 1. Strengths

- **`cmd/internal/couchcore/precondition_test.go:89`** — the agreement test is the right shape for this refactor. It compares `CheckResumePreconditions` against `DecideResume` on an `asParked` transform and asserts the same *diagnostic code*, not just the same yes/no. The comment explaining why it does not use `cloneThreadRecord` (`cloneArgv` normalizes nil argv to empty, silently repairing the shape under comparison) is the kind of thing that only gets written when someone actually hit it.
- **`cmd/internal/couchcore/relaunch_test.go:76`** — the refusal test asserts the *absence of effects* (`Lifecycle.trace` empty, `TriggeredQuits` empty, record unchanged), not just that an error came back. That is what makes it a mutation-detector; I confirmed it fires.
- **`cmd/internal/couchcore/relaunch_test.go:146`** — the three park-failure cases are produced by driving the stateful fake into the failure (no completion committed, child deliberately not killed, publish error) rather than by stubbing a return value. ARCH-MOCK in the good sense: production flow and test flow share the `PairLifecycle`/`Artifacts` boundary.
- **`cmd/internal/couchcore/relaunch_test.go:223`** — `TestRelaunchProceedsWhenOnlyPairsCleanupFailed` pins a distinction reading alone gets wrong: `CleanupAttempt` runs after `FinalizePark` (`park.go:642`) and its error lands in `ParkResult.CleanupError` with the park returning nil. Verified against the source; the atlas paragraph describing it is accurate.
- **`cmd/internal/couchcore/relaunch_test.go:328`** — the seam test goes through `DispatchOperation` with the production executors in the switcher's `{repo-scope, tag}` dialect, and the comment records that the first attempt (in `couchcmd`) passed for an unrelated reason. Honest test archaeology, and the right layer in the end.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — a resolver IO failure is reported as "the binding is not established", and the branch meant to prevent that is unreachable.** `cmd/internal/couchcore/relaunch.go:96-108`.

`ResolveEstablished` returns `(NativeBindingResolution{}, err)` for a genuine IO failure — `artifactpath.Resolve` or `os.UserHomeDir` in `ScopedThreadArtifactCollisionChecker` (`artifactcollision.go:291-304`), or `QuerySessionContext` in `SessionInventoryNativeBindingResolver` (`resume.go:222-236`). The zero resolution then hits `bindingResumeDiagnostic`'s `default` arm → `ResumeBindingUnbound`, so `CheckResumePreconditions` at line 103 returns first and `if bindingErr != nil` at line 106 never runs. I deleted lines 106-108 in a scratch worktree and `go test ./cmd/internal/couchcore/ -run 'Relaunch|Resume|Park'` stayed green — dead code, and the real cause is discarded. `ResumeContext` (`resume.go:279-283`) returns the resolver's error directly, so this is also an inconsistency between the two callers of the same seam. Fix sketch: resolve the binding only after the non-binding preconditions pass (so a nil profile still yields `ResumeProfileMissing`), then return `bindingErr` when it is not a `*ResumeRefusal`; add a case to `TestRelaunchRefusesBeforeParkingWhenTheResumeCouldNotSucceed` with a resolver that errors. ARCH-SECURE at-review: the failure path substitutes a fabricated value the operator reads as evidence about the binding.

**I-2 — relaunch on a thread that is not running refuses with `resume-live` and names no next action.** `cmd/internal/couchcore/relaunch.go:81-83`.

`soleParkableIncarnation` fails for *no* live/unknown incarnation as well as for two (`park.go:768-783`), and both are mapped to `refuseResume(ResumeLive, err.Error())`. On a parked row the operator gets `resume-live: park requires exactly one identified live or unknown incarnation` — a code that says the opposite of the thread's state and a message about park's internals. This is reachable: the Spec's panel form is "`Alt+n` relaunches the HIGHLIGHTED ROW", and highlighted rows are frequently parked. It is the same class as pair#181 M3's "refusals that name working steps". Fix sketch: split the two cases — keep `ResumeLive` for >1 incarnation, and for a thread with none refuse with its own code and a message that names the working gesture ("`<tag>` is not running — Enter resumes it"). Add the parked-row case to the refusal table.

**I-3 — resume's evidence-gathering preamble is re-derived in `Relaunch`, and the first copy already diverges.** `cmd/internal/couchcore/relaunch.go:88-101` vs `cmd/internal/couchcore/resume.go:262-283`.

Both derive `agent` from `thread.LatestLaunchProfile`, type-assert `c.Artifacts` to `NativeBindingResolver`, and compute `pathExists` from `c.Path.Physical`. They already differ: `ResumeContext` starts from `pathExists := false` and calls `c.Path.Physical` unconditionally; `Relaunch` starts from `true` and skips the call when `c.Path == nil` — so a nil `Path` makes relaunch *pass* the path precondition and then panic inside `ResumeContext` one step later. The plan's own DRY rationale ("two parallel derivations drift toward whichever cases each author thought about") applies to the evidence as much as to the rules it extracted. Fix sketch: a `resumeEvidence(ctx, thread, address) (NativeBindingResolution, bool, error)` helper both call, with one nil-`Path` policy. ARCH-DRY.

**I-4 — the declared operation reaches no operator surface, and the plan says it does.** `cmd/internal/couchtty/menu.go:1008` (`menuActionItems`); plan Chunk 2 preamble.

The plan opens M2 with "M1's operation is reachable from the switcher's action list the moment it is declared." It is not: `menuActionItems` returns hardcoded slices (`{"detach","park","name","describe"}`, `{"resume","archive","name","describe"}`, …) and consumes `Operations()` nowhere. `ParseCLI` is a closed flag set, so there is no CLI path either. The issue's Done-when bullet "An actor action `relaunch` appears alongside detach and park, confirmed like park, and reachable from the same declared-operation surface" therefore has no owning task in *either* milestone. The class is wider than the plan's enumeration: relaunch must be added to **six** hand-maintained per-operation sites, not the two Task 10 names —
`menu.go:1008` (`menuActionItems`), `menu.go:~1040` (`confirmationMenuItems`, since it is `ConfirmRequired`), `menu.go:1306` (post-operation frame restore), `menu.go:1320` (`operationNeedsProjectionRefresh`), `console.go:1375` (the `expectedExits` bridge), `console.go:1425` (`consumeExpectedParkExitLocked`). ARCH-PURPOSE (name the class, sweep the enumeration) and ARCH-DRY. No code change is needed at the M1 boundary; the plan revision is (see §7).

## 4. Minor findings

- `cmd/internal/couchcore/relaunch_test.go:76` — the plan's Task 2 table lists four refusal cases; the shipped test has three (`corruptAgent` → `ResumeAgentUnsupported` dropped). The rule is covered purely in `precondition_test.go`, and I confirmed the diagnostic *precedence* is right (profile checks run before the binding check, so a nil or unsupported profile yields the profile code even though `ResolveEstablished` was already called with `""`), but no test pins that precedence through `Relaunch`.
- `workshop/plans/000182-relaunch-an-actor-plan.md:159` — Task 1 Step 5 "Commit" is unticked although `e7c6c6e8` landed it. `sdlc milestone-close`'s plan-unchecked guard will refuse on it.
- Issue `## Revisions` still carries "**⚠ The estimate is now stale.**" while `## Estimate` has since been re-derived for the grown scope (3.58 → 6.20, with the chord and holding-pane lines present). The warning now contradicts the block above it.
- `cmd/internal/couchcore/relaunch.go:43` — `RelaunchResult{Outcome, Record, Handle}` does not compose with `finishOperation`'s existing arms (`console.go:1324` `ParkResult`, `console.go:1328` `StartResult`), so M2 gets a third type-switch arm. `RelaunchResult{Outcome, Start StartResult}` would reuse the resume arm as-is.
- `cmd/internal/artifactpath/manifest.go:524` — `relaunch.go` inserted between `pathops.go` and `procops.go`, breaking the list's alphabetical run. Nothing enforces the order; noted so the next insert does not compound it.
- `cmd/internal/couchcore/relaunch.go:111` — `ParkResult` is discarded, so `CleanupError` is dropped. Pre-existing condition, not drift: `CleanupError` is written at `park.go:643` and read by no production code anywhere, which is worth its own issue rather than a fix here.

## 5. Test coverage notes

- I could not run the full suite to confirm Task 7 Step 2's "exit 0": `cmd/internal/couchcore` and `cmd/internal/couchcmd` fail in this environment on pty spawn (`ptychild: start sh: operation not permitted`, `open pty: operation not permitted`) — the known agent-shell restriction, unrelated to the diff. Everything reachable without a pty passes: the relaunch, precondition, dispatch-declaration and arity tests all green, `cmd/internal/artifactpath` green, `go build ./...` clean.
- Mutation-verified: precondition-before-park (fails when reordered) ✅; `bindingErr` handling (suite stays green when deleted — dead) ❌.
- The success test's continuity assertion is real, not decorative: `native-root-1` reaches `child.Env` only via `BuildCouchResumeLaunchProfile` → `profileRaw` → `launcher.CouchLaunchProfileEnv` (`launch_existing.go:65`), so a resume that dropped `RequiredSessionID` would fail it.
- Gap worth closing with I-2: no test drives `Relaunch` against a parked or archived address.
- `TestRelaunchStopsAtAFailedParkAndNamesTheRecovery`'s "no child was started" assertion is weaker than it reads — `DecideResume` would refuse `ResumeParking` on the open transaction anyway, so an implementation that *did* call resume would still spawn nothing. The outcome assertion (`ParkIncomplete`, not `ParkedNotResumed`) is what actually pins the skip.

## 6. Architectural notes

- **ARCH-DRY** — flag, I-3 and I-4. Pass on the thing the milestone was for: `CheckResumePreconditions` has exactly one definition and `DecideResume` consumes it.
- **ARCH-PURE** — pass. `CheckResumePreconditions` is a pure function over `(record, binding, pathExists)` and `precondition_test.go` runs it with no fixture, no fake, no filesystem. `Couch.Relaunch` is a thin sequencer over injected seams and holds no rules of its own.
- **ARCH-PURPOSE** — flag, I-4. The operation is built; the "action appears alongside detach and park" half of the purpose is neither built nor scheduled.
- **ARCH-MOCK** — pass, and notably well done. Both halves run against the same `PairLifecycle` / `Artifacts` seams production uses, park failures come from the stateful fake's state rather than from stubbed returns, and `envWithLiveThread` models the real Pair handshake (trigger → completion → child death).
- **ARCH-CONSTRAINTS** — pass with a note. The plan's envelope (park's 15s + 5s, resume's 10s ack, ~30s worst case) is inherited from the components; `Relaunch` adds no composed deadline of its own and nothing at M1 sits on a keystroke path. When M2 puts this behind `Alt+n`, the ~30s worst case becomes an operator-visible hold — the spinner is the mitigation the plan already names, but a test that the composed path is bounded (not just each half) is the missing measurement.
- **ARCH-SECURE** — flag, I-1. Otherwise clean: `validateThreadAddress` parses at the boundary, argv reaches the child as a slice via `BuildCouchResumeLaunchProfile`, `CheckResumePreconditions` rejects a nil argv and an unsupported agent before either is used, and no error message carries anything sensitive.
- Process note, not a finding: `4821dda3 #182 M2: couch: intercept alt+n` landed on the branch while this review was running, i.e. M2 started before M1's boundary closed. It is outside the pinned window and I did not review it.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000182-relaunch-an-actor-plan.md`:

1. **Chunk 2 preamble is wrong.** Strike "M1's operation is reachable from the switcher's action list the moment it is declared." `menuActionItems` (`menu.go:1008`) is a hand-written list that consumes `Operations()` nowhere, and `ParseCLI` is a closed flag set — after M1 the operation has no operator-reachable surface at all.
2. **Task 10 Step 2 under-enumerates.** `endsItsOwnChild` covers the two `console.go` lists; relaunch must additionally be added to `menuActionItems` (`menu.go:1008`), `confirmationMenuItems` (it is `ConfirmRequired`), the post-operation frame-restore switch (`menu.go:1306`), and `operationNeedsProjectionRefresh` (`menu.go:1320`). Write the enumeration into the plan and sweep it in one round rather than discovering the sites one review at a time.
3. **Add a task owning the Done-when bullet** "An actor action `relaunch` appears alongside detach and park, confirmed like park, and reachable from the same declared-operation surface", with its confirmation copy — currently unowned by M1 and M2 both.
4. **Task 2's illustrative test table** lists four refusal cases; three shipped. Record that `ResumeAgentUnsupported` is pinned in `precondition_test.go` at the pure level instead, or add the case.
5. **Tick Task 1 Step 5** (`e7c6c6e8` landed it) so `milestone-close`'s plan-unchecked guard does not refuse.
6. In the issue, drop or amend the `## Revisions` "⚠ The estimate is now stale" paragraph — `## Estimate` has since been re-derived to 6.20 for the grown scope, so the warning contradicts the block it points at.

```findings
findings:
  - id: new
    severity: Important
    family: swallowed-cause-fabricated-diagnostic
    title: |
      A resolver IO failure is reported as "binding is not established", and the branch meant to catch it is dead code
    detail: |
      relaunch.go:96-108 — ResolveEstablished returns a ZERO resolution on a real
      IO error (artifactpath.Resolve / os.UserHomeDir / QuerySessionContext), so
      bindingResumeDiagnostic's default arm yields ResumeBindingUnbound and
      CheckResumePreconditions returns at line 103; `if bindingErr != nil` at 106
      is unreachable. Proved by deleting lines 106-108 in a scratch worktree and
      running -run 'Relaunch|Resume|Park': still green. ResumeContext
      (resume.go:279-283) returns the resolver's error directly, so the two
      callers of the same seam disagree. Resolve the binding only after the
      non-binding preconditions pass, return bindingErr when it is not a
      *ResumeRefusal, and pin it with an erroring-resolver case. ARCH-SECURE.
  - id: new
    severity: Important
    family: refusal-names-no-next-action
    title: |
      Relaunch on a thread that is not running refuses with `resume-live` and names no next action
    detail: |
      relaunch.go:81-83 maps every soleParkableIncarnation failure to
      refuseResume(ResumeLive, err.Error()), but park.go:768-783 fails for NO
      live/unknown incarnation as well as for two. On a parked row the operator
      gets "resume-live: park requires exactly one identified live or unknown
      incarnation" — a code contradicting the state and a message about park's
      internals. Reachable: the Spec's panel form relaunches the HIGHLIGHTED row,
      which is often parked. Split the cases; for a thread with no incarnation
      refuse with its own code naming the working gesture ("Enter resumes it").
  - id: new
    severity: Important
    family: parallel-derivation-drift
    title: |
      Resume's evidence-gathering preamble is re-derived in Relaunch, and the first copy already diverges
    detail: |
      relaunch.go:88-101 vs resume.go:262-283 both derive `agent` from the saved
      profile, type-assert c.Artifacts to NativeBindingResolver, and compute
      pathExists from c.Path.Physical. They already differ on nil Path: relaunch
      defaults pathExists to true and skips the call, so the precondition PASSES
      and ResumeContext then nil-derefs one step later. The plan extracted the
      RULES for exactly this reason; the evidence needs the same treatment —
      one resumeEvidence(ctx, thread, address) helper with one nil-Path policy.
      ARCH-DRY.
  - id: new
    severity: Important
    family: declared-source-hand-maintained-consumers
    title: |
      The declared operation reaches no operator surface, and the plan asserts it does
    detail: |
      menuActionItems (menu.go:1008) returns hardcoded slices and consumes
      Operations() nowhere; ParseCLI is a closed flag set. So the plan's Chunk 2
      premise ("reachable from the switcher's action list the moment it is
      declared") is false, and the Done-when bullet "an actor action relaunch
      appears alongside detach and park ... reachable from the same
      declared-operation surface" has no owning task in M1 or M2. The class is
      six hand-maintained per-operation sites, not the two Task 10 enumerates:
      menu.go:1008, confirmationMenuItems, menu.go:1306, menu.go:1320,
      console.go:1375, console.go:1425. Fix the plan now and sweep the
      enumeration in one M2 round. ARCH-PURPOSE, ARCH-DRY.
  - id: new
    severity: Minor
    family: done-when-untested
    title: |
      No Relaunch-level test for the agent-unsupported and profile-missing refusals
    detail: |
      The plan's Task 2 table lists four cases; three shipped. The rules are
      covered purely in precondition_test.go and the diagnostic PRECEDENCE is
      correct (profile checks run before the binding check, so an unsupported
      agent still yields ResumeAgentUnsupported even though ResolveEstablished
      was already called with it), but nothing pins that precedence through
      Relaunch.
  - id: new
    severity: Minor
    family: plan-record-lags-code
    title: |
      Task 1 Step 5 is unticked though committed, and the issue's stale-estimate warning contradicts the re-derived Estimate block
    detail: |
      plan line 159 ("Step 5: Commit") is unchecked although e7c6c6e8 landed it —
      milestone-close's plan-unchecked guard will refuse on it. Separately, the
      issue's ## Revisions still says "⚠ The estimate is now stale" while ##
      Estimate has since been re-derived for the grown scope (3.58 → 6.20).
  - id: new
    severity: Minor
    family: result-shape-forces-new-consumer-arm
    title: |
      RelaunchResult does not compose with finishOperation's existing ParkResult/StartResult arms
    detail: |
      relaunch.go:43 — console.go:1324 and :1328 already know how to land a
      ParkResult address and force-switch onto a StartResult handle.
      RelaunchResult{Outcome, Record, Handle} matches neither, so M2 adds a third
      arm. RelaunchResult{Outcome, Start StartResult} would reuse the resume arm
      unchanged.
  - id: new
    severity: Minor
    family: manifest-ordering
    title: |
      relaunch.go inserted out of alphabetical order in NonArtifactSources
    detail: |
      artifactpath/manifest.go:524 places it between pathops.go and procops.go.
      Nothing enforces the order; noted so the next insert does not compound it.
```
