---
gate: boundary-review
issue: 467
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T07:00:57-07:00"
      dispatched: "2026-10-12T06:56:35-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: 'The dialog signal isn''t tied to the cast tab: an open while paused is missed, and any tab''s close clears it'
          detail: 'chrome.go only counts dialogOpened when castTab is set (empty while paused), accepts dialogClosed from any session, and never re-checks on a cast-tab change or destroy. Confirmed with temporary loop tests. The driver''s calls then hang until the 5 s timeout (500, not the fixed sentence) while the recorder loop stalls. Fix: a per-session set of open dialogs in loopState; the signal is "cast tab''s session is in the set", recomputed on every change.'
          family: state-scope-unmodelled
          round: 1
        - id: BR-2
          severity: Important
          title: TestLiveDrive doesn't check the file chooser, the download denial or the restore after SetDriving("")
          detail: live_drive_test.go only checks the file and download clicks return no error. Task 5 and Done-when ask for fileChooserOpened seen with the intercept on, no download, and the intercept off after SetDriving(""). The fake-conn loop test only checks commands are sent, not real Chrome's behavior.
          family: live-conformance-unasserted
          round: 1
        - id: BR-3
          severity: Minor
          title: Per-seat rate buckets are reset wholesale at 256 instead of pruned when a seat leaves, as planned
          detail: server_drive.go limits() clears the whole map at maxSeatLimits; a departed seat's bucket stays until its key's next request. A reset also restores well-behaved seats' budgets. Prune on the controller's leaves, or record the change in Revisions.
          family: residue-bound-on-writer
          round: 1
        - id: BR-4
          severity: Minor
          title: Route budgets (no maxViewers x2 hello cap) and the all-pages file-chooser intercept differ from the plan text
          detail: Bounded equivalently by the hello rate times the grace period; record both in the plan's Revisions.
          family: plan-drift-unrecorded
          round: 1
        - id: BR-5
          severity: Minor
          title: A page the cast tab opens while paused-but-driving isn't closed (castTab is empty while paused)
          family: guard-scope-by-phase
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T07:08:49-07:00"
      dispatched: "2026-10-12T07:05:28-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Per-session dialogs set, recomputed in publish (chrome.go:545); TestChromeCastTabDialog fails with the old logic restored ("dialog signal never became true").
          round: 2
        - id: BR-2
          disposition: addressed
          note: live_drive_test.go now asserts Page.fileChooserOpened via tap, download denial against a positive control, and no chooser after SetDriving(""); not executed here (Chrome GPU crash on this host).
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: Sweep and Revisions entry landed, but TestDriveSeatTableBounded passes against the old reset (flood keys are dropped on ErrNoSeat, so the table never fills); limits() uses time.Now() inline, so the idle sweep can't be tested — inject now and pin that a full table sweeps idle keys and doesn't refill a seat's budget.
          round: 2
        - id: BR-4
          disposition: addressed
          note: Plan Revisions 2026-10-12 records the route budgets without the maxViewers x2 cap and the all-pages chooser intercept.
          round: 2
        - id: BR-5
          disposition: addressed
          note: castTab drops the PhaseLive gate (chrome_drive.go:90); restoring the gate makes TestChromeCastTabDialog fail ("never got Target.closeTarget T4").
          round: 2
      findings:
        - id: BR-6
          severity: Minor
          title: loopState.tap is a test-only event hook carried in the production loop struct
          detail: chrome.go:140; nil in production and only set on the loop goroutine, so harmless today. Prefer one injected observer seam if more test hooks accumulate.
          family: test-hook-in-production-state
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: 34a9eff353447e2edf56dde561955a1736ed346c
      blocked: false
    - "n": 3
      timestamp: "2026-10-12T07:25:56-07:00"
      dispatched: "2026-10-12T07:21:45-07:00"
      agent: claude
      findings:
        - id: BR-7
          severity: Important
          title: batcher caps acts and text runes but not body bytes; a paste over about 8 KB is refused (413) whole
          detail: 'drive.MaxBody is 8 KB. A probe of drive.js batcher: 9000 ASCII chars gave a 9198-byte batch and 3000 CJK chars 9078 bytes; admitBody answers 413 and the queue is dropped. Bound go() by serialized UTF-8 size (and split text by bytes), with a test that every sent body is at most MaxBody.'
          family: client-honors-wire-contract
          round: 3
        - id: BR-8
          severity: Important
          title: a captured drag leaving the frame left or top emits negative page coords; ParseBatch 400s and the up is never delivered
          detail: 'toPage does not clamp and the pointer is captured; coord() requires >= 0. Probe: down(50,50), move(-50,50), up(-60,50). The cast tab keeps a pressed button. Clamp in gestures'' pt/at and add a trace test that leaves the frame.'
          family: client-honors-wire-contract
          round: 3
        - id: BR-9
          severity: Important
          title: Task 9 live test uses a stateless spyInjector, not MemController over FakeInjector, and the plan has no Revisions entry
          detail: '2nd in family. Rule: a Task''s named files, fixtures and doubles are contract; any deviation gets a Revisions line in the milestone''s closing commit. Sweep M2: the spy, the "Cancel request" label, 1024x700 instead of tablet width, Rejoin by reload. The spy skips the seg, Confine and stale rules, so a wrong seg from the shell would pass.'
          family: plan-drift-unrecorded
          round: 3
        - id: BR-10
          severity: Minor
          title: offerRejoin's button is overwritten by the next render(), so the 401 Rejoin offer disappears
          family: shell-state-overwritten
          round: 3
        - id: BR-11
          severity: Minor
          title: drive.js copies annotate's toast (same .annotate-msg class, can overlap) and collab.poster's request options
          family: shared-helper-reuse
          round: 3
        - id: BR-12
          severity: Minor
          title: chord keyup is dropped when the modifier is released first; hover moves are sent while a button is held after a press on nothing
          family: input-pairing
          round: 3
        - id: BR-13
          severity: Minor
          title: batcher does not pace to the server's 30/s input budget; fast key auto-repeat can 429 and drop the queue
          family: client-honors-wire-contract
          round: 3
        - id: BR-14
          severity: Minor
          title: drive.css is served but no test fetches it; the import-walk test covers only JS modules
          family: served-asset-coverage
          round: 3
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 4
      timestamp: "2026-10-12T07:41:33-07:00"
      dispatched: "2026-10-12T07:34:55-07:00"
      agent: claude
      dispose:
        - id: BR-7
          disposition: addressed
          note: take() bounds by serialized UTF-8 size and splitText cuts by bytes; reverting the MAX_BODY break turns the property test at drive.test.mjs:103 red.
          round: 4
        - id: BR-8
          disposition: addressed
          note: wireAct clamps x/y into [0, MAX_COORD]; removing the clamp fails 3 tests, including the leaves-the-frame trace at drive.test.mjs:272.
          round: 4
        - id: BR-9
          disposition: addressed
          note: 'Live test runs MemController over FakeInjector (spy only records) and asserts an attributed Click in the stream; Revisions entry covers the spy, labels, viewport and Rejoin. Could not execute here: Chrome CDP EOF also fails the pre-existing TestLiveViewerPage.'
          round: 4
        - id: BR-10
          disposition: not-addressed
          note: 'The lost flag and early return in render() look correct, but no test fails without it: mountDrive has no harness. Fold lost into driveView (pure) and table-test it.'
          round: 4
        - id: BR-11
          disposition: addressed
          note: collab.js sayer (one shared .annotate-msg element) and poster(headers) are used by both drive.js and annotate.js; the copies are gone.
          round: 4
        - id: BR-12
          disposition: addressed
          note: keyTracker test covers the up after the modifier is released; reverting the !e.buttons hover guard fails the gestures test.
          round: 4
        - id: BR-13
          disposition: addressed
          note: PACE_MS=40 is enforced in go(); forcing wait=0 fails the pacing assertion in the property test.
          round: 4
        - id: BR-14
          disposition: addressed
          note: TestViewerImportsAreServed now walks the page's stylesheet links and requires drive.css to be served as text/css.
          round: 4
      findings:
        - id: BR-15
          severity: Important
          title: batcher stamps seg at send time, so acts queued before a new seg record are injected into a page the driver never saw
          detail: 'This is the 4th finding in family client-honors-wire-contract. Rule covering all of them: every field of a request is fixed at the moment of the user act it describes, and the gate only gathers and bounds, never re-derives. drive.js:107 reads deps.seg() inside take(), but the server''s seg check (injector.go:35) is meant to refuse acts aimed at a segment that has gone. Text and key acts have no id, so the stale check cannot catch them. A resume after a pause does the same, because the batch survives the pause and the keys textarea still queues input. Fix: stamp seg in push(), have take() end a batch where the stamp changes, and add property-test cases where the seg value changes between push and send.'
          family: client-honors-wire-contract
          round: 4
        - id: BR-16
          severity: Important
          title: a press or key held when control ends, or when a refusal drops the queue, is never released in the cast tab
          detail: 'This is the 2nd finding in family input-pairing. Rule: every down the driver starts is ended on every exit path (refusal, revoke, take, seg change, seat loss), and that duty belongs to the side that outlives the exit, the server. batcher.stop() and the refusal path drop queued up and key-up acts. gestures'' press is never reset, so the next drive sends left-button moves with no down. Nothing server-side releases held buttons or keys on a control change. Fix: on a control change, have TabInjector/WithTab dispatch mouseReleased and key-up for whatever the previous driver left down; add a reset() to gestures and keyTracker and call it from render() when the batch stops.'
          family: input-pairing
          round: 4
        - id: BR-17
          severity: Minor
          title: shell states (lost, driving-but-paused) live as flags in render() rather than in driveView
          detail: This is the 2nd finding in family state-scope-unmodelled. driveView should return the whole state set (none, asking, driving-live, driving-paused, other, lost) so render() only paints it, and every state can be table-tested without a DOM.
          family: state-scope-unmodelled
          round: 4
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 5
      timestamp: "2026-10-12T07:50:27-07:00"
      dispatched: "2026-10-12T07:48:08-07:00"
      agent: claude
      dispose:
        - id: BR-10
          disposition: addressed
          note: lost is now a driveView state (action rejoin) painted by every render(); table-tested in drive.test.mjs.
          round: 5
        - id: BR-15
          disposition: addressed
          note: seg stamped in push(), take() cuts at seg change, no coalescing across segs; batcher test goes red with the cut reverted.
          round: 5
        - id: BR-16
          disposition: addressed
          note: TabInjector tracks held button/keys, releases on other driver, other seg, refusal and control change (Releaser); gestures.reset and tracked.clear on end/refusal; TestControllerLetsGoOnRevoke goes red with Release disabled.
          round: 5
        - id: BR-17
          disposition: addressed
          note: driveView returns none/asking/driving/paused/other/lost with action and capture; render only paints; table test covers all rows.
          round: 5
      findings:
        - id: BR-18
          severity: Important
          title: atlas/webcast.md does not describe the cast tab's let-go (Releaser on control change, refusal, driver or page change) or per-act segments
          detail: Remote driving section lines 197-215 still says only that the batcher drops its queue at a refusal; add a Let go bullet and the one-segment-per-batch rule.
          family: atlas-tracks-surface
          round: 5
        - id: BR-19
          severity: Minor
          title: Release ends whatever is held when it reaches the tab, not the outgoing driver's hold
          detail: '3rd in family input-pairing. Rule: every release names the hold it ends (driver, seg) and is a no-op otherwise. The async release from changeLocked (controller_mem.go:162) could otherwise drop the next driver''s press if that driver''s batch runs first; pass by into Release and test revoke, grant, press, then late Release.'
          family: input-pairing
          round: 5
        - id: BR-20
          severity: Minor
          title: duplicate return nil at drive/tab.go:164; go vet reports unreachable code
          family: dead-code
          round: 5
        - id: BR-21
          severity: Minor
          title: held.keys appends every auto-repeat keydown, so a release sends N keyups for one key
          detail: '2nd in family. Rule: a server-held collection fed per input act is keyed by identity at its writer. Fix by deduping by key in TabInjector.note (tab.go:107) or filtering e.repeat in keyTracker.down.'
          family: residue-bound-on-writer
          round: 5
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 6
      timestamp: "2026-10-12T07:54:25-07:00"
      dispatched: "2026-10-12T07:52:35-07:00"
      agent: claude
      dispose:
        - id: BR-18
          disposition: addressed
          note: atlas/webcast.md:211-228 adds the Batches rules (one segment per batch) and a Letting go section listing four cases with Releaser naming the outgoing driver; webcast-format.md:601-603 matches tab.go.
          round: 6
        - id: BR-19
          disposition: addressed
          note: Release(ctx, by) is a no-op unless held.by==by (tab.go:75-85), and controller_mem.go:165 passes the outgoing driver. With the check reverted, release_test.go:112 fails.
          round: 6
        - id: BR-20
          disposition: addressed
          note: The duplicate return is gone (tab.go:176), and go vet ./cmd/internal/webcast/... is clean.
          round: 6
        - id: BR-21
          disposition: addressed
          note: note() dedups held keys by Key at the writer (tab.go:122). With the dedup reverted, release_test.go:116 fails with two keyups.
          round: 6
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      reviewed: 515639975d462d080c71d73aed630188d099ff34
      blocked: false
    - "n": 7
      timestamp: "2026-10-12T08:22:10-07:00"
      dispatched: "2026-10-12T08:18:40-07:00"
      agent: claude
      findings:
        - id: BR-22
          severity: Important
          title: M3 ships without drive_menu.go or a DriveMenu model and adds webcast.FakeDriven, but the plan has no M3 Revisions entry
          detail: 'The Core concepts row and Task 10 Files name couchtty/drive_menu.go and a DriveMenu model; the code is driveSpan/driveMenuRows in console_drive.go, generalized only by RenderTitledMenu, and tests run over a new exported webcast.FakeDriven that no table lists. This is the 3rd finding in family plan-drift-unrecorded; the M2 rule (any deviation gets a Revisions line in the closing commit) was stated but not enforced. Rule to fix: the milestone close mechanically checks every path in the Core concepts table and each Task''s Files list against the tree and refuses the close unless a Revisions line names each mismatch.'
          family: plan-drift-unrecorded
          round: 7
        - id: BR-23
          severity: Important
          title: driveResult reports every Grant/Deny/Revoke error as "that changed meanwhile", including a revoke that failed outright
          detail: 'In console_drive.go:261, a sink failure (recorder or liveGate stopped) or ErrClosed on Revoke leaves the driver driving and the guards on, but the operator is told it was a race. Switch on the error class: keep the race sentence for ErrNoSeat and NextControl refusals, and give a failure sentence with the cause otherwise, ideally from drive.Answer''s table. Add a FakeDriven Emit-failure test that goes red without the fix.'
          family: error-names-its-cause
          round: 7
        - id: BR-24
          severity: Minor
          title: TestDriveFromCouch skips the deny notice and Ctrl+click with askers only, and has a wrong comment at line 523
          detail: The plan promised revoke from every state. The askers-only Ctrl+click no-op is untested, "Said no to Bo." is never asserted, and the comment "nobody drives" is wrong because Bo drives at that point.
          family: test-gesture-coverage
          round: 7
      boundary: M3
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 8
      timestamp: "2026-10-12T08:59:18-07:00"
      dispatched: "2026-10-12T08:49:26-07:00"
      agent: claude
      dispose:
        - id: BR-22
          disposition: addressed
          note: Plan Revisions 2026-10-12 "M3 as built" names the missing drive_menu.go/DriveMenu (driveMenu source + RenderTitledMenu) and the exported webcast.FakeDriven; the two-links decision's new surface is also recorded.
          round: 8
        - id: BR-23
          disposition: addressed
          note: driveFailure classes errors (console_drive.go:223); reverting it to the race sentence turns TestDriveFromCouch red at console_drive_test.go:304 (FailEmits revoke).
          round: 8
        - id: BR-24
          disposition: addressed
          note: console_drive_test.go:244 asserts "Said no to Bo.", :274-282 covers askers-only Ctrl+click as a no-op, and :253's comment now follows control coming back.
          round: 8
      findings:
        - id: BR-25
          severity: Important
          title: 'README not updated for M3 surface: drive span/menu, Ctrl+click take-back, Alt+right-click drive link, --broadcast-list shares'
          detail: '2nd finding in this family. README.md:400/:864 still describe --broadcast-list as broadcast plus viewer count, :956 says right-click copies "the viewer link", and nothing covers the drive span or menu. Rule: a milestone commit that adds a gesture, flag, notice or output format updates README and atlas in the same commit; add README to Task 12 / the M3 close.'
          family: atlas-tracks-surface
          round: 8
        - id: BR-26
          severity: Minor
          title: driveFailure repeats drive.Answer's error classification; only its default branch is tested
          detail: '2nd finding in this family. Rule: one classification table per error set (drive.Answer''s), with per-audience sentences derived from it; driveFailure''s race, cast-ended and not-live sentences need a pure table test.'
          family: shared-helper-reuse
          round: 8
        - id: BR-27
          severity: Minor
          title: Failure to mint the drive token silently turns driving off, and Couch then says the cast isn't recorded
          detail: 'cast.go:236 drops the controller with no log; copyWebcastDriveLink then names the wrong cause. 2nd finding in this family; rule: every path that turns a feature off records the cause it gives the operator.'
          family: error-names-its-cause
          round: 8
        - id: BR-28
          severity: Minor
          title: Server.Viewers reads two counters separately; the view count can briefly be off by one, even negative
          detail: 'server.go:631 subtracts driveViewers from viewers read at a different moment; display only. 3rd finding in this family; rule: a count derived from two counters is read in one snapshot (one counter per link, or a mutex).'
          family: state-scope-unmodelled
          round: 8
      boundary: M3
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 9
      timestamp: "2026-10-12T09:08:10-07:00"
      dispatched: "2026-10-12T09:02:10-07:00"
      agent: claude
      dispose:
        - id: BR-25
          disposition: addressed
          note: README.md:400, :864-869 and :958-980 now cover the shares by kind, view vs drive link (Alt+right-click), the drive span and menu (grant, deny, take back) and Ctrl+click on the span. Checked against terminal_input.go:149-170 and console_drive.go.
          round: 9
        - id: BR-26
          disposition: not-addressed
          note: TestDriveFailure now pins every class, but driveFailure (console_drive.go:226) still repeats drive.Answer's errors.Is table instead of deriving from one classification. Minor.
          round: 9
        - id: BR-27
          disposition: addressed
          note: cast.go:159-171 sets driveOff and logs it; copyWebcastDriveLink names DriveOff(); TestCastWithoutDriving pins the cause (fails to compile without the fix).
          round: 9
        - id: BR-28
          disposition: addressed
          note: server.go:67-69/:152-157 keep one counter per link and Viewers reads each directly, so no count is derived from two reads; the race isn't testable deterministically, and TestDriveOnlyUnderTheDriveLink checks the counts.
          round: 9
      findings:
        - id: BR-29
          severity: Important
          title: copyWebcastDriveLink reads c.wcast.cast twice under separate locks; a cast ending in between panics on a nil DriveOff
          detail: 'console_webcast.go:284 calls c.wcastCast() twice; if webcastEnded resets c.wcast between the two calls, (*Cast)(nil).DriveOff() dereferences nil. Read cast in the same locked snapshot as u at :275 and delete wcastCast. Same rule as BR-28: a decision over several fields of shared state reads them in one snapshot.'
          family: read-in-one-snapshot
          round: 9
      boundary: M3
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 10
      timestamp: "2026-10-12T09:24:36-07:00"
      dispatched: "2026-10-12T09:10:15-07:00"
      agent: claude
      dispose:
        - id: BR-26
          disposition: addressed
          note: TestDriveFailure (console_drive_test.go:320-336) pins every branch exactly; the operator-side switch remains a distinct per-audience sentence table, acceptable for a Minor.
          round: 10
        - id: BR-29
          disposition: addressed
          note: console_webcast.go:275-276 reads update, phase and cast in one locked snapshot; wcastCast deleted, so no second read remains.
          round: 10
      boundary: M3
      recipe: milestone-review
      isolation: fenced
      reviewed: 0204724cbeb2faaf5d4ef387e3a6fd1f2cc6cd8d
      blocked: false
