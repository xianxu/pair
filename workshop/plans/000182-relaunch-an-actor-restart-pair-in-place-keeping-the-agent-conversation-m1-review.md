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
