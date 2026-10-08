# Boundary Review — pair#395 (milestone M1)

| field | value |
|-------|-------|
| issue | 395 — Couch broadcast: stream the composed Couch screen, view-only, to a browser viewer |
| repo | pair |
| issue file | workshop/issues/000395-couch-broadcast-stream-the-composed-couch-screen-view-only-to-a-remote-couch-watch.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 6986321e216f9da591464e5991bcb03a5675fdf4..fe074624c72f68b26a9e49408dfa653c9c2d5daa |
| command | sdlc milestone-close --issue 395 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-07T15:36:36-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers what the plan claims for this boundary. There is a Presenter tap at the single paint convergence point, with a privacy class set by `Select`/`Panel`. The pure `IndicatorShown`/`ViewerFrame`/`Stream` checks are colocated with emulator-backed tests. The hub withholds frames, stops after the grace period, resyncs slow viewers and enforces a viewer cap, and its seeded interleaving test checks real invariants. `go test -race ./cmd/internal/broadcast/` and `go test ./cmd/internal/terminal/ -run 'Tap|Panel'` pass. The `couchtty` failures and the one `terminal` stall-test failure are sandbox pty/mkdir denials ("operation not permitted"), not caused by this diff.

One real bug blocks a clean SHIP. `Hub.Activate()` arms the grace timer even when the last accepted frame already showed `LIVE ⏸`. If the operator's screen then stays idle, the broadcast ends with `ErrIndicatorHidden` one second later. I confirmed this with a scratch overlay test (a live frame is offered, then `Activate()`, then grace fires, and the hub ends). The fix is small, and it should land before M3 wires Console ordering on top of it.

1. **Strengths**
   - `presenter.go:380`: the tap fires only after both the write and the `PresentView` transition succeed, and only clones when a tap is set. So viewers get exactly the frames that were presented, and there is no cost while no broadcast is running.
   - `indicator.go:24`: `IndicatorShown` checks the content *and* the red background cell by cell at column 0 of the last row. `TestIndicatorShown` covers the cases where a lookalike should be rejected: unstyled, other style, other row, offset, clipped, and the Starting label.
   - The switcher frame (`console_menu.go:217`) includes the chrome row, so `ViewerFrame`'s carried-through tab bar keeps the indicator visible. Switcher frames therefore pass the withholding check and are not starved into a grace stop.
   - The hub's resync logic (`hub.go:268`) never sends a diff against a frame the viewer missed. `TestHubSlowViewerResyncs` and the property test check this through a real VT emulator, not by restating the implementation. The Log records that the property test was mutation-checked against the three failure modes it targets.
   - `artifactpath/manifest.go` classifies every new production source.

2. **Critical findings:** none.

3. **Important findings**
   - `hub.go:131` (`Activate`): it sets `missing = true` and arms grace whenever `!missing`. That flag only means "the indicator is absent", though; before `Activate`, it stays false whatever the frame showed. So `Activate()` after a live frame has already been accepted puts the hub in "missing" with LIVE on screen. With no further paint, the broadcast dies after 1s. Console's tap/repaint/`Activate` calls run on different goroutines, so M3 can easily produce that ordering.
     - **Fix:** track `shown bool` (whether the last offered frame passed `IndicatorShown`) on every `accept`. `Activate` should arm only when `!shown`.
     - **Test:** add a `TestHubActivateAfterLiveFrame` regression test (offer live, Activate, fire grace, expect not ended). Also let the interleaving test call `Activate` at a random step instead of always first.
     - **ARCH-ORDER:** this is the flag-constellation hazard, since `active`/`missing`/`graceC` have unwritten legal combinations. Collapsing them into an explicit `inactive | shown | hidden(timer)` state would make it unrepresentable.

4. **Minor findings**
   - The plan says the end of a broadcast delivers `Message{End: true, Reason}` (or `End{Reason}`) to subscribers. The code only closes the channels, so M2's server must read `Hub.Err()`, and it must tell a hub end apart from its own `Subscription.Close`. Either implement the End message or revise the plan.
   - The plan's `TestTapNotCalledOnFailedPaint` also asks for the case where the `PresentView` transition is refused. Only the write-failure case is tested.
   - The real 100ms ticker runs for the hub's whole life even with no subscribers. It's negligible, but it could stop while no subscriber is resyncing.
   - `placeholderBody` truncates by byte (`text[:len(text)-1]`). That's safe only because `PlaceholderText` is ASCII; worth a comment.

