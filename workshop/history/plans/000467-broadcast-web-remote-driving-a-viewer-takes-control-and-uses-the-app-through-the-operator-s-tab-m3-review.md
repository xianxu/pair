# Boundary Review — pair#467 (milestone M3)

| field | value |
|-------|-------|
| issue | 467 — broadcast-web: remote driving — a viewer takes control and uses the app through the operator's tab |
| repo | pair |
| issue file | workshop/issues/000467-broadcast-web-remote-driving-a-viewer-takes-control-and-uses-the-app-through-the-operator-s-tab.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 28b9cf0507978b3dd66a6656a507c2e80851f943..fbf49dd7e672bd2957a0ff5eafad00bead0ea6c7 |
| command | sdlc milestone-close --issue 467 --milestone M3 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T08:22:10-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Diff read; now checking menu routing, sanitization, and controller semantics, then running the tests.

Sanitization checks out. Running the M3 tests under `-race` now.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M3 does what the plan's Task 10 asks for. Couch shows a drive span after the Chrome cell and `💭N`. A click opens a "Remote driving" menu, which reuses the notes menu's box and gesture router. In that menu a left click grants, a right click denies, and any click on the driver's row revokes. Ctrl+click on the span revokes whether the menu is open or closed. The grant notice says what it turns off. The span clears when the cast ends. I ran `go test -race -count=20 -run 'TestDrive|TestRenderDriveSpan' ./cmd/internal/couchtty/` and it passed (14.2 s). `TestCastDriving -race -count=5` also passed. To build couchtty I first had to generate the runtime bundle with `go run ./cmd/internal/runtimebundle/generatecmd`. No new correctness defect needs to be fixed before the boundary. Two things should be fixed before it, and both are cheap:
- **Plan drift:** the Core concepts table and Task 10 still name `couchtty/drive_menu.go` and a `DriveMenu` model. No such file exists. Neither the table nor the fake that replaced "FakeBrowser with its controller" has a `## Revisions` line. This is the third time in this family.
- **Error notice:** every failed control op tells the operator "that changed meanwhile". That includes a revoke that genuinely failed.

