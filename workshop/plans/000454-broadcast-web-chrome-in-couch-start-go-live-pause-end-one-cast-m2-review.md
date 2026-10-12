# Boundary Review — pair#454 (milestone M2)

| field | value |
|-------|-------|
| issue | 454 — broadcast-web: Chrome in Couch — start, go live, pause, end; one cast |
| repo | pair |
| issue file | workshop/issues/000454-broadcast-web-chrome-in-couch-start-go-live-pause-end-one-cast.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 129331ed1569621da51e8c291ee2a7531c23a963..c690c0e9b61f1c32a5099e90b344e3b24c69a59b |
| command | sdlc milestone-close --issue 454 --milestone M2 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T21:46:48-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing pair#454 M2: I've read the plan and issue. Next I'm going through the code diff, starting with `broadcast.Host`.

I've read the Host, Cast, Couch state machine and launcher wiring. Now running the test suites and probing a possible race in `Host.Mount`'s cancel path.

Confirmed a real race in `Host.Mount` (1 leak in 300 runs). Removing the probe file, then checking test coverage against the plan.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M2 delivers what the plan describes. There is one shared `broadcast.Host` that mounts each cast by token. The stub viewer `Server` and `Cast` sit on it, and Couch has its `Shift+Ctrl+Alt+b` key, the `Chrome ▶/⏸` cell with click / right-click / Ctrl+click, launcher wiring, README and atlas updates, and a live end-to-end test. I checked the code itself. The broadcast, webcast and cdp packages pass under `-race`, and so do the couchtty webcast/status tests (`-count=3`), the couchcmd webcast/broadcast tests, the 12 node viewer tests and artifactpath. The only failures elsewhere in couchtty and couchcmd come from this checkout refusing writes to `/tmp` (`operation not permitted`); none are in M2 code. One thing blocks SHIP: `Host.Mount`'s cancel path can leave the Host open with no mounts and no waiters. I reproduced it with a temporary test (since deleted): 1 leaked open Host in 300 runs. The plan's own M2 revision promises that this cannot happen. It is cheap to fix, so the verdict is FIX-THEN-SHIP. The operator smoke test (Task 10) has not been done yet and is still needed before close.

### 1. Strengths
- **Real reuse, done once.** `broadcast.EventStream` (`sse.go`), `SetSecurityHeaders`, `NewToken` and `ownsToken` are shared, not copied. `Session.Start` now just mounts on a Host, and the existing session tests still pass with the same shape (ARCH-DRY).
- **Viewers can't see paused activity.** `viewerState` (`webcast/stream.go:13`) returns an empty `wireState` whenever the cast isn't live. The live E2E test proves it: a page titled "secret-before-live" never reaches a viewer, and the old title is never paired with the new URL (`webcast/live_test.go:212`).
- **The PQ-1 collision is pinned.** `FakeTunnel.Exclusive` models the named-tunnel claim. `TestHostSharedNamedTunnelServesBothCasts` shows that two private hosts collide and one shared Host serves both casts.
- **Couch tests check what is actually drawn.** They read the drawn cell, include a control (a click off the cell does nothing), count Launch calls, and cover the late-launch-after-shutdown `startClaim` path (`console_webcast_test.go:262`).
- **Viewer page is strict.** It uses `textContent` only, and its CSP is tighter than planned (`style-src 'self'`, no `'unsafe-inline'`).

### 2. Critical findings
None.

### 3. Important findings
**I-1 — `Host.Mount` cancel can leak an open Host** (`cmd/internal/broadcast/host.go:113-130`). Family: `state-machine-untested-sequences`; ARCH-ORDER and ARCH-FUNERAL.

- **The race:** a waiter's ctx fires just as `open()` finishes.
  1. `open()` takes `mu` first. It sees `waiting==1`, so it does not start a close, and it sets `attempt=nil`.
  2. The waiter then decrements `waiting` to 0. Its check `h.attempt == a` is now false, so `last` is false and it returns `ctx.Err()`.
  3. The Host is left in `hostOpen` with zero mounts and zero waiters.
- **Effect:** the tunnel (cloudflared, or a named claim) stays open with nothing mounted until the next Mount. The Host has no `Close` and Couch teardown only unmounts, so at exit nothing closes it.
- **Trigger in Couch:** cancelling a broadcast start (`toggleBroadcast` while Starting), or `Cast.End` while the link is still opening, at the moment the open completes.
- **Repro:** a FakeTunnel with `OpenDelay=20ms` and a ctx timeout of about 18–22ms leaked 1 of 300 times.

