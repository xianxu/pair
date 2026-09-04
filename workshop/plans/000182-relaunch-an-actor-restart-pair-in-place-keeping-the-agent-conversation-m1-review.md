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

---

## Re-review — 2026-09-04T11:04:55-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 182 — Relaunch an actor: restart Pair in place, keeping the agent conversation |
| repo | pair |
| issue file | workshop/issues/000182-relaunch-an-actor-restart-pair-in-place-keeping-the-agent-conversation.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 4a7d96e2df70b9ad0fea2482bc2dc3d6f1816637..4a7d96e2df70b9ad0fea2482bc2dc3d6f1816637 |
| command | sdlc milestone-close --issue 182 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-04T11:04:55-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M1 deliverable itself is sound and well-pinned: `CheckResumePreconditions` is a genuinely pure predicate whose agreement with `DecideResume` is asserted over nine record shapes without IO, `Couch.Relaunch` raises every visible refusal before the destructive park, all four outcomes have tests, and `atlas/couch.md` carries the outcome table. Two of the four claimed fixes I verified by reverting them in a scratch worktree — both went red, so BR-2 and BR-3 are real. **BR-4 is not addressed**: `resumeEvidence` was created but has exactly one caller (`relaunch.go:99`), `ResumeContext` still derives the same three facts itself with a different nil-Path policy, and the helper's own doc comment asserts a sharing that does not exist at HEAD — the "fix reads as protection while consolidating nothing" case the checklist exists to catch. One new Important: the M2 key-layer commit `4821dda3` landed inside this round's range and makes `Alt+n` a dead key — the interceptor returns `HitRelaunch` and `processInput`'s switch has no arm for it, so the chord is stolen from the hosted Pair and dropped silently. Nothing is Critical; none of this blocks the boundary once disposed.

**A note on the window itself:** the pinned range was `4a7d96e2..4a7d96e2` — base equals head, so every recipe returned an empty diff. I reviewed `9acfd8e5..4a7d96e2` instead (the prior round's `Review-Window:` trailer records `88fe1de0..9acfd8e5`), which is what brought commit `4821dda3` into scope. Full `go test ./cmd/...` cannot be verified here: every failure is a pty-spawn `operation not permitted` from this agent environment, matching the known limitation; the couchcore/couchtty logic tests pass.

## 1. Strengths

- `cmd/internal/couchcore/precondition_test.go:89` — `TestResumePreconditionsMatchDecideResumeOnAPostParkRecord` asserts the *agreement* between the extracted predicate and `DecideResume` on a post-park transform, over nine shapes, and the `asParked` helper deliberately avoids `cloneThreadRecord` because `cloneArgv` would repair the "incomplete profile" case into a false agreement. That comment is the difference between a test and a real one.
- `cmd/internal/couchcore/relaunch_test.go:159-174` — the refusal test's core assertion is "nothing was destroyed" (no lifecycle trace, no triggered quits, revision unchanged), not "it refused". The post-setup baseline (`before`, line 143) instead of "still live" is correct: the not-running case's setup *is* removing the incarnations.
- `cmd/internal/couchcore/relaunch_test.go:259` — `TestRelaunchProceedsWhenOnlyPairsCleanupFailed` pins a genuinely non-obvious boundary (a `CleanupAttempt` failure lands in `ParkResult.CleanupError` with the park returning nil), and the comment records that it was found by asserting the opposite.
- `cmd/internal/couchcore/relaunch_test.go:364` — the dispatch-seam test runs through `DispatchOperation` with production executors and its comment records that a `couchcmd`-level attempt passed for a reason unrelated to what it claimed. That is the honest version of the pair#181 M3 `Tab → archive` lesson.
- `cmd/internal/couchtty/keys.go:255` — replacing `FeedHit`'s hand-written kind list with `kind.intercepts()` is the right generalisation, and the comment naming it "the third copy proving the point" is the kind of note that stops the fourth.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — `HitRelaunch` is intercepted and then dropped; `Alt+n` is a dead key inside couch.** `cmd/internal/couchtty/keys.go:94` declares it and `keys.go:255` intercepts it, but `console.go:601-610`'s switch has arms only for `HitSwitch`/`HitPark`/`HitPrevious`/`HitDetach` and no `default`. So the chord is consumed off the child's input stream and falls through with no effect, no notice, and no way for the operator to tell it from a wedged terminal. Before `4821dda3` it reached Pair and did the in-place reload; README.md:141 still documents that behaviour and `menu.go:19` `menuControls` has no `Alt+n` row.

> **This is the 2nd finding in family `declared-source-hand-maintained-consumers`.** Do NOT fix this instance. The rule: **a value the interceptor can emit must reach a handler by construction, not by a hand-maintained switch.** `intercepts()` and `hit()` derive; `processInput`'s switch (console.go:603-610) and the legacy branches at `keys.go:234,237` do not — three per-hit sites remain after the commit that claimed to close the class. Write the enumeration: a test that walks every `seqKind` through `hit()` plus the legacy branches and asserts each non-`HitNone` value routes to a console handler, so a declared-but-undispatched chord cannot ship. Fold this into the plan's Step 2b, which already commits to the operation-side enumeration — the two are the same rule at two seams.

**I-2 — see BR-4's `not-addressed` disposition below.** The consolidation the prior round asked for was not delivered; the helper has one caller and `ResumeContext` still carries the parallel derivation.

## 4. Minor findings

- `relaunch.go:96-110` — the binding is resolved before `CheckResumePreconditions` runs, so a thread with a missing profile still pays a `QuerySessionContext` scan. Harmless (the precedence is correct — the profile checks refuse first), but the ordering is the reverse of "cheapest refusal first".
- `menu.go:1320` `operationNeedsProjectionRefresh` already fail-safes to `true`, so its row in the plan's six-site table is a no-op site. Worth marking as such so the M2 sweep does not report six changes when five are real.
- `cmd/internal/couchcmd/run.go:556` — the CLI renderer's `default` arm would print a `RelaunchResult` as a Go struct dump. Unreachable today (`ExecuteLiveOwner` refuses first, as the seam test's comment records), but it is one routing change away.