5. **Test coverage notes**
   - Tests use real `Render` output through the `vt` emulator, which is good.
   - The gap is ordering. The property test always calls `Activate()` before any offer, which is the one ordering that hides the bug above. Randomize where `Activate` happens in the sequence.

6. **Architectural notes**
   - **ARCH-DRY: pass.** The `LiveLabel`/`LiveSGR` constants are shared and ready for `RenderStatusRow` in M3. Add the drawer↔checker contract test then.
   - **ARCH-PURE: pass.** `Stream`, `ViewerFrame` and `IndicatorShown` are pure and tested without IO; the hub injects its timer and ticks.
   - **ARCH-PURPOSE: pass** for M1's scope.
   - **ARCH-MOCK: N/A.** M1 has no external dependency; `FakeTunnel` is M2's.
   - **ARCH-CONSTRAINTS: pass.** `Offer` never blocks (latest frame wins), and viewers and queue depth are bounded.
   - **ARCH-SECURE: pass.** Frames are in-process, and both `ViewerFrame` and `IndicatorShown` validate geometry. The token and server boundaries are M2.
   - **ARCH-ORDER: flag** (see Important).
   - **ARCH-FUNERAL: pass.** Nothing durable is created; the hub goroutine and ticker end on `Close`. The plan should make M2's `Session` own calling `Hub.Close` so the goroutine can't leak.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry recording that the hub ends by closing subscriber queues (`Hub.Err()` carries the reason) instead of sending an `End` message, or keep the plan and implement the message.
   - Record that `Activate` must not arm grace when the indicator is already shown, and that this precondition is checked by a regression test.

```findings
findings:
  - id: new
    severity: Important
    family: hub-state-flag-constellation
    title: |
      Hub.Activate arms grace even when the last accepted frame shows LIVE, so an idle screen ends the broadcast after 1s
    detail: |
      hub.go Activate sets missing=true whenever !missing, but before Activate the missing flag is never set from frame content. Reproduced with an overlay test: offer(live), Activate(), fire grace leads to ErrIndicatorHidden. Track a shown bool from every accept (or collapse active/missing/graceC into an explicit inactive|shown|hidden state, ARCH-ORDER) and arm only when not shown; add a regression test and randomize the Activate position in TestHubRandomInterleavings.
  - id: new
    severity: Minor
    family: plan-code-drift
    title: |
      Plan promises a final End message to subscribers; hub only closes channels
    detail: |
      M2 server must read Hub.Err() and distinguish a hub end from Subscription.Close. Implement the End message or add a Revisions entry.
  - id: new
    severity: Minor
    family: plan-test-coverage-gap
    title: |
      TestTapNotCalledOnFailedPaint omits the refused PresentView transition case the plan lists
  - id: new
    severity: Minor
    family: idle-background-work
    title: |
      Hub resync ticker runs for the hub lifetime even with no resyncing subscriber
```

---

## Re-review — 2026-10-07T15:52:10-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 395 — Couch broadcast: stream the composed Couch screen, view-only, to a browser viewer |
| repo | pair |
| issue file | workshop/issues/000395-couch-broadcast-stream-the-composed-couch-screen-view-only-to-a-remote-couch-watch.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 6986321e216f9da591464e5991bcb03a5675fdf4..21269f4020ed1b19db6c1ef0311e726d53823690 |
| command | sdlc milestone-close --issue 395 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-07T15:52:10-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