---

# Gate ledger — pair#467 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T07:00:57-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `state-scope-unmodelled` The dialog signal isn't tied to the cast tab: an open while paused is missed, and any tab's close clears it
  chrome.go only counts dialogOpened when castTab is set (empty while paused), accepts dialogClosed from any session, and never re-checks on a cast-tab change or destroy. Confirmed with temporary loop tests. The driver's calls then hang until the 5 s timeout (500, not the fixed sentence) while the recorder loop stalls. Fix: a per-session set of open dialogs in loopState; the signal is "cast tab's session is in the set", recomputed on every change.
- **BR-2** [Important] `live-conformance-unasserted` TestLiveDrive doesn't check the file chooser, the download denial or the restore after SetDriving("")
  live_drive_test.go only checks the file and download clicks return no error. Task 5 and Done-when ask for fileChooserOpened seen with the intercept on, no download, and the intercept off after SetDriving(""). The fake-conn loop test only checks commands are sent, not real Chrome's behavior.
- **BR-3** [Minor] `residue-bound-on-writer` Per-seat rate buckets are reset wholesale at 256 instead of pruned when a seat leaves, as planned
  server_drive.go limits() clears the whole map at maxSeatLimits; a departed seat's bucket stays until its key's next request. A reset also restores well-behaved seats' budgets. Prune on the controller's leaves, or record the change in Revisions.
