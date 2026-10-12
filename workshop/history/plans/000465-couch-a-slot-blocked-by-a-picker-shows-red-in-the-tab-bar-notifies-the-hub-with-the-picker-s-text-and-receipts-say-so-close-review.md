# Boundary Review — pair#465 (whole-issue close)

| field | value |
|-------|-------|
| issue | 465 — couch: a slot blocked by a picker shows red in the tab bar, notifies the hub with the picker's text, and receipts say so |
| repo | pair |
| issue file | workshop/issues/000465-couch-a-slot-blocked-by-a-picker-shows-red-in-the-tab-bar-notifies-the-hub-with-the-picker-s-text-and-receipts-say-so.md |
| boundary | whole-issue close |
| milestone | — |
| window | d209a7606fced42d297a1ceffe6e6b4ebc2d62b2..069d567b34456828336b118824ed495b036bccb6 |
| command | sdlc close --issue 465 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T09:57:57-07:00 |
| verdict | REWORK |

## Review

Checking a possible nil-conn panic on the redial path, then the delivery hold logic.

Binding validity doesn't matter to the client; rerunning verbosely to see which error path was taken.

Confirmed a crash: if the broker drops the hello and is gone before the redial, the deferred close calls `Close` on a nil `net.Conn`. Now running the changed packages' tests.

```verdict
verdict: REWORK
confidence: high
```

The feature itself is delivered and matches the Spec and Plan. A slot behind a picker turns red in the tab bar and the switcher. The hub gets one specific notice about a minute after the picker opens, and that holds even while the auto-deny countdown keeps repainting. A held message's receipt reads `held` with `blocked by a picker: <title>`. Tests pass for every package the change touches: `couchmessage`, `couchcmd`, `couchtty`, `wrapcmd` and `artifactpath` (this needed `make runtimebundle-generate` first in this checkout). One thing blocks shipping. The rewrite of the hello fallback in `SessionClient.session` added a crash: if the redial fails, the wrapper calls `Close` on a nil connection. I reproduced it with a throwaway test. The panic happens in the wrapper's session goroutine and nothing recovers it, so it takes down the wrapper and with it the operator's agent session. The fix is one line plus a regression test.

### 1. Strengths
- **One parser, two users (ARCH-DRY).** `couchmessage.ReadPicker` (`cmd/internal/couchmessage/picker.go:412`) serves both the hub notice and the wrapper's receipt. `dialogTitle` and `stripTailMarkup` now delegate to it instead of keeping their own copies. The truncation code that was written inline before is now one shared helper, `truncateBytes` (`wait_notice.go:283`).
- **The picker is reported the moment it opens or closes.** `setPickerLocked` (`wrap.go:2194`) is the only code that writes `pickerActive`; I grepped for other `Swap`/`Store` calls and found none. It reports under `overlayMu`, which correctly avoids the trap that the countdown repaint never lets output go quiet.
- **Older Couch builds keep working.** Session frames decode strictly, so the wrapper offers `hello-v3`, then `hello-v2`, then `hello`. `TestHelloV2SessionClaimingPickerEnds` and `TestNegotiationNewWrapperV2BrokerKeepsSettled` pin that.
- **The picker's wait counts from when it opened** (`PickerSince`, `wait_notice.go:344`), while episodes stay keyed on `LastActivity`. The comment explains why the two clocks are never compared.
- **The Console handoff never blocks.** `SetBlockedThreads` (`console_blocked.go:9`) replaces any update not yet taken, so the message-service loop never waits on a Console lock.

### 2. Critical findings
- **`cmd/internal/couchmessage/session_transport.go:302`: crash when the redial fails.**
  - `defer func() { conn.Close() }()` closes over the variable `conn`.
  - In the fallback loop, `conn, err = (&net.Dialer{}).DialContext(...)` sets `conn` to a nil interface when the dial fails, then returns.
  - The deferred `conn.Close()` then crashes with a nil pointer dereference.
  - The old code was safe: it evaluated `defer conn.Close()` at defer time and only deferred the second close after a successful dial.
  - **Trigger:** the broker accepts the connection, drops the newer hello without acking, and is gone before the redial. A Couch exiting or restarting during the handshake does exactly this.
  - **Reproduced:** a listener that reads the hello, then closes and removes its socket. `session()` crashed at line 302, called from line 406.
  - **Fix:** `defer func() { if conn != nil { conn.Close() } }()`, or dial into a temporary and assign it to `conn` only on success. Add that repro as a regression test in `session_negotiation_test.go`.