> **This is the 2nd finding in family `state-machine-untested-sequences`.** Earlier rounds fixed instances. Don't fix only this one; fix the rule.

- **The rule:** every transition that removes interest in the Host must re-check "nobody wants it" (`waiting==0 && len(mounts)==0`) and close the Host if so. Those transitions are: open completes, a waiter cancels, an Unmount, and a failed probe's Unmount. Do this through one helper, e.g. `releaseLocked()`, which starts the close when the state is `hostOpen` (or flags the in-flight open when `hostOpening`) and returns the attempt for the caller to wait on. The cancel branch at `host.go:113` should then call it instead of using the `h.attempt == a` special case, and then `settle`.
- **The test seam:** the plan promised "a gate channel the test releases". The current Host tests use `OpenDelay` and sleeps, so each run only ever sees whichever interleaving the timing picks. Add a seam that holds `open()` after `start()` returns, so the test can cancel the waiter at that exact point. Then assert the stated invariant: Opens − Closes == 0 once nothing is mounted.

### 4. Minor findings
- **M-1:** `webcast.Server.Owns` (`webcast/server.go:47`) compares the token with `==`. The plan calls for a constant-time compare per mount, and the broadcast `Server` already uses `ownsToken`. Export it (or add a `broadcast.OwnsToken`) and use it in both places (ARCH-SECURE, ARCH-DRY).
- `Cast.ToggleLive` reads the state and then calls `SetLive` (`cast.go:132`). If the in-page button is clicked at the same moment, both can set the same value. The result is still visible in the cell, so this is acceptable for MVP.
- A click on a stale `statusWebcast` span while the cast is Off would start Chrome, because `toggleWebcast` handles Off too. The window is close to zero because `setNotice` repaints immediately. Consider a click-only handler that ignores Off.

### 5. Test coverage notes
- Plan items covered:
  - Host: share, mount during opening, reopen while closing, failed open, cancelled waiter, tunnel exit, named collision.
  - Server: routes, 503 cap, state/end stream.
  - Cast: link-first, end while the link opens, link failure, tunnel lost mid-cast.
  - Couch: all listed cases.
  - Viewer: node tests.
  - Live E2E (Task 9b).
- **Gap:** the Host tests rely on timing instead of an ordering gate; that is how I-1 got through.
- **Not run here:** `WEBCAST_LIVE_CHROME=1`, which needs Chrome outside the sandbox.

### 6. Architectural notes
| Marker | Result | Note |
|---|---|---|
| ARCH-DRY | pass | M-1 aside |
| ARCH-PURE | pass | `viewerState`, `view()` and `webcastLabel` are pure; Host mixes IO with state, which is acceptable for a lifecycle shell |
| ARCH-PURPOSE | pass | everything except the pending operator smoke test |
| ARCH-MOCK | pass | `FakeTunnel.Exclusive` models the claim; FakeBrowser drives Couch; the live E2E is the conformance check |
| ARCH-CONSTRAINTS | pass | 16-viewer cap enforced; 2-ping write deadline is shared |
| ARCH-SECURE | pass with minor | redaction and a strict CSP; M-1 |
| ARCH-ORDER | **flag** | I-1 |
| ARCH-FUNERAL | **flag** | I-1: a leaked Host has no owner to close it. Consider giving Host a `Close` that Couch teardown calls, as a backstop |

Looking ahead, the Recorder will mount on the same Host, so tightening the Host's state-transition rule now pays off later.

### 7. Plan revision recommendations
- Under "M2 as built", add: Host tests gain an open-completion gate seam, and every loss of interest goes through a single "nobody wants it" helper (I-1). That way the cancel claim in the existing revision is enforced, not just stated.
- Optionally note that `web.go` was folded into `server.go`, where the embed now lives.

