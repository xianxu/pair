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

---

## Re-review — 2026-09-04T16:56:08-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 182 — Relaunch an actor: restart Pair in place, keeping the agent conversation |
| repo | pair |
| issue file | workshop/issues/000182-relaunch-an-actor-restart-pair-in-place-keeping-the-agent-conversation.md |
| boundary | whole-issue close |
| milestone | — |
| window | 88fe1de011b4c6be58e5a8b20eed89dfa4000f5d..511ebd1a9a815ddcea2df9a4524121b4c5c6a4b2 |
| command | sdlc close --issue 182 |
| reviewer | claude |
| timestamp | 2026-09-04T16:56:08-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The relaunch operation itself is in good shape: the check ordering that is the whole design is implemented as written and pinned by tests that assert the *state left behind* rather than the error text, the chord reaches a handler and the operator ending is now both corrected in `## Done when` and tested, and this round's two structural fixes are real — I verified by mutation that dropping `RowAction` from an operation turns `TestRowActionDeclarationsAndTheMenuAgreeInBothDirections` red, and that the `knownSequences` loop in `TestEveryInterceptedChordHasAHandler` is the direction that would have caught the original intercepted-and-dropped `alt+n`. What keeps this from SHIP is that both fixes stopped one step short of the class they were written to close — `intercepts()`/`hit()` are still two hand-maintained switches over one enum and the new guard *skips* the kinds `intercepts()` forgets (mutation-verified green), and `menuActionItems`'s new production filter makes the sweep's second direction unfalsifiable while turning its failure into a silent UI drop (mutation-verified: an undeclared action vanishes from the switcher with zero test failures) — plus the `pair#186` split is still unpropagated into three artifacts, and the new intercepted chord reached neither `atlas/couch.md`'s chord section nor README's couch section. All four are cheap.

### 1. Strengths

- **`relaunch.go:71-140`** — the refusal-before-park ordering reads exactly as the Spec specifies it, and the two preconditions that *cannot* be checked early are named in the comment rather than hoped over. `relaunch_test.go:157-172` asserts the property that matters ("it refuses and nothing was destroyed" — lifecycle trace empty, no quits triggered, revision and incarnation count unchanged), not just the refusal.
- **`console_relaunch_chord_test.go:311`** — the second loop, over `knownSequences` filtered to intercepting kinds, is derived from the production table and is the direction that would have caught the original defect. That is the load-bearing half of BR-10's rule and it now exists.
- **`ops.go:103-127` + `menu.go:1077-1097`** — `Operation.RowAction` is load-bearing in production, not test-only. I confirmed the declared→offered direction has teeth by adding `RowAction: true` to `stop`: `menu_action_sweep_test.go:94` fails with the right message.
- **`console_relaunch_chord_test.go:72-76`** — the `FocusPanel` ending is asserted, and the `## Done when` bullet was rewritten to say what the code does *and* why. A corrected claim with a test behind it is the right shape.
- **`atlas/couch.md:774-781`** — the `COUCH_INPUT_TRACE` entry says plainly that the tap sits before the Interceptor and captures pasted secrets. That is ARCH-SECURE done properly rather than ceremonially.

### 2. Critical findings

None.

### 3. Important findings

**I-1 — `intercepts()` and `hit()` are still two hand-maintained switches, and the new guard skips whatever `intercepts()` forgets** (`keys.go:50-76`, `console_relaunch_chord_test.go:334`). **This is the 8th finding in family `declared-source-hand-maintained-consumers`** — the rule is already written (BR-10: "a value the interceptor can emit must reach a handler by construction"), so the deliverable is not this site but finishing the enumeration. `hit()` already *is* the mapping; `intercepts()` duplicates its domain, and the guard's `if !sequence.kind.intercepts() { continue }` gates on the very list that can be wrong. Verified in a scratch worktree: removing `seqRelaunch` from `intercepts()` leaves `TestEveryInterceptedChordHasAHandler` green. `keys.go:270-276`'s own comment records that this exact omission has already shipped once. Fix: `func (k seqKind) intercepts() bool { return k.hit() != HitNone }` — one list, and the test's skip clause becomes derived rather than independent. ARCH-DRY, ARCH-PURPOSE.

**I-2 — `declaredRowActions` makes the sweep's second direction unfalsifiable and converts its failure into a silent drop** (`menu.go:1085-1097`, `menu_action_sweep_test.go:96-100`). The test builds `offered` from `menuActionItems`, which now filters through `declaredRowActions`, so `offered ⊆ declared` holds by construction and the loop can never fail. Verified: adding `"bogus"` to the live row's list in `menuActionItemsForState` produces **no** test failure — the item is silently removed from the switcher instead. That is the same class the guard exists for, now invisible. Fix: build the test's `offered` set from `menuActionItemsForState` (pre-filter) so the direction has teeth; keep the production filter or drop it, but it must not be the only enforcement.

**I-3 — the `pair#186` split still has unswept consumers, in the same files the fix edited.** **This is the 3rd finding in family `plan-record-lags-code`.** The rule was stated correctly in `lessons.md`; the enumeration was written and then partially applied. Still live: (a) `## Done when` bullet 2 claims "verified on the real stack by rebuilding Pair between the two observations" — that is Task 11, which `## Revisions` says moved to `pair#186`, and the smoke test proved conversation survival, not a rebuilt binary; (b) `## Done when` "A relaunch does not change what `ctrl+backspace` returns to, proved by test" — moved with the holding surface, no such test exists, bullet unmarked while the spinner bullet *was* marked `(moved to pair#186)`; (c) the plan's M2 Integration-points table still lists `onExit` and `finishOperation` as modified "for the held pane" and describes `onRelaunchHotkey` as returning the operator to the actor, which the code and the corrected Done-when both contradict, and the new `## Revisions` does not name either; (d) `workshop/projects/couch.md:201` still labels the row `[pair#182 M2]` (a dangling reference link, no detail block) though the issue's Revisions removes M2's `Mx` tag. Note (a) is also ARCH-PURPOSE: the rebuilt-binary check is the issue's headline claim, so deferring it needs to be visible in `## Done when`, not only in Revisions.

