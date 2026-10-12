# Boundary Review — pair#467 (milestone M1)

| field | value |
|-------|-------|
| issue | 467 — broadcast-web: remote driving — a viewer takes control and uses the app through the operator's tab |
| repo | pair |
| issue file | workshop/issues/000467-broadcast-web-remote-driving-a-viewer-takes-control-and-uses-the-app-through-the-operator-s-tab.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | ad30341aeb7ace1812b7fcc2b77415cbd9592889..1af28a69a603a7b004d4df28b6812c5a58b8ea8a |
| command | sdlc milestone-close --issue 467 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T07:00:56-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the M1 diff — the drive and recorder packages are done; next is the Chrome guards, cast wiring and the HTTP routes.

Now checking the recorder and route tests, then running the suite.

I found two Important issues (one bug, one test gap). The rest only needs minor fixes, so the verdict is **FIX-THEN-SHIP**.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers the Go core the plan describes. The injector's rules now exist only in `drive.TabInjector`, and both the real recorder and the fake tab run them. Attribution works by holding the in-page observer before the driver's input and taking its queued items after, all inside one recorder-queue input, so the driver's records carry `by` in stream order. That design is sound and well tested. `go test ./cmd/internal/webcast/... -race` passes on the pinned head.

Two things should be fixed before the boundary:

1. **The dialog signal is tracked in the wrong scope.** Only an "open" event on the cast tab while live is counted. A "closed" event from any tab clears it. I confirmed both failures with temporary tests in this checkout:
   - A dialog opened on the cast tab while paused is never signalled after going live.
   - Another session's dialog closing clears the cast tab's signal.

   In both cases the driver's input calls hang until the 5 s timeout and the driver gets a 500. Decision 7 and the TL's request were for a fixed sentence instead. The recorder's loop is also stalled meanwhile.
2. **The live test doesn't check what it claims.** The issue's Done-when lists file chooser and download in the live test, and the plan also asks for the restore after `SetDriving("")`. The live test only checks that the file-input and download clicks return no error.

## 1. Strengths
- **One rule set, two tabs:** `drive/tab.go:38-68` holds the injector's rules once, and `FakeInjector` is now just those rules over `fakeTab`. `onTarget` is pure and covered for self, child, shadow root, frame document, out-of-process frame owner and missing node (`tab_test.go:38-56`).
- **Attribution by hold/take inside one queue input** (`recorder/drive.go:37-57`) avoids guessing from arrival time. The release paths are tested: stale target, failed take, dialog in the middle of a dispatch (`recorder/drive_test.go:176-246`). The 1 s expiry in the page means the observer can't stay stuck.
- **`liveGate` reads the recorder's own `Opened()` flag** (`cast_drive.go:38-44`). `TestCastDriving` covers the gap where Chrome publishes live before the recorder opens. `TestMemControllerSinkRefusalChangesNothing` shows a refused change leaves state and subscribers untouched.
- **`cdpAct` is a pure, table-tested mapping.** The live run caught the user-agent shadow DOM bug, which the fake couldn't (`recorder/drive.go:155-158`).
- **`admitBody` is now shared** between the board and drive routes (`server_board.go:149`). It's a real consolidation (ARCH-DRY).

## 2. Critical findings
None.

