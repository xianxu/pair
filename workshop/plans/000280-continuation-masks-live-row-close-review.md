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