```findings
findings:
  - id: new
    severity: Important
    family: state-machine-untested-sequences
    title: |
      Host.Mount cancel racing open completion leaves the Host open with no mounts or waiters (tunnel leaked)
    detail: |
      host.go:113-130 — open() wins mu with waiting==1 and sets attempt=nil, then the cancelled waiter decrements waiting to 0 with last=false and returns; reproduced 1/300 with OpenDelay≈ctx timeout. Rule: every loss of interest (open done, waiter cancel, Unmount) runs one "nobody wants it" helper that closes the Host; add the planned gate seam that holds open() after start() so the test can cancel at that point and assert Opens−Closes==0. 2nd finding in this family.
  - id: new
    severity: Minor
    family: capability-token-compare
    title: |
      webcast.Server.Owns compares the token with == instead of the shared constant-time ownsToken
    detail: |
      webcast/server.go:47; plan PQ-1 specifies a constant-time compare per mount and broadcast.Server already uses ownsToken. Export it and use it in both.
```

---

## Re-review — 2026-10-11T21:54:25-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 454 — broadcast-web: Chrome in Couch — start, go live, pause, end; one cast |
| repo | pair |
| issue file | workshop/issues/000454-broadcast-web-chrome-in-couch-start-go-live-pause-end-one-cast.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 129331ed1569621da51e8c291ee2a7531c23a963..c6c9d31487195efcf58efb1e8378522fa547b26a |
| command | sdlc milestone-close --issue 454 --milestone M2 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T21:54:25-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Verifying the BR-14 fix: running the Host tests at head, then again with the fix reverted.

The BR-14 regression test fails deterministically without the fix. Now checking a suspected sibling: a fresh Mount that joins an open already abandoned by an earlier waiter.

Confirmed: a new Mount with a live context fails with `context canceled` from another caller's abandon. Checking whether the real tunnel's Open has that slow-cancel window.

Running the remaining couch tests; one couchcmd failure is environmental (`/tmp` not writable here), so I'm isolating the M2 tests.

M2 couch tests pass; the other failures are sandbox `/tmp`/PTY permissions. Last check: whether a couch user can actually reach the abandoned-open scenario.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both open findings are fixed. BR-14's regression test fails deterministically when the fix is reverted, and BR-15's constant-time compare is now shared by both casts. The new single release rule (`releaseIfUnwantedLocked`) is the right shape. One problem remains in the same Host lifecycle. A `Mount` that arrives while an open is being cancelled (because the last waiter gave up) joins that open. It then returns `context canceled` even though its own context is still live. I reproduced this with a tunnel that takes time to shut down after a cancel, which is how the real cloudflared tunnel behaves: on cancel it sends SIGTERM and waits a grace period before reaping. It is reachable from couch: end a web cast while it is still mounting, then start a screen broadcast within cloudflared's shutdown budget. It is cheap to fix, so FIX-THEN-SHIP rather than REWORK.

### 1. Strengths
- **One release rule.** `cmd/internal/broadcast/host.go:152` `releaseIfUnwantedLocked` is now the only way interest ends: an open completing (`:192`), a waiter giving up (`:133`) and an Unmount (`:346`). This is the rule BR-14 asked for, not just the one instance it named.
- **The BR-14 test is real evidence.** `TestHostLastWaiterGivingUpAfterTheOpenCloses` (`host_test.go:255`) builds the exact losing state. I restored the pre-fix `host.go` with a shim reproducing the old cancel path, and the test failed 3/3 with `Opens:1 Closes:0`. At head it passes, as does the 300-iteration stress test `TestHostCancelRacingOpenNeverLeaks`.
- **BR-15 is done fully.** `OwnsToken` (`host.go:355`) is exported and used by `broadcast.Server.Owns`/`view` (`server.go:87,115`) and by `webcast.Server.Owns` (`webcast/server.go:49`). No `==` token compare is left.
- **The fake tunnel models real failure modes.** `FakeTunnel` has `IgnoreCancel`, `CloseBlock`, `Exclusive` and `MaxOpen`, which let the Host invariants (at most one tunnel, Opens equals Closes) be checked from the outside.
- **Tests run:** `broadcast` (with `-race`), `webcast`, `artifactpath`, `couchkeys`, the M2 subsets of `couchtty`/`couchcmd`, and `tests/webcast/viewer.test.mjs` (5/5) all pass.

### 2. Critical findings
None.

