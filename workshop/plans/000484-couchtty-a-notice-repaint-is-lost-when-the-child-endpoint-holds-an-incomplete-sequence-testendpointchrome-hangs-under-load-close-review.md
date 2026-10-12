# Boundary Review — pair#484 (whole-issue close)

| field | value |
|-------|-------|
| issue | 484 — couchtty: a notice repaint is lost when the child endpoint holds an incomplete sequence (TestEndpointChrome* hangs under load) |
| repo | pair |
| issue file | workshop/issues/000484-couchtty-a-notice-repaint-is-lost-when-the-child-endpoint-holds-an-incomplete-sequence-testendpointchrome-hangs-under-load.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9ca0ae3cfbdddf33b8a4818bb95dee6428fe15e2..dc3548373edafa0b3e13972d8a2039597ef0fb0c |
| command | sdlc close --issue 484 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T11:24:42-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Lock ordering looks consistent so far. Next I'll run the new tests, then revert the fix to check that they fail without it.

The new tests pass. Now I'm disabling the lock to confirm they go red without the fix, then running the whole package.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The fix works where it was aimed. `publishMu` orders the chrome paint and the resize-geometry path, and both new tests prove it. I generated the runtime bundle assets first (the package won't build without them). Then I swapped `publishMu` for a no-op lock in the review checkout, and both new tests failed exactly as the issue Log says: `"[brain]"` on the status row, and `terminal: incompatible chrome geometry`. With the fix restored, the targeted tests pass with `-race -count=3`, and the whole `couchtty` package passes with `-race` (117 s). Lock order is consistent: `publishMu` is always taken before `c.mu`, and `terminalError` and `Stop` never take `publishMu` again.

What keeps this from SHIP: the Spec says the lock covers "snapshot through publish and commit in every console-driven publication". The select path breaks that. `publishSelection` releases `publishMu` before `selectActorContext` commits `c.focus` and `c.active`. So the same race class is still open on the switcher→actor transition, and two of the four lock sites have no test.

**1. Strengths**
- The root cause was measured, not guessed: the trace in the Log shows the stale publish landing 18 ms late. The incomplete child sequence is correctly written off as incidental timing.
- The `publishHook` / `holdFirstPublish` seam (`terminal_migration_test.go:41`) pins the interleaving deterministically, and I confirmed both tests go red without the fix.
- `onResize` (`console.go:1362-1384`) holds the lock only around `c.size` plus `presenter.Resize`, and releases it before `child.Resize` and the follow-up paint. That keeps the critical section small with no re-entrancy.
- `publishSelection` is a clean extraction that removes the inline block in `selectActorContext` (ARCH-DRY passes).
- The `lessons.md` entry states the rule as a general principle, which makes it reusable.

**2. Critical findings:** none.