- **BR-4** [Minor] `plan-drift-unrecorded` Route budgets (no maxViewers x2 hello cap) and the all-pages file-chooser intercept differ from the plan text
  Bounded equivalently by the hello rate times the grace period; record both in the plan's Revisions.
- **BR-5** [Minor] `guard-scope-by-phase` A page the cast tab opens while paused-but-driving isn't closed (castTab is empty while paused)

## Round 2 — 2026-10-12T07:08:49-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Per-session dialogs set, recomputed in publish (chrome.go:545); TestChromeCastTabDialog fails with the old logic restored ("dialog signal never became true").
- BR-2 — addressed — live_drive_test.go now asserts Page.fileChooserOpened via tap, download denial against a positive control, and no chooser after SetDriving(""); not executed here (Chrome GPU crash on this host).
- BR-3 — not-addressed — Sweep and Revisions entry landed, but TestDriveSeatTableBounded passes against the old reset (flood keys are dropped on ErrNoSeat, so the table never fills); limits() uses time.Now() inline, so the idle sweep can't be tested — inject now and pin that a full table sweeps idle keys and doesn't refill a seat's budget.
- BR-4 — addressed — Plan Revisions 2026-10-12 records the route budgets without the maxViewers x2 cap and the all-pages chooser intercept.
- BR-5 — addressed — castTab drops the PhaseLive gate (chrome_drive.go:90); restoring the gate makes TestChromeCastTabDialog fail ("never got Target.closeTarget T4").