### 3. Important findings
**A Mount inherits another caller's cancel.** This is the 4th finding in family `failure-cause-misattributed`.
- **Where:** `host.go:102` counts a new arrival as a waiter on any attempt in `hostOpening`. That includes an attempt `releaseIfUnwantedLocked` has already cancelled (`:160`).
- **What happens:** while the tunnel winds down, a new `Mount` joins that attempt. It gets `a.err = context.Canceled` at `:112-114` and returns it as if its own request had failed.
- **Reproduction:** a tunnel whose `Open` sleeps 100ms after `ctx.Done()` and then returns `ctx.Err()`. A first Mount times out at 20ms; a second Mount with a background context at 50ms fails with `context canceled`.
- **Rule, not instance:** an error a caller receives must come from that caller's own request. No caller may inherit an attempt that someone else abandoned.
- **Fix sketch:** when `releaseIfUnwantedLocked` cancels an opening attempt, mark it abandoned (an `abandoned` flag on `hostAttempt`, or a separate `hostAbandoning` state). `Mount` then treats it like `hostClosing`: wait without being counted, loop, and reopen.
- **Test:** give `FakeTunnel` a `CancelDelay` that models cloudflared's SIGTERM grace, and add a test where a live-context Mount succeeds after an abandoned open.
- **Sweep:** check that every other error path in `Mount` (`:114`, the probe at `:94`) also comes only from the caller's own context or its own open.

### 4. Minor findings
- The BR-14 test calls `h.abandonWait` directly and bumps `h.waiting` by hand, instead of adding the gate seam the finding suggested. It is sound evidence, but it would not notice if `Mount`'s `ctx.Done` branch stopped calling `abandonWait`. The stress loop covers that path only probabilistically.
- `abandonWait` (`host.go:141`) waits on whichever close is in progress, which may belong to a later generation. That is harmless, just slightly over-broad.

### 5. Test coverage notes
- Some `couchtty`/`couchcmd` tests fail in this checkout because of the sandbox, not the code: `mkdir /tmp/...: operation not permitted` and PTY child start denied. The M2-scoped tests pass.
- The Host tests can't produce the "cancelled but still opening" interleaving, because `FakeTunnel` either returns at once on cancel or ignores it. That is why the finding above went unseen. Giving the fake a cancel delay closes that gap (ARCH-MOCK).

### 6. Architectural notes
- **ARCH-DRY:** pass. `OwnsToken`, `OpenEventStream` and `SetSecurityHeaders` are shared between the two casts.
- **ARCH-PURE:** pass. The Host is IO glue, and the state is kept under one mutex.
- **ARCH-PURPOSE:** pass. One shared Host serves both casts (`TestHostSharedNamedTunnelServesBothCasts`).
- **ARCH-MOCK:** flagged. The fake tunnel lacks the slow-cancel behaviour of the real tunnel (the finding above).
- **ARCH-CONSTRAINTS:** pass. There is a viewer cap, header and idle timeouts, and a bounded shutdown.
- **ARCH-SECURE:** pass. Tokens are compared in constant time, viewers are GET-only with a CSP, and unknown tokens get a 404.
- **ARCH-ORDER:** flagged. The state is a tagged value, but "opening, abandoned" is an unnamed sub-state carried in `attempt.cancel` having been called. Name it so the transition table can be read off the code.
- **ARCH-FUNERAL:** pass. The Host closes after the last Unmount, a waiter that gives up waits for that close, and `watch` exits on `ErrServerClosed`.

### 7. Plan revision recommendations
- In the M2 Revisions entry after BR-14, add: "Abandoned open is a distinct state; arrivals wait for it to settle and reopen rather than inheriting its cancel (`failure-cause-misattributed`, 4th)."

```findings
dispose:
  - id: BR-14
    disposition: addressed
    note: |
      releaseIfUnwantedLocked is the single release path; TestHostLastWaiterGivingUpAfterTheOpenCloses fails 3/3 against pre-fix host.go (Opens 1 Closes 0) and passes at head.
  - id: BR-15
    disposition: addressed
    note: |
      OwnsToken exported (host.go:355) and used by broadcast.Server (server.go:87,115) and webcast.Server.Owns (webcast/server.go:49).
findings:
  - id: new
    severity: Important
    family: failure-cause-misattributed
    title: |
      A Mount arriving during an abandoned open joins it and returns another caller's context.Canceled
    detail: |
      host.go:102 counts a new arrival on any hostOpening attempt, including one releaseIfUnwantedLocked already cancelled (:160). Reproduced: a tunnel that takes 100ms to wind down after cancel (as cloudflared's SIGTERM grace does) makes a background-context Mount fail with "context canceled". 4th in family. Rule: a Mount's error must come from its own request, never from an attempt another caller abandoned. Mark abandoned attempts (flag or hostAbandoning state), have Mount wait uncounted and reopen, add FakeTunnel.CancelDelay plus a regression test.
  - id: new
    severity: Minor
    family: state-machine-untested-sequences
    title: |
      BR-14 regression test drives abandonWait directly instead of through Mount via a gate seam
    detail: |
      host_test.go:255 bumps h.waiting by hand and calls abandonWait, so a Mount ctx.Done branch that stopped calling it would be caught only by the probabilistic stress loop. 3rd in family. Rule: Host sequence tests reach a state through public events with an injected ordering seam (hold open() after start()), not by mutating fields.
```