## 5. Test coverage notes

- The four-outcome design exists so consumers switch on `Outcome` instead of parsing errors; `operation_queue.go:69` and `finishOperation` (console.go:1321-1327) do carry `value` alongside `err`, so partial results survive — but nothing pins that. The dispatch-seam test covers only the success path. One case asserting a `ParkedNotResumed` result reaches the caller through `DispatchOperation` *alongside* its error would make the contract enforced rather than observed.
- No test pins the nil-Path policy `resumeEvidence`'s doc comment states. Setting `c.Path = nil` and asserting `ResumePathMissing` would be two lines and would make the comment true.
- ARCH-MOCK passes cleanly: `FakeThreadArtifactCollisionChecker.ResolveEstablished` (artifactcollision_fake.go:116-127) mirrors `SessionInventoryNativeBindingResolver`'s contract exactly, returning the resolution alongside a typed `ResumeRefusal`. `erroringBindingArtifacts` (relaunch_test.go:390) models the real IO failure faithfully — a zero resolution plus an untyped error — which is why the BR-2 revert test is meaningful.

## 6. Architectural notes

- **ARCH-DRY — flag.** Two live instances: the `resumeEvidence` non-consolidation (BR-4) and the per-hit dispatch enumeration (I-1). Both are the same shape: a helper or predicate introduced as the single source, with a consumer left hand-maintained beside it.
- **ARCH-PURE — pass.** `CheckResumePreconditions`, `bindingResumeDiagnostic`, `isBindingDiagnostic`, `hasOccupiedIncarnation` are pure and tested without IO. `resumeEvidence` is correctly the IO seam and correctly a method on `*Couch`. `Couch.Relaunch` is thin glue over injected `PairLifecycle` + `Artifacts`.
- **ARCH-PURPOSE — flag (I-1).** The shadow-sweep on `Operations()` as a declared source: BR-5's plan correction enumerates the six operation-side consumers, which is the right answer. But the same round shipped a *seventh* hand-maintained consumer class at the hit layer, and shipped it half-wired. The class is "declaration sites whose consumers do not derive," and it now has two enumerations, only one of which is written down.
- **ARCH-MOCK — pass.** Both halves of relaunch run against the stateful `pairlifecycletest.Fake` and the artifact fake through the same seams production uses; `conformance_live_test.go` exists for the live check.
- **ARCH-CONSTRAINTS — flag (BR-1, unchanged).** Verified against the code: `couch.go:119` is the 15s `CompletionTimeout`, `couch.go:107` sets `resumeRegistrationTimeout` to 5s. The plan's "5s exact-child-death wait" is not a separate budget (child death is awaited *inside* the 15s window, `park.go:549-556`) and the "10s blocked-start acknowledgement" is 5s. Worst case ~20s, not ~30s. Over-budgeting, so no decision changes — but M2's spinner copy will be written against these numbers.
- **ARCH-SECURE — pass.** `validateThreadAddress` parses at the boundary; `hasOccupiedIncarnation` handles the empty-incarnation record; the resolver-IO fix (BR-2) is precisely the "degrade visibly rather than substituting a fabricated value downstream code reads as evidence" rule, and it now does.