### Raised

- **BR-6** [Minor] `test-hook-in-production-state` loopState.tap is a test-only event hook carried in the production loop struct
  chrome.go:140; nil in production and only set on the loop goroutine, so harmless today. Prefer one injected observer seam if more test hooks accumulate.

## Round 3 — 2026-10-12T07:25:56-07:00 (claude) — BLOCKED

### Raised

- **BR-7** [Important] `client-honors-wire-contract` batcher caps acts and text runes but not body bytes; a paste over about 8 KB is refused (413) whole
  drive.MaxBody is 8 KB. A probe of drive.js batcher: 9000 ASCII chars gave a 9198-byte batch and 3000 CJK chars 9078 bytes; admitBody answers 413 and the queue is dropped. Bound go() by serialized UTF-8 size (and split text by bytes), with a test that every sent body is at most MaxBody.
- **BR-8** [Important] `client-honors-wire-contract` a captured drag leaving the frame left or top emits negative page coords; ParseBatch 400s and the up is never delivered
  toPage does not clamp and the pointer is captured; coord() requires >= 0. Probe: down(50,50), move(-50,50), up(-60,50). The cast tab keeps a pressed button. Clamp in gestures' pt/at and add a trace test that leaves the frame.
- **BR-9** [Important] `plan-drift-unrecorded` Task 9 live test uses a stateless spyInjector, not MemController over FakeInjector, and the plan has no Revisions entry
  2nd in family. Rule: a Task's named files, fixtures and doubles are contract; any deviation gets a Revisions line in the milestone's closing commit. Sweep M2: the spy, the "Cancel request" label, 1024x700 instead of tablet width, Rejoin by reload. The spy skips the seg, Confine and stale rules, so a wrong seg from the shell would pass.