**1. Strengths**
- **Pure core, thin shell (ARCH-PURE).** `driveSpan`, `driveMenuRows` and `driveNotices` (`cmd/internal/couchtty/console_drive.go:99-160`) are pure and table-tested without IO. `applyDrive` is thin glue on the loop.
- **Stable menu ids** keyed `driver:<id>` / `ask:<id>` (`console_drive.go:176-192`) and pruned on each delivery. A click acts on the row that was drawn, so a click on a row whose state has since changed does nothing.
- **Notices survive coalescing.** They are computed as a diff from the previous state to the new one, so the controller's latest-only `offer` delivery doesn't lose a grant notice.
- **The fake is consolidated (ARCH-DRY, ARCH-MOCK).** The test-local `drivenBrowser` is promoted to `webcast.FakeDriven` (`cmd/internal/webcast/fake_drive.go`), and the cast test and the Couch test share it. `TestDriveFromCouch` drives the real HTTP routes, the real `MemController` and the real mouse routing (lesson #470 BR-25).
- **Untrusted names are sanitized everywhere they are drawn.** That covers the span (`reserve.go:275`), the menu (`wrapNote`) and notices (`reserve.go:344`), all with a `MaxName` of 40 runes. `TestRenderDriveSpan` pins an escape-sequence injection.

**2. Critical findings**
None.

**3. Important findings**
- **The M3 plan drift is unrecorded (ARCH-PURPOSE / plan traceability).** The table row "`driveSpan` / `DriveMenu` model | `cmd/internal/couchtty/drive_menu.go` | new" and Task 10's "Files: `couchtty/drive_menu.go` (+ test)" don't match the code:
  - There is no `drive_menu.go` and no `DriveMenu` type. The model is `driveMenuRows` / `driveSpan` in `console_drive.go`.
  - The "generalized rows" turned out to be just a title parameter (`RenderTitledMenu`).
  - The console test uses a new exported `webcast.FakeDriven`, which no integration row lists. The plan said "a `Cast` over `FakeBrowser` with its controller".
  
  The Revisions section has no M3 entry. The M2 revision set a rule: "a Task's named files, fixtures and doubles are contract, and any deviation gets a line here in the milestone's closing commit". M3 didn't follow it.
  
  The template classes a table/code contradiction as Critical. I've put it at Important because the entity exists and behaves as described; only its location and name differ.
  
  This is the third finding in this family, so fixing the line is not enough. The rule should be enforced mechanically: a close step that checks every path in the Core concepts table and in every Task's `Files:` against the tree, and refuses the milestone close unless a Revisions line names each mismatch.
- **Every control-op error collapses into one sentence.** `driveResult` (`console_drive.go:261-268`) maps any error from Grant, Deny or Revoke to "Remote driving: that changed meanwhile."
  - That is only true for the race errors: `ErrNoSeat`, and `NextControl` refusing because the state moved.
  - `Revoke` can also fail because `sink.Put` failed: the recorder or `liveGate` has stopped, or the controller returned `ErrClosed`. In that case the driver still drives and the cast-tab guards stay on, but the operator is told it was a race.
  - Fix: switch on the error class. Keep the race sentence for `drive.ErrNoSeat` and the transition-refusal errors. For anything else, say the op failed and include the cause. Better, take the sentence from the same table `drive.Answer` uses, so there is one source of truth (ARCH-DRY).
  - Add a test that goes red without the fix: make a `FakeDriven` whose `Emit` fails after a grant, Ctrl+click the span, and assert the notice isn't the race sentence.

**4. Minor findings**
- In `console_drive_test.go:523-524`, the comment "(nobody drives: nothing to do, no error)" is wrong: Bo drives at that point, and the test asserts the revoke.
- `TestDriveFromCouch` doesn't assert the deny notice ("Said no to Bo.").
- The plan promised revoke "from each state". Ctrl+click on the span with only askers and no driver isn't exercised. It's a no-op by design, but no test pins that.
- `FakeDriven.Emit` now applies to the injector before writing to `out`; the old fake wrote to the stream first. I traced it as unreachable today, because `MemController.Input` checks the driver only after `changeLocked` returns. Even so, ordering stream first would match the recorder's queue semantics.

**5. Test coverage notes**
- The span, rows and notices are fully tabled for every control state.
- The end-to-end test covers grant (left click), deny (right click), revoke from the driver row, Ctrl+left revoke with the menu closed while someone asks (Bo's request survives), Ctrl+right revoke with the menu open, and the span clearing at cast end.
- It is stable at `-race -count=20`.
- Gaps: the error path of `driveResult`, the deny notice, and Ctrl+click with askers only.

**6. Architectural notes for upcoming work**
- **ARCH-DRY:** pass. The menu box and router are reused, and the fake is consolidated. Note that the error-sentence mapping above duplicates what `drive.Answer` already knows.
- **ARCH-PURE:** pass.
- **ARCH-PURPOSE:** pass on delivery. The plan drift is flagged above.
- **ARCH-MOCK:** pass. `FakeDriven` is a stateful fake behind the `Driven` seam that production uses, and M1's live Chrome tests serve as the conformance check.
- **ARCH-CONSTRAINTS:** pass. The span has a 40-rune name cap and the subscription is latest-only, never blocking.
- **ARCH-SECURE:** pass. Names are sanitized at every place they are drawn.
- **ARCH-ORDER:** pass.
  - Couch's control is replaced wholesale from the controller's latest state, a late delivery from an ended cast is dropped, and notices are a pure diff.
  - The subscriber goroutine's lifetime is bounded by the cast (the channel closes on `MemController.Close`) or by `c.stop`.
- **ARCH-FUNERAL:** pass. The `ids` map is pruned per delivery, `forgetDrive` resets at cast end, and the subscription ends with the cast.
- **For M4:** the real `driven` fixture should exercise the Couch-driven sequence (grant, deny, revoke) so that `TestEventLinesAttribute` derives from Couch-originated `ctl` records.

**7. Plan revision recommendations**
- Add "2026-10-12, M3 as built" to Revisions, and correct the Core concepts row to match:
  - The drive span and menu model is `driveSpan` / `driveMenuRows` / `driveNotices` in `couchtty/console_drive.go`. There is no `drive_menu.go` and no `DriveMenu` type.
  - The notes menu is generalized only by a title (`RenderTitledMenu`).
  - Console tests run over `webcast.FakeDriven` (`cmd/internal/webcast/fake_drive.go`, new): the Driven fake, which replaces `cast_drive_test.go`'s `drivenBrowser`. Add it to the Integration points table.

```findings
findings:
  - id: new
    severity: Important
    family: plan-drift-unrecorded
    title: |
      M3 ships without drive_menu.go or a DriveMenu model and adds webcast.FakeDriven, but the plan has no M3 Revisions entry
    detail: |
      The Core concepts row and Task 10 Files name couchtty/drive_menu.go and a DriveMenu model; the code is driveSpan/driveMenuRows in console_drive.go, generalized only by RenderTitledMenu, and tests run over a new exported webcast.FakeDriven that no table lists. This is the 3rd finding in family plan-drift-unrecorded; the M2 rule (any deviation gets a Revisions line in the closing commit) was stated but not enforced. Rule to fix: the milestone close mechanically checks every path in the Core concepts table and each Task's Files list against the tree and refuses the close unless a Revisions line names each mismatch.
  - id: new
    severity: Important
    family: error-names-its-cause
    title: |
      driveResult reports every Grant/Deny/Revoke error as "that changed meanwhile", including a revoke that failed outright
    detail: |
      In console_drive.go:261, a sink failure (recorder or liveGate stopped) or ErrClosed on Revoke leaves the driver driving and the guards on, but the operator is told it was a race. Switch on the error class: keep the race sentence for ErrNoSeat and NextControl refusals, and give a failure sentence with the cause otherwise, ideally from drive.Answer's table. Add a FakeDriven Emit-failure test that goes red without the fix.
  - id: new
    severity: Minor
    family: test-gesture-coverage
    title: |
      TestDriveFromCouch skips the deny notice and Ctrl+click with askers only, and has a wrong comment at line 523
    detail: |
      The plan promised revoke from every state. The askers-only Ctrl+click no-op is untested, "Said no to Bo." is never asserted, and the comment "nobody drives" is wrong because Bo drives at that point.
```

---

## Re-review — 2026-10-12T08:59:17-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 467 — broadcast-web: remote driving — a viewer takes control and uses the app through the operator's tab |
| repo | pair |
| issue file | workshop/issues/000467-broadcast-web-remote-driving-a-viewer-takes-control-and-uses-the-app-through-the-operator-s-tab.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 28b9cf0507978b3dd66a6656a507c2e80851f943..3f4f65b642ddd9c435588a71173b622fd5b89e04 |
| command | sdlc milestone-close --issue 467 --milestone M3 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T08:59:17-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking the remaining Couch-side changes and the tests, then the prior findings.

Now running the tests and checking the BR-23/BR-24 regression evidence.

Now the revert check for BR-23: making `driveFailure` give the race sentence for every error and confirming the test goes red.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M3 does what the plan's Task 10 asks for. Couch now shows who is driving and who is asking after the Chrome cell (🕹/✋). It has a drive menu where the operator can grant, deny or take control back, and Ctrl+click on the span takes control back from any state, whether the menu is open or closed. The late operator decision to split the cast into a view link and a drive link is wired through consistently: the drive routes return 404 under the view link, and a view-link stream does not hold a seat. That holds in the unit test (`TestDriveOnlyUnderTheDriveLink`), the Couch test and the live Chrome test, and it is recorded in the plan's Revisions and in both atlas pages. I checked all three prior findings and they are fixed. For BR-23 I reverted the fix (`driveFailure` reporting a race for every error), and `TestDriveFromCouch` failed at `console_drive_test.go:304`. One thing should be fixed before the boundary: README.md was not updated. The new gesture, span and menu are missing, and README now describes `--broadcast-list` and the Chrome cell's right-click wrongly.

**Strengths**
1. The drive menu is a third source of the existing notes menu, with one box and one gesture router (`console_notes.go:359`, `RenderTitledMenu`), not a parallel menu (ARCH-DRY).
2. The pure core is well split: `driveMenuRows`, `driveSpan`, `driveNotices` and `driveFailure` (`console_drive.go:50-233`) do no IO and are tested directly. The glue code only gets state and runs actions.
3. Ordering is safe. `applyDrive` drops deliveries from a cast that has ended, and it runs on the same terminal loop as `webcastEnded`, so a late delivery can't bring back the span after the cast ends. Menu ids are stable per row key and pruned on every delivery, so they don't build up.
4. `webcast.FakeDriven` (`fake_drive.go`) is a real stateful fake that Couch's tests drive through the cast's real HTTP routes. `FailEmits` lets a test reach the failure branch, and the live Chrome test checks that the view link offers no drive button.
5. Neither link's token leaks: both copy notices leave it out, `ShareStatus` has no field for it, and the viewer-chosen driver name is sanitized in the tab bar and in `--broadcast-list`.

**Critical findings**
None.

**Important findings**
- **README.md is not updated for M3's user-facing surface** (ARCH-PURPOSE / docs gate). Missing from README:
  - the 🕹/✋ span and its menu;
  - Ctrl+click on the span to take control back;
  - Alt+right-click on the Chrome cell to copy the drive link;
  - the new output of `couch --broadcast-list`, which now prints one line per link (`screen`, `web view`, `web drive`) and a `shares` key in JSON.

  README.md:400 and :864 still say the command lists "the running broadcast and its viewer count". README.md:956 still says right-click copies "the viewer link", with no mention that there are now two links. This is the **2nd finding in family `atlas-tracks-surface`**. The rule: a milestone commit that adds a gesture, flag, notice or output format updates README and atlas in the same commit. Task 12 lists only atlas, so the plan needs README added there or in M3's close.

**Minor findings**
- ARCH-DRY / test coverage: `driveFailure` (`console_drive.go:223`) repeats the error classification that `drive.Answer` already does. Only its default branch is tested; the race, cast-ended and not-live sentences have no table test, which would be cheap for a pure function.
- ARCH-SECURE / error-names-its-cause: if `broadcast.NewToken` fails for the drive token (`cast.go:236`), driving is turned off with no log. Couch then says "This cast can't be driven (it isn't recorded)", which names the wrong cause.
- `Server.Viewers()` (`server.go:631`) reads two counters separately, so it can briefly report a view count off by one (even −1) while a drive viewer is joining or leaving. It only affects the display.
- `copyWebcastDriveLink` is nearly a copy of `copyWebcastLink` (`console_webcast.go:270`).

**Test coverage notes**
- Tests run and passed: `go test` for `couchtty`, `webcast/...` and `couchmessage`. `TestDrive*` passed 10/10 under `-race`. The couchcmd broadcast and shares tests passed.
- The full `couchcmd` package failed in `TestSwitchAgentProducesOrientationThroughLayoutAndWrapper` (launcher layout signal killed). That test is in the launcher acceptance path, which this diff doesn't touch, so I think it comes from the sandbox.
- Before the couchtty and couchcmd tests would build, I had to generate the runtime bundle in the disposable checkout.
- The cases BR-24 named are now covered: "Said no to Bo." is asserted, Ctrl+click with only askers is a no-op, and the comment is corrected.

**Architectural notes for upcoming work**
- ARCH-PURE pass; ARCH-MOCK pass (fake plus live Chrome check); ARCH-CONSTRAINTS pass (`Shares` is cheap and on demand); ARCH-ORDER pass (loop-serialized, stale casts dropped); ARCH-FUNERAL pass (id map pruned, state cleared at cast end, subscriber goroutine ends with the channel or `c.stop`). ARCH-SECURE and ARCH-DRY pass, apart from the minor findings above.
- For M4: if the recorded `driven` fixture is captured through the drive link, record which token's URL the fixture uses, so the replay never embeds a live token.

**Plan revision recommendations**
- Add README to Task 12's Files and steps (or to M3's close): the drive span and menu, the Ctrl+click take-back, Alt+right-click for the drive link, the view/drive split on the Chrome cell, and the `--broadcast-list` share lines and JSON `shares`.

```findings
dispose:
  - id: BR-22
    disposition: addressed
    note: |
      Plan Revisions 2026-10-12 "M3 as built" names the missing drive_menu.go/DriveMenu (driveMenu source + RenderTitledMenu) and the exported webcast.FakeDriven; the two-links decision's new surface is also recorded.
  - id: BR-23
    disposition: addressed
    note: |
      driveFailure classes errors (console_drive.go:223); reverting it to the race sentence turns TestDriveFromCouch red at console_drive_test.go:304 (FailEmits revoke).
  - id: BR-24
    disposition: addressed
    note: |
      console_drive_test.go:244 asserts "Said no to Bo.", :274-282 covers askers-only Ctrl+click as a no-op, and :253's comment now follows control coming back.
findings:
  - id: new
    severity: Important
    family: atlas-tracks-surface
    title: |
      README not updated for M3 surface: drive span/menu, Ctrl+click take-back, Alt+right-click drive link, --broadcast-list shares
    detail: |
      2nd finding in this family. README.md:400/:864 still describe --broadcast-list as broadcast plus viewer count, :956 says right-click copies "the viewer link", and nothing covers the drive span or menu. Rule: a milestone commit that adds a gesture, flag, notice or output format updates README and atlas in the same commit; add README to Task 12 / the M3 close.
  - id: new
    severity: Minor
    family: shared-helper-reuse
    title: |
      driveFailure repeats drive.Answer's error classification; only its default branch is tested
    detail: |
      2nd finding in this family. Rule: one classification table per error set (drive.Answer's), with per-audience sentences derived from it; driveFailure's race, cast-ended and not-live sentences need a pure table test.
  - id: new
    severity: Minor
    family: error-names-its-cause
    title: |
      Failure to mint the drive token silently turns driving off, and Couch then says the cast isn't recorded
    detail: |
      cast.go:236 drops the controller with no log; copyWebcastDriveLink then names the wrong cause. 2nd finding in this family; rule: every path that turns a feature off records the cause it gives the operator.
  - id: new
    severity: Minor
    family: state-scope-unmodelled
    title: |
      Server.Viewers reads two counters separately; the view count can briefly be off by one, even negative
    detail: |
      server.go:631 subtracts driveViewers from viewers read at a different moment; display only. 3rd finding in this family; rule: a count derived from two counters is read in one snapshot (one counter per link, or a mutex).
```

---

## Re-review — 2026-10-12T09:08:09-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 467 — broadcast-web: remote driving — a viewer takes control and uses the app through the operator's tab |
| repo | pair |
| issue file | workshop/issues/000467-broadcast-web-remote-driving-a-viewer-takes-control-and-uses-the-app-through-the-operator-s-tab.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 28b9cf0507978b3dd66a6656a507c2e80851f943..2191cad0ae42fa2a44f951a7fcc93c1b32d6a002 |
| command | sdlc milestone-close --issue 467 --milestone M3 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T09:08:09-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Prior findings checked; running the affected packages' tests now.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M3 is ready to ship after one small fix. Three of the four open findings are now fixed:
- **BR-25:** the README covers the drive link, the drive span and menu, Ctrl+click take-back, Alt+right-click, and the shares in `--broadcast-list`.
- **BR-27:** `Cast.DriveOff` records why driving is off, the token failure is logged, and the notice names the actual cause.
- **BR-28:** there is now one counter per link, so neither count is worked out from two reads.

**BR-26 is only half fixed.** `driveFailure` has its table test now, but it still repeats `drive.Answer`'s error classification. That stays Minor and doesn't block.

**The round-2 fix added one new problem.** `copyWebcastDriveLink` reads `c.wcast.cast` twice, under two separate lock acquisitions. If the cast ends between the two reads, the second read gets nil and `DriveOff()` dereferences it. The fix is one line: read the cast in the same locked snapshot as `u`.

I ran these from the review window: stat, name-status, the targeted diffs, and the round-2 commit. `go test` passes for `webcast`, `webcast/drive`, `couchmessage`, `couchtty` and `couchcmd`. `couchtty` and `couchcmd` only build after the runtime bundle is generated, so I generated it first. The Drive, Share and Webcast tests in `couchtty` also pass under `-race`.

1. **Strengths**
   - The pure core is well separated: `driveMenuRows`, `driveSpan`, `driveNotices`, `driveFailure` and `formatShares` are all pure and table-tested (`console_drive.go:49-118`, `broadcast_list.go:59`).
   - `applyDrive` throws away deliveries from a cast that has since ended. Menu IDs stay stable per row key and are pruned when a row disappears, so a click acts on the row that was drawn (`console_drive.go:122-160`).
   - The menu reuses the notes box through `RenderTitledMenu` instead of a second copy of the menu code (`notes_menu.go:84-97`). This is a good example of ARCH-DRY.
   - Viewer names are cleaned up where they enter (`controller_mem.go:75`, `label`), and they are sanitized again in the CLI output (`rowtext.Sanitize`). Shares never carry a link or a token (`protocol.go` `ShareStatus`).
   - `TestDriveOnlyUnderTheDriveLink` now checks the per-link counts end to end.

2. **Critical findings:** none.

3. **Important findings**
   - **The drive-link notice can crash Couch** (`cmd/internal/couchtty/console_webcast.go:284`). `c.wcastCast() != nil && c.wcastCast().DriveOff() != ""` takes `c.mu` twice. If `webcastEnded` resets `c.wcast` between the two calls, the second returns nil and `(*Cast).DriveOff` panics on the nil pointer. The first read at `:275` already snapshotted `u` and `running`.
     - **Fix:** read `cast := c.wcast.cast` in that same locked block, then use `case u.Link != "" && cast != nil && cast.DriveOff() != "":`. Then delete `wcastCast`.
     - **Rule (the same one as BR-28):** a decision that depends on several fields of shared state reads them all in one snapshot.

4. **Minor findings**
   - BR-26 is still open: `driveFailure` and `drive.Answer` each classify the same error set on their own. A `drive.Class(err)` used by both would keep the two from drifting apart.
   - The comment at `console_notes.go:359-363` says right-click "takes control back from the driver too". That is accurate, but the README only says "Click the driver's row".
   - The branch where `NewToken` fails (`cast.go:161`) is untested. In practice it can't be reached, because the token comes from crypto/rand.

5. **Test coverage notes**
   - BR-27's regression evidence is `TestCastWithoutDriving`, which checks `DriveOff()` and would not compile without the fix.
   - BR-28 can't be shown to go red with a deterministic test, since it was a race. The structural change removes the derivation, so I accept it as fixed.
   - `TestDriveFailure` now pins every class.
   - No test exercises `copyWebcastDriveLink`'s DriveOff notice. A test with an unrecorded cast that asserts "This cast can't be driven: the cast isn't recorded." would cover it.

6. **Architecture**
   - **ARCH-DRY:** flag, BR-26 only (Minor).
   - **ARCH-PURE:** pass.
   - **ARCH-PURPOSE:** pass. M3 delivers the menu, both links and the shares.
   - **ARCH-MOCK:** pass. `FakeDriven` plus the live viewer test.
   - **ARCH-CONSTRAINTS:** pass. Every drive-menu action runs only on an operator click, nothing is unbounded, and the counters are atomic.
   - **ARCH-SECURE:** pass. Names are cleaned at `Join`, and no token appears in shares or notices.
   - **ARCH-ORDER:** flag. This is the new finding above: the cast can end between two reads.
   - **ARCH-FUNERAL:** pass. `forgetDrive` clears the drive state when the cast ends, and menu IDs are pruned.

7. **Plan revision recommendations:** none. The plan's M3 Revisions entry (from BR-22) still matches the code.

```findings
dispose:
  - id: BR-25
    disposition: addressed
    note: |
      README.md:400, :864-869 and :958-980 now cover the shares by kind, view vs drive link (Alt+right-click), the drive span and menu (grant, deny, take back) and Ctrl+click on the span. Checked against terminal_input.go:149-170 and console_drive.go.
  - id: BR-26
    disposition: not-addressed
    note: |
      TestDriveFailure now pins every class, but driveFailure (console_drive.go:226) still repeats drive.Answer's errors.Is table instead of deriving from one classification. Minor.
  - id: BR-27
    disposition: addressed
    note: |
      cast.go:159-171 sets driveOff and logs it; copyWebcastDriveLink names DriveOff(); TestCastWithoutDriving pins the cause (fails to compile without the fix).
  - id: BR-28
    disposition: addressed
    note: |
      server.go:67-69/:152-157 keep one counter per link and Viewers reads each directly, so no count is derived from two reads; the race isn't testable deterministically, and TestDriveOnlyUnderTheDriveLink checks the counts.
findings:
  - id: new
    severity: Important
    family: read-in-one-snapshot
    title: |
      copyWebcastDriveLink reads c.wcast.cast twice under separate locks; a cast ending in between panics on a nil DriveOff
    detail: |
      console_webcast.go:284 calls c.wcastCast() twice; if webcastEnded resets c.wcast between the two calls, (*Cast)(nil).DriveOff() dereferences nil. Read cast in the same locked snapshot as u at :275 and delete wcastCast. Same rule as BR-28: a decision over several fields of shared state reads them in one snapshot.
```

---

## Re-review — 2026-10-12T09:24:36-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 467 — broadcast-web: remote driving — a viewer takes control and uses the app through the operator's tab |
| repo | pair |
| issue file | workshop/issues/000467-broadcast-web-remote-driving-a-viewer-takes-control-and-uses-the-app-through-the-operator-s-tab.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 28b9cf0507978b3dd66a6656a507c2e80851f943..0204724cbeb2faaf5d4ef387e3a6fd1f2cc6cd8d |
| command | sdlc milestone-close --issue 467 --milestone M3 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T09:24:36-07:00 |
| verdict | SHIP |

## Review

Focus on the open findings first.

Disposing the two open findings (both look fixed in code); now running the package tests and skimming the rest of the M3 diff.

Generated assets are a build step; generating them in this disposable checkout to run couchtty/couchcmd tests.

couchtty passes under -race; the couchcmd race is in `replay_test.go`, which this window doesn't touch (pre-existing). Now skimming the M3 code.

Viewer names are sanitized when they join and again at the tab bar. Checking the plan's M3 checklist and Revisions next.

```verdict
verdict: SHIP
confidence: high
```

M3 is ready to ship. Both open findings from earlier rounds are fixed. Couch's drive span, its menu (grant, deny, take back) and Ctrl+click take-back are delivered as Task 10 describes. The plan's Revisions record the changes made along the way: no `drive_menu.go`, the drive menu reuses the notes menu, the new `FakeDriven` fake, and the operator's split into a view link and a drive link. README and atlas cover the new surface.

On checks: `go vet` is clean, and the `webcast/...` and `couchmessage` tests pass under `-race`. The `couchtty` and `couchcmd` tests need their embedded runtime assets generated first. I generated them in the disposable checkout (`runtimebundle-generate`), after which `couchtty` passed under `-race`. `couchcmd` still failed on one test: a data race in `TestRunReplayServesTheLink`. That test lives in `replay_test.go:104/122`, which this window never touches, so the race predates M3 and doesn't affect the verdict. Nothing I found blocks the gate.

**1. Strengths**
- **The two links are enforced on the server.** `Server.drivesUnder` (`server.go`) means the drive routes answer 404 under the view link, and a stream can hold a seat (`Attach`) only on the drive link. A viewer on the view link can't get a seat by sending a forged `?viewer=` id.
- **Viewer names are cleaned twice.** They are cleaned when a viewer joins (`drive/controller_mem.go` `label`: control, format and separator characters removed, length capped). The tab bar (`reserve.go:275`) and the CLI listing (`broadcast_list.go`, `rowtext.Sanitize`) clean them again before showing them.
- **No stale drive state after a cast ends.** `applyDrive` and `webcastEnded` both run through `runTerminalCommand`, so they never interleave. A delivery from a cast that has already ended is dropped by the check `c.wcast.cast == cast`.
- **Menu ids can't pile up.** Ids stay stable while a row shows, and an id is deleted once its row is gone (`console_drive.go` `applyDrive`).
- **No new token reaches any output.** Neither `ShareStatus` nor the notices carry a token, and a failure to make the drive link is reported to the operator through `DriveOff`.

**2. Critical findings:** none.

**3. Important findings:** none.

**4. Minor findings** (not raised as new findings)
- `--broadcast-list --json` prints `"shares": null` when nothing is shared. Adding `omitempty` or using an empty slice would give consumers a steadier shape.
- `copyWebcastLink` and `copyWebcastDriveLink` share most of their structure. It's tolerable at two copies.

**5. Test coverage notes**
- `TestDriveFailure` checks the exact sentence for every class of control failure.
- `TestDriveFromCouch` goes through the cast's real HTTP routes over `FakeDriven`, including a revoke that fails.
- The fix for BR-29 has no deterministic test. That's acceptable: the second read was removed outright (`wcastCast` is deleted), so the race can no longer be written.
- `couchtty` takes about 5.5 minutes for one run under `-race`, which makes the plan's `-count=20` expensive.

**6. Architecture**
- **ARCH-DRY: pass.** The drive menu reuses the notes menu's box and gesture handling. `driveFailure` is a separate sentence table for the operator; see the BR-26 note below.
- **ARCH-PURE: pass.** `driveMenuRows`, `driveSpan`, `driveNotices` and `driveFailure` are pure and tested directly.
- **ARCH-PURPOSE: pass.** Every M3 item in Task 10 is delivered.
- **ARCH-MOCK: pass.** `FakeDriven` is a fake that keeps state across calls and sits behind `Driven`, and the live test from M1/M2 still exists.
- **ARCH-CONSTRAINTS: pass.** The span updates when control changes, not on a timer, and the per-link viewer counts are atomic.
- **ARCH-SECURE: pass.** See the two-links and name-cleaning strengths above.
- **ARCH-ORDER: pass.** Control's state machine stays in `MemController`. Couch only mirrors it, serialized on its event loop, and ignores deliveries from a cast that has ended.
- **ARCH-FUNERAL: pass.** The drive token, the drive state and the menu ids all end with the cast (`forgetDrive`), and menu ids are pruned on every update.
- **For M4:** if the operator's failure sentences and the viewer's sentences in `drive.Answer` drift apart, a shared `drive.Class(err)` that both map from would remove the second classification.

**7. Plan revision recommendations:** none. The Revisions entry for M3 and the two links matches the code.

```findings
dispose:
  - id: BR-26
    disposition: addressed
    note: |
      TestDriveFailure (console_drive_test.go:320-336) pins every branch exactly; the operator-side switch remains a distinct per-audience sentence table, acceptable for a Minor.
  - id: BR-29
    disposition: addressed
    note: |
      console_webcast.go:275-276 reads update, phase and cast in one locked snapshot; wcastCast deleted, so no second read remains.
```