## 7. Plan revision recommendations

- **`## Revisions` — envelope correction.** Replace the ARCH-CONSTRAINTS paragraph (plan lines 32-37) with the measured budgets, each citing its constant: `PairLifecycleController.CompletionTimeout` 15s (`couch.go:119`, covering both the completion wait and the exact-child-death wait, `park.go:549-556`) and `resumeRegistrationTimeout` 5s (`couch.go:107`, consumed at `launch_existing.go:109-111`). Worst case ~20s.
- **`## Revisions` — landed entities.** The M1 Core-concepts table never gained `resumeEvidence`, `hasOccupiedIncarnation`, or `ResumeNotRunning`, all of which shipped. pair#181 M3 recorded its unplanned entities this way; do the same.
- **Tick what landed.** Plan line 159 (Task 1 Step 5) and lines 472/476 (Task 8 Steps 1 and 2-5) are unchecked although `e7c6c6e8` and `4821dda3` landed them; `milestone-close`'s plan-unchecked guard will refuse on line 159.
- **Issue `## Revisions`.** Drop or amend "⚠ The estimate is now stale" — `## Estimate` has since been re-derived for the grown scope (3.58 → 6.20) and `estimate_hours: 6.20` matches it. The warning now contradicts the block above it.
- **Step 2b.** Extend it from the six operation sites to the hit-dispatch enumeration as well (see I-1), so one task owns the whole rule.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan lines 32-37 unchanged; couch.go:119 is the 15s CompletionTimeout and couch.go:107 sets resumeRegistrationTimeout to 5s, so ~20s worst case, not ~30s.
  - id: BR-2
    disposition: addressed
    note: |
      Verified by revert: replacing the bindingErr guard with `_ = bindingErr` turns "the resolver itself fails" red with resume-binding-unbound.
  - id: BR-3
    disposition: addressed
    note: |
      Verified by revert: deleting the hasOccupiedIncarnation gate turns "the thread is not running at all" red with resume-live.
  - id: BR-4
    disposition: not-addressed
    note: |
      resumeEvidence has ONE caller (relaunch.go:99); ResumeContext (resume.go:298-321) still derives pathExists, the NativeBindingResolver assert and agent itself, with a different nil-Path policy, and the helper's "Shared because" comment is false at HEAD.
  - id: BR-5
    disposition: addressed
    note: |
      Plan now carries the six-site table plus Step 2b, and the issue's M2 bullet names the sweep; I verified all six sites exist. Delivery is M2's.
  - id: BR-6
    disposition: not-addressed
    note: |
      relaunch_test.go still has five cases; no agent-unsupported or profile-missing case reaches Relaunch.
  - id: BR-7
    disposition: not-addressed
    note: |
      Plan:159 still unticked, and now Task 8's steps (472/476) too though 4821dda3 landed them; the stale-estimate warning still contradicts the re-derived block; the M1 table never gained resumeEvidence / hasOccupiedIncarnation / ResumeNotRunning.
  - id: BR-8
    disposition: not-addressed
    note: |
      relaunch.go:42-46 unchanged. finishOperation does carry value alongside err, so the third arm is still what M2 must add.
  - id: BR-9
    disposition: not-addressed
    note: |
      manifest.go:524 still places relaunch.go between pathops.go and procops.go.
findings:
  - id: new
    severity: Important
    family: declared-source-hand-maintained-consumers
    title: |
      Alt+n is intercepted and then silently dropped: HitRelaunch has no arm in processInput's switch
    detail: |
      2nd finding in this family, so the deliverable is the RULE, not this site.
      keys.go:94 declares HitRelaunch and keys.go:255 intercepts both chords, but
      console.go:603-610 handles only Switch/Park/Previous/Detach and has no
      default -- so the chord is consumed off the child's input stream and does
      nothing. Before 4821dda3 it reached Pair and reloaded the workbench;
      README.md:141 still documents that, and menu.go:19 menuControls has no
      Alt+n row. Rule: a value the interceptor can emit must reach a handler by
      construction. Three per-hit sites still enumerate by hand (console.go's
      switch, keys.go:234, keys.go:237) after the commit that claimed to close
      the class. Write the enumeration -- a test walking every seqKind through
      hit() plus the legacy branches, asserting each non-HitNone value routes --
      and fold it into Step 2b, which already owns the operation-side half of the
      same rule. ARCH-DRY, ARCH-PURPOSE.
  - id: new
    severity: Minor
    family: review-window-degenerate
    title: |
      The gate handed this round a base == head window, so every diff recipe returned empty
    detail: |
      base and head were both 4a7d96e2, so stat, name-status and full diff were
      all empty and a reviewer following the recipes literally would have had
      nothing to inspect. I reviewed 9acfd8e5..4a7d96e2 instead, derived from the
      prior round's `Review-Window: 88fe1de0..9acfd8e5` trailer -- which is what
      brought the unreviewed commit 4821dda3 into scope. Worth a look at how the
      boundary computes BASE_SHA when the previous round's fix commit is HEAD.