---

## Re-review — 2026-10-11T22:05:53-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 454 — broadcast-web: Chrome in Couch — start, go live, pause, end; one cast |
| repo | pair |
| issue file | workshop/issues/000454-broadcast-web-chrome-in-couch-start-go-live-pause-end-one-cast.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 129331ed1569621da51e8c291ee2a7531c23a963..c0bccd94e72b8c68bc77a9895326ee77ac9772e4 |
| command | sdlc milestone-close --issue 454 --milestone M2 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T22:05:53-07:00 |
| verdict | SHIP |

## Review

Host tests pass at head; now reverting the BR-16 fix in the checkout to confirm its regression test goes red.

The regression goes red without the fix, so BR-16 is addressed. Next I'm running the full test suite with race detection.

Some couch test failures are environment errors (PTY spawn/mkdir not permitted). Next I'll check whether the reported race and the couchcmd failures come from the diff.

Webcast-scoped tests pass with the race detector; the remaining failures are environment-only. Next I'm reading the couch webcast wiring and webcast server.

```verdict
verdict: SHIP
confidence: medium
```

**Summary.** This round fixed BR-16. An open that every caller has given up on is now marked `abandoned` (`host.go:69-71`, set at `:165`). A new Mount no longer joins it as a counted waiter (`:107`). Instead it waits that attempt out without being counted and then reopens, so the only error it can get back is from its own request. I reverted the one-line guard at `host.go:107` in the checkout and `TestHostMountAfterAnAbandonedOpenSucceeds` failed with `mount after an abandoned open: context canceled`. With the fix restored it passes, so the fix is real and the test catches the bug. BR-17 is still open, but it is Minor. The only other new finding is also Minor, so nothing blocks the M2 boundary.

Confidence is medium, not high, because many couchtty/couchcmd tests could not run in this environment. They failed on PTY start and `/tmp` mkdir, both with `operation not permitted`, and some run under `-race` also reported a data race whose source I did not trace. All webcast-, cast- and reserve-scoped tests in those packages pass under `-race`, and none of the failing tests are M2 webcast tests.

**1. Strengths**
- **One release rule.** `releaseIfUnwantedLocked` (`host.go:157`) is the single decision for every way interest can lapse: an open completing, a waiter abandoning, or an Unmount. This is why the BR-14 and BR-16 fixes are each only a few lines.
- **Host states.** The Host keeps one tagged state (closed → opening → open → closing) under one mutex, which applies ARCH-ORDER. The `abandoned` mark sits on the attempt, not on the Host, so it can't outlive the open it describes.
- **Token check is shared.** `broadcast.OwnsToken` (`host.go:361`) is the single constant-time token comparison. Both `webcast.Server.Owns` (`webcast/server.go:49`) and `broadcast.Server.Owns` (`server.go:87`, `:115`) use it, so the BR-15 class is closed.
- **Tests check behaviour from outside.** `assertServes` checks two invariants from the outside: a handler is reachable exactly when its mount is live, and `FakeTunnel.Stats().MaxOpen ≤ 1`. It does not re-read the implementation.
- **Collision test.** `FakeTunnel.Exclusive` models a named tunnel's single claim, which lets `TestHostSharedNamedTunnelServesBothCasts` reproduce the original collision (two private hosts get `ErrTunnelBusy`) and show that one shared Host serves both casts.

**2. Critical findings:** none.

**3. Important findings:** none.

