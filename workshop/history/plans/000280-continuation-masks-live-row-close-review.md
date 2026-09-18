# Boundary Review — pair#280 (whole-issue close)

| field | value |
|-------|-------|
| issue | 280 — A retained continuation failure masks a live thread's state in the switcher |
| repo | pair |
| issue file | workshop/issues/000280-continuation-masks-live-row.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8a4f445b2511939530b54cfe7d09557ba050ed66..5070c2daccc9a12bb3de34455eb46e9fc6ca44a8 |
| command | sdlc close --issue 280 |
| reviewer | claude |
| timestamp | 2026-09-17T22:38:57-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The code is correct and I'd ship it: the retry bug is reproduced and fixed, dismissal is a proper CAS store transition, and the state text and actions compose. But this gate's Core-concepts rule makes one Critical finding block the close. The plan lists `ThreadStore.DismissFailedContinuation` and `Couch.DismissContinuation` as PURE entities, yet both read and write the file-backed store, and their tests run against a temp-dir store. That is the mismatch `workshop/lessons.md` already records as a lesson ("PURE fixtures must be literal at their direct boundary", #149). The fix is prose only: one `## Revisions` entry. There are also two Important findings, both cheap. Each is a documented claim that the code doesn't fully back: why `Pending`/`Running` are safe to leave displacing the state, and "every refusal of a failed request names both exits".

**What I ran.**
- All the #280 tests pass.
- I checked three mutations in a scratch copy, and each turned the tests red:
  - the switcher sending `ref` again;
  - `relaunch` removed from `ContinuationRefuses`;
  - the store's `Failed` precondition removed.
- I could not confirm the full `go test ./...` claim here. The PTY-child tests in couchcore, couchtty, couchcmd and launcher fail in this environment with "operation not permitted". That is the environment, not this diff.

## 1. Strengths
- **The retry bug was proven on the real path before it was fixed.** `TestContinuationFailedRowKeepsAnExplicitRetry` (`console_continuation_test.go:57-76`) moved off a fake onto `DispatchOperation` + `CouchLiveOwnerExecutor`. Putting `ref` back turns both switcher tests red.
- **The store transition is clean.** `DismissFailedContinuation` (`continuation_store.go:64-79`) follows the revision-CAS pattern of its siblings. Every refusal is checked for writing nothing, by an unchanged revision (`continuation_store_test.go:171-213`). Retry vs dismiss is driven in both orders.
- **The action filter can still fail.** `TestContinuationRefusesMatchesTheGuardForEveryRowAction` (`continuation_guard_test.go:76-133`) takes its oracle from how the guard actually behaves. A listed operation must be refused by the guard's own wording, and an unlisted one must succeed. The filter isn't correct just by how it's built.
- **Deleting instead of adding a terminal phase is well reasoned.** Records decode strictly, so a new phase would break older binaries that are still running. The reasoning is recorded in the code, the plan and the atlas.
- **The state × phase table** (`menu_continuation_test.go:16-44`) runs over `AllThreadStates()`.

## 2. Critical findings
- **The Core concepts table calls IO entities PURE** (`workshop/plans/000280-dismiss-continuation-plan.md:36-37`, ARCH-PURE).
  - The rows `ThreadStore.DismissFailedContinuation` and `Couch.DismissContinuation` sit under "Pure entities". Both do store IO, and their tests need a mutable filesystem.
  - The dismissal rule itself is an anonymous closure, so no test can reach it without IO.
  - **Fix:** add a Revisions entry that moves both rows to "Integration points" (they wrap the file-backed store). Optionally, pull the rule out as a pure `dismissFailed(*ThreadRecord, requestID) error` with literal-value tests, and list that as the PURE row.

## 3. Important findings
- **The claim that `Pending`/`Running` are "bounded" overstates what the code does** (`menu_render.go:419-421`, `menu.go:1259`, `atlas/couch.md:163`, and the issue's Revisions; ARCH-ORDER).
  - The 30s deadline (`continuation_recovery.go:196`) only starts once `r.Target != nil`.
  - It only runs while this Couch's console scans the address, and `continuationAddresses()` covers only open panes and existing watches.
  - So a `Running` or `Pending` request left behind when the owner dies, on a thread with no pane, reads "continuing…" or "continuation queued" forever. That is the same stale-request bug this issue fixes.
  - **Cheap fix:** narrow the stated bound to "while a live owner watches the thread" and name the orphaned case and its way out (Retry on the recovery row). Or compose these phases too for rows that aren't live.
- **"Every refusal of a failed request names both exits" isn't true yet** (`atlas/couch.md:160`, the issue Log; ARCH-PURPOSE). These refusals are caused by a retained failed request but don't mention dismiss:
  - `recovery_execute.go:179` and `:266` ("another continuation is unresolved; recover its retained checkpoint or archive");
  - `archiveContinuationVacant` (`detach.go:436`, `:447`);
  - the warm-reattach refusal from `validateContinuationWarm` (`resume.go:479`).
  - **Fix:** route these through `continuationExits(phase)` when the phase is `Failed`, or narrow the atlas and Log claim to the guard, publish and `pair continue`.

## 4. Minor findings
- `var menuLiveActions` was inserted between `menuActionItems`' doc comment and the function (`menu.go:1205-1214`), so that doc now attaches to the var. Move the var above the comment.
- ARCH-DRY:
  - The exits for each phase are spelled out twice (`continuationExits` / `continuationExitsFor`, `continuation.go:378-389`), plus a third time in `runcli.go:147`.
  - `DismissContinuation`'s 8-try stale-revision loop copies `advanceContinuation`'s (`continuation.go:154-167`); a shared retry-on-stale helper would serve both.
- The state-text test lists the phases by hand, and there is no `checkpoint.AllPhases()`. A new phase added with a displacing case would go unnoticed.
- The `continuationArguments(bootstrap bool)` parameter now also serves dismiss, which isn't a bootstrap. The name misleads.
- `c.menu.Orientation[address]` is only cleared when a request completes. After a dismissal, "Copy orientation prompt" stays on offer until Couch exits (in memory only).
- `ContinuationRefuses`' doc says a table test proves the list. The test only drives `relaunch` and `prepare-switch-agent` from the list; `switch-agent` (the accept step), `resume` and `start` are not driven.
- The switcher shows the raw `dismiss-continuation` (there's no `menuItemLabel` entry), while the messages and README say "Dismiss continuation". Retry already had this mismatch.

## 5. Test coverage notes
- The new tests exercise the production dispatchers over a real temp-dir store, and the mutations show they catch regressions.
- Entity sweep: `TestRowActionDeclarationsAndTheMenuAgreeInBothDirections` needed no change, because its Unusable + `Failed` row already offers dismiss.
- Enter reachability: `TestEveryOfferedActionIsReachableFromEnter` still never builds a continuation row, so pressing Enter on dismiss or retry isn't covered by that sweep.
- Nothing tests a `Running` request with no watcher, which is the case the bounded-displacement claim depends on.

## 6. Architectural notes for upcoming work
- ARCH-DRY: flag (minor, above). `menuLiveActions` and the reuse of `requestRecord` are good.
- ARCH-PURE: flag (the Critical table finding). The switcher functions are genuinely pure.
- ARCH-PURPOSE: flag (the "every refusal" sweep). Otherwise the four Done-when items, the relaunch unblock and the retry repair are delivered.
- ARCH-MOCK: pass. Production executors run over the portable file store, and the retry test left its fake.
- ARCH-CONSTRAINTS: pass. One CAS write, no process effects.
- ARCH-SECURE: pass. The request ID must match exactly, strict decoding is untouched, and there's no new persisted shape.
- ARCH-ORDER: flag (the `Pending`/`Running` bound). The race between CAS writes and the stale-ID cases are tested.
- ARCH-FUNERAL: pass. Dismissal is how a failed request ends, and the checkpoint copy is bounded at one per thread.
- Once #278 unifies `--list` and the switcher's state text, compose continuation state from one shared helper rather than per-surface text.

## 7. Plan revision recommendations
- **Revisions (ARCH-PURE):** move `ThreadStore.DismissFailedContinuation` and `Couch.DismissContinuation` to Integration points (wrapping the file-backed thread store). If the rule is extracted, name it as the PURE row.
- **Revisions (ARCH-ORDER):** `Pending`/`Running` are bounded only while a live owner watches the thread. Record the orphaned-owner case and its exit.
- **Revisions (ARCH-PURPOSE):** either list which refusals name both exits, or record the sweep extension to `recovery_execute`, archive and warm reattach.

```findings
findings:
  - id: new
    severity: Critical
    family: pure-row-must-be-io-free
    title: |
      Core concepts table lists DismissFailedContinuation and Couch.DismissContinuation as PURE, but both do store IO
    detail: |
      Plan lines 36-37. Both read and write the file-backed thread store, and their tests need a temp-dir store (mutable filesystem). The dismissal rule is an inline closure no test can reach without IO. Add a Revisions entry moving both rows to Integration points, or extract the pure rule and name it (lessons.md, "PURE fixtures must be literal at their direct boundary").
  - id: new
    severity: Important
    family: unbacked-existing-behavior-claim
    title: |
      The stated reason Pending/Running may keep displacing the state (bounded in time) does not hold without a watching owner
    detail: |
      The 30s deadline (continuation_recovery.go:196) starts only once Target is set, and runs only while the console scans the address (panes plus watches). A Running or Pending request left by a dead owner on a thread with no pane reads "continuing…" forever. Narrow the claim in menu_render.go:419, menu.go:1259, atlas/couch.md:163 and the issue Revisions, or compose these phases on rows that are not live.
  - id: new
    severity: Important
    family: refusal-names-every-exit
    title: |
      "Every refusal of a failed request names both exits" is false for recovery, archive and warm-reattach refusals
    detail: |
      recovery_execute.go:179 and :266, archiveContinuationVacant (detach.go:436,447) and the validateContinuationWarm refusal (resume.go:479) are caused by a retained failed request but never mention dismiss. Route them through continuationExits(phase), or narrow the atlas (couch.md:160) and issue Log claim.
  - id: new
    severity: Minor
    family: doc-comment-attachment
    title: |
      menuLiveActions was inserted between menuActionItems' doc comment and the function
    detail: |
      menu.go:1205-1214. The doc now attaches to the var; move the var above the comment block.
  - id: new
    severity: Minor
    family: single-source-per-fact
    title: |
      Exits for each phase are written twice (continuationExits and continuationExitsFor), and the stale-revision loop is copied from advanceContinuation
    detail: |
      continuation.go:378-389 plus runcli.go:147. DismissContinuation's 8-try loop duplicates advanceContinuation (continuation.go:154-167); a shared retry-on-stale helper would serve both.
  - id: new
    severity: Minor
    family: vocabulary-enumerated-by-test
    title: |
      The state-text test lists continuation phases by hand, with no checkpoint.AllPhases()
  - id: new
    severity: Minor
    family: naming-matches-role
    title: |
      The continuationArguments(bootstrap) parameter now also serves dismiss, which is not a bootstrap
  - id: new
    severity: Minor
    family: in-memory-state-follows-record
    title: |
      menu.Orientation survives a dismissal, so Copy orientation prompt stays on offer for a dropped handoff
  - id: new
    severity: Minor
    family: doc-claims-match-test-reach
    title: |
      ContinuationRefuses' doc says a table test proves the list, but switch-agent, resume and start are not driven
```

---

## Re-review — 2026-09-17T23:01:19-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 280 — A retained continuation failure masks a live thread's state in the switcher |
| repo | pair |
| issue file | workshop/issues/000280-continuation-masks-live-row.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8a4f445b2511939530b54cfe7d09557ba050ed66..d4f60372a21c303d34339625e426370342b2be44 |
| command | sdlc close --issue 280 |
| reviewer | claude |
| timestamp | 2026-09-17T23:01:19-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The dismiss feature is sound, and seven of the ten prior findings are disposed below: BR-1, BR-2, BR-5, BR-6, BR-8, BR-9 and BR-10 as addressed, BR-3 and BR-4 as not-addressed. BR-7's named test was fixed, but its siblings remain, so they are raised again as a new finding. The core pieces work: the store transition, the declared operation, the retry repair through the production dispatcher, and the composed state text and actions. Two problems block SHIP:

- **The fix for BR-9 broke switch-agent.** The new line `delete(c.menu.Orientation, address)` at `console_continuation.go:169` fires for every address the continuation scan visits that has no request. That includes every hosted thread with a pane, so switch-agent's **Copy orientation prompt** is wiped within one 500 ms tick. I reproduced this in a scratch copy of HEAD. A scan tick removed a switch-agent prompt on a thread with no continuation. The test passes again when line 169 is removed, and then the new `TestVanishedRequestTakesItsOrientationPromptWithIt` fails. So this diff introduced the regression.
- **BR-3 and BR-4 each have a leftover instance.**
  - BR-3: the plan still says "Both are bounded in time", and the atlas says a Pending row keeps retry, which it doesn't.
  - BR-4: one refusal inside the retained-request block still doesn't name the exits.
  - The test that claims every wrapped site is driven actually drives 3 of 7.

1. **Strengths**
   - The pure dismissal rule `checkpoint.CheckDismissible` (`request.go:54-65`) is tested on literal requests with no IO. This properly settles BR-2.
   - The retry bug was reproduced through the production path (`DispatchOperation` + `CouchLiveOwnerExecutor` over a temp-dir store; `console_continuation_test.go:58-77`) before it was fixed. The assertion is RetryContinuation's own first refusal, not a bare `err == nil`.
   - Every store refusal is checked two ways: by its message, and by an unchanged revision (`continuation_store_test.go:174-217`). Both retry/dismiss orders are driven through the revision check.
   - `TestContinuationRefusesMatchesTheGuardForEveryRowAction` walks every declared row action. Each one is either driven through the production dispatcher or exempt with a stated reason, so a new action can't arrive unclassified. This properly settles BR-1.
   - Deleting the request rather than adding a phase is well argued: records decode strictly, so a new phase would break older binaries. The reasoning is recorded in the store, the operation and the atlas.

2. **Critical**
   - `cmd/internal/couchtty/console_continuation.go:169`: the scan now deletes orientation prompts it didn't create, including switch-agent's (full detail in the findings block below).

3. **Important**
   - BR-3 not-addressed: the plan and atlas claims (see dispose notes).
   - BR-4 not-addressed: `recovery_execute.go:268` still doesn't name the exits.
   - `continuation_guard_test.go:165`: "Each site is driven" is false. Four wrapped sites are never reached.

4. **Minor**
   - Phase lists are still written out by hand in two tests, now that `checkpoint.AllPhases` exists.
   - `runcli.go:147` names both exits whatever the request's phase.
   - Retry and dismiss are appended to the action list in three separate branches of `menuActionItems` (`menu.go:1223-1226,1250,1257`). It's small now; a helper would keep the order in one place.

5. **Test coverage notes**
   - No test combines a switch-agent orientation prompt with a continuation scan tick. That gap is why the Critical passed `go test ./...`.
   - The guard-agreement test detects a guard refusal by the phrase "Dismiss continuation drops it". `withContinuationExits` produces the same phrase, so the test can't tell the guard from a wrapped non-guard refusal. That makes its "refused BY THE GUARD" claim wider than what it checks.
   - The launcher's hosted-thread refusal has no test.

6. **Architecture notes**
   - **ARCH-DRY:** pass. There is one `checkpoint.Exits` and one `writeRequestRecord` loop. The only duplication left is the minor menu one above.
   - **ARCH-PURE:** pass. `CheckDismissible`, `rootStateText`, `continuationLabel` and `menuActionItems` are pure.
   - **ARCH-PURPOSE:** flag. The BR-4 sweep missed a site in the same block, and the test-reach claim overstates coverage.
   - **ARCH-MOCK:** pass. Tests use the real temp-dir store and the production executors.
   - **ARCH-CONSTRAINTS:** pass. Dismissal is one guarded write, and the scan cadence is unchanged.
   - **ARCH-SECURE:** pass. `request-id` must match exactly, records are still read strictly, and there is no new persisted shape.
   - **ARCH-ORDER:** flag. `menu.Orientation` is keyed by address, has two writers, and records no provenance. The pruning rule can't be stated correctly until each entry records which writer made it. Store-level ordering is fine.
   - **ARCH-FUNERAL:** pass. Dismissal gives a failed request an end. Couch's checkpoint copy stays bounded at one per thread and is replaced on the next publish or removed at archive.

7. **Plan revision recommendations**
   - Rewrite the `rootStateText` bullet (plan lines 82-87). It currently says both "that was wrong" and "Both are bounded in time". It should say that in-flight phases are not bounded without a watching owner.
   - `continuationGuard` bullet (plan line 65): Pending/Running messages DID change; they now read "Retry continuation reconciles it" via `checkpoint.Exits`.
   - Revisions entry for BR-4: either list the sites that are actually driven, or record the per-site table plus the call-site check.
   - Add a Revisions entry recording that orientation prompts now carry provenance, so the continuation scan prunes only its own.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Composition is live-only (menu.go:1238) and menuLiveActions holds no archive; the guard-agreement test loops every declared RowAction (driven or exempt with a reason).
  - id: BR-2
    disposition: addressed
    note: |
      Pure table now lists CheckDismissible (literal-request tests in request_dismiss_test.go); both IO entries sit in Integration points (plan lines 112-118).
  - id: BR-3
    disposition: not-addressed
    note: |
      Code, issue and atlas:158 fixed, but plan:83 still asserts "Both are bounded in time... Pending is picked up by the owner's scan", and atlas:165 says an in-flight request "keeps the restricted retry set" while a Pending row offers only name/describe (menu.go:1265, pinned by TestFailedContinuationComposesWithALiveRowsActions).
  - id: BR-4
    disposition: not-addressed
    note: |
      Seven sites wrapped, but recovery_execute.go:268 (AdmitRecoveryGeneration "...inspect or archive"), inside the same retained-request block, is not. Wrap at the block's return rather than per site.
  - id: BR-5
    disposition: addressed
    note: |
      menuLiveActions now sits above menuActionItems' doc comment (menu.go:1205-1208).
  - id: BR-6
    disposition: addressed
    note: |
      One checkpoint.Exits; writeRequestRecord serves advanceContinuation and DismissContinuation; the duplicate helpers are gone.
  - id: BR-7
    disposition: addressed
    note: |
      The state-text test iterates checkpoint.AllPhases; sibling hand lists raised separately.
  - id: BR-8
    disposition: addressed
    note: |
      Renamed operatorFacing; no bootstrap references remain.
  - id: BR-9
    disposition: addressed
    note: |
      Dismissal now prunes the prompt, but the fix over-prunes: see the new Critical in the same family.
  - id: BR-10
    disposition: addressed
    note: |
      The doc names what is driven; SwitchAgent re-runs PrepareAgentSwitch (switchagent.go:256), which holds the guard.
findings:
  - id: new
    severity: Critical
    family: in-memory-state-follows-record
    title: |
      The continuation scan now deletes switch-agent's Copy orientation prompt within 500 ms (console_continuation.go:169)
    detail: |
      This is the 2nd finding in family in-memory-state-follows-record. continuationAddresses() covers every pane, and ContinuationRequests returns no status for a record without a request, so line 169 deletes menu.Orientation for every hosted thread on each tick. That includes the entry watchOrientation writes (console_switchagent.go:37) and that finishOrientation deliberately keeps after an unconfirmed delivery, whose notice says "Copy orientation prompt is available in actions". Reproduced on HEAD in a scratch copy: the test fails, and passes once line 169 is removed. Rule (ARCH-ORDER): console state that mirrors a record fact carries the identity of the fact that produced it, and is pruned only when THAT fact vanishes. menu.Orientation has two producers and no provenance. Record the producer (for example, the continuation request ID) and let the scan prune only continuation-produced entries. Sweep every prune site under that rule: line 169 (new); line 117 (pre-existing, same defect for a record that retains a Complete request); line 222. Add a regression test with a switch-agent prompt that survives a scan tick.
  - id: new
    severity: Important
    family: doc-claims-match-test-reach
    title: |
      The claim that each exits-wrapped refusal site is driven is false: 3 of 7 sites are reached
    detail: |
      This is the 2nd finding in family doc-claims-match-test-reach. The test comment (continuation_guard_test.go:165), the plan's BR-4 revision and the issue Log all claim every withContinuationExits site is driven. The test reaches only recovery_execute.go:179, detach.go:436 and continuation_recovery.go:34. Four sites are never reached: recovery_execute.go:266, :272 and :278, and detach.go:447. Rule: a claim of test reach over a set of sites is checked by the test against that set, or it names exactly the subset driven. Here, a table with one row per wrapped site, plus a source scan asserting that the count of withContinuationExits( call sites equals the table's rows. The guard-agreement test's "refused BY THE GUARD" has the same problem: its oracle phrase is shared with withContinuationExits. Match the guard's own "continuation <id> is failed; " prefix.
  - id: new
    severity: Minor
    family: vocabulary-enumerated-by-test
    title: |
      Continuation phases are still written out by hand in menu_action_sweep_test.go:88 and checkpoint/recovery_request_test.go:61
    detail: |
      This is the 2nd finding in family vocabulary-enumerated-by-test. Rule: once a vocabulary has an All*() enumerator, no test restates it. Replace both lists with checkpoint.AllPhases(), and add a check that a []Phase{ literal appears only in AllPhases.
  - id: new
    severity: Minor
    family: refusal-names-every-exit
    title: |
      pair continue --retry on a hosted thread names both of Failed's exits regardless of the request's phase (runcli.go:147)
    detail: |
      This is the 2nd finding in family refusal-names-every-exit. Rule: a refusal names exactly the exits valid for the request's actual phase, through checkpoint.Exits(phase, tag). A site that does not know the phase uses phase-neutral wording rather than assuming Failed. Here, dismiss is offered even when the hosted request may be pending or running, which CheckDismissible refuses.
```

---

## Re-review — 2026-09-17T23:18:52-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 280 — A retained continuation failure masks a live thread's state in the switcher |
| repo | pair |
| issue file | workshop/issues/000280-continuation-masks-live-row.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8a4f445b2511939530b54cfe7d09557ba050ed66..e46e6b625269b1ef50035baaa6b72a2467d70d5f |
| command | sdlc close --issue 280 |
| reviewer | claude |
| timestamp | 2026-09-17T23:18:52-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All six open findings from round 3 are fixed. I checked each against the code and, where behaviour changed, with a mutation in a scratch copy of HEAD. Each fix's test goes red without it.
- **Orientation prompts (BR-11):** prompts now record which producer wrote them. The round-3 regression (the scan pruning every hosted thread's prompt) turns `TestSwitchAgentOrientationPromptSurvivesContinuationScans` red. So does a prune that ignores the producer.
- **Refusal wording (BR-4, BR-12):** each check wraps its refusals once, where the check returns. The routing test has one row per call site plus a scan that the call sites are the rows. Removing the wrap on the retained recovery branch fails both the row and the scan.

Two new Minor findings, neither blocking:
- **Stranded prompt:** the producer fix leaves a gap. When one request replaces another before the scan sees the first one disappear, the first request's prompt is never removed. I reproduced this. It is the third finding in `in-memory-state-follows-record`, so the recommendation is the rule, not a patch at one site.
- **Orphaned doc comment:** a new function was inserted under another function's doc comment.

**Strengths**
- **Provenance helpers:** `setOrientationLocked` is the only writer, `dropOrientationLocked` only prunes its own producer's entry, and `supersedeOrientationLocked` is an explicit wipe (`console_continuation.go:94-120`). That is the right shape for a map with two writers.
- **One wrap per check:** extracting `admitRetainedRecovery` (`recovery_execute.go:230`) lets its one caller wrap every refusal once. That catches the missed `AdmitRecoveryGeneration` case by construction, not site by site.
- **Guard-agreement test:** it checks both the guard's own prefix (`continuation <id> is failed; `, which the wrapper never produces) and that a refusal writes nothing (`continuation_guard_test.go:139-150`). This catches an operation that parks first and refuses later in the same words.
- **Phase-list scan:** `TestPhaseListIsWrittenOnlyInAllPhases` scans all of `cmd/`. Putting a hand list back in `menu_action_sweep_test.go:88` turns it red.
- **Dismissal store write:** `CheckDismissible` is a pure rule tested on literal requests. The store applies it inside its revision check, and a command-line dismiss without an id binds to the request read at that revision (`continuation.go:421-423`).

**Critical findings:** none.

**Important findings:** none.

**Minor findings**
1. **Stranded prompt when a request is replaced** (`console_continuation.go:145-147`). This is the 3rd finding in family `in-memory-state-follows-record`.
   - **What happens:** when the scan sees a new request ID for an address, it replaces the watch but never drops the prompt the old request produced. After that, the only prunes compare against the new request's producer, so the old prompt and its `orientationFrom` entry stay until a switch-agent launch or a Couch restart. "Copy orientation prompt" stays offered for a handoff that no longer exists.
   - **Reproduced:** prompt from request A → scan shows request B → B completes → B disappears. The prompt survives, and the actions are `[detach relaunch park switch-agent name describe copy-orientation]`.
   - **Regression:** the base (`8a4f445b`) deleted the address's prompt when any request on it completed.
   - **When it bites:** A disappears and B appears within one scan interval. The window is unbounded while a partial scan error keeps the disappearance loop disabled.
   - **The rule:** a view of a record fact lives inside, or is derived from, the object that tracks that fact's identity. Pruning at a list of events is not enough, and this family's three findings are three events that list missed (dismissal, a scan with no request, replacement). Concretely: keep the continuation's prompt on `continuationWatch`, so replacing or deleting the watch removes it. Or reconcile in one place after every change to `c.continuations`: drop any continuation-produced entry whose request ID differs from the address's current watch. Add a replacement-sequence test.
2. **Orphaned doc comment** (`recovery_execute.go:224-230`). This is the 2nd finding in family `doc-comment-attachment`.
   - `admitRetainedRecovery` was inserted under `prepareAbsentContinuation`'s doc comment. The combined comment now starts "prepareAbsentContinuation snapshots…" and attaches to the wrong function, and `prepareAbsentContinuation` has no doc.
   - **The rule:** a Go doc comment begins with the name of the declaration it documents. A package-level scan can enforce it: fail when a function's doc starts with the name of another declaration in the same file.
   - **Prevalence:** a rough scan of the changed files found this one new instance. The other hits were pre-existing prose.

**Test coverage notes**
- **Passing at HEAD:** targeted runs of checkpoint, couchcore (continuation, dismiss, refusal, operation tests) and couchtty (continuation, orientation, sweep, state-text) all pass.
- **Not run here:** tests that start pty children or write under `/tmp` fail in this environment with "operation not permitted", unrelated to this diff. The implementor's full `go test ./...` run is the evidence for those.
- **Pinned by tests:** the new refusal-routing table and scan, the guard's write-nothing check and the phase-list scan each fail when their fix is reverted.
- **Not pinned:** no test covers the `pair continue --retry` call site in `runcli.go:147` itself. Changing it to `Exits(checkpoint.Failed, …)` would keep everything green. The phase-neutral wording is pinned in `Exits` itself, which is acceptable for a message-only Minor.

**Architecture principles**
- **ARCH-DRY: pass.** There is one exits wording, one retry-on-stale loop and one writer of `menu.Orientation`.
- **ARCH-PURE: pass.** `CheckDismissible`, `Exits`, `ContinuationRefuses`, `withContinuationExits`, `rootStateText` and `menuActionItems` are pure.
- **ARCH-PURPOSE: pass.** Every Done-when item is delivered. The `Settled()` predicate is declined with a stated reason.
- **ARCH-MOCK: pass.** Tests use a temp-dir store and the production dispatcher and executors.
- **ARCH-CONSTRAINTS: pass.** Nothing new runs on hot paths. The scans are test-time only.
- **ARCH-SECURE: pass.** Dismissal goes through `resolveOperationThread` and the store's revision check. Messages carry only the tag.
- **ARCH-ORDER: flag.** Minor 1: pruning at a list of events rather than following the fact's identity.
- **ARCH-FUNERAL: pass, with one gap.** Dismissal names its end, and Couch's private checkpoint copy is named as left for replacement. The gap: `orientationFrom` entries strand with the prompt (Minor 1).

**Plan revision recommendation:** if Minor 1 is fixed by moving the prompt onto `continuationWatch`, add a Revisions entry. The entry should say that continuation-produced prompts are owned by the watch, not pruned by producer at each event.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      Plan:83 now states the "bounded in time" claim is false; atlas:155-165 and menu.go:1258-1263 describe the per-phase sets as statements of which actions are offered, not time bounds, matching menuActionItems.
  - id: BR-4
    disposition: addressed
    note: |
      All four non-guard checks wrap once where they return (RecoverThread:179, prepareAbsentContinuation:291 via admitRetainedRecovery, archiveContinuationVacant, validateContinuationWarm); unwrapping :291 in a scratch copy turns the row and the site scan red.
  - id: BR-11
    disposition: addressed
    note: |
      Each prompt records its producer; the round-3 unconditional prune and a producer-blind prune each turn TestSwitchAgentOrientationPromptSurvivesContinuationScans red. A replacement-sequence gap in the same family is raised separately.
  - id: BR-12
    disposition: addressed
    note: |
      One row per withContinuationExits call site (4), each driven, plus a productionCallsTo scan that the call sites equal the rows; the guard oracle matches the guard's own "continuation <id> is failed; " prefix and requires an unchanged revision.
  - id: BR-13
    disposition: addressed
    note: |
      Both lists use AllPhases; restating the list in menu_action_sweep_test.go:88 turns TestPhaseListIsWrittenOnlyInAllPhases red.
  - id: BR-14
    disposition: addressed
    note: |
      runcli.go:147 uses Exits("", tag), whose conditional dismiss wording TestExitsWithAnUnknownPhaseAreConditional pins; the call site itself is not pinned (acceptable for a message-only Minor).
findings:
  - id: new
    severity: Minor
    family: in-memory-state-follows-record
    title: |
      A request replaced before the scan sees it vanish strands its orientation prompt (console_continuation.go:145-147)
    detail: |
      This is the 3rd finding in family in-memory-state-follows-record. When the scan sees a new request ID it replaces the watch without dropping the old request's continuation-produced prompt; every later prune compares against the new producer, so Copy orientation prompt stays offered until a switch-agent launch or restart. Reproduced (A's prompt, B seen, B complete, B vanished: prompt survives); base cleared it on any completion for the address. Window: A vanishes and B appears within one scan interval, or while a partial scan error disables the vanish loop. Rule: a view of a record fact lives inside, or is derived from, the object that tracks that fact's identity, not pruned at an enumerated list of events (this family's three findings are three missed events: dismissal, a scan with no request, replacement). Fix: keep the continuation's prompt on continuationWatch, or reconcile continuation-produced entries against the current watch's request ID in one place after every change to c.continuations. Add a replacement-sequence test.
  - id: new
    severity: Minor
    family: doc-comment-attachment
    title: |
      admitRetainedRecovery was inserted under prepareAbsentContinuation's doc comment (recovery_execute.go:224-230)
    detail: |
      This is the 2nd finding in family doc-comment-attachment. The combined comment starts "prepareAbsentContinuation snapshots..." but attaches to admitRetainedRecovery, and prepareAbsentContinuation has no doc. Rule: a Go doc comment begins with the name of the declaration it documents; enforce it with a package-level scan that fails when a function's doc starts with the name of another declaration in the same file. Prevalence: 1 new instance in this diff (BR-5 was the first in the family).
```