```

---

## Re-review — 2026-09-04T13:55:31-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 182 — Relaunch an actor: restart Pair in place, keeping the agent conversation |
| repo | pair |
| issue file | workshop/issues/000182-relaunch-an-actor-restart-pair-in-place-keeping-the-agent-conversation.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 4a7d96e2df70b9ad0fea2482bc2dc3d6f1816637..4ae9d278e40a3153bedbab8110b34986cdcb0d55 |
| command | sdlc milestone-close --issue 182 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-04T13:55:31-07:00 |
| verdict | REWORK |

## Review

I inspected the pinned range, ran the affected suites, and verified three claimed fixes by reverting them in place (restoring the tree each time). Two new defects are reproducible with tests I wrote and ran.

```verdict
verdict: REWORK
confidence: high
```

The M1 operation itself remains sound, and three of this round's four claimed fixes are real and mutation-verified: removing `case HitRelaunch` (console.go:614) turns `TestRelaunchChordBytesFromAnActorReachTheConfirmation` red, and reverting `StartedChild` to the concrete `StartResult` assertion turns `TestRelaunchResultIsAdoptedByTheConsole/a_completed_relaunch_is_adopted` red. What blocks the boundary is two reachable defects the round introduced. **C-1:** `menuActionItems` now offers `relaunch` in the Tab→actions list, but `reduceActionKey`'s Enter switch (`menu.go:549`) has no arm for it and no default — selecting it and pressing Enter produces no frame, no effect, and no notice. That is the issue's *first* Done-when bullet shipped half-wired, and it is the third instance of the family the last two rounds escalated: the same round that fixed the dead `Alt+n` chord shipped a dead action-list row. **C-2:** the `resumeEvidence` consolidation (which does address BR-4) made `ResumeContext`'s **warm/detached** branch call `ResolveEstablished`, which it deliberately never did. I proved the drift differentially — the same test passes at `4a7d96e2` and fails at `4ae9d278`: a warm reattach now aborts when the binding resolver hits a real IO error, on the exact path pair#181 M2 exists to keep reachable.

## 1. Strengths

- **`cmd/internal/couchtty/console_relaunch_chord_test.go:21`** — `newChordFixture` drives real bytes through `Run`'s own input loop over an `io.Pipe`, which is the only layer that could have seen the dead chord. Its comment says exactly that, and reverting the dispatch arm confirms it fires. This is the right correction to "every prior test stopped at the Interceptor."
- **`cmd/internal/couchtty/menu.go:1063-1080`** — building the confirmation item as `frame.Action + " " + label` makes the "first word IS the dispatch id" invariant structural instead of a per-case string. The two halves of the operator's bug (mislabelled screen, and Enter refusing at the `id == frame.Action` guard) collapse into one fix, and `TestRelaunchConfirmationNavigatesAndDispatchesRelaunch` pins both — including `\x1bOB` application-cursor mode, which is how the operator actually arrives from an nvim pane.
- **`cmd/internal/couchcore/ops.go:111-125`** — `StartedChild` is the right generalisation of BR-8: adoption became a property of the result rather than a third type-switch arm, `RelaunchResult.Started()` correctly returns `false` for the three non-success outcomes, and the table test covers all four.
- **`cmd/internal/couchtty/menu_inflight_frame_test.go:49`** — the second subtest ("not in flight, a dead thread's confirmation is discarded") is what makes the in-flight exemption a scoped exemption rather than a hole. Most fixes of this shape ship only the positive case.
- **`workshop/lessons.md:3225-3228`** — the "probe first when a name and an intent disagree" entry, recording that `restoreMenuPrefixPreservingStart` truncates rather than restores and that relaunch's absence from the park/leave list was deliberate. A near-miss that would have shipped a regression as a cleanup, written down.

## 2. Critical findings

**C-1 — `relaunch` is offered in the switcher's action list and Enter on it does nothing.** `cmd/internal/couchtty/menu.go:549` (the Enter switch) vs `menu.go:1027` (the item).

`menuActionItems` returns `{"detach","relaunch","park","name","describe"}`, but `reduceActionKey`'s `switch frame.SelectedItem` routes only `park`/`archive` (confirmation), `detach`/`resume` (dispatch), `name`/`describe` (text frame). `relaunch` falls through a switch with no default and the reducer returns `state, nil`. I confirmed it with a scratch test through `ReduceMenu`: after Tab → Down, `SelectedItem == "relaunch"`; after Enter, `frame.Kind` unchanged, `frame.Action == ""`, `Notice == ""`, `effects == []`. The smoke test missed it because it only exercised `Alt+n`.

> **This is the 3rd finding in family `declared-source-hand-maintained-consumers`.** Do NOT fix this instance. The rule: **an item a surface can OFFER must be routed by construction.** `menuActionItems` and `reduceActionKey`'s Enter switch are two hand-written lists that must agree and are checked by nothing; the same shape produced BR-10 one seam over. Measured prevalence this round: the plan's table names **six** per-operation sites; the sweep actually touched **twelve** (`menu.go:467`, `:502`, `:506`, `:609`, `:1027`, `confirmationMenuItems`, `:1258`, `:1308`, `:1337`, `:1351`, `:1462`; `console.go:1467`, `:1487`), and **two more remain unswept** — `menu.go:549` (this Critical) and `console.go:1445`, whose landing arm is still a literal `== "resume"` even though `RelaunchResult` now satisfies `StartedChild` and needs the same force-switch. Extend Step 2b's enumeration test from "every `PresentationTUI` operation appears in `menuActionItems`" to "…**and** Enter on it yields a dispatch effect or a frame" — one loop over `menuActionItems(row)` per row state, asserting no item is a silent no-op. That test catches C-1, `console.go:1445`, and the next one. ARCH-DRY, ARCH-PURPOSE.

**C-2 — the `resumeEvidence` consolidation made the warm/detached reattach resolve a binding it deliberately never resolved, and it now fails when that resolver errors.** `cmd/internal/couchcore/resume.go:347-353`, helper at `:153-170`.

`ResumeContext`'s `else` branch (`VerifiedPark == nil`) now calls `c.resumeEvidence`, which always calls `ResolveEstablished`, and **discards the binding** — it wants only `pathExists`. It tolerates a typed `*ResumeRefusal` but returns any other error. At `4a7d96e2` that branch called the resolver zero times. Differential proof, same test file in a scratch worktree at base and at HEAD:

```
base 4a7d96e2: --- PASS: TestScratchWarmReattachSurvivesAResolverIOFailure
head 4ae9d278: --- FAIL: warm reattach refused because the binding
               resolver failed: resolve home directory: permission denied