- **BR-10** [Minor] `shell-state-overwritten` offerRejoin's button is overwritten by the next render(), so the 401 Rejoin offer disappears
- **BR-11** [Minor] `shared-helper-reuse` drive.js copies annotate's toast (same .annotate-msg class, can overlap) and collab.poster's request options
- **BR-12** [Minor] `input-pairing` chord keyup is dropped when the modifier is released first; hover moves are sent while a button is held after a press on nothing
- **BR-13** [Minor] `client-honors-wire-contract` batcher does not pace to the server's 30/s input budget; fast key auto-repeat can 429 and drop the queue
- **BR-14** [Minor] `served-asset-coverage` drive.css is served but no test fetches it; the import-walk test covers only JS modules

## Round 4 — 2026-10-12T07:41:33-07:00 (claude) — BLOCKED

### Disposed

- BR-7 — addressed — take() bounds by serialized UTF-8 size and splitText cuts by bytes; reverting the MAX_BODY break turns the property test at drive.test.mjs:103 red.
- BR-8 — addressed — wireAct clamps x/y into [0, MAX_COORD]; removing the clamp fails 3 tests, including the leaves-the-frame trace at drive.test.mjs:272.
- BR-9 — addressed — Live test runs MemController over FakeInjector (spy only records) and asserts an attributed Click in the stream; Revisions entry covers the spy, labels, viewport and Rejoin. Could not execute here: Chrome CDP EOF also fails the pre-existing TestLiveViewerPage.
- BR-10 — not-addressed — The lost flag and early return in render() look correct, but no test fails without it: mountDrive has no harness. Fold lost into driveView (pure) and table-test it.
- BR-11 — addressed — collab.js sayer (one shared .annotate-msg element) and poster(headers) are used by both drive.js and annotate.js; the copies are gone.
- BR-12 — addressed — keyTracker test covers the up after the modifier is released; reverting the !e.buttons hover guard fails the gestures test.
- BR-13 — addressed — PACE_MS=40 is enforced in go(); forcing wait=0 fails the pacing assertion in the property test.
- BR-14 — addressed — TestViewerImportsAreServed now walks the page's stylesheet links and requires drive.css to be served as text/css.