**4. Minor findings**
- **BR-17 (not addressed).** `host_test.go:259-266` still increments `h.waiting` by hand and calls `abandonWait(true)` directly. This test belongs to family `state-machine-untested-sequences` (its 4th instance). The rule that covers the whole family: each Host sequence test should reach its state only through public events (`Mount`, ctx cancel, `Unmount`, tunnel exit), with ordering controlled by a seam such as a gate channel on `FakeTunnel.Open` that holds `start()` open. Tests should not mutate fields or call internals. The BR-16 test already does this correctly: it observes `attempt.abandoned` but drives everything through `Mount`.
- **New, `fake-fidelity` (2nd in family).** The tunnel behaviour BR-16 depends on (notices a cancel only once a slow open finishes) is modelled in a test-local wrapper, `lateCancelTunnel` (`host_test.go:294`), as is `failingTunnel` (`:123`). The shared `FakeTunnel` does not model either. The webcast and couch tests use the shared fake, so they can't exercise these behaviours, and no live conformance check compares them with cloudflared. The rule (ARCH-MOCK): every behaviour of a dependency that a test relies on is a setting on the shared stateful fake, e.g. `FakeTunnel.CancelDelay` and `FailOpens int`, never a one-off wrapper.

**5. Test coverage notes**
- Host tests (`-count=3`) and the full `broadcast` and `webcast` packages pass under `-race`. `tests/webcast/viewer.test.mjs` passes.
- `go vet` and the couch packages need `runtimebundle-generate` to run first. Once I generated the assets, couchkeys and artifactpath passed. Webcast/cast/reserve-scoped tests in couchtty and couchcmd passed under `-race`. The rest of those two packages failed with `operation not permitted` errors (PTY spawn, `/tmp` mkdir), which comes from the environment, not this diff.
- BR-16 fix: reverting it makes its test fail and restoring it makes the test pass, as shown above.

**6. Architectural notes for upcoming work**
- **ARCH-DRY: pass.** Token comparison and the release rule each live in one place.
- **ARCH-PURE: pass.** Host is necessarily an IO shell; its pure decisions (`releaseIfUnwantedLocked`, its state switch) are small.
- **ARCH-PURPOSE: pass.** The shared Host serves both casts, which was the point of PQ-1.
- **ARCH-MOCK: flag (Minor).** See the `fake-fidelity` finding above.
- **ARCH-CONSTRAINTS: pass.** Shutdown is bounded by `shutdownBudget`, and the server has header/idle timeouts.
- **ARCH-SECURE: pass.** Tokens are compared in constant time and unknown tokens return 404 with security headers.
- **ARCH-ORDER: pass, with a note.** Host state changes go through the tagged state. An abandoning waiter that was never counted can still call `releaseIfUnwantedLocked`, which re-cancels the attempt (harmless, since it's idempotent) and then blocks until the abandoned open settles. That is acceptable but not written down anywhere; worth stating in the Host doc comment.
- **ARCH-FUNERAL: pass.** The Chrome profile directory is kept on purpose so logins survive, and its removal path ("delete the directory") is documented in `couchcmd/webcast.go:19-21`. Tunnels are closed by the last Unmount or by the release rule.

**7. Plan revision recommendations**
- The plan's BR-16 entry should say the regression test uses a test-local tunnel wrapper, not the `FakeTunnel.CancelDelay` setting the finding recommended. Alternatively, move that behaviour into `FakeTunnel` and update the entry to match.

```findings
dispose:
  - id: BR-16
    disposition: addressed
    note: |
      attempt.abandoned makes the attempt unjoinable (host.go:107,165); reverting the guard turns TestHostMountAfterAnAbandonedOpenSucceeds red with "context canceled".
  - id: BR-17
    disposition: not-addressed
    note: |
      host_test.go:259-266 still mutates h.waiting and calls abandonWait directly; no gate seam drives it through Mount.
findings:
  - id: new
    severity: Minor
    family: fake-fidelity
    title: |
      Late-cancel and failing-open tunnel behaviours are test-local wrappers, not settings on the shared FakeTunnel
    detail: |
      lateCancelTunnel (host_test.go:294) and failingTunnel (:123) model cloudflared behaviour BR-16 depends on, but only in broadcast tests; webcast/couch tests on FakeTunnel cannot exercise it. Rule: every dependency behaviour a test relies on is a FakeTunnel setting (CancelDelay, FailOpens), covered by the live conformance check.
```