**Verdict: SHIP.** All four findings from the first round are fixed. BR-1 is fixed in code, and the regression tests fail when the fix is reverted. I swapped Activate's `if h.shown` for `if false` in an overlay copy of the code. Then `idle live screen at activate keeps running` failed ("grace armed 1 times"), and so did `TestHubRandomInterleavings` seed 1 ("activate with LIVE showing ended the hub"). BR-2 and BR-3 are settled by `## Revisions` entries in the plan. Those entries match the code: the hub ends by closing every subscriber queue and putting the reason in `Hub.Err()`, and the tap's `err == nil` guard has no public path that reaches it. BR-4 is fixed: `run()` only listens to the tick channel while `resyncing > 0`. `go test -race ./cmd/internal/broadcast/` passes. Three tests fail in `terminal`: `TestPresenterSurvivesTransientParentStallAndRestoresModes`, `TestPresenterRestoresModesWhenHostResumesAfterFailure` and `TestPresenterReleaseReportsModesNotRestoredWhenHostStaysStalled`. All three fail with "operation not permitted" at `presenter_stall_test.go:161`, and they fail the same way on a `git archive` of the base commit. So the cause is the environment, not this diff.

**1. Strengths**
- `hub.go:40-55`: the hub's on/off flags became one `watch` type with three states (`off | shown | hidden`), and the transition table is written above it (ARCH-ORDER). `shown` is updated on every frame the hub accepts, including frames before `Activate`.
- `hub_test.go:hubInterleaving` now calls `Activate` at a random step, or never, and checks a rule stated on its own: grace never ends the hub while LIVE is showing or before `Activate`. The overlay mutant shows this test catches the BR-1 bug.
- `indicator.go`: the status row and `IndicatorShown` read the same constants, so what gets drawn and what gets checked can't drift apart (ARCH-DRY). Frames without the indicator are held back from viewers before any rendering happens.
- `presenter.go:380`: the tap runs only after `PresentView` succeeds, and it gets a cloned frame. The privacy class is set together with `selected` in `Select` and `Panel`. `console_menu.go:230` is the only production caller of `Panel`, and it passes `FramePrivate`.
- `privacy.go`: the placeholder keeps the frame's size and its tab bar, so viewers still see the LIVE indicator.

**2. Critical:** none.

**3. Important:** none.

**4. Minor**
- `hub.go:224`: a `time.Ticker` is still created for the whole life of the hub, even though the loop only listens to it while a viewer is resyncing. The idle goroutine never wakes, so the cost is negligible. No test checks that the loop stays idle with no resyncing viewer, which is acceptable.
- `privacy.go:placeholderBody`: shortening the text one byte at a time is only correct because `PlaceholderText` is plain ASCII. It would break if the text gained multi-byte characters.

**5. Test coverage**
- The hub is driven with an injected clock (`After`/`Ticks`) and runs through randomised orderings. `TestHubRealTickerResyncsQuietScreen` also exercises the real ticker gate end to end.
- The stream, privacy and indicator code is all pure, and its tests do no IO.

**6. Architecture**
- ARCH-DRY pass. ARCH-PURE pass: indicator, privacy and stream are pure, and the hub is a thin loop around them.
- ARCH-PURPOSE pass for M1's scope.
- ARCH-MOCK: not applicable in M1. Nothing here calls an external binary or service; the tunnel arrives in later milestones.
- ARCH-CONSTRAINTS pass: `Offer` never blocks and the newest frame wins, queues are bounded, and viewers are capped.
- ARCH-SECURE pass: switcher frames are tagged private and replaced before they reach any viewer.
- ARCH-ORDER pass: the `watch` state is explicit. The goroutine's lifetime is bounded by `done`, and `do()` returns once the hub has ended.
- ARCH-FUNERAL pass: everything is in memory, and subscriber queues are closed when the hub ends.
- For M2, per the Revisions entry: the server must turn a closed queue into `event: end` carrying `Hub.Err()`. It must also tell that apart from a queue closed by its own `Subscription.Close`.

**7. Plan revisions:** none needed. The two new Revisions entries match the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Tagged off/shown/hidden watch; overlay mutant (Activate ignores shown) turns the new subtest and TestHubRandomInterleavings red.
  - id: BR-2
    disposition: addressed
    note: |
      Plan Revisions 2026-10-07 documents close-queues + Hub.Err() and the M2 server mapping; matches hub.go end().
  - id: BR-3
    disposition: addressed
    note: |
      Plan Revisions entry records the refused-PresentView branch as defence in depth with no public trigger; consistent with presenter.go:380.
  - id: BR-4
    disposition: addressed
    note: |
      run() selects tickC only while resyncing > 0; TestHubRealTickerResyncsQuietScreen exercises the gate and checks resyncing returns to 0.
```