### Raised

- **BR-15** [Important] `client-honors-wire-contract` batcher stamps seg at send time, so acts queued before a new seg record are injected into a page the driver never saw
  This is the 4th finding in family client-honors-wire-contract. Rule covering all of them: every field of a request is fixed at the moment of the user act it describes, and the gate only gathers and bounds, never re-derives. drive.js:107 reads deps.seg() inside take(), but the server's seg check (injector.go:35) is meant to refuse acts aimed at a segment that has gone. Text and key acts have no id, so the stale check cannot catch them. A resume after a pause does the same, because the batch survives the pause and the keys textarea still queues input. Fix: stamp seg in push(), have take() end a batch where the stamp changes, and add property-test cases where the seg value changes between push and send.
- **BR-16** [Important] `input-pairing` a press or key held when control ends, or when a refusal drops the queue, is never released in the cast tab
  This is the 2nd finding in family input-pairing. Rule: every down the driver starts is ended on every exit path (refusal, revoke, take, seg change, seat loss), and that duty belongs to the side that outlives the exit, the server. batcher.stop() and the refusal path drop queued up and key-up acts. gestures' press is never reset, so the next drive sends left-button moves with no down. Nothing server-side releases held buttons or keys on a control change. Fix: on a control change, have TabInjector/WithTab dispatch mouseReleased and key-up for whatever the previous driver left down; add a reset() to gestures and keyTracker and call it from render() when the batch stops.