**3. Important findings**
- **The select path commits focus outside `publishMu`** (`terminal.go:165-175`, ARCH-PURPOSE). This is plausible from reading the code; I did not reproduce it.
  - `publishSelection` calls `presenter.Select` and then unlocks. `c.focus = FocusActor(id)` and `c.active = id` are set afterwards, in a separate `c.mu` section.
  - Failure scenario: the switcher is open and the operator picks an actor. A notice arrives from another goroutine (for example webcast's `GoTracked(setNotice)`) inside that gap. Its `paintNow` still sees `focus.IsPanel()==true` and calls `showMenuPublishing`. That puts the panel back up over the endpoint that was just selected, and sets `focus=Panel`.
  - `selectActorContext` then overwrites focus to actor, and its own `paintNow` gets an `IsRoutingAnswer` from `UpdateChrome`, which is silently dropped. The screen is left showing the switcher while focus says actor.
  - The `already` check (`terminal.go:159`) is also read outside the lock.
  - Fix sketch: read `already` and commit `active`/`focus`/`menu.ActiveAddress` inside `publishSelection`, under `publishMu`, once `Select` succeeds. That matches how `showMenuPublishing` commits `c.focus` under the lock.
  - Full list of snapshot/commit sites and their status: `paintNow` (fine), `showMenuPublishing` (fine, commits focus under the lock), `onResize` (fine), `publishSelection` (commit leaks out of the lock).
- **Two of the four ordered sites have no regression test** (`terminal.go:134`, `console_menu.go:211`).
  - Only the `"chrome"` and `"resize"` hook sites are tested.
  - `"select"` has a hook that no test uses.
  - The panel path has no hook at all.
  - If the lock were removed from `publishSelection` or `showMenu`, no test would fail.
  - Fix sketch: add a hook to `showMenuPublishing`. Then add one test that parks a select or panel publish while a notice tries to overtake it. That test would also cover the finding above.

**4. Minor findings**
- `console.go:1255`: the comment "Mid-transition: showMenu's presenter.Panel has cleared the endpoint and c.focus has not caught up yet" no longer describes a reachable case for `showMenu`, which now commits focus under `publishMu`. Today that branch is reachable only through the select-path gap above. Update the comment once that gap is closed.
- `console.go:73`: "Held across snapshot-to-commit" is true for `paintNow` and `showMenu`, but not yet for the select path (same family as the first Important finding).

**5. Test coverage notes**
- Both new tests are deterministic, need no load, and fail without the fix (verified by reverting).
- The 50 ms `time.After` is just a chance to overtake; it is not a correctness wait, so it can't flake the fixed build.
- The Log admits the load soak is a weak detector. The deterministic tests are the real evidence, and that holds for the paths they cover.

**6. Architectural notes**
- ARCH-DRY: passes. `publishSelection` consolidates the select logic; `showMenu` and `showMenuPublishing` is a clean locked/unlocked pair.
- ARCH-PURE: passes. This is ordering glue in the IO shell; the rendering (`RenderStatusRow`, `StyledRows`) stays pure.
- ARCH-PURPOSE: flagged. The "one ordering owner for presentation" invariant is enforced for state snapshots but not for the select path's state commit (first Important finding).
- `Present`, `Refresh` and `SetOverlay` stay outside the lock. That looks right, because they publish no snapshot of console state.
- No atlas or README surface was added.

**7. Plan revision recommendations**
- Add a `## Revisions` entry once fixed. It should say that `publishSelection` also commits `active`/`focus` under `publishMu`, and that the select and panel sites have a deterministic interleaving test.

```findings
findings:
  - id: new
    severity: Important
    family: publish-commit-outside-ordering-lock
    title: |
      Select path commits focus/active after releasing publishMu, so a concurrent paint can re-panel over the new selection
    detail: |
      terminal.go:165-175: publishSelection unlocks before selectActorContext sets c.active/c.focus, and the `already` read at :159 is also unlocked. A notice paint in that gap (switcher to actor) sees IsPanel()==true and calls showMenuPublishing, which re-panels over the endpoint just selected and sets focus=Panel; selectActorContext then overwrites focus to actor and its paintNow gets a routing answer that is silently dropped. Result: the screen shows the switcher while focus is the actor. All snapshot/commit sites: paintNow ok, showMenuPublishing ok, onResize ok, publishSelection leaks its commit. Fix: do the `already` read and the active/focus/menu.ActiveAddress commit inside publishSelection under publishMu.
  - id: new
    severity: Important
    family: ordering-site-without-regression-test
    title: |
      Only the chrome and resize sites have interleaving tests; select and panel publishes are unpinned
    detail: |
      Hook "select" (terminal.go:134) is never used by a test; showMenuPublishing (console_menu.go:211) has no hook. Removing publishMu from publishSelection or showMenu fails no test. Add a panel hook and one test that parks a select/panel publish while a notice tries to overtake it.
  - id: new
    severity: Minor
    family: stale-concurrency-comment
    title: |
      Comments claim invariants the select path does not yet meet
    detail: |
      console.go:1255 still says the mid-transition case comes from showMenu, which now commits focus under publishMu, so only the select gap can reach it. console.go:73 says the lock is held "snapshot-to-commit", which the select path does not do. Update both once the select commit moves under the lock.
```

---

## Re-review — 2026-10-12T12:54:55-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 484 — couchtty: a notice repaint is lost when the child endpoint holds an incomplete sequence (TestEndpointChrome* hangs under load) |
| repo | pair |
| issue file | workshop/issues/000484-couchtty-a-notice-repaint-is-lost-when-the-child-endpoint-holds-an-incomplete-sequence-testendpointchrome-hangs-under-load.md |
| boundary | whole-issue close |
| milestone | — |
| window | 373e763635f8f50a9a6e842efccb8a93cc0ca1de..b3980a69db28f0f627ceb120e23e780e7eda243e |
| command | sdlc close --issue 484 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T12:54:55-07:00 |
| verdict | SHIP |

## Review

Prior findings look addressed; checking one remaining comment claim and running the tests.

I found no blocking problems. All three findings from round 1 are fixed, the full test suite passes, and only a minor comment wording issue remains.

```verdict
verdict: SHIP
confidence: high
```

The fix makes the console's state snapshot and its publication to the presenter happen as one step, under a single lock (`publishMu`). It covers all four places that paint: `paintNow`, `showMenu`, the select path and `onResize`. The round-1 problem was that the select path committed the new focus only after releasing the lock. It now holds the lock from the `already` read through the commit of focus, active and `menu.ActiveAddress` (`terminal.go:157-180`). I checked the two new regression tests by reverting the fixes in this checkout:
- Without the select-path fix, `TestPaintDuringSelectionCannotRestoreTheMenu` fails ("the menu came back over the selected endpoint").
- Without the lock in `showMenu`, `TestNoticeDuringMenuPublicationIsShown` fails (row shows `"[brain]"`).

Both pass once the fixes are restored, and the full couchtty package passes under `-race` (116 s). One setup note: the package wouldn't compile in the review checkout until I ran the runtimebundle `generatecmd` to produce the embedded assets. That's an environment step, not a code problem. The only remaining item is one comment that claims more than the code guarantees.

1. **Strengths**
   - Each site has a single owner for the lock. `showMenu` and `showMenuPublishing` are split so that `paintNow`, which already holds the lock, never takes it twice (`console_menu.go:204-211`).
   - The test hook (`publishHook`/`beforePublish`) runs inside the locked span. That lets each test park one paint mid-publication and try to overtake it, so the tests are deterministic instead of relying on load.
   - The resize path holds the lock across both the size change and `presenter.Resize`, and still handles its error after unlocking (`console.go:1365-1389`).
   - The lesson recorded in `workshop/lessons.md` is a general rule, not a note about this one bug.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - The comment at `console.go:1255-1261` says "The console's own transitions can no longer show this." It names only `showMenu` and selection. But the attach path at `console.go:542` also sets `c.focus = FocusActor(...)` when `c.active == ""`, without going through `presenter.Select`. So at least one other console transition may still reach the "no endpoint" case the comment calls impossible. This is the 2nd finding in family `stale-concurrency-comment`. The rule for the whole family: a comment that says a state is unreachable must name the code that writes that state, so it can be checked. Rewording this comment fixes this instance and follows the rule.

5. **Test coverage notes:** All four publishing sites (chrome, panel, select, resize) now have tests that pin their interleaving, and I confirmed the select and panel tests fail without their fixes. The load soak is a weak detector, as the issue's own log admits; the deterministic tests are the real evidence.

6. **Architectural notes**
   - **ARCH-DRY: pass.** `publishSelectionPublishing` is the old select code moved into its own function, not a copy.
   - **ARCH-PURE: pass.** The lock is glue around existing calls to the presenter; it adds no new business logic.
   - **ARCH-PURPOSE: pass.** I went through every site the Spec lists, and each one now derives its ordering from `publishMu`. Calls like `Refresh`, `SetOverlay` and `Present` don't snapshot console state, so they stay outside the Spec's scope.
   - For future work: any new site that snapshots state and publishes it should go through this lock and add a matching `beforePublish` site.

7. **Plan revision recommendations:** none.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      terminal.go:157-180 holds publishMu from the already-read through the focus/active/ActiveAddress commit; reverting it makes TestPaintDuringSelectionCannotRestoreTheMenu fail.
  - id: BR-2
    disposition: addressed
    note: |
      Added panel hook (console_menu.go:234) and select/panel tests; reverting each lock makes its test fail (verified in this checkout).
  - id: BR-3
    disposition: addressed
    note: |
      console.go:73 snapshot-to-commit claim now holds on all four sites; the 1255 comment was rewritten (the remaining overclaim is a new Minor finding below).
findings:
  - id: new
    severity: Minor
    family: stale-concurrency-comment
    title: |
      paintNow routing-answer comment says the console's own transitions can no longer reach it, but the attach path sets FocusActor without a presenter Select
    detail: |
      This is the 2nd finding in family stale-concurrency-comment. console.go:1255-1261 claims the case is unreachable via the console's own transitions and names only showMenu and selection. console.go:542 (attach when c.active is empty and the attach is not background) sets c.focus=FocusActor without presenter.Select, so it may still reach this case. Rule for the family: a comment that says a state is unreachable must name the code that writes that state, so it can be checked. Instances in this window: console.go:1255 only (console.go:68-74 and terminal.go:154-156 now match the code).
```

---

## Re-review — 2026-10-12T12:57:56-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 484 — couchtty: a notice repaint is lost when the child endpoint holds an incomplete sequence (TestEndpointChrome* hangs under load) |
| repo | pair |
| issue file | workshop/issues/000484-couchtty-a-notice-repaint-is-lost-when-the-child-endpoint-holds-an-incomplete-sequence-testendpointchrome-hangs-under-load.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7da888f2155ad2deb14fa31771f0aaddefb5b5a4..be0e87d795bbc1f9546834430b89842aadb63e86 |
| command | sdlc close --issue 484 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T12:57:56-07:00 |
| verdict | SHIP |

## Review

Checking whether a mid-run Attach is always followed by a selection, so I can verify the new comment's claim.

Confirming whether a mid-run attach is followed by a switch (the "selection that follows" the comment relies on).

```verdict
verdict: SHIP
confidence: high
```

BR-4 is fixed, and this window contains no new defects. The only code change rewrites the comment on the `paintNow` routing-answer branch (`cmd/internal/couchtty/console.go:1256-1263`). The comment no longer claims the console's own transitions can never reach the branch. It now names the code that writes the state: `showMenu` and selection set `c.focus` together with the presentation under `publishMu`, so they can't reach it, but `Attach` still can. That follows the rule stated for the `stale-concurrency-comment` family. The other two files are generated gate and review records under `workshop/plans/`. The interdiff doesn't touch `## Spec`, `## Revisions` or `## Done when`, so there's no scope change. There are no conflict resolutions to judge.

1. **Strengths**
   - The comment names `Attach` as the writer. I checked it against the code: `console.go:542` sets `c.focus = FocusActor(handleID)` when `c.active == ""` and the attach isn't background, without calling `presenter.Select`. At startup, `Run` (`console.go:688-694`) then calls `switchTo(initial, ...)`, which is "the selection that follows" the comment describes.
   - The reason for skipping is kept, and still correct: escalating here would cause the pair#265 exit with no input involved.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - Not raised, just for the record: the new wording says the attach reaches this branch "before Run's first selection". That describes startup only. An attach can also happen mid-run, through the `attach` operation (`console.go:2122-2146`), when the last pane has exited and `c.active` is empty. A non-background resume still gets a following selection (`forceSwitch`, `console.go:2042-2045`). I didn't trace whether every other non-background start is also followed by one. The comment correctly names the code that writes the state, which is what the family rule asks for, so I'm not raising this. If it's ever reworded, "before a selection presents the endpoint" would cover both cases.

5. **Test coverage notes:** The change is comment-only, so no regression test is needed. The behavioural tests for the ordered sites were confirmed in round 2.

6. **Architectural notes**
   - **ARCH-DRY: pass.** The change is comment-only.
   - **ARCH-PURE: pass.** No logic changed.
   - **ARCH-PURPOSE: pass.** BR-4 named a single instance, and this window confirms the comment was the only one; it now follows the family rule.

7. **Plan revision recommendations:** none.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      console.go:1256-1263 now names the writers: showMenu/selection commit focus under publishMu, and Attach (console.go:542) can still set FocusActor before a selection presents the endpoint. Verified against the Run path (console.go:688-694).
```