### 3. Important findings
- None.

### 4. Minor findings
- **`peer_delivery.go:1051`:** `pickerTitle` is cached "once per picker", but it is cleared only when a dispatch sees the latch down. If picker A closes and picker B opens between two 1s dispatches, the receipt keeps A's title.
- **`peer_delivery.go:1051`:** when the title is not recognized (for example Codex's image capture, which also sets the latch), the cache stays empty. The 60-line tail is then re-read on every dispatch (1s poll) while holding `inputAdmission` and `d.mu`. That is cheap, but the "read once" claim in the plan does not hold here.
- **`wrap.go:1306`:** Codex image capture now shows red and sends a hub notice reading `behind a picker (picker text not recognized)`. That may be noise, since an image capture is not really a picker. Worth a sentence in the atlas, or exclude it.
- **`menu_render.go:943`:** the switcher's selected row is never red, while the active chip in the tab bar is. That is probably intentional, but the comment says "as the tab bar draws it".

### 5. Test coverage notes
- Every plan test item has a test:
  - `TestReadPicker*`
  - the negotiation tests
  - `TestPickerReportedToCouchAtOnce`
  - `TestPeerDeliveryHeldByAPickerNamesIt` and `TestPeerDeliveryExpiresBehindAPickerWithItsReason`
  - `TestPickerNotice*` and `TestBlockedThreadsReachTheConsoleView`
  - the `couchtty` tests for render precedence and the switcher row
- Gap: the hello-fallback path where the redial fails has no test. That is exactly how the crash above shipped.

### 6. Architectural notes
- **ARCH-DRY: pass.** One `ReadPicker`; `truncateBytes` is shared.
- **ARCH-PURE: pass.** `ReadPicker`, `classifyWait`, `pickerSummary` and `ReduceMenu(MenuEventBlocked)` are pure and tested without IO. The tail read happens at the wrapper's edge.
- **ARCH-PURPOSE: pass.** Every Done-when item is delivered and the operator smoke test is logged. Answering pickers from the hub is out of scope per the Spec.
- **ARCH-MOCK: pass.** Tests use the existing fake broker and recorded screens.
- **ARCH-CONSTRAINTS: pass, with one note.** The 15s tick reads memory only. The tail is re-read on every dispatch when the title is unrecognized (Minor above).
- **ARCH-SECURE: pass.** Picker text comes from the agent's screen. It is bounded per part (200 bytes for detail and title, 160 for options) and only sent as message text.
- **ARCH-ORDER: flag.** The fallback's error path drops the connection-ownership invariant (the Critical above). Otherwise it passes: the latch has a single writer under `overlayMu`, and the registry derives `pickerSince` from the false→true transition.
- **ARCH-FUNERAL: pass.** Nothing durable is created. The blocked set and `pickerSince` live and die with the process and session entry.
- **Docs:** `atlas/couch.md` and the couch `SKILL.md` are updated. There is no new flag, key or command, so README is fine.

### 7. Plan revision recommendations
- §1 of the plan still says the wrapper reports the picker through the settle probe "when output goes quiet (3s)". The code reports it at the latch flip (`setPickerLocked`), and the issue Log records that decision. Add a `## Revisions` entry so the plan stops describing the settle-probe path. §1 also says `Picker` rides on "hello-v2 sessions"; it should say hello-v3.

```findings
findings:
  - id: new
    severity: Critical
    family: error-path-resource-ownership
    title: |
      SessionClient.session crashes on a nil conn when the hello fallback redial fails
    detail: |
      session_transport.go:302 defers a closure over conn; the fallback loop reassigns conn from DialContext, which returns a nil interface on error, so the deferred Close crashes and takes the wrapper down (no recover). Reproduced with a broker that drops the hello then removes its socket. Guard with a nil check (or dial into a temporary) and add the repro as a regression test.
  - id: new
    severity: Minor
    family: per-episode-cache-invalidation
    title: |
      pickerTitle stays stale across back-to-back pickers, and an unrecognized title re-reads the tail on every dispatch
    detail: |
      peer_delivery.go:1051 clears the cache only when a dispatch sees the latch down, and caches nothing when ReadPicker finds no title, so it re-renders 60 tail lines every 1s under inputAdmission.
  - id: new
    severity: Minor
    family: plan-matches-code
    title: |
      Plan section 1 still says the settle probe reports Picker on hello-v2; the code reports it at the latch flip on hello-v3
```

---

## Re-review — 2026-10-12T10:04:36-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 465 — couch: a slot blocked by a picker shows red in the tab bar, notifies the hub with the picker's text, and receipts say so |
| repo | pair |
| issue file | workshop/issues/000465-couch-a-slot-blocked-by-a-picker-shows-red-in-the-tab-bar-notifies-the-hub-with-the-picker-s-text-and-receipts-say-so.md |
| boundary | whole-issue close |
| milestone | — |
| window | d209a7606fced42d297a1ceffe6e6b4ebc2d62b2..f37e7e65678c4bcde17ae39c1d729ba5ec8901ea |
| command | sdlc close --issue 465 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T10:04:36-07:00 |
| verdict | SHIP |

## Review

Checking the wrapper and transport diffs went fine. Next I'll look at the Couch-side changes and run the tests.

All touched packages are passing so far. Next I'm re-running the Couch packages and reverting the BR-5 fix to confirm its regression test catches it.

```verdict
verdict: SHIP
confidence: high
```

All seven earlier findings are fixed, and I checked each fix against code and tests instead of relying on the commit messages. The Critical one, BR-5, is really fixed. `session_transport.go:308-328` now gives each hello attempt its own connection, and only the accepted one reaches the deferred `Close`. I put back the old pattern (one shared `conn`, a deferred closure, reassigned by the redial) in the review checkout. `TestNegotiationFallbackRedialFailureReturnsError` then failed with the same nil-pointer panic, and passed once I restored the file. The touched packages all pass: `couchmessage`, `couchcmd`, `couchtty`, `wrapcmd` and `artifactpath`. `wrapcmd` and `artifactpath` first needed the runtime bundle assets generated (`go run ./cmd/internal/runtimebundle/generatecmd`); that step is part of the build, not a defect. The code delivers all three parts of the Spec: the red chip, the early hub notice and the held receipt. Nothing blocks shipping; there are two Minor documentation notes.

## 1. Strengths
- **Picker reported when the latch changes** (`wrap.go:2198` `setPickerLocked`). Every write to `pickerActive` goes through it; grep finds no direct `Swap`/`Store` left. It runs under `overlayMu`, so reports arrive in the order the latch changed. The sink never blocks. This fixes the countdown-repaint gap (BR-3) at the source.
- **Version skew.** `hello-v3` falls back to v2, then to plain hello. Strict decoding refuses `Picker` on v1/v2 sessions in both directions (`session_protocol.go:76-90`, `session_transport.go:138`). That protects a Couch that hasn't been restarted.
- **One parser for both readers.** `couchmessage.ReadPicker` serves the hub notice and the receipt, and the old `dialogTitle`/`stripTailMarkup` now delegate to it (ARCH-DRY).
- **No lock coupling between service and Console.** `SetBlockedThreads` is a latest-value channel that never blocks, and Console state changes go through `ReduceMenu`.
- **Watcher episode keys.** The key now copes with a nil `Settled` on the picker path (`wait_notice.go:340`). The wait is measured from `PickerSince`, and the episode is still keyed by `LastActivity`, so a repainting picker gets exactly one notice.

## 2. Critical findings
None.

## 3. Important findings
None.

## 4. Minor findings
- **README not updated for the red state.** README.md:766-775 describes how tabs and switcher rows are coloured (fading, and what never fades) but doesn't mention the new red "behind a picker" state or that a blocked thread never fades. `atlas/couch.md` and the couch `SKILL.md` were updated; README wasn't. This is a visual state, not a typed command or flag, so Minor.
- **Plan body still describes the old design in §1 and the Tests list.** They still say "settle probe" and "hello-v2". The Revisions entry correctly says these are superseded, so this is acceptable as written.

## 5. Test coverage notes
- Picker hold: `TestPeerDeliveryHeldByAPickerNamesIt` asserts `Held`, the detail text, survival past the deadline and the paste after the picker closes. `TestPeerDeliveryExpiresBehindAPickerWithItsReason` covers expiry past the hold limit.
- Title read: `TestPickerHoldDetailReadsOncePerEpisode` checks that an unrecognised title reads the tail once, and that a second picker is read afresh.
- Watcher order: `TestPickerArrivingMidEpisodeDoesNotNotifyTwice` and `TestPickerNoticeCountsFromOpeningThroughRepaints` cover the ARCH-ORDER sequences.
- Not covered: a wrapper reconnecting while a picker is up. The client marks its state dirty on reconnect and re-sends, but `pickerSince` restarts from the first frame of the new session. After a Couch restart, the minute is counted again from reconnect. That is acceptable.

## 6. Architectural notes
- **ARCH-DRY: pass.** One parser, and `truncateBytes` is shared with the generic notice.
- **ARCH-PURE: pass.** `ReadPicker`, `classifyWait`, `pickerSummary`, `ReduceMenu` and `RenderStatusRow` are pure. IO stays in the watcher tick and the dispatcher.
- **ARCH-PURPOSE: pass.** Red chip, early notice and named receipt are all delivered, and the operator's smoke test is logged.
- **ARCH-MOCK: pass.** Tests drive real session sockets and recorded screens from the 2.1.296 fixtures; no stateless mocks of stateful interactions.
- **ARCH-CONSTRAINTS: pass.** The 15s tick reads memory only. Tail IO happens once per episode with a 2s timeout. The receipt reads the title once per picker episode.
- **ARCH-SECURE: pass.** Text read from the screen is bounded per part (200/200/160 bytes) and cut on a rune boundary. An unrecognised picker produces a visible "(picker text not recognized)" rather than invented text.
- **ARCH-ORDER: pass.** Registry state changes go through `Advance`. The latch has a single writer under `overlayMu`. `pickerEpisode` replaced the guess "a reader happened to see the latch go down".
- **ARCH-FUNERAL: pass.** Nothing durable is created. The registry's picker state lives with its session, the blocked set is replaced on every change, and watcher episodes are pruned each tick.

## 7. Plan revision recommendations
None required; the existing Revisions entry already covers §1 and §5.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      peer_delivery.go:438-443 sets judged, operatorHold = true, true before returning; TestPeerDeliveryHeldByAPickerNamesIt asserts Held.
  - id: BR-2
    disposition: addressed
    note: |
      picker_test.go covers PickerQuestion via the Enter-to-select footer and PickerUnknown ("Pick a model"), matching Revision (a).
  - id: BR-3
    disposition: addressed
    note: |
      Picker is now reported when the latch changes (setPickerLocked), not gated on quiet; TestPickerReportedToCouchAtOnce covers the repainting countdown.
  - id: BR-4
    disposition: addressed
    note: |
      TestPickerArrivingMidEpisodeDoesNotNotifyTwice flips Picker mid-episode after the generic notice and asserts one notice.
  - id: BR-5
    disposition: addressed
    note: |
      Each attempt owns its conn (session_transport.go:308-328). Reverting to a shared deferred conn makes TestNegotiationFallbackRedialFailureReturnsError panic with a nil pointer; restored, it passes.
  - id: BR-6
    disposition: addressed
    note: |
      Title cached per pickerEpisode (bumped on the latch's false-to-true flip in setPickerLocked); TestPickerHoldDetailReadsOncePerEpisode pins one read per episode and a fresh read for the next picker.
  - id: BR-7
    disposition: addressed
    note: |
      Plan Revisions entry (2026-10-12, build and close review) supersedes section 1: reported when the latch flips, on hello-v3; atlas/couch.md:266-280 agrees.
findings:
  - id: new
    severity: Minor
    family: readme-surface-coverage
    title: |
      README tab/switcher colour section does not mention the red behind-a-picker state
    detail: |
      README.md:766-775 lists the tab and switcher colour states (idle fade, what never fades); the new red state, and the fact that a blocked thread never fades, appear only in atlas/couch.md and SKILL.md.
```