```

The comment three lines above still asserts the old contract ("Resolve the binding only where it is the authority… asking on the warm path refused the thread here… which is why relaxing that alone left the operator's detached thread unreachable"), so the code and its stated contract now disagree. Reachable causes are ordinary, not exotic: `QuerySessionContext` returns an error from `runtime.ListFiles` (`ErrStorageAbsent` when the Pair data root is missing) and from `runtime.ReadFile(ledger, 8<<20)` (`ErrReadLimit` once a thread's ledger history sidecar passes 8 MB — `sessioninventory/runtime_os.go:157`), besides `os.UserHomeDir`. Nothing pins either direction: `TestDetachedResumeDoesNotRequireAnEstablishedBinding` is a pure `DecideResume` test and never reaches `ResumeContext`.

Fix sketch: split the helper along the axis its callers actually differ on — a `pathExists(thread)` (or `resumeEvidence` returning path-only when the caller says the binding is not its authority) so the warm branch pays no resolver call, keeping one nil-`Path` policy in one place. Then pin it: a warm-reattach case with `erroringBindingArtifacts` asserting success, and a cold case asserting the error still surfaces. ARCH-CONSTRAINTS (the warm path now pays a `ListFiles` + up-to-8 MB ledger read + proof validation + a possible `PublishSessionInventoryValidations` **write**, and throws the answer away), ARCH-PURE (one helper bundling two independent observations forces every caller to take both failures).

## 3. Important findings

**I-1 — `COUCH_INPUT_TRACE` is a new operator-facing surface that records every keystroke, and it is documented nowhere.** `cmd/internal/couchtty/inputtrace.go:59`.

A grep of the whole tree finds the name only in `inputtrace.go`, one comment in `console.go:129`, and the issue Log — not in `README.md`, not in `atlas/couch.md`, not in `menuControls`. Two things need saying in the same place: that it exists and how to turn it on, and **what it captures**. `pumpStdin` traces every chunk before the Interceptor splits it, so the file contains everything the operator types or pastes into the hosted agent, prompts and credentials included. `0600` and opt-in are the right defaults; the missing piece is telling the operator, at the moment they enable it, that the file is a full keystroke log. Add the env var to `atlas/couch.md`'s couch-env paragraph (beside `COUCH_STORE_DIR`/`COUCH_THREAD_*`) with that sentence. ARCH-SECURE, and the Docs update gate.

## 4. Minor findings

- `cmd/internal/couchtty/inputtrace.go:58-67` — `newInputTracer` returns `nil` when the file cannot be opened, so an unwritable `COUCH_INPUT_TRACE` path produces an empty trace that reads as "the terminal sent nothing" — the exact ambiguity the probe exists to remove. **2nd in family `swallowed-cause-fabricated-diagnostic`**; the rule below, not this site.
- `cmd/internal/couchtty/menu.go:505-506` — the notice suffix comes from an inline `map[string]string{"park":…,"relaunch":…}` indexed by `event.Operation`, parallel to the `||` condition one line above. A third operation added to the condition and not the map yields "only a running thread can be " with an empty tail.
- `cmd/internal/couchcore/ops.go:121` — no `var _ StartedChild = RelaunchResult{}` / `= StartResult{}`, and no test walking the `ResultStart`-declared operations. A `Started()` moved to a pointer receiver would silently stop satisfying the interface.
- `cmd/internal/couchtty/inputtrace.go:76` — `record` does a synchronous `Fprintf` under a mutex on the keystroke pump; a slow or full filesystem stalls operator input. Opt-in, so acceptable, but it is the tightest latency path in the console.
- `atlas/couch.md:329-330` — "…attach failure aborts the exact newly started actor" runs well past the file's wrap column where the surrounding paragraph was re-flowed.

## 5. Test coverage notes

- Mutation-verified this round: `HitRelaunch` dispatch arm ✅ (removing it reds two tests), `StartedChild` adoption ✅ (reverting to the concrete assertion reds the completed-relaunch case). Not verifiable by revert: the `resumeEvidence` consolidation — no test changes behavior when `ResumeContext` re-derives its own evidence, which is expected for a structural DRY fix but is also why C-2 slipped through.
- `cmd/internal/couchtty` non-pty tests pass; `cmd/internal/artifactpath` passes. Full `go test ./cmd/...` still cannot be confirmed here: every failure is `ptychild: start sh: operation not permitted` / `open pty: operation not permitted` from this agent environment, the known limitation, unrelated to the diff.
- The nil-`Path` policy `resumeEvidence`'s doc comment states is still unpinned — two lines (`c.Path = nil`, assert `ResumePathMissing`) would make the comment true, and it is now load-bearing for two callers rather than one.
- `TestRelaunchResultIsAdoptedByTheConsole` calls `finishOperation` directly, so it proves adoption but not the *landing*. Nothing asserts where the operator ends after a successful relaunch — which is how `console.go:1445`'s `== "resume"` literal stayed unnoticed.

## 6. Architectural notes

- **ARCH-DRY — flag (C-1).** Real wins: `endsItsOwnChild` collapses the two exit lists, `bindingRefusalDiagnostic` derives the sentence from the same code that chose the refusal, `confirmationMenuItems` builds from `frame.Action`. But the round's own evidence is that the class is bigger than the plan's table: twelve sites swept against six enumerated, two still unrouted.
- **ARCH-PURE — pass, with C-2's note.** `renderInputBytes`, `bindingRefusalDiagnostic`, `endsItsOwnChild`, `CheckResumePreconditions` and the menu reducers are pure and tested without IO; `inputTracer` is correctly the thin shell. The one flaw is `resumeEvidence` bundling two independent observations into one IO call.
- **ARCH-PURPOSE — flag (C-1).** Done-when bullet one is "an actor action `relaunch` appears alongside detach and park, confirmed like park, and reachable from the same declared-operation surface." It appears and it is not reachable. Separately, BR-10 asked for the class enumeration and got the instance; the class then produced C-1 in the same round, which is the ledger reporting exactly what the rule predicts.
- **ARCH-MOCK — pass, and well done.** `erroringBindingArtifacts` models the real IO failure (zero resolution + untyped error), the chord tests run production `Run` against `hostty.NewFakeHost` + `ptychild.NewFakeChild`, and park failures still come from the stateful fake's state. Production and test flow share the same boundary.
- **ARCH-CONSTRAINTS — flag.** C-2 adds unbudgeted repeated IO (and a possible catalog write) to the warm path. BR-1's envelope error is unchanged. `inputTracer.record` sits on the keystroke path with no bound.
- **ARCH-SECURE — flag (I-1).** The new keystroke log is opt-in and `0600`, which is right; what is missing is the operator-facing statement of what lands in it. Otherwise clean: refusal messages carry no sensitive data, and the new `bindingRefusalDiagnostic` strings are all static.
- Not a finding, recorded: after a successful relaunch the operator ends in the **switcher**, not on the actor — `onRelaunchHotkey` sets `FocusPanel()` and `finishOperation`'s force-switch arm is keyed on `"resume"`. Deliberate and commented, owned by M2 Task 9/10 Step 3.

## 7. Plan revision recommendations

1. **The six-site table under-counts by half.** Replace it with the measured twelve, and mark `menu.go:549` (action-list Enter dispatch) and `console.go:1445` (the `== "resume"` landing arm) as the two still open. Note that `operationNeedsProjectionRefresh` already fail-safes to `true` and is a no-op site, so the sweep should not report it as a change.
2. **Step 2b's enumeration test must assert routing, not presence.** "Every `PresentationTUI` operation appears in `menuActionItems`" would have passed while C-1 shipped. Add "and Enter on each offered item yields a dispatch effect or a frame," and the hit-layer half BR-10 asked for.
3. **Tick what shipped, and say so.** Task 1 Step 5 (line 159), Task 8 Steps 1 and 2-5, Task 10 Steps 2 and 3 (partially), and Task 11's real-stack verification all landed; `milestone-close`'s plan-unchecked guard will refuse, and the plan currently understates M1's delivered scope by an entire milestone's worth of tasks. Add a `## Revisions` entry recording that M2's key layer and menu sweep landed inside M1's window.
4. **Fix the operating envelope (BR-1).** Every duration must cite the constant producing it: `CompletionTimeout` 15s (`couch.go:119`) covers child death; `resumeRegistrationTimeout` is 5s (`couch.go:107`), not 10s. Worst case ~20s.
5. **Add `resumeEvidence`, `hasOccupiedIncarnation`, `ResumeNotRunning` and `StartedChild` to the M1 Core-concepts table**, and record C-2's split so the helper's shape is decided in the plan rather than at the next boundary.
6. **In the issue**, drop the `## Revisions` "⚠ The estimate is now stale" paragraph — `## Estimate` was re-derived to 6.20 for the grown scope and the warning now contradicts the block it points at.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan lines 32-37 unchanged; the plan file was not touched in this window at all.
  - id: BR-4
    disposition: addressed
    note: |
      resumeEvidence now has two callers and ResumeContext derives none of the three facts itself; the consolidation's own side effect on the warm path is raised separately as a new Critical.
  - id: BR-6
    disposition: not-addressed
    note: |
      relaunch_test.go still has five refusal cases; no agent-unsupported or profile-missing case reaches Relaunch.
  - id: BR-7
    disposition: not-addressed
    note: |
      plan:159 still unticked; Task 8 and parts of Task 10/11 now shipped and unticked too; the issue's stale-estimate warning still contradicts the re-derived Estimate block.
  - id: BR-8
    disposition: addressed
    note: |
      StartedChild generalises the arm instead of adding a third; verified by revert — restoring the concrete StartResult assertion reds TestRelaunchResultIsAdoptedByTheConsole/a_completed_relaunch_is_adopted.
  - id: BR-9
    disposition: not-addressed
    note: |
      manifest.go:524 still places relaunch.go between pathops.go and procops.go; the new inputtrace.go entry was inserted in order, so the convention is being followed for new rows only.
  - id: BR-10
    disposition: not-addressed
    note: |
      The instance is fixed and genuinely pinned (removing the case reds two byte-driven tests), but the requested deliverable was the enumeration and no test walks seqKind/InterceptorHit to a handler; the same class then shipped a new Critical at menu.go:549. README.md:141 and menuControls are also still unchanged.
  - id: BR-11
    disposition: addressed
    note: |
      This round's window is a real range — 4a7d96e2..4ae9d278, 10 commits, 18 files; every diff recipe returned content.