## 3. Important findings
- **The dialog signal isn't tied to the cast tab's session** (`chrome.go` dialogOpened/dialogClosed cases; `chrome_drive.go:56-98`; ARCH-ORDER).
  - **What's wrong:**
    - `dialogOpened` only counts when `castTab(s)` is set, and that is empty while paused.
    - `dialogClosed` is accepted from any session.
    - Nothing re-evaluates the signal when the cast tab changes or is destroyed.
  - **Confirmed:** a dialog opened on T1 while paused is not signalled after going live. A `javascriptDialogClosed` on another session S7 clears T1's signal.
  - **Effect:** each input call blocks until the 5 s timeout, then the deferred take blocks 5 s for each observer context. The driver gets a 500 instead of the fixed sentence.
  - **Fix:** keep a set of sessions with an open dialog in `loopState`, updated on open and close for every session and cleared on detach or destroy. Recompute the signal (cast tab's session in that set) whenever the dialog set or the cast tab changes. Add loop tests for: dialog opened while paused, another tab's dialog closing, cast tab switch with a dialog open, and the dialog's tab being destroyed.
- **The live test doesn't check the file chooser, download or restore** (`live_drive_test.go:168-174`; ARCH-MOCK conformance).
  - **What's wrong:** the file-input and download clicks only check for no error. The plan wanted three checks: `Page.fileChooserOpened` seen with the intercept on, no downloaded file, and the intercept off after `SetDriving("")`. None is made.
  - **Why it matters:** the fake-connection loop test only checks that the commands are sent. Nothing checks that real Chrome behaves the way the fake assumes. The Log's "asserted in the loop test" isn't live evidence.
  - **Fix:** listen for `Page.fileChooserOpened` on the cast session during the file click. Use `Browser.setDownloadBehavior` in the test Chrome, or watch the `Browser.downloadWillBegin`/`downloadProgress` "canceled" state. Then call `SetDriving("")` and check that the file chooser is shown normally again (no `fileChooserOpened` event).

## 4. Minor findings
- **Per-seat rate buckets:** the plan says they are pruned when the controller reports a seat leaving (bounded by live seats + 1). The code instead drops the whole map at 256 entries (`server_drive.go:150`), and a departed seat's bucket stays until that key's next request. A wholesale reset also resets well-behaved seats' budgets (ARCH-FUNERAL).
- **`hello` cap missing:** the plan's cap of `maxViewers×2` live seats isn't implemented. The code relies on the rate limit plus the grace period, which bounds it equivalently, but no Log entry or Revision records the change.
- **New targets while paused:** `closeOpened` uses `castTab`, which is empty while paused. A page the cast tab opens while paused-but-driving isn't closed, and could become the cast tab on resume.
- **Missing test:** the plan's test with a revoke enqueued "between hold and take" doesn't exist. It can't happen structurally (the run is one queue input), so a one-line note in the plan would close it.
- **Receiver name:** `MemController` methods still use the receiver `f` (from "fake"). Cosmetic.

## 5. Test coverage notes
- Well covered:
  - Hold/take attribution and ordering, the release paths, and a revoke queued first.
  - The control gate before the recorder opens, and a revoke overtaking a blocked batch (`cast_drive_test.go`).
  - The routes walked through every `Answer` row, per-seat rate limits, a seat surviving while its stream lives, and `FuzzDriveInput`.
- Gaps:
  - Dialog signal scoping (above).
  - Real-Chrome effect of the guards (above).
  - No test of new targets opened while paused.

## 6. Architectural notes for upcoming work
- **ARCH-DRY: pass.** The rules exist once and `admitBody` is shared. M3 should still generalize the notes menu rather than copy it, as planned.
- **ARCH-PURE: pass.** `onTarget`, `cdpAct`, `editCommand` and `TabInjector`'s rules are pure. All CDP traffic is behind `cdpTab` and the `Tab` seam.
- **ARCH-PURPOSE: pass for M1's scope.** Shell, Couch and fixture are planned for M2–M4.
- **ARCH-MOCK: flag (Important above).** The fake connection models the commands sent, but the live check doesn't confirm the guards' effect.
- **ARCH-CONSTRAINTS: pass.** Per-seat budgets, a 64-act batch cap, and the 5 s call timeout are enforced. The dialog bug breaks the "returns at once" bound only in the unsignalled cases.
- **ARCH-SECURE: pass.**
  - The driver name reaches the page via `json.Marshal` into the expression and `textContent` on the page side.
  - Bodies are size-capped and parsed strictly.
  - Seat keys are 32 random bytes from `crypto/rand`.
  - Errors never echo the input back.
- **ARCH-ORDER: flag.** `dialogSignal` is state carried between events whose transitions depend on scope (session, cast tab, phase), and that scope isn't modelled (Important above). The single-queue ordering of `ctl` records and injection is explicit and tested.
- **ARCH-FUNERAL: mostly pass.** Seats and subscriptions die at grace or `Close`, and `followDriving` ends when the controller closes. The abandoned `inputCall` goroutines are bounded by `callTimeout`. The one deviation is how the per-seat bucket map is pruned (Minor above).

## 7. Plan revision recommendations
- Add a Revisions entry saying the file-chooser intercept applies to every page session, not just the cast tab. The Log mentions this, but the Integration-points text still says cast tab.
- Add a Revisions entry for the route budgets as built: a 256-entry wholesale reset instead of pruning on leave, and no `maxViewers×2` hello cap. Or implement the planned pruning.
- Update the Task 5 wording, or the test, so the live checks for file chooser, download and restore are real.
- Note that the "revoke between hold and take" test is structurally impossible and so isn't written.

```findings
findings:
  - id: new
    severity: Important
    family: state-scope-unmodelled
    title: |
      The dialog signal isn't tied to the cast tab: an open while paused is missed, and any tab's close clears it
    detail: |
      chrome.go only counts dialogOpened when castTab is set (empty while paused), accepts dialogClosed from any session, and never re-checks on a cast-tab change or destroy. Confirmed with temporary loop tests. The driver's calls then hang until the 5 s timeout (500, not the fixed sentence) while the recorder loop stalls. Fix: a per-session set of open dialogs in loopState; the signal is "cast tab's session is in the set", recomputed on every change.
  - id: new
    severity: Important
    family: live-conformance-unasserted
    title: |
      TestLiveDrive doesn't check the file chooser, the download denial or the restore after SetDriving("")
    detail: |
      live_drive_test.go only checks the file and download clicks return no error. Task 5 and Done-when ask for fileChooserOpened seen with the intercept on, no download, and the intercept off after SetDriving(""). The fake-conn loop test only checks commands are sent, not real Chrome's behavior.
  - id: new
    severity: Minor
    family: residue-bound-on-writer
    title: |
      Per-seat rate buckets are reset wholesale at 256 instead of pruned when a seat leaves, as planned
    detail: |
      server_drive.go limits() clears the whole map at maxSeatLimits; a departed seat's bucket stays until its key's next request. A reset also restores well-behaved seats' budgets. Prune on the controller's leaves, or record the change in Revisions.
  - id: new
    severity: Minor
    family: plan-drift-unrecorded
    title: |
      Route budgets (no maxViewers x2 hello cap) and the all-pages file-chooser intercept differ from the plan text
    detail: |
      Bounded equivalently by the hello rate times the grace period; record both in the plan's Revisions.
  - id: new
    severity: Minor
    family: guard-scope-by-phase
    title: |
      A page the cast tab opens while paused-but-driving isn't closed (castTab is empty while paused)
```

---

## Re-review — 2026-10-12T07:08:49-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 467 — broadcast-web: remote driving — a viewer takes control and uses the app through the operator's tab |
| repo | pair |
| issue file | workshop/issues/000467-broadcast-web-remote-driving-a-viewer-takes-control-and-uses-the-app-through-the-operator-s-tab.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | ad30341aeb7ace1812b7fcc2b77415cbd9592889..34a9eff353447e2edf56dde561955a1736ed346c |
| command | sdlc milestone-close --issue 467 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T07:08:49-07:00 |
| verdict | SHIP |

## Review

Prior-round fixes live in the last commit. Let me look at that first.

Need to keep the `tap` field; revert only the logic.

Still reviewing: the prior-round fixes check out in the unit tests. Real Chrome crashed inside the sandbox, so I'm trying the live test once outside it.

```verdict
verdict: SHIP
confidence: medium
```

All three round-1 fixes that change behaviour (BR-1, BR-2 and BR-5) hold up, and BR-4's plan changes are recorded. Undoing BR-1 makes `TestChromeCastTabDialog` fail ("dialog signal never became true"). Undoing BR-5 makes it fail too ("never got Target.closeTarget T4"). `go test -race ./cmd/internal/webcast/...` and `go vet` pass at head. Two things stop this from being high confidence. First, BR-3's new test still passes against the old reset code, so it doesn't prove the sweep. That's a Minor finding and doesn't block. Second, I couldn't run `TestLiveDrive`: Chrome's GPU process crashes on this host, both inside and outside the sandbox ("GPU process isn't usable"). So my BR-2 check is from reading the code, not from a run.

**1. Strengths**
- `chrome.go:545`: the dialog signal is now worked out from `s.dialogs[sessionOf(s, castTab(s))]` after every event the loop handles. It's computed from state instead of being a flag flipped by events, so the cases that broke before can't happen: a dialog opened while paused, another tab's close event, a tab switch, a closed tab. A session's entry is also removed when the page detaches (`chrome.go:399`).
- `chrome_drive.go:73`: `dialogSignal.set` is one idempotent setter, replacing the separate `open`/`closed` pair.
- `chrome_loop_test.go:470`: `TestChromeCastTabDialog` walks through every case from BR-1 and BR-5 in order. Both regressions turn red when their fix is removed.
- `live_drive_test.go:89`: the download check now has a positive control. With downloads allowed, the operator's download lands in the temp folder; while a viewer drives, the driver's doesn't. This turns "no error came back" into evidence that downloads really are denied.
- `liveGate` (`cast_drive.go:37`) asks the recorder whether it has opened rather than trusting Chrome's published phase. That's a careful choice about ordering.

**2. Critical findings**
None.

**3. Important findings**
None.

**4. Minor findings**
- BR-3, still open (see dispositions): `TestDriveSeatTableBounded` passes against the old reset. Each flood key is dropped as soon as the controller refuses it, so the table never fills. `limits()` calls `time.Now()` directly (`server_drive.go:155`), so the minute-long idle sweep can't be tested without waiting a minute. Also, when the table is full of keys used in the last minute, a real new seat gets a zero-rate budget and is refused with 429 "busy".
- `loopState.tap` (`chrome.go:140`) is a hook for tests only, living in the production struct. It's nil in production and only set on the loop goroutine, so it's harmless. Worth noting if more hooks like it pile up.
- `live_drive_test.go:255`: the check that the file-chooser intercept is off after driving only looks for an event that doesn't arrive. If the operator's click missed `#file`, it would pass anyway. It would be stronger if it also confirmed the click landed (for example, via a focus or click record).

**5. Test coverage notes**
- Undoing each fix and re-running its test:

  | Fix | Undone | Result |
  |---|---|---|
  | BR-1 | dialog logic | test fails |
  | BR-5 | only the `castTab` phase gate | test fails |
  | BR-3 | `server_drive.go` back to the base version | **test still passes** |

- I couldn't run `TestLiveDrive` here (Chrome crashes on this host). The new assertions are in the code and would fail if the behaviour broke: the `Page.fileChooserOpened` tap, the download count staying the same after the driver's click, and no chooser event after driving ends.

**6. Architecture**
- ARCH-DRY: pass. The two dialog setters are merged into one, and `castTab` is the one place that decides which tab the guards protect.
- ARCH-PURE: pass, with the BR-3 caveat. `limits()` reads the clock inline even though the `allow*` funcs it sits next to take `now` as an argument.
- ARCH-PURPOSE: pass. M1's tasks 1–7 are delivered, and the plan's Revisions record the differences.
- ARCH-MOCK: pass. The fake connection (`startPeer`) shares the production loop, and `TestLiveDrive` checks the real-Chrome behaviour (not run here).
- ARCH-CONSTRAINTS: pass. The seat table is capped, requests in flight per seat are capped, and hello is rate-limited.
- ARCH-SECURE: pass. Unknown seat keys are refused and dropped, and request bodies are size-capped and strictly decoded.
- ARCH-ORDER: pass. The dialog state is a per-session set the signal is computed from, not a separate flag. The recorder's hold, dispatch and take run as one queue input.
- ARCH-FUNERAL: pass. Dialog entries die when their page detaches, and seat entries are dropped when refused or swept when idle. `inputCall` goroutines are bounded by `callTimeout`.

**7. Plan revision recommendations**
None needed. The 2026-10-12 Revisions entry matches the code: choosers intercepted on all pages, `castTab` in every phase, the route budgets, the idle sweep, and the impossible "revoke between hold and take" ordering.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Per-session dialogs set, recomputed in publish (chrome.go:545); TestChromeCastTabDialog fails with the old logic restored ("dialog signal never became true").
  - id: BR-2
    disposition: addressed
    note: |
      live_drive_test.go now asserts Page.fileChooserOpened via tap, download denial against a positive control, and no chooser after SetDriving(""); not executed here (Chrome GPU crash on this host).
  - id: BR-3
    disposition: not-addressed
    note: |
      Sweep and Revisions entry landed, but TestDriveSeatTableBounded passes against the old reset (flood keys are dropped on ErrNoSeat, so the table never fills); limits() uses time.Now() inline, so the idle sweep can't be tested — inject now and pin that a full table sweeps idle keys and doesn't refill a seat's budget.
  - id: BR-4
    disposition: addressed
    note: |
      Plan Revisions 2026-10-12 records the route budgets without the maxViewers x2 cap and the all-pages chooser intercept.
  - id: BR-5
    disposition: addressed
    note: |
      castTab drops the PhaseLive gate (chrome_drive.go:90); restoring the gate makes TestChromeCastTabDialog fail ("never got Target.closeTarget T4").
findings:
  - id: new
    severity: Minor
    family: test-hook-in-production-state
    title: |
      loopState.tap is a test-only event hook carried in the production loop struct
    detail: |
      chrome.go:140; nil in production and only set on the loop goroutine, so harmless today. Prefer one injected observer seam if more test hooks accumulate.
```