- **BR-17** [Minor] `state-scope-unmodelled` shell states (lost, driving-but-paused) live as flags in render() rather than in driveView
  This is the 2nd finding in family state-scope-unmodelled. driveView should return the whole state set (none, asking, driving-live, driving-paused, other, lost) so render() only paints it, and every state can be table-tested without a DOM.

## Round 5 — 2026-10-12T07:50:27-07:00 (claude) — BLOCKED

### Disposed

- BR-10 — addressed — lost is now a driveView state (action rejoin) painted by every render(); table-tested in drive.test.mjs.
- BR-15 — addressed — seg stamped in push(), take() cuts at seg change, no coalescing across segs; batcher test goes red with the cut reverted.
- BR-16 — addressed — TabInjector tracks held button/keys, releases on other driver, other seg, refusal and control change (Releaser); gestures.reset and tracked.clear on end/refusal; TestControllerLetsGoOnRevoke goes red with Release disabled.
- BR-17 — addressed — driveView returns none/asking/driving/paused/other/lost with action and capture; render only paints; table test covers all rows.

### Raised

- **BR-18** [Important] `atlas-tracks-surface` atlas/webcast.md does not describe the cast tab's let-go (Releaser on control change, refusal, driver or page change) or per-act segments
  Remote driving section lines 197-215 still says only that the batcher drops its queue at a refusal; add a Let go bullet and the one-segment-per-batch rule.
- **BR-19** [Minor] `input-pairing` Release ends whatever is held when it reaches the tab, not the outgoing driver's hold
  3rd in family input-pairing. Rule: every release names the hold it ends (driver, seg) and is a no-op otherwise. The async release from changeLocked (controller_mem.go:162) could otherwise drop the next driver's press if that driver's batch runs first; pass by into Release and test revoke, grant, press, then late Release.
- **BR-20** [Minor] `dead-code` duplicate return nil at drive/tab.go:164; go vet reports unreachable code
- **BR-21** [Minor] `residue-bound-on-writer` held.keys appends every auto-repeat keydown, so a release sends N keyups for one key
  2nd in family. Rule: a server-held collection fed per input act is keyed by identity at its writer. Fix by deduping by key in TabInjector.note (tab.go:107) or filtering e.repeat in keyTracker.down.

## Round 6 — 2026-10-12T07:54:25-07:00 (claude) — passed

### Disposed

- BR-18 — addressed — atlas/webcast.md:211-228 adds the Batches rules (one segment per batch) and a Letting go section listing four cases with Releaser naming the outgoing driver; webcast-format.md:601-603 matches tab.go.
- BR-19 — addressed — Release(ctx, by) is a no-op unless held.by==by (tab.go:75-85), and controller_mem.go:165 passes the outgoing driver. With the check reverted, release_test.go:112 fails.
- BR-20 — addressed — The duplicate return is gone (tab.go:176), and go vet ./cmd/internal/webcast/... is clean.
- BR-21 — addressed — note() dedups held keys by Key at the writer (tab.go:122). With the dedup reverted, release_test.go:116 fails with two keyups.

## Round 7 — 2026-10-12T08:22:10-07:00 (claude) — BLOCKED

### Raised

- **BR-22** [Important] `plan-drift-unrecorded` M3 ships without drive_menu.go or a DriveMenu model and adds webcast.FakeDriven, but the plan has no M3 Revisions entry
  The Core concepts row and Task 10 Files name couchtty/drive_menu.go and a DriveMenu model; the code is driveSpan/driveMenuRows in console_drive.go, generalized only by RenderTitledMenu, and tests run over a new exported webcast.FakeDriven that no table lists. This is the 3rd finding in family plan-drift-unrecorded; the M2 rule (any deviation gets a Revisions line in the closing commit) was stated but not enforced. Rule to fix: the milestone close mechanically checks every path in the Core concepts table and each Task's Files list against the tree and refuses the close unless a Revisions line names each mismatch.
- **BR-23** [Important] `error-names-its-cause` driveResult reports every Grant/Deny/Revoke error as "that changed meanwhile", including a revoke that failed outright
  In console_drive.go:261, a sink failure (recorder or liveGate stopped) or ErrClosed on Revoke leaves the driver driving and the guards on, but the operator is told it was a race. Switch on the error class: keep the race sentence for ErrNoSeat and NextControl refusals, and give a failure sentence with the cause otherwise, ideally from drive.Answer's table. Add a FakeDriven Emit-failure test that goes red without the fix.
