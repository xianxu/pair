# Boundary Review — pair#182 (whole-issue close)

| field | value |
|-------|-------|
| issue | 182 — Relaunch an actor: restart Pair in place, keeping the agent conversation |
| repo | pair |
| issue file | workshop/issues/000182-relaunch-an-actor-restart-pair-in-place-keeping-the-agent-conversation.md |
| boundary | whole-issue close |
| milestone | — |
| window | 88fe1de011b4c6be58e5a8b20eed89dfa4000f5d..94543ef8d56eeb2f2cdc26f31617cce8adc26dc4 |
| command | sdlc close --issue 182 |
| reviewer | claude |
| timestamp | 2026-09-04T16:37:19-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The relaunch operation itself is the strongest work in this branch: the order-of-checks design is real, it is enforced by tests that assert a refused relaunch performed *zero* lifecycle work, and the five defects the operator smoke test found were each fixed at their class rather than their site (`StartedChild`, `endsItsOwnChild`, `OperationConfirms`, `pastParticiple`, the in-flight frame exemption). I revert-verified the round-8 claimed fixes: deleting the `HitRelaunch` arm reds `TestRelaunchChordBytesFromAnActorReachTheConfirmation`, and `refuseBinding` is now genuinely the only constructor for a binding refusal at all three sites. The suite is green apart from the known environment-blocked pty/`fork exec` tests. What does not ship clean is the **record**: the `pair#186` split was written into exactly one artifact (the issue's `## Revisions`) and propagated to none of the three others that restate the same scope — the plan doc still declares `paneState`/`RenderHoldingPane`/`holding.go` as this milestone's deliverables, the project file still describes M2 as "the gesture and a surface that outlives its child", and the issue's own `## Done when` still claims the operator ends on the same actor when `onRelaunchHotkey` deliberately does the opposite. None of that blocks the gate; all of it makes the close record false.

## 1. Strengths

- **`relaunch.go:105-137` — the order of the checks is the design, and it is tested as such.** Preconditions → `soleParkableIncarnation` → park → resume, with `TestRelaunchRefusesBeforeParkingWhenTheResumeCouldNotSucceed` asserting not just the outcome but that `env.Lifecycle.trace` is empty and no quits were triggered. A destructive-first composition proved by absence of effect is the right shape.
- **`resume.go:190-215` — `CheckResumePreconditions` extracted with `DecideResume` calling it**, plus `TestResumePreconditionsMatchDecideResumeOnAPostParkRecord` walking nine shapes and asserting the two callers agree. The anti-drift test, not merely the extraction — this is what stops the `pair#181` archive-guard divergence recurring.
- **`ops.go:111-125` — `StartedChild` makes adoption a property of the result.** A concrete `StartResult` assertion left relaunch's child spawned and unadopted; generalising to an interface (with `StartResult.Started()` returning itself) is the correct fix, and `TestRelaunchResultIsAdoptedByTheConsole` pins all four outcomes including the two that must *not* attach.
- **`console_relaunch_chord_test.go:22-45` — a fixture that drives real bytes through `Run`'s input loop.** Every prior test stopped at the `Interceptor` and could not see a consumed-then-dropped chord. Revert-verified: removing the switch arm fails it in 6s.
- **`relaunch.go:95-103` — the parked-row refusal names the state it is actually in.** `hasOccupiedIncarnation` split out so a parked highlighted row gets "is not running, so there is no Pair to restart; Enter resumes it" instead of `resume-live` describing a state it is not in.
- **`atlas/couch.md` on `COUCH_INPUT_TRACE`** states that the tap sits upstream of the interceptor and therefore captures prompts and pasted secrets — the disclosure comes before the how-to, which is the right order for a debugging instrument (ARCH-SECURE).

## 2. Critical findings

None.

## 3. Important findings

**I-1 — The `pair#186` split was recorded in one artifact and propagated to none of its three consumers.** (`workshop/plans/000182-relaunch-an-actor-plan.md`, `workshop/projects/couch.md:201`, issue `## Done when`)

This is the 2nd finding in family `plan-record-lags-code` (BR-7 is the first and is still open), so the deliverable is the rule, not the site. **Rule: a scope change is not recorded until every artifact that restates the scope either derives from, or cites, the one entry that made it.** The enumeration is four artifacts and it is writable today:

- The plan has **no `## Revisions` section at all**, in violation of AGENTS.md §1. Its M2 Core-concepts table still lists `paneState` and `RenderHoldingPane` (`cmd/internal/couchtty/holding.go`) as `new` — neither symbol nor the file exists anywhere in the tree — and lists `onExit` as modified "non-fatal for a holding pane", which is not what was done to it. Tasks 8, 10, 11 and 12 carry unticked steps although Task 8 and Task 10 Step 2b shipped and Tasks 9/11/12 moved out. `milestone-close`'s plan-unchecked guard will refuse on this.
- `workshop/projects/couch.md:201` still reads `- [ ] the gesture and a surface that outlives its child [pair#182 M2]`, and there is no `pair#186` row anywhere in the project.
- The issue's `## Done when` still claims *"the operator ends on the same actor, not in the switcher"*. `console.go:1383-1387` sets `c.focus = FocusPanel()` unconditionally on an actor-focused `Alt+n` — the comment says so ("until the holding surface exists to keep the operator in place") — and `finishOperation` has no relaunch arm in the resume force-switch at `console.go:1552`, so the operator ends in the switcher. The `## Revisions` delta lists the pane, the two consequences, the key docs and the rebuilt-binary verification as moved; it does not list this bullet.
- BR-7's second half is unchanged: `## Revisions` still says "⚠ The estimate is now stale" while `## Estimate` has since been re-derived to 6.20.

Separately: `## Plan` marks M2 `- [x]` but no commit in the range carries a `Review-Verdict:` trailer for an M2 `milestone-close`, and `## Log` has no `closed M2` line. Either run `sdlc milestone-close --issue 182 --milestone M2`, or waive `--no-verdict` and say in `--verified` that this close review *is* M2's boundary.

**I-2 — Two guards written to prevent exactly this class have a hand-maintained list as their source, so the new chord and the new operation both walked past them.** (`cmd/internal/couchtty/menu.go:19`, `cmd/internal/couchtty/menu_action_sweep_test.go:16`)

This is the **7th finding in family `declared-source-hand-maintained-consumers`**. Earlier rounds fixed instances. Do not fix these two sites — state and fix the rule. **Rule: a guard whose source is a hand-maintained list is a guard the next addition can skip. The guard must derive from the same table the production path consumes.** Two live instances, both in this window:

- `TestREADMEDocumentsEveryPanelControl` (`couchcmd/readme_test.go:61`) exists so "adding a key to the UI makes this test fail until its operator documentation has a home in README" — its own comment. It iterates `couchtty.MenuControls()`. `Alt+n` and `Ctrl+Alt+n` were added to `knownSequences` (`keys.go:170-171`) and never to `menuControls`, so the guard could not fire, and `README.md:141` still tells the operator `Alt+n` in "any pane" reloads pair in place — which is now false inside couch. The fix is to derive the guard from the chord rows the interceptor actually consumes, so a new `seqKind` cannot reach the operator without a README row.
- `TestEveryOfferedActionIsReachableFromEnter` tests the **converse** of what the plan's Task 10 Step 2b specified. The plan asked for a test that "walks `Operations()` asserting every `PresentationTUI` operation the switcher can reach appears in `menuActionItems`"; the delivered test walks `menuActionItems` and asserts each offered action is reachable. Offered⇒reachable cannot catch declared-but-unreachable — the exact failure it was written for — so a new declared row action can still ship invisible with the suite green. The issue's `## Plan` M2 line claims this test lands "the six-site sweep that makes the declared operation actually reachable", which overstates it.

The cheap version of the fix is a field on `Operation` naming row-action membership, with `menuActionItems` deriving membership (ordering can stay hand-authored); that also collapses several of the seven restatements the round-8 note enumerated (`consumeExpectedParkExitLocked`, `operationNeedsProjectionRefresh`, `reduceOperationResult`'s case list, `reduceParkHotkey`'s case list, `menuOperationProgressText`, `pastParticiple`, `menuActionItems`), which are still seven.

## 4. Minor findings

- **`onRelaunchHotkey`'s panel branch (`console.go:1378-1380`) has zero tests.** Done-when bullet "`Alt+n` on the switcher relaunches the highlighted row and leaves the operator in the switcher" and plan Task 10 Step 3 ("two tests, because the endings differ") are both unmet; only the actor branch is driven. 2nd in family `done-when-untested` — the rule's enumeration is the Done-when list itself: every bullet the issue still claims cites either the test that pins it or the `## Revisions` line that moved it.
- **The review window handed to this round is the whole 111-commit branch** (base `88fe1de` = branch point; 164 files, +24177/−4669), containing `pair#170`, `#181` and `#185` work that already passed its own close gates. 2nd in `review-window-degenerate` (BR-11 was the empty variant of the same rule): the window should be the un-reviewed span — M1's close at `b7ec5e64`, or at most the first `#182` commit. I reviewed `b18f958e..HEAD` for code.
- `NonArtifactSources` ordering (BR-9) is unfixed and has now grown a second offender (`threadreason.go` at manifest.go:503, `relaunch.go` at :524; `menu_render.go`/`menu_refresh.go` at :559-560 predate this window). The family closes permanently with one `sort.SliceIsSorted` assertion over the manifest lists.

## 5. Test coverage notes

- `env -u PAIR_SESSION_ID -u PAIR_TAG go test ./cmd/...`: every failure is the known environment restriction (`ptychild: start …: operation not permitted`, `fork/exec /bin/ps: operation not permitted`) in `couchcore`/`couchtty`/`couchcmd`/`hostty`/`ptychild`/`pair-go`. `couchtty` has exactly one failure, `TestNotificationPTYConformance`, same cause. No logic failure in the diff.
- The relaunch suite drives the real `PairLifecycleController` against `pairlifecycletest.Fake` plus the fake artifact/proc/runner seams, and asserts the resumed child's *env* carries the original native session id — that is the conversation-continuity claim proved rather than asserted (ARCH-MOCK).
- BR-23's mutex fix has no test, which is inherent: it is a latent race with no reproducer today. `make test-race` is the only pin available; the field access is now correct.
- BR-6 remains: `relaunch_test.go` covers five refusals, not the agent-unsupported / profile-missing pair; those are pinned only at `CheckResumePreconditions`, so the diagnostic precedence through `Relaunch` is unpinned.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag (I-2).** The consolidations this diff *did* make are genuine and well-motivated; the flag is the two guards whose source is a hand-maintained list, plus the seven per-operation restatements still standing.
- **ARCH-PURE — pass.** `CheckResumePreconditions`, `RelaunchOutcome`/`RelaunchResult`, `bindingResumeDiagnostic`/`bindingRefusalDiagnostic`, `endsItsOwnChild`, `pastParticiple`, `clearsPreviousNotice`, `Feed.Row` and `renderInputBytes` are pure and unit-tested with no IO. `Couch.Relaunch` is a thin ordered shell over injected seams; `onRelaunchHotkey` only reduces the menu.
- **ARCH-PURPOSE — flag (I-1).** The operation and the gesture both ship and are real; the split is defensible engineering. What under-delivers is the record, and one Done-when bullet the code contradicts.
- **ARCH-MOCK — pass.** `refuseBinding` as the sole constructor gives the real resolver and its stateful fake one message per status by construction, and `TestTheResolverAndItsFakeRefuseWithTheActionableSentence` pins the fake's half.
- **ARCH-CONSTRAINTS — flag (BR-1, Minor).** Verified against the constants: `CompletionTimeout` is 15s (`couch.go:119`) and `resumeRegistrationTimeout` is 5s (`couch.go:107`), so the worst case is ~20s, not the plan's ~30s from a "5s exact-child-death wait" and a "10s blocked-start acknowledgement" that do not exist. Runtime behaviour is otherwise sound: capacity-one queue, no fan-out, nothing blocking on the keystroke path.
- **ARCH-SECURE — pass.** `COUCH_INPUT_TRACE` is the only new surface: opt-in, `0600`, path injected from the composition root rather than read in the constructor, open failure surfaced at control priority instead of degrading to a silent empty file, closed in `teardown`. Relaunch parses no new persisted format.
- **For `pair#186`:** the `expectedExits` bridge is gated on `err == nil` (`console.go:1521`), so a `ParkedNotResumed` relaunch — child dead, no replacement — can still raise a spurious exit notice if the completion wins the race. `#186`'s "with no exit there is nothing to suppress" reasoning covers the success path only; the park-ok-resume-failed path ends with a holding pane that has no child coming. Worth deciding there rather than discovering it.

## 7. Plan revision recommendations

Add to `workshop/plans/000182-relaunch-an-actor-plan.md` (the file currently has no `## Revisions` section):

```
## Revisions

### 2026-09-04 — M2's holding surface moves to pair#186

**Reason.** M1 is delivered and smoke-tested on the real stack; the branch was
111 commits. Carrying a working feature behind an unstarted surface is how a
branch becomes unreviewable.

**Delta.**
- Chunk 2's Core concepts table: `paneState` and `RenderHoldingPane`
  (`couchtty/holding.go`) are REMOVED — they were never written and belong to
  pair#186. `onExit`'s row is corrected: it was modified for publishNotice, not
  for a holding pane. `finishOperation`'s row is corrected likewise.
- Task 9 (a pane that outlives its child), Task 10 Steps 1 and 3, Task 11
  (rebuilt-binary verification) and Task 12 Step 1 (menuControls + README +
  atlas key docs) move to pair#186.
- Task 8 and Task 10 Step 2b are DELIVERED here. Step 2b's specified test —
  walk Operations() asserting every switcher-reachable PresentationTUI
  operation appears in menuActionItems — was NOT written; the delivered
  TestEveryOfferedActionIsReachableFromEnter tests the converse direction.
- Task 1 Step 5 is ticked: e7c6c6e8 landed it.

**Operating envelope, corrected.** The worst case is ~20s, not ~30s: park's
15s CompletionTimeout (couch.go:119) absorbs the child-death wait, then the
resume spawn's 5s resumeRegistrationTimeout (couch.go:107). The plan's earlier
"5s exact-child-death wait" and "10s blocked-start acknowledgement" named no
constant that exists.
```

And in the issue: mark the "operator ends on the same actor" Done-when bullet as moved to `pair#186` (it is the direct consequence of the pane that outlives its child), drop the now-false "⚠ The estimate is now stale" note from `## Revisions`, and update `workshop/projects/couch.md:201` to describe M2 as the gesture only, with a `pair#186` row for the surface.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan envelope untouched; verified 15s CompletionTimeout (couch.go:119) + 5s resumeRegistrationTimeout (couch.go:107) = ~20s, not ~30s.
  - id: BR-6
    disposition: not-addressed
    note: |
      relaunch_test.go still has five cases; agent-unsupported and profile-missing remain pinned only at CheckResumePreconditions.
  - id: BR-7
    disposition: not-addressed
    note: |
      plan Task 1 Step 5 still unticked; the stale-estimate warning still contradicts the re-derived Estimate. Subsumed by the rule in the new plan-record-lags-code finding.
  - id: BR-9
    disposition: not-addressed
    note: |
      manifest.go:524 unchanged, and threadreason.go:503 is a second offender; one sort assertion over the manifest lists closes the family.
  - id: BR-10
    disposition: not-addressed
    note: |
      Instance fixed and revert-verified (deleting the arm reds the chord test), but no enumeration from InterceptorHit to a handler exists, and README:141 plus menuControls still carry no Alt+n row.
  - id: BR-22
    disposition: addressed
    note: |
      refuseBinding is now the only binding-refusal constructor and is called from resume.go:211, resume.go:316 and artifactcollision_fake.go:126; the fake's wording is pinned by test.
  - id: BR-23
    disposition: addressed
    note: |
      pumpStdin now loads c.trace under c.mu (console.go:1195-1200); no test pins it, which is inherent for a latent race.
findings:
  - id: new
    severity: Important
    family: plan-record-lags-code
    title: |
      The pair#186 split was written into one artifact and propagated to none of its three consumers
    detail: |
      2nd in this family, so the deliverable is the RULE: a scope change is not
      recorded until every artifact restating the scope derives from or cites the
      one entry that made it. Enumeration, all live: the plan doc has no
      "## Revisions" at all and its M2 Core-concepts table still declares
      paneState and RenderHoldingPane in couchtty/holding.go, none of which exist
      anywhere in the tree, with Tasks 8/10/11/12 carrying unticked steps though
      Task 8 and Step 2b shipped; workshop/projects/couch.md:201 still describes
      M2 as "the gesture and a surface that outlives its child" and lists no
      pair#186 row; the issue's "## Done when" still claims "the operator ends on
      the same actor, not in the switcher" while console.go:1383-1387 sets
      FocusPanel unconditionally and its own comment says why; and BR-7's stale
      estimate warning still stands. Also: "## Plan" marks M2 [x] with no
      Review-Verdict trailer and no "closed M2" log line in the range.
  - id: new
    severity: Important
    family: declared-source-hand-maintained-consumers
    title: |
      Both guards written to catch this class read a hand-maintained list, so the new chord and the new operation walked past them
    detail: |
      7th in this family. Do NOT fix these two sites -- the rule is: a guard whose
      source is a hand-maintained list is a guard the next addition can skip; it
      must derive from the table the production path consumes. (1)
      TestREADMEDocumentsEveryPanelControl (couchcmd/readme_test.go:61) exists so
      "adding a key to the UI makes this test fail until its documentation has a
      home in README" -- it iterates menuControls (menu.go:19). Alt+n and
      Ctrl+Alt+n went into knownSequences (keys.go:170-171) and never into
      menuControls, so the guard could not fire and README.md:141 still tells the
      operator Alt+n reloads pair in place in "any pane", which is false inside
      couch. (2) TestEveryOfferedActionIsReachableFromEnter tests the CONVERSE of
      plan Task 10 Step 2b: it walks menuActionItems asserting each offered action
      is reachable, where the plan asked it to walk Operations() asserting each
      switcher-reachable PresentationTUI operation is OFFERED. Offered-implies-
      reachable cannot catch declared-but-unreachable, the exact failure it was
      written for, and the issue's Plan line claims otherwise. Cheap fix: a
      row-action field on Operation that menuActionItems derives membership from,
      which also collapses several of the seven restatements still standing.
  - id: new
    severity: Minor
    family: done-when-untested
    title: |
      onRelaunchHotkey's panel branch is production code with no test, and the Done-when bullet it serves is unpinned
    detail: |
      2nd in this family, so the rule rather than the site: the enumeration IS the
      Done-when list -- every bullet the issue still claims must cite the test
      that pins it or the Revisions line that moved it. console.go:1378-1380
      (panel focus takes the highlighted row) is never exercised; only the
      actor-focus branch is driven by console_relaunch_chord_test.go. Done-when
      "Alt+n on the switcher relaunches the highlighted row and leaves the
      operator in the switcher" and plan Task 10 Step 3's "two tests, because the
      endings differ" are both unmet.
  - id: new
    severity: Minor
    family: review-window-degenerate
    title: |
      The gate handed this round the whole 111-commit branch, including three other issues' already-closed work
    detail: |
      2nd in this family (BR-11 was the empty variant of the same rule). Base
      88fe1de is the branch point, so the window is 164 files and +24177/-4669
      spanning pair#170, pair#181 and pair#185, each of which already passed its
      own close gate with its own review doc in this very diff. The window should
      be the un-reviewed span: M1's close at b7ec5e64, or at most the first #182
      commit. I reviewed b18f958e..HEAD for the code, which is #182 plus the
      #183/#184/#185/#186 issue-sync commits.
```