**I-4 — the new intercepted chord reached no couch-facing doc.** **This is the 2nd finding in family `new-surface-undocumented`.** `atlas/couch.md:436` reads "**Alt+d is Couch's own detach** (`pair#170`), intercepted like Alt+x and for the same reason…" — the exact slot for an Alt+n counterpart, which is absent; `atlas/couch.md:355` still frames the interception set as "two lifecycle chords". README's couch section (`README.md:382-389`) likewise presents Alt+x/Alt+d as the complete grid, and neither doc records the switcher-row form or the deliberate `FocusPanel` ending. `TestREADMEDocumentsEveryPanelControl` could not catch this: it substring-matches `control.Keys` against the whole README, and "Alt+n" was already present in Pair's own keybinding table. Fix: an atlas paragraph beside Alt+d, a couch-section README sentence, and — since this is the same shape as I-2 — consider scoping the README guard to the couch section.

### 4. Minor findings

- `menuActionItems` now rebuilds the entire `Operations()` table (≈15 structs with `[]ArgSpec` literals) on every action-menu render (`menu_render.go:200`) for an O(n·m) `slices.Contains`. Trivial in absolute terms, but it is a keystroke-path render; a package-level memoized set would cost nothing.
- `processInput` (`console.go:636-638`) still consumes the chord bytes when a hit has no handler, so a gap is silent at runtime even though the test now catches it. A status notice on the nil branch would make it visible in a build where the test was skipped.

### 5. Test coverage notes

- Mutation-verified this round: declared→offered has teeth; offered→declared does not (I-2); `intercepts()` omission is not caught (I-1).
- The whole-issue suite is green except `ptychild`-backed tests (`TestNotificationPTYConformance`, `TestBlockedRunnerCancellationConformance/pty/*`, `couchcmd/run_test.go` interactive-launch cases), which fail with `operation not permitted` — a known environment restriction on pty allocation in this session, not a code defect. `go build ./...` and `go vet` on both couch packages are clean.
- Still uncovered and re-raised as prior findings: the agent-unsupported and profile-missing refusals at the `Relaunch` level (BR-6), and `onRelaunchHotkey`'s panel branch (BR-26).

### 6. Architectural notes

- **ARCH-DRY — flag (I-1).** Two switches over one enum; the derivation is one line.
- **ARCH-PURE — pass.** `CheckResumePreconditions`, `RowActions`, `declaredRowActions` and the outcome types are pure; IO stays behind `Threads`/`PairLifecycle`/`Path`/`Artifacts`. `hitHandlers()` is a table, not logic.
- **ARCH-PURPOSE — flag (I-1, I-3).** Both this round's fixes resolved the site and left an enumerable sibling; the rebuilt-binary proof — the reason the issue exists — moved out without the Done-when following it.
- **ARCH-MOCK — pass.** Fakes sit behind the same seams production uses, and `runner_contract_test.go` is a real fake-vs-pty conformance check.
- **ARCH-CONSTRAINTS — flag.** BR-1 stands: the plan's ~30s envelope names two budgets that do not exist as constants (measured: `CompletionTimeout` 15s at `couch.go:119`, `resumeRegistrationTimeout` 5s at `couch.go:107`; child death is awaited *inside* the 15s at `park.go:552`). Plus the Minor render allocation above.
- **ARCH-SECURE — pass.** Chord parsing is exact-match with a bounded held buffer; the trace is opt-in, 0600, fails loudly on an unopenable path, and its capture surface is documented.

### 7. Plan revision recommendations

The plan's new `## Revisions` is the right shape but incomplete. Append to it:

- **The M2 Integration-points table's `onExit` and `finishOperation` rows are `pair#186`'s.** The holding-pane modifications they describe do not exist here; `finishOperation`'s actual change was child adoption via `StartedChild`, not installing a child into a held pane.
- **`onRelaunchHotkey` does not return the operator to the actor.** The bullet under Integration points still says "from an actor it relaunches that actor and returns to it"; the shipped code sets `FocusPanel` deliberately and the issue's `## Done when` has been corrected to match. Correct it here too, or the plan re-teaches the wrong behaviour.
- **Tick or annotate Task 1 Step 5 (line 159, landed in `e7c6c6e8`) and Task 7 Step 3 (line 352, the M1 milestone-close at `b7ec5e64`).** The existing "unticked boxes below are stale" note covers only Task 8 and Task 10 Step 2b.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      plan lines 32-35 unchanged; measured CompletionTimeout=15s (couch.go:119) and resumeRegistrationTimeout=5s (couch.go:107), child death awaited inside the 15s (park.go:552).
  - id: BR-6
    disposition: not-addressed
    note: |
      relaunch_test.go:76 still has five cases; no agent-unsupported or profile-missing case at the Relaunch level.
  - id: BR-7
    disposition: not-addressed
    note: |
      the stale-estimate half is resolved in the issue's Revisions; plan lines 159 and 352 are still unticked for work that landed (e7c6c6e8, b7ec5e64) and the new Revisions note does not name them.
  - id: BR-9
    disposition: not-addressed
    note: |
      artifactpath/manifest.go:524 still places relaunch.go between pathops.go and procops.go.
  - id: BR-10
    disposition: addressed
    note: |
      hitHandlers table plus a knownSequences-driven test; the load-bearing direction is derived from the production table. Residual raised as new (intercepts() is still hand-maintained).
  - id: BR-24
    disposition: addressed
    note: |
      plan gained a Revisions section naming the moved tasks and the entities declared-but-absent; project row and the FocusPanel Done-when corrected; M2's Mx tag dropped. Remaining unswept consumers raised as new.
  - id: BR-25
    disposition: addressed
    note: |
      menuControls and README both fixed; RowAction gives membership one source and the declared-to-offered direction is mutation-verified red. The converse direction's unfalsifiability raised as new.
  - id: BR-26
    disposition: not-addressed
    note: |
      console.go:1372-1373 (panel branch takes the highlighted row) still has no test; console_relaunch_chord_test.go drives only the actor branch.
  - id: BR-27
    disposition: withdrawn
    note: |
      sdlc documents branch-point..HEAD as the whole-issue integration window by design (ariadne close.go:975-987), so the base is not a gate defect; the branch carrying four issues is a hygiene fact, not a window bug.