- **BR-24** [Minor] `test-gesture-coverage` TestDriveFromCouch skips the deny notice and Ctrl+click with askers only, and has a wrong comment at line 523
  The plan promised revoke from every state. The askers-only Ctrl+click no-op is untested, "Said no to Bo." is never asserted, and the comment "nobody drives" is wrong because Bo drives at that point.

## Round 8 — 2026-10-12T08:59:18-07:00 (claude) — BLOCKED

### Disposed

- BR-22 — addressed — Plan Revisions 2026-10-12 "M3 as built" names the missing drive_menu.go/DriveMenu (driveMenu source + RenderTitledMenu) and the exported webcast.FakeDriven; the two-links decision's new surface is also recorded.
- BR-23 — addressed — driveFailure classes errors (console_drive.go:223); reverting it to the race sentence turns TestDriveFromCouch red at console_drive_test.go:304 (FailEmits revoke).
- BR-24 — addressed — console_drive_test.go:244 asserts "Said no to Bo.", :274-282 covers askers-only Ctrl+click as a no-op, and :253's comment now follows control coming back.

### Raised

- **BR-25** [Important] `atlas-tracks-surface` README not updated for M3 surface: drive span/menu, Ctrl+click take-back, Alt+right-click drive link, --broadcast-list shares
  2nd finding in this family. README.md:400/:864 still describe --broadcast-list as broadcast plus viewer count, :956 says right-click copies "the viewer link", and nothing covers the drive span or menu. Rule: a milestone commit that adds a gesture, flag, notice or output format updates README and atlas in the same commit; add README to Task 12 / the M3 close.
- **BR-26** [Minor] `shared-helper-reuse` driveFailure repeats drive.Answer's error classification; only its default branch is tested
  2nd finding in this family. Rule: one classification table per error set (drive.Answer's), with per-audience sentences derived from it; driveFailure's race, cast-ended and not-live sentences need a pure table test.
- **BR-27** [Minor] `error-names-its-cause` Failure to mint the drive token silently turns driving off, and Couch then says the cast isn't recorded
  cast.go:236 drops the controller with no log; copyWebcastDriveLink then names the wrong cause. 2nd finding in this family; rule: every path that turns a feature off records the cause it gives the operator.
- **BR-28** [Minor] `state-scope-unmodelled` Server.Viewers reads two counters separately; the view count can briefly be off by one, even negative
  server.go:631 subtracts driveViewers from viewers read at a different moment; display only. 3rd finding in this family; rule: a count derived from two counters is read in one snapshot (one counter per link, or a mutex).

## Round 9 — 2026-10-12T09:08:10-07:00 (claude) — BLOCKED

### Disposed

- BR-25 — addressed — README.md:400, :864-869 and :958-980 now cover the shares by kind, view vs drive link (Alt+right-click), the drive span and menu (grant, deny, take back) and Ctrl+click on the span. Checked against terminal_input.go:149-170 and console_drive.go.
- BR-26 — not-addressed — TestDriveFailure now pins every class, but driveFailure (console_drive.go:226) still repeats drive.Answer's errors.Is table instead of deriving from one classification. Minor.
- BR-27 — addressed — cast.go:159-171 sets driveOff and logs it; copyWebcastDriveLink names DriveOff(); TestCastWithoutDriving pins the cause (fails to compile without the fix).
- BR-28 — addressed — server.go:67-69/:152-157 keep one counter per link and Viewers reads each directly, so no count is derived from two reads; the race isn't testable deterministically, and TestDriveOnlyUnderTheDriveLink checks the counts.

### Raised

- **BR-29** [Important] `read-in-one-snapshot` copyWebcastDriveLink reads c.wcast.cast twice under separate locks; a cast ending in between panics on a nil DriveOff
  console_webcast.go:284 calls c.wcastCast() twice; if webcastEnded resets c.wcast between the two calls, (*Cast)(nil).DriveOff() dereferences nil. Read cast in the same locked snapshot as u at :275 and delete wcastCast. Same rule as BR-28: a decision over several fields of shared state reads them in one snapshot.

## Round 10 — 2026-10-12T09:24:36-07:00 (claude) — passed

### Disposed

- BR-26 — addressed — TestDriveFailure (console_drive_test.go:320-336) pins every branch exactly; the operator-side switch remains a distinct per-audience sentence table, acceptable for a Minor.
- BR-29 — addressed — console_webcast.go:275-276 reads update, phase and cast in one locked snapshot; wcastCast deleted, so no second read remains.

## Open findings

- **BR-3** [Minor] `residue-bound-on-writer` Per-seat rate buckets are reset wholesale at 256 instead of pruned when a seat leaves, as planned
- **BR-6** [Minor] `test-hook-in-production-state` loopState.tap is a test-only event hook carried in the production loop struct