findings:
  - id: new
    severity: Critical
    family: declared-source-hand-maintained-consumers
    title: |
      relaunch is offered in the switcher's action list and Enter on it does nothing
    detail: |
      3rd finding in this family, so the deliverable is the RULE, not this site.
      menuActionItems (menu.go:1027) now returns "relaunch", but reduceActionKey's
      Enter switch (menu.go:549) routes only park/archive, detach/resume and
      name/describe and has no default -- so Enter on the row produces no frame,
      no effect and no notice. Confirmed with a scratch test through ReduceMenu:
      after Tab then Down, SelectedItem == "relaunch"; after Enter, frame.Action
      == "" and effects == []. That is the issue's FIRST Done-when bullet shipped
      half-wired, and the smoke test missed it because it drove only Alt+n.
      Rule - an item a surface can OFFER must be routed by construction.
      Measured prevalence this round - the plan names six per-operation sites,
      the sweep touched twelve (menu.go 467, 502, 506, 609, 1027,
      confirmationMenuItems, 1258, 1308, 1337, 1351, 1462; console.go 1467,
      1487), and two remain - menu.go:549 and console.go:1445, whose landing arm
      is still a literal == "resume" though RelaunchResult now satisfies
      StartedChild. Extend Step 2b's enumeration from "appears in menuActionItems"
      to "and Enter on it yields a dispatch effect or a frame". ARCH-DRY,
      ARCH-PURPOSE.
  - id: new
    severity: Critical
    family: shared-helper-widens-caller-contract
    title: |
      The resumeEvidence consolidation makes a warm reattach resolve a binding it never resolved, and fail when that resolver errors
    detail: |
      resume.go:347-353 - the VerifiedPark == nil branch now calls resumeEvidence,
      which always calls ResolveEstablished and whose binding it then DISCARDS; it
      wanted only pathExists. Any non-ResumeRefusal error now aborts the warm
      reattach. At 4a7d96e2 that branch called the resolver zero times. Proved
      differentially with one test in a scratch worktree - PASS at base, at HEAD
      "warm reattach refused because the binding resolver failed: resolve home
      directory: permission denied". The comment three lines above still states the
      old contract ("Resolve the binding only where it is the authority ... which
      is why relaxing that alone left the operator's detached thread
      unreachable"), so code and contract now disagree, and nothing pins either
      direction - TestDetachedResumeDoesNotRequireAnEstablishedBinding is a pure
      DecideResume test. Reachable causes are ordinary - ListFiles ErrStorageAbsent
      and ReadFile ErrReadLimit once a ledger sidecar passes 8 MB
      (sessioninventory/runtime_os.go:157), besides os.UserHomeDir. Split the
      helper along the axis the callers differ on so the warm branch pays no
      resolver call, keeping ONE nil-Path policy; pin both directions.
      ARCH-CONSTRAINTS (the warm path now pays a ListFiles, an up-to-8MB read,
      proof validation and a possible catalog write, and throws the answer away).
  - id: new
    severity: Important
    family: new-surface-undocumented
    title: |
      COUCH_INPUT_TRACE is a new operator surface that records every keystroke and is documented nowhere
    detail: |
      inputtrace.go:59. A tree-wide grep finds the name only in inputtrace.go, one
      console.go comment and the issue Log - not README.md, not atlas/couch.md, not
      menuControls. Two things need saying together - that it exists and how to
      enable it, and WHAT it captures. pumpStdin traces every chunk before the
      Interceptor splits it, so the file holds everything typed or pasted into the
      hosted agent, prompts and credentials included. 0600 and opt-in are the right
      defaults; the gap is telling the operator at the moment they enable it. Add
      it to atlas/couch.md beside COUCH_STORE_DIR / COUCH_THREAD_*. ARCH-SECURE.
  - id: new
    severity: Minor
    family: swallowed-cause-fabricated-diagnostic
    title: |
      newInputTracer swallows its open error, so a failed probe reads as "the terminal sent nothing"
    detail: |
      2nd finding in this family, so the deliverable is the RULE, not this site.
      inputtrace.go:58-67 returns nil when os.OpenFile fails, so an unwritable
      COUCH_INPUT_TRACE path yields an empty trace indistinguishable from "no bytes
      arrived" - the exact ambiguity the probe exists to remove, and the same shape
      as BR-2's zero resolution read as "unbound". Rule - an instrument that cannot
      start must say so; the INABILITY to observe must never be presentable as an
      observation. A one-line setNotice on the open failure satisfies it without
      letting a diagnostic take the console down.
```