findings:
  - id: new
    severity: Important
    family: declared-source-hand-maintained-consumers
    title: |
      intercepts() and hit() are still two hand-maintained switches over one enum, and the new guard skips whatever intercepts() forgets
    detail: |
      8th in this family, so the deliverable is the enumeration, not the site. keys.go:50-76 keeps two switches over seqKind where hit() alone is the
      mapping, and console_relaunch_chord_test.go:334 gates its sweep on intercepts() -- the list that can be wrong. Verified in a scratch worktree:
      removing seqRelaunch from intercepts() leaves TestEveryInterceptedChordHasAHandler green. keys.go:270-276 records that this exact omission has
      already shipped once. Fix: intercepts() returns k.hit() != HitNone, so one list remains and the test's skip clause derives from it. ARCH-DRY, ARCH-PURPOSE.
  - id: new
    severity: Important
    family: guard-unfalsifiable-by-construction
    title: |
      menuActionItems filters through declaredRowActions, so the sweep's offered-implies-declared direction can never fail and its failure is now a silent UI drop
    detail: |
      menu.go:1085-1097 filters the row's offer through couchcore.RowActions() before menu_action_sweep_test.go:96-100 reads it, so offered is a subset of
      declared by construction. Verified: adding "bogus" to the live row in menuActionItemsForState produces zero test failures and the item simply
      disappears from the switcher. The rule: a guard must be able to fail, and production must not coerce its input into agreement. Fix: build the test's
      offered set from menuActionItemsForState, pre-filter.
  - id: new
    severity: Important
    family: plan-record-lags-code
    title: |
      the pair#186 split is still unpropagated into four artifacts, including the two the fix edited
    detail: |
      3rd in this family; the rule is already in lessons.md, so the deliverable is the sweep. Live: the issue's Done-when bullet 2 still claims the
      rebuilt-binary verification that Revisions moved to pair#186 (the smoke test proved conversation survival, not a rebuilt binary), and the
      ctrl+backspace bullet is likewise unmarked with no test; the plan's M2 Integration-points table still lists onExit/finishOperation as holding-pane
      modifications and describes onRelaunchHotkey as returning to the actor, contradicting the code and the corrected Done-when; projects/couch.md:201
      still labels the row [pair#182 M2] with no detail block though the Mx tag was dropped. ARCH-PURPOSE on bullet 2 -- the rebuilt binary is the point of the issue.
  - id: new
    severity: Important
    family: new-surface-undocumented
    title: |
      the Alt+n interception reached neither atlas/couch.md's chord section nor README's couch section, and the README guard could not notice
    detail: |
      2nd in this family. atlas/couch.md:436 has "Alt+d is Couch's own detach (pair#170), intercepted like Alt+x" with no Alt+n counterpart, and line 355
      still frames the set as "two lifecycle chords"; README.md:382-389 does the same, documenting Alt+n only inside Pair's own keybinding table, and
      neither records the switcher-row form or the deliberate FocusPanel ending. TestREADMEDocumentsEveryPanelControl substring-matches control.Keys
      against the whole README, so the pre-existing Pair-table "Alt+n" satisfied it. Fix: an atlas paragraph beside Alt+d, a couch-section README sentence,
      and scope the README guard to the couch section.
  - id: new
    severity: Minor
    family: operating-envelope
    title: |
      menuActionItems rebuilds the whole Operations() table on every action-menu render
    detail: |
      menu.go:1096 calls couchcore.RowActions(), which constructs ~15 Operation structs with ArgSpec slices, once per render at menu_render.go:200 -- a
      keystroke-path render. Negligible in absolute terms; a package-level memoized set costs nothing.
  - id: new
    severity: Minor
    family: declared-source-hand-maintained-consumers
    title: |
      processInput still consumes the chord bytes when a hit has no handler
    detail: |
      console.go:636-638 drops the hit silently on the nil branch, so a gap is invisible at runtime even though the test now catches it at build time. A
      status notice there would make the missing case observable to an operator.
```

---

## Re-review — 2026-09-04T17:39:21-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 182 — Relaunch an actor: restart Pair in place, keeping the agent conversation |
| repo | pair |
| issue file | workshop/issues/000182-relaunch-an-actor-restart-pair-in-place-keeping-the-agent-conversation.md |
| boundary | whole-issue close |
| milestone | — |
| window | 88fe1de011b4c6be58e5a8b20eed89dfa4000f5d..f1201093d867b891db3c26c936b09eb1d6367f10 |
| command | sdlc close --issue 182 |
| reviewer | claude |
| timestamp | 2026-09-04T17:39:21-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 10's five findings are all genuinely closed, and I verified the two load-bearing ones by mutation rather than by commit message: adding `"bogus"` to the live row's offer now reds `TestRowActionDeclarationsAndTheMenuAgreeInBothDirections` (BR-29's guard can fail again), and deleting `case seqRelaunch` from `hit()` reds five chord tests through `intercepts()` (BR-28's two switches are now one). The relaunch operation itself is in good shape: the check order is the design, the four outcomes are distinct, adoption is a property of the result, and the docs gate is genuinely satisfied — `atlas/couch.md:438-455` and `README.md:387-397` both carry the chord, the scope deviation and the deliberate `FocusPanel` ending. What keeps this off SHIP is one live behaviour defect the still-open BR-26 was pointing at: `onRelaunchHotkey`'s panel branch reads the selection off `CurrentFrame()`, which is zero on every frame except the root, so `Alt+n` from the switcher's action list or from a confirmation refuses with "relaunch: no thread selected" while a row is plainly highlighted — the Done-when bullet that has never had a test. Plus one bookkeeping gap: this plan's Core-concepts table is enforced by no contract test, and two of its rows name symbols that exist nowhere.

## 1. Strengths

- **`relaunch.go:99-124`** — the refusal order reads exactly as the Spec promises, and the two failure states carry their *own* recovery text (`:128-134` names park's retry/recover/abandon; `:139-142` names Enter). `refuseResume(ResumeNotRunning, …)` at `:100` is the fix for a code that used to contradict the state it described, and it is still correct.
- **`console.go:1470-1473` + `keys.go:70-72`** — the chord path now has exactly one mapping. `hitHandlers()` is data a test walks, `intercepts()` derives from `hit()`, and `TestEveryInterceptedChordHasAHandler` walks `knownSequences` (a production table) rather than a list a new chord could skip. Revert-verified.
- **`menu.go:1075-1080` + `menu_action_sweep_test.go:87-107`** — deleting `declaredRowActions` instead of keeping a filter that made the guard unfalsifiable is the right call, and the comment records *why* the tempting fix was wrong. Mutation-verified red.
- **`couchcmd/readme_test.go:58-77`** — scoping the README guard to the couch section and making a missing marker `t.Fatalf` rather than a skip is the correct shape for a guard that silently stopped checking.
- **`menu.go:129-144` `setBookkeepingNotice`** — a refresh's frame-validity message no longer erases an operation's own result. That is the one outcome (`park-ok-resume-failed`) whose entire value is the sentence it would have overwritten.
- **`atlas/couch.md:785-800`** — the `COUCH_INPUT_TRACE` entry puts the disclosure (the tap is upstream of the Interceptor; the file holds prompts and pasted secrets) *before* the how-to. ARCH-SECURE done substantively.

## 2. Critical findings

None.

## 3. Important findings

**`cmd/internal/couchtty/console.go:1379` — the panel branch reads the selection off the wrong frame, so `Alt+n` in the switcher refuses whenever the operator has drilled in.**
`target = c.menu.CurrentFrame().SelectedAddress`. `SelectedAddress` is populated only on the root frame — `reduceRootKey`'s Tab handler builds the actions frame with `Thread:` set and `SelectedAddress` zero (`menu.go:457-459`), and confirmation/text frames likewise. So from any non-root frame the target is the zero address and the operator gets `relaunch: no thread selected` on the status row while the row is highlighted and its action list is open on screen. Confirmed in a scratch worktree: from the root frame `Alt+n` opens the relaunch confirmation; after one `Tab` the same call produces no frame change at all. Fix sketch: take the root selection — `c.menu.Frames[0].SelectedAddress` with a length guard — which is also what `reduceParkHotkey` collapses to (`menu.go:547-548`).

**This is the 3rd finding in family `done-when-untested`.** So the deliverable is the rule, not this line: *every `## Done when` bullet the issue still claims must cite the test that pins it, or the `## Revisions` line that moved it.* Measured prevalence on this issue: eight bullets still claimed here; seven cite a test (`relaunch_test.go`, `console_relaunch_chord_test.go`, `menu_action_sweep_test.go`), and the one that does not — "Alt+n on the switcher relaunches the highlighted row" — is the one that turns out to be broken. Write the enumeration (a bullet-to-test map, checked once), and the panel test it forces will red on the line above.

**`workshop/plans/000182-relaunch-an-actor-plan.md:126-133, 411-431` — this plan's Core-concepts table is enforced by nothing, and two of its rows name symbols that exist nowhere in the tree.**
`TestCoreConceptsContract` exists precisely to turn "a repeatedly drifting architecture table" into an executable contract, but it reads `conceptPlans` (`core_concepts_contract_test.go:206-214`), a literal list of two plan filenames; `couchcore/plan_contract_test.go` pins `000149-…` and `000151-…` the same way. `000182-…-plan.md` was never added to either, so its table is prose. Live consequence: the M2 table still declares `paneState` (`console.go`, "new") and `RenderHoldingPane` (`cmd/internal/couchtty/holding.go`, "new") — `grep -rn 'paneState\|RenderHoldingPane' cmd/` returns nothing and `holding.go` does not exist. The Revisions entry disclaims them in prose, which is why this is Important rather than Critical, but a row marked `new` that the guard cannot see is the drift the guard was built for.

**This is the 10th finding in family `declared-source-hand-maintained-consumers`.** The rule is by now well established and is exactly this: *a guard whose input is a hand-maintained list is a guard the next addition can skip.* The enumeration here is "every plan under `workshop/plans/` that contains a Core concepts table" — discovered by scanning the directory rather than by naming files, with the existing `planned`-status escape hatch (`core_concepts_contract_test.go:271-277`) carrying rows whose work has moved out. That single change makes both phantom rows fail loudly instead of needing a reader to notice the Revisions paragraph.

## 4. Minor findings

Carried, unchanged this round — see the dispositions block:

- BR-1 — the plan's operating envelope still says ~30s over budgets that do not exist as named constants (measured: 15s `CompletionTimeout` + 5s `resumeRegistrationTimeout` ≈ 20s).
- BR-6 — `relaunch_test.go:76` still has five refusal cases; the agent-unsupported / profile-missing precedence is pinned only at `CheckResumePreconditions`, never through `Relaunch`.
- BR-7 — plan lines 159 and 352 unticked for work that landed (`e7c6c6e8`, `b7ec5e64`); `sdlc close`'s plan-unchecked guard will refuse on them.
- BR-9 — `artifactpath/manifest.go:524` still places `relaunch.go` between `pathops.go` and `procops.go`; `threadreason.go:503` is the second offender.
- BR-26 — the panel branch still has no test, and it now has a defect (above).
- `AllInterceptorHits()` (`keys.go:97-100`) is a third hand-maintained enumeration, but it is not load-bearing — the sweep's other direction walks `knownSequences`, so a stale entry can only mask an unreachable hit. Noted, not raised.

## 5. Test coverage notes

Suite state: `./cmd/internal/couchcore`, `./cmd/internal/couchtty` and `./cmd/internal/couchcmd` pass for every relaunch-, chord- and README-related selector. The full `go test ./cmd/...` shows failures confined to pty/`fork-exec` conformance tests (`TestPtyRunner*`, `TestChild*`, `TestNotificationPTYConformance`, `TestOSHostConformsToTheFake`, `/bin/ps` enumeration) — the known environment limitation in this harness, not diff-related.

Two mutation checks passed: BR-29's guard (adding an undeclared action to the live row reds both sweep directions) and BR-28's derivation (removing `seqRelaunch` from `hit()` reds five chord tests). The gap is `console.go:1372-1393`: the actor branch is driven end-to-end from raw bytes, the panel branch is driven by nothing, and that is where the defect is.

## 6. Architectural notes

- **ARCH-DRY — flag.** One flag, and it is the plan-contract list above; the code-side duplication this issue was about (`intercepts()`/`hit()`, `declaredRowActions`, the two exit lists, the three confirmation lists) is genuinely collapsed now.
- **ARCH-PURE — pass.** `CheckResumePreconditions` is a total pure predicate over `(record, binding, pathExists)`; `precondition_test.go` runs it with no exec, net or fs. The IO — `workingPathExists`, `resumeEvidence`, `PairLifecycle.Park`, `ResumeContext` — sits on `*Couch` behind injected seams, and `Relaunch` is thin composition over them.
- **ARCH-PURPOSE — flag.** The issue's headline claim (an old binary yields the *current* one) is explicitly not measured here; the split to `pair#186` is recorded in both `## Revisions` and the Done-when bullet, so this is a declared deferral rather than a silent under-delivery. The residual is NEW-1: a bullet still claimed here that the code does not deliver on one of its two paths.
- **ARCH-MOCK — pass.** Both halves run through the same seams production uses — `PairLifecycle` against `pairlifecycletest.Fake` with `TriggerQuitHook` modelling Pair's real quit/cleanup/child-death sequence, `Artifacts` against the collision-checker fake, `Path` against `FakePathOps`. `conformance_live_test.go` is the live counterpart. No relaunch test reaches a real binary.
- **ARCH-CONSTRAINTS — flag (Minor, BR-1).** The mechanism is right: relaunch runs on the bounded capacity-one operation queue off the terminal event loop, with a progress notice and spinner for the gap (`menu.go:1538-1539`). Only the plan's stated numbers are wrong, and they over-budget, so nothing downstream changes. `menuActionItems` no longer rebuilds `Operations()` per render.
- **ARCH-SECURE — pass.** `COUCH_INPUT_TRACE` is the only new surface: opt-in, `0600`, path injected from the composition root (`couchcmd/run.go:373`) rather than read in the constructor, open failure surfaced at control priority instead of degrading to a silent empty file, closed in `teardown`, read under `c.mu` in `pumpStdin`. Relaunch parses no new persisted format and mints no credential.

## 7. Plan revision recommendations

- **`workshop/plans/000182-relaunch-an-actor-plan.md` — one `## Revisions` entry covering the enforcement gap**, not another prose disclaimer: either mark the `paneState` / `RenderHoldingPane` rows with a `planned` status so the contract's existing skip covers them, or move both rows out of this plan into `pair#186`'s, and add this plan to whatever list `TestCoreConceptsContract` reads (ideally replacing that list with directory discovery, per NEW-2).
- **Same file, Tasks 1/7** — tick line 159 (`Step 5: Commit`, landed in `e7c6c6e8`) and line 352 (`milestone-close --issue 182 --milestone M1`, landed in `b7ec5e64`). The existing Revisions note declares Task 8's boxes "stale" but says nothing about these two, and `sdlc close`'s plan-unchecked guard reads boxes, not prose.
- **Same file, "Operating envelope"** — restate as ~20s with the two constants cited by file:line (`couch.go:119` `CompletionTimeout: 15s`; `couch.go:107` `resumeRegistrationTimeout: 5s`, with child death awaited *inside* the 15s at `park.go:552`), and say explicitly that the figure is derived rather than declared.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      plan lines 32-37 untouched; measured 15s CompletionTimeout (couch.go:119) + 5s resumeRegistrationTimeout (couch.go:107), child death inside the 15s (park.go:552).
  - id: BR-6
    disposition: not-addressed
    note: |
      relaunch_test.go:76 still has five refusal cases; no agent-unsupported or profile-missing case reaches Relaunch.
  - id: BR-7
    disposition: not-addressed
    note: |
      plan lines 159 and 352 still unticked for landed work (e7c6c6e8, b7ec5e64); the new Revisions note names only Task 8's boxes.
  - id: BR-9
    disposition: not-addressed
    note: |
      manifest.go:524 still places relaunch.go between pathops.go and procops.go; threadreason.go:503 is still the second offender.
  - id: BR-26
    disposition: not-addressed
    note: |
      console.go:1379 still has no test, and the untested branch turns out to be wrong — raised as a new Important finding.
  - id: BR-28
    disposition: addressed
    note: |
      intercepts() is now k.hit() != HitNone (keys.go:72); revert-verified — deleting the seqRelaunch arm of hit() reds five chord tests.
  - id: BR-29
    disposition: addressed
    note: |
      declaredRowActions and RowActions() deleted; the sweep reads Operation.RowAction directly. Mutation-verified — adding "bogus" to the live row reds both directions.
  - id: BR-30
    disposition: addressed
    note: |
      Done-when bullets 2/ctrl+backspace/child-exited, the plan's integration-points table and projects/couch.md:201 all now carry the pair#186 split.
  - id: BR-31
    disposition: addressed
    note: |
      atlas/couch.md:438-455 and README.md:387-397 both document the chord, the scope deviation and the FocusPanel ending; readme_test.go:58-77 scopes the guard and fatals on a missing marker.
  - id: BR-32
    disposition: addressed
    note: |
      the per-render Operations() rebuild is gone with declaredRowActions; menu.go no longer calls couchcore.RowActions() at all.
  - id: BR-33
    disposition: addressed
    note: |
      console.go:639-644 sets a status notice on the nil-handler branch, so a gap is observable at runtime and not only at build time.
findings:
  - id: new
    severity: Important
    family: done-when-untested
    title: |
      onRelaunchHotkey reads the selection off CurrentFrame(), so Alt+n from the switcher refuses whenever the operator has drilled into a frame
    detail: |
      console.go:1379 takes c.menu.CurrentFrame().SelectedAddress, but SelectedAddress is
      populated only on the ROOT frame — the actions frame is built with Thread set and
      SelectedAddress zero (menu.go:457-459), as are confirmation and text frames. So from
      any non-root frame the target is the zero address and the operator gets
      "relaunch: no thread selected" on the status row while the row is highlighted and its
      action list is open. Verified in a scratch worktree: from the root frame the panel
      branch opens the relaunch confirmation; after one Tab the same call changes nothing.
      Reachable on an ordinary path — Alt+n pressed a second time on the confirmation it
      just opened hits it too. 3rd in this family, so the deliverable is the rule, not the
      line: every Done-when bullet still claimed must cite the test that pins it or the
      Revisions line that moved it. Eight bullets are still claimed here; seven cite a test,
      and the one that does not is the one that is broken. Fix: read the root selection
      (c.menu.Frames[0].SelectedAddress, length-guarded), which is what reduceParkHotkey
      already collapses to. ARCH-PURPOSE.
  - id: new
    severity: Important
    family: declared-source-hand-maintained-consumers
    title: |
      this plan's Core concepts table is enforced by no contract test, and two of its rows name symbols that exist nowhere
    detail: |
      TestCoreConceptsContract exists to turn a drifting architecture table into an
      executable contract, but it reads conceptPlans (core_concepts_contract_test.go:206-214),
      a literal list of two filenames; couchcore/plan_contract_test.go pins 000149 and 000151
      the same way. 000182-relaunch-an-actor-plan.md is in neither, so its table is prose.
      Live: the M2 table declares paneState (console.go, "new") and RenderHoldingPane
      (cmd/internal/couchtty/holding.go, "new"); grep -rn 'paneState|RenderHoldingPane' cmd/
      returns nothing and holding.go does not exist. The Revisions entry disclaims them in
      prose, which is why this is Important and not Critical. 10th in this family, so the
      rule: a guard whose input is a hand-maintained list is a guard the next addition can
      skip. The enumeration is "every plan under workshop/plans/ carrying a Core concepts
      table", discovered by scanning the directory, with the existing planned-status skip
      carrying rows whose work has moved to pair#186. ARCH-DRY.
```

---

## Re-review — 2026-09-04T18:08:27-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 182 — Relaunch an actor: restart Pair in place, keeping the agent conversation |
| repo | pair |
| issue file | workshop/issues/000182-relaunch-an-actor-restart-pair-in-place-keeping-the-agent-conversation.md |
| boundary | whole-issue close |
| milestone | — |
| window | 88fe1de011b4c6be58e5a8b20eed89dfa4000f5d..58d3470de758574c3a6c99f24a022f3ef18d0989 |
| command | sdlc close --issue 182 |
| reviewer | claude |
| timestamp | 2026-09-04T18:08:27-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both Important findings from round 11 are genuinely closed, and I verified each by reverting rather than by reading the commit message: restoring `target = c.menu.CurrentFrame().SelectedAddress` reds `TestSwitcherRelaunchFindsTheThreadFromAnyDepth/drilled_into_the_row's_actions`, and pointing the plan's `endsItsOwnChild` row back at `console.go` reds `TestCoreConceptsContract/PURE/\`endsItsOwnChild\`` with two IO-seam violations. BR-34's fix is the right shape — `MenuState.SelectedThreadAddress` answers "which thread is the operator pointing at" once, from any depth, rather than patching the one call site — and BR-35's fix pins this plan's six live rows in an executable contract while deferring the discovery-based class to pair#188 with a *measured* cost (14 rows, five real assertion failures) recorded in the test's own comment, not just in the issue. All eight still-claimed `## Done when` bullets now have a named test behind them, which is the substance the `done-when-untested` family was asking for. Nothing Critical or Important remains: what keeps this off a bare SHIP is four Minors carried unaddressed for six-plus rounds plus two new ones introduced by this round's two commits, all cheap and all non-blocking.

## 1. Strengths

- **`cmd/internal/couchtty/menu.go:1577-1595`** — the BR-34 fix answers the *question* rather than the site, and the doc comment records why a second copy in the next chord handler would be a second chance to pick the wrong frame. It correctly routes the `leave` confirmation (which binds no thread) back to the root selection via the existing `menuFrameBindsThread` predicate rather than adding a second exception.
- **`cmd/internal/couchtty/core_concepts_contract_test.go:61-68, 234`** — pinning the plan's six live rows and marking the two moved rows `planned — pair#186` makes the existing planned-skip carry the split, so the rows come back on automatically when pair#186 flips them. Revert-verified falsifiable.
- **`cmd/internal/couchtty/menu.go:1420-1434`** — moving `endsItsOwnChild` out of `console.go` because a PURE row must not live beside `os`/`io` imports is a real fix, not bookkeeping: the contract test now enforces the classification the plan claims.
- **`cmd/internal/couchcore/relaunch.go:88-124`** — the refusal order still reads exactly as the Spec promises, with `hasOccupiedIncarnation` before `soleParkableIncarnation` so a parked row (the ordinary switcher `Alt+n` target) gets "is not running… Enter resumes it" instead of a code naming a state it is not in.
- **`cmd/internal/couchtty/console_relaunch_chord_test.go:72-77`** — the actor branch's deliberate `FocusPanel` ending *is* pinned, so the Done-when bullet that was rewritten to match the code is not merely prose.
- **`cmd/internal/couchtty/inputtrace.go:69-75`** — `0600`, path-as-parameter, and an error (not `nil`) on an unopenable path, so "could not observe" can never present as "nothing arrived". ARCH-SECURE done substantively.

## 2. Critical findings

None.

## 3. Important findings

None.

## 4. Minor findings

**New this round:**

- `cmd/internal/couchtty/core_concepts_contract_test.go:196-225` — the `conceptPlans` doc comment is now two stacked accounts of the *same* deferral, already diverged: paragraph one says discovery "was tried and backed out… ~10 rows across #151/#160… tracked as its own issue"; paragraph two says it "brings 14 unpinned rows across pair#121, pair#181 and pair#182… so it is pair#188". A reader cannot tell which measurement is current. **2nd in family `parallel-derivation-drift`** — so the rule, not the line: *when a comment's claim is superseded, replace it; appending leaves two live accounts of one fact and no way to date them.*
- `cmd/internal/couchtty/menu.go:1423` — the moved comment still says the second hand-written list is "the switch below". `consumeExpectedParkExitLocked` is now in `console.go`; what is below is `operationNeedsProjectionRefresh`. **4th in family `plan-record-lags-code`** — the rule: *a relocation must carry its comment's positional references with it, or resolve them to names instead of positions.*

**Carried, still open (all verified unchanged this round):**

- BR-1 — plan lines 32-37 still claim "~30s worst case" over a "5s exact-child-death wait (`couch.go:119`)" and a "10s blocked-start acknowledgement" that do not exist. Measured: `CompletionTimeout` 15s (`couch.go:119`), `resumeRegistrationTimeout` 5s (`couch.go:107`), child death awaited *inside* the same 15s context (`park.go:549-555`). Real worst case ≈20s.
- BR-6 — `relaunch_test.go:76` still has five refusal cases; agent-unsupported and profile-missing are pinned only at `CheckResumePreconditions`, never through `Relaunch`.
- BR-7 — plan lines 159 and 352 unticked for work that landed (`e7c6c6e8`, `b7ec5e64`). The issue's stale-estimate half *is* now resolved by the Revisions rewrite.
- BR-9 — `artifactpath/manifest.go:524` still places `relaunch.go` between `pathops.go` and `procops.go`; `threadreason.go:503` remains the second offender.

## 5. Test coverage notes

`go build ./...` and `go vet ./cmd/internal/couchtty ./cmd/internal/couchcore` are clean. `go test ./cmd/internal/couchcore -run 'Relaunch|Resume|Park|Precondition'` is green. The only failures in `./cmd/internal/{couchtty,couchcore,couchcmd}` are `ptychild: … operation not permitted` — the known environment restriction on spawning a pty child from this shell (`TestNotificationPTYConformance`, `TestPtyRunner*`, `TestSpawnPostAckFailures…/pty`, `TestInteractiveLaunch*`), plus one deadlock that cascades from those EPERMs inside the same test binary. None is attributable to this diff.

The relaunch surface is well pinned: four outcomes each with a test, both chord aliases driven as raw bytes through `Run`'s own input loop rather than by calling handlers, `Alt+Shift+N` asserted to reach the child untouched, the confirmation's item-id invariant, adoption of the started child, and both switcher depths. The gap BR-6 names is the only meaningful one left, and it is two table rows.

## 6. Architectural notes

- **ARCH-DRY** — pass, with the two comment Minors above. `intercepts()` derives from `hit()`, `menuActionItems` reads `Operation.RowAction`, `endsItsOwnChild` has its two call sites, and `SelectedThreadAddress` is now the single answer to a question that had started to fork. `AllInterceptorHits()` remains hand-maintained but is not load-bearing (the sweep's other direction walks `knownSequences`).
- **ARCH-PURE** — pass, and improved this round: the `endsItsOwnChild` move makes a PURE-declared entity actually live in an IO-free file, enforced. `Couch.Relaunch` stays a thin ordered shell over the pure `CheckResumePreconditions`.
- **ARCH-PURPOSE** — pass. The purpose (Alt+n replaces the process, conversation survives) is delivered and reachable; the holding pane and rebuilt-binary verification left for pair#186 with Done-when bullets marked and a Revisions entry, not silently dropped. BR-34's class ask — a written bullet→test map — was not produced as an artifact, but I checked the enumeration by hand and all eight claimed bullets are pinned, so the rule holds in substance. Recording it as a map would make the next Done-when edit self-checking; worth doing in pair#186 rather than here.
- **ARCH-MOCK** — pass. Relaunch runs entirely through injected `PairLifecycle`, `Artifacts`, `FakePathOps` and `FakeRunner`; production and test flows share the same seam, and `conformance_live_test.go` carries the live check.
- **ARCH-CONSTRAINTS** — flagged at Minor (BR-1). The implementation's envelope is bounded and correct; it is the plan's *description* of it that is wrong, and over-stated rather than under-stated, so no downstream decision changes.
- **ARCH-SECURE** — pass. `COUCH_INPUT_TRACE` is opt-in, `0600`, fails loudly on an unopenable path, and `atlas/couch.md:785-800` leads with the disclosure that the tap is upstream of the Interceptor and the file holds prompts and pasted secrets.

## 7. Plan revision recommendations

- Tick lines 159 (`Task 1 Step 5: Commit`) and 352 (`Task 7 Step 3: sdlc milestone-close M1`) — both landed (`e7c6c6e8`, `b7ec5e64`) — or extend the existing "their unticked boxes below are stale" Revisions note to name them; it currently covers only Task 8 and Task 10 Step 2b.
- Correct the **Operating envelope (ARCH-CONSTRAINTS)** paragraph (lines 32-37) to the measured budgets: `CompletionTimeout` 15s (`couch.go:119`) with child death awaited inside it (`park.go:549-555`), plus `resumeRegistrationTimeout` 5s (`couch.go:107`, consumed at `launch_existing.go:110-111`) — ≈20s worst case. Cite the constant for each duration so the number cannot drift from its source again.

```findings
dispose:
  - id: BR-34
    disposition: addressed
    note: |
      SelectedThreadAddress (menu.go:1577-1595) fixes the question not the site; revert-verified — restoring CurrentFrame().SelectedAddress reds the drilled-in subtest.
  - id: BR-35
    disposition: addressed
    note: |
      Plan joined conceptPlans, six live rows pinned, two moved rows marked `planned — pair#186`; revert-verified — repointing the endsItsOwnChild row at console.go reds the contract on two IO seams.
  - id: BR-26
    disposition: addressed
    note: |
      The panel branch now has TestSwitcherRelaunchFindsTheThreadFromAnyDepth, and all eight still-claimed Done-when bullets are pinned by a named test.
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan lines 32-37 untouched; measured 15s CompletionTimeout (couch.go:119) + 5s resumeRegistrationTimeout (couch.go:107), child death inside the 15s (park.go:549-555).
  - id: BR-6
    disposition: not-addressed
    note: |
      relaunch_test.go:76 still has five refusal cases; agent-unsupported and profile-missing reach Relaunch in none of them.
  - id: BR-7
    disposition: not-addressed
    note: |
      Plan lines 159 and 352 still unticked for landed work; the issue's stale-estimate half IS now resolved by the Revisions rewrite.
  - id: BR-9
    disposition: not-addressed
    note: |
      manifest.go:524 still places relaunch.go between pathops.go and procops.go; threadreason.go:503 is still the second offender.
findings:
  - id: new
    severity: Minor
    family: parallel-derivation-drift
    title: |
      conceptPlans carries two contradictory accounts of the same deferral in one doc comment
    detail: |
      core_concepts_contract_test.go:196-225 is one contiguous comment holding two
      descriptions of the same deferred discovery work: "~10 rows across #151/#160 …
      tracked as its own issue" and "14 unpinned rows across pair#121, pair#181 and
      pair#182 … so it is pair#188". A reader cannot tell which measurement is current.
      2nd in family parallel-derivation-drift, so the rule rather than the line: when a
      comment's claim is superseded, REPLACE it — appending leaves two live accounts of
      one fact with no way to date them. ARCH-DRY.
  - id: new
    severity: Minor
    family: plan-record-lags-code
    title: |
      the moved endsItsOwnChild comment still cites "the switch below", which is no longer below it
    detail: |
      menu.go:1423 names the second hand-written list as "the switch below", but
      consumeExpectedParkExitLocked stayed in console.go when 58d3470d moved the function;
      what is below it now is operationNeedsProjectionRefresh. 4th in family
      plan-record-lags-code, so the rule: a relocation must carry its comment's positional
      references with it, or resolve them to names instead of positions.
```
